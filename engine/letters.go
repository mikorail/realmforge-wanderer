package engine

// Letters, messengers and the town newspaper (prefix ln).
//
// LETTERS: people you have met (and their kin, a rival's kin, parted companions, rival adventurers) write to you from
// their own situations: what lifesim says they did, requests, warnings, invitations, grudges, gifts. A letter is
// written on a day, takes days to reach you (distance / 40 km a day via tvKm, a few days of jitter, plus a rift cost
// across planets), and is handed over when you are in a town: on entering it, or by asking the postmaster. Letters
// on dangerous roads are sometimes lost. Mail screen: mode `ln` (ability ln_mail, or a talk option with a post
// clerk). You reply with stock replies (thanks, promise, decline, coin, gift, rumour, accept/regret an invitation),
// or write to anyone you know; replies travel too and change their opinion (`remember`) when they arrive.
// Invitations are exported through lnInvites() for the family/festival code.
//
// NEWSPAPER: a crier (role "crier", 45% of towns, in the centre zone) cries a headline once per day when you enter
// the plaza and tells the news on request; a printer (role "printer", tier >= 1 towns) sells today's paper
// (Kind "book") built from lifesim, worldsim, tradeways, hazards, wars and the endgame; some rumour items are false
// or stale (marked in Item.Extra["ln:fake"]). Reading once a day gives a little XP and may leave a secret.
//
// State: g.ext("ln") global (mail, contacts, companions, deeds) and g.ext("lnl") planet-local (cooldowns, reported
// events, crier days). Side rngs only: NewRng(seed+":ln:"+key); never g.R in generation.

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

func init() {
	unLocalKeys = append(unLocalKeys, "lnl")
	turnHooks = append(turnHooks, lnTurn)
	enterHooks = append(enterHooks, lnEnter)
	zoneHooks = append(zoneHooks, lnZone)
	talkHooks = append(talkHooks, lnTalk)
	killHooks = append(killHooks, lnKill)
	sideHooks = append(sideHooks, lnSide)
	modeKeys["ln"] = lnKeys
	modeDraws["ln"] = func(g *Game) { g.lnDraw() }
}

// ---------------------------------------------------------------- state

type lnLetter struct {
	ID      int    `json:"i"`
	From    string `json:"f,omitempty"` // person key (planet-local) or ""
	Planet  int    `json:"pl,omitempty"`
	Name    string `json:"n"`
	Role    string `json:"r,omitempty"`
	Place   string `json:"pc,omitempty"`
	Kind    string `json:"k"` // news plea warn gift invite threat note reply taunt
	Sent    int    `json:"s"`
	Due     int    `json:"du"`
	Got     int    `json:"g,omitempty"`
	Text    string `json:"t"`
	Read    bool   `json:"rd,omitempty"`
	Replied bool   `json:"rp,omitempty"`
	Coin    int    `json:"c,omitempty"`
	Gift    string `json:"gf,omitempty"`
	Taken   bool   `json:"tk,omitempty"`
	Ask     int    `json:"as,omitempty"`
	Burg    int    `json:"b,omitempty"` // invitation: where
	Day     int    `json:"d,omitempty"` // invitation: when (lvDay)
	IKind   string `json:"ik,omitempty"`
	Decl    bool   `json:"dc,omitempty"`
	Lost    bool   `json:"lo,omitempty"`
}

type lnOut struct {
	ID     int    `json:"i"`
	To     string `json:"t,omitempty"`
	Planet int    `json:"pl,omitempty"`
	Name   string `json:"n"`
	Kind   string `json:"k"`
	Coin   int    `json:"c,omitempty"`
	Gift   string `json:"g,omitempty"`
	Val    int    `json:"v,omitempty"`
	Rumour string `json:"ru,omitempty"`
	Sent   int    `json:"s"`
	Due    int    `json:"du"`
	W      int    `json:"w,omitempty"`
	Done   bool   `json:"dn,omitempty"`
	Burg   int    `json:"b,omitempty"`
}

type lnContact struct {
	Key    string `json:"k"` // person key
	Planet int    `json:"pl,omitempty"`
	Name   string `json:"n"`
	Role   string `json:"r,omitempty"`
	Place  string `json:"pc,omitempty"`
	Burg   int    `json:"b,omitempty"`
	Op     int    `json:"o,omitempty"`
}

type lnComp struct {
	Name string `json:"n"`
	Town string `json:"t,omitempty"`
	Seen int    `json:"s"`
	Last int    `json:"l,omitempty"`
	Dead bool   `json:"d,omitempty"`
}

type lnState struct {
	Init     bool                  `json:"i,omitempty"`
	Next     int                   `json:"x,omitempty"`
	Day      int                   `json:"d"`
	Post     []*lnLetter           `json:"p,omitempty"` // on the road to you
	Box      []*lnLetter           `json:"b,omitempty"` // delivered
	Out      []*lnOut              `json:"o,omitempty"`
	Contacts map[string]*lnContact `json:"c,omitempty"`
	Comps    map[string]*lnComp    `json:"cm,omitempty"`
	Deeds    map[string]bool       `json:"dd,omitempty"`
	XPDay    int                   `json:"xd,omitempty"`
	Told     int                   `json:"to,omitempty"` // letters lost that you heard about

	view     string
	tab, sel int
	scroll   int
	cur      int
	menuT    string
	menu     []Option
	pick     []*lnContact
	to       *lnContact
	re       *lnLetter
	fromTalk bool
}

type lnlState struct {
	Last  map[string]int    `json:"l,omitempty"` // person key -> day of their last letter
	Seen  map[string]string `json:"s,omitempty"` // person key -> signature of the last life line reported
	Rep   map[string]int    `json:"r,omitempty"` // tags already reported (counts)
	Crier map[int]int       `json:"c,omitempty"` // burg -> day the crier cried
	Gen   int               `json:"g,omitempty"` // day of the last global letter (threat cap)
}

func lnSt(g *Game) *lnState {
	s := g.ext("ln", func() any { return &lnState{} }).(*lnState)
	if s.Contacts == nil {
		s.Contacts = map[string]*lnContact{}
	}
	if s.Comps == nil {
		s.Comps = map[string]*lnComp{}
	}
	if s.Deeds == nil {
		s.Deeds = map[string]bool{}
	}
	return s
}

func lnl(g *Game) *lnlState {
	s := g.ext("lnl", func() any { return &lnlState{} }).(*lnlState)
	if s.Last == nil {
		s.Last = map[string]int{}
		s.Seen = map[string]string{}
		s.Rep = map[string]int{}
		s.Crier = map[int]int{}
	}
	return s
}

func lnPlanet(g *Game) int {
	if g.W == nil {
		return 0
	}
	return g.W.uniIdx
}

func lnDeed(g *Game, key, text string) {
	s := lnSt(g)
	if s.Deeds[key] || g.Cur == nil || g.W == nil {
		return
	}
	s.Deeds[key] = true
	g.deed("life", text, 0, 0)
}

func lnPlain(s string) string { return strings.TrimSpace(Plain(s)) }

func lnLc(s string) string {
	if s == "" {
		return s
	}
	return strings.ToLower(s[:1]) + s[1:]
}

func lnGenre(g *Game) string {
	if g.Genre == "" {
		return "fantasy"
	}
	return g.Genre
}

func lnVoiceOf(g *Game) lnVoice {
	if v, ok := lnVoices[lnGenre(g)]; ok {
		return v
	}
	return lnVoices["fantasy"]
}

func lnDate(g *Game, day int) string {
	if g.W == nil {
		return fmt.Sprintf("day %d", day)
	}
	d := g.W.DateOf(g.calendarCulture(), day)
	return fmt.Sprintf("%s %s", ordinal(d.Day), d.Name)
}

func lnHash(g *Game, parts ...any) uint64 {
	return HashStr(g.Seed + ":ln:" + fmt.Sprint(parts...))
}

func lnPick(r *Rng, a []string) string { return a[r.Int(0, len(a)-1)] }

func lnBurgName(g *Game, id int) string {
	if b := g.W.burgByID[id]; b != nil {
		return b.Name
	}
	return "somewhere"
}

func lnPlayerP(g *Game) int {
	if g.W == nil || g.Cur == nil || !g.W.In(g.Cur.WX, g.Cur.WY) {
		return -1
	}
	return g.W.Idx(g.Cur.WX, g.Cur.WY)
}

func lnInTown(g *Game) *BurgData {
	if g.W == nil || g.Cur == nil || g.OnWorld || g.Cur.Kind != "town" || g.Cur.Burg <= 0 {
		return nil
	}
	return g.W.burgByID[g.Cur.Burg]
}

func (s *lnState) unread() int {
	n := 0
	for _, l := range s.Box {
		if !l.Read {
			n++
		}
	}
	return n
}

// ---------------------------------------------------------------- delivery times

