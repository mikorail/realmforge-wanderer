package engine

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"unicode"
)

// Dialogue voice engine (dv): every NPC speaks in their own voice. dvVoice resolves who is speaking
// (the culture's VoicePack, the notable's PersonVoice on top, mood and opinion, role, caste), dvSay
// produces a line of a given kind, dvTalkTag a look-text hint, dvQuip an occasional one-liner.
// Text choice never draws g.R: it is hashed from actor id + kind + day (cpChoose).

type dvSpeaker struct {
	A        *Actor
	Culture  *CultureData
	Pack     *VoicePack   // the culture's pack voice, nil when the world has none
	Person   *PersonVoice // the notable's own habits, nil for townsfolk without one
	Register string       // formal blunt flowery terse earthy courtly
	Temper   string       // calm proud anxious wry gruff warm cold eager ("" = none)
	Tone     int          // -1 plainer, 0 as culture, 1 grander
	Rank     string       // honorific key of the speaker: low common high noble ruler priest elder
	RoleKey  string       // guard priest merchant elder child or ""
	Opinion  int          // of the player, -100..100
	Mood     int          // -100 grim .. 100 elated
	Met      bool
	Child    bool
}

// ---- resolving a speaker ----

func dvPersonData(g *Game, a *Actor, lp *lvPerson) *PersonData {
	id := a.SimPer
	if id == 0 && lp != nil && strings.HasPrefix(lp.Key, "s") {
		id, _ = strconv.Atoi(lp.Key[1:])
	}
	if id == 0 || g.W == nil {
		return nil
	}
	for i := range g.W.People {
		if g.W.People[i].ID == id {
			return &g.W.People[i]
		}
	}
	return nil
}

// dvCulture finds the culture an NPC speaks for: the person's own, else the town's, else the land's.
func dvCulture(g *Game, a *Actor, lp *lvPerson, pd *PersonData) *CultureData {
	if pd != nil {
		if c := g.W.cultByID[pd.Culture]; c != nil {
			return c
		}
	}
	if lp != nil {
		if c := g.W.cultByID[lp.Culture]; c != nil {
			return c
		}
	}
	if bd := g.localBurg(); bd != nil {
		if c := g.W.cultByID[bd.Culture]; c != nil {
			return c
		}
	}
	if a.State > 0 {
		if st := g.W.stateByID[a.State]; st != nil {
			if c := g.W.cultByID[st.Culture]; c != nil {
				return c
			}
		}
	}
	return g.lxCultureAt()
}

// dvRegisterFor derives a register for a culture without a pack from its sphere.
func dvRegisterFor(c *CultureData) string {
	if c == nil {
		return "earthy"
	}
	if r, ok := dvSphereRegister[c.Sphere]; ok {
		return r
	}
	return dvRegisters[int(HashStr("dv:reg:"+c.Sphere+":"+c.Name)%uint64(len(dvRegisters)))]
}

func dvRankKey(g *Game, a *Actor) string {
	t := strings.ToLower(a.Title + " " + a.Role)
	switch {
	case a.Role == "priest" || strings.Contains(t, "priest") || strings.Contains(t, "abbot") || strings.Contains(t, "imam"):
		return "priest"
	case strings.Contains(t, "king") || strings.Contains(t, "queen") || strings.Contains(t, "emperor") || strings.Contains(t, "ruler") || strings.Contains(t, "chief"):
		return "ruler"
	case a.Role == "elder":
		return "elder"
	case a.Caste != "":
		high, out := g.casteClass(a)
		switch {
		case out:
			return "low"
		case high && a.Rank >= 3:
			return "noble"
		case high:
			return "high"
		case a.Rank >= 2:
			return "high"
		}
	}
	if a.Role == "notable" {
		return "high"
	}
	return "common"
}

func dvRoleKey(a *Actor, child bool) string {
	switch {
	case child:
		return "child"
	case a.Role == "guard" || a.Role == "sergeant":
		return "guard"
	case a.Role == "priest":
		return "priest"
	case a.Role == "elder":
		return "elder"
	case isTrader(a):
		return "merchant"
	}
	return ""
}

// dvTemperFromTraits gives townsfolk a temper from their personality sliders.
func dvTemperFromTraits(p *lvPerson) string {
	switch {
	case p.Trait[4] >= 75:
		return "anxious"
	case p.Trait[3] <= 25:
		return "gruff"
	case p.Trait[3] >= 75:
		return "warm"
	case p.Trait[2] <= 20:
		return "cold"
	case p.Trait[2] >= 80:
		return "eager"
	}
	return ""
}

