package engine

// Taming data (prefix tm): flavour lines and names. Picked by hash, never by g.R.

// tmPetNames are given to beasts at the moment they join you; the player can rename them.
var tmPetNames = []string{
	"Ash", "Bramble", "Cinder", "Dusk", "Ember", "Fern", "Gale", "Hazel", "Ivy", "Juniper", "Kestrel", "Lark", "Moss", "Nettle",
	"Onyx", "Pip", "Quill", "Rook", "Sorrel", "Thistle", "Umber", "Vesper", "Willow", "Yarrow", "Bracken", "Clover", "Dune",
	"Flint", "Gorse", "Heron", "Indigo", "Jasper", "Kite", "Linden", "Marrow", "Nimbus", "Orrin", "Pebble", "Rowan", "Saffron",
	"Tansy", "Ulmo", "Vale", "Wren", "Zephyr", "Barley", "Copper", "Dapple", "Echo", "Fable",
}

// tmEat is what a beast does when it takes an offering, by trust level reached.
var tmEatLines = []string{
	"%s sniffs the offering, then takes it from your open hand.",
	"%s edges closer, nostrils working, and eats.",
	"%s watches you the whole time it eats, but it does eat.",
	"%s takes the food gently and does not shy from your hand.",
	"%s lowers its head and accepts the gift.",
}

var tmSpookLines = []string{
	"%s startles at your hand and shies away.",
	"%s flinches from the offering and backs off.",
	"%s stamps and snorts; you moved too fast.",
}

var tmSnapLines = []string{
	"%s rounds on you, hackles up!",
	"%s lunges at the hand that holds the food!",
	"%s takes your clumsiness for a threat and attacks!",
}

var tmCalmLines = []string{
	"You speak low and slow. %s's ears turn toward you and its stance eases.",
	"You hum under your breath. %s stops fidgeting and watches you.",
	"You sit down, slow and small. %s forgets, for a while, to be afraid.",
}

var tmHumLines = []string{
	"%s noses at you, curious now.",
	"%s settles beside you and tolerates your company.",
	"%s lets you stand close without a fuss.",
}

// tmDietLine describes what a beast eats.
var tmDietLine = map[byte]string{
	'h': "It grazes and browses: plants, grain, fruit and herbs would please it.",
	'c': "It hunts flesh: meat, or the corpse of a kill, would please it.",
	'o': "It eats nearly anything: meat, fish, plants, grubs or a cooked dish.",
	'i': "It lives on insects: grubs and the like would please it.",
	'p': "It feeds on fish: a fresh fillet or fish would please it.",
	's': "It scavenges: carrion and meat would please it.",
}

// tmFishWords, tmMeatWords and tmGrubWords sort pack items into food classes by name.
var tmFishWords = []string{"fillet", "fish", " eel", "roe", "trout", "salmon", "tuna", "shrimp", "crab", "herring", "sardine", "carp", "kipper", "squid", "lobster", "prawn"}
var tmMeatWords = []string{"meat", "flesh", "jerky", "steak", "sausage", "ham", "haunch", "liver", "bacon", "venison", "mutton", "beef", "pork", "ribs", "chop", "roast", "cutlet"}
var tmGrubWords = []string{"grub", "larva", "beetle", "cricket", "insect", "termite", "locust", "maggot", "ant "}

// tmWildGenres sell lost-age beasts at the stable (recent extinctions and Pleistocene megafauna).
var tmWildGenres = map[string]bool{"fantasy": true, "mythic": true, "wasteland": true, "twilight": true, "biopunk": true}
