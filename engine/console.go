package engine

// Cheat console (prefix cn). Press / to open a text console and type classic commands: god, noclip,
// map, unlockall, gold 5000, level 20, rep all 800, spawn wolf 3 ... Type `help` for the list.
// Using any cheat marks the run as sandboxed (score 0 in the hall of fame), like the F2 sandbox menu.

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

type cnState struct {
	Noclip   bool     `json:"nc,omitempty"`
	NoHunger bool     `json:"nh,omitempty"`
	Hist     []string `json:"h,omitempty"`
	buf      string
	out      []string
	hi       int // history cursor
}

func (g *Game) cn() *cnState {
	return g.ext("cn", func() any { return &cnState{} }).(*cnState)
}

type cnCmd struct {
	name  string
	alias []string
	usage string
	help  string
	cheat bool
	run   func(g *Game, a []string) string
}

var cnCmds []cnCmd

func init() {
	mainKeys["/"] = func(g *Game) {
		st := g.cn()
		st.buf, st.hi = "", len(st.Hist)
		if len(st.out) == 0 {
			st.out = []string{"{{K|Cheat console. Type help for the commands, Esc to close.}}"}
		}
		g.Mode = "cn"
	}
	modeKeys["cn"] = cnKey
	modeDraws["cn"] = cnDraw
	turnHooks = append(turnHooks, func(g *Game) {
		if st := g.cn(); st.NoHunger && g.P != nil && !g.Dead {
			g.Hunger, g.Thirst = 0, 0
			if g.Breath < 30 {
				g.Breath = 30 + g.P.M(TOU)*5
			}
		}
	})
	cnCmds = []cnCmd{
		{"help", []string{"?", "commands"}, "help [command]", "list the commands, or explain one", false, cnHelp},
		{"clear", []string{"cls"}, "clear", "clear the console", false, func(g *Game, a []string) string { g.cn().out = nil; return "" }},
		{"god", []string{"godmode", "iddqd"}, "god", "toggle god mode: you cannot be hurt", true, func(g *Game, a []string) string {
			sbSetGod(g, !g.sb().God)
			return fmt.Sprintf("God mode %s.", cnOnOff(g.sb().God))
		}},
		{"noclip", []string{"ghost", "idspispopd"}, "noclip", "toggle walking through walls and creatures' tiles", true, func(g *Game, a []string) string {
			st := g.cn()
			st.Noclip = !st.Noclip
			return fmt.Sprintf("Noclip %s.", cnOnOff(st.Noclip))
		}},
		{"nohunger", []string{"nofood", "full"}, "nohunger", "toggle never being hungry, thirsty or out of breath", true, func(g *Game, a []string) string {
			st := g.cn()
			st.NoHunger = !st.NoHunger
			return fmt.Sprintf("No hunger %s.", cnOnOff(st.NoHunger))
		}},
		{"heal", []string{"fullheal", "restore"}, "heal", "full health, cure all statuses, no hunger or thirst", true, func(g *Game, a []string) string {
			sbRun(g, 10)
			return "You are whole."
		}},
		{"hp", nil, "hp <n>", "set your current HP", true, func(g *Game, a []string) string {
			n, ok := cnInt(a, 0)
			if !ok {
				return "usage: hp <n>"
			}
			g.P.HP = clampi(n, 1, g.P.MaxHP)
			return fmt.Sprintf("HP %d/%d.", g.P.HP, g.P.MaxHP)
		}},
		{"maxhp", nil, "maxhp <n>", "set your maximum (and current) HP", true, func(g *Game, a []string) string {
			n, ok := cnInt(a, 0)
			if !ok || n < 1 {
				return "usage: maxhp <n>"
			}
			g.P.MaxHP, g.P.HP = n, n
			return fmt.Sprintf("Max HP %d.", n)
		}},
		{"kill", []string{"killall", "nuke"}, "kill", "slay every hostile in the zone", true, func(g *Game, a []string) string {
			n := 0
			for _, x := range append([]*Actor{}, g.Cur.Actors...) {
				if x.HP > 0 && g.Hostile(x) {
					g.hurt(x, x.HP+1000, g.P, "console")
					n++
				}
			}
			g.cleanup()
			return fmt.Sprintf("%d hostiles slain.", n)
		}},
		{"gold", []string{"money", "give$", "rosebud"}, "gold <n>", "add (or with a minus, remove) coin", true, func(g *Game, a []string) string {
			n, ok := cnInt(a, 0)
			if !ok {
				return "usage: gold <n>"
			}
			g.P.Gold = maxi(0, g.P.Gold+n)
			return fmt.Sprintf("%+d %s: you hold %d.", n, g.Sk.Currency, g.P.Gold)
		}},
		{"xp", []string{"exp"}, "xp <n>", "gain experience", true, func(g *Game, a []string) string {
			n, ok := cnInt(a, 0)
			if !ok || n < 0 {
				return "usage: xp <n>"
			}
			g.gainXP(n)
			return fmt.Sprintf("+%d XP.", n)
		}},
		{"level", []string{"lvl", "setlevel"}, "level <n>", "raise yourself to level n", true, func(g *Game, a []string) string {
			n, ok := cnInt(a, 0)
			if !ok || n < 1 || n > 60 {
				return "usage: level <1..60>"
			}
			for g.P.Level < n {
				g.gainXP(XPFor(g.P.Level) - g.XP)
			}
			return fmt.Sprintf("You are level %d.", g.P.Level)
		}},
		{"points", []string{"ap", "sp", "mp"}, "points <ap|sp|mp> <n>", "add attribute, skill or mutation points", true, func(g *Game, a []string) string {
			if len(a) < 2 {
				return "usage: points <ap|sp|mp> <n>"
			}
			n, ok := cnInt(a, 1)
			if !ok {
				return "usage: points <ap|sp|mp> <n>"
			}
			switch strings.ToLower(a[0]) {
			case "ap":
				g.AP += n
			case "sp":
				g.SP += n
			case "mp":
				g.MP += n
			default:
				return "usage: points <ap|sp|mp> <n>"
			}
			return fmt.Sprintf("AP %d, SP %d, MP %d.", g.AP, g.SP, g.MP)
		}},
		{"stat", []string{"attr"}, "stat <str|agi|tou|int|wil|ego|all> <n>", "set an attribute score", true, func(g *Game, a []string) string {
			if len(a) < 2 {
				return "usage: stat <str|agi|tou|int|wil|ego|all> <n>"
			}
			n, ok := cnInt(a, 1)
			if !ok {
				return "usage: stat <str|agi|tou|int|wil|ego|all> <n>"
			}
			names := map[string]int{"str": STR, "agi": AGI, "tou": TOU, "int": INT, "wil": WIL, "ego": EGO}
			k := strings.ToLower(a[0])
			if k == "all" {
				for i := range g.P.Attr {
					g.P.Attr[i] = n
				}
				return fmt.Sprintf("All attributes %d.", n)
			}
			i, ok := names[k]
			if !ok {
				return "unknown attribute: " + a[0]
			}
			g.P.Attr[i] = n
			return fmt.Sprintf("%s %d.", AttrNames[i], n)
		}},
		{"skill", []string{"skills"}, "skill <id|all> <rank>", "set a skill rank (ids: " + strings.Join(SkillIDs, " ") + ")", true, func(g *Game, a []string) string {
			if len(a) < 2 {
				return "usage: skill <id|all> <rank>"
			}
			n, ok := cnInt(a, 1)
			if !ok {
				return "usage: skill <id|all> <rank>"
			}
			if strings.ToLower(a[0]) == "all" {
				for _, id := range SkillIDs {
					g.Skills[id] = n
				}
				return fmt.Sprintf("Every skill at rank %d.", n)
			}
			g.Skills[strings.ToLower(a[0])] = n
			return fmt.Sprintf("%s rank %d.", a[0], n)
		}},
		{"power", []string{"powers"}, "power <id|all> <level>", "learn a power (ids: see the character screen); `power list` lists them", true, func(g *Game, a []string) string {
			if len(a) == 0 || a[0] == "list" {
				var ids []string
				for _, p := range PowerDefs {
					if !p.Defect {
						ids = append(ids, p.ID)
					}
				}
				return strings.Join(ids, " ")
			}
			n := 1
			if v, ok := cnInt(a, 1); ok {
				n = v
			}
			if g.P.Powers == nil {
				g.P.Powers = map[string]int{}
			}
			if strings.ToLower(a[0]) == "all" {
				for _, p := range PowerDefs {
					if !p.Defect {
						g.P.Powers[p.ID] = n
					}
				}
				return fmt.Sprintf("Every power at level %d.", n)
			}
			for _, p := range PowerDefs {
				if p.ID == strings.ToLower(a[0]) {
					g.P.Powers[p.ID] = n
					return fmt.Sprintf("%s level %d.", p.ID, n)
				}
			}
			return "unknown power (try `power list`)"
		}},
		{"rep", []string{"reputation"}, "rep <home|all|faction> <n>", "set reputation: with your home realm, every realm and faith, or a faction key like state:3", true, cnRep},
		{"pardon", []string{"wanted", "clearbounty"}, "pardon", "clear every bounty on your head", true, func(g *Game, a []string) string {
			st := g.uwSt()
			for k := range st.Bounty {
				st.Bounty[k] = 0
			}
			return "Every bounty is cleared."
		}},
		{"karma", nil, "karma <axis 0..4> <n>", "shift a karma axis (0 mercy, 1 honesty, 2 courage, 3 generosity, 4 piety)", true, func(g *Game, a []string) string {
			ax, ok1 := cnInt(a, 0)
			n, ok2 := cnInt(a, 1)
			if !ok1 || !ok2 || ax < 0 || ax > 4 {
				return "usage: karma <axis 0..4> <n>"
			}
			g.crKarma(ax, n, "console")
			return "Karma shifted."
		}},
		{"map", []string{"reveal", "fullmap", "mapall"}, "map", "reveal the whole world map and this zone", true, func(g *Game, a []string) string {
			cnReveal(g)
			return "The map is revealed."
		}},
		{"unlockall", []string{"unlock", "openall"}, "unlockall", "reveal the map, open every lock here, max skills, learn every tongue", true, func(g *Game, a []string) string {
			cnReveal(g)
			n := 0
			z := g.Cur
			for y := 0; y < MH; y++ {
				for x := 0; x < MW; x++ {
					if z.At(x, y) == TLocked {
						z.Set(x, y, TDoorOpen)
						n++
					}
				}
			}
			for _, id := range SkillIDs {
				g.Skills[id] = maxi(g.Skills[id], 3)
			}
			tongues := 0
			if st := g.lx(); st != nil {
				for _, cu := range g.W.Cultures {
					st.Flu[cu.Name] = 100
					tongues++
				}
			}
			g.recompute()
			return fmt.Sprintf("Map revealed, %d locks opened, skills at rank 3, %d tongues learned.", n, tongues)
		}},
		{"goto", []string{"tp", "teleport"}, "goto <town|dungeon|tower|<wx> <wy>>", "teleport to the nearest city, dungeon or tower, or to a parasang", true, cnGoto},
		{"warp", nil, "warp <wx> <wy> <zx> <zy> <z>", "jump straight into a zone", true, func(g *Game, a []string) string {
			v := make([]int, 5)
			for i := range v {
				n, ok := cnInt(a, i)
				if !ok {
					return "usage: warp <wx> <wy> <zx> <zy> <z>"
				}
				v[i] = n
			}
			g.OnWorld = false
			g.Warp(v[0], v[1], clampi(v[2], 0, 2), clampi(v[3], 0, 2), v[4])
			return fmt.Sprintf("Warped to %s.", g.Cur.Name)
		}},
		{"spawn", []string{"summon"}, "spawn <creature> [count] [level]", "conjure creatures beside you; `spawn list` lists the ids", true, cnSpawn},
		{"give", []string{"item"}, "give <kind> [rarity]", "conjure an item: longblade shortblade axe cudgel ranged armor backpack oddity tonic light loot; rarity common..unique", true, func(g *Game, a []string) string {
			if len(a) == 0 {
				return "usage: give <kind> [rarity]"
			}
			rar := "common"
			if len(a) > 1 {
				rar = strings.ToLower(a[1])
			}
			it := g.sbItem(strings.ToLower(a[0]), rar)
			if it == nil {
				return "nothing came of that"
			}
			g.give(it)
			return "You receive " + it.Display() + "."
		}},
		{"time", []string{"hour", "settime"}, "time <0..23>", "move the clock to that hour (it moves forward)", true, func(g *Game, a []string) string {
			h, ok := cnInt(a, 0)
			if !ok || h < 0 || h > 23 {
				return "usage: time <0..23>"
			}
			cur := (g.Turn + startMinutes) % DayLen
			g.Turn += (h*60 - cur + DayLen) % DayLen
			g.recompute()
			return fmt.Sprintf("It is %02d:00.", h)
		}},
		{"day", []string{"skipdays", "wait"}, "day <n>", "skip n days forward (the world catches up)", true, func(g *Game, a []string) string {
			n, ok := cnInt(a, 0)
			if !ok || n < 1 || n > 400 {
				return "usage: day <1..400>"
			}
			g.Turn += n * DayLen
			return fmt.Sprintf("%d days pass.", n)
		}},
		{"weather", nil, "weather <id|off>", "force the weather (ids: " + cnWeatherIDs() + ")", true, func(g *Game, a []string) string {
			if len(a) == 0 {
				return "usage: weather <id|off>"
			}
			if strings.ToLower(a[0]) == "off" {
				g.wxForce = nil
				return "The weather is natural again."
			}
			for i := range weatherKinds {
				if weatherKinds[i].ID == strings.ToLower(a[0]) {
					k := weatherKinds[i]
					g.wxForce = &Weather{Kind: &k, Temp: 15, Season: g.Season()}
					return "The weather: " + k.Name + "."
				}
			}
			return "unknown weather; ids: " + cnWeatherIDs()
		}},
		{"pos", []string{"where", "seed"}, "pos", "where you are, the seed and the clock", false, func(g *Game, a []string) string {
			return fmt.Sprintf("%s · parasang %d,%d zone %d,%d z=%d · tile %d,%d · seed %s · day %d turn %d", g.Cur.Name, g.Cur.WX, g.Cur.WY, g.Cur.ZX, g.Cur.ZY, g.Cur.Z, g.P.X, g.P.Y, g.Seed, g.Day(), g.Turn)
		}},
	}
}

