package engine

// Data for generated literature (literature.go, prefix gl). Template placeholders:
// {title} {author} {realm} {place} {year} {fact} {clue} {lover} {deity} {ruler} {subj} {foe} {n} {adj}.

type glForm struct {
	Name   string   // what the work is called
	Titles []string // title grammar
	Open   []string // opening paragraph
	Frame  []string // paragraph that carries {fact}
	Close  []string
	Biases []string // biases that suit the form, weighted by repetition
	Tone   string   // short tag shown to the buyer
}

var glForms = map[string]*glForm{
	"history": {
		Name: "history", Tone: "sober",
		Titles: []string{"A {adj} History of {subj}", "Annals of {subj}", "The {adj} Account of {subj}", "{subj}: A Chronicle in Several Books", "Concerning {subj}, Set Down for the Record"},
		Open:   []string{"The author, {author}, undertakes to set down what befell at {place}, so that those who come after are not left to rumour.", "It is the duty of the learned to record. I, {author}, have gathered the testimony of witnesses and weighed it as best I could."},
		Frame:  []string{"In the year {year}, as every ledger of {realm} will confirm: {fact}", "The record for the year {year} reads thus. {fact}", "I give the matter plainly. {fact} (Year {year}.)"},
		Close:  []string{"Here ends the book. Let no copyist improve upon it.", "So much for the year {year}. Later books must answer for later years."},
		Biases: []string{"propagandist", "partisan", "cynical", "pious", "pious", "romantic"},
	},
	"epic": {
		Name: "epic", Tone: "heroic",
		Titles: []string{"The {subj}: An Epic in {n} Cantos", "Song of {subj}", "The {adj} Lay of {subj}", "Arms and the Hearth: {subj}", "{subj}, Sung Entire"},
		Open:   []string{"Sing of {place}, and of the {n} nights when {realm} stood against all comers. So begins {author}, who was not there, but heard it from those who were.", "Let the harp be tuned to the old measure. {author} sings of {subj}."},
		Frame:  []string{"Then in the year {year} the deed was done, and the verse remembers it so: {fact}", "Hear what was wrought. {fact} Thus the singers fixed it in the year {year}."},
		Close:  []string{"The song is not finished; the song is never finished.", "Here the harp is laid down. Whoever wishes the rest may pay the singer."},
		Biases: []string{"romantic", "romantic", "propagandist", "pious"},
	},
	"sermon": {
		Name: "sermon", Tone: "devout",
		Titles: []string{"On the Judgement of {subj}", "A Sermon Preached After {subj}", "The {adj} Warning: {subj}", "Wrath and Mercy at {place}", "Seven Lessons from {subj}"},
		Open:   []string{"Brothers and sisters of {realm}, I, {author}, come to you with a heavy heart, bearing the lesson of {place}.", "Hear the word, and hear it with an open heart. {author} speaks."},
		Frame:  []string{"In the year {year} {deity} turned a face upon us, and the world saw it: {fact}", "Do you doubt that it was ordained? {fact} Mark the year, {year}, and mark the sin that went before."},
		Close:  []string{"Go, and sin less. The offering box is by the door.", "Let the faithful repeat it three times, and the faithless once, in private."},
		Biases: []string{"pious", "pious", "pious", "partisan", "cynical"},
	},
	"satire": {
		Name: "satire", Tone: "biting",
		Titles: []string{"The Merry Lies of {subj}", "How {realm} Won {subj}, by Those Who Lost It", "A Modest Account of {subj}", "The {adj} Fool's Guide to {subj}", "Lord {ruler} and the Matter of {place}"},
		Open:   []string{"{author} writes this for the pleasure of anyone who has ever been governed.", "Dear reader: everything below is true, except where it is funny, and the funny parts are truer."},
		Frame:  []string{"Consider the year {year}, when, according to the court historians, {fact} Note how closely the official account resembles a receipt.", "It is said in {realm}, with a straight face, that {fact} (Year {year}.) I have seen straight faces and I have seen this one."},
		Close:  []string{"I would sign this, but I enjoy my teeth.", "Burn it if you must. The good jokes survive fire."},
		Biases: []string{"cynical", "cynical", "cynical", "partisan"},
	},
	"travelogue": {
		Name: "travelogue", Tone: "curious",
		Titles: []string{"A Traveller's Notes on {place}", "Ten Roads to {place}", "{adj} Wanderings: {subj}", "What I Heard at {place}", "The Road Past {subj}"},
		Open:   []string{"{author}, a wayfarer of no fixed address, sets down what the innkeepers of {place} told over several suppers.", "I travelled light and listened heavily. This is what the road told me."},
		Frame:  []string{"They still talk of it there. In the year {year}, I was told: {fact}", "The locals showed me where it happened, and charged for the view. {fact} (Year {year}.)"},
		Close:  []string{"The inn was poor and the beer good. I recommend both.", "I left on the morning coach. The road goes on without me."},
		Biases: []string{"romantic", "cynical", "romantic", "pious"},
	},
	"lovepoem": {
		Name: "love poem", Tone: "tender",
		Titles: []string{"Lines for {lover}, Lost at {place}", "{lover}, Beloved of {place}", "A {adj} Letter in Verse", "Evening Songs After {subj}", "What {lover} Said at the Gate"},
		Open:   []string{"{author} writes, and the paper is wet. {lover} has been gone since {place}.", "Not every record is a ledger. Some are a hand held out across years."},
		Frame:  []string{"The year was {year}. The world noted this: {fact} I noted only that you were absent.", "In {year} they say that {fact} I say your door was shut and the lamp still lit."},
		Close:  []string{"If you read this, wait for me at the second bridge.", "I have no more lines. The rest is silence, which you always understood."},
		Biases: []string{"romantic", "romantic", "romantic", "pious"},
	},
	"dispatch": {
		Name: "military dispatch", Tone: "terse",
		Titles: []string{"Dispatches from {place}", "Orders of the Day: {subj}", "Field Report on {subj}", "To the High Command, Concerning {place}", "After-Action Notes, {place}"},
		Open:   []string{"FROM: {author}, in the field. TO: the high command of {realm}. SUBJECT: {subj}. PRIORITY: immediate.", "Report follows. I regret the state of the paper."},
		Frame:  []string{"Year {year}. Summary of events: {fact}", "Situation, year {year}: {fact} Recommend we see to the horses first."},
		Close:  []string{"Request reinforcement, rations and a better class of messenger.", "Will report again if the courier survives."},
		Biases: []string{"propagandist", "partisan", "cynical", "propagandist"},
	},
	"natural": {
		Name: "natural history", Tone: "observant",
		Titles: []string{"Observations on {subj}", "Of Beasts, Weather and {place}", "A Naturalist at {place}", "{adj} Notes on the Land Around {place}", "Field Studies After {subj}"},
		Open:   []string{"{author}, who prefers moss to men, records what the land at {place} did and what it is doing now.", "I took no side in the matter. The birds took none either; I envy them."},
		Frame:  []string{"The land bore the mark of it. In the year {year}: {fact} The soil, I note, has not forgotten.", "From the field notes, year {year}. {fact} Observed from a safe and cowardly distance."},
		Close:  []string{"Specimens are enclosed in the lower drawer. Please do not feed them.", "More observations will follow when the weather permits."},
		Biases: []string{"cynical", "pious", "romantic", "cynical"},
	},
	"memoir": {
		Name: "memoir", Tone: "personal",
		Titles: []string{"Memoirs of {author}", "What I Saw at {place}", "A Life Near {subj}", "{adj} Recollections", "I Was There: {subj}"},
		Open:   []string{"I, {author}, am old enough to be believed and young enough to be careless. Here is what I remember.", "They told me to write it down before I forget, and then told me what to forget."},
		Frame:  []string{"I remember the year {year} the way you remember a toothache. {fact}", "It was {year}. I stood where I could see. {fact} I have told this story at a hundred hearths and it improves each time."},
		Close:  []string{"That is all I can swear to. The rest I merely believe.", "My hand tires. The ink tires sooner."},
		Biases: []string{"partisan", "romantic", "cynical", "propagandist"},
	},
	"pamphlet": {
		Name: "pamphlet", Tone: "incendiary",
		Titles: []string{"The Truth They Burned: {subj}", "Notice to the People of {realm}", "Ten Questions on {subj}", "What the Court Will Not Say of {place}", "A {adj} Warning to {realm}"},
		Open:   []string{"Printed in haste and at risk. Whoever finds this, pass it on and do not name the printer. {author}.", "To the people of {realm}: they will tell you this is a lie. Ask yourself who profits from that."},
		Frame:  []string{"In the year {year}, {fact} Now ask who was paid, and by whom.", "It is written in the public record and ignored in the public square: {fact} (Year {year}.)"},
		Close:  []string{"Read it. Copy it. Burn the original if the watch is at the door.", "Pass this on. If you are caught with it, you never saw it."},
		Biases: []string{"partisan", "cynical", "propagandist", "pious"},
	},
}

