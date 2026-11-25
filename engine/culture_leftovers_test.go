package engine

import (
	"strings"
	"testing"
)

func msgsSince(g *Game, before int) string {
	var parts []string
	for i := before; i < len(g.Msgs); i++ {
		parts = append(parts, g.Msgs[i].T)
	}
	return strings.Join(parts, "\n")
}

func attachTestCulturePack(t *testing.T, g *Game) (*CultureData, *BurgData) {
	t.Helper()
	bd := g.localBurg()
	if bd == nil {
		t.Fatalf("expected a local burg in town zone")
	}
	cu := g.W.cultByID[bd.Culture]
	if cu == nil {
		t.Fatalf("expected culture %d to exist", bd.Culture)
	}
	orig := cu.Pack
	t.Cleanup(func() {
		cu.Pack = orig
		loTqWorld = nil
	})
	cu.Pack = &CulturePack{
		V: cpVersion,
		Customs: &CustomsPack{
			Law: &LawPack{
				Judge: "the Amber Magistrate",
				Entries: []LawEntry{
					{Crime: "theft", Punish: "threefold restitution", Note: "The guild marks the ledger."},
					{Crime: "assault", Punish: "a silver fine", Note: "Double if drawn in the square."},
					{Crime: "murder", Punish: "blood-price or the gallows", Note: "No sanctuary after dusk."},
					{Crime: "smuggling", Punish: "seizure and the stocks", Note: "Harbour wardens keep half."},
					{Crime: "trespass", Punish: "a warning and a toll", Note: ""},
				},
			},
		},
		Voice: &VoicePack{
			Register: "formal",
			Greet:    []string{"Peace to your road, {name}."},
			Farewell: []string{"Walk under clear skies."},
			Thanks:   []string{"My house is in your debt."},
			Apology:  []string{"Forgive this clumsiness."},
			Agree:    []string{"It shall be done as you ask."},
			Refuse:   []string{"That I cannot grant."},
			Praise:   []string{"Well struck and nobly done!"},
			Insults:  []string{"You have the manners of a ditch-crow!"},
			Curses:   []string{"May rust eat your blade!"},
			Oaths:    []string{"By the hearth-stone, I swear it."},
			Gossip:   []string{"Lean close, for the walls have ears."},
		},
		Material: &MaterialPack{
			TradeGoods: []TradeGood{
				{Name: "saffron lacquer", Kind: "luxury", Value: 45},
				{Name: "cedar resin", Kind: "timber", Value: 18},
			},
			Buildings: &BuildingStyle{
				Materials: []string{"marble", "timber"},
				House:     "carved marble courtyards with cedar shutters",
				Hall:      "pillared marble halls with bronze lintels",
				Temple:    "domed marble sanctuaries facing the dawn",
				Gate:      "twin marble arches hung with brass bells",
				Roof:      "glazed cobalt tiles",
				Color:     "chalk white and sea blue",
				Ornament:  "carved wave-scrolls",
			},
			Dress: map[string]string{
				"low":      "rough undyed linen tunics",
				"common":   "indigo-hemmed wool coats with brass clasps",
				"high":     "crimson silk robes embroidered with cranes",
				"ritual":   "saffron vestments bordered in silver",
				"war":      "lacquered lamellar over dark indigo",
				"mourning": "unbleached ash-grey shawls",
				"wedding":  "gold-threaded scarlet mantles",
			},
			Music: &MusicStyle{
				Instruments: []string{"moon-lute", "bamboo flute", "bronze gong", "horsehair fiddle"},
				Forms:       []string{"harbour ballad", "lantern reel"},
				Scale:       "pentatonic fifths",
			},
			Games: []GameRule{
				{Name: "Crown of Bones", Rule: "Cast three carved dice and call the highest pair."},
				{Name: "Citadel Siege", Rule: "Advance carved stones across the board to trap the king."},
			},
		},
	}
	loTqWorld = nil
	return cu, bd
}

