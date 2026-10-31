package engine

import "math"

// Region tiers 1-8 (prefix qc): terrain danger and depth raise the level of hostile wildlife.

var qcBiomeDanger = [17]int{1, 3, 3, 1, 0, 2, 1, 2, 1, 2, 3, 4, 2, 4, 4, 4, 6}

func init() {
	zoneHooks = append(zoneHooks, qcTierZone)
}

// qcTier = clamp(1 + terrain danger/2 + |z|/2, 1, 8); danger = biome danger + distance from the start / 8.
func (g *Game) qcTier(wx, wy, z int) int {
	w := g.W
	wx, wy = clampi(wx, 0, w.W-1), clampi(wy, 0, w.H-1)
	b := w.biomeAt(w.Idx(wx, wy))
	d := int(math.Hypot(float64(wx-g.StartX), float64(wy-g.StartY)))
	danger := qcBiomeDanger[b] + d/8
	depth := abs(z)
	if z <= towerBase {
		depth = towerBase - z + 1
	}
	return clampi(1+danger/2+depth/2, 1, 8)
}

func qcTierLevel(tier int) int { return tier*3 - 2 }

// qcRescale lifts an already-spawned creature to level lvl the way Spawn scales one.
func qcRescale(a *Actor, lvl int) {
	up := lvl - a.Level
	if up <= 0 {
		return
	}
	a.Level = lvl
	for i := range a.Attr {
		a.Attr[i] += up / 2
	}
	a.MaxHP += up * 4
	a.HP = a.MaxHP
	a.AVNat += up / 3
	for _, p := range a.Body {
		if p.Natural != nil && p.Natural.DN > 0 {
			p.Natural.DS += up / 3
			p.Natural.PV += up / 4
		}
	}
	if xp := 10 + lvl*lvl*5; a.XP < xp {
		a.XP = xp
	}
}

func qcTierZone(g *Game, zn *Zone, r *Rng) {
	if zn.Kind == "town" {
		return
	}
	lvl := qcTierLevel(g.qcTier(zn.WX, zn.WY, zn.Z))
	for _, a := range zn.Actors {
		if a.AI != "hostile" || a.Boss || a.Role != "" || a.SimMon > 0 || a.SimPer > 0 || a.Quest > 0 || a.Comp != nil || a.Faction == "player" || len(a.Faction) > 6 && a.Faction[:6] == "state:" {
			continue
		}
		if a.Level < lvl {
			qcRescale(a, lvl)
		}
	}
}
