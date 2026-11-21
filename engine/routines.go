package engine

import (
	"fmt"
	"hash/fnv"
	"strings"
)

// Daily routines for town folk (prefix rn). Always on, gated by the host pref "routines" (default true).
//
// aiAct calls rnAct right after snShelterAct (storms win). For a non-hostile townsperson in a town zone
// the hour of day picks an activity (sleep at home, wash at the well, work at the job site, eat, an
// errand, tavern or home at dusk, the temple or square on a holy or feast day) and a free destination
// tile from a per-zone site cache. They walk there one step every 2 to 4 game minutes (never faster than
// the player), at most rnBudget fresh path searches per turn, results cached for rnDMLife turns.
// Nothing here draws g.R. Saved: a day-keyed errand per actor id (capped), and the flags rnsleep, rntalk
// and rnaway on actors. Everything else is rebuilt on entering a zone.

const (
	rnBudget = 6  // distance-field searches per game minute
	rnErrCap = 48 // saved errands
	rnDMLife = 25 // turns a cached distance field stays valid
	rnDMCap  = 40
)

func rnHash(parts ...any) uint64 {
	h := fnv.New64a()
	fmt.Fprint(h, parts...)
	return h.Sum64()
}

// rnErr is a saved errand: day, kind (market well square visit done) and the actor visited.
type rnErr struct {
	D int    `json:"d"`
	K string `json:"k"`
	W int    `json:"w,omitempty"`
}

type rnClaim struct{ id, until int }
type rnMap struct {
	d    []int
	turn int
}

// rnState: only Day and Err are saved.
type rnState struct {
	Day int           `json:"d,omitempty"`
	Err map[int]rnErr `json:"e,omitempty"`

	tr      map[int]*rnAct
	zc      *rnZone
	dm      map[int]*rnMap
	dmZone  *Zone
	bud     int
	budTurn int
	claim   map[int]rnClaim
	lastMsg int
	festKey int
	fest    string
}

// rnAct is the transient routine state of one actor.
type rnAct struct {
	id         int
	gap        int // game minutes between steps, 2..4
	seen       int
	label      string
	key        string
	dest       int
	last       int
	blocked    int
	fails      int
	failUntil  int
	idx        int
	bad        map[int]int
	bedKind    string
	awakeUntil int
	talkUntil  int
	dwellUntil int
	with       string
	leg        int
	way        int
	wayDay     int
	errDone    int
	job        string
	shop       bool
	guard      bool
	shopLabel  string
	profDone   bool
	init       bool // started the talk we are in
	tmp        rnErr
}

func rnSt(g *Game) *rnState {
	st := g.ext("rn", func() any { return &rnState{} }).(*rnState)
	if st.claim == nil {
		st.claim = map[int]rnClaim{}
	}
	if st.tr == nil {
		st.tr = map[int]*rnAct{}
	}
	return st
}

func init() {
	unLocalKeys = append(unLocalKeys, "rn")
	enterHooks = append(enterHooks, rnEnter)
	turnHooks = append(turnHooks, rnTurn)
	talkHooks = append(talkHooks, rnTalk)
	rpPrefs = append(rpPrefs, "routines")
}

func rnEnter(g *Game, zn *Zone) {
	st := rnSt(g)
	st.zc, st.dm, st.dmZone = nil, nil, nil
	st.tr = map[int]*rnAct{}
	st.claim = map[int]rnClaim{}
}

// rnTurn: new day, drop yesterday's errands.
func rnTurn(g *Game) {
	if g == nil || g.P == nil {
		return
	}
	st := rnSt(g)
	if len(st.claim) > 300 {
		for k, c := range st.claim {
			if c.until <= g.Turn {
				delete(st.claim, k)
			}
		}
	}
	if d := g.Day(); d != st.Day {
		st.Day = d
		for k, e := range st.Err {
			if e.D != d {
				delete(st.Err, k)
			}
		}
	}
}

