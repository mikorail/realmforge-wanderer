package engine

import (
	"fmt"
	"sort"
	"strings"
)

// companyhall.go (prefix cx): companion depth on top of companions.go, compbanter.go, compquest.go,
// comppowers.go and camplife.go (none of which is edited).
//   1. Mercenary hall: in towns of tier 1+ a roster of 3-6 recruitable people per fortnight (real life-sim
//      townsfolk plus fresh makeNPC adventurers, side rngs only), hired through joinCompanion.
//   2. Approval: each companion holds six values (mercy greed honour piety curiosity violence) drawn from
//      traits, quirks and culture. Kills (killHook), temples and ruins (enterHook) and read-only polling of
//      karma, theft, quests and gold (turn hook) move approval, which nudges loyalty and can cause comments,
//      demands, sulks, leaving and (telegraphed twice, with a way to talk them down) betrayal.
//   3. Level-up perk choices from a small tree; effects are written straight into the companion actor
//      (HP, AV, DV, attributes) or read as tags by the hourly hook.
//   4. "Talk by the fire": one scene a day after a camp rest, bond ranks with a small perk each.
// State: g.ext("cx") (planet-local: rosters) and g.ext("cxc") (global: records keyed by actor id).
// Screen: mode "cx" (tab Hall, tab Ledger, dialogue panels). All text from templates and world data.

func init() {
	unLocalKeys = append(unLocalKeys, "cx")
	turnHooks = append(turnHooks, cxTurn)
	killHooks = append(killHooks, cxKill)
	enterHooks = append(enterHooks, cxEnter)
	talkHooks = append(talkHooks, cxTalk)
	modeKeys["cx"] = cxKeys
	modeDraws["cx"] = func(g *Game) { g.cxDraw() }
	paEntries = append(paEntries,
		paEntry{Name: "Mercenary hall", Keywords: "hire recruit companions adventurers mercenary hall roster join party", Cat: "Screen", Label: "town",
			Cond: func(g *Game) bool { return g.cxHallTown() },
			Func: func(g *Game) { g.cxOpen(0, nil) }},
		paEntry{Name: "Companion ledger", Keywords: "companions approval loyalty perks bond demands ledger", Cat: "Screen", Label: "screen",
			Cond: func(g *Game) bool { return len(g.cxRows()) > 0 },
			Func: func(g *Game) { g.cxOpen(1, nil) }},
		paEntry{Name: "Talk by the fire", Keywords: "camp fire companion talk bond story evening", Cat: "Life", Label: "ability",
			Cond: func(g *Game) bool { return len(g.cxFireWho()) > 0 },
			Func: func(g *Game) { g.cxFireStart() }})
}

var cxVKey = [6]string{"mercy", "greed", "honour", "piety", "curious", "violence"}

// ---------------------------------------------------------------- state

type cxEntry struct {
	ID      string    `json:"id"`
	Key     string    `json:"k,omitempty"` // life-sim person key
	Name    string    `json:"n"`
	Species string    `json:"sp,omitempty"`
	Calling string    `json:"c"`
	Off     int       `json:"o"` // level offset from yours
	Fee     int       `json:"f,omitempty"`
	Share   int       `json:"sh,omitempty"` // percent of loot
	Loyal   int       `json:"l"`
	Q       [2]string `json:"q"`
	Val     [6]int    `json:"v"`
	Bio     string    `json:"b"`
	Female  bool      `json:"fe,omitempty"`
	Real    bool      `json:"r,omitempty"`
	Age     int       `json:"a,omitempty"`
	Culture int       `json:"cu,omitempty"`
}

type cxRoster struct {
	Per int        `json:"p"`
	E   []*cxEntry `json:"e,omitempty"`
}

type cxHall struct {
	Rost  map[int]*cxRoster `json:"r,omitempty"`
	Hired int               `json:"h,omitempty"`

	tab, sel, burg int
	text           []string
	menu           []Option
	msel           int
	scene          bool
	talker         *Actor
}

func (g *Game) cx() *cxHall {
	s := g.ext("cx", func() any { return &cxHall{Rost: map[int]*cxRoster{}} }).(*cxHall)
	if s.Rost == nil {
		s.Rost = map[int]*cxRoster{}
	}
	return s
}

type cxDemand struct {
	K   string `json:"k"` // share, mercy
	Due int    `json:"d"`
	Amt int    `json:"a,omitempty"`
}

type cxRec struct {
	Val       [6]int    `json:"v"`
	Q         [2]string `json:"q"`
	App       int       `json:"ap"`
	Bond      int       `json:"bo,omitempty"`
	Rank      int       `json:"rk,omitempty"`
	Calling   string    `json:"c,omitempty"`
	Share     int       `json:"sh,omitempty"`
	PKey      string    `json:"pk,omitempty"`
	Bio       string    `json:"bi,omitempty"`
	Hired     int       `json:"hi,omitempty"`
	Sulk      int       `json:"su,omitempty"` // turn the sulk ends
	LastOrd   string    `json:"lo,omitempty"`
	Ignored   bool      `json:"ig,omitempty"`
	Warn      int       `json:"w,omitempty"`
	WarnAt    int       `json:"wa,omitempty"`
	Kind      string    `json:"kd,omitempty"` // betrayal kind: sell theft ambush
	Dem       *cxDemand `json:"dm,omitempty"`
	Pend      int       `json:"pe,omitempty"` // perk picks waiting
	Perks     []string  `json:"pf,omitempty"`
	LvlSeen   int       `json:"ls,omitempty"`
	Said      int       `json:"sd,omitempty"`
	Talked    int       `json:"tk,omitempty"` // day+1 of the last fire talk
	Stories   int       `json:"st,omitempty"`
	Pledge    int       `json:"pl,omitempty"` // day until which you promised mercy
	Cool      int       `json:"cl,omitempty"` // no escalation before this turn
	Retry     int       `json:"rt,omitempty"`
	Scene     string    `json:"sc,omitempty"`
	Leaving   bool      `json:"lv,omitempty"`
	LeaveAt   int       `json:"la,omitempty"`
	Log       []string  `json:"lg,omitempty"`
	TempleDay int       `json:"td,omitempty"`
	KillBuf   int       `json:"kb,omitempty"`
}

type cxcState struct {
	R       map[int]*cxRec  `json:"r,omitempty"`
	Seen    map[string]bool `json:"s,omitempty"`
	Snap    [5]int          `json:"sn"`
	SnapSet bool            `json:"ss,omitempty"`
	Stolen  int             `json:"st,omitempty"`
	Gold    int             `json:"go,omitempty"`
	Quests  int             `json:"qu,omitempty"`
	Hour    int             `json:"h,omitempty"`
}

func (g *Game) cxc() *cxcState {
	s := g.ext("cxc", func() any { return &cxcState{R: map[int]*cxRec{}, Seen: map[string]bool{}} }).(*cxcState)
	if s.R == nil {
		s.R = map[int]*cxRec{}
	}
	if s.Seen == nil {
		s.Seen = map[string]bool{}
	}
	return s
}

// ---------------------------------------------------------------- records

func cxPickQuirks(r *Rng) [2]string {
	n := len(cxQuirks)
	i := r.Int(0, n-1)
	j := r.Int(0, n-1)
	if j == i {
		j = (j + 1) % n
	}
	return [2]string{cxQuirks[i].ID, cxQuirks[j].ID}
}

func cxMakeVals(cu *CultureData, tr [5]int, q [2]string) [6]int {
	var v [6]int
	if tr[3] >= 70 {
		v[cxMercy]++
	} else if tr[3] <= 30 {
		v[cxMercy]--
		v[cxGreed]++
	}
	if tr[1] >= 70 {
		v[cxHonour]++
	} else if tr[1] <= 30 {
		v[cxHonour]--
	}
	if tr[0] >= 70 {
		v[cxCurious]++
	} else if tr[0] <= 30 {
		v[cxCurious]--
	}
	if tr[4] >= 70 || tr[3] <= 25 {
		v[cxViolence]++
	}
	if cu != nil && len(cu.Values) > 0 {
		t := strings.ToLower(strings.Join(cu.Values, " "))
		has := func(ws ...string) bool {
			for _, w := range ws {
				if strings.Contains(t, w) {
					return true
				}
			}
			return false
		}
		if has("faith", "piety", "devot", "reverence", "sacred", "ritual") {
			v[cxPiety]++
		}
		if has("honour", "honor", "loyal", "duty", "oath") {
			v[cxHonour]++
		}
		if has("mercy", "compassion", "hospitality", "kindness", "peace") {
			v[cxMercy]++
		}
		if has("wealth", "trade", "commerce", "prosper", "profit") {
			v[cxGreed]++
		}
		if has("learning", "knowledge", "wisdom", "curios", "craft", "art") {
			v[cxCurious]++
		}
		if has("war", "strength", "valor", "valour", "conquest", "martial", "glory") {
			v[cxViolence]++
		}
	}
	for _, id := range q {
		if k := cxQuirkByID(id); k != nil {
			for i := range v {
				v[i] += k.V[i]
			}
		}
	}
	for i := range v {
		v[i] = clampi(v[i], -2, 2)
	}
	return v
}

func cxGuessCalling(a *Actor) string {
	if c, ok := cxRoleCalling[strings.ToLower(a.Title)]; ok {
		return c
	}
	best, bi := -1, STR
	for _, i := range []int{STR, AGI, INT, WIL} {
		if a.Attr[i] > best {
			best, bi = a.Attr[i], i
		}
	}
	switch bi {
	case AGI:
		return "scout"
	case INT:
		return "scholar"
	case WIL:
		return "mystic"
	}
	return "warrior"
}

