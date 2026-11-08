package engine

import (
	"strings"
	"testing"
)

func TestTcCalledShotsAllPartsAndMode(t *testing.T) {
	g := newGame(t, nil)
	clearAround(g)
	g.P.Attr[AGI] = 30
	g.P.Attr[STR] = 24
	w := g.MakeWeapon("longblade", 3, g.R)
	g.equip(w)

	// 1. Head -> stunned
	foe := placeFoe(g, "bandit", 2)
	foe.HP, foe.MaxHP, foe.DVNat = 500, 500, -20
	hit := false
	for i := 0; i < 20 && !hit; i++ {
		hit = g.tcCalledShot(foe, "head")
	}
	if !hit || foe.St("stunned") <= 0 {
		t.Fatalf("expected head called shot to hit and stun, hit=%v stunned=%d", hit, foe.St("stunned"))
	}

	// 2. Arm -> disarm weapon + weakened
	sword := g.MakeWeapon("shortblade", 1, g.R)
	for _, bp := range foe.Body {
		if bp.Slot == "hand" {
			bp.Item = sword
			break
		}
	}
	hit = false
	for i := 0; i < 20 && !hit; i++ {
		hit = g.tcCalledShot(foe, "arm")
	}
	if !hit || foe.St("weakened") <= 0 || foe.Equipped(sword) != nil {
		t.Fatalf("expected arm called shot to disarm and weaken, hit=%v weakened=%d equipped=%v", hit, foe.St("weakened"), foe.Equipped(sword))
	}

	// 3. Leg -> crippled / slowed
	hit = false
	for i := 0; i < 20 && !hit; i++ {
		hit = g.tcCalledShot(foe, "leg")
	}
	if !hit || foe.St("crippled") <= 0 || foe.St("slowed") <= 0 {
		t.Fatalf("expected leg called shot to cripple and slow, hit=%v crippled=%d slowed=%d", hit, foe.St("crippled"), foe.St("slowed"))
	}

	// 4. Wing -> grounded
	flier := placeFoe(g, "harpy", 3)
	flier.HP, flier.MaxHP, flier.DVNat = 500, 500, -20
	flier.Set("fly")
	hit = false
	for i := 0; i < 20 && !hit; i++ {
		hit = g.tcCalledShot(flier, "wing")
	}
	if !hit || flier.St("grounded") <= 0 || flier.Has("fly") {
		t.Fatalf("expected wing called shot to ground flier, hit=%v grounded=%d fly=%v", hit, flier.St("grounded"), flier.Has("fly"))
	}

	// 5. Eye -> blind
	hit = false
	for i := 0; i < 20 && !hit; i++ {
		hit = g.tcCalledShot(foe, "eye")
	}
	if !hit || foe.St("blind") <= 0 {
		t.Fatalf("expected eye called shot to blind, hit=%v blind=%d", hit, foe.St("blind"))
	}

	// 6. V key / "calledshot" mode
	g.Key("V")
	if g.Mode != "calledshot" {
		t.Fatalf("expected V key to open calledshot mode, got %q", g.Mode)
	}
	g.Render()
	g.Key("ArrowDown")
	if g.Sel != 1 {
		t.Fatalf("expected ArrowDown in calledshot mode to move selection to 1, got %d", g.Sel)
	}
	g.Key("Enter")
	if g.Mode != "main" {
		t.Fatalf("expected Enter in calledshot mode to return to main, got %q", g.Mode)
	}
}

