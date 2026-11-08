package engine

import (
	"strings"
	"testing"
)

func TestWyPlanetTraits(t *testing.T) {
	g := newGame(t, nil)

	// Deterministic traits across planet IDs
	seenTraits := map[string]bool{}
	for pid := 0; pid < 4; pid++ {
		ts := wyPlanetTrait(g.W, pid)
		if len(ts) < 1 || len(ts) > 2 {
			t.Fatalf("expected 1-2 traits for planet %d, got %v", pid, ts)
		}
		ts2 := wyPlanetTrait(g.W, pid)
		if strings.Join(ts, ",") != strings.Join(ts2, ",") {
			t.Fatalf("expected deterministic traits for planet %d, got %v vs %v", pid, ts, ts2)
		}
		for _, tr := range ts {
			seenTraits[tr] = true
		}
	}
	for _, want := range wyAllPlanetTraits {
		if !seenTraits[want] {
			t.Errorf("expected trait %q to appear across planet IDs 0..3, got %v", want, seenTraits)
		}
	}

	// Low gravity: +2 throw bonus and -50% fall damage
	g.wySetPlanetTraits(g.PlanetIndex(), "low_gravity")
	if got := wyThrowBonus(g); got != 2 {
		t.Fatalf("expected wyThrowBonus == 2 on low_gravity planet, got %d", got)
	}
	if got := wyFallDamage(g, 20); got != 10 {
		t.Fatalf("expected wyFallDamage(20) == 10 on low_gravity planet, got %d", got)
	}

	// Toxic air: deals 1 poison damage every 10 turns outdoors unless protected
	g.wySetPlanetTraits(g.PlanetIndex(), "toxic_air")
	g.Cur.Z = 0
	g.Cur.Underwater = false
	g.Turn = 20
	hp0 := g.P.HP
	wyPlanetTurn(g)
	if g.P.HP != hp0-1 {
		t.Fatalf("expected toxic_air to deal 1 damage outdoors on turn 20 (hp %d -> %d)", hp0, g.P.HP)
	}
	// Equip a gasmask item -> no damage
	mask := &Item{ID: g.newID(), Kind: "armor", Slot: "face", Name: "rebreather gasmask", Effect: "gasmask"}
	g.P.Inv = append(g.P.Inv, mask)
	hp1 := g.P.HP
	wyPlanetTurn(g)
	if g.P.HP != hp1 {
		t.Fatalf("expected gasmask to protect from toxic_air (hp %d -> %d)", hp1, g.P.HP)
	}

	// Tidally locked: west half (WX < W/2) is permanent day, east half (WX >= W/2) is permanent night
	g.wySetPlanetTraits(g.PlanetIndex(), "tidally_locked")
	g.WX = 1
	locked, day := wyTidallyLockedLight(g)
	if !locked || !day {
		t.Fatalf("expected west half of tidally_locked planet to be daylight, got locked=%v day=%v", locked, day)
	}
	wyPlanetTurn(g)
	if g.Cur.Dark {
		t.Fatalf("expected west half surface zone not to be dark")
	}
	g.WX = g.W.W - 1
	locked, day = wyTidallyLockedLight(g)
	if !locked || day {
		t.Fatalf("expected east half of tidally_locked planet to be night, got locked=%v day=%v", locked, day)
	}
	wyPlanetTurn(g)
	if !g.Cur.Dark {
		t.Fatalf("expected east half surface zone to be dark")
	}

	// Dead ruin: spawns construct remnant and precursor relic in surface wilds zone
	g.wySetPlanetTraits(g.PlanetIndex(), "dead_ruin")
	wild := newZone("test.wild", g.WX, g.WY, 0, 0, 0)
	wild.Kind = "wild"
	for i := range wild.T {
		wild.T[i] = TGrass
	}
	wyZoneHook(g, wild, NewRng("test:dead_ruin"))
	foundCon, foundRelic := false, false
	for _, a := range wild.Actors {
		if a.Has("dead_ruin_remnant") {
			foundCon = true
		}
	}
	for _, it := range wild.Items {
		if it.Extra["dead_ruin_relic"] > 0 {
			foundRelic = true
		}
	}
	if !foundCon || !foundRelic {
		t.Fatalf("expected dead_ruin zone hook to spawn construct (%v) and relic (%v)", foundCon, foundRelic)
	}
}

