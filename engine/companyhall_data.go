package engine

// Companion depth data (prefix cx): quirks, callings for hall recruits, perk tree, bond ranks and all
// the text templates. Placeholders ({me} {you} {town} {kin} {realm} {faith} {place} {house} {coin}) are
// filled by cbFill (compbanter.go); extra ones ({amt} {foe} {item} {rival} {calling}) by the caller.

// Values a companion can hold, each -2..2.
const (
	cxMercy = iota
	cxGreed
	cxHonour
	cxPiety
	cxCurious
	cxViolence
)

var cxVNames = [6]string{"mercy", "greed", "honour", "piety", "curiosity", "violence"}

// cxVWords: [value][0 = holds the opposite, 1 = holds it].
var cxVWords = [6][2]string{
	{"hard-hearted", "merciful"},
	{"open-handed", "coin-hungry"},
	{"loose with oaths", "honour-bound"},
	{"faithless", "devout"},
	{"incurious", "inquisitive"},
	{"averse to bloodshed", "eager for a fight"},
}

type cxQuirk struct {
	ID, Name, Text string
	V              [6]int
	Loy            int
	Tag            string // firm turncoat sulky forgiving haggler
}

var cxQuirks = []cxQuirk{
	{ID: "soft", Name: "Soft-hearted", Text: "Cannot walk past a beggar or a beaten stray.", V: [6]int{2, 0, 0, 0, 0, -1}},
	{ID: "ruthless", Name: "Ruthless", Text: "Finishes what is started and sees no use in prisoners.", V: [6]int{-2, 0, 0, 0, 0, 1}},
	{ID: "coin", Name: "Coin-minded", Text: "Keeps count of every share and remembers who owes what.", V: [6]int{0, 2, -1, 0, 0, 0}, Tag: "haggler"},
	{ID: "open", Name: "Open-handed", Text: "Gives away what is not nailed down.", V: [6]int{1, -2, 0, 0, 0, 0}, Loy: 3},
	{ID: "oath", Name: "Oath-bound", Text: "Treats a promise as a debt, to be paid in full.", V: [6]int{0, 0, 2, 0, 0, 0}, Loy: 6, Tag: "firm"},
	{ID: "slip", Name: "Slippery", Text: "Always has one eye on the nearest exit and a price in mind.", V: [6]int{0, 1, -2, 0, 0, 0}, Tag: "turncoat"},
	{ID: "devout", Name: "Devout", Text: "Prays at dawn and expects the road to allow for it.", V: [6]int{1, 0, 0, 2, 0, 0}},
	{ID: "scoff", Name: "Scoffer", Text: "Has no patience for priests, shrines or sermons.", V: [6]int{0, 0, 0, -2, 1, 0}},
	{ID: "nosy", Name: "Inquisitive", Text: "Wants to see inside every ruin and behind every door.", V: [6]int{0, 0, 0, 0, 2, 0}},
	{ID: "home", Name: "Homebody", Text: "Would rather be anywhere with a roof and a hearth.", V: [6]int{0, 0, 1, 0, -2, -1}},
	{ID: "brawl", Name: "Brawler", Text: "Grins before a fight and hums after it.", V: [6]int{-1, 0, 0, 0, 0, 2}},
	{ID: "peace", Name: "Peaceable", Text: "Has seen enough burning roofs for one life.", V: [6]int{1, 0, 0, 0, 0, -2}, Loy: 2},
	{ID: "stub", Name: "Stubborn", Text: "Digs in when slighted and stays dug in.", V: [6]int{0, 0, 1, 0, 0, 0}, Tag: "sulky"},
	{ID: "cheer", Name: "Cheerful", Text: "Forgives fast and sings badly, often.", V: [6]int{1, 0, 0, 0, 0, 0}, Loy: 5, Tag: "forgiving"},
	{ID: "proud", Name: "Proud", Text: "Keeps a long memory for slights and a short one for thanks.", V: [6]int{0, 0, 1, 0, 0, 1}, Tag: "sulky"},
	{ID: "grasp", Name: "Treasure-hungry", Text: "Weighs every ruin by what it might hold.", V: [6]int{-1, 2, 0, 0, 1, 0}, Tag: "haggler"},
	{ID: "loyal", Name: "Dog-loyal", Text: "Sticks to a chosen leader through weather and worse.", V: [6]int{0, 0, 1, 0, 0, 0}, Loy: 8, Tag: "firm"},
	{ID: "bitter", Name: "Bitter", Text: "Expects the worst of people and is seldom let down.", V: [6]int{-1, 1, -1, 0, 0, 0}, Tag: "turncoat"},
}

