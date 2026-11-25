package engine

import (
	"strings"
	"testing"
)

func TestDisguiseAndAliasCover(t *testing.T) {
	g := newGame(t, nil)
	g.P.Gold = 500

	for _, id := range []string{"watch_tabard", "priest_robes", "merchant_papers", "guild_hood"} {
		if ok, msg := g.dgAcquireDisguise(id); !ok {
			t.Fatalf("dgAcquireDisguise(%s) failed: %s", id, msg)
		}
	}

	ok, _ := g.dgDonDisguise("watch_tabard", "Sergeant Vane")
	if !ok || !g.dgHasCover() {
		t.Fatal("expected active disguise cover after dgDonDisguise")
	}
	st := dgSt(g)
	if st.Alias.Name != "Sergeant Vane" || st.Alias.Cover != 100 {
		t.Fatalf("unexpected alias state: %+v", st.Alias)
	}

	// Place an angry guard nearby and a local bounty, then slip past.
	if bd := g.localBurg(); bd != nil {
		g.uwSt().Bounty[bd.State] = 200
	}
	guard := g.Spawn("guard", 2, g.R)
	guard.Role = "guard"
	guard.Angry = true
	guard.X = clampi(g.P.X+2, 1, MW-2)
	guard.Y = g.P.Y
	g.Cur.Actors = append(g.Cur.Actors, guard)

	if !g.dgSlipPastGuards() {
		t.Fatal("expected disguise to let fugitive slip past hostile guard")
	}
	if guard.Angry {
		t.Fatal("expected guard aggro to be calmed while disguised")
	}
	if st.Alias.BountiesEvaded == 0 {
		t.Fatal("expected BountiesEvaded to increment")
	}

	// Degrade cover until blown.
	blown := g.dgDegradeCover(120, "caught picking lock")
	if !blown || g.dgHasCover() {
		t.Fatal("expected cover to be blown after heavy degradation")
	}
}

func TestBlackmailLeverageAndScandals(t *testing.T) {
	g := newGame(t, nil)

	// Seed a knowledge.go secret fact and a g.Secrets entry.
	kn := knSt(g)
	f := kn.add(knFact{Kind: "secret", Text: "The magistrate skimmed harbor tolls.", True: true, Day: g.Day()})
	kn.Learned = append(kn.Learned, knLearn{Fact: f.ID, Name: "Dockhand", Day: g.Day()})
	g.Secrets = append(g.Secrets, Secret{Text: "The factor forged salt permits.", Kind: "secret", Year: g.W.Year})

	notable := g.Spawn("villager", 2, g.R)
	notable.Name = "Factor Orin"
	notable.Proper = true
	notable.Role = "merchant"
	notable.State = g.HomeState

	lev1, ok := g.dgGatherLeverage(notable)
	if !ok || lev1 == nil || lev1.Source != "kn" {
		t.Fatalf("expected leverage compiled from kn secret, got %+v ok=%v", lev1, ok)
	}

	goldBefore := g.P.Gold
	payout, ok := g.dgExtort(lev1.ID)
	if !ok || payout <= 0 || g.P.Gold != goldBefore+payout {
		t.Fatalf("dgExtort failed: ok=%v payout=%d", ok, payout)
	}
	if dgSt(g).Discounts[lev1.BurgID] <= 0 {
		t.Fatal("expected local merchant discount after extortion")
	}

	// Compile second secret from g.Secrets (after clearing remaining kn facts) and test witness silencing + scandal leak.
	knSt(g).Facts = nil
	lev2, ok := g.dgGatherLeverage(notable)
	if !ok || lev2 == nil || lev2.Source != "secret" {
		t.Fatalf("expected second leverage from g.Secrets, got %+v ok=%v", lev2, ok)
	}
	g.uwSt().Bounty[lev2.StateID] = 300
	if ok, _ := g.dgSilenceWitness(lev2.ID); !ok {
		t.Fatal("dgSilenceWitness failed")
	}
	if g.uwSt().Bounty[lev2.StateID] >= 300 {
		t.Fatalf("expected bounty reduction, still %d", g.uwSt().Bounty[lev2.StateID])
	}

	// Leak scandal to town crier.
	g.Secrets = append(g.Secrets, Secret{Text: "Hidden contraband in the cellar.", Kind: "secret", Year: g.W.Year})
	lev3, ok := g.dgGatherLeverage(notable)
	if !ok {
		t.Fatal("expected third leverage")
	}
	if ok, msg := g.dgLeakScandal(lev3.ID); !ok || !strings.Contains(msg, notable.Name) {
		t.Fatalf("dgLeakScandal failed: ok=%v msg=%q", ok, msg)
	}
	if dgSt(g).Scandals[notable.Name] != 1 || len(dgSt(g).CrierLog) == 0 {
		t.Fatal("expected scandal count and town crier entry")
	}
}