// rnTalk (talkHooks): talking to a sleeper wakes them.
func rnTalk(g *Game, a *Actor) []Option {
	if g == nil || a == nil {
		return nil
	}
	if a.Has("rnsleep") {
		delete(a.Flags, "rnsleep")
		g.rnStateFor(a).awakeUntil = g.Turn + 90
		g.Msg(g.The(a) + " wakes with a start and blinks at you.")
	}
	return nil
}

// ---------------------------------------------------------------- zone site cache

type rnZone struct {
	z     *Zone
	built int
	sites map[string][]int
}

var rnCaps = map[string]int{"roof": 400, "road": 300}

var rnPassT = map[Terrain]bool{TDoor: true, TDoorOpen: true, TDown: true, TUp: true, TGateArch: true, TGateClosed: true, TPortal: true, TUpdraft: true}
var rnBadT = map[Terrain]bool{TBridge: true, TTrap: true, TLava: true, TVoid: true, TChasm: true, TShallow: true, TDeep: true,
	TQuicksand: true, TTar: true, TFalls: true, TSpray: true, TGeyser: true, TFumarole: true, THotSpring: true}

func (c *rnZone) add(name string, i int) {
	lst := c.sites[name]
	cp := 64
	if v, ok := rnCaps[name]; ok {
		cp = v
	}
	if len(lst) >= cp {
		return
	}
	for _, v := range lst {
		if v == i {
			return
		}
	}
	c.sites[name] = append(lst, i)
}

// stand: a tile where someone may wait: walkable, off the zone border, not a door, stairs or gate, and
// not next to one (nobody stands in a doorway).
func (c *rnZone) stand(x, y int) bool {
	z := c.z
	if x < 2 || y < 2 || x >= MW-2 || y >= MH-2 {
		return false
	}
	t := z.At(x, y)
	if !t.D().Walk || rnPassT[t] || rnBadT[t] || z.WaterAt(x, y) > 0 {
		return false
	}
	for dy := -1; dy <= 1; dy++ {
		for dx := -1; dx <= 1; dx++ {
			if rnPassT[z.At(x+dx, y+dy)] {
				return false
			}
		}
	}
	return true
}

func (c *rnZone) near(name string, x, y int) {
	for _, d := range dirs8 {
		if c.stand(x+d[0], y+d[1]) {
			c.add(name, (y+d[1])*MW+x+d[0])
		}
	}
}

func (g *Game) rnZoneOf() *rnZone {
	st := rnSt(g)
	z := g.Cur
	if c := st.zc; c != nil && c.z == z && g.Turn-c.built < 200 {
		return c
	}
	c := &rnZone{z: z, built: g.Turn, sites: map[string][]int{}}
	st.zc = c
	for i, t := range z.T {
		x, y := i%MW, i/MW
		switch t {
		case TWell:
			c.near("well", x, y)
		case TStall:
			c.near("stall", x, y)
		case TAnvil:
			c.near("forge", x, y)
		case TAltar:
			c.near("altar", x, y)
		case TShelf:
			c.near("shelf", x, y)
		case TOre:
			c.near("ore", x, y)
		case TTree:
			c.near("tree", x, y)
		case TShallow, TDeep:
			c.near("water", x, y)
		case TCrop:
			if x >= 2 && y >= 2 && x < MW-2 && y < MH-2 {
				c.add("crop", i)
			}
		case TBed:
			if x >= 2 && y >= 2 && x < MW-2 && y < MH-2 {
				c.add("bed", i)
			}
		}
		if roofed(t) && t != TBed && c.stand(x, y) {
			c.add("roof", i)
		}
		if !roofed(t) && x%3 == 0 && y%2 == 0 && c.stand(x, y) {
			switch t {
			case TCobble, TRoad, TGrass, TDirt, TSand, TSnow:
				if abs(x-40) <= 8 && abs(y-12) <= 4 {
					c.add("square", i)
				}
				c.add("road", i)
			}
		}
	}
	// the tavern: roofed tiles around an innkeeper
	for _, a := range z.Actors {
		r := strings.ToLower(a.Role + " " + a.Title)
		if a.HP <= 0 || !(strings.Contains(r, "innkeeper") || strings.Contains(r, "tavern")) {
			continue
		}
		for dy := -4; dy <= 4; dy++ {
			for dx := -4; dx <= 4; dx++ {
				x, y := a.X+dx, a.Y+dy
				if z.In(x, y) && roofed(z.At(x, y)) && z.At(x, y) != TBed && c.stand(x, y) {
					c.add("tavern", y*MW+x)
				}
			}
		}
	}
	return c
}

