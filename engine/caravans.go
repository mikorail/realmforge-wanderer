package engine

import (
	"fmt"
	"sort"
	"strings"
)

// Visible trade caravans (prefix cv). A caravan is a named company with a home town, a destination
// town, wagons, guards and a cargo of real trade goods (tradeways.go goods, culture goods from
// cultgoods.go). Its whole life is a PURE FUNCTION of (seed, game hour, caravan id): the registry
// (cvReg) picks each company's route over the real roads (worldsim.go route), and `at(hour)` says where
// it is on the day clock: out to the destination, a three-day layover, back home, a layover, again.
// Nothing about a caravan's movement is stored. Only what the player did is saved, in g.ext("cv")
// (planet-local): trips robbed, caravans destroyed, goods bought, zones already populated, and the
// player's own passage or guard contract.
//
// Seen: a section sign on the world map within three parasangs of you, with an info line when the cursor
// is on it. Met: when you walk into the sub-zone it camps in, the wagons (crate tiles with cargo items
// beside them), the master, drivers and guards are placed by an enter hook with a side rng
// (seed:cv:spawn:key:zone). Talk to the master: trade at caravan prices, buy passage, hire on as a
// guard (the pay comes on arrival), ride on (an auto-travel loop: days pass, raiders may strike, you
// arrive on the destination's parasang). Ambush: strike anyone of the company and every one of them
// turns on you; the first blow is a crime (bounty and rep loss) unless their realm is at war with
// yours (then it is a lawful prize). NPC bandits raid a trip now and then (derived, flavour: fewer
// wagons and guards after the raid parasang), and wars or plagues over the route cancel the trip.
//
// Hooks outside this file (listed in the report): cvWorldGlyph, cvInfo in render.go drawWorld.

func init() {
	unLocalKeys = append(unLocalKeys, "cv")
	enterHooks = append(enterHooks, cvEnter)
	turnHooks = append(turnHooks, cvTurn)
	talkHooks = append(talkHooks, cvTalk)
	killHooks = append(killHooks, cvKill)
	sideHooks = append(sideHooks, cvSide)
}

// ---------------------------------------------------------------- saved state (player-made changes only)

type cvJoin struct {
	ID     int  `json:"id"`
	Trip   int  `json:"t"`
	Guard  bool `json:"g,omitempty"`
	Fee    int  `json:"f,omitempty"`
	Pay    int  `json:"p,omitempty"`
	Start  int  `json:"s"`
	Fought int  `json:"ft,omitempty"`
}

type cvState struct {
	Robbed map[string]int   `json:"rob"`  // "id:trip" -> hour the player struck the company
	Dead   map[int]int      `json:"dead"` // caravan id -> hour it was wiped out
	Sold   map[string]int   `json:"sold"` // "id:trip:good" -> units the player bought
	Seen   map[string]bool  `json:"seen"` // "id:trip:step" -> its zone has been populated
	Wagons map[string][]int `json:"wag"`  // zone key -> wagon tiles to clear when it breaks camp
	Join   *cvJoin          `json:"join,omitempty"`
	Rides  int              `json:"rides,omitempty"`
	Looted int              `json:"loot,omitempty"`
	Wiped  int              `json:"wiped,omitempty"`
}

func (g *Game) cv() *cvState {
	s := g.ext("cv", func() any { return &cvState{} }).(*cvState)
	if s.Robbed == nil {
		s.Robbed = map[string]int{}
	}
	if s.Dead == nil {
		s.Dead = map[int]int{}
	}
	if s.Sold == nil {
		s.Sold = map[string]int{}
	}
	if s.Seen == nil {
		s.Seen = map[string]bool{}
	}
	if s.Wagons == nil {
		s.Wagons = map[string][]int{}
	}
	return s
}

// ---------------------------------------------------------------- the pure registry

type cvGoodRow struct {
	Good *tqGood
	Qty  int
}

type cvCar struct {
	ID     int
	Name   string
	Master string
	Home   *BurgData
	Dest   *BurgData
	Path   []int // parasangs from home to destination
	Hpp    int   // hours to cross one parasang
	Lay    int   // hours resting at each end
	L      int   // hours of one leg
	Per    int   // hours of a full round trip
	Phase  int
	Wagons int
	Guards int
	Goods  []cvGoodRow
}

type cvRegistry struct {
	list []*cvCar
	byID map[int]*cvCar
}

var cvCache = map[*WorldData]*cvRegistry{}

func cvWar(w *WorldData, a, b int) bool {
	if a <= 0 || b <= 0 || a == b {
		return false
	}
	if s := w.stateByID[a]; s != nil {
		for _, x := range s.Wars {
			if x == b {
				return true
			}
		}
	}
	if s := w.stateByID[b]; s != nil {
		for _, x := range s.Wars {
			if x == a {
				return true
			}
		}
	}
	return false
}

