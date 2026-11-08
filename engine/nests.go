package engine

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// Nests, living ruins and ghost towns (prefix nt).
//
// Nests: breeding-season nests of bestiary egg-layers (birds, reptiles, dinosaurs, pterosaurs, insects,
// arachnids) in fitting wild zones and lost worlds. A nest is a floor item (Kind "nest", Extra ntnest = hash
// of zone key + tile) with eggs (Kind "egg") on the same tile and parent actors (flags ntpar, ntn:<hash>).
// State lives on those items and actors (spring lays, summer hatches, autumn and winter empty the nest), so
// the zone hook only uses side rngs. Taking an egg enrages the parents (a pursuit record in g.ext("nt"):
// they hunt you for days across nearby zones). Eggs in the pack hatch when kept warm; the hatchling imprints
// as a taming.go companion (tmJoin) and grows by days. Realmforge brood lairs hold a brood clutch.
// Living ruins: dead burgs get an occupant per half-year (enter hook). Ghost towns: a four-stage story.
// State g.ext("nt"), planet-local (unLocalKeys).

func init() {
	unLocalKeys = append(unLocalKeys, "nt")
	zoneHooks = append(zoneHooks, ntZone)
	enterHooks = append(enterHooks, ntEnter)
	turnHooks = append(turnHooks, ntTurn)
	killHooks = append(killHooks, ntKill)
	talkHooks = append(talkHooks, ntTalk)
	sideHooks = append(sideHooks, ntSide)
}

type ntPur struct {
	Sp     string `json:"sp"`
	Zone   string `json:"z"`
	WX     int    `json:"wx"`
	WY     int    `json:"wy"`
	Until  int    `json:"u"`
	Par    int    `json:"p"`
	Spawns int    `json:"s,omitempty"`
	Last   int    `json:"l,omitempty"`
}

type ntGrow struct {
	Sp      string `json:"sp"`
	Born    int    `json:"b"`
	Stage   int    `json:"st"`
	Pause   int    `json:"pa,omitempty"`
	LastDay int    `json:"ld,omitempty"`
}

type ntRuin struct {
	Epoch int    `json:"e"`
	Occ   string `json:"o"`
	Log   int    `json:"l,omitempty"`
	Tiles []int  `json:"t,omitempty"`
	Orig  []int  `json:"or,omitempty"`
}

type ntGhost struct {
	Stage  int    `json:"st"`
	Cause  string `json:"c"`
	Zone   string `json:"z"`
	Seeded bool   `json:"se,omitempty"`
	HX     int    `json:"hx,omitempty"`
	HY     int    `json:"hy,omitempty"`
	Done   bool   `json:"d,omitempty"`
	Year   int    `json:"y,omitempty"`
	Event  string `json:"ev,omitempty"`
}

type ntState struct {
	Pur      map[string]*ntPur  `json:"pur,omitempty"`
	Grow     map[int]*ntGrow    `json:"grow,omitempty"`
	Ruin     map[string]*ntRuin `json:"ruin,omitempty"`
	Ghost    map[int]*ntGhost   `json:"ghost,omitempty"`
	Firsts   map[string]bool    `json:"first,omitempty"`
	Raids    int                `json:"raids,omitempty"`
	Hatched  int                `json:"hatched,omitempty"`
	PoachDay int                `json:"poach,omitempty"`
}

func (g *Game) nt() *ntState {
	st := g.ext("nt", func() any { return &ntState{} }).(*ntState)
	if st.Pur == nil {
		st.Pur = map[string]*ntPur{}
	}
	if st.Grow == nil {
		st.Grow = map[int]*ntGrow{}
	}
	if st.Ruin == nil {
		st.Ruin = map[string]*ntRuin{}
	}
	if st.Ghost == nil {
		st.Ghost = map[int]*ntGhost{}
	}
	if st.Firsts == nil {
		st.Firsts = map[string]bool{}
	}
	return st
}

func (st *ntState) first(k string) bool {
	if st.Firsts[k] {
		return false
	}
	st.Firsts[k] = true
	return true
}

