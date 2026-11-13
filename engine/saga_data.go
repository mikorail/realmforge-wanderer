package engine

// Data tables for the end-of-life saga (prefix sg): chronicler voices by genre and culture sphere,
// sentence pools per chapter theme, epithets and motifs from traits, epitaph lines, quotes.
// Templates use {slot} markers; a template whose slot is empty is skipped (saga.go sgCompose).

type sgVoice struct {
	Role   string   // who tells it
	Noun   string   // "Saga", "Lay" ...
	Medium string   // how it is kept
	Open   []string // opening formulas for the prologue ({n} {role} {home})
	Close  []string // closing formulas for the epilogue
}

// sgGenreVoice: the default chronicler of each genre.
var sgGenreVoice = map[string]sgVoice{
	"fantasy":    {"chronicler", "Chronicle", "set down in the long book", []string{"Let it be written, in the book kept for such things, how {n} came to be.", "Here the {role} begins the account of {n}, as it was gathered from those who knew."}, []string{"So ends the chronicle, and the ink is dry.", "Thus the {role} closes the book on {n}."}},
	"mythic":     {"rhapsode", "Rhapsody", "sung at the festival fires", []string{"Sing, memory, of {n}, who came out of {home} and would not stay.", "Of {n} I sing, as the rhapsodes sing, with the god-light on the water."}, []string{"So the rhapsode lowers the lyre, and the fire takes the rest.", "Let the gods keep what the song cannot hold."}},
	"steampunk":  {"gazetteer correspondent", "Gazette", "printed in the evening edition", []string{"From our own correspondent: the life of {n}, set out for the record.", "The Gazette is obliged to report the career of {n}, lately of {home}."}, []string{"Further particulars will appear in the next number.", "This concludes our correspondent's report, filed by telegraph."}},
	"dieselpunk": {"newsreel narrator", "Newsreel", "cut for the picture houses", []string{"The camera finds {n}, out of {home}, with the road still on the boots.", "Tonight's feature: {n}, and the miles that made a name."}, []string{"And that, friends, is the picture. Good night.", "The reel runs out, and the lamp goes dark."}},
	"atompunk":   {"broadcast announcer", "Broadcast", "carried on the evening wave", []string{"Good evening, listeners. Tonight we remember {n}, of {home}.", "Stand by for a special report on the life of {n}."}, []string{"This has been a special broadcast. Stay tuned.", "We now return you to the regular signal."}},
	"cyberpunk":  {"netcaster", "Feed", "mirrored on a thousand dead channels", []string{"Signal up. This is the feed of {n}, rebuilt from logs, rumours and what the cameras caught.", "Pull the thread on {n}, and here is the whole weave, unredacted."}, []string{"End of feed. The archive keeps what the market forgot.", "Mirror this. They will try to scrub it."}},
	"biopunk":    {"spore-singer", "Bloom-Song", "sung into the hive-wall", []string{"The wall remembers, and I sing what it remembers: {n}, of {home}.", "Listen, clutch and cell: the song of {n} begins in the soil."}, []string{"So the song sinks into the wall, and the wall will grow it again.", "Let the mycelium carry the rest."}},
	"solarpunk":  {"commons-bard", "Commons Tale", "told at the shared table", []string{"Gather round the shared table, for this is the tale of {n}, as the commons tell it.", "Pass the bowl, and I will tell you how {n} walked out of {home}."}, []string{"So the tale goes back into the commons, free for anyone to tell again.", "Plant it where it will grow."}},
	"wasteland":  {"vault-scribe", "Ledger", "scratched into the vault ledger", []string{"Entry opened, vault ledger: the account of {n}, of {home}, as witnessed and as reported.", "For the ledger, in the scribe's own hand: {n}, a name worth the ink."}, []string{"Entry closed. The ledger is sealed and the dust may settle.", "Filed, witnessed and sealed against the dust."}},
	"twilight":   {"grave-reader", "Vigil", "read aloud at the wake", []string{"At the wake we read what the candle shows: the life of {n}, of {home}.", "The grave-reader opens the vigil with the name of {n}, and the candles lean closer."}, []string{"The last candle gutters. The vigil is kept.", "Let the dusk keep them, as it keeps all things."}},
}

var sgDefaultVoice = sgVoice{"chronicler", "Chronicle", "set down in the long book", []string{"Here begins the account of {n}, of {home}.", "This is the telling of {n}."}, []string{"So ends the telling.", "Here the account is closed."}}

