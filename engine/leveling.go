package engine

import "fmt"

// Leveling (WP4). Qud total XP to reach level x is 15x^3+100; Wanderer's gainXP subtracts a per-level step,
// so XPForLevel(l) is the step from l to l+1 on that curve, scaled down by skXPScale to match our kill XP.
const skXPScale = 2

func skXPTotal(l int) int { return 15*l*l*l + 100 }

// XPForLevel is the XP needed to go from level l to l+1.
func XPForLevel(l int) int {
	if l < 1 {
		l = 1
	}
	return (skXPTotal(l+1) - skXPTotal(l)) / skXPScale
}

// killXPTier scales XP by the tier gap (tier = level/5): equal or higher victim x1, one below x0.5, two x0.1, more 0.
func skKillXP(g *Game, d, src *Actor) int {
	lv := g.P.Level
	if src != nil && src != g.P && src.Level > 0 {
		lv = src.Level
	}
	gap := lv/5 - d.Level/5
	switch {
	case gap <= 0:
		return d.XP
	case gap == 1:
		return d.XP / 2
	case gap == 2:
		return d.XP / 10
	}
	return 0
}

// killXP is what kill() should award: g.gainXP(g.killXP(d, src)).
func (g *Game) killXP(d, src *Actor) int { return skKillXP(g, d, src) }

type skLevelState struct {
	Done int             `json:"done,omitempty"` // highest level already processed by skOnLevel
	Seen map[string]bool `json:"seen,omitempty"` // zones that already paid discovery XP
}

func (g *Game) skLevels() *skLevelState {
	return g.ext("skills_lv", func() any { return &skLevelState{Seen: map[string]bool{}} }).(*skLevelState)
}

// skSPFor: 2 SP per level, +1 per 2 Intelligence above 10.
func skSPFor(intel int) int { return 2 + maxi(0, (intel-10)/2) }

// skOnLevel grants level-up points for every level not yet processed. Call it right after Level++ in gainXP
// (it replaces the SP/MP/AP lines there). SP by INT; MP for mutants only; attribute cadence every 3 levels:
// n = level/3, odd n gives 1 AP, even n gives +1 to every attribute.
func (g *Game) skOnLevel() {
	st := g.skLevels()
	if st.Done == 0 || st.Done >= g.P.Level {
		st.Done = g.P.Level - 1
	}
	for l := st.Done + 1; l <= g.P.Level; l++ {
		sp := skSPFor(g.P.Score(INT))
		g.SP += sp
		if g.Genotype != "truekin" {
			g.MP++
		}
		note := fmt.Sprintf("+%d SP", sp)
		if l%3 == 0 {
			if n := l / 3; n%2 == 1 {
				g.AP++
				note += ", +1 AP"
			} else {
				for i := range g.P.Attr {
					g.P.Attr[i]++
				}
				g.P.MaxHP += l
				g.P.HP += l
				note += ", +1 to every attribute"
			}
		}
		g.Msg("{{K|Level " + fmt.Sprint(l) + ": " + note + ". (K: skills)}}")
	}
	st.Done = g.P.Level
}

// skDiscover pays XP the first time the player enters a town, ruin or lair zone: 250 x tier / 25.
func (g *Game) skDiscover(zn *Zone) {
	if zn == nil || g.P == nil {
		return
	}
	switch zn.Kind {
	case "town", "ruin", "lair", "crypt", "dungeon", "wonder":
	default:
		return
	}
	st := g.skLevels()
	if st.Done == 0 {
		st.Done = g.P.Level // first entry of a new game: nothing to catch up on
	}
	if st.Seen[zn.Key] {
		return
	}
	st.Seen[zn.Key] = true
	if len(st.Seen) == 1 {
		return // the starting settlement does not count
	}
	tier := 1 + maxi(zn.Z, 0)/2 + g.P.Level/5
	xp := 250 * tier / 25
	g.Msg(fmt.Sprintf("{{K|New place: +%d XP.}}", xp))
	g.gainXP(xp)
}
