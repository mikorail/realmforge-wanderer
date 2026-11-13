package engine

import (
	"fmt"
	"math"
	"sort"
	"strings"
)

// Journeys (prefix jy): sky craft (balloons, airships, griffons, kites, jet-packs, drones, cloud-boats)
// bought or hired from riggers and flown across the world map; personal cliff gliders; passage with
// caravans and river/lake ferries with day-by-day events; and weather route puzzles (floods, passes
// closed by blizzards, fires, fog, sandstorms) with workarounds. State g.ext("jy"), planet-local.
// Nothing here draws g.R: events come from side rngs of seed + place + turn.

type jyRide struct {
	Kind  string  `json:"k"` // caravan ferry lake
	From  string  `json:"fr"`
	To    int     `json:"to"` // burg id
	Name  string  `json:"n"`
	Km    float64 `json:"km"`
	Days  int     `json:"d"`
	Day   int     `json:"dy"`
	Pace  float64 `json:"pc"`
	Via   []int   `json:"via"`
	Fare  int     `json:"fare"`
	Start int     `json:"st"`
	Quiet int     `json:"q"`
	Guard int     `json:"g"`
	Delay int     `json:"dl"`
}

type jyState struct {
	Craft   string          `json:"c,omitempty"`
	Parked  bool            `json:"pk,omitempty"`
	At      int             `json:"at"`
	From    int             `json:"fr"`
	Air     bool            `json:"air,omitempty"`
	Fuel    float64         `json:"fu,omitempty"`
	Hull    int             `json:"hu,omitempty"`
	Rent    bool            `json:"rent,omitempty"`
	Until   int             `json:"until,omitempty"`
	Glider  bool            `json:"gl,omitempty"`
	SkipAt  int             `json:"sk,omitempty"` // parasang+1 where the prompt was declined
	Cleared map[int]int     `json:"cl,omitempty"` // parasang → window+1 in which a route puzzle is solved
	Flights int             `json:"fl,omitempty"`
	Glides  int             `json:"gd,omitempty"`
	Rides   int             `json:"rd,omitempty"`
	Km      float64         `json:"km,omitempty"`
	Deeds   map[string]bool `json:"dd,omitempty"`
	Ride    *jyRide         `json:"ride,omitempty"`
	Title   string          `json:"t,omitempty"`
}

func (g *Game) jy() *jyState {
	s := g.ext("jy", func() any { return &jyState{} }).(*jyState)
	if s.Deeds == nil {
		s.Deeds = map[string]bool{}
	}
	if s.Cleared == nil {
		s.Cleared = map[int]int{}
	}
	return s
}

var (
	jyEscFn func()
	jyNav   bool
	jyBlkDx int
	jyBlkDy int
)

func (g *Game) jyDeed(key, text string) {
	st := g.jy()
	if st.Deeds[key] {
		return
	}
	st.Deeds[key] = true
	g.Msg("{{Y|" + text + "}}")
	if g.Cur != nil {
		g.deed("journey", text, 0, 0)
	}
}

// ---- craft helpers ----

func (g *Game) jyGK() string {
	if g.Genre == "" {
		return "fantasy"
	}
	return g.Genre
}

func (g *Game) jySkin() string {
	if jyGenres[g.Genre] {
		return g.Genre
	}
	return ""
}

func jyPick(m map[string]string, skin string) string {
	if s, ok := m[skin]; ok {
		return s
	}
	return m[""]
}

func (d *jyDef) name(g *Game) string { return jyPick(d.Names, g.jySkin()) }
func (d *jyDef) fuel(g *Game) string { return jyPick(d.Fuel, g.jySkin()) }

func (g *Game) jyDefByID(id string) *jyDef {
	for i := range jyDefs {
		if jyDefs[i].ID == id {
			return &jyDefs[i]
		}
	}
	return nil
}

func (g *Game) jyAvail() []*jyDef {
	var out []*jyDef
	gk := g.jyGK()
	for i := range jyDefs {
		d := &jyDefs[i]
		if d.Gen == "*" || strings.Contains(" "+d.Gen+" ", " "+gk+" ") {
			out = append(out, d)
		}
	}
	return out
}

func (st *jyState) def(g *Game) *jyDef {
	if st.Craft == "" {
		return nil
	}
	return g.jyDefByID(st.Craft)
}

func (g *Game) jyPrice(n int) int { return maxi(1, n*(100-g.Skills["persuasion"]*8)/100) }

func (g *Game) jySpeed(d *jyDef, np, dx, dy int) float64 {
	dir, str := g.shWind(np)
	cos := math.Cos(math.Atan2(float64(dy), float64(dx)) - dir)
	m := 1 + d.Wind*(0.2+0.6*str)*cos
	if d.Wind >= 0.6 && str < 0.15 {
		m = 0.45
	}
	m = math.Max(0.4, math.Min(1.8, m))
	v := d.Kmd * m
	if g.shCompanions()+1 > d.Cap {
		v *= 0.75
	}
	if g.jy().Hull*10 < d.Hull*3 {
		v *= 0.85
	}
	if shStormy(g.shWeather(np)) {
		v *= 0.6
	}
	return v
}

func jyDirName(dx, dy int) string {
	n := map[[2]int]string{{0, -1}: "north", {0, 1}: "south", {1, 0}: "east", {-1, 0}: "west", {1, -1}: "north-east", {-1, -1}: "north-west", {1, 1}: "south-east", {-1, 1}: "south-west"}
	return n[[2]int{dx, dy}]
}

// ---- menus ----

func (g *Game) jyMenu(title string, lines []string, opts []Option, esc func()) {
	st := g.jy()
	st.Title = title
	g.Talk = nil
	g.Popup = nil
	for _, l := range lines {
		g.Popup = append(g.Popup, Wrap(l, 100)...)
	}
	g.Opts = opts
	g.Sel = 0
	jyEscFn = esc
	jyNav = false
	g.Mode = "jy"
}

func (g *Game) jyClose() {
	jyEscFn, jyNav = nil, false
	g.Talk = nil
	if g.OnWorld {
		g.Mode = "world"
	} else {
		g.Mode = "main"
	}
}

func jyKeys(g *Game, k string) {
	if jyNav {
		if dx, dy, ok := arrowOf(k); ok {
			jyNav = false
			g.Mode = "world"
			g.WorldMove(dx, dy)
			return
		}
	}
	g.keyMenu(k, func() {
		if f := jyEscFn; f != nil {
			jyEscFn = nil
			f()
			return
		}
		if g.jy().Ride != nil {
			g.jyRideNext()
			return
		}
		g.jyClose()
	})
}

func jyDraw(g *Game) {
	s := g.sc
	if g.OnWorld {
		pop, info := g.Popup, g.wmInfo
		g.drawWorld()
		g.Popup, g.wmInfo = pop, info
	}
	h := len(g.Popup) + len(g.Opts) + 5
	y0 := maxi(1, (MH-h)/2)
	s.Box(2, y0, SW-4, h, CDarkCyan, g.jy().Title)
	y := y0 + 1
	for _, l := range g.Popup {
		s.TextB(5, y, l, CGray, CBgDark, SW-3)
		y++
	}
	y++
	for i, o := range g.Opts {
		bg := CBgDark
		if i == g.Sel {
			bg = CBgSel
		}
		s.TextB(5, y, fmt.Sprintf("%c) %s", 'a'+i, o.Text), CWhite, bg, SW-3)
		y++
	}
}

func jyOpt(t string, f func()) Option { return Option{t, f} }

// ---- hooks into the world map ----