func TestApprenticeLifecycle(t *testing.T) {
	g := newGame(t, nil)

	vill := g.Spawn("villager", 1, g.R)
	vill.Name = "Young Pip"
	vill.Proper = true
	vill.X = clampi(g.P.X+1, 1, MW-2)
	vill.Y = g.P.Y
	g.Cur.Actors = append(g.Cur.Actors, vill)

	app, ok := g.dgTakeApprentice(vill)
	if !ok || app == nil {
		t.Fatal("dgTakeApprentice failed")
	}
	if vill.AI != "follower" || vill.Faction != "player" {
		t.Fatalf("expected apprentice to become player follower, got AI=%s Faction=%s", vill.AI, vill.Faction)
	}

	if ok, _ := g.dgTrainApprentice("blades"); !ok {
		t.Fatal("first dgTrainApprentice failed")
	}
	if ok, _ := g.dgTrainApprentice("stealth"); !ok {
		t.Fatal("second dgTrainApprentice failed")
	}

	// Trigger killHooks XP gain.
	foe := g.Spawn("bandit", 3, g.R)
	dgKill(g, foe, g.P)
	if app.Level < 2 {
		t.Fatalf("expected apprentice to level up after training + kill, got Lv %d (XP %d)", app.Level, app.XP)
	}

	grad, ok := g.dgGraduateApprentice()
	if !ok || grad == nil || !grad.Graduated {
		t.Fatal("dgGraduateApprentice failed")
	}
	if dgSt(g).Apprentice != nil || len(dgSt(g).Graduates) != 1 {
		t.Fatal("expected apprentice moved to Graduates slice")
	}
	if vill.AI != "wander" {
		t.Fatalf("expected graduated apprentice to wander independently, got AI=%s", vill.AI)
	}
}

func TestConstructAssemblyAndDeployment(t *testing.T) {
	g := newGame(t, nil)

	combos := [][4]string{
		{"brass", "treads", "sawblade", "steam"},
		{"iron", "heavy_legs", "maul", "ley_crystal"},
		{"bone", "wings", "bolt_thrower", "void_cell"},
		{"flesh", "spider_legs", "claws", "bio_heart"},
	}
	var first *gcConstruct
	for i, c := range combos {
		con, ok := g.gcAssembleConstruct("", c[0], c[1], c[2], c[3])
		if !ok || con == nil {
			t.Fatalf("gcAssembleConstruct combo %d failed", i)
		}
		if con.MaxHP <= 0 || con.Damage <= 0 || con.Speed <= 0 {
			t.Fatalf("invalid construct stats: %+v", con)
		}
		if first == nil {
			first = con
		}
	}

	act, ok := g.gcDeployConstruct(first.ID)
	if !ok || act == nil || act.AI != "follower" || act.Faction != "player" {
		t.Fatalf("gcDeployConstruct failed: ok=%v act=%+v", ok, act)
	}
	if !first.Deployed || first.ActorID != act.ID {
		t.Fatal("expected construct Deployed flag and ActorID to be set")
	}

	if !g.gcRecallConstruct(first.ID) || first.Deployed {
		t.Fatal("gcRecallConstruct failed")
	}
}

