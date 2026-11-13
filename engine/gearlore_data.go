package engine

// Name and story data for gearlore.go (prefix gg).

// ggTagWord: the stem of an heirloom's title per victim tag, by genre ("" = default).
var ggTagWord = map[string]map[string]string{
	"undead": {"": "Grave", "cyberpunk": "Ghost", "steampunk": "Revenant", "mythic": "Shade", "wasteland": "Ghoul", "biopunk": "Blight", "twilight": "Wight"},
	"ruler":  {"": "King", "mythic": "Tyrant", "steampunk": "Baron", "dieselpunk": "Boss", "cyberpunk": "Exec", "wasteland": "Warlord", "twilight": "Crown"},
	"dragon": {"": "Wyrm", "mythic": "Drakon", "cyberpunk": "Daemon", "wasteland": "Behemoth", "biopunk": "Leviathan"},
	"legend": {"": "Horror", "mythic": "Titan", "cyberpunk": "Myth", "twilight": "Dread"},
	"fire":   {"": "Cinder", "cyberpunk": "Slag", "steampunk": "Boiler", "wasteland": "Flare"},
	"ice":    {"": "Rime", "cyberpunk": "Cryo", "steampunk": "Frost", "wasteland": "Static"},
	"man":    {"": "Oath", "cyberpunk": "Wire", "steampunk": "Rivet", "wasteland": "Raider", "mythic": "Hero"},
	"beast":  {"": "Fang", "wasteland": "Mutt", "cyberpunk": "Feral"},
}

// ggSuffix: title tail per tag for weapons (by skill) and armour.
var ggWeaponTail = map[string][]string{
	"longblade":  {"cleaver", "edge", "bane"},
	"shortblade": {"biter", "tongue", "kiss"},
	"axe":        {"render", "bite", "splitter"},
	"cudgel":     {"breaker", "knell", "fall"},
	"missile":    {"song", "sting", "bane"},
}
var ggArmorTail = []string{"ward", "mantle", "hide", "keep"}

// ggDefaultTitles: titles for gear with no telling history.
var ggDefaultTitles = map[string][]string{
	"":           {"The Quiet Hour", "Last Light", "Dawnwatch", "The Long Patience", "Fangbiter", "Old Faithful"},
	"steampunk":  {"The Steady Hand", "Brassheart", "Fourth Bell"},
	"dieselpunk": {"Last Round", "The Night Shift", "Ironclad Promise"},
	"atompunk":   {"Half-Life", "Bright Tomorrow", "The Quiet Hour"},
	"cyberpunk":  {"Zero Hour", "Static Prayer", "The Quiet Hour"},
	"biopunk":    {"Slow Bloom", "The Gentle Graft", "Quiet Strain"},
	"solarpunk":  {"Noon Vow", "Greenwatch", "The Open Hand"},
	"wasteland":  {"Dust Promise", "Last Mile", "The Quiet Hour"},
	"twilight":   {"Vigil", "The Pale Hour", "Candle-End"},
	"mythic":     {"Muse's Gift", "The Hour of Heroes", "Dawnwatch"},
}

var ggEffectNames = map[int]string{1: "a flaming edge", 2: "a frost bite", 3: "a ward against the dead", 4: "a thirst for blood", 5: "a rending edge",
	6: "a slayer's edge", 7: "a crown-cracking blow", 8: "a keen edge",
	11: "a mending touch", 12: "a cleansing ward", 13: "a stout heart"}

var ggTagVictims = map[string]string{"undead": "the restless dead", "ruler": "rulers", "dragon": "dragons", "legend": "legends", "fire": "burning things",
	"ice": "creatures of ice", "man": "men", "beast": "beasts"}

// ggRoleClause: short stories for previous owners, by Realmforge role.
var ggRoleClause = map[string]string{
	"ruler": "who ruled, and was feared for it", "general": "who led armies and lost count of the dead", "hero": "who earned a name by deeds and kept it by luck",
	"heir": "who never quite came into the inheritance", "prophet": "who preached to anyone who stood still", "artist": "who made things and sold fewer than hoped",
	"inventor": "who tinkered through every winter", "magician": "who kept odd hours and odder company", "thinker": "who argued with the dead and won",
	"merchant": "who counted everything twice", "priest": "who blessed it once, in a hurry",
}

var ggCurseMarks = []string{"a hanged man's knot cut into the inside", "a name scratched out so hard the metal is thin", "a smear of something that will not wash off",
	"three notches, one for each owner who came to a bad end"}
var ggBlessMarks = []string{"a tiny sun scratched inside, worn smooth by thumbs", "a prayer in a hand too small to read without a lens",
	"a maker's mark that sits warm against the skin", "a stitch of red thread, renewed by every owner"}
