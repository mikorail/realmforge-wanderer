package engine

import (
	"fmt"
)

// Interactive Tactical Environment (prefix te):
// 1. Dynamic tavern brawls: flipping tables for cover, swinging from chandeliers, throwing tankards, and kicking foes into braziers.
// 2. Environmental combat hazards: cutting scaffold ropes to crush enemies, shooting beehives, and puncturing wine/oil barrels.

type teState struct {
	FlippedTables map[string]bool `json:"flipped,omitempty"`
	PuncturedCasks map[string]bool `json:"casks,omitempty"`
}

func (g *Game) te() *teState {
	st := g.ext("te", func() any { return &teState{} }).(*teState)
	if st.FlippedTables == nil {
		st.FlippedTables = map[string]bool{}
	}
	if st.PuncturedCasks == nil {
		st.PuncturedCasks = map[string]bool{}
	}
	return st
}

func init() {
	unLocalKeys = append(unLocalKeys, "te")
	zoneHooks = append(zoneHooks, teZoneHook)
}

func teZoneHook(g *Game, zn *Zone, r *Rng) {
	if g == nil || zn == nil {
		return
	}
	sr := NewRng(fmt.Sprintf("%s:te:%s", g.Seed, zn.Key))

	// 1. In towns and taverns, place chandeliers, barrels, and braziers
	if zn.Kind == "town" && zn.Z == 0 {
		tePopulateTownTactics(g, zn, sr)
	}

	// 2. In wild and forest zones, place wild beehives hanging from trees
	if zn.Kind == "wild" || zn.Kind == "forest" {
		tePopulateWildHazards(g, zn, sr)
	}
}

func tePopulateTownTactics(g *Game, zn *Zone, sr *Rng) {
	// Hang chandeliers over tavern common rooms
	chandeliers := [][2]int{{24, 7}, {28, 7}, {40, 8}}
	for _, pt := range chandeliers {
		cx, cy := pt[0], pt[1]
		if zn.In(cx, cy) && zn.At(cx, cy).D().Walk {
			zn.Set(cx, cy, TChandelier)
			zn.Notes[cy*MW+cx] = "An iron chandelier suspended by chains. Bump to leap and swing across the room!"
		}
	}

	// Place oil & wine barrels in cellars, tavern corners, and docks
	barrels := [][2]int{{22, 5}, {29, 9}, {15, 14}, {65, 12}}
	for _, pt := range barrels {
		bx, by := pt[0], pt[1]
		if zn.In(bx, by) && zn.At(bx, by).D().Walk {
			zn.Set(bx, by, TBarrel)
			zn.Notes[by*MW+bx] = "A heavy oak cask of oil and wine. Bump or strike to puncture and flood the floor."
		}
	}
}

func tePopulateWildHazards(g *Game, zn *Zone, sr *Rng) {
	if !sr.Chance(0.4) {
		return
	}
	// Look for a tree to hang a beehive on
	for y := 4; y < MH-4; y++ {
		for x := 4; x < MW-4; x++ {
			if zn.At(x, y) == TTree && zn.At(x+1, y).D().Walk {
				zn.Set(x+1, y, TBeehive)
				zn.Notes[y*MW+(x+1)] = "A wild humming beehive swollen with stinging bees. Bump or shoot to drop on enemies!"
				return
			}
		}
	}
}