// jyWorldMove is the one-line hook at the top of WorldMove. It returns true when it took the step.
func jyWorldMove(g *Game, dx, dy int) bool {
	if g.ijNoFastTravel() { // a vow against fast travel (challenges.go)
		return false
	}
	st := g.jy()
	w := g.W
	cur := w.Idx(g.WX, g.WY)
	if st.SkipAt != 0 && st.SkipAt != cur+1 {
		st.SkipAt = 0
	}
	nx, ny := g.WX+dx, g.WY+dy
	if st.Air {
		if !w.In(nx, ny) {
			return true
		}
		g.jyFlyStep(dx, dy, w.Idx(nx, ny))
		return true
	}
	if !w.In(nx, ny) || st.SkipAt == cur+1 {
		return false
	}
	np := w.Idx(nx, ny)
	if st.Craft != "" && st.Parked && st.At == cur {
		g.jyTakeoffMenu(dx, dy, cur)
		return true
	}
	if st.Glider && w.Biome[cur] != 0 && w.Biome[np] != 0 && w.Height[cur] >= 60 && w.Height[cur]-w.Height[np] >= 10 {
		g.jyGlideMenu(dx, dy, cur)
		return true
	}
	return false
}

func (g *Game) jyTakeoffMenu(dx, dy, cur int) {
	st := g.jy()
	d := st.def(g)
	if d == nil {
		st.Craft = ""
		return
	}
	dir, str := g.shWind(cur)
	lines := []string{fmt.Sprintf("{{C|Your %s}} waits here. Hull %d/%d.", d.name(g), st.Hull, d.Hull)}
	if d.FuelMax > 0 {
		lines = append(lines, fmt.Sprintf("%s: %.1f of %.0f days.", titleCase(d.fuel(g)), st.Fuel, d.FuelMax))
	}
	lines = append(lines, "Wind: "+shWindName(dir, str)+". "+jyAdvice(d, dir, str, dx, dy))
	opts := []Option{
		jyOpt("{{B|Take off "+jyDirName(dx, dy)+".}}", func() {
			switch {
			case shStormy(g.shWeather(cur)):
				g.Msg("{{y|The storm is too wild to leave the ground.}}")
				g.jyClose()
			case d.FuelMax > 0 && st.Fuel < 0.3:
				g.Msg("{{y|There is no " + d.fuel(g) + " to speak of. A rigger can refill you.}}")
				g.jyClose()
			default:
				st.Parked, st.Air, st.From = false, true, cur
				st.Flights++
				g.jyDeed("fly", "You left the ground in a "+d.name(g)+" for the first time.")
				g.Mode = "world"
				g.WorldMove(dx, dy)
			}
		}),
		jyOpt("Walk instead.", func() { st.SkipAt = cur + 1; g.jyClose(); g.WorldMove(dx, dy) }),
	}
	g.jyMenu("Take off", lines, opts, g.jyClose)
}

func jyAdvice(d *jyDef, dir, str float64, dx, dy int) string {
	cos := math.Cos(math.Atan2(float64(dy), float64(dx)) - dir)
	switch {
	case d.Wind < 0.3 || str < 0.18:
		return "Wind matters little."
	case cos > 0.4:
		return "A tailwind on this heading."
	case cos < -0.4:
		return "A headwind on this heading."
	}
	return "A crosswind on this heading."
}

// ---- flying ----

func (g *Game) jyFlyStep(dx, dy, np int) {
	st := g.jy()
	w := g.W
	d := st.def(g)
	if d == nil {
		st.Air = false
		return
	}
	km := g.tvKm()
	if dx != 0 && dy != 0 {
		km *= math.Sqrt2
	}
	v := g.jySpeed(d, np, dx, dy)
	cost := maxi(60, int(km/v*DayLen))
	days := float64(cost) / DayLen
	if d.FuelMax > 0 && st.Fuel < days*0.999 {
		g.Msg(fmt.Sprintf("{{y|Not enough %s for that leg (%.1f days left, %.1f needed). Land, or fly a shorter way.}}", d.fuel(g), st.Fuel, days))
		g.jyNavMenu()
		return
	}
	if d.FuelMax > 0 {
		st.Fuel = math.Max(0, st.Fuel-days)
	}
	ate, drank, _ := g.tvPass(cost)
	g.WX, g.WY = np%w.W, np/w.W
	g.CX, g.CY = g.WX, g.WY
	g.Known[np] = true
	for _, dd := range dirs8 {
		if w.In(g.WX+dd[0], g.WY+dd[1]) {
			if q := w.Idx(g.WX+dd[0], g.WY+dd[1]); w.Burg[q] > 0 {
				g.Known[q] = true
			}
		}
	}
	st.Km += km
	msg := fmt.Sprintf("You fly %s over %s: %.0f km in %s.", jyDirName(dx, dy), w.RegionName(np), km, tvDuration(cost))
	if ate > 0 || drank > 0 {
		msg += fmt.Sprintf(" {{K|(%d meals, %d drams)}}", ate, drank)
	}
	g.Msg(msg)
	wsOnMove(g)
	if st.Km > 2500 {
		g.jyDeed("far", "You flew more than 2,500 km above the world.")
	}
	g.jyAirEvents(np, d, g.jyNavMenu)
}

func (g *Game) jyHull(n int, why string) bool {
	st := g.jy()
	st.Hull -= n
	if st.Hull > 0 {
		g.Msg(fmt.Sprintf("{{R|%s}} Hull %d.", why, st.Hull))
		return false
	}
	g.jyCrash(why)
	return true
}

func (g *Game) jyCrash(why string) {
	st := g.jy()
	d := st.def(g)
	nm := "craft"
	if d != nil {
		nm = d.name(g)
	}
	p := g.W.Idx(g.WX, g.WY)
	r := NewRng(fmt.Sprintf("%s:jy:crash:%d:%d", g.Seed, p, g.Turn))
	g.Msg("{{R|" + why + " The " + nm + " comes apart and you fall!}}")
	st.Craft, st.Air, st.Parked = "", false, false
	if g.W.Biome[p] == 0 {
		g.jyDitch(r)
		return
	}
	g.hurt(g.P, r.Int(4, 11)+g.P.Level/3, nil, "fall")
	if g.P.HP <= 0 {
		return
	}
	g.jyDeed("crash", "You survived the crash of a flying machine.")
	g.jyArrive(p)
}

// jyDitch washes you ashore after a loss over water.
func (g *Game) jyDitch(r *Rng) {
	w := g.W
	st := g.jy()
	p := w.Idx(g.WX, g.WY)
	g.hurt(g.P, r.Int(3, 8), nil, "drown")
	if g.P.HP <= 0 {
		return
	}
	for rad := 1; rad <= 8; rad++ {
		for yy := -rad; yy <= rad; yy++ {
			for xx := -rad; xx <= rad; xx++ {
				if abs(xx) != rad && abs(yy) != rad {
					continue
				}
				if !w.In(g.WX+xx, g.WY+yy) {
					continue
				}
				q := w.Idx(g.WX+xx, g.WY+yy)
				if w.Biome[q] != 0 && w.Biome[q] != 16 {
					g.Msg("{{c|You are swept ashore after hours in the water.}}")
					g.tvPass(240)
					g.jyArrive(q)
					return
				}
			}
		}
	}
	g.jyArrive(st.From)
	_ = p
}

// jyArrive enters the zone at parasang p (a town zone when it holds a surface town).
func (g *Game) jyArrive(p int) {
	w := g.W
	st := g.jy()
	st.Air = false
	g.Known[p] = true
	g.Talk = nil
	if bid := w.Burg[p]; bid > 0 {
		if bd := w.burgByID[bid]; bd != nil && bd.Alive && bd.Layer == "surface" {
			wx, wy, z := unTownZone(w, bd)
			g.WX, g.WY, g.CX, g.CY = wx, wy, wx, wy
			g.enterZone(g.GetZone(wx, wy, 1, 1, z), 40, 13, true)
			g.Mode = "main"
			g.recompute()
			return
		}
	}
	wx, wy := p%w.W, p/w.W
	g.WX, g.WY, g.CX, g.CY = wx, wy, wx, wy
	g.enterZone(g.GetZone(wx, wy, 1, 1, 0), 40, 12, true)
	g.Mode = "main"
	g.recompute()
}

