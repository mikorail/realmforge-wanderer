package engine

// Afflictions data (prefix dz): the catalogue of chronic and progressive conditions. Attribute indexes in
// dzFx.A follow STR AGI TOU INT WIL EGO. Effects ride on the status registry (Speed, DV, Hit, AV) and on
// dzScoreMod for attributes. Herb cures name a brewing property of alchemy.go (alProps ids).

type dzFx struct {
	Speed, DV, Hit, AV int
	A                  [6]int
}

func (f dzFx) zero() bool { return f == dzFx{} }

type dzStage struct {
	Name                 string
	Days                 int // days in this stage before the next (0 = last)
	Text                 string
	Fx                   dzFx
	Night                dzFx // extra effect at night (boons and night-blindness)
	Drain                int  // HP lost per hour
	Hunger, Thirst       int  // extra per hour
}

type dzCure struct {
	Kind string // herb doctor priest shrine kill grave ward time
	Prop string // herb: alchemy property id
	Rank int    // priest: faith rank needed (1..3)
	Max  int    // works fully only while stage <= Max (1-based); 0 any
	Pow  int    // 0 arrest, 1 step back a stage, 2 cure
	Hint string
}

type dzDef struct {
	ID, Name string
	Skin     map[string]string
	Subst    map[string]string // genre -> id that stands in
	Genres   string            // space list of genres where it can occur; "" any
	Kind     string            // disease curse bite taint parasite poison
	Loop     bool
	Natural  int // percent chance per day to fade at stage 1, or per cycle for loops
	Spread   int // 0 none, 1 companions, 2 companions and townsfolk
	Fee      int
	Boon     string
	Lore     string
	Stages   []dzStage
	Cures    []dzCure
}

func dzS(name string, days int, text string, fx dzFx, drain int) dzStage {
	return dzStage{Name: name, Days: days, Text: text, Fx: fx, Drain: drain}
}

func dzFxA(speed, dv, hit, av int, a [6]int) dzFx { return dzFx{Speed: speed, DV: dv, Hit: hit, AV: av, A: a} }

