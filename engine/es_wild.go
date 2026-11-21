package engine

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"sync"
)

// Wilderness traces (state key "esd", function prefix ed). The countryside holds the leftovers of real
// stories, one trace per roughly three wild zones: monster lairs with the traces of their victims,
// abandoned camps of caravans, armies and rival adventurers, hermit cabins of people from the life
// simulation, roadside shrines, milestones and gibbets, cursed ground, and trail signs of the world in
// motion. Each scene is built in a zone hook from a side rng (NewRng(seed+":es:..."), never g.R) on free
// natural tiles away from roads and towns, tied to one data record (monster id, rival slot, person key,
// event index, traveller id) kept in g.ext("esd") so the same scene reads the same way after save and load.
// Texts live in es_wild_data.go. Everything works without a culture pack.

type edScene struct {
	Kind    string `json:"k"` // lair hunt camp hermit shrine milestone gibbet fairy battle spring trail
	Ref     string `json:"r"` // the record: m:<monster> r:<slot> p:<person> e:<event> t:<traveller> b:<burg> o:<outbreak>
	Zone    string `json:"z"`
	X       int    `json:"x,omitempty"`
	Y       int    `json:"y,omitempty"`
	Day     int    `json:"d"`
	Idx     int    `json:"i,omitempty"` // tile index + 1 of the inscription whose reading pays off
	Book    string `json:"b,omitempty"` // name of a book whose reading pays off
	Near    bool   `json:"n,omitempty"` // pays when the player stands beside (X, Y)
	Pay     string `json:"p,omitempty"`
	PayKind string `json:"pk,omitempty"`
	PayP    int    `json:"pp,omitempty"`
	Taint   string `json:"t,omitempty"` // status applied on the tainted tiles
	Tiles   []int  `json:"ti,omitempty"`
	Paid    bool   `json:"pd,omitempty"`
	Fact    int    `json:"f,omitempty"`
}

type edState struct {
	List  []*edScene      `json:"l,omitempty"`
	Smoke map[string]bool `json:"sm,omitempty"`
	Said  map[string]bool `json:"sd,omitempty"`
}

func init() {
	unLocalKeys = append(unLocalKeys, "esd")
	zoneHooks = append(zoneHooks, edZone)
	enterHooks = append(enterHooks, edEnter)
	turnHooks = append(turnHooks, edTurn)
	talkHooks = append(talkHooks, edTalk)
	skAddStatus(&skStatusDef{ID: "edfey", Name: "Fey-dizzy", Glyph: 'f', AGI: -2, INT: -2, Speed: -10})
	skAddStatus(&skStatusDef{ID: "edchill", Name: "Dead's chill", Glyph: 'c', Hit: -2, DV: -2, WIL: -2})
}

func edSt(g *Game) *edState {
	s := g.ext("esd", func() any { return &edState{} }).(*edState)
	if s.Smoke == nil {
		s.Smoke = map[string]bool{}
	}
	if s.Said == nil {
		s.Said = map[string]bool{}
	}
	return s
}

var edPh = regexp.MustCompile(`\{[a-z_]+\}`)

// edFill replaces {key} with kv[key] and strips every placeholder left over, line by line.
func edFill(s string, kv map[string]string) string {
	for k, v := range kv {
		s = strings.ReplaceAll(s, "{"+k+"}", v)
	}
	s = edPh.ReplaceAllString(s, "")
	lines := strings.Split(s, "\n")
	for i, l := range lines {
		l = strings.Join(strings.Fields(l), " ")
		l = strings.NewReplacer(" .", ".", " ,", ",", " :", ":", " ;", ";").Replace(l)
		lines[i] = l
	}
	return strings.Join(lines, "\n")
}

func edUp(s string) string {
	if s == "" {
		return s
	}
	r := []rune(s)
	if r[0] >= 'a' && r[0] <= 'z' {
		r[0] -= 32
	}
	return string(r)
}

func edDot(s string) string { return strings.TrimRight(strings.TrimSpace(s), ".!? ") + "." }

func edFirstSentence(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexAny(s, ".!?"); i > 10 {
		return s[:i]
	}
	return strings.TrimRight(s, ".!? ")
}

func edTech(g *Game) bool {
	switch g.Genre {
	case "steampunk", "cyberpunk", "wasteland", "solarpunk", "biopunk":
		return true
	}
	return false
}

func edAgeText(hours int) string {
	switch {
	case hours < 36:
		return "a day"
	case hours < 24*30:
		return fmt.Sprintf("%d days", maxi(2, (hours+12)/24))
	}
	return "many weeks"
}

// ---------------------------------------------------------------- world helpers

var edBurgCache struct {
	sync.Mutex
	w    *WorldData
	list []*BurgData
}

func edBurgs(w *WorldData) []*BurgData {
	edBurgCache.Lock()
	defer edBurgCache.Unlock()
	if edBurgCache.w == w && edBurgCache.list != nil {
		return edBurgCache.list
	}
	var out []*BurgData
	for i := range w.Burgs {
		b := &w.Burgs[i]
		if b.Alive && (b.Layer == "" || b.Layer == "surface") && b.P >= 0 && b.P < len(w.Biome) && !w.IsWater(b.P) {
			out = append(out, b)
		}
	}
	if out == nil {
		out = []*BurgData{}
	}
	edBurgCache.w, edBurgCache.list = w, out
	return out
}

// edNearest returns the n nearest living towns of a parasang, nearest first, with their distances.
func edNearest(w *WorldData, p, n int) ([]*BurgData, []int) {
	type bd struct {
		b *BurgData
		d int
	}
	var all []bd
	for _, b := range edBurgs(w) {
		all = append(all, bd{b, wsCheb(w, b.P, p)})
	}
	sort.SliceStable(all, func(i, j int) bool {
		if all[i].d != all[j].d {
			return all[i].d < all[j].d
		}
		return all[i].b.ID < all[j].b.ID
	})
	var bs []*BurgData
	var ds []int
	for i := 0; i < len(all) && i < n; i++ {
		bs = append(bs, all[i].b)
		ds = append(ds, all[i].d)
	}
	return bs, ds
}

// edDir names the direction from a zone to a parasang.
func edDir(zn *Zone, w *WorldData, p2 int) string {
	dx := (p2%w.W-zn.WX)*3 + (1 - zn.ZX)
	dy := (p2/w.W-zn.WY)*3 + (1 - zn.ZY)
	return bhDir(dx, dy)
}

// edEvNear finds the nearest event (by parasang distance, then index) within rad that pred accepts.
func edEvNear(w *WorldData, p, rad int, pred func(e *EventData) bool) (int, int) {
	best, bd := -1, 99
	cx, cy := p%w.W, p/w.W
	for y := cy - rad; y <= cy+rad; y++ {
		for x := cx - rad; x <= cx+rad; x++ {
			if !w.In(x, y) {
				continue
			}
			q := w.Idx(x, y)
			d := maxi(abs(x-cx), abs(y-cy))
			for _, i := range w.eventsAt[q] {
				if i < 0 || i >= len(w.Events) || !pred(&w.Events[i]) {
					continue
				}
				if d < bd || (d == bd && i < best) {
					best, bd = i, d
				}
			}
		}
	}
	return best, bd
}

var edWarName = regexp.MustCompile(`(?:[Tt]he )?([A-Z][A-Za-z'\-]+(?: [A-Z][A-Za-z'\-]+){0,3} (?:War|Rebellion|Uprising|Siege|Revolt|Revolution))`)

func edWarOf(text string) string {
	if m := edWarName.FindStringSubmatch(text); m != nil {
		return "the " + m[1]
	}
	return "the war"
}

func edIsWar(e *EventData) bool { return e.Kind == "war" }

func edMonsterByID(w *WorldData, id int) *MonsterData {
	for i := range w.Monsters {
		if w.Monsters[i].ID == id {
			return &w.Monsters[i]
		}
	}
	return nil
}

// edLiving lists the living, open-country monsters within rad parasangs of p.
func (g *Game) edLiving(p, rad int) []*MonsterData {
	var out []*MonsterData
	for i := range g.W.Monsters {
		m := &g.W.Monsters[i]
		if m.Slain || g.Slain[m.ID] || m.Layer != "surface" || m.P < 0 || m.P >= len(g.W.Biome) || g.W.IsWater(m.P) || g.W.Burg[m.P] != 0 {
			continue
		}
		if wsCheb(g.W, m.P, p) <= rad {
			out = append(out, m)
		}
	}
	return out
}

