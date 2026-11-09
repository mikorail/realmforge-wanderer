package engine

import (
	"fmt"
	"strings"
)

// Naval Sailing, Flagships & High-Seas Piracy (prefix nv).
// Commandeer sailing caravels, longships, or submarine bathyspheres on sea / water tiles.
// Boarding actions across gangplanks, firing swivel cannons, and kraken sea encounters.

const (
	nvFlagCaravel     = "caravel"
	nvFlagLongship    = "longship"
	nvFlagBathysphere = "bathysphere"
)

// NavalVesselDef defines the characteristics of flagships that can be commandeered.
type NavalVesselDef struct {
	Kind        string
	Name        string
	MaxHull     int
	Armor       int
	Speed       int
	SwivelGuns  int
	MaxSwivels  int
	CanSubmerge bool
	Desc        string
}

var NavalVesselDefs = map[string]NavalVesselDef{
	nvFlagCaravel: {
		Kind:        nvFlagCaravel,
		Name:        "Sailing Caravel",
		MaxHull:     45,
		Armor:       3,
		Speed:       110,
		SwivelGuns:  2,
		MaxSwivels:  20,
		CanSubmerge: false,
		Desc:        "A multi-masted caravel built for oceanic exploration and open-sea gunnery.",
	},
	nvFlagLongship: {
		Kind:        nvFlagLongship,
		Name:        "War Longship",
		MaxHull:     35,
		Armor:       2,
		Speed:       135,
		SwivelGuns:  1,
		MaxSwivels:  15,
		CanSubmerge: false,
		Desc:        "A shallow-draught clinker longship driven by oars and square sail, feared in boarding raids.",
	},
	nvFlagBathysphere: {
		Kind:        nvFlagBathysphere,
		Name:        "Submarine Bathysphere",
		MaxHull:     60,
		Armor:       6,
		Speed:       85,
		SwivelGuns:  2,
		MaxSwivels:  12,
		CanSubmerge: true,
		Desc:        "A riveted brass and steel diving vessel with ballast tanks and reinforced quartz portholes.",
	},
}

// NavalFlagship represents the player's active commanded vessel.
type NavalFlagship struct {
	Kind       string `json:"k"`
	Name       string `json:"n"`
	Hull       int    `json:"h"`
	MaxHull    int    `json:"mh"`
	Armor      int    `json:"ar"`
	Speed      int    `json:"sp"`
	SwivelGuns int    `json:"sg"`
	SwivelAmmo int    `json:"sa"`
	Submerged  bool   `json:"sub"`
	Crew       int    `json:"cr"`
}

type nvPos struct {
	X int `json:"x"`
	Y int `json:"y"`
}

// NavalState holds persistent naval and high-seas piracy records.
type NavalState struct {
	Flagship      *NavalFlagship `json:"flag,omitempty"`
	Infamy        int            `json:"inf"`
	PlunderGold   int            `json:"pg"`
	KrakensSlayed int            `json:"ks"`
	PrizesTaken   int            `json:"pt"`
	Gangplanks    []nvPos        `json:"gp,omitempty"`
}

func nvSt(g *Game) *NavalState {
	return g.ext("nv", func() any { return &NavalState{} }).(*NavalState)
}

// CommandeerVessel assigns a flagship of the given kind to the player.
func (g *Game) CommandeerVessel(kind, name string) bool {
	def, ok := NavalVesselDefs[kind]
	if !ok {
		def = NavalVesselDefs[nvFlagCaravel]
	}
	st := nvSt(g)
	vName := name
	if strings.TrimSpace(vName) == "" {
		vName = def.Name
	}

	st.Flagship = &NavalFlagship{
		Kind:       def.Kind,
		Name:       vName,
		Hull:       def.MaxHull,
		MaxHull:    def.MaxHull,
		Armor:      def.Armor,
		Speed:      def.Speed,
		SwivelGuns: def.SwivelGuns,
		SwivelAmmo: 10,
		Submerged:  false,
		Crew:       8,
	}

	g.Msg(fmt.Sprintf("You take command of %s, hoisting your banner over the deck.", vName))
	return true
}

// SubmergeBathysphere toggles submarine diving mode on bathysphere flagships.
func (g *Game) SubmergeBathysphere() (bool, string) {
	st := nvSt(g)
	if st.Flagship == nil {
		return false, "You do not have a flagship to submerge."
	}
	if st.Flagship.Kind != nvFlagBathysphere {
		return false, fmt.Sprintf("%s is a surface vessel and cannot submerge.", st.Flagship.Name)
	}

	st.Flagship.Submerged = !st.Flagship.Submerged
	if st.Flagship.Submerged {
		g.P.AddSt("submerged_diving", 1000)
		msg := "Ballast tanks flood with sea water; the bathysphere descends silently into the abyssal depths."
		g.Msg(msg)
		return true, msg
	}

	delete(g.P.Status, "submerged_diving")
	msg := "Compressed air blows the ballast; the bathysphere breaks through the churning surface foam."
	g.Msg(msg)
	return true, msg
}

