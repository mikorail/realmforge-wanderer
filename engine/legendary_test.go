package engine

import (
	"strings"
	"testing"
)

func lgFindOpt(g *Game, prefix string) *Option {
	for i := range g.Opts {
		if strings.Contains(g.Opts[i].Text, prefix) {
			return &g.Opts[i]
		}
	}
	return nil
}

// lgPlaced puts a legendary wolf with the given opinion next to the player.
func lgPlaced(g *Game, word string, rep int) *Actor {
	a := placeFoe(g, "wolf", 6)
	makeLegendary(g, a, NewRng("lgtest"), g.Cur, "wild")
	a.Faction = "legend:" + word
	g.Rep[a.Faction] = rep
	lgSt(g).Opin[a.ID] = word
	return a
}

func TestLegendaryMakeStats(t *testing.T) {
	g := newGame(t, nil)
	b := g.Spawn("wolf", 6, NewRng("x"))
	hp, lv, str := b.MaxHP, b.Level, b.Attr[STR]
	a := makeLegendary(g, b, NewRng("y"), g.Cur, "wild")
	if a.MaxHP != hp*2 || a.Level != lv*3/2 || a.Attr[STR] != str+1 || a.XP != a.Level*250 || a.F != CMagenta || !a.Has("legend") {
		t.Fatalf("bad legendary: %+v", a)
	}
	got := false
	for _, it := range a.Inv {
		if rarityIndex(it.Rarity) >= 2 {
			got = true
		}
	}
	if !got || !a.Proper || a.Title != "wolf" && a.Title == "" {
		t.Fatalf("expected rare drop and name, inv=%d name=%s", len(a.Inv), a.Name)
	}
}

func TestLegendaryZonesAndDeed(t *testing.T) {
	g := newGame(t, nil)
	found := 0
	for dy := -8; dy <= 8 && found == 0; dy++ {
		for dx := -8; dx <= 8 && found == 0; dx++ {
			wx, wy := g.Cur.WX+dx, g.Cur.WY+dy
			if !g.W.In(wx, wy) {
				continue
			}
			zn := g.GetZone(wx, wy, 1, 1, 0)
			for _, a := range zn.Actors {
				if a.Has("legend") {
					found++
					if zn.Notes[a.Y*MW+a.X] == "" {
						t.Fatalf("legendary %s has no lair note", a.Name)
					}
				}
			}
		}
	}
	if found == 0 {
		t.Skip("no legendary within 17x17 zones")
	}
}

func TestLegendaryKillDeed(t *testing.T) {
	g := newGame(t, nil)
	a := lgPlaced(g, "hate", -450)
	n := len(g.Deeds)
	g.kill(a, g.P)
	if len(g.Deeds) <= n || !strings.Contains(g.Deeds[len(g.Deeds)-1].Text, a.Name) || len(lgSt(g).Slain) != 1 {
		t.Fatal("no deed for the slain legendary")
	}
}

func TestRitualV2(t *testing.T) {
	g := newGame(t, nil)
	clearAround(g)
	a := lgPlaced(g, "love", 380)
	g.openTalk(a)
	if lgFindOpt(g, "Speak as kin") != nil {
		t.Fatal("kin menu before the ritual")
	}
	g.ritual(a)
	if lgSt(g).Rit[a.Faction] != 1 {
		t.Fatalf("ritual not counted: %v", lgSt(g).Rit)
	}
	o := lgFindOpt(g, "Speak as kin")
	if o == nil {
		t.Fatal("no kin option after ritual")
	}
	o.Do()
	for _, want := range []string{"How do they regard", "Buy a secret", "Ask for a secret", "travel with you"} {
		if lgFindOpt(g, want) == nil {
			t.Fatalf("missing %q", want)
		}
	}
	// buy a secret with coin
	g.P.Gold = 5000
	ns := len(g.Secrets)
	lgFindOpt(g, "Buy a secret. (").Do()
	if g.P.Gold >= 5000 || len(g.Secrets) != ns+1 {
		t.Fatalf("secret not bought: gold %d secrets %d->%d", g.P.Gold, ns, len(g.Secrets))
	}
	// water
	g.P.Inv = append(g.P.Inv, g.MakeWater())
	d := g.lgDrams()
	g.lgKinMenu(a, "")
	if o := lgFindOpt(g, "with water"); o != nil {
		o.Do()
		if g.lgDrams() >= d && len(g.Secrets) == ns+1 {
			t.Fatal("water neither spent nor refused")
		}
	}
	// ask for a secret costs reputation
	g.lgKinMenu(a, "")
	before := g.Rep[a.Faction]
	lgFindOpt(g, "Ask for a secret").Do()
	if g.Rep[a.Faction] > before {
		t.Fatal("asking should cost reputation")
	}
	// learn a skill
	g.Rep[a.Faction] = 1000
	sk := g.lgSkillFor(a)
	rk := g.Skills[sk]
	g.lgKinMenu(a, "")
	lgFindOpt(g, "Learn "+skillName(sk)).Do()
	if g.Skills[sk] != rk+1 || g.Rep[a.Faction] >= 1000 {
		t.Fatalf("skill not learned: %d -> %d rep %d", rk, g.Skills[sk], g.Rep[a.Faction])
	}
	// gift of power at Revered
	g.Rep[a.Faction] = 1200
	id := g.lgPowerFor(a)
	g.lgKinMenu(a, "")
	lgFindOpt(g, "Receive").Do()
	if g.P.Powers[id] != 1 {
		t.Fatalf("power %s not granted", id)
	}
	// recruit the legend
	g.Rep[a.Faction] = 1200
	g.lgKinMenu(a, "")
	lgFindOpt(g, "travel with you").Do()
	if a.AI != "follower" || a.Comp == nil {
		t.Fatal("legend did not join")
	}
}

func TestRitualBandsLockOptions(t *testing.T) {
	g := newGame(t, nil)
	clearAround(g)
	a := lgPlaced(g, "like", 140)
	g.Talk = a
	g.ritual(a)
	g.Rep[a.Faction] = 100
	g.lgKinMenu(a, "")
	if lgFindOpt(g, "Buy a secret. (") != nil || lgFindOpt(g, "needs Liked") == nil {
		t.Fatal("secrets should be locked below Liked")
	}
	if lgBand(-700) != 0 || lgBand(0) != 2 || lgBand(300) != 3 || lgBand(700) != 4 || lgBand(950) != 5 {
		t.Fatal("band thresholds")
	}
}

func TestOathbreaker(t *testing.T) {
	g := newGame(t, nil)
	clearAround(g)
	a := lgPlaced(g, "love", 380)
	g.Talk = a
	g.ritual(a)
	g.Rep["state:1"] = 300
	g.kill(a, g.P)
	if g.Rep["state:1"] > 200 || lgSt(g).Oath != 1 {
		t.Fatalf("oathbreaker penalty missing: %d", g.Rep["state:1"])
	}
}

func TestLegendarySaveLoad(t *testing.T) {
	g := newGame(t, nil)
	clearAround(g)
	a := lgPlaced(g, "love", 380)
	g.Talk = a
	g.ritual(a)
	g.openTalk(a)
	g.Mode, g.Talk = "main", nil
	g2, err := LoadGame(g.W, g.Save())
	if err != nil {
		t.Fatal(err)
	}
	if lgSt(g2).Rit[a.Faction] != 1 || lgSt(g2).Opin[a.ID] != "love" {
		t.Fatalf("state lost: %+v", lgSt(g2))
	}
}
