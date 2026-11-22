package engine

// Static prose for the Encyclopedia (prefix enc). Everything that can be read from live game data
// (names, numbers, world facts) is built in encyclopedia.go; only the explanations the engine
// does not carry anywhere else live here.

// encCats is the category strip, in order. Index 0 is "All".
var encCats = []struct{ ID, Name string }{
	{"", "All"},
	{"attr", "Skills"},
	{"status", "Statuses"},
	{"combat", "Combat"},
	{"item", "Items"},
	{"species", "Species"},
	{"realm", "Realms"},
	{"cook", "Cooking"},
	{"magic", "Magic"},
	{"culture", "Cultures"},
	{"faith", "Faiths"},
	{"place", "Places"},
	{"person", "People"},
}

// encAttrText: what each attribute does (index STR..EGO).
var encAttrText = [6]string{
	"Raw might. Adds to melee damage and carrying capacity, and opens the heavier weapons and armor.",
	"Speed of hand and foot. Raises DV (dodging) and the chance to hit, and helps you slip away from trouble.",
	"Endurance. Sets your hit points, your hold against poison and disease, and how long you last in a long fight.",
	"Reason and learning. Helps with lore, tinkering, identifying oddities and the research behind the more studied kinds of magic.",
	"Mental strength. Sets mental armor against fear, charm and psychic attacks, and powers the mental gifts.",
	"Presence. Governs persuasion and prices, how warmly strangers answer you, and the loyalty of the companions you win.",
}

// encSkillExtra: skills that have no entry in skillDesc.
var encSkillExtra = map[string]string{
	"spear":   "Skill with spears and polearms. Reach and bracing; see the skill trees (K).",
	"flail":   "Skill with flails: weapons that blunt shields and ignore some armor.",
	"whip":    "Skill with whips: long reach, little force.",
	"staff":   "Skill with staves: sweeping blows that knock a foe off balance.",
	"fist":    "Skill with fist weapons and bare hands.",
	"hook":    "Skill with hooked blades, which snag and tear.",
	"thrown":  "Skill with thrown weapons: javelins, knives, darts.",
	"firearm": "Skill with firearms. Loud, long ranged and hungry for ammunition.",
	"heavy":   "Skill with great weapons that need two hands and real strength.",
	"shield":  "Skill with shields: a better block, and the shield bash.",
}

// encDamageText: what each damage type is and where it comes from.
var encDamageText = map[string]string{
	"slash":   "Cuts from blades and axes, and from claws. Armor turns it well; bone and hide of the heavily plated turn it better.",
	"pierce":  "Points: spears, arrows, bullets, fangs, thorns. Slips through gaps, but bone-bodied things barely feel it.",
	"blunt":   "Crushing blows from cudgels, flails, staves and fists. Armor helps less; the brittle fear it.",
	"fire":    "Flame, lava, burning oil and explosions. Sets grass and creatures alight (burning). Frost things take extra.",
	"cold":    "Frost and ice. Slows and can freeze a target solid (frozen). Fire things take extra.",
	"shock":   "Lightning and electricity. Arcs between foes; creatures standing in water take extra.",
	"poison":  "Venoms and toxins, often as a lingering effect (poisoned) that ends in sickness. The dead and the constructed do not feel it.",
	"acid":    "Corrosive spit and spills. It eats at armor and at flesh alike.",
	"holy":    "Radiance and smiting. The dead and the infernal flinch from it; the sanctified shrug it off.",
	"dark":    "Shadow and necrotic force. The living wither; the dead drink it in.",
	"psychic": "Blows to the mind. Ignores armor, is resisted by Willpower (mental armor), and means nothing to the mindless.",
}

// encStanceText: effect line per stance key, appended to live numbers.
var encStanceText = map[string]string{
	"guard": "Raises DV and chance to parry; you strike a little less surely. Costs no fatigue.",
	"aggr":  "Hit harder and surer; guard drops, and holding it tires you. Heavy fatigue stops the bonus.",
	"evade": "Best dodge; weak attacks. Costs some fatigue to hold.",
	"aim":   "Steady your aim: best ranged accuracy; you are easy to hit and dodge poorly.",
}

// encClassText: weapon classes of the combat layer.
var encClassText = map[string]string{
	"blade":   "Swords and daggers. Quick, accurate; criticals open wounds.",
	"axe":     "Axes. Heavy hits that can sever limbs.",
	"spear":   "Spears and polearms. Reach, and a pinning thrust.",
	"staff":   "Staves and clubs. Blunt blows that unbalance.",
	"unarmed": "Fists and grapples.",
	"bow":     "Bows and thrown weapons. Range and a crippling shot.",
	"firearm": "Guns. Loud, deadly and ammunition-hungry.",
}

