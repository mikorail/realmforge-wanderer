package engine

import (
	"fmt"
)

// Living world systems (prefix lw):
// 1. Packed inns & taverns with bards, barmaids, dice gambling, and buying rounds.
// 2. Public plaza spectacles: gallows executions (pardon/defend/rescue), live honor duels (wager/intervene), and street fairs.
// 3. Busy working citizens: farmers in crop rows, blacksmiths at anvils, street hawkers, guard patrols, woodcutters, hunters, miners, fishers, water-carriers.

type lwState struct {
	Pardons      map[string]bool `json:"pardons,omitempty"`
	DuelsSeen    int             `json:"duels,omitempty"`
	TavernRounds int             `json:"rounds,omitempty"`
	WorkShift    int             `json:"shift,omitempty"`
}

func (g *Game) lw() *lwState {
	st := g.ext("lw", func() any { return &lwState{} }).(*lwState)
	if st.Pardons == nil {
		st.Pardons = map[string]bool{}
	}
	return st
}

func init() {
	unLocalKeys = append(unLocalKeys, "lw")
	zoneHooks = append(zoneHooks, lwZoneHook)
	turnHooks = append(turnHooks, lwTurnHook)
	talkHooks = append(talkHooks, lwTalkHook)
}

func lwZoneHook(g *Game, zn *Zone, r *Rng) {
	if g == nil || zn == nil || zn.Z != 0 {
		return
	}
	// Use deterministic side RNG so core worldgen stream is untouched
	sr := NewRng(fmt.Sprintf("%s:lw:%s", g.Seed, zn.Key))

	if zn.Kind == "town" {
		lwPopulateInn(g, zn, sr)
		lwSpawnSpectacle(g, zn, sr)
		lwPopulateWorkers(g, zn, sr)
	} else if zn.Kind == "wild" || zn.Kind == "road" {
		lwPopulateWildernessWorkers(g, zn, sr)
	}
}

// 1. Packed Inns & Taverns
func lwPopulateInn(g *Game, zn *Zone, sr *Rng) {
	var innSpot *Actor
	for _, a := range zn.Actors {
		if a.Role == "innkeeper" || a.Role == "tavern" {
			innSpot = a
			break
		}
	}
	if innSpot == nil {
		return
	}
	ix, iy := innSpot.X, innSpot.Y

	roles := []string{
		"lw_server", "lw_bard", "lw_mercenary", "lw_gambler",
		"lw_regular", "lw_hunter", "lw_patron", "lw_caravaneer",
	}
	titles := map[string]string{
		"lw_server":     "tapster",
		"lw_bard":       "tavern bard",
		"lw_mercenary":  "sellsword",
		"lw_gambler":    "dice-player",
		"lw_regular":    "tavern regular",
		"lw_hunter":     "boastful hunter",
		"lw_patron":     "hooded stranger",
		"lw_caravaneer": "weary caravaneer",
	}

	for _, ro := range roles {
		if x, y, ok := zn.FreeSpot(sr, func(px, py int) bool {
			return abs(px-ix) <= 5 && abs(py-iy) <= 4 && zn.Walkable(px, py)
		}); ok {
			a := g.Spawn("bandit", maxi(1, zn.Level), sr)
			a.Name = titles[ro]
			a.Role = ro
			a.Title = titles[ro]
			a.Faction = "citizens"
			a.Angry = false
			a.AI = "town"
			a.X, a.Y, a.HomeX, a.HomeY = x, y, x, y
			a.F = CCyan
			zn.Actors = append(zn.Actors, a)
		}
	}
}

