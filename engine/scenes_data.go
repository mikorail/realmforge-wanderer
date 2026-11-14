package engine

// Data for scenes.go (prefix ms): which disaster becomes which rescue scene, and the words each scene
// and migration uses.

// msClassOf maps a Realmforge disaster kind (hz) to a scene class.
var msClassOf = map[string]string{
	"quake":     "dig",
	"landslide": "dig",
	"avalanche": "dig",
	"tornado":   "dig",
	"flood":     "hold",
	"locust":    "hold",
	"wildfire":  "fire",
	"eruption":  "evac",
	"tsunami":   "shore",
	"storm":     "shore",
	"plague":    "carry",
}

type msClassDef struct {
	Title   string   // scene title, %s = place
	Brief   string   // announcement
	Victim  string   // singular noun of who is in need
	Goal    string   // title of the one who waits at the goal
	GoalMsg string   // what the goal says
	Role    string   // role of the people in need
	GoalRol string   // role of the goal actor
	Mins    int      // time limit, minutes
	Reward  int      // base gold
	Won     []string // news lines on success, %s = place, %s = hero
	Lost    []string // news lines on failure, %s = place
}

var msClasses = map[string]*msClassDef{
	"dig": {
		Title: "Trapped under the ruins near %s", Victim: "trapped survivor", Role: "ms_trapped", Mins: 150, Reward: 40,
		Brief: "{{R|Cries come from under fallen masonry!}} Survivors are pinned under the rubble. Bump a survivor to dig them out; time is short.",
		Won:   []string{"%s: survivors were pulled alive from the ruins by the wanderer %s.", "%s: %s led the digging and the living were carried out of the rubble."},
		Lost:  []string{"%s: the digging came too late; the dead were laid out in rows.", "%s: no one reached those under the stones in time."},
	},
	"hold": {
		Title: "Hold the line at %s", Victim: "wrecker", Goal: "watch captain", Role: "ms_raider", GoalRol: "ms_warden", Mins: 140, Reward: 55,
		GoalMsg: "\"Looters and wreckers are coming for what the disaster left. Hold this post with me until the last wave breaks.\"",
		Brief:   "{{R|Raiders are coming for the wrecked place!}} Hold the post against three waves. Speak with the captain.",
		Won:     []string{"%s: the post held against the wreckers; %s fought at the captain's side.", "%s: raiders broke on the watch line, with the wanderer %s among the defenders."},
		Lost:    []string{"%s: wreckers overran the post and many were robbed and cut down.", "%s: the line broke; the raiders took what the disaster had left."},
	},
	"fire": {
		Title: "Fight the fire at %s", Victim: "blaze", Goal: "fire warden", Role: "ms_fire", GoalRol: "ms_warden", Mins: 100, Reward: 45,
		GoalMsg: "\"Beat the flames out, or fell the trees ahead of them to starve it. Take the buckets: water puts a blaze out at once.\"",
		Brief:   "{{R|Fire runs through the dry growth!}} Beat out the blazes (bump), douse them with water buckets, or fell trees ahead of them as firebreaks.",
		Won:     []string{"%s: the fire was stopped before the houses; %s was at the front of the line.", "%s: the blaze died at a firebreak cut by the wanderer %s."},
		Lost:    []string{"%s: the fire burned out of reach and took homes with it.", "%s: the fire got away and people were lost in the smoke."},
	},
	"evac": {
		Title: "Evacuate %s before the mountain", Victim: "townsperson", Goal: "road marshal", Role: "ms_evac", GoalRol: "ms_marshal", Mins: 160, Reward: 50,
		GoalMsg: "\"The mountain is waking. Bring them to me, to the road, and I will send them on. Quickly.\"",
		Brief:   "{{R|The mountain is shaking: evacuate!}} Persuade townsfolk to follow you and lead them to the road marshal.",
		Won:     []string{"%s: the town was emptied ahead of the ash by the wanderer %s and the marshals.", "%s: %s led a column of people down the road just before the mountain spoke."},
		Lost:    []string{"%s: some did not leave in time and were lost under the ash.", "%s: the evacuation was too slow; the ash took those who stayed."},
	},
	"shore": {
		Title: "Survivors on the shore at %s", Victim: "survivor", Goal: "relief warden", Role: "ms_survivor", GoalRol: "ms_camp", Mins: 150, Reward: 45,
		GoalMsg: "\"There are survivors out in the surf and wreckage. Haul them to this fire and we will warm them.\"",
		Brief:   "{{R|Survivors cling to wreckage in the shallows!}} Wade out, help them up and lead them to the relief fire.",
		Won:     []string{"%s: ships' survivors were pulled from the surf by the wanderer %s.", "%s: the sea gave back living men and women, with the wanderer %s hauling them ashore."},
		Lost:    []string{"%s: the sea kept what it took; survivors drowned in the surf.", "%s: help came too late for those in the water."},
	},
	"carry": {
		Title: "Plague patients near %s", Victim: "patient", Goal: "field physician", Role: "ms_patient", GoalRol: "ms_healer", Mins: 170, Reward: 50,
		GoalMsg: "\"Those who still walk can be saved if they reach my tent. Bring them one at a time, and keep your own face covered.\"",
		Brief:   "{{R|The sick lie by the road, too weak to walk alone!}} Help them up and lead them to the physician's tent.",
		Won:     []string{"%s: patients were carried to the physician's tent by the wanderer %s.", "%s: %s brought the sick to the healers and many lived."},
		Lost:    []string{"%s: the sick died by the roadside, unable to reach the healers.", "%s: the plague claimed those no one carried in time."},
	},
}

// msCause: a column of refugees' reason, by disaster kind (used when the origin was struck).
var msCauseWord = map[string]string{
	"quake": "the earthquake", "tsunami": "the great wave", "eruption": "the eruption", "storm": "the great storm",
	"tornado": "the whirlwind", "flood": "the flood", "landslide": "the landslide", "avalanche": "the avalanche",
	"locust": "the locusts", "wildfire": "the wildfire", "war": "the fighting",
}

var msRefugeeAsk = []string{
	"\"We walked from %s. Our granaries are gone and the road is long. Will you walk with us as far as %s?\"",
	"\"%s is behind us, and nothing of it left to return to. We are making for %s; the road is not safe.\"",
}

var msPilgrimLines = []string{
	"\"We go to %s, to the shrine, for the high feast. All who walk with us are blessed at the gate.\"",
	"\"%s is a long road, but the road is the prayer. Join us and be blessed.\"",
}

// msFightPairs: beasts that fight over ground. Both are existing creature templates.
var msFightPairs = [][2]string{
	{"wolf", "boar"}, {"bear", "wolf"}, {"boar", "bear"}, {"jaguar", "wolf"}, {"bear", "jaguar"}, {"boar", "jaguar"},
}

var msVictimTitles = map[string]string{
	"ms_trapped": "trapped survivor", "ms_patient": "plague patient", "ms_survivor": "shipwreck survivor", "ms_evac": "townsperson",
}

var msGratefulLines = []string{
	"\"I will not forget that you came for me.\"",
	"\"My family owes you more than I can say.\"",
	"\"Take this. It is little against a life.\"",
}
