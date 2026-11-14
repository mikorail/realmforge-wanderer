package engine

// Quests that people give you (prefix nq). A person in a town, when you talk to them, may have a request
// that grows out of their real situation in the life simulation: their goal, family, rivals, debts, jailed or
// missing relatives, sick kin, vacant shops, crimes at large. Requests are built deterministically from that
// state (side rng seed:nq:..., never g.R) by ~28 templates. Accepted quests are mirrored into g.Quests as Kind
// "npcq" (inert for the core quest code, shown in the journal) and run through steps: talk to a named person,
// pay, hand something over, stay in a town for a few hours, find an item in the wilds, kill a named rival.
// Outcomes change the simulation (people come home, leave jail, marry, die) and write life lines and news;
// a lapsed deadline lets the situation resolve on its own, usually badly. State g.ext("nq"), planet-local.

import (
	"fmt"
	"strconv"
	"strings"
)

type nqStep struct {
	K     string `json:"k"`           // talk pay item stay place kill
	Who   string `json:"w,omitempty"` // person key
	Role  string `json:"r,omitempty"` // or any actor of these roles ("a|b"), "*" = any person of the burg
	Burg  int    `json:"b,omitempty"`
	Opt   string `json:"o,omitempty"`
	Say   string `json:"s,omitempty"`
	Check string `json:"c,omitempty"`
	DC    int    `json:"d,omitempty"`
	Fail  string `json:"f,omitempty"`
	Cost  int    `json:"$,omitempty"`
	Need  string `json:"n,omitempty"` // obj (the quest's item) or herb
	Eff   string `json:"e,omitempty"`
	Mins  int    `json:"m,omitempty"`
	Tries int    `json:"t,omitempty"`
	Done  bool   `json:"x,omitempty"`
}

type nqQuest struct {
	ID      int      `json:"i"`
	Tpl     string   `json:"tp"`
	Title   string   `json:"t"`
	Offer   string   `json:"o"`
	Giver   string   `json:"g"`
	GBurg   int      `json:"gb"`
	Subj    string   `json:"s,omitempty"`
	Subj2   string   `json:"s2,omitempty"`
	Steps   []nqStep `json:"st"`
	Cur     int      `json:"c"`
	Stage   int      `json:"sg"` // 1 active, 3 done, 4 failed
	Start   int      `json:"sd"`
	Due     int      `json:"du"`
	Grace   int      `json:"gr,omitempty"`
	Coin    int      `json:"co,omitempty"`
	XP      int      `json:"xp,omitempty"`
	Rep     int      `json:"rp,omitempty"`
	Op      int      `json:"op,omitempty"`
	Extra   string   `json:"ex,omitempty"` // item secret favour
	Num     int      `json:"n,omitempty"`
	Item    string   `json:"it,omitempty"`
	Place   string   `json:"pl,omitempty"` // heir book captive
	Zone    string   `json:"z,omitempty"`
	TP      int      `json:"tpar,omitempty"`
	Fx      string   `json:"fx,omitempty"` // effect when the deadline lapses
	Placed  bool     `json:"pd,omitempty"`
	NPC     int      `json:"npc,omitempty"`
	Foe     int      `json:"foe,omitempty"`
	FoeDead bool     `json:"fd,omitempty"`
	Spawned bool     `json:"sp,omitempty"`
	Mins    int      `json:"mn,omitempty"`
	Opened  bool     `json:"op2,omitempty"`
	Late    bool     `json:"la,omitempty"`
	Result  string   `json:"rs,omitempty"`
}

func (q *nqQuest) cur() *nqStep {
	if q.Stage != 1 || q.Cur < 0 || q.Cur >= len(q.Steps) {
		return nil
	}
	return &q.Steps[q.Cur]
}

type nqState struct {
	Q        []*nqQuest      `json:"q,omitempty"`
	Next     map[string]int  `json:"n,omitempty"`  // person -> no new request before this day
	BurgNext map[int]int     `json:"bn,omitempty"` // burg -> no new request before this day
	Favour   map[string]bool `json:"fv,omitempty"`
	Firsts   map[string]bool `json:"fi,omitempty"`
	Done     int             `json:"dn,omitempty"`
	Failed   int             `json:"fl,omitempty"`
	Day      int             `json:"d,omitempty"`

	cache    map[string]*nqProp
	cacheDay int
}

func init() {
	unLocalKeys = append(unLocalKeys, "nq")
	talkHooks = append(talkHooks, nqTalk)
	turnHooks = append(turnHooks, nqTurn)
	killHooks = append(killHooks, nqKill)
	enterHooks = append(enterHooks, nqEnter)
}

func nqSt(g *Game) *nqState {
	s := g.ext("nq", func() any { return &nqState{} }).(*nqState)
	if s.Next == nil {
		s.Next = map[string]int{}
	}
	if s.BurgNext == nil {
		s.BurgNext = map[int]int{}
	}
	if s.Favour == nil {
		s.Favour = map[string]bool{}
	}
	if s.Firsts == nil {
		s.Firsts = map[string]bool{}
	}
	return s
}

// nqProp is a request before it is accepted.
type nqProp struct {
	Tpl, Title, Offer string
	Steps             []nqStep
	Days, Grace       int
	Coin, XP, Rep, Op int
	Extra             string
	Subj, Subj2       string
	Num               int
	Item              string
	Give              *nqGive
	Zone              string
	TP                int
	Place             string
	Fx                string
}

type nqGive struct {
	Kind, Name, Text, Lore string
	Glyph                  rune
	Col                    byte
}

type nqTpl struct {
	ID    string
	Goals string // goals that make this template likelier
	W     int
	Build func(x *nqX) *nqProp
}

var nqTpls = []nqTpl{
	{"missing_spouse", "love family", 3, func(x *nqX) *nqProp { return nqMissing(x, "spouse") }},
	{"missing_child", "family", 3, func(x *nqX) *nqProp { return nqMissing(x, "child") }},
	{"missing_friend", "love fame", 2, func(x *nqX) *nqProp { return nqMissing(x, "friend") }},
	{"debt_collect", "wealth justice", 3, nqDebtCollect},
	{"debt_forgive", "safety", 2, nqDebtForgive},
	{"debt_pay", "safety family", 2, nqDebtPay},
	{"letter", "love family fame justice revenge safety", 3, func(x *nqX) *nqProp { return nqLetter(x, false) }},
	{"parcel", "wealth faith knowledge", 3, func(x *nqX) *nqProp { return nqLetter(x, true) }},
	{"revenge_kill", "revenge", 2, func(x *nqX) *nqProp { return nqRevenge(x, "kill") }},
	{"revenge_scare", "revenge justice", 3, func(x *nqX) *nqProp { return nqRevenge(x, "scare") }},
	{"revenge_expose", "revenge justice", 3, func(x *nqX) *nqProp { return nqRevenge(x, "expose") }},
	{"protect_child", "family safety", 3, func(x *nqX) *nqProp { return nqProtect(x, "child") }},
	{"protect_shop", "wealth safety", 3, func(x *nqX) *nqProp { return nqProtect(x, "shop") }},
	{"protect_wedding", "love family", 3, func(x *nqX) *nqProp { return nqProtect(x, "wedding") }},
	{"heirloom", "family faith", 3, func(x *nqX) *nqProp { return nqHeir(x, false) }},
	{"lostbook", "knowledge", 3, func(x *nqX) *nqProp { return nqHeir(x, true) }},
	{"innocence", "justice family", 3, func(x *nqX) *nqProp { return nqJailed(x, true) }},
	{"bail", "family wealth", 3, func(x *nqX) *nqProp { return nqJailed(x, false) }},
	{"vouch", "wealth fame", 3, nqVouch},
	{"matchmaking", "love", 5, nqMatchmake},
	{"stolen", "justice wealth", 3, nqStolen},
	{"feud", "revenge safety", 3, nqFeud},
	{"funeral", "faith family", 4, nqFuneral},
	{"sick", "family safety faith", 4, nqSick},
	{"rescue", "family safety", 1, nqRescue},
	{"stake", "wealth", 4, nqStake},
	{"watch", "justice safety", 4, nqWatch},
	{"fame", "fame", 5, nqFame},
}

