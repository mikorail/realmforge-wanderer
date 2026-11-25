package engine

import (
	"slices"
	"strings"
	"testing"
)

func TestCombatExploreExtRegistration(t *testing.T) {
	if !slices.Contains(unLocalKeys, "cc") {
		t.Fatal("expected cc in unLocalKeys")
	}
	if !slices.Contains(unLocalKeys, "sm") {
		t.Fatal("expected sm in unLocalKeys")
	}
	if _, ok := modeKeys["cc"]; !ok {
		t.Fatal("expected modeKeys[cc]")
	}
	if _, ok := modeDraws["cc"]; !ok {
		t.Fatal("expected modeDraws[cc]")
	}
	if _, ok := modeKeys["sm"]; !ok {
		t.Fatal("expected modeKeys[sm]")
	}
	if _, ok := modeDraws["sm"]; !ok {
		t.Fatal("expected modeDraws[sm]")
	}
}

func TestChemComboElementalReactions(t *testing.T) {
	g := newGame(t, nil)
	z := g.Cur
	if z == nil {
		t.Fatal("expected current zone")
	}

	spawnAt := func(name string, x, y int) *Actor {
		a := g.Spawn("bandit", 2, g.R)
		if a == nil {
			t.Fatal("spawn failed")
		}
		a.Name = name
		a.AI = "hostile"
		a.Angry = true
		a.MaxHP = 80
		a.HP = 80
		a.X, a.Y = x, y
		z.Set(x, y, TDirt)
		z.Actors = append(z.Actors, a)
		return a
	}

	// 1. Wet + Shock: chains lightning to adjacent wet foe.
	t1 := spawnAt("Wet Raider", 20, 10)
	t2 := spawnAt("Soaked Ally", 21, 10)
	t1.AddSt("wet", 5)
	t2.AddSt("wet", 5)
	hp1Before, hp2Before := t1.HP, t2.HP
	combo, bonus := ccApplyElementalCombo(g, t1, "shock", "")
	if combo != "wet_shock" || bonus <= 0 {
		t.Fatalf("expected wet_shock combo, got %q (%d)", combo, bonus)
	}
	if t1.HP >= hp1Before || t2.HP >= hp2Before {
		t.Fatalf("expected chain lightning damage to both targets: t1 %d->%d, t2 %d->%d", hp1Before, t1.HP, hp2Before, t2.HP)
	}
	if t1.St("dazed") <= 0 {
		t.Fatal("expected primary wet+shock target to be dazed")
	}

	// 2. Oil + Fire: ignites spreading flame tiles (z.Fire) and applies burning.
	t3 := spawnAt("Oiled Brigand", 26, 10)
	t3.AddSt("oiled", 6)
	z.Set(27, 10, TDirt)
	z.Liq[10*MW+27] = 2
	z.Fire[10*MW+26] = 0
	z.Fire[10*MW+27] = 0
	combo, bonus = ccApplyElementalCombo(g, t3, "fire", "")
	if combo != "oil_fire" || bonus <= 0 {
		t.Fatalf("expected oil_fire combo, got %q (%d)", combo, bonus)
	}
	if t3.St("burning") <= 0 {
		t.Fatal("expected oiled target to catch fire")
	}
	if z.Fire[10*MW+26] == 0 || z.Fire[10*MW+27] == 0 {
		t.Fatalf("expected spreading fire tiles, got center=%d adj=%d", z.Fire[10*MW+26], z.Fire[10*MW+27])
	}

	// 3. Frost + Heavy Bludgeon (cudgel/heavy): shatters frozen target + AoE shrapnel.
	t4 := spawnAt("Frozen Brute", 32, 10)
	t5 := spawnAt("Adjacent Flanker", 33, 10)
	t4.AddSt("frozen", 4)
	hp4Before, hp5Before := t4.HP, t5.HP
	combo, bonus = ccApplyElementalCombo(g, t4, "", "cudgel")
	if combo != "frost_shatter" || bonus <= 0 {
		t.Fatalf("expected frost_shatter combo, got %q (%d)", combo, bonus)
	}
	if t4.HP >= hp4Before || t5.HP >= hp5Before {
		t.Fatalf("expected shatter + shrapnel damage: t4 %d->%d, t5 %d->%d", hp4Before, t4.HP, hp5Before, t5.HP)
	}
	if t4.St("frozen") != 0 {
		t.Fatal("expected frozen status to be cleared upon shatter")
	}
	if t5.St("bleeding") <= 0 {
		t.Fatal("expected adjacent foe to bleed from ice shrapnel")
	}

	// 4. Smoke + Backstab/Unaware (shortblade): guaranteed critical + bleeding.
	t6 := spawnAt("Smoked Lookout", 38, 10)
	g.ccAddSmoke(z.Key, t6.X, t6.Y, 6)
	hp6Before := t6.HP
	combo, bonus = ccApplyElementalCombo(g, t6, "", "shortblade")
	if combo != "smoke_backstab" || bonus <= 0 {
		t.Fatalf("expected smoke_backstab combo, got %q (%d)", combo, bonus)
	}
	if t6.HP >= hp6Before || t6.St("bleeding") <= 0 {
		t.Fatalf("expected smoke backstab damage and bleeding: hp %d->%d bleed=%d", hp6Before, t6.HP, t6.St("bleeding"))
	}

	if g.cc().Combos < 4 {
		t.Fatalf("expected at least 4 recorded combos, got %d", g.cc().Combos)
	}
}