func TestWyRiftStormsAndStarMap(t *testing.T) {
	g := newGame(t, nil)

	// Search across 15-day seasonal windows for a rift storm
	stormFound := false
	for window := 0; window < 12; window++ {
		g.Turn = window * 15 * DayLen
		if wyIsRiftStorm(g, 0) {
			stormFound = true
			break
		}
	}
	if !stormFound {
		t.Fatalf("expected wyIsRiftStorm to trigger in at least one 15-day window")
	}

	// Running wyRiftStormTurn during an active storm spawns a rift-displaced creature
	wyRiftStormTurn(g)
	displaced := false
	for _, a := range g.Cur.Actors {
		if a.Has("rift_displaced") {
			displaced = true
			break
		}
	}
	if !displaced {
		t.Fatalf("expected wyRiftStormTurn to spawn a rift_displaced creature during storm")
	}

	// Sealed rift check
	g.wy().ClosedRifts[0] = g.Day() / 15
	if !wyRiftClosed(g, 0) {
		t.Fatalf("expected wyRiftClosed(0) to be true when sealed for current season")
	}

	// Star-map screen mode
	OpenStarMap(g)
	if g.Mode != "starmap" {
		t.Fatalf("expected Mode == starmap, got %q", g.Mode)
	}
	lines := g.wyStarMapLines()
	if len(lines) < 3 || !strings.Contains(strings.Join(lines, "\n"), "[HERE]") {
		t.Fatalf("expected star-map lines to include [HERE] marker, got %v", lines)
	}
	g.Render()
	wyStarMapKey(g, "Down")
	wyStarMapKey(g, "Up")
	wyStarMapKey(g, "Escape")
	if g.Mode != "main" {
		t.Fatalf("expected Escape from starmap to return to main mode, got %q", g.Mode)
	}
}

func TestWyOffworldTradeGoods(t *testing.T) {
	g := newGame(t, nil)
	it := g.MakeWeapon("longblade", 2, g.R)

	if mul := g.wyOffworldPriceMul(it); mul != 1.0 {
		t.Fatalf("expected untagged item price multiplier 1.0, got %v", mul)
	}

	// Tag with current planet -> 1.0x
	wyTagOffworldItem(it, g.PlanetIndex())
	if mul := g.wyOffworldPriceMul(it); mul != 1.0 {
		t.Fatalf("expected same-planet item price multiplier 1.0, got %v", mul)
	}

	// Tag with another planet -> 2.0x
	otherPlanet := g.PlanetIndex() + 1
	wyTagOffworldItem(it, otherPlanet)
	if it.Extra["planet_origin:1"] == 0 && it.Extra["planet_origin"] == 0 {
		t.Fatalf("expected planet_origin tag in it.Extra, got %v", it.Extra)
	}
	if mul := g.wyOffworldPriceMul(it); mul != 2.0 {
		t.Fatalf("expected off-world item price multiplier 2.0, got %v", mul)
	}
}

