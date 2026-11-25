package engine

// Data for sieges.go (prefix sq): role steps, news and talk lines. Placeholders filled by sqFill:
// {T} the besieged town, {A} the besieging realm, {D} the defending realm, {Cap} the captain of the garrison,
// {Qm} the quartermaster, {Off} the besiegers' field officer, {Dest} the town a relief ride goes to,
// {Days} days left on that ride.
//
// Effect tokens (separated by ";"): coin:+n|-n (units of local money scaled by level), xp:n, hp:+n|-n (never lethal),
// rep:d:n and rep:a:n (standing with the defending or attacking realm), prog:n (the besiegers' progress),
// mor:n (defenders' morale), food:n (the town's stores), truce:n, relief:n, sick:n, merit:n (siege rank),
// side:d|a (which side you serve), takefood:n (hand over n rations from the pack), ride:start|fast|end,
// flee, deed:text, news:text. Anything else goes to crApply (karma axes mer hon cou gen pie, item:food, time:n ...).
//
// Requirements (Req, comma separated): sq:food:n, sq:coin:n, sq:contra, sq:phase:<name> (that phase or later),
// sq:town (you stand in the besieged town); everything else goes to crReqOK (skill:, stat:, gold:, item:, karma:).

type sqCh struct {
	T, Req, Chk, Win, WT, Lose, LT, WN, LN string
}

type sqStep struct {
	ID, Text string
	Ch       []sqCh
}

// Steps that start a clock (choices left before the scene turns against you).
var sqClockSteps = map[string]int{"a_assault1": 3, "x_start2": 2}

// The step shown when a clock runs out, by role letter.
var sqLateStep = map[string]string{"a": "a_late", "x": "x_late"}

