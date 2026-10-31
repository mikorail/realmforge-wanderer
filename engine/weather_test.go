package engine

import (
	"os"
	"strings"
	"testing"
)

func TestSeasonFromDay(t *testing.T) {
	cases := map[int]int{0: 0, SeasonDays - 1: 0, SeasonDays: 1, 2 * SeasonDays: 2, 3*SeasonDays + 5: 3, YearDays: 0, YearDays + SeasonDays: 1}
	for day, want := range cases {
		if got := SeasonOfDay(day); got != want {
			t.Errorf("day %d: season %d, want %d", day, got, want)
		}
	}
	g := newGame(t, nil)
	g.Turn = 0
	if g.Season() != 0 || !strings.Contains(g.timeOfDay(), "day 1") {
		t.Errorf("game should open in spring on day 1: %d %s", g.Season(), g.timeOfDay())
	}
	g.Turn = (2*SeasonDays+3)*DayLen - startMinutes
	if g.Season() != 2 {
		t.Errorf("season at day %d = %s", g.Day(), SeasonNames[g.Season()])
	}
	// winter is colder than summer at the same place and hour
	w := g.W
	p := w.Idx(g.WX, g.WY)
	summer := w.TempAt(p, (SeasonDays+14)*DayLen)
	winter := w.TempAt(p, (3*SeasonDays+14)*DayLen)
	if winter >= summer {
		t.Errorf("winter %d° not colder than summer %d°", winter, summer)
	}
}

func TestWeatherScheduleDeterministic(t *testing.T) {
	w := world(t, "ember-crown")
	w2, err := ParseWorld(mustRead(t, "ember-crown"))
	if err != nil {
		t.Fatal(err)
	}
	kinds := map[string]int{}
	changes := 0
	for p := 0; p < len(w.Biome); p += 37 {
		prev := ""
		for b := 0; b < 200; b++ {
			turn := b * wxBucket
			a := w.WeatherAt(p, turn)
			if c := w2.WeatherAt(p, turn); c.Kind != a.Kind || c.Temp != a.Temp {
				t.Fatalf("schedule differs between loads at p=%d turn=%d", p, turn)
			}
			if e := w.WeatherAt(p, turn+wxBucket-1); e.Kind != a.Kind {
				t.Fatalf("weather changed inside a bucket at p=%d turn=%d", p, turn)
			}
			kinds[a.Kind.ID]++
			if prev != "" && prev != a.Kind.ID {
				changes++
			}
			prev = a.Kind.ID
		}
	}
	if len(kinds) < 5 || changes == 0 || kinds["clear"] == 0 {
		t.Errorf("weather too uniform: %v (%d changes)", kinds, changes)
	}
	t.Logf("kinds: %v, changes %d", kinds, changes)
	// the cyberpunk world gets its acid rain
	nt := world(t, "neon-tide")
	genre := map[string]int{}
	for p := 0; p < len(nt.Biome); p += 23 {
		for b := 0; b < 200; b++ {
			genre[nt.WeatherAt(p, b*wxBucket).Kind.ID]++
		}
	}
	if genre["acidrain"] == 0 && genre["smog"] == 0 {
		t.Errorf("no genre weather in a cyberpunk world: %v", genre)
	}
}