// dvVoice resolves a speaker; nil only for a nil actor or world.
func (g *Game) dvVoice(a *Actor) *dvSpeaker {
	if a == nil || g.W == nil {
		return nil
	}
	lp := lvPersonFor(g, a)
	pd := dvPersonData(g, a, lp)
	sp := &dvSpeaker{A: a, Rank: dvRankKey(g, a)}
	sp.Culture = dvCulture(g, a, lp, pd)
	if sp.Culture != nil {
		if pk := sp.Culture.Pack; pk != nil && pk.Voice != nil {
			sp.Pack = pk.Voice
		}
	}
	if sp.Pack != nil && sp.Pack.Register != "" {
		sp.Register = sp.Pack.Register
	} else {
		sp.Register = dvRegisterFor(sp.Culture)
	}
	if _, ok := dvBank[sp.Register]; !ok {
		sp.Register = "earthy"
	}
	if pd != nil && pd.Voice != nil {
		sp.Person = pd.Voice
		sp.Tone = pd.Voice.Tone
		sp.Temper = pd.Voice.Temper
	}
	age := -1
	if lp != nil {
		sp.Opinion, sp.Mood, sp.Met = lp.Opinion, lp.Mood, lp.Met
		if sp.Temper == "" {
			sp.Temper = dvTemperFromTraits(lp)
		}
		age = lp.lvAge(lvYear(g))
	} else {
		sp.Opinion = clampi(g.Rep[a.Faction]/4, -100, 100)
	}
	sp.Child = a.Role == "child" || (age >= 0 && age < 14)
	if age >= 60 && sp.Rank == "common" {
		sp.Rank = "elder"
	}
	sp.RoleKey = dvRoleKey(a, sp.Child)
	return sp
}

// ---- helpers ----

func dvFirstSentence(s string) string {
	for i, r := range s {
		if (r == '.' || r == '!' || r == '?') && i+1 < len(s) && s[i+1] == ' ' {
			return s[:i+1]
		}
	}
	return s
}

func dvLowerFirst(s string) string {
	r := []rune(s)
	if len(r) < 2 || !unicode.IsUpper(r[0]) || !unicode.IsLower(r[1]) {
		return s // keeps "I ..." and acronyms
	}
	r[0] = unicode.ToLower(r[0])
	return string(r)
}

func dvUpperFirst(s string) string {
	r := []rune(s)
	if len(r) == 0 {
		return s
	}
	r[0] = unicode.ToUpper(r[0])
	return string(r)
}

// dvShortest keeps the shorter half of a list (cold or gruff speakers).
func dvShortest(list []string) []string {
	if len(list) < 3 {
		return list
	}
	c := append([]string(nil), list...)
	sort.SliceStable(c, func(i, j int) bool { return len(c[i]) < len(c[j]) })
	return c[:(len(c)+1)/2]
}

// dvInsertBeforeEnd puts ", word" before the closing punctuation of a line.
func dvInsertBeforeEnd(s, word string) string {
	s = strings.TrimRight(s, " ")
	if s == "" || word == "" {
		return s
	}
	end := ""
	for strings.HasSuffix(s, ".") || strings.HasSuffix(s, "!") || strings.HasSuffix(s, "?") || strings.HasSuffix(s, "...") {
		end = s[len(s)-1:] + end
		s = s[:len(s)-1]
	}
	return s + ", " + word + end
}

func (sp *dvSpeaker) honorifics() map[string]string {
	if sp.Pack != nil {
		return sp.Pack.Honorifics
	}
	return nil
}

// dvStrip removes the culture's honorific terms that dvSay itself appends (plainer tone).
func (sp *dvSpeaker) strip(s string) string {
	for _, h := range sp.honorifics() {
		if len(h) > 3 {
			s = strings.ReplaceAll(s, ", "+h, "")
		}
	}
	return s
}

// listenerKey is how the speaker's culture ranks the player.
func (g *Game) dvListenerKey() string {
	s := g.socialStanding()
	switch {
	case s >= 12:
		return "noble"
	case s >= 8:
		return "high"
	case s < 3:
		return "stranger"
	}
	return "common"
}

// dvAddress builds the name form: formal for strangers and the high-ranked, familiar for friends.
func (g *Game) dvAddress(sp *dvSpeaker) string {
	friend := sp.Opinion >= 25 || (sp.Met && sp.Opinion >= 10)
	high := sp.Rank == "noble" || sp.Rank == "ruler" || sp.Rank == "high"
	formal := !friend || (high && sp.Opinion < 55)
	if sp.Tone < 0 {
		formal = false
	}
	name := g.P.Name
	if sp.Pack == nil || sp.Pack.Address == nil {
		return name
	}
	form := sp.Pack.Address.Familiar
	if formal {
		form = sp.Pack.Address.Formal
	}
	if form == "" {
		return name
	}
	if !strings.Contains(form, "{name}") {
		if formal {
			return form
		}
		return name
	}
	title := ""
	if h := sp.honorifics(); h != nil {
		title = h[g.dvListenerKey()]
		if title == "" {
			title = h["stranger"]
		}
	}
	return strings.TrimSpace(cpFill(form, map[string]string{"name": name, "title": title}))
}

