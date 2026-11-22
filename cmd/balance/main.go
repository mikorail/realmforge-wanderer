// Balance tables: prints pacing and economy numbers computed from the real engine formulas.
// It evaluates formulas (Spawn, XPFor, price, travel pace, deadline windows, fares, ambush odds); it plays nothing.
// go run ./cmd/balance [world.json]   (default public/worlds/hollow-crown.json)
package main

import (
	"fmt"
	"math"
	"os"
	"strings"

	"wanderer/engine"
)

func main() {
	path := "public/worlds/hollow-crown.json"
	if len(os.Args) > 1 {
		path = os.Args[1]
	}
	b, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}
	w, err := engine.ParseWorld(string(b))
	if err != nil {
		panic(err)
	}
	_ = engine.CharOptions(w)
	g := engine.NewGame(w, engine.CharSpec{Name: "Bal", Species: "human", Calling: "warrior", Attr: [6]int{17, 14, 17, 10, 12, 12}, Start: 0})
	r := engine.NewRng("balance")
	fmt.Printf("world %s, parasang %.0f km, day = %d minutes\n", path, g.BalKm(), engine.DayLen)

	safe("xp", func() { tableXP(g, r) })
	safe("hp", func() { tableHP(w, r) })
	safe("fights", func() { tableFights(w, r) })
	safe("survival", func() { tableSurvival(g, r) })
	safe("income", func() { tableIncome(g, r) })
	safe("deadlines", func() { tableDeadlines(g) })
	safe("caravans", func() { tableCaravans(g) })
	safe("ambush", func() { tableAmbush(g) })
	safe("healing", func() { tableHealing(w) })
	safe("powers", tablePowers)
	safe("aging", tableAging)
}

// safe runs one table and reports a panic instead of losing the rest.
func safe(name string, f func()) {
	defer func() {
		if e := recover(); e != nil {
			fmt.Printf("\n!! table %s failed: %v\n", name, e)
		}
	}()
	f()
}

func head(s string) { fmt.Printf("\n== %s\n", s) }

// ---------------------------------------------------------------- XP

func tableXP(g *engine.Game, r *engine.Rng) {
	head("XP per kill vs XP needed (victim XP = 10 + 5 L^2; tier penalty x0.5 one tier below, x0.1 two, 0 more)")
	fmt.Println("lvl  need  kill@L  kills/lvl | kill@L-3 kills | kill@L-5 kills | kill@L-10 kills")
	for _, l := range []int{1, 2, 3, 5, 8, 10, 13, 16, 20, 25, 30} {
		g.P.Level = l
		need := engine.XPFor(l)
		row := fmt.Sprintf("%3d %5d", l, need)
		for i, d := range []int{0, 3, 5, 10} {
			v := g.Spawn("rat", 1, r)
			v.Level = maxi(1, l-d)
			v.XP = 10 + v.Level*v.Level*5
			xp := g.BalKillXP(v)
			k := "inf"
			if xp > 0 {
				k = fmt.Sprintf("%.1f", float64(need)/float64(xp))
			}
			if i == 0 {
				row += fmt.Sprintf(" %7d %9s |", xp, k)
			} else {
				row += fmt.Sprintf(" %8d %5s |", xp, k)
			}
		}
		fmt.Println(row)
	}
	g.P.Level = 1
	fmt.Println("flat XP rewards as % of the level need (guard trip 30, raid won 20+6L, incursion 60+10L, deadline 20+4L, discovery 10*tier):")
	for _, l := range []int{1, 3, 5, 10, 20} {
		need := float64(engine.XPFor(l))
		fmt.Printf("  L%-2d need %5.0f: guard %4.0f%%  raid %4.0f%%  incursion %4.0f%%  deadline %4.0f%%  discovery %4.0f%%\n", l, need,
			100*30/need, 100*float64(20+6*l)/need, 100*float64(60+10*l)/need, 100*float64(20+4*l)/need, 100*float64(10*(1+l/5))/need)
	}
}

// ---------------------------------------------------------------- build helpers

type build struct {
	name   string
	spec   engine.CharSpec
	skill  string
	key    int // attribute that gets the attribute points
	weapon string
}