func (g *Game) jyNavMenu() {
	st := g.jy()
	d := st.def(g)
	if d == nil {
		g.jyClose()
		return
	}
	w := g.W
	p := w.Idx(g.WX, g.WY)
	dir, str := g.shWind(p)
	lines := []string{fmt.Sprintf("{{C|%s}} over %s. Hull %d/%d.", titleCase(d.name(g)), w.RegionName(p), st.Hull, d.Hull)}
	if d.FuelMax > 0 {
		lines = append(lines, fmt.Sprintf("%s: %.1f days.", titleCase(d.fuel(g)), st.Fuel))
	}
	lines = append(lines, "Wind: "+shWindName(dir, str)+". Press a direction key to keep flying.")
	note := "flat ground"
	switch {
	case w.Biome[p] == 0 && !d.Float:
		note = "open water: no landing"
	case w.Biome[p] == 0:
		note = "you can only rest on the water"
	case w.Biome[p] == 16:
		note = "lava flats: no landing"
	case w.Height[p] >= 70:
		note = "steep ground: a rough landing"
	case w.Burg[p] > 0:
		note = "a town below"
	}
	opts := []Option{
		jyOpt("{{B|Land here}} {{K|("+note+")}}", func() { g.jyLand(false) }),
		jyOpt("Keep flying.", func() { g.Mode = "world" }),
	}
	g.jyMenu("Aloft", lines, opts, func() { g.Mode = "world" })
	jyNav = true
}

func (g *Game) jyLand(forced bool) {
	st := g.jy()
	w := g.W
	d := st.def(g)
	if d == nil {
		g.jyClose()
		return
	}
	p := w.Idx(g.WX, g.WY)
	r := NewRng(fmt.Sprintf("%s:jy:land:%d:%d", g.Seed, p, g.Turn))
	if w.Biome[p] == 0 || w.Biome[p] == 16 {
		if forced {
			st.Craft, st.Air = "", false
			g.Msg("{{R|You come down hard with nowhere to land.}}")
			g.jyDitch(r)
			return
		}
		g.Msg("{{y|There is nowhere to set down here.}}")
		g.jyNavMenu()
		return
	}
	if w.Height[p] >= 70 && r.Chance(0.3) {
		if g.jyHull(r.Int(8, 22), "You come down on a slope and the skids bite rock.") {
			return
		}
		g.hurt(g.P, r.Int(1, 5), nil, "fall")
		if g.P.HP <= 0 {
			return
		}
	}
	st.Air, st.Parked, st.At = false, true, p
	g.Msg(fmt.Sprintf("You set the %s down near %s.", d.name(g), w.RegionName(p)))
	if st.Rent && g.Turn > st.Until {
		g.Msg("{{y|The hire has run out: the rigger's agents, tracking the craft, reclaim it before you can object.}}")
		st.Craft, st.Parked, st.Rent = "", false, false
	}
	g.jyDeed("land", "You landed a flying machine in the wilds.")
	g.jyArrive(p)
}

// jyAirEvents rolls at most one event for the leg and calls next when flying goes on.
func (g *Game) jyAirEvents(np int, d *jyDef, next func()) {
	st := g.jy()
	w := g.W
	r := NewRng(fmt.Sprintf("%s:jy:air:%d:%d", g.Seed, np, g.Turn/30))
	nx, ny := np%w.W, np/w.W
	lvl := g.P.Level
	if _, enemy, ok := g.frontierAt(nx, ny); ok && r.Chance(0.4) {
		g.jyDeed("front", "You flew over a war front.")
		g.jyMenu("Anti-air fire", []string{"{{R|Anti-air batteries of " + g.realmName(enemy) + " open up}} as you cross the lines."}, []Option{
			jyOpt("Dive into the cloud and run.", func() {
				if g.P.Score(AGI)+lvl+r.Int(1, 20) >= 24 {
					g.Msg("You slip through the flak untouched.")
					next()
				} else if !g.jyHull(r.Int(8, 20), "Shrapnel tears the craft.") {
					next()
				}
			}),
			jyOpt("Fly the white flag and pay the toll.", func() {
				if g.cuSpend(g.jyPrice(25 + lvl*3)) {
					g.Msg("The gunners wave you through, grumbling.")
					next()
				} else if !g.jyHull(r.Int(8, 20), "You cannot pay; they fire anyway.") {
					next()
				}
			}),
		}, next)
		return
	}
	if shStormy(g.shWeather(np)) && r.Chance(0.35+d.Risk*3) {
		g.jyMenu("Storm", []string{"{{B|A storm closes around you}}, thick with cloud and lightning."}, []Option{
			jyOpt("Climb above it.", func() {
				if d.FuelMax > 0 {
					st.Fuel = math.Max(0, st.Fuel-0.3)
				}
				if d.FuelMax == 0 && r.Chance(0.3) {
					if g.jyHull(r.Int(5, 14), "The climb is rough.") {
						return
					}
				} else {
					g.Msg("You climb into cold clear air, the storm boiling below.")
				}
				next()
			}),
			jyOpt("Ride it out.", func() {
				if r.Chance(0.7) {
					if g.jyHull(r.Int(6, 20), "The storm batters you.") {
						return
					}
				} else {
					g.Msg("Luck holds; the storm tosses you and lets go.")
				}
				next()
			}),
			jyOpt("Land at once.", func() { g.jyLand(false) }),
		}, next)
		return
	}
	lawless := w.State[np] == 0
	pc := 0.04
	if lawless {
		pc += 0.07
	}
	if g.Genre == "wasteland" {
		pc += 0.05
	}
	if r.Chance(pc) {
		who := "Sky pirates"
		g.jyMenu("Raiders aloft", []string{"{{R|" + who + "}} sweep out of the clouds, hooks and guns ready."}, []Option{
			jyOpt("Fight them off.", func() {
				if g.P.Score(AGI)+lvl+r.Int(1, 20) >= 26 {
					g.Msg("You drive them off.")
					g.gainXP(12 + lvl*2)
					g.cuEarn(8 + lvl*2)
					g.jyDeed("pirates", "You beat off sky pirates.")
					next()
				} else {
					g.hurt(g.P, r.Int(2, 8), nil, "blade")
					if g.P.HP > 0 && !g.jyHull(r.Int(6, 16), "They rake the craft before breaking off.") {
						next()
					}
				}
			}),
			jyOpt("Outrun them.", func() {
				if r.Chance(math.Max(0.2, math.Min(0.85, d.Kmd/220))) {
					g.Msg("You pull away and they fall behind.")
					next()
				} else if !g.jyHull(r.Int(6, 16), "They keep pace and fire.") {
					next()
				}
			}),
			jyOpt("Pay them off.", func() {
				if g.cuSpend(g.jyPrice(12 + lvl*5)) {
					g.Msg("They take the purse and veer away.")
					next()
				} else {
					g.Msg("{{y|You cannot pay.}}")
					if !g.jyHull(r.Int(6, 16), "They open fire.") {
						next()
					}
				}
			}),
		}, next)
		return
	}
	switch {
	case w.Height[np] >= 70 && r.Chance(0.06):
		if g.jyHull(r.Int(3, 12), "A falling rock from the cliffs clips you.") {
			return
		}
	case w.Biome[np] != 0 && r.Chance(0.06):
		if g.jyHull(r.Int(2, 7), "A flock of birds smashes into the craft.") {
			return
		}
	case r.Chance(d.Risk * 0.5):
		if g.jyHull(r.Int(3, 10), "Something shudders and fails in the rigging.") {
			return
		}
	case r.Chance(0.2):
		g.Msg(Pick(r, []string{"{{C|Clouds part below you and the land lies spread like a map.}}", "{{C|A river shines far below; a line of wagons crawls along beside it.}}", "{{C|You pass a skein of geese, wing to wing.}}"}))
	}
	next()
}

