package engine

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

func TestStealthSystem(t *testing.T) {
	g := newGame(t, nil)
	clearAround(g)

	st := g.ss()
	st.Sneak = false
	if got := g.ssStep(g.P.X+1, g.P.Y, 1000); got != 1000 {
		t.Fatalf("expected normal step cost 1000 when not sneaking, got %d", got)
	}
	st.Sneak = true
	if got := g.ssStep(g.P.X+1, g.P.Y, 1000); got != 1500 {
		t.Fatalf("expected sneak step cost 1500, got %d", got)
	}

	// Trail recording and rendering
	st.addMark(g.Cur.Key, ssMark{X: g.P.X + 1, Y: g.P.Y, T: g.Turn, W: 42, B: true, Seen: true})
	gl, col, ok := g.ssTrailAt(g.P.X+1, g.P.Y)
	if !ok || gl != ',' || col != CDarkRed {
		t.Fatalf("expected blood trail ',' CDarkRed at (%d,%d), got ok=%v gl=%q col=%d", g.P.X+1, g.P.Y, ok, gl, col)
	}

	// Hiding and noise
	st.Alert = map[int]int{}
	st.Detected = 0
	if status := g.ssStatus(); status != "Hidden" {
		t.Fatalf("expected status Hidden, got %q", status)
	}
	foe := placeFoe(g, "bandit", 2)
	foe.Angry = true
	g.ssNoise(g.P.X, g.P.Y, 5)
	if st.Alert[foe.ID] <= g.Turn {
		t.Fatalf("expected ssNoise to set Loud and alert nearby foe")
	}

	// Disguises
	if g.HomeState > 0 {
		fac := fmt.Sprintf("state:%d", g.HomeState)
		sp := g.stateSphere(g.HomeState)
		if sp != "" {
			g.P.Attr[EGO] = 22
			g.Skills["persuasion"] = 3
			for _, slot := range []string{"body", "head", "back"} {
				if bp := g.P.PartFor(slot); bp != nil {
					bp.Item = &Item{ID: g.newID(), Kind: "armor", Slot: slot, Name: "regional garb", Region: sp}
				}
			}
			if g.ssPieces(fac) < 2 || !g.ssDisguised(fac) {
				t.Fatalf("expected player in regional attire to be disguised for %s (pieces=%d, rating=%d)", fac, g.ssPieces(fac), g.ssDisguiseRating(fac))
			}
		}
	}
}

func TestAlchemySystem(t *testing.T) {
	g := newGame(t, nil)
	clearAround(g)

	if alPropByID["healing"] == nil || alPropByID["nightsight"] == nil {
		t.Fatalf("expected healing and nightsight in alPropByID")
	}
	if alRecipeByID["restorative"] == nil || alRecipeByID["nightglass"] == nil {
		t.Fatalf("expected restorative and nightglass in alRecipeByID")
	}

	// Brewed potion with nightsight grants alnight and extends dark sight
	pot := g.alMakeItem(map[string]int{"healing": 2, "nightsight": 2}, nil, 0, alMethods[0])
	if pot == nil {
		t.Fatalf("expected alMakeItem to build potion")
	}
	g.give(pot)
	if !g.alUseItem(pot) {
		t.Fatalf("expected alUseItem to drink brewed tonic")
	}
	if g.P.St("alnight") <= 0 {
		t.Fatalf("expected drinking night-sight potion to grant alnight status")
	}

	// Verify alnight extends visibility in dark zone
	z := g.Cur
	wasDark := z.Dark
	z.Dark = true
	for i := range z.Lit {
		z.Lit[i] = false
	}
	for _, bp := range g.P.Body {
		if bp.Item != nil && bp.Item.Light > 0 {
			bp.Item = nil
		}
	}
	tx := clampi(g.P.X+8, 1, MW-2)
	for x := mini(g.P.X, tx); x <= maxi(g.P.X, tx); x++ {
		z.Set(x, g.P.Y, TFloor)
	}
	g.recompute()
	if !g.visible(tx, g.P.Y) {
		t.Fatalf("expected tile at distance 8 to be visible in dark with alnight")
	}
	z.Dark = wasDark

	// Brewing from ingredients sharing a tag property
	g.P.Attr[INT] = 26
	g.Skills["lore"] = 5
	z.Set(g.P.X+1, g.P.Y, TWell)
	ing1 := g.makeIngredient("sunroot", "mending", 2, CGreen)
	ing2 := g.makeIngredient("moonleaf", "mending", 2, CGreen)
	g.give(ing1)
	g.give(ing2)
	msg := g.alBrew([]*Item{ing1, ing2})
	if msg == "" || g.al().Made == 0 {
		t.Fatalf("expected alBrew to succeed with two mending ingredients, got msg=%q made=%d", msg, g.al().Made)
	}
}

