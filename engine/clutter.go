package engine

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// Clutter (prefix cl): small pickable everyday things (cutlery, trinkets, toys, tools, relics...) scattered by
// place, wild finds (fruit, nuts, mushrooms, eggs, shells, bones...) that spoil, collections of six that pay 4x
// at curio dealers, a Collections screen on F9, and lost-thing errands (mirrored into g.Quests as Kind "clutter").
// Zone hook uses a side rng only (seed:cl:zoneKey) and only adds items. State: g.ext("cl") planet-local (errands)
// and g.ext("clg") global (collections, firsts). Items in town buildings carry Extra["uw:own"] so crime.go treats
// taking them as theft. Optional hooks for shared files are listed in the report.

func init() {
	unLocalKeys = append(unLocalKeys, "cl")
	zoneHooks = append(zoneHooks, clZone)
	turnHooks = append(turnHooks, clTurn)
	talkHooks = append(talkHooks, clTalk)
	killHooks = append(killHooks, clKill)
	mainKeys["F9"] = func(g *Game) { g.Mode, g.Sel = "clutter", 0 }
	modeKeys["clutter"] = clKeys
	modeDraws["clutter"] = clDraw
	clParse()
}

// ---------------------------------------------------------------- catalogue

type clDef struct {
	Idx, Set int
	Name     string
	Flav     string
	Gl       rune
	Col      byte
	Wt, Val  int
	Skin     map[byte]string
}

type clSet struct {
	ID, Title string
	Where     []string
	Adj       map[byte]string
	Items     []int
}

type clNat struct {
	Name, Tag, Flav, Kind string
	Gl                    rune
	Col                   byte
	Wt, Val, Life         int
	Where                 []string
	Biomes                []int
	Seasons               string
}

var (
	clDefs  []*clDef
	clSets  []*clSet
	clNats  []*clNat
	clByTag = map[string][]int{}
	clByNam = map[string]*clDef{}
)

var clColors = map[string]byte{"g": CGray, "w": CWhite, "y": CYellow, "b": CBrown, "c": CCyan, "m": CMagenta, "r": CRed,
	"G": CGreen, "B": CBlue, "o": COrange, "k": CDarkGray, "C": CDarkCyan}

var clGenreLetter = map[string]byte{"fantasy": 'f', "mythic": 'm', "steampunk": 's', "dieselpunk": 'd', "atompunk": 'a',
	"cyberpunk": 'c', "biopunk": 'b', "solarpunk": 'o', "wasteland": 'w', "twilight": 't'}

func clGenreKey(genre string) byte {
	if k, ok := clGenreLetter[genre]; ok {
		return k
	}
	return 'f'
}

func clKV(s, sep string) map[byte]string {
	m := map[byte]string{}
	for _, p := range strings.Split(s, sep) {
		if i := strings.Index(p, "="); i == 1 {
			m[p[0]] = p[2:]
		}
	}
	return m
}

func clParse() {
	var cur *clSet
	for _, ln := range strings.Split(clSetData, "\n") {
		ln = strings.TrimSpace(ln)
		if ln == "" {
			continue
		}
		p := strings.Split(ln, "|")
		if strings.HasPrefix(ln, "@") {
			cur = &clSet{ID: p[0][1:], Title: p[1], Where: strings.Split(p[2], ","), Adj: clKV(p[3], ",")}
			clSets = append(clSets, cur)
			for _, w := range cur.Where {
				clByTag[w] = append(clByTag[w], len(clSets)-1)
			}
			continue
		}
		if cur == nil || len(p) < 6 {
			continue
		}
		wt, _ := strconv.Atoi(p[3])
		val, _ := strconv.Atoi(p[4])
		d := &clDef{Idx: len(clDefs), Set: len(clSets) - 1, Name: p[0], Gl: []rune(p[1])[0], Col: clColors[p[2]], Wt: wt, Val: val, Flav: p[5]}
		if len(p) > 6 {
			d.Skin = clKV(p[6], ";")
		}
		clDefs = append(clDefs, d)
		cur.Items = append(cur.Items, d.Idx)
		clByNam[d.Name] = d
	}
	for _, ln := range strings.Split(clNatData, "\n") {
		p := strings.Split(strings.TrimSpace(ln), "|")
		if len(p) < 12 {
			continue
		}
		wt, _ := strconv.Atoi(p[3])
		val, _ := strconv.Atoi(p[4])
		life, _ := strconv.Atoi(p[6])
		n := &clNat{Name: p[0], Gl: []rune(p[1])[0], Col: clColors[p[2]], Wt: wt, Val: val, Tag: p[5], Life: life, Kind: p[7],
			Where: strings.Split(p[8], ","), Flav: p[11]}
		if p[9] != "-" && p[9] != "" {
			for _, b := range strings.Split(p[9], ",") {
				v, _ := strconv.Atoi(b)
				n.Biomes = append(n.Biomes, v)
			}
		}
		if p[10] != "-" {
			n.Seasons = p[10]
		}
		clNats = append(clNats, n)
	}
}

