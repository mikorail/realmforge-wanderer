package engine

import (
	"fmt"
	"sort"
	"strings"
)

// Heists (prefix ht). A fence or underworld contact offers a job against a real target in the world:
// a merchant house vault, a noble's manor, a temple treasury, a collector's gallery, a caravan depot or
// a guildmaster's safe. The plan has four steps: CASING (intel gathered by watching, bribing, reading a
// floor plan, asking around), APPROACH, CREW AND GEAR (up to two specialists from the town's lives
// people or your companions, read-only), and THE JOB. The job builds a vault building in the town
// district you stand in (side rng seed:ht:zoneKey, additive carving only): a hall with patrolling
// guards flagged htguard, a locked vault door set up through locks.go's lkLock, tripwires, loot with
// risk tags, a hidden alcove with an extra prize, a sewer grate and a rope for the way out. An alarm
// clock (heat) rises with sightings, noise, tripwires and lifted loot; at 30 the house guards turn on you,
// at 50 the town watch is called, at 60 the gates are chained, at 75 reinforcements arrive. Being caught
// is arrest, not death (uwAddBounty + uwJail). Fenced loot cools over days; sentimental items are hunted
// by heirs and collectors and may bring a bounty; witnesses who saw your face may too.
// Public helpers: htHeat(g) and htCool(g, n) (used by safehouse.go). State g.ext("ht"), planet-local.
// Mode "ht", palette "Heist plan". Rumours go to htRumourSink when set (nil-safe).

// htRumourSink, when set, is told about a heist (kind "heist").
var htRumourSink func(g *Game, kind, subject, text string)

const (
	htW, htH     = 15, 11
	htMaxCrew    = 2
	htCooldown   = 3 // days between offers
	htGateHeat   = 60
	htCalledHeat = 50
	htAlarmHeat  = 30
)

type htCrew struct {
	Key  string `json:"k"`
	Name string `json:"n"`
	Spec string `json:"s"`
	Cut  int    `json:"c,omitempty"` // percent of the payout; 0 for companions
	Comp bool   `json:"cp,omitempty"`
}

type htGuard struct {
	ID    int      `json:"id"`
	Route [][2]int `json:"r,omitempty"`
	I     int      `json:"i,omitempty"`
	Dog   bool     `json:"d,omitempty"`
}

type htJob struct {
	ID       int             `json:"id"`
	Burg     int             `json:"b"`
	Kind     string          `json:"kd"`
	Owner    string          `json:"ow"`
	Place    string          `json:"pl"`
	Giver    string          `json:"gv"`
	State    int             `json:"st"`
	Payout   int             `json:"po"`
	Deadline int             `json:"dl"`
	Phase    string          `json:"ph"` // casing live escape fled done failed
	Intel    map[string]bool `json:"in,omitempty"`
	Approach string          `json:"ap,omitempty"`
	Crew     []htCrew        `json:"cr,omitempty"`
	Tier     int             `json:"ti"`
	GuardN   int             `json:"gn"`
	Dogs     bool            `json:"dg,omitempty"`
	Alarms   bool            `json:"al,omitempty"`
	ShiftHr  int             `json:"sh"`
	Want     int             `json:"wa"` // index of the mark in the target's loot list
	Bribes   int             `json:"bb,omitempty"`
	AskDay   int             `json:"ad,omitempty"`
	ZoneKey  string          `json:"zk,omitempty"`
	X0       int             `json:"x0,omitempty"`
	Y0       int             `json:"y0,omitempty"`
	Built    bool            `json:"bt,omitempty"`
	PanelOn  bool            `json:"pn,omitempty"`
	Wires    [][2]int        `json:"wi,omitempty"`
	Guards   []htGuard       `json:"gd,omitempty"`
	Sighted  map[int]bool    `json:"sg,omitempty"`
	Taken    map[int]bool    `json:"tk,omitempty"`
	Witness  int             `json:"wt,omitempty"`
	Alarmed  bool            `json:"am,omitempty"`
	Called   bool            `json:"cl,omitempty"`
	Reinf    bool            `json:"rf,omitempty"`
	Opened   bool            `json:"op,omitempty"`
	Noised   bool            `json:"nz,omitempty"`
	Day0     int             `json:"d0,omitempty"`
	Mark     bool            `json:"mk,omitempty"` // the mark was lifted
}

type htHunt struct {
	Name  string `json:"n"`
	Burg  int    `json:"b"`
	State int    `json:"s"`
	Due   int    `json:"d"`
	Risk  int    `json:"r"`
}

type htState struct {
	Next     int              `json:"n"`
	Job      *htJob           `json:"j,omitempty"`
	Heat     int              `json:"h,omitempty"`
	HeatBurg int              `json:"hb,omitempty"`
	Done     int              `json:"dn,omitempty"`
	Failed   int              `json:"fl,omitempty"`
	Caught   int              `json:"ca,omitempty"`
	Earned   int              `json:"e,omitempty"`
	LastDay  int              `json:"ld,omitempty"`
	Hunts    []htHunt         `json:"hu,omitempty"`
	Closed   map[string][]int `json:"cs,omitempty"`
	Log      []string         `json:"l,omitempty"`
	Sold     int              `json:"so,omitempty"`
	DayDone  int              `json:"dd,omitempty"`

	tab, sel int
	conf     string
	rows     []htRow
}

type htRow struct {
	Text string
	Do   func()
	Dim  bool
}

func (g *Game) ht() *htState {
	s := g.ext("ht", func() any { return &htState{Closed: map[string][]int{}} }).(*htState)
	if s.Closed == nil {
		s.Closed = map[string][]int{}
	}
	if s.Job != nil {
		if s.Job.Intel == nil {
			s.Job.Intel = map[string]bool{}
		}
		if s.Job.Sighted == nil {
			s.Job.Sighted = map[int]bool{}
		}
		if s.Job.Taken == nil {
			s.Job.Taken = map[int]bool{}
		}
	}
	return s
}

func htSt(g *Game) *htState { return g.ht() }

func init() {
	unLocalKeys = append(unLocalKeys, "ht")
	talkHooks = append(talkHooks, htTalk)
	turnHooks = append(turnHooks, htTurn)
	zoneHooks = append(zoneHooks, htZone)
	enterHooks = append(enterHooks, htEnter)
	killHooks = append(killHooks, htKill)
	sideHooks = append(sideHooks, htSide)
	modeKeys["ht"] = htKeys
	modeDraws["ht"] = func(g *Game) { g.htDraw() }
	paEntries = append(paEntries, paEntry{Name: "Heist plan", Keywords: "heist job burglary rob vault casing crew approach fence heat plan thief", Cat: "Screen", Label: "screen",
		Func: func(g *Game) { g.htOpen(0) }})
}

// ---------------------------------------------------------------- small helpers

func htSafe(f func()) {
	defer func() { _ = recover() }()
	f()
}

func htFill(t string, kv ...string) string { return strings.NewReplacer(kv...).Replace(t) }

func (g *Game) htHour() int { return ((g.Turn + startMinutes) % DayLen) / 60 }

func (g *Game) htMinute() int { return (g.Turn + startMinutes) % 60 }

func (g *Game) htTarget(id string) *htTargetDef {
	for i := range htTargets {
		if htTargets[i].ID == id {
			return &htTargets[i]
		}
	}
	return &htTargets[0]
}

func htApproach(id string) *htApproachDef {
	for i := range htApproaches {
		if htApproaches[i].ID == id {
			return &htApproaches[i]
		}
	}
	return nil
}

func (g *Game) htBurgName(id int) string { return g.jbBurgName(id) }

func (g *Game) htLog(s string) {
	st := g.ht()
	st.Log = append(st.Log, fmt.Sprintf("Day %d: %s", DayOf(g.Turn)+1, s))
	if len(st.Log) > 12 {
		st.Log = st.Log[len(st.Log)-12:]
	}
}

func (g *Game) htNews(burg int, text string) {
	lvSt(g).addNews(lvDay(g), burg, "%s", text)
}

func (g *Game) htRumour(subject, text string) {
	if htRumourSink != nil {
		htSafe(func() { htRumourSink(g, "heist", subject, text) })
	}
}

func (g *Game) htActor(id int) *Actor {
	if g.Cur == nil {
		return nil
	}
	for _, a := range g.Cur.Actors {
		if a.ID == id && a.HP > 0 {
			return a
		}
	}
	return nil
}

// htStanding: how far the underworld trusts you. Read-only over uw and the fq underworld chain.
func (g *Game) htStanding() int {
	n := 0
	htSafe(func() {
		u := g.uwSt()
		n += u.Stolen + u.Fenced + u.Won + u.Jailed + u.Crowd/10
		for _, f := range g.fqFacs() {
			if f != nil && f.Kind == "uw" {
				n += g.fqStanding(f) / 10
			}
		}
	})
	return n
}

