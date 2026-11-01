package engine

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func uniGame(t *testing.T) (*Game, *Universe) {
	t.Helper()
	w := world(t, "ember-crown")
	u := unUni(w)
	if len(u.Planets) < 2 || len(u.Rifts) == 0 || len(u.ends) == 0 {
		t.Skip("ember-crown is not a multi-planet universe file")
	}
	g := NewGame(w, CharSpec{Name: "Rifter", Species: "human", Calling: "warrior", Attr: [6]int{18, 16, 16, 14, 14, 14}})
	g.P.MaxHP, g.P.HP = 9999, 9999
	return g, u
}

// useGate bumps the gate in the current zone and accepts the first destination.
func useGate(t *testing.T, g *Game) {
	t.Helper()
	x, y, ok := g.unGate(g.Cur)
	if !ok || g.Cur.At(x, y) != TPortal {
		t.Fatalf("no rift gate in %s", g.Cur.Name)
	}
	g.P.X, g.P.Y = x, y+1
	g.interact(x, y)
	if g.Mode != "rift" {
		t.Fatalf("bumping the gate opened mode %q", g.Mode)
	}
	g.Key("y")
}

func TestUniverseLoads(t *testing.T) {
	w := world(t, "ember-crown")
	u := unUni(w)
	if u.Kind != "wanderer-universe" {
		t.Skip("not a universe file")
	}
	if w != u.Planets[u.Start] || w.uniIdx != u.Start {
		t.Fatal("ParseWorld should return the start planet")
	}
	for i, p := range u.Planets {
		if p.uni != u || p.uniIdx != i || len(p.Burgs) == 0 || p.W == 0 {
			t.Fatalf("planet %d not indexed", i)
		}
	}
	n := 0
	for _, m := range u.ends {
		n += len(m)
	}
	if n == 0 {
		t.Fatal("no rift ends indexed")
	}
}

func TestSingleWorldLoads(t *testing.T) {
	b, err := os.ReadFile("../public/worlds/ember-crown.json")
	if err != nil {
		t.Fatal(err)
	}
	var raw struct {
		Kind    string            `json:"kind"`
		Planets []json.RawMessage `json:"planets"`
	}
	single := string(b)
	if json.Unmarshal(b, &raw) == nil && raw.Kind == "wanderer-universe" {
		single = string(raw.Planets[0])
	}
	w, err := ParseWorld(single)
	if err != nil {
		t.Fatal(err)
	}
	if u := unUni(w); len(u.Planets) != 1 || len(u.Rifts) != 0 {
		t.Fatalf("single world should be a universe of one, got %d planets", len(u.Planets))
	}
	g := NewGame(w, CharSpec{Name: "Solo", Species: "human", Calling: "warrior", Attr: [6]int{14, 14, 14, 14, 14, 14}})
	if g.UniversePlanets() != 1 || g.PlanetIndex() != 0 || len(unSide(g)) != 0 {
		t.Fatal("single world game should know of one planet")
	}
	g2, err := LoadGame(w, g.Save())
	if err != nil || g2.W != w {
		t.Fatalf("single world save/load: %v", err)
	}
	// a save made before planets existed (no "uni" store) still loads on the start planet
	var m map[string]any
	_ = json.Unmarshal([]byte(g.Save()), &m)
	if st, ok := m["store"].(map[string]any); ok {
		delete(st, "uni")
	}
	old, _ := json.Marshal(m)
	g3, err := LoadGame(w, string(old))
	if err != nil || g3.W != w || g3.Cur == nil {
		t.Fatalf("old save: %v", err)
	}
}

