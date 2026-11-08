package engine

// Text tables for burials.go (prefix bu): epitaphs, grave goods, curses, genre-skinned bunker kits,
// pirate names and the shifting-ground messages.

// buSphereGroup folds a culture sphere into the burial customs below.
func buSphereGroup(sphere string) string {
	switch sphere {
	case "norse", "celtic", "slavic":
		return "mound"
	case "steppe":
		return "kurgan"
	case "sinic", "japonic", "indic", "persian", "arabic":
		return "tomb"
	case "nahua", "maya", "andean":
		return "pyramid"
	}
	return "cairn"
}

var buMoundWord = map[string][]string{
	"mound":   {"barrow", "burial mound", "turf-covered tumulus"},
	"kurgan":  {"kurgan", "steppe tumulus", "horse-grave mound"},
	"tomb":    {"spirit-way tomb", "tumulus tomb", "walled grave-mound"},
	"pyramid": {"stepped tomb-mound", "buried shrine-tomb", "earthen pyramid"},
	"cairn":   {"cairn", "stone-heaped barrow", "grave mound"},
}

// buEpitaphFlourish ends an epitaph; {n} is the dead person's name.
var buEpitaphFlourish = map[string][]string{
	"mound": {
		"May the earth be kind. Let no hand lift the stones that cover {n}.",
		"Who breaks this door breaks faith with the dead, and the dead keep count.",
		"We laid {n} here with sword and cup. Leave both.",
		"The grass will keep {n}. The grass is patient.",
	},
	"kurgan": {
		"Ride on, {n}. The sky is wide and the horses are fast.",
		"Here {n} sleeps beneath the sky they loved. Let the wind pass over.",
		"Take nothing. The herds of the dead are not for the living.",
	},
	"tomb": {
		"{n} rests in the way of ancestors. Disturb the rite and the house of {n} will answer.",
		"Honour the ways. Burn incense. Walk on.",
		"Seals set, names written, wards laid. Leave {n} in peace.",
	},
	"pyramid": {
		"{n} walks the nine layers beneath. The road is long; do not hinder them.",
		"The sun has taken {n}'s face. The earth holds the rest.",
	},
	"cairn": {
		"Stone upon stone for {n}. One for each who grieved.",
		"Here lies {n}. Pass quietly.",
		"{n}, remembered. The cairn is the promise.",
	},
}

// Grave-good nouns per genre: weapon, adornment, vessel. {n} is the dead person's name.
type buGraveKit struct {
	Weapon, Ornament, Vessel []string
	Coin                     string
}