func cxQuirkByID(id string) *cxQuirk {
	for i := range cxQuirks {
		if cxQuirks[i].ID == id {
			return &cxQuirks[i]
		}
	}
	return nil
}

// cxCallings: what a hall recruit can be. Attr is added to the base 14s of makeNPC.
type cxCalling struct {
	Key     string
	Attr    [6]int
	Weapons []string
	Armor   bool
	Blurb   []string
}

var cxCallings = []cxCalling{
	{Key: "warrior", Attr: [6]int{3, 1, 2, 0, 0, 0}, Weapons: []string{"longblade", "axe", "cudgel"}, Armor: true,
		Blurb: []string{"Knows the front line and the long wait behind it.", "Has the scars to prove most of the stories.", "Fights in the first rank and sleeps in the last."}},
	{Key: "scout", Attr: [6]int{0, 3, 1, 1, 1, 0}, Weapons: []string{"shortblade", "longblade"}, Armor: false,
		Blurb: []string{"Reads tracks, weather and moods before they turn.", "Finds the way when the map and the land disagree.", "Counts exits first and people second."}},
	{Key: "rogue", Attr: [6]int{0, 3, 0, 1, 0, 2}, Weapons: []string{"shortblade"}, Armor: false,
		Blurb: []string{"Opens locks, pockets and the occasional conversation.", "Lives by quick hands and a quicker exit.", "Has a name for every back alley and a price for most of them."}},
	{Key: "mystic", Attr: [6]int{0, 0, 1, 1, 3, 1}, Weapons: []string{"cudgel"}, Armor: false,
		Blurb: []string{"Keeps vigils and prefers to be asked before being thanked.", "Sees omens in small things and is sometimes right.", "Carries a lamp, a staff and a long list of quiet duties."}},
	{Key: "scholar", Attr: [6]int{0, 0, 0, 3, 2, 1}, Weapons: []string{"cudgel", "shortblade"}, Armor: false,
		Blurb: []string{"Carries more books than is wise and has read all of them.", "Counts, sketches and annotates everything on the road.", "Has argued with archives and won."}},
	{Key: "tinker", Attr: [6]int{1, 1, 2, 3, 0, 0}, Weapons: []string{"cudgel", "axe"}, Armor: true,
		Blurb: []string{"Mends what breaks and breaks what refuses to be mended.", "Keeps a pouch of tools and a pocket of spare screws.", "Cannot see a latch without wondering how it works."}},
}

func cxCallingByKey(k string) *cxCalling {
	for i := range cxCallings {
		if cxCallings[i].Key == k {
			return &cxCallings[i]
		}
	}
	return &cxCallings[0]
}

// cxRoleCalling maps a life-sim trade to a calling.
var cxRoleCalling = map[string]string{
	"guard": "warrior", "sergeant": "warrior", "soldier": "warrior", "smith": "warrior", "blacksmith": "warrior", "miner": "warrior",
	"hunter": "scout", "fisher": "scout", "herder": "scout", "carter": "scout", "sailor": "scout", "ferryman": "scout",
	"thief": "rogue", "smuggler": "rogue", "gambler": "rogue", "beggar": "rogue", "tavern": "rogue",
	"priest": "mystic", "healer": "mystic", "monk": "mystic", "herbalist": "mystic",
	"scribe": "scholar", "teacher": "scholar", "clerk": "scholar", "scholar": "scholar", "bard": "scholar",
	"weaver": "tinker", "potter": "tinker", "carpenter": "tinker", "mason": "tinker", "tailor": "tinker", "baker": "tinker",
}

