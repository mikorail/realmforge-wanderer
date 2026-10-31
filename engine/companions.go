package engine

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// Companions: recruited followers with orders, gear, growth, loyalty and memory of home.
// Followers created by powers (charms, echoes) have no recruit record; they get a
// "summoned" record lazily, take orders, but ignore loyalty.

// Companion is the per-follower record (Actor.Comp). Actor.Order holds the standing order.
type Companion struct {
	Faction  string   `json:"fa,omitempty"`  // original faction (state:N, …) before joining
	AI       string   `json:"ai,omitempty"`  // original AI (town, guard)
	F        byte     `json:"f,omitempty"`   // original colour
	HomeKey  string   `json:"hk,omitempty"`  // zone they were recruited in
	HomeX    int      `json:"hx,omitempty"`  // their old home position there
	HomeY    int      `json:"hy,omitempty"`  //
	Town     string   `json:"tn,omitempty"`  // home settlement name
	Kin      []string `json:"kin,omitempty"` // "father Aldo", remembered from the household
	Loyalty  int      `json:"lo"`
	XP       int      `json:"xp,omitempty"`
	Kills    int      `json:"k,omitempty"`
	Target   int      `json:"tg,omitempty"` // actor id for the attack order
	GX       int      `json:"gx,omitempty"` // guard post
	GY       int      `json:"gy,omitempty"`
	Since    int      `json:"since,omitempty"` // turn they joined
	Summoned bool     `json:"sum,omitempty"`
	MealAt   int      `json:"ma,omitempty"` // turn+1 of the last shared meal (loyalty bonus cooldown)
}

// Orders (Actor.Order); "" means follow.
const (
	OrdFollow   = ""
	OrdStay     = "stay"
	OrdGuard    = "guard"
	OrdAttack   = "attack"
	OrdDistance = "distance"
	OrdPassive  = "passive"
)

var orderNames = map[string]string{OrdFollow: "following", OrdStay: "waiting", OrdGuard: "guarding", OrdAttack: "attacking", OrdDistance: "keeping distance", OrdPassive: "staying out of fights"}

const maxCompanions = 3

func (g *Game) comp(a *Actor) *Companion {
	if a.Comp == nil {
		a.Comp = &Companion{Loyalty: 100, Summoned: true, Since: g.Turn}
	}
	return a.Comp
}

func isFollower(a *Actor) bool { return a != nil && a.AI == "follower" && a.HP > 0 }

// localCompanions: followers in the current zone.
func (g *Game) localCompanions() []*Actor {
	var out []*Actor
	if g.Cur == nil {
		return out
	}
	for _, a := range g.Cur.Actors {
		if isFollower(a) {
			out = append(out, a)
		}
	}
	return out
}

type waitingComp struct {
	A *Actor
	Z *Zone
}

// waitingCompanions: followers left behind in other zones (ordered to stay or guard).
func (g *Game) waitingCompanions() []waitingComp {
	var out []waitingComp
	keys := make([]string, 0, len(g.Zones))
	for k := range g.Zones {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		z := g.Zones[k]
		if z == g.Cur {
			continue
		}
		for _, a := range z.Actors {
			if isFollower(a) {
				out = append(out, waitingComp{a, z})
			}
		}
	}
	return out
}

func (g *Game) companionCount() int { return len(g.localCompanions()) + len(g.waitingCompanions()) }

// joinCompanion turns a recruited townsperson into a companion.
func (g *Game) joinCompanion(a *Actor) {
	c := &Companion{Faction: a.Faction, AI: a.AI, F: a.F, HomeKey: g.Cur.Key, HomeX: a.HomeX, HomeY: a.HomeY, Town: g.Cur.Name, Since: g.Turn}
	if bd := g.W.burgByID[g.Cur.Burg]; bd != nil && g.Cur.Burg > 0 {
		c.Town = bd.Name
	}
	for _, id := range a.Kin {
		if k := g.findActor(id); k != nil {
			c.Kin = append(c.Kin, kinRelation(a, k)+" "+givenOf(k.Name))
		}
	}
	c.Loyalty = 55 + g.Skills["persuasion"]*5
	if a.Kinship {
		c.Loyalty += 15
	}
	a.Comp = c
	a.Order = OrdFollow
	a.AI = "follower"
	a.Faction = "player"
	a.F = CGreen
	a.Angry = false
	g.Msg(fmt.Sprintf("{{G|%s joins you.}} {{K|(o: companion orders)}}", a.Name))
}

