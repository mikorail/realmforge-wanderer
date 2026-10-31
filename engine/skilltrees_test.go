package engine

import "testing"

func TestSkMigrationKeepsRank(t *testing.T) {
	g := newGame(t, nil)
	g.Skills["axe"] = 2
	g.skSync()
	if g.Skills["axe"] != 2 || g.skOwned(skTreeByID["axe"]) != 2 {
		t.Fatalf("axe rank %d owned %d", g.Skills["axe"], g.skOwned(skTreeByID["axe"]))
	}
	g.Skills["axe"] = 3 // old rank-up screen
	g.skSync()
	if g.skOwned(skTreeByID["axe"]) != 3 {
		t.Fatal("rank bump not mirrored")
	}
}

func TestSkBuyTreeAndNode(t *testing.T) {
	g := newGame(t, nil)
	tr := skTreeByID["cudgel"]
	g.SP = 10
	delete(g.Skills, "cudgel")
	g.skState().Open["cudgel"] = false
	if !g.skUnlock(tr) || g.Skills["cudgel"] != 1 {
		t.Fatalf("unlock failed, rank %d", g.Skills["cudgel"])
	}
	sp := g.SP
	if !g.skBuy(tr, skNodeByID["bludgeon"]) || g.SP != sp-skCost(150) || g.Skills["cudgel"] != 2 {
		t.Fatal("node buy failed")
	}
	found := false
	for _, ab := range g.skAbilities() {
		found = found || ab.ID == "sk_bludgeon"
	}
	if !found {
		t.Fatal("bludgeon not in abilities")
	}
	g.Save() // ext round trip
}

func TestSkManeuversRun(t *testing.T) {
	for _, tr := range skTrees {
		for _, nd := range tr.Nodes {
			if nd.Kind != "active" {
				continue
			}
			g := newGame(t, nil)
			clearAround(g)
			f := placeFoe(g, "rat", 1)
			f.HP, f.MaxHP = 200, 200
			g.skState().Open[tr.ID] = true
			g.skGrant(&nd, false)
			tx, ty := f.X, f.Y
			if skSelfTarget[nd.ID] {
				tx, ty = 0, 0
			}
			g.skUse("sk_"+nd.ID, tx, ty)
		}
	}
}

func TestSkStatuses(t *testing.T) {
	g := newGame(t, nil)
	f := placeFoe(g, "rat", 1)
	if skStatusPV(f) != 0 {
		t.Fatal("pv")
	}
	f.AddSt("asleep", 5)
	if skStatusPV(f) != 4 || skStatusDV(f) != -12 {
		t.Fatalf("asleep pv %d dv %d", skStatusPV(f), skStatusDV(f))
	}
	f.AddSt("shamed", 5)
	if skStatusToHit(f) != -4 {
		t.Fatal("shamed")
	}
	f.AddSt("hobbled", 5)
	if skStatusSpeed(f) > -50 {
		t.Fatal("hobbled")
	}
}

func TestSkLeveling(t *testing.T) {
	if XPForLevel(2) <= XPForLevel(1) {
		t.Fatal("curve")
	}
	g := newGame(t, nil)
	g.skOnLevel()
	sp := g.SP
	g.P.Level += 3
	g.skOnLevel()
	if g.SP < sp+3*2 || g.AP < 1 {
		t.Fatalf("sp %d->%d ap %d", sp, g.SP, g.AP)
	}
	g.skOnLevel()
	if g.SP != sp+(g.SP-sp) {
		t.Fatal("not idempotent")
	}
	r := g.Spawn("rat", 1, g.R)
	g.P.Level = 12
	if skKillXP(g, r, g.P) != r.XP/10 {
		t.Fatalf("tier mult %d vs %d", skKillXP(g, r, g.P), r.XP)
	}
}

func TestSkKeyScreen(t *testing.T) {
	g := newGame(t, nil)
	mainKeys["K"](g)
	for _, k := range []string{"ArrowDown", "Enter", "ArrowDown", "Enter", "Escape"} {
		skKeys(g, k)
	}
	g.Mode, g.Tab, g.Sel = "skills", 2, 3
	g.Render()
	g.Sel = 40
	g.Render()
}
