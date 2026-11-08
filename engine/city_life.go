package engine

import (
	"fmt"
	"strings"
)

// City life systems (prefix cy):
// 1. Detective mysteries (cyMystery): murder investigation in towns with 3 suspects, 3 clues
//    (physical trace, witness statement, motive note), and courtroom accusation before the
//    magistrate/elder ("Consulting Detective" deed & title on solving).
// 2. City jobs for coin (cyWorkJob / cyWorkShift): 4-hour shifts as porter, scribe, guard_shift,
//    or healer_aide with attribute checks, gold wages, and town reputation.
// 3. Night-life in cities (cyIsNight, cyEnsureNightActors): between 21:00 and 05:00 in town zones,
//    spawns a Night-Guild Operative, a Lantern Smuggler, and a Restless Town Ghost; removes them
//    when dawn breaks.

const ZTown = "town"

type cyMystery struct {
	BurgID     int       `json:"burgId"`
	Town       string    `json:"town"`
	Victim     string    `json:"victim"`
	Suspects   [3]string `json:"suspects"`
	CulpritIdx int       `json:"culpritIdx"`
	Clues      [3]string `json:"clues"`
	FoundClues [3]bool   `json:"foundClues"`
	Solved     bool      `json:"solved"`
	Accused    bool      `json:"accused"`
}

type cyState struct {
	Mysteries   map[int]*cyMystery `json:"mysteries,omitempty"`
	JobsWorked  int                `json:"jobsWorked,omitempty"`
	TitleEarned bool               `json:"titleEarned,omitempty"`
	ShadowDone  map[int]bool       `json:"shadowDone,omitempty"`
}

func (g *Game) cy() *cyState {
	st := g.ext("cy", func() any { return &cyState{} }).(*cyState)
	if st.Mysteries == nil {
		st.Mysteries = map[int]*cyMystery{}
	}
	if st.ShadowDone == nil {
		st.ShadowDone = map[int]bool{}
	}
	return st
}

func init() {
	unLocalKeys = append(unLocalKeys, "cy")
	enterHooks = append(enterHooks, func(g *Game, zn *Zone) {
		g.cyEnsureNightActors(zn)
	})
	turnHooks = append(turnHooks, func(g *Game) {
		if g != nil && g.Cur != nil && g.Cur.Kind == ZTown {
			g.cyEnsureNightActors(g.Cur)
		}
	})
	talkHooks = append(talkHooks, cyTalkOptions)
}

// ---- 1. Detective Mysteries ----

func (g *Game) cyResolveBurgID(burgID int) (int, string, string) {
	if g == nil || g.W == nil {
		return burgID, "the town", "state:1"
	}
	if burgID <= 0 {
		if bd := g.localBurg(); bd != nil {
			burgID = bd.ID
		} else if g.Cur != nil && g.Cur.Burg > 0 {
			burgID = g.Cur.Burg
		} else if g.StartBurg > 0 {
			burgID = g.StartBurg
		} else if len(g.W.Burgs) > 0 {
			burgID = g.W.Burgs[0].ID
		} else {
			burgID = 1
		}
	}
	town := "the settlement"
	fac := "state:1"
	if bd := g.W.burgByID[burgID]; bd != nil {
		town = bd.Name
		if bd.State > 0 {
			fac = fmt.Sprintf("state:%d", bd.State)
		}
	} else if g.Cur != nil {
		if g.Cur.Name != "" {
			town = g.Cur.Name
		}
		if g.Cur.Faction != "" {
			fac = g.Cur.Faction
		}
	}
	return burgID, town, fac
}

