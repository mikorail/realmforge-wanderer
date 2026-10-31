package engine

import (
	"fmt"
	"math"
)

// lqPriceFactor: Qud's Ego curve m = clamp(0.07*(EgoMod+repShift)+0.35, 0.05, 0.95), softened
// to a multiplier around 1 for price(): buying costs less and selling pays more as m rises.
func (g *Game) lqPriceFactor(buy bool) float64 {
	shift := 0
	if g.Talk != nil {
		switch rep := g.Rep[g.Talk.Faction]; {
		case rep >= 600:
			shift = 3
		case rep >= 250:
			shift = 1
		case rep <= -600:
			shift = -3
		case rep <= -250:
			shift = -1
		}
	}
	m := 0.07*float64(g.P.M(EGO)+shift) + 0.35
	m = math.Min(0.95, math.Max(0.05, m))
	if buy {
		return math.Min(1.5, math.Max(0.6, math.Sqrt(0.35/m)))
	}
	return math.Min(1.6, math.Max(0.7, math.Sqrt(m/0.35)))
}

// lqTalk: water trade with any trader. Buy water by the dram, sell what you carry.
func lqTalk(g *Game, a *Actor) []Option {
	if !(a.Role == "merchant" || a.Role == "smith" || a.Role == "apothecary" || a.Role == "exotic" || a.Role == "elder") {
		return nil
	}
	g.Talk = a
	var out []Option
	room := 0
	for _, it := range g.lqContainers() {
		if it.Charges == 0 || lqKey(it) == "fresh" {
			room += it.Power - it.Charges
		}
	}
	buyP := maxi(1, int(math.Round(2*g.lqPriceFactor(true))))
	sellP := maxi(1, int(math.Round(1*g.lqPriceFactor(false))))
	if room > 0 && g.P.Gold >= buyP {
		n := mini(room, g.P.Gold/buyP)
		out = append(out, Option{fmt.Sprintf("{{B|Buy water: fill up, %d drams for %d %s.}}", n, n*buyP, g.Sk.Currency), func() {
			got := g.lqGain(n)
			g.P.Gold -= got * buyP
			g.Msg(fmt.Sprintf("You buy %d drams of water for %d.", got, got*buyP))
			g.talkMenu(g.greeting(a))
		}})
	}
	if d := g.lqDrams(); d > 4 {
		n := d - 4
		out = append(out, Option{fmt.Sprintf("{{B|Sell water: %d drams (keeping 4) for %d %s.}}", n, n*sellP, g.Sk.Currency), func() {
			if g.lqPay(n) {
				g.P.Gold += n * sellP
				g.Msg(fmt.Sprintf("You sell %d drams of water for %d.", n, n*sellP))
			}
			g.talkMenu(g.greeting(a))
		}})
	}
	return out
}
