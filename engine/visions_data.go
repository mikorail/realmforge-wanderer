package engine

import "strings"

// Text tables for visions.go (prefix vs): omen skins per genre, townsfolk reactions, seers' verse
// fragments and recurring dream motifs. Placeholders: {omen} {town} {lost} {nem} {scar} {kills}.

var vsOmenIDs = []string{"comet", "bloodmoon", "calf", "ravens", "rift"}

// omen id -> duration in days
var vsOmenDays = map[string]int{"comet": 7, "bloodmoon": 2, "calf": 4, "ravens": 4, "rift": 3}

// genre|omen|name|sight (what you see)|news (what the criers say)
var vsSkinRows = []string{
	"fantasy|comet|a hairy star|A pale hairy star drags its tail across the dark.|Astrologers swear a hairy star hangs over the roofs.",
	"fantasy|bloodmoon|a blood moon|The moon rises the colour of a fresh wound.|The moon rose red last night, and dogs howled until dawn.",
	"fantasy|calf|a two-headed calf|A calf with two heads, both lowing, is shown at the edge of town.|A two-headed calf was born in a byre, and the priests are called.",
	"fantasy|ravens|a murder of ravens|Ravens wheel in black thousands and settle on every roof.|Ravens have come in thousands and will not be driven off.",
	"fantasy|rift|a sign of the rift|The air shimmers like heat over water, though it is cold.|A shimmering in the air, they say, is the rift breathing.",
	"mythic|comet|a blazing star|A blazing star crosses the heavens from the east.|A blazing star has crossed the heavens; the gods are said to be restless.",
	"mythic|bloodmoon|a bleeding moon|The moon is veiled in red, like a goddess weeping.|The moon bleeds, and the oracles have shut their doors.",
	"mythic|calf|a monstrous birth|A calf with two heads and a human gaze is born in a temple byre.|A monstrous birth in the temple herd is read as a divine message.",
	"mythic|ravens|the god's messengers|Black birds crowd the sacred ways, silent and watching.|The messengers' birds are everywhere; priests argue over what they carry.",
	"mythic|rift|a thinning of the veil|The veil between worlds shows its weave for a few breaths.|The veil is thin; the gate-priests have begun their vigils.",
	"cyberpunk|comet|a burning orbital streak|A dead satellite burns up in a long orange scratch.|A de-orbiting relay lit the sky; brokers are betting on where the pieces fell.",
	"cyberpunk|bloodmoon|a red-shifted moon|Smog and a dead sensor net paint the moon blood red.|The moon went red over the stacks and half the grid flickered.",
	"cyberpunk|calf|a two-headed clone-calf|A vat-grown calf with two heads is displayed behind glass.|A two-headed vat calf got out of a lab; the corp denies it.",
	"cyberpunk|ravens|a flock of dead-eyed drones|Surveillance drones perch on every ledge, rotors still.|Hundreds of drones landed at once and went dark; nobody claims them.",
	"cyberpunk|rift|a ghost signal|Every screen shows the same flickering gate for one second.|A ghost signal hijacked the billboards: a gate that is not there.",
	"steampunk|comet|a comet over the smokestacks|A comet burns through the smoke like a lamp through gauze.|The observatory announces a comet; the clockmakers have sold out of lenses.",
	"steampunk|bloodmoon|a copper moon|The moon rises copper-red through the boiler fumes.|A copper moon, the almanac says, means pressure troubles by the week's end.",
	"steampunk|calf|a two-headed calf in a jar|A two-headed calf, stuffed, is the fair's newest wonder.|A two-headed calf is the talk of the exhibition; the curate frowns.",
	"steampunk|ravens|a conclave of ravens|Ravens have taken over the gantries and signal wires.|Ravens are cutting the telegraph wires; engineers blame the weather.",
	"steampunk|rift|a leyline flare|Brass instruments hum and the needles all point the same wrong way.|Every compass swung wild this morning; the guild blames the leylines.",
	"solarpunk|comet|a bright wanderer|A bright wanderer star hangs above the glass roofs.|A bright wanderer star shines over the commons; the night gardeners are out in numbers.",
	"solarpunk|bloodmoon|a rust-red moon|The moon turns rust red, and the glow-moss on the walls dims.|A rust-red moon unsettled the pollinators; the hives are quiet.",
	"solarpunk|calf|a two-headed calf|A two-headed calf is born in the commons herd; the herders are anxious.|A two-headed calf at the commons farm has the herders worried about the seed lines.",
	"solarpunk|ravens|a great gathering of crows|Crows gather on the solar vanes in uncanny silence.|Crows gather on the arrays and will not leave; the engineers are puzzled.",
	"solarpunk|rift|a shimmer in the canopy|A shimmer ripples through the canopy where no wind blows.|A shimmer in the canopy was seen by many; the stewards are cautious.",
	"biopunk|comet|a spore star|A star trailing a glittering tail seeds the sky with faint blue sparks.|A spore star burns above; the vat-wards are on watch for a bloom.",
	"biopunk|bloodmoon|a flesh-red moon|The moon swells and flushes like a gland.|The moon went flesh-red and every culture in the vats surged.",
	"biopunk|calf|a two-headed calf|A two-headed calf, fully healthy, is shown by the tenders.|A two-headed calf, healthy and whole, was cut free from the pens.",
	"biopunk|ravens|a swarm of carrion-wings|Scavenger birds blanket the tenders' roofs, bright-eyed and quiet.|Scavengers swarm the roofs; the tenders fear a die-off.",
	"biopunk|rift|a pulse in the tissue|The walls pulse once, slowly, as if the town had a heart.|The living walls pulsed in unison, and the gate-fungus grew overnight.",
	"wasteland|comet|a burning contrail|A long contrail burns in the sky: something old is falling.|A burning thing fell from the sky; scavengers are already marching.",
	"wasteland|bloodmoon|a rust moon|The moon is red as an old wound through the dust.|The moon went red in the dust; the tribes say it hungers.",
	"wasteland|calf|a two-headed calf|A calf with two heads, both breathing, in a salvage pen.|A two-headed calf is born in the pens, and the elders mutter about the rain.",
	"wasteland|ravens|carrion crows by the thousand|Crows lie in dark rows along every wall.|Crows by the thousand came to the walls; the old ones say a death is near.",
	"wasteland|rift|a hole in the air|A gap in the air shows another sky for a breath, then closes.|A hole opened in the air out past the dunes and showed a green sky.",
	"twilight|comet|a pale cinder star|A cinder-coloured star slides across the long dusk.|A cinder star in the dusk, and the lamp-keepers say it is the last of something.",
	"twilight|bloodmoon|a red hour|The dusk deepens to a deep red and stays.|The red hour came, and the wards guttered in their sockets.",
	"twilight|calf|a two-headed calf|A two-headed calf, grey as ash, stumbles by the ward-stones.|A two-headed calf was born by the ward-stones; the keepers are afraid.",
	"twilight|ravens|a parliament of ravens|Ravens sit on every ward-stone, facing the town.|Ravens sit on the ward-stones and face the town. They have not moved.",
	"twilight|rift|a rent in the dusk|The dusk tears for a heartbeat and light from somewhere else leaks through.|The dusk tore and light from elsewhere leaked through, the keepers say.",
}

