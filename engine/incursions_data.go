package engine

// Data for cross-genre incursions (incursions.go, prefix ci): what strangers from each genre carry,
// how they talk and what they get wrong about this world.

type ciGenre struct {
	ID      string
	Folk    string // what a person of that world is called ("a chrome-ribbed raider")
	Given   []string
	Family  []string
	Weapon  string
	WSkill  string
	Ranged  bool
	Class   string // tech, magic or wild: how the gear behaves in a world of another class
	Looks   []string
	Greets  []string
	Words   [][2]string // foreign word, meaning
	Goods   []string    // odd goods: names for strange trinkets (artifact kind)
	Goods2  []string    // what the strange trinkets do ("flames", "heal", ...)
	Customs []string    // what they get wrong here
	Creed   []string    // prophet sermon lines
	Smell   string
	Skill   string // technique they can teach
	Tech    string // the technique, in their words
	Recipe  string // a recipe title
	Thing   string // the stolen artefact
}

var ciGenres = []ciGenre{
	{ID: "fantasy", Folk: "wandering knight", Given: []string{"Aldric", "Maren", "Ysolde", "Torwyn", "Brannoch", "Elowen"}, Family: []string{"of the Silver Vale", "Oathsworn", "Ravenhelm", "of Highmere"},
		Weapon: "runed blade", WSkill: "longblade", Class: "magic",
		Looks:  []string{"a surcoat stitched with a faded heraldic beast", "a blade whose runes crawl with pale fire", "a dented helm under one arm"},
		Greets: []string{"Well met, stranger. Name your lord and your quarrel.", "By the Seven, a road at last!"},
		Words:  [][2]string{{"hearthrite", "the vow made over a shared fire"}, {"gramarye", "the old craft of binding words"}},
		Goods:  []string{"rune-lantern", "wyrm-tooth charm", "sigil-stone", "oath ring"}, Goods2: []string{"flames", "heal", "forcewall", "sense"},
		Customs: []string{"kneels and offers a sword hilt-first to a startled passer-by", "demands to know which lord holds this land and is baffled by the answer", "mistakes a market scale for a judge's altar and swears on it"},
		Creed:   []string{"The Wyrm-Moon rises; every blade must be blessed before it wakes.", "Our order fell, and the fall followed me through the sky."},
		Smell:   "cold iron and woodsmoke", Skill: "longblade", Tech: "the high guard of the Silver Vale", Recipe: "Pilgrim's honeyed oatcake", Thing: "the grail-lamp of Highmere"},
	{ID: "mythic", Folk: "hero out of legend", Given: []string{"Theron", "Kallisto", "Lykaon", "Iphis", "Damaris", "Orestes"}, Family: []string{"son of Aegeus", "the Hundred-Songed", "of the Argive host", "Oracle-touched"},
		Weapon: "sun-bronze xiphos", WSkill: "longblade", Class: "magic",
		Looks:  []string{"a laurel crown gone brown at the edges", "a bronze greave etched with a stag", "a cloak the colour of wine-dark sea"},
		Greets: []string{"Hail, mortal host. Xenia binds us both.", "The gods have flung me far, but you have a roof, I think."},
		Words:  [][2]string{{"xenia", "guest-right, sacred between host and traveller"}, {"kleos", "glory that outlives the body"}},
		Goods:  []string{"ichor vial", "laurel of Apollo", "owl-eye amulet", "thread of the Fates"}, Goods2: []string{"heal", "sense", "forcewall", "lightning"},
		Customs: []string{"pours a libation on the road and waits for a god to answer", "insists the local magistrate is a minor deity and asks for his name-day", "offers the tavern keeper a hecatomb of nine bulls"},
		Creed:   []string{"The gods have turned their faces; a hero must walk until they look back.", "Fate is a thread and someone has cut mine and tied it to your sky."},
		Smell:   "olive oil and hot stone", Skill: "missile", Tech: "the Scythian bowman's draw", Recipe: "ambrosial figs in honey", Thing: "the Thread of Ariadne"},
	{ID: "steampunk", Folk: "steam inventor", Given: []string{"Ignatius", "Philippa", "Cornelius", "Odalys", "Barnaby", "Winifred"}, Family: []string{"Cogsworth", "Brasswick", "Tinderbox", "Gearhart"},
		Weapon: "pressure-lance", WSkill: "cudgel", Class: "tech",
		Looks:  []string{"goggles pushed up over a sooty brow and a back-rig that wheezes steam", "brass fingers that click when she thinks", "a coat full of pockets, each of them ticking"},
		Greets: []string{"Splendid! A fixed point on the chart! Tell me, is there a coaling station near?", "Pardon the soot, I have been fired out of a cannon, in a manner of speaking."},
		Words:  [][2]string{{"sprocket-mad", "obsessed with a project past reason"}, {"boilerplate", "a promise nobody means to keep"}},
		Goods:  []string{"aether valve", "brass homunculus", "pocket orrery", "pressure-gauge locket"}, Goods2: []string{"lightning", "sense", "teleport", "forcewall"},
		Customs: []string{"tries to fill a fountain with coal and is astonished it will not light", "writes a patent application on a market stall's cheese-cutter", "asks which guild regulates the local aetheric weather"},
		Creed:   []string{"The Great Engine winds down; only the faithful keep a spring wound.", "Every cog must turn or the sky will stop; I have seen the sky stop."},
		Smell:   "hot oil and coal smoke", Skill: "tinkering", Tech: "the escapement trick that makes small gears obey", Recipe: "boiler-kettle stew", Thing: "the Orrery Heart"},
	{ID: "dieselpunk", Folk: "airship deserter", Given: []string{"Mack", "Dolores", "Hank", "Vesper", "Rafferty", "Lil"}, Family: []string{"Kowalczyk", "Brannigan", "Ironside", "Dunmore"},
		Weapon: "drum-fed trench gun", WSkill: "missile", Ranged: true, Class: "tech",
		Looks:  []string{"a flight jacket with a squadron patch burned half off", "oil-black knuckles and a limp from a crash", "a dented canteen and a tin badge"},
		Greets: []string{"Easy. I'm not looking for trouble, just a map and a mechanic.", "You got a radio? No? Swell. Of course you don't."},
		Words:  [][2]string{{"flak", "heavy hostile fire; trouble"}, {"dry dock", "a safe place to rest and mend"}},
		Goods:  []string{"flight compass", "ration tin", "signal flare gun", "lucky dogtag"}, Goods2: []string{"sense", "flames", "teleport", "heal"},
		Customs: []string{"shouts for a mechanic when a cart wheel sticks", "salutes the town watch and then asks for their rank, three times", "tries to sell a pack of cigarettes as medicine"},
		Creed:   []string{"The Big Dark Fleet is coming; only a full tank and clean hands will do.", "I flew into the storm and the storm had a door."},
		Smell:   "avgas and wet canvas", Skill: "missile", Tech: "the three-count burst, patient and exact", Recipe: "tin-can campfire chili", Thing: "the squadron's navigation orb"},
	{ID: "atompunk", Folk: "atomic-age scientist", Given: []string{"Dr. Reyes", "Wendell", "Marjorie", "Dr. Okafor", "Buzz", "Lucille"}, Family: []string{"Pennyworth", "Atlas", "Vandermeer", "Starling"},
		Weapon: "ray pistol", WSkill: "missile", Ranged: true, Class: "tech",
		Looks:  []string{"a white lab coat, scorched, with a radiation badge gone the colour of rust", "a chrome jetpack with the fuel gauge stuck", "horn-rimmed glasses over eyes that never stop calculating"},
		Greets: []string{"Fascinating! Zero background radiation, unusual! Are you the local committee?", "Pardon me, I appear to have been displaced across a dimensional membrane."},
		Words:  [][2]string{{"fission-bright", "full of hopeful energy"}, {"duck-and-cover", "doing the sensible thing, too late"}},
		Goods:  []string{"isotope battery", "pocket Geiger tube", "ion-lamp", "ray-gun cell"}, Goods2: []string{"lightning", "sense", "flames", "forcewall"},
		Customs: []string{"takes a reading off a campfire and announces it is unshielded", "asks the baker to sign a safety form for the oven", "tries to interview a cow for a census"},
		Creed:   []string{"The second sun is nearly kindled; join the Counting and be ready.", "All is lit by a hidden atom; I have seen it open."},
		Smell:   "ozone and hot copper", Skill: "tinkering", Tech: "the circuit-ward that tames a failing machine", Recipe: "dehydrated space-pudding", Thing: "the Reactor Key"},
	{ID: "cyberpunk", Folk: "chrome raider", Given: []string{"Kira", "Jax-9", "Nyx", "Dex", "Vee", "Rook"}, Family: []string{"Vex", "Null", "Kowloon", "Static"},
		Weapon: "monowire whip", WSkill: "shortblade", Class: "tech",
		Looks:  []string{"a glowing cheek-plate and a coil of monowire at the wrist", "neural ports ringed with scar tissue and a mirrored visor", "a jacket that flickers advertisements nobody here can read"},
		Greets: []string{"Hey, choom. No signal out here. Can you ping me a street?", "Your net's dead, huh? Whole world's offline. Wild."},
		Words:  [][2]string{{"choom", "a trusted friend"}, {"flatline", "to die; or to stop caring"}, {"jack in", "to join a thing wholeheartedly"}},
		Goods:  []string{"memory shard", "braindance chip", "optic overclock", "data-ice sliver"}, Goods2: []string{"sense", "lightning", "teleport", "forcewall"},
		Customs: []string{"waves a hand at a wall expecting an interface and mutters at the lack of one", "tries to pay with a credit chit, then hacks a market scale and breaks it", "asks which megacorp owns the local inn"},
		Creed:   []string{"The Net is a god and it is dying; come, jack the last signal.", "Flesh is a rumour. Chrome is a promise. I have seen the other side of the wall."},
		Smell:   "ozone and cheap noodles", Skill: "shortblade", Tech: "the one-breath wire-draw", Recipe: "back-alley noodle broth", Thing: "the corporate black-box ledger"},
	{ID: "biopunk", Folk: "gene-wright", Given: []string{"Sable", "Marrow", "Thessaly", "Fenn", "Calyx", "Ossian"}, Family: []string{"Grafter", "of the Vat", "Spore-born", "Ten-Lung"},
		Weapon: "thorn-spitter", WSkill: "missile", Ranged: true, Class: "wild",
		Looks:  []string{"skin lined with a faint mycelial web and a second heartbeat in the throat", "a satchel that slowly breathes", "a gait too fluid for a spine"},
		Greets: []string{"Do not breathe too hard; my spores are mostly friendly.", "Your genome is so tidy! Are you sure you are alive?"},
		Words:  [][2]string{{"graft-kin", "family chosen by shared tissue"}, {"bloom", "a quick, sudden improvement"}},
		Goods:  []string{"living lantern-pod", "mycelial map", "chrysalis seed", "second-skin patch"}, Goods2: []string{"heal", "sense", "forcewall", "phase"},
		Customs: []string{"tries to graft a market apple back onto its tree", "offers to splice a farmer's ox for a better yield, unasked", "is aghast that nobody here keeps a vat"},
		Creed:   []string{"The Great Bloom is closing; seed yourselves before the frost.", "Flesh is clay; the sky is a vat; I have seen the lid."},
		Smell:   "wet loam and sweet rot", Skill: "survival", Tech: "the grafter's wound-binding", Recipe: "fermented bloom-cap broth", Thing: "the Mother Culture"},
	{ID: "solarpunk", Folk: "sun-gardener", Given: []string{"Alder", "Iris", "Tamsin", "Soren", "Linnea", "Okoro"}, Family: []string{"Greenhand", "of the Canopy", "Sunwell", "Rootsong"},
		Weapon: "sunsteel sickle", WSkill: "axe", Class: "wild",
		Looks:  []string{"a cloak of living moss and a solar-glass pendant that hums", "bare feet and a satchel of seedlings", "a hat woven of copper wire and sweet grass"},
		Greets: []string{"Peace and good light to you. Is there a garden near?", "What a lovely brown world! Do your soils sing?"},
		Words:  [][2]string{{"canopy-kin", "all who share a common sky"}, {"gleaning", "taking only what will regrow"}},
		Goods:  []string{"sun-seed", "photon-honey jar", "rain-song flute", "solar-glass lens"}, Goods2: []string{"heal", "flames", "sense", "forcewall"},
		Customs: []string{"is distressed that nobody composts and starts a heap in the market square", "asks which tree council governs the city and meets only silence", "pays in seeds, expecting them to be planted at once"},
		Creed:   []string{"The Long Day is ending; plant, plant now while there is light.", "The sun was a great seed; I have seen it split."},
		Smell:   "sun-warmed leaves and wet soil", Skill: "survival", Tech: "the green-hand's way of reading soil", Recipe: "sun-roasted root and petal salad", Thing: "the Heart Seed"},
	{ID: "wasteland", Folk: "scavenger", Given: []string{"Rust", "Dagny", "Mad Joss", "Pike", "Corrie", "Hollis"}, Family: []string{"Dustwalker", "Scrap-King", "Two-Wheel", "Gutter"},
		Weapon: "rebar maul", WSkill: "cudgel", Class: "wild",
		Looks:  []string{"goggles, a rag mask and a coat stitched from tire rubber", "a patchwork duster hung with dented tin charms", "a geiger-clicker on a string that ticks at nothing"},
		Greets: []string{"Hold up, hold up. I ain't a raider. Got any water?", "Green grass. Real green grass. Somebody pinch me."},
		Words:  [][2]string{{"clean-water", "a gift beyond price"}, {"dust-eater", "an outsider who will not last"}},
		Goods:  []string{"pre-war can of peaches", "jury-rigged compass", "bottle-cap sack", "rad-ticker"}, Goods2: []string{"sense", "heal", "flames", "teleport"},
		Customs: []string{"drinks from a horse trough, no questions asked, and refuses to pay for it", "hoards everything in sight, including a candle stub and a doorknob", "searches a graveyard for anything useful"},
		Creed:   []string{"The Dust is coming for everyone; fill your jugs and keep the dead close.", "The sky ate the old world once. It is hungry again."},
		Smell:   "dry dust and burnt rubber", Skill: "survival", Tech: "the scavenger's trick for finding water", Recipe: "road-stew in a hubcap", Thing: "the last vault-seed crate"},
	{ID: "twilight", Folk: "vigil-bearer", Given: []string{"Morwen", "Caius", "Isolde", "Valen", "Thessaly", "Edric"}, Family: []string{"Ashgrave", "of the Hollow Hour", "Veilkeeper", "Dusk-sworn"},
		Weapon: "grave-silver stiletto", WSkill: "shortblade", Class: "magic",
		Looks:  []string{"a pale face and a lantern that burns without oil", "a coat the grey of old ash and eyes that do not quite reflect", "a rosary of small bones"},
		Greets: []string{"Do not fear the dark, friend; I keep it company.", "The veil is thin where I fell. Say nothing of what you see on my shoulder."},
		Words:  [][2]string{{"vigil", "a watch kept against the dark, often for the dead"}, {"hollow hour", "the moment between night and morning"}},
		Goods:  []string{"grave-silver charm", "ash-lantern", "mourning veil", "bone-white candle"}, Goods2: []string{"sense", "flames", "phase", "forcewall"},
		Customs: []string{"is unnerved at sunrise and keeps a hand on a rail until noon", "tries to quiet the dead in a market square that has none", "stops at every doorstep and waits to be invited"},
		Creed:   []string{"The Hollow Hour is lengthening; keep a lamp and speak the names of the lost.", "I have seen the world's last candle; it was lit from the other side."},
		Smell:   "cold candle wax and old stone", Skill: "lore", Tech: "the names that quiet restless things", Recipe: "mourners' black bread", Thing: "the Last Candle"},
}

