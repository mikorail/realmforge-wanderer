package engine

// Crossroads templates (prefix cr). Effect strings: mer/hon/cou/gen/pie:<n> karma (+ = mercy, honesty, courage,
// generosity, piety); gold:<n> hp:<n> time:<min> rep:realm|faith|home|out:<n> fav:<n> op:P|O:<n> loy:<n> bounty:<n>
// item:food|tonic|loot|oddity|scrap|weapon|book lose:food xp:<n> news:<text> deed:<text> spawn:bandit:<n>
// fu:<followup id>:<days>. Check "kind:dc" uses checks.go kinds. Fl: lie (no lying under a Truth vow), tell (no
// speaking under a Silence vow). Th is the theme. Trig: t town, r road junction, c camp, d after big deeds.

type crCh struct{ T, Req, Chk, Win, WT, Lose, LT, Fl string }

type crTpl struct {
	ID, Th, Trig, Need, Gn, Title, Text string
	Ch                                  []crCh
	Dec                                 string
	Fu                                  bool
}

var crTpls = []crTpl{
	// ---- justice ----
	{ID: "loaf", Th: "justice", Trig: "t", Need: "person", Title: "The Stolen {loaf}",
		Text: "A ragged child is pinned against a stall by {P}, who is shouting about a stolen {loaf}. A {guard} is already shouldering through the crowd. The child looks at you as if you were the only door in the world.",
		Ch: []crCh{
			{"Pay for the {loaf}.", "gold:5", "", "gen:3;mer:2;gold:-5;op:P:3;fu:thief_back:5", "{P} grumbles but takes the coin. The child bolts, and looks back once.", "", "", ""},
			{"Let the {guard} deal with it.", "", "", "hon:2;mer:-3;rep:realm:3;fu:thief_knife:6", "The {guard} drags the child off. The crowd approves. You do not feel better.", "", "", ""},
			{"Talk {P} into letting the child go.", "", "persuasion:9", "mer:4;op:P:1;fu:thief_back:6", "{P} waves a hand in disgust. The child is gone before the hand drops.", "mer:1;op:P:-3", "{P} snaps that thieves hang for less, and you are no better.", ""}}},
	{ID: "noble_accuse", Th: "justice", Trig: "t", Need: "person,realm", Title: "A Noble's Accusation",
		Text: "{P}, a favourite of {R}, accuses a servant of stealing a jewel. You saw the servant that night, nowhere near the house. The servant stares at the floor and says nothing; the {guard}s wait for a word.",
		Ch: []crCh{
			{"Testify to what you saw.", "", "", "hon:5;cou:3;op:P:-5;rep:realm:-4", "The servant is freed. {P} will remember who spoke.", "", "", ""},
			{"Back the accusation. It pays.", "", "", "hon:-6;gold:30;rep:realm:5;op:P:5", "The servant is led away. {P} pays you quietly.", "", "", "lie"},
			{"Stay out of it.", "", "", "cou:-3", "Not your quarrel. The servant is led away.", "", "", ""}}, Dec: "cou:-3;mer:-2"},
	{ID: "false_witness", Th: "justice", Trig: "t", Need: "person", Title: "False Witness for Hire",
		Text: "In a back alley {P} offers coin to swear that a stranger broke a lease in {T}. The stranger is a poor man with a family. {P} needs one more voice, and you have an honest face.",
		Ch: []crCh{
			{"Take the coin and swear.", "", "", "gold:40;hon:-6;mer:-3;op:P:4", "The court believes you. The poor man's door is nailed shut by evening.", "", "", "lie"},
			{"Refuse and walk away.", "", "", "hon:3", "{P} spits, then finds someone cheaper.", "", "", ""},
			{"Warn the {guard}s.", "", "persuasion:9", "hon:4;rep:realm:5;op:P:-6;fu:thief_knife:5", "The {guard}s arrive, and {P} spends the night learning who you are.", "hon:1;op:P:-4", "The {guard}s shrug. Nothing is proved, and {P} now knows your face.", "tell"}}},
	{ID: "debtor", Th: "justice", Trig: "t", Need: "person", Title: "The Debtor's Stocks",
		Text: "A man sits in the stocks of {T} for a debt of thirty {coin}. {P} holds the paper and is in no hurry. His children sit at his feet and do not ask you for anything, which is worse.",
		Ch: []crCh{
			{"Pay the debt.", "gold:30", "", "gold:-30;gen:6;mer:3;op:P:-1;xp:20", "The stocks open. The man cannot speak. His children can.", "", "", ""},
			{"Haggle {P} down.", "", "persuasion:11", "gold:-10;gen:2;mer:2;hon:1", "{P} accepts a third and glares.", "gold:-5;op:P:-2", "You pay a little and he refuses the rest.", ""},
			{"Leave him there.", "", "", "gen:-3;mer:-2", "Debts are debts.", "", "", ""}}},
	{ID: "mob", Th: "justice", Trig: "t", Title: "The Mob",
		Text: "A crowd in {T} has seized a stranger and is dragging him toward the river. Nobody can say what he did. Someone says he is a {undead}-lover; someone else, a spy for {W}. The rope is already out.",
		Ch: []crCh{
			{"Face them down.", "", "intimidate:11", "cou:5;mer:4;rep:realm:3;xp:25", "The crowd falters under your stare and melts away.", "hp:-8;cou:2", "They throw you aside, and bloody your face before you stand.", ""},
			{"Shout something clever.", "", "persuasion:10", "mer:3;cou:2;hon:1", "A laugh breaks the spell. The stranger is gone before anyone remembers him.", "mer:-1", "They are not in a mood for cleverness.", ""},
			{"Join in.", "", "", "mer:-8;hon:-2;rep:realm:2;spawn:bandit:0", "It is over fast. You will not forget the sound.", "", "", ""}}, Dec: "mer:-3;cou:-3"},
	{ID: "bribe_guard", Th: "justice", Trig: "t", Need: "person", Title: "The Guard's Cut",
		Text: "A {guard} takes coin from {P}, a market woman, for the right to stay in {T}'s square. When she has nothing more to pay, he turns her cart over.",
		Ch: []crCh{
			{"Pay him on her behalf.", "gold:15", "", "gold:-15;gen:3;op:P:5;rep:realm:-1", "He pockets it with a thin smile. She will not forget.", "", "", ""},
			{"Report him to the watch-captain.", "", "persuasion:11", "hon:4;rep:realm:5;op:P:4;fu:thief_knife:8", "He is stripped of his badge. He knows who spoke.", "hon:1;op:P:1;rep:realm:-3", "The captain is his cousin.", "tell"},
			{"Look away.", "", "", "cou:-3;mer:-2", "The cart rolls into the gutter.", "", "", ""}}, Dec: "cou:-3"},
	// ---- mercy and cruelty ----
	{ID: "prisoners", Th: "mercy", Trig: "rc", Title: "The Captives",
		Text: "A patrol of {R} has taken three {bandit} and tied them to a cart. The sergeant, tired, says they can go to the gallows in {N}, or you can pay for the rope and save everyone the walk. One captive is hardly more than a boy.",
		Ch: []crCh{
			{"Free the captives.", "", "", "mer:6;rep:realm:-8;rep:out:12;fu:spared_allies:7", "The sergeant swears. The bandits scatter without a word of thanks, though one looks back.", "", "", ""},
			{"Buy their lives (forty {coin}).", "gold:40", "", "gold:-40;mer:4;gen:2;rep:out:6;fu:spared_allies:9", "The sergeant takes the coin and the rope back. The boy cries.", "", "", ""},
			{"Take the rope.", "", "", "mer:-8;rep:realm:5;rep:out:-10", "You tie it yourself. It is quicker than the gallows.", "", "", ""}}},
	{ID: "wounded_foe", Th: "mercy", Trig: "rcd", Title: "The Wounded Enemy",
		Text: "A soldier of {W}, bleeding at the roadside, looks at you without hope. His sword is under his own hand but he doesn't lift it. 'Do what you came to do,' he says.",
		Ch: []crCh{
			{"Bind his wounds.", "item:food", "", "lose:food;mer:6;fu:spared_allies:8;pie:1", "He drinks, sleeps, and wakes to find you gone, and a debt on his name.", "", "", ""},
			{"Leave him his sword and go.", "", "", "mer:2;cou:1;fu:spared_vendetta:9", "You leave him alive and armed. That might matter later.", "", "", ""},
			{"Finish it.", "", "", "mer:-6;hon:-1;xp:5", "It is over quickly.", "", "", ""}}},
	{ID: "deserter", Th: "mercy", Trig: "rc", Need: "realm", Title: "The Deserter",
		Text: "A boy in the torn colours of {R} begs you for shelter. He walked away from a line that was about to break. A patrol is probably a few hours behind him. A reward for deserters is posted in every town.",
		Ch: []crCh{
			{"Hide him and send him on.", "", "", "mer:5;cou:3;rep:realm:-6;fu:deserter_thanks:10", "He weeps, then he is gone before dawn.", "", "", ""},
			{"Turn him in for the reward.", "", "", "gold:25;rep:realm:6;hon:1;mer:-5", "The patrol thanks you. The boy does not.", "", "", ""},
			{"Tell him to run.", "", "", "mer:1", "He runs. You do not learn where.", "", "", ""}}},
	{ID: "starving_child", Th: "mercy", Trig: "t", Need: "famine", Title: "A Hungry Child",
		Text: "The famine has reached {T}. A child sits against a wall, too tired to hold out a hand. Your pack has food in it. Fifty people could say the same.",
		Ch: []crCh{
			{"Give food.", "item:food", "", "lose:food;gen:5;mer:3;pie:1", "The child eats with both hands and doesn't say a word.", "", "", ""},
			{"Give coin and tell her where the temple kitchen is.", "gold:5", "", "gold:-5;gen:2;mer:1;fav:2", "A small kindness, which she can spend.", "", "", ""},
			{"Walk on. You can't feed a town.", "", "", "gen:-3;mer:-2", "True, which does not help.", "", "", ""}}},
	{ID: "caged_beast", Th: "mercy", Trig: "t", Title: "A Beast in Chains",
		Text: "A showman in {T} charges a copper to poke a chained, half-starved creature with a stick. It has stopped flinching. The showman lifts his own stick toward you in invitation.",
		Ch: []crCh{
			{"Break the chains.", "", "tinkering:10", "mer:5;cou:2;fu:freed_beast:6;rep:realm:-2", "The links give. The creature stands, bewildered, then bolts into the dark.", "bounty:10;hp:-3;mer:1", "The showman raises an alarm before the last link breaks.", ""},
			{"Buy it (thirty {coin}) and free it.", "gold:30", "", "gold:-30;mer:4;gen:2;fu:freed_beast:6", "The showman is thrilled, the creature uncomprehending, then gone.", "", "", ""},
			{"Pay a copper and take the stick.", "", "", "mer:-5;gold:-1", "It stares at you.", "", "", ""}}, Dec: "mer:-1"},
	{ID: "widow", Th: "mercy", Trig: "d", Need: "town", Title: "The Widow's Question",
		Text: "A woman from {T} has walked three days to find the one who killed her husband. She has found you. She does not weep. She asks: was he afraid at the end? It is a question with no right answer.",
		Ch: []crCh{
			{"Tell her the truth, gently.", "", "", "hon:4;mer:3;pie:1", "She nods. She thanks you, which is the hardest part.", "", "", ""},
			{"Give her coin, and ask forgiveness.", "gold:40", "", "gold:-40;gen:4;mer:4;hon:1", "She takes it and doesn't forgive you, but she takes it.", "", "", ""},
			{"Say he died like a coward, if it helps.", "", "", "hon:-3;mer:2", "She thanks you for that. It was the wrong thing, and you both know it.", "", "", "lie"},
			{"Tell her to go home.", "", "", "mer:-4", "She leaves without a word, and the road is quiet.", "", "", ""}}},
	// ---- loyalty (companions) ----
	{ID: "comp_favour", Th: "loyalty", Trig: "rct", Need: "comp", Title: "A Companion's Favour",
		Text: "{C} asks, haltingly, for a day of your time. An old score needs settling in {N}, a man who owes {C} a debt of honour. It is a day, and your name in a stranger's mouth, and the {guard}s of {R} take a dim view of private justice.",
		Ch: []crCh{
			{"Go with {C}.", "", "", "loy:15;time:480;rep:realm:-5;gen:2", "It takes a day. {C} says little afterward, but walks closer.", "", "", ""},
			{"Offer to buy the debt instead.", "gold:30", "", "gold:-30;loy:8;gen:3", "{C} accepts, with an odd look.", "", "", ""},
			{"Tell {C} it isn't worth it.", "", "persuasion:10", "loy:3;mer:-1", "{C} nods slowly, though the matter is not closed.", "loy:-8", "{C} says nothing and the silence is expensive.", ""},
			{"Refuse flatly.", "", "", "loy:-12;cou:-1", "{C} takes it in silence.", "", "", ""}}},
	{ID: "comp_debt", Th: "loyalty", Trig: "rct", Need: "comp", Title: "A Loan Between Friends",
		Text: "{C} asks to borrow forty {coin} for a sick relative at home. They say they will repay it. They also look at the floor as they say it, the way a person does who is used to refusal.",
		Ch: []crCh{
			{"Give it freely.", "gold:40", "", "gold:-40;loy:20;gen:5", "{C} is quiet for a long time, and then simply says thank you.", "", "", ""},
			{"Lend it, with terms.", "gold:40", "", "gold:-40;loy:6;gen:1", "A loan, then. {C} nods, understanding both sides.", "", "", ""},
			{"Say you're short.", "", "", "loy:-6;gen:-2", "{C} says it's fine, and means it less each time.", "", "", "lie"}}},
	{ID: "comp_home", Th: "loyalty", Trig: "r", Need: "comp", Title: "Homesickness",
		Text: "{C} stops at the junction and looks down the road toward home. It is a day's walk. They don't ask, which makes it plain they want to.",
		Ch: []crCh{
			{"Take the detour.", "", "", "loy:12;time:600;gen:1", "A night by a stranger's hearth, and {C} sleeps well for the first time in weeks.", "", "", ""},
			{"Promise to go later.", "", "", "loy:2;hon:-1", "{C} smiles. They seem to know you mean it, mostly.", "", "", "lie"},
			{"Say there is no time.", "", "", "loy:-7", "True, and cold.", "", "", ""}}},
	{ID: "comp_blame", Th: "loyalty", Trig: "d", Need: "comp", Title: "Whose Fault",
		Text: "After the fight, {C} sits wrapped in a bandage and says what everyone has been thinking: you made the wrong call back there. The words are fair. They are also loud enough for strangers to hear.",
		Ch: []crCh{
			{"Take the blame.", "", "", "loy:10;hon:4;cou:2", "{C} hadn't expected that. It helps more than any argument.", "", "", ""},
			{"Point out what {C} did wrong.", "", "persuasion:11", "loy:2;hon:1", "A hard exchange, but fair, and settled.", "loy:-8;mer:-1", "It becomes a shouting match, and a poor one.", ""},
			{"Tell {C} to shut up.", "", "intimidate:9", "loy:-6;mer:-2", "{C} says nothing, and keeps saying it.", "loy:-14;mer:-2", "{C} says something sharp and walks apart.", ""}}},
	// ---- faith ----
	{ID: "heretic", Th: "faith", Trig: "t", Need: "faith,person", Title: "The Heretic",
		Text: "A {priest} of {F} is raising a mob in {T}. They hunt a heretic who preached against {D}. {P} says he is hiding in the loft above the grain-store you passed this morning.",
		Ch: []crCh{
			{"Hide the heretic.", "", "", "mer:4;cou:3;pie:-4;rep:faith:-10;fav:-3;fu:spared_allies:8", "The loft stays quiet. The mob turns away. The heretic leaves by the river gate.", "", "", ""},
			{"Tell the {priest} where.", "", "", "pie:4;mer:-6;rep:faith:10;fav:3", "The mob goes up the stairs. You do not wait to see.", "", "", "tell"},
			{"Argue with the {priest}.", "", "persuasion:12", "pie:2;mer:2;cou:3;rep:faith:3", "A hard argument. The {priest} grumbles, and relents.", "rep:faith:-5;hp:-4", "The {priest} tells the crowd that you are a heretic too.", ""}}},
	{ID: "relic_sale", Th: "faith", Trig: "t", Need: "faith", Title: "A Relic, Cheap",
		Text: "A pilgrim offers a finger-bone of a saint of {F} for twenty {coin}, in a cloth of faded blue. It might be real. {D} might be pleased by it, or by your suspicion.",
		Ch: []crCh{
			{"Buy it.", "gold:20", "", "gold:-20;pie:3;fav:3;item:oddity", "Whether real or not, it feels like something.", "", "", ""},
			{"Examine it.", "", "lore:11", "hon:2;xp:15", "A pig's bone. You tell the pilgrim, who looks at the ground.", "gold:-10;pie:1", "You can't tell. You pay ten, to be safe.", ""},
			{"Call him a fraud.", "", "", "hon:2;pie:-2;mer:-2", "A crowd gathers, and the pilgrim leaves in a hurry.", "", "", ""}}},
	{ID: "shrine_defiled", Th: "faith", Trig: "t", Need: "faith", Title: "The Defiled Shrine",
		Text: "Someone has smeared filth across the small shrine of {D} in {T}. Three youths are laughing at it. An old woman stands before it with a rag, not wiping, only standing.",
		Ch: []crCh{
			{"Scrub it clean yourself.", "", "", "pie:4;gen:1;time:60;fav:4;rep:faith:5", "An hour, and your hands smell. The old woman presses yours briefly.", "", "", ""},
			{"Drive off the youths.", "", "intimidate:9", "pie:2;mer:-1;cou:2;fav:2", "They scatter, jeering from a distance.", "hp:-4;cou:1", "They are bigger than they looked.", ""},
			{"Join the laughter.", "", "", "pie:-6;mer:-2;fav:-4;rep:faith:-6", "It is not even funny.", "", "", ""}}},
	{ID: "sermon", Th: "faith", Trig: "t", Need: "faith", Title: "The Public Confession",
		Text: "A {priest} of {F} invites any who have wronged another to confess aloud on the temple steps. The crowd is watching. A few of them are watching you in particular, though that may be your imagination.",
		Ch: []crCh{
			{"Confess something true.", "", "", "hon:5;pie:4;fav:3;rep:faith:4;gold:-5", "It costs you a silence and a little coin. The crowd's silence afterward is almost kind.", "", "", ""},
			{"Mock the practice.", "", "", "pie:-5;cou:2;rep:faith:-6;fav:-3", "A few laugh. Most do not.", "", "", ""},
			{"Slip away.", "", "", "cou:-2", "The steps are crowded. You are not.", "", "", ""}}},
	{ID: "tithe", Th: "faith", Trig: "t", Need: "faith", Title: "The Tithe",
		Text: "The temple of {D} in {T} has sent men door to door to collect the tithe, in a year when the harvest was thin. A widow is arguing in her doorway. The collector has the book open, and a hand on the latch.",
		Ch: []crCh{
			{"Pay her tithe.", "gold:15", "", "gold:-15;gen:4;pie:2;fav:2;op:P:2", "The collector writes in the book and moves on.", "", "", ""},
			{"Argue for mercy.", "", "persuasion:10", "mer:4;rep:faith:-2;pie:-1", "He sighs and strikes it from the book, for this year.", "rep:faith:-4", "He does not even look up.", ""},
			{"Pay your own and keep walking.", "gold:5", "", "gold:-5;pie:1;gen:-1", "Your tithe is recorded. Hers is not your business.", "", "", ""}}},
	{ID: "false_prophet", Th: "faith", Trig: "t", Need: "faith", Title: "The Miracle",
		Text: "A man in white claims in {T}'s square that {D} spoke to him, and that those who give him their savings will be spared the coming fire. A hundred people nod. You see a thread run up his sleeve.",
		Ch: []crCh{
			{"Expose the trick.", "", "perception:11", "hon:5;pie:-1;cou:3;xp:20", "The thread comes out. The square goes silent, and then ugly. You leave him to it.", "hon:1;hp:-5", "The crowd turns on you instead.", ""},
			{"Say nothing, but watch.", "", "", "hon:-1", "The savings are collected. By evening he is gone.", "", "", ""},
			{"Kneel with the crowd.", "", "", "pie:2;hon:-2", "It costs nothing. You are not sure what it is worth.", "", "", ""}}},
	// ---- commerce ----
	{ID: "fair_deal", Th: "commerce", Trig: "t", Need: "person", Title: "A Fair Weight?",
		Text: "{P} sells grain by the sack to the poor of {T}. You notice, while she is distracted, that the scale's left arm is bent. Customers pay the same and take less. They are not the sort who count.",
		Ch: []crCh{
			{"Point it out to her.", "", "", "hon:3;op:P:-2", "She flushes, then fixes the scale, and the neighbourhood eats better.", "", "", "tell"},
			{"Blackmail her for a cut.", "", "persuasion:9", "gold:30;hon:-4;gen:-3;op:P:-6", "She pays, with eyes like knives.", "op:P:-8;hon:-1", "She laughs and the crowd turns to look.", ""},
			{"Tell the customers.", "", "", "hon:3;mer:2;op:P:-5;rep:realm:2", "A shouting match that lasts the afternoon.", "", "", "tell"},
			{"Buy elsewhere.", "", "", "", "You buy elsewhere. It is none of your business.", "", "", ""}}},
	{ID: "smuggler", Th: "commerce", Trig: "tr", Title: "Cargo, No Questions",
		Text: "A man at a crossing in {R} offers a hundred {coin} to carry a sealed crate to {N}, no questions, no looking. The {guard}s check the road. The crate is heavy and smells faintly of oil.",
		Ch: []crCh{
			{"Take it.", "", "", "gold:100;hon:-3;cou:2;bounty:20", "The crate crosses. You do not know what is in it.", "", "", ""},
			{"Look inside first.", "", "perception:10", "hon:1;xp:15", "Powder. You decline, and he finds someone else.", "gold:50;hon:-2;hp:-3", "A lid that bites. He takes the crate and pays you half, in apology.", ""},
			{"Refuse.", "", "", "hon:1", "He nods and melts away.", "", "", ""},
			{"Report him.", "", "", "rep:realm:8;hon:2;gold:10;fu:thief_knife:10", "The {guard}s are delighted. Smugglers have long memories.", "", "", "tell"}}},
	{ID: "rival_stall", Th: "commerce", Trig: "t", Need: "person", Title: "A Rival's Stall",
		Text: "{P} offers twenty {coin} to knock a rival's stall into the street at night. The rival is a thin woman who sells honey. {P}'s goods are slightly worse.",
		Ch: []crCh{
			{"Do the job.", "", "", "gold:20;mer:-3;hon:-3;op:P:5;bounty:10", "Honey on cobblestones. In the morning someone cries.", "", "", ""},
			{"Warn the honey seller.", "", "", "hon:3;mer:2;op:P:-4", "She thanks you with a jar.", "", "", ""},
			{"Say no and leave.", "", "", "", "Neither of them thanks you.", "", "", ""}}},
	{ID: "cheap_blade", Th: "commerce", Trig: "t", Need: "town", Title: "Too Cheap",
		Text: "A tinker offers a good blade at a third of its value, no questions. The maker's mark has been filed off. The man is sweating despite the cool morning.",
		Ch: []crCh{
			{"Buy it.", "gold:25", "", "gold:-25;item:weapon;hon:-2;bounty:15", "A good blade, if the owner doesn't come looking.", "", "", ""},
			{"Report the tinker.", "", "", "hon:3;rep:realm:4;gold:10", "The {guard}s take him away, mid-protest.", "", "", "tell"},
			{"Walk on.", "", "", "", "Not your trouble.", "", "", ""}}},
	{ID: "loan_shark", Th: "commerce", Trig: "t", Need: "person", Title: "The Loan",
		Text: "{P} presses a loan on a farmer at forty in the hundred, a week. The farmer has a sick ox and a hopeless look. {P} spots you watching, and smiles like a person with a bargain to offer.",
		Ch: []crCh{
			{"Pay the loan for the farmer.", "gold:30", "", "gold:-30;gen:4;op:P:-4;mer:2", "{P} tears up the paper, and keeps the smile.", "", "", ""},
			{"Warn the farmer.", "", "", "hon:2;mer:2;op:P:-3", "The farmer thanks you, and takes the loan anyway.", "", "", ""},
			{"Take a cut of the loan.", "", "", "gold:15;gen:-3;mer:-3;op:P:3", "A good business. It does not feel like one.", "", "", ""}}},
	{ID: "toll", Th: "commerce", Trig: "r", Need: "realm", Title: "The Toll",
		Text: "At the crossroads, two men with the colours of {R} demand a toll far beyond the posted rate. They are not quite officials. Others in the queue pay without looking up.",
		Ch: []crCh{
			{"Pay.", "gold:15", "", "gold:-15", "A few coins, and a bad taste.", "", "", ""},
			{"Argue the posted rate.", "", "persuasion:10", "hon:1;gold:-5;xp:10", "They fold at the sight of your book of laws.", "gold:-20;hp:-3", "They don't fold. You pay double.", ""},
			{"Fight your way through.", "", "intimidate:12", "cou:3;mer:-1", "They fall back muttering.", "hp:-10;spawn:bandit:1;rep:realm:-5", "They aren't alone.", ""}}},
	// ---- duty ----
	{ID: "eviction", Th: "duty", Trig: "t", Need: "realm", Title: "The Eviction",
		Text: "A captain of {R} orders you to help clear a tenement for a lord's new granary. The tenants are poor, the order is legal, and the captain has already named a fee. A girl peers over the stair at you.",
		Ch: []crCh{
			{"Obey the order.", "", "", "gold:30;rep:realm:8;mer:-5;gen:-2", "It is done by noon. The street is silent.", "", "", ""},
			{"Refuse openly.", "", "", "mer:3;cou:3;rep:realm:-8", "The captain takes your name down.", "", "", ""},
			{"Warn the tenants quietly.", "", "", "mer:5;hon:-1;cou:2;gen:1", "They are gone by the time the soldiers arrive.", "", "", "tell"}}},
	{ID: "night_watch", Th: "duty", Trig: "t", Need: "person", Title: "A Hand on the Watch",
		Text: "{P}, a {guard} short of men tonight, asks you to take a shift on the wall of {T}. The pay is poor, the night is cold, and a lantern has gone out on the west stair where people have been going missing.",
		Ch: []crCh{
			{"Take the shift.", "", "", "gold:20;time:360;rep:realm:4;cou:2;op:P:5", "A cold, slow night, and nothing happens. It is worth more than the pay.", "", "", ""},
			{"Offer a few coins instead.", "gold:10", "", "gold:-10;gen:1;op:P:1", "{P} takes it without pride.", "", "", ""},
			{"Decline politely.", "", "", "", "There are other nights.", "", "", ""}}},
	{ID: "levy", Th: "war", Trig: "tr", Need: "war", Title: "The War Levy",
		Text: "Agents of {R} are collecting grain, men and gold for the war against {W}. A village headman offers you silence and a cart of seed-corn if you will say they were never here.",
		Ch: []crCh{
			{"Pay a levy on the village's behalf.", "gold:40", "", "gold:-40;gen:4;rep:realm:6;mer:2", "The agents take it and ride on. The village sleeps.", "", "", ""},
			{"Hide the seed-corn.", "", "", "hon:-3;mer:3;rep:realm:-6;cou:2", "The agents find only empty barns.", "", "", "lie"},
			{"Join the levy.", "", "", "rep:realm:10;mer:-3;gen:-2;gold:30", "The agents are thrilled, and the village is thin by spring.", "", "", ""}}},
	{ID: "messenger", Th: "duty", Trig: "rc", Title: "The Dying Messenger",
		Text: "A rider lies on the verge, an arrow in his side, a sealed letter in his fist. 'To {N},' he whispers. 'To the {elder}.' Then he dies. The seal is the colour of {R}.",
		Ch: []crCh{
			{"Deliver it unopened.", "", "", "hon:4;time:300;rep:realm:5;gold:20;xp:20", "A long walk to {N}. The {elder} reads it with a face of stone, then pays you.", "", "", ""},
			{"Read it first.", "", "", "item:book;hon:-2;xp:10", "Names, dates, and a plot you were not meant to know.", "", "", "tell"},
			{"Sell it to the highest bidder.", "", "", "gold:50;hon:-5;rep:realm:-5", "Someone pays well for what you don't understand.", "", "", ""}}},
	{ID: "border_check", Th: "duty", Trig: "r", Need: "realm", Title: "Border Search",
		Text: "Soldiers of {R} are searching travellers. They find nothing on you, but a frightened woman behind you is carrying a child under her cloak, with the {guard}s only two travellers away.",
		Ch: []crCh{
			{"Create a distraction.", "", "acrobatics:10", "mer:4;cou:3;rep:realm:-3", "A dropped bag, a shout, a grateful look. She slips through.", "hp:-3;rep:realm:-5", "You are the one they search, and fine.", ""},
			{"Say nothing.", "", "", "cou:-2", "She is taken aside.", "", "", ""},
			{"Point her out.", "", "", "rep:realm:5;mer:-6;hon:-1", "It is the law, you tell yourself.", "", "", "tell"}}},
	// ---- family ----
	{ID: "letter_home", Th: "family", Trig: "t", Need: "town", Title: "A Letter from Home",
		Text: "A courier in {T} finds you with a folded letter. It is from your kin. It is short and polite, and ends: 'We need fifty {coin} before the new moon.' There is no signature, only a smear of ink where a thumb pressed.",
		Ch: []crCh{
			{"Send what they ask.", "gold:50", "", "gold:-50;gen:4;mer:2", "A quiet debt, discharged.", "", "", ""},
			{"Send a little.", "gold:15", "", "gold:-15;gen:1", "It will have to do.", "", "", ""},
			{"Burn the letter.", "", "", "gen:-4;mer:-2", "The ink takes a long time to go.", "", "", ""}}, Dec: "gen:-2"},
	{ID: "orphan", Th: "family", Trig: "t", Need: "town", Title: "A Child Who Claims You",
		Text: "A ten-year-old at the gate of {T} announces that you are their cousin, with a story that is nearly believable. They are hungry, they know no trade, and they are being sheltered by no one.",
		Ch: []crCh{
			{"Take them to the temple kitchen and pay for a month.", "gold:20", "", "gold:-20;mer:4;gen:3;pie:1;fu:thief_back:12", "They eat in silence and watch you leave.", "", "", ""},
			{"Test the story.", "", "empathy:10", "hon:2;mer:2;xp:15", "A lie, but a loving one. You tell them so, kindly, and give them a meal.", "mer:-1", "You can't tell. You turn your back and feel it all day.", ""},
			{"Send them away.", "", "", "mer:-4;gen:-2", "A cold word. The child goes without crying.", "", "", ""}}},
	{ID: "feud", Th: "family", Trig: "t", Need: "person,rival", Title: "The Feud",
		Text: "{P} and {O} have not spoken for a year, over a boundary stone. Today each asks you, separately, to speak for them at the {elder}'s court in {T}. Both are decent people. Both are wrong a little.",
		Ch: []crCh{
			{"Mediate between them.", "", "persuasion:11", "mer:3;hon:2;op:P:4;op:O:4;xp:25", "An afternoon of sullen negotiation. By sunset, a handshake.", "op:P:-2;op:O:-2", "Both leave angrier than they came.", ""},
			{"Side with {P}.", "", "", "op:P:6;op:O:-6;hon:-1", "{P} is grateful. {O} says nothing.", "", "", ""},
			{"Side with {O}.", "", "", "op:O:6;op:P:-6;hon:-1", "{O} is grateful. {P} says nothing.", "", "", ""}}},
	// ---- plague ----
	{ID: "quarantine", Th: "plague", Trig: "tr", Need: "plague", Title: "The Shut Gate",
		Text: "The gate of {N} is shut against {H}. A line of travellers waits in the rain; some are coughing. A gate-warden calls through the grill that no one passes, though a few coins have been seen to open a smaller door.",
		Ch: []crCh{
			{"Bribe the warden.", "gold:30", "", "gold:-30;hon:-3;rep:realm:-3;hp:-3", "The small door opens. You step into the stink of a town in grief.", "", "", ""},
			{"Wait with the others.", "", "", "time:480;cou:1", "A long wet day. The gate does not open.", "", "", ""},
			{"Leave herbs and bread at the gate.", "item:food", "", "lose:food;gen:3;mer:3;fav:2", "The warden takes them, silently.", "", "", ""}}},
	{ID: "sick_pilgrim", Th: "plague", Trig: "rc", Need: "plague", Title: "A Pilgrim Who Coughs",
		Text: "A pilgrim stumbles into your camp with a flush on her cheeks and the early signs of {H}. She asks only for water and a place by the fire. Others on the road would not let her near them.",
		Ch: []crCh{
			{"Share a tonic and the fire.", "item:tonic", "", "lose:tonic;mer:6;gen:3;hp:-3;pie:1", "She sleeps well. In the morning she is a little better, and you are a little worse.", "", "", ""},
			{"Give her water and send her on.", "", "", "mer:-1;gen:1", "She nods, and goes.", "", "", ""},
			{"Drive her off.", "", "", "mer:-6;cou:-1", "She goes with a look you carry for days.", "", "", ""}}},
	{ID: "pyre", Th: "plague", Trig: "t", Need: "plague", Title: "The Unburied",
		Text: "In the poor quarter of {T}, bodies lie where they fell from {H}. The kin are sick or gone. A single {priest} is trying to build a pyre alone, and the smoke is thick.",
		Ch: []crCh{
			{"Help build the pyre.", "", "endurance:9", "pie:4;mer:3;cou:3;fav:3;time:180", "A grim afternoon of work. The {priest} says a few words, and you add yours.", "hp:-6;pie:2;mer:2", "The smoke and the work leave you coughing blood for a day.", ""},
			{"Pay gravediggers.", "gold:25", "", "gold:-25;gen:3;mer:1", "Three men take the coin, and the work gets done.", "", "", ""},
			{"Give the quarter a wide berth.", "", "", "cou:-2", "You go around.", "", "", ""}}},
	{ID: "plague_doctor", Th: "plague", Trig: "t", Need: "plague", Title: "The {doc}'s Request",
		Text: "A {doc} in {T} presses a leather satchel on you, full of tinctures for {H}. He asks you to carry it to {N}, where the sick are many. He cannot leave his patients.",
		Ch: []crCh{
			{"Deliver the satchel.", "", "", "gen:3;mer:3;time:300;rep:realm:3;item:tonic;xp:25", "A hard road, a grateful stranger, and a thin bottle pressed into your hand.", "", "", ""},
			{"Sell the tinctures.", "", "", "gold:60;gen:-4;mer:-3;hon:-3", "A good sum, and the memory of the doctor's face.", "", "", ""},
			{"Refuse.", "", "", "", "He nods, a man who expected no better.", "", "", ""}}},
	// ---- war and famine ----
	{ID: "refugees", Th: "war", Trig: "r", Need: "war", Title: "Refugees on the Road",
		Text: "Families of {W} trudge along the road, carrying bundles and children. They are footsore and afraid, and some have been robbed. A ragged man asks if you will guide them to the border post at {N}.",
		Ch: []crCh{
			{"Escort them.", "", "", "mer:4;cou:2;time:420;rep:realm:-2;fu:refugee_kin:9;xp:20", "A slow journey. At the border post, a child hands you a carved bird.", "", "", ""},
			{"Give food and wish them well.", "item:food", "", "lose:food;gen:3;mer:2", "They share it among themselves, carefully.", "", "", ""},
			{"Rob them while they are weak.", "", "", "gold:20;mer:-8;gen:-4;hon:-3", "They have almost nothing. You take it.", "", "", ""},
			{"Pass by.", "", "", "mer:-1", "The road is long.", "", "", ""}}},
	{ID: "grain_hoard", Th: "famine", Trig: "t", Need: "famine,person", Title: "The Hoard",
		Text: "While {T} starves, {P} sits on a barn of grain, waiting for the price to rise. The gate is guarded by a single boy with a stick. Hundreds go hungry within a hundred paces.",
		Ch: []crCh{
			{"Break in and distribute it.", "", "streetwise:11", "gen:6;mer:3;hon:-1;op:P:-8;rep:realm:-3", "By dawn the barn is empty and the town is full of bread.", "bounty:25;hp:-4;gen:1", "The boy screams and the {guard}s come running.", ""},
			{"Buy the grain at his price (sixty {coin}).", "gold:60", "", "gold:-60;gen:5;mer:3;op:P:2;rep:realm:3", "He sells smiling. The grain goes out the same day.", "", "", ""},
			{"Report him to the {guard}s.", "", "persuasion:11", "hon:3;rep:realm:5;op:P:-6;mer:2", "The {guard}s seize the barn, and the price falls.", "hon:1;op:P:-3", "They are his friends.", "tell"},
			{"Do nothing.", "", "", "mer:-3;gen:-1", "The hunger remains.", "", "", ""}}},
	{ID: "press_gang", Th: "war", Trig: "t", Need: "war", Title: "The Press-Gang",
		Text: "Recruiters of {R} seize a boy from a baker's stall in {T}, to fill the line against {W}. His mother is held back by two soldiers. The boy is terrified; the sergeant, bored.",
		Ch: []crCh{
			{"Buy the boy's freedom (thirty {coin}).", "gold:30", "", "gold:-30;mer:4;gen:3;rep:realm:-3", "The sergeant counts the coin and releases the boy to his mother.", "", "", ""},
			{"Vouch for the boy as your servant.", "", "persuasion:11", "mer:4;hon:-2", "The sergeant waves a hand. The boy trots over, wide-eyed.", "rep:realm:-3", "The sergeant is not fooled.", "lie"},
			{"Let it happen.", "", "", "mer:-2;rep:realm:2", "The cart rolls on.", "", "", ""}}},
	{ID: "foragers", Th: "famine", Trig: "rc", Need: "famine", Title: "The Starving Camp",
		Text: "A cold camp by the road holds thirty people from villages hit by famine in {R}. A woman asks if you have anything: bread, a few coins, a word of where to go. The fire is nearly out.",
		Ch: []crCh{
			{"Share your food.", "item:food", "", "lose:food;gen:5;mer:3", "It goes quickly, and nobody says thank you, because they are eating.", "", "", ""},
			{"Tell them where to find help.", "", "", "gen:1;hon:1", "They nod. You are not sure it's true.", "", "", ""},
			{"Offer to hunt for them tonight.", "", "survival:10", "gen:4;mer:3;time:300;xp:25", "A long night and a deer. The fire burns until morning.", "hp:-3;gen:2", "You find nothing but tracks.", ""}}},
	// ---- revenge, secrets ----
	{ID: "smear", Th: "revenge", Trig: "t", Need: "person,rival", Title: "A Whisper Campaign",
		Text: "{O} offers you fifty {coin} to spread a rumour that {P} has been cheating the {elder}'s tax-roll. It is not true. It would ruin {P} anyway.",
		Ch: []crCh{
			{"Take the money and spread it.", "", "", "gold:50;hon:-6;mer:-3;op:P:-8;op:O:5;news:Rumours say {P} of {T} has been cheating the tax-roll.", "By next week it will be the only thing anyone says about {P}.", "", "", "lie"},
			{"Refuse and warn {P}.", "", "", "hon:3;op:P:6;op:O:-5", "{P} thanks you with a stiff nod, understanding more than they say.", "", "", "tell"},
			{"Decline and say nothing.", "", "", "", "A coin unspent.", "", "", ""}}},
	{ID: "old_wrong", Th: "revenge", Trig: "d", Need: "person", Title: "Old Wrongs",
		Text: "{P} tells you over a cup that the man who killed their brother ten years ago lives in {N} under another name. 'I'd go myself,' they say, 'if I weren't a coward.' They look at you, then look at the cup.",
		Ch: []crCh{
			{"Hunt the man down.", "", "", "mer:-4;cou:2;op:P:8;time:600;fu:spared_vendetta:6", "You find a tired old man and a sharp knife. It is done. It does not feel the way {P} thought.", "", "", ""},
			{"Say the law should judge him.", "", "", "hon:2;mer:2;op:P:-2", "{P} nods and drinks.", "", "", ""},
			{"Tell {P} to let it go.", "", "persuasion:10", "mer:3;op:P:2;xp:15", "A long conversation. {P} puts the cup down at last.", "op:P:-3", "{P} stares at you, then leaves the table.", ""}}},
	{ID: "blackmail", Th: "secrets", Trig: "t", Need: "person,secret", Title: "A Shameful Secret",
		Text: "By chance you learn something about {P}: {PS}. It is the sort of thing a person would pay to keep hidden, and your purse is not as heavy as you'd like.",
		Ch: []crCh{
			{"Blackmail {P}.", "", "persuasion:9", "gold:60;hon:-5;mer:-3;op:P:-10;fu:black_back:8", "{P} pays, trembling with a look you will see again.", "gold:10;op:P:-6;hon:-2", "{P} laughs and tells you to do your worst.", "tell"},
			{"Keep the secret.", "", "", "hon:2", "It sits in your head. It will stay there.", "", "", ""},
			{"Tell {P} what you know, and that you won't use it.", "", "", "hon:4;op:P:6;mer:1", "{P} blinks, and the fear drains away.", "", "", "tell"},
			{"Expose it.", "", "", "hon:-1;mer:-3;op:P:-12;news:It is whispered in {T} that {P}: {PS}.", "By tomorrow, everyone will know.", "", "", "tell"}}},
	{ID: "lost_letters", Th: "secrets", Trig: "t", Title: "The Lost Satchel",
		Text: "A satchel lies in the road in {T}, full of letters tied with ribbon. Love letters, from the look of the hand, and from the address, ones that should not exist. A name on the cover is known to you.",
		Ch: []crCh{
			{"Return it unread.", "", "", "hon:4;mer:1;gold:5", "The owner takes it with trembling hands and says nothing for a long time.", "", "", ""},
			{"Read them first.", "", "", "item:book;hon:-2;xp:10", "A great deal more than you wanted to know.", "", "", "tell"},
			{"Sell them to a rival.", "", "", "gold:45;hon:-5;mer:-3", "Someone pays well to damage someone else.", "", "", ""}}},
	{ID: "plot", Th: "secrets", Trig: "tr", Need: "realm", Title: "Overheard",
		Text: "At an inn, through a thin wall, you hear three voices plan an act against {Ruler} of {R}: a poisoned cup at the next feast. One voice is certain. One is afraid. One is silent.",
		Ch: []crCh{
			{"Report them to the {guard}s.", "", "", "hon:3;rep:realm:12;mer:-2;gold:30", "The {guard}s act quickly. The afraid voice weeps all night.", "", "", "tell"},
			{"Warn the afraid one to leave town.", "", "persuasion:10", "mer:3;cou:2", "A quiet word, and a coach leaves before dawn.", "mer:-1", "The silent voice sees you.", "tell"},
			{"Join them.", "", "", "rep:realm:-15;hon:-2;cou:2;rep:out:8", "A stranger among strangers, in a plot with no clear end.", "", "", "lie"},
			{"Say nothing.", "", "", "cou:-2", "You sleep poorly.", "", "", ""}}},
	{ID: "ideology", Th: "secrets", Trig: "t", Need: "move", Title: "The Pamphlet",
		Text: "Someone presses a pamphlet from the {Mv} into your hand in {T}: a call for reform in {R}. The {guard}s are tearing others from hands nearby. A student begs you to carry it to {N}.",
		Ch: []crCh{
			{"Carry the pamphlets.", "", "", "cou:4;rep:realm:-6;mer:1;time:200", "A few copies, hidden in a hat. In {N} someone takes them with glowing eyes.", "", "", ""},
			{"Hand them over to the {guard}s.", "", "", "rep:realm:6;mer:-3;hon:1", "The student's face is not one you'll forget.", "", "", "tell"},
			{"Refuse politely.", "", "", "", "A shrug, and the student moves on.", "", "", ""}}},
	{ID: "monster_bounty", Th: "duty", Trig: "tr", Need: "monster", Title: "The Beast Near the Road",
		Text: "Villagers beg you to hunt {M}, which has taken livestock and a child's pet. The beast is old and very strong. They offer sixty {coin} and nothing else, though the headman's eyes say more.",
		Ch: []crCh{
			{"Take the contract.", "", "", "cou:4;gold:30;rep:realm:3;xp:30", "You promise to look. {M} will not wait for a promise.", "", "", ""},
			{"Offer to guard the village for a night.", "", "", "cou:2;gen:2;time:420", "A long night and a quiet one.", "", "", ""},
			{"Say it isn't your trouble.", "", "", "cou:-3", "The headman nods, a man used to this.", "", "", ""}}},
	{ID: "feast_toast", Th: "justice", Trig: "t", Need: "feast", Title: "The Toast",
		Text: "At the {X} in {T} the {elder} offers a cup to every guest and asks each to toast the realm. The wine is thin and the oath is thick; a hand rests on a sword at each guest's shoulder.",
		Ch: []crCh{
			{"Raise the cup, and mean it.", "", "", "rep:realm:6;pie:1", "The crowd drinks. For a moment you are almost one of them.", "", "", ""},
			{"Raise the cup, lie in your teeth.", "", "", "rep:realm:4;hon:-3", "No one can tell.", "", "", "lie"},
			{"Decline.", "", "", "cou:3;rep:realm:-6", "A silence ripples out and doesn't stop.", "", "", ""}}},
	// ---- genre-skinned ----
	{ID: "g_fantasy", Th: "justice", Trig: "r", Gn: "fantasy", Title: "The Troll's Toll",
		Text: "A troll the size of a barn squats under the bridge near {T}. It wants a coin for every cart, and a story for every traveller. It does not seem to want to hurt anyone, but the bridge is the only one for a day.",
		Ch: []crCh{
			{"Pay the coin and tell a tale.", "gold:2", "", "gold:-2;hon:1;cou:1;xp:15", "It listens with a head cocked. It cries at the end.", "", "", ""},
			{"Trick it.", "", "bluff:11", "hon:-2;cou:3;xp:20", "You cross unpaid. The troll does not seem to mind.", "hp:-6;cou:1", "It picks you up by one ankle and drops you in the river.", "lie"},
			{"Draw steel.", "", "", "cou:3;mer:-3;spawn:bandit:0;hp:-6", "A brutal fight. You win, and feel no better.", "", "", ""}}},
	{ID: "g_mythic", Th: "faith", Trig: "t", Gn: "mythic", Title: "A Guest at the Door",
		Text: "At dusk an old beggar knocks at the door of the house where you lodge in {T}. The master pretends not to hear. By xenia, one must not turn a stranger away; {D} walks in beggars' clothing, and takes offence slowly, and long.",
		Ch: []crCh{
			{"Welcome him in and give him your place.", "", "", "gen:4;pie:5;fav:4;mer:3", "He eats little, blesses the roof, and is gone at dawn.", "", "", ""},
			{"Test him with a riddle.", "", "lore:11", "pie:2;xp:25", "He smiles, answers, and asks one back.", "pie:-1;hp:-3", "He answers, and you do not.", ""},
			{"Close the door.", "", "", "pie:-6;mer:-4;fav:-5", "You hear the wind rise in the night.", "", "", ""}}},
	{ID: "g_steampunk", Th: "duty", Trig: "t", Gn: "steampunk", Title: "The Overpressure",
		Text: "At a factory in {T} the boiler gauge is in the red, and the foreman will not vent it: the line must run to meet the quota. Forty workers stand close. A boy tries to catch your eye.",
		Ch: []crCh{
			{"Vent the boiler yourself.", "", "tinkering:10", "cou:4;mer:4;op:P:-3;xp:25", "Steam howls. The foreman rages, and then breathes.", "hp:-10;cou:2", "The valve sticks, then gives, scalding your arm.", ""},
			{"Tell the foreman he is a fool.", "", "persuasion:11", "mer:2;hon:2;xp:15", "He pales, then nods, and shouts an order.", "", "", "tell"},
			{"Leave. Quotas are quotas.", "", "", "mer:-4;cou:-2", "You are two streets away when it goes.", "", "", ""}}},
	{ID: "g_dieselpunk", Th: "justice", Trig: "t", Gn: "dieselpunk", Title: "The Picket Line",
		Text: "A strike has closed the foundry gates of {T}. Men with clubs stand shoulder to shoulder. A foreman at the gate offers you three days of a rough wage to cross their line and fill an empty post.",
		Ch: []crCh{
			{"Cross the line.", "", "", "gold:45;mer:-3;gen:-2;rep:realm:3", "They spit as you pass. The wage is real.", "", "", ""},
			{"Stand with the strikers.", "", "", "mer:2;cou:3;gen:2;rep:realm:-4", "A cold night around a barrel fire, and someone offers you stew.", "", "", ""},
			{"Mediate.", "", "persuasion:12", "hon:3;xp:30;gold:20", "A fragile deal and a hundred grateful men.", "hp:-4", "A brick flies over your head and the talks collapse.", ""}}},
	{ID: "g_atompunk", Th: "mercy", Trig: "t", Gn: "atompunk", Title: "Shelter Lottery",
		Text: "The sirens of {T} are rehearsal, they say, but the shelter inspector is deadly serious: the community shelter has room for a hundred and three hundred have come. He holds a drum of numbered tokens and a revolver.",
		Ch: []crCh{
			{"Give up your token.", "", "", "mer:5;gen:4;cou:2", "A child takes it. You do not know whether you'll regret it.", "", "", ""},
			{"Take the token and go in.", "", "", "mer:-3;gen:-2", "The door closes behind you. The wind outside is only wind.", "", "", ""},
			{"Argue the numbers.", "", "persuasion:12", "hon:2;mer:2;xp:20", "A dozen more places are found, behind a locked door.", "hp:-3", "The inspector's patience ends at the muzzle.", ""}}},
	{ID: "g_cyberpunk", Th: "secrets", Trig: "t", Gn: "cyberpunk", Title: "The Fixer's Chip",
		Text: "A dying runner in {T} presses a data-chip into your hand: the memories of a woman who knew too much about a corpsec cover-up. 'Sell it. Burn it. Give it to her mother.' He gives no more instruction than that.",
		Ch: []crCh{
			{"Take it to her mother.", "", "", "hon:3;mer:3;gen:2;rep:out:5;xp:25", "The old woman watches the chip run, and then simply holds your wrist.", "", "", ""},
			{"Sell it to the highest bidder.", "", "", "gold:120;hon:-4;gen:-3;bounty:20", "Someone pays more than it is worth.", "", "", ""},
			{"Wipe it.", "", "", "hon:-1;cou:1", "A clean blank, and no one the wiser.", "", "", ""}}},
	{ID: "g_biopunk", Th: "mercy", Trig: "t", Gn: "biopunk", Title: "The Vat-Born",
		Text: "A grown thing slumps in an alley of {T}, half-human, half something the vats got wrong. It looks up with a child's eyes and says one word: 'Please.' The {doc}s are paid to collect such things, and a few carry knives.",
		Ch: []crCh{
			{"Hide it and tend it.", "", "empathy:10", "mer:6;cou:3;rep:realm:-3;xp:25", "It sleeps against your pack. By morning it is stronger.", "mer:3;hp:-3", "It is too far gone; it dies gently, holding your hand.", ""},
			{"Put it out of its pain.", "", "", "mer:-1;cou:1;pie:1", "It is over in a moment.", "", "", ""},
			{"Sell it to the {doc}s.", "", "", "gold:80;mer:-6;gen:-2", "They take it away in a wet sack.", "", "", ""}}},
	{ID: "g_solarpunk", Th: "justice", Trig: "t", Gn: "solarpunk", Title: "The Commons Vote",
		Text: "A drought has come to the commons of {T}. The water council must allot what the cistern holds between the orchard and the school garden. You are asked to cast the deciding vote, as an outsider with no stake.",
		Ch: []crCh{
			{"Vote for the orchard.", "", "", "gen:1;mer:-1;rep:realm:2", "The orchard lives. The school garden browns.", "", "", ""},
			{"Vote for the school garden.", "", "", "gen:2;mer:2;rep:realm:2", "The children plant on the dry earth with their hands.", "", "", ""},
			{"Propose to share and ration both.", "", "persuasion:11", "gen:3;hon:2;xp:30", "A bitter compromise, and everyone gets half.", "gen:1", "The council shouts you down.", ""}}},
	{ID: "g_wasteland", Th: "justice", Trig: "r", Gn: "wasteland", Title: "Water Rights",
		Text: "At a well in the wastes, a dozen settlers and a dozen raiders face off across the stones. Neither can drink without the other letting them. You hold a rifle and a canteen and the only fresh face for forty miles.",
		Ch: []crCh{
			{"Broker a split.", "", "persuasion:12", "hon:3;mer:3;cou:2;xp:30", "A tense hour of haggling and an uneasy dawn.", "hp:-6;cou:1", "A shot rings out, and then several.", ""},
			{"Fight beside the settlers.", "", "", "cou:3;mer:-1;rep:out:-10;spawn:bandit:2", "A dozen raiders and a few corpses, and the well is theirs.", "", "", ""},
			{"Sell water to both.", "", "", "gold:40;gen:-3;hon:-1", "Everyone pays you. Nobody likes you.", "", "", ""}}},
	{ID: "g_twilight", Th: "faith", Trig: "cr", Gn: "twilight", Title: "A Pale Bargain",
		Text: "At the edge of {T}'s lantern-light a tall, pale figure offers you a season of life in exchange for a cup of your blood. It stands very still. Its voice is the temperature of a cellar.",
		Ch: []crCh{
			{"Accept.", "", "", "hp:-6;pie:-5;cou:2;item:oddity", "The cup is warm for a long minute. The figure bows, and is gone.", "", "", ""},
			{"Refuse, and keep walking.", "", "", "pie:2;cou:2", "It watches you go, and does not follow.", "", "", ""},
			{"Bargain.", "", "persuasion:12", "xp:40;cou:3;hon:-1", "It names a trifle, and a promise. You keep both.", "hp:-10;pie:-2", "It takes more than you offered.", ""}}},
	// ---- follow-ups ----
	{ID: "thief_back", Th: "mercy", Fu: true, Title: "The Child Returns",
		Text: "At a bend in the road near {T}, a figure steps out: the child you once helped, older now and carrying a bundle. They hold it out. 'I heard you were looking for something,' they say. 'I found it.'",
		Ch: []crCh{
			{"Accept the gift.", "", "", "gen:1;item:loot;mer:2;xp:15", "A small, good thing, given with pride.", "", "", ""},
			{"Refuse, and press coin on them.", "gold:10", "", "gold:-10;gen:3;mer:2", "They keep the coin and the bundle, and leave grinning.", "", "", ""}}, Dec: "gen:-1"},
	{ID: "thief_knife", Th: "revenge", Fu: true, Title: "A Knife in the Crowd",
		Text: "Someone in the crowd near {T} knows your face. Their hand reaches for the knife at their belt: it is the person you turned in, or someone near to them. Their eyes are not cold, only tired.",
		Ch: []crCh{
			{"Disarm them.", "", "acrobatics:10", "cou:3;mer:1;xp:20", "A quick wrench and the knife is on the floor. They sag.", "hp:-8;mer:-1", "The knife finds your side.", ""},
			{"Talk them down.", "", "persuasion:11", "mer:3;hon:1;xp:20", "A long minute. Then a crumpled sob.", "hp:-5", "They cut you, then run.", ""},
			{"Strike first.", "", "", "mer:-5;cou:1;spawn:bandit:0", "It is over fast. It was a mistake.", "", "", ""}}},
	{ID: "spared_allies", Th: "mercy", Fu: true, Title: "Debts Repaid",
		Text: "A hard knock at your door-flap, in {R}. Those you spared: three of them, armed and tired, with a sack. 'We owe a life,' says one. 'We didn't forget it.' They put the sack on the ground.",
		Ch: []crCh{
			{"Accept their help.", "", "", "gold:50;item:loot;rep:out:5;xp:25", "It is a good haul, and a better memory.", "", "", ""},
			{"Refuse and thank them.", "", "", "mer:2;pie:1", "They bow, and leave the sack.", "", "", ""}}},
	{ID: "spared_vendetta", Th: "revenge", Fu: true, Title: "A Vendetta",
		Text: "The one you left alive did not forgive. He waits at a bend in the road near {N}, with friends behind him. 'I said I would find you,' he says, 'and I did.'",
		Ch: []crCh{
			{"Fight.", "", "", "cou:3;spawn:bandit:2;hp:-3", "It is a short and ugly fight.", "", "", ""},
			{"Try to end it with words.", "", "persuasion:12", "mer:3;hon:2;xp:30", "He lowers the blade. Neither of you trusts it.", "hp:-10;spawn:bandit:1", "Words do not hold him. They never did.", ""},
			{"Pay him off.", "gold:50", "", "gold:-50;cou:-2", "The matter is settled, or is at least deferred.", "", "", ""}}},
	{ID: "freed_beast", Th: "mercy", Fu: true, Title: "A Wild Thing Returns",
		Text: "Something large keeps pace with you at the edge of the trees near {T}. It does not close. When you stop, it stops. At last it lays down at the edge of the path a gift: the carcass of a fine deer.",
		Ch: []crCh{
			{"Accept the gift.", "", "", "mer:2;pie:1;item:food;xp:20", "It watches you take it, and then vanishes.", "", "", ""},
			{"Bow, and leave it be.", "", "", "mer:3;pie:2", "It makes a low sound and is gone.", "", "", ""}}},
	{ID: "black_back", Th: "revenge", Fu: true, Title: "The Blackmailed",
		Text: "A letter is waiting for you at an inn near {T}, from {P}: they know you hold their secret, and they would like to meet to 'settle things privately', on a country road, after dark.",
		Ch: []crCh{
			{"Go armed.", "", "", "cou:2;spawn:bandit:2;hp:-3", "{P} brought friends. So did you, in a sense.", "", "", ""},
			{"Go alone and apologise.", "", "", "hon:3;mer:2;op:P:8;gold:-30", "{P} listens. A long silence. They take the coin, but are softened.", "", "", ""},
			{"Do not go.", "", "", "cou:-3", "The letter burns.", "", "", ""}}},
	{ID: "deserter_thanks", Th: "mercy", Fu: true, Title: "A Visit from the Deserter",
		Text: "A tall young man stands at the roadside near {N}, dressed in honest clothes with a plough-callus on his palm. 'You might not remember me,' he says. 'I wanted you to see that I lived.'",
		Ch: []crCh{
			{"Share a meal.", "", "", "mer:3;gen:2;item:food", "A short meal. He talks of his first harvest.", "", "", ""},
			{"Nod and go on.", "", "", "mer:1", "A good thing, in a hard year.", "", "", ""}}},
	{ID: "refugee_kin", Th: "mercy", Fu: true, Title: "Word from the Border",
		Text: "A stranger finds you in {T} with a message: the people you escorted have settled near {N}, and a woman there has named her newborn after you. They have a place for you at their table.",
		Ch: []crCh{
			{"Visit them.", "", "", "mer:3;gen:2;time:300;item:food;xp:30", "A table too small and a welcome too large.", "", "", ""},
			{"Send a gift.", "gold:10", "", "gold:-10;gen:3", "A small thing, and gladly given.", "", "", ""}}},
}
