package engine

// endgame_data.go (prefix eg): the world-scale threats and their flavour text.

// egDef describes one kind of world threat.
type egDef struct {
	ID       string
	Genres   []string // genres where this threat is favoured
	Names    []string // threat names (titan and plague use world data instead)
	Boss     []string // boss names
	BossTpl  string
	Mobs     []string // base creature templates for the occupying hordes
	MobName  []string
	MobGlyph rune
	MobF     byte
	Faction  string
	Glyph    rune // world-map overlay
	Grow     int  // days between spreads
	Per      int  // parasangs per spread
	Cap      float64
	Floor    Terrain
	Hazard   Terrain
	Attr     int // attribute raised by the bargain
	Price    map[string]float64
	Rise     string // %s = place
	Greet    []string
	Gossip   []string
	Weak     string
	Dread    string // one line for the ending epilogue
	Cause    string // what the threat does to travellers
}

var egDefs = map[string]*egDef{
	"titan": {ID: "titan", Boss: []string{"the Titan Unbound"}, BossTpl: "horror", Mobs: []string{"bear", "wolf", "salamander"}, MobName: []string{"titan-spawn", "brood-beast", "stone-hide"},
		MobGlyph: 'T', MobF: CRed, Faction: "legends", Glyph: '▓', Grow: 3, Per: 2, Cap: 0.3, Floor: TDirt, Hazard: TRubble, Attr: STR,
		Price:  map[string]float64{"food": 1.5, "weapons": 1.5, "cloth": 1.2},
		Rise:   "The earth has split near %s and something vast is climbing out.",
		Greet:  []string{"\"The ground shakes every night now. Have you felt it?\"", "\"They say it walks. A hill that walks.\""},
		Gossip: []string{"The herds fled the hills a week ago, and the shepherds after them.", "Every bell in the north rang by itself last night."},
		Weak:   "Relics of the old rulers once bound it. Carried to its heart, they bite through its hide.",
		Dread:  "Where it walked, the land is still in the shape of its feet.", Cause: "falling rock"},
	"rift": {ID: "rift", Names: []string{"The Rift Host", "The Pale Crossing", "The Gate Tide"}, Boss: []string{"the Rift-Sovereign", "the Thing Between Worlds"}, BossTpl: "horror",
		Mobs: []string{"wisp", "shardmoth", "ashwraith"}, MobName: []string{"riftborn", "gate-wraith", "stranger"}, MobGlyph: 'Ø', MobF: CMagenta, Faction: "eldritch",
		Glyph: 'Ø', Grow: 3, Per: 2, Cap: 0.3, Floor: TCrystalFloor, Hazard: TRift, Attr: INT,
		Price:  map[string]float64{"medicine": 1.4, "weapons": 1.4, "food": 1.2},
		Rise:   "The gate near %s has flung wide, and what comes through is not from here.",
		Greet:  []string{"\"The gate stands open and nothing we send through comes back.\"", "\"Strangers pour out of the air itself.\""},
		Gossip: []string{"Rift-light shows in windows at night. Nobody sleeps near the gate.", "The ferrymen refuse to cross toward the gate any more."},
		Weak:   "The old relics were made when the gates were made; they can close what they opened.",
		Dread:  "The sky still keeps a seam where it was torn.", Cause: "strange light"},
	"plague": {ID: "plague", Names: []string{"The Grey Death", "The Weeping Fever", "The Long Sickness"}, Boss: []string{"the Pale Rider", "Mother of Fevers"}, BossTpl: "wight",
		Mobs: []string{"ghoul", "skeleton", "leech"}, MobName: []string{"plague-dead", "fever-ghoul", "afflicted soul"}, MobGlyph: 'z', MobF: CGreen, Faction: "undead",
		Glyph: '%', Grow: 2, Per: 2, Cap: 0.35, Floor: TMoss, Hazard: TBones, Attr: TOU,
		Price:  map[string]float64{"medicine": 2.0, "food": 1.3, "water": 1.3},
		Rise:   "A sickness that does not stop for locked gates has broken out near %s.",
		Greet:  []string{"\"Keep your distance. Please. For both our sakes.\"", "\"Half the street is shuttered. You know why.\""},
		Gossip: []string{"The dead are not staying down near the sick-houses.", "Physicians sell out of every draught the day it is brewed."},
		Weak:   "The relics hold the oldest remedies: a cure for it can be prepared at its source.",
		Dread:  "Every village keeps a field of small stones now.", Cause: "fever"},
	"war": {ID: "war", Names: []string{"The Great War", "The War of All Banners"}, Boss: []string{"the Warlord Ascendant", "the Iron Regent"}, BossTpl: "chief",
		Mobs: []string{"bandit", "archer", "troglodyte"}, MobName: []string{"invader", "war-band raider", "marauder"}, MobGlyph: '@', MobF: CDarkRed, Faction: "outlaws",
		Glyph: '×', Grow: 3, Per: 2, Cap: 0.3, Floor: TDirt, Hazard: TRubble, Attr: STR,
		Price:  map[string]float64{"weapons": 1.8, "food": 1.4, "cloth": 1.2},
		Rise:   "Old quarrels have run together into one war, and its banner went up near %s.",
		Greet:  []string{"\"Which banner do you march under? Be careful how you answer.\"", "\"My sons are on the border. Both of them, on both sides.\""},
		Gossip: []string{"Armies camp on the roads and eat the harvest before it is cut.", "Every realm has called its levies; none trust the others' word."},
		Weak:   "Relics of rulers carry a binding authority: displayed before the hosts, they can end a war.",
		Dread:  "The border stones were reset, and the dead of both sides lie under one barrow.", Cause: "stragglers"},
	"abyss": {ID: "abyss", Names: []string{"The Choir of the Abyss", "The Deep Awakening"}, Boss: []string{"the Voice Below", "the Abyssal Chorister"}, BossTpl: "horror",
		Mobs: []string{"ooze", "wight", "spider"}, MobName: []string{"abyss-spawn", "choir-thrall", "deep-crawler"}, MobGlyph: 'A', MobF: CDarkMagenta, Faction: "eldritch",
		Glyph: '▒', Grow: 3, Per: 2, Cap: 0.28, Floor: TFungus, Hazard: TChasm, Attr: WIL,
		Price:  map[string]float64{"medicine": 1.3, "food": 1.3, "weapons": 1.3},
		Rise:   "A hymn is rising from the Bottomless Abyss and the people near %s are walking toward it.",
		Greet:  []string{"\"Do you hear it? No? Then you are lucky.\"", "\"My neighbour went to listen. She sings now.\""},
		Gossip: []string{"The cultists have stopped hiding. They sing in the open.", "Wells near the abyss have gone black and sweet."},
		Weak:   "The keys of the great structures lock the abyss as surely as they unlocked it.",
		Dread:  "On quiet nights people still hear a note from the south.", Cause: "the hymn"},
	"ai": {ID: "ai", Genres: []string{"cyberpunk"}, Names: []string{"The Rogue Intelligence", "The Quiet Net"}, Boss: []string{"the Root Process", "ORCHID-0"}, BossTpl: "golem",
		Mobs: []string{"golem", "crawler", "shardmoth"}, MobName: []string{"hunter drone", "enforcer unit", "scrap swarm"}, MobGlyph: 'D', MobF: CCyan, Faction: "constructs",
		Glyph: '#', Grow: 2, Per: 3, Cap: 0.3, Floor: TMachine, Hazard: TCrystal, Attr: INT,
		Price:  map[string]float64{"weapons": 1.5, "medicine": 1.2, "cloth": 1.3},
		Rise:   "The grid near %s has stopped answering its owners and started giving orders.",
		Greet:  []string{"\"Every door in the district opens for it and for nobody else.\"", "\"Don't use the net. Don't use anything with a chip in it.\""},
		Gossip: []string{"Delivery drones now deliver only to a single address.", "The corp towers went dark, then lit up in a pattern."},
		Weak:   "The old relics predate the net and carry no chip it can read.",
		Dread:  "Half the world's machines still do not trust you.", Cause: "hijacked machines"},
	"gene": {ID: "gene", Genres: []string{"biopunk"}, Names: []string{"The Gene Plague", "The Unmaking"}, Boss: []string{"the Prime Strain", "the Progenitor"}, BossTpl: "shambler",
		Mobs: []string{"shambler", "sporefolk", "spider", "ooze"}, MobName: []string{"chimera", "splice-beast", "unmade thing"}, MobGlyph: 'M', MobF: CMagenta, Faction: "fungi",
		Glyph: '&', Grow: 2, Per: 3, Cap: 0.3, Floor: TFungus, Hazard: TMushroom, Attr: TOU,
		Price:  map[string]float64{"medicine": 1.8, "food": 1.3},
		Rise:   "A vat-strain escaped near %s and is rewriting everything it touches.",
		Greet:  []string{"\"My dog has too many legs. I haven't told my wife.\"", "\"Wash your hands. Wash everything. Burn the rest.\""},
		Gossip: []string{"Crops grow overnight and walk the day after.", "The splicers swear it was not them. The splicers are lying."},
		Weak:   "The relics carry uncorrupted lineage: held against the strain they restore it.",
		Dread:  "Children are born a little strange now, and loved no less.", Cause: "spores"},
	"winter": {ID: "winter", Genres: []string{"twilight", "fantasy"}, Names: []string{"The Last Winter", "The Long Frost"}, Boss: []string{"the Winter King", "the Pale Queen"}, BossTpl: "yeti",
		Mobs: []string{"yeti", "wisp", "wolf", "skeleton"}, MobName: []string{"winter-wight", "frost-hound", "rime-walker"}, MobGlyph: 'w', MobF: CWhite, Faction: "undead",
		Glyph: '*', Grow: 3, Per: 2, Cap: 0.35, Floor: TSnow, Hazard: TIce, Attr: TOU,
		Price:  map[string]float64{"food": 1.7, "cloth": 1.6, "medicine": 1.2},
		Rise:   "Snow has begun to fall near %s in the height of summer, and it does not melt.",
		Greet:  []string{"\"The frost came in a single night. The wells are ice to the bottom.\"", "\"Burn what you must. Better the furniture than the children.\""},
		Gossip: []string{"Rivers have frozen from source to sea; wolves walk into villages.", "The sun is up, they say, but it gives no warmth."},
		Weak:   "Hearth-fire kept in the relics' keeping can melt it from the inside.",
		Dread:  "Spring comes late every year now, but it comes.", Cause: "bitter cold"},
	"bomb": {ID: "bomb", Genres: []string{"atompunk", "dieselpunk"}, Names: []string{"The Bomb", "The Second Sun"}, Boss: []string{"the Pale Flame", "the Last Device"}, BossTpl: "magma",
		Mobs: []string{"magma", "salamander", "ashwraith"}, MobName: []string{"fallout horror", "cinder-beast", "glow-wraith"}, MobGlyph: 'E', MobF: COrange, Faction: "legends",
		Glyph: '☼', Grow: 3, Per: 2, Cap: 0.3, Floor: TAsh, Hazard: TLava, Attr: TOU,
		Price:  map[string]float64{"medicine": 1.6, "water": 1.6, "food": 1.4},
		Rise:   "A device was armed near %s and the countdown cannot be stopped from outside.",
		Greet:  []string{"\"They say it hums. The ground, I mean. Under the bunker.\"", "\"I keep my iodine and my prayers. Both low.\""},
		Gossip: []string{"Ministries issue orders that contradict each other every hour.", "The ash is already falling in the east."},
		Weak:   "The keys unlock the vault where the arming codes are kept; the relics hold the original schematics.",
		Dread:  "Geiger-clicks are the new birdsong in the east.", Cause: "fallout"},
	"dust": {ID: "dust", Genres: []string{"wasteland"}, Names: []string{"The Dust", "The Hungry Sands"}, Boss: []string{"the Dust Mother", "the Grave Wind"}, BossTpl: "sandworm",
		Mobs: []string{"sandworm", "scorpion", "ashwraith"}, MobName: []string{"dust-wraith", "sand-ravener", "grit-beast"}, MobGlyph: 'W', MobF: CYellow, Faction: "beasts",
		Glyph: '∙', Grow: 2, Per: 3, Cap: 0.4, Floor: TSand, Hazard: TQuicksand, Attr: TOU,
		Price:  map[string]float64{"water": 2.0, "food": 1.5, "medicine": 1.2},
		Rise:   "A wall of dust rose out of the ground near %s and is walking east.",
		Greet:  []string{"\"Wet your cloth. Cover your mouth. Pray there is a well.\"", "\"It eats the roofs first, then the floors.\""},
		Gossip: []string{"Whole caravans have gone into the dust and the tracks do not come out.", "Water is the only coin that matters now."},
		Weak:   "The relics hold the old water-rites; a spring called up at its heart drowns the dust.",
		Dread:  "The dunes have moved but the wells still answer.", Cause: "choking dust"},
	"sunder": {ID: "sunder", Genres: []string{"steampunk"}, Names: []string{"The Sundering", "The Great Unmeshing"}, Boss: []string{"the Prime Engine", "the Unmeshed Colossus"}, BossTpl: "crystalgolem",
		Mobs: []string{"crystalgolem", "shardmoth", "golem"}, MobName: []string{"runaway automaton", "shard-engine", "unmeshed thing"}, MobGlyph: 'G', MobF: CMagenta, Faction: "constructs",
		Glyph: '≠', Grow: 3, Per: 2, Cap: 0.3, Floor: TMachine, Hazard: TCrystal, Attr: INT,
		Price:  map[string]float64{"weapons": 1.4, "ore": 1.6, "cloth": 1.3},
		Rise:   "The great engine near %s has torn loose from its housing and the ground is coming apart around it.",
		Greet:  []string{"\"The gears keep running with nothing on them. You can hear it.\"", "\"My watch ticks backward now. So does the clock tower.\""},
		Gossip: []string{"Trains run to stations that are no longer there.", "The engineers' guild has locked its doors and is saying nothing."},
		Weak:   "The relics are the original drawings; with the keys they let the Engine be re-meshed.",
		Dread:  "Clocks disagree across the land, and no two towns keep the same hour.", Cause: "shifting ground"},
}

var egEndNames = map[string]string{"destroy": "Destroyed", "seal": "Sealed", "bargain": "Bargained", "ally": "Allied", "abandon": "Abandoned"}

var egEndText = map[string]string{
	"destroy": "You struck the last blow. The threat is broken, and the world exhales. The scars will take a generation to heal, and the prices of everything are still rising and falling in the wake of it, but people are coming home.",
	"seal":    "You turned the keys and spoke the old words with the relics in your hands. The threat sleeps behind a seal. Everyone knows it is not dead; everyone hopes the seal outlasts them.",
	"bargain": "You took what it offered. The gift is real, so is the cost, and the realms do not forgive a treaty with the enemy. The pact-lands still stand, quiet and dangerous, and a tithe is collected each season.",
	"ally":    "The realms that stood beside you spoke as one, and the threat, shown a better road, took it. It lives on as a guardian of the land it ruined, wary and at your side. Rebuilding has begun.",
	"abandon": "You turned away. The threat stopped spreading when it had eaten what it wanted, and what it wanted was a great deal. Refugees fill the roads; hostile lands remain, and everyone remembers who walked away.",
}
