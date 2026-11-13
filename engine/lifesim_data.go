package engine

import "strings"

// lifesim_data.go: tables for the life simulation (lifesim.go, prefix ls).

// lsLifespans: typical lifespans in years by species id (substring match, lowercase). Default 70.
var lsLifespans = []struct {
	id string
	n  int
}{
	{"void_elf", 600}, {"elf", 600}, {"dwarf", 250}, {"orc", 50}, {"goblin", 45}, {"halfling", 110},
	{"ratfolk", 35}, {"lizardfolk", 90}, {"giant", 150}, {"ogor", 60}, {"troll", 90}, {"saurian", 100},
	{"merfolk", 120}, {"celestial", 300}, {"beastman", 55}, {"chitin", 20}, {"warp_fiend", 80},
}

func lsLifespan(species string) int {
	s := strings.ToLower(species)
	for _, e := range lsLifespans {
		if strings.Contains(s, e.id) {
			return e.n
		}
	}
	return 70
}

// lsAdultAge: years until a child takes a job (16, less for short-lived peoples).
func lsAdultAge(species string) int {
	l := lsLifespan(species)
	return clampi(l*16/70, 8, 16)
}

// lsShopJobs are trades whose holder owns a shop that passes on at death or retirement.
var lsShopJobs = map[string]bool{
	"baker": true, "butcher": true, "smith": true, "merchant": true, "innkeeper": true, "tailor": true,
	"apothecary": true, "brewer": true, "grocer": true, "jeweler": true, "jeweller": true, "bookseller": true,
	"potter": true, "tanner": true, "carpenter": true, "weaver": true, "cobbler": true, "shopkeeper": true,
	"trader": true, "miller": true, "herbalist": true, "fishmonger": true, "chandler": true, "tavernkeeper": true,
	"cooper": true, "mason": true, "glazier": true, "shipwright": true, "armourer": true, "armorer": true,
	"fletcher": true, "barber": true, "dyer": true, "locksmith": true, "stablemaster": true, "ostler": true,
}

// lsEntryJobs: first jobs of those who grow up without a parent's trade.
var lsEntryJobs = []string{"laborer", "farmhand", "apprentice", "servant", "porter", "laborer", "farmhand", "apprentice"}

// lsPortJobs are added for harbour towns.
var lsPortJobs = []string{"fisher", "dockhand", "sailor"}

// lsLadder: promotions by job: next rung, wealth needed, minimum years past adulthood, minimum conscientiousness.
var lsLadder = map[string]struct {
	next          string
	wealth, years int
	con           int
}{
	"laborer":    {"craftsman", 60, 3, 40},
	"farmhand":   {"farmer", 50, 3, 35},
	"apprentice": {"craftsman", 40, 3, 35},
	"servant":    {"clerk", 60, 4, 45},
	"porter":     {"craftsman", 60, 4, 40},
	"dockhand":   {"sailor", 40, 2, 30},
	"fisher":     {"fisher", 0, 0, 0},
	"craftsman":  {"shopkeeper", 160, 6, 50},
	"clerk":      {"scribe", 120, 5, 55},
	"acolyte":    {"priest", 60, 4, 45},
	"guard":      {"sergeant", 120, 5, 50},
	"sergeant":   {"captain", 300, 8, 55},
	"sailor":     {"mate", 120, 6, 50},
	"mate":       {"captain", 300, 8, 55},
	"scribe":     {"scholar", 200, 8, 55},
}

// lsIncome: coin earned per working day, by job (shops and posts pay more).
func lsIncome(job string) int {
	switch job {
	case "":
		return 0
	case "laborer", "farmhand", "servant", "porter", "apprentice", "dockhand", "fisher":
		return 1
	case "farmer", "craftsman", "clerk", "guard", "sailor", "acolyte", "scribe":
		return 2
	case "sergeant", "priest", "mate", "scholar", "shopkeeper":
		return 3
	case "captain", "noble", "lord", "lady", "governor", "magistrate", "merchant", "trader":
		return 5
	}
	if lsShopJobs[job] {
		return 3
	}
	return 2
}

func lsIsSoldier(job string) bool {
	j := strings.ToLower(job)
	for _, k := range []string{"guard", "soldier", "captain", "knight", "warden", "sergeant", "mercenary", "watch", "marshal", "militia"} {
		if strings.Contains(j, k) {
			return true
		}
	}
	return false
}

func lsIsTrader(job string) bool {
	switch strings.ToLower(job) {
	case "merchant", "trader", "caravaneer", "peddler", "minstrel", "pilgrim", "sailor":
		return true
	}
	return false
}

// lsGoals: machine ids and the line shown for each.
var lsGoalLines = map[string]string{
	"wealth":    "to put by %d coin",
	"love":      "to find someone to marry",
	"revenge":   "to avenge a wrong",
	"faith":     "to serve the faith: a shrine, an order",
	"knowledge": "to write a book of what they know",
	"fame":      "to be remembered in song",
	"justice":   "to bring a wrongdoer to the guards",
	"family":    "to raise a family",
	"safety":    "to find a safer place to live",
}

var lsGoalIDs = []string{"wealth", "love", "revenge", "faith", "knowledge", "fame", "justice", "family", "safety"}

// lsSecretsOut: how a secret comes out.
var lsSecretsOut = []string{
	"a servant talked", "a letter went astray", "a quarrel in the tavern", "a drunken boast",
	"a rival dug it up", "an old friend let it slip",
}

var lsDutyCauses = []string{"of a fever", "of a wasting sickness", "of the flux", "of a fall", "in their sleep"}
