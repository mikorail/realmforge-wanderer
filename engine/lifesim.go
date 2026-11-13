package engine

// lifesim.go (B, prefix ls): what people do. A day-clock agent simulation over the shared lvState people
// (lives_types.go): work, friendships and feuds, courtship, marriage, births, careers, shops changing hands,
// crime and punishment, travel and migration, illness, plague, war, old age, death and inheritance, goals, secrets.
//
// Full daily detail for people within 6 parasangs of the player and for Realmforge notables ("s<id>"), within a
// per-call person-day budget. Everyone else is advanced by coarse expectation-based chunks (same rules, seeded
// rates) and caught up when the player approaches. Nothing here draws g.R: every roll comes from side rngs seeded by
// world seed + person key + day, so a run replays identically. State: lvSt(g) (people, news) plus g.ext("ls")
// (planet-local) for the bookkeeping that does not belong to a person.

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
)

const (
	lsBudget  = 1500 // person-days of full detail per call
	lsMaxDays = 120  // days simulated one by one per call; older gaps are coarse
	lsNearP   = 6    // parasangs around the player that get full detail
)

type lsSick struct {
	Until int    `json:"u"`
	Dis   string `json:"d,omitempty"`
}

type lsFeud struct {
	Stage int `json:"s"` // 0 insults, 1 brawl, 2 duel/killing
	Day   int `json:"d"`
}

type lsChg struct {
	Seq  int    `json:"q"`
	Kind string `json:"k"` // grow birth shop death wed come go crime misc
	Text string `json:"t"`
}

// lsState is the bookkeeping saved in g.ext("ls"): planet-local (see init).
type lsState struct {
	Init    bool              `json:"i,omitempty"`
	Last    map[string]int    `json:"la,omitempty"` // person key -> day they were last simulated
	EnsureD int               `json:"ed,omitempty"` // day lvEnsureTown last ran for the burgs around the player
	EnsureB int               `json:"eb,omitempty"`
	InBurg  int               `json:"ib,omitempty"` // burg the player stood in at the last call
	Sick    map[string]lsSick `json:"si,omitempty"`
	Mourn   map[string]int    `json:"mo,omitempty"` // no courting before this day
	Court   map[string]int    `json:"co,omitempty"` // "a|b" -> progress 0..100
	Wooing  map[string]string `json:"wo,omitempty"` // both directions
	Preg    map[string]int    `json:"pr,omitempty"` // mother -> due day
	Jail    map[string]int    `json:"ja,omitempty"` // person -> release day
	Mig     map[string]bool   `json:"mi,omitempty"` // settling at the destination
	From    map[string]int    `json:"fr,omitempty"` // burg left on the current journey
	Feud    map[string]*lsFeud
	Avenge  map[string]string `json:"av,omitempty"` // victim -> killer (kin seek revenge)
	Crim    map[string]int    `json:"cr,omitempty"` // criminals at large -> day of the crime
	Rec     map[string]int    `json:"re,omitempty"` // crimes on record
	GoalD   map[string]int    `json:"gd,omitempty"` // day the current goal began
	Vacant  map[int][]string  `json:"va,omitempty"` // burg -> shop trades without an owner
	Chg     map[int][]lsChg   `json:"ch,omitempty"`
	Seq     int               `json:"sq,omitempty"`
	Seen    map[int]int       `json:"se,omitempty"` // burg -> change seq the player has been told up to
	Notes   map[string]bool   `json:"no,omitempty"`

	cacheW *WorldData
	near   map[int][]int
}

func init() {
	unLocalKeys = append(unLocalKeys, "ls")
	turnHooks = append(turnHooks, lsTurn)
	enterHooks = append(enterHooks, lsEnter)
	killHooks = append(killHooks, lsKillHook)
}

func lsSt(g *Game) *lsState {
	s := g.ext("ls", func() any { return &lsState{} }).(*lsState)
	if s.Last == nil {
		s.Last = map[string]int{}
		s.Sick = map[string]lsSick{}
		s.Mourn = map[string]int{}
		s.Court = map[string]int{}
		s.Wooing = map[string]string{}
		s.Preg = map[string]int{}
		s.Jail = map[string]int{}
		s.Mig = map[string]bool{}
		s.From = map[string]int{}
		s.Feud = map[string]*lsFeud{}
		s.Avenge = map[string]string{}
		s.Crim = map[string]int{}
		s.Rec = map[string]int{}
		s.GoalD = map[string]int{}
		s.Vacant = map[int][]string{}
		s.Chg = map[int][]lsChg{}
		s.Seen = map[int]int{}
		s.Notes = map[string]bool{}
	}
	// maps may be missing after loading a save that omitted empty ones
	if s.Sick == nil {
		s.Sick = map[string]lsSick{}
	}
	if s.Mourn == nil {
		s.Mourn = map[string]int{}
	}
	if s.Court == nil {
		s.Court = map[string]int{}
	}
	if s.Wooing == nil {
		s.Wooing = map[string]string{}
	}
	if s.Preg == nil {
		s.Preg = map[string]int{}
	}
	if s.Jail == nil {
		s.Jail = map[string]int{}
	}
	if s.Mig == nil {
		s.Mig = map[string]bool{}
	}
	if s.From == nil {
		s.From = map[string]int{}
	}
	if s.Feud == nil {
		s.Feud = map[string]*lsFeud{}
	}
	if s.Avenge == nil {
		s.Avenge = map[string]string{}
	}
	if s.Crim == nil {
		s.Crim = map[string]int{}
	}
	if s.Rec == nil {
		s.Rec = map[string]int{}
	}
	if s.GoalD == nil {
		s.GoalD = map[string]int{}
	}
	if s.Vacant == nil {
		s.Vacant = map[int][]string{}
	}
	if s.Chg == nil {
		s.Chg = map[int][]lsChg{}
	}
	if s.Seen == nil {
		s.Seen = map[int]int{}
	}
	if s.Notes == nil {
		s.Notes = map[string]bool{}
	}
	if s.Last == nil {
		s.Last = map[string]int{}
	}
	return s
}

// lsCtx is one call's working set.
type lsCtx struct {
	g      *Game
	st     *lvState
	ls     *lsState
	day    int
	yr     int
	seed   string
	all    []string // sorted person keys (newborns appended)
	byBurg map[int][]*lvPerson
	pop    map[int]int // living people by home burg
	shops  map[int]int
	feast  map[int]bool
	wars   map[int][]int
	pP     int // player parasang
	pBurg  int // player burg (0 = none)
}

func lsMakeCtx(g *Game) *lsCtx {
	c := &lsCtx{g: g, st: lvSt(g), ls: lsSt(g), seed: g.W.Seed}
	c.day = lvDay(g)
	c.yr = g.W.Year + c.day/YearDays
	c.byBurg, c.pop, c.shops = map[int][]*lvPerson{}, map[int]int{}, map[int]int{}
	c.feast, c.wars = map[int]bool{}, map[int][]int{}
	c.pP, c.pBurg = lsPlayerLoc(g)
	c.all = make([]string, 0, len(c.st.People))
	for k := range c.st.People {
		c.all = append(c.all, k)
	}
	sort.Strings(c.all)
	return c
}

func (c *lsCtx) setDay(day int) {
	c.day = day
	c.yr = c.g.W.Year + day/YearDays
}

func (c *lsCtx) rng(key string) *Rng {
	return NewRng(c.seed + ":ls:" + key + ":" + strconv.Itoa(c.day))
}

func (c *lsCtx) rebuild() {
	for k := range c.byBurg {
		delete(c.byBurg, k)
	}
	for k := range c.pop {
		delete(c.pop, k)
	}
	for k := range c.shops {
		delete(c.shops, k)
	}
	for k := range c.feast {
		delete(c.feast, k)
	}
	for _, k := range c.all {
		p := c.st.People[k]
		if p == nil || !p.Alive {
			continue
		}
		if p.Burg > 0 {
			c.byBurg[p.Burg] = append(c.byBurg[p.Burg], p)
		}
		c.pop[p.Home]++
		if lsShopJobs[p.Job] {
			c.shops[p.Home]++
		}
	}
}

func lsPlayerLoc(g *Game) (p, burg int) {
	w := g.W
	wx, wy := g.WX, g.WY
	if g.Cur != nil && !g.OnWorld {
		wx, wy = g.Cur.WX, g.Cur.WY
		if g.Cur.Kind == "town" && g.Cur.Burg > 0 {
			burg = g.Cur.Burg
		}
	}
	if w.In(wx, wy) {
		p = w.Idx(wx, wy)
	}
	return
}

func (c *lsCtx) burg(id int) *BurgData { return c.g.W.burgByID[id] }

func (c *lsCtx) burgName(id int) string {
	if b := c.burg(id); b != nil {
		return b.Name
	}
	return "the road"
}

func (c *lsCtx) capOf(burg int) int {
	b := c.burg(burg)
	if b == nil {
		return 12
	}
	return clampi(12+int(math.Sqrt(math.Max(0, b.Pop))*3), 12, 70)
}

// lsLoc: the parasang a person is at (burg, destination or home), -1 when unknown.
func (c *lsCtx) loc(p *lvPerson) int {
	for _, id := range [3]int{p.Burg, p.Dest, p.Home} {
		if id > 0 {
			if b := c.burg(id); b != nil {
				return b.P
			}
		}
	}
	return -1
}

func lsCheb(w *WorldData, a, b int) int {
	if a < 0 || b < 0 {
		return 999
	}
	return wsCheb(w, a, b)
}

func (c *lsCtx) feastAt(burg int) bool {
	if v, ok := c.feast[burg]; ok {
		return v
	}
	v := false
	if b := c.burg(burg); b != nil && b.Alive {
		v = c.g.W.FeastOn(b, c.day) != nil
	}
	c.feast[burg] = v
	return v
}

func (c *lsCtx) warsOf(state int) []int {
	if v, ok := c.wars[state]; ok {
		return v
	}
	v := c.g.warsOf(state)
	c.wars[state] = v
	return v
}

