package engine

import (
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"
)

// swFresh parses a world file anew (the shared test cache must not see injected social data).
func swFresh(t *testing.T, name string, strip bool) *WorldData {
	t.Helper()
	b, err := os.ReadFile("../public/worlds/" + name + ".json")
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	if strip {
		var u map[string]any
		if err := json.Unmarshal(b, &u); err != nil {
			t.Fatal(err)
		}
		for _, pl := range u["planets"].([]any) {
			p := pl.(map[string]any)
			delete(p, "forces")
			for _, st := range p["states"].([]any) {
				for _, k := range []string{"socialOrder", "forces", "reforms", "persecutions", "famine", "mixing"} {
					delete(st.(map[string]any), k)
				}
			}
			for _, c := range p["cultures"].([]any) {
				delete(c.(map[string]any), "prestige")
				delete(c.(map[string]any), "origin")
			}
			for _, pe := range p["people"].([]any) {
				delete(pe.(map[string]any), "mixed")
			}
		}
		nb, _ := json.Marshal(u)
		s = string(nb)
	}
	w, err := ParseWorld(s)
	if err != nil {
		t.Fatal(err)
	}
	return w
}

func swSpec() CharSpec {
	return CharSpec{Name: "Tester", Species: "human", Calling: "warrior", Attr: [6]int{18, 16, 16, 14, 14, 14}}
}

// swHome: the start burg the default spec would pick (most populous surface town with notables).
func swHome(w *WorldData) *BurgData {
	var b *BurgData
	for i := range w.Burgs {
		if w.Burgs[i].Alive && w.Burgs[i].Layer == "surface" && len(w.peopleByBrg[w.Burgs[i].ID]) > 0 && (b == nil || w.Burgs[i].Pop > b.Pop) {
			b = &w.Burgs[i]
		}
	}
	return b
}

func swRole(zn *Zone, role string) []*Actor {
	var out []*Actor
	for _, a := range zn.Actors {
		if a.Role == role {
			out = append(out, a)
		}
	}
	return out
}

func swNoteWith(zn *Zone, s string) bool {
	for _, n := range zn.Notes {
		if strings.Contains(n, s) {
			return true
		}
	}
	return false
}

func TestSocialWorldOldWorldUnchanged(t *testing.T) {
	w := swFresh(t, "ember-crown", true)
	if w.Forces != nil || w.States[1].SocialOrder != nil {
		t.Fatal("strip failed")
	}
	g := NewGame(w, swSpec())
	keys := []string{}
	for zy := 0; zy < 3; zy++ {
		for zx := 0; zx < 3; zx++ {
			zn := g.GetZone(g.WX, g.WY, zx, zy, 0)
			keys = append(keys, zn.Key)
			for _, a := range zn.Actors {
				switch a.Role {
				case "unfree", "construct", "reformer", "refugee", "striker", "captive":
					t.Errorf("old world got a %s", a.Role)
				}
			}
		}
	}
	// the same zones with the hook removed must be identical
	saved := zoneHooks
	var without []func(*Game, *Zone, *Rng)
	for _, h := range zoneHooks {
		if !sameFunc(h, swZone) {
			without = append(without, h)
		}
	}
	zoneHooks = without
	defer func() { zoneHooks = saved }()
	g2 := NewGame(swFresh(t, "ember-crown", true), swSpec())
	for _, k := range keys {
		a, b := g.Zones[k], g2.Zones[k]
		if b == nil {
			var wx, wy, zx, zy, z int
			fmtSscan(k, &wx, &wy, &zx, &zy, &z)
			b = g2.GetZone(wx, wy, zx, zy, z)
		}
		if len(a.T) != len(b.T) || len(a.Actors) != len(b.Actors) {
			t.Fatalf("zone %s differs without social data", k)
		}
		for i := range a.T {
			if a.T[i] != b.T[i] {
				t.Fatalf("zone %s terrain differs at %d", k, i)
			}
		}
		for i := range a.Actors {
			if a.Actors[i].Name != b.Actors[i].Name || a.Actors[i].X != b.Actors[i].X {
				t.Fatalf("zone %s actor %d differs", k, i)
			}
		}
	}
}

func sameFunc(a, b func(*Game, *Zone, *Rng)) bool {
	return reflect.ValueOf(a).Pointer() == reflect.ValueOf(b).Pointer()
}

