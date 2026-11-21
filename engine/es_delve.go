package engine

import (
	"fmt"
	"sort"
	"strings"
)

// Delve stories (prefix ev). Every dungeon, crypt, mine, hideout and tower tells a short story of 3 to 5
// beats, room by room across its levels, built from the site's real data: the ruin's town and the event that
// ruined it, the dead ruler in the crypt, the order or villain that uses the hideout, the boss and its
// resistances, past lives from the hall of fame. A beat is a little scene (decor on free walkable tiles, a
// carved note, a readable letter or book) placed by a zone hook with a side rng, never g.R, and never changing
// the layout. Reading all beats pays a small bonus at the deepest hall. State g.ext("esv"), planet-local
// (planets.go unLocalKeys): the beats found per site key, so the journal lists them under the site name.

type evState struct {
	Found map[string]int    `json:"f,omitempty"` // site key -> bit mask of beats read
	Keys  map[string]string `json:"k,omitempty"` // site tag on the items -> site key
	Done  map[string]bool   `json:"d,omitempty"` // site keys whose finale was paid
}

func (g *Game) ev() *evState {
	s := g.ext("esv", func() any { return &evState{} }).(*evState)
	if s.Found == nil {
		s.Found = map[string]int{}
	}
	if s.Keys == nil {
		s.Keys = map[string]string{}
	}
	if s.Done == nil {
		s.Done = map[string]bool{}
	}
	return s
}

func init() {
	unLocalKeys = append(unLocalKeys, "esv") // planet-keyed by site key
	zoneHooks = append(zoneHooks, evZone)
	turnHooks = append(turnHooks, evTurn)
}

type evDecor struct {
	DX, DY int
	T      Terrain
}

// evScene is one beat: what you see, what you read, what it adds up to.
type evScene struct {
	Name   string   // the readable item
	Book   bool     // a book instead of a letter
	Look   string   // what you see (first line of the item text)
	Lines  []string // what it says (2 to 5 lines)
	Note   string   // a carved note on a free tile beside the scene
	Sum    string   // journal line
	Decor  []evDecor
	Loot   int    // tier of an extra loot item beside the scene (0 none)
	Goods  string // a grave good laid beside the scene
	Anchor string // stair boss key sarc ore
}

// evCtx is everything a site's beats are written from. All shared choices are made once from a site rng so
// every level, in any order of visiting, tells the same story.
type evCtx struct {
	g          *Game
	w          *WorldData
	s          *Site
	key        string
	wx, wy     int
	zx, zy     int
	p          int
	bd         *BurgData
	cu         *CultureData
	pk         *CulturePack
	cur        string
	boss       string
	place      string
	ruler      *PersonData
	fall       *EventData
	vl         *vlEntry
	order      *Order
	target     *PersonData
	leader     *PersonData
	leaderName string
	thief      string
	expYear    int
}

func evSiteKeyHash(key string) int { return int(HashStr("es:"+key)%900000) + 1 }

// evSiteLevel: the site a zone belongs to and its level (dungeon z, tower floor), or nil.
func evSiteLevel(g *Game, zn *Zone) (*Site, int) {
	if zn.Z <= towerBase-1 {
		s := g.siteAt(zn.WX, zn.WY, zn.ZX, zn.ZY)
		if s == nil || s.Kind != "tower" {
			return nil, 0
		}
		f := towerBase - zn.Z
		if f < 1 || f > s.Depth {
			return nil, 0
		}
		return s, f
	}
	if zn.Z < 1 {
		return nil, 0
	}
	s := g.siteAt(zn.WX, zn.WY, zn.ZX, zn.ZY)
	if s == nil || s.Kind != "dungeon" || zn.Z > s.Depth {
		return nil, 0
	}
	if zn.Kind == "dungeon" || zn.Kind == "crypt" && s.Ruin && zn.Z == 1 {
		return s, zn.Z
	}
	return nil, 0
}

func evNear(w *WorldData, p, q int) int {
	return abs(p%w.W-q%w.W) + abs(p/w.W-q/w.W)
}

