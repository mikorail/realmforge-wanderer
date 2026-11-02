package engine

import (
	"math"
	"strings"
	"testing"
)

// shBFS finds a path of parasangs from a to b stepping on cells where ok holds (a and b always allowed).
func shBFS(w *WorldData, a, b int, ok func(p int) bool) []int {
	prev := map[int]int{a: -1}
	q := []int{a}
	for len(q) > 0 {
		p := q[0]
		q = q[1:]
		if p == b {
			var out []int
			for p != -1 {
				out = append([]int{p}, out...)
				p = prev[p]
			}
			return out
		}
		for _, d := range dirs8 {
			x, y := p%w.W+d[0], p/w.W+d[1]
			if !w.In(x, y) {
				continue
			}
			n := w.Idx(x, y)
			if _, seen := prev[n]; seen || n != b && !ok(n) {
				continue
			}
			prev[n] = p
			q = append(q, n)
		}
	}
	return nil
}

// shPorts finds two surface ports joined by sea and by land, where the land road is long.
func shPorts(t *testing.T, g *Game) (a, b *BurgData, sea, land []int) {
	t.Helper()
	w := g.W
	var ports []*BurgData
	for i := range w.Burgs {
		if bd := &w.Burgs[i]; bd.Alive && bd.Port && bd.Layer == "surface" {
			ports = append(ports, bd)
		}
	}
	for _, x := range ports {
		for _, y := range ports {
			if x == y {
				continue
			}
			d := maxi(abs(x.P%w.W-y.P%w.W), abs(x.P/w.W-y.P/w.W))
			if d < 4 || d > 14 {
				continue
			}
			s := shBFS(w, x.P, y.P, w.IsWater)
			l := shBFS(w, x.P, y.P, func(p int) bool { return !w.IsWater(p) })
			if s != nil && l != nil && len(l) >= len(s) {
				return x, y, s, l
			}
		}
	}
	t.Skip("no pair of ports joined by sea and land")
	return
}

// shDockAt puts the player in the zone of bd that has a shipwright and returns the shipwright.
func shDockAt(t *testing.T, g *Game, bd *BurgData) *Actor {
	t.Helper()
	w := g.W
	for zy := 0; zy < 3; zy++ {
		for zx := 0; zx < 3; zx++ {
			zn := g.GetZone(bd.P%w.W, bd.P/w.W, zx, zy, 0)
			for _, a := range zn.Actors {
				if a.Role == "shipwright" {
					g.enterZone(zn, a.X, a.Y+1, true)
					g.Mode = "main"
					return a
				}
			}
		}
	}
	t.Fatalf("no shipwright in %s", bd.Name)
	return nil
}

func shPick(t *testing.T, g *Game, opts []Option, sub string) {
	t.Helper()
	for _, o := range opts {
		if strings.Contains(o.Text, sub) {
			o.Do()
			return
		}
	}
	var ts []string
	for _, o := range opts {
		ts = append(ts, o.Text)
	}
	t.Fatalf("no option %q in %v", sub, ts)
}

// shBuy buys the first ship on offer at bd.
func shBuy(t *testing.T, g *Game, bd *BurgData) *shShip {
	t.Helper()
	a := shDockAt(t, g, bd)
	g.P.Gold = 100000
	g.Talk = a
	shPick(t, g, g.shWrightTalk(a, bd), "ships for sale")
	g.Opts[0].Do()
	sp := g.sh().Ship
	if sp == nil || sp.Port != bd.ID || sp.At != bd.P {
		t.Fatalf("no ship bought: %+v", sp)
	}
	g.Talk = a
	shPick(t, g, g.shWrightTalk(a, bd), "Hire")
	g.Talk = a
	shPick(t, g, g.shWrightTalk(a, bd), "stores")
	return sp
}

// shSailPath sails along parasangs ps (ps[0] is the port you leave).
func shSailPath(g *Game, ps []int) {
	w := g.W
	for _, p := range ps[1:] {
		g.WorldMove(p%w.W-g.WX, p/w.W-g.WY)
	}
}

