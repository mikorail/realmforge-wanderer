package engine

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// Sieges and war events (prefix sq): a live siege of a real town while two realms are at war in the world data.
//
// Built on what exists and not duplicated: war.go (warsOf, the soldier() builder, merit and ranks, enlistment, frontier
// skirmishes), es_war.go (static battlefield and siege scenes from the chronicle), urban_evolve.go (wartime gate
// portcullises and ballistas; here the gates are closed and guarded only while a siege runs and are opened again
// afterwards), worldsim.go (armies on the road), rumourmill.go (reads lvSt(g).News, which every phase change writes to,
// so a siege becomes a rumour with no further wiring), realmstory.go and endgame (crises; read-only), contractboard.go
// (patrol and guard kinds are separate), caravans.go, crime.go (uwContraband, read only).
//
// A siege is a planet-local record per (attacking realm, besieged burg). Plans are pure functions of (seed, 70-day epoch,
// war pair) via HashStr: a war pair gets a siege in an epoch about half the time, starting on day 0..29 of the epoch and
// lasting 10..40 days; at most two are live per planet. Phases: muster, encirclement, assault, breach, then sack (3 days)
// or an early end by relief, truce, camp plague, or the siege being lifted. Every day the record advances from a side
// rng (seed, siege id, day); the player moves the counters (progress, morale, food, truce, relief, camp sickness) through
// five roles, each a short sequence of lettered-choice steps: defender, attacker, smuggler, envoy, relief rider (and a
// sortie). The player is never killed by a step (hp losses leave one point), and the sack always leaves a way out.
// Nothing here draws g.R except the check rolls in the steps. State g.ext("sq"), planet-local.
//
// Public hooks for other files: sqPriceMod, sqGreetLine, sqOccupier, sqSacked, sqRumours, sqRumourSink, sqFallSink.

func init() {
	unLocalKeys = append(unLocalKeys, "sq")
	turnHooks = append(turnHooks, sqTurn)
	enterHooks = append(enterHooks, sqEnter)
	talkHooks = append(talkHooks, sqTalk)
	sideHooks = append(sideHooks, sqSide)
	modeKeys["sq"] = sqKeys
	modeDraws["sq"] = func(g *Game) { g.sqDraw() }
	paEntries = append(paEntries, paEntry{Name: "Siege front", Keywords: "siege war camp besieged garrison defend attack smuggle envoy truce relief sortie", Cat: "Screen", Label: "screen",
		Cond: func(g *Game) bool { return g.W != nil && g.sqFocus() != nil },
		Func: func(g *Game) {
			if s := g.sqFocus(); s != nil {
				g.sqOpen(s, "")
			}
		}})
}

// sqRumourSink, when set, hears every phase change: kind (siege relief fall truce plague lifted), subject "burg:<id>", text.
var sqRumourSink func(g *Game, kind, subject, text string)

// sqFallSink, when set, is called once when a town falls to the attacking realm (burg id, realm id).
var sqFallSink func(g *Game, burg int, to int)

const (
	sqEpochLen = 70
	sqMaxAct   = 2
)

var sqPhases = []string{"muster", "encircle", "assault", "breach", "sack"}

func sqPhaseIdx(p string) int {
	for i, n := range sqPhases {
		if n == p {
			return i
		}
	}
	return 0
}

type sqSiege struct {
	ID     int    `json:"i"`
	Att    int    `json:"a"`
	Def    int    `json:"d"`
	Burg   int    `json:"b"`
	Start  int    `json:"s"`
	Dur    int    `json:"u"`
	Phase  string `json:"p"`
	Prog   int    `json:"pg"`
	Mor    int    `json:"mo"`
	Food   int    `json:"fo"`
	Truce  int    `json:"tr"`
	Relief int    `json:"re"`
	Sick   int    `json:"si"`
	SickOn bool   `json:"so,omitempty"`
	Bias   int    `json:"bi"`
	Last   int    `json:"l"`
	Sack   int    `json:"sk,omitempty"`
	Ann    string `json:"an,omitempty"`
	CapN   string `json:"cn"`
	QmN    string `json:"qn"`
	OffN   string `json:"on"`
}

type sqHist struct {
	ID   int    `json:"i"`
	Burg int    `json:"b"`
	Att  int    `json:"a"`
	Def  int    `json:"d"`
	Out  string `json:"o"`
	Day  int    `json:"y"`
}

type sqRumour struct {
	Kind    string `json:"k"`
	Subject string `json:"s"`
	Text    string `json:"t"`
	Burg    int    `json:"b"`
	Day     int    `json:"d"`
}

type sqRide struct {
	SID  int `json:"s"`
	Dest int `json:"d"`
	Due  int `json:"u"`
}

type sqCur struct {
	SID   int
	Step  string
	Clock int
	Timed bool
	Res   bool
	Lines []string
	Next  string
}

type sqState struct {
	Next    int              `json:"n"`
	Last    int              `json:"l"`
	Act     []*sqSiege       `json:"a,omitempty"`
	Hist    []*sqHist        `json:"h,omitempty"`
	Plans   map[string]bool  `json:"pl"`
	Fallen  map[int]int      `json:"fa"`
	Sacked  map[int]int      `json:"sk"`
	Applied map[string]int   `json:"ap"`
	Closed  map[string][]int `json:"cl"`
	Eng     map[string][]int `json:"en"`
	Pass    map[string]int   `json:"ps"`
	Side    map[int]string   `json:"sd"`
	Cool    map[string]int   `json:"co"`
	Hint    map[string]int   `json:"hi"`
	Merit   int              `json:"m,omitempty"`
	Served  int              `json:"sv,omitempty"`
	Ride    *sqRide          `json:"r,omitempty"`
	Rums    []sqRumour       `json:"ru,omitempty"`

	cur *sqCur
}

func (g *Game) sq() *sqState {
	mk := func() any {
		return &sqState{Plans: map[string]bool{}, Fallen: map[int]int{}, Sacked: map[int]int{}, Applied: map[string]int{}, Closed: map[string][]int{},
			Eng: map[string][]int{}, Pass: map[string]int{}, Side: map[int]string{}, Cool: map[string]int{}, Hint: map[string]int{}}
	}
	s := g.ext("sq", mk).(*sqState)
	if s.Plans == nil {
		s.Plans = map[string]bool{}
	}
	if s.Fallen == nil {
		s.Fallen = map[int]int{}
	}
	if s.Sacked == nil {
		s.Sacked = map[int]int{}
	}
	if s.Applied == nil {
		s.Applied = map[string]int{}
	}
	if s.Closed == nil {
		s.Closed = map[string][]int{}
	}
	if s.Eng == nil {
		s.Eng = map[string][]int{}
	}
	if s.Pass == nil {
		s.Pass = map[string]int{}
	}
	if s.Side == nil {
		s.Side = map[int]string{}
	}
	if s.Cool == nil {
		s.Cool = map[string]int{}
	}
	if s.Hint == nil {
		s.Hint = map[string]int{}
	}
	return s
}

// ---------------------------------------------------------------- public reads

// sqOccupier is the realm that holds the burg after a siege, or 0.
func sqOccupier(g *Game, burg int) int {
	if g == nil || g.W == nil {
		return 0
	}
	return g.sq().Fallen[burg]
}

// sqSacked tells whether the burg was sacked (and on what day).
func sqSacked(g *Game, burg int) (int, bool) {
	if g == nil || g.W == nil {
		return 0, false
	}
	d, ok := g.sq().Sacked[burg]
	return d, ok
}

// sqRumours lists recent siege rumours, oldest first.
func sqRumours(g *Game) []sqRumour {
	if g == nil || g.W == nil {
		return nil
	}
	return append([]sqRumour(nil), g.sq().Rums...)
}

// sqBesieged: the live siege of a burg, or nil.
func (g *Game) sqBesieged(burg int) *sqSiege {
	for _, s := range g.sq().Act {
		if s.Burg == burg {
			return s
		}
	}
	return nil
}

