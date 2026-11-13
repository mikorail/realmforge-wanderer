package engine

// Solarpunk crops (prefix gd). Sun 1 = wants open sun, 0 = shade-lover. Seasons: 0 spring .. 3 winter.
type gdCrop struct {
	ID, Name string
	Kind     string // food herb flower
	Tag      string // cooking tag for herbs
	Days     int
	Season   [4]bool
	Sun      int
	Hardy    bool
	Yield    [2]int
	Price    int
	Rare     bool
	Col      byte
}

var gdCrops = []gdCrop{
	{"sunmaize", "sunmaize", "food", "", 8, [4]bool{true, true, false, false}, 1, false, [2]int{3, 5}, 6, false, CYellow},
	{"beanvine", "bean-vine", "food", "", 6, [4]bool{true, true, true, false}, 1, false, [2]int{3, 6}, 5, false, CGreen},
	{"rootbeet", "root-beet", "food", "", 7, [4]bool{true, false, true, true}, 0, true, [2]int{2, 4}, 5, false, CRed},
	{"greenleaf", "greenleaf", "food", "", 4, [4]bool{true, false, true, false}, 0, false, [2]int{2, 4}, 4, false, CGreen},
	{"pepperbloom", "pepper-bloom", "herb", "ember", 7, [4]bool{false, true, false, false}, 1, false, [2]int{2, 3}, 8, false, COrange},
	{"honeygourd", "honey-gourd", "food", "", 9, [4]bool{false, true, true, false}, 1, false, [2]int{2, 4}, 7, false, CYellow},
	{"wintkale", "winter kale", "herb", "rime", 6, [4]bool{false, false, true, true}, 0, true, [2]int{2, 4}, 7, false, CCyan},
	{"healmoss", "healmoss", "herb", "mending", 5, [4]bool{true, false, true, false}, 0, false, [2]int{2, 3}, 9, false, CGreen},
	{"dreampoppy", "dream-poppy", "herb", "keen", 7, [4]bool{true, true, false, false}, 1, false, [2]int{2, 3}, 10, false, CMagenta},
	{"sunflower", "sunflower", "flower", "", 6, [4]bool{false, true, true, false}, 1, false, [2]int{2, 4}, 6, false, CYellow},
	{"lavender", "lavender", "flower", "", 8, [4]bool{true, true, false, false}, 1, true, [2]int{2, 4}, 7, false, CMagenta},
	{"glowbulb", "glowbulb", "herb", "keen", 10, [4]bool{true, false, true, false}, 0, false, [2]int{1, 3}, 12, false, CCyan},
	{"emberroot", "ember-tuber", "herb", "ember", 9, [4]bool{false, false, true, false}, 1, false, [2]int{2, 3}, 9, false, COrange},
	{"frostberry", "frostberry", "food", "", 8, [4]bool{false, false, true, true}, 0, true, [2]int{2, 4}, 8, false, CCyan},
	{"starorchid", "star-orchid", "flower", "", 14, [4]bool{true, true, true, false}, 0, false, [2]int{1, 2}, 40, true, CWhite},
	{"ironbulb", "iron-bulb", "herb", "thick", 12, [4]bool{false, true, true, false}, 1, true, [2]int{1, 3}, 25, true, CGray},
}

var gdCropByID = func() map[string]*gdCrop {
	m := map[string]*gdCrop{}
	for i := range gdCrops {
		m[gdCrops[i].ID] = &gdCrops[i]
	}
	return m
}()

// weather solar / wind factors by weather kind id
var gdSolarW = map[string]float64{"clear": 1, "pollen": 1, "heat": 1.1, "overcast": 0.5, "rain": 0.3, "storm": 0.2, "tempest": 0.2, "snow": 0.25, "blizzard": 0.1,
	"fog": 0.35, "frost": 0.9, "sporerain": 0.3, "spores": 0.5}
var gdWindW = map[string]float64{"clear": 0.7, "pollen": 0.6, "heat": 0.5, "overcast": 1, "rain": 1.3, "storm": 2.2, "tempest": 2.4, "snow": 1, "blizzard": 2,
	"fog": 0.3, "frost": 0.8, "sporerain": 1.1, "spores": 0.6}
var gdSeasonSun = [4]float64{1, 1.15, 0.85, 0.6}

// grid level costs in bits (tier -> count) and what each level gives
var gdGridCost = [6]map[int]int{nil, {1: 4}, {1: 4, 2: 3}, {2: 4, 3: 3}, {3: 4, 4: 3}, {4: 4, 5: 3, 6: 2}}
var gdGridPerk = [6]string{"", "outputs +25%, seeds 10% cheaper", "the purifier runs free for everyone", "the freight lift runs free, seeds 20% cheaper", "outputs +100%, gardeners pay well for produce", "the grid-wrights gift you a heliograph"}
