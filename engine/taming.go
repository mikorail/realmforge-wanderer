package engine

import (
	"fmt"
	"sort"
	"strings"
)

// Taming (prefix tm): feed wild or wounded bestiary animals over several attempts until they trust
// you, then keep them as beast companions or (rideable species) as mounts through transport.go.
// Ostlers also sell bred beasts. State g.ext("tm"). Flavour never draws g.R (hashes only); dice for
// checks at play time use g.R like every other check.

type tmProg struct {
	Trust int  `json:"t"`
	Last  int  `json:"l"`           // turn of the last feeding
	Seen  int  `json:"s"`           // turn of the last contact
	Tries int  `json:"n"`           // attempts so far
	Calm  int  `json:"c,omitempty"` // turn until which a calming word helps
	Cowed bool `json:"w,omitempty"` // won by dominance: less loyal
}

type tmRide struct {
	Species string  `json:"sp"`
	Given   string  `json:"gn,omitempty"`
	Level   int     `json:"lv"`
	XP      int     `json:"xp,omitempty"`
	Loyalty int     `json:"lo"`
	Stam    float64 `json:"st"` // flight minutes left (flying mounts)
	At      int     `json:"at"` // turn stamina was last updated
	Starve  float64 `json:"sv,omitempty"`
	Rides   int     `json:"r,omitempty"`
}

type tmState struct {
	Prog  map[string]*tmProg `json:"p,omitempty"`
	Fed   map[int]int        `json:"f,omitempty"` // tamed beast actor id → turn last fed
	Ride  *tmRide            `json:"r,omitempty"`
	Tamed int                `json:"n,omitempty"`

	nameBuf string // text entry (not saved)
	nameFor int    // actor id being renamed, 0 = the mount
}

func (g *Game) tm() *tmState {
	st := g.ext("tm", func() any { return &tmState{} }).(*tmState)
	if st.Prog == nil {
		st.Prog = map[string]*tmProg{}
	}
	if st.Fed == nil {
		st.Fed = map[int]int{}
	}
	return st
}

func init() {
	talkHooks = append(talkHooks, tmTalk)
	turnHooks = append(turnHooks, tmTurn)
	killHooks = append(killHooks, tmKill)
	sideHooks = append(sideHooks, tmSide)
	modeKeys["tmname"] = tmNameKey
	modeDraws["tmname"] = tmNameDraw
	tmRegisterMounts()
}

// ---- species rules ----

func tmHasAny(s string, words []string) bool {
	for _, w := range words {
		if strings.Contains(s, w) {
			return true
		}
	}
	return false
}

// tmRideClass says how a species can be ridden: "ground", "fly", "swim" or "" (not at all).
func tmRideClass(s *bsSp) string {
	if s.Body == "humanoid" {
		return ""
	}
	switch s.Group {
	case "mam", "syn", "bird", "rept", "dino", "ptero":
	default:
		return ""
	}
	if s.Diet == 'f' || s.Diet == 'i' {
		return ""
	}
	if s.Aquatic() {
		if s.Fresh && s.Size >= 4 && (s.Group == "mam" || s.Group == "rept" || s.Group == "dino") {
			return "swim"
		}
		return ""
	}
	if s.Has("fly") {
		if s.Size >= 5 {
			return "fly"
		}
		return ""
	}
	if s.Size < 4 {
		return ""
	}
	if s.Fresh && s.Size >= 5 && (s.Group == "mam" || s.Group == "rept" || s.Group == "dino") {
		return "swim"
	}
	switch s.Diet {
	case 'h', 'o':
		return "ground"
	case 'c':
		if (s.Group == "bird" || s.Group == "dino" || s.Group == "rept") && s.Size >= 5 {
			return "ground"
		}
	}
	return ""
}

func tmTameable(s *bsSp) bool {
	if s == nil {
		return false
	}
	switch s.Group {
	case "mam", "syn", "bird", "rept", "dino", "ptero", "amph":
	default:
		return false
	}
	if s.Diet == 'f' || s.Size < 2 {
		return false
	}
	if s.Aquatic() {
		return tmRideClass(s) == "swim"
	}
	return true
}

// tmMountOnly: species that cannot follow on land and so only ever become mounts.
func tmMountOnly(s *bsSp) bool { return s.Aquatic() }

func tmWild(a *Actor) *bsSp {
	if a == nil || a.HP <= 0 || a.AI == "follower" || !strings.HasPrefix(a.Tpl, "bs_") {
		return nil
	}
	s := bsByTpl[a.Tpl]
	if !tmTameable(s) {
		return nil
	}
	return s
}

func tmWounded(a *Actor) bool { return a.MaxHP > 0 && a.HP*10 < a.MaxHP*6 }

// ---- food ----

// tmFoodClass sorts a pack item: plant, meat, fish, grub, dish or "".
func tmFoodClass(it *Item) string {
	if it == nil {
		return ""
	}
	n := " " + strings.ToLower(it.Name)
	switch it.Kind {
	case "corpse":
		if tmHasAny(n, tmFishWords) {
			return "fish"
		}
		return "meat"
	case "dish":
		return "dish"
	case "ingredient", "food":
		switch {
		case tmHasAny(n, tmFishWords):
			return "fish"
		case tmHasAny(n, tmGrubWords):
			return "grub"
		case tmHasAny(n, tmMeatWords):
			return "meat"
		}
		return "plant"
	}
	return ""
}