var cxBioReasons = []string{
	"a quarrel over a debt", "a season of bad harvests", "the garrison being disbanded", "a guild closing its doors",
	"a feud nobody remembers the start of", "a wedding that did not happen", "the road simply looking better than the farm",
	"a bad winter and a worse landlord", "a misunderstanding with the watch",
}

var cxBioDeeds = []string{
	"Has walked with caravans between {town} and the coast.",
	"Fought for {realm} in the border skirmishes and took no pay for the last season.",
	"Once stood guard over a shrine of {faith} and still keeps the habit of silence at dusk.",
	"Has crossed the same bridge near {town} more often than anyone cares to count.",
	"Carries a letter from {home} that has never been answered.",
	"Spent a winter in {town} waiting for work that did not come.",
}

// ---------------- lines ----------------

var cxLines = map[string][]string{
	"hired": {
		"Mind the terms and I will mind the road, {you}.",
		"A fair offer. Let us see whether the work is as fair.",
		"I have walked with worse. We will manage.",
	},
	"levelup": {
		"Something clicked into place today. I know what I want to improve.",
		"I have been thinking about how I fight. There is more I could learn.",
		"Give me a moment to decide what I have learned on this road.",
	},
	"sulk": {
		"I am not in the mood for orders today, {you}. Ask me tomorrow.",
		"Do what you like. I am walking, not listening.",
		"No. Not today. You know why.",
	},
	"demand:share": {
		"We have done well. I want my cut, {you}, and I want it now.",
		"That purse came from my work too. Split it, or we will have words.",
		"I count {amt} as my share. Do you count it the same?",
	},
	"demand:mercy": {
		"That was not needed, {you}. Promise me you will spare those who yield.",
		"I did not sign on for this. Give me your word it will not happen again.",
		"They had put it down. Swear you will let the next one walk away.",
	},
	"leave": {
		"I have made up my mind. I will not walk this road with you any longer.",
		"This is where our ways part, {you}. Do not follow.",
		"I have stayed too long. Give me my share of the road home.",
	},
	"stay:win": {
		"Hm. Fair enough. I will give it another try.",
		"You do know how to talk. I will stay, for now.",
	},
	"stay:lose": {
		"No. My mind is made up.",
		"Nothing you say will change it. Farewell.",
	},
	"round:win": {
		"You have a way of making sense. All right. We start fresh.",
		"I had my say. So have you. Let us move on.",
		"Fine. I will take your word, {you}. Do not waste it.",
	},
	"round:lose": {
		"Words are cheap. Show me.",
		"No. Not yet. Give it time.",
		"You mean it, maybe. But I am not ready.",
	},
	"warn1:sell": {
		"{me} was seen in a back room with a stranger who left a heavy purse on the table.",
		"{me} went quiet whenever strangers asked about your route, and then did not look you in the eye.",
	},
	"warn1:theft": {
		"{me} was caught weighing your pack with their eyes when they thought you were not looking.",
		"{me} lingers by your gear a little longer than there is any reason for.",
	},
	"warn1:ambush": {
		"{me} sharpens a blade at night, long after the work is done, and says nothing.",
		"{me} walks behind you now, and does not like being asked why.",
	},
	"warn2:sell": {
		"{me} muttered an address and a price to a messenger, and the messenger left in a hurry.",
		"A rival's token, small and bright, fell out of {me}'s sleeve and was picked up quickly.",
	},
	"warn2:theft": {
		"Your pack was disturbed in the night. Nothing was gone, yet. {me} says it was the wind.",
		"{me} asked what the heaviest thing in your pack is worth. Too lightly.",
	},
	"warn2:ambush": {
		"{me} stopped talking to you altogether. They watch the road ahead and the shadows beside it.",
		"{me} slipped away at dusk and returned late, smiling for the first time in days.",
	},
	"betray:sell": {
		"{me}: \"Nothing personal. The price was right, and your trail was easy to read.\"",
		"{me}: \"They paid for where you sleep. I merely told them.\"",
	},
	"betray:theft": {
		"{me}: \"You will not miss it as much as I will need it.\"",
		"{me}: \"Keep the rest. I have what I came for.\"",
	},
	"betray:ambush": {
		"{me}: \"You should have been kinder. Now you will have to be quicker.\"",
		"{me}: \"This road ends here, {you}. For one of us.\"",
	},
	"talk:ask": {
		"I am listening. Say what you came to say.",
		"If you are here to talk, talk. Do not waste my time.",
	},
}

