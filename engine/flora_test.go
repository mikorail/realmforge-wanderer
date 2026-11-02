package engine

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

// flZoneOfBiome generates a wild surface zone (not the feature centre) of a parasang with biome b.
func flZoneOfBiome(g *Game, b int, minH int) *Zone {
	w := g.W
	for y := 0; y < w.H; y++ {
		for x := 0; x < w.W; x++ {
			p := w.Idx(x, y)
			if w.Biome[p] != b || w.Burg[p] > 0 || w.Height[p] < minH {
				continue
			}
			zn := g.GetZone(x, y, 0, 0, 0)
			if zn.Kind == "surface" {
				return zn
			}
		}
	}
	return nil
}

func flSetSeason(g *Game, s int) {
	t := (s*SeasonDays+2)*DayLen - startMinutes
	for t < 0 {
		t += YearDays * DayLen
	}
	g.Turn = t
}

func TestFlSpeciesFitBiomeAndTerrain(t *testing.T) {
	g := newGame(t, nil)
	seen := map[int]int{}
	for _, b := range []int{1, 2, 3, 4, 5, 6, 7, 8, 12} {
		zn := flZoneOfBiome(g, b, 0)
		if zn == nil {
			continue
		}
		temp, ht := g.flClimate(zn)
		plants := 0
		for i, ter := range zn.T {
			sp := flSpeciesFor(g, zn, i%MW, i/MW)
			if sp == nil {
				continue
			}
			plants++
			if !flSlotFits(sp.Cl, ter) {
				t.Fatalf("biome %d: %s (class %d) on %s", b, sp.Name, sp.Cl, ter.D().Name)
			}
			native := false // the biome has its own species of this class (else a neighbour's plant bled in)
			for k := range flCat {
				if c := &flCat[k]; c.Cl == sp.Cl && c.Lay == flSurf && c.Bio != nil && flBioOK(c, b) {
					native = true
				}
			}
			if native && !flBioOK(sp, b) {
				t.Fatalf("biome %d: %s does not grow here", b, sp.Name)
			}
			// the strict stage won whenever any species of the class fits the climate
			strict := false
			for _, id := range g.flCandidates(zn, sp.Cl) {
				c := flCat[id-1]
				if temp >= c.TLo && temp <= c.THi && ht >= c.HLo && ht <= c.HHi {
					strict = true
				}
			}
			if strict && (temp < sp.TLo || temp > sp.THi || ht < sp.HLo || ht > sp.HHi) {
				t.Fatalf("biome %d (%d°C, h%d): %s outside its band", b, temp, ht, sp.Name)
			}
		}
		seen[b] = plants
	}
	if len(seen) < 5 {
		t.Fatalf("too few biomes sampled: %v", seen)
	}
	if zn := flZoneOfBiome(g, 1, 0); zn != nil {
		for i, ter := range zn.T {
			if ter == TDeadTree {
				if sp := flSpeciesFor(g, zn, i%MW, i/MW); sp == nil || sp.G != flGCactus && sp.G != flGSnag {
					t.Fatalf("hot desert snag should be a cactus or dead tree: %+v", sp)
				}
			}
		}
	}
}

func TestFlAlpineOnlyHigh(t *testing.T) {
	for _, sp := range flCat {
		if sp.HLo >= 60 && sp.HHi < sp.HLo {
			t.Fatalf("%s bad band", sp.Name)
		}
	}
	if len(flCat) < 150 || len(flCat) > 0xFFF0 {
		t.Fatalf("catalogue size %d", len(flCat))
	}
	names := map[string]bool{}
	for _, sp := range flCat {
		k := strings.ToLower(sp.Name)
		if names[k] {
			t.Fatalf("duplicate species %s", sp.Name)
		}
		names[k] = true
		if sp.Prod != "" {
			if _, ok := cookTags[sp.PTag]; !ok {
				t.Fatalf("%s: unknown cooking tag %q", sp.Name, sp.PTag)
			}
			if sp.PSeas == 0 {
				t.Fatalf("%s: product with no season", sp.Name)
			}
		}
	}
}

