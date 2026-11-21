package engine

// Data for routines.go (prefix rn): job keywords, site nouns and the look-text phrases.

// rnJobKeys maps words found in a person's title, role or job to a routine job. First match wins.
var rnJobKeys = []struct {
	job   string
	words []string
}{
	{"patrol", []string{"guard", "sergeant", "sentry", "warden", "watchman", "watch"}},
	{"smith", []string{"smith", "armorer", "farrier", "machinist", "tinker", "metalworker", "forger"}},
	{"field", []string{"farmer", "herder", "shepherd", "gardener", "planter", "harvester", "peasant", "labourer", "laborer", "ploughman"}},
	{"temple", []string{"priest", "monk", "nun", "pilgrim", "acolyte", "cleric", "sexton", "verger", "shaman", "imam", "chaplain", "preacher"}},
	{"academy", []string{"student", "scribe", "scholar", "astronomer", "clerk", "teacher", "tutor", "sage", "librarian", "alchemist", "physician"}},
	{"tavern", []string{"drunk", "cook", "brewer", "barmaid", "tapster", "innkeeper"}},
	{"mine", []string{"miner", "quarryman", "digger"}},
	{"water", []string{"fisher", "boatman", "sailor", "ferryman", "dockhand", "stevedore"}},
	{"wood", []string{"woodcutter", "carpenter", "logger", "lumberjack", "cooper"}},
	{"play", []string{"child", "urchin", "orphan"}},
	{"market", []string{"weaver", "potter", "baker", "hawker", "vendor", "peddler", "pedlar", "trader", "grocer", "butcher", "fishmonger", "glassblower", "painter", "carver", "jeweller", "jeweler", "dyer", "tanner", "miller", "merchant", "rag-picker"}},
}

// rnWorkSites: where a job works, in order of preference (site names of rnZone).
var rnWorkSites = map[string][]string{
	"smith":   {"forge", "stall"},
	"field":   {"crop"},
	"temple":  {"altar"},
	"academy": {"shelf", "square"},
	"tavern":  {"tavern"},
	"mine":    {"ore"},
	"water":   {"water"},
	"wood":    {"tree"},
	"play":    {"square"},
	"market":  {"stall", "square"},
	"idle":    {"stall", "square"},
}

// rnRoles are the roles routines apply to. Everything else (trainers, installers, ferrymen, rulers, event actors) keeps its own AI.
var rnRoles = map[string]bool{
	"villager": true, "elder": true, "priest": true, "notable": true, "guard": true, "sergeant": true, "scholar": true,
	"merchant": true, "smith": true, "apothecary": true, "tailor": true, "bookseller": true, "exotic": true, "homeless": true,
}

// rnShopRoles keep a stall or shop staffed while it is open.
var rnShopRoles = map[string]bool{"merchant": true, "smith": true, "apothecary": true, "tailor": true, "bookseller": true, "exotic": true}

// rnSkipTitles are titles of villagers that belong to a place or a scene of their own.
var rnSkipTitles = []string{"gladiator", "fence", "pickpocket", "master of", "drillmaster", "arena"}

// rnSkipFlagPrefixes: flags that mean another system drives this actor.
var rnSkipFlagPrefixes = []string{"fw", "cj", "uw", "hz", "sw", "lw", "tr", "cq", "nq", "mx", "ci", "cb", "prisoner", "captive", "escort"}

// rnNoun: what a site is called in "on their way to ...".
var rnNoun = map[string]string{
	"well": "the well", "stall": "the market", "forge": "the forge", "altar": "the temple", "crop": "the fields",
	"shelf": "the academy", "ore": "the diggings", "water": "the water's edge", "tree": "the trees", "square": "the square",
	"tavern": "the tavern", "bed": "bed", "home": "home", "post": "their post", "round": "their round",
}

// rnAt: what they do once they are there, keyed "act/site"; several phrases are picked by hash.
var rnAt = map[string][]string{
	"well/well":     {"washing at the well", "drawing water at the well", "drinking from the well"},
	"work/forge":    {"working at the forge", "hammering at the forge"},
	"work/stall":    {"selling at the market", "haggling at a stall"},
	"work/crop":     {"working the fields", "tending the crops"},
	"work/altar":    {"tending the shrine", "praying at the altar"},
	"work/shelf":    {"reading among the shelves", "copying at the academy"},
	"work/ore":      {"working the ore seam", "chipping at the rock"},
	"work/water":    {"mending nets at the water's edge", "fishing from the bank"},
	"work/tree":     {"gathering wood", "sizing up the timber"},
	"work/square":   {"idling on the square", "playing on the square", "reading on the square"},
	"work/tavern":   {"working the tavern floor", "nursing a cup"},
	"eat/tavern":    {"eating at the tavern", "having a midday pot at the tavern"},
	"eat/stall":     {"eating a bite at the market", "chewing bread at the market"},
	"eat/square":    {"eating on the square", "having a bite on the square"},
	"errand/stall":  {"running an errand at the market", "looking over the stalls"},
	"errand/well":   {"fetching water at the well", "waiting their turn at the well"},
	"errand/square": {"crossing the square", "pausing on the square"},
	"dusk/tavern":   {"drinking at the tavern", "unwinding at the tavern"},
	"gather/altar":  {"praying with the congregation", "keeping the holy day at the temple"},
	"gather/square": {"joining the gathering", "keeping the feast on the square"},
	"shelter/roof":  {"sheltering from the rain", "waiting out the rain under a roof"},
	"shop/stall":    {"tending a stall", "minding the stall"},
	"shop/forge":    {"working at the forge", "hammering at the forge"},
	"shop/shop":     {"minding their shop", "keeping the shop"},
	"patrol/post":   {"standing watch", "keeping watch"},
	"patrol/round":  {"patrolling the streets", "walking their round"},
	"rest/home":     {"resting at home"},
}

// rnStop: the line when two neighbours stop to talk within sight.
var rnStop = []string{
	"%s and %s stop to talk.",
	"%s and %s stand a moment, talking.",
	"%s stops to speak with %s.",
}