func loyaltyWord(l int) string {
	switch {
	case l >= 80:
		return "{{G|devoted}}"
	case l >= 55:
		return "{{g|loyal}}"
	case l >= 35:
		return "{{y|steady}}"
	case l >= 20:
		return "{{W|wavering}}"
	}
	return "{{R|disgruntled}}"
}

func compXPFor(lv int) int { return lv*lv*20 + 30 }

func (g *Game) addLoyalty(a *Actor, n int) {
	c := g.comp(a)
	if c.Summoned {
		return
	}
	c.Loyalty = clampi(c.Loyalty+n, 0, 100)
}

// ---- growth ----

func (g *Game) compGainXP(a *Actor, n int) {
	if n <= 0 || !isFollower(a) {
		return
	}
	c := g.comp(a)
	c.XP += n
	for c.XP >= compXPFor(a.Level) {
		c.XP -= compXPFor(a.Level)
		a.Level++
		hp := maxi(2, 3+a.M(TOU))
		a.MaxHP += hp
		a.HP = mini(a.MaxHP, a.HP+hp)
		at := []int{STR, AGI, TOU}[a.Level%3]
		a.Attr[at]++
		extra := ""
		if a.Level%3 == 0 {
			a.DVNat++
			extra = ", +1 DV"
		}
		if a.Level%4 == 0 {
			a.AVNat++
			extra += ", +1 AV"
		}
		g.Msg(fmt.Sprintf("{{G|%s grows more seasoned: level %d!}} {{K|(+%d HP, +1 %s%s)}}", a.Name, a.Level, hp, AttrShort[at], extra))
	}
}

// compOnKill: called at the start of kill(). Companion deaths reach the chronicle; kills share XP.
func (g *Game) compOnKill(d, src *Actor) {
	if d.AI == "follower" {
		c := g.comp(d)
		if c.Summoned {
			return
		}
		by := "the perils of the road"
		if src != nil && src != d {
			by = g.the(src)
		}
		g.Msg(fmt.Sprintf("{{R|Your companion %s has fallen.}}", d.Name))
		home := ""
		if c.Town != "" {
			home = " of " + c.Town
		}
		g.deed("companion", fmt.Sprintf("%s%s, companion of the wanderer %s, fell in %s, slain by %s.", d.Name, home, g.P.Name, g.Cur.Name, by), 0, 0)
		return
	}
	if src == nil || src != g.P && src.AI != "follower" {
		return
	}
	for _, a := range g.localCompanions() {
		if a == d {
			continue
		}
		if a == src {
			g.comp(a).Kills++
			g.compGainXP(a, d.XP)
		} else if a.Order != OrdPassive && dist(a, d) <= 8 {
			g.compGainXP(a, d.XP/2)
		}
	}
}

// ---- loyalty ----

func (g *Game) compFaith(a *Actor) string {
	if a.Religion > 0 {
		return fmt.Sprintf("faith:%d", a.Religion)
	}
	return ""
}

// compRep: hook from AddRep. Reputation gained or lost with a companion's realm or faith moves their loyalty.
func (g *Game) compRep(f string, n int) {
	if g.Cur == nil || n == 0 {
		return
	}
	for _, a := range g.localCompanions() {
		c := g.comp(a)
		if c.Summoned || f != c.Faction && f != g.compFaith(a) {
			continue
		}
		if n < 0 {
			g.addLoyalty(a, n/4-1)
			if n <= -50 {
				g.Msg(fmt.Sprintf("{{r|%s glares at you: those are %s people.}}", a.Name, possessiveWord(a)))
			}
		} else {
			g.addLoyalty(a, maxi(1, n/20))
		}
	}
}

