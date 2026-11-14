package engine

// Genre skins for the stance/technique layer (prefix cs). Every genre starts from csSkinBase and
// overrides only the names it reskins; unknown genres fall back to the base (fantasy) wording.

type csSkin struct {
	Stance  [4]string         // guard, aggressive, evasive, steady aim
	Tech    map[string]string // class -> active technique name
	Passive map[string]string // class -> on-hit technique name
	Finish  string
	Riposte string
	Bash    string
	Combo   string
}

var csClassNames = map[string]string{"blade": "blade", "axe": "axe", "spear": "spear", "staff": "staff/club",
	"unarmed": "unarmed", "bow": "bow/thrown", "firearm": "firearm"}

var csClassOrder = []string{"blade", "axe", "spear", "staff", "unarmed", "bow", "firearm"}

var csSkinBase = csSkin{
	Stance: [4]string{"Guard", "Aggressive stance", "Evasive stance", "Steady aim"},
	Tech: map[string]string{"blade": "Lunge", "axe": "Cleave", "spear": "Pin", "staff": "Sweep",
		"unarmed": "Disarm", "bow": "Pinning shot", "firearm": "Suppressing burst"},
	Passive: map[string]string{"blade": "Open wound", "axe": "Hew", "spear": "Skewer", "staff": "Unbalance",
		"unarmed": "Stagger blow", "bow": "Hobble", "firearm": "Hold down"},
	Finish: "Finisher", Riposte: "Riposte", Bash: "Shield bash", Combo: "Momentum",
}

var csSkins = map[string]csSkin{
	"fantasy": {Stance: [4]string{"High guard", "Battle fury", "Fleet footwork", "Hunter's patience"},
		Finish: "Coup de grace", Combo: "Battle rhythm"},
	"mythic": {Stance: [4]string{"Aegis stance", "Heroic rage", "Winged step", "Archer's calm"},
		Tech:   map[string]string{"blade": "Heroic thrust", "axe": "Titan's cleave", "spear": "Pin the beast", "staff": "Sweep of ages", "unarmed": "Wrestler's throw"},
		Finish: "Kleos strike", Riposte: "Answering blow", Combo: "Glory"},
	"cyberpunk": {Stance: [4]string{"Hardened guard", "Overclock", "Ghost dodge", "Target lock"},
		Tech:    map[string]string{"blade": "Monowire lunge", "axe": "Rotor cleave", "spear": "Spike pin", "staff": "Arc sweep", "unarmed": "Joint lock", "bow": "Rail pin", "firearm": "Suppression burst"},
		Passive: map[string]string{"blade": "Hamstring cut", "unarmed": "Neural jab", "firearm": "Tracking round"},
		Finish:  "Kill-switch", Riposte: "Counter-script", Bash: "Plate slam", Combo: "Chain"},
	"wasteland": {Stance: [4]string{"Dig in", "Blood-up", "Scrabble", "Hold your breath"},
		Tech:   map[string]string{"blade": "Gut stab", "axe": "Hack and haul", "spear": "Skewer pin", "staff": "Knee-breaker", "unarmed": "Dirty grab", "bow": "Pin to the dirt", "firearm": "Spray and pray"},
		Finish: "Mercy kill", Riposte: "Cheap shot", Bash: "Door slam", Combo: "Frenzy"},
	"steampunk": {Stance: [4]string{"Fencer's guard", "Full pressure", "Pivot and glide", "Steady bore"},
		Tech:   map[string]string{"blade": "Gentleman's lunge", "axe": "Piston cleave", "spear": "Boiler pin", "staff": "Cane sweep", "unarmed": "Baritsu throw", "firearm": "Volley fire"},
		Finish: "Escapement strike", Riposte: "Parry-riposte", Combo: "Pressure"},
	"dieselpunk": {Stance: [4]string{"Cover up", "Pedal down", "Duck and weave", "Steady sights"},
		Tech:   map[string]string{"blade": "Trench thrust", "axe": "Wrecker's swing", "unarmed": "Brawler's hook", "firearm": "Tommy burst"},
		Finish: "Knockout", Combo: "Rally"},
	"atompunk": {Stance: [4]string{"Brace", "Overdrive", "Hover-step", "Calibrated aim"},
		Tech:   map[string]string{"blade": "Ion lunge", "axe": "Plasma cleave", "firearm": "Ray burst", "bow": "Needle pin"},
		Finish: "Critical mass", Combo: "Charge"},
	"biopunk": {Stance: [4]string{"Carapace guard", "Frenzy", "Slither", "Predator's stillness"},
		Tech:    map[string]string{"blade": "Barb lunge", "axe": "Rend", "spear": "Stinger pin", "staff": "Lash sweep", "unarmed": "Constricting hold", "bow": "Spore dart"},
		Passive: map[string]string{"blade": "Laceration", "unarmed": "Venom jab"},
		Finish:  "Devour", Riposte: "Reflex strike", Bash: "Shell bash", Combo: "Bloodrush"},
	"solarpunk": {Stance: [4]string{"Rooted stance", "Rising sun", "Wind step", "Still water"},
		Tech:   map[string]string{"blade": "Reed lunge", "axe": "Grove cleave", "spear": "Thorn pin", "staff": "Breeze sweep", "unarmed": "Gentle disarm", "bow": "Rooting shot"},
		Finish: "Harvest", Riposte: "Flowing counter", Combo: "Flow"},
	"twilight": {Stance: [4]string{"Veil guard", "Hungering stance", "Shade step", "Hush"},
		Tech:   map[string]string{"blade": "Gloaming lunge", "axe": "Dusk cleave", "spear": "Nail the shadow", "staff": "Gravewind sweep", "unarmed": "Wraith grasp"},
		Finish: "Last light", Riposte: "Shadow counter", Combo: "Dread"},
}

func csSkinFor(genre string) csSkin {
	s := csSkinBase
	o, ok := csSkins[genre]
	if !ok {
		return s
	}
	for i, n := range o.Stance {
		if n != "" {
			s.Stance[i] = n
		}
	}
	s.Tech, s.Passive = csMerge(s.Tech, o.Tech), csMerge(s.Passive, o.Passive)
	for _, p := range []struct {
		dst *string
		src string
	}{{&s.Finish, o.Finish}, {&s.Riposte, o.Riposte}, {&s.Bash, o.Bash}, {&s.Combo, o.Combo}} {
		if p.src != "" {
			*p.dst = p.src
		}
	}
	return s
}

func csMerge(a, b map[string]string) map[string]string {
	out := map[string]string{}
	for k, v := range a {
		out[k] = v
	}
	for k, v := range b {
		out[k] = v
	}
	return out
}