func edHabitOf(m *MonsterData) *edHabit {
	blob := strings.ToLower(m.Species + " " + m.Name + " " + m.Lore)
	for i := range edHabits[:len(edHabits)-1] {
		for _, k := range edHabits[i].Keys {
			if strings.Contains(blob, k) {
				return &edHabits[i]
			}
		}
	}
	return &edHabits[len(edHabits)-1]
}

// ---------------------------------------------------------------- scene context

type edCtx struct {
	g    *Game
	w    *WorldData
	zn   *Zone
	r    *Rng
	pt   *tgPainter
	rd   []uint8
	p    int
	cu   *CultureData
	near *BurgData
	nd   int
	st   *edState
}

func edRoadDist(zn *Zone) []uint8 {
	d := make([]uint8, MW*MH)
	for i := range d {
		d[i] = 255
	}
	var q []int
	for i, t := range zn.T {
		if t == TRoad {
			d[i] = 0
			q = append(q, i)
		}
	}
	for h := 0; h < len(q); h++ {
		i := q[h]
		if d[i] >= 9 {
			continue
		}
		x, y := i%MW, i/MW
		for _, dd := range dirs8 {
			nx, ny := x+dd[0], y+dd[1]
			if nx < 0 || ny < 0 || nx >= MW || ny >= MH {
				continue
			}
			j := ny*MW + nx
			if d[j] > d[i]+1 {
				d[j] = d[i] + 1
				q = append(q, j)
			}
		}
	}
	return d
}

func (c *edCtx) hasRoad() bool {
	for _, v := range c.rd {
		if v == 0 {
			return true
		}
	}
	return false
}

func (c *edCtx) clearOK(x, y int) bool {
	t := c.zn.At(x, y)
	return c.pt.ok(x, y) && tgNatural(t) && t != TRock && t != TCrystal && t.D().Water == 0 && c.zn.ActorAt(x, y) == nil
}

// patch finds a w by h block of clearable natural ground whose distance to any road is in [minR, maxR].
func (c *edCtx) patch(w, h, minR, maxR int, ok func(x0, y0 int) bool) (int, int, bool) {
	for try := 0; try < 140; try++ {
		x0, y0 := c.r.Int(3, MW-3-w), c.r.Int(3, MH-3-h)
		good := true
		for y := y0; y < y0+h && good; y++ {
			for x := x0; x < x0+w; x++ {
				rd := int(c.rd[y*MW+x])
				if !c.clearOK(x, y) || rd < minR || rd > maxR {
					good = false
					break
				}
			}
		}
		if good && (ok == nil || ok(x0, y0)) {
			return x0, y0, true
		}
	}
	return 0, 0, false
}

// ground clears a tile to the zone's plain floor.
func (c *edCtx) ground(x, y int) {
	switch c.zn.At(x, y) {
	case TTree, TDeadTree, TBush, TMushroom, TReeds, TBones, TRubble, TTallGrass:
		c.zn.Set(x, y, c.g.floorFor(c.zn))
		c.pt.changed = true
	}
}

func (c *edCtx) set(x, y int, t Terrain) {
	if !c.zn.In(x, y) {
		return
	}
	c.zn.Set(x, y, t)
	c.pt.changed = true
}

func (c *edCtx) clearPatch(x0, y0, w, h int) {
	for y := y0; y < y0+h; y++ {
		for x := x0; x < x0+w; x++ {
			c.ground(x, y)
		}
	}
}

func (c *edCtx) prop(x, y int, name, lore string, gl rune, col byte) *Item {
	it := &Item{ID: c.g.newID(), Kind: "scrap", Name: name, Glyph: gl, F: col, Value: 1, Count: 1, Lore: lore, X: x, Y: y}
	c.zn.Items = append(c.zn.Items, it)
	return it
}

func (c *edCtx) book(x, y int, name, text string) *Item {
	it := c.g.MakeBook(name, text)
	it.X, it.Y, it.Ident = x, y, true
	c.zn.Items = append(c.zn.Items, it)
	return it
}

func (c *edCtx) inscribe(x, y int, text string) int {
	c.set(x, y, TInscription)
	i := y*MW + x
	c.zn.Notes[i] = text
	return i
}

func (c *edCtx) bones(x0, y0, w, h int, chance float64) {
	for y := y0; y < y0+h; y++ {
		for x := x0; x < x0+w; x++ {
			if c.r.Chance(chance) && c.zn.At(x, y).D().Walk && c.zn.At(x, y) != TInscription && c.zn.ActorAt(x, y) == nil {
				c.set(x, y, TBones)
			}
		}
	}
}

func (c *edCtx) name() string { return c.g.cultureName(c.cu, c.r) }

func (c *edCtx) add(s *edScene) *edScene {
	s.Zone = c.zn.Key
	s.Day = c.g.Day()
	c.st.List = append(c.st.List, s)
	return s
}

func (c *edCtx) townName() string {
	if c.near != nil {
		return c.near.Name
	}
	return "the nearest town"
}

func (c *edCtx) state() string { return c.w.StateName(c.w.State[c.p]) }

func (c *edCtx) faith() (*ReligionData, *FaithPack) {
	if c.p >= len(c.w.Religion) {
		return nil, nil
	}
	rl := c.w.relByID[c.w.Religion[c.p]]
	if rl == nil {
		return nil, nil
	}
	return rl, rl.Lore
}

func (c *edCtx) food() string {
	if c.cu != nil && c.cu.Food != "" {
		return strings.ToLower(c.cu.Food)
	}
	return "bread"
}

func (c *edCtx) kv(extra ...string) map[string]string {
	m := map[string]string{"town": c.townName(), "state": c.state(), "cur": c.g.Sk.Currency, "food": c.food()}
	for i := 0; i+1 < len(extra); i += 2 {
		m[extra[i]] = extra[i+1]
	}
	return m
}

func (c *edCtx) goods() string {
	if edTech(c.g) {
		return Pick(c.r, edGoodsTech)
	}
	return Pick(c.r, edGoods)
}

func (c *edCtx) weapon() string {
	if n := c.g.Sk.Nouns["longblade"]; len(n) > 0 {
		return Pick(c.r, n)
	}
	return "sword"
}

func (c *edCtx) armour() string {
	if n := c.g.Sk.Armor["body"]; len(n) > 0 {
		return n[0]
	}
	return "mail"
}

func (c *edCtx) shield() string {
	if s, ok := edShieldTech[c.g.Genre]; ok {
		return s
	}
	return "shield"
}

func (c *edCtx) house() string {
	if c.cu != nil && len(c.cu.Houses) > 0 {
		return Pick(c.r, c.cu.Houses)
	}
	return "the Marches"
}

// ---------------------------------------------------------------- zone hook

func edZone(g *Game, zn *Zone, _ *Rng) {
	w := g.W
	if w == nil || zn.Z != 0 || zn.Sky || zn.Underwater || !w.In(zn.WX, zn.WY) || (zn.Kind != "surface" && zn.Kind != "lair") {
		return
	}
	p := w.Idx(zn.WX, zn.WY)
	if p < 0 || p >= len(w.Biome) || w.IsWater(p) || zn.Burg > 0 || w.Burg[p] != 0 {
		return
	}
	if g.siteAt(zn.WX, zn.WY, zn.ZX, zn.ZY) != nil {
		return
	}
	r := NewRng(w.Seed + ":es:zone:" + zn.Key)
	c := &edCtx{g: g, w: w, zn: zn, r: r, pt: tgNewPainter(zn), rd: edRoadDist(zn), p: p, st: edSt(g)}
	if p < len(w.Culture) {
		c.cu = w.cultByID[w.Culture[p]]
	}
	var nb []*BurgData
	var nds []int
	nb, nds = edNearest(w, p, 1)
	if len(nb) > 0 {
		c.near, c.nd = nb[0], nds[0]
	}
	if zn.Kind == "lair" {
		for _, a := range zn.Actors {
			if a.SimMon > 0 && a.HP > 0 {
				if m := edMonsterByID(w, a.SimMon); m != nil {
					c.lair(m, true, a.X, a.Y)
					break
				}
			}
		}
		c.finish()
		return
	}
	if c.near != nil && c.nd <= 1 {
		return
	}
	mons := g.edLiving(p, 2)
	gate := 0.34
	if len(mons) > 0 {
		gate = 0.6
	}
	if !r.Chance(gate) {
		return
	}
	road := c.hasRoad()
	war, _ := edEvNear(w, p, 5, edIsWar)
	type cand struct {
		kind string
		wt   int
	}
	var cs []cand
	if len(mons) > 0 {
		cs = append(cs, cand{"hunt", 10})
	}
	cs = append(cs, cand{"camp", 3})
	if war >= 0 {
		cs = append(cs, cand{"battle", 2})
	}
	if c.near != nil && c.nd <= 9 {
		cs = append(cs, cand{"hermit", 2})
	}
	if road {
		cs = append(cs, cand{"shrine", 3}, cand{"milestone", 3}, cand{"gibbet", 2})
	}
	cs = append(cs, cand{"fairy", 1}, cand{"spring", 2})
	if t, _ := g.edPassed(p); t != nil {
		cs = append(cs, cand{"trail", 3})
	}
	for try := 0; try < 4 && len(cs) > 0; try++ {
		tot := 0
		for _, k := range cs {
			tot += k.wt
		}
		roll := r.Int(0, tot-1)
		pick := 0
		for i, k := range cs {
			if roll < k.wt {
				pick = i
				break
			}
			roll -= k.wt
		}
		ok := false
		switch cs[pick].kind {
		case "hunt":
			ok = c.lair(mons[r.Int(0, len(mons)-1)], false, 0, 0)
		case "camp":
			ok = c.camp()
		case "battle":
			ok = c.battle()
		case "hermit":
			ok = c.hermit()
		case "shrine":
			ok = c.shrine()
		case "milestone":
			ok = c.milestone()
		case "gibbet":
			ok = c.gibbet()
		case "fairy":
			ok = c.fairy()
		case "spring":
			ok = c.spring()
		case "trail":
			ok = c.trail()
		}
		if ok {
			break
		}
		cs = append(cs[:pick], cs[pick+1:]...)
	}
	c.finish()
}