func cnWeatherIDs() string {
	var ids []string
	for _, k := range weatherKinds {
		ids = append(ids, k.ID)
	}
	return strings.Join(ids, " ")
}

func cnOnOff(b bool) string {
	if b {
		return "on"
	}
	return "off"
}

func cnInt(a []string, i int) (int, bool) {
	if i >= len(a) {
		return 0, false
	}
	n, err := strconv.Atoi(a[i])
	return n, err == nil
}

func cnReveal(g *Game) {
	for k := range g.Cur.Seen {
		g.Cur.Seen[k] = 1
	}
	for p := range g.W.Biome {
		g.Known[p] = true
	}
}

func cnRep(g *Game, a []string) string {
	if len(a) < 2 {
		return "usage: rep <home|all|faction> <n>"
	}
	n, ok := cnInt(a, 1)
	if !ok {
		return "usage: rep <home|all|faction> <n>"
	}
	switch strings.ToLower(a[0]) {
	case "home":
		g.Rep[stateFaction(g.HomeState)] = n
		return fmt.Sprintf("Reputation with your home realm: %d.", n)
	case "all":
		for k := range g.Rep {
			g.Rep[k] = n
		}
		for _, s := range g.W.States {
			g.Rep[stateFaction(s.ID)] = n
		}
		for _, r := range g.W.Religions {
			g.Rep["faith:"+strconv.Itoa(r.ID)] = n
		}
		return fmt.Sprintf("Reputation with everyone: %d.", n)
	}
	g.Rep[a[0]] = n
	return fmt.Sprintf("Reputation with %s: %d.", a[0], n)
}