// tmFoodQ is how much a species likes an item: 0 refuses, 1 accepts, 2 favourite.
func tmFoodQ(s *bsSp, it *Item) int {
	c := tmFoodClass(it)
	if c == "" {
		return 0
	}
	if c == "dish" {
		meaty := tmHasAny(" "+strings.ToLower(it.Name), tmMeatWords) || tmHasAny(" "+strings.ToLower(it.Name), tmFishWords)
		switch s.Diet {
		case 'o':
			return 2
		case 'h':
			if !meaty {
				return 1
			}
		case 'c', 's':
			if meaty {
				return 1
			}
		}
		return 0
	}
	herb := it.Kind == "ingredient" && it.Glyph == '♣'
	switch s.Diet {
	case 'h':
		if c == "plant" {
			return 1 + boolInt(herb || it.Value >= 12)
		}
	case 'c':
		switch c {
		case "meat":
			return 2
		case "fish":
			return 1
		}
	case 'o':
		return 1 + boolInt(c == "meat" && it.Kind == "ingredient")
	case 'i':
		if c == "grub" {
			return 2
		}
	case 'p':
		if c == "fish" {
			return 2
		}
	case 's':
		switch c {
		case "meat":
			return 2
		case "fish", "grub":
			return 1
		}
	}
	return 0
}

type tmFood struct {
	It *Item
	Q  int
}

func (g *Game) tmFoods(s *bsSp) []tmFood {
	var out []tmFood
	seen := map[string]bool{}
	for _, it := range g.P.Inv {
		q := tmFoodQ(s, it)
		if q == 0 || seen[it.Name+it.Kind] {
			continue
		}
		seen[it.Name+it.Kind] = true
		out = append(out, tmFood{it, q})
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Q > out[j].Q })
	if len(out) > 9 {
		out = out[:9]
	}
	return out
}

func tmQWord(q int) string {
	if q >= 2 {
		return "a favourite"
	}
	return "will do"
}

// ---- difficulty ----

func (g *Game) tmNeed(s *bsSp, a *Actor) int {
	n := 3
	if s.Size >= 5 {
		n++
	}
	if s.Size >= 7 {
		n++
	}
	switch s.Temper {
	case "predator", "pack":
		n += 2
	case "territorial", "skittish":
		n++
	}
	if s.Extinct() {
		n++
	}
	if s.Rarity >= 4 {
		n++
	}
	if tmWounded(a) {
		n--
	}
	return clampi(n, 2, 9)
}

func (g *Game) tmDC(s *bsSp, a *Actor, q int) int {
	dc := dcEasy + s.Size/2 + s.Rarity/3
	switch s.Temper {
	case "passive":
		dc--
	case "skittish":
		dc++
	case "territorial":
		dc += 2
	case "predator", "pack":
		dc += 3
	}
	if s.Extinct() {
		dc++
	}
	if tmWounded(a) {
		dc -= 2
	}
	if q >= 2 {
		dc--
	}
	return clampi(dc, dcTrivial, dcLegendary)
}

func tmKey(a *Actor) string { return fmt.Sprintf("%s:%d", a.Tpl, a.ID) }

func (g *Game) tmProgOf(a *Actor) *tmProg {
	st := g.tm()
	p := st.Prog[tmKey(a)]
	if p == nil {
		p = &tmProg{}
		st.Prog[tmKey(a)] = p
	}
	p.Seen = g.Turn
	return p
}

// tmWhyNot is "" when the animal can be approached, else the reason it cannot.
func (g *Game) tmWhyNot(a *Actor) string {
	if g.Hostile(a) && !tmWounded(a) {
		return "It is far too wild and hale to come near. Weaken it first, until it is hurt and hungry for peace."
	}
	if a.Angry && a.HP*2 > a.MaxHP {
		return "It is enraged and wants only your blood. Let it tire, or leave it be."
	}
	return ""
}

func (g *Game) tmLook(a *Actor, s *bsSp) string {
	t := fmt.Sprintf("%s. %s", g.The(a), tmDietLine[s.Diet])
	if tmWounded(a) {
		t += " It is hurt, and weak with it."
	}
	if why := g.tmWhyNot(a); why != "" {
		t += "\n\n" + why
	}
	return t
}

func tmBar(n, need int) string {
	b := strings.Repeat("#", clampi(n, 0, need)) + strings.Repeat("-", maxi(0, need-n))
	return "{{K|Trust [}}{{G|" + b + "}}{{K|]}}"
}

// ---- the wild-animal menu ----

// tmOpenTalk takes over bumping a tameable wild animal (one line in openTalk).
func (g *Game) tmOpenTalk(a *Actor) bool {
	if tmWild(a) == nil {
		return false
	}
	g.Talk, g.Mode, g.Sel, g.Popup = a, "talk", 0, nil
	g.tmBeastMenu(a, "")
	return true
}