func TestCaptiveSubdueRansomAndRecruitOath(t *testing.T) {
	g := newGame(t, nil)
	z := g.Cur

	// Subdue a weakened foe and ransom them to a guard.
	foe := g.Spawn("bandit", 3, g.R)
	foe.Name = "Captain Varek"
	foe.Proper = true
	foe.AI = "hostile"
	foe.Faction = "bandits"
	foe.MaxHP = 50
	foe.HP = 12 // < 40%
	foe.X, foe.Y = g.P.X+1, g.P.Y
	z.Actors = append(z.Actors, foe)

	ok, msg := g.ccSubdue(foe)
	if !ok {
		t.Fatalf("ccSubdue failed: %s", msg)
	}
	if len(g.cc().Captives) != 1 {
		t.Fatalf("expected 1 captive, got %d", len(g.cc().Captives))
	}
	capID := g.cc().Captives[0].ID

	guard := g.Spawn("guard", 3, g.R)
	guard.Name = "Gate Warden"
	guard.AI = "guard"
	guard.Role = "guard"
	guard.Faction = "bandits"

	// Verify talkHook surfaces ransom option.
	opts := ccTalk(g, guard)
	foundRansomOpt := false
	for _, o := range opts {
		if strings.Contains(strings.ToLower(o.Text), "ransom") {
			foundRansomOpt = true
			break
		}
	}
	if !foundRansomOpt {
		t.Fatal("expected ccTalk to include ransom option for guard")
	}

	goldBefore := g.P.Gold
	repBefore := g.Rep["bandits"]
	ok, payout, msg := g.ccRansom(capID, guard)
	if !ok || payout <= 0 {
		t.Fatalf("ccRansom failed: %s", msg)
	}
	if g.P.Gold <= goldBefore {
		t.Fatalf("expected gold to rise above %d, got %d", goldBefore, g.P.Gold)
	}
	if g.Rep["bandits"] <= repBefore {
		t.Fatalf("expected standing rise, got %d -> %d", repBefore, g.Rep["bandits"])
	}
	if len(g.cc().Captives) != 0 {
		t.Fatal("expected captive list to be empty after ransom")
	}

	// Subdue a second foe and recruit them under oath.
	foe2 := g.Spawn("bandit", 2, g.R)
	foe2.Name = "Maren the Scout"
	foe2.Proper = true
	foe2.AI = "hostile"
	foe2.MaxHP = 40
	foe2.HP = 10
	foe2.X, foe2.Y = g.P.X+1, g.P.Y
	z.Actors = append(z.Actors, foe2)

	ok, msg = g.ccSubdue(foe2)
	if !ok {
		t.Fatalf("second ccSubdue failed: %s", msg)
	}
	cap2ID := g.cc().Captives[0].ID
	ok, comp, msg := g.ccRecruitOath(cap2ID)
	if !ok || comp == nil {
		t.Fatalf("ccRecruitOath failed: %s", msg)
	}
	if comp.AI != "follower" || comp.Comp == nil || !comp.Has("oathbound") {
		t.Fatalf("expected oathbound follower companion, got AI=%q comp=%v", comp.AI, comp.Comp)
	}
}