func clHas(l []string, s string) bool {
	for _, v := range l {
		if v == s {
			return true
		}
	}
	return false
}

func clName(genre string, d *clDef) string {
	k := clGenreKey(genre)
	if s, ok := d.Skin[k]; ok {
		return s
	}
	if a := clSets[d.Set].Adj[k]; a != "" {
		return a + " " + d.Name
	}
	return d.Name
}

func (g *Game) clMake(d *clDef) *Item {
	return &Item{ID: g.newID(), Kind: "clutter", Name: clName(g.Genre, d), Glyph: d.Gl, F: d.Col, Value: d.Val, Lore: d.Flav,
		Extra: map[string]int{"cl:i": d.Idx + 1, "cl:w": d.Wt}}
}

func (g *Game) clMakeNat(n *clNat) *Item {
	if d := clByNam[n.Name]; d != nil && n.Kind == "c" {
		return g.clMake(d)
	}
	idx := 0
	for i, o := range clNats {
		if o == n {
			idx = i + 1
		}
	}
	it := &Item{ID: g.newID(), Kind: "clutter", Name: n.Name, Glyph: n.Gl, F: n.Col, Value: n.Val, Lore: n.Flav,
		Extra: map[string]int{"cl:n": idx, "cl:w": n.Wt}}
	if n.Kind == "i" {
		it.Kind, it.Effect, it.Count, it.Power, it.Glyph = "ingredient", n.Tag, 1, 120, '%'
		it.Glyph = n.Gl
		if n.Life > 0 {
			it.Extra["cl:life"] = n.Life
		}
	}
	return it
}

// clRot turns a fruit rotten: no sale value, still carries its flavour tag for the cauldron.
func clRot(it *Item) {
	it.Name = "rotten " + it.Name
	it.Value = 0
	it.F = CBrown
	it.Lore = "Gone soft and sour. No good for the table; the cauldron may still want it."
	delete(it.Extra, "cl:life")
	it.Extra["cl:rot"] = 1
}

// clWeight: weight of a clutter item from its hundredths (hook for itemWeight).
func clWeight(it *Item) int {
	w := it.Extra["cl:w"]
	if w <= 0 {
		return 1
	}
	return (maxi(it.Count, 1)*w + 70) / 100
}

// ---------------------------------------------------------------- state

type clLost struct {
	QID    int    `json:"q"`
	Giver  string `json:"g"`
	Def    int    `json:"d"`
	Reward int    `json:"r"`
}

type clLocal struct {
	Lost  []clLost       `json:"lost,omitempty"`
	Asked map[string]int `json:"asked,omitempty"`
}

type clGlobal struct {
	Done   map[string]int  `json:"done,omitempty"`
	Found  map[string]bool `json:"found,omitempty"`
	Firsts map[string]bool `json:"firsts,omitempty"`
	Sold   int             `json:"sold,omitempty"`
}

func clL(g *Game) *clLocal {
	l := g.ext("cl", func() any { return &clLocal{} }).(*clLocal)
	if l.Asked == nil {
		l.Asked = map[string]int{}
	}
	return l
}

func clG(g *Game) *clGlobal {
	s := g.ext("clg", func() any { return &clGlobal{} }).(*clGlobal)
	if s.Done == nil {
		s.Done, s.Found, s.Firsts = map[string]int{}, map[string]bool{}, map[string]bool{}
	}
	return s
}