// rnFree: nobody stands, sleeps on or is heading for tile i.
func (g *Game) rnFree(a *Actor, i int) bool {
	x, y := i%MW, i/MW
	if x == g.P.X && y == g.P.Y {
		return false
	}
	if o := g.Cur.ActorAt(x, y); o != nil && o != a {
		return false
	}
	if c, ok := rnSt(g).claim[i]; ok && c.id != a.ID && c.until > g.Turn {
		return false
	}
	return true
}

// rnDM: the distance field to a tile; nil when the per-turn search budget is spent.
func (g *Game) rnDM(dest int) []int {
	st := rnSt(g)
	if st.dmZone != g.Cur || st.dm == nil {
		st.dm, st.dmZone = map[int]*rnMap{}, g.Cur
	}
	if m := st.dm[dest]; m != nil && g.Turn-m.turn <= rnDMLife {
		return m.d
	}
	if st.budTurn != g.Turn {
		st.budTurn, st.bud = g.Turn, 0
	}
	if st.bud >= rnBudget {
		return nil
	}
	st.bud++
	if len(st.dm) >= rnDMCap {
		old, ot := -1, 1<<30
		for k, m := range st.dm {
			if m.turn < ot {
				old, ot = k, m.turn
			}
		}
		delete(st.dm, old)
	}
	d := g.distMap(dest%MW, dest/MW, false)
	st.dm[dest] = &rnMap{d, g.Turn}
	return d
}

// ---------------------------------------------------------------- per-actor state and profile

func (g *Game) rnStateFor(a *Actor) *rnAct {
	st := rnSt(g)
	s := st.tr[a.ID]
	if s == nil {
		s = &rnAct{id: a.ID, gap: 2 + a.ID%3, dest: -1, bad: map[int]int{}}
		st.tr[a.ID] = s
	}
	if !s.profDone {
		s.profDone = true
		txt := strings.ToLower(a.Title + " " + a.Role)
		if k, ok := lvb(g).Bind[a.ID]; ok {
			if p := lvSt(g).People[k]; p != nil {
				txt += " " + strings.ToLower(p.Job+" "+p.Role)
			}
		}
		s.guard = a.AI == "guard" || a.Role == "guard" || a.Role == "sergeant"
		s.shop = !s.guard && rnShopRoles[a.Role]
		s.job = "idle"
		for _, jk := range rnJobKeys {
			for _, w := range jk.words {
				if strings.Contains(txt, w) {
					s.job = jk.job
					break
				}
			}
			if s.job != "idle" {
				break
			}
		}
		if s.shop {
			s.shopLabel = "shop/shop"
			for dy := -3; dy <= 3; dy++ {
				for dx := -3; dx <= 3; dx++ {
					switch g.Cur.At(a.HomeX+dx, a.HomeY+dy) {
					case TStall:
						s.shopLabel = "shop/stall"
					case TAnvil:
						if s.shopLabel != "shop/stall" {
							s.shopLabel = "shop/forge"
						}
					}
				}
			}
		}
	}
	return s
}

func (g *Game) rnEligible(a *Actor) bool {
	if a == nil || a == g.P || a.HP <= 0 || a.Tpl != "npc" {
		return false
	}
	switch a.AI {
	case "town", "still", "guard":
	default:
		return false
	}
	if !rnRoles[a.Role] || a.Order != "" || a.Comp != nil || a.Quest != 0 || a.Leader != 0 || a.Angry || a.Boss ||
		a.SimMon != 0 || a.Teach != "" || a.Faction == "captive" || (g.Mode == "talk" && g.Talk == a) {
		return false
	}
	t := strings.ToLower(a.Title)
	for _, w := range rnSkipTitles {
		if strings.Contains(t, w) {
			return false
		}
	}
	for f := range a.Flags {
		for _, pre := range rnSkipFlagPrefixes {
			if strings.HasPrefix(f, pre) {
				return false
			}
		}
	}
	for _, q := range g.Quests {
		if q != nil && !q.Done && !q.Closed && q.GiverID == a.ID {
			return false
		}
	}
	return !g.Hostile(a)
}

