package engine

// Static tables for cultgoods.go (prefix cg).

// cgDrinkLiquid maps words in a pack drink name to a liquid of liquids.go (lqTable keys).
var cgDrinkLiquid = []struct{ Word, Liquid string }{
	{"mead", "honey"}, {"honey", "honey"}, {"nectar", "honey"},
	{"wine", "wine"}, {"ale", "wine"}, {"beer", "wine"}, {"spirit", "wine"}, {"liquor", "wine"}, {"brandy", "wine"},
	{"sake", "wine"}, {"cider", "wine"}, {"kumis", "wine"}, {"arrack", "wine"}, {"rum", "wine"}, {"grog", "wine"},
	{"brew", "wine"}, {"toddy", "wine"}, {"gin", "wine"}, {"whisky", "wine"}, {"whiskey", "wine"}, {"vodka", "wine"},
}

// cgBiasSlant: how a work leans on its subject (for noticing that two works on one event disagree).
var cgBiasSlant = map[string]int{"propagandist": 1, "partisan": 1, "pious": 1, "romantic": 1, "cynical": -1, "neutral": 0}

var cgVerseIntro = []string{
	"{name} clears their throat. \"Here is a verse of our people, the kind we say at the hearth.\"",
	"\"A verse? Gladly.\" {name} settles and recites in a low, even voice.",
	"{name} closes their eyes a moment, then speaks the lines the way they were taught.",
	"\"You asked for it.\" {name} straightens and delivers the verse as a singer would.",
}

var cgVerseThanks = []string{
	"They look pleased that you listened to the end.",
	"\"Say it to the next person who asks,\" they add. \"That is how it lives.\"",
	"A few heads nearby nod along with the last line.",
}

var cgCraftRoles = map[string]bool{"smith": true, "tailor": true, "artisan": true, "armorer": true, "weaver": true, "potter": true}
var cgDishRoles = map[string]bool{"merchant": true, "innkeeper": true, "tavern": true}
var cgGoodsRoles = map[string]bool{"merchant": true}
var cgVerseRoles = map[string]bool{"bard": true, "innkeeper": true, "tavern": true, "minstrel": true}

// cgGoodWeight: carrying weight of a trade good by its pack kind.
var cgGoodWeight = map[string]int{"luxury": 1, "drug": 1, "craft": 2, "food": 2, "tool": 3, "timber": 4, "ore": 3, "metal": 3, "cloth": 1}
