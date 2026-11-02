package engine

// Weapon and armour variety (wp): data tables. Mechanics live in wpweapons.go.

// wpFamilies are the new weapon families. Index+1 is stored in Item.Extra["wpf"].
var wpFamilies = []string{
	"spear", "flail", "whip", "staff", "fist", "hook", "great",
	"javelin", "taxe", "tknife", "chakram", "boomerang", "bolas",
	"sling", "blowgun", "crossbow", "longbow", "atlatl",
	"pistol", "longgun", "repeater",
	"shield",
}

// wpGenres are every genre a world can have (fantasy is the base skin).
var wpGenres = []string{"fantasy", "mythic", "steampunk", "dieselpunk", "atompunk", "cyberpunk", "biopunk", "solarpunk", "wasteland", "twilight"}

// wpNouns: genre → family → {plain noun (tier 0-2), fine noun (tier 3+)}.
var wpNouns = map[string]map[string][2]string{
	"fantasy": {
		"spear": {"spear", "halberd"}, "flail": {"flail", "morningstar"}, "whip": {"bullwhip", "scourge"}, "staff": {"quarterstaff", "warstaff"},
		"fist": {"cestus", "war claws"}, "hook": {"war sickle", "hooked sword"}, "great": {"greatsword", "zweihander"},
		"javelin": {"javelin", "barbed javelin"}, "taxe": {"throwing axe", "francisca"}, "tknife": {"throwing knife", "balanced throwing knife"},
		"chakram": {"war ring", "chakram"}, "boomerang": {"boomerang", "war boomerang"}, "bolas": {"bolas", "weighted bolas"},
		"sling": {"sling", "staff sling"}, "blowgun": {"blowgun", "reed blowpipe"}, "crossbow": {"crossbow", "arbalest"},
		"longbow": {"longbow", "yew warbow"}, "atlatl": {"spear-thrower", "carved atlatl"},
		"pistol": {"dragon pistol", "wheellock pistol"}, "longgun": {"arquebus", "hand cannon"}, "repeater": {"pepperbox", "organ gun"},
		"shield": {"shield", "heraldic shield"},
	},
	"mythic": {
		"spear": {"dory", "sarissa"}, "flail": {"chain-flail", "Typhon's flail"}, "whip": {"serpent lash", "Fury's scourge"}, "staff": {"shepherd's staff", "thyrsus"},
		"fist": {"caestus", "myrmex"}, "hook": {"sickle", "harpe"}, "great": {"rhomphaia", "Titan-blade"},
		"javelin": {"akontion", "Achillean javelin"}, "taxe": {"sagaris", "throwing labrys"}, "tknife": {"throwing knife", "Hermes' dart"},
		"chakram": {"bronze discus", "war discus"}, "boomerang": {"lagobolon", "curved stick of Artemis"}, "bolas": {"hunting bolas", "Atalanta's bolas"},
		"sling": {"sling", "Rhodian sling"}, "blowgun": {"reed pipe", "pipe of Pan"}, "crossbow": {"gastraphetes", "oxybeles"},
		"longbow": {"Scythian bow", "bow of Herakles"}, "atlatl": {"amentum", "ankyle-thrower"},
		"pistol": {"fire-tube", "Hephaestean fire-tube"}, "longgun": {"thunder-reed", "Cyclops thunder-reed"}, "repeater": {"bronze spitfire", "Talos repeater"},
		"shield": {"aspis", "hoplon"},
	},
	"steampunk": {
		"spear": {"boarding pike", "steam-lance"}, "flail": {"chain-flail", "piston flail"}, "whip": {"drover's whip", "galvanic whip"}, "staff": {"brass-shod cane", "swordcane"},
		"fist": {"knuckleduster", "piston gauntlet"}, "hook": {"billhook", "boarding hook"}, "great": {"claymore", "steam-saw blade"},
		"javelin": {"harpoon", "rocket harpoon"}, "taxe": {"tomahawk", "cog-axe"}, "tknife": {"throwing knife", "spring dart"},
		"chakram": {"cog-disc", "buzz-disc"}, "boomerang": {"gyro-boomerang", "clockwork boomerang"}, "bolas": {"bolas", "chain bolas"},
		"sling": {"catapult sling", "spring sling"}, "blowgun": {"dart-pipe", "pneumatic dart-pipe"}, "crossbow": {"spring crossbow", "repeating crossbow"},
		"longbow": {"longbow", "composite bow"}, "atlatl": {"spear-thrower", "spring-thrower"},
		"pistol": {"flintlock pistol", "percussion pistol"}, "longgun": {"rifled musket", "Minie rifle"}, "repeater": {"pepperbox revolver", "Gatling crank-gun"},
		"shield": {"boiler-plate shield", "riveted shield"},
	},
	"dieselpunk": {
		"spear": {"bayonet-pike", "cavalry lance"}, "flail": {"chain flail", "grapple flail"}, "whip": {"drover's whip", "cable lash"}, "staff": {"steel staff", "piston staff"},
		"fist": {"brass knuckles", "trench knuckles"}, "hook": {"cargo hook", "grappling hook"}, "great": {"two-handed cleaver", "chain-saw blade"},
		"javelin": {"harpoon", "spear-gun bolt"}, "taxe": {"hatchet", "tomahawk"}, "tknife": {"throwing knife", "trench dart"},
		"chakram": {"saw-disc", "buzz disc"}, "boomerang": {"boomerang", "steel boomerang"}, "bolas": {"bolas", "chain bolas"},
		"sling": {"slingshot", "wrist-rocket"}, "blowgun": {"dart tube", "gas-dart tube"}, "crossbow": {"crossbow", "compound crossbow"},
		"longbow": {"hunting bow", "recurve bow"}, "atlatl": {"spear thrower", "harpoon launcher"},
		"pistol": {"revolver", "service automatic"}, "longgun": {"bolt-action rifle", "scoped rifle"}, "repeater": {"submachine gun", "drum-fed tommy gun"},
		"shield": {"trench shield", "armoured shield"},
	},
	"atompunk": {
		"spear": {"atomic lance", "ray-tipped pike"}, "flail": {"magno-flail", "gyro-flail"}, "whip": {"shock whip", "arc lash"}, "staff": {"aluminium staff", "ion staff"},
		"fist": {"atomic knuckles", "power fist"}, "hook": {"hook-hand", "crane hook"}, "great": {"atomic cleaver", "ray-edged greatblade"},
		"javelin": {"rocket spear", "atomic javelin"}, "taxe": {"chrome tomahawk", "gyro-axe"}, "tknife": {"throwing star", "ray dart"},
		"chakram": {"saucer disc", "buzzing saucer"}, "boomerang": {"ray-boomerang", "returning saucer"}, "bolas": {"magnet bolas", "ion bolas"},
		"sling": {"slingshot", "ray-slingshot"}, "blowgun": {"dart tube", "needle gun"}, "crossbow": {"bolt-gun", "magnetic crossbow"},
		"longbow": {"compound bow", "chrome bow"}, "atlatl": {"dart launcher", "gyro-launcher"},
		"pistol": {"atomic revolver", "ray revolver"}, "longgun": {"lever-action rifle", "gamma rifle"}, "repeater": {"burp gun", "plasma repeater"},
		"shield": {"chrome shield", "deflector shield"},
	},
	"cyberpunk": {
		"spear": {"stun lance", "mono-spear"}, "flail": {"chain-flail", "monowire flail"}, "whip": {"shock whip", "monowire"}, "staff": {"telescopic staff", "carbon bo"},
		"fist": {"knuckle rig", "gorilla fists"}, "hook": {"kama", "mantis hook"}, "great": {"chainblade", "thermal greatsword"},
		"javelin": {"smart javelin", "plasma dart"}, "taxe": {"tomahawk", "smart hatchet"}, "tknife": {"shuriken", "smart dart"},
		"chakram": {"smart disc", "hardlight disc"}, "boomerang": {"drone boomerang", "homing disc"}, "bolas": {"taser bolas", "net bolas"},
		"sling": {"slingshot", "mag-sling"}, "blowgun": {"needler", "toxin needler"}, "crossbow": {"carbon crossbow", "smart crossbow"},
		"longbow": {"compound bow", "smart bow"}, "atlatl": {"dart launcher", "smart launcher"},
		"pistol": {"smartpistol", "heavy pistol"}, "longgun": {"tech rifle", "rail rifle"}, "repeater": {"smart SMG", "power SMG"},
		"shield": {"riot shield", "ballistic shield"},
	},
	"biopunk": {
		"spear": {"bone pike", "stinger lance"}, "flail": {"seed-pod flail", "tendril flail"}, "whip": {"tendril whip", "nettle lash"}, "staff": {"rootwood staff", "living staff"},
		"fist": {"bone knuckles", "grown claws"}, "hook": {"hooked claw", "sickle talon"}, "great": {"ribcage greatblade", "mantis greatblade"},
		"javelin": {"spine javelin", "barbed quill"}, "taxe": {"chitin hatchet", "beak axe"}, "tknife": {"quill dart", "venom spine"},
		"chakram": {"shell disc", "spiral shell"}, "boomerang": {"wing-bone boomerang", "homing seed"}, "bolas": {"vine bolas", "sticky bolas"},
		"sling": {"tendon sling", "spore sling"}, "blowgun": {"quill pipe", "venom pipe"}, "crossbow": {"tendon crossbow", "spine crossbow"},
		"longbow": {"sinew bow", "living bow"}, "atlatl": {"gland-thrower", "spine-thrower"},
		"pistol": {"spore-launcher", "acid spitter"}, "longgun": {"acid-pod rifle", "hive rifle"}, "repeater": {"hive repeater", "swarm gun"},
		"shield": {"shell shield", "carapace shield"},
	},
	"solarpunk": {
		"spear": {"bamboo spear", "solar lance"}, "flail": {"chain-flail", "kinetic flail"}, "whip": {"hemp whip", "vine whip"}, "staff": {"bamboo staff", "carbon quarterstaff"},
		"fist": {"garden claws", "kinetic knuckles"}, "hook": {"pruning hook", "harvest sickle"}, "great": {"clearing blade", "carbon greatsword"},
		"javelin": {"bamboo javelin", "coil javelin"}, "taxe": {"hatchet", "throwing adze"}, "tknife": {"seed dart", "throwing knife"},
		"chakram": {"solar disc", "kinetic disc"}, "boomerang": {"boomerang", "glide-wing"}, "bolas": {"hemp bolas", "kelp bolas"},
		"sling": {"sling", "elastic sling"}, "blowgun": {"bamboo blowgun", "seed blowgun"}, "crossbow": {"bamboo crossbow", "compound crossbow"},
		"longbow": {"longbow", "laminated bow"}, "atlatl": {"atlatl", "carbon atlatl"},
		"pistol": {"coil pistol", "solar pistol"}, "longgun": {"coil rifle", "rail rifle"}, "repeater": {"pulse repeater", "coil carbine"},
		"shield": {"wicker shield", "sun-panel shield"},
	},
	"wasteland": {
		"spear": {"rebar spear", "sign-post pike"}, "flail": {"chain flail", "wrecking-ball flail"}, "whip": {"cable whip", "barbed-wire whip"}, "staff": {"lead pipe", "pipe staff"},
		"fist": {"knuckle spikes", "scrap power fist"}, "hook": {"meat hook", "tow hook"}, "great": {"road-sign greatblade", "chainsaw"},
		"javelin": {"rebar javelin", "spike javelin"}, "taxe": {"tomahawk", "hubcap axe"}, "tknife": {"throwing spike", "rail spike"},
		"chakram": {"hubcap", "saw blade"}, "boomerang": {"boomerang", "bent rebar"}, "bolas": {"chain bolas", "tire bolas"},
		"sling": {"slingshot", "wrist-rocket"}, "blowgun": {"pipe blowgun", "nail-dart tube"}, "crossbow": {"scrap crossbow", "rebar crossbow"},
		"longbow": {"scav bow", "compound bow"}, "atlatl": {"spear-thrower", "pipe launcher"},
		"pistol": {"zip gun", "pipe pistol"}, "longgun": {"pipe rifle", "scrap musket"}, "repeater": {"scrap SMG", "nail-gun"},
		"shield": {"car-door shield", "stop-sign shield"},
	},
	"twilight": {
		"spear": {"hunter's spear", "saw-spear"}, "flail": {"censer flail", "chained reliquary"}, "whip": {"penitent's scourge", "thorned lash"}, "staff": {"pilgrim's staff", "gravedigger's staff"},
		"fist": {"iron claws", "beast claws"}, "hook": {"hunter's sickle", "reaper's hook"}, "great": {"executioner's sword", "cathedral greatblade"},
		"javelin": {"silver javelin", "saint's javelin"}, "taxe": {"throwing hatchet", "hunter's axe"}, "tknife": {"throwing knife", "silver knife"},
		"chakram": {"chakram", "moon wheel"}, "boomerang": {"boomerang", "raven-wing"}, "bolas": {"bolas", "silver bolas"},
		"sling": {"sling", "chapel sling"}, "blowgun": {"blowpipe", "plague pipe"}, "crossbow": {"hand crossbow", "heavy arbalest"},
		"longbow": {"longbow", "hunter's bow"}, "atlatl": {"spear-thrower", "bone thrower"},
		"pistol": {"wheellock pistol", "hunter's pistol"}, "longgun": {"matchlock musket", "quicksilver rifle"}, "repeater": {"blunderbuss", "volley gun"},
		"shield": {"tower-chapel shield", "gilded shield"},
	},
}