var buGraveByGenre = map[string]buGraveKit{
	"default": {
		Weapon:   []string{"{n}'s oathblade", "the burial sword of {n}", "{n}'s last spear"},
		Ornament: []string{"torc of {n}", "ring of {n}", "funeral diadem of {n}", "gold-wire brooch of {n}"},
		Vessel:   []string{"funerary urn of {n}", "drinking horn of {n}", "libation cup of {n}"},
		Coin:     "grave-coin",
	},
	"steampunk": {
		Weapon:   []string{"{n}'s brass-hilted sabre", "the dress pistol of {n}"},
		Ornament: []string{"pocket chronometer of {n}", "signet of {n}"},
		Vessel:   []string{"sealed canopic flask of {n}", "gilt reliquary of {n}"},
		Coin:     "sovereign",
	},
	"dieselpunk": {
		Weapon:   []string{"{n}'s service revolver", "the ceremonial sabre of {n}"},
		Ornament: []string{"medal of {n}", "cigarette case of {n}"},
		Vessel:   []string{"sealed flask of {n}", "funerary lockbox of {n}"},
		Coin:     "silver mark",
	},
	"atompunk": {
		Weapon:   []string{"{n}'s ray pistol", "dress laser of {n}"},
		Ornament: []string{"wristwatch of {n}", "lodge pin of {n}"},
		Vessel:   []string{"lead-lined casket box of {n}", "time capsule of {n}"},
		Coin:     "commemorative cap",
	},
	"cyberpunk": {
		Weapon:   []string{"{n}'s custom handgun", "monofilament blade of {n}"},
		Ornament: []string{"neural jack-pin of {n}", "memory shard of {n}"},
		Vessel:   []string{"cold-storage case of {n}", "data urn of {n}"},
		Coin:     "credit chip",
	},
	"biopunk": {
		Weapon:   []string{"{n}'s bone-grown blade", "thorn-lance of {n}"},
		Ornament: []string{"seed-pearl necklace of {n}", "living amulet of {n}"},
		Vessel:   []string{"spore-sealed urn of {n}", "culture jar of {n}"},
		Coin:     "seed-coin",
	},
	"solarpunk": {
		Weapon:   []string{"{n}'s sunsteel blade", "grove-warden's staff of {n}"},
		Ornament: []string{"solar locket of {n}", "garland-ring of {n}"},
		Vessel:   []string{"seed urn of {n}", "memory bowl of {n}"},
		Coin:     "sun-token",
	},
	"wasteland": {
		Weapon:   []string{"{n}'s pipe-rifle", "burial machete of {n}"},
		Ornament: []string{"bottle-cap amulet of {n}", "dog-tags of {n}"},
		Vessel:   []string{"sealed jerrycan of {n}", "tin box of {n}"},
		Coin:     "cap",
	},
	"twilight": {
		Weapon:   []string{"{n}'s ash-forged blade", "gilded dirk of {n}"},
		Ornament: []string{"dusk-glass ring of {n}", "mourning crown of {n}"},
		Vessel:   []string{"urn of {n}'s last light", "gilded reliquary of {n}"},
		Coin:     "tithe-coin",
	},
	"mythic": {
		Weapon:   []string{"{n}'s oathblade", "the heroic spear of {n}", "{n}'s shield-breaker"},
		Ornament: []string{"torc of {n}", "laurel of {n}", "amulet of {n}"},
		Vessel:   []string{"funerary amphora of {n}", "libation cup of {n}"},
		Coin:     "obol",
	},
}

func buKitFor(genre string) buGraveKit {
	if k, ok := buGraveByGenre[genre]; ok {
		return k
	}
	return buGraveByGenre["default"]
}

// buCurses: id, intro text shown when the seal breaks (ward) or the goods are lifted (curse).
var buWardText = map[string]string{
	"poison":   "A breath of old grave-air spills out, sweet and wrong.",
	"collapse": "The lintel groans. Packed earth and stones begin to come down.",
	"none":     "The seal gives with a long sigh. Nothing stirs.",
}

var buCurseText = map[string]string{
	"ghost":   "The air in the chamber turns cold as the first piece leaves its place.",
	"sick":    "A dry, weighing chill settles into your bones as you lift the goods.",
	"both":    "Cold wraps your throat. A figure forms in the dark where the dead lies.",
	"quiet":   "The chamber stays silent. Perhaps the dead had no grudge.",
	"omen":    "Somewhere above, every bird falls silent at once.",
	"hexword": "You find a line scratched inside the lid: the name of whoever lifts these things, and a date.",
}

// bunker kits per genre
type buBunkerKit struct {
	Place   string   // what the sealed place is called
	Hatch   string   // the hidden entrance
	Lock    string   // the lock
	Crate   string   // ration crates
	Arms    []string // pre-war weapon names
	Med     string
	Core    string // data core / archive
	Relic   string
	Hazard  string // description of a leak
	Turret  string
	Note    string // what the notes are
	Dunes   string
	NoteTxt []string
}