// nearBurgs: alive surface burgs within 12 parasangs of a burg, nearest first (at most 8). Cached per world.
func (c *lsCtx) nearBurgs(burg int) []int {
	ls := c.ls
	if ls.cacheW != c.g.W || ls.near == nil {
		ls.cacheW, ls.near = c.g.W, map[int][]int{}
	}
	if v, ok := ls.near[burg]; ok {
		return v
	}
	b := c.burg(burg)
	var out []int
	if b != nil {
		type cand struct{ id, d int }
		var cs []cand
		for _, o := range c.g.ws().burgList(c.g) {
			if o.ID == burg || o.Layer != b.Layer {
				continue
			}
			if d := wsCheb(c.g.W, b.P, o.P); d <= 12 {
				cs = append(cs, cand{o.ID, d})
			}
		}
		sort.Slice(cs, func(i, j int) bool {
			if cs[i].d != cs[j].d {
				return cs[i].d < cs[j].d
			}
			return cs[i].id < cs[j].id
		})
		for i := 0; i < len(cs) && i < 8; i++ {
			out = append(out, cs[i].id)
		}
	}
	ls.near[burg] = out
	return out
}

// lsDays: days to walk between two burgs (road is faster).
func (c *lsCtx) days(a, b *BurgData) int {
	d := wsCheb(c.g.W, a.P, b.P)
	pace := 28.0
	w := c.g.W
	if a.P < len(w.Road) && b.P < len(w.Road) && w.Road[a.P] > 0 && w.Road[b.P] > 0 {
		pace = 40
	}
	return maxi(1, int(math.Ceil(float64(d)*c.g.tvKm()/pace)))
}

// ---- entry points ----

func lsTurn(g *Game) {
	if lvDay(g) > lvSt(g).Day || (lsSt(g).EnsureD+10 <= lvDay(g)) {
		lsAdvance(g)
	}
}

func lsEnter(g *Game, zn *Zone) {
	if zn == nil || g.W == nil {
		return
	}
	lsEnsureNear(g, true)
	lsAdvance(g)
	if zn.Kind == "town" && zn.Burg > 0 && !g.OnWorld {
		for _, l := range g.lsChangeLines(zn.Burg) {
			g.Msg("{{c|" + l + "}}")
		}
	}
}

func lsKillHook(g *Game, d, src *Actor) {
	if d == nil || src == nil || src != g.P || d.Role == "" {
		return
	}
	if p := lvPersonOf(g, d); p != nil && p.Alive {
		g.lsKill(p.Key, "slain by a stranger")
	}
}

// lsKill ends a person's life from outside the simulation (the player killed them, a script did): inheritance,
// mourning, shops and news follow as for any death.
func (g *Game) lsKill(key, cause string) {
	p := lvGet(g, key)
	if p == nil || !p.Alive || g.W == nil {
		return
	}
	c := lsMakeCtx(g)
	c.rebuild()
	lsDie(c, p, cause, 0.9, c.rng(key+":kill"), nil)
}

// lsEnsureNear calls lvEnsureTown for the player's burg and the burgs within lsNearP parasangs (every ~10 days).
func lsEnsureNear(g *Game, force bool) {
	ls := lsSt(g)
	day := lvDay(g)
	pp, pb := lsPlayerLoc(g)
	if !force && ls.EnsureD > 0 && day-ls.EnsureD < 10 && pb == ls.EnsureB {
		return
	}
	if g.W == nil || pp < 0 {
		return
	}
	ls.EnsureD, ls.EnsureB = day, pb
	if pb > 0 {
		lvEnsureTown(g, pb)
	}
	if g.OnWorld || pb > 0 || g.Cur != nil {
		for _, b := range g.ws().burgList(g) {
			if b.ID != pb && wsCheb(g.W, b.P, pp) <= lsNearP {
				lvEnsureTown(g, b.ID)
			}
		}
	}
}

// lsAdvance runs the simulation up to today's game day (once per day; catch-up on arrival).
func lsAdvance(g *Game) {
	if g.W == nil {
		return
	}
	st := lvSt(g)
	ls := lsSt(g)
	today := lvDay(g)
	lsEnsureNear(g, false)
	if !ls.Init {
		ls.Init = true
		if st.Day < today && len(st.People) == 0 {
			st.Day = today
		}
	}
	if today <= st.Day {
		return
	}
	if len(st.People) == 0 {
		st.Day = today
		return
	}
	c := lsMakeCtx(g)
	prev := st.Day
	start := maxi(prev, today-lsMaxDays) // days after start are simulated one by one
	ndays := today - start
	for _, k := range c.all {
		if _, ok := ls.Last[k]; !ok {
			ls.Last[k] = prev
		}
	}
	// who gets full detail: nearest to the player and notables, within the budget
	type dp struct {
		k string
		d int
	}
	var dl []dp
	for _, k := range c.all {
		p := st.People[k]
		if p == nil || !p.Alive {
			continue
		}
		d := lsCheb(g.W, c.loc(p), c.pP)
		if d <= lsNearP || strings.HasPrefix(k, "s") {
			if strings.HasPrefix(k, "s") && d > lsNearP {
				d = lsNearP // notables rank behind the very nearest only by key
			}
			dl = append(dl, dp{k, d})
		}
	}
	sort.SliceStable(dl, func(i, j int) bool {
		if dl[i].d != dl[j].d {
			return dl[i].d < dl[j].d
		}
		return dl[i].k < dl[j].k
	})
	maxDetail := maxi(20, lsBudget/maxi(1, ndays))
	if len(dl) > maxDetail {
		dl = dl[:maxDetail]
	}
	detail := make(map[string]bool, len(dl))
	for _, e := range dl {
		detail[e.k] = true
	}
	c.setDay(start)
	c.rebuild()
	// coarse catch-up
	for _, k := range c.all {
		p := st.People[k]
		if p == nil || !p.Alive {
			continue
		}
		last := ls.Last[k]
		if detail[k] {
			if last < start {
				lsCoarse(c, p, start-last)
				ls.Last[k] = start
			}
		} else if today-last >= 7 {
			lsCoarse(c, p, today-last)
			ls.Last[k] = today
		}
	}
	// detail days
	for day := start + 1; day <= today; day++ {
		c.setDay(day)
		c.rebuild()
		for _, e := range dl {
			p := st.People[e.k]
			if p == nil || !p.Alive {
				continue
			}
			lsPersonDay(c, p)
		}
		lsBurgDay(c)
		for _, e := range dl {
			ls.Last[e.k] = day
		}
		st.Day = day
	}
	st.Day = today
	if ls.InBurg == c.pBurg && c.pBurg > 0 {
		if _, ok := ls.Seen[c.pBurg]; ok {
			ls.Seen[c.pBurg] = ls.Seq
		}
	}
	ls.InBurg = c.pBurg
}

// lsBurgDay: per-day burg-level upkeep (vacant shops, outbreaks announced once).
func lsBurgDay(c *lsCtx) {
	if c.day%10 != 0 {
		return
	}
	ls := c.ls
	for id, vs := range ls.Vacant {
		if len(vs) == 0 {
			delete(ls.Vacant, id)
			continue
		}
		// after a season nobody has bought it: the trade is lost
		if c.day%56 == 0 && len(vs) > 0 {
			ls.Vacant[id] = vs[1:]
		}
	}
}

// ---- news ----

func lsIsNotable(p *lvPerson) bool { return strings.HasPrefix(p.Key, "s") }

// lsNews adds a news line: sure for notables, with chance for everyone else.
func lsNews(c *lsCtx, p *lvPerson, burg int, chance float64, r *Rng, format string, a ...any) {
	if lsIsNotable(p) || (r != nil && r.Chance(chance)) {
		c.st.addNews(c.day, burg, format, a...)
	}
}

func lsChange(c *lsCtx, burg int, kind, format string, a ...any) {
	if burg <= 0 {
		return
	}
	ls := c.ls
	ls.Seq++
	l := append(ls.Chg[burg], lsChg{Seq: ls.Seq, Kind: kind, Text: fmt.Sprintf(format, a...)})
	if len(l) > 24 {
		l = l[len(l)-24:]
	}
	ls.Chg[burg] = l
}

func lsHer(p *lvPerson) string {
	if p.Female {
		return "her"
	}
	return "his"
}

func lsJobName(p *lvPerson) string {
	if p.Job != "" && p.Job != "retired" {
		return p.Job
	}
	if p.Role != "" && p.Role != "townsperson" && p.Role != "child" {
		return p.Role
	}
	return "townsperson"
}

// lsNewsLines: the sim's recent happenings as world-news lines (report: wsNewsLines hook).
func (g *Game) lsNewsLines() []wsLine {
	if g.W == nil {
		return nil
	}
	lsAdvance(g)
	st := lvSt(g)
	today := lvDay(g)
	here := 0
	if g.W.In(g.WX, g.WY) {
		here = g.W.Idx(g.WX, g.WY)
	}
	var out []wsLine
	for i := len(st.News) - 1; i >= 0 && len(out) < 14; i-- {
		n := st.News[i]
		if today-n.Day > 40 {
			break
		}
		p := here
		if b := g.W.burgByID[n.Burg]; b != nil {
			p = b.P
		}
		out = append(out, wsLine{P: p, Text: n.Text})
	}
	return out
}

// lsChangeLines: what changed in a town since the player last saw it (once per visit).
func (g *Game) lsChangeLines(burg int) []string {
	if g.W == nil || burg <= 0 {
		return nil
	}
	ls := lsSt(g)
	last, seen := ls.Seen[burg]
	ls.Seen[burg] = ls.Seq
	if !seen {
		return nil
	}
	var grow, birth int
	var out []string
	for _, ch := range ls.Chg[burg] {
		if ch.Seq <= last {
			continue
		}
		switch ch.Kind {
		case "grow":
			grow++
		case "birth":
			birth++
		default:
			out = append(out, ch.Text)
		}
	}
	nums := []string{"No", "A", "Two", "Three", "Four", "Five", "Six", "Seven"}
	num := func(n int) string {
		if n < len(nums) {
			return nums[n]
		}
		return strconv.Itoa(n)
	}
	if grow == 1 {
		out = append(out, "A child has grown up.")
	} else if grow > 1 {
		out = append(out, num(grow)+" children have grown up.")
	}
	if birth == 1 {
		out = append(out, "A baby was born.")
	} else if birth > 1 {
		out = append(out, num(birth)+" babies were born.")
	}
	if len(out) > 6 {
		out = out[len(out)-6:]
	}
	return out
}

// ---- helpers on people ----

func lsHas(l []string, k string) bool {
	for _, x := range l {
		if x == k {
			return true
		}
	}
	return false
}

func lsAdd(l []string, k string, max int) []string {
	if lsHas(l, k) {
		return l
	}
	l = append(l, k)
	if len(l) > max {
		l = l[len(l)-max:]
	}
	return l
}

func lsDel(l []string, k string) []string {
	for i, x := range l {
		if x == k {
			out := make([]string, 0, len(l)-1)
			out = append(out, l[:i]...)
			return append(out, l[i+1:]...)
		}
	}
	return l
}

