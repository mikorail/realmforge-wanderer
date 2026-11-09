package engine

import (
	"strings"
	"testing"
)

// Tests covering all 6 brainstormed systems in The Wanderer:
// 1. Naval Sailing, Flagships & High-Seas Piracy (engine/naval.go)
// 2. Dynastic Bloodline Curses, Heirs & Pretender Quests (engine/dynasties_rogue.go)
// 3. Subterranean Verticality, Spore Canopies & Acoustic Stealth (engine/subterranean.go)
// 4. Thaumaturgical Smog, Spell Backfires & Crystalline Mutations (engine/thaumaturgy.go)
// 5. Living Faiths: Divine Oaths, Inquisitors & Shrines (engine/saints_faith.go)
// 6. Trans-Continental Banking, Loans & Commodity Arbitrage (engine/banking_economy.go)

func TestExpansionNavalSystem(t *testing.T) {
	g := newGame(t, nil)

	// 1. Commandeer vessels
	if !g.CommandeerVessel(nvFlagCaravel, "Sea Sovereign") {
		t.Fatalf("expected caravel commandeering to succeed")
	}
	st := nvSt(g)
	if st.Flagship == nil || st.Flagship.Kind != nvFlagCaravel || st.Flagship.Name != "Sea Sovereign" {
		t.Fatalf("expected flagship to be Sea Sovereign caravel, got %+v", st.Flagship)
	}

	// Commandeer longship
	g.CommandeerVessel(nvFlagLongship, "Storm Dragon")
	if st.Flagship.Kind != nvFlagLongship {
		t.Fatalf("expected longship flagship, got %s", st.Flagship.Kind)
	}

	// Commandeer submarine bathysphere
	g.CommandeerVessel(nvFlagBathysphere, "Abyssal Nautilus")
	if st.Flagship.Kind != nvFlagBathysphere || !st.Flagship.Submerged == false {
		t.Fatalf("expected bathysphere flagship, got %+v", st.Flagship)
	}

	// 2. Submarine bathysphere submerge / surface toggle
	ok, _ := g.SubmergeBathysphere()
	if !ok || !st.Flagship.Submerged {
		t.Fatalf("expected bathysphere to submerge")
	}
	if _, hasSt := g.P.Status["submerged_diving"]; !hasSt {
		t.Fatalf("expected player to gain submerged_diving status")
	}
	// Surface
	ok, _ = g.SubmergeBathysphere()
	if !ok || st.Flagship.Submerged {
		t.Fatalf("expected bathysphere to surface")
	}
	if _, hasSt := g.P.Status["submerged_diving"]; hasSt {
		t.Fatalf("expected submerged_diving status to be cleared upon surfacing")
	}

	// 3. Firing Swivel Cannon
	st.Flagship.SwivelAmmo = 5
	target := &Actor{
		ID:    g.newID(),
		Name:  "Corsair Skiff",
		HP:    40,
		MaxHP: 40,
		X:     g.P.X + 3,
		Y:     g.P.Y,
	}
	dmg, fired := g.FireSwivelCannon(target)
	if !fired || dmg <= 0 {
		t.Fatalf("expected swivel cannon to fire and deal damage, got fired=%v dmg=%d", fired, dmg)
	}
	if target.HP >= 40 {
		t.Fatalf("expected target to take damage from swivel, got HP=%d", target.HP)
	}
	if st.Flagship.SwivelAmmo != 4 {
		t.Fatalf("expected swivel ammo to decrement to 4, got %d", st.Flagship.SwivelAmmo)
	}

	// 4. Gangplank deployment
	gx, gy := g.P.X+1, g.P.Y
	g.Cur.Set(gx, gy, TDeep)
	if !g.DeployGangplank(gx, gy) {
		t.Fatalf("expected gangplank to deploy across deep water")
	}
	if g.Cur.At(gx, gy) != TBridge {
		t.Fatalf("expected tile to become TBridge, got %v", g.Cur.At(gx, gy))
	}
	if len(st.Gangplanks) != 1 {
		t.Fatalf("expected 1 gangplank recorded, got %d", len(st.Gangplanks))
	}

	// 5. Kraken Sea Encounter
	kraken := g.SpawnKrakenEncounter()
	if kraken == nil || kraken.Tpl != "abyssal_kraken" {
		t.Fatalf("expected Abyssal Kraken to spawn, got %+v", kraken)
	}
	foundTentacle := false
	for _, a := range g.Cur.Actors {
		if a != nil && a.Tpl == "kraken_tentacle" {
			foundTentacle = true
			break
		}
	}
	if !foundTentacle {
		t.Fatalf("expected kraken tentacles to spawn with the encounter")
	}

	// Slaying the kraken triggers killHook
	kraken.HP = 0
	for _, hook := range killHooks {
		hook(g, kraken, g.P)
	}
	if st.KrakensSlayed != 1 {
		t.Fatalf("expected KrakensSlayed to be 1, got %d", st.KrakensSlayed)
	}

	// 6. Boarding action
	if !g.BoardingEncounter("The Crimson Galleon") {
		t.Fatalf("expected boarding encounter to succeed")
	}
	if st.PrizesTaken != 1 || st.Infamy <= 0 {
		t.Fatalf("expected PrizesTaken=1 and Infamy > 0, got prizes=%d infamy=%d", st.PrizesTaken, st.Infamy)
	}
}

