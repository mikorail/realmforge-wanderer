package engine

// Arcane (ar): data tables for spell research, pacts and masters. Logic in arcane.go, screen in arcane_ui.go.

// arGenreSkin names the three families of flavour and the per-genre nouns.
type arGenreSkin struct {
	Fam    string // arcane tech bio
	School string // what the craft is called
	Power  string // one researched power
	Pact   string // a pact
	Rite   string // a ceremony
	Master string // a teacher
	Site   string // a research site word
}

var arGenres = map[string]arGenreSkin{
	"fantasy":    {"arcane", "Arcana", "spell", "pact", "rite", "archmage", "sanctum"},
	"mythic":     {"arcane", "Theurgy", "rite-gift", "oath", "sacrifice", "oracle", "temenos"},
	"twilight":   {"arcane", "Hollow Craft", "whisper", "vow", "vigil", "pale hermit", "crypt"},
	"steampunk":  {"tech", "Contrivance", "contrivance", "contract", "calibration", "savant", "workshop"},
	"dieselpunk": {"tech", "Gambits", "gambit", "arrangement", "run", "fixer", "garage"},
	"atompunk":   {"tech", "Protocols", "protocol", "directive", "procedure", "doctor", "lab"},
	"cyberpunk":  {"tech", "Programs", "program", "licence", "ritual-run", "netrunner sage", "server room"},
	"biopunk":    {"bio", "Mutagenics", "mutagen", "symbiosis", "culture-rite", "geneticist", "gene vat"},
	"solarpunk":  {"bio", "Bloom-song", "bloom", "covenant", "gathering", "grove-keeper", "greenhouse"},
	"wasteland":  {"bio", "Jury-rigging", "jury-rig", "bargain", "burn", "old-timer", "stash"},
}

func arSkinFor(genre string) arGenreSkin {
	if s, ok := arGenres[genre]; ok {
		return s
	}
	return arGenres["fantasy"]
}

var arElems = []string{"fire", "frost", "lightning", "force", "life", "death", "mind", "shadow"}

var arElemWord = map[string]map[string]string{
	"arcane": {"fire": "Ember", "frost": "Rime", "lightning": "Storm", "force": "Sundering", "life": "Verdant", "death": "Grave", "mind": "Mind", "shadow": "Umbral"},
	"tech":   {"fire": "Thermal", "frost": "Cryo", "lightning": "Arc", "force": "Kinetic", "life": "Medic", "death": "Entropy", "mind": "Neural", "shadow": "Stealth"},
	"bio":    {"fire": "Cautery", "frost": "Chill", "lightning": "Galvanic", "force": "Muscle", "life": "Graft", "death": "Necrotic", "mind": "Psychic", "shadow": "Pale"},
}

var arForms = []string{"bolt", "cone", "ball", "line", "heal", "ward", "summon", "teleport", "slow", "blind", "drain"}

var arFormNoun = map[string]map[string]string{
	"arcane": {"bolt": "Lance", "cone": "Fan", "ball": "Burst", "line": "Ray", "heal": "Mending", "ward": "Ward", "summon": "Calling", "teleport": "Step", "slow": "Fetter", "blind": "Veil", "drain": "Siphon"},
	"tech":   {"bolt": "Round", "cone": "Spread", "ball": "Charge", "line": "Beam", "heal": "Patch", "ward": "Shield", "summon": "Drone", "teleport": "Jump", "slow": "Throttle", "blind": "Jammer", "drain": "Leech"},
	"bio":    {"bolt": "Spit", "cone": "Spray", "ball": "Spore-burst", "line": "Lash", "heal": "Regrowth", "ward": "Carapace", "summon": "Brood", "teleport": "Slip", "slow": "Sap", "blind": "Ink", "drain": "Feeding"},
}

var arFormDesc = map[string]string{
	"bolt": "a bolt at one target", "cone": "a cone that sweeps everything ahead", "ball": "a burst that blooms around a point",
	"line": "a beam that pierces everything along a line", "heal": "mends you and allies close by", "ward": "wards you and allies close by",
	"summon": "calls a servant to fight for you for a while", "teleport": "carries you to a place you can see",
	"slow": "drags a target's pace down", "blind": "robs a target of its sight", "drain": "pulls life out of a target and into you",
}

