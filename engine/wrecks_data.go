package engine

// Sunken treasure data (prefix wr): ship names, cargo labels per genre, logbook voices, rumours,
// diving gear. Pure tables; the logic lives in wrecks.go.

// ship names: a genre list of whole names, or adjective + noun when the genre has none (or half the time)
var wrShipWhole = map[string][]string{
	"cyberpunk":  {"Neon Gull", "Static Tide", "Ghostline", "Blackout Runner", "Chrome Heron", "Zero Hour", "Packet Loss", "Dead Channel"},
	"wasteland":  {"Rust Gull", "Last Jerrycan", "Salt Mule", "Dust Heron", "Scrapper's Hope", "Thirsty Annie", "Bone Barge"},
	"steampunk":  {"Brass Heron", "Iron Duchess", "Steam Albatross", "Ember Clipper", "Her Majesty's Folly", "Cog and Compass"},
	"dieselpunk": {"Hard Knock", "Oil Queen", "Gray Wolf", "Steel Gannet", "Pitch Lady", "Kestrel Eight"},
	"atompunk":   {"Atomic Heron", "Sputnik Gull", "Rocket Maiden", "Fallout Queen", "Chrome Albatross"},
	"biopunk":    {"Living Tide", "Spore Mother", "Bloom Gull", "Marrow Wake", "Third Lung", "Gill Duchess"},
	"solarpunk":  {"Sunrise Heron", "Green Wake", "Windfall", "Kite Dancer", "Seed of Morrow", "Bright Current"},
	"mythic":     {"Argo's Daughter", "Poseidon's Gift", "Nereid", "Golden Fleece", "Siren's Wake", "Tyche's Bargain"},
	"twilight":   {"Pale Vigil", "Lantern Gull", "Mourning Star", "Hollow Bell", "Candle of Saint Elm", "Last Rites"},
}

var wrShipA = []string{"Grey", "Swift", "Salt", "Storm", "Black", "Silver", "Red", "Gilded", "Wandering", "Golden", "Restless", "Proud", "Merciful", "Hollow", "Bright", "Wayward"}
var wrShipB = []string{"Heron", "Widow", "Wolf", "Maiden", "Serpent", "Star", "Fortune", "Tide", "Wake", "Petrel", "Dolphin", "Gannet", "Vigil", "Promise", "Cormorant", "Lantern"}

// wrCargoDef is one kind of cargo; Name varies by genre ("" is the default).
type wrCargoDef struct {
	Value int
	Name  map[string]string
}

var wrCargoDefs = map[string]wrCargoDef{
	"spice":      {90, map[string]string{"": "casks of clove and nutmeg", "wasteland": "crates of dried herbs and seed-grain", "cyberpunk": "vacuum-packed flavour concentrate", "biopunk": "pods of spore-spice", "solarpunk": "crates of saffron and vanilla", "atompunk": "tins of exotic spice"}},
	"tin":        {60, map[string]string{"": "ingots of tin and copper", "steampunk": "boiler plate and brass bar", "dieselpunk": "coils of rolled steel", "atompunk": "lead-lined ore sacks", "cyberpunk": "salvaged server racks", "wasteland": "ingots of scrap copper", "biopunk": "bales of chitin plate", "solarpunk": "crates of photovoltaic wafers", "mythic": "ingots of bronze", "twilight": "bars of pewter and silver"}},
	"treasury":   {200, map[string]string{"": "the royal treasury chests", "cyberpunk": "the vault's bearer-bond cores", "biopunk": "bundled bone-coin", "wasteland": "the warlord's bottlecap chests", "atompunk": "the ministry's gold bullion", "mythic": "the temple tribute of the king", "solarpunk": "the commons' reserve of trade-tokens"}},
	"relics":     {150, map[string]string{"": "reliquaries and temple plate", "cyberpunk": "shrine-servers and icon cores", "wasteland": "relics of the old world", "biopunk": "sealed totem-pods", "twilight": "saints' bones in silver cases"}},
	"powder":     {80, map[string]string{"": "kegs of alchemist's fire", "steampunk": "kegs of black powder", "dieselpunk": "kegs of black powder", "atompunk": "sealed isotope canisters", "cyberpunk": "crates of blast charges", "biopunk": "clutches of spore-bombs", "solarpunk": "stacks of battery cells", "wasteland": "jerry cans of fuel", "mythic": "jars of Greek fire", "twilight": "casks of blessed lamp-oil"}},
	"vats":       {170, map[string]string{"": "sealed apothecary jars", "mythic": "amphorae of ambrosia", "steampunk": "cases of chemical reagents", "dieselpunk": "crates of penicillin", "atompunk": "isotope medicine", "cyberpunk": "cryo-vats of gene stock", "biopunk": "gene-vats", "solarpunk": "seed-vaults", "wasteland": "water-purifier cores", "twilight": "reliquary jars of salt and blood"}},
	"silk":       {75, map[string]string{"": "bolts of silk", "wasteland": "parachute silk and canvas", "cyberpunk": "bolts of smart-fabric", "biopunk": "bales of spun mycelium"}},
	"wine":       {50, map[string]string{"": "amphorae of wine", "wasteland": "crates of distilled spirits", "cyberpunk": "crates of designer liquor", "biopunk": "casks of fermented nectar"}},
	"contraband": {110, map[string]string{"": "smuggled goods under false bottoms", "cyberpunk": "black-market chips and implants", "wasteland": "pre-fall weapons", "twilight": "forbidden books and relics"}},
}