// cxRecOf returns the record of a recruited companion, creating it deterministically for those who
// joined by other routes (side rng seed:cx:rec:id).
func (g *Game) cxRecOf(a *Actor) *cxRec {
	if a == nil || a.Comp == nil || a.Comp.Summoned || g.W == nil {
		return nil
	}
	s := g.cxc()
	if r := s.R[a.ID]; r != nil {
		return r
	}
	r := NewRng(fmt.Sprintf("%s:cx:rec:%d", g.W.Seed, a.ID))
	var cu *CultureData
	if bd := g.cbHomeBurg(a.Comp); bd != nil {
		cu = g.W.cultByID[bd.Culture]
	}
	q := cxPickQuirks(r)
	var tr [5]int
	for i := range tr {
		tr[i] = r.Int(15, 85)
	}
	rec := &cxRec{Q: q, Val: cxMakeVals(cu, tr, q), Calling: cxGuessCalling(a), LvlSeen: a.Level, App: r.Int(0, 10), Hired: g.Turn}
	s.R[a.ID] = rec
	return rec
}

func cxHasTag(rec *cxRec, tag string) bool {
	for _, id := range rec.Q {
		if k := cxQuirkByID(id); k != nil && k.Tag == tag {
			return true
		}
	}
	return false
}

func cxPerkTag(rec *cxRec, tag string) bool {
	for _, id := range rec.Perks {
		if p := cxPerkByID(id); p != nil && p.Tag == tag {
			return true
		}
	}
	return false
}

func cxHasPerk(rec *cxRec, id string) bool {
	for _, p := range rec.Perks {
		if p == id {
			return true
		}
	}
	return false
}

type cxRow struct {
	A     *Actor
	Local bool
}

// cxRows: every recruited companion, those in this zone first.
func (g *Game) cxRows() []cxRow {
	var out []cxRow
	if g.Cur == nil || g.W == nil {
		return out
	}
	for _, a := range g.cbTalkers() {
		out = append(out, cxRow{a, true})
	}
	for _, w := range g.waitingCompanions() {
		if w.A.Comp != nil && !w.A.Comp.Summoned {
			out = append(out, cxRow{w.A, false})
		}
	}
	return out
}

func cxApprovalWord(app int) string {
	switch {
	case app >= 60:
		return "{{G|adoring}}"
	case app >= 30:
		return "{{g|fond}}"
	case app >= 10:
		return "{{y|warm}}"
	case app > -10:
		return "even"
	case app > -30:
		return "{{W|cool}}"
	case app > -60:
		return "{{r|sour}}"
	}
	return "{{R|hostile}}"
}

func cxBand(app int) string {
	switch {
	case app >= 25:
		return "warm"
	case app > -10:
		return "even"
	}
	return "cool"
}

func cxValuesText(v [6]int) string {
	var out []string
	for i := 0; i < 6; i++ {
		switch {
		case v[i] >= 2:
			out = append(out, "very "+cxVWords[i][1])
		case v[i] == 1:
			out = append(out, cxVWords[i][1])
		case v[i] == -1:
			out = append(out, cxVWords[i][0])
		case v[i] <= -2:
			out = append(out, "very "+cxVWords[i][0])
		}
	}
	if len(out) == 0 {
		return "no strong feelings"
	}
	return strings.Join(out, ", ")
}

// ---------------------------------------------------------------- lines

func (g *Game) cxHash(salt string, a *Actor, mod int) uint64 {
	return HashStr(fmt.Sprintf("%s:cx:%s:%d:%d", g.W.Seed, salt, a.ID, g.Turn/30)) + uint64(mod)
}

func (g *Game) cxPickFrom(list []string, a *Actor, salt string) string {
	if len(list) == 0 {
		return ""
	}
	return list[int(g.cxHash(salt, a, 0)%uint64(len(list)))]
}

func (g *Game) cxLineFor(key string, a *Actor) string {
	if l, ok := cxLines[key]; ok {
		return g.cxPickFrom(l, a, key)
	}
	if l, ok := cxValueLines[key]; ok {
		return g.cxPickFrom(l, a, key)
	}
	return ""
}

// cxSay: a one-liner in the banter style.
func (g *Game) cxSay(a *Actor, rec *cxRec, key string, vars map[string]string) {
	t := g.cxLineFor(key, a)
	if t == "" {
		return
	}
	g.cbSay(a, g.cbFill(a, t, vars))
	if rec != nil {
		rec.Said = g.Turn
	}
}

// cxSayRaw: lines that already carry the speaker and quotes.
func (g *Game) cxSayRaw(a *Actor, key string, vars map[string]string) {
	t := g.cxLineFor(key, a)
	if t == "" {
		return
	}
	g.Msg("{{R|" + g.cbFill(a, t, vars) + "}}")
}

func (g *Game) cxRivalName(a *Actor) string {
	if nem := nxSt(g).Nem; nem != nil && nem.Name != "" {
		return nem.Name
	}
	if a.Comp != nil && strings.HasPrefix(a.Comp.Faction, "state:") {
		for i := range g.W.States {
			s := &g.W.States[i]
			if s.Alive && s.ID > 0 && g.atWar(a.Comp.Faction, fmt.Sprintf("state:%d", s.ID)) {
				return s.FullName
			}
		}
	}
	return "a rival of the road"
}

// ---------------------------------------------------------------- approval

// cxApprove changes approval. pol is the act's polarity in value space (+1 an act of that value,
// -1 its opposite, 0 neutral); a one-liner follows when the companion's reaction matches the line.
func (g *Game) cxApprove(a *Actor, rec *cxRec, d int, why string, vi, pol int) {
	if d == 0 || rec == nil {
		return
	}
	if d < 0 {
		if cxHasTag(rec, "forgiving") {
			d = d * 7 / 10
		}
		if cxPerkTag(rec, "steady") {
			d = d * 3 / 4
		}
		if d == 0 {
			d = -1
		}
	}
	rec.App = clampi(rec.App+d, -100, 100)
	rec.Log = append(rec.Log, fmt.Sprintf("day %d: %+d %s", g.Day(), d, why))
	if len(rec.Log) > 8 {
		rec.Log = rec.Log[len(rec.Log)-8:]
	}
	if vi < 0 || abs(d) < 4 || g.Turn-rec.Said < 120 || g.cbCombat() {
		return
	}
	switch {
	case pol > 0 && d > 0:
		g.cxSay(a, rec, "up:"+cxVKey[vi], nil)
	case pol < 0 && d < 0:
		g.cxSay(a, rec, "down:"+cxVKey[vi], nil)
	}
}

// cxAct applies an act with polarity pol and strength base to a companion according to their value.
func (g *Game) cxAct(a *Actor, rec *cxRec, vi, pol, base int, why string) {
	raw := base * pol * rec.Val[vi]
	d := raw / 2
	if raw != 0 && d == 0 {
		d = raw / abs(raw)
	}
	g.cxApprove(a, rec, d, why, vi, pol)
}

// cxEvent: an act seen by every companion in the zone.
func (g *Game) cxEvent(vi, pol, base int, why string) {
	for _, a := range g.cbTalkers() {
		if rec := g.cxRecOf(a); rec != nil {
			g.cxAct(a, rec, vi, pol, base, why)
		}
	}
}

func cxKill(g *Game, d, src *Actor) {
	if g.P == nil || g.W == nil || d == nil || g.Cur == nil {
		return
	}
	if d.Comp != nil && d.AI == "follower" {
		delete(g.cxc().R, d.ID)
		return
	}
	loc := g.cbTalkers()
	if len(loc) == 0 {
		return
	}
	byYou := src == g.P
	if !byYou && (src == nil || src.AI != "follower") {
		return
	}
	innocent := byYou && !g.Hostile(d) && !d.Angry && d.AI != "hostile" && crIsPerson(d)
	yield := d.Has("nxy") || d.Has("yield") || d.St("surrender") > 0
	for _, a := range loc {
		rec := g.cxRecOf(a)
		if rec == nil {
			continue
		}
		if innocent || yield {
			g.cxAct(a, rec, cxMercy, -1, 10, "you killed someone who had yielded or meant no harm")
			if innocent {
				g.cxAct(a, rec, cxHonour, -1, 4, "a killing without cause")
			}
			if rec.Pledge > g.Day() {
				rec.Pledge = 0
				g.cxApprove(a, rec, -20, "you broke your word to spare the yielding", -1, 0)
				g.addLoyalty(a, -8)
				g.Msg(fmt.Sprintf("{{R|%s: you gave your word, and broke it.}}", givenOf(a.Name)))
			}
			if rec.Val[cxMercy] >= 1 && rec.Dem == nil && g.Turn >= rec.Cool && rec.Scene == "" && g.Turn >= rec.Sulk {
				rec.Dem = &cxDemand{K: "mercy", Due: g.Turn + 1440}
				rec.Scene = "demand"
				rec.Cool = g.Turn + 2880
			}
		} else if rec.KillBuf < 3 {
			rec.KillBuf++
			g.cxAct(a, rec, cxViolence, 1, 2, "a fight won")
		}
	}
}