// ---- cliff gliders ----

func (g *Game) jyCliffBump(nx, ny int) bool {
	st := g.jy()
	if !st.Glider || g.Cur == nil || g.Cur.Z != 0 || g.Cur.At(nx, ny) != TCliff {
		return false
	}
	dx, dy := sign(nx-g.P.X), sign(ny-g.P.Y)
	if dx == 0 && dy == 0 {
		return false
	}
	nm := jyPick(jyGliderNames, g.jySkin())
	cur := g.W.Idx(g.WX, g.WY)
	dir, str := g.shWind(cur)
	lines := []string{
		"You stand at the lip of the cliff face with your " + nm + " rigged.",
		"Wind: " + shWindName(dir, str) + ".",
	}
	opts := []Option{
		jyOpt("{{B|Leap from the cliff and glide across the zone.}}", func() {
			g.jyClose()
			g.jyZoneGlide(dx, dy)
		}),
	}
	if g.W.In(g.WX+dx, g.WY+dy) {
		np := g.W.Idx(g.WX+dx, g.WY+dy)
		if g.W.Biome[cur] != 0 && g.W.Biome[np] != 0 && (g.W.Height[cur] >= 60 || g.W.Height[cur]-g.W.Height[np] >= 8) {
			opts = append(opts, jyOpt("{{C|Catch the high updraft and glide overland "+jyDirName(dx, dy)+".}}", func() {
				g.jyGlide(dx, dy, cur)
			}))
		}
	}
	opts = append(opts, jyOpt("Step back from the edge.", g.jyClose))
	g.jyMenu("Cliff launch", lines, opts, g.jyClose)
	return true
}

func (g *Game) jyZoneGlide(dx, dy int) bool {
	st := g.jy()
	if !st.Glider || g.Cur == nil {
		return false
	}
	z := g.Cur
	lx, ly := -1, -1
	for step := 2; step <= 14; step++ {
		tx, ty := g.P.X+dx*step, g.P.Y+dy*step
		if !z.In(tx, ty) {
			break
		}
		if z.Walkable(tx, ty) && z.ActorAt(tx, ty) == nil && z.At(tx, ty) != TCliff {
			lx, ly = tx, ty
		}
	}
	if lx < 0 {
		g.Msg("There is no clear landing beyond the cliff.")
		return false
	}
	g.P.X, g.P.Y = lx, ly
	st.Glides++
	nm := jyPick(jyGliderNames, g.jySkin())
	g.Msg(fmt.Sprintf("{{C|You leap from the cliff and soar across on your %s!}}", nm))
	g.jyDeed("glide", "You launched yourself from a cliff and glided down on cloth and wind.")
	g.recompute()
	g.spend(1000)
	return true
}

func (g *Game) jyGlideMenu(dx, dy, cur int) {
	st := g.jy()
	nm := jyPick(jyGliderNames, g.jySkin())
	dir, str := g.shWind(cur)
	lines := []string{"You stand on the edge of the high ground, the land falling away " + jyDirName(dx, dy) + "ward. Your " + nm + " is on your back.", "Wind: " + shWindName(dir, str) + "."}
	g.jyMenu("Cliff launch", lines, []Option{
		jyOpt("{{B|Launch and glide.}}", func() {
			if shStormy(g.shWeather(cur)) || str > 0.9 {
				g.Msg("{{y|The wind is too wild to launch.}}")
				g.jyClose()
				return
			}
			g.jyGlide(dx, dy, cur)
		}),
		jyOpt("Climb down on foot.", func() { st.SkipAt = cur + 1; g.jyClose(); g.WorldMove(dx, dy) }),
	}, g.jyClose)
}

func (g *Game) jyGlide(dx, dy, cur int) {
	st := g.jy()
	w := g.W
	r := NewRng(fmt.Sprintf("%s:jy:glide:%d:%d", g.Seed, cur, g.Turn))
	alt := maxi(2, mini(5, 1+(w.Height[cur]-50)/12))
	dir, str := g.shWind(cur)
	if math.Cos(math.Atan2(float64(dy), float64(dx))-dir) > 0.5 && str > 0.4 {
		alt++
	}
	steps := 0
	km := g.tvKm()
	if dx != 0 && dy != 0 {
		km *= math.Sqrt2
	}
	p := cur
	g.Mode = "world"
	for alt > 0 {
		nx, ny := p%w.W+dx, p/w.W+dy
		if !w.In(nx, ny) {
			break
		}
		np := w.Idx(nx, ny)
		if w.Biome[np] == 0 || w.Height[np] > w.Height[p]+2 {
			break
		}
		if w.Height[p]-w.Height[np] < 3 {
			alt--
		}
		alt--
		g.tvPass(maxi(120, int(km/240*DayLen)))
		p = np
		g.WX, g.WY, g.CX, g.CY = nx, ny, nx, ny
		g.Known[np] = true
		steps++
		if str > 0.75 && r.Chance(0.3) {
			g.Msg("{{R|A violent gust slams the " + jyPick(jyGliderNames, g.jySkin()) + " into a spin!}}")
			g.hurt(g.P, r.Int(2, 6), nil, "fall")
			if g.P.HP <= 0 {
				return
			}
			break
		}
	}
	if steps == 0 {
		g.Msg("The ground does not fall away far enough to glide.")
		g.jyClose()
		return
	}
	st.Glides++
	g.Msg(fmt.Sprintf("You glide %s for %d parasangs (%.0f km) down to %s.", jyDirName(dx, dy), steps, float64(steps)*km, w.RegionName(p)))
	g.jyDeed("glide", "You launched yourself from a cliff and glided down on cloth and wind.")
	if steps >= 4 {
		g.jyDeed("glide4", "You glided more than 600 km from a single cliff.")
	}
	wsOnMove(g)
	if w.Height[p] >= 70 && r.Chance(0.25) {
		g.hurt(g.P, r.Int(1, 5), nil, "fall")
		if g.P.HP <= 0 {
			return
		}
		g.Msg("You land badly on the rocks.")
	}
	g.jyArrive(p)
}

// ---- riggers ----

func (g *Game) jyRiggerTalk(a *Actor) []Option {
	bd := g.trHere()
	if bd == nil {
		return nil
	}
	return []Option{{"{{B|I want to talk of flight.}}", func() { g.Talk = a; g.jyRigMenu(a, bd) }}}
}

