package engine

// Tables for letters.go (prefix ln): genre voices of the town crier, the printer and the post, letter openings, bodies
// and replies. Plain data, no logic.

// lnVoice is how a genre names and sounds its news trade.
type lnVoice struct {
	Paper   string   // masthead of the daily paper
	Unit    string   // what the thing is: broadsheet, datashard...
	Crier   string   // title of the town crier
	Printer string   // title of the seller
	Post    string   // title of the post clerk
	Cry     []string // how the crier announces a headline; %s is the headline
	Quiet   []string // when there is nothing to cry
	Head    string   // section headers
	Market  string
	Obit    string
	Wed     string
	Rumour  string
	Board   string
	Sold    string // line when you buy; %s is the unit
}

var lnVoices = map[string]lnVoice{
	"fantasy": {"The Daily Broadsheet", "broadsheet", "town crier", "printer", "postmaster",
		[]string{"The crier rings his bell: \"Hear ye, hear ye! %s\"", "\"Oyez!\" cries the town crier. \"%s\"", "A hand-bell clangs. \"News! %s\""},
		[]string{"The crier clears his throat, finds nothing new, and rings his bell for luck.", "\"All is quiet in the realm,\" the crier announces, almost apologetically."},
		"Tidings", "Prices at the Market", "Deaths", "Banns and Weddings", "The Rumour Mill", "The Notice Board", "The printer folds a still-damp %s into your hands."},
	"mythic": {"The Herald's Tablet", "tablet", "herald", "tablet-scribe", "runner-master",
		[]string{"The herald lifts his staff and proclaims: \"Citizens! %s\"", "\"Hear the Muses' news,\" sings the herald. \"%s\"", "A herald, crowned with laurel, announces: \"%s\""},
		[]string{"The herald waits on the stones for an omen, and none comes.", "\"The gods keep their counsel today,\" the herald says."},
		"Proclaimed", "Prices of the Agora", "The Departed", "Weddings and Offerings", "What the Muses Whisper", "Notices of the Agora", "The scribe presses a fresh wax-and-clay %s on you."},
	"twilight": {"The Lamplight Courier", "pale broadsheet", "lamplighter", "pressman", "night-postmaster",
		[]string{"The lamplighter calls softly down the street: \"%s\"", "Between two lamps, a reedy voice: \"%s\"", "The lamplighter pauses with his pole. \"%s\""},
		[]string{"The lamplighter shrugs; the fog has swallowed all news tonight.", "\"Nothing stirs,\" the lamplighter murmurs. \"Which is its own kind of news.\""},
		"Late Tidings", "Prices at the Night Market", "Passed Beyond", "Bannings and Bridals", "Whispers in the Fog", "The Lantern Board", "The pressman wraps a smudged %s in grey paper."},
	"steampunk": {"The Brass Gazette", "gazette", "newsboy", "press-master", "post-engineer",
		[]string{"A newsboy bawls, waving a damp sheet: \"Extra! Extra! %s\"", "\"Read all about it!\" squeals the newsboy. \"%s\"", "Over the hiss of the steam-press: \"%s\""},
		[]string{"The newsboy kicks a cog. \"Slow day, guv. Nothing's broke.\"", "\"Nothing to shout,\" the newsboy admits, and sells you a pencil."},
		"Dispatches", "Prices on the Exchange", "Obituaries", "Marriages and Births", "Gossip from the Boiler-Room", "Situations and Wants", "The press-master hands over a still-warm %s."},
	"solarpunk": {"The Commons Zine", "zine", "town speaker", "zine-maker", "post-cyclist",
		[]string{"The speaker steps onto the commons stage: \"Neighbours, %s\"", "From the solar-lit stage: \"Share this: %s\"", "A volunteer with a megaphone: \"%s\""},
		[]string{"The speaker smiles. \"A quiet day, neighbours. Go and water something.\"", "\"No news today. Good,\" says the speaker, \"that is also a harvest.\""},
		"This Week on the Commons", "Swap and Share Rates", "Remembered", "Pairings and Newborns", "Overheard at the Compost", "Commons Board", "The zine-maker staples a fresh %s together for you."},
	"dieselpunk": {"The Daily Wire", "late edition", "newsie", "newsstand man", "mail clerk",
		[]string{"A newsie bawls: \"Wire! Wire! %s\"", "\"Late edition!\" hollers the newsie. \"%s\"", "A kid on the corner, waving a folded sheet: \"%s\""},
		[]string{"The newsie spits. \"Nothin' doin' today, mister.\"", "\"Slow wire,\" says the newsie. \"Come back at noon.\""},
		"Front Page", "Ticker and Prices", "Death Notices", "Weddings and Births", "Street Talk", "Help Wanted", "The newsstand man slides a %s across the counter."},
	"atompunk": {"The Daily Atom", "paper", "announcer", "newsstand clerk", "post clerk",
		[]string{"The loudspeaker crackles: \"Attention, citizens. %s\"", "The plaza announcer, bright as a jingle: \"%s\"", "From a pole-mounted horn: \"%s\""},
		[]string{"The loudspeaker plays a cheerful jingle and says nothing at all.", "\"No bulletins at this hour, citizens. Carry on.\""},
		"Headlines", "Prices at the Co-op", "Passings", "Engagements and Arrivals", "Water-Cooler Talk", "Community Board", "The clerk folds today's %s into a neat square for you."},
	"cyberpunk": {"NetFeed Shard", "datashard", "street broadcaster", "feed vendor", "dead-drop clerk",
		[]string{"A street broadcaster, jacked into the plaza speakers: \"Top feed: %s\"", "The ad-wall glitches into text: \"%s\"", "A drone buzzes overhead, reading out the feed: \"%s\""},
		[]string{"The ad-wall flickers through nothing but ads.", "\"Feed's dead quiet,\" says the broadcaster. \"Somebody paid for that.\""},
		"Top Feed", "Ticker", "Flatlined", "Hitchings and Spawn", "Vox Pop", "Gig Board", "The vendor slots a cheap %s into your reader."},
	"biopunk": {"The Spore Bulletin", "pressed bulletin-leaf", "scent-herald", "bulletin grower", "spore-post keeper",
		[]string{"The scent-herald exhales a trail of pheromone and speaks: \"%s\"", "A herald, pale with spores: \"%s\"", "Through the humid air, a clicking voice: \"%s\""},
		[]string{"The herald's gland pulses once and falls still: nothing to report.", "\"The colony is calm,\" the herald clicks."},
		"Colony Tidings", "Prices at the Vat Market", "Composted", "Pairings and Hatchlings", "Spore Talk", "Brood Board", "The grower plucks a fresh %s from the wall and presses it into your hand."},
	"wasteland": {"The Chalkboard", "scrap sheet", "bell-ringer", "board-keeper", "mailbag keeper",
		[]string{"The bell-ringer clangs a pipe on a rusted drum: \"Hear it! %s\"", "Someone bangs a pot. \"Scavver news! %s\"", "A ragged voice atop the water tank: \"%s\""},
		[]string{"The bell-ringer bangs the drum twice and shrugs. No news is the best news out here.", "\"Nothing worth the breath,\" the bell-ringer rasps."},
		"Chalked Up", "Barter Rates", "Boneyard List", "Hitched and Hatched", "Wasteland Talk", "Scavver's Board", "The board-keeper copies the chalkboard onto a %s for you."},
}