func possessiveWord(a *Actor) string {
	if isFemale(a) {
		return "her"
	}
	return "his"
}

// companionsTick runs once a game minute (from turn()): hourly loyalty drift and departures.
func (g *Game) companionsTick() {
	for _, a := range g.localCompanions() {
		c := g.comp(a)
		if c.Summoned {
			continue
		}
		if g.Turn%60 == 0 {
			hour := g.Turn / 60
			if g.Hunger > 2400 {
				g.addLoyalty(a, -1)
			}
			worst := 0
			if c.Faction != "" {
				worst = g.Rep[c.Faction]
			}
			if fk := g.compFaith(a); fk != "" {
				if fr, ok := g.Rep[fk]; ok && fr < worst {
					worst = fr
				}
			}
			switch {
			case worst <= -250:
				g.addLoyalty(a, -2)
			case worst < 0 && hour%2 == 0:
				g.addLoyalty(a, -1)
			case worst >= 250 && c.Loyalty < 75 && hour%3 == 0:
				g.addLoyalty(a, 1)
			}
			if c.Loyalty == 20 {
				g.Msg(fmt.Sprintf("{{W|%s mutters and lags behind. Their loyalty is wavering.}}", a.Name))
			}
		}
		if c.Loyalty < 10 {
			g.Msg(fmt.Sprintf("{{R|%s has had enough of you and leaves your company.}}", a.Name))
			g.releaseCompanion(a)
		}
	}
}

// releaseCompanion ends the bond: they return to their old faction and head home.
func (g *Game) releaseCompanion(a *Actor) {
	c := g.comp(a)
	a.Order = ""
	if c.Summoned {
		g.removeActor(g.Cur, a)
		return
	}
	a.AI = nonEmpty(c.AI, "town")
	a.Faction = c.Faction
	if c.F != 0 {
		a.F = c.F
	}
	a.HomeX, a.HomeY = c.HomeX, c.HomeY
	a.Comp = nil
	if c.HomeKey == g.Cur.Key {
		return
	}
	g.removeActor(g.Cur, a)
	if hz := g.Zones[c.HomeKey]; hz != nil {
		a.X, a.Y = c.HomeX, c.HomeY
		hz.Actors = append(hz.Actors, a)
		g.nudge(hz, a)
	}
}

func (g *Game) removeActor(z *Zone, a *Actor) {
	for i, o := range z.Actors {
		if o == a {
			z.Actors = append(z.Actors[:i], z.Actors[i+1:]...)
			return
		}
	}
}

// compComes: whether a follower comes along on a zone change (hook in enterZone).
func (g *Game) compComes(a *Actor) bool {
	if a.Order == OrdStay || a.Order == OrdGuard {
		return false
	}
	if a.Order == OrdAttack {
		a.Order = OrdFollow
	}
	return a.Comp != nil && !a.Comp.Summoned || abs(a.X-g.P.X) <= 8 && abs(a.Y-g.P.Y) <= 8
}

// ---- orders ----

// ownPeople: companions will not raise a hand against their own realm.
func (g *Game) ownPeople(a, o *Actor) bool {
	c := a.Comp
	if c == nil || c.Summoned || !strings.HasPrefix(c.Faction, "state:") {
		return false
	}
	if o.Faction == c.Faction {
		return true
	}
	return o.Comp != nil && o.Comp.Faction == c.Faction
}

func (g *Game) setOrder(a *Actor, ord string) {
	c := g.comp(a)
	a.Order = ord
	switch ord {
	case OrdStay:
		g.Msg(fmt.Sprintf("%s will wait here.", a.Name))
	case OrdGuard:
		c.GX, c.GY = a.X, a.Y
		g.Msg(fmt.Sprintf("%s takes up a guard post.", a.Name))
	case OrdDistance:
		g.Msg(fmt.Sprintf("%s will keep their distance.", a.Name))
	case OrdPassive:
		g.Msg(fmt.Sprintf("%s will stay out of fights.", a.Name))
	default:
		g.Msg(fmt.Sprintf("%s falls in behind you.", a.Name))
	}
}