// cyGetMystery deterministically creates or retrieves the murder mystery for a town.
func (g *Game) cyGetMystery(burgID int) *cyMystery {
	if g == nil {
		return nil
	}
	id, town, _ := g.cyResolveBurgID(burgID)
	st := g.cy()
	if m, ok := st.Mysteries[id]; ok && m != nil {
		return m
	}
	r := NewRng(fmt.Sprintf("%s:cy:mystery:%d", g.Seed, id))
	victims := []string{
		"Master Coin-Assayer Vaelen",
		"Guild-Factor Orla of the Weigh-House",
		"Archivist Toren of the Old Charter",
		"Harbor-Bailiff Maren",
	}
	suspectSets := [][3]string{
		{"Aldous the Ledger-Clerk", "Sera the Night-Watch Apothecary", "Voss the Debt-Collector"},
		{"Captain Halrik of the Gate", "Mirela the Silk-Broker", "Brother Caelen the Bell-warden"},
		{"Dorian the Lock-Wright", "Lyra the Spice-Factor", "Garrick the Tavern-Bailiff"},
	}
	vName := Pick(r, victims)
	sSet := Pick(r, suspectSets)
	culprit := r.Int(0, 2)
	culpritName := sSet[culprit]

	clues := [3]string{
		fmt.Sprintf("Physical trace: A torn cuff stained with rare ink and seal-wax matching %s's kit was caught on the window latch.", culpritName),
		fmt.Sprintf("Witness statement: A lamplighter swore they saw %s hurrying from %s's doorway just before the midnight bell.", culpritName, vName),
		fmt.Sprintf("Motive note: A half-burned ledger page shows %s owed a ruinous sum to %s and faced exposure at dawn.", culpritName, vName),
	}

	m := &cyMystery{
		BurgID:     id,
		Town:       town,
		Victim:     vName,
		Suspects:   sSet,
		CulpritIdx: culprit,
		Clues:      clues,
	}
	st.Mysteries[id] = m
	return m
}

// cyDiscoverClue records finding one of the 3 clues (0: physical trace, 1: witness statement, 2: motive note).
func (g *Game) cyDiscoverClue(burgID, clueIdx int) string {
	if g == nil {
		return ""
	}
	m := g.cyGetMystery(burgID)
	if m == nil {
		return ""
	}
	idx := clampi(clueIdx, 0, 2)
	already := m.FoundClues[idx]
	m.FoundClues[idx] = true
	clueText := m.Clues[idx]
	if !already {
		g.gainXP(15)
		g.Msg(fmt.Sprintf("{{C|[Mystery Clue Discovered] %s (+15 XP)}}", clueText))
	} else {
		g.Msg(fmt.Sprintf("{{c|[Mystery Clue] %s}}", clueText))
	}
	return clueText
}

func (m *cyMystery) cluesFoundCount() int {
	if m == nil {
		return 0
	}
	n := 0
	for _, f := range m.FoundClues {
		if f {
			n++
		}
	}
	return n
}

// cyAccuseSuspect presents an accusation before the town magistrate/elder.
// Correct accusation awards 120 gold, +35 town rep, +80 XP, and the "Consulting Detective" deed/title.
// Wrong accusation imposes -20 town rep and a 30 gold false-accusation fine.
func (g *Game) cyAccuseSuspect(burgID, suspectIdx int) (correct bool, msg string) {
	if g == nil || g.P == nil {
		return false, ""
	}
	id, town, fac := g.cyResolveBurgID(burgID)
	m := g.cyGetMystery(id)
	if m == nil {
		return false, ""
	}
	idx := clampi(suspectIdx, 0, 2)
	suspectName := m.Suspects[idx]
	m.Accused = true

	if idx == m.CulpritIdx {
		m.Solved = true
		g.P.Gold += 120
		g.AddRep(fac, 35, " for solving the murder of "+m.Victim)
		g.gainXP(80)
		g.cy().TitleEarned = true
		if g.P.Title == "" {
			g.P.Title = "Consulting Detective"
		}
		g.deed("detective", fmt.Sprintf("Consulting Detective: %s solved the murder of %s in %s and unmasked %s.", g.P.Name, m.Victim, town, suspectName), 0, 0)
		msg = fmt.Sprintf("Confronted with the evidence (%d/3 clues), %s confesses before the magistrate! You are awarded 120 %s, +35 reputation, +80 XP, and hailed as Consulting Detective!", m.cluesFoundCount(), suspectName, g.Sk.Currency)
		g.Msg("{{G|" + msg + "}}")
		return true, msg
	}

	// False accusation penalty
	fine := mini(30, maxi(0, g.P.Gold))
	g.P.Gold -= fine
	g.AddRep(fac, -20, " for a false murder accusation")
	msg = fmt.Sprintf("Your accusation against %s collapses under scrutiny! The magistrate fines you 30 %s and reprimands you (-20 reputation).", suspectName, g.Sk.Currency)
	g.Msg("{{R|" + msg + "}}")
	return false, msg
}

