package engine

import (
	"strings"
	"testing"
)

func TestAutoExploreUnseenAndItems(t *testing.T) {
	g := newGame(t, nil)
	z := g.Cur
	z.Actors = nil
	z.Items = nil
	for y := 0; y < MH; y++ {
		for x := 0; x < MW; x++ {
			if x == 0 || y == 0 || x == MW-1 || y == MH-1 {
				z.Set(x, y, TWall)
			} else {
				z.Set(x, y, TFloor)
			}
			z.Seen[y*MW+x] = 1
			z.Fire[y*MW+x] = 0
			z.Liq[y*MW+x] = 0
		}
	}
	z.Traps = nil
	g.P.X, g.P.Y = 10, 10
	g.P.HP = g.P.MaxHP

	// Fully explored zone with no items should take 0 steps and report nothing left
	if steps := g.AutoExplore(); steps != 0 {
		t.Fatalf("expected 0 steps in fully seen empty zone, got %d", steps)
	}
	if len(g.Msgs) == 0 || !strings.Contains(g.Msgs[len(g.Msgs)-1].T, "Nothing left to explore here.") {
		t.Fatalf("expected 'Nothing left to explore here.' message, got %+v", g.Msgs)
	}

	// Clutter item should be ignored by AutoExplore
	cl := &Item{ID: g.newID(), Kind: "clutter", Name: "tin spoon", Glyph: ',', F: CGray, X: 14, Y: 10}
	z.Items = []*Item{cl}
	if steps := g.AutoExplore(); steps != 0 {
		t.Fatalf("expected clutter item to be ignored by AutoExplore, got %d steps", steps)
	}

	// Non-clutter floor item should be targeted and reached
	tonic := g.MakeTonic("heal", 1)
	tonic.X, tonic.Y = 15, 10
	z.Items = []*Item{tonic}
	steps := g.AutoExplore()
	if steps != 5 || g.P.X != 15 || g.P.Y != 10 {
		t.Fatalf("expected 5 steps to reach tonic at (15,10), got steps=%d pos=(%d,%d)", steps, g.P.X, g.P.Y)
	}

	// Mark a strip at x >= 22 as unseen and verify AutoExplore walks toward it
	z.Items = nil
	for x := 22; x < MW-1; x++ {
		z.Seen[10*MW+x] = 0
	}
	steps = g.AutoExplore()
	if steps <= 0 || g.P.X <= 15 {
		t.Fatalf("expected AutoExplore to step toward unseen tiles, got steps=%d pos=(%d,%d)", steps, g.P.X, g.P.Y)
	}
}

func TestAutoExploreStopConditions(t *testing.T) {
	g := newGame(t, nil)
	z := g.Cur
	z.Actors = nil
	z.Items = nil
	for y := 0; y < MH; y++ {
		for x := 0; x < MW; x++ {
			if x == 0 || y == 0 || x == MW-1 || y == MH-1 {
				z.Set(x, y, TWall)
			} else {
				z.Set(x, y, TFloor)
			}
			z.Seen[y*MW+x] = 0
		}
	}
	g.P.X, g.P.Y = 10, 10
	g.recompute()

	// 1. Stop when hostile is visible
	foe := g.Spawn("bandit", 2, NewRng("qol:foe"))
	foe.AI = "hostile"
	foe.Angry = true
	foe.Faction = "outlaws"
	foe.X, foe.Y = 13, 10
	z.Actors = []*Actor{foe}
	g.recompute()
	if steps := g.AutoExplore(); steps != 0 {
		t.Fatalf("expected AutoExplore to refuse when hostile is visible, got %d steps", steps)
	}
	z.Actors = nil

	// 2. Stop when HP is below 40%
	g.P.HP = g.P.MaxHP * 30 / 100
	if steps := g.AutoExplore(); steps != 0 {
		t.Fatalf("expected AutoExplore to refuse when HP < 40%%, got %d steps", steps)
	}
	g.P.HP = g.P.MaxHP

	// 3. Stop when trap/hazard is adjacent
	z.Set(11, 10, TTrap)
	if steps := g.AutoExplore(); steps != 0 {
		t.Fatalf("expected AutoExplore to stop when adjacent to TTrap, got %d steps", steps)
	}
	z.Set(11, 10, TFloor)

	// 4. Keybindings Tab and v invoke AutoExplore
	if mainKeys["Tab"] == nil || mainKeys["v"] == nil {
		t.Fatalf("expected mainKeys[Tab] and mainKeys[v] to be registered")
	}
}

