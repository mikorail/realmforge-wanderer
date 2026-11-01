package engine

import "testing"

func plFindRole(g *Game, role string) *Actor {
	for i := range g.W.Burgs {
		b := &g.W.Burgs[i]
		if !b.Alive || b.Layer != "surface" || g.W.cityTier[b.ID] < 1 {
			continue
		}
		cp := g.cityPlanFor(b)
		for s := range cp.districts {
			zn := g.GetZone(b.P%g.W.W, b.P/g.W.W, s[0], s[1], 0)
			for _, a := range zn.Actors {
				if a.Role == role {
					return a
				}
			}
		}
	}
	return nil
}

func TestPlSaloonPlacedAndApplies(t *testing.T) {
	g := newGame(t, nil)
	a := plFindRole(g, "modder")
	if a == nil {
		t.Fatal("no saloon in any tier>=1 city")
	}
	it := g.MakeWeapon("longblade", 2, g.R)
	g.give(it)
	pv, gold := it.PV, g.P.Gold
	g.P.Gold = 5000
	g.Talk = a
	plSaloon(g, a, "hi")
	plPickMod(g, a, it)
	pickOpt(t, g, "sharp")
	if it.PV != pv+2 || g.P.Gold >= 5000 || it.Extra["mod:sharp"] != 1 {
		t.Fatalf("mod not applied: pv %d gold %d (was %d)", it.PV, g.P.Gold, gold)
	}
	plPickStrip(g, a, it)
	pickOpt(t, g, "sharp")
	if it.PV != pv || it.Extra["mod:sharp"] != 0 {
		t.Fatal("mod not stripped")
	}
	g.plPay(a, 0)
	plExtraSlot(g, a, it)
	if plSlots(it) != 2 {
		t.Fatalf("slots %d", plSlots(it))
	}
}

func TestPlHouseSellsAndRefuses(t *testing.T) {
	g := newGame(t, map[string]int{"flames": 1})
	a := plFindRole(g, "geneticist")
	if a == nil {
		t.Skip("no great-city slums/academy in this world")
	}
	g.Talk = a
	g.P.Gold = 9000
	id := plOffers(g, a)[0]
	g.MP = 0
	plBuyList(g, a, false)
	pickOpt(t, g, g.Sk.PowerName(id))
	if g.mutLevel(id) < 1 || g.P.Gold >= 9000 {
		t.Fatalf("not sold: lv %d gold %d", g.mutLevel(id), g.P.Gold)
	}
	// defect removal
	g.P.Powers["glutton"] = 1
	plDefectList(g, a)
	pickOpt(t, g, "Glutton")
	if g.P.Pow("glutton") != 0 {
		t.Fatal("defect not removed")
	}
	g.Genotype = "truekin"
	plHouse(g, a, "")
	if hasOpt(g, "Buy a mutation") {
		t.Fatal("True Kin offered mutations")
	}
}

func TestPlHideoutAndPurge(t *testing.T) {
	g := newGame(t, nil)
	g.P.Gold = 1000
	a := &Actor{Role: "geneticist", Faction: "pl"}
	g.Talk = a
	plHouse(g, a, "")
	pickOpt(t, g, "Purge")
	if !g.plGlimmerPurged() {
		t.Fatal("not purged")
	}
}