func (g *Game) htMoveStanding(d int, why string) {
	htSafe(func() {
		for _, f := range g.fqFacs() {
			if f != nil && f.Kind == "uw" {
				g.fqMove(f.Key, d, why)
				return
			}
		}
	})
}

// htHeat is the heat the watch holds against you after heists, 0..100.
func htHeat(g *Game) int {
	if g == nil {
		return 0
	}
	return g.ht().Heat
}

// htCool lowers heat by n (lying low, fences, time).
func htCool(g *Game, n int) {
	if g == nil || n <= 0 {
		return
	}
	st := g.ht()
	if st.Heat <= 0 {
		return
	}
	st.Heat = maxi(0, st.Heat-n)
	if st.Heat == 0 {
		st.HeatBurg = 0
	}
}

func (g *Game) htWarm(n int, why string) {
	st := g.ht()
	j := st.Job
	if j == nil {
		return
	}
	if a := htApproach(j.Approach); a != nil {
		n = n * a.Heat / 100
	}
	if htCrewHas(j, "lookout") && strings.Contains(why, "saw") {
		n = n * 2 / 3
	}
	if n < 1 {
		n = 1
	}
	st.Heat = mini(100, st.Heat+n)
	st.HeatBurg = j.Burg
	if why != "" {
		g.Msg("{{R|" + why + "}}")
	}
	if st.Heat >= htAlarmHeat && !j.Alarmed {
		g.htAlarm(j)
	}
	if st.Heat >= htCalledHeat && !j.Called {
		j.Called = true
		for _, a := range g.Cur.Actors {
			if a.HP > 0 && uwIsGuardOf(a, j.State) {
				a.Angry = true
			}
		}
		g.Msg("{{R|Whistles answer from the street. The town watch is coming.}}")
	}
	if st.Heat >= 75 && !j.Reinf {
		j.Reinf = true
		g.htReinforce(j)
	}
}

func htCrewHas(j *htJob, spec string) bool {
	for _, c := range j.Crew {
		if c.Spec == spec {
			return true
		}
	}
	return false
}

func (g *Game) htHas(words ...string) *Item {
	for _, it := range g.P.Inv {
		n := strings.ToLower(it.Name)
		for _, w := range words {
			if strings.Contains(n, w) {
				return it
			}
		}
	}
	return nil
}

func (g *Game) htGearHave(id string) bool {
	switch id {
	case "picks":
		return len(g.lkPicks()) > 0
	case "rope":
		return g.htHas("rope", "grappl") != nil
	case "sedative":
		return g.htHas("sedative", "sleeping draught", "lullaby", "poppy") != nil
	case "smoke":
		return g.htHas("smoke") != nil
	case "papers":
		for _, it := range g.P.Inv {
			if uwIsForged(it) {
				return true
			}
		}
		return g.htHas("livery", "house colours") != nil
	case "digging":
		return g.htHas("shovel", "spade", "pickaxe", "pick-axe", "mattock") != nil
	}
	return false
}

func (g *Game) htDisguised(j *htJob) bool {
	if g.ssDisguised(uwFac(j.State)) {
		return true
	}
	return g.htGearHave("papers") && j.Approach == "disguise"
}

func htKnown(j *htJob, id string) bool { return j != nil && j.Intel != nil && j.Intel[id] }

// ---------------------------------------------------------------- offers

func htTalk(g *Game, a *Actor) []Option {
	var out []Option
	st := g.ht()
	fence := false
	htSafe(func() { fence = uwIsFence(a) || a.Role == "bmdealer" })
	if fence && g.htStanding() >= 1 {
		if st.Job == nil && DayOf(g.Turn)-st.LastDay >= htCooldown {
			out = append(out, Option{"{{M|Ask about a job.}}", func() { g.htOfferMenu(a) }})
		}
		if st.Job != nil {
			out = append(out, Option{"{{M|About the job...}}", func() { g.htReport(a) }})
		}
		if g.htLootCount() > 0 {
			out = append(out, Option{"{{M|Fence the heist goods.}}", func() { g.htFenceMenu(a, 0) }})
		}
	}
	if (a.Role == "elder" || a.Role == "notable") && g.Cur != nil {
		for _, it := range g.P.Inv {
			if it.Extra != nil && it.Extra["htsent"] > 0 && it.Extra["htburg"] == g.Cur.Burg {
				it := it
				out = append(out, Option{"{{G|Return " + it.Name + " quietly.}}", func() { g.htReturn(a, it) }})
				break
			}
		}
	}
	return out
}

func (g *Game) htReport(a *Actor) {
	st := g.ht()
	j := st.Job
	if j == nil {
		g.talkMenu("\"Nothing between us.\"")
		return
	}
	if it := g.htMarkItem(j); it != nil {
		g.htDeliver(a, it)
		return
	}
	opts := []Option{
		{"{{y|Drop the job.}}", func() { g.htAbandon(); g.talkMenu("\"Walk away, then. Quietly.\"") }},
		{"Never mind.", func() { g.talkMenu("\"Don't be long.\"") }},
	}
	txt := fmt.Sprintf("\"%s, %s. Deadline in %s. Show me the %s when you have it.\"", j.Place, g.htBurgName(j.Burg), jbClock(j.Deadline-g.Turn), g.htMarkName(j))
	g.uwOpenMenu("The job", txt, opts, false)
	g.Talk = a
}

func (g *Game) htMarkName(j *htJob) string {
	t := g.htTarget(j.Kind)
	if j.Want >= 0 && j.Want < len(t.Loot) {
		return t.Loot[j.Want].Name
	}
	return "prize"
}

func (g *Game) htMarkItem(j *htJob) *Item {
	for _, it := range g.P.Inv {
		if it.Extra != nil && it.Extra["htjob"] == j.ID && it.Extra["htmark"] > 0 {
			return it
		}
	}
	return nil
}

func (g *Game) htOfferMenu(a *Actor) {
	j := g.htMakeJob(a)
	if j == nil {
		g.talkMenu("\"Nothing worth your time just now.\"")
		return
	}
	t := g.htTarget(j.Kind)
	pitch := htFill(t.Pitch, "{owner}", j.Owner)
	line := htFill(htContactLines[int(HashStr(fmt.Sprint(g.Seed, ":ht:line:", j.ID, a.ID)))%len(htContactLines)], "{pitch}", pitch)
	px, py := g.Cur.WX, g.Cur.WY
	tx, ty, _ := g.jbBurgXY(j.Burg)
	txt := fmt.Sprintf("%s\n\nTarget: %s in %s (%d parasangs %s). Payout: %s for the %s. Deadline: %s.",
		line, htFill(t.Place, "{owner}", j.Owner), g.htBurgName(j.Burg), jbPar(tx-px, ty-py), jbDirName(tx-px, ty-py),
		g.cuFormat(j.Payout), g.htMarkName(j), jbClock(j.Deadline-g.Turn))
	opts := []Option{
		{"{{G|Take the job.}}", func() {
			g.ht().Job = j
			g.htLog("Took a job from " + a.Name + ": " + t.Short + " in " + g.htBurgName(j.Burg))
			g.Msg("{{Y|A job is on. Open the Heist plan (palette) to case it and plan it.}}")
			g.talkMenu("\"Case it first. Rushed jobs end in chains.\"")
		}},
		{"Not interested.", func() { g.talkMenu("\"Suit yourself.\"") }},
	}
	g.uwOpenMenu("A job", txt, opts, false)
	g.Talk = a
}

