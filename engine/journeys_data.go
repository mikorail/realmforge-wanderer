package engine

// Data for journeys.go (prefix jy): sky craft with genre skins, passenger-ride flavour text.

// jyDef is one kind of flying machine or beast. Kmd is km per day in still air, Wind how much
// the wind helps and hinders (0..1), Risk the base chance of trouble per leg, FuelMax the days of
// lift/fuel it carries (0 = needs none), Cap the people it lifts (you plus companions).
type jyDef struct {
	ID      string
	Names   map[string]string // genre → name; "" is the default
	Gen     string            // space-separated genres it is sold in; "*" = anywhere
	Kmd     float64
	Cap     int
	FuelMax float64
	Wind    float64
	Risk    float64
	Price   int
	Rent    int // hire for six days
	Hull    int
	Float   bool // can alight on water
	Fuel    map[string]string
	Blurb   string
}

var jyDefs = []jyDef{
	{ID: "balloon", Gen: "*", Kmd: 80, Cap: 3, FuelMax: 4, Wind: 0.95, Risk: 0.05, Price: 420, Rent: 90, Hull: 60,
		Names: map[string]string{"": "hot-air balloon", "steampunk": "brass gas-balloon", "dieselpunk": "barrage-balloon gondola", "mythic": "wind-bag of Aeolus", "solarpunk": "sun-heated balloon",
			"biopunk": "gas-bladder floater", "twilight": "lantern balloon", "wasteland": "patched scrap balloon", "atompunk": "weather-balloon pod", "cyberpunk": "lifting-gas pod"},
		Fuel:  map[string]string{"": "burner fuel", "steampunk": "lift gas", "dieselpunk": "hydrogen", "mythic": "stored wind", "solarpunk": "sun-heat", "biopunk": "marsh gas", "twilight": "lamp oil", "wasteland": "scavenged propane", "atompunk": "helium", "cyberpunk": "helium cells"},
		Blurb: "Cheap and quiet, but it goes where the wind goes."},
	{ID: "airship", Gen: "steampunk dieselpunk atompunk solarpunk mythic fantasy", Kmd: 150, Cap: 8, FuelMax: 7, Wind: 0.35, Risk: 0.04, Price: 1900, Rent: 380, Hull: 120,
		Names: map[string]string{"": "sky-galleon", "steampunk": "steam airship", "dieselpunk": "zeppelin", "atompunk": "atomic skyliner", "solarpunk": "sun-sail airship", "mythic": "sky-trireme"},
		Fuel:  map[string]string{"": "lift-crystals", "steampunk": "coal and gas", "dieselpunk": "diesel and hydrogen", "atompunk": "reactor rods", "solarpunk": "stored light", "mythic": "ichor"},
		Blurb: "Fast, roomy and sturdy; ruinous to buy."},
	{ID: "gryphon", Gen: "fantasy mythic twilight biopunk", Kmd: 130, Cap: 2, FuelMax: 4, Wind: 0.4, Risk: 0.06, Price: 950, Rent: 190, Hull: 50,
		Names: map[string]string{"": "griffon-post mount", "mythic": "winged horse", "twilight": "night-owl courier", "biopunk": "grown glider-beast"},
		Fuel:  map[string]string{"": "raw meat", "mythic": "ambrosia oats", "twilight": "mice and moths", "biopunk": "nutrient slurry"},
		Blurb: "A trained flyer: rests and feeds, lands anywhere with a ledge."},
	{ID: "kite", Gen: "solarpunk biopunk wasteland fantasy twilight", Kmd: 65, Cap: 1, FuelMax: 0, Wind: 1.0, Risk: 0.07, Price: 230, Rent: 50, Hull: 35,
		Names: map[string]string{"": "storm-kite", "solarpunk": "kite-sail", "biopunk": "membrane sail", "wasteland": "scrap sail-kite", "twilight": "shadow-silk kite"},
		Blurb: "No fuel at all, but you are at the wind's mercy."},
	{ID: "jetpack", Gen: "atompunk cyberpunk wasteland dieselpunk", Kmd: 210, Cap: 1, FuelMax: 1.6, Wind: 0.1, Risk: 0.09, Price: 1250, Rent: 260, Hull: 40,
		Names: map[string]string{"": "rocket pack", "atompunk": "jet-pack", "cyberpunk": "thruster rig", "wasteland": "salvaged rocket pack", "dieselpunk": "rocket harness"},
		Fuel:  map[string]string{"": "rocket fuel", "atompunk": "hydrazine", "cyberpunk": "power cells", "wasteland": "moonshine fuel", "dieselpunk": "kerosene"},
		Blurb: "Very fast, very thirsty, very loud."},
	{ID: "drone", Gen: "cyberpunk atompunk solarpunk", Kmd: 105, Cap: 2, FuelMax: 3, Wind: 0.3, Risk: 0.04, Price: 820, Rent: 160, Hull: 55,
		Names: map[string]string{"": "lifter drone", "cyberpunk": "cargo drone", "atompunk": "ducted-fan lifter", "solarpunk": "solar quadcopter"},
		Fuel:  map[string]string{"": "charge", "cyberpunk": "battery charge", "atompunk": "isotope charge", "solarpunk": "sun-charge"},
		Blurb: "Steady and wind-tolerant; the batteries are the limit."},
	{ID: "cloudboat", Gen: "mythic fantasy twilight", Kmd: 100, Cap: 4, FuelMax: 0, Wind: 0.6, Risk: 0.03, Price: 1500, Rent: 300, Hull: 80, Float: true,
		Names: map[string]string{"": "cloud skiff", "mythic": "cloud-boat", "twilight": "mist-barge"},
		Blurb: "Needs no fuel and floats on water as well as air."},
}

