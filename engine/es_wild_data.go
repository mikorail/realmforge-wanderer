package engine

// Text banks for wilderness traces (prefix ed, see es_wild.go). Templates use {placeholders} that
// edFill replaces; whatever is left over is stripped, so no raw braces can reach the player.

// edHabit is how a living monster behaves, told in the marks it leaves, the pattern of its kills and
// the advice a survivor would give a hunter. The group is picked from the monster's own species and lore.
type edHabit struct {
	Keys    []string
	Mark    []string
	Pattern []string
	Hint    []string
}

var edHabits = []edHabit{
	{Keys: []string{"dragon", "wyrm", "drake", "fire", "flame", "phoenix", "salamander", "ember", "sun"},
		Mark: []string{"bark blackened to the height of a mounted rider, the resin run to glass", "grass burnt in a long tongue, the stones cracked and fused"},
		Pattern: []string{"It hunts at dusk, always down the same valley, and burns what it does not eat",
			"It takes the lead beast of a train first, then waits for the rest to scatter",
			"It comes back to a kill on the third night, and only then"},
		Hint: []string{"Its breath is spent after a kill. Strike while it sulks, not before",
			"Wet ground and low cover blunt its fire. Fight from a ditch"}},
	{Keys: []string{"lake", "loch", "kraken", "eel", "leviathan", "sea", "mist", "serpent", "hydra", "bog", "marsh"},
		Mark: []string{"reeds flattened in a wide swathe down to the waterline, the mud furrowed by something dragged", "a trail of slime and torn bark from the shore up to the roots"},
		Pattern: []string{"It takes whatever comes to drink at first mist, and never farther than a hundred paces from the water",
			"It surfaces only when the air is still, and lets the fog do the hiding",
			"It drags its kills under and leaves the gear on the bank"},
		Hint: []string{"Stay off the wet ground. Draw it onto dry land and it tires fast",
			"It wants fog. Go on a clear, windy noon"}},
	{Keys: []string{"harpy", "griffin", "roc", "wyvern", "eagle", "owl", "bat", "hawk", "bird", "winged", "storm"},
		Mark: []string{"branches snapped from above, a long scratch in the bark at the height of the crowns", "feathers, and dropped bones with no tooth marks, as if let fall from a height"},
		Pattern: []string{"It drops from the sun side, takes one, and carries them up",
			"It nests on the high ground and hunts the open slopes at midday",
			"It never takes a man who walks under thick trees"},
		Hint: []string{"It will not follow you under heavy canopy. Fight beneath the trees",
			"Keep your back to rock. It strikes from behind or not at all"}},
	{Keys: []string{"ghoul", "wraith", "wight", "lich", "vampire", "undead", "shade", "ghost", "revenant", "bone"},
		Mark: []string{"frost on the bark in the shape of hands, the grass dead in a ring", "bark scored by long nails, and no tracks leading up to the marks"},
		Pattern: []string{"It walks only at night, along roads it knew in life",
			"It takes the one who sleeps last and leaves the rest untouched",
			"It never crosses a stream"},
		Hint: []string{"It fades at first light. Wait for dawn, or carry a flame",
			"A stream between you and it is as good as a wall"}},
	{Keys: []string{"titan", "giant", "colossus", "ogre", "troll", "behemoth", "mammoth", "qilin"},
		Mark: []string{"trunks leaning one way, stones the size of carts shifted from their beds", "a trough of trampled earth three paces wide, the saplings pressed flat"},
		Pattern: []string{"It follows the same track every moon, trampling what stands in it",
			"It sleeps through the day and walks at night, noisily",
			"It takes carts whole and picks them apart at leisure"},
		Hint: []string{"It is slow to turn. Keep moving and keep to its flank",
			"Do not stand in its track. Wait to either side"}},
	{Keys: []string{"wolf", "lion", "bear", "beast", "tiger", "hound", "boar", "cat", "panther", "jackal", "pack"},
		Mark: []string{"deep claw rakes on the trunks at a man's shoulder, a scent mark at every fork of the track", "tufts of coarse fur snagged in the bark, and fresh rakes over the old"},
		Pattern: []string{"It drives the lead animal into a gully and takes the stragglers",
			"It strikes at the rear of a column and is gone before the front turns",
			"It returns to the same carcass until nothing is left"},
		Hint: []string{"Fight with your back to a wall. It breaks off when it is hurt",
			"It will not leave its lair while there is food in it. Come quick, not quiet"}},
	{Keys: []string{"spider", "snake", "basilisk", "scorpion", "viper", "adder", "cobra", "worm", "centipede", "venom"},
		Mark: []string{"bark gone grey and weeping, the bones picked pale to the marrow", "webbing and shed skin in the branches, and a smell like old copper"},
		Pattern: []string{"It waits beside the trail and does not chase; its victims are found a day later",
			"It bites once and follows at a distance until the poison does its work",
			"It takes the one at the back of the line"},
		Hint: []string{"Carry an antidote. It breaks off when it is hurt",
			"Do not give it your back. It strikes at the moment you turn"}},
	{Keys: []string{"golem", "construct", "automaton", "machine", "drone", "engine", "iron", "clockwork", "mech"},
		Mark: []string{"ruts of heavy feet, scorched circles at regular spacing", "bark scraped bright by something hard and square"},
		Pattern: []string{"It patrols a fixed loop at fixed hours and does not eat",
			"It stops what moves and leaves what lies still",
			"It never leaves its beat, even to chase"},
		Hint: []string{"Strike between its passes, at the joints",
			"It will not leave its loop. Stay beyond the marked ground and shoot"}},
	{Keys: []string{""},
		Mark: []string{"claw rakes in the bark as high as a rider's saddle, fur snagged in the splinters", "a ring of picked bones and a smell no animal makes"},
		Pattern: []string{"It takes the last of any party and lets the rest run",
			"It strikes at the turn of the light and is gone before it is seen",
			"It comes back to the same place by the same path"},
		Hint: []string{"It guards its lair fiercely, so lure it out onto open ground",
			"Go in numbers, or do not go"}},
}