// orderAttack: the targeting hook (Pend "order:<id>", 0 = all companions).
func (g *Game) orderAttack(pend string, tx, ty int) {
	id, _ := strconv.Atoi(strings.TrimPrefix(pend, "order:"))
	t := g.Cur.ActorAt(tx, ty)
	if t == nil || t.HP <= 0 || t == g.P {
		g.Msg("There is no one there to attack.")
		return
	}
	if isFollower(t) {
		g.Msg("Your companions will not turn on each other.")
		return
	}
	n := 0
	for _, a := range g.localCompanions() {
		if id != 0 && a.ID != id {
			continue
		}
		if g.ownPeople(a, t) {
			g.Msg(fmt.Sprintf("{{W|%s refuses: \"I'll not raise a hand against my own people.\"}}", a.Name))
			g.addLoyalty(a, -3)
			continue
		}
		g.comp(a).Target = t.ID
		a.Order = OrdAttack
		n++
	}
	if n > 0 {
		g.Msg(fmt.Sprintf("{{G|You point at %s.}} %d %s to the attack.", g.the(t), n, map[bool]string{true: "companion moves", false: "companions move"}[n == 1]))
	}
}

func (g *Game) compEnemy(a, o *Actor) bool {
	return o != a && o.HP > 0 && o != g.P && g.Enemies(a, o) && !g.ownPeople(a, o)
}

func (g *Game) nearestFoe(a *Actor, cx, cy, r int) *Actor {
	var best *Actor
	bd := 1 << 20
	for _, o := range g.Cur.Actors {
		if !g.compEnemy(a, o) {
			continue
		}
		if maxi(abs(o.X-cx), abs(o.Y-cy)) > r {
			continue
		}
		if d := dist(a, o); d < bd {
			best, bd = o, d
		}
	}
	return best
}

func (g *Game) compFight(a, t *Actor) {
	d := dist(a, t)
	if d <= 1 {
		g.melee(a, t)
		return
	}
	if m := a.Missile(); m != nil && d <= m.Range && g.losClear(a.X, a.Y, t.X, t.Y) {
		g.shoot(a, t.X, t.Y, m)
		return
	}
	if !g.stepToward(a, g.distMap(t.X, t.Y, a.Has("flying")), false) {
		g.wander(a)
	}
}

func (g *Game) compFollow(a *Actor, near int) {
	if dist(a, g.P) > near {
		g.stepToward(a, g.distMap(g.P.X, g.P.Y, a.Has("flying")), false)
	} else if g.R.Chance(0.2) {
		g.wander(a)
	}
}

// companionAct: follower AI under orders (hook at the top of aiAct). False = use the default AI.
func (g *Game) companionAct(a *Actor) bool {
	if a.AI != "follower" || a.Comp == nil && a.Order == "" {
		return false
	}
	c := g.comp(a)
	switch a.Order {
	case OrdStay:
		if t := g.nearestFoe(a, a.X, a.Y, 1); t != nil {
			g.melee(a, t)
		} else if m := a.Missile(); m != nil {
			if t := g.nearestFoe(a, a.X, a.Y, m.Range); t != nil && g.losClear(a.X, a.Y, t.X, t.Y) {
				g.shoot(a, t.X, t.Y, m)
			}
		}
	case OrdGuard:
		if t := g.nearestFoe(a, c.GX, c.GY, 6); t != nil {
			g.compFight(a, t)
		} else if a.X != c.GX || a.Y != c.GY {
			g.stepToward(a, g.distMap(c.GX, c.GY, a.Has("flying")), false)
		}
	case OrdAttack:
		t := g.findActor(c.Target)
		if t == nil || t.HP <= 0 || g.ownPeople(a, t) {
			a.Order, c.Target = OrdFollow, 0
			if g.visible(a.X, a.Y) {
				g.Msg(fmt.Sprintf("%s looks to you for orders.", a.Name))
			}
			g.compFollow(a, 2)
			return true
		}
		g.compFight(a, t)
	case OrdPassive:
		if t := g.nearestFoe(a, a.X, a.Y, 5); t != nil {
			if !g.stepToward(a, g.distMap(t.X, t.Y, a.Has("flying")), true) && dist(a, t) <= 1 {
				g.melee(a, t)
			}
			return true
		}
		g.compFollow(a, 2)
	case OrdDistance:
		t := g.nearestFoe(a, g.P.X, g.P.Y, 14)
		if t == nil {
			g.compFollow(a, 3)
			return true
		}
		d := dist(a, t)
		m := a.Missile()
		switch {
		case d <= 2 && g.stepToward(a, g.distMap(t.X, t.Y, a.Has("flying")), true):
		case d <= 1:
			g.melee(a, t)
		case m != nil && d <= m.Range && g.losClear(a.X, a.Y, t.X, t.Y):
			g.shoot(a, t.X, t.Y, m)
		default:
			g.compFollow(a, 3)
		}
	default:
		if t := g.nearestFoe(a, g.P.X, g.P.Y, 12); t != nil && dist(a, t) < 14 {
			g.compFight(a, t)
			return true
		}
		g.compFollow(a, 2)
	}
	return true
}