// sqPriceMod: rationing in a besieged town (food and water dear, you are paid more for it, other goods cheaper to buy
// from you), and a small occupation levy in a fallen one. buy = the player buys.
func sqPriceMod(g *Game, a *Actor, it *Item, buy bool) float64 {
	if g == nil || g.W == nil || g.Cur == nil || g.Cur.Kind != "town" || it == nil {
		return 1
	}
	food := it.Kind == "food" || it.Kind == "water" || it.Kind == "dish" || it.Kind == "ingredient"
	if s := g.sqBesieged(g.Cur.Burg); s != nil {
		if sqPhaseIdx(s.Phase) < 1 {
			return 1
		}
		hunger := float64(100-s.Food) / 100
		if food {
			if buy {
				return 1.15 + hunger*1.6
			}
			return 1 + hunger*0.5
		}
		if buy {
			return 1
		}
		return 0.88
	}
	if _, ok := g.sq().Fallen[g.Cur.Burg]; ok {
		if buy {
			return 1.1
		}
		return 0.92
	}
	return 1
}

// sqGreetLine: a line for a talking actor in a besieged or occupied town, or "".
func sqGreetLine(g *Game, a *Actor) string {
	if g == nil || g.W == nil || g.Cur == nil || a == nil || g.Cur.Kind != "town" {
		return ""
	}
	h := int(HashStr(fmt.Sprint(g.Seed, ":sq:greet:", a.ID, ":", g.Day())) % 1000)
	if s := g.sqBesieged(g.Cur.Burg); s != nil {
		var pool []string
		if a.Role == "guard" || a.Has("sqgate") || a.Has("sqcapt") {
			pool = sqGuardLines[s.Phase]
		} else {
			pool = sqFolkLines[s.Phase]
		}
		if len(pool) == 0 {
			return ""
		}
		return g.sqFill(s, pool[h%len(pool)])
	}
	if att, ok := g.sq().Fallen[g.Cur.Burg]; ok {
		return strings.NewReplacer("{A}", g.realmName(att)).Replace(sqOccLines[h%len(sqOccLines)])
	}
	return ""
}

// ---------------------------------------------------------------- helpers

func (g *Game) sqTown(s *sqSiege) string {
	if b := g.W.burgByID[s.Burg]; b != nil {
		return b.Name
	}
	return "the town"
}

func (g *Game) sqFill(s *sqSiege, t string) string {
	way := "old tunnel"
	if b := g.W.burgByID[s.Burg]; b != nil && b.Port {
		way = "night boat"
	}
	dest, days := "the next town", "a few"
	if r := g.sq().Ride; r != nil {
		if b := g.W.burgByID[r.Dest]; b != nil {
			dest = b.Name
		}
		days = strconv.Itoa(maxi(0, r.Due-g.Day()))
	} else if d := g.sqReliefDest(s); d != nil {
		dest = d.Name
	}
	return strings.NewReplacer("{T}", g.sqTown(s), "{A}", g.realmName(s.Att), "{D}", g.realmName(s.Def), "{Cap}", s.CapN, "{Qm}", s.QmN,
		"{Off}", s.OffN, "{Way}", way, "{Dest}", dest, "{Days}", days).Replace(t)
}

func (g *Game) sqBurgXY(burg int) (int, int, bool) {
	b := g.W.burgByID[burg]
	if b == nil || g.W.W == 0 {
		return 0, 0, false
	}
	return b.P % g.W.W, b.P / g.W.W, true
}

// sqDist: parasangs (Chebyshev) between the player and the burg.
func (g *Game) sqDist(s *sqSiege) int {
	x, y, ok := g.sqBurgXY(s.Burg)
	if !ok {
		return 99
	}
	return maxi(abs(g.WX-x), abs(g.WY-y))
}

func (g *Game) sqNear() *sqSiege {
	var best *sqSiege
	bd := 3
	for _, s := range g.sq().Act {
		if d := g.sqDist(s); d < bd {
			best, bd = s, d
		}
	}
	return best
}

func (g *Game) sqByID(id int) *sqSiege {
	for _, s := range g.sq().Act {
		if s.ID == id {
			return s
		}
	}
	return nil
}

// sqFocus: the siege the screen shows: the open one, else the nearest, else the one a relief ride is for.
func (g *Game) sqFocus() *sqSiege {
	st := g.sq()
	if st.cur != nil {
		if s := g.sqByID(st.cur.SID); s != nil {
			return s
		}
	}
	if s := g.sqNear(); s != nil {
		return s
	}
	if st.Ride != nil {
		return g.sqByID(st.Ride.SID)
	}
	return nil
}

func (g *Game) sqAtWar(a, d int) bool {
	for _, e := range g.warsOf(a) {
		if e == d {
			return true
		}
	}
	return false
}

func sqRank(merit int) int {
	r := 0
	for i, m := range warMerit {
		if merit >= m {
			r = i
		}
	}
	return r
}

func (g *Game) sqUnit() int { return 5 + 3*g.P.Level }

func (g *Game) sqFoodCount() int {
	n := 0
	for _, it := range g.P.Inv {
		if it.Kind == "food" {
			n += maxi(1, it.Count)
		}
	}
	return n
}

func (g *Game) sqTakeFood(n int) int {
	taken := 0
	for taken < n {
		found := false
		for i, it := range g.P.Inv {
			if it.Kind != "food" {
				continue
			}
			found = true
			if it.Count > 1 {
				it.Count--
			} else {
				g.P.Inv = append(g.P.Inv[:i], g.P.Inv[i+1:]...)
			}
			taken++
			break
		}
		if !found {
			break
		}
	}
	return taken
}

func sqBar(v int) string {
	v = clampi(v, 0, 100)
	n := (v + 5) / 10
	return "[" + strings.Repeat("#", n) + strings.Repeat("-", 10-n) + "]"
}

// ---------------------------------------------------------------- planning

func (g *Game) sqPickBurg(att, def int, key string) int {
	w := g.W
	var atts []int
	for i := range w.Burgs {
		b := &w.Burgs[i]
		if b.Alive && b.Layer == "surface" && b.State == att {
			atts = append(atts, b.P)
		}
	}
	if len(atts) == 0 || w.W == 0 {
		return 0
	}
	type cand struct{ id, d int }
	var cs []cand
	taken := map[int]bool{}
	for _, s := range g.sq().Act {
		taken[s.Burg] = true
	}
	for i := range w.Burgs {
		b := &w.Burgs[i]
		if !b.Alive || b.Layer != "surface" || b.State != def || taken[b.ID] || b.Capital {
			continue
		}
		if _, f := g.sq().Fallen[b.ID]; f {
			continue
		}
		best := 1 << 30
		for _, p := range atts {
			if d := abs(p%w.W-b.P%w.W) + abs(p/w.W-b.P/w.W); d < best {
				best = d
			}
		}
		cs = append(cs, cand{b.ID, best})
	}
	if len(cs) == 0 {
		return 0
	}
	sort.Slice(cs, func(i, j int) bool {
		if cs[i].d != cs[j].d {
			return cs[i].d < cs[j].d
		}
		return cs[i].id < cs[j].id
	})
	n := mini(3, len(cs))
	return cs[int(HashStr(fmt.Sprint(g.Seed, ":sq:pick:", key))%uint64(n))].id
}

// sqPlan creates the sieges whose start day has come (deterministic from seed, epoch and war pair).
func (g *Game) sqPlan(day int) {
	st := g.sq()
	ep := day / sqEpochLen
	type pr struct{ a, b int }
	var prs []pr
	for i := range g.W.States {
		s := &g.W.States[i]
		if !s.Alive {
			continue
		}
		for _, e := range g.warsOf(s.ID) {
			if e > s.ID {
				prs = append(prs, pr{s.ID, e})
			}
		}
	}
	sort.Slice(prs, func(i, j int) bool {
		if prs[i].a != prs[j].a {
			return prs[i].a < prs[j].a
		}
		return prs[i].b < prs[j].b
	})
	for _, p := range prs {
		key := fmt.Sprintf("%d:%d:%d", ep, p.a, p.b)
		if st.Plans[key] {
			continue
		}
		h := HashStr(fmt.Sprintf("%s:%d:sq:plan:%s", g.Seed, g.W.uniIdx, key))
		start := ep*sqEpochLen + int(h%30)
		if day < start {
			continue
		}
		st.Plans[key] = true
		if int(h/7%100) >= 55 {
			continue
		}
		dur := 10 + int(h/1000%31)
		if day >= start+dur || len(st.Act) >= sqMaxAct {
			continue
		}
		att, def := p.a, p.b
		if h/13%2 == 1 {
			att, def = def, att
		}
		burg := g.sqPickBurg(att, def, key)
		if burg == 0 {
			continue
		}
		st.Next++
		s := &sqSiege{ID: st.Next, Att: att, Def: def, Burg: burg, Start: start, Dur: dur, Phase: "muster", Mor: 70, Food: 80, Last: start - 1,
			Bias: int(h/100%31) - 15}
		g.sqNames(s)
		st.Act = append(st.Act, s)
		for d := start; d <= day && g.sqByID(s.ID) != nil; d++ {
			g.sqAdvance(s, d, day)
		}
		if g.sqByID(s.ID) != nil {
			s.Last = day
		}
	}
}