func TestFortifiedWildernessCampAndAmbush(t *testing.T) {
	g := newGame(t, nil)
	z := g.Cur
	z.Kind = "wild"
	z.Actors = nil
	g.P.X, g.P.Y = 30, 12
	for dy := -4; dy <= 4; dy++ {
		for dx := -4; dx <= 4; dx++ {
			z.Set(g.P.X+dx, g.P.Y+dy, TGrass)
		}
	}

	ok, msg := g.ccPitchCamp()
	if !ok {
		t.Fatalf("ccPitchCamp failed: %s", msg)
	}
	camp := g.cc().Camps[z.Key]
	if camp == nil || !camp.Active || camp.Barricades == 0 || camp.Tripwires == 0 || !camp.Watchfire {
		t.Fatalf("expected full fortified camp, got %+v", camp)
	}

	twBefore := camp.Tripwires
	tripTriggered, spawned, msg := g.ccCampAmbush()
	if !tripTriggered || spawned == 0 {
		t.Fatalf("expected tripwire ambush trigger and raiders, got trip=%v spawned=%d msg=%s", tripTriggered, spawned, msg)
	}
	if camp.Tripwires != twBefore-1 {
		t.Fatalf("expected 1 tripwire consumed, got %d -> %d", twBefore, camp.Tripwires)
	}
	if g.P.St("fortified") <= 0 {
		t.Fatal("expected player to receive fortified defense bonus during ambush")
	}
}

func TestSmugglerPassagesAndLowTide(t *testing.T) {
	g := newGame(t, nil)
	z := g.Cur
	st := g.sm()

	// Register a tide-locked sea-cave and a town cellar tunnel.
	st.Passages["seacave:test"] = &smPassage{
		ID:         "seacave:test",
		Kind:       "sea_cave",
		Name:       "Salt-Arch Grotto",
		FromKey:    z.Key,
		WX:         g.WX,
		WY:         g.WY,
		X:          10,
		Y:          10,
		TideLocked: true,
		Discovered: true,
	}
	st.Passages["cellar:test"] = &smPassage{
		ID:           "cellar:test",
		Kind:         "cellar_tunnel",
		Name:         "Old Market Cellar Tunnel",
		FromKey:      z.Key,
		DistrictFrom: "Dockside Cellar",
		DistrictTo:   "Outskirts Gate-Drain",
		WX:           g.WX,
		WY:           g.WY,
		X:            15,
		Y:            10,
		TideLocked:   false,
		Discovered:   true,
	}

	// Set hour to 10:00 (high tide: 10 is outside 2..6 and 14..18).
	g.Turn = 10*60 - startMinutes
	if g.Turn < 0 {
		g.Turn += DayLen
	}
	if smLowTide(g) {
		t.Fatalf("expected high tide at hour %d", g.Hour())
	}
	if ok, _ := g.smUseTunnel("seacave:test"); ok {
		t.Fatal("expected tide-locked sea-cave to be blocked at high tide")
	}

	// Set hour to 15:00 (low tide: inside 14..18).
	g.Turn = 15*60 - startMinutes
	if g.Turn < 0 {
		g.Turn += DayLen
	}
	if !smLowTide(g) {
		t.Fatalf("expected low tide at hour %d", g.Hour())
	}
	goldBefore := g.P.Gold
	if ok, msg := g.smUseTunnel("seacave:test"); !ok {
		t.Fatalf("expected sea-cave to open at low tide: %s", msg)
	}
	if g.P.Gold <= goldBefore {
		t.Fatal("expected first sea-cave visit to award smuggler cache coin")
	}

	// Cellar tunnel works regardless of tide and relocates player past gates.
	g.P.X, g.P.Y = 5, 5
	for y := 2; y < MH-2; y++ {
		for x := 2; x < MW-2; x++ {
			z.Set(x, y, TCobble)
		}
	}
	if ok, msg := g.smUseTunnel("cellar:test"); !ok {
		t.Fatalf("cellar tunnel failed: %s", msg)
	}
	if g.P.X == 5 {
		t.Fatal("expected cellar tunnel to relocate player across town")
	}
}

