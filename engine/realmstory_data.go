package engine

// Data for realmstory.go (prefix rs): the twelve crisis archetypes. Every string may carry the
// placeholders {realm} {ruler} {capital} {faith} {culture} {a} {b} {foe} {house}; rsFill fills them from
// the realm's real data. Stage 0 is the whisper, stage 1 the open trouble, stage 2 the breaking point.

type rsArch struct {
	ID     string
	Noun   string    // "a succession dispute"
	Third  string    // third entry point: duel, relic or card
	Name   string    // the crisis' name on the screen
	Brief  string    // one line of what is going on
	Lab    [5]string // choice labels: side A, side B, expose, broker, profit
	Chk    [3]string // checkKinds ids for expose, broker, profit
	Notice [3]string // town notice per stage
	Say    [3]string // what townsfolk say per stage
	Info   string    // what the informant tells
	Doc    string    // the document's text
	Facts  [2]string // learned in steps: who stands behind it, what is hidden
	Out    [4]string // results of a, b, expose, broker
	Tag    [5]string // aftermath tag after a, b, expose, broker, profit
	Items  [2]string // document name, relic name
	Price  int       // percent added to prices per stage step
}

var rsArchs = []rsArch{
	{ID: "succession", Noun: "a succession dispute", Third: "duel",
		Name: "The {house} succession", Brief: "{ruler} has no settled heir, and two claimants, {a} and {b}, each say the throne is theirs.",
		Lab:    [5]string{"Side with {a}", "Side with {b}", "Show that the will was forged", "Broker a regency council", "Sell your word to both claimants"},
		Chk:    [3]string{"lore", "persuasion", "streetwise"},
		Notice: [3]string{"By order of the court: talk of the succession is to be left to the court.", "Two banners now hang in {capital}: one for {a}, one for {b}. Taking down either is a fine.", "The gates of {capital} are watched. Armed retainers of {a} and {b} may not enter the same street."},
		Say:    [3]string{"They say {ruler} is slow to name an heir. People talk, quietly.", "Half of {capital} wears {a}'s colour and half {b}'s. Mind which you wear.", "Both claimants have armed men in the city. It will be settled with steel if it is not settled soon."},
		Info:   "I copy for the court. There are two wills, and the ink on one is newer than the seal on it. Whoever shows that in public ends this.",
		Doc:    "A copy of the old will of {ruler}, naming a successor, with a second hand adding {b}'s name below the seal.",
		Facts:  [2]string{"{a} has the old nobles; {b} has the garrison and the guilds.", "The will that favours {b} was altered after the seal was set."},
		Out:    [4]string{"{a} is named heir. {b}'s supporters leave the city angry.", "{b} is named heir and the garrison cheers. The old nobles go home to sulk.", "The altered will is shown at court. The forger is named and the true heir is confirmed.", "A regency council of both houses and the guilds is sworn in. Neither claimant is pleased, and neither is dead."},
		Tag:    [5]string{"{a} was named heir", "{b} was named heir", "the forged will was exposed", "a regency council rules", "claimants bought and sold"},
		Items:  [2]string{"a copied will", "a sealed will-casket"}, Price: 2},
	{ID: "unrest", Noun: "a plague of unrest", Third: "card",
		Name: "The unrest in {realm}", Brief: "Strikes, bread riots and torn notices spread from town to town in {realm}; nobody agrees who started it.",
		Lab:    [5]string{"Back the wardens who keep order", "Back the leaders of the unrest", "Unmask who pays the agitators", "Bring both sides to talk in the market", "Sell supplies to both camps"},
		Chk:    [3]string{"perception", "persuasion", "streetwise"},
		Notice: [3]string{"Assemblies of more than ten are discouraged. Torn notices will be replaced at the finder's cost.", "Curfew at dusk by order of the wardens of {realm}. Tearing this notice is an offence.", "Shops are shut by order. The wardens will answer force with force."},
		Say:    [3]string{"Wages are late again, and someone has been handing out leaflets in the lanes.", "There was a riot by the granary last night. The wardens charged, and now nobody trusts them.", "They have barricaded two streets. The wardens say they will clear them by dawn."},
		Info:   "Someone is paying the street speakers. The coin is from one house, and it is not a house that wants better wages, only a weaker crown.",
		Doc:    "A tally of coin paid to street speakers across {realm}, signed with a merchant's chop.",
		Facts:  [2]string{"The unrest began over late wages and a price rise nobody explained.", "Some of the loudest speakers are paid by a rival of the crown."},
		Out:    [4]string{"The wardens clear the streets. Order returns, and so does fear.", "The leaders' demands are read in the hall. Wages are paid, and some officials are not forgiven.", "The paymaster is named. The crowds go home, and the real quarrel is argued in court.", "Both sides meet under a market awning. A wage and price table is agreed and posted."},
		Tag:    [5]string{"the wardens restored order", "the leaders won their demands", "the paymaster was unmasked", "a market truce holds", "both camps were supplied"},
		Items:  [2]string{"a leaflet bundle", "a strongbox of strike pay"}, Price: 3},
	{ID: "heresy", Noun: "a heresy trial", Third: "relic",
		Name: "The trial of {a}", Brief: "{a} stands accused before the court of {faith} of teaching a doctrine the temple calls false, and the town is split over it.",
		Lab:    [5]string{"Side with the court of {faith}", "Defend {a}", "Show the trial is a land grab", "Arrange a public disputation", "Take payment from the accused's patrons"},
		Chk:    [3]string{"lore", "persuasion", "bluff"},
		Notice: [3]string{"The court of {faith} will hear a matter of doctrine. All faithful are invited.", "Tracts by {a} are forbidden. Possession is a crime. By order of the temple.", "The trial of {a} is set for the next holy day in {capital}. Pilgrims are warned to lodge early."},
		Say:    [3]string{"They say a teacher in {capital} has been explaining {faith} in new ways.", "Some say {a} is a saint, and some say a liar. Neither says it loudly.", "The pyres are being made ready, or the platform for the pardon. Nobody will say which."},
		Info:   "The temple wants {a}'s lands more than {a}'s silence. A letter in the vestry shows who gets the estate if {a} is found guilty.",
		Doc:    "A deed of transfer, unsigned, giving the estate of {a} to the temple on a verdict of guilt.",
		Facts:  [2]string{"{a} taught that the old rites could be read another way.", "The temple would gain {a}'s estate on a guilty verdict."},
		Out:    [4]string{"The court finds {a} guilty. Tracts are burned and the town goes quiet.", "{a} is acquitted amid shouting. The temple's judges leave by a side door.", "The deed is read aloud in the court. The charge is dropped and the temple pays costs.", "A public disputation is held. Neither side wins, and both claim to have."},
		Tag:    [5]string{"{a} was condemned", "{a} was freed", "the land grab was exposed", "the disputation ended in a draw", "the accused's patrons paid up"},
		Items:  [2]string{"an unsigned deed", "a banned tract"}, Price: 1},
	{ID: "cult", Noun: "a cult in the capital", Third: "relic",
		Name: "The hidden lodge of {capital}", Brief: "A secret lodge meets under {capital}; its members include clerks, watchmen and at least one noble house.",
		Lab:    [5]string{"Back the watch's purge", "Speak for the lodge", "Expose the lodge's patrons", "Offer the lodge a lawful chapel", "Sell relics to the lodge"},
		Chk:    [3]string{"perception", "persuasion", "streetwise"},
		Notice: [3]string{"Lost: a signet ring with a spiral mark. Return to the watch house in {capital}. No questions.", "By order of the watch: unlawful assemblies in cellars are forbidden. Report strangers who chalk spirals.", "Cellars of {capital} are searched by the watch. Householders must open their doors."},
		Say:    [3]string{"People chalk a spiral on doors in the lower town. Nobody says why.", "Three clerks were not at their desks this week. The watch asks questions and gets none.", "They say the lodge has a noble house behind it. The watch say they have names."},
		Info:   "The lodge pays for silence. I carry their notes. The names on them reach higher than the watch house, and the watch captain is one of them.",
		Doc:    "A page of names in cipher with a spiral stamped in wax; several look like court clerks' hands.",
		Facts:  [2]string{"The lodge recruits clerks, watchmen and traders with debts.", "The watch captain and one noble house stand behind it."},
		Out:    [4]string{"The watch purges the lodge. Some are jailed, some vanish, and the rumours grow.", "The lodge is allowed to meet. Some in {capital} thank you; the temple does not.", "The patrons are named in open court. The lodge scatters and the captain is replaced.", "A chapel is chartered for the lodge, with the watch at its door. It is called a compromise."},
		Tag:    [5]string{"the lodge was purged", "the lodge was tolerated", "the lodge's patrons were exposed", "the lodge got a chartered chapel", "the lodge bought relics"},
		Items:  [2]string{"a page of cipher", "a spiral-marked reliquary"}, Price: 1},
	{ID: "warweary", Noun: "border war weariness", Third: "card",
		Name: "The war with {foe}", Brief: "{realm} has been at war with {foe} too long. The fields lack hands, the treasury lacks coin, and the war party and the peace party no longer speak.",
		Lab:    [5]string{"Back the war party", "Back the peace party", "Expose the profiteers who prolong the war", "Carry a truce offer to {foe}'s envoy", "Sell to the army contractors"},
		Chk:    [3]string{"streetwise", "persuasion", "bluff"},
		Notice: [3]string{"Levy notice: one adult in each household to report to the muster of {realm}.", "Deserters will be hanged. Informers will be paid. By order of the marshal.", "The harvest levy is doubled for the war. Those who hide grain are traitors."},
		Say:    [3]string{"My cousin has been at the border two summers. We have not heard in a month.", "The contractors are rich and the soldiers are hungry. Make of that what you will.", "Even the marshal's men are talking about home. This cannot go on."},
		Info:   "The contractors lose if there is peace. I have seen their orders. They bought the same sacks of flour three times and shipped none.",
		Doc:    "A contractor's ledger showing grain and arms bought for the border that never arrived.",
		Facts:  [2]string{"The war party wants the border pass; the peace party wants the harvest.", "Several contractors profit from keeping the war alive."},
		Out:    [4]string{"The war party gets its levy. The border holds, and the villages empty further.", "The peace party gets a truce motion. Heralds ride to {foe} under a white banner.", "The contractors are named. Their charters are pulled, and the muster rolls grow honest.", "Your offer is carried across the border. A truce is signed for one season, which is more than anyone expected."},
		Tag:    [5]string{"the war levy was granted", "peace talks were opened", "the war profiteers were exposed", "a truce was brokered", "the contractors were served"},
		Items:  [2]string{"a contractor's ledger", "a marshal's sealed order"}, Price: 4},
	{ID: "famine", Noun: "famine and hoarding", Third: "relic",
		Name: "The hungry months in {realm}", Brief: "Grain is short in {realm} and the granaries of {capital} are said to be full behind locked doors.",
		Lab:    [5]string{"Back the requisition of grain", "Back the granary riot", "Expose the hoarders", "Broker a grain-sharing deal", "Buy low and sell dear"},
		Chk:    [3]string{"perception", "persuasion", "streetwise"},
		Notice: [3]string{"Bread is sold by weight. One loaf a day per household in {realm}.", "Hoarding grain is a hanging matter. Reports to the nearest warden will be rewarded.", "The granaries of {capital} are closed to all but the court. Queues are forbidden."},
		Say:    [3]string{"Bread was cheaper last winter. The millers say the harvest was fine. Who has it, then?", "The queue at the bakery starts at midnight now. Some give up and walk to the next town.", "They say the granaries are full. If so, someone will hang for it, or the whole town will."},
		Info:   "Count the carts, not the loaves. Grain goes into the old customs house at night and out by water under other names. It's three merchant houses.",
		Doc:    "A customs ledger with grain landed under false names and no sales recorded.",
		Facts:  [2]string{"The harvest was fair; the shortage is partly made.", "Three merchant houses hold the grain in the old customs house."},
		Out:    [4]string{"Grain is requisitioned by decree. The queues shrink, and the merchants remember it.", "The granary doors are broken. Grain is shared in the street and the guard looks away.", "The hoarders are named. Their stocks are opened at a fair price and the bakers fire their ovens.", "A sharing deal is struck: the merchants release grain, the crown pays them over time."},
		Tag:    [5]string{"grain was requisitioned", "the granaries were broken open", "the hoarders were exposed", "a grain-sharing deal feeds the town", "the hoard was played for profit"},
		Items:  [2]string{"a customs ledger", "a locked granary tally"}, Price: 6},
	{ID: "usurper", Noun: "a usurper general", Third: "duel",
		Name: "General {b}'s ambition", Brief: "General {b} commands more soldiers than the crown can pay, and says openly that {ruler} is too weak to hold {realm}.",
		Lab:    [5]string{"Stand with the crown", "Stand with General {b}", "Reveal the general's letters", "Broker a pardon and a posting far away", "Sell arms to the general's camp"},
		Chk:    [3]string{"lore", "persuasion", "streetwise"},
		Notice: [3]string{"The army is loyal to {realm}. Rumours to the contrary are treason.", "Soldiers of the general's regiments are barred from the market quarter after dark.", "The garrison of {capital} has changed its colours. Loyal subjects are asked to remain at home."},
		Say:    [3]string{"The general's men are paid on time. The crown's are not. Think on that.", "General {b} dined with three governors last week. They all came home silent.", "The soldiers will follow the general. It's only a matter of the day."},
		Info:   "The general writes to a foreign court. I carried a letter; I did not like the seal on it. Find his letter-case in the quartermaster's rooms.",
		Doc:    "A letter in General {b}'s hand promising a foreign court the border passes in return for support.",
		Facts:  [2]string{"The general's regiments are paid from his own estates.", "He has written to a foreign court."},
		Out:    [4]string{"The crown outbids the general for the regiments. He is retired with honours, and watched.", "General {b} takes the capital without a battle. The crown's officers resign.", "The letters are read at court. The regiments disown the general and he is arrested.", "The general accepts a distant posting and a pardon. The court calls it a victory."},
		Tag:    [5]string{"the crown kept the army", "General {b} took power", "the general's treason was exposed", "the general was pardoned and posted away", "the general's camp was armed"},
		Items:  [2]string{"a letter-case", "the general's seal"}, Price: 3},
	{ID: "guild", Noun: "a trade-guild strangle", Third: "card",
		Name: "The guild tolls of {realm}", Brief: "The great guilds of {realm} charge a toll on every road, bridge and stall, and small traders are leaving or being broken.",
		Lab:    [5]string{"Back the crown's charter court", "Back the guild masters", "Expose the toll ledgers", "Draft a fair tariff", "Take a rebate from the guilds"},
		Chk:    [3]string{"lore", "persuasion", "bluff"},
		Notice: [3]string{"Guild dues for the season are posted on the guildhall door of {capital}. Unpaid dues are doubled.", "Stalls without a guild chop will be closed. Inspectors will visit.", "The road and bridge tolls of {realm} are raised again, by agreement of the guild masters."},
		Say:    [3]string{"I used to sell cloth on market day. Now I pay a chop, a toll and an inspector.", "Three stalls on my row closed this month. The guild bought the stock cheap.", "Everything costs more because everything crosses a toll. Even a loaf."},
		Info:   "The tolls are not recorded where the court can see. There's a second ledger in the guildhall. The crown's tax and the guild's share don't add up.",
		Doc:    "A second set of toll accounts, kept apart from the one shown to the crown's assessors.",
		Facts:  [2]string{"The guilds pay a share of the tolls to the crown's officials.", "A second set of toll accounts is kept at the guildhall."},
		Out:    [4]string{"The charter court strips the guilds of their roadside tolls. They are furious, and the roads are cheaper.", "The guild masters' charter is renewed. Prices stay high and the roads stay safe.", "The second ledger is read at court. Fines are laid, and the tolls are cut by half.", "A new tariff is posted at every bridge and stall. It is not liked, and it is obeyed."},
		Tag:    [5]string{"the guilds lost their tolls", "the guilds kept their charter", "the toll ledgers were exposed", "a fair tariff was posted", "the guilds paid a rebate"},
		Items:  [2]string{"a toll ledger page", "a guild strongbox"}, Price: 3},
	{ID: "haunted", Noun: "a haunted dynasty", Third: "relic",
		Name: "The curse of the {house} line", Brief: "Nobody of the {house} line has died in bed for four generations, and the household of {ruler} says the dead of the house walk in the old wing.",
		Lab:    [5]string{"Side with the exorcists", "Side with the household", "Show the truth of the first crime", "Broker rites of rest", "Sell the heirloom they fear"},
		Chk:    [3]string{"lore", "persuasion", "bluff"},
		Notice: [3]string{"The old wing of the palace is closed. Servants are forbidden to speak of the noises.", "Prayers for the dead of the {house} line are said at dusk in every temple of {capital}.", "The court has hired a seer. Visitors to the palace are searched for charms."},
        Say:    [3]string{"They say the old wing is empty but lit at night. I don't go that way.", "Servants are leaving the palace in numbers. Ask one why. They all say the same thing.", "The {house} line is called cursed in every tavern, and nobody laughs."},
		Info:   "It is not a ghost story. It is a record: the first of the line took the throne by a crime the house paid to forget. There's a letter in the crypt that says so.",
		Doc:    "A page from the old house chronicle, torn out, describing how the first of the line took the throne.",
		Facts:  [2]string{"The household says the dead of the house walk the old wing.", "The first of the line took the throne by a crime the house hid."},
		Out:    [4]string{"The exorcists seal the old wing and the household breathes easier, for a time.", "The household keeps its dead. The old wing stays open and the rumours grow quiet.", "The first crime is read aloud and the house makes amends at the grave. The wing is opened.", "Rites of rest are held by the house and the temples together. The old wing is quiet."},
		Tag:    [5]string{"the old wing was sealed", "the household kept its dead", "the first crime was told aloud", "rites of rest were held", "the cursed heirloom was sold"},
		Items:  [2]string{"a torn chronicle page", "a cursed house heirloom"}, Price: 1},
	{ID: "missingheir", Noun: "a missing heir", Third: "card",
		Name: "The lost heir of {house}", Brief: "The heir of {ruler} vanished years ago. A stranger now says they are that heir, and {a} rules as regent and says otherwise.",
		Lab:    [5]string{"Back regent {a}", "Back the claimant", "Find out who the claimant really is", "Broker a trial by witnesses", "Sell what you know to both sides"},
		Chk:    [3]string{"perception", "persuasion", "bluff"},
		Notice: [3]string{"All persons who knew the heir of {house} in youth are asked to speak to the regent's clerk.", "Persons claiming royal blood without proof will be put in the stocks. By order of regent {a}.", "A public hearing on the matter of the heir will be held in {capital}. Doors open at dawn."},
		Say:    [3]string{"They say a stranger turned up at the palace gate with a ring and a story.", "The old nurse says the claimant knows things only the heir could know. The regent says nurses are old.", "If the claimant is real, the regent has ruled illegally. You can see why nobody wants to say it."},
		Info:   "I was a page of the old household. The claimant has the ring, the scar and the voice, but not the left-handedness. The heir wrote with the left. Ask who taught the claimant.",
		Doc:    "A household record of the heir's birth marks, with a note about left-handed writing.",
		Facts:  [2]string{"The claimant has the ring and knows the household.", "The heir wrote left-handed; the claimant does not."},
		Out:    [4]string{"The regent's rule is confirmed. The claimant is escorted from {capital}, protesting.", "The claimant is crowned heir by acclamation. The regent withdraws to a country estate.", "The claimant's true name is shown and the claim fails. The regent stays, with less trust than before.", "A hearing of witnesses is held. The claimant is judged genuine, with a regent's council beside them."},
		Tag:    [5]string{"the regent's rule was confirmed", "the claimant was accepted as heir", "the claimant was shown a fraud", "witnesses settled the claim", "both sides bought news"},
		Items:  [2]string{"a household record", "a signet ring"}, Price: 1},
	{ID: "debt", Noun: "ruinous debt", Third: "card",
		Name: "The debts of {realm}", Brief: "{realm} owes more to {foe}'s bankers than it earns in a year, and the lenders want the customs of {capital} in pledge.",
		Lab:    [5]string{"Back austerity and repayment", "Back repudiating the debt", "Expose the lenders' terms", "Restructure the loan", "Lend coin at interest yourself"},
		Chk:    [3]string{"lore", "persuasion", "streetwise"},
		Notice: [3]string{"A new levy is placed on salt, oil and cloth to meet the obligations of the treasury of {realm}.", "Officials will be paid in scrip until further notice. By order of the treasurer.", "The customs of {capital} are pledged to the lenders. Duties are payable at the new counting-house."},
		Say:    [3]string{"They have put a levy on salt. They say it's temporary.", "I was paid in paper this month. The baker will not take paper.", "The counting-house is staffed by foreigners. They count in a different way, they say."},
		Info:   "The loan terms are worse than the court has said. There's a clause that gives the lenders the customs outright after two missed payments. The treasurer hid it.",
		Doc:    "The clause of the loan that hands the customs of {capital} to the lenders after two missed payments.",
		Facts:  [2]string{"The loan was taken to pay the border war and the palace.", "The lenders may seize the customs after two missed payments."},
		Out:    [4]string{"The levy stands and payments are met. The treasury recovers, and the poor pay for it.", "The debt is repudiated. The lenders curse {realm} and close their purses.", "The clause is read in court. The lenders are forced to renegotiate and the treasurer resigns.", "The loan is restructured over twenty years. Nobody is happy, and nobody is ruined."},
		Tag:    [5]string{"the levy paid the debt", "the debt was repudiated", "the loan's terms were exposed", "the loan was restructured", "you lent at interest"},
		Items:  [2]string{"a loan clause", "a lender's strongbox"}, Price: 3},
	{ID: "rebel", Noun: "a rebel province", Third: "duel",
		Name: "The revolt of the border province", Brief: "A province of {realm} has stopped sending taxes, and its people say they will answer to their own council.",
		Lab:    [5]string{"Back the crown's garrison", "Back the provincial council", "Expose the governor's graft", "Broker a charter of autonomy", "Run goods through the cordon"},
		Chk:    [3]string{"perception", "persuasion", "streetwise"},
		Notice: [3]string{"Tax carts from the border province are overdue. Reports of delay will be heard by the treasurer.", "Travel to the border province requires a pass from the garrison of {capital}.", "The border province is under a cordon. Those who cross it without leave are rebels."},
		Say:    [3]string{"The border folk say the governor takes twice what the crown asks.", "The council in the border province levies its own tax. They say it's fairer.", "The cordon is leaking. Smugglers know three ways through, and the guards know them too."},
		Info:   "The governor of the province skims the taxes. That's the whole of it. I have the tally. The people are not against the crown; they're against him.",
		Doc:    "The governor's private tally of taxes taken in the border province against the sum sent to the crown.",
		Facts:  [2]string{"The governor takes twice what the crown asks.", "The people oppose the governor, not the crown."},
		Out:    [4]string{"The garrison marches in. The council is dissolved, the governor kept, and the border goes quiet like a held breath.", "The council is recognised as the province's voice. The crown sulks and the taxes trickle in.", "The governor's tally is read at court. He is removed, and the province pays its taxes again.", "A charter of autonomy is signed. The province keeps its council and pays a lower tax."},
		Tag:    [5]string{"the garrison retook the province", "the provincial council won recognition", "the governor's graft was exposed", "a charter of autonomy was signed", "goods ran through the cordon"},
		Items:  [2]string{"a tax tally", "the governor's strongbox"}, Price: 2},
}

// Generic worsening lines for a crisis left to run.
var rsWorseLines = []string{
	"It has got worse since anyone tried to stop it. Nobody has, and everybody knows.",
	"Things are bad now. Whatever it was before, it's worse for being ignored.",
	"The trouble has been left too long. People have stopped hoping, and some have stopped asking.",
}

// Generic resolution wording for profit, ignore and failures.
var rsProfitLines = []string{
	"You sell to both sides, quietly, and leave with a full purse and a worse conscience. The trouble grows.",
	"You find a buyer on each side. The coin is good. The crisis is not smaller for it.",
}

var rsIgnoreLines = []string{
	"You turn away. The trouble will not.",
	"You decide it is not your business. It will not agree.",
}