// wrCargoOrder lists which cargo each source of loss tends to carry.
var wrCargoOrder = map[string][]string{
	"war":        {"powder", "treasury", "tin", "powder", "vats"},
	"storm":      {"spice", "silk", "tin", "wine", "relics", "vats"},
	"tsunami":    {"spice", "silk", "tin", "wine", "relics"},
	"plague":     {"vats", "silk", "spice"},
	"expedition": {"relics", "treasury", "vats", "tin"},
	"embargo":    {"contraband"},
	"spice":      {"spice"},
	"kraken":     {"tin", "treasury", "silk", "spice"},
	"light":      {"treasury"},
}

// wrLogs are the captain's last words by source. Placeholders: {captain} {ship} {realm} {year} {place} {cargo}.
var wrLogs = map[string][]string{
	"war": {
		"Day 3 out of port. {realm} has sent every hull it owns after us and we carry {cargo}. The enemy sails closer by the hour. If this book is found, tell them the {ship} struck no colours.",
		"They have holed us below the line twice. The pumps cannot keep up. I have locked {cargo} in my own strongbox and the key stays on my neck. Year {year}, a bad year to be a sailor of {realm}.",
	},
	"storm": {
		"The glass fell all night and by dawn the sea stood up like a wall. {cargo} shifted in the hold and took the {ship} onto her side. I write this lashed to the table in the cabin, where the air still holds.",
		"Three days of storm. The mast is gone, the helmsman is gone. We drift in the dark with {cargo} aboard and nothing to steer by. Year {year}. If found, return the strongbox to the owners in {realm}.",
	},
	"tsunami": {
		"The sea drew back from the shore as if breathing in, and every captain knew what comes after. We ran for deep water with {cargo} aboard. The wave caught us anyway. {realm} will count its dead for years.",
	},
	"plague": {
		"The port would not take us. The fever is in all of us, and {realm} turned its guns on the harbour mouth. I have chained the hatches and sail out to die with {cargo}, so that no one is made to touch it.",
		"Eleven dead, nineteen sick. We were told to keep to the open sea until it passed. It will not pass. This log is the last honest account of the {ship}.",
	},
	"kraken": {
		"It rose beside us in broad daylight, an arm thick as a mainmast. The lookout screamed once. I have {cargo} below and no way to save it. The sea around is black with its ink. Year {year}.",
		"Something has been following the {ship} for six days. Tonight it took the watch, three at a time, without a splash. I think it wants the cargo, or the hull. I cannot tell which.",
	},
	"expedition": {
		"We sailed to find the edge of the charts and carried {cargo} to trade for it. The stars are wrong here and the compass sulks. Whoever reads this, the {ship} did not turn back, and neither did we.",
		"Provisions short. The crew mutters. {realm} promised us glory and sent us out with {cargo} and not enough water. Year {year}, and no landfall.",
	},
	"embargo": {
		"The embargo stands and the patrol boats are thick as flies, but a cargo of {cargo} pays three times over if it reaches the quay. We ran dark. They saw us anyway. Burn this book if you are {realm}'s officer.",
	},
	"spice": {
		"Holds full to the hatches with {cargo}, we cleared the harbour with the whole town cheering. The ship was overloaded and I knew it. Year {year}, and the sea is gentle, which is the worst sign.",
	},
	"light": {
		"The tower stood two hundred years and I was keeper for forty. Tonight the sea comes up the stairs. {cargo} lies in the lamp-room where the harbour dues were kept. I will not leave the light.",
	},
	"lost": {
		"We have lost the {ship}. I write for whoever finds her: {cargo} is aboard, and the strongbox is in my cabin where the air still holds.",
	},
}

