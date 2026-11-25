package engine

import (
	"fmt"
	"strings"
	"testing"
)

func TestWave6FaithPlay(t *testing.T) {
	g := newGame(t, nil)
	if len(g.W.Religions) == 0 {
		t.Skip("world has no religions")
	}
	var rel1, rel2 *ReligionData
	for i := range g.W.Religions {
		r := &g.W.Religions[i]
		if r.ID > 0 && r.Name != "" {
			if rel1 == nil {
				rel1 = r
			} else if rel2 == nil {
				rel2 = r
				break
			}
		}
	}
	if rel1 == nil {
		t.Fatal("no valid religion found")
	}

	// Per-faith blessing profile
	bl := fpBlessingFor(g.W, rel1)
	if bl.StatusID == "" || bl.Title == "" || bl.Summary == "" || bl.Quote == "" {
		t.Fatalf("incomplete blessing profile: %+v", bl)
	}

	// Named holy sites deterministic
	sites := g.fpHolySites()
	if len(sites) == 0 {
		t.Fatal("expected named holy sites to be generated")
	}
	if sites2 := g.fpGenSites(); len(sites2) != len(sites) || sites2[0].ID != sites[0].ID {
		t.Fatal("fpGenSites is not deterministic")
	}

	// Coin offering grants blessing & rep, blocks second offering same day
	g.P.Gold = 500
	repKey := fmt.Sprintf("faith:%d", rel1.ID)
	rep0 := g.Rep[repKey]
	ok, msg := g.fpOfferCoin(rel1, 15)
	if !ok {
		t.Fatalf("fpOfferCoin failed: %s", msg)
	}
	if g.P.St(bl.StatusID) <= 0 {
		t.Fatalf("expected blessing status %s to be active", bl.StatusID)
	}
	if g.Rep[repKey] <= rep0 {
		t.Fatalf("expected faith rep to rise: %d -> %d", rep0, g.Rep[repKey])
	}
	if ok2, _ := g.fpOfferCoin(rel1, 15); ok2 {
		t.Fatal("expected second offering on same day to be rejected")
	}

	// Item offering on next day
	g.Turn += DayLen
	food := g.MakeFood(g.R, 1)
	food.Count = 1
	g.give(food)
	ok, msg = g.fpOfferItem(rel1, food)
	if !ok {
		t.Fatalf("fpOfferItem failed: %s", msg)
	}
	if g.P.St(bl.StatusID) <= 0 {
		t.Fatalf("expected blessing after item offering")
	}

	// Pilgrimage vow & completion on entering holy site
	site := sites[0]
	g.fpVowPilgrimage(site)
	if g.fp().ActiveVow != site.ID || !g.fp().KnownSites[site.ID] {
		t.Fatalf("pilgrimage vow not recorded: %+v", g.fp())
	}
	zn := g.GetZone(site.WX, site.WY, 1, 1, 0)
	fpEnter(g, zn)
	if !g.fp().Visited[site.ID] || g.fp().ActiveVow != "" {
		t.Fatalf("pilgrimage not completed on arrival: visited=%v active=%q", g.fp().Visited[site.ID], g.fp().ActiveVow)
	}

	// Conversion
	targetRel := rel1
	if g.HomeFaith == rel1.ID && rel2 != nil {
		targetRel = rel2
	}
	g.fp().Heretic[targetRel.ID] = true
	g.fpConvert(targetRel)
	if g.HomeFaith != targetRel.ID {
		t.Fatalf("conversion failed: HomeFaith=%d want %d", g.HomeFaith, targetRel.ID)
	}
	if g.fp().Heretic[targetRel.ID] {
		t.Fatal("expected conversion to clear heretic mark for new faith")
	}

	// Heresy suspicion & trial resolution
	trialRel := rel1
	if trialRel.ID == g.HomeFaith && rel2 != nil {
		trialRel = rel2
	}
	g.fp().Heretic[trialRel.ID] = true
	if !g.fpUnderSuspicion(nil, trialRel) {
		t.Fatal("expected branded heretic to be under suspicion")
	}
	priest := g.Spawn("villager", 3, g.R)
	priest.Role = "priest"
	priest.Name = "Father Test"
	g.Talk = priest
	g.fpBeginTrial(priest, trialRel)
	if len(g.Opts) < 4 {
		t.Fatalf("expected 4 heresy trial options, got %d", len(g.Opts))
	}
	g.P.Gold = 500
	g.fpResolveTrial(priest, trialRel, 0, "recant", 40)
	if g.fp().Heretic[trialRel.ID] {
		t.Fatal("expected recanting to clear heretic status")
	}
	if len(g.fp().Trials) == 0 || g.fp().Trials[len(g.fp().Trials)-1].Outcome != "recanted" {
		t.Fatalf("expected recorded recanted trial, got %+v", g.fp().Trials)
	}
}

