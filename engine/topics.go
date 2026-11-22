package engine

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// Topic webs (prefix tp): conversation as asking about subjects. One talk option, "Ask about...", opens a list of
// subjects built from the speaker (role, trust, culture, what they know). Each answer is assembled from real world
// data and the culture pack, reveals a fact to the case-notes system (knowledge.go), unlocks follow-up topics,
// moves the speaker's opinion (lives.go memories) and is remembered (g.ext("tp"), planet-local, capped).
// Reluctant speakers can be talked round with a skill check (checks.go); tpChoice is a reusable branching helper.
// Nothing here draws g.R for text; checks use the game's own roll helpers.

const (
	tpCap  = 240 // remembered (speaker, topic) pairs
	tpPage = 9   // topics per page
)

func init() {
	unLocalKeys = append(unLocalKeys, "tp")
	talkHooks = append(talkHooks, tpTalk)
	tpTopics = tpTable()
}

// ---------------------------------------------------------------- state

type tpAsk struct {
	Day  int    `json:"d"`
	N    int    `json:"n"`
	Fail int    `json:"f,omitempty"` // day+1 of a failed attempt
	Gist string `json:"g,omitempty"`
}

type tpState struct {
	Asked map[string]*tpAsk `json:"a,omitempty"` // "<who>|<topic>"
	Facts map[string]int    `json:"f,omitempty"` // dedupe key -> knowledge fact id
	Last  map[string]string `json:"l,omitempty"` // who -> last topic id
}

func tpSt(g *Game) *tpState {
	s := g.ext("tp", func() any { return &tpState{} }).(*tpState)
	if s.Asked == nil {
		s.Asked = map[string]*tpAsk{}
	}
	if s.Facts == nil {
		s.Facts = map[string]int{}
	}
	if s.Last == nil {
		s.Last = map[string]string{}
	}
	return s
}

func (s *tpState) has(who, topic string) bool {
	r := s.Asked[who+"|"+topic]
	return r != nil && r.N > 0
}

func (s *tpState) prune() {
	if len(s.Facts) > 400 {
		s.Facts = map[string]int{}
	}
	if len(s.Asked) <= tpCap {
		return
	}
	keys := make([]string, 0, len(s.Asked))
	for k := range s.Asked {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		a, b := s.Asked[keys[i]], s.Asked[keys[j]]
		if a.Day != b.Day {
			return a.Day < b.Day
		}
		return keys[i] < keys[j]
	})
	for _, k := range keys[:len(s.Asked)-tpCap+20] {
		delete(s.Asked, k)
	}
}

// ---------------------------------------------------------------- context

type tpCtx struct {
	g     *Game
	a     *Actor
	p     *lvPerson
	syn   *lvPerson
	bd    *BurgData
	cu    *CultureData
	pack  *CulturePack
	cust  *CustomsPack
	vp    *VoicePack
	stt   *StateData
	rel   *ReligionData
	faith *FaithPack
	per   *PersonData
	voice *PersonVoice
	lvl   int
	boost int
	tags  map[string]bool
	child bool
	tone  string
	key   string
	vars  map[string]string
}

var tpRawRe = regexp.MustCompile(`\s?\{[a-z]+\}`)

func tpKeyFor(g *Game, a *Actor) string {
	if p := lvPersonOf(g, a); p != nil {
		return p.Key
	}
	return fmt.Sprint("a", a.ID)
}

func tpLevelFor(g *Game, a *Actor, p *lvPerson) int {
	if a.Angry {
		return -1
	}
	t := clampi(g.Rep[a.Faction]/8, -30, 30)
	if p != nil {
		t += p.Opinion
		if p.Met {
			t += 8
		}
	}
	switch {
	case t < -25:
		return -1
	case t < 15:
		return 0
	case t < 45:
		return 1
	}
	return 2
}

func (g *Game) tpCtxFor(a *Actor) *tpCtx {
	w := g.W
	c := &tpCtx{g: g, a: a, tags: map[string]bool{}}
	c.p = lvPersonOf(g, a)
	c.key = tpKeyFor(g, a)
	if g.Cur != nil && g.Cur.Burg > 0 {
		c.bd = w.burgByID[g.Cur.Burg]
	}
	if c.bd != nil {
		c.cu = w.cultByID[c.bd.Culture]
	}
	if c.cu == nil {
		c.cu = g.lxCultureAt()
	}
	if c.cu != nil && c.cu.Pack != nil {
		c.pack = c.cu.Pack
		c.cust = c.pack.Customs
		c.vp = c.pack.Voice
	}
	sid := a.State
	if c.bd != nil && c.bd.State > 0 {
		sid = c.bd.State
	}
	if sid <= 0 && w.In(g.WX, g.WY) {
		sid = w.State[w.Idx(g.WX, g.WY)]
	}
	if sid > 0 {
		c.stt = w.stateByID[sid]
	}
	c.rel = w.relByID[a.Religion]
	if c.rel == nil && c.stt != nil {
		c.rel = w.relByID[c.stt.Religion]
	}
	if c.rel != nil {
		c.faith = c.rel.Lore
	}
	if a.SimPer > 0 {
		for i := range w.People {
			if w.People[i].ID == a.SimPer {
				c.per = &w.People[i]
				c.voice = c.per.Voice
				break
			}
		}
	}
	c.lvl = tpLevelFor(g, a, c.p)
	// tone
	switch {
	case c.voice != nil && c.voice.Temper != "":
		c.tone = tpToneOf(c.voice.Temper)
	case c.p != nil && c.p.Trait[4] > 65:
		c.tone = "nervous"
	case c.p != nil && c.p.Trait[3] < 35:
		c.tone = "gruff"
	default:
		c.tone = "warm"
	}
	// who they are
	age := -1
	if c.p != nil && c.p.Born > 0 {
		age = c.p.lvAge(lvYear(g))
	}
	for _, t := range tpRoleTags[a.Role] {
		c.tags[t] = true
	}
	if age >= 55 {
		c.tags["elder"], c.tags["past"], c.tags["customs"] = true, true, true
	}
	if a.Role == "child" || (age >= 0 && age < 15) {
		c.child = true
		c.tags = map[string]bool{"child": true, "rumour": true}
	}
	c.vars = map[string]string{"name": g.P.Name, "title": nonEmpty(a.Title, a.Role), "time": strings.ToLower(SeasonNames[g.Season()]), "kin": "kin"}
	if c.stt != nil {
		c.vars["realm"] = c.stt.Name
	}
	if c.bd != nil {
		c.vars["town"] = c.bd.Name
	}
	if c.cu != nil {
		c.vars["culture"] = c.cu.Name
		if f := strings.Split(c.cu.Food, ","); len(f) > 0 {
			c.vars["item"] = strings.TrimSpace(f[0])
		}
	}
	if c.rel != nil {
		c.vars["faith"], c.vars["deity"] = c.rel.Name, c.rel.Deity
	}
	if c.p != nil {
		for _, k := range c.p.Kin {
			if r := c.p.Rel[k]; r != "" {
				c.vars["kin"] = r
				break
			}
		}
	}
	return c
}

func (c *tpCtx) who() *lvPerson {
	if c.p != nil {
		return c.p
	}
	if c.syn == nil {
		c.syn = &lvPerson{Key: c.key, Name: c.a.Name, Alive: true}
	}
	return c.syn
}

func (c *tpCtx) realm() string {
	if c.stt != nil {
		return c.stt.Name
	}
	return "this land"
}

func (c *tpCtx) town() string {
	if c.bd != nil {
		return c.bd.Name
	}
	return "this place"
}