func TestExpansionDynastiesSystem(t *testing.T) {
	g := newGame(t, nil)

	// 1. Setup dynasty with Dragon-Heart
	g.SetupDynasty("House of Val-Kor", TraitDragonHeart)
	st := dySt(g)
	if st.HouseName != "House of Val-Kor" || st.Bloodline != TraitDragonHeart {
		t.Fatalf("expected House of Val-Kor with Dragon-Heart, got %+v", st)
	}
	if st.Heirloom == nil || len(st.Heirloom.Deeds) == 0 {
		t.Fatalf("expected heirloom weapon to be initialized")
	}

	// 2. Accrue deeds on heirloom
	oldPV := st.Heirloom.PV
	g.AccrueHeirloomDeed("Slew the Abyssal Terror of the Iron Deep")
	if len(st.Heirloom.Deeds) != 2 || st.Heirloom.PV <= oldPV {
		t.Fatalf("expected deed to be recorded and PV to increase")
	}

	// 3. Test Giant Stature and Red Miasma trait effects
	g.SetupDynasty("House of Giants", TraitGiantStature)
	if st.Bloodline != TraitGiantStature {
		t.Fatalf("expected Giant Stature trait")
	}

	g.SetupDynasty("House of Crimson", TraitRedMiasma)
	g.P.AddSt("poisoned", 50)
	g.applyBloodlinePerks()
	if _, poisoned := g.P.Status["poisoned"]; poisoned {
		t.Fatalf("expected Red Miasma to cleanse poison")
	}

	// 4. Continue as generational heir
	genBefore := st.Generation
	if !g.ContinueDynastyHeir() {
		t.Fatalf("expected heir continuation to succeed")
	}
	if st.Generation != genBefore+1 {
		t.Fatalf("expected generation to increment to %d, got %d", genBefore+1, st.Generation)
	}
	// Verify heirloom is in heir's inventory
	foundHeirloom := false
	for _, it := range g.P.Inv {
		if it != nil && it.Name == st.Heirloom.Name {
			foundHeirloom = true
			break
		}
	}
	if !foundHeirloom {
		t.Fatalf("expected heirloom to be transferred to heir inventory")
	}

	// 5. Pretender / Usurper Quests
	claimant := &PretenderClaimant{
		ID:          "pretender_alistair",
		Name:        "Prince Alistair the Exiled",
		Realm:       "The Sunken Marches",
		UsurperName: "Usurper King Morvath",
		CitadelTown: "Oakhaven Citadel",
		Title:       "Rightful Heir",
	}
	if !g.SwearFealtyPretender(claimant) {
		t.Fatalf("expected swearing fealty to succeed")
	}
	if st.SwornClaimant == nil || st.SwornClaimant.Name != "Prince Alistair the Exiled" {
		t.Fatalf("expected sworn claimant to be Alistair")
	}

	// Infiltrate citadel and depose usurper
	goldBefore := g.P.Gold
	if !g.InfiltrateCitadel() {
		t.Fatalf("expected citadel infiltration coup to succeed")
	}
	if !st.CitadelInfiltrated {
		t.Fatalf("expected CitadelInfiltrated to be true")
	}
	if g.P.Gold <= goldBefore {
		t.Fatalf("expected royal gold reward from deposed usurper")
	}
}