// ---- 2. City Jobs for Coin ----

type cyJobSpec struct {
	ID      string
	Title   string
	AttrA   int
	AttrB   int
	MinGold int
	MaxGold int
	RepGain int
	Desc    string
}

var cyJobSpecs = map[string]cyJobSpec{
	"porter": {
		ID:      "porter",
		Title:   "Docks & Market Porter",
		AttrA:   STR,
		AttrB:   STR,
		MinGold: 15,
		MaxGold: 30,
		RepGain: 0,
		Desc:    "You haul heavy crates and timber bales across the market wharf for four hours.",
	},
	"scribe": {
		ID:      "scribe",
		Title:   "Chancery Scribe",
		AttrA:   INT,
		AttrB:   INT,
		MinGold: 18,
		MaxGold: 35,
		RepGain: 0,
		Desc:    "You copy tax ledgers, charters, and shipping manifests by candle-light for four hours.",
	},
	"guard_shift": {
		ID:      "guard_shift",
		Title:   "Battlement Guard Shift",
		AttrA:   TOU,
		AttrB:   STR,
		MinGold: 20,
		MaxGold: 40,
		RepGain: 5,
		Desc:    "You walk the town battlements in full kit for a four-hour watch.",
	},
	"healer_aide": {
		ID:      "healer_aide",
		Title:   "Infirmary Healer's Aide",
		AttrA:   WIL,
		AttrB:   INT,
		MinGold: 18,
		MaxGold: 32,
		RepGain: 5,
		Desc:    "You boil linen bandages, compound poultices, and tend fevered townsfolk for four hours.",
	},
}

// cyWorkJob runs a 4-hour town work shift ("porter", "scribe", "guard_shift", "healer_aide").
func (g *Game) cyWorkJob(jobID string) (earned int, msg string) {
	if g == nil || g.P == nil {
		return 0, ""
	}
	spec, ok := cyJobSpecs[strings.ToLower(strings.TrimSpace(jobID))]
	if !ok {
		spec = cyJobSpecs["porter"]
	}
	st := g.cy()
	st.JobsWorked++

	// 4 hours pass (240 turns/minutes)
	g.Turn += 240

	modA := g.P.M(spec.AttrA)
	modB := g.P.M(spec.AttrB)
	mod := maxi(modA, modB)
	r := NewRng(fmt.Sprintf("%s:cy:job:%s:%d", g.Seed, spec.ID, st.JobsWorked))
	passed := r.Dice(2, 6)+mod >= dcEasy

	mid := (spec.MinGold + spec.MaxGold) / 2
	if passed {
		earned = clampi(r.Int(mid, spec.MaxGold)+maxi(0, mod), spec.MinGold, spec.MaxGold)
	} else {
		earned = clampi(r.Int(spec.MinGold, mid), spec.MinGold, spec.MaxGold)
	}

	g.P.Gold += earned
	g.gainXP(10)

	_, _, fac := g.cyResolveBurgID(0)
	if spec.RepGain > 0 {
		g.AddRep(fac, spec.RepGain, " for working a "+spec.Title+" shift")
	}

	repNote := ""
	if spec.RepGain > 0 {
		repNote = fmt.Sprintf(", +%d rep", spec.RepGain)
	}
	msg = fmt.Sprintf("%s You earn %d %s%s.", spec.Desc, earned, g.Sk.Currency, repNote)
	g.Msg("{{G|" + msg + "}}")
	return earned, msg
}

