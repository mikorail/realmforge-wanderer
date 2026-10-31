package engine

import (
	"fmt"
	"sort"
	"strings"
)

// ---- local calendars and feasts ----
//
// Realmforge exports per culture twelve month names (month 0 = the dark of the year, midwinter) and its
// feasts (name, kind, month on that 12-month scale, a text with {realm}/{town} placeholders); each realm
// exports its own feast days (the culture's feasts, month shifted per realm).
//
// Mapping onto the Wanderer's 112-day year (4 seasons × 28 days, day 0 = first day of spring, weather.go):
// a culture with n months splits the year into n months; month k (in year order from spring) covers days
// [ceil(k·112/n), ceil((k+1)·112/n)), so 12 months last 10/9/9 days and every 3 months make one season.
// Year-order month k is Realmforge month (k + n/6) mod n: spring starts at Realmforge month 2, the month of
// the spring feasts, and winter covers months 11, 0, 1.
//
// A feast falls on one day of its month: hash(world seed, culture, feast name) picks the day, so the
// schedule is fixed per world and nothing here draws from g.R. Feasts of a town come from its realm when
// the realm shares the town's culture, otherwise from the town's culture; old world files without any
// calendar data fall back to the default month names and three generic feasts.

// defaultMonths mirrors Realmforge MONTH_FAMILIES.default.
var defaultMonths = []string{"Frostmoot", "Thawmonth", "Seedtide", "Rainmoon", "Bloomtide", "Highsun", "Emberfall", "Harvest", "Leafturn", "Mistmoon", "Darkening", "Longnight"}

// feastText mirrors Realmforge FEAST_TEXT (used when a feast arrives without text).
var feastText = map[string]string{
	"spring":    "People of {realm} wake the year with song and fresh green boughs at {town}.",
	"harvest":   "Granaries of {realm} are opened at {town}; the whole town eats until dusk.",
	"dead":      "Families in {realm} set out lamps and food for their dead at {town}.",
	"midwinter": "The longest night in {realm} is spent by firelight in {town}, with stories and gifts.",
	"sea":       "Boats of {realm} are blessed and raced off {town}; sailors are toasted ashore.",
	"war":       "Warriors of {realm} parade and wrestle in the squares of {town} to honour the old victories.",
	"learning":  "Scholars and singers of {realm} compete in {town}; the best verse is carried through the streets.",
	"sun":       "Bonfires on the hills of {realm} greet the high sun above {town}.",
}

var fallbackFeasts = []FeastData{
	{Name: "Midwinter Feast", Kind: "midwinter", Month: 0},
	{Name: "Spring Festival", Kind: "spring", Month: 2},
	{Name: "Harvest Home", Kind: "harvest", Month: 8},
}

// per feast kind: the extra meal tag, the dish word, decoration glyph and colours, greeting
var feastStyle = map[string]struct {
	Tag, Dish string
	Glyph     rune
	Cols      []byte
	Greet     string
}{
	"spring":    {"spry", "honey cakes", '♣', []byte{CGreen, CMagenta, CYellow}, "Fresh boughs and a fresh year to you!"},
	"harvest":   {"hearty", "harvest loaf", '♠', []byte{CYellow, COrange, CBrown}, "Eat, eat! The granaries are open today."},
	"dead":      {"mending", "lamp-night bread", '¤', []byte{CYellow, COrange, CWhite}, "Light a lamp for your dead tonight, traveller."},
	"midwinter": {"rime", "spiced cakes", '¤', []byte{CYellow, CRed, CWhite}, "Warm yourself by our fires; the long night is ours."},
	"sea":       {"deep", "fish platter", '¶', []byte{CBlue, CCyan, CWhite}, "Did you see the boats race? Drink to the sailors!"},
	"war":       {"thick", "victory roast", '♦', []byte{CRed, CYellow, CWhite}, "Watch the wrestling! Our champions honour the old victories."},
	"learning":  {"keen", "scholar's sweets", '♪', []byte{CCyan, CMagenta, CYellow}, "Stay for the verses; the best of them is carried through the streets tonight."},
	"sun":       {"ember", "bonfire skewers", '▲', []byte{COrange, CYellow, CRed}, "The high sun is with us! Come to the bonfires."},
}

