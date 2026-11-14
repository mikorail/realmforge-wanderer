package engine

import (
	"fmt"
	"sort"
	"strings"
)

// Deadlines and rival races (prefix dl). The world keeps running whether or not the wanderer is
// there: up to three time-sensitive events per planet are active at once, generated from the world
// seed (side rng NewRng(seed:dl:n), never g.R at generation): a plague spreading town to town
// unless a cure is delivered, a siege with a relief window, a hostage with an execution date, a
// caravan about to leave, a festival, a flood closing a pass, a vault that opens for a few days under an
// omen (eclipse, datastorm, ...) and a lost relic. The vault and the relic are raced by a named rival
// company (leaders come from rival.go when one is alive): signs (rumours, tracks, a cleared hall) reveal
// them, and the player may sabotage them or ally with them. Every event has a countdown (sidebar,
// Deadlines screen mode "dl", ability dl_view, journal lines via dlLines), warnings at stages, a
// reward (gold, rep, XP, an early bonus) when the clock is beaten and a world consequence on lapse
// (news, the dead of lvSt, reputation). State g.ext("dl"), planet-local ("dl" in unLocalKeys).

type dlEv struct {
	ID      int    `json:"id"`
	Kind    string `json:"k"`
	Burg    int    `json:"b"`
	Src     int    `json:"s,omitempty"`
	Next    int    `json:"nx,omitempty"`
	Site    int    `json:"p"`
	Start   int    `json:"st"`
	Open    int    `json:"op,omitempty"`
	Due     int    `json:"du"`
	RDue    int    `json:"rd,omitempty"`
	Stage   int    `json:"sg"`
	Sign    int    `json:"sn,omitempty"`
	State   string `json:"ss"` // active won lost shared
	Ended   int    `json:"en,omitempty"`
	Known   bool   `json:"kn,omitempty"`
	RSeen   bool   `json:"rs,omitempty"`
	Person  string `json:"pe,omitempty"`
	PKey    string `json:"pk,omitempty"`
	Group   string `json:"gr,omitempty"`
	Leader  string `json:"ld,omitempty"`
	Cost    int    `json:"co,omitempty"`
	Reward  int    `json:"rw,omitempty"`
	Sab     int    `json:"sb,omitempty"`
	Ally    bool   `json:"al,omitempty"`
	Omen    string `json:"om,omitempty"`
	Caravan string `json:"cv,omitempty"`
	Result  string `json:"rt,omitempty"`
}

type dlState struct {
	Next   int      `json:"n"`
	NextAt int      `json:"na"`
	Evs    []*dlEv  `json:"e,omitempty"`
	Log    []string `json:"l,omitempty"`
	Sel    int      `json:"-"`
}

func init() {
	unLocalKeys = append(unLocalKeys, "dl")
	turnHooks = append(turnHooks, dlTurn)
	enterHooks = append(enterHooks, dlEnter)
	talkHooks = append(talkHooks, dlTalk)
	sideHooks = append(sideHooks, dlSide)
	modeKeys["dl"] = dlKeys
	modeDraws["dl"] = func(g *Game) { g.dlDraw() }
}

func (g *Game) dl() *dlState {
	return g.ext("dl", func() any { return &dlState{} }).(*dlState)
}

func dlKindOf(id string) *dlKindDef {
	for i := range dlKinds {
		if dlKinds[i].ID == id {
			return &dlKinds[i]
		}
	}
	return nil
}

func (e *dlEv) race() bool { return e.Kind == "vault" || e.Kind == "relic" }
func (e *dlEv) live() bool { return e.State == "active" }

func dlf(f string, args ...any) string {
	n := strings.Count(f, "%s")
	for len(args) < n {
		args = append(args, "")
	}
	return fmt.Sprintf(f, args[:n]...)
}

func (g *Game) dlBurgName(id int) string {
	if b := g.W.burgByID[id]; b != nil {
		return b.Name
	}
	return "a distant town"
}

func (g *Game) dlStateFac(burg int) string {
	if b := g.W.burgByID[burg]; b != nil && b.State > 0 {
		return fmt.Sprintf("state:%d", b.State)
	}
	return ""
}

func (g *Game) dlRep(e *dlEv, n int, why string) {
	if f := g.dlStateFac(e.Burg); f != "" {
		g.AddRep(f, n, why)
	}
}

func (g *Game) dlNews(burg int, format string, a ...any) {
	lvSt(g).addNews(lvDay(g), burg, format, a...)
}

func (g *Game) dlLog(s string) {
	st := g.dl()
	st.Log = append(st.Log, fmt.Sprintf("Day %d: %s", DayOf(g.Turn)+1, s))
	if len(st.Log) > 14 {
		st.Log = st.Log[len(st.Log)-14:]
	}
}

// dlClock renders minutes as "5d 3h", "7h 20m", "40m".
func dlClock(m int) string {
	if m < 0 {
		m = 0
	}
	d, h, mi := m/DayLen, (m%DayLen)/60, m%60
	switch {
	case d > 0:
		return fmt.Sprintf("%dd %dh", d, h)
	case h > 0:
		return fmt.Sprintf("%dh %02dm", h, mi)
	}
	return fmt.Sprintf("%dm", mi)
}