// FireSwivelCannon fires grapeshot or solid ball from a ship's mounted swivel cannon at a target.
func (g *Game) FireSwivelCannon(target *Actor) (int, bool) {
	st := nvSt(g)
	if st.Flagship == nil {
		g.Msg("You have no mounted swivel cannons without a flagship.")
		return 0, false
	}
	if st.Flagship.SwivelAmmo <= 0 {
		g.Msg("The swivel cannon mounts are out of powder and shot.")
		return 0, false
	}
	if target == nil || target.HP <= 0 {
		return 0, false
	}

	// Distance check: swivel range up to 8 tiles
	dx := abs(target.X - g.P.X)
	dy := abs(target.Y - g.P.Y)
	dist := maxi(dx, dy)
	if dist > 8 {
		g.Msg("Target is beyond the effective range of the swivel cannon.")
		return 0, false
	}

	st.Flagship.SwivelAmmo--

	// Swivel damage: base 3d8 + 4
	r := g.R
	dmg := r.Int(3, 24) + 4
	if target.Tpl == "abyssal_kraken" || strings.Contains(target.Name, "Tentacle") {
		dmg = dmg * 3 / 2 // Swivels rip through soft kraken flesh
	}

	g.hurt(target, dmg, g.P, "explosive")
	target.AddSt("stunned", 2)

	g.Msg(fmt.Sprintf("A deafening roar tears across the water as the swivel cannon discharges! %s takes %d shrapnel damage.", target.Name, dmg))
	return dmg, true
}

// DeployGangplank lays down bridge planks across sea tiles to bridge two decks or dock.
func (g *Game) DeployGangplank(x, y int) bool {
	if g.Cur == nil {
		return false
	}
	if x <= 0 || x >= MW-1 || y <= 0 || y >= MH-1 {
		return false
	}

	t := g.Cur.At(x, y)
	if t != TDeep && t != TShallow && t != TFloor && t != TBoat {
		return false
	}

	g.Cur.Set(x, y, TBridge)
	st := nvSt(g)
	st.Gangplanks = append(st.Gangplanks, nvPos{X: x, Y: y})
	g.Msg("A heavy iron-banded oak gangplank is thrown across the gap with a resonant clatter!")
	return true
}

// SpawnKrakenEncounter summons the Abyssal Kraken and grasping tentacles on water.
func (g *Game) SpawnKrakenEncounter() *Actor {
	if g.Cur == nil {
		return nil
	}
	zn := g.Cur

	// Find water tiles for kraken and tentacles
	var waters []nvPos
	for y := 2; y < MH-2; y++ {
		for x := 2; x < MW-2; x++ {
			t := zn.At(x, y)
			if t == TDeep || t == TShallow {
				if zn.ActorAt(x, y) == nil {
					waters = append(waters, nvPos{X: x, Y: y})
				}
			}
		}
	}

	if len(waters) < 4 {
		// Provide water area if currently restricted
		for cy := g.P.Y - 3; cy <= g.P.Y+3; cy++ {
			for cx := g.P.X + 2; cx <= g.P.X+6; cx++ {
				if cx > 0 && cx < MW-1 && cy > 0 && cy < MH-1 {
					zn.Set(cx, cy, TDeep)
					waters = append(waters, nvPos{X: cx, Y: cy})
				}
			}
		}
	}

	// Main Kraken body
	kp := waters[0]
	kraken := &Actor{
		ID:      g.newID(),
		Name:    "Abyssal Kraken",
		Proper:  true,
		Glyph:   'K',
		F:       CRed,
		X:       kp.X,
		Y:       kp.Y,
		Tpl:     "abyssal_kraken",
		Faction: "monsters",
		Level:   12,
		HP:      150,
		MaxHP:   150,
		AVNat:   6,
		DVNat:   8,
		Speed:   90,
		AI:      "hostile",
		Boss:    true,
		Desc:    "A titanic multi-eyed leviathan rising from the dark trench, surrounded by churning foam.",
	}
	zn.Actors = append(zn.Actors, kraken)

	// Tentacles
	tentacleCount := mini(4, len(waters)-1)
	for i := 1; i <= tentacleCount; i++ {
		tp := waters[i]
		tentacle := &Actor{
			ID:      g.newID(),
			Name:    fmt.Sprintf("Kraken Tentacle #%d", i),
			Glyph:   'S',
			F:       CMagenta,
			X:       tp.X,
			Y:       tp.Y,
			Tpl:     "kraken_tentacle",
			Faction: "monsters",
			Level:   8,
			HP:      35,
			MaxHP:   35,
			AVNat:   3,
			DVNat:   12,
			Speed:   100,
			AI:      "hostile",
			Desc:    "A barbed, rubbery tentacle thrashing the waves and seeking ship timbers to crush.",
		}
		zn.Actors = append(zn.Actors, tentacle)
	}

	g.Msg("The sea boils black with foul ink! The Abyssal Kraken erupts from the swell!")
	return kraken
}

