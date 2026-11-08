package engine

import (
	"fmt"
	"strings"
)

// Cultural & Subrace Depth (prefix cd):
// 1. Inter-tribal moots, grand martial kurultais, and ancestral memory shrines unlocking subrace combat stances.
// 2. Fully mobile nomadic tent-cities and wagon caravans that pack up and migrate across the world map with the seasons.
// 3. Subterranean mushroom-forest silkworm farms and bioluminescent coral cultivation domes on the sea floor.

type cdState struct {
	ActiveStance string          `json:"stance,omitempty"`
	KnownStances map[string]bool `json:"stances,omitempty"`
	NomadWX      int             `json:"nomad_wx,omitempty"`
	NomadWY      int             `json:"nomad_wy,omitempty"`
	SilkHarvests int             `json:"silk,omitempty"`
	CoralPearls  int             `json:"pearls,omitempty"`
}

func (g *Game) cd() *cdState {
	st := g.ext("cd", func() any { return &cdState{} }).(*cdState)
	if st.KnownStances == nil {
		st.KnownStances = map[string]bool{}
	}
	return st
}

func init() {
	unLocalKeys = append(unLocalKeys, "cd")
	zoneHooks = append(zoneHooks, cdZoneHook)
	turnHooks = append(turnHooks, cdTurnHook)
	talkHooks = append(talkHooks, cdTalkHook)
}

func cdTurnHook(g *Game) {
	// Apply active subrace stance combat bonuses
	p := g.P
	if p == nil {
		return
	}
	st := g.cd()
	switch st.ActiveStance {
	case "mountain_giant":
		p.AddSt("stalwart", 2)
	case "desert_viper":
		p.AddSt("swift", 2)
	case "steppe_hawk":
		p.AddSt("farsight", 2)
	case "deep_mole":
		p.AddSt("nightvision", 2)
	}
}

func cdZoneHook(g *Game, zn *Zone, r *Rng) {
	if g == nil || zn == nil {
		return
	}
	sr := NewRng(fmt.Sprintf("%s:cd:%s", g.Seed, zn.Key))

	// 1. Subterranean Fungal Silkworm Farms (Z > 0)
	if zn.Z > 0 {
		cdSpawnSubterraneanSilkworms(g, zn, sr)
		return
	}

	// 2. Bioluminescent Coral Cultivation Domes (Underwater / Coastal)
	if zn.WaterAt(MW/2, MH/2) > 0 || zn.Biome == 0 || zn.Biome == 1 {
		cdSpawnCoralDomes(g, zn, sr)
	}

	// 3. Grand Kurultai Moots and Ancestral Memory Shrines (Wild / Steppe / Crags)
	if zn.Kind == "wild" || zn.Kind == "mountain" || zn.Kind == "hills" {
		cdSpawnKurultaiAndAncestralShrines(g, zn, sr)
	}
}

func cdSpawnSubterraneanSilkworms(g *Game, zn *Zone, sr *Rng) {
	if !sr.Chance(0.3) {
		return
	}
	fx, fy := 25+sr.Int(0, 30), 8+sr.Int(0, 8)
	if !zn.In(fx, fy) || !zn.At(fx, fy).D().Walk {
		return
	}

	// Place fungal silk racks
	for dx := -1; dx <= 1; dx++ {
		for dy := -1; dy <= 1; dy++ {
			x, y := fx+dx, fy+dy
			if zn.In(x, y) && zn.At(x, y).D().Walk && (dx != 0 || dy != 0) {
				zn.Set(x, y, TSilkwormRack)
				zn.Notes[y*MW+x] = "Trays of glowing cavern silkworms spinning shimmering ghost-silk. Bump to harvest."
			}
		}
	}

	// Subterranean weaver
	weaver := g.Spawn("human", zn.Level, sr)
	if weaver != nil {
		weaver.Name = "Deep Fungal Weaver"
		weaver.Role = "cd_weaver"
		weaver.Faction = "citizens"
		weaver.Desc = "Pale-skinned artisan with silver needle-shuttles, harvesting gossamer thread."
		weaver.X, weaver.Y = fx, fy
		zn.Actors = append(zn.Actors, weaver)
	}
}

func cdSpawnCoralDomes(g *Game, zn *Zone, sr *Rng) {
	if !sr.Chance(0.35) {
		return
	}
	dx, dy := 20+sr.Int(0, 40), 6+sr.Int(0, 12)
	for k := 0; k < 3; k++ {
		cx, cy := dx+k*3, dy
		if zn.In(cx, cy) {
			zn.Set(cx, cy, TCoralDome)
			zn.Notes[cy*MW+cx] = "A living nacreous coral dome glowing with turquoise light. Bump to harvest pearls."
		}
	}

	diver := g.Spawn("human", zn.Level, sr)
	if diver != nil {
		diver.Name = "Pearl-Reef Cultivator"
		diver.Role = "cd_diver"
		diver.Faction = "citizens"
		diver.Desc = "Glistening with salt and carrying a mesh pouch of glowing reef pearls."
		diver.X, diver.Y = dx+1, dy+1
		zn.Actors = append(zn.Actors, diver)
	}
}

