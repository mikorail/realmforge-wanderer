package engine

import (
	"reflect"
	"testing"
)

func mrGame(t *testing.T, name string) *Game {
	t.Helper()
	w := world(t, name)
	return NewGame(w, CharSpec{Name: "Tester", Species: "human", Calling: "warrior", Attr: [6]int{18, 18, 16, 18, 14, 18}})
}

func TestMrSectsDeterministic(t *testing.T) {
	for _, n := range []string{"ember-crown", "neon-tide", "spore-mother"} {
		g1 := mrGame(t, n)
		a := g1.mrGenSects()
		b := g1.mrGenSects()
		if !reflect.DeepEqual(a, b) {
			t.Fatalf("%s: sect generation not deterministic", n)
		}
		if len(a) < 6 || len(a) > 10 {
			t.Fatalf("%s: %d sects", n, len(a))
		}
		al := map[string]int{}
		for _, s := range a {
			al[s.Align]++
			if s.Name == "" || s.Leader == "" || len(s.Halls) < 2 || len(s.Halls) > 5 || s.Art.Tech[2].Name == "" || len(s.Rivals) == 0 {
				t.Errorf("%s: incomplete %+v", n, s)
			}
			states := map[int]bool{}
			for _, h := range s.Halls {
				states[h.State] = true
			}
			t.Logf("%s: %s [%s %s] HQ %s halls=%d states=%d art=%s", n, s.Name, s.Kind, s.Align, s.HQName, len(s.Halls), len(states), s.Art.Name)
		}
		if al[mrDemonic] != 1 || al[mrOrthodox] == 0 || al[mrUnorthodox] == 0 {
			t.Errorf("%s: alignments %v", n, al)
		}
	}
}

func TestMrHQAndHallNPCs(t *testing.T) {
	g := mrGame(t, "ember-crown")
	s := g.Sects()[0]
	zn := g.GetZone(s.WX, s.WY, 1, 1, 0)
	found := false
	for _, a := range zn.Actors {
		if a.Has("mrlead:" + s.ID) {
			found = true
		}
	}
	if !found || zn.Name != s.HQName {
		t.Fatalf("HQ not built: %v %q", found, zn.Name)
	}
	h := s.Halls[0]
	tz := g.GetZone(h.WX, h.WY, 1, 1, 0)
	ok := false
	for _, a := range tz.Actors {
		if a.Has("mrhall:" + s.ID) {
			ok = true
		}
	}
	if !ok {
		t.Fatalf("no hall master in %s", h.Town)
	}
}

func TestMrJoinRankBreakthroughTechnique(t *testing.T) {
	g := mrGame(t, "ember-crown")
	st := g.mr()
	var s *MrSect
	for _, x := range g.Sects() {
		if x.Align == mrOrthodox {
			s = x
			break
		}
	}
	mrRoll = func(*Game) float64 { return 0 }
	defer func() { mrRoll = func(g *Game) float64 { return g.R.Float() } }()
	clearAround(g)
	g.Cur.Actors = []*Actor{g.P}
	g.mrJoin(s)
	if st.Sect != s.ID || len(st.Techs) != 1 {
		t.Fatalf("join failed %+v", st)
	}
	if ok, _ := g.mrPromote(s); ok {
		t.Fatal("promoted without contribution")
	}
	st.Contrib = 1000
	st.Realm = 3
	for i := 0; i < 3; i++ {
		if ok, msg := g.mrPromote(s); !ok {
			t.Fatalf("promote %d: %s", i, msg)
		}
	}
	if st.Rank != 3 || len(st.Techs) != 3 {
		t.Fatalf("rank %d techs %v", st.Rank, st.Techs)
	}
	// breakthrough success
	st.Realm, st.Prog = 0, mrBreakNeed[0]
	hp, str := g.P.MaxHP, g.P.Attr[s.Art.Stat]
	ok := g.mrBreakthrough()
	if !ok || st.Realm != 1 || g.P.MaxHP <= hp || g.P.Attr[s.Art.Stat] <= str {
		t.Fatalf("breakthrough failed: ok=%v realm %d hp %d>%d str %d>%d last=%v", ok, st.Realm, g.P.MaxHP, hp, g.P.Attr[s.Art.Stat], str, g.Msgs[len(g.Msgs)-1])
	}
	// deviation
	mrRoll = func(*Game) float64 { return 0.99 }
	st.Prog = mrBreakNeed[1]
	mrApply := st.Realm
	if g.mrBreakthrough() || st.Realm != mrApply || st.Dev != 1 || st.Prog >= mrBreakNeed[1] {
		t.Fatalf("expected deviation: %+v", st)
	}
	// technique use
	foe := placeFoe(g, "bandit", 2)
	foe.Faction, foe.Angry = "outlaws", true
	st.Realm = 4
	g.mrQi()
	st.Qi = 100
	var id string
	for _, tid := range st.Techs {
		_, tt := g.mrTech(tid)
		if tt.Kind == "palm" || tt.Kind == "dash" || tt.Kind == "burst" || tt.Kind == "flurry" || tt.Kind == "drain" {
			id = tid
			break
		}
	}
	if id == "" {
		t.Skip("no targeted technique for this sect")
	}
	if len(g.mrAbilities()) != 3 {
		t.Fatalf("abilities %d", len(g.mrAbilities()))
	}
	before := foe.HP
	g.mrUse(mrAbID(id), foe.X, foe.Y)
	_, tt := g.mrTech(id)
	if st.Qi < 100-tt.Cost || st.Qi > 100-tt.Cost+3 || g.P.Cool[mrAbID(id)] <= 0 || g.P.Cool[mrAbID(id)] > tt.Cool {
		t.Fatalf("qi %d cool %d", st.Qi, g.P.Cool[mrAbID(id)])
	}
	_ = before
}

func TestMrSaveLoad(t *testing.T) {
	g := mrGame(t, "ember-crown")
	s := g.Sects()[0]
	g.mrJoin(s)
	st := g.mr()
	st.Contrib, st.Realm, st.Prog = 123, 2, 55
	g.mrApply()
	st.Mission = g.mrNewMission(s)
	g2, err := LoadGame(g.W, g.Save())
	if err != nil {
		t.Fatal(err)
	}
	st2 := g2.mr()
	if st2.Sect != s.ID || st2.Contrib != 123 || st2.Realm != 2 || st2.Prog != 55 || st2.Mission == nil || len(st2.Techs) != 1 || st2.App != st.App {
		t.Fatalf("state lost: %+v", st2)
	}
	if g2.P.MaxHP != g.P.MaxHP {
		t.Fatal("maxhp changed")
	}
}

func TestMrPillsAndRumors(t *testing.T) {
	g := mrGame(t, "neon-tide")
	st := g.mr()
	g.give(g.mrPill(1))
	if !g.mrUseItem(g.mrPackItems()[0]) || st.Prog == 0 {
		t.Fatal("pill did nothing")
	}
	_ = g.mrRumors()
	_ = mrSide(g)
}