var builds = []build{
	{"bruiser (warrior STR19 AGI14 TOU18)", engine.CharSpec{Name: "A", Species: "human", Calling: "warrior", Attr: [6]int{17, 14, 17, 10, 12, 12}}, "longblade", 0, "longblade"},
	{"duelist (rogue STR12 AGI20 TOU12)", engine.CharSpec{Name: "B", Species: "human", Calling: "rogue", Attr: [6]int{12, 18, 12, 14, 12, 13}}, "shortblade", 1, "shortblade"},
	{"scholar (INT20 TOU12 STR12)", engine.CharSpec{Name: "C", Species: "human", Calling: "scholar", Attr: [6]int{12, 14, 12, 18, 12, 12}}, "shortblade", 3, "shortblade"},
}

func tierFor(l int) int { return clamp(l/3, 0, 5) }

// levelled builds a character at level l with the real gainXP, gear of tier l/3 in every slot, key skill 1 + l/5.
func levelled(w *engine.WorldData, b build, l int) *engine.Game {
	spec := b.spec
	g := engine.NewGame(w, spec)
	r := engine.NewRng("bal:" + spec.Name)
	for g.P.Level < l {
		g.BalGainXP(engine.XPFor(g.P.Level) - g.XP)
	}
	g.P.Attr[b.key] += g.AP
	g.AP = 0
	g.Skills[b.skill] = 1 + l/5
	t := tierFor(l)
	handDone := false
	for _, p := range g.P.Body {
		p.Item = nil
		switch p.Slot {
		case "hand":
			if !handDone {
				p.Item = g.MakeWeapon(b.weapon, t, r)
				handDone = true
			}
		case "body", "head", "feet", "back", "hands", "face":
			p.Item = g.MakeArmor(p.Slot, t, r)
		}
	}
	g.P.HP = g.P.MaxHP
	return g
}