func cxEnter(g *Game, zn *Zone) {
	if g.P == nil || g.W == nil || zn == nil {
		return
	}
	st := g.cxc()
	for _, a := range g.cbTalkers() {
		rec := g.cxRecOf(a)
		if rec == nil {
			continue
		}
		if zn.Burg > 0 && g.cbTemple(zn, a) && (rec.TempleDay == 0 || g.Day()+1-rec.TempleDay >= 3) {
			rec.TempleDay = g.Day() + 1
			g.cxAct(a, rec, cxPiety, 1, 5, "a stop at a temple of their faith")
		}
		if zn.Kind == "wonder" || zn.Kind == "ruin" || zn.Kind == "crypt" {
			key := fmt.Sprintf("%d:%s", a.ID, zn.Key)
			if !st.Seen[key] {
				st.Seen[key] = true
				g.cxAct(a, rec, cxCurious, 1, 6, "seeing "+nonEmpty(zn.Name, "an old place"))
			}
		}
	}
}

// ---------------------------------------------------------------- turn hook

func cxTurn(g *Game) {
	if g.P == nil || g.W == nil || g.Cur == nil || g.Dead {
		return
	}
	loc := g.cbTalkers()
	if len(loc) == 0 {
		return
	}
	for _, a := range loc {
		if rec := g.cxRecOf(a); rec != nil {
			g.cxMinute(a, rec)
		}
	}
	st := g.cxc()
	if h := g.Turn / 60; h != st.Hour {
		st.Hour = h
		g.cxHourly(loc, h)
	}
	g.cxScenes(loc)
}

func (g *Game) cxMinute(a *Actor, rec *cxRec) {
	if a.Level > rec.LvlSeen {
		rec.LvlSeen = a.Level
		if len(g.cxOffer(a, rec)) > 0 {
			rec.Pend++
			if rec.Scene == "" {
				rec.Scene = "perk"
			}
			g.cxSay(a, rec, "levelup", nil)
		}
	}
	if g.Turn < rec.Sulk && !cxPerkTag(rec, "fearless") {
		if a.Order != rec.LastOrd {
			a.Order = rec.LastOrd
			if !rec.Ignored {
				rec.Ignored = true
				g.Msg(fmt.Sprintf("{{W|%s ignores your order and sulks.}}", a.Name))
			}
		}
	} else {
		rec.LastOrd = a.Order
		rec.Ignored = false
	}
}

func (g *Game) cxScenes(loc []*Actor) {
	if g.Mode != "main" || g.nearHostile() || g.ijSt().Sleep != "" {
		return
	}
	for _, a := range loc {
		rec := g.cxRecOf(a)
		if rec == nil || rec.Scene == "" {
			continue
		}
		sc := rec.Scene
		rec.Scene = ""
		g.cxOpenScene(a, rec, sc)
		return
	}
}

func (g *Game) cxHourly(loc []*Actor, hour int) {
	st := g.cxc()
	ks := crKs(g)
	var dAx [5]int
	dS, dQ, gain := 0, 0, 0
	stolen := g.uwSt().Stolen
	quests := 0
	for _, q := range g.Quests {
		if q != nil && q.Done && q.Closed {
			quests++
		}
	}
	if st.SnapSet {
		for i := range dAx {
			dAx[i] = ks.Ax[i] - st.Snap[i]
		}
		dS = stolen - st.Stolen
		dQ = quests - st.Quests
		gain = g.P.Gold - st.Gold
	}
	st.Snap, st.SnapSet, st.Stolen, st.Quests = ks.Ax, true, stolen, quests
	demanded := false
	for _, a := range loc {
		rec := g.cxRecOf(a)
		if rec == nil {
			continue
		}
		c := a.Comp
		rec.KillBuf = 0
		// karma axes, read only
		if d := dAx[0]; d > 0 {
			g.cxAct(a, rec, cxMercy, 1, mini(d, 8), "your mercy")
		}
		if d := dAx[1]; d != 0 {
			g.cxAct(a, rec, cxHonour, sign(d), mini(abs(d), 8), "your honesty")
		}
		if d := dAx[2]; d > 0 {
			g.cxAct(a, rec, cxViolence, 1, mini(d, 4), "your boldness")
		}
		if d := dAx[3]; d != 0 {
			g.cxAct(a, rec, cxGreed, -sign(d), mini(abs(d), 8), "how you handled coin")
		}
		if d := dAx[4]; d != 0 {
			g.cxAct(a, rec, cxPiety, sign(d), mini(abs(d), 8), "how you treated the holy")
		}
		if dS > 0 {
			g.cxAct(a, rec, cxHonour, -1, mini(dS, 2)*8, "theft")
			g.cxAct(a, rec, cxGreed, 1, 4, "easy coin")
		}
		if dQ > 0 {
			g.cxAct(a, rec, cxHonour, 1, mini(dQ, 3)*4, "a job done properly")
			g.cxApprove(a, rec, 3*mini(dQ, 3), "work finished", -1, 0)
		}
		// preferred stance
		switch {
		case rec.Val[cxViolence] >= 1 && (a.Order == OrdAttack || a.Order == OrdFollow):
			g.cxApprove(a, rec, 1, "orders that suit them", -1, 0)
		case rec.Val[cxViolence] >= 1 && (a.Order == OrdPassive || a.Order == OrdStay):
			g.cxApprove(a, rec, -1, "kept out of the fight", -1, 0)
		case rec.Val[cxViolence] <= -1 && (a.Order == OrdDistance || a.Order == OrdPassive):
			g.cxApprove(a, rec, 1, "orders that suit them", -1, 0)
		case rec.Val[cxViolence] <= -1 && a.Order == OrdAttack:
			g.cxApprove(a, rec, -1, "pushed into a fight", -1, 0)
		}
		// shares and demands
		if gain >= 15 && rec.Share > 0 {
			pay := gain * rec.Share / 100
			if cxPerkTag(rec, "bargainer") {
				pay /= 2
			}
			if pay > 0 {
				if g.cuSpend(pay) {
					g.Msg(fmt.Sprintf("{{Y|Your share to %s: %s.}}", givenOf(a.Name), g.cuFormat(pay)))
					g.cxApprove(a, rec, 1, "paid their share", -1, 0)
				} else {
					g.cxApprove(a, rec, -3, "an unpaid share", -1, 0)
				}
			}
		}
		if gain >= 60 && !demanded && rec.Val[cxGreed] >= 1 && rec.Share == 0 && rec.Dem == nil && rec.Scene == "" && g.Turn >= rec.Cool && g.Turn >= rec.Sulk {
			amt := gain * 15 / 100
			if cxPerkTag(rec, "bargainer") {
				amt /= 2
			}
			rec.Dem = &cxDemand{K: "share", Due: g.Turn + 1440, Amt: maxi(5, amt)}
			rec.Scene = "demand"
			rec.Cool = g.Turn + 2880
			demanded = true
		}
		if rec.Dem != nil && g.Turn > rec.Dem.Due {
			rec.Dem = nil
			g.cxApprove(a, rec, -6, "a demand ignored", -1, 0)
			g.addLoyalty(a, -2)
			g.Msg(fmt.Sprintf("{{W|%s is annoyed that you let their request lapse.}}", a.Name))
		}
		// drift and nudges
		if hour%12 == 0 {
			rec.App -= sign(rec.App)
		}
		switch {
		case rec.App >= 70 && hour%2 == 0:
			g.addLoyalty(a, 1)
		case rec.App >= 40 && hour%3 == 0:
			g.addLoyalty(a, 1)
		case rec.App <= -60:
			g.addLoyalty(a, -1)
		case rec.App <= -30 && hour%2 == 0:
			g.addLoyalty(a, -1)
		}
		if rec.Rank >= 3 && c.Loyalty < 60 {
			g.addLoyalty(a, 60-c.Loyalty)
		} else if rec.Rank >= 2 && c.Loyalty < 35 {
			g.addLoyalty(a, 1)
		}
		// perk tags
		if cxPerkTag(rec, "medic") && g.P.HP < g.P.MaxHP && !g.nearHostile() {
			g.P.HP = mini(g.P.MaxHP, g.P.HP+2)
		}
		if cxPerkTag(rec, "forager") {
			g.Hunger = maxi(0, g.Hunger-40)
		}
		if cxPerkTag(rec, "lore") && !g.nearHostile() {
			g.gainXP(2)
		}
		g.cxBehave(a, rec)
	}
	st.Gold = g.P.Gold
}

// cxBehave: sulk, leave and betray thresholds.
func (g *Game) cxBehave(a *Actor, rec *cxRec) {
	c := a.Comp
	if rec.Rank >= 3 {
		rec.Warn, rec.Leaving = 0, false
		return
	}
	firm := cxHasTag(rec, "firm")
	betrayApp := -60
	if cxHasTag(rec, "turncoat") {
		betrayApp = -45
	}
	if g.Turn >= rec.Cool && !firm && c.Loyalty <= 22 && rec.App <= betrayApp {
		switch {
		case rec.Warn == 0:
			rec.Kind = g.cxBetrayKind(a, rec)
			rec.Warn, rec.WarnAt = 1, g.Turn
			g.cxWarn(a, rec, "warn1:")
		case rec.Warn == 1 && g.Turn-rec.WarnAt >= 720:
			rec.Warn, rec.WarnAt = 2, g.Turn
			g.cxWarn(a, rec, "warn2:")
		case rec.Warn == 2 && g.Turn-rec.WarnAt >= 720:
			g.cxBetray(a, rec)
		}
		return
	}
	if rec.Warn > 0 && (c.Loyalty > 30 || rec.App > -35) {
		rec.Warn = 0
	}
	leaveLoy, leaveApp := 30, -25
	if firm {
		leaveLoy, leaveApp = 18, -40
	}
	if rec.Leaving {
		if g.Turn-rec.LeaveAt > 480 {
			g.cxDepart(a, rec, "{{R|"+a.Name+" has gone their own way.}}")
		}
		return
	}
	if c.Loyalty <= leaveLoy && rec.App <= leaveApp && g.Turn >= rec.Cool {
		rec.Leaving, rec.LeaveAt = true, g.Turn
		rec.Scene = "leave"
		return
	}
	sulkApp := -15
	if cxHasTag(rec, "sulky") {
		sulkApp = -5
	}
	if rec.App <= sulkApp && c.Loyalty < 50 && g.Turn >= rec.Sulk+1440 && !cxPerkTag(rec, "fearless") {
		rec.Sulk = g.Turn + 1440
		rec.LastOrd = a.Order
		rec.Ignored = false
		g.cxSay(a, rec, "sulk", nil)
		g.Msg(fmt.Sprintf("{{W|%s sulks and will refuse new orders for a day.}}", a.Name))
	}
}