func lsPair(a, b string) string {
	if a > b {
		a, b = b, a
	}
	return a + "|" + b
}

func lsChildren(g *Game, p *lvPerson) []*lvPerson {
	var out []*lvPerson
	ad := lsAdultAge(p.Species)
	for _, k := range p.Kin {
		q := lvGet(g, k)
		if q != nil && q.Born > p.Born && q.Born-p.Born >= ad-1 && lsHas(q.Kin, p.Key) {
			out = append(out, q)
		}
	}
	return out
}

func lsParents(g *Game, p *lvPerson) []*lvPerson {
	var out []*lvPerson
	for _, k := range p.Kin {
		q := lvGet(g, k)
		if q != nil && q.Born < p.Born && p.Born-q.Born >= lsAdultAge(q.Species)-1 && lsHas(q.Kin, p.Key) {
			out = append(out, q)
		}
	}
	return out
}

func lsFriend(a, b *lvPerson) {
	a.Friends = lsAdd(a.Friends, b.Key, 6)
	b.Friends = lsAdd(b.Friends, a.Key, 6)
	a.Rivals = lsDel(a.Rivals, b.Key)
	b.Rivals = lsDel(b.Rivals, a.Key)
}

func lsRival(c *lsCtx, a, b *lvPerson) {
	a.Rivals = lsAdd(a.Rivals, b.Key, 4)
	b.Rivals = lsAdd(b.Rivals, a.Key, 4)
	a.Friends = lsDel(a.Friends, b.Key)
	b.Friends = lsDel(b.Friends, a.Key)
	k := lsPair(a.Key, b.Key)
	if c.ls.Feud[k] == nil {
		c.ls.Feud[k] = &lsFeud{Day: c.day}
	}
}

func lsMood(p *lvPerson, d int) { p.Mood = clampi(p.Mood+d, -100, 100) }

func lsSoldiersIn(c *lsCtx, burg int) int {
	n := 0
	for _, q := range c.byBurg[burg] {
		if lsIsSoldier(q.Job) {
			n++
		}
	}
	return n
}

// ---- the day of one person ----

func lsPersonDay(c *lsCtx, p *lvPerson) {
	ls := c.ls
	r := c.rng(p.Key)
	age := p.lvAge(c.yr)
	life := lsLifespan(p.Species)
	adultAge := lsAdultAge(p.Species)
	if p.Burg <= 0 {
		lsRoad(c, p, r, age, life)
		return
	}
	if lsHealth(c, p, r, age, life) {
		return
	}
	if p.Role == "child" && age >= adultAge {
		lsGrow(c, p, r)
	}
	if age < adultAge {
		lsMoodStep(c, p, 1)
		return
	}
	_, sick := ls.Sick[p.Key]
	feast := c.feastAt(p.Burg)
	// needs and mood
	if feast {
		lsMood(p, 6)
		p.Wealth = maxi(0, p.Wealth-2)
		lsGoalBump(p, "fame", 2)
		lsGoalBump(p, "faith", 2)
	}
	lsMoodStep(c, p, 1)
	if sick {
		lsMood(p, -1)
		return
	}
	// work
	inc := lsIncome(p.Job)
	if inc > 0 {
		earn := inc
		if r.Chance(0.33) {
			earn++
		}
		p.Wealth = maxi(0, p.Wealth+earn-1-p.Wealth/250)
	} else {
		p.Wealth = maxi(0, p.Wealth-1)
	}
	if age >= life*3/4 && p.Job != "retired" && p.Job != "" && r.Chance(0.004) {
		lsRetire(c, p, r)
	} else if r.Chance(0.04) {
		lsCareer(c, p, r, age-adultAge)
	}
	// friends and rivals
	soc := 0.3
	if feast {
		soc = 0.7
	}
	if r.Chance(soc) {
		lsSocial(c, p, r)
	}
	lsFeudStep(c, p, r, age, life)
	if p.Burg <= 0 { // the feud ended in a jailing or exile
		return
	}
	lsCourtship(c, p, r, age, life, feast)
	lsPregnancy(c, p, r, age, life)
	lsCrime(c, p, r)
	if p.Burg <= 0 {
		return
	}
	lsTravel(c, p, r, age)
	if p.Burg > 0 && p.Dest == 0 {
		lsMigrate(c, p, r, age)
	}
	lsGoalStep(c, p, r, 1, age, life)
	lsSecret(c, p, r, 1)
}

// lsMoodStep: mood drifts toward a baseline set by neuroticism and mourning.
func lsMoodStep(c *lsCtx, p *lvPerson, n int) {
	base := 15 + (50-p.Trait[4])/4
	if c.ls.Mourn[p.Key] > c.day {
		base -= 30
	}
	d := (base - p.Mood) / 15
	if d == 0 && p.Mood != base {
		if base > p.Mood {
			d = 1
		} else {
			d = -1
		}
	}
	for i := 0; i < n && i < 6; i++ {
		p.Mood = clampi(p.Mood+d, -100, 100)
	}
}

// ---- health, plague, war, old age ----

type lsHaz struct {
	tot float64
	p   [4]float64
	txt [4]string
}

// lsHazard: daily chance of death from accident, old age, an outbreak at their burg, or war.
func lsHazard(c *lsCtx, p *lvPerson, age, life int) lsHaz {
	var h lsHaz
	h.p[0], h.txt[0] = 0.00006, "in an accident"
	if age >= life*7/10 {
		x := float64(age-life*7/10) / float64(maxi(1, life*3/10))
		if x > 1.6 {
			x = 1.6
		}
		h.p[1], h.txt[1] = (0.03+0.9*x*x)/YearDays, "of old age"
	}
	bid := p.Burg
	if bid <= 0 {
		bid = p.Home
	}
	if b := c.burg(bid); b != nil {
		if ob := c.g.W.hzOutbreakAt(b.P); ob != nil && b.Alive {
			h.p[2] = 0.0012
			if p.Wealth >= 150 {
				h.p[2] /= 2
			}
			h.txt[2] = "of " + lsOutbreakName(ob)
		}
		if lsIsSoldier(p.Job) {
			if ws := c.warsOf(b.State); len(ws) > 0 {
				h.p[3] = 0.0002
				if _, _, ok := c.g.frontierAt(b.P%c.g.W.W, b.P/c.g.W.W); ok {
					h.p[3] = 0.0012
				}
				h.txt[3] = "in the war against " + c.g.realmName(ws[0])
			}
		}
	}
	h.tot = h.p[0] + h.p[1] + h.p[2] + h.p[3]
	return h
}

func lsOutbreakName(ob *HzOutbreak) string {
	n := ob.Name
	if n == "" {
		n = "the plague"
	}
	if !strings.HasPrefix(strings.ToLower(n), "the ") {
		n = "the " + n
	}
	return n
}

func (h lsHaz) cause(r *Rng) string {
	x := r.Float() * h.tot
	for i := 0; i < 4; i++ {
		if x < h.p[i] {
			return h.txt[i]
		}
		x -= h.p[i]
	}
	return h.txt[1]
}

// lsHealth: sickness, plague and the daily hazards. Reports whether the person died.
func lsHealth(c *lsCtx, p *lvPerson, r *Rng, age, life int) bool {
	ls := c.ls
	h := lsHazard(c, p, age, life)
	if s, ok := ls.Sick[p.Key]; ok {
		if c.day >= s.Until {
			delete(ls.Sick, p.Key)
			lsMood(p, 5)
		} else {
			d := 0.003
			if age < lsAdultAge(p.Species) || age >= life*7/10 {
				d = 0.01
			}
			if s.Dis != "" {
				d = 0.008
				if age >= life*7/10 {
					d = 0.02
				}
			}
			if p.Wealth >= 150 {
				d /= 2
			}
			if r.Chance(d) {
				cause := "of a fever"
				if s.Dis != "" {
					cause = "of " + s.Dis
				} else if r.Chance(0.5) {
					cause = lsDutyCauses[r.Int(0, len(lsDutyCauses)-1)]
				}
				lsDie(c, p, cause, 0.35, r, nil)
				return true
			}
		}
	} else {
		pr := 0.004
		if age >= life*7/10 {
			pr = 0.008
		}
		dis := ""
		if h.p[2] > 0 {
			pr = 0.03
			dis = strings.TrimPrefix(h.txt[2], "of ")
		}
		if r.Chance(pr) {
			ls.Sick[p.Key] = lsSick{Until: c.day + r.Int(6, 20), Dis: dis}
			lsMood(p, -6)
			if dis != "" {
				key := "pl:" + strconv.Itoa(p.Burg) + ":" + dis
				if !ls.Notes[key] {
					ls.Notes[key] = true
					c.st.addNews(c.day, p.Burg, "%s has struck %s: the first are in their beds.", strings.ToUpper(dis[:1])+dis[1:], c.burgName(p.Burg))
					lsChange(c, p.Burg, "misc", "Sickness has come to the town: %s.", dis)
				}
			}
		}
	}
	if h.tot > 0 && r.Chance(h.tot) {
		lsDie(c, p, h.cause(r), 0.35, r, nil)
		return true
	}
	return false
}