// sgSphereVoice overlays the genre voice for the older genres (fantasy, mythic, twilight, solarpunk, biopunk):
// the chronicler of the hero's own culture.
type sgSphereV struct {
	Role, Noun string
	Open       []string
}

var sgSphereVoice = map[string]sgSphereV{
	"norse":      {"skald", "Saga", []string{"Hear, then, how {n} fared out of {home}, and what the norns wove for them.", "A saga I have, and the mead is poured: of {n}, who walked far."}},
	"celtic":     {"bard", "Lay", []string{"Sit by the fire and let the bard speak: of {n}, of {home}, and the roads of the hollow hills.", "I have the lay of {n} by heart, as my teacher had it."}},
	"latin":      {"annalist", "Annals", []string{"Be it recorded in the annals that {n} came of {home}, in a year not unremarked.", "The annalist sets down the deeds of {n}, without ornament, as is proper."}},
	"arabic":     {"rawi", "Tale", []string{"It is related, and God knows best, that there lived one called {n}, of {home}.", "The rawi begins: they say, and the saying is old, that {n} walked out of {home}."}},
	"persian":    {"dastan-teller", "Dastan", []string{"In the old manner I tell it: the dastan of {n}, who left {home} behind.", "O listeners, the tale of {n} is a garden with many gates."}},
	"indic":      {"katha-reciter", "Katha", []string{"Let the katha be recited: of {n}, born at {home}, and the long wheel of their days.", "Hear, O hearers, the story of {n}, as it is remembered."}},
	"sinic":      {"court historian", "Annals", []string{"The historian records, as the office requires: {n}, of {home}, and what became of them.", "In the plain manner of the archive: the life of {n}."}},
	"japonic":    {"monogatari-reciter", "Monogatari", []string{"A tale is told, as tales are, of {n} from {home}, and of the seasons that wore them.", "Like leaves on the water, these are the days of {n}."}},
	"nahua":      {"tlacuilo", "Codex", []string{"Thus it is painted, and thus it is told: {n}, out of {home}.", "In the book of painted days the figure of {n} is drawn walking."}},
	"maya":       {"day-keeper scribe", "Count of Days", []string{"By the count of days it is carved: the life of {n}, of {home}.", "The scribe cuts the glyph of {n} into the stone, and the days follow."}},
	"andean":     {"knot-keeper", "Quipu", []string{"The knots are read, one by one, and they say: {n}, of {home}.", "Pull the cord of {n} through your fingers, and the years come loose."}},
	"yoruba":     {"oriki-singer", "Oriki", []string{"I call the praise-name of {n}, child of {home}, and the drums answer.", "Let the drum speak the lineage of deeds that is {n}."}},
	"mande":      {"djeli", "Epic", []string{"I am the djeli, and the memory of the people is in my mouth: hear of {n}, of {home}.", "Listen, for I tell it as my father's fathers told it: {n}."}},
	"bantu":      {"village storyteller", "Telling", []string{"Come near the fire, children, and hear of {n}, who came out of {home}.", "In the evening the elders say it so: {n}."}},
	"ethiopic":   {"debtera", "Gadl", []string{"In the name of the Light, I write the acts of {n}, of {home}, as the church writes its saints.", "Here is set out the gadl, the struggle, of {n}."}},
	"steppe":     {"epic-singer", "Epic", []string{"The horse-head fiddle sings of {n}, who left {home} for the sky of the grass.", "Wind and grass, listen: the long song of {n}."}},
	"polynesian": {"navigator-chanter", "Chant", []string{"I chant the voyage of {n}, from {home}, by the stars that did not lie.", "The canoe of this telling is launched: {n} at the steering oar."}},
	"inuit":      {"elder tale-teller", "Telling", []string{"When the dark is long, we tell of {n}, of {home}, and how they fared in the cold.", "Sit close. The old story of {n} needs no light."}},
	"slavic":     {"gusli-player", "Byliny", []string{"Strike the strings: the byliny of {n}, of {home}, in the old measure.", "Over the wide land, over the long road, goes the song of {n}."}},
	"sea":        {"shanty-master", "Ballad", []string{"Haul away, then, for a ballad of {n}, who shipped out of {home}.", "I give you the ballad of {n}, to be sung at the capstan."}},
	"eldritch":   {"dreaming scribe", "Dream-Record", []string{"The record is dreamt, and written on waking: {n}, of {home}, whom the deep remembers.", "What is written here was not entirely written by a hand: {n}."}},
	"turtle":     {"slow-keeper", "Long Count", []string{"The slow-keeper counts it in the long way: {n}, of {home}.", "Take your time with this one: {n}."}},
}