// wpGun is a genre's firearm profile; index 0 pistol, 1 long gun, 2 repeater.
type wpGun struct {
	Tag, Ammo string
	Mag       [3]int
	RL        [3]int // reload energy (1000 = one turn)
	Burst     [3]int
	Jam       int // percent per pull
	Elem      string
	Smart     bool // +3 to hit
	PV        int  // extra penetration
}

var wpGuns = map[string]wpGun{
	"fantasy":    {Tag: "powder", Ammo: "powder and shot", Mag: [3]int{1, 1, 4}, RL: [3]int{2000, 2500, 3000}, Burst: [3]int{1, 1, 1}, Jam: 6, PV: 1},
	"mythic":     {Tag: "powder", Ammo: "thunder-seeds", Mag: [3]int{1, 1, 3}, RL: [3]int{2000, 2500, 3000}, Burst: [3]int{1, 1, 1}, Jam: 5, Elem: "fire"},
	"steampunk":  {Tag: "powder", Ammo: "paper cartridges", Mag: [3]int{1, 1, 6}, RL: [3]int{1800, 2200, 2500}, Burst: [3]int{1, 1, 1}, Jam: 4, PV: 1},
	"dieselpunk": {Tag: "round", Ammo: "rounds", Mag: [3]int{6, 5, 20}, RL: [3]int{1200, 1500, 1500}, Burst: [3]int{1, 1, 3}, PV: 2},
	"atompunk":   {Tag: "round", Ammo: "atomic rounds", Mag: [3]int{6, 8, 24}, RL: [3]int{1200, 1500, 1500}, Burst: [3]int{1, 1, 3}, PV: 2},
	"cyberpunk":  {Tag: "cell", Ammo: "caseless magazines", Mag: [3]int{12, 4, 30}, RL: [3]int{1000, 1200, 1200}, Burst: [3]int{1, 1, 3}, Elem: "shock", Smart: true, PV: 3},
	"biopunk":    {Tag: "pod", Ammo: "spore pods", Mag: [3]int{3, 4, 12}, RL: [3]int{1200, 1500, 1500}, Burst: [3]int{1, 1, 3}, Elem: "poison", PV: 1},
	"solarpunk":  {Tag: "cell", Ammo: "charge cells", Mag: [3]int{8, 6, 16}, RL: [3]int{1000, 1200, 1200}, Burst: [3]int{1, 1, 2}, Elem: "shock", PV: 2},
	"wasteland":  {Tag: "scrap", Ammo: "pipe shells", Mag: [3]int{1, 2, 12}, RL: [3]int{1500, 1800, 1500}, Burst: [3]int{1, 1, 3}, Jam: 12, PV: 1},
	"twilight":   {Tag: "powder", Ammo: "quicksilver shot", Mag: [3]int{1, 1, 3}, RL: [3]int{1800, 2500, 3000}, Burst: [3]int{1, 1, 1}, Jam: 5, PV: 1},
}