// htMakeJob builds the offer deterministically from the world seed, the contact and the week.
func (g *Game) htMakeJob(giver *Actor) *htJob {
	if g.Cur == nil || g.W == nil {
		return nil
	}
	week := DayOf(g.Turn) / 7
	r := NewRng(fmt.Sprint(g.Seed, ":ht:offer:", g.Cur.Burg, ":", giver.ID, ":", week))
	px, py := g.Cur.WX, g.Cur.WY
	type cand struct {
		b *BurgData
		d int
	}
	var cs []cand
	ids := make([]int, 0, len(g.W.burgByID))
	for id := range g.W.burgByID {
		ids = append(ids, id)
	}
	sort.Ints(ids)
	for _, id := range ids {
		b := g.W.burgByID[id]
		if b == nil || b.P < 0 || b.P >= g.W.W*g.W.H {
			continue
		}
		wx, wy, z := unTownZone(g.W, b)
		if z != 0 || !g.W.In(wx, wy) || g.W.IsWater(g.W.Idx(wx, wy)) {
			continue
		}
		d := jbPar(b.P%g.W.W-px, b.P/g.W.W-py)
		if b.ID == g.Cur.Burg {
			d += 6
		}
		if d > 14 {
			continue
		}
		cs = append(cs, cand{b, d})
	}
	if len(cs) == 0 {
		return nil
	}
	sort.SliceStable(cs, func(i, k int) bool { return cs[i].d < cs[k].d })
	if len(cs) > 6 {
		cs = cs[:6]
	}
	bd := cs[r.Int(0, len(cs)-1)].b
	tier := g.W.cityTier[bd.ID]
	var kinds []string
	kinds = append(kinds, "vault", "vault")
	if bd.State > 0 {
		kinds = append(kinds, "manor", "safe")
	}
	if tier >= 1 || bd.Capital {
		kinds = append(kinds, "display", "temple")
	}
	if bd.Port || tier >= 1 {
		kinds = append(kinds, "depot")
	}
	kinds = append(kinds, "temple")
	kind := Pick(r, kinds)
	st := g.ht()
	st.Next++
	cu := g.W.cultByID[bd.Culture]
	owner := g.cultureName(cu, r)
	if kind == "manor" && bd.Governor != "" {
		owner = bd.Governor
	}
	t := g.htTarget(kind)
	j := &htJob{ID: int(HashStr(fmt.Sprint(g.Seed, ":ht:id:", bd.ID, ":", week, ":", giver.ID))%900000) + 1, Burg: bd.ID, Kind: kind, Owner: owner,
		Giver: giver.Name, State: bd.State, Phase: "casing", Intel: map[string]bool{}, Sighted: map[int]bool{}, Taken: map[int]bool{}}
	j.Place = htFill(t.Place, "{owner}", owner)
	j.Tier = clampi(2+tier+r.Int(0, 1), 2, 5)
	j.GuardN = clampi(2+t.Guards+r.Int(0, 1)+tier/2, 2, 4)
	j.Dogs = r.Chance(0.35)
	j.Alarms = r.Chance(0.5 + 0.1*float64(tier))
	j.ShiftHr = r.Int(21, 23)
	j.Want = r.Int(0, len(t.Loot)-1)
	lvl := maxi(1, g.P.Level)
	j.Payout = (260 + 170*j.Tier + 90*tier) * (100 + 4*lvl) / 100
	j.Deadline = g.Turn + (7+r.Int(0, 7))*DayLen
	return j
}

func (g *Game) htAbandon() {
	st := g.ht()
	if st.Job == nil {
		return
	}
	g.htDropLoot(st.Job)
	st.Failed++
	g.htMoveStanding(-3, " (you dropped a job)")
	g.htLog("Dropped the job at " + st.Job.Place)
	st.Job = nil
	st.LastDay = DayOf(g.Turn)
}

func (g *Game) htDropLoot(j *htJob) {
	for _, it := range append([]*Item{}, g.P.Inv...) {
		if it.Extra != nil && it.Extra["htjob"] == j.ID && it.Extra["htmark"] > 0 {
			g.removeInv(it)
		}
	}
}

// ---------------------------------------------------------------- casing

func (g *Game) htInTarget(j *htJob) bool {
	return g.Cur != nil && g.Cur.Kind == "town" && g.Cur.Burg == j.Burg
}

func (g *Game) htLearn(j *htJob, ids ...string) {
	for _, id := range ids {
		if !j.Intel[id] {
			j.Intel[id] = true
			for _, d := range htIntel {
				if d.ID == id {
					g.Msg("{{G|Intel: " + d.Label + ".}}")
				}
			}
		}
	}
}

func (g *Game) htCasingRows(j *htJob) []htRow {
	var rows []htRow
	here := g.htInTarget(j)
	where := func(ok bool, s string) string {
		if !ok {
			return "{{K|" + s + " (be in " + g.htBurgName(j.Burg) + ")}}"
		}
		return s
	}
	dusk := g.htHour() >= 17 && g.htHour() <= 22
	rows = append(rows, htRow{Text: where(here, "Watch the house at dusk (one hour)"), Dim: !here || j.Intel["shifts"], Do: func() {
		if !here {
			g.Msg("You must be in " + g.htBurgName(j.Burg) + ".")
			return
		}
		if !dusk {
			g.Msg("{{y|The watch changes at dusk. Come back between 17:00 and 22:00.}}")
			return
		}
		g.Turn += 60
		g.htLearn(j, "shifts")
		g.Msg(fmt.Sprintf("{{Y|You watch %d guards. The watch changes at %02d:00.}}", j.GuardN, j.ShiftHr))
	}})
	cost := maxi(10, j.Payout/12)
	rows = append(rows, htRow{Text: where(here, fmt.Sprintf("Bribe a servant (%s)", g.cuFormat(cost))), Dim: !here || j.Bribes >= 2, Do: func() {
		if !here || j.Bribes >= 2 {
			return
		}
		if !g.cuSpend(cost) {
			g.Msg("{{y|You cannot cover the servant's price.}}")
			return
		}
		j.Bribes++
		if j.Bribes == 1 {
			g.htLearn(j, "locks", "dogs")
		} else {
			g.htLearn(j, "alarms", "secret")
		}
	}})
	pc := maxi(20, j.Payout/8)
	rows = append(rows, htRow{Text: fmt.Sprintf("Buy a floor plan from the contact (%s)", g.cuFormat(pc)), Dim: g.htPlanItem(j) != nil, Do: func() {
		if g.htPlanItem(j) != nil {
			g.Msg("You already carry the plan.")
			return
		}
		if !g.cuSpend(pc) {
			g.Msg("{{y|The contact wants more than you carry.}}")
			return
		}
		it := &Item{ID: g.newID(), Kind: "letter", Name: "floor plan of " + j.Place, Glyph: '?', F: CYellow, Count: 1, Value: 5,
			Text: "A hand-drawn plan: halls, the vault door and a note on the back about the way in.", Extra: map[string]int{"htplan": j.ID}}
		g.give(it)
		g.Msg("{{G|You pay for a floor plan.}}")
	}})
	rows = append(rows, htRow{Text: "Study the floor plan", Dim: g.htPlanItem(j) == nil || j.Intel["layout"], Do: func() {
		if g.htPlanItem(j) == nil {
			g.Msg("You carry no plan for this job.")
			return
		}
		g.Turn += 30
		g.htLearn(j, "layout", "secret", "locks")
	}})
	rows = append(rows, htRow{Text: where(here, "Ask the townsfolk (once a day)"), Dim: !here || j.AskDay == DayOf(g.Turn)+1, Do: func() {
		if !here || j.AskDay == DayOf(g.Turn)+1 {
			return
		}
		j.AskDay = DayOf(g.Turn) + 1
		r := NewRng(fmt.Sprint(g.Seed, ":ht:ask:", j.ID, ":", DayOf(g.Turn)))
		roll := r.Int(1, 20) + g.P.M(EGO) + 2*g.Skills["persuasion"]
		switch {
		case roll >= 16:
			g.htLearn(j, "dogs", "shifts")
		case roll >= 10:
			g.htLearn(j, Pick(r, []string{"shifts", "dogs", "locks"}))
		default:
			g.Msg("Nobody will say much. A few look at you a bit too long.")
			if r.Chance(0.3) {
				g.htWarm(1, "")
			}
		}
		g.Turn += 20
	}})
	rows = append(rows, htRow{Text: "{{y|Drop this job}}", Do: func() {
		st := g.ht()
		if st.conf != "drop" {
			st.conf = "drop"
			g.Msg("{{y|Press Enter again to drop the job.}}")
			return
		}
		st.conf = ""
		g.htAbandon()
	}})
	return rows
}

func (g *Game) htPlanItem(j *htJob) *Item {
	for _, it := range g.P.Inv {
		if it.Extra != nil && it.Extra["htplan"] == j.ID {
			return it
		}
	}
	return nil
}

// ---------------------------------------------------------------- approach and crew

func (g *Game) htSecretCount(j *htJob) int {
	n := 0
	htSafe(func() {
		if j.ZoneKey == "" {
			return
		}
		if z := sxSt(g).Z[j.ZoneKey]; z != nil {
			for _, s := range z.S {
				if !s.Found {
					n++
				}
			}
		}
	})
	return n
}

func (g *Game) htApproachOK(j *htJob, id string) (bool, string) {
	switch id {
	case "sneak":
		if g.ssMod() < 2 {
			return true, "your sneak rating is low: wear lighter gear"
		}
		return true, ""
	case "disguise":
		if g.ssDisguised(uwFac(j.State)) || g.htGearHave("papers") {
			return true, ""
		}
		return false, "needs the realm's attire (two pieces) or forged papers/livery"
	case "con":
		if g.Skills["persuasion"] >= 1 || g.P.M(EGO) >= 2 {
			return true, ""
		}
		return false, "needs Persuasion or a strong Ego"
	case "smash":
		if g.P.M(STR) >= 1 || htCrewHas(j, "muscle") {
			return true, ""
		}
		return false, "needs muscle: Strength or a hired bruiser"
	case "tunnel":
		if !j.Intel["secret"] {
			return false, "needs the secret way in (intel)"
		}
		if g.htGearHave("digging") || htCrewHas(j, "digger") {
			return true, ""
		}
		return false, "needs a shovel or a digger"
	case "roof":
		if g.htGearHave("rope") {
			return true, ""
		}
		return false, "needs rope"
	}
	return false, "?"
}

