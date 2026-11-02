package engine

import (
	"reflect"
	"strings"
	"testing"
)

func TestBestiaryCatalogue(t *testing.T) {
	gr, er := bsCounts()
	t.Logf("species %d", len(bsAll))
	for _, k := range bsSortedKeys(gr) {
		t.Logf("  group %-20s %d", k, gr[k])
	}
	for _, k := range bsSortedKeys(er) {
		t.Logf("  era   %-20s %d", k, er[k])
	}
	if len(bsAll) < 400 {
		t.Fatalf("catalogue has %d species, want >= 400", len(bsAll))
	}
	names := map[string]bool{}
	g := newGame(t, nil)
	r := NewRng("bs:test:spawn")
	for _, s := range bsAll {
		low := strings.ToLower(s.Name)
		if names[low] {
			t.Fatalf("duplicate name %q", s.Name)
		}
		names[low] = true
		if s.Size < 1 || s.Size > 7 || !bsTempers[s.Temper] || bsGroupNames[s.Group] == "" || s.Era == "" {
			t.Fatalf("bad entry %+v", s)
		}
		for _, b := range s.Biomes {
			if b < 1 || b > 16 {
				t.Fatalf("%s: bad biome %d", s.Name, b)
			}
		}
		if len(s.Biomes) == 0 && s.Sea == 0 && s.Reef == 0 && !s.Fresh && !s.Coast && !s.Cave && !s.Sky {
			t.Fatalf("%s has no habitat", s.Name)
		}
		if s.Extinct() && s.Era != "recent" && (s.Period == "" || s.Ma <= 0) {
			t.Fatalf("%s: prehistoric species without a period", s.Name)
		}
		if tplByID[s.Tpl()] == nil {
			t.Fatalf("%s: template not registered", s.Name)
		}
		a := g.bsSpawn(s, 1, r)
		if a.MaxHP <= 0 || a.Name != s.Display() || a.Tpl != s.Tpl() {
			t.Fatalf("%s: bad spawn %+v", s.Name, a)
		}
		if s.Hostile() != (a.Faction == "beasts") {
			t.Fatalf("%s: hostility mismatch", s.Name)
		}
	}
	for i := range Templates {
		if strings.HasPrefix(Templates[i].ID, "bs_") {
			t.Fatalf("bestiary leaked into Templates: %s", Templates[i].ID)
		}
	}
	if s := bsSpecies("Tyrannosaurus"); s == nil || s.When() != "Late Cretaceous, ~68 million years ago" {
		t.Fatalf("Tyrannosaurus period wrong: %+v", s)
	}
	if d := bsSpecies("Woolly mammoth").Display(); d != "woolly mammoth" {
		t.Fatalf("display %q", d)
	}
	if d := bsSpecies("Steller's sea cow").Display(); d != "Steller's sea cow" {
		t.Fatalf("display %q", d)
	}
	if s := bsSpecies("dodo"); s == nil || s.When() != "Extinct since 1662" {
		t.Fatalf("dodo lookup wrong: %+v", s)
	}
	if bsAge(0.012) != "12,000 years ago" || bsAge(2.6) != "2.6 million years ago" || bsAge(0.3) != "300,000 years ago" {
		t.Fatalf("ages: %s / %s / %s", bsAge(0.012), bsAge(2.6), bsAge(0.3))
	}
}

func bsActorsIn(zn *Zone) []*Actor {
	var out []*Actor
	for _, a := range zn.Actors {
		if strings.HasPrefix(a.Tpl, "bs_") {
			out = append(out, a)
		}
	}
	return out
}

func TestBestiaryForestHasHerbivores(t *testing.T) {
	g := newGame(t, nil)
	w := g.W
	herb, zones, total := 0, 0, 0
	for p := 0; p < len(w.Biome) && zones < 40; p++ {
		if b := w.Biome[p]; b < 5 || b > 9 || w.Burg[p] > 0 {
			continue
		}
		zn := g.GetZone(p%w.W, p/w.W, 0, 0, 0)
		if zn.Kind != "surface" {
			continue
		}
		zones++
		for _, a := range bsActorsIn(zn) {
			total++
			if s := bsByTpl[a.Tpl]; s.Diet == 'h' && !s.Hostile() {
				herb++
			}
		}
	}
	t.Logf("%d forest zones: %d animals, %d herbivores", zones, total, herb)
	if zones == 0 || herb == 0 {
		t.Fatalf("no herbivores in %d forest zones", zones)
	}
}

