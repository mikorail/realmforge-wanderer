package engine

import (
	"strings"
	"testing"
)

func TestTrVariantsForAllSpecies(t *testing.T) {
	species := []string{
		"orc", "goblin", "elf", "wood_elf", "dark_elf", "void_elf",
		"dwarf", "chaos_dwarf", "lizardfolk", "saurian", "troll", "ogor",
		"beastman", "ratfolk", "merfolk", "human", "halfling", "giant",
		"celestial", "tomb_undead", "warp_fiend", "chitin_swarm",
	}
	for _, sp := range species {
		vs := trVariantsForSpecies(sp)
		if len(vs) < 3 {
			t.Fatalf("species %q returned only %d variants, want >= 3", sp, len(vs))
		}
		for _, v := range vs {
			if v.ID == "" || v.Name == "" || v.SkinHex == "" || v.AccentHex == "" || v.Totem == "" || v.Color == 0 {
				t.Fatalf("species %q has incomplete variant: %+v", sp, v)
			}
		}
		r1 := NewRng("tr_test:" + sp)
		r2 := NewRng("tr_test:" + sp)
		p1 := trPickVariant(sp, 13, r1)
		p2 := trPickVariant(sp, 13, r2)
		if p1.ID != p2.ID {
			t.Fatalf("trPickVariant not deterministic for %q: %s vs %s", sp, p1.ID, p2.ID)
		}
	}
}

func TestTrApplyVariant(t *testing.T) {
	g := newGame(t, nil)
	a := g.Spawn("bandit", 3, NewRng("tr_apply"))
	origHP := a.HP
	origAV := a.AVNat
	origDV := a.DVNat
	origSpd := a.Speed
	v := trVariantsForSpecies("orc")[0] // Crimson Bloodfang
	trApplyVariant(a, v)

	if a.F != v.Color {
		t.Fatalf("expected actor color %d, got %d", v.Color, a.F)
	}
	if a.HP != origHP+v.HP || a.AVNat != origAV+v.AV || a.DVNat != origDV+v.DV || a.Speed != origSpd+v.Speed {
		t.Fatalf("stats not updated properly by trApplyVariant")
	}
	if !a.Has("tr_var:" + v.ID) {
		t.Fatalf("missing tr_var:%s flag", v.ID)
	}
	if !a.Has("tr_hex:" + v.SkinHex + ":" + v.AccentHex) {
		t.Fatalf("missing tr_hex flag")
	}
	if !a.Has("tr_totem:" + v.Totem) {
		t.Fatalf("missing tr_totem:%s flag", v.Totem)
	}
	if !strings.HasPrefix(a.Name, v.Name+" ") {
		t.Fatalf("expected actor name prefixed with %q, got %q", v.Name, a.Name)
	}
	if !strings.Contains(a.Desc, v.Totem) {
		t.Fatalf("expected actor desc to mention totem %q, got %q", v.Totem, a.Desc)
	}
}

func TestTrSpawnCampAndInteractions(t *testing.T) {
	g := newGame(t, nil)
	clearAround(g)
	zn := g.Cur
	zn.Kind = ZWild
	zn.Z = 0

	r2 := trSideRng(g.W, zn.WX, zn.WY, zn.Z)
	trSpawnCamp(g, zn, r2)

	var warriors int
	var shaman, chief *Actor
	for _, a := range zn.Actors {
		switch a.Role {
		case "tr_warrior":
			warriors++
		case "tr_shaman":
			shaman = a
		case "tr_chief":
			chief = a
		}
	}
	if warriors < 3 {
		t.Fatalf("expected at least 3 tribal warriors in camp, got %d", warriors)
	}
	if shaman == nil {
		t.Fatalf("expected a Tribal Shaman (tr_shaman) in camp")
	}
	if chief == nil || !chief.Boss {
		t.Fatalf("expected a boss Tribal Chieftain (tr_chief) in camp")
	}
	tid := trTribeOfActor(chief)
	if tid == "" {
		t.Fatalf("expected chief to have a tribe ID via trTribeOfActor")
	}

	// Verify talk options on chieftain
	opts := trTalkHook(g, chief)
	if len(opts) < 3 {
		t.Fatalf("expected 3 talk options on chieftain, got %d", len(opts))
	}

	// 1. Parley
	g.P.Gold = 100
	chief.Angry = true
	chief.AI = "hostile"
	if !g.trParley(chief) {
		t.Fatalf("expected trParley to succeed")
	}
	if g.P.Gold != 75 || g.trTribeSt().Rep[tid] < 25 || chief.Angry || chief.AI != "town" {
		t.Fatalf("parley state mismatch: gold=%d rep=%d angry=%v ai=%s", g.P.Gold, g.trTribeSt().Rep[tid], chief.Angry, chief.AI)
	}

	// 2. War-paint
	v := trVariantOfActor(g, shaman)
	av0, dv0 := g.P.AVNat, g.P.DVNat
	if !g.trApplyWarPaint(v) {
		t.Fatalf("expected trApplyWarPaint to succeed")
	}
	if !g.P.Has("tr_paint:"+v.ID) || !g.P.Has("tr_hex:"+v.SkinHex+":"+v.AccentHex) {
		t.Fatalf("expected war-paint and hex flags on player")
	}
	if g.P.AVNat != av0+1 || g.P.DVNat != dv0+1 {
		t.Fatalf("expected +1 AV and +1 DV from war-paint, got AV %d->%d DV %d->%d", av0, g.P.AVNat, dv0, g.P.DVNat)
	}

	// 3. Challenge Chieftain to Rite of Strength
	g.P.Attr[STR] = 26
	inv0 := len(g.P.Inv)
	if !g.trChallengeChief(chief) {
		t.Fatalf("expected trChallengeChief to succeed with high STR")
	}
	if !g.P.Has("tr_bloodchief:" + tid) {
		t.Fatalf("expected player to receive tr_bloodchief:%s flag", tid)
	}
	if g.trTribeSt().Rep[tid] < 125 {
		t.Fatalf("expected +100 standing from defeating chieftain, got %d", g.trTribeSt().Rep[tid])
	}
	if len(g.P.Inv) <= inv0 {
		t.Fatalf("expected unique Tribal Totem Relic added to inventory")
	}

	// 4. Side hook displays active war-paint and Blood-Chief title
	lines := strings.Join(trSideHook(g), " | ")
	if !strings.Contains(lines, "War-paint:") || !strings.Contains(lines, "Blood-Chief") {
		t.Fatalf("expected trSideHook to display War-paint and Blood-Chief, got %q", lines)
	}
}