func mustRead(t *testing.T, name string) string {
	t.Helper()
	b, err := os.ReadFile("../public/worlds/" + name + ".json")
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// openField turns the current zone into an open grass field with nobody else in it.
func openField(g *Game) {
	z := g.Cur
	for i := range z.T {
		z.T[i] = TGrass
		z.Fire[i] = 0
	}
	z.Dark, z.Underwater, z.Sky = false, false, false
	z.Actors = nil
	g.Turn = 4 * 60 // noon
}

func visCount(g *Game) int {
	g.recompute()
	n := 0
	for _, v := range g.vis {
		if v {
			n++
		}
	}
	return n
}

func TestFogReducesSight(t *testing.T) {
	g := newGame(t, nil)
	openField(g)
	g.wxForce = &Weather{Kind: WeatherByID("clear"), Temp: 15}
	clear := visCount(g)
	g.wxForce = &Weather{Kind: WeatherByID("fog"), Temp: 10}
	fog := visCount(g)
	if fog*3 > clear || fog < 20 {
		t.Errorf("fog sight %d vs clear %d", fog, clear)
	}
	// indoors under a roof the fog does not matter
	g.Cur.Set(g.P.X, g.P.Y, TFloor)
	if in := visCount(g); in != clear {
		t.Errorf("roofed tile still fogged: %d vs %d", in, clear)
	}
	// underground: no weather at all
	g.Cur.Set(g.P.X, g.P.Y, TGrass)
	g.Cur.Z = 1
	if g.activeWeather() != nil {
		t.Errorf("weather underground")
	}
	g.Cur.Z = 0
	if s := g.weatherLine(); !strings.Contains(s, "Fog") {
		t.Errorf("side panel line: %q", s)
	}
}

func TestRainDousesFire(t *testing.T) {
	g := newGame(t, nil)
	openField(g)
	g.wxForce = &Weather{Kind: WeatherByID("rain"), Temp: 12}
	fi := (g.P.Y+3)*MW + g.P.X + 5
	roof := (g.P.Y+3)*MW + g.P.X + 8
	g.Cur.T[roof] = TFloor
	g.Cur.Fire[fi], g.Cur.Fire[roof] = 200, 200
	g.P.AddSt("burning", 100)
	r := g.R.S
	for i := 0; i < 80; i++ {
		g.Turn++
		g.weatherTick()
	}
	if g.R.S != r {
		t.Errorf("weather drew from the game rng")
	}
	if g.Cur.Fire[fi] != 0 {
		t.Errorf("rain did not put out the fire")
	}
	if g.Cur.Fire[roof] == 0 {
		t.Errorf("rain put out a fire under a roof")
	}
	if g.P.St("burning") > 0 {
		t.Errorf("rain did not put out the burning player")
	}
}

func TestWeatherChangeMessageAndSave(t *testing.T) {
	g := newGame(t, nil)
	openField(g)
	g.wxForce = &Weather{Kind: WeatherByID("clear"), Temp: 12}
	g.weatherTick()
	g.wxForce = &Weather{Kind: WeatherByID("snow"), Temp: -3}
	g.weatherTick()
	if last := g.Msgs[len(g.Msgs)-1].T; !strings.Contains(last, "Snow") {
		t.Errorf("no weather change message: %q", last)
	}
	// snow slows you down
	if c := g.weatherStep(g.P.X+1, g.P.Y, 1000); c <= 1000 {
		t.Errorf("snow did not slow movement: %d", c)
	}
	g.wxForce = &Weather{Kind: WeatherByID("storm"), Temp: 12}
	if g.weatherMissile(g.P) <= 0 {
		t.Errorf("storm does not hinder shots")
	}
	g.wxForce = nil
	g2, err := LoadGame(g.W, g.Save())
	if err != nil {
		t.Fatal(err)
	}
	if g2.WxLast != g.WxLast || g2.WeatherHere().Kind != g.WeatherHere().Kind {
		t.Errorf("weather state lost in save/load")
	}
}

func TestColdMakesHungry(t *testing.T) {
	g := newGame(t, nil)
	openField(g)
	g.wxForce = &Weather{Kind: WeatherByID("frost"), Temp: -20}
	h := g.Hunger
	for i := 0; i < 60; i++ {
		g.Turn++
		g.weatherTick()
	}
	if g.Hunger <= h {
		t.Errorf("cold did not raise hunger")
	}
	g.wxForce = &Weather{Kind: WeatherByID("heat"), Temp: 38}
	th := g.Thirst
	for i := 0; i < 60; i++ {
		g.Turn++
		g.weatherTick()
	}
	if g.Thirst <= th {
		t.Errorf("heat did not raise thirst")
	}
}