func TestExpansionSubterraneanSystem(t *testing.T) {
	g := newGame(t, nil)
	g.Cur.Z = 2 // Subterranean depth

	if !g.IsSubterranean() {
		t.Fatalf("expected zone to be recognized as subterranean at Z=2")
	}

	// 1. Mushroom canopy climbing
	mx, my := g.P.X+1, g.P.Y
	g.Cur.Set(mx, my, TMushroom)
	if !g.ClimbMushroomCanopy() {
		t.Fatalf("expected climbing adjacent mushroom canopy to succeed")
	}
	if _, ok := g.P.Status["canopy_perch"]; !ok {
		t.Fatalf("expected player to have canopy_perch status")
	}

	// 2. Vibration calculation: on canopy vibration must be 0!
	vibOnCanopy := g.AcousticVibrationLevel(g.P)
	if vibOnCanopy != 0 {
		t.Fatalf("expected 0 vibration while on canopy, got %d", vibOnCanopy)
	}

	// Descend and check ground vibration
	g.DescendCanopy()
	if _, ok := g.P.Status["canopy_perch"]; ok {
		t.Fatalf("expected canopy_perch to be cleared")
	}
	g.Cur.Set(g.P.X, g.P.Y, TFloor)
	vibOnFloor := g.AcousticVibrationLevel(g.P)
	if vibOnFloor <= 0 {
		t.Fatalf("expected positive vibration on stone floor, got %d", vibOnFloor)
	}

	// 3. Canopy leaping and drop ambush
	g.ClimbMushroomCanopy()
	targetMushroomX, targetMushroomY := g.P.X+2, g.P.Y
	g.Cur.Set(targetMushroomX, targetMushroomY, TMushroom)
	if !g.LeapMushroomCanopy(targetMushroomX, targetMushroomY) {
		t.Fatalf("expected leaping to another canopy to succeed")
	}
	if g.P.X != targetMushroomX || g.P.Y != targetMushroomY {
		t.Fatalf("expected player to leap to target canopy coordinates")
	}

	// Drop ambush onto hostile foe
	foeX, foeY := g.P.X+1, g.P.Y
	g.Cur.Set(foeX, foeY, TFloor)
	foe := &Actor{
		ID:    g.newID(),
		Name:  "Cave Stalker",
		HP:    30,
		MaxHP: 30,
		X:     foeX,
		Y:     foeY,
	}
	g.Cur.Actors = append(g.Cur.Actors, foe)
	if !g.LeapMushroomCanopy(foeX, foeY) {
		t.Fatalf("expected drop ambush leap to succeed")
	}
	if foe.HP >= 30 {
		t.Fatalf("expected drop ambush to damage foe, got HP=%d", foe.HP)
	}

	// 4. Blind apex predator and acoustic decoy
	pred := g.SpawnBlindAbyssPredator(g.P.X+4, g.P.Y)
	if pred == nil || pred.Tpl != "abyssal_tremor_beast" {
		t.Fatalf("expected Abyssal Tremor-Beast to spawn")
	}

	// Throw acoustic decoy to divert predator
	decoyX, decoyY := g.P.X+5, g.P.Y+2
	g.Cur.Set(decoyX, decoyY, TFloor)
	if !g.ThrowAcousticDecoy(decoyX, decoyY) {
		t.Fatalf("expected throwing acoustic decoy to succeed")
	}
	st := sbSt(g)
	if st.DecoysThrown != 1 || st.LastNoiseVal != 8 {
		t.Fatalf("expected decoy noise recorded in state, got %+v", st)
	}
}

