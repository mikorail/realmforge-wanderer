package engine

// Built-in voice bank for worlds whose cultures carry no VoicePack: six registers, four or five lines per kind.
// Lines are bare speech (no quotes); {name} is filled with the way the speaker addresses the player.

var dvKinds = []string{"greet", "farewell", "thanks", "apology", "agree", "refuse", "praise", "insult", "curse", "oath", "gossipOpen"}

var dvRegisters = []string{"formal", "blunt", "flowery", "terse", "earthy", "courtly"}

var dvBank = map[string]map[string][]string{
	"formal": {
		"greet":      {"Good day to you, {name}. How may I be of service?", "You are welcome here, {name}. Please, speak.", "I bid you good day. State your purpose.", "Well met, {name}. The hour is yours.", "Greetings. Be at ease and say what you need."},
		"farewell":   {"Farewell, {name}. May your road be kept.", "I take my leave of you with good wishes.", "Go with care, {name}.", "Until we meet again, then.", "Our business is concluded. Good day."},
		"thanks":     {"I am obliged to you, {name}.", "You have my thanks, and my regard.", "Your kindness will be remembered.", "It is gratefully received.", "I thank you, truly."},
		"apology":    {"I beg your pardon, {name}. The fault was mine.", "Please accept my apology.", "I regret the offence, and ask your forbearance.", "It was ill done of me. Forgive it."},
		"agree":      {"Quite so, {name}.", "I concur entirely.", "That is just and proper.", "It shall be as you say.", "Agreed, with all my heart."},
		"refuse":     {"I regret that I must decline, {name}.", "That cannot be permitted.", "It is not within my power, nor my duty.", "I am afraid the answer is no.", "I must refuse, and I ask you not to press it."},
		"praise":     {"A fine deed, {name}, and well done.", "You do yourself honour.", "Commendable. Truly commendable.", "Few could have managed it so well.", "I am impressed, and I say so plainly."},
		"insult":     {"You are beneath notice, {name}.", "What a coarse and graceless creature you are.", "I had expected better of anyone.", "You disgrace the very ground you stand on.", "Spare me your insolence."},
		"curse":      {"May your name be struck from every register.", "Ill fortune attend you, {name}.", "May you be forgotten by all who matter.", "A plague upon your house and your ledgers."},
		"oath":       {"Upon my honour, it is true.", "I swear it by all I hold dear.", "Let the record show I have sworn it.", "By my name and by my word."},
		"gossipOpen": {"It is said, and I repeat it with reservation, that", "I am told, though I cannot vouch for it, that", "Word has reached me that", "One hears, in certain quarters, that"},
	},
	"blunt": {
		"greet":      {"What do you want, {name}?", "Right. You're here. Say it.", "Yes? Make it quick.", "Oh, it's you. Go on, then.", "Speak up, {name}. I haven't all day."},
		"farewell":   {"Get on, then.", "See you. Maybe.", "Go. I've work.", "Right. Off you go, {name}.", "That's all. Away with you."},
		"thanks":     {"Fair enough. Thanks.", "Good. I owe you.", "Ta, {name}.", "That was decent of you.", "Right. Thanks."},
		"apology":    {"I was wrong. There. Happy?", "My mistake, {name}.", "Sorry. Won't happen again.", "Fine. I'm sorry."},
		"agree":      {"Aye, that's right.", "You've said it.", "No arguing that.", "Fine by me.", "Good. Do it."},
		"refuse":     {"No.", "Not a chance, {name}.", "Not happening.", "Ask someone else.", "I said no. I meant it."},
		"praise":     {"Not bad. Not bad at all.", "Well done, {name}. Truly.", "That took guts.", "You've earned that.", "Good work."},
		"insult":     {"You're a fool, {name}.", "Get out of my sight, you lump.", "Useless. Absolutely useless.", "You stink of failure.", "Idiot."},
		"curse":      {"Rot, then.", "Go and choke, {name}.", "Damn you and all your kin.", "May it all go wrong for you."},
		"oath":       {"I swear it, flat out.", "On my life. Straight.", "Truth, and I'll stand by it.", "My word. That's all there is."},
		"gossipOpen": {"Here's what I heard.", "Not that I care, but", "Word is, plain and simple,", "You want news? Fine:"},
	},
	"flowery": {
		"greet":      {"Ah, {name}! The day brightens at your coming.", "Welcome, welcome, light of the road! What brings you to my door?", "A blessing on your footsteps, {name}, and on the ground that bears them.", "Peace be upon you, honoured traveller, and upon all your house.", "Well met, well met! The very sight of you is a gift."},
		"farewell":   {"Go gently, {name}, and may the wind carry sweetness to you.", "Until the stars bring us together again.", "May every road you take be lined with fruit and shade.", "Peace go before you and peace come behind.", "Depart in beauty, {name}."},
		"thanks":     {"You have filled my heart like a cup, {name}.", "Words are small vessels for this much gratitude.", "May the gods repay your kindness a hundredfold.", "A thousand thanks, and a thousand more.", "Your goodness is a lamp in a dark room."},
		"apology":    {"I lay my shame before you like a mat, {name}. Forgive me.", "Forgive a foolish tongue, I beg you.", "My regret is deeper than the well, and as dark.", "Let the wind carry off my careless words."},
		"agree":      {"As surely as the sun rises, {name}, so it is.", "Your words are honey, and true honey at that.", "It is so, and it is well that it is so.", "I could not have said it better with a hundred tongues.", "Let it be, as you have spoken."},
		"refuse":     {"Alas, {name}, the river will not flow uphill for me.", "I would give you the moon, but this I cannot give.", "My heart says yes and my hands say no. Forgive me.", "It breaks me to refuse, yet refuse I must.", "Alas, it cannot be."},
		"praise":     {"Your deeds shine, {name}, like polished bronze in the morning.", "Truly a marvel! The songs will want you.", "The gods themselves must be envious.", "What grace, what skill! I am dazzled.", "You make the common day a feast."},
		"insult":     {"You, {name}, are a wilted weed in the garden of the world.", "A dung-beetle has more grace than you.", "Your face would curdle a river.", "May your tongue be as thin as your wit.", "Oh, what a sorry flower you are."},
		"curse":      {"May your fields bear only thistles, {name}.", "May the moon turn her face from you.", "May your well run dry and your song run sour.", "May the crows make their nest in your hearth."},
		"oath":       {"By the sun that sees all, it is true.", "I swear by every star, {name}.", "By the sweetness of my mother's bread, I speak truly.", "By the river and the mountain, I swear."},
		"gossipOpen": {"Oh, such a tale has come to my ears! They say that", "Lean closer, {name}, for the wind has whispered that", "A little bird, bright as a coin, sang to me that", "It is told in the courtyards, softly, that"},
	},
	"terse": {
		"greet":      {"Hail.", "Speak.", "Yes?", "Well?", "Here."},
		"farewell":   {"Go.", "Done.", "Fare well.", "Leave.", "Aye. Go."},
		"thanks":     {"Thanks.", "Owed.", "Fair.", "Gratitude.", "Good."},
		"apology":    {"My fault.", "Wrong. Mine.", "Mistake.", "Sorry."},
		"agree":      {"Yes.", "So.", "True.", "Good.", "Aye."},
		"refuse":     {"No.", "Not so.", "Leave it.", "Refused.", "Never."},
		"praise":     {"Good.", "Strong.", "Skilled.", "Respect.", "Well."},
		"insult":     {"Fool.", "Wretch.", "Liar.", "Nothing.", "Coward."},
		"curse":      {"Fall.", "Ash.", "Rot.", "Freeze.", "Gone."},
		"oath":       {"On my life.", "By iron.", "I swear.", "True.", "My word."},
		"gossipOpen": {"Heard:", "Word:", "Listen.", "Rumour:"},
	},
	"earthy": {
		"greet":      {"Well now, {name}. Come in, sit, don't be shy.", "Mornin', {name}. Mind the mud.", "Eh? Oh, it's yourself. What's afoot?", "Hullo there! Come to the fire.", "There you are, {name}. Been wondering what the dogs were barking at."},
		"farewell":   {"Mind how you go, {name}.", "Off with you, then. Watch the road.", "Well, I've turnips to see to. Safe home.", "Take care of your feet, {name}.", "Aye, go on. The kettle's waiting."},
		"thanks":     {"That's right kind, {name}.", "Bless you for it, I'll not forget.", "Good as bread, that.", "I'll stand you a drink for that one.", "Can't say fairer. Thanks."},
		"apology":    {"Ah, I'm a clumsy old ox, forgive me.", "Sorry, {name}. Spoke before I thought.", "That was daft of me, that was.", "My bad, friend. Won't do it twice."},
		"agree":      {"Aye, that's the way of it.", "Right enough, {name}.", "Couldn't say it better, and I've tried.", "That's sound as a good barrel.", "Aye. Aye."},
		"refuse":     {"Nay, I'll not, {name}.", "Not for all the cheese in the valley.", "Can't do it, friend. Not and sleep at night.", "You'd get more from a stone than from me on that.", "No, and that's flat."},
		"praise":     {"Now that's a proper job, {name}.", "Well, I'll be. Handsome work.", "You've the knack, no mistake.", "That'll be talked of at the well.", "Good on you."},
		"insult":     {"You great turnip, {name}.", "You've the brains of a boiled egg.", "Get on, you muck-spreader.", "Daft as a bucket of frogs, you are.", "Bah, a pox on your boots."},
		"curse":      {"May your milk sour and your roof leak.", "May the pigs eat your porridge.", "May you step in it every day of your life.", "May your hens all stop laying, {name}."},
		"oath":       {"As I stand here, it's true.", "By the soil under me, honest.", "Cross my heart and hope the barn burns if I lie.", "On my mother's grave, {name}."},
		"gossipOpen": {"You'll not believe it, but", "Over the fence I heard that", "Spilled with the soup, this:", "Heard it at the well, so it must be true:"},
	},
	"courtly": {
		"greet":      {"My compliments, {name}. You grace this place.", "Ah, {name}. How very agreeable to see you.", "Good day, and welcome, if you will permit the liberty.", "Your presence is noted, and appreciated, {name}.", "Charmed. Do tell me what I may do for you."},
		"farewell":   {"Until the next occasion, {name}. Do take care.", "I shall count the hours. Farewell.", "You will excuse me. Good day, and good fortune.", "Do give my regards to anyone who deserves them.", "A pleasure, as ever."},
		"thanks":     {"You are too kind, {name}. Truly.", "I am in your debt, and delighted to be.", "How very gracious. I shall not forget it.", "My gratitude, freely and handsomely given.", "It was well done, and I thank you."},
		"apology":    {"I do beg your pardon, {name}. I was careless.", "Please forgive the lapse. It will not recur.", "How unforgivably clumsy of me.", "I am quite mortified. My apologies."},
		"agree":      {"Precisely, {name}. I could not agree more.", "You have the right of it, my friend.", "Just so. Just so.", "Delightfully put, and entirely correct.", "I find myself in full accord."},
		"refuse":     {"I fear that is quite impossible, {name}.", "Regretfully, no. You understand.", "Alas, there are limits even to my generosity.", "It cannot be done, and I would rather say so honestly.", "I must decline, with all due courtesy."},
		"praise":     {"Splendid, {name}. Simply splendid.", "A performance worthy of the finest hall.", "How gracefully done.", "You do us all credit.", "Remarkable. One hardly knows what to say."},
		"insult":     {"How tiresome you are, {name}.", "I have met furniture with more wit.", "One would hardly expect better of your sort.", "Your manners do rather speak for themselves.", "Do leave. The air improves."},
		"curse":      {"I hope fortune forgets your address, {name}.", "May every door close softly in your face.", "May your invitations all be lost in the post.", "May your enemies be witty and your friends be few."},
		"oath":       {"Upon my honour, {name}, as a person of rank.", "I give you my solemn word.", "On my name and my house, I swear it.", "You have my pledge, sealed as it were."},
		"gossipOpen": {"Between ourselves, {name}, it appears that", "One does hear things, you know. For instance that", "The latest whisper in the salons is that", "I would not repeat it, but I am told that"},
	},
}