func cvReg(g *Game) *cvRegistry {
	w := g.W
	if r, ok := cvCache[w]; ok {
		return r
	}
	reg := &cvRegistry{byID: map[int]*cvCar{}}
	cvCache[w] = reg
	var burgs []*BurgData
	for i := range w.Burgs {
		b := &w.Burgs[i]
		if b.Alive && (b.Layer == "" || b.Layer == "surface") && b.P >= 0 && b.P < len(w.Biome) && !w.IsWater(b.P) {
			burgs = append(burgs, b)
		}
	}
	if len(burgs) < 2 {
		return reg
	}
	sort.Slice(burgs, func(i, j int) bool { return burgs[i].ID < burgs[j].ID })
	n := clampi(len(burgs)/4, 3, 36)
	km := tvDefaultKm
	if w.KmPer > 0 {
		km = int(w.KmPer)
	}
	hpp := maxi(24, km) // a laden wagon makes about 24 km a day
	rs := &wsState{}
	goodsHere := g.tqGoodsHere()
	for i := 0; i < n; i++ {
		h := HashStr(fmt.Sprintf("%s:cv:%d", w.Seed, i))
		home := burgs[int(h%uint64(len(burgs)))]
		type cand struct {
			b *BurgData
			d int
		}
		var cs []cand
		for _, b := range burgs {
			if b.ID == home.ID {
				continue
			}
			d := wsCheb(w, home.P, b.P)
			if d < 2 || d > 14 || cvWar(w, home.State, b.State) {
				continue
			}
			cs = append(cs, cand{b, d})
		}
		sort.Slice(cs, func(a, b int) bool {
			if cs[a].d != cs[b].d {
				return cs[a].d < cs[b].d
			}
			return cs[a].b.ID < cs[b].b.ID
		})
		if len(cs) > 8 {
			cs = cs[:8]
		}
		var path []int
		var dest *BurgData
		for k := 0; k < len(cs); k++ {
			c := cs[(int(h>>8%uint64(len(cs)))+k)%len(cs)]
			pth := rs.route(g, home.P, c.b.P)
			if len(pth) >= 2 && len(pth) <= 18 {
				path, dest = pth, c.b
				break
			}
		}
		if dest == nil {
			continue
		}
		c := &cvCar{ID: i, Home: home, Dest: dest, Path: path, Hpp: hpp, Lay: 72}
		c.L = (len(path) - 1) * hpp
		c.Per = 2 * (c.L + c.Lay)
		c.Phase = int((h >> 24) % uint64(c.Per))
		c.Wagons = 2 + int((h>>16)%3)
		c.Guards = 2 + int((h>>20)%4)
		if home.Garrison > 150 {
			c.Guards++
		}
		rr := NewRng(fmt.Sprintf("%s:cv:master:%d", w.Seed, i))
		c.Master = g.cultureName(w.cultByID[home.Culture], rr)
		first := strings.Fields(c.Master)[0]
		switch (h >> 32) % 4 {
		case 0:
			c.Name = fmt.Sprintf("the %s Wagon Company", home.Name)
		case 1:
			c.Name = fmt.Sprintf("%s's Wagons", first)
		case 2:
			c.Name = fmt.Sprintf("the %s-%s Road Company", home.Name, dest.Name)
		default:
			c.Name = fmt.Sprintf("the %s Trading Train", home.Name)
		}
		// cargo: what the home town makes best, nothing banned at either end
		type scored struct {
			d *tqGood
			s float64
		}
		var sc []scored
		for _, d := range goodsHere {
			if g.tqBanned(home.State, d) || g.tqBanned(dest.State, d) {
				continue
			}
			sc = append(sc, scored{d, w.Sector(home, d.Prod)*3 + float64(HashStr(fmt.Sprintf("%s:cv:g:%d:%s", w.Seed, i, d.ID))%100)/100})
		}
		sort.Slice(sc, func(a, b int) bool {
			if sc[a].s != sc[b].s {
				return sc[a].s > sc[b].s
			}
			return sc[a].d.ID < sc[b].d.ID
		})
		for k := 0; k < len(sc) && k < 3; k++ {
			c.Goods = append(c.Goods, cvGoodRow{sc[k].d, 6 + int(HashStr(fmt.Sprintf("%s:cv:q:%d:%d", w.Seed, i, k))%8) + c.Wagons*2})
		}
		reg.list = append(reg.list, c)
		reg.byID[c.ID] = c
	}
	return reg
}

type cvPos struct {
	P      int  // parasang
	I      int  // index into Path
	Trip   int  // 2*cycle + leg (rest at the far end belongs to leg 0)
	Leg    int  // 0 out, 1 back
	Left   int  // hours until it moves on
	Moving bool // on the road
	Rest   bool // laying over at a town
}

// at is the pure position of the caravan at a game hour.
func (c *cvCar) at(h int) cvPos {
	if h < 0 {
		h = 0
	}
	u0 := h + c.Phase
	cyc, u := u0/c.Per, u0%c.Per
	n := len(c.Path)
	var pos cvPos
	switch {
	case u < c.L:
		i := u / c.Hpp
		pos = cvPos{P: c.Path[i], I: i, Moving: true, Left: c.Hpp - u%c.Hpp, Leg: 0}
	case u < c.L+c.Lay:
		pos = cvPos{P: c.Path[n-1], I: n - 1, Rest: true, Left: c.L + c.Lay - u, Leg: 0}
	case u < 2*c.L+c.Lay:
		v := u - c.L - c.Lay
		i := v / c.Hpp
		pos = cvPos{P: c.Path[n-1-i], I: n - 1 - i, Moving: true, Left: c.Hpp - v%c.Hpp, Leg: 1}
	default:
		pos = cvPos{P: c.Path[0], I: 0, Rest: true, Left: c.Per - u, Leg: 1}
	}
	pos.Trip = cyc*2 + pos.Leg
	return pos
}

// stepsDone is how many parasangs of its current leg the caravan has crossed.
func (c *cvCar) stepsDone(pos cvPos) int {
	if pos.Leg == 0 {
		return pos.I
	}
	return len(c.Path) - 1 - pos.I
}

func (c *cvCar) tripKey(trip int) string { return fmt.Sprintf("%d:%d", c.ID, trip) }