// ---- the companions screen (o) ----

func (g *Game) openCompanions() {
	g.Mode, g.Sel = "comp", 0
	g.Opts = nil
	g.Talk = nil
	add := func(t string, f func()) { g.Opts = append(g.Opts, Option{t, f}) }
	loc := g.localCompanions()
	for _, a := range loc {
		a := a
		add(fmt.Sprintf("Speak with {{G|%s}}", a.Name), func() { g.openTalk(a) })
	}
	if len(loc) > 0 {
		all := func(ord string) func() {
			return func() {
				for _, a := range g.localCompanions() {
					g.setOrder(a, ord)
				}
				g.Mode = "main"
			}
		}
		add("All: follow me closely.", all(OrdFollow))
		add("All: wait here.", all(OrdStay))
		add("All: hold this position and guard it.", all(OrdGuard))
		add("{{R|All: attack my target.}}", func() { g.beginTarget("order:0") })
		add("All: keep your distance.", all(OrdDistance))
		add("All: stay out of fights.", all(OrdPassive))
	}
	add("Close.", func() { g.Mode = "main" })
}

func (g *Game) drawCompanions() {
	s := g.sc
	x, y, w, _ := g.panel("Companions")
	loc := g.localCompanions()
	wait := g.waitingCompanions()
	if len(loc)+len(wait) == 0 {
		s.TextB(x, y, "{{K|You travel alone. Villagers and guards who trust you may join you (\"Travel with me.\").}}", CGray, CBgDark, x+w)
		y += 2
	}
	row := func(a *Actor, where string) {
		c := g.comp(a)
		s.Put(x, y, a.Glyph, a.F, CBgDark)
		hb, _ := Bar(a.HP, a.MaxHP, 10)
		bar := "{{" + hpColor(a.HP, a.MaxHP) + "|" + strings.Repeat("█", hb) + "}}{{K|" + strings.Repeat("░", 10-hb) + "}}"
		lo := "{{K|bound}}"
		if !c.Summoned {
			lo = loyaltyWord(c.Loyalty)
		}
		s.TextB(x+2, y, fmt.Sprintf("{{G|%-20s}} lv %-2d %s %3d/%-3d  %s  {{c|%s}}", a.Name, a.Level, bar, a.HP, a.MaxHP, lo, orderNames[a.Order]), CGray, CBgDark, x+w)
		y++
		var info []string
		if !c.Summoned {
			info = append(info, fmt.Sprintf("XP %d/%d", c.XP, compXPFor(a.Level)), fmt.Sprintf("%d kills", c.Kills))
			if c.Town != "" {
				info = append(info, "from "+c.Town)
			}
		}
		if where != "" {
			info = append(info, "{{W|"+where+"}}")
		}
		s.TextB(x+4, y, "{{K|"+strings.Join(info, " · ")+"}}", CGray, CBgDark, x+w)
		y++
	}
	for _, a := range loc {
		row(a, "")
	}
	for _, wc := range wait {
		row(wc.A, "waiting at "+wc.Z.Name+" — return there to take them along")
	}
	y++
	for i, o := range g.Opts {
		bg := CBgDark
		if i == g.Sel {
			bg = CBgSel
		}
		s.Fill(x, y, w, 1, bg)
		s.TextB(x, y, fmt.Sprintf("{{W|%c)}} %s", 'a'+i, o.Text), CGray, bg, x+w)
		y++
	}
}