func clDoneCount(g *Game) int { return len(clG(g).Done) }

// clCarry: carry capacity from completed collections (hook for capacity()).
func clCarry(g *Game) int { return clDoneCount(g) }

// clPriceMod: sale multiplier (hook for price() in the sell branch). Curio dealers and fences pay more for
// collectibles; every completed collection adds 1% to all your sales.
func clPriceMod(g *Game, a *Actor, it *Item) float64 {
	f := 1 + 0.01*float64(clDoneCount(g))
	if it != nil && (it.Extra["cl:i"] > 0 || it.Extra["cl:n"] > 0) && a != nil {
		switch {
		case a.Role == "exotic":
			f *= 1.5
		case uwIsFence(a):
			f *= 1.3
		}
	}
	return f
}

func (g *Game) clFirst(key, text string) {
	s := clG(g)
	if s.Firsts[key] {
		return
	}
	s.Firsts[key] = true
	g.deed("event", text, 0, 0)
}

// clHeld: carried count per catalogue definition.
func clHeld(g *Game) map[int]int {
	m := map[int]int{}
	for _, it := range g.P.Inv {
		if i := it.Extra["cl:i"]; i > 0 && it.Extra["cl:q"] == 0 {
			m[i-1] += maxi(it.Count, 1)
		}
	}
	return m
}

func clSetHeld(held map[int]int, s *clSet) int {
	n := 0
	for _, di := range s.Items {
		if held[di] > 0 {
			n++
		}
	}
	return n
}

// ---------------------------------------------------------------- per-turn

func clTurn(g *Game) {
	if g.P == nil || g.Cur == nil {
		return
	}
	st := clG(g)
	for _, it := range g.P.Inv {
		if it.Extra == nil {
			continue
		}
		key := ""
		if i := it.Extra["cl:i"]; i > 0 {
			key = "i" + strconv.Itoa(i)
		} else if n := it.Extra["cl:n"]; n > 0 {
			key = "n" + strconv.Itoa(n)
		}
		if key != "" && !st.Found[key] {
			st.Found[key] = true
			if it.Lore != "" {
				g.Msg("{{K|" + it.Lore + "}}")
			}
			if it.Extra["cl:i"] > 0 {
				g.clFirst("first", fmt.Sprintf("The wanderer %s began keeping small things: %s.", g.P.Name, it.Name))
			} else if it.Kind == "ingredient" {
				g.clFirst("forage", fmt.Sprintf("The wanderer %s foraged %s from the wild.", g.P.Name, it.Name))
			} else {
				g.clFirst("nature", fmt.Sprintf("The wanderer %s picked up %s lying about.", g.P.Name, it.Name))
			}
		}
		if it.Extra["cl:life"] > 0 {
			if it.Extra["cl:d"] == 0 {
				it.Extra["cl:d"] = g.Day() + 1
			}
		}
	}
	for _, q := range g.Quests {
		if q.Kind == "clutter" && !q.Closed {
			carried := false
			for _, it := range g.P.Inv {
				if it.Extra["cl:q"] == q.ID {
					carried = true
				}
			}
			if carried && !q.Done {
				g.Msg("{{G|You found it. Take it back to " + q.Giver + ".}}")
			}
			q.Done = carried
		}
	}
	if g.Turn%60 != 0 {
		return
	}
	for _, it := range g.P.Inv {
		if it.Extra != nil && it.Extra["cl:life"] > 0 && it.Extra["cl:d"] > 0 && g.Day()-(it.Extra["cl:d"]-1) >= it.Extra["cl:life"] {
			g.Msg("{{y|Your " + it.Name + " has gone bad.}}")
			clRot(it)
		}
	}
	held := clHeld(g)
	for _, s := range clSets {
		if st.Done[s.ID] == 0 && clSetHeld(held, s) == len(s.Items) {
			st.Done[s.ID] = g.Day() + 1
			g.Msg(fmt.Sprintf("{{Y|Collection complete: %s! +1 carrying capacity, +1%% on every sale.}}", s.Title))
			g.deed("event", fmt.Sprintf("The wanderer %s completed a collection of %s.", g.P.Name, strings.ToLower(s.Title)), 0, 0)
			g.gainXP(40)
		}
	}
}