// nqX is the working set of a template builder.
type nqX struct {
	g   *Game
	p   *lvPerson
	r   *Rng
	day int
	gb  int
	c   *lsCtx
}

func (x *nqX) ctx() *lsCtx {
	if x.c == nil {
		x.c = lsMakeCtx(x.g)
		x.c.rebuild()
	}
	return x.c
}

func (x *nqX) days(a, b int) int {
	c := x.ctx()
	ba, bb := c.burg(a), c.burg(b)
	if ba == nil || bb == nil {
		return 4
	}
	return c.days(ba, bb)
}

func (x *nqX) bname(b int) string { return lvBurgName(x.g, b) }

// townsfolk finds a living adult in a burg that satisfies ok (a deterministic pick), seeding the burg's people if needed.
func (x *nqX) townsfolk(burg int, ok func(q *lvPerson) bool) *lvPerson {
	if burg <= 0 {
		return nil
	}
	if !lvSt(x.g).Seeded[burg] {
		lvEnsureTown(x.g, burg)
	}
	l := lvPresent(x.g, burg)
	if len(l) == 0 {
		return nil
	}
	off := x.r.Int(0, len(l)-1)
	for i := range l {
		q := l[(i+off)%len(l)]
		if q.Key != x.p.Key && nqAdult(x.g, q) && ok(q) {
			return q
		}
	}
	return nil
}

func nqAdult(g *Game, q *lvPerson) bool { return q.lvAge(lvYear(g)) >= lsAdultAge(q.Species) }

func nqBurgOf(q *lvPerson) int {
	switch {
	case q.Burg > 0:
		return q.Burg
	case q.Dest > 0:
		return q.Dest
	}
	return q.Home
}

func nqRelWord(p *lvPerson, key string) string {
	if p.Spouse == key {
		if lv := p.Rel[key]; lv != "" {
			return lv
		}
		return "spouse"
	}
	if w := p.Rel[key]; w != "" {
		return w
	}
	if lsHas(p.Kin, key) {
		return "kinsman"
	}
	return "friend"
}

func nqCoin(g *Game, base int) int { return base + g.P.Level*2 }

// ---------------------------------------------------------------- offering

func nqMayOffer(g *Game, st *nqState, p *lvPerson, day int) bool {
	if g.Cur == nil || g.Cur.Kind != "town" || g.Cur.Burg <= 0 || g.OnWorld || !p.Alive || p.Burg != g.Cur.Burg || p.Opinion <= -25 {
		return false
	}
	if _, jailed := lsSt(g).Jail[p.Key]; jailed || !nqAdult(g, p) {
		return false
	}
	if st.Next[p.Key] > day || st.BurgNext[g.Cur.Burg] > day || nqOpen(st) >= 3 {
		return false
	}
	for _, q := range st.Q {
		if q.Stage == 1 && q.Giver == p.Key {
			return false
		}
	}
	return HashStr(g.W.Seed+":nq:has:"+p.Key+":"+strconv.Itoa(day/4))%100 < 40
}

func nqOpen(st *nqState) int {
	n := 0
	for _, q := range st.Q {
		if q.Stage == 1 {
			n++
		}
	}
	return n
}

func nqOfferFor(g *Game, p *lvPerson) *nqProp {
	st := nqSt(g)
	day := lvDay(g)
	if st.cache == nil || st.cacheDay != day {
		st.cache, st.cacheDay = map[string]*nqProp{}, day
	}
	if v, ok := st.cache[p.Key]; ok {
		return v
	}
	var pr *nqProp
	if nqMayOffer(g, st, p, day) {
		pr = nqBuild(g, p, day)
	}
	st.cache[p.Key] = pr
	return pr
}

func nqBuild(g *Game, p *lvPerson, day int) *nqProp {
	r := NewRng(fmt.Sprintf("%s:nq:pick:%s:%d", g.W.Seed, p.Key, day/4))
	x := &nqX{g: g, p: p, r: r, day: day, gb: g.Cur.Burg}
	type cw struct {
		t *nqTpl
		w int
	}
	var cs []cw
	for i := range nqTpls {
		t := &nqTpls[i]
		w := t.W
		if p.GoalID != "" && strings.Contains(t.Goals, p.GoalID) {
			w *= 3
		}
		cs = append(cs, cw{t, w})
	}
	for len(cs) > 0 {
		tot := 0
		for _, c := range cs {
			tot += c.w
		}
		n := r.Int(0, tot-1)
		i := 0
		for ; i < len(cs)-1; i++ {
			if n < cs[i].w {
				break
			}
			n -= cs[i].w
		}
		t := cs[i].t
		cs = append(cs[:i], cs[i+1:]...)
		x.r = NewRng(fmt.Sprintf("%s:nq:b:%s:%s:%d", g.W.Seed, t.ID, p.Key, day/4))
		if pr := t.Build(x); pr != nil {
			pr.Tpl = t.ID
			if pr.Extra == "" {
				switch n := x.r.Int(0, 99); {
				case n < 25:
					pr.Extra = "item"
				case n < 40:
					if p.Secret != "" {
						pr.Extra = "secret"
					}
				case n < 60:
					pr.Extra = "favour"
				}
			}
			return pr
		}
	}
	return nil
}

