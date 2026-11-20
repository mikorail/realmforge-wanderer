package engine

// Fallback bank for ambient barks and overheard chatter (barks.go). Used when the speaker's culture has no
// voice pack. Placeholders: {town} {realm} {ruler} {coin} {feast} {name}; barks.go fills every one.

// baBank: one-liners per setting (and per situation: wanted armed feast death wed plague famous infamous).
var baBank = map[string][]string{
	"market": {
		"Fresh and honest! Come and see, nothing here older than a week!",
		"Fair prices today in {town}! Fair for me, at least!",
		"Buy two, and I will swear the second one is free. Mostly free.",
		"Last of the good stock, and I will not see its like again this season!",
		"Weighed true, priced fair, and not one {coin} of it stolen!",
		"Step up, step up. A looker never starved a merchant, but it does not feed him either.",
	},
	"guard": {
		"Move along, nothing to see here.",
		"Keep your hands where I can see them and we shall get along.",
		"Quiet night, quiet day. I like it quiet.",
		"Gate closes at dusk. Do not be the one banging on it.",
		"By order of {ruler}, no brawling in {town}. I mean it.",
		"Eyes open, everyone. Trouble likes a crowd.",
	},
	"tavern": {
		"A full cup and an empty purse, that is a good night in {town}!",
		"Stew is hot, ale is cold, and my cat has not been in the stew today!",
		"Last round before the rain, friends!",
		"Sit anywhere. Break nothing. Pay before you sing.",
		"Who put a coin in my ale? Not a complaint. Do it again.",
		"If the song is bad, at least have the decency to sing it loudly.",
	},
	"temple": {
		"Peace be with the weary. There is room on the bench for you.",
		"A candle is a small coin for a long comfort.",
		"The doors are open to all who come with a quiet heart.",
		"Bring no quarrels past the threshold, friends.",
		"Pray if you can, listen if you cannot.",
		"What is lost in the dark is found in the morning, they say. Bring a lamp anyway.",
	},
	"road": {
		"Safe roads to you, stranger!",
		"Mind the ruts past the ford, they swallow a wheel as soon as look at it.",
		"Long way to {town}, but the walking is honest.",
		"Anyone seen a stray goat? Brown, sulks, answers to nothing.",
		"Dust, dust, and more dust. And then, I am told, more dust.",
		"Rest while the sun is kind. The road will wait.",
	},
	"crowd": {
		"Make way, make way, there are people trying to walk here!",
		"Did you see that? No? Never mind, it is gone now.",
		"Pardon! Pardon! My elbow has a mind of its own!",
		"Is it market day? It feels like market day. It is always market day.",
		"{town} never sleeps, it only sulks until morning.",
		"Mind your purse in this press, friend. Everyone else is.",
	},
	"night": {
		"Lamps out soon. Go home, the dark is no friend of strangers.",
		"Late for honest folk, early for the other sort.",
		"Quiet at last. I do love {town} when it is quiet.",
		"Mind the shadows by the wall, they have a way of moving.",
		"One more lamp, one more story, then sleep. Always one more.",
		"Cold tonight, and getting colder.",
	},
	"rain": {
		"Rain again! The roofs will be earning their keep today.",
		"Get under cover, you will melt like a sugar cake!",
		"Good for the crops, bad for the boots, that is what I always say.",
		"Mud to the knee by noon, mark my words.",
		"At least it washes the street. Not that it needed it. Well, it needed it.",
		"Wet as a fish, and I do not even fish.",
	},
	"smith": {
		"Hammer rings true when the iron is willing!",
		"Blades sharpened, hinges mended, no questions about the blood.",
		"A good edge is patience and a little shouting.",
		"Hot work, thirsty work. Someone fetch water, or I fetch a grudge.",
		"Bring me your broken things, and I will bring them back worse. Joking! Better!",
		"Sparks fly, coin follows. That is the trade.",
	},
	"farm": {
		"Sun on the fields, mud on the boots. A good season, if the gods are kind.",
		"The barley is high and the crows are higher in spirit.",
		"Plough early, curse late. That is how it goes.",
		"Rain will come when it comes. The land does not care for my opinion.",
		"Another year, another field full of stones. Where do they come from?",
		"Good soil in {town}, but it asks a deal of a man's back.",
	},
	"harbor": {
		"Tide turns soon! Anyone not aboard is a swimmer!",
		"Gulls and gossip, that is all the harbor has before noon.",
		"Mind the ropes, mind the planks, mind your own business!",
		"Salt on the wind and coin in the hold, that is a good day!",
		"Who is hauling this? Not me, I only supervise.",
		"Fish by the basket, and not one of them lied to!",
	},
	"school": {
		"Quiet in the cloister, please. Even a pen can be too loud.",
		"Read it twice, and think once. That is the whole of learning.",
		"A question asked is a door half open.",
		"No running with the ink, I beg you!",
		"The book will wait for you. That is its great virtue.",
		"Knowledge is heavy to carry, but it never rots.",
	},
	"wanted": {
		"Hold there, wanderer. {realm} has a long memory and a short patience for your sort.",
		"I know that face. There is a notice with it on, up on the wall.",
		"Do not think we have forgotten what you did, stranger.",
		"One wrong step in {town} and you will see the inside of a cell.",
		"There is a bounty on your head. I would hate to be tempted.",
	},
	"armed": {
		"Sheathe that blade in {town}, stranger. Now.",
		"Steel stays in its scabbard here. That is the law of {ruler}.",
		"Armed in the street? Put it away before someone makes a story of it.",
		"Easy, wanderer. A drawn weapon makes my hand twitch.",
		"I see that weapon. You see my spear. Let us both be sensible.",
	},
	"feast": {
		"Happy {feast}, one and all!",
		"{feast} at last! Raise a cup and forget your debts for a day!",
		"Bring something to the table for {feast}, or at least bring a good appetite.",
		"Is the {feast} fare ready yet? My stomach has been counting the hours.",
		"Joy of {feast} to you, stranger. Even the guards are smiling.",
	},
	"death": {
		"A sad business, that. {town} will feel the loss.",
		"Black cloth for the doors this week, I fear.",
		"Say a word for the dead, friend. They have no one else to say it.",
	},
	"wed": {
		"Joy to the couple! There is a cake somewhere in this, I hope.",
		"A wedding in {town}! The gods smile, even on the bills.",
		"Lucky pair. May the first quarrel be a short one.",
	},
	"plague": {
		"Keep a cloth to your face, friends. The sickness walks in {town}.",
		"Do not crowd the well. Do not crowd anything. Please.",
		"They say the fever came on the road. Close the door behind you.",
	},
	"famous": {
		"That is the wanderer folk speak well of. Make way!",
		"Good to see you in {town}, friend. Your name is a good one here.",
		"A friend of {realm}! Let it be said the road did something right today.",
	},
	"infamous": {
		"Watch that one. The tales are not kind.",
		"I would not turn my back on the wanderer, if I were you.",
		"Of all the roads, why did that one choose ours?",
	},
	"high": {
		"Order, if you please. We are not the sort of people to shout.",
		"One expects better manners in a town of {town}'s standing.",
		"A word to the wise is worth a word to the foolish, doubled.",
	},
	"low": {
		"Hard times, hard hands. We manage, we always manage.",
		"Another day, another coin short. Still, the sun is free.",
		"The great folk argue and the rest of us pay for it.",
	},
}