func (c *cvCar) bound(pos cvPos) *BurgData {
	if pos.Leg == 0 {
		return c.Dest
	}
	return c.Home
}

func (c *cvCar) origin(pos cvPos) *BurgData {
	if pos.Leg == 0 {
		return c.Home
	}
	return c.Dest
}

// cvHalted: a war or plague over the road holds the company in town (the trip is cancelled).
func (g *Game) cvHalted(p int) bool {
	for _, e := range g.ws().Ev {
		if (e.Kind == "invasion" || e.Kind == "plague") && e.covers(g.W, p) {
			return true
		}
	}
	return false
}

// cvActive is the position of a caravan that exists right now (not wiped out, not cancelled).
func (g *Game) cvActive(c *cvCar, h int) (cvPos, bool) {
	if at, ok := g.cv().Dead[c.ID]; ok && h < at+480 {
		return cvPos{}, false
	}
	pos := c.at(h)
	if pos.Moving && g.cvHalted(pos.P) {
		return pos, false
	}
	return pos, true
}

type cvCond struct {
	Wagons, Guards int
	Raid           string
	Robbed         bool
}

// cvCondOf: what NPC bandits and the player have done to this trip so far.
func (g *Game) cvCondOf(c *cvCar, pos cvPos) cvCond {
	cd := cvCond{Wagons: c.Wagons, Guards: c.Guards}
	n := len(c.Path)
	h := HashStr(fmt.Sprintf("%s:cv:raid:%d:%d", g.Seed, c.ID, pos.Trip))
	if h%100 < 24 && n > 2 {
		step := 1 + int((h>>8)%uint64(n-2))
		if c.stepsDone(pos) >= step {
			cd.Wagons = maxi(1, cd.Wagons-1)
			cd.Guards = maxi(1, cd.Guards-1)
			p := c.Path[step]
			if pos.Leg == 1 {
				p = c.Path[n-1-step]
			}
			cd.Raid = "bandits struck near " + g.wsPlace(p)
		}
	}
	if _, ok := g.cv().Robbed[c.tripKey(pos.Trip)]; ok {
		cd.Robbed = true
		cd.Wagons = 1
	}
	return cd
}

// ---------------------------------------------------------------- per-hour snapshot for the map

type cvSeen struct {
	c   *cvCar
	pos cvPos
}

var cvSnapW *WorldData
var cvSnapHour = -1
var cvSnapAt map[int][]cvSeen

func (g *Game) cvSnap() map[int][]cvSeen {
	h := g.Turn / 60
	if cvSnapW == g.W && cvSnapHour == h && cvSnapAt != nil {
		return cvSnapAt
	}
	cvSnapW, cvSnapHour, cvSnapAt = g.W, h, map[int][]cvSeen{}
	for _, c := range cvReg(g).list {
		if pos, ok := g.cvActive(c, h); ok {
			cvSnapAt[pos.P] = append(cvSnapAt[pos.P], cvSeen{c, pos})
		}
	}
	return cvSnapAt
}

func (g *Game) cvNear(p int) bool {
	return g.W != nil && wsCheb(g.W, p, g.W.Idx(g.WX, g.WY)) <= 3
}

// cvWorldGlyph is the world-map glyph hook: a caravan within three parasangs of you.
func (g *Game) cvWorldGlyph(p int, gl rune, f byte) (rune, byte) {
	if g.W == nil || !g.cvNear(p) || len(g.cvSnap()[p]) == 0 {
		return gl, f
	}
	return '§', CYellow
}

func (g *Game) cvLine(s cvSeen) string {
	c, pos := s.c, s.pos
	cd := g.cvCondOf(c, pos)
	where := "on the road toward " + c.bound(pos).Name
	if pos.Rest {
		where = "resting in " + g.wsPlace(pos.P)
	}
	line := fmt.Sprintf("{{Y|%s}} {{K|(%s leads %d wagons and %d guards, %s)}}", titleCase(c.Name), c.Master, cd.Wagons, cd.Guards, where)
	if cd.Raid != "" {
		line += " {{R|" + cd.Raid + "}}"
	}
	if cd.Robbed {
		line += " {{R|stripped by raiders}}"
	}
	if j := g.cv().Join; j != nil && j.ID == c.ID {
		line += " {{G|you ride with them}}"
	}
	return line
}

// cvInfo is the world-map info hook: caravans on the cursor's parasang that you can see.
func (g *Game) cvInfo(p int) []string {
	if g.W == nil || !g.cvNear(p) {
		return nil
	}
	var out []string
	for _, s := range g.cvSnap()[p] {
		out = append(out, g.cvLine(s))
	}
	return out
}

// ---------------------------------------------------------------- zones: the wagon train

func cvFlag(key string) string { return "cv:" + key }

// cvTag reads a caravan actor's flag: "cv:<id>:<trip>:<step>".
func cvTag(a *Actor) (key string, id int, ok bool) {
	for f, on := range a.Flags {
		if !on || !strings.HasPrefix(f, "cv:") {
			continue
		}
		var trip, step int
		if n, _ := fmt.Sscanf(f[3:], "%d:%d:%d", &id, &trip, &step); n == 3 {
			return f[3:], id, true
		}
	}
	return "", 0, false
}

func (g *Game) cvKeyOf(c *cvCar, pos cvPos) string {
	return fmt.Sprintf("%d:%d:%d", c.ID, pos.Trip, pos.I)
}

