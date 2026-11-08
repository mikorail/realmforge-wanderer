package engine

import (
	"fmt"
)

// Deep Ecology & Frontier Life (prefix ef):
// 1. Dynamic lumberjack camps that physically fell forest tiles and float log rafts downriver to sawmills.
// 2. Nomadic shepherd flocks with sheepdogs migrating between high mountain pastures in summer and lowland valleys in winter.
// 3. Seasonal apex predator migrations and monster herd breeding seasons that displace local game.
// 4. Wind-driven wild grassfires and volcanic cinder storms that scorch biomes and leave fertile ash beds in subsequent years.

type efState struct {
	FelledTrees map[string]int `json:"felled,omitempty"` // zoneKey:pos -> turn
	AshHarvests map[string]int `json:"ash,omitempty"`
	LumberLogs  int            `json:"logs,omitempty"`
}

func (g *Game) ef() *efState {
	st := g.ext("ef", func() any { return &efState{} }).(*efState)
	if st.FelledTrees == nil {
		st.FelledTrees = map[string]int{}
	}
	if st.AshHarvests == nil {
		st.AshHarvests = map[string]int{}
	}
	return st
}

func init() {
	unLocalKeys = append(unLocalKeys, "ef")
	zoneHooks = append(zoneHooks, efZoneHook)
	turnHooks = append(turnHooks, efTurnHook)
	talkHooks = append(talkHooks, efTalkHook)
}

func efTurnHook(g *Game) {
	// Wind-driven grassfire propagation and dynamic lumberjack tree-felling
	zn := g.Cur
	if zn == nil || zn.Z != 0 {
		return
	}

	// 1. Dynamic lumberjack tree felling
	if g.Turn%30 == 0 {
		efTickLumberjacks(g, zn)
	}

	// 2. Wind-driven wild grassfires
	if g.Turn%10 == 0 {
		efTickWildfires(g, zn)
	}
}

func efZoneHook(g *Game, zn *Zone, r *Rng) {
	if g == nil || zn == nil || zn.Z != 0 {
		return
	}
	sr := NewRng(fmt.Sprintf("%s:ef:%s", g.Seed, zn.Key))

	if zn.Kind == "wild" || zn.Kind == "forest" || zn.Kind == "road" {
		// 1. Lumberjack camps along riverbanks
		efSpawnLumberCamp(g, zn, sr)

		// 2. Nomadic shepherd flocks with seasonal elevation migration
		efSpawnShepherdFlock(g, zn, sr)

		// 3. Apex predator migrations & maternal breeding dens
		efSpawnApexPredatorsAndBroods(g, zn, sr)

		// 4. Fertile ash beds from past wild grassfires
		efSpawnFertileAshBeds(g, zn, sr)
	}
}

func efSpawnLumberCamp(g *Game, zn *Zone, sr *Rng) {
	// Search for a river cell adjacent to trees
	hasRiver := false
	var rx, ry int
	for y := 4; y < MH-4; y++ {
		for x := 4; x < MW-4; x++ {
			if zn.At(x, y) == TShallow || zn.At(x, y) == TDeep {
				hasRiver = true
				rx, ry = x, y
				break
			}
		}
		if hasRiver {
			break
		}
	}

	if !hasRiver || !sr.Chance(0.35) {
		return
	}

	// Find dry ground adjacent to river
	cx, cy := rx+2, ry
	if !zn.In(cx, cy) || !zn.At(cx, cy).D().Walk {
		return
	}

	// Lumberjack camp shelter & log saw
	zn.Set(cx, cy, TCrate)
	zn.Notes[cy*MW+cx] = "Sawyer's log-rack and crosscut saw bench."

	// Float log raft on the river water
	if zn.In(rx, ry) {
		zn.Set(rx, ry, TBoat)
		zn.Notes[ry*MW+rx] = "A moored timber log raft. Step aboard to ride downstream."
	}

	// Lumberjacks
	jack := g.Spawn("human", zn.Level, sr)
	if jack != nil {
		jack.Name = "River Sawyer"
		jack.Role = "ef_lumberjack"
		jack.Faction = "citizens"
		jack.Desc = "Shouldering a double-bitted felling axe, notching pine trunks to float downriver."
		jack.X, jack.Y = cx+1, cy
		zn.Actors = append(zn.Actors, jack)
	}
}

func efSpawnShepherdFlock(g *Game, zn *Zone, sr *Rng) {
	if !sr.Chance(0.3) {
		return
	}
	season := g.Season()
	// High elevation in summer, sheltered valleys in winter
	sx, sy := 20+sr.Int(0, 40), 6+sr.Int(0, 12)
	if !zn.In(sx, sy) || !zn.At(sx, sy).D().Walk {
		return
	}

	shepherd := g.Spawn("human", zn.Level, sr)
	if shepherd != nil {
		if season == 0 || season == 1 {
			shepherd.Name = "Alpine Shepherd"
			shepherd.Desc = "Grazing sheep across high summer wildflower meadows."
		} else {
			shepherd.Name = "Lowland Drover"
			shepherd.Desc = "Driving the winter flock down to sheltered valley pastures."
		}
		shepherd.Role = "ef_shepherd"
		shepherd.Faction = "citizens"
		shepherd.X, shepherd.Y = sx, sy
		zn.Actors = append(zn.Actors, shepherd)
	}

	dog := g.Spawn("dog", zn.Level, sr)
	if dog != nil {
		dog.Name = "Border Sheepdog"
		dog.Role = "ef_sheepdog"
		dog.Faction = "citizens"
		dog.Desc = "Alert and circling, nipping heels to keep the flock tightly bunched."
		dog.X, dog.Y = sx+1, sy
		zn.Actors = append(zn.Actors, dog)
	}

	for k := 0; k < 4; k++ {
		sheep := g.Spawn("goat", 1, sr)
		if sheep != nil {
			sheep.Name = "Mountain Meadow Sheep"
			sheep.Faction = "citizens"
			sheep.X, sheep.Y = sx-2+k, sy+1
			zn.Actors = append(zn.Actors, sheep)
		}
	}
}

