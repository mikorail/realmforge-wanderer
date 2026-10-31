package engine

import "testing"

func hintCount(g *Game) int {
	n := 0
	for _, m := range g.Msgs {
		if len(m.T) > 8 && m.T[:8] == "{{c|Tip:" {
			n++
		}
	}
	return n
}

func TestHintsFireOnce(t *testing.T) {
	g := newGame(t, nil)
	g.hintState().Shown = map[string]bool{}
	g.Msgs = nil
	hintEnter(g, g.Cur)
	if hintCount(g) != 1 {
		t.Fatalf("town tip expected, got %d", hintCount(g))
	}
	hintEnter(g, g.Cur)
	if hintCount(g) != 1 {
		t.Fatal("tip repeated")
	}
	g.P.HP = 1
	g.Turn++
	hintTurn(g)
	n := hintCount(g)
	hintTurn(g)
	if hintCount(g) != n {
		t.Fatal("turn tip repeated")
	}
}

func TestHintsRespectPref(t *testing.T) {
	g := newGame(t, nil)
	g.hintState().Shown = map[string]bool{}
	g.SetPref("hints", false)
	g.Msgs = nil
	hintEnter(g, g.Cur)
	g.P.HP = 1
	hintTurn(g)
	hintKill(g, g.P, nil)
	if hintCount(g) != 0 {
		t.Fatal("tips shown with hints off")
	}
	g.SetPref("hints", true)
	hintEnter(g, g.Cur)
	if hintCount(g) != 1 {
		t.Fatal("tip should show once hints back on")
	}
}
