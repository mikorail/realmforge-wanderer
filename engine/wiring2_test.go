package engine

import (
	"strings"
	"testing"
)

func TestWiringWave2All(t *testing.T) {
	g := newGame(t, nil)

	// 1. Night-sight reading alnight in recompute()
	g.Cur.Dark = true
	for i := range g.Cur.Lit {
		g.Cur.Lit[i] = false
	}
	for i := range g.P.Body {
		g.P.Body[i].Item = nil
	}
	tx, ty := clampi(g.P.X+8, 1, MW-2), g.P.Y
	for x := mini(g.P.X, tx); x <= maxi(g.P.X, tx); x++ {
		g.Cur.Set(x, ty, TFloor)
	}
	g.recompute()
	if g.visible(tx, ty) {
		t.Fatalf("expected tile at distance 8 in pitch dark to be unseen without alnight")
	}
	g.P.AddSt("alnight", 50)
	g.recompute()
	if !g.visible(tx, ty) {
		t.Fatalf("expected alnight status to grant night-sight out to distance 8+")
	}
	delete(g.P.Status, "alnight")
	g.Cur.Dark = false

	// 2. Footprint tiles on the map (ssTrailAt in drawMap)
	fx, fy := clampi(g.P.X+2, 1, MW-2), g.P.Y
	g.Cur.Set(fx, fy, TFloor)
	g.Cur.Items = nil
	g.ss().Trails[g.Cur.Key] = []ssMark{{X: fx, Y: fy, T: g.Turn, Seen: true, B: true}}
	g.recompute()
	sc := g.Render()
	if got := sc.G[fy*SW+fx]; got != ',' {
		t.Fatalf("expected revealed blood trail glyph ',' at (%d,%d), got %q", fx, fy, got)
	}

	// 3. Trade goods valued by local tqPrice in price(), and honest merchants refusing banned goods (uwNoSale)
	var good *tqGood
	for i := range tqGoods {
		if tqGoods[i].Ban != "" {
			good = &tqGoods[i]
			break
		}
	}
	if good == nil {
		good = &tqGoods[0]
	}
	it := g.tqNewItem(good, 3, good.Base, 0)
	bd := g.localBurg()
	if bd == nil && len(g.W.Burgs) > 0 {
		bd = &g.W.Burgs[0]
		g.Cur.Kind = "town"
		g.Cur.Burg = bd.ID
	}
	if bd != nil {
		merch := &Actor{ID: g.newID(), Name: "merchant", Role: "merchant", State: bd.State, Faction: stateFaction(bd.State)}
		g.Talk = merch
		wantBuy := g.tqPrice(bd, good, true) * 3
		if got := g.price(it, true); got != wantBuy {
			t.Fatalf("expected merchant buy price %d for 3x %s, got %d", wantBuy, good.ID, got)
		}
		wantSell := g.tqPrice(bd, good, false) * 3
		if got := g.price(it, false); got != wantSell {
			t.Fatalf("expected merchant sell price %d for 3x %s, got %d", wantSell, good.ID, got)
		}
		// Add an active embargo on good in bd.State so tqBanned(bd.State, good) is true
		g.tq().Ev = append(g.tq().Ev, &tqEvent{Kind: "embargo", Realm: bd.State, Good: good.ID, Start: -10, End: g.tqToday() + 50})
		if !g.tqBanned(bd.State, good) {
			t.Fatalf("expected %s to be banned in state %d during embargo", good.ID, bd.State)
		}
		if !g.uwNoSale(it, merch) {
			t.Fatalf("expected honest merchant to refuse banned trade good %s", good.ID)
		}
	}

	// 4. buDigSites and buRivalLoot wired into rival.go + rvCull wired into endgame.go
	sites := g.buDigSites()
	if len(sites) == 0 {
		t.Fatalf("expected buDigSites() to return live hoard parasangs")
	}
	g.rvEnsure()
	rv := g.rv().R[0]
	rv.Alive = true
	rv.Pos = sites[0]
	rv.Dest = sites[0]
	rv.Act = "hoard"
	g.rvResolveHoard(rv, g.rvToday())
	if g.buRivalLoot(sites[0]) {
		t.Fatalf("expected hoard at %d to already be looted by rival", sites[0])
	}
	// Test rvCull
	rv.Alive = true
	rv.Allied = false
	rv.Pos = sites[0]
	g.rvCull(sites[0], 1, 1.0, "slain by the test threat")
	if rv.Alive {
		t.Fatalf("expected rvCull with frac=1.0 to kill rival at %d", sites[0])
	}

	// 5. In-zone cliff launch for gliders
	g.jy().Glider = true
	g.Cur.Z = 0
	cx, cy := clampi(g.P.X+1, 2, MW-10), g.P.Y
	g.Cur.Set(cx, cy, TCliff)
	for step := 2; step <= 6; step++ {
		g.Cur.Set(g.P.X+step, cy, TFloor)
	}
	if !g.jyCliffBump(cx, cy) {
		t.Fatalf("expected jyCliffBump to open cliff launch menu when bumping TCliff with glider")
	}
	g.jyClose()
	oldX := g.P.X
	if !g.jyZoneGlide(1, 0) || g.P.X <= oldX+1 {
		t.Fatalf("expected jyZoneGlide(1,0) to glide across the cliff (oldX=%d, newX=%d)", oldX, g.P.X)
	}

	// 6. Researched powers (ar_<id>) for sandbox and companions
	pw := g.arSandboxGrant()
	if pw == nil || g.ar().Powers[pw.ID] == nil {
		t.Fatalf("expected arSandboxGrant to register a researched power")
	}
	comp := g.Spawn("bandit", 4, NewRng("test:comp"))
	comp.AI = "follower"
	comp.Faction = "player"
	comp.X, comp.Y = clampi(g.P.X+1, 1, MW-2), g.P.Y
	g.Cur.Actors = append(g.Cur.Actors, comp)
	cpList := g.cpPowers(comp)
	foundAr := false
	for _, cp := range cpList {
		if cp.ID == "ar_"+pw.ID {
			foundAr = true
			break
		}
	}
	if !foundAr {
		t.Fatalf("expected companion cpPowers to include researched power ar_%s", pw.ID)
	}
	foe := g.Spawn("bandit", 2, NewRng("test:foe"))
	foe.AI = "hostile"
	foe.Angry = true
	foe.Faction = "outlaws"
	foe.X, foe.Y = clampi(comp.X+2, 1, MW-2), comp.Y
	g.Cur.Actors = append(g.Cur.Actors, foe)
	for x := comp.X; x <= foe.X; x++ {
		g.Cur.Set(x, comp.Y, TFloor)
	}
	foeHP0 := foe.HP
	used := g.cpUse(comp, cpPower{ID: "ar_" + pw.ID, Name: pw.Name, Lv: pw.Mag}, []*Actor{foe}, 0, false, false)
	if !used || foe.HP >= foeHP0 {
		t.Fatalf("expected companion to cast ar_%s on foe (used=%v, hp %d -> %d)", pw.ID, used, foeHP0, foe.HP)
	}

	// 7. Predator nest range (10 tiles instead of 6)
	var predSp *bsSp
	for _, s := range bsAll {
		if s.Temper == "predator" {
			predSp = s
			break
		}
	}
	if predSp != nil {
		par := g.bsSpawn(predSp, 4, NewRng("test:nestpar"))
		par.HomeX, par.HomeY = clampi(g.P.X+9, 1, MW-2), g.P.Y
		par.X, par.Y = par.HomeX, par.HomeY
		par.Angry = false
		par.AI = "wander"
		par.Set("ntpar")
		par.Set(ntNestFlag(999))
		delete(g.P.Flags, "sneak")
		delete(g.P.Status, "hidden")
		g.Cur.Actors = append(g.Cur.Actors, par)
		g.ntParents()
		if !par.Angry {
			t.Fatalf("expected nest parent at distance 9 (<= 10) to rear up and defend its nest")
		}
	}

	// 8. fmHeir wired to a legacy run (fmContinueAsHeir)
	parentName := g.P.Name
	g.P.Level = 10
	st := g.fm()
	st.Kids = append(st.Kids, &fmChild{
		ID:      g.newID(),
		Name:    "Kaelen Legacy",
		Species: g.P.Species,
		Born:    g.Day() - YearDays*5, // 5 years old -> grown
		Attr:    [6]int{14, 14, 14, 12, 12, 12},
		Calling: "warden",
		Gift:    "a steady hand",
		Bond:    15,
	})
	if h := g.fmHeir(); h == nil || h.Name != "Kaelen Legacy" {
		t.Fatalf("expected fmHeir() to find grown child Kaelen Legacy, got %+v", h)
	}
	g.Dead = true
	g.DeathText = "You fell in battle."
	if !g.fmContinueAsHeir() {
		t.Fatalf("expected fmContinueAsHeir() to succeed")
	}
	if g.Dead || g.P.Name != "Kaelen Legacy" || strings.EqualFold(g.P.Name, parentName) {
		t.Fatalf("expected player to continue as Kaelen Legacy, got Dead=%v Name=%q", g.Dead, g.P.Name)
	}
	if len(g.hf().Legends) == 0 || g.hf().Legends[len(g.hf().Legends)-1].Name != parentName {
		t.Fatalf("expected parent %q to be recorded in Hall of Fame legends", parentName)
	}
}
