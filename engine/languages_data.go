package engine

// Data for languages.go: the core vocabulary (most useful words first: this is also the order in
// which you learn them), sound shifts between related tongues, grammar choices, gadget names.

var lxConcepts = []string{
	// courtesy and basics
	"hello", "farewell", "thanks", "please", "yes", "no", "sorry", "welcome", "friend", "enemy",
	"help", "stop", "go", "come", "give", "take", "buy", "sell", "water", "food",
	"bread", "meat", "fire", "house", "road", "river", "mountain", "sea", "sun", "moon",
	"night", "day", "gold", "coin", "sword", "shield", "king", "god", "danger", "one",
	"two", "three", "four", "five", "six", "seven", "eight", "nine", "ten", "big",
	"small", "hot", "cold", "good", "bad", "kill", "love", "death", "life", "town",
	"gate", "inn", "temple", "market", "guard", "thief",
	// people
	"man", "woman", "child", "mother", "father", "name", "stranger", "lord", "law", "peace",
	"war", "truth", "lie", "honor", "shame", "secret", "luck", "fear", "hope", "anger",
	"family", "guest", "gift", "promise", "oath", "debt", "price", "cheap", "dear", "healer",
	"priest", "scholar", "merchant", "smith", "soldier", "farmer", "hunter", "beggar", "servant", "elder",
	// verbs
	"eat", "drink", "sleep", "walk", "run", "fight", "see", "hear", "speak", "know",
	"want", "need", "find", "lose", "open", "close", "wait", "stay", "leave", "return",
	"carry", "hide", "pay", "steal", "heal", "teach", "learn", "read", "write", "pray",
	// questions and places in time and space
	"where", "who", "what", "why", "how much", "here", "there", "now", "never", "always",
	"east", "west", "north", "south", "left", "right", "up", "down", "near", "far",
	"new", "old", "young", "strong", "weak", "rich", "poor", "fast", "slow", "dark",
	"light", "red", "blue", "green", "white", "black",
	// the world
	"wind", "rain", "snow", "storm", "tree", "forest", "field", "stone", "iron", "silver",
	"bone", "blood", "horse", "dog", "wolf", "bird", "fish", "bear", "snake", "ship",
	"boat", "bridge", "wall", "tower", "castle", "door", "key", "rope", "bow", "arrow",
	"spear", "armor", "knife", "axe", "book", "letter", "map", "song", "story", "dream",
	"ghost", "demon", "spirit", "magic", "poison", "medicine", "home", "village", "city", "land",
	"world", "star", "sky", "earth", "island", "desert", "cave", "well", "fruit", "wine",
	"salt", "milk", "egg", "honey",
	// the strange: genre props
	"data", "machine", "vat", "engine", "signal", "wire", "steam", "crystal", "spell", "relic",
	"ruin", "seed", "hive", "wheel", "glass", "oil",
}

// lxShifts are regular sound changes; each tongue of a family adopts three of them.
var lxShifts = [][2]string{
	{"t", "d"}, {"k", "g"}, {"p", "b"}, {"s", "h"}, {"l", "r"}, {"m", "n"}, {"v", "w"}, {"f", "p"},
	{"d", "z"}, {"b", "v"}, {"c", "k"}, {"a", "o"}, {"e", "i"}, {"o", "u"}, {"i", "e"}, {"u", "a"},
	{"th", "t"}, {"sh", "s"}, {"ch", "k"}, {"r", "l"}, {"g", "k"}, {"n", "m"},
}

var lxOrders = []string{"SVO", "SOV", "VSO", "OSV", "VOS", "SOV"}

var lxOrderText = map[string]string{
	"SVO": "subject, verb, object",
	"SOV": "subject, object, verb",
	"VSO": "verb first, then subject, object",
	"OSV": "object first, then subject, verb",
	"VOS": "verb first, then object, subject",
}

// lxPluralEnds: candidate plural suffixes; the tongue takes whichever fits its sounds.
var lxPluralEnds = []string{"s", "n", "i", "en", "oi", "ar", "im", "k", "a", "u"}

// lxTranslators: the carried gadget, per genre.
var lxTranslators = map[string]string{
	"mythic":     "linguist's prism",
	"steampunk":  "clockwork translator",
	"dieselpunk": "field translator",
	"atompunk":   "universal translator",
	"cyberpunk":  "phrasebook app",
	"biopunk":    "babel fish",
	"solarpunk":  "speaking-seed",
	"wasteland":  "scavenged translator",
	"twilight":   "linguist's prism",
}

var lxPrimerNames = map[string]string{
	"cyberpunk": "language pack",
	"biopunk":   "language culture",
	"atompunk":  "phrasebook reel",
}

// lxProudWords: culture values that make a clumsy greeting an insult.
var lxProudWords = []string{"honor", "honour", "pride", "proud", "dignity", "face", "respect", "ancest", "tradition", "oath", "glory", "valor", "duty", "purity"}