func (g *Game) htApproachRows(j *htJob) []htRow {
	var rows []htRow
	for _, a := range htApproaches {
		a := a
		ok, why := g.htApproachOK(j, a.ID)
		mark := "  "
		if j.Approach == a.ID {
			mark = "* "
		}
		txt := fmt.Sprintf("%s{{W|%s}}: %s", mark, a.Name, a.Desc)
		if why != "" {
			txt += " {{K|(" + why + ")}}"
		}
		rows = append(rows, htRow{Text: txt, Dim: !ok, Do: func() {
			if ok {
				j.Approach = a.ID
				g.Msg("{{Y|Approach: " + a.Name + ".}}")
			} else {
				g.Msg("{{y|Not yet: " + why + ".}}")
			}
		}})
	}
	return rows
}

func htSpecOf(key string, seed string) string {
	return htSpecs[int(HashStr(seed+":ht:spec:"+key))%len(htSpecs)].ID
}

func (g *Game) htCandidates() []htCrew {
	var out []htCrew
	seen := map[string]bool{}
	for _, a := range g.localCompanions() {
		k := fmt.Sprint("c:", a.ID)
		out = append(out, htCrew{Key: k, Name: a.Name, Spec: htSpecOf(k, g.Seed), Comp: true})
		seen[k] = true
	}
	if g.Cur != nil && g.Cur.Burg > 0 {
		var ps []*lvPerson
		htSafe(func() { ps = lvPresent(g, g.Cur.Burg) })
		for _, p := range ps {
			if p.Role == "elder" || p.Opinion < -25 || HashStr(g.Seed+":ht:cand:"+p.Key)%5 != 0 {
				continue
			}
			k := "p:" + p.Key
			if seen[k] {
				continue
			}
			out = append(out, htCrew{Key: k, Name: p.Name, Spec: htSpecOf(k, g.Seed), Cut: 10 + int(HashStr(k)%9)})
			if len(out) >= 8 {
				break
			}
		}
	}
	return out
}

func (g *Game) htCrewRows(j *htJob) []htRow {
	var rows []htRow
	for _, gr := range htGearList {
		have := g.htGearHave(gr.ID)
		m := "{{R|missing}}"
		if have {
			m = "{{G|ready}}"
		}
		rows = append(rows, htRow{Text: fmt.Sprintf("Gear: %s %s {{K|%s}}", gr.Name, m, gr.Why), Dim: true})
	}
	for i := range j.Crew {
		c := j.Crew[i]
		cn := c
		rows = append(rows, htRow{Text: fmt.Sprintf("{{G|Crew}} %s, %s {{K|(Enter to let go)}}", cn.Name, cn.Spec), Do: func() {
			for k := range j.Crew {
				if j.Crew[k].Key == cn.Key {
					j.Crew = append(j.Crew[:k], j.Crew[k+1:]...)
					break
				}
			}
		}})
	}
	for _, c := range g.htCandidates() {
		c := c
		have := false
		for _, o := range j.Crew {
			if o.Key == c.Key {
				have = true
			}
		}
		if have {
			continue
		}
		cut := "no cut"
		if c.Cut > 0 {
			cut = fmt.Sprintf("%d%% of the payout", c.Cut)
		}
		var desc string
		for _, s := range htSpecs {
			if s.ID == c.Spec {
				desc = s.Desc
			}
		}
		rows = append(rows, htRow{Text: fmt.Sprintf("Hire %s, %s {{K|%s. %s}}", c.Name, c.Spec, cut, desc), Dim: len(j.Crew) >= htMaxCrew, Do: func() {
			if len(j.Crew) >= htMaxCrew {
				g.Msg("{{y|You can take two specialists at most.}}")
				return
			}
			j.Crew = append(j.Crew, c)
			g.Msg("{{G|" + c.Name + " is in.}}")
		}})
	}
	return rows
}

// ---------------------------------------------------------------- building the job

func (g *Game) htSite(zn *Zone) (int, int, bool) {
	type cand struct{ x, y, d int }
	var cs []cand
	for y := 1; y+htH < MH; y++ {
		for x := 1; x+htW < MW; x++ {
			cs = append(cs, cand{x, y, abs(x+htW/2-g.P.X) + abs(y+htH/2-g.P.Y)})
		}
	}
	sort.SliceStable(cs, func(i, j int) bool { return cs[i].d < cs[j].d })
	okCell := func(x, y int) bool {
		t := zn.At(x, y)
		if !t.D().Walk || t == TUp || t == TDown || t == TDoor || t == TDoorOpen || t == TLocked || t == TUpdraft || t == TPortal || zn.WaterAt(x, y) > 0 {
			return false
		}
		if _, n := zn.Notes[y*MW+x]; n {
			return false
		}
		return zn.ActorAt(x, y) == nil && len(zn.ItemsAt(x, y)) == 0
	}
	for _, c := range cs {
		good := true
		for y := c.y; y < c.y+htH && good; y++ {
			for x := c.x; x < c.x+htW; x++ {
				if !okCell(x, y) {
					good = false
					break
				}
			}
		}
		if good && c.d < 60 {
			return c.x, c.y, true
		}
	}
	return 0, 0, false
}

func (g *Game) htLootItem(j *htJob, d htLootDef, mark, prize bool, x, y int) *Item {
	lvl := maxi(1, g.P.Level)
	v := d.Base * (100 + 20*(j.Tier-2)) / 100 * (100 + 3*lvl) / 100
	if prize {
		v = v * 5 / 2
	}
	day := DayOf(g.Turn)
	it := &Item{ID: g.newID(), Kind: "trophy", Name: d.Name, Glyph: d.Glyph, F: CYellow, Count: 1, Value: v, Weight: 2, Lore: d.Text, X: x, Y: y,
		Extra: map[string]int{"htjob": j.ID, "htrisk": d.Risk, "htburg": j.Burg, "htday": day, "htcool": day + 4 + 2*d.Risk}}
	if d.Sent {
		it.Extra["htsent"] = 1
	}
	if mark {
		it.Extra["htmark"] = 1
	}
	if prize {
		it.Extra["htprize"] = 1
	}
	g.uwMarkStolen(it, j.State)
	return it
}