func TestExpansionThaumaturgySystem(t *testing.T) {
	g := newGame(t, nil)

	// 1. Smog accumulation
	if g.GetThaumicSmog() != 0 {
		t.Fatalf("expected initial smog to be 0")
	}
	g.AddThaumicSmog(30)
	if g.GetThaumicSmog() != 30 {
		t.Fatalf("expected smog to be 30, got %d", g.GetThaumicSmog())
	}

	// 2. Thaumaturgical research
	g.P.Attr[INT] = 18 // High INT increases success
	invCountBefore := len(g.P.Inv)
	_, _ = g.ConductThaumaturgicalResearch("Aether-Dynamics")
	st := thmSt(g)
	if st.ResearchSuccesses == 0 && st.BackfiresTriggered == 0 {
		t.Fatalf("expected either research success or backfire recorded")
	}

	// 3. Explicit spell backfires
	// Temporal Echo
	msg := g.TriggerSpellBackfire(BackfireTemporalEcho)
	if !strings.Contains(msg, "temporal echo") {
		t.Fatalf("expected temporal echo message, got %s", msg)
	}
	if st.TemporalEchoTurns <= 0 {
		t.Fatalf("expected temporal echo turns active")
	}

	// Localized Gravity Inversion
	msg = g.TriggerSpellBackfire(BackfireGravityInversion)
	if !strings.Contains(msg, "gravity inversion") {
		t.Fatalf("expected gravity inversion message, got %s", msg)
	}
	if _, ok := g.P.Status["gravity_inverted"]; !ok {
		t.Fatalf("expected gravity_inverted status on player")
	}

	// Warp Cascade
	_ = g.TriggerSpellBackfire(BackfireWarpCascade)
	if st.ManaSickness <= 0 {
		t.Fatalf("expected mana sickness from warp cascade")
	}

	// 4. Mana Sickness and Crystalline Skin mutation
	g.ApplyManaSickness(40)
	if !st.HasCrystallineSkin {
		t.Fatalf("expected Crystalline Skin mutation at 30+ mana sickness")
	}
	if _, ok := g.P.Status["crystalline_skin"]; !ok {
		t.Fatalf("expected crystalline_skin status on player")
	}
	if g.P.AVNat < 4 {
		t.Fatalf("expected +4 AV from crystalline skin mutation, got AVNat=%d", g.P.AVNat)
	}

	// 5. Mana Leech spawning
	leech := g.SpawnManaLeech(g.P.X+1, g.P.Y)
	if leech == nil || leech.Tpl != "mana_leech" {
		t.Fatalf("expected Mana Leech to spawn")
	}
	if st.LeechesSpawned != 1 {
		t.Fatalf("expected LeechesSpawned to be 1")
	}

	// 6. Purging mana sickness
	if !g.PurgeThaumicSickness() {
		t.Fatalf("expected purging mana sickness to succeed")
	}
	if st.ManaSickness != 0 {
		t.Fatalf("expected mana sickness to be 0 after purge")
	}
	_ = invCountBefore
}

func TestExpansionSaintsFaithSystem(t *testing.T) {
	g := newGame(t, nil)

	// 1. Swear Divine Oath of the Iron Fist
	if !g.SwearDivineOath(OathIronFist) {
		t.Fatalf("expected swearing Oath of the Iron Fist to succeed")
	}
	st := sfSt(g)
	if st.ActiveOath != OathIronFist || st.Piety <= 0 {
		t.Fatalf("expected Oath of the Iron Fist active with piety, got %+v", st)
	}

	// 2. Divine Invocations
	ok, msg := g.InvokeDivinePower("Fist of Heaven")
	if !ok || !strings.Contains(msg, "Fist of Heaven") {
		t.Fatalf("expected Fist of Heaven invocation to succeed, got ok=%v msg=%s", ok, msg)
	}

	// 3. Taboo violation: equip bladed weapon under Oath of Iron Fist
	blade := &Item{
		ID:    g.newID(),
		Name:  "Steel Broadsword",
		Kind:  "weapon",
		Skill: "blade",
	}
	g.P.Body[0].Item = blade
	g.CheckOathTaboos()

	if !st.IsApostate {
		t.Fatalf("expected player to be branded an Apostate for wielding a blade under Oath of the Iron Fist")
	}
	if st.ActiveOath != "" {
		t.Fatalf("expected active oath to be revoked upon apostasy")
	}
	if _, ok := g.P.Status["apostate_curse"]; !ok {
		t.Fatalf("expected apostate_curse status on player")
	}

	// 4. Inquisitor Hit Squad dispatch
	squad := g.SpawnInquisitorSquad()
	if len(squad) < 2 {
		t.Fatalf("expected inquisitor hit squad to contain multiple members, got %d", len(squad))
	}
	if squad[0].Tpl != "grand_inquisitor" {
		t.Fatalf("expected squad leader to be Grand Inquisitor, got %s", squad[0].Tpl)
	}

	// 5. Pray at living saint shrine
	hpBefore := g.P.HP
	g.P.HP = 10
	ok, pMsg := g.PrayAtSaintShrine("Saint Valeria the Unsleeping")
	if !ok || !strings.Contains(pMsg, "Saint Valeria") {
		t.Fatalf("expected prayer at saint shrine to succeed, got %s", pMsg)
	}
	if g.P.HP <= 10 {
		t.Fatalf("expected health restored from saint shrine prayer")
	}
	if _, ok := g.P.Status["saintly_blessing"]; !ok {
		t.Fatalf("expected saintly_blessing status from shrine")
	}
	_ = hpBefore
}

