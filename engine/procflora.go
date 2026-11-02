package engine

import (
	"fmt"
)

// Procedural biome- and genre-skinned flora, curiosities, and ambient micro-structures (prefix pf).
// Uses deterministic side RNG keyed by world seed + zone key so core RNG streams stay untouched.

func init() {
	zoneHooks = append(zoneHooks, pfZoneHook)
	sideHooks = append(sideHooks, pfSideHook)
}

type pfSpec struct {
	Kind  string
	Glyph rune
	Color byte
	Value int
	Names map[string]string
	Lore  string
}

var pfBiomeFlora = map[int][]pfSpec{
	1: { // hot desert
		{Kind: "ingredient", Glyph: '♣', Color: CYellow, Value: 18,
			Names: nm("sun-glass lotus", "steampunk", "lens-petal lotus", "dieselpunk", "kerosene-thistle", "atompunk", "trinitite-bloom", "cyberpunk", "silicon-succulent", "biopunk", "blister-agave", "solarpunk", "heliotrope-succulent", "mythic", "solar-lotus", "wasteland", "slag-cactus", "twilight", "amber-rose"),
			Lore:  "Translucent petals fused from wind-blown silica and desert dew."},
		{Kind: "oddity", Glyph: '♦', Color: COrange, Value: 28,
			Names: nm("fulgurite root", "steampunk", "voltaic sand-tube", "cyberpunk", "arc-glass nodule", "atompunk", "vitrified core", "wasteland", "lightning-slag", "twilight", "thunder-amber"),
			Lore:  "A hollow tube of desert sand flash-melted where a bolt struck the dunes."},
	},
	2: { // cold desert / steppe
		{Kind: "ingredient", Glyph: '♣', Color: CYellow, Value: 15,
			Names: nm("steppe wormwood", "steampunk", "brass-sage", "dieselpunk", "tarmac-scrub", "cyberpunk", "wire-tumbleweed", "biopunk", "sinew-shrub", "wasteland", "ash-sage", "twilight", "grave-artemisia"),
			Lore:  "Bitter silver-grey leaves that thrive in dry, whistling winds."},
	},
	3: { // savanna
		{Kind: "ingredient", Glyph: '♣', Color: COrange, Value: 16,
			Names: nm("lion's-mane Acacia pod", "steampunk", "clockwork dandelion", "cyberpunk", "fiber-optic burr", "biopunk", "nectar-gland pod", "solarpunk", "sun-savanna gourd", "wasteland", "razor-seed pod", "twilight", "gilded seed-pod"),
			Lore:  "Splits open with a dry snap, scattering golden down on the thermals."},
	},
	4: { // grassland
		{Kind: "ingredient", Glyph: '♣', Color: CGreen, Value: 14,
			Names: nm("prairie star-clover", "steampunk", "spring-coil clover", "dieselpunk", "grease-flax", "cyberpunk", "neon-clover", "solarpunk", "honey-trefoil", "mythic", "asphodel sprig", "wasteland", "mutant rye-head", "twilight", "pale asphodel"),
			Lore:  "Four-part leaves that fold tight before a storm breaks."},
	},
	5: { // tropical seasonal forest
		{Kind: "ingredient", Glyph: '♣', Color: CMagenta, Value: 22,
			Names: nm("monsoon-orchid", "steampunk", "aether-orchid", "cyberpunk", "holo-orchid", "biopunk", "venom-calyx", "solarpunk", "rain-catcher bloom", "wasteland", "acid-orchid", "twilight", "wax-orchid"),
			Lore:  "Drinks directly from humid air through silver-velvet aerial roots."},
	},
	6: { // temperate deciduous forest
		{Kind: "ingredient", Glyph: '♣', Color: CGreen, Value: 16,
			Names: nm("hart's-tongue fern", "steampunk", "copper-frond fern", "cyberpunk", "circuit-fern", "biopunk", "vein-frond", "solarpunk", "canopy-lace fern", "mythic", "silver-bough sprig", "wasteland", "strontium-fern", "twilight", "autumn-sorrow frond"),
			Lore:  "Unfurls in damp woodland shade, its underside dusted with russet spores."},
	},
	7: { // tropical rainforest
		{Kind: "ingredient", Glyph: '*', Color: CMagenta, Value: 30,
			Names: nm("jaguar-liana sap", "steampunk", "phlogiston-latex", "cyberpunk", "neuro-stimm sap", "biopunk", "adrenal-nectar", "solarpunk", "canopy-elixir", "wasteland", "mutagen-sap", "twilight", "crimson-resin"),
			Lore:  "Milky crimson sap tapped from a strangler vine; tingles sharply on the tongue."},
	},
	8: { // temperate rainforest
		{Kind: "ingredient", Glyph: '♣', Color: CCyan, Value: 20,
			Names: nm("ghost-pipe lichen", "steampunk", "condenser-moss", "cyberpunk", "neon-moss", "biopunk", "lymph-moss", "solarpunk", "dew-sponge lichen", "wasteland", "rad-lichen", "twilight", "shroud-moss"),
			Lore:  "Pale, translucent stalks that glow faintly in old-growth fog."},
	},
	9: { // taiga / boreal
		{Kind: "ingredient", Glyph: '♣', Color: CDarkGreen, Value: 18,
			Names: nm("resin-pine cone", "steampunk", "pitch-valve cone", "dieselpunk", "turpentine-burr", "cyberpunk", "cryo-resin node", "biopunk", "amber-cyst", "wasteland", "tar-cone", "twilight", "black-spruce gall"),
			Lore:  "Heavy with aromatic amber pitch that burns with a clear, steady flame."},
	},
	10: { // tundra
		{Kind: "ingredient", Glyph: '♣', Color: CCyan, Value: 22,
			Names: nm("frost-gentian", "steampunk", "mercury-gentian", "cyberpunk", "cryo-bloom", "atompunk", "heavy-water lily", "biopunk", "antifreeze-blossom", "mythic", "hyperborean lotus", "wasteland", "permafrost-thistle", "twilight", "rime-bell"),
			Lore:  "Deep blue chalice-petals warm enough to melt a ring in the snow."},
	},
	11: { // glacier
		{Kind: "oddity", Glyph: '♦', Color: CWhite, Value: 34,
			Names: nm("glacier-heart prism", "steampunk", "aether-ice lens", "cyberpunk", "superconductor shard", "atompunk", "cryo-isotope core", "mythic", "boreal diamond", "wasteland", "blue-ice core", "twilight", "frozen tear"),
			Lore:  "Compressed over millennia until it refracts light like cut sapphire."},
	},
	12: { // wetland / marsh
		{Kind: "ingredient", Glyph: '♣', Color: CGreen, Value: 19,
			Names: nm("bog-lantern reed", "steampunk", "marsh-gas bulb", "dieselpunk", "sump-cattail", "cyberpunk", "phosphor-reed", "biopunk", "bile-bladderwort", "wasteland", "sludge-lotus", "twilight", "corpse-candle reed"),
			Lore:  "Hollow pith filled with cool marsh-phosphor that draws night insects."},
	},
	13: { // volcanic / ash
		{Kind: "ingredient", Glyph: '♣', Color: CRed, Value: 26,
			Names: nm("ash-orchid", "steampunk", "cinder-valve bloom", "dieselpunk", "slag-poppy", "atompunk", "fission-poppy", "cyberpunk", "thermal-filament flower", "biopunk", "magma-polyp", "mythic", "phoenix-feather reed", "wasteland", "cinder-thistle", "twilight", "pyre-lily"),
			Lore:  "Roots in cooling basalt cracks, drinking sulfur and geothermal steam."},
	},
	14: { // crystal
		{Kind: "oddity", Glyph: '♦', Color: CMagenta, Value: 36,
			Names: nm("singing-quartz cluster", "steampunk", "resonator-geode", "cyberpunk", "piezo-crystal array", "atompunk", "scintillator-shard", "solarpunk", "Sun-chime crystal", "mythic", "star-choir shard", "wasteland", "shatter-prism", "twilight", "vesper-crystal"),
			Lore:  "Hums a faint, pure fifth whenever metal passes within a handspan."},
	},
	15: { // fungal
		{Kind: "ingredient", Glyph: '♣', Color: CMagenta, Value: 24,
			Names: nm("violet-gill truffle", "steampunk", "bellows-puffball", "cyberpunk", "myco-data node", "biopunk", "marrow-truffle", "atompunk", "rad-morel", "wasteland", "glow-cap", "twilight", "grave-truffle"),
			Lore:  "Dense, marbled flesh that smells of rain-wet iron and nutmeg."},
	},
	16: { // cinder / scorched
		{Kind: "oddity", Glyph: '♦', Color: COrange, Value: 30,
			Names: nm("ember-glass nodule", "steampunk", "clinker-pearl", "dieselpunk", "coke-slag pearl", "atompunk", "fallout-trinitite", "cyberpunk", "plasma-bead", "wasteland", "scorch-pearl", "twilight", "cinder-reliquary"),
			Lore:  "Warm to the touch even in rain, holding a trapped spark at its centre."},
	},
}

