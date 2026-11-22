package engine

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// Cutscene panels (prefix ct2): short framed text panels, 2 to 5 pages each, for births, weddings and
// funerals of people you know, a first arrival in a realm or city, level milestones, the death of a
// companion, rift crossings and the ending of the great threat. Mode `ctn`: Enter, Space or Esc turns the
// page (Left goes back), after the last page the game goes on.
//
// Nothing here edits other files: scenes are fired from an enter hook (arrivals, rifts), a kill hook
// (companions) and a turn hook that polls state (levels, the life simulation once a day, endings). A scene
// that fires while something else is on screen, or while foes are in sight, waits in a queue. Text comes
// from templates filled with real names, places and years; speech goes through the culture voice (dvSay)
// when there is one. The portrait band is an ASCII frame with an emblem and a speaker line (the shell's
// sprite portrait is tied to the talk window and is not used). Pref "cutscenes" (unset = on); the palette
// entry "Replay last scene" shows the last one again.
//
// State g.ext("ct2") (global; first-time keys carry the world seed). Never draws g.R.

type ct2Page struct {
	Who  string `json:"w,omitempty"` // speaker name, "" = the narrator
	Role string `json:"r,omitempty"`
	Text string `json:"t"`
}

type ct2Scene struct {
	Kind  string     `json:"k"`
	Title string     `json:"ti"`
	Pages []ct2Page  `json:"p"`
	Day   int        `json:"d"`
}

type ct2Snap struct {
	D bool     `json:"d,omitempty"`
	S string   `json:"s,omitempty"`
	K []string `json:"k,omitempty"`
}

type ct2State struct {
	Last     *ct2Scene            `json:"l,omitempty"`
	Cur      *ct2Scene            `json:"c,omitempty"`
	Page     int                  `json:"pg,omitempty"`
	Queue    []*ct2Scene          `json:"q,omitempty"`
	Seen     map[string]bool      `json:"s,omitempty"`
	Snap     map[string]*ct2Snap  `json:"sn,omitempty"`
	Lv       int                  `json:"lv,omitempty"`
	LvInit   bool                 `json:"li,omitempty"`
	Boot     bool                 `json:"b,omitempty"`
	Seed     string               `json:"sd,omitempty"`
	Planet   string               `json:"pl,omitempty"`
	Day      int                  `json:"dy,omitempty"`
	Count    int                  `json:"n,omitempty"`
	LastBark int                  `json:"-"`
}

func ct2St(g *Game) *ct2State {
	s := g.ext("ct2", func() any { return &ct2State{} }).(*ct2State)
	if s.Seen == nil {
		s.Seen = map[string]bool{}
	}
	if s.Snap == nil {
		s.Snap = map[string]*ct2Snap{}
	}
	return s
}

func init() {
	enterHooks = append(enterHooks, ct2Enter)
	turnHooks = append(turnHooks, ct2Turn)
	killHooks = append(killHooks, ct2Kill)
	modeKeys["ctn"] = ct2Keys
	modeDraws["ctn"] = func(g *Game) { g.ct2Draw() }
	paEntries = append(paEntries, paEntry{Name: "Replay last scene", Keywords: "cutscene story panel recap birth wedding funeral arrival", Cat: "Screen",
		Func: func(g *Game) {
			st := ct2St(g)
			if st.Last == nil || len(st.Last.Pages) == 0 {
				g.Msg("{{K|No scene to replay yet.}}")
				return
			}
			ct2Open(g, st.Last)
		}})
}

// ---------------------------------------------------------------- showing scenes

func ct2Safe(g *Game) bool {
	if g.P == nil || g.Dead || g.Mode != "main" || g.Cur == nil {
		return false
	}
	for _, a := range g.Cur.Actors {
		if a != g.P && a.HP > 0 && g.Hostile(a) && g.visible(a.X, a.Y) {
			return false
		}
	}
	return true
}

func ct2Open(g *Game, sc *ct2Scene) {
	st := ct2St(g)
	st.Cur, st.Last, st.Page = sc, sc, 0
	g.Mode, g.Sel = "ctn", 0
}