func (c *edCtx) finish() {
	if c.pt.changed {
		c.g.connect(c.zn)
		c.g.computeLight(c.zn)
	}
}

// ---------------------------------------------------------------- lairs and hunting grounds

func (c *edCtx) lair(m *MonsterData, inner bool, mx, my int) bool {
	hab := edHabitOf(m)
	hr := NewRng(fmt.Sprintf("%s:es:hab:%d", c.w.Seed, m.ID))
	mark, pat, hint := Pick(hr, hab.Mark), Pick(hr, hab.Pattern), Pick(hr, hab.Hint)
	where := nonEmpty(m.Lair, c.w.RegionName(m.P))
	base := c.kv("mon", m.Name, "mark", mark, "pattern", edUp(pat), "hint", hint, "where", where)
	pay := fmt.Sprintf("%s lairs at %s. %s. A hunter should know this: %s. The bounty board in %s is the place to ask what it is worth.",
		m.Name, where, edUp(pat), strings.ToLower(hint[:1])+hint[1:], c.townName())
	ref := fmt.Sprintf("m:%d", m.ID)
	if !inner {
		x0, y0, ok := c.patch(6, 3, 4, 255, nil)
		if !ok {
			return false
		}
		c.clearPatch(x0, y0, 6, 3)
		for i, tx := range []int{x0, x0 + 4} {
			ty := y0 + i
			c.set(tx, ty, TDeadTree)
			c.prop(tx, ty, "tree raked with claw marks", edUp(mark)+".", 'τ', CBrown)
		}
		c.bones(x0, y0, 6, 3, 0.2)
		idx := c.inscribe(x0+2, y0+2, edFill("Herders' cairn of {town}: {mon} hunts here. {pattern}. {hint}. Leave a stone if you came back. Ask at the bounty board in {town}.", base))
		c.add(&edScene{Kind: "hunt", Ref: ref, X: x0 + 2, Y: y0 + 2, Idx: idx + 1, Pay: pay, PayKind: "rumour", PayP: m.P})
		return true
	}
	// the lair itself: a victim's trace, well away from the beast
	far := func(x0, y0 int) bool { return abs(x0+4-mx)+abs(y0+2-my) >= 14 }
	x0, y0, ok := c.patch(9, 6, 3, 255, far)
	if !ok {
		return false
	}
	c.clearPatch(x0, y0, 9, 6)
	c.bones(x0, y0, 9, 6, 0.3)
	c.set(x0+8, y0+1, TDeadTree)
	c.prop(x0+8, y0+1, "tree raked with claw marks", edUp(mark)+".", 'τ', CBrown)
	kit := c.r.Int(0, 3)
	if hab == &edHabits[0] || hab == &edHabits[2] {
		if c.r.Chance(0.6) {
			kit = 3
		}
	}
	sc := &edScene{Kind: "lair", Ref: ref, Pay: pay, PayKind: "rumour", PayP: m.P}
	switch kit {
	case 0: // a traveller's bones and diary
		who := c.name()
		bs, _ := edNearest(c.w, c.p, 2)
		from, to := c.townName(), c.townName()
		if len(bs) > 1 {
			to = bs[1].Name
		}
		kv := c.kv("name", who, "from", from, "to", to, "d", fmt.Sprint(c.r.Int(3, 19)), "weapon", c.weapon(), "pack", "satchel")
		for k, v := range base {
			kv[k] = v
		}
		c.prop(x0+3, y0+2, edFill("bones in a rotted "+c.armour(), kv), "The bones of a traveller. A dagger lies by the hand, still in its sheath.", ',', CWhite)
		c.prop(x0+4, y0+3, edFill(Pick(c.r, edTravellerGear), kv), "", '/', CDarkGray)
		name := "Diary of " + who
		text := edFill(Pick(c.r, edTravellerLines)+"\n"+Pick(c.r, edTravellerLines[1:]), kv)
		c.book(x0+2, y0+3, name, text)
		sc.Book, sc.X, sc.Y = name, x0+2, y0+3
	case 1: // a merchant's overturned cart and ledger
		who := c.name()
		from, to := c.townName(), c.townName()
		ref2 := ""
		if t := c.wsCaravan(); t != nil {
			if b := c.w.burgByID[t.Home]; b != nil {
				from = b.Name
			}
			if b := c.w.burgByID[t.To]; b != nil {
				to = b.Name
			}
			ref2 = fmt.Sprintf(" t:%d", t.ID)
		} else if bs, _ := edNearest(c.w, c.p, 2); len(bs) > 1 {
			from, to = bs[0].Name, bs[1].Name
		}
		kv := c.kv("name", who, "from", from, "to", to, "goods", c.goods(), "n", fmt.Sprint(c.r.Int(4, 14)), "m", fmt.Sprint(c.r.Int(6, 60)))
		for k, v := range base {
			kv[k] = v
		}
		c.set(x0+3, y0+2, TCrate)
		c.prop(x0+3, y0+2, "overturned cart, one wheel stove in", "A cart on its side with the traces cut. The axle is bent as though something sat on it.", '■', CBrown)
		c.prop(x0+5, y0+3, edFill("spilled "+kv["goods"], kv), "", '%', CBrown)
		name := "Ledger of " + who
		c.book(x0+4, y0+2, name, edFill(Pick(c.r, edLedgerLines), kv))
		sc.Book, sc.X, sc.Y, sc.Ref = name, x0+4, y0+2, ref+ref2
	case 2: // a knight's broken shield and last note
		who := "Sir " + c.name()
		house := c.house()
		kv := c.kv("name", who, "n", fmt.Sprint(c.r.Int(3, 9)), "kin", Pick(c.r, []string{"mother", "brother", "sister", "wife", "squire"}))
		for k, v := range base {
			kv[k] = v
		}
		charge, field := Pick(c.r, edCharges), Pick(c.r, edFields)
		c.prop(x0+3, y0+2, fmt.Sprintf("split %s, %s on a field of %s", c.shield(), charge, field),
			fmt.Sprintf("The arms of House %s of %s, cleft from the rim to the boss.", house, c.state()), '◘', CGray)
		c.prop(x0+5, y0+3, "broken "+c.weapon(), "", '/', CDarkGray)
		name := "Last note of " + who
		c.book(x0+4, y0+2, name, edFill(Pick(c.r, edKnightNotes), kv)+"\n\nSigned, "+who+" of House "+house+".")
		sc.Book, sc.X, sc.Y = name, x0+4, y0+2
	default: // a nest of stolen shiny things
		j1, j2 := Pick(c.r, edNestJunk), Pick(c.r, edNestJunk)
		it := c.prop(x0+4, y0+2, "nest of stolen shiny things: "+j1+" and "+j2, "A heap of bright rubbish lined with torn cloth, gathered by something that likes to glitter. The bones around it are travellers'.", '*', CYellow)
		it.Value = 30 + m.Power/2
		c.prop(x0+2, y0+3, "signet ring of House "+c.house(), "A ring still on its finger bone.", '=', CYellow).Value = 40
		sc.Near, sc.X, sc.Y = true, x0+4, y0+2
	}
	c.add(sc)
	return true
}