func (g *Game) dvVars(sp *dvSpeaker, extra map[string]string) map[string]string {
	v := map[string]string{"name": g.dvAddress(sp)}
	if bd := g.localBurg(); bd != nil {
		v["town"] = bd.Name
	}
	if st := g.W.stateByID[sp.A.State]; st != nil {
		v["realm"] = st.Name
	}
	if sp.Culture != nil {
		v["culture"] = sp.Culture.Name
	}
	if rel := g.W.relByID[sp.A.Religion]; rel != nil {
		v["faith"], v["deity"] = rel.Name, rel.Deity
	}
	for k, x := range extra {
		if x != "" {
			v[k] = x
		}
	}
	return v
}

func (sp *dvSpeaker) list(kind string) []string {
	if sp.Pack != nil {
		var l []string
		switch kind {
		case "greet":
			l = sp.Pack.Greet
		case "farewell":
			l = sp.Pack.Farewell
		case "thanks":
			l = sp.Pack.Thanks
		case "apology":
			l = sp.Pack.Apology
		case "agree":
			l = sp.Pack.Agree
		case "refuse":
			l = sp.Pack.Refuse
		case "praise":
			l = sp.Pack.Praise
		case "insult":
			l = sp.Pack.Insults
		case "curse":
			l = sp.Pack.Curses
		case "oath":
			l = sp.Pack.Oaths
		case "gossipOpen":
			l = sp.Pack.Gossip
		}
		if len(l) > 0 {
			return l
		}
	}
	return dvBank[sp.Register][kind]
}

var dvPleasantKinds = map[string]bool{"greet": true, "farewell": true, "thanks": true}
var dvHesitateKinds = map[string]bool{"greet": true, "refuse": true, "apology": true, "agree": true, "gossipOpen": true}
var dvTicKinds = map[string]bool{"greet": true, "agree": true, "refuse": true, "gossipOpen": true, "thanks": true, "praise": true}
var dvHonorKinds = map[string]bool{"greet": true, "farewell": true, "thanks": true, "apology": true, "agree": true, "refuse": true}

// dvSay returns one line of the given kind in the NPC's voice (bare speech, no quotes; callers wrap them).
// It returns "" when the actor or kind is unknown, so callers keep their existing line as the last resort.
func (g *Game) dvSay(a *Actor, kind string, vars map[string]string) string {
	sp := g.dvVoice(a)
	if sp == nil {
		return ""
	}
	list := sp.list(kind)
	if len(list) == 0 {
		return ""
	}
	day := fmt.Sprint(DayOf(g.Turn))
	id := fmt.Sprint(a.ID)
	w := g.W
	cold := sp.Temper == "cold" || sp.Temper == "gruff" || (sp.Opinion <= -25 && dvPleasantKinds[kind])
	warm := sp.Temper == "warm" || (sp.Opinion >= 40 && dvPleasantKinds[kind])
	if cold {
		list = dvShortest(list)
	}
	pick := ""
	if kind == "oath" && sp.Person != nil && sp.Person.Oath != "" {
		pick = sp.Person.Oath
	}
	if pick == "" && sp.RoleKey == "priest" && kind == "greet" {
		if rel := w.relByID[a.Religion]; rel != nil && rel.Lore != nil && rel.Lore.Clergy != nil {
			cl := rel.Lore.Clergy
			pick = cl.GreetStranger
			if a.Religion == g.HomeFaith && cl.GreetFaithful != "" {
				pick = cl.GreetFaithful
			}
		}
	}
	if pick == "" && sp.RoleKey != "" {
		if rl := dvRoleBank[sp.RoleKey][kind]; len(rl) > 0 {
			mod := 3
			if sp.Pack != nil && !sp.Child {
				mod = 4
			}
			if w.cpIdx(mod, id, kind, day, "role") == 0 {
				pick = w.cpChoose(rl, id, kind, day, "rl")
			}
		}
	}
	if pick == "" {
		pick = w.cpChoose(list, id, kind, day)
	}
	s := cpFill(pick, g.dvVars(sp, vars))
	if dvPleasantKinds[kind] && cold {
		s = dvFirstSentence(s)
	}
	if sp.Tone < 0 {
		s = sp.strip(s)
	}
	// honorific: grander speakers, or the low-born before a person of standing
	if dvHonorKinds[kind] && sp.Tone >= 0 && !cold {
		lk := g.dvListenerKey()
		low := (sp.Rank == "low" || sp.Rank == "common") && (lk == "high" || lk == "noble")
		if (sp.Tone > 0 || low) && !strings.Contains(s, g.P.Name) {
			if h := sp.honorifics(); h != nil {
				hon := h[lk]
				if hon == "" && sp.Tone > 0 {
					hon = h["high"]
				}
				if hon != "" && lk != "common" {
					s = dvInsertBeforeEnd(s, hon)
				}
			}
		}
	}
	// temper
	switch {
	case warm && dvPleasantKinds[kind]:
		s += " " + w.cpChoose(dvWarmClauses, id, kind, day, "warm")
	case sp.Temper == "anxious" && dvHesitateKinds[kind]:
		s = w.cpChoose(dvHesitations, id, kind, day, "hes") + dvLowerFirst(s)
	case sp.Temper == "eager" && strings.HasSuffix(s, "."):
		s = strings.TrimSuffix(s, ".") + "!"
	case sp.Temper == "" && sp.Mood <= -40 && dvHesitateKinds[kind]:
		s = w.cpChoose(dvGlum, id, kind, day, "glum") + dvLowerFirst(s)
	}
	// a tic, sparingly: at most one per reply, 1 in 3
	if sp.Person != nil && len(sp.Person.Tics) > 0 && dvTicKinds[kind] && len(s) > 10 && w.cpIdx(3, id, kind, day, "tic") == 0 {
		tic := w.cpChoose(sp.Person.Tics, id, kind, day, "tics")
		if tic != "" && !strings.Contains(strings.ToLower(s), strings.ToLower(tic)) {
			s = dvUpperFirst(tic) + ", " + dvLowerFirst(s)
		}
	}
	return strings.TrimSpace(s)
}