// sgSphereGenres are the genres whose chronicler follows the culture sphere.
var sgSphereGenres = map[string]bool{"fantasy": true, "mythic": true, "twilight": true, "solarpunk": true, "biopunk": true}

// ---- epithets and motifs from traits ----

var sgTraitEpithet = map[string]string{
	"brutal": "the Red-Handed", "merciful": "the Merciful", "greedy": "the Gold-Hoarder", "pious": "the Devout", "wanderer": "the Far-Walker",
	"scholar": "the Lettered", "glutton": "the Hearty", "teetotal": "the Sober", "nightowl": "the Night-Walker", "hydrophile": "the Well-Finder",
	"iconoclast": "the God-Breaker", "mourner": "the Mourner", "companionable": "the Fellow-Traveller", "gregarious": "the Well-Spoken",
	"delver": "the Deep-Delver", "seafarer": "the Salt-Walker", "giantkiller": "the Giant-Killer", "survivor": "the Unkillable",
	"murderer": "the Cold", "ascetic": "the Hollow-Purse", "stormtouched": "the Storm-Touched", "haunted": "the Haunted",
}

var sgTraitMotif = map[string]string{
	"brutal": "the smell of blood", "merciful": "an open hand", "greedy": "the weight of coin", "pious": "the small lit altar", "wanderer": "the road itself",
	"scholar": "the margins of old books", "glutton": "a full table", "teetotal": "a clear head", "nightowl": "the dark hours", "hydrophile": "the sound of clean water",
	"iconoclast": "broken idols", "mourner": "the empty place at the fire", "companionable": "the shared fire", "gregarious": "the noise of strangers' laughter",
	"delver": "the cold breath of stone", "seafarer": "salt on the wind", "giantkiller": "the shadow of large things", "survivor": "one breath from the end",
	"murderer": "the thing that cannot be taken back", "ascetic": "an empty purse", "stormtouched": "weather that meant harm", "haunted": "something that followed",
}

// sgKindEpithet: when no trait fits, from what the life was mostly made of (deed kind counts).
var sgKindEpithet = map[string]string{"slay": "the Slayer", "relic": "the Relic-Finder", "discovery": "the Seeker", "kinship": "the Beloved", "rival": "the Contender", "nemesis": "the Hunted", "quest": "the Oath-Keeper", "endgame": "the World-Turner", "ship": "the Sea-Roamer", "companion": "the Beast-Friend"}

// ---- quotes: what a person remembers of you, by the kind of memory ----

var sgQuoteByKind = map[string][]string{
	"gift":      {"They gave when they owed me nothing, and I have not forgotten it.", "A gift, out of nowhere, and no hook in it."},
	"help":      {"When the rest looked away, they stayed and helped.", "They did not have to help. They did."},
	"saved":     {"I am alive because they came when they came.", "Say what you like of them; they pulled me out."},
	"spared":    {"They could have ended me and did not.", "I stood at the edge of the blade, and they lowered it."},
	"quest":     {"They finished what they promised, which is rarer than gold.", "They took the task and carried it to the end."},
	"trade":     {"A fair dealer. Count your fingers anyway.", "They haggled hard and paid what they owed."},
	"insult":    {"They spoke to me as if I were dirt.", "A sharp tongue, that one, and a long memory."},
	"theft":     {"They took from me and never looked back.", "My purse is lighter for having met them."},
	"attack":    {"They raised a hand against me, and I have the mark to show.", "I remember the blow before I remember the face."},
	"kill":      {"They killed one of ours, and no road will be long enough.", "Blood on their hands, and not for the first time."},
	"witnessed": {"I saw what they did, and I will say no more.", "I was there. It was not small."},
	"talk":      {"They sat and listened, which few do.", "A good talker, and a better listener."},
	"met":       {"We spoke once, and I remembered it afterwards.", "They came through here and left the place changed by a degree."},
}

var sgQuoteByOpinion = [][2]any{{-60, "I would not shed a tear."}, {-20, "Their name leaves a taste behind."}, {20, "Nothing much to say; they passed through."}, {55, "A good one, that. I would hear the name again."}, {101, "There was no one like them, and I will say so to anyone."}}