func TestAutoPickupRules(t *testing.T) {
	g := newGame(t, nil)
	clearAround(g)
	g.Cur.Actors = nil
	g.Cur.Items = nil

	st := g.ql()
	if !st.Gold || !st.Ammo || st.Tonics || st.Ingredients {
		t.Fatalf("unexpected default qlState: %+v", st)
	}

	goldBefore := g.P.Gold
	invBefore := len(g.P.Inv)

	goldIt := &Item{ID: g.newID(), Kind: "gold", Name: "coins", Count: 25, X: g.P.X, Y: g.P.Y}
	ammoIt := g.MakeAmmo(10)
	ammoIt.X, ammoIt.Y = g.P.X, g.P.Y
	tonicIt := g.MakeTonic("heal", 1)
	tonicIt.X, tonicIt.Y = g.P.X, g.P.Y
	ingIt := g.makeIngredient("wild mint", "spry", 1, CGreen)
	ingIt.X, ingIt.Y = g.P.X, g.P.Y
	ownedGold := &Item{ID: g.newID(), Kind: "gold", Name: "shop coins", Count: 50, X: g.P.X, Y: g.P.Y, Extra: map[string]int{"uw:own": 1}}

	g.Cur.Items = []*Item{goldIt, ammoIt, tonicIt, ingIt, ownedGold}
	g.qlAutoPickup()

	if g.P.Gold != goldBefore+25 {
		t.Fatalf("expected gold to increase by 25 (ignoring owned gold), got %d -> %d", goldBefore, g.P.Gold)
	}
	// Tonic and ingredient should still be on the ground
	rem := g.Cur.ItemsAt(g.P.X, g.P.Y)
	if len(rem) != 3 {
		t.Fatalf("expected tonic, ingredient, and owned gold to remain on floor, got %d items", len(rem))
	}

	// Toggle tonics and ingredients on
	g.qlToggleRule("tonics")
	g.qlToggleRule("ingredients")
	if !st.Tonics || !st.Ingredients {
		t.Fatalf("expected Tonics and Ingredients rules to be toggled on")
	}
	g.qlAutoPickup()
	rem = g.Cur.ItemsAt(g.P.X, g.P.Y)
	if len(rem) != 1 || rem[0] != ownedGold {
		t.Fatalf("expected only ownedGold to remain on floor after enabling tonics/ingredients, got %+v", rem)
	}
	if len(g.P.Inv) <= invBefore {
		t.Fatalf("expected inventory to contain picked up items")
	}
}

func TestFilterMsgsAndCategories(t *testing.T) {
	g := newGame(t, nil)
	g.Msgs = nil

	g.Msg("You hit the cave bear for 8 damage.")
	g.Msg("You discover {{Y|the Sunken Shrine}}! Secret recorded in your journal.")
	g.Msg("You sold an iron axe to the merchant for 18 coins.")
	g.Msg("{{m|Rival}}: Kaelen the scout was last seen near Oakhaven.")
	g.Msg("You wait a moment.")

	if got := len(g.FilterMsgs("all", "")); got != 5 {
		t.Fatalf("expected 5 messages for 'all', got %d", got)
	}
	if got := g.FilterMsgs("combat", ""); len(got) != 1 || !strings.Contains(got[0].T, "cave bear") {
		t.Fatalf("expected 1 combat message, got %+v", got)
	}
	if got := g.FilterMsgs("lore", ""); len(got) != 1 || !strings.Contains(got[0].T, "Sunken Shrine") {
		t.Fatalf("expected 1 lore message, got %+v", got)
	}
	if got := g.FilterMsgs("trade", ""); len(got) != 1 || !strings.Contains(got[0].T, "merchant") {
		t.Fatalf("expected 1 trade message, got %+v", got)
	}
	if got := g.FilterMsgs("rumor", ""); len(got) != 1 || !strings.Contains(got[0].T, "Rival") {
		t.Fatalf("expected 1 rumor message, got %+v", got)
	}
	if got := g.FilterMsgs("all", "OAKHAVEN"); len(got) != 1 {
		t.Fatalf("expected case-insensitive query 'OAKHAVEN' to match 1 message, got %d", len(got))
	}
}

func TestExportDiary(t *testing.T) {
	g := newGame(t, nil)
	g.Turn = DayLen * 5
	g.Kills = 7
	g.fm().Km = 42.5
	g.deed("slay", "Tester slew the ash wyrm of Red Peak.", 0, 0)
	g.Feasts = append(g.Feasts, FeastLog{Name: "Harvest Vigil", Town: "Oakhaven", Year: g.W.Year, Day: 3, Text: "Danced until dawn."})
	g.fm().Songs = append(g.fm().Songs, &fmSong{Text: "How Tester stood at Red Peak.", Sung: 3})

	md := g.ExportDiary()
	for _, want := range []string{
		"# Chronicle of Tester",
		"- **Name**: Tester",
		"- **Calling**:",
		"- **Genre**:",
		"- **Planet**:",
		"- **Level**:",
		"- **Score**:",
		"- **Turns / Days**:",
		"42.5 km",
		"- **Kills**: 7",
		"## Equipped Gear",
		"## Family & Fame",
		"## Major Deeds",
		"Tester slew the ash wyrm of Red Peak.",
		"## Feasts & Songs",
		"Harvest Vigil",
		"How Tester stood at Red Peak.",
		"## Final Fate",
	} {
		if !strings.Contains(md, want) {
			t.Errorf("ExportDiary missing %q in:\n%s", want, md)
		}
	}
	if strings.Contains(md, "—") {
		t.Errorf("ExportDiary must not contain em dashes")
	}
}
