package engine

// Text tables for knowledge.go (prefix kn). Plain data: no logic.

var knHeights = []string{"tall", "short", "stocky", "slight", "broad-shouldered", "wiry", "lanky", "heavyset"}

var knMarks = []string{
	"a grey cloak", "a red scarf", "a scar on the chin", "ink-stained cuffs", "a limp", "a missing finger",
	"a braided plait", "a patched boot", "a ring on a cord", "a hooded look", "a crooked nose", "a slow, careful walk",
}

// knTones are the way people pass things on, picked by personality in knTone.
var knToneExtra = []string{"leaning in, eager", "with a grin", "talking fast", "happily"}
var knToneSneer = []string{"with a sneer", "flatly", "without looking at you", "with a shrug"}
var knToneNerv = []string{"in a low, nervous whisper", "glancing over their shoulder", "wringing their hands", "very quietly"}
var knToneCons = []string{"carefully", "ticking it off on their fingers", "slowly, choosing words", "precisely"}
var knToneOpen = []string{"with relish", "eyes bright", "as if telling a tale", "with a wink"}
var knToneMild = []string{"mildly", "after a moment", "with a sigh", "idly"}

var knDirs = map[string]string{"north": "south", "south": "north", "east": "west", "west": "east", "left": "right", "right": "left", "upstream": "downstream", "downstream": "upstream"}

var knOpeners = []string{"I heard that", "They say", "Folk are saying", "The word is that", "Someone told me that", "It is said that"}

var knRefusals = []string{"\"I have nothing to say to you.\"", "\"Ask someone else.\"", "\"Not to you.\"", "\"Walk on, wanderer.\""}

var knNothing = []string{"\"Nothing worth the telling.\"", "\"I have heard little lately.\"", "\"Quiet days. Nothing to pass on.\"", "\"I keep to my own work, and hear little.\""}

// knCaseKinds maps a Realmforge event kind to the title of the matter it can turn into.
var knCaseKinds = map[string]string{
	"intrigue": "A plot whispered about",
	"death":    "A death left unexplained",
	"coup":     "The matter of the palace men",
	"capture":  "The disappearance",
}

var knMotives = []string{
	"was owed a debt they could not pay",
	"quarrelled bitterly with the dead man's house",
	"had been seen counting coin they should not have had",
	"was heard cursing the victim in the market",
	"had a grudge no one could quite name",
}
