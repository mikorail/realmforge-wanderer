package engine

import (
	"fmt"
	"strings"
)

// Procedural character, creature, and environment asset variations (prefix pa).
// Uses deterministic side RNG streams keyed by world seed + zone key so core RNG streams stay untouched.

var paTemplates = []CTpl{
	{ID: "stag", Glyph: 'd', F: CBrown, Lvl: 2, Attr: [6]int{15, 16, 14, 6, 10, 8}, HP: 14, DV: 2, Speed: 120, Faction: "beasts", Biomes: []int{3, 4, 5, 6, 9, 10}, Where: "s", Body: "quadruped", Bite: [3]int{1, 6, 1},
		Names: nm("great stag", "steampunk", "brass-antlered hart", "dieselpunk", "barbed-wire elk", "atompunk", "rad-stag", "cyberpunk", "chrome-hart", "biopunk", "crown-splice", "mythic", "ceryneian hind", "wasteland", "two-headed stag", "twilight", "white hart"), Desc: "Crowned with branching antlers, poised to bolt or gore."},
	{ID: "vulture", Glyph: 'v', F: CDarkGray, Lvl: 2, Attr: [6]int{10, 17, 12, 6, 10, 4}, HP: 10, DV: 3, Speed: 125, Faction: "beasts", Biomes: []int{1, 2, 3, 13}, Where: "sk", Body: "avian", Flags: "fly pack", Bite: [3]int{1, 4, 1},
		Names: nm("carrion condor", "steampunk", "clockwork buzzard", "cyberpunk", "scav-drone", "biopunk", "corpse-kite", "atompunk", "featherless vulture", "wasteland", "ash buzzard", "twilight", "gallows kite"), Desc: "Circles patiently above the dying."},
	{ID: "mantis", Glyph: 'I', F: CGreen, Lvl: 4, Attr: [6]int{17, 19, 14, 6, 12, 6}, HP: 20, AV: 3, DV: 2, Speed: 115, Faction: "vermin", Biomes: []int{5, 6, 7, 8}, Where: "s", Body: "insect", Flags: "bleed", Bite: [3]int{2, 4, 2},
		Names: nm("scythe mantis", "steampunk", "clockwork reaper", "cyberpunk", "blade-mantis drone", "biopunk", "razor-mantid", "wasteland", "rust mantis", "twilight", "orchid stalker"), Desc: "Serrated forelimbs folded in lethal prayer."},
	{ID: "toad", Glyph: 'u', F: CDarkGreen, Lvl: 3, Attr: [6]int{16, 12, 16, 4, 10, 4}, HP: 18, AV: 2, Speed: 95, Faction: "beasts", Biomes: []int{7, 8, 12}, Where: "s", Body: "quadruped", Flags: "aquatic poison", Bite: [3]int{1, 6, 1},
		Names: nm("mire bull-toad", "cyberpunk", "sludge-hopper", "biopunk", "bloat-toad", "atompunk", "three-eyed toad", "wasteland", "bile toad", "twilight", "boggulper"), Desc: "Warty throat-sac pulsing with acrid slime."},
	{ID: "basilisk", Glyph: 'D', F: CYellow, Lvl: 7, Attr: [6]int{21, 12, 22, 8, 16, 10}, HP: 48, AV: 7, Speed: 90, Faction: "beasts", Biomes: []int{1, 2, 13, 14}, Where: "sdr", Body: "quadruped", Flags: "poison", Bite: [3]int{2, 6, 3},
		Names: nm("stone basilisk", "cyberpunk", "stasis-gaze drake", "steampunk", "calcifier engine", "biopunk", "petrifier lizard", "mythic", "cockatrice", "wasteland", "glass-eye drake", "twilight", "salt basilisk"), Desc: "Eight legs and a heavy, mineral stare."},
	{ID: "treant", Glyph: '♣', F: CDarkGreen, Lvl: 8, Attr: [6]int{24, 8, 25, 10, 18, 10}, HP: 64, AV: 8, Speed: 75, Faction: "spirits", Biomes: []int{5, 6, 7, 8, 9}, Where: "s", Body: "golem", Flags: "regen", Bite: [3]int{2, 8, 3},
		Names: nm("ancient treant", "steampunk", "ironwood hulk", "cyberpunk", "arbor-mech", "biopunk", "strangler-colossus", "solarpunk", "grove-warden", "mythic", "dryad-oak", "wasteland", "charred deadwood", "twilight", "gallows-ent"), Desc: "Centuries of ring-wood knotted into slow, crushing limbs."},
	{ID: "centipede", Glyph: 'j', F: COrange, Lvl: 3, Attr: [6]int{14, 17, 13, 2, 10, 4}, HP: 15, AV: 3, Speed: 120, Faction: "vermin", Biomes: []int{5, 7, 8, 12, 15}, Where: "sdr", Body: "insect", Flags: "poison", Bite: [3]int{1, 6, 1},
		Names: nm("giant centipede", "cyberpunk", "cable-crawler", "steampunk", "chain-skitterer", "biopunk", "hundred-spine", "wasteland", "rust centipede", "twilight", "coffin-centipede"), Desc: "A yard of lacquered plates and venomous forcipules."},
	{ID: "mammoth", Glyph: 'M', F: CBrown, Lvl: 8, Attr: [6]int{26, 8, 26, 6, 14, 8}, HP: 72, AV: 6, Speed: 85, Faction: "beasts", Biomes: []int{9, 10, 11}, Where: "s", Body: "quadruped", Flags: "frost", Bite: [3]int{3, 6, 4},
		Names: nm("woolly mammoth", "steampunk", "steam-behemoth", "dieselpunk", "trench-mastodon", "cyberpunk", "cryo-strider", "biopunk", "tusk-titan", "wasteland", "rad-mastodon", "twilight", "rime-behemoth"), Desc: "Shaggy flanks and curved ivory tusks the length of spears."},
	{ID: "cactoid", Glyph: 'f', F: CGreen, Lvl: 3, Attr: [6]int{14, 12, 18, 4, 12, 4}, HP: 20, AV: 4, Speed: 80, Faction: "fungi", Biomes: []int{1, 2}, Where: "s", Body: "humanoid", Flags: "ranged bleed", Bite: [3]int{1, 6, 1},
		Names: nm("needle cactoid", "cyberpunk", "flechette-pod", "biopunk", "thorn-husk", "atompunk", "spine-mutant", "wasteland", "razor-cactus", "twilight", "blood-thorn"), Desc: "Ribbed green flesh bristling with bone-white spines."},
	{ID: "phoenix", Glyph: 'P', F: COrange, Lvl: 10, Attr: [6]int{20, 20, 20, 14, 20, 16}, HP: 76, AV: 4, DV: 4, Speed: 130, Faction: "elementals", Biomes: []int{13, 16}, Where: "sk", Body: "avian", Flags: "fly fire regen", Bite: [3]int{2, 8, 4}, Elem: "fire",
		Names: nm("ember phoenix", "cyberpunk", "plasma-valkyrie", "steampunk", "phlogiston-raptor", "atompunk", "atomic firebird", "mythic", "immortal Bennu", "wasteland", "cinder-roc", "twilight", "dying-sun bird"), Desc: "Feathers of white-hot ash that ignite the air with every wingbeat."},
	{ID: "bat", Glyph: 'b', F: CDarkGray, Lvl: 1, Attr: [6]int{8, 19, 9, 4, 10, 4}, HP: 6, DV: 4, Speed: 135, Faction: "vermin", Where: "dr", Body: "avian", Flags: "fly pack bleed", Bite: [3]int{1, 3, 0},
		Names: nm("cave vampire-bat", "cyberpunk", "sonar-flitter", "steampunk", "bellows-bat", "biopunk", "leech-wing", "wasteland", "blind flitter", "twilight", "vesper bat"), Desc: "Leathery wings and needle teeth clicking in the dark."},
	{ID: "worm", Glyph: 'w', F: CBrown, Lvl: 6, Attr: [6]int{22, 10, 22, 2, 12, 4}, HP: 46, AV: 6, Speed: 85, Faction: "vermin", Where: "d", Body: "serpent", Flags: "burrow acid", Bite: [3]int{2, 6, 3}, Elem: "acid",
		Names: nm("rock-borer worm", "cyberpunk", "ferro-auger worm", "steampunk", "drill-wyrm", "biopunk", "acid-gullet", "wasteland", "slag-worm", "twilight", "marrow-borer"), Desc: "A grinding ring of teeth that chews living granite into slurry."},
	{ID: "mimic", Glyph: '■', F: COrange, Lvl: 5, Attr: [6]int{20, 12, 20, 8, 14, 6}, HP: 34, AV: 6, Speed: 90, Faction: "eldritch", Where: "dr", Body: "ooze", Flags: "bleed", Bite: [3]int{2, 6, 3},
		Names: nm("coffer mimic", "cyberpunk", "decoy-locker", "steampunk", "spring-jaw trunk", "biopunk", "flesh-chest", "wasteland", "trap-crate", "twilight", "reliquary maw"), Desc: "Iron-bound oak until the lid splits into a wet row of fangs."},
	{ID: "stalker", Glyph: 'n', F: CMagenta, Lvl: 6, Attr: [6]int{18, 20, 15, 10, 14, 8}, HP: 28, DV: 5, Speed: 125, Faction: "eldritch", Where: "dr", Body: "quadruped", Flags: "bleed", Bite: [3]int{2, 6, 2},
		Names: nm("phase-stalker", "cyberpunk", "optic-camo hound", "biopunk", "chameleon-fiend", "mythic", "unseen hound", "wasteland", "mirage prowler", "twilight", "veil-hound"), Desc: "Flickers between here and a half-step sideways."},
	{ID: "fungalbeast", Glyph: 'Q', F: CMagenta, Lvl: 7, Attr: [6]int{22, 8, 24, 6, 16, 6}, HP: 56, AV: 5, Speed: 80, Faction: "fungi", Biomes: []int{15}, Where: "sd", Body: "golem", Flags: "poison regen", Bite: [3]int{2, 6, 3},
		Names: nm("spore-behemoth", "biopunk", "mycelial juggernaut", "cyberpunk", "bloom-hulk", "atompunk", "rad-puffball hulk", "wasteland", "bloat-hulk", "twilight", "cordyceps giant"), Desc: "A lumbering mass of shelf-fungi venting choking violet clouds."},
	{ID: "mindleech", Glyph: 'ψ', F: CCyan, Lvl: 7, Attr: [6]int{12, 16, 16, 18, 20, 16}, HP: 36, DV: 3, Speed: 105, Faction: "eldritch", Where: "dr", Body: "tentacled", Flags: "leech", Bite: [3]int{1, 8, 3},
		Names: nm("mind-leech", "cyberpunk", "neuro-jack parasite", "biopunk", "synapse-worm", "mythic", "oneiric-eater", "wasteland", "psionic crawler", "twilight", "memory-leech"), Desc: "Translucent tendrils that hum behind your eyes before they strike."},
	{ID: "sylph", Glyph: 's', F: CCyan, Lvl: 5, Attr: [6]int{10, 22, 12, 12, 16, 14}, HP: 22, DV: 6, Speed: 135, Faction: "elementals", Where: "k", Body: "humanoid", Flags: "fly shock", Bite: [3]int{1, 6, 2}, Elem: "shock",
		Names: nm("gale sylph", "cyberpunk", "ion-vortex", "steampunk", "aether-elemental", "solarpunk", "thermal-dancer", "mythic", "aura-nymph", "wasteland", "dust-devil spirit", "twilight", "sighing wraith"), Desc: "Spun from high-altitude ice and singing wind."},
	{ID: "wyvern", Glyph: 'W', F: CDarkGreen, Lvl: 7, Attr: [6]int{22, 16, 20, 8, 14, 10}, HP: 52, AV: 4, DV: 2, Speed: 115, Faction: "beasts", Biomes: []int{10, 13}, Where: "sk", Body: "avian", Flags: "fly poison", Bite: [3]int{2, 6, 3},
		Names: nm("crag wyvern", "cyberpunk", "strike-vtoldrone", "steampunk", "steam-drake", "biopunk", "wing-viper", "mythic", "lindworm", "wasteland", "scorch-wyvern", "twilight", "ash-drake"), Desc: "Bat-winged serpent with a barbed, dripping tail-stinger."},
	{ID: "cloudjelly", Glyph: 'J', F: CMagenta, Lvl: 4, Attr: [6]int{12, 14, 16, 4, 12, 4}, HP: 24, DV: 2, Speed: 90, Faction: "beasts", Where: "k", Body: "tentacled", Flags: "fly shock", Bite: [3]int{1, 6, 2}, Elem: "shock",
		Names: nm("sky-medusa", "cyberpunk", "static-blimp", "biopunk", "hydrogen-polyp", "solarpunk", "cloud-bell", "wasteland", "rad-jelly", "twilight", "drift-shroud"), Desc: "A gas-filled bell trailing thirty feet of stinging filaments."},
	{ID: "siren", Glyph: 'N', F: CCyan, Lvl: 5, Attr: [6]int{14, 18, 14, 14, 16, 18}, HP: 28, DV: 3, Speed: 110, Faction: "beasts", Where: "e", Body: "humanoid", Flags: "aquatic bleed", Bite: [3]int{1, 6, 2},
		Names: nm("abyssal siren", "cyberpunk", "sonar-mimic", "biopunk", "lure-mermaid", "mythic", "nereid-huntress", "wasteland", "brine-witch", "twilight", "drowned cantor"), Desc: "Sings in the pressure-dark with a jaw that unhinges."},
	{ID: "angler", Glyph: 'a', F: CYellow, Lvl: 6, Attr: [6]int{20, 14, 20, 4, 12, 4}, HP: 38, AV: 4, Speed: 105, Faction: "beasts", Where: "e", Body: "fish", Flags: "aquatic bleed", Bite: [3]int{2, 6, 3},
		Names: nm("lantern-angler", "cyberpunk", "strobe-sub", "biopunk", "lure-maw", "wasteland", "glow-jaw", "twilight", "trench-gulper"), Desc: "A bobbing cold spark in the black water, then a cage of glass-clear teeth."},
	{ID: "leviathan", Glyph: 'L', F: CDarkBlue, Lvl: 11, Attr: [6]int{28, 12, 28, 10, 18, 12}, HP: 98, AV: 8, Speed: 95, Faction: "eldritch", Where: "e", Body: "serpent", Flags: "aquatic boss", Bite: [3]int{3, 8, 5},
		Names: nm("trench leviathan", "cyberpunk", "abyssal dreadnought", "biopunk", "pelagic titan", "mythic", "charybdis-spawn", "wasteland", "abyss-maw", "twilight", "drowned god-wyrm"), Desc: "Coils thick as lighthouse towers sliding through the silt."},
	{ID: "gargoyle", Glyph: 'Y', F: CGray, Lvl: 5, Attr: [6]int{19, 14, 20, 6, 14, 6}, HP: 32, AV: 7, Speed: 105, Faction: "constructs", Where: "rk", Body: "avian", Flags: "fly", Bite: [3]int{1, 8, 2},
		Names: nm("stone gargoyle", "steampunk", "brass gargoyle", "cyberpunk", "perch-interceptor", "biopunk", "calcified-harpy", "mythic", "karyatid-fiend", "wasteland", "concrete gargoyle", "twilight", "weeping gargoyle"), Desc: "Rainwater drips from its granite fangs as it unfolds from the cornice."},
	{ID: "mummy", Glyph: 'z', F: CYellow, Lvl: 6, Attr: [6]int{19, 10, 22, 10, 16, 10}, HP: 40, AV: 4, Speed: 85, Faction: "undead", Where: "r", Body: "humanoid", Flags: "undead poison", Bite: [3]int{2, 6, 2},
		Names: nm("embalmed priest", "cyberpunk", "cryo-wrapped exec", "steampunk", "wax-sealed revenant", "biopunk", "resin-husk", "mythic", "tomb-kheru", "wasteland", "duct-wrapped ghoul", "twilight", "gilded mummy"), Desc: "Wrapped in bitumen-soaked linen inscribed with older laws."},
	{ID: "phantom", Glyph: 'p', F: CCyan, Lvl: 6, Attr: [6]int{10, 20, 14, 14, 18, 14}, HP: 26, DV: 6, Speed: 115, Faction: "undead", Where: "dr", Body: "humanoid", Flags: "undead fly frost", Bite: [3]int{1, 8, 3}, Elem: "frost",
		Names: nm("weeping phantom", "cyberpunk", "corrupted engram", "steampunk", "aether-spectre", "atompunk", "shadow-burn ghost", "mythic", "eidolon", "wasteland", "fallout spectre", "twilight", "mourning shade"), Desc: "A cold outline of someone who died with unfinished business."},
	{ID: "lich", Glyph: 'L', F: CMagenta, Lvl: 11, Attr: [6]int{16, 16, 22, 22, 24, 20}, HP: 78, AV: 6, DV: 3, Speed: 105, Faction: "undead", Where: "r", Body: "humanoid", Flags: "undead ranged boss frost", Bite: [3]int{2, 6, 4}, Elem: "frost",
		Names: nm("arch-lich", "cyberpunk", "undying AI-hierophant", "steampunk", "clockwork necromancer", "biopunk", "immortal-strain", "mythic", "deathless magus", "wasteland", "vault-necromancer", "twilight", "ash-pontiff"), Desc: "A jewel-eyed skull in rotted regalia, still issuing decrees to the dust."},
}

