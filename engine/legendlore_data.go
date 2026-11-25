package engine

// legendlore_data.go (prefix lb): tables for legendlore.go. Text is built from templates and world data.

// lbEvKinds: the chronicle event kinds a legend can hang on, and the kinds of thing each tends to leave behind.
var lbEvClasses = map[string][]string{
	"war":          {"weapon", "armor", "weapon"},
	"capture":      {"weapon", "armor", "crown"},
	"peace":        {"crown", "weapon", "relic"},
	"fall":         {"crown", "weapon", "relic"},
	"coup":         {"crown", "weapon"},
	"death":        {"weapon", "crown", "relic"},
	"monster":      {"weapon", "armor", "weapon"},
	"disaster":     {"relic", "armor", "tome"},
	"eco_disaster": {"relic", "armor"},
	"faith":        {"relic", "tome", "relic"},
	"god":          {"relic", "tome"},
	"wonder":       {"tome", "relic"},
}

var lbEvPhrase = map[string]string{
	"war": "the wars of that age", "capture": "the sacking of a capital", "peace": "the signing of a hard peace", "fall": "the fall of a realm",
	"coup": "a night of knives", "death": "the end of a reign", "monster": "the waking of a great beast", "disaster": "a great calamity",
	"eco_disaster": "a ruinous accident", "faith": "a change of faith", "god": "a god's gift", "wonder": "a wonder won and lost",
}

var lbFate = map[string]string{
	"war": "fell in battle", "capture": "was cut down in the sack", "peace": "was forced out of office", "fall": "fled the ruin of the realm",
	"coup": "was thrown down by conspirators", "death": "died", "monster": "was taken by the beast", "disaster": "was lost in the calamity",
	"eco_disaster": "was lost when the works failed", "faith": "was struck down for the new faith", "god": "was not seen again",
	"wonder": "was lost when the wonder fell",
}

// the kind of foe whose slaying wakes a legend (tags from gearlore.go ggTags)
var lbKillByEv = map[string][]string{
	"war": {"man", "ruler"}, "capture": {"man"}, "peace": {"man"}, "fall": {"undead"}, "coup": {"man", "ruler"}, "death": {"undead"},
	"monster": {"beast", "dragon"}, "disaster": {"beast", "fire"}, "eco_disaster": {"fire", "beast"}, "faith": {"undead"}, "god": {"undead", "legend"},
	"wonder": {"beast", "undead"},
}

var lbKillWord = map[string]string{"man": "men", "ruler": "rulers and captains", "undead": "the restless dead", "beast": "beasts", "dragon": "dragons and wyrms",
	"fire": "burning things", "ice": "creatures of ice", "legend": "legends"}

// guardian trick by the kind of event it came from (ids of drGims)
var lbGimByEv = map[string]string{
	"war": "adds", "capture": "adds", "peace": "weakness", "fall": "safezone", "coup": "mirror", "death": "braziers",
	"monster": "pillars", "disaster": "lava", "eco_disaster": "ice", "faith": "braziers", "god": "mirror", "wonder": "pillars",
}

var lbSitePrefer = map[string][]string{
	"war": {"barrow", "catacomb"}, "capture": {"catacomb", "barrow"}, "peace": {"vault", "catacomb"}, "fall": {"catacomb", "barrow"},
	"coup": {"catacomb", "vault"}, "death": {"barrow", "catacomb"}, "monster": {"chasm", "mine", "abyss"}, "disaster": {"sunken", "chasm"},
	"eco_disaster": {"vault", "arcane"}, "faith": {"catacomb", "arcane"}, "god": {"arcane", "abyss"}, "wonder": {"arcane", "vault"},
}

var lbNamePre = []string{"Grave", "Dawn", "Ash", "Oath", "Star", "Crown", "Wolf", "Thorn", "Dusk", "Gale", "Iron", "Sorrow", "Ember", "Frost", "Bright", "Hollow", "Widow", "Rook", "Salt", "Hush", "Kestrel", "Marrow", "Bell"}
var lbNameSuf = []string{"song", "splitter", "bane", "fall", "keeper", "reaver", "tide", "wake", "ward", "bloom", "brand", "sworn", "cry", "mourn", "guard", "light"}

var lbNounRelic = []string{"reliquary", "finger-bone", "sigil", "phial", "icon", "hand-lamp"}
var lbNounTome = []string{"codex", "ledger of names", "psalter", "atlas"}

var lbBossEpithet = []string{"the Sleepless", "the Unburied", "the Unquiet", "Whose Oath Held", "the Faithful Dead", "the Last Bearer"}

var lbMakerTitle = []string{"the Smith", "the Maker", "of the Low Forge", "the Elder Artificer"}

// lbBossBack: backstory lines for the guardian, by event kind.
var lbBossBack = map[string]string{
	"war":          "%s carried %s through the wars and was buried with it in the old hall. They have not stopped fighting.",
	"capture":      "%s held the last stair of a sacked city with %s and was sealed in below. The stair is still held.",
	"peace":        "%s signed a hard peace with %s on the table and was never allowed to leave the room.",
	"fall":         "%s fled the fall of the realm with %s and went to ground here. The realm is gone; the flight goes on.",
	"coup":         "%s was thrown down by those they trusted, and %s was thrown down with them. They wait for the knock at the door.",
	"death":        "%s was laid here with %s, and the rites went wrong. Someone has to keep it.",
	"monster":      "%s hunted the great beast with %s, and the lair took them both. What stands guard now is not entirely either.",
	"disaster":     "%s kept %s through the calamity and would not leave the ruin it made.",
	"eco_disaster": "%s tended the works and %s when they failed, and the failure never ended for them.",
	"faith":        "%s bore %s for a faith that was crushed, and keeps the watch for a dawn that did not come.",
	"god":          "%s was given %s by a god that then fell silent, and they have been waiting for the rest of the message.",
	"wonder":       "%s guarded %s when the wonder fell, and is still guarding what is left.",
}

