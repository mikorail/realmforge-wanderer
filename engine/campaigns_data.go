package engine

// Campaign data (prefix cmp): plot templates and faction questline archetypes. Text uses placeholders
// {boss} {ally} {wit} {vic} (cast), {town} (home), {town2} (second town), {site} (dungeon), {realm}, {ruler}.

// cmpClueT is one clue slot: K = talk rumour doc book dungeon; W = cast role giving it (talk); L = A or B (town);
// N = topic or item name; T = what you learn; C/D = optional skill check and difficulty.
type cmpClueT struct {
	K, W, L, N, T, C string
	D                int
}

type cmpActT struct {
	Title, Brief string
	Clues        []cmpClueT
}

type cmpTpl struct {
	ID, Title, Hook string
	Roles           [4]string // what boss, ally, wit, vic are in this story
	Pref            []string  // notable roles preferred for the boss
	VicPref         []string
	Acts            [5]cmpActT
	End             map[string]string // expose join blackmail kill walk
}

func cmpT(who, topic, text, chk string, dc int) cmpClueT {
	return cmpClueT{K: "talk", W: who, N: topic, T: text, C: chk, D: dc}
}
func cmpR(loc, text string) cmpClueT {
	return cmpClueT{K: "rumour", L: loc, N: "the talk in the streets", T: text}
}
func cmpD(loc, name, text string) cmpClueT {
	return cmpClueT{K: "doc", L: loc, N: name, T: text}
}
func cmpB(loc, name, text string) cmpClueT {
	return cmpClueT{K: "book", L: loc, N: name, T: text}
}
func cmpX(name, text string) cmpClueT { return cmpClueT{K: "dungeon", L: "S", N: name, T: text} }