// htBuild carves the vault building and populates it. Side rng only.
func (g *Game) htBuild(zn *Zone, j *htJob) bool {
	x0, y0, ok := g.htSite(zn)
	if !ok {
		return false
	}
	r := NewRng(fmt.Sprint(g.Seed, ":ht:", zn.Key))
	if zn.Notes == nil {
		zn.Notes = map[int]string{}
	}
	j.X0, j.Y0, j.ZoneKey = x0, y0, zn.Key
	at := func(dx, dy int) (int, int) { return x0 + dx, y0 + dy }
	set := func(dx, dy int, t Terrain) { x, y := at(dx, dy); zn.Set(x, y, t); delete(zn.Notes, y*MW+x) }
	note := func(dx, dy int, s string) { x, y := at(dx, dy); zn.Notes[y*MW+x] = s }
	for dy := 0; dy < 9; dy++ {
		for dx := 0; dx < htW; dx++ {
			t := TFloor
			if dy == 0 || dy == 8 || dx == 0 || dx == htW-1 || dx == 9 {
				t = TWall
			} else if dx >= 10 {
				t = TCarpet
			}
			set(dx, dy, t)
		}
	}
	for dy := 9; dy < htH; dy++ {
		for dx := 0; dx < htW; dx++ {
			if zn.At(x0+dx, y0+dy) != TFloor {
				set(dx, dy, TCobble)
			}
		}
	}
	set(4, 8, TDoor)
	set(9, 4, TLocked)
	vx, vy := at(9, 4)
	L := &lkLock{ID: lkID(zn.Key, vx, vy), Kind: "door", Name: "vault door", X: vx, Y: vy, Tier: j.Tier, Lvl: zn.Level}
	if htKnown(j, "layout") || htKnown(j, "locks") {
		note(9, 4, fmt.Sprintf("The vault door: grade %d lock.", j.Tier))
	}
	if htCrewHas(j, "lockman") {
		L.Tier = maxi(1, L.Tier-1)
	}
	g.lkSt().Locks[L.ID] = L
	// the hidden alcove: walls (12,1) and (13,2); alcove (13,1)
	set(12, 1, TWall)
	set(13, 2, TWall)
	set(13, 1, TFloor)
	prize := htPrizes[r.Int(0, len(htPrizes)-1)]
	t := g.htTarget(j.Kind)
	px, py := at(13, 1)
	zn.Items = append(zn.Items, g.htLootItem(j, htLootDef{Name: prize.Name, Base: 120, Risk: 2, Glyph: '$', Text: prize.Text}, false, true, px, py))
	// loot in the vault
	spots := [][2]int{{11, 3}, {12, 5}, {11, 6}, {13, 4}}
	for i, d := range t.Loot {
		if i >= len(spots) {
			break
		}
		x, y := at(spots[i][0], spots[i][1])
		zn.Items = append(zn.Items, g.htLootItem(j, d, i == j.Want, false, x, y))
	}
	// a strongbox via the lock helpers
	cx, cy := at(11, 1)
	CL := &lkLock{Tier: clampi(j.Tier-1, 1, 5), Name: "strongbox"}
	g.lkPlaceChest(zn, r, cx, cy, CL)
	// escape ways
	set(1, 10, TRubble)
	note(1, 10, htSewerNote)
	if j.Approach == "roof" {
		set(6, 1, TScaffold)
		note(6, 1, htRoofNote)
	}
	// tunnel breach
	if j.Approach == "tunnel" {
		set(12, 8, TRubble)
		for dy := 9; dy < htH; dy++ {
			set(12, dy, TDirt)
		}
		note(12, 10, "The mouth of your tunnel.")
	}
	// tripwires
	j.Wires = nil
	if j.Alarms {
		for i := 0; i < 3; i++ {
			dx, dy := r.Int(3, 8), r.Int(2, 6)
			if zn.At(x0+dx, y0+dy) == TFloor && (dx != 4 || dy != 7) {
				j.Wires = append(j.Wires, [2]int{x0 + dx, y0 + dy})
				if htKnown(j, "alarms") || htKnown(j, "layout") {
					note(dx, dy, htWireNote)
				}
			}
		}
	}
	if j.Intel["secret"] {
		note(13, 2, "The plaster here sounds hollow.")
	}
	// guards
	bd := g.W.burgByID[j.Burg]
	cu := g.W.cultByID[0]
	if bd != nil {
		cu = g.W.cultByID[bd.Culture]
	}
	routes := [][][2]int{
		{{2, 2}, {8, 2}, {8, 6}, {2, 6}},
		{{3, 4}, {7, 4}},
		{{8, 3}, {8, 5}},
		{{1, 9}, {13, 9}},
	}
	n := j.GuardN
	if htCrewHas(j, "muscle") {
		n--
	}
	j.Guards = nil
	for i := 0; i < n && i < len(routes) && bd != nil; i++ {
		a := g.makeNPC("guard", bd, cu, r)
		a.Name = Pick(r, htGuardNames)
		a.Title = "guard"
		a.AI, a.Angry = "still", false
		a.Set("htguard")
		rt := make([][2]int, len(routes[i]))
		for k, p := range routes[i] {
			rt[k] = [2]int{x0 + p[0], y0 + p[1]}
		}
		a.X, a.Y, a.HomeX, a.HomeY = rt[0][0], rt[0][1], rt[0][0], rt[0][1]
		zn.Actors = append(zn.Actors, a)
		j.Guards = append(j.Guards, htGuard{ID: a.ID, Route: rt})
	}
	if j.Dogs {
		a := g.Spawn("wolf", zn.Level, r)
		a.Name, a.AI = "guard dog", "still"
		a.Set("htguard")
		a.X, a.Y, a.HomeX, a.HomeY = x0+6, y0+3, x0+6, y0+3
		zn.Actors = append(zn.Actors, a)
		j.Guards = append(j.Guards, htGuard{ID: a.ID, Dog: true, Route: [][2]int{{x0 + 6, y0 + 3}, {x0 + 6, y0 + 6}}})
	}
	// gear used at the start
	if it := g.htHas("sedative", "sleeping draught", "lullaby", "poppy"); it != nil {
		g.uwTake(&g.P.Inv, it)
		for i, gd := range j.Guards {
			if (j.Dogs && gd.Dog) || (!j.Dogs && i == 0) {
				if a := g.htActor(gd.ID); a != nil {
					a.AddSt("asleep", 150)
					g.Msg("{{Y|Your sedative is already working on the " + a.Name + ".}}")
				}
			}
		}
	}
	// where you start
	sx, sy := 4, 10
	switch j.Approach {
	case "disguise", "con":
		sx, sy = 4, 9
	case "tunnel":
		sx, sy = 12, 10
	case "roof":
		sx, sy = 6, 2
	}
	g.P.X, g.P.Y = x0+sx, y0+sy
	j.Built = true
	return true
}

func (g *Game) htBegin(j *htJob) {
	if j.Phase != "casing" || g.Cur == nil {
		return
	}
	if !g.htInTarget(j) {
		g.Msg("{{y|The job is in " + g.htBurgName(j.Burg) + ". Go there first.}}")
		return
	}
	if j.Approach == "" {
		g.Msg("{{y|Pick an approach first.}}")
		return
	}
	if ok, why := g.htApproachOK(j, j.Approach); !ok {
		g.Msg("{{y|Your approach fails: " + why + ".}}")
		return
	}
	if g.Turn > j.Deadline {
		g.Msg("{{y|The deadline has passed.}}")
		return
	}
	if !g.htBuild(g.Cur, j) {
		g.Msg("{{y|There is no quiet corner in this district for the job. Try another district of the town.}}")
		return
	}
	j.Phase, j.Day0 = "live", DayOf(g.Turn)
	g.recompute()
	g.Mode = "main"
	g.Msg("{{Y|You are at the job: " + j.Place + ".}}")
	// con: one roll at the door
	switch j.Approach {
	case "con":
		bonus := 0
		if htCrewHas(j, "face") {
			bonus = 4
		}
		roll := g.R.Int(1, 20) + g.P.M(EGO) + 2*g.Skills["persuasion"] + bonus
		if roll >= 10+j.Tier*2 {
			g.Msg("{{G|The door guard shrugs and waves you in. They will not look at you twice.}}")
			g.htMute(j, 3)
		} else {
			g.Msg("{{R|The door guard does not buy it, and calls for the sergeant.}}")
			g.htWarm(14, "")
		}
	case "smash":
		g.Msg("{{Y|You do not have to be quiet. You do have to be fast.}}")
	case "disguise":
		g.Msg("{{G|In the house colours, nobody looks at your face.}}")
	}
	g.htLog("Began the job at " + j.Place)
}

// htMute: a successful con puts guards at ease (they sight you less) for a while via the Sighted set.
func (g *Game) htMute(j *htJob, n int) {
	for i := range j.Guards {
		if i < n {
			j.Sighted[j.Guards[i].ID] = true
		}
	}
}

func htZone(g *Game, zn *Zone, r *Rng) {
	// Heists build on demand (htBegin); a regenerated zone with a live job gets its layout again.
	st := g.ht()
	j := st.Job
	if j == nil || j.Phase != "live" || j.Built || zn == nil || zn.Key != j.ZoneKey {
		return
	}
	_ = r
}

func htEnter(g *Game, zn *Zone) {
	st := g.ht()
	j := st.Job
	if j != nil && (j.Phase == "escape" || j.Phase == "live") && zn != nil && zn.Key != j.ZoneKey && j.ZoneKey != "" {
		g.htFled(j, "You leave the district behind.")
	}
	g.htGates()
}

func htKill(g *Game, d, src *Actor) {
	st := g.ht()
	j := st.Job
	if j == nil || d == nil || !d.Has("htguard") || src != g.P {
		return
	}
	if !d.Angry && g.htDisguised(j) == false {
		g.htWarm(5, "A guard dies. Somebody will have heard it.")
	}
}

// ---------------------------------------------------------------- the job in play

func htTurn(g *Game) {
	if g == nil || g.Cur == nil || g.P == nil {
		return
	}
	st := g.ht()
	if d := DayOf(g.Turn); d != st.DayDone {
		if st.DayDone == 0 || d < st.DayDone {
			st.DayDone = d
		}
		for n := 0; st.DayDone < d && n < 60; n++ {
			st.DayDone++
			if st.Job == nil || (st.Job.Phase != "live" && st.Job.Phase != "escape") {
				htCool(g, 3)
			}
			g.htDay()
		}
		st.DayDone = d
	}
	j := st.Job
	if j != nil && j.Phase == "casing" && g.Turn > j.Deadline {
		g.Msg("{{y|The job has gone cold: the deadline passed.}}")
		st.Failed++
		g.htMoveStanding(-2, " (a job left to rot)")
		g.htLog("Missed the deadline at " + j.Place)
		st.Job = nil
		st.LastDay = DayOf(g.Turn)
		j = nil
	}
	g.htGates()
	if j == nil || g.Cur.Key != j.ZoneKey || (j.Phase != "live" && j.Phase != "escape") {
		return
	}
	g.htLive(j)
}

func (g *Game) htRect(j *htJob, x, y int) bool {
	return x >= j.X0 && x < j.X0+htW && y >= j.Y0 && y < j.Y0+9
}