// Role lines (guard, priest, merchant, elder, child) mixed into a register's lines now and then.
var dvRoleBank = map[string]map[string][]string{
	"guard": {
		"greet":    {"Keep the peace and we'll get along.", "Halt. Ah, only a traveller. Go on.", "Mind the watch, stranger."},
		"farewell": {"Move along, then.", "Stay out of trouble.", "The watch is watching."},
		"refuse":   {"Not on my post.", "Orders. I can't.", "That is against the law, and I enforce it."},
	},
	"priest": {
		"greet":    {"Peace be with you, child of the road.", "The gods see you come and go. Be welcome.", "Blessings upon you, traveller."},
		"farewell": {"Go with the blessing.", "May the gods walk beside you.", "Peace on your path."},
		"refuse":   {"That is forbidden to us.", "The rite does not allow it.", "I cannot, and the gods would not wish me to."},
	},
	"merchant": {
		"greet":    {"Looking or buying? Either way, welcome.", "Fine goods, fair prices. Come see.", "Ah, a customer! What can I find you?"},
		"farewell": {"Come back when your purse is heavier.", "Safe journey, and spend wisely.", "Tell your friends where to shop."},
		"thanks":   {"Pleasure doing business.", "A pleasure, and a fair bargain for us both.", "Coin well spent, I promise it."},
		"refuse":   {"My prices are already fair.", "No credit. Don't ask again.", "Not for sale. Not at that."},
	},
	"elder": {
		"greet":    {"Sit, sit. The young are always in a hurry.", "Ah, a new face. I have seen many.", "Come, child of the road. What do you seek?"},
		"farewell": {"Go. You will understand it later.", "Walk well. I will watch from here.", "Remember what I said, if you can."},
		"refuse":   {"In my years I have learned not to, child.", "No. Some things are not for the asking.", "You will thank me for refusing."},
	},
	"child": {
		"greet":    {"Hi! Are you a real traveller?", "Mum says not to talk to strangers. Hello.", "Do you have anything to show me?"},
		"farewell": {"Bye! Come back!", "Will you visit again?", "I have to go play now."},
		"refuse":   {"Nuh-uh!", "No! You can't make me.", "I'll tell my father!"},
	},
}