// dlEnd is the turn the event is lost at.
func (e *dlEv) end() int {
	if e.race() && e.RDue > 0 && !e.Ally {
		return e.RDue
	}
	return e.Due
}

// dlCount returns the label and minutes left of the clock shown to the player.
func (g *Game) dlCount(e *dlEv) (string, int) {
	switch {
	case e.Open > g.Turn:
		if e.Kind == "festival" {
			return "feast in", e.Open - g.Turn
		}
		return "opens in", e.Open - g.Turn
	case e.race() && e.RSeen && e.RDue > 0 && !e.Ally:
		return "rivals arrive in", e.RDue - g.Turn
	case e.Kind == "festival":
		return "feast ends in", e.Due - g.Turn
	case e.Kind == "vault":
		return "closes in", e.Due - g.Turn
	case e.Kind == "caravan":
		return "leaves in", e.Due - g.Turn
	case e.Kind == "hostage":
		return "execution in", e.Due - g.Turn
	}
	return "left", e.Due - g.Turn
}

func (g *Game) dlTitle(e *dlEv) string {
	k := dlKindOf(e.Kind)
	if k == nil {
		return e.Kind
	}
	if e.Kind == "hostage" {
		return dlf(k.Title, g.dlBurgName(e.Burg))
	}
	if e.Kind == "vault" && e.Omen != "" {
		return "Vault opens near " + g.dlBurgName(e.Burg) + " under " + e.Omen
	}
	return dlf(k.Title, g.dlBurgName(e.Burg))
}

func (g *Game) dlGoal(e *dlEv) string {
	k := dlKindOf(e.Kind)
	t := g.dlBurgName(e.Burg)
	switch e.Kind {
	case "plague", "siege":
		return dlf(k.Goalt, g.dlBurgName(e.Src), t)
	case "hostage":
		return dlf(k.Goalt, e.Person, t)
	case "caravan":
		return dlf(k.Goalt, t, e.Caravan)
	}
	return dlf(k.Goalt, t)
}

func (g *Game) dlSiteName(e *dlEv) string {
	w := g.W
	return fmt.Sprintf("%s (%d,%d)", g.wsPlace(e.Site), e.Site%w.W, e.Site/w.W)
}

// ---- generation ----

func (g *Game) dlActive() []*dlEv {
	var out []*dlEv
	for _, e := range g.dl().Evs {
		if e.live() {
			out = append(out, e)
		}
	}
	return out
}