// victims of a lair, by kit

var edTravellerLines = []string{
	"Day {d}. {from} behind us, {to} ahead. The track is quiet, too quiet. I have marked the trees, and the marks are not an animal's work.",
	"Day {d}. Found the signs again today: {mark}. The guide will not go on. I have paid him off and will walk it alone.",
	"{pattern}. That is what the herders told me at {from}, and I laughed. I am not laughing. I will write the rest by the fire if I am let.",
}

var edTravellerGear = []string{
	"{weapon} snapped at the hilt",
	"{weapon}, its edge folded over from a downward blow",
	"empty {pack}, ripped from the seam",
}

var edLedgerLines = []string{
	"Ledger of {name}, {goods} factor of {from}. Carried: {n} loads for {to}. Paid at the last toll: {m} {cur}. Last line, in haste: something has kept pace with us two days. {pattern}. We turn the cart about.",
	"{name}, trader in {goods}, {from} to {to}. Loads checked at the ford: {n}. Beasts lost: two. In the margin: do not stop at this ridge. {mark}.",
	"Day book of {name}, {goods}, {from}. Sold at {to}: none. The mule would not pull past the marks. Someone has written under this line, in another hand, the single word run.",
}

var edKnightNotes = []string{
	"To my lord at {town}. I ride out to see what has been taking the herds. If I am not back in {n} days, send no one to look for me. {pattern}. Tell my {kin} the shield was good.",
	"Sworn to {state}, I will not leave this road unwatched. It is as the shepherds say: {pattern}. I am going in on foot, as the horse will not.",
	"Brother, if this reaches you, burn it. {mark} is the thing they warned us of. I made a mistake of pride, and I hope you will not repeat it.",
}

var edCharges = []string{"a rampant lion", "three wheat sheaves", "a white tower", "a crowned stag", "two crossed keys", "a black ship", "a rising sun", "a barbed wheel", "a heron", "a torn banner"}
var edFields = []string{"red", "green", "blue", "black", "silver", "gold", "grey", "purple"}
var edShieldTech = map[string]string{"cyberpunk": "riot shield", "wasteland": "scrap plate", "steampunk": "riveted buckler", "biopunk": "chitin shield", "solarpunk": "woven shield", "twilight": "tarnished shield"}

var edNestJunk = []string{"spoons", "buttons", "a dented crown", "a child's whistle", "coins", "a pocket watch", "brass keys", "a ring of keys", "a looking glass", "belt buckles", "a handful of rings", "a bent medal"}

