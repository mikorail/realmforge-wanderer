package engine

import (
	"fmt"
	"math"
)

// Elevation and depth in metres (prefix tg). Realmforge heights are Azgaar-style 0..100 with sea
// level at 20: land maps nonlinearly up to ~8,800 m peaks, the sea floor down to ~-6,000 m. Strata
// below the surface are tgStrataM thick each; sky isles float tgSkyM above the ground below.
// Altitude matters: thin air above 3,500 m (winded, slow healing, thirst), altitude sickness above
// 5,000 m until you acclimatise, and deep strata grow hot (thirst, heat exhaustion).

const (
	tgSea       = 20   // Azgaar sea level
	tgStrataM   = 300  // metres per stratum below the surface
	tgSkyM      = 2000 // sky isles float this far above the ground below
	tgThinAir   = 3500
	tgSickAir   = 5000
	tgDeathZone = 7500
	tgAcclimMin = 3 * DayLen // minutes above 2,500 m to acclimatise to thin air
)

// tgMetres converts an Azgaar height (0..100, 20 = sea level) to metres above (or below) the sea.
// Monotone in h; planet kinds deepen or shallow the seas and lift volcanic peaks a little.
func tgMetres(h float64, planetKind string) float64 {
	if h >= tgSea {
		m := math.Pow(h-18, 2.06) - math.Pow(2, 2.06) + 1 // 20 → 1 m, 100 → ~8,760 m
		if planetKind == "volcanic" {
			m *= 1.08
		}
		return m
	}
	d := 6000 * math.Pow((tgSea-math.Max(h, 0))/tgSea, 1.6)
	switch planetKind {
	case "ocean":
		d *= 1.3
	case "desert":
		d *= 0.6
	case "shattered":
		d *= 1.15
	}
	return -math.Max(d, 1)
}

// tgParasangM is the elevation of parasang p in metres.
func (w *WorldData) tgParasangM(p int) float64 {
	if p < 0 || p >= len(w.Height) {
		return 0
	}
	return tgMetres(float64(w.Height[p]), w.PlanetKind)
}

// tgSurfaceM is the ground (or sea floor) elevation under tile (x, y) of a surface zone, smoothly
// interpolated between parasang centres so neighbouring zones agree.
func (w *WorldData) tgSurfaceM(wx, wy, zx, zy, x, y int) float64 {
	if w == nil || w.W == 0 {
		return 0
	}
	gx := float64(wx*3*MW+zx*MW+x) / float64(3*MW)
	gy := float64(wy*3*MH+zy*MH+y) / float64(3*MH)
	fx, fy := gx-0.5, gy-0.5
	x0, y0 := int(math.Floor(fx)), int(math.Floor(fy))
	tx, ty := fx-float64(x0), fy-float64(y0)
	at := func(px, py int) float64 {
		return w.tgParasangM(w.Idx(clampi(px, 0, w.W-1), clampi(py, 0, w.H-1)))
	}
	a := at(x0, y0) + (at(x0+1, y0)-at(x0, y0))*tx
	b := at(x0, y0+1) + (at(x0+1, y0+1)-at(x0, y0+1))*tx
	m := a + (b-a)*ty
	// the parasang's own sign wins: land never reads below the sea, the sea never above it
	if w.In(wx, wy) {
		own := w.tgParasangM(w.Idx(wx, wy))
		if own >= 0 && m < 1 {
			m = math.Max(1, own*0.5)
		}
		if own < 0 && m > -1 {
			m = math.Min(-1, own*0.5)
		}
	}
	return m
}

// tgElev is where tile (x, y) of zone zn sits: metres relative to the sea (ground, sea floor or
// sky), and how far below the surface it is (strata, deep layers, sea floor; 0 elsewhere).
func (g *Game) tgElev(zn *Zone, x, y int) (alt, below float64) {
	if zn == nil || g.W == nil {
		return 0, 0
	}
	ground := g.W.tgSurfaceM(zn.WX, zn.WY, zn.ZX, zn.ZY, x, y)
	switch {
	case zn.Z <= towerBase-1:
		return math.Max(ground, 0) + float64(towerBase-zn.Z)*5, 0
	case zn.Z < 0:
		return math.Max(ground, 0) + tgSkyM, 0
	case zn.Z == 0:
		if ground < 0 && zn.Kind != "sea" && !g.W.IsWater(g.W.Idx(zn.WX, zn.WY)) {
			ground = 1
		}
		return ground, 0
	case zn.Underwater:
		return ground, -ground
	}
	d := float64(zn.Z * tgStrataM)
	return math.Max(ground, 0) - d, d
}

// tgPlayerAlt is the player's altitude above the sea (negative below).
func (g *Game) tgPlayerAlt() float64 {
	if g.Cur == nil {
		return 0
	}
	a, _ := g.tgElev(g.Cur, g.P.X, g.P.Y)
	if g.Cur.Z == 0 && g.Cur.WaterAt(g.P.X, g.P.Y) > 0 && a < 0 {
		return 0 // swimming at the surface of the sea
	}
	return a
}