// clKill: rarely something small is found on a dead person.
func clKill(g *Game, d, src *Actor) {
	if g.Cur == nil || d == nil || d == g.P || !(d.Role != "" || d.Species != "" || d.Proper) {
		return
	}
	h := HashStr(fmt.Sprintf("%s:cl:kill:%d:%d", g.W.Seed, d.ID, g.Turn))
	if h%100 >= 14 {
		return
	}
	r := NewRng(fmt.Sprintf("%s:cl:kill2:%d:%d", g.W.Seed, d.ID, g.Turn))
	tags := []string{"home", "travel", "games", "jewel"}
	if di := clPick(r, Pick(r, tags), 1); di >= 0 {
		it := g.clMake(clDefs[di])
		it.X, it.Y = d.X, d.Y
		g.Cur.Items = append(g.Cur.Items, it)
	}
}

// ---------------------------------------------------------------- placement

var clFloorOK = map[Terrain]bool{TFloor: true, TCarpet: true, TGrass: true, TTallGrass: true, TSand: true, TDirt: true, TSnow: true,
	TAsh: true, TMoss: true, TFungus: true, TCrystalFloor: true, TCobble: true, TRubble: true, TBones: true}

var clIndoor = map[Terrain]bool{TFloor: true, TCarpet: true, TBed: true, TShelf: true, TStall: true, TBrazier: true, TAnvil: true,
	TCrate: true, TSeat: true, TStatue: true, TMachine: true, TAltar: true, TThrone: true, TPillar: true}

var clFurniture = map[Terrain]bool{TBed: true, TShelf: true, TStall: true, TBrazier: true, TAnvil: true, TCrate: true, TSeat: true,
	TWall: true, TAltar: true, TMachine: true}

// clPick: a random catalogue definition index for a place tag; richer places lean to the pricier ones.
func clPick(r *Rng, tag string, rich int) int {
	sets := clByTag[tag]
	if len(sets) == 0 {
		return -1
	}
	best := -1
	for t := 0; t <= rich; t++ {
		s := clSets[Pick(r, sets)]
		di := Pick(r, s.Items)
		if best < 0 || clDefs[di].Val > clDefs[best].Val {
			best = di
		}
	}
	return best
}

type clPlacer struct {
	g      *Game
	zn     *Zone
	taken  map[int]bool
	stairs map[int]bool
}

func clNewPlacer(g *Game, zn *Zone) *clPlacer {
	p := &clPlacer{g: g, zn: zn, taken: map[int]bool{}, stairs: map[int]bool{}}
	for _, it := range zn.Items {
		p.taken[it.Y*MW+it.X] = true
	}
	for y := 0; y < MH; y++ {
		for x := 0; x < MW; x++ {
			switch zn.At(x, y) {
			case TDown, TUp, TPortal, TDoor, TDoorOpen:
				for dy := -1; dy <= 1; dy++ {
					for dx := -1; dx <= 1; dx++ {
						p.stairs[(y+dy)*MW+x+dx] = true
					}
				}
			}
		}
	}
	return p
}

func (p *clPlacer) ok(x, y int) bool {
	zn := p.zn
	if !zn.In(x, y) || !clFloorOK[zn.At(x, y)] || zn.At(x, y).D().Water > 0 || p.taken[y*MW+x] || p.stairs[y*MW+x] {
		return false
	}
	return zn.ActorAt(x, y) == nil
}

func (p *clPlacer) near(x, y int, f func(t Terrain) bool) bool {
	for _, d := range dirs8 {
		if f(p.zn.At(x+d[0], y+d[1])) {
			return true
		}
	}
	return false
}

func (p *clPlacer) put(it *Item, x, y int, owned bool) {
	it.X, it.Y = x, y
	if owned {
		it.Extra["uw:own"] = 1
	}
	p.taken[y*MW+x] = true
	p.zn.Items = append(p.zn.Items, it)
}

type clRoom struct {
	tiles []int
	role  string
}