var ciGenreIdx = func() map[string]int {
	m := map[string]int{}
	for i, g := range ciGenres {
		m[g.ID] = i
	}
	return m
}()

var ciRoles = []string{"invader", "refugee", "explorer", "merchant", "hunter", "prophet"}

var ciRoleLabel = map[string]string{"invader": "invader", "refugee": "refugee", "explorer": "lost explorer", "merchant": "merchant of strange goods", "hunter": "hunter", "prophet": "prophet", "thief": "thief"}

// Role-specific talk. {n} their name, {home} their origin, {w} a foreign word, {m} its meaning.
var ciRoleIntro = map[string][]string{
	"refugee":  {"\"{home} burns behind me. They are following. Please, is there a gate, a door, anything?\"", "\"I have nothing, and nothing is exactly what they will take. Help me reach the way out.\""},
	"explorer": {"\"I came through the sky by accident and have lost my way home. Do you know where the tear is?\"", "\"Fascinating place, truly, but I should like to be home by the new moon.\""},
	"merchant": {"\"Goods from {home}, friend, a little odd, but never a dull day with them. Prices? I make them up as I go.\"", "\"You will not find these on any local stall. I do not take your {cur}. Or, well, I do, on a whim.\""},
	"hunter":   {"\"I am tracking someone who crossed over. If you have seen a stranger, tell me, and I will make it worth your while.\"", "\"Nobody hides from me. Not even through the sky.\""},
	"prophet":  {"\"Do you not feel it? The sky is wearing thin. I came to warn you.\"", "\"Listen: the end is not far, and I am the only one who knows its name.\""},
	"invader":  {"\"Kneel or run. Your world is a prize and I mean to take it.\""},
	"thief":    {"\"You will not find it.\""},
}