// rnThreat: a hostile creature close by; the normal AI takes over.
func (g *Game) rnThreat(a *Actor) bool {
	for _, o := range g.Cur.Actors {
		if o != a && o.HP > 0 && o.AI != "follower" && (o.AI == "hostile" || o.Angry) && dist(a, o) < 14 {
			return true
		}
	}
	return false
}

// ---------------------------------------------------------------- the hook

// rnAct (ai.go hook after snShelterAct): returns true when it used the actor's action.
func (g *Game) rnAct(a *Actor) bool {
	if g.Cur == nil || g.P == nil || g.OnWorld || g.Cur.Kind != "town" || g.Cur.Burg <= 0 || a == g.P {
		return false
	}
	if !g.Pref("routines") {
		rnClear(a)
		return g.rnDrift(a)
	}
	if !g.rnEligible(a) {
		rnClear(a)
		return false
	}
	if g.rnThreat(a) {
		rnClear(a)
		return false
	}
	s := g.rnStateFor(a)
	s.seen = g.Turn
	if a.Has("rntalk") {
		if s.talkUntil > g.Turn {
			s.label = "chatting with " + s.with
			return true
		}
		delete(a.Flags, "rntalk")
		s.talkUntil = 0
		if s.init {
			s.init = false
			g.rnErrandDone(a, s)
		}
	}
	if s.failUntil > g.Turn {
		return g.rnDrift(a)
	}
	act := g.rnPlan(a, s)
	if act != "sleep" && a.Has("rnsleep") {
		delete(a.Flags, "rnsleep")
	}
	switch act {
	case "sleep":
		return g.rnDoSleep(a, s)
	case "shop":
		return g.rnDoShop(a, s)
	case "patrol", "post":
		return g.rnDoGuard(a, s, act)
	case "shelter":
		return g.rnDoShelter(a, s)
	case "errand":
		return g.rnDoErrand(a, s)
	}
	for _, site := range g.rnSiteOrder(a, s, act) {
		if len(g.rnZoneOf().sites[site]) > 0 && g.rnGoto(a, s, act, site, true) {
			return true
		}
	}
	return g.rnDrift(a)
}

func rnClear(a *Actor) {
	if a.Flags != nil {
		delete(a.Flags, "rnsleep")
		delete(a.Flags, "rntalk")
	}
}

func rnAway(a *Actor) {
	if abs(a.X-a.HomeX) > 6 || abs(a.Y-a.HomeY) > 4 {
		a.Set("rnaway")
	}
}

// rnDrift: someone our routine led away from home walks back once the routine has nothing for them.
func (g *Game) rnDrift(a *Actor) bool {
	if !a.Has("rnaway") {
		return false
	}
	if abs(a.X-a.HomeX) <= 3 && abs(a.Y-a.HomeY) <= 2 {
		delete(a.Flags, "rnaway")
		return false
	}
	s := g.rnStateFor(a)
	if g.rnWalkTo(a, s, a.HomeY*MW+a.HomeX, 2) == rnUnreach {
		delete(a.Flags, "rnaway")
		return false
	}
	s.label = "on their way home"
	return true
}

// ---------------------------------------------------------------- plan

func (g *Game) rnFestKind() string {
	st := rnSt(g)
	key := g.Day()*10000 + g.Cur.Burg
	if st.festKey != key {
		st.festKey, st.fest = key, ""
		if _, hd := g.rtTownHoly(); hd != nil {
			st.fest = "holy"
		} else if _, f := g.feastToday(); f != nil {
			st.fest = "feast"
		}
	}
	return st.fest
}

func (g *Game) rnRainy() bool {
	w := g.activeWeather()
	return w != nil && w.Kind != nil && w.Kind.Overlay == "rain"
}

