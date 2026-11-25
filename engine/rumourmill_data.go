package engine

// Rumour mill data (prefix rm). Each kind has a full telling T (with names) and a blurred telling B
// (names dropped) that a rumour decays into after a few retellings. Placeholders: {a} {b} {place} {n} {x}.
// {place} turns into "somewhere to the north" when the detail is lost on the road.

type rmTpl struct {
	T, B string
	Tag  string // short label on the screen
	Life int    // days it stays in the air (0 = default)
	Base int    // how often the source gets it right, 0..100
}

var rmTpls = map[string]rmTpl{
	"battle": {T: "{a} and {b} have come to blows near {place}; folk say {n} are dead.", B: "Soldiers have clashed near {place}; folk say {n} are dead.", Tag: "war", Base: 80},
	"muster": {T: "{a} is mustering {n} spears on the {b} border, near {place}.", B: "Spears are mustering on a border near {place}, {n} of them by one count.", Tag: "war", Base: 75},
	"death":  {T: "{a}, {x} of {place}, is dead: {b}.", B: "Someone of standing in {place} has died: {b}.", Tag: "death", Base: 92},
	"raid":   {T: "Raiders struck the {a} company on the road toward {place}; {n} wagons are lost.", B: "A caravan was hit on the road toward {place}.", Tag: "road", Base: 75},
	"lost":   {T: "The {a} company never came in to {place}; carters found {n} burnt wagons and no one to ask.", B: "A whole caravan has gone missing on the road to {place}.", Tag: "road", Base: 70},
	"beast":  {T: "{a} has been seen near {place}; {x}.", B: "Something big and hungry has been seen near {place}; {x}.", Tag: "beast", Base: 70},
	"counter": {T: "Armed strangers have been asking after a wanderer and paying well for word of the road they take.", B: "Hard-looking strangers are asking after someone on the road.", Tag: "hunt", Life: 12, Base: 60},
	"regrow": {T: "{a} {x}, so travellers say.", B: "Something has happened to {a}: travellers say it {x}.", Tag: "town", Base: 85},
	"feast":  {T: "{a} keeps the {x} today; the road to {place} is thick with pilgrims.", B: "A feast is being kept at {place}, and the roads are thick with pilgrims.", Tag: "feast", Life: 4, Base: 95},
	"free":   {T: "{x}", B: "{x}", Tag: "tidings", Base: 88},
	"ev:plague":      {T: "Sickness has taken hold around {place}; some {n} are said to lie ill.", B: "A sickness is abroad near {place}; the numbers are whispered.", Tag: "plague", Base: 85},
	"ev:invasion":    {T: "Foreign soldiers have crossed over near {place}; they number {n}, or so the runners say.", B: "Strangers in arms have crossed a border near {place}.", Tag: "war", Base: 80},
	"ev:desertion":   {T: "A garrison near {place} has walked off; {n} men took their pay and went home.", B: "Soldiers are deserting near {place}.", Tag: "war", Base: 80},
	"ev:famine":      {T: "Granaries stand empty around {place}; {n} families are on the roads.", B: "There is hunger near {place}.", Tag: "famine", Base: 85},
	"ev:bandits":     {T: "A band of {n} has been taking tolls by force near {place}.", B: "Bandits are working the road near {place}.", Tag: "road", Base: 80},
	"ev:boom":        {T: "Trade is booming at {place}; merchants are paying double for porters.", B: "Money is flowing at {place}.", Tag: "trade", Base: 85},
	"ev:bust":        {T: "The market at {place} has collapsed; {n} stalls stand shuttered.", B: "Trade has collapsed near {place}.", Tag: "trade", Base: 85},
	"ev:slavepen":    {T: "Whispers say a pen of chained people is kept near {place}, {n} by the last count.", B: "There is talk of a slave pen near {place}.", Tag: "crime", Base: 70},
}

var rmEvKinds = map[string]string{"plague": "ev:plague", "invasion": "ev:invasion", "desertion": "ev:desertion", "famine": "ev:famine", "bandits": "ev:bandits", "boom": "ev:boom", "bust": "ev:bust", "slavepen": "ev:slavepen"}

var rmBeastDeeds = []string{
	"it has taken {n} head of stock this month",
	"drovers will not use the pass after dark",
	"it has been tearing up fences, {n} now",
	"the herders say it is bigger than any wolf",
}

var rmBlurWho = []string{"a captain", "someone of rank", "an old hand", "a person of note"}

var rmCauseAlt = []string{"a knife in the dark", "a sudden fever", "a fall from a horse", "bad wine", "a quarrel that went too far"}

var rmDirs = []string{"east", "south-east", "south", "south-west", "west", "north-west", "north", "north-east"}

var rmTellers = []string{
	"\"I had it from a carter who had it from a ferryman:\"",
	"\"They are saying in the market:\"",
	"\"Keep this quiet, but:\"",
	"\"Everyone has it, so you may as well too:\"",
	"\"A pedlar came through at dawn and told us:\"",
}
