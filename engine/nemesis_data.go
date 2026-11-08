package engine

// Nemesis text and tables (prefix nx). Placeholders: {N} nemesis, {P} player, {R} player's realm, {W} wrong done.

// nxScar is what a lost fight leaves on the nemesis: a mark in the dossier and a stronger body next time.
type nxScar struct {
	Name string
	HP   int // percent more health
	AV   int
	DV   int
	Flag string // regen fire frost
}

var nxScars = []nxScar{
	{"a half-burned face, which they wear like a banner", 10, 1, 0, ""},
	{"a notched blade-arm that heals wrong and hits harder", 5, 2, 0, ""},
	{"an iron plate screwed over a cracked skull", 15, 2, 0, ""},
	{"a stolen fire-trick, learned in the weeks between", 10, 0, 0, "fire"},
	{"a frost-oath sworn to a cold god", 10, 0, 1, "frost"},
	{"a knitting of black veins that closes wounds", 15, 0, 0, "regen"},
	{"a lost eye and a sharper aim for it", 5, 0, 2, ""},
	{"a new, heavier armour hammered from your own broken steel", 10, 3, 0, ""},
	{"a bitter discipline learned from a hermit", 10, 1, 1, ""},
	{"a hundred-day fast that left them lean and fast", 5, 0, 3, ""},
}

var nxEscortTpl = []string{"bandit", "bandit", "archer", "bandit"}

var nxTaunts = []string{
	"\"{P}. I have crossed {W} to find you. Tonight your name ends.\"",
	"\"You walk like {R} still owes you something. It owes me more.\"",
	"\"I was patient for years. I am done being patient, {P}.\"",
	"\"Run if you like. I will be at the next fire.\"",
	"\"Do you even know what your people did? I carry it every day.\"",
	"\"They told me you were dangerous. They told me nothing about your face. I will remember it.\"",
	"\"This road is mine now, {P}. Pay the toll in blood.\"",
	"\"Count your scars. I intend to add one more.\"",
}

var nxScarTaunts = []string{
	"\"You gave me {S}. I came back to thank you.\"",
	"\"{S}. That is your doing, {P}, and I wear it for you.\"",
	"\"Look at {S}. Look well. Did you think I would stay down?\"",
}

var nxTruceLines = []string{
	"\"Hm. Nobody ever tried words. Very well: until the days run out, I keep my hand off my hilt.\"",
	"\"You speak plainly. I will not forgive you. I will wait.\"",
	"\"Fine. A few days. Do not mistake it for peace.\"",
}

var nxFailLines = []string{
	"\"Pretty words. They did not feed my dead.\" Steel comes out.",
	"\"Your people said as much, and then they burned us.\" They lunge.",
	"\"No. Not you, not now, not ever.\" Their retinue closes in.",
}

var nxYieldLines = []string{
	"\"Enough... enough. I yield. Finish it, or do not.\"",
	"\"You win the day, {P}. Choose what that is worth.\"",
}

var nxFleeLines = []string{
	"{N} snarls \"This is not over!\" and is gone into the confusion.",
	"{N} throws down a smoke-pot, curses your name, and vanishes.",
	"{N} drags themself clear and slips away, bleeding and sworn.",
}

var nxSpareLines = []string{
	"\"I do not understand you. But I owe you my life. My blade is yours.\"",
	"\"A debt, then. I will walk with you until it is paid.\"",
}

var nxKillTitles = []string{"the Unforgiving", "the Patient", "the Hollow Oath", "Who Remembers", "the Long Grudge", "the Avenger"}