// lsDie: the end of a life: records, inheritance, shop, mourning, vengeance, news.
func lsDie(c *lsCtx, p *lvPerson, cause string, newsChance float64, r *Rng, killer *lvPerson) {
	if !p.Alive {
		return
	}
	g, ls := c.g, c.ls
	age := p.lvAge(c.yr)
	burg := p.Burg
	if burg <= 0 {
		burg = p.Home
	}
	p.Alive = false
	p.Died = c.yr
	p.Cause = cause
	p.Dest, p.ArriveD = 0, 0
	p.addLife(c.yr, "death", "Died "+cause+".")
	delete(ls.Sick, p.Key)
	delete(ls.Jail, p.Key)
	delete(ls.Preg, p.Key)
	delete(ls.Mig, p.Key)
	delete(ls.Crim, p.Key)
	if w := ls.Wooing[p.Key]; w != "" {
		delete(ls.Wooing, w)
		delete(ls.Wooing, p.Key)
		delete(ls.Court, lsPair(p.Key, w))
	}
	if c.pop[p.Home] > 0 {
		c.pop[p.Home]--
	}
	// the dead leave their rivals' lists; killers fulfil revenge goals
	for _, k := range p.Rivals {
		if q := lvGet(g, k); q != nil && q.Alive {
			q.Rivals = lsDel(q.Rivals, p.Key)
			if q.GoalID == "revenge" {
				lsGoalDone(c, q, c.rng(q.Key+":rv"), "saw "+p.Name+" in "+lsPoss(p)+" grave")
			}
		}
	}
	for _, k := range p.Friends {
		if q := lvGet(g, k); q != nil && q.Alive {
			q.Friends = lsDel(q.Friends, p.Key)
			lsMood(q, -8)
		}
	}
	if kl := ls.Avenge[p.Key]; kl != "" && killer == nil {
		killer = lvGet(g, kl)
	}
	// spouse
	var spouse *lvPerson
	if p.Spouse != "" {
		if sp := lvGet(g, p.Spouse); sp != nil && sp.Alive {
			spouse = sp
			sp.Spouse = ""
			lsMood(sp, -35)
			ls.Mourn[sp.Key] = c.day + 70
			sp.addLife(c.yr, "death", "Lost "+p.Name+".")
		}
	}
	// kin: mourning, vengeance
	var heirs []*lvPerson
	for _, k := range p.Kin {
		q := lvGet(g, k)
		if q == nil || !q.Alive {
			continue
		}
		heirs = append(heirs, q)
		d := -8
		if lsHas(q.Kin, p.Key) && (q.Born > p.Born && q.Born-p.Born >= 12 || p.Born > q.Born && p.Born-q.Born >= 12) {
			d = -16
		}
		lsMood(q, d)
		ls.Mourn[q.Key] = maxi(ls.Mourn[q.Key], c.day+28)
		if killer != nil && killer.Key != q.Key && q.GoalID != "revenge" && d == -16 {
			q.Rivals = lsAdd(q.Rivals, killer.Key, 4)
			ls.Avenge[p.Key] = killer.Key
			q.GoalID, q.GoalPct = "revenge", 0
			q.Goal = "to avenge " + p.Name
			ls.GoalD[q.Key] = c.day
		}
	}
	// inheritance: half to the spouse (all if no kin), rest among children, else among kin
	if p.Wealth > 0 {
		kids := lsChildren(g, p)
		var kidsAlive []*lvPerson
		for _, q := range kids {
			if q.Alive {
				kidsAlive = append(kidsAlive, q)
			}
		}
		w := p.Wealth
		switch {
		case spouse != nil && len(kidsAlive) == 0:
			spouse.Wealth += w
		case spouse != nil:
			spouse.Wealth += w / 2
			w -= w / 2
			fallthrough
		case len(kidsAlive) > 0:
			for _, q := range kidsAlive {
				q.Wealth += w / len(kidsAlive)
			}
		case len(heirs) > 0:
			for _, q := range heirs {
				q.Wealth += w / len(heirs)
			}
		}
		p.Wealth = 0
	}
	if lsShopJobs[p.Job] {
		lsPassShop(c, p, true)
	}
	for _, q := range c.byBurg[p.Burg] { // employees lose their boss
		if q.Boss == p.Key {
			q.Boss = ""
		}
	}
	// news and visible change
	who := lsJobName(p)
	old := age >= lsLifespan(p.Species)*6/10
	if old {
		lsChange(c, burg, "death", "The old %s has died.", who)
	} else if p.Role != "child" {
		lsChange(c, burg, "death", "%s the %s has died.", p.Name, who)
	} else {
		lsChange(c, burg, "death", "A child of the town has died.")
	}
	lsNews(c, p, burg, newsChance, r, "%s, %s of %s, died %s.", p.Name, who, c.burgName(burg), cause)
	if killer != nil && killer.GoalID == "revenge" && lsHas(killer.Rivals, p.Key) {
		lsGoalDone(c, killer, c.rng(killer.Key+":rv"), "killed "+p.Name)
	}
}

func lsPoss(p *lvPerson) string {
	if p.Female {
		return "her"
	}
	return "his"
}

// lsPassShop: a shop owner dies or retires: child, spouse or employee takes it over, else it falls vacant.
func lsPassShop(c *lsCtx, p *lvPerson, dead bool) {
	g := c.g
	job := p.Job
	burg := p.Home
	var heir *lvPerson
	rel := ""
	free := func(q *lvPerson) bool { return q != nil && q.Alive && !lsShopJobs[q.Job] && q.Key != p.Key }
	age := func(q *lvPerson) int { return q.lvAge(c.yr) }
	kids := lsChildren(g, p)
	sort.Slice(kids, func(i, j int) bool { return kids[i].Born < kids[j].Born })
	for _, q := range kids {
		if free(q) && age(q) >= lsAdultAge(q.Species) && q.Home == burg {
			heir = q
			if q.Female {
				rel = "daughter"
			} else {
				rel = "son"
			}
			break
		}
	}
	if heir == nil {
		if sp := lvGet(g, p.Spouse); free(sp) && age(sp) >= lsAdultAge(sp.Species) && sp.Home == burg {
			heir, rel = sp, "husband"
			if sp.Female {
				rel = "wife"
			}
		}
	}
	if heir == nil {
		for _, q := range c.byBurg[burg] {
			if q.Boss == p.Key && free(q) && age(q) >= lsAdultAge(q.Species) {
				heir, rel = q, "former apprentice"
				break
			}
		}
	}
	if !dead {
		p.Job = "retired"
	}
	owner := "the late"
	if !dead {
		owner = "the"
	}
	if heir == nil {
		c.ls.Vacant[burg] = append(c.ls.Vacant[burg], job)
		if len(c.ls.Vacant[burg]) > 3 {
			c.ls.Vacant[burg] = c.ls.Vacant[burg][1:]
		}
		lsChange(c, burg, "shop", "The %s's shop stands shuttered.", job)
		return
	}
	heir.Job, heir.Boss = job, ""
	heir.addLife(c.yr, "job", "Took over the "+job+"'s shop from "+p.Name+".")
	lsMood(heir, 8)
	if dead {
		lsChange(c, burg, "shop", "The %s's shop now belongs to %s %s, %s.", job, lsHer(p), rel, heir.Name)
	} else {
		lsChange(c, burg, "shop", "The %s's shop now belongs to %s %s, %s.", job, lsHer(p), rel, heir.Name)
	}
	_ = owner
	lsNews(c, heir, burg, 0.3, c.rng(heir.Key+":shop"), "%s takes over the %s's shop in %s.", heir.Name, job, c.burgName(burg))
}

func lsRetire(c *lsCtx, p *lvPerson, r *Rng) {
	job := p.Job
	p.addLife(c.yr, "job", "Retired from work as a "+job+".")
	if lsShopJobs[job] {
		lsPassShop(c, p, false)
	} else {
		p.Job = "retired"
	}
	lsChange(c, p.Home, "misc", "The %s has retired.", job)
}

// ---- growing up, careers ----

func lsGrow(c *lsCtx, p *lvPerson, r *Rng) {
	g := c.g
	p.Role = "townsperson"
	p.Wealth += r.Int(5, 20)
	lsMood(p, 10)
	var parent *lvPerson
	for _, q := range lsParents(g, p) {
		if q.Alive && lsShopJobs[q.Job] && q.Home == p.Home {
			parent = q
			break
		}
	}
	if parent != nil && r.Chance(0.5) {
		p.Job, p.Boss = "apprentice", parent.Key
		p.addLife(c.yr, "job", "Became an apprentice to "+parent.Name+", the "+parent.Job+".")
	} else {
		jobs := lsEntryJobs
		if b := c.burg(p.Home); b != nil && b.Port {
			jobs = append(append([]string{}, jobs...), lsPortJobs...)
		}
		p.Job = jobs[r.Int(0, len(jobs)-1)]
		p.addLife(c.yr, "job", "Grew up and found work as a "+p.Job+".")
	}
	if p.GoalID == "" || p.GoalID == "grow" {
		lsNewGoal(c, p, r)
	}
	lsChange(c, p.Home, "grow", "A child has grown up.")
	lsNews(c, p, p.Home, 0.04, r, "%s comes of age in %s and takes work as a %s.", p.Name, c.burgName(p.Home), p.Job)
}

func lsCareer(c *lsCtx, p *lvPerson, r *Rng, yearsAdult int) {
	// a vacant shop for those with savings
	if vs := c.ls.Vacant[p.Home]; len(vs) > 0 && p.Wealth >= 100 && !lsShopJobs[p.Job] && p.Burg == p.Home && r.Chance(0.35) {
		job := vs[0]
		c.ls.Vacant[p.Home] = vs[1:]
		p.Wealth -= 80
		old := p.Job
		p.Job = job
		p.addLife(c.yr, "job", "Bought the vacant "+job+"'s shop, leaving work as a "+old+".")
		lsChange(c, p.Home, "shop", "A new %s, %s, has opened the old shop.", job, p.Name)
		lsNews(c, p, p.Home, 0.3, r, "%s buys the empty %s's shop in %s.", p.Name, job, c.burgName(p.Home))
		return
	}
	l, ok := lsLadder[p.Job]
	if !ok || l.next == p.Job || p.Wealth < l.wealth || yearsAdult < l.years || p.Trait[1] < l.con {
		return
	}
	next := l.next
	if next == "shopkeeper" {
		if c.shops[p.Home] >= maxi(2, c.capOf(p.Home)/4) {
			return
		}
		trades := []string{"baker", "smith", "tailor", "potter", "carpenter", "weaver", "cobbler", "brewer", "tanner", "butcher"}
		next = trades[r.Int(0, len(trades)-1)]
		p.Wealth -= 60
		c.shops[p.Home]++
	}
	old := p.Job
	p.Job = next
	p.Boss = ""
	lsMood(p, 8)
	p.addLife(c.yr, "job", fmt.Sprintf("Rose from %s to %s.", old, next))
	lsNews(c, p, p.Home, 0.08, r, "%s is now a %s in %s.", p.Name, next, c.burgName(p.Home))
	if lsShopJobs[next] {
		lsChange(c, p.Home, "shop", "%s has set up as a %s.", p.Name, next)
	}
}

// ---- friends and rivals ----