// wpAmmoNames: ammo tag → name (guns use their genre profile name).
var wpAmmoNames = map[string]string{"arrow": "arrows", "bolt": "crossbow bolts", "stone": "sling stones", "dart": "darts"}

// wpArmorType is one kind of armour. AV grows by tier/2.
type wpArmorType struct {
	Slot   string
	AV, DV int
	W      int
	Name   string // generic name
	Genres []string
	Names  map[string]string // genre → name
	STR    int
}

var wpArmorTypes = map[string]*wpArmorType{
	"gambeson":   {Slot: "body", AV: 2, W: 5, Name: "padded gambeson"},
	"quilted":    {Slot: "body", AV: 2, W: 4, Name: "quilted cotton armour"},
	"leather":    {Slot: "body", AV: 2, W: 5, Name: "boiled leather armour"},
	"hide":       {Slot: "body", AV: 2, W: 6, Name: "hide armour"},
	"slat":       {Slot: "body", AV: 3, W: 7, Name: "slat armour"},
	"chitin":     {Slot: "body", AV: 3, W: 4, Name: "chitin plates", Genres: []string{"fantasy", "biopunk", "twilight", "wasteland"}},
	"mail":       {Slot: "body", AV: 4, DV: -1, W: 10, Name: "mail hauberk"},
	"scale":      {Slot: "body", AV: 4, DV: -1, W: 11, Name: "scale coat"},
	"lamellar":   {Slot: "body", AV: 4, W: 9, Name: "lamellar coat"},
	"brigandine": {Slot: "body", AV: 4, DV: -1, W: 9, Name: "brigandine"},
	"cuirass":    {Slot: "body", AV: 5, DV: -1, W: 12, Name: "cuirass"},
	"oyoroi":     {Slot: "body", AV: 5, DV: -1, W: 12, Name: "great lamellar harness"},
	"plate":      {Slot: "body", AV: 6, DV: -2, W: 16, Name: "plate harness"},
	"kevlar": {Slot: "body", AV: 4, W: 6, Name: "ballistic vest", Genres: []string{"dieselpunk", "atompunk", "cyberpunk", "biopunk", "solarpunk", "wasteland"},
		Names: map[string]string{"dieselpunk": "flak jacket", "atompunk": "ballistic vest", "cyberpunk": "ballistic weave vest", "biopunk": "keratin weave vest",
			"solarpunk": "ballistic hemp vest", "wasteland": "patched kevlar vest"}},
	"powerframe": {Slot: "body", AV: 8, DV: -2, W: 20, Name: "power armour frame", STR: 2, Genres: []string{"steampunk", "dieselpunk", "atompunk", "cyberpunk", "biopunk", "solarpunk", "wasteland"},
		Names: map[string]string{"steampunk": "clockwork exo-harness", "dieselpunk": "diesel walker harness", "atompunk": "power armour frame", "cyberpunk": "exoskeleton frame",
			"biopunk": "muscle-graft frame", "solarpunk": "solar exo-frame", "wasteland": "salvaged power frame"}},
}