func init() {
	for i := range paTemplates {
		tplByID[paTemplates[i].ID] = &paTemplates[i]
	}
	zoneHooks = append(zoneHooks, paZoneHook)
	talkHooks = append(talkHooks, paTalkHook)
}

type paMorph struct {
	Prefix string
	Color  byte
	HP     int // percentage delta e.g. +25 or -20
	AV, DV int
	Speed  int
	Elem   string
	Flag   string
	Trait  string
}

var paCommonMorphs = []paMorph{
	{Prefix: "elder", Color: CGray, HP: 30, AV: 2, Speed: -5, Trait: "Its hide is thick with the scars of many winters."},
	{Prefix: "albino", Color: CWhite, HP: 10, DV: 2, Speed: 5, Trait: "Pigmentless skin and pale pink eyes gleam in the gloom."},
	{Prefix: "battle-scarred", Color: CDarkRed, HP: 20, AV: 1, Trait: "Old spear-notches and healed burns cross its flank."},
	{Prefix: "rabid", Color: CRed, HP: -10, DV: -1, Speed: 15, Flag: "bleed", Trait: "Flecked with bloody foam, it lunges at every shadow."},
	{Prefix: "alpha", Color: CYellow, HP: 35, AV: 1, DV: 1, Speed: 5, Trait: "Broad-chested and dominant, larger than any of its pack."},
	{Prefix: "runt", Color: CBrown, HP: -25, DV: 3, Speed: 12, Trait: "Undersized and wiry, surviving by sheer skittish speed."},
	{Prefix: "two-headed", Color: CMagenta, HP: 25, AV: 1, Trait: "Twin necks snap and hiss at each other between bites."},
	{Prefix: "chitin-plated", Color: CDarkGreen, HP: 15, AV: 3, Speed: -5, Trait: "Overlapping plates of ridged horn armour its back."},
	{Prefix: "venom-gorged", Color: CGreen, HP: 10, Elem: "poison", Flag: "poison", Trait: "Swollen glands along its jaw weep chartreuse venom."},
	{Prefix: "phase-touched", Color: CDarkMagenta, DV: 4, Speed: 10, Trait: "Its edges blur and double whenever you try to fix your gaze."},
}