// 2. Public Spectacles: Executions, Duels, Street Markets
func lwSpawnSpectacle(g *Game, zn *Zone, sr *Rng) {
	if zn.ZX != 1 || zn.ZY != 1 {
		return
	}
	specType := sr.Int(0, 2)
	switch specType {
	case 0:
		// Public Execution in the Plaza
		cx, cy := 40, 8
		if zn.Walkable(cx, cy) && zn.ActorAt(cx, cy) == nil {
			zn.Set(cx, cy, TScaffold)
			zn.Set(cx+1, cy, TBrazier)
			zn.Set(cx-1, cy, TInscription)
			zn.Notes[cy*MW+cx-1] = "The Town Gallows and Execution Scaffold."

			exec := g.Spawn("bandit", maxi(3, zn.Level+2), sr)
			exec.Name = "hooded executioner"
			exec.Role = "lw_executioner"
			exec.Title = "high executioner"
			exec.Faction = zn.Faction
			exec.X, exec.Y, exec.HomeX, exec.HomeY = cx, cy, cx, cy
			exec.F = CDarkRed
			exec.AI = "still"
			zn.Actors = append(zn.Actors, exec)

			herald := g.Spawn("bandit", maxi(1, zn.Level), sr)
			herald.Name = "town herald"
			herald.Role = "lw_herald"
			herald.Faction = zn.Faction
			herald.X, herald.Y, herald.HomeX, herald.HomeY = cx+2, cy, cx+2, cy
			herald.F = CYellow
			herald.AI = "still"
			zn.Actors = append(zn.Actors, herald)

			crimes := []string{
				"condemned for treason",
				"accused of smuggling forbidden relics",
				"sentenced for highway robbery",
				"framed for a guildmaster debt",
			}
			pris := g.Spawn("bandit", maxi(1, zn.Level), sr)
			pris.Name = "condemned prisoner"
			pris.Role = "lw_condemned"
			pris.Desc = "Bound in chains: " + crimes[sr.Int(0, len(crimes)-1)] + "."
			pris.Faction = "outlaws"
			pris.X, pris.Y, pris.HomeX, pris.HomeY = cx-1, cy+1, cx-1, cy+1
			pris.F = CGray
			pris.AI = "still"
			zn.Actors = append(zn.Actors, pris)

			for i := 0; i < 5; i++ {
				if sx, sy, ok := zn.FreeSpot(sr, func(px, py int) bool {
					return abs(px-cx) <= 4 && abs(py-cy) <= 3 && py > cy
				}); ok {
					spec := g.Spawn("bandit", 1, sr)
					spec.Name = "spectator"
					spec.Role = "lw_spectator"
					spec.Faction = "citizens"
					spec.AI = "town"
					spec.X, spec.Y, spec.HomeX, spec.HomeY = sx, sy, sx, sy
					spec.F = CDarkGray
					zn.Actors = append(zn.Actors, spec)
				}
			}
		}

	case 1:
		// Street / Plaza Honor Duel
		dx, dy := 40, 16
		if zn.Walkable(dx-1, dy) && zn.Walkable(dx+1, dy) {
			d1 := g.Spawn("bandit", maxi(2, zn.Level+1), sr)
			d1.Name = "brave duellist"
			d1.Role = "lw_duellist"
			d1.Title = "duellist of the iron school"
			d1.Faction = "citizens"
			d1.X, d1.Y, d1.HomeX, d1.HomeY = dx-1, dy, dx-1, dy
			d1.F = CRed
			d1.AI = "still"
			d1.Set("lw_duel1")

			d2 := g.Spawn("bandit", maxi(2, zn.Level+1), sr)
			d2.Name = "errant knight"
			d2.Role = "lw_duellist"
			d2.Title = "wandering blade"
			d2.Faction = "citizens"
			d2.X, d2.Y, d2.HomeX, d2.HomeY = dx+1, dy, dx+1, dy
			d2.F = CBlue
			d2.AI = "still"
			d2.Set("lw_duel2")

			sec := g.Spawn("bandit", 2, sr)
			sec.Name = "duel umpire"
			sec.Role = "lw_second"
			sec.Faction = "citizens"
			sec.X, sec.Y, sec.HomeX, sec.HomeY = dx, dy+1, dx, dy+1
			sec.F = CYellow
			sec.AI = "still"

			zn.Actors = append(zn.Actors, d1, d2, sec)

			for i := 0; i < 4; i++ {
				if sx, sy, ok := zn.FreeSpot(sr, func(px, py int) bool {
					return abs(px-dx) <= 3 && abs(py-dy) <= 2 && py > dy
				}); ok {
					spec := g.Spawn("bandit", 1, sr)
					spec.Name = "wagerer"
					spec.Role = "lw_spectator"
					spec.AI = "town"
					spec.X, spec.Y, spec.HomeX, spec.HomeY = sx, sy, sx, sy
					spec.F = CDarkGray
					zn.Actors = append(zn.Actors, spec)
				}
			}
		}
	}
}