// tgFmt renders metres with thousands separators: 1240 → "1,240".
func tgFmt(m float64) string {
	n := int(math.Round(math.Abs(m)/10) * 10)
	if math.Abs(m) < 100 {
		n = int(math.Round(math.Abs(m)))
	}
	s := fmt.Sprint(n)
	out := ""
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			out += ","
		}
		out += string(c)
	}
	if m < 0 {
		return "-" + out
	}
	return out
}

// tgRockTemp is the rock temperature (°C) at depth d metres below the surface of parasang p.
func (g *Game) tgRockTemp(p int, d float64) int {
	base := 12.0
	if p >= 0 && p < len(g.W.Temp) {
		base = float64(g.W.Temp[p])
	}
	grad := 20.0 // °C per km
	if g.W.PlanetKind == "volcanic" {
		grad = 30
	}
	if g.W.PlanetKind == "ice" {
		grad = 16
	}
	if d < 200 {
		return int(base)
	}
	// caves settle toward the yearly mean, then the geothermal gradient takes over
	return int(math.Round(10 + (base-10)*0.5 + d/1000*grad))
}

// tgElevLine is the sidebar line: "1,240 m above the sea", "-340 m", "1,800 m below the surface".
func (g *Game) tgElevLine() string {
	zn := g.Cur
	if zn == nil {
		return ""
	}
	alt, below := g.tgElev(zn, g.P.X, g.P.Y)
	switch {
	case zn.Underwater:
		return fmt.Sprintf("{{B|%s m, under the sea}}", tgFmt(-below))
	case zn.Z > 0 && zn.Z < 100:
		t := g.tgRockTemp(g.W.Idx(zn.WX, zn.WY), below)
		col := "K"
		if t >= 40 {
			col = "O"
		}
		return fmt.Sprintf("{{K|%s m below the surface}} {{%s|%d°C}}", tgFmt(below), col, t)
	case zn.Z < 0:
		return "{{C|" + tgFmt(alt) + " m aloft}}" + tgAirTag(alt)
	}
	if g.Cur.WaterAt(g.P.X, g.P.Y) >= 1 && alt < 0 {
		return fmt.Sprintf("{{B|Sea, %s m deep}}", tgFmt(-alt))
	}
	if tag := tgAirTag(alt); tag != "" {
		return "{{K|" + tgFmt(alt) + " m up,}}" + tag
	}
	return "{{K|" + tgFmt(alt) + " m above the sea}}"
}

func tgAirTag(alt float64) string {
	switch {
	case alt >= tgDeathZone:
		return " {{R|death zone}}"
	case alt >= tgSickAir:
		return " {{O|very thin air}}"
	case alt >= tgThinAir:
		return " {{W|thin air}}"
	}
	return ""
}

// tgInfo is the world-map info for parasang p: elevation and notable landforms.
func (g *Game) tgInfo(p int) []string {
	w := g.W
	m := w.tgParasangM(p)
	var out []string
	if w.IsWater(p) {
		out = append(out, fmt.Sprintf("{{B|Sea depth %s m}}", tgFmt(m)))
	} else {
		out = append(out, "Elevation {{Y|"+tgFmt(m)+" m}}"+tgAirTag(m))
	}
	if v, act := tgVolcanoAt(w, p); v {
		if act {
			out = append(out, "{{R|An active volcano smokes here}}")
		} else {
			out = append(out, "{{O|A dormant volcano}}")
		}
	}
	if !w.IsWater(p) {
		var lf []string
		if tgCliffsAround(w, p) {
			lf = append(lf, "escarpments")
		}
		if tgGlacial(w, p) {
			lf = append(lf, "glacier ice")
		}
		if tgCanyonAt(w, p) {
			lf = append(lf, "canyons")
		}
		if len(lf) > 0 {
			s := lf[0]
			for i := 1; i < len(lf); i++ {
				s += ", " + lf[i]
			}
			out = append(out, "{{w|Terrain: "+s+"}}")
		}
	}
	return out
}

// ---- altitude and depth effects ----

type tgState struct {
	Acclim  int `json:"ac,omitempty"`  // minutes spent high up (decays when low)
	Warned  int `json:"wn,omitempty"`  // last air band warned about (0 none, 1 thin, 2 sick, 3 death)
	HeatW   int `json:"hw,omitempty"`  // last heat band warned about
	Sink    int `json:"sk,omitempty"`  // minutes stuck in quicksand
	Soak    int `json:"so,omitempty"`  // minutes soaking in a hot spring
	SoakDay int `json:"sd,omitempty"`  // day the last soak bonus was given
	LastHP  int `json:"lhp,omitempty"` // HP seen last minute (thin air slows healing)
}