func paBiomeMorph(biome, z int, genre string, r *Rng) paMorph {
	if r.Chance(0.35) {
		switch genre {
		case "steampunk":
			return paMorph{Prefix: "brass-grafted", Color: COrange, HP: 20, AV: 2, Trait: "Riveted brass pistons hiss along its joints."}
		case "dieselpunk":
			return paMorph{Prefix: "exhaust-belching", Color: CDarkGray, HP: 20, AV: 2, Elem: "fire", Trait: "Oily black smoke coughs from a rusted dorsal stack."}
		case "atompunk":
			return paMorph{Prefix: "isotope-glowing", Color: CGreen, HP: 15, Speed: 10, Elem: "acid", Trait: "Cherenkov-green veins pulse beneath its translucent hide."}
		case "cyberpunk":
			return paMorph{Prefix: "chrome-spliced", Color: CCyan, HP: 15, AV: 2, DV: 1, Trait: "Subdermal chrome wiring and red targeting optics glint."}
		case "biopunk":
			return paMorph{Prefix: "vat-warped", Color: CMagenta, HP: 25, Flag: "regen", Trait: "Asymmetric muscle-bundles twitch with engineered growth."}
		case "solarpunk":
			return paMorph{Prefix: "sun-crested", Color: CYellow, HP: 15, DV: 2, Trait: "Photosynthetic frills fan across its shoulders like stained glass."}
		case "mythic":
			return paMorph{Prefix: "oracle-marked", Color: CWhite, HP: 20, DV: 2, Trait: "Gold-leaf patterns curve across its brow like a temple frieze."}
		case "wasteland":
			return paMorph{Prefix: "scrap-plated", Color: CDarkOrange, HP: 20, AV: 3, Trait: "Jagged road-sign iron is bolted crudely to its hide."}
		case "twilight":
			return paMorph{Prefix: "gilt-tarnished", Color: CYellow, HP: 15, AV: 2, Trait: "Flaking gold leaf and grave-wax cling to its limbs."}
		}
	}
	if z > 0 {
		return Pick(r, []paMorph{
			{Prefix: "blind-cave", Color: CWhite, DV: 3, Speed: 10, Trait: "Eyeless sockets and long, trembling sensory bristles."},
			{Prefix: "geode-crusted", Color: CMagenta, HP: 20, AV: 3, Trait: "Sharp violet quartz crystals sprout along its spine."},
			{Prefix: "spore-ridden", Color: CDarkMagenta, HP: 15, Elem: "poison", Flag: "poison", Trait: "Pallid cave-fungi fruit from its back, puffing spores."},
		})
	}
	switch biome {
	case 1, 2:
		return paMorph{Prefix: "dune-scarred", Color: CYellow, HP: 10, DV: 2, Speed: 5, Trait: "Sand-scoured scales and lidded eyes built for blinding dust."}
	case 9, 10, 11:
		return paMorph{Prefix: "frost-rimed", Color: CCyan, HP: 20, AV: 1, Elem: "frost", Flag: "frost", Trait: "Hoarfrost clings to its pelt and its breath hangs like fog."}
	case 7, 8, 12:
		return paMorph{Prefix: "mire-bloated", Color: CDarkGreen, HP: 20, Flag: "poison", Elem: "poison", Trait: "Reeking of peat-water and draped in trailing pond-weed."}
	case 13, 16:
		return paMorph{Prefix: "cinder-born", Color: COrange, HP: 20, AV: 1, Elem: "fire", Flag: "fire", Trait: "Embers glow in the cracks of its slag-blackened hide."}
	case 14:
		return paMorph{Prefix: "prism-faceted", Color: CMagenta, HP: 15, AV: 3, Trait: "Light splinters across its crystalline carapace."}
	case 15:
		return paMorph{Prefix: "mycelial", Color: CMagenta, HP: 20, Flag: "regen", Trait: "Laced with glowing hyphae that stitch wounds shut."}
	}
	return Pick(r, paCommonMorphs)
}