func clRooms(zn *Zone) []clRoom {
	seen := make([]bool, MW*MH)
	var out []clRoom
	for i := 0; i < MW*MH; i++ {
		if seen[i] || !clIndoor[zn.T[i]] {
			continue
		}
		var comp []int
		stack := []int{i}
		seen[i] = true
		for len(stack) > 0 {
			c := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			comp = append(comp, c)
			x, y := c%MW, c/MW
			for _, d := range [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
				nx, ny := x+d[0], y+d[1]
				if nx < 0 || ny < 0 || nx >= MW || ny >= MH {
					continue
				}
				n := ny*MW + nx
				if !seen[n] && clIndoor[zn.T[n]] {
					seen[n] = true
					stack = append(stack, n)
				}
			}
		}
		cnt := map[Terrain]int{}
		for _, c := range comp {
			cnt[zn.T[c]]++
		}
		role := "home"
		switch {
		case len(comp) > 160:
			role = "shop"
		case cnt[TAltar] > 0:
			role = "temple"
		case cnt[TShelf] >= 2:
			role = "study"
		case cnt[TStall] > 0:
			role = "shop"
		case cnt[TBrazier] > 0 || cnt[TAnvil] > 0:
			role = "forge"
		}
		sort.Ints(comp)
		out = append(out, clRoom{comp, role})
	}
	return out
}

func clZone(g *Game, zn *Zone, _ *Rng) {
	if zn == nil || zn.Sky || zn.Underwater || zn.Kind == "sea" || zn.Kind == "lair" || zn.Kind == "wonder" || g.W == nil {
		return
	}
	r := NewRng(g.W.Seed + ":cl:" + zn.Key)
	p := clNewPlacer(g, zn)
	if zn.Kind == "town" {
		rich := 0
		if bd := g.W.burgByID[zn.Burg]; bd != nil {
			rich = clampi(int(bd.Pop)/4, 0, 2)
		}
		for _, rm := range clRooms(zn) {
			if len(rm.tiles) < 6 {
				continue
			}
			tag := rm.role
			n := 1 + r.Int(0, 1+rich)
			if rm.role == "home" {
				n++
			}
			if rm.role == "shop" {
				n = mini(n, 3)
			}
			for k := 0; k < n; k++ {
				t := tag
				if tag == "home" {
					t = Pick(r, []string{"home", "home", "kitchen", "tavern"})
				}
				di := clPick(r, t, rich)
				if di < 0 {
					continue
				}
				for try := 0; try < 24; try++ {
					c := Pick(r, rm.tiles)
					x, y := c%MW, c/MW
					if !p.ok(x, y) || (r.Chance(0.65) && !p.near(x, y, func(t Terrain) bool { return clFurniture[t] })) {
						continue
					}
					p.put(g.clMake(clDefs[di]), x, y, true)
					break
				}
			}
			if rm.role == "home" && r.Chance(0.3) {
				g.clPlaceNat(p, r, rm.tiles, "home")
			}
		}
		return
	}
	name := strings.ToLower(zn.Name)
	var tags []string
	count := 0
	switch zn.Kind {
	case "ruin":
		tags, count = []string{"ruin", "ruin", "dungeon", "home"}, 3+r.Int(0, 4)
	case "crypt":
		tags, count = []string{"crypt", "crypt", "temple", "relic"}, 3+r.Int(0, 3)
	case "dungeon", "tower", "underdark":
		tags, count = []string{"dungeon", "dungeon", "crypt"}, 3+r.Int(0, 4)
		if strings.Contains(name, "mine") {
			tags = []string{"mine", "mine", "dungeon"}
		} else if strings.Contains(name, "catacomb") || strings.Contains(name, "barrow") {
			tags = []string{"crypt", "crypt", "dungeon"}
		}
	case "outskirts":
		tags, count = []string{"outdoors", "home"}, 2+r.Int(0, 3)
	case "surface":
		tags, count = []string{"outdoors"}, 1+r.Int(0, 3)
	case "camp":
		tags, count = []string{"outdoors", "travel", "games"}, 2+r.Int(0, 2)
	default:
		return
	}
	for k := 0; k < count; k++ {
		t := Pick(r, tags)
		if t == "outdoors" && r.Chance(0.2) {
			t = "coast"
		}
		di := clPick(r, t, 0)
		if di < 0 {
			continue
		}
		for try := 0; try < 40; try++ {
			x, y := r.Int(1, MW-2), r.Int(1, MH-2)
			if !p.ok(x, y) || (t == "coast" && !p.near(x, y, func(t Terrain) bool { return t.D().Water > 0 })) {
				continue
			}
			p.put(g.clMake(clDefs[di]), x, y, false)
			break
		}
	}
	g.clPlaceNat(p, r, nil, "wild")
}