func TestArcaneSystem(t *testing.T) {
	g := newGame(t, nil)
	clearAround(g)

	pw := g.arSandboxGrant()
	if pw == nil || g.ar().Powers[pw.ID] == nil {
		t.Fatalf("expected arSandboxGrant to grant a researched power")
	}
	abs := g.arAbilities()
	found := false
	for _, ab := range abs {
		if ab.ID == "ar_"+pw.ID {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected arAbilities to include ar_%s", pw.ID)
	}

	// Research design & execution
	srcs := []arSrc{
		{Key: "ember_herb", Label: "ember herb", Kind: "herb", Elem: "fire", Pot: 3},
		{Key: "cinder_ore", Label: "cinder ore", Kind: "ore", Elem: "fire", Pot: 3},
	}
	designed := g.arDesign(srcs, "fire")
	if designed == nil || designed.Elem != "fire" {
		t.Fatalf("expected fire attack power design, got %+v", designed)
	}

	// Pacts at an altar
	g.Cur.Actors = nil
	g.Cur.Name = "Ancient Shrine"
	g.Cur.Set(g.P.X, g.P.Y, TAltar)
	g.P.Attr[WIL] = 26
	g.P.Attr[EGO] = 26
	g.P.Gold = 5000
	pats := g.arPatrons()
	if len(pats) == 0 {
		t.Fatalf("expected arPatrons to return patrons")
	}
	terms := g.arTermsFor(pats[0])
	if len(terms) == 0 {
		t.Fatalf("expected arTermsFor to return terms")
	}
	res := g.arSeal(pats[0], terms[0])
	if len(g.ar().Pacts) == 0 {
		t.Fatalf("expected arSeal to seal pact with high WIL/EGO at altar, got %q", res)
	}
}

func TestNestsSystem(t *testing.T) {
	g := newGame(t, nil)
	clearAround(g)
	g.Cur.Actors = nil

	// Brood nest generation via ntZone
	boss := g.Spawn("bear", 6, NewRng("nt:boss"))
	boss.Name = "Brood of Ashfang"
	boss.X, boss.Y = g.P.X+3, g.P.Y
	boss.Set("brood")
	g.Cur.Actors = []*Actor{boss}
	ntZone(g, g.Cur, NewRng("nt:zone"))
	hasNest := false
	for _, it := range g.Cur.Items {
		if it.Kind == "nest" && it.Extra["ntbrood"] > 0 {
			hasNest = true
			break
		}
	}
	if !hasNest {
		t.Fatalf("expected ntZone with brood boss to place brood nest")
	}

	// ntParents 10-tile range
	var predSp *bsSp
	for _, s := range bsAll {
		if s.Temper == "predator" && ntNestable(s) {
			predSp = s
			break
		}
	}
	if predSp == nil {
		for _, s := range bsAll {
			if s.Temper == "predator" {
				predSp = s
				break
			}
		}
	}
	if predSp != nil {
		par := g.bsSpawn(predSp, 4, NewRng("nt:par"))
		par.HomeX, par.HomeY = clampi(g.P.X+9, 1, MW-2), g.P.Y
		par.X, par.Y = par.HomeX, par.HomeY
		par.Angry = false
		par.AI = "wander"
		par.Set("ntpar")
		par.Set(ntNestFlag(555))
		g.Cur.Actors = append(g.Cur.Actors, par)
		g.ntParents()
		if !par.Angry {
			t.Fatalf("expected nest parent at distance 9 (<= 10) to become angry")
		}
	}

	// Egg hatching & growth
	var nestSp *bsSp
	for _, s := range bsAll {
		if ntNestable(s) && tmTameable(s) && !tmMountOnly(s) {
			nestSp = s
			break
		}
	}
	if nestSp != nil {
		egg := g.ntMakeEgg(nestSp, 888, g.P.X, g.P.Y)
		egg.Extra["ntwarm"] = ntNeedDays(nestSp) * DayLen
		egg.Extra["ntlast"] = g.Turn - 30
		g.give(egg)
		g.ntEggs(g.nt())
		if g.nt().Hatched == 0 || len(g.nt().Grow) == 0 {
			t.Fatalf("expected warmed egg to hatch into a growing companion")
		}
		var kidID int
		for id := range g.nt().Grow {
			kidID = id
		}
		g.Turn += (12 + 4*nestSp.Size) * DayLen
		g.tm().Fed[kidID] = g.Turn
		g.ntGrowTick(g.nt())
		if _, stillGrowing := g.nt().Grow[kidID]; stillGrowing {
			t.Fatalf("expected hatchling %d to reach full growth", kidID)
		}
	}
}

func TestRivalSystem(t *testing.T) {
	g := newGame(t, nil)
	g.rvEnsure()
	if len(g.rv().R) == 0 {
		t.Fatalf("expected rvEnsure to create 1-3 rivals")
	}
	dos := g.rvDossier()
	if len(dos) == 0 {
		t.Fatalf("expected rvDossier to return lines")
	}

	rv := g.rv().R[0]
	rv.Alive = true
	sites := g.buDigSites()
	if len(sites) > 0 {
		rv.Pos = sites[0]
		rv.Dest = sites[0]
		rv.Act = "hoard"
		clears0 := rv.Clears
		g.rvResolveHoard(rv, g.rvToday())
		if rv.Clears <= clears0 {
			t.Fatalf("expected rvResolveHoard to increment rival Clears")
		}
	}

	rv.Alive = true
	rv.Allied = false
	g.rvCull(rv.Pos, 1, 1.0, "slain in test")
	if rv.Alive {
		t.Fatalf("expected rvCull with frac=1.0 to kill rival")
	}
}

func TestStartsSystem(t *testing.T) {
	got := stParse("heir+undead+fallen")
	if len(got) != 2 || got[0] != "heir" || got[1] != "undead" {
		t.Fatalf("expected stParse('heir+undead+fallen') = [heir undead], got %v", got)
	}

	w := world(t, "ember-crown")
	gHeir := NewGame(w, CharSpec{Name: "HeirTest", Species: "human", Calling: "warrior", Attr: [6]int{16, 16, 16, 14, 14, 14}, Scenario: "heir"})
	if gHeir.stGet() == nil || !gHeir.stGet().has("heir") {
		t.Fatalf("expected heir scenario to be applied")
	}

	gFallen := NewGame(w, CharSpec{Name: "FallenTest", Species: "human", Calling: "warrior", Attr: [6]int{16, 16, 16, 14, 14, 14}, Scenario: "fallen"})
	if gFallen.stGet() == nil || len(gFallen.stGet().Locked) == 0 {
		t.Fatalf("expected fallen scenario to have locked powers")
	}

	gUndead := NewGame(w, CharSpec{Name: "UndeadTest", Species: "human", Calling: "warrior", Attr: [6]int{16, 16, 16, 14, 14, 14}, Scenario: "undead"})
	if gUndead.Rep["undead"] < 300 {
		t.Fatalf("expected undead scenario to set undead rep >= 300, got %d", gUndead.Rep["undead"])
	}
}

func TestHallOfFameSystem(t *testing.T) {
	g := newGame(t, nil)
	g.Turn = DayLen * 3
	g.Kills = 5
	g.deed("slay", "Tester slew a troll.", 0, 0)

	if sc := g.hfScore(); sc <= 0 {
		t.Fatalf("expected positive hfScore, got %d", sc)
	}

	raw := g.HFSummary()
	var parsed map[string]any
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		t.Fatalf("HFSummary invalid JSON: %v", err)
	}
	if int(parsed["kills"].(float64)) != 5 {
		t.Fatalf("expected kills=5 in HFSummary, got %v", parsed["kills"])
	}

	leg := HFLegend{
		Name:    "Aldric the Old",
		Species: "human",
		Calling: "warrior",
		Year:    g.W.Year - 20,
		Cause:   "fell to an ancient dragon",
		Place:   g.Cur.Name,
		Level:   12,
		Score:   1800,
		Deeds:   []string{"Aldric slew the red wyrm."},
	}
	g.HFSetLegends([]HFLegend{leg})
	if len(g.hf().Legends) != 1 {
		t.Fatalf("expected 1 legend in hfState")
	}

	g.hfPlaceOld(g.Cur, &g.hf().Legends[0])
	foundOld := false
	for _, a := range g.Cur.Actors {
		if a.Has("hf_legend") && a.Name == "Aldric the Old" {
			foundOld = true
			break
		}
	}
	if !foundOld {
		t.Fatalf("expected hfPlaceOld to place Aldric the Old in town")
	}

	g.hfPlaceGrave(g.Cur, &g.hf().Legends[0])
	foundGrave := false
	for _, note := range g.Cur.Notes {
		if strings.Contains(note, "Aldric the Old") {
			foundGrave = true
			break
		}
	}
	if !foundGrave {
		t.Fatalf("expected hfPlaceGrave to add grave inscription note")
	}
}