func paApplyMorph(a *Actor, m paMorph) {
	if a == nil || a.Proper || a.Boss || a.Role != "" || a.Faction == "outlaws" {
		return
	}
	a.Name = m.Prefix + " " + a.Name
	if m.Color != 0 {
		a.F = m.Color
	}
	if m.HP != 0 {
		a.MaxHP = maxi(4, a.MaxHP*(100+m.HP)/100)
		a.HP = a.MaxHP
	}
	a.AVNat = maxi(0, a.AVNat+m.AV)
	a.DVNat += m.DV
	a.Speed = clampi(a.Speed+m.Speed, 60, 160)
	if m.Flag != "" {
		a.Set(m.Flag)
	}
	if m.Elem != "" {
		for _, bp := range a.Body {
			if bp.Natural != nil && bp.Natural.Elem == "" {
				bp.Natural.Elem = m.Elem
			}
		}
	}
	if m.Trait != "" {
		if a.Desc != "" && !strings.HasSuffix(a.Desc, ".") {
			a.Desc += "."
		}
		a.Desc += " " + m.Trait
	}
	a.XP = maxi(5, a.XP*6/5)
}

// ---- Procedural Character Portraits & NPC Variations ----

var paBuilds = []string{
	"wiry and weather-beaten", "broad-shouldered", "tall and gaunt", "compact and sinewy",
	"stocky and square-jawed", "lithe and quick-eyed", "stooped but watchful", "statuesque",
}