func lsSocial(c *lsCtx, p *lvPerson, r *Rng) {
	list := c.byBurg[p.Burg]
	if len(list) < 2 {
		return
	}
	q := list[r.Int(0, len(list)-1)]
	if q == p || !q.Alive || q.Burg != p.Burg || lsHas(p.Kin, q.Key) || q.lvAge(c.yr) < lsAdultAge(q.Species) {
		return
	}
	aff := 50 + (p.Trait[3]+q.Trait[3]-100)/4 - abs(p.Trait[0]-q.Trait[0])/4 - (p.Trait[4]+q.Trait[4]-100)/10 +
		(p.Trait[2]+q.Trait[2]-100)/10 + r.Int(-25, 25)
	if p.Faith == q.Faith && p.Faith != 0 {
		aff += 6
	}
	if p.Culture == q.Culture {
		aff += 4
	}
	switch {
	case lsHas(p.Friends, q.Key):
		if aff < 28 {
			p.Friends, q.Friends = lsDel(p.Friends, q.Key), lsDel(q.Friends, p.Key)
			p.addLife(c.yr, "friend", "Fell out with "+q.Name+".")
		} else {
			lsMood(p, 1)
		}
	case lsHas(p.Rivals, q.Key):
		if aff > 78 {
			p.Rivals, q.Rivals = lsDel(p.Rivals, q.Key), lsDel(q.Rivals, p.Key)
			delete(c.ls.Feud, lsPair(p.Key, q.Key))
			p.addLife(c.yr, "friend", "Made peace with "+q.Name+".")
			q.addLife(c.yr, "friend", "Made peace with "+p.Name+".")
		}
	case aff > 72 && len(p.Friends) < 6:
		lsFriend(p, q)
		p.addLife(c.yr, "friend", "Became friends with "+q.Name+".")
		q.addLife(c.yr, "friend", "Became friends with "+p.Name+".")
		lsMood(p, 4)
		lsMood(q, 4)
	case aff < 24 && len(p.Rivals) < 4:
		lsRival(c, p, q)
		p.addLife(c.yr, "feud", "Fell out with "+q.Name+".")
		q.addLife(c.yr, "feud", "Fell out with "+p.Name+".")
		lsMood(p, -4)
	}
}

// ---- feuds: insult, brawl, duel, killing ----

func lsFeudStep(c *lsCtx, p *lvPerson, r *Rng, age, life int) {
	if len(p.Rivals) == 0 {
		return
	}
	q := lvGet(c.g, p.Rivals[r.Int(0, len(p.Rivals)-1)])
	if q == nil || !q.Alive || q.Burg != p.Burg || q.lvAge(c.yr) < lsAdultAge(q.Species) {
		return
	}
	k := lsPair(p.Key, q.Key)
	f := c.ls.Feud[k]
	if f == nil {
		f = &lsFeud{Day: c.day}
		c.ls.Feud[k] = f
	}
	pressure := float64(100-p.Trait[3]+p.Trait[4]) / 100
	ch := 0.0025 * pressure
	if c.ls.Avenge[q.Key] != "" || c.ls.Avenge[p.Key] != "" || p.GoalID == "revenge" {
		ch *= 5
	}
	if f.Stage > 0 && c.day-f.Day > 40 { // cools off
		f.Stage--
		f.Day = c.day
	}
	if !r.Chance(ch) {
		return
	}
	f.Day = c.day
	notable := lsIsNotable(p) || lsIsNotable(q)
	burg := p.Burg
	switch f.Stage {
	case 0:
		f.Stage = 1
		lsMood(p, -5)
		lsMood(q, -5)
		p.addLife(c.yr, "feud", "Traded insults with "+q.Name+".")
		q.addLife(c.yr, "feud", "Traded insults with "+p.Name+".")
		if notable {
			c.st.addNews(c.day, burg, "%s and %s have fallen out loudly in %s.", p.Name, q.Name, c.burgName(burg))
		}
	case 1:
		f.Stage = 2
		lsMood(p, -8)
		lsMood(q, -8)
		p.addLife(c.yr, "feud", "Came to blows with "+q.Name+".")
		q.addLife(c.yr, "feud", "Came to blows with "+p.Name+".")
		if r.Chance(0.3) && lsSoldiersIn(c, burg) > 0 {
			fine := mini(15, p.Wealth)
			p.Wealth -= fine
			lsNews(c, p, burg, 0.2, r, "The watch of %s breaks up a brawl between %s and %s.", c.burgName(burg), p.Name, q.Name)
		} else {
			lsNews(c, p, burg, 0.12, r, "%s and %s brawl in the streets of %s.", p.Name, q.Name, c.burgName(burg))
		}
	default:
		murder := p.Trait[3] < 35 && p.Trait[4] > 55 && r.Chance(0.3)
		if murder || c.ls.Avenge[q.Key] != "" && r.Chance(0.4) {
			lsMurder(c, p, q, r)
			return
		}
		// a duel
		pw := 0.5
		if lsIsSoldier(p.Job) {
			pw += 0.15
		}
		if lsIsSoldier(q.Job) {
			pw -= 0.15
		}
		pw += float64(q.lvAge(c.yr)-age) / float64(life) * 0.3
		win, lose := p, q
		if !r.Chance(pw) {
			win, lose = q, p
		}
		lsGoalBump(win, "fame", 15)
		if r.Chance(0.35) {
			lsDie(c, lose, "in a duel with "+win.Name, 0.8, r, win)
			win.addLife(c.yr, "feud", "Killed "+lose.Name+" in a duel.")
			c.st.addNews(c.day, burg, "%s killed %s in a duel at %s.", win.Name, lose.Name, c.burgName(burg))
			delete(c.ls.Feud, k)
			if r.Chance(0.25) { // the duel was not entirely lawful
				lsPunish(c, win, "duelling", r, 0.4)
			}
		} else {
			lsMood(lose, -15)
			lsMood(win, 5)
			f.Stage = 1
			win.addLife(c.yr, "feud", "Beat "+lose.Name+" in a duel.")
			lose.addLife(c.yr, "feud", "Lost a duel to "+win.Name+".")
			lsNews(c, win, burg, 0.5, r, "%s bested %s in a duel at %s.", win.Name, lose.Name, c.burgName(burg))
		}
	}
}

func lsMurder(c *lsCtx, k, v *lvPerson, r *Rng) {
	burg := k.Burg
	lsDie(c, v, "murdered by "+k.Name, 0.95, r, k)
	k.addLife(c.yr, "crime", "Killed "+v.Name+".")
	c.ls.Avenge[v.Key] = k.Key
	delete(c.ls.Feud, lsPair(k.Key, v.Key))
	c.ls.Rec[k.Key] += 3
	c.st.addNews(c.day, burg, "%s was found dead in %s; %s is blamed.", v.Name, c.burgName(burg), k.Name)
	lsPunish(c, k, "murder", r, 0.5+0.08*float64(mini(3, lsSoldiersIn(c, burg))))
}

// ---- crime and punishment ----

func lsCrime(c *lsCtx, p *lvPerson, r *Rng) {
	if p.Trait[1] >= 40 || p.Trait[3] >= 45 || p.Wealth >= 60 || !r.Chance(0.006) {
		return
	}
	list := c.byBurg[p.Burg]
	var v *lvPerson
	for i := 0; i < 4 && len(list) > 0; i++ {
		q := list[r.Int(0, len(list)-1)]
		if q != p && q.Alive && q.Burg == p.Burg && q.Wealth > 80 && !lsHas(p.Kin, q.Key) {
			v = q
			break
		}
	}
	if v == nil {
		return
	}
	crime := "theft"
	take := mini(v.Wealth/4, 30)
	if p.Trait[4] > 60 && r.Chance(0.25) {
		crime = "assault"
		take = mini(take, 10)
		lsMood(v, -12)
	}
	v.Wealth -= take
	p.Wealth += take
	lsMood(v, -6)
	v.Rivals = lsAdd(v.Rivals, p.Key, 4)
	v.Friends = lsDel(v.Friends, p.Key)
	v.addLife(c.yr, "crime", "Was robbed by "+p.Name+".")
	if crime == "assault" {
		v.addLife(c.yr, "crime", "Was beaten by "+p.Name+".")
	}
	p.addLife(c.yr, "crime", "Committed "+crime+" against "+v.Name+".")
	c.ls.Rec[p.Key]++
	ch := 0.2 + 0.07*float64(mini(4, lsSoldiersIn(c, p.Burg)))
	if v.GoalID == "justice" {
		ch += 0.2
	}
	if r.Chance(ch) {
		p.Wealth += take // returned in full below
		lsPunish(c, p, crime, r, ch)
		v.Wealth += mini(take, p.Wealth)
		p.Wealth -= mini(take, p.Wealth)
	} else {
		c.ls.Crim[p.Key] = c.day
		lsNews(c, p, p.Burg, 0.25, r, "A %s was reported in %s: %s lost coin.", crime, c.burgName(p.Burg), v.Name)
	}
}

// lsPunish: a fine, days in jail or exile; murder may end on the gallows.
func lsPunish(c *lsCtx, p *lvPerson, crime string, r *Rng, caught float64) {
	if !p.Alive || p.Burg <= 0 {
		return
	}
	g := c.g
	delete(c.ls.Crim, p.Key)
	burg := p.Burg
	rec := c.ls.Rec[p.Key]
	switch {
	case crime == "murder" && r.Chance(0.5):
		lsDie(c, p, "on the gallows for murder", 1, r, nil)
	case crime == "murder" || rec >= 4:
		lsExile(c, p, r, crime)
	case rec >= 2 || crime == "assault" || crime == "duelling":
		days := r.Int(3, 20)
		p.Burg, p.Dest, p.ArriveD = 0, p.Home, c.day+days
		c.ls.Jail[p.Key] = p.ArriveD
		c.ls.From[p.Key] = burg
		lsMood(p, -12)
		p.addLife(c.yr, "crime", fmt.Sprintf("Jailed in %s for %s (%d days).", c.burgName(burg), crime, days))
		lsChange(c, burg, "crime", "%s was jailed for %s.", p.Name, crime)
		lsNews(c, p, burg, 0.5, r, "%s of %s is jailed for %s.", p.Name, c.burgName(burg), crime)
	default:
		fine := mini(p.Wealth, r.Int(8, 25))
		p.Wealth -= fine
		lsMood(p, -6)
		p.addLife(c.yr, "crime", fmt.Sprintf("Fined %d coin for %s.", fine, crime))
		lsNews(c, p, burg, 0.3, r, "%s is fined for %s in %s.", p.Name, crime, c.burgName(burg))
	}
	// the one who caught them may keep a goal
	for _, q := range c.byBurg[burg] {
		if q.GoalID == "justice" && q.Alive && q != p && r.Chance(0.3) {
			lsGoalDone(c, q, c.rng(q.Key+":jus"), "saw "+p.Name+" before the guards")
			break
		}
	}
	_ = g
}

