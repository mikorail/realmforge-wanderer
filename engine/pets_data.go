package engine

import "strings"

// pets_data.go (prefix pq): pet species, tricks, names and the play-mode table. All data, no logic.

type pqSp struct {
	ID     string
	Name   string            // default noun
	Alt    map[string]string // genre id -> noun (e.g. steampunk "clockwork hound")
	Young  string            // noun for a youngster ("kitten")
	Glyph  rune
	Col    byte
	Body   string // MakeBody plan
	HP     int
	Spd    int
	Bite   [3]int // dice n, sides, pv
	Mech   bool   // eats scrap, not food
	Fly    bool
	Price  int // base price, currency units
	Wt     int // weight among strays (0 = never a stray)
	Smart  int // highest trick level it can reach (1-3)
	Perks  []string
	Apt    []string // tricks it practises faster
	Genres string   // space separated genre ids; "" = every genre
	Desc   string
	Names  []string
}

var pqSpecies = []pqSp{
	{ID: "cat", Name: "cat", Young: "kitten", Glyph: 'f', Col: CYellow, Body: "quadruped", HP: 12, Spd: 120, Bite: [3]int{1, 3, 0}, Price: 40, Wt: 6, Smart: 2,
		Perks: []string{"vermin", "charm"}, Apt: []string{"warm", "steal"}, Genres: "",
		Desc:  "A lean, unimpressed cat that keeps its own counsel and the granary free of mice.",
		Names: []string{"Ash", "Mittens", "Soot", "Pepper", "Thistle", "Biscuit", "Wick", "Marrow", "Juniper", "Sorrel"}},
	{ID: "dog", Name: "dog", Young: "puppy", Glyph: 'd', Col: CBrown, Body: "quadruped", HP: 24, Spd: 110, Bite: [3]int{1, 4, 1}, Price: 70, Wt: 7, Smart: 3,
		Perks: []string{"alarm", "nose", "charm"}, Apt: []string{"guard", "fetch", "warn"}, Genres: "",
		Desc:  "A loyal, shaggy dog with a good nose and a worse sense of its own size.",
		Names: []string{"Rex", "Bramble", "Tuck", "Moss", "Brindle", "Patch", "Gruff", "Biscuit", "Clover", "Hob"}},
	{ID: "hawk", Name: "hawk", Young: "fledgling hawk", Glyph: 'h', Col: CBrown, Body: "avian", HP: 10, Spd: 130, Bite: [3]int{1, 4, 1}, Fly: true, Price: 120, Wt: 2, Smart: 3,
		Perks: []string{"eyes"}, Apt: []string{"scout", "warn"}, Genres: "",
		Desc:  "A hooded-eyed hawk that rides your shoulder or the thermals, and sees the world twice as far.",
		Names: []string{"Skirl", "Talon", "Gale", "Rook", "Sable", "Vesper", "Kestrel", "Windle"}},
	{ID: "ferret", Name: "ferret", Young: "kit ferret", Glyph: 'e', Col: CBrown, Body: "quadruped", HP: 9, Spd: 125, Bite: [3]int{1, 3, 0}, Price: 45, Wt: 4, Smart: 2,
		Perks: []string{"vermin", "nose"}, Apt: []string{"steal", "herbs"}, Genres: "",
		Desc:  "A long, quick, thieving ferret that pours itself into any hole it can find.",
		Names: []string{"Noodle", "Slink", "Pip", "Weasel", "Quill", "Nib", "Fennel", "Skein"}},
	{ID: "monkey", Name: "monkey", Young: "baby monkey", Glyph: 'y', Col: CDarkOrange, Body: "quadruped", HP: 14, Spd: 115, Bite: [3]int{1, 3, 0}, Price: 90, Wt: 2, Smart: 3,
		Perks: []string{"charm"}, Apt: []string{"steal", "fetch", "herbs"}, Genres: "",
		Desc:  "A clever monkey with hands for hands and opinions about everyone's pockets.",
		Names: []string{"Jabber", "Figgy", "Tamarind", "Biko", "Mango", "Chitter", "Pocket", "Olo"}},
	{ID: "lizard", Name: "lizard", Young: "hatchling lizard", Glyph: 'l', Col: CGreen, Body: "quadruped", HP: 11, Spd: 105, Bite: [3]int{1, 3, 0}, Price: 35, Wt: 3, Smart: 2,
		Perks: []string{"vermin", "nose"}, Apt: []string{"herbs", "scout"}, Genres: "",
		Desc:  "A sun-loving lizard with a flicking tongue that tastes what the air has been up to.",
		Names: []string{"Flick", "Basalt", "Emberly", "Scree", "Gecko", "Sunny", "Dart", "Pebble"}},
	{ID: "mouse", Name: "mouse", Young: "pup mouse", Glyph: 'm', Col: CGray, Body: "quadruped", HP: 5, Spd: 120, Bite: [3]int{1, 2, 0}, Price: 12, Wt: 3, Smart: 2,
		Perks: []string{"alarm"}, Apt: []string{"steal", "warn"}, Genres: "",
		Desc:  "A small brave mouse that rides in your hood and squeaks at what you have not noticed.",
		Names: []string{"Crumb", "Nutmeg", "Whisk", "Mote", "Tick", "Button", "Seed", "Squeak"}},
	{ID: "robodog", Name: "robo-dog", Alt: map[string]string{"steampunk": "clockwork hound", "wasteland": "scrap hound", "dieselpunk": "iron hound", "atompunk": "atomic hound", "cyberpunk": "robo-dog"},
		Young: "pup unit", Glyph: 'D', Col: CCyan, Body: "quadruped", HP: 26, Spd: 110, Bite: [3]int{1, 5, 2}, Mech: true, Price: 140, Wt: 5, Smart: 2,
		Perks: []string{"alarm", "nose"}, Apt: []string{"guard", "fetch"}, Genres: "steampunk dieselpunk atompunk cyberpunk wasteland",
		Desc:  "A jointed hound of tin and servos, tail wagging on a spring, ears swivelling on motors.",
		Names: []string{"Sprocket", "Rivet", "Unit-9", "Cog", "Bolt", "Tinker", "Static", "Gizmo"}},
	{ID: "drone", Name: "drone", Alt: map[string]string{"solarpunk": "kite-drone", "wasteland": "salvage drone", "atompunk": "hover-eye"},
		Young: "micro-drone", Glyph: 'o', Col: CCyan, Body: "avian", HP: 9, Spd: 130, Bite: [3]int{1, 3, 1}, Mech: true, Fly: true, Price: 160, Wt: 3, Smart: 3,
		Perks: []string{"eyes", "alarm"}, Apt: []string{"scout", "warn"}, Genres: "cyberpunk solarpunk atompunk wasteland dieselpunk",
		Desc:  "A palm-sized hovering drone with one lens, a whining fan and an unreasonable attachment to you.",
		Names: []string{"Ping", "Halo", "Argus", "Wisp", "Lumen", "Beep", "Kite", "Zed"}},
	{ID: "imp", Name: "imp", Alt: map[string]string{"twilight": "gloam imp", "mythic": "sprite"},
		Young: "imp-child", Glyph: 'i', Col: CRed, Body: "avian", HP: 13, Spd: 120, Bite: [3]int{1, 4, 1}, Fly: true, Price: 110, Wt: 4, Smart: 3,
		Perks: []string{"charm", "eyes"}, Apt: []string{"steal", "warm", "scout"}, Genres: "fantasy mythic twilight",
		Desc:  "A small bat-winged imp, mischievous but bound by some strange fondness to follow you about.",
		Names: []string{"Snicker", "Cinder", "Grim", "Nettle", "Brimble", "Vex", "Spindle", "Smut"}},
	{ID: "biopet", Name: "bio-pet", Alt: map[string]string{"solarpunk": "vine-pet"},
		Young: "bud-pet", Glyph: 'b', Col: CGreen, Body: "quadruped", HP: 16, Spd: 110, Bite: [3]int{1, 4, 1}, Price: 100, Wt: 5, Smart: 2,
		Perks: []string{"nose", "charm"}, Apt: []string{"herbs", "warm"}, Genres: "biopunk solarpunk",
		Desc:  "A soft-bodied engineered companion, half pup, half fungus, humming faintly when pleased.",
		Names: []string{"Spore", "Gill", "Bloom", "Mycel", "Truffle", "Nub", "Lobe", "Fern"}},
}

