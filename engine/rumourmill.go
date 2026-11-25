package engine

import (
	"fmt"
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// Rumour network (prefix rm). Real happenings become rumours with a source town, a day and a truth value:
// wars and musters between realms (war.go), worldsim events (plague, invasion, famine, bandits, boom,
// bust, slave pens), deaths of notables, caravans lost or raided (caravans.go), the wanderer's own deeds,
// beasts seen near their lairs, counter squads asking after the wanderer (counters.go), burned or saved
// towns (regrowth.go), feast days and the tidings in lvSt(g).News. A rumour is stored once; what a town
// hears is a pure function of (rumour, town): it arrives after distance / 40 km a day, fades after about
// 20 days, and is retold with distortion that grows with the parasangs it travelled (names blurred,
// numbers inflated, the place lost, now and then a false twist). Nothing here draws g.R: all picks are
// hashes of the seed. NPCs share the local pool through "Ask about the rumours" (a talk option) and
// rmLine(g, a) for the barks. Heard rumours are listed on the Rumours screen (mode "rm", palette
// "Rumours"); walking to where one happened checks it, pays a little XP and rep and puts the true account
// into the case notes (knLearn). Rumours of beasts and bandits feed the contract board through
// rmContractSeeds. State g.ext("rm"), planet-local ("rm" in unLocalKeys).

func init() {
	unLocalKeys = append(unLocalKeys, "rm")
	turnHooks = append(turnHooks, rmTurn)
	enterHooks = append(enterHooks, rmEnter)
	talkHooks = append(talkHooks, rmTalk)
	modeKeys["rm"] = rmKeys
	modeDraws["rm"] = func(g *Game) { g.rmDraw() }
	paEntries = append(paEntries, paEntry{Name: "Rumours", Keywords: "rumors gossip heard tidings news verify truth", Cat: "Screen", Label: "screen",
		Func: func(g *Game) { g.rm().sel = 0; g.Mode = "rm" }})
	saRumourSink = rmSink
	fqRumourSink = rmSink
	dzRumourSink = rmSink
	mgRumourSink = rmSink
	sqRumourSink = rmSink
	htRumourSink = rmSink
}

// rmSink receives external rumours from socialplay, factionquests, afflictions, monsterecology, sieges and heists.
func rmSink(g *Game, kind, subject, text string) {
	if g == nil || g.W == nil || text == "" {
		return
	}
	burg := 0
	if strings.HasPrefix(subject, "burg:") {
		if id, err := strconv.Atoi(strings.TrimPrefix(subject, "burg:")); err == nil && g.W.burgByID[id] != nil {
			burg = id
		}
	}
	if burg == 0 && g.Cur != nil && g.Cur.Burg > 0 {
		burg = g.Cur.Burg
	}
	if burg == 0 {
		if b := g.rmNearestBurg(g.W.Idx(g.WX, g.WY)); b != nil {
			burg = b.ID
		}
	}
	b := g.W.burgByID[burg]
	if b == nil {
		return
	}
	key := fmt.Sprintf("sink:%s:%s:%d:%d", kind, subject, lvDay(g), HashStr(text)%99991)
	g.rmAdd("free", key, burg, rmRum{P: b.P, Place: b.Name, A: subject, X: text})
}

// rmSeed is what the contract board reads: a hunt suggested by a rumour in the air of a town.
type rmSeed struct {
	RID   int
	Kind  string
	Text  string
	Name  string
	P     int
	Place string
	Truth int
}

type rmRum struct {
	ID    int    `json:"i"`
	Kind  string `json:"k"`
	Key   string `json:"ke"`
	Burg  int    `json:"b"`
	P     int    `json:"p"`
	Day   int    `json:"d"`
	A     string `json:"a,omitempty"`
	B     string `json:"o,omitempty"`
	Place string `json:"pl,omitempty"`
	Alt   string `json:"al,omitempty"`
	N     int    `json:"n,omitempty"`
	X     string `json:"x,omitempty"`
	Truth int    `json:"t"`
	False bool   `json:"fl,omitempty"`
	Life  int    `json:"li"`
}

type rmHeard struct {
	RID     int    `json:"r"`
	Kind    string `json:"k"`
	Town    int    `json:"t"`
	Day     int    `json:"d"`
	Text    string `json:"x"`
	Hops    int    `json:"h"`
	Lie     bool   `json:"l,omitempty"`
	Ver     int    `json:"v,omitempty"` // 1 true, -1 false
	Corr    int    `json:"c,omitempty"`
	Diff    int    `json:"df,omitempty"`
	Via     string `json:"w,omitempty"`
	Real    string `json:"re,omitempty"`
	SrcBurg int    `json:"sb"`
	SrcDay  int    `json:"sd"`
	P       int    `json:"p"`
}

type rmState struct {
	Next  int             `json:"n"`
	Rums  []*rmRum        `json:"r,omitempty"`
	Heard []*rmHeard      `json:"h,omitempty"`
	Seen  map[string]bool `json:"s,omitempty"`
	Asked map[string]int  `json:"a,omitempty"`
	Init  bool            `json:"in,omitempty"`
	Hour  int             `json:"ho,omitempty"`
	Day   int             `json:"da,omitempty"`
	Deeds int             `json:"dd,omitempty"`
	Sent  int             `json:"se,omitempty"`
	Vers  int             `json:"ve,omitempty"`

	sel      int
	poolTown int
	poolDay  int
	poolNext int
	poolLen  int
	pool     []*rmRum
}

func (g *Game) rm() *rmState {
	s := g.ext("rm", func() any { return &rmState{Seen: map[string]bool{}, Asked: map[string]int{}} }).(*rmState)
	if s.Seen == nil {
		s.Seen = map[string]bool{}
	}
	if s.Asked == nil {
		s.Asked = map[string]int{}
	}
	return s
}

// ---------------------------------------------------------------- geometry and text

func (g *Game) rmKm(a, b *BurgData) float64 {
	if a == nil || b == nil {
		return 0
	}
	w := g.W
	dx, dy := float64(a.P%w.W-b.P%w.W), float64(a.P/w.W-b.P/w.W)
	return math.Hypot(dx, dy) * g.tvKm()
}

func (g *Game) rmHops(r *rmRum, town int) int {
	sb, tb := g.W.burgByID[r.Burg], g.W.burgByID[town]
	if sb == nil || tb == nil {
		return 0
	}
	par := g.rmKm(sb, tb) / g.tvKm()
	return clampi(int(par/3), 0, 5)
}

func rmDir(dx, dy int) string {
	if dx == 0 && dy == 0 {
		return "nearby"
	}
	ang := math.Atan2(float64(dy), float64(dx)) * 180 / math.Pi
	i := int(math.Round(ang/45)+8) % 8
	return rmDirs[i]
}

func rmCanTwist(kind string) bool { return kind != "counter" && kind != "feast" }

func rmCanBeFalse(kind string) bool {
	switch kind {
	case "free", "counter", "feast", "regrow":
		return false
	}
	return true
}

var rmNumRe = regexp.MustCompile(`\b\d{1,3}\b`)

func rmScale(s string, pct int) string {
	return rmNumRe.ReplaceAllStringFunc(s, func(m string) string {
		n := 0
		fmt.Sscanf(m, "%d", &n)
		return fmt.Sprint(n * pct / 100)
	})
}

func rmCut(s string, n int) string {
	rs := []rune(s)
	if len(rs) <= n {
		return s
	}
	return string(rs[:maxi(1, n-1)]) + "…"
}

// rmText tells a rumour: hops retellings old, twisted or not, as heard in town.
func (g *Game) rmText(r *rmRum, hops int, twist bool, town int) string {
	tp, ok := rmTpls[r.Kind]
	if !ok {
		tp = rmTpls["free"]
	}
	h := HashStr(fmt.Sprint(g.Seed, ":rm:t:", r.ID, ":", town))
	if r.Kind == "free" {
		s := r.X
		if hops >= 2 {
			for _, sep := range []string{", ", "; ", " - "} {
				if i := strings.Index(s, sep); i > 0 && len(s)-i > 30 {
					s = s[:i] + "."
					break
				}
			}
		}
		s = rmScale(s, 100+20*hops)
		if twist {
			s = rmScale(s, 300)
			s = strings.TrimSuffix(s, ".") + ", and worse, some say."
		}
		return s
	}
	a, b, place, n, x := r.A, r.B, r.Place, r.N, r.X
	if twist {
		switch r.Kind {
		case "battle", "muster":
			a, b = b, a
		case "death":
			b = rmCauseAlt[int(h%uint64(len(rmCauseAlt)))]
		default:
			if r.Alt != "" {
				place = r.Alt
			}
			n *= 3
		}
	}
	n = n * (100 + 20*hops) / 100
	x = strings.ReplaceAll(x, "{n}", fmt.Sprint(n))
	tpl := tp.T
	if hops >= 2 {
		tpl = tp.B
	}
	if hops >= 3 && place != "" {
		sb, tb := g.W.burgByID[r.Burg], g.W.burgByID[town]
		if sb != nil && tb != nil {
			w := g.W
			place = "somewhere " + rmDir(sb.P%w.W-tb.P%w.W, sb.P/w.W-tb.P/w.W) + " of here"
		} else {
			place = "somewhere out there"
		}
	}
	return strings.NewReplacer("{a}", a, "{b}", b, "{place}", place, "{n}", fmt.Sprint(n), "{x}", x).Replace(tpl)
}

type rmView struct {
	Text string
	Hops int
	Lie  bool
}

// rmView is what the people of a town say about a rumour: pure function of (rumour, town).
func (g *Game) rmView(r *rmRum, town int) rmView {
	hops := g.rmHops(r, town)
	h := HashStr(fmt.Sprint(g.Seed, ":rm:v:", r.ID, ":", town))
	twist := r.False
	if !twist && hops >= 1 && rmCanTwist(r.Kind) && int(h%100) < mini(8*hops, 35) {
		twist = true
	}
	return rmView{Text: g.rmText(r, hops, twist, town), Hops: hops, Lie: twist}
}

// ---------------------------------------------------------------- creation

func (g *Game) rmNearestBurg(p int) *BurgData {
	var best *BurgData
	bd := 1 << 30
	for _, b := range g.ws().burgList(g) {
		if d := wsCheb(g.W, b.P, p); d < bd || (d == bd && best != nil && b.ID < best.ID) {
			best, bd = b, d
		}
	}
	return best
}

func (g *Game) rmAlt(burg int, salt string) string {
	sb := g.W.burgByID[burg]
	if sb == nil {
		return ""
	}
	var near []*BurgData
	for _, b := range g.ws().burgList(g) {
		if b.ID != burg && wsCheb(g.W, b.P, sb.P) <= 7 {
			near = append(near, b)
		}
	}
	if len(near) == 0 {
		return ""
	}
	sort.Slice(near, func(i, j int) bool { return near[i].ID < near[j].ID })
	return near[int(HashStr(fmt.Sprint(g.Seed, ":rm:alt:", salt))%uint64(len(near)))].Name
}

func (g *Game) rmAdd(kind, key string, burg int, f rmRum) *rmRum {
	st := g.rm()
	if burg <= 0 || g.W.burgByID[burg] == nil || st.Seen[key] {
		return nil
	}
	if len(st.Seen) > 500 {
		fresh := make(map[string]bool, len(st.Rums)+16)
		for _, r := range st.Rums {
			if r != nil && r.Key != "" {
				fresh[r.Key] = true
			}
		}
		st.Seen = fresh
	}
	st.Seen[key] = true
	tp, ok := rmTpls[kind]
	if !ok {
		kind, tp = "free", rmTpls["free"]
	}
	st.Next++
	f.ID, f.Kind, f.Key, f.Burg, f.Day = st.Next, kind, key, burg, lvDay(g)
	base := tp.Base
	if base == 0 {
		base = 85
	}
	f.Truth = base
	f.Life = tp.Life
	if f.Life == 0 {
		f.Life = 20 + int(HashStr(key)%6)
	}
	if f.Alt == "" {
		f.Alt = g.rmAlt(burg, key)
	}
	if rmCanBeFalse(kind) && int(HashStr(fmt.Sprint(g.Seed, ":rm:f:", key))%100) >= base {
		f.False = true
	}
	st.Rums = append(st.Rums, &f)
	if len(st.Rums) > 120 {
		st.Rums = st.Rums[len(st.Rums)-120:]
	}
	return &f
}

// ---------------------------------------------------------------- the pool of a town

func (g *Game) rmPool(town int) []*rmRum {
	st := g.rm()
	tb := g.W.burgByID[town]
	if tb == nil {
		return nil
	}
	day := lvDay(g)
	if st.pool != nil && st.poolTown == town && st.poolDay == day && st.poolNext == st.Next && st.poolLen == len(st.Rums) {
		return st.pool
	}
	var out []*rmRum
	for _, r := range st.Rums {
		sb := g.W.burgByID[r.Burg]
		if sb == nil {
			continue
		}
		age := day - r.Day
		arrive := int(math.Ceil(g.rmKm(sb, tb) / 40))
		if age >= arrive && age <= r.Life {
			out = append(out, r)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Day != out[j].Day {
			return out[i].Day > out[j].Day
		}
		return out[i].ID > out[j].ID
	})
	st.poolTown, st.poolDay, st.poolNext, st.poolLen, st.pool = town, day, st.Next, len(st.Rums), out
	return out
}

func (g *Game) rmHear(r *rmRum, town int, via string) *rmHeard {
	st := g.rm()
	v := g.rmView(r, town)
	for _, h := range st.Heard {
		if h.RID != r.ID {
			continue
		}
		if h.Town == town {
			return h
		}
		if h.Text == v.Text {
			h.Corr++
		} else {
			h.Diff++
		}
		if h.Ver == 0 && v.Hops < h.Hops {
			h.Text, h.Hops, h.Lie, h.Town, h.Day = v.Text, v.Hops, v.Lie, town, lvDay(g)
		}
		return h
	}
	nh := &rmHeard{RID: r.ID, Kind: r.Kind, Town: town, Day: lvDay(g), Text: v.Text, Hops: v.Hops, Lie: v.Lie, Via: via,
		Real: g.rmText(r, 0, false, r.Burg), SrcBurg: r.Burg, SrcDay: r.Day, P: r.P}
	st.Heard = append(st.Heard, nh)
	if len(st.Heard) > 80 {
		st.Heard = st.Heard[len(st.Heard)-80:]
	}
	return nh
}

// ---------------------------------------------------------------- public helpers for other files

// rmLine is one rumour of the local pool for a speaker, as a plain sentence ("" when there is none). The
// barks call it; hearing it puts it on the Rumours screen.
func rmLine(g *Game, a *Actor) string {
	if g == nil || g.W == nil || g.Cur == nil || a == nil || g.Cur.Kind != "town" || g.Cur.Burg <= 0 {
		return ""
	}
	pool := g.rmPool(g.Cur.Burg)
	if len(pool) == 0 {
		return ""
	}
	h := HashStr(fmt.Sprint(g.Seed, ":rm:line:", a.ID, ":", g.Turn/360))
	r := pool[int(h%uint64(len(pool)))]
	nh := g.rmHear(r, g.Cur.Burg, "overheard")
	return "Have you heard? " + nh.Text
}

// rmContractSeeds lists up to two beast or bandit rumours in the air of a town for the contract board.
func rmContractSeeds(g *Game, burg int) []rmSeed {
	if g == nil || g.W == nil {
		return nil
	}
	var out []rmSeed
	for _, r := range g.rmPool(burg) {
		if (r.Kind == "beast" || r.Kind == "ev:bandits") && r.P >= 0 && r.P < g.W.W*g.W.H {
			v := g.rmView(r, burg)
			out = append(out, rmSeed{RID: r.ID, Kind: "hunt", Text: v.Text, Name: r.A, P: r.P, Place: r.Place, Truth: r.Truth})
			if len(out) >= 2 {
				break
			}
		}
	}
	return out
}

// rmConfirm settles a rumour in the notes because the wanderer found the thing it spoke of.
func rmConfirm(g *Game, rid int) {
	if g == nil || g.W == nil {
		return
	}
	for _, h := range g.rm().Heard {
		if h.RID == rid && h.Ver == 0 {
			g.rmVerify(h)
		}
	}
}

// ---------------------------------------------------------------- field checks

func (g *Game) rmTownName(id int) string {
	if b := g.W.burgByID[id]; b != nil {
		return b.Name
	}
	return "a distant town"
}

func (g *Game) rmVerify(h *rmHeard) {
	if h.Ver != 0 {
		return
	}
	st := g.rm()
	st.Vers++
	from := &lvPerson{Key: fmt.Sprintf("rm:%d", h.Town), Name: "talk in " + g.rmTownName(h.Town)}
	kn := knSt(g)
	subj := fmt.Sprintf("burg:%d", h.SrcBurg)
	if h.Lie {
		h.Ver = -1
		f := kn.add(knFact{Kind: "rumour", Subject: subj, Text: h.Text, True: false, Day: lvDay(g), Burg: h.SrcBurg, Ver: -1})
		g.knLearn(f, from, 1, "gossip")
	} else {
		h.Ver = 1
	}
	f := kn.add(knFact{Kind: "rumour", Subject: subj, Text: h.Real, True: true, Day: lvDay(g), Burg: h.SrcBurg, Ver: 1})
	g.knLearn(f, from, 2, "gossip")
	xp := 6 + g.P.Level/2
	if h.Ver == 1 {
		g.Msg("{{G|The rumour holds up:}} " + h.Real)
		if g.W.In(g.WX, g.WY) {
			if sid := g.W.State[g.W.Idx(g.WX, g.WY)]; sid > 0 {
				g.AddRep(fmt.Sprintf("state:%d", sid), 1, " for setting a rumour straight")
			}
		}
	} else {
		xp = 3 + g.P.Level/4
		g.Msg("{{y|It is not as you heard:}} " + h.Real)
	}
	g.gainXP(xp)
	g.Msg(fmt.Sprintf("{{c|Noted in your case notes. (+%d xp)}}", xp))
}

func rmEnter(g *Game, zn *Zone) {
	if g.W == nil || g.P == nil || zn == nil || !g.W.In(zn.WX, zn.WY) {
		return
	}
	st := g.rm()
	if len(st.Heard) == 0 {
		return
	}
	p := g.W.Idx(zn.WX, zn.WY)
	for _, h := range st.Heard {
		if h.Ver == 0 && h.P >= 0 && h.P == p {
			g.rmVerify(h)
		}
	}
}

// ---------------------------------------------------------------- talk

func rmTalk(g *Game, a *Actor) []Option {
	if g.W == nil || g.Cur == nil || g.P == nil || a == nil || a.HP <= 0 || a.Angry || a.AI == "hostile" {
		return nil
	}
	if g.Cur.Kind != "town" || g.Cur.Burg <= 0 {
		return nil
	}
	town := g.Cur.Burg
	out := []Option{{"Ask about the rumours on the road", func() { g.rmAsk(a, town) }}}
	if len(g.rm().Heard) > 0 {
		out = append(out, Option{"Ask what to make of the rumours you have heard (screen)", func() { g.rm().sel = 0; g.Mode = "rm" }})
	}
	return out
}

func (g *Game) rmAsk(a *Actor, town int) {
	st := g.rm()
	day := lvDay(g)
	key := fmt.Sprintf("%d:%d", a.ID, day)
	if st.Asked[key] > 0 {
		g.Msg("\"Nothing new since this morning.\"")
		return
	}
	if len(st.Asked) > 300 {
		st.Asked = map[string]int{}
	}
	st.Asked[key] = 1
	pool := g.rmPool(town)
	if len(pool) == 0 {
		g.Msg("\"Quiet times. Nothing but the weather and the price of grain.\"")
		return
	}
	h := HashStr(fmt.Sprint(g.Seed, ":rm:ask:", a.ID, ":", day))
	g.Msg(Pick(NewRng(fmt.Sprint(g.Seed, ":rm:teller:", h)), rmTellers))
	n := mini(2, len(pool))
	start := int(h % uint64(len(pool)))
	for i := 0; i < n; i++ {
		r := pool[(start+i)%len(pool)]
		had := false
		for _, o := range st.Heard {
			if o.RID == r.ID {
				had = true
			}
		}
		nh := g.rmHear(r, town, "ask")
		g.Msg("\"" + nh.Text + "\"")
		if !had {
			g.Msg("{{c|(Added to your rumours.)}}")
		}
	}
}

// ---------------------------------------------------------------- feeding the network

func rmTurn(g *Game) {
	if g.W == nil || g.P == nil {
		return
	}
	st := g.rm()
	hr := g.Turn / 60
	if st.Init && hr == st.Hour {
		return
	}
	first := !st.Init
	st.Init, st.Hour = true, hr
	day := lvDay(g)
	g.rmPollNews(st, day)
	g.rmPollEvents(st)
	g.rmPollCaravans(st)
	g.rmPollDeeds(st)
	g.rmPollMisc(st)
	if first || day != st.Day {
		st.Day = day
		g.rmDaily(st, day)
	}
}

func (g *Game) rmBurgP(id int) int {
	if b := g.W.burgByID[id]; b != nil {
		return b.P
	}
	return -1
}

func (g *Game) rmPollNews(st *rmState, day int) {
	added := 0
	for _, n := range lvSt(g).News {
		if n.Day < day-8 || n.Text == "" || added >= 4 {
			continue
		}
		key := fmt.Sprintf("n:%d:%d:%d", n.Day, n.Burg, HashStr(n.Text)%100000)
		if g.rmAdd("free", key, n.Burg, rmRum{X: n.Text, P: g.rmBurgP(n.Burg)}) != nil {
			added++
		}
	}
}

func (g *Game) rmPollEvents(st *rmState) {
	hour := g.wsHour()
	for _, e := range g.ws().Ev {
		kind, ok := rmEvKinds[e.Kind]
		if !ok || e.P < 0 || e.P >= g.W.W*g.W.H || e.Start < hour-12*24 {
			continue
		}
		b := g.rmNearestBurg(e.P)
		if b == nil {
			continue
		}
		g.rmAdd(kind, fmt.Sprintf("ev:%d", e.ID), b.ID, rmRum{Place: g.wsPlace(e.P), P: e.P, N: 8 + e.Str*7})
	}
}

func (g *Game) rmPollCaravans(st *rmState) {
	cv := g.cv()
	reg := cvReg(g)
	hour := g.Turn / 60
	var ids []int
	for id, hr := range cv.Dead {
		if hr >= hour-10*24 {
			ids = append(ids, id)
		}
	}
	sort.Ints(ids)
	for _, id := range ids {
		c := reg.byID[id]
		if c == nil || c.Home == nil || c.Dest == nil {
			continue
		}
		g.rmAdd("lost", fmt.Sprintf("cd:%d", id), c.Home.ID, rmRum{A: c.Name, Place: c.Dest.Name, N: c.Wagons, P: -1})
	}
	var keys []string
	for k, hr := range cv.Robbed {
		if hr >= hour-10*24 {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	for _, k := range keys {
		var id int
		if _, err := fmt.Sscanf(k, "%d:", &id); err != nil {
			continue
		}
		c := reg.byID[id]
		if c == nil || c.Home == nil || c.Dest == nil {
			continue
		}
		g.rmAdd("raid", "cr:"+k, c.Dest.ID, rmRum{A: c.Name, Place: c.Dest.Name, N: maxi(1, c.Wagons/2), P: -1})
	}
}

func (g *Game) rmPollDeeds(st *rmState) {
	if st.Deeds > len(g.Deeds) {
		st.Deeds = len(g.Deeds)
	}
	from := maxi(st.Deeds, len(g.Deeds)-3)
	for i := from; i < len(g.Deeds); i++ {
		d := g.Deeds[i]
		if d.Kind == "kn" || d.Planet != g.W.uniIdx || d.Cell < 0 || d.Cell >= g.W.W*g.W.H || d.Text == "" {
			continue
		}
		if b := g.rmNearestBurg(d.Cell); b != nil {
			g.rmAdd("free", fmt.Sprintf("d:%d:%d", i, HashStr(d.Text)%1000), b.ID, rmRum{X: d.Text, P: d.Cell})
		}
	}
	st.Deeds = len(g.Deeds)
}

func (g *Game) rmPollMisc(st *rmState) {
	if s := g.ct().Sent; s > st.Sent {
		st.Sent = s
		burg := 0
		if g.Cur != nil && g.Cur.Kind == "town" {
			burg = g.Cur.Burg
		}
		if burg <= 0 && g.W.In(g.WX, g.WY) {
			if b := g.rmNearestBurg(g.W.Idx(g.WX, g.WY)); b != nil {
				burg = b.ID
			}
		}
		g.rmAdd("counter", fmt.Sprintf("ct:%d", s), burg, rmRum{P: -1})
	}
	rg := g.rg()
	var ids []int
	for id := range rg.Burned {
		ids = append(ids, id)
	}
	sort.Ints(ids)
	for _, id := range ids {
		b := g.W.burgByID[id]
		if b == nil {
			continue
		}
		x := "was burned"
		if rg.Why[id] == "blood" {
			x = "ran red"
		}
		g.rmAdd("regrow", fmt.Sprintf("rg:b:%d:%d", id, rg.Burned[id]), id, rmRum{A: b.Name, Place: b.Name, X: x, P: b.P})
	}
	ids = ids[:0]
	for id := range rg.Saved {
		ids = append(ids, id)
	}
	sort.Ints(ids)
	for _, id := range ids {
		b := g.W.burgByID[id]
		if b == nil {
			continue
		}
		g.rmAdd("regrow", fmt.Sprintf("rg:s:%d:%d", id, rg.Saved[id]), id, rmRum{A: b.Name, Place: b.Name, X: "was pulled back from ruin", P: b.P})
	}
}

func (g *Game) rmDaily(st *rmState, day int) {
	w := g.W
	// wars and musters between realms
	emitted := 0
	for i := range w.States {
		s := &w.States[i]
		if !s.Alive || emitted >= 2 {
			continue
		}
		for _, e := range g.warsOf(s.ID) {
			if e <= s.ID || emitted >= 2 {
				continue
			}
			bucket := day / 12
			h := HashStr(fmt.Sprint(g.Seed, ":rm:war:", s.ID, ":", e, ":", bucket))
			if h%100 >= 60 {
				continue
			}
			sa, sb := w.stateByID[s.ID], w.stateByID[e]
			if sa == nil || sb == nil {
				continue
			}
			ca, cb := w.burgByID[sa.Capital], w.burgByID[sb.Capital]
			if ca == nil || cb == nil {
				continue
			}
			mx, my := (ca.P%w.W+cb.P%w.W)/2, (ca.P/w.W+cb.P/w.W)/2
			mid := w.Idx(mx, my)
			if w.IsWater(mid) {
				mid = ca.P
			}
			kind := "battle"
			if (h/100)%2 == 0 {
				kind = "muster"
			}
			g.rmAdd(kind, fmt.Sprintf("war:%d:%d:%d", s.ID, e, bucket), ca.ID,
				rmRum{A: g.realmName(s.ID), B: g.realmName(e), Place: w.RegionName(mid), Alt: cb.Name, N: 40 + int(h/200%360), P: mid})
			emitted++
		}
	}
	// beasts seen near their lairs
	if len(w.Monsters) > 0 {
		h := HashStr(fmt.Sprint(g.Seed, ":rm:beast:", day))
		if h%100 < 55 {
			m := &w.Monsters[int(h/100)%len(w.Monsters)]
			if !m.Slain && !g.Slain[m.ID] && m.P >= 0 && m.P < w.W*w.H && !w.IsWater(m.P) && (m.Layer == "" || m.Layer == "surface") {
				if b := g.rmNearestBurg(m.P); b != nil {
					n := 3 + int(h/7%9)
					x := rmBeastDeeds[int(h/13)%len(rmBeastDeeds)]
					g.rmAdd("beast", fmt.Sprintf("bs:%d:%d", m.ID, day/10), b.ID, rmRum{A: m.Name, Place: g.wsPlace(m.P), P: m.P, N: n, X: x})
				}
			}
		}
	}
	// feast days near the wanderer
	if w.In(g.WX, g.WY) {
		pp := w.Idx(g.WX, g.WY)
		bl := append([]*BurgData(nil), g.ws().burgList(g)...)
		sort.SliceStable(bl, func(i, j int) bool {
			a, b := wsCheb(w, pp, bl[i].P), wsCheb(w, pp, bl[j].P)
			if a != b {
				return a < b
			}
			return bl[i].ID < bl[j].ID
		})
		for i := 0; i < len(bl) && i < 4; i++ {
			if f := w.FeastOn(bl[i], g.Day()); f != nil {
				g.rmAdd("feast", fmt.Sprintf("fs:%d:%s:%d", bl[i].ID, f.Name, day/YearDays), bl[i].ID, rmRum{A: bl[i].Name, X: f.Name, Place: bl[i].Name, P: bl[i].P})
			}
		}
	}
	// let the old ones go
	keep := st.Rums[:0]
	for _, r := range st.Rums {
		if day-r.Day <= r.Life+6 {
			keep = append(keep, r)
		}
	}
	st.Rums = keep
	if len(st.Asked) > 0 {
		suf := ":" + strconv.Itoa(day)
		for k := range st.Asked {
			if !strings.HasSuffix(k, suf) {
				delete(st.Asked, k)
			}
		}
	}
}

// ---------------------------------------------------------------- screen

func rmFeel(h *rmHeard) (word, why string) {
	switch {
	case h.Hops == 0:
		why = "told where it happened"
	case h.Hops <= 2:
		why = "a few retellings old"
	default:
		why = "passed along many times"
	}
	if h.Corr > 0 {
		why += fmt.Sprintf("; the same story in %d other town(s)", h.Corr)
	}
	if h.Diff > 0 {
		why += "; other towns tell it differently"
	}
	switch {
	case h.Hops >= 3 || h.Diff > 0:
		word = "shaky"
	case h.Hops <= 1 || h.Corr > 0:
		word = "firm"
	default:
		word = "fair"
	}
	return
}

func (g *Game) rmRows() []*rmHeard {
	st := g.rm()
	out := append([]*rmHeard(nil), st.Heard...)
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Day != out[j].Day {
			return out[i].Day > out[j].Day
		}
		return out[i].RID > out[j].RID
	})
	return out
}

func rmKeys(g *Game, k string) {
	st := g.rm()
	rows := g.rmRows()
	switch k {
	case "Escape", "q":
		g.Mode = "main"
	case "ArrowUp", "N8", "k":
		st.sel = maxi(0, st.sel-1)
	case "ArrowDown", "N2", "j":
		st.sel = mini(maxi(0, len(rows)-1), st.sel+1)
	case "PageDown":
		st.sel = mini(maxi(0, len(rows)-1), st.sel+8)
	case "PageUp":
		st.sel = maxi(0, st.sel-8)
	case "t":
		if st.sel >= len(rows) {
			return
		}
		h := rows[st.sel]
		if h.P < 0 || h.Hops >= 3 {
			g.Msg("You do not know where to look for this one.")
			return
		}
		x, y := h.P%g.W.W, h.P/g.W.W
		g.ceSetQuestTarget(x, y)
		g.Msg(fmt.Sprintf("{{c|The quest arrow now points to (%d,%d), where this is said to have happened.}}", x, y))
	}
}

func (g *Game) rmDraw() {
	sc := g.sc
	st := g.rm()
	x, y, w, h := g.panel("Rumours")
	bottom := 2 + 1 + h - 1
	sc.TextB(x, y, "{{K|Up/Down choose  t point the quest arrow at the place  Esc close}}", CGray, CBgDark, x+w)
	y += 2
	rows := g.rmRows()
	if len(rows) == 0 {
		sc.TextB(x, y, "You have heard nothing worth keeping. Ask townsfolk about the rumours on the road.", CGray, CBgDark, x+w)
		return
	}
	if st.sel >= len(rows) {
		st.sel = len(rows) - 1
	}
	top := maxi(0, st.sel-6)
	for i := top; i < len(rows) && y < bottom-11; i++ {
		r := rows[i]
		mark := "  "
		if i == st.sel {
			mark = "> "
		}
		tag := rmTpls[r.Kind].Tag
		state := "{{K|unchecked}}"
		switch r.Ver {
		case 1:
			state = "{{G|true}}"
		case -1:
			state = "{{R|false}}"
		}
		sc.TextB(x, y, fmt.Sprintf("%s{{W|%-8s}} %s  %s", mark, tag, rmCut(r.Text, w-30), state), CGray, CBgDark, x+w)
		y++
	}
	y++
	r := rows[st.sel]
	lines := []string{r.Text}
	word, why := rmFeel(r)
	age := lvDay(g) - r.SrcDay
	lines = append(lines, "")
	lines = append(lines, fmt.Sprintf("Started in {{Y|%s}}, about %d days before you heard it in %s (day %d).", g.rmTownName(r.SrcBurg), maxi(0, r.Day-r.SrcDay), g.rmTownName(r.Town), r.Day+1))
	lines = append(lines, fmt.Sprintf("Feels {{W|%s}}: %s. It happened about %d days ago.", word, why, maxi(0, age)))
	switch {
	case r.Ver == 1:
		lines = append(lines, "{{G|Verified: it is true.}} "+r.Real)
	case r.Ver == -1:
		lines = append(lines, "{{R|Verified: it was not true as told.}} The truth: "+r.Real)
	case r.P < 0:
		lines = append(lines, "{{K|Nothing on the road could settle this one.}}")
	case r.Hops >= 3:
		lines = append(lines, "{{K|The place was lost in the retelling. Ask nearer to where it began.}}")
	default:
		lines = append(lines, fmt.Sprintf("{{K|Walk to (%d,%d) to check it. A checked rumour pays a little and goes into your case notes.}}", r.P%g.W.W, r.P/g.W.W))
	}
	for _, d := range lines {
		for _, ln := range Wrap(d, w) {
			if y < bottom {
				sc.TextB(x, y, ln, CGray, CBgDark, x+w)
				y++
			}
		}
	}
	if y < bottom-1 {
		sc.TextB(x, bottom-1, fmt.Sprintf("{{K|%d heard, %d checked}}", len(rows), st.Vers), CGray, CBgDark, x+w)
	}
}
