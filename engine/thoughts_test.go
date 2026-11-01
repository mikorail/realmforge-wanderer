package engine

import "testing"

func thTick(g *Game, n int) {
	for i := 0; i < n; i++ {
		g.Turn++
		thTurn(g)
	}
}

func TestThCounts(t *testing.T) {
	if len(thDefs) < 40 || len(thTraits) < 20 {
		t.Fatalf("thoughts %d traits %d", len(thDefs), len(thTraits))
	}
	g := newGame(t, nil)
	seen := map[string]bool{}
	for _, d := range thDefs {
		if seen[d.ID] || d.Flav == "" || d.title(g) == "" {
			t.Fatalf("bad thought %s", d.ID)
		}
		seen[d.ID] = true
		_ = thFill(g, d.Flav)
	}
}

func TestThKillDiscoversAndCabinetFlow(t *testing.T) {
	g := newGame(t, nil)
	clearAround(g)
	before := g.P.Attr
	f := placeFoe(g, "rat", 1)
	g.kill(f, g.P)
	if !thFound(g, "blood_on_blade") {
		t.Fatal("kill did not find thought")
	}
	if msg := thStart(g, "blood_on_blade"); msg == "" || thSlotOf(g, "blood_on_blade") == nil {
		t.Fatal("start failed")
	}
	if g.P.Attr[INT] != before[INT]-1 {
		t.Fatalf("no researching penalty: %v vs %v", g.P.Attr, before)
	}
	thTick(g, 400)
	if !thDone(g, "blood_on_blade") {
		t.Fatal("not internalised")
	}
	if g.P.Attr[STR] != before[STR]+1 || g.P.Attr[INT] != before[INT] {
		t.Fatalf("effect wrong: %v vs %v", g.P.Attr, before)
	}
	g.Save()
	if !thForget(g, "blood_on_blade") || g.P.Attr != before {
		t.Fatalf("forget did not revert: %v vs %v", g.P.Attr, before)
	}
}

func TestThSlotsLimit(t *testing.T) {
	g := newGame(t, nil)
	n := thSlotCount(g)
	for _, d := range thDefs[:n+2] {
		thFind(g, d.ID)
		thStart(g, d.ID)
	}
	if len(thSt(g).Slots) != n {
		t.Fatalf("slots %d want %d", len(thSt(g).Slots), n)
	}
	thForget(g, thSt(g).Slots[0].ID)
	if len(thSt(g).Slots) != n-1 {
		t.Fatal("forget did not free slot")
	}
}

func TestThTraitsAndRep(t *testing.T) {
	g := newGame(t, nil)
	s := thSt(g)
	s.C["kills"] = 50
	s.C["prayers"] = 9
	thEarn(g)
	thSync(g)
	if !thHasTrait(g, "brutal") || !thHasTrait(g, "pious") {
		t.Fatal("traits not earned")
	}
	pr := &Actor{Role: "priest", Name: "Ansel"}
	if r := thTraitRep(g, pr); r != 0 {
		t.Fatalf("priest rep %d", r) // pious+faith, brutal-violent
	}
	if r := thTraitRep(g, &Actor{Role: "villager", Name: "V"}); r >= 0 {
		t.Fatalf("villager rep %d", r)
	}
}

func TestThComposureAndStats(t *testing.T) {
	g := newGame(t, nil)
	clearAround(g)
	thSync(g)
	start := thComposure(g)
	f := placeFoe(g, "horror", 5)
	f.AI = "still"
	g.recompute()
	thTick(g, 10)
	if thComposure(g) >= start {
		t.Fatalf("composure %d did not drop from %d", thComposure(g), start)
	}
	if thPerception(g) < 1 || thFame(g) < 0 {
		t.Fatal("stats")
	}
	g.deed("quest", "x", 0, 0)
	thSync(g)
	if thFame(g) < 8 {
		t.Fatal("fame from deed")
	}
}

func TestThScreenAndTalk(t *testing.T) {
	g := newGame(t, nil)
	thFind(g, "old_dust")
	g.Key("J")
	if g.Mode != "cabinet" {
		t.Fatalf("mode %s", g.Mode)
	}
	g.Key("Enter")
	g.Render()
	g.Key("Tab")
	g.Render()
	g.Key("Escape")
	thFind(g, "doctrine_doubt")
	thSlotOnly(g, "doctrine_doubt")
	for _, a := range g.Cur.Actors {
		if a.Role == "priest" || a.Role == "merchant" {
			g.openTalk(a)
			thTalk(g, a)
			break
		}
	}
}

func thSlotOnly(g *Game, id string) {
	thSt(g).Slots = append(thSt(g).Slots, thSlot{ID: id})
	thSync(g)
}
