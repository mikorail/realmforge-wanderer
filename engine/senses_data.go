package engine

// Noise and smell data tables (prefix ns). See senses.go.

type nsKw struct {
	K string
	V int
}

// nsTileNoise: noise made by one ordinary step on a tile (halved when sneaking).
var nsTileNoise = map[Terrain]int{
	TFloor: 2, TRoad: 2, TCobble: 3, TGrass: 1, TSand: 1, TDirt: 1, TSnow: 2, TIce: 3, TAsh: 1,
	TMoss: 0, TFungus: 1, TCarpet: 0, TCrystalFloor: 4, TRubble: 5, TBones: 5, TObsidian: 4,
	TSalt: 3, TSulfur: 3, TTallGrass: 3, TBush: 3, TDeadTree: 3, TReeds: 3, TBridge: 3,
	TShallow: 5, TDeep: 4, TKelp: 3, TCoral: 4, TFalls: 3, TGlacier: 3, TTree: 2, TCrop: 2,
}

var nsTileWhy = map[Terrain]string{
	TRubble: "gravel and rubble crunch", TBones: "bones crunch", TCrystalFloor: "crystal shards ring",
	TShallow: "water splashes", TDeep: "swimming splashes", TTallGrass: "dry leaves and stalks rustle",
	TBush: "branches rustle", TDeadTree: "dry leaves rustle", TReeds: "reeds rustle", TBridge: "planks creak",
	TCobble: "cobbles ring", TIce: "ice creaks", TSnow: "snow crunches", TRoad: "hard road underfoot",
	TObsidian: "glassy stone rings", TCoral: "coral crackles", TKelp: "water churns",
}

// Worn armour noise by name keyword (first match wins).
var nsArmourLoud = []nsKw{
	{"power", 3}, {"exo", 3}, {"plate", 3}, {"full harness", 3}, {"lamellar", 2}, {"scale", 2}, {"chain", 2},
	{"mail", 2}, {"ring", 1}, {"brigandine", 1}, {"splint", 2}, {"iron", 1}, {"steel", 1}, {"bronze", 1},
	{"helm", 1}, {"greave", 1}, {"gauntlet", 1}, {"pauldron", 1},
	{"silk", -1}, {"felt", -1}, {"padded", -1}, {"soft", -1}, {"cloth", -1}, {"shadow", -2}, {"silent", -2},
	{"muffled", -2}, {"cloak", -1}, {"leather", 0},
}

// Wielded weapon noise by name keyword (added to the base fight noise).
var nsWeaponLoud = []nsKw{
	{"greatsword", 3}, {"great", 3}, {"warhammer", 4}, {"hammer", 3}, {"maul", 4}, {"mace", 3}, {"flail", 3},
	{"morningstar", 3}, {"halberd", 2}, {"axe", 2}, {"club", 2}, {"staff", 1}, {"chainsaw", 6}, {"power", 3},
	{"dagger", -3}, {"knife", -3}, {"stiletto", -3}, {"dirk", -3}, {"shiv", -3}, {"garrote", -4},
	{"claw", -2}, {"whip", -1}, {"rapier", -2}, {"silent", -4}, {"muffled", -3},
}

// Ranged noise by name keyword, checked in order.
var nsShotLoud = []nsKw{
	{"suppress", 5}, {"silenc", 5}, {"blowgun", 2}, {"sling", 3}, {"crossbow", 4}, {"bow", 4},
	{"atlatl", 3}, {"dart", 2}, {"javelin", 3}, {"blaster", 10}, {"laser", 8}, {"plasma", 11}, {"arc", 9},
	{"cannon", 20}, {"blunderbuss", 18}, {"musket", 17}, {"rifle", 16}, {"shotgun", 18}, {"pistol", 14},
	{"revolver", 14}, {"gun", 15}, {"carbine", 15}, {"repeater", 15},
}

// Items that rattle in the pack.
var nsRattle = []nsKw{{"bell", 2}, {"rattle", 2}, {"chime", 1}, {"chain", 1}, {"cowbell", 3}, {"jangling", 2}, {"tin cup", 1}}

// Scent tables.
var nsCookedWords = []string{"cooked", "smoked", "dried", "jerky", "cured", "salted", "roast", "stew", "baked", "pickled", "fried"}
var nsMeatWords = []string{"meat", "flesh", "haunch", "carcass", "steak", "liver", "entrails", "viscera", "offal", "tripe", "organ", "marrow"}
var nsRotWords = []string{"rotten", "rotting", "spoiled", "tainted", "carrion", "putrid", "maggot", "decay"}
var nsPerfumeWords = []string{"perfume", "musk", "cologne", "rosewater", "scented", "attar", "balm", "civet"}
var nsHerbWords = []string{"herb", "sage", "mint", "lavender", "thyme", "rosemary", "garlic", "juniper", "pine needle", "wormwood", "sweetgrass", "cedar"}
var nsFurWords = []string{"fur", "pelt", "fleece", "wool", "hide"}
var nsSmokeWords = []string{"smoked", "pipe", "tobacco", "pipeweed", "torch"}
var nsContraWords = []string{"opium", "hashish", "poppy tar", "smuggled", "contraband", "absinthe", "black lotus", "poppy resin", "snuff", "spiceweed"}

// Creatures that follow a scent trail (name or template contains).
var nsHoundWords = []string{"hound", "dog", "wolf", "jackal", "hyena", "fox", "warg", "dire", "bloodhound", "tracker", "lion", "tiger", "panther", "bear", "wildcat", "cat"}

// Threshold labels.
const (
	nsNoiseMuted  = 4
	nsNoiseLoud   = 10
	nsScentStrong = 4
	nsScentReek   = 9
)

var nsWindNames = [8]string{"north", "northeast", "east", "southeast", "south", "southwest", "west", "northwest"}