// value lines: "up:<value>" when you do something they cherish, "down:<value>" when you do not.
var cxValueLines = map[string][]string{
	"up:mercy":       {"That was well done, {you}. Mercy costs little and buys a lot.", "I am glad you spared them. I would have asked, but you did not need asking.", "There is more to you than the blade, {you}. I like that."},
	"down:mercy":     {"That was cruel, {you}. They had done nothing to deserve it.", "I will not forget what I saw there. I wish I could.", "You did not have to do that. We both know it."},
	"up:greed":       {"Now that is how to work: more in the purse than out of it.", "Coin well taken. I like it when you think about the bottom line.", "You know the value of a thing. That is rare, {you}."},
	"down:greed":     {"You gave that away like it was nothing. Our purse has feelings, too.", "Charity is fine, {you}, but not on my share.", "We could have kept that. We needed it."},
	"up:honour":      {"A promise kept. That is what I have been looking for, {you}.", "You did that cleanly. No tricks. I respect it.", "Word and deed together. I will follow that."},
	"down:honour":    {"That was not honest, {you}, and I will not pretend it was.", "I did not join to be a thief's shadow.", "You cheated. I saw it. I will remember."},
	"up:piety":       {"Walking by {faith} brings luck. Thank you for stopping.", "A fine thing, to remember the gods on the way. I felt it.", "I prayed for a leader like this one. I am not disappointed."},
	"down:piety":     {"You pass sacred ground as if it were a market stall. It hurts to watch.", "The gods are keeping count, {you}, even if you are not.", "I do not like how you treat what is holy."},
	"up:curiosity":   {"Look at this place! I could stay a month, {you}.", "Thank you for taking the long way. There is always something to learn.", "That was worth the detour. Let us do more of those."},
	"down:curiosity": {"Another dead end. We could have seen something new.", "You never stop to look at anything, {you}. Not once.", "Moving on again. I wanted to read the carvings."},
	"up:violence":    {"Now that was a fight! My arm still sings from it.", "Good work, {you}. Steel talks better than words.", "Keep that up and nobody will trouble us again."},
	"down:violence":  {"More blood. Must there always be more blood?", "I wish you would think before you swing, {you}.", "That could have been settled with words, if you had tried."},
}

// ---------------- perks ----------------

type cxPerkDef struct {
	ID, Name, Branch, Text string
	Tier                   int
	Req                    string // another perk id, "" for none
	HP, AV, DV             int
	Attr                   [6]int
	Tag                    string // medic forager lore steady negotiator bargainer fearless
}