func (g *Game) sqNames(s *sqSiege) {
	bd := g.W.burgByID[s.Burg]
	if bd == nil {
		s.CapN, s.QmN, s.OffN = "the captain", "the quartermaster", "the field officer"
		return
	}
	cu := g.W.cultByID[bd.Culture]
	r := NewRng(fmt.Sprint(g.Seed, ":sq:cap:", s.ID))
	s.CapN = g.makeNPC("guard", bd, cu, r).Name
	s.QmN = g.makeNPC("villager", bd, cu, r).Name
	s.OffN = "the field officer"
	if o := g.soldier(s.Att, NewRng(fmt.Sprint(g.Seed, ":sq:off:", s.ID))); o != nil {
		s.OffN = o.Name
	}
}

// ---------------------------------------------------------------- daily simulation

func (g *Game) sqAdvance(s *sqSiege, day, today int) {
	r := NewRng(fmt.Sprint(g.Seed, ":sq:day:", s.ID, ":", day))
	if s.Phase == "sack" {
		if day-s.Sack >= 3 {
			g.sqEnd(s, "fall", day, today)
		}
		return
	}
	sc := func(n int) int {
		f := float64(n) * 30 / float64(s.Dur)
		v := int(f)
		if r.Float() < f-float64(v) {
			v++
		}
		return v
	}
	if !g.sqAtWar(s.Att, s.Def) {
		s.Truce += 35
	}
	t := (day - s.Start) * 100 / s.Dur
	pi := 0
	switch {
	case t < 15:
		pi = 0
	case t < 40:
		pi = 1
	case t < 65:
		pi = 2
	default:
		pi = 2
		if s.Prog >= 50 {
			pi = 3
		}
	}
	if pi == 2 && s.Prog >= 70 {
		pi = 3
	}
	if cur := sqPhaseIdx(s.Phase); pi < cur {
		pi = cur
	}
	s.Phase = sqPhases[pi]
	if s.Ann != s.Phase {
		g.sqAnnounce(s, s.Phase, day, today)
	}
	slow := 0
	if s.Sick > 40 {
		slow = 1
	}
	switch pi {
	case 0:
		s.Prog += r.Int(0, 1)
	case 1:
		s.Prog += maxi(0, sc(r.Int(1, 2))-slow)
		s.Food -= sc(2) + r.Int(0, 1)
		s.Mor -= sc(1)
	case 2:
		s.Prog += maxi(0, sc(r.Int(2, 3))-slow)
		s.Food -= sc(2) + r.Int(0, 1)
		s.Mor -= sc(2)
	case 3:
		s.Prog += maxi(0, sc(r.Int(3, 5))-slow)
		s.Food -= sc(2) + r.Int(0, 1)
		s.Mor -= sc(3)
	}
	if pi >= 1 {
		if r.Int(1, 100) <= 40 {
			s.Relief += r.Int(2, 5)
		}
		s.Truce += r.Int(0, 2)
	}
	if !s.SickOn && t >= 33 && r.Int(1, 100) <= 7 {
		s.SickOn = true
	}
	if s.SickOn {
		s.Sick += r.Int(1, 4)
	}
	s.Prog, s.Mor, s.Food = clampi(s.Prog, 0, 100), clampi(s.Mor, 0, 100), clampi(s.Food, 0, 100)
	s.Truce, s.Relief, s.Sick = clampi(s.Truce, 0, 100), clampi(s.Relief, 0, 100), clampi(s.Sick, 0, 100)
	switch {
	case s.Truce >= 100:
		g.sqEnd(s, "truce", day, today)
	case s.Relief >= 100:
		g.sqEnd(s, "relief", day, today)
	case s.Sick >= 100:
		g.sqEnd(s, "plague", day, today)
	case pi == 3 && s.Prog >= 100, pi >= 2 && s.Mor <= 0:
		g.sqFall(s, day, today)
	case day >= s.Start+s.Dur:
		score := s.Prog + (100-s.Food)/3 - s.Mor/3 + s.Bias
		if score >= 66 {
			g.sqFall(s, day, today)
		} else {
			g.sqEnd(s, "lifted", day, today)
		}
	}
}

func (g *Game) sqFall(s *sqSiege, day, today int) {
	st := g.sq()
	s.Phase, s.Sack = "sack", day
	st.Fallen[s.Burg] = s.Att
	st.Sacked[s.Burg] = day
	if st.Ride != nil && st.Ride.SID == s.ID {
		st.Ride = nil
	}
	g.sqAnnounce(s, "fall", day, today)
	s.Ann = "sack"
	if sqFallSink != nil {
		sqFallSink(g, s.Burg, s.Att)
	}
}

func (g *Game) sqEnd(s *sqSiege, out string, day, today int) {
	st := g.sq()
	if out != "fall" {
		g.sqAnnounce(s, out, day, today)
	}
	st.Hist = append(st.Hist, &sqHist{ID: s.ID, Burg: s.Burg, Att: s.Att, Def: s.Def, Out: out, Day: day})
	if len(st.Hist) > 12 {
		st.Hist = st.Hist[len(st.Hist)-12:]
	}
	keep := st.Act[:0]
	for _, x := range st.Act {
		if x.ID != s.ID {
			keep = append(keep, x)
		}
	}
	st.Act = keep
	if st.Ride != nil && st.Ride.SID == s.ID {
		st.Ride = nil
	}
}

func (g *Game) sqAnnounce(s *sqSiege, kind string, day, today int) {
	s.Ann = kind
	tpl, ok := sqNewsTpl[kind]
	if !ok {
		return
	}
	text := g.sqFill(s, tpl)
	st := g.sq()
	lvSt(g).addNews(day, s.Burg, "%s", text)
	rk := kind
	switch kind {
	case "muster", "encircle", "assault", "breach":
		rk = "siege"
	}
	st.Rums = append(st.Rums, sqRumour{Kind: rk, Subject: "burg:" + strconv.Itoa(s.Burg), Text: text, Burg: s.Burg, Day: day})
	if len(st.Rums) > 30 {
		st.Rums = st.Rums[len(st.Rums)-30:]
	}
	if sqRumourSink != nil {
		sqRumourSink(g, rk, "burg:"+strconv.Itoa(s.Burg), text)
	}
	if day == today && g.P != nil && g.sqDist(s) <= 3 {
		g.Msg("{{R|" + text + "}}")
	}
}

// ---------------------------------------------------------------- hooks: time

func sqTurn(g *Game) {
	if g == nil || g.W == nil || g.P == nil || g.Cur == nil {
		return
	}
	st := g.sq()
	day := g.Day()
	if day != st.Last {
		from := st.Last + 1
		if st.Last == 0 || day-st.Last > 120 {
			from = day
		}
		for d := from; d <= day; d++ {
			g.sqPlan(d)
			for _, s := range append([]*sqSiege(nil), st.Act...) {
				if s.Last < d && g.sqByID(s.ID) != nil {
					g.sqAdvance(s, d, day)
					s.Last = d
				}
			}
		}
		st.Last = day
		if r := st.Ride; r != nil && day > r.Due {
			g.Msg("{{R|You are too late: the relief you promised cannot be raised in time.}}")
			if s := g.sqByID(r.SID); s != nil {
				g.AddRep(stateFaction(s.Def), -10, " for a failed ride")
				s.Mor = clampi(s.Mor-4, 0, 100)
			}
			st.Ride = nil
		}
		g.sqSync(g.Cur)
	}
	if g.Turn%10 == 0 {
		g.sqRecloseGates()
	}
}

