package engine

import (
	"fmt"
	"testing"
)

func qcHomeElder(g *Game) *Actor {
	for _, a := range g.Cur.Actors {
		if a.Role == "elder" {
			return a
		}
	}
	return nil
}

func TestQcSpineBuildsAndAdvances(t *testing.T) {
	g := newGame(t, nil)
	ch := g.qcBuildSpine()
	for _, c := range ch {
		t.Logf("%s: %s", c.Kind, c.Title)
	}
	if len(ch) < 5 || len(ch) > 7 {
		t.Fatalf("chapters = %d", len(ch))
	}
	for _, c := range ch {
		t.Logf("%s: %s", c.Kind, c.Title)
	}
	el := qcHomeElder(g)
	if el == nil {
		t.Skip("no elder")
	}
	g.Talk = el
	g.qcStart(el)
	if g.qcChapter() == "" {
		t.Fatal("no chapter after start")
	}
	// walk the chapters by satisfying each condition
	st := g.qcSpine()
	for guard := 0; guard < 10 && !st.Finished; guard++ {
		c := g.qcCur()
		switch c.Kind {
		case "visit":
			g.Read[c.Key+"1"] = true
			g.qcPoll()
			g.Cur = g.GetZone(c.P%g.W.W, c.P/g.W.W, 1, 1, 0)
			g.qcPoll()
		case "talk":
			el2 := *el
			el2.Role = "elder"
			g.Cur.Burg = c.Burg
			for _, o := range qcTalk(g, &el2) {
				if o.Text == "{{W|Ask about the old tale.}}" {
					o.Do()
				}
			}
		case "boss", "depth":
			g.Cur = g.GetZone(c.P%g.W.W, c.P/g.W.W, 1, 1, 0)
			qcKill(g, &Actor{Boss: true, X: 5, Y: 5}, g.P)
			g.qcPoll()
		case "relic":
			g.give(&Item{ID: g.newID(), Name: "relic", SimArt: c.ArtID})
			g.qcPoll()
		case "slay":
			g.Slain[c.SimMon] = true
			g.qcPoll()
		case "return":
			g.Cur = g.GetZone(g.StartX, g.StartY, 1, 1, 0)
			g.qcFinish(el, "gave")
		}
	}
	if !st.Finished {
		t.Fatalf("spine stuck at step %d/%d (%s)", st.Step, len(st.Chaps), g.qcCur().Kind)
	}
	if !st.Flags["gave"] {
		t.Error("choice not stored")
	}
}

func TestQcSaveLoadSpine(t *testing.T) {
	g := newGame(t, nil)
	if el := qcHomeElder(g); el != nil {
		g.Talk = el
		g.qcStart(el)
	}
	g2, err := LoadGame(g.W, g.Save())
	if err != nil {
		t.Fatal(err)
	}
	if g2.qcChapter() != g.qcChapter() {
		t.Errorf("chapter %q vs %q", g2.qcChapter(), g.qcChapter())
	}
}

func TestQcVillageQuests(t *testing.T) {
	g := newGame(t, nil)
	el := qcHomeElder(g)
	if el == nil {
		t.Skip("no elder")
	}
	g.Talk = el
	made := 0
	for _, k := range []string{"escort", "inscription", "peace", "rescue", "stolen"} {
		g2 := newGame(t, nil)
		e2 := qcHomeElder(g2)
		q := g2.qcMakeVQ(k, e2, NewRng("t"+k))
		if q != nil {
			made++
			t.Logf("%s: %s", k, q.Title)
		}
	}
	if made < 3 {
		t.Errorf("only %d village quest kinds feasible", made)
	}
	g.qcOfferVQ(el)
	if len(g.qcSpine().VQ) != 1 {
		t.Fatal("no quest offered")
	}
}