// drawCompanionsSide lists companions in the side panel; returns the next free row.
func (g *Game) drawCompanionsSide(x0, y int) int {
	loc := g.localCompanions()
	if len(loc) == 0 {
		return y
	}
	s := g.sc
	s.TextB(x0, y, "{{K|Companions (o):}}", CGray, CBgDark, SW)
	y++
	for _, a := range loc {
		nm := []rune(a.Name)
		if len(nm) > 18 {
			nm = nm[:18]
		}
		s.Put(x0, y, a.Glyph, a.F, CBgDark)
		tag := ""
		if a.Order == OrdStay || a.Order == OrdGuard {
			tag = "{{K|·}}"
		}
		s.TextB(x0+2, y, "{{G|"+string(nm)+"}}"+tag, CGray, CBgDark, SW-7)
		hb, _ := Bar(a.HP, a.MaxHP, 6)
		for i := 0; i < 6; i++ {
			col := CDarkGray
			if i < hb {
				col = colorCode[hpColor(a.HP, a.MaxHP)[0]]
			}
			s.Put(SW-7+i, y, '▪', col, CBgDark)
		}
		y++
	}
	return y + 1
}

// ---- talking with a companion ----

// companionTalk replaces the dialogue menu for a follower (hook in openTalk).
func (g *Game) companionTalk(a *Actor, text string) {
	g.Talk = a
	g.Mode = "talk"
	g.Popup = Wrap(text, 100)
	g.Opts = nil
	g.Sel = 0
	add := func(t string, f func()) { g.Opts = append(g.Opts, Option{t, f}) }
	add("{{C|I have orders for you.}}", func() { g.compOrderMenu(a) })
	add("{{W|Take this.}} {{K|(give an item)}}", func() { g.compGiveMenu(a) })
	add("Hand me something you carry.", func() { g.compTakeMenu(a) })
	if inst := g.augInstallerHere(); inst != nil {
		add(fmt.Sprintf("{{O|Let the %s fit you with a %s.}}", inst.Title, g.augSkin().Noun), func() {
			g.Talk = inst
			g.augMenuFor(inst, a)
		})
	}
	add("Tell me about yourself.", func() { g.companionTalk(a, g.compAbout(a)) })
	add("How are you holding up?", func() { g.companionTalk(a, g.compMood(a)) })
	add("{{r|Let's part ways.}}", func() {
		g.Msg(fmt.Sprintf("You part ways with %s.", a.Name))
		g.releaseCompanion(a)
		g.Mode, g.Talk = "main", nil
	})
	add("Goodbye.", func() { g.Mode, g.Talk = "main", nil })
}

func (g *Game) compGreeting(a *Actor) string {
	c := g.comp(a)
	switch {
	case c.Summoned:
		return "It regards you, waiting to be told."
	case c.Loyalty >= 80:
		return "\"Wherever you lead, " + givenOf(g.P.Name) + ".\""
	case c.Loyalty >= 35:
		return "\"Yes? What do we do?\""
	}
	return "\"What now?\" They don't hide their impatience."
}