var cmpTpls = []*cmpTpl{
	{
		ID: "throne", Title: "A Crown in Shadow", Pref: []string{"general", "ruler", "hero"}, VicPref: []string{"heir", "ruler"},
		Roles: [4]string{"the conspirator", "the fixer", "the nervous courtier", "the rightful claimant"},
		Hook:  "Whispers in {town}: {boss} keeps strange company, and the night guard has been changed twice this season.",
		Acts: [5]cmpActT{
			{"Whispers", "Rumour says {boss} is reaching for something that is not theirs. Find who is willing to talk.", []cmpClueT{
				cmpR("A", "Servants say {boss} burned letters at midnight and doubled the watch at the postern."),
				cmpT("wit", "the night at court", "{wit} admits, after a cup, that {boss} asked for the roll of every house loyal to the old line.", "", 0)}},
			{"The paper trail", "Money moves where {boss} wants it. Follow it.", []cmpClueT{
				cmpD("A", "torn ledger leaf", "Coin from {boss}'s own purse to captains on the {realm} border, none of it for wages."),
				cmpT("ally", "the guard rota", "{ally} lets slip that the household guard will be changed on a night already chosen.", "streetwise", 8)}},
			{"The hidden cache", "Something was hidden in the dark, away from honest eyes.", []cmpClueT{
				cmpX("sealed packet", "A packet from {site}: names of nobles who have sworn to {boss}, each with a seal-print."),
				cmpB("B", "tutor's diary", "{vic}'s old tutor wrote that {boss} twice asked who stands next in the line of succession.")}},
			{"Pressure", "The plot has a victim. Learn what has been done to them.", []cmpClueT{
				cmpT("vic", "the rightful line", "{vic} confides that two meals were poisoned and the cook who tasted them has vanished.", "empathy", 9),
				cmpD("B", "forged writ", "A writ of regency in {boss}'s favour, the ink still fresh and the seal a copy.")}},
			{"The proof", "You need something no one can argue with.", []cmpClueT{
				cmpX("signed compact", "The signed compact between {boss} and the border captains, sealed away in {site}. With it, {boss} can be broken, bought or struck down.")}},
		},
		End: map[string]string{
			"expose":    "You lay the compact before the {realm} council. {boss} is taken in chains and the succession holds. Nobles you never met send their thanks.",
			"join":      "{boss} reads the compact and smiles. \"A clever one. There is room beside me.\" Within the season the throne changes hands without a sword drawn.",
			"blackmail": "{boss} pays, white to the lips, and keeps paying. The plot sleeps, but you know where it sleeps.",
			"kill":      "{boss} falls with the compact still in reach. The border captains drift home, and the conspiracy dies with its head.",
			"walk":      "You leave the compact where it lies. In time the {realm} learns what sleeping on a throne can cost.",
		},
	},
	{
		ID: "heir", Title: "The Lost Heir", Pref: []string{"ruler", "general", "hero"}, VicPref: []string{"heir", "hero"},
		Roles: [4]string{"the usurper", "the foster-keeper", "the old midwife", "the lost heir"},
		Hook:  "Whispers in {town}: an old nurse swears the line of {boss} is not the oldest, and that a child was sent away.",
		Acts: [5]cmpActT{
			{"A child sent away", "A child vanished from the records long ago. Someone remembers.", []cmpClueT{
				cmpR("A", "Folk of the lower town say a cradle left the hall by night, years back, and the nurse was dismissed without pay."),
				cmpT("wit", "the lying-in", "{wit} remembers the birth: a second child, a hurried bundle, a purse pressed into the wrong hands.", "empathy", 8)}},
			{"The foster road", "Where the child went, and with whom.", []cmpClueT{
				cmpD("B", "parish register", "A register page with a name scraped out and the date of a christening in {town2}."),
				cmpT("ally", "the foster home", "{ally} was paid for years to raise a child and ask nothing. Their silence has been wearing thin.", "persuasion", 9)}},
			{"The token", "Proof that can be held in the hand.", []cmpClueT{
				cmpX("token of the house", "The signet of the old line, hidden in {site}, with a note: for the child, when the child is grown."),
				cmpB("B", "foster's letter-book", "Letters in a careful hand: money received from {boss}'s steward, and the promise to 'keep the matter quiet until the end'.")}},
			{"The living heir", "The heir may be closer than anyone supposed.", []cmpClueT{
				cmpT("vic", "who you really are", "{vic} has long felt out of place and half-guessed why. Shown the signet, they go very quiet.", "empathy", 9),
				cmpD("A", "steward's receipt", "A receipt in {boss}'s steward's hand for 'the boy's keep, final payment, no further correspondence'.")}},
			{"The proof", "Proof that no court can brush off.", []cmpClueT{
				cmpX("deed of birth", "A witnessed deed of birth in {site}, naming {vic} as the elder child of the old line.")}},
		},
		End: map[string]string{
			"expose":    "You put the deed before the court. {vic} is acknowledged, {boss} steps aside, and the line of {town} is mended at last.",
			"join":      "You show {boss} the deed and name your price. The heir is quietly shipped off again, and your purse is heavy.",
			"blackmail": "{boss} pays for your silence, and you are paid again each season. {vic} never learns why the money flows.",
			"kill":      "{boss} falls, and in the confusion the claim of {vic} goes unchallenged. You are not thanked in public.",
			"walk":      "You leave the deed in the dark. {vic} lives and dies an ordinary life, and {boss} never knows how close it came.",
		},
	},
	{
		ID: "curse", Title: "The Old Sin", Pref: []string{"prophet", "priest", "ruler", "general"}, VicPref: []string{"hero", "priest", "artist"},
		Roles: [4]string{"the heir to the sin", "the keeper of the shrine", "the village elder", "the wronged line"},
		Hook:  "Whispers in {town}: wells turn sour and cattle drop in the fields; the old ones say it is no blight, but a debt.",
		Acts: [5]cmpActT{
			{"Sour wells", "Something is wrong in the land and the old folk say they know what. Ask.", []cmpClueT{
				cmpR("A", "Every sour well in {town} lies along the same line, and the oldest women say they have seen this before, in their grandmothers' days."),
				cmpT("wit", "the old sin", "{wit} tells it grudgingly: a family once sworn to protect a place was paid to look away from a slaughter, and {boss}'s house grew rich on it.", "empathy", 8)}},
			{"The old record", "Dig out what was written down, and what was torn out.", []cmpClueT{
				cmpB("B", "chronicle fragment", "A chronicle fragment: the oath, the breaking of it, and the curse laid at the grave by the last of the wronged."),
				cmpT("ally", "the shrine rolls", "{ally} keeps the shrine rolls and admits pages were cut out in {boss}'s grandfather's time.", "persuasion", 9)}},
			{"The place of the wrong", "The sin has a place. Go there.", []cmpClueT{
				cmpX("oathstone shard", "A shard of the oathstone in {site}, cracked through, with the names of those who swore and those who sold them out."),
				cmpD("A", "grandsire's will", "A will leaving the shrine lands to {boss}'s family with the line: 'and may none ask whence the coin came'.")}},
			{"The wronged", "The wronged have descendants. Find them.", []cmpClueT{
				cmpT("vic", "their grandmother's story", "{vic} has an unbroken family tale of the betrayal, and the names match what you found.", "empathy", 9),
				cmpD("B", "grave-tally", "A mason's tally of a mass grave outside {town2}, unmarked in any official record.")}},
			{"The reckoning", "To end the curse, the sin must be spoken aloud, or paid for in another coin.", []cmpClueT{
				cmpX("confession scroll", "A scroll in {site} that must be read at the grave by someone of the guilty house: the way the sin can be named and set down.")}},
		},
		End: map[string]string{
			"expose":    "You read the sin aloud before {town}. {boss} stands in the square and does not deny it. The wells run clear within a month.",
			"join":      "{boss} would rather the sin stay buried. You help bury it, and are paid for it. The wells stay sour, and the paying goes on.",
			"blackmail": "{boss} pays you to keep the sin quiet and the wells go on souring. You tell yourself it is only business.",
			"kill":      "{boss} dies on the old grave. Some say the curse is paid. The wells have still not cleared.",
			"walk":      "You leave the matter to the dead. Cattle go on dying, and in a few years {town} thins out.",
		},
	},
	{
		ID: "vault", Title: "The Vault Job", Pref: []string{"hero", "merchant", "general", "ruler"}, VicPref: []string{"ruler", "merchant", "priest"},
		Roles: [4]string{"the mastermind", "the safecracker", "the old keeper", "the owner of the vault"},
		Hook:  "Whispers in {town}: a crew is hiring in the back rooms, and one of them has been asking about an old strongroom.",
		Acts: [5]cmpActT{
			{"The crew", "A crew is forming. Find who is paying.", []cmpClueT{
				cmpR("A", "A dozen idle hands have lately gone rich and silent in the same week; the tavern talk says {boss} is behind it."),
				cmpT("ally", "the work", "{ally} would not say whose job it is, but let slip it needs a lock-wright and a patient man.", "streetwise", 8)}},
			{"The vault", "Learn what is being robbed, and who owns it.", []cmpClueT{
				cmpB("B", "vault inventory", "An inventory of {vic}'s old strongroom: reliquaries, a charter box, and 'one item not to be opened'."),
				cmpT("wit", "the strongroom doors", "{wit} guarded the doors for years and knows the false step, the false wall, and the day the watch changes.", "persuasion", 9)}},
			{"The plan", "Plans on paper leave marks.", []cmpClueT{
				cmpD("A", "chalk plan", "A plan chalked on a table's underside: the vault wall, the watch's hours, and a line labelled 'the way out'."),
				cmpX("tool roll", "A roll of picks and a hand-drawn map left in {site}, the secondary way into the vault.")}},
			{"The owner", "Does the owner know?", []cmpClueT{
				cmpT("vic", "the strongroom's contents", "{vic} is calm and says nothing in the strongroom is worth the trouble; their eyes say otherwise.", "empathy", 9),
				cmpD("B", "insurer's note", "A note from a lender: the vault's contents are pledged against a debt that cannot be paid. A robbery would clear it.")}},
			{"The proof", "You need the mastermind's own hand.", []cmpClueT{
				cmpX("mastermind's strongbox", "A strongbox in {site} holding the crew's pay-sheet in {boss}'s hand: the date, the take, and the shares.")}},
		},
		End: map[string]string{
			"expose":    "You warn {vic}, and the watch takes the crew on the night. {boss} is named, fined and finished in {town}. {vic} sends a reward and a hard look.",
			"join":      "You take a share and turn a blind eye. On the night, the strongroom opens like a flower and closes empty.",
			"blackmail": "{boss} would rather pay than hang. The crew never learns why the job was called off.",
			"kill":      "{boss} falls before the night. The crew scatters. {vic} never knows what was saved.",
			"walk":      "You let the job run. In the morning the strongroom is empty and {town} is poorer.",
		},
	},
	{
		ID: "plague", Title: "The Quiet Sickness", Pref: []string{"ruler", "priest", "general"}, VicPref: []string{"priest", "hero", "artist"},
		Roles: [4]string{"the official hiding it", "the bought physician", "the gravedigger", "the quarantined"},
		Hook:  "Whispers in {town}: the graveyard is busier than the registers say, and a street has been bricked up 'for repairs'.",
		Acts: [5]cmpActT{
			{"The bricked street", "A street is closed and no one will say why.", []cmpClueT{
				cmpR("A", "People in {town} say the bricked street holds a sickness, and that folk have been taken there by night."),
				cmpT("wit", "the extra graves", "{wit} has dug twice the registered graves this season and was told to keep their mouth shut.", "persuasion", 8)}},
			{"The lie", "Who is lying, and to whom?", []cmpClueT{
				cmpD("A", "burial register", "The burial register, scrubbed and rewritten: dozens of 'age' and 'misadventure' entries over the same few weeks."),
				cmpT("ally", "the sickroom", "{ally}, a physician, has been paid to call it something else; they are half-glad to say so.", "empathy", 9)}},
			{"The source", "Where did the sickness come from?", []cmpClueT{
				cmpX("sampler's case", "A sealed case of samples in {site}, and a note naming the cargo that carried the sickness into {town}."),
				cmpB("B", "physician's casebook", "The casebook shows the same sickness in {town2} a season before. Someone chose not to warn the other towns.")}},
			{"The quarantined", "Those behind the bricks are people.", []cmpClueT{
				cmpT("vic", "the bricked street", "{vic} was shut in with the sick and passed a message through the wall; they are begging for food and truth.", "empathy", 9),
				cmpD("B", "order of silence", "A written order to the watch to keep the street shut, signed by {boss}.")}},
			{"The proof", "A written order is not enough. Find the whole.", []cmpClueT{
				cmpX("ledger of the dead", "A secret ledger in {site}: every death, every payment, and the plan to burn the street if it spread.")}},
		},
		End: map[string]string{
			"expose":    "You make the ledger public. The bricks come down, the sick are tended, and {boss} is called to answer. Many live who would have burned.",
			"join":      "You take {boss}'s coin and say nothing. The street burns in the end, and the register records another misadventure.",
			"blackmail": "{boss} pays for your silence and quietly opens the street. Some live. Some do not.",
			"kill":      "{boss} falls. In the chaos the bricks are torn down by the people themselves. It is not neat, but the sick are fed.",
			"walk":      "You walk away. The street stays shut a little longer, and the cost is not written in any register.",
		},
	},
	{
		ID: "smuggling", Title: "The Night Trade", Pref: []string{"general", "hero", "merchant", "ruler"}, VicPref: []string{"merchant", "hero"},
		Roles: [4]string{"the ring's master", "the harbour fixer", "the porter", "the honest inspector"},
		Hook:  "Whispers in {town}: crates leave the town by night with no tally, and the inspector who counted them is no longer at his post.",
		Acts: [5]cmpActT{
			{"The empty tallies", "Goods vanish between the dock and the book.", []cmpClueT{
				cmpR("A", "Honest traders in {town} swear the tallies are cooked: more goes out than comes in, and the watch looks away."),
				cmpT("wit", "the night shift", "{wit} hauls crates at an hour no tally covers, and is paid triple to not ask what is in them.", "streetwise", 8)}},
			{"The marks", "Contraband leaves marks.", []cmpClueT{
				cmpD("A", "chalked manifest", "A manifest scratched on the back of a crate lid: weights, a false origin, and a price in code."),
				cmpT("ally", "the watch hours", "{ally} sells the watch hours and the cart routes; they will say so only to someone who looks worth it.", "persuasion", 9)}},
			{"The cache", "Contraband is kept somewhere.", []cmpClueT{
				cmpX("cache ledger", "A cache in {site} and a ledger of what has passed through it, and what was paid to whom."),
				cmpB("B", "buyer's order book", "The order book of a buyer in {town2}: the same goods, the same marks, a standing monthly order.")}},
			{"The vanished inspector", "The honest man knew too much.", []cmpClueT{
				cmpT("vic", "what they counted", "{vic} was bought, then threatened, then asked to look away; their notes are hidden somewhere only they know.", "empathy", 9),
				cmpD("B", "inspector's note", "A note of the inspector's, pinned beneath a floorboard: the dates, the crates, and {boss}'s name in full.")}},
			{"The proof", "The master's own hand.", []cmpClueT{
				cmpX("master's seal-book", "The master's own seal-book in {site}: every shipment, every bribe, every town's cut.")}},
		},
		End: map[string]string{
			"expose":    "You put the seal-book before the magistrates. {boss} loses the trade and the town; the watch has a bad month.",
			"join":      "You take a seat at the table. Within a season the trade grows a second route and a quieter conscience.",
			"blackmail": "{boss} pays well to keep the seal-book unread. You do not ask where the money was made.",
			"kill":      "{boss} falls. The ring splits in a week and its pieces fight over the pieces.",
			"walk":      "You let the trade run. The tallies stay cooked and the inspector's post stays empty.",
		},
	},
	{
		ID: "cult", Title: "The Rising Faith", Pref: []string{"prophet", "priest", "hero"}, VicPref: []string{"priest", "artist", "hero"},
		Roles: [4]string{"the prophet", "the first acolyte", "the apostate", "the missing devotee"},
		Hook:  "Whispers in {town}: a new faith is taking root in cellars, and some who attended no longer come home.",
		Acts: [5]cmpActT{
			{"The cellar meetings", "A new faith recruits by night.", []cmpClueT{
				cmpR("A", "People in {town} say {boss} preaches in cellars and that the congregation grows every week, and shrinks every month."),
				cmpT("wit", "why they left", "{wit} left the circle after the second rite and will not say what was asked of the faithful.", "empathy", 8)}},
			{"The rites", "Learn what is promised, and what is asked.", []cmpClueT{
				cmpB("B", "tract of the faithful", "A tract: the promise of a new age and the price of admission, paid in coin, in kin, and in silence."),
				cmpT("ally", "the first rite", "{ally} is a true believer but not a hard one; shown the tract, they admit the rites are growing worse.", "persuasion", 9)}},
			{"The sanctum", "Every cult has somewhere it keeps what it cannot show.", []cmpClueT{
				cmpX("sanctum scrap", "A sanctum in {site}: robes, a bloodied altar, and a list of those 'taken into the new age'."),
				cmpD("A", "devotee's farewell", "A farewell note, pushed under a door: 'I go to be remade; do not look for me.'")}},
			{"The missing", "Someone is missing, and may still be found.", []cmpClueT{
				cmpT("vic", "what they saw", "{vic} was taken to the sanctum and let go as a warning. They will say what they saw only to someone gentle.", "empathy", 9),
				cmpD("B", "ritual calendar", "A calendar with a date ringed: the night of the great rite, when the faithful 'will be gathered'.")}},
			{"The proof", "The prophet's own words.", []cmpClueT{
				cmpX("prophet's book", "The prophet's own book in {site}, with the true doctrine, the true aims, and the plan for the great rite.")}},
		},
		End: map[string]string{
			"expose":    "You read the prophet's book aloud in {town}. The faithful leave in dozens. {boss} flees, and the cellars go quiet.",
			"join":      "You take the third rite. The faith has a new voice in {town}, and the great rite goes ahead.",
			"blackmail": "{boss} pays and moves the rites to another town. The cult is not ended, only turned away.",
			"kill":      "{boss} dies in the sanctum. The faithful scatter, some lost, some free. A few call you a martyr-maker.",
			"walk":      "You leave the cult to its rites. By the next season, the great rite is held and {town} is not the same.",
		},
	},
}

