package engine

import "testing"

func TestCodexBestiaryRecords(t *testing.T) {
	g := newGame(t, nil)
	clearAround(g)
	f := placeFoe(g, "rat", 1)
	if !g.Hostile(f) || !g.visible(f.X, f.Y) {
		t.Skip("foe not hostile/visible")
	}
	codexTurn(g)
	c := g.codex()
	b := c.Beasts["rat"]
	if b == nil || b.Seen != 1 {
		t.Fatalf("rat not seen: %+v", b)
	}
	codexTurn(g)
	if b.Seen != 1 {
		t.Errorf("recounted same creature")
	}
	for i := 0; i < 3; i++ {
		f := placeFoe(g, "rat", 1)
		g.kill(f, g.P)
	}
	if b.Kills != 3 {
		t.Errorf("kills = %d", b.Kills)
	}
	out := g.codexLines(3, 100)
	if len(out) < 4 {
		t.Errorf("bestiary lines: %v", out)
	}
}

func TestCodexAtlas(t *testing.T) {
	g := newGame(t, nil)
	codexEnter(g, g.Cur)
	c := g.codex()
	if len(c.Places) == 0 {
		t.Fatal("no place recorded for starting town")
	}
	if len(g.codexLines(4, 100)) < 3 {
		t.Error("atlas empty")
	}
}

func TestCodexRandomBooks(t *testing.T) {
	g := newGame(t, nil)
	p := g.W.Idx(g.WX, g.WY)
	for k := 0; k < 6; k++ {
		b := g.cxBookOfKind(k, NewRng("bk"), p)
		if b == nil || b.Name == "" || len(b.Text) < 20 {
			t.Errorf("kind %d: %+v", k, b)
		}
		t.Logf("%d: %s (%s)", k, b.Name, cxBookKind(b.Name))
	}
	if g.RandomBook(g.R, p) == nil {
		t.Error("RandomBook nil")
	}
}

func TestCodexJournalAndSave(t *testing.T) {
	g := newGame(t, nil)
	b := g.cxBookOfKind(1, g.R, g.W.Idx(g.WX, g.WY))
	g.give(b)
	g.Read["book:"+b.Name] = true
	codexEnter(g, g.Cur)
	g.cxScanBooks()
	g.Mode = "journal"
	for tab := 0; tab < len(codexTabNames); tab++ {
		g.Tab = tab
		g.Render()
		g.Key("PageDown")
	}
	g.Key("Escape")
	if g.Mode == "journal" {
		t.Error("escape did not close")
	}
	s := g.Save()
	g2, err := LoadGame(g.W, s)
	if err != nil {
		t.Fatal(err)
	}
	c := g2.codex()
	if len(c.Places) == 0 || len(c.Books) == 0 || !c.Books[b.Name].Read {
		t.Errorf("state lost: %d places %d books", len(c.Places), len(c.Books))
	}
}