var dzDefs = []*dzDef{
	{ID: "wasting_lung", Name: "Wasting lung", Kind: "disease", Fee: 40, Natural: 4, Subst: map[string]string{"wasteland": "rad_sores"},
		Lore: "Dust, ash and old smoke settle in the chest and do not leave.",
		Stages: []dzStage{
			dzS("Dry cough", 6, "A dry cough that comes in the mornings.", dzFxA(-3, 0, 0, 0, [6]int{}), 0),
			dzS("Breathless", 8, "You stop to breathe on every slope.", dzFxA(-12, -1, 0, 0, [6]int{-1, 0, -2, 0, 0, 0}), 0),
			dzS("Wasting", 0, "Your ribs show and every breath rattles.", dzFxA(-25, -2, -2, 0, [6]int{-3, 0, -4, 0, 0, 0}), 1),
		},
		Cures: []dzCure{
			{Kind: "herb", Prop: "cleansing", Max: 2, Pow: 1, Hint: "A cleansing draught clears the chest for a while."},
			{Kind: "herb", Prop: "healing", Pow: 0, Hint: "A healing brew holds the wasting off."},
			{Kind: "doctor", Max: 2, Pow: 2, Hint: "A physician can drain and treat it while it is young."},
			{Kind: "priest", Rank: 1, Max: 2, Pow: 1, Hint: "A priest's blessing eases the breath."},
			{Kind: "shrine", Pow: 2, Hint: "Pilgrims go to a place of clean air and holy water to be rid of it."},
			{Kind: "ward", Pow: 0, Hint: "A master's amulet of clean air keeps it from worsening."},
		}},
	{ID: "marsh_fever", Name: "Marsh fever", Kind: "disease", Loop: true, Natural: 20, Fee: 35, Spread: 0,
		Lore: "Bitten at dusk beside still water. It leaves, and it comes back.",
		Stages: []dzStage{
			dzS("Chills", 2, "Shivering at dusk, hot by night.", dzFxA(-8, -1, 0, 0, [6]int{}), 0),
			dzS("Burning fever", 3, "A fever that turns the world to smear.", dzFxA(-20, -3, -2, 0, [6]int{-2, 0, -2, 0, 0, 0}), 1),
			dzS("Remission", 5, "You feel nearly well. It will come back.", dzFxA(0, 0, 0, 0, [6]int{}), 0),
		},
		Cures: []dzCure{
			{Kind: "herb", Prop: "feverc", Pow: 2, Hint: "A fever-cure brew ends it for good."},
			{Kind: "herb", Prop: "antidote", Pow: 1, Hint: "An antidote breaks a single bout."},
			{Kind: "doctor", Pow: 2, Hint: "Any physician knows the bitter bark that cures it."},
			{Kind: "priest", Rank: 1, Pow: 1, Hint: "Priests pray the fever down."},
		}},
	{ID: "rot_blight", Name: "Rot-blight", Kind: "disease", Spread: 2, Fee: 60, Subst: map[string]string{"cyberpunk": "data_virus"},
		Lore: "A creeping rot in the flesh, spread by touch and by bad cold meat.",
		Stages: []dzStage{
			dzS("Dark spots", 5, "Dark spots on the skin. The smell is faint but there.", dzFxA(-3, 0, 0, 0, [6]int{0, 0, 0, 0, 0, -1}), 0),
			dzS("Spreading rot", 6, "The rot spreads. People step away from you.", dzFxA(-8, -1, 0, 0, [6]int{0, 0, -2, 0, 0, -3}), 1),
			dzS("Gangrene", 0, "Flesh dies and falls away from the bone.", dzFxA(-20, -3, -2, 0, [6]int{-2, 0, -4, 0, 0, -5}), 2),
		},
		Cures: []dzCure{
			{Kind: "herb", Prop: "cleansing", Max: 2, Pow: 2, Hint: "A cleansing wash and salve stops it early."},
			{Kind: "doctor", Max: 2, Pow: 2, Hint: "A physician can cut out the dead flesh and burn it clean."},
			{Kind: "priest", Rank: 2, Pow: 1, Hint: "A priest of rank can burn the rot with blessed fire."},
			{Kind: "ward", Pow: 0, Hint: "A ward-amulet from a master holds the rot back."},
		}},
	{ID: "the_shakes", Name: "The shakes", Kind: "poison", Fee: 45, Natural: 6,
		Lore: "Too much poison and bad water leave the hands unsteady.",
		Stages: []dzStage{
			dzS("Tremor", 8, "Your fingers will not stay still on a cup.", dzFxA(0, 0, -1, 0, [6]int{}), 0),
			dzS("The shakes", 12, "Your hands shake. Aim and balance suffer.", dzFxA(0, -1, -3, 0, [6]int{0, -2, 0, 0, 0, 0}), 0),
			dzS("Palsy", 0, "Your whole body trembles.", dzFxA(-8, -3, -5, 0, [6]int{0, -4, 0, -1, 0, 0}), 0),
		},
		Cures: []dzCure{
			{Kind: "herb", Prop: "calm", Pow: 1, Hint: "A calming brew steadies the hands for a spell."},
			{Kind: "herb", Prop: "antidote", Max: 2, Pow: 2, Hint: "An antidote drives the old poison out while it is young."},
			{Kind: "doctor", Max: 2, Pow: 2, Hint: "A physician can purge it early."},
			{Kind: "ward", Pow: 0, Hint: "A master's amulet steadies the nerves."},
		}},
	{ID: "night_blind", Name: "Night-blindness", Kind: "disease", Natural: 10, Fee: 30,
		Lore: "Too long in the dark and the eyes forget the dusk.",
		Stages: []dzStage{
			{Name: "Dim at dusk", Days: 6, Text: "Dusk comes early to your eyes.", Night: dzFxA(0, -2, -2, 0, [6]int{})},
			{Name: "Night-blind", Days: 10, Text: "After dark you see only shapes.", Night: dzFxA(-10, -4, -4, 0, [6]int{}), Fx: dzFxA(0, 0, 0, 0, [6]int{0, 0, 0, -1, 0, 0})},
			{Name: "Blind by dark", Text: "After dark you are close to blind.", Night: dzFxA(-20, -6, -6, 0, [6]int{}), Fx: dzFxA(0, 0, -1, 0, [6]int{0, 0, 0, -2, 0, 0})},
		},
		Cures: []dzCure{
			{Kind: "herb", Prop: "nightsight", Pow: 2, Hint: "A night-sight brew sharpens the eyes at once."},
			{Kind: "time", Pow: 2, Hint: "Three days in daylight, outdoors, and the eyes recover."},
			{Kind: "doctor", Max: 2, Pow: 2, Hint: "A physician prescribes liver and greens."},
		}},
	{ID: "fungal_spores", Name: "Fungal spores", Kind: "disease", Spread: 1, Fee: 50, Subst: map[string]string{"biopunk": "spore_bloom"},
		Lore: "Grey spores settle in the lungs and skin and begin to feed.",
		Stages: []dzStage{
			dzS("Itch", 4, "An itch under the skin that spreads.", dzFxA(0, 0, 0, 0, [6]int{}), 0),
			dzS("Grey patches", 8, "Grey patches crust over your arms.", dzFxA(-5, 0, 0, 0, [6]int{0, 0, 0, 0, -1, 0}), 0),
			dzS("Fruiting", 0, "Small caps push out of the patches.", dzFxA(-15, -1, 0, 0, [6]int{0, 0, 0, -2, -3, -3}), 1),
		},
		Cures: []dzCure{
			{Kind: "herb", Prop: "cleansing", Max: 2, Pow: 2, Hint: "A cleansing wash kills the young growth."},
			{Kind: "doctor", Max: 2, Pow: 2, Hint: "A physician scrapes and burns the patches."},
			{Kind: "priest", Rank: 2, Pow: 1, Hint: "A blessed flame can sear it out."},
			{Kind: "ward", Pow: 0, Hint: "A ward-amulet keeps spores from taking."},
		}},
	{ID: "lycanthropy", Name: "Lycanthropy", Kind: "bite", Fee: 0, Skin: map[string]string{"biopunk": "Splice-fury", "cyberpunk": "Berserk firmware", "wasteland": "The howling mutation"},
		Boon: "At night you are faster, stronger and harder to hit. By day you are slower of mind. The beast takes hold in the dark and you may lose hours.",
		Lore: "A bite from something that walks as a beast at night.",
		Stages: []dzStage{
			{Name: "Fever dreams", Days: 4, Text: "You dream of running on four legs.", Fx: dzFxA(-3, 0, 0, 0, [6]int{}), Night: dzFxA(5, 0, 1, 0, [6]int{1, 0, 0, 0, 0, 0})},
			{Name: "The change", Days: 8, Text: "Your teeth ache and hair comes in on the backs of your hands.", Fx: dzFxA(0, 0, 0, 0, [6]int{0, 0, 0, -1, -1, 0}), Night: dzFxA(15, 1, 2, 1, [6]int{3, 1, 1, 0, 0, 0}), Hunger: 20},
			{Name: "The beast", Text: "By night you are a beast, and the day is a fog.", Fx: dzFxA(0, 0, 0, 0, [6]int{0, 0, 0, -2, -3, -2}), Night: dzFxA(25, 2, 3, 2, [6]int{5, 2, 2, 0, 0, 0}), Hunger: 40},
		},
		Cures: []dzCure{
			{Kind: "kill", Max: 2, Pow: 2, Hint: "Kill the one who bit you before the beast takes hold, and the curse breaks."},
			{Kind: "herb", Prop: "antidote", Max: 1, Pow: 1, Hint: "A purging antidote can hold it back in the first days."},
			{Kind: "priest", Rank: 3, Pow: 2, Hint: "Only a high priest can lift it by rite."},
			{Kind: "shrine", Max: 2, Pow: 2, Hint: "A holy place can wash it out of you in the early days."},
			{Kind: "ward", Pow: 0, Hint: "A master's silver amulet keeps the change at bay."},
		}},
	{ID: "vampiric_taint", Name: "Vampiric taint", Kind: "bite", Skin: map[string]string{"cyberpunk": "Bloodlust code", "biopunk": "Hematic strain"},
		Boon: "At night you are swift, strong and sure. By day you are weak and the sun is hard on you. You hunger often.",
		Lore: "The bite of the blood-drinkers leaves a thirst that grows.",
		Stages: []dzStage{
			{Name: "Pale", Days: 5, Text: "You are pale and the light hurts.", Fx: dzFxA(0, 0, -1, 0, [6]int{0, 0, -1, 0, 0, 0}), Night: dzFxA(5, 0, 1, 0, [6]int{})},
			{Name: "Thirst", Days: 9, Text: "Food no longer fills you. You watch the veins in people's throats.", Fx: dzFxA(-5, 0, 0, 0, [6]int{0, 0, -2, 0, 0, 0}), Night: dzFxA(12, 1, 2, 0, [6]int{2, 2, 0, 0, 0, 0}), Hunger: 25},
			{Name: "The taint", Text: "The dark is your country and the sun is an enemy.", Fx: dzFxA(-12, 0, -2, 0, [6]int{0, 0, -3, 0, 0, 0}), Night: dzFxA(20, 2, 3, 1, [6]int{4, 3, 0, 0, 0, 0}), Hunger: 45},
		},
		Cures: []dzCure{
			{Kind: "kill", Max: 2, Pow: 2, Hint: "Destroy the one who sired the taint before it roots."},
			{Kind: "priest", Rank: 3, Pow: 2, Hint: "A high priest can cast the taint out."},
			{Kind: "shrine", Max: 2, Pow: 2, Hint: "A holy place of the sun burns it out in the early days."},
			{Kind: "ward", Pow: 0, Hint: "A blessed amulet keeps the thirst down."},
		}},
	{ID: "ghost_sickness", Name: "Ghost-sickness", Kind: "curse", Skin: map[string]string{"cyberpunk": "Ghost in the wires", "atompunk": "Echo sickness"}, Natural: 3,
		Lore: "The dead leave a cold in the living who disturb them.",
		Stages: []dzStage{
			dzS("Chill", 4, "A cold that no fire reaches.", dzFxA(-5, 0, 0, 0, [6]int{0, 0, 0, 0, -1, 0}), 0),
			dzS("Haunted", 6, "You hear someone breathe beside you.", dzFxA(-10, -2, 0, 0, [6]int{0, 0, 0, -1, -3, 0}), 0),
			dzS("Fading", 0, "You are thin as smoke and the cold has settled in.", dzFxA(-20, -4, -3, 0, [6]int{0, 0, -3, 0, -5, 0}), 1),
		},
		Cures: []dzCure{
			{Kind: "grave", Pow: 2, Hint: "Return something of value to the grave you disturbed and lay it down."},
			{Kind: "priest", Rank: 2, Pow: 2, Hint: "A priest of rank can lay the dead to rest."},
			{Kind: "herb", Prop: "calm", Max: 1, Pow: 1, Hint: "A calming brew quiets the whispers early on."},
			{Kind: "shrine", Pow: 1, Hint: "A holy place gives rest from the cold."},
			{Kind: "ward", Pow: 0, Hint: "A master's amulet keeps the dead at bay."},
		}},
	{ID: "mummy_curse", Name: "Mummy's curse", Kind: "curse", Skin: map[string]string{"wasteland": "Vault curse"},
		Lore: "Tomb dust and a king's anger make the living wither.",
		Stages: []dzStage{
			dzS("Dry skin", 5, "Your skin cracks and flakes.", dzFxA(-3, 0, 0, 0, [6]int{0, 0, -1, 0, 0, 0}), 0),
			dzS("Withering", 8, "You wither. Your hair comes out in handfuls.", dzFxA(-10, 0, 0, 0, [6]int{-2, 0, -3, 0, 0, 0}), 1),
			dzS("Dust", 0, "You crumble a little more each day.", dzFxA(-22, -2, -2, 0, [6]int{-4, -3, -5, 0, 0, 0}), 2),
		},
		Cures: []dzCure{
			{Kind: "grave", Pow: 2, Hint: "Return a worthy offering to the tomb that cursed you."},
			{Kind: "priest", Rank: 3, Pow: 1, Hint: "A high priest can slow it."},
			{Kind: "shrine", Pow: 1, Hint: "A holy place can ease the wither."},
			{Kind: "ward", Pow: 0, Hint: "A master's amulet holds the dust back."},
		}},
	{ID: "item_binding", Name: "Item binding", Kind: "curse", Skin: map[string]string{"cyberpunk": "Locked implant", "biopunk": "Graft bond"},
		Lore: "A cursed thing has taken hold of you. The longer you keep it, the more it keeps you.",
		Stages: []dzStage{
			dzS("Bound", 10, "You find it hard to put the thing down.", dzFxA(0, 0, 0, 0, [6]int{0, 0, 0, 0, -1, 0}), 0),
			dzS("Possessed", 14, "It speaks to you in your own voice.", dzFxA(0, -1, 0, 0, [6]int{0, 0, 0, 0, -3, -2}), 0),
			dzS("Claimed", 0, "It wears you more than you wear it.", dzFxA(0, 0, -2, 0, [6]int{0, 0, 0, -2, -5, -4}), 1),
		},
		Cures: []dzCure{
			{Kind: "priest", Rank: 1, Pow: 2, Hint: "A priest can break the binding and unmake the curse on the thing."},
			{Kind: "shrine", Pow: 2, Hint: "A holy place can unbind you."},
			{Kind: "herb", Prop: "cleansing", Max: 1, Pow: 1, Hint: "A cleansing wash loosens the first grip."},
			{Kind: "ward", Pow: 0, Hint: "A master's amulet keeps it from deepening."},
		}},
	{ID: "witch_hex", Name: "Witch-hex", Kind: "curse", Natural: 6, Skin: map[string]string{"steampunk": "Tinker's jinx", "cyberpunk": "Daemon hex"},
		Lore: "A curse of ill luck laid by someone who means it.",
		Stages: []dzStage{
			dzS("Bad luck", 5, "Small things go wrong all day.", dzFxA(0, -1, -1, 0, [6]int{}), 0),
			dzS("Hexed", 8, "Doors stick and blades slip.", dzFxA(0, -2, -2, 0, [6]int{0, 0, 0, -1, -1, -2}), 0),
			dzS("Doom-marked", 0, "Animals shy from you.", dzFxA(-8, -3, -3, 0, [6]int{0, 0, 0, 0, -2, -3}), 0),
		},
		Cures: []dzCure{
			{Kind: "kill", Pow: 2, Hint: "Kill the one who laid it and it falls apart."},
			{Kind: "priest", Rank: 2, Pow: 2, Hint: "A priest of rank can turn the hex."},
			{Kind: "herb", Prop: "cleansing", Max: 2, Pow: 1, Hint: "A cleansing bath washes the first of it away."},
			{Kind: "ward", Pow: 0, Hint: "A ward-amulet turns the hex aside."},
		}},
	{ID: "god_mark", Name: "God-mark", Kind: "curse", Skin: map[string]string{"cyberpunk": "Root sign", "biopunk": "Prime mark"},
		Boon: "Willpower, ego and resolve rise with the mark. The mark burns and hungers, and priests of other faiths do not like the sight of it. Prayer at a temple holds the burn off.",
		Lore: "A god has laid a hand on you. It is a gift and a debt.",
		Stages: []dzStage{
			dzS("Touched", 8, "A warm brand on your palm that others cannot see.", dzFxA(0, 0, 1, 0, [6]int{0, 0, 0, 0, 1, 0}), 0),
			dzS("Marked", 14, "The mark glows when you lie. Strangers stare.", dzFxA(0, 1, 1, 0, [6]int{0, 0, 0, 1, 2, 1}), 1),
			dzS("Chosen", 0, "You are a lamp in a dark house, and it costs you.", dzFxA(0, 2, 2, 1, [6]int{0, 0, 0, 2, 4, 2}), 2),
		},
		Cures: []dzCure{
			{Kind: "priest", Rank: 3, Pow: 2, Hint: "Only a high priest can break the mark."},
			{Kind: "shrine", Pow: 2, Hint: "At the god's own place you may lay the mark down, or take it up."},
			{Kind: "priest", Rank: 1, Pow: 0, Hint: "Any priest of the faith can pray over it and hold the burn off for a week."},
		}},
	{ID: "iron_sickness", Name: "Iron-sickness", Kind: "taint", Natural: 8, Fee: 55,
		Lore: "Metal that was tainted in the forge or the grave slowly poisons whoever carries it.",
		Stages: []dzStage{
			dzS("Metal taste", 6, "There is a taste of rust in everything.", dzFxA(-3, 0, 0, 0, [6]int{}), 0),
			dzS("Iron-sick", 10, "Your joints ache and your gums bleed.", dzFxA(-8, 0, -1, 0, [6]int{-2, 0, -2, 0, 0, 0}), 0),
			dzS("Blood-rust", 0, "Your blood runs thick and slow.", dzFxA(-18, 0, 0, -1, [6]int{-4, -2, -4, 0, 0, 0}), 1),
		},
		Cures: []dzCure{
			{Kind: "time", Pow: 2, Hint: "Put the tainted metal down and the early sickness fades with days."},
			{Kind: "herb", Prop: "cleansing", Pow: 1, Hint: "A cleansing draught flushes the metal out."},
			{Kind: "doctor", Pow: 1, Hint: "A physician can draw the blood clean."},
			{Kind: "priest", Rank: 1, Pow: 1, Hint: "A priest's blessing helps the body clear it."},
		}},
	{ID: "sea_madness", Name: "Sea-madness", Kind: "curse", Natural: 5, Skin: map[string]string{"cyberpunk": "Deep-call"},
		Lore: "Too long on the open water, or too close to something under it.",
		Stages: []dzStage{
			dzS("Salt-eyed", 4, "You stare at the horizon longer than you mean to.", dzFxA(0, 0, 0, 0, [6]int{0, 0, 0, -1, 0, 0}), 0),
			dzS("Hearing the deep", 6, "Something sings below the boards.", dzFxA(0, -2, -1, 0, [6]int{0, 0, 0, -2, -3, 0}), 0),
			dzS("Drawn to the water", 0, "You want very badly to walk into the sea.", dzFxA(-10, -3, -3, 0, [6]int{0, 0, 0, -3, -5, -3}), 0),
		},
		Cures: []dzCure{
			{Kind: "time", Pow: 2, Hint: "Three days on dry land, away from the water, and it lifts in the early stage."},
			{Kind: "herb", Prop: "calm", Pow: 1, Hint: "A calming brew holds the song off."},
			{Kind: "priest", Rank: 1, Pow: 2, Hint: "A priest can pray the song out of you."},
			{Kind: "shrine", Pow: 2, Hint: "A holy place away from the sea quiets it."},
			{Kind: "ward", Pow: 0, Hint: "An amulet against the deep holds it off."},
		}},
	{ID: "brain_wasting", Name: "Creeping wasting", Kind: "disease", Fee: 90,
		Lore: "Bad meat from the brain and spine of beasts that were already sick. It takes months and there is no cure.",
		Stages: []dzStage{
			dzS("Forgetful", 12, "You lose words and walk into doors.", dzFxA(0, 0, 0, 0, [6]int{0, 0, 0, -1, 0, 0}), 0),
			dzS("Stumbling", 14, "Your legs do not obey you well.", dzFxA(-5, -1, -2, 0, [6]int{0, -2, 0, -3, -2, 0}), 0),
			dzS("Wasting", 0, "You are a stranger in your own head.", dzFxA(-20, -3, -4, 0, [6]int{0, -5, 0, -6, -4, 0}), 2),
		},
		Cures: []dzCure{
			{Kind: "doctor", Max: 1, Pow: 1, Hint: "A physician can slow it a stage if it is found at once."},
			{Kind: "priest", Rank: 3, Pow: 1, Hint: "A high priest's prayer can turn it back a stage."},
			{Kind: "shrine", Pow: 1, Hint: "Some holy places are said to lift it a little."},
			{Kind: "ward", Pow: 0, Hint: "A master's amulet holds it where it is."},
		}},
	{ID: "gut_parasites", Name: "Parasites", Kind: "parasite", Natural: 5, Fee: 30,
		Lore: "Worms and flukes from raw meat and foul water.",
		Stages: []dzStage{
			{Name: "Hungry", Days: 5, Text: "You are hungry all the time and eat more than you should.", Hunger: 30},
			{Name: "Worms", Days: 9, Text: "You pass worms. You are thin and tired.", Fx: dzFxA(-6, 0, 0, 0, [6]int{0, 0, -1, 0, 0, 0}), Hunger: 60},
			{Name: "Wasting gut", Text: "Nothing stays in you. You are skin and bone.", Fx: dzFxA(-12, 0, 0, 0, [6]int{-2, 0, -3, 0, 0, 0}), Drain: 1, Hunger: 100},
		},
		Cures: []dzCure{
			{Kind: "herb", Prop: "antidote", Pow: 2, Hint: "A bitter antidote clears the gut of worms."},
			{Kind: "herb", Prop: "cleansing", Pow: 1, Hint: "A cleansing draught flushes them."},
			{Kind: "doctor", Pow: 2, Hint: "A physician has a purge for it."},
			{Kind: "priest", Rank: 1, Pow: 1, Hint: "A priest's blessing eases the gut."},
		}},
	{ID: "rad_sores", Name: "Radiation sores", Genres: "wasteland atompunk", Kind: "disease", Fee: 70,
		Lore: "The old fire leaves sores that do not close.",
		Stages: []dzStage{
			dzS("Sores", 5, "Small sores on the hands and face.", dzFxA(-3, 0, 0, 0, [6]int{0, 0, 0, 0, 0, -1}), 0),
			dzS("Weeping sores", 8, "Sores weep and the hair falls out.", dzFxA(-10, 0, 0, 0, [6]int{0, 0, -2, 0, 0, -2}), 0),
			dzS("Radiation burns", 0, "Your skin sloughs in sheets.", dzFxA(-15, 0, 0, 0, [6]int{-2, 0, -4, 0, 0, -3}), 1),
		},
		Cures: []dzCure{
			{Kind: "herb", Prop: "cleansing", Max: 2, Pow: 2, Hint: "A cleansing wash and a clean room heal the early sores."},
			{Kind: "herb", Prop: "healing", Pow: 1, Hint: "A healing salve closes the sores for a while."},
			{Kind: "doctor", Max: 2, Pow: 2, Hint: "A medic with a purifier can treat it early."},
			{Kind: "shrine", Pow: 1, Hint: "Old clinics and vaults still hold the machines that treat it."},
			{Kind: "ward", Pow: 0, Hint: "A ward-charm tuned by a master keeps the dose off."},
		}},
	{ID: "data_virus", Name: "Neural virus", Genres: "cyberpunk", Kind: "disease", Spread: 1, Fee: 80,
		Lore: "A hostile program running in your implants and your head.",
		Stages: []dzStage{
			dzS("Glitching", 4, "Your vision stutters. Numbers swim in your eyes.", dzFxA(0, 0, -1, 0, [6]int{0, 0, 0, -1, 0, 0}), 0),
			dzS("Corrupted", 7, "Your hands act a half-second late.", dzFxA(0, -2, -2, 0, [6]int{0, 0, 0, -2, -2, 0}), 0),
			dzS("Overwritten", 0, "Something else is in the driver's seat.", dzFxA(-12, -3, -4, 0, [6]int{0, 0, 0, -4, -4, -3}), 0),
		},
		Cures: []dzCure{
			{Kind: "doctor", Max: 2, Pow: 2, Hint: "A ripperdoc can flush the firmware while it is young."},
			{Kind: "shrine", Pow: 2, Hint: "A clean server hub can overwrite the infection."},
			{Kind: "ward", Pow: 0, Hint: "A firewall charm from a master holds it."},
		}},
	{ID: "spore_bloom", Name: "Spore bloom", Genres: "biopunk", Kind: "disease", Spread: 2, Fee: 70,
		Lore: "A living strain that treats the body as a garden.",
		Stages: []dzStage{
			dzS("Bloom-itch", 4, "A rash that spreads in a ring.", dzFxA(0, 0, 0, 0, [6]int{}), 0),
			dzS("Mycelial blush", 7, "Threads show under the skin.", dzFxA(-8, 0, -1, 0, [6]int{0, 0, 0, 0, -2, 0}), 0),
			dzS("Fruiting body", 0, "You are mostly garden.", dzFxA(-18, -1, -2, 0, [6]int{0, 0, 0, -3, -4, -3}), 1),
		},
		Cures: []dzCure{
			{Kind: "herb", Prop: "cleansing", Max: 2, Pow: 2, Hint: "A cleansing wash and burning kill the young bloom."},
			{Kind: "doctor", Max: 2, Pow: 2, Hint: "A splicer-doctor can cut and cauterise it early."},
			{Kind: "ward", Pow: 0, Hint: "A master's charm keeps the strain asleep."},
		}},
	{ID: "ember_fever", Name: "Ember fever", Kind: "disease", Natural: 8, Fee: 40,
		Lore: "Breathed fire and hot ash leave a fever in the lungs.",
		Stages: []dzStage{
			{Name: "Warm", Days: 3, Text: "You run hot and thirsty.", Fx: dzFxA(-3, 0, 0, 0, [6]int{}), Thirst: 15},
			{Name: "Cinder lung", Days: 7, Text: "Every breath smells of smoke.", Fx: dzFxA(-10, 0, 0, 0, [6]int{0, 0, -2, 0, 0, 0}), Thirst: 40},
			{Name: "Burning", Text: "You burn from the inside.", Fx: dzFxA(-18, 0, 0, 0, [6]int{-2, 0, -4, 0, 0, 0}), Drain: 1, Thirst: 80},
		},
		Cures: []dzCure{
			{Kind: "herb", Prop: "frostres", Pow: 2, Hint: "A frost-ward brew cools the fever."},
			{Kind: "herb", Prop: "cleansing", Pow: 1, Hint: "A cleansing draught washes the ash out."},
			{Kind: "doctor", Max: 2, Pow: 2, Hint: "A physician will pack you in cold cloth."},
			{Kind: "ward", Pow: 0, Hint: "A cool-stone amulet keeps it down."},
		}},
	{ID: "venom_sick", Name: "Lingering venom", Kind: "poison", Fee: 50,
		Lore: "Some bites leave more than a wound.",
		Stages: []dzStage{
			dzS("Numb limb", 3, "The bitten limb is numb.", dzFxA(-5, 0, -1, 0, [6]int{}), 0),
			dzS("Creeping numbness", 6, "The numbness climbs.", dzFxA(-12, 0, -2, 0, [6]int{0, -2, 0, 0, 0, 0}), 0),
			dzS("Paralytic fits", 0, "Fits take your legs from under you.", dzFxA(-25, -3, -4, 0, [6]int{0, -4, 0, 0, 0, 0}), 1),
		},
		Cures: []dzCure{
			{Kind: "herb", Prop: "antidote", Pow: 2, Hint: "A true antidote draws the venom out."},
			{Kind: "doctor", Max: 2, Pow: 2, Hint: "A physician can cup and bleed it early."},
			{Kind: "priest", Rank: 1, Pow: 1, Hint: "A priest's blessing eases it."},
		}},
	{ID: "frost_rot", Name: "Frost-rot", Kind: "disease", Fee: 45,
		Lore: "Cold that gets into the toes and fingers and stays.",
		Stages: []dzStage{
			dzS("Numb toes", 4, "Your toes have no feeling.", dzFxA(-6, 0, 0, 0, [6]int{}), 0),
			dzS("Black toes", 7, "The tips turn black.", dzFxA(-15, -2, 0, 0, [6]int{0, -2, 0, 0, 0, 0}), 0),
			dzS("Frost-rot", 0, "Dead flesh must come off or it spreads.", dzFxA(-30, -4, 0, 0, [6]int{-2, -4, 0, 0, 0, 0}), 1),
		},
		Cures: []dzCure{
			{Kind: "herb", Prop: "fireres", Pow: 1, Hint: "A warming brew brings the blood back."},
			{Kind: "doctor", Max: 2, Pow: 2, Hint: "A physician can warm and trim it early."},
			{Kind: "shrine", Pow: 1, Hint: "Hot springs and holy fires are said to help."},
			{Kind: "priest", Rank: 1, Pow: 1, Hint: "A priest's blessing eases the pain."},
		}},
	{ID: "hollow_soul", Name: "Soul-hollow", Kind: "curse", Skin: map[string]string{"cyberpunk": "Ghost-drain"},
		Lore: "Something fed on you and took more than blood.",
		Stages: []dzStage{
			dzS("Numb of heart", 6, "Nothing delights you.", dzFxA(0, 0, 0, 0, [6]int{0, 0, 0, 0, -1, -1}), 0),
			dzS("Hollow", 10, "You have to remember to feel.", dzFxA(0, 0, -1, 0, [6]int{0, 0, 0, -1, -3, -3}), 0),
			dzS("Empty", 0, "A hollow shell that walks and swings a sword.", dzFxA(-10, 0, -2, 0, [6]int{0, 0, 0, -3, -5, -4}), 1),
		},
		Cures: []dzCure{
			{Kind: "kill", Pow: 2, Hint: "Kill the thing that fed on you and what it took comes back."},
			{Kind: "priest", Rank: 2, Pow: 2, Hint: "A priest of rank can call the soul home."},
			{Kind: "shrine", Pow: 2, Hint: "A holy place remembers what you were."},
			{Kind: "ward", Pow: 0, Hint: "A master's amulet keeps what is left."},
		}},
}