// ct2Push shows a scene now when the screen is free, otherwise queues it (never more than 6 waiting).
func ct2Push(g *Game, sc *ct2Scene) {
	if sc == nil || len(sc.Pages) == 0 || !g.Pref("cutscenes") {
		return
	}
	st := ct2St(g)
	sc.Day = lvDay(g)
	st.Count++
	if st.Cur == nil && ct2Safe(g) {
		ct2Open(g, sc)
		return
	}
	if len(st.Queue) < 6 {
		st.Queue = append(st.Queue, sc)
	}
}

func ct2Flush(g *Game) {
	st := ct2St(g)
	if st.Cur != nil || len(st.Queue) == 0 || !ct2Safe(g) {
		return
	}
	sc := st.Queue[0]
	st.Queue = st.Queue[1:]
	ct2Open(g, sc)
}

func ct2Close(g *Game) {
	st := ct2St(g)
	st.Cur, st.Page = nil, 0
	g.Mode = "main"
}

func ct2Keys(g *Game, k string) {
	st := ct2St(g)
	sc := st.Cur
	if sc == nil || len(sc.Pages) == 0 {
		ct2Close(g)
		return
	}
	switch k {
	case "Enter", " ", "Escape", "ArrowRight", "N6":
		st.Page++
		if st.Page >= len(sc.Pages) {
			ct2Close(g)
		}
	case "ArrowLeft", "N4", "Backspace":
		if st.Page > 0 {
			st.Page--
		}
	}
}

var ct2Emblems = map[string][3]string{
	"wedding":   {"  (o)(o)  ", "   \\  /   ", "   '--'   "},
	"birth":     {"   .-.    ", "  (o.o)   ", "   \\_/    "},
	"funeral":   {"    +     ", "  --+--   ", "    |     "},
	"arrival":   {" /\\  /\\   ", "/  \\/  \\  ", "|  []  |  "},
	"level":     {"    /\\    ", "   /**\\   ", "  /____\\  "},
	"companion": {"  .---.   ", " | x x |  ", "  \\_-_/   "},
	"rift":      {"  .-\"\"-.  ", " ( @  @ ) ", "  '-..-'  "},
	"ending":    {"  \\ | /   ", " -- O --  ", "  / | \\   "},
}

var ct2Colors = map[string]byte{"wedding": CMagenta, "birth": CGreen, "funeral": CDarkGray, "arrival": CCyan, "level": CYellow,
	"companion": CRed, "rift": CMagenta, "ending": COrange}

func (g *Game) ct2Draw() {
	st := ct2St(g)
	sc := st.Cur
	if sc == nil || len(sc.Pages) == 0 {
		return
	}
	s := g.sc
	st.Page = clampi(st.Page, 0, len(sc.Pages)-1)
	pg := sc.Pages[st.Page]
	x, y, w, h := g.panel("Scene")
	col := ct2Colors[sc.Kind]
	if col == 0 {
		col = CCyan
	}
	s.TextB(x, y, "{{M|"+sc.Title+"}}", CGray, CBgDark, x+w)
	// portrait band: a frame with the emblem, the speaker beside it
	s.Box(x, y+2, 14, 7, col, "")
	em := ct2Emblems[sc.Kind]
	for i := 0; i < 3; i++ {
		s.TextB(x+2, y+4+i, em[i], col, CBgDark, x+13)
	}
	nx := x + 17
	if pg.Who != "" {
		s.TextB(nx, y+3, "{{W|"+pg.Who+"}}", CGray, CBgDark, x+w)
		if pg.Role != "" {
			s.TextB(nx, y+4, "{{c|"+pg.Role+"}}", CGray, CBgDark, x+w)
		}
		s.TextB(nx, y+6, "{{K|speaks}}", CGray, CBgDark, x+w)
	} else {
		s.TextB(nx, y+3, "{{W|The chronicle}}", CGray, CBgDark, x+w)
		s.TextB(nx, y+6, "{{K|tells}}", CGray, CBgDark, x+w)
	}
	s.TextB(nx, y+7, fmt.Sprintf("{{K|page %d of %d}}", st.Page+1, len(sc.Pages)), CGray, CBgDark, x+w)
	yy := y + 10
	for _, l := range Wrap(pg.Text, w-2) {
		if yy >= y+h-2 {
			break
		}
		s.TextB(x, yy, l, CGray, CBgDark, x+w)
		yy++
	}
	foot := "{{K|Enter or Esc to go on, Left to go back.}}"
	if st.Page == len(sc.Pages)-1 {
		foot = "{{K|Enter or Esc to close.}}"
	}
	s.TextB(x, y+h-1, foot, CGray, CBgDark, x+w)
}

