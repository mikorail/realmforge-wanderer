package engine

// Data for games.go (prefix gm): game and instrument names per culture sphere and genre skin,
// hustler titles, arena narration. Index of the six table games:
// 0 trick-taking cards, 1 liar's dice, 2 hazard dice, 3 tile game, 4 siege board game, 5 shell game.

const gmGameCount = 6

var gmGameNames = map[string][gmGameCount]string{
	"sinic":      {"ye-zi leaf-cards", "cup-and-bones bluff", "tsung dice-dash", "bamboo tiles", "siege of the tiger", "thimble and pea"},
	"japonic":    {"hana-fuda petals", "chinchiro-style cup bluff", "sai-koro dice", "kaze tiles", "castle and ronin", "cup-and-bean"},
	"indic":      {"ganjifa cards", "kauri-cup bluff", "pasa dice", "jade tiles", "chaturanga skirmish", "cobra-basket pea"},
	"sea":        {"kartu leaf-cards", "shell-cup bluff", "cowrie dice", "coral tiles", "tiger and goats", "coconut-shell game"},
	"steppe":     {"felt-card tricks", "knucklebone bluff", "shagai bones", "felt tiles", "wolf and sheep", "cup-and-pebble"},
	"persian":    {"ganjifeh cards", "pomegranate-cup bluff", "nard dice", "mosaic tiles", "shatranj skirmish", "walnut-shell game"},
	"arabic":     {"tarneeb cards", "cup-and-pebble bluff", "zahr dice", "mosaic tiles", "tabula siege", "three-cups game"},
	"mande":      {"leather-card tricks", "cowrie-cup bluff", "oware-bone dice", "calabash tiles", "siege of the fort", "calabash and seed"},
	"yoruba":     {"opon cards", "ayo seed bluff", "cowrie-cast dice", "ivory tiles", "ayo siege", "calabash and seed"},
	"bantu":      {"bone-card tricks", "pebble-cup bluff", "knuckle dice", "bone tiles", "mancala siege", "cup-and-seed game"},
	"ethiopic":   {"gebeta cards", "seed-cup bluff", "gebeta dice", "clay tiles", "senet siege", "pot-and-bean game"},
	"nahua":      {"amate cards", "patolli bean bluff", "patolli dice", "obsidian tiles", "ullamaliztli siege", "gourd-and-bean game"},
	"maya":       {"bark-paper cards", "jade-cup bluff", "bean-and-jade dice", "jade tiles", "siege of the temple", "gourd-and-bean game"},
	"andean":     {"quipu cards", "llama-bone bluff", "taba dice", "stone tiles", "siege of the apu", "cup-and-pebble"},
	"turtle":     {"stick-and-hide cards", "plum-stone bluff", "plum-stone dice", "reed tiles", "siege of the lodge", "moccasin game"},
	"inuit":      {"ivory-card tricks", "seal-bone bluff", "ajaqaq bones", "ivory tiles", "siege of the floe", "bone-cup game"},
	"polynesian": {"tapa cards", "shell-cup bluff", "kone dice", "coral tiles", "konane siege", "coconut-shell game"},
	"norse":      {"rune-card tricks", "liar's bone-dice", "hazard bone-dice", "rune tiles", "hnefatafl raid", "cup-and-ring game"},
	"celtic":     {"ogham-card tricks", "cup-and-bones bluff", "knuckle-bone hazard", "ogham tiles", "brandubh raid", "thimble and pea"},
	"latin":      {"tarot tricks", "perudo cup bluff", "hazard dice", "domino tiles", "ludus latrunculi", "shell game"},
	"slavic":     {"durak card tricks", "knucklebone bluff", "kosti dice", "birch tiles", "siege of the kremlin", "walnut-shell game"},
	"eldritch":   {"hollow-card tricks", "liar's teeth", "lot-dice of the void", "rune-bone tiles", "siege of the nine", "the three veils"},
}

var gmGenreNames = map[string][gmGameCount]string{
	"cyberpunk": {"holo-poker", "ICE-breaker bluff", "crash-roll", "neon tiles", "grid-siege", "three-cup scam"},
	"wasteland": {"scrap-card tricks", "bottle-cap bluff", "bottle-cap dice", "tin tiles", "raid on the vault", "can-and-cap scam"},
	"atompunk":  {"rocket-card tricks", "chrome-cup bluff", "craps", "tube tiles", "rocket siege", "cup-and-ball"},
}

var gmGameFallback = [gmGameCount]string{"trick cards", "liar's dice", "hazard dice", "tile game", "siege game", "shell game"}

// gmNames: the six game names for a culture sphere in a genre. Cyberpunk, wasteland and atompunk
// skins win over the sphere; an unknown sphere falls back to the plain names.
func gmNames(genre, sphere string) [gmGameCount]string {
	if n, ok := gmGenreNames[genre]; ok {
		return n
	}
	if n, ok := gmGameNames[sphere]; ok {
		return n
	}
	return gmGameFallback
}