var wrCityTales = []string{
	"The sea came in a single night. Those who ran were saved; those who counted their coin were not.",
	"Bells were still ringing in the temple when the water closed over the roofs. They say the ringing never fully stopped.",
	"The magistrates sealed the vault before the wave came, and no one lived to say where they put the key.",
}

// wrGear is what the shipwright sells for deep work.
type wrGearDef struct {
	ID    string
	Tier  int // 0 = tool, 1 = rig, 2 = bell, 3 = deep suit
	Air   int // minutes of air
	Price int
	Name  map[string]string
	Desc  string
}

var wrGearDefs = []wrGearDef{
	{"rig", 1, 60, 180, map[string]string{"": "diver's leather hood and air-bladder", "mythic": "sponge-diver's bronze mask and weights", "steampunk": "brass diving helmet and air-hose", "dieselpunk": "hard-hat diving rig", "atompunk": "rebreather aqualung", "cyberpunk": "subdermal O2 rebreather", "biopunk": "gill-graft symbiote", "solarpunk": "algae-lung vest", "wasteland": "gas-mask and salvaged scuba tank", "twilight": "ghost-glass diving mask and bellows"},
		"60 minutes of air and a safe two fathoms. Carry it in your pack; it feeds on its own."},
	{"bell", 2, 120, 650, map[string]string{"": "diving bell", "mythic": "bronze diving cauldron", "steampunk": "riveted diving bell", "dieselpunk": "steel diving bell", "atompunk": "pressure-sphere bell", "cyberpunk": "tethered dive pod", "biopunk": "grown air-bell", "solarpunk": "glass dive-bubble", "wasteland": "boiler-tank bell", "twilight": "bell of blessed bronze"},
		"120 minutes of air, three fathoms, and a winch: with it salvage lots weigh nothing."},
	{"suit", 3, 90, 1500, map[string]string{"": "deep-sea suit", "mythic": "suit of oiled hides and pearl-glass", "steampunk": "armoured pressure suit", "dieselpunk": "armoured pressure suit", "atompunk": "atmospheric diving suit", "cyberpunk": "hardshell dive frame", "biopunk": "pressure-adapted graft suit", "solarpunk": "braced kelp-weave suit", "wasteland": "welded armour suit", "twilight": "leaden saint's habit"},
		"90 minutes of air and the pressure of four fathoms. Salvage lots weigh nothing."},
	{"line", 0, 0, 90, map[string]string{"": "sounding line", "mythic": "lead and knotted cord", "cyberpunk": "depth-ping probe", "atompunk": "echo-sounder", "steampunk": "brass sounding lead", "biopunk": "listening polyp", "solarpunk": "chime-sounder", "wasteland": "weighted wire", "dieselpunk": "echo-sounder"},
		"Cast it from the ship screen to feel what lies beneath."},
}

var wrDirNames = map[string]string{"N": "north", "NE": "north-east", "E": "east", "SE": "south-east", "S": "south", "SW": "south-west", "W": "west", "NW": "north-west"}

var wrHazardText = map[string]string{
	"eel":     "eels in the timbers",
	"shark":   "sharks circling",
	"current": "a hard current",
	"ghost":   "the drowned crew still on watch",
	"kraken":  "something very large asleep nearby",
}

var wrRumourOpen = []string{
	"\"Ask any old hand on this quay and you will hear of %s.",
	"\"%s? Aye. Every harbour has one ship it never stops talking about.",
	"\"My grandfather used to tell this over the rope-walk. %s.",
}