// clPlaceNat scatters wild finds. mode "home" puts pantry goods on the given room tiles; "wild" works the zone.
func (g *Game) clPlaceNat(p *clPlacer, r *Rng, tiles []int, mode string) {
	zn := p.zn
	if mode == "home" {
		var c []*clNat
		for _, n := range clNats {
			if clHas(n.Where, "home") {
				c = append(c, n)
			}
		}
		for try := 0; try < 20 && len(c) > 0; try++ {
			t := Pick(r, tiles)
			if p.ok(t%MW, t/MW) {
				p.put(g.clMakeNat(Pick(r, c)), t%MW, t/MW, true)
				return
			}
		}
		return
	}
	under := zn.Z != 0 || zn.Kind == "dungeon" || zn.Kind == "tower" || zn.Kind == "underdark" || zn.Kind == "crypt"
	if under && zn.Kind != "dungeon" && zn.Kind != "underdark" {
		return
	}
	if !under && zn.Kind != "surface" && zn.Kind != "outskirts" && zn.Kind != "camp" {
		return
	}
	season := "puaw"[g.Season()%4]
	wet := false
	if !under {
		if w := g.W.WeatherAt(g.W.Idx(zn.WX, zn.WY), g.Turn); w.Kind != nil && w.Kind.Wet >= 2 {
			wet = true
		}
	}
	n := 2 + r.Int(0, 4)
	if under {
		n = 1 + r.Int(0, 2)
	} else if wet {
		n += 2
	}
	for k := 0; k < n; k++ {
		var cand []*clNat
		for _, nt := range clNats {
			if clHas(nt.Where, "home") || (under != clHas(nt.Where, "cave")) {
				continue
			}
			if len(nt.Biomes) > 0 && !under {
				okb := false
				for _, b := range nt.Biomes {
					if b == zn.Biome {
						okb = true
					}
				}
				if !okb {
					continue
				}
			}
			if nt.Seasons != "" && !strings.ContainsRune(nt.Seasons, rune(season)) {
				continue
			}
			if clHas(nt.Where, "rain") && !wet && !r.Chance(0.35) {
				continue
			}
			cand = append(cand, nt)
		}
		if len(cand) == 0 {
			return
		}
		nt := Pick(r, cand)
		for try := 0; try < 60; try++ {
			x, y := r.Int(1, MW-2), r.Int(1, MH-2)
			if !p.ok(x, y) {
				continue
			}
			tree := func(t Terrain) bool { return t == TTree || t == TPalm }
			good := true
			for _, w := range nt.Where {
				switch w {
				case "tree", "forest", "hive":
					good = good && p.near(x, y, tree)
				case "bush":
					good = good && p.near(x, y, func(t Terrain) bool { return t == TBush })
				case "coast", "water":
					good = good && p.near(x, y, func(t Terrain) bool { return t.D().Water > 0 })
				}
			}
			if !good {
				continue
			}
			p.put(g.clMakeNat(nt), x, y, false)
			break
		}
	}
}

// ---------------------------------------------------------------- talk: dealers and errands

func clSetPay(g *Game, s *clSet) int {
	t := 0
	for _, di := range s.Items {
		t += clDefs[di].Val
	}
	return t * 4
}