// wpBallistic: armour types that shrug off bullets and bolts (shoot PV penalty).
var wpBallistic = map[string]int{"kevlar": 3, "powerframe": 2, "plate": 1}

// wpShieldType is an off-hand shield class.
type wpShieldType struct {
	Block, AV, DV, W int
	Full             bool // blocks missiles as well as blows
	Name             string
	Genres           []string
}

var wpShieldTypes = map[string]*wpShieldType{
	"buckler": {Block: 15, DV: 1, W: 2, Name: "buckler"},
	"round":   {Block: 25, AV: 1, W: 5, Name: "round shield"},
	"kite":    {Block: 30, AV: 1, W: 7, Name: "kite shield"},
	"tower":   {Block: 40, AV: 2, DV: -1, W: 12, Full: true, Name: "tower shield"},
	"pavise":  {Block: 45, AV: 2, DV: -2, W: 14, Full: true, Name: "pavise"},
	"riot":    {Block: 35, AV: 2, DV: -1, W: 8, Full: true, Name: "riot shield", Genres: []string{"dieselpunk", "atompunk", "cyberpunk", "solarpunk", "wasteland"}},
	"energy":  {Block: 40, AV: 1, W: 1, Full: true, Name: "energy shield", Genres: []string{"atompunk", "cyberpunk", "solarpunk", "steampunk"}},
}