// nqTalk: dialogue options (hook in talkMenu): the step of an active quest this person can finish, a favour you are
// owed, and "Is something troubling you?" when they have a request.
func nqTalk(g *Game, a *Actor) []Option {
	if g.W == nil || g.Cur == nil || a == nil || a.AI == "hostile" || a.AI == "follower" {
		return nil
	}
	st := nqSt(g)
	var out []Option
	for _, q := range st.Q {
		s := q.cur()
		if s == nil || !nqMatch(g, q, s, a) {
			continue
		}
		q, s := q, s
		label := s.Opt
		if s.Check != "" {
			label = g.checkLabel(s.Check, s.DC) + " " + label
		}
		if s.Cost > 0 {
			label += fmt.Sprintf(" {{K|(%d %s)}}", s.Cost, g.Sk.Currency)
		}
		out = append(out, Option{label, func() { nqDoStep(g, a, q, s) }})
	}
	if a.Tpl != "npc" {
		return out
	}
	p := lvPersonOf(g, a)
	if p == nil || !p.Alive {
		return out
	}
	if st.Favour[p.Key] {
		out = append(out, Option{"Ask " + p.Name + " for the favour they owe you.", func() { nqFavour(g, p) }})
	}
	for _, q := range st.Q {
		if q.Stage != 1 || q.Item == "" || q.Opened || q.Cur != 0 || (q.Tpl != "letter" && q.Tpl != "parcel") || nqFindItem(g, q, "obj") == nil {
			continue
		}
		if s := q.cur(); s != nil && nqMatch(g, q, s, a) {
			continue
		}
		q := q
		out = append(out, Option{"{{R|Break the seal and read the " + q.Item + ".}}", func() { nqOpenLetter(g, a, q) }})
		break
	}
	if pr := nqOfferFor(g, p); pr != nil {
		out = append(out, Option{"Is something troubling you?", func() { nqAsk(g, a, p, pr) }})
	}
	return out
}

func nqAsk(g *Game, a *Actor, p *lvPerson, pr *nqProp) {
	g.Talk, g.Mode, g.Sel = a, "talk", 0
	extra := ""
	if w := nqExtraWords[pr.Extra]; w != "" {
		extra = " and " + w
	}
	g.Popup = Wrap(pr.Offer+fmt.Sprintf("\n\n{{K|(About %d days. Reward: %d %s%s.)}}", pr.Days, pr.Coin, g.Sk.Currency, extra), 100)
	g.Opts = []Option{
		{"{{G|I'll see to it.}}", func() { nqAccept(g, a, p, pr) }},
		{"I cannot help just now.", func() {
			st := nqSt(g)
			st.Next[p.Key] = lvDay(g) + 6
			st.cache = nil
			g.talkMenu("\"I understand. Another time, perhaps.\"")
		}},
	}
}

func nqAccept(g *Game, a *Actor, p *lvPerson, pr *nqProp) {
	st := nqSt(g)
	day := lvDay(g)
	q := &nqQuest{ID: g.newID(), Tpl: pr.Tpl, Title: pr.Title, Offer: pr.Offer, Giver: p.Key, GBurg: g.Cur.Burg, Subj: pr.Subj, Subj2: pr.Subj2,
		Steps: pr.Steps, Stage: 1, Start: day, Due: day + pr.Days, Grace: pr.Grace, Coin: pr.Coin, XP: pr.XP, Rep: pr.Rep, Op: pr.Op,
		Extra: pr.Extra, Num: pr.Num, Item: pr.Item, Place: pr.Place, Zone: pr.Zone, TP: pr.TP, Fx: pr.Fx}
	st.Q = append(st.Q, q)
	st.BurgNext[g.Cur.Burg] = day + 2
	st.cache = nil
	mq := &Quest{ID: q.ID, Kind: "npcq", Title: q.Title, Text: q.Offer, Giver: p.Name, GiverID: -1, Faction: "player", Target: q.Zone, TP: q.TP}
	g.Quests = append(g.Quests, mq)
	p.remember(day, "quest", "You promised to help: "+q.Title+".", 3)
	if pr.Give != nil {
		g.give(nqMakeItem(g, q, pr.Give))
	}
	switch q.Tpl {
	case "rescue":
		if s := lvGet(g, q.Subj); s != nil {
			nqTakeAway(g, s, q)
		}
	}
	nqSync(g, q)
	g.Msg("{{W|New request: " + q.Title + "}}")
	if !st.Firsts["first"] {
		st.Firsts["first"] = true
		g.deed("quest", fmt.Sprintf("The wanderer %s took up a request from %s of %s.", g.P.Name, p.Name, lvBurgName(g, p.Home)), 0, 0)
	}
	if g.Cur.Key == q.Zone {
		nqPlace(g, g.Cur, q)
	}
	g.talkMenu("\"Thank you. I knew you looked the sort.\"\n\n{{K|(Added to your journal. " + nqHint(g, q) + ")}}")
}

func nqMakeItem(g *Game, q *nqQuest, gv *nqGive) *Item {
	it := &Item{ID: g.newID(), Kind: gv.Kind, Name: gv.Name, Glyph: gv.Glyph, F: gv.Col, Value: 1, Weight: 1, Quest: q.ID, Ident: true,
		Text: gv.Text, Lore: gv.Lore, Extra: map[string]int{"nq:q": q.ID}}
	if gv.Kind == "letter" {
		it.Weight = 0
	}
	return it
}

func nqFindItem(g *Game, q *nqQuest, need string) *Item {
	for _, it := range g.P.Inv {
		switch need {
		case "obj":
			if it.Quest == q.ID || (it.Extra != nil && it.Extra["nq:q"] == q.ID) {
				return it
			}
		case "herb":
			if it.Kind == "tonic" || (it.Kind == "ingredient" && (it.Effect == "mending" || strings.Contains(strings.ToLower(it.Name), "herb"))) {
				return it
			}
		}
	}
	return nil
}

// ---------------------------------------------------------------- steps

func nqMatch(g *Game, q *nqQuest, s *nqStep, a *Actor) bool {
	if g.Cur == nil || g.OnWorld {
		return false
	}
	switch s.K {
	case "talk", "pay", "item":
	default:
		return false
	}
	if s.Burg > 0 && g.Cur.Burg != s.Burg {
		return false
	}
	if s.Who != "" {
		p := lvPersonOf(g, a)
		return p != nil && p.Key == s.Who && p.Alive
	}
	switch s.Role {
	case "":
		return false
	case "captive":
		return q.NPC != 0 && a.ID == q.NPC
	case "*":
		p := lvPersonOf(g, a)
		return p != nil && p.Alive && p.Key != q.Giver && a.Tpl == "npc"
	}
	for _, r := range strings.Split(s.Role, "|") {
		if a.Role == r {
			return true
		}
	}
	return false
}

func nqDoStep(g *Game, a *Actor, q *nqQuest, s *nqStep) {
	if q.cur() != s {
		return
	}
	var it *Item
	if s.Need != "" {
		if it = nqFindItem(g, q, s.Need); it == nil {
			g.talkMenu("You do not have what is needed with you.")
			return
		}
	}
	if s.Check != "" && !g.roll2d6Check(g.checkMod(s.Check), s.DC) {
		s.Tries++
		if p := lvPersonOf(g, a); p != nil {
			p.remember(lvDay(g), "insult", "You pressed them on another's account.", -3)
		}
		if s.Tries >= 3 {
			nqFail(g, q, "They will not hear you out again.")
			g.talkMenu("{{R|[Failure]}}\n\n" + s.Fail + "\n\n{{R|They will hear no more of it.}}")
			return
		}
		g.talkMenu("{{R|[Failure]}}\n\n" + s.Fail)
		return
	}
	if s.Cost > 0 && !g.cuSpend(s.Cost) {
		g.talkMenu("You cannot afford that.")
		return
	}
	txt := nqEffect(g, q, s, a, it)
	if s.Say != "" {
		txt = s.Say + "\n\n" + txt
	}
	s.Done = true
	q.Cur++
	if q.Cur >= len(q.Steps) {
		txt += "\n\n" + nqFinish(g, q, a)
	} else {
		nqSync(g, q)
		g.Msg("{{c|" + nqHint(g, q) + "}}")
	}
	g.talkMenu(strings.TrimSpace(txt))
}