var buBunkerByGenre = map[string]buBunkerKit{
	"wasteland": {
		Place: "sealed shelter", Hatch: "a rust-welded steel hatch", Lock: "a keypad lock, its screen dead green", Crate: "sealed ration crate",
		Arms: []string{"pre-war assault carbine", "greased riot shotgun", "sealed service rifle"}, Med: "pre-war med-kit", Core: "pre-war data core", Relic: "unlabelled pre-war device",
		Hazard: "the air hisses with old leaking coolant", Turret: "rusted sentry gun", Note: "a water-stained supervisor's note",
		NoteTxt: []string{"If you are reading this the shelter is open. The code is %s. Do not trust the lower door.", "Sealed her up. Code %s. Whoever finds this, take the medicine and leave the rest.", "Override %s. They never sent the relief convoy."},
	},
	"atompunk": {
		Place: "fallout shelter", Hatch: "a lead-plated hatch", Lock: "a combination dial and a faded sign: AUTHORISED ONLY", Crate: "civil-defence ration crate",
		Arms: []string{"laser rifle (boxed)", "pre-war ray pistol", "sealed plasma caster"}, Med: "radiation med-kit", Core: "reel-to-reel archive", Relic: "prototype device",
		Hazard: "the old air scrubbers vent a faint glowing haze", Turret: "civil-defence sentry", Note: "a typed civil-defence memo",
		NoteTxt: []string{"MEMO: shelter access code revised to %s. Destroy this notice.", "Reminder to all wardens: dial %s for entry. Stockpiles are unrationed."},
	},
	"dieselpunk": {
		Place: "armoured bunker", Hatch: "a riveted blast hatch", Lock: "a three-wheel combination lock", Crate: "army ration crate",
		Arms: []string{"crated trench gun", "oiled service rifle", "sealed submachine gun"}, Med: "field surgeon's kit", Core: "sealed dispatch case", Relic: "experimental engine part",
		Hazard: "a ruptured pipe breathes gas into the passage", Turret: "automatic gun-turret", Note: "a coded dispatch",
		NoteTxt: []string{"Dispatch: bunker lock set to %s. Burn after reading.", "To the garrison: should the line fall, the key is %s."},
	},
	"cyberpunk": {
		Place: "data vault", Hatch: "a sealed maintenance hatch under a collapsed sublevel", Lock: "a biometric panel with a physical override", Crate: "vacuum-sealed ration brick",
		Arms: []string{"corporate security carbine", "black-site railgun pistol", "monowire garotte (boxed)"}, Med: "trauma-pack", Core: "encrypted data core", Relic: "prototype neural rig",
		Hazard: "a cooling leak floods the vestibule with fumes", Turret: "corporate sentry drone", Note: "a scrap of auth-chit printout",
		NoteTxt: []string{"temp override: %s. valid until audit. do NOT log.", "auth: %s. your boss does not know this exists."},
	},
	"biopunk": {
		Place: "gene bank", Hatch: "a membrane-sealed hatch grown over with fungus", Lock: "a living lock that answers to a scent-phrase", Crate: "cold-grown ration vessel",
		Arms: []string{"engineered thorn-gun", "bred spine-lance", "sealed venom launcher"}, Med: "culture of healing mould", Core: "genome archive", Relic: "viable seed-engine",
		Hazard: "spores vent from a ruptured vat", Turret: "defensive tissue-guard", Note: "a smeared lab notebook page",
		NoteTxt: []string{"Lock phrase scent-keyed: %s. The guards will not touch you if you hum.", "Entry: %s. The seed vault is still alive. Do not breathe deep."},
	},
	"twilight": {
		Place: "sealed reliquary", Hatch: "a hatch-stone under the ash", Lock: "a seal of black wax and a cipher dial", Crate: "wax-sealed provision chest",
		Arms: []string{"mourning-guard blade", "ashen warden's bow", "gilded dusk-lance"}, Med: "mourner's salve", Core: "sealed archive tablet", Relic: "forgotten relic of the old light",
		Hazard: "a thin grey mist leaks from the broken seal", Turret: "gilded guardian", Note: "a wax-sealed vigil-note",
		NoteTxt: []string{"Open with %s. The old light slept here when we could not keep it.", "The cipher reads %s. May the dusk be kind."},
	},
}