var wpShieldOrder = []string{"buckler", "round", "kite", "tower", "pavise", "riot", "energy"}

// wpGear is one sphere weapon, armour or shield.
type wpGear struct {
	Name, Fam, Desc string // Fam: a legacy skill (longblade...) or a wpFamilies id; for armour the armour type, for shields the shield type
}

type wpSphereKit struct {
	Weapons []wpGear
	Armor   []wpGear
	Shield  *wpGear
}

// wpSpheres: the weapons, armour and shields of each culture sphere (keys match attireTable).
var wpSpheres = map[string]*wpSphereKit{
	"sinic": {Weapons: []wpGear{
		{"jian", "longblade", "A straight double-edged sword, called the gentleman of weapons."},
		{"dao", "longblade", "A broad single-edged sabre, the soldier's blade."},
		{"guandao", "spear", "A heavy crescent blade on a long pole: a polearm that reaches two paces."},
		{"qiang", "spear", "A red-tasselled spear, the king of weapons."},
		{"meteor hammer", "flail", "Two weights on a long rope, swung to wrap around guards."},
		{"chu-ko-nu", "crossbow", "A repeating crossbow fed from a box of bolts."},
	}, Armor: []wpGear{{"lamellar coat", "lamellar", "Laced iron scales in overlapping rows."}, {"padded brigandine", "brigandine", "Cloth studded over hidden plates."}},
		Shield: &wpGear{"rattan shield", "round", "Woven rattan, light and springy."}},
	"japonic": {Weapons: []wpGear{
		{"katana", "longblade", "A curved, single-edged blade folded many times."},
		{"wakizashi", "shortblade", "The short companion sword, worn at the hip."},
		{"naginata", "spear", "A curved blade on a long shaft that sweeps from two paces."},
		{"yari", "spear", "A straight spear for the front rank."},
		{"kusarigama", "hook", "A sickle on a weighted chain: it hooks legs and trips."},
		{"tetsubo", "great", "A studded iron club that needs both hands."},
		{"shuriken", "tknife", "Small throwing blades, hidden in the sleeve."},
		{"yumi", "longbow", "A tall asymmetric bow of laminated bamboo."},
	}, Armor: []wpGear{{"o-yoroi", "oyoroi", "Box-like lamellar of lacquered plates, laced in silk."}, {"do-maru", "lamellar", "A wrapped lamellar cuirass for foot soldiers."}},
		Shield: &wpGear{"tate", "pavise", "A standing wooden shield planted before archers."}},
	"indic": {Weapons: []wpGear{
		{"talwar", "longblade", "A curved sabre with a disc-pommelled hilt."},
		{"katar", "fist", "A push-dagger gripped in the fist: fast, punching blows."},
		{"urumi", "whip", "A flexible sword of steel ribbons that lashes from two paces."},
		{"khanda", "longblade", "A broad straight sword with a spiked hilt."},
		{"chakram", "chakram", "A sharpened steel ring that spins out and may come back."},
		{"gada", "cudgel", "A heavy mace with a round head on a long haft."},
		{"bagh nakh", "fist", "Tiger claws hidden in the palm."},
	}, Armor: []wpGear{{"coat of a thousand nails", "brigandine", "Padded cloth studded with nails and plates."}, {"zirah bagtar", "mail", "Mail with plates set into the chest."}},
		Shield: &wpGear{"dhal", "round", "A round shield of hide or steel with four bosses."}},
	"sea": {Weapons: []wpGear{
		{"kris", "shortblade", "A wavy-bladed dagger thought to hold a spirit."},
		{"kampilan", "great", "A long single-edged sword with a forked tip, swung two-handed."},
		{"parang", "longblade", "A heavy jungle blade for wood and war."},
		{"karambit", "hook", "A claw-curved knife that hooks and trips."},
		{"sumpit", "blowgun", "A long blowpipe that spits poisoned darts."},
		{"tombak", "spear", "A spear with a long leaf-shaped head."},
	}, Armor: []wpGear{{"rattan cuirass", "slat", "Rattan and horn laced in rows."}, {"brass-plated mail", "mail", "Mail with brass plates over the chest."}},
		Shield: &wpGear{"kalasag", "tower", "A tall narrow wooden shield, carved and painted."}},
	"steppe": {Weapons: []wpGear{
		{"steppe sabre", "longblade", "A gently curved cavalry blade."},
		{"horse lance", "spear", "A long lance for the charge."},
		{"shestopyor", "cudgel", "A flanged mace with six blades."},
		{"nagaika", "whip", "A short heavy riding whip with a weighted tip."},
		{"horn bow", "longbow", "A recurved composite bow of horn, wood and sinew."},
		{"throwing lasso-weights", "bolas", "Corded weights thrown at legs."},
	}, Armor: []wpGear{{"felt lamellar", "lamellar", "Iron lamellae over thick felt."}, {"hide coat", "hide", "Layered hide, light enough to ride in."}},
		Shield: &wpGear{"wicker shield", "round", "Willow wicker bound with leather."}},
	"persian": {Weapons: []wpGear{
		{"shamshir", "longblade", "A deeply curved slashing sabre."},
		{"tabarzin", "axe", "A crescent saddle-axe."},
		{"pesh-kabz", "shortblade", "A T-spined dagger that punches through mail."},
		{"gorz", "cudgel", "An ox-headed mace of heroes."},
		{"zupin", "javelin", "A short throwing spear."},
	}, Armor: []wpGear{{"zereh mail", "mail", "Fine riveted mail."}, {"chahar aina", "plate", "Four mirrors: polished plates over mail."}},
		Shield: &wpGear{"sipar", "round", "A round cane shield wrapped in silk."}},
	"arabic": {Weapons: []wpGear{
		{"scimitar", "longblade", "A curved desert blade."},
		{"kilij", "longblade", "A sabre that widens toward a heavy tip."},
		{"khopesh", "hook", "A sickle-sword whose crook pulls shields and ankles."},
		{"jambiya", "shortblade", "A curved dagger worn at the belt."},
		{"rumh", "spear", "A long cavalry lance of cane."},
	}, Armor: []wpGear{{"dir'", "mail", "A mail shirt worn under the robe."}, {"quilted jubba", "gambeson", "A thick quilted coat."}},
		Shield: &wpGear{"daraqa", "buckler", "A small leather buckler."}},
	"mande": {Weapons: []wpGear{
		{"takoba", "longblade", "A straight double-edged sword of the Sahel."},
		{"kaskara", "longblade", "A long cross-hilted broadsword."},
		{"sahel lance", "spear", "A cavalry lance with a long iron head."},
		{"throwing iron", "tknife", "A many-bladed throwing knife that spins as it flies."},
		{"war club", "cudgel", "A hardwood club, knobbed and heavy."},
	}, Armor: []wpGear{{"lifidi", "quilted", "Quilted cotton armour for horse and rider."}, {"leather cuirass", "leather", "Thick tooled leather."}},
		Shield: &wpGear{"hide shield", "round", "A round shield of oryx hide."}},
	"yoruba": {Weapons: []wpGear{
		{"ida", "longblade", "A broad straight sword of the war chiefs."},
		{"ofangbo", "hook", "A curved hooking blade."},
		{"ada", "longblade", "A heavy cutlass-machete."},
		{"oko", "spear", "A spear with a long iron head."},
		{"ofa bow", "longbow", "A hunter's bow."},
	}, Armor: []wpGear{{"padded war coat", "gambeson", "Quilted cotton dyed indigo."}, {"leather war vest", "leather", "Hardened leather with charms sewn in."}},
		Shield: &wpGear{"hide shield", "round", "A shield of buffalo hide."}},
	"bantu": {Weapons: []wpGear{
		{"assegai", "javelin", "A light throwing spear."},
		{"iklwa", "spear", "A short stabbing spear named for the sound it makes."},
		{"knobkerrie", "cudgel", "A club with a heavy knob, thrown or swung."},
		{"induku", "staff", "A fighting stick, long and hard."},
		{"throwing knife", "tknife", "A spinning many-bladed throwing knife."},
	}, Armor: []wpGear{{"hide armour", "hide", "Layered cowhide."}},
		Shield: &wpGear{"isihlangu", "tower", "A tall oval cowhide war shield."}},
	"ethiopic": {Weapons: []wpGear{
		{"shotel", "hook", "A sickle-sword that reaches around a shield to strike."},
		{"gorade", "longblade", "A straight broadsword."},
		{"tor", "spear", "A long iron spear."},
		{"alanga", "whip", "A braided hide whip."},
	}, Armor: []wpGear{{"lion-mane cape", "hide", "A war cape of lion mane over hide."}, {"mail shirt", "mail", "Mail traded from the coast."}},
		Shield: &wpGear{"gasha", "round", "A round hide shield with silver bosses."}},
	"nahua": {Weapons: []wpGear{
		{"macuahuitl", "great", "A wooden sword edged with obsidian blades: it tears flesh."},
		{"tepoztopilli", "spear", "A broad-headed polearm edged with obsidian."},
		{"atlatl", "atlatl", "A spear-thrower that hurls darts with great force."},
		{"cuahuitl club", "cudgel", "A hardwood war club."},
		{"tematlatl", "sling", "A woven maguey sling."},
	}, Armor: []wpGear{{"ichcahuipilli", "quilted", "Quilted cotton armour soaked in brine: it stops arrows."}, {"tlahuiztli", "gambeson", "A feathered warrior suit over quilting."}},
		Shield: &wpGear{"chimalli", "round", "A round shield of reed and feathers."}},
	"maya": {Weapons: []wpGear{
		{"obsidian spear", "spear", "A thrusting spear tipped with obsidian."},
		{"hadz' club", "cudgel", "A club set with jade and flint."},
		{"obsidian dagger", "shortblade", "A sacrificial blade."},
		{"wub blowgun", "blowgun", "A blowgun for birds and enemies alike."},
		{"hul'che", "atlatl", "A dart-thrower."},
	}, Armor: []wpGear{{"quilted cotton jacket", "quilted", "Thick quilted cotton."}},
		Shield: &wpGear{"woven shield", "round", "A flexible woven shield."}},
	"andean": {Weapons: []wpGear{
		{"bolas", "bolas", "Stone weights on cords, thrown to tangle legs."},
		{"huaraca", "sling", "A braided wool sling with a long reach."},
		{"champi", "cudgel", "A star-headed bronze mace."},
		{"chuki", "spear", "A long lance."},
	}, Armor: []wpGear{{"quilted unku", "quilted", "A quilted cotton tunic."}},
		Shield: &wpGear{"hide shield", "round", "A shield of hide painted with the sun."}},
	"turtle": {Weapons: []wpGear{
		{"tomahawk", "taxe", "A light axe for close work or throwing."},
		{"ball-headed war club", "cudgel", "A club carved from one root, with a ball head."},
		{"gunstock club", "hook", "A club with a blade set in its crook."},
		{"lance", "spear", "A long lance with an eagle-feather pennant."},
		{"sinew bow", "longbow", "A sinew-backed bow."},
	}, Armor: []wpGear{{"rod armour", "slat", "Rods of hardwood laced together."}, {"elk-hide shirt", "hide", "Thick elk hide."}},
		Shield: &wpGear{"bison-hide shield", "round", "A painted bison-hide shield."}},
	"inuit": {Weapons: []wpGear{
		{"harpoon", "javelin", "A toggle-headed harpoon."},
		{"ulu", "shortblade", "A crescent knife."},
		{"snow knife", "shortblade", "A long bone knife for cutting snow blocks."},
		{"sinew-backed bow", "longbow", "A short bow backed with braided sinew."},
	}, Armor: []wpGear{{"bone-slat armour", "slat", "Slats of bone and ivory laced in rows."}}},
	"polynesian": {Weapons: []wpGear{
		{"taiaha", "staff", "A long fighting staff, struck and thrust."},
		{"mere", "cudgel", "A short flat club of greenstone."},
		{"leiomano", "hook", "A club edged with shark teeth that catches and tears."},
		{"tewhatewha", "axe", "An axe-shaped long club."},
		{"maka sling", "sling", "A plaited coconut-fibre sling."},
	}, Armor: []wpGear{{"coconut-fibre armour", "gambeson", "Thick armour knotted from coconut fibre."}}},
	"norse": {Weapons: []wpGear{
		{"seax", "shortblade", "A long single-edged knife."},
		{"bearded axe", "axe", "An axe whose lower blade hooks shields."},
		{"dane axe", "great", "A broad axe on a long haft, swung with both hands."},
		{"atgeir", "spear", "A hewing spear of the sagas."},
		{"throwing spear", "javelin", "A light spear for the first volley."},
	}, Armor: []wpGear{{"byrnie", "mail", "A mail shirt to the thigh."}, {"padded tunic", "gambeson", "A quilted wool tunic."}},
		Shield: &wpGear{"linden shield", "round", "A round shield of linden boards."}},
	"celtic": {Weapons: []wpGear{
		{"claymore", "great", "A great two-handed sword of the highlands."},
		{"sgian-dubh", "shortblade", "A small black knife kept in the stocking."},
		{"lochaber axe", "spear", "A polearm with a hooked axe blade."},
		{"sling", "sling", "A shepherd's sling."},
		{"broadsword", "longblade", "A basket-hilted broadsword."},
	}, Armor: []wpGear{{"leather jack", "leather", "A thick leather coat."}, {"ring mail", "mail", "Mail of riveted rings."}},
		Shield: &wpGear{"targe", "buckler", "A small round studded targe."}},
	"latin": {Weapons: []wpGear{
		{"gladius", "shortblade", "A short stabbing sword of the legions."},
		{"spatha", "longblade", "A long cavalry sword."},
		{"pilum", "javelin", "A heavy javelin whose shank bends on impact."},
		{"hasta", "spear", "A thrusting spear."},
		{"plumbata", "tknife", "A lead-weighted throwing dart."},
	}, Armor: []wpGear{{"lorica segmentata", "plate", "Banded iron strips over the torso."}, {"lorica hamata", "mail", "Long-sleeved mail."}, {"lorica squamata", "scale", "Overlapping bronze scales."}},
		Shield: &wpGear{"scutum", "tower", "A curved rectangular legion shield."}},
	"slavic": {Weapons: []wpGear{
		{"shashka", "longblade", "A guardless cavalry sabre."},
		{"kisten", "flail", "A weight on a chain at the end of a handle."},
		{"berdysh", "spear", "A long crescent axe on a pole."},
		{"sulitsa", "javelin", "A throwing spear."},
		{"bulava", "cudgel", "A pear-headed mace."},
		{"nagaika", "whip", "A braided Cossack whip."},
	}, Armor: []wpGear{{"kolchuga", "mail", "A mail shirt."}, {"bekhterets", "lamellar", "Small plates laced over mail."}},
		Shield: &wpGear{"kite shield", "kite", "A long almond-shaped shield."}},
	"eldritch": {Weapons: []wpGear{
		{"glass-fanged scourge", "whip", "A lash of black glass teeth that sings when it moves."},
		{"moon-sickle", "hook", "A crescent of pale metal that hooks at things not quite there."},
		{"eyeless spear", "spear", "A spear whose head watches you when you are not looking."},
		{"bone-chime flail", "flail", "Chimes of bone on a chain; foes cannot think while it rings."},
		{"whispering knife", "tknife", "It returns. It always returns."},
		{"star-pinned staff", "staff", "A staff with a cold star pinned to its head."},
	}, Armor: []wpGear{{"drowned chitin", "chitin", "Plates shed by something from below."}},
		Shield: &wpGear{"mirror-black ward", "kite", "A shield that reflects a sky that is not this one."}},
}