func TestExpeditionLogisticsAndPorterTrain(t *testing.T) {
	g := newGame(t, nil)
	g.P.Gold = 500
	destP := g.W.Idx(g.WX, g.WY)

	ok, msg := g.smStartExpedition(2, 3, true, destP)
	if !ok {
		t.Fatalf("smStartExpedition failed: %s", msg)
	}
	ex := g.sm().Exped
	if ex == nil || !ex.Active || ex.Rations <= 0 || !ex.Cartographer {
		t.Fatalf("unexpected expedition state: %+v", ex)
	}

	rat0 := ex.Rations
	summary, alive := g.smExpeditionTick()
	if !alive || summary == "" {
		t.Fatalf("expedition tick failed: alive=%v summary=%q", alive, summary)
	}
	if ex.Rations >= rat0 {
		t.Fatalf("expected rations to drop on tick: %d -> %d", rat0, ex.Rations)
	}
	if ex.GearWear <= 0 {
		t.Fatalf("expected gear wear to rise on tick, got %d", ex.GearWear)
	}

	wearBefore := ex.GearWear
	ok, msg = g.smRepairWheels()
	if !ok {
		t.Fatalf("smRepairWheels failed: %s", msg)
	}
	if ex.GearWear >= wearBefore {
		t.Fatalf("expected gear wear to decrease: %d -> %d", wearBefore, ex.GearWear)
	}

	ratBefore := ex.Rations
	ok, added, msg := g.smForageRations()
	if !ok || added <= 0 {
		t.Fatalf("smForageRations failed: added=%d msg=%s", added, msg)
	}
	if ex.Rations <= ratBefore {
		t.Fatalf("expected rations to increase: %d -> %d", ratBefore, ex.Rations)
	}
}

func TestMovingPOIsDeterminismAndWindows(t *testing.T) {
	g := newGame(t, nil)

	// Day 0 is lunar phase 0 (new moon) -> Night-Market active.
	g.Turn = 12*60 - startMinutes
	if g.Turn < 0 {
		g.Turn += DayLen
	}
	poisA := g.smMovingPOIs()
	poisB := g.smMovingPOIs()
	if len(poisA) != 3 {
		t.Fatalf("expected 3 moving POIs, got %d", len(poisA))
	}
	for i := range poisA {
		if poisA[i].ID != poisB[i].ID || poisA[i].P != poisB[i].P {
			t.Fatalf("smMovingPOIs not deterministic at index %d", i)
		}
	}
	if poisA[0].Kind != "night_market" || !poisA[0].Active {
		t.Fatalf("expected night_market active on new-moon day, got %+v", poisA[0])
	}
	if poisA[1].Kind != "floating_island" || !poisA[1].Active {
		t.Fatalf("expected floating_island active, got %+v", poisA[1])
	}

	// Advance 3 days: floating island should drift to a new 3-day bucket; night-market dormant.
	g.Turn += 3 * DayLen
	poisDay3 := g.smMovingPOIs()
	if poisDay3[0].Active {
		t.Fatalf("expected night_market dormant on lunar day %d", smMoonPhase(g.Day()))
	}
	if poisDay3[1].ID == poisA[1].ID {
		t.Fatalf("expected floating_island to advance to next 3-day window: %s == %s", poisDay3[1].ID, poisA[1].ID)
	}

	// Night hour (22:00): ghost-ship becomes boardable.
	g.Turn = g.Day()*DayLen + 22*60 - startMinutes
	poisNight := g.smMovingPOIs()
	if poisNight[2].Kind != "ghost_ship" || !poisNight[2].Active {
		t.Fatalf("expected ghost_ship active at night (hour %d), got %+v", g.Hour(), poisNight[2])
	}
	if ok, msg := g.smVisitPOI("ghost_ship"); !ok {
		t.Fatalf("expected ghost_ship boarding to succeed at night: %s", msg)
	}
}

func TestCombatExploreModesDrawAndKeys(t *testing.T) {
	g := newGame(t, nil)

	// Mode "cc" across all 3 tabs.
	g.ccOpen()
	for _, k := range []string{"1", "Down", "2", "3", "Tab", "Escape"} {
		g.Key(k)
		g.Render()
	}

	// Mode "sm" across all 3 tabs.
	g.smOpen()
	for _, k := range []string{"1", "Down", "2", "3", "Tab", "Escape"} {
		g.Key(k)
		g.Render()
	}
}
