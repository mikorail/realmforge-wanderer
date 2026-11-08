package engine

import (
	"strings"
	"testing"
)

func TestCyDetectiveMystery(t *testing.T) {
	g := newGame(t, nil)
	burgID := g.StartBurg
	if burgID <= 0 {
		burgID = 1
	}

	m := g.cyGetMystery(burgID)
	if m == nil || m.Victim == "" || len(m.Suspects) != 3 || m.CulpritIdx < 0 || m.CulpritIdx > 2 {
		t.Fatalf("unexpected mystery: %+v", m)
	}

	// Discover all 3 clues
	for i := 0; i < 3; i++ {
		clue := g.cyDiscoverClue(burgID, i)
		if clue == "" || !m.FoundClues[i] {
			t.Fatalf("expected clue %d to be discovered, got %q", i, clue)
		}
		if !strings.Contains(clue, m.Suspects[m.CulpritIdx]) {
			t.Fatalf("expected clue %d to implicate culprit %q, got %q", i, m.Suspects[m.CulpritIdx], clue)
		}
	}

	// Accuse wrong suspect first -> false accusation fine (-30 gold, -20 rep)
	g.P.Gold = 100
	_, _, fac := g.cyResolveBurgID(burgID)
	rep0 := g.Rep[fac]
	wrongIdx := (m.CulpritIdx + 1) % 3
	ok, msg := g.cyAccuseSuspect(burgID, wrongIdx)
	if ok || m.Solved {
		t.Fatalf("expected wrong accusation to fail, got ok=%v msg=%q", ok, msg)
	}
	if g.P.Gold != 70 || g.Rep[fac] != rep0-20 {
		t.Fatalf("expected -30 gold and -20 rep on wrong accusation, got gold=%d rep=%d (was %d)", g.P.Gold, g.Rep[fac], rep0)
	}

	// Accuse actual culprit -> +120 gold, +35 rep, +80 XP, "Consulting Detective" deed & title
	goldBefore := g.P.Gold
	repBefore := g.Rep[fac]
	xpBefore := g.XP + g.P.Level*1000
	ok, msg = g.cyAccuseSuspect(burgID, m.CulpritIdx)
	if !ok || !m.Solved {
		t.Fatalf("expected culprit accusation to succeed, got ok=%v msg=%q", ok, msg)
	}
	if g.P.Gold != goldBefore+120 {
		t.Fatalf("expected +120 gold on solving mystery, got %d (was %d)", g.P.Gold, goldBefore)
	}
	if g.Rep[fac] != repBefore+35 {
		t.Fatalf("expected +35 rep on solving mystery, got %d (was %d)", g.Rep[fac], repBefore)
	}
	if g.XP+g.P.Level*1000 <= xpBefore {
		t.Fatalf("expected +80 XP on solving mystery")
	}
	foundDeed := false
	for _, d := range g.Deeds {
		if strings.Contains(d.Text, "Consulting Detective") {
			foundDeed = true
			break
		}
	}
	if !foundDeed {
		t.Fatalf("expected Consulting Detective deed to be recorded in g.Deeds")
	}
}

func TestCyCityJobs(t *testing.T) {
	g := newGame(t, nil)
	_, _, fac := g.cyResolveBurgID(0)

	cases := []struct {
		job     string
		minGold int
		maxGold int
		repGain int
	}{
		{"porter", 15, 30, 0},
		{"scribe", 18, 35, 0},
		{"guard_shift", 20, 40, 5},
		{"healer_aide", 18, 32, 5},
	}

	for _, tc := range cases {
		t0 := g.Turn
		g0 := g.P.Gold
		r0 := g.Rep[fac]
		earned, msg := g.cyWorkJob(tc.job)
		if earned < tc.minGold || earned > tc.maxGold {
			t.Fatalf("job %q earned %d gold, expected [%d, %d] (msg=%q)", tc.job, earned, tc.minGold, tc.maxGold, msg)
		}
		if g.Turn != t0+240 {
			t.Fatalf("job %q should advance 240 turns (4 hours), got %d -> %d", tc.job, t0, g.Turn)
		}
		if g.P.Gold != g0+earned {
			t.Fatalf("job %q did not credit gold: %d -> %d (earned %d)", tc.job, g0, g.P.Gold, earned)
		}
		if g.Rep[fac] != r0+tc.repGain {
			t.Fatalf("job %q rep delta mismatch: %d -> %d (want +%d)", tc.job, r0, g.Rep[fac], tc.repGain)
		}
	}
}

func TestCyNightLifeInCities(t *testing.T) {
	g := newGame(t, nil)
	if g.Cur.Kind != ZTown {
		t.Fatalf("expected start zone to be town, got %q", g.Cur.Kind)
	}

	// Turn 0 is 08:00 (daytime)
	g.Turn = 0
	if g.cyIsNight() {
		t.Fatalf("expected Turn 0 (08:00) not to be night (Hour=%d)", g.cyHour())
	}

	// Advance to 22:00 (14 hours * 60 minutes = 840 turns)
	g.Turn = 14 * 60
	if !g.cyIsNight() {
		t.Fatalf("expected Turn 840 (22:00) to be night (Hour=%d)", g.cyHour())
	}

	g.cyEnsureNightActors(g.Cur)

	var guild, smuggler, ghost *Actor
	for _, a := range g.Cur.Actors {
		switch a.Role {
		case "cy_nightguild":
			guild = a
		case "cy_smuggler":
			smuggler = a
		case "cy_ghost":
			ghost = a
		}
	}
	if guild == nil || smuggler == nil || ghost == nil {
		t.Fatalf("expected all 3 night actors to spawn in town at night: guild=%v smuggler=%v ghost=%v", guild != nil, smuggler != nil, ghost != nil)
	}
	if !ghost.Has("undead") || g.Hostile(ghost) {
		t.Fatalf("expected Restless Town Ghost to have undead flag and be peaceful unless attacked")
	}

	// Verify ghost talkHook whispers a mystery clue
	opts := cyTalkOptions(g, ghost)
	if len(opts) == 0 {
		t.Fatalf("expected talk options on Restless Town Ghost")
	}
	opts[0].Do()
	m := g.cyGetMystery(g.StartBurg)
	if m.cluesFoundCount() == 0 {
		t.Fatalf("expected ghost whisper to reveal a murder mystery clue")
	}

	// Advance to 09:00 next day (25 hours * 60 = 1500 turns) -> night actors vanish
	g.Turn = 25 * 60
	if g.cyIsNight() {
		t.Fatalf("expected 09:00 not to be night (Hour=%d)", g.cyHour())
	}
	g.cyEnsureNightActors(g.Cur)
	for _, a := range g.Cur.Actors {
		if isCyNightRole(a.Role) {
			t.Fatalf("expected night-life actor %q (%s) to vanish by day", a.Name, a.Role)
		}
	}
}