var pqSpByID = func() map[string]*pqSp {
	m := map[string]*pqSp{}
	for i := range pqSpecies {
		m[pqSpecies[i].ID] = &pqSpecies[i]
	}
	return m
}()

func (s *pqSp) genreOK(genre string) bool {
	if s.Genres == "" {
		return true
	}
	if genre == "" || genre == "fantasy" {
		return strings.Contains(" "+s.Genres+" ", " fantasy ")
	}
	return strings.Contains(" "+s.Genres+" ", " "+genre+" ")
}

// noun is what the species is called in this genre.
func (s *pqSp) noun(genre string) string {
	if n, ok := s.Alt[genre]; ok {
		return n
	}
	return s.Name
}

func (s *pqSp) has(list []string, id string) bool {
	for _, x := range list {
		if x == id {
			return true
		}
	}
	return false
}

// ---- tricks ----

type pqTrick struct {
	ID, Name string
	Desc     [3]string // by level 1-3
}

var pqTricks = []pqTrick{
	{"fetch", "Fetch", [3]string{"fetches small things (coins, scrap, herbs) lying in the open and drops them at your feet.", "fetches from further away and more often.", "fetches almost anything small within sight."}},
	{"guard", "Guard", [3]string{"fights harder when you are in danger (+1 bite).", "bites hard and shields you when hurt (+2 bite).", "a fierce guardian (+3 bite, more health)."}},
	{"scout", "Scout", [3]string{"ranges ahead and reveals nearby ground outdoors.", "reveals a wider ring of ground.", "sees far; the map opens around it."}},
	{"herbs", "Find herbs", [3]string{"now and then nosed out a herb or fungus where you walk.", "finds herbs more often.", "a reliable forager."}},
	{"steal", "Steal small things", [3]string{"lifts a few coins from the unwary in towns (risky).", "better at not being seen.", "a master pilferer."}},
	{"warn", "Warn of danger", [3]string{"growls, chirps or hisses at unseen hostiles within range.", "warns from further off, and tells which way.", "warns early and exactly."}},
	{"warm", "Warm you", [3]string{"curls up against you, sparing a little health and the cold.", "warms you faster.", "a living hot-water bottle."}},
}