func TestQcRescueAndStolenPlace(t *testing.T) {
	g := newGame(t, nil)
	el := qcHomeElder(g)
	for _, k := range []string{"rescue", "stolen"} {
		q := g.qcMakeVQ(k, el, NewRng(k))
		if q == nil {
			t.Logf("%s infeasible", k)
			continue
		}
		q.ID, q.Burg, q.Faction = g.newID(), g.Cur.Burg, el.Faction
		g.qcSpine().VQ = append(g.qcSpine().VQ, q)
		var wx, wy, zx, zy, z int
		if _, err := sscanKey(q.Target, &wx, &wy, &zx, &zy, &z); err != nil {
			t.Fatal(err)
		}
		zn := g.GetZone(wx, wy, zx, zy, z)
		g.qcPlaceVQ(q, zn)
		if k == "rescue" && q.A == 0 || k == "stolen" && len(zn.Items) == 0 {
			t.Errorf("%s not placed", k)
		}
		if k == "stolen" {
			for _, it := range zn.Items {
				if it.Quest == q.ID {
					g.give(it)
				}
			}
			g.qcVQPoll()
			if q.Stage != 1 {
				t.Error("goods in pack should make the task ready")
			}
		}
	}
}

func TestQcTiers(t *testing.T) {
	g := newGame(t, nil)
	for z := -3; z < 12; z++ {
		tr := g.qcTier(g.StartX, g.StartY, z)
		if tr < 1 || tr > 8 {
			t.Fatalf("tier %d", tr)
		}
	}
	if g.qcTier(g.StartX, g.StartY, 12) <= g.qcTier(g.StartX, g.StartY, 0) {
		t.Error("depth should raise the tier")
	}
	a := g.Spawn("rat", 1, g.R)
	qcRescale(a, 10)
	if a.Level != 10 || a.MaxHP < 30 {
		t.Errorf("rescale: %+v", a)
	}
}

func TestQcBiomesDeterministic(t *testing.T) {
	g := newGame(t, nil)
	n := 0
	for y := 0; y < g.W.H; y++ {
		for x := 0; x < g.W.W; x++ {
			if b := g.qcBiomeAt(x, y); b >= 0 {
				n++
				if g.qcBiomeAt(x, y) != b || g.qcBiomeName(x, y) == "" {
					t.Fatal("not stable")
				}
			}
		}
	}
	if n == 0 {
		t.Error("no special biomes in the world")
	}
}

func TestQcModes(t *testing.T) {
	g := newGame(t, nil)
	if g.qcKillXP(50) != 50 || g.qcXP(50) != 50 {
		t.Error("classic changes xp")
	}
	g.GameMode = "wander"
	if g.qcKillXP(50) != 0 || g.qcXP(50) != 100 {
		t.Error("wander xp")
	}
	g2 := NewGame(world(t, "ember-crown"), CharSpec{Name: "W", Species: "human", Calling: "warrior", Attr: [6]int{18, 16, 16, 14, 14, 14}, GameMode: "wander"})
	for f := range creatureFactions {
		if g2.Rep[f] != 0 {
			t.Fatalf("%s rep %d", f, g2.Rep[f])
		}
	}
}

func TestQcRoleplayRestore(t *testing.T) {
	g := NewGame(world(t, "ember-crown"), CharSpec{Name: "R", Species: "human", Calling: "warrior", Attr: [6]int{18, 16, 16, 14, 14, 14}, GameMode: "roleplay"})
	if g.qcMode().Check == "" {
		t.Fatal("no checkpoint on entering the home town")
	}
	g.P.Gold = 777
	g.hurt(g.P, 9999, nil, "test")
	g.qcOnDeath()
	if g.Mode != "qcrestore" {
		t.Fatalf("mode %s", g.Mode)
	}
	g.qcRestore()
	if g.Dead || g.Mode != "main" || g.P.HP <= 0 {
		t.Fatalf("not restored: dead=%v mode=%s hp=%d", g.Dead, g.Mode, g.P.HP)
	}
	if g.qcMode().Check == "" {
		t.Error("checkpoint lost after restore")
	}
	c := newGame(t, nil)
	c.hurt(c.P, 9999, nil, "test")
	if c.qcOnDeath() {
		t.Error("classic must not offer restore")
	}
}

func TestQcHazards(t *testing.T) {
	g := newGame(t, nil)
	clearAround(g)
	g.Cur.Set(g.P.X+1, g.P.Y, TLava)
	hp := g.P.HP
	g.Turn = 3
	qcHazardTick(g)
	if g.P.HP >= hp {
		t.Error("lava heat did nothing")
	}
}

func sscanKey(k string, a ...*int) (int, error) {
	return fmt.Sscanf(k, "%d.%d.%d.%d.%d", a[0], a[1], a[2], a[3], a[4])
}
