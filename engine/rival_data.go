package engine

// rival_data.go: text pools for the rival adventurer (rival.go, prefix rv).

var rvCallings = []string{"warrior", "scout", "scholar", "tinker", "mystic", "merchant", "rogue"}

// rvRoleCalling maps a notable's role to the calling they wander under.
var rvRoleCalling = map[string][]string{
	"hero":     {"warrior", "warrior", "scout"},
	"general":  {"warrior"},
	"magician": {"mystic", "scholar"},
	"merchant": {"merchant", "rogue"},
	"prophet":  {"mystic"},
}

// rvWeapon: the weapon family each calling favours.
var rvWeapon = map[string][]string{
	"warrior":  {"longblade", "axe"},
	"scout":    {"shortblade", "longblade"},
	"scholar":  {"cudgel", "shortblade"},
	"tinker":   {"axe", "cudgel"},
	"mystic":   {"cudgel", "shortblade"},
	"merchant": {"shortblade", "longblade"},
	"rogue":    {"shortblade", "shortblade", "axe"},
}

// rvAttr: the attribute a calling leans on.
var rvAttr = map[string]int{"warrior": STR, "scout": AGI, "scholar": INT, "tinker": INT, "mystic": WIL, "merchant": EGO, "rogue": AGI}

var rvEpithets = []string{
	"the Unbroken", "the Second Son", "Oathless", "the Lantern", "Salt-Hand", "the Patient", "Tallyman", "the Quiet Knife",
	"Longstride", "of the Hundred Doors", "the Late", "Gravewalker", "the Fortunate", "Ironheel", "the Unpaid", "Two-Coats",
	"the Careful", "Shortfuse", "the Gilded", "Hollow-Cheek", "the Reader", "Wolfbane", "the Debtor", "Sunder",
}

// rvAmbush: what a hostile rival says as they step out.
var rvAmbush = []string{
	"\"I told you this one was mine. You should have listened.\"",
	"\"Alone out here? Convenient. I have been wanting a word with steel.\"",
	"\"Every prize you reach for, I have already had my hands on. Tonight I take the rest.\"",
	"\"Fame has room for one name in this country. Draw.\"",
	"\"You cost me coin and standing. Pay in blood.\"",
}

var rvGreetFriendly = []string{
	"\"Well met! Still alive, then. I had a wager with myself.\"",
	"\"Sit. Whatever you are chasing, there is room for two at this table.\"",
	"\"You again. Good. The road is lonelier without a rival worth beating.\"",
}

var rvGreetWary = []string{
	"\"Hm. You. Keep your hands where I can see them and we will get along.\"",
	"\"I know that look. You are after the same thing I am.\"",
	"\"Say your piece. My patience is for people I have not raced.\"",
}

var rvGreetHostile = []string{
	"\"You have a nerve, showing your face in my town.\"",
	"\"Do not smile at me. Everything you hold, I mean to take.\"",
	"\"Walk away while you can. The next time we meet, there will be no tavern around us.\"",
}

// rvMisfortune: what kills a rival away from the player.
var rvMisfortune = []string{
	"a pack that was larger than reported", "a collapsed gallery", "a poisoned blade in the dark", "something that lived below the third level",
	"a fall into a flooded shaft", "an ambush by the very guides they hired", "a trap they had boasted of disarming", "an old guardian that would not die",
}

// rvAllyQuips: said when a rival joins for one dungeon.
var rvAllyQuips = []string{
	"\"One dungeon. Split what we find fairly, and tomorrow we are rivals again.\"",
	"\"Fine. Two blades are better than one, and I would rather you were in front of me than behind.\"",
	"\"Agreed. Do not die; I would hate to lose to a corpse.\"",
}

// rvRumours: ambition lines a rival drops when asked what they are after.
var rvAimLines = map[string]string{
	"travel": "\"I am bound for %s. Business.\"",
	"delve":  "\"%s. There is a purse in it, and a name.\"",
	"hunt":   "\"%s. The bounty is fat and the beast is fat-headed.\"",
	"idle":   "\"Nothing yet. Resting, listening, deciding.\"",
}
