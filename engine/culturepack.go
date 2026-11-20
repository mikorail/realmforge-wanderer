package engine

import "strings"

// The culture pack: optional per-culture depth shipped by Realmforge (customs, voice, faith lore,
// literature, material life, language) plus a small voice per notable person.
//
// SHARED SCHEMA. The TypeScript mirror is realmforge/src/lib/adventure/culturepack.ts; keep both in step.
// Every field is optional and omitted when empty, so old world files load unchanged. Placeholders left in
// text for the engine to fill: {name} {title} {realm} {town} {culture} {faith} {deity} {time} {kin} {item}.
// Wanderer code that consumes a pack guards for nil (cpOf returns nil for a culture without one).

const cpVersion = 1

// cpActs is the vocabulary of acts a taboo may name; acts the engine cannot detect are flavour only.
var cpActs = []string{
	"draw_weapon_indoors", "refuse_hospitality", "refuse_gift", "insult_ancestors", "blaspheme",
	"work_on_holy_day", "speak_before_elder", "bare_head", "bare_arms", "public_anger", "stare",
	"eat_unclean", "eat_alone", "enter_unbidden", "turn_back", "break_oath", "haggle_sacred",
	"wear_forbidden_color", "touch_head", "left_hand", "whistle_night", "waste_food",
	"drink_to_excess", "shirk_duty",
}

type CulturePack struct {
	V          int             `json:"v"`
	Customs    *CustomsPack    `json:"customs,omitempty"`
	Voice      *VoicePack      `json:"voice,omitempty"`
	Literature *LiteraturePack `json:"literature,omitempty"`
	Material   *MaterialPack   `json:"material,omitempty"`
	Lang       *LangPack       `json:"lang,omitempty"`
}

// ---- customs ----

type Taboo struct {
	ID       string `json:"id"`
	Act      string `json:"act"` // one of cpActs
	Text     string `json:"text"`
	Severity int    `json:"severity"` // 1 frown, 2 insult, 3 blood or banishment
	Penalty  string `json:"penalty"`  // none fine scorn exile lash mutilation hang weregild ordeal enslave duel
	Rebuke   string `json:"rebuke"`
	Where    string `json:"where,omitempty"` // home hall temple market road table feast; empty = everywhere
}

type Etiquette struct {
	ID     string `json:"id"`
	Kind   string `json:"kind"` // greeting gesture gift food speech dress space time
	Text   string `json:"text"`
	Reward int    `json:"reward,omitempty"`
}

type Hospitality struct {
	GuestRight int      `json:"guestRight"` // days a guest is protected
	Offers     []string `json:"offers"`
	Duties     []string `json:"duties"`
	Breach     string   `json:"breach"`
}

type Rite struct {
	ID    string   `json:"id"` // birth naming comingofage betrothal wedding funeral oath harvest departure homecoming initiation
	Name  string   `json:"name"`
	Steps []string `json:"steps"`
	Roles []string `json:"roles,omitempty"`
	Gifts []string `json:"gifts,omitempty"`
	Dress string   `json:"dress,omitempty"`
	Song  string   `json:"song,omitempty"`
}

type FeastRitual struct {
	Feast     string   `json:"feast"` // matches FeastData.Name
	Steps     []string `json:"steps"`
	Offerings []string `json:"offerings"`
	Dress     string   `json:"dress,omitempty"`
	Food      string   `json:"food,omitempty"`
	Song      string   `json:"song,omitempty"`
	Taboo     string   `json:"taboo,omitempty"`
	Custom    string   `json:"custom,omitempty"`
}

type LawEntry struct {
	Crime  string `json:"crime"` // theft murder adultery blasphemy debt trespass assault treason forgery smuggling
	Punish string `json:"punish"`
	Note   string `json:"note"`
}

type LawPack struct {
	Judge   string     `json:"judge"`
	Ordeal  string     `json:"ordeal,omitempty"`
	Entries []LawEntry `json:"entries"`
}

type Ideals struct {
	Death  string `json:"death,omitempty"`
	Host   string `json:"host,omitempty"`
	Name   string `json:"name,omitempty"`
	Wealth string `json:"wealth,omitempty"`
}

type CustomsPack struct {
	Taboos      []Taboo       `json:"taboos,omitempty"`
	Etiquette   []Etiquette   `json:"etiquette,omitempty"`
	Hospitality *Hospitality  `json:"hospitality,omitempty"`
	Rites       []Rite        `json:"rites,omitempty"`
	Festivals   []FeastRitual `json:"festivals,omitempty"`
	Law         *LawPack      `json:"law,omitempty"`
	Ideals      *Ideals       `json:"ideals,omitempty"`
}

