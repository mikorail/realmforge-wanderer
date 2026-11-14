package engine

// Narrative director data (prefix nd): genre vocabulary, arc templates and the story-beat lines. Placeholders:
// {foe} {friend} {town} {other} {realm} {enemy} {verse} {reason} {me}, and the genre words {Msgr}/{msgr}
// {letter} {fest} {agents} {omen} {mood}.

type ndVoc struct{ Msgr, Letter, Fest, Agents, Omen, Mood, Rumour string }

var ndVocs = map[string]ndVoc{
	"fantasy":    {"a mud-spattered rider", "sealed letter", "harvest fair", "hired blades", "a ring of crows", "uneasy", "tavern talk"},
	"mythic":     {"a winged herald", "oracle-sealed tablet", "feast of the gods", "the Furies' tools", "an owl on the left hand", "fate-heavy", "an oracle's whisper"},
	"steampunk":  {"a telegraph boy", "telegram", "grand exhibition", "hired brutes in goggles", "a plume of black steam", "anxious", "gossip over the boiler-house wall"},
	"dieselpunk": {"a dispatch rider", "coded dispatch", "victory dance", "gunmen in long coats", "a searchlight sweeping low", "jumpy", "wireless chatter"},
	"atompunk":   {"a courier in a silver suit", "punch-card message", "Atomic Fair", "men in grey suits", "static on every dial", "wary", "a diner rumour"},
	"cyberpunk":  {"a courier drone", "encrypted ping", "night market", "corp-paid mercs", "a dead-channel flicker", "wired", "a feed leak"},
	"biopunk":    {"a carrier-bird grown in a vat", "spore-sealed note", "grafting festival", "handlers with engineered hounds", "a sweetish rot on the wind", "queasy", "a murmur through the culture-beds"},
	"solarpunk":  {"a glider-post runner", "woven letter", "solstice commons feast", "saboteurs", "a dimming of the grid lights", "restless", "commons-board chatter"},
	"wasteland":  {"a ragged runner", "scrap-paper note", "trade-day", "raiders", "a dust column on the horizon", "grim", "camp-fire talk"},
	"twilight":   {"a pale courier", "ash-sealed letter", "vigil of lamps", "the hollow-eyed", "a lamp guttering with no wind", "hushed", "a whisper in the dusk"},
}

type ndStageDef struct {
	K    string // letter rumour reveal ambush boss
	Sub  string // letter kind: taunt warn plea note
	From string // letter sender: foe friend town
	L    []string
}

type ndTpl struct {
	ID      string
	Title   string
	Flavour string
	Needs   string // nem rival war mentor love prophecy beast person
	W       int
	Spawn   []string
	Stages  [6]ndStageDef
	Tragic  []string // aftermath when the bound friend died
}

// ndFlavours: the ending flavours steered across lives, with a one-line description.
var ndFlavours = map[string]string{
	"triumph":    "a clean victory",
	"tragedy":    "a loss that cost dearly",
	"redemption": "a debt paid and a name mended",
	"vengeance":  "an old grudge settled",
	"peace":      "a hard-won quiet",
	"mystery":    "a question left open",
	"sacrifice":  "a price paid for others",
	"ruin":       "pride come to dust",
	"legacy":     "a lesson handed on",
}

var ndFlavourOrder = []string{"triumph", "tragedy", "redemption", "vengeance", "peace", "mystery", "sacrifice", "ruin", "legacy"}

var ndStageNames = [6]string{"inciting incident", "rising action", "twist", "crisis", "climax", "aftermath"}