// wsCaravan finds a real caravan on the day clock whose home or goal is near this zone.
func (c *edCtx) wsCaravan() *wsTrav {
	for _, t := range c.g.ws().Trav {
		if t.Kind != "caravan" || t.Broken {
			continue
		}
		for _, id := range []int{t.Home, t.To} {
			if b := c.w.burgByID[id]; b != nil && wsCheb(c.w, b.P, c.p) <= 8 {
				return t
			}
		}
	}
	return nil
}

// edPassed finds the most recent traveller to cross a parasang within the last days.
func (g *Game) edPassed(p int) (*wsTrav, int) {
	hour := g.wsHour()
	var best *wsTrav
	bh := 24 * 7
	for _, t := range g.ws().Trav {
		if t.Broken || t.Hpp <= 0 || (t.Kind != "caravan" && t.Kind != "army" && t.Kind != "merc") {
			continue
		}
		for i, q := range t.Path {
			if q != p {
				continue
			}
			age := hour - (t.Start + i*t.Hpp)
			if age >= 0 && age < bh {
				best, bh = t, age
			}
		}
	}
	return best, bh
}

// ---------------------------------------------------------------- camps

func (c *edCtx) fireRing(cx, cy, ageDays int, label string) {
	for _, d := range dirs8 {
		if c.r.Chance(0.75) {
			c.set(cx+d[0], cy+d[1], TRubble)
		}
	}
	t := TDirt
	look := fmt.Sprintf("a cold fire ring of blackened stones, the ash gone to paste (about %d days old)", maxi(2, ageDays))
	if ageDays <= 2 {
		t = TAsh
		look = "a fire ring of blackened stones, the ash still faintly warm"
	}
	c.set(cx, cy, t)
	c.prop(cx, cy, look, label, '*', CDarkGray)
}

func (c *edCtx) bedroll(x, y int, name, lore string) {
	c.set(x, y, TBed)
	c.prop(x, y, name, lore, '=', CDarkRed)
}

func (c *edCtx) graves(x0, y0, n int, who string) int {
	for i := 0; i < n; i++ {
		c.set(x0+i*2, y0, TRubble)
		c.prop(x0+i*2, y0, "hasty grave", "A mound of piled stones with a broken stick for a marker.", '%', CDarkGray)
	}
	return c.inscribe(x0+n, y0+1, edFill(Pick(c.r, edHastyGraveLines), c.kv("n", fmt.Sprint(n), "who", who)))
}

// mapScrap makes a map fragment marking a real site (true) or a deliberately false one.
func (c *edCtx) mapScrap(x, y int, sc *edScene) bool {
	sites := c.g.rvSites(c.p, 9)
	if len(sites) == 0 {
		return false
	}
	s := sites[c.r.Int(0, len(sites)-1)]
	truth := c.r.Chance(0.6)
	pp := s.P
	var text, pay, kind string
	if truth {
		text = edFill(Pick(c.r, edMapTrue), c.kv("site", s.S.Name, "thing", Pick(c.r, edThings)))
		pay = fmt.Sprintf("%s is a place near %s. A map scrap in an old camp marked it.", s.S.Name, c.w.RegionName(s.P))
		kind = "dungeon"
	} else {
		// a decoy: a land parasang near the real one with no site in any of its nine zones
		pp = -1
		for try := 0; try < 30 && pp < 0; try++ {
			q := s.P + c.r.Int(-3, 3) + c.r.Int(-3, 3)*c.w.W
			if q < 0 || q >= len(c.w.Biome) || c.w.IsWater(q) || c.w.Burg[q] != 0 || q%c.w.W < 1 || q%c.w.W >= c.w.W-1 {
				continue
			}
			clean := true
			for zy := 0; zy < 3 && clean; zy++ {
				for zx := 0; zx < 3; zx++ {
					if c.g.siteAt(q%c.w.W, q/c.w.W, zx, zy) != nil {
						clean = false
						break
					}
				}
			}
			if clean {
				pp = q
			}
		}
		if pp < 0 {
			return false
		}
		text = Pick(c.r, edMapFalse)
		pay = fmt.Sprintf("A map scrap marked a hoard near %s. The ink was newer than the paper, and the coast was wrong.", c.w.RegionName(pp))
		kind = "rumour"
	}
	text += fmt.Sprintf("\n\nThe cross lies about %d parasangs to the %s of here, near %s.", maxi(1, wsCheb(c.w, pp, c.p)), edDir(c.zn, c.w, pp), c.w.RegionName(pp))
	name := "map scrap"
	if truth {
		name = "map scrap of " + s.S.Name
	}
	c.book(x, y, name, text)
	sc.Book, sc.Pay, sc.PayKind, sc.PayP = name, pay, kind, pp
	return true
}

func (c *edCtx) camp() bool {
	r := c.r
	war, _ := edEvNear(c.w, c.p, 5, edIsWar)
	var rivals []*rvRival
	c.g.rvEnsure()
	for _, rv := range c.g.rv().R {
		if rv.Alive {
			rivals = append(rivals, rv)
		}
	}
	n := r.Int(0, 99)
	switch {
	case war >= 0 && n < 40:
		return c.armyCamp(war)
	case len(rivals) > 0 && n < 70:
		return c.rivalCamp(rivals[r.Int(0, len(rivals)-1)])
	}
	return c.caravanCamp()
}

func (c *edCtx) caravanCamp() bool {
	x0, y0, ok := c.patch(9, 6, 4, 255, nil)
	if !ok {
		return false
	}
	c.clearPatch(x0, y0, 9, 6)
	from, to := c.townName(), c.townName()
	ref := "b:0"
	if c.near != nil {
		ref = fmt.Sprintf("b:%d", c.near.ID)
	}
	if t := c.wsCaravan(); t != nil {
		if b := c.w.burgByID[t.Home]; b != nil {
			from = b.Name
		}
		if b := c.w.burgByID[t.To]; b != nil {
			to = b.Name
		}
		ref = fmt.Sprintf("t:%d", t.ID)
	} else if bs, _ := edNearest(c.w, c.p, 3); len(bs) > 1 {
		from, to = bs[0].Name, bs[1+c.r.Int(0, len(bs)-2)].Name
	}
	age := c.r.Int(2, 9)
	why := Pick(c.r, edCampWhy)
	c.fireRing(x0+4, y0+2, age, "The ring is small: a dozen people, a few beasts.")
	c.bedroll(x0+1, y0+3, "half-packed bedroll, one boot left beside it", "Rolled to the middle and abandoned.")
	c.bedroll(x0+7, y0+3, "slept-in bedroll gone stiff with damp", "")
	c.prop(x0+6, y0+1, "a wagon's wheel rim and a bent pot", "", 'o', CBrown)
	kv := c.kv("name", c.name(), "from", from, "to", to, "goods", c.goods(), "why", why)
	sc := &edScene{Kind: "camp", Ref: ref}
	name := "Manifest of the " + from + " caravan"
	c.book(x0+3, y0+3, name, edFill("{from} to {to}, loaded with {goods}. Wagons: three. Hands: twelve. The camp was "+why+". Last line: owed at the ford, and not paid.", kv))
	sc.Book, sc.X, sc.Y = name, x0+3, y0+3
	sc.Pay = fmt.Sprintf("A caravan from %s bound for %s left this camp in a hurry. Its goods did not come home.", from, to)
	sc.PayKind, sc.PayP = "rumour", c.p
	if strings.Contains(why, "sickness") || strings.Contains(why, "raid") {
		c.graves(x0+2, y0+5, c.r.Int(2, 3), "caravan of "+from)
	}
	if c.r.Chance(0.35) {
		sc2 := &edScene{Kind: "camp", Ref: ref}
		if c.mapScrap(x0+5, y0+3, sc2) {
			sc2.X, sc2.Y = x0+5, y0+3
			c.add(sc2)
		}
	}
	c.add(sc)
	return true
}