func ntCap(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// ---- species ----

var ntSpByHash map[int]*bsSp

func ntSpHash(s *bsSp) int { return int(HashStr("nt:"+s.ID)%2000000000) + 1 }

func ntSpOf(h int) *bsSp {
	if ntSpByHash == nil {
		ntSpByHash = map[int]*bsSp{}
		for _, s := range bsAll {
			ntSpByHash[ntSpHash(s)] = s
		}
	}
	return ntSpByHash[h]
}

func ntNestable(s *bsSp) bool {
	if s == nil || s.Aquatic() || s.Reef > 0 {
		return false
	}
	switch s.Group {
	case "bird", "rept", "dino", "ptero", "ins", "arach":
		return true
	}
	return false
}

func ntNestWord(s *bsSp) string {
	switch s.Group {
	case "rept", "dino":
		return "nest mound"
	case "ptero":
		return "cliff nest"
	case "ins":
		return "brood chamber"
	case "arach":
		return "egg-sac web"
	}
	return "nest"
}

// ntBlack: eggs the black market and collectors fight over.
func ntBlack(s *bsSp) bool {
	return s != nil && (s.Rarity >= 4 || s.Extinct() || bcLostWorld(s))
}

func ntEggValue(s *bsSp) int {
	mult := [6]int{0, 1, 2, 3, 8, 20}[s.Rarity]
	v := (8 + s.Size*6) * mult
	if s.Extinct() {
		v *= 2
	}
	return v
}

func ntNeedDays(s *bsSp) int { return clampi(4+s.Size*2, 5, 18) }

func ntNestH(zkey string, x, y int) int {
	return int(HashStr(fmt.Sprintf("%s@%d,%d", zkey, x, y))%1000000000) + 1
}

func ntFlagInt(a *Actor, prefix string) int {
	for k := range a.Flags {
		if strings.HasPrefix(k, prefix) {
			n, _ := strconv.Atoi(k[len(prefix):])
			return n
		}
	}
	return 0
}

func ntNestFlag(h int) string { return "ntn:" + strconv.Itoa(h) }

func (g *Game) ntMakeEgg(s *bsSp, h, x, y int) *Item {
	it := &Item{ID: g.newID(), Kind: "egg", Name: "egg of " + s.Display(), Glyph: 'o', F: CWhite, Value: ntEggValue(s), Weight: 1 + s.Size/2,
		Effect: "hearty", X: x, Y: y, Extra: map[string]int{"ntsp": ntSpHash(s), "ntnest": h, "ntday": g.Day()},
		Lore: fmt.Sprintf("An egg of %s. Keep it warm in your pack and it may hatch; collectors pay well, and cooks use it as a hearty ingredient.", s.Display())}
	if ntBlack(s) {
		it.F = CMagenta
		it.Rarity = "rare"
		if s.Rarity >= 5 || s.Extinct() {
			it.Rarity = "legendary"
		}
	}
	return it
}

// ---- zone: creating nests ----

func ntSpot(zn *Zone, r *Rng, s *bsSp) (int, int, bool) { return bsSpot(zn, r, s, -1, -1) }

func ntZone(g *Game, zn *Zone, _ *Rng) {
	if g.W == nil || len(g.W.Biome) == 0 || zn == nil {
		return
	}
	for _, a := range zn.Actors {
		if a.Has("brood") {
			g.ntBroodNest(zn, a)
			break
		}
	}
	r := NewRng(g.W.Seed + ":nt:zone:" + zn.Key)
	var pool []*bsSp
	var ws []float64
	chance, nn := 0.0, 1
	if kind := g.bsLostAt(zn.WX, zn.WY, zn.ZX, zn.ZY, zn.Z); kind != "" && bsLostZoneOK(zn, kind) {
		p, w := bsLostPool(kind)
		for i, s := range p {
			if ntNestable(s) {
				pool = append(pool, s)
				ws = append(ws, w[i])
			}
		}
		chance, nn = 0.8, r.Int(1, 2)
	} else {
		switch zn.Kind {
		case "surface":
			chance = 0.28
		case "outskirts":
			chance = 0.1
		default:
			return
		}
		if zn.Z != 0 || zn.Sky || zn.Underwater {
			return
		}
		pl := g.bsPlaceOf(zn)
		for _, s := range bsAll {
			if !ntNestable(s) {
				continue
			}
			if w := s.weight(pl); w > 0 {
				pool = append(pool, s)
				ws = append(ws, w)
			}
		}
	}
	if len(pool) == 0 || !r.Chance(chance) {
		return
	}
	for i := 0; i < nn; i++ {
		s := bsDraw(r, pool, ws)
		if s == nil {
			break
		}
		x, y, ok := ntSpot(zn, r, s)
		if !ok {
			continue
		}
		h := ntNestH(zn.Key, x, y)
		zn.Items = append(zn.Items, &Item{ID: g.newID(), Kind: "nest", Name: s.Display() + " " + ntNestWord(s), Glyph: '¤', F: CBrown, Weight: 1, X: x, Y: y,
			Extra: map[string]int{"ntnest": h, "ntsp": ntSpHash(s)}, Lore: "Twigs, mud and down: someone's home. Leave it be, or take what is in it."})
	}
	g.ntRefresh(zn)
}

func (g *Game) ntBroodNest(zn *Zone, boss *Actor) {
	for _, it := range zn.Items {
		if it.Kind == "nest" && it.Extra["ntbrood"] > 0 {
			return
		}
	}
	r := NewRng(g.W.Seed + ":nt:brood:" + zn.Key)
	x, y, ok := zn.FreeSpot(r, func(x, y int) bool { return abs(x-boss.X) <= 5 && abs(y-boss.Y) <= 3 })
	if !ok {
		return
	}
	h := ntNestH(zn.Key, x, y)
	boss.Set(ntNestFlag(h))
	zn.Items = append(zn.Items, &Item{ID: g.newID(), Kind: "nest", Name: "brood nest", Glyph: '¤', F: CDarkRed, Weight: 1, X: x, Y: y,
		Extra: map[string]int{"ntnest": h, "ntbrood": 1}, Lore: "A heap of bones and scorched earth. Whatever lies in it is warm."})
	line := strings.TrimPrefix(boss.Name, "Brood of ")
	for i, n := 0, r.Int(2, 3); i < n; i++ {
		zn.Items = append(zn.Items, &Item{ID: g.newID(), Kind: "egg", Name: "brood egg of " + line, Glyph: 'o', F: CDarkRed, Value: 150 + boss.Level*20,
			Weight: 3, Rarity: "legendary", X: x, Y: y, Extra: map[string]int{"ntnest": h, "ntbrood": 1, "ntday": g.Day()},
			Lore: "The egg of a slain monster's line. It cannot be hatched, but there are buyers for such things."})
	}
}

// ntRefresh brings every nest of a zone in line with the season: spring lays, summer hatches, autumn and
// winter empty it. Side rngs only; never touches g.Cur.
func (g *Game) ntRefresh(zn *Zone) {
	yr, season := g.Day()/YearDays, g.Season()
	for _, n := range append([]*Item(nil), zn.Items...) {
		if n.Kind != "nest" || n.Extra == nil || n.Extra["ntnest"] == 0 || n.Extra["ntbrood"] > 0 {
			continue
		}
		s := ntSpOf(n.Extra["ntsp"])
		if s == nil {
			continue
		}
		h := n.Extra["ntnest"]
		if season >= 2 {
			if n.Extra["ntyr"] != 0 {
				n.Extra["ntyr"] = 0
				ntClearNest(zn, h)
			}
			continue
		}
		if n.Extra["ntyr"] != yr+1 {
			g.ntLay(zn, n, s, h, yr)
			n.Extra["ntyr"] = yr + 1
		}
		if season == 1 && n.Extra["nthatch"] != yr+1 {
			n.Extra["nthatch"] = yr + 1
			g.ntHatchNest(zn, n, s, h, yr)
		}
	}
}

func ntClearNest(zn *Zone, h int) {
	keep := zn.Items[:0]
	for _, it := range zn.Items {
		if it.Kind == "egg" && it.Extra != nil && it.Extra["ntnest"] == h {
			continue
		}
		keep = append(keep, it)
	}
	zn.Items = keep
	fl := ntNestFlag(h)
	ak := zn.Actors[:0]
	for _, a := range zn.Actors {
		if a.Has(fl) && !a.Angry && !a.Has("ntavenge") {
			continue
		}
		ak = append(ak, a)
	}
	zn.Actors = ak
}

func (g *Game) ntLay(zn *Zone, n *Item, s *bsSp, h, yr int) {
	r := NewRng(fmt.Sprintf("%s:nt:lay:%d:%d", g.W.Seed, h, yr))
	maxEggs := clampi(8-s.Size, 2, 6)
	cnt := r.Int(1, maxEggs)
	if s.Group == "ins" || s.Group == "arach" {
		cnt = r.Int(3, 6)
	}
	for i := 0; i < cnt; i++ {
		zn.Items = append(zn.Items, g.ntMakeEgg(s, h, n.X, n.Y))
	}
	np := 1 + boolInt(r.Chance(0.6))
	if s.Group == "ins" {
		np = 2 + r.Int(0, 1)
	}
	for i := 0; i < np; i++ {
		x, y, ok := bsSpot(zn, r, s, n.X, n.Y)
		if !ok {
			continue
		}
		lvl := tplByID[s.Tpl()].Lvl
		if s.Hostile() {
			lvl = maxi(lvl, zn.Level-2)
		}
		a := g.bsSpawn(s, lvl, r)
		a.X, a.Y, a.HomeX, a.HomeY = x, y, n.X, n.Y
		a.Set("ntpar")
		a.Set(ntNestFlag(h))
		zn.Actors = append(zn.Actors, a)
	}
}

func (g *Game) ntHatchNest(zn *Zone, n *Item, s *bsSp, h, yr int) {
	r := NewRng(fmt.Sprintf("%s:nt:hatch:%d:%d", g.W.Seed, h, yr))
	fl := ntNestFlag(h)
	parents := 0
	for _, a := range zn.Actors {
		if a.HP > 0 && a.Has(fl) && a.Has("ntpar") {
			parents++
		}
	}
	keep := zn.Items[:0]
	var spawn int
	for _, it := range zn.Items {
		if it.Kind == "egg" && it.Extra != nil && it.Extra["ntnest"] == h && r.Chance(0.55) {
			spawn++
			continue
		}
		keep = append(keep, it)
	}
	zn.Items = keep
	for i := 0; i < spawn; i++ {
		x, y, ok := bsSpot(zn, r, s, n.X, n.Y)
		if !ok {
			continue
		}
		a := g.bsSpawn(s, 1, r)
		a.X, a.Y, a.HomeX, a.HomeY = x, y, n.X, n.Y
		a.AI = "wander"
		ntScale(g, a, s, 0)
		a.Name = "hatchling " + s.Display()
		a.Set("ntyg")
		a.Set(fl)
		if parents == 0 {
			a.Set("ntorph")
		}
		zn.Actors = append(zn.Actors, a)
	}
}

// ---- growth stages ----

func ntScale(g *Game, a *Actor, s *bsSp, stage int) {
	ref := g.bsSpawn(s, tplByID[s.Tpl()].Lvl, NewRng(g.W.Seed+":nt:ref:"+s.ID))
	f := [3]float64{0.4, 0.7, 1}[clampi(stage, 0, 2)]
	ratio := 1.0
	if a.MaxHP > 0 {
		ratio = float64(a.HP) / float64(a.MaxHP)
	}
	a.Level = maxi(1, int(float64(ref.Level)*f+0.5))
	for i := range a.Attr {
		v := ref.Attr[i]
		switch i {
		case 0, 2:
			v = int(float64(v)*f + 0.5)
		case 1:
			v = int(float64(v)*(1+f)/2 + 0.5)
		}
		a.Attr[i] = maxi(3, v)
	}
	a.MaxHP = maxi(3, int(float64(ref.MaxHP)*f+0.5))
	a.HP = clampi(int(ratio*float64(a.MaxHP)+0.5), 1, a.MaxHP)
	a.AVNat = ref.AVNat * stage / 2
	a.DVNat, a.Speed = ref.DVNat, ref.Speed
	for i, p := range a.Body {
		if p.Natural == nil || i >= len(ref.Body) || ref.Body[i].Natural == nil {
			continue
		}
		rn := ref.Body[i].Natural
		p.Natural.DN = rn.DN
		p.Natural.DS = maxi(1, int(float64(rn.DS)*f+0.5))
		p.Natural.PV = rn.PV * boolInt(stage >= 2)
	}
}

func ntStageTitle(s *bsSp, stage int) string {
	switch stage {
	case 0:
		return "tamed hatchling " + s.Display()
	case 1:
		return "tamed young " + s.Display()
	}
	return "tamed " + s.Display()
}

// ---- turn ----

func ntTurn(g *Game) {
	if g.P == nil || g.Dead || g.Cur == nil || g.W == nil {
		return
	}
	st := g.nt()
	g.ntParents()
	g.ntRaidCheck(st)
	if g.Turn%5 == 0 {
		g.ntPursue(st)
		g.ntAdopt(st)
		g.ntSpotNests(st)
	}
	g.ntEggs(st)
	if len(st.Ghost) > 0 {
		g.ntStory(st)
	}
	if g.Turn%60 == 0 && len(st.Grow) > 0 {
		g.ntGrowTick(st)
	}
}

func (g *Game) ntParents() {
	p := g.P
	rad := 6
	if p.Has("sneak") || p.St("hidden") > 0 {
		rad = 3
	}
	for _, a := range g.Cur.Actors {
		if a.HP <= 0 || !a.Has("ntpar") || a.Angry || a.Has("ntavenge") || a.AI == "follower" {
			continue
		}
		s := bsByTpl[a.Tpl]
		if s == nil {
			continue
		}
		if maxi(abs(p.X-a.HomeX), abs(p.Y-a.HomeY)) <= rad && s.Temper != "passive" && s.Temper != "skittish" && !(s.Temper == "herd" && s.Size <= 2) {
			g.Msg(fmt.Sprintf("{{R|%s rears up to defend its nest.}}", ntCap(g.The(a))))
			h := ntFlagInt(a, "ntn:")
			for _, o := range g.Cur.Actors {
				if o.HP > 0 && o.Has("ntpar") && o.Has(ntNestFlag(h)) {
					g.provoke(o)
					o.AI = "hostile"
				}
			}
			continue
		}
		if g.Turn%4 == 0 && maxi(abs(a.X-a.HomeX), abs(a.Y-a.HomeY)) > 4 {
			g.bsStepTo(a, &Actor{X: a.HomeX, Y: a.HomeY})
		}
	}
}

func (g *Game) ntRaidCheck(st *ntState) {
	for _, it := range append([]*Item(nil), g.P.Inv...) {
		if it.Extra == nil || it.Extra["ntnest"] == 0 {
			continue
		}
		switch {
		case it.Kind == "nest":
			g.removeInv(it)
			g.Cur.Items = append(g.Cur.Items, it)
			g.Msg("{{K|You leave the nesting material where it lay.}}")
		case it.Kind == "egg" && it.Extra["ntraid"] == 0:
			g.ntRaid(st, it)
		}
	}
}

func (g *Game) ntRaid(st *ntState, it *Item) {
	it.Extra["ntraid"] = 1
	it.Extra["ntlast"] = g.Turn
	h := it.Extra["ntnest"]
	s := ntSpOf(it.Extra["ntsp"])
	st.Raids++
	yr := g.Day() / YearDays
	for _, o := range g.Cur.Items {
		if o.Kind == "nest" && o.Extra != nil && o.Extra["ntnest"] == h {
			o.Extra["ntraid"] = yr + 1
		}
	}
	fl := ntNestFlag(h)
	var par []*Actor
	for _, a := range g.Cur.Actors {
		if a.HP > 0 && a.Has(fl) && !a.Has("ntavenge") && !a.Has("ntyg") {
			par = append(par, a)
		}
	}
	rage := func() {
		for _, a := range par {
			g.provoke(a)
			a.AI = "hostile"
		}
	}
	switch {
	case it.Extra["ntbrood"] > 0:
		rage()
		g.Msg("{{R|The brood-mother's roar shakes the lair as you lift her egg.}}")
		if st.first("broodegg") {
			g.deed("nest", fmt.Sprintf("The wanderer %s stole an egg from a monster's brood.", g.P.Name), 0, 0)
		}
	case s == nil:
	case s.Temper == "passive":
		g.Msg(fmt.Sprintf("{{K|The %s pay no mind as you take the egg.}}", s.Display()))
	case s.Temper == "skittish":
		for _, a := range par {
			g.bs().alarm[a.ID] = g.Turn + 150
		}
		g.Msg(fmt.Sprintf("{{y|The %s bolt, shrieking, as you take the egg.}}", s.Display()))
	default:
		rage()
		days := 2 + s.Size/2 + boolInt(s.Rarity >= 4)
		if len(par) > 0 {
			st.Pur[strconv.Itoa(h)] = &ntPur{Sp: s.ID, Zone: g.Cur.Key, WX: g.Cur.WX, WY: g.Cur.WY, Until: g.Turn + days*DayLen, Par: len(par)}
			g.Msg(fmt.Sprintf("{{R|The %s screams: you have robbed its nest. It will hunt you.}}", s.Display()))
		} else {
			g.Msg("{{K|No parent is left to avenge the egg.}}")
		}
	}
	if s != nil {
		if st.first("raid") {
			g.deed("nest", fmt.Sprintf("The wanderer %s robbed the nest of %s %s.", g.P.Name, bsArticle(s.Display()), s.Display()), 0, 0)
		}
		if ntBlack(s) {
			g.Msg("{{M|Collectors and poachers would pay dearly for this egg.}}")
			if st.first("raidrare") {
				g.deed("nest", fmt.Sprintf("The wanderer %s took the egg of a rare %s.", g.P.Name, s.Display()), 0, 0)
			}
		}
	}
}

func (g *Game) ntSpotNests(st *ntState) {
	for _, n := range g.Cur.Items {
		if n.Kind != "nest" || n.Extra == nil || n.Extra["ntseen"] > 0 || !g.visible(n.X, n.Y) || abs(n.X-g.P.X) > 12 {
			continue
		}
		n.Extra["ntseen"] = 1
		eggs := 0
		for _, o := range g.Cur.Items {
			if o.Kind == "egg" && o.X == n.X && o.Y == n.Y {
				eggs++
			}
		}
		switch {
		case eggs > 0:
			g.Msg(fmt.Sprintf("{{c|You spot a %s with %d egg%s.}}", n.Name, eggs, map[bool]string{true: "", false: "s"}[eggs == 1]))
		case n.Extra["ntbrood"] > 0:
		default:
			g.Msg(fmt.Sprintf("{{K|You spot an empty %s.}}", n.Name))
		}
	}
}

func (g *Game) ntPursue(st *ntState) {
	if len(st.Pur) == 0 {
		return
	}
	keys := make([]string, 0, len(st.Pur))
	for k := range st.Pur {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	z := g.Cur
	for _, k := range keys {
		pu := st.Pur[k]
		if g.Turn > pu.Until || pu.Par <= 0 {
			delete(st.Pur, k)
			continue
		}
		if z.Key == pu.Zone || z.Z != 0 || z.Sky || z.Underwater || (z.Kind != "surface" && z.Kind != "outskirts") {
			continue
		}
		if maxi(abs(z.WX-pu.WX), abs(z.WY-pu.WY)) > 3 || g.Turn-pu.Last < 90 || pu.Spawns >= 5 {
			continue
		}
		s := bsByID[pu.Sp]
		if s == nil {
			delete(st.Pur, k)
			continue
		}
		h, _ := strconv.Atoi(k)
		fl := ntNestFlag(h)
		here := false
		for _, a := range z.Actors {
			if a.HP > 0 && a.Has("ntavenge") && a.Has(fl) {
				here = true
			}
		}
		if here {
			continue
		}
		r := NewRng(fmt.Sprintf("%s:nt:av:%d:%d", g.W.Seed, h, g.Turn))
		x, y, ok := z.FreeSpot(r, func(x, y int) bool { return abs(x-g.P.X) >= 12 || abs(y-g.P.Y) >= 7 })
		if !ok {
			continue
		}
		a := g.bsSpawn(s, maxi(tplByID[s.Tpl()].Lvl, z.Level-2), r)
		a.X, a.Y, a.HomeX, a.HomeY = x, y, x, y
		a.Name = "vengeful " + s.Display()
		a.MaxHP += a.MaxHP / 4
		a.HP = a.MaxHP
		a.Set("ntpar")
		a.Set("ntavenge")
		a.Set(fl)
		g.provoke(a)
		a.AI = "hostile"
		z.Actors = append(z.Actors, a)
		pu.Last = g.Turn
		pu.Spawns++
		g.Msg(fmt.Sprintf("{{R|A %s crashes out of the wild: the parent has tracked you down.}}", s.Display()))
	}
}

func (g *Game) ntAdopt(st *ntState) {
	for _, a := range g.Cur.Actors {
		if a.HP <= 0 || !a.Has("ntyg") || !a.Has("ntorph") || a.AI == "follower" || dist(a, g.P) > 1 || !g.tmSlotFree() {
			continue
		}
		s := bsByTpl[a.Tpl]
		if s == nil {
			continue
		}
		delete(a.Flags, "ntyg")
		delete(a.Flags, "ntorph")
		g.tmJoin(a, s, 85, g.Cur.Key)
		ntScale(g, a, s, 0)
		a.Title = ntStageTitle(s, 0)
		st.Grow[a.ID] = &ntGrow{Sp: s.ID, Born: g.Day()}
		g.Msg(fmt.Sprintf("{{G|The orphaned hatchling presses against your leg and will not leave. %s is yours.}}", a.Name))
		if st.first("orphan") {
			g.deed("companion", fmt.Sprintf("The wanderer %s took in an orphaned %s hatchling.", g.P.Name, s.Display()), 0, 0)
		}
		return
	}
}

// ---- eggs in the pack ----

func (g *Game) ntWarmth() int {
	z := g.Cur
	if z.Kind == "town" || g.homeHere() != nil {
		return 2
	}
	for dy := -2; dy <= 2; dy++ {
		for dx := -2; dx <= 2; dx++ {
			x, y := g.P.X+dx, g.P.Y+dy
			if z.In(x, y) && (z.Fire[y*MW+x] > 0 || z.At(x, y) == TBrazier) {
				return 2
			}
		}
	}
	if z.Underwater || z.Sky {
		return 0
	}
	if z.Z > 0 {
		return 1
	}
	t := 14
	if p := g.W.Idx(z.WX, z.WY); p >= 0 && p < len(g.W.Temp) {
		t = g.W.Temp[p]
	}
	if t <= 0 || g.Season() == 3 && t < 10 {
		return 0
	}
	return 1
}

func (g *Game) ntEggs(st *ntState) {
	for _, it := range append([]*Item(nil), g.P.Inv...) {
		if it.Kind != "egg" || it.Extra == nil || it.Extra["ntsp"] == 0 || it.Extra["ntdead"] > 0 || it.Extra["ntbrood"] > 0 {
			continue
		}
		last := it.Extra["ntlast"]
		if last == 0 {
			it.Extra["ntlast"] = g.Turn
			continue
		}
		dt := g.Turn - last
		if dt < 20 {
			continue
		}
		it.Extra["ntlast"] = g.Turn
		dt = mini(dt, 120)
		s := ntSpOf(it.Extra["ntsp"])
		if s == nil {
			continue
		}
		switch g.ntWarmth() {
		case 2:
			it.Extra["ntwarm"] += dt
			it.Extra["ntchill"] = maxi(0, it.Extra["ntchill"]-dt)
		case 1:
			it.Extra["ntwarm"] += dt / 2
			it.Extra["ntchill"] = maxi(0, it.Extra["ntchill"]-dt/2)
		default:
			it.Extra["ntchill"] += dt
			if it.Extra["ntchill"] > 900 && it.Extra["ntwarn"] == 0 {
				it.Extra["ntwarn"] = 1
				g.Msg("{{y|The egg is chilling in the cold. Find a fire, a roof or milder ground.}}")
			}
			if it.Extra["ntchill"] > 1800 {
				it.Extra["ntdead"] = 1
				it.Name = "cold egg of " + s.Display()
				it.Value = 1
				it.Effect = ""
				it.F = CGray
				it.Rarity = ""
				g.Msg("{{R|The egg has gone cold. Nothing will hatch from it now.}}")
				continue
			}
		}
		if it.Extra["ntwarm"] >= ntNeedDays(s)*DayLen {
			g.ntHatch(st, it, s)
		}
	}
}

func (g *Game) ntHatch(st *ntState, it *Item, s *bsSp) {
	if !g.tmSlotFree() {
		if it.Extra["ntfull"] == 0 {
			it.Extra["ntfull"] = 1
			g.Msg("{{y|The egg rocks and cracks, but you have no room for another follower. It waits.}}")
		}
		return
	}
	g.removeInv(it)
	r := NewRng(fmt.Sprintf("%s:nt:born:%d:%d", g.W.Seed, it.ID, g.Day()))
	a := g.bsSpawn(s, 1, r)
	x, y, ok := g.nearestFree(g.Cur, g.P.X, g.P.Y)
	if !ok {
		x, y = g.P.X, g.P.Y
	}
	a.X, a.Y, a.HomeX, a.HomeY = x, y, x, y
	g.Cur.Actors = append(g.Cur.Actors, a)
	g.tmJoin(a, s, 92, g.Cur.Key)
	ntScale(g, a, s, 0)
	a.Title = ntStageTitle(s, 0)
	st.Grow[a.ID] = &ntGrow{Sp: s.ID, Born: g.Day()}
	st.Hatched++
	g.Msg(fmt.Sprintf("{{G|The egg splits. A %s hatchling blinks at you and imprints: %s will follow you anywhere. Feed it by its diet (talk to it).}}", s.Display(), a.Name))
	g.gainXP(10 + 8*s.Size)
	if st.first("hatch") || ntBlack(s) && st.first("hatchrare") {
		g.deed("companion", fmt.Sprintf("The wanderer %s hatched an egg of %s %s and raised it.", g.P.Name, bsArticle(s.Display()), s.Display()), 0, 0)
	}
}

func (g *Game) ntGrowTick(st *ntState) {
	ids := make([]int, 0, len(st.Grow))
	for id := range st.Grow {
		ids = append(ids, id)
	}
	sort.Ints(ids)
	day := g.Day()
	for _, id := range ids {
		gr := st.Grow[id]
		var a *Actor
		for _, c := range g.localCompanions() {
			if c.ID == id {
				a = c
			}
		}
		s := bsByID[gr.Sp]
		if s == nil {
			delete(st.Grow, id)
			continue
		}
		if a == nil {
			continue
		}
		if gr.LastDay != day {
			gr.LastDay = day
			if f := g.tm().Fed[id]; f > 0 && g.Turn-f > 4320 {
				gr.Pause++
			}
		}
		eff := day - gr.Born - gr.Pause
		stage := 0
		switch {
		case eff >= 10+4*s.Size:
			stage = 2
		case eff >= 4+2*s.Size:
			stage = 1
		}
		if stage <= gr.Stage {
			continue
		}
		gr.Stage = stage
		ntScale(g, a, s, stage)
		a.Title = ntStageTitle(s, stage)
		if stage == 1 {
			g.Msg(fmt.Sprintf("{{G|%s has grown into a young %s.}}", a.Name, s.Display()))
		} else {
			g.Msg(fmt.Sprintf("{{G|%s is fully grown: a mighty %s.}}", a.Name, s.Display()))
			delete(st.Grow, id)
			if st.first("grown") {
				g.deed("companion", fmt.Sprintf("The %s %s, raised from an egg by the wanderer %s, came into its full strength.", s.Display(), a.Name, g.P.Name), 0, 0)
			}
		}
	}
}

// ---- kills ----

func ntKill(g *Game, d, src *Actor) {
	if d == nil || g.W == nil {
		return
	}
	st := g.nt()
	delete(st.Grow, d.ID)
	if d.Has("ntcause") {
		g.ntLaidToRest(st, d)
		return
	}
	h := ntFlagInt(d, "ntn:")
	if h == 0 || d.Has("ntyg") {
		return
	}
	key := strconv.Itoa(h)
	pu := st.Pur[key]
	if d.Has("ntavenge") && pu != nil {
		if nz := g.Zones[pu.Zone]; nz != nil {
			for _, a := range nz.Actors {
				if a.HP > 0 && a.Has("ntpar") && !a.Has("ntavenge") && a.Has(ntNestFlag(h)) {
					g.removeActor(nz, a)
					break
				}
			}
			ntOrphans(g, nz, h)
		}
	}
	if pu != nil {
		pu.Par--
		if pu.Par <= 0 {
			delete(st.Pur, key)
			g.Msg("{{G|The last of the hunting parents is dead.}}")
		}
	}
	if g.Cur != nil {
		ntOrphans(g, g.Cur, h)
	}
	if (src == g.P || src != nil && src.AI == "follower") && st.first("parent") {
		g.deed("nest", fmt.Sprintf("The wanderer %s slew a beast defending its nest.", g.P.Name), 0, 0)
	}
}

func ntOrphans(g *Game, zn *Zone, h int) {
	fl := ntNestFlag(h)
	for _, a := range zn.Actors {
		if a.HP > 0 && a.Has(fl) && a.Has("ntpar") {
			return
		}
	}
	n := 0
	for _, a := range zn.Actors {
		if a.HP > 0 && a.Has(fl) && a.Has("ntyg") && !a.Has("ntorph") {
			a.Set("ntorph")
			n++
		}
	}
	if n > 0 && zn == g.Cur {
		g.Msg("{{c|The hatchlings call for parents that will not come. They are orphans now; one might follow you if you stood close.}}")
	}
}

// ---- enter ----

func ntEnter(g *Game, zn *Zone) {
	if zn == nil || g.W == nil {
		return
	}
	g.ntRefresh(zn)
	st := g.nt()
	if bd := g.ntRuinBurg(zn); bd != nil {
		if g.ntGhostTown(bd) {
			g.ntGhostEnter(zn, bd)
		} else {
			g.ntRuinApply(zn, bd)
		}
	}
	// poachers follow a rare egg
	if zn.Kind != "town" && zn.Z == 0 && st.PoachDay != g.Day() && g.ntBlackEgg() != nil {
		st.PoachDay = g.Day()
		r := NewRng(fmt.Sprintf("%s:nt:poach:%d:%s", g.W.Seed, g.Day(), zn.Key))
		if r.Chance(0.4) {
			for i := 0; i < 2+r.Int(0, 1); i++ {
				if x, y, ok := zn.FreeSpot(r, func(x, y int) bool { return abs(x-g.P.X) >= 8 }); ok {
					a := g.Spawn("bandit", zn.Level, r)
					a.Name = "egg poacher"
					a.X, a.Y = x, y
					a.Angry = true
					zn.Actors = append(zn.Actors, a)
				}
			}
			g.Msg("{{R|Word of the rare egg you carry has spread: poachers have come for it.}}")
		}
	} else if zn.Kind == "town" && g.ntBlackEgg() != nil && st.first(fmt.Sprintf("rumour%d", g.Day())) {
		g.Msg("{{y|Whispers in the market: someone is carrying an egg a collector would kill for.}}")
	}
}

func (g *Game) ntBlackEgg() *Item {
	for _, it := range g.P.Inv {
		if it.Kind == "egg" && it.Extra != nil && it.Extra["ntdead"] == 0 && (it.Extra["ntbrood"] > 0 || ntBlack(ntSpOf(it.Extra["ntsp"]))) {
			return it
		}
	}
	return nil
}

// ---- living ruins ----

func (g *Game) ntRuinBurg(zn *Zone) *BurgData {
	if zn.Kind != "ruin" || zn.Z != 0 || zn.ZX != 1 || zn.ZY != 1 {
		return nil
	}
	p := g.W.Idx(zn.WX, zn.WY)
	if p < 0 || p >= len(g.W.Burg) {
		return nil
	}
	bd := g.W.burgByID[g.W.Burg[p]]
	if bd == nil || bd.Alive {
		return nil
	}
	return bd
}

func (g *Game) ntGhostTown(bd *BurgData) bool {
	if gs := g.nt().Ghost[bd.ID]; gs != nil {
		return !gs.Done
	}
	return NewRng(fmt.Sprintf("%s:nt:gt:%d", g.W.Seed, bd.ID)).Chance(0.55)
}

func ntPickOcc(r *Rng, season int) string {
	type w struct {
		id string
		w  float64
	}
	ws := []w{{"", 22}, {"bandits", 20}, {"monks", 10}, {"den", 15}, {"nomads", 10}, {"plague", 8}, {"haunted", 12}}
	if season == 3 {
		ws[4].w = 2
	}
	tot := 0.0
	for _, x := range ws {
		tot += x.w
	}
	v := r.Float() * tot
	for _, x := range ws {
		if v -= x.w; v <= 0 {
			return x.id
		}
	}
	return ""
}

func (g *Game) ntRuinApply(zn *Zone, bd *BurgData) {
	st := g.nt()
	epoch := g.Day() / (SeasonDays * 2)
	r := NewRng(fmt.Sprintf("%s:nt:occ:%d:%d", g.W.Seed, bd.ID, epoch))
	occ := ntPickOcc(r, g.Season())
	rs := st.Ruin[zn.Key]
	if rs != nil && rs.Epoch == epoch+1 {
		return
	}
	if rs != nil && rs.Occ == occ {
		rs.Epoch = epoch + 1
		return
	}
	if rs == nil {
		rs = &ntRuin{}
		st.Ruin[zn.Key] = rs
	}
	g.ntRuinClear(zn, rs)
	rs.Epoch, rs.Occ = epoch+1, occ
	if occ != "" {
		g.ntRuinPopulate(zn, rs, occ, epoch)
	}
	g.ntRuinLog(zn, rs, occ)
	if def, ok := ntOccs[occ]; ok {
		g.Msg(def.Enter)
	} else if occ == "" {
		g.Msg("{{K|The ruin is quiet. Whoever sheltered here has moved on.}}")
	}
}

func (g *Game) ntRuinClear(zn *Zone, rs *ntRuin) {
	ak := zn.Actors[:0]
	for _, a := range zn.Actors {
		if a.Has("ntocc") && a.AI != "follower" {
			continue
		}
		ak = append(ak, a)
	}
	zn.Actors = ak
	ik := zn.Items[:0]
	for _, it := range zn.Items {
		if it.Extra != nil && it.Extra["ntocc"] > 0 {
			continue
		}
		ik = append(ik, it)
	}
	zn.Items = ik
	for i, idx := range rs.Tiles {
		if idx >= 0 && idx < len(zn.T) && i < len(rs.Orig) {
			zn.T[idx] = Terrain(rs.Orig[i])
		}
	}
	rs.Tiles, rs.Orig = nil, nil
}

func (g *Game) ntSetTile(zn *Zone, rs *ntRuin, x, y int, t Terrain) {
	if !zn.In(x, y) || zn.ActorAt(x, y) != nil || (x == g.P.X && y == g.P.Y) {
		return
	}
	idx := y*MW + x
	rs.Tiles = append(rs.Tiles, idx)
	rs.Orig = append(rs.Orig, int(zn.T[idx]))
	zn.T[idx] = t
}

func ntOcc(a *Actor, occ string) {
	a.Set("ntocc")
	a.Set("nto:" + occ)
}

func (g *Game) ntTrimLoot(zn *Zone, n int) {
	keep := zn.Items[:0]
	for _, it := range zn.Items {
		if n > 0 && it.SimArt == 0 && it.Quest == 0 && (it.Kind == "weapon" || it.Kind == "armor" || it.Kind == "ranged") && (it.Extra == nil || it.Extra["ntocc"] == 0) {
			n--
			continue
		}
		keep = append(keep, it)
	}
	zn.Items = keep
}

func (g *Game) ntRuinPopulate(zn *Zone, rs *ntRuin, occ string, epoch int) {
	r := NewRng(fmt.Sprintf("%s:nt:occp:%s:%d", g.W.Seed, zn.Key, epoch))
	put := func(a *Actor, x, y int) {
		a.X, a.Y, a.HomeX, a.HomeY = x, y, x, y
		ntOcc(a, occ)
		zn.Actors = append(zn.Actors, a)
	}
	loot := func(n, tier int) {
		for i := 0; i < n; i++ {
			if x, y, ok := zn.FreeSpot(r, nil); ok {
				it := g.Loot(r, tier)
				it.X, it.Y = x, y
				if it.Extra == nil {
					it.Extra = map[string]int{}
				}
				it.Extra["ntocc"] = 1
				zn.Items = append(zn.Items, it)
			}
		}
	}
	peaceful := func(a *Actor, name string, hpDiv int) {
		a.Name, a.Faction, a.AI, a.Angry = name, "ntfolk", "wander", false
		a.MaxHP = maxi(4, a.MaxHP/hpDiv)
		a.HP = a.MaxHP
	}
	cx, cy, ok := zn.FreeSpot(r, nil)
	if !ok {
		return
	}
	near := func() (int, int, bool) {
		return zn.FreeSpot(r, func(x, y int) bool { return abs(x-cx) <= 4 && abs(y-cy) <= 3 })
	}
	switch occ {
	case "bandits":
		g.ntSetTile(zn, rs, cx, cy, TBrazier)
		n := r.Int(3, 5)
		for i := 0; i < n; i++ {
			tp := "bandit"
			if i%3 == 2 {
				tp = "archer"
			}
			if i == 0 && n >= 4 {
				tp = "chief"
			}
			if x, y, ok := near(); ok {
				a := g.Spawn(tp, zn.Level, r)
				if tp != "chief" {
					a.Name = "ruin squatter"
				}
				put(a, x, y)
			}
		}
		loot(r.Int(2, 3), zn.Level/3+1)
	case "monks":
		g.ntSetTile(zn, rs, cx, cy, TAltar)
		for i := 0; i < r.Int(1, 2); i++ {
			if x, y, ok := near(); ok {
				a := g.Spawn("bandit", zn.Level, r)
				peaceful(a, "hermit monk", 1)
				a.Inv = append(a.Inv, g.makeIngredient("hermit's tea leaves", "mending", 3, CGreen))
				put(a, x, y)
			}
		}
		if x, y, ok := near(); ok {
			it := g.MakeBook("A Hermit's Meditations", "Hold still and the ruin tells its own story: a hearth here, a voice there. Nothing is ever really lost, only left alone.")
			it.X, it.Y, it.Extra = x, y, map[string]int{"ntocc": 1}
			zn.Items = append(zn.Items, it)
		}
		g.ntTrimLoot(zn, 2)
	case "den":
		var pool []*bsSp
		var ws []float64
		pl := g.bsPlaceOf(zn)
		for _, s := range bsAll {
			if s.Hostile() && s.Size >= 3 && s.Size <= 5 && !s.Aquatic() {
				if w := s.weight(pl); w > 0 {
					pool = append(pool, s)
					ws = append(ws, w)
				}
			}
		}
		for i := 0; i < r.Int(1, 2); i++ {
			if x, y, ok := near(); ok {
				var a *Actor
				if s := bsDraw(r, pool, ws); s != nil {
					a = g.bsSpawn(s, maxi(tplByID[s.Tpl()].Lvl, zn.Level), r)
				} else {
					a = g.Spawn("bear", zn.Level, r)
				}
				put(a, x, y)
			}
		}
		for i := 0; i < 6; i++ {
			if x, y, ok := near(); ok {
				g.ntSetTile(zn, rs, x, y, TBones)
			}
		}
		loot(1, zn.Level/3+2)
	case "nomads":
		for i := 0; i < 4; i++ {
			if x, y, ok := near(); ok {
				g.ntSetTile(zn, rs, x, y, TCarpet)
			}
		}
		for i := 0; i < r.Int(3, 4); i++ {
			if x, y, ok := near(); ok {
				a := g.Spawn("bandit", zn.Level, r)
				peaceful(a, "nomad", 1)
				if i == 0 {
					a.Role, a.Name = "merchant", "nomad trader"
					a.Inv = append(a.Inv, g.makeIngredient("nomad travel bread", "hearty", 3, CBrown), g.makeIngredient("salted road meat", "thick", 2, CRed), g.MakeFood(r, 2))
				}
				put(a, x, y)
			}
		}
	case "plague":
		for i := 0; i < r.Int(2, 4); i++ {
			if x, y, ok := near(); ok {
				a := g.Spawn("bandit", zn.Level, r)
				peaceful(a, "plague refugee", 3)
				put(a, x, y)
			}
		}
		g.ntTrimLoot(zn, 2)
	case "haunted":
		for i := 0; i < r.Int(2, 3); i++ {
			if x, y, ok := zn.FreeSpot(r, nil); ok {
				a := g.Spawn("ashwraith", zn.Level, r)
				a.Name = "restless dead"
				put(a, x, y)
			}
		}
		loot(2, zn.Level/3+2)
	}
}

func (g *Game) ntRuinLog(zn *Zone, rs *ntRuin, occ string) {
	if rs.Log == 0 {
		x, y, ok := zn.FreeSpot(NewRng(g.W.Seed+":nt:log:"+zn.Key), nil)
		if !ok {
			return
		}
		rs.Log = y*MW + x + 1
	}
	idx := rs.Log - 1
	who := "No one lives here now."
	if d, ok := ntOccs[occ]; ok {
		who = d.Who
	}
	zn.T[idx] = TInscription
	zn.Notes[idx] = fmt.Sprintf("Ruin log, year %d: %s", g.W.Year+g.Day()/YearDays, who)
}

// ---- ghost towns ----

func ntCauseOf(w *WorldData, bd *BurgData) (cause, text string, year int) {
	evs := w.eventsAt[bd.P]
	for i := len(evs) - 1; i >= 0; i-- {
		e := w.Events[evs[i]]
		t := strings.ToLower(e.Text)
		for _, cw := range ntCauseWords {
			for _, k := range cw.Words {
				if strings.Contains(t, k) {
					return cw.Cause, e.Text, e.Y
				}
			}
		}
	}
	if len(evs) > 0 {
		e := w.Events[evs[len(evs)-1]]
		text, year = e.Text, e.Y
	}
	cs := []string{"plague", "curse", "beast", "lord"}
	return cs[HashStr(fmt.Sprintf("%s:nt:cause:%d", w.Seed, bd.ID))%4], text, year
}

func (g *Game) ntDiary(gs *ntGhost, bd *BurgData, n int) *Item {
	who := ntWriters[HashStr(fmt.Sprintf("%s:nt:writer:%d", g.W.Seed, bd.ID))%uint64(len(ntWriters))]
	year := gs.Year
	if year == 0 {
		year = g.W.Year - 10
	}
	ev := gs.Event
	if ev != "" {
		ev = "(" + ev + ")"
	}
	txt := strings.NewReplacer("{burg}", bd.Name, "{who}", who, "{year}", strconv.Itoa(year), "{event}", ev).Replace(ntDiaries[gs.Cause][n-1])
	it := g.MakeBook(fmt.Sprintf("Diary of %s (page %d)", who, n), txt)
	it.Extra = map[string]int{"ntg": bd.ID, "ntd": n}
	return it
}

func (g *Game) ntGhostEnter(zn *Zone, bd *BurgData) {
	st := g.nt()
	gs := st.Ghost[bd.ID]
	if gs == nil {
		c, t, y := ntCauseOf(g.W, bd)
		gs = &ntGhost{Cause: c, Zone: zn.Key, Event: t, Year: y}
		st.Ghost[bd.ID] = gs
	}
	if gs.Done {
		return
	}
	if !gs.Seeded {
		gs.Seeded = true
		r := NewRng(fmt.Sprintf("%s:nt:gts:%d", g.W.Seed, bd.ID))
		if x, y, ok := zn.FreeSpot(r, nil); ok {
			it := g.ntDiary(gs, bd, 1)
			it.X, it.Y = x, y
			zn.Items = append(zn.Items, it)
		}
		for i := 0; i < 2+r.Int(0, 1); i++ {
			if x, y, ok := zn.FreeSpot(r, nil); ok {
				a := g.Spawn("ashwraith", zn.Level, r)
				a.Name = "shade of " + bd.Name
				a.X, a.Y = x, y
				a.Set("ntgh")
				zn.Actors = append(zn.Actors, a)
			}
		}
		g.Msg(fmt.Sprintf("{{M|%s is a ghost town. Doors hang open on empty rooms, and somewhere a page lies waiting to be read.}}", bd.Name))
		return
	}
	switch gs.Stage {
	case 0:
		g.Msg("{{K|The ghost town keeps its secrets. A diary page lies somewhere in the ruin.}}")
	case 1:
		g.Msg("{{c|The cellar hatch is still sealed. You need the key from the rubble.}}")
	case 2:
		g.Msg("{{R|Something stirs below the old well.}}")
	}
}

func ntHasItem(g *Game, key string, id int) *Item {
	for _, it := range g.P.Inv {
		if it.Extra != nil && it.Extra[key] == id {
			return it
		}
	}
	return nil
}

func (g *Game) ntStory(st *ntState) {
	ids := make([]int, 0, len(st.Ghost))
	for id := range st.Ghost {
		ids = append(ids, id)
	}
	sort.Ints(ids)
	for _, bid := range ids {
		gs := st.Ghost[bid]
		bd := g.W.burgByID[bid]
		if gs.Done || bd == nil {
			continue
		}
		switch gs.Stage {
		case 0:
			if ntHasDiary(g, bid) {
				gs.Stage = 1
				g.ntReveal(st, gs, bd)
			}
		case 1:
			if g.Cur.Key != gs.Zone || maxi(abs(g.P.X-gs.HX), abs(g.P.Y-gs.HY)) > 1 {
				continue
			}
			if ntHasItem(g, "ntgkey", bid) == nil {
				if st.first(fmt.Sprintf("nokey%d", bid)) {
					g.Msg("{{y|The cellar hatch is bolted shut and rusted. It needs a key.}}")
				}
				continue
			}
			gs.Stage = 2
			g.give(g.ntDiary(gs, bd, 2))
			g.give(g.ntDiary(gs, bd, 3))
			g.Msg("{{C|The key turns. The hatch groans open on the cellar, and the last pages of the diary lie on a shelf at the foot of the steps.}}")
			g.Msg(ntReveal[gs.Cause])
			g.ntSpawnCause(gs, bd)
		}
	}
}

func ntHasDiary(g *Game, bid int) bool {
	return ntHasItem(g, "ntg", bid) != nil
}

func (g *Game) ntReveal(st *ntState, gs *ntGhost, bd *BurgData) {
	zn := g.Zones[gs.Zone]
	if zn == nil {
		return
	}
	r := NewRng(fmt.Sprintf("%s:nt:hatch:%d", g.W.Seed, bd.ID))
	if x, y, ok := zn.FreeSpot(r, func(x, y int) bool { return abs(x-40) < 25 }); ok {
		zn.Set(x, y, TInscription)
		zn.Notes[y*MW+x] = fmt.Sprintf("A cellar hatch beside the old well of %s, bolted and rusted. Stand next to it with the key.", bd.Name)
		gs.HX, gs.HY = x, y
	}
	if x, y, ok := zn.FreeSpot(r, nil); ok {
		zn.Items = append(zn.Items, &Item{ID: g.newID(), Kind: "scrap", Name: "rusted cellar key", Glyph: '¬', F: CYellow, Value: 1, X: x, Y: y,
			Extra: map[string]int{"ntgkey": bd.ID}, Lore: "A heavy iron key, flaking with rust. It opens the cellar of " + bd.Name + "."})
	}
	g.Msg(fmt.Sprintf("{{C|The diary speaks of a cellar under the old well of %s and a key lost in the rubble. Find both.}}", bd.Name))
	if st.first("diary") {
		g.deed("ghost", fmt.Sprintf("The wanderer %s pieced together the fall of %s from a dead villager's diary.", g.P.Name, bd.Name), 0, 0)
	}
}

func (g *Game) ntSpawnCause(gs *ntGhost, bd *BurgData) {
	zn := g.Cur
	r := NewRng(fmt.Sprintf("%s:nt:cause:%d", g.W.Seed, bd.ID))
	tp, name := "ghoul", "the Thing Below "+bd.Name
	switch gs.Cause {
	case "plague":
		tp, name = "ashwraith", "the Fever-Shade of "+bd.Name
	case "curse":
		tp, name = "wight", "the Curse of "+bd.Name
	case "lord":
		tp, name = "chief", "the Bandit Lord of "+bd.Name
	}
	lvl := zn.Level + 2
	a := g.Spawn(tp, lvl, r)
	a.Name, a.Proper, a.Boss = name, true, true
	a.MaxHP += a.MaxHP / 2
	a.HP = a.MaxHP
	a.XP += 150
	a.Set("ntcause")
	a.Set("ntg:" + strconv.Itoa(bd.ID))
	if x, y, ok := g.nearestFree(zn, gs.HX, gs.HY); ok {
		a.X, a.Y = x, y
	} else {
		a.X, a.Y = gs.HX, gs.HY
	}
	zn.Actors = append(zn.Actors, a)
	if gs.Cause == "lord" {
		for i := 0; i < 2; i++ {
			b := g.Spawn("bandit", lvl-1, r)
			b.Name = "lord's oathbreaker"
			if x, y, ok := g.nearestFree(zn, a.X, a.Y); ok {
				b.X, b.Y = x, y
				zn.Actors = append(zn.Actors, b)
			}
		}
	}
}

func (g *Game) ntLaidToRest(st *ntState, d *Actor) {
	bid := ntFlagInt(d, "ntg:")
	gs := st.Ghost[bid]
	bd := g.W.burgByID[bid]
	if gs == nil || bd == nil || gs.Done {
		return
	}
	gs.Stage, gs.Done = 3, true
	zn := g.Cur
	ak := zn.Actors[:0]
	gone := 0
	for _, a := range zn.Actors {
		if a.Has("ntgh") && a.HP > 0 {
			gone++
			continue
		}
		ak = append(ak, a)
	}
	zn.Actors = ak
	r := NewRng(fmt.Sprintf("%s:nt:reward:%d", g.W.Seed, bid))
	for i := 0; i < 2; i++ {
		it := g.Loot(r, zn.Level/3+2)
		it.X, it.Y = d.X, d.Y
		zn.Items = append(zn.Items, it)
	}
	zn.Items = append(zn.Items, &Item{ID: g.newID(), Kind: "trophy", Name: "town seal of " + bd.Name, Glyph: '♦', F: CYellow, Value: 140 + zn.Level*10, X: d.X, Y: d.Y,
		Lore: "The iron seal of " + bd.Name + ", recovered from its cellar."})
	g.P.Gold += 100 + zn.Level*30
	g.gainXP(160)
	if zn.In(gs.HX, gs.HY) {
		zn.Notes[gs.HY*MW+gs.HX] = fmt.Sprintf("The cellar of %s lies open and empty. Laid to rest in the year %d by %s.", bd.Name, g.W.Year+g.Day()/YearDays, g.P.Name)
	}
	g.Msg(fmt.Sprintf("{{G|%s is destroyed. The shades of %s sigh and fade; the dead are at rest.}}", d.Name, bd.Name))
	if gone > 0 {
		g.Msg("{{c|The ghosts of the town leave their windows and walk, at last, out of it.}}")
	}
	g.deed("ghost", fmt.Sprintf("The wanderer %s laid the ghost town of %s to rest, ending the work of %s.", g.P.Name, bd.Name, ntCauseNames[gs.Cause]), 0, 0)
}

// ---- talk, side ----

func (g *Game) ntSellable(rare bool) []*Item {
	var out []*Item
	for _, it := range g.P.Inv {
		if it.Kind != "egg" || it.Extra == nil || it.Extra["ntdead"] > 0 {
			continue
		}
		black := it.Extra["ntbrood"] > 0 || ntBlack(ntSpOf(it.Extra["ntsp"]))
		if black && !rare {
			continue
		}
		out = append(out, it)
	}
	return out
}

func (g *Game) ntSell(a *Actor, eggs []*Item, pct int, line string) {
	tot := 0
	for _, it := range eggs {
		tot += it.Value
		g.removeInv(it)
	}
	tot = tot * pct / 100 * (100 + g.Skills["persuasion"]*5) / 100
	g.P.Gold += tot
	g.Msg(fmt.Sprintf("{{W|You sell %d egg%s for %d.}}", len(eggs), map[bool]string{true: "", false: "s"}[len(eggs) == 1], tot))
	if tot >= 300 && g.nt().first("blackmarket") {
		g.deed("nest", fmt.Sprintf("The wanderer %s sold a rare egg for a fortune.", g.P.Name), 0, 0)
	}
	g.talkMenu(line)
}

func ntTalk(g *Game, a *Actor) []Option {
	if a == nil || g.P == nil || g.Cur == nil {
		return nil
	}
	var out []Option
	if g.bsCollector(a) {
		if eggs := g.ntSellable(true); len(eggs) > 0 {
			out = append(out, Option{fmt.Sprintf("{{W|Sell your eggs.}} {{K|(%d, collector's price)}}", len(eggs)), func() {
				g.ntSell(a, eggs, 100, "\"Remarkable. Where did you find them? No, do not tell me.\"")
			}})
		}
	} else if a.Role == "merchant" && !a.Has("ntocc") {
		if eggs := g.ntSellable(false); len(eggs) > 0 {
			out = append(out, Option{fmt.Sprintf("{{W|Sell eggs for the pot.}} {{K|(%d)}}", len(eggs)), func() {
				g.ntSell(a, eggs, 50, "\"Fresh, good. The cook will be glad of them.\"")
			}})
		}
	}
	if a.Has("ntocc") && !g.Hostile(a) {
		occ := ""
		for k := range a.Flags {
			if strings.HasPrefix(k, "nto:") {
				occ = k[4:]
			}
		}
		if d, ok := ntOccs[occ]; ok && d.Chat != "" {
			out = append(out, Option{"{{c|Ask about the ruin.}}", func() { g.talkMenu(d.Chat) }})
		}
		switch occ {
		case "monks":
			out = append(out, Option{"{{G|Ask for a poultice.}}", func() {
				k := fmt.Sprintf("poultice%d", g.Day())
				if !g.nt().first(k) {
					g.talkMenu("\"You have had what I can spare today.\"")
					return
				}
				g.P.HP = mini(g.P.MaxHP, g.P.HP+g.P.MaxHP/2)
				g.Msg("{{G|The hermit binds your wounds with bitter herbs.}}")
				g.talkMenu("\"Go gently.\"")
			}})
		case "plague":
			for _, it := range g.P.Inv {
				if it.Kind == "tonic" {
					tn := it
					out = append(out, Option{"{{G|Give them a tonic.}}", func() {
						if tn.Count > 1 {
							tn.Count--
						} else {
							g.removeInv(tn)
						}
						g.gainXP(30)
						g.give(g.makeIngredient("refugee's lucky herbs", "mending", 2, CGreen))
						g.Msg("{{G|They weep, drink, and press a bundle of herbs into your hand.}}")
						if g.nt().first("plaguehelp") {
							g.deed("nest", fmt.Sprintf("The wanderer %s shared a tonic with plague refugees sheltering in a ruin.", g.P.Name), 0, 0)
						}
						g.talkMenu("\"May you never know this fever.\"")
					}})
					break
				}
			}
		}
	}
	return out
}

func ntSide(g *Game) []string {
	if g.P == nil || g.W == nil {
		return nil
	}
	var out []string
	for _, it := range g.P.Inv {
		if it.Kind != "egg" || it.Extra == nil || it.Extra["ntsp"] == 0 || it.Extra["ntbrood"] > 0 {
			continue
		}
		if it.Extra["ntdead"] > 0 {
			out = append(out, "{{R|Egg: cold}}")
			continue
		}
		s := ntSpOf(it.Extra["ntsp"])
		if s == nil {
			continue
		}
		pct := clampi(it.Extra["ntwarm"]*100/(ntNeedDays(s)*DayLen), 0, 100)
		out = append(out, fmt.Sprintf("{{y|Egg: %d%% hatched}}", pct))
		if len(out) >= 2 {
			break
		}
	}
	return out
}
