package engine

import (
	"fmt"
	"strings"
)

// Subterranean Verticality, Spore Canopies & Acoustic Stealth (prefix sb).
// Giant mushroom canopy leaping/climbing in deep subterranean strata.
// Vibration-based acoustic stealth: blind deep-abyss apex predators tracking footstep sounds/vibrations.

type sbPos struct {
	X int `json:"x"`
	Y int `json:"y"`
}

// SubterraneanState records canopy perches, acoustic noise history, and abyssal stalkers.
type SubterraneanState struct {
	LastNoisePos      *sbPos  `json:"lnp,omitempty"`
	LastNoiseVal      int     `json:"lnv"`
	CanopyJumps       int     `json:"cj"`
	ApexPredatorsShed int     `json:"aps"`
	DecoysThrown      int     `json:"dec"`
	Canopies          []sbPos `json:"cnp,omitempty"`
}

func sbSt(g *Game) *SubterraneanState {
	return g.ext("sb", func() any { return &SubterraneanState{} }).(*SubterraneanState)
}

// IsSubterranean checks if the current zone is in deep underground strata.
func (g *Game) IsSubterranean() bool {
	if g.Cur == nil {
		return false
	}
	return g.Cur.Z > 0 || strings.Contains(strings.ToLower(g.Cur.Kind), "dungeon") || strings.Contains(strings.ToLower(g.Cur.Name), "cave")
}

// ClimbMushroomCanopy climbs from ground level up onto a giant mushroom cap.
func (g *Game) ClimbMushroomCanopy() bool {
	if g.Cur == nil {
		return false
	}
	zn := g.Cur
	px, py := g.P.X, g.P.Y

	// Check if player is on or next to a giant mushroom or fungus
	hasMushroom := false
	for dy := -1; dy <= 1; dy++ {
		for dx := -1; dx <= 1; dx++ {
			x, y := px+dx, py+dy
			if x > 0 && x < MW-1 && y > 0 && y < MH-1 {
				t := zn.At(x, y)
				if t == TMushroom || t == TFungus {
					hasMushroom = true
					break
				}
			}
		}
		if hasMushroom {
			break
		}
	}

	if !hasMushroom {
		g.Msg("There is no giant mushroom trunk nearby to climb.")
		return false
	}

	g.P.AddSt("canopy_perch", 400)
	g.Msg("You scale the fibrous fungal stalk, hauling yourself up onto the broad, springy mushroom canopy.")
	return true
}

// DescendCanopy safely steps down from a mushroom canopy to the cavern floor.
func (g *Game) DescendCanopy() bool {
	if g.P == nil || g.P.Status == nil {
		return false
	}
	if _, ok := g.P.Status["canopy_perch"]; !ok {
		g.Msg("You are already on the cavern floor.")
		return false
	}

	delete(g.P.Status, "canopy_perch")
	g.Msg("You slide down the spore-dusted stem, landing quietly on the damp subterranean floor.")
	return true
}

// LeapMushroomCanopy springs across the void to another canopy or performs a drop ambush.
func (g *Game) LeapMushroomCanopy(tx, ty int) bool {
	if g.Cur == nil || g.P == nil {
		return false
	}
	if _, ok := g.P.Status["canopy_perch"]; !ok {
		g.Msg("You must be atop a mushroom canopy to leap between caps.")
		return false
	}

	dx := abs(tx - g.P.X)
	dy := abs(ty - g.P.Y)
	dist := maxi(dx, dy)
	if dist > 3 || dist == 0 {
		g.Msg("Target is beyond leaping distance (maximum 3 tiles).")
		return false
	}

	zn := g.Cur
	targetTile := zn.At(tx, ty)
	foe := zn.ActorAt(tx, ty)
	st := sbSt(g)

	// Case 1: Drop Ambush onto a hostile foe below
	if foe != nil && foe != g.P {
		delete(g.P.Status, "canopy_perch")
		g.P.X, g.P.Y = tx, ty
		dmg := g.R.Int(10, 30) + (g.P.Attr[STR]-10)*2
		g.hurt(foe, dmg, g.P, "crushing")
		foe.AddSt("stunned", 3)
		st.CanopyJumps++
		g.Msg(fmt.Sprintf("You plummet from the canopy in a drop ambush! You crash down onto %s for %d critical crushing damage!", foe.Name, dmg))
		g.PuffSporeCloud(tx, ty)
		return true
	}

	// Case 2: Leap onto another mushroom cap
	if targetTile == TMushroom || targetTile == TFungus {
		g.P.X, g.P.Y = tx, ty
		st.CanopyJumps++
		g.Msg("You leap through the misty vault, rebounding cleanly onto the adjoining mushroom canopy!")
		return true
	}

	// Case 3: Leap down to free floor
	if zn.Walkable(tx, ty) {
		delete(g.P.Status, "canopy_perch")
		g.P.X, g.P.Y = tx, ty
		st.CanopyJumps++
		g.Msg("You vault from the canopy and land crouched on the cavern floor below.")
		g.PuffSporeCloud(tx, ty)
		return true
	}

	g.Msg("You cannot leap into impassable stone.")
	return false
}

// PuffSporeCloud releases a burst of disorienting bioluminescent spores.
func (g *Game) PuffSporeCloud(x, y int) {
	if g.Cur == nil {
		return
	}
	zn := g.Cur
	for dy := -1; dy <= 1; dy++ {
		for dx := -1; dx <= 1; dx++ {
			nx, ny := x+dx, y+dy
			if nx > 0 && nx < MW-1 && ny > 0 && ny < MH-1 {
				if a := zn.ActorAt(nx, ny); a != nil && a != g.P {
					a.AddSt("blinded", 4)
				}
			}
		}
	}
	g.Msg("A thick burst of glowing spores billows outward, obscuring sight in the subterranean gloom.")
}