// ---- ladder archetypes ----

// cmpLadPool lists the middle step types for each order kind (and sect kind), in preference order.
var cmpLadPool = map[string][]string{
	"holy":     {"slay", "relic", "pilgrim", "tithe", "tasks"},
	"monastic": {"pilgrim", "relic", "tithe", "tasks", "slay"},
	"cult":     {"slay", "tithe", "relic", "tasks", "pilgrim"},
	"secret":   {"relic", "tasks", "slay", "tithe", "pilgrim"},
	"thieves":  {"tithe", "tasks", "relic", "slay", "pilgrim"},
	"merchant": {"tithe", "tasks", "pilgrim", "relic", "slay"},
	"scholars": {"relic", "pilgrim", "tasks", "tithe", "slay"},
	"sect":     {"slay", "pilgrim", "relic", "tasks", "tithe"},
	"clan":     {"slay", "tasks", "pilgrim", "relic", "tithe"},
	"guild":    {"tithe", "tasks", "pilgrim", "relic", "slay"},
	"":         {"slay", "relic", "pilgrim", "tithe", "tasks"},
}

var cmpLadFinal = map[string]string{
	"holy": "rank", "monastic": "rank", "cult": "relic", "secret": "rank", "thieves": "tasks", "merchant": "rank", "scholars": "rank",
	"sect": "rank", "clan": "rank", "guild": "rank", "cult2": "relic",
}

var cmpLadVerbs = map[string][]string{
	"slay":  {"Cut down %d enemies of %s", "Put %d foes of %s to the sword", "Drive off or destroy %d enemies of %s"},
	"tasks": {"Carry out %d commissions for %s", "Fulfil %d tasks in the service of %s", "Do %d errands the leadership will name"},
}

var cmpLadFoot = []string{
	"\"A seat at the table is not given, it is earned. Go and earn it.\"",
	"\"Words are cheap here. Show me hands that have worked.\"",
	"\"Everyone says they are loyal. Prove it.\"",
}

var cmpLadReport = []string{
	"\"You did what was asked, and more than one would have done. Take this.\"",
	"\"Good. The leadership has noted it.\"",
	"\"You have a name here now. Do not waste it.\"",
	"\"This is how a society grows: one proven hand at a time.\"",
}
