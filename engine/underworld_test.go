package engine

import (
	"strings"
	"testing"
)

func uwNPCNear(g *Game, role string) *Actor {
	a := g.Spawn("villager", 2, g.R)
	a.Role, a.AI, a.Proper, a.Name = role, "still", true, "Mark"
	a.Faction = uwFac(g.uwStateHere())
	a.State = g.uwStateHere()
	a.X, a.Y = g.P.X+1, g.P.Y
	g.Cur.Set(a.X, a.Y, TFloor)
	for _, o := range g.Cur.Actors {
		if o.X == a.X && o.Y == a.Y {
			o.X, o.Y = 0, 0
		}
	}
	g.Cur.Actors = append(g.Cur.Actors, a)
	return a
}

func TestUwLawsDifferByRealm(t *testing.T) {
	g := newGame(t, nil)
	seen := map[uwLaws]bool{}
	for _, s := range g.W.States {
		seen[g.uwLawsFor(s.ID)] = true
	}
	if len(seen) < 3 {
		t.Fatalf("laws too uniform: %d variants", len(seen))
	}
	st := g.uwStateHere()
	it := &Item{Kind: "scrap", Name: "x", Extra: map[string]int{"stolen": st + 1}}
	if !g.uwIllegal(it, st) || g.uwIllegal(it, st+1) {
		t.Fatal("stolen flag should be illegal only in the victim realm")
	}
	if !g.uwIllegal(g.uwVenom(g.R), st) || !g.uwIllegal(g.uwPapers(), st) {
		t.Fatal("poison and forgeries are always illegal")
	}
}

func TestUwBountyJailAndSave(t *testing.T) {
	g := newGame(t, nil)
	st := g.uwStateHere()
	g.uwAddBounty(st, 100, "")
	if g.uwBounty(st) <= 0 {
		t.Fatal("no bounty")
	}
	hot := g.MakeFood(g.R, 1)
	g.uwMarkStolen(hot, st)
	g.give(hot)
	t0 := g.Turn
	g.uwJail(st)
	if g.uwBounty(st) != 0 || g.Turn <= t0 {
		t.Fatal("jail should clear bounty and pass time")
	}
	for _, it := range g.P.Inv {
		if it == hot {
			t.Fatal("stolen goods not confiscated")
		}
	}
	g.uwAddBounty(st, 55, "")
	g2, err := LoadGame(g.W, g.Save())
	if err != nil {
		t.Fatal(err)
	}
	if g2.uwBounty(st) != g.uwBounty(st) {
		t.Fatal("bounty lost on load")
	}
	g.uwPayBounty(st)
	g.P.Gold = 1000
	if !g.uwPayBounty(st) || g.uwBounty(st) != 0 {
		t.Fatal("pay failed")
	}
}

func TestUwSteal(t *testing.T) {
	g := newGame(t, nil)
	v := uwNPCNear(g, "villager")
	v.Attr[INT] = 16
	hits := 0
	for i := 0; i < 40; i++ {
		it := g.MakeFood(g.R, 1)
		it.Name, it.Value = "loaf", 20
		v.Inv = []*Item{it}
		if g.uwAttempt(v, uwLoot{label: "loaf", it: it}, false) {
			hits++
			if !uwIsStolen(g.P.Inv[len(g.P.Inv)-1]) {
				t.Fatal("stolen item not flagged")
			}
		}
		v.Angry = false
	}
	if hits == 0 {
		t.Fatalf("steal hits %d/40", hits)
	}
	if g.uwStealChance(v, nil, true, false) >= g.uwStealChance(v, nil, false, false) {
		t.Fatal("worn items should be harder")
	}
}

func TestUwChestAndKey(t *testing.T) {
	g := newGame(t, nil)
	g.Skills["tinkering"] = 3
	g.P.Attr[AGI], g.P.Attr[INT] = 30, 30
	x, y := g.P.X+1, g.P.Y
	g.Cur.Set(x, y, TCrate)
	g.Mode = "main"
	n := len(g.P.Inv)
	for i := 0; i < 5 && !g.uwSt().Looted[g.uwChestKey(x, y)]; i++ {
		g.uwPickChest(x, y)
	}
	if !g.uwSt().Looted[g.uwChestKey(x, y)] || len(g.P.Inv) <= n {
		t.Fatal("chest not looted")
	}
	g.Key("X")
	if g.Mode != "uwmenu" {
		t.Fatalf("X mode %s", g.Mode)
	}
	g.Key("Escape")
}