// base cooldown (turns) and range per form
var arFormCool = map[string]int{"bolt": 8, "cone": 12, "ball": 14, "line": 12, "heal": 40, "ward": 40, "summon": 90, "teleport": 40, "slow": 18, "blind": 22, "drain": 12}

// weights of forms per focus element
var arFormW = map[string]map[string]int{
	"fire":      {"bolt": 5, "cone": 5, "ball": 6, "line": 3, "ward": 1, "summon": 2},
	"frost":     {"bolt": 4, "cone": 4, "line": 4, "ball": 3, "slow": 6, "ward": 2},
	"lightning": {"bolt": 6, "line": 6, "ball": 3, "cone": 2, "teleport": 3, "slow": 1},
	"force":     {"ball": 5, "cone": 5, "ward": 6, "teleport": 4, "bolt": 3, "line": 2, "slow": 2},
	"life":      {"heal": 9, "ward": 5, "summon": 4, "drain": 1, "slow": 1},
	"death":     {"drain": 7, "summon": 5, "blind": 3, "bolt": 3, "cone": 2, "slow": 2},
	"mind":      {"slow": 5, "blind": 5, "ward": 3, "teleport": 4, "bolt": 3, "summon": 2, "drain": 2},
	"shadow":    {"blind": 6, "teleport": 5, "summon": 4, "drain": 3, "cone": 2, "bolt": 2, "slow": 2},
}

// which form a known power nudges a researched one towards
var arPowForm = map[string]string{
	"flames": "bolt", "freeze": "line", "lightning": "bolt", "psiblast": "bolt", "teleport": "teleport", "phase": "teleport",
	"heal": "heal", "regen": "heal", "forcewall": "ward", "forcebubble": "ward", "leech": "drain", "syphon": "drain",
	"beguile": "summon", "fugue": "summon", "corrosive": "cone", "stunforce": "ball", "disintegrate": "line",
	"arcanebolt": "bolt", "slumber": "blind", "clairvoyance": "blind", "timedilation": "slow",
}

var arPowElem = map[string]string{
	"flames": "fire", "freeze": "frost", "lightning": "lightning", "psiblast": "mind", "teleport": "force", "phase": "shadow",
	"heal": "life", "regen": "life", "forcewall": "force", "forcebubble": "force", "leech": "death", "syphon": "death",
	"beguile": "mind", "fugue": "mind", "corrosive": "death", "stunforce": "force", "disintegrate": "force",
	"arcanebolt": "force", "slumber": "mind", "clairvoyance": "mind", "timedilation": "mind", "sprint": "lightning",
}

var arEpithets = []string{"Aldra's", "Venn's", "Orrin's", "Sethe's", "Maro's", "Quill's", "Ysolde's", "Brask's", "Tamsin's", "Kell's",
	"Iro's", "Dray's", "Hale's", "Nym's", "Pell's", "Ruvo's", "Sable's", "Thane's", "Ulla's", "Wren's", "Corvin's", "Dessa's", "Fenn's", "Gault's"}

var arQualNames = []string{"crude", "rough", "sound", "fine", "masterful"}

// flaws: kind, label, which elements favour it
type arFlawDef struct {
	ID, Label, Desc string
}

var arFlaws = map[string]arFlawDef{
	"scorch": {"scorch", "scorching backlash", "the working burns the caster"},
	"numb":   {"numb", "numbing recoil", "the working numbs and stuns the caster"},
	"wild":   {"wild", "wild surge", "the working may spill into wild magic"},
	"hex":    {"hex", "lingering hex", "a failed working leaves a curse behind"},
	"thirst": {"thirst", "parching draw", "the working drinks the caster dry"},
	"summon": {"summon", "unruly echo", "a failed working calls something hostile"},
}

var arElemFlaws = map[string][]string{
	"fire": {"scorch", "scorch", "wild"}, "frost": {"numb", "numb", "thirst"}, "lightning": {"numb", "scorch", "wild"},
	"force": {"wild", "numb", "scorch"}, "life": {"thirst", "hex", "wild"}, "death": {"hex", "hex", "summon"},
	"mind": {"wild", "hex", "numb"}, "shadow": {"hex", "summon", "wild"},
}