func TestPoliticsNemesisClutterGamesTamingBeastcraft(t *testing.T) {
	g := newGame(t, nil)
	clearAround(g)

	// 1. Politics
	realm := g.HomeState
	if realm > 0 {
		court := g.poCourt(realm)
		if court[0].Name == "" {
			t.Fatalf("expected poCourt ruler to have a name")
		}
		facs := g.poFacs(realm)
		if len(facs) < 2 {
			t.Fatalf("expected at least 2 political factions, got %d", len(facs))
		}
		g.poStart(realm, "succession", g.poDay())
		cr := g.poCrisisAt(realm)
		if cr == nil {
			t.Fatalf("expected active crisis after poStart")
		}
		g.poResolve(realm, cr)
		if !cr.Done || cr.Winner == "" {
			t.Fatalf("expected poResolve to finish crisis with a winner")
		}
	}

	// 2. Nemesis
	nx := nxEnsure(g)
	if nx.Nem == nil {
		nx.Nem = g.nxGenerate()
	}
	if nx.Nem == nil {
		t.Fatalf("expected nxGenerate to produce a nemesis")
	}
	nemActor := nxBuild(g, nx.Nem, NewRng("nx:build"))
	if nemActor == nil || nemActor.Name == "" {
		t.Fatalf("expected nxBuild to build nemesis actor")
	}

	// 3. Clutter
	if len(clSets) == 0 || len(clDefs) == 0 {
		t.Fatalf("expected clutter sets and defs to be parsed")
	}
	set0 := clSets[0]
	for _, di := range set0.Items {
		g.give(g.clMake(clDefs[di]))
	}
	clTurn(g)
	if clDoneCount(g) < 1 || clCarry(g) < 1 {
		t.Fatalf("expected completing a clutter set to grant collection completion and +1 carry capacity")
	}

	// 4. Games & entertainment
	bouts := g.gmBouts()
	win, logLines := gmSimBout(bouts[0], NewRng("gm:bout"), true)
	if (win != 0 && win != 1) || len(logLines) == 0 {
		t.Fatalf("expected gmSimBout to return winner 0/1 and narration lines")
	}
	inst := g.gmMakeInstrument([4]string{"lute", "flute", "drum", "harp"}, 0, 2)
	g.give(inst)
	if g.gmInstrument() == nil {
		t.Fatalf("expected gmInstrument to find lute in pack")
	}

	// 5. Taming & mounts
	var tameSp *bsSp
	for _, s := range bsAll {
		if tmTameable(s) && !tmMountOnly(s) {
			tameSp = s
			break
		}
	}
	if tameSp != nil {
		beast := g.bsSpawn(tameSp, 3, NewRng("tm:beast"))
		beast.X, beast.Y = g.P.X+1, g.P.Y
		g.Cur.Actors = append(g.Cur.Actors, beast)
		g.tmJoin(beast, tameSp, 80, g.Cur.Key)
		if beast.AI != "follower" || !beast.Has("tm") {
			t.Fatalf("expected tmJoin to make beast a tamed follower")
		}
		g.tmSetMount(tameSp, "Swiftfang", 3, 85, 0)
		if g.tmRideRec() == nil {
			t.Fatalf("expected tmSetMount to register active mount")
		}
		g.tmUnsaddle()
	}

	// 6. Beastcraft
	var craftSp *bsSp
	for _, s := range bsAll {
		if s.Size >= 4 {
			craftSp = s
			break
		}
	}
	if craftSp != nil {
		for _, matName := range []string{"hide", "sinew", "bone", "horn", "pelt"} {
			it := &Item{
				ID:     g.newID(),
				Kind:   "trophy",
				Name:   strings.ToLower(craftSp.Display()) + " " + matName,
				Glyph:  '~',
				F:      CGray,
				Value:  10,
				Weight: 1,
				Effect: "bsp:" + craftSp.ID,
			}
			bcStamp(it)
			g.give(it)
		}
		rc := bcRecipes[0]
		if !g.bcCanMake(rc) {
			t.Fatalf("expected bcCanMake(%s) to be true with beast materials in pack", rc.Name)
		}
		var picks []*Item
		for _, sl := range rc.Slots {
			cands := g.bcCandidates(sl, picks)
			picks = append(picks, cands[0])
		}
		crafted := g.bcBuild(rc, picks, 2)
		if crafted == nil || crafted.Name == "" {
			t.Fatalf("expected bcBuild to produce crafted item")
		}
	}
}