func sqEnter(g *Game, zn *Zone) {
	if g == nil || g.W == nil || zn == nil {
		return
	}
	st := g.sq()
	g.sqSync(zn)
	day := g.Day()
	if r := st.Ride; r != nil && zn.Kind == "town" && zn.Burg == r.Dest {
		if g.sqByID(r.SID) != nil {
			g.Msg("{{Y|You have reached " + g.W.burgByID[r.Dest].Name + ". Present the letter from the Siege front screen.}}")
		}
	}
	if zn.Z != 0 {
		return
	}
	for _, s := range st.Act {
		x, y, ok := g.sqBurgXY(s.Burg)
		if !ok || maxi(abs(zn.WX-x), abs(zn.WY-y)) > 1 || st.Hint[zn.Key] == day {
			continue
		}
		st.Hint[zn.Key] = day
		if zn.Kind == "town" && zn.Burg == s.Burg {
			g.Msg("{{R|" + g.sqFill(s, sqPhaseDesc[s.Phase]) + "}}")
		} else if sqPhaseIdx(s.Phase) >= 2 {
			g.Msg("{{K|A dark smoke stands over " + g.sqTown(s) + " to the " + jbDirName(x-zn.WX, y-zn.WY) + ".}}")
		} else {
			g.Msg("{{K|Campfires of " + g.realmName(s.Att) + " mark the plain near " + g.sqTown(s) + ".}}")
		}
		break
	}
}

func sqSide(g *Game) []string {
	if g == nil || g.W == nil || g.P == nil {
		return nil
	}
	var out []string
	if s := g.sqNear(); s != nil {
		out = append(out, fmt.Sprintf("{{R|Siege of %s}} {{K|%s, day %d/%d}}", g.sqTown(s), s.Phase, maxi(1, g.Day()-s.Start+1), s.Dur))
		out = append(out, fmt.Sprintf("{{K|Stores %d%% Morale %d%%}}", s.Food, s.Mor))
	}
	if r := g.sq().Ride; r != nil {
		if b := g.W.burgByID[r.Dest]; b != nil {
			out = append(out, fmt.Sprintf("{{Y|Ride to %s: %d days}}", b.Name, maxi(0, r.Due-g.Day())))
		}
	}
	return out
}

// ---------------------------------------------------------------- zones

func (g *Game) sqZoneMode(zn *Zone) (string, *sqSiege, int) {
	st := g.sq()
	if zn.Z != 0 {
		return "", nil, 0
	}
	if zn.Kind == "town" && zn.Burg > 0 {
		for _, s := range st.Act {
			if s.Burg == zn.Burg {
				return "town", s, s.ID*10 + sqPhaseIdx(s.Phase) + 1
			}
		}
		if att, ok := st.Fallen[zn.Burg]; ok {
			return "occ", nil, 900000 + att
		}
		return "", nil, 0
	}
	if zn.Kind == "surface" {
		for _, s := range st.Act {
			x, y, ok := g.sqBurgXY(s.Burg)
			if ok && maxi(abs(zn.WX-x), abs(zn.WY-y)) <= 1 {
				return "camp", s, s.ID*10 + sqPhaseIdx(s.Phase) + 1
			}
		}
	}
	return "", nil, 0
}

func (g *Game) sqSync(zn *Zone) {
	if zn == nil || g.W == nil {
		return
	}
	st := g.sq()
	mode, s, stamp := g.sqZoneMode(zn)
	if st.Applied[zn.Key] == stamp {
		return
	}
	g.sqClear(zn)
	delete(st.Applied, zn.Key)
	switch mode {
	case "town":
		g.sqPlaceTown(zn, s)
	case "camp":
		g.sqPlaceCamp(zn, s)
	case "occ":
		g.sqPlaceOcc(zn, st.Fallen[zn.Burg])
	}
	if stamp != 0 {
		st.Applied[zn.Key] = stamp
	}
}

func (g *Game) sqClear(zn *Zone) {
	st := g.sq()
	ki := zn.Items[:0]
	for _, it := range zn.Items {
		if it.Extra != nil && it.Extra["sqx"] == 1 {
			continue
		}
		ki = append(ki, it)
	}
	zn.Items = ki
	ka := zn.Actors[:0]
	for _, a := range zn.Actors {
		if a.Has("sqx") {
			continue
		}
		ka = append(ka, a)
	}
	zn.Actors = ka
	if e := st.Eng[zn.Key]; len(e) >= 3 {
		for i := 0; i+2 < len(e); i += 3 {
			if zn.At(e[i], e[i+1]) == TBallista {
				zn.Set(e[i], e[i+1], Terrain(e[i+2]))
			}
		}
	}
	delete(st.Eng, zn.Key)
	for _, idx := range st.Closed[zn.Key] {
		if idx >= 0 && idx < len(zn.T) && zn.T[idx] == TGateClosed {
			zn.T[idx] = TDoorOpen
		}
	}
	delete(st.Closed, zn.Key)
}

func (g *Game) sqItem(name, lore string, gl rune, f byte, sid int) *Item {
	return &Item{ID: g.newID(), Kind: "scrap", Name: name, Glyph: gl, F: f, Value: 1, Weight: 900, Lore: lore, Ident: true,
		Extra: map[string]int{"sqx": 1, "sqid": sid}}
}

func (g *Game) sqMark(a *Actor, s *sqSiege, flags ...string) {
	a.Set("sqx")
	if s != nil {
		a.Set("sq" + strconv.Itoa(s.ID))
	}
	for _, f := range flags {
		a.Set(f)
	}
	a.HomeX, a.HomeY = a.X, a.Y
}

func (g *Game) sqPlaceCamp(zn *Zone, s *sqSiege) {
	st := g.sq()
	bucket := sqPhaseIdx(s.Phase) + 1
	r := NewRng(fmt.Sprint(g.Seed, ":sq:camp:", zn.Key, ":", s.ID*10+bucket))
	cx, cy, ok := zn.FreeSpot(r, func(x, y int) bool { return x > 12 && x < MW-12 && y > 5 && y < MH-5 })
	if !ok {
		return
	}
	near := func(rad int) func(x, y int) bool {
		return func(x, y int) bool { return abs(x-cx) <= rad && abs(y-cy) <= rad }
	}
	lore := "Part of the besiegers' camp of " + g.realmName(s.Att) + "."
	for i := 0; i < 3+2*bucket; i++ {
		if x, y, ok := zn.FreeSpot(r, near(7)); ok {
			it := g.sqItem("siege tent", lore, '^', CBrown, s.ID)
			it.X, it.Y = x, y
			zn.Items = append(zn.Items, it)
		}
	}
	for i := 0; i < 2+bucket/2; i++ {
		if x, y, ok := zn.FreeSpot(r, near(6)); ok {
			it := g.sqItem("camp fire", lore, '*', COrange, s.ID)
			it.X, it.Y = x, y
			zn.Items = append(zn.Items, it)
		}
	}
	if bucket >= 3 {
		for i := 0; i < 2; i++ {
			if x, y, ok := zn.FreeSpot(r, near(8)); ok && zn.At(x, y).D().Water == 0 {
				st.Eng[zn.Key] = append(st.Eng[zn.Key], x, y, int(zn.At(x, y)))
				zn.Set(x, y, TBallista)
			}
		}
		if x, y, ok := zn.FreeSpot(r, near(8)); ok {
			it := g.sqItem("battering ram", lore, '=', CBrown, s.ID)
			it.X, it.Y = x, y
			zn.Items = append(zn.Items, it)
		}
	}
	for i := 0; i < mini(12, 2+2*bucket); i++ {
		x, y, ok := zn.FreeSpot(r, near(8))
		if !ok {
			break
		}
		a := g.soldier(s.Att, r)
		if a == nil {
			return
		}
		a.X, a.Y = x, y
		a.AI = "still"
		a.Title = "besieger"
		a.Desc = fmt.Sprintf("A soldier of %s, camped outside %s.", g.realmName(s.Att), g.sqTown(s))
		g.sqMark(a, s, "sqarmy")
		zn.Actors = append(zn.Actors, a)
	}
	if x, y, ok := zn.FreeSpot(r, near(4)); ok {
		if a := g.soldier(s.Att, NewRng(fmt.Sprint(g.Seed, ":sq:off:", s.ID))); a != nil {
			a.X, a.Y = x, y
			a.AI = "still"
			a.Name, a.Proper, a.Title = s.OffN, true, "field officer"
			a.Desc = fmt.Sprintf("%s commands the siege of %s for %s.", s.OffN, g.sqTown(s), g.realmName(s.Att))
			g.sqMark(a, s, "sqarmy", "sqoff")
			zn.Actors = append(zn.Actors, a)
		}
	}
}