// cvZoneFor is the sub-zone of the parasang the company camps in.
func (g *Game) cvZoneFor(c *cvCar, pos cvPos) (int, int) {
	h := HashStr(fmt.Sprintf("%s:cv:zone:%d:%d:%d", g.Seed, c.ID, pos.Trip, pos.I))
	if g.W.Burg[pos.P] > 0 {
		i := int(h % 8)
		if i >= 4 {
			i++
		}
		return i % 3, i / 3
	}
	i := int(h % 9)
	return i % 3, i / 3
}

func cvDir(dx, dy int) string {
	s := ""
	switch {
	case dy < 0:
		s = "north"
	case dy > 0:
		s = "south"
	}
	switch {
	case dx < 0:
		s += "west"
	case dx > 0:
		s += "east"
	}
	if s == "" {
		return "here"
	}
	return s
}

// cvZoneActors lists the live caravan people in a zone.
func cvZoneActors(zn *Zone) []*Actor {
	var out []*Actor
	for _, a := range zn.Actors {
		if _, _, ok := cvTag(a); ok && a.HP > 0 {
			out = append(out, a)
		}
	}
	return out
}

// cvSweep removes caravan folk, cargo and wagons whose company has moved on.
func (g *Game) cvSweep(zn *Zone, quiet bool) {
	st := g.cv()
	reg := cvReg(g)
	h := g.Turn / 60
	p := g.W.Idx(zn.WX, zn.WY)
	gone := false
	keep := zn.Actors[:0]
	for _, a := range zn.Actors {
		key, id, ok := cvTag(a)
		if ok {
			c := reg.byID[id]
			stale := c == nil
			if !stale {
				pos, act := g.cvActive(c, h)
				stale = !act || pos.P != p || g.cvKeyOf(c, pos) != key
			}
			if stale && !a.Angry {
				gone = true
				continue
			}
		}
		keep = append(keep, a)
	}
	zn.Actors = keep
	if !gone {
		return
	}
	left := false
	for _, a := range zn.Actors {
		if _, _, ok := cvTag(a); ok {
			left = true
		}
	}
	if left {
		return
	}
	items := zn.Items[:0]
	for _, it := range zn.Items {
		if it.Extra != nil && it.Extra["cv"] == 1 {
			continue
		}
		items = append(items, it)
	}
	zn.Items = items
	if tiles := st.Wagons[zn.Key]; len(tiles) > 0 {
		for _, i := range tiles {
			if i >= 0 && i < len(zn.T) && zn.T[i] == TCrate {
				zn.T[i] = TDirt
			}
		}
		delete(st.Wagons, zn.Key)
		g.computeLight(zn)
	}
	if !quiet && g.Cur == zn {
		g.Msg("{{K|The wagons have rolled on; only ruts are left.}}")
	}
}

func cvEnter(g *Game, zn *Zone) {
	if g.W == nil || !g.W.In(zn.WX, zn.WY) || zn.Z != 0 {
		return
	}
	reg := cvReg(g)
	if len(reg.list) == 0 {
		return
	}
	st := g.cv()
	h := g.Turn / 60
	p := g.W.Idx(zn.WX, zn.WY)
	g.cvSweep(zn, true)
	g.cvCheckJoin(p)
	if zn.Kind != "surface" && zn.Kind != "outskirts" {
		return
	}
	for _, c := range reg.list {
		pos, ok := g.cvActive(c, h)
		if !ok || pos.P != p {
			continue
		}
		zx, zy := g.cvZoneFor(c, pos)
		if zn.ZX != zx || zn.ZY != zy {
			continue
		}
		key := g.cvKeyOf(c, pos)
		if st.Seen[key] {
			continue
		}
		if len(st.Seen) > 300 {
			st.Seen = map[string]bool{}
		}
		st.Seen[key] = true
		g.cvSpawn(zn, c, pos, key)
	}
}

func cvOpenTile(t Terrain) bool {
	switch t {
	case TGrass, TTallGrass, TSand, TDirt, TSnow, TAsh, TMoss, TRubble, TFloor, TCobble, TRoad:
		return true
	}
	return false
}

func (g *Game) cvFree(zn *Zone, x, y int) bool {
	if x < 3 || y < 2 || x > MW-4 || y > MH-3 || !cvOpenTile(zn.At(x, y)) || zn.ActorAt(x, y) != nil || (x == g.P.X && y == g.P.Y) {
		return false
	}
	for _, it := range zn.Items {
		if it.X == x && it.Y == y {
			return false
		}
	}
	return true
}

func (g *Game) cvCentre(zn *Zone, r *Rng) (int, int, bool) {
	best, bx, by := -1, 0, 0
	for i := 0; i < 60; i++ {
		x, y := r.Int(9, MW-10), r.Int(4, MH-5)
		n := 0
		for dx := -6; dx <= 6; dx++ {
			for dy := -2; dy <= 2; dy++ {
				if g.cvFree(zn, x+dx, y+dy) {
					n++
				}
			}
		}
		if n > best {
			best, bx, by = n, x, y
		}
	}
	return bx, by, best >= 40
}

func cvPlace(zn *Zone, a *Actor, cx, cy, rad int, r *Rng) bool {
	x, y, ok := zn.FreeSpot(r, func(x, y int) bool { return abs(x-cx) <= rad && abs(y-cy) <= rad })
	if !ok {
		return false
	}
	a.X, a.Y, a.HomeX, a.HomeY = x, y, x, y
	zn.Actors = append(zn.Actors, a)
	return true
}

