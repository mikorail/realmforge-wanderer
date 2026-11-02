package engine

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

// nmWorld parses ember-crown fresh and injects a names block around a wild parasang that has a road
// and a river (so the generic bridge, milestone and sidebar paths all run). Returns the world, that
// parasang, and the neighbour the injected road runs to.
func nmWorld(t *testing.T, inject bool) (*WorldData, int, int) {
	t.Helper()
	w := swFresh(t, "ember-crown", false)
	w.Names = nil
	p, q := -1, -1
	for i := range w.Road {
		if w.Road[i] == 0 || w.River[i] == 0 || w.Burg[i] != 0 || w.IsWater(i) {
			continue
		}
		x, y := i%w.W, i/w.W
		for _, d := range [][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
			if w.In(x+d[0], y+d[1]) && w.Road[w.Idx(x+d[0], y+d[1])] > 0 && w.Burg[w.Idx(x+d[0], y+d[1])] == 0 {
				p, q = i, w.Idx(x+d[0], y+d[1])
				break
			}
		}
		if p >= 0 {
			break
		}
	}
	if p < 0 {
		t.Skip("no road crossing a river in ember-crown")
	}
	if !inject {
		return w, p, q
	}
	var a, b *BurgData
	for i := range w.Burgs {
		bd := &w.Burgs[i]
		if bd.Alive && bd.Layer == "surface" {
			if a == nil {
				a = bd
			} else if b == nil {
				b = bd
			}
		}
	}
	peak := p
	w.Names = &NamesData{
		Rivers:   []NmRiver{{ID: 7, Name: "Kalvik", Parasangs: []int{p - w.W, p, p + w.W}}},
		Routes:   []NmRoute{{ID: 3, Kind: "road", Name: "the Salt Road", A: a.ID, B: b.ID, Parasangs: []int{a.P, p, q, b.P}}},
		Features: []NmFeature{{Kind: "forest", Name: "the Ducner Woods", Parasangs: []int{p, q}}, {Kind: "range", Name: "the Wufe Mountains", Parasangs: []int{p}, Peak: &peak}},
		Burgs:    map[int]NmBurg{a.ID: {Tier: "metropolis", Harbor: "Port of " + a.Name}, b.ID: {Tier: "hamlet"}},
	}
	return w, p, q
}

func nmNotes(zn *Zone) []string {
	var out []string
	for i, t := range zn.T {
		if t == TInscription {
			out = append(out, zn.Notes[i])
		}
	}
	return out
}

func TestNamesParse(t *testing.T) {
	js := `{"rivers":[{"id":1,"name":"Wohe","parasangs":[3,4]}],"routes":[{"id":2,"kind":"sea","name":"the Myowa Run","a":1,"b":2,"parasangs":[5,6]}],` +
		`"features":[{"kind":"range","name":"the Fopu Spine","parasangs":[0,1],"peak":0}],"burgs":{"12":{"tier":"town","harbor":"Diqia Haven"}}}`
	w := swFresh(t, "ember-crown", false)
	var nd NamesData
	if err := json.Unmarshal([]byte(js), &nd); err != nil {
		t.Fatal(err)
	}
	w.Names = &nd
	if f := w.nmFeatures(0); len(f) != 1 || f[0].Peak == nil || *f[0].Peak != 0 {
		t.Fatalf("peak at parasang 0 lost: %+v", f)
	}
	if b, ok := w.nmBurg(12); !ok || b.Harbor != "Diqia Haven" {
		t.Fatalf("burg names: %+v", b)
	}
	if rv := w.nmRiver(4, false); rv == nil || rv.Name != "Wohe" {
		t.Fatal("river lookup")
	}
	if rs := w.nmRoutes(6, "sea"); len(rs) != 1 {
		t.Fatal("route lookup")
	}
}