func evCtxAt(g *Game, s *Site, wx, wy, zx, zy int) *evCtx {
	w := g.W
	p := w.Idx(wx, wy)
	c := &evCtx{g: g, w: w, s: s, key: siteKey(wx, wy, zx, zy), wx: wx, wy: wy, zx: zx, zy: zy, p: p, cur: g.Sk.Currency}
	c.bd = w.burgByID[w.Burg[p]]
	c.cu = w.cultByID[w.Culture[p]]
	if c.bd != nil && c.bd.Culture > 0 && w.cultByID[c.bd.Culture] != nil {
		c.cu = w.cultByID[c.bd.Culture]
	}
	if c.cu != nil {
		c.pk = c.cu.Pack
	}
	c.place = w.RegionName(p)
	if c.bd != nil {
		c.place = c.bd.Name
	}
	if th, ok := themes[s.Theme]; ok {
		c.boss = th.boss
	}
	if s.Theme == "hideout" || s.Outlaw != nil {
		c.boss = "chief"
	}
	r := NewRng(w.Seed + ":es:delve:" + c.key + ":site")
	// the ruin's fall: the last fall, war or disaster the chronicle puts on the town
	if c.bd != nil && (s.Ruin || !c.bd.Alive) {
		evs := w.eventsAt[c.bd.P]
		for i := len(evs) - 1; i >= 0 && c.fall == nil; i-- {
			if e := &w.Events[evs[i]]; e.Kind == "fall" || e.Kind == "disaster" || e.Kind == "war" || e.Kind == "capture" {
				c.fall = e
			}
		}
		if c.fall == nil && len(evs) > 0 {
			c.fall = &w.Events[evs[len(evs)-1]]
		}
	}
	// the dead ruler of the tomb
	st := w.State[p]
	if c.bd != nil && c.bd.State > 0 {
		st = c.bd.State
	}
	var dead []*PersonData
	for i := range w.People {
		pp := &w.People[i]
		if !pp.Alive && pp.Died > 0 && (pp.State == st && st != 0 || c.bd != nil && pp.Burg == c.bd.ID) && (pp.Role == "ruler" || pp.Title != "") {
			dead = append(dead, pp)
		}
	}
	if len(dead) == 0 {
		for i := range w.People {
			if pp := &w.People[i]; !pp.Alive && pp.Died > 0 && pp.State == st && st != 0 {
				dead = append(dead, pp)
			}
		}
	}
	sort.Slice(dead, func(i, j int) bool { return dead[i].Died < dead[j].Died })
	if len(dead) > 8 {
		dead = dead[len(dead)-8:]
	}
	if len(dead) > 0 {
		c.ruler = dead[r.Int(0, len(dead)-1)]
	}
	// the expedition's leader, a living person of a town nearby
	var near []*PersonData
	for i := range w.People {
		pp := &w.People[i]
		if pp.Alive && pp.Burg > 0 {
			if b := w.burgByID[pp.Burg]; b != nil && b.Alive && evNear(w, b.P, p) <= 14 {
				near = append(near, pp)
			}
		}
	}
	if len(near) > 0 {
		c.leader = near[r.Int(0, len(near)-1)]
		c.leaderName = c.leader.Name
	} else {
		c.leaderName = evName(c, r)
	}
	c.thief = evName(c, r)
	c.expYear = w.Year - r.Int(2, 11)
	// the villain or order that uses a hideout
	if c.bd != nil && c.bd.Alive {
		c.vl = g.vlAt(c.bd, zx, zy)
	}
	best := 99
	for _, o := range g.Orders() {
		if o.Kind != "cult" && o.Kind != "secret" && o.Kind != "thieves" {
			continue
		}
		if d := evNear(w, w.Idx(o.WX, o.WY), p); d <= 14 && d < best {
			c.order, best = o, d
		}
	}
	// a real bounty target
	if c.vl != nil {
		c.target = c.vl.P
	} else {
		var cs []*PersonData
		for i := range w.People {
			if pp := &w.People[i]; bhCriminal(pp) && !vlDead(g, pp.ID) {
				cs = append(cs, pp)
			}
		}
		if len(cs) > 0 {
			sort.Slice(cs, func(i, j int) bool { return cs[i].ID < cs[j].ID })
			c.target = cs[r.Int(0, len(cs)-1)]
		}
	}
	_ = w
	return c
}

