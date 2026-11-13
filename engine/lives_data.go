package engine

// Tables for lives.go (prefix lv): trades by sector, personality, goals, fears, secrets and life-line phrasing.

// lvSectorTrades: trades for each Realmforge sector name (WorldData.SectorNames), most common first.
var lvSectorTrades = map[string][]string{
	"farming":     {"farmer", "farmer", "herder", "miller", "carter", "farmer"},
	"industry":    {"smith", "tanner", "carpenter", "mason", "smith"},
	"mining":      {"miner", "smelter", "miner"},
	"trade":       {"merchant", "carter", "innkeeper", "clerk", "baker", "merchant"},
	"crafts":      {"weaver", "potter", "cobbler", "baker", "carpenter", "tailor"},
	"scholarship": {"scribe", "tutor", "scribe"},
	"military":    {"guard", "guard", "sergeant"},
	"faith":       {"priest", "acolyte", "healer"},
	"seafaring":   {"sailor", "fisher", "sailor", "ferryman"},
}

// lvCommonTrades fill out any town.
var lvCommonTrades = []string{"baker", "innkeeper", "healer", "beggar", "farmer", "carter", "guard", "scribe", "priest", "smith"}

// lvTradeWealth: rough coin by trade.
var lvTradeWealth = map[string]int{
	"merchant": 260, "innkeeper": 150, "smith": 130, "baker": 80, "healer": 100, "priest": 70, "scribe": 70, "clerk": 60, "tutor": 70,
	"guard": 60, "sergeant": 90, "farmer": 40, "herder": 35, "miller": 70, "carter": 50, "tanner": 60, "carpenter": 70, "mason": 70,
	"miner": 35, "smelter": 60, "weaver": 55, "potter": 50, "cobbler": 45, "tailor": 90, "acolyte": 20, "sailor": 45, "fisher": 35,
	"ferryman": 70, "beggar": 3, "elder": 160, "child": 0,
}

// lvTradeActorRole: the makeNPC role to spawn for a trade.
var lvTradeActorRole = map[string]string{
	"guard": "guard", "sergeant": "guard", "smith": "smith", "merchant": "merchant", "healer": "apothecary", "priest": "priest",
	"acolyte": "priest", "ferryman": "ferry", "tailor": "tailor",
}

// lvActorTrades: which person trades an existing actor of this role may be bound to.
var lvActorTrades = map[string][]string{
	"guard": {"guard", "sergeant"}, "smith": {"smith"}, "merchant": {"merchant", "innkeeper", "clerk"}, "apothecary": {"healer"},
	"priest": {"priest", "acolyte"}, "ferry": {"ferryman", "sailor"}, "tailor": {"tailor", "weaver"}, "bookseller": {"scribe", "tutor"},
}

// lvCivilian: trades a plain "villager" actor may take.
var lvCivilian = map[string]bool{
	"farmer": true, "herder": true, "miller": true, "carter": true, "tanner": true, "carpenter": true, "mason": true, "miner": true,
	"smelter": true, "weaver": true, "potter": true, "cobbler": true, "baker": true, "innkeeper": true, "clerk": true, "scribe": true,
	"tutor": true, "sailor": true, "fisher": true, "beggar": true, "acolyte": true,
}

// lvTraitWords: Realmforge trait word (lowercase) -> shifts of [openness, conscientiousness, extraversion, agreeableness, neuroticism].
var lvTraitWords = map[string][5]int{
	"brave": {5, 5, 10, 0, -25}, "craven": {-5, -10, -10, 0, 30}, "generous": {5, 0, 10, 25, -5}, "greedy": {0, 5, 0, -25, 10},
	"honorable": {-5, 20, 0, 15, -5}, "just": {5, 15, 0, 10, -5}, "cruel": {0, 0, 0, -35, 10}, "charming": {10, 0, 25, 15, -10},
	"ambitious": {10, 20, 10, -10, 10}, "pious": {-15, 15, 0, 10, 0}, "content": {-5, 0, 0, 10, -25}, "paranoid": {-10, 5, -15, -20, 35},
	"lucky": {5, -5, 10, 5, -10}, "scholar": {30, 15, -15, 0, 0}, "strong": {0, 10, 10, 0, -10}, "cynical": {5, 0, -10, -25, 10},
	"sickly": {0, 0, -10, 0, 25}, "prophet": {20, 5, 5, 0, 15}, "visionary": {35, 0, 5, 0, 10}, "zealot": {-25, 15, 5, -15, 15},
	"zealous": {-20, 15, 5, -10, 10}, "creative": {35, 0, 5, 5, 5}, "clever": {25, 10, 0, -5, 0}, "hero": {5, 10, 15, 10, -15},
	"champion": {0, 15, 15, 0, -10}, "wise": {20, 10, 0, 10, -15}, "lustful": {5, -10, 15, -5, 5}, "lazy": {0, -30, -5, 0, 0},
	"diligent": {0, 30, 0, 5, 0}, "kind": {5, 0, 5, 30, -5}, "proud": {0, 5, 10, -15, 5}, "humble": {0, 5, -10, 15, -5},
	"wrathful": {0, -10, 5, -25, 25}, "patient": {0, 15, -5, 10, -20},
}

