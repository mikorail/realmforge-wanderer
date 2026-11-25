package engine

// Data tables for hunting.go (prefix hn): weights, rank titles, sign words, grade words, lodge and law lines.

// hnSizeKg is the typical live weight in kg by bestiary size 1..7.
var hnSizeKg = [8]int{0, 1, 8, 45, 160, 500, 2000, 8000}

var hnRankTitle = [6]string{"Greenhorn", "Tracker", "Stalker", "Woodsman", "Master tracker", "Legend of the hunt"}

// hnXPAt is the tracking practice needed for each next rank.
var hnXPAt = [5]int{20, 70, 160, 320, 560}

// hnGradeWord by field-dressing grade 0..3; hnGradeValue is the percent of value kept.
var hnGradeWord = [4]string{"ragged", "clean", "good", "prime"}
var hnGradeValue = [4]int{55, 100, 140, 210}
var hnMountWord = [4]string{"patchy", "fair", "handsome", "magnificent"}

// hnSigns: what each group leaves behind, to read.
var hnSigns = map[string][]string{
	"mam":  {"fresh prints", "droppings", "rubbed bark", "a bedded patch of grass", "torn undergrowth"},
	"bird": {"scattered feathers", "scratchings in the dirt", "a dust bath", "droppings under a branch"},
	"rept": {"a sinuous belly-trail", "a shed skin", "a flattened track", "scored earth"},
	"dino": {"deep three-toed prints", "broken saplings", "great droppings", "a trampled swath"},
	"":     {"tracks", "droppings", "disturbed ground"},
}

var hnSizeWord = [8]string{"", "tiny", "small", "light", "heavy", "huge", "enormous", "colossal"}

// hnOwner is the word for who holds a preserve, by ideology.
var hnOwner = map[string]string{
	"monarchy": "the crown", "autocracy": "the ruler", "theocracy": "the temple", "oligarchy": "the founding houses",
	"technocracy": "the ministry",
}

var hnLodgeGreet = []string{
	"\"Boots dry? Good. A hunter with dry boots is a hunter who is listening.\"",
	"\"Game is moving. Ask, and I will tell you what is open and what is not.\"",
	"\"The lodge keeps the law and keeps the larder. Which do you need?\"",
}

var hnLodgeThanks = []string{
	"\"Good dressing. That is worth the price.\"",
	"\"Clean work. I will hang the hides myself.\"",
	"\"Fair weights, fair price. Come again.\"",
}

var hnRefusePoach = "\"No. I will not touch anything with a poacher's mark on it. Take it to a fence.\""