// lnDelay: days a letter takes between a burg (on planet pl) and where you are.
func lnDelay(g *Game, burg, pl int, r *Rng) int {
	if pl != lnPlanet(g) {
		return 9 + 6 + r.Int(0, 6) // the rift costs a week or more
	}
	bd := g.W.burgByID[burg]
	here := lnPlayerP(g)
	if bd == nil || here < 0 {
		return 3 + r.Int(1, 5)
	}
	km := float64(wsCheb(g.W, bd.P, here)) * g.tvKm()
	d := int(km/40.0+0.999) + 1 + r.Int(0, 3)
	return maxi(1, d)
}

func lnLostChance(g *Game, burg, pl int) float64 {
	c := 0.02
	if pl != lnPlanet(g) {
		return c + 0.05
	}
	bd := g.W.burgByID[burg]
	here := lnPlayerP(g)
	if bd == nil || here < 0 {
		return c
	}
	d := wsCheb(g.W, bd.P, here)
	c += float64(mini(8, d/4)) * 0.01
	if len(g.warsOf(bd.State)) > 0 {
		c += 0.04
	}
	if bd.Bandit {
		c += 0.04
	}
	return c
}

// lnQueue sends a letter on its way to you.
func lnQueue(g *Game, l *lnLetter, burg int, r *Rng, day int) {
	s := lnSt(g)
	s.Next++
	l.ID = s.Next
	l.Sent = day
	l.Due = day + lnDelay(g, burg, l.Planet, r)
	l.Lost = r.Chance(lnLostChance(g, burg, l.Planet))
	s.Post = append(s.Post, l)
	if len(s.Post) > 40 {
		s.Post = s.Post[len(s.Post)-40:]
	}
}

// ---------------------------------------------------------------- composing

func lnTone(op int) int {
	switch {
	case op >= 40:
		return 2
	case op <= -30:
		return 0
	case op >= 15:
		return 2
	}
	return 1
}

func lnCompose(g *Game, r *Rng, tone int, name string, paras ...string) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf(lnPick(r, lnOpen[tone]), g.P.Name))
	for _, p := range paras {
		if p != "" {
			b.WriteString("\n\n" + p)
		}
	}
	sg, ok := lnSign[lnGenre(g)]
	if !ok {
		sg = lnSign[""]
	}
	b.WriteString("\n\n" + sg[tone] + "\n" + name)
	return b.String()
}

// lnFirst turns a third-person life line into first person.
func lnFirst(e lvEvent) string {
	t := strings.TrimSuffix(strings.TrimSpace(e.Text), ".")
	if t == "" {
		return ""
	}
	w := strings.SplitN(t, " ", 2)[0]
	switch w {
	case "Jailed", "Fined", "Exiled", "Robbed":
		return "I was " + lnLc(t)
	case "Courtship":
		return "My " + lnLc(t)
	case "Was":
		return "I " + lnLc(t)
	}
	return "I " + lnLc(t)
}

func lnPlace(g *Game, p *lvPerson) int {
	if p.Burg > 0 {
		return p.Burg
	}
	return p.Home
}

func lnContactOf(g *Game, p *lvPerson) *lnContact {
	b := lnPlace(g, p)
	return &lnContact{Key: p.Key, Planet: lnPlanet(g), Name: p.Name, Role: p.Role, Place: lnBurgName(g, b), Burg: b, Op: p.Opinion}
}

func lnCKey(c *lnContact) string { return strconv.Itoa(c.Planet) + ":" + c.Key }

func lnUpsert(g *Game, c *lnContact) {
	if c.Key == "" {
		return
	}
	lnSt(g).Contacts[lnCKey(c)] = c
}

func lnRefreshContacts(g *Game) {
	if g.W == nil {
		return
	}
	for _, p := range lvSt(g).People {
		if p.Met && p.Alive {
			lnUpsert(g, lnContactOf(g, p))
		}
	}
}

// ---------------------------------------------------------------- generation

type lnCtx struct {
	g    *Game
	s    *lnState
	loc  *lnlState
	day  int
	yr   int
	news []wsLine
	got  bool
	cap  bool
}

func (c *lnCtx) nearNews(p, rad int) []string {
	if !c.got {
		c.got = true
		c.news = c.g.wsNewsLines()
	}
	var out []string
	for _, l := range c.news {
		if wsCheb(c.g.W, l.P, p) <= rad {
			out = append(out, lnPlain(l.Text))
		}
	}
	return out
}

func (c *lnCtx) letter(p *lvPerson, kind string, text string) *lnLetter {
	g := c.g
	b := lnPlace(g, p)
	return &lnLetter{From: p.Key, Planet: lnPlanet(g), Name: p.Name, Role: p.Role, Place: lnBurgName(g, b), Kind: kind, Text: text}
}

func (c *lnCtx) send(p *lvPerson, l *lnLetter, r *Rng) {
	lnQueue(c.g, l, lnPlace(c.g, p), r, c.day)
	c.loc.Last[p.Key] = c.day
	lnUpsert(c.g, lnContactOf(c.g, p))
}

func lnTick(g *Game) {
	s := lnSt(g)
	d := lvDay(g)
	if !s.Init {
		s.Init, s.Day = true, d
		return
	}
	if d <= s.Day {
		return
	}
	start := maxi(s.Day+1, d-14)
	s.Day = d
	lnTrackComps(g, d)
	for day := start; day <= d; day++ {
		lnGenDay(g, day)
	}
	lnRunOut(g, d)
	lnDeliver(g, false)
}

