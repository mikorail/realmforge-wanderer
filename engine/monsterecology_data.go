package engine

// Monster ecology data (prefix mg): guild names, population words, den names, notices and farmer talk.
// Templates take {a} {b} {town} {place} {sp} {old} {season} {n} and are filled by mgFill.

const (
	mgHerd = iota
	mgGame
	mgBirds
	mgFish
	mgPred
	mgApex
	mgGuilds
)

var mgGuildName = [mgGuilds]string{"Grazing herds", "Small game", "Birds", "Fish", "Predators", "Apex hunters"}

var mgGuildShort = [mgGuilds]string{"herds", "small game", "birds", "fish", "predators", "apex hunters"}

// mgForageBiome: how much a biome feeds grazers (1 = plenty), index = biome id 0..16.
var mgForageBiome = [17]float64{0, 0.25, 0.3, 1.2, 1.3, 0.8, 1.0, 0.9, 1.0, 0.7, 0.5, 0.1, 0.9, 0.2, 0.3, 0.5, 0.05}

var mgSeasonNames = [4]string{"spring", "summer", "autumn", "winter"}

var mgPopWords = []string{"gone", "scarce", "thin", "fair", "good", "plentiful", "teeming"}

var mgPredWords = []string{"none", "few", "some", "a fair number", "many", "a great many", "a plague of them"}

// mgBroodWord is what the young of a group are called.
var mgBroodWord = map[string]string{
	"mam": "cub", "syn": "cub", "bird": "chick", "rept": "hatchling", "dino": "hatchling", "ptero": "hatchling",
	"amph": "larva", "fish": "fry", "ins": "grub", "arach": "spiderling", "crust": "brood", "myria": "brood", "arth": "brood",
}

var mgAlphaFirst = []string{"Red", "Grey", "Black", "Pale", "Old", "Scar", "Long", "Split", "Hollow", "Ash", "Iron", "Gaunt", "Torn", "Bone", "Night", "Cold", "Wide", "Crooked"}

var mgAlphaLast = []string{"maw", "fang", "ear", "tooth", "claw", "hide", "eye", "back", "tail", "paw", "jaw", "coat", "horn", "wing", "snout", "brow"}

var mgDenPlace = []string{"Hollow", "Hole", "Holt", "Lair", "Rocks", "Thicket", "Burrow", "Crag", "Dell", "Earth", "Bluff", "Scrape"}

var mgDenSite = []string{
	"The den lies in a scrape under a tilted slab, ringed with gnawed bones.",
	"A low mouth in the bank, the earth in front of it packed hard by many feet.",
	"A thicket pushed flat in a ring; the middle smells strongly of old meat.",
	"A hollow under roots. Scat and shed fur mark the paths into it.",
	"A heap of dung and bones against a rock face, with a dark gap behind it.",
}

var mgTrackWords = []string{"fresh {sp} tracks", "a scatter of {sp} droppings", "tufts of {sp} fur on a thorn", "a scratched {sp} trail-mark"}

var mgRemainWords = []string{"gnawed bones of a deer", "a half-eaten carcass", "a stripped sheep skull", "the wing-bones of a large bird", "a heap of cracked marrow bones"}

var mgNoticeSelf = []string{
	"NOTICE. The farmers of {town} thank whoever drove the {old} from this place. Since {season} there are new tracks here: {sp}. Keep stock in at night and do not let children past the stone.",
	"Scratched on the stone by the farmers of {town}: the {old} are gone, but {sp} have taken the den. We are not done with this. Warn the drovers.",
	"A board nailed to a post: \"{town} herders: the {old} left. Something else has moved in, and by the tracks it is {sp}. Mind your flocks.\"",
}

var mgNoticeVacant = []string{
	"A notice cut in the stone: the den is empty, for now. Farmers of {town} pay to know if anything moves in again.",
}

// herd commentary by state; mgFill replaces {place} {town} {sp}
var mgLineEarly = []string{
	"\"The herds came through early this year. I have never seen the {sp} on the move so soon.\"",
	"\"Early, they are, the whole lot of them. Hard to say if it is the weather or something chasing them.\"",
}
var mgLineLate = []string{
	"\"The herds are late this year. We keep looking at the hills and they do not come down.\"",
	"\"Late again. The old folk say a late herd means a hard winter.\"",
}
var mgLineGood = []string{
	"\"Good grazing, and the {sp} are plentiful. Folk eat well when the herds do.\"",
	"\"Fat herds this year. Even the {sp} are easy to find.\"",
}
var mgLineThin = []string{
	"\"The {sp} are thin on the ground. Something has been at them, or they have gone elsewhere.\"",
	"\"You will not find many {sp} around {place} this year. I would not count on the hunt.\"",
}
var mgLineOver = []string{
	"\"Too many mouths on the grass. The herds have eaten the pasture bare around {place}, and the barley price shows it.\"",
}
var mgLinePred = []string{
	"\"The {sp} are getting bold. Two of my neighbours lost stock this month.\"",
	"\"It is a bad year for the wolves, and the wolves are not the only ones. Keep to the road after dusk.\"",
}
var mgLineLean = []string{
	"\"Lean year. When the game is gone the hunters come to the farms, and the beasts do too.\"",
}
var mgLineBirds = []string{
	"\"The birds were overhead for three days. When they pass, the weather turns.\"",
}
var mgLineFish = []string{
	"\"The fish are running in the river. Half the village is out with nets.\"",
}
var mgLineDenNew = []string{
	"\"Something new moved into the old den near {place}. The tracks are {sp}. Be careful out that way.\"",
}
var mgLineDenOld = []string{
	"\"They say a wanderer cleared the den near {place}. We have slept better since.\"",
}
var mgLineQuiet = []string{
	"\"Nothing out of the ordinary. The herds do what herds do.\"",
}

var mgRaidText = []string{
	"Howling and a clatter of stock-bells: something has come down from the hills into the fields.",
	"A farmer's shout carries across the furrows. Lean times have pushed {sp} out of the wild.",
	"Shapes move along the fence line in the dusk. The animals have come looking for easy meat.",
}

var mgRaidWithdraw = "The hunters slink back toward the hills."
