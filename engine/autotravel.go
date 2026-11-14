package engine

import (
	"fmt"
	"math"
	"sort"
	"strings"
)

// Auto-travel (prefix at). World map: pick a destination (towns, your waypoints, the quest marker, deadline
// sites, the map cursor), preview the route (km, days, food and water against the pack), then travel by
// repeated WorldMove calls so every existing hook, encounter and event still fires. The route comes from
// places.go wplRoute (Dijkstra over tvMinutes, so roads and wings/mounts are already preferred; read-only
// use). Stops on interrupts, resumes with "continue the trip", keeps a trip log and a waypoint-to-waypoint
// itinerary. In a zone it walks to stairs, the gate, doors, wells and stalls (qol.go auto-explore covers
// exploring; this covers going to a named thing). State g.ext("at"). Never draws g.R.

type atDest struct {
	Name   string `json:"n"`
	X      int    `json:"x"`
	Y      int    `json:"y"`
	Planet int    `json:"p,omitempty"`
}

type atTrip struct {
	To     string `json:"to"`
	Turn   int    `json:"tn"`
	Days   int    `json:"d"`
	Steps  int    `json:"s"`
	Result string `json:"r"`
}

type atState struct {
	Dest     *atDest  `json:"dest,omitempty"`
	Queue    []atDest `json:"q,omitempty"`
	Log      []atTrip `json:"log,omitempty"`
	NightDay int      `json:"nd,omitempty"`

	sel, tab, stage int
	plan            *atPlan
	planKey         string
	entries         []atEntry
	zkey            string
	zinfo           []atZoneInfo
}

func (g *Game) at() *atState {
	return g.ext("at", func() any { return &atState{NightDay: -1} }).(*atState)
}

func init() {
	modeKeys["atv"] = atKeys
	modeDraws["atv"] = atDraw
	mainKeys["q"] = func(g *Game) { g.atOpen() }
}

// atOpen opens the auto-travel screen (hook: keyWorld `case "q": g.atOpen()`).
func (g *Game) atOpen() {
	if g.P == nil || g.W == nil {
		return
	}
	st := g.at()
	st.sel, st.stage, st.plan, st.planKey = 0, 0, nil, ""
	st.tab = 0
	g.Mode = "atv"
}

func (g *Game) atClose() {
	if g.OnWorld {
		g.Mode = "world"
	} else {
		g.Mode = "main"
	}
}

// ---- destinations

type atEntry struct {
	Group string
	Name  string
	X, Y  int
	Note  string
}

func (g *Game) atEntries() []atEntry {
	w := g.W
	st := g.at()
	pl := g.PlanetIndex()
	var out []atEntry
	seen := map[int]bool{}
	add := func(grp, name string, x, y int, note string) {
		if !w.In(x, y) {
			return
		}
		out = append(out, atEntry{grp, name, x, y, note})
	}
	if st.Dest != nil && st.Dest.Planet == pl {
		add("Resume", "Continue the trip to "+st.Dest.Name, st.Dest.X, st.Dest.Y, "")
	}
	if g.OnWorld && (g.CX != g.WX || g.CY != g.WY) {
		add("Map cursor", fmt.Sprintf("Cursor (%d,%d)", g.CX, g.CY), g.CX, g.CY, "")
	}
	for _, q := range g.Quests {
		if q.Done || q.Closed || (q.TP == 0 && q.Target == "") {
			continue
		}
		add("Quest marker", q.Title, q.TP%w.W, q.TP/w.W, q.Giver)
	}
	for _, e := range g.dlActive() {
		if !e.Known {
			continue
		}
		p := e.Site
		if p <= 0 {
			if b := w.burgByID[e.Burg]; b != nil {
				p = b.P
			}
		}
		lbl, left := g.dlCount(e)
		add("Deadlines", g.dlTitle(e), p%w.W, p/w.W, fmt.Sprintf("%s %s", lbl, tvDuration(maxi(0, left))))
	}
	for _, p := range g.wpl().List {
		if p.Planet == pl {
			add("Waypoints", p.Name, p.WX, p.WY, p.Note)
		}
	}
	type tw struct {
		b *BurgData
		d float64
	}
	var towns []tw
	for i := range w.Burgs {
		b := &w.Burgs[i]
		if !b.Alive || !w.In(b.P%w.W, b.P/w.W) || (!g.Known[b.P] && !b.Capital) {
			continue
		}
		d := math.Hypot(float64(b.P%w.W-g.WX), float64(b.P/w.W-g.WY))
		if d == 0 {
			continue
		}
		towns = append(towns, tw{b, d})
	}
	sort.Slice(towns, func(i, j int) bool { return towns[i].d < towns[j].d })
	for i, t := range towns {
		if i >= 10 {
			break
		}
		if seen[t.b.P] {
			continue
		}
		note := t.b.Kind
		if t.b.Capital {
			note = "capital"
		}
		add("Towns near", t.b.Name, t.b.P%w.W, t.b.P/w.W, note)
	}
	return out
}