func TestFlDeterministicPerZone(t *testing.T) {
	g := newGame(t, nil)
	zn := flZoneOfBiome(g, 6, 0)
	if zn == nil {
		zn = flZoneOfBiome(g, 4, 0)
	}
	if zn == nil {
		t.Skip("no temperate land")
	}
	key := zn.Key
	a := make([]uint16, len(zn.T))
	for i := range zn.T {
		a[i] = g.flID(zn, i)
	}
	terr := append([]Terrain(nil), zn.T...)
	h := newGame(t, nil)
	z2 := h.GetZone(zn.WX, zn.WY, zn.ZX, zn.ZY, zn.Z)
	if z2.Key != key {
		t.Fatal("key mismatch")
	}
	mixed := map[uint16]bool{}
	for i := range z2.T {
		if z2.T[i] != terr[i] {
			t.Fatalf("terrain differs at %d", i)
		}
		if id := h.flID(z2, i); id != a[i] {
			t.Fatalf("species differs at %d: %d vs %d", i, id, a[i])
		} else if z2.T[i] == TTree && id > 0 {
			mixed[id] = true
		}
	}
	// a cache wipe (as after loading) gives the same answer
	z2.Flora.idx = nil
	for i := range z2.T {
		if h.flID(z2, i) != a[i] {
			t.Fatalf("rebuilt cache differs at %d", i)
		}
	}
	if len(mixed) < 2 {
		t.Logf("only %d tree species in %s", len(mixed), key)
	}
}

func flPlantTree(g *Game, name string) (int, int) {
	clearAround(g)
	for _, bp := range g.P.Body {
		if bp.Slot == "hand" && bp.Item != nil && bp.Item.Kind == "weapon" {
			bp.Item = nil
		}
	}
	x, y := g.P.X+1, g.P.Y
	g.Cur.Set(x, y, TTree)
	if !flSetSpecies(g, g.Cur, x, y, name) {
		panic("no species " + name)
	}
	return x, y
}

func TestFlSeasonChangesLook(t *testing.T) {
	g := newGame(t, nil)
	x, y := flPlantTree(g, "sugar maple")
	i := y*MW + x
	flSetSeason(g, 1)
	gs, fs := g.flTile(g.Cur, i, TTree, '♣', CGreen)
	flSetSeason(g, 2)
	ga, fa := g.flTile(g.Cur, i, TTree, '♣', CGreen)
	flSetSeason(g, 3)
	gw, fw := g.flTile(g.Cur, i, TTree, '♣', CGreen)
	if gs != '♣' || fa != CRed || gw != 'τ' || fw != CBrown || fs == fa || ga != '♣' {
		t.Fatalf("season looks: summer %c/%d autumn %c/%d winter %c/%d", gs, fs, ga, fa, gw, fw)
	}
	if look := g.flLook(g.Cur, x, y); !strings.Contains(look, "sugar maple") || !strings.Contains(look, "Bare for the winter") {
		t.Fatalf("winter look: %s", look)
	}
	flSetSeason(g, 2)
	if look := g.flLook(g.Cur, x, y); !strings.Contains(look, "Autumn has turned it scarlet") {
		t.Fatalf("autumn look: %s", look)
	}
	// a flower shows only in bloom
	g.Cur.Set(x, y, TGrass)
	flSetSpecies(g, g.Cur, x, y, "bluebell")
	flSetSeason(g, 0)
	if gl, f := g.flTile(g.Cur, i, TGrass, '.', CDarkGreen); gl != flGFlower || f != CBlue {
		t.Fatalf("bluebell in spring: %c %d", gl, f)
	}
	flSetSeason(g, 2)
	if gl, _ := g.flTile(g.Cur, i, TGrass, '.', CDarkGreen); gl != '.' {
		t.Fatalf("bluebell in autumn should not show: %c", gl)
	}
}