type vsSkin struct{ Name, Sight, News string }

var vsSkins = vsBuildSkins()

func vsBuildSkins() map[string]map[string]vsSkin {
	out := map[string]map[string]vsSkin{}
	for _, row := range vsSkinRows {
		f := strings.Split(row, "|")
		if len(f) < 5 {
			continue
		}
		if out[f[0]] == nil {
			out[f[0]] = map[string]vsSkin{}
		}
		out[f[0]][f[1]] = vsSkin{f[2], f[3], f[4]}
	}
	return out
}

// short sidebar names
var vsOmenShort = map[string]string{"comet": "Comet", "bloodmoon": "Blood moon", "calf": "Omen calf", "ravens": "Ravens", "rift": "Rift sign"}

// townsfolk and holy-folk reactions per omen
var vsFolk = map[string][]string{
	"comet": {
		"\"Have you seen {omen}? My gran says such a star comes before a king dies, or a harvest does.\"",
		"\"{omen}, up there every night. The children can't sleep and the merchants raise their prices.\"",
		"\"They say {omen} is a promise or a threat. Nobody agrees which. I bar the door either way.\"",
	},
	"bloodmoon": {
		"\"Stay in after dark. When there's {omen}, the things in the ditches get bold.\"",
		"\"Did you see it? {omen}. Nothing good walks under that light.\"",
		"\"The soldiers sharpen their blades on nights like this, and so does everyone else.\"",
	},
	"calf": {
		"\"You heard what was born here? {omen}. The whole of {town} is watching it.\"",
		"\"Don't look at it too long. {omen} brings bad luck to those who stare.\"",
		"\"Some say the calf speaks at night. I won't go near the pen to learn.\"",
	},
	"ravens": {
		"\"{omen}. They arrived in the night, and they're still here. Count them and you count the dead.\"",
		"\"The old women feed them and pray. I'd rather they simply left.\"",
		"\"They say ravens know who is next. I keep my head down.\"",
	},
	"rift": {
		"\"The air has been strange all day. They say it is {omen}. I smell lightning where there is no storm.\"",
		"\"{omen}. Whatever lives on the far side of the gate is looking at us.\"",
		"\"My dog won't go near the gate square, and she goes everywhere.\"",
	},
}