func lsExile(c *lsCtx, p *lvPerson, r *Rng, why string) {
	burg := p.Burg
	near := c.nearBurgs(burg)
	if len(near) == 0 {
		lsDie(c, p, "on the gallows for "+why, 1, r, nil)
		return
	}
	dest := near[len(near)-1]
	b, d := c.burg(burg), c.burg(dest)
	if b == nil || d == nil {
		return
	}
	p.Burg, p.Dest = 0, dest
	p.ArriveD = c.day + c.days(b, d)
	c.ls.Mig[p.Key] = true
	c.ls.From[p.Key] = burg
	delete(c.ls.Jail, p.Key)
	lsMood(p, -20)
	p.addLife(c.yr, "crime", "Exiled from "+c.burgName(burg)+" for "+why+".")
	lsChange(c, burg, "crime", "%s was driven out of town for %s.", p.Name, why)
	lsNews(c, p, burg, 0.8, r, "%s is exiled from %s for %s.", p.Name, c.burgName(burg), why)
}

// ---- courtship, marriage, births ----

func lsTier(w int) int {
	switch {
	case w < 30:
		return 0
	case w < 150:
		return 1
	case w < 600:
		return 2
	}
	return 3
}

func lsEligible(c *lsCtx, p, q *lvPerson, pAge int) bool {
	if q == p || !q.Alive || q.Spouse != "" || q.Burg != p.Burg || p.Species != q.Species {
		return false
	}
	life := lsLifespan(p.Species)
	qAge := q.lvAge(c.yr)
	if qAge < lsAdultAge(q.Species) || qAge >= life*8/10 || c.ls.Mourn[q.Key] > c.day || c.ls.Wooing[q.Key] != "" {
		return false
	}
	if abs(qAge-pAge) > maxi(8, life/5) || lsHas(p.Kin, q.Key) || lsHas(q.Kin, p.Key) {
		return false
	}
	if p.Female == q.Female && HashStr(lsPair(p.Key, q.Key))%10 != 0 {
		return false
	}
	if abs(lsTier(p.Wealth)-lsTier(q.Wealth)) > 1 || abs(p.Trait[3]-q.Trait[3]) > 45 || abs(p.Trait[0]-q.Trait[0]) > 55 {
		return false
	}
	return true
}

func lsCourtship(c *lsCtx, p *lvPerson, r *Rng, age, life int, feast bool) {
	ls := c.ls
	if p.Spouse != "" || age >= life*8/10 || ls.Mourn[p.Key] > c.day {
		return
	}
	if w := ls.Wooing[p.Key]; w != "" {
		q := lvGet(c.g, w)
		if q == nil || !q.Alive || q.Spouse != "" || q.Burg != p.Burg {
			delete(ls.Wooing, p.Key)
			if q != nil {
				delete(ls.Wooing, q.Key)
			}
			delete(ls.Court, lsPair(p.Key, w))
			return
		}
		if p.Key > q.Key { // one side advances the courtship per day
			return
		}
		k := lsPair(p.Key, q.Key)
		step := r.Int(2, 8) + (p.Trait[3]+q.Trait[3])/40
		if feast {
			step += 6
		}
		if r.Chance(0.15) && abs(p.Trait[3]-q.Trait[3]) > 30 {
			step = -10
		}
		ls.Court[k] += step
		lsGoalBump(p, "love", step/2)
		lsGoalBump(q, "love", step/2)
		if ls.Court[k] >= 100 {
			lsMarry(c, p, q, r, feast)
		} else if ls.Court[k] < 0 {
			delete(ls.Court, k)
			delete(ls.Wooing, p.Key)
			delete(ls.Wooing, q.Key)
			p.addLife(c.yr, "marriage", "Courtship with "+q.Name+" came to nothing.")
			lsMood(p, -6)
			lsMood(q, -6)
		}
		return
	}
	ch := 0.008
	if p.GoalID == "love" {
		ch = 0.02
	}
	if feast {
		ch *= 4
	}
	if !r.Chance(ch) {
		return
	}
	list := c.byBurg[p.Burg]
	for i := 0; i < 5 && len(list) > 0; i++ {
		q := list[r.Int(0, len(list)-1)]
		if lsEligible(c, p, q, age) {
			ls.Wooing[p.Key], ls.Wooing[q.Key] = q.Key, p.Key
			ls.Court[lsPair(p.Key, q.Key)] = 10
			return
		}
	}
}

func lsMarry(c *lsCtx, a, b *lvPerson, r *Rng, feast bool) {
	ls := c.ls
	delete(ls.Wooing, a.Key)
	delete(ls.Wooing, b.Key)
	delete(ls.Court, lsPair(a.Key, b.Key))
	a.Spouse, b.Spouse = b.Key, a.Key
	// the poorer moves to the richer one's household
	if a.Home != b.Home {
		if a.Wealth < b.Wealth {
			a.Home = b.Home
		} else {
			b.Home = a.Home
		}
	}
	lsMood(a, 25)
	lsMood(b, 25)
	a.Friends = lsAdd(a.Friends, b.Key, 6)
	b.Friends = lsAdd(b.Friends, a.Key, 6)
	a.addLife(c.yr, "marriage", "Married "+b.Name+".")
	b.addLife(c.yr, "marriage", "Married "+a.Name+".")
	if a.GoalID == "love" {
		lsGoalDone(c, a, c.rng(a.Key+":love"), "married "+b.Name)
	}
	if b.GoalID == "love" {
		lsGoalDone(c, b, c.rng(b.Key+":love"), "married "+a.Name)
	}
	where := c.burgName(a.Burg)
	lsChange(c, a.Burg, "wed", "%s and %s were wed.", a.Name, b.Name)
	if feast {
		lsNews(c, a, a.Burg, 0.7, r, "%s and %s were wed in %s on the feast day.", a.Name, b.Name, where)
	} else {
		lsNews(c, a, a.Burg, 0.5, r, "%s and %s were wed in %s.", a.Name, b.Name, where)
	}
}

// lsPregnancy: conception and birth for a married woman (once a day).
func lsPregnancy(c *lsCtx, p *lvPerson, r *Rng, age, life int) {
	if !p.Female || p.Spouse == "" {
		return
	}
	ls := c.ls
	if due, ok := ls.Preg[p.Key]; ok {
		if c.day < due {
			return
		}
		delete(ls.Preg, p.Key)
		dad := lvGet(c.g, p.Spouse)
		if dad == nil || !dad.Alive {
			dad = nil
		}
		lsBirth(c, p, dad, r)
		return
	}
	if age < lsAdultAge(p.Species) || age > life/2 {
		return
	}
	sp := lvGet(c.g, p.Spouse)
	if sp == nil || !sp.Alive || sp.Female == p.Female || sp.Burg != p.Burg {
		return
	}
	if lsBirthChance(c, p) && r.Chance(lsConceive(c, p)/YearDays) {
		ls.Preg[p.Key] = c.day + 84
	}
}

func lsConceive(c *lsCtx, p *lvPerson) float64 {
	life := float64(lsLifespan(p.Species))
	rate := 0.35 * math.Min(2, 70/life)
	if rate < 0.05 {
		rate = 0.05
	}
	cp := float64(c.capOf(p.Home))
	n := float64(c.pop[p.Home])
	f := 1 - n/cp
	if f < 0 {
		f = 0
	}
	return rate * f * 1.6
}

func lsBirthChance(c *lsCtx, p *lvPerson) bool {
	if c.pop[p.Home] >= c.capOf(p.Home) {
		return false
	}
	return len(lsChildren(c.g, p)) < 6
}

// lsBirth creates a child person.
func lsBirth(c *lsCtx, mom, dad *lvPerson, r *Rng) {
	g, st := c.g, c.st
	burg := mom.Home
	if r.Chance(0.012) {
		lsDie(c, mom, "in childbirth", 0.4, r, nil)
		return
	}
	if r.Chance(0.04) {
		mom.addLife(c.yr, "child", "Lost a child at birth.")
		lsMood(mom, -20)
		if dad != nil {
			lsMood(dad, -15)
		}
		return
	}
	st.Next++
	key := "g" + strconv.Itoa(burg) + ":" + strconv.Itoa(st.Next)
	rg := NewRng(c.seed + ":ls:" + key + ":birth")
	child := &lvPerson{Key: key, Role: "child", Species: mom.Species, Culture: mom.Culture, State: mom.State, Faith: mom.Faith,
		Female: rg.Chance(0.5), Born: c.yr, Alive: true, Home: burg, Burg: mom.Burg}
	if child.Burg <= 0 {
		child.Burg = burg
	}
	var cu *CultureData
	if b := c.burg(burg); b != nil {
		cu = g.W.cultByID[b.Culture]
	}
	if cu == nil {
		cu = g.W.cultByID[mom.Culture]
	}
	given := g.cultureName(&CultureData{Given: lsGivenOf(cu)}, rg)
	house := ""
	if dad != nil {
		house = lsLast(dad.Name)
	}
	if house == "" {
		house = lsLast(mom.Name)
	}
	child.Name = given
	if house != "" {
		child.Name += " " + house
	}
	for i := range child.Trait {
		base := mom.Trait[i]
		if dad != nil {
			base = (mom.Trait[i] + dad.Trait[i]) / 2
		}
		child.Trait[i] = clampi(base+rg.Int(-18, 18), 0, 100)
	}
	child.Mood = 20
	child.Goal, child.GoalID = "to grow up", "grow"
	st.People[key] = child
	c.all = append(c.all, key)
	c.ls.Last[key] = c.day
	c.pop[burg]++
	c.byBurg[child.Burg] = append(c.byBurg[child.Burg], child)
	// kin: parents, siblings
	parents := []*lvPerson{mom}
	if dad != nil {
		parents = append(parents, dad)
	}
	var sibs []*lvPerson
	for _, p := range parents {
		for _, q := range lsChildren(g, p) {
			if q != child && q.Alive || q != child && !q.Alive {
				dup := false
				for _, s := range sibs {
					if s == q {
						dup = true
					}
				}
				if !dup {
					sibs = append(sibs, q)
				}
			}
		}
	}
	for _, p := range parents {
		child.Kin = lsAdd(child.Kin, p.Key, 16)
		p.Kin = lsAdd(p.Kin, key, 16)
		lsMood(p, 15)
		lsGoalBump(p, "family", 40)
		if p.GoalID == "family" && p.GoalPct >= 100 {
			lsGoalDone(c, p, c.rng(p.Key+":fam"), "raised a family")
		}
	}
	for _, s := range sibs {
		child.Kin = lsAdd(child.Kin, s.Key, 16)
		s.Kin = lsAdd(s.Kin, key, 16)
	}
	child.addLife(c.yr, "birth", "Born in "+c.burgName(burg)+" to "+mom.Name+lsAnd(dad)+".")
	mom.addLife(c.yr, "child", "Gave birth to "+child.Name+".")
	if dad != nil {
		dad.addLife(c.yr, "child", "Became father of "+child.Name+".")
	}
	if lvKnowledgeOf != nil {
		child.Knows = lvKnowledgeOf(g, child)
	}
	lsChange(c, burg, "birth", "A baby was born.")
	lsNews(c, mom, burg, 0.12, r, "%s and %s of %s have a child, %s.", mom.Name, lsNameOr(dad), c.burgName(burg), child.Name)
}