// ---- plan (preview)

type atPlan struct {
	Legs     []atDest
	Steps    int
	Road     int
	Mins     int
	Km       float64
	OK       bool
	Why      string
	Meals    int
	Drams    int
	PackMeal int
	PackDram int
	Warn     []string
}

// atPack counts meals and drams you carry.
func (g *Game) atPack() (meals, drams int, avgPow float64) {
	tot, n := 0, 0
	for _, it := range g.P.Inv {
		if it.Kind == "food" {
			c := maxi(1, it.Count)
			meals += c
			tot += maxi(it.Power, 300) * c
			n += c
		}
	}
	for _, it := range g.lqContainers() {
		if lqKey(it) == "fresh" {
			drams += it.Charges
		}
	}
	avgPow = 450
	if n > 0 {
		avgPow = float64(tot) / float64(n)
	}
	return
}

func (g *Game) atNeed(mins int, avgPow float64) (meals, drams int) {
	gain := float64(mins) / float64(2+g.Skills["endurance"])
	if g.slowHunger() {
		gain = 0
	}
	if e := float64(g.Hunger) + gain - 900; e > 0 {
		meals = int(math.Ceil(e / avgPow))
	}
	if e := float64(g.Thirst) + gain - 600; e > 0 {
		drams = int(math.Ceil(e / 450))
	}
	return
}

// atMakePlan routes through the legs from where you stand. It restores your position.
func (g *Game) atMakePlan(legs []atDest) *atPlan {
	p := &atPlan{Legs: legs, OK: true}
	wx, wy := g.WX, g.WY
	defer func() { g.WX, g.WY = wx, wy }()
	for _, l := range legs {
		if l.Planet != g.PlanetIndex() {
			p.OK, p.Why = false, "That place lies on another world."
			return p
		}
		path, mins, ok := g.wplRoute(l.X, l.Y)
		if !ok {
			p.OK = false
			p.Why = "No land route to " + l.Name + ": it needs a ship, ferry or rail, or a pass is closed."
			return p
		}
		for _, c := range path {
			if g.W.Road[g.W.Idx(c[0], c[1])] > 0 {
				p.Road++
			}
		}
		p.Steps += len(path)
		p.Mins += mins
		g.WX, g.WY = l.X, l.Y
	}
	p.Km = float64(p.Steps) * g.tvKm()
	pm, pd, avg := g.atPack()
	p.PackMeal, p.PackDram = pm, pd
	p.Meals, p.Drams = g.atNeed(p.Mins, avg)
	if p.Meals > pm {
		p.Warn = append(p.Warn, fmt.Sprintf("Short of food: need about %d meals, carry %d.", p.Meals, pm))
	}
	if p.Drams > pd {
		p.Warn = append(p.Warn, fmt.Sprintf("Short of water: need about %d drams, carry %d.", p.Drams, pd))
	}
	if g.P.HP*100 < g.P.MaxHP*atLowHPPct {
		p.Warn = append(p.Warn, "You are hurt; rest first.")
	}
	for _, e := range g.dlActive() {
		if e.Known && g.Turn+p.Mins > e.end() {
			p.Warn = append(p.Warn, "Deadline passes before arrival: "+g.dlTitle(e))
		}
	}
	return p
}