// BoardingEncounter starts a boarding conflict against a hostile vessel.
func (g *Game) BoardingEncounter(shipName string) bool {
	if g.Cur == nil {
		return false
	}
	zn := g.Cur
	st := nvSt(g)

	st.Infamy += 10
	st.PrizesTaken++
	st.PlunderGold += 250

	// Spawn pirate/marine captain and corsairs
	cx := clampi(g.P.X+3, 2, MW-3)
	cy := g.P.Y
	capt := &Actor{
		ID:      g.newID(),
		Name:    fmt.Sprintf("Captain of %s", shipName),
		Glyph:   '@',
		F:       CYellow,
		X:       cx,
		Y:       cy,
		Tpl:     "corsair_captain",
		Faction: "pirates",
		Level:   7,
		HP:      45,
		MaxHP:   45,
		AVNat:   4,
		DVNat:   12,
		Speed:   105,
		AI:      "hostile",
		Boss:    true,
		Gold:    180,
		Desc:    "A seasoned sea raider brandishing a boarding cutlass.",
	}
	zn.Actors = append(zn.Actors, capt)

	g.Msg(fmt.Sprintf("Grappling hooks bite into the bulwarks! You initiate boarding action against %s!", shipName))
	return true
}

func init() {
	unLocalKeys = append(unLocalKeys, "nv")

	turnHooks = append(turnHooks, func(g *Game) {
		st := nvSt(g)
		if st.Flagship == nil {
			return
		}
		// Submerged breathing shield
		if st.Flagship.Submerged {
			if g.P.Status != nil {
				delete(g.P.Status, "waterlogged")
			}
		}
	})

	killHooks = append(killHooks, func(g *Game, d, src *Actor) {
		if d == nil {
			return
		}
		st := nvSt(g)
		if d.Tpl == "abyssal_kraken" {
			st.KrakensSlayed++
			st.Infamy += 25
			g.Msg("The sea settles as the carcass of the Abyssal Kraken sinks into the dark depths.")
		}
	})

	talkHooks = append(talkHooks, func(g *Game, a *Actor) []Option {
		if a == nil || (a.Role != "shipwright" && a.Role != "harbourmaster" && a.Role != "sailor") {
			return nil
		}
		st := nvSt(g)
		var opts []Option
		if st.Flagship == nil {
			opts = append(opts, Option{
				Text: "Commandeer a naval sailing caravel (300 gold)",
				Do: func() {
					if g.P.Gold >= 300 {
						g.P.Gold -= 300
						g.CommandeerVessel(nvFlagCaravel, "Sea Hound")
					} else {
						g.Msg("You do not have enough gold to secure the caravel.")
					}
				},
			})
			opts = append(opts, Option{
				Text: "Commandeer a war longship (250 gold)",
				Do: func() {
					if g.P.Gold >= 250 {
						g.P.Gold -= 250
						g.CommandeerVessel(nvFlagLongship, "Raven Prow")
					} else {
						g.Msg("You do not have enough gold for the longship.")
					}
				},
			})
			opts = append(opts, Option{
				Text: "Commission a submarine bathysphere (500 gold)",
				Do: func() {
					if g.P.Gold >= 500 {
						g.P.Gold -= 500
						g.CommandeerVessel(nvFlagBathysphere, "Nautilus")
					} else {
						g.Msg("The brassworks demand 500 gold for a pressurized bathysphere.")
					}
				},
			})
		} else {
			opts = append(opts, Option{
				Text: fmt.Sprintf("Restock swivel cannon ammunition (50 gold) [%d in hold]", st.Flagship.SwivelAmmo),
				Do: func() {
					if g.P.Gold >= 50 {
						g.P.Gold -= 50
						st.Flagship.SwivelAmmo += 10
						g.Msg("Your ship hold is restocked with blackpowder charges and canister shot.")
					} else {
						g.Msg("Not enough coin for swivel charges.")
					}
				},
			})
		}
		return opts
	})
}