func (c *edCtx) armyCamp(ev int) bool {
	e := &c.w.Events[ev]
	st := c.w.stateByID[e.St]
	if st == nil {
		return false
	}
	x0, y0, ok := c.patch(13, 7, 4, 255, nil)
	if !ok {
		return false
	}
	c.clearPatch(x0, y0, 13, 7)
	war := edWarOf(e.Text)
	old := c.w.Year - e.Y
	kind := Pick(c.r, edArmyKinds)
	days := maxi(3, c.r.Int(3, 14))
	for row := 0; row < 2; row++ {
		for col := 0; col < 4; col++ {
			x, y := x0+1+col*3, y0+row*3
			if c.r.Chance(0.75) {
				if old > 6 {
					c.prop(x, y, "rotted heap of bedding, a few rusted buckles", "", '=', CDarkGray)
				} else {
					c.bedroll(x, y, "bedroll left in a neat row", "The ground beneath it is still flat.")
				}
			}
		}
	}
	c.fireRing(x0+6, y0+3, days+old*60, "One of several in a long line.")
	c.fireRing(x0+11, y0+4, days+old*60, "")
	c.prop(x0+12, y0+1, "horse lines: a rope strung between stakes, dung", "", '=', CBrown)
	foe := "its enemies"
	if len(st.Wars) > 0 {
		foe = c.w.StateName(st.Wars[0])
	}
	to := c.townName()
	kv := c.kv("state", st.Name, "foe", foe, "war", war, "kind", kind, "year", fmt.Sprint(e.Y), "to", to, "dir", edDir(c.zn, c.w, e.P))
	name := "Marching orders of the " + st.Name + " " + kind
	text := edFill("Orders of the {state} {kind}, year {year}. By {war}, the column will make for {to}, to the {dir}, and forage along the way. No man leaves the line without leave. Rations for four days; after that the countryside pays.", kv)
	c.book(x0+4, y0+5, name, text)
	idx := c.graves(x0+2, y0+6, c.r.Int(2, 3), "the "+kind+" of "+st.Name)
	sc := &edScene{Kind: "camp", Ref: fmt.Sprintf("e:%d", ev), Book: name, X: x0 + 4, Y: y0 + 5, Idx: idx + 1,
		PayKind: "history", PayP: e.P, Pay: fmt.Sprintf("In the year %d, %s", e.Y, lowerFirst(e.Text))}
	c.add(sc)
	return true
}

func (c *edCtx) rivalCamp(rv *rvRival) bool {
	x0, y0, ok := c.patch(9, 5, 4, 255, nil)
	if !ok {
		return false
	}
	c.clearPatch(x0, y0, 9, 5)
	name := c.g.rvName(rv)
	home := "somewhere far"
	if b := c.w.burgByID[rv.Home]; b != nil {
		home = b.Name
	}
	r := NewRng(fmt.Sprintf("%s:es:rv:%d:%d", c.w.Seed, rv.Slot, rv.Gen)) // stable per rival
	site, via := "a place I will not name", c.townName()
	var sp *rvSite
	if sites := c.g.rvSites(c.p, 9); len(sites) > 0 {
		s := sites[r.Int(0, len(sites)-1)]
		sp = &s
		site = s.S.Name
	}
	if bs, _ := edNearest(c.w, c.p, 2); len(bs) > 1 {
		via = bs[1].Name
	}
	c.fireRing(x0+4, y0+2, c.r.Int(1, 5), "A careful fire. Whoever made it knew to cover the smoke.")
	c.bedroll(x0+2, y0+3, "half-packed bedroll, a whetstone left on it", "Packed in a hurry or in a fine mood.")
	boast := edFill(Pick(r, edCampBoast), c.kv("n", fmt.Sprint(maxi(2, rv.Clears+rv.Kills)), "rv", name, "calling", strings.ToLower(rv.Calling)))
	end := Pick(r, edRivalEnd)
	text := fmt.Sprintf("Page torn from the diary of %s, %s.\n\nOut of %s, bound for %s by way of %s. %s\n\n%s", name, rv.Title, home, site, via, boast, end)
	bk := "Diary page of " + name
	c.book(x0+6, y0+3, bk, text)
	sc := &edScene{Kind: "camp", Ref: fmt.Sprintf("r:%d", rv.Slot), Book: bk, X: x0 + 6, Y: y0 + 3, PayKind: "rumour", PayP: c.p,
		Pay: fmt.Sprintf("%s is bound for %s. The diary page says nothing of waiting for anyone.", name, site)}
	if sp != nil {
		sc.PayKind, sc.PayP = "dungeon", sp.P
		sc.Pay = fmt.Sprintf("%s is a place near %s, which the rival %s is bound for. A race, if you hurry.", site, c.w.RegionName(sp.P), name)
	}
	if c.r.Chance(0.3) {
		sc2 := &edScene{Kind: "camp", Ref: fmt.Sprintf("r:%d", rv.Slot)}
		if c.mapScrap(x0+7, y0+1, sc2) {
			sc2.X, sc2.Y = x0+7, y0+1
			c.add(sc2)
		}
	}
	c.add(sc)
	return true
}

// ---------------------------------------------------------------- hermits

func edKinOf(g *Game, p *lvPerson) (name, rel string) {
	if sp := lvGet(g, p.Spouse); sp != nil {
		return sp.Name, "spouse"
	}
	for _, k := range p.Kin {
		if q := lvGet(g, k); q != nil {
			rel := p.Rel[k]
			if rel == "" {
				rel = "kin"
			}
			return q.Name, rel
		}
	}
	for _, k := range p.Friends {
		if q := lvGet(g, k); q != nil {
			return q.Name, "friend"
		}
	}
	return "", ""
}

func edReasonKind(p *lvPerson) (kind, why string) {
	for i := len(p.Life) - 1; i >= 0; i-- {
		e := p.Life[i]
		switch e.Kind {
		case "death", "crime", "feud", "illness", "marriage", "job":
			return e.Kind, strings.ToLower(strings.TrimRight(e.Text, "."))
		}
	}
	return "default", ""
}