// Which forms suit which kind of event.
var glKindForms = map[string][]string{
	"war":          {"history", "dispatch", "epic", "satire", "pamphlet", "memoir"},
	"capture":      {"dispatch", "epic", "history", "memoir", "pamphlet"},
	"peace":        {"history", "satire", "sermon", "travelogue"},
	"coup":         {"pamphlet", "satire", "memoir", "history"},
	"death":        {"memoir", "lovepoem", "history", "sermon", "satire"},
	"fall":         {"history", "epic", "sermon", "lovepoem", "memoir"},
	"found":        {"epic", "history", "travelogue", "memoir"},
	"disaster":     {"natural", "sermon", "travelogue", "memoir"},
	"eco_disaster": {"natural", "sermon", "pamphlet", "memoir"},
	"monster":      {"natural", "epic", "travelogue", "sermon"},
	"wonder":       {"travelogue", "natural", "epic", "lovepoem"},
	"faith":        {"sermon", "pamphlet", "history", "satire"},
	"god":          {"sermon", "epic", "natural", "pamphlet"},
	"intrigue":     {"pamphlet", "satire", "memoir", "dispatch"},
	"rebel":        {"pamphlet", "dispatch", "satire", "memoir"},
	"uprising":     {"pamphlet", "epic", "dispatch", "memoir"},
	"luminary":     {"epic", "lovepoem", "travelogue", "memoir"},
}

