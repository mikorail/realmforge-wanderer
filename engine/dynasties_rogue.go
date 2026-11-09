package engine

import (
	"fmt"
	"strings"
)

// Dynastic Bloodline Curses, Heirs & Pretender Quests (prefix dy).
// Generational heir continuation inheriting ancestral bloodline traits:
// ('Dragon-Heart', 'Red Miasma', 'Giant Stature', 'Seer Lunacy').
// Ancestral heirloom weapons passed down with accrued deeds.
// Pretender / usurper quests: swear fealty to an exiled claimant and infiltrate citadel.

const (
	TraitDragonHeart = "Dragon-Heart"
	TraitRedMiasma   = "Red Miasma"
	TraitGiantStature = "Giant Stature"
	TraitSeerLunacy  = "Seer Lunacy"
)

// BloodlineTraitDef details the boons and curses of dynastic bloodlines.
type BloodlineTraitDef struct {
	ID        string
	Name      string
	PerkDesc  string
	CurseDesc string
}

var BloodlineTraits = map[string]BloodlineTraitDef{
	TraitDragonHeart: {
		ID:        TraitDragonHeart,
		Name:      "Dragon-Heart",
		PerkDesc:  "+2 Melee PV, +50% fire resistance, bursts into Dragon Fury when low on health.",
		CurseDesc: "Metabolism burns fiercely, requiring rapid nourishment.",
	},
	TraitRedMiasma: {
		ID:        TraitRedMiasma,
		Name:      "Red Miasma",
		PerkDesc:  "Complete immunity to poisons and plagues; vents toxic miasma when grievously struck.",
		CurseDesc: "Tainted blood unsettles commoners and courtiers alike.",
	},
	TraitGiantStature: {
		ID:        TraitGiantStature,
		Name:      "Giant Stature",
		PerkDesc:  "+15 Max HP, +2 AV, +50% encumbrance threshold, and crushing knockback on heavy blows.",
		CurseDesc: "Bulky frame reduces defensive dodge (-2 DV).",
	},
	TraitSeerLunacy: {
		ID:        TraitSeerLunacy,
		Name:      "Seer Lunacy",
		PerkDesc:  "Clairvoyant second sight detects unseen horrors, hidden doors, and distant stairs.",
		CurseDesc: "Susceptible to prophetic vertigo and cosmic visions that blur waking reality.",
	},
}

// DynastyHeirloom is a sacred weapon passed through generations, collecting historical deeds.
type DynastyHeirloom struct {
	Name       string   `json:"n"`
	Kind       string   `json:"k"`
	BaseSkill  string   `json:"sk"`
	PV         int      `json:"pv"`
	DN         int      `json:"dn"`
	DS         int      `json:"ds"`
	Deeds      []string `json:"deeds"`
	PowerBonus int      `json:"pb"`
}

// PretenderClaimant represents an exiled legitimate heir seeking the throne.
type PretenderClaimant struct {
	ID          string `json:"id"`
	Name        string `json:"n"`
	Realm       string `json:"rlm"`
	UsurperName string `json:"usr"`
	CitadelTown string `json:"twn"`
	Title       string `json:"ttl"`
}

// DynastyState stores generational bloodline, heirloom, and pretender quest progress.
type DynastyState struct {
	HouseName          string             `json:"hname"`
	Generation         int                `json:"gen"`
	Bloodline          string             `json:"trait"`
	Heirloom           *DynastyHeirloom   `json:"heirloom,omitempty"`
	Prestige           int                `json:"prest"`
	SwornClaimant      *PretenderClaimant `json:"clm,omitempty"`
	CitadelInfiltrated bool               `json:"cit_done"`
}

func dySt(g *Game) *DynastyState {
	return g.ext("dy", func() any {
		return &DynastyState{
			Generation: 1,
			HouseName:  "House of the Wanderer",
			Bloodline:  TraitDragonHeart,
		}
	}).(*DynastyState)
}