func TestShBuyAndSailBeatsWalking(t *testing.T) {
	g := newGame(t, nil)
	a, b, sea, land := shPorts(t, g)
	sp := shBuy(t, g, a)
	if sp.Crew == 0 || sp.Prov < 20 {
		t.Fatalf("crew %d stores %.0f", sp.Crew, sp.Prov)
	}
	g.sh().quiet = true
	g.wxForce = &Weather{Kind: WeatherByID("clear"), Temp: 15}
	walk := 0
	for _, p := range land[1:] {
		walk += g.tvMinutes(p)
	}
	t0 := g.Turn
	if !g.shSetSail() || g.Mode != "world" || !sp.Aboard {
		t.Fatalf("could not set sail: mode %s", g.Mode)
	}
	shSailPath(g, sea)
	if sp.Aboard || sp.Port != b.ID || g.W.Idx(g.WX, g.WY) != b.P {
		t.Fatalf("did not dock at %s: at %d port %d aboard %v", b.Name, sp.At, sp.Port, sp.Aboard)
	}
	sail := g.Turn - t0
	t.Logf("%s → %s: sail %s (%d legs) vs walk %s (%d legs)", a.Name, b.Name, tvDuration(sail), len(sea)-1, tvDuration(walk), len(land)-1)
	if sail >= walk {
		t.Errorf("sailing %d min is not faster than walking %d", sail, walk)
	}
	g.Key("Enter")
	if g.Mode != "main" || g.Cur.Burg != b.ID {
		t.Errorf("could not go ashore at %s", b.Name)
	}
}

func TestShWindAndStorm(t *testing.T) {
	g := newGame(t, nil)
	a, _, sea, _ := shPorts(t, g)
	sp := shBuy(t, g, a)
	sp.Kind = "caravel" // sails
	np := sea[1]
	dx, dy := np%g.W.W-a.P%g.W.W, np/g.W.W-a.P/g.W.W
	g.wxForce = &Weather{Kind: WeatherByID("clear"), Temp: 15}
	st := g.sh()
	st.quiet = true
	h := math.Atan2(float64(dy), float64(dx))
	st.wind = &[2]float64{h, 0.6}
	tail := g.shLegMinutes(sp, np, dx, dy)
	st.wind = &[2]float64{h + math.Pi, 0.6}
	head := g.shLegMinutes(sp, np, dx, dy)
	st.wind = &[2]float64{0, 0.05}
	calm := g.shLegMinutes(sp, np, dx, dy)
	if !(tail < head && head < calm) {
		t.Errorf("tailwind %d, headwind %d, calm %d", tail, head, calm)
	}
	sp.Kind = "steamer"
	if eng := g.shLegMinutes(sp, np, dx, dy); eng >= calm {
		t.Errorf("an engine should not be becalmed: %d vs %d", eng, calm)
	}
	sp.Kind = "caravel"
	hull := sp.Hull
	g.wxForce = &Weather{Kind: WeatherByID("storm"), Temp: 12}
	g.shSetSail()
	g.WorldMove(dx, dy)
	if g.sh().Ship != nil && sp.Hull >= hull {
		t.Errorf("storm did no damage: %d → %d", hull, sp.Hull)
	}
}

func TestShPirateBoarding(t *testing.T) {
	g := newGame(t, nil)
	a, _, sea, _ := shPorts(t, g)
	shBuy(t, g, a)
	st := g.sh()
	st.quiet = true
	g.wxForce = &Weather{Kind: WeatherByID("clear"), Temp: 15}
	g.shSetSail()
	shSailPath(g, sea[:2])
	st.Fight = &shFight{Kind: "pirate", Name: "the Black Gull", Hull: 20, MaxHull: 20, Guns: 1, Kmd: 200, Crew: 3, Level: 1, Prize: 50}
	g.shFightMenu()
	if g.Mode != "ship" {
		t.Fatalf("no fight screen: %s", g.Mode)
	}
	shPick(t, g, g.Opts, "board")
	if g.Cur.Key != shBoardKey || len(st.Fight.Foes) != 3 {
		t.Fatalf("no boarding zone: %s foes %v", g.Cur.Key, st.Fight)
	}
	crew := 0
	for _, x := range g.Cur.Actors {
		if x.Has("shcrew") {
			crew++
		}
	}
	if crew == 0 {
		t.Errorf("no crew fights beside you")
	}
	g.P.HP, g.P.MaxHP = 999, 999
	gold := g.P.Gold
	for i := 0; i < 400 && st.Fight != nil; i++ {
		// walk to the nearest pirate and fight
		var foe *Actor
		for _, x := range g.Cur.Actors {
			if x.HP > 0 && g.Hostile(x) && (foe == nil || abs(x.X-g.P.X)+abs(x.Y-g.P.Y) < abs(foe.X-g.P.X)+abs(foe.Y-g.P.Y)) {
				foe = x
			}
		}
		if foe == nil {
			g.turn()
			continue
		}
		if abs(foe.X-g.P.X) <= 1 && abs(foe.Y-g.P.Y) <= 1 {
			foe.HP = 0
			g.kill(foe, g.P)
			g.turn()
			continue
		}
		g.P.X, g.P.Y = foe.X-1, foe.Y
		g.turn()
	}
	if st.Fight != nil {
		t.Fatalf("boarding never ended")
	}
	if g.P.Gold <= gold {
		t.Errorf("no prize: %d → %d", gold, g.P.Gold)
	}
	items := 0
	for _, it := range g.Cur.Items {
		if it.Kind != "corpse" {
			items++
		}
	}
	if items == 0 {
		t.Errorf("no plunder on their deck")
	}
	g.MoveP(0, -30) // off the edge: back to the helm
	g.P.Y = 0
	g.MoveP(0, -1)
	if g.Mode != "world" || !g.OnWorld {
		t.Errorf("walking off the deck should take the helm, mode %s", g.Mode)
	}
	for _, x := range g.Cur.Actors {
		if x.Has("shcrew") {
			t.Errorf("hired hands left behind in %s", g.Cur.Key)
		}
	}
}

