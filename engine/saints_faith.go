package engine

import (
	"fmt"
	"strings"
)

// Living Faiths: Divine Oaths, Inquisitors & Shrines (prefix sf).
// Swearing sacred vows with strict taboos (Oath of the Iron Fist, Oath of the Sunken Eye) granting divine invocations.
// Inquisitor hit squads pursuing apostates/heretics through town alleys.
// Praying at living saint shrines to receive visions and blessings.

const (
	OathIronFist  = "Oath of the Iron Fist"
	OathSunkenEye = "Oath of the Sunken Eye"
)

// DivineOathDef outlines the taboo and power of a holy vow.
type DivineOathDef struct {
	ID             string
	Name           string
	TabooDesc      string
	InvocationName string
	InvocationDesc string
}

var DivineOaths = map[string]DivineOathDef{
	OathIronFist: {
		ID:             OathIronFist,
		Name:           OathIronFist,
		TabooDesc:      "May never wield edged blades or piercing arrows; only bare fists and blunt cudgels.",
		InvocationName: "Fist of Heaven",
		InvocationDesc: "Calls down a radiant concussive hammer blow that shatters bone and stuns heretics.",
	},
	OathSunkenEye: {
		ID:             OathSunkenEye,
		Name:           OathSunkenEye,
		TabooDesc:      "May never carry open flame, lanterns, or torches into dark sacred halls.",
		InvocationName: "Abyssal Deluge",
		InvocationDesc: "Channels a consecrated deluge of crushing holy water, purging fiends and unholy flame.",
	},
}

// SaintsFaithState stores active oaths, piety, heresy status, and shrine blessings.
type SaintsFaithState struct {
	ActiveOath       string          `json:"oath"`
	Piety            int             `json:"piety"`
	IsApostate       bool            `json:"apo"`
	HeresyLevel      int             `json:"her"`
	InquisitorsSent  int             `json:"inq_sent"`
	InquisitorsSlain int             `json:"inq_slain"`
	VisitedShrines   map[string]bool `json:"shrines,omitempty"`
}

func sfSt(g *Game) *SaintsFaithState {
	return g.ext("sf", func() any {
		return &SaintsFaithState{
			Piety:          50,
			VisitedShrines: map[string]bool{},
		}
	}).(*SaintsFaithState)
}

// SwearDivineOath binds the player to an ecclesiastical vow.
func (g *Game) SwearDivineOath(oathID string) bool {
	st := sfSt(g)
	def, ok := DivineOaths[oathID]
	if !ok {
		return false
	}

	st.ActiveOath = def.Name
	st.Piety = 70
	st.IsApostate = false
	st.HeresyLevel = 0

	g.Msg(fmt.Sprintf("You kneel before the altar and swear the %s: %s", def.Name, def.TabooDesc))
	return true
}

// CheckOathTaboos inspects the player's equipment and actions for holy violations.
func (g *Game) CheckOathTaboos() {
	st := sfSt(g)
	if st.ActiveOath == "" || st.IsApostate || g.P == nil {
		return
	}

	// Oath of the Iron Fist: forbidden bladed / piercing weapons
	if st.ActiveOath == OathIronFist {
		for _, bp := range g.P.Body {
			if bp != nil && bp.Item != nil && bp.Item.Kind == "weapon" {
				sk := strings.ToLower(bp.Item.Skill)
				if sk == "blade" || sk == "shortblade" || sk == "spear" || sk == "axe" {
					g.BreakOath(fmt.Sprintf("You drew a forbidden edged weapon (%s), profaning the Oath of the Iron Fist!", bp.Item.Name))
					return
				}
			}
		}
	}

	// Oath of the Sunken Eye: forbidden torch or light in darkness
	if st.ActiveOath == OathSunkenEye && g.Cur != nil && g.Cur.Dark {
		for _, bp := range g.P.Body {
			if bp != nil && bp.Item != nil && bp.Item.Kind == "light" {
				g.BreakOath("You held aloft a profane flame in the sacred dark, breaking the Oath of the Sunken Eye!")
				return
			}
		}
	}
}