// baGenre: a few lines coloured by genre, mixed into the bank for the same setting (fantasy needs none).
var baGenre = map[string]map[string][]string{
	"mythic": {
		"market": {"Ambrosia and honeycomb, blessed by a minor god! The god is not here, but the honeycomb is."},
		"guard":  {"The oracle says trouble comes at dusk. The oracle is usually right. Be somewhere else."},
		"crowd":  {"Did a god pass by? Everyone is staring at the sky."},
		"road":   {"Leave an offering at the shrine ahead, or the road remembers."},
	},
	"steampunk": {
		"market": {"Brass fittings, good as new! Pressure-tested by my own brother-in-law!"},
		"guard":  {"Boilers are not toys. Stay clear of the pipes, citizen."},
		"crowd":  {"That soot is not coming out of my coat, I tell you."},
		"road":   {"The steam-coach is late again. Mind the tracks."},
	},
	"dieselpunk": {
		"market": {"Ration stamps accepted, no questions, bring exact change!"},
		"guard":  {"Papers ready, citizen. Papers ready."},
		"crowd":  {"Another siren drill. Walk, do not run, and look cheerful."},
		"road":   {"Fuel is scarce on this stretch. Walk if you must."},
	},
	"atompunk": {
		"market": {"Chrome and chrome! Everything is the future this season!"},
		"guard":  {"Radiation badges at the checkpoint, please. Yes, even you."},
		"crowd":  {"Did the lights flicker, or is it just me? Nobody says."},
		"road":   {"Mind the glow past the ridge. Pretty, but not good for you."},
	},
	"cyberpunk": {
		"market": {"Chips, cheap! Do not ask whose they were before!"},
		"guard":  {"Scan your ID, chummer. Do not make a thing of it."},
		"crowd":  {"Neon in the eyes, empty in the pockets. Welcome to {town}."},
		"road":   {"Drone patrol at the junction. Look boring."},
	},
	"biopunk": {
		"market": {"Grown fresh this morning! The roots have not even finished complaining!"},
		"guard":  {"Do not feed the culture vats, citizen. Do not touch them either."},
		"crowd":  {"Something is growing on the wall again. Nobody planted it."},
		"road":   {"The brambles lean toward walkers past the bend. Walk quickly."},
	},
	"solarpunk": {
		"market": {"Sun-dried and glad of it! Bring your own basket, and a smile!"},
		"guard":  {"Keep the panels clear, and keep the peace. That is the whole job."},
		"crowd":  {"Bees on the balconies, bells on the bicycles. A good day in {town}."},
		"road":   {"Share the shade with strangers, that is the road custom."},
	},
	"wasteland": {
		"market": {"Clean water, no rust! Pure as it gets! Which is not very pure!"},
		"guard":  {"Walls up, eyes out. The raiders count on us getting bored."},
		"crowd":  {"Keep your voice down. Sound travels, and so do the hungry."},
		"road":   {"Fresh tracks past the dune. Do not follow them."},
	},
	"twilight": {
		"market": {"Candles that burn without smoke, and not a ghost in sight! Mostly!"},
		"guard":  {"The lamps stay lit. That is the first rule and the last."},
		"crowd":  {"It is darker than it was a moment ago. Stay close."},
		"road":   {"Do not answer the voice on the road. Not even a polite no."},
	},
}

