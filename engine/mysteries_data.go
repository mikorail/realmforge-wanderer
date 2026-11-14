package engine

import "strconv"

// Mystery generator data (prefix mx): genre skins, crime kinds, motives, alibis, clue templates.

// Scene slots in mxSkin.Scenes.
const (
	mxHome = iota
	mxShop
	mxAlley
	mxInn
	mxStudy
	mxDepot
)

type mxSkin struct {
	Scenes  [6]string // home shop alley inn study depot
	Weapons [3]string // blade, blunt, shot
	Poison  string    // vessel
	Mark    string    // a small personal object that can be dropped
	Watch   string    // who keeps order
	Tool    string    // what a forger or burglar carries
}

var mxSkins = map[string]mxSkin{
	"fantasy":    {[6]string{"cottage", "shop's back room", "moonlit alley", "inn room", "lord's study", "harbour warehouse"}, [3]string{"dagger", "cudgel", "crossbow bolt"}, "stoppered vial", "signet button", "the watch", "lockpick"},
	"mythic":     {[6]string{"courtyard house", "stall's storeroom", "colonnade alley", "xenon guest room", "archive of the archon", "quayside granary"}, [3]string{"bronze xiphos", "olive-wood club", "javelin"}, "libation cup", "bronze fibula", "the city guard", "bronze pick"},
	"steampunk":  {[6]string{"terraced flat", "shopfront office", "gaslit alley", "boarding-house room", "magistrate's study", "boiler warehouse"}, [3]string{"stiletto", "pipe wrench", "pepperbox ball"}, "laudanum bottle", "brass cufflink", "the constabulary", "skeleton key"},
	"dieselpunk": {[6]string{"tenement flat", "corner shop back office", "fire-escape alley", "motel room", "manager's office", "garage and depot"}, [3]string{"switchblade", "tyre iron", "revolver slug"}, "flask", "lighter", "the precinct", "jemmy"},
	"atompunk":   {[6]string{"prefab bungalow", "drugstore stockroom", "service alley", "motor-lodge room", "laboratory office", "reactor annex"}, [3]string{"letter opener", "bakelite telephone", "pistol slug"}, "pill bottle", "tie clip", "the security office", "lock gun"},
	"cyberpunk":  {[6]string{"capsule flat", "noodle stall back room", "neon alley", "capsule-hotel pod", "server room", "freight dock"}, [3]string{"mono-filament wire", "stun baton", "flechette round"}, "toxin ampoule", "access chip", "corporate security", "decker's spike"},
	"biopunk":    {[6]string{"grown-wood hab", "tissue stall back room", "root-tunnel alley", "hive-inn cell", "greenhouse laboratory", "culture vat store"}, [3]string{"bone scalpel", "thorn club", "spore dart"}, "culture flask", "gene-tag", "the guild wardens", "grafting knife"},
	"solarpunk":  {[6]string{"terrace home", "co-op workshop", "trellis walkway", "guest pod", "greenhouse office", "seed depot"}, [3]string{"pruning knife", "mallet", "bolt-gun dart"}, "tincture bottle", "woven token", "the commons wardens", "grafting shears"},
	"wasteland":  {[6]string{"shanty", "scrap stall back room", "wreck-lined alley", "doss-house bunk", "bunker office", "convoy yard"}, [3]string{"shiv", "length of rebar", "scrap-gun slug"}, "canteen", "dog-tag", "the road wardens", "wire hook"},
	"twilight":   {[6]string{"shuttered townhouse", "chandlery back room", "fog-bound alley", "inn garret", "chantry study", "crypt-side vault"}, [3]string{"silver letter-knife", "candlestick", "wooden stake"}, "chalice", "black glove", "the vigil", "bone key"},
}

// mxKindDef describes one sort of crime.
type mxKindDef struct {
	ID     string
	Noun   string // "death", "theft"...
	Body   bool
	Scenes []int
	Title  string // %s = victim name
	Crime  string // law category
}

var mxKinds = map[string]*mxKindDef{
	"murder":   {"murder", "death", true, []int{mxAlley, mxHome, mxInn, mxShop}, "The death of %s", "murder"},
	"poison":   {"poison", "death", true, []int{mxHome, mxInn, mxStudy}, "The poisoned cup of %s", "murder"},
	"accident": {"accident", "death", true, []int{mxDepot, mxAlley, mxShop}, "The accident that befell %s", "murder"},
	"theft":    {"theft", "theft", false, []int{mxShop, mxStudy, mxDepot}, "Theft from %s", "theft"},
	"arson":    {"arson", "fire", false, []int{mxShop, mxHome}, "The burning of %s's property", "arson"},
	"kidnap":   {"kidnap", "disappearance", false, []int{mxHome, mxAlley}, "The disappearance of %s", "kidnapping"},
	"forgery":  {"forgery", "forgery", false, []int{mxStudy}, "The forged will of %s", "forgery"},
	"shipment": {"shipment", "loss", false, []int{mxDepot}, "The missing shipment of %s", "theft"},
}