func (g *Game) jyRigMenu(a *Actor, bd *BurgData) {
	st := g.jy()
	g.Talk = a
	g.Mode = "talk"
	dir, str := g.shWind(bd.P)
	g.Popup = Wrap(fmt.Sprintf("\"Wind is %s today. What do you need?\"", shWindName(dir, str)), 100)
	g.Opts = nil
	again := func() { g.jyRigMenu(a, bd) }
	d := st.def(g)
	if d == nil {
		for _, c := range g.jyAvail() {
			c := c
			price := g.jyPrice(c.Price)
			g.Opts = append(g.Opts, Option{fmt.Sprintf("{{B|Buy a %s}} {{K|(%.0f km/day, %d seat, %s)}} — %s", c.name(g), c.Kmd, c.Cap, c.Blurb, g.cuFormat(price)), func() {
				if !g.cuSpend(price) {
					g.talkMenu("\"Come back with the money.\"")
					return
				}
				st.Craft, st.Parked, st.At, st.Hull, st.Rent = c.ID, true, bd.P, c.Hull, false
				st.Fuel = c.FuelMax / 2
				g.Msg("The " + c.name(g) + " is yours, moored in " + bd.Name + ".")
				g.jyDeed("own", "You bought a flying machine.")
				again()
			}})
			rent := g.jyPrice(c.Rent)
			g.Opts = append(g.Opts, Option{fmt.Sprintf("Hire a %s for six days — %s", c.name(g), g.cuFormat(rent)), func() {
				if !g.cuSpend(rent) {
					g.talkMenu("\"The deposit first.\"")
					return
				}
				st.Craft, st.Parked, st.At, st.Hull, st.Rent, st.Until = c.ID, true, bd.P, c.Hull, true, g.Turn+6*DayLen
				st.Fuel = c.FuelMax
				g.Msg("You sign the hire: the " + c.name(g) + " is moored here, fuelled. After six days they will reclaim it wherever it lands.")
				again()
			}})
		}
	} else if st.Parked && st.At == bd.P {
		if need := d.FuelMax - st.Fuel; need > 0.05 {
			cost := g.jyPrice(int(math.Ceil(need * float64(maxi(1, d.Price/45)))))
			g.Opts = append(g.Opts, Option{fmt.Sprintf("Refill the %s ({{K|%.1f days}}) — %s", d.fuel(g), need, g.cuFormat(cost)), func() {
				if g.cuSpend(cost) {
					st.Fuel = d.FuelMax
					g.Msg("Tanks and burners are topped up.")
				}
				again()
			}})
		}
		if dmg := d.Hull - st.Hull; dmg > 0 {
			cost := g.jyPrice(dmg * maxi(1, d.Price/200))
			g.Opts = append(g.Opts, Option{fmt.Sprintf("Repair the hull ({{K|%d points}}) — %s", dmg, g.cuFormat(cost)), func() {
				if g.cuSpend(cost) {
					st.Hull = d.Hull
					g.Msg("The rigger patches and re-seams her.")
				}
				again()
			}})
		}
		if !st.Rent {
			g.Opts = append(g.Opts, Option{"Sell the " + d.name(g) + " — " + g.cuFormat(d.Price/2), func() {
				g.cuEarn(d.Price / 2)
				st.Craft, st.Parked = "", false
				again()
			}})
		}
	} else {
		g.Opts = append(g.Opts, Option{"Where is my machine?", func() {
			where := "it is aloft"
			if st.Parked {
				where = "you left it near " + g.W.RegionName(st.At)
			}
			g.talkMenu("\"Your " + d.name(g) + ": " + where + ".\"")
		}})
	}
	if !st.Glider {
		price := g.jyPrice(90)
		nm := jyPick(jyGliderNames, g.jySkin())
		g.Opts = append(g.Opts, Option{fmt.Sprintf("Buy a %s {{K|(launch from a cliff and glide down)}} — %s", nm, g.cuFormat(price)), func() {
			if g.cuSpend(price) {
				st.Glider = true
				g.Msg("You strap on the " + nm + ". On a high cliff, walk toward the drop on the world map to launch.")
			}
			again()
		}})
	}
	g.Opts = append(g.Opts, Option{"What of the roads and the weather?", func() {
		lines := g.jyNews()
		if len(lines) == 0 {
			lines = []string{"Nothing is closed that I have heard of."}
		}
		g.talkMenu("\"" + strings.Join(lines, " ") + "\"")
	}})
	g.Opts = append(g.Opts, Option{"Never mind.", func() { g.talkMenu(g.greeting(a)) }})
	g.Sel = 0
}

// ---- passenger rides ----

type jyTrip struct {
	To   *BurgData
	Kind string
	Name string
	Km   float64
	Days int
	Fare int
	Via  []int
	Down bool
	Pace float64
}

func (g *Game) jyCaravanTrips(bd *BurgData) []jyTrip {
	w := g.W
	n := w.nmData()
	if n == nil {
		return nil
	}
	seen := map[int]bool{bd.ID: true}
	var out []jyTrip
	for _, ri := range w.nmRoutes(bd.P, "road") {
		rt := &n.Routes[ri]
		for _, q := range rt.Parasangs {
			id := w.Burg[q]
			if id <= 0 || seen[id] {
				continue
			}
			to := w.burgByID[id]
			if to == nil || !to.Alive || to.Layer != "surface" {
				continue
			}
			via := trVia(rt.Parasangs, bd.P, q)
			if len(via) < 2 {
				continue
			}
			seen[id] = true
			km := float64(len(via)-1) * g.tvKm()
			out = append(out, jyTrip{To: to, Kind: "caravan", Name: rt.Name, Km: km, Days: maxi(1, int(math.Ceil(km/30))), Fare: 4 + int(km/16), Via: via, Pace: 30})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Km < out[j].Km })
	if len(out) > 6 {
		out = out[:6]
	}
	return out
}

func (g *Game) jyFerryTrips(bd *BurgData) []jyTrip {
	w := g.W
	var out []jyTrip
	if rv := w.nmRiver(bd.P, true); rv != nil {
		i0 := -1
		for i, q := range rv.Parasangs {
			if q == bd.P {
				i0 = i
				break
			}
		}
		if i0 < 0 {
			for i, q := range rv.Parasangs {
				if wsCheb(w, q, bd.P) <= 1 {
					i0 = i
					break
				}
			}
		}
		if i0 >= 0 {
			seen := map[int]bool{bd.ID: true}
			for j, q := range rv.Parasangs {
				id := w.Burg[q]
				if j == i0 || id <= 0 || seen[id] {
					continue
				}
				to := w.burgByID[id]
				if to == nil || !to.Alive || to.Layer != "surface" {
					continue
				}
				seen[id] = true
				var via []int
				if j > i0 {
					via = append(via, rv.Parasangs[i0:j+1]...)
				} else {
					for k := i0; k >= j; k-- {
						via = append(via, rv.Parasangs[k])
					}
				}
				down := j > i0
				pace := 35.0
				if down {
					pace = 70
				}
				km := float64(abs(j-i0)) * g.tvKm()
				out = append(out, jyTrip{To: to, Kind: "ferry", Name: rv.Name, Km: km, Days: maxi(1, int(math.Ceil(km/pace))), Fare: 3 + int(km/28), Via: via, Down: down, Pace: pace})
			}
		}
	}
	// lake crossings
	seenF := map[*NmFeature]bool{}
	seen := map[int]bool{bd.ID: true}
	check := func(p int) {
		for _, f := range w.nmFeatures(p) {
			if f.Kind != "lake" || seenF[f] {
				continue
			}
			seenF[f] = true
			for _, q := range f.Parasangs {
				cands := []int{q}
				for _, d := range dirs8 {
					if w.In(q%w.W+d[0], q/w.W+d[1]) {
						cands = append(cands, w.Idx(q%w.W+d[0], q/w.W+d[1]))
					}
				}
				for _, c := range cands {
					id := w.Burg[c]
					if id <= 0 || seen[id] {
						continue
					}
					to := w.burgByID[id]
					if to == nil || !to.Alive || to.Layer != "surface" {
						continue
					}
					seen[id] = true
					km := math.Max(g.tvKm()*0.6, math.Hypot(float64(to.P%w.W-bd.P%w.W), float64(to.P/w.W-bd.P/w.W))*g.tvKm()*1.15)
					out = append(out, jyTrip{To: to, Kind: "lake", Name: f.Name, Km: km, Days: maxi(1, int(math.Ceil(km/55))), Fare: 3 + int(km/30), Via: []int{bd.P, to.P}, Pace: 55})
				}
			}
		}
	}
	check(bd.P)
	for _, d := range dirs8 {
		if w.In(bd.P%w.W+d[0], bd.P/w.W+d[1]) {
			check(w.Idx(bd.P%w.W+d[0], bd.P/w.W+d[1]))
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Km < out[j].Km })
	if len(out) > 6 {
		out = out[:6]
	}
	return out
}