func TestFlChopYieldsWood(t *testing.T) {
	g := newGame(t, nil)
	x, y := flPlantTree(g, "sessile oak")
	ax := g.MakeWeapon("axe", 1, g.R)
	g.P.Inv = append(g.P.Inv, ax)
	if !g.equip(ax) {
		t.Fatal("could not wield the axe")
	}
	g.MoveP(1, 0)
	var logs *Item
	for _, it := range g.P.Inv {
		if it.Name == "oak logs" {
			logs = it
		}
	}
	if logs == nil || logs.Count < 2 || logs.Kind != "scrap" {
		t.Fatalf("no oak logs: %+v", logs)
	}
	if g.Cur.At(x, y) == TTree || !g.Cur.Flora.Felled[y*MW+x] {
		t.Fatal("tree still standing")
	}
	if look := g.flLook(g.Cur, x, y); !strings.Contains(look, "stump of a felled sessile oak") {
		t.Fatalf("stump look: %s", look)
	}
	if gl, _ := g.flTile(g.Cur, y*MW+x, g.Cur.At(x, y), '.', CBrown); gl != 'ⁿ' {
		t.Fatalf("stump glyph %c", gl)
	}
}

func TestFlGatherInSeason(t *testing.T) {
	g := newGame(t, nil)
	x, y := flPlantTree(g, "sessile oak")
	flSetSeason(g, 0)
	g.MoveP(1, 0)
	for _, it := range g.P.Inv {
		if it.Name == "acorns" {
			t.Fatal("acorns in spring")
		}
	}
	flSetSeason(g, 2)
	g.MoveP(1, 0)
	var ac *Item
	for _, it := range g.P.Inv {
		if it.Name == "acorns" {
			ac = it
		}
	}
	if ac == nil || ac.Kind != "ingredient" || ac.Effect != "lasting" {
		t.Fatalf("no acorns: %+v", ac)
	}
	n := ac.Count
	g.MoveP(1, 0) // once a year
	if ac.Count != n {
		t.Fatal("gathered twice in one year")
	}
	_ = x
	_ = y
	// walkable plant underfoot: pickup gathers
	g.Cur.Set(g.P.X, g.P.Y, TBush)
	flSetSpecies(g, g.Cur, g.P.X, g.P.Y, "bramble")
	g.pickup()
	found := false
	for _, it := range g.P.Inv {
		if it.Name == "blackberries" {
			found = true
		}
	}
	if !found {
		t.Fatal("no blackberries from the bramble underfoot")
	}
}

func TestFlOldSaveLoads(t *testing.T) {
	g := newGame(t, nil)
	x, y := flPlantTree(g, "sessile oak")
	ax := g.MakeWeapon("axe", 1, g.R)
	g.P.Inv = append(g.P.Inv, ax)
	g.equip(ax)
	g.MoveP(1, 0)
	s := g.Save()
	h, err := LoadGame(g.W, s)
	if err != nil {
		t.Fatal(err)
	}
	if !h.Cur.Flora.Felled[y*MW+x] || !strings.Contains(h.flLook(h.Cur, x, y), "sessile oak") {
		t.Fatal("felled state lost on load")
	}
	// a save from before flora (no "flora" keys) loads and grows its species back
	old := regexp.MustCompile(`,"flora":\{[^{}]*(\{[^{}]*\}[^{}]*)*\}`).ReplaceAllString(s, "")
	if strings.Contains(old, `"flora"`) {
		t.Fatal("strip failed")
	}
	o, err := LoadGame(g.W, old)
	if err != nil {
		t.Fatal(err)
	}
	o.Render()
	n := 0
	for i := range o.Cur.T {
		if o.flID(o.Cur, i) > 0 {
			n++
		}
	}
	if n == 0 && strings.Contains(string(o.Cur.Kind), "town") {
		t.Log("no plants in the start town")
	}
}

// FL_DUMP=1 go test -run TestFlDump ./engine prints sample looks.
func TestFlDump(t *testing.T) {
	if os.Getenv("FL_DUMP") == "" {
		t.Skip()
	}
	g := newGame(t, nil)
	for _, b := range []int{1, 3, 4, 5, 6, 7, 8, 12, 13} {
		zn := flZoneOfBiome(g, b, 0)
		if zn == nil {
			continue
		}
		cnt := map[string]int{}
		var looks []string
		for i := range zn.T {
			if sp := flSpeciesFor(g, zn, i%MW, i/MW); sp != nil {
				if cnt[sp.Name] == 0 && len(looks) < 3 {
					looks = append(looks, Plain(g.flLook(zn, i%MW, i/MW)))
				}
				cnt[sp.Name]++
			}
		}
		t.Logf("biome %d %s: %v", b, zn.Key, cnt)
		for _, l := range looks {
			t.Log("   ", l)
		}
	}
}