func TestSocialWorldSlaveTownAndMovementQuest(t *testing.T) {
	w := swFresh(t, "ember-crown", false)
	home := swHome(w)
	st := w.stateByID[home.State]
	st.SocialOrder = &SocialOrderData{Kind: "slave", Name: "Unfree", Form: "Bondage", FormID: "bondage", Workers: "bondsfolk", Since: 120, Pressure: 62, Movement: true}
	g := NewGame(w, swSpec())
	zn := g.Cur
	if zn.Burg != home.ID {
		t.Fatalf("start %d, want %d", zn.Burg, home.ID)
	}
	un := swRole(zn, "unfree")
	if len(un) < 2 {
		t.Fatalf("want unfree workers, got %d", len(un))
	}
	if un[0].Caste != "bondsfolk" {
		t.Errorf("unfree stratum %q", un[0].Caste)
	}
	found := false
	for _, s := range g.strataFor(home) {
		found = found || s == "bondsfolk"
	}
	if !found {
		t.Errorf("strata %v lack the bound stratum", g.strataFor(home))
	}
	if !swNoteWith(zn, "MARKET BLOCK") {
		t.Error("no market block notice")
	}
	t.Logf("unfree: %s | %s | %s", un[0].Name, un[0].Desc, g.swGreeting(un[0]))
	ref := swRole(zn, "reformer")
	if len(ref) != 1 {
		t.Fatalf("want one reformer, got %d", len(ref))
	}
	a := ref[0]
	// helping the unfree: movement up, crown down
	g.P.Gold = 100
	crown0 := g.Rep[stateFaction(home.State)]
	g.Talk = un[0]
	g.swHelpUnfree(un[0])
	if g.Rep[swMvFaction(home.State)] <= 0 || g.Rep[stateFaction(home.State)] >= crown0 {
		t.Errorf("help rep: mv %d crown %d→%d", g.Rep[swMvFaction(home.State)], crown0, g.Rep[stateFaction(home.State)])
	}
	if n := g.FactionName(swMvFaction(home.State)); !strings.Contains(n, "Abolitionists") {
		t.Errorf("faction name %q", n)
	}
	// both quest kinds: pamphlets and the captive (the kind is a hash of the giver id)
	kinds := map[string]bool{}
	origID := a.ID
	for id := origID; id < origID+40 && len(kinds) < 2; id++ {
		a.ID = id
		g.Cur = zn
		g.Talk = a
		g.swOffer(a)
		s := swSt(g)
		q := s.Quests[s.Asked[a.ID]]
		if q == nil || kinds[q.Kind] {
			continue
		}
		kinds[q.Kind] = true
		switch q.Kind {
		case "pamphlets":
			tb := w.burgByID[q.Burg]
			tz := g.GetZone(tb.P%w.W, tb.P/w.W, 1, 1, 0)
			g.enterZone(tz, 40, 13, true)
			if q.Stage != 1 {
				t.Fatalf("pamphlets not posted in %s", tb.Name)
			}
		case "captive":
			var wx, wy, zx, zy, z int
			fmtSscan(q.Zone, &wx, &wy, &zx, &zy, &z)
			cz := g.GetZone(wx, wy, zx, zy, z)
			g.enterZone(cz, 40, 13, true)
			caps := swRole(cz, "captive")
			if len(caps) != 1 {
				t.Fatalf("captive not placed (%d)", len(caps))
			}
			crown := g.Rep[stateFaction(home.State)]
			g.Talk = caps[0]
			g.swFreeCaptive(caps[0], q)
			if q.Stage != 0 {
				t.Fatal("freed with overseers watching")
			}
			for _, m := range cz.Actors {
				if m.Name == "overseer" {
					m.HP = 0
				}
			}
			g.swFreeCaptive(caps[0], q)
			if q.Stage != 1 || len(swRole(cz, "captive")) != 0 || g.Rep[stateFaction(home.State)] >= crown {
				t.Fatalf("captive not freed: stage %d", q.Stage)
			}
		}
		g.enterZone(zn, 40, 13, true)
		gold, mv := g.P.Gold, g.Rep[swMvFaction(home.State)]
		g.Talk = a
		g.swTurnIn(a, q)
		if !q.Closed || g.P.Gold <= gold || g.Rep[swMvFaction(home.State)] <= mv {
			t.Fatalf("%s quest not rewarded", q.Kind)
		}
	}
	if !kinds["pamphlets"] || !kinds["captive"] {
		t.Errorf("quest kinds seen: %v", kinds)
	}
}

func TestSocialWorldConstructsAndForces(t *testing.T) {
	w := swFresh(t, "ember-crown", false)
	home := swHome(w)
	st := w.stateByID[home.State]
	st.SocialOrder = &SocialOrderData{Kind: "slave", Name: "Unfree", Form: "Golem labour", FormID: "golems", Workers: "golems", Construct: true, Since: 100}
	st.Forces = []ForceData{{Kind: "strikes", Name: "Factory strikes", N: 210, Note: "strike in year 210"}, {Kind: "genevats", Name: "Gene vats", N: 0.7}}
	st.Reforms = &ReformsData{Enacted: []string{"land"}}
	w.Forces = &WorldForces{Smog: []int{home.ID}}
	g := NewGame(w, swSpec())
	zn := g.Cur
	if len(swRole(zn, "construct")) == 0 || len(swRole(zn, "striker")) < 2 {
		t.Fatalf("constructs %d strikers %d", len(swRole(zn, "construct")), len(swRole(zn, "striker")))
	}
	vat := false
	for _, a := range zn.Actors {
		vat = vat || a.Title == "vat-tender"
	}
	if !vat || !swNoteWith(zn, "BINDING YARD") {
		t.Error("vat clinic or binding yard missing")
	}
	if !g.swSmogHere() || g.swSight(60) != 12 && !roofed(zn.At(g.P.X, g.P.Y)) {
		t.Error("smog does not cap sight")
	}
	m := swRole(zn, "striker")[0]
	food := &Item{Kind: "food", Value: 10}
	if f := g.swPriceMod(m, food, true); f >= 1 {
		t.Errorf("land reform food mod %f", f)
	}
}