var jyGenres = map[string]bool{"mythic": true, "steampunk": true, "dieselpunk": true, "atompunk": true, "cyberpunk": true, "biopunk": true, "solarpunk": true, "wasteland": true, "twilight": true}

// jyGliderNames are the personal cliff gliders.
var jyGliderNames = map[string]string{"": "wind-glider", "steampunk": "canvas-and-brass glider", "dieselpunk": "hang-wing", "mythic": "Daedalus wings", "solarpunk": "kite-wing", "cyberpunk": "wingsuit",
	"atompunk": "delta wing", "biopunk": "membrane glider", "wasteland": "scrap hang-glider", "twilight": "bat-silk glider"}

var jyRiggerTitle = map[string]string{"": "rigger", "steampunk": "aeronaut-rigger", "dieselpunk": "hangar chief", "mythic": "windwright", "solarpunk": "sail-rigger", "cyberpunk": "drone wrangler",
	"atompunk": "hangar tech", "biopunk": "beast-handler", "wasteland": "scrap rigger", "twilight": "owl-keeper"}

// ---- rides ----

var jyCampTalk = []string{
	"a drover swears the river here changes course every spring",
	"the cook tells a long story about a mule that could count",
	"a pilgrim shares a flask and a rumour of treasure in the hills",
	"two guards argue over which inn on the road serves the worst stew",
	"a child asks whether the stars are the same in the next kingdom",
	"an old merchant explains, at length, how toll-keepers cheat",
	"someone hums a road song and the whole camp slowly joins in",
	"a tinker mends a wheel-rim and sells a charm against bad axles",
}

var jyFerryTalk = []string{
	"the ferryman points out a sandbank that has wrecked three barges",
	"a fishwife offers smoked eel and opinions on the harbourmaster",
	"a passenger swears something large and pale follows the boat at night",
	"the poleman sings a counting song to keep the rhythm",
	"a cheesemonger and a priest play dice on an upturned crate",
	"a heron stands on the prow for a mile as if it had paid fare",
}

var jyRockfall = []string{
	"A rockfall blocks the track: the carters stand about shouting.",
	"A slope has let go across the way, and the lead wagon stops dead.",
}

var jySick = []string{"a young drover", "an old pilgrim", "the cook's wife", "a trader's boy", "a hired guard"}

var jyThiefName = []string{"a quiet traveller", "a hired hand", "a cheerful pedlar"}
