package engine

import "testing"

func cuCalm(g *Game) func() {
	bd := g.W.burgByID[g.Cur.Burg]
	port, ban := bd.Port, bd.Bandit
	bd.Port, bd.Bandit = false, false
	var saved []struct{ i, v int }
	for dy := -2; dy <= 2; dy++ {
		for dx := -2; dx <= 2; dx++ {
			if x, y := g.Cur.WX+dx, g.Cur.WY+dy; g.W.In(x, y) {
				i := g.W.Idx(x, y)
				if g.W.State[i] != bd.State {
					saved = append(saved, struct{ i, v int }{i, g.W.State[i]})
					g.W.State[i] = bd.State
				}
			}
		}
	}
	old := g.Cur.Name
	g.Cur.Name = "Quiet Square"
	return func() {
		bd.Port, bd.Bandit = port, ban
		for _, s := range saved {
			g.W.State[s.i] = s.v
		}
		g.Cur.Name = old
	}
}

func TestCuDefsDeterministicAndVaried(t *testing.T) {
	for _, name := range []string{"ember-crown", "neon-tide", "spore-mother"} {
		w := world(t, name)
		g := NewGame(w, CharSpec{Name: "T", Species: "human", Calling: "warrior", Attr: [6]int{18, 16, 16, 14, 14, 14}})
		h := NewGame(w, CharSpec{Name: "T", Species: "human", Calling: "warrior", Attr: [6]int{18, 16, 16, 14, 14, 14}})
		names := map[string]bool{}
		for _, s := range w.States {
			a, b := g.cuDef(s.ID), h.cuDef(s.ID)
			if a.Sing != b.Sing || a.Rate != b.Rate || a.Kind != b.Kind {
				t.Fatalf("%s: realm %d not deterministic", name, s.ID)
			}
			if a.Rate <= 0 {
				t.Fatalf("bad rate")
			}
			names[a.Kind+a.Sing] = true
		}
		if len(names) < 2 {
			t.Errorf("%s: only %d currency kinds", name, len(names))
		}
	}
}

func TestCuGoldSyncAndEarnSpend(t *testing.T) {
	g := newGame(t, nil)
	g.P.Gold = 0
	g.cuSync()
	g.cuSync()
	g.P.Gold += 100 // legacy call site
	g.cuSync()
	if g.cuTotal() < 98 || g.cuTotal() > 102 || g.P.Gold != g.cuTotal() {
		t.Fatalf("gold %d total %d", g.P.Gold, g.cuTotal())
	}
	if g.cu().Held[g.cuLocal().ID] == 0 {
		t.Error("gain not held in local money")
	}
	g.cuEarn(50)
	before := g.P.Gold
	if !g.cuSpend(60) {
		t.Fatal("cannot spend 60")
	}
	if d := before - g.P.Gold; d < 58 || d > 64 {
		t.Errorf("spent %d", d)
	}
	g.P.Gold -= 10 // legacy loss
	g.cuSync()
	if g.P.Gold != g.cuTotal() {
		t.Error("gold out of sync after legacy loss")
	}
	if g.cuSpend(100000) {
		t.Error("overspend allowed")
	}
	if s := g.cuFormat(20); s == "" {
		t.Error("format empty")
	}
}

func TestCuForeignNeedsChanger(t *testing.T) {
	g := newGame(t, nil)
	defer cuCalm(g)()
	g.cuSync()
	st := g.cu()
	st.Held = map[string]int{}
	loc := g.cuLocal()
	other := 0
	for _, s := range g.W.States {
		if s.ID != g.cuRealm() && s.ID > 0 {
			other = s.ID
			break
		}
	}
	fd := g.cuDef(other)
	st.Held[fd.ID] = 500
	g.cuFinish()
	if ok, _ := g.cuForeign(); ok {
		t.Fatal("foreign money accepted in a quiet inland town")
	}
	if g.cuSpend(20) {
		t.Error("foreign coin spent with no changer")
	}
	ch := g.makeNPC("changer", g.W.burgByID[g.Cur.Burg], g.W.cultByID[g.W.burgByID[g.Cur.Burg].Culture], g.R)
	ch.X, ch.Y = g.P.X+2, g.P.Y
	g.Cur.Actors = append(g.Cur.Actors, ch)
	if !g.cuSpend(20) {
		t.Fatal("foreign coin refused with a changer present")
	}
	// exchange through talk
	opts := cuTalk(g, ch)
	if len(opts) < 2 {
		t.Fatalf("changer options: %d", len(opts))
	}
	before := g.cuUnits(fd.ID)
	opts[0].Do()
	if g.cuUnits(fd.ID) >= before || g.cuUnits(loc.ID) == 0 {
		t.Errorf("exchange did nothing: %d -> %d, local %d", before, g.cuUnits(fd.ID), g.cuUnits(loc.ID))
	}
	g.Mode = "main"
}

func TestCuSaveAndSide(t *testing.T) {
	g := newGame(t, nil)
	g.cuSync()
	g.cuEarn(77)
	if len(cuSide(g)) == 0 {
		t.Error("no sidebar line")
	}
	g.flushExt()
	h, err := LoadGame(g.W, g.Save())
	if err != nil {
		t.Fatal(err)
	}
	if h.cuWallet() != g.cuWallet() {
		t.Errorf("wallet %q != %q", h.cuWallet(), g.cuWallet())
	}
}

func TestCuChangerSpawns(t *testing.T) {
	g := newGame(t, nil)
	n := 0
	for i := range g.W.Burgs {
		bd := &g.W.Burgs[i]
		if !bd.Alive || g.W.cityTier[bd.ID] == 0 {
			continue
		}
		cp := g.cityPlanFor(bd)
		for s, k := range cp.districts {
			if k != "market" && k != "harbor" {
				continue
			}
			p := bd.P
			zn := g.GetZone(p%g.W.W, p/g.W.W, s[0], s[1], 0)
			found := false
			for _, a := range zn.Actors {
				found = found || a.Role == "changer"
			}
			if !found {
				t.Fatalf("no changer in %s", zn.Name)
			}
			n++
			if n >= 3 {
				return
			}
		}
	}
	if n == 0 {
		t.Skip("no market district in world")
	}
}