func buBunkerKitFor(genre string) buBunkerKit {
	if k, ok := buBunkerByGenre[genre]; ok {
		return k
	}
	return buBunkerByGenre["wasteland"]
}

// buBunkerGenre: genres that carry pre-war sealed caches.
func buBunkerGenre(g string) bool {
	switch g {
	case "wasteland", "atompunk", "dieselpunk", "cyberpunk", "biopunk", "twilight":
		return true
	}
	return false
}

var buVaultTypes = []string{"stores", "armoury", "medbay", "archive", "relicvault"}

var buVaultDesc = map[string]string{
	"stores":     "shelves of sealed provisions, still good",
	"armoury":    "racks of weapons, oiled and waiting",
	"medbay":     "cabinets of medical supplies",
	"archive":    "banks of data cores and records",
	"relicvault": "a locked cage of strange devices under dust sheets",
}

// pirate hoards
var buPirateTitles = []string{"Captain", "Black", "Red", "One-Eyed", "Old", "Mad", "Salt", "Grey"}
var buPirateEpithets = []string{"Redbeard", "Gallowtide", "the Drowner", "Saltblood", "Hookhand", "Longknife", "the Gull", "Blackwater", "Sevenscars", "Windfall", "Brinewife", "Coldharbour"}

var buHoardWhere = []string{
	"in a tide-cave behind the shingle",
	"under a cairn on a bare islet",
	"in a sea cave that floods at high water",
	"beneath a wreck-timber marker on a skerry",
	"in a sinkhole above the surf-line",
}

var buHoardGuards = []string{"crabs", "trap", "tide", "ghost"}

var buHolderKinds = []string{"tavern", "rival", "ship"}

var buHolderHint = map[string]string{
	"tavern": "was traded over a drinking-table in the harbour taverns of %s",
	"rival":  "is carried by the captain's old first mate, last seen near %s",
	"ship":   "went down with a captured ship, its logbook washed up somewhere near %s",
}

// shifting ground
var buFloodMsg = []string{
	"Something glints in the retreating floodwater.",
	"The flood has gone down, leaving something bright in the silt.",
	"The river has changed its bed overnight; something shines in the mud.",
}
var buDuneMsg = []string{
	"The dunes have walked since you were last here. Something glints where the sand has peeled back.",
	"The sandstorm has scoured the slope bare, and something dull and metal shows through.",
}
var buQuakeMsg = []string{
	"An old quake has split the ground; something sits in the crack.",
	"A fissure from the last tremor shows the edge of something worked.",
}
var buCalveMsg = []string{
	"A slab of the glacier has calved; something dark lies in the new ice-face.",
	"Meltwater has cut a blue gully, and something is frozen into its wall.",
}
var buShiftItemNames = map[string][]string{
	"flood": {"silt-caked coffer", "waterlogged satchel", "river-worn reliquary"},
	"dune":  {"sand-scoured chest", "half-buried urn", "wind-bared strongbox"},
	"quake": {"fissure-cracked casket", "tomb-fragment", "tumbled grave-cache"},
	"calve": {"ice-bound pack", "frozen traveller's satchel", "glacier-cased box"},
}

// appraisal flavour
var buAppraiseLine = []string{
	"turns it over slowly under the lamp.",
	"scratches the base with a thumbnail and hums.",
	"holds it to the light, then smells it.",
	"consults a thumbed ledger, then a magnifier.",
}

// unusual loot names (unidentified, valued by an appraiser)
var buOddNames = []string{"worn seal-ring", "chipped idol", "etched medallion", "tarnished clasp", "lacquered box", "old reliquary", "bone-and-silver comb", "weighted dice-set", "faded standard-tip", "engraved flask"}

var buOddMakers = []string{
	"a goldsmith of a fallen house", "a forgotten court craftsman", "a smuggler's guild artisan", "a war-captain's household",
	"the vanished workshops of the old league", "a temple foundry", "someone who signed with a single rune",
}