func feastStyleOf(kind string) (string, string, rune, []byte, string) {
	st, ok := feastStyle[kind]
	if !ok {
		st = feastStyle["harvest"]
	}
	return st.Tag, st.Dish, st.Glyph, st.Cols, st.Greet
}

const journalTabs = 5 // Quests, Secrets & Places, Chronicle, Cookbook, Calendar

// FeastLog is a feast you joined (the journal's Calendar tab).
type FeastLog struct {
	Name string `json:"n"`
	Town string `json:"t"`
	Year int    `json:"y"`
	Day  int    `json:"d"` // game day
	Text string `json:"x"`
}

// CalDate is a day of a culture's calendar.
type CalDate struct {
	Year  int
	Ord   int // month in year order from spring (0..n-1)
	Day   int // 1-based day of the month
	Len   int // days in the month
	Name  string
	DoY   int // day of the 112-day year
	Total int // months in the year
}

func monthsOf(cu *CultureData) []string {
	if cu != nil && len(cu.Months) > 0 {
		return cu.Months
	}
	return defaultMonths
}

func monthStart(k, n int) int { return (k*YearDays + n - 1) / n }

// monthOrdOf is the year-order month of a day of the year.
func monthOrdOf(doy, n int) int {
	k := doy * n / YearDays
	for k+1 < n && monthStart(k+1, n) <= doy {
		k++
	}
	for k > 0 && monthStart(k, n) > doy {
		k--
	}
	return k
}

// rfMonth: Realmforge month index (0 = midwinter) of year-order month k; ordOfRF is its inverse.
func rfMonth(k, n int) int { return (k + n/6) % n }
func ordOfRF(m, n int) int { return ((m-n/6)%n + n) % n }

// DateOf is the calendar date of a game day in a culture's months.
func (w *WorldData) DateOf(cu *CultureData, day int) CalDate {
	ms := monthsOf(cu)
	n := len(ms)
	if day < 0 {
		day = 0
	}
	doy := day % YearDays
	k := monthOrdOf(doy, n)
	st := monthStart(k, n)
	return CalDate{Year: w.Year + day/YearDays, Ord: k, Day: doy - st + 1, Len: monthStart(k+1, n) - st, Name: ms[rfMonth(k, n)], DoY: doy, Total: n}
}

func ordinal(n int) string {
	suf := "th"
	if n%100 < 11 || n%100 > 13 {
		switch n % 10 {
		case 1:
			suf = "st"
		case 2:
			suf = "nd"
		case 3:
			suf = "rd"
		}
	}
	return fmt.Sprintf("%d%s", n, suf)
}

// calendarCulture: the culture whose calendar the side panel shows (the town's, the land's, else home's).
func (g *Game) calendarCulture() *CultureData {
	if b := g.feastTown(); b != nil {
		if cu := g.W.cultByID[b.Culture]; cu != nil {
			return cu
		}
	}
	if cu := g.localCulture(); cu != nil && cu.ID > 0 {
		return cu
	}
	if s := g.W.stateByID[g.HomeState]; s != nil {
		return g.W.cultByID[s.Culture]
	}
	return nil
}

// dateLine is the side-panel date: "7th of Seedtide, Y221".
func (g *Game) dateLine() string {
	d := g.W.DateOf(g.calendarCulture(), g.Day())
	return fmt.Sprintf("{{c|%s of %s}} {{K|Y%d}}", ordinal(d.Day), d.Name, d.Year)
}

// feastTown is the living town whose zone you stand in.
func (g *Game) feastTown() *BurgData {
	if g.Cur == nil || g.OnWorld || g.Cur.Kind != "town" || g.Cur.Burg <= 0 {
		return nil
	}
	b := g.W.burgByID[g.Cur.Burg]
	if b == nil || !b.Alive {
		return nil
	}
	return b
}

