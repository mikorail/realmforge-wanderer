package engine

// Wayfarer templates (prefix wf). Effect strings are the crossroads ones (mer hon cou gen pie karma, gold hp time rep
// op item lose xp news deed bounty) plus: coin:+n|-n (units of local money scaled by level), hostile, foe:<tpl>:<n>,
// clue (nearest site hint, noted in the case notes), tip (the same hint, journal only), map (a map item and the hint),
// rumour:<text> (queued through saQueueRumour), job (the template's errand: escort parcel message grave).
// Placeholders: the crossroads set ({T} {N} {R} {Ruler} {W} {F} {D} {coin} {bandit} {guard} ...) and {E} {Et} (who
// speaks), {Dest} {DestR} (nearest other town and its realm), {Gr} {Pv} {Cu} (greeting, proverb and name of the local
// culture pack voice), {Pg} {Pgr} {Pgc} {Pgy} {PgT} (a real dead notable), {Od1} {Od2}, {Rd} {Ra}, {Fr}.

var wfOdds = []string{"a brass bird that sings when it is cold", "a bottle of river-fog", "a map of a town that was never built", "three buttons from a dead man's coat",
	"a lamp that burns without oil", "a knot of seven different cords", "a tooth set in silver", "a clay whistle that only dogs answer",
	"a compass that points at the nearest quarrel", "a spoon that is always slightly warm", "a jar of someone's last laugh", "a glass eye with a clear iris"}

var wfRiddles = [][2]string{
	{"What walks with you from dawn but leaves no tracks, and grows longer as the sun sets low?", "your shadow"},
	{"I have cities but no houses, rivers but no water, roads but no feet. What am I?", "a map"},
	{"The more of me you take, the more you leave behind.", "footsteps"},
	{"I speak without a mouth and answer when you call, though you hear me only in empty places.", "an echo"},
	{"Whole I can hold nothing, broken I hold a whole life's promise.", "an egg"},
	{"What breaks the moment you say its name?", "silence"},
	{"I am light as a feather, yet the strongest cannot hold me for long.", "breath"},
	{"Everyone is given me at birth and no one keeps me at death. I am spent and never saved.", "time"},
}

func wfPrepOdds(g *Game, c map[string]string, r *Rng) bool {
	a := r.Int(0, len(wfOdds)-1)
	b := (a + 1 + r.Int(0, len(wfOdds)-2)) % len(wfOdds)
	c["Od1"], c["Od2"] = wfOdds[a], wfOdds[b]
	return true
}

func wfPrepRiddle(g *Game, c map[string]string, r *Rng) bool {
	x := wfRiddles[r.Int(0, len(wfRiddles)-1)]
	c["Rd"], c["Ra"] = x[0], x[1]
	return true
}

func wfPrepFront(g *Game, c map[string]string, r *Rng) bool {
	c["Fr"] = c["W"]
	if c["~W"] != "" || c["Fr"] == "" {
		c["Fr"] = "raiders from the hills"
	}
	return true
}

func wfPrepDead(g *Game, c map[string]string, r *Rng) bool { return wfDead(g, c, r) }