func TestWave6LoreLibrary(t *testing.T) {
	g := newGame(t, nil)
	if len(g.W.Events) == 0 {
		t.Skip("world has no history events")
	}

	// Event scroll built from real WorldData.Events
	scroll := g.llMakeScroll(NewRng("w6:ll"), 0)
	if scroll == nil || scroll.Extra["ll"] != 1 || scroll.Extra["ll:ev"] != 1 {
		t.Fatalf("invalid event scroll: %+v", scroll)
	}
	if !strings.Contains(scroll.Text, g.W.Events[0].Text) {
		t.Fatalf("event scroll does not contain real event text")
	}

	// Sync into collection log & mark read
	g.give(scroll)
	g.llSync()
	key := "b:" + scroll.Name
	en := g.ll().Entries[key]
	if en == nil || en.Category != "Scroll" {
		t.Fatalf("scroll not synced into lore library: %+v", en)
	}
	xp0 := g.XP
	g.llMarkRead(en)
	if !en.Read || !g.ll().EventsRead[0] {
		t.Fatalf("scroll not marked read / event not attested: read=%v ev=%v", en.Read, g.ll().EventsRead[0])
	}
	if g.XP <= xp0 {
		t.Fatal("reading lore scroll did not grant XP")
	}

	// Milestone at 5 works read
	for i := 1; i <= 5; i++ {
		bk := g.MakeBook(fmt.Sprintf("Chronicle Vol %d", i), fmt.Sprintf("Body of volume %d.", i))
		g.give(bk)
	}
	g.llSync()
	for _, e := range g.ll().Entries {
		g.llMarkRead(e)
	}
	if !g.ll().Milestones[5] {
		t.Fatalf("expected 5-work library milestone to trigger, milestones=%+v", g.ll().Milestones)
	}

	// Reader screen open and navigation
	g.llOpen()
	if g.Mode != "ll" {
		t.Fatalf("expected mode ll, got %q", g.Mode)
	}
	llKeys(g, "Enter")
	if g.ll().reading == "" {
		t.Fatal("expected Enter to open selected entry for reading")
	}
	llKeys(g, "Escape")
	llKeys(g, "Escape")
	if g.Mode != "main" {
		t.Fatalf("expected Escape to return to main, got %q", g.Mode)
	}

	// Archive talk on scholar/bookseller
	sch := g.Spawn("villager", 2, g.R)
	sch.Role = "scholar"
	sch.Name = "Archivist Test"
	opts := llTalk(g, sch)
	if len(opts) < 2 {
		t.Fatalf("expected scholar archive talk options, got %d", len(opts))
	}
	g.Talk = sch
	g.P.Gold = 500
	invBefore := len(g.P.Inv)
	g.llOfferScroll(sch)
	if len(g.Opts) > 0 && strings.HasPrefix(g.Opts[0].Text, "Buy it") {
		g.Opts[0].Do()
		if len(g.P.Inv) <= invBefore {
			t.Fatal("buying scroll from scholar did not add item to pack")
		}
	}
}

