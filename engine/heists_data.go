package engine

// Heist data tables (prefix ht). Text only; heists.go does the work.

type htLootDef struct {
	Name  string
	Base  int  // value at tier 2
	Sent  bool // sentimental: an heir or collector will hunt for it
	Risk  int  // 1 common goods .. 3 unmistakable
	Glyph rune
	Text  string
}

type htTargetDef struct {
	ID     string
	Place  string // "{owner}'s counting-house"
	Short  string
	Pitch  string // what the contact says
	Guards int
	Loot   []htLootDef
}

var htTargets = []htTargetDef{
	{ID: "vault", Place: "the counting-house of {owner}", Short: "merchant house vault",
		Pitch: "{owner} keeps a vault under the counting-house and trusts the lock more than the men guarding it.", Guards: 0,
		Loot: []htLootDef{
			{"bar of stamped silver", 120, false, 1, '=', "Stamped with the house mark. Cold and heavy."},
			{"sealed letters of credit", 90, false, 2, '?', "Worth more to the person who can cash them than to the one who holds them."},
			{"ledger of secret debts", 70, false, 2, '+', "Names of people who would pay to see this burned."},
			{"jewelled scale weights", 80, false, 1, '*', "A set of weights worked in gold and garnet."}}},
	{ID: "manor", Place: "the manor of {owner}", Short: "noble's manor",
		Pitch: "{owner} sleeps in the east wing and keeps the family plate in a locked strong room off the hall.", Guards: 1,
		Loot: []htLootDef{
			{"silver table service", 130, false, 1, '=', "Forty pieces, each stamped with a crest."},
			{"signet ring of the house", 110, true, 3, '*', "Old, worn smooth by generations of thumbs."},
			{"portrait in a gilt frame", 100, true, 3, '#', "The painted face looks very much like the owner."},
			{"ancestral dress sword", 140, true, 3, '/', "A court blade with a name engraved on the guard."}}},
	{ID: "temple", Place: "the treasury of the temple of {owner}", Short: "temple treasury",
		Pitch: "The treasury of {owner} holds a year of offerings behind one iron door, and the night priest is old.", Guards: 0,
		Loot: []htLootDef{
			{"gilded offering bowl", 100, false, 2, 'u', "Dented where pilgrims dropped coin."},
			{"reliquary of a saint", 150, true, 3, '*', "Pilgrims travel for months to touch the glass."},
			{"votive coins in a bag", 80, false, 1, '$', "Small gifts of the poor, counted by the handful."},
			{"silver processional lamp", 90, true, 2, '!', "Carried through the streets on feast days."}}},
	{ID: "display", Place: "the gallery of {owner}", Short: "collector's gallery",
		Pitch: "{owner} collects rarities and shows them off behind glass. The cases are locked, and the rest is vanity.", Guards: 1,
		Loot: []htLootDef{
			{"painted burial vase", 120, false, 2, 'U', "Very old. The painting shows a procession of mourners."},
			{"cased fossil with a gold label", 100, false, 2, '%', "A collector would call it priceless."},
			{"illuminated psalter", 130, true, 3, '+', "The pages shine. A hundred scribes' years went into it."},
			{"jade seal of a vanished king", 150, true, 3, '*', "Carved with the name of a court no one remembers."}}},
	{ID: "depot", Place: "the depot of the {owner} caravan company", Short: "caravan depot",
		Pitch: "{owner} stores the tolls and the best bales in a yard behind the stables. It is guarded by hired blades who are bored.", Guards: 1,
		Loot: []htLootDef{
			{"bale of saffron", 90, false, 1, '&', "Light, fragrant and worth its weight in silver."},
			{"crate of dyed silk", 110, false, 1, '&', "Sealed with the company's wax."},
			{"strongbox of road tolls", 100, false, 1, '$', "Coins from three realms, mixed."},
			{"manifest of secret routes", 70, false, 2, '?', "Rival companies would pay to read the safe roads."}}},
	{ID: "safe", Place: "the strong room of the guildmaster {owner}", Short: "guildmaster's safe",
		Pitch: "{owner} keeps the guild's charters and spare funds in a safe in their own office. They also keep a bad temper.", Guards: 1,
		Loot: []htLootDef{
			{"guild charter on vellum", 100, true, 3, '?', "The founding charter, with the first masters' marks."},
			{"master's seal ring", 90, true, 3, '*', "Presses the guild's seal in wax. Quite unmistakable."},
			{"purse of guild funds", 120, false, 1, '$', "Dues for the quarter, in good coin."},
			{"cut gemstones in a pouch", 130, false, 1, '*', "Loose and unmarked: a thief's dream."}}},
}