func clTalk(g *Game, a *Actor) []Option {
	var out []Option
	if a.Role == "exotic" || uwIsFence(a) {
		held := clHeld(g)
		for _, s := range clSets {
			s := s
			if clSetHeld(held, s) < len(s.Items) {
				continue
			}
			pay := clSetPay(g, s)
			clean := true
			for _, it := range g.P.Inv {
				if it.Extra["cl:i"] > 0 && uwIsStolen(it) && !uwIsFence(a) {
					clean = false
				}
			}
			if !clean {
				continue
			}
			if a.Gold < pay {
				out = append(out, Option{fmt.Sprintf("{{K|Your complete set of %s would fetch %d, more than they hold.}}", strings.ToLower(s.Title), pay), func() { g.talkMenu(g.greeting(a)) }})
				continue
			}
			out = append(out, Option{fmt.Sprintf("{{Y|Sell the complete set of %s (%d).}}", strings.ToLower(s.Title), pay), func() {
				for _, di := range s.Items {
					for _, it := range append([]*Item{}, g.P.Inv...) {
						if it.Extra["cl:i"] == di+1 && it.Extra["cl:q"] == 0 {
							one := it
							if it.Stack() && it.Count > 1 {
								it.Count--
								cp := *it
								cp.Count = 1
								one = &cp
							} else {
								g.removeInv(it)
							}
							a.Inv = append(a.Inv, one)
							break
						}
					}
				}
				a.Gold -= pay
				g.cuEarn(pay)
				clG(g).Sold++
				g.Msg(fmt.Sprintf("{{G|%s pays %d for the complete set of %s.}}", a.Name, pay, strings.ToLower(s.Title)))
				g.clFirst("setsale", fmt.Sprintf("The wanderer %s sold a complete collection of %s to %s.", g.P.Name, strings.ToLower(s.Title), a.Name))
				g.talkMenu(g.greeting(a))
			}})
		}
	}
	l := clL(g)
	for _, ls := range l.Lost {
		ls := ls
		for _, q := range g.Quests {
			if q.ID == ls.QID && q.Done && !q.Closed && q.Giver == a.Name {
				q := q
				out = append(out, Option{"{{G|I found what you lost.}}", func() {
					for _, it := range g.P.Inv {
						if it.Extra["cl:q"] == q.ID {
							g.removeInv(it)
							break
						}
					}
					q.Done, q.Closed = true, true
					g.cuEarn(ls.Reward)
					g.gainXP(10 + ls.Reward/2)
					g.AddRep(a.Faction, 15, " for returning a lost thing")
					g.deed("quest", fmt.Sprintf("The wanderer %s returned %s's lost %s.", g.P.Name, a.Name, strings.ToLower(clName(g.Genre, clDefs[ls.Def]))), 0, 0)
					g.talkMenu(fmt.Sprintf("%s clutches it, eyes bright. \"I never thought I would see it again. Take %d.\"", a.Name, ls.Reward))
				}})
			}
		}
	}
	if (a.Role == "villager" || a.Role == "elder" || a.Role == "merchant" || a.Role == "notable") && g.Cur.Kind == "town" && l.Asked[a.Name] == 0 && !clHasOpen(g) &&
		HashStr(fmt.Sprintf("%s:cl:ask:%s", g.W.Seed, a.Name))%100 < 30 {
		out = append(out, Option{"\"You look like you have lost something.\"", func() { clAsk(g, a) }})
	}
	return out
}

func clHasOpen(g *Game) bool {
	for _, q := range g.Quests {
		if q.Kind == "clutter" && !q.Closed {
			return true
		}
	}
	return false
}

func clAsk(g *Game, a *Actor) {
	l := clL(g)
	l.Asked[a.Name] = g.Day() + 1
	r := NewRng(fmt.Sprintf("%s:cl:lost:%s:%d", g.W.Seed, a.Name, g.Day()))
	tag := Pick(r, []string{"toys", "jewel", "stationery", "trinket", "music", "scholar", "games"})
	var set *clSet
	for _, s := range clSets {
		if s.ID == tag {
			set = s
		}
	}
	if set == nil {
		set = clSets[0]
	}
	d := clDefs[Pick(r, set.Items)]
	x, y, ok := g.Cur.FreeSpot(r, nil)
	if !ok {
		g.talkMenu("\"No, nothing. Forgive me.\"")
		return
	}
	q := &Quest{ID: g.newID(), Kind: "clutter", Giver: a.Name, GiverID: -1, Faction: a.Faction, TP: g.W.Idx(g.Cur.WX, g.Cur.WY), Name: clName(g.Genre, d)}
	q.Reward = 15 + d.Val*3
	q.Title = fmt.Sprintf("Find %s's %s", a.Name, q.Name)
	it := g.clMake(d)
	it.Quest = q.ID
	it.Extra["cl:q"] = q.ID
	it.Lore = "Lost by " + a.Name + ". " + d.Flav
	it.X, it.Y = x, y
	g.Cur.Items = append(g.Cur.Items, it)
	q.Text = fmt.Sprintf("\"Oh, you noticed. I have lost my %s somewhere in %s. It is nothing to others, everything to me. I would pay for its return.\"", strings.ToLower(q.Name), g.Cur.Name)
	l.Lost = append(l.Lost, clLost{QID: q.ID, Giver: a.Name, Def: d.Idx, Reward: q.Reward})
	g.Quests = append(g.Quests, q)
	g.Msg("{{W|New quest: " + q.Title + "}}")
	g.clFirst("lostask", fmt.Sprintf("The wanderer %s promised to find a lost %s for %s.", g.P.Name, strings.ToLower(q.Name), a.Name))
	g.talkMenu(q.Text + "\n\n{{K|(Quest added to your journal.)}}")
}