func (g *Game) rnPlan(a *Actor, s *rnAct) string {
	m := (g.Turn + startMinutes) % DayLen
	if s.guard {
		if m >= 300 && m < 1260 {
			return "patrol"
		}
		return "post"
	}
	if s.shop {
		if m >= 390 && m < 1260 {
			return "shop"
		}
		return "sleep"
	}
	bed := 1260 + int(rnHash(a.ID, "bed")%100)
	up := 270 + int(rnHash(a.ID, "up")%60)
	if m >= bed || m < up {
		return "sleep"
	}
	mm := (m - int(rnHash(a.ID, "off")%40) + DayLen) % DayLen
	if g.rnRainy() && !roofed(g.Cur.At(a.X, a.Y)) && rnHash(a.ID, "rain", g.Day())%2 == 0 {
		return "shelter"
	}
	if g.rnFestKind() != "" && mm >= 540 && mm < 1200 {
		return "gather"
	}
	switch {
	case mm < 420:
		return "well"
	case mm < 720:
		return "work"
	case mm < 840:
		return "eat"
	case mm < 1080:
		return "errand"
	}
	return "dusk"
}

func (g *Game) rnSiteOrder(a *Actor, s *rnAct, act string) []string {
	switch act {
	case "well":
		return []string{"well"}
	case "work":
		if s.job == "idle" && rnHash(a.ID, "idle", g.Day())%5 >= 2 {
			return nil // an ordinary day of wandering
		}
		return rnWorkSites[s.job]
	case "eat":
		return []string{"tavern", "stall", "square"}
	case "dusk":
		if rnHash(a.ID, "dusk", g.Day())%2 == 0 {
			return []string{"tavern"}
		}
	case "gather":
		if g.rnFestKind() == "holy" {
			return []string{"altar", "square"}
		}
		return []string{"square", "stall"}
	}
	return nil
}

// ---------------------------------------------------------------- walking

const (
	rnMoving = iota
	rnArrived
	rnUnreach
)

// rnWalkTo: one gated step toward dest (a tile index); arrived within tol tiles.
func (g *Game) rnWalkTo(a *Actor, s *rnAct, dest, tol int) int {
	if dest < 0 {
		return rnUnreach
	}
	if maxi(abs(a.X-dest%MW), abs(a.Y-dest/MW)) <= tol {
		return rnArrived
	}
	rnSt(g).claim[dest] = rnClaim{a.ID, g.Turn + 30}
	if g.Turn-s.last < s.gap {
		return rnMoving
	}
	dm := g.rnDM(dest)
	if dm == nil {
		return rnMoving
	}
	if dm[a.Y*MW+a.X] >= 1<<20 {
		return rnUnreach
	}
	if g.stepToward(a, dm, false) {
		s.last = g.Turn
		s.blocked = 0
		rnAway(a)
	} else {
		s.blocked++
	}
	return rnMoving
}

func (s *rnAct) unreachable(g *Game, dest int) {
	s.bad[dest] = g.Turn + 80
	s.dest, s.key = -1, ""
	s.fails++
	if s.fails >= 3 {
		s.fails, s.failUntil = 0, g.Turn+60
	}
}

func rnPhrase(key string, id int) string {
	l := rnAt[key]
	if len(l) == 0 {
		return ""
	}
	return l[int(rnHash(id, key)%uint64(len(l)))]
}

// rnFidget: a small shuffle around a spot so workers do not stand like statues.
func (g *Game) rnFidget(a *Actor, s *rnAct, cx, cy, rad int) {
	if g.Turn-s.last < s.gap || rnHash(g.Turn, a.ID)%7 != 0 {
		return
	}
	d := dirs8[int(rnHash(g.Turn, a.ID, "f")%8)]
	nx, ny := a.X+d[0], a.Y+d[1]
	if maxi(abs(nx-cx), abs(ny-cy)) > rad || !g.rnZoneOf().stand(nx, ny) || !g.canEnter(a, nx, ny) {
		return
	}
	a.X, a.Y = nx, ny
	s.last = g.Turn
}

