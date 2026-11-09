package engine

import (
	"fmt"
	"strings"
)

// Thaumaturgical Smog, Spell Backfires & Crystalline Mutations (prefix thm).
// Arcane spell research backfires (temporal echoes duplicating actions, localized gravity inversion).
// Mana sickness & crystalline skin mutations (+AV, vulnerable to sonic, attracts mana leeches).

const (
	BackfireTemporalEcho     = "temporal_echo"
	BackfireGravityInversion = "gravity_inversion"
	BackfireWarpCascade      = "warp_cascade"
)

// ThaumaturgyState tracks ambient smog, research backfires, mutations, and mana leeches.
type ThaumaturgyState struct {
	SmogDensity           map[string]int `json:"smog,omitempty"`
	ManaSickness          int            `json:"ms"`
	HasCrystallineSkin    bool           `json:"crys"`
	TemporalEchoTurns     int            `json:"te"`
	GravityInversionTurns int            `json:"gi"`
	LeechesSpawned        int            `json:"ls"`
	BackfiresTriggered    int            `json:"bf"`
	ResearchSuccesses     int            `json:"rs"`
}

func thmSt(g *Game) *ThaumaturgyState {
	return g.ext("thm", func() any {
		return &ThaumaturgyState{
			SmogDensity: map[string]int{},
		}
	}).(*ThaumaturgyState)
}

// GetThaumicSmog returns the smog concentration of the current zone.
func (g *Game) GetThaumicSmog() int {
	if g.Cur == nil {
		return 0
	}
	st := thmSt(g)
	if st.SmogDensity == nil {
		st.SmogDensity = map[string]int{}
	}
	return st.SmogDensity[g.Cur.Key]
}

// AddThaumicSmog increases smog concentration in the current zone.
func (g *Game) AddThaumicSmog(amount int) {
	if g.Cur == nil {
		return
	}
	st := thmSt(g)
	if st.SmogDensity == nil {
		st.SmogDensity = map[string]int{}
	}
	st.SmogDensity[g.Cur.Key] = clampi(st.SmogDensity[g.Cur.Key]+amount, 0, 100)
}

// ConductThaumaturgicalResearch attempts arcane inquiry into high-tier spell theories.
func (g *Game) ConductThaumaturgicalResearch(field string) (bool, string) {
	st := thmSt(g)
	smog := g.GetThaumicSmog()

	// Risk of backfire scales with ambient smog and low intelligence
	intMod := Mod(g.P.Attr[INT])
	backfireChance := 20 + (smog / 2) - (intMod * 5)
	backfireChance = clampi(backfireChance, 10, 85)

	if g.R.Int(1, 100) <= backfireChance {
		// Backfire!
		st.BackfiresTriggered++
		types := []string{BackfireTemporalEcho, BackfireGravityInversion, BackfireWarpCascade}
		bType := types[g.R.Int(0, len(types)-1)]
		return false, g.TriggerSpellBackfire(bType)
	}

	// Success
	st.ResearchSuccesses++
	g.AddThaumicSmog(5)
	rewardScroll := &Item{
		ID:     g.newID(),
		Name:   fmt.Sprintf("Research Treatise on %s", field),
		Glyph:  '?',
		F:      CMagenta,
		Kind:   "book",
		Value:  240,
		Weight: 1,
		Lore:   fmt.Sprintf("Empirical findings on %s, penned amid glowing thaumic vapour.", field),
	}
	g.P.Inv = append(g.P.Inv, rewardScroll)
	msg := fmt.Sprintf("Your research into %s yields crystalline insights! A treatise is transcribed.", field)
	g.Msg(msg)
	return true, msg
}

// TriggerSpellBackfire manifests arcane anomalies from ruptured spell weaves.
func (g *Game) TriggerSpellBackfire(kind string) string {
	st := thmSt(g)
	st.BackfiresTriggered++
	g.AddThaumicSmog(15)

	switch kind {
	case BackfireTemporalEcho:
		st.TemporalEchoTurns = 12
		if g.Cur != nil {
			echo := &Actor{
				ID:      g.newID(),
				Name:    fmt.Sprintf("Temporal Echo of %s", g.P.Name),
				Proper:  true,
				Glyph:   '@',
				F:       CCyan,
				X:       clampi(g.P.X+1, 1, MW-2),
				Y:       g.P.Y,
				Tpl:     "temporal_echo",
				Faction: "player",
				Level:   g.P.Level,
				HP:      g.P.HP / 2,
				MaxHP:   g.P.MaxHP / 2,
				Speed:   g.P.Speed,
				AI:      "follower",
				Desc:    "A translucent, stuttering refraction of yourself from seconds yet to come.",
			}
			g.Cur.Actors = append(g.Cur.Actors, echo)
		}
		msg := "Time fractals shatter! A shimmering temporal echo manifests, mirroring your every motion."
		g.Msg(msg)
		return msg

	case BackfireGravityInversion:
		st.GravityInversionTurns = 8
		if g.P != nil {
			g.P.AddSt("gravity_inverted", 8)
		}
		msg := "A localized gravity inversion pulls everything toward the vault ceiling! Boots leave the stone."
		g.Msg(msg)
		return msg

	case BackfireWarpCascade:
		g.ApplyManaSickness(20)
		if g.Cur != nil {
			for _, a := range g.Cur.Actors {
				if a != nil && a.HP > 0 {
					g.hurt(a, g.R.Int(5, 15), g.P, "arcane")
				}
			}
		}
		msg := "A violet warp cascade erupts through the chamber, discharging raw aether and sickening the soul."
		g.Msg(msg)
		return msg
	}
	return "The arcane weave flickers violently."
}