func cnGoto(g *Game, a []string) string {
	if len(a) == 0 {
		return "usage: goto <town|dungeon|tower> or goto <wx> <wy>"
	}
	switch strings.ToLower(a[0]) {
	case "town", "city":
		sbRun(g, 13)
		return "You arrive."
	case "dungeon":
		sbRun(g, 11)
		return "You arrive."
	case "tower":
		sbRun(g, 12)
		return "You arrive."
	}
	x, ok1 := cnInt(a, 0)
	y, ok2 := cnInt(a, 1)
	if !ok1 || !ok2 || !g.W.In(x, y) {
		return "usage: goto <town|dungeon|tower> or goto <wx> <wy>"
	}
	g.sbJump(g.W.Idx(x, y), 1, 1, 0)
	return "You arrive at " + g.Cur.Name + "."
}

func cnSpawn(g *Game, a []string) string {
	if len(a) == 0 {
		return "usage: spawn <creature> [count] [level]"
	}
	if a[0] == "list" {
		var ids []string
		for _, t := range Templates {
			ids = append(ids, t.ID)
		}
		sort.Strings(ids)
		return strings.Join(ids, " ")
	}
	id := strings.ToLower(a[0])
	if tplByID[id] == nil {
		return "unknown creature (try `spawn list`)"
	}
	n, lvl := 1, maxi(1, g.P.Level)
	if v, ok := cnInt(a, 1); ok {
		n = clampi(v, 1, 20)
	}
	if v, ok := cnInt(a, 2); ok {
		lvl = clampi(v, 1, 60)
	}
	made := 0
	for i := 0; i < n; i++ {
		x, y, ok := cnFreeNear(g)
		if !ok {
			break
		}
		m := g.Spawn(id, lvl, g.R)
		m.X, m.Y = x, y
		g.Cur.Actors = append(g.Cur.Actors, m)
		made++
	}
	g.recompute()
	return fmt.Sprintf("%d x %s appear.", made, id)
}