func (g *Game) atPlanFor(legs []atDest) *atPlan {
	st := g.at()
	key := fmt.Sprintf("%d,%d,%d,%v,%d,%d", g.WX, g.WY, g.Turn/60, g.canSwim(), g.Hunger/50, len(g.P.Inv))
	for _, l := range legs {
		key += fmt.Sprintf("|%d,%d", l.X, l.Y)
	}
	if st.plan == nil || st.planKey != key {
		st.plan, st.planKey = g.atMakePlan(legs), key
	}
	return st.plan
}

// ---- running a trip

func (g *Game) atNoPack(kind string) bool {
	m, d, _ := g.atPack()
	if kind == "food" {
		return m == 0
	}
	return d == 0
}

func (g *Game) atDlSig() string {
	n, s := 0, 0
	for _, e := range g.dl().Evs {
		if e.live() {
			n++
			s += e.Stage
		} else {
			s += 1000
		}
	}
	return fmt.Sprintf("%d/%d/%d", len(g.dl().Evs), n, s)
}

func atImportant(raw string) bool {
	plain := Plain(raw)
	for _, b := range atBenignPrefixes {
		if strings.HasPrefix(plain, b) {
			return false
		}
	}
	for _, c := range []string{"{{R|", "{{W|", "{{Y|", "{{O|"} {
		if strings.Contains(raw, c) {
			return true
		}
	}
	low := strings.ToLower(plain)
	for _, k := range atImportantWords {
		if strings.Contains(low, k) {
			return true
		}
	}
	return false
}

func (g *Game) atLogTrip(to string, t0, steps int, res string) {
	st := g.at()
	st.Log = append(st.Log, atTrip{To: to, Turn: t0, Days: (g.Turn - t0) / DayLen, Steps: steps, Result: res})
	if len(st.Log) > atMaxLog {
		st.Log = st.Log[len(st.Log)-atMaxLog:]
	}
}

// atGo sets a destination (replacing any queue unless legs are given) and starts walking.
func (g *Game) atGo(legs []atDest) {
	if len(legs) == 0 {
		return
	}
	st := g.at()
	d := legs[0]
	st.Dest = &d
	st.Queue = append([]atDest{}, legs[1:]...)
	g.atRun()
}

// atContinue resumes an interrupted trip.
func (g *Game) atContinue() {
	if g.at().Dest == nil {
		g.Msg("You have no trip to continue.")
		return
	}
	g.atRun()
}

func (g *Game) atRun() {
	st := g.at()
	d := st.Dest
	if d == nil {
		return
	}
	stop := func(why string, steps, t0 int) {
		g.Msg("{{Y|You halt your journey: " + why + ".}} {{K|(q, then Enter: continue the trip)}}")
		g.atLogTrip(d.Name, t0, steps, "halted: "+why)
	}
	if g.ijNoFastTravel() {
		g.Msg("Not in this vow.")
		return
	}
	if d.Planet != g.PlanetIndex() {
		g.Msg("{{R|That place lies on another world.}}")
		st.Dest, st.Queue = nil, nil
		return
	}
	if g.Dead || g.P.HP*100 < g.P.MaxHP*atLowHPPct {
		g.Msg("{{R|You are too hurt to set out; rest first.}}")
		return
	}
	if !g.OnWorld {
		g.openWorldMap()
		if g.Mode != "world" {
			return
		}
	} else {
		g.Mode = "world"
	}
	t0, steps := g.Turn, 0
	if g.WX == d.X && g.WY == d.Y {
		g.atArrive(d, t0, 0)
		return
	}
	path, _, ok := g.wplRoute(d.X, d.Y)
	if !ok {
		g.Msg("{{R|There is no land route to " + d.Name + ": a ship, ferry or rail is needed, or a pass is closed.}}")
		g.atLogTrip(d.Name, t0, 0, "no route")
		return
	}
	g.Msg(fmt.Sprintf("You set out for {{C|%s}}.", d.Name))
	dl0 := g.atDlSig()
	for _, c := range path {
		dx, dy := c[0]-g.WX, c[1]-g.WY
		if abs(dx) > 1 || abs(dy) > 1 {
			break
		}
		hp0, n0, t1 := g.P.HP, len(g.Msgs), g.Turn
		g.WorldMove(dx, dy)
		g.CX, g.CY = g.WX, g.WY
		if g.Dead {
			return
		}
		if g.Mode != "world" || g.WX != c[0] || g.WY != c[1] {
			if g.Mode == "main" && g.nearHostile() {
				stop("hostile in view", steps, t0)
			} else {
				stop("something happened on the way", steps, t0)
			}
			return
		}
		steps++
		if g.P.HP < hp0 {
			stop("you are hurt", steps, t0)
			return
		}
		if g.P.HP*100 < g.P.MaxHP*atLowHPPct {
			stop("your health is low", steps, t0)
			return
		}
		if (g.Hunger >= atHungerWarn && g.atNoPack("food")) || (g.Thirst >= atThirstWarn && g.atNoPack("water")) {
			stop("hunger or thirst with nothing in the pack", steps, t0)
			return
		}
		nn := len(g.Msgs) - n0
		if nn <= 0 || len(g.Msgs) >= 300 {
			nn = 0
			for i := len(g.Msgs) - 1; i >= 0 && g.Msgs[i].Turn > t1; i-- {
				nn++
			}
		}
		for i := maxi(0, len(g.Msgs)-nn); i < len(g.Msgs); i++ {
			if atImportant(g.Msgs[i].T) {
				stop("a notice on the road", steps, t0)
				return
			}
		}
		if s := g.atDlSig(); s != dl0 {
			stop("a deadline event changed", steps, t0)
			return
		}
		np := g.W.Idx(g.WX, g.WY)
		day := (g.Turn + startMinutes) / DayLen
		if gdNight(g.Turn) && g.W.State[np] == 0 && g.W.Burg[np] == 0 && st.NightDay != day {
			st.NightDay = day
			stop("night falls over the wilds; camp, or continue", steps, t0)
			return
		}
	}
	g.atArrive(d, t0, steps)
}