var vsHoly = map[string][]string{
	"comet":     {"\"The heavens write in fire. Read it with fasting, not with fear.\"", "\"Such a sign precedes great turnings. Be ready, be kind.\""},
	"bloodmoon": {"\"This night the veil is thin and the debt is called. Keep your oaths.\"", "\"We keep vigil and bar the shrine. Come in if you wish; we ask no questions on such a night.\""},
	"calf":      {"\"An unnatural birth is a warning of unnatural times. We pray for the herd and for the town.\"", "\"We shall not harm the creature. It did not ask to be a sign.\""},
	"ravens":    {"\"The black ones carry the names of the dying. We say them back so they are not alone.\"", "\"They sit where death will sit. Be gentle with the old this week.\""},
	"rift":      {"\"The way between spheres is open a hand's breadth. Hold fast to what is yours.\"", "\"Do not cross while the sign stands. What returns is not always what left.\""},
}

// seers: titles, openers and imagery per genre
var vsSeerTitle = map[string]string{
	"fantasy": "oracle", "mythic": "pythia", "cyberpunk": "pattern-reader", "steampunk": "tape augur",
	"solarpunk": "weather sibyl", "biopunk": "spore reader", "wasteland": "bone reader", "twilight": "dusk seer",
}

var vsSeerWay = map[string]string{
	"fantasy": "casts bones and reads the fall", "mythic": "breathes the smoke and speaks in tongues",
	"cyberpunk": "reads the noise in dead feeds", "steampunk": "reads the punched tape the engine spits out",
	"solarpunk": "reads the pollen on the wind", "biopunk": "reads the pattern of a culture in a dish",
	"wasteland": "throws knucklebones into the dust", "twilight": "reads the shadows of the guttering lamps",
}

var vsOpeners = map[string][]string{
	"fantasy":   {"The bones tumble and settle, and the seer murmurs:", "She breathes on the glass and reads the mist:", "The candle leans; the voice comes from somewhere else:"},
	"mythic":    {"Smoke curls up through the cracked stone, and the voice says:", "Her eyes roll white and the god speaks through her:", "The tripod trembles, and the words come slow:"},
	"cyberpunk": {"Static resolves into a pattern; the reader mutters:", "The dead feed stutters into a voice, and the reader repeats:", "He squints at the noise and says it slowly:"},
	"steampunk": {"The engine chatters and a strip of tape curls out; the augur reads:", "Brass gears settle on a number, and she intones:", "The little bell rings thrice; the augur says:"},
	"solarpunk": {"She holds a leaf up to the light and reads its veins:", "The pollen falls in a pattern; the sibyl sings:", "Wind-chimes ring in an unfamiliar order. She says:"},
	"biopunk":   {"The culture dish blooms in unlikely colours; the reader says:", "The tissue folds and she follows it with a fingertip:", "The dish pulses once. The voice that follows is not hers:"},
	"wasteland": {"The knucklebones scatter in the dust, and he mutters:", "She squints at a scatter of rusted bolts and says:", "The wind drops, the dust settles into a shape, and the voice says:"},
	"twilight":  {"The lamps gutter and their shadows cross; the seer says:", "She lays her hand on a cold stone and the stone speaks:", "The dusk deepens at her words, and she whispers:"},
}

