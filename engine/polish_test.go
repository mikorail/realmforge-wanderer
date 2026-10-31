package engine

import (
	"strings"
	"testing"
)

// addInstaller puts an augment installer next to the player in the current zone.
func addInstaller(t *testing.T, g *Game) *Actor {
	t.Helper()
	bd := &g.W.Burgs[0]
	inst := g.makeNPC("augmenter", bd, g.W.cultByID[bd.Culture], NewRng("polish:inst"))
	inst.Title = g.augSkin().Installer
	inst.AI = "still"
	inst.X, inst.Y = g.P.X+1, g.P.Y
	g.Cur.Set(inst.X, inst.Y, TFloor)
	g.Cur.Actors = append(g.Cur.Actors, inst)
	return inst
}

func pickOpt(t *testing.T, g *Game, sub string) {
	t.Helper()
	for _, o := range g.Opts {
		if strings.Contains(o.Text, sub) {
			o.Do()
			return
		}
	}
	var l []string
	for _, o := range g.Opts {
		l = append(l, o.Text)
	}
	t.Fatalf("no option containing %q in %v", sub, l)
}

func hasOpt(g *Game, sub string) bool {
	for _, o := range g.Opts {
		if strings.Contains(o.Text, sub) {
			return true
		}
	}
	return false
}

func TestCompanionAugments(t *testing.T) {
	g := newGame(t, nil)
	a := recruitOne(t, g)
	a.Attr[TOU], a.Level = 16, 1 // capacity 3
	for _, p := range a.Body {
		p.Item = nil
	}
	inst := addInstaller(t, g)
	g.P.Gold = 5000
	pLoad := g.P.AugLoad()
	str := a.Score(STR)
	g.give(g.MakeAugment("servo"))
	g.give(g.MakeAugment("dermal"))

	// the companion's talk menu leads to the installer, working on the companion
	g.openTalk(a)
	pickOpt(t, g, "fit you with")
	if g.Talk != inst {
		t.Fatal("installer menu not opened")
	}
	pickOpt(t, g, "in "+a.Name)
	if a.Score(STR) != str+2 || a.AugLoad() != 2 || g.P.AugLoad() != pLoad {
		t.Fatalf("companion STR %d→%d load %d, player load %d", str, a.Score(STR), a.AugLoad(), g.P.AugLoad())
	}
	// capacity is the companion's own: dermal (2) no longer fits into 3
	if !hasOpt(g, "needs 2 capacity, 1 free") {
		t.Fatal("companion capacity not enforced")
	}
	// the installer's own menu can switch patients
	g.augMenu(inst)
	pickOpt(t, g, "Work on "+a.Name)
	if !hasOpt(g, "Remove the") {
		t.Fatal("companion augment not listed for removal")
	}
	pickOpt(t, g, "Work on me")

	// effects follow the companion through save/load
	h, err := LoadGame(g.W, g.Save())
	if err != nil {
		t.Fatal(err)
	}
	var b *Actor
	for _, o := range h.Cur.Actors {
		if o.ID == a.ID {
			b = o
		}
	}
	if b == nil || b.AugLoad() != 2 || b.Score(STR) != str+2 {
		t.Fatal("companion augment lost in save/load")
	}
	// remove it again: back in the player's pack, STR restored
	g.augMenuFor(inst, a)
	pickOpt(t, g, "Remove the")
	if a.AugLoad() != 0 || a.Score(STR) != str {
		t.Fatal("removal did not restore the companion")
	}
	found := false
	for _, it := range g.P.Inv {
		if it.Kind == "augment" && it.Effect == "servo" {
			found = true
		}
	}
	if !found {
		t.Fatal("removed augment not returned to the player")
	}
	// refit and part ways: the augment stays in them
	g.augMenuFor(inst, a)
	pickOpt(t, g, "in "+a.Name)
	g.releaseCompanion(a)
	if a.Comp != nil || a.AugLoad() != 2 {
		t.Fatal("augment lost on parting ways")
	}
}

func TestCompanionFood(t *testing.T) {
	g := newGame(t, nil)
	a := recruitOne(t, g)
	var dish *Item
	for i := range g.W.Cultures {
		if dish = g.makeDish(&g.W.Cultures[i], 2); dish != nil {
			break
		}
	}
	if dish == nil {
		t.Fatal("no culture has a dish")
	}
	g.give(dish)
	a.Comp.Loyalty = 50
	msg := g.compGive(a, dish)
	if len(a.Meal) == 0 || a.St("fed") <= 0 || a.Comp.Loyalty <= 50 {
		t.Fatalf("dish not eaten: meal %v fed %d loyalty %d (%s)", a.Meal, a.St("fed"), a.Comp.Loyalty, msg)
	}
	ing := g.makeIngredient("wild onion", "hearty", 1, CGreen)
	g.give(ing)
	if !giftable(ing) {
		t.Fatal("ingredients not giftable")
	}
	l := a.Comp.Loyalty
	g.compGive(a, ing)
	if a.Comp.Loyalty <= l {
		t.Fatal("raw ingredient gave no loyalty")
	}

	// shared meals: a small bonus, at most every 6 hours
	a.Comp.Loyalty = 50
	g.applyMeal("Test stew", map[string]int{"spry": 1}, 600, true)
	if a.Comp.Loyalty != 53 || a.mealLv("spry") != 1 {
		t.Fatalf("shared meal: loyalty %d spry %d", a.Comp.Loyalty, a.mealLv("spry"))
	}
	g.applyMeal("Test stew", map[string]int{"spry": 1}, 600, true)
	if a.Comp.Loyalty != 53 {
		t.Fatalf("shared-meal bonus not rate limited: %d", a.Comp.Loyalty)
	}
	g.Turn += 400
	g.applyMeal("Test stew", map[string]int{"spry": 1}, 600, true)
	if a.Comp.Loyalty != 56 {
		t.Fatalf("shared meal after 6h: %d", a.Comp.Loyalty)
	}
	// hunger still costs loyalty
	g.Hunger = 3000
	g.Turn = (g.Turn/60 + 1) * 60
	g.companionsTick()
	if a.Comp.Loyalty != 55 {
		t.Fatalf("starving did not cost loyalty: %d", a.Comp.Loyalty)
	}
}

func TestMusicStateWeather(t *testing.T) {
	g := newGame(t, nil)
	m := g.MusicState()
	if m.Season != g.Season() {
		t.Fatalf("season %d, want %d", m.Season, g.Season())
	}
	n := m.Bolt
	g.lightning()
	if g.MusicState().Bolt != n+1 {
		t.Fatal("lightning did not bump the music bolt counter")
	}
}
