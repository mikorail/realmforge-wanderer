package engine

// Static text and tables for display cases (display.go) and museums (museum.go). Prefix hm.

// case kinds
const (
	hmKSet    = "set"
	hmKGear   = "gear"
	hmKHead   = "head"
	hmKRelic  = "relic"
	hmKGrave  = "grave"
	hmKUnique = "unique"
	hmKFossil = "fossil"
	hmKFind   = "find"
)

var hmKindName = map[string]string{
	hmKSet: "Collection", hmKGear: "Heirloom", hmKHead: "Mounted head", hmKRelic: "Relic",
	hmKGrave: "Grave good", hmKUnique: "Unique item", hmKFossil: "Fossil", hmKFind: "Great find",
}

var hmKindCol = map[string]string{
	hmKSet: "Y", hmKGear: "O", hmKHead: "o", hmKRelic: "M", hmKGrave: "c", hmKUnique: "W", hmKFossil: "y", hmKFind: "G",
}

// museum wings
const (
	hmWNature = 0
	hmWAntiq  = 1
	hmWFolk   = 2
)

var hmWingName = []string{"Natural history", "Antiquities", "Folk life"}
var hmWingDesc = []string{
	"fossils and mounted beasts",
	"relics, grave goods and appraised finds",
	"collections and the small things of daily life",
}

// museum milestones: total donated items per town museum
var hmMileN = []int{5, 15, 40}

var hmSpouseLines = []string{
	"%s has dusted the cases and turned the newest piece to the light. \"It looks well there.\"",
	"%s stands a while before the cases. \"You have been busy while you were away.\"",
	"%s has put a lamp beside the cases. \"So the visitors can see what you have done.\"",
	"%s smiles at the new piece. \"Every time I think I know all your stories, another one goes on a shelf.\"",
}

var hmSpouseAsk = []string{
	"\"I polish them on rest days. The %s is my favourite, though I would never tell the others.\"",
	"\"Guests ask about the cases before they ask about the soup. I take that as praise.\"",
	"\"They are good company. They do not argue, and they do not ask when you will be home.\"",
}

var hmBurgleLines = []string{
	"{{R|The door of your home has been forced. Thieves came while you were away.}}",
	"{{R|Someone has been through your home: the cases stand open and the lock is cut.}}",
}

var hmCuratorGreet = []string{
	"The curator looks up from a ledger. \"The collection is never finished. How may the museum help you?\"",
	"\"Mind the cases. If you have brought something for the rolls, I am listening.\"",
	"\"We keep what the land and the years give up. Have you something for us?\"",
}

var hmPlaqueFmt = []string{
	"Donated by %s in Y%d: %s.",
	"Given to the people of this town by %s in Y%d: %s.",
	"The gift of %s, Y%d: %s.",
}

var hmTitleNames = []string{"Friend of the %s", "Benefactor of the %s", "Patron of the %s"}

var hmRewardNames = []string{"curator's loupe", "scholar's lens", "keeper's key-ring", "registrar's seal"}