func (g *Game) atArrive(d *atDest, t0, steps int) {
	st := g.at()
	g.Msg(fmt.Sprintf("{{G|You arrive at %s.}} {{K|(Enter steps in)}}", d.Name))
	g.atLogTrip(d.Name, t0, steps, "arrived")
	if len(st.Queue) > 0 {
		n := st.Queue[0]
		st.Queue = st.Queue[1:]
		st.Dest = &n
		g.Msg("{{K|Next leg: " + n.Name + ".}}")
		g.atRun()
		return
	}
	st.Dest = nil
}

// ---- in-zone walk

type atZoneInfo struct {
	Kind atZoneKind
	Dist int // -1 not found
}

func (g *Game) atZoneMatch(k atZoneKind, x, y int) bool {
	t := g.Cur.At(x, y)
	for _, q := range k.Tiles {
		if t == q {
			return true
		}
	}
	return false
}

// atZonePath finds the shortest walk over seen, steppable tiles to the nearest tile of kind k (or beside it
// when the tile itself blocks, like a stall or portcullis). It returns the steps after the start.
func (g *Game) atZonePath(k atZoneKind) ([][2]int, bool) {
	z, p := g.Cur, g.P
	n := MW * MH
	prev := make([]int32, n)
	for i := range prev {
		prev[i] = -2
	}
	start := p.Y*MW + p.X
	prev[start] = -1
	q := []int{start}
	for len(q) > 0 {
		c := q[0]
		q = q[1:]
		cx, cy := c%MW, c/MW
		goal := false
		if g.atZoneMatch(k, cx, cy) {
			goal = true
		} else {
			for _, d := range dirs8 {
				nx, ny := cx+d[0], cy+d[1]
				if z.In(nx, ny) && g.atZoneMatch(k, nx, ny) && !g.qlCanStep(z, nx, ny) && z.Seen[ny*MW+nx] > 0 {
					goal = true
					break
				}
			}
		}
		if goal {
			var rev [][2]int
			for i := c; i != start; i = int(prev[i]) {
				rev = append(rev, [2]int{i % MW, i / MW})
			}
			for i, j := 0, len(rev)-1; i < j; i, j = i+1, j-1 {
				rev[i], rev[j] = rev[j], rev[i]
			}
			return rev, true
		}
		for _, d := range dirs8 {
			nx, ny := cx+d[0], cy+d[1]
			if !z.In(nx, ny) {
				continue
			}
			ni := ny*MW + nx
			if prev[ni] != -2 || z.Seen[ni] == 0 || !g.qlCanStep(z, nx, ny) {
				continue
			}
			prev[ni] = int32(c)
			q = append(q, ni)
		}
	}
	return nil, false
}

