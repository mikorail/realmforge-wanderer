package engine

import "testing"

func TestTvCrossingTakesDays(t *testing.T) {
	g := newGame(t, nil)
	p := g.W.Idx(g.WX, g.WY)
	min := g.tvMinutes(p)
	if min < DayLen*2 || min > DayLen*20 {
		t.Fatalf("crossing a %v km region took %d minutes, want days", g.tvKm(), min)
	}
	g.Skills["survival"] = 0
	g.W.Road[p], g.W.Height[p] = 1, 20
	road := g.tvMinutes(p)
	g.W.Road[p], g.W.Height[p] = 0, 80
	mount := g.tvMinutes(p)
	if road >= mount {
		t.Errorf("road %d should beat a mountain crossing %d", road, mount)
	}
}

func TestTvJourneyEatsFromPack(t *testing.T) {
	g := newGame(t, nil)
	food := g.MakeFood(g.R, 1)
	food.Count = 10
	g.give(food)
	g.Hunger, g.Thirst = 0, 0
	t0 := g.Turn
	g.tvJourney(5 * DayLen)
	if g.Turn-t0 != 5*DayLen {
		t.Fatalf("journey passed %d turns", g.Turn-t0)
	}
	if food.Count >= 10 {
		t.Errorf("five days on the road ate nothing from the pack")
	}
	if g.Hunger > 2400 {
		t.Errorf("starving with food in the pack: hunger %d", g.Hunger)
	}
}

func TestTvDuration(t *testing.T) {
	for min, want := range map[int]string{90: "1 hour", 300: "5 hours", DayLen: "1 day", DayLen + 360: "1 day, 6 hours", 5 * DayLen: "5 days"} {
		if got := tvDuration(min); got != want {
			t.Errorf("tvDuration(%d) = %q, want %q", min, got, want)
		}
	}
}
