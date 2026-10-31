package engine

import "testing"

// firstFrontier finds a surface parasang on a war front.
func firstFrontier(g *Game, t *testing.T) (wx, wy, own, foe int) {
	t.Helper()
	w := g.W
	for y := 0; y < w.H; y++ {
		for x := 0; x < w.W; x++ {
			if o, f, ok := g.frontierAt(x, y); ok && w.Biome[w.Idx(x, y)] > 0 {
				return x, y, o, f
			}
		}
	}
	t.Skip("no frontier")
	return
}

func TestWarFrontier(t *testing.T) {
	g := newGame(t, nil)
	wx, wy, own, foe := firstFrontier(g, t)
	if !g.atWar(stateFaction(own), stateFaction(foe)) && !g.atWar(stateFaction(foe), stateFaction(own)) {
		t.Fatalf("%d and %d not at war", own, foe)
	}
	if _, _, ok := g.frontierAt(wx, wy); !ok {
		t.Fatal("not a frontier")
	}
	// a parasang with no warring neighbour is not a frontier
	for y := 0; y < g.W.H; y++ {
		for x := 0; x < g.W.W; x++ {
			if g.W.State[g.W.Idx(x, y)] == 0 {
				if _, _, ok := g.frontierAt(x, y); ok {
					t.Fatal("wilds flagged as frontier")
				}
				return
			}
		}
	}
}

func TestWarSkirmish(t *testing.T) {
	g := newGame(t, nil)
	w := g.W
	found := 0
	for y := 0; y < w.H && found == 0; y++ {
		for x := 0; x < w.W && found == 0; x++ {
			if _, _, ok := g.frontierAt(x, y); !ok || w.Biome[w.Idx(x, y)] == 0 || w.Burg[w.Idx(x, y)] > 0 {
				continue
			}
			for zy := 0; zy < 3; zy++ {
				for zx := 0; zx < 3; zx++ {
					zn := g.GetZone(x, y, zx, zy, 0)
					var sold []*Actor
					for _, a := range zn.Actors {
						if a.Has("soldier") {
							sold = append(sold, a)
						}
					}
					if len(sold) == 0 {
						continue
					}
					found++
					fac := map[string]bool{}
					for _, a := range sold {
						fac[a.Faction] = true
					}
					if len(fac) != 2 {
						t.Fatalf("expected two factions, got %v", fac)
					}
					var a, b *Actor
					for _, s := range sold {
						if a == nil {
							a = s
						} else if s.Faction != a.Faction {
							b = s
						}
					}
					if b == nil || !g.Enemies(a, b) {
						t.Fatalf("skirmishers are not enemies: %v %v", a, b)
					}
					banners := 0
					for _, n := range zn.Notes {
						if len(n) > 0 {
							banners++
						}
					}
					if banners == 0 {
						t.Error("no banners")
					}
				}
			}
		}
	}
	if found == 0 {
		t.Skip("no skirmish spawned")
	}
}

func TestWarEnlistKillContractAndRank(t *testing.T) {
	g := newGame(t, nil)
	_, _, own, foe := firstFrontier(g, t)
	clearAround(g)
	elder := &Actor{ID: g.newID(), Name: "Old", Role: "elder", Faction: stateFaction(own), State: own, Tpl: "npc"}
	g.Rep[stateFaction(own)] = 10
	g.Talk = elder
	hasEnlist := false
	for _, o := range warTalk(g, elder) {
		if Plain(o.Text) != "" {
			hasEnlist = true
			o.Do()
		}
	}
	if !hasEnlist {
		t.Fatal("no enlist option")
	}
	es := g.war()
	if es.Realm != own {
		t.Fatalf("not enlisted: %+v", es)
	}
	g.warOffer(elder)
	if es.C == nil {
		t.Fatal("no contract offered")
	}
	es.C = &warContract{Kind: "kill", Enemy: foe, Need: 2, Pay: 100}
	gold, repFoe := g.P.Gold, g.Rep[stateFaction(foe)]
	for i := 0; i < 2; i++ {
		s := g.soldier(foe, g.R)
		s.X, s.Y = g.P.X+1, g.P.Y
		g.Cur.Actors = append(g.Cur.Actors, s)
		g.kill(s, g.P)
	}
	if es.C != nil || es.Done != 1 || g.P.Gold < gold+100 {
		t.Errorf("contract not completed: %+v gold %d->%d", es, gold, g.P.Gold)
	}
	if g.Rep[stateFaction(foe)] >= repFoe {
		t.Error("no rep loss with enemy")
	}
	// promotion
	g.warAddMerit(40)
	if g.warRank(es) < 2 {
		t.Errorf("rank %d merit %d", g.warRank(es), es.Merit)
	}
	if g.warStanding() <= 0 {
		t.Error("no standing bonus")
	}
	if len(warSide(g)) == 0 {
		t.Error("no sidebar")
	}
	g.warAddMerit(1000)
	if warRanks[g.warRank(es)] != "Marshal" {
		t.Error("not marshal")
	}
	// killing own soldier discharges
	s := g.soldier(own, g.R)
	s.X, s.Y = g.P.X+1, g.P.Y
	g.Cur.Actors = append(g.Cur.Actors, s)
	g.kill(s, g.P)
	if es.Realm != 0 {
		t.Error("not discharged")
	}
	g.Mode = "war"
	g.Render()
}
