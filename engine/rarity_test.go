package engine

import (
	"strings"
	"testing"
)

func TestRarityDistribution(t *testing.T) {
	g := newGame(t, nil)
	r := NewRng("rarity-dist")
	for _, lv := range []int{1, 15, 35} {
		cnt := map[string]int{}
		for i := 0; i < 20000; i++ {
			cnt[g.rollRarity(r, lv)]++
		}
		t.Logf("level %d: %v", lv, cnt)
		if cnt["unique"] != 0 {
			t.Fatalf("unique rolled at level %d", lv)
		}
		for k, v := range cnt {
			if k != "common" && v > cnt["common"] && lv == 1 {
				t.Fatalf("level 1: %s beats common", k)
			}
		}
		if lv == 1 && (cnt["common"] < 14000 || cnt["legendary"] != 0) {
			t.Fatalf("level 1 distribution off: %v", cnt)
		}
		if lv == 35 && cnt["legendary"] == 0 {
			t.Fatalf("no legendary at 35: %v", cnt)
		}
		if lv == 15 && cnt["legendary"] == 0 {
			t.Fatalf("no legendary at 15: %v", cnt)
		}
	}
}

func TestRarityValueMonotonic(t *testing.T) {
	g := newGame(t, nil)
	prev := 0
	for _, rar := range rarityOrder[:6] {
		r := NewRng("val")
		w := g.MakeWeapon("longblade", 2, r)
		base := w.Value
		g.applyRarity(w, r, rar)
		if rar == "common" {
			if w.Value != base || w.Rarity != "common" {
				t.Fatal("common changed")
			}
		}
		if w.Value < prev || (rar != "common" && w.Value == prev) {
			t.Fatalf("%s value %d not above %d", rar, w.Value, prev)
		}
		if w.Name == "" {
			t.Fatal("empty name")
		}
		prev = w.Value
	}
}

func TestRarityConsumablesStayCommon(t *testing.T) {
	g := newGame(t, nil)
	r := NewRng("c")
	for _, it := range []*Item{g.MakeAmmo(5), g.MakeFood(r, 1), g.MakeTonic("heal", 1)} {
		g.applyRarity(it, r, "legendary")
		if it.Rarity != "common" || it.Cat != "" {
			t.Fatalf("%s changed", it.Name)
		}
	}
}

func TestRarityUnique(t *testing.T) {
	g := newGame(t, nil)
	r := NewRng("u")
	u := g.MakeUnique(g.MakeWeapon("axe", 3, r), "Doomcleaver", "Forged in grief.")
	if u.Rarity != "unique" || !u.Ident || u.Name != "Doomcleaver" || u.Lore == "" || u.F != CYellow {
		t.Fatalf("bad unique %+v", u)
	}
	if !strings.HasPrefix(u.RarityTag(), "{{W|[Unique]}}") {
		t.Fatalf("tag %q", u.RarityTag())
	}
	for i := 0; i < 500; i++ {
		if g.rollRarity(r, 40) == "unique" {
			t.Fatal("rolled unique")
		}
	}
}

func TestRarityCategoriesAndGenres(t *testing.T) {
	g := newGame(t, nil)
	r := NewRng("cats")
	seen := map[string]bool{}
	for i := 0; i < 4000; i++ {
		it := g.MakeWeapon("longblade", 3, r)
		g.applyRarity(it, r, Pick(r, []string{"rare", "ultrarare", "legendary", "mythic"}))
		if it.Cat != "" {
			seen[it.Cat] = true
		}
		if it.Cat == "omni" && rarityIndex(it.Rarity) < 4 {
			t.Fatal("omni below legendary")
		}
		if it.Name == "" || it.RarityTag() == "" {
			t.Fatal("empty name/tag")
		}
	}
	for _, c := range rarityCats {
		if !seen[c] {
			t.Errorf("category %s never rolled", c)
		}
	}
	for _, genre := range []string{"fantasy", "mythic", "steampunk", "dieselpunk", "atompunk", "cyberpunk", "biopunk", "solarpunk"} {
		g.Genre = genre
		for _, c := range rarityCats {
			if g.CatLabel(c) == "" {
				t.Fatalf("%s/%s empty label", genre, c)
			}
			it := g.MakeArmor("body", 4, r)
			g.applyCategory(it, r, c, 4)
			g.rarityName(it, r, 4, c)
			if it.Name == "" {
				t.Fatalf("%s/%s empty name", genre, c)
			}
		}
		if bd := g.bossDrop(r, 20); bd == nil || bd.Name == "" || rarityIndex(bd.Rarity) < 2 {
			t.Fatalf("%s bad boss drop %+v", genre, bd)
		}
	}
	g.Genre = "cyberpunk"
	if g.CatLabel("biblical") != "Neo-Gospel" {
		t.Fatal("cyberpunk biblical")
	}
}

func TestRarityCategoryDamage(t *testing.T) {
	g := newGame(t, nil)
	w := &Item{Kind: "weapon", Rarity: "legendary", Cat: "biblical"}
	if g.categoryBonusDamage(w, &Actor{Faction: "undead"}) < 1 {
		t.Fatal("no bonus vs undead")
	}
	if g.categoryBonusDamage(w, &Actor{Faction: "beasts"}) != 0 {
		t.Fatal("bonus vs beasts")
	}
	if g.categoryBonusDamage(nil, nil) != 0 {
		t.Fatal("nil")
	}
}
