package engine

import "strings"

// CTpl is a creature archetype; names and descriptions are re-skinned per genre.
type CTpl struct {
	ID      string
	Glyph   rune
	F       byte
	Lvl     int
	Attr    [6]int
	HP      int
	AV, DV  int
	Speed   int
	Faction string
	Biomes  []int  // nil = any
	Where   string // s surface, d deep, k sky, e sea floor, r ruins/tombs
	Body    string
	Flags   string // ranged flee pack aquatic fly burrow regen undead armed boss fire frost acid poison bleed
	Names   map[string]string
	Desc    string
	Bite    [3]int // dice n, sides, pv for the natural weapon override
	Elem    string
}

func nm(def string, kv ...string) map[string]string {
	m := map[string]string{"": def}
	for i := 0; i+1 < len(kv); i += 2 {
		for _, g := range strings.Split(kv[i], ",") {
			m[g] = kv[i+1]
		}
	}
	return m
}

var forests = []int{5, 6, 7, 8, 9}
var opens = []int{3, 4, 1, 2}

var Templates = []CTpl{
	{ID: "rat", Glyph: 'r', F: CBrown, Lvl: 1, Attr: [6]int{10, 16, 10, 6, 8, 6}, HP: 6, Speed: 110, Faction: "vermin", Where: "sdr", Body: "quadruped", Bite: [3]int{1, 3, 0},
		Names: nm("giant rat", "steampunk", "clockwork rat", "dieselpunk", "trench rat", "atompunk", "rad-rat", "cyberpunk", "scrap drone", "biopunk", "splice-rat", "mythic", "plague rat", "wasteland", "tunnel rat", "twilight", "plague rat"), Desc: "It skitters, all teeth and hunger."},
	{ID: "beetle", Glyph: 'b', F: CDarkGreen, Lvl: 2, Attr: [6]int{14, 12, 14, 4, 8, 4}, HP: 10, AV: 4, Speed: 90, Faction: "vermin", Where: "sd", Body: "insect", Bite: [3]int{1, 4, 1},
		Names: nm("dung beetle", "steampunk,dieselpunk", "mechanical scarab", "cyberpunk", "crawler drone", "biopunk", "tank-beetle", "atompunk", "glowing beetle", "wasteland", "rust beetle", "twilight", "tomb beetle"), Desc: "A shell the size of a cartwheel."},
	{ID: "snake", Glyph: 's', F: CGreen, Lvl: 2, Attr: [6]int{10, 18, 10, 4, 10, 6}, HP: 8, Speed: 100, Faction: "beasts", Biomes: []int{1, 3, 4, 5, 7, 12}, Where: "s", Body: "serpent", Flags: "poison",
		Names: nm("viper", "cyberpunk", "snake drone", "biopunk", "spit-asp", "mythic", "lamia-hatchling", "atompunk", "two-headed viper", "wasteland", "pipe viper", "twilight", "bile adder"), Desc: "Its scales shift with the light."},
	{ID: "boar", Glyph: 'q', F: CBrown, Lvl: 2, Attr: [6]int{16, 12, 16, 4, 10, 6}, HP: 14, AV: 2, Speed: 100, Faction: "beasts", Biomes: forests, Where: "s", Body: "quadruped", Bite: [3]int{1, 6, 2},
		Names: nm("tusked boar", "mythic", "erymanthian boar", "atompunk", "rad-hog", "biopunk", "tusk-hog", "cyberpunk", "feral pig", "wasteland", "tusk-hog", "twilight", "briar boar"), Desc: "Bristle and tusk and spite."},
	{ID: "wolf", Glyph: 'C', F: CGray, Lvl: 3, Attr: [6]int{15, 17, 14, 6, 12, 8}, HP: 14, Speed: 120, Faction: "beasts", Biomes: []int{4, 6, 9, 10, 3}, Where: "s", Body: "quadruped", Flags: "pack", Bite: [3]int{1, 6, 1},
		Names: nm("dire wolf", "steampunk", "brass hound", "dieselpunk", "war dog", "atompunk", "two-headed dog", "cyberpunk", "cyber-hound", "biopunk", "gene-hound", "mythic", "lykaon wolf", "solarpunk", "feral wolf", "wasteland", "mange hound", "twilight", "pale hound"), Desc: "Grey and lean, it moves with its kin."},
	{ID: "bear", Glyph: 'B', F: CBrown, Lvl: 5, Attr: [6]int{22, 12, 20, 6, 12, 8}, HP: 34, AV: 3, Speed: 95, Faction: "beasts", Biomes: []int{6, 8, 9, 10}, Where: "s", Body: "quadruped", Bite: [3]int{2, 6, 2},
		Names: nm("cave bear", "steampunk", "steam golem", "dieselpunk", "armored bear", "atompunk", "mutant bear", "cyberpunk", "security mech", "biopunk", "chimera", "mythic", "calydonian bear", "wasteland", "scrap bear", "twilight", "bile bear"), Desc: "A mountain of fur and fury."},
	{ID: "jaguar", Glyph: 'f', F: CYellow, Lvl: 4, Attr: [6]int{16, 20, 14, 6, 12, 10}, HP: 18, Speed: 130, Faction: "beasts", Biomes: []int{5, 7}, Where: "s", Body: "quadruped", Bite: [3]int{1, 8, 2}, Flags: "bleed",
		Names: nm("jaguar", "cyberpunk", "stealth-cat drone", "biopunk", "shadow-cat splice", "mythic", "nemean cub", "wasteland", "ruin prowler", "twilight", "shade stalker"), Desc: "Rosettes in the shade, then teeth."},
	{ID: "scorpion", Glyph: 'S', F: COrange, Lvl: 3, Attr: [6]int{14, 14, 14, 4, 10, 4}, HP: 14, AV: 5, Speed: 100, Faction: "beasts", Biomes: []int{1, 2, 13}, Where: "sd", Body: "insect", Flags: "poison", Bite: [3]int{1, 6, 1},
		Names: nm("giant scorpion", "atompunk", "radscorpion", "cyberpunk", "stinger bot", "biopunk", "sting-crab", "wasteland", "glow scorpion", "twilight", "crypt scorpion"), Desc: "Its tail hangs like a question."},
	{ID: "sandworm", Glyph: 'W', F: CYellow, Lvl: 9, Attr: [6]int{26, 8, 26, 2, 14, 6}, HP: 80, AV: 6, Speed: 80, Faction: "beasts", Biomes: []int{1, 2}, Where: "s", Body: "serpent", Flags: "burrow", Bite: [3]int{3, 6, 4},
		Names: nm("sand wyrm", "cyberpunk", "tunnel-borer", "biopunk", "dune maw", "mythic", "sand drakon", "wasteland", "dune borer", "twilight", "sand maw"), Desc: "The dunes themselves seem to breathe."},
	{ID: "yeti", Glyph: 'Y', F: CWhite, Lvl: 6, Attr: [6]int{22, 12, 20, 8, 14, 8}, HP: 40, AV: 2, Speed: 100, Faction: "beasts", Biomes: []int{10, 11, 9}, Where: "s", Body: "humanoid", Flags: "frost", Bite: [3]int{2, 4, 2},
		Names: nm("yeti", "cyberpunk", "arctic mech", "biopunk", "frost-ape splice", "mythic", "hyperborean giant", "wasteland", "snow ghoul", "twilight", "frost beast"), Desc: "A shaggy giant with frost in its beard."},
	{ID: "wisp", Glyph: 'w', F: CCyan, Lvl: 4, Attr: [6]int{6, 20, 10, 12, 16, 10}, HP: 12, DV: 4, Speed: 120, Faction: "spirits", Biomes: []int{11, 10, 14, 12}, Where: "sdk", Body: "ooze", Flags: "fly frost", Bite: [3]int{1, 6, 3}, Elem: "frost",
		Names: nm("frost wisp", "cyberpunk", "rogue hologram", "steampunk,dieselpunk", "ghost-light", "biopunk", "cold spore-cloud", "mythic", "boreas shade", "wasteland", "static wisp", "twilight", "pale wisp"), Desc: "Cold light that hates warmth."},
	{ID: "leech", Glyph: 'l', F: CDarkRed, Lvl: 1, Attr: [6]int{8, 10, 10, 2, 6, 2}, HP: 6, Speed: 80, Faction: "vermin", Biomes: []int{12, 7, 8}, Where: "se", Body: "serpent", Flags: "aquatic bleed", Bite: [3]int{1, 3, 1},
		Names: nm("bog leech", "cyberpunk", "nanoleech", "biopunk", "blood-worm", "wasteland", "sludge leech", "twilight", "blood leech"), Desc: "A sucking mouth on a muscle."},
	{ID: "croc", Glyph: 'K', F: CDarkGreen, Lvl: 5, Attr: [6]int{20, 12, 18, 4, 12, 6}, HP: 30, AV: 5, Speed: 90, Faction: "beasts", Biomes: []int{12, 7}, Where: "s", Body: "quadruped", Flags: "aquatic", Bite: [3]int{2, 6, 3},
		Names: nm("crocodile", "mythic", "sobek's child", "cyberpunk", "sewer gator", "biopunk", "gator splice", "atompunk", "mutant gator", "wasteland", "sewer gator", "twilight", "mire drake"), Desc: "Ridged, patient, ancient."},
	{ID: "spider", Glyph: 'x', F: CDarkMagenta, Lvl: 3, Attr: [6]int{12, 18, 12, 6, 10, 6}, HP: 12, Speed: 110, Faction: "vermin", Biomes: []int{5, 6, 7, 8, 15}, Where: "sdr", Body: "insect", Flags: "poison", Bite: [3]int{1, 4, 1},
		Names: nm("giant spider", "steampunk", "clockwork spider", "cyberpunk", "spider-drone", "biopunk", "silk-weaver", "mythic", "arachne's spawn", "wasteland", "web crawler", "twilight", "lace spider"), Desc: "Eight eyes, all on you."},
	{ID: "sporefolk", Glyph: 'm', F: CMagenta, Lvl: 3, Attr: [6]int{12, 10, 14, 8, 12, 10}, HP: 16, Speed: 90, Faction: "fungi", Biomes: []int{15}, Where: "sd", Body: "humanoid", Flags: "poison", Bite: [3]int{1, 4, 1},
		Names: nm("myconid", "cyberpunk", "fungal host", "biopunk", "spore-folk", "atompunk", "glowing ghoul-fungus", "wasteland", "fungal host", "twilight", "mold thrall"), Desc: "A walking mushroom, puffing spores."},
	{ID: "shambler", Glyph: 'M', F: CDarkMagenta, Lvl: 6, Attr: [6]int{20, 8, 22, 6, 14, 8}, HP: 44, AV: 3, Speed: 80, Faction: "fungi", Biomes: []int{15, 12}, Where: "sd", Body: "golem", Flags: "regen", Bite: [3]int{2, 6, 2},
		Names: nm("shambling mound", "biopunk", "mycelial colossus", "cyberpunk", "garbage golem", "wasteland", "garbage heap", "twilight", "grave mound"), Desc: "Rot heaped into the shape of a person."},
	{ID: "crystalgolem", Glyph: 'G', F: CMagenta, Lvl: 7, Attr: [6]int{22, 8, 22, 4, 16, 4}, HP: 50, AV: 10, Speed: 80, Faction: "crystals", Biomes: []int{14}, Where: "sd", Body: "golem",
		Names: nm("crystal golem", "cyberpunk", "silicate construct", "steampunk", "quartz automaton", "biopunk", "geode-beast", "wasteland", "glass construct", "twilight", "glass sentinel"), Desc: "Light refracts through its facets in sullen rainbows."},
	{ID: "shardmoth", Glyph: 'v', F: CCyan, Lvl: 2, Attr: [6]int{6, 20, 8, 6, 10, 6}, HP: 6, DV: 3, Speed: 140, Faction: "crystals", Biomes: []int{14}, Where: "sk", Body: "avian", Flags: "fly bleed",
		Names: nm("shard moth", "cyberpunk", "razor drone", "wasteland", "glint moth", "twilight", "pale moth"), Desc: "Wings like broken glass."},
	{ID: "salamander", Glyph: 'L', F: CRed, Lvl: 5, Attr: [6]int{16, 16, 16, 6, 12, 8}, HP: 26, AV: 4, Speed: 100, Faction: "beasts", Biomes: []int{16, 13}, Where: "sd", Body: "quadruped", Flags: "fire", Bite: [3]int{1, 6, 2}, Elem: "fire",
		Names: nm("fire salamander", "cyberpunk", "incinerator bot", "steampunk", "boiler-lizard", "biopunk", "pyro-newt", "wasteland", "flare lizard", "twilight", "ember newt"), Desc: "Its skin shimmers with heat."},
	{ID: "ashwraith", Glyph: 'W', F: CDarkGray, Lvl: 5, Attr: [6]int{12, 16, 12, 10, 16, 12}, HP: 22, DV: 3, Speed: 110, Faction: "undead", Biomes: []int{13}, Where: "sr", Body: "humanoid", Flags: "undead fly", Bite: [3]int{1, 6, 2},
		Names: nm("ash wraith", "cyberpunk", "smoke ghost", "dieselpunk", "gas ghost", "atompunk", "glowing one", "wasteland", "glowing shade", "twilight", "ash revenant"), Desc: "Cinders in the shape of grief."},
	{ID: "magma", Glyph: 'E', F: COrange, Lvl: 9, Attr: [6]int{24, 10, 24, 6, 18, 8}, HP: 70, AV: 8, Speed: 90, Faction: "elementals", Biomes: []int{16}, Where: "sd", Body: "golem", Flags: "fire", Elem: "fire",
		Names: nm("magma elemental", "cyberpunk", "reactor golem", "steampunk", "furnace titan", "biopunk", "lava-hive", "wasteland", "slag elemental", "twilight", "cinder colossus"), Desc: "Molten rock walking in a body of slag."},
	{ID: "skeleton", Glyph: 'z', F: CWhite, Lvl: 3, Attr: [6]int{14, 14, 14, 4, 10, 4}, HP: 14, AV: 2, Speed: 100, Faction: "undead", Where: "dr", Body: "humanoid", Flags: "undead armed",
		Names: nm("skeleton", "cyberpunk", "chrome husk", "steampunk", "clockwork corpse", "dieselpunk", "gas-mask revenant", "atompunk", "feral ghoul", "biopunk", "fungal host", "solarpunk", "blighted husk", "mythic", "shade-warrior", "wasteland", "bleached husk", "twilight", "hollow bones"), Desc: "Bones that refuse to lie down."},
	{ID: "ghoul", Glyph: 'Z', F: CGreen, Lvl: 5, Attr: [6]int{18, 14, 18, 6, 12, 4}, HP: 28, AV: 2, Speed: 105, Faction: "undead", Where: "dr", Body: "humanoid", Flags: "undead", Bite: [3]int{1, 6, 2},
		Names: nm("ghoul", "cyberpunk", "cyberpsycho", "biopunk", "spore-thrall", "atompunk", "glowing ghoul", "mythic", "lamia", "wasteland", "glow ghoul", "twilight", "plague thrall"), Desc: "It reeks of the grave."},
	{ID: "wight", Glyph: 'V', F: CMagenta, Lvl: 9, Attr: [6]int{20, 16, 20, 14, 20, 16}, HP: 60, AV: 5, Speed: 110, Faction: "undead", Where: "r", Body: "humanoid", Flags: "undead armed leech",
		Names: nm("barrow wight", "cyberpunk", "rogue AI chassis", "steampunk", "brass revenant", "biopunk", "hive-lord", "mythic", "lich-king's shade", "wasteland", "vault stalker", "twilight", "fallen lord"), Desc: "A crowned dead thing that remembers being king."},
	{ID: "ooze", Glyph: 'o', F: CGreen, Lvl: 2, Attr: [6]int{12, 6, 16, 2, 8, 2}, HP: 16, Speed: 70, Faction: "oozes", Where: "dr", Body: "ooze", Flags: "acid", Elem: "acid",
		Names: nm("green slime", "cyberpunk", "grey goo", "biopunk", "protoplasm", "atompunk", "toxic sludge", "steampunk", "tar ooze", "wasteland", "toxic sludge", "twilight", "black ichor"), Desc: "It quivers. It dissolves."},
	{ID: "crawler", Glyph: 'c', F: CGray, Lvl: 4, Attr: [6]int{16, 14, 16, 4, 10, 4}, HP: 20, AV: 4, Speed: 100, Faction: "vermin", Where: "d", Body: "insect", Flags: "pack", Bite: [3]int{1, 6, 2},
		Names: nm("cave crawler", "cyberpunk", "maintenance crawler", "biopunk", "tunnel-grub", "steampunk", "sewer crawler", "wasteland", "duct crawler", "twilight", "crypt crawler"), Desc: "Antennae twitch in the dark."},
	{ID: "troglodyte", Glyph: 't', F: CDarkCyan, Lvl: 4, Attr: [6]int{16, 14, 16, 8, 10, 8}, HP: 22, AV: 2, Speed: 100, Faction: "outlaws", Where: "d", Body: "humanoid", Flags: "armed pack",
		Names: nm("troglodyte", "cyberpunk", "tunnel rat", "biopunk", "cave-splice", "steampunk", "mole-man", "wasteland", "tunnel dweller", "twilight", "pale delver"), Desc: "Pale, eyeless, armed with sharpened stone."},
	{ID: "horror", Glyph: 'H', F: CMagenta, Lvl: 11, Attr: [6]int{26, 14, 26, 18, 22, 18}, HP: 90, AV: 6, Speed: 105, Faction: "eldritch", Where: "d", Body: "tentacled", Flags: "regen",
		Names: nm("deep horror", "cyberpunk", "black-ICE manifestation", "biopunk", "abyssal brood-mother", "mythic", "chthonic spawn", "wasteland", "deep mutant", "twilight", "abyssal watcher"), Desc: "Its outline hurts to look at."},
	{ID: "bandit", Glyph: '@', F: CDarkRed, Lvl: 2, Attr: [6]int{15, 15, 14, 10, 10, 10}, HP: 14, Speed: 100, Faction: "outlaws", Where: "sd", Body: "humanoid", Flags: "armed pack",
		Names: nm("brigand", "mythic", "raider", "steampunk", "highwayman", "dieselpunk", "deserter", "atompunk", "raider", "cyberpunk", "ganger", "biopunk", "feral splicer", "solarpunk", "saboteur", "wasteland", "raider", "twilight", "cutthroat"), Desc: "Hungry eyes and a ready blade."},
	{ID: "archer", Glyph: '@', F: CDarkOrange, Lvl: 3, Attr: [6]int{12, 18, 12, 10, 10, 10}, HP: 12, Speed: 100, Faction: "outlaws", Where: "s", Body: "humanoid", Flags: "ranged armed pack",
		Names: nm("brigand archer", "steampunk", "rifleman", "dieselpunk", "sniper", "atompunk", "raider gunner", "cyberpunk", "ganger shooter", "biopunk", "spine-spitter", "mythic", "slinger", "wasteland", "raider gunner", "twilight", "crossbow thug"), Desc: "Keeps their distance."},
	{ID: "chief", Glyph: '@', F: CRed, Lvl: 7, Attr: [6]int{20, 16, 20, 12, 14, 16}, HP: 44, AV: 3, Speed: 100, Faction: "outlaws", Where: "sd", Body: "humanoid", Flags: "armed boss",
		Names: nm("brigand chief", "steampunk", "gang boss", "dieselpunk", "warlord", "atompunk", "raider boss", "cyberpunk", "gang boss", "biopunk", "alpha splicer", "mythic", "pirate king", "wasteland", "raider warboss", "twilight", "gang lord"), Desc: "Scarred, decorated with trophies."},
	{ID: "harpy", Glyph: 'h', F: CBrown, Lvl: 4, Attr: [6]int{12, 18, 12, 8, 10, 6}, HP: 16, DV: 2, Speed: 120, Faction: "beasts", Where: "k", Body: "avian", Flags: "fly",
		Names: nm("harpy", "cyberpunk", "aerial drone", "steampunk", "sky-pirate", "biopunk", "carrion-wing", "wasteland", "carrion flier", "twilight", "carrion crow"), Desc: "Shrieking, wings like knives."},
	{ID: "roc", Glyph: 'R', F: CYellow, Lvl: 8, Attr: [6]int{24, 14, 22, 6, 14, 10}, HP: 60, AV: 3, Speed: 110, Faction: "beasts", Where: "k", Body: "avian", Flags: "fly", Bite: [3]int{2, 6, 3},
		Names: nm("roc", "cyberpunk", "gunship drone", "steampunk", "war-ornithopter", "biopunk", "sky-whale calf", "mythic", "stymphalian bird", "wasteland", "giant buzzard", "twilight", "storm raptor"), Desc: "Its shadow covers whole isles."},
	{ID: "skyray", Glyph: 'y', F: CCyan, Lvl: 3, Attr: [6]int{12, 16, 12, 6, 10, 6}, HP: 14, Speed: 110, Faction: "beasts", Where: "k", Body: "fish", Flags: "fly shock", Elem: "shock",
		Names: nm("sky-ray", "cyberpunk", "static drone", "biopunk", "gas-ray", "wasteland", "smog ray", "twilight", "gloom ray"), Desc: "It glides on the wind, crackling."},
	{ID: "crab", Glyph: 'k', F: CRed, Lvl: 2, Attr: [6]int{14, 10, 14, 2, 8, 2}, HP: 12, AV: 5, Speed: 90, Faction: "beasts", Where: "e", Body: "insect", Flags: "aquatic", Bite: [3]int{1, 6, 1},
		Names: nm("giant crab", "cyberpunk", "seabed drone", "biopunk", "reef-crab", "wasteland", "rust crab", "twilight", "brine crab"), Desc: "Claws click hungrily."},
	{ID: "shark", Glyph: 'F', F: CGray, Lvl: 5, Attr: [6]int{20, 16, 18, 2, 10, 4}, HP: 30, AV: 2, Speed: 130, Faction: "beasts", Where: "e", Body: "fish", Flags: "aquatic bleed", Bite: [3]int{2, 6, 2},
		Names: nm("reef shark", "cyberpunk", "hunter-killer sub", "biopunk", "gene-shark", "mythic", "ketos", "wasteland", "mutant shark", "twilight", "drowned shark"), Desc: "Rows of teeth, black eyes."},
	{ID: "kraken", Glyph: 'X', F: CMagenta, Lvl: 10, Attr: [6]int{26, 12, 24, 10, 18, 10}, HP: 90, AV: 5, Speed: 100, Faction: "eldritch", Where: "e", Body: "tentacled", Flags: "aquatic",
		Names: nm("kraken spawn", "cyberpunk", "abyssal leviathan-drone", "biopunk", "deep brood", "mythic", "scylla's child", "wasteland", "sludge horror", "twilight", "tide brood"), Desc: "Tentacles uncoil from the abyss."},
	{ID: "eel", Glyph: 'e', F: CYellow, Lvl: 3, Attr: [6]int{12, 18, 12, 2, 8, 2}, HP: 12, Speed: 120, Faction: "beasts", Where: "e", Body: "serpent", Flags: "aquatic shock", Elem: "shock",
		Names: nm("shock eel", "cyberpunk", "power-line eel", "wasteland", "wire eel", "twilight", "storm eel"), Desc: "It hums with stored lightning."},
	{ID: "golem", Glyph: 'g', F: CGray, Lvl: 6, Attr: [6]int{22, 8, 22, 4, 16, 4}, HP: 46, AV: 8, Speed: 80, Faction: "constructs", Where: "r", Body: "golem",
		Names: nm("stone guardian", "steampunk", "clockwork sentinel", "dieselpunk", "iron sentry", "atompunk", "robo-guard", "cyberpunk", "sentry turret-mech", "biopunk", "gene-warden", "solarpunk", "old security bot", "mythic", "bronze talos", "wasteland", "rusted sentry", "twilight", "gilded guardian"), Desc: "It was built to guard this place, and it still does."},
	{ID: "turret", Glyph: 'T', F: CRed, Lvl: 5, Attr: [6]int{10, 16, 18, 4, 14, 4}, HP: 24, AV: 6, Speed: 100, Faction: "constructs", Where: "r", Body: "golem", Flags: "ranged still",
		Names: nm("arrow trap-idol", "steampunk", "steam turret", "dieselpunk", "machine-gun nest", "atompunk", "laser turret", "cyberpunk", "smart turret", "biopunk", "spine-pod", "mythic", "dart idol", "wasteland", "scrap turret", "twilight", "cursed idol"), Desc: "Tracking you."},
}