func (g *Game) cyWorkShift(jobID string) (earned int, msg string) {
	return g.cyWorkJob(jobID)
}

// ---- 3. Night-Life in Cities ----

// Hour returns the current hour of the day (0..23), where Turn 0 is 08:00.
func (g *Game) cyHour() int {
	if g == nil {
		return 8
	}
	h := (g.Turn/60 + 8) % 24
	if h < 0 {
		h += 24
	}
	return h
}

// cyIsNight reports whether the current game hour is in the night window (21:00..04:59).
func (g *Game) cyIsNight() bool {
	h := g.cyHour()
	return h >= 21 || h < 5
}

func isCyNightRole(role string) bool {
	return role == "cy_nightguild" || role == "cy_smuggler" || role == "cy_ghost"
}

// cyEnsureNightActors spawns night-life NPCs in town zones at night and removes them by day.
func (g *Game) cyEnsureNightActors(zn *Zone) {
	if g == nil {
		return
	}
	if zn == nil {
		zn = g.Cur
	}
	if zn == nil || zn.Kind != ZTown {
		return
	}

	if !g.cyIsNight() {
		// Remove peaceful night-life actors when day breaks
		keep := zn.Actors[:0]
		for _, a := range zn.Actors {
			if a != nil && isCyNightRole(a.Role) && !a.Angry && a.AI != "follower" {
				continue
			}
			keep = append(keep, a)
		}
		zn.Actors = keep
		return
	}

	hasRole := func(role string) bool {
		for _, a := range zn.Actors {
			if a != nil && a.HP > 0 && a.Role == role {
				return true
			}
		}
		return false
	}

	r := NewRng(fmt.Sprintf("%s:cy:night:%s:%d", g.Seed, zn.Key, g.Day()))

	// 1. Night-Guild Operative
	if !hasRole("cy_nightguild") {
		op := &Actor{
			ID:      g.newID(),
			Name:    "Night-Guild Operative",
			Proper:  false,
			Glyph:   '@',
			F:       CDarkMagenta,
			Tpl:     "bandit",
			Faction: "underworld",
			Level:   maxi(2, g.P.Level),
			Attr:    [6]int{14, 18, 14, 15, 14, 14},
			HP:      30,
			MaxHP:   30,
			AVNat:   3,
			DVNat:   5,
			Speed:   110,
			Body:    MakeBody("humanoid"),
			AI:      "town",
			Role:    "cy_nightguild",
			Species: "human",
			Gold:    120,
			Desc:    "A cloaked operative of the Night-Guild who deals in shadow contracts, smoke bombs, and lockpicks before dawn.",
			Flags:   map[string]bool{"night_npc": true},
		}
		smoke := g.MakeGrenade("flash", 2)
		smoke.Name = "night-guild smoke bomb"
		pick := &Item{ID: g.newID(), Kind: "tonic", Name: "whisper-steel lockpick set", Glyph: '~', F: CGray, Count: 2, Value: 25, Weight: 1, Effect: "lockpick"}
		op.Inv = []*Item{smoke, pick}
		if x, y, ok := zn.FreeSpot(r, nil); ok {
			op.X, op.Y = x, y
		} else {
			op.X, op.Y = clampi(g.P.X+2, 2, MW-3), clampi(g.P.Y, 2, MH-3)
		}
		op.HomeX, op.HomeY = op.X, op.Y
		zn.Actors = append(zn.Actors, op)
	}

	// 2. Lantern Smuggler
	if !hasRole("cy_smuggler") {
		sm := &Actor{
			ID:      g.newID(),
			Name:    "Lantern Smuggler",
			Proper:  false,
			Glyph:   '@',
			F:       CYellow,
			Tpl:     "merchant",
			Faction: "underworld",
			Level:   maxi(2, g.P.Level),
			Attr:    [6]int{13, 16, 14, 15, 14, 16},
			HP:      28,
			MaxHP:   28,
			AVNat:   2,
			DVNat:   4,
			Speed:   100,
			Body:    MakeBody("humanoid"),
			AI:      "town",
			Role:    "cy_smuggler",
			Species: "human",
			Gold:    250,
			Desc:    "A hooded factor carrying a shuttered brass lantern, offering off-world curios and contraband at midnight prices.",
			Flags:   map[string]bool{"night_npc": true},
		}
		offItem := g.MakeOddity(r, 2)
		offItem.Name = "smuggled astral shard"
		offItem.Value = maxi(15, offItem.Value/2)
		wyTagOffworldItem(offItem, g.PlanetIndex()+1)
		tonic := g.MakeTonic("heal", 2)
		tonic.Value = maxi(8, tonic.Value/2)
		sm.Inv = []*Item{offItem, tonic}
		if x, y, ok := zn.FreeSpot(r, nil); ok {
			sm.X, sm.Y = x, y
		} else {
			sm.X, sm.Y = clampi(g.P.X-2, 2, MW-3), clampi(g.P.Y, 2, MH-3)
		}
		sm.HomeX, sm.HomeY = sm.X, sm.Y
		zn.Actors = append(zn.Actors, sm)
	}

	// 3. Restless Town Ghost
	if !hasRole("cy_ghost") {
		gh := &Actor{
			ID:      g.newID(),
			Name:    "Restless Town Ghost",
			Proper:  false,
			Glyph:   'G',
			F:       CCyan,
			Tpl:     "wraith",
			Faction: zn.Faction,
			Level:   maxi(2, g.P.Level),
			Attr:    [6]int{10, 16, 12, 16, 18, 16},
			HP:      24,
			MaxHP:   24,
			AVNat:   2,
			DVNat:   6,
			Speed:   100,
			Body:    MakeBody("humanoid"),
			AI:      "town",
			Role:    "cy_ghost",
			Species: "spirit",
			Desc:    "A translucent shade drifting along the cobblestones by moonlight, murmuring of unsolved murders and buried silver.",
			Flags:   map[string]bool{"undead": true, "night_npc": true},
		}
		if x, y, ok := zn.FreeSpot(r, nil); ok {
			gh.X, gh.Y = x, y
		} else {
			gh.X, gh.Y = clampi(g.P.X, 2, MW-3), clampi(g.P.Y+2, 2, MH-3)
		}
		gh.HomeX, gh.HomeY = gh.X, gh.Y
		zn.Actors = append(zn.Actors, gh)
	}
}

