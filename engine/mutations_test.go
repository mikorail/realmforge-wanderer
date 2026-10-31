package engine

import (
	"encoding/json"
	"testing"
)

func muGame(t *testing.T, spec CharSpec) *Game {
	t.Helper()
	spec.Name, spec.Species = "Tester", "human"
	if spec.Calling == "" {
		spec.Calling = "warrior"
	}
	spec.Attr = [6]int{18, 16, 16, 14, 14, 14}
	return NewGame(world(t, "ember-crown"), spec)
}

func TestMuCapFormula(t *testing.T) {
	for l, want := range map[int]int{1: 1, 2: 2, 5: 3, 10: 6, 18: 10, 40: 10} {
		if muCap(l) != want {
			t.Errorf("cap(%d)=%d want %d", l, muCap(l), want)
		}
	}
}

func TestMuSpendWithinCap(t *testing.T) {
	g := muGame(t, CharSpec{Powers: map[string]int{"flames": 1}})
	g.MP = 10
	if g.mutLevel("flames") != 1 {
		t.Fatal("flames level")
	}
	if ok, _ := g.mutRaise("flames"); ok { // level 1 cap is 1
		t.Fatal("raised past cap")
	}
	g.P.Level = 4
	if ok, msg := g.mutRaise("flames"); !ok || g.mutLevel("flames") != 2 || g.MP != 9 {
		t.Fatalf("raise: %v %s lv=%d mp=%d", ok, msg, g.mutLevel("flames"), g.MP)
	}
	if ok, _ := g.mutBuy("freeze"); !ok || g.MP != 5 || g.mutLevel("freeze") != 1 {
		t.Fatal("buy")
	}
	// raising through the character screen path is folded in by mutSync
	g.P.Powers["freeze"]++
	g.mutSync()
	if g.mutBase("freeze") != 2 {
		t.Fatalf("outside change lost: %d", g.mutBase("freeze"))
	}
}

func TestMuRapidAdvancement(t *testing.T) {
	g := muGame(t, CharSpec{Powers: map[string]int{"flames": 1, "psiblast": 1}})
	g.P.Level = 5
	muLevelPoll(g)
	if g.mu().RapidDue != 1 {
		t.Fatal("rapid not due at 5")
	}
	if ok, _ := g.mutRapid("psiblast"); ok {
		t.Fatal("rapid on mental")
	}
	if ok, _ := g.mutRapid("flames"); !ok || g.mutLevel("flames") != 4 {
		t.Fatalf("rapid: %d", g.mutLevel("flames"))
	}
	g.P.Level = 15
	muLevelPoll(g)
	if g.mu().RapidDue != 1 || !muRapidLevel(25) || muRapidLevel(10) {
		t.Fatal("rapid levels")
	}
}

func TestMuTrueKinNoMutations(t *testing.T) {
	g := muGame(t, CharSpec{Genotype: "truekin", Calling: "artifex", Powers: map[string]int{"flames": 2, "brittle": 1}})
	if len(g.P.Powers) != 0 || g.MP != 0 || g.Calling != "artifex" {
		t.Fatalf("truekin kept powers %v mp=%d calling=%s", g.P.Powers, g.MP, g.Calling)
	}
	if ok, _ := g.mutBuy("flames"); ok {
		t.Fatal("truekin bought a mutation")
	}
	g.gainXP(100000)
	muLevelPoll(g)
	if g.MP != 0 {
		t.Fatalf("truekin MP %d", g.MP)
	}
	if muCasteName("cyberpunk", "artifex") == "Artifex" {
		t.Fatal("caste not skinned")
	}
}

func TestMuMorphotypes(t *testing.T) {
	g := muGame(t, CharSpec{Powers: map[string]int{"flames": 1, "psiblast": 1}})
	mp := g.MP
	g.muNewGame(CharSpec{Morph: "chimera"})
	if g.mutLevel("psiblast") != 0 || g.mutLevel("flames") != 1 || g.MP <= mp {
		t.Fatalf("chimera: psi=%d flames=%d mp=%d/%d", g.mutLevel("psiblast"), g.mutLevel("flames"), g.MP, mp)
	}
	g.MP = 9
	if ok, _ := g.mutBuy("teleport"); ok {
		t.Fatal("chimera bought mental")
	}
	e := muGame(t, CharSpec{Powers: map[string]int{"flames": 1, "psiblast": 2}})
	e.muNewGame(CharSpec{Morph: "esper"})
	if e.mutLevel("flames") != 0 || e.mutLevel("psiblast") != 2 || muGlimmer(e) <= 2 {
		t.Fatal("esper")
	}
}

func TestMuGlimmerHunters(t *testing.T) {
	g := muGame(t, CharSpec{Powers: map[string]int{"psiblast": 3}})
	if muGlimmer(g) != 3 {
		t.Fatalf("glimmer %d", muGlimmer(g))
	}
	for _, id := range []string{"teleport", "beguile", "forcewall", "slumber", "arcanebolt", "sense", "clairvoyance"} {
		g.P.Powers[id] = 4
	}
	g.mutSync()
	if muGlimmer(g) < 20 {
		t.Fatalf("glimmer %d", muGlimmer(g))
	}
	g.P.Powers["psiblast"] = 40 // p capped at .9
	g.mutSync()
	zn := g.Cur
	zn.Burg = 0
	clearAround(g)
	found := false
	for turn := 0; turn < 40 && !found; turn++ {
		g.Turn = turn * 7
		muGlimmerEnter(g, zn)
		for _, a := range zn.Actors {
			if a.Has("hunter") {
				found = true
			}
		}
	}
	if !found {
		t.Fatal("no hunter spawned")
	}
	n := len(zn.Actors)
	muGlimmerEnter(g, zn)
	if len(zn.Actors) != n {
		t.Fatal("second spawn while hunters present")
	}
	if len(muGlimmerSide(g)) == 0 {
		t.Fatal("no sidebar line")
	}
}

func TestMuOffersDeterministicAndSave(t *testing.T) {
	a := muGame(t, CharSpec{Powers: map[string]int{"flames": 1}})
	b := muGame(t, CharSpec{Powers: map[string]int{"flames": 1}})
	if len(a.mu().Pending) != 3 || a.mu().Pending[0] != b.mu().Pending[0] {
		t.Fatal("offers not deterministic")
	}
	a.P.Level = 4
	a.mutRaise("flames")
	l, err := LoadGame(a.W, a.Save())
	if err != nil {
		t.Fatal(err)
	}
	if l.mutLevel("flames") != 2 {
		t.Fatalf("after load %d", l.mutLevel("flames"))
	}
}

func TestMuCharOptionsJSON(t *testing.T) {
	var o struct {
		Genotypes, Morphotypes, Castes, Modes, Callings []map[string]any
	}
	if err := json.Unmarshal([]byte(CharOptions(world(t, "ember-crown"))), &o); err != nil {
		t.Fatal(err)
	}
	if len(o.Genotypes) != 2 || len(o.Morphotypes) != 3 || len(o.Castes) != 12 || len(o.Modes) != 3 || len(o.Callings) != 7 {
		t.Fatalf("options %d %d %d %d %d", len(o.Genotypes), len(o.Morphotypes), len(o.Castes), len(o.Modes), len(o.Callings))
	}
}

func TestMuDefectsRepugnant(t *testing.T) {
	g := muGame(t, CharSpec{Powers: map[string]int{"repugnant": 1}})
	for k, v := range g.Rep {
		if k == "state:0" && v != -50 {
			t.Fatalf("rep %d", v)
		}
	}
}