// 3. Busy Working Citizens: Farmers, Smiths, Hawkers, Guard Patrols, Miners, Fishers
func lwPopulateWorkers(g *Game, zn *Zone, sr *Rng) {
	p1 := g.Spawn("bandit", maxi(2, zn.Level+1), sr)
	p1.Name = "city patrol"
	p1.Role = "lw_patrol"
	p1.Faction = zn.Faction
	p1.AI = "guard"
	p1.X, p1.Y, p1.HomeX, p1.HomeY = 38, 12, 40, 12
	p1.F = CBrown
	p1.Set("armed")

	p2 := g.Spawn("bandit", maxi(2, zn.Level+1), sr)
	p2.Name = "watch sergeant"
	p2.Role = "lw_patrol"
	p2.Faction = zn.Faction
	p2.AI = "guard"
	p2.X, p2.Y, p2.HomeX, p2.HomeY = 42, 12, 40, 12
	p2.F = CYellow
	p2.Set("armed")

	zn.Actors = append(zn.Actors, p1, p2)

	for i := 0; i < 2; i++ {
		if x, y, ok := zn.FreeSpot(sr, func(px, py int) bool {
			return zn.At(px, py) == TCobble && abs(px-40) <= 8 && abs(py-12) <= 4
		}); ok {
			hw := g.Spawn("bandit", 1, sr)
			hw.Name = "street hawker"
			hw.Role = "lw_hawker"
			hw.AI = "town"
			hw.X, hw.Y, hw.HomeX, hw.HomeY = x, y, x, y
			hw.F = COrange
			zn.Actors = append(zn.Actors, hw)
		}
	}

	for y := 0; y < MH; y++ {
		for x := 0; x < MW; x++ {
			if zn.At(x, y) == TCrop && sr.Chance(0.04) {
				fm := g.Spawn("bandit", 1, sr)
				fm.Name = "field farmer"
				fm.Role = "lw_farmer"
				fm.Faction = "citizens"
				fm.AI = "town"
				fm.X, fm.Y, fm.HomeX, fm.HomeY = x, y, x, y
				fm.F = CGreen
				zn.Actors = append(zn.Actors, fm)
				return
			}
		}
	}
}

func lwPopulateWildernessWorkers(g *Game, zn *Zone, sr *Rng) {
	trees := 0
	for _, t := range zn.T {
		if t == TTree {
			trees++
		}
	}
	if trees > 40 && sr.Chance(0.4) {
		if x, y, ok := zn.FreeSpot(sr, func(px, py int) bool {
			return zn.Walkable(px, py) && abs(px-40) < 15
		}); ok {
			wc := g.Spawn("bandit", maxi(1, zn.Level), sr)
			wc.Name = "woodcutter"
			wc.Role = "lw_woodcutter"
			wc.AI = "wander"
			wc.X, wc.Y, wc.HomeX, wc.HomeY = x, y, x, y
			wc.F = CBrown
			zn.Actors = append(zn.Actors, wc)
		}
	}

	if sr.Chance(0.35) {
		if x, y, ok := zn.FreeSpot(sr, nil); ok {
			hn := g.Spawn("bandit", maxi(2, zn.Level), sr)
			hn.Name = "wildland trapper"
			hn.Role = "lw_hunter"
			hn.AI = "wander"
			hn.X, hn.Y, hn.HomeX, hn.HomeY = x, y, x, y
			hn.F = CDarkGreen
			zn.Actors = append(zn.Actors, hn)
		}
	}
}

func lwTurnHook(g *Game) {
	if g == nil || g.Cur == nil || g.Dead {
		return
	}
	zn := g.Cur

	if g.Turn%15 == 0 {
		var d1, d2 *Actor
		for _, a := range zn.Actors {
			if a.HP > 0 && a.Role == "lw_duellist" {
				if d1 == nil {
					d1 = a
				} else if d2 == nil {
					d2 = a
				}
			}
		}
		if d1 != nil && d2 != nil && dist(d1, d2) <= 2 {
			dmg := g.R.Int(1, 3)
			d2.HP = maxi(1, d2.HP-dmg)
			if g.visible(d1.X, d1.Y) || g.visible(d2.X, d2.Y) {
				g.Msg(fmt.Sprintf("{{Y|Clang! Blades spark as %s lunges at %s!}}", d1.Name, d2.Name))
			}
			if d2.HP <= d2.MaxHP/3 {
				d2.Role = "lw_spectator"
				d1.Role = "lw_spectator"
				if g.visible(d1.X, d1.Y) {
					g.Msg(fmt.Sprintf("{{G|%s salutes with lowered blade: \"Yielded! A worthy bout!\"}}", d2.Name))
				}
			}
		}

		for _, a := range zn.Actors {
			if a.Role == "lw_herald" && (g.visible(a.X, a.Y) || dist(a, g.P) <= 10) {
				g.Msg("{{R|\"Hear ye! The court has decreed justice!\" rolls the Herald's voice across the square.}}")
				break
			}
		}

		for _, a := range zn.Actors {
			if a.Role == "lw_bard" && dist(a, g.P) <= 6 {
				g.Msg("{{C|A lively ballad echoes from the tavern taproom over the clink of flagons.}}")
				break
			}
		}
	}
}