var paFeatures = []string{
	"a faded duelling scar across the cheek",
	"calloused fingers stained with ink and oil",
	"ritual ash-marks traced along the brow",
	"pierced ears hung with small river-pearls",
	"sun-creased amber eyes",
	"a silver-stitched scar at the collarbone",
	"a tattoo of a coiled serpent on the wrist",
	"wind-chapped skin and a wry half-smile",
	"braided locks pinned with carved bone",
	"a copper signet ring worn on a leather cord",
}

var paMannerisms = []string{
	"keeps one hand resting near their belt",
	"tallies figures on a small wax tablet",
	"hums a slow folk air of the region",
	"watches the road with restless eyes",
	"whittles a piece of seasoned driftwood",
	"polishes a tarnished brass token",
	"adjusts a travel-stained mantle against the draught",
	"speaks in measured, unhurried tones",
}

func paDecorateActor(g *Game, a *Actor, r *Rng) {
	if a == nil || strings.Contains(a.Desc, "Appears ") {
		return
	}
	if a.Role == "" && !a.Proper {
		return
	}
	ar := r
	if a.SimPer > 0 {
		ar = NewRng(fmt.Sprintf("%s:pa:per:%d", g.W.Seed, a.SimPer))
	}
	build := Pick(ar, paBuilds)
	feat := Pick(ar, paFeatures)
	man := Pick(ar, paMannerisms)
	line := fmt.Sprintf("Appears %s, with %s; %s.", build, feat, man)
	if a.Desc == "" {
		a.Desc = line
	} else {
		a.Desc += " " + line
	}
}

// ---- Procedural Wandering Characters ----

type paWandererDef struct {
	Role  string
	Title string
	Color byte
	Fact  string
	Desc  string
}

var paWandererDefs = []paWandererDef{
	{Role: "naturalist", Title: "field naturalist", Color: CGreen, Fact: "state:0", Desc: "Loaded with specimen jars, pressed leaves, and a charcoal sketchbook."},
	{Role: "pilgrim", Title: "wayside pilgrim", Color: CWhite, Fact: "state:0", Desc: "Dust-healed sandals, a carved staff, and a string of worn prayer-beads."},
	{Role: "prospector", Title: "independent prospector", Color: COrange, Fact: "state:0", Desc: "Picks, loupe-goggles, and a heavy sack of cracked geodes."},
	{Role: "chronicler", Title: "wandering chronicler", Color: CCyan, Fact: "state:0", Desc: "Carries oilskin scroll-tubes and a lutestring of ballad-sheets."},
	{Role: "forager", Title: "wilderness herbalist", Color: CGreen, Fact: "state:0", Desc: "Smells of crushed mint, woodsmoke, and dried cave-mushrooms."},
	{Role: "beastkeeper", Title: "nomad beast-handler", Color: CYellow, Fact: "state:0", Desc: "Wears thick falconer's gauntlets and whistles low signals."},
}