func (g *Game) atZoneInfos() []atZoneInfo {
	st := g.at()
	key := fmt.Sprintf("%s/%d,%d/%d", g.Cur.Key, g.P.X, g.P.Y, g.Turn/5)
	if st.zkey == key && st.zinfo != nil {
		return st.zinfo
	}
	out := make([]atZoneInfo, 0, len(atZoneKinds))
	for _, k := range atZoneKinds {
		d := -1
		if path, ok := g.atZonePath(k); ok {
			d = len(path)
		}
		out = append(out, atZoneInfo{k, d})
	}
	st.zkey, st.zinfo = key, out
	return out
}

// atWalk auto-walks to the named in-zone feature and stops on hostiles, wounds and hazards.
func (g *Game) atWalk(kind string) {
	if g.P == nil || g.Dead || g.Cur == nil || g.OnWorld {
		return
	}
	var k *atZoneKind
	for i := range atZoneKinds {
		if atZoneKinds[i].Key == kind {
			k = &atZoneKinds[i]
		}
	}
	if k == nil {
		return
	}
	g.recompute()
	if g.qlHostileVisible() {
		g.Msg("{{R|You cannot auto-walk with enemies in sight!}}")
		return
	}
	z := g.Cur
	steps := 0
	for steps < atWalkSteps && !g.Dead && g.Cur == z && g.Mode == "main" {
		path, ok := g.atZonePath(*k)
		if !ok {
			g.Msg("{{Y|You know of no way to " + k.Label + " from here.}}")
			return
		}
		if len(path) == 0 {
			break
		}
		ox, oy := g.P.X, g.P.Y
		nx, ny := path[0][0], path[0][1]
		wasDoor := z.At(nx, ny) == TDoor
		hp0 := g.P.HP
		g.MoveP(nx-ox, ny-oy)
		moved := g.P.X != ox || g.P.Y != oy || (wasDoor && z.At(nx, ny) == TDoorOpen)
		if !moved {
			g.Msg("Something blocks the way.")
			return
		}
		steps++
		if g.Dead || g.Cur != z || g.Mode != "main" {
			return
		}
		if g.qlHostileVisible() {
			g.Msg("{{R|You spot a hostile creature and stop.}}")
			return
		}
		if g.P.HP < hp0 || g.P.HP*100 < g.P.MaxHP*atLowHPPct {
			g.Msg("{{R|You are hurt and stop.}}")
			return
		}
		if g.qlHazardNear() {
			g.Msg("{{O|You stop near a hazard.}}")
			return
		}
	}
	if steps == 0 {
		g.Msg("You are already at " + k.Label + ".")
		return
	}
	msg := fmt.Sprintf("{{G|You reach %s.}} {{K|(%d steps)}}", k.Label, steps)
	if k.Hint != "" {
		msg += " " + k.Hint
	}
	g.Msg(msg)
}

// ---- abilities (hooks in Abilities / usePower)

// atAbilities: auto-travel and in-zone walk-to entries.
// Hook: powers.go Abilities(): `out = append(out, g.atAbilities()...)`.
func (g *Game) atAbilities() []Ability {
	if g.P == nil || g.W == nil || g.Cur == nil {
		return nil
	}
	out := []Ability{{ID: "at_travel", Name: "Auto-travel (q)", Target: "self"}}
	if st := g.at(); st.Dest != nil {
		out = append(out, Ability{ID: "at_continue", Name: "Continue the trip to " + st.Dest.Name, Target: "self"})
	}
	if !g.OnWorld {
		for _, zi := range g.atZoneInfos() {
			if zi.Dist >= 0 && (zi.Kind.Key == "down" || zi.Kind.Key == "up" || zi.Kind.Key == "gate" || zi.Kind.Key == "door") {
				out = append(out, Ability{ID: "at_w_" + zi.Kind.Key, Name: "Walk to " + zi.Kind.Label, Target: "self"})
			}
		}
	}
	return out
}