// ---------------------------------------------------------------- text helpers

func ct2Pick(g *Game, list []string, parts ...any) string {
	if len(list) == 0 {
		return ""
	}
	return list[int(HashStr(g.W.Seed+":ct2:"+fmt.Sprint(parts...))%uint64(len(list)))]
}

func ct2Fill(s string, kv ...string) string {
	return strings.NewReplacer(kv...).Replace(s)
}

func ct2Town(g *Game, id int) string {
	if bd := g.W.burgByID[id]; bd != nil {
		return bd.Name
	}
	return "the town"
}

func ct2Say(s string) string {
	if s == "" {
		return ""
	}
	return "\"" + s + "\""
}

// ct2Voice: one line in the voice of a simulated person (their culture, notable voice, temper), "" when none.
func ct2Voice(g *Game, p *lvPerson, kind string) string {
	if p == nil {
		return ""
	}
	a := &Actor{ID: 700000 + int(HashStr(p.Key)%90000), Name: p.Name, Proper: true, Tpl: "npc", AI: "follower", Role: p.Role,
		Species: p.Species, State: p.State, Religion: p.Faith}
	if strings.HasPrefix(p.Key, "s") {
		if n, err := strconv.Atoi(p.Key[1:]); err == nil {
			a.SimPer = n
		}
	}
	return g.dvSay(a, kind, nil)
}

func ct2Role(p *lvPerson) string {
	r := titleCase(nonEmpty(p.Job, p.Role))
	return r
}

func ct2Rite(g *Game, culture int, ids ...string) (string, string) {
	p := cgPack(g.W, culture)
	if p == nil || p.Customs == nil {
		return "", ""
	}
	for _, id := range ids {
		for _, r := range p.Customs.Rites {
			if r.ID == id && r.Name != "" {
				step := ""
				if len(r.Steps) > 0 {
					step = r.Steps[0]
				}
				return r.Name, step
			}
		}
	}
	return "", ""
}

func ct2BestMemory(p *lvPerson) *lvMemory {
	var best *lvMemory
	for i := range p.Mem {
		m := &p.Mem[i]
		if best == nil || abs(m.Weight) >= abs(best.Weight) {
			best = m
		}
	}
	return best
}

func ct2Stance(op int) string {
	switch {
	case op >= 60:
		return "a friend"
	case op >= 20:
		return "someone they liked"
	case op > -20:
		return "a passing stranger"
	case op > -60:
		return "someone they distrusted"
	}
	return "an enemy"
}

func ct2Age(g *Game, p *lvPerson) int { return p.lvAge(lvYear(g)) }

func ct2When(g *Game) string {
	return fmt.Sprintf("%s, Year %d", fwDate(g, lvDay(g)), lvYear(g))
}

// ---------------------------------------------------------------- scene builders