// ---- voice ----

type Idiom struct {
	Say     string `json:"say"`
	Meaning string `json:"meaning"`
}

type AddressForms struct {
	Formal   string `json:"formal"`
	Familiar string `json:"familiar"`
}

type VoicePack struct {
	Register   string              `json:"register"` // formal blunt flowery terse earthy courtly
	Greet      []string            `json:"greet,omitempty"`
	Farewell   []string            `json:"farewell,omitempty"`
	Thanks     []string            `json:"thanks,omitempty"`
	Apology    []string            `json:"apology,omitempty"`
	Agree      []string            `json:"agree,omitempty"`
	Refuse     []string            `json:"refuse,omitempty"`
	Praise     []string            `json:"praise,omitempty"`
	Insults    []string            `json:"insults,omitempty"`
	Curses     []string            `json:"curses,omitempty"`
	Oaths      []string            `json:"oaths,omitempty"`
	Fillers    []string            `json:"fillers,omitempty"`
	Proverbs   []string            `json:"proverbs,omitempty"`
	Idioms     []Idiom             `json:"idioms,omitempty"`
	Honorifics map[string]string   `json:"honorifics,omitempty"` // low common high noble ruler priest elder stranger
	Address    *AddressForms       `json:"address,omitempty"`
	Humour     string              `json:"humour,omitempty"`
	Barks      map[string][]string `json:"barks,omitempty"` // market guard tavern temple road night rain crowd smith farm harbor school
	Gossip     []string            `json:"gossip,omitempty"`
}

// PersonVoice is a notable's own speech habits, layered on the culture's voice.
type PersonVoice struct {
	Tone    int      `json:"tone,omitempty"` // -1 plainer, 0 as culture, 1 grander
	Tics    []string `json:"tics,omitempty"`
	Oath    string   `json:"oath,omitempty"`
	Temper  string   `json:"temper,omitempty"` // calm proud anxious wry gruff warm cold eager
	Beliefs []string `json:"beliefs,omitempty"`
	Quirks  []string `json:"quirks,omitempty"`
	Topic   string   `json:"topic,omitempty"`
}

// ---- faith ----

type Liturgy struct {
	Name  string   `json:"name"`
	Steps []string `json:"steps"`
}

type Scripture struct {
	Title    string   `json:"title"`
	Excerpts []string `json:"excerpts"`
}

type Saint struct {
	Name   string `json:"name"`
	Legend string `json:"legend"`
	Feast  string `json:"feast,omitempty"`
}

type Heresy struct {
	Name  string `json:"name"`
	Claim string `json:"claim"`
}

type HolyDay struct {
	Name  string `json:"name"`
	Month int    `json:"month"`
	Rite  string `json:"rite"`
}

type ClergyVoice struct {
	GreetFaithful string `json:"greetFaithful"`
	GreetStranger string `json:"greetStranger"`
	Title         string `json:"title,omitempty"`
}

type FaithPack struct {
	Prayers     []string     `json:"prayers,omitempty"`
	Liturgy     *Liturgy     `json:"liturgy,omitempty"`
	Scripture   *Scripture   `json:"scripture,omitempty"`
	Saints      []Saint      `json:"saints,omitempty"`
	Heresies    []Heresy     `json:"heresies,omitempty"`
	Afterlife   string       `json:"afterlife,omitempty"`
	HolyDays    []HolyDay    `json:"holyDays,omitempty"`
	Blessings   []string     `json:"blessings,omitempty"`
	Sins        []string     `json:"sins,omitempty"`
	Clergy      *ClergyVoice `json:"clergy,omitempty"`
	Observances []string     `json:"observances,omitempty"`
}

// ---- literature ----

type PoeticForm struct {
	Name    string   `json:"name"`
	Rule    string   `json:"rule"`
	Example []string `json:"example"`
}

type Work struct {
	ID      string   `json:"id"`
	Form    string   `json:"form"` // epic ballad history sermon satire tale lament hymn proverb love dispatch travelogue riddle
	Title   string   `json:"title"`
	Author  string   `json:"author"`
	Bias    string   `json:"bias"` // propagandist pious cynical partisan romantic neutral
	Year    int      `json:"year"`
	Event   *int     `json:"event,omitempty"` // index into WorldData.Events, nil = a legend
	Lines   []string `json:"lines"`
	Reveals string   `json:"reveals,omitempty"`
}

type LiteraturePack struct {
	Forms []PoeticForm `json:"forms,omitempty"`
	Works []Work       `json:"works,omitempty"`
	Taste string       `json:"taste,omitempty"`
}