// encItemKinds: item classes in the order shown. Text is generic; counts come from the live inventory.
var encItemKinds = []struct{ Kind, Name, Text string }{
	{"weapon", "Weapons", "Melee arms. Each belongs to a weapon skill; damage is dice plus bonuses, and Penetration Value (PV) decides how much of the target's armor it beats. Two-handed weapons fill both hands."},
	{"ranged", "Ranged weapons", "Bows, slings, guns and their kin. They use the missile skill (or firearms) and need ammunition of the right sort."},
	{"ammo", "Ammunition", "Arrows, bolts, bullets and shot. Stacks; spent when fired, and sometimes recovered."},
	{"armor", "Armor", "Worn on a body slot. Adds AV (absorbs damage) and sometimes subtracts DV. Heavy armor tires and slows the clumsy."},
	{"light", "Light sources", "Torches, lanterns and glowstones. They push back the dark of caves, nights and deep layers."},
	{"food", "Food", "Eaten to ease hunger. Some feed better or last longer; cooked meals grant tagged effects (see Cooking)."},
	{"ingredient", "Cooking ingredients", "Herbs, fungi, and the parts of beasts. Each carries a cooking tag. Combine two or three at a campfire or hearth."},
	{"water", "Water and drink", "Keeps thirst at bay. Carried in flasks and skins (see the Liquids screen, P)."},
	{"tonic", "Tonics and draughts", "Potions of healing, speed, strength, antidotes and the products of brewing. Some are unknown until tried or identified."},
	{"grenade", "Thrown devices", "Fire pots, frost flasks, flash powder. Thrown at a target or tile for an area effect."},
	{"book", "Books", "Read for lore, secrets, history and sometimes a skill or a place on the map."},
	{"letter", "Letters", "Messages you carry, deliver or read. Often tied to a quest or a person."},
	{"artifact", "Relics and artifacts", "Old things of power. Unidentified until studied; lore and tinkering help."},
	{"scrap", "Scrap and salvage", "Material to craft with at a workbench (T) or sell. Tinkering lets you take gear apart for it."},
	{"trophy", "Trophies and curios", "Keepsakes and proofs of a deed. Sets of curios are worth much more together (Collections, F9)."},
	{"key", "Keys", "Open the locks they were cut for."},
}

// encOtherStatuses: statuses applied elsewhere in the engine that have no table row. Name, text.
var encOtherStatuses = []struct{ ID, Name, Text string }{
	{"stunned", "Stunned", "Cannot act and loses DV. Wears off in a few turns. Blows from heavy weapons can cause it."},
	{"poisoned", "Poisoned", "Loses a little health every few turns, and ends in sickness. An antidote or a mending meal cuts it short."},
	{"burning", "Burning", "Takes fire damage each turn and sets nearby grass alight. Water puts it out."},
	{"bleeding", "Bleeding", "Loses a point of health each turn and leaves blood behind. Binding, healing or time stop it."},
	{"frozen", "Frozen", "Held fast in ice: cannot act and loses DV. Rime meals let you shake it off."},
	{"slowed", "Slowed", "Acts less often."},
	{"hasted", "Hasted", "Acts more often."},
	{"confused", "Confused", "Stumbles in random directions and attacks at random."},
	{"blind", "Blind", "Cannot see. Cannot aim at range, and sees nothing of the map."},
	{"shielded", "Shielded", "A protective effect absorbs part of the next blows."},
	{"mighty", "Mighty", "Strength is raised for a while."},
	{"mired", "Mired", "Stuck in mud, tar or sand; moving costs more."},
	{"entangled", "Entangled", "Held by webs, vines or nets until you struggle free."},
	{"crippled", "Crippled", "A damaged limb: slower, and weaker in a fight."},
	{"regen", "Regenerating", "Health returns each turn."},
	{"fed", "Well fed", "A cooked meal is working in you. Hover the meal in the cook screen (C) for what it gives; it fades with time."},
	{"rested", "Rested", "Slept well. Fatigue is low."},
	{"phased", "Phased", "Insubstantial: pass through walls and blows for a while."},
	{"stealth", "Hidden", "Foes have a harder time noticing you."},
	{"weakened", "Weakened", "Strength is sapped for a while."},
	{"winded", "Winded", "Out of breath; slower and less sure until it passes."},
	{"plague", "Plague", "A sickness caught from a place or a person. See Hazards in the journal."},
}

// encStatusNote: one extra line for table statuses whose effect is not a number.
var encStatusNote = map[string]string{
	"asleep":       "Cannot act. Attackers strike with extra penetration. Wakes, dazed, when hurt.",
	"terrified":    "Flees from the source of fear.",
	"syphoned":     "Loses a point of health each turn.",
	"choking":      "Loses health each turn until it clears.",
	"beguiled":     "Under another's charm.",
	"proselytized": "Swayed toward another's creed.",
	"glotrot":      "A rotting sickness of the mouth and throat.",
	"metabolizing": "Digesting something unusual.",
	"inspired":     "Heartened and clear-headed.",
	"hobbled":      "Cannot run; each step is slow.",
	"prone":        "Knocked down. Spends a turn getting up; costs DV.",
	"tumbling":     "Rolling clear of harm; hard to hit.",
	"trailblazing": "Quick over rough ground.",
}

// encMagicNote: general magic explainers.
const encMagicIntro = "Powers come from your genotype, your calling and your start. Active powers have a cooldown and a target (self, a direction, or a spot in sight); passive ones simply are. Defects are powers that cost you. Press {{W|a}} to use them."

const encArcaneIntro = "Beyond the powers you were born with, you can research your own: gather sources (fires, relics, strange creatures), design a power from an element and a form, test it, refine it, and bind patrons for more. Overreaching can backlash. Open it with {{W|F8}}."

// encCookIntro: how cooking works.
const encCookIntro = "Cook at a campfire or a town hearth ({{W|C}}) with no hostiles in sight. Combine 2 ingredients (3 with Survival 2) and each one's tag adds to the meal. A tag at two levels is stronger. A meal lasts about a day. New combinations are entered in your cookbook under a name from the land where you first cooked them."

const encRepIntro = "Every realm, faith and kind of creature keeps a regard for you. Deeds shift it: kind acts and gifts raise it; theft, murder and broken oaths lower it. Past a certain point people turn hostile on sight."

const encCombatIntro = "A swing hits if your to-hit beats the target's DV (dodge). A hit does its dice, then the target's AV (armor) soaks that much unless your weapon's PV (penetration) beats it. Damage types matter: many creatures resist or fear particular types."

// encTipLines: shown in the footer by context.
const encKeysList = "type to search  Tab/Left/Right category  Up/Down pick  Enter open  Esc back"
const encKeysDetail = "Up/Down links  Enter follow  PgUp/PgDn scroll  Esc/Backspace back  F1 close"
