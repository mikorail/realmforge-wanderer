package engine

// First-time tips for the newer systems (prefix h2). Order here is only the table; the order shown
// follows detection. Texts name the key to press and stay short and genre-neutral.
var h2Tips = map[string]string{
	"lk":       "A locked door or chest. Bump it to try your picks (a lockpick wears down) or carry the right key; keys are found, bought or cut from a mould.",
	"lx":       "You do not speak their tongue. Pick 'Teach me a few words' when you talk, buy a phrasebook, or choose 'Ask about the tongues' to see your fluency.",
	"cr":       "A crossroads card: a choice that shapes your story. Read the options, then pick one with the arrow keys and Enter.",
	"gchack":   "That machine can be hacked. Bump it and choose 'Jack in'; failed runs raise heat and a trace, so jack out in time.",
	"geiger":   "Radiation here. A geiger counter clicks faster as it climbs; rad-away, sealed gear and clean water help. Leave the hot ground to recover.",
	"gcbeacon": "A beacon-post stands cold while decay creeps in. Bump it with oil, a light or scrap to relight it, and ask the elders about the decay.",
	"gd":       "A device or shrine with its own screen. Bump it to tend it, feed it, or pray; the result needs time, so come back later.",
	"aim":      "A hostile in the clear with a missile weapon in hand: f fires and auto-aims at the best target, F lets you aim by hand.",
	"goods":    "You carry trade goods. Prices differ from town to town: buy low, sell high, and press F6 for the ledger.",
	"egg":      "You carry an egg. Keep it warm and it may hatch; leave it cold or rough-handled and it spoils.",
	"hungry":   "You are hungry. Press e to eat something from your pack; cooking (C) stretches rations further.",
	"tired":    "You are tired. Press F5 to sleep; rest in a bed or a safe place and you wake clear-headed.",
	"injury":   "You are injured. Injuries linger and weaken you until they heal with rest, a healer or the right remedy.",
	"person":   "Someone you know is nearby. Talk and pick 'Who do I know?' to open the People screen with what you learned about them.",
	"secret":   "You sense a hidden way nearby. Press s to search near walls, panels and odd inscriptions.",
	"tame":     "You tamed a beast. It follows you and wants feeding; check its state and name it in the taming screen.",
	"collect":  "You picked up a collectible. Press F9 to see your collections and which pieces you still lack.",
	"brew":     "Those ingredients can be brewed. Press F4 to open the alchemy screen and mix them into potions.",
	"sneak":    "Foes nearby have not noticed you. Press a and pick Sneak to creep past, or strike unseen for bonus damage.",
	"rival":    "A rival is in sight. Rivals race you for fame and prizes and can turn hostile; press I to read the Hunters dossier.",
	"nemesis":  "A nemesis hunts you. Press I to read the Hunters dossier and learn how to face, flee or end it.",
	"lv2":      "Level 2. Spend your points: c for attributes and skills, K for the skill trees, V for mutations or called shots.",
	"lv5":      "Level 5. Spent points on c, K and V yet? Unspent points are wasted power.",
	"lv10":     "Level 10. Check c, K and V again for points, and F10 for the places and titles you have earned.",
	"achieve":  "You earned an achievement. Press F10 to read your achievements and titles, and wear one.",
	"place":    "A place worth marking. Press F10 to open Places, name it, and use it to find your way back.",
}