func (g *Game) cxBetrayKind(a *Actor, rec *cxRec) string {
	switch {
	case rec.Val[cxGreed] >= 1:
		return "theft"
	case rec.Val[cxViolence] >= 1:
		return "ambush"
	case rec.Val[cxHonour] <= -1:
		return "sell"
	}
	return []string{"sell", "theft", "ambush"}[int(HashStr(fmt.Sprintf("%s:cx:kind:%d", g.W.Seed, a.ID))%3)]
}

func (g *Game) cxWarn(a *Actor, rec *cxRec, prefix string) {
	t := g.cxLineFor(prefix+rec.Kind, a)
	if t == "" {
		return
	}
	g.Msg("{{R|" + g.cbFill(a, t, nil) + "}} {{K|(Companion ledger: talk them round before it comes to worse.)}}")
	rec.Log = append(rec.Log, fmt.Sprintf("day %d: something is wrong", g.Day()))
}

func (g *Game) cxBetray(a *Actor, rec *cxRec) {
	c := a.Comp
	vars := map[string]string{"rival": g.cxRivalName(a)}
	g.cxSayRaw(a, "betray:"+rec.Kind, vars)
	name := a.Name
	switch rec.Kind {
	case "theft":
		var best *Item
		for _, it := range g.P.Inv {
			if it.Quest != 0 {
				continue
			}
			if best == nil || it.Value*maxi(1, it.Count) > best.Value*maxi(1, best.Count) {
				best = it
			}
		}
		if best != nil {
			it := g.takeOne(best)
			a.Inv = append(a.Inv, it)
			g.Msg(fmt.Sprintf("{{R|%s takes your %s and slips away.}}", name, it.Name))
		} else if amt := g.P.Gold / 5; amt > 0 && g.cuSpend(amt) {
			a.Gold += amt
			g.Msg(fmt.Sprintf("{{R|%s empties a fifth of your purse and slips away.}}", name))
		}
		a.Set("cxthief")
		g.deed("companion", fmt.Sprintf("%s robbed the wanderer %s and fled.", name, g.P.Name), 0, 0)
		g.cxDepart(a, rec, "")
	case "sell":
		state := 0
		if strings.HasPrefix(c.Faction, "state:") {
			fmt.Sscanf(c.Faction, "state:%d", &state)
		}
		if state <= 0 && g.Cur != nil && strings.HasPrefix(g.Cur.Faction, "state:") {
			fmt.Sscanf(g.Cur.Faction, "state:%d", &state)
		}
		if state > 0 {
			g.uwAddBounty(state, 100+g.P.Level*10, "sold out by "+name)
		}
		g.Msg(fmt.Sprintf("{{R|%s has sold word of you to %s.}}", name, vars["rival"]))
		g.deed("companion", fmt.Sprintf("%s sold the wanderer %s to %s.", name, g.P.Name, vars["rival"]), 0, 0)
		g.cxDepart(a, rec, "")
	default:
		delete(g.cxc().R, a.ID)
		a.AI, a.Faction, a.Angry, a.Order, a.Comp, a.F = "hostile", "outlaws", true, "", nil, CRed
		r := NewRng(fmt.Sprintf("%s:cx:ambush:%d:%d", g.W.Seed, a.ID, g.Day()))
		for i := 0; i < 2; i++ {
			b := g.Spawn("bandit", maxi(2, g.P.Level-1), r)
			if x, y, ok := g.nearestFree(g.Cur, a.X, a.Y); ok {
				b.X, b.Y, b.AI = x, y, "hostile"
				g.Cur.Actors = append(g.Cur.Actors, b)
			}
		}
		g.Msg(fmt.Sprintf("{{R|%s turns on you, and hired blades step out of cover!}}", name))
		g.deed("companion", fmt.Sprintf("%s turned on the wanderer %s in %s.", name, g.P.Name, g.Cur.Name), 0, 0)
	}
}

// cxDepart ends the bond and removes the record.
func (g *Game) cxDepart(a *Actor, rec *cxRec, msg string) {
	if msg != "" {
		g.Msg(msg)
	}
	delete(g.cxc().R, a.ID)
	if a.Comp != nil {
		g.releaseCompanion(a)
	}
}

// ---------------------------------------------------------------- hall: talk hook, roster, hire

func (g *Game) cxHallTown() bool {
	return g.Cur != nil && g.W != nil && g.Cur.Kind == "town" && g.Cur.Burg > 0 && g.W.cityTier[g.Cur.Burg] >= 1
}

func cxTalk(g *Game, a *Actor) []Option {
	if a == nil || g.P == nil || a.HP <= 0 || a.Angry || a.AI == "hostile" || a.AI == "follower" || !g.cxHallTown() {
		return nil
	}
	ok := false
	switch a.Role {
	case "innkeeper", "tavern", "lw_innkeeper", "sergeant":
		ok = true
	case "guard":
		ok = strings.Contains(strings.ToLower(a.Title), "captain") || strings.Contains(strings.ToLower(a.Title), "sergeant")
	}
	if !ok && strings.Contains(strings.ToLower(a.Title), "captain") {
		ok = true
	}
	if !ok {
		return nil
	}
	return []Option{{"Ask who is looking for work (mercenary hall)", func() { g.cxOpen(0, a) }}}
}

func (g *Game) cxRoster(burg int) *cxRoster {
	st := g.cx()
	per := g.Day() / 14
	r := st.Rost[burg]
	if r == nil || r.Per != per {
		r = g.cxGenRoster(burg, per)
		st.Rost[burg] = r
	}
	return r
}