func (c *edCtx) hermit() bool {
	g := c.g
	bs, _ := edNearest(c.w, c.p, 3)
	var home *BurgData
	for _, b := range bs {
		if wsCheb(c.w, b.P, c.p) <= 9 {
			home = b
			break
		}
	}
	if home == nil {
		return false
	}
	lvEnsureTown(g, home.ID)
	var keys []string
	skip := map[string]bool{"guard": true, "innkeeper": true, "baker": true, "smith": true, "healer": true, "priest": true, "merchant": true, "child": true}
	for k, p := range lvSt(g).People {
		if !strings.HasPrefix(k, "g") || !p.Alive || p.Burg != home.ID || p.Home != home.ID || skip[p.Role] || skip[p.Job] || p.lvAge(lvYear(g)) < 28 {
			continue
		}
		keys = append(keys, k)
	}
	if len(keys) == 0 {
		return false
	}
	sort.Strings(keys)
	x0, y0, ok := c.patch(14, 7, 4, 255, nil)
	if !ok {
		return false
	}
	p := lvSt(g).People[keys[c.r.Int(0, len(keys)-1)]]
	alive := c.r.Chance(0.55)
	// the person leaves town for good (frozen like a missing person; never drawn again in town)
	day := lvDay(g)
	ls := lsSt(g)
	p.Burg, p.Dest, p.ArriveD = 0, p.Home, day+400000
	ls.Jail[p.Key] = p.ArriveD
	ls.From[p.Key] = home.ID
	p.Zone = c.zn.Key
	kind, why := edReasonKind(p)
	kv := c.kv("town", home.Name, "why", why, "job", nonEmpty(p.Job, "a trade"), "name", p.Name)
	reasons := edHermitReasons[kind]
	line := edFill(Pick(c.r, reasons), kv)
	if pk := c.w.cpOf(p.Culture); pk != nil && pk.Voice != nil && len(pk.Voice.Proverbs) > 0 && c.r.Chance(0.5) {
		line += " " + edDot(c.w.cpChoose(pk.Voice.Proverbs, "hermit", p.Key))
	}
	if !alive {
		p.Alive, p.Died = false, lvYear(g)
		p.Cause = "alone in a cabin in the wilds"
		p.addLife(lvYear(g), "death", "Died alone in a cabin in the wilds.")
	}
	c.clearPatch(x0, y0, 14, 7)
	// cabin: 7 by 5 at the left, garden at the right
	for y := y0; y < y0+5; y++ {
		for x := x0; x < x0+7; x++ {
			if x == x0 || x == x0+6 || y == y0 || y == y0+4 {
				c.set(x, y, TWoodWall)
			} else {
				c.set(x, y, TWoodFloor)
			}
		}
	}
	c.set(x0+3, y0+4, TDoor)
	wall := edFill(Pick(c.r, edHermitWall), map[string]string{"line": line}) +
		fmt.Sprintf("\n\nBeneath it, smaller: %s, who was %s of %s, born in the year %d.", p.Name, lowerFirst(nonEmpty(p.Job, "someone")), home.Name, p.Born)
	widx := c.inscribe(x0+3, y0, wall)
	if alive {
		c.set(x0+5, y0+1, TBrazier)
	} else {
		c.set(x0+5, y0+1, TRubble)
		c.prop(x0+5, y0+1, "cold hearth, the ash undisturbed", "", '%', CDarkGray)
	}
	c.bedroll(x0+1, y0+1, "narrow bed, the blanket folded square", "")
	if !alive {
		c.set(x0+2, y0+2, TBones)
		c.prop(x0+2, y0+2, edFill(Pick(c.r, edHermitDead), nil), "", ',', CWhite)
	}
	// a book or two they read
	for i := 0; i < 2; i++ {
		if bk := g.GenBook(c.r, c.p); bk != nil {
			bk.X, bk.Y, bk.Ident = x0+2+i*2, y0+3, true
			c.zn.Items = append(c.zn.Items, bk)
		}
	}
	// the note to a loved one
	kin, rel := edKinOf(g, p)
	sc := &edScene{Kind: "hermit", Ref: "p:" + p.Key, Idx: widx + 1, X: x0 + 3, Y: y0 + 4}
	if kin != "" {
		who := kin
		if rel != "" && rel != "kin" && rel != "spouse" && rel != "friend" {
			who = "my " + rel + " " + kin
		}
		name := "Unsent letter to " + kin
		c.book(x0+3, y0+2, name, edUp(edFill(Pick(c.r, edHermitLoved), c.kv("kin", who, "town", home.Name))))
		sc.Book = name
		sc.Pay = fmt.Sprintf("%s left %s and lives alone in the wilds. %s still has a letter that was never sent to %s.", p.Name, home.Name, p.Name, kin)
		sc.PayKind, sc.PayP = "grief", c.p
	}
	// a garden of herbs
	for gy := 1; gy <= 3; gy++ {
		for gx := 8; gx <= 12; gx++ {
			c.set(x0+gx, y0+gy, TCrop)
		}
	}
	for i := 0; i < 3; i++ {
		h := herbs[c.r.Int(0, len(herbs)-1)]
		if h.Under {
			h = herbs[0]
		}
		it := c.g.makeIngredient(h.Name, h.Tag, 1, CGreen)
		it.X, it.Y = x0+8+i*2, y0+1+i
		it.Name = h.Name + " (tended)"
		c.zn.Items = append(c.zn.Items, it)
	}
	if alive {
		cu := c.w.cultByID[p.Culture]
		a := g.makeNPC("villager", home, cu, c.r)
		a.Name, a.Proper, a.Title, a.Role, a.AI = p.Name, true, "hermit", "hermit", "still"
		a.Desc = fmt.Sprintf("%s %s", Pick(c.r, edHermitAlive), "They left "+home.Name+" and have not gone back.")
		a.X, a.Y = x0+9, y0+4
		a.Set("edhermit")
		c.zn.Actors = append(c.zn.Actors, a)
		lvb(g).Bind[a.ID] = p.Key
	}
	c.add(sc)
	return true
}

func edTalk(g *Game, a *Actor) []Option {
	if g.W == nil || !a.Has("edhermit") || a.HP <= 0 {
		return nil
	}
	p := lvGet(g, lvb(g).Bind[a.ID])
	if p == nil {
		return nil
	}
	var sc *edScene
	st := edSt(g)
	for _, s := range st.List {
		if s.Kind == "hermit" && s.Ref == "p:"+p.Key {
			sc = s
			break
		}
	}
	if sc == nil {
		return nil
	}
	home := g.W.burgByID[p.Home]
	town := "town"
	if home != nil {
		town = home.Name
	}
	var out []Option
	out = append(out, Option{"Ask why you left " + town + ".", func() {
		kind, why := edReasonKind(p)
		r := NewRng(g.Seed + ":es:hermit:say:" + p.Key)
		kv := map[string]string{"town": town, "why": why, "job": nonEmpty(p.Job, "a trade"), "name": p.Name}
		line := edFill(Pick(r, edHermitReasons[kind]), kv)
		text := fmt.Sprintf("\"%s\"", line)
		if why != "" {
			text += fmt.Sprintf("\n\n\"They will tell you it was because %s. Say what you like. That is my side of it.\"", why)
		}
		if sc.Fact == 0 {
			fa := knSt(g).add(knFact{Kind: "event", Subject: p.Key, True: true, Source: p.Key, Day: lvDay(g), Burg: p.Home, Lvl: 1,
				Text: fmt.Sprintf("%s of %s left town and lives alone in the wilds. In their own words: %s", p.Name, town, line)})
			sc.Fact = fa.ID
			knGive(p, fa.ID)
		}
		if fa := knSt(g).fact(sc.Fact); fa != nil {
			g.knLearn(fa, p, 2, "ask")
		}
		p.Met = true
		g.showPopup(p.Name, text)
	}})
	if !st.Said["herb:"+p.Key] {
		out = append(out, Option{"Ask about the garden.", func() {
			st.Said["herb:"+p.Key] = true
			it := g.makeIngredient("tended bitterroot", "mending", 2, CGreen)
			g.give(it)
			g.showPopup(p.Name, "\"Bitterroot, mostly. It mends what the walk out here breaks. Take two bunches; the plot forgives me.\"")
		}})
	}
	return out
}

// ---------------------------------------------------------------- roadside: shrines, milestones, gibbets

func (c *edCtx) roadSpot(w, h int) (int, int, bool) { return c.patch(w, h, 2, 4, nil) }

func (c *edCtx) shrine() bool {
	x0, y0, ok := c.roadSpot(3, 3)
	if !ok {
		return false
	}
	c.clearPatch(x0, y0, 3, 3)
	rl, fp := c.faith()
	deity := "the old powers"
	if rl != nil {
		deity = nonEmpty(rl.Deity, rl.Name)
	}
	kv := c.kv("deity", deity)
	text := edFill(Pick(c.r, edShrineGeneric), kv)
	pay := ""
	if fp != nil {
		if len(fp.Saints) > 0 {
			sa := fp.Saints[c.r.Int(0, len(fp.Saints)-1)]
			text = fmt.Sprintf("A wayside shrine to %s, saint of %s.\n%s", sa.Name, rl.Name, sa.Legend)
			pay = fmt.Sprintf("%s is remembered among %s at wayside shrines: %s", sa.Name, rl.Name, edDot(edFirstSentence(sa.Legend)))
		}
		if len(fp.Prayers) > 0 {
			text += "\n\nCut into the lintel: " + edDot(c.w.cpChoose(fp.Prayers, "shrine", c.zn.Key))
		} else if len(fp.Blessings) > 0 {
			text += "\n\nCut into the lintel: " + edDot(c.w.cpChoose(fp.Blessings, "shrine", c.zn.Key))
		}
	}
	if pay == "" && rl != nil {
		pay = fmt.Sprintf("A wayside shrine honours %s of %s. Travellers leave offerings here for the road.", deity, rl.Name)
	}
	c.set(x0+1, y0+1, TAltar)
	c.zn.Notes[(y0+1)*MW+x0+1] = text
	for i, d := range [][2]int{{-1, 0}, {1, 0}, {0, 1}} {
		if i == 2 || c.r.Chance(0.7) {
			it := edFill(Pick(c.r, edShrineOffers), kv)
			c.prop(x0+1+d[0], y0+1+d[1], it, "Left for the road by someone who got where they were going.", '*', CYellow)
		}
	}
	ref := "b:0"
	if c.near != nil {
		ref = fmt.Sprintf("b:%d", c.near.ID)
	}
	c.add(&edScene{Kind: "shrine", Ref: ref, X: x0 + 1, Y: y0 + 1, Near: true, Pay: pay, PayKind: "faith", PayP: -1})
	return true
}