var sqSteps = []sqStep{
	// ------------------------------------------------------------ defender
	{ID: "d_start", Text: "Captain {Cap} has men on the wall of {T} and not enough of them. The pay is small and it comes on time, which counts for something when the markets are shut. \"Take a post,\" says the captain. \"Tell me what you are good for.\"", Ch: []sqCh{
		{T: "Take the oath and stand a night watch on the wall.", Win: "side:d", WT: "You are given a spear-length of parapet and a cloak that smells of someone else.", WN: "d_watch"},
		{T: "Offer to carry water to the fire-crews.", Win: "side:d", WT: "A bucket is put in each hand. \"The roofs go up whenever they throw pitch.\"", WN: "d_water"},
		{T: "Ask to be set to work on the broken stretch of wall.", Req: "sq:phase:assault", Win: "side:d", WT: "\"Mind the loose stone. The masons are asleep on their feet.\"", WN: "d_repair"},
	}},
	{ID: "d_watch", Text: "Night on the wall above the main gate. The camp of {A} lies in the dark as a spill of small fires. Wind brings voices, a dog and, once, someone singing.", Ch: []sqCh{
		{T: "Watch the dark below the wall.", Chk: "perception:10", Win: "coin:6;mor:+3;prog:-4;rep:d:+8;merit:3;xp:8", WT: "A shape that is too regular to be a bush: a mantlet being rolled toward the wall. You shout, the fire-pots go over, and the shape goes back the way it came.", Lose: "coin:2;prog:+3;mor:-2;rep:d:+2", LT: "You see nothing until the first ladder knocks the parapet beside your hand.", WN: "", LN: "d_ladders"},
		{T: "Stand the whole night in the cold and keep the others awake.", Chk: "endurance:10", Win: "coin:7;mor:+2;rep:d:+5;merit:3", WT: "You are stiff at dawn and the watch is whole. The sergeant says nothing, which from a sergeant is thanks.", Lose: "coin:3;hp:-4", LT: "You last until the small hours and then sit down in the lee of a merlon. Someone drags you up by the collar.", WN: "", LN: ""},
		{T: "Slip away for an hour of sleep.", Win: "coin:2;prog:+2;mor:-1;rep:d:-6", WT: "Nothing happens. Nothing happens on the next post either, and the sergeant has noticed the gap.", WN: ""},
	}},
	{ID: "d_ladders", Text: "Ladders strike the parapet. Hooks bite the stone and the first helmets rise over the edge on your stretch of wall.", Ch: []sqCh{
		{T: "Throw the ladders down.", Chk: "intimidate:10", Win: "coin:9;prog:-5;mor:+3;rep:d:+10;merit:4;xp:10", WT: "You get your shoulder under the nearest ladder and heave. It goes over slowly and then all at once.", Lose: "hp:-9;prog:+4;mor:-3;coin:4;rep:d:+3", LT: "The ladder will not move and a spear takes you in the arm. Others come up and tip it for you, and carry you back.", WN: "", LN: ""},
		{T: "Run for the stair and shout for the reserve.", Win: "prog:+2;mor:-3;rep:d:-12;cou:-1", WT: "The reserve comes. So do the stares of the men you left.", WN: ""},
	}},
	{ID: "d_water", Text: "Pitch and fire-arrows have set a row of roofs alight behind the market. The bucket chain is short of hands, and the cisterns are lower than anyone will say aloud.", Ch: []sqCh{
		{T: "Carry buckets until the fires are down.", Chk: "endurance:9", Win: "coin:5;mor:+2;rep:d:+6;merit:2;xp:6", WT: "By dawn the roofs are black beams and wet ash, and nothing past them has caught.", Lose: "coin:2;hp:-3", LT: "You are hauling your tenth bucket when your arms stop working. They put you on a bench with a cup of what is left in the cistern.", WN: "", LN: ""},
		{T: "Take your water to the wounded in the hospital hall instead.", Win: "coin:3;mor:+2;rep:d:+4;gen:+1", WT: "The hall is a granary with straw on the floor. Nobody says much. A man grips your wrist when you set the cup down and does not let go for a while.", WN: ""},
	}},
	{ID: "d_repair", Text: "A stretch of the curtain wall has been broken by stones and sapping. Masons are shouting for hands, timber and anything that will fill a hole before the next assault.", Ch: []sqCh{
		{T: "Shore the breach with timber and rubble.", Chk: "tinkering:10", Win: "coin:12;prog:-8;mor:+4;rep:d:+12;merit:5;xp:12", WT: "You and the masons make a slope of the gap, steep enough that a man must climb it with his hands. The captain looks at your work for a long time and then claps your shoulder.", Lose: "coin:4;prog:+3;hp:-7", LT: "Your cribbing collapses under the first load and takes your legs with it. The masons dig you out and start over.", WN: "", LN: ""},
		{T: "Haul stone for the masons.", Chk: "endurance:9", Win: "coin:8;prog:-4;mor:+2;rep:d:+6;merit:3", WT: "Basket after basket until your back is a hot bar. The wall is thicker at dawn.", Lose: "coin:3;hp:-5", LT: "You carry until something in your back says no. The masons find another pair of hands.", WN: "", LN: ""},
	}},

	// ------------------------------------------------------------ sortie
	{ID: "x_start", Text: "Captain {Cap} has spent three nights watching the besiegers' siege engines. \"Before dawn there is an hour when the pickets sleep. I want fire among those timbers. I need someone who will go out the postern and come back.\"", Ch: []sqCh{
		{T: "Lead the sortie through the postern.", Req: "sq:town", Win: "side:d", WT: "\"Twelve men. They will follow if you do not look back.\"", WN: "x_start2"},
		{T: "Go out with the sortie as one of the swords.", Req: "sq:town", Win: "side:d", WT: "\"Keep close to the torch-bearer. Do not stop to loot.\"", WN: "x_follow"},
	}},
	{ID: "x_start2", Text: "Twelve men wait in the dark behind the postern. The bar is lifted without a sound. Beyond it, the ditch, the stakes and the sleeping line of {A}'s camp. Every minute out there is a minute the pickets can wake.", Ch: []sqCh{
		{T: "Rally the men and go over the ditch in a rush.", Chk: "persuasion:11", Win: "mor:+2", WT: "They go with you, low and quick, and the first tents are behind you before the pickets have found their spears.", Lose: "hp:-4", LT: "The men hang back at the stakes. When they come, they come strung out and noisy.", WN: "x_burn", LN: "x_rout"},
		{T: "Crawl the ditch and lead them along the ground.", Chk: "acrobatics:11", Win: "mor:+1", WT: "You hold your breath and the dark holds you. Twelve men creep behind you, tied to a cord.", Lose: "hp:-3", LT: "A stake you did not see takes your thigh. You bite your sleeve and keep going.", WN: "x_burn", LN: "x_rout"},
	}},
	{ID: "x_burn", Text: "The engines loom out of the dark: a ram under its hide roof, a stone-thrower, stacks of mantlets. Pitch jars are passed up.", Ch: []sqCh{
		{T: "Fire the ram and the thrower together.", Chk: "acrobatics:10", Win: "coin:15;prog:-12;mor:+6;rep:d:+15;merit:6;xp:15;deed:{T} burned the engines of {A} in a sortie before dawn.;news:A sortie from {T} burned the siege engines of {A} before dawn.", WT: "The pitch takes at once. Behind you the camp wakes to red light and shouting, and by then the postern is open and waiting.", Lose: "coin:5;prog:-3;mor:-2;hp:-10;rep:d:+4", LT: "The jar slips and breaks on stone. One engine catches, the other does not, and the camp is awake. You run.", WN: "", LN: ""},
		{T: "Cut the ropes and wedge the ram's wheels.", Chk: "tinkering:11", Win: "coin:10;prog:-8;mor:+3;rep:d:+9;merit:4;xp:10", WT: "Quiet work. By the time they roll it forward, the ram will jam on the first slope.", Lose: "coin:4;prog:-2;hp:-6", LT: "A rope you cut was holding something heavy. You get away with a bruise and a story.", WN: "", LN: ""},
	}},
	{ID: "x_follow", Text: "You go out with the torch-bearer. Over the ditch, past the first line of sleeping men, and the siege engines are in front of you. A picket stirs and raises his head.", Ch: []sqCh{
		{T: "Silence the picket before he shouts.", Chk: "intimidate:10", Win: "coin:10;prog:-6;mor:+3;rep:d:+8;merit:4;xp:10", WT: "A hand over his mouth, a knee in his back. Behind you the pitch goes on the engines and the sortie runs back with its eyes shining.", Lose: "coin:4;prog:-2;hp:-8;rep:d:+3", LT: "He shouts. You hit him, but the shout has already gone out. The sortie scrambles home with half its work done.", WN: "", LN: ""},
		{T: "Hold the line of retreat and let the others work.", Win: "coin:5;prog:-3;mor:+1;rep:d:+4;merit:2", WT: "You stand at the ditch with a drawn blade and a dry mouth. Fire blooms behind you and you do not look at it.", WN: ""},
	}},
	{ID: "x_rout", Text: "The sortie is a mess of men and torches in the wrong place. A horn goes in the camp. You are cut off from the postern by a line of spears.", Ch: []sqCh{
		{T: "Break through the line and drag the others after you.", Chk: "intimidate:11", Win: "coin:6;prog:-2;rep:d:+6;merit:3;xp:8", WT: "You go through the line with your blade and your voice and the others come after you. Not everyone.", Lose: "coin:2;hp:-12;mor:-3;rep:d:+2", LT: "The line closes. The way back is a gauntlet of blades and you take your share.", WN: "", LN: ""},
		{T: "Hide in the ditch until the camp settles.", Chk: "streetwise:11", Win: "coin:2;hp:-2", WT: "You lie with your face in cold mud while boots go back and forth. Near dawn you crawl back to the postern.", Lose: "hp:-9;mor:-2;rep:d:-3", LT: "A lantern finds you. You run, bleeding, and no one has waited for you.", WN: "", LN: ""},
	}},
	{ID: "x_late", Text: "The pickets are awake. Horns and shouting. The sortie is a thing of the past.", Ch: []sqCh{
		{T: "Run for the postern.", Win: "hp:-6;mor:-2;coin:2", WT: "You run. The postern is still open and a man you do not know hauls you through by the arm.", WN: ""},
	}},

	// ------------------------------------------------------------ attacker
	{ID: "a_start", Text: "{Off}, an officer of {A}'s army, is hiring hands outside {T}. The coin is not much and what it promises is a share. \"The walls will come down eventually,\" says the officer. \"I would like them to come down while I am still alive. What can you do?\"", Ch: []sqCh{
		{T: "Take the coin and scout the walls.", Win: "side:a", WT: "A purse, a charcoal sketch of the wall and a pointing finger. \"Weak places. Ditches. Gates.\"", WN: "a_scout"},
		{T: "Ask to join the sappers.", Win: "side:a", WT: "\"Good. Nobody volunteers. You will need to be brave and small.\"", WN: "a_sap"},
		{T: "Join a foraging party.", Win: "side:a", WT: "\"Wagons leave at first light. Bring a sack.\"", WN: "a_forage"},
		{T: "Take a place in the assault.", Req: "sq:phase:assault", Win: "side:a", WT: "\"The horn goes at dawn. Second rank. Do not be early and do not be late.\"", WN: "a_assault1"},
	}},
	{ID: "a_scout", Text: "{T} lies behind its ditch, quiet as a held breath. The officer wants to know where the wall is thin, where the sentries change and which gate is the weakest.", Ch: []sqCh{
		{T: "Crawl along the ditch at dusk and study the wall.", Chk: "streetwise:10", Win: "coin:8;prog:+5;rep:a:+8;merit:3;xp:8", WT: "You find a stretch where the old mortar is bleeding out. You mark it with a cairn of three stones the officer will know.", Lose: "coin:3;prog:+1;hp:-6", LT: "A crossbow bolt takes the ground at your feet. You bring back a sketch that is half guess.", WN: "", LN: ""},
		{T: "Question the farmers who sell eggs at the lines.", Chk: "persuasion:10", Win: "coin:6;prog:+4;rep:a:+6;merit:2;xp:6", WT: "An old woman has a cousin on the wall. She is angry at both armies and tells you which tower sleeps.", Lose: "coin:2;prog:+1", LT: "They sell you eggs and say nothing else.", WN: "", LN: ""},
	}},
	{ID: "a_sap", Text: "The tunnel runs from the ditch toward the wall, a low black throat held up with props. The air is thick and the earth drips. The sappers say that listening is the hardest part, because the defenders are listening too.", Ch: []sqCh{
		{T: "Set the props true and keep the roof up.", Chk: "tinkering:10", Win: "coin:10;prog:+7;rep:a:+10;merit:4;xp:10", WT: "By the end of the shift the tunnel is under the foundation stones. The sappers fill the end with brushwood and pitch.", Lose: "coin:4;prog:+2;hp:-9", LT: "A prop gives. You are buried to the chest and dug out by cursing men. The tunnel is still there.", WN: "", LN: ""},
		{T: "Dig, carry and say nothing.", Chk: "endurance:9", Win: "coin:7;prog:+4;rep:a:+5;merit:2", WT: "Hours of mud in the dark. The tunnel advances by the length of a man.", Lose: "coin:2;hp:-5", LT: "The bad air gets to you and you are carried out grey in the face.", WN: "", LN: ""},
	}},
	{ID: "a_forage", Text: "The villages around {T} have been picked clean by two armies and the weather. A few farms still have a cow or a store of grain behind a false wall. The officer does not care how you get it, and does not want to hear.", Ch: []sqCh{
		{T: "Take what you need by force.", Chk: "intimidate:10", Win: "coin:8;prog:+3;rep:a:+6;merit:2;mer:-1;gen:-1;item:food", WT: "The farmer lets you in the barn with his eyes on the floor. The sack you bring back is heavy.", Lose: "coin:2;hp:-6;mer:-1", LT: "He has a pitchfork and a son. You come back with a bruise and half a sack.", WN: "", LN: ""},
		{T: "Pay a fair price.", Req: "sq:coin:4", Win: "coin:-4;prog:+1;rep:a:+3;mer:+1;item:food", WT: "They are startled to be paid. The price is more than fair and they know it. They load the cart anyway.", WN: ""},
		{T: "Take the grain and leave something for the farmers.", Chk: "persuasion:10", Win: "coin:6;prog:+2;rep:a:+4;merit:1;gen:+1;item:food", WT: "You haggle them into a share they can live on. They give you milk for the road.", Lose: "coin:2;prog:+1", LT: "They will not talk. You come back with the officer's wagon half empty.", WN: "", LN: ""},
	}},
	{ID: "a_assault1", Text: "The horns go at dawn. Ladders, rams and shields move in a slow tide toward the wall of {T}. You are in the second rank, with the ditch ahead and arrows coming down like rain. Time is short: the ranks behind you are closing.", Ch: []sqCh{
		{T: "Climb a ladder.", Chk: "acrobatics:10", Win: "prog:+2", WT: "Rungs, hands, stone. You are at the top before you have thought about it.", Lose: "hp:-5", LT: "A rung gives under you. You scramble up the next ladder on your knees.", WN: "a_assault2", LN: "a_assault2"},
		{T: "Push the ram toward the gate.", Chk: "intimidate:10", Win: "prog:+2", WT: "The ram goes forward under its roof of hides, and the roof is the only thing between you and the boiling water.", Lose: "hp:-7", LT: "The water comes down anyway. You keep pushing.", WN: "a_assault2", LN: "a_assault2"},
		{T: "Hold the shield wall and wait for the breach.", Win: "prog:+1", WT: "You stand in the line and hold your shield up. Something heavy bounces off it and goes on into someone else.", WN: "a_assault_end"},
	}},
	{ID: "a_assault2", Text: "The parapet is a narrow stage of stone and noise. Defenders pour out of the tower stair, and the ladders on both sides of you are thick with men. Time is short.", Ch: []sqCh{
		{T: "Fight along the wall and take the tower.", Chk: "intimidate:11", Win: "prog:+8", WT: "You go at them with your blade and shoulder, and the line behind you comes up over the stone.", Lose: "hp:-10", LT: "A spear throws you back against the merlon and the defenders are on you.", WN: "a_assault_win", LN: "a_assault_fall"},
		{T: "Plant the banner on the parapet and hold it.", Chk: "persuasion:11", Win: "prog:+6", WT: "A cloth on a pole. Men follow a cloth better than a command.", Lose: "hp:-8", LT: "Whoever holds the banner is a mark. You are a mark.", WN: "a_assault_win", LN: "a_assault_fall"},
	}},
	{ID: "a_assault_win", Text: "The tower bell goes silent. The assault holds the stretch of wall. {Off} claps your shoulder and says something that is nearly pride.", Ch: []sqCh{
		{T: "Take your share and drag yourself out of the press.", Win: "coin:20;prog:+10;rep:a:+15;merit:6;xp:20;deed:{T} was assaulted by {A} and the wanderer took the wall.;news:{A}'s men took a stretch of the wall of {T}.", WT: "Pay, a bandage, a place by a fire. You are alive and the officer knows your name.", WN: ""},
	}},
	{ID: "a_assault_fall", Text: "You are thrown from the wall into a press of men and you are carried back out of the fight by the same tide that took you in.", Ch: []sqCh{
		{T: "Crawl out of the ditch.", Win: "coin:5;hp:-8;prog:+2;rep:a:+3;merit:1", WT: "Someone drags you out by the belt. The wall stands, and the dead are very quiet.", WN: ""},
	}},
	{ID: "a_assault_end", Text: "The assault comes and goes about you like a sea around a post. Afterward the officer counts heads and finds yours.", Ch: []sqCh{
		{T: "Take what pay a man gets for standing in a line.", Win: "coin:6;prog:+1;rep:a:+3;merit:1", WT: "It is not glory, but nobody has to carry you.", WN: ""},
	}},
	{ID: "a_late", Text: "The horns sound the retreat. You were too slow, and the tide has turned without you.", Ch: []sqCh{
		{T: "Fall back with the others.", Win: "coin:2;hp:-4;rep:a:-2", WT: "You run with the rest, and the arrows follow.", WN: ""},
	}},

	// ------------------------------------------------------------ smuggler
	{ID: "s_start", Text: "Hunger has made a trade. Quiet hands in {T} pay well for food, and quiet hands outside pay well for what comes out. Quartermaster {Qm} cannot be seen to deal with outsiders, but a note passed through a gutter says that no questions will be asked.", Ch: []sqCh{
		{T: "Bring food through the lines to the town.", Req: "sq:food:2", Win: "", WT: "You shoulder the sack. The lines of {A} lie between you and the wall.", WN: "s_in"},
		{T: "Buy grain from the camp followers and run it in.", Req: "sq:coin:8", Win: "coin:-8;item:food;item:food;item:food", WT: "The sutler names a price that would shame a priest. You pay it, and take three sacks.", WN: "s_in"},
		{T: "Take people out by the {Way}.", Win: "", WT: "A family, three children and a cart that will not go. They have one night.", WN: "s_out"},
		{T: "Carry a sealed packet of contraband for the quartermaster.", Req: "sq:contra", Win: "", WT: "\"The guard will search you. Let them find nothing and say nothing.\"", WN: "s_contra"},
	}},
	{ID: "s_in", Text: "Pickets walk the ditch and a ring of fires keeps the dark at the edge. Two hundred paces of mud lie between you and the old drain.", Ch: []sqCh{
		{T: "Crawl the drain under the ditch.", Chk: "streetwise:11", Win: "", WT: "The drain is slime and cold, and you go through it with the sack in front of you.", Lose: "", LT: "Your elbow hits an iron grate. It rings.", WN: "s_in2", LN: "s_caught"},
		{T: "Walk in among the sutlers as one of them.", Chk: "bluff:11", Win: "", WT: "Nobody looks at your face. They look at your sack and then at the next sack.", Lose: "", LT: "The sutler you stood beside points. \"That one is not ours.\"", WN: "s_in2", LN: "s_caught"},
		{T: "Run the gap between pickets in the dark.", Chk: "acrobatics:12", Win: "", WT: "Count to sixty, move on forty-five. Mud, ditch, stakes. You go through without breathing.", Lose: "hp:-4", LT: "A stake you did not see tears a long gash across your shin.", WN: "s_in2", LN: "s_caught"},
	}},
	{ID: "s_in2", Text: "At the inner end of the drain a rusted grating swings open and {Qm} the quartermaster is waiting with a lantern and a ledger. \"Bless you,\" is all that is said, and it is enough.", Ch: []sqCh{
		{T: "Hand over the food and take your pay.", Win: "takefood:3;coin:14;food:+4;mor:+3;rep:d:+10;merit:2;xp:12;gen:+1;news:Someone has run grain through the lines of {A} into {T}.", WT: "The sacks vanish into the dark of the stores. The quartermaster counts out coin and writes nothing.", WN: ""},
		{T: "Keep a share for yourself and hand over the rest.", Win: "takefood:2;coin:9;food:+2;mor:+1;rep:d:+5;merit:1;xp:8", WT: "\"Fair,\" says the quartermaster, in the voice of a person who has decided not to argue.", WN: ""},
		{T: "Sell it in the market for what hunger will pay.", Win: "takefood:3;coin:30;food:+1;rep:d:-6;mer:-1;gen:-1", WT: "The price is shameful. The crowd that gathers does not stay long, and some of them will remember your face.", WN: ""},
	}},
	{ID: "s_caught", Text: "A lantern, a spear at your throat and a picket in a cloak of {A}'s colours. \"What is in the sack?\"", Ch: []sqCh{
		{T: "Talk it out.", Chk: "persuasion:11", Win: "mor:+1;rep:a:+2", WT: "You are a sutler's cousin lost in the dark, and it is not a very good story, but the picket is tired.", Lose: "takefood:3;hp:-6;rep:a:-5", LT: "He takes the sack and the argument, and puts you out of the camp with a boot.", WN: "", LN: ""},
		{T: "Drop the sack and run.", Win: "takefood:3;hp:-2", WT: "They do not chase. They have the sack, and they are hungry too.", WN: ""},
		{T: "Pay the picket to look the other way.", Req: "sq:coin:6", Win: "coin:-6;rep:a:-1;hon:-1", WT: "The coin disappears into a glove. \"Go on, then. Keep to the drain.\"", WN: "s_in2"},
	}},
	{ID: "s_out", Text: "A family with three children and no way out. The gates are shut by order, but there is an old tunnel under the north wall, or a boat on the water if the quartermaster's friend is not at his post.", Ch: []sqCh{
		{T: "Lead them through the tunnel at night.", Chk: "streetwise:11", Win: "coin:12;food:+3;mor:+1;rep:d:+6;mer:+1;gen:+1;merit:2;xp:12;news:Folk are slipping out of {T} by night.", WT: "Dark, damp, a child asleep in your arms. At the far end the family stands up in the weeds under stars, and no one is shouting.", Lose: "hp:-6;rep:d:-2;gen:-1", LT: "A picket finds you at the mouth of the tunnel. The family is turned back to the town, and you are turned loose.", WN: "", LN: ""},
		{T: "Row them out in a boat before dawn.", Chk: "survival:11", Win: "coin:14;food:+3;mor:+1;rep:d:+6;mer:+1;merit:2;xp:12", WT: "Oars wrapped in cloth, a fog on the water. You put them on a bank a mile from the nearest camp.", Lose: "hp:-4;rep:d:-2", LT: "A skiff with a lantern. You pull hard for the reeds and lose the boat, but not the family.", WN: "", LN: ""},
	}},
	{ID: "s_contra", Text: "The sealed packet is warm against your ribs. At the sally-port a guard of {D} holds up a hand. \"Open your pack.\"", Ch: []sqCh{
		{T: "Say it is a letter for the captain.", Chk: "bluff:11", Win: "coin:12;rep:d:+3;hon:-1;merit:1;xp:8", WT: "The guard reads the name and lets you through. It was, in fact, a letter.", Lose: "coin:-3;hp:-3;rep:d:-6", LT: "The guard finds more than a letter. He does not care for what he finds.", WN: "", LN: ""},
		{T: "Slip him a coin.", Req: "sq:coin:4", Win: "coin:-4;coin:12;hon:-1", WT: "The coin makes a small noise. So does the gate.", WN: ""},
	}},

	// ------------------------------------------------------------ envoy
	{ID: "e_start", Text: "Both camps want to talk, and neither will say so first. {Cap} on the wall and {Off} in the tents each have a message for the other, and each would pay a stranger to carry it with no flag on their back.", Ch: []sqCh{
		{T: "Carry the defenders' terms to {Off}.", Win: "", WT: "The messenger's white cloth is hung on a pole over your shoulder.", WN: "e_def"},
		{T: "Carry {A}'s demand to {Cap}.", Win: "", WT: "A scroll with a seal of black wax. You are told to read it first, so you will not be surprised.", WN: "e_att"},
		{T: "Offer to broker a truce yourself.", Req: "sq:phase:encircle", Win: "", WT: "Neither side has asked. Both will listen.", WN: "e_broker"},
		{T: "Arrange a ransom for the prisoners on both sides.", Win: "", WT: "There are men in cellars and men in tents. Someone must be willing to be the middle.", WN: "e_ransom"},
	}},
	{ID: "e_def", Text: "{Off} sits under an awning with a map held down by stones. The terms are read aloud twice. The officer's face tells you nothing at all.", Ch: []sqCh{
		{T: "Press for a month's truce.", Chk: "persuasion:11", Win: "coin:10;truce:+22;rep:d:+6;rep:a:+4;merit:3;xp:10", WT: "\"A month,\" says the officer slowly. \"I will take it to the marshal.\" It is not a yes. It is not a no.", Lose: "coin:3;truce:+3;rep:a:-2", LT: "The officer refuses and has your pack searched on the way out of the camp.", WN: "", LN: ""},
		{T: "Read what they will really accept.", Chk: "empathy:10", Win: "coin:4;truce:+8", WT: "Not what they say but how they say it. There is a price, and it is lower than the one they named.", Lose: "truce:+1", LT: "The officer is closed to you. You come away with nothing.", WN: "e_terms", LN: ""},
		{T: "Lie about the strength of the garrison.", Chk: "bluff:12", Win: "coin:8;prog:-6;truce:+6;hon:-1;xp:10", WT: "\"They have six hundred on the wall,\" you say, and it is a fine lie, and it holds. The officer asks for another day to think.", Lose: "prog:+3;truce:-4;rep:d:-4", LT: "The officer knows the real number, and says it. You are escorted out with your lie hanging in the air.", WN: "", LN: ""},
	}},
	{ID: "e_att", Text: "{Cap} reads the demand slowly, and then once more. The sound of the camp comes through the slit of the window. \"Tell them what they want is more than the wall is worth,\" says the captain, \"but let me hear what you think.\"", Ch: []sqCh{
		{T: "Persuade {Cap} to open the gates for a fair price.", Chk: "persuasion:12", Win: "coin:12;truce:+18;rep:d:+4;rep:a:+8;merit:3;xp:12;mer:+1", WT: "The captain is not persuaded, but the captain has a council, and the council has been awake for nine nights.", Lose: "coin:3;truce:-2;rep:a:-2", LT: "\"I will not parley with a messenger who does not know what they carry.\"", WN: "", LN: ""},
		{T: "Tell {Cap} the real state of the camp.", Chk: "bluff:11", Win: "mor:+4;prog:-3;rep:d:+10;rep:a:-12;hon:-1;xp:10;coin:8", WT: "Fevers in the lines. Rations short. The marshal quarrels with the sapper-captain. {Cap} listens with the stillness of someone adding a column of figures.", Lose: "rep:a:-4;rep:d:-2", LT: "The captain does not believe you, and the camp, when it hears, believes it even less.", WN: "", LN: ""},
	}},
	{ID: "e_terms", Text: "You know now what each side will take, and what neither will say. A compromise exists, thin as a plank over a ditch. You are the only one who can see both ends of it.", Ch: []sqCh{
		{T: "Carry the compromise back honestly.", Chk: "persuasion:10", Win: "coin:12;truce:+22;rep:d:+5;rep:a:+5;merit:3;xp:12;mer:+1;hon:+1", WT: "Honest words, plainly put. It takes two days, and neither side loves it, but both sign.", Lose: "coin:3;truce:+4", LT: "Each side suspects what the other will keep. You carry it back and forth three times.", WN: "", LN: ""},
		{T: "Let each side think the other has already agreed.", Chk: "bluff:12", Win: "coin:15;truce:+34;hon:-2;xp:14", WT: "A small lie told in two tents. By morning they are both too proud to retract, and the truce is a fact.", Lose: "truce:-10;rep:d:-5;rep:a:-5;hon:-1", LT: "Each side finds the other's lie in the same hour. The truce is dead and you are not welcome in either camp.", WN: "", LN: ""},
	}},
	{ID: "e_broker", Text: "You gather the right people in the right tent: a priest from {T}, a quartermaster from the camp, one officer from each side. Between them the dead are mentioned only as a figure.", Ch: []sqCh{
		{T: "Argue that the cost of the siege is more than the town is worth.", Chk: "persuasion:12", Win: "coin:18;truce:+26;rep:d:+6;rep:a:+6;merit:4;xp:16", WT: "Numbers, in the end. The camp is sick, the town is hungry and both sides know it. By evening there are marks on paper.", Lose: "coin:4;truce:+4", LT: "The numbers are good, and nobody will say them aloud. The tent empties.", WN: "", LN: ""},
		{T: "Cite the old custom of truce at the harvest.", Chk: "lore:11", Win: "coin:14;truce:+20;rep:d:+4;rep:a:+4;merit:3;xp:14", WT: "The priest nods, and then the officer, who has not been to temple in a decade. Custom is a rope that no one wants to be seen cutting.", Lose: "coin:3;truce:+3", LT: "You remember the custom wrongly and are corrected by a priest.", WN: "", LN: ""},
	}},
	{ID: "e_ransom", Text: "The prisoners are a handful: a courier taken at the road, three archers, a young lieutenant of {A} and a clerk of {D} with a good head for figures. Each side wants its own back and neither will trade first.", Ch: []sqCh{
		{T: "Work out an even exchange.", Chk: "persuasion:10", Win: "coin:14;truce:+8;rep:d:+5;rep:a:+5;mer:+1;merit:2;xp:10", WT: "Man for man, a day apart. You stand at the midpoint between the lines while they cross, and they look at each other and not at you.", Lose: "coin:2;truce:+1", LT: "Neither side trusts the middle. You return with the prisoners where they began.", WN: "", LN: ""},
		{T: "Take a broker's fee from both.", Chk: "bluff:11", Win: "coin:24;truce:+4;rep:d:-2;rep:a:-2;hon:-1", WT: "Two purses, no questions. Both sides get what they asked for, a little late and a little short.", Lose: "coin:4;rep:d:-3;rep:a:-3", LT: "Both sides learn that you took two fees. Neither is amused.", WN: "", LN: ""},
	}},

	// ------------------------------------------------------------ relief
	{ID: "r_start", Text: "Captain {Cap} presses a sealed letter into your hand. \"The nearest lord is at {Dest}. The roads are watched. Every day matters. If the lord moves, this wall will hold, and if not, I do not know.\"", Ch: []sqCh{
		{T: "Take the letter and ride for {Dest}.", Req: "sq:town", Win: "ride:start;side:d", WT: "The seal is still warm. They open a side gate for you and shut it behind.", WN: ""},
		{T: "Ask for a fast horse first.", Req: "sq:town,sq:coin:8", Win: "coin:-8;ride:fast;side:d", WT: "A bay mare with a mean eye and a long stride. The stable-master does not ask for the horse back.", WN: ""},
		{T: "Say you will think about it.", Win: "", WT: "\"Think quickly,\" says the captain.", WN: ""},
	}},
	{ID: "r_arrive", Text: "The lord's hall at {Dest} is full of men who would rather it were someone else's problem. You hand the letter to a steward who reads it aloud, and a captain asks, \"How long can {T} hold?\"", Ch: []sqCh{
		{T: "Argue that the relief must march now.", Chk: "persuasion:11", Win: "relief:+38;rep:d:+15;coin:20;merit:6;xp:20;ride:end;news:A rider from {T} has reached {Dest} begging for relief.", WT: "You are heard, and the room goes quiet, and a horn sounds in the courtyard. By evening the banners are out.", Lose: "relief:+14;rep:d:+5;coin:8;merit:2;ride:end", LT: "They promise a force. It will take a week to gather. Better than nothing.", WN: "", LN: ""},
		{T: "Lay out the roads and the weak flank of the camp.", Chk: "lore:10", Win: "relief:+32;rep:d:+12;coin:16;merit:5;xp:16;ride:end", WT: "A map on the table, stones for the camps. The captain begins to move the stones and then everyone else does.", Lose: "relief:+12;rep:d:+4;coin:6;ride:end", LT: "Your map is wrong in two places. They send a smaller force than you wanted.", WN: "", LN: ""},
		{T: "Hand over the letter and say nothing more.", Win: "relief:+16;rep:d:+8;coin:8;ride:end", WT: "Letters have their own weight. A small force is promised.", WN: ""},
	}},
}