// lvTraitGoals: a few goal ids a notable trait leans toward.
var lvTraitGoals = map[string][]string{
	"greedy": {"wealth"}, "ambitious": {"fame", "wealth"}, "pious": {"faith"}, "zealot": {"faith"}, "zealous": {"faith"}, "prophet": {"faith"},
	"scholar": {"knowledge"}, "visionary": {"knowledge", "fame"}, "just": {"justice"}, "honorable": {"justice"}, "cruel": {"revenge"},
	"wrathful": {"revenge"}, "charming": {"love", "fame"}, "lustful": {"love"}, "paranoid": {"safety"}, "craven": {"safety"}, "kind": {"family"},
	"generous": {"family"}, "hero": {"fame"}, "champion": {"fame"}, "creative": {"knowledge", "fame"},
}

// lvRoleGoals: goal ids by Realmforge role or trade.
var lvRoleGoals = map[string][]string{
	"ruler": {"wealth", "fame", "safety"}, "general": {"fame", "justice"}, "hero": {"fame", "justice"}, "heir": {"fame", "safety", "love"},
	"prophet": {"faith"}, "priest": {"faith"}, "artist": {"fame", "love"}, "thinker": {"knowledge"}, "magician": {"knowledge"}, "inventor": {"knowledge", "fame"},
	"merchant": {"wealth"}, "innkeeper": {"wealth", "family"}, "smith": {"wealth", "fame"}, "baker": {"family", "wealth"}, "healer": {"knowledge", "faith"},
	"scribe": {"knowledge"}, "clerk": {"wealth", "safety"}, "tutor": {"knowledge"}, "guard": {"justice", "safety"}, "sergeant": {"justice", "fame"},
	"farmer": {"family", "safety"}, "herder": {"safety", "family"}, "sailor": {"wealth", "love"}, "fisher": {"safety", "family"}, "beggar": {"safety", "revenge"},
	"acolyte": {"faith", "knowledge"}, "carter": {"wealth", "love"}, "miner": {"wealth", "safety"},
}

var lvAllGoals = []string{"wealth", "love", "revenge", "faith", "knowledge", "fame", "justice", "family", "safety"}

// lvGoalText: one-line goal text; %s is a name or place where noted by lvGoalNeeds.
var lvGoalText = map[string][]string{
	"wealth":    {"save enough coin to buy a better place in the world", "get out of debt and then get rich", "build a business that outlives them", "put away a hoard against bad years"},
	"love":      {"win the heart of %s", "find someone to share the years with", "marry for love, not for the house", "see %s notice them at last"},
	"revenge":   {"see %s pay for an old wrong", "make the one who ruined their family suffer", "settle a score that has festered for years"},
	"faith":     {"make a pilgrimage and be forgiven", "serve the faith with a clean heart", "see the true faith honoured in this town", "earn a place among the devout"},
	"knowledge": {"learn what is in the old books", "master the craft until no one can teach them more", "find out what really happened in the old days", "see a far country with their own eyes"},
	"fame":      {"be remembered after they are gone", "make a name that is spoken in other towns", "win the town's respect", "do one deed worth a song"},
	"justice":   {"see the guilty answer for their crimes", "make the town fairer for the poor", "clear a name that was wrongly blackened", "see the law kept, whoever breaks it"},
	"family":    {"keep the family fed and together", "see the children well settled", "bring a scattered family home", "leave something to the children"},
	"safety":    {"stay out of trouble and keep their head down", "find a safe place behind strong walls", "never go hungry again", "keep clear of the powerful"},
}

var lvGoalNeedsName = map[string]bool{"love": true, "revenge": true}

