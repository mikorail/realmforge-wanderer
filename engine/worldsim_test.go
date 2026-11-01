package engine

import (
	"strings"
	"testing"
)

func wsTestGame(t *testing.T) *Game {
	g := newGame(t, nil)
	g.Turn = 24 * 60 * 5
	g.wsAdvance()
	return g
}

func wsLeaveTown(g *Game) {
	g.Cur = g.GetZone(g.Cur.WX, g.Cur.WY, 1, 1, 0)
}

func TestWsCaravansMove(t *testing.T) {
	g := wsTestGame(t)
	s := g.ws()
	var tr *wsTrav
	for _, x := range s.Trav {
		if x.Kind == "caravan" && len(x.Path) >= 6 {
			tr = x
			break
		}
	}
	if tr == nil {
		t.Fatal("no caravan spawned")
	}
	for _, p := range tr.Path {
		if g.W.IsWater(p) {
			t.Fatal("path crosses water")
		}
	}
	roads := 0
	for _, p := range tr.Path {
		if g.W.Road[p] > 0 || g.W.Burg[p] > 0 {
			roads++
		}
	}
	if roads < len(tr.Path)/2 {
		t.Errorf("caravan off the roads: %d/%d", roads, len(tr.Path))
	}
	a, _ := tr.at(tr.Start)
	b, _ := tr.at(tr.Start + 3*tr.Hpp)
	if a == b {
		t.Error("caravan did not move over days")
	}
	if tr.Path[0] != a {
		t.Error("starts away from its town")
	}
	// determinism
	g2 := wsTestGame(t)
	if len(g2.ws().Trav) != len(s.Trav) {
		t.Errorf("not deterministic: %d vs %d", len(g2.ws().Trav), len(s.Trav))
	}
	// advancing time retires finished walkers
	g.Turn += 24 * 60 * 30
	g.wsAdvance()
	for _, x := range g.ws().Trav {
		if x.ID == tr.ID {
			t.Error("old caravan not retired")
		}
	}
}

func TestWsArmySpawnsOnEnter(t *testing.T) {
	g := newGame(t, nil)
	wx, wy, own, foe := firstFrontier(g, t)
	g.Turn = 24 * 60 * 3
	g.wsAdvance()
	s := g.ws()
	path := s.route(g, g.W.Idx(wx, wy), g.warBurg(foe).P)
	if len(path) < 3 {
		path = []int{g.W.Idx(wx, wy), g.W.Idx(wx, wy), g.W.Idx(wx, wy)}
	}
	h := g.wsHour()
	army := &wsTrav{ID: 777777, Kind: "army", Path: path, Start: h - 2*8, Hpp: 8, Realm: own, Foe: foe, Size: 8, Home: g.warBurg(own).ID}
	s.Trav = append(s.Trav, army)
	p, _ := army.at(h)
	wx, wy = p%g.W.W, p/g.W.W
	if g.W.Burg[p] > 0 {
		t.Skip("army on a town")
	}
	zn := g.GetZone(wx, wy, 1, 1, 0)
	g.enterZone(zn, 40, 12, true)
	n := 0
	for _, a := range zn.Actors {
		if id, ok := wsTagOf(a); ok && id == army.ID && a.Has("soldier") {
			n++
		}
	}
	if n < 4 {
		t.Errorf("army soldiers = %d", n)
	}
	// entering again does not duplicate
	g.enterZone(zn, 40, 12, true)
	m := 0
	for _, a := range zn.Actors {
		if id, ok := wsTagOf(a); ok && id == army.ID && a.Has("soldier") {
			m++
		}
	}
	if m != n {
		t.Errorf("duplicated %d -> %d", n, m)
	}
	// once it marches on, the zone empties of it
	g.Turn += 60 * 8 * 6
	g.wsAdvance()
	other := g.GetZone(wx, wy, 0, 0, 0)
	g.enterZone(other, 5, 5, true)
	g.enterZone(zn, 40, 12, true)
	for _, a := range zn.Actors {
		if id, ok := wsTagOf(a); ok && id == army.ID {
			t.Fatal("army still standing where it was")
		}
	}
}

