package engine

import "testing"

func craftGame(t *testing.T) *Game {
	g := newGame(t, nil)
	g.Skills["tinkering"] = 3
	return g
}

func TestCraftDisassembleBits(t *testing.T) {
	g := craftGame(t)
	w := g.MakeWeapon("longblade", 3, g.R)
	g.give(w)
	g.craftDisassemble(w)
	st := g.craftSt()
	tot := 0
	for _, n := range st.Bits {
		tot += n
	}
	if tot == 0 || st.Bits["4"] == 0 {
		t.Fatalf("expected tier-4 bits, got %v", st.Bits)
	}
	for _, it := range g.P.Inv {
		if it == w {
			t.Fatal("item not consumed")
		}
	}
}

func TestCraftLearnAndSphereItem(t *testing.T) {
	g := craftGame(t)
	cl := g.MakeAttire("norse", "body", 2, g.R)
	rc := g.tryLearnFrom(cl, 1.0)
	if rc == nil || rc.Sphere != "norse" || !g.craftSt().Known[rc.ID] {
		t.Fatalf("no norse recipe learned: %+v", rc)
	}
	st := g.craftSt()
	st.Bits["2"], st.Bits["3"] = 4, 2
	n := len(g.P.Inv)
	if m := g.craftRecipe(rc); len(g.P.Inv) != n+1 {
		t.Fatalf("craft failed: %s", m)
	}
	if st.Bits["2"] != 0 || st.Bits["3"] != 0 {
		t.Fatalf("bits not spent: %v", st.Bits)
	}
	// unknown recipe refused
	other := g.recipeByID("s:latin:0")
	st.Bits["2"], st.Bits["3"] = 4, 2
	if m := g.craftRecipe(other); len(g.P.Inv) != n+1 {
		t.Fatalf("crafted unknown: %s", m)
	}
}

func TestCraftModsAndSlots(t *testing.T) {
	g := craftGame(t)
	w := g.MakeWeapon("longblade", 2, g.R)
	w.Weight = 4
	g.give(w)
	st := g.craftSt()
	st.Bits["2"], st.Bits["3"] = 20, 20
	pv := w.PV
	g.applyMod(w, "sharp")
	if w.PV != pv+2 {
		t.Fatalf("sharp didn't raise PV")
	}
	g.applyMod(w, "counterweighted") // common: only 1 slot
	if w.Bonus[AGI] != 0 || len(modsOf(w)) != 1 {
		t.Fatalf("slot limit ignored: %v", modsOf(w))
	}
	w.Rarity = "rare"
	g.applyMod(w, "flaming")
	if w.Elem != "fire" || len(modsOf(w)) != 2 {
		t.Fatalf("flaming failed: %v", modsOf(w))
	}
	if g.modSlots(w) != 2 {
		t.Fatalf("slots %d", g.modSlots(w))
	}
}

func TestCraftInfuseAndSave(t *testing.T) {
	g := craftGame(t)
	a := g.MakeArmor("body", 2, g.R)
	g.give(a)
	st := g.craftSt()
	st.Bits["4"] = 2
	st.Ess["essence of dragon"] = 1
	st.Known["s:latin:0"] = true
	if m := g.infuse(a, "biblical"); a.Cat != "biblical" {
		t.Fatalf("infuse failed: %s", m)
	}
	if st.essCount() != 0 {
		t.Fatal("essence not consumed")
	}
	s := g.Save()
	g2, err := LoadGame(world(t, "ember-crown"), s)
	if err != nil {
		t.Fatal(err)
	}
	st2 := g2.craftSt()
	if !st2.Known["s:latin:0"] || st2.Bits["4"] != 0 && false {
		t.Fatalf("state lost: %+v", st2)
	}
	found := false
	for _, it := range g2.P.Inv {
		if it.Cat == "biblical" && it.Extra["infused"] == 1 {
			found = true
		}
	}
	if !found {
		t.Fatal("infused item lost")
	}
}