var mxKindOrder = []string{"murder", "poison", "accident", "theft", "arson", "kidnap", "forgery", "shipment"}

// Motive kinds with one or two statements each: %[1]s culprit, %[2]s victim.
var mxMotiveTxt = map[string][]string{
	"revenge": {"%[1]s had sworn to settle an old wrong done by %[2]s.", "%[1]s blamed %[2]s for a loss that was never forgiven."},
	"greed":   {"%[1]s was deep in debt, and %[2]s's property would clear it.", "%[1]s stood to gain a great deal from %[2]s's ruin."},
	"love":    {"%[1]s was bound to %[2]s by a love that had turned sour.", "%[1]s could not bear that %[2]s had chosen another."},
	"fear":    {"%[1]s feared what %[2]s would do if the truth came out.", "%[1]s lived in dread of %[2]s and saw no other way out."},
	"silence": {"%[2]s knew a secret that would have ruined %[1]s.", "%[2]s had threatened to speak of what %[1]s did in the dark."},
}

// Document clues by motive. %[1]s culprit, %[2]s victim, %[3]s noun of the crime.
var mxDocTxt = map[string][]string{
	"revenge": {"A diary page in %[2]s's hand: \"%[1]s has not forgiven me, and I do not think %[1]s will.\"", "A hoarded grudge-list in %[1]s's hand with %[2]s's name underlined twice."},
	"greed":   {"%[2]s's ledger: %[1]s owed a heavy sum, due the very week of the %[3]s.", "A pawn-ticket and a note of debt: %[1]s needed coin fast, and %[2]s held the means."},
	"love":    {"A bundle of letters in %[1]s's hand, and a last one from %[2]s breaking it off.", "A pressed flower and a bitter, unsent note from %[1]s to %[2]s."},
	"fear":    {"A hidden note: %[2]s meant to report %[1]s before the week was out.", "%[2]s's draft complaint naming %[1]s, never delivered."},
	"silence": {"A draft letter in %[2]s's hand, threatening to expose %[1]s's secret.", "A blackmail note, half burned, with %[1]s's name on the outside."},
}

// Relationship wording: public reasons each sort of suspect might have.
var mxRelTxt = map[string][]string{
	"rival":        {"%s and the dead had quarrelled in public more than once.", "%s was a known enemy of the dead."},
	"spouse":       {"%s was the dead's spouse and stands to inherit.", "%s shared the dead's house and its troubles."},
	"kin":          {"%s is kin, and the family's property passes to kin.", "%s argued with the dead over an inheritance."},
	"servant":      {"%s worked for the dead and was owed wages.", "%s had been scolded by the dead the day before."},
	"employer":     {"%s employed the dead and wanted something kept quiet.", "%s had a falling-out with the dead over work."},
	"friend":       {"%s was a close friend, and friends hide grudges well.", "%s was seen arguing with the dead last week."},
	"debtor":       {"%s owed the dead money and could not pay.", "%s begged the dead for more time on a debt."},
	"outsider":     {"%s arrived in town only days ago and nobody knows why.", "%s is a stranger who was asking after the dead."},
	"acquaintance": {"%s had dealings with the dead that nobody understands.", "%s was among the last to speak with the dead."},
}

var mxSecrets = []string{
	"has been meeting a lover in secret",
	"has been skimming coin from the till",
	"carried untaxed goods that night",
	"gambled away the household savings",
	"has been forging small entries in a ledger",
	"was sneaking out to a banned gathering",
	"owes a dangerous creditor",
}

var mxAlibiPlaces = []string{
	"at the tavern", "at the evening vigil", "mending tools at the shed", "asleep at home", "walking the streets on an errand",
	"at the market until late", "tending a sick relative", "at the baths", "playing dice with friends", "settling the accounts",
}

var mxSpots = []string{"a bench", "a stack of crates", "a barrel", "the doorframe", "a heap of refuse", "a loose floorboard", "a coat-hook"}

var mxPlaceRoles = map[string]bool{"guard": true, "sergeant": true, "ruler": true, "elder": true, "general": true, "se_investigator": true, "se_sentry": true}

// mxSecretTop gives word for time of day.
func mxHourWord(h int) string {
	h = ((h % 24) + 24) % 24
	switch {
	case h == 0:
		return "midnight"
	case h < 5:
		return strconv.Itoa(h) + " in the small hours"
	case h < 12:
		return strconv.Itoa(h) + " in the morning"
	case h == 12:
		return "noon"
	case h < 18:
		return strconv.Itoa(h-12) + " in the afternoon"
	}
	return strconv.Itoa(h-12) + " in the evening"
}