func cdSpawnKurultaiAndAncestralShrines(g *Game, zn *Zone, sr *Rng) {
	if sr.Chance(0.25) {
		// Ancestral Memory Shrine
		sx, sy := 35, 10
		if zn.In(sx, sy) && zn.At(sx, sy).D().Walk {
			zn.Set(sx, sy, TMonument)
			zn.Notes[sy*MW+sx] = "Ancient Megalithic Shrine of Ancestral Memory. Meditate here to awaken subrace combat stances."
		}
	}

	if sr.Chance(0.2) {
		// Nomadic Kurultai / Moot encampment
		kx, ky := 20+sr.Int(0, 30), 8+sr.Int(0, 8)
		for dy := -2; dy <= 2; dy++ {
			for dx := -3; dx <= 3; dx++ {
				x, y := kx+dx, ky+dy
				if zn.In(x, y) && (dx == -3 || dx == 3 || dy == -2 || dy == 2) {
					zn.Set(x, y, TTent)
				}
			}
		}
		// Kurultai Chieftain
		chief := g.Spawn("human", zn.Level+2, sr)
		if chief != nil {
			chief.Name = "Grand Kurultai Khan"
			chief.Role = "cd_khan"
			chief.Faction = "citizens"
			chief.Desc = "Draped in snow-leopard furs, holding a horsetail battle standard under the peace pact."
			chief.X, chief.Y = kx, ky
			zn.Actors = append(zn.Actors, chief)
		}
	}
}

// cdBump handles interactive bumps for culture depth features
func cdBump(g *Game, x, y int) bool {
	zn := g.Cur
	if zn == nil || !zn.In(x, y) {
		return false
	}
	t := zn.At(x, y)
	st := g.cd()

	switch t {
	case TSilkwormRack:
		g.spend(800)
		st.SilkHarvests++
		g.give(&Item{
			ID: g.newID(), Kind: "ingredient", Name: "subterranean ghost-silk", Value: 25, Weight: 1, Glyph: '%', F: CCyan,
		})
		g.gainXP(20)
		g.Msg("{{C|You harvest delicate strands of shimmering subterranean ghost-silk!}}")
		return true

	case TCoralDome:
		g.spend(800)
		st.CoralPearls++
		g.give(&Item{
			ID: g.newID(), Kind: "treasure", Name: "bioluminescent coral pearl", Value: 35, Weight: 1, Glyph: '*', F: CCyan,
		})
		g.gainXP(25)
		g.Msg("{{C|You pry free a luminous turquoise coral pearl from the cultivation dome!}}")
		return true

	case TMonument:
		note := zn.Notes[y*MW+x]
		if strings.Contains(note, "Ancestral Memory") {
			g.spend(1000)
			// Unlock all stances upon communing
			st.KnownStances["mountain_giant"] = true
			st.KnownStances["desert_viper"] = true
			st.KnownStances["steppe_hawk"] = true
			st.KnownStances["deep_mole"] = true

			g.showPopup("Ancestral Memory Awakened",
				"You place your palms against the ancient weathered dolmen.\n\n"+
					"Echoes of bygone warrior dynasties flood your mind! You have unlocked all four Subrace Martial Stances:\n"+
					"• Stance of the Mountain Giant (Colossal armor & crushing blows)\n"+
					"• Stance of the Desert Viper (Blistering evasion & poison strikes)\n"+
					"• Stance of the Steppe Hawk (Eagle bow range & swift disengage)\n"+
					"• Stance of the Deep Mole (Armor-piercing & subterranean tremorsense)\n\n"+
					"Speak with the Khan or commune at shrines to adopt a stance.")
			g.gainXP(50)
			return true
		}
	}

	return false
}

func cdTalkHook(g *Game, a *Actor) []Option {
	var opts []Option
	if g == nil || a == nil {
		return opts
	}

	st := g.cd()

	switch a.Role {
	case "cd_khan":
		opts = append(opts, Option{
			Text: "{{Y|Pledge martial honor to adopt the Stance of the Steppe Hawk.}}",
			Do: func() {
				st.ActiveStance = "steppe_hawk"
				g.gainXP(30)
				g.talkMenu("\"May your arrows fly true across the wide steppes, warrior!\"")
			},
		})
		opts = append(opts, Option{
			Text: "{{Y|Pledge martial honor to adopt the Stance of the Mountain Giant.}}",
			Do: func() {
				st.ActiveStance = "mountain_giant"
				g.gainXP(30)
				g.talkMenu("\"Stand unmoved as the iron peaks against the storm!\"")
			},
		})

	case "cd_weaver":
		opts = append(opts, Option{
			Text: "{{C|Trade for shimmering Ghost-Silk Cowl (30 gold).}}",
			Do: func() {
				if !g.cuSpend(30) {
					g.talkMenu("\"Thirty gold for hand-woven ghost-silk.\"")
					return
				}
				g.give(&Item{
					ID: g.newID(), Kind: "armor", Name: "ghost-silk cowl", Slot: "head", AV: 1, DV: 2, Value: 40, Weight: 1, Glyph: '[', F: CCyan,
				})
				g.talkMenu("\"Woven in silence from cavern worms. It whispers away footsteps in the dark.\"")
			},
		})

	case "cd_diver":
		opts = append(opts, Option{
			Text: "{{C|Trade for Bioluminescent Coral Charm (25 gold).}}",
			Do: func() {
				if !g.cuSpend(25) {
					g.talkMenu("\"Twenty-five coppers for living reef coral.\"")
					return
				}
				g.give(&Item{
					ID: g.newID(), Kind: "trinket", Name: "bioluminescent coral charm", Value: 35, Weight: 1, Glyph: '*', F: CCyan,
				})
				g.talkMenu("\"It glows warmly when cold currents sweep the deep!\"")
			},
		})
	}

	return opts
}