func (g *Game) dlGenerate() {
	st := g.dl()
	w := g.W
	if g.Cur == nil || !w.In(g.Cur.WX, g.Cur.WY) {
		return
	}
	bl := append([]*BurgData(nil), g.ws().burgList(g)...)
	if len(bl) < 3 {
		return
	}
	pp := w.Idx(g.Cur.WX, g.Cur.WY)
	sort.SliceStable(bl, func(i, j int) bool {
		a, b := wsCheb(w, pp, bl[i].P), wsCheb(w, pp, bl[j].P)
		if a != b {
			return a < b
		}
		return bl[i].ID < bl[j].ID
	})
	r := NewRng(fmt.Sprint(g.Seed, ":dl:", st.Next))
	st.Next++
	used := map[string]bool{}
	usedB := map[int]bool{}
	for _, e := range g.dlActive() {
		used[e.Kind] = true
		usedB[e.Burg] = true
	}
	// kind by weight, not one already running
	tot := 0
	for _, k := range dlKinds {
		if !used[k.ID] {
			tot += k.Weight
		}
	}
	if tot == 0 {
		return
	}
	roll := r.Int(0, tot-1)
	var kd *dlKindDef
	for i := range dlKinds {
		k := &dlKinds[i]
		if used[k.ID] {
			continue
		}
		if roll < k.Weight {
			kd = k
			break
		}
		roll -= k.Weight
	}
	if kd == nil {
		return
	}
	// target town: near the wanderer
	pool := bl[:mini(10, len(bl))]
	var cands []*BurgData
	for _, b := range pool {
		if !usedB[b.ID] && wsCheb(w, pp, b.P) <= 7 {
			cands = append(cands, b)
		}
	}
	if len(cands) == 0 {
		for _, b := range pool {
			if !usedB[b.ID] {
				cands = append(cands, b)
			}
		}
	}
	if len(cands) == 0 {
		return
	}
	tgt := cands[r.Int(0, len(cands)-1)]
	dpp := maxf(0.6, g.tvKm()/28)
	days := func(d int) float64 { return float64(d) * dpp }
	e := &dlEv{ID: st.Next, Kind: kd.ID, Burg: tgt.ID, Site: tgt.P, Start: g.Turn, State: "active", Known: true}
	st.Next++
	lvl := g.P.Level
	base := map[string]int{"plague": 160, "siege": 200, "hostage": 120, "caravan": 40, "festival": 50, "flood": 90, "vault": 260, "relic": 220}[kd.ID]
	e.Reward = base + base*lvl/8 + r.Int(0, base/3)
	near := func(of *BurgData, skip ...int) []*BurgData {
		o := append([]*BurgData(nil), bl...)
		sort.SliceStable(o, func(i, j int) bool {
			a, b := wsCheb(w, of.P, o[i].P), wsCheb(w, of.P, o[j].P)
			if a != b {
				return a < b
			}
			return o[i].ID < o[j].ID
		})
		var out []*BurgData
	next:
		for _, b := range o {
			if b.ID == of.ID {
				continue
			}
			for _, s := range skip {
				if s == b.ID {
					continue next
				}
			}
			out = append(out, b)
			if len(out) >= 4 {
				break
			}
		}
		return out
	}
	window := func(total float64) int {
		return int((total*1.35 + float64(kd.Min) + float64(r.Int(0, 3))) * DayLen)
	}
	switch kd.ID {
	case "plague", "siege":
		nb := near(tgt)
		if len(nb) == 0 {
			return
		}
		src := nb[r.Int(0, mini(2, len(nb)-1))]
		e.Src = src.ID
		if kd.ID == "plague" {
			for _, b := range nb {
				if b.ID != src.ID {
					e.Next = b.ID
					break
				}
			}
		}
		e.Due = g.Turn + window(days(wsCheb(w, pp, src.P)+wsCheb(w, src.P, tgt.P)))
	case "hostage":
		e.Due = g.Turn + window(days(wsCheb(w, pp, tgt.P)))
		e.Cost = 70 + r.Int(0, 100) + lvl*18
		e.Person, e.PKey = g.dlPickHostage(tgt, r)
	case "caravan":
		e.Due = g.Turn + window(days(wsCheb(w, pp, tgt.P)))*3/4
		e.Cost = 30 + lvl*6
		e.Caravan = Pick(r, dlCaravanNames)
	case "festival":
		e.Open = g.Turn + window(days(wsCheb(w, pp, tgt.P)))
		e.Due = e.Open + DayLen
	case "flood", "vault", "relic":
		site := g.dlPickSite(tgt, r)
		e.Site = site
		d := wsCheb(w, pp, site)
		eta := days(d)*1.15 + 2
		switch kd.ID {
		case "flood":
			e.Due = g.Turn + window(days(d))
		case "vault", "relic":
			far := near(tgt)
			from := tgt
			if len(far) > 0 {
				from = far[len(far)-1]
			}
			e.Src = from.ID
			e.Group, e.Leader = g.dlRivalName(r)
			if kd.ID == "vault" {
				e.Omen = dlOmens[g.Genre]
				if e.Omen == "" {
					e.Omen = "a black eclipse"
				}
				e.Open = g.Turn + int((eta*0.8+1+r.Float()*2)*DayLen)
				e.RDue = e.Open + int((0.2+r.Float()*3)*DayLen)
				e.Due = e.Open + 4*DayLen
			} else {
				e.RDue = g.Turn + int((eta+r.Float()*3.5-1)*DayLen)
				if e.RDue < g.Turn+2*DayLen {
					e.RDue = g.Turn + 2*DayLen
				}
				e.Due = e.RDue
				e.Known = false
			}
		}
	}
	st.Evs = append(st.Evs, e)
	st.NextAt = g.Turn + int((1.5+r.Float()*2.5)*DayLen)
	if len(st.Evs) > 16 {
		st.Evs = st.Evs[len(st.Evs)-16:]
	}
	if e.Known {
		_, left := g.dlCount(e)
		g.Msg("{{Y|Word reaches you:}} " + g.dlTitle(e) + " ({{R|" + dlClock(left) + "}}). See the Deadlines screen.")
		if !e.race() {
			g.dlNews(e.Burg, "%s", g.dlTitle(e)+".")
		}
	}
	g.dlLog("Word of: " + g.dlTitle(e) + ".")
}

func (g *Game) dlPickSite(tgt *BurgData, r *Rng) int {
	w := g.W
	tx, ty := tgt.P%w.W, tgt.P/w.W
	var c []int
	for dy := -3; dy <= 3; dy++ {
		for dx := -3; dx <= 3; dx++ {
			x, y := tx+dx, ty+dy
			if !w.In(x, y) || maxi(abs(dx), abs(dy)) < 2 {
				continue
			}
			p := w.Idx(x, y)
			if w.IsWater(p) {
				continue
			}
			c = append(c, p)
		}
	}
	if len(c) == 0 {
		return tgt.P
	}
	return c[r.Int(0, len(c)-1)]
}

