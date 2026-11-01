package engine

import (
	"fmt"
	"strings"
)

// Slain beasts (prefix sl). A sim monster exported with slain=true never spawns: its lair holds
// its remains (a trophy relic and an inscription naming the year and the slayer) and, for about
// half of them (deterministic per monster), a "Brood of <name>" at ~60% of the parent's might.
// placeSimMonster (zone.go) hands slain monsters here; bounties, quests and secrets skip them.

func init() {
	killHooks = append(killHooks, slKill)
}

// slKill: a dead brood ends its parent's line on this planet (no more brood bounties).
func slKill(g *Game, d, _ *Actor) {
	if !d.Has("brood") {
		return
	}
	for i := range g.W.Monsters {
		if m := &g.W.Monsters[i]; m.Slain && d.Name == "Brood of "+m.Name {
			g.Slain[m.ID] = true
		}
	}
}

// slBroodSpawns: the brood really appears (its lair is open country, not a town or the sea).
func slBroodSpawns(w *WorldData, m *MonsterData) bool {
	return w.slHasBrood(m) && m.Layer == "surface" && m.P >= 0 && m.P < len(w.Burg) && w.Burg[m.P] == 0 && !w.IsWater(m.P)
}

// slHasBrood: does this slain monster leave offspring in its lair?
func (w *WorldData) slHasBrood(m *MonsterData) bool {
	if !m.Slain || m.Immortal {
		return false
	}
	return NewRng(fmt.Sprintf("%s:brood:%d", w.Seed, m.ID)).Chance(0.5)
}

func slDeath(m *MonsterData) string {
	s := "slain"
	if m.SlainYear > 0 {
		s += fmt.Sprintf(" in the year %d", m.SlainYear)
	}
	if m.SlainBy != "" {
		s += " by " + m.SlainBy
	}
	return s
}

// slSecret is the rumour told of a slain beast instead of "X lairs at Y".
func (g *Game) slSecret(m *MonsterData) string {
	where := nonEmpty(m.Lair, g.W.RegionName(m.P))
	s := fmt.Sprintf("%s was %s. Its bones still lie at %s.", m.Name, slDeath(m), where)
	if g.W.slHasBrood(m) {
		s += " They say its brood still nests there."
	}
	return s
}

func (g *Game) slPlaceRemains(zn *Zone, r *Rng, m *MonsterData) {
	x, y, ok := zn.FreeSpot(r, func(x, y int) bool { return abs(x-40) < 20 && abs(y-12) < 8 })
	if !ok {
		return
	}
	for dy := -3; dy <= 3; dy++ {
		for dx := -6; dx <= 6; dx++ {
			if r.Chance(0.3) && zn.Walkable(x+dx, y+dy) && zn.At(x+dx, y+dy).D().Water == 0 {
				zn.Set(x+dx, y+dy, TBones)
			}
		}
	}
	lore := fmt.Sprintf("The remains of %s, %s.", m.Name, slDeath(m))
	if m.Lore != "" {
		lore += " " + m.Lore
	}
	name := "Bones of " + m.Name
	if m.Trophy != "" {
		name = m.Trophy + " (remains)"
	}
	it := &Item{ID: g.newID(), Kind: "trophy", Name: name, Glyph: '♦', F: CDarkGray, Value: 80 + m.Power*2, Lore: lore, X: x, Y: y}
	zn.Items = append(zn.Items, it)
	// a stone over the bones
	for _, d := range [][2]int{{0, -1}, {1, 0}, {-1, 0}, {0, 1}} {
		sx, sy := x+d[0], y+d[1]
		if zn.Walkable(sx, sy) && zn.ActorAt(sx, sy) == nil {
			zn.Set(sx, sy, TInscription)
			zn.Notes[sy*MW+sx] = fmt.Sprintf("Here fell {{R|%s}}, %s.", m.Name, slDeath(m))
			break
		}
	}
	if zn.Kind == "town" || zn.Kind == "ruin" || zn.Burg > 0 {
		return // the town grew over the old lair: only the bones and the stone remain
	}
	if m.Lair != "" {
		zn.Name = m.Lair
	} else {
		zn.Name = "Lair of " + m.Name
	}
	zn.Kind = "lair"
	if g.W.slHasBrood(m) {
		if a := g.slBrood(zn, r, m); a != nil {
			zn.Name += " (brood)"
		}
	}
}

// slBrood spawns the offspring: same shape as the parent, about 60% of its might.
func (g *Game) slBrood(zn *Zone, r *Rng, m *MonsterData) *Actor {
	x, y, ok := zn.FreeSpot(r, func(x, y int) bool { return abs(x-40) < 22 && abs(y-12) < 9 })
	if !ok {
		return nil
	}
	lvl := clampi(m.Power*6/70, 5, 24)
	a := g.Spawn("horror", lvl, r)
	a.Name, a.Proper, a.Boss = "Brood of "+m.Name, true, true
	a.Desc = fmt.Sprintf("Offspring of %s, which was %s. It has its parent's shape and half its fury.", m.Name, slDeath(m))
	if sp := []rune(strings.ToLower(m.Species)); len(sp) > 0 {
		a.Glyph = sp[0]
	}
	a.F = CDarkRed
	a.MaxHP = maxi(a.MaxHP, m.HP*6/10)
	a.HP = a.MaxHP
	a.AVNat += int(m.Armor * 14 * 0.6)
	a.Faction = "legends"
	a.XP = (400 + m.Power*20) * 6 / 10
	a.Gold = (100 + m.Power*3) * 6 / 10
	a.X, a.Y = x, y
	a.Set("brood")
	zn.Actors = append(zn.Actors, a)
	return a
}
