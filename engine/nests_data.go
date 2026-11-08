package engine

// Text tables for nests.go (prefix nt): living-ruin occupants, ghost-town diaries and reveals.

type ntOccDef struct {
	ID    string
	Who   string // sentence for the ruin log
	Enter string // message on arrival after a change
	Chat  string // what an occupant says when asked about the ruin
}

var ntOccs = map[string]ntOccDef{
	"bandits": {"bandits", "A band of outlaws has made camp in the ruin.", "{{R|Woodsmoke and rough voices: outlaws have squatted in these ruins.}}",
		"\"Keep walking, stranger. This roof is ours now.\""},
	"monks": {"monks", "A hermit has taken up a simple life among the stones.", "{{G|A thread of incense drifts between the broken walls: a hermit keeps a shrine here.}}",
		"\"The old town is gone, but the silence is good company. Rest if you need to.\""},
	"den": {"den", "Something large has made its den here.", "{{R|Gnawed bones and a heavy musk: some beast has denned in the ruin.}}",
		""},
	"nomads": {"nomads", "Travellers have pitched their tents among the walls for the season.", "{{Y|Tents and cook-smoke: nomads have camped in the ruin for the season.}}",
		"\"We follow the grass. The ruin keeps off the wind, and the dead do not mind us.\""},
	"plague": {"plague", "Plague refugees shelter here. Keep your distance.", "{{y|Coughing from the rubble: refugees from a plague-struck road shelter in these ruins.}}",
		"\"Stay back. The fever is in our camp. If you have any tonic to spare...\""},
	"haunted": {"haunted", "The dead walk here at night. Do not stay.", "{{M|The air is wrong. Pale shapes drift where nobody should be: the ruin is haunted.}}",
		""},
}

// ntCauseWords maps chronicle keywords to a ghost-town cause.
var ntCauseWords = []struct {
	Cause string
	Words []string
}{
	{"plague", []string{"plague", "pestilence", "fever", "disease", "famine", "blight", "epidemic", "poison"}},
	{"curse", []string{"curse", "sorcer", "demon", "cult", "ritual", "witch", "necro", "doom"}},
	{"beast", []string{"dragon", "beast", "monster", "devour", "slain", "wyrm", "horror"}},
	{"lord", []string{"sack", "raid", "bandit", "siege", "war", "burn", "conquer", "rebel", "outlaw", "usurp"}},
}

var ntCauseNames = map[string]string{"plague": "a plague", "curse": "a curse", "beast": "a beast", "lord": "a bandit lord"}

var ntWriters = []string{"Edda", "Corvin", "Maren", "Tobiah", "Ysolde", "Brannoc", "Liese", "Hadrik", "Orla", "Wenn", "Iven", "Sabine"}

// ntDiaries: three pages per cause. {burg} {who} {year} {event}
var ntDiaries = map[string][3]string{
	"plague": {
		"The fever came in {year} with the grain carts. Nobody in {burg} thought much of it. {event} The healers say wash and wait. I have started to count the carts that leave for the burying ground; there is a cellar under the old well where we keep what matters. - {who}",
		"We have sealed the lower doors. The sick are put in the cellar so the rest can breathe. I hold the key and the key is heavy. Half of {burg} coughs. The bells stopped on the ninth day. - {who}",
		"I went down tonight. They are not quiet. The fever did not stop at death in {burg}; it learned to stand. If you read this, burn the cellar, or give them rest. I am so tired. - {who}",
	},
	"curse": {
		"It began with the well in {year}. {event} Water went bitter, then the birds fell silent over {burg}. Elders whisper of a debt owed. I keep a spare key under the cellar hatch, by the old well. - {who}",
		"A man in grey walked through {burg} and nobody saw his face. By morning three houses were empty and their doors barred from inside. The priest will not leave the cellar. - {who}",
		"It is bound to the cellar and bound to us. Every soul that dies in {burg} stays. The one who laid the curse still stands down there, waiting for someone to answer. I will try. - {who}",
	},
	"beast": {
		"Something took the herds in {year}, then the herdsmen. {event} The council debates a hunt while {burg} bars its doors. The old cellar by the well is the one place that holds. I keep the key on a cord. - {who}",
		"It came at night and tore off the roof of the mill. We hid in the cellar. It circled above, patient, and I heard it breathing through the floor. {burg} is a graveyard of shutters. - {who}",
		"It did not leave. It nested in the cellar after we fled and fed on whatever came. Whoever you are, it is wounded in its left foreleg and it hates fire. Bring friends. - {who}",
	},
	"lord": {
		"The riders came to {burg} in {year} and called it a tax. {event} Their captain took the guildhall and the keys of the cellar. I hid the spare under a loose stone beside the old well. - {who}",
		"He holds the cellar now, with the town's silver. Anyone who spoke against him was hanged on the gate. {burg} sends its young to his service or to the ravens. - {who}",
		"They fled in the spring and left him holding a town of the dead. Still he sits below with his hoard and his oath-breakers, counting. If you take it from him, take {burg}'s name back with it. - {who}",
	},
}

var ntReveal = map[string]string{
	"plague": "{{M|The cellar breathes cold. The sickness that emptied this town still stands below, fever-bright and furious.}}",
	"curse":  "{{M|A grey figure waits among the cellar shelves: the curse made flesh, and it knows your name.}}",
	"beast":  "{{R|Eyes catch the lantern light deep in the cellar. The beast that emptied this town never left.}}",
	"lord":   "{{R|Torches and a throne of crates: the bandit lord who broke this town still holds court in his cellar.}}",
}
