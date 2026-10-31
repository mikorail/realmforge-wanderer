package engine

import "fmt"

// Survival states (WP3, prefix lq). The core counters stay (Hunger rises, Thirst rises); these
// give them the reference's named states and add the consequences.

func lqThirstState(th int) string {
	switch {
	case th >= 1800:
		return "Dehydrated"
	case th >= 1350:
		return "Parched"
	case th >= 900:
		return "Thirsty"
	}
	return "Quenched"
}

func lqHungerState(h int) string {
	switch {
	case h >= 2400:
		return "Famished"
	case h >= 1200:
		return "Hungry"
	}
	return "Sated"
}

// lqQuickPenalty: -10 Quickness while Famished (hook into Quick() if wanted).
func (g *Game) lqQuickPenalty() int {
	if g.Hunger >= 2400 {
		return -10
	}
	return 0
}

// lqSurvivalTick runs once per game minute (turnHook): auto-drink at Thirsty, the Parched
// warning, and the dehydration toughness save.
func lqSurvivalTick(g *Game) {
	p := g.P
	if p == nil || g.Dead {
		return
	}
	if g.Thirst == 1350 {
		g.Msg("{{O|Your mouth is dry and your head aches. You are parched.}}")
	}
	if g.Thirst >= 900 && g.Turn%3 == 0 && g.Pref("autodrink") && !g.nearHostile() {
		for _, it := range g.lqContainers() {
			if it.Charges > 0 && lqKey(it) == "fresh" {
				it.Charges--
				g.lqQuench(450)
				g.Msg(fmt.Sprintf("{{B|You take a drink from your %s. (%d left)}}", it.Name, it.Charges))
				break
			}
		}
	}
	if g.Thirst >= 1800 && g.Turn%5 == 0 {
		if g.R.Dice(1, 20)+p.M(TOU) < 12 {
			g.Msg("{{R|Dehydration wracks you.}}")
			g.hurt(p, 2, nil, "dehydration")
		}
	}
}

// ---- water as money ----

// lqDrams counts the fresh water you carry.
func (g *Game) lqDrams() int {
	n := 0
	for _, it := range g.lqContainers() {
		if lqKey(it) == "fresh" {
			n += it.Charges
		}
	}
	return n
}

// lqPay spends n drams of fresh water; false (and nothing taken) when you carry fewer.
func (g *Game) lqPay(n int) bool {
	if n <= 0 {
		return true
	}
	if g.lqDrams() < n {
		return false
	}
	for _, it := range g.lqContainers() {
		if lqKey(it) != "fresh" || it.Charges <= 0 {
			continue
		}
		t := mini(n, it.Charges)
		it.Charges -= t
		g.lqRename(it)
		n -= t
		if n == 0 {
			break
		}
	}
	return true
}

// lqGain pours n drams of fresh water into containers with room; returns how many fit.
func (g *Game) lqGain(n int) int {
	got := 0
	for _, it := range g.lqContainers() {
		if it.Charges > 0 && lqKey(it) != "fresh" {
			continue
		}
		t := mini(n-got, it.Power-it.Charges)
		if t > 0 {
			g.lqSet(it, "fresh", it.Charges+t)
			got += t
		}
	}
	return got
}

// lqPayMixed covers a price with gold first, then drams (1 dram = 1 coin). False, nothing
// taken, if you cannot cover it.
func (g *Game) lqPayMixed(price int) bool {
	if g.P.Gold >= price {
		g.P.Gold -= price
		return true
	}
	short := price - g.P.Gold
	if !g.lqPay(short) {
		return false
	}
	g.P.Gold = 0
	return true
}
