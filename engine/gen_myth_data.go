package engine

// Domains of the gods and omen tables (prefix gd).
type gdDomainDef struct {
	ID, Name string
	Words    []string
	Gives    []string // item kinds the god likes as offerings
	Bless    string
	Miracle  string // the miracle this domain does cheaply
}

var gdDomains = []gdDomainDef{
	{"war", "war", []string{"war", "battle", "blood", "blade", "sword", "spear", "conquer", "glory", "valor", "warrior", "thunder", "iron", "fight"}, []string{"weapon", "armor"}, "Hit +3, speed, +1 armour", "calm"},
	{"harvest", "the harvest", []string{"harvest", "grain", "field", "farm", "crop", "earth", "soil", "fertil", "growth", "bread", "rain", "spring"}, []string{"food", "ingredient"}, "armour and will, slow mending", "rain"},
	{"death", "death", []string{"death", "grave", "dead", "underworld", "shadow", "night", "ancestor", "bone", "afterlife", "ash", "rest", "void"}, []string{"trophy", "corpse"}, "evasion and accuracy", "calm"},
	{"sea", "the sea", []string{"sea", "ocean", "tide", "river", "water", "wave", "storm", "fish", "sail", "deep", "flood"}, []string{"water"}, "speed and evasion", "rain"},
	{"trade", "trade", []string{"trade", "coin", "gold", "merchant", "wealth", "road", "market", "fortune", "contract", "luck", "travel"}, []string{"treasure"}, "silver tongue and wit", "sign"},
	{"wisdom", "wisdom", []string{"wisdom", "knowledge", "truth", "learn", "scholar", "book", "word", "light", "law", "mind", "star", "moon", "sky"}, []string{"book"}, "mind and will", "sign"},
	{"hearth", "hearth and healing", []string{"hearth", "home", "heal", "mercy", "life", "love", "family", "kind", "peace", "compassion", "mother", "sun"}, []string{"tonic", "food"}, "mending and sturdiness", "heal"},
	{"wild", "the wild", []string{"beast", "hunt", "forest", "wild", "wolf", "animal", "spirit", "nature", "tree", "stag", "bear"}, []string{"trophy", "ingredient"}, "agility and evasion", "calm"},
}

type gdSignDef struct {
	Text string
	Sky  bool
}

var gdSigns = map[string][]gdSignDef{
	"plague":  {{"the moon rises ringed with sickly green", true}, {"flies gather thick on the well-stones", false}, {"every dog in the street howls at once", false}},
	"war":     {{"a comet drags a red split tail across the sky", true}, {"ravens circle the barracks and will not land", false}, {"the moon bleeds red", true}},
	"famine":  {{"the stars blur behind a dry haze", true}, {"mice file out of the granaries", false}, {"cattle lie down facing the empty east", false}},
	"bandits": {{"owls cry at noon along the road", false}, {"a falling star burns out over the hills", true}, {"crows pick at a dropped purse and fly off with it", false}},
	"boom":    {{"fish leap in the shallows in shining shoals", false}, {"a golden dawn, three days running", true}, {"bees swarm into the market and settle on the scales", false}},
	"storm":   {{"swallows skim the ground and the wind drops dead", false}, {"a halo around the moon", true}, {"the cattle bunch together and lower their heads", false}},
}

var gdOmenKinds = []string{"plague", "war", "famine", "bandits", "boom", "storm", "storm", "bandits", "boom"}