// images by subject
var vsImgDL = map[string][]string{
	"plague":   {"A grey breath crawls through the lanes, door to door.", "Small bells ring for the sick, and nobody answers."},
	"siege":    {"Iron stands at the gate and the gate has not yet learned to bend.", "Many feet at the wall, and thin bread in the larder."},
	"hostage":  {"A rope waits on a quiet morning, and a voice asks to be remembered.", "Someone sits in a small room counting the days on a wall."},
	"caravan":  {"Wheels that will not wait, and a purse that will not wait either.", "A line of lamps leaves at first light, with something you need among the bales."},
	"festival": {"Bells ring for a joy that is borrowed against the coming winter.", "Bright cloth and loud songs, and one stranger who is not what they seem."},
	"flood":    {"The river forgets its banks, and the low road goes under.", "Water finds the old path, and the old path is a door."},
	"vault":    {"A door opens once beneath a darkened sky, and closes without apology.", "A locked thing sleeps, and the sky will turn the key for a few days only."},
	"relic":    {"A lost thing is wanted by many hands and held by none.", "Something that was buried to be forgotten is remembered by the wrong people."},
}
var vsImgRival = []string{"A wanderer with a hunter's stride walks toward a prize you wanted.", "Another's footprints cross your own, ahead and eager.", "A stranger's lamp burns where you meant to go."}
var vsImgBoss = []string{"Something old sleeps beneath stone, dreaming of teeth.", "A guardian who has not slept in a hundred years waits for a worthy fool.", "A hall of echoes where one echo does not belong to you."}
var vsImgWar = []string{"Two crowns trade iron and call it argument.", "Banners that were friends once are now stitched with each other's blood.", "Drums on two horizons, and a thin road between them."}
var vsImgOmen = []string{"The sky will wear a strange face and the townsfolk will whisper.", "Above, something will be written that no hand held the chalk for.", "Look up, and then look at who is looking up beside you."}

var vsDirPrecise = map[string]string{
	"north": "toward where the cold wind is born", "north-east": "toward where the first frost meets the first light",
	"east": "toward where the sun is born", "south-east": "toward where the warm rain begins at dawn",
	"south": "toward where the geese go to winter", "south-west": "toward where the evening is thickest and warm",
	"west": "toward where the sun is buried", "north-west": "toward where the last snow lies at dusk",
}
var vsDirAmbig = map[string]string{
	"north": "toward the cold, or toward whatever the cold is hiding", "north-east": "toward the morning's edge, or the winter's",
	"east": "toward the sunrise, or whoever else is waiting for it", "south-east": "toward the soft ground, or what lies under it",
	"south": "toward the warm place that is not warm", "south-west": "toward the setting light, or the place it is falling on",
	"west": "toward the end of the day, or the end of something else", "north-west": "toward the grey where the road forgets itself",
}
var vsTimeMeta = []string{
	"before the cock crows twice", "before the moon changes its face", "within the span of a hand",
	"when the leaves have turned or the snow has come", "in a season that is not yet named",
}

// prophecy verse hints (cryptic lines) and what they mean
var vsVerseHint = [3][]string{
	{"A great shadow waits where the road forgets itself; spare it or spill it, the price is the same.", "The quarry has your name in its mouth and has not decided what to do with it."},
	{"Between two skies a gate sleeps; it opens when you are no longer afraid to leave.", "The soil you stand on will ask whether you love it more than the road."},
	{"Thrones weigh you on a scale you cannot see; some will kneel, some will bar the door.", "A crown is only a hat until someone tells you to bow."},
}

// dream motifs: id -> source id, name, five stage lines
type vsMotif struct {
	ID, Src, Name string
	Lines         [5]string
}