func (g *Game) tmBeastMenu(a *Actor, text string) {
	s := tmWild(a)
	if s == nil {
		g.Mode, g.Talk = "main", nil
		return
	}
	p := g.tmProgOf(a)
	need := g.tmNeed(s, a)
	if text == "" {
		text = g.tmLook(a, s)
	}
	g.Talk, g.Mode = a, "talk"
	g.Popup = Wrap(text+"\n\n"+tmBar(p.Trust, need), 100)
	g.Opts = nil
	g.Sel = 0
	add := func(t string, f func()) { g.Opts = append(g.Opts, Option{t, f}) }
	if g.tmWhyNot(a) == "" {
		foods := g.tmFoods(s)
		for _, f := range foods {
			f := f
			dc := g.tmDC(s, a, f.Q)
			add(fmt.Sprintf("%s {{G|Offer %s}} {{K|(%s)}}", g.checkLabel("survival", dc), f.It.Display(), tmQWord(f.Q)), func() { g.tmFeed(a, s, f.It, f.Q) })
		}
		if len(foods) == 0 {
			add("{{K|(You carry nothing it would eat.)}}", func() { g.tmBeastMenu(a, "") })
		}
		add(g.checkLabel("persuasion", g.tmDC(s, a, 0)-1)+" {{C|Calm it with your voice.}}", func() { g.tmCalm(a, s) })
		add(g.checkLabel("intimidate", g.tmDC(s, a, 0)+1)+" {{R|Stare it down.}}", func() { g.tmStare(a, s) })
	}
	add("Look it over.", func() { g.tmBeastMenu(a, s.Lore()+"\n\n"+g.tmLook(a, s)) })
	add("Back away.", func() { g.Mode, g.Talk = "main", nil })
}

// tmRoll rolls a check and returns success and the margin over the difficulty.
func (g *Game) tmRoll(kind string, dc, bonus int) (bool, int) {
	tot := g.R.Dice(2, 6) + g.checkMod(kind) + bonus
	return tot >= dc, tot - dc
}

func (g *Game) tmPick(lines []string, a *Actor, salt int) string {
	r := NewRng(fmt.Sprintf("%s:tm:say:%d:%d", g.Seed, a.ID, salt))
	return fmt.Sprintf(Pick(r, lines), g.The(a))
}

func (g *Game) tmSlotFree() bool { return g.companionCount() < maxCompanions }

func (g *Game) tmFeed(a *Actor, s *bsSp, it *Item, q int) {
	p := g.tmProgOf(a)
	need := g.tmNeed(s, a)
	if g.Turn-p.Last < 15 && p.Last > 0 {
		g.tmBeastMenu(a, g.The(a)+" has eaten and wants no more for now. Give it a little while.")
		return
	}
	gain := 1
	dc := g.tmDC(s, a, q)
	bonus := boolInt(p.Calm > g.Turn) * 2
	if p.Trust+gain >= need && !tmMountOnly(s) && !g.tmSlotFree() {
		g.tmBeastMenu(a, g.The(a)+" would follow you, but you already lead as many companions as you can keep.")
		return
	}
	if p.Trust+gain >= need && tmMountOnly(s) && g.tr().Mount != nil {
		g.tmBeastMenu(a, g.The(a)+" would carry you, but you already keep a mount. Stable or sell it first.")
		return
	}
	ok, margin := g.tmRoll("survival", dc, bonus)
	p.Tries++
	p.Last = g.Turn
	g.spend(1000)
	if g.Mode == "dead" || a.HP <= 0 {
		return
	}
	if ok {
		g.takeOne(it)
		if margin >= 4 {
			gain++
		}
		p.Trust += gain
		g.gainXP(2 + dc/2)
		if p.Trust >= need {
			g.tmTame(a, s, p)
			return
		}
		g.tmBeastMenu(a, g.tmPick(tmEatLines, a, p.Tries))
		return
	}
	switch {
	case margin <= -4:
		p.Trust = maxi(0, p.Trust-1)
		g.Msg("{{R|" + g.tmPick(tmSnapLines, a, p.Tries) + "}}")
		g.provoke(a)
		g.Mode, g.Talk = "main", nil
	case margin <= -2:
		p.Trust = maxi(0, p.Trust-1)
		g.Msg("{{O|" + g.tmPick(tmSpookLines, a, p.Tries) + "}}")
		if s.Temper == "skittish" || s.Temper == "herd" {
			g.bs().alarm[a.ID] = g.Turn + 25
		}
		g.Mode, g.Talk = "main", nil
	default:
		g.tmBeastMenu(a, g.The(a)+" ignores the offering. Try again, more slowly.")
	}
}

func (g *Game) tmCalm(a *Actor, s *bsSp) {
	p := g.tmProgOf(a)
	g.spend(1000)
	if g.Mode == "dead" || a.HP <= 0 {
		return
	}
	ok, margin := g.tmRoll("persuasion", g.tmDC(s, a, 0)-1, 0)
	if ok {
		p.Calm = g.Turn + 40
		g.gainXP(3)
		g.tmBeastMenu(a, g.tmPick(tmCalmLines, a, p.Tries+1)+" {{K|(feeding goes easier for a while)}}")
		return
	}
	if margin <= -4 {
		g.Msg("{{O|" + g.tmPick(tmSpookLines, a, p.Tries+7) + "}}")
		if s.Temper == "skittish" || s.Temper == "herd" {
			g.bs().alarm[a.ID] = g.Turn + 25
		}
		g.Mode, g.Talk = "main", nil
		return
	}
	g.tmBeastMenu(a, g.The(a)+" does not seem to care for your voice.")
}

