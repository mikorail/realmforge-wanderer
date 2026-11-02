package engine

import (
	"strings"
	"testing"
)

// trInject lays a test line from the start town to another surface town and an air route between them.
func trInject(t *testing.T, g *Game) (from, to *BurgData) {
	t.Helper()
	w := g.W
	from = w.burgByID[g.Cur.Burg]
	if from == nil {
		t.Fatal("start zone has no town")
	}
	best := 0
	for i := range w.Burgs {
		b := &w.Burgs[i]
		if !b.Alive || b.Layer != "surface" || b.ID == from.ID {
			continue
		}
		d := abs(b.P%w.W-from.P%w.W) + abs(b.P/w.W-from.P/w.W)
		if d >= 4 && d <= 12 && (to == nil || d < best) {
			to, best = b, d
		}
	}
	if to == nil {
		t.Fatal("no second town")
	}
	var ps []int
	x, y := from.P%w.W, from.P/w.W
	tx, ty := to.P%w.W, to.P/w.W
	ps = append(ps, from.P)
	for x != tx || y != ty {
		if x != tx {
			x += sign(tx - x)
		} else {
			y += sign(ty - y)
		}
		ps = append(ps, w.Idx(x, y))
	}
	w.Transport = &TransportData{
		Rail:  []TrRailData{{ID: 1, Name: "Test Railway", Kind: "railway", Stations: []int{from.ID, to.ID}, Parasangs: ps}},
		Air:   []TrAirData{{A: from.ID, B: to.ID, Kind: "airship"}},
		Docks: []int{from.ID, to.ID},
	}
	trReindex(w)
	t.Cleanup(func() { w.Transport = nil; trReindex(w) })
	return
}

func TestTrTrainBeatsWalkingAndCostsFare(t *testing.T) {
	g := newGame(t, nil)
	from, to := trInject(t, g)
	trips, _ := g.trRailTrips(from.ID)
	if len(trips) != 1 || trips[0].To.ID != to.ID {
		t.Fatalf("trips %+v", trips)
	}
	tr := trips[0]
	walk := 0
	for _, p := range tr.Via[1:] {
		walk += g.tvMinutes(p)
	}
	if tr.Minutes*4 >= walk {
		t.Fatalf("train %d min vs walking %d min", tr.Minutes, walk)
	}
	if tr.Minutes >= DayLen {
		t.Errorf("train took %s, want hours", tvDuration(tr.Minutes))
	}
	g.P.Gold += 5000
	g.cuEarn(5000)
	gold := g.P.Gold
	t0 := g.Turn
	if !g.trRide(from, tr) {
		t.Fatal("ride refused")
	}
	if g.P.Gold >= gold {
		t.Errorf("no fare paid: %d → %d", gold, g.P.Gold)
	}
	if g.Turn-t0 != tr.Minutes {
		t.Errorf("ride took %d turns, want %d", g.Turn-t0, tr.Minutes)
	}
	if g.WX != to.P%g.W.W || g.WY != to.P/g.W.W || g.Cur.Burg != to.ID {
		t.Errorf("arrived at %d,%d burg %d, want %s", g.WX, g.WY, g.Cur.Burg, to.Name)
	}
	if g.tr().Rides != 1 {
		t.Errorf("rides %d", g.tr().Rides)
	}
	// the destination town has a stationmaster and a dockmaster
	roles := map[string]bool{}
	for _, a := range g.Cur.Actors {
		roles[a.Role] = true
	}
	for _, r := range []string{"stationmaster", "dockmaster", "ostler"} {
		if !roles[r] {
			t.Errorf("no %s in %s", r, to.Name)
		}
	}
	// air passage back
	air := g.trAirTrips(to.ID)
	if len(air) != 1 || !g.trRide(to, air[0]) || g.Cur.Burg != from.ID {
		t.Errorf("air passage back failed: %+v", air)
	}
}

func TestTrNoFareNoRide(t *testing.T) {
	g := newGame(t, nil)
	from, _ := trInject(t, g)
	g.cuSync()
	for g.P.Gold > 0 && g.cuSpend(g.P.Gold) {
	}
	trips, _ := g.trRailTrips(from.ID)
	if g.trRide(from, trips[0]) {
		t.Error("rode without paying")
	}
}

func TestTrCutLineRunsNoTrains(t *testing.T) {
	g := newGame(t, nil)
	from, _ := trInject(t, g)
	g.W.Transport.Rail[0].Cut = 300
	trips, cut := g.trRailTrips(from.ID)
	if len(trips) != 0 || len(cut) != 1 {
		t.Errorf("cut line: %d trips, %d cut", len(trips), len(cut))
	}
}

