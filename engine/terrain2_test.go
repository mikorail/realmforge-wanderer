package engine

import (
	"strings"
	"testing"
)

func TestTgMetresMonotone(t *testing.T) {
	for _, pk := range []string{"terran", "ocean", "desert", "volcanic", "ice", "shattered", ""} {
		prev := tgMetres(0, pk)
		for h := 1; h <= 100; h++ {
			m := tgMetres(float64(h), pk)
			if m <= prev {
				t.Fatalf("%s: metres not increasing at h=%d: %.0f <= %.0f", pk, h, m, prev)
			}
			prev = m
		}
	}
	if m := tgMetres(20, "terran"); m < 0 || m > 10 {
		t.Errorf("sea level should be ~0 m, got %.0f", m)
	}
	if m := tgMetres(19, "terran"); m >= 0 || m < -200 {
		t.Errorf("just below sea level should be shallow, got %.0f", m)
	}
	if m := tgMetres(100, "terran"); m < 8000 || m > 9500 {
		t.Errorf("top peak should be ~8,800 m, got %.0f", m)
	}
	if m := tgMetres(0, "terran"); m > -5000 || m < -7000 {
		t.Errorf("deepest sea should be ~-6,000 m, got %.0f", m)
	}
	if m := tgMetres(50, "terran"); m < 800 || m > 2000 {
		t.Errorf("mid uplands out of range: %.0f", m)
	}
	if s := tgFmt(1243); s != "1,240" {
		t.Errorf("tgFmt(1243) = %q", s)
	}
	if s := tgFmt(-340); s != "-340" {
		t.Errorf("tgFmt(-340) = %q", s)
	}
}

func TestTgElevationLines(t *testing.T) {
	g := newGame(t, nil)
	if l := g.tgElevLine(); !strings.Contains(l, "m above the sea") {
		t.Errorf("town sidebar: %q", l)
	}
	// strata: depth from strata thickness
	zn := g.GetZone(g.WX, g.WY, 0, 0, 4)
	g.enterZone(zn, 40, 12, true)
	if l := g.tgElevLine(); !strings.Contains(l, "1,200 m below the surface") {
		t.Errorf("strata 4 sidebar: %q", l)
	}
	if hot := g.tgRockTemp(g.W.Idx(g.WX, g.WY), 4200); hot < 80 {
		t.Errorf("4.2 km down should be hot, got %d°C", hot)
	}
	info := strings.Join(g.tgInfo(g.W.Idx(g.WX, g.WY)), "|")
	if !strings.Contains(info, "Elevation") {
		t.Errorf("world info: %q", info)
	}
	t.Logf("sidebar %q · %s", g.tgElevLine(), info)
}

// tgFindZone searches the world for a surface zone (generated fresh) matching ok.
func tgFind(g *Game, pick func(p int) bool) (int, bool) {
	for p := range g.W.Biome {
		if pick(p) {
			return p, true
		}
	}
	return 0, false
}

func countT(zn *Zone, ts ...Terrain) int {
	n := 0
	for _, t := range zn.T {
		for _, u := range ts {
			if t == u {
				n++
			}
		}
	}
	return n
}

func TestTgCliffNeedsClimbing(t *testing.T) {
	g := newGame(t, nil)
	w := g.W
	var cz *Zone
	for p := range w.Biome {
		if !tgCliffsAround(w, p) {
			continue
		}
		wx, wy := p%w.W, p/w.W
		for _, zc := range [][2]int{{2, 1}, {0, 1}, {1, 2}, {1, 0}} {
			zn := g.genZone(wx, wy, zc[0], zc[1], 0)
			if countT(zn, TCliff) > 10 {
				cz = zn
				break
			}
		}
		if cz != nil {
			break
		}
	}
	if cz == nil {
		t.Skip("no escarpment in this world")
	}
	g.Zones[cz.Key] = cz
	g.enterZone(cz, 40, 12, true)
	// stand next to a cliff tile with open ground on our side
	for i, tt := range cz.T {
		x, y := i%MW, i/MW
		if tt != TCliff || x < 2 || x > MW-3 || y < 2 || y > MH-3 {
			continue
		}
		for _, d := range dirs8 {
			sx, sy := x-d[0], y-d[1]
			if cz.At(sx, sy) == TGrass || cz.At(sx, sy) == TDirt || cz.At(sx, sy) == TSand || cz.At(sx, sy) == TSnow || cz.At(sx, sy) == TAsh {
				cz.Actors = nil
				g.P.X, g.P.Y = sx, sy
				g.MoveP(d[0], d[1])
				if g.P.X == x && g.P.Y == y {
					t.Fatalf("walked onto a cliff without climbing")
				}
				g.P.Powers = map[string]int{"climbing": 1}
				g.MoveP(d[0], d[1])
				if g.P.X != x || g.P.Y != y {
					t.Fatalf("could not climb the cliff with the climbing power")
				}
				t.Logf("cliff in %s (%s)", cz.Key, cz.Name)
				return
			}
		}
	}
	t.Skip("no cliff with open ground beside it")
}