func efSpawnApexPredatorsAndBroods(g *Game, zn *Zone, sr *Rng) {
	season := g.Season()
	isBreeding := season == 0 || season == 1

	if sr.Chance(0.25) {
		// Apex predator den
		dx, dy := 15+sr.Int(0, 50), 4+sr.Int(0, 16)
		if !zn.In(dx, dy) || !zn.At(dx, dy).D().Walk {
			return
		}

		if isBreeding {
			// Maternal alpha protecting nest
			alpha := g.Spawn("bear", zn.Level+2, sr)
			if alpha != nil {
				alpha.Name = "Maternal Alpha Ursa"
				alpha.Desc = "Massive, scarred, and fiercely protective of her newborn cubs."
				alpha.HP += 25
				alpha.X, alpha.Y = dx, dy
				zn.Actors = append(zn.Actors, alpha)
			}
			for k := 0; k < 2; k++ {
				cub := g.Spawn("bear", 1, sr)
				if cub != nil {
					cub.Name = "Furry Bear Cub"
					cub.Desc = "Tumbling and playful under the watchful eye of its mother."
					cub.X, cub.Y = dx+1+k, dy
					zn.Actors = append(zn.Actors, cub)
				}
			}
		} else {
			// Migrating apex predator pack
			for k := 0; k < 3; k++ {
				pred := g.Spawn("wolf", zn.Level+1, sr)
				if pred != nil {
					pred.Name = "Migrating Timber Alpha"
					pred.Desc = "Sleek winter coat, migrating along river corridors in search of deer."
					pred.X, pred.Y = dx+k, dy
					zn.Actors = append(zn.Actors, pred)
				}
			}
		}
	}
}

func efSpawnFertileAshBeds(g *Game, zn *Zone, sr *Rng) {
	// In savannah or dry brush regions, generate rich ash beds from historic grassfires
	if !sr.Chance(0.25) {
		return
	}
	ax, ay := 25+sr.Int(0, 30), 8+sr.Int(0, 8)
	for dy := -2; dy <= 2; dy++ {
		for dx := -3; dx <= 3; dx++ {
			x, y := ax+dx, ay+dy
			if zn.In(x, y) && zn.At(x, y) == TGrass {
				zn.Set(x, y, TAshBed)
				if sr.Chance(0.15) {
					// Rare wild bloom on ash bed
					zn.Items = append(zn.Items, &Item{
						ID: g.newID(), Kind: "ingredient", Name: "scarlet fire-poppy", Glyph: '%', F: CRed, Weight: 1, X: x, Y: y,
					})
				}
			}
		}
	}
}

func efTickLumberjacks(g *Game, zn *Zone) {
	for _, a := range zn.Actors {
		if a.Role == "ef_lumberjack" {
			// Find adjacent tree to fell
			for dy := -1; dy <= 1; dy++ {
				for dx := -1; dx <= 1; dx++ {
					tx, ty := a.X+dx, a.Y+dy
					if zn.In(tx, ty) && zn.At(tx, ty) == TTree {
						zn.Set(tx, ty, TDeadTree)
						g.ef().LumberLogs++
						if zn.In(tx, ty) {
							zn.Items = append(zn.Items, &Item{
								ID: g.newID(), Kind: "scrap", Name: "hewn pine log", Glyph: '=', F: CBrown, Weight: 3, X: tx, Y: ty,
							})
						}
						return
					}
				}
			}
		}
	}
}

func efTickWildfires(g *Game, zn *Zone) {
	// Look for burning tiles spreading to dry grass
	for y := 2; y < MH-2; y++ {
		for x := 2; x < MW-2; x++ {
			if zn.At(x, y) == TLava || zn.At(x, y) == TBrazier {
				// Chance to ignite adjacent dry grass
				for dy := -1; dy <= 1; dy++ {
					for dx := -1; dx <= 1; dx++ {
						nx, ny := x+dx, y+dy
						if zn.In(nx, ny) && (zn.At(nx, ny) == TGrass || zn.At(nx, ny) == TTallGrass) && g.R.Chance(0.05) {
							zn.Set(nx, ny, TAshBed)
						}
					}
				}
			}
		}
	}
}

func efTalkHook(g *Game, a *Actor) []Option {
	var opts []Option
	if g == nil || a == nil {
		return opts
	}

	switch a.Role {
	case "ef_lumberjack":
		opts = append(opts, Option{
			Text: "{{G|Buy freshly hewn lumber rafts for downstream journey (15 gold).}}",
			Do: func() {
				if !g.cuSpend(15) {
					g.talkMenu("\"Fifteen coppers for a well-lashed log raft.\"")
					return
				}
				g.P.AddSt("swift", 10)
				g.gainXP(20)
				g.talkMenu("\"Raft's untied! Push off into the main current and enjoy the river breeze!\"")
			},
		})

	case "ef_shepherd":
		opts = append(opts, Option{
			Text: "{{Y|Purchase a fleece of virgin mountain wool (8 gold).}}",
			Do: func() {
				if !g.cuSpend(8) {
					g.talkMenu("\"Eight gold for the finest alpine fleece.\"")
					return
				}
				g.give(&Item{ID: g.newID(), Kind: "trade", Name: "alpine sheep fleece", Value: 12, Weight: 2, Glyph: '%', F: CWhite})
				g.talkMenu("\"Warmest wool in the province, shorn just yesterday morning!\"")
			},
		})
	}

	return opts
}