var pfDeepSpecs = []pfSpec{
	{Kind: "ingredient", Glyph: '♣', Color: CCyan, Value: 26,
		Names: nm("blind-cave morel", "steampunk", "damp-valve fungus", "cyberpunk", "optic-fiber mycelium", "biopunk", "nerve-cordyceps", "atompunk", "uranium-lichen", "wasteland", "vault-mould", "twilight", "catacomb-lily"),
		Lore:  "Grows in total darkness where subterranean drips echo through limestone."},
	{Kind: "oddity", Glyph: '♦', Color: CMagenta, Value: 35,
		Names: nm("loadstone geode", "steampunk", "magneto-nodule", "cyberpunk", "ferro-fluid cyst", "biopunk", "calcified heart-stone", "mythic", "earth-marrow stone", "wasteland", "bunker-slag core", "twilight", "ossuary-opal"),
		Lore:  "Cracked open to reveal needle-like crystals that tug toward true north."},
}

var pfSkySpecs = []pfSpec{
	{Kind: "ingredient", Glyph: '♣', Color: CCyan, Value: 28,
		Names: nm("sky-feather fern", "steampunk", "aether-plume frond", "cyberpunk", "ion-ribbon plant", "biopunk", "gas-bladder bromeliad", "solarpunk", "cloud-anchor vine", "mythic", "zephyr-lily", "wasteland", "stratosphere-lichen", "twilight", "starlight-thistle"),
		Lore:  "Buoyant, hollow stems that trail in high-altitude currents without soil."},
	{Kind: "oddity", Glyph: '♦', Color: CYellow, Value: 38,
		Names: nm("thunder-egg hailstone", "steampunk", "lightning-capacitor pebble", "cyberpunk", "static-charge core", "mythic", "sky-anvil shard", "wasteland", "ion-slag bead", "twilight", "fallen-star fragment"),
		Lore:  "Concentric shells of rime and ozone-charged glass spun inside a stormhead."},
}