// Warm kind clauses appended to greetings, farewells and thanks.
var dvWarmClauses = []string{"It is good to see you.", "You are always welcome.", "I hope the road was kind.", "Take care of yourself.", "I mean that."}

// Anxious hesitations placed in front of a line.
var dvHesitations = []string{"Um... ", "Ah, well... ", "I... yes. ", "Oh. Er... "}

// Gloomy openers for a speaker in a very low mood with no temper of their own.
var dvGlum = []string{"Hm. ", "*sigh* ", "Ah. "}

// One-liners per register, for dvQuip when the pack has no barks.
var dvQuips = map[string][]string{
	"formal":  {"Order, in all things, is the first kindness.", "One should always be punctual, even in small matters.", "The proper thing is rarely the easy thing.", "Every duty done is a debt paid."},
	"blunt":   {"Work doesn't do itself.", "Mind your business, I'll mind mine.", "If it's broken, fix it.", "Talk is cheap. Bread isn't."},
	"flowery": {"What a morning! Even the dust dances.", "The world is a garden, if one is willing to look.", "A song lives in every stone, if you listen.", "The sky wears her best colours today."},
	"terse":   {"Quiet day.", "Hm.", "Watching.", "Nothing to report."},
	"earthy":  {"Weather's turning, I can feel it in my knee.", "Never met a problem a good stew couldn't help.", "Honest work, honest bread.", "My grandmother said much the same."},
	"courtly": {"The company is lacking this season, I find.", "One does so tire of the same faces.", "Such a pity about the manners nowadays.", "I do love a well-turned phrase."},
}

// Sphere to register for packless worlds (unlisted spheres hash into the six).
var dvSphereRegister = map[string]string{
	"sinic": "formal", "japonic": "formal", "nahua": "formal", "maya": "formal", "andean": "formal",
	"indic": "flowery", "persian": "flowery", "arabic": "flowery", "ethiopic": "flowery",
	"steppe": "blunt", "slavic": "blunt", "nordic": "blunt", "celtic": "earthy",
	"mande": "earthy", "yoruba": "earthy", "bantu": "earthy", "polynesian": "earthy", "turtle": "earthy", "sea": "earthy",
	"frankish": "courtly", "latin": "courtly", "eldritch": "terse",
}

// Tag words for the look text, by temper then by register.
var dvTemperTag = map[string]string{
	"cold": "coldly", "gruff": "gruffly", "warm": "warmly", "anxious": "nervously",
	"proud": "loftily", "wry": "dryly", "eager": "eagerly", "calm": "calmly",
}
var dvRegisterTag = map[string]string{
	"formal": "formally", "blunt": "bluntly", "flowery": "floridly", "terse": "tersely",
	"earthy": "in a homely way", "courtly": "with courtly polish",
}