// baProverbs: used when a culture has no pack of its own.
var baProverbs = []string{
	"A full granary makes a quiet winter.",
	"Slow water cuts the deepest stone.",
	"Never trust a lamp that burns for free.",
	"Better a hard truth at the gate than a soft lie at the hearth.",
	"The bell does not ring itself.",
	"A guest is a gift for three days and a bill after.",
	"Who sows kindness reaps a crowd.",
	"The crooked road teaches the straight foot.",
	"Count your coin in the dark and your friends in the light.",
	"Every lock has a patient thief.",
}

var baOpeners = []string{"Have you heard?", "Did you hear?", "Word is that", "They say that", "I heard that", "Folk are saying that"}

var baReplies = []string{
	"No! Truly?",
	"I would not have guessed it.",
	"Hah. I heard something close to that.",
	"Well. That explains a great deal.",
	"Do not tell me you believe that.",
	"Say it softer, the walls have ears in {town}.",
	"It does not surprise me, I will say that.",
	"Mark my words, there is more to it.",
}

var baFollows = []string{
	"So I said, nobody listens to me, and yet here we are.",
	"Well, I only repeat what I hear.",
	"Do not repeat it. Or do, I cannot stop you.",
	"We shall see what the week brings.",
	"Anyway. Back to work, back to work.",
	"I would not wager on it either way.",
}

// complaints about prices and rulers.
var baComplaints = []string{
	"Prices! A loaf costs what a pie cost last year, and {ruler} does nothing.",
	"The tax collector came again. Third time this season. {realm} bleeds us dry.",
	"Salt is dear, wool is dear, even a kind word costs a {coin} in this town.",
	"They say {ruler} is clever. Then why is my purse always empty?",
	"I paid double for a plain knife yesterday. A plain knife!",
	"The toll at the gate went up again. Soon it will cost more to leave {town} than to stay.",
}

var baComplaintReplies = []string{
	"Do not speak so loud. The guards hear everything.",
	"You are right, and you are wasting your breath.",
	"My mother always said it would come to this.",
	"I would complain if I had the time. I have the debts instead.",
}

var baDuelEnds = []string{
	"Hah. You win that one. Do not let it go to your head.",
	"Fine, fine. Yours is the older saying. Mine is the truer.",
	"We could go on all day. I have work to do, thank the gods.",
}

var baDuelCaps = []string{"Ah, but", "And yet they also say", "Then answer me this", "Well, my grandmother said"}

// baIntro: how overheard exchanges are announced to the player.
var baIntro = []string{
	"You overhear %s and %s.",
	"%s and %s are talking nearby.",
	"You catch a snatch of talk between %s and %s.",
}
