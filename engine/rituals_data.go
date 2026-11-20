package engine

// Fallback texts for rituals.go (prefix rt): used when a world file has no culture pack or faith lore.
// Placeholders are the culturepack ones ({name} {title} {realm} {town} {culture} {faith} {deity} {time} {kin} {item}).

// rtFallbackRites: generic rite-of-passage steps by id (same ids as Realmforge's Rite.ID, with "naming" and
// "comingofage" as the ceremonies' birth and age kinds).
var rtFallbackRites = map[string]*Rite{
	"wedding": {ID: "wedding", Name: "The wedding", Roles: []string{"officiant", "witness"}, Gifts: []string{"bread", "wine", "cloth"}, Dress: "festive clothes",
		Steps: []string{
			"The families gather in {town} and the couple are brought before {deity}'s altar.",
			"The officiant asks each in turn whether they come freely, and each answers aloud for the witnesses.",
			"The couple exchange tokens and a cup is shared between them.",
			"The guests cheer, and the long table is opened.",
		}},
	"funeral": {ID: "funeral", Name: "The funeral", Roles: []string{"officiant", "mourner"}, Gifts: []string{"flowers", "incense"}, Dress: "plain dark clothes",
		Steps: []string{
			"The dead lie in state while the kin keep the vigil, and no one speaks above a murmur.",
			"The officiant names the dead and the deeds they leave behind.",
			"Each mourner lays a small gift beside the body.",
			"The procession carries the dead out, and the wake is held after.",
		}},
	"naming": {ID: "naming", Name: "The naming", Roles: []string{"officiant", "godparent"}, Gifts: []string{"cloth", "coin"}, Dress: "clean clothes",
		Steps: []string{
			"The child is carried in and shown to the household.",
			"The officiant speaks the name three times so that {deity} and the ancestors may hear it.",
			"The guests give their blessings, and the child is passed from arm to arm.",
		}},
	"comingofage": {ID: "comingofage", Name: "The coming of age", Roles: []string{"elder", "sponsor"}, Gifts: []string{"tools", "coin"}, Dress: "festive clothes",
		Steps: []string{
			"The youth stands before the elders of {town}, no longer counted a child.",
			"The elders ask what the youth has learned, and the youth answers.",
			"The youth is given the garb and duties of a grown person.",
			"The kin crowd round with gifts and good wishes.",
		}},
	"betrothal": {ID: "betrothal", Name: "The betrothal", Roles: []string{"go-between", "witness"}, Gifts: []string{"sweets", "cloth"}, Dress: "festive clothes",
		Steps: []string{
			"The two families meet to speak of the match and to weigh the gifts.",
			"The terms are said aloud and agreed.",
			"A token is exchanged, and the betrothal is witnessed.",
		}},
	"oath": {ID: "oath", Name: "The oath-taking", Roles: []string{"oath-taker", "witness"}, Gifts: []string{"salt"}, Dress: "clean clothes",
		Steps: []string{
			"The oath-taker stands before witnesses with an open hand.",
			"The oath is spoken in full, and {deity} is called to hear it.",
			"A mark is made, and a witness answers that it was heard.",
		}},
	"initiation": {ID: "initiation", Name: "The initiation", Roles: []string{"elder", "initiate"}, Gifts: []string{"coin"}, Dress: "plain clothes",
		Steps: []string{
			"The initiate is led apart and told what is expected.",
			"A trial is set, small but real, and the initiate must bear it.",
			"The initiate is given a name, a sign, or a place among the members.",
		}},
	"birth": {ID: "birth", Name: "The birth rite", Roles: []string{"midwife", "elder"}, Gifts: []string{"cloth", "food"}, Dress: "clean clothes",
		Steps: []string{
			"The household waits while the midwife works, and the door stays closed.",
			"At the first cry, the news is carried out to the neighbours.",
			"A small gift is left at the threshold for the newcomer.",
		}},
}

// rtRiteAliases maps the ceremony kinds and loose ids to Realmforge rite ids.
var rtRiteAliases = map[string]string{
	"age": "comingofage", "coming of age": "comingofage", "namings": "naming", "birth": "birth",
	"bethrothal": "betrothal",
}

// rtFallbackPrayers: used when a faith has no pack prayers.
var rtFallbackPrayers = []string{
	"{deity}, keep us in the hours of light and in the hours of dark.",
	"{deity}, you who see all roads, watch over the one who walks them.",
	"May {deity} grant us bread, shelter, and a steady heart.",
	"{deity}, hear us. We are small, and we are grateful to be heard.",
}

var rtFallbackSins = []string{"cruelty to the helpless", "a lie told for gain", "greed that left others hungry", "cowardice when courage was owed", "forgetting the gods"}

var rtFallbackFestSteps = []string{
	"The townsfolk gather at the heart of {town} and the keepers of the day call everyone together.",
	"The old words of the day are said, and each household brings a share to the common table.",
	"The people join hands, or voices, or both, and the rite is carried round the square.",
}

// rtGreetFallback: clergy greetings when the faith has no clergy voice.
var rtGreetFaithful = []string{"Peace of {deity} be on you, faithful one.", "Welcome back to the house of {deity}.", "{deity} keeps you, I see. Come in."}
var rtGreetStranger = []string{"Stranger, you stand in the house of {deity}. Be welcome, and be quiet.", "A traveller. {deity} sees all who enter, you know.", "Whatever brings you, {deity}'s door is open."}

// rtHolyHours: hours (24h clock) of daily liturgy.
var rtHolyHours = []struct {
	From, To int
	Name     string
}{{5, 7, "dawn prayer"}, {11, 13, "noon prayer"}, {17, 19, "evening prayer"}, {21, 22, "night vigil"}}

// rtCustomKind: keyword in the festival's custom (a contest or game) to the check it uses.
var rtCustomKind = []struct {
	Words []string
	Kind  string
}{
	{[]string{"race", "run", "dance", "leap", "jump", "climb", "wrest", "tumbl"}, "acrobatics"},
	{[]string{"riddle", "poem", "verse", "song", "sing", "tale", "story", "recit", "lore", "chess", "game"}, "lore"},
	{[]string{"arrow", "archer", "shoot", "throw", "target", "dart"}, "perception"},
	{[]string{"drink", "feast", "eat", "carry", "lift", "tug", "haul"}, "endurance"},
	{[]string{"bargain", "haggl", "debate", "speech", "toast"}, "persuasion"},
}

// rtTabooWords: keyword in a feast's Taboo to what the engine can detect.
var rtTabooWords = []struct {
	Words []string
	What  string
}{
	{[]string{"weapon", "blade", "sword", "steel", "iron", "arms", "armed", "spear", "knife", "bow"}, "weapon"},
	{[]string{"meat", "flesh", "blood", "slaughter"}, "meat"},
	{[]string{"wine", "alcohol", "liquor", "beer", "mead", "drunk"}, "drink"},
	{[]string{"black", "red", "white", "yellow", "green", "blue", "purple", "gold"}, "colour"},
}
