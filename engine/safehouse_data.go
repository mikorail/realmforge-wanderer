package engine

// Safehouse and lodging data (prefix sy). Text and numbers only.

type syUpgDef struct {
	ID     string
	Name   string
	Max    int
	Cost   []int // base units per level
	Days   []int // build days per level
	Desc   string
	Upkeep int // extra weekly upkeep per level
}

var syUpgrades = []syUpgDef{
	{ID: "lock", Name: "Better locks", Max: 3, Cost: []int{80, 160, 300}, Days: []int{1, 2, 3}, Upkeep: 0,
		Desc: "Each grade cuts the weekly burglary chance."},
	{ID: "hidden", Name: "Hidden stash", Max: 1, Cost: []int{220}, Days: []int{3}, Upkeep: 2,
		Desc: "A false panel and a hollow behind it. Burglars never find what lies there."},
	{ID: "tunnel", Name: "Bolt-hole tunnel", Max: 1, Cost: []int{380}, Days: []int{5}, Upkeep: 4,
		Desc: "A way out under the floor. A raid will not catch you in bed, and you can slip out when hunted."},
	{ID: "kitchen", Name: "Rain barrel and kitchen", Max: 1, Cost: []int{90}, Days: []int{2}, Upkeep: 1,
		Desc: "Water and a hearth: lying low costs you much less hunger and thirst."},
	{ID: "bed", Name: "A proper bed", Max: 1, Cost: []int{70}, Days: []int{1}, Upkeep: 0,
		Desc: "Sleep is better: lying low cools you faster."},
}

var syStayOptions = []int{1, 3, 7, 14, 30}

var syInnLines = []string{
	"\"A room? Of course. Clean sheets and a chest with a lock. The key stays with you.\"",
	"\"We have a room at the back, quiet and out of the draught. Per night, and I do not haggle on nights.\"",
	"\"A bed and a lockable chest. If you do not come back, I keep your things for a fortnight, then I sell them. Fair?\"",
}

var syHeldLines = []string{
	"The innkeeper is holding your things until day %d.",
	"Your chest is held by the innkeeper. They sell it on day %d.",
}

var syRaidLines = []string{
	"The watch kicked in the door of your %s looking for you.",
	"Somebody tipped off the guards about your %s.",
}

var syBurgLines = []string{
	"Thieves visited your %s while you were away.",
	"Your %s was broken into. Locks hold off fewer thieves than you would like.",
}

var syNewsRaid = "The watch searched a house at the edge of town after a tip-off, but found nobody home."
var syNewsSold = "The innkeeper sold the goods of a guest who never came back."
