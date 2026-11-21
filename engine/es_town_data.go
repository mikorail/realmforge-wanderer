package engine

// Text tables for es_town.go (prefix eh): graves, memorials and houses with a story. Tokens in these
// strings are replaced by ehRep: [n] the dead, [k] a living relative, [t] the town, [y] a year, [c] a cause,
// [w] a war or plague name, [r] the realm, [j] a trade.

// ehCauseLine: the line under the dates, by kind of death.
var ehCauseLine = map[string][]string{
	"old":      {"Died full of years.", "Lived long. Went quietly.", "Old, and tired, and done."},
	"battle":   {"Died of wounds taken in battle.", "Fell under arms.", "The wounds did not close."},
	"war":      {"Fell in [w].", "Did not come back from [w]."},
	"sick":     {"Taken by [c].", "Carried off by [c]."},
	"plague":   {"Taken by [w].", "One of those [w] took."},
	"poison":   {"Poisoned.", "The cup was poisoned."},
	"spies":    {"Poisoned by [c].", "Struck down by [c]."},
	"accident": {"Killed in [c].", "Taken by [c], without warning."},
	"deposed":  {"Thrown down: [c].", "Lost the throne, then the rest: [c]."},
	"monster":  {"[c]. Nothing was found to bury.", "[c]. The grave holds only a name."},
	"murder":   {"Murdered.", "Killed by another's hand."},
	"duel":     {"Killed [c].", "Died [c]. It was called fair."},
	"gallows":  {"Hanged.", "Died on the rope."},
	"child":    {"A child of [t].", "Gone before they had grown."},
	"other":    {"Died [c].", "[c]."},
}

// ehFamily: what the family says, cut below the epitaph or told by the sexton.
var ehFamily = map[string][]string{
	"old":      {"[k] tends this stone every feast day.", "\"Rest now. You worked enough for three.\" (cut by [k])", "The grass is kept short here."},
	"battle":   {"\"We have your sword. We would rather have you.\" (cut by [k])", "[k] still sets a place at the table.", "Someone leaves a ration of bread here on the day of the battle."},
	"war":      {"\"Sent home in a letter, not a box.\" (cut by [k])", "[k] paid for the stone with a soldier's pension."},
	"sick":     {"\"The fever went quick. We were glad of that much.\" (cut by [k])", "Fresh flowers. Someone comes often."},
	"plague":   {"Buried in haste, at night, with lime.", "\"We could not hold your hand at the end.\" (cut by [k])"},
	"poison":   {"The family will not say who poured the cup.", "\"We know who. We are waiting.\" (cut by [k])"},
	"spies":    {"\"Not by any of us.\" (cut deep, then half scratched out)", "The family asked for the stone to face the border."},
	"accident": {"\"Such a stupid way to go. Forgive us for laughing at the wake.\" (cut by [k])", "\"You always rode too fast.\" (cut by [k])"},
	"deposed":  {"No family signed this stone.", "The name was chiselled off once, and cut again later."},
	"monster":  {"The stone was paid for by the town. No body came back.", "[k] still walks out to the old road at dusk."},
	"murder":   {"\"Justice. Not prayers.\" (cut by [k])", "[k] has sworn on this stone. The oath is written on the back."},
	"duel":     {"\"Pride killed you, and the other one is still walking.\" (cut by [k])", "The family wanted no blade carved on it."},
	"gallows":  {"No family came.", "Someone pushed a single coin into the earth here."},
	"child":    {"A wooden toy sits on the mound, weathered grey.", "\"Our small one.\" (cut by [k])"},
	"other":    {"[k] keeps the stone clean.", "Moss has been scraped off by hand."},
}

// ehVerse: a closing line when the culture has no pack.
var ehVerse = []string{
	"What we were, you are. What we are, you will be.",
	"The road ends here. Walk softly past.",
	"Remembered while one of us remembers.",
	"Gone ahead. We will follow when it is time.",
	"Here is the body. The rest is elsewhere.",
}

// ehToys: grave goods for a child.
var ehToys = []string{"carved wooden horse", "rag doll with button eyes", "clay whistle", "string of painted beads", "little tin soldier", "spinning top"}

// ehKeeps: small things a dead person leaves in a house that a relative would want back.
var ehKeeps = []string{"worn brass ring", "lock of hair tied with thread", "bone comb", "folded love letter", "carved pipe", "copper locket", "pair of spectacles", "prayer cord"}

// ehOfferings by feast kind, when the culture has no pack.
var ehOfferings = map[string][]string{
	"spring":    {"a sprig of green", "a bowl of milk"},
	"harvest":   {"a sheaf of grain", "a heel of new bread"},
	"dead":      {"a stub of candle", "a cup of water for the thirsty dead"},
	"midwinter": {"a lamp left burning", "a sprig of evergreen"},
	"sea":       {"a string of shells", "a fish-hook bent straight"},
	"war":       {"a bent blade", "a soldier's button"},
	"learning":  {"a page of copied text", "a stub of chalk"},
	"sun":       {"a disc of hammered tin", "a heap of yellow petals"},
	"":          {"a coin pressed into the wax", "a bundle of dried flowers"},
}

// ehGraffiti: scratched lines by event kind; [y] year, [t] town, [r] realm, [n] a name.
var ehGraffiti = map[string][]string{
	"uprising": {"[y]. WE ASKED FOR BREAD.", "REMEMBER [y]. THEY CLOSED THE GATES ON US.", "The riot of [y] started here. Count the bricks they threw."},
	"coup":     {"[y]: SOLDIERS IN THE PALACE. NOBODY VOTED.", "Whoever rules [r] this week, the rent is the same."},
	"rebel":    {"THE BANNER WILL COME BACK. [y].", "[r] is not your realm. ([y])"},
	"capture":  {"[y]. THEY CAME OVER THE WALL. WE REMEMBER.", "Under the new flag since [y]. Same old walls."},
	"faith":    {"The priests lie. Ask them about [y].", "[y]: they burned the books in this square."},
	"crime":    {"[n] IS A THIEF.", "LOCK YOUR DOORS. [n] IS STILL OUT.", "[n] robbed my mother and walks free."},
	"feud":     {"[n] MURDERER.", "Ask [n] where they were that night."},
}

// ehHouseLook: the line shown when you step into a house with a story.
var ehHouseLook = map[string]string{
	"plague":   "A cross is daubed on the door. Inside, the bedding has been dragged into a heap and burnt on the floor.",
	"burnt":    "The roof beams are black. Ash lies ankle deep and the air still smells of smoke.",
	"workshop": "Benches, tools laid out in order, a thick coat of dust over all of it. Nobody has worked here for a long time.",
	"shop":     "The shutters are nailed shut. Empty shelves, a counter with a ledger still on it.",
	"jail":     "The door was forced. Chairs are overturned and a cupboard stands open and empty.",
	"cradle":   "A small bed by the wall, a black cloth pinned above it. The room is very clean.",
	"hoard":    "You have to turn sideways to get in. Crates, sacks and broken things fill the room to the walls.",
}