func TestWyLibraryBooksRumoursAndDreams(t *testing.T) {
	g := newGame(t, nil)
	xp0 := g.XP + g.P.Level*1000

	// 1. Library Book
	passage := g.wyReadLibraryBook(5, 5)
	if passage == "" || !strings.Contains(passage, "ruin at (") {
		t.Fatalf("unexpected library book passage: %q", passage)
	}
	if g.XP+g.P.Level*1000 <= xp0 {
		t.Fatalf("expected reading library book to grant +15 XP")
	}
	if len(g.wyRevealedSites()) == 0 {
		t.Fatalf("expected wyReadLibraryBook to reveal a site in wyRevealedSites")
	}

	// 2. Rumours (accurate and twisted)
	npc := &Actor{ID: 777, Name: "Old Harn", Role: "villager", Proper: true, HP: 10, MaxHP: 10}
	sawAccurate, sawTwisted := false, false
	for i := 0; i < 6; i++ {
		txt, acc := g.wyRumour(npc)
		if txt == "" {
			t.Fatalf("expected non-empty rumour text")
		}
		if acc {
			sawAccurate = true
		} else {
			sawTwisted = true
		}
	}
	if !sawAccurate || !sawTwisted {
		t.Fatalf("expected both accurate (%v) and twisted (%v) rumours across 6 calls", sawAccurate, sawTwisted)
	}
	twistedMap := g.wyTwistedRumours()
	if len(twistedMap) == 0 {
		t.Fatalf("expected twisted rumour coordinates to be recorded")
	}
	var twCoord [2]int
	for k := range twistedMap {
		twCoord = k
		break
	}
	tz := newZone("twist.zone", twCoord[0], twCoord[1], 1, 1, 0)
	tz.Kind = "wild"
	for i := range tz.T {
		tz.T[i] = TGrass
	}
	wyOnEnterZone(g, tz)
	if _, stillThere := g.wyTwistedRumours()[twCoord]; stillThere {
		t.Fatalf("expected visiting twisted rumour parasang to consume the flag")
	}

	// 3. Historical Dreams
	dream := wyOnRest(g)
	if dream == "" || g.wyDreamHint() != dream {
		t.Fatalf("expected wyOnRest to record and return dream hint, got %q", dream)
	}
}

func TestWyProphecyCompanionArcsAndEpilogues(t *testing.T) {
	g := newGame(t, nil)

	// Prophecy initialization & dynamic resolution
	pr := g.wyEnsureProphecy()
	if pr == nil || pr.Verses[0] == "" || pr.Verses[1] == "" || pr.Verses[2] == "" {
		t.Fatalf("expected 3-verse prophecy to be initialized, got %+v", pr)
	}
	foe := placeFoe(g, "rat", 2)
	foe.Boss = true
	wyOnKill(g, foe, g.P)
	g.unSt().Trips = 2
	g.Rep["state:1"] = 180
	res := g.wyProphecyResolution()
	if !strings.Contains(res, "fulfilled in blood") || !strings.Contains(res, "across the stars") || !strings.Contains(res, "Champion of the Realms") {
		t.Fatalf("unexpected prophecy resolution: %q", res)
	}

	// Companion Arc: trust -> crisis -> bond -> lethal sacrifice
	comp := recruitOne(t, g)
	comp.Comp.Loyalty = 80
	if stg := g.wyAdvanceCompanionArc(comp, "trust"); stg != "trust" {
		t.Fatalf("expected stage trust, got %q", stg)
	}
	if stg := g.wyAdvanceCompanionArc(comp, "crisis"); stg != "crisis" {
		t.Fatalf("expected stage crisis, got %q", stg)
	}
	if out := g.wyAdvanceCompanionArc(comp, "bond"); out != "bond" {
		t.Fatalf("expected outcome bond, got %q", out)
	}

	// Lethal blow interception
	g.P.HP = 5
	g.P.MaxHP = 30
	if !g.wyCheckCompanionSacrifice(20) {
		t.Fatalf("expected bonded companion to intercept lethal blow")
	}
	if comp.HP != 0 || g.P.HP != 10 {
		t.Fatalf("expected companion HP=0 and player HP=10 (MaxHP/3), got comp.HP=%d P.HP=%d", comp.HP, g.P.HP)
	}

	// Companion betrayal when loyalty < 15
	comp2 := &Actor{
		ID:     g.newID(),
		Name:   "GrimVane",
		Proper: true,
		AI:     "follower",
		HP:     20,
		MaxHP:  20,
		Comp:   &Companion{Loyalty: 10},
	}
	g.Cur.Actors = append(g.Cur.Actors, comp2)
	if out := g.wyAdvanceCompanionArc(comp2, "crisis"); out != "betrayal" || comp2.AI != "hostile" {
		t.Fatalf("expected low-loyalty companion to betray player, got outcome=%q ai=%q", out, comp2.AI)
	}

	// Epilogues
	eps := g.wyBuildEpilogues()
	joined := strings.Join(eps, "\n")
	if !strings.Contains(joined, comp.Name) || !strings.Contains(joined, "GrimVane") || !strings.Contains(joined, "Prophecy:") {
		t.Fatalf("expected epilogues to include sacrificed companion, betrayer, and prophecy:\n%s", joined)
	}
}