// rnPick chooses a free candidate tile for a site, hashed per actor and day so a crowd spreads out.
func (g *Game) rnPick(a *Actor, s *rnAct, cands []int) bool {
	n := len(cands)
	start := int(rnHash(a.ID, g.Day()) % uint64(n))
	for k := 0; k < n && k < 24; k++ {
		i := cands[(start+s.idx+k)%n]
		if s.bad[i] > g.Turn || !g.rnFree(a, i) {
			continue
		}
		s.dest = i
		s.blocked = 0
		rnSt(g).claim[i] = rnClaim{a.ID, g.Turn + 30}
		return true
	}
	return false
}

// rnGoto walks to a site of the zone and stays; false when the site cannot be used.
func (g *Game) rnGoto(a *Actor, s *rnAct, act, site string, fidget bool) bool {
	key := act + "/" + site
	cands := g.rnZoneOf().sites[site]
	if len(cands) == 0 {
		return false
	}
	if s.key != key || s.dest < 0 || !g.rnFree(a, s.dest) || s.blocked >= 6 {
		if s.key == key && s.blocked >= 6 {
			s.idx++
		}
		if !g.rnPick(a, s, cands) {
			s.key, s.dest = "", -1
			return false
		}
		s.key = key
	}
	switch g.rnWalkTo(a, s, s.dest, 0) {
	case rnArrived:
		s.label = rnPhrase(key, a.ID)
		if fidget {
			g.rnFidget(a, s, s.dest%MW, s.dest/MW, 1)
		}
	case rnUnreach:
		s.unreachable(g, s.dest)
		s.label = "looking for a way through"
	default:
		s.label = "on their way to " + rnNoun[site]
	}
	return true
}

// ---------------------------------------------------------------- sleep

func (g *Game) rnWakes(a *Actor) bool {
	if a.HP < a.MaxHP || a.Angry {
		return true
	}
	d := dist(a, g.P)
	n := g.ns().Noise
	return d <= 1 && n >= nsNoiseMuted+2 || d <= 4 && n >= nsNoiseLoud
}

func (g *Game) rnDoSleep(a *Actor, s *rnAct) bool {
	if a.Has("rnsleep") {
		if g.rnWakes(a) {
			delete(a.Flags, "rnsleep")
			s.awakeUntil = g.Turn + 90
			if g.visible(a.X, a.Y) {
				g.Msg(g.The(a) + " stirs awake.")
			}
			return g.rnDrift(a)
		}
		s.label = "sleeping, z z z"
		return true
	}
	if s.awakeUntil > g.Turn {
		return g.rnDrift(a)
	}
	c := g.rnZoneOf()
	if s.key != "sleep/bed" || s.dest < 0 || !g.rnFree(a, s.dest) || s.blocked >= 6 {
		home := a.HomeY*MW + a.HomeX
		best, bd, kind := -1, 1<<20, "bed"
		for _, name := range []string{"bed", "roof"} {
			rad := 30
			if name == "roof" {
				rad = 20
			}
			for _, i := range c.sites[name] {
				if s.bad[i] > g.Turn || !g.rnFree(a, i) {
					continue
				}
				if d := maxi(abs(i%MW-home%MW), abs(i/MW-home/MW)); d < bd && d <= rad {
					best, bd = i, d
				}
			}
			if best >= 0 {
				kind = name
				break
			}
		}
		if best < 0 {
			best, kind = home, "home"
		}
		s.dest, s.bedKind, s.key, s.blocked = best, kind, "sleep/bed", 0
		rnSt(g).claim[best] = rnClaim{a.ID, g.Turn + 600}
	}
	switch g.rnWalkTo(a, s, s.dest, 0) {
	case rnArrived:
		if s.bedKind == "home" {
			s.label = rnPhrase("rest/home", a.ID)
		} else {
			a.Set("rnsleep")
			s.label = "sleeping, z z z"
		}
	case rnUnreach:
		s.unreachable(g, s.dest)
		return false
	default:
		if s.bedKind == "home" {
			s.label = "on their way home"
		} else {
			s.label = "heading to bed"
		}
	}
	return true
}

// ---------------------------------------------------------------- shop, guard, rain