// Event kinds worth a book, with a weight for how famous they become.
var glKindScore = map[string]int{
	"fall": 6, "capture": 5, "war": 5, "coup": 5, "disaster": 5, "eco_disaster": 5, "peace": 4, "wonder": 4,
	"monster": 3, "faith": 3, "death": 3, "intrigue": 3, "god": 2, "found": 2, "rebel": 2, "uprising": 2, "luminary": 2,
}

// Subject phrases by event kind ({realm} {place}).
var glSubjects = map[string][]string{
	"war":          {"the {realm} War", "the Quarrel of {realm}", "the {place} Campaign"},
	"capture":      {"the Fall of {place}", "the Sack of {place}", "the Breaking of {place}"},
	"peace":        {"the Peace of {place}", "the Treaty at {place}", "the Quiet After {realm}'s War"},
	"coup":         {"the Coup in {realm}", "the Night of Knives in {realm}", "the Overthrow at {place}"},
	"death":        {"the Last Years of {realm}'s Ruler", "the Passing of the Crown of {realm}", "the Funeral at {place}"},
	"fall":         {"the End of {realm}", "the Last Days of {realm}", "the Throne Left Empty"},
	"found":        {"the Founding of {realm}", "the Making of {realm}", "the First Years of {realm}"},
	"disaster":     {"the {place} Calamity", "the Year of Ruin at {place}", "the Great Trouble of {place}"},
	"eco_disaster": {"the Burning of {place}", "the Ashen Season at {place}", "the Poisoning of {place}"},
	"monster":      {"the Beast of {place}", "the Thing Beneath {place}", "the Terror at {place}"},
	"wonder":       {"the Wonder of {place}", "the Marvel at {place}", "the Pilgrims' Prize of {place}"},
	"faith":        {"the Reform in {realm}", "the Altars of {realm}", "the Faith Question of {realm}"},
	"god":          {"the Gift to {realm}", "the Sign over {realm}", "the Divine Visit"},
	"intrigue":     {"the Shadow Affair of {realm}", "the Quiet War on {realm}", "the Whispering Council"},
	"rebel":        {"the {place} Uprising", "the Tithe Revolt of {place}", "the Refusal at {place}"},
	"uprising":     {"the {place} Uprising", "the Day {place} Rose", "the Stand at {place}"},
	"luminary":     {"the Triumph at {place}", "the Great Games of {place}", "the Hero of {place}"},
}

var glBiasAdj = map[string][]string{
	"propagandist": {"Glorious", "Triumphant", "Official"},
	"pious":        {"Blessed", "Sacred", "Humble"},
	"cynical":      {"Honest", "Candid", "Unvarnished"},
	"partisan":     {"Loyal", "Faithful", "Righteous"},
	"romantic":     {"Tender", "Golden", "Longing"},
}

