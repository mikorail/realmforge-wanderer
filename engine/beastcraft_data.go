package engine

// Beast-crafting data (beastcraft.go): material kinds, tags and the recipe table.

// bcMatDef describes one kind of animal material. Suffix lists the name endings that identify it
// on an item (bsProducts in zwildlife.go names its drops "<species> <suffix>").
type bcMatDef struct {
	ID     string
	Label  string
	Suffix []string
	AV     int // armour bonus when worn
	DV     int
	Weight int
}

var bcMats = []bcMatDef{
	{"hide", "hide", []string{"hide"}, 0, 0, 0},
	{"pelt", "pelt", []string{"pelt"}, 0, 1, 0},
	{"scale", "scales", []string{"scales"}, 1, 0, 1},
	{"carapace", "carapace", []string{"carapace"}, 1, -1, 2},
	{"shell", "shell", []string{"shell"}, 1, -1, 2},
	{"bone", "bone", []string{"bones", "bone"}, 0, 0, 0},
	{"horn", "horn", []string{"horns", "horn", "antlers", "antler"}, 0, 0, 0},
	{"ivory", "ivory", []string{"ivory"}, 0, 0, 0},
	{"feather", "feathers", []string{"feathers"}, 0, 1, -1},
	{"sinew", "sinew", []string{"sinew"}, 0, 0, 0},
	{"fat", "fat", []string{"blubber", "fat"}, 0, 0, 0},
	{"gland", "venom gland", []string{"venom gland"}, 0, 0, 0},
	{"ichor", "ichor", []string{"ichor"}, 0, 0, 0},
	{"organ", "organs", []string{"organs"}, 0, 0, 0},
}

var bcMatByID = func() map[string]*bcMatDef {
	m := map[string]*bcMatDef{}
	for i := range bcMats {
		m[bcMats[i].ID] = &bcMats[i]
	}
	return m
}()

// bcTagDef is a material tag: what it adds to armour, weapons, tonics and mounts.
type bcTagDef struct {
	Name   string
	Desc   string
	AV, DV int
	PV, DB int
	Elem   string      // weapon element
	ABonus map[int]int // armour attribute bonus
	WBonus map[int]int // weapon attribute bonus
	Extra  int         // extra tonic doses
	Unique bool
}

var bcTags = map[string]bcTagDef{
	"sturdy":      {Name: "sturdy", Desc: "tough and heavy", AV: 1, PV: 1},
	"light":       {Name: "light", Desc: "light and springy", DV: 1, WBonus: map[int]int{AGI: 1}},
	"warm":        {Name: "warm", Desc: "thick and warm", ABonus: map[int]int{TOU: 1}},
	"sharp":       {Name: "sharp", Desc: "hard and keen", PV: 2, ABonus: map[int]int{STR: 1}},
	"keen":        {Name: "keen", Desc: "fine-grained", DB: 1, ABonus: map[int]int{AGI: 1}},
	"supple":      {Name: "supple", Desc: "supple and strong", DV: 1, PV: 1},
	"venomous":    {Name: "venomous", Desc: "laced with venom", Elem: "poison", ABonus: map[int]int{WIL: 1}, Extra: 1},
	"rime":        {Name: "rime", Desc: "cold to the touch", Elem: "frost", ABonus: map[int]int{TOU: 1}, AV: 1},
	"ember":       {Name: "ember", Desc: "warm as a coal", Elem: "fire", ABonus: map[int]int{STR: 1}},
	"storm":       {Name: "storm", Desc: "crackling", Elem: "shock", DV: 1},
	"caustic":     {Name: "caustic", Desc: "bitter and corrosive", Elem: "acid", AV: 1},
	"mending":     {Name: "mending", Desc: "slow to wither", ABonus: map[int]int{TOU: 1}, Extra: 1},
	"ancient":     {Name: "ancient", Desc: "from a lost age", AV: 1, PV: 1, ABonus: map[int]int{TOU: 1}, Extra: 1, Unique: true},
	"storied":     {Name: "storied", Desc: "sung of in a hundred tales", ABonus: map[int]int{WIL: 1, EGO: 1}, WBonus: map[int]int{EGO: 1}, DB: 1, Extra: 1, Unique: true},
	"dawnlit":     {Name: "dawnlit", Desc: "glowing faintly gold", Elem: "fire", DV: 1, ABonus: map[int]int{WIL: 1}, Extra: 1, Unique: true},
	"ghostly":     {Name: "ghostly", Desc: "pale and half unreal", DV: 2, Extra: 1, Unique: true},
	"voidtouched": {Name: "voidtouched", Desc: "cold from somewhere beyond", Elem: "shock", ABonus: map[int]int{AGI: 1}, WBonus: map[int]int{AGI: 1}, Extra: 1, Unique: true},
	"thunderous":  {Name: "thunderous", Desc: "heavy with old thunder", PV: 1, ABonus: map[int]int{STR: 2}, WBonus: map[int]int{STR: 1}, Extra: 1, Unique: true},
}