var sqGates = [][2]int{{40, 0}, {40, MH - 1}, {0, 12}, {MW - 1, 12}}

func (g *Game) sqPlaceTown(zn *Zone, s *sqSiege) {
	st := g.sq()
	bd := g.W.burgByID[s.Burg]
	if bd == nil {
		return
	}
	cu := g.W.cultByID[bd.Culture]
	pi := sqPhaseIdx(s.Phase)
	bucket := pi + 1
	r := NewRng(fmt.Sprint(g.Seed, ":sq:town:", zn.Key, ":", s.ID*10+bucket))
	lore := "A sign of the siege of " + bd.Name + "."
	put := func(a *Actor, flags ...string) {
		x, y, ok := zn.FreeSpot(r, nil)
		if !ok {
			return
		}
		a.X, a.Y = x, y
		g.sqMark(a, s, flags...)
		zn.Actors = append(zn.Actors, a)
	}
	if pi >= 1 && pi < 4 {
		for _, gt := range sqGates {
			gx, gy := gt[0], gt[1]
			if !zn.In(gx, gy) || zn.At(gx, gy) != TDoorOpen {
				continue
			}
			zn.Set(gx, gy, TGateClosed)
			st.Closed[zn.Key] = append(st.Closed[zn.Key], gy*MW+gx)
			ix, iy := gx, gy
			switch {
			case gy == 0:
				iy = 1
			case gy == MH-1:
				iy = MH - 2
			case gx == 0:
				ix = 1
			default:
				ix = MW - 2
			}
			if fx, fy, ok := g.nearestFree(zn, ix, iy); ok {
				a := g.makeNPC("guard", bd, cu, r)
				a.X, a.Y = fx, fy
				a.AI = "still"
				a.Title = "gate guard"
				a.Desc = fmt.Sprintf("A guard of %s, watching the shut gate.", bd.Name)
				g.sqMark(a, s, "sqgate")
				zn.Actors = append(zn.Actors, a)
			}
		}
	}
	if a := g.makeNPC("guard", bd, cu, NewRng(fmt.Sprint(g.Seed, ":sq:cap:", s.ID))); a != nil {
		a.Name, a.Proper, a.Title, a.AI = s.CapN, true, "captain of the garrison", "still"
		a.Desc = fmt.Sprintf("%s holds %s for %s.", s.CapN, bd.Name, g.realmName(s.Def))
		put(a, "sqcapt")
	}
	if a := g.makeNPC("villager", bd, cu, NewRng(fmt.Sprint(g.Seed, ":sq:qm:", s.ID))); a != nil {
		a.Name, a.Proper, a.Title, a.AI = s.QmN, true, "quartermaster", "still"
		a.Desc = fmt.Sprintf("%s counts the grain of %s, and does not like the number.", s.QmN, bd.Name)
		put(a, "sqqm")
	}
	if pi < 4 {
		for i := 0; i < mini(6, 2+bucket); i++ {
			a := g.makeNPC("villager", bd, cu, r)
			a.Title = "refugee"
			a.Desc = fmt.Sprintf("A refugee from the villages around %s.", bd.Name)
			put(a, "sqref")
		}
		if pi >= 2 {
			for i := 0; i < mini(4, bucket); i++ {
				a := g.makeNPC("villager", bd, cu, r)
				a.Title, a.AI, a.F = "wounded soldier", "still", CRed
				a.HP = maxi(1, a.MaxHP/3)
				a.Desc = "A soldier with a bandaged wound, resting where there was room."
				put(a, "sqwound")
			}
			for i := 0; i < 2+bucket; i++ {
				if x, y, ok := zn.FreeSpot(r, func(x, y int) bool { return x < 8 || x > MW-9 || y < 4 || y > MH-5 }); ok {
					it := g.sqItem("burning rubble", lore, '*', COrange, s.ID)
					it.X, it.Y = x, y
					zn.Items = append(zn.Items, it)
				}
			}
		}
	} else {
		for i := 0; i < 5; i++ {
			a := g.soldier(s.Att, r)
			if a == nil {
				break
			}
			a.AI = "still"
			a.Title = "looter"
			a.Desc = fmt.Sprintf("A soldier of %s, loose in %s.", g.realmName(s.Att), bd.Name)
			put(a, "sqarmy", "sqocc")
		}
		for i := 0; i < 6; i++ {
			if x, y, ok := zn.FreeSpot(r, nil); ok {
				it := g.sqItem("fire in the street", lore, '*', CRed, s.ID)
				it.X, it.Y = x, y
				zn.Items = append(zn.Items, it)
			}
		}
	}
}

func (g *Game) sqPlaceOcc(zn *Zone, att int) {
	bd := g.W.burgByID[zn.Burg]
	if bd == nil {
		return
	}
	r := NewRng(fmt.Sprint(g.Seed, ":sq:occ:", zn.Key, ":", att))
	for i := 0; i < 5; i++ {
		x, y, ok := zn.FreeSpot(r, nil)
		if !ok {
			break
		}
		a := g.soldier(att, r)
		if a == nil {
			return
		}
		a.X, a.Y = x, y
		a.AI = "still"
		a.Title = "occupier"
		a.Desc = fmt.Sprintf("A soldier of %s, holding %s.", g.realmName(att), bd.Name)
		g.sqMark(a, nil, "sqarmy", "sqocc")
		zn.Actors = append(zn.Actors, a)
	}
}

// sqRecloseGates: guards drop the portcullis again once you have passed through it, unless they let you in.
func (g *Game) sqRecloseGates() {
	zn := g.Cur
	st := g.sq()
	idxs := st.Closed[zn.Key]
	if len(idxs) == 0 || st.Pass[zn.Key] == g.Day() {
		return
	}
	guard := false
	for _, a := range zn.Actors {
		if a.Has("sqgate") && a.HP > 0 {
			guard = true
		}
	}
	if !guard {
		return
	}
	for _, idx := range idxs {
		if idx < 0 || idx >= len(zn.T) || zn.T[idx] == TGateClosed {
			continue
		}
		gx, gy := idx%MW, idx/MW
		if maxi(abs(g.P.X-gx), abs(g.P.Y-gy)) >= 3 {
			zn.T[idx] = TGateClosed
			g.Msg("{{K|The guards drop the portcullis again: the gate is shut by order.}}")
		}
	}
}

// ---------------------------------------------------------------- talk

func (g *Game) sqActorSiege(a *Actor) *sqSiege {
	for _, s := range g.sq().Act {
		if a.Has("sq" + strconv.Itoa(s.ID)) {
			return s
		}
	}
	if g.Cur != nil && g.Cur.Kind == "town" {
		return g.sqBesieged(g.Cur.Burg)
	}
	return nil
}