// ---- chapter titles and sentence pools ----

var sgTitles = map[string][]string{
	"wander": {"The First Roads", "Dust and Small Coin", "Before the Name"},
	"first":  {"The First Great Deed", "When the Name Was Spoken", "A Mark Upon the Land"},
	"foe":    {"The Shadow on the Road", "A Name to Match Theirs", "The Whetstone"},
	"love":   {"Fire, Kin and Grief", "What the Road Gave and Took", "The Hearth and the Empty Place"},
	"trial":  {"The Great Trial", "When the World Leaned", "The Hour of Choosing"},
	"road":   {"The Road Widens", "Years of the Long Walk", "Further Than the Maps"},
	"end":    {"The End of the Road", "The Last Days", "What Remained"},
}

type sgPool struct{ Open, Mid, Close []string }

var sgPools = map[string]sgPool{
	"prologue": {
		Open: []string{"{open}"},
		Mid: []string{
			"{n} was born a {sp} of {home}, in {realm}, and was bred to the craft of a {call}.",
			"The {role} reckons the world then to have stood in the {era}, in the year {y0}.",
			"In those days {realm} was held by {ruler}, and {home} paid what it was asked to pay.",
			"The land was {world}, on the world called {planet}, and it was neither kinder nor crueller than most.",
			"Of the year {y0} the old records say: \"{event}\"",
			"No one marked the birth of the {call}; the marks came later, and were many.",
		},
		Close: []string{"Such was the country {n} left behind, and such was the world that waited.", "That was the world, and {n} had not yet been written into it."},
	},
	"wander": {
		Open: []string{"{n} left {home} with a {call}'s pack and a name no one yet knew.", "The first roads were not kind to {n}, who did not yet know what to ask of them.", "In those days {n} was no one: a {sp} {call} out of {home}, walking."},
		Mid: []string{
			"By the year {cy} the road had carried {n} to {pl1}, and then to {pl2}.",
			"At {pl1} no one knew the name {n}; at {pl2}, a few did.",
			"{folk}, {frole} of {fhome}, would say of those years: \"{quote}\"",
			"Of these days the {role} knows only that {deed1}.",
			"{n} slept where the road ended and rose where it began, and {motif} kept them company.",
			"{days} days passed, and {pl1} grew small behind {n}.",
			"At {pl3} {n} stopped long enough to be remembered, and not long enough to be bound.",
		},
		Close: []string{"So the long apprenticeship of the road went on, and {n} did not know it was an apprenticeship.", "Whatever {n} was meant to become, it was not yet; but the road had begun to shape it."},
	},
	"first": {
		Open: []string{"It is told that {deed1}.", "Then, at {pl1}, {deed1}.", "The first deed the {role} counts to {n} is this: {deed1}."},
		Mid: []string{
			"Also: {deed2}.",
			"The tale of it travelled faster than {n} did, and reached {pl2} before the {call} did.",
			"{folk} of {fhome} remembered it so: \"{quote}\"",
			"After that the name {n} was said in {pl1} with a different weight.",
			"Some say that was the day {epi} was born, though it took years for the name to stick.",
			"{motif} was with {n} that day, as it would be on many days after.",
		},
		Close: []string{"From then on {n} walked as one who has been noticed.", "Fame is a small thing at first, like a stone in a boot; {n} had begun to feel it."},
	},
	"foe": {
		Open: []string{"Every tale wants a shadow, and the shadow of {n} was {foe}, {foeepi}.", "It was in those years that {foe} entered the account, {foeepi}, and would not leave it.", "The road, which is patient, produced {foe}, {foeepi}, to stand against {n}."},
		Mid: []string{
			"{foewhy}",
			"{deed1}.",
			"The feud had its seasons: {deed2}.",
			"{foefate}",
			"{folk}, {frole} of {fhome}, said of the quarrel: \"{quote}\"",
			"Between {pl1} and {pl2} their paths crossed, and neither forgot.",
		},
		Close: []string{"Such enemies are the whetstone of a name, and {n} was sharper for it.", "A feud like that is a kind of kinship; the {role} will not say which of them was the poorer for its ending."},
	},
	"love": {
		Open: []string{"The {role} would be a poor one who sang only of blades: {n} loved, and was loved, and lost.", "Not every deed of {n} was done at arm's length; some were done at the fire.", "There is a part of the account that is not about the road."},
		Mid: []string{
			"In {pl1}, {n} took {spouse} for a spouse, and the road became a house for a while.",
			"{spousefate}",
			"A child came: {kid}, who would carry the name onward.",
			"{comp} walked beside {n} for a time.",
			"{compfell}",
			"{deed1}.",
			"{deed2}.",
			"{folk}, {frole} of {fhome}, said: \"{quote}\"",
		},
		Close: []string{"What the road gave it also took, in the old way, and {n} walked on lighter and heavier for it.", "The grief and the joy were the same weight, if you asked the {role}, and {n} carried both."},
	},
	"trial": {
		Open: []string{"Then came the great trial: {trial}.", "The world had its own fever in those years, and its name was {trial}.", "What {n} had been walking toward, without knowing it, was {trial}."},
		Mid: []string{
			"{deed1}.",
			"{deed2}.",
			"{endline}",
			"{pl1} remembers the smoke, and {pl2} remembers the waiting.",
			"{folk} of {fhome} said, long afterwards: \"{quote}\"",
			"{comp} stood with {n} when it mattered.",
			"{motif} did not leave {n} for a moment of it.",
		},
		Close: []string{"{n} came out of it older by more than the days it took.", "Whatever else is said of {n}, no one says that {n} was absent when the world leaned."},
	},
	"road": {
		Open: []string{"The road widened, as roads do for those who keep to them.", "There followed years that are hard to tell in order, as the best years are.", "{n} kept walking, and the walking became a kind of argument with the map."},
		Mid: []string{
			"By the year {cy} {n} had stood in {pl1} and in {pl2}, and had come to {pl3}.",
			"{deed1}.",
			"{deed2}.",
			"{folk}, {frole} of {fhome}, said: \"{quote}\"",
			"{days} days and {km} km lay behind {n} by now, and {motif} had grown into the shape of the life.",
			"At {pl2} the {call}'s trade paid, and at {pl1} it did not; the {role} counts both.",
		},
		Close: []string{"The years went the way years go; the name remained.", "And still the road went on ahead, as it does."},
	},
	"end": {
		Open: []string{"{enddeath}"},
		Mid: []string{
			"{days} days of wandering lay behind {n}, {km} km of road, and {kills} foes laid low.",
			"{deed1}.",
			"{folk}, {frole} of {fhome}, said when told: \"{quote}\"",
			"The {role} will not call it a good ending, or a bad one; only that it was {n}'s own.",
			"{motif} was the last thing {n} knew, or so those nearby said.",
		},
		Close: []string{"And here the road, which had no more use for {n}, turned back toward the world.", "That is the whole of it, as far as anyone saw."},
	},
	"epilogue": {
		Open: []string{"{song}", "What remained of {n} was a name, and names are heavier than they look."},
		Mid: []string{
			"{compwho}",
			"{spousewho}",
			"{kidwho}",
			"{foewho}",
			"The {role} says that {epi} is how {n} is called now, in {world}, wherever the telling reaches.",
			"The grave-marker reads, in the {role}'s copy: \"{epitaph1}\"",
			"Your story is told in {world}: other wanderers will meet the name {n} in the old tales, and in the mouths of the old, and some will walk the same road.",
		},
		Close: []string{"{close}"},
	},
}

// ---- epitaph lines ----

var sgEpitaphDeath = []string{"The road ended here; the road goes on.", "Death was one more place on the map.", "They did not die small.", "Dust, and then a name.", "Here the wanderer rests, who rarely did."}
var sgEpitaphRetire = []string{"Walked far, rested well.", "They put the road down, and no one made them.", "The long walk ended at their own door."}
var sgEpitaphByTrait = map[string]string{
	"brutal": "Many fell before them. They fell too.", "merciful": "Mercy was the sharpest thing they carried.", "wanderer": "Every road they took is still open.",
	"scholar": "Read, and was read.", "pious": "Their prayers outlast the stone.", "companionable": "They never walked alone for long.",
	"mourner": "They carried the names of others.", "delver": "Stone knew them best.", "seafarer": "Salt keeps what earth gives up.", "survivor": "They died the one time they had to.",
}

// ---- ending deeds ----

var sgEndWord = map[string]string{"destroy": "struck it down", "seal": "sealed it away", "bargain": "bargained with it", "ally": "turned it into an ally", "abandon": "turned away from it"}