func TestWave6AncientLayers(t *testing.T) {
	g := newGame(t, nil)
	sites := g.al2Sites()
	if len(sites) == 0 {
		t.Fatal("expected pre-collapse ancient layer sites to be generated")
	}
	site := sites[0]
	if site.EvText == "" || site.FragNames[0] == "" || site.FragNames[1] == "" || site.FragNames[2] == "" {
		t.Fatalf("site missing event text or 3 fragments: %+v", site)
	}

	// Scholar talk reveals an unknown site
	sch := g.Spawn("villager", 2, g.R)
	sch.Role = "scholar"
	sch.Name = "Scholar Test"
	g.Talk = sch
	opts := al2Talk(g, sch)
	if len(opts) != 1 {
		t.Fatalf("expected 1 ancient layers talk option, got %d", len(opts))
	}
	opts[0].Do()
	if !g.al2Prog(site).Known {
		t.Fatal("scholar talk did not mark ancient site as known")
	}

	// Zone generation places 3 fragments and stele; entering discovers site
	zn := g.GetZone(site.WX, site.WY, site.ZX, site.ZY, site.Depth)
	al2Enter(g, zn)
	if !g.al2Prog(site).Discovered {
		t.Fatal("al2Enter did not mark site discovered")
	}
	var frags []*Item
	for _, it := range zn.Items {
		if it != nil && it.Extra != nil && it.Extra["al2"] == site.ID+1 {
			frags = append(frags, it)
		}
	}
	if len(frags) != al2FragsPerSite {
		t.Fatalf("expected %d fragments placed in site zone, got %d", al2FragsPerSite, len(frags))
	}

	// Collecting all 3 fragments reconstructs the real WorldData.Events record
	secretsBefore := len(g.Secrets)
	for _, it := range frags {
		g.give(it)
	}
	al2Turn(g)
	pr := g.al2Prog(site)
	if pr.count() != 3 || !pr.Revealed {
		t.Fatalf("expected 3/3 fragments and Revealed=true, got count=%d revealed=%v", pr.count(), pr.Revealed)
	}
	if len(g.Secrets) <= secretsBefore {
		t.Fatal("reconstructing ancient record did not add a secret")
	}

	// Read-only link to legendlore.go must not mutate lb().P
	lbBefore := len(g.lb().P)
	for _, s := range sites {
		if s.LinkedArm != "" {
			note := g.al2LegendNote(s)
			if !strings.Contains(note, "Linked legend") {
				t.Fatalf("unexpected legend note: %q", note)
			}
			break
		}
	}
	if len(g.lb().P) != lbBefore {
		t.Fatalf("al2LegendNote mutated legendlore state: %d -> %d", lbBefore, len(g.lb().P))
	}
}

func TestWave6LegacyPlay(t *testing.T) {
	g := newGame(t, nil)
	g.P.Name = "Alden Vance"
	g.P.Level = 10
	g.P.Gold = 400
	g.deed("slay", "Alden Vance slew the Ash Drake of the North.", 0, 0)

	// Set positive and negative reputations to verify 50% inheritance
	stKey := fmt.Sprintf("state:%d", g.HomeState)
	g.Rep[stKey] = 80
	faKey := ""
	if g.HomeFaith > 0 {
		faKey = fmt.Sprintf("faith:%d", g.HomeFaith)
		g.Rep[faKey] = -60
	}

	// Give a distinctive weapon to inherit
	blade := g.MakeWeapon("longblade", 3, g.R)
	blade.Name = "Vance Ancestral Blade"
	blade.Value = 500
	g.give(blade)
	g.equip(blade)

	// Add a grown descendant in family state
	fm := g.fm()
	heir := &fmChild{
		Name:    "Mira Vance",
		Species: g.P.Species,
		Calling: "warrior",
		Born:    g.Day() - 4*365,
		Bond:    10,
		Attr:    [6]int{16, 15, 16, 14, 14, 14},
	}
	fm.Kids = append(fm.Kids, heir)

	if !g.lpHasHeir() || len(g.lpCandidates()) != 1 {
		t.Fatalf("expected 1 eligible heir, got %d", len(g.lpCandidates()))
	}

	// Simulate death and death hook
	g.Dead = true
	g.DeathText = "fell defending the gate"
	g.lpOnDeath()
	if !g.lp().Pending {
		t.Fatal("lpOnDeath did not flag pending succession")
	}

	// Continue as heir
	if !g.lpContinueAsHeir(0) {
		t.Fatal("lpContinueAsHeir returned false")
	}
	if g.Dead || g.P.Name != "Mira Vance" || g.lp().Gen != 2 {
		t.Fatalf("unexpected post-succession state: dead=%v name=%q gen=%d", g.Dead, g.P.Name, g.lp().Gen)
	}
	if g.Rep[stKey] != 40 {
		t.Fatalf("expected state rep halved 80 -> 40, got %d", g.Rep[stKey])
	}
	if faKey != "" && g.Rep[faKey] != -30 {
		t.Fatalf("expected faith rep halved -60 -> -30, got %d", g.Rep[faKey])
	}

	// Verify inherited heirloom provenance
	foundHeirloom := false
	for _, bp := range g.P.Body {
		if bp != nil && bp.Item != nil && bp.Item.Extra != nil && bp.Item.Extra["lp_gen"] == 1 && strings.Contains(bp.Item.Lore, "Alden Vance") {
			foundHeirloom = true
		}
	}
	for _, it := range g.P.Inv {
		if it != nil && it.Extra != nil && it.Extra["lp_gen"] == 1 && strings.Contains(it.Lore, "Alden Vance") {
			foundHeirloom = true
		}
	}
	if !foundHeirloom {
		t.Fatal("inherited heirloom missing lp_gen or forebear provenance")
	}

	// Verify gravestones placed in zone notes
	if len(g.lpl().Graves) == 0 {
		t.Fatal("expected gravestone/memorial to be recorded")
	}
	hasStone := false
	for _, note := range g.Cur.Notes {
		if strings.Contains(note, "gravestone of Alden Vance") {
			hasStone = true
			break
		}
	}
	if !hasStone {
		t.Fatal("expected forebear gravestone inscription in zone notes")
	}

	// Verify NPC legend talk
	npc := g.Spawn("villager", 1, g.R)
	npc.Name = "Old Tom"
	npc.AI = "town"
	npc.Faction = stateFaction(g.HomeState)
	g.Talk = npc
	opts := lpTalk(g, npc)
	if len(opts) == 0 || !strings.Contains(opts[0].Text, "Alden Vance") {
		t.Fatalf("expected NPC legend option for Alden Vance, got %+v", opts)
	}
	xp0 := g.XP
	opts[0].Do()
	if g.XP <= xp0 {
		t.Fatal("hearing forebear legend did not grant XP")
	}
}