// carrier tables: substring of the attacker's template or name -> affliction and percent chance per hit
type dzCarrier struct {
	Sub string
	ID  string
	P   int
}

var dzCarriers = []dzCarrier{
	{"werewolf", "lycanthropy", 40}, {"werebear", "lycanthropy", 40}, {"werecat", "lycanthropy", 40}, {"wererat", "lycanthropy", 40}, {"lycan", "lycanthropy", 40}, {"wolfman", "lycanthropy", 35},
	{"vampire", "vampiric_taint", 30}, {"nosferatu", "vampiric_taint", 30}, {"strigoi", "vampiric_taint", 30}, {"bloodbat", "vampiric_taint", 15},
	{"ghost", "ghost_sickness", 25}, {"wraith", "ghost_sickness", 25}, {"specter", "ghost_sickness", 25}, {"spectre", "ghost_sickness", 25}, {"phantom", "ghost_sickness", 25}, {"banshee", "ghost_sickness", 30}, {"shade", "ghost_sickness", 20}, {"wight", "ghost_sickness", 25}, {"haunt", "ghost_sickness", 20},
	{"mummy", "mummy_curse", 30},
	{"zombie", "rot_blight", 25}, {"ghoul", "rot_blight", 25}, {"revenant", "rot_blight", 20}, {"carrion", "rot_blight", 12}, {"plague", "rot_blight", 20},
	{"spore", "fungal_spores", 30}, {"fungal", "fungal_spores", 25}, {"myconid", "fungal_spores", 30}, {"mushroom", "fungal_spores", 20}, {"shrieker", "fungal_spores", 15}, {"shambler", "fungal_spores", 25},
	{"witch", "witch_hex", 25}, {"hag", "witch_hex", 25}, {"warlock", "witch_hex", 20}, {"hexer", "witch_hex", 30},
	{"kraken", "sea_madness", 20}, {"siren", "sea_madness", 25}, {"drowned", "sea_madness", 20}, {"merrow", "sea_madness", 15}, {"leviathan", "sea_madness", 15},
	{"mosquito", "marsh_fever", 35}, {"gnat", "marsh_fever", 25}, {"midge", "marsh_fever", 25}, {"tsetse", "marsh_fever", 35},
	{"leech", "gut_parasites", 20}, {"lamprey", "gut_parasites", 15}, {"tick", "gut_parasites", 20},
	{"viper", "venom_sick", 25}, {"adder", "venom_sick", 25}, {"cobra", "venom_sick", 30}, {"serpent", "venom_sick", 20}, {"scorpion", "venom_sick", 30}, {"asp", "venom_sick", 25},
	{"ember", "ember_fever", 20}, {"cinder", "ember_fever", 20}, {"magma", "ember_fever", 15}, {"salamander", "ember_fever", 20}, {"hellhound", "ember_fever", 25},
	{"yeti", "frost_rot", 15}, {"wendigo", "frost_rot", 20}, {"frost", "frost_rot", 10},
	{"lich", "hollow_soul", 25}, {"soul", "hollow_soul", 20}, {"devourer", "hollow_soul", 20}, {"syphon", "hollow_soul", 20},
	{"drone", "data_virus", 20}, {"daemon", "data_virus", 30}, {"netrunner", "data_virus", 30}, {"cyber", "data_virus", 15}, {"glitch", "data_virus", 25}, {"hacker", "data_virus", 25},
	{"chimera", "spore_bloom", 20}, {"splice", "spore_bloom", 20}, {"vat", "spore_bloom", 25},
	{"irradiated", "rad_sores", 25}, {"glowing", "rad_sores", 25}, {"mutant", "rad_sores", 10},
}

var dzVague = map[string]string{"disease": "a creeping sickness", "curse": "a curse of some kind", "bite": "something that came with the bite", "taint": "a slow sickness from something you carried", "parasite": "something living in your gut", "poison": "a lingering poison"}

var dzWardRoles = map[string]bool{"seer": true, "scholar": true, "priest": true}

var dzHerbalistRoles = map[string]bool{"herbalist": true, "apothecary": true, "gardener": true}
