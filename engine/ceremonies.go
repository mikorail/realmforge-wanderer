package engine

// Ceremonies of people you know (prefix fw): weddings, funerals, namings, comings of age and feast
// processions. A daily scan of the life simulation (people you have met or who like you) notices who
// married, died, had a child or grew up and schedules a rite a few days later in their burg (stored in
// g.ext("fw"), planet-local). When the day comes and you stand in a fitting district of that burg an
// enter/turn hook sets up the site (decor, the bound people, an officiant) with a side rng, and a choice
// screen (mode `fwc`) opens when you walk up to it. Everything feeds back through lvPerson.remember,
// life lines, lifesim news, deeds and reputation. Missing the rite of someone close costs a little
// Opinion with the kin. No g.R in generation; checks use the normal 2d6 roll.
//
// Optional source of invitations (letters): set fwInvites from the letters file.

import (
	"fmt"
	"sort"
	"strings"
)

const fwKey = "fw"

type fwSnap struct {
	D bool   `json:"d,omitempty"`
	S string `json:"s,omitempty"`
}

type fwDeco struct {
	X   int     `json:"x"`
	Y   int     `json:"y"`
	Old Terrain `json:"o"`
}

type fwCer struct {
	ID       string            `json:"i"`
	Kind     string            `json:"k"` // wedding funeral birth age feast
	Burg     int               `json:"b"`
	Day      int               `json:"d"`
	End      int               `json:"e"`
	Subj     []string          `json:"s,omitempty"`
	Told     bool              `json:"t,omitempty"`
	Hinted   bool              `json:"h,omitempty"`
	Attended bool              `json:"a,omitempty"`
	Done     bool              `json:"dn,omitempty"`
	Opened   bool              `json:"o,omitempty"`
	Clean    bool              `json:"cl,omitempty"`
	Sus      bool              `json:"su,omitempty"`
	Culp     string            `json:"cu,omitempty"`
	Inv      bool              `json:"iv,omitempty"`
	Zone     string            `json:"z,omitempty"`
	FX       int               `json:"fx,omitempty"`
	FY       int               `json:"fy,omitempty"`
	Acts     []int             `json:"ac,omitempty"`
	New      []int             `json:"nw,omitempty"`
	Lead     int               `json:"ld,omitempty"`
	Off      int               `json:"of,omitempty"`
	AI       map[int]string    `json:"ai,omitempty"`
	Deco     []fwDeco          `json:"dc,omitempty"`
	Used     map[string]bool   `json:"u,omitempty"`
	Stage    int               `json:"sg,omitempty"`
	DX       int               `json:"dx,omitempty"`
	DY       int               `json:"dy,omitempty"`
	Dest     string            `json:"de,omitempty"`
	Name     string            `json:"n,omitempty"`
	Extra    map[string]string `json:"x,omitempty"`
}

type fwInq struct {
	Victim string `json:"v"`
	Culp   string `json:"c"`
	Burg   int    `json:"b"`
}

type fwState struct {
	Cers   []*fwCer           `json:"c,omitempty"`
	Snap   map[string]*fwSnap `json:"sn,omitempty"`
	Seen   map[string]bool    `json:"se,omitempty"` // children already known
	Kids   map[string]string  `json:"ki,omitempty"` // child key -> parent, still children
	Inq    []fwInq            `json:"iq,omitempty"`
	Mourn  map[string]int     `json:"mo,omitempty"` // kin key -> day the funeral you attended ended
	Firsts map[string]bool    `json:"fi,omitempty"`
	Last   int                `json:"la,omitempty"`
	Cur    string             `json:"cu,omitempty"`
	Menu   string             `json:"me,omitempty"`
	Phase  int                `json:"ph,omitempty"`
	Res    []string           `json:"re,omitempty"`
	N      int                `json:"n,omitempty"`
	Chk    int                `json:"ck,omitempty"`
}

// fwInv is one invitation from the letters system.
type fwInv struct {
	From string // person key
	Burg int
	Day  int
	Kind string // wedding funeral birth age
}

// fwInvites, when set by the letters file, lists invitations to ceremonies.
var fwInvites func(g *Game) []fwInv

func init() {
	unLocalKeys = append(unLocalKeys, fwKey)
	enterHooks = append(enterHooks, fwEnter)
	turnHooks = append(turnHooks, fwTurn)
	talkHooks = append(talkHooks, fwTalk)
	sideHooks = append(sideHooks, fwSide)
	modeKeys["fwc"] = fwcKeys
	modeDraws["fwc"] = func(g *Game) { g.fwDraw() }
	skAddStatus(&skStatusDef{ID: "fwblessed", Name: "Blessed by a rite", Glyph: '+', WIL: 2, EGO: 2})
}

func fwSt(g *Game) *fwState {
	s := g.ext(fwKey, func() any { return &fwState{} }).(*fwState)
	if s.Snap == nil {
		s.Snap = map[string]*fwSnap{}
	}
	if s.Seen == nil {
		s.Seen = map[string]bool{}
	}
	if s.Kids == nil {
		s.Kids = map[string]string{}
	}
	if s.Mourn == nil {
		s.Mourn = map[string]int{}
	}
	if s.Firsts == nil {
		s.Firsts = map[string]bool{}
	}
	return s
}

func fwRng(g *Game, parts ...any) *Rng {
	s := g.W.Seed + ":fw"
	for _, p := range parts {
		s += fmt.Sprintf(":%v", p)
	}
	return NewRng(s)
}

func fwKnown(p *lvPerson) bool { return p.Met || p.Opinion >= 10 }

func fwCerByID(st *fwState, id string) *fwCer {
	for _, c := range st.Cers {
		if c.ID == id {
			return c
		}
	}
	return nil
}

func fwDate(g *Game, day int) string {
	d := g.W.DateOf(g.calendarCulture(), day)
	return fmt.Sprintf("%s of %s", ordinal(d.Day), d.Name)
}

func fwBurgName(g *Game, id int) string {
	if b := g.W.burgByID[id]; b != nil {
		return b.Name
	}
	return "the town"
}

// fwReach: the burg is where you are or within a few days' travel.
func fwReach(g *Game, burg int) bool {
	bd := g.W.burgByID[burg]
	if bd == nil || g.P == nil {
		return false
	}
	pp, pb := lsPlayerLoc(g)
	if pb == burg {
		return true
	}
	return pp >= 0 && wsCheb(g.W, bd.P, pp) <= 8
}

func fwFirst(g *Game, key, text string) {
	st := fwSt(g)
	if st.Firsts[key] {
		return
	}
	st.Firsts[key] = true
	if g.Cur != nil {
		g.deed("life", text, 0, 0)
	}
}

func fwPerson(g *Game, key string) *lvPerson { return lvGet(g, key) }

func fwNames(g *Game, c *fwCer) []string {
	var out []string
	for _, k := range c.Subj {
		if p := lvGet(g, k); p != nil {
			out = append(out, p.Name)
		}
	}
	return out
}

func fwTitle(g *Game, c *fwCer) string {
	n := fwNames(g, c)
	switch c.Kind {
	case "wedding":
		if len(n) >= 2 {
			return "The wedding of " + n[0] + " and " + n[1]
		}
		return "A wedding"
	case "funeral":
		if len(n) >= 1 {
			return "The funeral of " + n[0]
		}
		return "A funeral"
	case "birth":
		if len(n) >= 1 {
			return "The naming of " + n[0]
		}
		return "A naming"
	case "age":
		if len(n) >= 1 {
			return n[0] + " comes of age"
		}
		return "A coming of age"
	}
	return "The " + nonEmpty(c.Name, "procession")
}

func fwOccasion(c *fwCer) string {
	switch c.Kind {
	case "wedding":
		return "wedding"
	case "funeral":
		return "funeral"
	case "birth":
		return "naming"
	case "age":
		return "coming-of-age rite"
	}
	return "festival"
}

// ---------------------------------------------------------------- scanning the life simulation

func fwSchedule(g *Game, kind string, burg, day int, subj []string, inv bool) *fwCer {
	st := fwSt(g)
	ks := append([]string{}, subj...)
	sort.Strings(ks)
	id := fmt.Sprintf("%s:%s:%d", kind, strings.Join(ks, "|"), lvYear(g))
	if c := fwCerByID(st, id); c != nil {
		return c
	}
	c := &fwCer{ID: id, Kind: kind, Burg: burg, Day: day, End: day + 1, Subj: subj, Inv: inv}
	st.Cers = append(st.Cers, c)
	c.Told = fwReach(g, burg)
	fwAnnounce(g, c)
	return c
}

