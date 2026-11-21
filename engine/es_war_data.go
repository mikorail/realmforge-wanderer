package engine

// Word banks for es_war.go (prefix ew): what a battlefield, a siege camp, a ruined suburb, a last stand,
// a disaster site, a plague pit and a famine road leave behind, by era. Eras: old (swords and bows),
// powder (steam and diesel), atom, tech (cyber, bio, solar) and ruin (wasteland, twilight).

func ewEra(genre string) string {
	switch genre {
	case "steampunk", "dieselpunk":
		return "powder"
	case "atompunk":
		return "atom"
	case "cyberpunk", "biopunk", "solarpunk":
		return "tech"
	case "wasteland", "twilight":
		return "ruin"
	}
	return "old"
}

// field litter that is not a culture's weapon
var ewDebris = map[string][]string{
	"old":    {"arrowheads trodden into the turf", "a split shield rim", "a dented helmet, the strap cut", "a mail shirt torn open at the side", "a broken bowstave", "a horse's skull with the bridle still on"},
	"powder": {"spent cartridge cases, green with age", "a burst powder flask", "a shattered rifle stock", "a dented brass gorget", "canister shot sunk in the mud", "a horse's skull with the bridle still on"},
	"atom":   {"spent power cells, leaking", "a ray rifle with a cracked focus lens", "a crumpled radiation cape", "a dented bubble helmet", "a scorched field radio", "shell splinters in the ground"},
	"tech":   {"spent smart-rounds, still tagged", "a cracked visor, the display dead", "a slagged drone rotor", "burnt-out armour plates", "a severed comms cable", "shell splinters in the ground"},
	"ruin":   {"hand-loaded shell cases", "a rebar spear, bent double", "scrap-plate armour, holed", "a gas mask with the filter torn out", "a broken crossbow made from car springs", "a dog's collar with spikes"},
}

var ewEngine = map[string][]string{
	"old":    {"the charred frame of a trebuchet", "the iron head of a ram, its housing burnt away", "a mangonel's broken throwing arm"},
	"powder": {"a burst siege mortar", "a cannon off its carriage, the wheel smashed", "the boiler of a siege engine, split open"},
	"atom":   {"a scorched rocket rail", "a siege cannon with its barrel peeled open", "the shell of a mobile reactor, cold"},
	"tech":   {"a gutted siege walker", "a railgun mount, the coils slagged", "a breaching drone, crashed and stripped"},
	"ruin":   {"a burnt-out battering truck", "a junk catapult made of a crane arm", "a welded ram on a bus chassis"},
}

var ewMissile = map[string]string{"old": "arrowheads, a fistful, pulled from the stones", "powder": "flattened musket balls, prised from the stones", "atom": "melted slugs, fused to the stones", "tech": "spent flechettes, picked from the stones", "ruin": "bolts and nails, pulled from the stones"}

var ewCamp = map[string][]string{
	"old":    {"tent pegs in a broken ring", "a cook-pot holed by a stone", "a whetstone worn to a sliver", "a gnawed ox bone by a cold fire"},
	"powder": {"tent pegs in a broken ring", "a dented coffee kettle", "empty ration tins", "a cracked pipe and a twist of tobacco"},
	"atom":   {"empty ration tins", "a dented coffee flask", "a field cot, rusted", "a torn propaganda poster"},
	"tech":   {"empty ration pouches", "a dead heater cell", "a cracked tablet, the battery swollen", "a folding cot, snapped"},
	"ruin":   {"empty cans, licked clean", "a cook-pot made from a hubcap", "a scrap of tarp, burnt along one edge", "a gnawed bone by a cold fire"},
}

var ewScav = []string{"crow feathers among gnawed bones", "slit packs and empty pouches: looters came after", "a fox den dug in among the ribs", "teeth marks on a thigh bone"}

var ewToy = map[string][]string{
	"old":    {"a carved wooden horse", "a rag doll with button eyes", "a clay whistle shaped like a bird"},
	"powder": {"a tin soldier, the paint gone", "a rag doll with button eyes", "a wooden spinning top"},
	"atom":   {"a tin rocket with a wind-up key", "a rag doll with button eyes", "a plastic ray gun"},
	"tech":   {"a toy drone with a broken rotor", "a soft toy, its eyes still glowing faintly", "a puzzle cube, half solved"},
	"ruin":   {"a doll made of wire and rags", "a toy car with no wheels", "a bottle-cap necklace"},
}

var ewCart = map[string]string{"old": "an abandoned handcart, its shafts in the dirt", "powder": "a steam barrow, its boiler cold", "atom": "a scooter cart, out of fuel", "tech": "a stalled cargo trike, battery dead", "ruin": "a shopping trolley piled with rags"}

var ewRoll = map[string]string{"old": "muster roll", "powder": "casualty list", "atom": "casualty list", "tech": "unit roster", "ruin": "list of names"}

// remains by disaster kind (items only: the hazards marks are hazards_zone.go's)
var ewDisRemains = map[string][]string{
	"quake":     {"a body under a fallen lintel", "a small body wrapped in a blanket", "a row of the dead under sacking"},
	"tsunami":   {"a body caught in the wrack, wrapped later in sailcloth", "a drowned ox, bloated", "a row of the dead under sailcloth"},
	"flood":     {"a body caught in the reeds, wrapped later in sailcloth", "a drowned ox, bloated", "a row of the dead under sacking"},
	"storm":     {"a body pinned under a roof beam", "a drowned sailor in oilskins", "a row of the dead under sailcloth"},
	"tornado":   {"a body thrown against a wall", "a dead dog in a tree fork", "a row of the dead under sacking"},
	"eruption":  {"a figure curled up in grey ash", "a mule stiff under the ash", "a row of the dead under ash-grey sacking"},
	"landslide": {"a boot sticking out of the mud", "a body dug out and left on a door", "a row of the dead under sacking"},
	"avalanche": {"a frozen body dug out of the snow", "a mule frozen standing", "a row of the dead under sacking"},
	"wildfire":  {"a blackened body by a burnt fence", "the bones of goats in a burnt pen", "a row of the dead under sacking"},
	"locust":    {"a starved ox, ribs showing", "an empty seed sack", "a row of the dead under sacking"},
}

var ewDisWhat = map[string]string{"quake": "the shaking", "tsunami": "the wave", "flood": "the water", "storm": "the storm", "tornado": "the wind", "eruption": "the ash", "landslide": "the mud", "avalanche": "the snow", "wildfire": "the fire", "locust": "the hunger after the swarm"}

// fallback voice lines when the culture has no pack (cpFill placeholders allowed)
var ewVoiceBank = map[string][]string{
	"oath":      {"I swear it on my name.", "On my blood and my house, it is true.", "May I be forgotten if I lie."},
	"farewell":  {"Keep well, and keep the house.", "Do not mourn long. There is work.", "Remember me at the hearth."},
	"proverb":   {"A roof is only a roof; the hands that built it can build another.", "What the fire takes, the field gives back.", "The road is long, but it is a road."},
	"prayer":    {"May the earth lie light on them.", "Let them be warm wherever they have gone.", "Count them among the remembered."},
	"afterlife": {"The dead go home ahead of us.", "They are with their fathers now.", "They have gone on, and we follow in our time."},
	"curse":     {"May their wells go dry.", "Let their names rot.", "Curse the hand that lit the first fire."},
}

var ewOrdinal = []string{"first", "second", "third", "fourth", "fifth", "sixth", "seventh", "eighth", "ninth"}
