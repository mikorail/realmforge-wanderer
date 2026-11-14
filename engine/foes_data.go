package engine

// Data tables for foes.go (enemy combat depth): damage types, resistance rules, elite affixes, phase lines.

var bwTypes = []string{"slash", "pierce", "blunt", "fire", "cold", "shock", "poison", "acid", "holy", "dark", "psychic"}

// bwFlagRes: creature flag -> type -> percent (positive resists, negative is a weakness, 100 immune).
var bwFlagRes = map[string]map[string]int{
	"fire":    {"fire": 80, "cold": -50, "acid": 10},
	"frost":   {"cold": 80, "fire": -50, "shock": 10},
	"undead":  {"poison": 100, "psychic": 100, "dark": 60, "cold": 30, "holy": -60, "fire": -25, "pierce": 20},
	"poison":  {"poison": 80, "acid": 10},
	"acid":    {"acid": 80, "slash": 10},
	"regen":   {"fire": -30, "acid": -30},
	"aquatic": {"shock": -35, "cold": 20, "fire": 10},
	"fly":     {"pierce": -10, "blunt": -10},
	"burrow":  {"shock": 40, "pierce": 10},
	"boss":    {"psychic": 25, "poison": 15},
}

// bwNameRes: substring of template id or name -> type -> percent.
var bwNameRes = []struct {
	Words []string
	Res   map[string]int
}{
	{[]string{"skeleton", "bone", "lich", "mummy", "revenant"}, map[string]int{"slash": 25, "pierce": 50, "blunt": -40, "holy": -50}},
	{[]string{"ghost", "wraith", "spirit", "shade", "phantom", "specter", "spectre"}, map[string]int{"slash": 60, "pierce": 60, "blunt": 60, "holy": -50, "dark": 60, "shock": -30}},
	{[]string{"demon", "imp", "devil", "fiend", "hellhound"}, map[string]int{"fire": 60, "dark": 50, "holy": -50, "cold": -20}},
	{[]string{"angel", "seraph", "cherub", "saint"}, map[string]int{"holy": 80, "dark": -50, "fire": 20}},
	{[]string{"troll", "ogre"}, map[string]int{"fire": -40, "acid": -40, "blunt": 10}},
	{[]string{"dragon", "drake", "wyrm", "wyvern"}, map[string]int{"slash": 15, "pierce": 10, "psychic": 20}},
	{[]string{"golem", "gargoyle", "statue", "stone", "rock", "boulder", "earth"}, map[string]int{"slash": 30, "pierce": 55, "blunt": -35, "poison": 100, "psychic": 100, "acid": -10}},
	{[]string{"ice", "frost", "yeti", "snow", "glacial", "rime"}, map[string]int{"cold": 80, "fire": -50, "blunt": -10}},
	{[]string{"lava", "magma", "ember", "flame", "inferno", "cinder", "salamander"}, map[string]int{"fire": 80, "cold": -45, "acid": 10}},
	{[]string{"slime", "ooze", "jelly", "pudding", "blob"}, map[string]int{"slash": 45, "pierce": 45, "blunt": -10, "fire": -20, "acid": 50}},
	{[]string{"mushroom", "fungus", "spore", "moss", "vine", "treant", "plant", "shambler", "root"}, map[string]int{"fire": -50, "slash": -15, "cold": -10, "poison": 40}},
	{[]string{"beetle", "scarab", "crab", "scorpion", "carapace", "shell", "tortoise", "turtle"}, map[string]int{"slash": 25, "pierce": 15, "blunt": -30, "fire": -10}},
	{[]string{"spider", "ant", "wasp", "bee", "mantis", "centipede", "insect"}, map[string]int{"blunt": -20, "fire": -20, "poison": 20}},
	{[]string{"wolf", "dog", "hound", "jackal", "bear", "boar", "cat", "lion", "tiger"}, map[string]int{"pierce": 5, "psychic": 10}},
	{[]string{"mind", "brain", "flayer", "elder", "star-spawn", "aberration", "eye"}, map[string]int{"psychic": -35, "dark": 30}},
	{[]string{"witch", "mage", "wizard", "sorcerer", "shaman", "priest", "cultist"}, map[string]int{"psychic": 15, "dark": 10}},
	{[]string{"drone", "robot", "automaton", "mech", "clockwork", "construct", "android", "turret", "cyborg"}, map[string]int{"poison": 100, "psychic": 100, "shock": -45, "acid": -15, "pierce": 15, "cold": 10}},
	{[]string{"rad", "mutant", "ghoul", "glow", "fallout"}, map[string]int{"poison": 40, "shock": -10, "holy": -10}},
	{[]string{"shark", "eel", "fish", "squid", "kraken", "serpent", "naga"}, map[string]int{"shock": -30, "cold": 15}},
	{[]string{"bandit", "raider", "thug", "cutthroat", "outlaw", "guard", "soldier"}, map[string]int{}},
}