var wfTpls = []wfTpl{
	{ID: "peddler", Title: "A Peddler With Odd Stock", Prep: wfPrepOdds, Weight: 2,
		Cast: []wfCast{{Title: "peddler", Glyph: '@', F: CYellow, Host: true}},
		Text: "A peddler sits on a roll of canvas beside the road, with a tray of things that do not belong together: {Od1}, {Od2}, and a jar labelled in a script nobody reads. {E} says each one has a story and every story has a price. {Gr}",
		Ch: []crCh{
			{"Buy something that hums.", "", "", "coin:-2;item:oddity;xp:10", "It hums for a day and then stops. It is still the strangest thing you own.", "", "", ""},
			{"Haggle for the jar.", "", "persuasion:10", "coin:-1;item:tonic;item:oddity;xp:12", "{E} sighs theatrically and lets it go for the price of a meal. The jar is warm.", "coin:-2", "{E} laughs at your offer and raises the price while you stand there.", ""},
			{"Ask what they have heard on the road.", "", "", "xp:5;clue", "'Folk talk, and I listen,' says {E}, and names a place the locals avoid.", "", "", ""}}},
	{ID: "lost_child", Title: "A Lost Child", Weight: 2,
		Cast: []wfCast{{Title: "lost child", Glyph: '@', F: CWhite, Host: true}},
		Text: "A small figure sits on a milestone, chewing a sleeve. {E} cannot say whose child they are, only that the cart left without them and that {Dest} is where the cart was going.",
		Job:  "escort", JobTitle: "Bring a lost child to {Dest}", JobText: "{E} was left behind on the road. Their family was bound for {Dest}.", JobEff: "coin:+3;rep:realm:+4;mer:3;xp:30", JobDays: 4,
		Ch: []crCh{
			{"Take them to {Dest}.", "", "", "job;mer:4;gen:2", "{E} takes your hand without a word and keeps pace, which is a promise of its own.", "", "", ""},
			{"Coax out which way home lies.", "", "empathy:9", "mer:3;xp:15;coin:+1", "Their pointing leads you to a farmstead where a mother runs out across the field. She presses what she has into your hands.", "mer:1", "They cry and cannot say. You leave them a crust and the sinking feeling you should have done more.", ""},
			{"Give them food and go on.", "item:food", "", "lose:food;mer:1;gen:2", "{E} eats like a wolf cub. You go on, and look back once.", "", "", ""}}},
	{ID: "pilgrim", Title: "A Footsore Pilgrim", Weight: 2,
		Cast: []wfCast{{Title: "pilgrim", Glyph: '@', F: CCyan, Host: true}},
		Text: "{E} sits by the roadside with their sandals in their lap and both feet in ribbons. {Gr} They are bound for {Dest} to give thanks to {D}, and they will not get there alone. 'The road is part of the prayer,' they say, 'but not all of it.'",
		Job:  "escort", JobTitle: "See a pilgrim to {Dest}", JobText: "{E} walks to {Dest} to give thanks to {D}.", JobEff: "fav:3;rep:faith:+6;pie:4;coin:+2;xp:30", JobDays: 5,
		Ch: []crCh{
			{"Walk with them to {Dest}.", "", "", "job;pie:2;mer:2", "They lean on your arm and begin to sing, off-key and happy.", "", "", ""},
			{"Bind their feet and send them on.", "item:food", "", "lose:food;mer:3;pie:2;xp:15", "You wrap the ribbons into something that will hold. 'Blessings on your own road,' says {E}.", "", "", ""},
			{"Ask what they prayed for.", "", "empathy:9", "xp:20;pie:1;tip", "'For my brother,' {E} says. 'He went to the hills for treasure and we have heard nothing.' They tell you which hills.", "", "They shake their head. It is not for strangers.", ""}}},
	{ID: "hermit_riddle", Title: "The Hermit's Riddle", Prep: wfPrepRiddle,
		Cast: []wfCast{{Title: "hermit", Glyph: '@', F: CGreen, Host: true}},
		Text: "Smoke from a hollow stump, a pot on three stones, and a person older than the stones. {E} looks you over. 'Answer a riddle and take what the hill gives,' they say. 'Answer wrong and take only the answer.' Then: \"{Rd}\"",
		Ch: []crCh{
			{"Think it through.", "", "lore:10", "item:oddity;xp:25;clue", "'{Ra},' you say. {E} slaps their knee and from the stump takes something wrapped in leaves, and a story about a place nearby.", "", "'No,' says {E}, kindly. 'It is {Ra}.' The hill keeps its gift.", ""},
			{"Offer a meal for a hint.", "item:food", "", "lose:food;xp:12;mer:1;gen:2;coin:+1", "Hunger loosens riddles. 'It was {Ra},' says {E}, and presses a few coins on you for the company.", "", "", ""},
			{"Ask for the reward outright.", "", "persuasion:12", "item:tonic;xp:10", "{E} snorts, but pours you a measure of something green from a gourd.", "hon:-1", "'Cheek is not wit,' says {E}, and turns back to the pot.", ""}}},
	{ID: "ghost_favour", Title: "A Ghost Asks a Favour", Need: "person", Prep: wfPrepDead, Weight: 1,
		Cast: []wfCast{{Name: "{Pg}", Title: "restless ghost", Glyph: '@', F: CCyan, Host: true}},
		Text: "A pale figure stands in the road where there is no one. It is {Pg}, {Pgr}, who died in the year {Pgy} of {Pgc}. The cold around them smells of rain. {Gr} says the ghost, in a voice like wind through a keyhole. 'Carry my last words to {PgT}. They never heard them.'",
		Job:  "grave", JobTitle: "Carry the last words of {Pg} to {PgT}", JobText: "{Pg}, who died in {Pgy}, asked that their last words be spoken in {PgT}.", JobEff: "pie:3;rep:realm:+3;xp:40;item:oddity", JobDays: 8,
		Ch: []crCh{
			{"Promise to carry the words.", "", "", "job;pie:2;mer:2", "'Say that I was not afraid,' the ghost says, and the road is empty.", "", "", ""},
			{"Ask what keeps them here.", "", "empathy:9", "xp:15;clue", "'A grave nobody tends,' says {Pg}. 'And the road I was walking when {Pgc} found me.' They name a place you did not know.", "", "The figure thins like breath on glass.", ""},
			{"Say a prayer for them.", "", "", "pie:3;fav:2;xp:15", "The cold lifts a little. 'That will do,' the ghost says, 'for now.'", "", "", ""},
			{"Order the dead to be gone.", "", "", "pie:-3;cou:2", "The ghost looks at you for a long time, then does as you say. You feel it watching from somewhere.", "", "", ""}}},
	{ID: "envoy", Title: "An Envoy Needs Safe Passage", Weight: 1,
		Cast: []wfCast{{Title: "envoy of {R}", Glyph: '@', F: CMagenta, Host: true}, {Title: "envoy's guard", Glyph: '@', F: CGray, N: 2}},
		Text: "{E}, an envoy of {R}, carries a sealed message to {DestR} in {Dest}. Their escort is down to two worn guards, and the road ahead has been talked of in low voices. 'Walk with us,' {E} says, 'and the seal will see you paid.'",
		Job:  "message", JobTitle: "Guard an envoy of {R} to {Dest}", JobText: "{E} carries a sealed message for {DestR} in {Dest}.", JobEff: "coin:+4;rep:realm:+8;xp:40;hon:2", JobDays: 5,
		Ch: []crCh{
			{"Escort the envoy to {Dest}.", "", "", "job;hon:1", "The guards fall in without a word. The envoy watches the hedges, not you.", "", "", ""},
			{"Offer to carry the message yourself.", "", "persuasion:12", "job;hon:1", "{E} hesitates, then hands over a tube of oilcloth. 'If it is lost, so am I.'", "", "'No,' says {E}. 'A seal is a promise, and promises are carried by those who made them.'", ""},
			{"Sell word of their route to {bandit}.", "", "", "coin:+3;hon:-6;rep:realm:-6;rumour:Word is that an envoy of {R} was sold out on the road.", "You make your report where the brush is thick. The coin is real. The silence afterwards is too.", "", "", ""}}},
	{ID: "soldier_news", Title: "A Wounded Soldier", Prep: wfPrepFront, Weight: 2,
		Cast: []wfCast{{Title: "wounded soldier of {R}", Glyph: '@', F: CGray, Host: true}},
		Text: "{E} lies against a milestone, bandaged in their own sash. They wear the colours of {R} and carry news before they carry anything else: the line against {Fr} is thinner than the heralds say. 'Somebody ought to know,' they say, and look at you.",
		Ch: []crCh{
			{"Dress the wounds (spend a tonic).", "item:tonic", "", "lose:tonic;mer:4;rep:realm:+5;xp:25;clue;rumour:A soldier of {R} told a wanderer how the line against {Fr} really stands.", "{E} tells you about three strongpoints and a river crossing nobody guards. A hand grips your wrist. 'Tell {T}.'", "", "", ""},
			{"Give water and listen.", "", "", "mer:2;xp:12;rumour:A soldier of {R} told a wanderer how the line against {Fr} really stands.", "You hold the flask while {E} talks. It is not tidy news.", "", "", ""},
			{"Take the pack and leave.", "", "", "mer:-4;item:loot;rep:realm:-4", "The pack is heavier than the soldier's friends will think fair.", "", "", ""}}},
	{ID: "standing_stone", Title: "A Strange Standing Stone", Weight: 2,
		Cast: []wfCast{{Name: "the standing stone", Title: "standing stone", Glyph: '|', F: CGray, Obj: true, Host: true}},
		Text: "A grey stone taller than a person stands where no stone should, carved with spirals older than {R}. The grass around it is cropped short. Ribbons and a few {coin}s lie at its foot. The air smells of cold iron, and something about it asks to be touched.",
		Ch: []crCh{
			{"Leave an offering.", "", "", "coin:-1;pie:3;fav:2;hp:+6;xp:15", "The coins sink into the turf. The spirals seem, briefly, to turn. Your aches ease.", "", "", ""},
			{"Put your hand to the stone.", "", "endurance:10", "xp:35;hp:+10;item:oddity", "A slow pulse runs up your arm like a second heartbeat. When it passes there is something in your palm.", "hp:-5;pie:-1", "The stone is a block of ice. Your hand comes away burned and the grass hisses.", ""},
			{"Search around its foot.", "", "perception:10", "item:loot;xp:10", "Under the ribbons, a pouch that was left a long time ago for someone who did not come.", "", "Only old ribbons and older bones.", ""}}},
	{ID: "dying_merchant", Title: "A Dying Merchant", Weight: 1,
		Cast: []wfCast{{Title: "dying {merchant}", Glyph: '@', F: CYellow, Host: true}, {Name: "an overturned cart", Title: "overturned cart", Glyph: '■', F: CBrown, Obj: true}},
		Text: "A cart lies on its side with one wheel still turning. {E}, a {merchant}, lies beside it with a bolt through the ribs and a parcel clutched to their chest. 'Not for me,' they whisper. 'For someone in {Dest}. They do not know it is coming.'",
		Job:  "parcel", JobTitle: "Deliver a dead merchant's parcel in {Dest}", JobText: "{E} gave you a parcel for someone in {Dest} and died on the road.", JobEff: "coin:+6;mer:3;xp:40;rep:realm:+3", JobDays: 6,
		Ch: []crCh{
			{"Promise to deliver the parcel.", "", "", "job;mer:2;gen:1", "'Thank you,' says {E}, and says nothing after that. You close their eyes.", "", "", ""},
			{"Tend them first (spend a tonic).", "item:tonic", "", "lose:tonic;mer:4;job;gen:2", "The bleeding slows. {E} will live, or live a day, and either way you carry the parcel with their blessing.", "", "", ""},
			{"Take the purse and leave.", "", "", "coin:+4;item:loot;mer:-7;gen:-3", "The purse is full. The dead have no use for it.", "", "", ""}}},
	{ID: "collapsed_bridge", Title: "The Collapsed Bridge", Weight: 2,
		Cast: []wfCast{{Name: "the broken bridge", Title: "collapsed bridge", Glyph: '=', F: CBrown, Obj: true, Host: true}, {Title: "stranded carter", Glyph: '@', F: CBrown}},
		Text: "The bridge over the gully has gone, one span lying in the water like a dropped comb. A carter on this side stares at the other. 'Two days round by the old road,' they say, 'or a good pair of boots in the shallows.'",
		Ch: []crCh{
			{"Ford the water at the shallows.", "", "acrobatics:10", "xp:20", "The stones are slick, but you pick your way across and wave back.", "hp:-5;time:30", "You go in to the waist and lose time and temper on the far bank.", ""},
			{"Find the old detour.", "", "survival:9", "xp:20;time:120;tip", "Two hours round by a goat track, and on the way you notice something marked on a boundary stone.", "time:240", "The detour was longer than it looked. Evening comes.", ""},
			{"Help the carters build a crossing.", "", "endurance:10", "xp:25;coin:+2;gen:2;time:90;rep:realm:+3", "Planks, rope and ten pairs of hands. They remember the favour.", "time:120;hp:-3", "The planks slip twice. Your hands are raw before it holds.", ""}}},
	{ID: "bandit_toll", Title: "A Toll on the Road", Weight: 2,
		Cast: []wfCast{{Tpl: "bandit", Title: "toll-keeper", Glyph: '@', F: CRed, Host: true}, {Tpl: "bandit", N: 2}},
		Text: "A rope across the road, and three {bandit} lounging in the shade of it. The one with the best boots names a toll as if reading from a ledger: 'Five {coin} a head, and you walk on with our good wishes.' The other two examine their nails and their knives.",
		Ch: []crCh{
			{"Pay the toll.", "", "", "coin:-2;cou:-1", "They wave you through with elaborate courtesy.", "", "", ""},
			{"Haggle.", "", "persuasion:11", "coin:-1;hon:1", "'Fine, fine. For a clever face.'", "coin:-3", "They laugh and raise the price to make a point.", ""},
			{"Face them down.", "", "intimidate:12", "xp:35;cou:3;rep:out:-3;item:loot", "The rope drops. Three pairs of eyes find something else to look at.", "hostile;cou:1", "'Funny,' says the one with the boots, and his hand moves.", ""},
			{"Cut the rope and fight.", "", "", "hostile;cou:2", "Steel clears the scabbards on both sides.", "", "", ""}}},
	{ID: "stranded_caravan", Title: "The Storm-Stranded Caravan", Weight: 2,
		Cast: []wfCast{{Title: "{merchant}", Glyph: '@', F: CYellow, Host: true}, {Title: "wagon driver", Glyph: '@', F: CBrown, N: 2}, {Name: "a bogged wagon", Title: "bogged wagon", Glyph: '■', F: CBrown, Obj: true}},
		Text: "The last storm left a wagon axle-deep in mud and its drivers sitting on the load with nothing to burn. {E}, the {merchant}, looks up hopefully. 'Three days we have been here. If you can lend a back or a ration, I can lend you my thanks, which is worth something in {Dest}.'",
		Ch: []crCh{
			{"Help dig the wagon out.", "", "endurance:10", "xp:25;coin:+3;gen:3;rep:realm:+2;time:120;item:loot", "It comes free with a sucking sound and a cheer. The {merchant} loads you with thanks and a little more.", "time:150;hp:-4;gen:1", "You are spattered and the axle is still stuck, but they thank you for trying.", ""},
			{"Share food and water.", "item:food", "", "lose:food;gen:4;mer:2;coin:+2;xp:15", "They eat as if it were a feast. Word of this will reach {Dest} before you do.", "", "", ""},
			{"Take what the storm left you.", "", "", "item:loot;item:loot;mer:-6;gen:-3;bounty:30", "A hand on the tailgate, a quick shifting of crates. You are far down the road before anyone looks.", "", "", ""}}},
	{ID: "wolves_stag", Title: "Wolves Driving a Stag", Weight: 2,
		Cast: []wfCast{{Name: "stag", Title: "stag", Tpl: "bs_red_deer", Glyph: 'd', F: CBrown, Alt: "wander", Host: true}, {Tpl: "wolf", N: 3, Alt: "wander"}},
		Text: "A stag bursts from the trees with its flanks heaving, and three wolves spread out behind it in the unhurried way of animals who know how this ends. The stag stops, trembling, a few paces from you. The wolves look at you, then at the stag, and wait to see what you are.",
		Ch: []crCh{
			{"Drive the wolves off.", "", "", "hostile;mer:3;cou:2;xp:10", "The wolves come at you instead, which is the next best thing to a plan.", "", "", ""},
			{"Let nature take its course.", "", "", "xp:5;item:food;mer:-1", "You step back. The pack does its work quickly and leaves some for you.", "", "", ""},
			{"Throw them a ration and slip away.", "item:food", "", "lose:food;mer:2;gen:2;xp:15", "The wolves take the ration and the deal. The stag goes off through the trees with its head up.", "", "", ""}}},
	{ID: "duel_of_honour", Title: "A Duel of Honour", Weight: 1,
		Cast: []wfCast{{Title: "duellist", Glyph: '@', F: CRed, Host: true}, {Title: "duellist", Glyph: '@', F: CRed}},
		Text: "Two figures pace out ground by the road with their coats off and their seconds nowhere. {E} turns to you with relief. 'You will do as a witness. A matter of honour, not a matter of blood, if it can be helped.' The other duellist says nothing and examines their blade.",
		Ch: []crCh{
			{"Stand witness and keep it fair.", "", "", "hon:3;xp:25;coin:+2;rep:realm:+2", "First blood, a handshake, and a purse pressed into your hand for the trouble.", "", "", ""},
			{"Try to stop it.", "", "persuasion:11", "mer:4;xp:25;gen:1", "You talk until they both feel foolish, which is sometimes the best satisfaction.", "hon:-1", "'It is not your quarrel,' they say, and turn their backs on you.", ""},
			{"Step in and fight both.", "", "", "hostile;cou:1", "Two blades turn to face you.", "", "", ""}}},
	{ID: "fortune_teller", Title: "A Fortune Teller", Prep: nil, Weight: 2,
		Cast: []wfCast{{Title: "fortune teller", Glyph: '@', F: CMagenta, Host: true}},
		Text: "In the manner of the {Cu}, {E} has laid out a cloth and a handful of things not worth stealing. 'I will tell you what is behind you, what is ahead, and what is beside you,' they say, 'and you may pick which you will not believe.' {Pv}",
		Ch: []crCh{
			{"Pay for a true reading.", "", "", "coin:-2;clue;xp:20;rumour:A fortune teller on the road said a name nobody had given them.", "{E} turns three cards, frowns, and tells you where a locked thing lies and what is inside.", "", "", ""},
			{"Let them read your palm for nothing.", "", "empathy:11", "tip;xp:10", "{E} traces a line with a dry fingertip. 'Here. Near water, near stone.'", "xp:3", "They sigh. 'A closed hand and a closed mind. Pay or pass.'", ""},
			{"Ask them to lift an ill luck.", "", "", "coin:-3;hp:+8;pie:1", "They burn a hair of yours and bury the ash. You feel lighter and slightly foolish.", "", "", ""}}},
	{ID: "noble_beggar", Title: "The Beggar With Clean Nails", Weight: 2,
		Cast: []wfCast{{Title: "beggar", Glyph: '@', F: CBrown, Host: true}},
		Text: "A beggar sits at the verge with a bowl and a broken staff, murmuring thanks to everyone and noticing everything. The rags are real. The way they sit, back straight and hands folded, is not.",
		Ch: []crCh{
			{"Give coin.", "", "", "coin:-1;gen:3;mer:2;coin:+6;item:loot;rep:realm:+8", "As you turn to go, the beggar's hand closes on yours. 'I am of the court of {R}, travelling as I am for reasons of my own. You paid when you did not know. I do not forget.'", "", "", ""},
			{"Offer a share of your food.", "item:food", "", "lose:food;gen:2;mer:2;item:loot;rep:realm:+4;coin:+3", "The beggar eats slowly, properly, and then names a house in {R}. 'My family will hear of this.'", "", "", ""},
			{"Study their hands.", "", "perception:11", "hon:1;xp:20;coin:+4;rep:realm:+6", "Clean nails, a pale ring-mark on one finger. 'Your hands give you away, my lord.' They laugh, and pay you for the silence.", "", "Hands are hands. You see nothing.", ""},
			{"Shoo them away.", "", "", "mer:-4;rep:realm:-3", "They go, without a word, and with an expression you cannot quite place.", "", "", ""}}},
	{ID: "surveyor", Title: "A Surveyor With a Map Fragment", Weight: 1,
		Cast: []wfCast{{Title: "surveyor of {R}", Glyph: '@', F: CCyan, Host: true}},
		Text: "{E} kneels over a chain and a brass sight with a roll of vellum weighted by stones. 'The crown wants the roads redrawn,' they say, 'and I would like to be somewhere warm. I have a fragment you might want, if you are the kind who walks off the road.'",
		Ch: []crCh{
			{"Trade stories for the fragment.", "", "lore:10", "map;xp:20", "A good story is worth a good map. {E} copies a corner of it onto a clean sheet.", "xp:5", "'Charming,' says {E}, who has heard better.", ""},
			{"Buy the fragment.", "", "", "coin:-3;map", "The vellum is creased and smudged and, in one corner, mysterious.", "", "", ""},
			{"Give them water.", "", "", "gen:1;tip;xp:8", "'Kind of you,' says {E}, and points on the roll at nothing in particular, which turns out to be something.", "", "", ""}}},
	{ID: "runaway", Title: "A Runaway", Weight: 1,
		Cast: []wfCast{{Title: "runaway", Glyph: '@', F: CWhite, Host: true}},
		Text: "{E} is crouched behind a boundary stone with a bundle and the look of someone who has left a good house for a better idea. 'They are marrying me to a stranger,' they say. 'I only want to reach {Dest}. I will pay what I can.'",
		Job:  "escort", JobTitle: "Take a runaway to {Dest}", JobText: "{E} ran from an arranged marriage and wants to reach {Dest}.", JobEff: "coin:+2;mer:3;gen:2;rep:realm:-3;xp:30", JobDays: 4,
		Ch: []crCh{
			{"Hide them and take them to {Dest}.", "", "", "job;mer:2;hon:-1", "{E} pulls a hood over their face and keeps close.", "", "", ""},
			{"Talk them into going home.", "", "persuasion:11", "hon:2;rep:realm:+3;xp:15", "They go back, slowly, with their chin up. Perhaps something will change.", "mer:-1", "'You do not understand,' they say, and are gone into the brush.", ""},
			{"Tell them the {guard} will hear about this.", "", "", "hon:1;mer:-3;rep:realm:+4;coin:+1", "They look at you as if you had slapped them, then run.", "", "", ""}}},
	{ID: "deserter", Title: "A Deserter", Weight: 1,
		Cast: []wfCast{{Title: "deserter of {R}", Glyph: '@', F: CGray, Host: true}},
		Text: "{E} is wearing the colours of {R} with the badges cut off, and holds a knife very carefully, as if afraid to drop it. 'I am not a bad person,' they say. 'I saw a village burn and then I did not want to be a soldier. Are you going to turn me in?'",
		Ch: []crCh{
			{"Let them go.", "", "", "mer:3;rep:realm:-3;xp:10", "They run. The knife is left behind in the dust, as a courtesy.", "", "", ""},
			{"Take them in as a prisoner.", "", "", "rep:realm:+6;coin:+3;mer:-3;xp:15", "They do not resist. The {guard}s of {R} pay for deserters, and do not ask what you saw.", "", "", ""},
			{"Share a meal and point them to the coast.", "item:food", "", "lose:food;gen:2;mer:3;xp:12", "They eat in four bites and disappear east at a steady walk.", "", "", ""}}},
	{ID: "knight_errant", Title: "A Knight Errant", Weight: 1,
		Cast: []wfCast{{Title: "knight errant", Glyph: '@', F: CWhite, Host: true}},
		Text: "A figure in mismatched plate has tied a ribbon across the road and stands beside it with a blunt practice blade, bowing to every traveller. 'A friendly bout, for honour and nothing else,' {E} announces. 'Win, and take my second blade. Lose, and take the story to the next town.'",
		Ch: []crCh{
			{"Accept the bout.", "", "acrobatics:11", "xp:40;item:weapon;cou:2", "A scrape, a feint, a turn of the wrist. {E} yields with real delight and hands over the second blade.", "hp:-6;cou:1", "You wake with a bruise the shape of a gauntlet and a good-natured stranger standing over you.", ""},
			{"Refuse politely.", "", "", "xp:2", "{E} bows deeper and lets you pass. 'The next, then.'", "", "", ""},
			{"Take up a real weapon.", "", "", "hostile", "The ribbon drops. {E} draws the sharp blade from under their cloak and the delight goes out of their face.", "", "", ""}}},
	{ID: "minstrel", Title: "A Minstrel by the Road", Weight: 2,
		Cast: []wfCast{{Title: "minstrel", Glyph: '@', F: CMagenta, Host: true}},
		Text: "{E} plays a small stringed thing on a stump and sings, loudly and a little flat, about {Ruler} of {R}. {Gr} The song has a verse about {T} that is not quite polite. They stop when they see you and raise an eyebrow, which is an invitation of a kind.",
		Ch: []crCh{
			{"Tip them and ask for a song.", "", "", "coin:-1;xp:15;gen:2;rumour:A minstrel on the road was singing of {Ruler} and {T}.", "{E} sings a song of the road with the verses sorted by who might be listening.", "", "", ""},
			{"Trade verses.", "", "lore:10", "xp:25;item:oddity", "Your verse is rough but true, and {E} pays you in the way minstrels do: with something odd from the bottom of a pocket.", "xp:5", "A better rhyme is needed. {E} claps anyway.", ""},
			{"Ask what news they carry.", "", "", "clue;xp:5", "Between verses, {E} names a place that does not appear in any song.", "", "", ""}}},
	{ID: "sick_traveller", Title: "A Sick Traveller", Weight: 1,
		Cast: []wfCast{{Title: "sick traveller", Glyph: '@', F: CGreen, Host: true}},
		Text: "{E} lies in the ditch with a fever, shivering in the sun, tin cup beside them. Others have gone around. Whatever it is, it does not look like plague. It looks like exposure and bad water.",
		Ch: []crCh{
			{"Tend them.", "", "endurance:10", "mer:5;xp:30;coin:+2", "You sit with {E} until the fever breaks. When they wake they press their last coins into your hand.", "hp:-5;mer:2", "You do what you can. You come away with a chill of your own.", ""},
			{"Leave a ration and keep your distance.", "item:food", "", "lose:food;mer:1;gen:1", "You leave it where they can reach it.", "", "", ""},
			{"Tell them where the {doc} of {Dest} keeps shop.", "", "", "mer:1;xp:5", "'Thank you,' says {E}, and then more weakly, 'I will try.'", "", "", ""}}},
	{ID: "loose_mule", Title: "A Loose Pack-Mule", Weight: 1,
		Cast: []wfCast{{Name: "loose pack-mule", Title: "mule", Tpl: "boar", Glyph: 'm', F: CBrown, Alt: "still", Host: true}},
		Text: "A pack-mule stands in the road with its halter trailing and a saddlebag half unbuckled. The brand on its flank is of {Dest}. There is no owner in sight, and a day of the road in the animal's coat.",
		Job:  "escort", JobTitle: "Return a stray mule to {Dest}", JobText: "A mule branded in {Dest} has wandered a long way from home.", JobEff: "coin:+4;mer:2;xp:25;item:loot", JobDays: 4,
		Ch: []crCh{
			{"Lead it to {Dest}.", "", "", "job;mer:2", "The mule falls in beside you with a resigned air.", "", "", ""},
			{"Look through the saddlebags.", "", "", "item:loot;mer:-2;xp:5", "A few things of use, which you were not exactly invited to take.", "", "", ""},
			{"Look for its owner.", "", "survival:10", "mer:3;xp:20;coin:+2", "Hoofprints, a dropped glove, a distant shout. The owner is down the road and very grateful.", "time:60", "You circle for an hour. The mule watches you the whole time.", ""}}},
	{ID: "herb_woman", Title: "The Herb-Woman", Weight: 2,
		Cast: []wfCast{{Title: "herb-woman", Glyph: '@', F: CGreen, Host: true}},
		Text: "{E} kneels in the scrub with a basket and a small knife. 'These only grow where nobody looks,' they say. They offer to sell you a little of what is in the basket and tell you what it will and will not cure.",
		Ch: []crCh{
			{"Buy a tonic.", "", "", "coin:-2;item:tonic", "A dark bottle with a leaf pressed in the stopper.", "", "", ""},
			{"Swap food for herbs.", "item:food", "", "lose:food;item:tonic;xp:5", "A fair trade for both of you.", "", "", ""},
			{"Ask about the ill luck on this ground.", "", "empathy:10", "pie:1;xp:20;tip", "{E} lowers her voice and talks about something buried not far from here.", "", "She shrugs and goes back to her knife.", ""}}},
	{ID: "ford_keeper", Title: "The Ford-Keeper", Weight: 1,
		Cast: []wfCast{{Title: "ford-keeper", Glyph: '@', F: CBlue, Host: true}, {Name: "a rope ferry", Title: "rope ferry", Glyph: '=', F: CBrown, Obj: true}},
		Text: "A rope across a brown river, and a raft that can take a person and a bundle if they pull. {E} sits beside it with a toll bowl. 'A coin for the crossing,' they say. 'Or pull with me and cross free. Or wade, if you are one of those.'",
		Ch: []crCh{
			{"Pay the crossing.", "", "", "coin:-1;xp:3", "A short ride, and nobody speaks.", "", "", ""},
			{"Wade it yourself.", "", "acrobatics:10", "xp:12", "Cold to the thigh, but across.", "hp:-4;time:30", "You lose your footing once and your temper twice.", ""},
			{"Pull the rope and cross free.", "", "endurance:9", "gen:1;xp:10;coin:+1", "The rope burns your hands and the ford-keeper gives you a few coins out of the bowl.", "time:60", "You pull until your arms shake. It is slow, and the ford-keeper says nothing.", ""}}},
	{ID: "pinned_woodcutter", Title: "A Pinned Woodcutter", Weight: 2,
		Cast: []wfCast{{Title: "woodcutter", Glyph: '@', F: CBrown, Host: true}, {Name: "a felled trunk", Title: "felled trunk", Glyph: 'T', F: CDarkGreen, Obj: true}},
		Text: "A trunk lies across {E}'s leg, and they have stopped shouting. Their axe stands in the stump where they left it. 'It is not as bad as it looks,' {E} says, in the voice of someone who knows it is exactly as bad as it looks.",
		Ch: []crCh{
			{"Lift the trunk.", "", "endurance:11", "mer:5;coin:+3;xp:30", "You heave, and {E} drags the leg free with a yelp. A handful of coins and a bundle of firewood is what they have to give.", "hp:-6;mer:1", "It will not budge, and something in your back says so first.", ""},
			{"Cut them free with a blade.", "", "tinkering:9", "mer:4;coin:+2;xp:25", "A wedge, a lever and a prayer, in that order.", "hp:-2", "Your wedge slips twice before it takes.", ""},
			{"Run to {Dest} for help.", "", "", "mer:-1;xp:5;time:60", "You hope it is enough.", "", "", ""}}},
	{ID: "mourner_cairn", Title: "A Mourner at a Cairn", Weight: 1,
		Cast: []wfCast{{Title: "mourner", Glyph: '@', F: CGray, Host: true}, {Name: "a roadside cairn", Title: "cairn", Glyph: '†', F: CGray, Obj: true}},
		Text: "{E} kneels beside a small cairn with a ribbon tied on the top stone. It is not a grave by any law of {R}; it is a place where someone died on the road, and a person who came back. They look up. {Pv}",
		Ch: []crCh{
			{"Share their silence.", "", "empathy:9", "mer:3;pie:2;xp:15;rumour:A mourner on the road told a stranger of the one who died there.", "You stand with {E} until the light changes. 'Thank you,' they say eventually. 'Nobody does.'", "", "You stand, but you do not know what to do with your hands. They hear that, too.", ""},
			{"Add a stone.", "", "", "pie:2;gen:1;xp:5", "A small stone on a small pile. {E} nods.", "", "", ""},
			{"Ask who lies there.", "", "", "xp:5;tip", "'My brother,' says {E}. 'He went looking for something that was not his to find.' They tell you where.", "", "", ""}}},
}