var lvFearsGeneric = []string{
	"the dark", "fire", "dying poor", "being forgotten", "the sea", "the old graves", "fever", "the tax collector", "being alone", "a war coming back",
	"falling from a height", "what the gods may think", "wolves on the road", "losing their sight", "being shamed in public", "a hard winter",
}
var lvFearByGoal = map[string][]string{
	"wealth": {"dying poor", "the tax collector", "being cheated"}, "love": {"being alone", "being refused", "growing old unloved"},
	"revenge": {"dying before it is done", "being found out"}, "faith": {"the wrath of the gods", "dying unforgiven"},
	"knowledge": {"dying ignorant", "the burning of books"}, "fame": {"being forgotten", "being laughed at"},
	"justice": {"the guilty walking free", "a corrupt court"}, "family": {"losing a child", "a hard winter", "fever"},
	"safety": {"war coming back", "wolves on the road", "the knock at the door"},
}

// lvSecrets: "Text" may contain %s (a person's name chosen at creation). Kind drives the life event.
type lvSecretDef struct {
	Kind string
	Text string
	Name bool // %s is a person's name
}

var lvSecrets = []lvSecretDef{
	{"theft", "Once stole from the temple poor-box and has never confessed it.", false},
	{"theft", "Took coin from a dead neighbour's purse and kept quiet about it.", false},
	{"affair", "Has long been carrying on in secret with %s.", true},
	{"affair", "Was once more than friends with %s, and wishes it were forgotten.", true},
	{"heresy", "Still prays in the old way in secret, against the town's faith.", false},
	{"heresy", "Does not believe a word of the faith they preach, and never has.", false},
	{"debt", "Owes a great sum to a moneylender in another town and is quietly selling things to pay it.", false},
	{"debt", "Gambled away the family savings and has told no one.", false},
	{"birth", "Is not who their papers say: a bastard of a noble house, raised under another name.", false},
	{"birth", "Was a foundling, and knows who left them at the door.", false},
	{"crime", "Fled another town years ago after a killing, and uses a borrowed name.", false},
	{"crime", "Helped hide a smuggler's cargo once, and has been paid to look away ever since.", false},
	{"crime", "Set a fire that burned a rival's shop, long ago, and nobody was ever blamed.", false},
	{"shame", "Ran from a battle and left a friend behind to die.", false},
	{"gift", "Keeps a small hoard of old coin buried under the hearth.", false},
}

// lvPersonality words per trait: [low, high].
var lvTraitWordsLowHigh = [5][2]string{
	{"set in their ways", "curious"},
	{"careless", "diligent"},
	{"reserved", "outgoing"},
	{"gruff", "kind"},
	{"calm", "anxious"},
}

// lvLifeFirst: apprenticeship phrasing.
var lvApprentice = []string{
	"Began as an apprentice %s, learning the work from %s.",
	"Went to learn the trade of %s under %s.",
	"Took up the work of a %s, guided by %s.",
}

var lvLossLines = []string{
	"Lost a parent to fever.", "Buried a brother after a bad winter.", "Lost the family home in a fire and began again.",
	"A hard harvest nearly ruined the household.", "Lived through a plague year; many neighbours did not.", "Was robbed on the road and lost a season's earnings.",
	"Was ill for most of a year and nearly died.", "Lost a dear friend in a brawl at the tavern.",
}

var lvWorkLines = []string{
	"Took over the work from an ageing master.", "Moved to a bigger workshop and did well for a time.", "Won a small honour from the town for good work.",
	"Was cheated by a partner and had to start over.", "Became known on the street for honest dealing.", "Was fined for a quarrel and still resents it.",
}

var lvMoveLines = []string{"Moved to %s from %s, looking for work.", "Came to %s from %s after a bad year.", "Followed family from %s to %s."}

var lvMoodWords = [][2]interface{}{{-60, "grim"}, {-25, "low"}, {-5, "uneasy"}, {15, "steady"}, {45, "cheerful"}, {101, "elated"}}

var lvOpinionWords = [][2]interface{}{{-75, "hates you"}, {-40, "despises you"}, {-15, "distrusts you"}, {15, "neutral"}, {40, "likes you"}, {75, "trusts you"}, {101, "adores you"}}

var lvGreetWarm = []string{
	"\"%s! Good to see you. Sit, what can I do for you?\"", "\"Ah, my friend. The town speaks well of you, and so do I.\"",
	"\"There you are. I hoped you would come by again.\"",
}
var lvGreetFriendly = []string{"\"Back again, wanderer? Welcome.\"", "\"Ah, I remember you. Speak freely.\"", "\"Good day. Your face is a welcome one.\""}
var lvGreetCold = []string{"\"You. What do you want this time?\"", "\"I have not forgotten you. Be quick.\"", "\"Make it short, wanderer.\""}
var lvGreetHostile = []string{"\"Keep your distance from me.\" They do not meet your eyes.", "\"You have a nerve, coming near me.\"", "\"I know what you are. Say your piece and go.\""}