func (g *Game) htLive(j *htJob) {
	zn := g.Cur
	// noise: vault lock
	vx, vy := j.X0+9, j.Y0+4
	if L := g.lkSt().Locks[lkID(zn.Key, vx, vy)]; L != nil && L.Broken && !j.Noised {
		j.Noised = true
		g.htWarm(25, "The vault door crashes open. The whole house heard that.")
	}
	if zn.At(vx, vy) != TLocked && !j.Opened {
		j.Opened = true
		g.Msg("{{G|The vault is open.}}")
	}
	if g.ss().Loud > g.Turn && g.Turn%5 == 0 {
		g.htWarm(3, "")
	}
	// tripwires
	for i := 0; i < len(j.Wires); i++ {
		if g.P.X == j.Wires[i][0] && g.P.Y == j.Wires[i][1] {
			delete(zn.Notes, j.Wires[i][1]*MW+j.Wires[i][0])
			j.Wires = append(j.Wires[:i], j.Wires[i+1:]...)
			g.htWarm(18, "A tripwire sings and a bell rings somewhere in the house.")
			break
		}
	}
	// loot
	for _, it := range g.P.Inv {
		if it.Extra == nil || it.Extra["htjob"] != j.ID || j.Taken[it.ID] {
			continue
		}
		j.Taken[it.ID] = true
		if it.Extra["htmark"] > 0 {
			j.Mark = true
		}
		g.htWarm(it.Extra["htrisk"]*3, "")
		if j.Phase == "live" {
			j.Phase = "escape"
			g.Msg("{{Y|The take is in your hands. Now get out: the grate in the yard, the rope, or the street.}}")
		}
	}
	g.htPanelCheck(j)
	g.htMoveGuards(j)
	g.htSight(j)
	// the way out
	if j.Phase == "escape" {
		if g.P.X == j.X0+1 && g.P.Y == j.Y0+10 {
			g.htEscapeVia(j, "sewer")
		} else if j.Approach == "roof" && g.P.X == j.X0+6 && g.P.Y == j.Y0+1 {
			g.htEscapeVia(j, "roof")
		} else if !g.htRect(j, g.P.X, g.P.Y) && (g.htDisguised(j) || !g.htAngryNear(j, 10)) {
			g.htFled(j, "You are out of sight of the house.")
		}
	}
	// caught
	if g.P.HP*4 <= g.P.MaxHP && g.htAngryNear(j, 1) {
		g.htCaught(j)
	}
}

func (g *Game) htAngryNear(j *htJob, d int) bool {
	for _, a := range g.Cur.Actors {
		if a.HP <= 0 || !a.Angry || !(a.Has("htguard") || uwIsGuardOf(a, j.State)) {
			continue
		}
		if abs(a.X-g.P.X) <= d && abs(a.Y-g.P.Y) <= d && g.losClear(a.X, a.Y, g.P.X, g.P.Y) {
			return true
		}
	}
	return false
}

func (g *Game) htPanelCheck(j *htJob) {
	px, py := j.X0+13, j.Y0+2
	if j.PanelOn || abs(g.P.X-px) > 1 || abs(g.P.Y-py) > 1 {
		return
	}
	p := 4 + 3*(maxi(0, g.P.M(INT))+g.Skills["perception"])
	if j.Intel["secret"] {
		p *= 3
	}
	if g.R.Int(1, 100) <= p {
		j.PanelOn = true
		g.Cur.Set(px, py, TDoorOpen)
		delete(g.Cur.Notes, py*MW+px)
		g.Msg("{{G|A panel slides: there is a hidden alcove in the vault wall.}}")
	}
}

func (g *Game) htMoveGuards(j *htJob) {
	changing := g.htHour() == j.ShiftHr && g.htMinute() < 15
	for i := range j.Guards {
		gd := &j.Guards[i]
		a := g.htActor(gd.ID)
		if a == nil || a.Angry || a.AI != "still" || !g.ssAwake(a) || len(gd.Route) == 0 {
			continue
		}
		if changing && i == 0 {
			continue
		}
		tg := gd.Route[gd.I%len(gd.Route)]
		if a.X == tg[0] && a.Y == tg[1] {
			gd.I = (gd.I + 1) % len(gd.Route)
			tg = gd.Route[gd.I]
		}
		dx, dy := sign(tg[0]-a.X), sign(tg[1]-a.Y)
		for _, st := range [][2]int{{dx, dy}, {dx, 0}, {0, dy}} {
			if st[0] == 0 && st[1] == 0 {
				continue
			}
			nx, ny := a.X+st[0], a.Y+st[1]
			if g.Cur.ActorAt(nx, ny) != nil || (nx == g.P.X && ny == g.P.Y) {
				continue
			}
			if g.Cur.At(nx, ny) == TDoor {
				g.Cur.Set(nx, ny, TDoorOpen)
				break
			}
			if g.canEnter(a, nx, ny) {
				a.X, a.Y = nx, ny
				break
			}
		}
	}
}

func (g *Game) htSight(j *htJob) {
	changing := g.htHour() == j.ShiftHr && g.htMinute() < 15
	dis := g.htDisguised(j)
	for i := range j.Guards {
		gd := j.Guards[i]
		a := g.htActor(gd.ID)
		if a == nil || a.Angry || !g.ssAwake(a) {
			continue
		}
		d := maxi(abs(a.X-g.P.X), abs(a.Y-g.P.Y))
		if d > 7 || !g.losClear(a.X, a.Y, g.P.X, g.P.Y) {
			continue
		}
		chance := 55 - 4*g.ssMod() + 3*d
		if gd.Dog {
			chance += 15
		}
		if dis {
			chance /= 3
		}
		if changing && i == 0 {
			chance /= 2
		}
		if htCrewHas(j, "lookout") {
			chance = chance * 2 / 3
		}
		chance = clampi(chance, 4, 85)
		if j.Sighted[a.ID] && !g.R.Chance(0.2) {
			continue
		}
		if !g.R.Chance(float64(chance) / 100) {
			continue
		}
		if !j.Sighted[a.ID] {
			j.Sighted[a.ID] = true
			if !dis {
				j.Witness++
			}
		}
		if dis {
			g.htWarm(2, "")
			continue
		}
		a.Angry, a.AI = true, "hostile"
		g.htWarm(10, "The "+a.Name+" saw you and shouts.")
	}
}

func (g *Game) htAlarm(j *htJob) {
	j.Alarmed = true
	for i := range j.Guards {
		if a := g.htActor(j.Guards[i].ID); a != nil {
			a.Angry, a.AI = true, "hostile"
			delete(a.Status, "asleep")
		}
	}
	g.Msg("{{R|ALARM! A bell is ringing in the house.}}")
}

func (g *Game) htReinforce(j *htJob) {
	bd := g.W.burgByID[j.Burg]
	if bd == nil {
		return
	}
	cu := g.W.cultByID[bd.Culture]
	r := NewRng(fmt.Sprint(g.Seed, ":ht:reinf:", g.Cur.Key))
	n := 0
	for i := 0; i < 2; i++ {
		a := g.makeNPC("guard", bd, cu, r)
		a.Angry, a.AI = true, "hostile"
		if x, y, ok := g.Cur.FreeSpot(r, func(x, y int) bool { return !g.htRect(j, x, y) && abs(x-g.P.X)+abs(y-g.P.Y) < 22 }); ok {
			a.X, a.Y, a.HomeX, a.HomeY = x, y, x, y
			g.Cur.Actors = append(g.Cur.Actors, a)
			n++
		}
	}
	if n > 0 {
		g.Msg("{{R|Reinforcements pour into the street.}}")
	}
}

func (g *Game) htEscapeVia(j *htJob, how string) {
	if g.htAngryNear(j, 1) {
		g.Msg("{{y|A guard is on you. You cannot slip away like this.}}")
		return
	}
	switch how {
	case "sewer":
		htCool(g, 20)
		g.htFled(j, "You drop through the grate and crawl through the dark. When you come up, the shouting is far behind you.")
	default:
		htCool(g, 15)
		g.htFled(j, "You haul yourself up the rope and over the roofs.")
	}
	zn := g.Cur
	r := NewRng(fmt.Sprint(g.Seed, ":ht:out:", zn.Key))
	if x, y, ok := zn.FreeSpot(r, func(x, y int) bool { return !g.htRect(j, x, y) && abs(x-g.P.X)+abs(y-g.P.Y) > 18 }); ok {
		g.P.X, g.P.Y = x, y
		g.recompute()
	}
}