func clamp(v, lo, hi int) int { return maxi(lo, mini(hi, v)) }
func maxi(a, b int) int {
	if a > b {
		return a
	}
	return b
}
func mini(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func pHit(bonus, dv int) float64 {
	n := 1.0 // natural 20
	for roll := 2; roll <= 19; roll++ {
		if roll+bonus >= dv {
			n++
		}
	}
	return n / 20
}

// dmgPerAction is the expected damage of one action of a against d (all weapons, no crits, no specials).
func dmgPerAction(g *engine.Game, a, d *engine.Actor, skill int, isPlayer bool) float64 {
	tot := 0.0
	for _, wp := range a.Weapons() {
		sk := skill
		if !isPlayer {
			sk = a.Level / 5
		}
		pv := wp.PV + mini(a.M(engine.STR), wp.MaxStr)
		if isPlayer && sk >= 3 {
			pv++
		}
		ds := maxi(wp.DS, 1)
		avg := float64(maxi(wp.DN, 1))*(float64(ds)+1)/2 + float64(wp.DB)
		tot += pHit(a.M(engine.AGI)+sk, d.DV()) * engine.BalPenExpect(pv, d.AV()) * avg
	}
	return tot
}

func speedOf(a *engine.Actor) float64 { return math.Max(float64(a.Quick()), 20) / 100 }

// ---------------------------------------------------------------- HP

func tableHP(w *engine.WorldData, r *engine.Rng) {
	head("Player HP by level (real gainXP; TOU floor fix: at least 3 HP per level) vs enemy HP of an on-level template")
	fmt.Println("lvl | " + strings.Join(func() []string {
		var s []string
		for _, b := range builds {
			s = append(s, strings.Fields(b.name)[0])
		}
		return s
	}(), " | ") + " | wolf-like(lvl3 base)  bear(lvl5 base)  boss(chief L+3)")
	g0 := engine.NewGame(w, builds[0].spec)
	for _, l := range []int{1, 3, 5, 8, 10, 15, 20, 25, 30} {
		row := fmt.Sprintf("%3d |", l)
		for _, b := range builds {
			g := levelled(w, b, l)
			row += fmt.Sprintf(" %4d(+%d TOU%d) |", g.P.MaxHP, g.P.Level, g.P.Score(engine.TOU))
		}
		wolf := g0.Spawn("wolf", l, r)
		bear := g0.Spawn("bear", l, r)
		boss := g0.Spawn("chief", l+3, r)
		fmt.Printf("%s %d  %d  %d\n", row, wolf.MaxHP, bear.MaxHP, boss.MaxHP)
	}
}

// ---------------------------------------------------------------- fights

func tableFights(w *engine.WorldData, r *engine.Rng) {
	head("Hits to kill, by level pair (1v1, no crits/specials; gear tier = L/3 for both sides; loss% = share of player HP lost per fight)")
	fmt.Println("build | L | foes(avg of non-boss templates lvl L-3..L, fitting surface/dungeon) | player actions to kill | foe actions to kill player | loss%  | worst template (loss%)")
	for _, b := range builds {
		for _, l := range []int{1, 2, 3, 5, 8, 10, 15, 20, 25, 30} {
			g := levelled(w, b, l)
			sumP, sumF, sumLoss, n := 0.0, 0.0, 0.0, 0
			worstName, worst := "", 0.0
			for i := range engine.Templates {
				t := &engine.Templates[i]
				if t.Has("boss") || t.Lvl > l || t.Lvl < maxi(1, mini(l-3, 7)) || !strings.ContainsAny(t.Where, "sdr") {
					continue
				}
				e := g.Spawn(t.ID, l, r)
				mine := dmgPerAction(g, g.P, e, g.Skills[b.skill], true)
				theirs := dmgPerAction(g, e, g.P, 0, false) * speedOf(e) / speedOf(g.P)
				if mine <= 0 {
					mine = 0.01
				}
				ttk := float64(e.MaxHP) / mine
				loss := ttk * theirs / float64(g.P.MaxHP) * 100
				pk := 999.0
				if theirs > 0 {
					pk = math.Min(999, float64(g.P.MaxHP)/theirs/speedOf(e)*speedOf(g.P))
				}
				sumP += ttk
				sumF += pk
				sumLoss += loss
				n++
				if loss > worst {
					worst, worstName = loss, t.ID
				}
			}
			if n == 0 {
				continue
			}
			fmt.Printf("%-9s L%-2d n=%-2d  %5.1f   %6.1f   %5.0f%%   %s (%.0f%%)\n", strings.Fields(b.name)[0], l, n, sumP/float64(n), sumF/float64(n), sumLoss/float64(n), worstName, worst)
		}
	}
	fmt.Println("boss (chief, 1.5x HP, spawned L+3) against the bruiser:")
	for _, l := range []int{1, 3, 5, 8, 10, 15, 20, 25, 30} {
		g := levelled(w, builds[0], l)
		e := g.Spawn("chief", l+3, r)
		mine := dmgPerAction(g, g.P, e, g.Skills["longblade"], true)
		theirs := dmgPerAction(g, e, g.P, 0, false) * speedOf(e) / speedOf(g.P)
		fmt.Printf("  L%-2d boss L%-2d HP %4d AV %2d DV %2d | you need %5.1f actions, it needs %5.1f | loss %3.0f%% of %d HP\n", l, l+3, e.MaxHP, e.AV(), e.DV(),
			float64(e.MaxHP)/maxf(mine, 0.01), float64(g.P.MaxHP)/maxf(theirs, 0.01), float64(e.MaxHP)/maxf(mine, 0.01)*theirs/float64(g.P.MaxHP)*100, g.P.MaxHP)
	}
}

func maxf(a, b float64) float64 { return math.Max(a, b) }

// ---------------------------------------------------------------- survival

func tableSurvival(g *engine.Game, r *engine.Rng) {
	head("Food and water per day (hunger +1 and thirst +1 every 2+Endurance minutes; ration = MakeFood Power, dram -450 thirst)")
	food := g.MakeFood(r, 1)
	water := g.MakeWater()
	merchant := g.Spawn("bandit", 1, r)
	bf, sf := g.BalPrice(food, true, merchant), g.BalPrice(food, false, merchant)
	bw, sw := g.BalPrice(water, true, merchant), g.BalPrice(water, false, merchant)
	fmt.Printf("ration: buy %d sell %d (Value %d, restores %d) | water skin: buy %d sell %d (%d drams) => %.1f gold a dram\n", bf, sf, food.Value, food.Power, bw, sw, water.Charges, float64(bw)/float64(water.Charges))
	fmt.Println("endurance | ticks/day | rations/day | drams/day | gold/day (buying) | hours to Hungry (1200) | to Famished (2400) | to Thirsty (900)")
	for e := 0; e <= 4; e++ {
		ticks := float64(engine.DayLen) / float64(2+e)
		rat, dr := ticks/float64(food.Power), ticks/450
		gold := rat*float64(bf) + dr*float64(bw)/float64(water.Charges)
		fmt.Printf("   %d      |  %5.0f    |    %.2f     |   %.2f    |      %5.1f        |      %5.1f             |      %5.1f            |     %5.1f\n", e, ticks, rat, dr, gold,
			1200*float64(2+e)/60, 2400*float64(2+e)/60, 900*float64(2+e)/60)
	}
	fmt.Println("start pack: 4 rations + one 8-dram skin + 40-70 gold: days of food:", fmt.Sprintf("%.1f food-days, %.1f water-days", float64(4*food.Power)/720, 8*450/720.0))
}

// ---------------------------------------------------------------- income

func tableIncome(g *engine.Game, r *engine.Rng) {
	head("Prices (real price(): buy x1.4 less Ego/Persuasion/rep, sell x0.4 plus the same, floor 0.75 / cap 0.8)")
	merchant := g.Spawn("bandit", 1, r)
	fmt.Println("tier | weapon buy / sell | body armour buy / sell | ranged buy")
	for t := 0; t <= 5; t++ {
		wp := g.MakeWeapon("longblade", t, r)
		ar := g.MakeArmor("body", t, r)
		rg := g.MakeRanged(t, r)
		fmt.Printf("  %d  | %4d / %-4d | %4d / %-4d | %4d\n", t, g.BalPrice(wp, true, merchant), g.BalPrice(wp, false, merchant), g.BalPrice(ar, true, merchant), g.BalPrice(ar, false, merchant), g.BalPrice(rg, true, merchant))
	}
	tonic := g.MakeTonic("heal", 1)
	fmt.Printf("heal tonic buy %d sell %d | tent value 60 | house 300+300*cityTier | training 80*(rank+1)^2 | ships 260..4800 | priest full heal 10+3L (L10 = 40, L20 = 70) | ship wage 1/head/day\n",
		g.BalPrice(tonic, true, merchant), g.BalPrice(tonic, false, merchant))
	fmt.Println("average sell price of a random Loot() item by tier (500 draws):")
	for t := 0; t <= 5; t++ {
		sum := 0
		for i := 0; i < 500; i++ {
			it := g.Loot(r, t)
			sum += g.BalPrice(it, false, merchant)
		}
		fmt.Printf("  tier %d: %.1f gold\n", t, float64(sum)/500)
	}

	head("Gold per day of work vs the food bill (food+water about 7-9 gold a day at Endurance 0)")
	bill := foodBill(g, r, merchant)
	fmt.Printf("food+water bill: %.1f gold/day\n", bill)
	km := g.BalKm()
	fmt.Println("kind | gold | days | gold/day | net of food | note")
	for _, d := range []int{3, 5, 8} {
		days := BalContractDaysWalk(km, d)
		fee := float64(engine.BalGuardFee(d))
		fmt.Printf("tq guard %d parasangs | %.0f | %.1f walking | %.1f | %+.1f | allowance %.1f days\n", d, fee, days, fee/days, fee/days-bill, engine.BalContractDays(d))
	}
	for _, kmv := range []float64{80, 160, 320} {
		days := kmv / 24
		pay := float64(engine.BalCaravanGuardPay(kmv))
		fare := float64(engine.BalCaravanFare(kmv))
		walk := kmv / 30
		fmt.Printf("caravan guard %.0f km | %.0f | %.1f | %.1f | %+.1f | passenger fare %.0f (walking 30 km/d: %.1f days, %.0f food)\n", kmv, pay, days, pay/days, pay/days-bill, fare, walk, walk*bill)
	}
	for _, kind := range []string{"plague", "siege", "hostage", "caravan", "flood", "festival"} {
		for _, l := range []int{1, 5, 10, 20} {
			if l != 1 && l != 10 {
				continue
			}
			rew := float64(engine.BalDeadlineReward(kind, l))
			days := g.BalDeadlineWindowDays(kind, 4)
			fmt.Printf("deadline %-8s L%-2d (4 parasangs) | %.0f | window %.1f | %.1f | %+.1f | quest reward for comparison 60+25L = %d\n", kind, l, rew, days, rew/days, rew/days-bill, 60+25*l)
		}
	}
	fmt.Println("kills: outlaw gold 0..6L (avg 3L), boss 20..60 x L (avg 40L), elite +8L:")
	for _, l := range []int{1, 5, 10, 20} {
		fmt.Printf("  L%-2d outlaw %d  boss %d  elite +%d   | quest 60+25L = %d\n", l, 3*l, 40*l, 8*l, 60+25*l)
	}
	fmt.Println("trade ledger run (buy at 0.55 x base x1.1, sell at 1.4 x base x0.9; weight limit = capacity 20+3*STR less 30 for kit), 5 parasangs (about 27 days):")
	for _, gd := range []struct {
		name string
		base int
		wt   int
	}{{"grain", 6, 2}, {"iron", 16, 3}, {"wine", 24, 3}, {"spice", 40, 1}, {"silk", 55, 1}, {"relics", 90, 1}} {
		unit := float64(gd.base) * (1.4*0.9 - 0.55*1.1)
		units := float64(20+3*g.P.Score(engine.STR)-30) / float64(gd.wt)
		capital := units * float64(gd.base) * 0.55 * 1.1
		fmt.Printf("  %-7s profit/unit %5.1f, %3.0f units, capital %5.0f, profit %6.0f per run, %5.1f/day\n", gd.name, unit, units, capital, unit*units, unit*units/27)
	}
}

func foodBill(g *engine.Game, r *engine.Rng, merchant *engine.Actor) float64 {
	food, water := g.MakeFood(r, 1), g.MakeWater()
	ticks := float64(engine.DayLen) / 2
	return ticks/float64(food.Power)*float64(g.BalPrice(food, true, merchant)) + ticks/450*float64(g.BalPrice(water, true, merchant))/float64(water.Charges)
}

// BalContractDaysWalk is how many days walking d parasangs takes on grassland.
func BalContractDaysWalk(km float64, d int) float64 { return float64(d) * km / 30 }

// ---------------------------------------------------------------- deadlines

func tableDeadlines(g *engine.Game) {
	head("Deadline window vs days to walk (window = (walk at 22 km/d x 1.35 + min + 1.5) days; caravan x 0.75)")
	km := g.BalKm()
	paces := []struct {
		n string
		v float64
	}{{"road 36", 36}, {"grass 30", 30}, {"forest 24", 24}, {"desert 22", 22}, {"tundra 18", 18}, {"rainforest 15", 15}, {"wetland 14", 14}, {"mountain forest 12", 12}, {"glacier 10", 10}}
	fmt.Printf("%-9s %2s %8s |", "kind", "ps", "window")
	for _, p := range paces {
		fmt.Printf(" %-15s|", p.n)
	}
	fmt.Println(" (walk days / slack ratio)")
	for _, k := range engine.BalDeadlineKinds() {
		if k.ID == "festival" || k.ID == "vault" || k.ID == "relic" {
			continue
		}
		for _, ps := range []float64{2, 4, 8, 12} {
			wd := g.BalDeadlineWindowDays(k.ID, ps)
			fmt.Printf("%-9s %2.0f %7.1fd |", k.ID, ps, wd)
			for _, p := range paces {
				walk := ps * km / p.v
				fmt.Printf(" %5.1fd x%-4.2f   |", walk, wd/walk)
			}
			fmt.Println()
		}
	}
	fmt.Println("trade contracts (haul/guard): allowance vs walking on grass at 30 km/d")
	for _, d := range []int{3, 5, 8, 12, 20} {
		fmt.Printf("  %2d parasangs: allowance %5.1f days, walk %5.1f days, ratio %.2f\n", d, engine.BalContractDays(d), BalContractDaysWalk(km, d), engine.BalContractDays(d)/BalContractDaysWalk(km, d))
	}
	fmt.Println("one parasang crossing, days by biome pace:")
	for _, p := range paces {
		fmt.Printf("  %-16s %.1f days\n", p.n, math.Max(240/1440.0, km/p.v))
	}
}

// ---------------------------------------------------------------- caravans

func tableCaravans(g *engine.Game) {
	head("Caravan fares vs walking (wagon 24 km/day)")
	for _, km := range []float64{80, 160, 320, 640} {
		fmt.Printf("  %.0f km: wagon %.1f days, fare %d, guard pay %d; walking on grass %.1f days\n", km, km/24, engine.BalCaravanFare(km), engine.BalCaravanGuardPay(km), km/30)
	}
	fmt.Println("  road events: 22% of moving days raiders (2-6 dmg or +20+6L xp), 18% flavour; tq road events 3.5% per crossing")
}

// ---------------------------------------------------------------- ambush

func tableAmbush(g *engine.Game) {
	head("Ambush frequency")
	fmt.Println("waylaid on the road per 100 world crossings (parasang crossings, not in towns):")
	fmt.Println("  biome pace | days | surv0 | surv1 | surv2 | surv3 | lawless surv0")
	km := g.BalKm()
	for _, pace := range []float64{36, 30, 24, 15, 12} {
		cost := int(math.Max(240, km/pace*1440))
		fmt.Printf("  %5.0f km/d | %4.1f | %4.0f | %4.0f | %4.0f | %4.0f | %4.0f\n", pace, float64(cost)/1440,
			100*engine.BalAmbush(0, false, cost), 100*engine.BalAmbush(1, false, cost), 100*engine.BalAmbush(2, false, cost), 100*engine.BalAmbush(3, false, cost), 100*engine.BalAmbush(0, true, cost))
	}
	en, rest, cd := engine.BalCounterOdds()
	fmt.Printf("counter squads (needs a grudge and a known habit, level 3+, surface): %d%% per zone entry, %d%% per camp rest, then %d to %d quiet days\n", en, rest, cd, cd+3)
	for _, perDay := range []float64{6, 12, 24} {
		cool := (float64(cd) + 1.5) * perDay
		p := float64(en) / 100
		per100 := 100 / (1/p + cool)
		fmt.Printf("  at %2.0f zone entries a day: about %.1f squads per 100 entries (%.1f per 10 days)\n", perDay, per100, per100*perDay/10)
	}
}

// ---------------------------------------------------------------- healing

func tableHealing(w *engine.WorldData) {
	head("Healing: natural regen every 40 - 4*TOUmod minutes (min 6) heals 1 + MaxHP/80; bed/inn every 15, tent 25, camp 30; hunger runs +1 per 2 min meanwhile")
	fmt.Println("build | L | MaxHP | regen min per tick | HP per tick | hours to heal 50% awake | in bed | priest cost for full heal")
	for _, b := range builds {
		for _, l := range []int{1, 5, 10, 20, 30} {
			g := levelled(w, b, l)
			per := engine.BalRegenPeriod(g.P.M(engine.TOU))
			half := float64(g.P.MaxHP) / 2
			step := float64(engine.BalRegenStep(g.P.MaxHP))
			fmt.Printf("%-9s L%-2d %4d  %3d  +%.0f  %6.1f h  %5.1f h  %d\n", strings.Fields(b.name)[0], l, g.P.MaxHP, per, step, half/step*float64(per)/60, half/step*float64(engine.BalBedHealEvery("inn"))/60, 10+3*l)
		}
	}
}

// ---------------------------------------------------------------- powers

func tablePowers() {
	head("Active powers: cooldown in minutes (one action = one minute) vs a typical fight of about 8 actions")
	var parts []string
	for _, p := range engine.PowerDefs {
		if p.Active && !p.Defect {
			parts = append(parts, fmt.Sprintf("%s %dpt/cd%d", p.ID, p.Cost, p.Cool))
		}
	}
	fmt.Println(strings.Join(parts, ", "))
}

// ---------------------------------------------------------------- aging

func tableAging() {
	head("Aging: human lifespan 72, start at 30-36% of it, 112-day year; stage thresholds 28/55/75/92% (pace 1, x4, x12)")
	for _, st := range []struct {
		n string
		f float64
	}{{"prime", 0.28}, {"mature", 0.55}, {"old", 0.75}, {"venerable", 0.92}} {
		yrs := (st.f - 0.33) * 72
		if yrs < 0 {
			yrs = 0
		}
		days := yrs * 112
		fmt.Printf("  %-9s after %5.1f years = %5.0f days at pace 1, %5.0f at x4, %4.0f at x12\n", st.n, yrs, days, days/4, days/12)
	}
}