type htPrizeDef struct{ Name, Text string }

var htPrizes = []htPrizeDef{
	{"hidden dowry chest", "Someone bricked this up on purpose. It rattles with old coin."},
	{"jewelled hip flask", "Empty, but studded with stones."},
	{"bundle of old deeds", "Deeds to farms, mills and a ferry, all in the owner's hand."},
	{"gold reliquary box", "Nothing inside but a lock of hair and a ring."},
	{"velvet-wrapped necklace", "A heavy collar of pearls, unworn for a generation."},
}

type htApproachDef struct {
	ID, Name, Desc string
	Heat           int // heat gain modifier percent
}

var htApproaches = []htApproachDef{
	{"sneak", "Sneak in", "Go through the front door at night, past the watch. Needs quiet feet.", 100},
	{"disguise", "Pass as staff", "Walk in wearing the house colours or carrying forged papers. Guards look at you less.", 60},
	{"con", "Talk your way in", "Persuade the door guard to let you in. One roll at the door decides how it goes.", 70},
	{"smash", "Smash and grab", "Break the vault door and run. Loud, quick, and the guards come at once.", 160},
	{"tunnel", "Tunnel from a cellar", "Dig in from outside the walls to the back of the vault. Needs a digger and the secret way.", 40},
	{"roof", "Rooftop and rope", "Drop through a skylight into the hall. Needs rope and light feet.", 80},
}

type htIntelDef struct {
	ID, Label string
}

var htIntel = []htIntelDef{
	{"shifts", "Guard shifts"},
	{"layout", "Floor plan"},
	{"locks", "Lock grade"},
	{"alarms", "Alarms"},
	{"dogs", "Dogs and wards"},
	{"secret", "Secret way in"},
}

type htSpecDef struct {
	ID, Name, Desc string
}

var htSpecs = []htSpecDef{
	{"lockman", "lockman", "Eases the vault lock by one grade."},
	{"lookout", "lookout", "Spots the watch early: sight heat falls by a third."},
	{"muscle", "muscle", "Takes one guard out of the hall before you start."},
	{"face", "face", "Smooth talk: your con and disguise rolls gain +4."},
	{"digger", "digger", "Digs the tunnel without you needing tools."},
	{"cutter", "cutter", "Knows a buyer: fenced heist goods fetch 10 percent more."},
}

var htGearList = []struct{ ID, Name, Why string }{
	{"picks", "lockpicks", "for the vault door"},
	{"rope", "rope", "for the rooftop"},
	{"sedative", "sedative draught", "to put a guard to sleep"},
	{"smoke", "smoke bomb", "to cover a run"},
	{"papers", "forged papers or house livery", "for the disguise"},
	{"digging", "shovel or pick", "for the tunnel"},
}

var htContactLines = []string{
	"\"There is a job, if your hands are steady. {pitch}\"",
	"\"I hear things. {pitch} Interested?\"",
	"\"Not for just anyone. {pitch} Pay is good, and nobody is going to ask who you are.\"",
	"\"A friend of a friend wants something moved from one cellar to another. {pitch}\"",
}

var htGuardNames = []string{"house guard", "hired blade", "night watchman", "door warden"}

var htSewerNote = "A sewer grate, loose on its hinge."
var htRoofNote = "Your rope hangs here, up to the roof."
var htWireNote = "A thin wire at ankle height."

var htRumourLines = []string{
	"{owner} was robbed in the night and the guards found nothing but an open vault.",
	"They say a thief walked out of {place} with a king's ransom and nobody saw a face.",
	"The alarm bells rang at {place} before dawn, and the gates were shut until noon.",
}

var htNewsLines = []string{
	"Thieves broke into {place}. The watch has doubled its patrols.",
	"A burglary at {place}: the owner says the loss is ruinous.",
}

var htNewsCaught = []string{
	"A thief was caught breaking into {place} and was taken in chains.",
}
