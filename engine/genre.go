package engine

// Skin re-dresses the same mechanics per Realmforge genre: a cyberpunk "mono-katana" and a
// fantasy "mithril longsword" are the same long-blade tier 3 item.
type Skin struct {
	ID          string
	PowersName  string // what mutations are called
	PowerVerb   string
	Ritual      string // the water-ritual equivalent
	RitualDesc  string
	Currency    string
	Book        string
	Light       [3]string // weak, medium, strong
	Mats        [6]string
	Nouns       map[string][]string // skill → nouns
	Ranged      []string
	Ammo        string
	Armor       map[string][]string // slot → nouns per tier band
	Heal        string
	Salve       string
	Haste       string
	Strength    string
	Antidote    string
	GrenFire    string
	GrenFrost   string
	GrenFlash   string
	Food        []string
	Water       string
	Scrap       string
	Callings    map[string]string
	Creatures   map[string]string
	PowerNames  map[string]string
	Bandits     string
	Undead      string
	WallColor   byte
	AccentColor byte
	Updraft     string
	Dive        string
	Ferry       string
	Kinship     string // tile inscription verb
}

var archetypes = []string{"warrior", "scout", "scholar", "tinker", "mystic", "merchant", "rogue"}

func baseSkin() Skin {
	return Skin{
		PowersName: "Gifts", PowerVerb: "manifest", Ritual: "Rite of Salt and Bread", RitualDesc: "You break bread and share salt",
		Currency: "gold", Book: "tome", Light: [3]string{"torch", "lantern", "glowstone"},
		Mats:   [6]string{"bronze", "iron", "steel", "mithril", "adamant", "starmetal"},
		Nouns:  map[string][]string{"longblade": {"longsword", "broadsword"}, "shortblade": {"dagger", "dirk"}, "axe": {"battleaxe", "hatchet"}, "cudgel": {"mace", "warhammer"}},
		Ranged: []string{"shortbow", "longbow", "crossbow"}, Ammo: "arrows",
		Armor: map[string][]string{"body": {"leather jerkin", "chain hauberk", "plate cuirass"}, "head": {"cap", "helm", "great helm"}, "feet": {"boots", "greaves"}, "hands": {"gloves", "gauntlets"}, "back": {"cloak", "mantle"}, "face": {"veil", "visor"}},
		Heal:  "healing draught", Salve: "wound salve", Haste: "quicksilver tonic", Strength: "giant's-blood tonic", Antidote: "antivenom",
		GrenFire: "firepot", GrenFrost: "rime flask", GrenFlash: "flash powder",
		Food: []string{"trail bread", "dried meat", "honeycake"}, Water: "waterskin", Scrap: "salvage",
		Callings:  map[string]string{"warrior": "Sellsword", "scout": "Ranger", "scholar": "Loremaster", "tinker": "Artificer", "mystic": "Priest", "merchant": "Merchant", "rogue": "Cutpurse"},
		Creatures: map[string]string{},
		PowerNames: map[string]string{
			"flames": "Fire Bolt", "freeze": "Rime Ray", "lightning": "Storm Call", "teleport": "Blink", "phase": "Ghostwalk", "regen": "Troll's Blood",
			"sense": "Second Sight", "forcewall": "Arcane Bulwark", "stinger": "Venom Fang", "carapace": "Stoneskin", "darkvision": "Darksight",
			"sprint": "Fleetfoot", "beguile": "Beguile", "heal": "Laying of Hands", "burrow": "Earthmeld", "gills": "Merrow-Gills", "wings": "Wings",
			"psiblast": "Mind Spike", "spikes": "Thornhide", "leech": "Life Drain",
			"swimming": "Strong Swimmer", "diving": "Deep Lungs", "climbing": "Spiderclimb", "clipping": "Shadowstep", "arcanebolt": "Arcane Missile", "slumber": "Sleep",
			"glimmer": "Dazzling Light", "forcepush": "Force Push", "clairvoyance": "Clairvoyance", "timedilation": "Time Dilation", "forcebubble": "Force Bubble",
			"syphon": "Syphon Vim", "stunforce": "Stunning Force", "disintegrate": "Disintegrate", "corrosive": "Corrosive Cloud", "fugue": "Temporal Fugue", "multiarms": "Multiple Arms",
		},
		Bandits: "Brigands", Undead: "the Restless Dead", WallColor: CGray, AccentColor: CYellow,
		Updraft: "wind-stair", Dive: "a sea-cave", Ferry: "book passage on a ship", Kinship: "Share salt",
	}
}