func (g *Game) rnDoShop(a *Actor, s *rnAct) bool {
	home := a.HomeY*MW + a.HomeX
	switch g.rnWalkTo(a, s, home, 2) {
	case rnArrived:
		s.label = rnPhrase(s.shopLabel, a.ID)
		g.rnFidget(a, s, a.HomeX, a.HomeY, 2)
	case rnUnreach:
		return false
	default:
		s.label = "on their way to their stall"
	}
	return true
}

func (g *Game) rnDoGuard(a *Actor, s *rnAct, act string) bool {
	home := a.HomeY*MW + a.HomeX
	day := g.Day()
	if act == "post" || s.leg == 0 && s.way == 0 && s.wayDay == day {
		return g.rnGuardAt(a, s, home, 1, "patrol/post")
	}
	if s.wayDay != day {
		s.wayDay, s.way, s.leg = day, 0, 0
		var c []int
		for _, i := range g.rnZoneOf().sites["road"] {
			if d := maxi(abs(i%MW-a.HomeX), abs(i/MW-a.HomeY)); d >= 8 && d <= 16 {
				c = append(c, i)
			}
		}
		if len(c) > 0 {
			s.way = c[int(rnHash(a.ID, "way", day)%uint64(len(c)))]
		}
		if s.way == 0 {
			return g.rnGuardAt(a, s, home, 1, "patrol/post")
		}
	}
	if s.leg == 0 {
		return g.rnGuardAt(a, s, home, 1, "patrol/post")
	}
	return g.rnGuardAt(a, s, s.way, 1, "patrol/round")
}

// rnGuardAt walks a guard to one end of the beat, waits a while, then flips the leg.
func (g *Game) rnGuardAt(a *Actor, s *rnAct, dest, tol int, key string) bool {
	if s.way != 0 && !g.rnFree(a, s.way) && dest == s.way && maxi(abs(a.X-dest%MW), abs(a.Y-dest/MW)) <= 3 {
		s.leg, s.dwellUntil = 0, 0
		dest, key = a.HomeY*MW+a.HomeX, "patrol/post"
	}
	switch g.rnWalkTo(a, s, dest, tol) {
	case rnArrived:
		s.label = rnPhrase(key, a.ID)
		if s.way == 0 {
			return true
		}
		if s.dwellUntil == 0 {
			s.dwellUntil = g.Turn + 8 + int(rnHash(a.ID, g.Turn)%12)
		}
		if g.Turn >= s.dwellUntil {
			s.leg ^= 1
			s.dwellUntil = 0
		}
	case rnUnreach:
		s.way = 0
		s.leg = 0
		return false
	default:
		s.label = rnPhrase("patrol/round", a.ID)
	}
	return true
}

func (g *Game) rnDoShelter(a *Actor, s *rnAct) bool {
	if roofed(g.Cur.At(a.X, a.Y)) {
		s.label = rnPhrase("shelter/roof", a.ID)
		return true
	}
	dm := g.snShelterMap(false)
	if v := dm[a.Y*MW+a.X]; v >= 15 {
		return false
	}
	s.label = "heading for cover"
	if g.Turn-s.last >= s.gap && g.stepToward(a, dm, false) {
		s.last = g.Turn
		rnAway(a)
	}
	return true
}

// ---------------------------------------------------------------- errands

func (g *Game) rnErrand(a *Actor, s *rnAct) rnErr {
	st := rnSt(g)
	day := g.Day()
	if s.errDone == day+1 {
		return rnErr{D: day, K: "done"}
	}
	if e, ok := st.Err[a.ID]; ok && e.D == day {
		return e
	}
	if s.tmp.D == day && s.tmp.K != "" {
		return s.tmp
	}
	h := rnHash(a.ID, "err", day) % 100
	e := rnErr{D: day}
	switch {
	case h < 30:
		e.K = "market"
		var c []*Actor
		for _, o := range g.Cur.Actors {
			if o != a && !o.Has("rnsleep") && g.rnEligible(o) {
				c = append(c, o)
			}
		}
		if len(c) > 0 {
			e.K, e.W = "visit", c[int(rnHash(a.ID, "who", day)%uint64(len(c)))].ID
		}
	case h < 60:
		e.K = "market"
	case h < 85:
		e.K = "well"
	default:
		e.K = "square"
	}
	if st.Err == nil {
		st.Err = map[int]rnErr{}
	}
	if len(st.Err) >= rnErrCap {
		for k, x := range st.Err {
			if x.D != day {
				delete(st.Err, k)
			}
		}
	}
	if len(st.Err) < rnErrCap {
		st.Err[a.ID] = e
	} else {
		s.tmp = e
	}
	return e
}