// cnFreeNear finds a free walkable tile near the player.
func cnFreeNear(g *Game) (int, int, bool) {
	for r := 1; r <= 6; r++ {
		for dy := -r; dy <= r; dy++ {
			for dx := -r; dx <= r; dx++ {
				x, y := g.P.X+dx, g.P.Y+dy
				if g.Cur.In(x, y) && g.Cur.Walkable(x, y) && g.Cur.ActorAt(x, y) == nil && !(x == g.P.X && y == g.P.Y) {
					return x, y, true
				}
			}
		}
	}
	return 0, 0, false
}

// cnNoclip is the one-line hook at the top of MoveP: with noclip on, step through anything inside the zone.
func cnNoclip(g *Game, dx, dy int) bool {
	st := g.cn()
	if !st.Noclip || g.Cur == nil || g.P == nil {
		return false
	}
	nx, ny := g.P.X+dx, g.P.Y+dy
	if !g.Cur.In(nx, ny) || g.Cur.ActorAt(nx, ny) != nil {
		return false // zone edges and creatures behave normally
	}
	g.P.X, g.P.Y = nx, ny
	g.spend(1000)
	g.recompute()
	return true
}

func cnHelp(g *Game, a []string) string {
	if len(a) > 0 {
		for _, c := range cnCmds {
			if c.name == strings.ToLower(a[0]) || cnHas(c.alias, strings.ToLower(a[0])) {
				al := ""
				if len(c.alias) > 0 {
					al = "  (also: " + strings.Join(c.alias, ", ") + ")"
				}
				return "{{W|" + c.usage + "}}  " + c.help + al
			}
		}
		return "no such command"
	}
	var names []string
	for _, c := range cnCmds {
		names = append(names, c.name)
	}
	return "Commands: " + strings.Join(names, " ") + ". `help <command>` explains one. Tab completes, Up recalls."
}