var ndTpls = []*ndTpl{
	{ID: "vendetta", Title: "The Hunter's Debt", Flavour: "vengeance", Needs: "nem", W: 10, Spawn: []string{"bandit", "archer", "bandit"},
		Stages: [6]ndStageDef{
			{"letter", "taunt", "foe", []string{"Three lines in {foe}'s hand: the account between you is still open, and the interest is high.", "There is no greeting, only {foe}'s mark pressed in the corner. The {letter} says you were seen, and that being seen is how it starts."}},
			{"rumour", "", "", []string{"Folk in {town} say {foe} has been asking after you by name, and smiling in a {mood} way when they do.", "{Rumour}: {foe} paid a ferryman in {other} to tell them which way you went."}},
			{"reveal", "", "", []string{"The truth surfaces at last: {reason} It changes how you read every scar between you.", "Someone who knew {foe} before the grudge tells you what they were then. Not a monster. Only wronged, and patient."}},
			{"ambush", "", "", []string{"{Agents} in {foe}'s pay step into your road. This is the hunt feeling for your guard.", "The {omen} was a warning. {Agents} spring from cover, sworn to {foe}."}},
			{"boss", "", "", []string{"{foe}'s own people close the road in earnest. This is the last turn of the hunt, and they know it.", "No more hirelings: {foe}'s oath-sworn come themselves, and the ground between you and the next town is theirs."}},
			{"letter", "note", "town", []string{"Whatever stood between you and {foe} is settled, one way or another. The people of {town} remember who walked away from it.", "The matter of {foe} is closed. A clerk in {town} has written your name under it in a steady hand."}},
		},
		Tragic: []string{"{friend} paid for your quarrel with {foe}. Nobody in {town} will say it to your face; they do not need to."}},
	{ID: "rivalry", Title: "The Other Wanderer", Flavour: "ruin", Needs: "rival", W: 8, Spawn: []string{"bandit", "archer", "chief"},
		Stages: [6]ndStageDef{
			{"letter", "taunt", "foe", []string{"{foe} writes that the last prize was theirs by right, and the next will be too. Race you.", "A {letter} from {foe}: a map with your last victory crossed out in red."}},
			{"rumour", "", "", []string{"{Rumour}: {foe} has been seen in {other} with a new blade and a longer reach.", "In {town} they are laying bets on you and {foe}. The odds are not in your favour."}},
			{"reveal", "", "", []string{"A drunk in {town} lets slip that {foe} is not racing you for coin. They are racing you because you remind them of who they were.", "You find {foe}'s old camp: a journal, a name scratched out, and your own description in the margin."}},
			{"ambush", "", "", []string{"{foe}'s hangers-on try to settle the race on the road, with {agents} to do the settling.", "Someone has tipped {foe}'s friends to your route. {Agents} are waiting."}},
			{"boss", "", "", []string{"{foe} stops pretending it is a race. They step out into the road alone with a full retinue behind.", "The rivalry reaches its edge: {foe} and their sworn company, in the way, with nothing left to say."}},
			{"letter", "note", "town", []string{"Whoever won the race, the roads are quieter. Talk in {town} is already turning it into a story.", "{foe} is out of your road. The story is going round {town} and getting better with each telling."}},
		},
		Tragic: []string{"{friend} was caught between you and {foe}, and did not walk out of it. It was never only a race."}},
	{ID: "warshadow", Title: "The Shadow of {enemy}", Flavour: "sacrifice", Needs: "war", W: 7, Spawn: []string{"bandit", "archer", "chief"},
		Stages: [6]ndStageDef{
			{"letter", "warn", "town", []string{"The reeve of {town} writes that scouts of {enemy} have been seen on the border, and asks whether you have seen them too.", "{Msgr} brings word: {enemy} is moving and {realm}'s roads are no longer safe after dusk."}},
			{"rumour", "", "", []string{"{Rumour}: a column of {enemy} was seen near {other}, and the farms behind it are empty.", "They say in {town} that {enemy} offers coin for maps of {realm}'s fords. Someone is selling."}},
			{"reveal", "", "", []string{"A captured courier carried the truth: {enemy}'s march is a feint, and the real blow is aimed at a place nobody is watching.", "A dying scout tells you {enemy} is guided by one of {realm}'s own, someone with a seat at a table you have visited."}},
			{"ambush", "", "", []string{"{Agents} of {enemy} cut across your road, hunting anyone who has seen too much.", "A raiding party of {enemy} breaks from the {omen} and goes for your throat."}},
			{"boss", "", "", []string{"The captain of {enemy}'s vanguard finds you at last, with everyone they could gather. Someone has to hold this ground.", "{enemy}'s best come for the thing you are carrying in your head. You cannot go round them."}},
			{"letter", "note", "town", []string{"The border held. It cost more than the ledgers will say, and {town} knows who stood in the gap.", "The vanguard of {enemy} broke on the ford. The reeve of {town} sends thanks and a list of the names that were not as lucky."}},
		},
		Tragic: []string{"{friend} was among those the war took. {town} has put up a small stone and a larger silence."}},
	{ID: "mentor", Title: "{friend}'s Last Lesson", Flavour: "legacy", Needs: "mentor", W: 9, Spawn: []string{"bandit", "archer", "bandit"},
		Stages: [6]ndStageDef{
			{"letter", "plea", "friend", []string{"{friend} writes that there is a lesson left unfinished, and that they are not as young as they were. Come when you can.", "A short {letter} in a shaky hand from {friend}: they have something to show you, and little time to do it."}},
			{"rumour", "", "", []string{"{Rumour}: {friend} has been closing accounts and giving things away in {town}, quietly.", "In {town} they say {friend} was asked about an old debt, and went pale."}},
			{"reveal", "", "", []string{"{friend} was not only a teacher. The old debt they carried is the shape of the thing that is now hunting you both.", "Among {friend}'s papers is a name you know, and a promise never kept. They taught you to avoid it."}},
			{"ambush", "", "", []string{"{Agents} come for {friend}'s papers, and you are standing nearest to them.", "Someone wants what {friend} taught you, or what they hid. {Agents} are on the road."}},
			{"boss", "", "", []string{"The one {friend} feared for twenty years finds the two of you in the same place, with the debt in their hand.", "At last the old enemy of {friend} steps out. Whatever lesson this is, it is the final exam."}},
			{"letter", "note", "town", []string{"{friend}'s pupils are speaking of you in {town}. Whatever happens now, what you carry is no longer yours alone.", "The lesson is finished. What you do with it is the rest of the story, and {town} will be watching."}},
		},
		Tragic: []string{"{friend} did not live to see it finished. They left you the lesson, and an empty chair at the table in {town}."}},
	{ID: "love", Title: "The Road to {friend}", Flavour: "peace", Needs: "love", W: 8, Spawn: []string{"bandit", "wolf", "archer"},
		Stages: [6]ndStageDef{
			{"letter", "note", "friend", []string{"A {letter} from {friend}: nothing important, they say, only that {town} is {mood} without you. They ask when you will pass through.", "{friend} writes about nothing and everything. There is a pressed flower in the fold."}},
			{"rumour", "", "", []string{"{Rumour}: {friend} has been turning down every offer in {town}, and watching the road.", "A trader from {town} tells you {friend} asks after you each market day, trying to sound careless."}},
			{"reveal", "", "", []string{"You learn why {friend} never says more: someone in {town} holds a claim over them, and thinks you are the thing standing in the way.", "A letter you were not meant to read shows {friend} has been refusing a good match, for a reason you recognise."}},
			{"ambush", "", "", []string{"Someone in {town} would rather you did not come back. {Agents} are waiting on the road.", "{Agents} stalk your road, and {friend}'s name is on the lips of the one who paid them."}},
			{"boss", "", "", []string{"The rival claimant for {friend} comes to settle it in the open, with every blade they could hire.", "Whoever holds the claim on {friend} arrives with a retinue. There is no clever way out of this one."}},
			{"letter", "note", "friend", []string{"{friend} writes that the road home is open at last. The {fest} in {town} will have a place laid for you.", "Whatever else you are, {friend} writes, you are welcome in {town}. They have kept the flower."}},
		},
		Tragic: []string{"{friend} did not live to see the end of it. The {fest} in {town} will go on, and you will not be able to look at it."}},
	{ID: "prophecy", Title: "What the Verses Meant", Flavour: "mystery", Needs: "prophecy", W: 9, Spawn: []string{"wisp", "wight", "ashwraith"},
		Stages: [6]ndStageDef{
			{"rumour", "", "", []string{"{Omen}. Somebody in {town} says it is nothing. Somebody else says the first verse: \"{verse}\"", "{Rumour}: a stranger in {other} recited \"{verse}\" and left before anyone could ask where they heard it."}},
			{"rumour", "", "", []string{"Three people in {town} dreamed the same dream last night. One of them wrote down \"{verse}\" and cannot say why.", "The old words come back to you unbidden: \"{verse}\" You have not told anyone that you know them."}},
			{"reveal", "", "", []string{"You see it now: the verse never meant what the tellers said. It was written by someone who had already seen you at the end.", "The prophecy is a map, not a promise. \"{verse}\" is a place, and the place is a day's walk from here."}},
			{"ambush", "", "", []string{"Things that should not walk are walking: {omen}, and then the road is not empty.", "The verse is stirring in the land itself. Hollow shapes come out of the dusk toward you."}},
			{"boss", "", "", []string{"What the verse called the shadow takes a shape in front of you, and the ground goes quiet.", "The thing the old words named stands in the road. Either it is the end of something, or the start."}},
			{"letter", "note", "town", []string{"Whatever the verse meant, the sky in {town} looks different this morning. Old women are laughing and nobody knows why.", "The scribes in {town} have rewritten the verse with your name in it, and left the last line blank."}},
		},
		Tragic: []string{"{friend} is gone, and the verse, you realise, never said who would pay."}},
	{ID: "beast", Title: "The Terror of {other}", Flavour: "triumph", Needs: "beast", W: 9, Spawn: []string{"wolf", "bear", "boar", "jaguar"},
		Stages: [6]ndStageDef{
			{"rumour", "", "", []string{"{Rumour}: {foe} took three goats from a farm outside {other}, and then the farmer.", "In {town} they will not walk to the well after dark. They say {foe} has been seen at the tree-line."}},
			{"letter", "plea", "town", []string{"The reeve of {town} asks anyone who can hold a weapon to help with {foe}. Nobody has come.", "A shaky {letter}: {foe} was seen again. The reeve is offering what the town can spare."}},
			{"reveal", "", "", []string{"You find where {foe} lairs: it is not hunger that drives it, but something it was guarding, and the people of {other} stole it first.", "The tracks tell you {foe} is not killing for food. It is clearing a path, and you can read to where."}},
			{"ambush", "", "", []string{"The creatures that follow {foe} test your camp: a pack, {omen}, and the sound of something large behind them.", "{Omen}. A pack breaks cover, driven or following; the thing itself is not far."}},
			{"boss", "", "", []string{"{foe} comes out of the dark and does not run. This is its ground, and you stand on it.", "The trail ends here. {foe} fills the road, and the wind carries the smell of what it has been eating."}},
			{"letter", "note", "town", []string{"{foe} is dead. {town} has already begun telling it wrong, in the way people tell the stories they need.", "The people of {other} have hung something of {foe}'s over the gate. The children dare each other to touch it."}},
		},
		Tragic: []string{"{foe} is dead, but {friend} is too. {town} cannot make up its mind whether to cheer."}},
	{ID: "debt", Title: "A Debt in {town}", Flavour: "redemption", Needs: "person", W: 8, Spawn: []string{"bandit", "archer", "bandit"},
		Stages: [6]ndStageDef{
			{"letter", "plea", "friend", []string{"{friend} writes that they are in trouble in {town} and have nobody else to ask. They are not asking for money.", "A badly folded {letter} from {friend}: you did them a kindness once, and they are about to ask you to do another."}},
			{"rumour", "", "", []string{"{Rumour}: {friend} owes the wrong people in {town}, and the wrong people have started to run out of patience.", "They say in {other} that {friend} left {town} in a hurry, and that someone went after them."}},
			{"reveal", "", "", []string{"The debt is not what {friend} said it was. It is an old wrong, theirs, and they have been trying to pay it quietly for years.", "You learn what {friend} did, once, and why they have never asked anyone for anything since."}},
			{"ambush", "", "", []string{"{Agents} are collecting for someone in {town}, and they have decided you are part of the account.", "Someone sent {agents} after {friend}, and you are the nearest piece of the debt."}},
			{"boss", "", "", []string{"The creditor, {foe}, has come in person. They would like to hear what you propose to do about it.", "{foe} has never done their own collecting before. That alone should tell you what is owed."}},
			{"letter", "note", "friend", []string{"{friend} writes that the account is closed. It is the first {letter} they have written without apologising.", "The debt is paid, {friend} writes, and the name they were ashamed of is just a name again."}},
		},
		Tragic: []string{"{friend} did not live to see the debt closed, but they closed it. {town} will tell you so, if you ask."}},
}