func TestNamesShown(t *testing.T) {
	w, p, q := nmWorld(t, true)
	g := NewGame(w, swSpec())
	g.Known[p] = true
	zn := g.GetZone(p%w.W, p/w.W, 1, 1, 0)
	g.enterZone(zn, 40, 13, true)
	side := strings.Join(nmSide(g), "\n")
	for _, s := range []string{"the Wufe Mountains", "the Kalvik", "the Salt Road"} {
		if !strings.Contains(side, s) {
			t.Errorf("sidebar lacks %q:\n%s", s, side)
		}
	}
	info := strings.Join(g.nmInfo(p), "\n")
	for _, s := range []string{"the Wufe Mountains", "(the summit)", "River: the Kalvik", "the Salt Road"} {
		if !strings.Contains(info, s) {
			t.Errorf("world map info lacks %q:\n%s", s, info)
		}
	}
	if strings.Contains(zn.Name, "by the river") {
		t.Errorf("zone still says 'by the river': %s", zn.Name)
	}
	// every zone of the parasang: a named bridge sign and a milestone toward q
	var notes []string
	for zy := 0; zy < 3; zy++ {
		for zx := 0; zx < 3; zx++ {
			notes = append(notes, nmNotes(g.GetZone(p%w.W, p/w.W, zx, zy, 0))...)
		}
	}
	all := strings.Join(notes, "\n---\n")
	if !strings.Contains(all, "Kalvik") || !strings.Contains(all, "crosses the Kalvik") {
		t.Errorf("no bridge named after the Kalvik:\n%s", all)
	}
	b := w.burgByID[w.Names.Routes[0].B]
	if !strings.Contains(all, "the Salt Road}}\nAhead: {{W|"+b.Name) {
		t.Errorf("no milestone toward %s (q=%d):\n%s", b.Name, q, all)
	}
	// town entry uses the tier word; the harbour / inn / gossip paths run without trouble
	a := w.burgByID[w.Names.Routes[0].A]
	tz := g.GetZone(a.P%w.W, a.P/w.W, 1, 1, 0)
	g.enterZone(tz, 40, 13, true)
	found := false
	for _, m := range g.Msgs {
		if strings.Contains(m.T, "You enter the metropolis of "+a.Name) {
			found = true
		}
	}
	if !found {
		t.Error("no tier message on entering " + a.Name)
	}
	g.enterZone(zn, 40, 13, true)
	hit := false
	for id := 1; id < 400 && !hit; id++ {
		hit = g.nmGossip(&Actor{ID: id}) != ""
	}
	if !hit {
		t.Error("no feature gossip")
	}
}

// zone generation with names stays deterministic and never moves g.R; old worlds generate the same as before
func TestNamesDeterministic(t *testing.T) {
	gen := func(inject bool) ([]string, uint64) {
		w, p, _ := nmWorld(t, inject)
		g := NewGame(w, swSpec())
		before := g.R.S
		var out []string
		for zy := 0; zy < 3; zy++ {
			for zx := 0; zx < 3; zx++ {
				zn := g.genZone(p%w.W, p/w.W, zx, zy, 0)
				out = append(out, zn.Name, fmt.Sprint(zn.T), fmt.Sprint(len(zn.Actors)), strings.Join(nmNotes(zn), "|"))
			}
		}
		if g.R.S != before {
			t.Errorf("zone generation moved g.R (inject=%v)", inject)
		}
		return out, g.R.S
	}
	a, _ := gen(true)
	b, _ := gen(true)
	if !reflect.DeepEqual(a, b) {
		t.Fatal("names zone generation is not deterministic")
	}
	c, _ := gen(false)
	d, _ := gen(false)
	if !reflect.DeepEqual(c, d) {
		t.Fatal("plain zone generation is not deterministic")
	}
	// with names, the only differences are signs and zone names: actor counts match
	for i := 2; i < len(a); i += 4 {
		if a[i] != c[i] {
			t.Errorf("names changed the population of zone %d: %s vs %s", i/4, a[i], c[i])
		}
	}
}

func TestNamesOldWorld(t *testing.T) {
	w, p, _ := nmWorld(t, false)
	g := NewGame(w, swSpec())
	zn := g.GetZone(p%w.W, p/w.W, 1, 1, 0)
	g.enterZone(zn, 40, 13, true)
	if l := nmSide(g); l != nil {
		t.Fatalf("old world sidebar: %v", l)
	}
	if l := g.nmInfo(p); l != nil {
		t.Fatalf("old world info: %v", l)
	}
	if s := g.nmGossip(&Actor{ID: 1}); s != "" {
		t.Fatal("old world gossip")
	}
	if s := g.nmSiteName("tower", p, "Tower of X"); s != "Tower of X" {
		t.Fatal("old world site renamed")
	}
}

func TestNamesSafeSign(t *testing.T) {
	zn := newZone("t", 0, 0, 0, 0, 0)
	for i := range zn.T {
		zn.T[i] = TWall
	}
	// a one-wide corridor: a sign inside it would cut it in two
	for x := 5; x < 30; x++ {
		zn.T[10*MW+x] = TCobble
	}
	if nmSafe(zn, 15, 10) {
		t.Fatal("sign allowed to block a corridor")
	}
	// an open square: fine
	for y := 3; y < 9; y++ {
		for x := 40; x < 50; x++ {
			zn.T[y*MW+x] = TCobble
		}
	}
	if !nmSafe(zn, 44, 5) {
		t.Fatal("sign refused in an open square")
	}
}
