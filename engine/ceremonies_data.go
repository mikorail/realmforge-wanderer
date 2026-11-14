package engine

// ceremonies_data.go: text pools for the ceremonies you attend (ceremonies.go, prefix fw).

var fwToastsGood = []string{
	"You raise your cup and speak of the long road two people choose to walk together. The hall goes quiet, then roars.",
	"Your toast is short and well aimed; it makes the bride laugh and the groom blush, and the old women nod.",
	"You tell a small true story about the pair, and the whole company drinks to it before you finish.",
	"Words come easily: to bread shared, to doors left open, to quarrels slept off. Cups bang on the table.",
}

var fwToastsBad = []string{
	"You lose the thread halfway, repeat yourself, and end on a word that does not belong at a wedding. Someone coughs.",
	"The toast wanders into a story about your travels. By the end nobody is looking at you, and the couple are not smiling.",
	"You mean well, but the joke lands on the wrong person. A long silence follows.",
}

var fwEulogyGood = []string{
	"You speak plainly of the dead: what they made, what they put up with, what they were kind about. People weep, and are glad of it.",
	"Your words find the shape of a life in a few sentences. An old friend of the dead grips your arm afterwards.",
	"You say what the family could not. The grief in the yard loosens a little.",
}

var fwEulogyBad = []string{
	"The words will not come right. You mix up a name, and a mourner turns away.",
	"Your eulogy sounds like one you could give for anyone. The kin are polite about it.",
}

var fwConsoleLines = []string{
	"You sit with them and say little. They tell you about the last winter, the small things, what was left unsaid.",
	"You hold a hand and listen. After a while they stop apologising for crying.",
	"You offer to help carry the bier-cloth and fetch water, and in the doing find something to say.",
}

var fwBlessings = []string{
	"You touch the child's brow and wish them a long road and a warm door at the end of it.",
	"You speak the old words you half-remember. The priest does not correct you, which is its own kind of blessing.",
	"You say what you wish for them: good luck, stubbornness, and people who stay.",
}

var fwStoryIntros = []string{
	"The young one sits on a barrel and demands a story, a real one, not a nursery tale. You tell about",
	"A small crowd drifts over when the youngster says you have been places. You tell them of",
	"\"Tell me something from out there,\" the young one says. You tell of",
}

// fwKeepsakes: small lore items kin press into your hand at a wake.
var fwKeepsakes = []struct{ name, lore string }{
	{"carved bone comb", "A comb with a notch worn in it by one thumb. Pressed on you at a wake: it belonged to the dead."},
	{"knotted mourning cord", "Three knots, one for each year the dead was known to the family. Given to you at their wake."},
	{"tin pilgrim badge", "A pilgrim badge, dented. The dead wore it to the end; the family gave it to you to carry further."},
	{"pressed-flower sachet", "Dry petals in a scrap of cloth. It smells faintly of a garden you will never see."},
	{"worn knife-sheath", "A well-oiled sheath with no knife. The family kept the blade and gave you the rest."},
	{"stub of a name-candle", "The last of the candle lit at the naming of the dead, saved for exactly this day."},
}

var fwHeirloomTalk = []string{
	"An aunt mentions that the dead kept a hidden tin under the hearthstone, and that the family would rather it stayed hidden.",
	"A cousin lets slip that an old ring is meant for the eldest grandchild, if the quarrel over it is ever settled.",
	"One of the kin says the dead once swore the family never to sell the house on the river, whatever was offered.",
}

var fwScandalPool = []string{
	"The priest asks, as the rite requires, whether anyone knows a reason these two should not be wed. The whole yard holds its breath.",
}

var fwAmbient = map[string][]string{
	"wedding": {"Garlands hang from every post and someone is tuning a fiddle.", "A child throws petals at nobody in particular.", "The air smells of bread and lamp oil."},
	"funeral": {"A bell tolls at long intervals.", "Mourners stand in low-voiced knots. The smoke of the braziers drifts across the paving.", "Black cloth hangs over the bier; someone has laid a sprig of rosemary on it."},
	"birth":   {"A cradle stands by the altar, hung with ribbons.", "Women pass the baby along the line to be admired.", "A drum taps a slow, happy rhythm."},
	"age":     {"The youth stands very straight, trying not to look nervous.", "Relatives press small gifts into the youth's hands.", "Somebody has baked a flat loaf marked with the youth's name."},
	"feast":   {"Pilgrims carry lanterns and chant in a low unison.", "The priest holds a staff wound with ribbons; the line behind it moves slowly.", "Incense hangs in the air."},
}

var fwDecoNotes = map[string]string{
	"altar":   "A rite is held here. The altar is dressed in cloth and flowers.",
	"bier":    "A black-draped bier. The dead lie in state here.",
	"cradle":  "A cradle with ribbons for the naming.",
	"table":   "A long table laid with the food of the day.",
	"brazier": "A brazier burns, giving light and smoke.",
	"shrine":  "The shrine the procession is walking to.",
}

var fwFeastShrine = []string{"the old shrine", "the stone of vows", "the well-chapel", "the hill-cairn"}