func TestSocialWorldRefugeeCamp(t *testing.T) {
	w := swFresh(t, "ember-crown", false)
	home := swHome(w)
	// a neighbour persecutes the faith of the home realm; refugees flee to the home realm
	var from *StateData
	for i := range w.States {
		if s := &w.States[i]; s.Alive && s.ID > 0 && s.ID != home.State && w.burgByID[s.Capital] != nil {
			from = s
			break
		}
	}
	hst := w.stateByID[home.State]
	from.Persecutions = []PersecutionData{{Axis: "faith", Target: hst.Religion, TargetName: w.relByID[hst.Religion].Name, Kind: "expulsion", Name: "expulsion", Since: 200, Haven: home.State}}
	g := NewGame(w, swSpec())
	var camp *Zone
	for _, id := range swHavenTowns(w, from, home.State) {
		b := w.burgByID[id]
		sl := swFreeSlots(g, b)
		if len(sl) == 0 {
			continue
		}
		camp = g.GetZone(b.P%w.W, b.P/w.W, sl[0][0], sl[0][1], 0)
		break
	}
	if camp == nil {
		t.Skip("no outskirts for a camp")
	}
	refs := swRole(camp, "refugee")
	if len(refs) < 4 || !strings.HasPrefix(camp.Name, "Refugee camp") {
		t.Fatalf("camp %q with %d refugees", camp.Name, len(refs))
	}
	t.Logf("%s: %s | %s", camp.Name, refs[0].Desc, g.swGreeting(refs[1]))
	var elder *Actor
	for _, a := range refs {
		if a.Has("swelder") {
			elder = a
		}
	}
	g.enterZone(camp, 40, 13, true)
	g.Talk = elder
	g.swOfferFeed(elder)
	g.give(g.MakeFood(NewRng("t"), 3))
	q := swSt(g).Quests[swSt(g).Asked[elder.ID]]
	rep := g.Rep[elder.Faction]
	g.swFeed(elder, q)
	if !q.Closed || g.Rep[elder.Faction] <= rep {
		t.Fatal("feeding the camp did nothing")
	}
	// the player shares the persecuted faith: the persecuting realm's traders overcharge
	g.HomeFaith = hst.Religion
	trader := &Actor{Role: "merchant", State: from.ID, Tpl: "npc"}
	if g.swPriceMod(trader, &Item{Kind: "weapon"}, true) <= 1 {
		t.Error("no persecution markup")
	}
	if gr := g.swGreeting(&Actor{Role: "guard", State: from.ID, Tpl: "npc"}); !strings.Contains(gr, "not welcome") {
		t.Errorf("guard greeting %q", gr)
	}
	if len(g.swRealmLines(hst)) == 0 {
		t.Error("no gossip about the refugees")
	}
}

func TestSocialWorldMixedNPC(t *testing.T) {
	w := swFresh(t, "ember-crown", false)
	home := swHome(w)
	st := w.stateByID[home.State]
	other := -1
	for i := range w.Cultures {
		if w.Cultures[i].ID != home.Culture && w.Cultures[i].ID > 0 {
			other = w.Cultures[i].ID
			break
		}
	}
	st.Mixing = []MixData{{A: home.Culture, B: other, Unions: 7, Since: 150, House: "Blendhouse"}}
	g := NewGame(w, swSpec())
	var mixed *Actor
	for zy := 0; zy < 3 && mixed == nil; zy++ {
		for zx := 0; zx < 3 && mixed == nil; zx++ {
			for _, a := range g.GetZone(g.WX, g.WY, zx, zy, 0).Actors {
				if a.Has("swmix") && a.House == "Blendhouse" {
					mixed = a
					break
				}
			}
		}
	}
	if mixed == nil {
		t.Fatal("no mixed household")
	}
	if !strings.Contains(mixed.Desc, w.cultByID[home.Culture].Name) || !strings.Contains(mixed.Desc, w.cultByID[other].Name) || len(mixed.Kin) == 0 {
		t.Errorf("mixed desc %q kin %v", mixed.Desc, mixed.Kin)
	}
	t.Logf("%s: %s", mixed.Name, mixed.Desc)
}