func (g *Game) jyTripMenu(a *Actor, trips []jyTrip, intro string) {
	g.Talk = a
	g.Mode = "talk"
	g.Popup = Wrap(intro, 100)
	g.Opts = nil
	for _, t := range trips {
		t := t
		fare := g.jyPrice(t.Fare)
		how := "caravan"
		if t.Kind != "caravan" {
			how = "ferry"
			if t.Kind == "ferry" {
				if t.Down {
					how = "ferry, downstream"
				} else {
					how = "ferry, upstream"
				}
			}
		}
		g.Opts = append(g.Opts, Option{fmt.Sprintf("{{B|Passage to %s}} {{K|(%s, %s, %.0f km, %s)}} — %s", t.To.Name, how, t.Name, t.Km, tvDuration(t.Days*DayLen), g.cuFormat(fare)), func() { g.jyRideStart(a, t, fare) }})
	}
	g.Opts = append(g.Opts, Option{"Never mind.", func() { g.talkMenu(g.greeting(a)) }})
	g.Sel = 0
}

func (g *Game) jyRideStart(a *Actor, t jyTrip, fare int) {
	if !g.cuSpend(fare) {
		g.Msg("{{y|\"No fare, no seat.\"}}")
		return
	}
	bd := g.trHere()
	from := ""
	if bd != nil {
		from = bd.Name
	}
	r := NewRng(fmt.Sprintf("%s:jy:guards:%d:%d", g.Seed, t.To.ID, g.Turn))
	st := g.jy()
	st.Ride = &jyRide{Kind: t.Kind, From: from, To: t.To.ID, Name: t.Name, Km: t.Km, Days: t.Days, Pace: t.Pace, Via: t.Via, Fare: fare, Start: g.Turn, Guard: r.Int(2, 5)}
	g.Msg(fmt.Sprintf("You pay %s and take a place with the %s for %s.", g.cuFormat(fare), map[string]string{"caravan": "caravan", "ferry": "ferry", "lake": "ferry"}[t.Kind], t.To.Name))
	g.Talk = nil
	g.jyRideNext()
}

func (g *Game) jyRideNext() {
	st := g.jy()
	r := st.Ride
	w := g.W
	if r == nil {
		g.jyClose()
		return
	}
	for r.Day < r.Days {
		r.Day++
		cost := DayLen
		if r.Day == r.Days {
			cost = maxi(240, minint(DayLen, int((r.Km/r.Pace-float64(r.Days-1))*DayLen)))
		}
		g.tvPass(cost)
		idx := mini(len(r.Via)-1, maxi(0, len(r.Via)*r.Day/r.Days-1))
		p := r.Via[idx]
		g.Known[p] = true
		// weather and hazards hold the group up
		if r.Delay < 2 {
			kind, why := g.jyBlockWhy(p)
			stormy := r.Kind != "caravan" && shStormy(g.shWeather(p))
			if kind == "fire" || (kind == "flood" && r.Kind == "caravan" && w.Road[p] == 0) || kind == "sand" || stormy {
				r.Delay++
				r.Days++
				g.tvPass(DayLen)
				if stormy {
					why = "a storm bars the water"
				}
				g.Msg("{{y|The master halts the party a day: " + why + ".}}")
			}
		}
		rng := NewRng(fmt.Sprintf("%s:jy:ride:%d:%d:%d", g.Seed, r.Start, r.To, r.Day))
		if r.Day == r.Days || !rng.Chance(0.5) {
			r.Quiet++
			continue
		}
		if r.Quiet > 0 {
			g.Msg(fmt.Sprintf("{{K|%d quiet day(s) pass on the %s.}}", r.Quiet, map[bool]string{true: "road", false: "water"}[r.Kind == "caravan"]))
			r.Quiet = 0
		}
		if g.jyRideEvent(r, rng, p) {
			return
		}
		if g.P.HP <= 0 {
			return
		}
	}
	g.jyRideEnd()
}