func (g *Game) tmStare(a *Actor, s *bsSp) {
	p := g.tmProgOf(a)
	need := g.tmNeed(s, a)
	g.spend(1000)
	if g.Mode == "dead" || a.HP <= 0 {
		return
	}
	ok, margin := g.tmRoll("intimidate", g.tmDC(s, a, 0)+1, 0)
	if ok {
		p.Trust++
		p.Cowed = true
		g.gainXP(4)
		if p.Trust >= need && (tmMountOnly(s) || g.tmSlotFree()) && !(tmMountOnly(s) && g.tr().Mount != nil) {
			g.tmTame(a, s, p)
			return
		}
		p.Trust = mini(p.Trust, need-1)
		g.tmBeastMenu(a, g.The(a)+" drops its eyes and gives way. It obeys, but it does not love you for it.")
		return
	}
	if margin <= -2 {
		g.Msg("{{R|" + g.tmPick(tmSnapLines, a, p.Tries+3) + "}}")
		g.provoke(a)
		g.Mode, g.Talk = "main", nil
		return
	}
	g.tmBeastMenu(a, g.The(a)+" holds your gaze, unimpressed.")
}

// ---- becoming a companion ----

func (g *Game) tmPetName(a *Actor) string {
	return tmPetNames[int(HashStr(fmt.Sprintf("%s:tm:name:%d", g.Seed, a.ID))%uint64(len(tmPetNames)))]
}

// tmJoin makes a spawned or tamed animal a companion with the given loyalty.
func (g *Game) tmJoin(a *Actor, s *bsSp, loyalty int, homeKey string) {
	c := &Companion{Faction: a.Faction, AI: "wander", F: a.F, HomeKey: homeKey, HomeX: a.X, HomeY: a.Y, Loyalty: clampi(loyalty, 5, 100), Since: g.Turn}
	a.Comp = c
	a.Order = OrdFollow
	a.AI = "follower"
	a.Faction = "player"
	a.F = CGreen
	a.Angry = false
	a.Set("tm")
	a.Name, a.Proper, a.Title = g.tmPetName(a), true, "tamed "+s.Display()
	g.tm().Fed[a.ID] = g.Turn
}

func (g *Game) tmTame(a *Actor, s *bsSp, p *tmProg) {
	st := g.tm()
	delete(st.Prog, tmKey(a))
	st.Tamed++
	g.gainXP(10*s.Size + 15*s.Rarity)
	lo := 45 + 5*g.Skills["persuasion"] + g.Skills["survival"]*3
	if p.Cowed {
		lo -= 15
	}
	if tmMountOnly(s) {
		g.removeActor(g.Cur, a)
		g.tmSetMount(s, "", maxi(1, a.Level), lo, 0)
		g.Msg(fmt.Sprintf("{{G|The %s lets you climb onto its back. You have a new mount.}}", s.Display()))
		g.Mode, g.Talk = "main", nil
		return
	}
	g.tmJoin(a, s, lo, g.Cur.Key)
	g.Msg(fmt.Sprintf("{{G|%s the %s is yours. It follows you.}} {{K|(o: companion orders; talk to it to feed or rename it)}}", a.Name, s.Display()))
	if s.Rarity >= 4 || s.Extinct() || s.Size >= 6 {
		g.deed("companion", fmt.Sprintf("The wanderer %s tamed %s %s and took it for a companion.", g.P.Name, bsArticle(s.Display()), s.Display()), 0, 0)
	}
	g.Mode, g.Talk = "main", nil
}

// ---- companion talk (one line in companionTalk) ----

func (g *Game) tmCompOptions(a *Actor) []Option {
	if !a.Has("tm") || a.AI != "follower" {
		return nil
	}
	s := bsByTpl[a.Tpl]
	if s == nil {
		return nil
	}
	var out []Option
	out = append(out, Option{"{{G|Feed it.}} {{K|(" + tmDietShort(s) + ")}}", func() { g.tmFeedMenu(a, s) }})
	out = append(out, Option{"{{W|Give it a name.}}", func() { g.tmNameStart(a.ID, a.Name) }})
	if tmRideClass(s) != "" && tmRideClass(s) != "swim" {
		out = append(out, Option{"{{B|Saddle up: ride it as a mount.}}", func() { g.tmSaddle(a, s) }})
	}
	return out
}

func tmDietShort(s *bsSp) string { return bsDietNames[s.Diet] }

func (g *Game) tmFeedMenu(a *Actor, s *bsSp) {
	st := g.tm()
	if g.Turn-st.Fed[a.ID] < 180 && st.Fed[a.ID] > 0 {
		g.companionTalk(a, a.Name+" is still full and nudges your hand away.")
		return
	}
	g.Popup = Wrap("What will you feed "+a.Name+"?", 100)
	g.Opts = nil
	g.Sel = 0
	for _, f := range g.tmFoods(s) {
		f := f
		g.Opts = append(g.Opts, Option{fmt.Sprintf("%s {{K|(%s)}}", f.It.Display(), tmQWord(f.Q)), func() {
			g.takeOne(f.It)
			st.Fed[a.ID] = g.Turn
			a.HP = mini(a.MaxHP, a.HP+4+2*s.Size)
			g.addLoyalty(a, 4+4*f.Q)
			g.companionTalk(a, fmt.Sprintf("%s eats with relish and leans against you. {{K|(loyalty %d)}}", a.Name, g.comp(a).Loyalty))
		}})
	}
	if len(g.Opts) == 0 {
		g.companionTalk(a, "You carry nothing it would eat.")
		return
	}
	g.Opts = append(g.Opts, Option{"Never mind.", func() { g.companionTalk(a, g.compGreeting(a)) }})
}

