package engine

// Companion banter lines (compbanter.go). Placeholders: {me} {you} {town} {kin} {realm} {faith}
// {place} {house} {coin}, plus per trigger {foe} {feast} {feasttown} {weather} {other} {otherrealm}
// {othertown} {victim}. Tags: old / tech (genre group), g:<genre>, sp:<species>, high / low (caste),
// kin (has kin at home), faith.
var cbLines = map[string][]cbLine{
	"home": {
		{"", "{town}. It looks smaller than I remember. Or I got bigger."},
		{"", "Home. Give me a moment, {you}. I want to remember it like this."},
		{"kin", "My {kin} will be somewhere about. Don't tell them what we've been doing."},
		{"kin", "If my {kin} asks, you're a respectable sort and I eat properly."},
		{"high", "They'll bow when they see me. Pay it no mind, {you}."},
		{"low", "Back in {town}, where they'd still have me sweeping the gutters. Not today."},
		{"tech", "Same smoke, same noise. {town} never changes, it just gets louder."},
		{"old", "The bells of {town}. I'd know them anywhere."},
		{"", "Everyone here knows the {house} name. Try not to ruin it."},
	},
	"realm": {
		{"", "{realm}'s banners. Feels good to be among my own."},
		{"", "We're in {realm} lands. People here will treat us fair."},
		{"high", "My family has friends in {place}. Doors open for us here."},
		{"low", "{realm} again. Same lords, different town."},
		{"tech", "{realm} territory. At least the permits here are in my name."},
	},
	"enemy": {
		{"", "{foe}. We're at war with these people, {you}. Watch your back."},
		{"", "Keep your voice down. If they hear a {realm} accent in {place}, there'll be knives."},
		{"", "I've lost friends to {foe}. Don't expect me to smile here."},
		{"high", "My house has blood debts with {foe}. Let's not linger."},
		{"tech", "{foe} checkpoints everywhere. Smile, keep walking."},
	},
	"temple": {
		{"", "A house of {faith}. Give me a moment to pray."},
		{"", "I haven't lit a candle since {town}. The {faith} will understand."},
		{"faith", "The priests here keep the {faith} properly. My {kin} would approve."},
		{"low", "Even the likes of me are welcome before the {faith}. That's why I keep it."},
		{"tech", "Still a few believers left in this age. The {faith} endures."},
	},
	"wonder": {
		{"", "{place}. I heard of it as a child and never thought I'd see it."},
		{"", "They tell stories of {place} back in {town}. None of them do it justice."},
		{"old", "The old songs were true, then. {place} stands."},
		{"tech", "{place}. All our machines and we've built nothing like it since."},
		{"kin", "My {kin} won't believe me. I've stood at {place}."},
	},
	"ruin": {
		{"", "Whoever lived in {place}, they didn't leave by choice."},
		{"", "Read the stones if you like. I'll keep watch. Places like this have long memories."},
		{"old", "Walk softly. The dead of {place} were never sung to rest."},
		{"tech", "Scavenged to the bones. Someone got here first."},
		{"faith", "Let the {faith} keep the souls of {place}."},
	},
	"wx_storm": {
		{"", "That's a proper {weather}. We should find a roof, {you}."},
		{"", "My {kin} always said storms come for the proud. Let's not stand on the hill."},
		{"old", "The sky is angry. Someone has broken an oath."},
		{"tech", "Lightning. Don't hold anything metal up, eh?"},
	},
	"wx_snow": {
		{"", "{weather}. My fingers are going numb."},
		{"", "Snow like this back in {town} meant a week by the fire. No such luck."},
		{"sp:dwarf", "Snow. Give me a tunnel and a forge any day."},
	},
	"wx_rain": {
		{"", "Wet to the bone. Grand."},
		{"", "{weather}. At least it keeps the bandits indoors."},
		{"tech", "This {weather} eats through boots. I'll need new ones."},
	},
	"wx_heat": {
		{"", "This heat. Drink while you can, {you}."},
		{"", "Back in {town} we'd sleep through the afternoon in heat like this."},
	},
	"wx_fog": {
		{"", "Can't see my own hand in this {weather}. Stay close."},
		{"old", "Fog like this is when the hollow folk walk. My {kin} swore it."},
	},
	"wx_omen": {
		{"", "A red moon. Nothing good comes of that."},
	},
	"feast": {
		{"", "It's the {feast}! Back in {town} we'd be dancing by now."},
		{"", "The {feast} in {feasttown}. Let's join them, {you}. One night won't hurt."},
		{"kin", "My {kin} makes the best food for the {feast}. This lot can't compete."},
		{"high", "In my family the {feast} meant long speeches. I prefer this."},
		{"low", "The {feast}. The one day the lords let us eat our fill."},
	},
	"meal": {
		{"", "Now that's a meal. Thank you, {you}."},
		{"", "Good food, good company. Could be worse."},
		{"kin", "My {kin} cooked like this. Almost."},
		{"tech", "Real food. Not ration paste. I could weep."},
	},
	"camp": {
		{"", "A fire, a full belly, and the stars. Some nights the road is kind."},
		{"", "Tell me about where you're from, {you}. I've told you all about {town}."},
		{"kin", "I wonder if my {kin} is looking at the same sky tonight."},
		{"old", "My {kin} used to tell stories by a fire like this. Monsters, heroes, the {faith}."},
		{"low", "Nobody to order me about out here. Just the fire and you."},
		{"tech", "No lights, no noise. I'd forgotten how dark the world gets."},
	},
	"levelup": {
		{"", "I'm getting the hang of this."},
		{"", "Did you see that? I'm not the same {me} who left {town}."},
		{"kin", "My {kin} wouldn't recognise me now."},
	},
	"neardeath": {
		{"", "I'm hurt, {you}! Badly!"},
		{"", "Not here. Not like this!"},
		{"kin", "Tell my {kin}... no. I'm not done yet!"},
		{"faith", "{faith}, keep me!"},
	},
	"grumble": {
		{"", "I didn't leave {town} for this."},
		{"", "Do you even know where we're going, {you}?"},
		{"kin", "My {kin} warned me about wanderers like you."},
		{"high", "I am not accustomed to being treated like a pack mule."},
	},
	"kill": {
		{"", "That's {victim} done."},
		{"", "One less!"},
		{"", "For {town}!"},
		{"old", "Back to the dark with you!"},
		{"tech", "Down and out."},
		{"high", "That is how a {house} fights."},
	},
	"pair": {
		{"", "{other}, how do they do things in {othertown}?"},
		{"", "You snore, {other}. Just so you know."},
		{"", "Ever think of going home, {other}?"},
		{"", "What's the first thing you'll do when this is over, {other}?"},
		{"", "{other}, does {you} ever tell you where we're going?"},
	},
	"pair_re": {
		{"", "Better than in {othertown}, {other}, I can promise you that."},
		{"", "And you talk in your sleep, {other}. We're even."},
		{"", "Every day, {other}. Then I remember why I left."},
		{"", "Sleep for a week, {other}. Then eat for another."},
		{"", "Never, {other}. I've stopped asking."},
	},
	"pair_kin": {
		{"", "{other}, do you remember the old well in {town}?"},
		{"", "Your mother would skin us both if she saw us now, {other}."},
		{"", "Who'd have thought, {other}? Two of us from {town}, this far from home."},
		{"", "Do you think they still talk about us in {town}, {other}?"},
	},
	"pair_kin_re": {
		{"", "Remember it? I fell in it, {other}. Twice."},
		{"", "She'd skin you first, {other}. You were always the bad influence."},
		{"", "Not me, {other}. I always knew you'd drag me somewhere."},
		{"", "Only to say we'll come to a bad end, {other}."},
	},
	"pair_rival": {
		{"", "Your {otherrealm} burned my cousin's farm, {other}."},
		{"", "Funny. Back home I'd be paid to kill you, {other}."},
		{"", "When the war ends, {other}, which side of the line will you stand on?"},
	},
	"pair_rival_re": {
		{"", "And {otherrealm} hanged my uncle. Out here we fight the same things, {other}."},
		{"", "Same here, {other}. Let's not tell anyone at home."},
		{"", "Whichever side {you} is on, {other}. Same as you."},
	},
}