var pfSeaSpecs = []pfSpec{
	{Kind: "ingredient", Glyph: '♣', Color: CCyan, Value: 27,
		Names: nm("abyssal kelp-bulb", "steampunk", "pressure-bladder kelp", "cyberpunk", "biolum-ribbon", "biopunk", "gill-frond", "mythic", "nereid's-tress", "wasteland", "brine-rot kelp", "twilight", "drowned-lantern weed"),
		Lore:  "Filled with buoyant oil that glows cold blue under crushing depth."},
	{Kind: "oddity", Glyph: '♦', Color: CWhite, Value: 40,
		Names: nm("abyssal kelp-pearl", "steampunk", "nautilus-valve pearl", "cyberpunk", "sonar-nacre sphere", "biopunk", "chitin-ambergris", "mythic", "pelagic moon-pearl", "wasteland", "tar-nacre nodule", "twilight", "brine-reliquary pearl"),
		Lore:  "Layered nacre secreted around a shard of ancient shipwreck bronze."},
}

func pfPickSpec(zn *Zone, r *Rng) pfSpec {
	if zn.Underwater {
		return Pick(r, pfSeaSpecs)
	}
	if zn.Sky {
		return Pick(r, pfSkySpecs)
	}
	if zn.Z > 0 || zn.Kind == "dungeon" || zn.Kind == "underdark" {
		if r.Chance(0.65) {
			return Pick(r, pfDeepSpecs)
		}
	}
	if list, ok := pfBiomeFlora[zn.Biome]; ok && len(list) > 0 {
		return Pick(r, list)
	}
	return pfBiomeFlora[4][0]
}

func pfMakeItem(g *Game, zn *Zone, r *Rng) *Item {
	sp := pfPickSpec(zn, r)
	name := sp.Names[g.Genre]
	if name == "" {
		name = sp.Names[""]
	}
	lvlBonus := maxi(0, zn.Level*2)
	it := &Item{
		ID:     g.newID(),
		Kind:   sp.Kind,
		Name:   name,
		Glyph:  sp.Glyph,
		F:      sp.Color,
		Weight: 1,
		Value:  sp.Value + lvlBonus,
		Count:  1,
		Lore:   sp.Lore,
	}
	return it
}

