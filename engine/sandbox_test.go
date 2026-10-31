package engine

import "testing"

func TestSandboxMenuAndActions(t *testing.T) {
	g := newGame(t, nil)
	clearAround(g)
	g.Key("F2")
	if g.Mode != "sandbox" {
		t.Fatalf("mode %q", g.Mode)
	}
	g.Render()
	if g.Sandboxed() {
		t.Fatal("opening alone should not mark sandbox")
	}
	for _, k := range []string{"b", "f", "h", "k", "j"} {
		g.Key(k)
		g.Render()
		if g.Mode != "sandbox" {
			t.Fatalf("key %s left mode %q", k, g.Mode)
		}
	}
	if !g.Sandboxed() {
		t.Fatal("not sandboxed")
	}
	for _, k := range []string{"c", "d", "e", "g"} {
		g.Key(k)
		g.Render()
		g.Key("Escape")
		g.Render()
	}
	g.Key("Escape")
	if g.Mode != "main" {
		t.Fatalf("mode %q", g.Mode)
	}
	g.Key("~")
	if g.Mode != "sandbox" {
		t.Fatal("~ should open")
	}
	g.Key("Escape")
}

func TestSandboxGodSurvivesBear(t *testing.T) {
	g := newGame(t, nil)
	clearAround(g)
	g.Key("F2")
	g.Key("a")
	g.Key("Escape")
	if !g.sb().God {
		t.Fatal("god off")
	}
	placeFoe(g, "bear", 15)
	for i := 0; i < 50; i++ {
		g.Key(".")
	}
	if g.Dead || g.P.HP <= 0 {
		t.Fatal("died in god mode")
	}
}

func TestSandboxFreeze(t *testing.T) {
	g := newGame(t, nil)
	clearAround(g)
	b := placeFoe(g, "bear", 8)
	hp := g.P.HP
	g.Key("F2")
	g.Key("i")
	g.Key("Escape")
	for i := 0; i < 20; i++ {
		g.Key(".")
	}
	if g.P.HP < hp || b.HP <= 0 {
		t.Fatalf("frozen bear acted: hp %d->%d", hp, g.P.HP)
	}
	g.Key("F2")
	g.Key("i")
	if b.Energy < -1000000 {
		t.Fatal("not unfrozen")
	}
}

func TestSandboxTeleportSpawnItem(t *testing.T) {
	g := newGame(t, nil)
	clearAround(g)
	n := len(g.Cur.Actors)
	g.Key("F2")
	g.Key("d")
	g.Key("a")
	g.Key("Enter")
	if len(g.Cur.Actors) != n+1 || g.Mode != "main" {
		t.Fatalf("spawn failed: %d -> %d mode %s", n, len(g.Cur.Actors), g.Mode)
	}
	for ri := range rarityOrder {
		g.Key("F2")
		g.Key("e")
		g.Key("a")
		g.Key(string(rune('a' + ri)))
	}
	if len(g.P.Inv) == 0 {
		t.Fatal("no items")
	}
	key := g.Cur.Key
	g.Key("F2")
	g.Key("c")
	st := g.sb()
	for dx := 1; dx < g.W.W; dx++ {
		x := (g.WX + dx*3) % g.W.W
		if p := g.W.Idx(x, g.WY); !g.W.IsWater(p) {
			st.cx, st.cy = x, g.WY
			break
		}
	}
	g.Render()
	g.Key("Enter")
	g.Render()
	g.Key("a")
	if g.Cur.Key == key {
		t.Fatal("teleport did not change zone")
	}
	for _, k := range []string{"l", "m", "n"} {
		g.Key("F2")
		g.Key(k)
		g.Render()
		g.Key("Escape")
	}
	if !g.Sandboxed() {
		t.Fatal("not sandboxed")
	}
}

func TestSandboxSaveStores(t *testing.T) {
	g := newGame(t, nil)
	g.Key("F2")
	g.Key("h")
	g.flushExt()
	if len(g.Store["sandbox"]) == 0 {
		t.Fatal("state not stored")
	}
}