func ct2Wedding(g *Game, a, b *lvPerson) *ct2Scene {
	town := ct2Town(g, nonZero(a.Burg, a.Home))
	sc := &ct2Scene{Kind: "wedding", Title: "A wedding in " + town}
	intro := fmt.Sprintf("%s. %s keeps the day: %s and %s are wed.", ct2When(g), town, a.Name, b.Name)
	if p := cgPack(g.W, a.Culture); p != nil && p.Material != nil && p.Material.Dress["wedding"] != "" {
		intro += " The couple wear " + strings.TrimSuffix(strings.ToLower(p.Material.Dress["wedding"]), ".") + "."
	}
	sc.Pages = append(sc.Pages, ct2Page{Text: intro})
	l1 := ct2Voice(g, a, "oath")
	if l1 == "" {
		l1 = ct2Fill(ct2Pick(g, []string{"I take you, {B}, as my own, in this place and before these people.",
			"{B}, whatever the roads bring, I will walk them with you.", "Before everyone who knows us, {B}: I am yours."}, a.Key, b.Key), "{B}", givenOf(b.Name))
	}
	sc.Pages = append(sc.Pages, ct2Page{Who: a.Name, Role: ct2Role(a), Text: ct2Say(l1)})
	l2 := ct2Voice(g, b, "thanks")
	if l2 == "" {
		l2 = ct2Fill(ct2Pick(g, []string{"And I you, {A}, for every season we are given.", "Then it is settled, {A}. I could not ask for more."}, b.Key, a.Key), "{A}", givenOf(a.Name))
	}
	sc.Pages = append(sc.Pages, ct2Page{Who: b.Name, Role: ct2Role(b), Text: ct2Say(l2)})
	end := ""
	if rn, step := ct2Rite(g, a.Culture, "wedding", "betrothal"); rn != "" {
		end = fmt.Sprintf("In the manner of the %s: %s.", rn, strings.TrimSuffix(step, "."))
	}
	switch {
	case a.Opinion >= 20 || b.Opinion >= 20:
		end += " They thought of you when the guest list was made, and word of it finds you."
	case a.Met || b.Met:
		end += " You had spoken with them once. Word of the day reaches you."
	default:
		end += " Word of the day reaches you."
	}
	sc.Pages = append(sc.Pages, ct2Page{Text: strings.TrimSpace(end)})
	return sc
}

func ct2Birth(g *Game, parent, other, child *lvPerson) *ct2Scene {
	town := ct2Town(g, nonZero(parent.Burg, parent.Home))
	sc := &ct2Scene{Kind: "birth", Title: "A birth in " + town}
	who := parent.Name
	if other != nil {
		who = parent.Name + " and " + other.Name
	}
	sc.Pages = append(sc.Pages, ct2Page{Text: fmt.Sprintf("%s. In %s a child is born to %s. They are named %s.", ct2When(g), town, who, child.Name)})
	l := ct2Voice(g, parent, "thanks")
	if l == "" {
		l = ct2Fill(ct2Pick(g, []string{"We have named them {C}. May the years be kind.", "Small, loud and ours: {C} is here.", "{C}. Say it, so the house learns it."}, parent.Key, child.Key), "{C}", givenOf(child.Name))
	}
	sc.Pages = append(sc.Pages, ct2Page{Who: parent.Name, Role: ct2Role(parent), Text: ct2Say(l)})
	if rn, step := ct2Rite(g, parent.Culture, "birth", "naming"); rn != "" {
		sc.Pages = append(sc.Pages, ct2Page{Text: fmt.Sprintf("The %s is kept: %s.", rn, strings.TrimSuffix(step, "."))})
	}
	end := "The household will not be the same, and the street already knows it."
	if parent.Opinion >= 30 {
		end = parent.Name + " hopes you will pass this way again and see the child grow."
	}
	sc.Pages = append(sc.Pages, ct2Page{Text: end})
	return sc
}

