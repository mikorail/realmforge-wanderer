package engine

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestStaplesSplitAndTag(t *testing.T) {
	st := staples("rice, coconut, fish sauce and chilli")
	if strings.Join(st, "|") != "rice|coconut|fish sauce|chilli" {
		t.Fatalf("staples = %q", st)
	}
	if stapleTag("fish sauce") != "deep" || stapleTag("chilli") != "ember" || stapleTag("rice") != "lasting" {
		t.Fatalf("tags: %s %s %s", stapleTag("fish sauce"), stapleTag("chilli"), stapleTag("rice"))
	}
	if _, ok := cookTags[stapleTag("taro")]; !ok {
		t.Fatal("fallback tag must be a real tag")
	}
}

func TestBeastCorpseIsIngredientPersonIsNot(t *testing.T) {
	g := newGame(t, nil)
	clearAround(g)
	f := placeFoe(g, "rat", 1)
	g.kill(f, g.P)
	var corpse *Item
	for _, it := range g.Cur.ItemsAt(f.X, f.Y) {
		if it.Kind == "corpse" {
			corpse = it
		}
	}
	if corpse == nil || corpse.Effect == "" {
		t.Fatalf("beast corpse should carry a tag: %+v", corpse)
	}
	for _, a := range g.Cur.Actors {
		if a.Role != "" {
			if corpseTag(a) != "" {
				t.Fatalf("townsfolk %s gave a cooking tag", a.Name)
			}
			break
		}
	}
}

func TestCookMealGrantsBuffsAndRecipe(t *testing.T) {
	g := newGame(t, nil)
	clearAround(g)
	g.P.Inv = append(g.P.Inv, g.makeIngredient("wild garlic", "hearty", 2, CGreen), g.makeIngredient("hare's sorrel", "spry", 1, CGreen))
	str, dv := g.P.Score(STR), g.P.DV()
	g.Hunger = 1500
	g.openCook()
	if g.Mode != "cook" {
		t.Fatalf("cook mode not opened: %s", g.Mode)
	}
	cands := g.cookCandidates()
	if len(cands) < 2 {
		t.Fatalf("expected 2 candidates, got %d", len(cands))
	}
	for i := range cands {
		g.Key(string(rune('a' + i)))
	}
	g.Key("Enter")
	if g.Mode != "main" || g.P.Meal == nil {
		t.Fatalf("meal not eaten: mode %s meal %v", g.Mode, g.P.Meal)
	}
	if g.P.Score(STR) != str+2 || g.P.DV() != dv+2 {
		t.Fatalf("buffs: STR %d→%d DV %d→%d", str, g.P.Score(STR), dv, g.P.DV())
	}
	if len(g.Cookbook) != 1 || g.Cookbook[0].Key != "hearty+spry" || g.Hunger >= 1500 {
		t.Fatalf("cookbook %+v hunger %d", g.Cookbook, g.Hunger)
	}
	// one wild garlic left, the sorrel used up
	n := 0
	for _, it := range g.P.Inv {
		if it.Kind == "ingredient" {
			n += it.Count
		}
	}
	if n != 1 {
		t.Fatalf("ingredients left = %d, want 1", n)
	}
	// survives a save round trip, then wears off
	b, _ := json.Marshal(g)
	var g2 Game
	if err := json.Unmarshal(b, &g2); err != nil || g2.MealName == "" || len(g2.Cookbook) != 1 {
		t.Fatalf("save lost the meal: %v %q", err, g2.MealName)
	}
	g.MealLeft = 1
	g.mealTick()
	if g.P.Meal != nil || g.P.Score(STR) != str {
		t.Fatal("meal should wear off")
	}
}

func TestEmberMealHalvesFire(t *testing.T) {
	g := newGame(t, nil)
	g.P.Meal = map[string]int{"ember": 1}
	hp := g.P.HP
	g.hurt(g.P, 10, nil, "fire")
	if hp-g.P.HP != 5 {
		t.Fatalf("fire dealt %d, want 5", hp-g.P.HP)
	}
	g.P.Meal = map[string]int{"rime": 2}
	hp = g.P.HP
	g.hurt(g.P, 8, nil, "frost")
	if hp-g.P.HP != 2 {
		t.Fatalf("frost dealt %d, want 2", hp-g.P.HP)
	}
}

func TestMerchantsSellLocalStaplesAndWildsHaveHerbs(t *testing.T) {
	g := newGame(t, nil)
	var bd *BurgData
	for i := range g.W.Burgs {
		if cu := g.W.cultByID[g.W.Burgs[i].Culture]; cu != nil && cu.Food != "" {
			bd = &g.W.Burgs[i]
			break
		}
	}
	if bd == nil {
		t.Fatal("no burg with a culture food line")
	}
	cu := g.W.cultByID[bd.Culture]
	m := g.makeNPC("merchant", bd, cu, NewRng("m"))
	found := false
	for _, it := range m.Inv {
		if it.Kind == "ingredient" && strings.Contains(cu.Food, it.Name) {
			found = true
		}
	}
	if !found {
		t.Fatalf("merchant of %s sells none of %q", cu.Name, cu.Food)
	}
	herbs := 0
	for zx := 0; zx < 3; zx++ {
		for zy := 0; zy < 3; zy++ {
			for wx := 0; wx < g.W.W && herbs == 0; wx += 7 {
				zn := g.GetZone(wx, g.WY, zx, zy, 0)
				if zn == nil || zn.Kind != "surface" {
					continue
				}
				for _, it := range zn.Items {
					if it.Kind == "ingredient" {
						herbs++
					}
				}
			}
		}
	}
	if herbs == 0 {
		t.Fatal("no wild herbs on any sampled surface zone")
	}
}

func TestSimpleMealFromFood(t *testing.T) {
	g := newGame(t, nil)
	clearAround(g)
	g.Hunger = 1500
	g.P.HP = g.P.MaxHP / 2
	g.openCook()
	// deselect nothing: Enter cooks plain food
	g.Key("Enter")
	if g.Hunger >= 1500 || g.P.HP <= g.P.MaxHP/2 {
		t.Fatalf("simple meal: hunger %d hp %d", g.Hunger, g.P.HP)
	}
}
