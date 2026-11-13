package engine

import (
	"fmt"
	"strings"
)

// Steampunk / dieselpunk / atompunk signature: boilers, fuel and machines (prefix gd). A workshop bench
// (mode gdm, tile device or smith talk) assembles a clockwork automaton from crafting bits: chassis,
// power core and weapon mount. It follows you as a Summoned-style follower, burns fuel by the minute
// (coal, oil, steam cells; names by genre), builds heat while fighting (overdrive doubles both), and
// overheats at the critical mark: a shutdown and a scalding steam burst, wear that needs bits to mend.
// Upgrades, a steam press that refines bits and warms the workshop, and fuel and mending for sky craft.

type gdAuto struct {
	ID      int             `json:"id"`
	Name    string          `json:"n"`
	Chassis string          `json:"ch"`
	Core    string          `json:"co"`
	Mount   string          `json:"mt"`
	Q       int             `json:"q"`
	Fuel    int             `json:"f"`
	Acc     int             `json:"ac,omitempty"`
	Heat    float64         `json:"h,omitempty"`
	Wear    int             `json:"w,omitempty"`
	Push    bool            `json:"p,omitempty"`
	Upg     map[string]bool `json:"u,omitempty"`
	Wreck   bool            `json:"wr,omitempty"`
	Warn    int             `json:"wn,omitempty"`
	Vent    int             `json:"vt,omitempty"`
}

type gdWorkSel struct{ ch, co, mt int }

type gdChassis struct {
	ID, Name         string
	HP, Spd, AV, STR int
	Cost             map[int]int
}
type gdCore struct {
	ID, Name  string
	Cap, Burn int
	Heat      float64
	Cost      map[int]int
	Atom      bool
}
type gdMount struct {
	ID, Name   string
	DN, DS, PV int
	Elem       string
	Cost       map[int]int
}

var gdChassisList = []gdChassis{
	{"scrapper", "scrapper frame", 30, 110, 2, 14, map[int]int{1: 3, 2: 2}},
	{"walker", "walker frame", 60, 100, 4, 18, map[int]int{2: 4, 3: 3}},
	{"colossus", "colossus frame", 100, 85, 7, 22, map[int]int{4: 4, 5: 4}},
}
var gdCoreList = []gdCore{
	{"boiler", "fire-tube boiler", 480, 100, 2.0, map[int]int{2: 3, 3: 2}, false},
	{"engine", "oil engine", 720, 70, 1.5, map[int]int{3: 3, 4: 2}, false},
	{"pile", "isotope pile", 1440, 40, 3.0, map[int]int{5: 3, 6: 2}, true},
}
var gdMountList = []gdMount{
	{"rivet", "rivet gun", 1, 8, 3, "", map[int]int{3: 2}},
	{"ram", "piston ram", 2, 4, 4, "", map[int]int{3: 3}},
	{"flame", "flame jet", 1, 6, 2, "fire", map[int]int{4: 2}},
	{"coil", "shock coil", 1, 6, 2, "shock", map[int]int{4: 3}},
}

type gdUpgrade struct {
	ID, Name, Desc string
	Cost           map[int]int
}

var gdUpgrades = []gdUpgrade{
	{"plating", "armour plating", "+2 AV", map[int]int{3: 2}},
	{"flues", "wide flues", "heat builds 25% slower", map[int]int{2: 2, 3: 1}},
	{"governor", "pressure governor", "vents itself at 90 heat", map[int]int{4: 2}},
	{"tank", "auxiliary tank", "+240 fuel capacity", map[int]int{3: 2}},
	{"servo", "reinforced mount", "+1 damage die size and PV", map[int]int{4: 3}},
	{"loop", "coolant loop", "cools faster", map[int]int{3: 2, 4: 1}},
}

var gdFuelDefs = map[string]struct {
	Mins, Price int
	Names       map[string]string
	Only        bool
}{
	"coal": {60, 4, map[string]string{"steampunk": "coal lump", "dieselpunk": "coal brick", "atompunk": "graphite rod"}, false},
	"oil":  {120, 10, map[string]string{"steampunk": "naphtha flask", "dieselpunk": "diesel can", "atompunk": "heavy-water jug"}, false},
	"cell": {300, 30, map[string]string{"steampunk": "steam cell", "dieselpunk": "fuel accumulator", "atompunk": "isotope cell"}, false},
}
var gdFuelOrder = []string{"coal", "oil", "cell"}

