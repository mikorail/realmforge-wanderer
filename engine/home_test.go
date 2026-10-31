package engine

import "testing"

func homeNPC(g *Game) *Actor {
	a := g.Spawn("villager", 1, g.R)
	a.Role = "elder"
	a.Name = "Elder Test"
	a.X, a.Y = g.P.X+1, g.P.Y
	g.Cur.Set(a.X, a.Y, TFloor)
	g.Cur.Actors = append(g.Cur.Actors, a)
	return a
}

func buyHouse(t *testing.T, g *Game) {
	t.Helper()
	a := homeNPC(g)
	g.P.Gold = 100000
	g.openTalk(a)
	for _, o := range g.Opts {
		if len(o.Text) > 10 && o.Text[:6] == "{{W|Bu" {
			o.Do()
			return
		}
	}
	t.Fatalf("no buy option")
}

func TestHomeBuyAndStash(t *testing.T) {
	g := newGame(t, nil)
	buyHouse(t, g)
	h := g.homeHere()
	if h == nil || g.Cur.At(h.SX, h.SY) != TShelf || g.Cur.Notes[h.SY*MW+h.SX] != "Your stash" {
		t.Fatalf("home not built: %+v", h)
	}
	if len(g.homeSt().Homes) != 1 {
		t.Fatal("expected 1 home")
	}
	g.Mode = "main"
	g.Key("H")
	if g.Mode != "stash" {
		t.Fatalf("mode %s", g.Mode)
	}
	n := len(g.P.Inv)
	name := g.P.Inv[0].Name
	g.Key("Enter")
	if len(g.P.Inv) != n-1 || len(g.homeSt().Stash) != 1 {
		t.Fatal("transfer failed")
	}
	g2, err := LoadGame(g.W, g.Save())
	if err != nil {
		t.Fatal(err)
	}
	st := g2.homeSt().Stash
	if len(st) != 1 || st[0].Name != name || len(g2.homeSt().Homes) != 1 {
		t.Fatalf("stash lost after load: %+v", st)
	}
	g.Key("Tab")
	g.Key("Enter")
	if len(g.P.Inv) != n || len(g.homeSt().Stash) != 0 {
		t.Fatal("take back failed")
	}
	g.P.HP = 1
	g.Hunger = 1500
	g.Key("s")
	if g.P.HP != g.P.MaxHP || g.Hunger != 0 {
		t.Fatal("sleep failed")
	}
}

func TestHomeReturnAndCompanion(t *testing.T) {
	g := newGame(t, nil)
	buyHouse(t, g)
	home := g.CurKey
	f := g.Spawn("villager", 1, g.R)
	f.AI, f.Faction = "follower", "player"
	f.X, f.Y = g.P.X, g.P.Y+1
	g.Cur.Set(f.X, f.Y, TFloor)
	g.Cur.Actors = append(g.Cur.Actors, f)
	g.Mode = "main"
	g.Key("H")
	g.Key("c")
	if f.AI != "town" || !f.Flags["stayhome"] {
		t.Fatal("companion did not stay")
	}
	g.Key("Escape")
	// walk away to a distant zone
	g.Warp(g.WX+3, g.WY, 1, 1, 0)
	if g.CurKey == home {
		t.Skip("warp landed home")
	}
	for _, a := range g.Cur.Actors {
		a.HP = 0
	}
	g.cleanup()
	g.Hunger = 0
	turn := g.Turn
	g.Key("F3")
	if g.Mode != "homepick" || len(g.Opts) != 1 {
		t.Fatalf("mode %s opts %d", g.Mode, len(g.Opts))
	}
	g.Key("a")
	if g.CurKey != home || g.Turn <= turn || g.Hunger == 0 {
		t.Fatalf("did not return home: %s turn %d hunger %d", g.CurKey, g.Turn-turn, g.Hunger)
	}
	found := false
	for _, a := range g.Cur.Actors {
		if a == f && a.AI == "town" {
			found = true
		}
	}
	if !found {
		t.Fatal("companion missing at home")
	}
}
