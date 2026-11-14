package engine

// Planet trait tables (prefix pt): the trait catalogue, which gear counters which trait, and the
// planet-kind hints that bias the deterministic pick. Pure data, no behaviour.

type pltTraitDef struct {
	ID      string
	Name    string // sidebar / star map name
	Tag     string // short sidebar tag
	Desc    string
	Counter string // what protects you (shown in warnings)
	Harsh   bool   // never the start planet's trait
	Kinds   []string
}

var pltTraitDefs = []pltTraitDef{
	{ID: "high_gravity", Name: "High Gravity", Tag: "heavy", Harsh: true,
		Desc:    "Everything weighs more: slower travel and walking, doubled fall damage.",
		Counter: "an exo-frame, power frame, grav-boots or anti-gravity charm",
		Kinds:   []string{"giant", "dense", "super", "iron", "heavy"}},
	{ID: "aether_rich", Name: "Aether-Rich", Tag: "aether",
		Desc:    "Spells and psionics recover faster, but raw aether twists flesh now and then.",
		Counter: "a grounding charm, ward or aether-dampener against the mutations",
		Kinds:   []string{"aether", "ley", "arcane", "magic", "crystal", "psi"}},
	{ID: "night_day_halves", Name: "Split Sky", Tag: "split sky",
		Desc:    "The north half lies in eternal day (thirst, glare), the south half in eternal night (dark, cold).",
		Counter: "a lantern or night-eye for the dark half, a veil or canteen for the lit half",
		Kinds:   []string{"twilight", "binary", "split", "eclipse"}},
	{ID: "ocean_world", Name: "Ocean World", Tag: "oceans",
		Desc:    "Flooded lowlands and shallows everywhere; boats are the road, wading soaks you.",
		Counter: "a boat, raft or canoe, or waterproof gear",
		Kinds:   []string{"ocean", "water", "sea", "archipelago", "island", "aqua", "flood"}},
	{ID: "storm_world", Name: "Storm World", Tag: "storms",
		Desc:    "Lightning walks the open ground all day long.",
		Counter: "a lightning rod, insulated or rubber gear, a grounding chain",
		Kinds:   []string{"storm", "tempest", "thunder", "gas giant", "cyclone"}},
	{ID: "fungal_bloom", Name: "Fungal Bloom", Tag: "spores",
		Desc:    "Spore mats carpet the ground and the air is thick with spores that sicken the unmasked.",
		Counter: "a mask, respirator, filter or spore-ward",
		Kinds:   []string{"fungal", "spore", "swamp", "jungle", "bloom", "mycel"}},
	{ID: "frozen", Name: "Frozen World", Tag: "frozen", Harsh: true,
		Desc:    "Ice and snow to the horizon; cold bites anyone without proper gear.",
		Counter: "furs, a parka, thermal gear, a heater or an ember meal",
		Kinds:   []string{"ice", "frozen", "frost", "snow", "glacier", "cold", "tundra"}},
	{ID: "war_ruined", Name: "Ruined by War", Tag: "war-torn",
		Desc:    "Old battlefields strew the lands with rubble and salvage: scavenging pays.",
		Counter: "",
		Kinds:   []string{"war", "battle", "ruin", "scorched", "wasteland", "glass"}},
	{ID: "sacred", Name: "Sacred World", Tag: "holy",
		Desc:    "Shrines dot the wilds and rest in town or beside an altar mends body and spirit.",
		Counter: "",
		Kinds:   []string{"holy", "sacred", "pilgrim", "temple", "divine", "garden", "eden"}},
	{ID: "trade_hub", Name: "Trade Hub", Tag: "markets",
		Desc:    "Caravans of every sphere meet here: merchants pay more and charge less.",
		Counter: "",
		Kinds:   []string{"hub", "trade", "market", "port", "station", "crossroad"}},
	{ID: "plague_world", Name: "Plague World", Tag: "plague", Harsh: true,
		Desc:    "Endemic sickness in the air and water; the unprotected fall ill.",
		Counter: "a mask, respirator, vaccine or antidote tonic",
		Kinds:   []string{"plague", "sick", "pest", "blight", "disease", "rot"}},
}

var pltTraitByID = func() map[string]*pltTraitDef {
	m := map[string]*pltTraitDef{}
	for i := range pltTraitDefs {
		m[pltTraitDefs[i].ID] = &pltTraitDefs[i]
	}
	return m
}()

// pltCounterTokens: an item counters a trait when its name, effect, category or kind contains one of
// these tokens (or its Extra holds the key). Worn and carried items both count.
var pltCounterTokens = map[string][]string{
	"high_gravity": {"exo", "power frame", "powerframe", "grav", "anti-grav", "antigrav", "levitat", "feather"},
	"aether_rich":  {"ground", "ward", "dampen", "null", "aegis", "leaden", "anti-magic", "antimagic"},
	"night_day_halves": {"lantern", "torch", "night", "darkvision", "lamp", "goggles", "visor", "veil", "shade",
		"glare", "sun"},
	"ocean_world":  {"boat", "raft", "canoe", "skiff", "kayak", "waterproof", "oilskin", "wetsuit", "diving", "waders"},
	"storm_world":  {"lightning rod", "rod", "insulat", "rubber", "grounding", "faraday", "storm-ward", "stormward"},
	"fungal_bloom": {"mask", "respirator", "filter", "rebreather", "gasmask", "gas mask", "spore", "mycel", "hood"},
	"frozen":       {"fur", "parka", "thermal", "heater", "warm", "wool", "winter", "cold-weather", "coat", "mantle", "snow", "insulated"},
	"plague_world": {"mask", "respirator", "filter", "vaccine", "antidote", "antitoxin", "plague", "physician", "remedy", "salve"},
}

// pltCounterKinds: item kinds that count outright (Item.Kind).
var pltCounterKinds = map[string][]string{
	"night_day_halves": {"light"},
}

// pltCounterMeals: meal tags (Actor.Meal) that count while the meal lasts.
var pltCounterMeals = map[string][]string{
	"frozen":       {"ember", "thick", "hearty"},
	"plague_world": {"mending"},
}

// pltScanTokens: gear that powers a survey scan (one charge, or one use of a stack).
var pltScanTokens = []string{"survey", "sensor", "scanner", "telescope", "spyglass", "astrolabe", "orrery",
	"signal", "lens", "star chart", "starchart", "sextant", "divin", "crystal ball", "seeing stone"}

// pltScanFallback: with no survey gear you may burn scrap (this many pieces) into a crude antenna.
const pltScanScrapCost = 3

// pltScanMinutes is the time a scan takes; pltScanGoldFallback never applies (time and items only).
const pltScanMinutes = 90
