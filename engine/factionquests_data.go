package engine

// Faction quest chain templates (prefix fq). Text only: factionquests.go fills the placeholders from real
// world data when a chain is built. Placeholders: {rep} the faction's representative, {fac} the faction,
// {rival} its rival, {realm} the home realm, {town} the representative's town, {town2} this step's
// destination town, {person} this step's contact, {item}, {foe}, {site}, {dir}, {n} (parasangs), {days},
// {enemy} a realm at war with the home realm, {contact} the last contact met before this step.
//
// Step kinds: meet (talk to {person} in {town2}), deliver ({item} to {person} in {town2}), scout (arrive at a
// place in the wild), kill ({foe} at a place in the wild), fetch ({item} from {site}), rescue ({person} held at
// {site}, walked to the representative's town), hold (stay alive N days; ambushers may come), parley (a choice
// at the representative), avenge (a wronged contact or hired knife comes for you; only when the Need flag was
// set by an earlier choice). Choices sit on a step and are shown when the step is handed in.

type fqOptT struct {
	Text, Win, Lose, Flag, Chk string
	DC, Cost, Fac, Riv, Gold   int
	Hostile                    bool
}

type fqChT struct {
	Prompt string
	Opts   []fqOptT
}

type fqStepT struct {
	Kind, Brief, Done string
	Days              int
	Ch                *fqChT
	Need              string // flag that must be set ("!flag" = must not be set)
	If, Alt           string // when flag If is set the brief reads Alt
	Amb               bool   // hold: ambushers may find you
}

type fqTplT struct {
	ID, Kinds, Title, Hook string
	Steps                  []fqStepT
}

func fqs(kind, brief, done string) fqStepT { return fqStepT{Kind: kind, Brief: brief, Done: done} }
func (s fqStepT) D(n int) fqStepT        { s.Days = n; return s }
func (s fqStepT) N(f string) fqStepT     { s.Need = f; return s }
func (s fqStepT) A() fqStepT             { s.Amb = true; return s }
func (s fqStepT) I(f, alt string) fqStepT {
	s.If, s.Alt = f, alt
	return s
}
func (s fqStepT) C(prompt string, opts ...fqOptT) fqStepT {
	s.Ch = &fqChT{Prompt: prompt, Opts: opts}
	return s
}

// fo is a plain choice: text, outcome, standing with the faction, with its rival, flag.
func fqo(text, win string, fac, riv int, flag string) fqOptT {
	return fqOptT{Text: text, Win: win, Fac: fac, Riv: riv, Flag: flag}
}
func (o fqOptT) G(n int) fqOptT       { o.Gold = n; return o }
func (o fqOptT) P(n int) fqOptT       { o.Cost = n; return o }
func (o fqOptT) H() fqOptT            { o.Hostile = true; return o }
func (o fqOptT) K(chk string, dc int, lose string) fqOptT {
	o.Chk, o.DC, o.Lose = chk, dc, lose
	return o
}