func (g *Game) rnErrandDone(a *Actor, s *rnAct) {
	day := g.Day()
	s.errDone = day + 1
	s.dwellUntil = 0
	if e, ok := rnSt(g).Err[a.ID]; ok && e.D == day {
		e.K = "done"
		rnSt(g).Err[a.ID] = e
	}
}

func (g *Game) rnDoErrand(a *Actor, s *rnAct) bool {
	e := g.rnErrand(a, s)
	switch e.K {
	case "visit":
		return g.rnVisit(a, s, e)
	case "market", "well", "square":
		sites := map[string][]string{"market": {"stall", "square"}, "well": {"well", "stall"}, "square": {"square"}}[e.K]
		for _, site := range sites {
			if len(g.rnZoneOf().sites[site]) == 0 || !g.rnGoto(a, s, "errand", site, true) {
				continue
			}
			if s.dest >= 0 && a.Y*MW+a.X == s.dest {
				if s.dwellUntil == 0 {
					s.dwellUntil = g.Turn + 20 + int(rnHash(a.ID, "dw", g.Day())%30)
				} else if g.Turn >= s.dwellUntil {
					g.rnErrandDone(a, s)
				}
			}
			return true
		}
		g.rnErrandDone(a, s)
	}
	return g.rnDrift(a)
}

func (g *Game) rnVisit(a *Actor, s *rnAct, e rnErr) bool {
	var o *Actor
	for _, x := range g.Cur.Actors {
		if x.ID == e.W {
			o = x
		}
	}
	if o == nil || o.HP <= 0 || o.Has("rnsleep") || !g.rnEligible(o) {
		g.rnErrandDone(a, s)
		return g.rnDrift(a)
	}
	if dist(a, o) <= 1 {
		s.talkUntil = g.Turn + 8 + int(rnHash(a.ID, o.ID, g.Day())%8)
		s.init = true
		os := g.rnStateFor(o)
		os.talkUntil, os.with = s.talkUntil, a.Name
		s.with = o.Name
		a.Set("rntalk")
		o.Set("rntalk")
		if st := rnSt(g); g.visible(a.X, a.Y) && dist(a, g.P) <= 8 && g.Turn-st.lastMsg > 25 {
			st.lastMsg = g.Turn
			l := rnStop[int(rnHash(a.ID, o.ID, g.Turn)%uint64(len(rnStop)))]
			g.Msg("{{K|" + fmt.Sprintf(l, g.The(a), g.The(o)) + "}}")
		}
		s.label = "chatting with " + o.Name
		return true
	}
	switch g.rnWalkTo(a, s, o.Y*MW+o.X, 1) {
	case rnUnreach:
		g.rnErrandDone(a, s)
		return g.rnDrift(a)
	}
	s.label = "on their way to see " + o.Name
	return true
}

// ---------------------------------------------------------------- look text

// rnLookTag completes "They are ...": what the actor is doing now ("" when nothing is known).
func (g *Game) rnLookTag(a *Actor) string {
	if g == nil || a == nil || g.Cur == nil || !g.Pref("routines") {
		return ""
	}
	if a.Has("rnsleep") {
		return "sleeping, z z z"
	}
	s := rnSt(g).tr[a.ID]
	if s == nil || g.Turn-s.seen > 6 {
		return ""
	}
	return s.label
}

// rnTalking: the neighbour an actor has stopped to talk with, for barks.go chatter (nil when none).
func (g *Game) rnTalking(a *Actor) *Actor {
	if a == nil || !a.Has("rntalk") || g.Cur == nil {
		return nil
	}
	for _, o := range g.Cur.Actors {
		if o != a && o.HP > 0 && o.Has("rntalk") && dist(a, o) <= 1 {
			return o
		}
	}
	return nil
}