func (g *Game) gdFuelName(k string) string {
	if n := gdFuelDefs[k].Names[g.Genre]; n != "" {
		return n
	}
	return k
}

func (g *Game) gdFuelItem(k string, n int) *Item {
	d := gdFuelDefs[k]
	return &Item{ID: g.newID(), Kind: "gdfuel", Effect: k, Name: g.gdFuelName(k), Glyph: '=', F: map[string]byte{"coal": CDarkGray, "oil": CBrown, "cell": CCyan}[k],
		Count: n, Value: d.Price * n / 2, Weight: 1, Lore: fmt.Sprintf("Fuel for boilers and engines: %s of run time each.", gdHours(d.Mins))}
}

func init() {
	zoneHooks = append(zoneHooks, gdSteamZone)
	killHooks = append(killHooks, gdSteamKill)
	talkHooks = append(talkHooks, gdSteamTalk)
	turnHooks = append(turnHooks, gdSteamTurn)
	sideHooks = append(sideHooks, gdSteamSide)
	gdDevOpen["bench"] = func(g *Game, d *Item) { g.gdWorkOpen() }
	gdDevOpen["press"] = gdPressOpen
}

func gdSteamZone(g *Game, zn *Zone, _ *Rng) {
	if !gdSteamy(g) || !gdTown(g, zn) || zn.Kind != "town" {
		return
	}
	r := NewRng(g.Seed + ":gd:steam:" + zn.Key)
	if !r.Chance(0.4) {
		return
	}
	var ax, ay int
	var ok bool
	if ax, ay, ok = g.gdFreeSpot(zn, r, nil); !ok {
		return
	}
	g.gdPlaceDev(zn, "bench", ax, ay, 1)
	pp := map[string]float64{"steampunk": 0.6, "dieselpunk": 0.4, "atompunk": 0.3}[g.Genre]
	if r.Chance(pp) {
		if x, y, ok := g.gdFreeSpot(zn, r, func(x, y int) bool { return abs(x-ax) <= 6 && abs(y-ay) <= 5 }); ok {
			g.gdPlaceDev(zn, "press", x, y, 1)
		}
	}
}

// ---- automaton ----

func gdChassisOf(id string) *gdChassis {
	for i := range gdChassisList {
		if gdChassisList[i].ID == id {
			return &gdChassisList[i]
		}
	}
	return &gdChassisList[0]
}
func gdCoreOf(id string) *gdCore {
	for i := range gdCoreList {
		if gdCoreList[i].ID == id {
			return &gdCoreList[i]
		}
	}
	return &gdCoreList[0]
}
func gdMountOf(id string) *gdMount {
	for i := range gdMountList {
		if gdMountList[i].ID == id {
			return &gdMountList[i]
		}
	}
	return &gdMountList[0]
}

func (au *gdAuto) cap() int {
	c := gdCoreOf(au.Core).Cap
	if au.Upg["tank"] {
		c += 240
	}
	return c
}

// gdAutoApply recomputes the actor's stats from its record (idempotent).
func (g *Game) gdAutoApply(a *Actor, au *gdAuto) {
	ch, mt := gdChassisOf(au.Chassis), gdMountOf(au.Mount)
	mul := 100 + 12*(au.Q-1)
	maxhp := ch.HP * mul / 100 * (100 - 4*mini(au.Wear, 8)) / 100
	if a.MaxHP > 0 && a.HP > 0 {
		a.HP = maxi(1, a.HP*maxhp/a.MaxHP)
	}
	a.MaxHP = maxhp
	a.HP = mini(a.HP, maxhp)
	a.Speed = ch.Spd
	a.AVNat = ch.AV
	if au.Upg["plating"] {
		a.AVNat += 2
	}
	a.Attr[STR] = ch.STR + au.Q
	ds, pv := mt.DS, mt.PV
	if au.Upg["servo"] {
		ds++
		pv++
	}
	if au.Push {
		a.Speed += 25
		ds += 2
		pv += 2
	}
	for _, p := range a.Body {
		if p.Slot == "hand" {
			p.Natural = natural(mt.Name, mt.DN, ds+au.Q/2, pv, mt.Elem)
			break
		}
	}
}

var gdAutoNames = []string{"Cog", "Brass", "Piston", "Rivet", "Sprocket", "Ratchet", "Clank", "Tinder"}