func nqHint(g *Game, q *nqQuest) string {
	s := q.cur()
	if s == nil {
		return ""
	}
	nm := func(k string) string { return lvName(g, k) }
	where := ""
	if s.Burg > 0 {
		where = " in " + lvBurgName(g, s.Burg)
	}
	switch {
	case s.K == "stay":
		return fmt.Sprintf("Stay in %s (about %d hours to go).", lvBurgName(g, s.Burg), maxi(0, s.Mins-q.Mins+59)/60)
	case s.K == "place":
		return "Look for it: " + nqZoneName(g, q)
	case s.K == "kill":
		return "Find and kill " + nm(s.Who) + where + "."
	case s.Who != "" && s.Who == q.Giver:
		return "Return to " + nm(s.Who) + " in " + lvBurgName(g, q.GBurg) + "."
	case s.Who != "":
		return "Find " + nm(s.Who) + where + "."
	case s.Role == "captive":
		return "Find the captive: " + nqZoneName(g, q)
	case s.Role == "*":
		return "Find someone to talk to" + where + "."
	}
	return "Find a " + strings.ReplaceAll(s.Role, "|", " or ") + where + "."
}

func nqZoneName(g *Game, q *nqQuest) string {
	if q.TP >= 0 && q.TP < len(g.W.Biome) {
		return "near " + g.W.RegionName(q.TP)
	}
	return "in the wilds"
}

// nqSync refreshes the journal mirror: text and map marker follow the current step.
func nqSync(g *Game, q *nqQuest) {
	var mq *Quest
	for _, m := range g.Quests {
		if m.ID == q.ID {
			mq = m
			break
		}
	}
	if mq == nil {
		return
	}
	mq.Title = q.Title
	mq.Text = q.Offer
	if h := nqHint(g, q); h != "" && q.Stage == 1 {
		mq.Text += "  Now: " + h
	}
	s := q.cur()
	if s == nil {
		return
	}
	burg := s.Burg
	if burg == 0 && s.Who != "" {
		if p := lvGet(g, s.Who); p != nil {
			burg = nqBurgOf(p)
		}
	}
	switch {
	case s.K == "place" || s.Role == "captive":
		mq.Target, mq.TP = q.Zone, q.TP
	case burg > 0 && g.W.burgByID[burg] != nil:
		b := g.W.burgByID[burg]
		mq.TP = b.P
		mq.Target = ZoneKey(b.P%g.W.W, b.P/g.W.W, 1, 1, 0)
	}
	if mq.TP >= 0 {
		g.Known[mq.TP] = true
	}
}

func nqClose(g *Game, q *nqQuest, stage int, result string) {
	q.Stage, q.Result = stage, result
	for _, m := range g.Quests {
		if m.ID == q.ID {
			m.Done, m.Closed = true, true
			if stage == 4 {
				m.Title += " (failed)"
			}
		}
	}
	st := nqSt(g)
	st.cache = nil
	// keep the history short
	var fin []*nqQuest
	for _, o := range st.Q {
		if o.Stage != 1 {
			fin = append(fin, o)
		}
	}
	if len(fin) > 30 {
		drop := fin[0]
		for i, o := range st.Q {
			if o == drop {
				st.Q = append(st.Q[:i], st.Q[i+1:]...)
				break
			}
		}
	}
}

// ---------------------------------------------------------------- outcomes

func nqSendHome(g *Game, p *lvPerson) {
	ls := lsSt(g)
	if !p.Alive || p.Home <= 0 {
		return
	}
	if _, jailed := ls.Jail[p.Key]; jailed {
		return
	}
	day := lvDay(g)
	if p.Burg == p.Home {
		return
	}
	days := 2
	if p.Burg > 0 {
		c := lsMakeCtx(g)
		if a, b := c.burg(p.Burg), c.burg(p.Home); a != nil && b != nil {
			days = c.days(a, b)
		}
		ls.From[p.Key] = p.Burg
	}
	p.Burg, p.Dest, p.ArriveD = 0, p.Home, day+days
}

func nqTakeAway(g *Game, s *lvPerson, q *nqQuest) {
	ls := lsSt(g)
	day := lvDay(g)
	ls.From[s.Key] = s.Burg
	s.Burg, s.Dest = 0, s.Home
	s.ArriveD = q.Due + q.Grace + 4
	ls.Jail[s.Key] = s.ArriveD
	s.addLife(lvYear(g), "crime", "Was carried off by bandits.")
	lvSt(g).addNews(day, s.Home, "%s of %s was carried off by bandits.", s.Name, lvBurgName(g, s.Home))
	if g.Cur != nil {
		b := lvb(g)
		var keep []*Actor
		for _, a := range g.Cur.Actors {
			if b.Bind[a.ID] == s.Key {
				delete(b.Bind, a.ID)
				continue
			}
			keep = append(keep, a)
		}
		g.Cur.Actors = keep
	}
}

func nqRelease(g *Game, j *lvPerson) {
	ls := lsSt(g)
	if _, ok := ls.Jail[j.Key]; !ok {
		return
	}
	delete(ls.Jail, j.Key)
	to := ls.From[j.Key]
	delete(ls.From, j.Key)
	if to <= 0 {
		to = j.Home
	}
	j.Burg, j.Dest, j.ArriveD = to, 0, 0
	lsMood(j, 15)
	j.addLife(lvYear(g), "crime", "Was set free.")
}

func nqNews(g *Game, burg int, format string, a ...any) {
	lvSt(g).addNews(lvDay(g), burg, format, a...)
}

