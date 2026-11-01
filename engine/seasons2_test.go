package engine

import (
	"strings"
	"testing"
)

// snWildZone finds a generated surface zone near the start whose biome grows herbs in every season.
func snWildZone(t *testing.T, g *Game) *Zone {
	t.Helper()
	for r := 1; r < 8; r++ {
		for dy := -r; dy <= r; dy++ {
			for dx := -r; dx <= r; dx++ {
				wx, wy := g.WX+dx, g.WY+dy
				if !g.W.In(wx, wy) {
					continue
				}
				zn := g.genZone(wx, wy, 1, 1, 0)
				if zn.Kind != "surface" {
					continue
				}
				ok := true
				for s := 0; s < 4; s++ {
					if len(g.snHerbPicks(zn, s)) == 0 {
						ok = false
					}
				}
				if ok {
					return zn
				}
			}
		}
	}
	t.Fatal("no wild zone with herbs in every season")
	return nil
}

func snNames(its []*Item) string {
	var s []string
	for _, it := range its {
		s = append(s, it.Name)
	}
	return strings.Join(s, ",")
}

func TestSeasonalHerbs(t *testing.T) {
	g := newGame(t, nil)
	zn := snWildZone(t, g)
	sets := map[string]bool{}
	for s := 0; s < 4; s++ {
		a, b := g.snHerbPicks(zn, s), g.snHerbPicks(zn, s)
		if snNames(a) != snNames(b) || a[0].X != b[0].X || a[0].Y != b[0].Y {
			t.Fatalf("season %d herbs not deterministic: %s vs %s", s, snNames(a), snNames(b))
		}
		for _, it := range a {
			if it.Extra["snH"] != s+1 {
				t.Fatalf("herb %s not marked for season %d", it.Name, s)
			}
			found := false
			for _, h := range snHerbs {
				if h.Name == it.Name && snIn(h.Seasons, s) {
					found = true
				}
			}
			if !found {
				t.Fatalf("%s out of season %d", it.Name, s)
			}
		}
		sets[snNames(a)] = true
	}
	if len(sets) < 3 {
		t.Fatalf("seasonal herb sets barely differ: %v", sets)
	}
	// seeding through the enter hook: this season's appear, they wither next season, the rng is untouched
	r := g.R.S
	g.Turn = 0
	g.snSeasonHerbs(zn)
	count := func(s int) int {
		n := 0
		for _, it := range zn.Items {
			if it.Extra != nil && it.Extra["snH"] == s+1 {
				n++
			}
		}
		return n
	}
	if count(0) == 0 {
		t.Fatal("no spring herbs seeded")
	}
	g.snSeasonHerbs(zn)
	n0 := count(0)
	g.Turn += SeasonDays * DayLen
	g.snSeasonHerbs(zn)
	if count(0) != 0 || count(1) == 0 {
		t.Fatalf("spring herbs left %d, summer %d", count(0), count(1))
	}
	if n0 != len(g.snHerbPicks(zn, 0)) {
		t.Fatalf("seeded twice in one season: %d", n0)
	}
	if g.R.S != r {
		t.Fatal("seasonal herbs drew from the game rng")
	}
	// cooking with a seasonal herb marks the dish in the cookbook
	h1 := g.snHerbPicks(zn, 1)[0]
	h2 := g.makeIngredient("wild garlic", "hearty", 1, CGreen)
	h1.ID = g.newID()
	g.P.Inv = append(g.P.Inv, h1, h2)
	g.CookSel = map[int]bool{h1.ID: true, h2.ID: true}
	g.cook([]*Item{h1, h2})
	if l := strings.Join(g.cookbookLines(), "\n"); !strings.Contains(l, "[summer]") {
		t.Fatalf("cookbook lacks the seasonal mark:\n%s", l)
	}
}

