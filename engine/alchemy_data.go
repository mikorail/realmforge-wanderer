package engine

// Herbalism and alchemy data (prefix al): properties, statuses, methods, bases, named recipes, books.

type alPropDef struct {
	ID, Name, Col string
	Bad           bool // harmful when drunk, useful as a poison or smoke
}

var alProps = []alPropDef{
	{"healing", "healing", "G", false}, {"stamina", "stamina", "C", false}, {"antidote", "antidote", "g", false},
	{"feverc", "fever-cure", "Y", false}, {"sleep", "sleep", "b", true}, {"calm", "calm", "c", false},
	{"rage", "rage", "R", true}, {"fireres", "fire-ward", "O", false}, {"frostres", "frost-ward", "B", false},
	{"nightsight", "night-sight", "M", false}, {"strength", "strength", "O", false}, {"swiftness", "swiftness", "C", false},
	{"poison", "poison", "g", true}, {"paralysis", "paralysis", "y", true}, {"hallucination", "hallucination", "m", true},
	{"cleansing", "cleansing", "W", false},
}

var alPropByID = func() map[string]*alPropDef {
	m := map[string]*alPropDef{}
	for i := range alProps {
		m[alProps[i].ID] = &alProps[i]
	}
	return m
}()

// alConflicts: opposed properties cancel each other and leave the drinker queasy.
var alConflicts = [][2]string{
	{"healing", "poison"}, {"calm", "rage"}, {"sleep", "stamina"}, {"swiftness", "paralysis"},
	{"antidote", "poison"}, {"cleansing", "hallucination"}, {"strength", "sleep"}, {"nightsight", "hallucination"},
}

// alTagProp: the cooking/beast tag of an ingredient decides its first property.
var alTagProp = map[string]string{
	"hearty": "strength", "spry": "swiftness", "thick": "frostres", "mending": "healing", "keen": "nightsight",
	"ember": "fireres", "rime": "frostres", "deep": "stamina", "lasting": "stamina", "venom": "poison",
	"venomous": "poison", "storm": "paralysis", "caustic": "cleansing", "light": "swiftness", "sturdy": "strength",
}

type alMethod struct {
	ID, Name, Noun string
	Diff, Pot      int
	Doses          int
	Tool           int // 0 none, 1 mortar, 2 alembic
}

var alMethods = []alMethod{
	{"infusion", "infusion", "infusion", 0, 0, 1, 0},
	{"decoction", "decoction", "decoction", 1, 1, 1, 0},
	{"distillation", "distillation", "distillate", 3, 2, 1, 2},
	{"ointment", "ointment", "ointment", 1, 0, 2, 0},
	{"pill", "pill", "pills", 2, -1, 3, 1},
	{"smoke", "smoke", "smoke bundle", 2, 0, 1, 0},
	{"coating", "poison coating", "coating", 2, 0, 3, 0},
}

type alBase struct {
	ID, Name, Liquid string // Liquid: the Effect key of the container ("" fresh water)
	Pot              int
}

var alBases = []alBase{
	{"water", "water", "", 0}, {"wine", "wine", "wine", 0}, {"oil", "oil", "oil", 0},
	{"honey", "honey", "honey", 0}, {"alcohol", "spirit (wine)", "wine", 0},
}

type alRecipeDef struct {
	ID, Name string
	Props    []string
}

var alRecipes = []alRecipeDef{
	{"restorative", "Restorative", []string{"healing", "stamina"}},
	{"febrifuge", "Febrifuge", []string{"feverc", "cleansing"}},
	{"countervenom", "Counter-venom", []string{"antidote", "cleansing"}},
	{"nightglass", "Nightglass", []string{"nightsight", "swiftness"}},
	{"giants", "Giant's draught", []string{"strength", "stamina"}},
	{"hearthward", "Hearth-ward", []string{"fireres", "frostres"}},
	{"stilltea", "Still tea", []string{"calm", "sleep"}},
	{"berserker", "Berserker's brew", []string{"rage", "strength"}},
	{"quicksilver", "Quicksilver", []string{"swiftness", "stamina"}},
	{"deathcap", "Deathcap", []string{"poison", "paralysis"}},
	{"dreamsmoke", "Dream-smoke", []string{"hallucination", "sleep"}},
	{"mendsalve", "Mending salve", []string{"healing", "cleansing"}},
	{"clearmind", "Clear mind", []string{"calm", "nightsight"}},
}

var alRecipeByID = func() map[string]*alRecipeDef {
	m := map[string]*alRecipeDef{}
	for i := range alRecipes {
		m[alRecipes[i].ID] = &alRecipes[i]
	}
	return m
}()

type alBookDef struct {
	Title   string
	Recipes []string
}

var alBooks = []alBookDef{
	{"A Gatherer's Primer", []string{"restorative"}},
	{"The Apothecary's Margin", []string{"febrifuge", "countervenom"}},
	{"Of Lamps and Night Roots", []string{"nightglass", "clearmind"}},
	{"Ward Against Flame and Frost", []string{"hearthward", "mendsalve"}},
	{"The Quiet Cup", []string{"stilltea", "giants"}},
	{"Black Herbal", []string{"deathcap", "dreamsmoke"}},
}

func init() {
	for _, d := range []*skStatusDef{
		{ID: "alheal", Name: "Mending", Glyph: '+', Tick: func(g *Game, a *Actor) {
			if a.HP < a.MaxHP {
				a.HP++
			}
		}},
		{ID: "alstam", Name: "Vigorous", Glyph: 'v', Speed: 8, Tick: func(g *Game, a *Actor) { delete(a.Status, "exhausted") }},
		{ID: "alnight", Name: "Night-sight", Glyph: 'n'},
		{ID: "alswift", Name: "Swift", Glyph: '>', Speed: 20, AGI: 2},
		{ID: "alstrong", Name: "Strong", Glyph: 'S', Hit: 1},
		{ID: "alfire", Name: "Fire-warded", Glyph: 'f', Tick: func(g *Game, a *Actor) { delete(a.Status, "burning") }},
		{ID: "alfrost", Name: "Frost-warded", Glyph: 'r', Tick: func(g *Game, a *Actor) { delete(a.Status, "frozen") }},
		{ID: "alcalm", Name: "Calm", Glyph: 'k', WIL: 3, MA: 2},
		{ID: "alrage", Name: "Raging", Glyph: 'R', Hit: 2, DV: -2, Speed: 5},
		{ID: "alsleep", Name: "Drugged sleep", Glyph: 'z', DV: -12, PV: 4, Tick: func(g *Game, a *Actor) { a.AddSt("stunned", 2) }},
		{ID: "alpara", Name: "Paralyzed", Glyph: 'P', DV: -6, Tick: func(g *Game, a *Actor) { a.AddSt("stunned", 2) }},
		{ID: "alhall", Name: "Hallucinating", Glyph: 'h', INT: -3, Hit: -2, MA: -2},
	} {
		skAddStatus(d)
	}
}