var bcUniqueTags = []string{"ancient", "storied", "dawnlit", "ghostly", "voidtouched", "thunderous"}

var bcQualWord = []string{"", "crude", "sound", "fine", "superb", "peerless"}

var bcQualRarity = []string{"", "common", "uncommon", "uncommon", "rare", "ultrarare"}

// bcSlot is one required material: any of the listed material ids.
type bcSlot struct {
	Mats string // space-separated material ids
}

// bcRecipe is one beast-craft recipe. Cat: armor weapon tonic bait trophy saddle.
type bcRecipe struct {
	ID    string
	Name  string // menu name
	Noun  string // the crafted thing, after the species name
	Cat   string
	Rank  int    // skill rank wanted (0-3, best of Tinkering and Survival)
	Slot  string // armour slot, weapon skill ("ranged" for a bow), tonic effect, bait kind
	Slots []bcSlot
	Basic bool // known from the start
}

var bcRecipes = []*bcRecipe{
	// armour
	{ID: "a_cap", Name: "hide cap", Noun: "hide cap", Cat: "armor", Slot: "head", Basic: true, Slots: []bcSlot{{"hide pelt"}}},
	{ID: "a_boots", Name: "hide boots", Noun: "hide boots", Cat: "armor", Slot: "feet", Rank: 0, Basic: true, Slots: []bcSlot{{"hide pelt"}}},
	{ID: "a_jerkin", Name: "leather jerkin", Noun: "leather jerkin", Cat: "armor", Slot: "body", Rank: 1, Slots: []bcSlot{{"hide pelt"}, {"hide pelt sinew"}}},
	{ID: "a_cloak", Name: "pelt cloak", Noun: "pelt cloak", Cat: "armor", Slot: "back", Rank: 1, Slots: []bcSlot{{"pelt"}, {"pelt hide feather"}}},
	{ID: "a_gloves", Name: "hide gloves", Noun: "hide gloves", Cat: "armor", Slot: "hands", Rank: 1, Slots: []bcSlot{{"hide pelt"}, {"sinew"}}},
	{ID: "a_bonehelm", Name: "bone helm", Noun: "bone helm", Cat: "armor", Slot: "head", Rank: 1, Slots: []bcSlot{{"bone"}, {"hide pelt sinew"}}},
	{ID: "a_mask", Name: "feathered mask", Noun: "feathered mask", Cat: "armor", Slot: "face", Rank: 1, Slots: []bcSlot{{"feather"}, {"feather hide pelt"}}},
	{ID: "a_scale", Name: "scale coat", Noun: "scale coat", Cat: "armor", Slot: "body", Rank: 2, Slots: []bcSlot{{"scale"}, {"scale"}, {"sinew hide"}}},
	{ID: "a_chitin", Name: "chitin cuirass", Noun: "chitin cuirass", Cat: "armor", Slot: "body", Rank: 2, Slots: []bcSlot{{"carapace"}, {"carapace"}, {"sinew"}}},
	{ID: "a_bracers", Name: "shell bracers", Noun: "shell bracers", Cat: "armor", Slot: "hands", Rank: 2, Slots: []bcSlot{{"shell carapace"}, {"sinew hide pelt"}}},
	{ID: "a_hornhelm", Name: "horned helm", Noun: "horned helm", Cat: "armor", Slot: "head", Rank: 2, Slots: []bcSlot{{"horn"}, {"bone shell carapace"}}},
	// weapons
	{ID: "w_knife", Name: "bone knife", Noun: "bone knife", Cat: "weapon", Slot: "shortblade", Basic: true, Slots: []bcSlot{{"bone"}}},
	{ID: "w_club", Name: "bone club", Noun: "bone club", Cat: "weapon", Slot: "cudgel", Rank: 1, Slots: []bcSlot{{"bone"}, {"bone sinew"}}},
	{ID: "w_spear", Name: "horn spear", Noun: "horn spear", Cat: "weapon", Slot: "longblade", Rank: 1, Slots: []bcSlot{{"horn"}, {"bone sinew hide"}}},
	{ID: "w_hbow", Name: "horn bow", Noun: "horn bow", Cat: "weapon", Slot: "ranged", Rank: 1, Slots: []bcSlot{{"horn"}, {"sinew"}}},
	{ID: "w_dirk", Name: "ivory dirk", Noun: "ivory dirk", Cat: "weapon", Slot: "shortblade", Rank: 2, Slots: []bcSlot{{"ivory"}}},
	{ID: "w_axe", Name: "tusk axe", Noun: "tusk axe", Cat: "weapon", Slot: "axe", Rank: 2, Slots: []bcSlot{{"ivory horn"}, {"bone"}, {"sinew"}}},
	{ID: "w_ibow", Name: "ivory bow", Noun: "ivory bow", Cat: "weapon", Slot: "ranged", Rank: 2, Slots: []bcSlot{{"ivory"}, {"sinew"}}},
	// tonics
	{ID: "t_salve", Name: "fat salve", Noun: "salve", Cat: "tonic", Slot: "salve", Basic: true, Slots: []bcSlot{{"fat"}}},
	{ID: "t_heal", Name: "restorative tonic", Noun: "restorative", Cat: "tonic", Slot: "heal", Rank: 1, Slots: []bcSlot{{"organ fat"}, {"fat ichor gland"}}},
	{ID: "t_anti", Name: "antivenin", Noun: "antivenin", Cat: "tonic", Slot: "antidote", Rank: 1, Slots: []bcSlot{{"gland ichor"}, {"organ fat"}}},
	{ID: "t_rage", Name: "rage draught", Noun: "rage draught", Cat: "tonic", Slot: "strength", Rank: 2, Slots: []bcSlot{{"gland organ"}, {"ichor gland"}}},
	// bait and lures
	{ID: "b_meat", Name: "meat lure", Noun: "meat lure", Cat: "bait", Slot: "meat", Basic: true, Slots: []bcSlot{{"fat organ sinew"}}},
	{ID: "b_sweet", Name: "sweet cake", Noun: "sweet cake", Cat: "bait", Slot: "plant", Rank: 0, Basic: true, Slots: []bcSlot{{"fat"}}},
	{ID: "b_fish", Name: "fish-oil lure", Noun: "fish-oil lure", Cat: "bait", Slot: "fish", Rank: 1, Slots: []bcSlot{{"fat organ"}}},
	{ID: "b_grub", Name: "grub lure", Noun: "grub lure", Cat: "bait", Slot: "grub", Rank: 1, Slots: []bcSlot{{"ichor gland organ"}}},
	// trophy and mount gear
	{ID: "r_head", Name: "mounted head", Noun: "head", Cat: "trophy", Rank: 1, Slots: []bcSlot{{"horn ivory bone feather shell"}, {"hide pelt"}}},
	{ID: "s_saddle", Name: "saddle", Noun: "saddle", Cat: "saddle", Slot: "saddle", Rank: 1, Slots: []bcSlot{{"hide pelt"}, {"sinew"}, {"bone horn shell"}}},
	{ID: "s_bard", Name: "barding", Noun: "barding", Cat: "saddle", Slot: "barding", Rank: 2, Slots: []bcSlot{{"scale carapace shell"}, {"hide pelt sinew"}}},
}

// bcRoleCats says what each trade teaches about beast materials.
var bcRoleCats = map[string]string{
	"tailor":      "armor",
	"smith":       "weapon",
	"apothecary":  "tonic",
	"beastkeeper": "bait saddle",
	"forager":     "bait tonic",
	"naturalist":  "tonic trophy",
	"trainer":     "armor weapon",
}
