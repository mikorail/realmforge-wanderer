package engine

// paEntry is one row of the command palette. Running it returns to the main screen and then either
// presses Key (exactly as the player would) or calls Func. Cond, when set, hides the entry.
type paEntry struct {
	Name     string
	Keywords string
	Cat      string
	Key      string // key sent to g.Key
	Label    string // binding shown at the right; defaults to Key
	Func     func(g *Game)
	Cond     func(g *Game) bool
}

var paEntries = []paEntry{
	// Screens
	{Name: "Inventory", Keywords: "items bag gear equipment", Cat: "Screen", Key: "i"},
	{Name: "Character", Keywords: "stats attributes sheet level", Cat: "Screen", Key: "c"},
	{Name: "Journal", Keywords: "quests notes log diary", Cat: "Screen", Key: "j"},
	{Name: "Reputation", Keywords: "factions standing fame", Cat: "Screen", Key: "r"},
	{Name: "Message log", Keywords: "history messages", Cat: "Screen", Key: "m"},
	{Name: "Skills", Keywords: "train skill tree points", Cat: "Screen", Key: "K"},
	{Name: "Mutations", Keywords: "mutation points", Cat: "Screen", Key: "V"},
	{Name: "Liquids", Keywords: "water drink flasks", Cat: "Screen", Key: "P"},
	{Name: "Hunts", Keywords: "bounty hunt trophies", Cat: "Screen", Key: "B"},
	{Name: "News", Keywords: "rumours rumors tidings", Cat: "Screen", Key: "N"},
	{Name: "Thought cabinet", Keywords: "thoughts mind ideas", Cat: "Screen", Key: "J"},
	{Name: "Quest spine", Keywords: "main story chain", Cat: "Screen", Key: "Q"},
	{Name: "Ledger", Keywords: "debts coin accounts money", Cat: "Screen", Key: "U"},
	{Name: "Court", Keywords: "politics offices crown", Cat: "Screen", Key: "A"},
	{Name: "Hunters", Keywords: "bounty hunters pursuit", Cat: "Screen", Key: "I"},
	{Name: "Endgame war room", Keywords: "end game finale", Cat: "Screen", Key: "E"},
	{Name: "Ship", Keywords: "sail boat vessel naval", Cat: "Screen", Key: "Z"},
	{Name: "Murim sects", Keywords: "martial sect clan wuxia", Cat: "Screen", Key: "G"},
	{Name: "Brewing", Keywords: "alchemy potions brew", Cat: "Screen", Key: "F4"},
	{Name: "Sleep", Keywords: "bed rest fatigue", Cat: "Screen", Key: "F5"},
	{Name: "Trade ledger", Keywords: "trade routes goods", Cat: "Screen", Key: "F6"},
	{Name: "Family", Keywords: "spouse children heirs kin", Cat: "Screen", Key: "F7"},
	{Name: "Arcane", Keywords: "magic spells arcana", Cat: "Screen", Key: "F8"},
	{Name: "Collections", Keywords: "clutter curios trophies", Cat: "Screen", Key: "F9"},
	{Name: "Journal+ places and achievements", Keywords: "achievements places visited", Cat: "Screen", Key: "F10"},
	{Name: "World map", Keywords: "travel overworld ascend", Cat: "Screen", Key: "<"},
	// Crafting and settlement
	{Name: "Cook", Keywords: "recipe meal food", Cat: "Craft", Key: "C"},
	{Name: "Workbench", Keywords: "craft crafting smith make", Cat: "Craft", Key: "T"},
	{Name: "Dig", Keywords: "shovel bury excavate", Cat: "Craft", Key: "D"},
	{Name: "Home", Keywords: "stash house base", Cat: "Place", Key: "H", Label: "H / F3"},
	{Name: "Return home", Keywords: "travel house base", Cat: "Place", Key: "F3"},
	{Name: "Companions", Keywords: "allies party followers", Cat: "People", Key: "o"},
	{Name: "Orders", Keywords: "command companions follow wait", Cat: "People", Key: "O"},
	{Name: "War", Keywords: "army battle conflict", Cat: "People", Key: "W"},
	{Name: "Steal", Keywords: "pickpocket theft thief", Cat: "People", Key: "X"},
	// Actions
	{Name: "Look", Keywords: "examine inspect cursor", Cat: "Action", Key: "x", Label: "x / L"},
	{Name: "Pick up", Keywords: "take grab loot", Cat: "Action", Key: "g", Label: "g / ,"},
	{Name: "Rest until healed", Keywords: "sleep heal wait", Cat: "Action", Key: "R"},
	{Name: "Rest a while", Keywords: "wait sleep", Cat: "Action", Key: "z"},
	{Name: "Fire missile", Keywords: "shoot bow ranged aim", Cat: "Action", Key: "f"},
	{Name: "Aim by hand", Keywords: "shoot manual cursor", Cat: "Action", Key: "F"},
	{Name: "Throw", Keywords: "grenade hurl", Cat: "Action", Key: "t"},
	{Name: "Search", Keywords: "traps secret doors", Cat: "Action", Key: "s"},
	{Name: "Eat", Keywords: "food hunger", Cat: "Action", Key: "e"},
	{Name: "Drink", Keywords: "water thirst", Cat: "Action", Key: "d"},
	{Name: "Wait one turn", Keywords: "pass skip", Cat: "Action", Key: "."},
	{Name: "Abilities", Keywords: "powers menu artifacts", Cat: "Action", Key: "a"},
	{Name: "Descend", Keywords: "stairs down", Cat: "Action", Key: ">"},
	// System
	{Name: "Cheat console", Keywords: "cheats debug commands god", Cat: "System", Key: "/"},
	{Name: "Sandbox", Keywords: "cheats debug menu", Cat: "System", Key: "F2"},
	{Name: "Help", Keywords: "keys controls manual", Cat: "System", Key: "?"},
	{Name: "Save", Keywords: "write game", Cat: "System", Key: "S"},
	{Name: "Settings", Keywords: "options preferences palette animations", Cat: "System", Label: "`", Func: func(g *Game) {
		g.Msg("Press ` to open Settings.")
	}},
}