// BreakOath revokes divine favor and alerts the holy inquisition.
func (g *Game) BreakOath(reason string) {
	st := sfSt(g)
	st.IsApostate = true
	st.ActiveOath = ""
	st.Piety = 0
	st.HeresyLevel += 60

	if g.P != nil {
		g.P.AddSt("apostate_curse", 1500)
	}

	g.Msg(fmt.Sprintf("SACRILEGE! %s You are branded an Apostate by the Holy Synod!", reason))
	g.SpawnInquisitorSquad()
}

// InvokeDivinePower discharges sacred energy granted by an active oath.
func (g *Game) InvokeDivinePower(powerName string) (bool, string) {
	st := sfSt(g)
	if st.IsApostate || st.ActiveOath == "" {
		g.Msg("You have no divine favor to invoke.")
		return false, "No active oath."
	}
	if st.Piety < 25 {
		g.Msg("Your piety is insufficient to invoke holy miracles.")
		return false, "Piety too low."
	}

	st.Piety -= 25

	if powerName == "Fist of Heaven" || st.ActiveOath == OathIronFist {
		// Radiant crushing damage on nearby foes
		if g.Cur != nil {
			for _, a := range g.Cur.Actors {
				if a != nil && a.HP > 0 && a.Faction != "player" && a.Faction != "allies" {
					dx := abs(a.X - g.P.X)
					dy := abs(a.Y - g.P.Y)
					if maxi(dx, dy) <= 4 {
						dmg := g.R.Int(12, 32)
						g.hurt(a, dmg, g.P, "radiant")
						a.AddSt("stunned", 3)
					}
				}
			}
		}
		msg := "Fist of Heaven! A colossal radiant fist descends from the heavens with a thunderous crack, crushing heretical foes!"
		g.Msg(msg)
		return true, msg
	}

	if powerName == "Abyssal Deluge" || st.ActiveOath == OathSunkenEye {
		if g.Cur != nil {
			for _, a := range g.Cur.Actors {
				if a != nil && a.HP > 0 && a.Faction != "player" {
					dx := abs(a.X - g.P.X)
					dy := abs(a.Y - g.P.Y)
					if maxi(dx, dy) <= 5 {
						dmg := g.R.Int(10, 26)
						g.hurt(a, dmg, g.P, "water")
					}
				}
			}
		}
		msg := "A surge of pressurized holy water roars across the ground, purging blasphemies and drowning flames!"
		g.Msg(msg)
		return true, msg
	}

	return false, "Unknown invocation."
}

// SpawnInquisitorSquad dispatches an elite holy hit squad to pursue apostates.
func (g *Game) SpawnInquisitorSquad() []*Actor {
	if g.Cur == nil {
		return nil
	}
	zn := g.Cur
	st := sfSt(g)
	st.InquisitorsSent++

	var squad []*Actor

	// Grand Inquisitor
	ix := clampi(g.P.X+4, 1, MW-2)
	iy := clampi(g.P.Y+2, 1, MH-2)
	inq := &Actor{
		ID:      g.newID(),
		Name:    "Grand Inquisitor Malachai",
		Proper:  true,
		Glyph:   'I',
		F:       CRed,
		X:       ix,
		Y:       iy,
		Tpl:     "grand_inquisitor",
		Faction: "inquisition",
		Level:   10,
		HP:      75,
		MaxHP:   75,
		AVNat:   6,
		DVNat:   8,
		Speed:   100,
		AI:      "hostile",
		Boss:    true,
		Desc:    "Clad in scarlet vestments and iron plate, wielding an executioner's flail against heretics.",
	}
	zn.Actors = append(zn.Actors, inq)
	squad = append(squad, inq)

	// Inquisitor Justiciars
	for i := 1; i <= 2; i++ {
		jx := clampi(ix-i, 1, MW-2)
		jy := clampi(iy+i, 1, MH-2)
		just := &Actor{
			ID:      g.newID(),
			Name:    fmt.Sprintf("Zealot Justiciar #%d", i),
			Glyph:   'z',
			F:       CYellow,
			X:       jx,
			Y:       jy,
			Tpl:     "zealot_justiciar",
			Faction: "inquisition",
			Level:   7,
			HP:      40,
			MaxHP:   40,
			AVNat:   4,
			DVNat:   10,
			Speed:   105,
			AI:      "hostile",
			Desc:    "A fanatical templar hunting oath-breakers through town alleys and dark roads.",
		}
		zn.Actors = append(zn.Actors, just)
		squad = append(squad, just)
	}

	g.Msg("Iron-shod boots slam on the cobbles! An Inquisitor hit squad closes in, brandishing brands and chains!")
	return squad
}

