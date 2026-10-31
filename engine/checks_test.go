package engine

import "testing"

func TestCheckChanceMath(t *testing.T) {
	if got := chanceFor(0, 7); got != 21.0/36 {
		t.Fatalf("mod0 dc7 = %v", got)
	}
	if chanceFor(0, 2) != 1 || chanceFor(0, 13) != 0 {
		t.Fatal("bounds")
	}
	prev := 2.0
	for dc := 2; dc <= 16; dc++ {
		c := chanceFor(0, dc)
		if c > prev {
			t.Fatalf("not monotonic at dc %d", dc)
		}
		prev = c
	}
	for mod := -3; mod < 6; mod++ {
		if chanceFor(mod+1, 10) < chanceFor(mod, 10) {
			t.Fatal("mod not monotonic")
		}
	}
}

func checkNPC(g *Game, role string) *Actor {
	a := g.Spawn("human", 2, g.R)
	a.Role, a.Name, a.Faction = role, "Tarn", g.Cur.Faction
	a.Inv = nil
	return a
}

func TestCheckOptionsPerRole(t *testing.T) {
	g := newGame(t, nil)
	for _, role := range []string{"merchant", "smith", "guard", "elder", "notable", "priest", "villager"} {
		a := checkNPC(g, role)
		opts := g.checkOptions(a)
		if len(opts) == 0 || len(opts) > 4 {
			t.Fatalf("%s: %d options", role, len(opts))
		}
		for _, o := range opts {
			if o.Text == "" || o.Do == nil {
				t.Fatalf("%s: bad option", role)
			}
		}
	}
}

func TestCheckExecuteAndNoRetry(t *testing.T) {
	g := newGame(t, nil)
	for _, role := range []string{"merchant", "guard", "elder", "priest", "villager"} {
		for i := 0; i < 6; i++ {
			a := checkNPC(g, role)
			g.Talk = a
			a.Angry = i%2 == 0
			opts := g.checkOptions(a)
			first := opts[0].Text
			opts[0].Do()
			for _, o := range g.checkOptions(a) {
				if o.Text == first {
					t.Fatalf("%s: used check offered again", role)
				}
			}
		}
	}
}

func TestInnerVoices(t *testing.T) {
	g := newGame(t, nil)
	a := checkNPC(g, "elder")
	a.Title, a.House, a.Caste = "high elder", "Vael", "noble"
	g.P.Attr[INT] = 20
	g.P.Attr[WIL] = 20
	vs := g.innerVoices(a)
	if len(vs) > 2 {
		t.Fatalf("too many voices: %d", len(vs))
	}
	for _, v := range vs {
		if v == "" {
			t.Fatal("empty voice")
		}
	}
}
