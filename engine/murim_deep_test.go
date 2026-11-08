package engine

import (
	"strings"
	"testing"
)

func TestMdBeggarMasterAlmsAndUnmask(t *testing.T) {
	g := newGame(t, nil)
	beggar := mdSpawnBeggarMaster(g, g.Cur, g.R)
	if beggar == nil || beggar.Role != "md_beggar" || !beggar.Has("md_hidden_master") {
		t.Fatalf("expected hidden beggar master in town, got %+v", beggar)
	}

	opts := mdTalkHook(g, beggar)
	if len(opts) < 2 {
		t.Fatalf("expected at least 2 dialogue options for beggar master, got %d", len(opts))
	}

	g.P.Gold = 50
	xpBefore := g.XP
	tech := g.mdOfferAlms(beggar)
	if tech == "" || !g.P.Has("md_master_blessed") || !g.P.Has(tech) {
		t.Fatalf("expected beggar alms to grant secret tech and blessing, got tech=%q", tech)
	}
	if g.XP <= xpBefore && g.P.Level == 1 {
		t.Fatalf("expected +80 XP from offering alms, xpBefore=%d xpNow=%d", xpBefore, g.XP)
	}

	// Test unmasking a second beggar into a hostile Level 12 Grandmaster
	beggar2 := &Actor{
		ID:    g.newID(),
		Name:  "ragged beggar",
		Role:  "md_beggar",
		HP:    20,
		MaxHP: 20,
		Body:  MakeBody("humanoid"),
	}
	beggar2.Set("md_hidden_master")
	g.mdUnmaskMaster(beggar2)
	if beggar2.Level != 12 || beggar2.MaxHP != 120 || beggar2.AVNat != 8 || beggar2.DVNat != 8 || !g.Hostile(beggar2) {
		t.Fatalf("expected unmasked master to be Lv12 (120 HP, 8 AV/DV, hostile), got lv=%d hp=%d av=%d dv=%d hostile=%v",
			beggar2.Level, beggar2.MaxHP, beggar2.AVNat, beggar2.DVNat, g.Hostile(beggar2))
	}
}

func TestMdSectGrudgeAndAvengers(t *testing.T) {
	g := newGame(t, nil)
	disc := placeFoe(g, "bandit", 3)
	disc.Name = "Zhao Feng"
	disc.Role = "disciple"
	disc.Faction = "sect_Crimson_Lotus"
	disc.Set("mr_disciple")

	mdOnKill(g, disc, g.P)
	if len(g.md().Grudges) != 1 {
		t.Fatalf("expected 1 sect grudge recorded, got %d", len(g.md().Grudges))
	}
	gr := g.md().Grudges[0]
	if gr.Stage != 1 || gr.DueDay != g.Day()+3 {
		t.Fatalf("unexpected initial grudge state: %+v", gr)
	}

	// Advance 3 days and verify Stage 1 avenger spawns and advances stage to 2
	g.Turn += 3 * DayLen
	av1 := g.mdCheckGrudgeHunter()
	if av1 == nil || av1.Role != "md_avenger" || !g.Hostile(av1) || gr.Stage != 2 {
		t.Fatalf("expected stage 1 avenger and stage->2, got av=%+v stage=%d", av1, gr.Stage)
	}
	if !strings.Contains(g.Msgs[len(g.Msgs)-1].T, g.P.Name) || !strings.Contains(g.Msgs[len(g.Msgs)-1].T, "Zhao Feng") {
		t.Fatalf("expected avenger to call out player and victim by name, got %q", g.Msgs[len(g.Msgs)-1].T)
	}
}

func TestMdLineageSeclusionAndDemonicArts(t *testing.T) {
	g := newGame(t, nil)
	clearAround(g)

	// 1. Lineage recording and application to heir
	g.P.Attr[STR] = 24
	g.md().Techs = []string{"md_finger"}
	lin := g.mdRecordLineage()
	if lin == nil || lin.MaxAttr != STR || lin.SecretTech != "md_finger" {
		t.Fatalf("unexpected lineage: %+v", lin)
	}
	strBefore := g.P.Attr[STR]
	if !g.mdApplyLineageToHeir() {
		t.Fatalf("expected mdApplyLineageToHeir to succeed")
	}
	if g.P.Attr[STR] != strBefore+2 || !g.P.Has("Lineage Disciple") || !g.P.Has("md_finger") {
		t.Fatalf("expected heir to gain +2 STR, Lineage Disciple, and md_finger")
	}

	// 2. Forbidden demonic arts, corruption aura, and qi deviation
	foe := placeFoe(g, "bandit", 4)
	foe.HP, foe.MaxHP = 300, 300
	g.P.MaxHP = 100
	g.P.HP = 50
	if !g.mdLearnDemonic("blood_sutra") {
		t.Fatalf("expected mdLearnDemonic to succeed")
	}
	for i := 0; i < 7; i++ {
		dmg := g.mdUseDemonic("blood_sutra", foe)
		if dmg <= 0 {
			t.Fatalf("expected positive demonic damage, got %d", dmg)
		}
	}
	if g.md().Corruption < 100 || !g.P.Has("md_demonic_aura") || !g.P.Has("md_deviation") {
		t.Fatalf("expected corruption>=100, demonic aura, and qi deviation; got corr=%d aura=%v dev=%v",
			g.md().Corruption, g.P.Has("md_demonic_aura"), g.P.Has("md_deviation"))
	}

	// 3. Closed-door seclusion clears deviation and breaks bottleneck
	clearAround(g)
	dayBefore := g.Day()
	wilBefore := g.P.Attr[WIL]
	maxHPBefore := g.P.MaxHP
	msg := g.mdSeclude(10)
	if g.Day() != dayBefore+10 {
		t.Fatalf("expected seclusion to advance 10 days, got %d -> %d", dayBefore, g.Day())
	}
	if g.P.Has("md_deviation") || g.P.Attr[WIL] <= wilBefore || g.P.MaxHP != maxHPBefore+5 || g.P.HP != g.P.MaxHP {
		t.Fatalf("expected seclusion to clear deviation, boost WIL & MaxHP, and restore full HP; msg=%q", msg)
	}
}