var fqTemplates = []fqTplT{
	// ------------------------------------------------------------------ realms
	{ID: "crown_writ", Kinds: "realm", Title: "Sealed Writs for {fac}",
		Hook: "{rep} of {fac} wants hands that belong to no family. A sealed writ must reach {person} in {town2}, and a raider called {foe} is bleeding the roads dry.",
		Steps: []fqStepT{
			fqs("deliver", "Carry the sealed writ to {person} in {town2}, {n} parasangs {dir}. Do not break the seal.", "{person} has the writ and a cold look for whoever sent it.").D(10).
				C("{person} reads the writ and asks what the roads are like.",
					fqo("Tell them the roads are quiet.", "\"Good. {fac} likes good news.\"", 2, 0, "honest"),
					fqo("Tell them the roads are bad and the crown should pay more.", "{person} winces and slips a coin into your hand. {rep} will hear of it.", -1, 0, "greedy").G(25)),
			fqs("kill", "{foe}, who robs the roads in the name of the old nobility, lairs near ({n} parasangs {dir}). End it and bring word to {rep} in {town}.", "{foe} is dead and the camp scattered.").D(12).
				C("{rep} asks what you took from the camp.",
					fqo("Hand over every coin to {fac}.", "\"Honest hands. I will remember it.\"", 4, 0, ""),
					fqo("Pardon the camp followers and give them the coin to get home.", "\"Merciful. Some will call it soft.\"", 2, 0, "mercy"),
					fqo("Keep the loot and say there was none.", "{rep} frowns but writes nothing down.", -4, 0, "skimmed").G(80)),
			fqs("hold", "Someone counted the missing coin. Keep out of sight for {days} days and let the matter cool.", "Nobody came for you.").N("skimmed").D(3).A(),
			fqs("parley", "{rep} names the price of your service to {fac}.", "The matter is closed.").
				C("{rep} offers you a choice.",
					fqo("Take a place in the service of {fac}.", "\"Then you are ours. Wear it well.\"", 6, 0, "sworn"),
					fqo("Ask for gold, not a post.", "A heavy purse changes hands.", 1, 0, "").G(140),
					fqo("Sell a copy of the writ's contents to {rival}'s envoys.", "{rival}'s man pays well and says nothing about it. You hope that holds.", -8, 6, "betray").G(160).H()),
			fqs("avenge", "{contact}, who lost a post when the writ leaked, has found you. Keep your head.", "{contact} will trouble nobody now.").N("betray"),
		}},
	{ID: "crown_seal", Kinds: "realm", Title: "The Dead Ruler's Seal",
		Hook: "{rep} of {fac} wants {item} back from {site}, before it finds a buyer in {rival}.",
		Steps: []fqStepT{
			fqs("fetch", "Bring {item} out of {site}, {n} parasangs {dir}, to {rep} in {town}.", "You have {item}.").D(14).
				C("{rep} turns the seal over in their hands. Who is it for?",
					fqo("Give it to {rep}.", "\"{fac} remembers who was honest.\"", 5, 0, "returned"),
					fqo("Offer it to {rival} for gold.", "{rival}'s agents are delighted. {rep} will learn of it.", -6, 5, "sold").G(170).H(),
					fqo("Keep it as insurance and say you found nothing.", "{rep} does not believe you, and cannot prove it.", -3, 0, "kept")),
			fqs("hold", "Word is out that someone took the seal. Keep your head down for {days} days.", "The talk has moved on.").D(3).A(),
			fqs("meet", "A scribe, {person} in {town2}, wants to question you about the seal.", "The scribe is satisfied.").
				C("{person} asks where the seal was lying.",
					fqo("Tell the whole truth.", "{person} writes it all down and thanks you.", 2, 0, "").K("lore", 9, "{person} catches a slip and notes it with a sour mouth."),
					fqo("Lie about the place.", "{person} nods. The lie holds, for now.", -1, 0, "lied").K("bluff", 11, "{person} knows the lie at once. {rep} will hear.")),
			fqs("avenge", "Hired knives from {rival} want what you sold. One of them is {foe}.", "The knives are broken.").N("sold"),
		}},
	{ID: "crown_border", Kinds: "realm", Title: "Eyes on the Frontier",
		Hook: "{fac} watches its border with {enemy}. {rep} needs someone no one knows to look, listen, and bring back the truth.",
		Steps: []fqStepT{
			fqs("scout", "Ride to the border country ({n} parasangs {dir}) and watch for {enemy}'s soldiers.", "You mark their camp and their road.").D(10),
			fqs("meet", "An informant, {person}, waits in {town2}. They want money for what they know.", "{person} has told you what they know.").
				C("{person} names a price and does not look at you.",
					fqo("Pay what they ask.", "{person} talks fast and honestly.", 2, 0, "paid").P(40),
					fqo("Lean on them until they talk.", "{person} talks. They will not forget the hand on their collar.", 0, 0, "threat").K("intimidate", 10, "{person} spits and says nothing; a guard has noticed."),
					fqo("Let them go and say nothing of them to {rep}.", "{person} slips off, grateful.", -2, 0, "spared")),
			fqs("kill", "The {enemy} scout-captain {foe} camps near ({n} parasangs {dir}). Take them before they take a village.", "{foe} will scout no more.").D(10),
			fqs("parley", "Report to {rep} in {town}.", "The report is given.").
				C("What does {rep} hear?",
					fqo("The plain truth.", "\"Plain, and useful. Good.\"", 4, 0, ""),
					fqo("The truth, a little larger than it was.", "\"Alarming. Well, better that than asleep.\"", 3, 0, "boast").G(40),
					fqo("Sell the routes to {rival} and tell {rep} nothing.", "{rival} pays handsomely for a map of your own people's patrols.", -9, 7, "betray").G(200).H()),
		}},
	{ID: "crown_champion", Kinds: "realm", Title: "The Champion's Challenge",
		Hook: "A foreign champion has been mocking {fac} in {town2}. {rep} wants {fac} answered without starting a war.",
		Steps: []fqStepT{
			fqs("meet", "Find the champion {person} in {town2}, {n} parasangs {dir}, and answer them for {fac}.", "{person} has been answered.").
				C("{person} laughs and names the terms.",
					fqo("Fight to first blood.", "You win, and {person} salutes the blow.", 4, 0, "dueled").K("acrobatics", 10, "{person} wins, and the hall is quiet when you leave."),
					fqo("Outdrink them at the long table.", "{person} slides under the bench, and the hall roars for {fac}.", 3, 0, "drank").K("endurance", 10, "You lose the bout and the bet."),
					fqo("Yield with grace and ask for a truce.", "{person} is surprised, then pleased. {rep} will be less so.", 0, 0, "yield")),
			fqs("kill", "A pack of {foe}'s hirelings has been thinning the road to {town2}. Clear it ({n} parasangs {dir}).", "The road is open again.").D(12),
			fqs("parley", "Bring word of the challenge to {rep} in {town}.", "The story is told.").
				C("{rep} asks how the court will hear it.",
					fqo("As a victory for {fac}.", "\"Let them sing it.\"", 4, 0, ""),
					fqo("As it happened.", "\"Honest, and dull. Fine.\"", 2, 0, "")),
		}},

	// ------------------------------------------------------------------ faiths
	{ID: "faith_pilgrim", Kinds: "faith", Title: "The Long Pilgrimage",
		Hook: "{rep} of {fac} asks for a pilgrim who can carry an offering and keep a vigil, as the old rites require.",
		Steps: []fqStepT{
			fqs("meet", "Visit the shrine-keeper {person} in {town2}, {n} parasangs {dir}, and receive the blessing of the road.", "{person} has blessed you for the way.").
				C("{person} asks why you walk.",
					fqo("For the faith.", "\"Then walk well.\"", 2, 0, ""),
					fqo("For myself.", "\"Honest. The road will answer you.\"", 1, 0, "")),
			fqs("deliver", "Carry {item} to {person} in {town2} as an offering for {fac}.", "The offering is made.").D(12).
				C("{person} weighs the offering in their hands. It seems light.",
					fqo("Leave all of it.", "\"A full gift. You are remembered.\"", 4, 0, "gave"),
					fqo("Admit you kept half for the road.", "\"Honest, at least.\"", -1, 0, "skimmed").G(40)),
			fqs("hold", "Keep the vigil. Stay out of towns and alive for {days} days; the faithful of {rival} dislike pilgrims.", "The vigil is kept.").D(2).A(),
			fqs("parley", "Return to {rep} in {town} and take, or refuse, the vow of service.", "The pilgrimage is complete.").
				C("{rep} offers the vow.",
					fqo("Swear it.", "\"Then you are one of ours.\"", 6, -2, "oath"),
					fqo("Decline, but give thanks.", "\"The door stays open.\"", 1, 0, "")),
		}},
	{ID: "faith_hymnal", Kinds: "faith", Title: "The Heretic's Hymnal",
		Hook: "{rep} of {fac} says a hymnal that contradicts the true doctrine lies in {site}. It must not be allowed to spread.",
		Steps: []fqStepT{
			fqs("fetch", "Take {item} out of {site}, {n} parasangs {dir}, and bring it to {rep} in {town}.", "You have {item}.").D(14).
				C("{rep} holds the book at arm's length.",
					fqo("Burn it as asked.", "\"The faith is cleaner for it.\"", 5, -3, "burn"),
					fqo("Copy it first, in secret.", "You keep the knowledge and the secret.", -3, 3, "copy").K("lore", 10, "{rep} sees the ink on your hands and is quiet."),
					fqo("Sell it to the faithful of {rival}.", "They pay with gladness, and they will use it.", -7, 6, "sold").G(150).H()),
			fqs("meet", "A heretic, {person}, wants to hear what you read. They wait in {town2}.", "You have spoken with the heretic.").N("copy").
				C("{person} asks you to read aloud.",
					fqo("Read it.", "{person} weeps, and gives you a little coin.", -2, 4, "read").G(30),
					fqo("Refuse, and warn them away.", "{person} nods. They will not wait long.", 2, 0, "")),
			fqs("avenge", "Zealots of {rival} want the book you sold. One of them is {foe}.", "The zealots are scattered.").N("sold"),
		}},
	{ID: "faith_hermit", Kinds: "faith", Title: "The Captive Priest",
		Hook: "A priest of {fac} was taken from the road, and {rep} fears the kidnappers will ask for more than gold.",
		Steps: []fqStepT{
			fqs("kill", "The band led by {foe} holds the captive near ({n} parasangs {dir}). Break it.", "The band is broken.").D(10),
			fqs("rescue", "Free {person} from {site} and walk them to {rep} in {town}.", "{person} is safe.").D(12),
			fqs("parley", "{person} is back. {rep} asks what should be done with the kidnappers' loot.", "The matter is closed.").
				C("{rep} waits for your answer.",
					fqo("Give the loot to the shrine.", "\"The shrine will pray for you.\"", 5, 0, ""),
					fqo("Return it to the people it was taken from.", "\"Just. The faith is not poorer for it.\"", 3, 0, "just"),
					fqo("Ask for a share as payment.", "\"A fair share, then.\"", 1, 0, "").G(120)),
		}},

	// ------------------------------------------------------------------ the realm's army and watch
	{ID: "watch_rounds", Kinds: "watch", Title: "Night Rounds",
		Hook: "{rep} of {fac} is short of sergeants. Walk the rounds with the watch, then see to a gang boss named {foe}.",
		Steps: []fqStepT{
			fqs("hold", "Walk the rounds for {days} days and keep your eyes open. A watch is only as good as its strangers.", "The rounds are done.").D(2).A(),
			fqs("meet", "A witness, {person}, saw {foe}'s crew at work in {town2}. They are afraid to talk.", "{person} gave their account.").
				C("{person} will talk, but only on terms.",
					fqo("Promise them protection and keep it.", "{person} talks and sleeps better.", 3, -1, "protected"),
					fqo("Pay them for it.", "{person} talks for coin and no more.", 1, 0, "paid").P(30),
					fqo("Lean on them.", "{person} talks. They will remember who frightened them.", 1, 0, "threat").K("intimidate", 10, "{person} clams up, and the watch will hear of your clumsiness.")),
			fqs("kill", "Take {foe} and the crew near ({n} parasangs {dir}).", "{foe} is down.").D(10),
			fqs("parley", "Report to {rep} in {town}.", "The report is filed.").
				C("{rep} asks if the witness should be named.",
					fqo("Name the witness and take the credit for the watch.", "\"Done, and quick.\"", 3, 0, "named"),
					fqo("Keep the witness out of it.", "\"Careful. Good.\"", 4, 0, ""),
					fqo("Tell {rival}'s people who talked.", "Word travels. {person} is not safe now.", -8, 6, "betray").G(100).H()),
			fqs("avenge", "{foe}'s old friends want a word with the sergeant who told. That is you.", "The friends are quiet.").N("named"),
		}},
	{ID: "watch_cache", Kinds: "watch", Title: "The Smugglers' Cache",
		Hook: "{rep} of {fac} has found out where a smugglers' cache lies. {item} is the proof, and the proof is in {site}.",
		Steps: []fqStepT{
			fqs("fetch", "Bring {item} from {site}, {n} parasangs {dir}, to {rep} in {town}.", "You hold the proof.").D(12),
			fqs("meet", "A fixer, {person} in {town2}, has heard you took it and would like to talk.", "You have heard the fixer out.").
				C("{person} offers a purse for the evidence.",
					fqo("Refuse and hand the proof to {rep}.", "{person} shrugs. The watch is pleased.", 4, -3, "honest"),
					fqo("Take the purse and lose the proof.", "{person} smiles. You have friends in the dark now.", -6, 5, "bribed").G(200).H(),
					fqo("Take the purse and hand over the proof anyway.", "{person} swears at you, and you have the purse and the credit.", 2, -4, "double").G(120).K("bluff", 11, "{person} sees through it. The purse is gone, and so is their patience.")),
			fqs("kill", "The smuggler chief {foe} still runs the cache. Clear them near ({n} parasangs {dir}).", "The chief is gone.").D(12).N("!bribed"),
			fqs("parley", "Report to {rep} in {town}.", "The watch is satisfied.").
				C("{rep} asks if there was more.",
					fqo("Say there was nothing more.", "\"A clean report.\"", 2, 0, ""),
					fqo("Give them names from the fixer's talk.", "\"That helps.\"", 3, -2, "names")),
			fqs("avenge", "{contact}, who bought you, wants either their money or your blood. They sent {foe}.", "{foe} is dealt with.").N("bribed"),
		}},

	// ------------------------------------------------------------------ merchant houses and leagues
	{ID: "house_caravan", Kinds: "house", Title: "The Lost Caravan",
		Hook: "A caravan of {fac} has vanished. {rep} wants the cargo found, and wants whoever took it to answer for it.",
		Steps: []fqStepT{
			fqs("scout", "Search the road ({n} parasangs {dir}) for signs of the caravan.", "You find the wreck of the caravan.").D(10),
			fqs("fetch", "Recover {item} from {site}, where the raiders took it, and bring it to {rep} in {town}.", "You hold the cargo.").D(12),
			fqs("kill", "The raiders are led by {foe}, who camps near ({n} parasangs {dir}).", "{foe} is dead.").D(12),
			fqs("parley", "Bring the goods to {rep} in {town}.", "The ledger is closed.").
				C("{rep} counts the cargo and looks at you.",
					fqo("Hand over all of it.", "\"The house does not forget an honest hand.\"", 5, 0, ""),
					fqo("Skim a fair-looking share.", "\"The count is a little short. I will not ask.\"", -2, 0, "skimmed").G(100),
					fqo("Sell part of it to {rival}'s factor.", "The factor pays and smiles. The house will notice.", -6, 5, "betray").G(180).H()),
			fqs("avenge", "{foe}'s partner is out for the cargo you sold. Mind the road.", "The partner is stopped.").N("betray"),
		}},
	{ID: "house_debt", Kinds: "house", Title: "A Debt Called In",
		Hook: "{fac} is owed a great deal by {person} in {town2}. {rep} would like the debt settled before the whole market hears of it.",
		Steps: []fqStepT{
			fqs("meet", "Find {person} in {town2}, {n} parasangs {dir}, and settle the debt.", "{person} has answered you.").
				C("{person} cannot pay in coin.",
					fqo("Forgive part of the debt.", "{person} is grateful. {rep} will be less so.", -2, 0, "forgave"),
					fqo("Demand it all, with a hand on the sword.", "{person} pays, bitterly.", 3, 0, "squeezed").G(90).K("intimidate", 10, "{person} has friends, and you leave without a coin."),
					fqo("Take goods in kind.", "{person} hands you a crate and a sour look.", 2, 0, "goods").G(40)),
			fqs("deliver", "Carry the receipt, {item}, to {person} in {town2}, so the matter is on the record.", "The receipt is filed.").D(10),
			fqs("hold", "Let the market settle for {days} days. The rival house is watching.", "The market is quiet.").D(3).A(),
			fqs("parley", "Report to {rep} in {town}.", "The books are balanced.").
				C("{rep} asks what to do with the debtor.",
					fqo("Let it lie.", "\"Peace is cheap, mostly.\"", 2, 0, ""),
					fqo("Break them in the market.", "\"Harsh. Effective.\"", 3, 0, "ruin"),
					fqo("Warn {rival}'s factor that the debtor is weak.", "{rival} moves in. The debtor is finished.", -6, 5, "betray").G(120).H()),
			fqs("avenge", "The debtor's kin hired {foe} to find whoever sold them out.", "{foe} is stopped.").N("betray"),
		}},
	{ID: "house_ledger", Kinds: "house", Title: "The Rival's Ledger",
		Hook: "{rep} suspects {rival} of cooking its books. {fac} would like proof, quietly.",
		Steps: []fqStepT{
			fqs("meet", "A clerk of {rival}, {person} in {town2}, may sell you a look at the books.", "You have spoken with the clerk.").
				C("{person} wants something for it.",
					fqo("Pay them.", "{person} describes the books in a whisper.", 2, 0, "paid").P(50),
					fqo("Charm it out of them.", "{person} lets slip what you need.", 2, 0, "charm").K("persuasion", 10, "{person} is polite and says nothing."),
					fqo("Threaten to expose their habits.", "{person} gives way with a thin smile.", 1, 0, "threat").K("intimidate", 11, "{person} calls a guard.")),
			fqs("fetch", "The books themselves lie at {site}, {n} parasangs {dir}. Bring {item} to {rep}.", "You hold the ledger.").D(12),
			fqs("parley", "Bring the ledger to {rep} in {town}.", "The ledger is delivered.").
				C("{rep} reads and smiles.",
					fqo("Give them all of it.", "\"{rival} is going to have a bad season.\"", 5, -4, ""),
					fqo("Give them most of it and keep a page.", "\"I will not count the pages.\"", 3, -2, "kept").G(60),
					fqo("Sell the ledger back to {rival}.", "{rival} pays double and swears you to silence.", -7, 6, "betray").G(220).H()),
		}},

	// ------------------------------------------------------------------ the thieves' underworld
	{ID: "uw_job", Kinds: "uw", Title: "The Quiet Job",
		Hook: "{rep} has a job for someone who can keep still. A vault in {site} holds {item}, and the guild wants it.",
		Steps: []fqStepT{
			fqs("scout", "Look over the approach near ({n} parasangs {dir}) and see how well it is watched.", "You know the guards' habits.").D(10),
			fqs("fetch", "Take {item} out of {site} and bring it to {rep} in {town}.", "You have it.").D(14),
			fqs("meet", "A fence, {person} in {town2}, offers more than the guild for it.", "You have heard the fence out.").
				C("{person} slides a purse across the table.",
					fqo("Refuse and take it to {rep}.", "{person} shrugs and looks bored.", 4, 0, "loyal"),
					fqo("Sell it to the fence.", "{person} pays on the nail.", -8, 3, "kept").G(200).H(),
					fqo("Return it to the true owner, quietly.", "The owner weeps. {rep} will hear of it.", -3, -5, "returned")),
			fqs("parley", "Hand in your work to {rep} in {town}.", "The job is closed.").
				C("{rep} counts and nods.",
					fqo("Take your cut.", "\"Fair enough.\"", 3, 0, "").G(150),
					fqo("Take a favour in place of coin.", "\"Favours keep. I will remember.\"", 5, 0, "favour")),
			fqs("avenge", "The guild's cutters have been told you sold the take. {foe} leads them.", "The cutters are quiet.").N("kept"),
		}},
	{ID: "uw_snitch", Kinds: "uw", Title: "Rats in the Walls",
		Hook: "Someone is selling the guild to the watch. {rep} wants the leak found and shut.",
		Steps: []fqStepT{
			fqs("meet", "{person} in {town2}, {n} parasangs {dir}, is the likely rat. Talk to them.", "You have spoken with {person}.").
				C("{person} sees what you are.",
					fqo("Silence them for good.", "The rat will talk no more. Their kin will remember.", 4, 0, "silenced"),
					fqo("Warn them off and let them run.", "{person} runs. The guild may never know.", 1, 0, "warned"),
					fqo("Sell them to the watch for a purse.", "The watch pays, and takes {person} away. The guild will not like it.", -7, 4, "betray").G(120).H()),
			fqs("kill", "The rival gang boss {foe} has been eating into the guild's trade near ({n} parasangs {dir}). Cut them out.", "{foe} is gone.").D(10),
			fqs("parley", "Report to {rep} in {town}.", "The report is given.").
				C("{rep} wants the whole story.",
					fqo("Tell it plainly.", "\"Straight. I like that.\"", 3, 0, ""),
					fqo("Leave out one detail.", "\"There is a gap. I will not press.\"", 1, 0, "gap")),
			fqs("avenge", "{contact}'s kin want blood for the rat. They are led by {foe}.", "The kin are quiet.").N("silenced"),
		}},

	// ------------------------------------------------------------------ rival-realm spies
	{ID: "spy_drop", Kinds: "spy", Title: "The Dead Drop",
		Hook: "{rep} works for {fac} and wants a letter moved without anyone watching. It is not a safe errand.",
		Steps: []fqStepT{
			fqs("deliver", "Carry {item} to {person} in {town2}, {n} parasangs {dir}. Do not be seen.", "The letter is delivered.").D(10).
				C("{person} reads, and asks what you saw on the road.",
					fqo("Say nothing.", "\"Good. That is how it is done.\"", 3, 0, "silent"),
					fqo("Copy the letter on the way out, if you can.", "You have a copy now.", 1, 0, "copied").K("tinkering", 10, "{person} sees you at it and says nothing, which is worse.")),
			fqs("hold", "Stay out of sight for {days} days. Counter-agents are about.", "You are still free.").D(3).A(),
			fqs("parley", "Report to {rep} in {town}.", "The matter is closed.").
				C("{rep} asks how to treat a copy.",
					fqo("Hand it over.", "\"Careful of you.\"", 3, 0, ""),
					fqo("Sell the contents to {rival}'s own spymaster.", "{rival}'s spymaster does not blink. You are richer and watched.", -8, 6, "betray").G(220).H()),
			fqs("avenge", "Agents of {fac} want a word about what you sold. One of them is {foe}.", "The agents are stopped.").N("betray"),
		}},
	{ID: "spy_double", Kinds: "spy", Title: "A Double Life",
		Hook: "{rep} of {fac} needs someone who can sit at two tables. Plans are kept at {site}, and a contact in {town2} will tell you how to get them.",
		Steps: []fqStepT{
			fqs("meet", "Meet {person} in {town2}, {n} parasangs {dir}. They serve {rival} and could be turned.", "You have met the contact.").
				C("{person} watches you across the table.",
					fqo("Offer them gold to turn.", "{person} takes it, and turns.", 2, -2, "turned").P(60),
					fqo("Offer them a way out.", "{person} listens, and then nods.", 2, -1, "turned").K("persuasion", 10, "{person} says no and says it kindly."),
					fqo("Warn them and walk away.", "{person} thanks you and leaves, unsure.", -2, 3, "warned")),
			fqs("fetch", "Take {item} out of {site} and bring it to {rep} in {town}.", "You have the plans.").D(14),
			fqs("parley", "Hand over the plans to {rep} in {town}.", "The plans are in other hands.").
				C("{rep} asks what to do with them.",
					fqo("Use them.", "\"Good. We will move before they do.\"", 5, -3, ""),
					fqo("Copy them and keep the copy.", "\"Prudent. I will not look too closely.\"", 2, 0, "copy").G(80),
					fqo("Burn them.", "\"A waste. And a risk, now they know.\"", -3, 3, "burn")),
		}},
}