func sqTalk(g *Game, a *Actor) []Option {
	if g == nil || g.W == nil || g.Cur == nil || a == nil || a.HP <= 0 {
		return nil
	}
	st := g.sq()
	s := g.sqActorSiege(a)
	var out []Option
	if s == nil {
		if att, ok := st.Fallen[g.Cur.Burg]; ok && g.Cur.Kind == "town" && !a.Has("sqx") {
			out = append(out, Option{"Ask about the occupation.", func() {
				g.talkMenu(strings.NewReplacer("{A}", g.realmName(att)).Replace(sqOccLines[int(HashStr(fmt.Sprint(g.Seed, a.ID))%uint64(len(sqOccLines)))]))
			}})
		}
		return out
	}
	switch {
	case a.Has("sqcapt"):
		out = append(out,
			Option{"{{W|Report to the captain: the siege front.}}", func() { g.sqOpen(s, "") }},
			Option{"Ask how the siege stands.", func() { g.talkMenu(g.sqFill(s, sqPhaseDesc[s.Phase])) }})
	case a.Has("sqqm"):
		out = append(out,
			Option{"Ask about the stores.", func() { g.talkMenu(g.sqStoresLine(s)) }},
			Option{"{{W|Offer to bring food into the town.}}", func() { g.sqOpen(s, "s_start") }})
	case a.Has("sqoff"):
		out = append(out, Option{"{{W|Ask for work with the army.}}", func() { g.sqOpen(s, "a_start") }},
			Option{"Ask how the siege goes.", func() { g.talkMenu(g.sqFill(s, sqPhaseDesc[s.Phase])) }})
	case a.Has("sqgate"):
		out = append(out, Option{"Ask the guard to open the gate.", func() { g.sqGateTalk(a, s) }})
	case a.Has("sqref"):
		out = append(out, Option{"Ask what happened to them.", func() {
			g.talkMenu(sqRefugeeLines[int(HashStr(fmt.Sprint(g.Seed, ":sq:ref:", a.ID))%uint64(len(sqRefugeeLines)))])
		}})
	case a.Has("sqwound"):
		out = append(out, Option{"Bring water to the wounded.", func() { g.sqWater(s, a) }})
	case a.Has("sqarmy"):
		out = append(out, Option{"Ask what the camp says.", func() {
			g.talkMenu(sqCampLines[int(HashStr(fmt.Sprint(g.Seed, ":sq:camp:", a.ID))%uint64(len(sqCampLines)))])
		}})
	default:
		if g.Cur.Kind == "town" && !a.Has("sqx") {
			out = append(out, Option{"Ask about the siege.", func() { g.talkMenu(sqGreetLine(g, a)) }})
		}
	}
	return out
}

func (g *Game) sqStoresLine(s *sqSiege) string {
	switch {
	case s.Food >= 70:
		return "\"The granaries are fair. We can hold if nobody wastes a crust.\""
	case s.Food >= 40:
		return "\"Half rations for soldiers and a third for everyone else. Do not ask me for more.\""
	case s.Food >= 15:
		return "\"I count the sacks twice a day. They do not multiply.\""
	}
	return "\"There is nothing. We are eating the horses, and soon not even those.\""
}

func (g *Game) sqWater(s *sqSiege, a *Actor) {
	st := g.sq()
	key := fmt.Sprint(s.ID, ":water:", a.ID)
	if st.Cool[key] == g.Day() {
		g.talkMenu("\"You brought us water already. Come tomorrow, if there is any.\"")
		return
	}
	st.Cool[key] = g.Day()
	g.P.HP = mini(g.P.MaxHP, g.P.HP)
	s.Mor = clampi(s.Mor+1, 0, 100)
	g.talkMenu("They drink slowly. \"Thank you. Nobody does this.\" Morale in the town rises a little.")
	g.AddRep(stateFaction(s.Def), 2, " for helping the wounded")
}

func (g *Game) sqGateTalk(a *Actor, s *sqSiege) {
	st := g.sq()
	zn := g.Cur
	open := func() {
		bestD, bi := 1<<30, -1
		for _, idx := range st.Closed[zn.Key] {
			if idx < 0 || idx >= len(zn.T) || zn.T[idx] != TGateClosed {
				continue
			}
			d := abs(idx%MW-a.X) + abs(idx/MW-a.Y)
			if d < bestD {
				bestD, bi = d, idx
			}
		}
		st.Pass[zn.Key] = g.Day()
		if bi >= 0 {
			zn.T[bi] = TGateArch
		}
	}
	prompt := "\"The gate is shut by order,\" says the guard. \"The captain's word opens it, and no one else's.\""
	br := []tpBranch{
		{Text: "Say you serve under the captain.", Hide: func() bool { return st.Side[s.ID] != "d" }, Win: "\"Right. In you go.\" The guard hauls the portcullis up.", OnWin: open},
		{Text: "Argue that the town needs everyone it can get.", Check: "persuasion", DC: 11, Win: "The guard glances at the wall, shrugs and turns the winch.", Lose: "\"Orders. Go and talk to the captain.\"", OnWin: open},
		{Text: "Slip them a coin.", Cost: 4 * g.sqUnit(), Win: "The coin disappears and the portcullis rises a hand's breadth, then more.", OnWin: open},
	}
	g.tpChoice(prompt, br)
}

// ---------------------------------------------------------------- the screen

type sqAct struct {
	label string
	ok    bool
	why   string
	do    func()
}

func (g *Game) sqOpen(s *sqSiege, step string) {
	st := g.sq()
	g.Mode, g.Talk, g.Popup, g.Opts, g.Sel = "sq", nil, nil, nil, 0
	st.cur = &sqCur{SID: s.ID}
	if step != "" {
		g.sqBegin(s, step)
	}
}

func (g *Game) sqBegin(s *sqSiege, step string) {
	st := g.sq()
	role := step[:1]
	key := fmt.Sprint(s.ID, ":", role)
	if st.Cool[key] == g.Day() {
		g.Msg("{{K|You have done your part for today. Come back tomorrow.}}")
		if st.cur != nil {
			st.cur.Step = ""
		}
		return
	}
	cur := st.cur
	if cur == nil {
		cur = &sqCur{SID: s.ID}
		st.cur = cur
	}
	g.sqEnter(cur, step)
	g.Sel = 0
}

func (g *Game) sqEnter(cur *sqCur, step string) {
	cur.Step, cur.Res, cur.Lines, cur.Next = step, false, nil, ""
	if n, ok := sqClockSteps[step]; ok {
		cur.Clock, cur.Timed = n, true
	}
}

func sqStepByID(id string) *sqStep {
	for i := range sqSteps {
		if sqSteps[i].ID == id {
			return &sqSteps[i]
		}
	}
	return nil
}

func (g *Game) sqReq(s *sqSiege, req string) (bool, string) {
	if req == "" {
		return true, ""
	}
	var rest []string
	for _, tk := range strings.Split(req, ",") {
		f := strings.Split(tk, ":")
		if f[0] != "sq" || len(f) < 2 {
			rest = append(rest, tk)
			continue
		}
		n := 0
		if len(f) > 2 {
			n, _ = strconv.Atoi(f[2])
		}
		switch f[1] {
		case "food":
			if g.sqFoodCount() < n {
				return false, "needs food in the pack"
			}
		case "coin":
			if !g.cuCanSpend(n * g.sqUnit()) {
				return false, "cannot afford it"
			}
		case "contra":
			if len(g.uwContraband(s.Def)) == 0 {
				return false, "needs contraband"
			}
		case "phase":
			if sqPhaseIdx(s.Phase) < sqPhaseIdx(f[2]) {
				return false, "too early in the siege"
			}
		case "town":
			if g.Cur == nil || g.Cur.Kind != "town" || g.Cur.Burg != s.Burg {
				return false, "you must be inside the town"
			}
		}
	}
	if len(rest) > 0 {
		return g.crReqOK(strings.Join(rest, ","), map[string]string{})
	}
	return true, ""
}

type sqOpt struct {
	label string
	ok    bool
	why   string
}

func (g *Game) sqStepOpts(s *sqSiege, step *sqStep) []sqOpt {
	var out []sqOpt
	for _, ch := range step.Ch {
		ok, why := g.sqReq(s, ch.Req)
		lab := g.sqFill(s, ch.T)
		if ch.Chk != "" {
			f := strings.Split(ch.Chk, ":")
			dc, _ := strconv.Atoi(f[1])
			lab = g.checkLabel(f[0], dc) + " " + lab
		}
		out = append(out, sqOpt{lab, ok, why})
	}
	return out
}

func (g *Game) sqChoose(s *sqSiege, i int) {
	st := g.sq()
	cur := st.cur
	step := sqStepByID(cur.Step)
	if step == nil || i < 0 || i >= len(step.Ch) {
		return
	}
	ch := step.Ch[i]
	eff, text, next := ch.Win, ch.WT, ch.WN
	tag := ""
	failed := false
	if ch.Chk != "" {
		f := strings.Split(ch.Chk, ":")
		dc, _ := strconv.Atoi(f[1])
		if g.roll2d6Check(g.checkMod(f[0]), dc) {
			g.gainXP(4 + dc)
			tag = "{{G|[" + checkKinds[f[0]].label + ": Success]}} "
		} else {
			eff, text, next = ch.Lose, ch.LT, ch.LN
			failed = true
			tag = "{{R|[" + checkKinds[f[0]].label + ": Failure]}} "
		}
	}
	lines := g.sqApply(s, eff)
	if cur.Timed {
		cur.Clock--
		if failed {
			cur.Clock--
		}
		if cur.Clock <= 0 && next != "" && !strings.HasSuffix(next, "_late") {
			if late, ok := sqLateStep[step.ID[:1]]; ok {
				next = late
			}
		}
	}
	cur.Lines = append(Wrap(tag+g.sqFill(s, text), 96), "")
	cur.Lines = append(cur.Lines, lines...)
	cur.Next, cur.Res = next, true
	if next == "" {
		cur.Timed = false
		if eff != "" {
			st.Cool[fmt.Sprint(s.ID, ":", step.ID[:1])] = g.Day()
			st.Served++
		}
	}
}