func nqEffect(g *Game, q *nqQuest, s *nqStep, a *Actor, it *Item) string {
	ls := lsSt(g)
	day, yr := lvDay(g), lvYear(g)
	giver, subj, other := lvGet(g, q.Giver), lvGet(g, q.Subj), lvGet(g, q.Subj2)
	var who *lvPerson
	if a != nil && a.Tpl == "npc" {
		who = lvPersonOf(g, a)
	}
	if giver == nil {
		giver = &lvPerson{Name: "someone"}
	}
	if subj == nil {
		subj = &lvPerson{Name: "someone"}
	}
	if other == nil {
		other = &lvPerson{Name: "someone"}
	}
	if it != nil && (s.Need == "obj" || s.Need == "herb") {
		g.removeInv(it)
	}
	switch s.Eff {
	case "word":
		subj.remember(day, "help", "You brought word from "+giver.Name+".", 4)
		lsMood(subj, 6)
		if HashStr(fmt.Sprint(q.ID))%2 == 0 {
			nqSendHome(g, subj)
		}
		return fmt.Sprintf("%s listens, then lets out a long breath. \"Tell %s I am well. I have been a fool not to write.\"", subj.Name, giver.Name)
	case "bringhome":
		subj.remember(day, "help", "You brought word from "+giver.Name+".", 5)
		nqSendHome(g, subj)
		return fmt.Sprintf("%s is quiet a while. \"Then I have stayed away too long. I will start for home at once.\"", subj.Name)
	case "collect":
		amt := mini(q.Num, subj.Wealth)
		subj.Wealth -= amt
		giver.Wealth += amt
		subj.Rivals = lsAdd(subj.Rivals, giver.Key, 4)
		subj.remember(day, "insult", "You dunned them for "+giver.Name+".", -4)
		giver.addLife(yr, "help", fmt.Sprintf("Got back %d coin owed by %s.", amt, subj.Name))
		return fmt.Sprintf("%s counts out %d coin with a sour face. \"There. Tell %s we are quits, and that I remember who sent you.\"", subj.Name, amt, giver.Name)
	case "forgive":
		subj.remember(day, "help", "You spoke for "+giver.Name+".", 2)
		giver.Wealth += q.Num / 2
		giver.addLife(yr, "help", subj.Name+" forgave a debt.")
		lsMood(giver, 12)
		return fmt.Sprintf("%s waves a hand. \"Fine. Half of it was always pride. Tell %s the debt is forgotten.\"", subj.Name, giver.Name)
	case "debtpay":
		subj.Wealth += q.Num
		subj.remember(day, "help", "You settled a debt for "+giver.Name+".", 6)
		giver.addLife(yr, "help", "A debt to "+subj.Name+" was cleared by a stranger.")
		lsMood(giver, 10)
		return fmt.Sprintf("%s takes the coin and nods. \"Paid in full. I will tell %s so.\"", subj.Name, giver.Name)
	case "deliver":
		txt := fmt.Sprintf("%s takes it and reads in silence.", subj.Name)
		if body, ok := nqLetterBody[giver.GoalID]; ok && q.Tpl == "letter" {
			txt += " " + fmt.Sprintf("{{K|\"%s\"}}", fmt.Sprintf(body, giver.Name))
		}
		subj.remember(day, "help", "You brought them word from "+giver.Name+".", 6)
		lsMood(subj, 8)
		if giver.GoalID == "love" && subj.Spouse == "" && giver.Spouse == "" {
			ls.Court[lsPair(giver.Key, subj.Key)] += 25
			ls.Wooing[giver.Key], ls.Wooing[subj.Key] = subj.Key, giver.Key
		}
		return txt
	case "scare":
		subj.Mood = clampi(subj.Mood-10, -100, 100)
		subj.remember(day, "insult", "You warned them off "+giver.Name+".", -8)
		delete(ls.Feud, lsPair(giver.Key, subj.Key))
		for _, k := range subj.Kin {
			if kin := lvGet(g, k); kin != nil && kin.Alive {
				kin.remember(day, "insult", "You threatened their kin "+subj.Name+".", -4)
			}
		}
		lsMood(giver, 10)
		return fmt.Sprintf("%s goes pale, then red. \"Tell %s I have no quarrel. None. Now go.\"", subj.Name, giver.Name)
	case "expose":
		lvRevealSecret(g, subj)
		subj.Mood = clampi(subj.Mood-25, -100, 100)
		subj.Wealth = maxi(0, subj.Wealth-15)
		subj.addLife(yr, "secret", "Their secret was cried in public: "+subj.Secret)
		for _, k := range subj.Kin {
			if kin := lvGet(g, k); kin != nil && kin.Alive {
				kin.remember(day, "insult", "You shamed their kin "+subj.Name+" in public.", -6)
			}
		}
		nqNews(g, subj.Home, "A scandal in %s: it is whispered that %s has been hiding something.", lvBurgName(g, subj.Home), subj.Name)
		lsGoalBump(giver, "revenge", 35)
		return "The word goes round the market like fire in dry grass. By evening nobody will speak of anything else."
	case "killed":
		lsGoalBump(giver, "revenge", 60)
		lsMood(giver, 12)
		return "It is done."
	case "guarded":
		return "The hours pass, and the trouble with them."
	case "heirdone":
		giver.addLife(yr, "help", "Got back a lost "+q.Item+" through a stranger.")
		lsMood(giver, 15)
		return fmt.Sprintf("%s turns the %s over in both hands, and cannot speak for a moment.", giver.Name, q.Item)
	case "witness":
		w := subj
		if q.Tpl == "innocence" {
			w = other
		}
		if w != nil && w.Key != "" {
			f := knSt(g).add(knFact{Kind: "sighting", Subject: q.Subj, Text: fmt.Sprintf("%s was with %s the whole of that night.", subj.Name, w.Name), True: true, Source: w.Key, Day: day, Burg: q.GBurg, Lvl: 1})
			knGive(w, f.ID)
			g.knLearn(f, w, 2, "ask")
			w.remember(day, "help", "They spoke up for "+subj.Name+" because of you.", 2)
		}
		g.give(nqMakeItem(g, q, &nqGive{Kind: "letter", Name: "witness statement", Glyph: '?', Col: CWhite, Text: "A signed account, setting out where " + subj.Name + " was that night."}))
		return "They think it over, then take pen and mark. \"I saw what I saw. Take it to the watch before I lose my nerve.\""
	case "release":
		nqRelease(g, subj)
		giver.remember(day, "help", "You freed "+subj.Name+".", 10)
		nqNews(g, q.GBurg, "%s was freed in %s after a stranger came forward with proof.", subj.Name, lvBurgName(g, q.GBurg))
		return fmt.Sprintf("The captain reads it twice, frowns, and sends for the keys. \"%s may go. See that nobody hears how.\"", subj.Name)
	case "bail":
		nqRelease(g, subj)
		giver.remember(day, "help", "You paid the bail for "+subj.Name+".", 10)
		nqNews(g, q.GBurg, "%s was bailed out in %s.", subj.Name, lvBurgName(g, q.GBurg))
		return fmt.Sprintf("The coin is counted. \"%s walks. Do not make me regret it.\"", subj.Name)
	case "vouch":
		giver.Wealth += 20
		lsGoalBump(giver, "wealth", 20)
		subj.remember(day, "help", "You spoke well of "+giver.Name+".", 2)
		giver.addLife(yr, "job", "Was spoken well of to "+subj.Name+" by a stranger.")
		return fmt.Sprintf("%s considers you. \"If %s means that much to you, then I shall look at the work again. Say so.\"", subj.Name, giver.Name)
	case "court":
		pair := lsPair(giver.Key, subj.Key)
		ls.Court[pair] += 35
		ls.Wooing[giver.Key], ls.Wooing[subj.Key] = subj.Key, giver.Key
		subj.remember(day, "gift", "You brought "+giver.Name+"'s gift.", 5)
		lsMood(subj, 10)
		return fmt.Sprintf("%s turns the gift over, and a smile creeps in. \"Well. Tell %s that I did not say no.\"", subj.Name, giver.Name)
	case "stolen":
		take := mini(subj.Wealth, q.Num)
		subj.Wealth -= take
		subj.remember(day, "insult", "You took back what they stole.", -8)
		g.give(nqMakeItem(g, q, &nqGive{Kind: "trophy", Name: nqSkinFor(g.Genre).Purse + " of " + giver.Name, Glyph: '$', Col: CYellow, Lore: "Stolen from " + giver.Name + "."}))
		return fmt.Sprintf("%s hands it over without meeting your eye. \"Take it. I never wanted trouble.\"", subj.Name)
	case "gotback":
		giver.Wealth += q.Num
		giver.addLife(yr, "help", "Got stolen goods back through a stranger.")
		lsMood(giver, 12)
		return fmt.Sprintf("%s counts everything twice. \"All here. All of it.\"", giver.Name)
	case "peace":
		giver.Rivals, subj.Rivals = lsDel(giver.Rivals, subj.Key), lsDel(subj.Rivals, giver.Key)
		delete(ls.Feud, lsPair(giver.Key, subj.Key))
		lsMood(subj, 8)
		lsMood(giver, 8)
		giver.addLife(yr, "friend", "Made peace with "+subj.Name+".")
		subj.addLife(yr, "friend", "Made peace with "+giver.Name+".")
		nqNews(g, q.GBurg, "%s and %s have settled their quarrel in %s.", giver.Name, subj.Name, lvBurgName(g, q.GBurg))
		return fmt.Sprintf("%s is silent, then shrugs. \"Years of it. I am tired. Tell %s I will meet them halfway.\"", subj.Name, giver.Name)
	case "priest":
		if who != nil && who.Alive && who.Burg == g.Cur.Burg {
			c := lsMakeCtx(g)
			if from, to := c.burg(who.Burg), c.burg(q.GBurg); from != nil && to != nil {
				ls.From[who.Key] = who.Burg
				who.Burg, who.Dest, who.ArriveD = 0, q.GBurg, day+c.days(from, to)
			}
		}
		return "\"The dead must be sent off properly. I will come as soon as I may.\""
	case "cure":
		delete(ls.Sick, subj.Key)
		subj.Mood = clampi(subj.Mood+10, -100, 100)
		subj.addLife(yr, "illness", "Recovered after a stranger brought a remedy.")
		giver.remember(day, "help", "You saved "+subj.Name+".", 10)
		return fmt.Sprintf("%s takes it with shaking hands. \"I have nothing to give you but thanks. %s will be on their feet in days.\"", giver.Name, subj.Name)
	case "rescue":
		delete(ls.Jail, subj.Key)
		delete(ls.From, subj.Key)
		subj.Burg = 0
		subj.Dest, subj.ArriveD = subj.Home, day+2
		subj.remember(day, "help", "You cut them loose from their captors.", 20)
		subj.addLife(yr, "help", "Was freed from bandits by a stranger.")
		if g.Cur != nil && a != nil {
			for i, o := range g.Cur.Actors {
				if o == a {
					g.Cur.Actors = append(g.Cur.Actors[:i], g.Cur.Actors[i+1:]...)
					break
				}
			}
		}
		return "The bonds fall away. \"I will run all the way home. Come after, if you can; they will want to thank you.\" They are gone through the grass."
	case "stake":
		if v := ls.Vacant[q.GBurg]; len(v) > 0 {
			giver.Job = v[0]
			ls.Vacant[q.GBurg] = v[1:]
		} else {
			giver.Job = nqTradeStake[int(HashStr(giver.Key))%len(nqTradeStake)]
		}
		giver.Wealth += 10
		lsGoalBump(giver, "wealth", 30)
		giver.addLife(yr, "job", "Opened a "+giver.Job+"'s shop with a stranger's backing.")
		nqNews(g, q.GBurg, "%s opens a new %s's shop in %s.", giver.Name, giver.Job, lvBurgName(g, q.GBurg))
		return fmt.Sprintf("%s grips your hand. \"A shop. My own. I will stock the first shelf with your name on it.\"", giver.Name)
	case "watch":
		if subj.Alive && subj.Burg > 0 {
			c := lsMakeCtx(g)
			c.rebuild()
			lsPunish(c, subj, "theft", c.rng(subj.Key+":nq"), 1)
		}
		lsGoalBump(giver, "justice", 40)
		return "The sergeant writes it down. \"We have been watching that one. Leave it with us.\""
	case "spread":
		lsGoalBump(giver, "fame", 25)
		nqNews(g, g.Cur.Burg, "Travellers in %s are talking of %s.", lvBurgName(g, g.Cur.Burg), giver.Name)
		if who != nil {
			who.remember(day, "talk", "You told them of "+giver.Name+".", 1)
		}
		return "They listen, nod, and promise to pass it on."
	}
	return ""
}