var pqTrickByID = func() map[string]*pqTrick {
	m := map[string]*pqTrick{}
	for i := range pqTricks {
		m[pqTricks[i].ID] = &pqTricks[i]
	}
	return m
}()

// pqTrickLv turns practice points into a trick level 0-3.
func pqTrickLv(prac int) int {
	switch {
	case prac >= 20:
		return 3
	case prac >= 10:
		return 2
	case prac >= 4:
		return 1
	}
	return 0
}

var pqMoodText = map[string]string{
	"starving": "{{R|starving}}", "hungry": "{{W|hungry}}", "frightened": "{{R|hurt and frightened}}", "lonely": "{{y|lonely}}",
	"content": "{{g|content}}", "playful": "{{G|playful}}", "devoted": "{{G|devoted}}",
}

// ---- play modes ----

type pqMode struct {
	ID, Name, Desc string
	StartOnly      bool    // cannot be switched on or off mid-run
	Mul            float64 // hall-of-fame score multiplier
	Excludes       []string
}

var pqModes = []pqMode{
	{ID: "peaceful", Name: "Peaceful explorer", Desc: "Hostile creatures and bandits are only rarely aggressive. They still defend themselves if struck.", Mul: 0.5},
	{ID: "story", Name: "Story mode", Desc: "Death is forgiving: you wake at the last inn you rested at, minus some coin.", Mul: 0.6, Excludes: []string{"ironman"}},
	{ID: "ironman", Name: "Iron man (marker)", Desc: "Marks the run as iron man: no manual saves, death ends it. Uses the challenge of the same name.", StartOnly: true, Mul: 1.15, Excludes: []string{"story"}},
	{ID: "onehit", Name: "One-hit tension", Desc: "Your hit points never grow past a thin ceiling; every blow matters.", StartOnly: true, Mul: 1.2},
	{ID: "cozy", Name: "No-hunger cozy", Desc: "Hunger and thirst never rise past a comfortable level.", Mul: 0.8},
}

var pqModeByID = func() map[string]*pqMode {
	m := map[string]*pqMode{}
	for i := range pqModes {
		m[pqModes[i].ID] = &pqModes[i]
	}
	return m
}()
