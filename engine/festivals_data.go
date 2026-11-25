package engine

// Data for festivals.go (prefix fv): the seven festival activities, which feast kinds run which, and the
// seasonal dressing of the festival grounds.

type fvAct struct {
	ID, Name, Short string
	Intro           string
	Opts            [3]string // lettered choices a b c ("" = unused)
	Kinds           [3]string // check kind per choice (checkKinds ids; "" = special handling)
	Legs            []string  // what each stage of the contest is called
	Fee             int       // entry fee in units of the base fee
}

var fvActs = []fvAct{
	{ID: "race", Name: "Footrace and obstacle run", Short: "the footrace", Fee: 1,
		Intro: "A course is chalked around the plaza: barrels to vault, a rope to duck, a ditch to leap, and the long straight to the stage.",
		Opts:  [3]string{"Run flat out", "Pace yourself over the whole course", "Take the obstacles at speed"},
		Kinds: [3]string{"", "endurance", "acrobatics"},
		Legs:  []string{"the barrels", "the rope and the ditch", "the final straight"}},
	{ID: "archery", Name: "Archery and throwing contest", Short: "the archery", Fee: 1,
		Intro: "Straw targets stand at three distances. A marshal counts arrows and calls every miss aloud.",
		Opts:  [3]string{"Shoot at the long targets", "Throw knives and axes at the boards", "Hit the swinging rings"},
		Kinds: [3]string{"", "", "perception"},
		Legs:  []string{"the near target", "the middle target", "the far target"}},
	{ID: "bout", Name: "Wrestling and duelling bout", Short: "the bout", Fee: 2,
		Intro: "A ring of rope and sand. The rules are plain: no killing, and a yield ends it. A champion waits for a challenger.",
		Opts:  [3]string{"Wrestle in the ring", "Duel with blunt blades", ""},
		Kinds: [3]string{"", "", ""}, Legs: []string{"the bout"}},
	{ID: "cook", Name: "Cooking and brewing contest", Short: "the cooking contest", Fee: 1,
		Intro: "Judges in aprons walk a row of pots and cups with spoons held ready. Your cookbook and pack count; nothing in them is used up.",
		Opts:  [3]string{"Present your best-known dish", "Improvise from what you carry", "Enter the festival dish of the day"},
		Kinds: [3]string{"survival", "survival", "survival"},
		Legs:  []string{"the look", "the taste"}},
	{ID: "story", Name: "Storytelling and verse contest", Short: "the storytelling", Fee: 1,
		Intro: "A bench by the stage, a bowl of water for the speaker, and a crowd that knows every proverb and will not be fooled.",
		Opts:  [3]string{"Say a proverb of the land and explain it", "Tell an old tale or epic", "Tell your own story"},
		Kinds: [3]string{"persuasion", "persuasion", "persuasion"},
		Legs:  []string{"the telling", "the ending"}},
	{ID: "market", Name: "Festival market stalls", Short: "the festival stalls", Fee: 0,
		Intro: "Stalls in the day's colours sell what is only made for this feast.",
		Opts:  [3]string{"", "", ""}, Kinds: [3]string{"", "", ""}},
	{ID: "dance", Name: "Dance and music", Short: "the dance", Fee: 1,
		Intro: "A ring of dancers around the musicians. The crowd claps the beat, and the best dancer is lifted on shoulders.",
		Opts:  [3]string{"Play for the dance", "Join the dance", "Lead the line dance"},
		Kinds: [3]string{"", "acrobatics", "persuasion"},
		Legs:  []string{"the first round", "the last round"}},
}

// fvKindActs: which activities run on which feast kind (feastStyle kinds in calendar.go).
var fvKindActs = map[string][]string{
	"spring":    {"race", "dance", "market", "story", "cook"},
	"harvest":   {"cook", "market", "race", "dance", "archery"},
	"dead":      {"story", "cook", "market", "dance", "race"},
	"midwinter": {"cook", "story", "market", "dance", "bout", "archery"},
	"sea":       {"race", "archery", "bout", "cook", "market", "dance", "story"},
	"war":       {"bout", "archery", "race", "market", "cook", "dance"},
	"learning":  {"story", "cook", "market", "dance", "archery"},
	"sun":       {"race", "dance", "archery", "market", "cook", "story"},
}

// fvKindBonus: the activity that suits the feast gets +1 to the check.
var fvKindBonus = map[string]string{
	"spring": "dance", "harvest": "cook", "dead": "story", "midwinter": "cook", "sea": "race", "war": "bout", "learning": "story", "sun": "race",
}

type fvDressT struct {
	Stage, Stall, Herald string
	Ring, Corner, Spare  Terrain
	Sign                 string
}

var fvDress = map[string]fvDressT{
	"spring":    {"flower stage", "blossom and honey stall", "spring herald", TBunting, TLantern, TBunting, "Spring fair: bring boughs, bring a good voice."},
	"harvest":   {"harvest board", "pie and cider stall", "harvest reeve", TBunting, TBarrel, TBarrel, "Harvest fair: sheaves, casks and prizes for the best of the year."},
	"dead":      {"lamp stage", "lamp and bread stall", "keeper of lamps", TLantern, TBrazier, TLantern, "Night of lamps: speak softly, remember the dead, enter the contests."},
	"midwinter": {"fireside stage", "mulled wine stall", "winter crier", TBrazier, TLantern, TBrazier, "The long night: fires, stories and warm cups."},
	"sea":       {"quay stage", "fish and net stall", "harbour crier", TBunting, TBarrel, TBunting, "Boat feast: races, throwing and a toast to every sailor."},
	"war":       {"banner stage", "sausage and ale stall", "marshal of the ring", TBunting, TBrazier, TDummy, "Victory feast: the ring is open to all who dare."},
	"learning":  {"reading stage", "ink and sweets stall", "master of verse", TLantern, TLantern, TBunting, "Scholars' feast: verse, tales and quiet contests."},
	"sun":       {"sunstage", "skewer and ember stall", "keeper of the bonfire", TBrazier, TBunting, TLantern, "High sun: bonfires, footraces and dancing."},
}

// built-in proverbs when the culture has no voice pack
var fvProverbs = []string{
	"A good net is mended before the storm.",
	"The road is shorter with a story in it.",
	"Salt keeps the meat and the friend alike.",
	"One honest weight outlasts three clever scales.",
	"Do not ask the river to carry what the bridge refused.",
	"A full granary makes a quiet town.",
	"The lamp you carry for another lights your own feet.",
	"Even the wall was once a pile of stones that agreed.",
}

// rival names' epithets
var fvEpithets = []string{"the Quick", "the Steady", "the Broad", "of the lower town", "the Younger", "the Loud", "the Ox", "the Lame-footed", "the Red", "from the hills"}

// market stock: id, label, base price, builder key
type fvStock struct {
	Label string
	Price int
}

var fvStalls = []fvStock{
	{"the festival dish of the day", 18},
	{"a bag of festival sweets", 8},
	{"a festival lantern", 30},
	{"a flask of the festival tonic", 40},
	{"a festival favour (a ribbon keepsake)", 12},
}

// result text by place
var fvPlaceText = [5]string{"", "You win. The crowd lifts its cups and the judges tie the ribbon of first place to your arm.", "Second place. A fine showing, and a ribbon to show for it.", "Third place, and a small ribbon. The crowd will remember you.", "You do not place. The crowd claps anyway, and somebody offers you a cup."}

var fvPlaceName = [5]string{"", "first", "second", "third", ""}