// ---- renaming ----

func (g *Game) tmNameStart(id int, cur string) {
	st := g.tm()
	st.nameBuf, st.nameFor = "", id
	g.Mode = "tmname"
}

func tmNameKey(g *Game, k string) {
	st := g.tm()
	switch k {
	case "Escape":
		tmNameDone(g, false)
	case "Enter":
		tmNameDone(g, true)
	case "Backspace":
		if rs := []rune(st.nameBuf); len(rs) > 0 {
			st.nameBuf = string(rs[:len(rs)-1])
		}
	default:
		if rs := []rune(k); len(rs) == 1 && rs[0] >= 32 && rs[0] < 127 && len([]rune(st.nameBuf)) < 18 {
			st.nameBuf += k
		}
	}
}

func tmNameDone(g *Game, ok bool) {
	st := g.tm()
	name := strings.TrimSpace(st.nameBuf)
	id := st.nameFor
	g.Mode, g.Talk = "main", nil
	if id == 0 {
		if ok && name != "" && st.Ride != nil && g.tr().Mount != nil {
			st.Ride.Given = name
			g.tr().Mount.Name = g.tmMountLabel(st.Ride)
			g.Msg(fmt.Sprintf("Your mount answers to %s now.", name))
		}
		return
	}
	a := g.findActor(id)
	if a == nil {
		return
	}
	if ok && name != "" {
		a.Name, a.Proper = name, true
		g.Msg(fmt.Sprintf("You name your companion {{G|%s}}.", name))
	}
	g.openTalk(a)
}

func tmNameDraw(g *Game) {
	s := g.sc
	st := g.tm()
	h := 7
	y0 := (MH - h) / 2
	s.Box(20, y0, 56, h, CYellow, "Give it a name")
	s.TextB(22, y0+2, "{{W|"+st.nameBuf+"}}{{Y|_}}", CGray, CBgDark, 74)
	s.TextB(22, y0+4, "{{K|type a name · Enter confirm · Esc cancel}}", CGray, CBgDark, 74)
}

// ---- ability: befriend a beast (hook in Abilities / usePower) ----

func (g *Game) tmAbilities() []Ability {
	var out []Ability
	if g.Cur == nil || g.P == nil {
		return nil
	}
	for _, a := range g.Cur.Actors {
		if tmWild(a) != nil && dist(a, g.P) <= 6 && g.visible(a.X, a.Y) {
			out = append(out, Ability{ID: "tm_befriend", Name: "Befriend a beast", Target: "tile", Cool: 0})
			break
		}
	}
	if g.tmRideRec() != nil {
		out = append(out, Ability{ID: "tm_mount", Name: "Tend your mount", Target: "self"})
	}
	return out
}

func (g *Game) tmUse(id string, tx, ty int) {
	switch id {
	case "tm_befriend":
		a := g.Cur.ActorAt(tx, ty)
		if a == nil || tmWild(a) == nil {
			g.Msg("There is no beast there you could befriend.")
			return
		}
		if dist(a, g.P) > 3 {
			g.Msg("{{K|Too far away. Walk up to it.}}")
			return
		}
		g.Talk, g.Mode, g.Sel = a, "talk", 0
		g.tmBeastMenu(a, "")
	case "tm_mount":
		g.tmMountMenu()
	}
}

// ---- mounts ----

func tmRegisterMounts() {
	for _, block := range bsData {
		for _, line := range strings.Split(block, "\n") {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			s, err := bsParse(line)
			if err != nil || !tmTameable(s) {
				continue
			}
			if c := tmRideClass(s); c != "" {
				trMounts = append(trMounts, tmMountDef(s, c))
			}
		}
	}
}

func tmMountDef(s *bsSp, class string) trMountDef {
	speed := bsTemplate(s).Speed
	carry := [8]int{0, 0, 0, 0, 25, 40, 70, 100}[s.Size]
	pace := float64(1.25) + float64(speed-85)*0.02
	rough := 0.45 + 0.05*float64(s.Size)
	if s.Mtn {
		rough += 0.2
	}
	if s.Has("armored") {
		rough -= 0.1
	}
	d := trMountDef{ID: "tm_" + s.ID, Name: s.Display(), Feed: "fodder", Genres: []string{"tm-only"}, Carry: carry}
	switch class {
	case "fly":
		pace = 1.7 + float64(speed-110)*0.02
		d.Carry = carry * 7 / 10
		rough = 1
	case "swim":
		pace, rough = 1, 1
	}
	d.Pace = clampf(pace, 1.15, 2.1)
	d.Rough = clampf(rough, 0.4, 1)
	d.Price = tmMountPrice(s)
	return d
}

