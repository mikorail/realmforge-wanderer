package engine

// Data for city jobs (cityjobs.go): eleven job lines, their mid-shift scenes, genre skins,
// rumours and night-life lines. Prefix cj.

// cjChoice is one lettered option in the middle of a shift. Kind names a cj check (cjMod); an
// empty Kind is a sure thing. Pay and LosePay are percent changes to the wage.
type cjChoice struct {
	Label   string
	Kind    string
	DC      int
	Pay     int
	LosePay int
	Hurt    int
	Win     string
	Lose    string
	Fx      string // catch let stop rumour patron skim run
}

type cjScene struct {
	Intro string
	Ch    []cjChoice
}

type cjLine struct {
	ID     string
	Title  string // default title (fantasy); genre skins in cjTitles
	Where  string
	Hours  int
	When   string // day eve any night
	Skill  string
	Base   [3]int
	Intro  string
	Scenes []cjScene
	Perk   string // shop room meal
	Shady  bool
}

var cjRankNames = [3]string{"apprentice", "journeyman", "master"}

// Shifts needed for rank 1 and 2 (journeyman, master).
var cjRankAt = [2]int{4, 11}

// Practice counters needed for the next skill rank (index = current rank).
var cjSkillAt = [3]int{6, 16, 32}

var cjLines = []*cjLine{
	{ID: "dock", Title: "dockhand", Where: "the quay", Hours: 4, When: "day", Skill: "endurance", Base: [3]int{14, 22, 34}, Perk: "room",
		Intro: "You fall in with the {title}s and start shifting crates, bales and barrels along the quay.",
		Scenes: []cjScene{
			{Intro: "A stack of crates is loaded wrong and the gang waits for someone to take the heavy end.", Ch: []cjChoice{
				{Label: "Lift the heavy end yourself.", Kind: "lift", DC: dcMedium, Pay: 40, LosePay: -20, Hurt: 3, Win: "It goes up with a grunt. The gang whistles and the tally-man marks you for extra.", Lose: "Your back twists and the load drops. You limp through the rest of the hour."},
				{Label: "Rig a lever from a spar and rope.", Kind: "mend", DC: dcMedium, Pay: 25, LosePay: -10, Win: "The lever works and the crates roll. The gang is impressed.", Lose: "The spar slips and the rope burns your palms, but nothing breaks."},
				{Label: "Haul your own share and leave the rest.", Win: "You do an honest share and no more."},
			}},
			{Intro: "A crate splits on the wharf and something tinkles in the straw.", Ch: []cjChoice{
				{Label: "Report it to the tally-man.", Pay: 15, Win: "The tally-man nods and slips you a tip for honesty.", Fx: "patron"},
				{Label: "Quietly pocket a trinket.", Kind: "streetwise", DC: dcMedium, Pay: 30, LosePay: -60, Win: "Nobody sees. A trinket weighs down your pocket.", Lose: "A hand closes on your collar. The foreman docks your wage."},
				{Label: "Count the manifest against the cargo.", Kind: "count", DC: dcMedium, Pay: 30, Win: "You spot a short count and the foreman is glad of it.", Lose: "The numbers swim. Better to say nothing."},
			}},
		}},
	{ID: "stall", Title: "stall hand", Where: "the market", Hours: 4, When: "day", Skill: "persuasion", Base: [3]int{12, 20, 32}, Perk: "shop",
		Intro: "You set out the wares, mind the awning and call the day's prices for the merchant.",
		Scenes: []cjScene{
			{Intro: "A tight-fisted customer wants a better price than the board says.", Ch: []cjChoice{
				{Label: "Haggle: hold the price and charm them.", Kind: "haggle", DC: dcMedium, Pay: 45, LosePay: -20, Win: "They pay in full and leave a tip for the patter.", Lose: "They storm off and the merchant frowns."},
				{Label: "Give a little and keep the sale.", Pay: 5, Win: "A small sale, a satisfied customer."},
				{Label: "Count the till before the rush.", Kind: "count", DC: dcEasy, Pay: 25, Win: "Your tally catches a miscount in the merchant's favour.", Lose: "You lose count and begin again."},
			}},
			{Intro: "A richly dressed patron lingers over the best goods and asks quiet questions.", Ch: []cjChoice{
				{Label: "Show them the good stock and talk it up.", Kind: "haggle", DC: dcChallenging, Pay: 50, LosePay: 0, Win: "They buy handsomely, and murmur a word about a place worth visiting.", Lose: "They hum and leave without buying.", Fx: "patron"},
				{Label: "Watch the stall for sticky fingers.", Kind: "patrol", DC: dcEasy, Pay: 20, Win: "You catch a boy reaching and he scampers off.", Lose: "A trinket vanishes while you look elsewhere."},
				{Label: "Mind your own business.", Win: "Nothing happens."},
			}},
		}},
	{ID: "forge", Title: "forge hand", Where: "the forge", Hours: 4, When: "day", Skill: "tinkering", Base: [3]int{15, 24, 38}, Perk: "shop",
		Intro: "You work the bellows, quench blades and carry stock between the fire and the anvil.",
		Scenes: []cjScene{
			{Intro: "A customer's broken part comes in and the smith nods at you: 'Fix it.'", Ch: []cjChoice{
				{Label: "Mend it with careful fitting.", Kind: "mend", DC: dcMedium, Pay: 45, LosePay: -15, Hurt: 2, Win: "It fits true. The smith checks it and grunts approval.", Lose: "The hot metal bites your wrist as it slips."},
				{Label: "Reforge it from scratch.", Kind: "lift", DC: dcChallenging, Pay: 60, LosePay: -20, Hurt: 3, Win: "A fine piece comes out of the fire, better than the old.", Lose: "You overheat the steel and it cracks."},
				{Label: "Hand it back to the smith.", Win: "The smith takes over with a sigh."},
			}},
			{Intro: "The coals are low and the order list is long.", Ch: []cjChoice{
				{Label: "Work the bellows hard.", Kind: "grit", DC: dcMedium, Pay: 30, LosePay: -10, Hurt: 1, Win: "The fire roars and the day's orders are met.", Lose: "Your lungs burn and you have to sit down."},
				{Label: "Sort the stock by size and use.", Kind: "count", DC: dcEasy, Pay: 25, Win: "A tidy stock room speeds the work.", Lose: "You mix two bins and lose an hour."},
				{Label: "Sweep and fetch water.", Win: "Humble work, honest pay."},
			}},
		}},
	{ID: "farm", Title: "farmhand", Where: "the fields", Hours: 5, When: "day", Skill: "survival", Base: [3]int{12, 19, 29}, Perk: "meal",
		Intro: "You rake, hoe and carry baskets between the rows under the open sky.",
		Scenes: []cjScene{
			{Intro: "Clouds mass on the horizon and the last rows still stand unharvested.", Ch: []cjChoice{
				{Label: "Race the weather and pick fast.", Kind: "grit", DC: dcMedium, Pay: 40, LosePay: -20, Hurt: 2, Win: "You bring the last of it in just as the first drops fall.", Lose: "The rain catches you and half the crop is lost."},
				{Label: "Read the sky and finish the high rows only.", Kind: "tend", DC: dcMedium, Pay: 25, Win: "Your guess is right and nothing is lost.", Lose: "You guess wrong and the low rows are soaked."},
				{Label: "Work steadily and let the weather come.", Win: "A steady day's work."},
			}},
			{Intro: "A beast has broken into the pen and the animals are panicking.", Ch: []cjChoice{
				{Label: "Calm the animals.", Kind: "tend", DC: dcMedium, Pay: 35, LosePay: -10, Win: "They settle under your hands and the farmer thanks you.", Lose: "A hoof catches you in the shin."},
				{Label: "Drive the intruder off with a staff.", Kind: "patrol", DC: dcMedium, Pay: 35, LosePay: -10, Hurt: 2, Win: "The intruder bolts over the hedge.", Lose: "It gets past you and tramples a row."},
				{Label: "Call the farmer.", Win: "The farmer sorts it himself."},
			}},
		}},
	{ID: "patrol", Title: "watch recruit", Where: "the guardhouse", Hours: 5, When: "any", Skill: "cudgel", Base: [3]int{16, 26, 40}, Perk: "meal",
		Intro: "You pull on a tabard and walk the {title}'s beat with the watch.",
		Scenes: []cjScene{
			{Intro: "A pickpocket's hand is in a merchant's purse, and only you have seen it.", Ch: []cjChoice{
				{Label: "Catch the thief by the wrist.", Kind: "patrol", DC: dcMedium, Pay: 35, LosePay: -10, Fx: "catch", Win: "You seize the culprit. The merchant weeps with relief and the sergeant marks you.", Lose: "The thief wriggles loose and is gone in the crowd."},
				{Label: "Let them go and look away.", Pay: 0, Fx: "let", Win: "You turn your head. The thief is gone."},
				{Label: "Call out a warning to the merchant.", Pay: 15, Win: "The merchant clutches the purse and the thief slinks off."},
			}},
			{Intro: "A brawl erupts outside a tavern, fists and stools flying.", Ch: []cjChoice{
				{Label: "Wade in and stop it.", Kind: "grit", DC: dcMedium, Pay: 40, LosePay: -10, Hurt: 3, Fx: "stop", Win: "You break it up with a few firm words and a firm hand.", Lose: "A stool catches you across the shoulders."},
				{Label: "Blow the whistle and hold the line.", Kind: "patrol", DC: dcEasy, Pay: 20, Win: "Help arrives and the brawlers scatter.", Lose: "No one comes and the fight burns itself out."},
				{Label: "Wait it out from a distance.", Win: "It ends with sore heads and no arrests."},
			}},
			{Intro: "Two night-folk argue in low voices and one of them lets a name slip.", Ch: []cjChoice{
				{Label: "Linger and listen.", Kind: "patrol", DC: dcMedium, Pay: 10, Fx: "rumour", Win: "You catch enough to be worth remembering.", Lose: "They notice you and fall silent."},
				{Label: "Move them along.", Pay: 5, Win: "They shuffle off."},
				{Label: "Walk on.", Win: "A quiet beat."},
			}},
		}},
	{ID: "clerk", Title: "scribe", Where: "the scriptorium", Hours: 4, When: "day", Skill: "lore", Base: [3]int{16, 25, 38}, Perk: "shop",
		Intro: "You copy ledgers, charters and manifests in a clear hand, hour after hour.",
		Scenes: []cjScene{
			{Intro: "The ledger of last season's accounts does not add up.", Ch: []cjChoice{
				{Label: "Count it all again, column by column.", Kind: "count", DC: dcMedium, Pay: 45, LosePay: -10, Win: "You find the slip. The clerk looks at you with new respect.", Lose: "The numbers refuse to agree."},
				{Label: "Copy it as it stands.", Win: "A faithful copy of a flawed ledger."},
				{Label: "Read the margin notes for the story.", Kind: "count", DC: dcChallenging, Pay: 20, Fx: "rumour", Win: "Between the lines is a bit of town gossip.", Lose: "The hand is too cramped to read."},
			}},
			{Intro: "A sealed letter crosses your desk and the seal looks wrong.", Ch: []cjChoice{
				{Label: "Report the forgery.", Kind: "count", DC: dcMedium, Pay: 35, Win: "The clerk takes it from you, grim and grateful.", Lose: "You are not sure after all."},
				{Label: "Copy it and say nothing.", Win: "You copy what you were given."},
				{Label: "Look closer at the ink and paper.", Kind: "mend", DC: dcChallenging, Pay: 45, Win: "You can tell where it was made, and by whom.", Lose: "It is only ink and paper to you."},
			}},
		}},
	{ID: "apoth", Title: "apothecary's helper", Where: "the apothecary", Hours: 4, When: "day", Skill: "lore", Base: [3]int{15, 24, 37}, Perk: "shop",
		Intro: "You grind herbs, boil linens and fill small bottles for the counter.",
		Scenes: []cjScene{
			{Intro: "A batch of tincture is half done and the master is called away.", Ch: []cjChoice{
				{Label: "Brew it to the recipe.", Kind: "brew", DC: dcMedium, Pay: 40, LosePay: -20, Win: "It clears, true and bright. The master tests it and nods.", Lose: "The batch clouds and has to be thrown out."},
				{Label: "Let it simmer and watch it.", Win: "A slow and safe day."},
				{Label: "Try a stronger variation.", Kind: "brew", DC: dcChallenging, Pay: 60, LosePay: -30, Hurt: 2, Win: "The stronger brew is a triumph.", Lose: "Fumes catch in your throat."},
			}},
			{Intro: "A feverish child is carried in and the master is out.", Ch: []cjChoice{
				{Label: "Mix a poultice from what you know.", Kind: "brew", DC: dcMedium, Pay: 35, Win: "The fever eases and the mother weeps her thanks.", Lose: "It does little. The master finishes the work.", Fx: "patron"},
				{Label: "Send a runner for the master.", Pay: 10, Win: "The master returns in time."},
				{Label: "Cool the child with water.", Kind: "tend", DC: dcEasy, Pay: 20, Win: "It helps, a little.", Lose: "The child cries on."},
			}},
		}},
	{ID: "stable", Title: "stablehand", Where: "the stable", Hours: 4, When: "day", Skill: "survival", Base: [3]int{13, 21, 31}, Perk: "room",
		Intro: "You muck out stalls, fork hay and brush down the beasts of travellers.",
		Scenes: []cjScene{
			{Intro: "A skittish mount will not be led out.", Ch: []cjChoice{
				{Label: "Gentle it with patience and a soft voice.", Kind: "tend", DC: dcMedium, Pay: 35, LosePay: -10, Hurt: 2, Win: "It follows you out as if led by an old friend.", Lose: "It rears and strikes you on the arm."},
				{Label: "Use a firm hand and a rope.", Kind: "lift", DC: dcMedium, Pay: 30, LosePay: -10, Hurt: 2, Win: "It gives in, snorting.", Lose: "The rope tears through your hands."},
				{Label: "Leave it for the master of horse.", Win: "The master of horse calms it with a word."},
			}},
			{Intro: "A rider hands you a purse and asks for their horse to be ready before dawn.", Ch: []cjChoice{
				{Label: "Do it properly: feed, brush, tack.", Kind: "tend", DC: dcEasy, Pay: 30, Win: "The rider tips you for the care.", Lose: "You rush it and forget the girth."},
				{Label: "Pocket the coin and do the minimum.", Win: "Good enough, and nothing more."},
				{Label: "Ask what they know of the road.", Kind: "haggle", DC: dcMedium, Pay: 10, Fx: "rumour", Win: "They tell you of a hazard on the road.", Lose: "They keep their counsel."},
			}},
		}},
	{ID: "mine", Title: "miner", Where: "the pit-head", Hours: 5, When: "day", Skill: "endurance", Base: [3]int{17, 27, 42}, Perk: "meal",
		Intro: "You swing a pick, fill carts and shore the roof with the shift crew.",
		Scenes: []cjScene{
			{Intro: "The face runs rich, but the roof ticks overhead.", Ch: []cjChoice{
				{Label: "Dig on and take the risk.", Kind: "grit", DC: dcChallenging, Pay: 55, LosePay: -20, Hurt: 4, Win: "A rich seam comes free before the roof settles.", Lose: "Stone drops and bruises you badly."},
				{Label: "Shore the roof first.", Kind: "mend", DC: dcMedium, Pay: 30, Win: "The timbers hold and the crew is pleased.", Lose: "The timbers slip but no harm is done."},
				{Label: "Move to a safer face.", Win: "Safe and slow."},
			}},
			{Intro: "A cart-wheel jams on the tracks with a full load behind it.", Ch: []cjChoice{
				{Label: "Heave the cart clear.", Kind: "lift", DC: dcMedium, Pay: 40, LosePay: -10, Hurt: 3, Win: "It scrapes free and the line runs again.", Lose: "You wrench your back."},
				{Label: "Fix the wheel with spare parts.", Kind: "mend", DC: dcMedium, Pay: 35, Win: "It spins true again.", Lose: "The part fits badly and the foreman grunts."},
				{Label: "Call the foreman.", Win: "The foreman curses and fixes it himself."},
			}},
		}},
	{ID: "tavern", Title: "tapster", Where: "the taproom", Hours: 5, When: "eve", Skill: "persuasion", Base: [3]int{13, 21, 33}, Perk: "room",
		Intro: "You pour, carry and clear tables as the taproom fills.",
		Scenes: []cjScene{
			{Intro: "A sour-faced drinker disputes the bill and his friends begin to murmur.", Ch: []cjChoice{
				{Label: "Haggle: settle it with charm.", Kind: "haggle", DC: dcMedium, Pay: 40, LosePay: -20, Win: "They laugh, pay up and tip you besides.", Lose: "You split the difference and no one is happy."},
				{Label: "Call for the keeper.", Win: "The keeper settles it."},
				{Label: "Throw them out.", Kind: "lift", DC: dcChallenging, Pay: 25, LosePay: -20, Hurt: 2, Win: "A stout push and they are in the street.", Lose: "A flung stool hits your shoulder."},
			}},
			{Intro: "A cloaked patron at the corner table waves you over and leaves a purse on the board.", Ch: []cjChoice{
				{Label: "Serve them with the best you have.", Kind: "haggle", DC: dcMedium, Pay: 40, Win: "They talk, you listen, and the tip is a good one.", Lose: "They finish and leave without a word.", Fx: "patron"},
				{Label: "Keep your head down and pour.", Win: "A long quiet night."},
				{Label: "Listen to the room's chatter.", Kind: "patrol", DC: dcMedium, Pay: 10, Fx: "rumour", Win: "You hear something worth keeping.", Lose: "Too much noise to hear any of it."},
			}},
		}},
	{ID: "back", Title: "back-room runner", Where: "the back room", Hours: 3, When: "night", Skill: "acrobatics", Base: [3]int{22, 34, 52}, Shady: true,
		Intro: "The curtain drops behind you. You count, carry and keep your mouth shut.",
		Scenes: []cjScene{
			{Intro: "A parcel must cross the district before the watch changes.", Ch: []cjChoice{
				{Label: "Run it through the alleys.", Kind: "street", DC: dcMedium, Pay: 50, LosePay: -50, Fx: "run", Win: "You slip past a lantern and no one sees.", Lose: "A watchman's lantern swings your way."},
				{Label: "Carry it openly like an errand-boy.", Kind: "haggle", DC: dcEasy, Pay: 25, Fx: "run", Win: "No one looks twice at a boy with a box.", Lose: "A watchman asks and you stammer."},
				{Label: "Pass it to another runner.", Win: "The runner takes it, and a cut of your wage."},
			}},
			{Intro: "The till of a night's takings sits in front of you. The count is the count.", Ch: []cjChoice{
				{Label: "Count it honestly.", Kind: "count", DC: dcEasy, Pay: 30, Win: "To the last coin. They notice.", Lose: "You lose count and start again."},
				{Label: "Skim a little off the top.", Kind: "street", DC: dcChallenging, Pay: 70, LosePay: -80, Hurt: 2, Fx: "skim", Win: "A few coins slip into your sleeve. No one is the wiser.", Lose: "A hand catches your sleeve. They remind you of the rules."},
				{Label: "Mind your own business.", Win: "A quiet shift."},
			}},
		}},
}