// bwGenreRes: small genre-wide shifts for every creature.
var bwGenreRes = map[string]map[string]int{
	"twilight":  {"dark": 25, "holy": -25},
	"mythic":    {"holy": 10, "dark": 10},
	"biopunk":   {"acid": -15, "fire": -10},
	"cyberpunk": {"shock": -10},
	"wasteland": {"poison": 20, "psychic": 10},
	"steampunk": {"shock": -5},
}

// bwPhysSkill maps a weapon skill to its damage type.
var bwPhysSkill = map[string]string{
	"shortblade": "slash", "longblade": "slash", "axe": "slash", "whip": "slash", "great": "slash",
	"cudgel": "blunt", "flail": "blunt", "staff": "blunt", "fist": "blunt", "heavy": "blunt",
	"spear": "pierce", "hook": "pierce", "ranged": "pierce", "thrown": "pierce", "bow": "pierce", "firearm": "pierce",
}

var bwNatPierce = []string{"bite", "fang", "tooth", "teeth", "horn", "tusk", "beak", "sting", "spike", "thorn", "spear", "arrow", "bolt", "dart"}
var bwNatSlash = []string{"claw", "talon", "slash", "blade", "scythe", "cut", "sword", "axe", "razor", "saw"}
var bwCauseType = []struct{ Word, Type string }{
	{"burn", "fire"}, {"fire", "fire"}, {"flame", "fire"}, {"lava", "fire"}, {"explo", "fire"},
	{"frost", "cold"}, {"cold", "cold"}, {"ice", "cold"}, {"freez", "cold"},
	{"shock", "shock"}, {"lightning", "shock"}, {"electr", "shock"}, {"bolt", "shock"},
	{"poison", "poison"}, {"venom", "poison"}, {"toxin", "poison"},
	{"acid", "acid"}, {"holy", "holy"}, {"smite", "holy"}, {"radiance", "holy"},
	{"dark", "dark"}, {"shadow", "dark"}, {"necro", "dark"},
	{"psychic", "psychic"}, {"mind", "psychic"}, {"psionic", "psychic"},
}

// Elite affixes.
type bwAffixDef struct {
	ID     string
	Prefix string // "Scarred X" style
	Word   string // shown in the sidebar
	Min    int    // minimum level
	Hint   string
}

var bwAffixes = []bwAffixDef{
	{"armoured", "Armoured", "armoured", 1, "Its hide is plated; blows glance off."},
	{"regen", "Mending", "regenerating", 2, "Its wounds close as you watch."},
	{"venom", "Venomous", "venomous", 1, "A green sheen wets its weapons."},
	{"vampiric", "Hungering", "vampiric", 3, "It drinks the blood it spills."},
	{"explosive", "Volatile", "explosive", 2, "It reeks of fumes; it will burst when it dies."},
	{"thief", "Scavenging", "food thief", 1, "It eyes your pack with hunger."},
	{"swift", "Quick", "swift", 1, "It moves before you can blink."},
	{"warded", "Warded", "warded", 3, "Faint runes crawl over it, turning aside magic."},
	{"frenzy", "Maddened", "frenzied", 2, "It is wild-eyed and bleeding already, and it does not care."},
}

var bwScarPrefix = []string{"Scarred", "Grizzled", "Old", "Notched", "Gaunt", "Veteran"}

// Phase kinds and their telegraph / action lines. {n} = the creature.
var bwPhaseKinds = []string{"enrage", "summon", "resist", "heal", "flee"}

var bwTele = map[string][]string{
	"enrage": {"{n} snarls and its eyes burn red.", "{n} shudders, rage rising in it like a tide.", "{n} beats its chest and bellows."},
	"summon": {"{n} lets out a calling cry.", "{n} raises a signal; something answers from the dark.", "{n} whistles sharply for aid."},
	"resist": {"{n} draws itself in and its skin begins to change.", "{n} turns inward; a strange stillness gathers about it.", "{n} hardens, shifting against the blows it has taken."},
	"heal":   {"{n} gathers its strength and the air grows still.", "{n} clutches its wounds and mutters.", "{n} drinks deep of something from the shadows."},
	"flee":   {"{n} falters and glances for an exit.", "{n} backs away, plainly hurt.", "{n} looks for a way out."},
}

var bwDone = map[string][]string{
	"enrage": {"{n} is ENRAGED and strikes faster!", "{n} flies into a frenzy!"},
	"summon": {"Reinforcements rush to {n}'s side!", "More of its kind answer {n}'s call!"},
	"resist": {"{n}'s body has changed: it now turns away %s and shrinks from %s!"},
	"heal":   {"{n} mends its worst wounds!"},
	"flee":   {"{n} turns tail and runs for cover!"},
}

// Behaviour words for casters and cowards.
var bwCasterWords = []string{"mage", "wizard", "witch", "shaman", "priest", "sorcerer", "cultist", "druid", "warlock", "necromancer", "oracle", "seer", "acolyte", "hierophant", "adept", "hexer"}
var bwCowardWords = []string{"goblin", "kobold", "rat", "bandit", "cutthroat", "thief", "scavenger", "jackal", "imp", "vermin", "raider"}