func TestGrimoireSpellWeavingAndCasting(t *testing.T) {
	g := newGame(t, nil)

	sp, ok := g.gcWeaveSpell("Crimson Arc", "bolt", "fire", "vampiric")
	if !ok || sp == nil {
		t.Fatal("gcWeaveSpell failed")
	}
	if sp.Power <= 0 || sp.Cooldown <= 0 {
		t.Fatalf("unexpected woven spell stats: %+v", sp)
	}

	// Spawn a hostile foe in range.
	foe := g.Spawn("bandit", 2, g.R)
	foe.X = clampi(g.P.X+2, 1, MW-2)
	foe.Y = g.P.Y
	foe.HP = 50
	foe.MaxHP = 50
	foe.Angry = true
	g.Cur.Actors = append(g.Cur.Actors, foe)

	g.P.HP = maxi(1, g.P.MaxHP-10)
	hpBefore := g.P.HP
	dmg, ok := g.gcCastWoven(sp.ID, foe.X, foe.Y)
	if !ok || dmg <= 0 {
		t.Fatalf("gcCastWoven failed: ok=%v dmg=%d", ok, dmg)
	}
	if g.P.HP <= hpBefore {
		t.Fatalf("expected vampiric spell to heal caster: %d -> %d", hpBefore, g.P.HP)
	}
	// Second cast immediately should be blocked by cooldown.
	if _, ok := g.gcCastWoven(sp.ID, foe.X, foe.Y); ok {
		t.Fatal("expected woven spell to respect cooldown before Blood-Ink Seal")
	}

	// Sell copy to a scholar.
	scholar := g.Spawn("villager", 2, g.R)
	scholar.Name = "Scholar Sera"
	scholar.Role = "scholar"
	goldBefore := g.P.Gold
	price, ok := g.gcSellSpellCopy(sp.ID, scholar)
	if !ok || price <= 0 || g.P.Gold != goldBefore+price || sp.SoldCopies != 1 {
		t.Fatalf("gcSellSpellCopy failed: ok=%v price=%d sold=%d", ok, price, sp.SoldCopies)
	}
}

func TestCursedRelicBargains(t *testing.T) {
	g := newGame(t, nil)

	// 1. Sleepless Edge: grows sharper each day without sleep.
	edge, ok := g.gcBindBargain("sleepless_edge")
	if !ok || edge == nil || !edge.Bound {
		t.Fatal("failed to bind sleepless_edge")
	}
	g.Turn += 2 * DayLen
	gcTurn(g)
	if edge.Stacks < 2 {
		t.Fatalf("expected Sleepless Edge to gain stacks over sleepless days, got %d", edge.Stacks)
	}

	// 2. Usurer's Band: turns lethal hit into survival + underworld debt.
	band, ok := g.gcBindBargain("usurers_band")
	if !ok || band == nil {
		t.Fatal("failed to bind usurers_band")
	}
	g.P.HP = 0
	g.Dead = true
	gcTurn(g)
	if g.Dead || g.P.HP <= 0 {
		t.Fatalf("expected Usurer's Band to prevent death: dead=%v hp=%d", g.Dead, g.P.HP)
	}
	if gcSt(g).UnderworldDebt < 500 || band.Triggers == 0 {
		t.Fatalf("expected underworld debt >= 500, got %d (triggers %d)", gcSt(g).UnderworldDebt, band.Triggers)
	}

	// 3. Blood-Ink Seal: casts woven spells on cooldown using HP.
	seal, ok := g.gcBindBargain("blood_ink_seal")
	if !ok || seal == nil {
		t.Fatal("failed to bind blood_ink_seal")
	}
	sp, ok := g.gcWeaveSpell("Void Nova", "nova", "void", "silent")
	if !ok {
		t.Fatal("gcWeaveSpell failed")
	}
	sp.ReadyTurn = g.Turn + 10 // force on cooldown
	g.Cur.Actors = nil         // avoid incidental kill/level-up healing
	g.P.MaxHP = 60
	g.P.HP = 60
	g.P.XP = 0
	hpBefore := g.P.HP
	if _, ok := g.gcCastWoven(sp.ID, g.P.X, g.P.Y); !ok {
		t.Fatal("expected Blood-Ink Seal to allow casting while on cooldown")
	}
	if g.P.HP >= hpBefore || seal.Triggers == 0 {
		t.Fatalf("expected Blood-Ink Seal to consume HP: %d -> %d (triggers %d)", hpBefore, g.P.HP, seal.Triggers)
	}
}