func lsGivenOf(cu *CultureData) []string {
	if cu == nil {
		return nil
	}
	return cu.Given
}

func lsAnd(d *lvPerson) string {
	if d == nil {
		return ""
	}
	return " and " + d.Name
}

func lsNameOr(d *lvPerson) string {
	if d == nil {
		return "her husband"
	}
	return d.Name
}

func lsLast(name string) string {
	if i := strings.LastIndex(name, " "); i > 0 {
		return name[i+1:]
	}
	return ""
}

// ---- travel, roads, migration ----

func lsTravel(c *lsCtx, p *lvPerson, r *Rng, age int) {
	if p.Dest != 0 {
		return
	}
	pr := 0.002 + float64(p.Trait[0]+p.Trait[2])/200*0.004
	if lsIsTrader(p.Job) {
		pr += 0.012
	}
	away := p.Home != p.Burg && p.Home > 0
	if away {
		pr = 0.07
	}
	near := c.nearBurgs(p.Burg)
	if len(near) == 0 {
		return
	}
	festDest := 0
	if !away {
		for _, id := range near {
			if c.feastAt(id) {
				if d := c.burg(id); d != nil && c.days(c.burg(p.Burg), d) <= 5 {
					festDest = id
					pr += 0.02
					break
				}
			}
		}
	}
	if !r.Chance(pr) {
		return
	}
	b := c.burg(p.Burg)
	var dest int
	switch {
	case away:
		dest = p.Home
	case festDest != 0:
		dest = festDest
	default:
		dest = near[r.Int(0, mini(len(near), 4)-1)]
		if lsIsTrader(p.Job) && len(near) > 1 {
			if o := c.burg(near[r.Int(0, len(near)-1)]); o != nil && c.burg(dest) != nil && o.Pop > c.burg(dest).Pop {
				dest = o.ID
			}
		}
	}
	d := c.burg(dest)
	if b == nil || d == nil || !d.Alive {
		return
	}
	lsDepart(c, p, dest, c.days(b, d))
}

func lsDepart(c *lsCtx, p *lvPerson, dest, days int) {
	c.ls.From[p.Key] = p.Burg
	p.Burg, p.Dest, p.ArriveD = 0, dest, c.day+days
}

// lsRoad: a person on the road or in jail: arrival, release, bandits, weather.
func lsRoad(c *lsCtx, p *lvPerson, r *Rng, age, life int) {
	ls := c.ls
	_, jailed := ls.Jail[p.Key]
	if p.Dest == 0 && !jailed { // lost track: home again
		p.Burg = p.Home
		return
	}
	if c.day >= p.ArriveD {
		lsArrive(c, p, r)
		return
	}
	if jailed {
		return
	}
	if h := lsHazard(c, p, age, life); h.p[1] > 0 && r.Chance(h.p[1]) {
		lsDie(c, p, "of old age on the road", 0.6, r, nil)
		return
	}
	danger := 0.004
	from := c.burg(ls.From[p.Key])
	to := c.burg(p.Dest)
	if from != nil && to != nil {
		if from.Bandit {
			danger += 0.004
		}
		w := c.g.W
		mid := w.Idx((from.P%w.W+to.P%w.W)/2, (from.P/w.W+to.P/w.W)/2)
		for _, e := range c.g.ws().Ev {
			if e.Kind == "bandits" && wsCheb(w, e.P, mid) <= 4 && c.day*DayLen >= e.Start*60-DayLen {
				danger += 0.012
				break
			}
		}
	}
	if !r.Chance(danger) {
		return
	}
	switch x := r.Float(); {
	case x < 0.55:
		lose := p.Wealth / 2
		p.Wealth -= lose
		lsMood(p, -12)
		p.addLife(c.yr, "crime", fmt.Sprintf("Robbed on the road, losing %d coin.", lose))
		lsNews(c, p, p.Home, 0.15, r, "%s was robbed on the road to %s.", p.Name, c.burgName(p.Dest))
	case x < 0.7:
		lsDie(c, p, "slain by bandits on the road", 0.6, r, nil)
	default:
		p.ArriveD += r.Int(1, 3)
		lsMood(p, -3)
	}
}

func lsArrive(c *lsCtx, p *lvPerson, r *Rng) {
	ls := c.ls
	dest := p.Dest
	if d := c.burg(dest); d == nil || !d.Alive {
		dest = p.Home
	}
	_, jailed := ls.Jail[p.Key]
	p.Burg, p.Dest, p.ArriveD = dest, 0, 0
	delete(ls.Jail, p.Key)
	delete(ls.From, p.Key)
	if jailed {
		lsMood(p, -4)
		return
	}
	if ls.Mig[p.Key] {
		delete(ls.Mig, p.Key)
		old := p.Home
		p.Home = dest
		c.pop[dest]++
		if c.pop[old] > 0 {
			c.pop[old]--
		}
		p.addLife(c.yr, "move", "Moved to "+c.burgName(dest)+".")
		lsChange(c, dest, "come", "%s has come to live in town.", p.Name)
		lsNews(c, p, dest, 0.25, r, "%s settles in %s.", p.Name, c.burgName(dest))
		if p.GoalID == "safety" {
			lsGoalDone(c, p, r, "found a safer home in "+c.burgName(dest))
		}
		if lsShopJobs[p.Job] { // a shop does not move with its owner
			p.Job = "laborer"
		}
	}
}

// lsMigrate: poor people walk to a bigger, richer town; refugees leave a ruined one.
func lsMigrate(c *lsCtx, p *lvPerson, r *Rng, age int) {
	if p.Home != p.Burg || p.Home <= 0 {
		return
	}
	home := c.burg(p.Home)
	if home == nil {
		return
	}
	ruined := !home.Alive
	ch := 0.0
	switch {
	case ruined:
		ch = 0.06
	case p.Wealth < 20 && lsTier(p.Wealth) == 0:
		ch = 0.003
	}
	if p.GoalID == "safety" && p.GoalPct >= 60 {
		ch += 0.02
	}
	if ch == 0 || !r.Chance(ch) {
		return
	}
	var best *BurgData
	for _, id := range c.nearBurgs(p.Home) {
		d := c.burg(id)
		if d == nil || !d.Alive || c.pop[id] >= c.capOf(id) {
			continue
		}
		if ruined || d.Pop > home.Pop*1.25 || p.GoalID == "safety" {
			best = d
			break
		}
	}
	if best == nil {
		return
	}
	days := c.days(home, best)
	lsDepart(c, p, best.ID, days)
	c.ls.Mig[p.Key] = true
	lsChange(c, home.ID, "go", "%s has left for %s.", p.Name, best.Name)
	// spouse and children go too
	var fam []*lvPerson
	if sp := lvGet(c.g, p.Spouse); sp != nil {
		fam = append(fam, sp)
	}
	for _, q := range lsChildren(c.g, p) {
		if q.Role == "child" {
			fam = append(fam, q)
		}
	}
	for _, q := range fam {
		if q.Alive && q.Burg == home.ID && q.Dest == 0 {
			lsDepart(c, q, best.ID, days)
			c.ls.Mig[q.Key] = true
		}
	}
}

// ---- goals ----

func lsGoalBump(p *lvPerson, id string, n int) {
	if p.GoalID == id {
		p.GoalPct = clampi(p.GoalPct+n, 0, 100)
	}
}

func lsGoalText(c *lsCtx, p *lvPerson, id string) string {
	if id == "wealth" {
		return fmt.Sprintf(lsGoalLines[id], lsWealthTarget(p))
	}
	if id == "revenge" && len(p.Rivals) > 0 {
		if q := lvGet(c.g, p.Rivals[0]); q != nil {
			return "to be revenged on " + q.Name
		}
	}
	if t := lsGoalLines[id]; t != "" {
		return t
	}
	return "to get by"
}

func lsWealthTarget(p *lvPerson) int {
	return 250 + int(HashStr(p.Key)%8)*75 + p.Wealth/2
}

func lsNewGoal(c *lsCtx, p *lvPerson, r *Rng) {
	t := p.Trait
	score := map[string]int{
		"wealth":    t[1]/2 + (100-t[3])/3 + 10,
		"love":      0,
		"revenge":   0,
		"faith":     t[0]/3 + 10,
		"knowledge": t[0]/2 + t[1]/4,
		"fame":      t[2]/2 + t[0]/4,
		"justice":   t[1]/3 + t[3]/3,
		"family":    0,
		"safety":    t[4] / 2,
	}
	if p.Spouse == "" {
		score["love"] = 30 + t[2]/3
	} else if len(lsChildren(c.g, p)) < 2 {
		score["family"] = 45
	}
	if len(p.Rivals) > 0 {
		score["revenge"] = 20 + (100-t[3])/3
	}
	best, bs := "wealth", -1
	for _, id := range lsGoalIDs { // fixed order: deterministic ties
		if s := score[id] + r.Int(0, 30); s > bs {
			best, bs = id, s
		}
	}
	p.GoalID, p.GoalPct = best, 0
	p.Goal = lsGoalText(c, p, best)
	c.ls.GoalD[p.Key] = c.day
}