// SetupDynasty establishes the founding lineage of the player's house.
func (g *Game) SetupDynasty(houseName, traitID string) {
	st := dySt(g)
	if strings.TrimSpace(houseName) != "" {
		st.HouseName = houseName
	}
	if _, ok := BloodlineTraits[traitID]; ok {
		st.Bloodline = traitID
	} else {
		st.Bloodline = TraitDragonHeart
	}

	st.Heirloom = &DynastyHeirloom{
		Name:      fmt.Sprintf("Ancestral Blade of %s", st.HouseName),
		Kind:      "weapon",
		BaseSkill: "blade",
		PV:        3,
		DN:        1,
		DS:        10,
		Deeds:     []string{"Forged at the founding of the bloodline."},
	}

	g.applyBloodlinePerks()
	g.Msg(fmt.Sprintf("The chronicle opens for %s, carrying the %s bloodline.", st.HouseName, st.Bloodline))
}

func (g *Game) applyBloodlinePerks() {
	st := dySt(g)
	if g.P == nil {
		return
	}
	switch st.Bloodline {
	case TraitDragonHeart:
		g.P.AVNat += 1
	case TraitRedMiasma:
		if g.P.Status != nil {
			delete(g.P.Status, "poisoned")
			delete(g.P.Status, "diseased")
		}
	case TraitGiantStature:
		g.P.MaxHP += 15
		g.P.HP += 15
		g.P.AVNat += 2
		g.P.DVNat -= 2
	case TraitSeerLunacy:
		g.P.DVNat += 1
	}
}

// AccrueHeirloomDeed records a momentous triumph onto the dynastic heirloom.
func (g *Game) AccrueHeirloomDeed(deed string) {
	st := dySt(g)
	if st.Heirloom == nil {
		return
	}
	st.Heirloom.Deeds = append(st.Heirloom.Deeds, deed)
	st.Heirloom.PowerBonus++
	st.Heirloom.PV++
	st.Prestige += 15
	g.Msg(fmt.Sprintf("The deeds of the ancestor are etched into %s: '%s'.", st.Heirloom.Name, deed))
}

// ContinueDynastyHeir hands over the run to the next generational scion of the house.
func (g *Game) ContinueDynastyHeir() bool {
	st := dySt(g)
	st.Generation++
	st.Prestige += 20

	heirName := fmt.Sprintf("Scion %s %d", st.HouseName, st.Generation)
	if g.P != nil {
		g.P.Name = heirName
		g.P.HP = g.P.MaxHP
		if g.P.Status != nil {
			for k := range g.P.Status {
				delete(g.P.Status, k)
			}
		}
	}

	// Deliver heirloom weapon into heir's inventory if missing
	if st.Heirloom != nil {
		found := false
		for _, it := range g.P.Inv {
			if it != nil && it.Name == st.Heirloom.Name {
				found = true
				break
			}
		}
		if !found {
			weapon := &Item{
				ID:     g.newID(),
				Name:   st.Heirloom.Name,
				Glyph:  '/',
				F:      CCyan,
				Kind:   "weapon",
				Skill:  st.Heirloom.BaseSkill,
				DN:     st.Heirloom.DN,
				DS:     st.Heirloom.DS,
				PV:     st.Heirloom.PV,
				Value:  450 + len(st.Heirloom.Deeds)*100,
				Weight: 4,
				Lore:   fmt.Sprintf("Heirloom of %s, bearing %d recorded ancestral deeds.", st.HouseName, len(st.Heirloom.Deeds)),
			}
			g.P.Inv = append(g.P.Inv, weapon)
		}
	}

	g.applyBloodlinePerks()
	g.Msg(fmt.Sprintf("%s takes up the ancestral heirloom, inheriting the %s curse for Generation %d.", heirName, st.Bloodline, st.Generation))
	return true
}

// SwearFealtyPretender pledges the player's blade to an exiled royal claimant.
func (g *Game) SwearFealtyPretender(claimant *PretenderClaimant) bool {
	if claimant == nil {
		return false
	}
	st := dySt(g)
	st.SwornClaimant = claimant
	g.Msg(fmt.Sprintf("You draw your heirloom and swear the sacred Oath of Fealty to %s, exiled sovereign of %s.", claimant.Name, claimant.Realm))
	return true
}