// What each bias adds in the author's own voice (one paragraph).
var glBiasVoice = map[string][]string{
	"propagandist": {"Let it be understood that {realm} acted nobly, and that the enemy's losses were far greater than the enemy admits.", "No honest reader can doubt the justice of {realm}'s cause; those who do are in the pay of {foe}."},
	"pious":        {"None of this was chance. {deity} weighs every hand, and the sins of the powerful were counted before the first blow fell.", "Mark how the proud were humbled and the humble kept. This is the pattern of heaven."},
	"cynical":      {"Follow the money, as the old saying goes; behind every banner there is a counting-house and behind every counting-house a man who stayed home.", "Heroes are what is left when the survivors need a story. The dead are not consulted."},
	"partisan":     {"I will not pretend to be neutral. I stood with {realm}, and I would do so again, and {foe} may say what it likes.", "Those who call the other side's crimes accidents and our own accidents crimes have taught me what a fair account looks like. This is not one."},
	"romantic":     {"It was a time of grand feeling, and the sky itself seemed to lean closer to listen.", "I cannot recall the battle without recalling the light, and I cannot recall the light without weeping a little."},
}

// Clue paragraphs ({clue}).
var glClueFrames = map[string][]string{
	"site":      {"A traveller's aside: {clue}", "The old folk add that {clue}", "I will say this for free: {clue}"},
	"identity":  {"What the court does not print: {clue}", "I give this only to those who read to the end: {clue}"},
	"crime":     {"Here is the part the official histories drop: {clue}", "Ask any servant of the old palace and watch their eyes. {clue}"},
	"technique": {"Let me pass on a trade secret learned in that country: {clue}", "I append, for the practical reader: {clue}"},
	"lore":      {"One further matter worth the ink: {clue}", "For the record, and against the day this is disputed: {clue}"},
}

type glTech struct{ Text, Skill string }

var glTechs = []glTech{
	{"Bind the grip with wet cord; it shrinks tight as it dries and the weapon will never turn in the hand.", "cudgel"},
	{"Hold the knife low and cut upward; the guard forgets his belly.", "shortblade"},
	{"Set the feet as if at a doorway and let the blade come from the hip; it saves the shoulder.", "longblade"},
	{"Choke up on the haft and let the head's weight do the work; swing from the hips, not the arms.", "axe"},
	{"Breathe out on the release, and loose between heartbeats, never on one.", "missile"},
	{"Boil the spring in oil before fitting it, and the mechanism will outlast its maker.", "tinkering"},
	{"Always ask the second question first; the answer to the first is what they have rehearsed.", "persuasion"},
	{"Dig the fire-pit downwind and line it with stones; it will not betray you at dawn.", "survival"},
	{"Land on the outside of the foot and roll; the knees will thank you in old age.", "acrobatics"},
	{"Count steps, not distance; a road measured in steps does not feel as long.", "endurance"},
	{"Copy a passage twice, once from the page and once from memory, then compare. The differences are the lesson.", "lore"},
}

var glCrimes = []string{
	"{ruler} sold the border passes to {foe} in a bad winter and told the council it was a tax.",
	"the granary fire in {realm} was set on {ruler}'s order, so that the rebels might be blamed.",
	"{ruler} had the true heir smothered in the nursery and the nurse sent abroad.",
	"the ledgers of {realm} were altered at {ruler}'s direction so that the debts fell on the temples.",
	"{ruler} pays {foe}'s envoy every quarter-day, and nobody on the council dares ask what for.",
	"the plague quarantine was lifted early on {ruler}'s order because the harvest fair was profitable.",
}

var glIdentityFrames = []string{
	"{a}, now of {where}, is no other than {d}, whom the registers list as dead in the year {dy}. A false funeral was held.",
	"the quiet figure called {a} is the one the world reports dead, {d}. Study the hands; they have not changed since {dy}.",
}

var glRoleHint = map[string]string{
	"history": "chronicler", "epic": "bard", "sermon": "preacher", "satire": "pamphleteer", "travelogue": "wayfarer",
	"lovepoem": "poet", "dispatch": "field officer", "natural": "naturalist", "memoir": "retired courtier", "pamphlet": "printer",
}

var glLoverNames = []string{"Ysolde", "Marek", "Calia", "Teren", "Ilyan", "Sabe", "Orrin", "Neve", "Tamsin", "Halric"}