var lbLoreNames = []string{"ballad sheet", "scorched diary page", "pilgrim's tally", "surveyor's note", "dead bearer's letter"}

// lbRumour: how a lead is told (%s item, %s person, %s place/direction)
var lbRumourLines = []string{
	"They say %s, the arm of %s, was last carried down into %s. %s.",
	"An old song names %s, that %s once bore. The singer swears it lies under %s. %s.",
	"The tale goes that %s went into the earth with %s at %s. %s.",
}

var lbCurseDefs = map[string]string{
	"thirst":   "Its bearer is always thirsty.",
	"blood":    "It takes a little blood from whoever carries it.",
	"unquiet": "Its last bearer's shade is not at rest; it comes looking.",
}

var lbCurseIDs = []string{"thirst", "blood", "unquiet"}

// lbFx: one power an arm may carry. For: w weapon, a armor and crowns, r relic and tome, any: wa r.
type lbFx struct {
	ID    string
	Name  string
	Desc  string
	For   string
	GG    int // gearlore heirloom effect (ggfx), 0 none
	Apply func(it *Item)
}

func lbEx(it *Item) map[string]int {
	if it.Extra == nil {
		it.Extra = map[string]int{}
	}
	return it.Extra
}

func lbAttr(a, v int) func(it *Item) { return func(it *Item) { addBonus(it, a, v) } }

func lbPass(id string) func(it *Item) { return func(it *Item) { lbEx(it)["lbp:"+id] = 1 } }

func lbCat(c string) func(it *Item) { return func(it *Item) { it.Cat = c } }

var lbFxs = []lbFx{
	{"flame", "a flaming edge", "Deals fire damage.", "w", 0, func(it *Item) { it.Elem = "fire" }},
	{"frost", "a frost bite", "Deals frost damage.", "w", 0, func(it *Item) { it.Elem = "frost" }},
	{"storm", "a storm in the steel", "Deals shock damage.", "w", 0, func(it *Item) { it.Elem = "shock" }},
	{"edge", "a keener edge", "Hits harder.", "w", 0, func(it *Item) { it.DS += 2; it.PV++ }},
	{"holy", "a ward against the dead", "Bonus damage to the restless dead and horrors.", "w", 0, lbCat("celestial")},
	{"gravebane", "a sacred fire", "Bonus damage to the restless dead and horrors.", "w", 0, lbCat("biblical")},
	{"beastbane", "an old hunger for beasts", "Bonus damage to beasts and vermin.", "w", 0, lbCat("eldritch")},
	{"machinebane", "a breaker of works", "Bonus damage to constructs.", "w", 0, lbCat("techno")},
	{"bleed", "a rending edge", "Wounds bleed.", "w", 5, nil},
	{"slayer", "a slayer's edge", "Extra damage to bosses and legends.", "w", 6, nil},
	{"thirst", "a thirst for blood", "Heals you as it kills.", "w", 4, nil},
	{"keen", "a keen strike", "Now and then, an extra cut.", "w", 8, nil},
	{"crowncrack", "a crown-cracking blow", "Stuns the unwary.", "w", 7, nil},
	{"mend", "a mending touch", "Heals slowly.", "a", 11, nil},
	{"cleanse", "a cleansing ward", "Burns off poison, fire and bleeding.", "a", 12, nil},
	{"heart", "a stout heart", "Steadies you.", "a", 13, nil},
	{"bulwark", "a bulwark", "Heavier armour.", "a", 0, func(it *Item) { it.AV += 2; it.DV++ }},
	{"might", "strength of the old bearers", "+2 Strength.", "wa", 0, lbAttr(STR, 2)},
	{"grace", "quickness", "+2 Agility.", "wa", 0, lbAttr(AGI, 2)},
	{"vigor", "a deep vigor", "+2 Toughness.", "waar", 0, lbAttr(TOU, 2)},
	{"mind", "sharpened wits", "+2 Intelligence.", "war", 0, lbAttr(INT, 2)},
	{"will", "an iron will", "+2 Willpower.", "war", 0, lbAttr(WIL, 2)},
	{"voice", "a commanding presence", "+2 Ego.", "war", 0, lbAttr(EGO, 2)},
	{"wind", "a second wind", "When near death, heals a good part of your health (about every three hours).", "war", 0, lbPass("wind")},
	{"reaper", "a reaper's mercy", "Heals a little on every kill.", "war", 0, lbPass("reaper")},
	{"fortune", "an old fortune", "Kills pay a little extra coin.", "war", 0, lbPass("fortune")},
	{"scholar", "a long memory", "Kills teach a little more.", "war", 0, lbPass("scholar")},
	{"recharge", "a slow tide of power", "Charges return over time.", "r", 0, lbPass("recharge")},
}

// lbAwakeNames: the three awakenings.
var lbAwakeNames = []string{"Stirred", "Waking", "Awake"}