// cjTitles: genre skins for the job titles (porter becomes a hoverboard courier, scribe a data
// clerk, farmhand a vat tender, ...). A missing entry keeps the fantasy title.
var cjTitles = map[string]map[string]string{
	"cyberpunk":  {"dock": "hoverboard courier", "stall": "kiosk runner", "forge": "ripperdoc's tech", "farm": "hydroponics tech", "patrol": "corpsec trainee", "clerk": "data clerk", "apoth": "pharm runner", "stable": "garage grease-monkey", "mine": "drill-rig hand", "tavern": "bar-back", "back": "fixer's runner"},
	"steampunk":  {"dock": "crane-hand", "stall": "counter clerk", "forge": "machinist's mate", "farm": "steam-plough hand", "patrol": "constable's mate", "clerk": "ledger clerk", "apoth": "chemist's assistant", "stable": "boiler-stoker", "mine": "pit-engine stoker", "tavern": "pot-boy", "back": "gaslight runner"},
	"dieselpunk": {"dock": "stevedore", "stall": "shop clerk", "forge": "assembly hand", "farm": "tractor hand", "patrol": "beat patrolman", "clerk": "typist", "apoth": "druggist's boy", "stable": "motor-pool hand", "mine": "ore-carter", "tavern": "barman's mate", "back": "numbers runner"},
	"atompunk":   {"dock": "freight loader", "stall": "soda-counter hand", "forge": "gunsmith's helper", "farm": "silo hand", "patrol": "deputy for a shift", "clerk": "records clerk", "apoth": "pharmacy aide", "stable": "garage hand", "mine": "uranium-pit hand", "tavern": "soda-jerk", "back": "back-room courier"},
	"biopunk":    {"dock": "beast-porter", "stall": "gene-stall hand", "forge": "flesh-sculptor's aide", "farm": "vat tender", "patrol": "warden's trainee", "clerk": "gene-registry clerk", "apoth": "gene-brewer's aide", "stable": "beast-handler", "mine": "marrow-digger", "tavern": "brew-vat hand", "back": "culture runner"},
	"solarpunk":  {"dock": "cargo-glider hand", "stall": "commons stall hand", "forge": "panel fitter", "farm": "garden steward", "patrol": "commons warden", "clerk": "archive keeper", "apoth": "herbalist's aide", "stable": "mount-groom", "mine": "reclaim-crew hand", "tavern": "cider-pourer", "back": "swap-market runner"},
	"wasteland":  {"dock": "scrap hauler", "stall": "trade-post hand", "forge": "scrap-smith's mate", "farm": "mutant-crop picker", "patrol": "wall-watch recruit", "clerk": "tally-keeper", "apoth": "still-tender", "stable": "mount-wrangler", "mine": "salvage-pit digger", "tavern": "still-hand", "back": "stash runner"},
	"mythic":     {"dock": "quay porter", "stall": "agora stall hand", "forge": "bronze-caster", "farm": "olive picker", "patrol": "city-guard recruit", "clerk": "temple scribe", "apoth": "root-cutter's aide", "stable": "horse-groom", "mine": "silver-miner", "tavern": "wine-pourer", "back": "shadow runner"},
	"twilight":   {"dock": "ferry-hand", "stall": "dusk-market hand", "forge": "ember-keeper", "farm": "blight-picker", "patrol": "lantern-watch recruit", "clerk": "chronicler's copyist", "apoth": "wardbrew's aide", "stable": "pale-mount groom", "mine": "pit-delver", "tavern": "candle-tapster", "back": "whisper runner"},
}