// dvSayQ is dvSay wrapped in the quotes the talk window uses, or "" when there is no voice.
func (g *Game) dvSayQ(a *Actor, kind string, vars map[string]string) string {
	if s := g.dvSay(a, kind, vars); s != "" {
		return "\"" + s + "\""
	}
	return ""
}

// dvTalkTag is a short hint for the look text, e.g. "(speaking formally)".
func (g *Game) dvTalkTag(a *Actor) string {
	if a == nil || a.Tpl != "npc" || a.AI == "hostile" {
		return ""
	}
	sp := g.dvVoice(a)
	if sp == nil {
		return ""
	}
	if t := dvTemperTag[sp.Temper]; t != "" {
		return "(speaking " + t + ")"
	}
	if t := dvRegisterTag[sp.Register]; t != "" {
		return "(speaking " + t + ")"
	}
	return ""
}

// dvQuip is an occasional one-liner (about one hour in three per speaker), "" otherwise.
func (g *Game) dvQuip(a *Actor) string {
	if a == nil || a.Tpl != "npc" || a.AI == "hostile" {
		return ""
	}
	sp := g.dvVoice(a)
	if sp == nil {
		return ""
	}
	w := g.W
	id := fmt.Sprint(a.ID)
	hour := fmt.Sprint(g.Turn / 60)
	if w.cpIdx(3, id, "quip", hour) != 0 {
		return ""
	}
	var pool []string
	if sp.Pack != nil {
		bk := ""
		switch {
		case g.cyIsNight() && len(sp.Pack.Barks["night"]) > 0:
			bk = "night"
		case sp.RoleKey == "guard":
			bk = "guard"
		case sp.RoleKey == "priest":
			bk = "temple"
		case a.Role == "smith":
			bk = "smith"
		case sp.RoleKey == "merchant":
			bk = "market"
		case a.Role == "ferry" || a.Role == "dockmaster":
			bk = "harbor"
		case a.Role == "innkeeper" || a.Role == "tavern":
			bk = "tavern"
		}
		pool = append(pool, sp.Pack.Barks[bk]...)
		if len(pool) == 0 {
			pool = append(pool, sp.Pack.Proverbs...)
		}
		if sp.Person != nil && sp.Person.Topic != "" && w.cpIdx(4, id, hour, "topic") == 0 {
			return cpFill("They go on about "+sp.Person.Topic+".", nil)
		}
	}
	if len(pool) == 0 {
		pool = dvQuips[sp.Register]
	}
	line := w.cpChoose(pool, id, "quip", hour, "l")
	if line == "" {
		return ""
	}
	return cpFill(line, g.dvVars(sp, nil))
}