func TestWave6FinalThreatAndNGPlus(t *testing.T) {
	g := newGame(t, nil)
	st := g.ftEnsure()
	if !st.Inited || st.Name == "" || st.BossName == "" || st.Stage != 0 {
		t.Fatalf("ftEnsure did not initialize threat: %+v", st)
	}

	// Stage 0 -> 1: Awaken & advance doom clock
	g.ftAwaken()
	if st.Stage != 1 || st.Doom < 8 {
		t.Fatalf("ftAwaken failed: stage=%d doom=%d", st.Stage, st.Doom)
	}
	doom0 := st.Doom
	g.Turn = (st.LastDay + 4) * DayLen
	g.Turn -= g.Turn % 60
	ftTurn(g)
	if st.Doom <= doom0 {
		t.Fatalf("expected doom clock to advance: %d -> %d", doom0, st.Doom)
	}

	// Stage 1 -> 2: Speak with scholar about omens
	sch := g.Spawn("villager", 2, g.R)
	sch.Role = "scholar"
	sch.Name = "Seer Test"
	g.Talk = sch
	opts := ftTalk(g, sch)
	if len(opts) == 0 {
		t.Fatal("expected omen investigation talk option in stage 1")
	}
	opts[0].Do()
	if st.Stage != 2 {
		t.Fatalf("expected stage 2 after investigating omens, got %d", st.Stage)
	}

	// Stage 2 -> 3: Gather all 3 wards (Lore, Power, Unity)
	g.Skills["lore"] = 2
	loreBurg := g.W.burgByID[st.LoreBurg]
	if loreBurg == nil {
		t.Fatal("missing LoreBurg")
	}
	g.Warp(loreBurg.P%g.W.W, loreBurg.P/g.W.W, 1, 1, 0)
	opts = ftTalk(g, sch)
	gotLore := false
	for _, o := range opts {
		if strings.Contains(o.Text, "Ward of Lore") {
			o.Do()
			gotLore = true
			break
		}
	}
	if !gotLore || !st.WardLore {
		t.Fatal("failed to secure Ward of Lore")
	}

	// Ward of Power: place harbinger and kill it
	pwrZn := g.GetZone(st.PowerP%g.W.W, st.PowerP/g.W.W, 1, 1, 0)
	g.ftPlaceHarbinger(pwrZn)
	var lt *Actor
	for _, a := range pwrZn.Actors {
		if a != nil && a.Has("ft_lt") {
			lt = a
			break
		}
	}
	if lt == nil {
		t.Fatal("ftPlaceHarbinger did not spawn lieutenant")
	}
	ftKill(g, lt, g.P)
	if !st.WardPower {
		t.Fatal("slaying lieutenant did not grant Ward of Power")
	}

	// Ward of Unity: talk to elder in UnitySt with high rep
	var unityBurg *BurgData
	for i := range g.W.Burgs {
		b := &g.W.Burgs[i]
		if b.Alive && b.State == st.UnitySt && (b.Layer == "" || b.Layer == "surface") {
			unityBurg = b
			break
		}
	}
	if unityBurg == nil {
		t.Fatal("no living burg found in UnitySt")
	}
	g.Rep[stateFaction(st.UnitySt)] = 100
	g.Warp(unityBurg.P%g.W.W, unityBurg.P/g.W.W, 1, 1, 0)
	elder := g.Spawn("villager", 2, g.R)
	elder.Role = "elder"
	g.Talk = elder
	opts = ftTalk(g, elder)
	gotUnity := false
	for _, o := range opts {
		if strings.Contains(o.Text, "Ward of Unity") {
			o.Do()
			gotUnity = true
			break
		}
	}
	if !gotUnity || !st.WardUnity || st.Stage != 3 {
		t.Fatalf("failed to secure Ward of Unity / stage 3: unity=%v stage=%d", st.WardUnity, st.Stage)
	}

	// Stage 3 -> 4: Forge counter-relic
	if !g.ftForgeRelic() || !st.Forged || st.Stage != 4 || !g.ftHasRelic() {
		t.Fatalf("ftForgeRelic failed: forged=%v stage=%d hasRelic=%v", st.Forged, st.Stage, g.ftHasRelic())
	}

	// Stage 4 -> 5: Sanctum boss defeat unlocks New Game+
	sanc := g.ftSanctumZone()
	var boss *Actor
	for _, a := range sanc.Actors {
		if a != nil && a.Has("ftboss") {
			boss = a
			break
		}
	}
	if boss == nil {
		t.Fatal("Sanctum missing World-Ender boss")
	}
	ftKill(g, boss, g.P)
	if !st.Defeated || st.Stage != 5 || st.Doom != 0 || !g.ftNGUnlocked() {
		t.Fatalf("defeating boss did not complete threat / unlock NG+: defeated=%v stage=%d doom=%d unlocked=%v", st.Defeated, st.Stage, st.Doom, g.ftNGUnlocked())
	}

	// New Game+ cycle transition & scaling
	if !g.ftStartNewGamePlus() {
		t.Fatal("ftStartNewGamePlus returned false")
	}
	if g.ftg().Cycle != 1 || len(g.ftg().Victories) != 1 {
		t.Fatalf("NG+ cycle not incremented: cycle=%d victories=%d", g.ftg().Cycle, len(g.ftg().Victories))
	}
	newSt := g.ft()
	if newSt.Stage != 1 || !strings.Contains(newSt.Name, "Cycle 2") {
		t.Fatalf("expected fresh Cycle 2 threat at stage 1, got stage=%d name=%q", newSt.Stage, newSt.Name)
	}
	carried := 0
	for _, it := range g.P.Inv {
		if it != nil && it.Extra != nil && it.Extra["ft_ng"] == 1 {
			carried++
		}
	}
	for _, bp := range g.P.Body {
		if bp != nil && bp.Item != nil && bp.Item.Extra != nil && bp.Item.Extra["ft_ng"] == 1 {
			carried++
		}
	}
	if carried == 0 || carried > 3 {
		t.Fatalf("expected 1..3 carried legacy items in NG+, got %d", carried)
	}
	// Verify NG+ zone scaling on a hostile actor
	testZn := newZone("test:ng", g.WX, g.WY, 0, 0, 1)
	foe := g.Spawn("wolf", 4, NewRng("w6:ngfoe"))
	foe.AI = "hostile"
	hp0, lvl0 := foe.MaxHP, foe.Level
	testZn.Actors = []*Actor{foe}
	ftZone(g, testZn, nil)
	if !foe.Has("ft_ng_scaled") || foe.MaxHP <= hp0 || foe.Level != lvl0+2 {
		t.Fatalf("NG+ enemy scaling failed: scaled=%v hp %d->%d lvl %d->%d", foe.Has("ft_ng_scaled"), hp0, foe.MaxHP, lvl0, foe.Level)
	}
}

