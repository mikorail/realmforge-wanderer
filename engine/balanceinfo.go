package engine

// Read-only views for cmd/balance (a table printer that evaluates the real formulas; it is not a bot).
// Nothing here changes the game; every function either calls the owning code or mirrors one expression and
// says which. Keep it in step when the owning constant moves.

// BalPenExpect is the expected number of penetrations of one attack (mirrors penetrate() in combat.go:
// up to 4 rounds of 3 rolls of d10-2+pv against av, another round only after three successes, pv-2 each round).
func BalPenExpect(pv, av int) float64 {
	total, chain := 0.0, 1.0
	for round := 0; round < 4; round++ {
		n := 0
		for d := 1; d <= 10; d++ {
			if d-2+pv > av {
				n++
			}
		}
		q := float64(n) / 10
		total += chain * 3 * q
		chain *= q * q * q
		pv -= 2
	}
	return total
}

// BalBasePace is km per day on foot through biome b before roads, mountains and Survival (travel.go).
func BalBasePace(b int) float64 { return tvBasePace(b) }

// BalKm is the width of one parasang in this world.
func (g *Game) BalKm() float64 { return g.tvKm() }

// BalGainXP awards XP through the real gainXP (level-ups, HP, points).
func (g *Game) BalGainXP(n int) { g.gainXP(n) }

// BalKillXP is what killing d would pay the player at the player's current level (leveling.go tiers).
func (g *Game) BalKillXP(d *Actor) int { return g.killXP(d, nil) }

// BalPrice is the real price() of an item at a merchant's stall, to buy or to sell.
func (g *Game) BalPrice(it *Item, buy bool, merchant *Actor) int {
	g.Talk = merchant
	v := g.price(it, buy)
	g.Talk = nil
	return v
}

// BalDeadlineKind is one deadline event kind: its id, minimum window in days and gold base.
type BalDeadlineKind struct {
	ID   string
	Min  int
	Base int
}

func BalDeadlineKinds() []BalDeadlineKind {
	var out []BalDeadlineKind
	for _, k := range dlKinds {
		out = append(out, BalDeadlineKind{k.ID, k.Min, dlRewardBase[k.ID]})
	}
	return out
}

// BalDeadlineWindowDays is the average window in days a deadline of this kind gets for a walk of the given
// parasangs (deadlines.go dl: window() with the average of its 0 to 3 day random part; caravans get 3/4).
func (g *Game) BalDeadlineWindowDays(kind string, parasangs float64) float64 {
	dpp := maxf(0.6, g.tvKm()/dlWalkKmPerDay)
	min := 0
	for _, k := range dlKinds {
		if k.ID == kind {
			min = k.Min
		}
	}
	w := parasangs*dpp*dlSlack + float64(min) + 1.5
	if kind == "caravan" {
		w = w * 3 / 4
	}
	return w
}

// BalDeadlineReward is the average gold of a deadline event at a level (base + base*lvl/8 + a third of base on average / 2).
func BalDeadlineReward(kind string, lvl int) int {
	b := dlRewardBase[kind]
	return b + b*lvl/8 + b/6
}

// BalContractDays is the average days a trade-ledger haul contract allows for d parasangs (tradeways.go tqOffers).
func BalContractDays(d int) float64 { return 11 - 2 + float64(d*tqDaysPerParasang) }

func BalGuardFee(d int) int             { return tqGuardFee(d) }
func BalCaravanFare(km float64) int     { return cvFare(km) }
func BalCaravanGuardPay(km float64) int { return cvGuardPay(km) }

// BalAmbush is the chance of a road ambush on a world-map crossing of cost minutes (game.go ambushChance).
func BalAmbush(survival int, lawless bool, cost int) float64 {
	return ambushChance(survival, lawless, cost)
}

// BalCounterOdds: percent per zone entry, percent per rest, and base cooldown days (counters.go).
func BalCounterOdds() (enter, rest, cooldown int) { return ctChanceEnter, ctChanceRest, ctCooldownDays }

// BalTravelMinutes is the time to cross into parasang p for the current player (travel.go).
func (g *Game) BalTravelMinutes(p int) int { return g.tvMinutes(p) }

// BalRegenPeriod is minutes per HP of natural healing (mirrors turn() in game.go without meals, powers or age).
func BalRegenPeriod(touMod int) int {
	r := 40 - touMod*4
	if r < 6 {
		r = 6
	}
	return r
}

// BalRegenStep is HP healed per natural-healing or sleep tick at this MaxHP (game.go regenStep).
func BalRegenStep(maxHP int) int { return 1 + maxHP/80 }

// BalBedHealEvery is minutes per HP when sleeping in a bed or inn, tent, camp (fatigue.go ijHealEvery).
func BalBedHealEvery(kind string) int { return ijHealEvery[kind] }