// FeastsOf: the feast days kept in a town.
func (w *WorldData) FeastsOf(b *BurgData) []FeastData {
	if b == nil {
		return nil
	}
	if s := w.stateByID[b.State]; s != nil && s.Culture == b.Culture && len(s.Feasts) > 0 {
		return s.Feasts
	}
	if cu := w.cultByID[b.Culture]; cu != nil && len(cu.Feasts) > 0 {
		return cu.Feasts
	}
	return fallbackFeasts
}

// FeastDoY: the day of the year a feast falls on in a culture's calendar.
func (w *WorldData) FeastDoY(f FeastData, cu *CultureData) int {
	n := len(monthsOf(cu))
	m := ((f.Month % 12) + 12) % 12 * n / 12
	k := ordOfRF(m, n)
	st := monthStart(k, n)
	ln := monthStart(k+1, n) - st
	cid := 0
	if cu != nil {
		cid = cu.ID
	}
	return st + int(HashStr(fmt.Sprintf("%s:feast:%d:%s", w.Seed, cid, f.Name))%uint64(ln))
}

// FeastOn: the feast a town keeps on a game day, if any.
func (w *WorldData) FeastOn(b *BurgData, day int) *FeastData {
	if b == nil {
		return nil
	}
	cu := w.cultByID[b.Culture]
	fs := w.FeastsOf(b)
	for i := range fs {
		if w.FeastDoY(fs[i], cu) == day%YearDays {
			return &fs[i]
		}
	}
	return nil
}

// nextFeast: the town's next feast after today and how many days away it is.
func (w *WorldData) nextFeast(b *BurgData, day int) (*FeastData, int) {
	cu := w.cultByID[b.Culture]
	fs := w.FeastsOf(b)
	var best *FeastData
	bd := YearDays + 1
	for i := range fs {
		d := floorMod(w.FeastDoY(fs[i], cu)-day%YearDays, YearDays)
		if d > 0 && d < bd {
			best, bd = &fs[i], d
		}
	}
	return best, bd
}

func (g *Game) feastToday() (*BurgData, *FeastData) {
	b := g.feastTown()
	if b == nil {
		return nil, nil
	}
	return b, g.W.FeastOn(b, g.Day())
}

// fillFeast replaces {realm}/{town} in a feast or festival line.
func (w *WorldData) fillFeast(s string, b *BurgData) string {
	realm := "the town"
	if st := w.stateByID[b.State]; st != nil && st.ID > 0 {
		realm = st.Name
	}
	return strings.NewReplacer("{realm}", realm, "{town}", b.Name).Replace(s)
}

func (w *WorldData) feastText(f *FeastData, b *BurgData) string {
	t := f.Text
	if t == "" {
		t = feastText[f.Kind]
	}
	return w.fillFeast(t, b)
}

func feastKey(b *BurgData, f *FeastData, year int) string {
	return fmt.Sprintf("%d:%s:%d", b.ID, f.Name, year)
}

// feastAnnounce: told once a day per town, on arrival and at dawn.
func (g *Game) feastAnnounce() {
	b, f := g.feastToday()
	if f == nil {
		return
	}
	note := fmt.Sprintf("%d:%d", b.ID, g.Day())
	if g.FeastNote == note {
		return
	}
	g.FeastNote = note
	g.Msg(fmt.Sprintf("{{W|Today is the %s in %s!}} %s", f.Name, b.Name, g.W.feastText(f, b)))
}

// calendarTick runs every turn: the dawn announcement.
func (g *Game) calendarTick() {
	if (g.Turn+startMinutes)%DayLen == 6*60 {
		g.feastAnnounce()
	}
}

// ---- talk ----

func (g *Game) feastGreeting(a *Actor) string {
	_, f := g.feastToday()
	if f == nil || a.AI == "follower" || g.Hostile(a) || HashStr(fmt.Sprintf("%d:%d:greet", a.ID, g.Day()))%3 == 0 {
		return ""
	}
	_, _, _, _, greet := feastStyleOf(f.Kind)
	return fmt.Sprintf("\"%s\" {{K|(It is the %s.)}}", greet, f.Name)
}