// ApplyManaSickness increases mana sickness, triggering crystalline skin at high saturation.
func (g *Game) ApplyManaSickness(amount int) {
	st := thmSt(g)
	st.ManaSickness += amount
	if g.P != nil {
		g.P.AddSt("mana_sickness", st.ManaSickness)
	}

	if st.ManaSickness >= 30 && !st.HasCrystallineSkin {
		g.ApplyCrystallineMutation(g.P)
	}
}

// ApplyCrystallineMutation petrifies the epidermis into resonant, protective crystal plates.
func (g *Game) ApplyCrystallineMutation(a *Actor) {
	if a == nil {
		return
	}
	st := thmSt(g)
	st.HasCrystallineSkin = true
	a.AddSt("crystalline_skin", 2000)
	a.AVNat += 4

	g.Msg("Mana sickness crystalizes within your skin! Hard, faceted gemstone plates sprout across your flesh (+4 AV, sonic vulnerability, attracts mana leeches).")
}

// SpawnManaLeech spawns a parasitic entity attracted to crystalline resonance.
func (g *Game) SpawnManaLeech(x, y int) *Actor {
	if g.Cur == nil {
		return nil
	}
	st := thmSt(g)
	st.LeechesSpawned++

	leech := &Actor{
		ID:      g.newID(),
		Name:    "Mana Leech",
		Glyph:   'm',
		F:       CMagenta,
		X:       x,
		Y:       y,
		Tpl:     "mana_leech",
		Faction: "monsters",
		Level:   5,
		HP:      20,
		MaxHP:   20,
		AVNat:   1,
		DVNat:   14,
		Speed:   110,
		AI:      "hostile",
		Desc:    "A fluttering, translucent parasite that drinks ambient magic and clings to crystalline mutations.",
	}
	g.Cur.Actors = append(g.Cur.Actors, leech)
	g.Msg("Drawn by the chiming resonance of your crystalline skin, a Mana Leech drops from the shadows!")
	return leech
}

// PurgeThaumicSickness clears mana sickness and heals the afflicted caster.
func (g *Game) PurgeThaumicSickness() bool {
	st := thmSt(g)
	if st.ManaSickness <= 0 {
		g.Msg("You are free of mana sickness.")
		return false
	}
	st.ManaSickness = 0
	if g.P != nil && g.P.Status != nil {
		delete(g.P.Status, "mana_sickness")
	}
	g.Msg("Herbal decoctions and grounding earth purge the residual mana sickness from your veins.")
	return true
}

func init() {
	unLocalKeys = append(unLocalKeys, "thm")

	// Zone hook: ruins and arcane towers start with ambient thaumaturgical smog
	zoneHooks = append(zoneHooks, func(g *Game, zn *Zone, r *Rng) {
		if zn == nil {
			return
		}
		if strings.Contains(strings.ToLower(zn.Kind), "tower") || strings.Contains(strings.ToLower(zn.Kind), "ruin") || strings.Contains(strings.ToLower(zn.Name), "sanctum") {
			st := thmSt(g)
			if st.SmogDensity == nil {
				st.SmogDensity = map[string]int{}
			}
			st.SmogDensity[zn.Key] = r.Int(20, 50)
		}
	})

	turnHooks = append(turnHooks, func(g *Game) {
		st := thmSt(g)
		// Gravity inversion timer
		if st.GravityInversionTurns > 0 {
			st.GravityInversionTurns--
			if st.GravityInversionTurns == 0 {
				if g.P != nil && g.P.Status != nil {
					delete(g.P.Status, "gravity_inverted")
					g.Msg("Gravity snaps back violently! You crash to the floor, bruising your knees.")
					g.hurt(g.P, 6, nil, "falling")
				}
			}
		}

		// Temporal echo timer
		if st.TemporalEchoTurns > 0 {
			st.TemporalEchoTurns--
		}

		// Crystalline skin attracts mana leeches in wild zones
		if st.HasCrystallineSkin && g.Cur != nil && g.Turn%250 == 0 {
			r := g.R
			if r.Int(1, 100) <= 35 && len(g.Cur.Actors) < 30 {
				x, y, ok := g.Cur.FreeSpot(r, func(x, y int) bool { return g.Cur.Walkable(x, y) })
				if ok {
					g.SpawnManaLeech(x, y)
				}
			}
		}
	})

	talkHooks = append(talkHooks, func(g *Game, a *Actor) []Option {
		if a == nil || (a.Role != "scholar" && a.Role != "mage" && a.Role != "alchemist") {
			return nil
		}
		var opts []Option
		opts = append(opts, Option{
			Text: "Conduct thaumaturgical research into chrono-divination",
			Do: func() {
				g.ConductThaumaturgicalResearch("Chrono-Divination")
			},
		})
		opts = append(opts, Option{
			Text: "Conduct thaumaturgical research into graviturgy",
			Do: func() {
				g.ConductThaumaturgicalResearch("Graviturgy")
			},
		})
		st := thmSt(g)
		if st.ManaSickness > 0 {
			opts = append(opts, Option{
				Text: "Purge mana sickness using alchemical salts (40 gold)",
				Do: func() {
					if g.P.Gold >= 40 {
						g.P.Gold -= 40
						g.PurgeThaumicSickness()
					} else {
						g.Msg("Not enough gold for the purifying salts.")
					}
				},
			})
		}
		return opts
	})
}