// lnSign: how a letter ends, by tone (cold, neutral, warm) and genre.
var lnSign = map[string][3]string{
	"":           {"Take this as a warning.", "Yours in good faith,", "With all my affection,"},
	"fantasy":    {"Take this as a warning.", "Yours in good faith,", "With all my affection,"},
	"mythic":     {"Let the gods be witness.", "May the gods favour you,", "Beloved of the Muses, be well,"},
	"twilight":   {"Beware the long dark.", "Under the same moon,", "Candle-lit and thinking of you,"},
	"steampunk":  {"I remain, with great displeasure,", "Yours faithfully,", "Your most devoted correspondent,"},
	"solarpunk":  {"I hope you will reflect on this.", "In solidarity,", "Warmly, from the commons,"},
	"dieselpunk": {"Don't make me say it twice.", "Regards,", "Your pal,"},
	"atompunk":   {"Consider yourself notified.", "Sincerely,", "Your friend in the neighbourhood,"},
	"cyberpunk":  {"-- last warning, end of message --", "-- signal out --", "-- stay online, choom --"},
	"biopunk":    {"Our colony remembers.", "In the same brood,", "Pheromones and fondness,"},
	"wasteland":  {"Watch your back out there.", "See you on the roads,", "Stay alive, friend,"},
}

var lnOpen = [3][]string{
	{"%s.", "To the wanderer %s, and I do not say it kindly.", "%s: read this and take heed."},
	{"%s,", "To %s,", "Greetings, %s."},
	{"Dear %s,", "My dear %s,", "Friend %s,"},
}

var lnNewsIntro = []string{"I have news from %s.", "Things have changed here in %s.", "You once asked how life was in %s. Here is how it goes.", "A letter from %s, as promised."}