var cxPerks = []cxPerkDef{
	{ID: "tough", Name: "Tough hide", Branch: "Body", Tier: 1, HP: 8, Attr: [6]int{0, 0, 1, 0, 0, 0}, Text: "+8 HP, +1 Toughness"},
	{ID: "hardened", Name: "Hardened", Branch: "Body", Tier: 2, Req: "tough", HP: 4, AV: 1, Text: "+4 HP, +1 AV"},
	{ID: "ironclad", Name: "Ironclad", Branch: "Body", Tier: 3, Req: "hardened", AV: 2, Attr: [6]int{0, 0, 1, 0, 0, 0}, Text: "+2 AV, +1 Toughness"},
	{ID: "strong", Name: "Heavy hands", Branch: "Edge", Tier: 1, DV: 1, Attr: [6]int{1, 0, 0, 0, 0, 0}, Text: "+1 Strength, +1 DV"},
	{ID: "nimble", Name: "Light feet", Branch: "Edge", Tier: 1, DV: 1, Attr: [6]int{0, 1, 0, 0, 0, 0}, Text: "+1 Agility, +1 DV"},
	{ID: "duelist", Name: "Duelist", Branch: "Edge", Tier: 2, Req: "nimble", DV: 2, Text: "+2 DV"},
	{ID: "reaver", Name: "Reaver", Branch: "Edge", Tier: 3, Req: "strong", AV: 1, Attr: [6]int{2, 0, 0, 0, 0, 0}, Text: "+2 Strength, +1 AV"},
	{ID: "keen", Name: "Keen mind", Branch: "Wits", Tier: 1, Attr: [6]int{0, 0, 0, 1, 1, 0}, Tag: "lore", Text: "+1 Intelligence, +1 Willpower; picks up bits of lore that add to your experience"},
	{ID: "forager", Name: "Forager", Branch: "Wits", Tier: 1, Tag: "forager", Text: "finds food on the road: you grow hungry more slowly"},
	{ID: "medic", Name: "Field medic", Branch: "Wits", Tier: 2, Req: "keen", Tag: "medic", Text: "patches you up when you rest wounded and out of danger"},
	{ID: "steady", Name: "Steady nerve", Branch: "Heart", Tier: 1, Attr: [6]int{0, 0, 0, 0, 1, 0}, Tag: "steady", Text: "+1 Willpower; takes your bad calls more calmly"},
	{ID: "bargainer", Name: "Bargainer", Branch: "Heart", Tier: 1, Tag: "bargainer", Text: "halves any share or coin they demand"},
	{ID: "peacemaker", Name: "Peacemaker", Branch: "Heart", Tier: 2, Req: "steady", Attr: [6]int{0, 0, 0, 0, 0, 1}, Tag: "negotiator", Text: "+1 Ego; easier to talk round when something goes wrong"},
	{ID: "fearless", Name: "Fearless", Branch: "Heart", Tier: 3, Req: "peacemaker", DV: 1, Attr: [6]int{0, 0, 0, 0, 1, 0}, Tag: "fearless", Text: "+1 DV, +1 Willpower; never sulks"},
}

func cxPerkByID(id string) *cxPerkDef {
	for i := range cxPerks {
		if cxPerks[i].ID == id {
			return &cxPerks[i]
		}
	}
	return nil
}

// ---------------- bond ----------------

var cxRankNames = [4]string{"acquaintance", "friend", "trusted", "sworn"}
var cxRankNeed = [4]int{0, 6, 14, 26}
var cxRankPerk = [4]string{
	"no bond yet",
	"+1 DV: they watch their footing around you",
	"+1 AV, +6 HP: they put themselves between you and harm",
	"+1 STR, AGI and TOU; loyalty never falls below 60 and betrayal is out of the question",
}

var cxFireOpen = map[string][]string{
	"warm": {
		"They shuffle closer to the fire and make room for you without being asked.",
		"{me} pokes the embers and smiles. \"Sit with me a while, {you}.\"",
		"The light plays on {me}'s face. \"I have been meaning to tell you something.\"",
	},
	"even": {
		"{me} looks up from the flames. \"Evening. Something on your mind?\"",
		"They nod to you across the fire and wait to see what you have to say.",
		"{me} makes space for you on the log. \"Long day.\"",
	},
	"cool": {
		"{me} keeps eyes on the fire. \"Talk, if you must. I am listening, barely.\"",
		"They do not move over, so you stand. \"Well?\"",
		"The fire is warm. {me} is not. \"What do you want?\"",
	},
}