func (c *edCtx) milestone() bool {
	x0, y0, ok := c.roadSpot(2, 2)
	if !ok {
		return false
	}
	bs, ds := edNearest(c.w, c.p, 2)
	if len(bs) == 0 {
		return false
	}
	c.clearPatch(x0, y0, 2, 2)
	var sb strings.Builder
	for i, b := range bs {
		sb.WriteString(fmt.Sprintf("%s: %d parasangs to the %s.\n", b.Name, maxi(1, ds[i]), edDir(c.zn, c.w, b.P)))
	}
	sb.WriteString("\nScratched over the weathered letters:\n")
	kv := c.kv()
	for i := 0; i < 2; i++ {
		who := c.name()
		if c.r.Chance(0.5) {
			sb.WriteString(who + " " + edFill(Pick(c.r, edGraffitiNames), kv) + ".\n")
		} else {
			sb.WriteString(edFill(Pick(c.r, edGraffitiLines), kv) + "\n")
		}
	}
	// one line from the real world: a war on this state's borders, or a sickness near
	st := c.w.stateByID[c.w.State[c.p]]
	switch {
	case st != nil && len(st.Wars) > 0:
		sb.WriteString(fmt.Sprintf("%s is at war with %s. Mind the border.\n", st.Name, c.w.StateName(st.Wars[0])))
	case c.w.Hazards != nil && len(c.w.Hazards.Outbreaks) > 0:
		ob := c.w.Hazards.Outbreaks[c.r.Int(0, len(c.w.Hazards.Outbreaks)-1)]
		sb.WriteString(fmt.Sprintf("%s again. Go round.\n", edUp(ob.Name)))
	}
	idx := c.inscribe(x0, y0, strings.TrimRight(sb.String(), "\n"))
	c.add(&edScene{Kind: "milestone", Ref: fmt.Sprintf("b:%d", bs[0].ID), X: x0, Y: y0, Idx: idx + 1})
	return true
}

// edGallows picks a real executed criminal: a life-sim hanging, else a dead villain or deposed tyrant.
func (c *edCtx) gallows() (name, role, crime, ref string, year int, ok bool) {
	g, w := c.g, c.w
	var keys []string
	for k, p := range lvSt(g).People {
		if !p.Alive && strings.Contains(p.Cause, "gallows") {
			keys = append(keys, k)
		}
	}
	if len(keys) > 0 {
		sort.Strings(keys)
		p := lvSt(g).People[keys[c.r.Int(0, len(keys)-1)]]
		return p.Name, nonEmpty(p.Job, "townsperson"), strings.TrimPrefix(p.Cause, "on the gallows for "), "p:" + p.Key, p.Died, true
	}
	var pool []*PersonData
	for i := range w.People {
		pd := &w.People[i]
		if pd.Alive || pd.Died <= 0 {
			continue
		}
		cl := strings.ToLower(pd.Cause)
		if pd.Role == "villain" || pd.Role == "criminal" || pd.Role == "traitor" || strings.Contains(cl, "deposed") || strings.Contains(cl, "coup") || strings.Contains(cl, "revolution") || strings.Contains(cl, "execut") || strings.Contains(cl, "hang") {
			pool = append(pool, pd)
		}
	}
	if len(pool) == 0 {
		return
	}
	var local []*PersonData
	for _, pd := range pool {
		if pd.State == w.State[c.p] {
			local = append(local, pd)
		}
	}
	if len(local) > 0 {
		pool = local
	}
	pd := pool[c.r.Int(0, len(pool)-1)]
	cl := strings.ToLower(pd.Cause)
	key := "default"
	switch {
	case strings.Contains(cl, "revolution"):
		key = "revol"
	case strings.Contains(cl, "defeat"):
		key = "defeat"
	case strings.Contains(cl, "coup"):
		key = "coup"
	case strings.Contains(cl, "deposed"):
		key = "deposed"
	}
	crime = edFill(Pick(c.r, edGibbetCrimes[key]), c.kv("state", w.StateName(pd.State)))
	return pd.Name, nonEmpty(pd.Title, pd.Role), crime, fmt.Sprintf("p:%d", pd.ID), pd.Died, true
}

func (c *edCtx) gibbet() bool {
	name, role, crime, ref, year, ok := c.gallows()
	if !ok {
		return false
	}
	x0, y0, ok := c.roadSpot(3, 3)
	if !ok {
		return false
	}
	c.clearPatch(x0, y0, 3, 3)
	c.set(x0+1, y0, TDeadTree)
	c.prop(x0+1, y0, "iron gibbet cage hanging from a dead tree", "What the crows left of "+name+" sways inside, in rags.", 'Ω', CGray)
	c.set(x0+1, y0+1, TBones)
	kv := c.kv("name", name, "role", role, "crime", crime, "year", fmt.Sprint(year))
	idx := c.inscribe(x0+2, y0+1, edFill(Pick(c.r, edGibbetPlacard), kv))
	c.add(&edScene{Kind: "gibbet", Ref: ref, X: x0 + 2, Y: y0 + 1, Idx: idx + 1, PayKind: "history", PayP: c.p,
		Pay: fmt.Sprintf("%s, %s, was hanged in the year %d for %s. Their gibbet stands by the road near %s.", name, role, year, edFill(crime, kv), c.townName())})
	return true
}

// ---------------------------------------------------------------- cursed or tainted ground

func (c *edCtx) fairy() bool {
	w := c.w
	var pool []*PersonData
	for i := range w.People {
		pd := &w.People[i]
		cl := strings.ToLower(pd.Cause)
		if !pd.Alive && pd.Died > 0 && (strings.Contains(cl, "hunting") || strings.Contains(cl, "horse")) {
			pool = append(pool, pd)
		}
	}
	if len(pool) == 0 {
		return false
	}
	pd := pool[c.r.Int(0, len(pool)-1)]
	x0, y0, ok := c.patch(9, 7, 3, 255, nil)
	if !ok {
		return false
	}
	c.clearPatch(x0, y0, 9, 7)
	cx, cy := x0+4, y0+3
	var tiles []int
	for dy := -3; dy <= 3; dy++ {
		for dx := -4; dx <= 4; dx++ {
			d := dx*dx + dy*dy*2
			switch {
			case d >= 12 && d <= 20:
				c.set(cx+dx, cy+dy, TFungus)
			case d < 5:
				tiles = append(tiles, (cy+dy)*MW+cx+dx)
			}
		}
	}
	c.prop(cx, cy, Pick(c.r, edFairyRing), "", 'o', CMagenta)
	note := edFill(edFairyNote[0], c.kv("name", pd.Name, "role", nonEmpty(pd.Title, pd.Role), "year", fmt.Sprint(pd.Died), "cause", strings.ToLower(pd.Cause), "state", w.StateName(pd.State)))
	c.inscribe(x0+8, y0+6, note)
	c.add(&edScene{Kind: "fairy", Ref: fmt.Sprintf("p:%d", pd.ID), X: cx, Y: cy, Taint: "edfey", Tiles: tiles, Idx: (y0+6)*MW + x0 + 8 + 1})
	return true
}

func (c *edCtx) battle() bool {
	ev, _ := edEvNear(c.w, c.p, 5, edIsWar)
	if ev < 0 {
		return false
	}
	e := &c.w.Events[ev]
	x0, y0, ok := c.patch(13, 7, 3, 255, nil)
	if !ok {
		return false
	}
	c.clearPatch(x0, y0, 13, 7)
	war := edWarOf(e.Text)
	st := c.w.StateName(e.St)
	var tiles []int
	for y := y0 + 1; y < y0+6; y++ {
		for x := x0 + 1; x < x0+12; x++ {
			switch c.r.Int(0, 5) {
			case 0:
				c.set(x, y, TBones)
				tiles = append(tiles, y*MW+x)
			case 1:
				c.set(x, y, TRubble)
				tiles = append(tiles, y*MW+x)
			case 2:
				if c.r.Chance(0.4) {
					c.set(x, y, TDeadTree)
				}
			}
		}
	}
	for i := 0; i < 3; i++ {
		x, y := x0+2+c.r.Int(0, 8), y0+2+c.r.Int(0, 3)
		if c.zn.At(x, y).D().Walk {
			c.prop(x, y, Pick(c.r, []string{"split " + c.shield() + ", the paint gone", "rusted " + c.weapon() + ", pitted through", "rags of a " + st + " banner", "helm full of rainwater and old bones"}), "", '/', CDarkGray)
		}
	}
	c.prop(x0+6, y0+3, Pick(c.r, edBattleLook), "", ',', CWhite)
	kv := c.kv("war", war, "year", fmt.Sprint(e.Y), "state", st)
	idx := c.inscribe(x0+12, y0+6, edFill(edBattleNote[0], kv))
	c.add(&edScene{Kind: "battle", Ref: fmt.Sprintf("e:%d", ev), X: x0 + 6, Y: y0 + 3, Taint: "edchill", Tiles: tiles, Idx: idx + 1,
		PayKind: "history", PayP: e.P, Pay: fmt.Sprintf("In the year %d, %s", e.Y, lowerFirst(e.Text))})
	return true
}