func TestExpansionBankingEconomySystem(t *testing.T) {
	g := newGame(t, nil)
	g.P.Gold = 600

	// 1. Promissory Notes issuance and redemption
	note, ok := g.IssuePromissoryNote(200)
	if !ok || note == nil {
		t.Fatalf("expected promissory note issuance to succeed")
	}
	if note.Kind != "promissory_note" || note.Value != 200 {
		t.Fatalf("expected 200 crown promissory note, got %+v", note)
	}
	// Check fee deducted (200 + 4 fee = 204)
	if g.P.Gold != 396 {
		t.Fatalf("expected 396 gold remaining after note and fee, got %d", g.P.Gold)
	}

	// Redeem note
	redeemedVal, rOk := g.RedeemPromissoryNote(note)
	if !rOk || redeemedVal != 200 {
		t.Fatalf("expected redemption of 200 gold, got val=%d ok=%v", redeemedVal, rOk)
	}
	if g.P.Gold != 596 {
		t.Fatalf("expected 596 gold after redeeming 200 note, got %d", g.P.Gold)
	}

	// 2. Sovereign Bank Loan
	goldBefore := g.P.Gold
	if !g.TakeBankLoan("Iron Vault Mercantile Bank", 500, 30) {
		t.Fatalf("expected bank loan approval")
	}
	st := bkSt(g)
	if len(st.Loans) != 1 || st.Loans[0].Principal != 500 {
		t.Fatalf("expected 1 active loan of 500 principal, got %+v", st.Loans)
	}
	if g.P.Gold != goldBefore+500 {
		t.Fatalf("expected gold to increase by loan principal")
	}

	// 3. Commodity Arbitrage Price Checking across biomes
	silkDesert := g.GetCommodityPrice("silk", "desert")
	silkTundra := g.GetCommodityPrice("silk", "tundra")
	if silkTundra <= silkDesert {
		t.Fatalf("expected silk to command a much higher price in tundra than desert, got tundra=%d desert=%d", silkTundra, silkDesert)
	}

	spicesDesert := g.GetCommodityPrice("spices", "desert")
	spicesTundra := g.GetCommodityPrice("spices", "tundra")
	if spicesTundra <= spicesDesert*2 {
		t.Fatalf("expected massive arbitrage spread on spices in tundra vs desert, got desert=%d tundra=%d", spicesDesert, spicesTundra)
	}

	saltCave := g.GetCommodityPrice("abyssal_salt", "cave")
	saltSurface := g.GetCommodityPrice("abyssal_salt", "forest")
	if saltSurface <= saltCave {
		t.Fatalf("expected abyssal salt to be cheaper in deep caves than surface, got cave=%d surface=%d", saltCave, saltSurface)
	}

	// 4. Loan Default and Debt Bounty Enforcers
	// Advance loan due day past current day to simulate default
	st.Loans[0].DueDay = g.Day() - 1
	g.CheckDebtDefault()
	if !st.Loans[0].Defaulted {
		t.Fatalf("expected loan to be marked defaulted")
	}
	if st.DefaultBounty <= 0 {
		t.Fatalf("expected debt bounty to be placed on player")
	}

	// Verify wiring: bhPlayerBounty includes defaulted debt bounty
	totalBounty := bhPlayerBounty(g, 0)
	if totalBounty < st.DefaultBounty {
		t.Fatalf("expected bhPlayerBounty to include banking debt bounty (%d), got total=%d", st.DefaultBounty, totalBounty)
	}

	// Spawn debt enforcers
	enforcers := g.SpawnDebtEnforcers()
	if len(enforcers) < 2 {
		t.Fatalf("expected debt enforcers squad to spawn")
	}
	if enforcers[0].Tpl != "debt_bailiff" {
		t.Fatalf("expected squad leader to be bailiff, got %s", enforcers[0].Tpl)
	}

	// Repay loan to clear debt and bounty
	loanID := st.Loans[0].ID
	g.P.Gold += 1000 // Ensure sufficient funds to settle
	if !g.RepayBankLoan(loanID) {
		t.Fatalf("expected loan repayment to succeed")
	}
	if len(st.Loans) != 0 {
		t.Fatalf("expected loans to be empty after repayment")
	}
	if st.DefaultBounty != 0 {
		t.Fatalf("expected default bounty to be cleared after full repayment, got %d", st.DefaultBounty)
	}
}
