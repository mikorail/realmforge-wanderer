package engine

// Family & Fame data (prefix fm): title registry, courtship and duel text, song flourishes.
// Logic lives in family.go.

// fmTitle is a reputation title. Style "pre" reads "Dragonbane Aldo", "post" reads "Aldo the Merciful".
// Tag is the trait tag (traits.go thRoleValues) NPCs like or dislike; Mod is the price shift in percent by
// NPC role ("*" is the default), negative = cheaper.
type fmTitle struct {
	ID, Name, Style, Tag, Desc, Perk string
	Mod                              map[string]int
	Cond                             func(g *Game, s *fmState) bool
}

func fmDeeds(s *fmState, k string) int { return s.DeedCt[k] }

var fmTitles = []fmTitle{
	{"dragonbane", "Dragonbane", "pre", "valor", "Slay a legendary or world-scale monster.", "guards and smiths give fairer prices",
		map[string]int{"guard": -6, "smith": -5, "*": -1}, func(g *Game, s *fmState) bool { return s.Big >= 1 }},
	{"penniless", "the Penniless", "post", "temperance", "Hold 400 and then end up with nothing.", "merchants haggle harder, priests pity you",
		map[string]int{"merchant": 5, "priest": -4, "villager": -2}, func(g *Game, s *fmState) bool { return s.Peak >= 400 && g.P.Gold <= 0 }},
	{"oathbreaker", "Oathbreaker", "pre", "blood", "Strike a yielded foe or break a sworn truce.", "nobody trusts you with credit",
		map[string]int{"priest": 12, "guard": 8, "*": 6}, func(g *Game, s *fmState) bool { return s.Oath >= 1 || fmDeeds(s, "broke the") >= 1 }},
	{"merciful", "the Merciful", "post", "mercy", "Spare three beaten foes.", "priests and healers favour you",
		map[string]int{"priest": -6, "apothecary": -5, "villager": -3}, func(g *Game, s *fmState) bool { return s.Spared+thSt(g).C["spared"] >= 3 }},
	{"bloodied", "the Bloodied", "post", "blood", "Kill a hundred creatures.", "villagers charge a frightened premium",
		map[string]int{"villager": 4, "merchant": 3, "guard": -2}, func(g *Game, s *fmState) bool { return thSt(g).C["kills"] >= 100 || g.Kills >= 100 }},
	{"wayfarer", "Wayfarer", "pre", "road", "Walk 2500 km.", "traders and ferrymen like a known traveller",
		map[string]int{"merchant": -4, "ferry": -6, "exotic": -3}, func(g *Game, s *fmState) bool { return s.Km >= 2500 }},
	{"peerless", "the Peerless", "post", "valor", "Reach fame 70 in any realm.", "everyone shaves a little off",
		map[string]int{"*": -3}, func(g *Game, s *fmState) bool { return s.fameMax() >= 70 || thFame(g) >= 70 }},
	{"kingmaker", "Kingmaker", "pre", "fellowship", "Back two or more causes at court.", "notables and elders deal gladly",
		map[string]int{"notable": -6, "elder": -4}, func(g *Game, s *fmState) bool { return fmDeeds(s, "kingmaker") >= 2 }},
	{"tombrobber", "Tomb-Robber", "pre", "heresy", "Rob a grave.", "priests and elders bristle; fences smile",
		map[string]int{"priest": 12, "elder": 6, "exotic": -5, "*": 2}, func(g *Game, s *fmState) bool { return fmDeeds(s, "robbed the grave") >= 1 }},
	{"riftwalker", "Rift-Walker", "pre", "road", "Set foot on three worlds.", "dealers in oddities respect you",
		map[string]int{"exotic": -6, "bookseller": -4}, func(g *Game, s *fmState) bool { return len(g.unRaw().Seen) >= 3 || g.unRaw().Trips >= 3 }},
	{"wormfeeder", "Wormfeeder", "pre", "dread", "Rise from the grave.", "the living keep their distance",
		map[string]int{"villager": 5, "priest": -3, "apothecary": -3}, func(g *Game, s *fmState) bool { return fmDeeds(s, "rose from the grave") >= 1 || s.Died >= 1 }},
	{"duelist", "the Duellist", "post", "valor", "Win five duels.", "guards and smiths salute you",
		map[string]int{"guard": -4, "smith": -3}, func(g *Game, s *fmState) bool { return s.DuelW >= 5 }},
	{"champion", "Tourney-Champion", "pre", "valor", "Win a city tournament.", "arena folk and nobles pay respect",
		map[string]int{"arena": -8, "notable": -4, "*": -2}, func(g *Game, s *fmState) bool { return s.Champs >= 1 }},
	{"songsung", "the Song-Sung", "post", "fellowship", "Have three songs of you sung in towns.", "bards charge you nothing",
		map[string]int{"bard": -10, "*": -2}, func(g *Game, s *fmState) bool { return s.sungCount() >= 3 }},
	{"hearth", "Hearth-Keeper", "pre", "fellowship", "Be wed with a child in the house.", "villagers trust a family person",
		map[string]int{"villager": -3, "merchant": -2}, func(g *Game, s *fmState) bool { return s.Spouse != nil && s.Spouse.Gone == "" && len(s.Kids) > 0 }},
}

