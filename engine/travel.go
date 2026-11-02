package engine

import (
	"fmt"
	"math"
)

// Overland travel at true scale (prefix tv). A parasang is a real region (about 160 km on the
// default Realmforge map), so a world-map step takes days, not hours: pace depends on roads,
// terrain, wings and Survival, and food and water are drawn from the pack along the way.

// tvDefaultKm is the parasang width on Realmforge's default map (72° of latitude over 50 rows).
const tvDefaultKm = 160

// tvKm is the width of one parasang in km.
func (g *Game) tvKm() float64 {
	if g.W.KmPer > 0 {
		return g.W.KmPer
	}
	return tvDefaultKm
}

// tvBasePace is km per day on foot through biome b, before roads and mountains.
func tvBasePace(b int) float64 {
	switch b {
	case 3, 4, 17, 18: // savanna, grassland, open terran / verdant
		return 30
	case 5, 6, 9: // seasonal / deciduous forest, taiga
		return 24
	case 8: // temperate rainforest
		return 20
	case 7: // tropical rainforest
		return 15
	case 12: // wetland
		return 14
	case 1, 2, 19: // deserts
		return 22
	case 10: // tundra
		return 18
	case 11, 21: // glacier
		return 10
	case 13, 14, 15, 16: // ashland, crystal fields, fungal forest, lava flats
		return 15
	}
	return 24
}

// tvPace is km per day across parasang p for the player.
func (g *Game) tvPace(p int) float64 {
	if g.P.Pow("wings") > 0 {
		return 90
	}
	pace := tvBasePace(g.W.Biome[p])
	if g.W.Road[p] > 0 {
		pace = math.Max(pace, 36)
	}
	if g.W.Height[p] >= 70 {
		pace *= 0.5
	}
	return pace * (1 + 0.08*float64(g.Skills["survival"])) * g.trPaceMul(p) // mounts and carts (transport.go)
}

// tvMinutes is how long crossing into parasang p takes (turns are minutes).
func (g *Game) tvMinutes(p int) int {
	return maxi(240, int(g.tvKm()/g.tvPace(p)*DayLen))
}

// tvDuration renders minutes of travel as "3 days" / "1 day, 6 hours" / "9 hours".
func tvDuration(min int) string {
	d, h := min/DayLen, (min%DayLen)/60
	plural := func(n int, w string) string {
		if n == 1 {
			return "1 " + w
		}
		return fmt.Sprintf("%d %ss", n, w)
	}
	switch {
	case d == 0:
		return plural(maxi(h, 1), "hour")
	case h == 0 || d >= 4:
		return plural(d, "day")
	}
	return plural(d, "day") + ", " + plural(h, "hour")
}

// tvProvision eats and drinks from the pack when hungry or thirsty on the road.
// It returns what was used so the arrival message can sum it up.
func (g *Game) tvProvision(ate, drank *int) {
	if g.Hunger > 900 {
		for _, it := range g.P.Inv {
			if it.Kind == "food" {
				g.Hunger = maxi(0, g.Hunger-maxi(it.Power, 300))
				g.consume(it)
				*ate++
				break
			}
		}
	}
	for g.Thirst > 600 {
		var src *Item
		for _, it := range g.lqContainers() {
			if lqKey(it) == "fresh" && it.Charges > 0 {
				src = it
				break
			}
		}
		if src == nil {
			return
		}
		src.Charges--
		g.lqRename(src)
		g.Thirst = maxi(0, g.Thirst-450)
		*drank++
	}
}

// tvPass passes cost minutes on the move (road, rail or air): hunger, thirst, meals, healing,
// rations from the pack and privation. It returns meals eaten, drams drunk and HP lost.
func (g *Game) tvPass(cost int) (ate, drank, hurt int) {
	for i := 0; i < cost; i++ {
		g.Turn++
		g.mealTick()
		if g.Turn%(2+g.Skills["endurance"]) != 0 && !g.slowHunger() {
			g.Hunger++
			g.Thirst++
		}
		if g.Turn%30 == 0 && g.P.HP < g.P.MaxHP {
			g.P.HP++
		}
		if g.Turn%120 == 0 {
			g.tvProvision(&ate, &drank)
		}
		// every half day without food or water wears you down
		if g.Turn%720 == 0 && (g.Hunger > 2400 || g.Thirst > 1800) && g.P.HP > 1 {
			g.P.HP--
			hurt++
		}
	}
	return
}

// tvJourney passes the time of crossing into the next parasang and reports the journey in one message.
func (g *Game) tvJourney(cost int) {
	ate, drank, hurt := g.tvPass(cost)
	msg := fmt.Sprintf("You travel %.0f km: %s on the way%s.", g.tvKm(), tvDuration(cost), g.trRiding())
	if ate > 0 || drank > 0 {
		msg += fmt.Sprintf(" {{K|(%d meals, %d drams of water from your pack)}}", ate, drank)
	}
	g.Msg(msg)
	if hurt > 0 {
		g.Msg("{{R|The road is hard without food and water.}}")
	}
	g.trJourney(cost) // mount feed, lameness, theft (transport.go)
}

// tvInfo is the world-map cursor's distance line.
func (g *Game) tvInfo(p int) []string {
	cx, cy := p%g.W.W, p/g.W.W
	if cx == g.WX && cy == g.WY {
		return []string{fmt.Sprintf("{{K|This region: %.0f km across, %s %s}}", g.tvKm(), tvDuration(g.tvMinutes(p)), g.trHow())}
	}
	dx, dy := float64(cx-g.WX), float64(cy-g.WY)
	km := math.Hypot(dx, dy) * g.tvKm()
	// estimate along the way at the pace of the destination and here
	pace := (g.tvPace(p) + g.tvPace(g.W.Idx(g.WX, g.WY))) / 2
	return []string{fmt.Sprintf("{{K|%.0f km away, about %s %s}}", km, tvDuration(int(km/pace*DayLen)), g.trHow())}
}
