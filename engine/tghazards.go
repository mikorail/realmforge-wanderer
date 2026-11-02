package engine

import "fmt"

// Terrain hazards and boons (prefix tg): slick glacier ice, quicksand and tar that mire you,
// waterfalls and spray that douse fire, hot springs that mend, fumaroles that poison, geysers
// that erupt on a fixed per-vent schedule and scald everything near, and obsidian/sulfur to mine.
// All rolls come from side hashes of seed + turn + tile, never g.R.

// tgRoll is a deterministic roll in [0,1) for a purpose, turn and tile.
func (g *Game) tgRoll(what string, i int) float64 {
	return NewRng(fmt.Sprintf("%s:tg:%s:%d:%d", g.W.Seed, what, g.Turn, i)).Float()
}

// tgStep adjusts the time to step onto (nx, ny) and applies entry effects (hook in stepTo).
func (g *Game) tgStep(nx, ny, cost int) int {
	z, p := g.Cur, g.P
	t := z.At(nx, ny)
	switch t {
	case TGlacier:
		if p.Pow("climbing") == 0 && p.Pow("wings") == 0 && g.tgRoll("slip", ny*MW+nx) < 0.12 && p.M(AGI)+g.Skills["survival"] < 3 {
			g.Msg("{{C|Your feet go out from under you on the glacier ice.}}")
			cost += 1000
		}
	case TQuicksand:
		if p.St("mired") == 0 {
			g.Msg("{{W|The sand sucks at your legs. Keep moving!}}")
		}
		p.AddSt("mired", 3)
		cost *= 2
	case TTar:
		if p.St("mired") == 0 {
			g.Msg("{{W|You wade into sticky black tar.}}")
		}
		p.AddSt("mired", 4)
		cost *= 2
	case TFalls:
		g.Msg("{{C|You push through the roaring curtain of water.}}")
		cost += 500
		fallthrough
	case TSpray:
		if p.St("burning") > 0 {
			p.Status["burning"] = 0
			g.Msg("The spray puts out the flames.")
		}
	case THotSpring:
		if z.At(p.X, p.Y) != THotSpring {
			g.Msg("{{C|You ease into the steaming spring.}}")
		}
	case TFumarole:
		g.Msg("{{O|Hot, rotten-smelling gas hisses round your feet.}}")
	}
	return cost
}

// tgGeyserPhase is a vent's state on a turn: 0 quiet, 1 rumbling, 2 erupting; and its period.
// Each vent has its own fixed period (45..104 minutes) and phase, from the seed, zone and tile.
func tgGeyserPhase(seed string, zn *Zone, i, turn int) (int, int) {
	h := HashStr(fmt.Sprintf("%s:tg:geyser:%s:%d", seed, zn.Key, i))
	period := 45 + int(h%60)
	m := (turn + int((h>>8)%uint64(period))) % period
	switch {
	case m < 3:
		return 2, period
	case m == period-1:
		return 1, period
	}
	return 0, period
}

func (g *Game) tgHazardTick(st *tgState) {
	z, p := g.Cur, g.P
	here := z.At(p.X, p.Y)
	// quicksand and tar
	switch here {
	case TQuicksand:
		p.AddSt("mired", 2)
		st.Sink++
		if st.Sink > 4 && st.Sink%3 == 0 {
			g.Msg("{{R|You sink to the waist in quicksand!}}")
			g.hurt(p, 1+st.Sink/8, nil, "quicksand")
		}
	case TTar:
		p.AddSt("mired", 2)
		st.Sink = 0
		if p.St("burning") > 0 {
			g.hurt(p, 2, nil, "burning tar")
		}
	default:
		st.Sink = 0
	}
	// hot springs
	if here == THotSpring {
		st.Soak++
		if st.Soak%6 == 0 && p.HP < p.MaxHP {
			p.HP++
		}
		if st.Soak == 20 {
			for _, s := range []string{"ill", "exhausted", "altsick", "heatstroke", "scalded"} {
				delete(p.Status, s)
			}
			if day := g.Turn / DayLen; st.SoakDay != day+1 {
				st.SoakDay = day + 1
				p.AddSt("regen", 20)
				g.Msg("{{G|The hot water loosens every knot in you.}}")
			}
		}
	} else {
		st.Soak = 0
	}
	// fumaroles
	near := false
	for _, d := range append(dirs8[:], [2]int{0, 0}) {
		if z.At(p.X+d[0], p.Y+d[1]) == TFumarole {
			near = true
		}
	}
	if near && g.Turn%4 == 0 && p.Pow("gills") == 0 {
		if p.St("poisoned") == 0 {
			g.Msg("{{O|You choke on sulfurous fumes.}}")
		}
		p.AddSt("poisoned", 4)
		g.Thirst++
	}
	// salt glare by day
	if here == TSalt && z.Z == 0 && g.daylight() && g.Turn%3 == 0 {
		g.Thirst++
	}
	g.tgGeysers()
}

// tgGeysers erupts the current zone's vents on their schedules.
func (g *Game) tgGeysers() {
	z := g.Cur
	for i, t := range z.T {
		if t != TGeyser {
			continue
		}
		x, y := i%MW, i/MW
		ph, _ := tgGeyserPhase(g.W.Seed, z, i, g.Turn)
		if ph == 0 {
			continue
		}
		if ph == 1 {
			if abs(g.P.X-x) <= 8 && abs(g.P.Y-y) <= 4 && g.visible(x, y) {
				g.Msg("{{C|The geyser gurgles and the ground trembles.}}")
			}
			continue
		}
		if p0, _ := tgGeyserPhase(g.W.Seed, z, i, g.Turn-1); p0 != 2 && g.visible(x, y) {
			g.Msg("{{Y|The geyser erupts in a roaring column of boiling water!}}")
		}
		for dy := -2; dy <= 2; dy++ {
			for dx := -2; dx <= 2; dx++ {
				if z.In(x+dx, y+dy) {
					z.Fire[(y+dy)*MW+x+dx] = 0
				}
			}
		}
		scald := func(a *Actor) {
			if a.HP <= 0 || abs(a.X-x) > 2 || abs(a.Y-y) > 2 {
				return
			}
			if a.Status != nil {
				a.Status["burning"] = 0
			}
			a.AddSt("scalded", 10)
			g.hurt(a, 2+int(g.tgRoll("scald", a.ID)*4), nil, "a geyser")
		}
		for _, a := range append([]*Actor{}, z.Actors...) {
			scald(a)
		}
		if !g.Dead {
			scald(g.P)
		}
	}
}

// tgBump mines obsidian and sulfur (hook in interact).
func tgBump(g *Game, x, y int) bool {
	z := g.Cur
	switch z.At(x, y) {
	case TObsidian:
		z.Set(x, y, TRubble)
		n := 1 + int(g.tgRoll("obs", y*MW+x)*3)
		g.give(&Item{ID: g.newID(), Kind: "trophy", Name: "obsidian shards", Glyph: '*', F: CDarkMagenta, Value: 6 * n, Weight: n})
		g.give(g.MakeScrap(g.R, 1))
		g.Msg(fmt.Sprintf("You knap %d keen obsidian shards from the glass.", n))
		g.spend(2000)
		return true
	case TSulfur:
		z.Set(x, y, TAsh)
		g.give(&Item{ID: g.newID(), Kind: "trophy", Name: "lump of sulfur", Glyph: '*', F: CYellow, Value: 8, Weight: 1})
		g.Msg("You scrape off a reeking lump of sulfur.")
		g.spend(1500)
		return true
	}
	return false
}