func TestWsPlagueSpreadsAndPrices(t *testing.T) {
	g := wsTestGame(t)
	bl := g.ws().burgList(g)
	b := bl[0]
	g.ws().Ev = nil
	base := g.wsPrice(b.P, "medicine")
	e := g.wsAddEvent(&wsEvent{Kind: "plague", P: b.P, R: 1, Str: 4, Start: g.wsHour(), End: g.wsHour() + 24*30, Name: "Plague in " + b.Name})
	if e == nil {
		t.Fatal("event not added")
	}
	if m := g.wsPrice(b.P, "medicine"); m < base*1.8 {
		t.Errorf("medicine %.2f vs %.2f", m, base)
	}
	r0 := e.R
	for d := g.wsHour()/24 + 1; d < g.wsHour()/24+8; d++ {
		g.wsDaily(d)
	}
	if e.R <= r0 && len(g.ws().Ev) < 2 {
		t.Error("plague did not spread")
	}
	// the sick, wardens, and infection
	g.Cur = g.GetZone(b.P%g.W.W, b.P/g.W.W, 1, 1, 0)
	if g.Cur.Kind != "town" {
		t.Skip("burg zone not a town")
	}
	g.enterZone(g.Cur, 40, 12, true)
	sick, ward := 0, 0
	for _, a := range g.Cur.Actors {
		if a.Has("wssick") {
			sick++
		}
		if a.Has("wswarden") {
			ward++
		}
	}
	if sick == 0 || ward == 0 {
		t.Errorf("sick %d wardens %d", sick, ward)
	}
	for i := 0; i < 300 && g.P.St("plague") == 0; i++ {
		wsInfectCheck(g)
	}
	if g.P.St("plague") == 0 {
		t.Error("never infected in a plague town")
	}
}

func TestWsFreeCaptivesNoBuying(t *testing.T) {
	g := wsTestGame(t)
	wsLeaveTown(g)
	zn := g.Cur
	g.P.X, g.P.Y = 10, 10
	tr := &wsTrav{ID: 555, Kind: "slaver", Home: g.ws().burgList(g)[0].ID, Hpp: 6, Path: []int{1, 2, 3}}
	r := NewRng("t")
	g.wsSlaverGroup(zn, tr, 40, 12, r)
	var cap *Actor
	var slavers []*Actor
	for _, a := range zn.Actors {
		if a.Has("captive") {
			cap = a
		}
		if a.Has("slaver") {
			slavers = append(slavers, a)
		}
	}
	if cap == nil || len(slavers) == 0 {
		t.Fatal("no coffle")
	}
	cap.X, cap.Y = slavers[0].X+1, slavers[0].Y
	g.Talk = cap
	g.talkMenu("hi")
	for _, o := range g.Opts {
		l := strings.ToLower(o.Text)
		for _, bad := range []string{"buy", "sell", "purchase", "ransom", "slave market", "trade"} {
			if strings.Contains(l, bad) {
				t.Errorf("forbidden option %q", o.Text)
			}
		}
	}
	if len(cap.Inv) != 0 {
		t.Error("captive has stock")
	}
	free := func() {
		for _, o := range g.Opts {
			if strings.Contains(o.Text, "Cut ") {
				o.Do()
				return
			}
		}
		t.Fatal("no free option")
	}
	free()
	if !cap.Has("captive") {
		t.Fatal("freed while the keepers watch")
	}
	for _, s := range slavers {
		s.HP = 0
		g.removeActor(zn, s)
	}
	rep := g.Rep[cap.Faction]
	g.Talk = cap
	g.talkMenu("hi")
	free()
	if cap.Has("captive") || !cap.Has("wsfreed") {
		t.Fatal("not freed")
	}
	if g.Rep[cap.Faction] <= rep {
		t.Error("no reputation for freeing")
	}
	g.Talk = cap
	g.talkMenu("hi")
	for _, o := range g.Opts {
		if strings.Contains(o.Text, "Invite") {
			o.Do()
		}
	}
	// report to a guard
	e := g.wsAddEvent(&wsEvent{Kind: "slavepen", P: g.W.Idx(g.Cur.WX, g.Cur.WY), Str: 3, Start: g.wsHour(), End: g.wsHour() + 100, Name: "pen"})
	guard := g.soldier(g.W.State[g.W.Idx(g.Cur.WX, g.Cur.WY)], r)
	if guard == nil {
		guard = g.soldier(1, r)
	}
	guard.Role = "guard"
	g.Talk = guard
	found := false
	for _, o := range wsTalk(g, guard) {
		if strings.Contains(o.Text, "Report slavers") {
			found = true
			o.Do()
		}
	}
	if !found || e.Reported == 0 {
		t.Error("report option failed")
	}
}

func TestWsNewsScreen(t *testing.T) {
	g := wsTestGame(t)
	g.wsAddEvent(&wsEvent{Kind: "plague", P: g.ws().burgList(g)[3].P, R: 2, Str: 3, Start: g.wsHour(), End: g.wsHour() + 500, Name: "Plague test"})
	mainKeys["N"](g)
	if g.Mode != "wsnews" {
		t.Fatal("mode not set")
	}
	sc := g.Render()
	txt := string(sc.G)
	if !strings.Contains(txt, "World news") || !strings.Contains(txt, "Plague") {
		t.Error("news screen missing content")
	}
	modeKeys["wsnews"](g, "Escape")
	if g.Mode != "main" {
		t.Error("did not close")
	}
	// gossip option on an ordinary NPC
	wsLeaveTown(g)
	a := g.makeNPC("villager", g.ws().burgList(g)[0], nil, g.R)
	g.Talk = a
	ok := false
	for _, o := range wsTalk(g, a) {
		if strings.Contains(o.Text, "happening on the roads") {
			ok = true
		}
	}
	if !ok {
		t.Error("no road gossip")
	}
}