func evName(c *evCtx, r *Rng) string {
	if c.cu != nil && len(c.cu.Given) > 0 {
		n := Pick(r, c.cu.Given)
		if len(c.cu.Houses) > 0 && r.Chance(0.6) {
			n += " " + Pick(r, c.cu.Houses)
		}
		return n
	}
	return Pick(r, []string{"Old Marn", "Tesk", "Ulli the Rope", "Hobb"})
}

// evPlan: the beats of a site's story in the order they are told.
func evPlan(c *evCtx) []string {
	s := c.s
	switch s.Theme {
	case "catacomb", "barrow":
		if s.Ruin {
			return []string{"camp", "lid", "thief", "wrong", "lair"}
		}
		if c.order != nil && c.order.Kind != "thieves" {
			return []string{"camp", "lid", "thief", "ritual", "lair"}
		}
		return []string{"camp", "lid", "thief", "survivor", "lair"}
	case "mine":
		return []string{"strike", "ledger", "collapse", "vein", "lair"}
	case "hideout":
		return []string{"poster", "ledger", "fence", "map", "lair"}
	case "trials":
		return []string{"graffiti", "log", "legend", "survivor", "lair"}
	case "abyss":
		return []string{"camp", "legend", "trail", "survivor", "lair"}
	case "magic":
		return []string{"camp", "trail", "ritual", "survivor", "lair"}
	case "arcane", "vault":
		return []string{"camp", "trail", "ritual", "survivor", "lair"}
	}
	return []string{"camp", "trail", "survivor", "lair"}
}

// evLevelOf spreads beats over the levels: the first at the way in, the last in the deepest hall.
func evLevelOf(i, n, depth int) int {
	if n <= 1 || depth <= 1 {
		return 1
	}
	if i == n-1 {
		return depth
	}
	return mini(depth, 1+i*depth/n)
}

func evZone(g *Game, zn *Zone, _ *Rng) {
	s, lv := evSiteLevel(g, zn)
	if s == nil {
		return
	}
	c := evCtxAt(g, s, zn.WX, zn.WY, zn.ZX, zn.ZY)
	plan := evPlan(c)
	depth := s.Depth
	r := NewRng(fmt.Sprintf("%s:es:delve:%s:%d", g.W.Seed, c.key, zn.Z))
	h := evSiteKeyHash(c.key)
	st := g.ev()
	var used [][2]int
	changed := false
	for i, id := range plan {
		if evLevelOf(i, len(plan), depth) != lv {
			continue
		}
		br := NewRng(fmt.Sprintf("%s:es:delve:%s:beat:%s", g.W.Seed, c.key, id))
		x, y, ok := evSpot(g, zn, r, id, used, lv, depth)
		if !ok {
			continue
		}
		sc := evBuild(c, id, br, zn, x, y, lv)
		if sc == nil {
			continue
		}
		if sc.Anchor == "key" || sc.Anchor == "ore" {
			x, y = evAnchorFix(zn, sc, x, y)
		}
		evPlace(g, zn, r, c, sc, i, x, y, h)
		used = append(used, [2]int{x, y})
		changed = true
	}
	if changed {
		st.Keys[fmt.Sprint(h)] = c.key
		g.computeLight(zn)
	}
}

func evSurvivorKey(zn *Zone) *Item {
	for _, it := range zn.Items {
		if it.Kind == "key" && it.Name == "iron key" {
			return it
		}
	}
	return nil
}

// evAnchorFix moves a key or ore anchored scene onto its anchor when it has one; the scene tile is a
// walkable tile beside the key or the vein.
func evAnchorFix(zn *Zone, sc *evScene, x, y int) (int, int) {
	switch sc.Anchor {
	case "key":
		if k := evSurvivorKey(zn); k != nil {
			return k.X, k.Y
		}
	case "ore":
		best, bx, by := 1<<30, x, y
		for yy := 1; yy < MH-1; yy++ {
			for xx := 1; xx < MW-1; xx++ {
				if zn.At(xx, yy) != TOre {
					continue
				}
				for _, d := range dirs8 {
					nx, ny := xx+d[0], yy+d[1]
					if evPlain(zn, nx, ny) && len(zn.ItemsAt(nx, ny)) == 0 && zn.ActorAt(nx, ny) == nil {
						if dd := abs(nx-x) + abs(ny-y); dd < best {
							best, bx, by = dd, nx, ny
						}
					}
				}
			}
		}
		return bx, by
	}
	return x, y
}