// stories by key. "v:<value>:+" or "-" when the value is held strongly; quirk ids; "secret" at trusted rank.
var cxStories = map[string][]string{
	"town": {
		"{me} talks about {town}: the smell of the market at dawn, the bridge where they learned to fish, the corner nobody could get past without a nod.",
		"\"I think of {town} most evenings,\" {me} says. \"Not the people. The sounds. The shutters, the carts, a dog that never stopped.\"",
	},
	"kin": {
		"\"{kin},\" {me} says, and then says nothing for a while. \"They would have liked you. Or not. They never liked anybody quickly.\"",
		"{me} turns a small thing over in their hands. \"{kin} gave me this. I never found out why.\"",
	},
	"road": {
		"{me} counts the road on their fingers: the places, the weather, the people who did them a kindness and the ones who did not.",
		"\"I did not know where this would lead,\" {me} says. \"I still do not. But I am glad I am not walking it alone.\"",
	},
	"soft": {
		"\"I once sat up three nights with a dying dog,\" {me} says. \"It was not even mine. It was the only company it had.\"",
		"{me} says, \"I have never been able to hurt something that stopped fighting. I think that is a weakness. I do not wish it away.\"",
	},
	"ruthless": {
		"\"Leave an enemy alive,\" {me} says, \"and they come back on a worse day. I learned that once and never again.\"",
		"{me} stares into the fire. \"I do not enjoy it. I just do not see the point of doing it halfway.\"",
	},
	"coin": {
		"\"Everything has a price,\" {me} says, \"and a cost. The trick is to know both before you shake hands.\"",
		"{me} laughs quietly. \"My mother used to say I could count before I could talk. She was not entirely wrong.\"",
	},
	"open": {
		"\"I never kept a coin for long,\" {me} says. \"It always found its way to somebody who needed it more.\"",
		"{me} grins. \"Everything I own, somebody else has worn first. Everything I used to own is on somebody else now.\"",
	},
	"oath": {
		"\"I swore an oath once, at a gate in {town},\" {me} says. \"I have never regretted it, even when it cost me.\"",
		"{me} says, \"A word given is a thing you cannot take back. That is the whole point of it.\"",
	},
	"slip": {
		"{me} smiles without humour. \"I can get into most rooms. Getting out of them is the part nobody teaches.\"",
		"\"I do not lie,\" {me} says. \"I just leave out the parts that would make people tell me no.\"",
	},
	"devout": {
		"{me} touches a worn charm. \"{faith} has never answered me aloud. But the silence has not been empty.\"",
		"\"I pray for you too, you know,\" {me} says. \"Not that you asked.\"",
	},
	"scoff": {
		"\"I went to temple as a child,\" {me} says. \"I counted the candles. I never found the god.\"",
		"{me} shrugs. \"If there is a god, they have a strange way of showing up. I will believe it when I am introduced.\"",
	},
	"nosy": {
		"{me} sketches in the dust with a stick. \"I have always wanted to know what is behind the next hill. And the one after that.\"",
		"\"I opened a sealed box once,\" {me} says, \"just to know what it held. I regret nothing and apologise to no one.\"",
	},
	"home": {
		"\"I never wanted to leave,\" {me} says. \"The road is long, and I keep looking back. But it is better with company.\"",
		"{me} sighs. \"A good bed, a low fire, a door that locks. That is the whole of what I ever asked for.\"",
	},
	"brawl": {
		"{me} flexes a scarred hand. \"First fight I ever won, I was nine. Lost the tooth. Still the best day of my life.\"",
		"\"I do not hate people,\" {me} says. \"I just like knowing exactly where I stand, and a fight tells you quick.\"",
	},
	"peace": {
		"{me} stirs the coals gently. \"I held a spear for five years. I would rather hold a ladle.\"",
		"\"Every fight I won took something with it,\" {me} says. \"I am still trying to find out what.\"",
	},
	"stub": {
		"\"I once refused to speak to my brother for two years over a boat,\" {me} says. \"It was a bad boat. He was right.\"",
		"{me} folds their arms. \"I am not stubborn. I am simply never wrong for very long.\"",
	},
	"cheer": {
		"{me} hums a tune, loses it, and starts another. \"Someone has to keep the mood up. It might as well be me.\"",
		"\"I am happy more often than makes sense,\" {me} says. \"I have given up trying to explain it.\"",
	},
	"proud": {
		"\"I was told once that I would amount to nothing,\" {me} says. \"I remember the exact day. I remember the exact voice.\"",
		"{me} sits straighter. \"I do not ask for much. I ask not to be treated as less than I am.\"",
	},
	"grasp": {
		"\"Gold does not keep you warm,\" {me} says, \"but it buys the wood that does. I have been cold, {you}. I will not be again.\"",
		"{me} weighs a coin in one palm. \"It is not about being rich. It is about never again having to ask.\"",
	},
	"loyal": {
		"\"I followed a captain once,\" {me} says. \"Eleven years. When they fell, I did not know who I was for a season.\"",
		"{me} looks at you across the fire. \"I do not give my trust cheaply. But once it is given, it stays.\"",
	},
	"bitter": {
		"{me} laughs without joy. \"People disappoint. I do not mind it any more. It saves me the surprise.\"",
		"\"I trusted someone once,\" {me} says. \"The rest of the lesson I learned alone.\"",
	},
	"v:mercy:+":     {"\"I think about the ones we did not kill,\" {me} says. \"I hope they remember it too.\""},
	"v:mercy:-":     {"\"Soft hearts get people killed,\" {me} says. \"I have the graves to prove it.\""},
	"v:greed:+":     {"\"There is nothing noble about being poor,\" {me} says. \"I tried it. I will take the coins.\""},
	"v:greed:-":     {"\"I am happiest when the pack is light,\" {me} says. \"Carrying less leaves more room for company.\""},
	"v:honour:+":    {"\"A person is the sum of what they keep their word on,\" {me} says. \"The rest is noise.\""},
	"v:honour:-":    {"\"Honour is a luxury,\" {me} says. \"The rich invented it so the poor would not ask for more.\""},
	"v:piety:+":     {"\"I take comfort in a thing being larger than me,\" {me} says. \"It makes my troubles smaller.\""},
	"v:piety:-":     {"\"I looked up once and saw nothing,\" {me} says. \"It was oddly freeing. I have not looked up for help since.\""},
	"v:curious:+":   {"\"Every road is a book,\" {me} says. \"I am only on the first few pages.\""},
	"v:curious:-":   {"\"I know what I need to know,\" {me} says. \"The rest only gets in the way.\""},
	"v:violence:+":  {"\"I trust my hands more than my words,\" {me} says. \"They have never let me down.\""},
	"v:violence:-":  {"\"The best fight I ever won,\" {me} says, \"was the one I walked away from.\""},
	"secret": {
		"{me} lowers their voice. \"I took coin once from {rival} to pass on a letter. I never found out what it said. I have wondered ever since.\"",
		"\"I have never told anyone,\" {me} says, \"but I left {town} the night I did because I was afraid. Not of a person. Of staying.\"",
		"{me} says, \"I have lied to you about one thing, {you}. A small one. I will tell you what it was, someday, when it stops mattering.\"",
		"\"There is a place near {town},\" {me} says, \"where I buried something I was ashamed of. I go there in my head more than I should.\"",
	},
}