// ---------------------------------------------------------------- screen (F9)

func clKeys(g *Game, k string) {
	switch k {
	case "Escape", "F9", "q":
		g.Mode = "main"
	case "ArrowUp":
		g.Sel = maxi(0, g.Sel-1)
	case "ArrowDown":
		g.Sel = mini(len(clSets)-1, g.Sel+1)
	}
}

func clDraw(g *Game) {
	s := g.sc
	st := clG(g)
	x, y, w, h := g.panel("Collections")
	bottom := y + h - 1
	done := clDoneCount(g)
	s.TextB(x, y, fmt.Sprintf("{{W|%d of %d collections complete}}   {{K|+%d carrying capacity, +%d%% on sales. Curio dealers and fences pay 4x for a full set of six.}}", done, len(clSets), done, done), CGray, CBgDark, x+w)
	y += 2
	held := clHeld(g)
	g.Sel = clampi(g.Sel, 0, len(clSets)-1)
	listW := 34
	for i, set := range clSets {
		if y+i >= bottom-3 {
			break
		}
		bg := CBgDark
		if i == g.Sel {
			bg = CBgSel
		}
		s.Fill(x, y+i, listW, 1, bg)
		c := "y"
		mark := " "
		if st.Done[set.ID] > 0 {
			c, mark = "G", "*"
		}
		s.TextB(x, y+i, fmt.Sprintf("{{%s|%s %-20s %d/6}}", c, mark, set.Title, clSetHeld(held, set)), CGray, bg, x+listW)
	}
	set := clSets[g.Sel]
	dx := x + listW + 2
	dy := y
	s.TextB(dx, dy, "{{W|"+set.Title+"}}", CGray, CBgDark, x+w)
	dy += 2
	for _, di := range set.Items {
		d := clDefs[di]
		line := ""
		switch {
		case held[di] > 0:
			line = fmt.Sprintf("{{G|[x]}} %s {{K|(%d)}}  {{y|worth %d}}", clName(g.Genre, d), held[di], d.Val)
		case st.Found[fmt.Sprintf("i%d", di+1)]:
			line = fmt.Sprintf("{{y|[+]}} %s {{K|(found before)}}", clName(g.Genre, d))
		default:
			line = "{{K|[ ] ???}}"
		}
		s.TextB(dx, dy, line, CGray, CBgDark, x+w)
		dy++
		if held[di] > 0 || st.Found[fmt.Sprintf("i%d", di+1)] {
			for _, l := range Wrap("{{K|"+d.Flav+"}}", x+w-dx-4) {
				s.TextB(dx+4, dy, l, CGray, CBgDark, x+w)
				dy++
			}
		}
	}
	dy++
	s.TextB(dx, dy, fmt.Sprintf("{{y|Set price at a curio dealer: %d}}", clSetPay(g, set)), CGray, CBgDark, x+w)
	for _, q := range g.Quests {
		if q.Kind == "clutter" && !q.Closed && bottom-2 > y {
			s.TextB(x, bottom-2, "{{W|Lost thing:}} "+q.Title, CGray, CBgDark, x+w)
		}
	}
	s.TextB(x, bottom, "{{K|Up/Down choose, Esc or F9 closes. Take small things with g or comma; items in homes are other people's property.}}", CGray, CBgDark, x+w)
}