// ---------------------------------------------------------------- news and talk lines

var sqNewsTpl = map[string]string{
	"muster":   "{A} is massing troops outside {T}. The roads fill with carts and hired swords.",
	"encircle": "{A}'s army has closed the roads around {T}. The market gates are shut.",
	"assault":  "{A} has begun to batter the walls of {T}.",
	"breach":   "A stretch of the wall of {T} has come down. The fight is in the breach.",
	"fall":     "{T} has fallen to {A}. The banners over the gate have changed.",
	"sack":     "{A}'s soldiers are loose in {T}.",
	"relief":   "A relief force from {D} has broken the siege of {T}.",
	"lifted":   "{A}'s army has marched away from {T}. The siege is lifted.",
	"truce":    "{A} and {D} have agreed a truce at {T}. The gates are open again.",
	"plague":   "Sickness in the camp has broken the siege of {T}. {A}'s army is leaving.",
}

var sqPhaseDesc = map[string]string{
	"muster":   "The army of {A} is gathering outside {T}. Tents, wagons and drilled lines. No blow has been struck yet.",
	"encircle": "The roads are cut. Nothing goes in or out of {T} and the stores begin to be counted.",
	"assault":  "The engines are in place. The walls of {T} are being hit day and night.",
	"breach":   "The wall has a gap in it, and the fight is in the gap.",
	"sack":     "The wall has been taken and {A}'s soldiers are in the streets. Anyone who can leave should leave.",
}