func TestShSalvageWreck(t *testing.T) {
	g := newGame(t, map[string]int{"gills": 1})
	a, _, sea, _ := shPorts(t, g)
	sp := shBuy(t, g, a)
	st := g.sh()
	st.quiet = true
	g.wxForce = &Weather{Kind: WeatherByID("clear"), Temp: 15}
	g.shSetSail()
	var p int
	ok := false
	for _, q := range sea[1:] {
		if g.W.seaOpen[q] {
			p, ok = q, true
			break
		}
	}
	shSailPath(g, sea[:2])
	if !ok {
		// no sea floor: the crew salvages into the hold
		p = sp.At
	}
	st.Wrecks = append(st.Wrecks, shWreck{P: p, Name: "the Lost Petrel"})
	sp.At = p
	g.shArrive(p)
	g.shMenu()
	shPick(t, g, g.Opts, "Dive")
	if !st.Wrecks[0].Salvaged {
		t.Fatalf("wreck not salvaged")
	}
	if ok {
		if !g.Cur.Underwater || g.Cur.Z != 1 {
			t.Fatalf("not on the sea floor: %s", g.Cur.Key)
		}
		gold := 0
		for _, it := range g.Cur.Items {
			if it.Kind == "gold" {
				gold += it.Count
			}
		}
		if gold == 0 {
			t.Errorf("wreck holds no treasure")
		}
		g.P.X, g.P.Y = 30, 12
		g.Ascend()
		if g.Cur.Key != shDeckKey {
			t.Errorf("climbing the line should return to the deck, got %s", g.Cur.Key)
		}
	} else if len(sp.Hold) == 0 && !strings.Contains(g.Msgs[len(g.Msgs)-1].T, "empty-handed") {
		t.Errorf("crew salvage gave nothing and said nothing")
	}
}

func TestShSaveLoadAndFishing(t *testing.T) {
	g := newGame(t, nil)
	a, _, sea, _ := shPorts(t, g)
	sp := shBuy(t, g, a)
	g.sh().quiet = true
	g.wxForce = &Weather{Kind: WeatherByID("clear"), Temp: 15}
	g.shSetSail()
	shSailPath(g, sea[:2])
	g.Key("Enter") // walk the deck
	if g.Cur.Key != shDeckKey {
		t.Fatalf("Enter at sea should open the deck, got %s", g.Cur.Key)
	}
	g.shMenu()
	n := len(g.P.Inv)
	for i := 0; i < 10 && len(g.P.Inv) == n; i++ {
		g.shFish()
	}
	if len(g.P.Inv) == n {
		t.Errorf("ten casts and no fish")
	}
	g.Render()
	s := g.Save()
	h, err := LoadGame(g.W, s)
	if err != nil {
		t.Fatal(err)
	}
	hs := h.sh().Ship
	if hs == nil || hs.Name != sp.Name || hs.At != sp.At || !hs.Aboard || h.Cur.Key != shDeckKey {
		t.Fatalf("ship lost in save: %+v cur %s", hs, h.CurKey)
	}
	h.Key("Z")
	h.Render()
	h.MoveP(-40, 0)
	h.P.X = 0
	h.MoveP(-1, 0)
	if h.Mode != "world" {
		t.Errorf("deck edge did not take the helm: %s", h.Mode)
	}
	h.Render()
}

func TestShOldSaveLoads(t *testing.T) {
	g := newGame(t, nil)
	s := g.Save()
	h, err := LoadGame(g.W, s)
	if err != nil {
		t.Fatal(err)
	}
	if h.sh().Ship != nil {
		t.Errorf("a ship from nowhere")
	}
	h.Key("Z")
	h.Key("w")
	h.Key("ArrowLeft")
	h.Render()
}