func (g *Game) compOrderMenu(a *Actor) {
	g.Popup = Wrap(fmt.Sprintf("%s is %s. What should they do?", a.Name, orderNames[a.Order]), 100)
	g.Opts = nil
	g.Sel = 0
	set := func(ord string) func() {
		return func() { g.setOrder(a, ord); g.Mode, g.Talk = "main", nil }
	}
	add := func(t string, f func()) { g.Opts = append(g.Opts, Option{t, f}) }
	add("Follow me closely.", set(OrdFollow))
	add("Stay here and wait for me.", set(OrdStay))
	add("Hold this position and guard it.", set(OrdGuard))
	add("{{R|Attack my target.}}", func() { g.Talk = nil; g.beginTarget(fmt.Sprintf("order:%d", a.ID)) })
	add("Keep your distance; fight from range.", set(OrdDistance))
	add("Stay out of fights.", set(OrdPassive))
	add("Never mind.", func() { g.companionTalk(a, g.compGreeting(a)) })
}

func itemScore(it *Item) float64 {
	if it == nil {
		return -1
	}
	return dmgAvg(it)*2 + float64(it.PV*2+it.AV*3+it.DV*2+it.Light+it.Range/2)
}

func giftable(it *Item) bool {
	switch it.Kind {
	case "weapon", "armor", "ranged", "light", "food", "tonic", "dish", "ingredient":
		return true
	}
	return false
}

func (g *Game) compGiveMenu(a *Actor) {
	g.Popup = Wrap("What will you hand "+a.Name+"?", 100)
	g.Opts = nil
	g.Sel = 0
	for _, it := range g.P.Inv {
		if !giftable(it) || len(g.Opts) >= 18 {
			continue
		}
		it := it
		g.Opts = append(g.Opts, Option{it.Display(), func() { g.companionTalk(a, g.compGive(a, it)) }})
	}
	g.Opts = append(g.Opts, Option{"Never mind.", func() { g.companionTalk(a, g.compGreeting(a)) }})
}

func (g *Game) takeOne(it *Item) *Item {
	if it.Stack() && it.Count > 1 {
		it.Count--
		cp := *it
		cp.ID, cp.Count = g.newID(), 1
		return &cp
	}
	g.removeInv(it)
	return it
}

// compGive: a companion takes a gift; gear is worn if it beats what they have, food and tonics are used.
func (g *Game) compGive(a *Actor, it *Item) string {
	switch it.Kind {
	case "dish":
		name := it.Name
		g.takeOne(it)
		g.feedCompanion(a, tagsOf(it.Effect), 1200)
		g.addLoyalty(a, 10)
		msg := fmt.Sprintf("They eat the %s while it is still warm. \"Now that is a meal.\"", name)
		if d := describeTags(a.Meal); d != "" {
			msg += " {{K|(" + a.Name + " feels:}} " + d + "{{K|)}}"
		}
		return msg
	case "ingredient":
		g.takeOne(it)
		a.HP = mini(a.MaxHP, a.HP+2)
		g.addLoyalty(a, 2)
		return "They chew it raw and shrug. \"Better than nothing. Cooked would be better.\""
	case "food":
		g.takeOne(it)
		a.HP = mini(a.MaxHP, a.HP+5)
		g.addLoyalty(a, 8)
		return "They eat gratefully. \"I won't forget a full belly.\""
	case "tonic":
		g.takeOne(it)
		switch it.Effect {
		case "heal":
			a.HP = mini(a.MaxHP, a.HP+12+a.Level*3)
		case "salve":
			delete(a.Status, "bleeding")
			delete(a.Status, "burning")
			a.HP = mini(a.MaxHP, a.HP+6)
		case "haste":
			a.AddSt("hasted", 40)
		case "strength":
			a.AddSt("mighty", 40)
		case "antidote":
			delete(a.Status, "poisoned")
		}
		g.addLoyalty(a, 5)
		return "They drink it down. \"Thank you.\""
	}
	slot := it.Slot
	if it.Kind == "light" {
		slot = "float"
	}
	part := a.PartFor(slot)
	if part == nil {
		return "\"I've no way to wear that.\""
	}
	if part.Item != nil && itemScore(it) <= itemScore(part.Item) {
		return "\"Keep it; " + strings.ToLower(part.Item.Display()) + " serves me better.\""
	}
	g.removeInv(it)
	msg := "They take " + it.Display() + " and put it on."
	if old := part.Item; old != nil {
		g.give(old)
		msg += " They hand you back " + old.Display() + "."
	}
	part.Item = it
	g.addLoyalty(a, 3)
	return msg
}

