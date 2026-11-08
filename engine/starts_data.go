package engine

// starts_data.go (prefix st): the scenario catalogue shown on the character-creation cards.

// stScen is one starting scenario. Kind "power" starts are picked one at a time; "twist" modifiers
// stack on top of at most one power start. The card shows Pitch (the premise) and Down (the cost).
type stScen struct {
	ID    string `json:"id"`
	Kind  string `json:"kind"` // power | twist
	Name  string `json:"name"`
	Pitch string `json:"pitch"`
	Down  string `json:"down"`
	Moves bool   `json:"moves,omitempty"` // the scenario picks the starting town itself
}

var stScens = []stScen{
	{ID: "transmig", Kind: "power", Name: "Transmigrator",
		Pitch: "You wake in a stranger's body already deep in the cultivation path: level 14, a high realm and three techniques from a sect you never joined.",
		Down:  "A stranger everywhere: no gold, the home realm distrusts you, and a rival sect's disciples hunt you."},
	{ID: "fallen", Kind: "power", Name: "Fallen God",
		Pitch: "Vast attributes and a few surviving gifts; the rest of your divinity waits behind a curse that breaks one seal at a time.",
		Down:  "Each seal you break (every 2 levels or a mighty kill) brings the doom of the world closer."},
	{ID: "heir", Kind: "power", Name: "Dynasty Heir", Moves: true,
		Pitch: "You start in the capital with a seat at court, the crown's favour and a ruler's purse.",
		Down:  "A court faction wants you dead and sends assassins; a rival realm has put a price on your head."},
	{ID: "relic", Kind: "power", Name: "Relic Bearer",
		Pitch: "You carry one of the world's famous relics, already known and already yours.",
		Down:  "Every realm's hunters and every bandit chief want it, and so want you."},
	{ID: "beast", Kind: "power", Name: "Beast-Bound",
		Pitch: "A tamed legendary beast is your companion and your mount; unsaddle it to fight beside you.",
		Down:  "You yourself are weak: lowered attributes and little health."},
	{ID: "survivor", Kind: "power", Name: "Last Survivor",
		Pitch: "The world-ending threat has already risen and holds much of the map. You start in a besieged town with a few allies.",
		Down:  "Most roads are occupied land; the end is not a matter of waiting for it."},
	{ID: "undead", Kind: "twist", Name: "Revenant",
		Pitch: "You are dead and walking: no hunger or thirst, cold and death powers, the undead let you pass.",
		Down:  "Town guards attack on sight and priests try to purge you."},
	{ID: "planet", Kind: "twist", Name: "Wrong Planet", Moves: true,
		Pitch: "You begin on another world, in a town beside a rift, as a stranger no one has heard of.",
		Down:  "No reputation, no homeland, and no road back you know."},
	{ID: "naked", Kind: "twist", Name: "Naked and Nameless", Moves: true,
		Pitch: "Experience comes twice as fast.",
		Down:  "5 HP, no gear, no powers, no gold, in a town at war."},
	{ID: "mirror", Kind: "twist", Name: "Mirror Life",
		Pitch: "Fate rolls your species, calling, attributes, powers and birthplace; you live with what comes.",
		Down:  "You cannot change the roll. Re-roll as often as you like before you begin."},
	{ID: "clock", Kind: "twist", Name: "Speedrun Clock",
		Pitch: "The world's great threat rises after 15 days instead of 60.",
		Down:  "You will meet it far weaker than you should."},
}

// stAlt renames scenarios in genres where it helps.
var stAlt = map[string]map[string]string{
	"fallen": {"mythic": "Exiled Immortal", "twilight": "Dethroned Sovereign", "cyberpunk": "Rogue Intelligence", "wasteland": "Ruined Demigod"},
	"undead": {"mythic": "Shade", "steampunk": "Galvanised Revenant", "dieselpunk": "Galvanised Revenant", "cyberpunk": "Ghost in the Shell", "biopunk": "Reanimate", "wasteland": "Rot-Walker"},
	"relic":  {"cyberpunk": "Keyholder", "wasteland": "Vault-Key Carrier"},
	"clock":  {"mythic": "Fate's Hourglass", "wasteland": "Countdown"},
}

// stList is the catalogue for a world's genre (CharOptions "scenarios").
func stList(genre string) []stScen {
	out := make([]stScen, len(stScens))
	copy(out, stScens)
	for i := range out {
		if n, ok := stAlt[out[i].ID][genre]; ok {
			out[i].Name = n
		}
	}
	return out
}

var stScenByID = func() map[string]*stScen {
	m := map[string]*stScen{}
	for i := range stScens {
		m[stScens[i].ID] = &stScens[i]
	}
	return m
}()

// stFallenPowers are the sealed gifts of the Fallen God, in unlock order; stFallenOpen start free.
var stFallenPowers = []string{"lightning", "flames", "teleport", "syphon", "timedilation", "phase", "stunforce", "fugue", "clairvoyance", "disintegrate"}
var stFallenOpen = []string{"heal", "sense", "darkvision", "regen"}

// stUndeadPowers: cold and death.
var stUndeadPowers = map[string]int{"freeze": 2, "leech": 2, "syphon": 1, "darkvision": 1}