func TestRiftTravelSwapsPlanetAndBack(t *testing.T) {
	g, u := uniGame(t)
	r := u.Rifts[0]
	if !g.UnGo(r.A.Planet, r.A.Burg) {
		t.Fatal("could not reach the rift town")
	}
	homeKey := g.Cur.Key
	homeZone := g.Cur
	homeKnown := len(g.Known)
	g.Slain[424242] = true
	useGate(t, g)
	if g.W != u.Planets[r.B.Planet] || g.PlanetIndex() != r.B.Planet {
		t.Fatalf("expected planet %d, on %d", r.B.Planet, g.PlanetIndex())
	}
	if unUni(g.W).gateZ[r.B.Planet][g.Cur.Key] != r.B.Burg {
		t.Fatalf("arrived in %s, not the paired burg", g.Cur.Name)
	}
	if g.Slain[424242] || g.Zones[homeKey] == homeZone {
		t.Fatal("planet state leaked across the rift")
	}
	last := g.Msgs[len(g.Msgs)-1].T
	found := false
	for _, m := range g.Msgs[maxi(0, len(g.Msgs)-6):] {
		found = found || strings.Contains(m.T, "You step through "+r.Name)
	}
	if !found {
		t.Fatalf("no travel message, last: %q", last)
	}
	if g.Seed != unSeed(u, r.B.Planet) || g.Genre != g.W.Genre {
		t.Fatal("seed/genre not swapped")
	}
	if side := unSide(g); len(side) != 1 || !strings.Contains(side[0], g.unPlanetName()) {
		t.Fatalf("sidebar: %v", side)
	}
	if g.unHome() != u.Planets[u.Start].Seed {
		t.Fatal("home seed must stay the start planet's")
	}
	useGate(t, g)
	if g.PlanetIndex() != r.A.Planet || g.Zones[homeKey] != homeZone || !g.Slain[424242] || len(g.Known) < homeKnown {
		t.Fatal("coming back did not restore the home planet")
	}
}

func TestSaveLoadAcrossPlanets(t *testing.T) {
	g, u := uniGame(t)
	r := u.Rifts[0]
	g.UnGo(r.A.Planet, r.A.Burg)
	g.Slain[777] = true
	useGate(t, g)
	g.Slain[888] = true
	cur := g.Cur.Key
	start := u.Planets[u.Start]
	g2, err := LoadGame(start, g.Save())
	if err != nil {
		t.Fatal(err)
	}
	if g2.W != u.Planets[r.B.Planet] || g2.Cur.Key != cur || !g2.Slain[888] || g2.Slain[777] {
		t.Fatalf("loaded on planet %d at %s", g2.PlanetIndex(), g2.Cur.Key)
	}
	if g2.Meta() == "" || !strings.Contains(g2.Meta(), `"seed":"`+start.Seed+`"`) {
		t.Fatalf("meta seed changed: %s", g2.Meta())
	}
	useGate(t, g2)
	if g2.PlanetIndex() != r.A.Planet || !g2.Slain[777] || g2.Slain[888] {
		t.Fatal("home planet state lost through save/load")
	}
	if _, err := LoadGame(start, g2.Save()); err != nil {
		t.Fatal(err)
	}
}

func TestSlainMonsterLeavesRemains(t *testing.T) {
	w := world(t, "ember-crown")
	var brood, bare *MonsterData
	for _, pw := range unUni(w).Planets {
		for i := range pw.Monsters {
			m := &pw.Monsters[i]
			if !m.Slain || m.Layer != "surface" || pw.IsWater(m.P) || pw.Burg[m.P] != 0 {
				continue
			}
			if pw.slHasBrood(m) && brood == nil {
				brood = m
			} else if !pw.slHasBrood(m) && bare == nil {
				bare = m
			}
		}
	}
	if brood == nil && bare == nil {
		// synthesise one so the test still exercises the path on older sample worlds
		for i := range w.Monsters {
			if w.Monsters[i].Layer == "surface" && !w.IsWater(w.Monsters[i].P) && w.Burg[w.Monsters[i].P] == 0 {
				w.Monsters[i].Slain, w.Monsters[i].SlainYear, w.Monsters[i].SlainBy = true, 100, "Test Hero"
				defer func(i int) { w.Monsters[i].Slain = false }(i)
				if w.slHasBrood(&w.Monsters[i]) {
					brood = &w.Monsters[i]
				} else {
					bare = &w.Monsters[i]
				}
				break
			}
		}
	}
	for _, m := range []*MonsterData{brood, bare} {
		if m == nil {
			continue
		}
		var pw *WorldData
		for _, p := range unUni(w).Planets {
			for i := range p.Monsters {
				if &p.Monsters[i] == m {
					pw = p
				}
			}
		}
		g := NewGame(w, CharSpec{Name: "Hunter", Species: "human", Calling: "warrior", Attr: [6]int{14, 14, 14, 14, 14, 14}})
		if pw != g.W {
			g.UnGo(pw.uniIdx, pw.Burgs[len(pw.Burgs)/2].ID)
			for _, b := range pw.Burgs {
				if g.W == pw {
					break
				}
				g.UnGo(pw.uniIdx, b.ID)
			}
		}
		if g.W != pw {
			t.Fatal("could not reach the monster's planet")
		}
		zn := g.GetZone(m.P%pw.W, m.P/pw.W, 1, 1, 0)
		var remains, br bool
		for _, a := range zn.Actors {
			if a.SimMon == m.ID {
				t.Fatalf("slain %s spawned alive", m.Name)
			}
			br = br || a.Name == "Brood of "+m.Name
		}
		for _, it := range zn.Items {
			remains = remains || it.Kind == "trophy" && strings.Contains(it.Lore, "slain")
		}
		if !remains {
			t.Fatalf("no remains of %s", m.Name)
		}
		if br != pw.slHasBrood(m) {
			t.Fatalf("brood of %s: got %v want %v", m.Name, br, pw.slHasBrood(m))
		}
		if m.Slain && strings.Contains(g.slSecret(m), "lairs at") {
			t.Fatal("secret should speak of the dead")
		}
	}
}