// htFled ends the escape: smoke, witnesses and the consequences.
func (g *Game) htFled(j *htJob, text string) {
	if j.Phase == "fled" || j.Phase == "done" {
		return
	}
	st := g.ht()
	if j.Phase == "live" && !j.Mark {
		// left without the take
		j.Phase = "casing"
		g.Msg("{{y|You pulled out empty-handed. The job still stands, but the house is awake.}}")
		g.htCleanup(j)
		return
	}
	j.Phase = "fled"
	g.Msg("{{G|" + text + "}}")
	if it := g.htHas("smoke"); it != nil {
		g.uwTake(&g.P.Inv, it)
		htCool(g, 10)
		g.Msg("{{Y|Your smoke bomb covers the last stretch.}}")
	}
	if j.Witness > 0 && !g.htDisguised(j) {
		p := 0.3 + 0.1*float64(j.Witness)
		if g.R.Chance(p) {
			g.uwAddBounty(j.State, 60+10*j.Witness+st.Heat/2, " (the house guards describe your face)")
		}
	}
	t := g.htTarget(j.Kind)
	g.htNews(j.Burg, htFill(Pick(NewRng(fmt.Sprint(g.Seed, ":ht:news:", j.ID)), htNewsLines), "{place}", j.Place))
	g.htRumour(j.Place, htFill(Pick(NewRng(fmt.Sprint(g.Seed, ":ht:rum:", j.ID)), htRumourLines), "{owner}", j.Owner, "{place}", j.Place))
	_ = t
	g.htLog("Got out of " + j.Place)
	g.htCleanup(j)
}

func (g *Game) htCleanup(j *htJob) {
	if g.Cur == nil {
		return
	}
	for _, w := range j.Wires {
		delete(g.Cur.Notes, w[1]*MW+w[0])
	}
}

func (g *Game) htCaught(j *htJob) {
	st := g.ht()
	st.Caught++
	state := j.State
	g.P.HP = maxi(g.P.HP, g.P.MaxHP/3)
	g.htNews(j.Burg, htFill(htNewsCaught[0], "{place}", j.Place))
	g.htDropLoot(j)
	for _, a := range g.Cur.Actors {
		if a.Has("htguard") || uwIsGuardOf(a, state) {
			a.Angry = false
			if a.Has("htguard") {
				a.AI = "still"
			}
		}
	}
	g.Msg("{{R|Hands seize you. The job is over.}}")
	g.uwAddBounty(state, 60+st.Heat, " (burglary)")
	g.uwJail(state)
	htCool(g, 30)
	g.P.X, g.P.Y = j.X0+4, j.Y0+10
	g.htMoveStanding(-4, " (caught on a job)")
	g.htLog("Caught at " + j.Place)
	st.Failed++
	g.htCleanup(j)
	st.Job = nil
	st.LastDay = DayOf(g.Turn)
	g.recompute()
}

// htSurrender is the screen action: give yourself up to a nearby guard.
func (g *Game) htSurrender() {
	j := g.ht().Job
	if j == nil || (j.Phase != "live" && j.Phase != "escape") || g.Cur == nil || g.Cur.Key != j.ZoneKey {
		g.Msg("There is nobody to surrender to.")
		return
	}
	g.Mode = "main"
	g.htCaught(j)
}

// ---------------------------------------------------------------- gates

func (g *Game) htGates() {
	if g.Cur == nil || g.Cur.Kind != "town" {
		return
	}
	st := g.ht()
	zn := g.Cur
	if st.Heat >= htGateHeat && st.HeatBurg > 0 && zn.Burg == st.HeatBurg {
		if _, done := st.Closed[zn.Key]; !done {
			var idx []int
			for i, t := range zn.T {
				if t == TGateArch {
					idx = append(idx, i)
				}
			}
			if len(idx) > 0 {
				for _, i := range idx {
					zn.T[i] = TLocked
				}
				st.Closed[zn.Key] = idx
				g.Msg("{{R|Chains rattle: the town gates are locked down.}}")
			}
		}
		return
	}
	if idx, done := st.Closed[zn.Key]; done && (st.Heat < 35 || st.HeatBurg != zn.Burg) {
		for _, i := range idx {
			if i >= 0 && i < len(zn.T) && zn.T[i] == TLocked {
				zn.T[i] = TGateArch
			}
		}
		delete(st.Closed, zn.Key)
		g.Msg("{{G|The gates are opened again.}}")
	}
}

// ---------------------------------------------------------------- payout, fences, hunts

func (g *Game) htLootCount() int {
	n := 0
	for _, it := range g.P.Inv {
		if it.Extra != nil && it.Extra["htjob"] > 0 {
			n++
		}
	}
	return n
}

func (g *Game) htDeliver(a *Actor, it *Item) {
	st := g.ht()
	j := st.Job
	if j == nil {
		return
	}
	pay := j.Payout
	if g.Turn > j.Deadline {
		pay /= 2
	}
	cuts := 0
	var names []string
	for _, c := range j.Crew {
		if c.Cut > 0 {
			cuts += pay * c.Cut / 100
			names = append(names, c.Name)
		}
	}
	g.uwTake(&g.P.Inv, it)
	got := g.cuEarn(pay - cuts)
	g.Msg("{{G|The contact counts out " + got + ".}}")
	if cuts > 0 {
		g.Msg(fmt.Sprintf("{{Y|%s took a cut: %s.}}", strings.Join(names, " and "), g.cuFormat(cuts)))
	}
	st.Done++
	st.Earned += pay - cuts
	st.LastDay = DayOf(g.Turn)
	g.htMoveStanding(8, " (a job done clean)")
	g.deedOnce("ht"+fmt.Sprint(j.ID), "heist", "Robbed "+j.Place+" in "+g.htBurgName(j.Burg)+".")
	g.gainXP(30 + 15*j.Tier)
	g.htLog("Delivered the take from " + j.Place)
	st.Job = nil
	if g.Talk != nil {
		g.talkMenu("\"Clean work. Come back when the dust has settled.\"")
	}
}

func (g *Game) htFenceFactor(it *Item, f *Actor) float64 {
	day := DayOf(g.Turn)
	fac := 1.0
	if it.Extra != nil && it.Extra["htcool"] > day {
		total := maxi(1, it.Extra["htcool"]-it.Extra["htday"])
		rem := it.Extra["htcool"] - day
		fac = 0.55 + 0.45*(1-float64(rem)/float64(total))
	}
	if it.Extra != nil && it.Extra["htsent"] > 0 {
		fac *= 0.7
	}
	if j := g.ht().Job; j != nil && htCrewHas(j, "cutter") {
		fac *= 1.1
	}
	return fac
}

func (g *Game) htFenceMenu(f *Actor, page int) {
	state := g.uwStateHere()
	var items []*Item
	for _, it := range g.P.Inv {
		if it.Extra != nil && it.Extra["htjob"] > 0 {
			items = append(items, it)
		}
	}
	var opts []Option
	for i, it := range items {
		if i < page*12 || i >= page*12+12 {
			continue
		}
		it := it
		pr := maxi(1, int(float64(g.uwFencePays(it, state, f))*g.htFenceFactor(it, f)))
		tag := ""
		if it.Extra["htcool"] > DayOf(g.Turn) {
			tag += " {{R|[hot: " + fmt.Sprint(it.Extra["htcool"]-DayOf(g.Turn)) + "d]}}"
		}
		if it.Extra["htsent"] > 0 {
			tag += " {{M|[heirloom]}}"
		}
		opts = append(opts, Option{fmt.Sprintf("%s%s: %s", it.Display(), tag, g.cuFormat(pr)), func() {
			if f.Gold < pr {
				g.Msg("\"I'm light on coin today.\"")
				g.htFenceMenu(f, page)
				return
			}
			f.Gold -= pr
			g.cuEarn(pr)
			one := g.uwTake(&g.P.Inv, it)
			f.Inv = append(f.Inv, one)
			g.ht().Sold++
			if one.Extra["htsent"] > 0 {
				g.ht().Hunts = append(g.ht().Hunts, htHunt{Name: one.Name, Burg: one.Extra["htburg"], State: g.htBurgState(one.Extra["htburg"]), Due: DayOf(g.Turn) + 3 + one.Extra["htrisk"], Risk: one.Extra["htrisk"]})
				g.Msg("{{M|The fence turns it over in their hands. \"Somebody is going to miss this.\"}}")
			}
			g.Msg("The fence pays " + g.cuFormat(pr) + ".")
			g.htFenceMenu(f, page)
		}})
	}
	if (page+1)*12 < len(items) {
		opts = append(opts, Option{"More...", func() { g.htFenceMenu(f, page+1) }})
	}
	opts = append(opts, Option{"Done.", func() { g.talkMenu("\"Pleasure.\"") }})
	g.uwOpenMenu("Heist goods", fmt.Sprintf("Their purse: %s. Hot goods fetch less until they cool; heirlooms fetch less and get hunted.", g.cuFormat(f.Gold)), opts, false)
	g.Talk = f
}

func (g *Game) htBurgState(id int) int {
	if b := g.W.burgByID[id]; b != nil {
		return b.State
	}
	return 0
}