// PrayAtSaintShrine blesses the player with saintly insight and holy protection.
func (g *Game) PrayAtSaintShrine(shrineName string) (bool, string) {
	st := sfSt(g)
	if st.VisitedShrines == nil {
		st.VisitedShrines = map[string]bool{}
	}
	st.VisitedShrines[shrineName] = true
	st.Piety = mini(100, st.Piety+35)

	if g.P != nil {
		g.P.AddSt("saintly_blessing", 600)
		g.P.HP = mini(g.P.MaxHP, g.P.HP+25)
		if g.P.Status != nil {
			delete(g.P.Status, "diseased")
			delete(g.P.Status, "poisoned")
		}
	}

	msg := fmt.Sprintf("You prostrate yourself before the Shrine of %s. Golden light infuses your spirit (+DV, cleansed toxins, restored piety).", shrineName)
	g.Msg(msg)
	return true, msg
}

func init() {
	unLocalKeys = append(unLocalKeys, "sf")

	turnHooks = append(turnHooks, func(g *Game) {
		g.CheckOathTaboos()
		st := sfSt(g)
		if st.ActiveOath != "" && g.Turn%200 == 0 {
			st.Piety = mini(100, st.Piety+2)
		}
	})

	killHooks = append(killHooks, func(g *Game, d, src *Actor) {
		if d == nil {
			return
		}
		st := sfSt(g)
		if d.Tpl == "grand_inquisitor" {
			st.InquisitorsSlain++
			g.Msg("The Grand Inquisitor collapses, his scarlet vestments soaking into the dirt.")
		}
	})

	zoneHooks = append(zoneHooks, func(g *Game, zn *Zone, r *Rng) {
		if zn == nil {
			return
		}
		if strings.Contains(strings.ToLower(zn.Kind), "temple") || strings.Contains(strings.ToLower(zn.Name), "shrine") {
			x, y, ok := zn.FreeSpot(r, func(x, y int) bool { return zn.At(x, y) == TFloor || zn.At(x, y) == TCobble })
			if ok {
				zn.Set(x, y, TAltar)
			}
		}
	})

	talkHooks = append(talkHooks, func(g *Game, a *Actor) []Option {
		if a == nil || (a.Role != "priest" && a.Role != "monk" && a.Role != "hermit") {
			return nil
		}
		st := sfSt(g)
		var opts []Option

		if st.ActiveOath == "" && !st.IsApostate {
			opts = append(opts, Option{
				Text: "Swear the Oath of the Iron Fist (Taboo: no blades/arrows; Boon: Fist of Heaven)",
				Do: func() {
					g.SwearDivineOath(OathIronFist)
				},
			})
			opts = append(opts, Option{
				Text: "Swear the Oath of the Sunken Eye (Taboo: no flames in dark; Boon: Abyssal Deluge)",
				Do: func() {
					g.SwearDivineOath(OathSunkenEye)
				},
			})
		} else if st.ActiveOath != "" {
			opts = append(opts, Option{
				Text: fmt.Sprintf("Seek confession and meditate on your %s (Piety: %d)", st.ActiveOath, st.Piety),
				Do: func() {
					st.Piety = mini(100, st.Piety+15)
					g.Msg("The priest murmurs absolution over your kneeling form.")
				},
			})
		} else if st.IsApostate {
			opts = append(opts, Option{
				Text: "Seek penance for your apostasy (200 gold donation)",
				Do: func() {
					if g.P.Gold >= 200 {
						g.P.Gold -= 200
						st.IsApostate = false
						st.HeresyLevel = 0
						if g.P.Status != nil {
							delete(g.P.Status, "apostate_curse")
						}
						g.Msg("The holy fathers accept your sacrificial gold and lift the ecclesiastical ban.")
					} else {
						g.Msg("You lack the coin required to atone for mortal apostasy.")
					}
				},
			})
		}
		return opts
	})
}