var fmTitleByID = func() map[string]*fmTitle {
	m := map[string]*fmTitle{}
	for i := range fmTitles {
		m[fmTitles[i].ID] = &fmTitles[i]
	}
	return m
}()

// fmTitleName joins a title to a name.
func fmTitleName(t *fmTitle, name string) string {
	if t.Style == "pre" {
		return t.Name + " " + name
	}
	return name + " " + t.Name
}

// deed text fragments counted for titles (fmScanDeeds)
var fmDeedKeys = map[string]string{
	"broke the": "broke the", "robbed the grave": "robbed the grave", "rose from the grave": "rose from the grave",
	"threw in with": "kingmaker", "funded": "kingmaker", "laid a gift": "kingmaker",
}

// ---- courtship ----

var fmInterests = []string{"lore", "road", "faith", "valor", "fellowship", "temperance", "greed", "mercy"}

var fmInterestWord = map[string]string{"lore": "books and old stories", "road": "far roads", "faith": "the temple", "valor": "feats of arms",
	"fellowship": "good company", "temperance": "quiet living", "greed": "a fat purse", "mercy": "kindness to the weak", "": "nothing in particular"}

var fmSpendLines = map[string][]string{
	"lore":       {"You trade old stories until the lamp burns low.", "You read aloud together; they correct your pronunciation and smile."},
	"road":       {"You describe the roads you have walked; their eyes keep drifting to the horizon.", "You draw a map in the dust for them. They keep it."},
	"faith":      {"You sit through the evening prayers together.", "You light a lamp at the shrine beside them."},
	"valor":      {"You show them a sword grip. Their laugh is unexpectedly warm.", "You talk of fights won. They ask sharp questions."},
	"fellowship": {"You share a loaf and a long, easy conversation.", "You help with the evening chores, and the household watches approvingly."},
	"temperance": {"You walk the edge of the town at dusk, not talking, and it is enough.", "You share a plain meal and plainer thoughts."},
	"greed":      {"You talk prices and plans; they warm to a person who can count.", "You compare ledgers. It is oddly romantic."},
	"mercy":      {"You tend a stray together.", "You give coin to the beggar at the gate and they notice you notice nothing."},
	"":           {"You spend a pleasant hour together.", "You walk the market side by side."},
}

var fmCoolLines = []string{"They are polite, and a little distant today.", "Your timing is poor; they are busy and curt.", "They listen, but their mind is elsewhere."}

var fmRefuseLines = []string{"\"My family would never agree.\"", "\"There is someone else the household favours.\"", "\"Not yet. Come back when I know you better.\""}

var fmCasteMult = map[string]float64{} // reserved

// ---- children ----

var fmChildGifts = []string{"their parent's laugh", "a stubborn jaw", "an uncanny sense of direction", "a head for numbers", "steady hands", "a love of animals",
	"a singing voice", "a quick temper", "endless curiosity", "a calm that unsettles adults"}

var fmCallings = [6]string{"warrior", "scout", "guard", "scholar", "priest", "bard"}

var fmChildPlay = []string{"You play at swords with sticks until both of you are out of breath.", "You carry them on your shoulders through the market.",
	"You teach them a rhyme from the road; they get it wrong, magnificently.", "You sit with them while they sleep, and do not mind the time."}

// ---- duels ----

var fmRuleName = map[string]string{"blood": "to first blood", "yield": "to yield", "death": "to the death"}
var fmRuleSev = map[string]int{"blood": 1, "yield": 2, "death": 3}

var fmInsults = []string{
	"%s sends word that you slighted the honour of their house, and demands satisfaction.",
	"%s says your fame is a loan against their own, and challenges you.",
	"%s, whom you passed without a greeting, calls it an insult and names a time.",
	"%s has heard the songs and does not believe them. A duel will settle it.",
}

var fmJilted = "%s, who was courting the same hand, calls you out. \"You took what I was owed. Meet me in the square.\""

var fmYieldLines = []string{"\"Enough! I yield.\"", "\"You have the day. I yield.\"", "They drop their guard and raise an open hand."}

// ---- songs ----

var fmFlairs = [][]string{
	{""},
	{"They say the crowd fell silent.", "A child in the crowd, they say, never forgot it.", "It was a bright day for the telling."},
	{"The sky itself held its breath.", "Even the old soldiers wept.", "Not a witness blinked, if the bards may be believed."},
	{"The mountains bowed, and the rivers remembered the name.", "Gods leaned from the clouds to watch, and did not interfere.", "Kings have asked for the tune and been refused."},
}

var fmSongOpen = []string{"Gather round, for I sing of %s:", "A verse for the road, a verse for the wanderer %s:", "Hear the ballad of %s:", "Listen well, for this is %s's tale:"}

// fmMajorKind: deed kinds worth a song.
var fmMajorKind = map[string]bool{"slay": true, "relic": true, "quest": true, "nemesis": true, "rival": true, "endgame": true, "murim": true,
	"murder": true, "villain": true, "crime": true, "event": true, "kinship": true, "bounty": true, "companion": true}

// ---- tournament ----

var fmTournName = []string{"the Spring Lists", "the Summer Games", "the Harvest Tourney", "the Winter Challenge"}
