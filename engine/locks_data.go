package engine

// Static tables for locks.go (prefix lk): lock tiers, picks, traps, magic locks, riddles, words.

var lkTierName = [6]string{"", "simple latch", "ward lock", "deadbolt", "mechanism", "masterwork lock"}

var lkTierBlurb = [6]string{"",
	"A latch and a bent nail of a lock. Anyone patient could open it.",
	"Wards cut into the keyhole turn away a careless pick.",
	"A deadbolt with several pins. It wants real tools and a steady hand.",
	"A clockwork mechanism; the pins reset when you slip.",
	"A masterwork puzzle-lock. Its maker meant it to outlive thieves.",
}

var lkPickName = [6]string{"", "bent-wire pick", "tension set", "thief's picks", "fine pickset", "master's picks"}

// price and wear of picks sold by locksmiths, by quality 1..5
var lkPickPrice = [6]int{0, 25, 70, 160, 360, 800}

type lkTrapDef struct {
	ID, Name, Sign string
}

var lkTrapDefs = []lkTrapDef{
	{"needle", "poison needle", "a pinprick hole by the keyhole, crusted green"},
	{"gas", "gas vent", "a grille behind the lock and a sour smell"},
	{"alarm", "alarm bell", "a thin wire running from the lid into the dark"},
	{"blades", "scything blades", "slits in the casing, bright with oil"},
	{"mimic", "mimic", "a lid that is warm to the touch, and breathes"},
	{"explosive", "powder charge", "a hot-metal whiff and a fuse-tail under the hasp"},
}

func lkTrapByID(id string) lkTrapDef {
	for _, t := range lkTrapDefs {
		if t.ID == id {
			return t
		}
	}
	return lkTrapDefs[0]
}

var lkMagicList = []string{"riddle", "blood", "password", "ally", "sun", "moon"}

var lkMagicText = map[string]string{
	"riddle":   "A ward speaks: it will open only for the right answer to a riddle.",
	"blood":    "A sigil is cut into the metal. It thirsts for blood.",
	"password": "A mouth of ward-letters waits for a spoken word.",
	"ally":     "Two hand-plates sit too far apart for one person. It wants a companion at your side.",
	"sun":      "A sunburst is inlaid in the lock. It yields only in daylight.",
	"moon":     "A crescent is inlaid in the lock. It yields only at night.",
}

var lkChestByKind = map[string]string{
	"dungeon":   "iron-bound chest",
	"tower":     "scholar's coffer",
	"crypt":     "burial coffer",
	"ruin":      "rotted strongbox",
	"camp":      "supply chest",
	"lair":      "hoard chest",
	"underdark": "stone coffer",
	"town":      "household chest",
	"caravel":   "sea chest",
	"steamer":   "purser's strongbox",
}

var lkWords = []string{
	"ember", "thistle", "marrow", "lantern", "cinder", "hollow", "saffron", "tallow", "gallows", "vesper",
	"quill", "oxbow", "pilgrim", "brine", "kestrel", "sorrel", "ashlar", "wicker", "morrow", "tinder",
	"bramble", "cairn", "dirge", "harrow", "sable", "ferrule", "mistral", "obol",
}

var lkSplitA = []string{"Veiled", "Ashen", "Sunken", "Gilded", "Hollow", "Drowned", "Sleepless", "Crowned"}
var lkSplitB = []string{"Reliquary", "Strongroom", "Hoard", "Vault", "Treasury"}

// static riddles used when the world has too little lore: question, answer, wrong answers, inscription
type lkStaticRiddle struct {
	Q, A string
	W    [3]string
	Hint string
}

var lkStaticRiddles = []lkStaticRiddle{
	{"I have no voice, yet I am heard in every empty hall. What am I?", "an echo", [3]string{"a bell", "a ghost", "the wind"}, "Scratched in the plaster: the dead answer you in your own voice. An echo."},
	{"The more you take, the more you leave behind. What am I?", "footsteps", [3]string{"memories", "coins", "shadows"}, "A line of boot-prints is chiselled by the door. Footsteps are what you leave."},
	{"I am always hungry and must be fed; the finger I touch will soon turn red. What am I?", "fire", [3]string{"a wolf", "a blade", "rust"}, "A black scorch-mark and one word beneath it: fire."},
	{"What is broken the moment you name it?", "silence", [3]string{"a promise", "a mirror", "an oath"}, "On the wall, in a hush of dust: silence, kept, is silence broken."},
}

// shop and locksmith flavour
var lkSmithLines = []string{
	"\"Every lock wants to be opened. I only speed it along.\"",
	"\"Pins, wards, springs. Learn what a lock is afraid of.\"",
	"\"Mind the watch. I sell tools, not alibis.\"",
}