func (g *Game) dlPickHostage(tgt *BurgData, r *Rng) (string, string) {
	st := lvSt(g)
	var keys []string
	for k, p := range st.People {
		if p != nil && p.Alive && p.Home == tgt.ID && p.Burg == tgt.ID {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	if len(keys) > 0 {
		k := keys[r.Int(0, len(keys)-1)]
		return st.People[k].Name, k
	}
	return Pick(r, dlLeaderFirst) + " " + Pick(r, dlLeaderLast), ""
}

func (g *Game) dlRivalName(r *Rng) (string, string) {
	grp := "the " + Pick(r, dlGroupAdj) + " " + Pick(r, dlGroupNoun)
	lead := Pick(r, dlLeaderFirst) + " " + Pick(r, dlLeaderLast)
	for _, rv := range g.rv().R {
		if rv != nil && rv.Alive {
			lead = g.rvName(rv)
			break
		}
	}
	return grp, lead
}

// ---- outcomes ----

func (g *Game) dlEndEvent(e *dlEv, state, text string) {
	e.State, e.Ended, e.Result = state, g.Turn, text
	g.dlLog(text)
	for _, it := range append([]*Item(nil), g.P.Inv...) {
		if it.Extra["dl"] == e.ID && it.Kind == "trophy" {
			g.removeInv(it)
		}
	}
}

func (g *Game) dlKillFolk(burg, n int, cause string) int {
	st := lvSt(g)
	var keys []string
	for k, p := range st.People {
		if p != nil && p.Alive && p.Home == burg {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	h := HashStr(fmt.Sprint(g.Seed, ":dlk:", burg, ":", g.Turn/DayLen))
	n = mini(n, len(keys))
	for i := 0; i < n; i++ {
		j := int((h + uint64(i)*7919) % uint64(len(keys)))
		p := st.People[keys[j]]
		keys = append(keys[:j], keys[j+1:]...)
		p.Alive = false
		p.Died = lvYear(g)
		p.Cause = cause
		p.addLife(lvYear(g), "death", "Died: "+cause+".")
	}
	return n
}

// dlWin pays out; early (more than half the window unspent) earns a bonus.
func (g *Game) dlWin(e *dlEv, extra string) {
	k := dlKindOf(e.Kind)
	town := g.dlBurgName(e.Burg)
	subj := town
	if e.Kind == "hostage" {
		subj = e.Person
	}
	span := maxi(1, e.end()-e.Start)
	early := float64(e.end()-g.Turn)/float64(span) > 0.5
	gold := e.Reward
	if early {
		gold = gold * 3 / 2
	}
	rep, xp := 6, 40+g.P.Level*10
	if early {
		rep += 3
		xp = xp * 3 / 2
	}
	line := dlf(k.Won, subj)
	g.dlEndEvent(e, "won", line)
	msg := "{{G|" + line + "}}"
	if early {
		msg += " {{Y|You beat the clock by a wide margin: extra reward.}}"
	}
	g.Msg(msg)
	if gold > 0 {
		g.Msg("Reward: " + g.cuEarn(gold) + ".")
	}
	if extra != "" {
		g.Msg(extra)
	}
	g.dlRep(e, rep, " for beating the clock")
	g.gainXP(xp)
	g.deed("deadline", fmt.Sprintf("%s beat the clock: %s", g.P.Name, line), 0, 0)
	g.dlNews(e.Burg, "%s", Plain(line)+" Folk credit the wanderer "+g.P.Name+".")
	if e.race() && e.Group != "" {
		g.dlNews(e.Burg, "%s of %s were beaten to the prize.", strings.ToUpper(e.Group[:1])+e.Group[1:], g.dlBurgName(e.Src))
	}
}

func (g *Game) dlLose(e *dlEv, why string) {
	k := dlKindOf(e.Kind)
	town := g.dlBurgName(e.Burg)
	subj := town
	if e.Kind == "hostage" {
		subj = e.Person
	}
	line := dlf(k.Lost, subj)
	g.dlEndEvent(e, "lost", line)
	if e.Known {
		g.Msg("{{R|" + line + "}}")
	}
	rep := -3
	switch e.Kind {
	case "plague":
		n := g.dlKillFolk(e.Burg, 5, "the plague")
		if e.Next > 0 {
			n += g.dlKillFolk(e.Next, 2, "the plague")
		}
		g.dlNews(e.Burg, "The plague has killed %d in %s and is passing down the road.", maxi(n, 12), town)
	case "siege":
		n := g.dlKillFolk(e.Burg, 4, "the fall of the town")
		g.dlNews(e.Burg, "%s fell to the besiegers; %d of its people are dead.", town, maxi(n, 9))
	case "hostage":
		if e.PKey != "" {
			if p := lvGet(g, e.PKey); p != nil && p.Alive {
				p.Alive, p.Died, p.Cause = false, lvYear(g), "executed as a hostage"
				p.addLife(lvYear(g), "death", "Executed as a hostage.")
			}
		}
		g.dlNews(e.Burg, "%s was executed at %s.", e.Person, town)
	case "caravan":
		rep = 0
		g.dlNews(e.Burg, "The %s has left %s.", e.Caravan, town)
	case "festival":
		rep = -1
		g.dlNews(e.Burg, "The feast of %s was held without the wanderer.", town)
	case "flood":
		g.dlNews(e.Burg, "The pass near %s is closed by the flood.", town)
	default:
		rep = -2
		g.dlNews(e.Burg, "%s", Plain(line)+" "+strings.ToUpper(e.Group[:1])+e.Group[1:]+", led by "+e.Leader+", is the talk of "+town+".")
		if why != "" {
			g.dlLog(why)
		}
	}
	if e.Known && rep != 0 {
		g.dlRep(e, rep, " (you were not there)")
	}
	if e.Known {
		g.deed("deadline", fmt.Sprintf("%s let the clock run out: %s", g.P.Name, line), 0, 0)
	}
}

// ---- turn / enter ----

func dlTurn(g *Game) {
	if g.W == nil || g.P == nil || g.Cur == nil || g.Turn%30 != 0 {
		return
	}
	st := g.dl()
	if st.NextAt == 0 {
		st.NextAt = g.Turn + 180
	}
	for _, e := range st.Evs {
		if !e.live() {
			continue
		}
		g.dlStep(e)
	}
	if g.Turn >= st.NextAt && len(g.dlActive()) < 3 {
		g.dlGenerate()
		if g.Turn >= st.NextAt { // nothing could be made: try later
			st.NextAt = g.Turn + DayLen/2
		}
	}
}

func (g *Game) dlStep(e *dlEv) {
	k := dlKindOf(e.Kind)
	if k == nil {
		e.State = "lost"
		return
	}
	town := g.dlBurgName(e.Burg)
	subj := town
	if e.Kind == "hostage" {
		subj = e.Person
	}
	// presence-based goals
	if e.Kind == "festival" && g.Turn >= e.Open && g.Cur.Kind == "town" && g.Cur.Burg == e.Burg {
		g.dlWin(e, "")
		return
	}
	// stage warnings
	span := maxi(1, e.end()-e.Start)
	f := float64(g.Turn-e.Start) / float64(span)
	th := [3]float64{0.5, 0.75, 0.92}
	if e.Stage < 3 && f >= th[e.Stage] && g.Turn < e.end() && e.Known {
		t := dlf(k.Stage[e.Stage], subj)
		g.Msg("{{Y|Deadline:}} " + t + " ({{R|" + dlClock(e.end()-g.Turn) + "}} left)")
		g.dlNews(e.Burg, "%s", t)
		if e.Kind == "plague" && e.Stage == 0 && e.Next > 0 {
			g.dlNews(e.Next, "The sickness from %s has reached %s.", town, g.dlBurgName(e.Next))
		}
		e.Stage++
	}
	if e.race() {
		g.dlSigns(e, f)
	}
	if e.race() && e.Ally && e.RDue > 0 && g.Turn >= e.RDue {
		g.dlShared(e)
		return
	}
	if g.Turn >= e.end() {
		g.dlLose(e, "")
	}
}

// dlShared: the allied company reached the prize first and keeps its word: a share.
func (g *Game) dlShared(e *dlEv) {
	gold := e.Reward * 3 / 10
	line := fmt.Sprintf("%s reached the prize near %s first, but kept their word and sent a share.", strings.ToUpper(e.Group[:1])+e.Group[1:], g.dlBurgName(e.Burg))
	g.dlEndEvent(e, "shared", line)
	g.Msg("{{Y|" + line + "}}")
	if gold > 0 {
		g.Msg("Share: " + g.cuEarn(gold) + ".")
	}
	g.gainXP(20 + g.P.Level*4)
	g.dlRep(e, 2, " for the shared prize")
}

// dlSigns reveals a rival race step by step: rumour, tracks, cleared hall.
func (g *Game) dlSigns(e *dlEv, f float64) {
	town := g.dlBurgName(e.Burg)
	src := g.dlBurgName(e.Src)
	grp := e.Group
	if e.Sign < 1 && f >= 0.25 {
		e.Sign = 1
		g.dlNews(e.Src, "%s, led by %s, bought rope, lamps and digging tools in %s and asked the way to %s.", strings.ToUpper(grp[:1])+grp[1:], e.Leader, src, town)
	}
	if e.Sign < 2 && f >= 0.55 {
		e.Sign = 2
		g.dlNews(e.Burg, "Riders of %s were seen on the roads near %s.", grp, town)
	}
	if e.Sign < 3 && f >= 0.85 {
		e.Sign = 3
		g.dlNews(e.Burg, "The wardens of the old hall near %s were found cut down; someone has cleared the way in.", town)
		if !e.RSeen {
			e.RSeen, e.Known = true, true
			g.Msg("{{Y|Word on the road:}} the old hall near " + town + " has been cleared out by " + grp + ". " + g.dlTitle(e) + ".")
		}
	}
}

func dlEnter(g *Game, zn *Zone) {
	if g.W == nil || g.P == nil || zn == nil || !g.W.In(zn.WX, zn.WY) {
		return
	}
	pz := g.W.Idx(zn.WX, zn.WY)
	for _, e := range g.dlActive() {
		if e.Kind == "plague" || e.Kind == "siege" || e.Kind == "hostage" || e.Kind == "caravan" || e.Kind == "festival" {
			continue
		}
		d := wsCheb(g.W, pz, e.Site)
		if e.race() && !e.RSeen && d <= 3 && d > 0 && e.Sign >= 2 {
			e.RSeen, e.Known = true, true
			g.Msg("{{Y|Fresh tracks of many horses and boots lead toward " + g.wsPlace(e.Site) + ".}} " + g.dlTitle(e))
			g.dlLog("Found the tracks of " + e.Group + ".")
			continue
		}
		if d == 0 && zn.Z == 0 && zn.Kind != "town" {
			g.dlClaim(e)
		}
	}
}

func (g *Game) dlClaim(e *dlEv) {
	if !e.live() {
		return
	}
	if e.Open > g.Turn {
		if e.Kind == "vault" {
			g.Msg("{{Y|The vault doors here are sealed. They will open in " + dlClock(e.Open-g.Turn) + ".}}")
		}
		return
	}
	if g.Turn >= e.Due {
		return
	}
	if e.race() && e.RDue > 0 && g.Turn >= e.RDue {
		if e.Ally {
			g.dlShared(e)
		} else {
			g.dlLose(e, "You reached the place too late.")
			g.Msg("{{R|Trampled earth, cold ashes and an empty vault: " + e.Group + " were here first.}}")
		}
		return
	}
	r := NewRng(fmt.Sprint(g.Seed, ":dlc:", e.ID))
	var bonus []string
	give := func(it *Item) {
		if it != nil {
			g.give(it)
			bonus = append(bonus, it.Display())
		}
	}
	switch e.Kind {
	case "flood":
		give(g.Loot(r, 3))
		give(g.Loot(r, 2))
	case "vault":
		give(g.Loot(r, 5))
		give(g.Loot(r, 4))
	case "relic":
		if arts := g.W.Artifacts; len(arts) > 0 {
			give(g.RelicItem(&arts[r.Int(0, len(arts)-1)]))
		} else {
			give(g.Loot(r, 5))
		}
	}
	extra := ""
	if len(bonus) > 0 {
		extra = "You carry out: " + strings.Join(bonus, ", ") + "."
	}
	g.dlWin(e, extra)
}

// ---- talk ----

func (g *Game) dlPack(e *dlEv) *Item {
	for _, it := range g.P.Inv {
		if it.Kind == "trophy" && it.Extra != nil && it.Extra["dl"] == e.ID {
			return it
		}
	}
	return nil
}

func dlTalk(g *Game, a *Actor) []Option {
	if g.W == nil || g.Cur == nil || g.P == nil || a == nil || a.HP <= 0 || a.Angry || a.Has("rvr") {
		return nil
	}
	var out []Option
	inTown := g.Cur.Kind == "town"
	for _, e := range g.dlActive() {
		e := e
		k := dlKindOf(e.Kind)
		town := g.dlBurgName(e.Burg)
		switch e.Kind {
		case "plague", "siege":
			if inTown && g.Cur.Burg == e.Src && g.dlPack(e) == nil {
				out = append(out, Option{"Take the " + k.Item + " meant for " + town, func() {
					it := &Item{ID: g.newID(), Name: k.Item + " for " + town, Kind: "trophy", Glyph: '!', F: CGreen, Weight: 1,
						Extra: map[string]int{"dl": e.ID}, Lore: "Bound for " + town + " before the deadline."}
					g.give(it)
					_, left := g.dlCount(e)
					g.Msg("You take the " + k.Item + ". " + town + " is waiting: {{R|" + dlClock(left) + "}} left.")
					g.Mode = "main"
				}})
			}
			if inTown && g.Cur.Burg == e.Burg {
				if it := g.dlPack(e); it != nil {
					out = append(out, Option{"Hand over the " + k.Item + " to the town", func() {
						g.removeInv(it)
						g.dlWin(e, "")
						g.Mode = "main"
					}})
				}
			}
		case "hostage":
			if inTown && g.Cur.Burg == e.Burg {
				out = append(out, Option{fmt.Sprintf("Pay the ransom for %s (%s)", e.Person, g.cuFormat(e.Cost)), func() {
					if !g.cuSpend(e.Cost) {
						g.Msg("You cannot raise the ransom of " + g.cuFormat(e.Cost) + ".")
						return
					}
					g.dlWin(e, "")
					g.Mode = "main"
				}})
			}
		case "caravan":
			if inTown && g.Cur.Burg == e.Burg {
				out = append(out, Option{fmt.Sprintf("Trade with the %s before it leaves (%s)", e.Caravan, g.cuFormat(e.Cost)), func() {
					if !g.cuSpend(e.Cost) {
						g.Msg("The caravan masters want " + g.cuFormat(e.Cost) + " up front.")
						return
					}
					r := NewRng(fmt.Sprint(g.Seed, ":dlb:", e.ID))
					var got []string
					for _, tier := range []int{3, 2} {
						if it := g.Loot(r, tier); it != nil {
							g.give(it)
							got = append(got, it.Display())
						}
					}
					g.dlWin(e, "You buy: "+strings.Join(got, ", ")+".")
					g.Mode = "main"
				}})
			}
		case "relic":
			if inTown && !e.Known && e.Sign >= 1 {
				out = append(out, Option{"Ask about the strangers buying digging tools", func() {
					e.Known, e.RSeen = true, true
					g.Msg("\"" + strings.ToUpper(e.Group[:1]) + e.Group[1:] + ", with " + e.Leader + " at the head. They are after a relic buried near " + town + ".\"")
					g.dlLog("Learned of " + e.Group + " and the relic near " + town + ".")
					g.Msg("{{Y|" + g.dlTitle(e) + ":}} {{R|" + dlClock(e.RDue-g.Turn) + "}} until they arrive.")
				}})
			}
		case "vault":
			if inTown && !e.RSeen && e.Sign >= 1 {
				out = append(out, Option{"Ask about the company bound for the vault", func() {
					e.RSeen = true
					g.Msg("\"" + strings.ToUpper(e.Group[:1]) + e.Group[1:] + ". " + e.Leader + " leads them, and they mean to be at the doors when they open.\"")
					g.dlLog("Learned of " + e.Group + " racing for the vault.")
				}})
			}
		}
	}
	if inTown && len(g.dl().Evs) > 0 {
		out = append(out, Option{"Ask about what is pressing in the land (deadlines)", func() {
			g.Mode = "dl"
			g.dl().Sel = 0
		}})
	}
	return out
}

// ---- sidebar ----

func dlSide(g *Game) []string {
	if g.W == nil || g.P == nil {
		return nil
	}
	act := g.dlActive()
	var vis []*dlEv
	for _, e := range act {
		if e.Known {
			vis = append(vis, e)
		}
	}
	sort.SliceStable(vis, func(i, j int) bool {
		_, a := g.dlCount(vis[i])
		_, b := g.dlCount(vis[j])
		return a < b
	})
	var out []string
	for i, e := range vis {
		if i >= 2 {
			break
		}
		_, left := g.dlCount(e)
		col := "G"
		switch {
		case left < DayLen:
			col = "R"
		case left < 3*DayLen:
			col = "Y"
		}
		out = append(out, fmt.Sprintf("{{%s|%s}} %s", col, dlShort(e.Kind), dlClock(left)))
	}
	return out
}

func dlShort(kind string) string {
	switch kind {
	case "plague":
		return "Plague"
	case "siege":
		return "Siege"
	case "hostage":
		return "Hostage"
	case "caravan":
		return "Caravan"
	case "festival":
		return "Feast"
	case "flood":
		return "Flood"
	case "vault":
		return "Vault"
	}
	return "Relic"
}

// ---- Deadlines view ----

func (g *Game) dlAbilities() []Ability {
	if g.P == nil || g.W == nil {
		return nil
	}
	return []Ability{{ID: "dl_view", Name: "Deadlines and rival races", Target: "self"}}
}

func (g *Game) dlAbUse(id string) {
	if id == "dl_view" {
		g.dl().Sel = 0
		g.Mode = "dl"
	}
}

func (g *Game) dlList() []*dlEv {
	var act, done []*dlEv
	for i := len(g.dl().Evs) - 1; i >= 0; i-- {
		e := g.dl().Evs[i]
		switch {
		case e.live() && e.Known:
			act = append(act, e)
		case !e.live() && e.Known:
			done = append(done, e)
		}
	}
	if len(done) > 6 {
		done = done[:6]
	}
	return append(act, done...)
}

func dlKeys(g *Game, k string) {
	st := g.dl()
	l := g.dlList()
	switch k {
	case "Escape", "q":
		g.Mode = "main"
	case "ArrowUp", "N8", "k":
		st.Sel = maxi(0, st.Sel-1)
	case "ArrowDown", "N2", "j":
		st.Sel = mini(maxi(0, len(l)-1), st.Sel+1)
	case "s", "a":
		if st.Sel >= len(l) {
			return
		}
		e := l[st.Sel]
		if !e.live() || !e.race() || !e.RSeen {
			g.Msg("There is no known rival on this one to deal with.")
			return
		}
		if e.Ally {
			g.Msg("You are already allied with " + e.Group + ".")
			return
		}
		if k == "s" {
			g.dlSabotage(e)
		} else {
			g.dlAlly(e)
		}
	}
}

func (g *Game) dlSabotage(e *dlEv) {
	if e.Sab >= 2 {
		g.Msg("Your hired hands will not try a third time.")
		return
	}
	cost := 60 + g.P.Level*10
	if !g.cuSpend(cost) {
		g.Msg("Saboteurs want " + g.cuFormat(cost) + ".")
		return
	}
	e.Sab++
	if g.R.Float() < 0.6 {
		e.RDue += 2 * DayLen
		g.Msg("{{G|Paid hands cut the rope bridges and foul the wells on " + e.Group + "'s road. They will be about two days late.}}")
		g.dlLog("Sabotaged " + e.Group + " (+2 days).")
	} else {
		e.RDue = maxi(g.Turn+60, e.RDue-360)
		g.Msg("{{R|The sabotage was found out. " + e.Group + " push harder, and " + g.dlBurgName(e.Burg) + " frowns on foul play.}}")
		g.dlRep(e, -2, " for foul play")
		g.dlLog("Sabotage against " + e.Group + " failed.")
	}
}

func (g *Game) dlAlly(e *dlEv) {
	cost := 100 + g.P.Level*15
	if !g.cuSpend(cost) {
		g.Msg("A pact with " + e.Leader + " costs " + g.cuFormat(cost) + " in good faith.")
		return
	}
	if g.R.Float() < 0.7 {
		e.Ally = true
		g.Msg("{{G|" + e.Leader + " clasps your wrist. " + strings.ToUpper(e.Group[:1]) + e.Group[1:] + " will share the prize if they reach it first, and will not trouble you if you do.}}")
		g.dlLog("Allied with " + e.Group + ".")
	} else {
		g.Msg("{{R|" + e.Leader + " pockets the coin and laughs. No pact.}}")
	}
}

func (g *Game) dlDetail(e *dlEv) []string {
	var out []string
	out = append(out, "{{W|"+g.dlTitle(e)+"}}")
	out = append(out, g.dlGoal(e))
	if e.live() {
		lab, left := g.dlCount(e)
		out = append(out, fmt.Sprintf("{{R|%s %s}}", lab, dlClock(left)))
		if e.Kind == "plague" && e.Next > 0 {
			out = append(out, "The sickness threatens "+g.dlBurgName(e.Next)+" next.")
		}
		if e.Kind == "hostage" {
			out = append(out, "Ransom: "+g.cuFormat(e.Cost)+".")
		}
		if e.Kind == "caravan" {
			out = append(out, "Goods: "+g.cuFormat(e.Cost)+" up front.")
		}
		if e.race() {
			out = append(out, "Site: "+g.dlSiteName(e)+".")
			if e.RSeen {
				out = append(out, fmt.Sprintf("Rival: %s, led by %s, out of %s.", e.Group, e.Leader, g.dlBurgName(e.Src)))
				if e.Ally {
					out = append(out, "{{G|Allied: they will share the prize.}}")
				} else {
					out = append(out, fmt.Sprintf("{{K|s sabotage (%d tries left)   a ally with them}}", 2-e.Sab))
				}
			} else {
				out = append(out, "{{K|You have seen no sign of rivals yet. Listen for rumours, look for tracks.}}")
			}
		}
		out = append(out, fmt.Sprintf("Reward: about %s, more if you are early.", g.cuFormat(e.Reward)))
	} else {
		out = append(out, "{{"+map[string]string{"won": "G", "lost": "R", "shared": "Y"}[e.State]+"|"+strings.ToUpper(e.State)+"}} "+e.Result)
	}
	return out
}

func (g *Game) dlDraw() {
	sc := g.sc
	st := g.dl()
	x, y, w, h := g.panel("Deadlines")
	sc.TextB(x, y, "{{K|Up/Down choose  s sabotage  a ally  Esc close}}", CGray, CBgDark, x+w)
	y += 2
	l := g.dlList()
	if len(l) == 0 {
		sc.TextB(x, y, "Nothing presses on the land right now. Word will come.", CGray, CBgDark, x+w)
		return
	}
	if st.Sel >= len(l) {
		st.Sel = len(l) - 1
	}
	bottom := 2 + 1 + h - 1
	for i, e := range l {
		if y >= bottom-9 {
			break
		}
		mark := "  "
		if i == st.Sel {
			mark = "> "
		}
		var tail string
		if e.live() {
			lab, left := g.dlCount(e)
			col := "G"
			if left < DayLen {
				col = "R"
			} else if left < 3*DayLen {
				col = "Y"
			}
			tail = fmt.Sprintf("{{%s|%s %s}}", col, lab, dlClock(left))
		} else {
			tail = "{{K|" + e.State + "}}"
		}
		sc.TextB(x, y, mark+g.dlTitle(e)+"  "+tail, CGray, CBgDark, x+w)
		y++
	}
	y++
	for _, d := range g.dlDetail(l[st.Sel]) {
		for _, ln := range Wrap(d, w) {
			if y < bottom {
				sc.TextB(x, y, ln, CGray, CBgDark, x+w)
				y++
			}
		}
	}
	if len(st.Log) > 0 && y < bottom-2 {
		y++
		sc.TextB(x, y, "{{K|Recent}}", CGray, CBgDark, x+w)
		y++
		for i := len(st.Log) - 1; i >= 0 && y < bottom; i-- {
			sc.TextB(x, y, "{{K|"+st.Log[i]+"}}", CGray, CBgDark, x+w)
			y++
		}
	}
}

// dlLines: plain lines for the journal (Journal+ tab, if wired).
func (g *Game) dlLines() []string {
	var out []string
	for _, e := range g.dlList() {
		if e.live() {
			lab, left := g.dlCount(e)
			out = append(out, fmt.Sprintf("%s: %s %s. %s", g.dlTitle(e), lab, dlClock(left), g.dlGoal(e)))
		} else {
			out = append(out, fmt.Sprintf("%s: %s. %s", g.dlTitle(e), e.State, e.Result))
		}
	}
	return out
}
