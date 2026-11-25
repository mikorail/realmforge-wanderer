package engine

// delvemech_data.go (prefix dr): text and tables for delvemech.go. No logic beyond small pickers.

// drKindRow is one room-level set piece.
type drKindRow struct {
	ID    string
	Name  string
	Carve bool   // needs a side chamber cut from solid rock (dungeon levels only)
	Hint  string // one line shown when you arrive on a level that has one
}

var drKindDefs = []drKindRow{
	{"plate", "Pressure plate vault", true, "Somewhere on this level a sealed vault waits for a weight on a worn floor plate."},
	{"lever", "Lever vault", true, "Iron levers stand about this level. One of them is for a sealed vault, and the others are not."},
	{"rune", "Rune door", true, "A door sealed with sigils is on this level; its answer is carved somewhere else."},
	{"turn", "Turning hall", true, "You hear stone shift behind the walls: a hall here turns on a lever."},
	{"flood", "Treasure room that floods", true, "A treasure room on this level has the look of a trap that is paid for in water."},
	{"light", "Braziers to light", false, "Cold braziers stand in this level. A carving says in what order they were lit."},
	{"mirror", "Mirror chamber", true, "Silvered glass glints from a chamber nearby. It shows more than it should."},
	{"guard", "Gallery of statues", false, "Statues stand in this level. Keep your eyes on them."},
	{"idol", "Idol and altar", false, "An idol that takes offerings stands somewhere here, and it takes what it likes."},
	{"arena", "Arena", true, "A sand-floored arena is cut into the rock here. Someone built it to be fought in."},
	{"bridge", "Chasm and bridge", true, "A chasm cut through the rock here is crossed by a plank bridge of doubtful age."},
}

// drKindsByTheme: which set pieces suit a site theme (weights by repetition).
var drKindsByTheme = map[string][]string{
	"catacomb": {"rune", "idol", "guard", "light", "mirror", "flood"},
	"barrow":   {"rune", "idol", "guard", "light", "flood", "plate"},
	"sunken":   {"flood", "plate", "bridge", "lever", "idol", "mirror"},
	"arcane":   {"mirror", "rune", "turn", "light", "plate", "lever"},
	"chasm":    {"bridge", "bridge", "flood", "guard", "arena", "idol"},
	"mine":     {"flood", "bridge", "lever", "plate", "arena", "turn"},
	"hideout":  {"lever", "plate", "idol", "arena", "turn", "rune"},
	"vault":    {"plate", "lever", "turn", "rune", "arena", "mirror"},
	"abyss":    {"mirror", "guard", "idol", "bridge", "flood", "arena"},
	"magic":    {"light", "mirror", "guard", "rune", "turn", "idol"},
	"trials":   {"arena", "plate", "lever", "turn", "bridge", "guard"},
}

var drSyms = []string{"sun", "moon", "serpent", "key", "eye", "hand", "tree", "wave", "crown", "flame"}

var drBrazierNames = []string{"Ash", "Ember", "Smoke", "Cinder", "Spark", "Soot"}

var drLeverLooks = []string{"its grip is worn bright by many hands", "its grip is furred with rust", "its grip is clean and cold, as if never touched"}

var drTrapLines = []string{
	"A dart whispers out of the wall.",
	"A hiss of old gas leaks from a seam.",
	"A bell rings somewhere in the stone.",
}

var drRuneClue = []string{
	"The keepers set the sigils as they were taught: %s, then %s, then %s.",
	"Touch them in the order of the old rite: %s, %s, %s.",
	"What the dead were told at the door: first the %s, after it the %s, last the %s.",
}

var drIdolBoons = []string{"heal", "might", "swift", "ward", "gold", "find", "insight"}
var drIdolCurses = []string{"weak", "slow", "hunt", "tithe"}

var drIdolNames = []string{"a squat idol with too many eyes", "a smiling idol of worn black stone", "a faceless idol holding out both hands", "an idol with a hollow, open mouth"}

var drArenaTaunts = []string{"The sand drinks, and the gate comes down.", "A cold voice from nowhere: 'Show us.'", "The crowd that is not there roars."}

var drFloodTell = []string{"Water seeps through the far wall.", "The floor is wet now, and rising.", "Water pours in from the ceiling seams."}

var drCollapseTell = []string{"Dust sifts from the ceiling.", "A beam cracks above you.", "Stones begin to fall."}

// ---- boss gimmicks ----

type drGimDef struct {
	ID    string
	Name  string
	Start string // shown when the fight begins
	Hint  string // shown on arrival
}

var drGims = map[string]drGimDef{
	"pillars":  {"pillars", "Warded pillars", "Crystal pillars rise around the master of the place; light runs from them into it. Shatter them (use the mechanism next to one).", "Something here is fed by what stands around it."},
	"lava":     {"lava", "Spreading fire", "The floor cracks and glows. Each time it is hurt the heat will spread.", "The deepest hall is warm to the touch."},
	"ice":      {"ice", "Spreading ice", "Frost races over the floor. Each time it is hurt the ice will spread.", "Your breath fogs in the deepest hall."},
	"braziers": {"braziers", "Warded by flame", "Braziers flare to life around the hall. While they burn, the master cannot be hurt; snuff them (use the mechanism next to one).", "Braziers stand in the deepest hall, waiting to be lit."},
	"adds":     {"adds", "Calling the host", "A horn sounds. More will come every little while.", "The deepest hall is built for a garrison."},
	"safezone": {"safezone", "Cursed ground", "The ground beyond a ring around the master goes foul. Stay in the ring; each blow it takes pulls the ring tighter.", "The ground of the deepest hall is not at peace."},
	"mirror":   {"mirror", "Mirror duel", "The master looks at you and, for a moment, so does something else. When it is half spent, it will call up your own reflection.", "There are mirrors in the deepest hall."},
	"weakness": {"weakness", "A known weakness", "You remember what the delve told you. The master has a tender side, and you know where.", ""},
}

// drGimByTheme: candidates per theme; the site key picks one.
var drGimByTheme = map[string][]string{
	"catacomb": {"braziers", "mirror"},
	"barrow":   {"safezone", "braziers"},
	"sunken":   {"ice", "adds"},
	"arcane":   {"pillars", "mirror"},
	"chasm":    {"lava", "safezone"},
	"mine":     {"lava", "pillars"},
	"hideout":  {"adds", "safezone"},
	"vault":    {"pillars", "adds"},
	"abyss":    {"mirror", "ice"},
	"magic":    {"pillars", "mirror"},
	"trials":   {"adds", "lava"},
}

// drGimByTpl: a template that always brings its own trick (50% of the time, by site hash).
var drGimByTpl = map[string]string{
	"golem": "pillars", "crystalgolem": "pillars", "kraken": "adds", "horror": "safezone", "chief": "adds", "wight": "braziers",
}

var drAddsTell = []string{"The master bellows for help.", "Boots and claws on the stone: more are coming.", "A call goes out through the dark."}