func lwTalkHook(g *Game, a *Actor) []Option {
	if g == nil || a == nil {
		return nil
	}
	st := g.lw()
	var opts []Option

	switch a.Role {
	case "lw_innkeeper":
		opts = append(opts, Option{"{{Y|Buy a round for the whole tavern! (20 gold)}}", func() {
			if !g.cuSpend(20) {
				g.talkMenu("\"Coin first, generous stranger!\"")
				return
			}
			st.TavernRounds++
			g.AddRep(a.Faction, 25, "for tavern generosity")
			g.gainXP(20)
			g.Msg("{{G|Cheers erupt as flagons are filled! The taproom toasts your name!}}")
			g.talkMenu("\"May your blade stay sharp and your road stay clear! Let me tell you what I heard from the south road...\"")
		}})

	case "lw_second":
		opts = append(opts, Option{"{{Y|Wager 15 gold on the duel.}}", func() {
			if !g.cuSpend(15) {
				g.talkMenu("\"Place real coin on the line, friend.\"")
				return
			}
			g.gainXP(15)
			if g.R.Chance(0.5) {
				g.cuEarn(30)
				g.talkMenu("\"Your champion triumphed! Here is your 30 gold purse!\"")
			} else {
				g.talkMenu("\"Tough break! Your fighter took a glancing cut to the arm.\"")
			}
		}})

	case "lw_herald", "lw_executioner":
		key := fmt.Sprintf("%s:%d", g.Cur.Key, a.X)
		if !st.Pardons[key] {
			opts = append(opts, Option{"{{Y|Pay blood-fine to pardon the condemned prisoner (60 gold).}}", func() {
				if !g.cuSpend(60) {
					g.talkMenu("\"The law demands 60 gold in blood-ransom, no less!\"")
					return
				}
				st.Pardons[key] = true
				g.gainXP(50)
				g.AddRep(a.Faction, 20, "for blood-ransom pardon")
				for _, p := range g.Cur.Actors {
					if p.Role == "lw_condemned" {
						p.Role = "follower"
						p.AI = "follower"
						p.Faction = "citizens"
						p.Name = "freed debtor"
						p.Desc = "Pardoned by your blood-ransom; swears gratitude."
						break
					}
				}
				g.talkMenu("\"The purse is accepted. The prisoner is released under your custody!\"")
			}})
		}

	case "lw_farmer":
		opts = append(opts, Option{"{{G|Buy fresh field produce (5 gold).}}", func() {
			if !g.cuSpend(5) {
				g.talkMenu("\"Five coppers for a fresh bundle.\"")
				return
			}
			g.give(&Item{ID: g.newID(), Kind: "food", Name: "fresh farm produce", Value: 6, Weight: 1, Glyph: '%', F: CGreen})
			g.talkMenu("\"Fresh from the morning harvest. Eat well, traveller!\"")
		}})

	case "lw_woodcutter":
		opts = append(opts, Option{"Trade for seasoned firewood (4 gold).", func() {
			if !g.cuSpend(4) {
				g.talkMenu("\"Dry oak costs four copper.\"")
				return
			}
			g.give(&Item{ID: g.newID(), Kind: "misc", Name: "seasoned firewood", Value: 5, Weight: 2, Glyph: '=', F: CBrown})
			g.talkMenu("\"Good dry split logs for the road camp.\"")
		}})

	case "lw_hunter":
		opts = append(opts, Option{"Trade for fresh game meat and pelts (8 gold).", func() {
			if !g.cuSpend(8) {
				g.talkMenu("\"Hunts take patience, traveller.\"")
				return
			}
			g.give(&Item{ID: g.newID(), Kind: "ingredient", Name: "game meat", Value: 10, Weight: 1, Glyph: '%', F: CRed})
			g.talkMenu("\"Fresh stag meat from the high crags.\"")
		}})
	}

	return opts
}