func (c *edCtx) spring() bool {
	w := c.w
	year, cause, ref := 0, "", ""
	var pp int
	if w.Hazards != nil {
		for i, ob := range w.Hazards.Outbreaks {
			for _, q := range ob.Parasangs {
				if wsCheb(w, q, c.p) <= 3 {
					year, cause, ref, pp = ob.Since, "an outbreak of "+strings.TrimPrefix(ob.Name, "the ")+" spread through the district", fmt.Sprintf("o:%d", i), q
					break
				}
			}
			if ref != "" {
				break
			}
		}
	}
	if ref == "" {
		ev, _ := edEvNear(w, c.p, 5, func(e *EventData) bool {
			t := strings.ToLower(e.Text)
			return (e.Kind == "disaster" || e.Kind == "eco_disaster") && (strings.Contains(t, "poison") || strings.Contains(t, "toxic") || strings.Contains(t, "plague") || strings.Contains(t, "blight") || strings.Contains(t, "fever") || strings.Contains(t, "pollut") || strings.Contains(t, "contamin"))
		})
		if ev < 0 {
			return false
		}
		e := &w.Events[ev]
		year, cause, ref, pp = e.Y, "the chronicle records: "+edFirstSentence(e.Text), fmt.Sprintf("e:%d", ev), e.P
	}
	x0, y0, ok := c.patch(9, 7, 3, 255, nil)
	if !ok {
		return false
	}
	c.clearPatch(x0, y0, 9, 7)
	cx, cy := x0+4, y0+3
	var tiles []int
	for dy := -1; dy <= 1; dy++ {
		for dx := -2; dx <= 2; dx++ {
			c.set(cx+dx, cy+dy, TShallow)
			tiles = append(tiles, (cy+dy)*MW+cx+dx)
		}
	}
	c.prop(cx, cy, "dead fish and reeds floating in scummed water", Pick(c.r, edSpringLook), '~', CDarkGreen)
	c.prop(cx+1, cy+1, "a bird gone stiff at the water's edge", "", ',', CWhite)
	idx := c.inscribe(x0+1, y0+1, "Do not drink or wade. This spring was fouled in the year "+fmt.Sprint(year)+". "+edUp(edDot(cause))+" "+c.townName()+" sends someone to check it each spring and has stopped sending the same person twice.")
	c.add(&edScene{Kind: "spring", Ref: ref, X: cx, Y: cy, Taint: "poisoned", Tiles: tiles, Idx: idx + 1, PayKind: "hazard", PayP: pp,
		Pay: fmt.Sprintf("A spring near %s was fouled in the year %d. Do not drink from it.", c.townName(), year)})
	return true
}

// ---------------------------------------------------------------- trail signs

func (c *edCtx) trail() bool {
	t, ageH := c.g.edPassed(c.p)
	if t == nil {
		return false
	}
	x0, y0, ok := c.patch(7, 3, 1, 4, nil)
	if !ok {
		x0, y0, ok = c.patch(7, 3, 4, 255, nil)
	}
	if !ok {
		return false
	}
	for y := y0; y < y0+3; y++ {
		for x := x0; x < x0+7; x++ {
			c.ground(x, y)
			if y != y0+1 {
				c.set(x, y, TDirt)
			}
		}
	}
	to := "its destination"
	if b := c.w.burgByID[t.To]; b != nil {
		to = b.Name
	}
	age := edAgeText(ageH)
	var look, pay string
	if t.Kind == "army" {
		look = edFill(Pick(c.r, edArmyLook), map[string]string{"state": c.w.StateName(t.Realm), "age": age, "to": to})
		pay = fmt.Sprintf("A column of %s, perhaps %d companies, marched through near %s about %s ago, bound for %s.", c.w.StateName(t.Realm), t.Size, c.townName(), age, to)
	} else {
		what := "caravan"
		if t.Kind == "merc" {
			what = "mercenary band"
		}
		look = edFill(Pick(c.r, edRutLook), map[string]string{"what": what, "age": age, "to": to})
		pay = fmt.Sprintf("A %s passed near %s about %s ago, bound for %s: %s.", what, c.townName(), age, to, t.Name)
	}
	c.prop(x0+3, y0+1, look, "", '=', CBrown)
	c.prop(x0+5, y0, "dung and a dropped strap", "", ',', CBrown)
	c.add(&edScene{Kind: "trail", Ref: fmt.Sprintf("t:%d", t.ID), X: x0 + 3, Y: y0 + 1, Near: true, Pay: pay, PayKind: "rumour", PayP: c.p})
	return true
}

// ---------------------------------------------------------------- play: smoke, taint, payoffs

func edEnter(g *Game, zn *Zone) {
	w := g.W
	if w == nil || zn.Z != 0 || zn.Sky || zn.Underwater || !w.In(zn.WX, zn.WY) || (zn.Kind != "surface" && zn.Kind != "lair" && zn.Kind != "outskirts") {
		return
	}
	p := w.Idx(zn.WX, zn.WY)
	if p < 0 || p >= len(w.Biome) || w.IsWater(p) {
		return
	}
	st := edSt(g)
	pick, place, at := "", "", -1
	if ev, d := edEvNear(w, p, 5, func(e *EventData) bool { return e.Kind == "war" && e.Y >= w.Year-3 && e.P >= 0 }); ev >= 0 && d >= 1 {
		pick, at = fmt.Sprintf("e:%d", ev), w.Events[ev].P
		place = g.wsPlace(at)
	}
	if pick == "" {
		hour := g.wsHour()
		for _, e := range g.ws().Ev {
			if (e.Kind == "invasion" || e.Kind == "bandits") && e.Start <= hour && (e.End == 0 || e.End > hour) && wsCheb(w, e.P, p) <= 4 && wsCheb(w, e.P, p) >= 1 {
				pick, at = fmt.Sprintf("w:%d", e.ID), e.P
				place = g.wsPlace(at)
				break
			}
		}
	}
	if pick == "" {
		return
	}
	k := zn.Key + "|" + pick
	if st.Smoke[k] {
		return
	}
	st.Smoke[k] = true
	r := NewRng(g.Seed + ":es:smoke:" + k)
	dir := edDir(zn, w, at)
	g.Msg("{{y|" + edFill(Pick(r, edSmokeLines), map[string]string{"dir": dir, "place": place}) + "}}")
	g.addSecret(fmt.Sprintf("Smoke stood on the horizon to the %s of %s. Something near %s was burning.", dir, zn.Name, place), "history", at, w.Year)
}

func edTurn(g *Game) {
	if g.W == nil || g.Cur == nil || g.P == nil {
		return
	}
	st := edSt(g)
	if len(st.List) == 0 {
		return
	}
	pi := g.P.Y*MW + g.P.X
	for _, s := range st.List {
		here := s.Zone == g.Cur.Key
		if here && s.Taint != "" {
			for _, t := range s.Tiles {
				if t == pi {
					if g.P.St(s.Taint) == 0 {
						g.Msg(edTaintMsg(s.Taint))
					}
					d := 40
					if s.Taint == "poisoned" {
						d = 8
					}
					g.P.AddSt(s.Taint, d)
					break
				}
			}
		}
		if s.Paid || s.Pay == "" {
			continue
		}
		hit := false
		switch {
		case s.Idx > 0 && here:
			hit = g.Read[g.Cur.Key+fmt.Sprint(s.Idx-1)]
		case s.Book != "":
			hit = g.Read["book:"+s.Book]
		case s.Near && here:
			hit = abs(g.P.X-s.X) <= 1 && abs(g.P.Y-s.Y) <= 1
		}
		if s.Idx > 0 && s.Book != "" && !hit {
			hit = g.Read["book:"+s.Book]
		}
		if !hit {
			continue
		}
		s.Paid = true
		if g.addSecret(s.Pay, s.PayKind, s.PayP, g.W.Year) {
			g.Msg("{{c|It adds up. You note it in your journal.}}")
			g.gainXP(8 + g.Skills["lore"]*4)
		}
	}
}

func edTaintMsg(id string) string {
	switch id {
	case "edfey":
		return "{{m|The air inside the ring hums, and the edges of things swim. You lose your footing in the light.}}"
	case "edchill":
		return "{{c|A chill climbs your legs from the old battlefield. Your nerve slips.}}"
	}
	return "{{G|The foul water stings your skin.}}"
}