var evFloors = map[Terrain]bool{TFloor: true, TDirt: true, TCobble: true, TCarpet: true, TCrystalFloor: true, TFungus: true, TRubble: true, TBones: true}

// evPlain: a free natural floor tile a prop may lie on.
func evPlain(zn *Zone, x, y int) bool {
	if !zn.In(x, y) || !evFloors[zn.At(x, y)] || zn.Notes[y*MW+x] != "" {
		return false
	}
	if zn.Traps != nil {
		if _, ok := zn.Traps[y*MW+x]; ok {
			return false
		}
	}
	return true
}

func evWalkN(zn *Zone, x, y int) int {
	n := 0
	for _, d := range dirs8 {
		if zn.Walkable(x+d[0], y+d[1]) {
			n++
		}
	}
	return n
}

func evNearStair(zn *Zone, x, y, rad int) bool {
	for dy := -rad; dy <= rad; dy++ {
		for dx := -rad; dx <= rad; dx++ {
			switch zn.At(x+dx, y+dy) {
			case TUp, TDown, TUpdraft, TDoor, TDoorOpen, TLocked, TPortal:
				return true
			}
		}
	}
	return false
}

func evBossPos(zn *Zone) (int, int, bool) {
	for _, a := range zn.Actors {
		if a.Boss && a.HP > 0 {
			return a.X, a.Y, true
		}
	}
	return 0, 0, false
}

func evStairPos(zn *Zone) (int, int, bool) {
	for y := 0; y < MH; y++ {
		for x := 0; x < MW; x++ {
			if zn.At(x, y) == TUp {
				return x, y, true
			}
		}
	}
	return 0, 0, false
}

// evSpot finds the centre tile of a scene: a free tile in a room (most neighbours open), apart from
// the other scenes of the level and near its anchor (stairs, the boss, a sarcophagus, the key).
func evSpot(g *Game, zn *Zone, r *Rng, id string, used [][2]int, lv, depth int) (int, int, bool) {
	ax, ay, anchored := 0, 0, false
	rad := 99
	switch id {
	case "lair":
		if x, y, ok := evBossPos(zn); ok {
			ax, ay, anchored, rad = x, y, true, 9
		}
	case "camp", "strike", "poster", "graffiti":
		if lv == 1 {
			if x, y, ok := evStairPos(zn); ok {
				ax, ay, anchored, rad = x, y, true, 14
			}
		}
	case "lid", "wrong":
		var sx, sy []int
		for y := 0; y < MH; y++ {
			for x := 0; x < MW; x++ {
				if zn.At(x, y) == TSarcophagus {
					sx, sy = append(sx, x), append(sy, y)
				}
			}
		}
		if len(sx) > 0 {
			i := r.Int(0, len(sx)-1)
			ax, ay, anchored, rad = sx[i], sy[i], true, 3
		}
	case "survivor":
		if k := evSurvivorKey(zn); k != nil {
			return k.X, k.Y, true
		}
	case "vein":
		return evOreSpot(zn, r, used)
	}
	apart := func(x, y int) bool {
		for _, u := range used {
			if maxi(abs(u[0]-x), abs(u[1]-y)) < 8 {
				return false
			}
		}
		return true
	}
	for pass := 0; pass < 3; pass++ {
		need := []int{7, 5, 3}[pass]
		for i := 0; i < 500; i++ {
			x, y := r.Int(2, MW-3), r.Int(2, MH-3)
			if !evPlain(zn, x, y) || len(zn.ItemsAt(x, y)) > 0 || zn.ActorAt(x, y) != nil || evWalkN(zn, x, y) < need || evNearStair(zn, x, y, 2) {
				continue
			}
			if pass < 2 && !apart(x, y) {
				continue
			}
			if anchored {
				d := maxi(abs(x-ax), abs(y-ay))
				if d > rad+pass*6 || d < 2 && id != "lid" && id != "wrong" {
					continue
				}
				if x == ax && y == ay {
					continue
				}
			}
			return x, y, true
		}
	}
	return 0, 0, false
}