func lsGoalDone(c *lsCtx, p *lvPerson, r *Rng, what string) {
	if !p.Alive || p.GoalID == "" {
		return
	}
	id := p.GoalID
	lsMood(p, 20)
	p.addLife(c.yr, "goal", "Reached a long goal: "+what+".")
	burg := p.Burg
	if burg <= 0 {
		burg = p.Home
	}
	switch id {
	case "wealth":
		lsNews(c, p, burg, 0.2, r, "%s of %s has grown rich: %s.", p.Name, c.burgName(burg), what)
	case "faith":
		if p.Wealth >= 150 {
			p.Wealth -= 100
			lsChange(c, burg, "misc", "%s has raised a small shrine.", p.Name)
			lsNews(c, p, burg, 0.5, r, "%s founds a shrine in %s.", p.Name, c.burgName(burg))
		} else {
			if p.Job == "laborer" || p.Job == "farmhand" || p.Job == "servant" || p.Job == "" {
				p.Job = "acolyte"
			}
			lsNews(c, p, burg, 0.3, r, "%s takes vows in %s.", p.Name, c.burgName(burg))
		}
	case "knowledge":
		lsNews(c, p, burg, 0.6, r, "%s of %s finished a book of what they know.", p.Name, c.burgName(burg))
	case "fame":
		lsNews(c, p, burg, 0.5, r, "The name of %s is sung in the taverns of %s.", p.Name, c.burgName(burg))
	case "justice", "revenge":
		lsNews(c, p, burg, 0.4, r, "%s: %s.", p.Name, what)
	default:
		lsNews(c, p, burg, 0.15, r, "%s of %s: %s.", p.Name, c.burgName(burg), what)
	}
	p.GoalID = ""
	lsNewGoal(c, p, r)
}

func lsGoalFail(c *lsCtx, p *lvPerson, r *Rng, why string) {
	p.addLife(c.yr, "goal", "Gave up on a long hope: "+why+".")
	lsMood(p, -8)
	p.GoalID = ""
	lsNewGoal(c, p, r)
}

// lsGoalStep: progress, success or failure of the current goal (scale = days covered).
func lsGoalStep(c *lsCtx, p *lvPerson, r *Rng, scale, age, life int) {
	if p.GoalID == "" || p.GoalID == "grow" {
		lsNewGoal(c, p, r)
		return
	}
	ls := c.ls
	if ls.GoalD[p.Key] == 0 {
		ls.GoalD[p.Key] = c.day
	}
	d := scale
	switch p.GoalID {
	case "wealth":
		p.GoalPct = clampi(p.Wealth*100/lsWealthTarget(&lvPerson{Key: p.Key}), 0, 100)
		if p.GoalPct >= 100 {
			lsGoalDone(c, p, r, fmt.Sprintf("put by %d coin", p.Wealth))
			return
		}
	case "love":
		if p.Spouse != "" {
			lsGoalDone(c, p, r, "found a spouse")
			return
		}
		if age >= life*7/10 {
			lsGoalFail(c, p, r, "too late for love")
			return
		}
	case "revenge":
		if len(p.Rivals) == 0 {
			lsGoalFail(c, p, r, "there is nobody left to blame")
			return
		}
		p.GoalPct = clampi(p.GoalPct+d/3, 0, 90)
	case "faith":
		p.GoalPct = clampi(p.GoalPct+d/8+boolInt(p.Job == "acolyte" || p.Job == "priest")*d/8, 0, 100)
		if p.GoalPct >= 100 {
			lsGoalDone(c, p, r, "served the faith")
			return
		}
	case "knowledge":
		sch := boolInt(p.Job == "scholar" || p.Job == "scribe" || p.Job == "priest" || p.Job == "clerk")
		p.GoalPct = clampi(p.GoalPct+(d*(1+sch)*(10+p.Trait[0]/10))/80, 0, 100)
		if p.GoalPct >= 100 {
			lsGoalDone(c, p, r, "wrote a book")
			return
		}
	case "fame":
		p.GoalPct = clampi(p.GoalPct+d/10, 0, 100)
		if p.GoalPct >= 100 {
			lsGoalDone(c, p, r, "became known far and wide")
			return
		}
	case "justice":
		p.GoalPct = clampi(p.GoalPct+d/6, 0, 95)
		if p.GoalPct >= 30 && scale == 1 && p.Burg > 0 {
			for _, k := range lsCrimKeys(c) {
				cr := lvGet(c.g, k)
				if cr != nil && cr.Alive && cr.Burg == p.Burg && cr != p && r.Chance(0.03) {
					lsPunish(c, cr, "theft", r, 1)
					lsGoalDone(c, p, r, "brought "+cr.Name+" to the guards")
					return
				}
			}
		}
	case "family":
		n := len(lsChildren(c.g, p))
		p.GoalPct = clampi(n*34+boolInt(p.Spouse != "")*20, 0, 100)
		if p.GoalPct >= 100 {
			lsGoalDone(c, p, r, "raised a family")
			return
		}
		if age >= life/2 && n == 0 {
			lsGoalFail(c, p, r, "the years for children are past")
			return
		}
	case "safety":
		danger := false
		if b := c.burg(p.Home); b != nil {
			danger = !b.Alive || c.g.W.hzOutbreakAt(b.P) != nil || len(c.warsOf(b.State)) > 0
		}
		if danger {
			p.GoalPct = clampi(p.GoalPct+d, 0, 100)
		} else {
			p.GoalPct = clampi(p.GoalPct-d, 0, 100)
		}
	}
	if c.day-ls.GoalD[p.Key] > 560 && p.GoalPct < 50 && p.GoalID != "safety" {
		lsGoalFail(c, p, r, "the years went by")
	}
}

func lsCrimKeys(c *lsCtx) []string {
	out := make([]string, 0, len(c.ls.Crim))
	for k := range c.ls.Crim {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// ---- secrets ----

func lsSecret(c *lsCtx, p *lvPerson, r *Rng, scale int) {
	if p.Secret == "" {
		return
	}
	ch := 0.0004 * float64(1+len(p.Rivals)) * float64(scale)
	if !r.Chance(ch) {
		return
	}
	how := lsSecretsOut[r.Int(0, len(lsSecretsOut)-1)]
	burg := p.Burg
	if burg <= 0 {
		burg = p.Home
	}
	p.addLife(c.yr, "secret", "Their secret came out ("+how+").")
	p.Secret = ""
	lsMood(p, -15)
	if list := c.byBurg[burg]; len(list) > 1 {
		q := list[r.Int(0, len(list)-1)]
		if q != p && q.Alive && q.lvAge(c.yr) >= lsAdultAge(q.Species) && !lsHas(p.Kin, q.Key) {
			lsRival(c, p, q)
			q.addLife(c.yr, "secret", "Learned a secret of "+p.Name+".")
		}
	}
	lsNews(c, p, burg, 0.5, r, "A secret of %s is out in %s: %s.", p.Name, c.burgName(burg), how)
	lsChange(c, burg, "misc", "The town talks about %s.", p.Name)
}

// ---- coarse catch-up ----

// lsCoarse applies what would have happened to someone over `days` days by seeded expectation, in chunks.
func lsCoarse(c *lsCtx, p *lvPerson, days int) {
	if days <= 0 || !p.Alive {
		return
	}
	sz := 28
	if days > sz*40 {
		sz = days/40 + 1
	}
	endDay, endYr := c.day, c.yr
	defer func() { c.day, c.yr = endDay, endYr }()
	for rem := days; rem > 0 && p.Alive; rem -= sz {
		n := mini(sz, rem)
		c.setDay(endDay - rem + n)
		lsCoarseChunk(c, p, n)
	}
}

func lsCoarseChunk(c *lsCtx, p *lvPerson, n int) {
	ls := c.ls
	r := NewRng(c.seed + ":ls:" + p.Key + ":coarse:" + strconv.Itoa(c.day) + ":" + strconv.Itoa(n))
	age := p.lvAge(c.yr)
	life := lsLifespan(p.Species)
	adultAge := lsAdultAge(p.Species)
	// travel resolves
	if p.Burg <= 0 {
		if p.Dest == 0 {
			p.Burg = p.Home
		} else if c.day >= p.ArriveD {
			lsArrive(c, p, r)
		} else {
			return
		}
	}
	// death by the hazards
	h := lsHazard(c, p, age, life)
	delete(ls.Sick, p.Key)
	if h.tot > 0 && r.Chance(1-math.Pow(1-h.tot, float64(n))) {
		lsDie(c, p, h.cause(r), 0.3, r, nil)
		return
	}
	// a long illness would have come and gone, but the sick do not linger across a chunk
	if p.Role == "child" && age >= adultAge {
		lsGrow(c, p, r)
	}
	if age < adultAge {
		return
	}
	lsMoodStep(c, p, 6)
	inc := lsIncome(p.Job)
	p.Wealth = maxi(0, p.Wealth+(inc*4/3-1-p.Wealth/250)*n*8/10)
	if age >= life*3/4 && p.Job != "retired" && p.Job != "" && r.Chance(1-math.Pow(0.996, float64(n))) {
		lsRetire(c, p, r)
	} else if r.Chance(1 - math.Pow(0.96, float64(n))) {
		lsCareer(c, p, r, age-adultAge)
	}
	if p.Burg <= 0 {
		return
	}
	list := c.byBurg[p.Burg]
	if len(list) > 1 {
		// friendships and quarrels
		if r.Chance(math.Min(0.9, 0.3*float64(n))) {
			c.setDay(c.day)
			lsSocial(c, p, r)
		}
		// marriage
		if p.Spouse == "" && age < life*8/10 && c.ls.Mourn[p.Key] <= c.day && r.Chance(math.Min(0.5, 0.012*float64(n))) {
			start := r.Int(0, len(list)-1)
			for i := 0; i < len(list) && i < 12; i++ {
				q := list[(start+i)%len(list)]
				if q.lvAge(c.yr) >= adultAge && lsEligible(c, p, q, age) {
					lsMarry(c, p, q, r, false)
					break
				}
			}
		}
	}
	// births, spread over the chunk
	if p.Female && p.Spouse != "" && age <= life/2 {
		if sp := lvGet(c.g, p.Spouse); sp != nil && sp.Alive && sp.Burg == p.Burg && lsBirthChance(c, p) {
			if r.Chance(math.Min(0.9, lsConceive(c, p)*float64(n)/YearDays)) {
				delete(ls.Preg, p.Key)
				lsBirth(c, p, sp, r)
			}
		}
	}
	// a poor family moves on
	if p.Burg == p.Home && p.Dest == 0 && p.Wealth < 20 && r.Chance(math.Min(0.4, 0.003*float64(n))) {
		if home := c.burg(p.Home); home != nil {
			for _, id := range c.nearBurgs(p.Home) {
				d := c.burg(id)
				if d != nil && d.Alive && d.Pop > home.Pop*1.25 && c.pop[id] < c.capOf(id) {
					lsChange(c, home.ID, "go", "%s has left for %s.", p.Name, d.Name)
					p.Dest, p.ArriveD = id, c.day
					ls.Mig[p.Key] = true
					lsArrive(c, p, r)
					break
				}
			}
		}
	}
	lsGoalStep(c, p, r, n, age, life)
	lsSecret(c, p, r, n)
}