func (g *Game) compTakeMenu(a *Actor) {
	g.Popup = Wrap("\"What do you need?\"", 100)
	g.Opts = nil
	g.Sel = 0
	for _, bp := range a.Body {
		if bp.Item == nil {
			continue
		}
		bp := bp
		it := bp.Item
		g.Opts = append(g.Opts, Option{it.Display() + " {{K|(" + strings.ToLower(bp.Name) + ")}}", func() {
			bp.Item = nil
			g.give(it)
			g.addLoyalty(a, -1)
			g.companionTalk(a, "They hand over "+it.Display()+".")
		}})
	}
	for _, it := range a.Inv {
		it := it
		g.Opts = append(g.Opts, Option{it.Display(), func() {
			for i, o := range a.Inv {
				if o == it {
					a.Inv = append(a.Inv[:i], a.Inv[i+1:]...)
					break
				}
			}
			g.give(it)
			g.companionTalk(a, "They hand over "+it.Display()+".")
		}})
	}
	if len(g.Opts) > 18 {
		g.Opts = g.Opts[:18]
	}
	g.Opts = append(g.Opts, Option{"Never mind.", func() { g.companionTalk(a, g.compGreeting(a)) }})
}

func (g *Game) compAbout(a *Actor) string {
	c := g.comp(a)
	if c.Summoned {
		return "It has no past to speak of; it is here because you willed it."
	}
	s := fmt.Sprintf("\"I am %s", a.Name)
	if c.Town != "" {
		s += ", of " + c.Town
	}
	s += ".\""
	if a.Caste != "" {
		s += " " + strings.TrimSpace(fmt.Sprintf("Among my people I was of the %s.", a.Caste))
	}
	if a.House != "" {
		s += " House " + a.House + " is my family."
	}
	if len(c.Kin) > 0 {
		s += " I left my " + strings.Join(c.Kin, " and my ") + " behind."
	}
	if strings.HasPrefix(c.Faction, "state:") {
		s += fmt.Sprintf(" I am a subject of %s, and I'll not fight my own.", g.FactionName(c.Faction))
	}
	if fk := g.compFaith(a); fk != "" {
		s += fmt.Sprintf(" I keep the %s.", g.FactionName(fk))
	}
	return s
}

func (g *Game) compMood(a *Actor) string {
	c := g.comp(a)
	if c.Summoned {
		return "It does not tire, nor does it doubt."
	}
	days := (g.Turn - c.Since) / 1440
	var s string
	switch {
	case c.Loyalty >= 80:
		s = "\"I'd follow you into the dark itself.\""
	case c.Loyalty >= 55:
		s = "\"Well enough. It's good to be on the road with you.\""
	case c.Loyalty >= 35:
		s = "\"I'm holding up.\""
	case c.Loyalty >= 20:
		s = "\"I've had doubts, if I'm honest.\""
	default:
		s = "\"Not well. I'm thinking of going home.\""
	}
	if g.Hunger > 2400 {
		s += " \"And we're starving.\""
	}
	if a.Meal != nil && a.Has("fed") {
		if d := describeTags(a.Meal); d != "" {
			s += "\n\n{{K|Well fed:}} " + d
		}
	}
	var augs []string
	for _, p := range a.Body {
		if p.Aug != nil {
			augs = append(augs, p.Aug.Name)
		}
	}
	if len(augs) > 0 {
		s += fmt.Sprintf("\n\n{{K|Augments (%d/%d):}} %s", a.AugLoad(), a.AugCapacity(), strings.Join(augs, ", "))
	}
	return fmt.Sprintf("%s\n\n{{K|Level %d, %d/%d XP, %d kills, %d days together. Loyalty:}} %s {{K|(%d)}}", s, a.Level, c.XP, compXPFor(a.Level), c.Kills, days, loyaltyWord(c.Loyalty), c.Loyalty)
}
