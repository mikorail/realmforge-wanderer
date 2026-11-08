package engine

import (
	"strings"
	"testing"
)

func TestCeScrapInventionsAndBlueprints(t *testing.T) {
	g := newGame(t, nil)
	g.craftSt().Bits["1"] = 10

	it, ok := g.ceDesignInvention("Storm Pike", "hand", "shock", 4)
	if !ok || it == nil || it.Name != "Storm Pike" || it.Elem != "shock" {
		t.Fatalf("expected ceDesignInvention to succeed, got ok=%v it=%+v", ok, it)
	}
	if g.craftSt().Bits["1"] != 6 {
		t.Fatalf("expected 4 bits spent (6 remaining), got %d", g.craftSt().Bits["1"])
	}
	if g.ce().Blueprints["Storm Pike"] == nil {
		t.Fatalf("expected blueprint saved in ceState.Blueprints")
	}

	it2, ok2 := g.ceCraftSavedInvention("Storm Pike")
	if !ok2 || it2 == nil || it2.ID == it.ID {
		t.Fatalf("expected ceCraftSavedInvention to craft new instance, got ok=%v it2=%+v", ok2, it2)
	}
	if g.craftSt().Bits["1"] != 2 {
		t.Fatalf("expected 2 bits remaining, got %d", g.craftSt().Bits["1"])
	}

	// Insufficient bits should fail
	if _, ok3 := g.ceCraftSavedInvention("Storm Pike"); ok3 {
		t.Fatalf("expected crafting with insufficient bits (2 < 4) to fail")
	}
}

func TestCeEnchantingAndCursedItems(t *testing.T) {
	g := newGame(t, nil)
	blade := g.MakeWeapon("longblade", 2, g.R)
	pvBefore := blade.PV

	msg := g.ceEnchant(blade, "fire")
	if !strings.Contains(msg, "ce_ench:fire") || blade.Elem != "fire" || blade.PV != pvBefore+1 || blade.Extra["ce_ench:fire"] != 1 {
		t.Fatalf("expected fire rune enchantment on blade, got msg=%q elem=%q pv=%d", msg, blade.Elem, blade.PV)
	}

	if !g.ceCanUnequip(blade) {
		t.Fatalf("expected uncursed blade to be unequippable")
	}
	g.ceCurseItem(blade)
	if g.ceCanUnequip(blade) || blade.Extra["cursed"] != 1 {
		t.Fatalf("expected cursed blade to block unequip")
	}
	if !g.ceUncurse(blade) || !g.ceCanUnequip(blade) {
		t.Fatalf("expected ceUncurse to cleanse curse and allow unequip")
	}
}

func TestCeFishingAndFreeDiving(t *testing.T) {
	g := newGame(t, nil)
	clearAround(g)
	g.Cur.Set(g.P.X+1, g.P.Y, TDeep)

	fish, fmsg := g.ceFish()
	if fish == nil || g.ce().BestFishLbs <= 0 {
		t.Fatalf("expected ceFish near water to catch fish and record BestFishLbs; fish=%+v msg=%q", fish, fmsg)
	}

	prize, dmsg := g.ceFreeDive()
	if prize == nil || g.ce().MaxDiveDepth <= 0 {
		t.Fatalf("expected ceFreeDive near deep water to recover prize and record MaxDiveDepth; prize=%+v msg=%q", prize, dmsg)
	}
}

func TestCeCartographyQuestArrowAndPhotoMode(t *testing.T) {
	g := newGame(t, nil)
	z := g.Cur

	// Unseen zone (<50%) should reject charting
	for i := range z.Seen {
		z.Seen[i] = 0
	}
	if it, _ := g.ceChartZone(); it != nil {
		t.Fatalf("expected ceChartZone to fail when <50%% seen")
	}

	// Mark >=60% seen and chart zone
	for i := 0; i < len(z.Seen)*60/100; i++ {
		z.Seen[i] = 1
	}
	mapItem, _ := g.ceChartZone()
	if mapItem == nil || mapItem.Value != 65 || !strings.HasPrefix(mapItem.Name, "Survey Map of ") || !g.ce().ChartedZones[z.Key] {
		t.Fatalf("expected Survey Map worth 65 and zone marked charted, got %+v", mapItem)
	}

	// Quest arrow
	g.WX, g.WY = 10, 10
	g.ceSetQuestTarget(10, 5) // due North
	arrow := g.ceQuestArrow()
	if !strings.Contains(arrow, "↑") || !strings.Contains(arrow, "5p") {
		t.Fatalf("expected north arrow '↑' and '5p', got %q", arrow)
	}
	g.ceSetQuestTarget(10, 10)
	if !strings.Contains(g.ceQuestArrow(), "◎") {
		t.Fatalf("expected target marker '◎' at 0 distance, got %q", g.ceQuestArrow())
	}

	// Photo mode
	photo := g.cePhotoMode("Sunset at the market")
	if !strings.Contains(photo, "+------------------------+") || !strings.Contains(photo, g.P.Name) || !strings.Contains(photo, "Sunset at the market") {
		t.Fatalf("unexpected photo output:\n%s", photo)
	}
	if len(g.ce().Photos) != 1 {
		t.Fatalf("expected photo stored in ceState.Photos, got %d", len(g.ce().Photos))
	}
}
