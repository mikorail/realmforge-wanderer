package engine

// Aging and legacy data (prefix ag): life stages, texts and tables. Pure data, no state.

// agYear is the length of a year in days: the same 112-day year lvYear uses.
const agYear = 112

type agStage struct {
	ID    string
	Name  string
	From  float64 // fraction of the lifespan at which the stage begins
	Attr  [6]int  // attribute shift against the chargen baseline (STR AGI TOU INT WIL EGO)
	XPMul float64 // wisdom and eagerness: multiplier on experience gained
	Regen int     // extra turns between healing ticks (slower recovery with age)
	Check map[string]int
	Note  string
}

var agStages = []agStage{
	{ID: "young", Name: "young", From: 0, Attr: [6]int{0, 1, 0, -1, -1, -1}, XPMul: 1.15, Regen: -2,
		Check: map[string]int{"bluff": 1, "acrobatics": 1, "lore": -1}, Note: "quick to heal and to learn, short on judgement"},
	{ID: "prime", Name: "in your prime", From: 0.28, Attr: [6]int{0, 0, 0, 0, 0, 0}, XPMul: 1, Regen: 0,
		Check: nil, Note: "body and mind in balance"},
	{ID: "mature", Name: "mature", From: 0.55, Attr: [6]int{-1, -1, 0, 1, 1, 1}, XPMul: 1.05, Regen: 2,
		Check: map[string]int{"lore": 1, "persuasion": 1, "empathy": 1}, Note: "a little slower, a good deal wiser"},
	{ID: "old", Name: "old", From: 0.75, Attr: [6]int{-2, -3, -2, 2, 2, 2}, XPMul: 1.15, Regen: 7,
		Check: map[string]int{"lore": 1, "persuasion": 1, "empathy": 1, "perception": 1}, Note: "wounds close slowly; people listen"},
	{ID: "venerable", Name: "venerable", From: 0.92, Attr: [6]int{-3, -4, -3, 3, 3, 2}, XPMul: 1.25, Regen: 14,
		Check: map[string]int{"lore": 2, "persuasion": 1, "empathy": 2, "perception": 1}, Note: "frail, revered, and not long for the road"},
}

// agSpeciesFallback: used when the world has no species record and lsLifespan knows nothing either.
var agSpeciesFallback = map[string]int{"human": 72, "mutant": 60, "construct": 150, "android": 120, "undead": 300}

// agLongTraits and agShortTraits adjust a lifespan by species trait words (lowercase substrings).
var agLongTraits = []string{"long-lived", "ageless", "ancient", "enduring", "regenerat"}
var agShortTraits = []string{"short-lived", "frail", "fleeting", "brief"}

var agBirthdayYoung = []string{
	"Another year behind you, and the road still looks wide open.",
	"A year older. The world has not yet noticed how much you plan to do in it.",
}
var agBirthdayPrime = []string{
	"Another year on the road. You are at the height of your strength and you know it.",
	"A year older. Nothing aches, and everything is possible.",
}
var agBirthdayMature = []string{
	"Another year on the road. Your knees have opinions about it now.",
	"A year older. You catch yourself giving advice nobody asked for, and it is good advice.",
	"Another year. You wake a little more slowly, and think a little more clearly.",
}
var agBirthdayOld = []string{
	"Another year, and you are still here to count it. The mornings are harder; the evenings are richer.",
	"A year older. Old companions of the road are fewer. You carry their stories for them.",
	"Another year. You rest more, and you are no longer ashamed of it.",
}
var agBirthdayVenerable = []string{
	"Another year, a gift you no longer assume. You count it slowly, and with thanks.",
	"A year older. The young step aside for you in doorways, and you let them.",
	"Another winter survived. You have begun to put your affairs in order without quite meaning to.",
}

var agBirthdays = map[string][]string{
	"young": agBirthdayYoung, "prime": agBirthdayPrime, "mature": agBirthdayMature, "old": agBirthdayOld, "venerable": agBirthdayVenerable,
}

var agStageChange = map[string]string{
	"prime":     "{{G|You have come into your full strength.}}",
	"mature":    "{{Y|You are mature now: slower on the stairs, quicker in the head. (STR and AGI ease, INT, WIL and EGO sharpen.)}}",
	"old":       "{{O|Age has settled on you. Old wounds speak up, but people lean in when you talk. (Recovery slows; wisdom grows.)}}",
	"venerable": "{{R|You are venerable. The road is short now, and every step is chosen. Consider your Legacy (F6).}}",
}