// nqFinish pays the rewards and writes the outcome.
func nqFinish(g *Game, q *nqQuest, a *Actor) string {
	st := nqSt(g)
	day, yr := lvDay(g), lvYear(g)
	giver := lvGet(g, q.Giver)
	coin, xp, op := q.Coin, q.XP, q.Op
	var notes []string
	if day > q.Due {
		q.Late = true
		coin /= 2
		op /= 2
		notes = append(notes, "{{y|Late: the reward is halved.}}")
	}
	if q.Opened {
		coin = coin * 3 / 4
		op -= 20
		notes = append(notes, "{{R|The broken seal costs you.}}")
	}
	if giver != nil {
		giver.remember(day, "quest", "You saw their request through: "+q.Title+".", op)
		giver.Mood = clampi(giver.Mood+8, -100, 100)
		giver.addLife(yr, "help", "A wanderer helped: "+q.Title+".")
		st.Next[giver.Key] = day + 20
		if giver.State != 0 && q.Rep > 0 {
			g.AddRep(fmt.Sprintf("state:%d", giver.State), q.Rep, " for helping one of its people")
		}
	}
	if coin > 0 {
		notes = append(notes, "{{G|"+g.cuEarn(coin)+"}}")
	}
	g.gainXP(xp)
	switch q.Extra {
	case "item":
		it := g.Loot(NewRng(fmt.Sprintf("%s:nq:rew:%d", g.W.Seed, q.ID)), g.P.Level/3+1)
		g.give(it)
		notes = append(notes, "They press "+it.Display()+" on you.")
	case "secret":
		if giver != nil && giver.Secret != "" {
			if g.addSecret(giver.Name+" confided: "+giver.Secret, "confidence", -1, g.W.Year) {
				notes = append(notes, "{{c|They trust you with a secret.}}")
			}
			lvRevealSecret(g, giver)
		} else if t, p, ok := g.nearestSite(); ok {
			if g.addSecret(t, "dungeon", p, g.W.Year) {
				notes = append(notes, "{{c|They tell you of a place worth seeing.}}")
			}
		}
	case "favour":
		if giver != nil {
			st.Favour[giver.Key] = true
			notes = append(notes, "{{c|They say you may ask a favour of them any time.}}")
		}
	}
	switch q.Tpl {
	case "protect_wedding":
		if k := lvGet(g, q.Subj); k != nil {
			if m := lvGet(g, q.Subj2); m != nil && k.Alive && m.Alive && k.Spouse == "" && m.Spouse == "" {
				c := lsMakeCtx(g)
				c.rebuild()
				lsMarry(c, k, m, c.rng(k.Key+":nq"), true)
			}
		}
	case "debt_collect", "debt_forgive", "debt_pay":
	}
	nqClose(g, q, 3, "done")
	st.Done++
	nm := "a person"
	if giver != nil {
		nm = giver.Name
	}
	g.Msg("{{G|Request done: " + q.Title + "}}")
	if !st.Firsts["t:"+q.Tpl] {
		st.Firsts["t:"+q.Tpl] = true
		g.deed("quest", fmt.Sprintf("The wanderer %s finished a request for %s (%s).", g.P.Name, nm, strings.ReplaceAll(q.Tpl, "_", " ")), 0, 0)
	}
	if st.Done == 5 || st.Done == 15 {
		g.deed("quest", fmt.Sprintf("The wanderer %s has now helped %d people with their troubles.", g.P.Name, st.Done), 0, 0)
	}
	line := nqThanks[int(HashStr(fmt.Sprint(q.ID))%uint64(len(nqThanks)))]
	if q.Tpl == "letter" || q.Tpl == "parcel" {
		line = "{{K|(Delivered.)}}"
	}
	return line + "\n" + strings.Join(notes, " ")
}