// reactions after your choice, by band.
var cxReact = map[string]map[string][]string{
	"listen": {
		"warm": {"{me} looks relieved to have been heard. \"Thank you. Most would have changed the subject.\""},
		"even": {"{me} nods. \"Thanks for hearing it out.\""},
		"cool": {"{me} blinks, as if surprised you stayed. \"Well. That is that, then.\""},
	},
	"share": {
		"warm": {"{me} laughs and shakes their head. \"I did not expect that from you, {you}. It makes us even.\""},
		"even": {"{me} considers it. \"Different world than mine. But I hear you.\""},
		"cool": {"{me} listens, grudgingly. \"I suppose I did not know that about you.\""},
	},
	"pry": {
		"warm": {"{me} hesitates, then smiles. \"You are persistent. I will say this much, and no more.\""},
		"even": {"{me} gives a little. \"That is as much as I will say tonight.\""},
		"cool": {"{me} stiffens. \"That is none of your business, {you}.\""},
	},
}

var cxRankUp = [4][]string{
	nil,
	{"{me} claps you on the shoulder. \"You are all right, {you}. I mean it.\""},
	{"{me} meets your eyes. \"I would trust you with my back in a dark hall. I mean to.\""},
	{"{me} takes your hand. \"Wherever the road goes, I walk it with you. I swear it by {faith}.\""},
}

var cxHallSigns = []string{
	"A board by the door lists names, callings and the terms each wants.",
	"A ledger of people looking for a leader hangs from a nail by the hearth.",
	"The innkeeper keeps a slate of those who would hire out their blades, their eyes, or their lore.",
}