// habits across the camps, hermits and shrines

var edGoods = []string{"salt", "wool", "iron nails", "dyed cloth", "wine", "tallow", "dried fish", "lamp oil", "grain", "seed stock"}
var edGoodsTech = []string{"copper wire", "batteries", "solder", "ration bricks", "med-kits", "filters", "fuel cells", "spare bearings", "radio tubes"}

var edCampWhy = []string{
	"left in haste; the packs are cut open and the tracks go in four directions",
	"moved on in good order, with only a bent pot forgotten",
	"abandoned after a night raid; the arrow holes in the tarp are all on the one side",
	"stopped by sickness, and the survivors went on without the dead",
}

var edHastyGraveLines = []string{
	"Here lie {n} of the {who}. We had no time to dig them deeper. Whoever you are, say a word for them.",
	"{n} buried here by their own. The rest went on. Do not move the stones.",
	"We could not carry them, so we left them here, with their boots on.",
}

var edArmyKinds = []string{"levy", "company", "column", "foragers", "vanguard"}

var edCampBoast = []string{
	"I have cleared {n} holes of what lives in them, and not one of them was a hard hole. {rv} does not boast, so I will.",
	"They will sing of this road. I have already written the first verse.",
	"{n} down and counting. The other fools can have the leavings.",
	"A true {calling} does not need a guide. I have a good map instead.",
}

var edRivalEnd = []string{
	"If you are reading this, you are behind me.",
	"Do not bother following. There will be nothing left.",
	"Somebody has been reading over my shoulder. If it is you, hello.",
}

var edMapTrue = []string{
	"A scrap of vellum with a hand-drawn coast and a cross inked firmly beside the name of {site}. The ink is the same age as the paper.",
	"A greasy map corner. A small cross, and under it in a hurry: {site}, mind the {thing}.",
}

var edMapFalse = []string{
	"A scrap of vellum with a hand-drawn coast and a cross inked firmly beside an empty headland. The ink is newer than the paper, and the coast is wrong in two places.",
	"A greasy map corner with a cross and the words in a flourish, a certain fortune. Whoever sold it was good at selling.",
}

var edThings = []string{"traps", "wardens", "water", "second door", "the stairs", "mould"}

// hermits

var edHermitReasons = map[string][]string{
	"death":    {"I left when the house became a grave I was sweeping.", "Everyone in {town} said the same kind words. I could not bear another."},
	"crime":    {"They put the blame on me for {why}, and I would rather be blamed by trees.", "A fine is paid and forgotten. The way they looked at me was not."},
	"feud":     {"The quarrel went on for years. I decided to be the one who walked away.", "There is no end to a feud where both sides are kin. So I went."},
	"illness":  {"I was ill, and watched them watch me. I do not need a wake before I have died.", "The sickness took the best of {town}. I took the long road out of the smell of it."},
	"marriage": {"We could not both be happy there, and she was the one who could.", "I stood at the wedding and it was not mine. I started walking that night."},
	"job":      {"The work became a wheel. I got off it.", "I had {job} to my name and nothing else. A cabin is a cheaper title."},
	"default":  {"I had a good life in {town} and could not name what was wrong with it.", "There is a kind of tired that sleep does not touch. The woods touch it."},
}

var edHermitWall = []string{
	"Carved above the door: {line}",
	"Scratched on the lintel in neat letters: {line}",
	"Burned into a plank with a poker: {line}",
}

var edHermitLoved = []string{
	"To {kin}. I am well. The garden does better than I do. If you come, bring salt, nothing else. I am not angry; I only could not stay. You will find the key under the third stone.",
	"{kin}, I wrote this four times and burned three. I miss your voice more than the town. I will not come back, but the path is open if you are brave.",
	"{kin}, there is a door here with your name on it. Whoever finds this, give it to them if they are in {town}, and tell them I said it was kinder this way.",
}

var edHermitAlive = []string{"A thin figure in patched clothes tends the bean rows, with the look of someone who talks mostly to goats.", "A weathered person sits on the step, mending a net or a sandal, and does not look up quickly."}
var edHermitDead = []string{"The bed has been made, and the blanket is folded square. Someone stopped here very quietly some time ago.", "A chair by the cold hearth, a book open on the seat, and a smell of old ash and mice."}

// shrines, waystones, gibbets