// atUse runs at_ abilities. Hook: powers.go usePower(): `if strings.HasPrefix(id, "at_") { g.atUse(id); return }`.
func (g *Game) atUse(id string) {
	switch {
	case id == "at_travel":
		g.atOpen()
	case id == "at_continue":
		g.atContinue()
	case strings.HasPrefix(id, "at_w_"):
		g.atWalk(strings.TrimPrefix(id, "at_w_"))
	}
}

// ---- screen

func atKeys(g *Game, k string) {
	st := g.at()
	var list []atEntry
	if st.tab == 0 && st.stage == 0 {
		st.entries = g.atEntries()
		list = st.entries
	}
	switch k {
	case "Escape", "q":
		if st.stage == 1 {
			st.stage = 0
			return
		}
		g.atClose()
		return
	case "Tab", "ArrowRight":
		st.tab, st.sel, st.stage = (st.tab+1)%3, 0, 0
		return
	case "ArrowLeft":
		st.tab, st.sel, st.stage = (st.tab+2)%3, 0, 0
		return
	case "1", "2", "3":
		st.tab, st.sel, st.stage = int(k[0]-'1'), 0, 0
		return
	case "c":
		if st.Dest != nil {
			g.atClose()
			g.atContinue()
		}
		return
	case "x":
		st.Queue = nil
		g.Msg("You clear your itinerary.")
		return
	}
	switch st.tab {
	case 0:
		atListKeys(g, k, list)
	case 1:
		infos := g.atZoneInfos()
		switch k {
		case "ArrowDown", "j":
			st.sel = mini(st.sel+1, len(infos)-1)
		case "ArrowUp", "k":
			st.sel = maxi(0, st.sel-1)
		case "Enter", "g", " ":
			if g.OnWorld || g.Cur == nil {
				g.Msg("Walk-to works inside a zone.")
				return
			}
			if st.sel < len(infos) {
				key := infos[st.sel].Kind.Key
				g.Mode = "main"
				g.atWalk(key)
			}
		}
	}
}

func atListKeys(g *Game, k string, list []atEntry) {
	st := g.at()
	st.sel = clampi(st.sel, 0, maxi(0, len(list)-1))
	switch k {
	case "ArrowDown", "j":
		st.sel = mini(st.sel+1, maxi(0, len(list)-1))
	case "ArrowUp", "k":
		st.sel = maxi(0, st.sel-1)
	case "a":
		if st.stage == 0 && len(list) > 0 {
			e := list[st.sel]
			if len(st.Queue) >= atMaxQueue {
				g.Msg("Your itinerary is full.")
				return
			}
			st.Queue = append(st.Queue, atDest{e.Name, e.X, e.Y, g.PlanetIndex()})
			g.Msg("Added to your itinerary: " + e.Name + ".")
		}
	case "Enter", "g":
		if len(list) == 0 && len(st.Queue) == 0 {
			return
		}
		if st.stage == 0 {
			st.stage = 1
			st.plan, st.planKey = nil, ""
			return
		}
		legs := g.atLegs(list)
		if len(legs) == 0 {
			return
		}
		p := g.atPlanFor(legs)
		if !p.OK {
			g.Msg("{{R|" + p.Why + "}}")
			return
		}
		st.stage = 0
		g.atClose()
		g.atGo(legs)
	}
}

// atLegs is the itinerary when one is queued (key g, or the chosen entry then the queue), else the entry.
func (g *Game) atLegs(list []atEntry) []atDest {
	st := g.at()
	var legs []atDest
	if st.sel < len(list) {
		e := list[st.sel]
		legs = append(legs, atDest{e.Name, e.X, e.Y, g.PlanetIndex()})
	}
	if len(st.Queue) > 0 {
		if len(legs) == 0 {
			return append([]atDest{}, st.Queue...)
		}
		legs = append(legs, st.Queue...)
	}
	return legs
}