var vsMotifs = []vsMotif{
	{"river", "blood", "the red river", [5]string{
		"You stand on a riverbank. The water at your feet is the wrong colour.",
		"The red river has risen to your knees, and faces drift past without turning.",
		"The river carries names now. Yours is among them, and others you never learned.",
		"On the far bank someone holds out a cup of the water and waits to see whether you will drink.",
		"You kneel and drink, and the river tastes only of rain. It goes quiet and clear.",
	}},
	{"bell", "lost", "the drowned bell", [5]string{
		"Far below, a bell tolls under water. You cannot tell whether you are hearing it or remembering it.",
		"The bell sounds nearer. Between the strokes you hear {lost} say your name.",
		"You dive and find the bell hung from a drowned tree; its clapper is a worn glove you recognise.",
		"The bell speaks at last, in {lost}'s voice: it was never your doing, only your burden.",
		"You lift the bell out of the water and it is only metal. The silence is gentle.",
	}},
	{"door", "nemesis", "the unlit door", [5]string{
		"A long corridor, and at its end a door that is just ajar. Someone is breathing behind it.",
		"The door is closer, and under it a line of light shaped like {nem}'s shadow.",
		"You hear {nem} speak through the wood, reciting your own steps back to you.",
		"The door opens inward. The room is yours; every item in it is something you have taken from someone.",
		"You step through and find the room empty but for a mirror. You do not look away from it.",
	}},
	{"hollow", "wound", "the hollow place", [5]string{
		"You wake in the dream with a place in your body that is cold and empty, as if something had been taken.",
		"The hollow has a shape now, {scar}-shaped, and wind plays through it like a flute.",
		"Small hands from the dark are filling the hollow with ash and calling it healing.",
		"You pour the ash out. Under it, the old wound is closing like a mouth that has finished speaking.",
		"The hollow is a vessel, and it holds light. You carry it carefully down a hill.",
	}},
	{"lamp", "mercy", "the lantern field", [5]string{
		"A dark field, and in the far distance one lantern that someone is carrying.",
		"More lanterns now, each carried by a figure who stops to look at you and nods.",
		"You recognise some of the faces: those you spared, those you fed, those you let go.",
		"They set the lanterns in the grass and the field is a map of the roads you have walked.",
		"You are handed a lantern of your own. It weighs nothing, and the way ahead is visible.",
	}},
}

// per source intros
var vsIntro = map[string][]string{
	"blood": {
		"You dream you have walked a thousand paces through those you have killed ({kills} so far, and they all know your name).",
		"Your hands are clean in the dream, and nobody believes you.",
		"The dead of your road form a quiet audience and ask nothing.",
	},
	"lost": {
		"{lost} sits at your fire in the dream and does not touch the food.",
		"You dream of {lost} laughing at a joke you do not remember telling.",
		"A familiar silhouette in the doorway is {lost}'s, turned from you.",
	},
	"nemesis": {
		"{nem} is at the edge of your dream, always closer than the last time.",
		"A letter in a hand you know too well: {nem}'s, and it says only 'soon'.",
		"You hear {nem}'s horse, though you are on foot and asleep.",
	},
	"wound": {
		"The old injury sings in your sleep and you answer it in a language you cannot speak awake.",
		"You dream of the blow itself, slowed to a hundred heartbeats.",
		"In the dream the wound is a window, and through it you see yourself sleeping.",
	},
	"mercy": {
		"You dream of those you let live; they are older now, and some of them remember kindly.",
		"A quiet dream: a table, a plain meal, nobody afraid of you.",
		"You hear a child's voice say thank you in a language you half know.",
	},
}

var vsGenreDream = map[string]string{
	"fantasy":   "A raven the colour of ink watches from the rafters of the dream.",
	"mythic":    "A voice that is not a voice speaks in verse; you only keep the last line.",
	"cyberpunk": "A dead feed scrolls your name in a font you do not own.",
	"steampunk": "Gears turn in the walls of the dream, counting something down.",
	"solarpunk": "Roots move under the dream floor, listening.",
	"biopunk":   "The dream has a pulse and you are inside it.",
	"wasteland": "Dust sifts through the dream, and each grain is a small bone.",
	"twilight":  "The dream is lit by a single guttering lamp that never goes out.",
}

var vsInsightLines = []string{
	"You wake knowing something you did not know before.",
	"The dream leaves a clean edge in your mind: a thing understood, not yet said.",
	"A fragment stays with you in the grey of waking, sharper than memory.",
}
