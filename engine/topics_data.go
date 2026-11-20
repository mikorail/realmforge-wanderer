package engine

// Text tables for topics.go (prefix tp). Nothing here draws g.R; callers pick with cpChoose / HashStr.

// tpToneOf maps a PersonVoice temper onto one of four speaking tones.
func tpToneOf(temper string) string {
	switch temper {
	case "proud", "cold":
		return "proud"
	case "anxious":
		return "nervous"
	case "gruff", "wry":
		return "gruff"
	}
	return "warm"
}

var tpHearsay = []string{
	"I am no %s, so take this as hearsay.",
	"That is not my trade. I only know what gets said.",
	"Ask a %s for the whole of it. What I have is scraps.",
	"I can only tell you what I have overheard.",
}

// hesitation lines when the speaker trusts you too little for a subject (shown before a check)
var tpReluct = map[string][]string{
	"warm":    {"They hesitate. \"That is not something I tell every passing face.\"", "\"Hm. Why do you want to know?\" A friendly face, but a careful one."},
	"proud":   {"They regard you down the length of their nose. \"And who are you to ask?\"", "\"Such matters are not for strangers.\""},
	"nervous": {"They glance about. \"I should not talk about that. Not with you.\"", "\"Please, ask me something else.\""},
	"gruff":   {"They scowl. \"Why should I tell you?\"", "\"Mind your own business, wanderer.\""},
}

// success and failure of a reluctance check: the tone changes the whole scene, not only the number
var tpWin = map[string][]string{
	"warm":    {"They study you, then smile and lean closer. \"You ask kindly. Very well.\"", "Their shoulders ease. \"All right. You seem honest enough. Listen.\""},
	"proud":   {"They lift their chin, as if granting a favour. \"You have manners, which is more than most. Hear it, then.\"", "\"Since you ask properly.\" They deign to answer."},
	"nervous": {"They glance around, then speak low and fast. \"Not so loud. All right, I will tell you.\"", "\"Oh, very well. But you did not hear it from me.\""},
	"gruff":   {"They grunt and fold their arms. \"Hah. Persistent, I will give you that. Fine.\"", "\"You will not let it lie, will you? Sit, then. Listen.\""},
}

var tpLose = map[string][]string{
	"warm":    {"Their smile thins. \"I am sorry, friend. I would rather not, and your pressing only makes me surer of it.\"", "\"No. Not today. Perhaps when we know each other better.\" The warmth has cooled."},
	"proud":   {"Their eyes go cold. \"You presume. This conversation of ours is finished.\"", "\"I do not answer questions put like that.\" They turn their shoulder to you."},
	"nervous": {"They flinch and step back. \"No. No, I cannot. Please do not ask me again.\"", "Their voice cracks. \"Leave it. Just leave it.\""},
	"gruff":   {"They spit to one side. \"Wasted breath. Go and pester someone else.\"", "\"Not a word. Not one. Try the next fool.\""},
}

// a tactless question, by tone (used when the culture has no rebuke of its own)
var tpTactless = map[string][]string{
	"warm":    {"They blink, then manage a tight smile. \"That was not kindly asked.\"", "\"I will pretend I did not hear that.\" The warmth drains from them."},
	"proud":   {"They stiffen. \"You forget yourself.\"", "\"Such a question is an insult, and you will not repeat it.\""},
	"nervous": {"They go pale. \"You should not ask that. Please. Not here.\"", "They look about to see who heard. \"Hush! What are you saying?\""},
	"gruff":   {"\"Watch your tongue.\" They look ready to throw you out.", "They bark a laugh with no humour in it. \"Brave. Or stupid.\""},
}

// opinion shifts (qualitative)
const (
	tpMsgWarm  = "{{G|They warm to you.}}"
	tpMsgCool  = "{{R|They think the less of you for it.}}"
	tpMsgGrief = "{{R|You have opened an old wound.}}"
)

var tpAgain = []string{
	"\"You asked me about %s before.\"",
	"\"We have been over %s already.\"",
	"\"%s? I told you.\"",
}