func minint(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// jyRideEvent runs one day's event; it returns true when it opened a menu (the ride resumes from it).
func (g *Game) jyRideEvent(r *jyRide, rng *Rng, p int) bool {
	st := g.jy()
	w := g.W
	lvl := g.P.Level
	cont := g.jyRideNext
	title := "On the road"
	if r.Kind != "caravan" {
		title = "On the water"
	}
	take := 3 + r.Fare/2
	roll := rng.Float()
	switch {
	case roll < 0.28: // talk round the fire / on deck
		line := Pick(rng, jyCampTalk)
		if r.Kind != "caravan" {
			line = Pick(rng, jyFerryTalk)
		}
		if len(w.Events) > 0 && rng.Chance(0.4) {
			e := w.Events[rng.Int(0, len(w.Events)-1)]
			text := e.Text
			if len(text) > 160 {
				text = text[:157] + "…"
			}
			line = fmt.Sprintf("a traveller recalls: \"In Y%d — %s\"", e.Y, text)
		}
		g.Msg("{{K|Near " + w.RegionName(p) + ", " + line + ".}}")
		return false
	case roll < 0.45: // theft
		who := Pick(rng, jyThiefName)
		g.jyMenu(title, []string{"In the night you wake to a hand at your purse: " + who + " is bending over you."}, []Option{
			jyOpt("Grab them.", func() {
				if g.P.Score(AGI)+lvl+rng.Int(1, 20) >= 22 {
					g.Msg("You catch the thief by the wrist. The guards haul them off.")
					g.gainXP(6 + lvl)
				} else if g.cuSpend(take) {
					g.Msg("They slip away with " + g.cuFormat(take) + ".")
				}
				cont()
			}),
			jyOpt("Let them go.", func() {
				if g.cuSpend(take) {
					g.Msg("It was only " + g.cuFormat(take) + ".")
				}
				cont()
			}),
		}, cont)
		return true
	case roll < 0.62: // ambush
		foe := "Bandits"
		if r.Kind != "caravan" {
			foe = "River pirates"
		}
		g.jyMenu(title, []string{"{{R|" + foe + " strike the " + map[bool]string{true: "caravan", false: "boat"}[r.Kind == "caravan"] + "!}} The " + fmt.Sprint(r.Guard) + " guards form up."}, []Option{
			jyOpt("{{B|Fight beside the guards.}}", func() {
				sc := maxi(g.P.Score(STR), g.P.Score(AGI)) + lvl + rng.Int(1, 20) + r.Guard*2
				if sc >= 31 {
					g.Msg("You and the guards cut them down. The master presses a few coins on you.")
					g.gainXP(15 + lvl*3)
					g.cuEarn(6 + lvl*2)
					g.jyDeed("guards", "You fought beside caravan guards and held the line.")
				} else {
					g.hurt(g.P, rng.Int(3, 10)+lvl/2, nil, "blade")
					if g.P.HP <= 0 {
						return
					}
					g.Msg("You are cut in the melee, but the guards carry the day.")
					g.gainXP(8 + lvl)
				}
				cont()
			}),
			jyOpt("Hide among the sacks.", func() {
				g.Msg("You hear the fight and wait it out. Some goods are lost.")
				cont()
			}),
			jyOpt("Pay them off.", func() {
				if g.cuSpend(take * 3) {
					g.Msg("They take the coin and melt away.")
				} else {
					g.Msg("{{y|You cannot pay; you have to fight.}}")
					g.hurt(g.P, rng.Int(3, 9), nil, "blade")
					if g.P.HP <= 0 {
						return
					}
				}
				cont()
			}),
		}, cont)
		return true
	case roll < 0.8: // sick traveller
		who := Pick(rng, jySick)
		g.jyMenu(title, []string{who + " is sick with fever and cannot sit up. The others look away."}, []Option{
			jyOpt("Share your water and food.", func() {
				g.Hunger += 120
				g.Thirst += 120
				g.gainXP(6 + lvl)
				g.Msg(titleCase(who) + " recovers by morning and presses a charm into your hand.")
				cont()
			}),
			jyOpt("Leave them to the wagons.", cont),
		}, cont)
		return true
	default:
		if r.Kind == "caravan" {
			g.jyMenu(title, []string{Pick(rng, jyRockfall)}, []Option{
				jyOpt("Help clear the stones.", func() {
					g.tvPass(180)
					if g.P.Score(STR)+rng.Int(1, 20) >= 22 {
						g.gainXP(6 + lvl)
						g.Msg("You heave a boulder aside and the wagons roll on.")
					} else {
						g.hurt(g.P, rng.Int(1, 4), nil, "crush")
					}
					cont()
				}),
				jyOpt("Wait with the others.", func() { g.tvPass(360); cont() }),
			}, cont)
			return true
		}
		g.Msg("{{y|The boat runs onto a sandbank; the crew pole her off after half a day.}}")
		g.tvPass(DayLen / 2)
	}
	_ = st
	return false
}

func (g *Game) jyRideEnd() {
	st := g.jy()
	r := st.Ride
	st.Ride = nil
	if r == nil {
		g.jyClose()
		return
	}
	to := g.W.burgByID[r.To]
	if to == nil {
		g.jyClose()
		return
	}
	st.Rides++
	g.Msg(fmt.Sprintf("You arrive at {{Y|%s}} after %s on the %s (%.0f km).", to.Name, tvDuration(r.Days*DayLen), r.Name, r.Km))
	if r.Kind == "caravan" {
		g.jyDeed("caravan", "You travelled with a merchant caravan.")
	} else {
		g.jyDeed("ferry", "You crossed country by ferry.")
	}
	for _, q := range r.Via {
		g.Known[q] = true
	}
	g.jyArrive(to.P)
}

// ---- zones ----

func jyZone(g *Game, zn *Zone, _ *Rng) {
	if zn.Kind != "town" || zn.ZX != 1 || zn.ZY != 1 || zn.Burg <= 0 {
		return
	}
	bd := g.W.burgByID[zn.Burg]
	if bd == nil || !bd.Alive {
		return
	}
	if _, _, z := unTownZone(g.W, bd); z != zn.Z || zn.Z != 0 || bd.Layer != "surface" {
		return
	}
	r := NewRng(fmt.Sprintf("%s:jy:place:%d", g.Seed, bd.ID))
	chance := 0.22
	if g.W.cityTier[bd.ID] >= 1 {
		chance = 0.7
	}
	if bd.Port || trIdx(g.W).dock[bd.ID] {
		chance = 0.9
	}
	if r.Chance(chance) && len(g.jyAvail()) > 0 {
		g.trPlace(zn, bd, "rigger", jyPick(jyRiggerTitle, g.jySkin()), fmt.Sprintf("The %s of %s, who sells and hires flying machines and cliff gliders.", jyPick(jyRiggerTitle, g.jySkin()), bd.Name), "SKY RIGGING — flying machines for sale or hire; cliff gliders; fuel and repairs.")
	}
	if trips := g.jyCaravanTrips(bd); len(trips) > 0 && r.Chance(0.65) {
		note := "CARAVAN GATE — passage with the wagons."
		for i, t := range trips {
			if i >= 4 {
				break
			}
			note += fmt.Sprintf(" To %s: %s.", t.To.Name, tvDuration(t.Days*DayLen))
		}
		g.trPlace(zn, bd, "caravanmaster", "caravan master", "A caravan master at the gate, hiring passage with the wagons.", note)
	}
	if trips := g.jyFerryTrips(bd); len(trips) > 0 && r.Chance(0.75) {
		note := "FERRY LANDING — passage by river and lake."
		for i, t := range trips {
			if i >= 4 {
				break
			}
			note += fmt.Sprintf(" To %s: %s.", t.To.Name, tvDuration(t.Days*DayLen))
		}
		g.trPlace(zn, bd, "ferryman", "ferry master", "A ferry master at the river landing, selling passage.", note)
	}
}

func jyTalk(g *Game, a *Actor) []Option {
	switch a.Role {
	case "rigger":
		return g.jyRiggerTalk(a)
	case "caravanmaster", "ferryman":
		bd := g.trHere()
		if bd == nil {
			return nil
		}
		var trips []jyTrip
		intro := "\"Wagons leave at dawn. Where to?\""
		if a.Role == "ferryman" {
			trips = g.jyFerryTrips(bd)
			intro = "\"The boat casts off with the tide. Downstream is quick, upstream is slow. Where to?\""
		} else {
			trips = g.jyCaravanTrips(bd)
		}
		if len(trips) == 0 {
			return nil
		}
		return []Option{{"{{B|I want passage.}}", func() { g.jyTripMenu(a, trips, intro) }}}
	}
	return nil
}

// ---- route puzzles ----

func (g *Game) jyWindow() int { return (g.Turn + startMinutes) / (2 * DayLen) }

// jyBlockWhy says whether parasang p is closed right now, by kind (flood pass avalanche sand fog fire snow) and why.
func (g *Game) jyBlockWhy(p int) (string, string) {
	w := g.W
	if p < 0 || p >= len(w.Biome) || w.Biome[p] == 0 {
		return "", ""
	}
	b := w.Biome[p]
	win := g.jyWindow()
	h := wxHash(w.wxSeed()^0x6a79, p, win)
	road, town := w.Road[p] > 0, w.Burg[p] > 0
	name := w.RegionName(p)
	season := g.Season()
	if w.Hazards != nil && !town {
		for _, e := range w.hzRecentAt(p) {
			if w.Year-e.Year > 0 || h >= 0.4 {
				continue
			}
			switch e.Kind {
			case "flood":
				if !road {
					return "flood", "Floodwater from this year's disaster still covers the way near " + name + " for two days."
				}
			case "wildfire":
				return "fire", "A wildfire still smoulders across the way near " + name + "."
			case "avalanche", "landslide":
				if !road {
					return "avalanche", "A slide has buried the track near " + name + "."
				}
			}
		}
	}
	if h >= 0.3 {
		return "", ""
	}
	k := w.WeatherAt(p, g.Turn).Kind
	id := ""
	if k != nil {
		id = k.ID
	}
	if w.River[p] > 0 && !road && !town {
		chance := 0.025
		if season == 0 {
			chance = 0.08
		}
		if h < chance {
			for i := 0; i < 6; i++ {
				if kk := w.WeatherAt(p, g.Turn-i*wxBucket).Kind; kk != nil && kk.Wet >= 2 {
					return "flood", "The river is in flood near " + name + ": the ford is under water for two days."
				}
			}
		}
	}
	if w.Height[p] >= 60 && season == 3 && !road && !town {
		if (id == "blizzard" && h < 0.22) || (id == "snow" && h < 0.08) {
			return "pass", "The pass near " + name + " is closed by the winter blizzards."
		}
	}
	if w.Height[p] >= 70 && season == 0 && !road && !town && h < 0.04 {
		return "avalanche", "The thaw has brought an avalanche down across the slope near " + name + "."
	}
	if (b == 1 || b == 2 || b == 19) && id == "sandstorm" && h < 0.3 {
		return "sand", "A sandstorm scours the dunes near " + name + ": nothing can be seen or followed."
	}
	if id == "fog" && (w.Height[p] >= 55 || b == 12) && !road && h < 0.15 {
		return "fog", "A fog bank hangs over " + name + ", too thick to tell the track from the drop."
	}
	if !town && season == 1 && (id == "heat" || k != nil && w.TempAt(p, g.Turn) >= 33) && h < 0.04 {
		switch b {
		case 3, 4, 5, 6, 9, 17, 18:
			return "fire", "A wildfire sweeps the dry land near " + name + "."
		}
	}
	if (b == 10 || b == 9 || b == 6) && (id == "snow" || id == "blizzard") && !road && !town && h < 0.05 {
		return "snow", "The snow lies too deep to wade near " + name + "."
	}
	return "", ""
}

// jyBlocked is the WorldMove hook: it reports a closed route and opens the list of workarounds.
func (g *Game) jyBlocked(np int) (bool, string) {
	kind, why := g.jyBlockWhy(np)
	if kind == "" {
		return false, ""
	}
	st := g.jy()
	win := g.jyWindow()
	if st.Cleared[np] == win+1 {
		return false, ""
	}
	how := ""
	switch {
	case g.P.Pow("wings") > 0:
		how = "fly over it"
	case g.P.Pow("phase") > 0 || g.P.Pow("teleport") > 0:
		how = "step past it"
	case kind == "flood" && (g.P.Pow("gills") > 0 || g.P.Has("swimmer")):
		how = "swim it"
	}
	if how != "" {
		g.Msg("{{K|" + why + " But you " + how + ".}}")
		return false, ""
	}
	jyBlkDx, jyBlkDy = np%g.W.W-g.WX, np/g.W.W-g.WY
	g.jyBlockMenu(np, kind, why, win)
	return true, why
}

func (g *Game) jyBlockMenu(np int, kind, why string, win int) {
	st := g.jy()
	w := g.W
	dx, dy := jyBlkDx, jyBlkDy
	r := NewRng(fmt.Sprintf("%s:jy:blk:%d:%d:%d", g.Seed, np, win, g.Turn/60))
	again := func() { g.Mode = "world"; g.WorldMove(dx, dy) }
	clear := func() { st.Cleared[np] = win + 1 }
	var opts []Option
	opts = append(opts, jyOpt("Wait for it to pass.", func() {
		left := (win+1)*2*DayLen - (g.Turn + startMinutes)
		g.tvPass(maxi(360, left))
		g.Msg("You wait out the " + map[string]string{"flood": "flood", "pass": "blizzard", "avalanche": "slide", "sand": "sandstorm", "fog": "fog", "fire": "fire", "snow": "snow"}[kind] + ".")
		again()
	}))
	hire := g.jyPrice(8 + g.P.Level*2)
	switch kind {
	case "flood":
		opts = append(opts,
			jyOpt(fmt.Sprintf("Hire a ferryman or a tinker's raft — %s", g.cuFormat(hire)), func() {
				if g.cuSpend(hire) {
					clear()
					g.Msg("A local poles you across on a raft.")
					again()
				}
			}),
			jyOpt("Search up and down the bank for a bridge or ford (half a day).", func() {
				g.tvPass(DayLen / 2)
				found := false
				for _, d := range dirs8 {
					for m := 1; m <= 2; m++ {
						x, y := np%w.W+d[0]*m, np/w.W+d[1]*m
						if w.In(x, y) {
							q := w.Idx(x, y)
							if (w.Road[q] > 0 && w.River[q] > 0) || w.Burg[q] > 0 {
								found = true
							}
						}
					}
				}
				if found {
					clear()
					g.Msg("You find a bridge a short walk away and cross by it.")
					g.jyDeed("flood", "You found a way across a flooded river.")
					again()
				} else {
					g.Msg("{{y|There is no crossing nearby. The water is too high to wade.}}")
					g.jyClose()
				}
			}))
	case "pass", "avalanche", "snow":
		opts = append(opts,
			jyOpt(fmt.Sprintf("Hire a local guide to break a trail — %s", g.cuFormat(hire)), func() {
				if g.cuSpend(hire) {
					clear()
					g.tvPass(DayLen / 2)
					g.Msg("The guide leads you around the worst of it.")
					again()
				}
			}))
		if g.P.Pow("climbing") > 0 || g.Skills["survival"] >= 2 {
			opts = append(opts, jyOpt("Climb over it yourself (a day, some risk).", func() {
				g.tvPass(DayLen)
				clear()
				if r.Chance(0.35) {
					g.hurt(g.P, r.Int(2, 8), nil, "fall")
					if g.P.HP <= 0 {
						return
					}
					g.Msg("You slip on the way but make it over.")
				} else {
					g.Msg("Hand over hand, you cross the obstruction.")
				}
				g.jyDeed("pass", "You crossed a closed mountain pass on your own.")
				again()
			}))
		}
	case "sand", "fog":
		opts = append(opts, jyOpt("Press on blindly (a day lost, risky).", func() {
			g.tvPass(DayLen)
			clear()
			if r.Chance(0.4) {
				g.Thirst += 200
				g.hurt(g.P, r.Int(2, 6), nil, "fall")
				if g.P.HP <= 0 {
					return
				}
				g.Msg("You lose the track and stumble about before finding it again.")
			}
			again()
		}))
	case "fire":
		opts = append(opts, jyOpt("Detour round the burn (two days).", func() {
			g.tvPass(2 * DayLen)
			clear()
			g.Msg("You swing wide of the flames, choking on smoke.")
			again()
		}))
	}
	opts = append(opts, jyOpt("Turn back.", g.jyClose))
	g.jyMenu("The way is closed", []string{"{{O|" + why + "}}"}, opts, g.jyClose)
}

// jyNews lists the closed routes near you.
func (g *Game) jyNews() []string {
	w := g.W
	if w == nil || g.Cur == nil {
		return nil
	}
	var out []string
	seen := map[string]bool{}
	for rad := 0; rad <= 7 && len(out) < 4; rad++ {
		for yy := -rad; yy <= rad; yy++ {
			for xx := -rad; xx <= rad; xx++ {
				if abs(xx) != rad && abs(yy) != rad {
					continue
				}
				if !w.In(g.WX+xx, g.WY+yy) {
					continue
				}
				kind, why := g.jyBlockWhy(w.Idx(g.WX+xx, g.WY+yy))
				if kind == "" || seen[kind+why] || len(out) >= 4 {
					continue
				}
				seen[kind+why] = true
				out = append(out, why)
			}
		}
	}
	return out
}

// jyNewsLines is the form wsNewsLines wants; see the report for the hook line.
func (g *Game) jyNewsLines() []wsLine {
	var out []wsLine
	for _, t := range g.jyNews() {
		out = append(out, wsLine{g.W.Idx(g.WX, g.WY), "{{O|Route closed}}: " + t})
	}
	return out
}

// ---- sidebar and enter ----

func jySide(g *Game) []string {
	st := g.jy()
	d := st.def(g)
	if d == nil || !st.Air {
		return nil
	}
	dir, str := g.shWind(g.W.Idx(g.WX, g.WY))
	s := fmt.Sprintf("{{C|%s}} hull %d/%d", titleCase(d.name(g)), st.Hull, d.Hull)
	if d.FuelMax > 0 {
		s += fmt.Sprintf(", %.1fd %s", st.Fuel, d.fuel(g))
	}
	return []string{s, "{{K|Wind " + shWindName(dir, str) + "}}"}
}

func jyEnter(g *Game, zn *Zone) {
	st := g.jy()
	if st.Air {
		st.Air = false
		if st.Craft != "" {
			st.Parked, st.At = true, g.W.Idx(zn.WX, zn.WY)
		}
	}
}

func init() {
	unLocalKeys = append(unLocalKeys, "jy")
	zoneHooks = append(zoneHooks, jyZone)
	talkHooks = append(talkHooks, jyTalk)
	sideHooks = append(sideHooks, jySide)
	enterHooks = append(enterHooks, jyEnter)
	modeKeys["jy"] = jyKeys
	modeDraws["jy"] = jyDraw
}