func fwAnnounce(g *Game, c *fwCer) {
	where := fwBurgName(g, c.Burg)
	n := fwNames(g, c)
	if len(n) == 0 {
		return
	}
	var t string
	switch c.Kind {
	case "wedding":
		if len(n) < 2 {
			return
		}
		t = fmt.Sprintf("%s and %s will be wed in %s on the %s.", n[0], n[1], where, fwDate(g, c.Day))
	case "funeral":
		t = fmt.Sprintf("The funeral of %s is to be held in %s on the %s.", n[0], where, fwDate(g, c.Day))
	case "birth":
		t = fmt.Sprintf("%s's naming is kept in %s on the %s.", n[0], where, fwDate(g, c.Day))
	case "age":
		t = fmt.Sprintf("%s comes of age in %s: the rite is on the %s.", n[0], where, fwDate(g, c.Day))
	default:
		return
	}
	lvSt(g).addNews(lvDay(g), c.Burg, "%s", t)
	if c.Told {
		g.Msg("{{c|Word reaches you: " + t + "}}")
	}
}

func fwScan(g *Game) {
	st := fwSt(g)
	day := lvDay(g)
	ls := lvSt(g)
	var keys []string
	for k, p := range ls.People {
		if fwKnown(p) {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	for _, k := range keys {
		p := ls.People[k]
		sn := st.Snap[k]
		if p.Role == "child" && !st.Seen[k] {
			st.Seen[k] = true
			st.Kids[k] = k
		}
		kids := lsChildren(g, p)
		if sn == nil {
			st.Snap[k] = &fwSnap{D: !p.Alive, S: p.Spouse}
			for _, q := range kids {
				st.Seen[q.Key] = true
				if q.Role == "child" {
					st.Kids[q.Key] = k
				}
			}
			continue
		}
		burg := p.Burg
		if burg <= 0 {
			burg = p.Home
		}
		if !sn.D && !p.Alive {
			if fwReach(g, burg) {
				r := fwRng(g, "death", k, day)
				c := fwSchedule(g, "funeral", burg, day+r.Int(1, 3), []string{k}, false)
				if strings.HasPrefix(p.Cause, "murdered by ") {
					c.Sus = true
					who := strings.TrimPrefix(p.Cause, "murdered by ")
					for _, q := range ls.People {
						if q.Name == who {
							c.Culp = q.Key
							break
						}
					}
				}
			}
		}
		if sn.S == "" && p.Spouse != "" && p.Alive {
			if q := lvGet(g, p.Spouse); q != nil && q.Alive && fwReach(g, burg) {
				r := fwRng(g, "wed", k, day)
				fwSchedule(g, "wedding", burg, day+r.Int(2, 5), []string{k, q.Key}, false)
			}
		}
		for _, q := range kids {
			if st.Seen[q.Key] {
				continue
			}
			st.Seen[q.Key] = true
			if q.Role == "child" {
				st.Kids[q.Key] = k
			}
			if q.Alive && q.Born >= lvYear(g)-1 && fwReach(g, burg) {
				r := fwRng(g, "birth", q.Key, day)
				fwSchedule(g, "birth", burg, day+r.Int(3, 7), []string{q.Key, k}, false)
			}
		}
		st.Snap[k] = &fwSnap{D: !p.Alive, S: p.Spouse}
	}
	var kk []string
	for k := range st.Kids {
		kk = append(kk, k)
	}
	sort.Strings(kk)
	for _, k := range kk {
		q := lvGet(g, k)
		if q == nil || !q.Alive {
			delete(st.Kids, k)
			continue
		}
		if q.Role != "child" {
			delete(st.Kids, k)
			burg := q.Burg
			if burg <= 0 {
				burg = q.Home
			}
			if fwReach(g, burg) {
				r := fwRng(g, "age", k, day)
				fwSchedule(g, "age", burg, day+r.Int(1, 4), []string{k}, false)
			}
		}
	}
	if fwInvites != nil {
		for _, iv := range fwInvites(g) {
			p := lvGet(g, iv.From)
			if p == nil {
				continue
			}
			subj := []string{p.Key}
			if iv.Kind == "wedding" && p.Spouse != "" {
				subj = append(subj, p.Spouse)
			}
			if iv.Kind != "wedding" && iv.Kind != "funeral" && iv.Kind != "birth" && iv.Kind != "age" {
				continue
			}
			c := fwSchedule(g, iv.Kind, iv.Burg, maxi(iv.Day, day), subj, true)
			c.Told = true
		}
	}
	// expiry
	for _, c := range st.Cers {
		if !c.Done && day > c.End {
			c.Done = true
			if c.Told && !c.Attended && c.Kind != "feast" {
				fwMissed(g, c)
			}
		}
	}
	// solved murders: the dead person's kin remember
	if len(st.Inq) > 0 {
		kn := knSt(g)
		var keep []fwInq
		for _, q := range st.Inq {
			solved := false
			for _, cs := range kn.Cases {
				if cs.Culprit == q.Culp && cs.Status == "solved" {
					solved = true
				}
			}
			if !solved {
				keep = append(keep, q)
				continue
			}
			if v := lvGet(g, q.Victim); v != nil {
				for _, kin := range fwKinOf(g, v) {
					g.lvMemoryAdd(kin, "help", "You found who killed "+v.Name+".", 15)
				}
				lvSt(g).addNews(day, q.Burg, "The killer of %s has been named, thanks to the wanderer %s.", v.Name, g.P.Name)
				fwFirst(g, "solved", fmt.Sprintf("The wanderer %s found who killed %s and gave the kin peace.", g.P.Name, v.Name))
			}
		}
		st.Inq = keep
	}
	// prune
	if len(st.Cers) > 40 {
		var keep []*fwCer
		for _, c := range st.Cers {
			if c.Done && (c.Zone == "" || c.Clean) && c.End < day-20 {
				continue
			}
			keep = append(keep, c)
		}
		st.Cers = keep
	}
}

func fwKinOf(g *Game, p *lvPerson) []*lvPerson {
	var out []*lvPerson
	seen := map[string]bool{}
	add := func(k string) {
		if k == "" || seen[k] || len(out) >= 5 {
			return
		}
		seen[k] = true
		if q := lvGet(g, k); q != nil && q.Alive {
			out = append(out, q)
		}
	}
	add(p.Spouse)
	for _, k := range p.Kin {
		add(k)
	}
	return out
}

// fwMissed: the kin of someone close remember you did not come.
func fwMissed(g *Game, c *fwCer) {
	var close bool
	var who []*lvPerson
	for _, k := range c.Subj {
		p := lvGet(g, k)
		if p == nil {
			continue
		}
		if p.Opinion >= 40 {
			close = true
		}
		if c.Kind == "wedding" || c.Kind == "age" {
			if p.Alive {
				who = append(who, p)
			}
		} else {
			who = append(who, fwKinOf(g, p)...)
		}
	}
	if !close {
		return
	}
	n := 0
	for _, p := range who {
		if n >= 3 {
			break
		}
		g.lvMemoryAdd(p, "missed", "You did not come to the "+fwOccasion(c)+".", -4)
		n++
	}
}

// ---------------------------------------------------------------- turn, enter, site setup

func fwEnter(g *Game, zn *Zone) {
	if g.W == nil || zn == nil {
		return
	}
	fwCleanup(g, zn)
	fwHere(g)
}

func fwTurn(g *Game) {
	if g.P == nil || g.W == nil || g.Cur == nil || g.Dead {
		return
	}
	st := fwSt(g)
	day := lvDay(g)
	if st.Last != day {
		st.Last = day
		fwScan(g)
		fwCleanup(g, g.Cur)
	}
	if st.Cur != "" && g.Mode == "main" && !g.OnWorld {
		if c := fwCerByID(st, st.Cur); c != nil {
			g.Mode, g.Sel = "fwc", 0
		} else {
			st.Cur = ""
		}
		return
	}
	if g.OnWorld || g.Mode != "main" {
		return
	}
	st.Chk++
	if st.Chk%20 == 1 {
		fwHere(g)
	}
	zn := g.Cur
	for _, c := range st.Cers {
		if c.Done || c.Zone != zn.Key || day < c.Day || day > c.End {
			continue
		}
		if c.Kind == "feast" && c.Stage == 0 && g.Turn%3 == 0 {
			fwWalk(g, c)
		}
		if !c.Opened && maxi(abs(g.P.X-c.FX), abs(g.P.Y-c.FY)) <= 4 && !g.nearHostile() {
			fwOpen(g, c)
			return
		}
	}
}

var fwWants = map[string][]string{
	"wedding": {"temple", "plaza", "palace"},
	"funeral": {"temple", "homes", "plaza"},
	"birth":   {"homes", "plaza"},
	"age":     {"plaza", "homes", "market"},
	"feast":   {"temple", "plaza"},
}

func fwWantKind(g *Game, bd *BurgData, kind string) string {
	cp := g.cityPlanFor(bd)
	for _, k := range fwWants[kind] {
		if hasDistrict(cp, k) {
			return k
		}
	}
	return ""
}

func fwFits(g *Game, zn *Zone, kind string) bool {
	bd := g.W.burgByID[zn.Burg]
	if bd == nil {
		return false
	}
	want := fwWantKind(g, bd, kind)
	if want == "" {
		return true
	}
	kd, _ := g.districtAt(zn.WX, zn.WY, zn.ZX, zn.ZY)
	return kd == "" || kd == want
}

func fwFeastHere(g *Game) {
	b, f := g.feastToday()
	if f == nil || b == nil {
		return
	}
	st := fwSt(g)
	year := g.W.DateOf(nil, g.Day()).Year
	id := fmt.Sprintf("feast:%d:%d:%s", b.ID, year, f.Name)
	if fwCerByID(st, id) != nil {
		return
	}
	day := lvDay(g)
	st.Cers = append(st.Cers, &fwCer{ID: id, Kind: "feast", Burg: b.ID, Day: day, End: day, Told: true, Name: f.Name + " procession"})
}

func fwHere(g *Game) {
	zn := g.Cur
	if zn == nil || g.OnWorld || zn.Kind != "town" || zn.Burg <= 0 || zn.Z != 0 || g.W == nil || g.P == nil {
		return
	}
	st := fwSt(g)
	day := lvDay(g)
	fwFeastHere(g)
	for _, c := range st.Cers {
		if c.Done || c.Burg != zn.Burg || day < c.Day || day > c.End || c.Zone != "" {
			continue
		}
		if !fwFits(g, zn, c.Kind) {
			if !c.Hinted && c.Kind != "feast" {
				c.Hinted, c.Told = true, true
				bd := g.W.burgByID[zn.Burg]
				g.Msg(fmt.Sprintf("{{c|%s is being held in the %s today.}}", fwTitle(g, c), strings.ToLower(districtNames[fwWantKind(g, bd, c.Kind)])))
			}
			continue
		}
		fwPrepare(g, zn, c)
	}
}

var fwPlain = map[Terrain]bool{TCobble: true, TRoad: true, TDirt: true, TGrass: true, TFloor: true, TWoodFloor: true, TTileFloor: true, TCarpet: true, TTallGrass: true, TSand: true, TMoss: true}

func fwOpenTile(g *Game, zn *Zone, x, y int) bool {
	return zn.In(x, y) && fwPlain[zn.At(x, y)] && zn.ActorAt(x, y) == nil && !(g.P.X == x && g.P.Y == y)
}

func fwSite(g *Game, zn *Zone, r *Rng) (int, int, bool) {
	bx, by, best := 0, 0, -1
	for i := 0; i < 400; i++ {
		cx, cy := r.Int(6, MW-7), r.Int(4, MH-5)
		n := 0
		for dy := -2; dy <= 2; dy++ {
			for dx := -4; dx <= 4; dx++ {
				if fwOpenTile(g, zn, cx+dx, cy+dy) {
					n++
				}
			}
		}
		if n > best {
			bx, by, best = cx, cy, n
		}
		if n == 45 {
			break
		}
	}
	return bx, by, best >= 20
}

func fwDeco1(g *Game, zn *Zone, c *fwCer, x, y int, t Terrain, note string) {
	if !fwOpenTile(g, zn, x, y) {
		return
	}
	c.Deco = append(c.Deco, fwDeco{X: x, Y: y, Old: zn.At(x, y)})
	zn.Set(x, y, t)
	if note != "" {
		zn.Notes[y*MW+x] = note
	}
}

// fwFind locates the actor bound to a person in any zone of their burg.
func fwFind(g *Game, p *lvPerson) (*Zone, *Actor) {
	b := lvb(g)
	for _, zn := range g.Zones {
		if zn == nil || zn.Burg <= 0 {
			continue
		}
		for _, a := range zn.Actors {
			if a.HP <= 0 {
				continue
			}
			if k, ok := b.Bind[a.ID]; ok && k == p.Key {
				return zn, a
			}
			if a.SimPer != 0 && strings.HasPrefix(p.Key, "s") && fmt.Sprintf("s%d", a.SimPer) == p.Key {
				return zn, a
			}
		}
	}
	return nil, nil
}

func fwDropActor(zn *Zone, a *Actor) {
	for i, o := range zn.Actors {
		if o == a {
			zn.Actors = append(zn.Actors[:i], zn.Actors[i+1:]...)
			return
		}
	}
}

func fwActorFor(g *Game, zn *Zone, p *lvPerson, bd *BurgData, r *Rng) *Actor {
	if z2, a := fwFind(g, p); a != nil {
		if z2 != zn {
			fwDropActor(z2, a)
			zn.Actors = append(zn.Actors, a)
		}
		return a
	}
	role := lvTradeActorRole[p.Role]
	if role == "" {
		role = "villager"
	}
	cu := g.W.cultByID[p.Culture]
	if cu == nil {
		cu = g.W.cultByID[bd.Culture]
	}
	a := g.makeNPC(role, bd, cu, r)
	a.Species = p.Species
	zn.Actors = append(zn.Actors, a)
	lvb(g).Bind[a.ID] = p.Key
	lvApply(g, a, p, zn)
	return a
}

func fwPlace(g *Game, zn *Zone, a *Actor, x, y int) {
	for rad := 0; rad <= 3; rad++ {
		for dy := -rad; dy <= rad; dy++ {
			for dx := -rad; dx <= rad; dx++ {
				if maxi(abs(dx), abs(dy)) != rad {
					continue
				}
				nx, ny := x+dx, y+dy
				if zn.Walkable(nx, ny) && zn.ActorAt(nx, ny) == nil && !(g.P.X == nx && g.P.Y == ny) && zn.At(nx, ny).D().Water < 2 {
					a.X, a.Y, a.HomeX, a.HomeY = nx, ny, nx, ny
					return
				}
			}
		}
	}
}

func fwHold(c *fwCer, a *Actor) {
	if a.Role == "guard" || a.AI == "guard" {
		return
	}
	if c.AI == nil {
		c.AI = map[int]string{}
	}
	if _, ok := c.AI[a.ID]; !ok {
		c.AI[a.ID] = a.AI
	}
	a.AI = "still"
	a.Set("fwx")
	c.Acts = append(c.Acts, a.ID)
}

func fwPriest(g *Game, burg int, skip map[string]bool) *lvPerson {
	for _, p := range lvPresent(g, burg) {
		if skip[p.Key] || p.lvAge(lvYear(g)) < 18 {
			continue
		}
		if p.Role == "priest" || p.Role == "acolyte" || p.Job == "priest" || p.Job == "acolyte" {
			return p
		}
	}
	return nil
}

func fwPrepare(g *Game, zn *Zone, c *fwCer) {
	bd := g.W.burgByID[zn.Burg]
	if bd == nil {
		return
	}
	r := fwRng(g, "prep", c.ID)
	lvEnsureTown(g, zn.Burg)
	lvBindZone(g, zn)
	cx, cy, ok := fwSite(g, zn, r)
	c.Zone = zn.Key
	if !ok {
		c.FX, c.FY = g.P.X, g.P.Y
		c.Opened = true
		return
	}
	c.FX, c.FY = cx, cy-1
	switch c.Kind {
	case "funeral":
		fwDeco1(g, zn, c, cx, cy-1, TSarcophagus, fwDecoNotes["bier"])
		for _, dx := range []int{-3, 3} {
			fwDeco1(g, zn, c, cx+dx, cy-1, TBrazier, fwDecoNotes["brazier"])
		}
		for dx := -3; dx <= 3; dx += 2 {
			fwDeco1(g, zn, c, cx+dx, cy-2, TLantern, "")
		}
	case "feast":
		for _, dx := range []int{-3, 3} {
			fwDeco1(g, zn, c, cx+dx, cy-1, TBrazier, fwDecoNotes["brazier"])
		}
		for dx := -3; dx <= 3; dx++ {
			fwDeco1(g, zn, c, cx+dx, cy-2, TBunting, "")
		}
	default:
		t, note := TAltar, fwDecoNotes["altar"]
		if c.Kind == "birth" {
			t, note = TBed, fwDecoNotes["cradle"]
		}
		fwDeco1(g, zn, c, cx, cy-1, t, note)
		for _, dx := range []int{-2, 2} {
			fwDeco1(g, zn, c, cx+dx, cy-1, TBrazier, fwDecoNotes["brazier"])
		}
		for dx := -4; dx <= 4; dx++ {
			fwDeco1(g, zn, c, cx+dx, cy-2, TBunting, "")
		}
		for dy := 0; dy <= 2; dy++ {
			fwDeco1(g, zn, c, cx, cy+dy, TCarpet, "")
		}
	}
	if c.Kind != "feast" {
		fwDeco1(g, zn, c, cx+4, cy, TStall, fwDecoNotes["table"])
	}
	used := map[string]bool{}
	for _, k := range c.Subj {
		used[k] = true
	}
	// officiant
	var off *Actor
	if pr := fwPriest(g, zn.Burg, used); pr != nil {
		used[pr.Key] = true
		off = fwActorFor(g, zn, pr, bd, r)
	} else {
		cu := g.W.cultByID[bd.Culture]
		off = g.makeNPC("priest", bd, cu, r)
		off.Set("fwnew")
		zn.Actors = append(zn.Actors, off)
		c.New = append(c.New, off.ID)
	}
	off.Set("fwoff")
	fwPlace(g, zn, off, cx, cy)
	fwHold(c, off)
	c.Off = off.ID
	slots := [][2]int{{-3, 1}, {-2, 2}, {2, 2}, {3, 1}, {-3, 0}, {3, 0}, {-1, 2}, {1, 2}}
	si := 0
	seat := func(p *lvPerson, x, y int) {
		if p == nil || !p.Alive || used[p.Key] && x == 0 && y == 0 {
			return
		}
		if p.Burg != zn.Burg && p.Burg > 0 {
			return
		}
		used[p.Key] = true
		a := fwActorFor(g, zn, p, bd, r)
		fwPlace(g, zn, a, x, y)
		fwHold(c, a)
	}
	guest := func(p *lvPerson) {
		if p == nil || !p.Alive || used[p.Key] || si >= len(slots) || p.Burg != zn.Burg || p.lvAge(lvYear(g)) < 8 {
			return
		}
		s := slots[si]
		si++
		seat(p, cx+s[0], cy+s[1])
	}
	switch c.Kind {
	case "wedding":
		for i, k := range c.Subj {
			if p := lvGet(g, k); p != nil && p.Alive {
				delete(used, k)
				seat(p, cx-1+2*i, cy+1)
			}
		}
	case "age":
		if p := lvGet(g, c.Subj[0]); p != nil {
			delete(used, p.Key)
			seat(p, cx, cy+1)
		}
		for _, q := range lsParents(g, p0(g, c)) {
			guest(q)
		}
	case "birth":
		if len(c.Subj) > 1 {
			if par := lvGet(g, c.Subj[1]); par != nil {
				delete(used, par.Key)
				seat(par, cx-1, cy+1)
				if sp := lvGet(g, par.Spouse); sp != nil {
					seat(sp, cx+1, cy+1)
				}
			}
		}
	case "funeral":
		if d := p0(g, c); d != nil {
			for _, kin := range fwKinOf(g, d) {
				guest(kin)
			}
		}
	case "feast":
		fwPilgrims(g, zn, c, bd, r, used, off)
	}
	for _, k := range c.Subj {
		p := lvGet(g, k)
		if p == nil {
			continue
		}
		for _, f := range p.Friends {
			guest(lvGet(g, f))
		}
		for _, f := range p.Kin {
			guest(lvGet(g, f))
		}
	}
	if c.Kind == "feast" {
		c.Lead = off.ID
		fwProcPath(g, zn, c, cx, cy, r, bd)
	}
	lines := fwAmbient[c.Kind]
	amb := ""
	if len(lines) > 0 {
		amb = " " + Pick(r, lines)
	}
	g.Msg("{{c|" + fwTitle(g, c) + " is under way near here." + amb + "}}")
	if c.Kind != "feast" {
		c.Told = true
	}
}

func p0(g *Game, c *fwCer) *lvPerson {
	if len(c.Subj) == 0 {
		return &lvPerson{}
	}
	if p := lvGet(g, c.Subj[0]); p != nil {
		return p
	}
	return &lvPerson{}
}

// ---------------------------------------------------------------- feast processions

func fwPilgrims(g *Game, zn *Zone, c *fwCer, bd *BurgData, r *Rng, used map[string]bool, off *Actor) {
	faith := fwFaith(g, bd)
	var pool []*lvPerson
	for _, p := range lvPresent(g, bd.ID) {
		if used[p.Key] || p.lvAge(lvYear(g)) < 16 || p.Role == "guard" || p.Role == "sergeant" || p.Role == "child" {
			continue
		}
		if p.Faith == faith || p.Role == "priest" || p.Role == "acolyte" || p.Met {
			pool = append(pool, p)
		}
	}
	sort.SliceStable(pool, func(i, j int) bool {
		if pool[i].Met != pool[j].Met {
			return pool[i].Met
		}
		return pool[i].Key < pool[j].Key
	})
	for i, p := range pool {
		if i >= 3 {
			break
		}
		used[p.Key] = true
		c.Subj = append(c.Subj, p.Key)
		a := fwActorFor(g, zn, p, bd, r)
		fwPlace(g, zn, a, off.X-1+i, off.Y+2)
		fwHold(c, a)
	}
}

func fwFaith(g *Game, bd *BurgData) int {
	if bd == nil || len(g.W.Religion) == 0 {
		return -1
	}
	return g.W.Religion[clampi(bd.P, 0, len(g.W.Religion)-1)]
}

func fwProcPath(g *Game, zn *Zone, c *fwCer, cx, cy int, r *Rng, bd *BurgData) {
	var dx, dy int
	found := false
	for i := 0; i < 200 && !found; i++ {
		x, y := r.Int(4, MW-5), r.Int(3, MH-4)
		if maxi(abs(x-cx), abs(y-cy)) >= 16 && fwOpenTile(g, zn, x, y) {
			dx, dy, found = x, y, true
		}
	}
	if !found {
		c.Stage = 1
		return
	}
	dest := Pick(r, fwFeastShrine)
	best := 5
	for _, m := range g.W.Wonders {
		if m.Destroyed || m.Layer != "" && m.Layer != "surface" {
			continue
		}
		if d := wsCheb(g.W, bd.P, m.P); d < best {
			best, dest = d, "the holy place of "+m.Name
		}
	}
	c.DX, c.DY, c.Dest = dx, dy, dest
	fwDeco1(g, zn, c, dx, dy, TAltar, fwDecoNotes["shrine"])
	c.DY = dy + 1
}

func fwStep(zn *Zone, a *Actor, tx, ty int) {
	sx, sy := 0, 0
	if tx > a.X {
		sx = 1
	} else if tx < a.X {
		sx = -1
	}
	if ty > a.Y {
		sy = 1
	} else if ty < a.Y {
		sy = -1
	}
	for _, m := range [][2]int{{sx, sy}, {sx, 0}, {0, sy}} {
		if m[0] == 0 && m[1] == 0 {
			continue
		}
		nx, ny := a.X+m[0], a.Y+m[1]
		if zn.Walkable(nx, ny) && zn.ActorAt(nx, ny) == nil && zn.At(nx, ny).D().Water < 2 {
			a.X, a.Y, a.HomeX, a.HomeY = nx, ny, nx, ny
			return
		}
	}
}

func fwWalk(g *Game, c *fwCer) {
	zn := g.Cur
	var lead *Actor
	for _, a := range zn.Actors {
		if a.ID == c.Lead {
			lead = a
		}
	}
	if lead == nil {
		c.Stage = 1
		return
	}
	if maxi(abs(lead.X-c.DX), abs(lead.Y-c.DY)) <= 1 {
		c.Stage = 1
		c.FX, c.FY = lead.X, lead.Y
		c.Opened = false
		if maxi(abs(g.P.X-lead.X), abs(g.P.Y-lead.Y)) < 30 {
			g.Msg("{{c|The procession reaches " + nonEmpty(c.Dest, "the shrine") + " and halts.}}")
		}
		return
	}
	if g.Turn%2 == 0 {
		fwStep(zn, lead, c.DX, c.DY)
	}
	c.FX, c.FY = lead.X, lead.Y
	i := 0
	for _, id := range c.Acts {
		if id == c.Lead {
			continue
		}
		for _, a := range zn.Actors {
			if a.ID == id && a.HP > 0 {
				i++
				if maxi(abs(a.X-lead.X), abs(a.Y-lead.Y)) > 1+i/2 {
					fwStep(zn, a, lead.X, lead.Y)
				}
			}
		}
	}
}

// ---------------------------------------------------------------- cleanup

func fwCleanup(g *Game, zn *Zone) {
	if zn == nil {
		return
	}
	st := fwSt(g)
	day := lvDay(g)
	for _, c := range st.Cers {
		if c.Zone != zn.Key || c.Clean || !(c.Done || day > c.End) {
			continue
		}
		for _, d := range c.Deco {
			zn.Set(d.X, d.Y, d.Old)
			delete(zn.Notes, d.Y*MW+d.X)
		}
		isNew := map[int]bool{}
		for _, id := range c.New {
			isNew[id] = true
		}
		keep := zn.Actors[:0:0]
		for _, a := range zn.Actors {
			if isNew[a.ID] {
				continue
			}
			if ai, ok := c.AI[a.ID]; ok {
				a.AI = nonEmpty(ai, "town")
				delete(a.Flags, "fwx")
				delete(a.Flags, "fwoff")
			}
			keep = append(keep, a)
		}
		zn.Actors = keep
		c.Clean = true
	}
}

// ---------------------------------------------------------------- talk, sidebar

func fwCerOfActor(g *Game, a *Actor) *fwCer {
	st := fwSt(g)
	if g.Cur == nil {
		return nil
	}
	for _, c := range st.Cers {
		if c.Zone == g.Cur.Key && !c.Done && c.Off == a.ID {
			return c
		}
	}
	return nil
}

func fwTalk(g *Game, a *Actor) []Option {
	if a == nil || g.W == nil {
		return nil
	}
	var out []Option
	if c := fwCerOfActor(g, a); c != nil && a.Has("fwoff") {
		out = append(out, Option{"Join the " + fwOccasion(c) + ".", func() {
			g.Mode = "main"
			fwOpen(g, c)
		}})
	}
	if p := lvPersonOf(g, a); p != nil {
		st := fwSt(g)
		if d, ok := st.Mourn[p.Key]; ok && lvDay(g)-d <= 40 && !st.Firsts["chk:"+p.Key] {
			out = append(out, Option{"Check on them since the funeral.", func() {
				st.Firsts["chk:"+p.Key] = true
				g.lvMemoryAdd(p, "help", "You came back to ask how they were bearing it.", 3)
				ls := []string{
					"They say the house is quiet and the bills are not. Then they laugh a little. \"It helps that you asked.\"",
					"They hold the cup in both hands. \"There is the matter of the roof, and the harvest. If you hear of honest work, think of us.\"",
				}
				g.talkMenu(ls[int(HashStr(p.Key))%len(ls)])
			}})
		}
	}
	return out
}

func fwSide(g *Game) []string {
	if g.W == nil || g.P == nil {
		return nil
	}
	st := fwSt(g)
	day := lvDay(g)
	var out []string
	for _, c := range st.Cers {
		if c.Done || !c.Told || c.Kind == "feast" || day > c.End || len(out) >= 2 {
			continue
		}
		n := fwNames(g, c)
		if len(n) == 0 {
			continue
		}
		kind := map[string]string{"wedding": "Wedding", "funeral": "Funeral", "birth": "Naming", "age": "Rite"}[c.Kind]
		nm := strings.Fields(n[0])[0]
		d := c.Day - day
		when := "today"
		if d > 0 {
			when = fmt.Sprintf("in %dd", d)
		}
		out = append(out, fmt.Sprintf("{{Y|%s}} %s, %s", kind, lvTrunc(nm, 8), when))
	}
	return out
}

// ---------------------------------------------------------------- choice screen

type fwOpt struct {
	label string
	ok    bool
	why   string
	do    func()
}

func fwOpen(g *Game, c *fwCer) {
	st := fwSt(g)
	st.Cur, st.Menu, st.Phase, st.Res = c.ID, "", 0, nil
	c.Opened = true
	if !c.Attended {
		c.Attended = true
		st.N++
		if c.Kind == "funeral" {
			if d := p0(g, c); d != nil {
				for _, kin := range fwKinOf(g, d) {
					st.Mourn[kin.Key] = lvDay(g)
				}
			}
		}
		fwFirst(g, "first:"+c.Kind, fwFirstText(g, c))
	}
	g.Mode, g.Sel = "fwc", 0
}

func fwFirstText(g *Game, c *fwCer) string {
	switch c.Kind {
	case "wedding":
		return fmt.Sprintf("The wanderer %s was a guest at a wedding in %s.", g.P.Name, fwBurgName(g, c.Burg))
	case "funeral":
		return fmt.Sprintf("The wanderer %s stood among the mourners at a funeral in %s.", g.P.Name, fwBurgName(g, c.Burg))
	case "birth":
		return fmt.Sprintf("The wanderer %s blessed a newborn at a naming in %s.", g.P.Name, fwBurgName(g, c.Burg))
	case "age":
		return fmt.Sprintf("The wanderer %s saw a youth come of age in %s.", g.P.Name, fwBurgName(g, c.Burg))
	}
	return fmt.Sprintf("The wanderer %s took part in a festival procession in %s.", g.P.Name, fwBurgName(g, c.Burg))
}

func fwClose(g *Game) {
	st := fwSt(g)
	st.Cur, st.Menu, st.Phase, st.Res = "", "", 0, nil
	g.Mode = "main"
}

func fwRes(g *Game, lines ...string) {
	st := fwSt(g)
	st.Res, st.Phase = lines, 1
	for _, l := range lines {
		g.Msg(l)
	}
}

func fwcKeys(g *Game, k string) {
	st := fwSt(g)
	c := fwCerByID(st, st.Cur)
	if c == nil {
		fwClose(g)
		return
	}
	if st.Phase == 1 {
		if k == "Enter" || k == " " || k == "Escape" {
			st.Phase, st.Res = 0, nil
			if g.Dead {
				fwClose(g)
			}
		}
		return
	}
	opts := fwOptions(g, c)
	pick := func(i int) {
		if i < 0 || i >= len(opts) {
			return
		}
		if !opts[i].ok {
			g.Msg("{{K|Not now: " + opts[i].why + ".}}")
			return
		}
		opts[i].do()
	}
	if i := letterIndex(k); i >= 0 {
		pick(i)
		return
	}
	switch k {
	case "ArrowUp":
		g.Sel = maxi(0, g.Sel-1)
	case "ArrowDown":
		g.Sel = mini(len(opts)-1, g.Sel+1)
	case "Enter", " ":
		pick(g.Sel)
	case "Escape":
		if st.Menu != "" {
			st.Menu = ""
			return
		}
		fwClose(g)
	}
}

func fwDrawLines(g *Game, c *fwCer) []string {
	n := fwNames(g, c)
	where := fwBurgName(g, c.Burg)
	switch c.Kind {
	case "wedding":
		if len(n) >= 2 {
			return []string{fmt.Sprintf("%s and %s stand before the altar in %s, garlands overhead. The priest waits with open hands; a long table is laid at the side.", n[0], n[1], where)}
		}
	case "funeral":
		t := fmt.Sprintf("%s lies in state on a draped bier, braziers burning at either end. The kin stand close, and a table is set for the wake.", nonEmpty(strings.Join(n, ""), "The dead"))
		if c.Sus {
			t += " Some say the death was no accident."
		}
		return []string{t}
	case "birth":
		if len(n) >= 1 {
			return []string{fmt.Sprintf("The child %s is held up to be named before the altar in %s. Ribbons hang from a cradle.", n[0], where)}
		}
	case "age":
		if len(n) >= 1 {
			return []string{fmt.Sprintf("%s steps up before the elders of %s, no longer a child. Kin crowd round with gifts.", n[0], where)}
		}
	case "feast":
		t := "The priest leads a slow procession of pilgrims"
		if c.Stage == 0 {
			return []string{t + " toward " + nonEmpty(c.Dest, "the shrine") + ", lanterns swinging."}
		}
		return []string{t + " has reached " + nonEmpty(c.Dest, "the shrine") + " and halts for the blessing."}
	}
	return []string{"A rite is held here."}
}

func (g *Game) fwDraw() {
	sc := g.sc
	st := fwSt(g)
	c := fwCerByID(st, st.Cur)
	if c == nil {
		return
	}
	x, y, w, _ := g.panel("Ceremony")
	sc.TextB(x, y, "{{M|"+fwTitle(g, c)+"}}", CGray, CBgDark, x+w)
	y += 2
	if st.Phase == 1 {
		for _, l := range st.Res {
			for _, wl := range Wrap(l, w-2) {
				sc.TextB(x, y, wl, CGray, CBgDark, x+w)
				y++
			}
			y++
		}
		sc.TextB(x, y, "{{K|Enter to go on.}}", CGray, CBgDark, x+w)
		return
	}
	for _, para := range fwDrawLines(g, c) {
		for _, l := range Wrap(para, w-2) {
			sc.TextB(x, y, l, CGray, CBgDark, x+w)
			y++
		}
	}
	y++
	opts := fwOptions(g, c)
	g.Sel = clampi(g.Sel, 0, len(opts)-1)
	for i, o := range opts {
		bg := CBgDark
		if i == g.Sel {
			bg = CBgSel
		}
		var ls []string
		if o.ok {
			ls = Wrap(fmt.Sprintf("{{W|%c)}} %s", 'a'+i, o.label), w-4)
		} else {
			ls = Wrap(fmt.Sprintf("{{K|%c) %s (%s)}}", 'a'+i, Plain(o.label), o.why), w-4)
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
	sc.TextB(x, y+1, "{{K|Letter or Enter to choose. Esc to step back.}}", CGray, CBgDark, x+w)
}

// ---------------------------------------------------------------- helpers for choices

func fwCheck(g *Game, kind string, dc int) bool { return g.roll2d6Check(g.checkMod(kind), dc) }

// fwGuests: the people bound to the actors at the site.
func fwGuests(g *Game, c *fwCer) []*lvPerson {
	var out []*lvPerson
	seen := map[string]bool{}
	b := lvb(g)
	for _, id := range c.Acts {
		if k, ok := b.Bind[id]; ok && !seen[k] {
			if p := lvGet(g, k); p != nil && p.Alive {
				seen[k] = true
				out = append(out, p)
			}
		}
	}
	return out
}

func fwHonorees(g *Game, c *fwCer) []*lvPerson {
	var out []*lvPerson
	switch c.Kind {
	case "funeral":
		return fwKinOf(g, p0(g, c))
	case "birth":
		for _, k := range c.Subj {
			if p := lvGet(g, k); p != nil && p.Alive && p.Role != "child" {
				out = append(out, p)
				if sp := lvGet(g, p.Spouse); sp != nil && sp.Alive {
					out = append(out, sp)
				}
			}
		}
	case "feast":
		if a := fwActor(g, c.Off); a != nil {
			if p := lvPersonOf(g, a); p != nil {
				out = append(out, p)
			}
		}
	default:
		for _, k := range c.Subj {
			if p := lvGet(g, k); p != nil && p.Alive {
				out = append(out, p)
			}
		}
		if c.Kind == "age" && len(out) > 0 {
			for _, par := range lsParents(g, out[0]) {
				if par.Alive {
					out = append(out, par)
				}
			}
		}
	}
	return out
}

func fwActor(g *Game, id int) *Actor {
	if g.Cur == nil {
		return nil
	}
	for _, a := range g.Cur.Actors {
		if a.ID == id {
			return a
		}
	}
	return nil
}

func fwBoost(g *Game, ps []*lvPerson, kind, text string, w int) {
	for _, p := range ps {
		g.lvMemoryAdd(p, kind, text, w)
	}
}

func fwFaithRep(g *Game, n int, why string) {
	bd := g.W.burgByID[g.Cur.Burg]
	if bd == nil {
		return
	}
	if f := fwFaith(g, bd); f >= 0 {
		g.AddRep(fmt.Sprintf("faith:%d", f), n, why)
	}
}

func fwStateRep(g *Game, n int, why string) {
	if bd := g.W.burgByID[g.Cur.Burg]; bd != nil && bd.State > 0 {
		g.AddRep(fmt.Sprintf("state:%d", bd.State), n, why)
	}
}

func fwUsed(c *fwCer, k string) bool { return c.Used[k] }
func fwUse(c *fwCer, k string) {
	if c.Used == nil {
		c.Used = map[string]bool{}
	}
	c.Used[k] = true
}

func fwTake(g *Game, it *Item) {
	if it.Count > 1 {
		it.Count--
		return
	}
	g.removeInv(it)
}

// fwGiftOpts: the gift submenu (coin or something from the pack).
func fwGiftOpts(g *Game, c *fwCer) []fwOpt {
	cur := g.Sk.Currency
	hon := fwHonorees(g, c)
	occ := fwOccasion(c)
	var o []fwOpt
	gift := func(label string, value int, pay func() bool, good string) {
		o = append(o, fwOpt{label: label, ok: true, do: func() {
			if !pay() {
				fwRes(g, "{{R|You do not have that to give.}}")
				return
			}
			w := clampi(2+value/20, 2, 15)
			if c.Kind == "funeral" {
				w = clampi(1+value/25, 1, 10)
			}
			fwBoost(g, hon, "gift", "You brought something to the "+occ+".", w)
			fwSt(g).Menu = ""
			fwUse(c, "gift")
			g.gainXP(3)
			if c.Kind == "feast" {
				fwFaithRep(g, 3, " for your offering")
			}
			fwRes(g, good)
		}})
	}
	if c.Kind == "funeral" {
		gift("Lay a wildflower on the bier.", 10, func() bool { return true }, "You lay a flower on the black cloth and stand a moment. The kin see it.")
	}
	for _, n := range []int{10, 40} {
		n := n
		gift(fmt.Sprintf("Give %d %s.", n, cur), n, func() bool {
			if g.P.Gold < n {
				return false
			}
			g.P.Gold -= n
			return true
		}, fmt.Sprintf("You press %d %s into waiting hands. It is accepted with thanks.", n, cur))
		if g.P.Gold < n {
			o[len(o)-1].ok, o[len(o)-1].why = false, "not enough coin"
		}
	}
	cnt := 0
	for _, it := range g.P.Inv {
		if cnt >= 4 {
			break
		}
		switch it.Kind {
		case "food", "dish", "tonic", "book", "trophy", "water":
		default:
			continue
		}
		if uwIsStolen(it) || it.Value <= 0 {
			continue
		}
		it := it
		cnt++
		gift("Give your "+it.Name+".", it.Value, func() bool {
			fwTake(g, it)
			return true
		}, "You hand over the "+it.Name+". It is received with both hands.")
	}
	o = append(o, fwOpt{label: "Step back.", ok: true, do: func() { fwSt(g).Menu = "" }})
	return o
}

func fwSecretFor(g *Game, c *fwCer) (who *lvPerson, text string, truth bool) {
	kn := knSt(g)
	for _, k := range c.Subj {
		p := lvGet(g, k)
		if p == nil {
			continue
		}
		for _, l := range kn.Learned {
			f := kn.Facts[l.Fact]
			if f == nil || f.Subject != k || f.Ver < 0 {
				continue
			}
			if f.Kind == "secret" || f.Kind == "crime" || f.Kind == "rumour" {
				return p, strings.TrimSuffix(Plain(f.Text), "."), f.True || f.Ver > 0
			}
		}
		if p.Secret != "" && lvSecretKnown(p) {
			return p, strings.TrimSuffix(Plain(p.Secret), "."), true
		}
	}
	return nil, "", false
}

func fwEat(g *Game, c *fwCer, dur int) string {
	bd := g.W.burgByID[c.Burg]
	if bd == nil {
		return "You eat what is put in front of you."
	}
	fs := g.W.FeastsOf(bd)
	if len(fs) == 0 {
		g.applyMeal("shared table food", tagsOf("hearty"), dur, true)
		return "You eat with the guests: plain, hearty food."
	}
	f := &fs[int(HashStr(c.ID))%len(fs)]
	name, dk := g.feastDish(bd, f)
	g.spendTurns(25)
	g.applyMeal(name, tagsOf(dk), dur, true)
	return "You eat " + name + " at the long table, passing the dish along with the others."
}

func fwOptions(g *Game, c *fwCer) []fwOpt {
	st := fwSt(g)
	if st.Menu == "gift" {
		return fwGiftOpts(g, c)
	}
	cur := g.Sk.Currency
	var o []fwOpt
	add := func(key, label string, ok bool, why string, do func()) {
		if fwUsed(c, key) {
			o = append(o, fwOpt{label: label, ok: false, why: "done"})
			return
		}
		o = append(o, fwOpt{label: label, ok: ok, why: why, do: func() { fwUse(c, key); do() }})
	}
	hon := fwHonorees(g, c)
	guests := fwGuests(g, c)
	occ := fwOccasion(c)
	eat := func() {
		fwRes(g, fwEat(g, c, DayLen/2))
		fwBoost(g, guests, "gift", "You shared their table at the "+occ+".", 1)
	}
	giftOpt := func(label string) {
		o = append(o, fwOpt{label: label, ok: true, do: func() { st.Menu, g.Sel = "gift", 0 }})
	}
	switch c.Kind {
	case "wedding":
		giftOpt("Give a gift to the couple.")
		add("toast", g.checkLabel("persuasion", dcMedium)+" Raise a toast to the couple.", true, "", func() {
			r := fwRng(g, "toast", c.ID, g.Turn)
			if fwCheck(g, "persuasion", dcMedium) {
				fwBoost(g, hon, "gift", "Your toast at their wedding was lovely.", 6)
				fwBoost(g, guests, "gift", "You gave a fine toast at a wedding.", 2)
				g.gainXP(14)
				fwRes(g, "{{G|[Persuasion: Success]}}", Pick(r, fwToastsGood))
				return
			}
			fwBoost(g, hon, "insult", "Your toast at their wedding embarrassed them.", -4)
			fwBoost(g, guests, "insult", "You gave an awkward toast at a wedding.", -1)
			lines := []string{"{{R|[Persuasion: Failure]}}", Pick(r, fwToastsBad)}
			if rv := fwRival(g, hon); rv != nil && r.Chance(0.5) {
				lines = append(lines, fwCrash(g, c, hon, rv))
			}
			fwRes(g, lines...)
		})
		add("dance", g.checkLabel("acrobatics", dcEasy)+" Dance with the guests.", true, "", func() {
			if fwCheck(g, "acrobatics", dcEasy) {
				fwBoost(g, guests, "gift", "You danced well at the wedding.", 2)
				fwBoost(g, hon, "gift", "You danced at their wedding.", 2)
				g.gainXP(10)
				fwRes(g, "{{G|[Acrobatics: Success]}}", "You are pulled into the reel and stay in it. By the third round the bride is cheering you on.")
				return
			}
			fwBoost(g, guests, "insult", "You tripped over your own feet at the wedding.", -1)
			fwRes(g, "{{R|[Acrobatics: Failure]}}", "You miss a step, take a bench with you, and rise to polite applause.")
		})
		add("kiss", g.checkLabel("streetwise", dcChallenging)+" Steal a kiss from the bride's cheek.", true, "", func() {
			if fwCheck(g, "streetwise", dcChallenging) {
				fwBoost(g, hon, "gift", "You stole a cheeky kiss at the wedding and got away with it.", 1)
				fwRes(g, "{{G|[Streetwise: Success]}}", "A quick kiss, a bow, gone before anyone is sure. The bride laughs; the groom pretends not to have seen.")
				return
			}
			fwBoost(g, hon, "insult", "You tried to steal a kiss at their wedding.", -10)
			fwBoost(g, guests, "insult", "You made a scene at a wedding.", -3)
			lines := []string{"{{R|[Streetwise: Failure]}}", "You are caught mid-kiss. The slap is heard across the yard."}
			if rv := fwRival(g, hon); rv != nil {
				lines = append(lines, fwCrash(g, c, hon, rv))
			}
			fwRes(g, lines...)
		})
		add("scene", "Start a scene: heckle the vows.", true, "", func() {
			fwBoost(g, hon, "insult", "You heckled at their wedding.", -8)
			fwBoost(g, guests, "insult", "You made a scene at a wedding.", -2)
			fwRes(g, "You shout over the priest. Heads turn; hands go to belts.", fwCrash(g, c, hon, fwRival(g, hon)))
		})
		who, text, truth := fwSecretFor(g, c)
		why := "you know nothing against them"
		if who != nil {
			why = ""
		}
		add("object", "Object when the priest asks for reasons they should not wed.", who != nil, why, func() {
			lines := []string{fwScandalPool[0]}
			if truth {
				lines = append(lines, fwCallOff(g, c, who, text)...)
			} else {
				fwBoost(g, hon, "insult", "You accused them at their wedding with a lie.", -20)
				fwBoost(g, guests, "insult", "You disrupted a wedding with a false claim.", -4)
				lines = append(lines, "You speak. It is a rumour, and a wrong one: the priest asks for proof, and there is none. The wedding goes on and you are shown to the gate.")
				fwFaithRep(g, -4, " for your false claim")
			}
			fwRes(g, lines...)
		})
		add("eat", "Eat at the wedding table.", true, "", eat)
	case "funeral":
		giftOpt("Lay a flower or an offering on the bier.")
		add("eulogy", g.checkLabel("persuasion", dcMedium)+" Speak a eulogy.", true, "", func() {
			r := fwRng(g, "eulogy", c.ID, g.Turn)
			if fwCheck(g, "persuasion", dcMedium) {
				fwBoost(g, hon, "gift", "You spoke well of their dead.", 6)
				fwBoost(g, guests, "gift", "You spoke a fine eulogy.", 2)
				g.gainXP(14)
				fwRes(g, "{{G|[Persuasion: Success]}}", Pick(r, fwEulogyGood))
				return
			}
			fwBoost(g, hon, "insult", "Your eulogy for their dead rang false.", -2)
			fwRes(g, "{{R|[Persuasion: Failure]}}", Pick(r, fwEulogyBad))
		})
		add("console", "Console the kin.", true, "", func() {
			g.spendTurns(20)
			fwBoost(g, hon, "help", "You sat with them at the funeral.", 4)
			g.gainXP(6)
			fwRes(g, fwConsoleLines[int(HashStr(c.ID))%len(fwConsoleLines)])
		})
		add("pocket", g.checkLabel("streetwise", dcChallenging)+" Pocket something from the bier.", true, "", func() {
			bd := g.W.burgByID[c.Burg]
			it := &Item{ID: g.newID(), Name: "grave-goods ring", Glyph: '=', F: CYellow, Kind: "trophy", Value: 45, Weight: 1, Lore: "A plain ring slipped from the bier at a funeral."}
			if bd != nil {
				g.uwMarkStolen(it, bd.State)
			}
			g.give(it)
			if fwCheck(g, "streetwise", dcChallenging) {
				fwRes(g, "{{G|[Streetwise: Success]}}", "While the bell tolls you ease a ring from the dead hand's cloth. No one sees. You are not proud of it.")
				return
			}
			g.lvWitness("theft", "robbed the dead at a funeral", -12)
			fwBoost(g, hon, "theft", "You robbed the bier at their funeral.", -15)
			if bd != nil {
				g.uwAddBounty(bd.State, 25, " (robbing the dead)")
			}
			fwRes(g, "{{R|[Streetwise: Failure]}}", "A mourner sees your hand on the bier and screams. Heads turn.")
		})
		if c.Sus {
			add("inquire", g.checkLabel("perception", dcMedium+1)+" Look into how the dead met their end.", true, "", func() {
				d := p0(g, c)
				if fwCheck(g, "perception", dcMedium+1) {
					g.gainXP(12)
					lines := []string{"{{G|[Perception: Success]}}", "You look at the hands, the cloth, the faces that will not meet yours. The death was no accident."}
					if c.Culp != "" {
						if cu := lvGet(g, c.Culp); cu != nil {
							g.knCrime(c.Burg, c.Culp, "the killing of "+d.Name)
							fwSt(g).Inq = append(fwSt(g).Inq, fwInq{Victim: d.Key, Culp: c.Culp, Burg: c.Burg})
							lines = append(lines, "You note it all down: a case is open, and the kin will remember it if you see it through.")
						}
					}
					fwBoost(g, hon, "help", "You took their dead's death seriously.", 3)
					fwRes(g, lines...)
					return
				}
				fwRes(g, "{{R|[Perception: Failure]}}", "You see only grief. Perhaps there is nothing to see; perhaps you are not looking right.")
			})
		}
		add("wake", "Join the wake feast.", true, "", func() {
			lines := []string{fwEat(g, c, DayLen/2)}
			d := p0(g, c)
			kin := fwKinOf(g, d)
			var warm *lvPerson
			for _, k := range kin {
				if k.Opinion >= 15 {
					warm = k
					break
				}
			}
			r := fwRng(g, "wake", c.ID)
			switch {
			case warm == nil:
				lines = append(lines, "The kin do not seem to know what to say to you.")
			case warm.Secret != "" && warm.Opinion >= 25:
				lvRevealSecret(g, warm)
				g.lvMemoryAdd(warm, "trust", "You were trusted with a secret at a wake.", 3)
				lines = append(lines, warm.Name+" draws you aside: \"I have told no one this.\" "+Plain(warm.Secret))
			case r.Chance(0.6):
				k := Pick(r, fwKeepsakes)
				g.give(&Item{ID: g.newID(), Name: k.name, Glyph: '*', F: CYellow, Kind: "trophy", Value: 12, Weight: 1, Lore: k.lore})
				lines = append(lines, warm.Name+" presses a keepsake of "+d.Name+" into your hand: a "+k.name+". \"They would want it carried.\"")
			default:
				lines = append(lines, Pick(r, fwHeirloomTalk))
			}
			fwBoost(g, guests, "gift", "You shared the wake meal.", 1)
			fwRes(g, lines...)
		})
	case "birth", "age":
		giftOpt("Give a gift to the family.")
		add("bless", "Give your blessing.", true, "", func() {
			g.P.AddSt("fwblessed", 600)
			fwBoost(g, hon, "gift", "You blessed their child's "+occ+".", 4)
			fwFaithRep(g, 3, " for your blessing")
			g.gainXP(8)
			fwRes(g, fwBlessings[int(HashStr(c.ID))%len(fwBlessings)], "{{G|You feel blessed in turn: steadier, warmer.}}")
		})
		if c.Kind == "age" {
			add("story", "Tell the youth a story from your travels.", len(g.W.Events) > 0, "you have no tales", func() {
				r := fwRng(g, "story", c.ID)
				e := g.W.Events[r.Int(0, len(g.W.Events)-1)]
				t := lvTrunc(Plain(e.Text), 150)
				if kid := lvGet(g, c.Subj[0]); kid != nil {
					g.lvMemoryAdd(kid, "story", "You told them a tale at their coming of age.", 8)
					kid.addLife(lvYear(g), "friend", "Heard a tale from the wanderer on coming of age.")
				}
				g.gainXP(8)
				fwRes(g, Pick(r, fwStoryIntros)+" the year "+fmt.Sprint(e.Y)+": "+t, "The youth's eyes do not leave your face. They will remember you.")
			})
		}
		add("eat", "Eat at the family table.", true, "", eat)
	case "feast":
		add("join", "Join the procession and walk with the pilgrims.", true, "", func() {
			g.spendTurns(40)
			fwFaithRep(g, 8, " for walking with the pilgrims")
			fwBoost(g, hon, "help", "You walked in their procession.", 5)
			fwBoost(g, guests, "help", "You walked in the procession.", 3)
			g.P.AddSt("fwblessed", 600)
			g.gainXP(15)
			fwRes(g, "You fall in behind the priest and walk, lantern in hand, the whole slow way. At the end the priest touches your forehead with ash and oil.", "{{G|You feel blessed.}}")
		})
		add("alms", fmt.Sprintf("Give alms (10 %s) to the priest.", cur), g.P.Gold >= 10, "not enough coin", func() {
			g.P.Gold -= 10
			fwFaithRep(g, 4, " for your alms")
			fwBoost(g, hon, "gift", "You gave alms at the procession.", 5)
			fwRes(g, "The priest closes your hand over a blessed token and says your name aloud to the crowd.")
		})
		add("disrupt", g.checkLabel("intimidate", dcMedium)+" Disrupt the procession.", true, "", func() {
			bd := g.W.burgByID[c.Burg]
			fwFirst(g, "disrupt", fmt.Sprintf("The wanderer %s broke up a holy procession in %s.", g.P.Name, fwBurgName(g, c.Burg)))
			fwBoost(g, guests, "insult", "You disrupted a holy procession.", -8)
			fwBoost(g, hon, "insult", "You disrupted their procession.", -20)
			fwFaithRep(g, -15, " for disrupting a holy rite")
			if fwCheck(g, "intimidate", dcMedium) {
				fwRes(g, "{{G|[Intimidate: Success]}}", "You shove through the line and kick over the censer. The pilgrims scatter, scandalised. The procession will not finish today.")
				c.Stage = 1
				if bd != nil {
					g.uwAddBounty(bd.State, 12, " (disrupting a rite)")
				}
				return
			}
			g.P.AddSt("shamed", 30)
			fwRes(g, "{{R|[Intimidate: Failure]}}", "The pilgrims close ranks and shove you out, and a hundred eyes remember your face.")
		})
	}
	o = append(o, fwOpt{label: "Slip away quietly.", ok: true, do: func() { fwClose(g) }})
	_ = cur
	return o
}

func fwRival(g *Game, hon []*lvPerson) *lvPerson {
	for _, h := range hon {
		for _, k := range h.Rivals {
			if q := lvGet(g, k); q != nil && q.Alive {
				return q
			}
		}
	}
	return nil
}

// fwCrash: a rival (or a drunken kinsman) charges in and picks a fight with you.
func fwCrash(g *Game, c *fwCer, hon []*lvPerson, rival *lvPerson) string {
	zn := g.Cur
	bd := g.W.burgByID[c.Burg]
	if zn == nil || bd == nil {
		return ""
	}
	r := fwRng(g, "crash", c.ID, g.Turn)
	var a *Actor
	if rival != nil {
		if z2, x := fwFind(g, rival); x != nil {
			if z2 != zn {
				fwDropActor(z2, x)
				zn.Actors = append(zn.Actors, x)
			}
			a = x
		}
	}
	if a == nil {
		cu := g.W.cultByID[bd.Culture]
		a = g.makeNPC("villager", bd, cu, r)
		if rival != nil {
			a.Name, a.Proper, a.Title = rival.Name, true, "rival"
			if rival.Burg == c.Burg {
				lvb(g).Bind[a.ID] = rival.Key
			}
		} else {
			a.Name, a.Proper, a.Title = "a drunken cousin", false, "brawler"
		}
		zn.Actors = append(zn.Actors, a)
		a.Set("fwnew")
		c.New = append(c.New, a.ID)
	}
	a.Angry, a.AI = true, "hostile"
	fwPlace(g, zn, a, g.P.X+2, g.P.Y)
	if rival != nil {
		return rival.Name + ", who has no love for the couple, pushes through the crowd with a fist raised: a brawl!"
	}
	return "A drunken cousin takes offence and lunges at you: a brawl!"
}

// fwCallOff: the wedding does not go on. Both are unmarried again; mood and news follow.
func fwCallOff(g *Game, c *fwCer, who *lvPerson, text string) []string {
	var a, b *lvPerson
	if len(c.Subj) >= 2 {
		a, b = lvGet(g, c.Subj[0]), lvGet(g, c.Subj[1])
	}
	day, yr := lvDay(g), lvYear(g)
	if a != nil && b != nil && a.Spouse == b.Key {
		a.Spouse, b.Spouse = "", ""
		a.Friends, b.Friends = lsDel(a.Friends, b.Key), lsDel(b.Friends, a.Key)
		lsMood(a, -25)
		lsMood(b, -25)
		ls := lsSt(g)
		ls.Mourn[a.Key], ls.Mourn[b.Key] = day+40, day+40
		a.addLife(yr, "marriage", "The wedding to "+b.Name+" was called off when the wanderer spoke against it.")
		b.addLife(yr, "marriage", "The wedding to "+a.Name+" was called off when the wanderer spoke against it.")
		lvSt(g).addNews(day, c.Burg, "The wedding of %s and %s was called off at the altar after an objection.", a.Name, b.Name)
	}
	fwBoost(g, []*lvPerson{a, b}, "attack", "You stopped their wedding.", -35)
	fwBoost(g, fwGuests(g, c), "insult", "You stopped a wedding.", -3)
	fwFaithRep(g, 2, " for speaking at the rite")
	fwFirst(g, "object", fmt.Sprintf("The wanderer %s stopped a wedding in %s by telling what they knew.", g.P.Name, fwBurgName(g, c.Burg)))
	g.gainXP(20)
	return []string{"You say it aloud: " + text + ". The silence breaks into shouting. The priest lays down the book; the wedding does not go on, and the pair will not forgive you soon."}
}