var edShrineGeneric = []string{
	"A flat-topped cairn before a niche, a stub of candle in it. A scratched sign of {deity} on the lintel, and a dried flower, and a clay cup of {food}.",
	"A weathered post with a bowl at its foot. Travellers have left {food}, a knot of thread and a few worn coins for {deity}.",
	"A wayside box of stone with a shingle roof. Inside, a carved token for {deity}, wax, and a few small bones of thanks.",
}

var edShrineOffers = []string{"a dried flower", "a knot of red thread", "a clay cup of {food}", "a stub of candle", "three worn coins", "a child's wooden toy", "a tarnished bell"}

var edGraffitiNames = []string{"was here, hungry", "walked to {town}, wished otherwise", "owes {cur} to a man in {town}", "loved it here, in spite of everything", "carried this stone for nothing"}
var edGraffitiLines = []string{"Do not drink at the second ford.", "Mind the dogs after the bend.", "The toll man cheats at the half.", "Home. Finally.", "Footsore, never fed.", "Turn back. Go round.", "Rain from here to {town}, every day."}

var edGibbetCrimes = map[string][]string{
	"deposed": {"treason against {state}", "tyranny and the murder of petitioners", "plotting against the realm"},
	"coup":    {"conspiracy against {state}", "the murder of the old guard"},
	"defeat":  {"cowardice and the loss of the war", "betrayal of the field"},
	"revol":   {"crimes against the people of {state}", "grinding the poor"},
	"default": {"banditry on the roads of {state}", "murder and robbery"},
	"gallows": {"{why}"},
}

var edGibbetPlacard = []string{
	"{name}, called {role}. Hanged on the order of the {state} court in the year {year} for {crime}. Let those who pass remember.",
	"Here hangs {name}, hanged for {crime}. The crows will not wait for the sentence to be read.",
	"{name}, {role}. Hanged {year}, for {crime}. Anyone who cuts this down answers for it.",
}

// cursed ground

var edFairyNames = []string{"the hunter's ring", "the white hart's ring", "the dancers' ring", "the wanderer's ring"}
var edFairyRing = []string{
	"A ring of pale toadstools in the grass, ten paces across. Inside the ring the air hums faintly and the grass is untrodden. Folk say it dizzies whoever steps in.",
	"A ring of toadstools, too even to be chance. The air inside is still and carries a thin sweet smell. The old tales warn that it dizzies and steals the hours.",
}
var edFairyNote = []string{
	"Chronicle: {name}, {role}, died in the year {year} of {cause}. The people here say they rode after a white hart into the ring, and the ring did not give them back. Their horse came home alone.",
}

var edBattleNames = []string{"the field of {state}", "the unquiet ground", "the crow field", "the spear meadow"}
var edBattleLook = []string{
	"Old bones, split shields and rusted arrowheads in the grass, and the crows will not land. Folk say standing too long here brings the dead's chill upon you.",
	"A churned field of weapons gone to red dust, with a smell of cold iron. Travellers say the ground still remembers, and that lingering here chills the nerve.",
}
var edBattleNote = []string{
	"In the year {year}, {war} was fought on this ground. The dead of {state} lie unburied under the grass. Do not stay past dusk.",
}

var edSpringLook = []string{
	"Dead fish and reeds float in a spring gone green and still. Nothing drinks here, and a warning post stands at the lip. It stings the skin of whoever wades in.",
	"A spring with pale scum, the bank dead for ten paces around. Birds do not land. The warning post says to keep out of the water.",
}
var edSpringNote = []string{
	"Do not drink or wade. This spring was fouled in the year {year}, when {cause}. {town} sends a man to check it each spring and has stopped sending the same man twice.",
}

// trail signs

var edRutLook = []string{
	"Fresh wheel ruts and hoofprints leave the road here, and the grass is trampled in a broad strip. A {what} passed {age} ago, heading for {to}.",
}
var edArmyLook = []string{
	"Churned earth, bootprints by the thousand and cold cook-fires in a long line. A column of {state} passed here {age} ago, heading for {to}.",
}

var edSmokeLines = []string{
	"A smudge of smoke stands on the horizon to the {dir}, thin at first and then dark. Something near {place} is burning.",
	"Smoke rises from the {dir}, too thick for a cookfire. It lies over {place}, and the birds are moving the other way.",
}
