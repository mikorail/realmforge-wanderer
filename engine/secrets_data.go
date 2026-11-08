package engine

// Text for secrets.go (sx): hints, reveals, clue inscriptions and rumours.

// sxHintLines: shown when passive Perception senses an unfound secret next to the player.
var sxHintLines = map[string][]string{
	"door": {
		"The draught from this wall is wrong.",
		"The dust lies differently along this stretch of wall.",
		"Mortar here is newer than the stone around it.",
		"A faint, cold breath seeps from a seam in the wall.",
	},
	"cut": {
		"You hear footsteps on the far side of this wall.",
		"The wall hums with the sound of another passage behind it.",
		"A thin line of old soot marks a seam in the stone.",
	},
	"panel": {
		"One flagstone sits a hair proud of its neighbours.",
		"The floor ahead rings hollow under your tread.",
		"A thread of dust has been swept into a seam in the floor.",
	},
	"vault": {
		"Something heavy is shut away behind this wall; the air tastes of old iron.",
		"The wall here is thicker on the outside than the inside.",
	},
	"hollow": {
		"The %s sounds hollow.",
		"Something is not right about the %s: it sits too light.",
		"A seam runs round the %s where there should be none.",
	},
}

// sxRevealLines: said when a secret is found.
var sxRevealLines = map[string][]string{
	"door":  {"A section of wall swings inward: a hidden door!", "You press the right stone and the wall grinds aside: a secret door!"},
	"cut":   {"The wall pivots on a hidden spindle: a shortcut!", "A concealed door sticks, then gives: a passage between the halls."},
	"panel": {"You find the hidden panel in the floor: a trapdoor!", "A flagstone lifts on a hinge: a trapdoor."},
	"vault": {"The wall gives way to a sealed vault door!", "Behind the false wall lies a sealed strongroom."},
}

// sxContainerNames per base terrain name: how the hollow thing is called and where the cache sits.
var sxContainerNames = map[string][]string{
	"statue":  {"A hollow in the statue's base holds a cache.", "The statue's head lifts off: a cache sits in the neck."},
	"altar":   {"A slab slides behind the altar, uncovering the offerings of generations.", "The altar's foot hides a reliquary."},
	"well":    {"A cord hangs from a ledge below the rim: a cache in an oilskin bag.", "A loose stone in the well shaft hides a cache."},
	"tree":    {"The tree is hollow, and something was left in it.", "A hollow in the trunk holds an oilskin bundle."},
	"pillar":  {"A panel in the pillar opens on a narrow cavity.", "The pillar is hollow at the foot: a cache."},
	"shelf":   {"The back of the shelf is false: a cache behind it.", "A book on the shelf is a box."},
	"tower":   {"A loose stone at the tower's base hides a cache.", "Behind the bell-rope housing lies a stash."},
	"bell":    {"A hollow in the bell-tower's cornerstone holds a cache.", "The bell yoke hides a small cache."},
	"sarcoph": {"A false bottom in the sarcophagus holds the grave goods.", "Under the bones, a second lid."},
}

// sxClueLines: inscriptions pointing at a secret. {dir} and {far} are filled in.
var sxClueLines = map[string][]string{
	"door": {
		"The masons left a door the ledgers never named: knock {far} to the {dir}.",
		"{far} to the {dir}, a wall keeps a room that no map shows.",
		"What the lord hid, he hid {far} to the {dir}, behind plain stone.",
	},
	"cut": {
		"The couriers cut through {far} to the {dir}; the wall there is only a door in disguise.",
		"There is a quicker way, {far} to the {dir}, for those who know the seam.",
	},
	"panel": {
		"Tread lightly {far} to the {dir}: the floor there is a lid.",
		"A thief fell {far} to the {dir} and was never seen again; his coin was found below.",
	},
	"vault": {
		"The strongroom lies {far} to the {dir}, behind a wall that pretends to be ordinary.",
		"Gold sleeps {far} to the {dir}; the way in is from the other side, or from above.",
	},
	"hollow": {
		"Not all that is carved is solid: look {far} to the {dir}.",
		"Someone left a gift {far} to the {dir}, inside what looks whole.",
	},
}

// sxKeeperLines: the secret-keeper's talk.
var sxKeeperLines = []string{
	"\"Walls have ears, friend, and sometimes doors.\"",
	"\"Some halls have more rooms than their builders admitted to.\"",
	"\"I know where the old masons were careless.\"",
}