func TestTcPositioningModifiers(t *testing.T) {
	g := newGame(t, nil)
	g.Cur.Actors = nil
	clearAround(g)
	g.P.X, g.P.Y = 20, 12
	foe := placeFoe(g, "bandit", 3)
	foe.X, foe.Y = 21, 12

	// No allies adjacent -> 0 flank bonus
	if b := g.tcFlankBonus(g.P, foe); b != 0 {
		t.Fatalf("expected 0 flank bonus with no allies, got %d", b)
	}

	// Ally on opposite side (22, 12) -> +2 flank bonus
	ally1 := g.Spawn("bandit", 2, NewRng("test:ally1"))
	ally1.AI = "follower"
	ally1.Faction = "player"
	ally1.X, ally1.Y = 22, 12
	g.Cur.Actors = append(g.Cur.Actors, ally1)
	if b := g.tcFlankBonus(g.P, foe); b != 2 {
		t.Fatalf("expected +2 flank bonus with opposite ally, got %d", b)
	}

	// Additional adjacent attacker at (21, 11) -> +3 total (+2 opposite + 1 extra)
	ally2 := g.Spawn("bandit", 2, NewRng("test:ally2"))
	ally2.AI = "follower"
	ally2.Faction = "player"
	ally2.X, ally2.Y = 21, 11
	g.Cur.Actors = append(g.Cur.Actors, ally2)
	if b := g.tcFlankBonus(g.P, foe); b != 3 {
		t.Fatalf("expected +3 flank bonus with opposite + side ally, got %d", b)
	}

	// High ground bonus: standing on stairs (TUp) or cliff (TCliff) vs floor
	g.Cur.Set(g.P.X, g.P.Y, TFloor)
	g.Cur.Set(foe.X, foe.Y, TFloor)
	if b := g.tcHighGroundBonus(g.P, foe); b != 0 {
		t.Fatalf("expected 0 high ground bonus on flat floor, got %d", b)
	}
	g.Cur.Set(g.P.X, g.P.Y, TUp)
	if b := g.tcHighGroundBonus(g.P, foe); b != 2 {
		t.Fatalf("expected +2 high ground bonus on TUp stairs, got %d", b)
	}
	g.Cur.Set(g.P.X, g.P.Y, TFloor)

	// Cover bonus: +2 DV adjacent to wall/pillar/tree against ranged (dist > 1)
	shooter := placeFoe(g, "archer", 3)
	shooter.X, shooter.Y = 26, 12
	for dy := -1; dy <= 1; dy++ {
		for dx := -1; dx <= 1; dx++ {
			g.Cur.Set(g.P.X+dx, g.P.Y+dy, TFloor)
		}
	}
	if b := g.tcCoverBonus(g.P, shooter); b != 0 {
		t.Fatalf("expected 0 cover bonus in the open, got %d", b)
	}
	g.Cur.Set(g.P.X, g.P.Y-1, TPillar)
	if b := g.tcCoverBonus(g.P, shooter); b != 2 {
		t.Fatalf("expected +2 cover bonus adjacent to TPillar vs ranged, got %d", b)
	}
	foe.X, foe.Y = g.P.X+1, g.P.Y
	if b := g.tcCoverBonus(g.P, foe); b != 0 {
		t.Fatalf("expected 0 cover bonus against adjacent melee attacker, got %d", b)
	}

	// Weather & time-of-day ranged modifier
	g.Cur.Z = 0
	g.Cur.Dark = false
	g.Cur.Underwater = false
	g.Cur.Set(g.P.X, g.P.Y, TGrass)
	g.Turn = 120 // 10:00 morning
	g.wxForce = &Weather{Kind: WeatherByID("rain"), Temp: 15}
	if mod := g.tcWeatherRangedMod(); mod >= 0 {
		t.Fatalf("expected negative ranged modifier in rain, got %d", mod)
	}
	g.wxForce = &Weather{Kind: WeatherByID("fog"), Temp: 12}
	if mod := g.tcWeatherRangedMod(); mod >= 0 {
		t.Fatalf("expected negative ranged modifier in fog, got %d", mod)
	}
	// Glare on snow at noon
	g.wxForce = &Weather{Kind: WeatherByID("clear"), Temp: -2}
	g.Turn = 240 // 12:00 noon
	g.Cur.Set(g.P.X, g.P.Y, TSnow)
	if mod := g.tcWeatherRangedMod(); mod >= 0 {
		t.Fatalf("expected negative ranged modifier from midday snow glare, got %d", mod)
	}
	g.wxForce = nil
}