// ---- Talk Hooks for Mysteries, Jobs & Night-Life ----

func cyTalkOptions(g *Game, a *Actor) []Option {
	if g == nil || a == nil {
		return nil
	}
	var opts []Option
	burgID, _, _ := g.cyResolveBurgID(0)

	// Night-Guild Operative dialogue
	if a.Role == "cy_nightguild" {
		opts = append(opts, Option{
			Text: "{{m|Take a Night-Guild shadow contract.}}",
			Do: func() {
				if g.cy().ShadowDone[a.ID] {
					if g.Talk == nil {
						g.Talk = a
					}
					g.talkMenu("\"You've already run tonight's drop. Keep to the shadows.\"")
					return
				}
				g.cy().ShadowDone[a.ID] = true
				g.P.Gold += 45
				g.gainXP(30)
				clue := g.cyDiscoverClue(burgID, 2)
				if g.Talk == nil {
					g.Talk = a
				}
				g.talkMenu(fmt.Sprintf("You slip a coded cipher beneath a shuttered manor door and return. The operative pays you 45 %s and shares an intercepted note:\n\n%s", g.Sk.Currency, clue))
			},
		})
	}

	// Lantern Smuggler dialogue
	if a.Role == "cy_smuggler" {
		opts = append(opts, Option{
			Text: "{{Y|Inspect smuggled off-world contraband.}}",
			Do:   func() { g.openTrade(a) },
		})
	}

	// Restless Town Ghost dialogue
	if a.Role == "cy_ghost" {
		opts = append(opts, Option{
			Text: "{{C|Listen to the ghost's hollow whisper.}}",
			Do: func() {
				m := g.cyGetMystery(burgID)
				nextClue := 0
				for i := 0; i < 3; i++ {
					if !m.FoundClues[i] {
						nextClue = i
						break
					}
				}
				clue := g.cyDiscoverClue(burgID, nextClue)
				if g.Talk == nil {
					g.Talk = a
				}
				g.talkMenu(fmt.Sprintf("\"I saw what was done in the dark...\" The spectre points a frost-white finger:\n\n%s", clue))
			},
		})
	}

	// Town Mystery investigation & Courtroom Accusation on elders, notables, guards
	if a.Role == "elder" || a.Role == "notable" || a.Role == "guard" || a.Role == "scholar" {
		m := g.cyGetMystery(burgID)
		if m != nil && !m.Solved {
			opts = append(opts, Option{
				Text: fmt.Sprintf("{{C|Investigate the murder of %s.}}", m.Victim),
				Do: func() {
					idx := m.cluesFoundCount() % 3
					clue := g.cyDiscoverClue(burgID, idx)
					if g.Talk == nil {
						g.Talk = a
					}
					g.talkMenu(fmt.Sprintf("You question witnesses and examine the scene of %s's murder:\n\n%s", m.Victim, clue))
				},
			})
			if a.Role == "elder" || a.Role == "notable" {
				for si := 0; si < 3; si++ {
					si := si
					sName := m.Suspects[si]
					opts = append(opts, Option{
						Text: fmt.Sprintf("{{Y|[Courtroom Accusation] Accuse %s of murdering %s.}}", sName, m.Victim),
						Do: func() {
							_, msg := g.cyAccuseSuspect(burgID, si)
							if g.Talk == nil {
								g.Talk = a
							}
							g.talkMenu(msg)
						},
					})
				}
			}
		}
	}

	// City Jobs for Coin
	switch a.Role {
	case "merchant", "ferry":
		opts = append(opts, Option{
			Text: "{{w|Work a 4-hour shift as a Porter (STR, 15-30 gold).}}",
			Do: func() {
				_, msg := g.cyWorkJob("porter")
				if g.Talk == nil {
					g.Talk = a
				}
				g.talkMenu(msg)
			},
		})
	case "scholar", "bookseller", "elder":
		opts = append(opts, Option{
			Text: "{{w|Work a 4-hour shift as a Scribe (INT, 18-35 gold).}}",
			Do: func() {
				_, msg := g.cyWorkJob("scribe")
				if g.Talk == nil {
					g.Talk = a
				}
				g.talkMenu(msg)
			},
		})
	case "guard":
		opts = append(opts, Option{
			Text: "{{w|Stand a 4-hour Battlement Guard Shift (TOU/STR, 20-40 gold, +5 rep).}}",
			Do: func() {
				_, msg := g.cyWorkJob("guard_shift")
				if g.Talk == nil {
					g.Talk = a
				}
				g.talkMenu(msg)
			},
		})
	case "priest", "apothecary":
		opts = append(opts, Option{
			Text: "{{w|Serve a 4-hour shift as Healer's Aide (WIL/INT, 18-32 gold, +5 rep).}}",
			Do: func() {
				_, msg := g.cyWorkJob("healer_aide")
				if g.Talk == nil {
					g.Talk = a
				}
				g.talkMenu(msg)
			},
		})
	}

	return opts
}