var pfMicroNotes = map[string][]string{
	"": {
		"Carved boundary-marks weathered into the stone by an older surveying party.",
		"A hollow cairn of stacked flat stones sheltering dried tinder for passing travellers.",
		"Wind-chimes of river-shell and bronze wire hanging from a weathered stake.",
	},
	"steampunk": {
		"A riveted brass pressure-marker stamped with an imperial survey cipher.",
		"An abandoned surveyor's tripod with a cracked aether-level vial.",
	},
	"dieselpunk": {
		"A stencilled fuel-cache marker half-buried in oil-stained gravel.",
		"Spent brass shell-casings stacked neatly atop a concrete boundary post.",
	},
	"atompunk": {
		"A yellow-enamel dosimeter post ticking softly in the breeze.",
		"A geodesic survey beacon with a faded civil-defence emblem.",
	},
	"cyberpunk": {
		"A solar-fed mesh repeater node blinking a slow amber heartbeat.",
		"A dead drop locker scarred with old street-gang tags and QR glyphs.",
	},
	"biopunk": {
		"A chitinous boundary-polyp secreting faint pheromonal territory markers.",
		"A shed carapace-husk anchored to the ground by calcified tendons.",
	},
	"solarpunk": {
		"A ceramic wind-harp tuned to hum when the afternoon thermals rise.",
		"A community seed-cache sealed in beeswax inside a hollow stone.",
	},
	"mythic": {
		"A boundary herm garlanded with dried laurel and libation-stained stone.",
		"An augury-slab scored with the flight-paths of sacred birds.",
	},
	"wasteland": {
		"A warning totem of rusted rebar and sun-bleached road signs.",
		"A scavengers' chalk-cipher scratched on a slab of reinforced concrete.",
	},
	"twilight": {
		"A tarnished gilt reliquary-post whose saint's name has been chiselled away.",
		"A cracked sundial inscribed: 'All hours return to ash.'",
	},
}

func pfZoneHook(g *Game, zn *Zone, r *Rng) {
	if zn == nil || zn.Kind == "town" {
		return
	}
	rr := NewRng(g.W.Seed + ":pf:" + zn.Key)

	// 1. Spawn 1-3 procedural biome/genre-skinned flora or curiosity items
	count := rr.Int(1, 3)
	for i := 0; i < count; i++ {
		x, y, ok := zn.FreeSpot(rr, func(x, y int) bool {
			t := zn.At(x, y)
			return t != TRoad && t != TBridge && t != TDown && t != TUp
		})
		if !ok {
			continue
		}
		it := pfMakeItem(g, zn, rr)
		it.X, it.Y = x, y
		zn.Items = append(zn.Items, it)
	}

	// 2. Add a procedural ambient micro-structure note
	if x, y, ok := zn.FreeSpot(rr, func(x, y int) bool {
		return x > 6 && x < MW-7 && y > 3 && y < MH-4
	}); ok {
		notes := pfMicroNotes[g.Genre]
		if len(notes) == 0 {
			notes = pfMicroNotes[""]
		}
		if zn.Notes == nil {
			zn.Notes = map[int]string{}
		}
		idx := y*MW + x
		if zn.Notes[idx] == "" {
			zn.Notes[idx] = Pick(rr, notes)
		}
	}
}

func pfSideHook(g *Game) []string {
	if g == nil || g.Cur == nil {
		return nil
	}
	zn := g.Cur
	rr := NewRng(fmt.Sprintf("%s:pf:side:%s", g.W.Seed, zn.Key))
	var atmo string
	switch {
	case zn.Underwater:
		atmo = Pick(rr, []string{
			"Currents: cold pelagic drift",
			"Light: bioluminescent motes",
			"Pressure: deep-trench hush",
		})
	case zn.Sky:
		atmo = Pick(rr, []string{
			"Updraft: thin singing thermals",
			"Horizon: cloud-decks below",
			"Air: crisp ozone and rime",
		})
	case zn.Z > 0:
		atmo = Pick(rr, []string{
			"Strata: damp mineral echo",
			"Air: cool subterranean draught",
			"Stone: weeping limestone seams",
		})
	default:
		switch zn.Biome {
		case 1, 2:
			atmo = "Air: dry alkali dust and heat-shimmer"
		case 3, 4:
			atmo = "Wind: long grass rippling in waves"
		case 5, 6:
			atmo = "Canopy: dappled leaf-shadow and loam"
		case 7, 8, 12:
			atmo = "Air: humid peat-mist and frog-song"
		case 9, 10, 11:
			atmo = "Wind: sharp needle-rime and frost"
		case 13, 16:
			atmo = "Air: warm sulfur drift and fine ash"
		case 14:
			atmo = "Light: prismatic Hum on the breeze"
		case 15:
			atmo = "Air: drifting violet spore-haze"
		default:
			atmo = "Air: quiet road-dust and wild herbs"
		}
	}
	return []string{"{{K|" + atmo + "}}"}
}