func TestWave6Task3Cleanup(t *testing.T) {
	g := newGame(t, nil)

	// 1. All 6 rumour sinks wired to rmSink
	sinks := []struct {
		name string
		fn   func(*Game, string, string, string)
		kind string
	}{
		{"saRumourSink", saRumourSink, "social"},
		{"fqRumourSink", fqRumourSink, "faction"},
		{"dzRumourSink", dzRumourSink, "affliction"},
		{"mgRumourSink", mgRumourSink, "ecology"},
		{"sqRumourSink", sqRumourSink, "siege"},
		{"htRumourSink", htRumourSink, "heist"},
	}
	for _, s := range sinks {
		if s.fn == nil {
			t.Fatalf("%s is nil", s.name)
		}
		before := len(g.rm().Rums)
		s.fn(g, s.kind, fmt.Sprintf("burg:%d", g.StartBurg), "Wave6 test rumour for "+s.name)
		if len(g.rm().Rums) <= before {
			t.Fatalf("%s did not record rumour into rumourmill", s.name)
		}
	}

	// 2. dzCheckMod wired into checkMod
	baseMod := g.checkMod("persuasion")
	g.dz().Conds = append(g.dz().Conds, &dzCond{ID: "rot_blight", Stage: 2})
	dzPen := g.dzCheckMod("persuasion")
	if dzPen != -5 {
		t.Fatalf("expected dzCheckMod(persuasion)=-5 for rot_blight stage 2, got %d", dzPen)
	}
	if got := g.checkMod("persuasion"); got != baseMod+dzPen {
		t.Fatalf("checkMod(persuasion)=%d want %d (base %d + dz %d)", got, baseMod+dzPen, baseMod, dzPen)
	}
	g.dz().Conds = nil

	// 3. Poached goods (hnp) illegal in uwIllegal
	poached := &Item{ID: g.newID(), Name: "poached venison", Kind: "food", Value: 20, Extra: map[string]int{"hnp": 1}}
	if !g.uwIllegal(poached, g.HomeState) {
		t.Fatal("expected poached item (Extra[hnp]=1) to be flagged by uwIllegal")
	}

	// 4. rnGuardChanging in heist guard shifts
	// Set turn so (Turn + startMinutes) % DayLen == 1265 (21:05, evening shift change)
	g.Turn = (1265 - startMinutes + DayLen) % DayLen
	if !g.rnGuardChanging() {
		t.Fatalf("expected rnGuardChanging() true at 21:05 (Turn=%d)", g.Turn)
	}
	// Set turn to noon (720)
	g.Turn = (720 - startMinutes + DayLen) % DayLen
	if g.rnGuardChanging() {
		t.Fatalf("expected rnGuardChanging() false at 12:00 (Turn=%d)", g.Turn)
	}
	// Verify htMoveGuards skips guard 0 during shift change and moves it outside shift change
	gdActor := g.Spawn("guard", 3, g.R)
	gdActor.AI = "still"
	gdActor.X, gdActor.Y = 20, 10
	g.Cur.Set(20, 10, TFloor)
	g.Cur.Set(21, 10, TFloor)
	g.Cur.Set(22, 10, TFloor)
	g.Cur.Actors = append(g.Cur.Actors, gdActor)
	job := &htJob{
		Guards: []htGuard{{ID: gdActor.ID, Route: [][2]int{{22, 10}, {20, 10}}}},
	}
	g.Turn = (1265 - startMinutes + DayLen) % DayLen
	g.htMoveGuards(job)
	if gdActor.X != 20 || gdActor.Y != 10 {
		t.Fatalf("guard 0 moved during shift change: (%d,%d)", gdActor.X, gdActor.Y)
	}
	g.Turn = (720 - startMinutes + DayLen) % DayLen
	g.htMoveGuards(job)
	if gdActor.X == 20 && gdActor.Y == 10 {
		t.Fatal("guard 0 did not patrol outside shift change")
	}

	// 5. Hunting festival contest registered via fvRegisterAct
	act := fvActByID("hunt")
	if act == nil {
		t.Fatal("hunting festival contest not registered via fvRegisterAct")
	}
	inHarvest := false
	for _, id := range fvKindActs["harvest"] {
		if id == "hunt" {
			inHarvest = true
			break
		}
	}
	if !inHarvest {
		t.Fatal("hunting festival contest missing from harvest feast activities")
	}
	play := fvExtPlays["hunt"]
	if play == nil {
		t.Fatal("hunting festival play handler not registered")
	}
	bd := g.W.burgByID[g.StartBurg]
	for ch := 0; ch < 3; ch++ {
		_, _, pre, ok := play(g, bd, &FeastData{Name: "Harvest Feast"}, ch)
		if !ok || pre == "" {
			t.Fatalf("hunting festival choice %d returned ok=%v pre=%q", ch, ok, pre)
		}
	}
}