func paSpawnWanderer(g *Game, zn *Zone, r *Rng) {
	if zn.Underwater || zn.Kind == "dungeon" || zn.Kind == "tower" || zn.Kind == "lair" {
		return
	}
	chance := 0.28
	if zn.Kind == "town" {
		chance = 0.45
	}
	if !r.Chance(chance) {
		return
	}
	x, y, ok := zn.FreeSpot(r, nil)
	if !ok {
		return
	}
	def := Pick(r, paWandererDefs)
	p := g.W.Idx(clampi(zn.WX, 0, g.W.W-1), clampi(zn.WY, 0, g.W.H-1))
	cu := g.W.cultByID[g.W.Culture[p]]
	sp := g.speciesOf(cu)
	plan := "humanoid"
	spID := "human"
	if sp != nil {
		plan = BodyPlanForSpecies(sp.ID)
		spID = sp.ID
	}
	lvl := maxi(2, zn.Level+1)
	a := &Actor{
		ID: g.newID(), Name: g.cultureName(cu, r), Proper: true, Title: def.Title,
		Glyph: '@', F: def.Color, Tpl: "npc", Faction: def.Fact, Level: lvl,
		Attr: [6]int{13, 14, 14, 15, 14, 15}, Speed: 100, AI: "town",
		Role: def.Role, Species: spID, Body: MakeBody(plan),
		MaxHP: 16 + lvl*4, HP: 16 + lvl*4, XP: 15 + lvl*8,
		X: x, Y: y, HomeX: x, HomeY: y,
		Desc: def.Desc,
	}
	if g.W.State[p] > 0 {
		a.Faction = fmt.Sprintf("state:%d", g.W.State[p])
		a.State = g.W.State[p]
	}
	if g.W.Religion[p] > 0 {
		a.Religion = g.W.Religion[p]
	}
	paDecorateActor(g, a, r)
	// Stock wanderers so they can also trade or drop thematic gear
	switch def.Role {
	case "forager", "naturalist":
		a.Gold = r.Int(40, 120)
		g.stockIngredients(a, cu, r)
		a.Inv = append(a.Inv, g.MakeTonic("salve", 2))
	case "prospector":
		a.Gold = r.Int(80, 220)
		a.Inv = append(a.Inv, g.MakeScrap(r, r.Int(4, 10)), g.MakeOddity(r, clampi(lvl/3+1, 1, 5)))
	case "chronicler", "pilgrim":
		a.Gold = r.Int(30, 90)
		a.Inv = append(a.Inv, g.GenBook(r, p), g.MakeWater())
	case "beastkeeper":
		a.Gold = r.Int(50, 140)
		// Companion beast beside the handler
		for _, d := range dirs8 {
			bx, by := x+d[0], y+d[1]
			if zn.Walkable(bx, by) && zn.ActorAt(bx, by) == nil {
				bTpl := Pick(r, []string{"wolf", "stag", "boar", "beetle", "jaguar"})
				pet := g.Spawn(bTpl, maxi(1, lvl-1), r)
				pet.Faction = a.Faction
				pet.AI = "town"
				pet.Name = "tamed " + pet.Name
				pet.Desc = "Collared in braided leather, pacing beside " + a.Name + "."
				pet.X, pet.Y = bx, by
				pet.HomeX, pet.HomeY = bx, by
				zn.Actors = append(zn.Actors, pet)
				break
			}
		}
	}
	zn.Actors = append(zn.Actors, a)
}

// ---- Procedural Environmental Setpieces & Biome Assets ----