// genreNames2 re-skins the newer powers per genre (spells, tech, psionics).
var genreNames2 = map[string]map[string]string{
	"mythic": {"swimming": "Poseidon's Stroke", "diving": "Pearl-Diver's Breath", "climbing": "Cliff-Goat", "clipping": "Shade-Slip", "arcanebolt": "Hecate's Dart", "slumber": "Breath of Hypnos",
		"glimmer": "Helios' Flare", "forcepush": "Boreas' Gust", "clairvoyance": "Sight of Tiresias", "timedilation": "Chronos' Drag", "forcebubble": "Aegis", "syphon": "Vampire's Kiss",
		"stunforce": "Thunderclap of Zeus", "disintegrate": "Wrath of Ares", "corrosive": "Lernaean Fume", "fugue": "Doppelganger of Proteus", "multiarms": "Hundred-Handed"},
	"steampunk": {"swimming": "Paddle Wheels", "diving": "Compressed-Air Tank", "climbing": "Grapple Gauntlets", "clipping": "Aetheric Shunt", "arcanebolt": "Aether Needle", "slumber": "Chloroform Atomiser",
		"glimmer": "Magnesium Flash", "forcepush": "Steam Ram", "clairvoyance": "Cartographic Engine", "timedilation": "Chronometer Drag", "forcebubble": "Pressure Shield", "syphon": "Transfusion Engine",
		"stunforce": "Concussion Pulse", "disintegrate": "Disruptor Cannon", "corrosive": "Vitriol Sprayer", "fugue": "Clockwork Doubles", "multiarms": "Automaton Arms"},
	"dieselpunk": {"swimming": "Flipper Rig", "diving": "Rebreather Tank", "climbing": "Piton Gloves", "clipping": "Tunnel Charge Step", "arcanebolt": "Tracer Burst", "slumber": "Knockout Gas",
		"glimmer": "Signal Flare", "forcepush": "Concussion Charge", "clairvoyance": "Aerial Recon", "timedilation": "Stutter Field", "forcebubble": "Armor Plating Surge", "syphon": "Blood Transfuser",
		"stunforce": "Shockwave Charge", "disintegrate": "Demolition Charge", "corrosive": "Mustard Canister", "fugue": "Decoy Squad", "multiarms": "Mechanic Arms"},
	"atompunk": {"swimming": "Webbed Hands", "diving": "Big Lungs", "climbing": "Sticky Fingers", "clipping": "Quantum Tunneling", "arcanebolt": "Force Blast", "slumber": "Hypnosis",
		"glimmer": "Photokinesis", "forcepush": "Telekinesis", "clairvoyance": "Clairvoyance", "timedilation": "Temporal Slowing", "forcebubble": "Force Bubble", "syphon": "Syphon Vim",
		"stunforce": "Stunning Force", "disintegrate": "Disintegration", "corrosive": "Acid Spit", "fugue": "Temporal Fugue", "multiarms": "Multiple Arms"},
	"cyberpunk": {"swimming": "Hydro Limbs", "diving": "Oxygen Reservoir", "climbing": "Gecko Grips", "clipping": "Ghost Shunt", "arcanebolt": "Smart Dart", "slumber": "Sleep Virus",
		"glimmer": "Flashbang Eyes", "forcepush": "Kinetic Ram", "clairvoyance": "Satellite Uplink", "timedilation": "Sandevistan", "forcebubble": "Hardlight Shell", "syphon": "Vampire Interface",
		"stunforce": "Shock Pulse", "disintegrate": "Disassembler Beam", "corrosive": "Nano-Acid Cloud", "fugue": "Digital Echoes", "multiarms": "Extra Cyberarms"},
	"biopunk": {"swimming": "Web-Toed Feet", "diving": "Whale Lungs", "climbing": "Gecko Pads", "clipping": "Phase Bloom", "arcanebolt": "Spine Dart", "slumber": "Narcotic Spores",
		"glimmer": "Biolume Burst", "forcepush": "Pressure Burp", "clairvoyance": "Mycelial Mapping", "timedilation": "Sluggish Pheromones", "forcebubble": "Chitin Bubble", "syphon": "Parasitic Tendrils",
		"stunforce": "Sonic Shriek", "disintegrate": "Enzyme Bloom", "corrosive": "Acid Spores", "fugue": "Budding Clones", "multiarms": "Extra Limbs"},
	"solarpunk": {"swimming": "Otter Strokes", "diving": "Kelp Lungs", "climbing": "Vine Grip", "clipping": "Rootslip", "arcanebolt": "Sunbeam", "slumber": "Lullaby Pollen",
		"glimmer": "Solar Flare", "forcepush": "Gale Burst", "clairvoyance": "Canopy Sight", "timedilation": "Sap Slowness", "forcebubble": "Sunlight Shield", "syphon": "Photosynthetic Leech",
		"stunforce": "Thunderbloom", "disintegrate": "Mulch Wave", "corrosive": "Acidic Fog", "fugue": "Seedling Doubles", "multiarms": "Many Branches"},
	"wasteland": {"swimming": "Sludge Paddling", "diving": "Scrubber Lungs", "climbing": "Rebar Scrambling", "clipping": "Rubble Slip", "arcanebolt": "Spark Round", "slumber": "Sedative Fumes",
		"glimmer": "Flare Burst", "forcepush": "Blast Shove", "clairvoyance": "Dead-Radio Sight", "timedilation": "Adrenal Crawl", "forcebubble": "Scrap Plate Bubble", "syphon": "Rad-Leech",
		"stunforce": "Concussive Slam", "disintegrate": "Melt Beam", "corrosive": "Acid Mist", "fugue": "Mirage Doubles", "multiarms": "Spare Arms"},
	"twilight": {"swimming": "Drowned Stroke", "diving": "Held-Breath Vigil", "climbing": "Gargoyle Grip", "clipping": "Mist Slip", "arcanebolt": "Dusk Dart", "slumber": "Lullaby of Ash",
		"glimmer": "Pale Flare", "forcepush": "Hollow Gale", "clairvoyance": "Inner Eye", "timedilation": "Stilled Hour", "forcebubble": "Ward of Vigil", "syphon": "Blood Tithe",
		"stunforce": "Tolling Shock", "disintegrate": "Unmaking Word", "corrosive": "Plague Breath", "fugue": "Wraith Doubles", "multiarms": "Eldritch Limbs"},
}