func (g *Game) htReturn(a *Actor, it *Item) {
	st := g.ht()
	g.uwTake(&g.P.Inv, it)
	for i := 0; i < len(st.Hunts); i++ {
		if st.Hunts[i].Name == it.Name {
			st.Hunts = append(st.Hunts[:i], st.Hunts[i+1:]...)
			break
		}
	}
	if s := g.htBurgState(g.Cur.Burg); s > 0 {
		g.AddRep(uwFac(s), 12, " (returned a stolen heirloom)")
	}
	g.htMoveStanding(-3, " (gave an heirloom back)")
	g.talkMenu("\"I will see it reaches its owner. You were never here.\"")
}

// htDay: once a day. Heirs and collectors decide whether the trail leads to you.
func (g *Game) htDay() {
	st := g.ht()
	day := DayOf(g.Turn)
	for i := 0; i < len(st.Hunts); i++ {
		h := st.Hunts[i]
		if day < h.Due {
			continue
		}
		st.Hunts = append(st.Hunts[:i], st.Hunts[i+1:]...)
		i--
		r := NewRng(fmt.Sprint(g.Seed, ":ht:hunt:", h.Name, ":", h.Due))
		p := 0.3 + 0.1*float64(h.Risk)
		if g.htHas("papers") != nil {
			p -= 0.1
		}
		if r.Chance(p) {
			g.htNews(h.Burg, "The heirs of "+h.Name+" offer a reward: the trail leads to a stranger who spent too freely.")
			g.uwAddBounty(h.State, 40*h.Risk, " (the "+h.Name+" is traced to you)")
		} else {
			g.htNews(h.Burg, "The search for the "+h.Name+" has gone cold.")
		}
	}
}

// ---------------------------------------------------------------- sidebar

func htSide(g *Game) []string {
	st := g.ht()
	var out []string
	if st.Heat > 0 {
		c := "y"
		if st.Heat >= 50 {
			c = "R"
		}
		out = append(out, fmt.Sprintf("{{%s|Heat %d}}", c, st.Heat))
	}
	if j := st.Job; j != nil && (j.Phase == "live" || j.Phase == "escape") && g.Cur != nil && g.Cur.Key == j.ZoneKey {
		out = append(out, "{{Y|Heist: "+j.Phase+"}}")
	}
	return out
}

// ---------------------------------------------------------------- screen

var htTabs = []string{"Job", "Approach", "Crew and gear", "Go"}

func (g *Game) htOpen(tab int) {
	st := g.ht()
	st.tab, st.sel, st.conf = tab, 0, ""
	g.Mode = "ht"
}

func (g *Game) htBuildRows() []htRow {
	st := g.ht()
	j := st.Job
	if j == nil {
		return nil
	}
	switch st.tab {
	case 0:
		return g.htCasingRows(j)
	case 1:
		return g.htApproachRows(j)
	case 2:
		return g.htCrewRows(j)
	}
	var rows []htRow
	live := (j.Phase == "live" || j.Phase == "escape") && g.Cur != nil && g.Cur.Key == j.ZoneKey
	rows = append(rows, htRow{Text: "{{G|Begin the job here and now}}", Dim: j.Phase != "casing", Do: func() { g.htBegin(j) }})
	rows = append(rows, htRow{Text: "{{R|Surrender to the nearest guard}}", Dim: !live, Do: func() { g.htSurrender() }})
	return rows
}

func htKeys(g *Game, k string) {
	st := g.ht()
	switch k {
	case "Escape", "q":
		g.Mode = "main"
		return
	case "Tab", "ArrowRight":
		st.tab = (st.tab + 1) % len(htTabs)
		st.sel, st.conf = 0, ""
		return
	case "ArrowLeft":
		st.tab = (st.tab + len(htTabs) - 1) % len(htTabs)
		st.sel, st.conf = 0, ""
		return
	}
	rows := g.htBuildRows()
	switch k {
	case "ArrowUp", "N8", "k":
		st.sel = maxi(0, st.sel-1)
		st.conf = ""
	case "ArrowDown", "N2", "j":
		st.sel = mini(maxi(0, len(rows)-1), st.sel+1)
		st.conf = ""
	case "Enter", " ":
		if st.sel < len(rows) && rows[st.sel].Do != nil {
			rows[st.sel].Do()
		}
	}
}

func (g *Game) htDraw() {
	st := g.ht()
	sc := g.sc
	x, y, w, h := g.panel("Heist plan")
	bottom := 2 + 1 + h - 1
	tabs := ""
	for i, t := range htTabs {
		c := "K"
		if i == st.tab {
			c = "W"
		}
		tabs += fmt.Sprintf("{{%s|[ %s ]}}  ", c, t)
	}
	sc.TextB(x, y, tabs, CGray, CBgDark, x+w)
	y++
	sc.TextB(x, y, fmt.Sprintf("{{K|Up/Down choose  Enter act  Tab next  Esc close.}}   Heat: {{%s|%d}}", map[bool]string{true: "R", false: "Y"}[st.Heat >= 50], st.Heat), CGray, CBgDark, x+w)
	y += 2
	j := st.Job
	if j == nil {
		for _, l := range Wrap("No job. A fence or black-market dealer sells work to people who have proved themselves: steal, fence or win in the pit first, then ask about a job. Completed: "+fmt.Sprint(st.Done)+", failed: "+fmt.Sprint(st.Failed)+", caught: "+fmt.Sprint(st.Caught)+".", w) {
			sc.TextB(x, y, l, CGray, CBgDark, x+w)
			y++
		}
		for _, l := range st.Log {
			if y < bottom {
				sc.TextB(x, y, "{{K|"+l+"}}", CGray, CBgDark, x+w)
				y++
			}
		}
		return
	}
	t := g.htTarget(j.Kind)
	head := fmt.Sprintf("{{W|%s}} in %s. Mark: %s. Payout %s. Deadline %s. Phase: %s.", j.Place, g.htBurgName(j.Burg), g.htMarkName(j), g.cuFormat(j.Payout), jbClock(j.Deadline-g.Turn), j.Phase)
	for _, l := range Wrap(head, w) {
		sc.TextB(x, y, l, CGray, CBgDark, x+w)
		y++
	}
	_ = t
	if st.tab == 0 {
		for _, d := range htIntel {
			v := "{{K|unknown}}"
			if j.Intel[d.ID] {
				v = g.htFact(j, d.ID)
			}
			sc.TextB(x, y, fmt.Sprintf("  %-16s %s", d.Label, v), CGray, CBgDark, x+w)
			y++
		}
	} else if st.tab == 1 {
		sc.TextB(x, y, "{{K|Pick how you go in. Intel and gear open more ways.}}", CGray, CBgDark, x+w)
		y++
	} else if st.tab == 3 {
		for _, l := range Wrap("Begin the job while standing in a district of "+g.htBurgName(j.Burg)+". A building is raised in the nearest open ground and you start by your approach. Escape through the yard grate (sewer), the rope (roof) or on foot out of sight.", w) {
			sc.TextB(x, y, l, CGray, CBgDark, x+w)
			y++
		}
		y = g.htDrawKnowledge(j, x, y, w, bottom)
	}
	y++
	rows := g.htBuildRows()
	if st.sel >= len(rows) {
		st.sel = maxi(0, len(rows)-1)
	}
	for i, r := range rows {
		if y >= bottom-2 {
			break
		}
		mark := "  "
		if i == st.sel {
			mark = "> "
		}
		txt := r.Text
		if r.Dim {
			txt = "{{K|" + stripMarkup(r.Text) + "}}"
		}
		for k, l := range Wrap(mark+txt, w) {
			if k > 0 {
				l = "  " + l
			}
			if y < bottom-2 {
				sc.TextB(x, y, l, CGray, CBgDark, x+w)
				y++
			}
		}
	}
}

func stripMarkup(s string) string {
	for {
		i := strings.Index(s, "{{")
		if i < 0 {
			return s
		}
		k := strings.Index(s[i:], "|")
		if k < 0 {
			return s
		}
		s = s[:i] + s[i+k+1:]
		s = strings.Replace(s, "}}", "", 1)
	}
}

func (g *Game) htDrawKnowledge(j *htJob, x, y, w, bottom int) int {
	n := g.htSecretCount(j)
	if n > 0 {
		g.sc.TextB(x, y, fmt.Sprintf("{{Y|Whispers of %d hidden way(s) in the zone's walls.}}", n), CGray, CBgDark, x+w)
		y++
	}
	return y
}

func (g *Game) htFact(j *htJob, id string) string {
	switch id {
	case "shifts":
		return fmt.Sprintf("%d guards, watch changes at %02d:00", j.GuardN, j.ShiftHr)
	case "layout":
		return "hall, vault behind a locked door, yard grate"
	case "locks":
		return fmt.Sprintf("grade %d vault lock", j.Tier)
	case "alarms":
		if j.Alarms {
			return "tripwires in the hall"
		}
		return "none"
	case "dogs":
		if j.Dogs {
			return "a guard dog"
		}
		return "no dogs"
	case "secret":
		return "a hollow panel in the vault wall and a cellar way for a tunnel"
	}
	return ""
}