func (g *Game) gdWorkOpen() {
	st := g.gd()
	st.wsel = gdWorkSel{}
	g.gdOpen("gdm", "Workshop", func(u *gdUI) { gdWorkBuild(g, u) })
}

func (g *Game) gdPressWarm() bool {
	if g.Cur == nil {
		return false
	}
	for _, d := range g.gdDevs(g.Cur, "press") {
		if g.gd().Press[gdDevKey(g.Cur, d)] > g.Turn {
			return true
		}
	}
	return false
}

func (g *Game) gdAutoHere() (*gdAuto, *Actor) {
	au := g.gd().Auto
	if au == nil || au.Wreck {
		return au, nil
	}
	a, z := g.gdFindActor(au.ID)
	if z != g.Cur {
		return au, nil
	}
	return au, a
}

func gdWorkBuild(g *Game, u *gdUI) {
	st := g.gd()
	sel := &st.wsel
	au := st.Auto
	warm := g.gdPressWarm()
	u.line("{{C|Bits:}} " + g.craftSt().bitsLine() + ".   Tinkering helps; a warm steam press halves the time and adds quality.")
	if au == nil || au.Wreck {
		cores := []gdCore{}
		for _, c := range gdCoreList {
			if !c.Atom || g.Genre == "atompunk" {
				cores = append(cores, c)
			}
		}
		sel.ch %= len(gdChassisList)
		sel.co %= len(cores)
		sel.mt %= len(gdMountList)
		ch, co, mt := gdChassisList[sel.ch], cores[sel.co], gdMountList[sel.mt]
		cost := map[int]int{}
		for _, m := range []map[int]int{ch.Cost, co.Cost, mt.Cost} {
			for k, v := range m {
				cost[k] += v
			}
		}
		if au != nil && au.Wreck {
			for k, v := range cost {
				cost[k] = (v + 1) / 2
			}
			u.line("{{R|Your automaton was wrecked.}} Rebuilding costs half.")
		}
		u.row("Chassis: "+ch.Name+fmt.Sprintf(" {{K|(%d HP, speed %d, AV %d)}}", ch.HP, ch.Spd, ch.AV), func() { sel.ch++ })
		u.row("Power core: "+co.Name+fmt.Sprintf(" {{K|(%s fuel, heat x%.1f)}}", gdHours(co.Cap), co.Heat), func() { sel.co++ })
		u.row("Weapon mount: "+mt.Name+fmt.Sprintf(" {{K|(%dd%d, PV %d %s)}}", mt.DN, mt.DS, mt.PV, mt.Elem), func() { sel.mt++ })
		u.line("Cost: " + gdCostLine(cost))
		if !g.gdHasBits(cost) {
			u.off("Assemble the automaton (not enough bits)")
		} else if g.companionCount() >= maxCompanions {
			u.off("Assemble the automaton (you lead as many companions as you can)")
		} else {
			u.row("{{W|Assemble the automaton.}}", func() {
				g.gdPayBits(cost)
				g.gdBuildAuto(ch, co, mt, warm)
			})
		}
	} else {
		_, a := g.gdAutoHere()
		g.gdAutoLines(u, au, a)
		if a != nil {
			g.gdAutoRows(u, au, a)
			for _, up := range gdUpgrades {
				up := up
				if au.Upg[up.ID] {
					continue
				}
				if !g.gdHasBits(up.Cost) {
					u.off(fmt.Sprintf("Upgrade: %s, %s (%s)", up.Name, up.Desc, gdCostLine(up.Cost)))
					continue
				}
				u.row(fmt.Sprintf("Upgrade: %s, %s {{K|(%s)}}", up.Name, up.Desc, gdCostLine(up.Cost)), func() {
					g.gdPayBits(up.Cost)
					if au.Upg == nil {
						au.Upg = map[string]bool{}
					}
					au.Upg[up.ID] = true
					g.gdAutoApply(a, au)
					g.spendTurns(10)
					g.Msg("{{G|You fit " + up.Name + " to " + a.Name + ".}}")
				})
			}
			u.row("{{r|Dismantle it for half its bits.}}", func() {
				g.gdAddBits(gdAutoCost(au), 50)
				g.removeActor(g.Cur, a)
				st.Auto = nil
				g.Msg("You strip the automaton down to parts.")
			})
		} else {
			u.line("{{K|It is not here; bring it to the workshop.}}")
		}
	}
	u.gap()
	u.line("{{Y|Fuel store}}")
	for _, k := range gdFuelOrder {
		k := k
		d := gdFuelDefs[k]
		pr := d.Price * 3
		u.row(fmt.Sprintf("Buy 3 %s {{K|(%s; you hold %d)}}", g.gdFuelName(k), g.gdPrice(pr), g.gdCount("gdfuel", k)), func() {
			if g.gdPay(pr) {
				g.gdGive(g.gdFuelItem(k, 3))
			}
		})
	}
	if jy := g.jy(); jy.Craft != "" {
		if jd := jy.def(g); jd != nil {
			if jd.FuelMax > 0 && jy.Fuel < jd.FuelMax {
				for _, k := range gdFuelOrder {
					k := k
					if g.gdCount("gdfuel", k) == 0 {
						continue
					}
					days := float64(gdFuelDefs[k].Mins) / 240
					u.row(fmt.Sprintf("Refuel your %s with 1 %s (+%.1f days)", jd.name(g), g.gdFuelName(k), days), func() {
						g.gdTake("gdfuel", k, 1)
						jy.Fuel = minF(jd.FuelMax, jy.Fuel+days)
					})
				}
			}
			if jy.Hull < jd.Hull {
				cost := map[int]int{2: 1}
				if g.gdHasBits(cost) {
					u.row(fmt.Sprintf("Mend your %s's hull (%s: +20)", jd.name(g), gdCostLine(cost)), func() {
						g.gdPayBits(cost)
						jy.Hull = mini(jd.Hull, jy.Hull+20)
					})
				}
			}
		}
	}
}