// ---- material ----

type CpDish struct {
	Name  string   `json:"name"`
	With  []string `json:"with"`
	Tag   string   `json:"tag"` // a cooking tag: hearty spry thick mending keen ember rime deep lasting venom
	Note  string   `json:"note,omitempty"`
	Feast string   `json:"feast,omitempty"`
}

type Cuisine struct {
	Staples []string `json:"staples"`
	Dishes  []CpDish `json:"dishes"`
	Drinks  []string `json:"drinks"`
	Meals   []string `json:"meals,omitempty"`
	Unclean []string `json:"unclean,omitempty"`
}

type Craft struct {
	Name  string   `json:"name"`
	Goods []string `json:"goods"`
	Mark  string   `json:"mark,omitempty"`
}

type TradeGood struct {
	Name  string `json:"name"`
	Kind  string `json:"kind"`
	Value int    `json:"value"`
}

type BuildingStyle struct {
	Materials []string `json:"materials"`
	House     string   `json:"house"`
	Hall      string   `json:"hall"`
	Temple    string   `json:"temple"`
	Gate      string   `json:"gate,omitempty"`
	Roof      string   `json:"roof,omitempty"`
	Color     string   `json:"color,omitempty"`
	Ornament  string   `json:"ornament,omitempty"`
}

type MusicStyle struct {
	Instruments []string `json:"instruments"`
	Forms       []string `json:"forms"`
	Scale       string   `json:"scale,omitempty"`
}

type GameRule struct {
	Name string `json:"name"`
	Rule string `json:"rule"`
}

type MaterialPack struct {
	Cuisine    *Cuisine          `json:"cuisine,omitempty"`
	Crafts     []Craft           `json:"crafts,omitempty"`
	TradeGoods []TradeGood       `json:"tradeGoods,omitempty"`
	Dress      map[string]string `json:"dress,omitempty"` // low common high ritual mourning wedding war
	Buildings  *BuildingStyle    `json:"buildings,omitempty"`
	Music      *MusicStyle       `json:"music,omitempty"`
	Games      []GameRule        `json:"games,omitempty"`
	Land       string            `json:"land,omitempty"`
}

// ---- language ----

type Phonology struct {
	Consonants string `json:"consonants"`
	Vowels     string `json:"vowels"`
	Shape      string `json:"shape"`
}

type Phrase struct {
	Say   string `json:"say"`
	Gloss string `json:"gloss"`
	Use   string `json:"use"` // greet toast thanks farewell curse oath blessing
}

type LangPack struct {
	Lexicon   [][2]string `json:"lexicon,omitempty"` // [word, gloss]
	Phonology *Phonology  `json:"phonology,omitempty"`
	Phrases   []Phrase    `json:"phrases,omitempty"`
	Numerals  []string    `json:"numerals,omitempty"`
}

// cpOf returns the culture's pack, or nil when the world file has none.
func (w *WorldData) cpOf(culture int) *CulturePack {
	if w == nil || culture < 0 || culture >= len(w.Cultures) {
		return nil
	}
	return w.Cultures[culture].Pack
}

// ---- shared helpers for everything that speaks pack text ----

// cpFill replaces the schema placeholders ({name} {title} {realm} {town} {culture} {faith} {deity} {time}
// {kin} {item}) with vars; placeholders without a value are left out cleanly (the surrounding space too).
func cpFill(s string, vars map[string]string) string {
	for _, k := range []string{"name", "title", "realm", "town", "culture", "faith", "deity", "time", "kin", "item"} {
		ph := "{" + k + "}"
		if !strings.Contains(s, ph) {
			continue
		}
		v := vars[k]
		if v == "" {
			s = strings.ReplaceAll(s, " "+ph, "")
			s = strings.ReplaceAll(s, ph+" ", "")
		}
		s = strings.ReplaceAll(s, ph, v)
	}
	return s
}

// cpChoose picks one entry of list by a stable hash of the world seed and the key parts (never g.R), so the
// same speaker says the same thing in the same situation, and varies when a part (the day, a topic) changes.
func (w *WorldData) cpChoose(list []string, parts ...string) string {
	if len(list) == 0 {
		return ""
	}
	return list[int(HashStr(w.Seed+":cp:"+strings.Join(parts, ":"))%uint64(len(list)))]
}

// cpIdx is cpChoose for an index (n options).
func (w *WorldData) cpIdx(n int, parts ...string) int {
	if n <= 0 {
		return 0
	}
	return int(HashStr(w.Seed+":cp:"+strings.Join(parts, ":")) % uint64(n))
}