var tpAgainDay = []string{
	"\"You asked me about %s only a moment ago.\"",
	"\"Again, %s? I have not changed my answer since this morning.\"",
}

// what a speaker says of the weather, by weather kind id
var tpWeatherSay = map[string][]string{
	"clear":     {"A clean sky. Good for the road.", "Clear as a bell. It will not last, it never does."},
	"overcast":  {"The sky is shut like a lid. Rain by evening, I would say.", "Grey and heavy. My knees say it is turning."},
	"rain":      {"Wet work today. The roads will be soup by nightfall.", "Rain, rain. Good for the fields, bad for everything else."},
	"storm":     {"Stay off the high ground and keep away from lone trees. The storm means it.", "Lightning has already taken one roof this season. Mind yourself."},
	"snow":      {"The snow is early. It is soft now, but it will not stay soft.", "Snow. The little ones love it. The rest of us count the firewood."},
	"blizzard":  {"No one travels in this. If you must, tie a line to your belt and keep it to a friend.", "A blizzard. People die between their own door and the well in weather like this."},
	"fog":       {"A fog like wet wool. Sound goes strange in it, so trust your feet, not your ears.", "Fog. Nothing to be done but wait for it to lift."},
	"sandstorm": {"Cover your face and wait it out. The sand is in everything anyway.", "A sandstorm. Water first, shelter second, pride last."},
	"ashfall":   {"The ash is down again. Tie a cloth over your mouth, it gets in the lungs.", "Grey snow that is not snow. We sweep it every morning and every morning it comes back."},
	"spores":    {"The spores are drifting. Do not breathe deep near the stalks.", "Spore drift. Harmless, mostly. Mostly."},
	"shimmer":   {"The crystal motes are up. Pretty, and not to be trusted.", "That glitter in the air has been known to turn a head."},
	"heat":      {"A killing sun. Rest at noon, walk at dusk, and drink before you are thirsty.", "Hot enough to cook an egg on a shield. Do not be proud about water."},
}

// what a season means to the folk who live in it
var tpSeasonSay = [4][]string{
	{"Spring: the thaw, the sowing, and the mud. Everyone is busy and cheerful about it.", "Spring is for planting and for weddings, in that order, and for mending what winter broke."},
	{"Summer: long days, the harvest of hay, and the roads at their best. Travellers come and go.", "Summer is when we make what must last. Work from dawn, rest in the heat of the day."},
	{"Autumn: the great harvest, the slaughter, the salting. Whatever is not in the store by now will be missed.", "Autumn is bread and preserves and counting. Then the first frost comes and the counting is over."},
	{"Winter: we sit close, we mend tools, and we tell the old stories. The road is for the foolish.", "Winter is long. We keep the fire, share the stores, and wait for spring to prove us right."},
}

// fallback proverbs when the culture has none of its own
var tpProverbs = []string{
	"A guest asks no questions at the door and answers them all by the fire.",
	"The road remembers every footstep and forgives none.",
	"Water finds its level and a liar finds his.",
	"He who counts the coin twice has already been robbed once.",
	"A short tongue keeps a long head.",
	"The oven that is lit for one burns for all.",
	"Do not curse the bridge that you have yet to cross.",
	"Salt is cheap until the day it is not.",
	"A rumour has no legs; it rides on every tongue it meets.",
	"The wolf does not ask the sheep for leave.",
	"Better a lean peace than a fat war.",
	"What the child hears at the table, the whole town hears by noon.",
}

// fallback hospitality, when the culture has no pack
var tpGuestGeneric = []string{"bread and a seat by the fire", "water to wash the road off", "a corner of the hall and a blanket"}

var tpGuestDuties = []string{"keeps the peace under the roof", "gives their name and says honestly where they are bound", "lifts no hand against the house", "shares the news they carry"}