func TestBestiaryKeepsExistingZoneGeneration(t *testing.T) {
	idx := -1
	for i, h := range zoneHooks {
		if reflect.ValueOf(h).Pointer() == reflect.ValueOf(bsZone).Pointer() {
			idx = i
		}
	}
	if idx != len(zoneHooks)-1 {
		t.Fatalf("bsZone should be the last zone hook, at %d of %d", idx, len(zoneHooks))
	}
	keys := [][5]int{}
	g := newGame(t, nil)
	w := g.W
	for p := 0; p < len(w.Biome) && len(keys) < 12; p += 37 {
		if !w.IsWater(p) && w.Burg[p] == 0 {
			keys = append(keys, [5]int{p % w.W, p / w.W, 0, 2, 0}, [5]int{p % w.W, p / w.W, 2, 1, 1})
		}
	}
	gen := func() []*Zone {
		g := newGame(t, nil)
		var out []*Zone
		for _, k := range keys {
			out = append(out, g.genZone(k[0], k[1], k[2], k[3], k[4]))
		}
		return out
	}
	with := gen()
	saved := zoneHooks
	zoneHooks = zoneHooks[:idx]
	without := gen()
	zoneHooks = saved
	for i := range with {
		a, b := with[i], without[i]
		if !reflect.DeepEqual(a.T, b.T) {
			t.Fatalf("zone %s terrain changed", a.Key)
		}
		var na []string
		for _, x := range a.Actors {
			if !strings.HasPrefix(x.Tpl, "bs_") {
				na = append(na, x.Tpl+"@"+x.Name)
			}
		}
		var nb []string
		for _, x := range b.Actors {
			nb = append(nb, x.Tpl+"@"+x.Name)
		}
		if !reflect.DeepEqual(na, nb) {
			t.Fatalf("zone %s actors changed:\n%v\n%v", a.Key, na, nb)
		}
	}
}

func bsFindLost(g *Game, kind string) (int, int, int, int, int, bool) {
	w := g.W
	for p := range w.Biome {
		for i := 0; i < 9; i++ {
			for _, z := range []int{0, 1, 2, 3, 4} {
				if g.bsLostAt(p%w.W, p/w.W, i%3, i/3, z) == kind {
					return p % w.W, p / w.W, i % 3, i / 3, z, true
				}
			}
		}
	}
	return 0, 0, 0, 0, 0, false
}

func TestBestiaryLostWorldDeterministic(t *testing.T) {
	g1 := newGame(t, nil)
	wx, wy, zx, zy, z, ok := bsFindLost(g1, "plateau")
	if !ok {
		t.Fatal("no lost plateau in the sample world")
	}
	g2 := newGame(t, nil)
	z1, z2 := g1.GetZone(wx, wy, zx, zy, z), g2.GetZone(wx, wy, zx, zy, z)
	if z1.Name != "The Lost Plateau" {
		t.Fatalf("zone %s is %q (%s)", z1.Key, z1.Name, z1.Kind)
	}
	a1, a2 := bsActorsIn(z1), bsActorsIn(z2)
	if len(a1) == 0 || len(a1) != len(a2) {
		t.Fatalf("lost world animals %d vs %d", len(a1), len(a2))
	}
	dinos := 0
	for i := range a1 {
		if a1[i].Tpl != a2[i].Tpl || a1[i].X != a2[i].X || a1[i].Y != a2[i].Y {
			t.Fatalf("lost world differs at %d: %s vs %s", i, a1[i].Tpl, a2[i].Tpl)
		}
		if bsByTpl[a1[i].Tpl].Group == "dino" {
			dinos++
		}
	}
	if dinos == 0 {
		t.Fatalf("no dinosaurs on the lost plateau: %v", a1)
	}
	var names []string
	for _, a := range a1 {
		names = append(names, a.Name)
	}
	t.Logf("%s at %d,%d: %s", z1.Name, wx, wy, strings.Join(names, ", "))
	for _, kind := range []string{"valley", "cavern", "trench"} {
		if wx, wy, zx, zy, z, ok := bsFindLost(g1, kind); ok {
			zn := g1.GetZone(wx, wy, zx, zy, z)
			t.Logf("%s (%s): %d animals", zn.Name, zn.Kind, len(bsActorsIn(zn)))
		}
	}
	// enter announces once and the rumour points at it
	bsEnter(g1, z1)
	if !g1.bs().Lost[g1.W.Seed+"|"+z1.Key] {
		t.Fatal("lost world not announced")
	}
}

func TestBestiaryFossils(t *testing.T) {
	g := newGame(t, nil)
	s := bsSpecies("Tyrannosaurus")
	it := g.bsMakeFossil(s, NewRng("bs:test:fossil"))
	if bsFossilSpecies(it) != s || it.Kind != "fossil" {
		t.Fatalf("fossil maps to %v", bsFossilSpecies(it))
	}
	text := strings.Join(it.Describe(g), "\n")
	if !strings.Contains(text, "Tyrannosaurus") || !strings.Contains(text, "Late Cretaceous, ~68 million years ago") || !strings.Contains(text, s.Desc) {
		t.Fatalf("fossil describe: %s", text)
	}
	g.P.Inv = append(g.P.Inv, it)
	g.Turn = (g.Turn/30 + 1) * 30
	bsTurn(g)
	if b := g.codex().Beasts[s.Tpl()]; b == nil || b.Lore == "" {
		t.Fatal("fossil not recorded in the bestiary")
	}
	// fossils lie in some strata
	found := 0
	w := g.W
	for p := 0; p < len(w.Biome) && found == 0; p += 7 {
		zn := g.GetZone(p%w.W, p/w.W, 0, 0, 2)
		for _, it := range zn.Items {
			if bsFossilSpecies(it) != nil {
				found++
			}
		}
	}
	if found == 0 {
		t.Fatal("no fossils found in strata")
	}
	// a collector buys them
	a := &Actor{ID: g.newID(), Name: "Ilse", Proper: true, Role: "bookseller", Faction: "state:0", AI: "town"}
	opts := bsTalk(g, a)
	if len(opts) == 0 || !strings.Contains(opts[0].Text, "fossil") {
		t.Fatalf("no fossil option: %v", opts)
	}
	g.Talk = a
	gold := g.P.Gold
	opts[0].Do()
	if g.P.Gold <= gold {
		t.Fatalf("selling fossils paid nothing (%d → %d)", gold, g.P.Gold)
	}
	for _, x := range g.P.Inv {
		if x.Kind == "fossil" {
			t.Fatal("fossil still in pack")
		}
	}
}