// nqFail ends a request unhappily: the situation resolves by itself.
func nqFail(g *Game, q *nqQuest, why string) {
	if q.Stage != 1 {
		return
	}
	st := nqSt(g)
	day, yr := lvDay(g), lvYear(g)
	giver, subj := lvGet(g, q.Giver), lvGet(g, q.Subj)
	ls := lsSt(g)
	h := int(HashStr(fmt.Sprintf("%s:nqf:%d", g.W.Seed, q.ID)) % 100)
	switch q.Fx {
	case "lostdie":
		if subj != nil && subj.Alive && h < 35 {
			g.lsKill(subj.Key, "lost on the road")
		}
	case "sickdie":
		if subj != nil && subj.Alive && h < 60 {
			g.lsKill(subj.Key, "of the sickness")
		}
	case "captivedie":
		if subj != nil && subj.Alive {
			if h < 45 {
				g.lsKill(subj.Key, "in captivity")
			} else {
				nqRelease(g, subj)
				if giver != nil {
					giver.Wealth = maxi(0, giver.Wealth-40)
					giver.addLife(yr, "crime", "Paid a ransom to bandits.")
				}
			}
		}
	case "debtjail":
		dj := subj
		if q.Tpl == "debt_forgive" || q.Tpl == "debt_pay" {
			dj = giver
		}
		if dj != nil && dj.Alive && dj.Burg > 0 {
			d := 3 + h%8
			ls.From[dj.Key] = dj.Burg
			dj.Burg, dj.Dest, dj.ArriveD = 0, dj.Home, day+d
			ls.Jail[dj.Key] = dj.ArriveD
			dj.addLife(yr, "crime", fmt.Sprintf("Jailed for debt (%d days).", d))
			nqNews(g, q.GBurg, "%s was jailed for debt in %s.", dj.Name, lvBurgName(g, q.GBurg))
		}
	case "robbed":
		if giver != nil {
			giver.Wealth = maxi(0, giver.Wealth-25)
			giver.addLife(yr, "crime", "Was robbed while the shop stood unguarded.")
		}
	case "jaildie":
		if subj != nil && subj.Alive && h < 25 {
			g.lsKill(subj.Key, "of fever in prison")
		}
	case "feudup":
		if giver != nil && subj != nil {
			k := lsPair(giver.Key, subj.Key)
			f := ls.Feud[k]
			if f == nil {
				f = &lsFeud{Day: day}
				ls.Feud[k] = f
			}
			f.Stage = 2
		}
	}
	if giver != nil && giver.Alive {
		giver.remember(day, "quest", "You let their request lapse: "+q.Title+".", -6)
		giver.Mood = clampi(giver.Mood-8, -100, 100)
		st.Next[giver.Key] = day + 12
	}
	st.Failed++
	nqClose(g, q, 4, why)
	g.Msg("{{R|Request failed: " + q.Title + ". " + why + "}}")
	if it := nqFindItem(g, q, "obj"); it != nil {
		g.removeInv(it)
	}
}

// ---------------------------------------------------------------- time

func nqTurn(g *Game) {
	if g.W == nil || g.Cur == nil {
		return
	}
	st := nqSt(g)
	if len(st.Q) == 0 {
		return
	}
	for _, q := range st.Q {
		s := q.cur()
		if s == nil {
			continue
		}
		switch s.K {
		case "stay":
			nqStay(g, st, q, s)
		case "place":
			if nqFindItem(g, q, "obj") != nil || nqHasQuestItem(g, q) {
				s.Done = true
				q.Cur++
				nqSync(g, q)
				g.Msg("{{G|You have the " + q.Item + ".}} {{c|" + nqHint(g, q) + "}}")
			}
		}
	}
	if d := lvDay(g); d != st.Day {
		st.Day = d
		nqDay(g, st, d)
	}
}

func nqHasQuestItem(g *Game, q *nqQuest) bool {
	for _, it := range g.P.Inv {
		if it.Quest == q.ID {
			return true
		}
	}
	return false
}

func nqStay(g *Game, st *nqState, q *nqQuest, s *nqStep) {
	if g.OnWorld || g.Cur.Kind != "town" || g.Cur.Burg != s.Burg {
		return
	}
	q.Mins++
	if !q.Spawned && q.Mins >= s.Mins/2 {
		q.Spawned = true
		r := NewRng(fmt.Sprintf("%s:nq:thug:%d", g.W.Seed, q.ID))
		f := g.Spawn("bandit", g.Cur.Level+1, r)
		f.Name, f.Desc = "troublemaker", "Someone who came looking for a fight and found a witness."
		if x, y, ok := g.Cur.FreeSpot(r, func(x, y int) bool {
			return abs(x-g.P.X) <= 14 && abs(y-g.P.Y) <= 8 && (abs(x-g.P.X) > 4 || abs(y-g.P.Y) > 3)
		}); ok {
			f.X, f.Y = x, y
			f.HomeX, f.HomeY = x, y
			g.Cur.Actors = append(g.Cur.Actors, f)
			q.Foe = f.ID
			g.Msg("{{R|Shouting nearby: trouble has come looking for the one you are guarding.}}")
		} else {
			q.FoeDead = true
		}
	}
	if q.Mins >= s.Mins && (q.Foe == 0 || q.FoeDead) {
		s.Done = true
		q.Cur++
		g.Msg("{{G|The watch is over. Nothing more will come of it today.}}")
		nqSync(g, q)
	}
}