func (g *Game) cxGenRoster(burg, per int) *cxRoster {
	out := &cxRoster{Per: per}
	bd := g.W.burgByID[burg]
	if bd == nil {
		return out
	}
	cu := g.W.cultByID[bd.Culture]
	r := NewRng(fmt.Sprintf("%s:cx:roster:%d:%d", g.W.Seed, burg, per))
	tier := g.W.cityTier[burg]
	n := clampi(2+tier+r.Int(0, 2), 3, 6)
	// real people of the town, sorted by key, shuffled by the side rng
	lvEnsure(g, burg)
	var keys []string
	for k, p := range lvSt(g).People {
		if p == nil || !p.Alive || p.Burg != burg || strings.HasPrefix(k, "s") || p.Role == "child" {
			continue
		}
		if age := p.lvAge(lvYear(g)); age < 18 || age > 55 {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for i := len(keys) - 1; i > 0; i-- {
		j := r.Int(0, i)
		keys[i], keys[j] = keys[j], keys[i]
	}
	want := mini(len(keys), r.Int(1, 2+boolInt(tier >= 2)))
	for i := 0; i < n; i++ {
		id := fmt.Sprintf("%d:%d:%d", burg, per, i)
		if i < want {
			out.E = append(out.E, g.cxBuildEntry(bd, cu, r, id, lvSt(g).People[keys[i]]))
		} else {
			out.E = append(out.E, g.cxBuildEntry(bd, cu, r, id, nil))
		}
	}
	return out
}

func (g *Game) cxBuildEntry(bd *BurgData, cu *CultureData, r *Rng, id string, p *lvPerson) *cxEntry {
	e := &cxEntry{ID: id, Off: r.Int(-2, 1), Q: cxPickQuirks(r)}
	var tr [5]int
	if p != nil {
		tr = p.Trait
		e.Real, e.Key, e.Name, e.Species, e.Female, e.Culture = true, p.Key, p.Name, p.Species, p.Female, p.Culture
		e.Age = p.lvAge(lvYear(g))
		if c, ok := cxRoleCalling[strings.ToLower(p.Role)]; ok {
			e.Calling = c
		} else if c, ok := cxRoleCalling[strings.ToLower(p.Job)]; ok {
			e.Calling = c
		} else {
			e.Calling = cxCallings[r.Int(0, len(cxCallings)-1)].Key
		}
	} else {
		for i := range tr {
			tr[i] = r.Int(15, 85)
		}
		e.Name = g.cultureName(cu, r)
		if sp := g.speciesOf(cu); sp != nil {
			e.Species = sp.ID
		}
		e.Female = r.Chance(0.5)
		e.Age = r.Int(19, 46)
		e.Calling = cxCallings[r.Int(0, len(cxCallings)-1)].Key
		if cu != nil {
			e.Culture = cu.ID
		}
	}
	e.Val = cxMakeVals(cu, tr, e.Q)
	lvl := maxi(1, g.P.Level+e.Off)
	e.Fee = 30 + lvl*25 + r.Int(0, 20)
	hag := false
	for _, id := range e.Q {
		if k := cxQuirkByID(id); k != nil {
			e.Loyal += k.Loy
			if k.Tag == "haggler" {
				hag = true
			}
		}
	}
	if hag && r.Chance(0.7) || r.Chance(0.3) {
		e.Share, e.Fee = r.Int(10, 20), 0
	}
	e.Loyal = clampi(40+r.Int(0, 15)+e.Loyal, 30, 75)
	e.Bio = g.cxBio(bd, cu, r, e, p)
	return e
}

func (g *Game) cxBio(bd *BurgData, cu *CultureData, r *Rng, e *cxEntry, p *lvPerson) string {
	cal := cxCallingByKey(e.Calling)
	if p != nil {
		s := fmt.Sprintf("%s of %s, aged %d.", titleCase(nonEmpty(p.Job, p.Role)), lvBurgName(g, p.Home), e.Age)
		if n := len(p.Life); n > 0 {
			s += " " + p.Life[n-1].Text
		}
		if p.Goal != "" {
			s += " Wants to " + p.Goal + "."
		}
		return s + " " + Pick(r, cal.Blurb)
	}
	home := bd.Name
	var near []string
	for i := range g.W.Burgs {
		b := &g.W.Burgs[i]
		if b.Alive && b.Layer == "surface" && b.ID != bd.ID && b.State == bd.State {
			near = append(near, b.Name)
		}
	}
	if len(near) > 0 {
		home = Pick(r, near)
	}
	faith := "the old ways"
	if n := len(g.W.Religion); n > 0 {
		faith = g.FactionName(fmt.Sprintf("faith:%d", g.W.Religion[clampi(bd.P, 0, n-1)]))
	}
	rep := strings.NewReplacer("{town}", bd.Name, "{home}", home, "{faith}", faith, "{realm}", g.FactionName(fmt.Sprintf("state:%d", bd.State)))
	return fmt.Sprintf("%s from %s, aged %d. Left after %s. %s %s", g.Sk.Calling(e.Calling), home, e.Age, Pick(r, cxBioReasons)+".", rep.Replace(Pick(r, cxBioDeeds)), Pick(r, cal.Blurb))
}

func (g *Game) cxLevel(e *cxEntry) int { return maxi(1, g.P.Level+e.Off) }

// cxBoundActor: the town actor already standing for a real person, if any.
func (g *Game) cxBoundActor(key string) *Actor {
	for id, k := range lvb(g).Bind {
		if k == key {
			if a := g.findActor(id); a != nil && lvEligible(a) {
				return a
			}
		}
	}
	return nil
}

func (g *Game) cxScale(a *Actor, e *cxEntry) {
	cal := cxCallingByKey(e.Calling)
	lvl := g.cxLevel(e)
	r := NewRng(fmt.Sprintf("%s:cx:gear:%s", g.W.Seed, e.ID))
	a.Level = lvl
	for i := range a.Attr {
		a.Attr[i] = 13 + cal.Attr[i] + lvl/3
	}
	a.MaxHP = 14 + lvl*4 + cal.Attr[TOU]*2
	a.HP = a.MaxHP
	a.XP = 20 + lvl*10
	tier := clampi(lvl/3, 0, 5)
	if len(a.Body) > 4 && a.Body[4].Item == nil {
		a.Body[4].Item = g.MakeWeapon(Pick(r, cal.Weapons), tier+1, r)
	}
	if cal.Armor {
		if p := a.PartFor("body"); p != nil {
			p.Item = g.MakeArmor("body", tier+1, r)
		}
		if p := a.PartFor("head"); p != nil {
			p.Item = g.MakeArmor("head", tier, r)
		}
	}
}

func (g *Game) cxActorFor(e *cxEntry, bd *BurgData, cu *CultureData) *Actor {
	var a *Actor
	if e.Real {
		a = g.cxBoundActor(e.Key)
	}
	if a == nil {
		r := NewRng(fmt.Sprintf("%s:cx:actor:%s", g.W.Seed, e.ID))
		a = g.makeNPC("villager", bd, cu, r)
		a.Name, a.Proper = e.Name, true
		if e.Species != "" {
			a.Species = e.Species
			a.Body = MakeBody(BodyPlanForSpecies(e.Species))
		}
		if x, y, ok := g.nearestFree(g.Cur, g.P.X, g.P.Y); ok {
			a.X, a.Y = x, y
		}
		g.Cur.Actors = append(g.Cur.Actors, a)
	}
	a.Title = strings.ToLower(g.Sk.Calling(e.Calling))
	a.Desc = fmt.Sprintf("%s, a %s for hire out of %s.", e.Name, a.Title, bd.Name)
	a.Role, a.Tpl = "", "npc"
	a.HomeX, a.HomeY = g.P.X, g.P.Y
	g.cxScale(a, e)
	return a
}

func (g *Game) cxHire(e *cxEntry) {
	st := g.cx()
	if g.companionCount() >= g.ijMaxCompanions(maxCompanions) {
		g.cxResult("\"You have company enough,\" says the hall's keeper. {{K|(Part ways with one in the ledger first.)}}")
		return
	}
	if e.Fee > 0 && !g.cuCanSpend(e.Fee) {
		g.cxResult("{{R|You cannot meet " + givenOf(e.Name) + "'s fee of " + g.cuFormat(e.Fee) + ".}}")
		return
	}
	bd := g.W.burgByID[g.Cur.Burg]
	if bd == nil {
		return
	}
	cu := g.W.cultByID[bd.Culture]
	a := g.cxActorFor(e, bd, cu)
	if e.Fee > 0 {
		g.cuSpend(e.Fee)
	}
	if e.Real {
		delete(lvb(g).Bind, a.ID)
		if p := lvGet(g, e.Key); p != nil {
			p.Burg, p.Dest, p.ArriveD = 0, p.Home, lvDay(g)+400
			p.addLife(lvYear(g), "move", "Took up with the wanderer "+g.P.Name+" as a companion.")
		}
	}
	g.joinCompanion(a)
	if a.Comp == nil {
		return
	}
	a.Comp.Loyalty = clampi(e.Loyal+g.Skills["persuasion"]*3, 25, 90)
	rec := &cxRec{Val: e.Val, Q: e.Q, Calling: e.Calling, Share: e.Share, PKey: e.Key, Bio: e.Bio, Hired: g.Turn, LvlSeen: a.Level, App: 5}
	g.cxc().R[a.ID] = rec
	if r := st.Rost[g.Cur.Burg]; r != nil {
		for i, x := range r.E {
			if x == e {
				r.E = append(r.E[:i], r.E[i+1:]...)
				break
			}
		}
	}
	st.Hired++
	g.cxSay(a, rec, "hired", nil)
	terms := "for " + g.cuFormat(e.Fee)
	if e.Share > 0 {
		terms = fmt.Sprintf("for %d%% of the loot", e.Share)
	}
	g.cxResult("{{G|" + a.Name + " joins you " + terms + ".}}")
}

// ---------------------------------------------------------------- perks

func (g *Game) cxOffer(a *Actor, rec *cxRec) []*cxPerkDef {
	maxTier := clampi(1+a.Level/4, 1, 3)
	var pool []*cxPerkDef
	for i := range cxPerks {
		p := &cxPerks[i]
		if cxHasPerk(rec, p.ID) || p.Tier > maxTier || (p.Req != "" && !cxHasPerk(rec, p.Req)) {
			continue
		}
		pool = append(pool, p)
	}
	r := NewRng(fmt.Sprintf("%s:cx:perk:%d:%d:%d", g.W.Seed, a.ID, a.Level, len(rec.Perks)))
	var out []*cxPerkDef
	for len(out) < 3 && len(pool) > 0 {
		i := r.Int(0, len(pool)-1)
		out = append(out, pool[i])
		pool = append(pool[:i], pool[i+1:]...)
	}
	return out
}

func (g *Game) cxApplyPerk(a *Actor, p *cxPerkDef) {
	a.MaxHP += p.HP
	a.HP = mini(a.MaxHP, a.HP+p.HP)
	a.AVNat += p.AV
	a.DVNat += p.DV
	for i, v := range p.Attr {
		a.Attr[i] += v
	}
}

func (g *Game) cxPerkDialog(a *Actor, rec *cxRec) {
	offers := g.cxOffer(a, rec)
	if len(offers) == 0 || rec.Pend <= 0 {
		rec.Pend = 0
		g.cxBack()
		return
	}
	lines := []string{fmt.Sprintf("{{Y|%s}} has reached level %d. Choose what they learn. {{K|(%d pick%s waiting)}}", a.Name, a.Level, rec.Pend, map[bool]string{true: "", false: "s"}[rec.Pend == 1])}
	var opts []Option
	for _, p := range offers {
		p := p
		opts = append(opts, Option{fmt.Sprintf("{{W|%s}} {{K|(%s)}}: %s", p.Name, p.Branch, p.Text), func() {
			rec.Perks = append(rec.Perks, p.ID)
			rec.Pend--
			g.cxApplyPerk(a, p)
			g.Msg(fmt.Sprintf("{{G|%s learns %s.}} {{K|(%s)}}", a.Name, p.Name, p.Text))
			if rec.Pend > 0 {
				g.cxPerkDialog(a, rec)
				return
			}
			g.cxBack()
		}})
	}
	opts = append(opts, Option{"Decide later.", func() { g.cxBack() }})
	g.cxDlg(lines, opts)
}

// ---------------------------------------------------------------- scenes, demands, talking round

func (g *Game) cxOpenScene(a *Actor, rec *cxRec, kind string) {
	st := g.cx()
	st.tab, st.sel, st.scene, st.talker = 1, 0, true, nil
	switch kind {
	case "perk":
		if rec.Pend > 0 {
			g.cxPerkDialog(a, rec)
		} else {
			st.scene = false
		}
	case "demand":
		if rec.Dem != nil {
			g.cxDemandDialog(a, rec)
		} else {
			st.scene = false
		}
	case "leave":
		g.cxLeaveDialog(a, rec)
	default:
		st.scene = false
	}
}

func (g *Game) cxDemandDialog(a *Actor, rec *cxRec) {
	d := rec.Dem
	if d == nil {
		g.cxBack()
		return
	}
	var opts []Option
	var lines []string
	switch d.K {
	case "share":
		lines = []string{"{{Y|" + a.Name + "}} " + g.cbFill(a, "\""+g.cxLineFor("demand:share", a)+"\"", map[string]string{"amt": g.cuFormat(d.Amt)})}
		opts = append(opts,
			Option{"Pay " + g.cuFormat(d.Amt) + ".", func() {
				if !g.cuSpend(d.Amt) {
					g.cxResult("{{R|You do not have enough.}}")
					return
				}
				rec.Dem = nil
				g.cxApprove(a, rec, 4, "paid what they asked", -1, 0)
				g.addLoyalty(a, 2)
				g.cxResult("{{G|" + givenOf(a.Name) + " takes the coin and nods.}}")
			}},
			Option{g.checkLabel("persuasion", 10) + " Haggle: half now, and a fair split later.", func() {
				rec.Dem = nil
				if g.roll2d6Check(g.checkMod("persuasion"), 10) {
					half := d.Amt / 2
					if half > 0 && !g.cuSpend(half) {
						g.cxResult("{{R|You cannot even pay half.}}")
						return
					}
					g.cxApprove(a, rec, 1, "a fair haggle", -1, 0)
					g.cxResult("{{G|[Persuasion: Success]}} " + givenOf(a.Name) + " takes half and shrugs.")
					return
				}
				g.cxApprove(a, rec, -4, "a failed haggle", -1, 0)
				g.cxResult("{{R|[Persuasion: Failure]}} " + givenOf(a.Name) + " scowls: \"Insulting.\"")
			}},
			Option{"{{r|Refuse.}}", func() {
				rec.Dem = nil
				g.cxAct(a, rec, cxGreed, 1, 8, "a refused share")
				g.cxApprove(a, rec, -2, "a refused share", -1, 0)
				g.addLoyalty(a, -3)
				g.cxResult(givenOf(a.Name) + " looks away. \"Noted.\"")
			}})
	default:
		lines = []string{"{{Y|" + a.Name + "}} " + g.cbFill(a, "\""+g.cxLineFor("demand:mercy", a)+"\"", nil)}
		opts = append(opts,
			Option{"{{G|I give my word: I will spare those who yield.}} {{K|(five days)}}", func() {
				rec.Dem = nil
				rec.Pledge = g.Day() + 5
				g.cxApprove(a, rec, 6, "your promise of mercy", -1, 0)
				g.cxResult("{{G|" + givenOf(a.Name) + " nods slowly. \"I will hold you to that.\"}}")
			}},
			Option{g.checkLabel("empathy", 10) + " Explain why it had to be done.", func() {
				rec.Dem = nil
				if g.roll2d6Check(g.checkMod("empathy"), 10) {
					g.cxApprove(a, rec, 2, "your explanation", -1, 0)
					g.cxResult("{{G|[Empathy: Success]}} " + givenOf(a.Name) + " does not like it, but understands.")
					return
				}
				g.cxApprove(a, rec, -4, "a poor excuse", -1, 0)
				g.cxResult("{{R|[Empathy: Failure]}} \"That is no reason,\" says " + givenOf(a.Name) + ".")
			}},
			Option{"{{r|It is not your place to ask.}}", func() {
				rec.Dem = nil
				g.cxApprove(a, rec, -6, "a request brushed aside", -1, 0)
				g.addLoyalty(a, -2)
				g.cxResult(givenOf(a.Name) + " says nothing, which is worse.")
			}})
	}
	opts = append(opts, Option{"Not now.", func() { g.cxBack() }})
	g.cxDlg(lines, opts)
}

func (g *Game) cxLeaveDialog(a *Actor, rec *cxRec) {
	lines := []string{"{{Y|" + a.Name + "}} " + g.cbFill(a, "\""+g.cxLineFor("leave", a)+"\"", nil)}
	cost := 60 + g.P.Level*20
	opts := []Option{
		{g.checkLabel("persuasion", 11) + " Ask them to stay.", func() {
			mod := g.checkMod("persuasion")
			if cxPerkTag(rec, "negotiator") {
				mod += 2
			}
			if g.roll2d6Check(mod, 11) {
				g.cxSettle(a, rec, 8, 10)
				g.cxResult("{{G|[Persuasion: Success]}} " + g.cbFill(a, "\""+g.cxLineFor("stay:win", a)+"\"", nil))
				return
			}
			g.cxResult("{{R|[Persuasion: Failure]}} " + g.cbFill(a, "\""+g.cxLineFor("stay:lose", a)+"\"", nil) + "\n\n{{K|They are leaving. Let them go, or pay.}}")
			rec.Retry = g.Turn + 360
		}},
		{fmt.Sprintf("Press %s into their hand to stay.", g.cuFormat(cost)), func() {
			if !g.cuSpend(cost) {
				g.cxResult("{{R|You do not have enough.}}")
				return
			}
			g.cxSettle(a, rec, 4, 10)
			g.cxResult("{{G|" + givenOf(a.Name) + " weighs the coin, then pockets it. \"For now.\"}}")
		}},
		{"{{r|Let them go.}}", func() {
			g.cxDepart(a, rec, fmt.Sprintf("{{Y|You part ways with %s.}}", a.Name))
			g.cxResult("They go. The road feels longer.")
		}},
		{"Not now.", func() { g.cxBack() }},
	}
	g.cxDlg(lines, opts)
}

// cxSettle: a row made up: warnings and sulks end, escalation pauses two days.
func (g *Game) cxSettle(a *Actor, rec *cxRec, app, loy int) {
	rec.Warn, rec.Sulk, rec.Leaving, rec.Ignored, rec.Scene = 0, 0, false, false, ""
	rec.Cool = g.Turn + 2880
	g.cxApprove(a, rec, app, "you made amends", -1, 0)
	g.addLoyalty(a, loy)
}

func (g *Game) cxRoundDialog(a *Actor, rec *cxRec) {
	if g.Turn < rec.Retry {
		g.cxResult("{{K|" + givenOf(a.Name) + " is not ready to hear more yet. Give it a few hours.}}")
		return
	}
	dc := 9 + clampi(-rec.App/20, 0, 5)
	if cxPerkTag(rec, "negotiator") {
		dc -= 2
	}
	try := func(kind string) func() {
		return func() {
			if g.roll2d6Check(g.checkMod(kind), dc) {
				g.cxSettle(a, rec, 15, 8)
				g.gainXP(6)
				g.cxResult("{{G|[" + checkKinds[kind].label + ": Success]}} " + g.cbFill(a, "\""+g.cxLineFor("round:win", a)+"\"", nil))
				return
			}
			rec.Retry = g.Turn + 360
			g.cxApprove(a, rec, -3, "an attempt that rang false", -1, 0)
			g.cxResult("{{R|[" + checkKinds[kind].label + ": Failure]}} " + g.cbFill(a, "\""+g.cxLineFor("round:lose", a)+"\"", nil))
		}
	}
	cost := 40 + g.P.Level*15
	if cxPerkTag(rec, "bargainer") {
		cost /= 2
	}
	lines := []string{"{{Y|" + a.Name + "}} " + g.cbFill(a, "\""+g.cxLineFor("talk:ask", a)+"\"", nil), "{{K|How do you make it right?}}"}
	opts := []Option{
		{g.checkLabel("persuasion", dc) + " Make your case.", try("persuasion")},
		{g.checkLabel("empathy", dc) + " Admit what you got wrong.", try("empathy")},
		{fmt.Sprintf("Make amends with %s.", g.cuFormat(cost)), func() {
			if !g.cuSpend(cost) {
				g.cxResult("{{R|You do not have enough.}}")
				return
			}
			g.cxSettle(a, rec, 10, 6)
			g.cxResult("{{G|" + givenOf(a.Name) + " accepts it, still wary but willing.}}")
		}},
		{"Not now.", func() { g.cxBack() }},
	}
	g.cxDlg(lines, opts)
}

// ---------------------------------------------------------------- bond: talk by the fire

func (g *Game) cxFirePossible(a *Actor) string {
	rec := g.cxRecOf(a)
	switch {
	case rec == nil:
		return "They are not a companion you can sit with."
	case g.cfSt().Day != g.Day()+1:
		return "Rest at a fire or a camp first."
	case g.nearHostile():
		return "Not with enemies near."
	case rec.Talked == g.Day()+1:
		return "They have said what they had to say today."
	case g.Turn < rec.Sulk:
		return "They are not speaking to you."
	}
	return ""
}

func (g *Game) cxFireWho() []*Actor {
	var out []*Actor
	if g.Cur == nil || g.W == nil || g.P == nil {
		return out
	}
	for _, a := range g.cbTalkers() {
		if g.cxFirePossible(a) == "" {
			out = append(out, a)
		}
	}
	return out
}

func (g *Game) cxFireStart() {
	who := g.cxFireWho()
	if len(who) == 0 {
		g.Msg("No one is in the mood to talk by the fire.")
		return
	}
	st := g.cx()
	st.tab, st.sel, st.scene, st.talker = 1, 0, true, nil
	if len(who) == 1 {
		g.cxFireTalk(who[0])
		return
	}
	var opts []Option
	for _, a := range who {
		a := a
		opts = append(opts, Option{"Sit with " + a.Name + ".", func() { g.cxFireTalk(a) }})
	}
	opts = append(opts, Option{"Not tonight.", func() { g.cxBack() }})
	g.cxDlg([]string{"Who do you sit with by the fire?"}, opts)
}

func (g *Game) cxStory(a *Actor, rec *cxRec) string {
	var cands []string
	if rec.Stories == 0 && rec.Bio != "" {
		cands = append(cands, "{me} tells you about their past. \""+rec.Bio+"\"")
	}
	if p := lvGet(g, rec.PKey); p != nil && rec.PKey != "" {
		n := len(p.Life)
		for i := maxi(0, n-3); i < n; i++ {
			cands = append(cands, fmt.Sprintf("{me} recalls the year %d: %s", p.Life[i].Y, p.Life[i].Text))
		}
		if p.Goal != "" {
			cands = append(cands, fmt.Sprintf("\"I still mean to %s,\" says {me}.", p.Goal))
		}
		if p.Fear != "" {
			cands = append(cands, fmt.Sprintf("{me} admits a fear: %s.", lvLower(p.Fear)))
		}
	}
	cands = append(cands, cxStories["town"]...)
	if a.Comp != nil && len(a.Comp.Kin) > 0 {
		cands = append(cands, cxStories["kin"]...)
	}
	for _, q := range rec.Q {
		cands = append(cands, cxStories[q]...)
	}
	cands = append(cands, cxStories["road"]...)
	for vi := 0; vi < 6; vi++ {
		if rec.Val[vi] >= 2 {
			cands = append(cands, cxStories["v:"+cxVKey[vi]+":+"]...)
		} else if rec.Val[vi] <= -2 {
			cands = append(cands, cxStories["v:"+cxVKey[vi]+":-"]...)
		}
	}
	if rec.Rank >= 2 {
		cands = append(cands, cxStories["secret"]...)
	}
	if len(cands) == 0 {
		return "{me} stares into the fire and says nothing."
	}
	return cands[rec.Stories%len(cands)]
}

func (g *Game) cxFireTalk(a *Actor) {
	rec := g.cxRecOf(a)
	if rec == nil {
		g.cxBack()
		return
	}
	vars := map[string]string{"rival": g.cxRivalName(a)}
	open := g.cbFill(a, g.cxPickFrom(cxFireOpen[cxBand(rec.App)], a, "open"), vars)
	story := g.cbFill(a, g.cxStory(a, rec), vars)
	lines := []string{open, "", story, "", "{{K|Bond: " + cxRankNames[rec.Rank] + ", " + fmt.Sprint(rec.Bond) + " points}}"}
	opts := []Option{
		{"Listen quietly.", func() { g.cxFireEnd(a, rec, "listen") }},
		{"Tell them a story of your own.", func() { g.cxFireEnd(a, rec, "share") }},
		{"Ask what they have left unsaid.", func() { g.cxFireEnd(a, rec, "pry") }},
		{"Not tonight.", func() { g.cxBack() }},
	}
	g.cxDlg(lines, opts)
}

func (g *Game) cxAligned(rec *cxRec) bool {
	for _, p := range [][2]int{{0, cxMercy}, {1, cxHonour}, {4, cxPiety}} {
		if g.crTrait(p[0]) >= 20 && rec.Val[p[1]] >= 1 {
			return true
		}
	}
	t := g.crTrait(3)
	return t >= 20 && rec.Val[cxGreed] <= -1 || t <= -20 && rec.Val[cxGreed] >= 1
}

func (g *Game) cxFireEnd(a *Actor, rec *cxRec, opt string) {
	band := cxBand(rec.App)
	bond, app := 2, 2
	var lines []string
	switch opt {
	case "share":
		lines = append(lines, "You tell them how you came to be called "+g.crEpithet()+".")
		if g.cxAligned(rec) {
			bond++
			app += 2
		}
	case "pry":
		bond = 1
		if band == "cool" || cxHasTag(rec, "sulky") {
			app = -3
		} else {
			app = 1
		}
	}
	rec.Talked = g.Day() + 1
	rec.Stories++
	rec.Bond += bond
	lines = append(lines, g.cbFill(a, g.cxPickFrom(cxReact[opt][band], a, opt), nil))
	g.cxApprove(a, rec, app, "a talk by the fire", -1, 0)
	g.addLoyalty(a, 1)
	for rec.Rank < 3 && rec.Bond >= cxRankNeed[rec.Rank+1] {
		rec.Rank++
		g.cxRankUp(a, rec)
		lines = append(lines, "", "{{G|"+a.Name+" is now your "+cxRankNames[rec.Rank]+".}} {{K|("+cxRankPerk[rec.Rank]+")}}")
		if l := cxRankUp[rec.Rank]; len(l) > 0 {
			lines = append(lines, g.cbFill(a, g.cxPickFrom(l, a, "rank"), nil))
		}
	}
	g.cxResult(lines...)
}

func (g *Game) cxRankUp(a *Actor, rec *cxRec) {
	switch rec.Rank {
	case 1:
		a.DVNat++
	case 2:
		a.AVNat++
		a.MaxHP += 6
		a.HP = mini(a.MaxHP, a.HP+6)
	case 3:
		a.Attr[STR]++
		a.Attr[AGI]++
		a.Attr[TOU]++
	}
	g.Msg(fmt.Sprintf("{{G|%s becomes your %s.}}", a.Name, cxRankNames[rec.Rank]))
}

// ---------------------------------------------------------------- screen

func (g *Game) cxOpen(tab int, talker *Actor) {
	st := g.cx()
	st.talker = talker
	st.tab, st.sel, st.scene = tab, 0, false
	st.text, st.menu, st.msel = nil, nil, 0
	if g.cxHallTown() {
		st.burg = g.Cur.Burg
	} else if tab == 0 {
		st.tab = 1
	}
	g.Mode = "cx"
}

func (g *Game) cxDlg(lines []string, opts []Option) {
	st := g.cx()
	st.text, st.menu, st.msel = lines, opts, 0
	g.Mode = "cx"
}

// cxResult shows lines and one Continue button.
func (g *Game) cxResult(lines ...string) {
	g.cxDlg(lines, []Option{{"Continue.", func() { g.cxBack() }}})
}

// cxBack leaves a dialogue: back to the screen, or to the map when it was a scene.
func (g *Game) cxBack() {
	st := g.cx()
	st.text, st.menu, st.msel = nil, nil, 0
	if st.scene {
		st.scene = false
		g.Mode = "main"
	}
}

func (g *Game) cxHallRows() []*cxEntry {
	if !g.cxHallTown() {
		return nil
	}
	return g.cxRoster(g.Cur.Burg).E
}

func cxKeys(g *Game, k string) {
	st := g.cx()
	if len(st.text) > 0 {
		switch k {
		case "Escape", "q":
			g.cxBack()
		case "ArrowUp", "N8", "k":
			st.msel = maxi(0, st.msel-1)
		case "ArrowDown", "N2", "j":
			st.msel = mini(maxi(0, len(st.menu)-1), st.msel+1)
		case "Enter", " ":
			if len(st.menu) == 0 {
				g.cxBack()
			} else if st.msel < len(st.menu) {
				st.menu[st.msel].Do()
			}
		default:
			if len(k) == 1 && k[0] >= '1' && k[0] <= '9' {
				if i := int(k[0] - '1'); i < len(st.menu) {
					st.msel = i
					st.menu[i].Do()
				}
			}
		}
		return
	}
	n := 0
	if st.tab == 0 {
		n = len(g.cxHallRows())
	} else {
		n = len(g.cxRows())
	}
	switch k {
	case "Escape", "q":
		tk := st.talker
		st.talker = nil
		if tk != nil && g.Talk != nil {
			g.Mode = "talk"
			g.talkMenu("\"Anything else?\"")
		} else {
			g.Mode = "main"
		}
	case "Tab", "ArrowRight", "ArrowLeft":
		if st.tab == 0 {
			st.tab = 1
		} else if g.cxHallTown() {
			st.tab = 0
		}
		st.sel = 0
	case "ArrowUp", "N8", "k":
		st.sel = maxi(0, st.sel-1)
	case "ArrowDown", "N2", "j":
		st.sel = mini(maxi(0, n-1), st.sel+1)
	case "Enter", "a":
		if st.sel >= n {
			return
		}
		if st.tab == 0 {
			g.cxHireDialog(g.cxHallRows()[st.sel])
		} else {
			g.cxActions(g.cxRows()[st.sel])
		}
	}
}

func (g *Game) cxHireDialog(e *cxEntry) {
	terms := g.cuFormat(e.Fee)
	if e.Share > 0 {
		terms = fmt.Sprintf("%d%% of whatever you earn", e.Share)
	}
	lines := []string{fmt.Sprintf("{{Y|%s}}, %s, level %d, asks %s. Starting loyalty about %s.", e.Name, g.Sk.Calling(e.Calling), g.cxLevel(e), terms, loyaltyWord(clampi(e.Loyal+g.Skills["persuasion"]*3, 25, 90)))}
	opts := []Option{
		{"{{G|Hire them.}}", func() { g.cxHire(e) }},
		{"Not now.", func() { g.cxBack() }},
	}
	g.cxDlg(lines, opts)
}

func (g *Game) cxActions(row cxRow) {
	a := row.A
	rec := g.cxRecOf(a)
	if rec == nil {
		return
	}
	lines := []string{fmt.Sprintf("{{Y|%s}}  {{K|%s, level %d}}", a.Name, g.Sk.Calling(rec.Calling), a.Level)}
	var opts []Option
	if !row.Local {
		lines = append(lines, "{{K|They wait elsewhere. Return to them to act.}}")
	} else {
		if why := g.cxFirePossible(a); why == "" {
			opts = append(opts, Option{"{{O|Talk by the fire.}}", func() { g.cxFireTalk(a) }})
		} else {
			lines = append(lines, "{{K|Talk by the fire: "+why+"}}")
		}
		if rec.Pend > 0 {
			opts = append(opts, Option{fmt.Sprintf("{{G|Choose a perk}} ({{K|%d waiting}}).", rec.Pend), func() { g.cx().scene = false; g.cxPerkDialog(a, rec) }})
		}
		if rec.Dem != nil {
			opts = append(opts, Option{"{{Y|Answer their request.}}", func() { g.cxDemandDialog(a, rec) }})
		}
		if rec.Leaving {
			opts = append(opts, Option{"{{R|Hear their parting words.}}", func() { g.cxLeaveDialog(a, rec) }})
		}
		if rec.Warn > 0 || g.Turn < rec.Sulk || rec.App <= -15 {
			opts = append(opts, Option{"{{c|Talk them round.}}", func() { g.cxRoundDialog(a, rec) }})
		}
		opts = append(opts, Option{"{{r|Part ways.}}", func() {
			g.cxDlg([]string{"Part ways with " + a.Name + "? They will go home."}, []Option{
				{"{{r|Yes, part ways.}}", func() {
					g.cxDepart(a, rec, fmt.Sprintf("You part ways with %s.", a.Name))
					g.cxBack()
				}},
				{"No.", func() { g.cxBack() }},
			})
		}})
	}
	opts = append(opts, Option{"Back.", func() { g.cxBack() }})
	g.cxDlg(lines, opts)
}

func (g *Game) cxDraw() {
	sc := g.sc
	st := g.cx()
	x, y, w, h := g.panel("Companions")
	bottom := 2 + 1 + h - 1
	col := func(on bool) string { return map[bool]string{true: "W", false: "K"}[on] }
	put := func(s string) {
		for _, ln := range Wrap(s, w) {
			if y < bottom {
				sc.TextB(x, y, ln, CGray, CBgDark, x+w)
				y++
			}
		}
	}
	sc.TextB(x, y, fmt.Sprintf("{{%s|[ Hall ]}}  {{%s|[ Ledger ]}}", col(st.tab == 0), col(st.tab == 1)), CGray, CBgDark, x+w)
	y++
	if len(st.text) > 0 {
		sc.TextB(x, y, "{{K|Up/Down choose  Enter or 1-9 pick  Esc back}}", CGray, CBgDark, x+w)
		y += 2
		for _, t := range st.text {
			if t == "" {
				y++
				continue
			}
			put(t)
		}
		y++
		for i, o := range st.menu {
			mark := "  "
			bg := CBgDark
			if i == st.msel {
				mark = "> "
				bg = CBgSel
			}
			if y < bottom {
				sc.Fill(x, y, w, 1, bg)
				sc.TextB(x, y, fmt.Sprintf("%s%d. %s", mark, i+1, o.Text), CGray, bg, x+w)
				y++
			}
		}
		return
	}
	if st.tab == 0 {
		sc.TextB(x, y, "{{K|Up/Down choose  Enter hire  Tab ledger  Esc close}}", CGray, CBgDark, x+w)
	} else {
		sc.TextB(x, y, "{{K|Up/Down choose  Enter actions  Tab hall  Esc close}}", CGray, CBgDark, x+w)
	}
	y += 2
	if st.tab == 0 {
		g.cxDrawHall(x, &y, w, bottom, put)
	} else {
		g.cxDrawLedger(x, &y, w, bottom, put)
	}
}

func (g *Game) cxDrawHall(x int, yp *int, w, bottom int, put func(string)) {
	sc := g.sc
	st := g.cx()
	rows := g.cxHallRows()
	if len(rows) == 0 {
		put("No hall here. Only towns of some size keep a roster of hired hands.")
		return
	}
	bd := g.W.burgByID[g.Cur.Burg]
	put("{{W|Hall of " + nonEmpty(g.Cur.Name, "the town") + "}}  {{K|" + cxHallSigns[int(HashStr(fmt.Sprint(g.Cur.Burg))%uint64(len(cxHallSigns)))] + "}}")
	put(fmt.Sprintf("{{K|The board changes in %d days. Companions: %d of %d.}}", 14-g.Day()%14, g.companionCount(), g.ijMaxCompanions(maxCompanions)))
	*yp++
	_ = bd
	if st.sel >= len(rows) {
		st.sel = len(rows) - 1
	}
	for i, e := range rows {
		terms := g.cuFormat(e.Fee)
		if e.Share > 0 {
			terms = fmt.Sprintf("%d%% of loot", e.Share)
		}
		mark := "  "
		if i == st.sel {
			mark = "> "
		}
		if *yp < bottom-9 {
			sc.TextB(x, *yp, fmt.Sprintf("%s{{W|%-20s}} %-14s {{K|L%-2d}} %s", mark, lvTrunc(e.Name, 20), lvTrunc(g.Sk.Calling(e.Calling), 14), g.cxLevel(e), terms), CGray, CBgDark, x+w)
			*yp++
		}
	}
	*yp++
	e := rows[st.sel]
	put(e.Bio)
	for _, id := range e.Q {
		if k := cxQuirkByID(id); k != nil {
			put("{{c|" + k.Name + ":}} " + k.Text)
		}
	}
	put("{{K|They seem " + cxValuesText(e.Val) + ". Loyalty at the start: }}" + loyaltyWord(clampi(e.Loyal+g.Skills["persuasion"]*3, 25, 90)))
}

func (g *Game) cxDrawLedger(x int, yp *int, w, bottom int, put func(string)) {
	sc := g.sc
	st := g.cx()
	rows := g.cxRows()
	if len(rows) == 0 {
		put("You travel alone. Recruit at a mercenary hall, or earn someone's trust.")
		return
	}
	if st.sel >= len(rows) {
		st.sel = len(rows) - 1
	}
	for i, r := range rows {
		rec := g.cxRecOf(r.A)
		if rec == nil {
			continue
		}
		mark := "  "
		if i == st.sel {
			mark = "> "
		}
		flag := ""
		switch {
		case rec.Leaving:
			flag = " {{R|leaving}}"
		case rec.Warn > 0:
			flag = " {{R|muttering}}"
		case g.Turn < rec.Sulk:
			flag = " {{W|sulking}}"
		case rec.Pend > 0:
			flag = " {{G|perk!}}"
		case rec.Dem != nil:
			flag = " {{Y|asks}}"
		}
		where := ""
		if !r.Local {
			where = " {{K|(waiting)}}"
		}
		if *yp < bottom-14 {
			sc.TextB(x, *yp, fmt.Sprintf("%s{{W|%-18s}} L%-2d %s  approval %s  %s%s%s", mark, lvTrunc(r.A.Name, 18), r.A.Level, loyaltyWord(r.A.Comp.Loyalty), cxApprovalWord(rec.App), cxRankNames[rec.Rank], flag, where), CGray, CBgDark, x+w)
			*yp++
		}
	}
	*yp++
	a := rows[st.sel].A
	rec := g.cxRecOf(a)
	if rec == nil {
		return
	}
	put(fmt.Sprintf("{{Y|%s}}  {{K|%s, approval %d, loyalty %d}}", a.Name, g.Sk.Calling(rec.Calling), rec.App, a.Comp.Loyalty))
	put("{{c|Values:}} " + cxValuesText(rec.Val))
	for _, id := range rec.Q {
		if k := cxQuirkByID(id); k != nil {
			put("{{c|" + k.Name + ":}} " + k.Text)
		}
	}
	next := ""
	if rec.Rank < 3 {
		next = fmt.Sprintf(", %d more to %s", cxRankNeed[rec.Rank+1]-rec.Bond, cxRankNames[rec.Rank+1])
	}
	put(fmt.Sprintf("{{c|Bond:}} %s (%d%s). %s", cxRankNames[rec.Rank], rec.Bond, next, cxRankPerk[rec.Rank]))
	if len(rec.Perks) > 0 {
		var ns []string
		for _, id := range rec.Perks {
			if p := cxPerkByID(id); p != nil {
				ns = append(ns, p.Name)
			}
		}
		put("{{c|Perks:}} " + strings.Join(ns, ", "))
	}
	if rec.Share > 0 {
		put(fmt.Sprintf("{{c|Terms:}} %d%% of your earnings", rec.Share))
	}
	if rec.Dem != nil {
		put("{{Y|They have asked something of you (Enter, then Answer).}}")
	}
	if rec.Pledge > g.Day() {
		put(fmt.Sprintf("{{c|Your word:}} spare the yielding for %d more days.", rec.Pledge-g.Day()))
	}
	for i := maxi(0, len(rec.Log)-4); i < len(rec.Log); i++ {
		put("{{K|" + rec.Log[i] + "}}")
	}
}
