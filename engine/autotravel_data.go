package engine

// Static tables for auto-travel (autotravel.go, prefix at).

// atZoneKind is one thing the in-zone "walk to" can head for.
type atZoneKind struct {
	Key   string
	Label string
	Tiles []Terrain
	Hint  string // shown on arrival
}

var atZoneKinds = []atZoneKind{
	{"down", "the stairs down", []Terrain{TDown}, "Press > to descend."},
	{"up", "the stairs up", []Terrain{TUp}, "Press < to climb."},
	{"gate", "the town gate", []Terrain{TGateArch, TGateClosed}, "Bump a portcullis to open it."},
	{"door", "the nearest door", []Terrain{TDoor}, ""},
	{"well", "the nearest well", []Terrain{TWell}, ""},
	{"stall", "the nearest market stall", []Terrain{TStall}, ""},
	{"altar", "the nearest altar", []Terrain{TAltar}, ""},
}

// atImportantWords make a travel message an interrupt even when it carries no warning colour.
var atImportantWords = []string{
	"quest", "deadline", "hostage", "siege", "plague", "caravan", "waylaid", "ambush", "army", "armies",
	"bandit", "raid", "war ", "storm", "blizzard", "flood", "starv", "dying", "stolen", "stole", "lame",
	"died", "dead", "event", "omen", "herald", "messenger",
}

// atBenignPrefixes are travel messages that never interrupt.
var atBenignPrefixes = []string{"You travel ", "You set out", "You arrive"}

const (
	atLowHPPct   = 50  // stop the trip when HP falls below this share
	atMaxLog     = 12  // trips kept in the log
	atMaxQueue   = 8   // waypoint-to-waypoint legs
	atWalkSteps  = 250 // in-zone auto-walk cap
	atHungerWarn = 1200
	atThirstWarn = 900
)