// lnReact follows a piece of personal news, by life-event kind.
var lnReact = map[string][]string{
	"marriage": {"I wanted you to hear it from me.", "I am happier than I expected to be.", "Come and see us if the road ever brings you this way."},
	"child":    {"I hardly sleep, and I do not mind.", "They say the child has my stubbornness.", "I hope to see you meet them one day."},
	"job":      {"The work is steady, which is all anyone can ask.", "It is not much, but it is mine.", "Folk here treat me differently now."},
	"move":     {"It still feels strange, but I will grow into it.", "The road was long and my feet are sore.", "Write to me at the new address."},
	"friend":   {"Life is easier with good company.", "You would like them.", "It is good to have someone to talk to."},
	"feud":     {"Say nothing of it to anyone, I beg you.", "I will not be the one to back down.", "I tell you only because I trust you."},
	"crime":    {"I am not proud of it.", "Do not think too badly of me.", "These things happen."},
	"goal":     {"A long road, and I am glad to have walked it.", "I could not have done it without a little luck.", "There is always another hope after the last one."},
	"secret":   {"Now you know what I kept from you.", "It cannot be helped now.", "I would rather you heard it kindly."},
}

var lnPleaCause = []string{"The year has been hard and my purse is empty.", "A debt has come due and I cannot meet it.", "My roof needs mending before the rains.", "Illness took my savings.", "A thief cleaned out my coffer."}
var lnPleaAsk = []string{"If you could spare %d coin I would not forget it.", "Could you send %d coin? I will repay you when I can.", "I would be grateful for %d coin, and ashamed to ask."}

var lnWarnIntro = []string{"Word has reached us: %s. Keep your eyes open.", "Travellers here speak of this: %s. I thought of you.", "Be careful on the roads: %s."}

var lnInviteWed = []string{"I am to be wed to %s, and the day is set for %s in %s. It would mean much if you were there.", "%s and I will be married on %s in %s. Come if you can, and bring your appetite."}
var lnInviteFun = []string{"%s has died (%s). The burying is on %s in %s. Come if you can.", "We lost %s (%s). Please come to the funeral in %s on %s, if the roads allow."}
var lnInviteFeast = []string{"There will be a feast in %s on %s. A seat is kept for you.", "We are holding a feast on %s in %s. It would not be the same without you."}

var lnThreatKin = []string{"You killed %s, my %s. I have not forgotten and I will not forget.", "My %s, %s, is dead by your hand. Look over your shoulder.", "You may have forgotten %s. I have not. They were my %s."}
var lnGrudge = []string{"I have not forgotten what you did: %s", "Do not come back here. %s", "You owe me for this: %s"}

var lnGiftLine = []string{"I enclose %d coin, a small thing for the road.", "I scraped together %d coin. Spend it on something warm.", "Please accept this %d coin; do not argue."}

var lnNote = []string{"Nothing much to report. I am %s these days.", "Life goes on. I am feeling %s.", "I write only to say I am %s and thinking of you."}

var lnFar = []string{"The world turns here in %s as ever. News of you has crossed even the rift, in pieces and out of order.", "I hear you walk other worlds now. I keep your name where I can.", "%s has not changed much since you left, though I have. Write when you can."}

var lnCompanion = []string{"Since we parted I have thought of the road more than I expected.", "I tell the tale of our travels to anyone who will listen, with improvements.", "Home is smaller than I remember, and quieter."}

var lnRivalLines = map[string][]string{
	"friendly": {"I had thought to race you for the next prize, but I would rather share the spoils. Meet me when you are near.", "You fight better than most. I hope you live long enough to prove it again."},
	"wary":     {"Stay out of my dungeons and I will stay out of yours.", "I have heard what you carry. I am not interested in taking it, yet."},
	"hostile":  {"The next relic is mine. If you are in my way, you will regret it.", "I am told you boast of your deeds. Bring them to my door and we shall see."},
}
var lnNemKin = []string{"You know whom I speak for: %s. Their blood is on your hands. We are coming.", "%s was one of ours. Run if you can, wanderer. Rest if you dare.", "Every road you take, we are one step behind. This is on account of %s."}

// lnAnswer: what a person writes back after your reply, by reply kind.
var lnAnswer = map[string][]string{
	"thanks":  {"Your letter found me well. Thank you for writing.", "It was good to hear from you."},
	"greet":   {"A kind word from afar is worth a great deal.", "Thank you for thinking of me."},
	"promise": {"I will hold you to that promise, in good humour.", "Your word is enough for me."},
	"decline": {"I understand. I will manage.", "I did not expect much. Thank you for answering."},
	"coin":    {"The coin arrived. You have my thanks.", "I will not forget this kindness."},
	"gift":    {"The gift arrived safe. I will keep it close.", "You should not have, but I am glad you did."},
	"rumour":  {"Your news was interesting. I will keep an ear out.", "I had not heard that. Thank you."},
	"accept":  {"I will tell the others to expect you.", "I will keep a place for you."},
	"regret":  {"A pity, but I understand.", "Another time, then."},
}

// lnReplyText: how a reply reads in the outgoing list.
var lnReplyText = map[string]string{
	"thanks": "thanks", "greet": "greetings", "promise": "a promise of help", "decline": "polite refusal", "coin": "coin",
	"gift": "a gift", "rumour": "a rumour", "accept": "acceptance", "regret": "regrets",
}