func atDraw(g *Game) {
	s := g.sc
	st := g.at()
	s.Fill(0, 0, SW, SH, CBgDark)
	s.Box(0, 0, SW, SH, CDarkCyan, "Auto-travel")
	tx := 2
	for i, t := range []string{"Destinations", "Walk (zone)", "Trip log"} {
		c := "K"
		if st.tab == i {
			c = "Y"
		}
		tx = s.TextB(tx, 1, fmt.Sprintf("{{%s|[ %d %s ]}} ", c, i+1, t), CGray, CBgDark, SW-1)
	}
	switch st.tab {
	case 0:
		atDrawDest(g)
	case 1:
		atDrawWalk(g)
	default:
		atDrawLog(g)
	}
}

func atDrawLog(g *Game) {
	s := g.sc
	st := g.at()
	y := 4
	if st.Dest != nil {
		s.TextB(3, y, fmt.Sprintf("{{Y|Interrupted trip}} to {{C|%s}} {{K|(press c to continue)}}", st.Dest.Name), CGray, CBgDark, SW-2)
		y += 2
	}
	if len(st.Log) == 0 {
		s.TextB(3, y, "{{K|No trips yet.}}", CGray, CBgDark, SW-2)
	}
	for i := len(st.Log) - 1; i >= 0 && y < SH-4; i-- {
		t := st.Log[i]
		col := "G"
		if t.Result != "arrived" {
			col = "Y"
		}
		s.TextB(3, y, fmt.Sprintf("{{W|%-26s}} {{K|%s}} %3d regions  {{%s|%s}}", atCut(t.To, 26), g.atWhen(t.Turn), t.Steps, col, t.Result), CGray, CBgDark, SW-2)
		y++
	}
	s.TextB(3, SH-3, "{{K|Tab/1-3 pages · c continue · Esc}}", CGray, CBgDark, SW-2)
}

func (g *Game) atWhen(turn int) string {
	return fmt.Sprintf("day %d", DayOf(turn)+1)
}

func atCut(s string, n int) string {
	if r := []rune(s); len(r) > n {
		return string(r[:n])
	}
	return s
}

func atDrawWalk(g *Game) {
	s := g.sc
	st := g.at()
	if g.OnWorld || g.Cur == nil {
		s.TextB(3, 4, "{{K|Walk-to works inside a zone: leave the world map first.}}", CGray, CBgDark, SW-2)
		return
	}
	s.TextB(3, 3, "{{W|Walk to...}}  {{K|(only places you have seen; qol Tab/v explores)}}", CGray, CBgDark, SW-2)
	infos := g.atZoneInfos()
	st.sel = clampi(st.sel, 0, len(infos)-1)
	for i, zi := range infos {
		bg := CBgDark
		if i == st.sel {
			bg = CBgSel
		}
		s.Fill(3, 5+i, 60, 1, bg)
		d := "{{K|not seen}}"
		if zi.Dist >= 0 {
			d = fmt.Sprintf("{{G|%d steps}}", zi.Dist)
		}
		s.TextB(4, 5+i, fmt.Sprintf("{{W|%-30s}} %s", zi.Kind.Label, d), CGray, bg, 64)
	}
	s.TextB(3, SH-3, "{{K|↑↓ choose · Enter walk there · Tab/1-3 pages · Esc}}", CGray, CBgDark, SW-2)
}