func cnHas(l []string, s string) bool {
	for _, x := range l {
		if x == s {
			return true
		}
	}
	return false
}

func cnFind(name string) *cnCmd {
	name = strings.ToLower(name)
	for i := range cnCmds {
		if cnCmds[i].name == name || cnHas(cnCmds[i].alias, name) {
			return &cnCmds[i]
		}
	}
	return nil
}

func cnExec(g *Game, line string) {
	st := g.cn()
	st.out = append(st.out, "{{Y|> "+line+"}}")
	f := strings.Fields(line)
	if len(f) == 0 {
		return
	}
	c := cnFind(f[0])
	if c == nil {
		st.out = append(st.out, "{{R|Unknown command.}} Type help.")
		return
	}
	res := c.run(g, f[1:])
	if c.cheat {
		g.sb().Used = true // marks the run: score 0 in the hall of fame
	}
	if res != "" {
		for _, l := range Wrap(res, SW-10) {
			st.out = append(st.out, l)
		}
	}
	if len(st.out) > 200 {
		st.out = st.out[len(st.out)-200:]
	}
	if g.Cur != nil && g.P != nil {
		g.recompute()
	}
}

func cnKey(g *Game, k string) {
	st := g.cn()
	switch k {
	case "Escape":
		g.Mode = "main"
	case "Enter":
		line := strings.TrimSpace(st.buf)
		st.buf = ""
		if line != "" {
			if n := len(st.Hist); n == 0 || st.Hist[n-1] != line {
				st.Hist = append(st.Hist, line)
				if len(st.Hist) > 40 {
					st.Hist = st.Hist[len(st.Hist)-40:]
				}
			}
			st.hi = len(st.Hist)
			cnExec(g, line)
		}
	case "Backspace":
		if rs := []rune(st.buf); len(rs) > 0 {
			st.buf = string(rs[:len(rs)-1])
		}
	case "ArrowUp":
		if st.hi > 0 {
			st.hi--
			st.buf = st.Hist[st.hi]
		}
	case "ArrowDown":
		if st.hi < len(st.Hist)-1 {
			st.hi++
			st.buf = st.Hist[st.hi]
		} else {
			st.hi, st.buf = len(st.Hist), ""
		}
	case "Tab":
		if !strings.Contains(st.buf, " ") {
			var hit []string
			for _, c := range cnCmds {
				if strings.HasPrefix(c.name, strings.ToLower(st.buf)) {
					hit = append(hit, c.name)
				}
			}
			if len(hit) == 1 {
				st.buf = hit[0] + " "
			} else if len(hit) > 1 {
				st.out = append(st.out, "{{K|"+strings.Join(hit, " ")+"}}")
			}
		}
	default:
		if rs := []rune(k); len(rs) == 1 && rs[0] >= 32 && rs[0] < 127 && len(st.buf) < 90 {
			st.buf += k
		}
	}
}

func cnDraw(g *Game) {
	s := g.sc
	st := g.cn()
	h := 22
	y0 := 1
	s.Box(2, y0, SW-4, h, CDarkCyan, "Console")
	rows := h - 4
	out := st.out
	if len(out) > rows {
		out = out[len(out)-rows:]
	}
	for i, l := range out {
		s.TextB(4, y0+1+i, l, CGray, CBgDark, SW-4)
	}
	s.TextB(4, y0+h-2, "{{W|> }}{{W|"+st.buf+"}}{{Y|_}}", CGray, CBgDark, SW-4)
}