func TestShortChallengeScenarios(t *testing.T) {
	g := newGame(t, nil)

	// 1. Escape the Besieged Capital
	run, ok := g.scnStart("escape_siege")
	if !ok || run == nil || run.Status != "active" {
		t.Fatal("failed to start escape_siege")
	}
	g.Turn += 20
	g.scnBreachCordon()
	g.scnBreachCordon()
	if run.Status != "won" || run.Rank == "" || run.Score <= 0 {
		t.Fatalf("expected escape_siege win with medal, got %+v", run)
	}

	// 2. Solve the 3-Day Murder Before the Festival
	run2, ok := g.scnStart("festival_murder")
	if !ok || run2 == nil || len(run2.Suspects) < 3 || run2.Culprit == "" {
		t.Fatalf("failed to start festival_murder: %+v", run2)
	}
	for i := 0; i < 3; i++ {
		if clue, ok := g.scnGatherClue(i); !ok || clue == "" {
			t.Fatalf("scnGatherClue(%d) failed", i)
		}
	}
	if ok, _ := g.scnAccuseCulprit(run2.Culprit); !ok || run2.Status != "won" {
		t.Fatalf("expected festival_murder win on accusing %s, got status=%s", run2.Culprit, run2.Status)
	}

	// 3. Deliver the Relic Across a Plague-Stricken Continent
	run3, ok := g.scnStart("plague_relic")
	if !ok || run3 == nil || g.scnHasRelic(run3) == nil {
		t.Fatal("failed to start plague_relic with cure relic in inventory")
	}
	if ok, _ := g.scnDeliverRelic(); !ok || run3.Status != "won" {
		t.Fatalf("expected plague_relic win on delivery, got status=%s", run3.Status)
	}

	if len(scnSt(g).Medals) != 3 {
		t.Fatalf("expected 3 scenario medals earned, got %d", len(scnSt(g).Medals))
	}

	// Verify timeout failure path.
	run4, ok := g.scnStart("escape_siege")
	if !ok {
		t.Fatal("failed to restart escape_siege")
	}
	g.Turn = run4.StartTurn + run4.LimitTurns + 5
	scnTurn(g)
	if run4.Status != "failed" {
		t.Fatalf("expected scenario timeout failure, got %s", run4.Status)
	}
}

func TestSocialMagicScenariosUIAndHooks(t *testing.T) {
	g := newGame(t, nil)

	// Verify planet-local keys are registered.
	have := map[string]bool{}
	for _, k := range unLocalKeys {
		have[k] = true
	}
	for _, want := range []string{"dg", "gc", "scn"} {
		if !have[want] {
			t.Fatalf("expected %q in unLocalKeys", want)
		}
	}

	// Verify modes draw and handle keys without panicking.
	for _, mode := range []string{"dg", "gc", "scn"} {
		if modeDraws[mode] == nil || modeKeys[mode] == nil {
			t.Fatalf("mode %q missing draw or key handler", mode)
		}
		g.Mode = mode
		g.Render()
		modeKeys[mode](g, "Tab")
		g.Render()
		modeKeys[mode](g, "Escape")
		if g.Mode != "main" {
			t.Fatalf("expected Escape to exit mode %q, got %q", mode, g.Mode)
		}
	}

	// Verify ext serialization flush works cleanly.
	g.flushExt()
	for _, k := range []string{"dg", "gc", "scn"} {
		if _, ok := g.Store[k]; !ok {
			t.Fatalf("expected %q in g.Store after flushExt", k)
		}
	}
}
