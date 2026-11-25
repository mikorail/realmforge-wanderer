package engine

// Data tables for workshop.go (prefix wk): craft lines, techniques, rank titles, the work orders each line
// can turn out, quality words, station words and the master's teaching lines. Text only; no rules here.

// wkTech is one technique of a craft line; a line has five, learned in order from masters.
type wkTech struct {
	Name string
	Text string
}

// wkLine is a craft line: which attribute drives its checks, which roles teach it, what the errand asks for.
type wkLine struct {
	ID, Name, Trade string
	Attr            int
	Mat             string // errand material, in words
	Techs           [5]wkTech
}

var wkLines = []wkLine{
	{ID: "smith", Name: "Smithing", Trade: "smith", Attr: STR, Mat: "scrap metal or ore", Techs: [5]wkTech{
		{"Even heats", "Steady fires. Your work is less likely to come out poor."},
		{"Folded metal", "Layered blades hold an edge. Weapons you forge are one check easier."},
		{"Quenching", "The right water at the right moment. Armour you forge takes wear more slowly."},
		{"Edge and temper", "A smith's eye for the last touch. Fine work comes within reach on any anvil."},
		{"The master's mark", "Your mark goes on the blade. Masterwork is possible and repairs cost you less."},
	}},
	{ID: "tailor", Name: "Tailoring", Trade: "tailor", Attr: AGI, Mat: "cloth, thread or pelts", Techs: [5]wkTech{
		{"Clean seams", "Straight stitches and honest hems."},
		{"Cut to the body", "Garments fit; clothing you make sits better on the wearer."},
		{"Lining and padding", "Soft armour you sew takes wear more slowly."},
		{"Dye and pattern", "Cloth that earns a second look. Fine work on any bench."},
		{"The master's cut", "Your cut is known. Masterwork is possible and mending costs you less."},
	}},
	{ID: "alchemist", Name: "Alchemy", Trade: "apothecary", Attr: INT, Mat: "herbs or reagents", Techs: [5]wkTech{
		{"Measures and weights", "You stop guessing the doses."},
		{"Clean glass", "Less waste in every batch."},
		{"Slow steeping", "Draughts come out stronger when you have the time."},
		{"Distilled essence", "Fine work comes within reach on any bench."},
		{"The master's receipt", "A receipt that others copy. Masterwork is possible."},
	}},
	{ID: "tanner", Name: "Tanning", Trade: "tanner", Attr: AGI, Mat: "raw hides or pelts", Techs: [5]wkTech{
		{"Scraping and salting", "Hides come off the rack without spoiling."},
		{"Oak bark", "Leather you work is tougher."},
		{"Smoke curing", "Leather armour takes wear more slowly."},
		{"Fine grain", "Fine work comes within reach on any bench."},
		{"The master's hide", "Your leather is known by touch. Masterwork is possible."},
	}},
	{ID: "cook", Name: "Cooking", Trade: "baker", Attr: WIL, Mat: "meat or herbs", Techs: [5]wkTech{
		{"Heat control", "Fewer burnt dinners."},
		{"Stocks and sauces", "Plain food tastes like more than it is."},
		{"Preserving", "What you cook keeps."},
		{"Seasoning", "Fine work comes within reach on any hearth."},
		{"The master's table", "People travel for your table. Masterwork is possible."},
	}},
	{ID: "carpenter", Name: "Carpentry", Trade: "carpenter", Attr: STR, Mat: "timber or planks", Techs: [5]wkTech{
		{"True cuts", "Square, level, plumb."},
		{"Joinery", "Shafts and staves you shape are one check easier."},
		{"Steaming and bending", "Bows and bent work come out better."},
		{"Seasoned wood", "Fine work comes within reach on any bench."},
		{"The master's joint", "Nothing you build comes apart. Masterwork is possible."},
	}},
}

// wkRankTitle by rank 0..5.
var wkRankTitle = [6]string{"Unschooled", "Novice", "Apprentice", "Journeyman", "Adept", "Master"}

// wkXPAt is the practice needed (cumulative) before a master will teach the technique of rank n+1.
var wkXPAt = [5]int{25, 90, 200, 380, 650}

// wkTradeLine maps a lives trade (lvPerson.Role) to the craft line it can teach.
var wkTradeLine = map[string]string{
	"smith": "smith", "smelter": "smith",
	"tailor": "tailor", "weaver": "tailor", "cobbler": "tailor",
	"healer": "alchemist", "apothecary": "alchemist",
	"tanner": "tanner",
	"baker":  "cook", "innkeeper": "cook",
	"carpenter": "carpenter", "mason": "carpenter",
}

// wkRoleLine maps an actor role to the repair work it can do.
var wkRoleLine = map[string]string{"smith": "smith", "tailor": "tailor", "tanner": "tanner"}

// wkQuality words: index q+1 for q = -1..2.
var wkQWord = [4]string{"poor", "fair", "fine", "masterwork"}
var wkQValue = [4]int{60, 100, 140, 220}

// wkToolWord by tier 0..3.
var wkToolWord = [4]string{"hand tools only", "a basic bench", "a full shop", "a master's shop"}

// wkHomeUpgrade is the price to raise a home bench to tier 1..3 (base units).
var wkHomeUpgrade = [4]int{0, 0, 180, 520}

// wkFabric / wkHide / wkWood name materials by item tier 0..5.
var wkFabric = [6]string{"linen", "wool", "quilted", "silk", "brocade", "weave-lined"}
var wkHide = [6]string{"rawhide", "cured", "boiled", "oiled", "studded", "dragon-grained"}

// wkRegionalNouns: the gear a culture's signature piece is, by line (fallback when the pack names nothing).
var wkRegionalNouns = map[string][]string{
	"smith":     {"blade", "axe", "knife", "mace"},
	"tailor":    {"festival coat", "robe", "sash-coat", "long vest"},
	"tanner":    {"riding boots", "walking boots", "travel boots"},
	"alchemist": {"remedy", "tincture", "draught"},
	"cook":      {"feast dish", "house stew", "traveller's pot"},
	"carpenter": {"hunting bow", "war-bow", "field bow"},
}

// wkErrandLines: what a master says when giving an errand.
var wkErrandLines = []string{
	"\"Bring me %d lots of %s and we will see about it.\"",
	"\"I am short. %d lots of %s, and I will show you the next step.\"",
	"\"Words are cheap. %d lots of %s are not.\"",
}

// wkTeachLines: what a master says when teaching.
var wkTeachLines = []string{
	"\"Watch my hands. Again. Now you.\"",
	"\"It is not strength. It is timing.\"",
	"\"Mistakes are the lesson. Make them here, not on a customer.\"",
	"\"Good. Do it two hundred times and it will be yours.\"",
}