func TestTcSpecialAmmoAndTacticalItems(t *testing.T) {
	g := newGame(t, nil)
	clearAround(g)

	// Special ammo kinds
	for _, kind := range []string{"bodkin", "broadhead", "fire", "frost", "poison", "silver", "emp"} {
		am := g.tcMakeSpecialAmmo(kind, 5)
		if am == nil || am.Kind != "ammo" || am.Count != 5 || am.Effect != kind {
			t.Fatalf("tcMakeSpecialAmmo(%q) = %+v", kind, am)
		}
	}

	// Apply ammo hits
	target := placeFoe(g, "bandit", 4)
	target.HP, target.MaxHP, target.AVNat = 500, 500, 4
	g.tcApplyAmmoHit(g.tcMakeSpecialAmmo("bodkin", 1), target)
	if target.St("cracked") <= 0 {
		t.Fatalf("expected bodkin arrow to crack armor")
	}
	g.tcApplyAmmoHit(g.tcMakeSpecialAmmo("broadhead", 1), target)
	if target.St("bleeding") <= 0 {
		t.Fatalf("expected broadhead arrow to cause bleeding")
	}
	g.tcApplyAmmoHit(g.tcMakeSpecialAmmo("fire", 1), target)
	if target.St("burning") <= 0 {
		t.Fatalf("expected fire arrow to ignite target")
	}
	g.tcApplyAmmoHit(g.tcMakeSpecialAmmo("frost", 1), target)
	if target.St("frozen") <= 0 {
		t.Fatalf("expected frost arrow to freeze target")
	}
	g.tcApplyAmmoHit(g.tcMakeSpecialAmmo("poison", 1), target)
	if target.St("poisoned") <= 0 {
		t.Fatalf("expected poison arrow to poison target")
	}

	undead := placeFoe(g, "skeleton", 3)
	undead.HP, undead.MaxHP = 200, 200
	hp0 := undead.HP
	g.tcApplyAmmoHit(g.tcMakeSpecialAmmo("silver", 1), undead)
	if undead.HP >= hp0-2 || undead.St("shaken") <= 0 {
		t.Fatalf("expected silver ammo to deal bonus damage and shake undead, hp %d -> %d", hp0, undead.HP)
	}

	bot := placeFoe(g, "golem", 5)
	bot.HP, bot.MaxHP = 200, 200
	g.tcApplyAmmoHit(g.tcMakeSpecialAmmo("emp", 1), bot)
	if bot.St("stunned") <= 0 {
		t.Fatalf("expected EMP ammo to stun construct")
	}

	// Tactical items: net, bolas, caltrops, smokebomb, flashbang
	foe := placeFoe(g, "bandit", 3)
	foe.HP, foe.MaxHP = 500, 500
	foe.X, foe.Y = g.P.X+2, g.P.Y

	net := g.tcMakeTacticalItem("net", 2)
	g.give(net)
	if !g.tcUseTacticalItem(net, foe.X, foe.Y) || foe.St("entangled") <= 0 || net.Count != 1 {
		t.Fatalf("expected net to entangle foe and decrement stack, entangled=%d count=%d", foe.St("entangled"), net.Count)
	}

	delete(foe.Status, "entangled")
	bolas := g.tcMakeTacticalItem("bolas", 1)
	g.give(bolas)
	if !g.tcUseTacticalItem(bolas, foe.X, foe.Y) || foe.St("prone") <= 0 || foe.St("hobbled") <= 0 {
		t.Fatalf("expected bolas to trip and hobble foe, prone=%d hobbled=%d", foe.St("prone"), foe.St("hobbled"))
	}

	delete(foe.Status, "bleeding")
	delete(foe.Status, "hobbled")
	cal := g.tcMakeTacticalItem("caltrops", 1)
	g.give(cal)
	if !g.tcUseTacticalItem(cal, foe.X, foe.Y) || foe.St("bleeding") <= 0 || foe.St("hobbled") <= 0 {
		t.Fatalf("expected caltrops to bleed and hobble foe at target tile")
	}

	smoke := g.tcMakeTacticalItem("smokebomb", 1)
	g.give(smoke)
	if !g.tcUseTacticalItem(smoke, foe.X, foe.Y) || !g.tcInSmoke(foe.X, foe.Y) || foe.St("blind") <= 0 {
		t.Fatalf("expected smokebomb to create smoke and blind foe inside")
	}

	delete(foe.Status, "stunned")
	flash := g.tcMakeTacticalItem("flashbang", 1)
	g.give(flash)
	if !g.tcUseTacticalItem(flash, foe.X, foe.Y) || foe.St("stunned") <= 0 || foe.St("confused") <= 0 {
		t.Fatalf("expected flashbang to stun and confuse foe")
	}
}

func TestTcMoraleRoutAndRally(t *testing.T) {
	g := newGame(t, nil)
	clearAround(g)

	// Spawn a boss leader and two regular outlaws, plus one undead to verify immunity
	leader := placeFoe(g, "chief", 6)
	leader.Boss = true
	leader.Faction = "outlaws"
	leader.X, leader.Y = g.P.X+2, g.P.Y

	grunt1 := placeFoe(g, "bandit", 2)
	grunt1.Faction = "outlaws"
	grunt1.X, grunt1.Y = g.P.X+3, g.P.Y

	grunt2 := placeFoe(g, "bandit", 2)
	grunt2.Faction = "outlaws"
	grunt2.X, grunt2.Y = g.P.X+3, g.P.Y+1

	skel := placeFoe(g, "skeleton", 2)
	skel.Faction = "outlaws"
	skel.Set("undead")
	skel.X, skel.Y = g.P.X+4, g.P.Y

	g.recompute()

	// Kill the leader -> grunts rout ("feared"), undead does not
	g.hurt(leader, leader.HP+100, g.P, "sword")
	if grunt1.St("feared") <= 0 || grunt2.St("feared") <= 0 {
		t.Fatalf("expected outlaws to rout when leader died: grunt1=%d grunt2=%d", grunt1.St("feared"), grunt2.St("feared"))
	}
	if skel.St("feared") > 0 {
		t.Fatalf("expected undead to be immune to morale rout")
	}

	// Surviving captain rallies the line on turn hook
	capt := placeFoe(g, "chief", 6)
	capt.Boss = true
	capt.Faction = "outlaws"
	capt.X, capt.Y = grunt1.X+1, grunt1.Y
	tcMoraleTurn(g)
	if grunt1.St("feared") != 0 || grunt2.St("feared") != 0 {
		t.Fatalf("expected surviving captain rally to clear fear on nearby allies: grunt1=%d grunt2=%d", grunt1.St("feared"), grunt2.St("feared"))
	}
	foundRallyMsg := false
	for _, m := range g.Msgs {
		if strings.Contains(m.T, "rallies the line!") {
			foundRallyMsg = true
			break
		}
	}
	if !foundRallyMsg {
		t.Fatalf("expected rally cry message containing 'rallies the line!'")
	}
}