// cjWatchTitles: the night watch by genre.
var cjWatchTitles = map[string]string{
	"cyberpunk": "night-shift corpsec", "steampunk": "lamplighter constable", "dieselpunk": "night patrolman",
	"atompunk": "night deputy", "biopunk": "glow-lantern warden", "solarpunk": "night warden",
	"wasteland": "wall sentry", "mythic": "night watchman", "twilight": "lantern watchman",
}

var cjWatchLines = []string{
	"\"Quiet so far. The harbour's the place for trouble after dark.\"",
	"\"Keep your blade sheathed and your purse close, friend. The watch sees all after dark.\"",
	"\"Cutpurses work the market lanes at night. Mind your coin.\"",
	"\"There is a fence who moves goods at the docks when the lamps are low. We know, but not the face.\"",
	"\"Curfew is curfew. Drawn steel on the street means a fine in a big town.\"",
	"\"Something walks the graveyard when the moon is out. Not my beat.\"",
}

var cjDrunkLines = []string{
	"\"Hic... they say the baker works through the night and sells the best loaf in town at dawn.\"",
	"\"Ye want to know a secret? The watch takes bribes... only small ones, mind.\"",
	"\"Did you see her? The ghost by the old graves? I swear she waved at me.\"",
	"\"There's a market after dark, down by the water. Don't tell the watch I told you.\"",
	"\"I owe three taverns and a fence. Don't ask which fence.\"",
	"\"Cheers, stranger! Another round and I'll tell you where the old tunnels run.\"",
}