// feastGossip: talk of today's feast or the next one coming within a fortnight.
func (g *Game) feastGossip(a *Actor) string {
	b := g.feastTown()
	if b == nil || HashStr(fmt.Sprintf("%d:%d:gossip", a.ID, g.Turn/600))%2 == 0 {
		return ""
	}
	cu := g.W.cultByID[b.Culture]
	if f := g.W.FeastOn(b, g.Day()); f != nil {
		s := fmt.Sprintf("\"Have you not heard? Today is the %s. %s\"", f.Name, g.W.feastText(f, b))
		if cu != nil && len(cu.Festivals) > 0 {
			s += "\n\n\"" + g.W.fillFeast(cu.Festivals[HashStr(f.Name)%uint64(len(cu.Festivals))], b) + "\""
		}
		return s
	}
	f, days := g.W.nextFeast(b, g.Day())
	if f == nil || days > 14 {
		return ""
	}
	d := g.W.DateOf(cu, g.Day()+days)
	when := fmt.Sprintf("in %d days", days)
	if days == 1 {
		when = "tomorrow"
	}
	return fmt.Sprintf("\"The %s is %s, on the %s of %s. Stay for it, if you can. %s\"", f.Name, when, ordinal(d.Day), d.Name, g.W.feastText(f, b))
}

// feastDish: the festive dish of a feast (the culture's signature dish plus the feast's own flavour).
func (g *Game) feastDish(b *BurgData, f *FeastData) (name, key string) {
	tag, word, _, _, _ := feastStyleOf(f.Kind)
	tags := []string{tag, tag}
	if cu := g.W.cultByID[b.Culture]; cu != nil {
		if _, k := localDish(cu); k != "" {
			tags = append(strings.Split(k, "+"), tag)
		}
	}
	sort.Strings(tags)
	return f.Name + " " + word, strings.Join(tags, "+")
}

func (g *Game) feastItem(b *BurgData, f *FeastData, n int) *Item {
	name, key := g.feastDish(b, f)
	return &Item{ID: g.newID(), Kind: "dish", Name: name, Effect: key, Glyph: '%', F: CYellow, Count: n, Value: 24, Power: 500, Weight: 1}
}

// canJoinFeast: a feast today, a friendly townsperson, and you have not joined this one this year.
func (g *Game) canJoinFeast(a *Actor) bool {
	b, f := g.feastToday()
	if f == nil || a == nil || a.AI == "follower" || g.Hostile(a) {
		return false
	}
	return !g.FeastSeen[feastKey(b, f, g.W.DateOf(nil, g.Day()).Year)]
}

// joinFeast: a free feast meal shared with your companions, a dish to take along, a little standing in
// the town's realm and a journal entry. Once per feast per town per year.
func (g *Game) joinFeast(a *Actor) {
	b, f := g.feastToday()
	if f == nil {
		return
	}
	year := g.W.DateOf(nil, g.Day()).Year
	key := feastKey(b, f, year)
	if g.FeastSeen[key] {
		return
	}
	if g.FeastSeen == nil {
		g.FeastSeen = map[string]bool{}
	}
	g.FeastSeen[key] = true
	text := g.W.feastText(f, b)
	cu := g.W.cultByID[b.Culture]
	if cu != nil && len(cu.Festivals) > 0 {
		text += " " + g.W.fillFeast(cu.Festivals[HashStr(fmt.Sprintf("%s:%d", f.Name, year))%uint64(len(cu.Festivals))], b)
	}
	g.Feasts = append(g.Feasts, FeastLog{Name: f.Name, Town: b.Name, Year: year, Day: g.Day(), Text: text})
	name, dk := g.feastDish(b, f)
	g.spendTurns(30)
	g.applyMeal(name, tagsOf(dk), DayLen, true)
	g.P.Inv = append(g.P.Inv, g.feastItem(b, f, 1))
	if b.State > 0 {
		g.AddRep(fmt.Sprintf("state:%d", b.State), 15, " for joining the "+f.Name)
	}
	g.gainXP(10)
	g.Talk, g.Mode = a, "talk"
	g.talkMenu(fmt.Sprintf("%s takes you by the arm into the %s. %s\n\nYou eat the {{W|%s}} with the townsfolk and are sent off with a portion wrapped for the road.\n\n{{W|(Recorded in your journal.)}}", g.the(a), f.Name, text, name))
}