func (g *Game) cvSpawn(zn *Zone, c *cvCar, pos cvPos, key string) {
	w := g.W
	st := g.cv()
	r := NewRng(fmt.Sprintf("%s:cv:spawn:%s:%s", g.Seed, key, zn.Key))
	cd := g.cvCondOf(c, pos)
	cx, cy, ok := g.cvCentre(zn, r)
	if !ok {
		return
	}
	flag := cvFlag(key)
	home := c.Home
	cu := w.cultByID[home.Culture]
	faction := fmt.Sprintf("cv:%d", home.State)
	pt := tgNewPainter(zn)
	var tiles []int
	x0 := cx - 3*cd.Wagons/2
	for i := 0; i < cd.Wagons; i++ {
		wx := x0 + 3*i
		if pt.set(wx, cy, TCrate, false) && pt.set(wx+1, cy, TCrate, false) {
			tiles = append(tiles, cy*MW+wx, cy*MW+wx+1)
			if cd.Robbed || len(c.Goods) == 0 {
				continue
			}
			row := c.Goods[i%len(c.Goods)]
			// the load, set out on the open tile in front of the wagon
			for _, tx := range []int{wx, wx + 1} {
				if g.cvFree(zn, tx, cy+1) {
					left := row.Qty - st.Sold[key+":"+row.Good.ID]
					if left <= 0 {
						break
					}
					n := mini(left, 3+r.Int(0, 4))
					it := g.tqNewItem(row.Good, n, g.tqPrice(home, row.Good, true), 0)
					it.Extra["cv"] = 1
					it.Text = "Bulk goods off the wagons of " + titleCase(c.Name) + "."
					it.X, it.Y = tx, cy+1
					zn.Items = append(zn.Items, it)
					break
				}
			}
		}
	}
	if len(tiles) > 0 {
		st.Wagons[zn.Key] = tiles
		pt.changed = true
	}
	if pt.changed {
		g.connect(zn)
		g.computeLight(zn)
	}
	// a coin box and a trinket of the home culture behind the first wagon
	if !cd.Robbed && g.cvFree(zn, x0, cy-1) {
		zn.Items = append(zn.Items, &Item{ID: g.newID(), Kind: "gold", Name: g.Sk.Currency, Glyph: '$', F: CYellow, Count: r.Int(25, 90) + cd.Wagons*15, X: x0, Y: cy - 1,
			Extra: map[string]int{"cv": 1}})
	}
	tag := func(a *Actor, role string) *Actor {
		a.Faction = faction
		a.Set(flag)
		a.Set(role)
		return a
	}
	m := g.makeNPC("merchant", home, cu, r)
	m.Name, m.Title = c.Master, "caravan master"
	m.Desc = fmt.Sprintf("Master of %s, bound from %s to %s with %d wagons.", c.Name, home.Name, c.Dest.Name, cd.Wagons)
	m.AI = "still"
	m.Gold = r.Int(60, 200)
	cvPlace(zn, tag(m, "cvmaster"), cx, cy+2, 3, r)
	for i := 0; i < cd.Wagons; i++ {
		d := g.makeNPC("villager", home, cu, r)
		d.Title = "wagon driver"
		d.Desc = fmt.Sprintf("A wagon driver of %s, on the road.", home.Name)
		d.AI, d.Gold = "still", r.Int(0, 12)
		cvPlace(zn, tag(d, "cvdriver"), cx, cy+2, 5, r)
	}
	for i := 0; i < cd.Guards; i++ {
		gd := g.makeNPC("guard", home, cu, r)
		gd.Title = "caravan guard"
		gd.Desc = fmt.Sprintf("A hired guard of %s, hand never far from the hilt.", c.Name)
		gd.AI = "wander"
		cvPlace(zn, tag(gd, "cvguard"), cx, cy+1, 6, r)
	}
	g.Msg("{{Y|You come upon " + titleCase(c.Name) + ", " + fmt.Sprint(cd.Wagons) + " wagons bound for " + c.bound(pos).Name + ".}}")
	if cd.Robbed {
		for _, a := range cvZoneActors(zn) {
			a.Angry = true
		}
	}
}

// ---------------------------------------------------------------- sidebar

func cvSide(g *Game) []string {
	if g.W == nil || g.Cur == nil || g.OnWorld || g.Cur.Z != 0 || !g.W.In(g.Cur.WX, g.Cur.WY) {
		return nil
	}
	reg := cvReg(g)
	if len(reg.list) == 0 {
		return nil
	}
	p := g.W.Idx(g.Cur.WX, g.Cur.WY)
	var out []string
	for _, s := range g.cvSnap()[p] {
		if j := g.cv().Join; j != nil && j.ID == s.c.ID {
			out = append(out, "{{G|With "+s.c.Name+"}}")
			continue
		}
		zx, zy := g.cvZoneFor(s.c, s.pos)
		if zx == g.Cur.ZX && zy == g.Cur.ZY {
			continue
		}
		out = append(out, "{{Y|"+titleCase(s.c.Name)+"}} {{K|camps "+cvDir(zx-g.Cur.ZX, zy-g.Cur.ZY)+"}}")
	}
	return out
}

// ---------------------------------------------------------------- talk: trade, passage, guard duty