func TestTrMountedIsFaster(t *testing.T) {
	g := newGame(t, nil)
	g.Skills["survival"] = 0
	p := g.W.Idx(g.WX, g.WY)
	g.W.Road[p], g.W.Height[p], g.W.Biome[p] = 1, 20, 4
	foot := g.tvMinutes(p)
	cap0 := g.capacity()
	g.P.Gold += 2000
	g.cuEarn(2000)
	if !g.trBuyMount(trMountByID("horse"), 100) {
		t.Fatal("could not buy a horse")
	}
	ride := g.tvMinutes(p)
	if ride >= foot {
		t.Errorf("riding %d min, on foot %d", ride, foot)
	}
	if g.capacity() <= cap0 {
		t.Error("saddlebags add no capacity")
	}
	g.tr().Cart = "wagon"
	if g.capacity() <= cap0+25 {
		t.Error("wagon adds no capacity")
	}
	// the horse eats on the road away from grass
	g.W.Biome[p] = 6
	feed := g.tr().Mount.Feed
	g.trJourney(2 * DayLen)
	if m := g.tr().Mount; m != nil && m.Feed >= feed {
		t.Errorf("horse did not eat: %v → %v", feed, m.Feed)
	}
	// stabled: no bonus
	g.tr().Mount = &trMount{Def: "horse", Name: "bay horse", Feed: 5}
	g.tr().Stable = &trStable{Burg: g.Cur.Burg, Since: g.Turn, Town: "here"}
	if g.tvMinutes(p) < foot {
		t.Error("a stabled horse still speeds you up")
	}
	g.tr().Stable = nil
	// it survives a save
	g2, err := LoadGame(g.W, g.Save())
	if err != nil {
		t.Fatal(err)
	}
	if m := g2.tr().Mount; m == nil || m.Def != "horse" || g2.tr().Cart != "wagon" {
		t.Errorf("mount after load: %+v cart %q", m, g2.tr().Cart)
	}
}

func TestTrStarvingBeastDies(t *testing.T) {
	g := newGame(t, nil)
	p := g.W.Idx(g.WX, g.WY)
	g.W.Biome[p] = 6
	g.tr().Mount = &trMount{Def: "horse", Name: "dun horse", Feed: 0, Starve: 3.5}
	g.trJourney(DayLen)
	if g.tr().Mount != nil {
		t.Error("a starving horse kept going")
	}
}

func TestTrOldWorldMountsNoStations(t *testing.T) {
	g := newGame(t, nil)
	if g.W.Transport != nil {
		t.Skip("bundled world already carries transport")
	}
	bd := g.W.burgByID[g.Cur.Burg]
	if trips, _ := g.trRailTrips(bd.ID); len(trips) != 0 {
		t.Error("trips in a world without rail")
	}
	offer := g.trOffer(bd)
	if len(offer) == 0 {
		t.Fatal("ostler sells nothing")
	}
	var a *Actor
	for _, x := range g.Cur.Actors {
		if x.Role == "stationmaster" || x.Role == "dockmaster" {
			t.Errorf("%s in an old world", x.Role)
		}
	}
	// an ostler appears in a freshly generated town centre
	for i := range g.W.Burgs {
		b := &g.W.Burgs[i]
		if !b.Alive || b.Layer != "surface" || b.ID == bd.ID {
			continue
		}
		zn := g.GetZone(b.P%g.W.W, b.P/g.W.W, 1, 1, 0)
		for _, x := range zn.Actors {
			if x.Role == "ostler" {
				a = x
			}
		}
		break
	}
	if a == nil {
		t.Fatal("no ostler in a town")
	}
	g.Talk = a
	opts := trTalk(g, a)
	found := false
	for _, o := range opts {
		if strings.HasPrefix(o.Text, "Buy a ") {
			found = true
		}
	}
	if !found {
		t.Errorf("ostler options: %v", opts)
	}
}

func TestTrGenreMounts(t *testing.T) {
	g := newGame(t, nil)
	bd := g.W.burgByID[g.Cur.Burg]
	g.Genre = "cyberpunk"
	got := map[string]bool{}
	for _, d := range g.trOffer(bd) {
		got[d.ID] = true
	}
	if !got["skiff"] && !got["motorbike"] {
		t.Errorf("cyberpunk ostler offers %v", got)
	}
}