func evOreSpot(zn *Zone, r *Rng, used [][2]int) (int, int, bool) {
	var xs, ys []int
	for y := 1; y < MH-1; y++ {
		for x := 1; x < MW-1; x++ {
			if zn.At(x, y) != TOre {
				continue
			}
			for _, d := range dirs8 {
				if nx, ny := x+d[0], y+d[1]; evPlain(zn, nx, ny) && len(zn.ItemsAt(nx, ny)) == 0 && zn.ActorAt(nx, ny) == nil {
					xs, ys = append(xs, nx), append(ys, ny)
					break
				}
			}
		}
	}
	if len(xs) == 0 {
		return 0, 0, false
	}
	i := r.Int(0, len(xs)-1)
	return xs[i], ys[i], true
}

// evSafeBlock: the tile may become a solid prop without cutting any path (its open neighbours form one arc).
func evSafeBlock(zn *Zone, x, y int) bool {
	if !evPlain(zn, x, y) || len(zn.ItemsAt(x, y)) > 0 || zn.ActorAt(x, y) != nil || evNearStair(zn, x, y, 1) {
		return false
	}
	ring := [][2]int{{-1, -1}, {0, -1}, {1, -1}, {1, 0}, {1, 1}, {0, 1}, {-1, 1}, {-1, 0}}
	open, trans := 0, 0
	prev := zn.Walkable(x+ring[7][0], y+ring[7][1])
	for _, d := range ring {
		cur := zn.Walkable(x+d[0], y+d[1])
		if cur {
			open++
		}
		if cur != prev {
			trans++
		}
		prev = cur
	}
	return open >= 3 && trans <= 2
}

func evPlace(g *Game, zn *Zone, r *Rng, c *evCtx, sc *evScene, beat, x, y, h int) {
	for _, d := range sc.Decor {
		if (d.DX != 0 || d.DY != 0) && evPlain(zn, x+d.DX, y+d.DY) && len(zn.ItemsAt(x+d.DX, y+d.DY)) == 0 && zn.ActorAt(x+d.DX, y+d.DY) == nil {
			zn.Set(x+d.DX, y+d.DY, d.T)
		}
	}
	if sc.Note != "" {
		ring := [][2]int{{0, -1}, {1, 0}, {-1, 0}, {0, 1}, {-1, -1}, {1, -1}, {-1, 1}, {1, 1}, {0, -2}, {2, 0}, {-2, 0}, {0, 2}}
		for _, d := range ring {
			if nx, ny := x+d[0], y+d[1]; evSafeBlock(zn, nx, ny) {
				zn.Set(nx, ny, TInscription)
				zn.Notes[ny*MW+nx] = sc.Note
				break
			}
		}
	}
	text := sc.Look + "\n" + strings.Join(sc.Lines, "\n")
	var it *Item
	if sc.Book {
		it = g.MakeBook(sc.Name, text)
	} else {
		it = &Item{ID: g.newID(), Kind: "letter", Name: sc.Name, Glyph: '?', F: CYellow, Value: 6, Text: text}
	}
	it.X, it.Y, it.Ident = x, y, true
	it.Extra = map[string]int{"es:b": beat + 1, "es:s": h}
	it.Lore = ""
	zn.Items = append(zn.Items, it)
	if sc.Loot > 0 {
		l := g.Loot(r, sc.Loot)
		if lx, ly, ok := evBeside(zn, x, y); ok {
			l.X, l.Y = lx, ly
			zn.Items = append(zn.Items, l)
		}
	}
	if sc.Goods != "" {
		if lx, ly, ok := evBeside(zn, x, y); ok {
			zn.Items = append(zn.Items, &Item{ID: g.newID(), Kind: "trophy", Name: sc.Goods, Glyph: '*', F: CYellow, Value: 25 + zn.Level*4, X: lx, Y: ly,
				Lore: "Laid with the dead by the old rite."})
		}
	}
}