func cvTalk(g *Game, a *Actor) []Option {
	if a.HP <= 0 || g.Cur == nil || g.Hostile(a) {
		return nil
	}
	key, id, ok := cvTag(a)
	if !ok {
		return nil
	}
	c := cvReg(g).byID[id]
	if c == nil {
		return nil
	}
	pos, act := g.cvActive(c, g.Turn/60)
	if !act {
		return nil
	}
	var out []Option
	switch {
	case a.Has("cvmaster"):
		out = append(out, Option{"{{W|Ask what the wagons carry.}}", func() { g.cvMarket(c, key) }})
		out = append(out, Option{"Ask where they are bound.", func() {
			g.talkMenu(fmt.Sprintf("\"We are %s, out of %s with %d wagons, bound for %s. %s\"", c.Name, c.Home.Name, g.cvCondOf(c, pos).Wagons, c.bound(pos).Name, g.cvRoadNote(c, pos)))
		}})
		out = append(out, g.cvJoinOptions(c, pos)...)
	case a.Has("cvguard"):
		out = append(out, Option{"Ask the guard about the road.", func() {
			g.talkMenu("\"Quiet enough, so far. " + g.cvRoadNote(c, pos) + "\"")
		}})
	}
	return out
}

func (g *Game) cvRoadNote(c *cvCar, pos cvPos) string {
	cd := g.cvCondOf(c, pos)
	switch {
	case cd.Robbed:
		return "Raiders stripped us, and I will not forget the faces."
	case cd.Raid != "":
		return "Bandits hit us near " + strings.TrimPrefix(cd.Raid, "bandits struck near ") + "; we lost a wagon and a guard."
	}
	return "The road has been kind this trip."
}

func (g *Game) cvPrices(c *cvCar, d *tqGood) (buy, sell int) {
	hb, db := g.tqPrice(c.Home, d, true), g.tqPrice(c.Dest, d, true)
	buy = maxi(1, (hb+db)/2)
	sell = maxi(1, g.tqPrice(c.Dest, d, false)*85/100)
	return
}

// cvMarket opens the caravan's stock: buy at caravan prices, sell goods you carry.
func (g *Game) cvMarket(c *cvCar, key string) {
	st := g.cv()
	var opts []Option
	lines := []string{"{{W|" + titleCase(c.Name) + "}} lay out their wares. Prices sit between " + c.Home.Name + " and " + c.Dest.Name + ".", ""}
	again := func() { g.cvMarket(c, key) }
	for _, row := range c.Goods {
		row := row
		d := row.Good
		left := row.Qty - st.Sold[key+":"+d.ID]
		buy, sell := g.cvPrices(c, d)
		name := g.tqName(d)
		if left > 0 {
			lines = append(lines, fmt.Sprintf("%s: %s each, %d left", name, g.cuFormat(buy), left))
			for _, n := range []int{1, 5} {
				n := n
				if left < n {
					continue
				}
				opts = append(opts, Option{fmt.Sprintf("Buy %d %s (%s)", n, name, g.cuFormat(buy*n)), func() {
					bought, spent := 0, 0
					for i := 0; i < n && row.Qty-st.Sold[key+":"+d.ID] > 0; i++ {
						if !g.cuSpend(buy) {
							break
						}
						g.tqAdd(d, 1, buy, 0, false)
						st.Sold[key+":"+d.ID]++
						bought++
						spent += buy
					}
					if bought == 0 {
						g.Msg("You can't afford that.")
					} else {
						g.Msg(fmt.Sprintf("You buy %d %s for %s.", bought, name, g.cuFormat(spent)))
					}
					again()
				}})
			}
		}
		if held := g.tqHeld(d); held > 0 {
			opts = append(opts, Option{fmt.Sprintf("Sell 1 of your %s (%s, you hold %d)", name, g.cuFormat(sell), held), func() {
				k, basis := g.tqTake(d, 1, false)
				if k > 0 {
					pay := g.cuEarn(sell)
					g.Msg(fmt.Sprintf("You sell %d %s for %s. {{K|(%+d on your cost)}}", k, name, pay, sell-basis))
				}
				again()
			}})
		}
	}
	for i, it := range g.cgGoods(c.Home.Culture) {
		if i >= 2 {
			break
		}
		it := it
		sk := fmt.Sprintf("%s:cg%d", key, i)
		if st.Sold[sk] > 0 {
			continue
		}
		price := maxi(1, it.Value*115/100)
		lines = append(lines, fmt.Sprintf("%s: %s", it.Name, g.cuFormat(price)))
		opts = append(opts, Option{fmt.Sprintf("Buy %s (%s)", it.Name, g.cuFormat(price)), func() {
			if !g.cuSpend(price) {
				g.Msg("You can't afford that.")
			} else {
				st.Sold[sk]++
				g.give(it)
				g.Msg("You buy " + it.Display() + ".")
			}
			again()
		}})
	}
	if len(opts) == 0 {
		lines = append(lines, "Nothing left to trade today.")
	}
	opts = append(opts, Option{"Done.", func() { g.jyClose() }})
	g.jyMenu("The wagons of "+titleCase(c.Name), lines, opts, func() { g.jyClose() })
}

func (g *Game) cvKmLeft(c *cvCar, pos cvPos) float64 {
	km := float64(tvDefaultKm)
	if g.W.KmPer > 0 {
		km = g.W.KmPer
	}
	return float64(len(c.Path)-1-c.stepsDone(pos)) * km
}

// cvFare is the passage price for km on a laden wagon (about 24 km a day).
func cvFare(km float64) int { return 4 + int(km/16) }

// cvGuardPay is the guard's pay for km: about twice the food a guard eats on the way (24 km a day).
func cvGuardPay(km float64) int { return 10 + int(km*3/8) }