// agTalkLines: what people say about your years, by your stage and by their relation to you.
// Key is stage; each has lines for an elder NPC (older than you), a younger one and a peer.
var agTalkLines = map[string][3][]string{
	"young": {
		{"\"Hm. You have the look of a pup who thinks the road is a race. Slow down, it is a long one.\"", "\"Young one, listen to someone who has already made your mistakes.\""},
		{"\"You look about my age. Do you know what you are doing out there? I never do.\"", "\"A fellow youngster! Everyone stares at us, have you noticed?\""},
		{"\"So young to be so far from home. Mind yourself.\"", "\"Plenty of fire in you, I see. Mind it does not burn the house down.\""},
	},
	"prime": {
		{"\"You are in your best years. Spend them well; I wish I had.\"", "\"A strong back and a clear eye. They will not stay, so use them.\""},
		{"\"People like you keep towns like this going. We are glad of it.\"", "\"You carry yourself like someone with places to be.\""},
		{"\"Good to meet someone who looks the way I feel on my best days.\"", "\"We are the same age, you and I, give or take. The world is ours for a while yet.\""},
	},
	"mature": {
		{"\"Ah, a seasoned traveller. You will forgive an old one for saying the years sit well on you.\"", "\"You have the patience of someone who has been wrong enough to learn.\""},
		{"\"You have a steady way about you. I hope I am like that when I am older.\"", "\"My mother says folk like you are what make a town trustworthy.\""},
		{"\"We are both past the reckless years, I think. Sit, have a drink.\"", "\"Grey at the temples? Ha. You carry it well.\""},
	},
	"old": {
		{"\"A fellow elder on the road, and with all your own teeth too. Sit, sit.\"", "\"We of long years know what a long road costs. Take the better chair.\""},
		{"\"Please, take my seat. You have walked more roads than I have names for.\"", "\"Grandmother used to say the old ones are the town's memory. May I ask you something?\""},
		{"\"The old ones are a rarity out here. What keeps you walking?\"", "\"You are older than most who come through. Is it true what they say about the road keeping a person?\""},
	},
	"venerable": {
		{"\"We two have been here a very long time, you and I. Few can say as much. Rest while you can.\"", "\"Look at us, the last of the old stock. Let us at least be comfortable.\""},
		{"\"I, I did not think anyone could live so long and still walk. It is an honour.\"", "\"Tell me your story, elder, before it goes with you. I will remember it.\""},
		{"\"The very old are listened to, in this town. Anything you say, I will carry on.\"", "\"They will sing of you, you know. Whether you want them to or not.\""},
	},
}

// agIllnesses: the ailments of old age. Statuses are registered in aging.go.
type agIllness struct {
	ID       string
	Name     string
	Msg      string
	MinStage int // index into agStages
	Dur      int // turns
}

var agIllnesses = []agIllness{
	{"ag_joints", "Aching joints", "{{O|Your joints seize up. Old bones remember every winter.}}", 2, 1800},
	{"ag_cough", "Winter cough", "{{O|A rattling cough settles in your chest and will not be argued with.}}", 3, 2400},
	{"ag_dim", "Failing eyes", "{{O|The world goes soft at the edges today. Your eyes are not what they were.}}", 3, 1500},
	{"ag_fog", "Memory fog", "{{O|A grey fog crosses your thoughts. You lose a name you had known for decades.}}", 4, 1500},
	{"ag_weak", "Frailty", "{{R|A deep weakness spreads through you. Your body asks, politely, to stop.}}", 4, 2000},
}

var agDeathCause = []string{
	"You died of old age in %s, %s, at the age of %d. You had made your peace, and the road ended gently.",
	"You died peacefully in %s, %s, aged %d. A long life, completely lived.",
	"Your long journey ended in %s, %s, at %d years. You laid down the road at last, and no one could say you had not earned it.",
}

var agRiteLines = []string{
	"The priest leads you to a basin of cold water under a moon-window. \"The Rite of Renewal does not make you young,\" they say. \"It reminds the body what young was.\"",
	"They wash your hands, your face, your feet. Bells you cannot see ring in a distant room. \"Breathe out the years,\" the priest says. \"Slowly.\"",
}

var agSpringLines = []string{
	"A clear spring wells from a crack in the rock, so cold it stings, and sings faintly in a tongue older than speech.",
	"Water rises in a shallow basin that was surely set there by hands, though no hands have tended it in a hundred years. It smells of rain and of something you cannot name.",
}

var agEpitaphs = []string{
	"Walked far, rested well.",
	"They left the road better than they found it.",
	"Not forgotten while the stories are told.",
	"Here ends a long journey, and begins the telling of it.",
	"The hearth is cold, but the warmth was real.",
}

// agCharities: where a will may send the remainder of the gold.
type agCharity struct {
	ID, Name, News string
}

var agCharities = []agCharity{
	{"none", "no one in particular", ""},
	{"poor", "the poor of the town", "%s's estate paid for bread for the poor of %s for a season."},
	{"temple", "the local temple", "%s's bequest paid for new lamps and a roof in the temple of %s."},
	{"school", "a school for the town's children", "%s's bequest opened a small school in %s."},
	{"watch", "the town watch", "%s's bequest paid for the watch of %s to be properly armed."},
	{"hospice", "a hospice for the old", "%s's bequest opened a hospice for the old of %s."},
}

var agHeirTags = []string{"heirloom", "keepsake", "bequest"}

// agLegendTrades: trades given to descendants who turn up in later lives.
var agLegendTrades = []string{"farmer", "potter", "weaver", "carter", "fisher", "carpenter", "tanner", "baker"}

var agDescTalkOpen = []string{
	"\"My grandmother used to say we come from a wanderer. People tell me I have the same chin.\"",
	"\"They say a famous traveller sleeps somewhere in our family tree. I never believed it until I saw a stranger with the same eyes.\"",
	"\"Our family keeps one old story: that we are kin to a wanderer who walked this world long ago.\"",
}