func paEnvironment(g *Game, zn *Zone, r *Rng) {
	if zn.Kind == "town" {
		paDecorateTown(g, zn, r)
		return
	}
	if zn.Kind != "surface" && zn.Kind != "underdark" && zn.Kind != "sky" && zn.Kind != "sea" {
		return
	}
	// 1. Scatter biome micro-flora & terrain props
	for i := 0; i < r.Int(5, 11); i++ {
		x, y, ok := zn.FreeSpot(r, nil)
		if !ok {
			continue
		}
		t := zn.At(x, y)
		if t == TRoad || t == TBridge || t == TDown || t == TUp || t == TUpdraft || t == TPortal {
			continue
		}
		switch zn.Biome {
		case 1, 2: // desert / steppe
			if r.Chance(0.35) {
				zn.Set(x, y, TBones)
				zn.Notes[y*MW+x] = "Sun-bleached ribs of a desert beast half-swallowed by drift sand."
			} else if r.Chance(0.25) {
				zn.Set(x, y, TDeadTree)
			}
		case 3, 4, 5, 6: // grasslands & temperate forests
			if r.Chance(0.4) {
				zn.Set(x, y, TMoss)
			} else if r.Chance(0.2) {
				zn.Set(x, y, TBush)
			}
		case 7, 8, 12: // jungle & marsh
			if r.Chance(0.45) {
				zn.Set(x, y, TReeds)
			} else if r.Chance(0.25) {
				zn.Set(x, y, TMushroom)
			}
		case 9, 10, 11: // taiga, tundra, glacier
			if r.Chance(0.35) {
				zn.Set(x, y, TIce)
			}
		case 13, 16: // volcanic / ash
			if r.Chance(0.35) {
				zn.Set(x, y, TBrazier)
				zn.Notes[y*MW+x] = "A smoking fumarole venting sulfurous heat from the crust."
			}
		case 14: // crystal
			if r.Chance(0.4) {
				zn.Set(x, y, TCrystal)
			}
		case 15: // fungal
			if r.Chance(0.45) {
				zn.Set(x, y, TMushroom)
			}
		}
	}

	// 2. Wayside road landmarks (milestones, lanterns, shrines along roads)
	for y := 2; y < MH-2; y++ {
		for x := 2; x < MW-2; x++ {
			if zn.At(x, y) == TRoad && r.Chance(0.018) {
				for _, d := range dirs8 {
					nx, ny := x+d[0], y+d[1]
					if zn.In(nx, ny) && zn.At(nx, ny) != TRoad && zn.Walkable(nx, ny) && zn.ActorAt(nx, ny) == nil {
						switch r.Int(0, 2) {
						case 0:
							zn.Set(nx, ny, TBrazier)
							zn.Notes[ny*MW+nx] = "A wayside road-lantern hung on an iron hook to guide night travellers."
						case 1:
							zn.Set(nx, ny, TInscription)
							p := g.W.Idx(clampi(zn.WX, 0, g.W.W-1), clampi(zn.WY, 0, g.W.H-1))
							zn.Notes[ny*MW+nx] = fmt.Sprintf("A weathered milestone of %s: 'Keep to the road; the wilds remember older kings.'", g.W.StateName(g.W.State[p]))
						default:
							zn.Set(nx, ny, TStatue)
							zn.Notes[ny*MW+nx] = "A mossy wayside herm heaped with small river-pebbles by passing caravans."
						}
						break
					}
				}
			}
		}
	}

	// 3. Procedural Environmental Setpiece (45% chance per wilderness/underdark zone)
	if !r.Chance(0.45) {
		return
	}
	cx, cy, ok := zn.FreeSpot(r, func(x, y int) bool {
		return x > 10 && x < MW-11 && y > 5 && y < MH-6
	})
	if !ok {
		return
	}
	p := g.W.Idx(clampi(zn.WX, 0, g.W.W-1), clampi(zn.WY, 0, g.W.H-1))
	switch r.Int(0, 5) {
	case 0: // Ancient Monolith Circle
		pts := [][2]int{{-3, 0}, {3, 0}, {0, -2}, {0, 2}, {-2, -2}, {2, -2}, {-2, 2}, {2, 2}}
		for _, pt := range pts {
			if zn.In(cx+pt[0], cy+pt[1]) && zn.At(cx+pt[0], cy+pt[1]) != TDown && zn.At(cx+pt[0], cy+pt[1]) != TUp {
				zn.Set(cx+pt[0], cy+pt[1], TPillar)
			}
		}
		zn.Set(cx, cy, TAltar)
		faith := "forgotten stars"
		if rl := g.W.relByID[g.W.Religion[p]]; rl != nil {
			faith = rl.Name
		}
		zn.Notes[cy*MW+cx] = "A ring of lichen-crusted standing stones aligned to the solstice, sacred to " + faith + "."
	case 1: // Megafauna Fossil Ribcage
		for dx := -3; dx <= 3; dx++ {
			if zn.In(cx+dx, cy) {
				zn.Set(cx+dx, cy, TBones)
			}
			if dx%2 == 0 {
				if zn.In(cx+dx, cy-1) {
					zn.Set(cx+dx, cy-1, TBones)
				}
				if zn.In(cx+dx, cy+1) {
					zn.Set(cx+dx, cy+1, TBones)
				}
			}
		}
		zn.Set(cx, cy, TInscription)
		zn.Notes[cy*MW+cx] = "The colossal fossilised ribcage of a primordial leviathan, its vertebrae fused into the stone."
	case 2: // Geothermal / Oasis Spring
		for dy := -1; dy <= 1; dy++ {
			for dx := -2; dx <= 2; dx++ {
				if zn.In(cx+dx, cy+dy) {
					if abs(dx) <= 1 && abs(dy) <= 1 {
						zn.Set(cx+dx, cy+dy, TShallow)
					} else {
						zn.Set(cx+dx, cy+dy, TMoss)
					}
				}
			}
		}
		zn.Set(cx, cy, TWell)
		zn.Notes[cy*MW+cx] = "A mineral-rich spring bubbling up through warm stone, ringed by lush moss."
	case 3: // Ruined Watchtower / Pre-Collapse Wreck
		for dy := -2; dy <= 2; dy++ {
			for dx := -2; dx <= 2; dx++ {
				if !zn.In(cx+dx, cy+dy) {
					continue
				}
				if abs(dx) == 2 || abs(dy) == 2 {
					if r.Chance(0.55) {
						zn.Set(cx+dx, cy+dy, TWall)
					} else {
						zn.Set(cx+dx, cy+dy, TRubble)
					}
				} else {
					zn.Set(cx+dx, cy+dy, TFloor)
				}
			}
		}
		zn.Set(cx, cy, TMachine)
		zn.Notes[cy*MW+cx] = "The gutted shell of an old watch-post; rusted mechanisms still tick softly in the rubble."
		if x, y, ok := zn.FreeSpot(r, func(x, y int) bool { return abs(x-cx) <= 2 && abs(y-cy) <= 2 }); ok {
			it := g.Loot(r, maxi(1, zn.Level/3))
			it.X, it.Y = x, y
			zn.Items = append(zn.Items, it)
		}
	case 4: // Bioluminescent Fairy Ring / Spore Grove
		ring := [][2]int{{-2, -1}, {-2, 1}, {2, -1}, {2, 1}, {-1, -2}, {1, -2}, {-1, 2}, {1, 2}}
		for _, pt := range ring {
			if zn.In(cx+pt[0], cy+pt[1]) {
				zn.Set(cx+pt[0], cy+pt[1], TMushroom)
			}
		}
		for dy := -1; dy <= 1; dy++ {
			for dx := -1; dx <= 1; dx++ {
				if zn.In(cx+dx, cy+dy) {
					zn.Set(cx+dx, cy+dy, TFungus)
				}
			}
		}
		zn.Notes[cy*MW+cx] = "A perfect ring of luminous caps humming with faint subterranean warmth."
	case 5: // Wayside Traveller's Shrine & Camp
		zn.Set(cx, cy, TBrazier)
		if zn.In(cx-2, cy) {
			zn.Set(cx-2, cy, TAltar)
			zn.Notes[cy*MW+cx-2] = "A small wayside shrine where travellers leave coins, dried herbs, and prayers for safe passage."
		}
		if zn.In(cx+2, cy) {
			zn.Set(cx+2, cy, TBed)
		}
		if zn.In(cx, cy-1) {
			zn.Set(cx, cy-1, TCrate)
		}
	}
}