// fallback taboos by act (text, rebuke), used when the culture has no pack
var tpTabooGeneric = map[string][2]string{
	"draw_weapon_indoors": {"No one draws steel under another's roof here.", "Put that away, stranger. This is a house."},
	"refuse_hospitality":  {"A cup or a crust that is offered is taken. Refusing it is an insult to the house.", "You would refuse my table? Then say so plainly."},
	"insult_ancestors":    {"The dead are spoken of kindly or not at all.", "Mind how you speak of the dead."},
	"blaspheme":           {"Mocking the gods in public brings a fine and worse looks.", "Not in this town, you will not."},
	"waste_food":          {"Bread is never thrown out; it goes to the birds or the poor.", "We do not waste food here."},
	"public_anger":        {"Shouting in the street shames everyone on it.", "Take your temper home."},
	"speak_before_elder":  {"The young do not speak before the elders have.", "Wait your turn."},
	"stare":               {"Staring at strangers or at a person's hands is rude.", "What are you looking at?"},
}

var tpTabooActs = []string{"draw_weapon_indoors", "refuse_hospitality", "insult_ancestors", "blaspheme", "waste_food", "public_anger", "speak_before_elder", "stare"}

// generic punishments for the law topic when the culture has no pack
var tpPunish = map[string][]string{
	"theft":   {"a fine of twice the value, or the stocks for a week", "the lash, and the goods returned threefold"},
	"murder":  {"the rope, unless the dead one's kin accept blood-price", "exile from the realm, and a bounty on their return"},
	"assault": {"a fine paid to the hurt party", "a night in the cells and a public apology"},
}

// the sector words of a town, as a speaker puts them
var tpSectorSay = map[string]string{
	"farming": "the fields and the barns", "industry": "the workshops and the forges", "mining": "the pits and the ore carts",
	"trade": "the market and the counting houses", "crafts": "the craftsmen's quarter", "scholarship": "the schools and the scriptoria",
	"military": "the garrison and its needs", "faith": "the temples and what pilgrims leave behind", "seafaring": "the harbour and the ships",
}

// roles whose idle chatter in the pack has a Barks setting
var tpBarkFor = map[string]string{
	"merchant": "market", "market": "market", "shop": "market", "exotic": "market", "factor": "market", "guard": "guard", "barracks": "guard",
	"smith": "smith", "forge": "smith", "armorer": "smith", "villager": "farm", "farmer": "farm", "priest": "temple", "temple": "temple",
	"scholar": "school", "ferry": "harbor", "shipwright": "harbor", "dockmaster": "harbor", "bard": "tavern", "gambler": "tavern",
}

// role tags: what a speaker is expert in
var tpRoleTags = map[string][]string{
	"guard": {"guard", "law", "war"}, "barracks": {"guard", "law", "war"}, "captain": {"guard", "law", "war"},
	"priest": {"priest", "faith", "customs"}, "shaman": {"priest", "faith", "customs"}, "seer": {"priest", "faith"}, "temple": {"priest", "faith"},
	"pilgrim":  {"faith", "road"},
	"merchant": {"merchant", "road"}, "smith": {"merchant"}, "forge": {"merchant"}, "armorer": {"merchant"}, "tanner": {"merchant"},
	"apothecary": {"merchant", "scholar"}, "herbalist": {"merchant"}, "bookseller": {"merchant", "scholar"}, "exotic": {"merchant", "road"},
	"tailor": {"merchant"}, "market": {"merchant"}, "shop": {"merchant"}, "factor": {"merchant", "road"}, "ostler": {"merchant", "road"},
	"ferry": {"merchant", "road"}, "shipwright": {"merchant", "road"}, "stationmaster": {"merchant", "road"}, "dockmaster": {"merchant", "road"},
	"elder": {"elder", "past", "law", "customs"}, "chief": {"elder", "past", "law", "customs", "war"}, "mentor": {"elder", "past"},
	"chronicler": {"past", "scholar"}, "scholar": {"scholar", "past"}, "curator": {"scholar", "past"}, "naturalist": {"scholar"},
	"notable": {"noble", "war", "law", "past"}, "palace": {"noble", "war", "law", "past"},
	"bard": {"past", "rumour"}, "gambler": {"rumour"}, "bookie": {"rumour"}, "thief": {"rumour"},
}