// teBump handles dynamic tactical combat interactions
func teBump(g *Game, x, y int) bool {
	zn := g.Cur
	if zn == nil || !zn.In(x, y) {
		return false
	}
	t := zn.At(x, y)
	p := g.P

	switch t {
	case TChandelier:
		// Leap and swing across the room up to 4 tiles in player's forward direction
		dx := x - p.X
		dy := y - p.Y
		if dx == 0 && dy == 0 {
			dx = 1
		}
		targetX, targetY := x+dx*2, y+dy*2
		if !zn.In(targetX, targetY) {
			targetX, targetY = x, y
		}

		g.spend(800)
		g.Msg("{{Y|WHOOSH!}} You leap into the air, grab the swinging iron chandelier, and sail across the room!")
		p.X, p.Y = targetX, targetY

		// If landing adjacent to or on an enemy, execute a devastating diving kick!
		if targetActor := zn.ActorAt(targetX, targetY); targetActor != nil && targetActor != p {
			dmg := 20 + g.R.Int(5, 15)
			g.hurt(targetActor, dmg, p, "a flying chandelier diving kick")
			targetActor.AddSt("dazed", 3)
			g.Msg(fmt.Sprintf("{{R|CRUNCH!}} Your flying drop-kick slams into %s for %d damage, dazing them!", targetActor.Name, dmg))
		}
		return true

	case TBarrel:
		// Puncture the barrel: floods 3x3 tiles with slick wine or oil
		zn.Set(x, y, TDirt)
		g.spend(600)
		g.Msg("{{C|KRAKK!}} You smash open the oak barrel! Liquid floods the surrounding flagstones!")

		for dy := -1; dy <= 1; dy++ {
			for dx := -1; dx <= 1; dx++ {
				nx, ny := x+dx, y+dy
				if zn.In(nx, ny) && zn.At(nx, ny).D().Walk {
					zn.Set(nx, ny, TShallow)
					// Douse or trip actors standing in the puddle
					if a := zn.ActorAt(nx, ny); a != nil {
						a.AddSt("slow", 3)
						g.Msg(fmt.Sprintf("%s slips in the sloshing slick!", a.Name))
					}
				}
			}
		}
		return true

	case TBeehive:
		// Drop the beehive, releasing an angry swarm
		zn.Set(x, y, TDirt)
		g.spend(500)
		g.Msg("{{Y|BUZZZZZ!}} The wild beehive falls and shatters! An enraged swarm of stinging bees bursts forth!")

		// Spawn hostile bee swarm actor
		swarm := g.Spawn("spider", zn.Level, g.R)
		if swarm != nil {
			swarm.Name = "Enraged Bee Swarm"
			swarm.Desc = "A blinding, buzzing cloud of angry stinging bees."
			swarm.Faction = "beasts"
			swarm.Glyph = 'δ'
			swarm.F = CYellow
			swarm.X, swarm.Y = x, y
			zn.Actors = append(zn.Actors, swarm)
		}
		return true

	case TScaffold:
		// Cutting scaffold ropes
		zn.Set(x, y, TDirt)
		g.spend(800)
		g.Msg("{{R|TIMBER!}} You sever the scaffold binding ropes! The entire wooden structure collapses!")

		// Deal crushing damage to anyone in 3x3 footprint
		for dy := -1; dy <= 1; dy++ {
			for dx := -1; dx <= 1; dx++ {
				nx, ny := x+dx, y+dy
				if a := zn.ActorAt(nx, ny); a != nil && a != p {
					crushDmg := 30 + g.R.Int(10, 20)
					g.hurt(a, crushDmg, p, "collapsing scaffolding timber")
					a.AddSt("pinned", 3)
					g.Msg(fmt.Sprintf("Falling timbers crush %s for %d damage!", a.Name, crushDmg))
				}
			}
		}
		// Scatter planks
		zn.Items = append(zn.Items, &Item{
			ID: g.newID(), Kind: "scrap", Name: "fallen scaffold plank", Glyph: '=', F: CBrown, Weight: 2, X: x, Y: y,
		})
		return true

	case TBrazier:
		// If enemy is standing adjacent, shove them into the brazier!
		for dy := -1; dy <= 1; dy++ {
			for dx := -1; dx <= 1; dx++ {
				nx, ny := x+dx, y+dy
				if a := zn.ActorAt(nx, ny); a != nil && a != p {
					burnDmg := 18 + g.R.Int(5, 10)
					g.hurt(a, burnDmg, p, "being kicked into a blazing brazier")
					a.AddSt("burning", 5)
					g.spend(800)
					g.Msg(fmt.Sprintf("{{R|SIZZLE!}} You kick %s violently into the glowing brazier! (%d fire dmg)", a.Name, burnDmg))
					return true
				}
			}
		}
		g.Msg("The fire in the brazier crackles with bright heat.")
		return true

	case TCrate, TSeat:
		// Flip table/benches for cover (+4 DV)
		st := g.te()
		posKey := fmt.Sprintf("%s:%d,%d", zn.Key, x, y)
		if !st.FlippedTables[posKey] {
			st.FlippedTables[posKey] = true
			p.AddSt("fortified", 4)
			g.spend(500)
			g.Msg("{{C|CLATTER!}} You flip the heavy table on its side, providing immediate sturdy half-cover (+4 DV)!")
			return true
		}
	}

	return false
}