func TestUwSearchAndFence(t *testing.T) {
	g := newGame(t, nil)
	st := g.uwStateHere()
	bad := g.MakeFood(g.R, 1)
	g.uwMarkStolen(bad, st)
	g.give(bad)
	g.Turn = 100
	bd := g.W.burgByID[g.Cur.Burg]
	items := g.uwContraband(bd.State)
	if len(items) == 0 {
		t.Fatal("no contraband found")
	}
	g.uwSearchMenu(bd.State, items, false)
	if g.Mode != "uwmenu" || len(g.Opts) < 3 {
		t.Fatalf("search menu %s %d", g.Mode, len(g.Opts))
	}
	g.P.Gold = 500
	g.Opts[0].Do()
	if g.Mode != "main" || len(g.uwContraband(bd.State)) != 0 {
		t.Fatal("submit should confiscate")
	}
	// fence buys hot goods above ordinary price
	f := uwNPCNear(g, "fence")
	f.Title = "fence"
	hot := g.MakeWeapon("axe", 2, g.R)
	plain := *hot
	g.uwMarkStolen(hot, st)
	if g.uwFencePays(hot, st, f) <= g.uwFencePays(&plain, st, f) {
		t.Fatal("fence should pay more for hot goods")
	}
	g.give(hot)
	g.openTalk(f)
	found := false
	for _, o := range g.Opts {
		if strings.Contains(o.Text, "Sell goods quietly") {
			found = true
			o.Do()
		}
	}
	if !found || g.Mode != "uwmenu" {
		t.Fatal("no fence menu")
	}
}

func TestUwDealerAndWager(t *testing.T) {
	g := newGame(t, nil)
	st := g.uwStateHere()
	d := uwNPCNear(g, "bmdealer")
	g.uwStockDealer(d, st, NewRng("t"))
	if len(d.Inv) < 5 {
		t.Fatalf("thin stock %d", len(d.Inv))
	}
	g.P.Gold = 5000
	g.uwDealerMenu(d, 0)
	n := len(g.P.Inv)
	g.Opts[0].Do()
	if len(g.P.Inv) != n+1 && !g.P.Inv[0].Stack() {
		t.Fatal("purchase failed")
	}
	pm := uwNPCNear(g, "pitmaster")
	g.uwWagerSides(pm)
	g.Opts[0].Do()
	before := g.P.Gold
	g.Opts[0].Do()
	if g.P.Gold == before && g.Mode != "uwmenu" {
		t.Fatal("wager did not resolve")
	}
	fs := g.uwFighters()
	od := uwOdds(fs)
	if od[0] <= 1 && od[1] <= 1 {
		t.Fatal("odds nonsense")
	}
}

func TestUwPitBout(t *testing.T) {
	g := newGame(t, nil)
	rc, ok := g.lot(g.Cur, NewRng("pit"), 13, 15, 8, 9)
	if !ok {
		t.Skip("no lot")
	}
	g.building(g.Cur, rc[0], rc[1], rc[2], rc[3], TWall, TSand)
	g.uwSt().Rings[g.Cur.Key] = rc
	pm := uwNPCNear(g, "pitmaster")
	g.P.Gold = 100
	g.uwStartBout(pm, 50)
	st := g.uwSt()
	if st.Pit == nil || len(st.Pit.Foes) == 0 {
		t.Fatal("no bout")
	}
	for _, a := range append([]*Actor{}, g.Cur.Actors...) {
		if a.Faction == "pit" {
			a.HP = 0
			g.kill(a, g.P)
		}
	}
	if st.Pit != nil || st.Wins != 1 || g.P.Gold <= 50 {
		t.Fatalf("bout not won: %+v gold %d", st.Pit, g.P.Gold)
	}
}