func (g *Game) cvJoinOptions(c *cvCar, pos cvPos) []Option {
	st := g.cv()
	if j := st.Join; j != nil {
		if j.ID != c.ID {
			return nil
		}
		return []Option{
			{"{{G|Ride on with the caravan.}}", func() { g.cvRide() }},
			{"Part ways with the caravan here.", func() {
				st.Join = nil
				g.talkMenu("\"Your choice. The wagons roll on at dawn with or without you.\"")
			}},
		}
	}
	if !pos.Moving {
		return nil
	}
	to := c.bound(pos)
	km := g.cvKmLeft(c, pos)
	fee := cvFare(km)
	pay := cvGuardPay(km)
	trip := pos.Trip
	return []Option{
		{fmt.Sprintf("{{W|Buy passage to %s (%s).}}", to.Name, g.cuFormat(fee)), func() {
			if !g.cuSpend(fee) {
				g.talkMenu("\"Coin first. The wagons are not a charity.\"")
				return
			}
			st.Join = &cvJoin{ID: c.ID, Trip: trip, Fee: fee, Start: g.Turn}
			g.Msg(fmt.Sprintf("You pay %s and take a place beside the %s.", g.cuFormat(fee), titleCase(c.Name)))
			g.talkMenu("\"Ride at the back and keep out of the guards' way. The wagons go on whenever you are ready.\"")
		}},
		{fmt.Sprintf("{{W|Hire on as a guard to %s (%s on arrival).}}", to.Name, g.cuFormat(pay)), func() {
			if g.P.Level < 2 {
				g.talkMenu("\"You? Come back when you have blood on your blade.\"")
				return
			}
			st.Join = &cvJoin{ID: c.ID, Trip: trip, Guard: true, Pay: pay, Start: g.Turn}
			g.Msg("You sign on as a guard of " + titleCase(c.Name) + ".")
			g.talkMenu("\"Walk the flank and watch the brush. You are paid when the wagons stand in " + to.Name + ".\"")
		}},
	}
}

// cvCheckJoin drops the contract when you walk away from the company.
func (g *Game) cvCheckJoin(p int) {
	st := g.cv()
	j := st.Join
	if j == nil {
		return
	}
	c := cvReg(g).byID[j.ID]
	ok := false
	if c != nil {
		if pos, act := g.cvActive(c, g.Turn/60); act && pos.P == p && pos.Trip == j.Trip && pos.Moving {
			ok = true
		}
	}
	if !ok {
		st.Join = nil
		g.Msg("{{K|You are no longer with the caravan.}}")
	}
}

// ---------------------------------------------------------------- riding along (auto-travel)

func (g *Game) cvRide() {
	st := g.cv()
	w := g.W
	j := st.Join
	if j == nil {
		g.jyClose()
		return
	}
	c := cvReg(g).byID[j.ID]
	if c == nil {
		st.Join = nil
		g.jyClose()
		return
	}
	for i := 0; i < 40; i++ {
		pos, ok := g.cvActive(c, g.Turn/60)
		if !ok || pos.Trip != j.Trip || pos.Rest || !pos.Moving {
			g.cvArrive(c, j, pos, ok)
			return
		}
		g.tvPass(maxi(60, pos.Left*60))
		npos := c.at(g.Turn / 60)
		g.WX, g.WY = npos.P%w.W, npos.P/w.W
		g.CX, g.CY = g.WX, g.WY
		g.Known[npos.P] = true
		wsOnMove(g)
		lsAdvance(g)
		if g.P.HP <= 0 {
			return
		}
		if g.cvRideEvent(c, j, npos) {
			return
		}
	}
	g.cvRide()
}

// cvRideEvent runs the day's event at the new parasang; true when it opened a menu.
func (g *Game) cvRideEvent(c *cvCar, j *cvJoin, pos cvPos) bool {
	rng := NewRng(fmt.Sprintf("%s:cv:ride:%d:%d:%d", g.Seed, c.ID, j.Trip, pos.I))
	roll := rng.Float()
	switch {
	case roll < 0.22 && pos.Moving:
		foe := Pick(rng, []string{"road bandits", "hungry deserters", "a gang of raiders", "masked highwaymen"})
		g.jyMenu("Raiders on the road", []string{"{{R|" + foe + " break from the brush and rush the wagons!}} The guards form up around the master."}, []Option{
			{"{{W|Stand with the guards.}}", func() {
				mod := (g.P.Score(STR)-10)/2 + g.P.Level/3
				if j.Guard {
					mod += 2
				}
				if g.roll2d6Check(mod, dcMedium) {
					j.Fought++
					g.Msg("{{G|You and the guards cut the raiders down before they reach the wagons.}}")
					g.gainXP(20 + 6*g.P.Level)
				} else {
					g.Msg("{{R|The raiders are driven off, but not before you take a hurt.}}")
					g.hurt(g.P, rng.Int(2, 6), nil, foe)
					if j.Guard {
						j.Pay = j.Pay * 3 / 4
					}
				}
				g.cvRide()
			}},
			{"Keep your head down.", func() {
				if j.Guard {
					j.Pay /= 2
					g.Msg("{{R|The guards hold without you. The master will remember it when pay is counted.}}")
				} else {
					g.Msg("The guards beat the raiders off while you stay behind the wagons.")
				}
				g.cvRide()
			}},
		}, func() { g.cvRide() })
		return true
	case roll < 0.40:
		g.Msg("{{K|" + Pick(rng, []string{
			"The wagons creak on beside a dry ditch; the drivers sing.",
			"A day of grey drizzle; nobody speaks above the wheels.",
			"The master reads the sky and calls an early camp.",
			"You share the evening fire and a pot of something salty with the drivers.",
			"A toll-keeper waves the company through for a bribe the master pays without a word.",
		}) + "}}")
	}
	return false
}