func TestCultureLeftovers(t *testing.T) {
	t.Run("VoiceKinds", func(t *testing.T) {
		g := newGame(t, nil)
		_, bd := attachTestCulturePack(t, g)

		speaker := &Actor{
			ID:      9001,
			Name:    "Maren of the Quay",
			Tpl:     "npc",
			Species: "human",
			Role:    "merchant",
			State:   bd.State,
			HP:      20,
			MaxHP:   20,
			X:       g.P.X + 1,
			Y:       g.P.Y,
		}
		g.Cur.Actors = append(g.Cur.Actors, speaker)

		// Every supported dvSay kind must produce non-empty speech from the pack.
		kinds := []string{
			"greet", "farewell", "thanks", "apology", "agree", "refuse",
			"praise", "insult", "curse", "oath", "gossipOpen",
		}
		for _, k := range kinds {
			line := g.dvSay(speaker, k, nil)
			if line == "" {
				t.Errorf("dvSay(%q) returned empty string", k)
			}
			q := g.dvSayQ(speaker, k, nil)
			if !strings.HasPrefix(q, "\"") || !strings.HasSuffix(q, "\"") {
				t.Errorf("dvSayQ(%q) not quoted: %q", k, q)
			}
		}

		// Exercise the wired callers in culturefill.go.
		if got := g.loApology(speaker); !strings.Contains(got, "Forgive this clumsiness") {
			t.Errorf("loApology() = %q, want pack apology", got)
		}
		if got := g.loGossipOpen(speaker); !strings.Contains(got, "Lean close") {
			t.Errorf("loGossipOpen() = %q, want pack gossip opener", got)
		}

		before := len(g.Msgs)
		g.loOath(speaker)
		g.loPraise(speaker)
		g.loCrowdPraise()
		g.loHurt(speaker)
		joined := msgsSince(g, before)
		for _, want := range []string{"hearth-stone", "nobly done", "clumsiness"} {
			if !strings.Contains(joined, want) {
				t.Errorf("expected log to contain %q, got:\n%s", want, joined)
			}
		}

		// Hostile bark caller (insult / curse) via loTurn.
		foe := &Actor{
			ID:      9002,
			Name:    "Ragged Cutthroat",
			Tpl:     "npc",
			Species: "human",
			AI:      "hostile",
			Angry:   true,
			State:   bd.State,
			HP:      20,
			MaxHP:   20,
			X:       g.P.X + 1,
			Y:       g.P.Y,
		}
		g.Cur.Set(foe.X, foe.Y, TFloor)
		g.Cur.Actors = append(g.Cur.Actors, foe)
		g.recompute()
		g.Turn += 20
		before = len(g.Msgs)
		loTurn(g)
		barkLog := msgsSince(g, before)
		if !strings.Contains(barkLog, "ditch-crow") && !strings.Contains(barkLog, "rust eat your blade") {
			t.Errorf("loTurn() did not emit insult or curse bark, got:\n%s", barkLog)
		}
	})

	t.Run("CrimeLawLine", func(t *testing.T) {
		g := newGame(t, nil)
		cu, bd := attachTestCulturePack(t, g)

		// Assault bounty should surface the assault law line even when bounty < 150.
		before := len(g.Msgs)
		g.uwAddBounty(bd.State, 40, " (assault)")
		logAssault := msgsSince(g, before)
		if !strings.Contains(logAssault, "a silver fine") {
			t.Errorf("uwAddBounty(assault) missing etLawLine, got:\n%s", logAssault)
		}

		// Murder bounty should surface the murder law line.
		before = len(g.Msgs)
		g.uwAddBounty(bd.State, 150, " (murder)")
		logMurder := msgsSince(g, before)
		if !strings.Contains(logMurder, "blood-price or the gallows") {
			t.Errorf("uwAddBounty(murder) missing etLawLine, got:\n%s", logMurder)
		}

		// Gate search menu should surface smuggling law line.
		g.uwSearchMenu(bd.State, []*Item{{Name: "unmarked spice", Value: 50}}, false)
		popup := strings.Join(g.Popup, "\n")
		if !strings.Contains(popup, "seizure and the stocks") {
			t.Errorf("uwSearchMenu() missing smuggling etLawLine, got:\n%s", popup)
		}
		g.uwClose()

		// Jail should log the local law line too.
		before = len(g.Msgs)
		g.uwJail(bd.State)
		logJail := msgsSince(g, before)
		if !strings.Contains(logJail, "By the law of the "+cu.Name) {
			t.Errorf("uwJail() missing etLawLine, got:\n%s", logJail)
		}

		// Clean fallback when there is no law pack.
		cu.Pack.Customs.Law = nil
		if got := g.etLawLine(bd.State, "theft"); got != "" {
			t.Errorf("expected empty etLawLine without law pack, got %q", got)
		}
		before = len(g.Msgs)
		g.uwAddBounty(bd.State, 60, " (theft)")
		logFallback := msgsSince(g, before)
		if strings.Contains(logFallback, "By the law of") {
			t.Errorf("unexpected law line without law pack:\n%s", logFallback)
		}
	})

	t.Run("TradeGoodsLedger", func(t *testing.T) {
		g := newGame(t, nil)
		cu, bd := attachTestCulturePack(t, g)

		goods := g.tqGoodsHere()
		var found *tqGood
		for _, d := range goods {
			if d.Name == "saffron lacquer" {
				found = d
				break
			}
		}
		if found == nil {
			t.Fatalf("expected culture trade good 'saffron lacquer' in tqGoodsHere()")
		}
		if loTqCulture(found) != cu.ID {
			t.Errorf("loTqCulture(%q) = %d, want %d", found.ID, loTqCulture(found), cu.ID)
		}
		if !g.tqCanBuy(bd, found) {
			t.Errorf("expected town to stock its own culture trade good %q", found.Name)
		}
		price := g.tqPrice(bd, found, true)
		if price <= 0 {
			t.Errorf("tqPrice(%q) = %d, want > 0", found.Name, price)
		}
		if byID := tqGoodByID(found.ID); byID != found {
			t.Errorf("tqGoodByID(%q) = %v, want %v", found.ID, byID, found)
		}
		it := g.tqNewItem(found, 2, price, 0)
		if got := tqItemGood(it); got != found {
			t.Errorf("tqItemGood() = %v, want %v", got, found)
		}

		// Clean fallback when culture has no pack.
		cu.Pack = nil
		for _, d := range g.tqGoodsHere() {
			if strings.HasPrefix(d.ID, "cg:") && loTqCulture(d) == cu.ID {
				t.Errorf("unexpected culture trade good %q after clearing pack", d.Name)
			}
		}
	})

	t.Run("MaterialPackInPlay", func(t *testing.T) {
		g := newGame(t, nil)
		cu, bd := attachTestCulturePack(t, g)

		// 1. Buildings: materials, look text, enter line.
		w, _ := g.pbMaterials(cu, "house", TWoodWall, TWoodFloor)
		if w != TMarbleWall {
			t.Errorf("pbMaterials() wall = %v, want TMarbleWall", w)
		}
		g.Cur.Set(2, 2, TGateArch)
		if look := g.pbLook(g.Cur, 2, 2); !strings.Contains(look, "brass bells") {
			t.Errorf("pbLook(gate) = %q, want pack gate description", look)
		}
		g.Cur.Set(3, 2, TMarbleWall)
		if look := g.pbLook(g.Cur, 3, 2); !strings.Contains(look, "cobalt tiles") {
			t.Errorf("pbLook(wall) = %q, want pack roof description", look)
		}
		delete(g.pb().Seen, bd.ID)
		before := len(g.Msgs)
		pbEnter(g, g.Cur)
		if enterLog := msgsSince(g, before); !strings.Contains(enterLog, "carved marble courtyards") {
			t.Errorf("pbEnter() = %q, want pack house description", enterLog)
		}

		// 2. Dress: pdLook, pdFeast, pdCeremony, attire.go stockTailor & tailorMenu.
		citizen := &Actor{ID: 9101, Name: "Sera", Tpl: "npc", Role: "merchant", State: bd.State, HP: 10, MaxHP: 10}
		if look := g.pdLook(citizen); !strings.Contains(look, "indigo-hemmed wool coats") {
			t.Errorf("pdLook() = %q, want common dress", look)
		}
		if feast := g.pdFeast(bd, &FeastData{Name: "Dawn", Kind: "holy"}, false); !strings.Contains(feast, "saffron vestments") {
			t.Errorf("pdFeast() = %q, want ritual dress", feast)
		}
		if cer := g.pdCeremony(&fwCer{Kind: "wedding", Burg: bd.ID}); len(cer) == 0 || !strings.Contains(cer[0], "scarlet mantles") {
			t.Errorf("pdCeremony(wedding) = %v, want wedding dress", cer)
		}
		tailor := &Actor{ID: 9102, Name: "Lina", Tpl: "npc", Role: "tailor", Level: 3, State: bd.State, HP: 10, MaxHP: 10}
		g.stockTailor(tailor, bd, NewRng("test-42"))
		hasDressLore := false
		for _, it := range tailor.Inv {
			if strings.Contains(it.Lore, "indigo-hemmed wool coats") {
				hasDressLore = true
				break
			}
		}
		if !hasDressLore {
			t.Errorf("stockTailor() did not apply Material.Dress lore to local body garment")
		}
		g.tailorMenu(tailor)
		if !strings.Contains(strings.Join(g.Popup, "\n"), "indigo-hemmed wool coats") {
			t.Errorf("tailorMenu() popup missing Material.Dress note: %v", g.Popup)
		}

		// 3. Music: pmSet, pmBuskTip, pmFestivalLine (festivals.go dance contest).
		insts := g.pmSet([4]string{"lute", "flute", "drum", "fiddle"})
		joinedInsts := strings.Join(insts[:], ",")
		for _, want := range []string{"moon-lute", "bamboo flute", "bronze gong", "horsehair fiddle"} {
			if !strings.Contains(joinedInsts, want) {
				t.Errorf("pmSet() = %v, missing %q", insts, want)
			}
		}
		lute := &Item{Name: "moon-lute", Kind: "tool"}
		if tip := g.pmBuskTip(lute); tip < 2 {
			t.Errorf("pmBuskTip(moon-lute) = %d, want >= 2", tip)
		}
		line, bonus := g.pmFestivalLine(bd, &FeastData{Name: "Lantern Night"}, lute)
		if (!strings.Contains(line, "harbour ballad") && !strings.Contains(line, "lantern reel")) || bonus < 1 {
			t.Errorf("pmFestivalLine() = (%q, %d), want local music form and bonus >= 1", line, bonus)
		}

		// 4. Games: pgNames and pgRule.
		names := g.pgNames()
		joinedGames := strings.Join(names[:], ",")
		if !strings.Contains(joinedGames, "crown of bones") || !strings.Contains(joinedGames, "citadel siege") {
			t.Errorf("pgNames() = %v, want pack game names", names)
		}
		foundRule := false
		for k := 0; k < gmGameCount; k++ {
			if strings.Contains(g.pgRule(k), "Cast three carved dice") {
				foundRule = true
				break
			}
		}
		if !foundRule {
			t.Errorf("pgRule() did not return pack game rule for any table kind")
		}
	})
}
