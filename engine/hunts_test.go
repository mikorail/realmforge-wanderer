package engine

import (
	"strings"
	"testing"
)

func bhTown(t *testing.T, g *Game) *BurgData {
	t.Helper()
	bd := g.W.burgByID[g.Cur.Burg]
	if bd == nil {
		t.Skip("start zone has no burg")
	}
	return bd
}

func TestBhMapClueAndDig(t *testing.T) {
	g := newGame(t, nil)
	r := NewRng("bhtest")
	c := bhNewChain(g, r, g.Cur.WX, g.Cur.WY, 1)
	if c == nil {
		t.Fatal("no cache")
	}
	c.Guard = ""
	it := bhMapItem(g, c, r)
	if it.Kind != "map" || !strings.Contains(bhMapText(g, c, 0), c.Region) {
		t.Fatal("bad map")
	}
	if !strings.Contains(bhMapText(g, c, 2), "parasang") {
		t.Fatal("no exact clue")
	}
	g.give(it)
	zn := g.GetZone(c.WX, c.WY, c.ZX, c.ZY, 0)
	if !c.Placed {
		t.Fatalf("cache not placed (zone kind %s)", zn.Kind)
	}
	g.enterZone(zn, c.X, c.Y, false)
	if bhSt(g).Clue[c.ID] < 2 {
		t.Fatal("clue did not sharpen on arrival")
	}
	g.P.X, g.P.Y = c.X, c.Y
	g.P.Inv = append(g.P.Inv, bhSpade(g))
	gold := g.P.Gold
	bhDig(g)
	if !c.Dug || g.P.Gold <= gold {
		t.Fatal("dig did not yield the cache")
	}
	gold = g.P.Gold
	bhDig(g)
	if g.P.Gold != gold {
		t.Fatal("cache paid twice")
	}
}

func TestBhChainRelic(t *testing.T) {
	g := newGame(t, nil)
	r := NewRng("bhchain")
	c := bhNewChain(g, r, g.Cur.WX, g.Cur.WY, 3)
	if c == nil || c.Of != 3 {
		t.Fatal("no chain")
	}
	st := bhSt(g)
	last := st.Caches[c.ID+2]
	if last == nil || last.Art == 0 || bhArt(g, last.Art) == nil {
		t.Fatal("final cache has no real relic")
	}
	if st.Caches[c.ID+1] == nil {
		t.Fatal("middle cache missing")
	}
}

func TestBhBoardAndMonsterBounty(t *testing.T) {
	g := newGame(t, nil)
	bd := bhTown(t, g)
	list := bhLedger(g, bd)
	if len(list) == 0 {
		t.Skip("no bounties near start")
	}
	for _, b := range list {
		if b.Name == "" || b.Reward <= 0 || b.Poster == "" {
			t.Fatalf("bad bounty %+v", b)
		}
	}
	// a synthetic monster bounty pays on proof
	st := bhSt(g)
	if len(g.W.Monsters) == 0 {
		g.W.Monsters = append(g.W.Monsters, MonsterData{ID: 9001, Name: "The Test Wyrm", Species: "Wyrm", Tier: "legendary", Power: 40, HP: 200, Layer: "surface",
			P: g.W.Idx(g.Cur.WX, g.Cur.WY), Trophy: "Wyrm Fang"})
	}
	m := &g.W.Monsters[0]
	if b := bhGenMonster(g, NewRng("m"), bd, g.Cur.WX, g.Cur.WY); b == nil || b.Name == "" {
		t.Fatal("monster bounty not generated from sim monsters")
	}
	b := &bhBounty{ID: st.next(), Kind: "monster", Name: m.Name, Mon: m.ID, Burg: bd.ID, State: bd.State, Reward: 300, Placed: true, Taken: true}
	st.Bounties[b.ID] = b
	d := &Actor{ID: g.newID(), Name: m.Name, SimMon: m.ID, X: g.P.X + 1, Y: g.P.Y}
	bhKill(g, d, g.P)
	if !b.Dead {
		t.Fatal("kill not recorded")
	}
	var proof *Item
	for _, it := range g.Cur.Items {
		if it.Extra != nil && it.Extra["bhp"] == b.ID {
			proof = it
		}
	}
	if proof == nil {
		t.Fatal("no proof dropped")
	}
	if bhProof(g, b) != nil {
		t.Fatal("proof counted before pickup")
	}
	g.give(proof)
	gold := g.P.Gold
	guard := &Actor{ID: g.newID(), Role: "guard", Name: "Warden", State: bd.State, Faction: stateFaction(bd.State)}
	g.Talk = guard
	bhBoard(g, guard, bd)
	found := false
	for _, o := range g.Opts {
		if strings.Contains(o.Text, "Hand in the proof") {
			o.Do()
			found = true
			break
		}
	}
	if !found || g.P.Gold != gold+300 || !b.Done {
		t.Fatalf("not paid: found=%v gold %d->%d", found, gold, g.P.Gold)
	}
}