func (g *Game) cvArrive(c *cvCar, j *cvJoin, pos cvPos, active bool) {
	st := g.cv()
	st.Join = nil
	st.Rides++
	dest := c.Dest
	if j.Trip%2 == 1 {
		dest = c.Home
	}
	p := g.W.Idx(g.WX, g.WY)
	if active {
		p = pos.P
	}
	arrived := p == dest.P
	if arrived {
		g.Msg(fmt.Sprintf("{{G|You arrive in %s with %s after %s on the road.}}", dest.Name, titleCase(c.Name), tvDuration(maxi(0, g.Turn-j.Start))))
		if j.Guard {
			pay := j.Pay + 4*j.Fought
			g.Msg(fmt.Sprintf("The master counts out your pay: %s.", g.cuEarn(pay)))
			g.gainXP(30 + 10*j.Fought)
			if g.Cur != nil {
				from := c.Home
				if j.Trip%2 == 1 {
					from = c.Dest
				}
				g.deed("quest", fmt.Sprintf("%s the %s guarded %s from %s to %s.", g.P.Name, g.Sk.Calling(g.Calling), titleCase(c.Name), from.Name, dest.Name), 0, 0)
			}
		}
	} else {
		g.Msg("{{y|The caravan is held up on the road and you part ways.}}")
	}
	for _, q := range c.Path {
		g.Known[q] = true
	}
	g.jyArrive(p)
}

// ---------------------------------------------------------------- ambush, theft, wipe-out

// cvAmbushCheck: the first blow against any of the company is a crime (or a lawful prize) and
// turns the whole company on the player.
func (g *Game) cvAmbushCheck(zn *Zone, extra *Actor) {
	if zn == nil {
		return
	}
	st := g.cv()
	reg := cvReg(g)
	var first *Actor
	hit := false
	for _, a := range zn.Actors {
		if _, _, ok := cvTag(a); !ok {
			continue
		}
		if first == nil {
			first = a
		}
		if a.Angry || a.HP <= 0 {
			hit = true
		}
	}
	if first == nil && extra != nil {
		first, hit = extra, true
	}
	if !hit || first == nil {
		return
	}
	for _, a := range zn.Actors {
		if _, _, ok := cvTag(a); ok && a.HP > 0 {
			a.Angry = true
		}
	}
	key, id, _ := cvTag(first)
	c := reg.byID[id]
	if c == nil {
		return
	}
	var tripN, step int
	fmt.Sscanf(key, "%d:%d:%d", &id, &tripN, &step)
	ck := fmt.Sprintf("%d:%d", id, tripN)
	if _, done := st.Robbed[ck]; done {
		return
	}
	st.Robbed[ck] = g.Turn / 60
	state := c.Home.State
	if state > 0 && g.HomeState > 0 && g.atWar(uwFac(g.HomeState), uwFac(state)) {
		g.Msg("{{G|" + g.uwRealm(state) + " is at war with your realm: their wagons are a lawful prize.}}")
		g.AddRep(uwFac(g.HomeState), 20, " for raiding an enemy caravan")
		return
	}
	g.uwAddBounty(state, 50+20*c.Wagons+4*g.P.Level, " (caravan robbery)")
	if state > 0 {
		g.AddRep(uwFac(state), -60, " for attacking their merchants")
	}
	g.deed("crime", fmt.Sprintf("%s the %s fell on %s on the road and was named a robber.", g.P.Name, g.Sk.Calling(g.Calling), titleCase(c.Name)), 0, 0)
}

func cvTurn(g *Game) {
	if g.W == nil || g.Cur == nil || g.P == nil || g.OnWorld || g.Cur.Z != 0 {
		return
	}
	// goods carried off the wagons in sight of the guards
	taken := false
	for _, it := range g.P.Inv {
		if it.Extra != nil && it.Extra["cv"] == 1 {
			it.Extra["cv"] = 2
			taken = true
			g.cv().Looted++
		}
	}
	zn := g.Cur
	folk := cvZoneActors(zn)
	if len(folk) == 0 {
		return
	}
	if taken {
		for _, a := range folk {
			if g.visible(a.X, a.Y) && (a.Has("cvguard") || a.Has("cvmaster")) {
				g.Msg("{{R|\"Thief! Stop that one!\"}}")
				for _, b := range folk {
					b.Angry = true
				}
				break
			}
		}
	}
	g.cvAmbushCheck(zn, nil)
	if g.Turn%60 == 0 {
		g.cvSweep(zn, false)
		if j := g.cv().Join; j != nil && g.W.In(zn.WX, zn.WY) {
			g.cvCheckJoin(g.W.Idx(zn.WX, zn.WY))
		}
	}
}

func cvKill(g *Game, d, src *Actor) {
	_, id, ok := cvTag(d)
	if !ok || g.Cur == nil {
		return
	}
	g.cvAmbushCheck(g.Cur, d)
	// the company is wiped out when its master and guards are all down
	alive := 0
	for _, a := range cvZoneActors(g.Cur) {
		if a != d && (a.Has("cvmaster") || a.Has("cvguard")) {
			alive++
		}
	}
	if alive > 0 {
		return
	}
	c := cvReg(g).byID[id]
	if c == nil {
		return
	}
	st := g.cv()
	if _, gone := st.Dead[c.ID]; gone && g.Turn/60 < st.Dead[c.ID]+480 {
		return
	}
	st.Dead[c.ID] = g.Turn / 60
	st.Wiped++
	g.Msg("{{R|" + titleCase(c.Name) + " is finished: its trade routes will stand empty for weeks.}}")
}
