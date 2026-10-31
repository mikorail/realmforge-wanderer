package engine

import "testing"

func TestInvCapacity(t *testing.T) {
	g := newGame(t, nil)
	c0 := g.capacity()
	g.P.Attr[STR] += 4
	if g.capacity() != c0+12 {
		t.Fatalf("STR: %d -> %d", c0, g.capacity())
	}
	c1 := g.capacity()
	old := 0
	if p := g.P.PartFor("back"); p != nil && p.Item != nil && p.Item.Extra != nil {
		old = p.Item.Extra["cap"] // the starting kit's pack
	}
	bp := g.MakeBackpack(5)
	if bp.Extra["cap"] != 60 || g.MakeBackpack(0).Extra["cap"] != 15 {
		t.Fatalf("pack caps")
	}
	g.give(bp)
	if !g.equip(bp) {
		t.Fatal("equip pack")
	}
	if g.capacity() != c1-old+60 {
		t.Fatalf("pack: %d -> %d", c1, g.capacity())
	}
}

func TestInvOverloaded(t *testing.T) {
	g := newGame(t, nil)
	if g.encumbrance() != 0 || !g.canMoveLoad() || g.encumbranceQuick() != 0 {
		t.Fatal("starts unburdened")
	}
	c, cp := g.burden()
	g.give(&Item{ID: g.newID(), Kind: "trophy", Name: "anvil", Weight: cp - c + 1})
	if g.encumbrance() != 1 || g.encumbranceQuick() >= 0 || !g.canMoveLoad() {
		t.Fatalf("burdened expected, got %d", g.encumbrance())
	}
	g.give(&Item{ID: g.newID(), Kind: "trophy", Name: "boulder", Weight: cp})
	if g.encumbrance() != 2 || g.canMoveLoad() {
		t.Fatalf("overloaded expected, got %d", g.encumbrance())
	}
}

func TestInvTabAndSort(t *testing.T) {
	g := newGame(t, nil)
	defer func() { invTab, invSort = 0, 0 }()
	g.give(g.MakeWeapon("axe", 2, g.R))
	g.give(g.MakeWeapon("axe", 2, g.R))
	invTab = 1
	l := g.invVisible()
	if len(l) == 0 {
		t.Fatal("no weapons")
	}
	for _, it := range l {
		if invCat(it) != 1 {
			t.Fatalf("non-weapon %s in weapons tab", it.Kind)
		}
	}
	invTab = 0
	for s := 0; s < len(invSorts); s++ {
		invSort = s
		a, b := g.invVisible(), g.invVisible()
		for i := range a {
			if a[i] != b[i] {
				t.Fatalf("sort %d unstable", s)
			}
		}
	}
	invSort = 2
	l = g.invVisible()
	for i := 1; i < len(l); i++ {
		if itemWeight(l[i-1]) < itemWeight(l[i]) {
			t.Fatal("weight order")
		}
	}
}

func TestInvRender(t *testing.T) {
	g := newGame(t, nil)
	defer func() { invTab, invSort = 0, 0 }()
	g.Mode = "inv"
	for _, k := range []string{"Tab", "Tab", "ArrowDown", "s", "ArrowLeft", "a"} {
		g.Key(k)
		g.Render()
	}
	g.Mode = "inv"
	invTab = 0
	s := g.Render()
	var txt []rune
	txt = append(txt, s.G...)
	if !contains(string(txt), "Weight") {
		t.Fatal("no Weight header")
	}
	g.Key("Escape")
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