// AcousticVibrationLevel computes footstep sound and bedrock vibration for an actor.
func (g *Game) AcousticVibrationLevel(a *Actor) int {
	if a == nil || g.Cur == nil {
		return 0
	}
	// Perched on canopy produces zero ground vibration
	if a.Status != nil {
		if _, ok := a.Status["canopy_perch"]; ok {
			return 0
		}
		if _, ok := a.Status["flying"]; ok {
			return 0
		}
	}

	base := 2
	// Terrain damping
	t := g.Cur.At(a.X, a.Y)
	switch t {
	case TMoss, TFungus:
		base = 0
	case TFloor, TRock, TCobble:
		base = 2
	case TShallow, TDeep:
		base = 4 // splashing water
	case TBridge, TWoodFloor:
		base = 3 // hollow creaking
	}

	// Heavy armor adds resonance (+1 per 3 AV)
	if a.AVNat > 0 {
		base += a.AVNat / 3
	}

	// Sneak skill / stealth status dampens noise
	if a.Status != nil {
		if _, ok := a.Status["sneak"]; ok {
			base = maxi(0, base-3)
		}
	}

	return clampi(base, 0, 10)
}

// SpawnBlindAbyssPredator creates an apex subterranean predator with tremorsense.
func (g *Game) SpawnBlindAbyssPredator(x, y int) *Actor {
	if g.Cur == nil {
		return nil
	}
	zn := g.Cur
	predator := &Actor{
		ID:      g.newID(),
		Name:    "Abyssal Tremor-Beast",
		Proper:  true,
		Glyph:   'T',
		F:       CRed,
		X:       x,
		Y:       y,
		Tpl:     "abyssal_tremor_beast",
		Faction: "monsters",
		Level:   9,
		HP:      65,
		MaxHP:   65,
		AVNat:   5,
		DVNat:   6,
		Speed:   115,
		AI:      "hostile",
		Boss:    true,
		Desc:    "An eyeless apex predator of the abyssal trenches with ear-horns that register the faintest vibration through bedrock.",
	}
	zn.Actors = append(zn.Actors, predator)
	g.Msg("A low, resonant click echoes through the subterranean vault. A blind apex predator stalks the stone.")
	return predator
}

// ThrowAcousticDecoy throws a stone or strikes a stalactite to create an acoustic diversion.
func (g *Game) ThrowAcousticDecoy(tx, ty int) bool {
	if g.Cur == nil {
		return false
	}
	zn := g.Cur
	if tx <= 0 || tx >= MW-1 || ty <= 0 || ty >= MH-1 {
		return false
	}

	st := sbSt(g)
	st.LastNoisePos = &sbPos{X: tx, Y: ty}
	st.LastNoiseVal = 8
	st.DecoysThrown++

	// Alert blind apex predators within 12 tiles to converge on the decoy point
	alerted := 0
	for _, a := range zn.Actors {
		if a != nil && a.Tpl == "abyssal_tremor_beast" && a.HP > 0 {
			dx := abs(a.X - tx)
			dy := abs(a.Y - ty)
			if maxi(dx, dy) <= 12 {
				// Move step toward distraction
				stepX := a.X
				if tx > a.X {
					stepX++
				} else if tx < a.X {
					stepX--
				}
				stepY := a.Y
				if ty > a.Y {
					stepY++
				} else if ty < a.Y {
					stepY--
				}
				if zn.Walkable(stepX, stepY) && zn.ActorAt(stepX, stepY) == nil {
					a.X, a.Y = stepX, stepY
				}
				alerted++
			}
		}
	}

	g.Msg(fmt.Sprintf("A sharp clack echoes as the decoy strikes (%d,%d)! The vibrations divert the attention of %d blind stalkers.", tx, ty, alerted))
	return true
}

func init() {
	unLocalKeys = append(unLocalKeys, "sb")

	// Zone hook: places giant mushroom canopies and blind predators in subterranean depths
	zoneHooks = append(zoneHooks, func(g *Game, zn *Zone, r *Rng) {
		if zn == nil || zn.Z <= 0 {
			return
		}
		// Scatter giant mushrooms for canopy climbing
		for i := 0; i < 4; i++ {
			x, y, ok := zn.FreeSpot(r, func(x, y int) bool { return zn.At(x, y) == TFloor })
			if ok {
				zn.Set(x, y, TMushroom)
			}
		}
	})

	turnHooks = append(turnHooks, func(g *Game) {
		if g.Cur == nil || !g.IsSubterranean() || g.P == nil {
			return
		}
		st := sbSt(g)
		noise := g.AcousticVibrationLevel(g.P)
		if noise > 0 {
			st.LastNoisePos = &sbPos{X: g.P.X, Y: g.P.Y}
			st.LastNoiseVal = noise
		}

		// Blind predators track vibrations
		if noise >= 3 {
			for _, a := range g.Cur.Actors {
				if a != nil && a.Tpl == "abyssal_tremor_beast" && a.HP > 0 {
					dx := abs(a.X - g.P.X)
					dy := abs(a.Y - g.P.Y)
					if maxi(dx, dy) <= 12 {
						// Stalk toward noise
						nx, ny := a.X, a.Y
						if g.P.X > a.X {
							nx++
						} else if g.P.X < a.X {
							nx--
						}
						if g.P.Y > a.Y {
							ny++
						} else if g.P.Y < a.Y {
							ny--
						}
						if g.Cur.Walkable(nx, ny) && g.Cur.ActorAt(nx, ny) == nil {
							a.X, a.Y = nx, ny
						}
					}
				}
			}
		}
	})
}