func paDecorateTown(g *Game, zn *Zone, r *Rng) {
	// Add town plaza fountain/well, street lanterns, and cultural banners in free town spots
	for i := 0; i < 3; i++ {
		x, y, ok := zn.FreeSpot(r, func(x, y int) bool {
			t := zn.At(x, y)
			return (t == TDirt || t == TCobble || t == TRoad) && x > 8 && x < MW-8 && y > 3 && y < MH-3
		})
		if !ok {
			continue
		}
		switch i {
		case 0:
			zn.Set(x, y, TWell)
			zn.Notes[y*MW+x] = "A carved stone town fountain where locals gather to draw water and trade news."
		case 1:
			zn.Set(x, y, TBrazier)
			zn.Notes[y*MW+x] = "A wrought-iron street brazier casting warm light across the cobbles."
		case 2:
			zn.Set(x, y, TStatue)
			zn.Notes[y*MW+x] = fmt.Sprintf("A civic monument of %s, hung with dyed cloth ribbons.", zn.Name)
		}
	}
}

func paZoneHook(g *Game, zn *Zone, r *Rng) {
	rr := NewRng(g.W.Seed + ":pa:" + zn.Key)
	// 1. Decorate existing NPCs & apply procedural creature morphs
	for _, a := range zn.Actors {
		if a.Role != "" || a.Proper {
			paDecorateActor(g, a, rr)
		} else if rr.Chance(0.38) {
			paApplyMorph(a, paBiomeMorph(zn.Biome, zn.Z, g.Genre, rr))
		}
	}
	// 2. Spawn 1-2 additional biome/depth-specific creatures from paTemplates in wild/delve zones
	if zn.Kind != "town" && rr.Chance(0.65) {
		where := byte('s')
		if zn.Sky {
			where = 'k'
		} else if zn.Underwater {
			where = 'e'
		} else if zn.Kind == "ruin" || zn.Kind == "tower" {
			where = 'r'
		} else if zn.Z > 0 || zn.Kind == "dungeon" || zn.Kind == "lair" {
			where = 'd'
		}
		var fits []string
		for i := range paTemplates {
			t := &paTemplates[i]
			if t.Fits(zn.Biome, where) && !t.Has("boss") && t.Lvl <= zn.Level+4 {
				fits = append(fits, t.ID)
			}
		}
		if len(fits) > 0 {
			id := Pick(rr, fits)
			if x, y, ok := zn.FreeSpot(rr, func(x, y int) bool { return abs(x-40) > 7 || abs(y-12) > 4 }); ok {
				a := g.Spawn(id, zn.Level, rr)
				a.X, a.Y = x, y
				if rr.Chance(0.45) {
					paApplyMorph(a, paBiomeMorph(zn.Biome, zn.Z, g.Genre, rr))
				}
				zn.Actors = append(zn.Actors, a)
			}
		}
	}
	// 3. Spawn procedural wandering characters
	paSpawnWanderer(g, zn, rr)
	// 4. Generate procedural environmental setpieces & props
	paEnvironment(g, zn, rr)
}

func paTalkHook(g *Game, a *Actor) []Option {
	if a == nil {
		return nil
	}
	var opts []Option
	switch a.Role {
	case "naturalist":
		opts = append(opts, Option{
			Text: "Ask about the local flora and fauna.",
			Do: func() {
				p := g.W.Idx(clampi(g.WX, 0, g.W.W-1), clampi(g.WY, 0, g.W.H-1))
				bn := BiomeName(g.W.Biome[p])
				g.Msg(fmt.Sprintf("{{G|%s}} taps their field journal: \"In these %s lands, watch the colour of a beast's hide. Frost-rimed, spore-crusted, and elder morphs hit harder than common stock.\"", a.Name, bn))
			},
		})
	case "chronicler":
		opts = append(opts, Option{
			Text: "Ask to hear a ballad of the realm's history.",
			Do: func() {
				if len(g.W.Events) > 0 {
					ev := g.W.Events[(a.ID*37)%len(g.W.Events)]
					g.Msg(fmt.Sprintf("{{C|%s}} strums a chord: \"In the year %d: %s\"", a.Name, ev.Y, ev.Text))
				} else {
					g.Msg(fmt.Sprintf("{{C|%s}} hums an old road-tune.", a.Name))
				}
			},
		})
	case "prospector", "forager":
		opts = append(opts, Option{
			Text: "Trade with the traveller.",
			Do: func() {
				g.openTrade(a)
			},
		})
	case "pilgrim":
		opts = append(opts, Option{
			Text: "Share a road-blessing with the pilgrim.",
			Do: func() {
				g.Msg(fmt.Sprintf("{{W|%s}} touches two fingers to their brow: \"May the road rise clear before your boots, wanderer.\"", a.Name))
				if g.P.HP < g.P.MaxHP {
					g.P.HP = mini(g.P.MaxHP, g.P.HP+4)
				}
			},
		})
	}
	return opts
}