// feastStock: on a feast day a town's merchants also sell the festive dish.
func (g *Game) feastStock(a *Actor) {
	b, f := g.feastToday()
	if f == nil || a.Role != "merchant" {
		return
	}
	flag := "feast:" + feastKey(b, f, g.W.DateOf(nil, g.Day()).Year)
	if a.Has(flag) {
		return
	}
	a.Set(flag)
	a.Inv = append(a.Inv, g.feastItem(b, f, 2))
}

// ---- decorations ----

// feastTile decks a town out on its feast day: garlands, lanterns or bonfires on open ground and bunting
// in the feast's colours along the walls. Purely cosmetic; f is nil on ordinary days.
func feastTile(f *FeastData, x, y int, t Terrain, gl rune, col byte, anim int) (rune, byte) {
	if f == nil {
		return gl, col
	}
	_, _, glyph, cols, _ := feastStyleOf(f.Kind)
	h := wxMix(uint64(x)*0x9e3779b97f4a7c15 ^ uint64(y)*0xc2b2ae3d27d4eb4f ^ HashStr(f.Name))
	switch t {
	case TWall:
		if (x+y)%3 == 0 {
			return gl, cols[(x/3+anim/4)%len(cols)]
		}
	case TDirt, TRoad, TCobble, TGrass:
		if h%29 == 0 {
			return glyph, cols[(int(h>>8)+anim/3)%len(cols)]
		}
	}
	return gl, col
}

// ---- journal ----

func (g *Game) calendarLines() []string {
	cu := g.calendarCulture()
	d := g.W.DateOf(cu, g.Day())
	who := "the common reckoning"
	if cu != nil && cu.Name != "" {
		who = "the " + cu.Name + " reckoning"
	}
	lines := []string{fmt.Sprintf("{{Y|Today}} the %s of %s, year %d (%s), by %s.", ordinal(d.Day), d.Name, d.Year, SeasonNames[g.Season()], who), ""}
	b := g.feastTown()
	var fs []FeastData
	if b != nil {
		fs = g.W.FeastsOf(b)
		lines = append(lines, "{{C|Months and feasts of "+b.Name+"}}")
	} else {
		if cu != nil && len(cu.Feasts) > 0 {
			fs = cu.Feasts
		}
		lines = append(lines, "{{C|Months}} {{K|(visit a town to learn its feast days)}}")
	}
	ms := monthsOf(cu)
	n := len(ms)
	for k := 0; k < n; k++ {
		st := monthStart(k, n)
		s := fmt.Sprintf("  %-18s {{K|%s, days %d-%d}}", ms[rfMonth(k, n)], SeasonNames[SeasonOfDay(st)], st+1, monthStart(k+1, n))
		for i := range fs {
			if fd := g.W.FeastDoY(fs[i], cu); fd >= st && fd < monthStart(k+1, n) {
				s += fmt.Sprintf(" {{W|%s}} {{K|on the %s}}", fs[i].Name, ordinal(fd-st+1))
			}
		}
		if k == d.Ord {
			s = "{{G|>}}" + s[1:]
		}
		lines = append(lines, s)
	}
	lines = append(lines, "", "{{C|Feasts you have joined}}")
	if len(g.Feasts) == 0 {
		lines = append(lines, "{{K|None yet. On a feast day, ask townsfolk to let you join.}}")
	}
	for i := len(g.Feasts) - 1; i >= 0; i-- {
		fl := g.Feasts[i]
		lines = append(lines, Wrap(fmt.Sprintf("{{W|%s}} in %s {{K|(Y%d)}}: %s", fl.Name, fl.Town, fl.Year, fl.Text), SW-6)...)
	}
	return lines
}