func ct2Funeral(g *Game, p *lvPerson) *ct2Scene {
	town := ct2Town(g, nonZero(p.Burg, p.Home))
	sc := &ct2Scene{Kind: "funeral", Title: "The funeral of " + p.Name}
	job := strings.ToLower(nonEmpty(p.Job, p.Role))
	t := fmt.Sprintf("%s. %s, %s of %s, is dead at %d.", ct2When(g), p.Name, article(job), town, ct2Age(g, p))
	if p.Cause != "" {
		t += " " + titleFirst(p.Cause) + "."
	}
	sc.Pages = append(sc.Pages, ct2Page{Text: t})
	ls := lvSt(g)
	var mourner *lvPerson
	cands := append([]string{p.Spouse}, p.Kin...)
	cands = append(cands, p.Friends...)
	for _, k := range cands {
		if q := ls.People[k]; q != nil && q.Alive {
			mourner = q
			break
		}
	}
	if mourner != nil {
		l := ct2Voice(g, mourner, "farewell")
		if l == "" {
			l = ct2Fill(ct2Pick(g, []string{"{N} was in {T} for as long as I can remember. We will not see their like again.",
				"They never asked much of anyone. Let the ground be easy on {N}.", "I will keep their place at the table a while."}, p.Key), "{N}", givenOf(p.Name), "{T}", town)
		}
		sc.Pages = append(sc.Pages, ct2Page{Who: mourner.Name, Role: ct2Role(mourner), Text: ct2Say(l)})
	}
	if n := len(p.Life); n > 0 {
		lines := []string{}
		for i := maxi(0, n-2); i < n; i++ {
			lines = append(lines, fmt.Sprintf("Year %d: %s", p.Life[i].Y, strings.TrimSuffix(p.Life[i].Text, ".")+"."))
		}
		sc.Pages = append(sc.Pages, ct2Page{Text: "What is remembered of them. " + strings.Join(lines, " ")})
	}
	end := "You had met, once. It is enough to feel the gap."
	if m := ct2BestMemory(p); m != nil {
		end = fmt.Sprintf("They remembered you: %s They counted you %s.", strings.TrimSuffix(m.Text, ".")+".", ct2Stance(p.Opinion))
	}
	if strings.HasPrefix(p.Cause, "murdered by ") {
		end += " There is talk in the street of who did it."
	}
	sc.Pages = append(sc.Pages, ct2Page{Text: end})
	return sc
}

func article(s string) string {
	if s == "" {
		return "a person"
	}
	if strings.ContainsRune("aeiou", rune(s[0])) {
		return "an " + s
	}
	return "a " + s
}