func SkinFor(genre string) *Skin {
	s := baseSkin()
	s.ID = genre
	switch genre {
	case "mythic":
		s.PowersName, s.Ritual, s.RitualDesc, s.Currency = "Boons", "Xenia", "You swear guest-friendship before the gods", "drachma"
		s.Mats = [6]string{"copper", "bronze", "orichalcum", "celestial bronze", "adamantine", "ichor-forged"}
		s.Nouns = map[string][]string{"longblade": {"xiphos", "kopis"}, "shortblade": {"parazonium", "knife"}, "axe": {"labrys", "pelekys"}, "cudgel": {"club", "hammer"}}
		s.Ranged, s.Ammo = []string{"bow", "sling", "javelin-thrower"}, "arrows"
		s.Armor = map[string][]string{"body": {"linothorax", "bronze cuirass", "muscle cuirass"}, "head": {"pilos", "corinthian helm", "crested helm"}, "feet": {"sandals", "greaves"}, "hands": {"wraps", "bracers"}, "back": {"chlamys", "himation"}, "face": {"veil", "mask"}}
		s.Heal, s.Salve, s.Haste, s.Strength = "ambrosia draught", "asclepian salve", "hermes' draught", "heraclean draught"
		s.Food = []string{"barley cake", "olives", "figs"}
		s.Callings = map[string]string{"warrior": "Hoplite", "scout": "Argonaut", "scholar": "Oracle", "tinker": "Forge-Wright", "mystic": "Hierophant", "merchant": "Emporos", "rogue": "Trickster"}
		s.PowerNames["flames"], s.PowerNames["lightning"], s.PowerNames["wings"], s.PowerNames["regen"] = "Hephaestus' Breath", "Zeus' Bolt", "Icarian Wings", "Hydra's Blood"
		s.PowerNames["phase"], s.PowerNames["beguile"], s.PowerNames["sense"] = "Shade-step", "Siren Song", "Oracular Sight"
		s.Bandits, s.Undead, s.Updraft, s.Ferry = "Raiders", "the Shades", "rainbow stair", "book passage on a trireme"
	case "steampunk":
		s.PowersName, s.PowerVerb, s.Ritual, s.RitualDesc, s.Currency, s.Book = "Contraptions", "engage", "Exchange of Calling Cards", "You exchange engraved calling cards and a pinch of snuff", "sovereigns", "pamphlet"
		s.Light = [3]string{"tallow candle", "carbide lamp", "aether lamp"}
		s.Mats = [6]string{"cast-iron", "wrought-iron", "crucible-steel", "brass-geared", "aether-steel", "mercurial"}
		s.Nouns = map[string][]string{"longblade": {"sabre", "cutlass"}, "shortblade": {"bayonet", "clasp-knife"}, "axe": {"boarding axe", "pick"}, "cudgel": {"wrench", "steam-hammer"}}
		s.Ranged, s.Ammo = []string{"pepperbox pistol", "rifled musket", "aether carbine"}, "cartridges"
		s.Armor = map[string][]string{"body": {"greatcoat", "riveted vest", "boiler plate"}, "head": {"bowler", "pith helmet", "diving helm"}, "feet": {"hobnail boots", "spat-greaves"}, "hands": {"kid gloves", "gauntlets"}, "back": {"cape", "pressure pack"}, "face": {"goggles", "rebreather"}}
		s.Heal, s.Salve, s.Haste, s.Strength, s.Antidote = "laudanum", "carbolic dressing", "galvanic elixir", "bovril tonic", "theriac"
		s.GrenFire, s.GrenFrost, s.GrenFlash = "incendiary bomb", "ether bomb", "magnesium flare"
		s.Food = []string{"tinned beef", "ship's biscuit", "pork pie"}
		s.Water, s.Scrap = "canteen", "cogs"
		s.Callings = map[string]string{"warrior": "Grenadier", "scout": "Explorer", "scholar": "Natural Philosopher", "tinker": "Tinker", "mystic": "Spiritualist", "merchant": "Industrialist", "rogue": "Cracksman"}
		s.PowerNames = map[string]string{"flames": "Flame Gout", "freeze": "Ether Sprayer", "lightning": "Tesla Coil", "teleport": "Aetheric Displacer", "phase": "Phase Engine", "regen": "Galvanic Heart",
			"sense": "Thermal Lenses", "forcewall": "Barricade Projector", "stinger": "Spring-Stiletto", "carapace": "Riveted Plating", "darkvision": "Night Goggles", "sprint": "Piston Legs",
			"beguile": "Mesmerism", "heal": "Field Surgery Kit", "burrow": "Drill Arm", "gills": "Diving Apparatus", "wings": "Ornithopter Wings", "psiblast": "Sonic Projector", "spikes": "Spiked Plating", "leech": "Vital Siphon"}
		s.Bandits, s.Undead, s.WallColor, s.AccentColor = "Highwaymen", "the Clockwork Dead", CBrown, COrange
		s.Updraft, s.Dive, s.Ferry, s.Kinship = "airship mooring", "a diving bell", "book passage on a steamer", "Exchange cards"
	case "dieselpunk":
		s.PowersName, s.PowerVerb, s.Ritual, s.RitualDesc, s.Currency, s.Book = "Augments", "engage", "Comrade's Oath", "You share a cigarette and swear the comrade's oath", "marks", "dossier"
		s.Light = [3]string{"matches", "flashlight", "arc lamp"}
		s.Mats = [6]string{"scrap", "iron", "tool-steel", "chrome", "tungsten", "uranium-tipped"}
		s.Nouns = map[string][]string{"longblade": {"sabre", "machete"}, "shortblade": {"trench knife", "bayonet"}, "axe": {"entrenching tool", "fire axe"}, "cudgel": {"truncheon", "pipe wrench"}}
		s.Ranged, s.Ammo = []string{"service pistol", "bolt-action rifle", "submachine gun"}, "rounds"
		s.Armor = map[string][]string{"body": {"trenchcoat", "flak vest", "trench plate"}, "head": {"field cap", "steel helmet", "tanker helmet"}, "feet": {"jackboots", "puttees"}, "hands": {"gloves", "gauntlets"}, "back": {"rucksack", "parachute pack"}, "face": {"gas mask", "goggles"}}
		s.Heal, s.Salve, s.Haste, s.Strength, s.Antidote = "morphine syrette", "field dressing", "pervitin", "ration stim", "atropine"
		s.GrenFire, s.GrenFrost, s.GrenFlash = "fire bottle", "cryo canister", "stun grenade"
		s.Food = []string{"ration tin", "black bread", "chocolate bar"}
		s.Water, s.Scrap = "canteen", "scrap"
		s.Callings = map[string]string{"warrior": "Stormtrooper", "scout": "Recon", "scholar": "Cryptographer", "tinker": "Mechanic", "mystic": "Chaplain", "merchant": "Smuggler", "rogue": "Saboteur"}
		s.PowerNames = map[string]string{"flames": "Flamethrower", "freeze": "Cryo Projector", "lightning": "Arc Gun", "teleport": "Displacement Belt", "phase": "Phase Field", "regen": "Serum Regimen",
			"sense": "Radio Direction Finder", "forcewall": "Sandbag Kit", "stinger": "Push Dagger Mod", "carapace": "Dermal Steel", "darkvision": "IR Goggles", "sprint": "Pervitin Glands",
			"beguile": "Propaganda Voice", "heal": "Medic Kit", "burrow": "Sapper Drill", "gills": "Frogman Kit", "wings": "Rocket Pack", "psiblast": "Loudhailer", "spikes": "Barbed Harness", "leech": "Transfusion Rig"}
		s.Bandits, s.Undead, s.WallColor, s.AccentColor = "Deserters", "the Gassed Dead", CDarkGray, CBrown
		s.Updraft, s.Dive, s.Ferry, s.Kinship = "zeppelin ladder", "a U-boat hatch", "stow away on a freighter", "Swear the oath"
	case "atompunk":
		s.PowersName, s.PowerVerb, s.Ritual, s.RitualDesc, s.Currency, s.Book = "Mutations", "manifest", "Neighborly Handshake", "You shake hands and share a soda-pop", "caps", "holotape"
		s.Light = [3]string{"glowstick", "flashlight", "atomic lamp"}
		s.Mats = [6]string{"junk", "steel", "titanium", "atomic", "plasma", "quantum"}
		s.Nouns = map[string][]string{"longblade": {"machete", "ripper"}, "shortblade": {"switchblade", "combat knife"}, "axe": {"fire axe", "hatchet"}, "cudgel": {"tire iron", "super-sledge"}}
		s.Ranged, s.Ammo = []string{"ray pistol", "laser rifle", "plasma caster"}, "energy cells"
		s.Armor = map[string][]string{"body": {"jumpsuit", "combat armor", "power armor"}, "head": {"cap", "combat helmet", "power helmet"}, "feet": {"sneakers", "combat boots"}, "hands": {"work gloves", "power gauntlets"}, "back": {"jetpack frame", "lead apron"}, "face": {"sunglasses", "rad mask"}}
		s.Heal, s.Salve, s.Haste, s.Strength, s.Antidote = "stimpak", "rad-away", "jet inhaler", "buffout", "anti-venom"
		s.GrenFire, s.GrenFrost, s.GrenFlash = "molotov", "cryo grenade", "pulse grenade"
		s.Food = []string{"canned beans", "snack cakes", "rad-roach jerky"}
		s.Water, s.Scrap = "water jug", "scrap"
		s.Callings = map[string]string{"warrior": "Ranger", "scout": "Scavver", "scholar": "Scientist", "tinker": "Engineer", "mystic": "Atomic Monk", "merchant": "Salesman", "rogue": "Grifter"}
		s.PowerNames["flames"], s.PowerNames["regen"], s.PowerNames["sense"], s.PowerNames["carapace"] = "Pyrokinesis", "Rapid Regeneration", "Sense Psychic", "Quills & Carapace"
		s.PowerNames["phase"], s.PowerNames["teleport"], s.PowerNames["lightning"], s.PowerNames["wings"] = "Phasing", "Teleportation", "Electrical Generation", "Wings"
		s.PowerNames["beguile"], s.PowerNames["psiblast"], s.PowerNames["gills"], s.PowerNames["burrow"] = "Domination", "Psionic Blast", "Gills", "Burrowing Claws"
		s.Bandits, s.Undead, s.WallColor, s.AccentColor = "Raiders", "the Ghouls", CGray, CGreen
		s.Updraft, s.Dive, s.Ferry, s.Kinship = "orbital lift", "an airlock", "hitch a ride on a hover-barge", "Shake hands"
	case "cyberpunk":
		s.PowersName, s.PowerVerb, s.Ritual, s.RitualDesc, s.Currency, s.Book = "Cyberware", "activate", "Handshake Protocol", "You swap encrypted keys and blood-oath ICE", "creds", "datashard"
		s.Light = [3]string{"chemlight", "headlamp", "flood drone"}
		s.Mats = [6]string{"rebar", "carbon", "ceramic", "mono", "thermal", "nanofilament"}
		s.Nouns = map[string][]string{"longblade": {"katana", "machete"}, "shortblade": {"shiv", "tanto"}, "axe": {"cleaver", "chopper"}, "cudgel": {"stun baton", "pipe"}}
		s.Ranged, s.Ammo = []string{"smartpistol", "flechette rifle", "rail gun"}, "caseless rounds"
		s.Armor = map[string][]string{"body": {"synthleather jacket", "kevlar vest", "armorweave"}, "head": {"hoodie", "smart helmet", "riot helmet"}, "feet": {"sneakers", "mag-boots"}, "hands": {"grip gloves", "knuckle rig"}, "back": {"cyberdeck harness", "drone pack"}, "face": {"shades", "rebreather"}}
		s.Heal, s.Salve, s.Haste, s.Strength, s.Antidote = "trauma stim", "nanopaste", "synaptic accelerant", "berserker stim", "detox shot"
		s.GrenFire, s.GrenFrost, s.GrenFlash = "thermite grenade", "cryo grenade", "EMP flashbang"
		s.Food = []string{"soy bar", "vat noodles", "kibble pack"}
		s.Water, s.Scrap = "water pouch", "e-waste"
		s.Callings = map[string]string{"warrior": "Street Samurai", "scout": "Runner", "scholar": "Netrunner", "tinker": "Techie", "mystic": "Data-Shaman", "merchant": "Fixer", "rogue": "Ghost"}
		s.PowerNames = map[string]string{"flames": "Plasma Projector", "freeze": "Cryo Emitter", "lightning": "Shock Hands", "teleport": "Spatial Jumper", "phase": "Optical Camo", "regen": "Nanite Swarm",
			"sense": "Bio-Scanner", "forcewall": "Hardlight Barrier", "stinger": "Mantis Blades", "carapace": "Subdermal Armor", "darkvision": "Low-light Optics", "sprint": "Kerenzikov",
			"beguile": "Neural Hijack", "heal": "Auto-Medic", "burrow": "Gorilla Arms", "gills": "Oxygen Lung", "wings": "Grav-Boots", "psiblast": "Sonic Shock", "spikes": "Monowire Mesh", "leech": "Bio-Leech"}
		s.Bandits, s.Undead, s.WallColor, s.AccentColor = "Gangers", "the Cyberpsychos", CDarkCyan, CMagenta
		s.Updraft, s.Dive, s.Ferry, s.Kinship = "AV pad", "a pressure lock", "buy a berth on a cargo hauler", "Swap keys"
	case "biopunk":
		s.PowersName, s.Ritual, s.RitualDesc, s.Currency, s.Book = "Splices", "Pheromone Kinship", "You trade pheromone sigils and taste each other's sweat", "biomass", "gene-scroll"
		s.Light = [3]string{"glowworm jar", "biolume", "firefly swarm"}
		s.Mats = [6]string{"bone", "chitin", "keratin", "ironwood", "spidersilk", "living"}
		s.Nouns = map[string][]string{"longblade": {"bone-sword", "mantis scythe"}, "shortblade": {"fang-knife", "stinger"}, "axe": {"chitin-axe", "claw"}, "cudgel": {"knot-club", "horn maul"}}
		s.Ranged, s.Ammo = []string{"spine-launcher", "acid spitter", "dart-gun"}, "spines"
		s.Armor = map[string][]string{"body": {"skin-suit", "chitin shell", "living carapace"}, "head": {"shell cap", "horned helm", "brain-hood"}, "feet": {"pad-boots", "hoof-greaves"}, "hands": {"grip-pads", "claw-gloves"}, "back": {"brood-sac", "gill-mantle"}, "face": {"filter-mask", "eye-cluster"}}
		s.Heal, s.Salve, s.Haste, s.Strength, s.Antidote = "regen-gel", "clot moss", "adrenal gland", "myostatin shot", "antitoxin"
		s.GrenFire, s.GrenFrost, s.GrenFlash = "fire-beetle", "rime-spore", "flash-pod"
		s.Food = []string{"vat-meat", "spore bread", "nutrient grub"}
		s.Water, s.Scrap = "water bladder", "biomass scraps"
		s.Callings = map[string]string{"warrior": "Gene-Brute", "scout": "Stalker", "scholar": "Geneticist", "tinker": "Vat-Wright", "mystic": "Spore-Priest", "merchant": "Biomass Broker", "rogue": "Mimic"}
		s.PowerNames["flames"], s.PowerNames["regen"], s.PowerNames["stinger"], s.PowerNames["carapace"] = "Pyro Glands", "Axolotl Genes", "Scorpion Tail", "Chitin Plating"
		s.PowerNames["wings"], s.PowerNames["gills"], s.PowerNames["burrow"], s.PowerNames["beguile"] = "Membrane Wings", "Gills", "Mole Claws", "Pheromone Thrall"
		s.Bandits, s.Undead, s.WallColor, s.AccentColor = "Feral Splicers", "the Fungal Hosts", CDarkGreen, CGreen
		s.Updraft, s.Dive, s.Ferry, s.Kinship = "gasbag tether", "a kelp-throat", "ride a sea-beast", "Trade sigils"
	case "solarpunk":
		s.PowersName, s.Ritual, s.RitualDesc, s.Currency, s.Book = "Symbionts", "Seedling Pact", "You plant a seedling together and share water", "credits", "zine"
		s.Light = [3]string{"solar lantern", "biolume", "sun-mirror"}
		s.Mats = [6]string{"salvage", "bamboo", "recycled-steel", "carbon-fiber", "graphene", "solar-forged"}
		s.Nouns = map[string][]string{"longblade": {"machete", "glaive"}, "shortblade": {"pruning knife", "multitool"}, "axe": {"woodsman's axe", "billhook"}, "cudgel": {"quarterstaff", "mallet"}}
		s.Ranged, s.Ammo = []string{"slingshot", "recurve bow", "coil-gun"}, "bolts"
		s.Armor = map[string][]string{"body": {"hemp jacket", "cork vest", "exo-frame"}, "head": {"sunhat", "bike helmet", "exo-helm"}, "feet": {"sandals", "trail boots"}, "hands": {"garden gloves", "exo-gloves"}, "back": {"solar pack", "kite-wing"}, "face": {"sun visor", "filter mask"}}
		s.Heal, s.Salve, s.Haste, s.Strength, s.Antidote = "herbal tincture", "aloe poultice", "yerba stim", "mycoprotein shake", "charcoal pill"
		s.GrenFire, s.GrenFrost, s.GrenFlash = "fire seed", "frost seed", "flash seed"
		s.Food = []string{"garden salad", "lentil cake", "rooftop honey"}
		s.Water, s.Scrap = "water bottle", "recyclables"
		s.Callings = map[string]string{"warrior": "Warden", "scout": "Pathfinder", "scholar": "Archivist", "tinker": "Fixer", "mystic": "Druid", "merchant": "Cooperative Trader", "rogue": "Saboteur"}
		s.PowerNames["regen"], s.PowerNames["sense"], s.PowerNames["spikes"], s.PowerNames["wings"] = "Mycelial Mending", "Root Network", "Bramble Skin", "Kite-Wings"
		s.PowerNames["beguile"], s.PowerNames["gills"], s.PowerNames["leech"] = "Kinship Pollen", "Algae Lung", "Photosynthetic Drain"
		s.Bandits, s.Undead, s.WallColor, s.AccentColor = "Saboteurs", "the Blighted", CBrown, CGreen
		s.Updraft, s.Dive, s.Ferry, s.Kinship = "kite-tether", "a reef-lock", "sail on a co-op ship", "Plant a seed"
	case "wasteland":
		s.PowersName, s.PowerVerb, s.Ritual, s.RitualDesc, s.Currency, s.Book = "Mutations", "manifest", "Shared Ration", "You split a ration tin and swap names by the fire", "chits", "salvaged log"
		s.Light = [3]string{"road flare", "hand lamp", "sodium floodlight"}
		s.Mats = [6]string{"scrap", "rebar", "road-steel", "reclaimed alloy", "vault-grade", "pre-fall"}
		s.Nouns = map[string][]string{"longblade": {"sheet-metal cleaver", "road machete"}, "shortblade": {"shiv", "box-cutter"}, "axe": {"fire axe", "hatchet"}, "cudgel": {"pipe club", "rail-spike maul"}}
		s.Ranged, s.Ammo = []string{"pipe pistol", "scrap rifle", "pump shotgun"}, "shells"
		s.Armor = map[string][]string{"body": {"leather duster", "tire-tread vest", "vault plate"}, "head": {"hard hat", "welder's helm", "riot helm"}, "feet": {"work boots", "shin-greaves"}, "hands": {"work gloves", "scrap gauntlets"}, "back": {"scav pack", "tarp cloak"}, "face": {"dust goggles", "gas mask"}}
		s.Heal, s.Salve, s.Haste, s.Strength, s.Antidote = "stim-shot", "purge paste", "runner's fizz", "bruiser's draught", "snake serum"
		s.GrenFire, s.GrenFrost, s.GrenFlash = "fuel-can bomb", "coolant bomb", "flare charge"
		s.Food = []string{"canned beans", "mole-rat jerky", "dust-corn cake"}
		s.Water, s.Scrap = "jerrycan", "scrap"
		s.Callings = map[string]string{"warrior": "Road Warrior", "scout": "Pathfinder", "scholar": "Archivist", "tinker": "Scrapwright", "mystic": "Ash Preacher", "merchant": "Caravaneer", "rogue": "Tunnel Thief"}
		s.PowerNames["flames"], s.PowerNames["regen"], s.PowerNames["sense"], s.PowerNames["carapace"] = "Burning Touch", "Rad-Fed Flesh", "Static Sense", "Hide Plates"
		s.PowerNames["phase"], s.PowerNames["teleport"], s.PowerNames["lightning"], s.PowerNames["wings"] = "Dust Step", "Rift Hop", "Arc Surge", "Scrap Wings"
		s.PowerNames["beguile"], s.PowerNames["psiblast"], s.PowerNames["gills"], s.PowerNames["burrow"] = "Hollow Voice", "Skull Ringer", "Filter Gills", "Tunnel Claws"
		s.Bandits, s.Undead, s.WallColor, s.AccentColor = "Raiders", "the Glowing Dead", CBrown, CGreen
		s.Updraft, s.Dive, s.Ferry, s.Kinship = "cable lift", "a flooded hatch", "hitch a ride on a caravan rig", "Split a ration"
	case "twilight":
		s.PowersName, s.PowerVerb, s.Ritual, s.RitualDesc, s.Currency, s.Book = "Insights", "awaken", "Vigil Oath", "You light a candle together and swear to keep the vigil", "echoes", "codex"
		s.Light = [3]string{"tallow torch", "oil lantern", "pale censer"}
		s.Mats = [6]string{"bone", "iron", "silvered steel", "blackened", "moonlit", "godbone"}
		s.Nouns = map[string][]string{"longblade": {"greatsword", "hunter's blade"}, "shortblade": {"misericorde", "stiletto"}, "axe": {"cleaver", "hewing axe"}, "cudgel": {"flail", "censer-mace"}}
		s.Ranged, s.Ammo = []string{"hand crossbow", "blunderbuss", "reliquary caster"}, "quicksilver shot"
		s.Armor = map[string][]string{"body": {"tattered coat", "hunter's garb", "gilded plate"}, "head": {"tricorn hat", "plague hood", "crowned helm"}, "feet": {"worn boots", "greaves"}, "hands": {"leather gloves", "gauntlets"}, "back": {"threadbare cloak", "ash mantle"}, "face": {"beaked mask", "gilded visor"}}
		s.Heal, s.Salve, s.Haste, s.Strength, s.Antidote = "blood vial", "pale ointment", "quickening draught", "ember tonic", "bitter tincture"
		s.GrenFire, s.GrenFrost, s.GrenFlash = "pyre jar", "grave-frost jar", "lantern flash"
		s.Food = []string{"hard biscuit", "salted eel", "grave honey"}
		s.Water, s.Scrap = "flask", "relics"
		s.Callings = map[string]string{"warrior": "Vigil Knight", "scout": "Wayfinder", "scholar": "Seer", "tinker": "Reliquarist", "mystic": "Wardenpriest", "merchant": "Ash Trader", "rogue": "Gutter Cutpurse"}
		s.PowerNames["flames"], s.PowerNames["freeze"], s.PowerNames["lightning"], s.PowerNames["regen"] = "Pyre Rite", "Grave Chill", "Sundered Bolt", "Blood Mending"
		s.PowerNames["sense"], s.PowerNames["phase"], s.PowerNames["beguile"], s.PowerNames["psiblast"] = "Inner Eye", "Mist Walk", "Whispered Lure", "Mind Rend"
		s.PowerNames["heal"], s.PowerNames["wings"], s.PowerNames["leech"] = "Last Mercy", "Gargoyle Wings", "Blood Tithe"
		s.Bandits, s.Undead, s.WallColor, s.AccentColor = "Cutthroats", "the Hollowed", CDarkGray, CMagenta
		s.Updraft, s.Dive, s.Ferry, s.Kinship = "bell-rope lift", "a drowned stair", "pay a ferryman", "Light a candle"
	}
	for k, v := range genreNames2[genre] {
		s.PowerNames[k] = v
	}
	return &s
}

func (s *Skin) Calling(a string) string {
	if v, ok := s.Callings[a]; ok {
		return v
	}
	return titleCase(a)
}

func (s *Skin) PowerName(id string) string {
	if v, ok := s.PowerNames[id]; ok {
		return v
	}
	if v, ok := baseSkin().PowerNames[id]; ok {
		return v
	}
	return titleCase(id)
}