func TestFestivalDishWindow(t *testing.T) {
	g := newGame(t, nil)
	b, f := toFeast(t, g)
	var m *Actor
	for _, a := range g.Cur.Actors {
		if a.Role == "merchant" {
			m = a
			break
		}
	}
	if m == nil {
		// no stall in the start district: make a townsperson the merchant
		for _, a := range g.Cur.Actors {
			if a.AI == "town" && a.HP > 0 && !g.Hostile(a) {
				m = a
				break
			}
		}
		if m == nil {
			t.Fatal("no townsfolk")
		}
		m.Role = "merchant"
	}
	name, key := g.W.snFestivalDish(b, f)
	tag, _, _, _, _ := feastStyleOf(f.Kind)
	if tagsOf(key)[tag] != 2 {
		t.Fatalf("festival dish %s (%s) lacks level 2 %s", name, key, tag)
	}
	if n2, k2 := g.W.snFestivalDish(b, f); n2 != name || k2 != key {
		t.Fatal("festival dish not deterministic")
	}
	feastDay := g.Day()
	has := func() bool {
		for _, it := range m.Inv {
			if it.Kind == "dish" && it.Name == name {
				return true
			}
		}
		return false
	}
	g.wxForce = &Weather{Kind: WeatherByID("clear"), Temp: 15}
	// a fortnight before: not yet on sale (unless another feast is due then)
	g.Turn = (feastDay-14)*DayLen + 600
	if g.Turn > 0 {
		g.openTrade(m)
		if _, ff, _ := g.snFeastWindow(); has() && (ff == nil || ff.Name == f.Name) {
			t.Fatal("festival dish on sale two weeks early")
		}
	}
	// two days before: on sale
	g.Turn = (feastDay-2)*DayLen + 600
	g.openTrade(m)
	if !has() {
		t.Fatal("festival dish not on sale two days before the feast")
	}
	// the feast day itself: still there
	g.Turn = feastDay*DayLen + 600
	g.openTrade(m)
	if !has() {
		t.Fatal("festival dish gone on the feast day")
	}
	// after the feast: taken off the stall
	g.Turn = (feastDay+5)*DayLen + 600
	g.openTrade(m)
	if has() {
		t.Fatal("festival dish still sold after the feast")
	}
	// the seasonal special is stocked once per season
	sp, _ := snSpecialFor(g.snTownCulture(), g.Season())
	n := 0
	for _, it := range m.Inv {
		if it.Name == sp {
			n++
		}
	}
	if sp != "" && n != 1 {
		t.Fatalf("seasonal special %q stocked %d times", sp, n)
	}
	// joining the feast writes the festival dish into the cookbook, marked
	g.Turn = feastDay*DayLen + 600
	if a := townsfolk(g); a != nil && g.canJoinFeast(a) {
		g.joinFeast(a)
		if l := strings.Join(g.cookbookLines(), "\n"); !strings.Contains(l, "[festival]") {
			t.Fatal("cookbook lacks the festival mark")
		}
	}
}

func TestStormShelter(t *testing.T) {
	g := newGame(t, nil)
	var folk []*Actor
	for _, a := range g.Cur.Actors {
		if a.AI == "town" && a.HP > 0 && !g.Hostile(a) && !roofed(g.Cur.At(a.X, a.Y)) {
			folk = append(folk, a)
		}
	}
	if len(folk) == 0 {
		t.Skip("no townsfolk outdoors")
	}
	g.wxForce = &Weather{Kind: WeatherByID("storm"), Temp: 12}
	if g.snFoul() == "" {
		t.Fatal("storm not foul")
	}
	// outdoor merchants close
	for _, a := range g.Cur.Actors {
		if a.Role == "merchant" && !roofed(g.Cur.At(a.X, a.Y)) {
			g.openTrade(a)
			if g.Mode == "trade" {
				t.Fatal("market open in a storm")
			}
			break
		}
	}
	for i := 0; i < 120; i++ {
		for _, a := range g.Cur.Actors {
			if a.HP > 0 && a != g.P {
				g.aiAct(a)
			}
		}
	}
	in := 0
	for _, a := range folk {
		if roofed(g.Cur.At(a.X, a.Y)) {
			in++
		}
	}
	if in*2 < len(folk) {
		t.Fatalf("only %d of %d townsfolk took shelter", in, len(folk))
	}
	// clear weather: they walk home
	g.wxForce = &Weather{Kind: WeatherByID("clear"), Temp: 15}
	for i := 0; i < 200; i++ {
		for _, a := range g.Cur.Actors {
			if a.HP > 0 && a != g.P {
				g.aiAct(a)
			}
		}
	}
	home, still := 0, 0
	for _, a := range folk {
		if abs(a.X-a.HomeX) <= 7 && abs(a.Y-a.HomeY) <= 5 {
			home++
		}
		if a.Has("snShelter") {
			still++
		}
	}
	if home*2 < len(folk) || still*3 > len(folk) {
		t.Fatalf("after the storm %d of %d are home, %d still sheltering", home, len(folk), still)
	}
}
