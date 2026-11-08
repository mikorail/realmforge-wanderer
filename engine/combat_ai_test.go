package engine

import (
	"strings"
	"testing"
)

func TestCaSmartActRetreatAndRegroup(t *testing.T) {
	g := newGame(t, nil)
	openField(g)
	g.P.X, g.P.Y = 20, 12

	foe := g.Spawn("bandit", 3, NewRng("ca_retreat"))
	foe.MaxHP = 40
	foe.HP = 8 // <= MaxHP/4
	foe.X, foe.Y = 22, 12
	g.Cur.Actors = []*Actor{foe}

	d0 := dist(foe, g.P)
	if !g.caSmartAct(foe, g.P) {
		t.Fatalf("expected caSmartAct to trigger wounded retreat")
	}
	if !foe.Has("ca_fleeing") {
		t.Fatalf("expected ca_fleeing flag when retreating")
	}
	if dist(foe, g.P) <= d0 {
		t.Fatalf("expected foe to step away from target: dist %d -> %d", d0, dist(foe, g.P))
	}

	// Move foe to dist >= 5 so it bandages and rallies
	foe.X, foe.Y = 26, 12
	g.recompute()
	hpBefore := foe.HP
	if !g.caSmartAct(foe, g.P) {
		t.Fatalf("expected caSmartAct to regroup at dist >= 5")
	}
	if foe.HP <= hpBefore || foe.Has("ca_fleeing") || !foe.Has("ca_regrouped") {
		t.Fatalf("expected foe to heal, clear ca_fleeing, and set ca_regrouped: hp=%d->%d fleeing=%v regrouped=%v",
			hpBefore, foe.HP, foe.Has("ca_fleeing"), foe.Has("ca_regrouped"))
	}
}

func TestCaSmartActArcherKitingAndPackFlank(t *testing.T) {
	g := newGame(t, nil)
	openField(g)
	g.P.X, g.P.Y = 20, 12
	g.P.HP, g.P.MaxHP = 200, 200

	// 1. Archer kiting at dist <= 2
	archer := g.Spawn("archer", 3, NewRng("ca_archer"))
	archer.HP, archer.MaxHP = 30, 30
	archer.X, archer.Y = 21, 12
	g.Cur.Actors = []*Actor{archer}
	if archer.Missile() == nil {
		t.Fatalf("expected archer template to have a missile weapon")
	}
	if !g.caSmartAct(archer, g.P) {
		t.Fatalf("expected archer at dist 1 to kite away via caSmartAct")
	}
	if dist(archer, g.P) <= 1 {
		t.Fatalf("expected archer to increase distance from target, got dist=%d", dist(archer, g.P))
	}

	// 2. Pack flanking opposite an adjacent ally
	wolf1 := g.Spawn("wolf", 3, NewRng("ca_wolf1"))
	wolf1.Set("pack")
	wolf1.Faction = "beasts"
	wolf1.X, wolf1.Y = 20, 11 // North of player (20, 12)

	wolf2 := g.Spawn("wolf", 3, NewRng("ca_wolf2"))
	wolf2.Set("pack")
	wolf2.Faction = "beasts"
	wolf2.HP, wolf2.MaxHP = 30, 30
	wolf2.X, wolf2.Y = 21, 12 // East of player; can step to (20, 13) South of player to flank opposite wolf1!

	g.Cur.Actors = []*Actor{wolf1, wolf2}
	if !g.caSmartAct(wolf2, g.P) {
		t.Fatalf("expected pack member wolf2 to flank and attack")
	}
	if wolf2.X != 20 || wolf2.Y != 13 {
		t.Fatalf("expected wolf2 to step to opposite flanking tile (20,13), got (%d,%d)", wolf2.X, wolf2.Y)
	}
}

func TestCaWeatherTurnFogAndDownwindScent(t *testing.T) {
	g := newGame(t, nil)
	openField(g)
	g.P.X, g.P.Y = 20, 12

	// 1. Rain douses fire on player and exposed outdoor actors, but not roofed actors
	exposed := g.Spawn("bandit", 2, NewRng("ca_exp"))
	exposed.X, exposed.Y = 22, 12
	exposed.AddSt("burning", 5)

	indoors := g.Spawn("bandit", 2, NewRng("ca_ind"))
	indoors.X, indoors.Y = 25, 12
	g.Cur.Set(25, 12, TFloor) // roofed tile
	indoors.AddSt("burning", 5)

	g.Cur.Actors = []*Actor{exposed, indoors}
	g.P.AddSt("burning", 5)
	g.wxForce = &Weather{Kind: WeatherByID("rain"), Temp: 14}

	caWeatherTurn(g)

	if g.P.St("burning") != 0 {
		t.Fatalf("expected rain to douse burning on player")
	}
	if exposed.St("burning") != 0 {
		t.Fatalf("expected rain to douse burning on exposed actor")
	}
	if indoors.St("burning") == 0 {
		t.Fatalf("expected roofed actor to remain burning during rain")
	}
	foundMsg := false
	for _, m := range g.Msgs {
		if strings.Contains(m.T, "The rain douses the flames on you.") {
			foundMsg = true
			break
		}
	}
	if !foundMsg {
		t.Fatalf("expected rain douse message in log")
	}

	// 2. Fog conceals distant shots and grants +2 ranged DV
	g.wxForce = &Weather{Kind: WeatherByID("fog"), Temp: 10}
	g.P.AddSt("hidden", 3)
	if !g.caFogConcealShot(g.P, 5) || g.P.St("hidden") <= 0 {
		t.Fatalf("expected fog to conceal ranged shot at dist 5 and preserve hidden status")
	}
	if g.caFogConcealShot(g.P, 2) {
		t.Fatalf("expected fog not to conceal point-blank shot at dist 2")
	}
	if dv := g.caWeatherRangedDV(exposed, 5); dv != 2 {
		t.Fatalf("expected +2 ranged concealment DV in fog at dist 5, got %d", dv)
	}
	if dv := g.caWeatherRangedDV(exposed, 2); dv != 0 {
		t.Fatalf("expected 0 ranged concealment DV at dist 2, got %d", dv)
	}

	// 3. Wind carries scent to downwind predators within 14 tiles
	g.wxForce = &Weather{Kind: WeatherByID("clear"), Temp: 16}
	delete(g.P.Status, "almask")
	wx, wy := g.caWindVec()
	if wx == 0 && wy == 0 {
		t.Fatalf("expected non-zero wind vector from caWindVec")
	}

	pred := g.Spawn("wolf", 3, NewRng("ca_pred"))
	pred.Faction = "beasts"
	pred.Set("pack")
	pred.AddSt("asleep", 10)
	pred.X, pred.Y = g.P.X+wx*5, g.P.Y+wy*5 // downwind

	if !g.caScentNotice(pred) {
		t.Fatalf("expected downwind predator at dist 5 to detect player scent")
	}
	if pred.St("asleep") != 0 || !pred.Has("ca_scented") {
		t.Fatalf("expected scent detection to wake predator and set ca_scented")
	}

	// Upwind predator does not smell player
	upwind := g.Spawn("wolf", 3, NewRng("ca_upwind"))
	upwind.Faction = "beasts"
	upwind.X, upwind.Y = g.P.X-wx*5, g.P.Y-wy*5
	if g.caScentNotice(upwind) {
		t.Fatalf("expected upwind predator not to detect player scent")
	}

	// Alchemical scent-mask ("almask") blocks downwind scent detection
	g.P.AddSt("almask", 10)
	if g.caScentNotice(pred) {
		t.Fatalf("expected almask status to mask player scent from downwind predator")
	}
}
