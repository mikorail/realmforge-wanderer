package engine

import "strings"

// Data tables for etiquette.go (et): act labels, rebuke fallbacks, duties, judge flavour.

var etActLabel = map[string]string{
	"draw_weapon_indoors": "drew a weapon under a roof", "refuse_hospitality": "refused hospitality",
	"refuse_gift": "refused a gift", "work_on_holy_day": "worked on a holy day", "bare_head": "went bareheaded",
	"bare_arms": "went bare-armed", "public_anger": "lost your temper in public", "eat_unclean": "ate forbidden food",
	"eat_alone": "ate alone at a shared table", "enter_unbidden": "walked in unbidden", "turn_back": "turned your back on an offer",
	"break_oath": "broke an oath", "haggle_sacred": "haggled on sacred ground", "shirk_duty": "shirked a guest's duty",
	"waste_food": "wasted food", "drink_to_excess": "drank to excess",
}

var etSevWord = []string{"", "a frown", "an insult", "blood or banishment"}

var etRebuke = [4][]string{
	nil,
	{"Not here, stranger. We do not do that.", "Mind yourself. That is not done in {town}.", "Your manners, traveller."},
	{"Have you no shame? Not in the land of the {culture}!", "You insult us, and this house, and this town.", "Only a barbarian would do that before {culture} eyes."},
	{"Seize them! The custom is broken!", "That is a blood matter. The law of {town} will answer.", "You have fouled the custom of {realm}. You will pay."},
}

var etPenaltyText = map[string]string{
	"none": "no more than a hard look", "fine": "a fine", "scorn": "public scorn", "exile": "exile from the town",
	"lash": "the lash", "mutilation": "mutilation", "hang": "the rope", "weregild": "weregild paid in coin",
	"ordeal": "an ordeal", "enslave": "bondage", "duel": "a duel",
}

var etKindWord = map[string]string{
	"greeting": "Greeting", "gesture": "Gesture", "gift": "Gift", "food": "Food", "speech": "Speech",
	"dress": "Dress", "space": "Space", "time": "Time",
}

// etDutyKinds maps keywords found in a pack's guest duties to the duty actions offered in talk.
var etDutyKinds = []struct {
	Key   string
	Words []string
}{
	{"tale", []string{"tale", "story", "song", "sing", "poem", "verse", "recite"}},
	{"news", []string{"news", "word", "tidings", "report"}},
	{"gift", []string{"gift", "token", "present", "offer", "leave a"}},
	{"chore", []string{"chore", "work", "wood", "water", "labour", "labor", "field", "sweep", "help", "tend"}},
}

var etDutyOption = map[string]string{
	"tale": "Tell a tale to your host", "news": "Share news from the road with your host",
	"gift": "Leave a gift for your host", "chore": "Do a chore for your host",
}

var etOfferLines = []string{
	"Stranger, the night is cold and the road long. Come in and share {item} with us.",
	"You look worn through, traveller. Sit at our hearth: {item} is yours.",
	"No guest of ours sleeps on the road. Take {item}, and a roof besides.",
}

var etDrinkWords = []string{"ale", "beer", "wine", "mead", "spirit", "brandy", "sake", "arrack", "rum", "whisky", "vodka",
	"liquor", "kumis", "soju", "toddy", "cider", "grog", "absinthe", "pulque", "chicha", "stout", "lager", "gin"}

// etJudge reads a judge description: dc added to pleas, fee multiplier (percent), a flavour line.
func etJudge(j string) (int, int, string) {
	l := strings.ToLower(j)
	has := func(ws ...string) bool {
		for _, w := range ws {
			if strings.Contains(l, w) {
				return true
			}
		}
		return false
	}
	switch {
	case has("king", "queen", "lord", "ruler", "emperor", "chief", "khan", "sultan", "tyrant", "warlord", "prince"):
		return 2, 120, "The seat is high and the temper short."
	case has("priest", "cleric", "oracle", "god", "temple", "abbot", "imam", "druid"):
		return 1, 100, "The judge speaks as one who weighs sins as well as deeds."
	case has("merchant", "guild", "factor", "company"):
		return 1, 130, "Everything here has a fee, and the fee has a clerk."
	case has("assembly", "moot", "thing", "jury", "peers", "council", "elders", "circle"):
		return -1, 90, "The assembly listens, and remembers who was kind."
	}
	return 0, 100, "The court sits and waits."
}

var etCrimeNames = []string{"trespass", "theft", "assault", "murder"}