func tgSt(g *Game) *tgState { return g.ext("tg", func() any { return &tgState{} }).(*tgState) }

func init() {
	skAddStatus(&skStatusDef{ID: "winded", Name: "Winded", Glyph: 'w', Speed: -10})
	skAddStatus(&skStatusDef{ID: "altsick", Name: "Altitude sickness", Glyph: 'a', AGI: -2, WIL: -3, Speed: -15, Tick: tgAltSickTick})
	skAddStatus(&skStatusDef{ID: "mired", Name: "Mired", Glyph: 'm', Speed: -60, DV: -4})
	skAddStatus(&skStatusDef{ID: "scalded", Name: "Scalded", Glyph: 's', AV: -1})
	skAddStatus(&skStatusDef{ID: "heatstroke", Name: "Heat exhaustion", Glyph: 'h', AGI: -2, Speed: -10})
	turnHooks = append(turnHooks, tgTurn)
}

func tgAltSickTick(g *Game, a *Actor) {
	if g.Turn%40 == 0 {
		g.hurt(a, 1, nil, "altitude sickness")
	}
}

// tgTurn runs once per game minute: altitude, depth heat and terrain hazards.
func tgTurn(g *Game) {
	if g.Cur == nil || g.Dead || g.OnWorld {
		return
	}
	st := tgSt(g)
	g.tgAltitudeTick(st)
	g.tgHeatTick(st)
	g.tgHazardTick(st)
	st.LastHP = g.P.HP
}

func (g *Game) tgAltitudeTick(st *tgState) {
	p := g.P
	alt := g.tgPlayerAlt()
	if alt >= 2500 {
		st.Acclim = mini(st.Acclim+1, 2*tgAcclimMin)
	} else if alt < 1500 && st.Acclim > 0 && g.Turn%3 == 0 {
		st.Acclim--
	}
	band := 0
	switch {
	case alt >= tgDeathZone:
		band = 3
	case alt >= tgSickAir:
		band = 2
	case alt >= tgThinAir:
		band = 1
	}
	if band > st.Warned {
		switch band {
		case 1:
			g.Msg("{{W|The air is thin up here. Your breath comes short.}}")
		case 2:
			g.Msg("{{O|The air is very thin. Your head pounds.}}")
		case 3:
			g.Msg("{{R|You are in the death zone. Nothing lives long this high.}}")
		}
	}
	st.Warned = band
	if band == 0 || p.Pow("wings") > 0 && alt < tgSickAir {
		return
	}
	acclim := st.Acclim >= tgAcclimMin
	if !acclim {
		p.AddSt("winded", 3)
	}
	// thin air: natural healing runs at half speed (one tick in two is lost)
	if p.HP == st.LastHP+1 && g.Turn%2 == 0 {
		p.HP--
	}
	if g.Turn%(6-band) == 0 {
		g.Thirst++ // dry, cold air
	}
	if band >= 2 {
		sick := !acclim || band == 3
		if sick && p.St("altsick") == 0 && NewRng(fmt.Sprintf("%s:tg:alt:%d", g.W.Seed, g.Turn/60)).Chance(0.35*float64(band-1)) {
			p.AddSt("altsick", 90)
			g.Msg("{{O|Altitude sickness: nausea, a splitting headache. Go lower.}}")
		}
	}
	if band == 3 && g.Turn%20 == 0 {
		g.hurt(p, 1, nil, "the death zone")
	}
}

func (g *Game) tgHeatTick(st *tgState) {
	zn := g.Cur
	p := g.P
	if zn.Z <= 0 || zn.Underwater || zn.Z >= 100 {
		st.HeatW = 0
		return
	}
	_, below := g.tgElev(zn, p.X, p.Y)
	t := g.tgRockTemp(g.W.Idx(zn.WX, zn.WY), below)
	band := 0
	switch {
	case t >= 55:
		band = 3
	case t >= 45:
		band = 2
	case t >= 35:
		band = 1
	}
	if band > st.HeatW {
		switch band {
		case 1:
			g.Msg(fmt.Sprintf("{{W|The rock is warm to the touch here (%d°C). You sweat.}}", t))
		case 2:
			g.Msg(fmt.Sprintf("{{O|Hot, close air (%d°C). Sweat stings your eyes.}}", t))
		case 3:
			g.Msg(fmt.Sprintf("{{R|The deep is an oven (%d°C). Drink, or turn back.}}", t))
		}
	}
	st.HeatW = band
	if band == 0 || p.mealLv("rime") > 0 && band < 3 {
		return
	}
	if g.Turn%(5-band) == 0 {
		g.Thirst++
	}
	if band == 3 && g.Thirst > 900 && p.St("heatstroke") == 0 && g.Turn%30 == 0 {
		p.AddSt("heatstroke", 60)
		g.Msg("{{R|Heat exhaustion takes you: dizzy, heavy limbs.}}")
	}
}