// sqApply runs an effect string and returns the lines to show.
func (g *Game) sqApply(s *sqSiege, eff string) []string {
	var lines, rest []string
	st := g.sq()
	u := g.sqUnit()
	for _, tk := range strings.Split(eff, ";") {
		tk = strings.TrimSpace(tk)
		if tk == "" {
			continue
		}
		f := strings.SplitN(tk, ":", 3)
		num := func(i int) int {
			if i < len(f) {
				n, _ := strconv.Atoi(strings.TrimPrefix(f[i], "+"))
				return n
			}
			return 0
		}
		switch f[0] {
		case "coin":
			n := num(1) * u
			if n >= 0 {
				lines = append(lines, "{{y|+"+g.cuEarn(n)+"}}")
			} else if g.cuSpend(-n) {
				lines = append(lines, "{{y|-"+g.cuFormat(-n)+"}}")
			}
		case "xp":
			g.gainXP(num(1))
		case "hp":
			n := num(1)
			if n < 0 {
				n = mini(-n, g.P.HP-1)
				if n > 0 {
					g.hurt(g.P, n, nil, "the siege")
					lines = append(lines, fmt.Sprintf("{{r|-%d HP}}", n))
				}
			} else {
				g.P.HP = mini(g.P.MaxHP, g.P.HP+n)
			}
		case "rep":
			realm := s.Def
			if len(f) > 1 && f[1] == "a" {
				realm = s.Att
			}
			if len(f) > 2 {
				n, _ := strconv.Atoi(strings.TrimPrefix(f[2], "+"))
				g.AddRep(stateFaction(realm), n, " ("+g.realmName(realm)+")")
			}
		case "prog":
			s.Prog = clampi(s.Prog+num(1), 0, 100)
		case "mor":
			s.Mor = clampi(s.Mor+num(1), 0, 100)
		case "food":
			s.Food = clampi(s.Food+num(1), 0, 100)
		case "truce":
			s.Truce = clampi(s.Truce+num(1), 0, 100)
		case "relief":
			s.Relief = clampi(s.Relief+num(1), 0, 100)
		case "sick":
			s.Sick = clampi(s.Sick+num(1), 0, 100)
		case "merit":
			before := sqRank(st.Merit)
			st.Merit += num(1)
			if after := sqRank(st.Merit); after > before {
				lines = append(lines, "{{G|Siege rank: "+warRanks[after]+"}}")
			}
			if es := g.war(); es.Realm == s.Att || es.Realm == s.Def {
				g.warAddMerit(num(1)/2 + 1)
			}
		case "side":
			side := f[1]
			if side == "d" || side == "a" {
				if st.Side[s.ID] == "" {
					st.Side[s.ID] = side
				}
			}
		case "takefood":
			n := g.sqTakeFood(num(1))
			if n > 0 {
				s.Food = clampi(s.Food+3*n, 0, 100)
				s.Mor = clampi(s.Mor+n, 0, 100)
				lines = append(lines, fmt.Sprintf("{{K|You hand over %d rations.}}", n))
			}
		case "ride":
			switch f[1] {
			case "start":
				lines = append(lines, g.sqRideStart(s, false))
			case "fast":
				lines = append(lines, g.sqRideStart(s, true))
			case "end":
				st.Ride = nil
			}
		case "flee":
			g.sqFlee(s)
		case "deed":
			g.deed("event", g.sqFill(s, strings.Join(f[1:], ":")), 0, 0)
		case "news":
			lvSt(g).addNews(g.Day(), s.Burg, "%s", g.sqFill(s, strings.Join(f[1:], ":")))
		default:
			rest = append(rest, tk)
		}
	}
	if len(rest) > 0 {
		lines = append(lines, g.crApply(strings.Join(rest, ";"), map[string]string{"Bid": strconv.Itoa(s.Burg), "T": g.sqTown(s)})...)
	}
	return lines
}

func (g *Game) sqReliefDest(s *sqSiege) *BurgData {
	w := g.W
	bx, by, ok := g.sqBurgXY(s.Burg)
	if !ok {
		return nil
	}
	var best *BurgData
	bd := 1 << 30
	for i := range w.Burgs {
		b := &w.Burgs[i]
		if !b.Alive || b.Layer != "surface" || b.State != s.Def || b.ID == s.Burg {
			continue
		}
		if _, f := g.sq().Fallen[b.ID]; f {
			continue
		}
		d := abs(b.P%w.W-bx) + abs(b.P/w.W-by)
		if d < bd || d == bd && b.ID < best.ID {
			best, bd = b, d
		}
	}
	return best
}

func (g *Game) sqRideStart(s *sqSiege, fast bool) string {
	st := g.sq()
	dest := g.sqReliefDest(s)
	if dest == nil {
		return "{{K|There is no one left to ride to.}}"
	}
	bx, by, _ := g.sqBurgXY(s.Burg)
	km := float64(abs(dest.P%g.W.W-bx)+abs(dest.P/g.W.W-by)) * g.sqKmPer()
	pace := 45.0
	if fast {
		pace = 70
	}
	days := clampi(int(km/pace)+2, 2, 25)
	st.Ride = &sqRide{SID: s.ID, Dest: dest.ID, Due: g.Day() + days}
	return fmt.Sprintf("{{Y|You must reach %s within %d days.}}", dest.Name, days)
}

func (g *Game) sqKmPer() float64 {
	if g.W.KmPer > 0 {
		return g.W.KmPer
	}
	return 160
}

func (g *Game) sqFlee(s *sqSiege) {
	x, y, ok := g.sqBurgXY(s.Burg)
	if !ok {
		return
	}
	g.sq().cur = nil
	g.Mode = "main"
	for _, d := range [][2]int{{0, -1}, {1, 0}, {0, 1}, {-1, 0}, {1, 1}, {-1, -1}, {1, -1}, {-1, 1}} {
		nx, ny := x+d[0], y+d[1]
		if !g.W.In(nx, ny) || g.W.IsWater(g.W.Idx(nx, ny)) {
			continue
		}
		zn := g.GetZone(nx, ny, 1, 1, 0)
		g.Msg("{{Y|You slip out through the old postern tunnel and come up among the weeds beyond the ditch.}}")
		g.enterZone(zn, MW/2, MH/2, true)
		return
	}
}

func (g *Game) sqHubActs(s *sqSiege) []sqAct {
	st := g.sq()
	dist := g.sqDist(s)
	side := st.Side[s.ID]
	ended := s.Phase == "sack"
	gate := func(maxD int, step string, block bool, blockWhy string, extra func() (bool, string)) sqAct {
		a := sqAct{ok: true}
		switch {
		case ended:
			a.ok, a.why = false, "the town has fallen"
		case dist > maxD:
			a.ok, a.why = false, "too far from "+g.sqTown(s)
		case block:
			a.ok, a.why = false, blockWhy
		}
		if a.ok && extra != nil {
			a.ok, a.why = extra()
		}
		a.do = func() { g.sqBegin(s, step) }
		return a
	}
	var acts []sqAct
	add := func(label string, a sqAct) {
		a.label = label
		acts = append(acts, a)
	}
	add("Serve the garrison of {T} (guard duty, water, repairs).", gate(1, "d_start", side == "a", "you serve the besiegers", nil))
	add("Lead or join a sortie from the town.", gate(1, "x_start", side == "a", "you serve the besiegers", func() (bool, string) { return g.sqReq(s, "sq:town") }))
	add("Take the pay of {A}'s army (scout, sap, forage, assault).", gate(2, "a_start", side == "d", "you serve the garrison", nil))
	add("Run food or people through the lines.", gate(2, "s_start", false, "", nil))
	add("Carry terms between the camps, or broker a truce.", gate(2, "e_start", false, "", nil))
	add("Ride for a relief force.", gate(1, "r_start", side == "a", "you serve the besiegers", func() (bool, string) {
		if st.Ride != nil {
			return false, "you already carry a letter"
		}
		return g.sqReq(s, "sq:town")
	}))
	if r := st.Ride; r != nil {
		if b := g.W.burgByID[r.Dest]; b != nil {
			ok := g.Cur != nil && g.Cur.Kind == "town" && g.Cur.Burg == r.Dest
			why := ""
			if !ok {
				why = "you are not in " + b.Name
			}
			rs := g.sqByID(r.SID)
			acts = append(acts, sqAct{label: "Deliver the letter to the lord at " + b.Name + ".", ok: ok && rs != nil, why: why, do: func() {
				g.sqBegin(rs, "r_arrive")
			}})
		}
	}
	if s.Phase == "sack" {
		inTown := g.Cur != nil && g.Cur.Kind == "town" && g.Cur.Burg == s.Burg
		acts = append(acts, sqAct{label: "Flee the sack by the postern tunnel.", ok: inTown, why: "you are not in the town", do: func() { g.sqFlee(s) }})
	}
	return acts
}