func titleFirst(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func nonZero(a, b int) int {
	if a != 0 {
		return a
	}
	return b
}

var ct2GateFallback = []string{"State your business and keep your blade sheathed.", "Welcome, traveller. Mind the watch, and the prices.", "You are late for the market and early for trouble. Pass."}

func ct2Arrival(g *Game, zn *Zone, bd *BurgData, newRealm bool) *ct2Scene {
	realm := g.realmName(bd.State)
	kind := "town"
	switch {
	case bd.Capital:
		kind = "capital"
	case bd.Port:
		kind = "port"
	}
	sc := &ct2Scene{Kind: "arrival", Title: "Arrival: " + bd.Name}
	cu := ""
	if c := g.W.cultByID[bd.Culture]; c != nil {
		cu = ", in the lands of the " + c.Name + " people"
	}
	sc.Pages = append(sc.Pages, ct2Page{Text: fmt.Sprintf("%s. You come to %s, a %s of %s%s. The road ends here for now, and the street begins.", ct2When(g), bd.Name, kind, realm, cu)})
	if newRealm {
		if st := g.W.stateByID[bd.State]; st != nil && bd.State > 0 {
			t := fmt.Sprintf("You have crossed into %s.", nonEmpty(st.FullName, st.Name))
			if st.Ruler != "" {
				t += " " + st.Ruler + " holds the seat"
				if st.Form != "" {
					t += ", under the form of a " + strings.ToLower(st.Form)
				}
				t += "."
			}
			if rel := g.W.relByID[st.Religion]; rel != nil && rel.Name != "" {
				t += " The faith of the realm is " + rel.Name + "."
			}
			if h := g.W.stateByID[g.HomeState]; h != nil && g.HomeState > 0 && g.HomeState != bd.State {
				for _, wr := range h.Wars {
					if wr == bd.State {
						t += " Your own realm is at war with it, and some here will know the colour of your coat."
					}
				}
			}
			sc.Pages = append(sc.Pages, ct2Page{Text: t})
		}
	}
	gate := &Actor{ID: 800000 + bd.ID, Name: "the gate warden", Tpl: "npc", AI: "follower", Role: "guard", State: bd.State}
	l := g.dvSay(gate, "greet", nil)
	if l == "" {
		l = ct2Pick(g, ct2GateFallback, "gate", bd.ID)
	}
	sc.Pages = append(sc.Pages, ct2Page{Who: "Gate warden", Role: titleCase(realm), Text: ct2Say(l)})
	rep := g.Rep[fmt.Sprintf("state:%d", bd.State)]
	end := "Nobody here knows your name yet."
	switch {
	case g.uwBounty(bd.State) > 0:
		end = fmt.Sprintf("A bounty of %d %s waits on your head in this realm. Walk softly.", g.uwBounty(bd.State), g.Sk.Currency)
	case rep >= 100:
		end = "Your name has travelled ahead of you, and kindly."
	case rep <= -100:
		end = "Your name has travelled ahead of you, and not kindly."
	}
	sc.Pages = append(sc.Pages, ct2Page{Text: end})
	return sc
}

var ct2LevelInner = []string{
	"You are not the same wanderer who set out. The road has been a patient teacher.",
	"Each scar has a name now, and so does each debt. You carry both without much weight.",
	"There is a quiet in you that was not there before. Others notice it before you do.",
	"Somewhere a person tells your name to a child as a warning. Somewhere another as a promise.",
	"The world is wider than the map said, and you are a little less small in it.",
	"You stop to look back, and the horizon you left is already a story.",
}

func ct2Level(g *Game) *ct2Scene {
	sc := &ct2Scene{Kind: "level", Title: fmt.Sprintf("Level %d", g.P.Level)}
	sc.Pages = append(sc.Pages, ct2Page{Text: fmt.Sprintf("%s. %s, %s, reaches level %d. They call you %s.", ct2When(g), g.P.Name, strings.ToLower(g.Calling), g.P.Level, g.crEpithet())})
	sc.Pages = append(sc.Pages, ct2Page{Text: fmt.Sprintf("Behind you: %d dead by your hand, %d deeds written down, %d days on the road.", g.Kills, len(g.Deeds), lvDay(g))})
	sc.Pages = append(sc.Pages, ct2Page{Who: g.P.Name, Role: "You", Text: ct2Say(ct2Pick(g, ct2LevelInner, "lv", g.P.Level))})
	return sc
}

var ct2LastWords = []string{"Do not stop for me. Go on.", "I would do it again. Tell them that.", "It was a good road. It was a good road.", "Leave the shield. Take the rest."}

func ct2Companion(g *Game, d *Actor) *ct2Scene {
	sc := &ct2Scene{Kind: "companion", Title: "The fall of " + givenOf(d.Name)}
	t := fmt.Sprintf("%s. %s, who walked with you, is dead.", ct2When(g), d.Name)
	if c := d.Comp; c != nil && c.Since > 0 && g.Turn > c.Since {
		t += fmt.Sprintf(" They followed you %d days and shared %d kills.", (g.Turn-c.Since)/1440, c.Kills)
	}
	sc.Pages = append(sc.Pages, ct2Page{Text: t})
	l := g.dvSay(d, "farewell", nil)
	if l == "" {
		l = ct2Pick(g, ct2LastWords, "cw", d.ID)
	}
	sc.Pages = append(sc.Pages, ct2Page{Who: d.Name, Role: "Companion", Text: ct2Say(l)})
	if c := d.Comp; c != nil && (c.Town != "" || len(c.Kin) > 0) {
		t := ""
		if c.Town != "" {
			t = "They came from " + c.Town + "."
		}
		if len(c.Kin) > 0 {
			t += " Their " + c.Kin[0] + " is still waiting there for word."
		}
		sc.Pages = append(sc.Pages, ct2Page{Text: strings.TrimSpace(t)})
	}
	sc.Pages = append(sc.Pages, ct2Page{Text: "There is one fewer voice at the fire tonight. You do not say it aloud, and the quiet says it for you."})
	return sc
}

var ct2GenreAir = map[string]string{
	"fantasy": "old stone and woodsmoke", "mythic": "incense and storm", "steampunk": "coal and hot brass", "cyberpunk": "ozone and neon rain",
	"biopunk": "wet green rot", "atompunk": "chrome and static", "dieselpunk": "oil and cold iron", "solarpunk": "cut grass and warm glass",
	"wasteland": "dust and rust", "twilight": "ash and cold lamps",
}

func ct2Rift(g *Game, from string) *ct2Scene {
	to := g.unPlanetName()
	sc := &ct2Scene{Kind: "rift", Title: "Through the rift"}
	sc.Pages = append(sc.Pages, ct2Page{Text: fmt.Sprintf("%s. The gate closes behind you. %s is gone the way a room is gone when the door shuts.", ct2When(g), nonEmpty(from, "The world you knew"))})
	air := ct2GenreAir[g.Genre]
	if air == "" {
		air = "something you have no word for"
	}
	sc.Pages = append(sc.Pages, ct2Page{Text: fmt.Sprintf("Beyond it lies %s. The air tastes of %s.", nonEmpty(to, "another world"), air)})
	sc.Pages = append(sc.Pages, ct2Page{Text: "Nobody here knows your name. Whatever you were owed, or owed, stays on the other side. You begin again as a stranger."})
	return sc
}

var ct2EndLines = map[string][]string{
	"destroy": {"It is done. The thing that hung over the world is broken, and the silence after is enormous.", "People will argue about the cost for a generation. Tonight they weep, and laugh, and sleep."},
	"seal":    {"The seal holds. What cannot be killed has been shut away, and the land breathes out.", "Someone will have to keep watch. It will not be forgotten who made the lock."},
	"bargain": {"A bargain is struck, and the world goes on, at a price that comes due on a schedule.", "You will remember the handshake longer than the victory."},
	"ally":    {"The enemy stands beside you now, strange and watchful. Old fears do not die in a day, but they bend.", "The world is stranger than it was, and for the first time you are glad of it."},
	"abandon": {"You turn away, and the world pays for it in slow coin.", "Perhaps someone else will take up what you laid down. Perhaps not."},
}

func ct2Ending(g *Game, kind, name string) *ct2Scene {
	sc := &ct2Scene{Kind: "ending", Title: "The ending: " + nonEmpty(egEndNames[kind], "A parting")}
	sc.Pages = append(sc.Pages, ct2Page{Text: fmt.Sprintf("%s. The matter of %s is over.", ct2When(g), nonEmpty(name, "the great threat"))})
	for i, l := range ct2EndLines[kind] {
		sc.Pages = append(sc.Pages, ct2Page{Text: l})
		_ = i
	}
	sc.Pages = append(sc.Pages, ct2Page{Text: fmt.Sprintf("%s, called %s, is not finished. But this part of the story is, and it will be told for a long time.", g.P.Name, g.crEpithet())})
	return sc
}

// ---------------------------------------------------------------- hooks

func ct2Enter(g *Game, zn *Zone) {
	if g.W == nil || g.P == nil || zn == nil {
		return
	}
	st := ct2St(g)
	seed := g.W.Seed
	boot := !st.Boot
	st.Boot = true
	// rift crossing: the planet changed since the last arrival
	if st.Seed != "" && st.Seed != seed {
		from := st.Planet
		st.Seed, st.Planet = seed, g.unPlanetName()
		ct2Push(g, ct2Rift(g, from))
	} else {
		st.Seed, st.Planet = seed, g.unPlanetName()
	}
	if zn.Kind == "town" && zn.Burg > 0 {
		bd := g.W.burgByID[zn.Burg]
		if bd != nil {
			ck := fmt.Sprintf("%s|b%d", seed, bd.ID)
			rk := fmt.Sprintf("%s|s%d", seed, bd.State)
			newCity, newRealm := !st.Seen[ck], !st.Seen[rk]
			st.Seen[ck], st.Seen[rk] = true, true
			if !boot && (newCity || newRealm) && bd.State > 0 {
				ct2Push(g, ct2Arrival(g, zn, bd, newRealm))
			}
		}
	}
	ct2Flush(g)
}

func ct2Kill(g *Game, d, src *Actor) {
	if d == nil || d == g.P || d.Comp == nil || d.Comp.Summoned || d.Has("pqpet") || d.AI != "follower" {
		return
	}
	ct2Push(g, ct2Companion(g, d))
}

func ct2Turn(g *Game) {
	if g.P == nil || g.W == nil || g.Dead {
		return
	}
	st := ct2St(g)
	// level milestones every five levels
	if !st.LvInit {
		st.Lv, st.LvInit = g.P.Level, true
	} else if g.P.Level/5 > st.Lv/5 {
		st.Lv = g.P.Level
		ct2Push(g, ct2Level(g))
	} else if g.P.Level > st.Lv {
		st.Lv = g.P.Level
	}
	// the ending of the great threat, fresh endings only
	if eg := g.eg(); eg.Phase == 3 && eg.Ending != "" && egDay(g)-eg.EndDay <= 2 {
		k := "end|" + g.W.Seed + "|" + eg.Name
		if !st.Seen[k] {
			st.Seen[k] = true
			ct2Push(g, ct2Ending(g, eg.Ending, eg.Name))
		}
	}
	if d := lvDay(g); d != st.Day {
		st.Day = d
		ct2Scan(g)
	}
	ct2Flush(g)
}

func ct2Kids(g *Game, p *lvPerson) []string {
	var out []string
	for _, c := range lsChildren(g, p) {
		out = append(out, c.Key)
	}
	sort.Strings(out)
	if len(out) > 12 {
		out = out[:12]
	}
	return out
}

// ct2Scan compares the people you know with the last day's snapshot: deaths, weddings, births.
func ct2Scan(g *Game) {
	st := ct2St(g)
	ls := lvSt(g)
	seed := g.W.Seed
	var keys []string
	for k, p := range ls.People {
		if fwKnown(p) {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	made := 0
	yr := lvYear(g)
	for _, k := range keys {
		p := ls.People[k]
		sk := seed + "|" + k
		sn := st.Snap[sk]
		kids := ct2Kids(g, p)
		if sn == nil {
			st.Snap[sk] = &ct2Snap{D: !p.Alive, S: p.Spouse, K: kids}
			continue
		}
		if !sn.D && !p.Alive && made < 2 {
			made++
			ct2Push(g, ct2Funeral(g, p))
		}
		if sn.S == "" && p.Spouse != "" && made < 2 {
			if q := ls.People[p.Spouse]; q != nil && q.Alive && p.Alive {
				pair := []string{p.Key, q.Key}
				sort.Strings(pair)
				wk := fmt.Sprintf("w|%s|%s|%d", seed, strings.Join(pair, "+"), yr)
				if !st.Seen[wk] {
					st.Seen[wk] = true
					made++
					ct2Push(g, ct2Wedding(g, p, q))
				}
			}
		}
		for _, ck := range kids {
			if made >= 2 || containsStr(sn.K, ck) {
				continue
			}
			c := ls.People[ck]
			if c == nil || !c.Alive || c.Born < yr-1 {
				continue
			}
			bk := "b|" + seed + "|" + ck
			if st.Seen[bk] {
				continue
			}
			st.Seen[bk] = true
			made++
			var other *lvPerson
			if p.Spouse != "" {
				other = ls.People[p.Spouse]
			}
			ct2Push(g, ct2Birth(g, p, other, c))
		}
		sn.D, sn.S, sn.K = !p.Alive, p.Spouse, kids
	}
	// a pruned snapshot: forget people who no longer exist
	if len(st.Snap) > 600 {
		for k := range st.Snap {
			if strings.HasPrefix(k, seed+"|") && ls.People[strings.TrimPrefix(k, seed+"|")] == nil {
				delete(st.Snap, k)
			}
		}
	}
}