var cjBeggarLines = []string{
	"\"Bless you. Nobody gives at night, but I see everything. The cutpurse works by the lamp-post.\"",
	"\"Kind of you. Here is a thing I know: the street healer takes coin but never turns away the poor.\"",
	"\"May luck follow you. The watch looks away from the docks around the third hour.\"",
}

var cjGhostLines = []string{
	"\"I lie where the lamp does not reach. They forgot my name.\"",
	"\"There is a hollow under the old wall. They hid something there, long ago.\"",
	"\"The living walk fast at night. They are right to.\"",
	"\"I counted the town's bells, once. Now I count its sleepers.\"",
}

// cjRumours: texts for addSecret ("overheard" while on a shift or at night).
var cjRumours = []string{
	"A night market sets up by the water after dark. Its stalls sell what the day shops will not.",
	"The watch changes at midnight and the alleys behind the market are unguarded for a quarter hour.",
	"A fence moves goods through the docks when the lamps are low. A password gets you in: say nothing and nod twice.",
	"Cutpurses work the market lanes by night, picking out whoever flashes coin.",
	"A baker in the lower town works through the night, and sells the first loaves to people who know to knock.",
	"The street healer who walks the slums at night never asks for a fee up front.",
	"The foreman on the quay is short-handed since two of his men were arrested for pilfering.",
	"A merchant is bribing the watch to look away from a certain warehouse.",
}