var sqGuardLines = map[string][]string{
	"muster":   {"The gate is shut by order. No one in or out without the captain's word.", "Banners and wagons out on the plain. The gate is shut by order until we know what they want."},
	"encircle": {"The gate is shut by order. They are all round us now.", "The gate is shut by order. Keep your head down on the wall."},
	"assault":  {"The gate is shut by order. If you want to help, report to the captain.", "The gate is shut by order. There is no way through while the engines are working."},
	"breach":   {"The gate is shut by order and the wall has a hole in it. Report to the captain or get below.", "The gate is shut by order. Pray the masons are faster than their engines."},
	"sack":     {"There is no order any more. Go while you can.", "The wall is theirs. Go by the postern if it is still free."},
}

var sqFolkLines = map[string][]string{
	"muster":   {"They say a great army is coming. The price of bread has already gone up.", "The carts are leaving. Those who can are leaving with them."},
	"encircle": {"Nothing comes in. We have flour for a month, and no one dares to count the cattle.", "The wells are fine. The grain is not."},
	"assault":  {"Fire at night. Every night. We have stopped sleeping.", "They say the wall will hold. They say it quietly."},
	"breach":   {"The wall is down on the north side. Do not go there.", "They are in the breach. They are in the breach."},
	"sack":     {"Run. They are in the streets. Run.", "I lost the children in the crowd. Have you seen two little ones?"},
}

var sqOccLines = []string{
	"The banners of {A} hang from the gate now. The people keep their heads down.",
	"The soldiers of {A} patrol the streets. We were taxed twice before they learned our names.",
	"We held for a long time. It did not matter.",
	"The gates are open. That is the worst of it: they are open and nobody wants to go through.",
}

var sqRefugeeLines = []string{
	"They burned our village for the wood. We walked three days to get here, and they shut the gate.",
	"My brother is on the wall. I do not know where he is now.",
	"The stores are low. They give us a quarter loaf and call it a feast.",
	"The soldiers took our cart and our cow. We were glad they took nothing else.",
}

var sqCampLines = []string{
	"Another week of this and I will be eating my belt.",
	"The captain says the wall comes down any day now. The captain has been saying it for a month.",
	"Half the camp has the flux. Do not drink from the stream below the horse lines.",
	"They will give us the town to sack when it falls. That is what we are here for.",
}

var sqWays = []string{"night boat", "old tunnel", "drain under the north wall"}