func evBeside(zn *Zone, x, y int) (int, int, bool) {
	for _, d := range dirs8 {
		nx, ny := x+d[0], y+d[1]
		if zn.Walkable(nx, ny) && zn.At(nx, ny).D().Water == 0 && len(zn.ItemsAt(nx, ny)) == 0 && zn.ActorAt(nx, ny) == nil && zn.At(nx, ny) != TDown && zn.At(nx, ny) != TUp {
			if _, tr := zn.Traps[ny*MW+nx]; !tr {
				return nx, ny, true
			}
		}
	}
	return 0, 0, false
}

// ---- reading ----

func evTurn(g *Game) {
	if g.P == nil || g.Cur == nil {
		return
	}
	st := g.ev()
	if len(st.Keys) == 0 {
		return
	}
	for _, it := range g.Cur.ItemsAt(g.P.X, g.P.Y) {
		if b := it.Extra["es:b"]; b > 0 && it.Extra["es:s"] > 0 {
			evFound(g, it, true)
		}
	}
	for _, it := range g.P.Inv {
		if it.Extra != nil && it.Extra["es:b"] > 0 {
			evFound(g, it, false)
		}
	}
}

// evFound records a beat as read. Standing on it shows the scene's look line once; picking it up counts too.
func evFound(g *Game, it *Item, glimpse bool) {
	st := g.ev()
	key := st.Keys[fmt.Sprint(it.Extra["es:s"])]
	if key == "" {
		return
	}
	bit := 1 << uint(it.Extra["es:b"]-1)
	if st.Found[key]&bit != 0 {
		return
	}
	st.Found[key] |= bit
	s := g.evSiteOf(key)
	if s == nil {
		return
	}
	look := strings.SplitN(it.Text, "\n", 2)[0]
	if glimpse && look != "" {
		g.Msg("{{Y|" + look + "}}")
	}
	g.Msg("{{C|A page of the story of " + s.Name + " falls into place.}}")
	p := g.W.Idx(g.Cur.WX, g.Cur.WY)
	g.addSecret(s.Name+": "+evSummary(it), "delve", p, g.W.Year)
	var wx, wy, zx, zy int
	fmt.Sscanf(key, "%d.%d.%d.%d", &wx, &wy, &zx, &zy)
	c := evCtxAt(g, s, wx, wy, zx, zy)
	plan := evPlan(c)
	all := 1<<uint(len(plan)) - 1
	if st.Found[key]&all == all && !st.Done[key] {
		st.Done[key] = true
		evFinale(g, c, plan)
	}
}

func evSummary(it *Item) string {
	lines := strings.Split(it.Text, "\n")
	if len(lines) > 1 {
		return strings.TrimSpace(Plain(lines[1]))
	}
	return it.Name
}

func (g *Game) evSiteOf(key string) *Site {
	var wx, wy, zx, zy int
	if _, err := fmt.Sscanf(key, "%d.%d.%d.%d", &wx, &wy, &zx, &zy); err != nil {
		return nil
	}
	return g.siteAt(wx, wy, zx, zy)
}

// evFinale: the whole story read, a small bonus: what the boss is weak to is learned, plus coin and a find.
func evFinale(g *Game, c *evCtx, plan []string) {
	weak, _ := evWeakness(g, c.boss)
	if len(weak) > 0 {
		k := c.boss
		if g.bw().Known[k] == nil {
			g.bw().Known[k] = map[string]bool{}
		}
		for _, t := range weak {
			g.bw().Known[k][t] = true
		}
	}
	r := NewRng(fmt.Sprintf("%s:es:delve:%s:finale", g.W.Seed, c.key))
	lvl := maxi(1, g.zoneLevel(c.wx, c.wy, 1)+c.s.Bonus)
	gold := 30*lvl + r.Int(10, 40)
	g.P.Gold += gold
	g.give(g.Loot(r, lvl/3+2))
	g.gainXP(40 + 10*lvl)
	extra := ""
	if len(weak) > 0 {
		extra = " You now know what the master of the place is weak to: " + strings.Join(weak, ", ") + "."
	}
	g.Msg(fmt.Sprintf("{{W|You have read the whole story of %s.}} Its last writer left something for the reader: %d %s and a find.%s", c.s.Name, gold, g.Sk.Currency, extra))
	g.deedOnce("es:"+c.key, "quest", fmt.Sprintf("The wanderer %s read the whole story of %s and learned how it ends.", g.P.Name, c.s.Name))
}