func tpUp(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func tpA(s string) string {
	if s != "" && strings.ContainsRune("aeiouAEIOU", rune(s[0])) {
		return "an " + s
	}
	return "a " + s
}

// fill runs the pack placeholders and strips anything left over (and em dashes from pack text).
func (c *tpCtx) fill(s string) string {
	s = cpFill(s, c.vars)
	s = tpRawRe.ReplaceAllString(s, "")
	s = strings.NewReplacer(" — ", ", ", "—", "-").Replace(s)
	return strings.TrimSpace(s)
}

func (c *tpCtx) pick(list []string, parts ...string) string {
	return c.g.W.cpChoose(list, append([]string{c.key}, parts...)...)
}

func (c *tpCtx) idx(n int, parts ...string) int {
	return c.g.W.cpIdx(n, append([]string{c.key}, parts...)...)
}

func (c *tpCtx) expert(t *tpTopic) bool {
	if len(t.Know) == 0 {
		return true
	}
	for _, k := range t.Know {
		if c.tags[k] {
			return true
		}
	}
	return false
}

func (c *tpCtx) visible(t *tpTopic, st *tpState) bool {
	if t.Avail != nil && !t.Avail(c) {
		return false
	}
	if t.After != "" && !st.has(c.key, t.After) {
		return false
	}
	if t.Common || c.expert(t) {
		return true
	}
	return c.lvl >= 1 && !c.child
}

// trim limits how many sentences a speaker gives: a stranger is told less, a terse voice says less.
func (c *tpCtx) trim(say []string) []string {
	n := 2 + c.lvl + c.boost
	if c.vp != nil {
		switch c.vp.Register {
		case "terse":
			n--
		case "formal", "flowery", "courtly":
			n++
		}
	}
	n = clampi(n, 1, 6)
	if len(say) > n {
		return say[:n]
	}
	return say
}

// flav puts one of the speaker's own tics in front of an answer now and then.
func (c *tpCtx) flav(s, topic string) string {
	if c.voice != nil && len(c.voice.Tics) > 0 && HashStr(c.key+topic)%3 == 0 {
		t := c.voice.Tics[int(HashStr(c.key+topic+"t"))%len(c.voice.Tics)]
		return tpUp(t) + ". " + s
	}
	return s
}

func (c *tpCtx) op(kind, text string, w int) bool {
	if c.p == nil {
		return false
	}
	c.g.lvMemoryAdd(c.p, kind, text, w)
	return true
}

// hasAct returns the first taboo of the culture naming one of the acts (pack or generic).
func (c *tpCtx) taboo(acts ...string) (txt, rebuke string, sev int, ok bool) {
	if c.cust != nil {
		for _, t := range c.cust.Taboos {
			for _, a := range acts {
				if t.Act == a {
					return t.Text, t.Rebuke, t.Severity, true
				}
			}
		}
	}
	return "", "", 0, false
}

// ---------------------------------------------------------------- topic model

type tpFact struct{ Kind, Subject, Text string }

type tpAnswer struct {
	Say   []string // sentences in the speaker's voice (one quote), trimmed by trust
	Narr  []string // plain paragraphs after the speech
	Fact  *tpFact
	Deny  string // they will not say
	Rep   int    // realm reputation change for a blunder
	Grief bool
}

type tpTopic struct {
	ID, Label, Prompt, After string
	LabelFn                  func(c *tpCtx) string
	Know                     []string
	Common                   bool
	Need                     int
	Kind                     string // check kind when the speaker is reluctant
	Tact                     int    // opinion on the first ask (+ warms, - tactless)
	Avail                    func(c *tpCtx) bool
	Ans                      func(c *tpCtx) tpAnswer
	Run                      func(c *tpCtx, back func())
}

var tpTopics []tpTopic

func tpLabel(c *tpCtx, t *tpTopic) string {
	if t.LabelFn != nil {
		return t.LabelFn(c)
	}
	return c.fill(t.Label)
}

func tpOptText(c *tpCtx, t *tpTopic, asked, follow bool) string {
	txt := "Ask about " + tpLabel(c, t) + "."
	if t.Prompt != "" {
		txt = c.fill(t.Prompt)
	}
	switch {
	case asked:
		return "{{K|" + txt + " (asked)}}"
	case follow:
		txt = "{{C|" + txt + "}}"
	}
	if t.Need > c.lvl && t.Kind != "" {
		txt = c.g.checkLabel(t.Kind, tpDC(c, t)) + " " + txt
	}
	return txt
}

func tpDC(c *tpCtx, t *tpTopic) int {
	dc := 8 + 2*(t.Need-c.lvl)
	if c.tone == "proud" || c.tone == "gruff" {
		dc++
	}
	return clampi(dc, 7, 13)
}

// ---------------------------------------------------------------- menu

func tpTalk(g *Game, a *Actor) []Option {
	if g == nil || a == nil || g.W == nil || g.Cur == nil || g.P == nil || a.Tpl != "npc" || a.AI == "hostile" || a.AI == "follower" || a.Angry {
		return nil
	}
	switch a.Role {
	case "foe", "invader", "captive":
		return nil
	}
	return []Option{{"{{c|Ask about...}} {{K|(subjects)}}", func() { g.tpOpen(a, 0) }}}
}

func (g *Game) tpShow(c *tpCtx, text string, opts []Option, back func()) {
	if g.Mode != "talk" || g.Talk == nil {
		return
	}
	g.Popup = Wrap(g.lxFilter(c.a, text), 100)
	g.Opts = opts
	g.Sel = 0
	delete(th2Back, g)
	if len(opts) > 0 {
		th2Back[g] = &th2Sub{first: &g.Opts[0], n: len(g.Opts), back: back}
	}
}

func (g *Game) tpBackToTalk() { g.talkMenu("\"Anything else?\"") }

func tpRecall(g *Game, c *tpCtx) string {
	st := tpSt(g)
	id := st.Last[c.key]
	if id == "" {
		return ""
	}
	r := st.Asked[c.key+"|"+id]
	if r == nil || g.Day()-r.Day > 40 {
		return ""
	}
	for i := range tpTopics {
		if tpTopics[i].ID == id {
			return fmt.Sprintf("You asked about %s before.", tpLabel(c, &tpTopics[i]))
		}
	}
	return ""
}

// tpGreeting: a speaker who remembers what you asked (optional hook in greeting()).
func (g *Game) tpGreeting(a *Actor) string {
	if g == nil || a == nil || g.W == nil || g.Cur == nil || a.Tpl != "npc" || a.AI == "hostile" || a.AI == "follower" {
		return ""
	}
	key := tpKeyFor(g, a)
	if tpSt(g).Last[key] == "" || HashStr(fmt.Sprint(key, g.Day()))%3 != 0 {
		return ""
	}
	c := g.tpCtxFor(a)
	if r := tpRecall(g, c); r != "" {
		return "\"Ah, you again. " + r + "\""
	}
	return ""
}

type tpEntry struct {
	t      *tpTopic
	asked  bool
	follow bool
}

func (g *Game) tpOpen(a *Actor, page int) {
	c := g.tpCtxFor(a)
	if !g.lxGossipOK(a) {
		g.talkMenu(g.lxGossipGarble(a))
		return
	}
	if c.lvl < 0 {
		line := "\"Not to you.\""
		if c.vp != nil && len(c.vp.Refuse) > 0 {
			line = "\"" + c.fill(c.pick(c.vp.Refuse, "refuse")) + "\""
		} else {
			line = Pick(NewRng(c.key), knRefusals)
		}
		g.talkMenu(line)
		return
	}
	st := tpSt(g)
	var fresh, follow, done []tpEntry
	for i := range tpTopics {
		t := &tpTopics[i]
		if !c.visible(t, st) {
			continue
		}
		switch {
		case st.has(c.key, t.ID) && t.Run == nil:
			done = append(done, tpEntry{t, true, false})
		case t.After != "":
			follow = append(follow, tpEntry{t, false, true})
		default:
			fresh = append(fresh, tpEntry{t, false, false})
		}
	}
	all := append(append(follow, fresh...), done...)
	if len(all) == 0 {
		g.talkMenu("\"I have nothing to tell.\"")
		return
	}
	pages := (len(all) + tpPage - 1) / tpPage
	if page >= pages || page < 0 {
		page = 0
	}
	lo, hi := page*tpPage, mini(len(all), (page+1)*tpPage)
	back := func() { g.tpBackToTalk() }
	var opts []Option
	for _, e := range all[lo:hi] {
		e := e
		opts = append(opts, Option{tpOptText(c, e.t, e.asked, e.follow), func() { g.tpAskTopic(c, e.t, page) }})
	}
	if pages > 1 {
		opts = append(opts, Option{fmt.Sprintf("{{K|More subjects... (%d/%d)}}", page+1, pages), func() { g.tpOpen(a, page+1) }})
	}
	opts = append(opts, Option{"{{K|Back.}}", back})
	head := "\"What would you know?\""
	if r := tpRecall(g, c); r != "" {
		head += " " + r
	}
	g.tpShow(c, head, opts, back)
}

func tpGist(say []string) string {
	if len(say) == 0 {
		return ""
	}
	s := Plain(say[0])
	if i := strings.Index(s, ". "); i > 20 {
		s = s[:i+1]
	}
	if len(s) > 130 {
		s = s[:127] + "..."
	}
	return s
}

func (g *Game) tpAskTopic(c *tpCtx, t *tpTopic, page int) {
	st := tpSt(g)
	k := c.key + "|" + t.ID
	rec := st.Asked[k]
	day := g.Day()
	back := func() { g.tpOpen(c.a, page) }
	if t.Run != nil {
		if rec == nil {
			rec = &tpAsk{}
			st.Asked[k] = rec
		}
		rec.N++
		rec.Day = day
		st.Last[c.key] = t.ID
		st.prune()
		t.Run(c, back)
		return
	}
	if rec != nil && rec.N > 0 {
		g.tpRemind(c, t, rec, page)
		return
	}
	if rec != nil && rec.Fail == day+1 {
		g.tpShow(c, c.pick(tpLose[c.tone], t.ID, "again")+"\n\n{{K|They have said what they will for today.}}", []Option{{"{{K|Back.}}", back}}, back)
		return
	}
	if c.lvl < t.Need && t.Kind != "" {
		dc := tpDC(c, t)
		label := checkKinds[t.Kind].label
		if g.roll2d6Check(g.checkMod(t.Kind), dc) {
			g.gainXP(4 + dc)
			c.boost = 1
			pre := "{{G|[" + label + ": Success]}}\n\n" + c.pick(tpWin[c.tone], t.ID) + "\n\n"
			g.tpAnswer(c, t, page, pre)
			return
		}
		if rec == nil {
			rec = &tpAsk{}
			st.Asked[k] = rec
		}
		rec.Fail = day + 1
		c.op("insult", "pushed me to say what I would not", -3)
		g.tpShow(c, "{{R|["+label+": Failure]}}\n\n"+c.pick(tpLose[c.tone], t.ID)+g.loApology(c.a)+"\n\n"+tpMsgCool, []Option{{"{{K|Back.}}", back}}, back)
		return
	}
	g.tpAnswer(c, t, page, "")
}

func (g *Game) tpFollowOpts(c *tpCtx, t *tpTopic, page int) []Option {
	st := tpSt(g)
	var opts []Option
	for i := range tpTopics {
		f := &tpTopics[i]
		if f.After != t.ID || !c.visible(f, st) {
			continue
		}
		opts = append(opts, Option{tpOptText(c, f, st.has(c.key, f.ID) && f.Run == nil, true), func() { g.tpAskTopic(c, f, page) }})
	}
	opts = append(opts, Option{"Ask about something else.", func() { g.tpOpen(c.a, page) }})
	opts = append(opts, Option{"{{K|Back.}}", func() { g.tpBackToTalk() }})
	return opts
}

func (g *Game) tpRemind(c *tpCtx, t *tpTopic, rec *tpAsk, page int) {
	lab := tpLabel(c, t)
	list := tpAgain
	if g.Day() == rec.Day {
		list = tpAgainDay
	}
	txt := fmt.Sprintf(c.pick(list, t.ID, fmt.Sprint(g.Day())), lab)
	if rec.Gist != "" {
		txt += " They give you the gist again: \"" + rec.Gist + "\""
	}
	opts := g.tpFollowOpts(c, t, page)
	g.tpShow(c, txt, opts, func() { g.tpOpen(c.a, page) })
}

func (g *Game) tpAnswer(c *tpCtx, t *tpTopic, page int, prefix string) {
	st := tpSt(g)
	back := func() { g.tpOpen(c.a, page) }
	ans := t.Ans(c)
	if ans.Deny != "" {
		g.tpShow(c, prefix+ans.Deny, []Option{{"{{K|Back.}}", back}}, back)
		return
	}
	expert := c.expert(t)
	say := ans.Say
	if expert {
		say = c.trim(say)
	} else {
		cls := "specialist"
		for _, k := range t.Know {
			cls = k
			break
		}
		say = append([]string{fmt.Sprintf(c.pick(tpHearsay, t.ID), cls)}, say[:mini(1, len(say))]...)
	}
	for i := range say {
		say[i] = c.fill(say[i])
	}
	text := prefix
	if len(say) > 0 {
		text += "\"" + c.flav(strings.Join(say, " "), t.ID) + "\""
	}
	for _, n := range ans.Narr {
		text += "\n\n" + c.fill(n)
	}
	if expert && ans.Fact != nil {
		g.tpReveal(c, t.ID+":"+fmt.Sprint(c.bdID()), ans.Fact)
	}
	rec := st.Asked[c.key+"|"+t.ID]
	first := rec == nil || rec.N == 0
	if rec == nil {
		rec = &tpAsk{}
		st.Asked[c.key+"|"+t.ID] = rec
	}
	rec.N++
	rec.Day = g.Day()
	rec.Fail = 0
	rec.Gist = tpGist(say)
	st.Last[c.key] = t.ID
	st.prune()
	if first {
		g.gainXP(2)
		switch {
		case t.Tact > 0:
			if c.op("help", "asked kindly after "+tpLabel(c, t), t.Tact) {
				text += "\n\n" + tpMsgWarm
			}
		case t.Tact < 0:
			if c.op("insult", "asked a tactless question about "+tpLabel(c, t), t.Tact) {
				if ans.Grief {
					text += "\n\n" + tpMsgGrief
				} else {
					text += "\n\n" + tpMsgCool
				}
			}
			if ans.Rep < 0 {
				g.AddRep(c.a.Faction, ans.Rep, " for a blasphemous question")
			}
		}
	}
	g.tpShow(c, text, g.tpFollowOpts(c, t, page), back)
}

func (c *tpCtx) bdID() int {
	if c.bd != nil {
		return c.bd.ID
	}
	return 0
}

// tpReveal records a fact in the knowledge system (case notes, rumour spread) and learns it from the speaker.
func (g *Game) tpReveal(c *tpCtx, key string, f *tpFact) *knFact {
	st := knSt(g)
	ts := tpSt(g)
	var kf *knFact
	if id, ok := ts.Facts[key]; ok {
		kf = st.fact(id)
	}
	p := c.who()
	if kf == nil {
		subj := f.Subject
		if subj == "" && c.bd != nil {
			subj = fmt.Sprintf("burg:%d", c.bd.ID)
		}
		kf = st.add(knFact{Kind: f.Kind, Subject: subj, Text: f.Text, True: true, Source: p.Key, Day: lvDay(g), Burg: c.bdID()})
		ts.Facts[key] = kf.ID
	}
	knGive(p, kf.ID)
	g.knLearn(kf, p, c.lvl, "ask")
	return kf
}

// ---------------------------------------------------------------- data helpers

// events: chronicle events near here or of this realm, newest first (misc entries excluded unless asked for).
func (c *tpCtx) events(kinds ...string) []int {
	g := c.g
	w := g.W
	seen := map[int]bool{}
	var out []int
	ok := func(i int) bool {
		if len(kinds) == 0 {
			return w.Events[i].Kind != "misc"
		}
		for _, k := range kinds {
			if k == w.Events[i].Kind {
				return true
			}
		}
		return false
	}
	for dy := -8; dy <= 8; dy++ {
		for dx := -8; dx <= 8; dx++ {
			if !w.In(g.WX+dx, g.WY+dy) {
				continue
			}
			for _, i := range w.eventsAt[w.Idx(g.WX+dx, g.WY+dy)] {
				if i >= 0 && i < len(w.Events) && !seen[i] && ok(i) {
					seen[i] = true
					out = append(out, i)
				}
			}
		}
	}
	if c.stt != nil {
		for i := len(w.Events) - 1; i >= 0 && len(out) < 40; i-- {
			if w.Events[i].St == c.stt.ID && ok(i) && !seen[i] {
				seen[i] = true
				out = append(out, i)
			}
		}
	}
	sort.Sort(sort.Reverse(sort.IntSlice(out)))
	return out
}

func (c *tpCtx) ago(year int) string {
	d := c.g.W.Year + c.g.Day()/YearDays - year
	switch {
	case d <= 0:
		return "this very year"
	case d == 1:
		return "a year ago"
	}
	return fmt.Sprintf("%d years ago", d)
}

func tpTier(g *Game, bd *BurgData) string {
	if n := g.W.nmData(); n != nil {
		if b, ok := n.Burgs[bd.ID]; ok && b.Tier != "" {
			return b.Tier
		}
	}
	switch g.W.cityTier[bd.ID] {
	case 2:
		return "great city"
	case 1:
		return "city"
	}
	if bd.Pop < 3 {
		return "village"
	}
	return "town"
}

func tpAt(w *WorldData, bd *BurgData) (int, int) { return bd.P % w.W, bd.P / w.W }

// ---------------------------------------------------------------- topic table

func tpHas(c *tpCtx) bool { return true }

func tpTable() []tpTopic {
	return []tpTopic{
		{ID: "town", Label: "the town", Common: true, Kind: "persuasion", Avail: func(c *tpCtx) bool { return c.bd != nil }, Ans: tpAnsTown},
		{ID: "town_notables", Label: "who is worth knowing here", After: "town", Common: true, Avail: func(c *tpCtx) bool { return c.bd != nil }, Ans: tpAnsNotables},
		{ID: "ruler", Label: "the ruler", Common: true, Know: []string{"noble", "guard", "elder", "scholar"}, Kind: "persuasion", Avail: func(c *tpCtx) bool { return c.stt != nil && c.stt.Ruler != "" }, Ans: tpAnsRuler},
		{ID: "ruler_past", Label: "who ruled before", After: "ruler", Know: []string{"noble", "elder", "scholar", "past"}, Need: 1, Kind: "persuasion", Avail: func(c *tpCtx) bool { return len(tpDeadRulers(c)) > 0 }, Ans: tpAnsRulerPast},
		{ID: "war", Label: "the war", Common: true, Know: []string{"guard", "war", "elder", "noble", "scholar"}, Kind: "persuasion", Avail: func(c *tpCtx) bool { return tpWarOf(c) != nil }, Ans: tpAnsWar},
		{ID: "war_cause", Label: "who started it", After: "war", Know: []string{"guard", "war", "elder", "noble", "scholar"}, Need: 1, Kind: "persuasion", Avail: func(c *tpCtx) bool { return tpWarOf(c) != nil }, Ans: tpAnsWarCause},
		{ID: "war_peace", Label: "the peace", After: "war_cause", Know: []string{"guard", "war", "elder", "noble", "scholar"}, Need: 1, Kind: "empathy", Avail: func(c *tpCtx) bool { return tpWarOf(c) != nil }, Ans: tpAnsWarPeace},
		{ID: "law", Label: "the law", Know: []string{"law"}, Kind: "intimidate", Avail: func(c *tpCtx) bool { return c.stt != nil || (c.cust != nil && c.cust.Law != nil) }, Ans: tpAnsLaw},
		{ID: "law_crime", Label: "what is done with thieves and killers", After: "law", Know: []string{"law"}, Need: 1, Kind: "persuasion", Ans: tpAnsLawCrime},
		{ID: "faith", Label: "the faith", Common: true, Know: []string{"priest", "faith", "elder"}, Kind: "persuasion", Avail: func(c *tpCtx) bool { return c.rel != nil }, Ans: tpAnsFaith},
		{ID: "faith_days", Label: "the holy days", After: "faith", Know: []string{"priest", "faith", "elder"}, Avail: func(c *tpCtx) bool { return c.rel != nil }, Ans: tpAnsHolyDays},
		{ID: "faith_saints", Label: "the saints and holy dead", After: "faith", Know: []string{"priest", "faith", "elder", "scholar"}, Avail: func(c *tpCtx) bool { return c.faith != nil && len(c.faith.Saints) > 0 }, Ans: tpAnsSaints},
		{ID: "faith_heresy", Label: "the heresies", After: "faith", Know: []string{"priest", "scholar"}, Need: 2, Kind: "persuasion", Avail: func(c *tpCtx) bool { return c.faith != nil && len(c.faith.Heresies) > 0 }, Ans: tpAnsHeresy},
		{ID: "customs", Label: "local customs", Common: true, Know: []string{"elder", "customs", "priest", "noble"}, Tact: 0, Avail: func(c *tpCtx) bool { return c.cu != nil }, Ans: tpAnsCustoms},
		{ID: "c_taboo", Label: "what is forbidden here", After: "customs", Know: []string{"elder", "customs", "priest", "guard"}, Tact: 1, Avail: func(c *tpCtx) bool { return c.cu != nil }, Ans: tpAnsTaboo},
		{ID: "c_guest", Label: "the rules of a guest", After: "customs", Know: []string{"elder", "customs", "merchant", "priest"}, Tact: 1, Avail: func(c *tpCtx) bool { return c.cu != nil }, Ans: tpAnsGuest},
		{ID: "c_rites", Label: "their rites and feasts", After: "customs", Know: []string{"elder", "customs", "priest"}, Avail: func(c *tpCtx) bool { return tpRiteAvail(c) }, Ans: tpAnsRites},
		{ID: "lodging", Label: "a roof for the night", Prompt: "Ask whether they can spare a bed for the night.", Common: true, Avail: func(c *tpCtx) bool { return c.bd != nil && !c.child }, Run: tpRunLodging},
		{ID: "road", Label: "the road and the neighbours", Common: true, Know: []string{"merchant", "road", "guard"}, Ans: tpAnsRoad},
		{ID: "road_neigh", Label: "the neighbouring realms", After: "road", Know: []string{"merchant", "guard", "noble", "scholar"}, Avail: func(c *tpCtx) bool { return c.stt != nil && len(c.stt.Relations) > 0 }, Ans: tpAnsNeighbours},
		{ID: "road_danger", Label: "the dangers of the road", After: "road", Know: []string{"guard", "merchant", "road", "elder"}, Avail: func(c *tpCtx) bool { return len(tpOutlawsNear(c)) > 0 }, Ans: tpAnsDanger},
		{ID: "trade", Label: "trade and prices", Common: true, Know: []string{"merchant"}, Kind: "persuasion", Avail: func(c *tpCtx) bool { return c.bd != nil }, Ans: tpAnsTrade},
		{ID: "trade_goods", Label: "local crafts and wares", After: "trade", Know: []string{"merchant", "scholar"}, Avail: func(c *tpCtx) bool { return c.cu != nil }, Ans: tpAnsGoods},
		{ID: "weather", Label: "the weather", Common: true, Ans: tpAnsWeather},
		{ID: "weather_season", Label: "the season", After: "weather", Common: true, Ans: tpAnsSeason},
		{ID: "rumours", Label: "what they have heard", Common: true, Ans: tpAnsRumours},
		{ID: "rumours_price", Label: "a juicier rumour", Prompt: "Ask whether they know something worth paying for.", After: "rumours", Know: []string{"merchant", "rumour"}, Avail: func(c *tpCtx) bool { return !c.child }, Run: tpRunRumourPrice},
		{ID: "self_work", Label: "their work", Common: true, Tact: 1, Ans: tpAnsWork},
		{ID: "self_family", Label: "their family", Common: true, Need: 1, Kind: "empathy", Tact: 2, Avail: func(c *tpCtx) bool { return c.p != nil }, Ans: tpAnsFamily},
		{ID: "past", Label: "the old days", Common: true, Know: []string{"elder", "past", "scholar"}, Avail: func(c *tpCtx) bool { return len(c.events()) > 0 }, Ans: tpAnsPast},
		{ID: "past_legend", Label: "the old tales", After: "past", Know: []string{"elder", "past", "scholar", "priest"}, Avail: func(c *tpCtx) bool { return tpLegendAvail(c) }, Ans: tpAnsLegend},
		{ID: "proverb", Label: "a saying of the place", Common: true, Ans: tpAnsProverb},
		{ID: "own", Label: "their own subject", Common: true, Tact: 1, Kind: "empathy",
			LabelFn: func(c *tpCtx) string { return strings.TrimPrefix(c.fill(c.voice.Topic), "their ") },
			Avail:   func(c *tpCtx) bool { return c.voice != nil && c.voice.Topic != "" }, Ans: tpAnsOwn},
		{ID: "rude_god", Label: "whether the god is real", Prompt: "Ask whether {deity} is anything more than a tale.", Common: true, Tact: -3, Avail: func(c *tpCtx) bool { return c.rel != nil && c.rel.Deity != "" }, Ans: tpAnsRudeGod},
		{ID: "rude_pay", Label: "their savings", Prompt: "Ask how much coin they have put by.", Common: true, Tact: -2, Ans: tpAnsRudePay},
		{ID: "rude_dead", Label: "their dead", Prompt: "Ask what became of their dead.", Common: true, Tact: -3, Avail: func(c *tpCtx) bool { return len(tpDeadKin(c)) > 0 }, Ans: tpAnsRudeDead},
	}
}

// ---------------------------------------------------------------- answers: town and rulers

func tpAnsTown(c *tpCtx) tpAnswer {
	g, bd := c.g, c.bd
	say := []string{fmt.Sprintf("This is %s, %s of %s.", bd.Name, tpA(tpTier(g, bd)), c.realm())}
	top, best := "", 0.0
	for i, n := range g.W.SectorNames {
		if i < len(bd.Sectors) && bd.Sectors[i] > best {
			top, best = n, bd.Sectors[i]
		}
	}
	if ph := tpSectorSay[top]; ph != "" {
		say = append(say, "Most of us live off "+ph+".")
	}
	if c.pack != nil && c.pack.Material != nil && c.pack.Material.Buildings != nil && c.pack.Material.Buildings.House != "" {
		b := c.pack.Material.Buildings
		s := "Our houses are " + lowerFirst(b.House) + "."
		if b.Hall != "" {
			s += " The hall: " + lowerFirst(b.Hall) + "."
		}
		say = append(say, s)
	} else if c.cu != nil && c.cu.Architecture != "" {
		say = append(say, "Our builders favour "+lowerFirst(c.cu.Architecture)+".")
	}
	if bd.Governor != "" {
		say = append(say, fmt.Sprintf("%s keeps the town for %s.", bd.Governor, c.realm()))
	}
	if bd.Port {
		say = append(say, "Ships tie up here, so you hear every tongue on the quay.")
	}
	if c.cu != nil && len(c.cu.Festivals) > 0 {
		say = append(say, g.W.fillFeast(c.pick(c.cu.Festivals, "town"), bd))
	}
	f := fmt.Sprintf("%s is %s of %s", bd.Name, tpA(tpTier(g, bd)), c.realm())
	if bd.Governor != "" {
		f += ", kept by " + bd.Governor
	}
	if ph := tpSectorSay[top]; ph != "" {
		f += ", and lives mostly off " + ph
	}
	return tpAnswer{Say: say, Fact: &tpFact{"event", "", f + "."}}
}

func tpAnsNotables(c *tpCtx) tpAnswer {
	w := c.g.W
	var ps []PersonData
	for _, i := range w.peopleByBrg[c.bd.ID] {
		if i >= 0 && i < len(w.People) && w.People[i].Alive {
			ps = append(ps, w.People[i])
		}
	}
	sort.SliceStable(ps, func(i, j int) bool { return ps[i].Renown > ps[j].Renown })
	if len(ps) == 0 {
		return tpAnswer{Deny: "\"Nobody of note, these days. We are plain folk.\""}
	}
	var names []string
	for i, p := range ps {
		if i >= 3 {
			break
		}
		names = append(names, fmt.Sprintf("%s, %s", p.Name, lowerFirst(nonEmpty(p.Title, p.Role))))
	}
	s := "Folk worth knowing in " + c.bd.Name + ": " + strings.Join(names, "; ") + "."
	return tpAnswer{Say: []string{s}, Fact: &tpFact{"event", "", s}}
}

func tpAnsRuler(c *tpCtx) tpAnswer {
	g, st := c.g, c.stt
	say := []string{fmt.Sprintf("%s rules %s.", st.Ruler, st.FullName)}
	if st.Capital > 0 {
		if b := g.W.burgByID[st.Capital]; b != nil {
			say = append(say, "The seat is at "+b.Name+".")
		}
	}
	if st.Ideology != "" {
		say = append(say, "Ours is a realm of the "+st.Ideology+" sort, whatever the ballads say.")
	}
	if st.Founded > 0 {
		say = append(say, fmt.Sprintf("The realm has stood since the year %d.", st.Founded))
	}
	for i := range g.W.People {
		p := &g.W.People[i]
		if p.ID == st.RulerID && len(p.Traits) > 0 {
			t := p.Traits[:mini(2, len(p.Traits))]
			say = append(say, "Between us, they say the ruler is "+strings.Join(t, " and ")+".")
			break
		}
	}
	return tpAnswer{Say: say, Fact: &tpFact{"event", "", fmt.Sprintf("%s rules %s.", st.Ruler, st.FullName)}}
}

func tpDeadRulers(c *tpCtx) []PersonData {
	if c.stt == nil {
		return nil
	}
	var out []PersonData
	for _, p := range c.g.W.People {
		if p.State == c.stt.ID && !p.Alive && p.Died > 0 && strings.Contains(strings.ToLower(p.Role+p.Title), "r") && (p.Role == "ruler" || strings.Contains(strings.ToLower(p.Title), "ruler") || p.ID == c.stt.RulerID) {
			out = append(out, p)
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Died > out[j].Died })
	return out
}

func tpAnsRulerPast(c *tpCtx) tpAnswer {
	ds := tpDeadRulers(c)
	var say []string
	for i, p := range ds {
		if i >= 2 {
			break
		}
		s := fmt.Sprintf("Before came %s, who died in the year %d", p.Name, p.Died)
		if p.Cause != "" {
			s += " (" + p.Cause + ")"
		}
		say = append(say, s+".")
	}
	return tpAnswer{Say: say, Fact: &tpFact{"event", "", strings.Join(say, " ")}}
}

// ---------------------------------------------------------------- answers: war and law

type tpWar struct {
	foe   *StateData
	now   bool
	wars  []EventData
	peace []EventData
}

func tpWarOf(c *tpCtx) *tpWar {
	if c.stt == nil {
		return nil
	}
	g, w := c.g, c.g.W
	tw := &tpWar{}
	if foes := g.warsOf(c.stt.ID); len(foes) > 0 {
		tw.foe = w.stateByID[foes[0]]
		tw.now = tw.foe != nil
	}
	for _, e := range w.Events {
		mine := e.St == c.stt.ID || (tw.foe != nil && e.St == tw.foe.ID)
		switch e.Kind {
		case "war":
			if mine {
				tw.wars = append(tw.wars, e)
			}
		case "peace":
			if mine || strings.Contains(e.Text, c.stt.Name) {
				tw.peace = append(tw.peace, e)
			}
		}
	}
	if !tw.now && len(tw.wars) == 0 && len(tw.peace) == 0 {
		return nil
	}
	return tw
}

func tpAnsWar(c *tpCtx) tpAnswer {
	tw := tpWarOf(c)
	g := c.g
	var say []string
	fact := ""
	if tw.now {
		say = append(say, fmt.Sprintf("%s is at war with %s. The people call it %s.", c.realm(), tw.foe.Name, g.warName(c.stt.ID, tw.foe.ID)))
		fact = fmt.Sprintf("%s and %s are at war.", c.realm(), tw.foe.Name)
	} else {
		say = append(say, "We are not at war now, thank the gods, but the last fighting is not forgotten.")
	}
	if n := len(tw.wars); n > 0 {
		e := tw.wars[n-1]
		say = append(say, fmt.Sprintf("The latest I know of: %s, %s.", lowerFirst(e.Text), c.ago(e.Y)))
		if fact == "" {
			fact = fmt.Sprintf("%s fought a war; %s (year %d).", c.realm(), lowerFirst(e.Text), e.Y)
		}
	}
	if tw.now && c.bd != nil {
		say = append(say, "The garrison watches the roads, and every cart is searched.")
	}
	return tpAnswer{Say: say, Fact: &tpFact{"event", "", fact}}
}

func tpAnsWarCause(c *tpCtx) tpAnswer {
	tw := tpWarOf(c)
	var say []string
	fact := ""
	if len(tw.wars) > 0 {
		e := tw.wars[0]
		say = append(say, fmt.Sprintf("As I know it, it began in the year %d: %s.", e.Y, lowerFirst(e.Text)))
		fact = fmt.Sprintf("The fighting began in the year %d: %s.", e.Y, lowerFirst(e.Text))
	} else {
		say = append(say, "Nobody agrees who struck first; each side blames the other.")
		fact = c.realm() + ": nobody agrees who started the war."
	}
	if tw.foe != nil {
		if rel, ok := c.stt.Relations[tw.foe.ID]; ok && rel < -20 {
			say = append(say, "The two crowns have loathed each other for years. The first blow only gave the grudge a name.")
		}
		if tw.foe.Ruler != "" {
			say = append(say, tw.foe.Ruler+" will tell you it was us, of course.")
		}
	}
	return tpAnswer{Say: say, Fact: &tpFact{"event", "", fact}}
}

func tpAnsWarPeace(c *tpCtx) tpAnswer {
	tw := tpWarOf(c)
	g := c.g
	var say []string
	fact := ""
	if tw.foe != nil && g.poPeace(c.stt.ID, tw.foe.ID) {
		say = append(say, "A truce holds for now. Nobody dares to trust it.")
		fact = "A truce holds between " + c.realm() + " and " + tw.foe.Name + "."
	}
	if n := len(tw.peace); n > 0 {
		e := tw.peace[n-1]
		say = append(say, fmt.Sprintf("The last peace I heard of: %s, %s.", lowerFirst(e.Text), c.ago(e.Y)))
		if fact == "" {
			fact = fmt.Sprintf("A peace was made in the year %d: %s.", e.Y, lowerFirst(e.Text))
		}
	}
	if tw.now {
		hope := "No one speaks of peace yet."
		if rel, ok := c.stt.Relations[tw.foe.ID]; ok && rel > -20 {
			hope = "There is talk of terms, quiet talk, in the back rooms."
		}
		say = append(say, hope)
		if fact == "" {
			fact = "On the war with " + tw.foe.Name + ": " + strings.ToLower(hope)
		}
	}
	if len(say) == 0 {
		say = append(say, "The peace was kept, and most of us would keep it again.")
		fact = c.realm() + " keeps its peace."
	}
	return tpAnswer{Say: say, Fact: &tpFact{"event", "", fact}}
}

func tpAnsLaw(c *tpCtx) tpAnswer {
	var say []string
	fact := ""
	if c.cust != nil && c.cust.Law != nil {
		l := c.cust.Law
		if l.Judge != "" {
			say = append(say, "Judgement here belongs to "+lowerFirst(l.Judge)+".")
		}
		n := len(l.Entries)
		for k := 0; k < n && k < 2; k++ {
			e := l.Entries[(c.idx(n, "law")+k)%n]
			s := fmt.Sprintf("For %s: %s.", e.Crime, strings.TrimSuffix(lowerFirst(e.Punish), "."))
			if e.Note != "" {
				s += " " + e.Note
			}
			say = append(say, s)
			if fact == "" {
				fact = fmt.Sprintf("In %s, %s", c.realm(), strings.ToLower(s[:1])+s[1:])
			}
		}
		if l.Ordeal != "" {
			say = append(say, "In doubtful cases the ordeal decides: "+lowerFirst(l.Ordeal)+".")
		}
	}
	if c.stt != nil {
		l := c.g.uwLawsFor(c.stt.ID)
		var bans []string
		if l.Arms {
			bans = append(bans, "war-weapons inside the walls")
		}
		if l.Heresy {
			bans = append(bans, "forbidden books")
		}
		if l.Drugs {
			bans = append(bans, "stimulant tonics")
		}
		if l.Occult {
			bans = append(bans, "occult relics")
		}
		if l.Augment {
			bans = append(bans, "unlicensed augments")
		}
		if len(bans) > 0 {
			s := "By the statutes of " + c.realm() + " these are forbidden: " + strings.Join(bans, ", ") + "."
			say = append(say, s)
			if fact == "" {
				fact = s
			}
		}
	}
	if len(say) == 0 {
		say = append(say, "The law is what the magistrate says it is, and he says it loudly.")
		fact = "Law in " + c.realm() + " is whatever the magistrate says."
	}
	return tpAnswer{Say: say, Fact: &tpFact{"event", "", fact}}
}

func tpAnsLawCrime(c *tpCtx) tpAnswer {
	var say []string
	for _, crime := range []string{"theft", "murder"} {
		found := false
		if c.cust != nil && c.cust.Law != nil {
			for _, e := range c.cust.Law.Entries {
				if e.Crime == crime {
					say = append(say, fmt.Sprintf("A %s? %s.", map[string]string{"theft": "thief", "murder": "killer"}[crime], tpUp(strings.TrimSuffix(e.Punish, "."))))
					found = true
					break
				}
			}
		}
		if !found {
			say = append(say, fmt.Sprintf("A %s gets %s.", map[string]string{"theft": "thief", "murder": "killer"}[crime], c.pick(tpPunish[crime], "pun", crime)))
		}
	}
	return tpAnswer{Say: say, Fact: &tpFact{"event", "", "Punishments in " + c.realm() + ": " + strings.Join(say, " ")}}
}

// ---------------------------------------------------------------- answers: faith

func tpAnsFaith(c *tpCtx) tpAnswer {
	var say, narr []string
	if c.faith != nil && c.faith.Clergy != nil && c.tags["priest"] {
		if c.lvl >= 1 && c.faith.Clergy.GreetFaithful != "" {
			say = append(say, c.faith.Clergy.GreetFaithful)
		} else if c.faith.Clergy.GreetStranger != "" {
			say = append(say, c.faith.Clergy.GreetStranger)
		}
	}
	say = append(say, fmt.Sprintf("We follow the %s, and %s is who we pray to.", c.rel.Name, nonEmpty(c.rel.Deity, "the unnamed one")))
	if c.rel.Doctrine != "" {
		say = append(say, "Its teaching: "+lowerFirst(c.rel.Doctrine)+".")
	}
	if c.faith != nil {
		if c.faith.Afterlife != "" {
			say = append(say, c.faith.Afterlife)
		}
		if len(c.faith.Prayers) > 0 && c.lvl >= 1 {
			narr = append(narr, "{{K|They murmur a prayer: \""+c.pick(c.faith.Prayers, "pray")+"\"}}")
		}
	}
	return tpAnswer{Say: say, Narr: narr, Fact: &tpFact{"event", "", fmt.Sprintf("In %s the %s is kept; its god is %s.", c.realm(), c.rel.Name, nonEmpty(c.rel.Deity, "unnamed"))}}
}

func tpAnsHolyDays(c *tpCtx) tpAnswer {
	g, w := c.g, c.g.W
	type hd struct {
		name, rite string
		days       int
	}
	var list []hd
	if c.faith != nil {
		for _, h := range c.faith.HolyDays {
			d := floorMod(w.FeastDoY(FeastData{Name: h.Name, Month: h.Month}, c.cu)-g.Day()%YearDays, YearDays)
			list = append(list, hd{h.Name, h.Rite, d})
		}
	}
	sort.SliceStable(list, func(i, j int) bool { return list[i].days < list[j].days })
	if len(list) == 0 && c.bd != nil {
		if f, d := w.nextFeast(c.bd, g.Day()); f != nil {
			list = append(list, hd{f.Name, w.feastText(f, c.bd), d})
		}
	}
	if len(list) == 0 {
		return tpAnswer{Deny: "\"We keep no days apart. Every day is the gods' day, if you like.\""}
	}
	var say []string
	fact := ""
	for i, h := range list {
		if i >= 2 {
			break
		}
		d := w.DateOf(c.cu, g.Day()+h.days)
		when := fmt.Sprintf("in %d days, on the %s of %s", h.days, ordinal(d.Day), d.Name)
		if h.days == 0 {
			when = "today"
		}
		s := fmt.Sprintf("%s falls %s.", h.name, when)
		if h.rite != "" && i == 0 {
			s += " " + tpUp(strings.TrimSuffix(c.fill(h.rite), ".")) + "."
		}
		say = append(say, s)
		if fact == "" {
			fact = fmt.Sprintf("The %s of the %s falls %s.", h.name, c.rel.Name, when)
		}
	}
	if c.faith != nil && len(c.faith.Observances) > 0 && c.lvl >= 2 {
		say = append(say, c.pick(c.faith.Observances, "obs"))
	}
	return tpAnswer{Say: say, Fact: &tpFact{"event", "", fact}}
}

func tpAnsSaints(c *tpCtx) tpAnswer {
	sa := c.faith.Saints
	var say []string
	fact := ""
	for k := 0; k < len(sa) && k < 2; k++ {
		s := sa[(c.idx(len(sa), "saint")+k)%len(sa)]
		line := s.Name + ": " + s.Legend
		if s.Feast != "" {
			line += " Their day is " + s.Feast + "."
		}
		say = append(say, line)
		if fact == "" {
			fact = "The " + c.rel.Name + " honours " + line
		}
	}
	return tpAnswer{Say: say, Fact: &tpFact{"event", "", fact}}
}

func tpAnsHeresy(c *tpCtx) tpAnswer {
	hs := c.faith.Heresies
	h := hs[c.idx(len(hs), "heresy")]
	s := fmt.Sprintf("The %s: %s. The faithful call it error, and some call it worse.", h.Name, lowerFirst(h.Claim))
	return tpAnswer{Say: []string{s}, Fact: &tpFact{"event", "", "Heresy against the " + c.rel.Name + ": the " + h.Name + ", which claims " + lowerFirst(h.Claim)}}
}

// ---------------------------------------------------------------- answers: customs

func tpAnsCustoms(c *tpCtx) tpAnswer {
	var say []string
	fact := ""
	if c.cust != nil && len(c.cust.Etiquette) > 0 {
		et := c.cust.Etiquette
		for k := 0; k < len(et) && k < 3; k++ {
			e := et[(c.idx(len(et), "etiq")+k)%len(et)]
			say = append(say, e.Text)
			if fact == "" {
				fact = "Manners among the " + c.cu.Name + ": " + e.Text
			}
		}
	} else {
		cu := c.cu
		if len(cu.Values) > 0 {
			say = append(say, "Among the "+cu.Name+", we hold "+strings.Join(cu.Values[:mini(3, len(cu.Values))], ", ")+" dear.")
		}
		if cu.Food != "" {
			say = append(say, "We eat "+cu.Food+".")
		}
		if cu.Dress != "" {
			say = append(say, "We dress in "+lowerFirst(cu.Dress)+".")
		}
		if len(cu.Feasts) > 0 {
			f := cu.Feasts[c.idx(len(cu.Feasts), "feast")]
			say = append(say, "The "+f.Name+" is the day we keep best.")
		}
		fact = "The " + cu.Name + " hold " + strings.Join(cu.Values, ", ") + " dear."
	}
	if len(say) == 0 {
		return tpAnswer{Deny: "\"Customs? We do what our parents did. Ask me what is odd about it.\""}
	}
	return tpAnswer{Say: say, Fact: &tpFact{"event", "", fact}}
}

func tpAnsTaboo(c *tpCtx) tpAnswer {
	var say []string
	fact := ""
	if c.cust != nil && len(c.cust.Taboos) > 0 {
		tb := c.cust.Taboos
		for k := 0; k < len(tb) && k < 2; k++ {
			t := tb[(c.idx(len(tb), "taboo")+k)%len(tb)]
			s := t.Text
			if t.Penalty != "" && t.Penalty != "none" && c.lvl >= 1 {
				s += " The usual answer to it is " + t.Penalty + "."
			}
			say = append(say, s)
			if fact == "" {
				fact = "Forbidden among the " + c.cu.Name + ": " + t.Text
			}
		}
	} else {
		for k := 0; k < 2; k++ {
			act := tpTabooActs[(c.idx(len(tpTabooActs), "gtaboo")+k)%len(tpTabooActs)]
			say = append(say, tpTabooGeneric[act][0])
			if fact == "" {
				fact = "Forbidden among the " + c.cu.Name + ": " + tpTabooGeneric[act][0]
			}
		}
	}
	say = append(say, "Good of you to ask before you stumble over it.")
	return tpAnswer{Say: say, Fact: &tpFact{"event", "", fact}}
}

func tpHosp(c *tpCtx) (days int, offers, duties []string, breach string) {
	days, offers, duties = 3, tpGuestGeneric, tpGuestDuties
	if c.cu != nil && c.cu.Food != "" {
		offers = append([]string{"a bowl of " + strings.Split(c.cu.Food, ",")[0]}, offers...)
	}
	if c.cust != nil && c.cust.Hospitality != nil {
		h := c.cust.Hospitality
		if h.GuestRight > 0 {
			days = h.GuestRight
		}
		if len(h.Offers) > 0 {
			offers = h.Offers
		}
		if len(h.Duties) > 0 {
			duties = h.Duties
		}
		breach = h.Breach
	}
	return
}

func tpAnsGuest(c *tpCtx) tpAnswer {
	days, offers, duties, breach := tpHosp(c)
	say := []string{fmt.Sprintf("A guest is under the roof's protection for %d days.", days),
		"We offer " + lowerFirst(c.pick(offers, "offer")) + ".",
		"In return a guest must " + strings.TrimSuffix(lowerFirst(c.pick(duties, "duty")), ".") + "."}
	if breach != "" {
		say = append(say, tpUp(strings.TrimSuffix(breach, "."))+".")
	}
	return tpAnswer{Say: say, Fact: &tpFact{"event", "", fmt.Sprintf("Among the %s a guest is protected for %d days under the roof.", c.cu.Name, days)}}
}

func tpRiteAvail(c *tpCtx) bool {
	return (c.cust != nil && len(c.cust.Rites) > 0) || (c.cu != nil && (len(c.cu.Feasts) > 0 || len(c.cu.Festivals) > 0))
}

func tpAnsRites(c *tpCtx) tpAnswer {
	if c.cust != nil && len(c.cust.Rites) > 0 {
		r := c.cust.Rites[c.idx(len(c.cust.Rites), "rite")]
		say := []string{"The " + r.ID + " rite is called " + r.Name + "."}
		for i, s := range r.Steps {
			if i >= 2 {
				break
			}
			say = append(say, s)
		}
		if len(r.Gifts) > 0 {
			say = append(say, "Gifts: "+strings.Join(r.Gifts[:mini(2, len(r.Gifts))], ", ")+".")
		}
		return tpAnswer{Say: say, Fact: &tpFact{"event", "", "Rite of the " + c.cu.Name + ": " + r.Name + " (" + r.ID + ")."}}
	}
	var say []string
	if len(c.cu.Feasts) > 0 {
		f := c.cu.Feasts[c.idx(len(c.cu.Feasts), "feast")]
		say = append(say, "At the "+f.Name+", "+lowerFirst(c.g.W.fillFeast(nonEmpty(f.Text, "all of us gather and eat together"), nonBurg(c.bd))))
	}
	if len(c.cu.Festivals) > 0 {
		say = append(say, c.g.W.fillFeast(c.pick(c.cu.Festivals, "fest"), nonBurg(c.bd)))
	}
	return tpAnswer{Say: say}
}

func nonBurg(b *BurgData) *BurgData {
	if b != nil {
		return b
	}
	return &BurgData{Name: "the town"}
}

// ---------------------------------------------------------------- answers: road, trade, weather

func tpAnsRoad(c *tpCtx) tpAnswer {
	g, w := c.g, c.g.W
	var say []string
	p := w.Idx(g.WX, g.WY)
	for k, ri := range w.nmRoutes(p, "road") {
		if k >= 1 {
			break
		}
		r := &w.nmData().Routes[ri]
		a, b := w.burgByID[r.A], w.burgByID[r.B]
		if a != nil && b != nil {
			say = append(say, fmt.Sprintf("The %s runs through here, from %s to %s.", r.Name, a.Name, b.Name))
		}
	}
	type nb struct {
		b      *BurgData
		d      int
		dx, dy int
	}
	var ns []nb
	for i := range w.Burgs {
		b := &w.Burgs[i]
		if !b.Alive || b.Layer != "surface" || (c.bd != nil && b.ID == c.bd.ID) {
			continue
		}
		x, y := tpAt(w, b)
		d := abs(x-g.WX) + abs(y-g.WY)
		if d > 0 && d <= 18 {
			ns = append(ns, nb{b, d, x - g.WX, y - g.WY})
		}
	}
	sort.SliceStable(ns, func(i, j int) bool { return ns[i].d < ns[j].d })
	fact := ""
	for i, n := range ns {
		if i >= 3 {
			break
		}
		km := float64(n.d) * g.tvKm()
		days := maxi(1, int(km/28+0.5))
		s := fmt.Sprintf("%s lies %s, some %d days on foot.", n.b.Name, arCompass(n.dx, n.dy), days)
		say = append(say, s)
		if fact == "" {
			fact = s
		}
	}
	if len(say) == 0 {
		return tpAnswer{Deny: "\"Nothing near. The nearest roof is a long walk from here.\""}
	}
	return tpAnswer{Say: say, Fact: &tpFact{"event", "", fact}}
}

func tpAnsNeighbours(c *tpCtx) tpAnswer {
	var friends, foes []string
	for id, v := range c.stt.Relations {
		s := c.g.W.stateByID[id]
		if s == nil || !s.Alive || id == c.stt.ID {
			continue
		}
		switch {
		case v >= 30:
			friends = append(friends, s.Name)
		case v <= -30:
			foes = append(foes, s.Name)
		}
	}
	sort.Strings(friends)
	sort.Strings(foes)
	var say []string
	if len(friends) > 0 {
		say = append(say, "Friends of "+c.realm()+": "+strings.Join(friends[:mini(3, len(friends))], ", ")+".")
	}
	if len(foes) > 0 {
		say = append(say, "No friends there: "+strings.Join(foes[:mini(3, len(foes))], ", ")+".")
	}
	if len(say) == 0 {
		return tpAnswer{Say: []string{"Our neighbours leave us be and we leave them be."}}
	}
	return tpAnswer{Say: say, Fact: &tpFact{"event", "", strings.Join(say, " ")}}
}

func tpOutlawsNear(c *tpCtx) []OutlawData {
	w, g := c.g.W, c.g
	var out []OutlawData
	for _, o := range w.Outlaws {
		x, y := o.P%w.W, o.P/w.W
		if abs(x-g.WX)+abs(y-g.WY) <= 14 {
			out = append(out, o)
		}
	}
	return out
}

func tpAnsDanger(c *tpCtx) tpAnswer {
	os := tpOutlawsNear(c)
	w, g := c.g.W, c.g
	o := os[c.idx(len(os), "outlaw")]
	x, y := o.P%w.W, o.P/w.W
	kind := "band of bandits"
	if o.Kind == "pirate" {
		kind = "pirate crew"
	}
	s := fmt.Sprintf("%s, a %s, works the country %s of here, near %s.", o.Name, kind, arCompass(x-g.WX, y-g.WY), w.RegionName(o.P))
	return tpAnswer{Say: []string{s, "Travel in numbers, or travel by day, or both."}, Fact: &tpFact{"event", "", s}}
}

func tpAnsTrade(c *tpCtx) tpAnswer {
	g := c.g
	bd := c.bd
	if tb := g.tqBurg(); tb != nil && tb.ID == bd.ID {
		type gr struct {
			d *tqGood
			r float64
		}
		var gs []gr
		for _, d := range g.tqGoodsHere() {
			if !g.tqCanBuy(bd, d) {
				continue
			}
			gs = append(gs, gr{d, float64(g.tqPrice(bd, d, true)) / (float64(d.Base) * 1.1)})
		}
		sort.SliceStable(gs, func(i, j int) bool { return gs[i].r > gs[j].r })
		var dear, cheap []string
		for i, e := range gs {
			if i < 2 && e.r > 1.12 {
				dear = append(dear, e.d.Name)
			}
		}
		for i := len(gs) - 1; i >= 0 && len(cheap) < 2; i-- {
			if gs[i].r < 0.88 {
				cheap = append(cheap, gs[i].d.Name)
			}
		}
		var say []string
		fact := ""
		if len(dear) > 0 {
			say = append(say, "Dear in this market just now: "+strings.Join(dear, " and ")+".")
			fact = "In " + bd.Name + " " + strings.Join(dear, " and ") + " sell dear"
		}
		if len(cheap) > 0 {
			say = append(say, "Cheap: "+strings.Join(cheap, " and ")+".")
			if fact == "" {
				fact = "In " + bd.Name + " " + strings.Join(cheap, " and ") + " sell cheap"
			} else {
				fact += ", while " + strings.Join(cheap, " and ") + " go cheap"
			}
		}
		if len(say) > 0 {
			say = append(say, "Buy where it is cheap, sell where it is dear. That is the whole of trade, and nobody follows it.")
			return tpAnswer{Say: say, Fact: &tpFact{"event", "", fact + "."}}
		}
		return tpAnswer{Say: []string{"Prices are steady at the moment. Nothing to make a man rich, nothing to ruin one."}}
	}
	// outside a town zone: speak from the town's sectors
	var top []string
	for i, n := range g.W.SectorNames {
		if i < len(bd.Sectors) && bd.Sectors[i] >= 0.17 && tpSectorSay[n] != "" {
			top = append(top, tpSectorSay[n])
		}
	}
	if len(top) == 0 {
		return tpAnswer{Say: []string{"We trade a little of everything and get rich on none of it."}}
	}
	s := bd.Name + " lives off " + strings.Join(top, " and ") + ", so what comes from those is cheap here."
	return tpAnswer{Say: []string{s}, Fact: &tpFact{"event", "", s}}
}

func tpAnsGoods(c *tpCtx) tpAnswer {
	var say []string
	fact := ""
	if c.pack != nil && c.pack.Material != nil {
		m := c.pack.Material
		if len(m.Crafts) > 0 {
			cr := m.Crafts[c.idx(len(m.Crafts), "craft")]
			s := fmt.Sprintf("Our craft of %s: %s.", cr.Name, strings.Join(cr.Goods[:mini(3, len(cr.Goods))], ", "))
			if cr.Mark != "" {
				s += " Our mark is " + lowerFirst(cr.Mark) + "."
			}
			say = append(say, s)
			fact = "The " + c.cu.Name + " are known for " + cr.Name + "."
		}
		if len(m.TradeGoods) > 0 {
			var ns []string
			for i, g := range m.TradeGoods {
				if i >= 3 {
					break
				}
				ns = append(ns, g.Name)
			}
			say = append(say, "Merchants carry away "+strings.Join(ns, ", ")+".")
		}
	}
	if len(say) == 0 {
		if c.cu.Food != "" {
			say = append(say, "We are known for our table: "+c.cu.Food+".")
		}
		if c.cu.Dress != "" {
			say = append(say, "And for our dress: "+lowerFirst(c.cu.Dress)+".")
		}
		fact = "The " + c.cu.Name + " are known for " + nonEmpty(c.cu.Food, "their table") + "."
	}
	if len(say) == 0 {
		return tpAnswer{Deny: "\"Nothing special. Honest goods, honestly priced.\""}
	}
	return tpAnswer{Say: say, Fact: &tpFact{"event", "", fact}}
}

func tpAnsWeather(c *tpCtx) tpAnswer {
	g := c.g
	w := g.WeatherHere()
	season := strings.ToLower(SeasonNames[g.Season()])
	say := []string{fmt.Sprintf("It is %s, and %s.", season, strings.ToLower(Plain(tempWord(w.Temp))))}
	if w.Kind != nil {
		if l := tpWeatherSay[w.Kind.ID]; len(l) > 0 {
			say = append(say, c.pick(l, "wx", w.Kind.ID, fmt.Sprint(g.Day())))
		} else {
			say = append(say, "The weather is "+strings.ToLower(w.Kind.Name)+", as you can see for yourself.")
		}
	}
	return tpAnswer{Say: say}
}

func tpAnsSeason(c *tpCtx) tpAnswer {
	g := c.g
	say := []string{c.pick(tpSeasonSay[g.Season()], "season")}
	d := g.W.DateOf(c.cu, g.Day())
	say = append(say, fmt.Sprintf("By our count it is the month of %s.", d.Name))
	fact := ""
	if c.bd != nil {
		if f, days := g.W.nextFeast(c.bd, g.Day()); f != nil && days <= 40 {
			say = append(say, fmt.Sprintf("The %s is %d days off.", f.Name, days))
			fact = fmt.Sprintf("The %s is kept at %s in %d days.", f.Name, c.bd.Name, days)
		}
	}
	a := tpAnswer{Say: say}
	if fact != "" {
		a.Fact = &tpFact{"event", "", fact}
	}
	return a
}

// ---------------------------------------------------------------- answers: rumours, self, past

func tpAnsRumours(c *tpCtx) tpAnswer {
	g := c.g
	p := c.who()
	var say []string
	facts := g.knSpeakable(p, maxi(0, c.lvl), false)
	st := knSt(g)
	learned := map[int]bool{}
	for _, l := range st.Learned {
		if l.From == p.Key {
			learned[l.Fact] = true
		}
	}
	r := NewRng(fmt.Sprint(g.Seed, ":tp:rum:", c.key, ":", g.Day()))
	n := 0
	for _, f := range facts {
		if f.Subject == "player" || (f.Kind == "location" && c.lvl < 1) || learned[f.ID] || n >= 1+c.lvl {
			continue
		}
		said := g.knSpeak(p, f, false)
		g.knLearn(said, p, c.lvl, "ask")
		say = append(say, fmt.Sprintf("%s %s.", Pick(r, knOpeners), lowerFirst(said.Text)))
		n++
	}
	if len(say) == 0 {
		ev := c.events()
		if len(ev) == 0 {
			return tpAnswer{Deny: Pick(NewRng(c.key+fmt.Sprint(g.Day())), knNothing)}
		}
		e := g.W.Events[ev[c.idx(mini(len(ev), 6), "rumour", fmt.Sprint(g.Day()/7))]]
		s := fmt.Sprintf("They say that %s, %s.", lowerFirst(e.Text), c.ago(e.Y))
		return tpAnswer{Say: []string{s}, Fact: &tpFact{"event", "", fmt.Sprintf("Folk say that %s (year %d).", lowerFirst(e.Text), e.Y)}}
	}
	return tpAnswer{Say: say}
}

func tpAnsWork(c *tpCtx) tpAnswer {
	g := c.g
	role := nonEmpty(c.a.Title, c.a.Role)
	var say []string
	if c.p != nil {
		role = lvTitleOf(c.p)
		say = append(say, fmt.Sprintf("I am %s.", lvAn(role)))
		for _, e := range c.p.Life {
			if e.Kind == "job" {
				say = append(say, fmt.Sprintf("I took up the work in the year %d and have kept at it.", e.Y))
				break
			}
		}
		if c.p.Goal != "" && c.lvl >= 1 {
			say = append(say, "Mostly, I want to "+c.p.Goal+".")
		}
	} else {
		say = append(say, fmt.Sprintf("I am %s, and the work keeps me fed.", tpA(role)))
	}
	if c.vp != nil {
		if bk := tpBarkFor[c.a.Role]; bk != "" && len(c.vp.Barks[bk]) > 0 {
			say = append(say, c.vp.Barks[bk][c.idx(len(c.vp.Barks[bk]), "bark", fmt.Sprint(g.Day()))])
		}
	}
	return tpAnswer{Say: say}
}

func tpAnsFamily(c *tpCtx) tpAnswer {
	lead := map[string]string{"warm": "Family? Gladly.", "proud": "My house, you mean.", "nervous": "Family. Well. All right.", "gruff": "Family. Hm."}[c.tone]
	return tpAnswer{Say: []string{lead}, Narr: []string{lvFamilyTalk(c.g, c.p)}}
}

func tpDeadKin(c *tpCtx) []*lvPerson {
	if c.p == nil {
		return nil
	}
	var out []*lvPerson
	for _, k := range c.p.Kin {
		if q := lvGet(c.g, k); q != nil && !q.Alive {
			out = append(out, q)
		}
	}
	return out
}

func tpAnsPast(c *tpCtx) tpAnswer {
	ev := c.events()
	e := c.g.W.Events[ev[c.idx(mini(len(ev), 8), "past")]]
	say := []string{fmt.Sprintf("I remember the tale of the year %d: %s.", e.Y, lowerFirst(e.Text)), "That was " + c.ago(e.Y) + "."}
	return tpAnswer{Say: say, Fact: &tpFact{"event", "", fmt.Sprintf("In the year %d, %s.", e.Y, lowerFirst(e.Text))}}
}

func tpLegendAvail(c *tpCtx) bool {
	return (c.pack != nil && c.pack.Literature != nil && len(c.pack.Literature.Works) > 0) || len(c.events()) > 1
}

func tpAnsLegend(c *tpCtx) tpAnswer {
	g := c.g
	if c.pack != nil && c.pack.Literature != nil && len(c.pack.Literature.Works) > 0 {
		ws := c.pack.Literature.Works
		wk := ws[c.idx(len(ws), "work")]
		say := []string{fmt.Sprintf("There is %s, \"%s\", by %s.", tpA(wk.Form), wk.Title, wk.Author)}
		if wk.Event != nil && *wk.Event >= 0 && *wk.Event < len(g.W.Events) {
			e := g.W.Events[*wk.Event]
			say = append(say, fmt.Sprintf("It tells of the year %d: %s.", e.Y, lowerFirst(e.Text)))
		} else {
			say = append(say, "It is a legend, older than the count of years, and nobody can prove it either way.")
		}
		var narr []string
		for i, l := range wk.Lines {
			if i >= 1 && i <= 2 {
				narr = append(narr, "{{K|\""+l+"\"}}")
			}
		}
		a := tpAnswer{Say: say, Narr: narr}
		if wk.Reveals != "" {
			a.Fact = &tpFact{"event", "", wk.Reveals}
		}
		return a
	}
	ev := c.events()
	e := g.W.Events[ev[1+c.idx(mini(len(ev)-1, 7), "legend")]]
	return tpAnswer{Say: []string{fmt.Sprintf("The old men tell of the year %d: %s.", e.Y, lowerFirst(e.Text))}, Fact: &tpFact{"event", "", fmt.Sprintf("In the year %d, %s.", e.Y, lowerFirst(e.Text))}}
}

func tpAnsProverb(c *tpCtx) tpAnswer {
	var say, narr []string
	if c.vp != nil && len(c.vp.Proverbs) > 0 {
		say = append(say, c.pick(c.vp.Proverbs, "prov"))
	} else {
		say = append(say, c.pick(tpProverbs, "prov"))
	}
	if c.vp != nil && len(c.vp.Idioms) > 0 && c.lvl >= 1 {
		i := c.vp.Idioms[c.idx(len(c.vp.Idioms), "idiom")]
		narr = append(narr, "{{K|Another turn of phrase: \""+i.Say+"\", which means "+lowerFirst(i.Meaning)+".}}")
	}
	return tpAnswer{Say: say, Narr: narr}
}

func tpAnsOwn(c *tpCtx) tpAnswer {
	lead := map[string]string{"warm": "You want to hear about that? Gladly.", "proud": "A subject worth my breath, at last.", "nervous": "Oh. You are interested? Truly?", "gruff": "Hah. Well, if you must."}[c.tone]
	say := []string{lead}
	if len(c.voice.Beliefs) > 0 {
		say = append(say, c.voice.Beliefs[c.idx(len(c.voice.Beliefs), "belief")])
	}
	var narr []string
	if len(c.voice.Quirks) > 0 {
		narr = append(narr, "{{K|"+c.voice.Quirks[c.idx(len(c.voice.Quirks), "quirk")]+"}}")
	}
	return tpAnswer{Say: say, Narr: narr}
}

// ---------------------------------------------------------------- answers: tactless questions

func tpAnsRudeGod(c *tpCtx) tpAnswer {
	_, rebuke, sev, ok := c.taboo("blaspheme")
	if !ok {
		rebuke, sev = c.pick(tpTactless[c.tone], "god"), 2
	}
	say := []string{rebuke}
	a := tpAnswer{Say: say}
	if sev >= 3 {
		a.Rep = -3
	}
	if c.lvl >= 2 && c.faith != nil && len(c.faith.Prayers) > 0 {
		a.Say = append(a.Say, "Still. You are a friend, so: "+c.pick(c.faith.Prayers, "rg"))
	}
	return a
}

func tpAnsRudePay(c *tpCtx) tpAnswer {
	say := []string{c.pick(tpTactless[c.tone], "pay")}
	if c.p != nil && c.lvl >= 1 {
		w := "less than you would think"
		if c.p.Wealth > 200 {
			w = "more than I let on"
		}
		say = append(say, "Since you must know: "+w+".")
	}
	return tpAnswer{Say: say}
}

func tpAnsRudeDead(c *tpCtx) tpAnswer {
	ds := tpDeadKin(c)
	say := []string{c.pick(tpTactless[c.tone], "dead")}
	if c.lvl >= 1 && len(ds) > 0 {
		say = append(say, fmt.Sprintf("I lost %s. That is all I will say.", ds[0].Name))
	}
	return tpAnswer{Say: say, Grief: true}
}

// ---------------------------------------------------------------- tpChoice: branching helper

// tpBranch is one choice in a branching conversation. Check is a checks.go kind ("" = none); WinFn, when set,
// builds the success text (so it can reveal things); WinCost is paid only on success.
type tpBranch struct {
	Text          string
	Check         string
	DC            int
	Cost          int
	WinCost       int
	Win, Lose     string
	WinFn         func() string
	OnWin, OnLose func()
	Next          func(win bool)
	Hide          func() bool
}

// tpChoice shows prompt with the branches as options (quest talk can reuse it). Esc and "Never mind" return to the talk menu.
func (g *Game) tpChoice(prompt string, branches []tpBranch) { g.tpChoiceBack(prompt, branches, nil) }

func (g *Game) tpChoiceBack(prompt string, branches []tpBranch, back func()) {
	if g.Talk == nil {
		return
	}
	if back == nil {
		back = func() { g.tpBackToTalk() }
	}
	var opts []Option
	for _, b := range branches {
		b := b
		if b.Hide != nil && b.Hide() {
			continue
		}
		txt := b.Text
		if b.Cost > 0 {
			txt += " {{K|(" + g.cuFormat(b.Cost) + ")}}"
		}
		if b.Check != "" {
			dc := b.DC
			if dc == 0 {
				dc = dcMedium
			}
			txt = g.checkLabel(b.Check, dc) + " " + txt
		}
		opts = append(opts, Option{txt, func() { g.tpResolve(b, back) }})
	}
	opts = append(opts, Option{"{{K|Never mind.}}", back})
	g.Popup = Wrap(prompt, 100)
	g.Opts = opts
	g.Sel = 0
	delete(th2Back, g)
	th2Back[g] = &th2Sub{first: &g.Opts[0], n: len(g.Opts), back: back}
}

func (g *Game) tpResolve(b tpBranch, back func()) {
	done := func(text string, win bool) {
		if g.Mode != "talk" || g.Talk == nil {
			return
		}
		g.Popup = Wrap(text, 100)
		next := back
		if b.Next != nil {
			next = func() { b.Next(win) }
		}
		g.Opts = []Option{{"Go on.", next}}
		g.Sel = 0
		delete(th2Back, g)
		th2Back[g] = &th2Sub{first: &g.Opts[0], n: 1, back: next}
	}
	if b.Cost > 0 && !g.cuSpend(b.Cost) {
		done("{{y|You cannot afford it.}}", false)
		return
	}
	win, tag := true, ""
	if b.Check != "" {
		dc := b.DC
		if dc == 0 {
			dc = dcMedium
		}
		lab := checkKinds[b.Check].label
		win = g.roll2d6Check(g.checkMod(b.Check), dc)
		if win {
			g.gainXP(4 + dc)
			tag = "{{G|[" + lab + ": Success]}}\n\n"
		} else {
			tag = "{{R|[" + lab + ": Failure]}}\n\n"
		}
	}
	if win {
		if b.WinCost > 0 && !g.cuSpend(b.WinCost) {
			done(tag+"You carry the argument, but you cannot cover the price.", false)
			return
		}
		text := b.Win
		if b.WinFn != nil {
			text = b.WinFn()
		}
		if b.OnWin != nil {
			b.OnWin()
		}
		done(tag+text, true)
		return
	}
	if b.OnLose != nil {
		b.OnLose()
	}
	done(tag+b.Lose, false)
}

// ---------------------------------------------------------------- sample 1: a stranger asks for lodging

func (g *Game) tpGuestSleep() {
	g.ijOnHomeSleep()
	g.spendTurns(480)
	if g.Dead {
		return
	}
	p := g.P
	p.HP = p.MaxHP
	delete(p.Status, "bleeding")
	delete(p.Status, "poisoned")
	g.Hunger, g.Thirst = 0, 0
	g.Msg("{{G|You sleep eight hours under a stranger's roof and wake rested.}}")
}

func tpRunLodging(c *tpCtx, back func()) {
	g := c.g
	st := tpSt(g)
	k := c.key + "|lodge"
	day := g.Day()
	deny := func(s string) { g.tpShow(c, s, []Option{{"{{K|Back.}}", back}}, back) }
	if r := st.Asked[k]; r != nil && r.Day == day && r.N > 1 {
		deny("\"You have had your bed today. Come back tomorrow, guest.\"")
		return
	}
	if g.nearHostile() {
		deny("\"Not with trouble at your heels. Not under my roof.\"")
		return
	}
	if (c.p != nil && c.p.Opinion < -10) || g.Rep[c.a.Faction] <= -80 {
		deny("\"Not under my roof. Find a ditch.\"")
		return
	}
	days, offers, duties, breach := tpHosp(c)
	offer := lowerFirst(c.fill(c.pick(offers, "lodge", fmt.Sprint(day))))
	duty := strings.TrimSuffix(lowerFirst(c.fill(c.pick(duties, "duty", fmt.Sprint(day)))), ".")
	prompt := fmt.Sprintf("\"A bed? Under my roof a guest is protected for %d days. I can give you %s. In return, a guest must %s.\"", days, offer, duty)
	if breach != "" && c.lvl >= 1 {
		prompt += "\n\n\"" + tpUp(strings.TrimSuffix(c.fill(breach), ".")) + ".\""
	}
	shy := c.lvl <= 0 && g.Rep[c.a.Faction] < 0
	_, rebuke, sev, _ := c.taboo("refuse_hospitality", "eat_alone", "refuse_gift")
	if rebuke == "" {
		rebuke, sev = tpTabooGeneric["refuse_hospitality"][1], 2
	}
	_, _, _, payInsult := c.taboo("refuse_gift", "haggle_sacred")
	if c.cust != nil && c.cust.Ideals != nil {
		h := strings.ToLower(c.cust.Ideals.Host + c.cust.Ideals.Wealth)
		if strings.Contains(h, "coin") || strings.Contains(h, "gold") || strings.Contains(h, "pay") {
			payInsult = true
		}
	}
	good := func() {
		c.op("help", "kept the customs of a guest in my house", 3)
		g.tpGuestSleep()
	}
	accept := tpBranch{
		Text:   "Accept the offer, and keep the customs of a guest.",
		Win:    "You take your place by the hearth. You take " + offer + " as it is given, you " + duty + ", and you sleep soundly. In the morning the host nods: \"A good guest. You will be welcome again.\"",
		Lose:   "Your host weighs you. \"A stranger with no one to vouch for... no. I am sorry. Not tonight.\"",
		OnWin:  good,
		OnLose: func() { c.op("insult", "asked for a bed and was turned away", -1) },
	}
	if shy {
		accept.Check, accept.DC = "persuasion", dcMedium
		accept.Text = "Give your name and your road, and ask to be vouched for."
	}
	pay := tpBranch{Text: "Offer coin for the bed instead.", Cost: 4 + g.P.Level}
	if payInsult {
		pay.Cost = 0
		pay.Win = "Your host stares at the coin as if it were a dead mouse. \"A guest is not a customer. Put that away, or find another roof.\""
		pay.OnWin = func() { c.op("insult", "tried to pay for a guest's bed", -4) }
	} else {
		pay.Win = "\"Fair enough.\" The coin disappears. A pallet in the loft, a blanket, a bowl of something hot. A bed is a bed, though they are less warm about it than a guest would have found them."
		pay.OnWin = func() { g.tpGuestSleep() }
	}
	refuse := tpBranch{
		Text:  "Wave the food away: you only want the bed.",
		Win:   "Your host's face closes. \"" + c.fill(rebuke) + "\"",
		OnWin: func() { c.op("insult", "refused the cup in my house", -2-sev) },
	}
	leave := tpBranch{Text: "Thank them and move on.", Win: "\"The road is long. May it be kind.\"", Next: func(bool) { back() }}
	g.tpChoiceBack(prompt, []tpBranch{accept, pay, refuse, leave}, back)
}

// ---------------------------------------------------------------- sample 2: haggling over a rumour's price

func (g *Game) tpPickRumour(c *tpCtx) *knFact {
	p := c.who()
	st := knSt(g)
	learned := map[int]bool{}
	for _, l := range st.Learned {
		if l.From == p.Key {
			learned[l.Fact] = true
		}
	}
	for _, f := range g.knSpeakable(p, 2, false) {
		if f.Subject == "player" || learned[f.ID] {
			continue
		}
		switch f.Kind {
		case "rumour", "secret", "event", "sighting":
			return f
		}
	}
	ev := c.events()
	if len(ev) == 0 {
		return nil
	}
	n := 0
	for k := range tpSt(g).Facts {
		if strings.HasPrefix(k, "rum:"+c.key) {
			n++
		}
	}
	e := g.W.Events[ev[c.idx(len(ev), "rum", fmt.Sprint(n))]]
	return g.tpReveal(c, fmt.Sprintf("rum:%s:%d", c.key, n), &tpFact{"rumour", "", fmt.Sprintf("Folk whisper that %s (year %d).", lowerFirst(e.Text), e.Y)})
}

func tpRunRumourPrice(c *tpCtx, back func()) {
	g := c.g
	f := g.tpPickRumour(c)
	if f == nil {
		g.tpShow(c, "\"Nothing worth buying. Truly.\"", []Option{{"{{K|Back.}}", back}}, back)
		return
	}
	p := c.who()
	base := knPrice(g) * 3
	if c.tags["merchant"] {
		base = base * 5 / 4
	}
	base = base * (10 - c.lvl) / 10
	learn := func(via string) func() string {
		return func() string {
			said := g.knSpeak(p, f, via == "paid")
			g.knLearn(said, p, c.lvl, via)
			return fmt.Sprintf("They lean in and say it low: \"%s.\"", strings.TrimSuffix(said.Text, "."))
		}
	}
	var open func(price int, tried map[string]bool)
	open = func(price int, tried map[string]bool) {
		ask := map[string]string{
			"warm":    "\"Oh, I know something worth the hearing. My memory is dear, though: %s.\"",
			"proud":   "\"What I know is not given away. %s, and I do not bargain.\"",
			"nervous": "\"I... might know something. It is risky to say. %s. Please.\"",
			"gruff":   "\"Knowledge costs. %s. Take it or leave it.\"",
		}[c.tone]
		prompt := fmt.Sprintf(ask, g.cuFormat(price))
		again := func(extra int) func(bool) {
			return func(win bool) {
				if win {
					back()
					return
				}
				t := map[string]bool{}
				for k, v := range tried {
					t[k] = v
				}
				open(price+extra, t)
			}
		}
		mk := func(id string, b tpBranch, extra int) tpBranch {
			b.Hide = func() bool { return tried[id] }
			b.Next = func(win bool) {
				tried[id] = true
				again(extra)(win)
			}
			return b
		}
		pay := tpBranch{Text: "Pay the asking price.", Cost: price, WinFn: learn("paid"), OnWin: func() { c.op("trade", "paid fairly for what I knew", 1) }, Next: func(bool) { back() }}
		haggle := mk("haggle", tpBranch{
			Text: "Haggle: it is hearsay, and you know it.", Check: "persuasion", DC: dcMedium, WinCost: price / 2, WinFn: learn("paid"),
			Lose:   "Their face shuts like a purse. \"Haggle with me, will you? The price just went up.\"",
			OnLose: func() { c.op("insult", "haggled over what I know", -2) },
		}, price/2)
		bluff := mk("bluff", tpBranch{
			Text: "Claim you already know most of it.", Check: "bluff", DC: dcChallenging, WinFn: learn("ask"),
			Lose:   "They laugh. \"Then you will not mind paying more to hear the rest.\"",
			OnLose: func() { c.op("insult", "tried to cheat me out of a rumour", -2) },
		}, price/2)
		press := mk("press", tpBranch{
			Text: "{{R|Lean on them until they talk.}}", Check: "intimidate", DC: dcChallenging, WinFn: learn("pressed"),
			OnWin:  func() { c.op("insult", "bullied me into talking", -6) },
			Lose:   "They square up to you. \"Try that again and I call the watch.\"",
			OnLose: func() { c.op("insult", "threatened me", -8) },
		}, price)
		walk := tpBranch{Text: "Walk away.", Win: "\"Suit yourself.\"", Next: func(bool) { back() }}
		g.tpChoiceBack(prompt, []tpBranch{pay, haggle, bluff, press, walk}, back)
	}
	open(base, map[string]bool{})
}