// InfiltrateCitadel executes the pretender's coup against the reigning usurper.
func (g *Game) InfiltrateCitadel() bool {
	st := dySt(g)
	if st.SwornClaimant == nil {
		g.Msg("You have sworn no fealty to any exiled pretender.")
		return false
	}
	if st.CitadelInfiltrated {
		g.Msg(fmt.Sprintf("The citadel of %s has already been seized for %s.", st.SwornClaimant.CitadelTown, st.SwornClaimant.Name))
		return false
	}

	st.CitadelInfiltrated = true
	st.Prestige += 75
	g.AccrueHeirloomDeed(fmt.Sprintf("Infiltrated the Citadel of %s and deposed usurper %s", st.SwornClaimant.CitadelTown, st.SwornClaimant.UsurperName))

	// Reward gold and royal artifact
	rewardGold := 600
	g.P.Gold += rewardGold

	royalRing := &Item{
		ID:     g.newID(),
		Name:   fmt.Sprintf("Signet of %s", st.SwornClaimant.Realm),
		Glyph:  '=',
		F:      CYellow,
		Kind:   "artifact",
		Value:  800,
		Weight: 1,
		Lore:   fmt.Sprintf("Presented by %s upon reclaiming the high citadel.", st.SwornClaimant.Name),
	}
	g.P.Inv = append(g.P.Inv, royalRing)

	g.Msg(fmt.Sprintf("The citadel falls! Usurper %s is deposed. %s ascends the high throne, rewarding you with %d gold and royal honors.",
		st.SwornClaimant.UsurperName, st.SwornClaimant.Name, rewardGold))
	return true
}

func init() {
	turnHooks = append(turnHooks, func(g *Game) {
		st := dySt(g)
		if g.P == nil {
			return
		}
		// Dragon-Heart low health fury
		if st.Bloodline == TraitDragonHeart {
			if g.P.HP < (g.P.MaxHP * 35 / 100) {
				g.P.AddSt("dragon_fury", 3)
			}
		}
		// Red Miasma passive protection
		if st.Bloodline == TraitRedMiasma {
			if g.P.Status != nil {
				delete(g.P.Status, "poisoned")
				delete(g.P.Status, "diseased")
			}
		}
		// Seer Lunacy occasional prophetic vision
		if st.Bloodline == TraitSeerLunacy && g.Turn%300 == 0 && g.Cur != nil {
			g.Msg("A veil of cosmic ether lifts: your mind perceives secret ley paths through the stone.")
		}
	})

	killHooks = append(killHooks, func(g *Game, d, src *Actor) {
		if d == nil || !d.Boss {
			return
		}
		st := dySt(g)
		if st.Heirloom != nil {
			g.AccrueHeirloomDeed(fmt.Sprintf("Vanquished %s in mortal combat", d.Name))
		}
	})

	talkHooks = append(talkHooks, func(g *Game, a *Actor) []Option {
		if a == nil {
			return nil
		}
		st := dySt(g)
		var opts []Option

		// Exiled claimant in tavern or ruin
		if strings.Contains(strings.ToLower(a.Role), "exile") || strings.Contains(strings.ToLower(a.Title), "claimant") || a.Tpl == "exiled_pretender" {
			if st.SwornClaimant == nil {
				opts = append(opts, Option{
					Text: fmt.Sprintf("Swear fealty to %s and promise to infiltrate the citadel", a.Name),
					Do: func() {
						claimant := &PretenderClaimant{
							ID:          fmt.Sprintf("pretender_%d", a.ID),
							Name:        a.Name,
							Realm:       "the Broken Marches",
							UsurperName: "Lord Vane the Tyrant",
							CitadelTown: "Highspire Citadel",
							Title:       "Rightful Sovereign",
						}
						g.SwearFealtyPretender(claimant)
					},
				})
			} else if !st.CitadelInfiltrated {
				opts = append(opts, Option{
					Text: "Report success in the citadel coup",
					Do: func() {
						g.InfiltrateCitadel()
					},
				})
			}
		}
		return opts
	})
}