func TestVillainHideout(t *testing.T) {
	g := NewGame(world(t, "ember-crown"), CharSpec{Name: "Sheriff", Species: "human", Calling: "warrior", Attr: [6]int{18, 16, 16, 14, 14, 14}})
	vs := g.vlList()
	if len(vs) < 2 {
		t.Fatalf("want at least 2 villains, got %d", len(vs))
	}
	v := vs[0]
	bx, by := v.Burg.P%g.W.W, v.Burg.P/g.W.W
	s := g.siteAt(bx, by, v.ZX, v.ZY)
	if s == nil || s.Name != vlHideoutName(v) {
		t.Fatalf("no hideout site for %s: %+v", v.P.Name, s)
	}
	zn := g.GetZone(bx, by, v.ZX, v.ZY, s.Depth)
	boss := g.vlEnsure(zn, v.P.ID)
	if boss == nil || boss.Name != v.P.Name || !g.Hostile(boss) {
		t.Fatalf("villain not in hideout: %+v", boss)
	}
	rare := false
	for _, it := range boss.Inv {
		rare = rare || rarityIndex(it.Rarity) >= 2
	}
	if !rare {
		t.Fatal("villain carries no rare+ drop")
	}
	if b := g.vlBounty(v.P); b == nil || b.Z != s.Depth || b.ZX != v.ZX {
		t.Fatal("no bounty pointing at the hideout")
	}
	g.enterZone(zn, boss.X, boss.Y+1, true)
	n := len(g.Deeds)
	g.kill(boss, g.P)
	if !vlDead(g, v.P.ID) || len(g.Deeds) <= n || g.vlBounty(v.P) != nil {
		t.Fatal("killing the villain should be a deed and end them")
	}
}

func TestEveryRiftWorks(t *testing.T) {
	for _, name := range []string{"ember-crown", "neon-tide", "spore-mother", "ashen-vaults", "hollow-crown"} {
		w := world(t, name)
		u := unUni(w)
		g := NewGame(w, CharSpec{Name: "Walker", Species: "human", Calling: "warrior", Attr: [6]int{14, 14, 14, 14, 14, 14}})
		for _, r := range u.Rifts {
			if !g.UnGo(r.A.Planet, r.A.Burg) {
				t.Fatalf("%s: cannot reach %s end A", name, r.Name)
			}
			useGate(t, g)
			if g.PlanetIndex() != r.B.Planet || u.gateZ[r.B.Planet][g.Cur.Key] != r.B.Burg {
				t.Fatalf("%s: %s led to planet %d %s", name, r.Name, g.PlanetIndex(), g.Cur.Name)
			}
		}
	}
}

func TestMurimMembershipStaysOnItsPlanet(t *testing.T) {
	g, u := uniGame(t)
	r := u.Rifts[0]
	g.UnGo(r.A.Planet, r.A.Burg)
	ss := g.Sects()
	if len(ss) == 0 {
		t.Skip("no sects on this planet")
	}
	g.mrJoin(ss[0])
	g.mr().Qi = 7
	useGate(t, g)
	if g.mr().Sect != "" || g.mrMine() != nil {
		t.Fatal("sect membership followed you to another planet")
	}
	if g.mr().Qi != 7 {
		t.Fatal("cultivation should travel with you")
	}
	for k := range g.Rep {
		if strings.HasPrefix(k, "mr:") && g.SectByID(k[3:]) == nil {
			t.Fatalf("foreign sect rep %s on this planet", k)
		}
	}
	useGate(t, g)
	if g.mr().Sect != ss[0].ID || g.mrMine() != ss[0] {
		t.Fatal("membership lost on returning")
	}
}