func clampf(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func tmFactor(s *bsSp) float64 {
	f := 1 + 0.3*float64(s.Rarity-1)
	if s.Extinct() {
		f *= 2
	}
	if s.Hostile() {
		f *= 1.5
	}
	return f
}

func tmMountPrice(s *bsSp) int {
	f := tmFactor(s)
	if s.Has("fly") {
		f *= 1.5
	}
	return int(float64(8*s.Size*s.Size+30) * f)
}

func tmBeastPrice(s *bsSp) int { return int(float64(10*s.Size*s.Size+25) * tmFactor(s)) }

func tmMaxStam(s *bsSp) float64 { return float64(1500 + 300*s.Size) }

func (g *Game) tmRideRec() *tmRide {
	st := g.tm()
	m := g.tr().Mount
	if st.Ride == nil || m == nil || m.Def != "tm_"+st.Ride.Species {
		return nil
	}
	return st.Ride
}

func (g *Game) tmMountLabel(r *tmRide) string {
	s := bsByID[r.Species]
	if s == nil {
		return r.Species
	}
	if r.Given != "" {
		return r.Given + " the " + s.Display()
	}
	return s.Display()
}

// tmSetMount makes a tamed or bought beast your mount (old mount is dropped by the caller).
func (g *Game) tmSetMount(s *bsSp, given string, lvl, loyalty, xp int) {
	st := g.tr()
	r := &tmRide{Species: s.ID, Given: given, Level: maxi(1, lvl), XP: xp, Loyalty: clampi(loyalty, 5, 100), Stam: tmMaxStam(s), At: g.Turn}
	st.Mount = &trMount{Def: "tm_" + s.ID, Name: g.tmMountLabel(r), Feed: 5}
	st.Stable = nil
	g.tm().Ride = r
}

func (g *Game) tmSaddle(a *Actor, s *bsSp) {
	if g.tr().Mount != nil {
		g.companionTalk(a, "You already keep a mount. Stable or sell it first.")
		return
	}
	g.removeActor(g.Cur, a)
	c := g.comp(a)
	g.tmSetMount(s, a.Name, a.Level, c.Loyalty, c.XP)
	delete(g.tm().Fed, a.ID)
	g.Msg(fmt.Sprintf("You saddle %s. {{K|(%s)}}", a.Name, g.tr().Mount.Name))
	g.Mode, g.Talk = "main", nil
}

func (g *Game) tmUnsaddle() {
	r := g.tmRideRec()
	if r == nil {
		return
	}
	s := bsByID[r.Species]
	if s == nil || tmMountOnly(s) {
		g.Msg("It will not leave the water.")
		return
	}
	if !g.tmSlotFree() {
		g.Msg("{{y|You already lead as many companions as you can keep.}}")
		return
	}
	rg := NewRng(fmt.Sprintf("%s:tm:unsaddle:%d", g.Seed, g.Turn))
	a := g.bsSpawn(s, r.Level, rg)
	a.X, a.Y = g.P.X, g.P.Y
	g.Cur.Actors = append(g.Cur.Actors, a)
	g.nudge(g.Cur, a)
	g.tmJoin(a, s, r.Loyalty, g.Cur.Key)
	if r.Given != "" {
		a.Name = r.Given
	}
	g.comp(a).XP = r.XP
	g.tr().Mount = nil
	g.tm().Ride = nil
	g.Msg(fmt.Sprintf("%s walks beside you.", a.Name))
}

func (g *Game) tmMountMenu() {
	r := g.tmRideRec()
	if r == nil {
		return
	}
	s := bsByID[r.Species]
	if s == nil {
		return
	}
	g.Talk = &Actor{Name: g.tr().Mount.Name, Glyph: s.Glyph, F: CGreen}
	g.Mode, g.Sel = "talk", 0
	g.Popup = Wrap(fmt.Sprintf("Your %s. Level %d, loyalty %d. %s", g.tr().Mount.Name, r.Level, r.Loyalty, tmDietLine[s.Diet]), 100)
	g.Opts = nil
	add := func(t string, f func()) { g.Opts = append(g.Opts, Option{t, f}) }
	for _, f := range g.tmFoods(s) {
		f := f
		add(fmt.Sprintf("{{G|Give %s}} {{K|(%s)}}", f.It.Display(), tmQWord(f.Q)), func() {
			g.takeOne(f.It)
			m := g.tr().Mount
			m.Feed += 0.5 * float64(f.Q)
			m.Starve = 0
			r.Loyalty = mini(100, r.Loyalty+3+2*f.Q)
			g.Msg(fmt.Sprintf("Your %s eats and nuzzles you. {{K|(loyalty %d)}}", m.Name, r.Loyalty))
			g.tmMountMenu()
		})
	}
	add("{{W|Give it a name.}}", func() { g.tmNameStart(0, r.Given) })
	if !tmMountOnly(s) {
		add("Unsaddle it: let it walk beside you.", func() { g.Mode, g.Talk = "main", nil; g.tmUnsaddle() })
	}
	add("Done.", func() { g.Mode, g.Talk = "main", nil })
}

// tmAdjust refines the mount pace of a bestiary mount (hook in trPaceMul): level bonus, flying
// stamina, and swimmers that only help on rivers.
func (g *Game) tmAdjust(p int, d *trMountDef, mul float64) float64 {
	if d == nil || !strings.HasPrefix(d.ID, "tm_") {
		return mul
	}
	r := g.tmRideRec()
	s := bsByID[strings.TrimPrefix(d.ID, "tm_")]
	if r == nil || s == nil {
		return mul
	}
	boost := 1 + 0.02*float64(clampi(r.Level, 1, 20)-1)
	switch tmRideClass(s) {
	case "fly":
		g.tmRegen(r, s, g.Turn)
		if r.Stam <= 0 {
			return 1
		}
		return 1 + (d.Pace-1)*boost
	case "swim":
		if p >= 0 && p < len(g.W.River) && g.W.River[p] > 0 {
			return 1 + (1.8-1)*boost
		}
		return 1
	}
	return 1 + (mul-1)*boost
}

func (g *Game) tmRegen(r *tmRide, s *bsSp, upTo int) {
	if r.At == 0 || r.At > upTo {
		r.At = upTo
	}
	r.Stam = clampf(r.Stam+0.6*float64(upTo-r.At), 0, tmMaxStam(s))
	r.At = upTo
}

// tmJourney is called after an overland crossing of cost minutes (hook in trJourney).
func (g *Game) tmJourney(cost int) {
	r := g.tmRideRec()
	if r == nil {
		return
	}
	s := bsByID[r.Species]
	if s == nil {
		return
	}
	m := g.tr().Mount
	r.Rides++
	if tmRideClass(s) == "fly" {
		g.tmRegen(r, s, g.Turn-cost)
		was := r.Stam
		r.Stam = maxf(0, r.Stam-float64(cost))
		r.At = g.Turn
		if r.Stam <= 0 && was > 0 {
			g.Msg(fmt.Sprintf("{{y|Your %s is spent and lands. You walk the rest of the way.}}", m.Name))
		} else if r.Stam < tmMaxStam(s)/4 && was >= tmMaxStam(s)/4 {
			g.Msg(fmt.Sprintf("{{y|Your %s is tiring on the wing.}}", m.Name))
		}
	}
	r.XP += cost / 20
	for r.XP >= compXPFor(r.Level) {
		r.XP -= compXPFor(r.Level)
		r.Level++
		g.Msg(fmt.Sprintf("{{G|Your %s grows more seasoned: level %d.}}", m.Name, r.Level))
	}
	if m.Starve > 0 {
		r.Loyalty = maxi(0, r.Loyalty-3)
	} else if r.Rides%3 == 0 {
		r.Loyalty = mini(100, r.Loyalty+1)
	}
	r.Starve = m.Starve
}

func maxf(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

// ---- hooks: talk, turn, kill, side ----

func tmTalk(g *Game, a *Actor) []Option {
	if a == nil {
		return nil
	}
	if s := tmWild(a); s != nil {
		return []Option{{"{{G|Try to win its trust.}}", func() { g.tmBeastMenu(a, "") }}}
	}
	if a.Role != "ostler" {
		return nil
	}
	out := []Option{{"{{B|Ask after bred beasts and rarer mounts.}}", func() { g.tmStockMenu(a) }}}
	if g.tmRideRec() != nil {
		out = append(out, Option{"{{W|See to my beast.}} {{K|(feed, rename, unsaddle)}}", func() { g.tmMountMenu() }})
	}
	return out
}

type tmOffer struct {
	S     *bsSp
	Mount bool
	Price int
}

func (g *Game) tmStock(bd *BurgData) []tmOffer {
	r := NewRng(fmt.Sprintf("%s:tm:ostler:%d", g.Seed, bd.ID))
	b := 0
	if bd.P >= 0 && bd.P < len(g.W.Biome) {
		b = g.W.Biome[bd.P]
	}
	river := bd.P >= 0 && bd.P < len(g.W.River) && g.W.River[bd.P] > 0
	var mounts, pets []*bsSp
	for _, s := range bsAll {
		if !tmTameable(s) || s.Rarity >= 5 {
			continue
		}
		if s.Extinct() && !(tmWildGenres[g.Genre] && (s.Era == "recent" || s.Era == "pleistocene")) {
			continue
		}
		home := false
		for _, x := range s.Biomes {
			home = home || x == b
		}
		home = home || s.Fresh && river
		c := tmRideClass(s)
		if c == "swim" && !river {
			continue
		}
		if c == "fly" && s.Size < 5 {
			continue
		}
		if !home {
			continue
		}
		if c != "" {
			mounts = append(mounts, s)
		}
		if !tmMountOnly(s) && s.Size >= 2 && s.Size <= 5 {
			pets = append(pets, s)
		}
	}
	var out []tmOffer
	take := func(pool []*bsSp, n int, mount bool) {
		for i := 0; i < n && len(pool) > 0; i++ {
			j := r.Int(0, len(pool)-1)
			s := pool[j]
			pool = append(pool[:j:j], pool[j+1:]...)
			price := tmBeastPrice(s)
			if mount {
				price = tmMountPrice(s)
			}
			out = append(out, tmOffer{s, mount, price})
		}
	}
	take(mounts, 3, true)
	take(pets, 2, false)
	return out
}

func (g *Game) tmStockMenu(ost *Actor) {
	bd := g.trHere()
	if bd == nil {
		return
	}
	g.Talk = ost
	g.Popup = Wrap("\"Beasts from the wilder yards, bred and gentled. They are not cheap, and they are not tame to strangers who do not feed them.\"", 100)
	g.Opts = nil
	g.Sel = 0
	disc := func(n int) int { return maxi(1, n*(100-g.Skills["persuasion"]*8)/100) }
	for _, o := range g.tmStock(bd) {
		o := o
		price := disc(o.Price)
		kind := "companion"
		if o.Mount {
			kind = "mount, " + tmRideClass(o.S)
		}
		g.Opts = append(g.Opts, Option{fmt.Sprintf("Buy %s. {{K|(%s)}} — %s", bsArticle(o.S.Display()), kind, g.cuFormat(price)), func() { g.tmBuy(ost, o, price) }})
	}
	g.Opts = append(g.Opts, Option{"Never mind.", func() { g.talkMenu(g.greeting(ost)) }})
}

func (g *Game) tmBuy(ost *Actor, o tmOffer, price int) {
	back := func() { g.talkMenu(g.greeting(ost)) }
	st := g.tr()
	if o.Mount {
		if !g.cuSpend(price) {
			g.Msg("{{y|You cannot afford it.}}")
			back()
			return
		}
		if st.Mount != nil && st.Stable == nil {
			if od := trMountByID(st.Mount.Def); od != nil {
				g.cuEarn(od.Price / 3)
			}
		}
		g.tmSetMount(o.S, "", bsLvlT[o.S.Size], 55, 0)
		g.Msg(fmt.Sprintf("You buy %s. It eyes you warily.", bsArticle(o.S.Display())))
		back()
		return
	}
	if !g.tmSlotFree() {
		g.Msg("{{y|You already lead as many companions as you can keep.}}")
		back()
		return
	}
	if !g.cuSpend(price) {
		g.Msg("{{y|You cannot afford it.}}")
		back()
		return
	}
	rg := NewRng(fmt.Sprintf("%s:tm:buy:%d:%s", g.Seed, g.Turn, o.S.ID))
	a := g.bsSpawn(o.S, bsLvlT[o.S.Size], rg)
	a.X, a.Y = g.P.X, g.P.Y
	g.Cur.Actors = append(g.Cur.Actors, a)
	g.nudge(g.Cur, a)
	g.tmJoin(a, o.S, 50+5*g.Skills["persuasion"], "")
	g.Msg(fmt.Sprintf("{{G|%s the %s follows you from the stable.}}", a.Name, o.S.Display()))
	back()
}

func tmTurn(g *Game) {
	if g.P == nil || g.Dead || g.Cur == nil {
		return
	}
	st := g.tm()
	if st.Ride != nil {
		m := g.tr().Mount
		if m == nil || m.Def != "tm_"+st.Ride.Species {
			if st.Ride.Starve >= 3 {
				g.deed("companion", fmt.Sprintf("%s, mount of the wanderer %s, starved on the road.", g.tmMountLabel(st.Ride), g.P.Name), 0, 0)
			}
			st.Ride = nil
		} else {
			st.Ride.Starve = m.Starve
		}
	}
	if g.Turn%60 != 0 {
		return
	}
	if r := g.tmRideRec(); r != nil {
		if g.tr().Mount.Starve > 0 {
			r.Loyalty = maxi(0, r.Loyalty-1)
		}
		if r.Loyalty <= 0 {
			g.Msg(fmt.Sprintf("{{R|Your %s tears free and bolts for the wild.}}", g.tr().Mount.Name))
			g.tr().Mount = nil
			st.Ride = nil
		}
	}
	graze := g.Cur.Kind != "town" && g.Cur.Z == 0
	for _, a := range g.localCompanions() {
		if !a.Has("tm") {
			continue
		}
		s := bsByTpl[a.Tpl]
		if s != nil && graze && (s.Diet == 'h' || s.Diet == 'o') {
			st.Fed[a.ID] = g.Turn
		}
		if g.Turn-st.Fed[a.ID] > 1440 && g.Turn%720 == 0 {
			g.addLoyalty(a, -2)
			g.Msg(fmt.Sprintf("{{y|%s is hungry and restless. Feed it.}}", a.Name))
		}
	}
	if g.Turn%1440 == 0 {
		for k, p := range st.Prog {
			if g.Turn-p.Seen > 1440 {
				p.Trust--
			}
			if p.Trust <= 0 && g.Turn-p.Seen > 1440 || g.Turn-p.Seen > 14400 {
				delete(st.Prog, k)
			}
		}
	}
}

func tmKill(g *Game, d, src *Actor) {
	if d == nil {
		return
	}
	st := g.tm()
	delete(st.Prog, tmKey(d))
	delete(st.Fed, d.ID)
	if src != nil && src.AI == "follower" && src.Has("tm") && strings.HasPrefix(d.Tpl, "bs_") && d.AI != "follower" {
		st.Fed[src.ID] = g.Turn
		if g.visible(src.X, src.Y) {
			g.Msg(fmt.Sprintf("{{K|%s feeds on its kill.}}", src.Name))
		}
	}
}

func tmSide(g *Game) []string {
	r := g.tmRideRec()
	if r == nil {
		return nil
	}
	s := bsByID[r.Species]
	if s == nil || tmRideClass(s) != "fly" {
		return nil
	}
	g.tmRegen(r, s, g.Turn)
	return []string{fmt.Sprintf("{{c|Wings: %d%%}}", int(100*r.Stam/tmMaxStam(s)))}
}