func minF(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func gdAutoCost(au *gdAuto) map[int]int {
	cost := map[int]int{}
	for _, m := range []map[int]int{gdChassisOf(au.Chassis).Cost, gdCoreOf(au.Core).Cost, gdMountOf(au.Mount).Cost} {
		for k, v := range m {
			cost[k] += v
		}
	}
	return cost
}

func (g *Game) gdBuildAuto(ch gdChassis, co gdCore, mt gdMount, warm bool) {
	st := g.gd()
	r := g.gdRng("auto", g.Turn, ch.ID, co.ID)
	q := clampi(1+g.tinkRank()+boolI(g.Skills["lore"] > 0)+boolI(warm)+r.Int(0, 1), 1, 5)
	x, y, ok := g.nearestFree(g.Cur, g.P.X+1, g.P.Y)
	if !ok {
		x, y = g.P.X, g.P.Y
	}
	name := Pick(r, gdAutoNames)
	a := &Actor{ID: g.newID(), Name: name, Proper: true, Glyph: 'Φ', F: CCyan, Tpl: "golem", Faction: "player", Level: clampi(g.P.Level, 1, 20),
		Attr: [6]int{16, 10, 16, 6, 10, 4}, AI: "follower", Body: MakeBody("humanoid"), X: x, Y: y, Title: "clockwork " + ch.ID,
		Comp: &Companion{Loyalty: 100, Summoned: true, Since: g.Turn}}
	a.Set("gdauto")
	a.Set("armed")
	au := &gdAuto{ID: a.ID, Name: name, Chassis: ch.ID, Core: co.ID, Mount: mt.ID, Q: q, Upg: map[string]bool{}}
	a.MaxHP = 0
	g.gdAutoApply(a, au)
	a.HP = a.MaxHP
	g.Cur.Actors = append(g.Cur.Actors, a)
	st.Auto = au
	if warm {
		g.spendTurns(30)
	} else {
		g.spendTurns(60)
	}
	g.Msg(fmt.Sprintf("{{G|%s the %s shudders awake (quality %d). It needs fuel.}}", name, ch.Name, q))
	g.gdFirst("auto", "built a first clockwork automaton.")
}

func (g *Game) gdAutoLines(u *gdUI, au *gdAuto, a *Actor) {
	hp := ""
	if a != nil {
		hp = fmt.Sprintf(" HP %d/%d.", a.HP, a.MaxHP)
	}
	u.line(fmt.Sprintf("{{C|%s}}, %s, %s, %s. Quality %d, wear %d.%s", au.Name, gdChassisOf(au.Chassis).Name, gdCoreOf(au.Core).Name, gdMountOf(au.Mount).Name, au.Q, au.Wear, hp))
	u.line(fmt.Sprintf("Fuel %s %s   Heat %s %d%%%s", gdBar(au.Fuel, au.cap(), 12), gdHours(au.Fuel), gdBar(int(au.Heat), 100, 10), int(au.Heat), map[bool]string{true: "   {{R|OVERDRIVE}}", false: ""}[au.Push]))
}

func (g *Game) gdAutoRows(u *gdUI, au *gdAuto, a *Actor) {
	for _, k := range gdFuelOrder {
		k := k
		if g.gdCount("gdfuel", k) == 0 || (gdCoreOf(au.Core).Atom && k != "cell") || au.Fuel >= au.cap() {
			continue
		}
		u.row(fmt.Sprintf("Fuel it with 1 %s (+%s)", g.gdFuelName(k), gdHours(gdFuelDefs[k].Mins)), func() { g.gdRefuel(au, a, k) })
	}
	u.row("Vent the boiler (-40 heat, costs 20 fuel).", func() { g.gdVent(au, a) })
	u.row(map[bool]string{true: "Ease off overdrive.", false: "Push it: overdrive (faster, harder, hotter, thirstier)."}[au.Push], func() {
		au.Push = !au.Push
		g.gdAutoApply(a, au)
	})
	if au.Wear > 0 || a.HP < a.MaxHP {
		cost := map[int]int{1: 1, 2: 1}
		if g.gdHasBits(cost) {
			u.row(fmt.Sprintf("Repair: patch plating, clear wear (%s)", gdCostLine(cost)), func() { g.gdRepair(au, a, cost) })
		} else {
			u.off("Repair (needs " + gdCostLine(cost) + ")")
		}
	}
}

func (g *Game) gdRefuel(au *gdAuto, a *Actor, k string) {
	if !g.gdTake("gdfuel", k, 1) {
		return
	}
	au.Fuel = mini(au.cap(), au.Fuel+gdFuelDefs[k].Mins)
	if au.Fuel > 0 {
		delete(a.Status, "stunned")
	}
	g.Msg("{{G|" + a.Name + " takes the fuel and hisses.}}")
	g.spendTurns(2)
}

func (g *Game) gdVent(au *gdAuto, a *Actor) {
	au.Heat = maxF(0, au.Heat-40)
	au.Fuel = maxi(0, au.Fuel-20)
	g.Msg("{{C|" + a.Name + " vents a long white plume.}}")
	g.spendTurns(1)
}

func maxF(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func (g *Game) gdRepair(au *gdAuto, a *Actor, cost map[int]int) {
	if !g.gdPayBits(cost) {
		return
	}
	au.Wear = maxi(0, au.Wear-1)
	g.gdAutoApply(a, au)
	a.HP = mini(a.MaxHP, a.HP+a.MaxHP*30/100)
	g.spendTurns(6)
	g.Msg("{{G|You patch " + a.Name + " up.}}")
}

// gdCompOptions: one-line hook in companionTalk for the automaton's field care.
func gdCompOptions(g *Game, a *Actor) []Option {
	if !a.Has("gdauto") {
		return nil
	}
	au := g.gd().Auto
	if au == nil || au.ID != a.ID {
		return nil
	}
	var out []Option
	say := func(t string) { g.companionTalk(a, t) }
	out = append(out, Option{"{{C|Check the gauges.}}", func() {
		say(fmt.Sprintf("Fuel %s (%s left). Heat %d%%%s. Wear %d. HP %d/%d.", gdBar(au.Fuel, au.cap(), 12), gdHours(au.Fuel), int(au.Heat), map[bool]string{true: ", overdrive on", false: ""}[au.Push], au.Wear, a.HP, a.MaxHP))
	}})
	for _, k := range gdFuelOrder {
		k := k
		if g.gdCount("gdfuel", k) == 0 || (gdCoreOf(au.Core).Atom && k != "cell") || au.Fuel >= au.cap() {
			continue
		}
		out = append(out, Option{fmt.Sprintf("{{O|Fuel it with 1 %s.}}", g.gdFuelName(k)), func() { g.gdRefuel(au, a, k); say("It takes the fuel.") }})
	}
	out = append(out, Option{"{{O|Vent the boiler.}}", func() { g.gdVent(au, a); say("A plume of steam.") }})
	out = append(out, Option{map[bool]string{true: "{{O|Ease off overdrive.}}", false: "{{R|Push it: overdrive.}}"}[au.Push], func() {
		au.Push = !au.Push
		g.gdAutoApply(a, au)
		say("The pitch of the engine changes.")
	}})
	cost := map[int]int{1: 1, 2: 1}
	if (au.Wear > 0 || a.HP < a.MaxHP) && g.gdHasBits(cost) {
		out = append(out, Option{fmt.Sprintf("{{O|Repair it.}} {{K|(%s)}}", gdCostLine(cost)), func() { g.gdRepair(au, a, cost); say("Good as it gets.") }})
	}
	return out
}

func gdSteamKill(g *Game, d, src *Actor) {
	if d == nil || !d.Has("gdauto") {
		return
	}
	if au := g.gd().Auto; au != nil && au.ID == d.ID {
		au.Wreck = true
		g.gdAddBits(gdAutoCost(au), 25)
		g.Msg("{{R|" + d.Name + " collapses in a cloud of steam. The workshop can rebuild it for half the bits.}}")
	}
}

func gdSteamTurn(g *Game) {
	if !gdSteamy(g) || g.Cur == nil {
		return
	}
	st := g.gd()
	au := st.Auto
	if au == nil || au.Wreck {
		return
	}
	a, z := g.gdFindActorFast(au.ID)
	if a == nil {
		if g.Turn%10 == 0 {
			if aa, _ := g.gdFindActor(au.ID); aa == nil {
				au.Wreck = true
				g.Msg("{{R|You sense that your automaton is lost for good.}}")
			}
		}
		return
	}
	_ = z
	if a.AI != "follower" || a.Order == OrdStay || a.Order == OrdGuard {
		au.Heat = maxF(0, au.Heat-1.5)
		return
	}
	co := gdCoreOf(au.Core)
	rate := co.Burn
	if au.Push {
		rate *= 2
	}
	au.Acc += rate
	for au.Acc >= 100 && au.Fuel > 0 {
		au.Fuel--
		au.Acc -= 100
	}
	if au.Fuel <= 0 {
		au.Acc = 0
		a.AddSt("stunned", 3)
		if g.Turn-au.Warn > 30 {
			au.Warn = g.Turn
			g.Msg("{{R|" + a.Name + " coughs and stills: out of fuel.}}")
		}
		return
	}
	combat := false
	for _, o := range g.Cur.Actors {
		if o.HP > 0 && o != a && g.Hostile(o) && abs(o.X-a.X) <= 8 && abs(o.Y-a.Y) <= 8 {
			combat = true
			break
		}
	}
	if combat {
		gain := co.Heat
		if au.Push {
			gain *= 2
		}
		if au.Upg["flues"] {
			gain *= 0.75
		}
		if au.Mount == "flame" {
			gain *= 1.3
		}
		au.Heat += gain
	} else {
		cool := 1.5
		if au.Upg["loop"] {
			cool = 2.5
		}
		au.Heat = maxF(0, au.Heat-cool)
	}
	if au.Upg["governor"] && au.Heat >= 90 && g.Turn-au.Vent > 30 {
		au.Vent = g.Turn
		au.Heat, au.Fuel = 50, maxi(0, au.Fuel-15)
		g.Msg("{{C|" + a.Name + "'s governor vents a hissing plume.}}")
	}
	switch {
	case au.Heat >= 100:
		au.Heat = 55
		au.Wear++
		a.AddSt("stunned", 8)
		for _, o := range g.Cur.Actors {
			if o.HP > 0 && o != a && abs(o.X-a.X) <= 1 && abs(o.Y-a.Y) <= 1 {
				g.hurt(o, 4+au.Q, a, "scalding steam")
			}
		}
		if abs(g.P.X-a.X) <= 1 && abs(g.P.Y-a.Y) <= 1 {
			g.hurt(g.P, 3, a, "scalding steam")
		}
		g.hurt(a, maxi(1, a.MaxHP/10), nil, "overheating")
		g.gdAutoApply(a, au)
		g.Msg("{{R|" + a.Name + " overheats! A steam burst and a shutdown. (wear " + fmt.Sprint(au.Wear) + ")}}")
	case au.Heat >= 85 && g.Turn-au.Warn > 15:
		au.Warn = g.Turn
		g.Msg("{{R|" + a.Name + "'s pressure gauge is in the red! Ease off or vent.}}")
	case au.Heat >= 70 && g.Turn-au.Warn > 30:
		au.Warn = g.Turn
		g.Msg("{{y|Steam shrieks from " + a.Name + "'s valves.}}")
	}
	if au.Wear >= 5 && g.Turn%120 == 0 {
		a.AddSt("stunned", 5)
		g.Msg("{{y|" + a.Name + " seizes up for a moment. It needs repairs.}}")
	}
}

// gdFindActorFast checks the current zone first (the common case), never scanning other zones.
func (g *Game) gdFindActorFast(id int) (*Actor, *Zone) {
	for _, a := range g.Cur.Actors {
		if a.ID == id && a.HP > 0 {
			return a, g.Cur
		}
	}
	return nil, nil
}

func gdSteamSide(g *Game) []string {
	if !gdSteamy(g) {
		return nil
	}
	au := g.gd().Auto
	if au == nil || au.Wreck {
		return nil
	}
	return []string{fmt.Sprintf("{{C|%s}} fuel %s heat %s", au.Name, gdBar(au.Fuel, au.cap(), 5), gdBar(int(au.Heat), 100, 5))}
}

// ---- smiths sell fuel; the steam press ----

func gdSteamTalk(g *Game, a *Actor) []Option {
	if !gdSteamy(g) || (a.Role != "smith" && a.Role != "machinist" && a.Role != "merchant") {
		return nil
	}
	return []Option{
		{"{{O|Buy fuel.}}", func() { gdFuelShop(g, a) }},
		{"{{O|Use the workshop.}}", func() { g.gdWorkOpen() }},
	}
}

func gdFuelShop(g *Game, a *Actor) {
	g.Popup = Wrap("\"Fuel for the boiler. By the sack.\"", 100)
	g.Opts = nil
	for _, k := range gdFuelOrder {
		k := k
		pr := gdFuelDefs[k].Price * 3
		g.Opts = append(g.Opts, Option{fmt.Sprintf("3 %s {{K|(%s)}}", g.gdFuelName(k), g.gdPrice(pr)), func() {
			if g.gdPay(pr) {
				g.gdGive(g.gdFuelItem(k, 3))
				gdFuelShop(g, a)
			}
		}})
	}
	g.Opts = append(g.Opts, Option{"Enough.", func() { g.talkMenu(g.greeting(a)) }})
	g.Sel = 0
}

func gdPressOpen(g *Game, d *Item) {
	key := gdDevKey(g.Cur, d)
	g.gdOpen("gdx", "Steam press", func(u *gdUI) {
		st := g.gd()
		warm := st.Press[key] > g.Turn
		if warm {
			u.line("{{O|The press is hot for another " + gdHours(st.Press[key]-g.Turn) + ".}} Workshop assembly here is faster and better.")
		} else {
			u.line("The press is cold.")
		}
		cheap := func(k string) bool { return g.gdCount("gdfuel", k) > 0 }
		if cheap("coal") {
			u.row("Stoke the press with 1 "+g.gdFuelName("coal")+" (hot for 3 hours).", func() {
				g.gdTake("gdfuel", "coal", 1)
				st.Press[key] = g.Turn + 180
				g.spendTurns(3)
			})
		} else {
			u.off("Stoke the press (needs " + g.gdFuelName("coal") + ")")
		}
		cs := g.craftSt()
		for t := 1; t <= 7; t++ {
			t := t
			if cs.Bits[bitKey(t)] < 3 {
				continue
			}
			if !warm {
				u.off(fmt.Sprintf("Refine 3%s into 1%s (stoke the press first)", bitLabel(t), bitLabel(t+1)))
				continue
			}
			u.row(fmt.Sprintf("Refine 3 %s bits into 1 %s bit.", bitLabel(t), bitLabel(t+1)), func() {
				cs.Bits[bitKey(t)] -= 3
				cs.Bits[bitKey(t+1)]++
				g.spendTurns(4)
				g.gdFirst("press", "refined scrap into finer stock on a steam press.")
			})
		}
		if g.gdCount("gdfuel", "coal") >= 4 {
			u.row("Compress 4 "+g.gdFuelName("coal")+" into 1 "+g.gdFuelName("cell")+".", func() {
				g.gdTake("gdfuel", "coal", 4)
				g.gdGive(g.gdFuelItem("cell", 1))
				g.spendTurns(5)
			})
		}
	})
}

var _ = strings.ToLower