// ---- patrons ----

var arKinds = []string{"spirit", "demon", "angel", "machine", "ancestor", "elder"}

var arKindLabel = map[string]map[string]string{
	"arcane": {"spirit": "spirit", "demon": "demon", "angel": "celestial", "machine": "bound construct", "ancestor": "ancestor", "elder": "elder thing"},
	"tech":   {"spirit": "ghost in the wire", "demon": "rogue daemon", "angel": "guardian system", "machine": "machine mind", "ancestor": "archived ancestor", "elder": "elder intelligence"},
	"bio":    {"spirit": "green spirit", "demon": "feral strain", "angel": "pure line", "machine": "hive mind", "ancestor": "ancestral memory", "elder": "deep progenitor"},
}

var arKindElems = map[string][]string{
	"spirit": {"life", "mind", "shadow"}, "demon": {"fire", "death", "shadow"}, "angel": {"life", "force", "lightning"},
	"machine": {"lightning", "mind", "force"}, "ancestor": {"life", "death", "frost"}, "elder": {"shadow", "mind", "death"},
}

// servants a patron sends, by rank 1..4
var arKindServants = map[string][4]string{
	"spirit": {"wisp", "wisp", "ashwraith", "ashwraith"}, "demon": {"ghoul", "salamander", "magma", "horror"},
	"angel": {"wisp", "wisp", "crystalgolem", "crystalgolem"}, "machine": {"golem", "golem", "crystalgolem", "golem"},
	"ancestor": {"skeleton", "skeleton", "wight", "wight"}, "elder": {"shambler", "shambler", "horror", "horror"},
}

var arElemAttr = map[string]int{"fire": STR, "frost": TOU, "lightning": AGI, "force": STR, "life": TOU, "death": WIL, "mind": INT, "shadow": AGI}
var arElemPower = map[string]string{"fire": "flames", "frost": "freeze", "lightning": "lightning", "force": "forcewall", "life": "heal", "death": "leech", "mind": "psiblast", "shadow": "phase"}
var arElemServant = map[string]string{"fire": "salamander", "frost": "yeti", "lightning": "wisp", "force": "golem", "life": "bear", "death": "skeleton", "mind": "wisp", "shadow": "ashwraith"}

var arFallbackPatrons = []string{"the Pale Hand", "Old Lantern", "the Quiet Below", "Mother Ashen", "the Hollow King", "the Thrice-Named"}

// ---- terms ----

var arTabooText = map[string]string{
	"night":  "never kill after dark",
	"temple": "never enter a temple",
	"beasts": "never slay wild beasts",
	"meat":   "never eat meat",
	"hoard":  "never hold more than %s",
}

var arMeatWords = []string{"meat", "flesh", "steak", "jerky", "haunch", "sausage", "ham", "bacon", "fillet", "ribs", "venison", "mutton", "pork", "beef", "chop", "roast", "drumstick", "cutlet", "brisket"}

// ---- masters ----

var arMasterRoles = map[string]bool{"magician": true, "thinker": true, "prophet": true, "mage": true, "sage": true, "scholar": true, "philosopher": true}

var arPlaces = map[string][]string{
	"mountain": {"mountain cloister", "cliffside hermitage", "snowbound monastery"},
	"forest":   {"wizard's hut", "hollow-tree hermitage", "mossy tower"},
	"wild":     {"lonely tower", "hermit's cairn", "windswept hut", "stone cell"},
}

var arRiddles = []string{
	"What has roots nobody sees, is taller than trees, up, up it goes, and yet never grows?",
	"The more of me there is, the less you see. What am I?",
	"I speak without a mouth and hear without ears. I have no body, but I come alive with wind. What am I?",
	"What is given before it is taken, kept only by being spent, and heavier the longer it is held?",
	"Walk into the fire and be unburnt; walk into the water and be unwet. What road is it?",
	"I am always coming, never arrive, and every wise person has already met me. What am I?",
}

var arMasterNames = []string{"Ovrel", "Isenth", "Maelor", "Quenna", "Dravik", "Sunnah", "Tobiel", "Ermine", "Halvard", "Kessra", "Lunthe", "Bosk", "Aurelio", "Neviss", "Yarrow", "Corbin"}