var tplByID = func() map[string]*CTpl {
	m := map[string]*CTpl{}
	for i := range Templates {
		m[Templates[i].ID] = &Templates[i]
	}
	return m
}()

func (t *CTpl) Name(genre string) string {
	if v, ok := t.Names[genre]; ok {
		return v
	}
	return t.Names[""]
}

func (t *CTpl) Fits(biome int, where byte) bool {
	if !strings.ContainsRune(t.Where, rune(where)) {
		return false
	}
	if where != 's' || t.Biomes == nil {
		return true
	}
	for _, b := range t.Biomes {
		if b == biome {
			return true
		}
	}
	return false
}

func (g *Game) newID() int {
	g.NextID++
	return g.NextID
}

// Spawn builds an actor from a template scaled to a level.
func (g *Game) Spawn(id string, lvl int, r *Rng) *Actor {
	t := tplByID[id]
	if t == nil {
		t = tplByID["rat"]
	}
	if lvl < t.Lvl {
		lvl = t.Lvl
	}
	up := lvl - t.Lvl
	a := &Actor{ID: g.newID(), Name: t.Name(g.Genre), Glyph: t.Glyph, F: t.F, Tpl: t.ID, Faction: t.Faction, Level: lvl, Attr: t.Attr,
		AVNat: t.AV + up/3, DVNat: t.DV, Speed: t.Speed, AI: "hostile", Desc: t.Desc, Body: MakeBody(t.Body)}
	for i := range a.Attr {
		a.Attr[i] += up / 2
	}
	a.MaxHP = t.HP + up*4 + r.Int(0, t.HP/4)
	a.HP = a.MaxHP
	a.XP = 10 + lvl*lvl*5
	if t.Bite[0] > 0 {
		for _, p := range a.Body {
			if p.Natural != nil {
				p.Natural.DN, p.Natural.DS, p.Natural.PV = t.Bite[0], t.Bite[1]+up/3, t.Bite[2]+up/4
				if t.Elem != "" {
					p.Natural.Elem = t.Elem
				}
			}
		}
	}
	for _, f := range strings.Fields(t.Flags) {
		a.Set(f)
	}
	if t.Elem != "" {
		for _, p := range a.Body {
			if p.Natural != nil && p.Natural.Elem == "" {
				p.Natural.Elem = t.Elem
			}
		}
	}
	if a.Has("poison") {
		for _, p := range a.Body {
			if p.Natural != nil && p.Natural.Elem == "" {
				p.Natural.Elem = "poison"
			}
		}
	}
	if a.Has("armed") {
		tier := clampi(lvl/3, 0, 5)
		skills := []string{"longblade", "shortblade", "axe", "cudgel"}
		w := g.MakeWeapon(Pick(r, skills), tier, r)
		if p := a.PartFor("hand"); p != nil {
			p.Item = w
		}
		if r.Chance(0.6) {
			if p := a.PartFor("body"); p != nil {
				p.Item = g.MakeArmor("body", tier, r)
			}
		}
	}
	if a.Has("ranged") {
		rw := g.MakeRanged(clampi(lvl/3, 0, 5), r)
		for _, p := range a.Body {
			if p.Slot == "hand" && p.Item == nil {
				p.Item = rw
				break
			}
		}
	}
	if a.Has("fly") {
		a.Set("flying")
	}
	if a.Has("boss") {
		a.MaxHP = a.MaxHP * 3 / 2
		a.HP = a.MaxHP
		a.Gold = r.Int(20, 60) * lvl
	}
	if a.Has("still") {
		a.Speed = 100
	}
	if a.Faction == "outlaws" {
		a.Gold += r.Int(0, 6*lvl)
	}
	return a
}

// pickCreature chooses a template for a biome / place kind near a difficulty level.
func pickCreature(r *Rng, biome int, where byte, lvl int) string {
	var ids []string
	var ws []float64
	for i := range Templates {
		t := &Templates[i]
		if !t.Fits(biome, where) || t.Has("boss") || t.Lvl > lvl+3 {
			continue
		}
		d := float64(abs(t.Lvl - lvl))
		w := 1.0 / (1 + d)
		if t.Biomes != nil {
			w *= 2.5
		}
		ids = append(ids, t.ID)
		ws = append(ws, w)
	}
	if len(ids) == 0 {
		return "rat"
	}
	tot := 0.0
	for _, w := range ws {
		tot += w
	}
	x := r.Float() * tot
	for i, w := range ws {
		if x -= w; x <= 0 {
			return ids[i]
		}
	}
	return ids[len(ids)-1]
}

func (t *CTpl) Has(f string) bool { return strings.Contains(" "+t.Flags+" ", " "+f+" ") }