// instruments per sphere: string, wind, drum, bowed/horn
var gmInstNames = map[string][4]string{
	"sinic":      {"pipa", "dizi flute", "war drum", "erhu"},
	"japonic":    {"shamisen", "shakuhachi", "taiko", "biwa"},
	"indic":      {"sitar", "bansuri", "tabla", "sarangi"},
	"sea":        {"kecapi zither", "suling flute", "kendang drum", "rebab"},
	"steppe":     {"dombra", "tsuur flute", "frame drum", "morin khuur"},
	"persian":    {"setar", "ney", "daf", "kamancheh"},
	"arabic":     {"oud", "ney flute", "darbuka", "rebab"},
	"mande":      {"kora", "bamboo flute", "djembe", "ngoni"},
	"yoruba":     {"kora", "wooden flute", "talking drum", "goje fiddle"},
	"bantu":      {"mbira", "reed flute", "log drum", "musical bow"},
	"ethiopic":   {"krar", "washint flute", "kebero drum", "masenqo"},
	"nahua":      {"huehuetl-guitar", "clay flute", "huehuetl drum", "conch horn"},
	"maya":       {"turtle-shell lyre", "clay whistle", "tunkul drum", "conch horn"},
	"andean":     {"charango", "quena", "bombo drum", "panpipes"},
	"turtle":     {"cedar flute", "bone whistle", "hand drum", "rattle-fiddle"},
	"inuit":      {"throat-lyre", "bone flute", "frame drum", "walrus-hide fiddle"},
	"polynesian": {"ukulele", "nose flute", "pahu drum", "conch horn"},
	"norse":      {"tagelharpa", "bone flute", "skin drum", "lur horn"},
	"celtic":     {"harp", "tin whistle", "bodhran", "fiddle"},
	"latin":      {"lute", "flute", "tambour", "fiddle"},
	"slavic":     {"gusli", "svirel flute", "tambourine", "gudok fiddle"},
	"eldritch":   {"bone harp", "marrow flute", "hollow drum", "hunting horn"},
}

var gmInstGenre = map[string][4]string{
	"cyberpunk": {"synth-deck", "vocoder", "pad-drum", "scratch-fiddle"},
	"wasteland": {"scrap guitar", "bone whistle", "oil-drum", "wire fiddle"},
	"atompunk":  {"transistor guitar", "tube clarinet", "bongos", "theremin"},
	"solarpunk": {"glass harp", "reed pipes", "seed-drum", "bow-chime"},
}

var gmInstFallback = [4]string{"lute", "flute", "drum", "fiddle"}

func gmInstSet(genre, sphere string) [4]string {
	if n, ok := gmInstGenre[genre]; ok {
		return n
	}
	if n, ok := gmInstNames[sphere]; ok {
		return n
	}
	return gmInstFallback
}

var gmQualityName = []string{"", "battered", "plain", "fine", "master's", "legendary"}

var gmDiceName = map[string]string{
	"cyberpunk": "loaded holo-dice", "wasteland": "weighted bottle-caps", "atompunk": "shaved dice",
}

var gmHustlerTitle = map[string][]string{
	"fantasy":    {"card sharp", "dice hustler", "tavern gambler"},
	"mythic":     {"fate-gambler", "oracle's bettor", "dice-priest"},
	"steampunk":  {"riverboat gambler", "card sharp", "faro dealer"},
	"dieselpunk": {"racket dealer", "card hustler", "dice man"},
	"atompunk":   {"casino shark", "craps hustler", "card shark"},
	"cyberpunk":  {"chip runner", "holo-dealer", "street hustler"},
	"biopunk":    {"spore-dealer", "bone-dice hustler", "card sharp"},
	"solarpunk":  {"token gambler", "seed-dice hustler", "game keeper"},
	"wasteland":  {"cap hustler", "scrap gambler", "dice runner"},
	"twilight":   {"blood-gambler", "pale dealer", "night card-sharp"},
}

var gmHouseName = map[string]string{
	"fantasy": "dicing hall", "mythic": "fate-house", "steampunk": "gaming saloon", "dieselpunk": "back-room casino",
	"atompunk": "casino lounge", "cyberpunk": "holo-den", "biopunk": "bone-dice den", "solarpunk": "games commons",
	"wasteland": "cap den", "twilight": "gloam-table room",
}

// spheres where turning down a wager is an insult
var gmProudSpheres = map[string]bool{"norse": true, "arabic": true, "celtic": true, "steppe": true, "persian": true, "indic": true, "sinic": true, "slavic": true, "nahua": true, "mande": true}

var gmInsultLine = map[string]string{
	"norse": "A man who will not wager is a man with no luck to lose, and no honour to stake.", "arabic": "To refuse a game offered over tea is to refuse the host.",
	"celtic": "\"Too good for a throw with a stranger, are we?\"", "steppe": "\"On the steppe a man who will not wager is a man who will not ride.\"",
	"persian": "\"You decline? A pity. The table remembers.\"", "indic": "\"You spurn the game? Then you spurn the one who offered it.\"",
	"sinic": "Declining is not rude; declining after sitting down is.", "slavic": "\"You sat and now you rise? Insulting.\"",
	"nahua": "\"The gods watch those who will not play.\"", "mande": "\"We offered you a place and you spit on it.\"",
}

// arena narration
var gmBlowHit = map[string][]string{
	"blade": {"%s opens %s's guard with a quick cut", "%s drives a thrust into %s's side", "%s hammers %s back with a pommel strike", "%s slashes across %s's forearm"},
	"guard": {"%s smashes a shield-rim into %s", "%s steps in and cuts %s down the shoulder", "%s drives %s back with a heavy blow", "%s bashes %s across the helm"},
	"sim":   {"%s feints and cuts %s low", "%s turns a parry into a vicious riposte on %s", "%s catches %s with a veteran's trick", "%s cracks %s across the ribs"},
	"beast": {"%s rakes %s open", "%s lunges and bites into %s", "%s slams %s into the sand", "%s tears at %s's flank"},
}
var gmBlowMiss = []string{"%s swings at air", "%s is read like a book and misses", "%s's blow glances off a raised arm", "%s stumbles on the blood-wet sand"}
var gmFinishLine = []string{"%s goes down and does not rise.", "%s is dragged from the sand by the bearers.", "%s yields to the crowd's roar.", "%s falls, and the pit goes quiet."}
var gmCrowdLine = []string{"The crowd bays.", "Coins ring on the rail.", "A drummer beats time.", "The bookies shout fresh odds."}