func lnGenDay(g *Game, day int) {
	s := lnSt(g)
	c := &lnCtx{g: g, s: s, loc: lnl(g), day: day, yr: lvYear(g)}
	if len(s.Post)+s.unread() >= 8 {
		return
	}
	st := lvSt(g)
	keys := make([]string, 0, 32)
	for k := range st.People {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	lnRefreshContacts(g)
	wrote := 0
	for _, k := range keys {
		p := st.People[k]
		if p == nil || !p.Alive || wrote >= 2 {
			continue
		}
		if p.Met && day-c.loc.Last[k] >= 6 {
			if lnGenPerson(c, p) {
				wrote++
			}
		}
		if !p.Met && lnAvenger(c, p) {
			wrote++
		}
	}
	lnGenSpecial(c, keys)
}

func lnNewEvents(p *lvPerson, loc *lnlState, yr int) []lvEvent {
	sig := func(e lvEvent) string { return fmt.Sprintf("%d|%s|%s", e.Y, e.Kind, e.Text) }
	last := loc.Seen[p.Key]
	idx := -1
	for i := range p.Life {
		if sig(p.Life[i]) == last {
			idx = i
		}
	}
	start := idx + 1
	if idx < 0 {
		start = maxi(0, len(p.Life)-2)
	}
	var out []lvEvent
	for _, e := range p.Life[start:] {
		switch e.Kind {
		case "birth", "death", "secret":
			continue
		}
		if e.Y >= yr-1 {
			out = append(out, e)
		}
	}
	return out
}

func lnSig(e lvEvent) string { return fmt.Sprintf("%d|%s|%s", e.Y, e.Kind, e.Text) }

func lnGenPerson(c *lnCtx, p *lvPerson) bool {
	g := c.g
	rr := NewRng(g.Seed + ":ln:" + p.Key + ":" + strconv.Itoa(c.day))
	op := p.Opinion
	ev := lnNewEvents(p, c.loc, c.yr)
	ch := 0.004 + float64(maxi(op, 0))/100*0.012
	if len(ev) > 0 && op >= 10 {
		ch += 0.18
	}
	if op <= -35 {
		ch = 0.008
	}
	if op > -35 && op < 10 && len(ev) == 0 {
		ch = 0.001
	}
	if !rr.Chance(ch) {
		return false
	}
	tone := lnTone(op)
	here := lnBurgName(g, lnPlace(g, p))
	name := p.Name
	switch {
	case op <= -35:
		if c.loc.Rep["g:"+p.Key] >= 3 || c.day-c.loc.Gen < 5 {
			return false
		}
		c.loc.Rep["g:"+p.Key]++
		c.loc.Gen = c.day
		bad := "you know what you did."
		w := 0
		for _, m := range p.Mem {
			if m.Weight < w {
				w, bad = m.Weight, m.Text
			}
		}
		l := c.letter(p, "threat", lnCompose(g, rr, 0, name, fmt.Sprintf(lnPick(rr, lnGrudge), bad)))
		c.send(p, l, rr)
		return true
	case len(ev) > 0 && op >= 10:
		e := ev[0]
		kind := "news"
		var l *lnLetter
		intro := fmt.Sprintf(lnPick(rr, lnNewsIntro), here)
		body := lnFirst(e) + "."
		react := ""
		if rs := lnReact[e.Kind]; len(rs) > 0 {
			react = lnPick(rr, rs)
		}
		t0 := strings.SplitN(strings.TrimSuffix(e.Text, "."), " ", 2)[0]
		ask := 0
		if e.Kind == "crime" && (t0 == "Jailed" || t0 == "Fined" || t0 == "Exiled" || t0 == "Robbed") {
			kind = "plea"
			ask = []int{20, 40, 60, 90}[rr.Int(0, 3)]
			react = fmt.Sprintf(lnPick(rr, lnPleaAsk), ask)
		}
		l = c.letter(p, kind, lnCompose(g, rr, tone, name, intro, body, react))
		l.Ask = ask
		c.loc.Seen[p.Key] = lnSig(ev[len(ev)-1])
		c.send(p, l, rr)
		return true
	}
	roll := rr.Int(0, 99)
	switch {
	case roll < 22 && op >= 45 && p.Wealth >= 80:
		coin := rr.Int(5, mini(40, maxi(6, p.Wealth/8)))
		l := c.letter(p, "gift", lnCompose(g, rr, tone, name, fmt.Sprintf(lnPick(rr, lnGiftLine), coin)))
		l.Coin = coin
		c.send(p, l, rr)
	case roll < 50 && p.Wealth < 40 && op >= 20:
		ask := []int{10, 25, 50}[rr.Int(0, 2)]
		l := c.letter(p, "plea", lnCompose(g, rr, tone, name, lnPick(rr, lnPleaCause), fmt.Sprintf(lnPick(rr, lnPleaAsk), ask)))
		l.Ask = ask
		c.send(p, l, rr)
	case roll < 70 && op >= 25:
		b := g.W.burgByID[lnPlace(g, p)]
		if b == nil {
			return false
		}
		ns := c.nearNews(b.P, 8)
		if len(ns) == 0 {
			return false
		}
		l := c.letter(p, "warn", lnCompose(g, rr, tone, name, fmt.Sprintf(lnPick(rr, lnWarnIntro), lnLc(strings.TrimSuffix(ns[rr.Int(0, len(ns)-1)], ".")))))
		c.send(p, l, rr)
	case roll < 80 && op >= 35:
		bid := lnPlace(g, p)
		l := c.letter(p, "invite", "")
		l.Burg, l.IKind = bid, "feast"
		arrive := lnDelay(g, bid, l.Planet, rr)
		l.Day = c.day + arrive + rr.Int(4, 12)
		l.Text = lnCompose(g, rr, tone, name, fmt.Sprintf(lnPick(rr, lnInviteFeast), here, lnDate(g, l.Day)))
		c.send(p, l, rr)
	default:
		if op < 15 {
			return false
		}
		mood := strings.ToLower(lvMoodWord(p))
		paras := []string{fmt.Sprintf(lnPick(rr, lnNote), mood)}
		if p.Goal != "" {
			paras = append(paras, "Lately I have been working on this: "+lnLc(strings.TrimSuffix(p.Goal, "."))+".")
		}
		l := c.letter(p, "note", lnCompose(g, rr, tone, name, paras...))
		c.send(p, l, rr)
	}
	return true
}

// lnAvenger: kin of someone you killed write a threat (they need not have met you).
func lnAvenger(c *lnCtx, q *lvPerson) bool {
	g := c.g
	if q.Opinion > -25 || c.day-c.loc.Last[q.Key] < 30 || c.loc.Rep["a:"+q.Key] >= 2 || c.day-c.loc.Gen < 5 {
		return false
	}
	var vic *lvPerson
	for _, m := range q.Mem {
		if m.Kind == "kill" && strings.HasPrefix(m.Text, "You killed their") {
			for _, k := range append(append([]string{}, q.Kin...), q.Friends...) {
				if v := lvGet(g, k); v != nil && !v.Alive && strings.Contains(m.Text, v.Name) {
					vic = v
				}
			}
		}
	}
	if vic == nil {
		return false
	}
	rr := NewRng(g.Seed + ":ln:av:" + q.Key + ":" + strconv.Itoa(c.day))
	if !rr.Chance(0.03) {
		return false
	}
	c.loc.Rep["a:"+q.Key]++
	c.loc.Gen = c.day
	rel := lvKinLabel(q, vic)
	var t string
	switch rr.Int(0, 2) {
	case 0:
		t = fmt.Sprintf(lnThreatKin[0], vic.Name, rel)
	case 1:
		t = fmt.Sprintf(lnThreatKin[1], rel, vic.Name)
	default:
		t = fmt.Sprintf(lnThreatKin[2], vic.Name, rel)
	}
	l := c.letter(q, "threat", lnCompose(g, rr, 0, q.Name, t))
	c.send(q, l, rr)
	return true
}

// lnGenSpecial: invitations from courtships and deaths, rivals, the nemesis' kin, parted companions, far contacts.
func lnGenSpecial(c *lnCtx, keys []string) {
	g := c.g
	st := lvSt(g)
	day := c.day
	// weddings in the making
	ls := lsSt(g)
	var pairs []string
	for k := range ls.Court {
		pairs = append(pairs, k)
	}
	sort.Strings(pairs)
	for _, pk := range pairs {
		if ls.Court[pk] < 55 || c.loc.Rep["w:"+pk] > 0 {
			continue
		}
		ab := strings.SplitN(pk, "|", 2)
		if len(ab) != 2 {
			continue
		}
		a, b := lvGet(g, ab[0]), lvGet(g, ab[1])
		if a == nil || b == nil || !a.Alive || !b.Alive || a.Spouse != "" || b.Spouse != "" {
			continue
		}
		var w *lvPerson
		var other *lvPerson
		for _, x := range [][2]*lvPerson{{a, b}, {b, a}} {
			if x[0].Met && x[0].Opinion >= 15 {
				w, other = x[0], x[1]
				break
			}
		}
		if w == nil {
			continue
		}
		rr := NewRng(g.Seed + ":ln:wed:" + pk + ":" + strconv.Itoa(day))
		c.loc.Rep["w:"+pk]++
		bid := w.Home
		l := c.letter(w, "invite", "")
		l.Burg, l.IKind = bid, "wedding"
		l.Day = day + lnDelay(g, bid, l.Planet, rr) + rr.Int(5, 10)
		l.Text = lnCompose(g, rr, lnTone(w.Opinion), w.Name, fmt.Sprintf(lnPick(rr, lnInviteWed), other.Name, lnDate(g, l.Day), lnBurgName(g, bid)))
		c.send(w, l, rr)
		return
	}
	// deaths among kin and friends of people you know
	for _, k := range keys {
		w := st.People[k]
		if w == nil || !w.Alive || !w.Met || w.Opinion < 15 || day-c.loc.Last[k] < 4 {
			continue
		}
		for _, qk := range append(append([]string{}, w.Kin...), w.Friends...) {
			q := st.People[qk]
			tag := "d:" + k + ":" + qk
			if q == nil || q.Alive || q.Died < c.yr || c.loc.Rep[tag] > 0 {
				continue
			}
			rr := NewRng(g.Seed + ":ln:fun:" + tag)
			c.loc.Rep[tag]++
			bid := q.Home
			l := c.letter(w, "invite", "")
			l.Burg, l.IKind = bid, "funeral"
			l.Day = day + lnDelay(g, bid, l.Planet, rr) + rr.Int(2, 4)
			cause := strings.TrimSpace(q.Cause)
			if cause == "" {
				cause = "after a short illness"
			}
			if rr.Chance(0.5) {
				l.Text = lnCompose(g, rr, 1, w.Name, fmt.Sprintf(lnInviteFun[0], q.Name, cause, lnDate(g, l.Day), lnBurgName(g, bid)))
			} else {
				l.Text = lnCompose(g, rr, 1, w.Name, fmt.Sprintf(lnInviteFun[1], q.Name, cause, lnBurgName(g, bid), lnDate(g, l.Day)))
			}
			c.send(w, l, rr)
			return
		}
	}
	if c.day-c.loc.Gen < 3 {
		return
	}
	// the hunters' dossier: nemesis kin and rival adventurers, read only through its text
	rr := NewRng(g.Seed + ":ln:dos:" + strconv.Itoa(day))
	if rr.Chance(0.03) {
		lnDossierLetter(c, rr)
	}
	// parted companions
	var names []string
	for n := range c.s.Comps {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		cp := c.s.Comps[n]
		if cp.Dead || day-cp.Seen < 6 || day-cp.Last < 20 {
			continue
		}
		r2 := NewRng(g.Seed + ":ln:comp:" + n + ":" + strconv.Itoa(day))
		if !r2.Chance(0.02) {
			continue
		}
		cp.Last = day
		town := nonEmpty(cp.Town, "home")
		l := &lnLetter{Name: cp.Name, Role: "former companion", Place: town, Kind: "note", Planet: lnPlanet(g)}
		l.Text = lnCompose(g, r2, 2, cp.Name, lnPick(r2, lnCompanion), "I am in "+town+" and the door is open to you.")
		lnQueue(g, l, 0, r2, day)
		return
	}
	// contacts on other planets
	var cks []string
	for k, ct := range c.s.Contacts {
		if ct.Planet != lnPlanet(g) && ct.Op >= 25 {
			cks = append(cks, k)
		}
	}
	sort.Strings(cks)
	for _, k := range cks {
		ct := c.s.Contacts[k]
		r2 := NewRng(g.Seed + ":ln:far:" + k + ":" + strconv.Itoa(day))
		if !r2.Chance(0.006) {
			continue
		}
		l := &lnLetter{From: ct.Key, Planet: ct.Planet, Name: ct.Name, Role: ct.Role, Place: ct.Place, Kind: "note"}
		l.Text = lnCompose(g, r2, lnTone(ct.Op), ct.Name, fmt.Sprintf(lnPick(r2, lnFar), nonEmpty(ct.Place, "home")))
		lnQueue(g, l, ct.Burg, r2, day)
		return
	}
}

func lnDossierLetter(c *lnCtx, rr *Rng) {
	g := c.g
	lines := g.nxDossier()
	if len(lines) == 0 {
		return
	}
	hunt, rival := false, -1
	for i, l := range lines {
		pl := lnPlain(l)
		if strings.Contains(pl, "Hunting you") {
			hunt = true
		}
		if pl == "Rival adventurer" {
			rival = i + 1
		}
	}
	if hunt && rr.Chance(0.5) && !strings.Contains(lnPlain(lines[0]), "No one has sworn") {
		nm := lnPlain(lines[0])
		l := &lnLetter{Name: "Kin of " + nm, Role: "unknown hand", Kind: "threat", Planet: lnPlanet(g)}
		l.Text = lnCompose(g, rr, 0, "one who speaks for "+nm, fmt.Sprintf(lnPick(rr, lnNemKin), nm))
		lnQueue(g, l, 0, rr, c.day)
		c.loc.Gen = c.day
		return
	}
	if rival >= 0 && rival < len(lines) {
		first := lnPlain(lines[rival])
		name := strings.SplitN(first, ",", 2)[0]
		mood := "wary"
		for _, l := range lines[rival:] {
			pl := lnPlain(l)
			if i := strings.Index(pl, "Mood:"); i >= 0 {
				for _, m := range []string{"friendly", "hostile", "wary"} {
					if strings.Contains(pl[i:], m) {
						mood = m
					}
				}
				break
			}
		}
		if name == "" || strings.Contains(first, "Dead") {
			return
		}
		tone := map[string]int{"friendly": 2, "wary": 1, "hostile": 0}[mood]
		l := &lnLetter{Name: name, Role: "rival adventurer", Kind: "taunt", Planet: lnPlanet(g)}
		l.Text = lnCompose(g, rr, tone, name, lnPick(rr, lnRivalLines[mood]))
		lnQueue(g, l, 0, rr, c.day)
		c.loc.Gen = c.day
	}
}

func lnTrackComps(g *Game, day int) {
	if g.Cur == nil {
		return
	}
	s := lnSt(g)
	for _, a := range g.Cur.Actors {
		if a.Comp == nil || a.HP <= 0 {
			continue
		}
		cp := s.Comps[a.Name]
		if cp == nil {
			cp = &lnComp{Name: a.Name}
			s.Comps[a.Name] = cp
		}
		cp.Seen, cp.Dead = day, false
		if a.Comp.Town != "" {
			cp.Town = a.Comp.Town
		}
	}
}

func lnKill(g *Game, d, src *Actor) {
	if d == nil || d.Comp == nil {
		return
	}
	if cp := lnSt(g).Comps[d.Name]; cp != nil {
		cp.Dead = true
	}
}

// ---------------------------------------------------------------- delivery and outgoing replies

func lnDeliver(g *Game, force bool) int {
	bd := lnInTown(g)
	s := lnSt(g)
	if bd == nil || len(s.Post) == 0 {
		return 0
	}
	d := lvDay(g)
	var keep []*lnLetter
	n := 0
	lost := 0
	for _, l := range s.Post {
		if l.Due > d {
			keep = append(keep, l)
			continue
		}
		if l.Lost {
			lost++
			continue
		}
		l.Got = d
		s.Box = append(s.Box, l)
		n++
		if l.From != "" {
			lnUpsert(g, &lnContact{Key: l.From, Planet: l.Planet, Name: l.Name, Role: l.Role, Place: l.Place, Op: 20})
		}
	}
	s.Post = keep
	if len(s.Box) > 60 {
		s.Box = s.Box[len(s.Box)-60:]
	}
	if n > 0 {
		g.Msg(fmt.Sprintf("{{Y|A %s presses %d letter%s into your hand at %s.}} {{K|(open the Mail ability)}}", lnCourierWord(g), n, map[bool]string{true: "s", false: ""}[n > 1], bd.Name))
		lnDeed(g, "first", fmt.Sprintf("The wanderer %s received a first letter, carried by messenger to %s.", g.P.Name, bd.Name))
	}
	if lost > 0 && s.Told < 6 {
		s.Told++
		g.Msg("{{y|You hear a courier was robbed on the road. Some post for you may never come.}}")
		lnDeed(g, "lost", fmt.Sprintf("The wanderer %s lost a letter to the dangers of the road.", g.P.Name))
	}
	return n
}

func lnCourierWord(g *Game) string {
	switch lnGenre(g) {
	case "cyberpunk":
		return "courier-runner"
	case "wasteland":
		return "dust-runner"
	case "steampunk", "dieselpunk":
		return "post-boy"
	}
	return "courier"
}

func lnRunOut(g *Game, day int) {
	s := lnSt(g)
	if len(s.Out) == 0 || g.W == nil {
		return
	}
	var keep []*lnOut
	for _, o := range s.Out {
		if o.Done && day-o.Due > 30 {
			continue
		}
		if !o.Done && o.Due <= day && o.Planet == lnPlanet(g) {
			lnApplyOut(g, o, day)
		}
		keep = append(keep, o)
	}
	if len(keep) > 40 {
		keep = keep[len(keep)-40:]
	}
	s.Out = keep
}

func lnApplyOut(g *Game, o *lnOut, day int) {
	o.Done = true
	p := lvGet(g, o.To)
	if p == nil || !p.Alive {
		if o.To != "" {
			if o.Coin > 0 {
				g.P.Gold += o.Coin
				g.Msg(fmt.Sprintf("{{y|Your letter to %s came back unopened; the %d coin you sent is returned.}}", o.Name, o.Coin))
			} else {
				g.Msg(fmt.Sprintf("{{y|Your letter to %s came back unopened.}}", o.Name))
			}
		}
		return
	}
	memo := "You wrote to them."
	switch o.Kind {
	case "thanks":
		memo = "You wrote to thank them."
	case "promise":
		memo = "You promised to help."
	case "decline":
		memo = "You declined their request."
	case "coin":
		memo = "You sent them coin."
	case "gift":
		memo = "You sent them a gift."
	case "rumour":
		memo = "You sent them news."
	case "accept":
		memo = "You promised to come."
	case "regret":
		memo = "You sent your regrets."
	case "apology":
		memo = "You apologised."
	case "defy":
		memo = "You answered them with scorn."
	}
	p.remember(day, "letter", memo, o.W)
	p.Mood = clampi(p.Mood+o.W/2, -100, 100)
	if o.Coin > 0 {
		p.Wealth += o.Coin
	}
	if o.Val > 0 {
		p.Wealth += o.Val / 2
	}
	lvCheckFriend(g, p)
	if rr := NewRng(g.Seed + ":ln:ans:" + strconv.Itoa(o.ID)); rr.Chance(0.45) && o.Kind != "defy" && o.Kind != "regret" {
		key := o.Kind
		if len(lnAnswer[key]) == 0 {
			key = "thanks"
		}
		b := lnPlace(g, p)
		l := &lnLetter{From: p.Key, Planet: o.Planet, Name: p.Name, Role: p.Role, Place: lnBurgName(g, b), Kind: "reply"}
		l.Text = lnCompose(g, rr, lnTone(p.Opinion), p.Name, lnPick(rr, lnAnswer[key]))
		lnQueue(g, l, b, rr, day)
	}
}

// ---------------------------------------------------------------- hooks

func lnTurn(g *Game) {
	if g.W == nil || g.P == nil || g.Cur == nil {
		return
	}
	lnTick(g)
	if g.Turn%30 == 0 {
		lnDeliver(g, false)
	}
}

func lnEnter(g *Game, zn *Zone) {
	if g.W == nil || g.P == nil || zn == nil {
		return
	}
	lnTick(g)
	lnTrackComps(g, lvDay(g))
	lnRunOut(g, lvDay(g))
	lnDeliver(g, false)
	bd := lnInTown(g)
	if bd == nil {
		return
	}
	loc := lnl(g)
	day := lvDay(g)
	for _, a := range zn.Actors {
		if a.Role == "crier" && loc.Crier[bd.ID] != day+1 {
			loc.Crier[bd.ID] = day + 1
			items := g.lnNewsFor(bd, day)
			v := lnVoiceOf(g)
			r := NewRng(fmt.Sprintf("%s:ln:cry:%d:%d", g.Seed, bd.ID, day))
			if len(items) == 0 {
				g.Msg("{{K|" + lnPick(r, v.Quiet) + "}}")
			} else {
				it := items[int(lnHash(g, "cry", bd.ID, day)%uint64(mini(len(items), 4)))]
				g.Msg("{{Y|" + fmt.Sprintf(lnPick(r, v.Cry), lnPlain(it.Text)) + "}}")
			}
			lnDeed(g, "crier", fmt.Sprintf("The wanderer %s heard the news cried in the plaza of %s.", g.P.Name, bd.Name))
			break
		}
	}
}

func lnSide(g *Game) []string {
	if g.W == nil || g.P == nil {
		return nil
	}
	if n := lnSt(g).unread(); n > 0 {
		return []string{fmt.Sprintf("{{Y|Mail: %d unread}}", n)}
	}
	return nil
}

func lnZone(g *Game, zn *Zone, _ *Rng) {
	if g.W == nil || zn.Kind != "town" || zn.Z != 0 || zn.Burg <= 0 {
		return
	}
	bd := g.W.burgByID[zn.Burg]
	if bd == nil || !bd.Alive {
		return
	}
	if !(strings.Contains(zn.Name, "Market") || strings.Contains(zn.Name, "Harbor") || !strings.Contains(zn.Name, ",")) {
		return
	}
	v := lnVoiceOf(g)
	tier := g.W.cityTier[bd.ID]
	h := func(s string) uint64 { return HashStr(fmt.Sprintf("%s:ln:place:%s:%d", g.Seed, s, bd.ID)) % 100 }
	if h("crier") < 45 {
		g.trPlace(zn, bd, "crier", v.Crier, fmt.Sprintf("The %s of %s, who calls the news in the plaza.", v.Crier, bd.Name), "")
	}
	if tier >= 1 {
		g.trPlace(zn, bd, "printer", v.Printer, fmt.Sprintf("The %s of %s, who sells the daily %s.", v.Printer, bd.Name, v.Unit), "")
	}
	if tier >= 1 || h("post") < 30 {
		g.trPlace(zn, bd, "postmaster", v.Post, fmt.Sprintf("The %s of %s, who takes and holds the post.", v.Post, bd.Name), "")
	}
	for _, a := range zn.Actors {
		switch a.Role {
		case "crier":
			a.F = CYellow
		case "printer":
			a.F = CWhite
		case "postmaster":
			a.F = CCyan
		}
	}
}

func lnIsPost(a *Actor) bool {
	switch a.Role {
	case "postmaster", "clerk", "messenger", "courier":
		return true
	}
	return false
}

func lnTalk(g *Game, a *Actor) []Option {
	if g.W == nil || g.P == nil || a == nil {
		return nil
	}
	var out []Option
	bd := lnInTown(g)
	v := lnVoiceOf(g)
	if lnIsPost(a) {
		out = append(out,
			Option{"Ask whether there is post for you.", func() { g.talkMenu(g.lnPostTalk()) }},
			Option{"Read the letters I have received.", func() { g.lnOpen(true, "list") }},
			Option{"Send a letter to someone I know.", func() { g.lnOpen(true, "people") }})
	}
	if a.Role == "crier" && bd != nil {
		out = append(out, Option{"Ask the " + v.Crier + " what is new.", func() { g.talkMenu(g.lnCrierTalk(bd, a)) }})
	}
	if a.Role == "printer" && bd != nil {
		day := lvDay(g)
		price := 3 + g.W.cityTier[bd.ID]*2
		out = append(out, Option{fmt.Sprintf("Buy today's %s. {{K|(%s)}}", v.Unit, g.cuFormat(price)), func() { g.talkMenu(g.lnBuyPaper(a, bd, day, price)) }},
			Option{"Read the " + strings.ToLower(v.Board) + ".", func() { g.talkMenu(g.lnBoardText(bd, day)) }})
	}
	return out
}

func (g *Game) lnPostTalk() string {
	s := lnSt(g)
	n := lnDeliver(g, true)
	pend := len(s.Post)
	un := s.unread()
	switch {
	case n > 0:
		return fmt.Sprintf("\"Here you are. %d letter%s came for you.\" (%d unread. Open the Mail ability or ask to read them.)", n, map[bool]string{true: "s", false: ""}[n > 1], un)
	case un > 0:
		return fmt.Sprintf("\"Nothing new, but you still have %d unread.\"", un)
	case pend > 0:
		return "\"Nothing yet. Post travels slowly. Some may be on the road.\""
	}
	return "\"Nothing for you today.\""
}

// ---------------------------------------------------------------- the crier and the paper

type lnItem struct {
	Text string
	P    int
}

// lnNewsFor: what is worth telling in a burg today, nearest first. Pure of (seed, world state, day).
func (g *Game) lnNewsFor(bd *BurgData, day int) []lnItem {
	type cand struct {
		lnItem
		d int
	}
	seen := map[string]bool{}
	var cs []cand
	add := func(text string, p, d int) {
		k := lnPlain(text)
		if k == "" || seen[k] {
			return
		}
		seen[k] = true
		cs = append(cs, cand{lnItem{text, p}, d})
	}
	for _, foe := range g.warsOf(bd.State) {
		if len(cs) >= 2 {
			break
		}
		add(fmt.Sprintf("{{R|War}}: %s is at war with %s", g.realmName(bd.State), g.realmName(foe)), bd.P, 1)
	}
	for _, l := range g.wsNewsLines() {
		if d := wsCheb(g.W, l.P, bd.P); d <= 14 {
			add(l.Text, l.P, d)
		}
	}
	for _, n := range lvSt(g).News {
		if day-n.Day > 20 {
			continue
		}
		p, d := bd.P, 0
		if b := g.W.burgByID[n.Burg]; b != nil {
			p = b.P
			d = wsCheb(g.W, b.P, bd.P)
		}
		if d <= 10 {
			add(n.Text, p, d)
		}
	}
	sort.SliceStable(cs, func(i, j int) bool {
		if cs[i].d != cs[j].d {
			return cs[i].d < cs[j].d
		}
		return lnPlain(cs[i].Text) < lnPlain(cs[j].Text)
	})
	out := make([]lnItem, 0, len(cs))
	for _, c := range cs {
		out = append(out, c.lnItem)
	}
	return out
}

func (g *Game) lnCrierTalk(bd *BurgData, a *Actor) string {
	day := lvDay(g)
	v := lnVoiceOf(g)
	items := g.lnNewsFor(bd, day)
	if len(items) == 0 {
		return "\"" + lnPick(NewRng(fmt.Sprintf("%s:ln:q:%d:%d", g.Seed, bd.ID, day)), v.Quiet) + "\""
	}
	off := int(lnHash(g, "crt", bd.ID, day) % uint64(len(items)))
	var lines []string
	for i := 0; i < mini(4, len(items)); i++ {
		lines = append(lines, "- "+strings.TrimSuffix(items[(off+i)%len(items)].Text, "."))
	}
	lnDeed(g, "crier", fmt.Sprintf("The wanderer %s heard the news cried in the plaza of %s.", g.P.Name, bd.Name))
	return "\"Today in " + bd.Name + " and round about:\"\n\n" + strings.Join(lines, "\n")
}

func lnNameIn(text string, names []string) string {
	for _, n := range names {
		if len(n) > 3 && strings.Contains(text, n) {
			return n
		}
	}
	return ""
}

// lnFalsify turns a true line into a false or outdated one.
func (g *Game) lnFalsify(r *Rng, text string, bd *BurgData) string {
	var names []string
	for _, b := range g.ws().burgList(g) {
		if wsCheb(g.W, b.P, bd.P) <= 10 {
			names = append(names, b.Name)
		}
	}
	sort.Strings(names)
	if n := lnNameIn(text, names); n != "" && len(names) > 1 {
		for i := 0; i < 8; i++ {
			if o := names[r.Int(0, len(names)-1)]; o != n {
				return strings.Replace(text, n, o, 1)
			}
		}
	}
	for i, ch := range text {
		if ch >= '2' && ch <= '9' {
			return text[:i] + string(rune('0'+(int(ch-'0')*3)%9+1)) + "0" + text[i+1:]
		}
	}
	if len(names) > 0 {
		return fmt.Sprintf("They say %s is under siege, though nobody has seen the army", names[r.Int(0, len(names)-1)])
	}
	return "They say a great army is on the road, though nobody has seen it"
}

type lnPaperPart struct {
	hdr   string
	lines []string
}

func (g *Game) lnPaperParts(bd *BurgData, day int) (parts []lnPaperPart, fakes int, secret string) {
	v := lnVoiceOf(g)
	items := g.lnNewsFor(bd, day)
	yr := lvYear(g)
	st := lvSt(g)

	// headlines
	var hl []string
	n := mini(3, len(items))
	for i := 0; i < n; i++ {
		hl = append(hl, strings.TrimSuffix(items[i].Text, ".")+".")
		if secret == "" {
			secret = lnPlain(hl[len(hl)-1])
		}
	}
	if len(hl) == 0 {
		hl = append(hl, "{{K|Nothing of note is reported today.}}")
	}
	parts = append(parts, lnPaperPart{v.Head, hl})

	// market
	var mk []string
	goods := g.tqGoodsHere()
	if len(goods) > 0 {
		off := int(lnHash(g, "mk", bd.ID, day/2) % uint64(len(goods)))
		for i := 0; i < mini(4, len(goods)); i++ {
			d := goods[(off+i)%len(goods)]
			mk = append(mk, fmt.Sprintf("%s: buy %s, sell %s.", g.tqName(d), g.cuFormat(g.tqPrice(bd, d, true)), g.cuFormat(g.tqPrice(bd, d, false))))
		}
	}
	for _, l := range g.tqNews() {
		if wsCheb(g.W, l.P, bd.P) <= 12 && len(mk) < 6 {
			mk = append(mk, l.Text)
		}
	}
	if len(mk) > 0 {
		parts = append(parts, lnPaperPart{v.Market, mk})
	}

	// deaths, weddings and births in the region
	var keys []string
	for k := range st.People {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	near := func(p *lvPerson) bool {
		b := g.W.burgByID[p.Home]
		return b != nil && wsCheb(g.W, b.P, bd.P) <= 8
	}
	var ob, wd []string
	for _, k := range keys {
		p := st.People[k]
		if !p.Alive && p.Died >= yr-1 && near(p) && len(ob) < 3 {
			cause := nonEmpty(p.Cause, "of age")
			ob = append(ob, fmt.Sprintf("%s, %s of %s, %s. Aged %d.", p.Name, lnAn(p.Role), lnBurgName(g, p.Home), cause, p.lvAge(yr)))
		}
		if p.Alive && p.Spouse != "" && k < p.Spouse && near(p) && len(wd) < 3 {
			for _, e := range p.Life {
				if e.Kind == "marriage" && e.Y >= yr-1 && strings.HasPrefix(e.Text, "Married") {
					wd = append(wd, fmt.Sprintf("%s and %s are wed in %s. Our good wishes.", p.Name, lnName(g, p.Spouse), lnBurgName(g, p.Home)))
					break
				}
			}
		}
		if len(wd) < 3 {
			for _, e := range p.Life {
				if e.Kind == "birth" && e.Y >= yr && near(p) && p.Born >= yr-1 {
					wd = append(wd, fmt.Sprintf("A child, %s, is born in %s.", p.Name, lnBurgName(g, p.Home)))
					break
				}
			}
		}
	}
	if len(ob) > 0 {
		parts = append(parts, lnPaperPart{v.Obit, ob})
	}
	if len(wd) > 0 {
		parts = append(parts, lnPaperPart{v.Wed, wd})
	}

	// rumours: real facts, then possibly false or stale lines
	var ru []string
	kn := knSt(g)
	var ids []int
	for id, f := range kn.Facts {
		if (f.Kind == "rumour" || f.Kind == "event") && day-f.Day < 40 && f.Subject != "player" && f.Lvl == 0 {
			ids = append(ids, id)
		}
	}
	sort.Ints(ids)
	for _, id := range ids {
		f := kn.Facts[id]
		if b := g.W.burgByID[f.Burg]; b != nil && wsCheb(g.W, b.P, bd.P) > 12 {
			continue
		}
		if len(ru) >= 2 {
			break
		}
		ru = append(ru, "They say "+lnLc(strings.TrimSuffix(f.Text, "."))+".")
		if !f.True {
			fakes++
		}
	}
	pool := items
	if len(pool) > 3 {
		pool = pool[3:]
	}
	var old []lvNews
	for _, nw := range st.News {
		if day-nw.Day > 25 {
			old = append(old, nw)
		}
	}
	for i := 0; len(ru) < 3; i++ {
		r := NewRng(fmt.Sprintf("%s:ln:paper:%d:%d:%d", g.Seed, bd.ID, day, i))
		var base string
		switch {
		case i < len(pool):
			base = lnPlain(pool[i].Text)
		case len(old) > 0 && i-len(pool) < len(old):
			base = lnPlain(old[(i-len(pool))%len(old)].Text)
			ru = append(ru, "{{K|Word is that}} "+lnLc(strings.TrimSuffix(base, "."))+".")
			fakes++ // stale: a month old, printed as new
			continue
		default:
			base = ""
		}
		if base == "" {
			break
		}
		if r.Chance(0.25) {
			base = lnPlain(g.lnFalsify(r, base, bd))
			fakes++
		}
		ru = append(ru, "They say "+lnLc(strings.TrimSuffix(base, "."))+".")
	}
	if len(ru) > 0 {
		parts = append(parts, lnPaperPart{v.Rumour, ru})
	}

	// notice board
	parts = append(parts, lnPaperPart{v.Board, g.lnBoardLines(bd, day)})
	return
}

func lnAn(role string) string {
	if role == "" {
		role = "townsperson"
	}
	return lvAn(role)
}

func lnName(g *Game, key string) string { return lvName(g, key) }

func (g *Game) lnBoardLines(bd *BurgData, day int) []string {
	var out []string
	if b := g.nxBounty(bd.State); b > 0 {
		out = append(out, fmt.Sprintf("{{R|Wanted}}: the wanderer %s, by order of %s. Reward %s.", g.P.Name, g.realmName(bd.State), g.cuFormat(b)))
	}
	for _, t := range lsSt(g).Vacant[bd.ID] {
		if len(out) >= 3 {
			break
		}
		out = append(out, "To let: the "+t+"'s shop stands empty. Apply within.")
	}
	n := 0
	for _, c := range g.tqOffers(bd) {
		if n >= 2 || g.tq().Taken[c.ID] {
			continue
		}
		n++
		out = append(out, g.tqContractText(c))
	}
	gen := []string{"Lost: a good dog, answers to nothing. Reward offered.", "Wanted: hands for the next harvest. Fair pay.", "Found: a purse. Describe the contents to claim it.", "Wanted: a guard for the night watch. Ask at the gate.", "For sale: a cart, one careful owner, one broken wheel.", "Wanted: a tutor for three unruly children.", "Notice: the well by the market is to be cleaned; drink from the other."}
	r := NewRng(fmt.Sprintf("%s:ln:board:%d:%d", g.Seed, bd.ID, day))
	for len(out) < 4 {
		out = append(out, lnPick(r, gen))
	}
	return out
}

func (g *Game) lnPaperText(bd *BurgData, day int) (string, int, string) {
	v := lnVoiceOf(g)
	parts, fakes, secret := g.lnPaperParts(bd, day)
	var b strings.Builder
	b.WriteString(fmt.Sprintf("{{Y|%s}}\n%s, %s\n", strings.ToUpper(v.Paper), bd.Name, lnDate(g, day)))
	for _, p := range parts {
		b.WriteString("\n{{c|" + strings.ToUpper(p.hdr) + "}}\n")
		for _, l := range p.lines {
			b.WriteString("- " + l + "\n")
		}
	}
	return b.String(), fakes, secret
}

func (g *Game) lnBoardText(bd *BurgData, day int) string {
	v := lnVoiceOf(g)
	return "{{c|" + strings.ToUpper(v.Board) + "}}\n\n- " + strings.Join(g.lnBoardLines(bd, day), "\n- ")
}

func (g *Game) lnBuyPaper(a *Actor, bd *BurgData, day, price int) string {
	v := lnVoiceOf(g)
	for _, it := range g.P.Inv {
		if it.Kind == "book" && it.Extra != nil && it.Extra["ln:burg"] == bd.ID+1 && it.Extra["ln:day"] == day+1 {
			return "\"You already have today's. Read it before you buy another.\""
		}
	}
	if !g.cuSpend(price) {
		return "\"That is " + g.cuFormat(price) + ", friend.\""
	}
	text, fakes, secret := g.lnPaperText(bd, day)
	it := g.MakeBook(fmt.Sprintf("%s of %s, %s", v.Paper, bd.Name, lnDate(g, day)), text)
	it.Value = 1
	it.Extra = map[string]int{"ln:burg": bd.ID + 1, "ln:day": day + 1, "ln:fake": fakes}
	g.give(it)
	g.Msg(fmt.Sprintf(v.Sold, v.Unit))
	lnDeed(g, "paper", fmt.Sprintf("The wanderer %s bought a first %s in %s.", g.P.Name, v.Unit, bd.Name))
	s := lnSt(g)
	extra := ""
	if s.XPDay != day+1 {
		s.XPDay = day + 1
		g.gainXP(3 + g.W.cityTier[bd.ID])
		extra = " You skim it over breakfast."
		if secret != "" && lnHash(g, "sec", bd.ID, day)%100 < 35 {
			if g.addSecret(secret, "news", bd.P, g.W.Year) {
				g.Msg("{{c|Something in the paper is worth remembering.}}")
			}
		}
	}
	return "\"Today's, hot off the press.\" It is in your pack." + extra
}

// ---------------------------------------------------------------- exports

// lnInvite is an open invitation, read by the family and festival code.
type lnInvite struct {
	From string
	Burg int
	Day  int
	Kind string // wedding funeral feast
}

// lnInvites lists the invitations you have received that are still ahead and not declined.
func (g *Game) lnInvites() []lnInvite {
	if g.W == nil || g.P == nil {
		return nil
	}
	s := lnSt(g)
	d := lvDay(g)
	var out []lnInvite
	for _, l := range s.Box {
		if l.Kind != "invite" || l.Decl || l.Planet != lnPlanet(g) || l.Day < d {
			continue
		}
		out = append(out, lnInvite{From: l.From, Burg: l.Burg, Day: l.Day, Kind: l.IKind})
	}
	return out
}

// ---------------------------------------------------------------- the Mail screen

func (g *Game) lnAbilities() []Ability {
	if g.Cur == nil || g.P == nil || g.W == nil {
		return nil
	}
	name := "Mail"
	if n := lnSt(g).unread(); n > 0 {
		name = fmt.Sprintf("Mail ({{Y|%d unread}})", n)
	}
	return []Ability{{ID: "ln_mail", Name: name, Target: "self"}}
}

func (g *Game) lnAbUse(id string) {
	if id == "ln_mail" {
		g.lnOpen(false, "list")
	}
}

func (g *Game) lnOpen(fromTalk bool, view string) {
	s := lnSt(g)
	s.fromTalk = fromTalk && g.Talk != nil
	s.view, s.tab, s.sel, s.scroll, s.menu, s.re, s.to = view, 0, 0, 0, nil, nil, nil
	lnRefreshContacts(g)
	if view == "people" {
		lnPickPeople(g)
		if len(s.pick) == 0 {
			s.view = "list"
		}
	}
	g.Mode = "ln"
}

func lnPickPeople(g *Game) {
	s := lnSt(g)
	s.pick = nil
	var ks []string
	for k := range s.Contacts {
		ks = append(ks, k)
	}
	sort.Strings(ks)
	for _, k := range ks {
		if c := s.Contacts[k]; c.Planet != lnPlanet(g) || lvGet(g, c.Key) == nil || lvGet(g, c.Key).Alive {
			s.pick = append(s.pick, c)
		}
	}
	sort.SliceStable(s.pick, func(i, j int) bool {
		if s.pick[i].Place != s.pick[j].Place {
			return s.pick[i].Place < s.pick[j].Place
		}
		return s.pick[i].Name < s.pick[j].Name
	})
}

func lnClose(g *Game) {
	s := lnSt(g)
	if s.fromTalk && g.Talk != nil {
		s.fromTalk = false
		g.Mode = "talk"
		g.talkMenu(g.greeting(g.Talk))
		return
	}
	g.Mode = "main"
}

func (s *lnState) inbox() []*lnLetter {
	l := append([]*lnLetter(nil), s.Box...)
	sort.SliceStable(l, func(i, j int) bool { return l[i].Got > l[j].Got || (l[i].Got == l[j].Got && l[i].ID > l[j].ID) })
	return l
}

func (s *lnState) byID(id int) *lnLetter {
	for _, l := range s.Box {
		if l.ID == id {
			return l
		}
	}
	return nil
}

func lnMove(sel *int, d, n int) {
	if n == 0 {
		*sel = 0
		return
	}
	*sel = clampi(*sel+d, 0, n-1)
}

func lnKeys(g *Game, k string) {
	s := lnSt(g)
	nav := func(n int) {
		switch k {
		case "ArrowDown", "N2", "j":
			lnMove(&s.sel, 1, n)
		case "ArrowUp", "N8", "k":
			lnMove(&s.sel, -1, n)
		case "PageDown":
			lnMove(&s.sel, 8, n)
		case "PageUp":
			lnMove(&s.sel, -8, n)
		}
	}
	switch s.view {
	case "read":
		l := s.byID(s.cur)
		switch k {
		case "Escape", "q", "Backspace":
			s.view = "list"
		case "ArrowDown", "N2", "j":
			s.scroll++
		case "ArrowUp", "N8", "k":
			s.scroll = maxi(0, s.scroll-1)
		case "PageDown":
			s.scroll += 12
		case "PageUp":
			s.scroll = maxi(0, s.scroll-12)
		case "r":
			if l != nil {
				lnReplyMenu(g, &lnContact{Key: l.From, Planet: l.Planet, Name: l.Name, Role: l.Role, Place: l.Place, Burg: l.Burg}, l)
			}
		case "d":
			if l != nil {
				lnDelete(s, l)
				s.view = "list"
			}
		}
	case "menu":
		nav(len(s.menu))
		switch k {
		case "Escape", "q":
			s.view = "list"
		case "Enter", " ":
			if s.sel >= 0 && s.sel < len(s.menu) {
				s.menu[s.sel].Do()
			}
		}
	case "people":
		nav(len(s.pick))
		switch k {
		case "Escape", "q":
			s.view = "list"
		case "Enter", " ":
			if s.sel >= 0 && s.sel < len(s.pick) {
				lnReplyMenu(g, s.pick[s.sel], nil)
			}
		}
	default:
		var n int
		if s.tab == 0 {
			n = len(s.Box)
		} else {
			n = len(s.Out) + len(s.Post)
		}
		nav(n)
		switch k {
		case "Escape", "q":
			lnClose(g)
		case "Tab", "ArrowRight", "ArrowLeft":
			s.tab, s.sel = 1-s.tab, 0
		case "w":
			lnPickPeople(g)
			s.view, s.sel = "people", 0
		case "Enter", " ":
			if s.tab == 0 {
				if in := s.inbox(); s.sel >= 0 && s.sel < len(in) {
					lnOpenLetter(g, in[s.sel])
				}
			}
		case "r":
			if s.tab == 0 {
				if in := s.inbox(); s.sel >= 0 && s.sel < len(in) {
					l := in[s.sel]
					lnReplyMenu(g, &lnContact{Key: l.From, Planet: l.Planet, Name: l.Name, Role: l.Role, Place: l.Place, Burg: l.Burg}, l)
				}
			}
		case "d":
			if s.tab == 0 {
				if in := s.inbox(); s.sel >= 0 && s.sel < len(in) {
					lnDelete(s, in[s.sel])
					lnMove(&s.sel, 0, len(s.Box))
				}
			}
		}
	}
}

func lnDelete(s *lnState, l *lnLetter) {
	for i, o := range s.Box {
		if o == l {
			s.Box = append(s.Box[:i], s.Box[i+1:]...)
			return
		}
	}
}

func lnOpenLetter(g *Game, l *lnLetter) {
	s := lnSt(g)
	s.view, s.cur, s.scroll = "read", l.ID, 0
	l.Read = true
	if !l.Taken && (l.Coin > 0 || l.Gift != "") {
		l.Taken = true
		if l.Coin > 0 {
			g.P.Gold += l.Coin
			g.Msg(fmt.Sprintf("{{G|Enclosed with the letter: %s.}}", g.cuFormat(l.Coin)))
			lnDeed(g, "coin", fmt.Sprintf("The wanderer %s received coin in the post from %s.", g.P.Name, l.Name))
		}
	}
	if l.Kind == "invite" {
		lnDeed(g, "invite", fmt.Sprintf("The wanderer %s was invited by letter to a %s.", g.P.Name, nonEmpty(l.IKind, "gathering")))
	}
}

// lnReplyMenu opens the stock reply menu. re is the letter being answered, or nil when writing fresh.
func lnReplyMenu(g *Game, c *lnContact, re *lnLetter) {
	s := lnSt(g)
	s.to, s.re = c, re
	var opts []Option
	add := func(t string, f func()) { opts = append(opts, Option{t, f}) }
	send := func(kind string, coin int, gift *Item, rum string) func() {
		return func() { lnSend(g, c, re, kind, coin, gift, rum) }
	}
	switch {
	case re != nil && re.Kind == "invite":
		add("Promise to come.", send("accept", 0, nil, ""))
		add("Send your regrets.", send("regret", 0, nil, ""))
	case re != nil && re.Kind == "threat":
		add("Apologise.", send("apology", 0, nil, ""))
		add("Answer with scorn.", send("defy", 0, nil, ""))
	default:
		if re != nil {
			add("Thank them.", send("thanks", 0, nil, ""))
			add("Promise help.", send("promise", 0, nil, ""))
			if re.Kind == "plea" {
				add("Decline politely.", send("decline", 0, nil, ""))
				if re.Ask > 0 {
					add(fmt.Sprintf("Send the %d coin they ask.", re.Ask), send("coin", re.Ask, nil, ""))
				}
			}
		} else {
			add("Send greetings.", send("greet", 0, nil, ""))
		}
		for _, n := range []int{10, 50, 200} {
			if re != nil && re.Ask == n {
				continue
			}
			add(fmt.Sprintf("Send %d coin.", n), send("coin", n, nil, ""))
		}
		add("Send a gift from your pack.", func() { lnGiftMenu(g, c, re) })
		if len(g.Secrets) > 0 {
			add("Tell them a rumour you have heard.", func() {
				sc := g.Secrets[int(lnHash(g, "rum", lvDay(g), len(g.Secrets))%uint64(len(g.Secrets)))]
				lnSend(g, c, re, "rumour", 0, nil, sc.Text)
			})
		}
	}
	add("Never mind.", func() { s.view = "list" })
	s.menu, s.menuT, s.sel, s.view = opts, "Reply to "+c.Name, 0, "menu"
}

func lnGiftMenu(g *Game, c *lnContact, re *lnLetter) {
	s := lnSt(g)
	var opts []Option
	for _, it := range g.P.Inv {
		it := it
		if g.P.Equipped(it) != nil || it.Kind == "artifact" || it.Kind == "letter" {
			continue
		}
		if len(opts) >= 14 {
			break
		}
		opts = append(opts, Option{it.Display(), func() { lnSend(g, c, re, "gift", 0, it, "") }})
	}
	if len(opts) == 0 {
		g.Msg("{{K|You have nothing fit to send.}}")
		lnReplyMenu(g, c, re)
		return
	}
	opts = append(opts, Option{"Never mind.", func() { lnReplyMenu(g, c, re) }})
	s.menu, s.menuT, s.sel, s.view = opts, "Send which gift to "+c.Name+"?", 0, "menu"
}

var lnWeights = map[string]int{"thanks": 3, "greet": 1, "promise": 5, "decline": -2, "accept": 4, "regret": -2, "apology": 5, "defy": -8, "rumour": 2}

func lnSend(g *Game, c *lnContact, re *lnLetter, kind string, coin int, gift *Item, rum string) {
	s := lnSt(g)
	day := lvDay(g)
	if coin > 0 && !g.cuSpend(coin) {
		g.Msg("{{y|You cannot afford that.}}")
		return
	}
	o := &lnOut{To: c.Key, Planet: c.Planet, Name: c.Name, Kind: kind, Coin: coin, Rumour: rum, Sent: day, Burg: c.Burg}
	w := lnWeights[kind]
	if kind == "decline" && re != nil && re.Kind == "plea" {
		w = -6
	}
	if coin > 0 {
		w = clampi(2+coin/12, 3, 18)
		if re != nil && re.Kind == "plea" && coin >= re.Ask {
			w += 4
		}
	}
	if gift != nil {
		o.Val = gift.Value
		o.Gift = gift.Display()
		w = clampi(2+gift.Value/12, 2, 14)
		if gift.Stack() && gift.Count > 1 {
			gift.Count--
		} else {
			g.removeInv(gift)
		}
	}
	o.W = w
	s.Next++
	o.ID = s.Next
	r := NewRng(g.Seed + ":ln:out:" + strconv.Itoa(o.ID))
	o.Due = day + lnDelay(g, c.Burg, c.Planet, r)
	s.Out = append(s.Out, o)
	if re != nil {
		re.Replied = true
		if kind == "regret" {
			re.Decl = true
		}
	}
	what := lnReplyText[kind]
	if o.Gift != "" {
		what = "a gift: " + o.Gift
	}
	g.Msg(fmt.Sprintf("{{G|You send %s to %s.}} {{K|(it should arrive in about %d days)}}", what, c.Name, o.Due-day))
	lnDeed(g, "reply", fmt.Sprintf("The wanderer %s wrote a first letter to %s.", g.P.Name, c.Name))
	if coin > 0 {
		lnDeed(g, "sendcoin", fmt.Sprintf("The wanderer %s sent coin by post to %s.", g.P.Name, c.Name))
	}
	if gift != nil {
		lnDeed(g, "sendgift", fmt.Sprintf("The wanderer %s sent a gift by post to %s.", g.P.Name, c.Name))
	}
	s.view, s.sel = "list", 0
	s.tab = 1
}

func (g *Game) lnDraw() {
	s := lnSt(g)
	sc := g.sc
	x, y, w, h := g.panel("Mail")
	switch s.view {
	case "read":
		l := s.byID(s.cur)
		if l == nil {
			s.view = "list"
			return
		}
		head := fmt.Sprintf("{{Y|From %s}}", l.Name)
		if l.Role != "" {
			head += " {{K|(" + l.Role + ")}}"
		}
		if l.Place != "" {
			head += " {{K|of " + l.Place + "}}"
		}
		lines := []string{head, fmt.Sprintf("{{K|Written %s, delivered %s}}", lnDate(g, l.Sent), lnDate(g, l.Got)), ""}
		for _, ln := range strings.Split(l.Text, "\n") {
			if ln == "" {
				lines = append(lines, "")
				continue
			}
			lines = append(lines, Wrap(ln, w)...)
		}
		if l.Coin > 0 {
			lines = append(lines, "", fmt.Sprintf("{{G|Enclosed: %s}}", g.cuFormat(l.Coin)))
		}
		if l.Ask > 0 {
			lines = append(lines, "", fmt.Sprintf("{{y|They ask for %s.}}", g.cuFormat(l.Ask)))
		}
		if l.Kind == "invite" {
			lines = append(lines, "", fmt.Sprintf("{{c|Invitation: %s in %s, %s.}}", nonEmpty(l.IKind, "gathering"), lnBurgName(g, l.Burg), lnDate(g, l.Day)))
		}
		if l.Replied {
			lines = append(lines, "", "{{K|You have answered this.}}")
		}
		view := h - 2
		s.scroll = clampi(s.scroll, 0, maxi(0, len(lines)-view))
		for i := 0; i < view && s.scroll+i < len(lines); i++ {
			sc.TextB(x, y+i, lines[s.scroll+i], CGray, CBgDark, x+w)
		}
		sc.Text(x, y+h-1, "{{K|r: reply   d: delete   Up/Down: scroll   Esc: back}}", CGray)
	case "menu", "people":
		title := s.menuT
		n := len(s.menu)
		if s.view == "people" {
			title, n = "Write to whom?", len(s.pick)
		}
		sc.Text(x, y, "{{Y|"+title+"}}", CGray)
		s.sel = clampi(s.sel, 0, maxi(0, n-1))
		view := h - 4
		off := maxi(0, s.sel-view+1)
		for i := 0; i < view && off+i < n; i++ {
			idx := off + i
			bg := CBgDark
			if idx == s.sel {
				bg = CBgSel
			}
			sc.Fill(x, y+2+i, w, 1, bg)
			if s.view == "people" {
				c := s.pick[idx]
				far := ""
				if c.Planet != lnPlanet(g) {
					far = " (another world)"
				}
				sc.TextB(x+1, y+2+i, fmt.Sprintf("%s {{K|%s, %s%s}}", c.Name, nonEmpty(c.Role, "someone"), nonEmpty(c.Place, "?"), far), CWhite, bg, x+w)
			} else {
				sc.TextB(x+1, y+2+i, s.menu[idx].Text, CWhite, bg, x+w)
			}
		}
		if s.view == "people" && n == 0 {
			sc.Text(x, y+2, "{{K|You have no one to write to yet. Talk with people to meet them.}}", CGray)
		}
		sc.Text(x, y+h-1, "{{K|Up/Down: choose   Enter: select   Esc: back}}", CGray)
	default:
		tabs := []string{fmt.Sprintf("Inbox (%d)", len(s.Box)), fmt.Sprintf("Sent and on the road (%d)", len(s.Out)+len(s.Post))}
		tx := x
		for i, t := range tabs {
			c := "K"
			if i == s.tab {
				c = "Y"
			}
			tx = sc.TextB(tx, y, fmt.Sprintf("{{%s|[ %s ]}} ", c, t), CGray, CBgDark, x+w)
		}
		view := h - 4
		if s.tab == 0 {
			in := s.inbox()
			if len(in) == 0 {
				sc.Text(x, y+2, "{{K|No letters. People you know write when they have news; ask at the post.}}", CGray)
			}
			s.sel = clampi(s.sel, 0, maxi(0, len(in)-1))
			off := maxi(0, s.sel-view+1)
			for i := 0; i < view && off+i < len(in); i++ {
				l := in[off+i]
				bg := CBgDark
				if off+i == s.sel {
					bg = CBgSel
				}
				yy := y + 2 + i
				sc.Fill(x, yy, w, 1, bg)
				mark := " "
				if !l.Read {
					mark = "{{Y|*}}"
				}
				first := strings.SplitN(l.Text, "\n\n", 3)
				snip := ""
				if len(first) > 1 {
					snip = strings.ReplaceAll(first[1], "\n", " ")
				}
				sc.TextB(x, yy, mark, CWhite, bg, x+w)
				sc.TextB(x+2, yy, lvTrunc(lnDate(g, l.Got), 12), CGray, bg, x+w)
				sc.TextB(x+15, yy, lvTrunc(l.Name, 20), CWhite, bg, x+w)
				sc.TextB(x+36, yy, lvTrunc(l.Kind, 7), CCyan, bg, x+w)
				sc.TextB(x+44, yy, lvTrunc(snip, w-44), CGray, bg, x+w)
			}
		} else {
			i := 0
			row := func(t string) {
				if i < view {
					sc.TextB(x, y+2+i, t, CGray, CBgDark, x+w)
				}
				i++
			}
			d := lvDay(g)
			for j := len(s.Out) - 1; j >= 0; j-- {
				o := s.Out[j]
				st := fmt.Sprintf("{{K|arrives in %d days}}", maxi(0, o.Due-d))
				if o.Done {
					st = "{{G|delivered}}"
				}
				what := lnReplyText[o.Kind]
				if o.Gift != "" {
					what = o.Gift
				}
				row(fmt.Sprintf("To %s: %s %s", lvTrunc(o.Name, 20), what, st))
			}
			if len(s.Post) > 0 {
				row("")
				row(fmt.Sprintf("{{K|%d letter%s for you still on the road.}}", len(s.Post), map[bool]string{true: "s", false: ""}[len(s.Post) > 1]))
			}
			if i == 0 {
				row("{{K|Nothing sent yet. Press w to write to someone you know.}}")
			}
		}
		sc.Text(x, y+h-1, "{{K|Enter: read   r: reply   w: write   d: delete   Tab: inbox/sent   Esc: close}}", CGray)
	}
}
