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

func TestCityRingOuterWards(t *testing.T) {
	g := newGame(t, nil)
	w := g.W
	found := 0
	for i := range w.Burgs {
		bd := &w.Burgs[i]
		if !bd.Alive || bd.Layer != "surface" || w.cityTier[bd.ID] < 2 {
			continue
		}
		cp := g.cityPlanFor(bd)
		px, py := bd.P%w.W, bd.P/w.W
		for k, kind := range cp.ring {
			ox, oy := floorDiv3(k[0]), floorDiv3(k[1])
			zx, zy := k[0]-ox*3, k[1]-oy*3
			got, owner := g.districtAt(px+ox, py+oy, zx, zy)
			if owner != bd || got != kind {
				t.Fatalf("%s ward %v: districtAt gave %q of %v, want %q", bd.Name, k, got, owner, kind)
			}
			zn := g.GetZone(px+ox, py+oy, zx, zy, 0)
			if zn.Kind != "town" || zn.Burg != bd.ID {
				t.Fatalf("%s ward %v generated as %q (%s)", bd.Name, k, zn.Kind, zn.Name)
			}
			found++
		}
	}
	if found == 0 {
		t.Fatal("no great city has outer wards")
	}
	t.Logf("%d outer wards", found)
}