func TestTgGeyserErupts(t *testing.T) {
	zn := newZone("1.2.0.0.0", 1, 2, 0, 0, 0)
	i := 12*MW + 40
	var starts []int
	_, period := tgGeyserPhase("seed", zn, i, 0)
	for turn := 1; turn < 600; turn++ {
		a, _ := tgGeyserPhase("seed", zn, i, turn)
		b, _ := tgGeyserPhase("seed", zn, i, turn-1)
		if a == 2 && b != 2 {
			starts = append(starts, turn)
		}
	}
	if len(starts) < 4 {
		t.Fatalf("geyser erupted only %d times in 600 minutes", len(starts))
	}
	for k := 1; k < len(starts); k++ {
		if starts[k]-starts[k-1] != period {
			t.Fatalf("eruptions not periodic: %v (period %d)", starts, period)
		}
	}
	// and it scalds whoever stands beside it
	g := newGame(t, nil)
	clearAround(g)
	g.Cur.Set(g.P.X+1, g.P.Y, TGeyser)
	gi := g.P.Y*MW + g.P.X + 1
	for turn := g.Turn; ; turn++ {
		if ph, _ := tgGeyserPhase(g.W.Seed, g.Cur, gi, turn); ph == 2 {
			g.Turn = turn
			break
		}
	}
	hp := g.P.HP
	g.tgGeysers()
	if g.P.HP >= hp || g.P.St("scalded") == 0 {
		t.Errorf("geyser did not scald the player (hp %d → %d)", hp, g.P.HP)
	}
}

func TestTgVolcanoZone(t *testing.T) {
	found := false
	for _, name := range []string{"ember-crown", "ashen-vaults", "hollow-crown", "neon-tide", "spore-mother"} {
		w := world(t, name)
		for _, pw := range w.uni.Planets {
			g := NewGame(pw, CharSpec{Name: "Tester", Species: "human", Calling: "warrior", Attr: [6]int{18, 16, 16, 14, 14, 14}})
			for p := range pw.Biome {
				v, act := tgVolcanoAt(pw, p)
				if !v {
					continue
				}
				zx, zy := tgVolcanoZone(pw, p)
				zn := g.genZone(p%pw.W, p/pw.W, zx, zy, 0)
				if !strings.Contains(zn.Name, "Crater") {
					continue // a town or a city ward took the zone
				}
				lava, rim := countT(zn, TLava), countT(zn, TCliff)
				if lava == 0 || rim < 10 {
					t.Fatalf("%s %s: volcano zone has %d lava, %d crater wall", name, zn.Key, lava, rim)
				}
				if !found {
					t.Logf("%s planet %s: %s active=%v lava=%d rim=%d", name, pw.PlanetKind, zn.Name, act, lava, rim)
				}
				found = true
			}
		}
	}
	if !found {
		t.Fatal("no volcano in any bundled world")
	}
}

func TestTgLandformCensus(t *testing.T) {
	for _, name := range []string{"ember-crown", "spore-mother"} {
		w := world(t, name)
		for _, pw := range w.uni.Planets {
			v, cl, gl, cy := 0, 0, 0, 0
			for p := range pw.Biome {
				if ok, _ := tgVolcanoAt(pw, p); ok {
					v++
				}
				if tgCliffsAround(pw, p) {
					cl++
				}
				if tgGlacial(pw, p) {
					gl++
				}
				if tgCanyonAt(pw, p) {
					cy++
				}
			}
			t.Logf("%s/%s (%s): %d parasangs, volcanoes %d, escarpments %d, glacial %d, canyons %d", name, pw.Name, pw.PlanetKind, len(pw.Biome), v, cl, gl, cy)
		}
	}
}

func TestTgFallIntoIceCave(t *testing.T) {
	g := newGame(t, nil)
	w := g.W
	p, ok := tgFind(g, func(p int) bool { return tgGlacial(w, p) && w.Burg[p] == 0 })
	if !ok {
		t.Skip("no glacier here")
	}
	below := g.genZone(p%w.W, p/w.W, 0, 0, 1)
	if below.Kind == "strata" && !strings.Contains(below.Name, "Ice") {
		t.Errorf("strata under a glacier is %q", below.Name)
	}
}

func TestTgOldSaveLoads(t *testing.T) {
	g := newGame(t, nil)
	g.spend(1000)
	s := g.Save()
	// a save from before this feature has no "tg" store
	s2 := strings.Replace(s, `"tg":`, `"tgx":`, 1)
	for _, raw := range []string{s, s2} {
		g2, err := LoadGame(g.W, raw)
		if err != nil {
			t.Fatalf("load: %v", err)
		}
		g2.spend(1000)
		_ = g2.tgElevLine()
	}
}

func TestTgThinAir(t *testing.T) {
	g := newGame(t, nil)
	w := g.W
	p, ok := tgFind(g, func(p int) bool { return w.tgParasangM(p) >= 5500 && w.Burg[p] == 0 })
	if !ok {
		t.Skip("no high peak")
	}
	zn := g.GetZone(p%w.W, p/w.W, 1, 1, 0)
	g.enterZone(zn, 40, 12, true)
	for i := 0; i < 120; i++ {
		g.turn()
	}
	if g.P.St("winded") == 0 {
		t.Errorf("not winded at %s", g.tgElevLine())
	}
	t.Logf("%s: %s, statuses %v", zn.Name, g.tgElevLine(), g.P.Status)
}
