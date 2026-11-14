package engine

// Data for deadlines.go (prefix dl): kinds, texts, stage warnings and rival group names.

type dlKindDef struct {
	ID     string
	Weight int
	Goal   string // deliver | ransom | attend | buy | claim
	Title  string // %s = town
	Goalt  string // one-line goal, %s = town
	Item   string // item name for deliver kinds
	Stage  [3]string
	Won    string
	Lost   string
	Min    int // minimum days of window
}

var dlKinds = []dlKindDef{
	{ID: "plague", Weight: 3, Goal: "deliver", Title: "Plague in %s", Goalt: "Carry the cure from %s to %s before the sickness spreads.", Item: "phial of cure",
		Stage: [3]string{"The sickness in %s is spreading; the next town will fall ill soon.", "Pyres burn in %s. Without the cure the plague will not stop.", "The last hours: %s is closing its gates on the dying."},
		Won:   "The cure reached %s. The fever broke within the week.",
		Lost:  "No cure came to %s. The plague ran through the streets and out along the roads.", Min: 6},
	{ID: "siege", Weight: 3, Goal: "deliver", Title: "Siege of %s", Goalt: "Break through with relief stores drawn at %s and reach %s before the larders empty.", Item: "relief stores",
		Stage: [3]string{"The siege of %s tightens; rations are halved.", "Defenders of %s are eating their horses and bartering their boots.", "The gate of %s will open to the besiegers within hours."},
		Won:   "The relief convoy reached %s and the besiegers drew off.",
		Lost:  "%s opened its gates to the besiegers. The garrison is dead or in chains.", Min: 5},
	{ID: "hostage", Weight: 2, Goal: "ransom", Title: "Hostage held at %s", Goalt: "Pay the ransom for %s in %s before the execution.", Item: "",
		Stage: [3]string{"The captors of %s have named the execution day.", "The scaffold is built. %s is still waiting for coin.", "The hangman sharpens his tools; the hour for %s is near."},
		Won:   "The ransom was paid and %s walked free.",
		Lost:  "No ransom was paid. %s was put to death on the appointed day.", Min: 4},
	{ID: "caravan", Weight: 2, Goal: "buy", Title: "Caravan leaving %s", Goalt: "Reach %s and trade with the %s before it leaves.", Item: "",
		Stage: [3]string{"The caravan at %s is loading its wagons.", "The caravan masters of %s are counting the last bales.", "The caravan at %s will roll out within hours."},
		Won:   "You traded with the caravan of %s before it left.",
		Lost:  "The caravan of %s rolled out and took its goods with it.", Min: 3},
	{ID: "festival", Weight: 2, Goal: "attend", Title: "Festival at %s", Goalt: "Be in %s on the feast day.", Item: "",
		Stage: [3]string{"Garlands are going up in %s.", "The feast of %s is almost here; the inns are full.", "Drums begin in %s. The feast is at hand."},
		Won:   "You stood among the revellers of %s and were remembered for it.",
		Lost:  "The feast of %s came and went without you.", Min: 3},
	{ID: "flood", Weight: 2, Goal: "claim", Title: "Flood closing the pass near %s", Goalt: "Cross the pass above %s before the water rises.", Item: "",
		Stage: [3]string{"Rivers near %s are running brown.", "The ford near %s is waist deep and rising.", "The pass near %s will be underwater by nightfall."},
		Won:   "You crossed the pass near %s before the flood and found what the waters left behind.",
		Lost:  "The flood closed the pass near %s. Travellers are turning back.", Min: 4},
	{ID: "vault", Weight: 3, Goal: "claim", Title: "Vault opens near %s", Goalt: "Reach the vault near %s while it stands open, before the rival company.", Item: "",
		Stage: [3]string{"Astronomers near %s agree: the seals will open on the day.", "Strange lights gather above the vault near %s.", "The vault near %s begins to open."},
		Won:   "You reached the vault near %s first and carried out its treasure.",
		Lost:  "The vault near %s closed again, emptied.", Min: 6},
	{ID: "relic", Weight: 3, Goal: "claim", Title: "Lost relic near %s", Goalt: "Reach the barrow near %s and take the relic before the rival company.", Item: "",
		Stage: [3]string{"Word of a buried relic near %s has spread.", "Two parties are now digging toward the relic near %s.", "The rival company is within a day of the relic near %s."},
		Won:   "You took the relic near %s before anyone else.",
		Lost:  "A rival company carried off the relic near %s.", Min: 6},
}

// the omen that opens a vault, by genre
var dlOmens = map[string]string{
	"fantasy":   "a black eclipse",
	"mythic":    "a black eclipse",
	"cyberpunk": "a datastorm",
	"steampunk": "a pressure surge in the leylines",
	"solarpunk": "a solar flare",
	"biopunk":   "a spore bloom",
	"wasteland": "a rad storm",
	"twilight":  "the long dusk",
}

var dlGroupAdj = []string{"Ashen", "Iron", "Gilded", "Hollow", "Crimson", "Grey", "Silent", "Broken", "Red", "Black", "Sable", "Thorned"}
var dlGroupNoun = []string{"Company", "Hand", "Wolves", "Delvers", "Reavers", "Pilgrims", "Compact", "Lantern", "Banner", "Brotherhood", "Crows", "Wardens"}
var dlLeaderFirst = []string{"Voss", "Marren", "Kellan", "Irsa", "Tolvik", "Dessa", "Orrin", "Baelin", "Yara", "Corbin", "Sefa", "Hadrik"}
var dlLeaderLast = []string{"the Lame", "Two-Rope", "of the Seven", "Coldhand", "the Younger", "Brightblade", "Mudfoot", "the Patient", "Sixfingers", "Redcloak"}

var dlCaravanNames = []string{"salt caravan", "spice caravan", "silk caravan", "ironmongers' caravan", "pilgrim caravan", "wine caravan", "furriers' caravan"}
