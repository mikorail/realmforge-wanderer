package engine

import (
	"strings"
	"testing"
)

func TestAttireEverySphereSlot(t *testing.T) {
	g := newGame(t, nil)
	r := NewRng("attire")
	for sp := range attireTable {
		for _, sl := range attireSlots {
			for _, tier := range []int{0, 4} {
				it := g.MakeAttire(sp, sl, tier, r)
				if it.Name == "" || it.Region != sp || it.Kind != "armor" || it.Slot != sl {
					t.Fatalf("bad attire %s/%s: %+v", sp, sl, it)
				}
			}
		}
	}
	if len(attireTable) != 22 {
		t.Fatalf("want 22 spheres, got %d", len(attireTable))
	}
}

func TestAttireStockAndAffinity(t *testing.T) {
	g := newGame(t, nil)
	r := NewRng("tailor")
	bd := &g.W.Burgs[0]
	tl := g.makeNPC("villager", bd, g.W.cultByID[bd.Culture], r)
	g.stockTailor(tl, bd, r)
	if len(tl.Inv) == 0 {
		t.Fatal("tailor has no stock")
	}
	// fake realm of a known sphere
	cu := &CultureData{ID: 9991, Sphere: "steppe"}
	st := &StateData{ID: 9991, Culture: 9991}
	g.W.cultByID[9991], g.W.stateByID[9991] = cu, st
	defer delete(g.W.cultByID, 9991)
	defer delete(g.W.stateByID, 9991)
	a := &Actor{State: 9991}
	if g.attireAffinity(a) != 0 || g.attireRemark(a) != "" {
		t.Fatal("expected neutral")
	}
	for _, sl := range []string{"body", "head", "back"} {
		it := g.MakeAttire("steppe", sl, 1, r)
		g.P.PartFor(sl).Item = it
	}
	if g.attireAffinity(a) != 150 || g.attireRemark(a) == "" {
		t.Fatalf("affinity %d remark %q", g.attireAffinity(a), g.attireRemark(a))
	}
	foe := &StateData{ID: 9992, Culture: 9991, Wars: nil}
	_ = foe
	g.P.PartFor("feet").Item = g.MakeAttire("norse", "feet", 1, r)
	st.Wars = []int{9993}
	g.W.cultByID[9993] = &CultureData{ID: 9993, Sphere: "norse"}
	g.W.stateByID[9993] = &StateData{ID: 9993, Culture: 9993}
	defer delete(g.W.cultByID, 9993)
	defer delete(g.W.stateByID, 9993)
	if g.attireAffinity(a) != 110 {
		t.Fatalf("war penalty: %d", g.attireAffinity(a))
	}
}

func TestAttireTransmog(t *testing.T) {
	g := newGame(t, nil)
	r := NewRng("mog")
	tl := &Actor{ID: 777, Level: 3}
	g.Talk = tl
	eq := g.MakeAttire("sinic", "body", 2, r)
	g.P.PartFor("body").Item = eq
	src := g.MakeAttire("norse", "body", 2, r)
	g.P.Inv = append(g.P.Inv, src)
	g.P.Gold = 1000
	g.tailorMenu(tl)
	for _, o := range g.Opts {
		if strings.Contains(o.Text, eq.Name) {
			o.Do()
			break
		}
	}
	var pick *Option
	for i := range g.Opts {
		if len(g.Opts[i].Text) > 9 && g.Opts[i].Text[:9] == "Look like" {
			pick = &g.Opts[i]
		}
	}
	if pick == nil {
		t.Fatal("no source option")
	}
	pick.Do()
	if eq.Look != src.Name || eq.ShownName() != src.Name || g.P.Gold >= 1000 || tl.Gold == 0 {
		t.Fatalf("look %q gold %d", eq.Look, g.P.Gold)
	}
	g.clearLook(eq)
	if eq.ShownName() != eq.Name {
		t.Fatal("clearLook failed")
	}
}
