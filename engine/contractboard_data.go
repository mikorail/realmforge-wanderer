package engine

// Contract board data (prefix jb): kinds, weights, base pay and the text grammars. Placeholders in
// the text lines are filled by jbFill: {town} {foe} {kind} {where} {dir} {n} {site} {dest} {who}
// {item} {days} {owner} {why}.

type jbKindDef struct {
	ID     string
	Name   string
	Weight int
	Base   int // base pay in base currency units at level 0
	PerLvl int // extra per level
}

var jbKinds = []jbKindDef{
	{"hunt", "Hunt", 22, 40, 18},
	{"clear", "Clear", 16, 70, 26},
	{"escort", "Escort", 14, 20, 8},
	{"deliver", "Deliver", 16, 18, 6},
	{"rescue", "Rescue", 8, 90, 28},
	{"recover", "Recover", 10, 50, 20},
	{"patrol", "Patrol", 8, 40, 14},
	{"guard", "Watch", 6, 30, 10},
}

func jbKindOf(id string) *jbKindDef {
	for i := range jbKinds {
		if jbKinds[i].ID == id {
			return &jbKinds[i]
		}
	}
	return &jbKinds[0]
}

// jbKmDay is the pace deadlines assume: kilometres a day on foot.
const jbKmDay = 22.0

var jbBeastAdj = []string{"Grey", "Red-eyed", "Limping", "Scarred", "Old", "Hollow-flanked", "Black-muzzled", "Lone", "Hungry", "Night-prowling", "Bold", "Cunning"}

var jbHuntTitles = []string{
	"Kill {foe}",
	"Bounty on {foe}",
	"End the terror of {where}",
	"Hunter wanted: {foe}",
}

var jbHuntText = []string{
	"{foe}, a {kind}, has been taking stock and drovers near {where}, about {n} parasangs {dir} of {town}. Bring it down.",
	"Herders near {where} ({n} parasangs {dir} of {town}) will not take the flocks out while {foe} lives. It is a {kind}. The board pays for its death.",
	"Carters lost two wagons to {foe} on the way through {where}. A {kind}, they say, {n} parasangs {dir} of {town}. Kill it.",
}

var jbRumourHuntText = []string{
	"Word from the road has it that something is at large near {where}, {n} parasangs {dir} of {town}: {foe}, a {kind}. The tale is unconfirmed. The board will pay if you find it and end it.",
}

var jbClearTitles = []string{
	"Clear out {site}",
	"Smoke out {site}",
	"Drive the vermin from {site}",
}

var jbClearText = []string{
	"Something has made a lair of {site}, {n} parasangs {dir} of {town}. Kill {need} of what lives there so the roads stay open.",
	"Folk near {site} hear noises at night. It lies {n} parasangs {dir} of {town}. The board wants {need} of its tenants dead.",
	"Travellers shun {site}, {n} parasangs {dir} of {town}. Go in and put down {need} of the things that haunt it.",
}

var jbEscortWhy = []string{
	"a cloth merchant with a locked chest",
	"a pilgrim bound for a shrine",
	"a scribe carrying copied records",
	"a physician called to a sickbed",
	"a widow going to her daughter's wedding",
	"a surveyor with chain and rod",
	"a young apprentice sent to a master",
	"a tax clerk with a satchel of coin",
	"a bard who has missed their caravan",
	"an old soldier going home",
}

var jbEscortText = []string{
	"{who} is {why}, bound for {dest}, {n} parasangs {dir} of {town}, and cannot go alone. Walk with them, keep them alive, and reach {dest} within {days} days.",
	"The roads are not safe. {who}, {why}, needs a guard to {dest} ({n} parasangs {dir}). Deliver them there within {days} days.",
}

var jbParcels = []string{
	"sealed letter case", "ledger of accounts", "bolt of dyed cloth", "box of medicines", "bundle of tax rolls",
	"folded treaty copy", "locked document tube", "crate of seed stock", "wrapped reliquary", "bundle of contracts",
}

var jbDeliverText = []string{
	"Carry the {item} to {who} in {dest}, {n} parasangs {dir} of {town}. It must be there within {days} days. Do not open it.",
	"{who} of {dest} is waiting on a {item}. Take it there ({n} parasangs {dir}) within {days} days and bring nothing back but their thanks.",
}

var jbCaptiveWho = []string{
	"a surveyor", "a young shepherd", "a courier", "a tinker", "a pilgrim", "a ferryman's daughter", "a scribe", "a hunter",
}

var jbRescueText = []string{
	"{who}, {owner}, was dragged off to {site}, {n} parasangs {dir} of {town}. Get them out alive and walk them home.",
	"{owner} begs the board for help: {who} is held in {site}, {n} parasangs {dir} of {town}. Cut them loose and bring them back.",
}

var jbLost = []string{
	"strongbox", "signet ring", "ledger", "family sword", "reliquary", "satchel of letters", "silver lamp", "survey case",
}

var jbRecoverText = []string{
	"{owner} lost a {item} when the road was raided and swears it lies somewhere in {site}, {n} parasangs {dir} of {town}. Find it and bring it to the board.",
	"A {item} belonging to {owner} was carried into {site} ({n} parasangs {dir} of {town}). Recover it and the board will see you paid.",
}

var jbPatrolText = []string{
	"The roads around {town} have been bad. Walk the country within two parasangs and put down {need} brigands or beasts that trouble travellers. Report within {days} days.",
	"The watch is stretched thin. The board wants {need} dangers cleared from the lanes about {town}. Take {days} days if you must.",
}

var jbGuardText = []string{
	"The night watch of {town} is short of hands. Stand a watch inside the town between dusk and dawn, {need} minutes in all, within {days} days.",
	"Thieves have been at the market stalls after dark. Walk {town} by night for {need} minutes and keep the peace. {days} days to do it.",
}

var jbPosters = []string{
	"the reeve", "the clerk of the board", "the town steward", "the watch captain", "the guildmaster of carters",
}

var jbOutcomeNews = map[string]string{
	"hunt":    "%s has put down a beast that troubled %s.",
	"clear":   "%s cleared a lair for the contract board of %s.",
	"escort":  "%s brought a traveller safely out of %s.",
	"deliver": "%s carried a parcel out of %s.",
	"rescue":  "%s brought a captive home to %s.",
	"recover": "%s recovered a lost item for a client in %s.",
	"patrol":  "%s walked the lanes for the board of %s.",
	"guard":   "%s stood a night watch at %s.",
}