func TestBhCapture(t *testing.T) {
	g := newGame(t, nil)
	bd := bhTown(t, g)
	st := bhSt(g)
	a := &Actor{ID: g.newID(), Name: "Rook", HP: 5, MaxHP: 30, AI: "hostile", Faction: "outlaws", X: g.P.X + 1, Y: g.P.Y}
	a.Set("bhT")
	g.Cur.Actors = append(g.Cur.Actors, a)
	b := &bhBounty{ID: st.next(), Kind: "person", Name: "Rook", Burg: bd.ID, State: bd.State, Reward: 200, Placed: true, Taken: true, Live: true, ActorID: a.ID}
	st.Bounties[b.ID] = b
	bhTurn(g)
	if !a.Has("yield") || a.AI != "still" {
		t.Fatal("wounded target did not yield")
	}
	var bind *Option
	for _, o := range bhTalk(g, a) {
		if strings.Contains(o.Text, "Bind") {
			o := o
			bind = &o
		}
	}
	if bind == nil {
		t.Fatal("no bind option")
	}
	bind.Do()
	if !a.Has("bhcap") || a.AI != "follower" || !b.Cap {
		t.Fatal("not captured")
	}
	guard := &Actor{ID: g.newID(), Role: "guard", Name: "Warden", State: bd.State}
	a.X, a.Y = g.P.X+1, g.P.Y
	gold := g.P.Gold
	bhBoard(g, guard, bd)
	for _, o := range g.Opts {
		if strings.Contains(o.Text, "Hand over the prisoner") {
			o.Do()
		}
	}
	if !b.Done || g.P.Gold != gold+300 {
		t.Fatalf("capture not paid: %d -> %d", gold, g.P.Gold)
	}
}

func TestBhHunters(t *testing.T) {
	g := newGame(t, nil)
	old := bhPlayerBounty
	defer func() { bhPlayerBounty = old }()
	zn := g.Cur
	p := g.W.Idx(zn.WX, zn.WY)
	if g.W.State[p] <= 0 {
		t.Skip("stateless start")
	}
	if n := bhMaybeHunters(g, zn, true); n != 0 {
		t.Fatal("hunters without a bounty")
	}
	bhPlayerBounty = func(g *Game, state int) int { return 500 }
	if n := bhMaybeHunters(g, zn, true); n == 0 {
		t.Fatal("no hunters despite bounty")
	}
	hs := 0
	for _, a := range zn.Actors {
		if a.Has("bhhunter") && g.Hostile(a) {
			hs++
		}
	}
	if hs == 0 {
		t.Fatal("hunters not hostile")
	}
}

func TestBhSaveLoadState(t *testing.T) {
	g := newGame(t, nil)
	c := bhNewChain(g, NewRng("x"), g.Cur.WX, g.Cur.WY, 2)
	if c == nil {
		t.Skip()
	}
	g.flushExt()
	if _, ok := g.Store["bh"]; !ok {
		t.Fatal("state not stored")
	}
}