func bsTestPut(g *Game, name string, dx, dy int) *Actor {
	s := bsSpecies(name)
	a := g.bsSpawn(s, 1, NewRng("bs:test:"+name))
	a.X, a.Y = g.P.X+dx, g.P.Y+dy
	g.Cur.Set(a.X, a.Y, TFloor)
	g.Cur.Actors = append(g.Cur.Actors, a)
	g.recompute()
	return a
}

func TestBestiaryBehaviourAndHunting(t *testing.T) {
	g := newGame(t, nil)
	clearAround(g)
	for dy := -6; dy <= 6; dy++ {
		for dx := -8; dx <= 8; dx++ {
			g.Cur.Set(g.P.X+dx, g.P.Y+dy, TFloor)
		}
	}
	g.Turn = 12 * 60 // daylight: diurnal and crepuscular animals awake
	deer := bsTestPut(g, "Roe deer", 2, 0)
	if g.Hostile(deer) {
		t.Fatal("deer should not be hostile")
	}
	d0 := dist(deer, g.P)
	for i := 0; i < 3; i++ {
		g.bsAct(deer)
	}
	if dist(deer, g.P) <= d0 {
		t.Fatalf("skittish deer did not flee (%d → %d)", d0, dist(deer, g.P))
	}
	// codex records the wildlife it sees
	bsTurn(g)
	if b := g.codex().Beasts[deer.Tpl]; b == nil || b.Seen == 0 || !strings.Contains(b.Lore, "herbivore") {
		t.Fatalf("deer not in codex: %+v", b)
	}
	moose := bsTestPut(g, "Moose", 0, 3)
	moose.Y = g.P.Y + 3
	g.bsAct(moose)
	if !moose.Flags["bs_warned"] || moose.Angry {
		t.Fatal("moose should warn first")
	}
	moose.Y = g.P.Y + 1
	g.bsAct(moose)
	if !moose.Angry {
		t.Fatal("moose should charge when you stay close")
	}
	// hunting yields
	stag := bsTestPut(g, "Red deer", -1, 0)
	n := len(g.Cur.Items)
	g.hurt(stag, 999, g.P, "test")
	var got []string
	for _, it := range g.Cur.Items[n:] {
		got = append(got, it.Name)
	}
	j := strings.Join(got, ", ")
	if !strings.Contains(j, "red deer meat") || !strings.Contains(j, "antlers") || !strings.Contains(j, "red deer hide") {
		t.Fatalf("red deer yields: %s", j)
	}
	if b := g.codex().Beasts[stag.Tpl]; b == nil || b.Kills != 1 {
		t.Fatalf("kill not counted: %+v", b)
	}
	// tusks are ivory
	ele := bsTestPut(g, "Woolly mammoth", 1, 1)
	n = len(g.Cur.Items)
	g.hurt(ele, 9999, g.P, "test")
	ivory := false
	for _, it := range g.Cur.Items[n:] {
		ivory = ivory || strings.HasSuffix(it.Name, "ivory")
	}
	if !ivory {
		t.Fatal("mammoth left no ivory")
	}
}

func TestBestiarySaveLoad(t *testing.T) {
	g := newGame(t, nil)
	wx, wy, zx, zy, z, ok := bsFindLost(g, "plateau")
	if !ok {
		t.Skip("no plateau")
	}
	zn := g.GetZone(wx, wy, zx, zy, z)
	bsEnter(g, zn)
	g.P.Inv = append(g.P.Inv, g.bsMakeFossil(bsSpecies("Trilobite"), NewRng("x")))
	n := len(bsActorsIn(zn))
	h, err := LoadGame(g.W, g.Save())
	if err != nil {
		t.Fatal(err)
	}
	hz := h.Zones[zn.Key]
	if hz == nil || len(bsActorsIn(hz)) != n {
		t.Fatalf("lost world animals lost in save/load")
	}
	if !h.bs().Lost[h.W.Seed+"|"+zn.Key] {
		t.Fatal("bestiary state lost in save/load")
	}
	found := false
	for _, it := range h.P.Inv {
		if s := bsFossilSpecies(it); s != nil && s.Name == "Trilobite" {
			found = true
		}
	}
	if !found {
		t.Fatal("fossil lost in save/load")
	}
	for _, a := range bsActorsIn(hz) {
		h.Cur = hz
		h.bsAct(a) // the brain works on loaded actors
	}
}
