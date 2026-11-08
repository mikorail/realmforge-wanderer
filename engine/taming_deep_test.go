package engine

import (
	"testing"
)

func TestTdBreedingOffspringAndCooldown(t *testing.T) {
	g := newGame(t, nil)
	clearAround(g)

	a1 := g.Spawn("wolf", 3, g.R)
	a1.Name = "Silverfang"
	a1.AI = "follower"
	a1.MaxHP, a1.HP = 40, 40
	a1.AVNat, a1.DVNat = 3, 4
	a1.Set("tm")
	a1.Set("tr_hex:#d0d8e8:#406090")
	g.Cur.Actors = append(g.Cur.Actors, a1)

	a2 := g.Spawn("wolf", 4, g.R)
	a2.Name = "Ashpelt"
	a2.AI = "follower"
	a2.MaxHP, a2.HP = 50, 50
	a2.AVNat, a2.DVNat = 4, 5
	a2.Set("tm")
	g.Cur.Actors = append(g.Cur.Actors, a2)

	child, msg := g.tdBreed(a1, a2)
	if child == nil {
		t.Fatalf("expected breeding to succeed, got nil (%s)", msg)
	}
	if child.Level != 5 || child.MaxHP < 60 || child.AVNat != 5 || child.DVNat != 6 {
		t.Fatalf("expected child lv=5, mhp>=60, av=5, dv=6; got lv=%d mhp=%d av=%d dv=%d",
			child.Level, child.MaxHP, child.AVNat, child.DVNat)
	}
	if !child.Has("td_bred") || !child.Has("tr_hex:#d0d8e8:#406090") || child.AI != "follower" {
		t.Fatalf("expected child to have td_bred, inherited morph flag, and follower AI; flags=%v ai=%q",
			child.Flags, child.AI)
	}

	// Immediate second breed should fail on cooldown
	if c2, _ := g.tdBreed(a1, a2); c2 != nil {
		t.Fatalf("expected breeding cooldown to block immediate re-breeding")
	}
}

func TestTdTricksFetchGuardAndHunt(t *testing.T) {
	g := newGame(t, nil)
	clearAround(g)
	g.Cur.Items = nil

	pet := g.Spawn("wolf", 3, g.R)
	pet.Name = "Hound"
	pet.AI = "follower"
	pet.X, pet.Y = g.P.X+1, g.P.Y
	pet.MaxHP, pet.HP = 60, 60
	pet.Set("tm")
	g.Cur.Actors = append(g.Cur.Actors, pet)

	if !g.tdTeachTrick(pet, "fetch") || !g.tdTeachTrick(pet, "guard") || !g.tdTeachTrick(pet, "hunt") {
		t.Fatalf("expected tdTeachTrick to succeed for fetch, guard, and hunt")
	}

	// 1. Fetch item within 4 tiles when no enemies are adjacent
	gem := &Item{ID: g.newID(), Name: "star sapphire", Kind: "misc", Glyph: '*', F: CCyan, X: pet.X + 2, Y: pet.Y, Value: 50}
	g.Cur.Items = []*Item{gem}
	invBefore := len(g.P.Inv)
	g.tdRunTricks()
	if len(g.Cur.Items) != 0 || len(g.P.Inv) < invBefore+2 { // +1 fetched gem, +1 hunted meat/pelt
		t.Fatalf("expected fetch to pick up floor item and hunt to add prey; floor=%d inv=%d->%d",
			len(g.Cur.Items), invBefore, len(g.P.Inv))
	}

	// 2. Guard intercept when player HP <= MaxHP/2
	g.P.MaxHP = 100
	g.P.HP = 40
	petHPBefore := pet.HP
	rem := g.tdGuardIntercept(20)
	if rem != 10 || pet.HP != petHPBefore-10 {
		t.Fatalf("expected guard pet to intercept 10 of 20 dmg; rem=%d petHP=%d->%d", rem, petHPBefore, pet.HP)
	}
}

func TestTdPackAnimalsAndAnimalReputation(t *testing.T) {
	g := newGame(t, nil)
	clearAround(g)

	pet := g.Spawn("bear", 4, g.R)
	pet.Name = "Bramble"
	pet.AI = "follower"
	pet.Set("tm")
	g.Cur.Actors = append(g.Cur.Actors, pet)

	if !g.tdEquipPack(pet) || !pet.Has("td_pack") || g.tdCarryBonus() != 60 {
		t.Fatalf("expected pack saddlebags to grant +60 carry bonus, got %d", g.tdCarryBonus())
	}

	sword := g.MakeWeapon("longblade", 2, g.R)
	g.P.Inv = append(g.P.Inv, sword)
	if !g.tdPackPut(pet, sword) || len(g.td().PackItems[pet.Name]) != 1 {
		t.Fatalf("expected tdPackPut to store item in pet saddlebags")
	}
	got := g.tdPackTake(pet, 0)
	if got != sword || len(g.td().PackItems[pet.Name]) != 0 {
		t.Fatalf("expected tdPackTake to retrieve sword, got %+v", got)
	}

	// Animal reputation: wild beasts become non-hostile at >= 40 and drop herbs
	wild := g.Spawn("wolf", 2, g.R)
	wild.Faction = "beasts"
	wild.AI = "hostile"
	wild.Angry = true
	g.Cur.Actors = append(g.Cur.Actors, wild)
	itemsBefore := len(g.Cur.Items)

	g.tdAddAnimalRep(45, "healed wounded beasts")
	if g.tdAnimalRep() != 45 || g.Hostile(wild) || len(g.Cur.Items) <= itemsBefore {
		t.Fatalf("expected AnimalRep>=40 to make wild beasts non-hostile and drop herb; rep=%d hostile=%v items=%d->%d",
			g.tdAnimalRep(), g.Hostile(wild), itemsBefore, len(g.Cur.Items))
	}

	// Killing young beasts lowers reputation; <= -40 makes wild predators hostile with +1 speed
	speedBefore := wild.Speed
	g.tdAddAnimalRep(-90, "raided nests")
	if g.tdAnimalRep() > -40 || !g.Hostile(wild) || wild.Speed != speedBefore+1 {
		t.Fatalf("expected AnimalRep<=-40 to make wild beasts hostile with +1 speed; rep=%d hostile=%v speed=%d->%d",
			g.tdAnimalRep(), g.Hostile(wild), speedBefore, wild.Speed)
	}
}