func sqKeys(g *Game, k string) {
	st := g.sq()
	s := g.sqFocus()
	if s == nil {
		g.Mode = "main"
		st.cur = nil
		return
	}
	if st.cur == nil {
		st.cur = &sqCur{SID: s.ID}
	}
	cur := st.cur
	if cur.Step != "" {
		step := sqStepByID(cur.Step)
		if step == nil {
			cur.Step = ""
			return
		}
		if cur.Res {
			if k == "Enter" || k == " " || k == "Escape" {
				if cur.Next != "" {
					g.sqEnter(cur, cur.Next)
				} else {
					cur.Step, cur.Res, cur.Timed = "", false, false
				}
				g.Sel = 0
			}
			return
		}
		opts := g.sqStepOpts(s, step)
		pick := func(i int) {
			if i < 0 || i >= len(opts) {
				return
			}
			if !opts[i].ok {
				g.Msg("{{K|You cannot choose that: " + opts[i].why + ".}}")
				return
			}
			g.sqChoose(s, i)
		}
		if i := letterIndex(k); i >= 0 {
			pick(i)
			return
		}
		switch k {
		case "Escape":
			cur.Step, cur.Timed = "", false
		case "ArrowUp":
			g.Sel = maxi(0, g.Sel-1)
		case "ArrowDown":
			g.Sel = mini(len(opts)-1, g.Sel+1)
		case "Enter", " ":
			pick(g.Sel)
		}
		return
	}
	acts := g.sqHubActs(s)
	pick := func(i int) {
		if i < 0 || i >= len(acts) {
			return
		}
		if !acts[i].ok {
			g.Msg("{{K|You cannot do that now: " + acts[i].why + ".}}")
			return
		}
		acts[i].do()
	}
	if i := letterIndex(k); i >= 0 {
		pick(i)
		return
	}
	switch k {
	case "Escape":
		st.cur = nil
		g.Mode = "main"
	case "ArrowUp":
		g.Sel = maxi(0, g.Sel-1)
	case "ArrowDown":
		g.Sel = mini(len(acts)-1, g.Sel+1)
	case "Enter", " ":
		pick(g.Sel)
	}
}

func sqWalls(p int) string {
	switch {
	case p < 20:
		return "untouched"
	case p < 45:
		return "scarred"
	case p < 70:
		return "battered"
	}
	return "breaking"
}

func (g *Game) sqDraw() {
	sc := g.sc
	st := g.sq()
	s := g.sqFocus()
	if s == nil || st.cur == nil {
		return
	}
	cur := st.cur
	x, y, w, _ := g.panel("Siege front")
	line := func(t string) {
		for _, l := range Wrap(t, w-2) {
			sc.TextB(x, y, l, CGray, CBgDark, x+w)
			y++
		}
	}
	if cur.Step != "" {
		step := sqStepByID(cur.Step)
		if step == nil {
			return
		}
		sc.TextB(x, y, "{{M|Siege of "+g.sqTown(s)+"}}", CGray, CBgDark, x+w)
		if cur.Timed && !cur.Res {
			sc.TextB(x+w-22, y, fmt.Sprintf("{{R|Time left: %d}}", maxi(0, cur.Clock)), CGray, CBgDark, x+w)
		}
		y += 2
		if cur.Res {
			for _, l := range cur.Lines {
				sc.TextB(x, y, l, CGray, CBgDark, x+w)
				y++
			}
			sc.TextB(x, y+1, "{{K|Enter to go on.}}", CGray, CBgDark, x+w)
			return
		}
		line(g.sqFill(s, step.Text))
		y++
		opts := g.sqStepOpts(s, step)
		g.Sel = clampi(g.Sel, 0, maxi(0, len(opts)-1))
		for i, op := range opts {
			bg := CBgDark
			if i == g.Sel {
				bg = CBgSel
			}
			var ls []string
			if op.ok {
				ls = Wrap(fmt.Sprintf("{{W|%c)}} %s", 'a'+i, op.label), w-4)
			} else {
				ls = Wrap(fmt.Sprintf("{{K|%c) %s (%s)}}", 'a'+i, Plain(op.label), op.why), w-4)
			}
			for j, l := range ls {
				sc.Fill(x, y, w, 1, bg)
				if j > 0 {
					l = "   " + l
				}
				sc.TextB(x, y, l, CGray, bg, x+w)
				y++
			}
		}
		sc.TextB(x, y+1, "{{K|Letter or Enter to choose. Escape to step back.}}", CGray, CBgDark, x+w)
		return
	}
	sc.TextB(x, y, "{{M|Siege of "+g.sqTown(s)+"}} {{K|"+g.realmName(s.Att)+" against "+g.realmName(s.Def)+"}}", CGray, CBgDark, x+w)
	y += 2
	line(g.sqFill(s, sqPhaseDesc[s.Phase]))
	y++
	line(fmt.Sprintf("Day %d of about %d. Walls: %s. Stores %s %d%%. Morale %s %d%%.", maxi(1, g.Day()-s.Start+1), s.Dur, sqWalls(s.Prog), sqBar(s.Food), s.Food, sqBar(s.Mor), s.Mor))
	var extra []string
	if s.Truce > 0 {
		extra = append(extra, "Truce talks "+sqBar(s.Truce))
	}
	if s.Relief > 0 {
		extra = append(extra, "Relief gathering "+sqBar(s.Relief))
	}
	if s.Sick > 30 {
		extra = append(extra, "fever in the camp")
	}
	if len(extra) > 0 {
		line("{{K|" + strings.Join(extra, ". ") + ".}}")
	}
	side := map[string]string{"d": "the garrison", "a": "the besiegers", "": "no one"}[st.Side[s.ID]]
	line(fmt.Sprintf("{{K|You serve %s. Siege rank: %s (%d merit). Duties done: %d.}}", side, warRanks[sqRank(st.Merit)], st.Merit, st.Served))
	if r := st.Ride; r != nil {
		if b := g.W.burgByID[r.Dest]; b != nil {
			line(fmt.Sprintf("{{Y|You carry a letter for %s: %d days left.}}", b.Name, maxi(0, r.Due-g.Day())))
		}
	}
	y++
	acts := g.sqHubActs(s)
	g.Sel = clampi(g.Sel, 0, maxi(0, len(acts)-1))
	for i, a := range acts {
		bg := CBgDark
		if i == g.Sel {
			bg = CBgSel
		}
		lab := g.sqFill(s, a.label)
		var ls []string
		if a.ok {
			ls = Wrap(fmt.Sprintf("{{W|%c)}} %s", 'a'+i, lab), w-4)
		} else {
			ls = Wrap(fmt.Sprintf("{{K|%c) %s (%s)}}", 'a'+i, lab, a.why), w-4)
		}
		for j, l := range ls {
			sc.Fill(x, y, w, 1, bg)
			if j > 0 {
				l = "   " + l
			}
			sc.TextB(x, y, l, CGray, bg, x+w)
			y++
		}
	}
	sc.TextB(x, y+1, "{{K|Letter or Enter to choose. Escape to leave.}}", CGray, CBgDark, x+w)
}