func atDrawDest(g *Game) {
	s := g.sc
	st := g.at()
	st.entries = g.atEntries()
	list := st.entries
	st.sel = clampi(st.sel, 0, maxi(0, len(list)-1))
	if st.stage == 1 {
		atDrawPreview(g, list)
		return
	}
	s.TextB(3, 3, fmt.Sprintf("{{c|%s}}   {{K|you stand at %d,%d · %.0f km per region}}", g.unPlanetName(), g.WX, g.WY, g.tvKm()), CGray, CBgDark, SW-2)
	y, last := 5, ""
	left := 54
	for i, e := range list {
		if y >= SH-5 {
			break
		}
		if e.Group != last {
			s.TextB(3, y, "{{Y|"+e.Group+"}}", CGray, CBgDark, 3+left)
			y++
			last = e.Group
		}
		bg := CBgDark
		if i == st.sel {
			bg = CBgSel
		}
		s.Fill(4, y, left, 1, bg)
		km := math.Hypot(float64(e.X-g.WX), float64(e.Y-g.WY)) * g.tvKm()
		s.TextB(5, y, fmt.Sprintf("{{W|%-30s}} {{K|%s %5.0f km}}", atCut(e.Name, 30), wplCompass(float64(e.X-g.WX), float64(e.Y-g.WY)), km), CGray, bg, 4+left)
		y++
	}
	if len(list) == 0 {
		s.TextB(4, y, "{{K|No destinations known yet.}}", CGray, CBgDark, SW-2)
	}
	dx := 3 + left + 3
	dw := SW - dx - 3
	if len(list) > 0 {
		e := list[st.sel]
		s.TextB(dx, 5, "{{Y|"+e.Name+"}}", CGray, CBgDark, dx+dw)
		s.TextB(dx, 7, fmt.Sprintf("{{K|Region}} %d,%d", e.X, e.Y), CGray, CBgDark, dx+dw)
		if e.Note != "" {
			for j, l := range Wrap("{{c|"+e.Note+"}}", dw) {
				s.TextB(dx, 8+j, l, CGray, CBgDark, dx+dw)
			}
		}
	}
	if len(st.Queue) > 0 {
		s.TextB(dx, 12, "{{Y|Itinerary}} {{K|(g or Enter goes: selection first, then these; x clears)}}", CGray, CBgDark, dx+dw)
		for i, q := range st.Queue {
			s.TextB(dx, 13+i, fmt.Sprintf("{{W|%d. %s}}", i+1, atCut(q.Name, dw-4)), CGray, CBgDark, dx+dw)
		}
	}
	if st.Dest != nil {
		s.TextB(dx, SH-6, fmt.Sprintf("{{Y|Interrupted:}} %s  {{K|(c continues)}}", atCut(st.Dest.Name, dw-24)), CGray, CBgDark, dx+dw)
	}
	s.TextB(3, SH-3, "{{K|↑↓ choose · Enter preview route · a add to itinerary · c continue trip · x clear · Tab pages · Esc}}", CGray, CBgDark, SW-2)
}

func atDrawPreview(g *Game, list []atEntry) {
	s := g.sc
	legs := g.atLegs(list)
	if len(legs) == 0 {
		return
	}
	p := g.atPlanFor(legs)
	y := 4
	names := make([]string, len(legs))
	for i, l := range legs {
		names[i] = l.Name
	}
	s.TextB(3, y, "{{Y|Route preview}}  "+strings.Join(names, " > "), CGray, CBgDark, SW-2)
	y += 2
	if !p.OK {
		s.TextB(3, y, "{{R|"+p.Why+"}}", CGray, CBgDark, SW-2)
		s.TextB(3, SH-3, "{{K|Esc back}}", CGray, CBgDark, SW-2)
		return
	}
	pct := 0
	if p.Steps > 0 {
		pct = p.Road * 100 / p.Steps
	}
	s.TextB(3, y, fmt.Sprintf("{{K|Distance}} {{W|%.0f km}} over %d regions ({{W|%d%%}} on roads)   {{K|Time}} {{W|%s}} %s", p.Km, p.Steps, pct, tvDuration(p.Mins), g.trHow()), CGray, CBgDark, SW-2)
	y += 2
	fc, wc := "G", "G"
	if p.Meals > p.PackMeal {
		fc = "R"
	}
	if p.Drams > p.PackDram {
		wc = "R"
	}
	s.TextB(3, y, fmt.Sprintf("{{K|Food}} need about {{%s|%d}} meals, carry %d    {{K|Water}} need about {{%s|%d}} drams, carry %d", fc, p.Meals, p.PackMeal, wc, p.Drams, p.PackDram), CGray, CBgDark, SW-2)
	y += 2
	for _, w := range p.Warn {
		s.TextB(3, y, "{{R|! "+w+"}}", CGray, CBgDark, SW-2)
		y++
	}
	y++
	s.TextB(3, y, "{{K|Stops for: hostiles, wounds, hunger or thirst with an empty pack, deadline events, notices, night in the wilds.}}", CGray, CBgDark, SW-2)
	s.TextB(3, SH-3, "{{K|Enter depart · Esc back}}", CGray, CBgDark, SW-2)
}