func nqDay(g *Game, st *nqState, day int) {
	for _, q := range st.Q {
		if q.Stage != 1 {
			continue
		}
		giver, subj := lvGet(g, q.Giver), lvGet(g, q.Subj)
		s := q.cur()
		switch {
		case giver == nil || !giver.Alive:
			nqFail(g, q, "The one who asked is dead.")
			continue
		case (q.Tpl == "bail" || q.Tpl == "innocence") && subj != nil && subj.Alive && !nqJailedNow(g, subj):
			nqClose(g, q, 4, "released")
			g.Msg("{{y|" + subj.Name + " was released on their own; the request is moot.}}")
			continue
		case subj != nil && !subj.Alive && s != nil && nqNeedsSubj(q, s):
			if s.K == "kill" {
				nqFail(g, q, "Someone else got there first.")
			} else {
				nqFail(g, q, subj.Name+" is dead.")
			}
			continue
		case day > q.Due+q.Grace:
			nqFail(g, q, "You took too long.")
			continue
		}
		nqSync(g, q)
	}
}

func nqJailedNow(g *Game, p *lvPerson) bool {
	_, ok := lsSt(g).Jail[p.Key]
	return ok
}

func nqNeedsSubj(q *nqQuest, s *nqStep) bool {
	return s.Who != "" && s.Who == q.Subj || s.K == "kill" || q.Place == "captive"
}

func nqKill(g *Game, d, src *Actor) {
	if g.W == nil || d == nil {
		return
	}
	st := nqSt(g)
	for _, q := range st.Q {
		if q.Stage != 1 {
			continue
		}
		if q.Foe != 0 && d.ID == q.Foe {
			q.FoeDead = true
		}
		if q.NPC != 0 && d.ID == q.NPC && q.Place == "captive" {
			if s := lvGet(g, q.Subj); s != nil && s.Alive {
				g.lsKill(s.Key, "slain with their captors")
			}
			nqFail(g, q, "The captive did not survive.")
			continue
		}
		s := q.cur()
		if s == nil || s.K != "kill" || src != g.P {
			continue
		}
		p := lvGet(g, s.Who)
		if p == nil || d.Name != p.Name || p.Alive {
			continue
		}
		s.Done = true
		q.Cur++
		g.Msg("{{c|" + nqEffect(g, q, &nqStep{Eff: "killed"}, nil, nil) + "}}")
		if q.Cur >= len(q.Steps) {
			g.Msg(strings.TrimSpace(nqFinish(g, q, nil)))
		} else {
			nqSync(g, q)
			g.Msg("{{c|" + nqHint(g, q) + "}}")
		}
	}
}

// ---------------------------------------------------------------- placement in the wilds

func nqEnter(g *Game, zn *Zone) {
	if zn == nil || g.W == nil {
		return
	}
	for _, q := range nqSt(g).Q {
		if q.Stage == 1 && !q.Placed && q.Zone != "" && q.Zone == zn.Key {
			nqPlace(g, zn, q)
		}
	}
}

func nqPut(zn *Zone, r *Rng, a *Actor, nx, ny, rad int) bool {
	x, y, ok := zn.FreeSpot(r, func(x, y int) bool { return nx < 0 || (abs(x-nx) <= rad && abs(y-ny) <= rad/2+1) })
	if !ok {
		if x, y, ok = zn.FreeSpot(r, nil); !ok {
			return false
		}
	}
	a.X, a.Y, a.HomeX, a.HomeY = x, y, x, y
	zn.Actors = append(zn.Actors, a)
	return true
}

func nqPlace(g *Game, zn *Zone, q *nqQuest) {
	q.Placed = true
	r := NewRng(fmt.Sprintf("%s:nqp:%d", g.W.Seed, q.ID))
	switch q.Place {
	case "heir", "book":
		col, gl := byte(CYellow), '♦'
		lore := "An old treasure, worn smooth by hands long dead."
		if q.Place == "book" {
			col, gl, lore = CCyan, '≡', "Its owner would give a great deal to have it back."
		}
		it := &Item{ID: g.newID(), Kind: "trophy", Name: q.Item, Glyph: gl, F: col, Value: 40, Weight: 1, Quest: q.ID, Ident: true, Lore: lore}
		if x, y, ok := zn.FreeSpot(r, nil); ok {
			it.X, it.Y = x, y
		}
		zn.Items = append(zn.Items, it)
		for i := 0; i < 2; i++ {
			m := g.Spawn(pickCreature(r, zn.Biome, 's', zn.Level+1), zn.Level+1, r)
			nqPut(zn, r, m, it.X, it.Y, 6)
		}
	case "captive":
		bd := g.W.burgByID[q.GBurg]
		s := lvGet(g, q.Subj)
		if bd == nil || s == nil {
			return
		}
		a := g.makeNPC("villager", bd, g.W.cultByID[bd.Culture], r)
		a.Name, a.Role, a.Title, a.AI, a.F = s.Name, "captive", "captive", "still", CBrown
		a.Desc = "Bound hand and foot beside a dead fire, watching the tree line."
		a.Quest = q.ID
		if !nqPut(zn, r, a, -1, 0, 0) {
			return
		}
		q.NPC = a.ID
		for i := 0; i < 3; i++ {
			m := g.Spawn("bandit", zn.Level+1, r)
			m.Name, m.Desc = "kidnapper", "One of the band that took "+s.Name+"."
			nqPut(zn, r, m, a.X, a.Y, 7)
		}
	}
}

// ---------------------------------------------------------------- letters, favours

func nqOpenLetter(g *Game, a *Actor, q *nqQuest) {
	q.Opened = true
	giver := lvGet(g, q.Giver)
	txt := "You break the seal."
	if giver != nil {
		body := nqLetterBody[giver.GoalID]
		if body == "" {
			body = "Please see that this reaches you safely. %s."
		}
		txt += " {{K|\"" + fmt.Sprintf(body, giver.Name) + "\"}}"
		if giver.Secret != "" {
			lvRevealSecret(g, giver)
			g.addSecret(giver.Name+"'s letter gives away: "+giver.Secret, "confidence", -1, g.W.Year)
			txt += "\n\n{{c|It tells you more than its sender meant to.}}"
		}
	}
	g.talkMenu(txt + "\n\n{{R|The seal is broken, and the recipient will notice.}}")
}

func nqFavour(g *Game, p *lvPerson) {
	st := nqSt(g)
	delete(st.Favour, p.Key)
	g.P.HP = g.P.MaxHP
	delete(g.P.Status, "bleeding")
	delete(g.P.Status, "poisoned")
	p.Mood = clampi(p.Mood+3, -100, 100)
	g.talkMenu(fmt.Sprintf("%s sets you by the fire, binds what needs binding, and will not hear of payment. \"We are even, now.\"\n\n{{G|You are fully healed.}}", p.Name))
}
