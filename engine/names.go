package engine

// names.go (nm): Realmforge's exported names for rivers, routes, geographic features and towns
// (WorldData.Names, src/lib/adventure/names.ts there), shown in play: a sidebar line for the
// feature / river / road you are in or beside, world-map info, bridges named after the river they
// cross, milestones where a road leaves a parasang (road + next town), named harbour gates and air
// docks, the town's size tier on entry, rift gate names, inn signs, street signposts in city
// districts, arena names, wilderness towers named after the feature they stand in, and gossip about
// the land. Old worlds without `names` behave as before. Never draws g.R or the zone rng: every pick
// hashes the world seed and ids.

import (
	"fmt"
	"sort"
	"strings"
	"sync"
)

// NamesData is the `names` block of a world file.
type NamesData struct {
	Rivers   []NmRiver      `json:"rivers"`
	Routes   []NmRoute      `json:"routes"`
	Features []NmFeature    `json:"features"`
	Burgs    map[int]NmBurg `json:"burgs"`

	once    sync.Once
	riverAt map[int][]int    // parasang → river indices
	routeAt map[int][]int    // parasang → route indices
	featAt  map[int][]int    // parasang → feature indices
	pos     []map[int]int    // route index → parasang → position along it
	byID    map[int]*NmRoute // route id → route
}

type NmRiver struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	Parasangs []int  `json:"parasangs"`
}

type NmRoute struct {
	ID        int    `json:"id"`
	Kind      string `json:"kind"` // road sea tunnel sky
	Name      string `json:"name"`
	A         int    `json:"a"` // burg ids at either end
	B         int    `json:"b"`
	Parasangs []int  `json:"parasangs"`
}

type NmFeature struct {
	Kind      string `json:"kind"` // range peak pass hills lake bay gulf sea ocean forest jungle taiga fungal desert barrens plain tundra ice ash crystal lava marsh
	Name      string `json:"name"`
	Parasangs []int  `json:"parasangs"`
	Peak      *int   `json:"peak,omitempty"` // ranges: the highest parasang
}

type NmBurg struct {
	Tier    string `json:"tier"` // hamlet village town city metropolis (genre words)
	Harbor  string `json:"harbor,omitempty"`
	Airdock string `json:"airdock,omitempty"`
}

func init() {
	zoneHooks = append(zoneHooks, nmZone)
	enterHooks = append(enterHooks, nmEnter)
	sideHooks = append(sideHooks, nmSide)
}

// nmKindOrder: the most specific feature first (sidebar, world map, gossip)
var nmKindOrder = map[string]int{"pass": 0, "peak": 1, "lake": 2, "bay": 3, "gulf": 4, "range": 5, "hills": 6, "marsh": 7, "forest": 8, "jungle": 9, "taiga": 10, "fungal": 11,
	"desert": 12, "barrens": 13, "crystal": 14, "ash": 15, "lava": 16, "ice": 17, "tundra": 18, "plain": 19, "sea": 20, "ocean": 21}

// nmData returns the world's names with their lookups built, or nil for an old world.
func (w *WorldData) nmData() *NamesData {
	n := w.Names
	if n == nil {
		return nil
	}
	n.once.Do(func() {
		n.riverAt, n.routeAt, n.featAt = map[int][]int{}, map[int][]int{}, map[int][]int{}
		n.byID = map[int]*NmRoute{}
		for i, r := range n.Rivers {
			for _, p := range r.Parasangs {
				n.riverAt[p] = nmAddOnce(n.riverAt[p], i)
			}
		}
		n.pos = make([]map[int]int, len(n.Routes))
		for i := range n.Routes {
			r := &n.Routes[i]
			n.byID[r.ID] = r
			n.pos[i] = map[int]int{}
			for k, p := range r.Parasangs {
				if _, ok := n.pos[i][p]; !ok {
					n.pos[i][p] = k
				}
				n.routeAt[p] = nmAddOnce(n.routeAt[p], i)
			}
		}
		for i, f := range n.Features {
			for _, p := range f.Parasangs {
				n.featAt[p] = append(n.featAt[p], i)
			}
		}
		for p, l := range n.featAt {
			sort.SliceStable(l, func(a, b int) bool {
				return nmKindOrder[n.Features[l[a]].Kind] < nmKindOrder[n.Features[l[b]].Kind]
			})
			n.featAt[p] = l
		}
	})
	return n
}

func nmAddOnce(l []int, i int) []int {
	if len(l) > 0 && l[len(l)-1] == i {
		return l
	}
	return append(l, i)
}

// nmRiver is the longest named river through p (or, with near, through a neighbouring parasang).
func (w *WorldData) nmRiver(p int, near bool) *NmRiver {
	n := w.nmData()
	if n == nil || p < 0 {
		return nil
	}
	best := -1
	try := func(q int) {
		for _, i := range n.riverAt[q] {
			if best < 0 || len(n.Rivers[i].Parasangs) > len(n.Rivers[best].Parasangs) {
				best = i
			}
		}
	}
	try(p)
	if best < 0 && near {
		x, y := p%w.W, p/w.W
		for _, d := range dirs8 {
			if w.In(x+d[0], y+d[1]) {
				try(w.Idx(x+d[0], y+d[1]))
			}
		}
	}
	if best < 0 {
		return nil
	}
	return &n.Rivers[best]
}

// nmRoutes are the routes through p of the given kinds ("" = any but sky/tunnel off the surface), longest first, one per name.
func (w *WorldData) nmRoutes(p int, kinds ...string) []int {
	n := w.nmData()
	if n == nil {
		return nil
	}
	var out []int
	seen := map[string]bool{}
	for _, i := range n.routeAt[p] {
		r := &n.Routes[i]
		if len(kinds) > 0 && !nmIn(r.Kind, kinds) || seen[r.Name] {
			continue
		}
		seen[r.Name] = true
		out = append(out, i)
	}
	sort.SliceStable(out, func(a, b int) bool { return len(n.Routes[out[a]].Parasangs) > len(n.Routes[out[b]].Parasangs) })
	return out
}

func nmIn(s string, l []string) bool {
	for _, x := range l {
		if x == s {
			return true
		}
	}
	return false
}

// nmFeatures are the named features covering p, most specific first.
func (w *WorldData) nmFeatures(p int) []*NmFeature {
	n := w.nmData()
	if n == nil {
		return nil
	}
	var out []*NmFeature
	for _, i := range n.featAt[p] {
		out = append(out, &n.Features[i])
	}
	return out
}

func (w *WorldData) nmBurg(id int) (NmBurg, bool) {
	if w.Names == nil || w.Names.Burgs == nil {
		return NmBurg{}, false
	}
	b, ok := w.Names.Burgs[id]
	return b, ok
}

// nmEnds: travelling along route i from p toward q, the town ahead and behind with parasang counts.
func (w *WorldData) nmEnds(i, p, q int) (ahead string, na int, behind string, nb int) {
	n := w.nmData()
	r := &n.Routes[i]
	ip, ok1 := n.pos[i][p]
	iq, ok2 := n.pos[i][q]
	if !ok1 {
		return
	}
	toB := !ok2 || iq >= ip
	name := func(id int) string {
		if b := w.burgByID[id]; b != nil {
			if !b.Alive {
				return "the ruins of " + b.Name
			}
			return b.Name
		}
		return "the road's end"
	}
	if toB {
		return name(r.B), len(r.Parasangs) - 1 - ip, name(r.A), ip
	}
	return name(r.A), ip, name(r.B), len(r.Parasangs) - 1 - ip
}

// nmHash: a deterministic pick index from the world seed and a key (never g.R)
func nmHash(w *WorldData, key string, n int) int {
	if n <= 0 {
		return 0
	}
	return int(HashStr(w.Seed+":nm:"+key) % uint64(n))
}

// nmPrep is the preposition for being in / by a feature.
func nmPrep(f *NmFeature, water bool) string {
	switch f.Kind {
	case "peak":
		return "below " + f.Name
	case "pass":
		return "at " + f.Name
	case "lake", "bay", "gulf", "sea", "ocean":
		if water {
			return "on " + f.Name
		}
		return "by " + f.Name
	}
	return "in " + f.Name
}

func nmCut(s string, n int) string {
	if len([]rune(s)) <= n {
		return s
	}
	return string([]rune(s)[:n-1]) + "…"
}

// ---- sidebar & world map ----

func nmSide(g *Game) []string {
	if g.Cur == nil || g.OnWorld || g.Cur.Z != 0 || g.W.Names == nil {
		return nil
	}
	w := g.W
	p := w.Idx(g.Cur.WX, g.Cur.WY)
	var out []string
	if fs := w.nmFeatures(p); len(fs) > 0 {
		out = append(out, "{{g|"+nmCut(nmPrep(fs[0], w.IsWater(p)), 30)+"}}")
	}
	here := g.Cur.At(g.P.X, g.P.Y)
	if rv := w.nmRiver(p, false); rv != nil && !w.IsWater(p) {
		v := "by the "
		if here == TBridge {
			v = "crossing the "
		} else if here.D().Water > 0 {
			v = "in the "
		}
		out = append(out, "{{B|"+nmCut(v+rv.Name, 30)+"}}")
	}
	if rs := w.nmRoutes(p, "road"); len(rs) > 0 && !w.IsWater(p) {
		v := "near "
		if here == TRoad || here == TBridge {
			v = "on "
		}
		out = append(out, "{{y|"+nmCut(v+w.Names.Routes[rs[0]].Name, 30)+"}}")
	} else if rs := w.nmRoutes(p, "sea"); len(rs) > 0 && w.IsWater(p) {
		out = append(out, "{{C|"+nmCut("on "+w.Names.Routes[rs[0]].Name, 30)+"}}")
	}
	return out
}

// nmInfo: world-map info lines for a parasang (render.go).
func (g *Game) nmInfo(p int) []string {
	w := g.W
	if w.Names == nil || p < 0 || p >= w.W*w.H {
		return nil
	}
	var out []string
	fs := w.nmFeatures(p)
	for i, f := range fs {
		if i >= 2 {
			break
		}
		l := "{{g|" + f.Name + "}}"
		if f.Peak != nil && *f.Peak == p {
			l += " (the summit)"
		}
		out = append(out, l)
	}
	if rv := w.nmRiver(p, false); rv != nil && !w.IsWater(p) {
		out = append(out, "{{B|River: the "+rv.Name+"}}")
	}
	var roads []string
	for i, ri := range w.nmRoutes(p) {
		if i >= 2 {
			break
		}
		roads = append(roads, w.Names.Routes[ri].Name)
	}
	if len(roads) > 0 {
		out = append(out, "{{y|"+strings.Join(roads, "; ")+"}}")
	}
	if g.Known[p] {
		if bd := w.burgByID[w.Burg[p]]; bd != nil && bd.Alive {
			if nb, ok := w.nmBurg(bd.ID); ok {
				l := titleCase(nb.Tier)
				if nb.Harbor != "" {
					l += " · " + nb.Harbor
				}
				out = append(out, "{{W|"+l+"}}")
			}
		}
	}
	return out
}

// ---- zones ----

// nmSafe: a solid sign at (x,y) keeps every walkable neighbour connected around it (one arc in the ring).
func nmSafe(zn *Zone, x, y int) bool {
	if x < 2 || y < 2 || x > MW-3 || y > MH-3 {
		return false
	}
	switch zn.At(x, y) {
	case TGrass, TDirt, TCobble, TSand, TSnow, TMoss, TAsh, TTallGrass, TFloor, TCrystalFloor, TFungus:
	default:
		return false
	}
	i := y*MW + x
	if zn.ActorAt(x, y) != nil || zn.Notes[i] != "" {
		return false
	}
	for _, it := range zn.Items {
		if it != nil && it.X == x && it.Y == y {
			return false
		}
	}
	open, arcs := 0, 0
	for k, d := range dirs8 {
		a := zn.Walkable(x+d[0], y+d[1])
		b := zn.Walkable(x+dirs8[(k+7)%8][0], y+dirs8[(k+7)%8][1])
		if a {
			open++
		}
		if a && !b {
			arcs++
		}
	}
	return open >= 3 && arcs <= 1
}

// nmSign puts an inscribed sign with text on the safe spot nearest (x,y) within rad; false if none.
func nmSign(zn *Zone, x, y, rad int, text string) (int, int, bool) {
	for d := 0; d <= rad; d++ {
		for dy := -d; dy <= d; dy++ {
			for dx := -d; dx <= d; dx++ {
				if abs(dx) != d && abs(dy) != d {
					continue
				}
				if nmSafe(zn, x+dx, y+dy) {
					zn.Set(x+dx, y+dy, TInscription)
					zn.Notes[(y+dy)*MW+x+dx] = text
					return x + dx, y + dy, true
				}
			}
		}
	}
	return 0, 0, false
}

func nmZone(g *Game, zn *Zone, _ *Rng) {
	w := g.W
	if w.Names == nil || w.nmData() == nil {
		return
	}
	if zn.Notes == nil {
		zn.Notes = map[int]string{}
	}
	p := w.Idx(zn.WX, zn.WY)
	if zn.Z == 0 && strings.HasSuffix(zn.Name, " by the river") {
		if rv := w.nmRiver(p, true); rv != nil {
			zn.Name = strings.TrimSuffix(zn.Name, "river") + rv.Name
		}
	}
	if zn.Kind == "town" && zn.Burg > 0 {
		if bd := w.burgByID[zn.Burg]; bd != nil {
			g.nmTown(zn, bd)
		}
		return
	}
	if zn.Z == 0 && !w.IsWater(p) {
		g.nmBridges(zn, p)
		g.nmMilestones(zn, p)
	}
}

var nmBridgeForms = []string{"%s Bridge", "%s Bridge", "the Bridge over the %s", "the Old %s Bridge", "%s Crossing"}
var nmBridgeModern = []string{"%s Bridge", "the %s Viaduct", "%s Overpass", "the %s Span"}

func (g *Game) nmBridgeName(river, key string) string {
	forms := nmBridgeForms
	if g.Genre != "fantasy" && g.Genre != "mythic" && g.Genre != "twilight" {
		forms = nmBridgeModern
	}
	return fmt.Sprintf(forms[nmHash(g.W, "br:"+key, len(forms))], river)
}

// nmBridges names the road bridges of a wild zone after the river they cross (up to two signs).
func (g *Game) nmBridges(zn *Zone, p int) {
	w := g.W
	rv := w.nmRiver(p, true)
	if rv == nil {
		return
	}
	road := "road"
	if rs := w.nmRoutes(p, "road"); len(rs) > 0 {
		road = w.Names.Routes[rs[0]].Name
	} else if t, ok := w.Terms["road"]; ok && t != "" {
		road = "the " + strings.ToLower(t)
	}
	seen := make([]bool, MW*MH)
	signs := 0
	for i, t := range zn.T {
		if t != TBridge || seen[i] || signs >= 2 {
			continue
		}
		// one span: flood its bridge tiles, then sign at the first dry end
		stack := []int{i}
		seen[i] = true
		var ends [][2]int
		for len(stack) > 0 {
			j := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			x, y := j%MW, j/MW
			for _, d := range dirs8 {
				nx, ny := x+d[0], y+d[1]
				if !zn.In(nx, ny) {
					continue
				}
				k := ny*MW + nx
				switch nt := zn.T[k]; {
				case nt == TBridge && !seen[k]:
					seen[k] = true
					stack = append(stack, k)
				case nt == TRoad:
					ends = append(ends, [2]int{nx, ny})
				}
			}
		}
		if len(ends) == 0 {
			continue
		}
		e := ends[0]
		name := g.nmBridgeName(rv.Name, fmt.Sprintf("%d:%d", rv.ID, p))
		text := fmt.Sprintf("{{Y|%s}}\n%s crosses the %s here.", name, upperFirst(road), rv.Name)
		if _, _, ok := nmSign(zn, e[0], e[1], 3, text); ok {
			signs++
		}
	}
}

func upperFirst(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// nmMilestones: where a named road leaves the parasang through this zone, a milestone names the
// road, the next town ahead and the one behind.
func (g *Game) nmMilestones(zn *Zone, p int) {
	w := g.W
	n := w.nmData()
	px, py := p%w.W, p/w.W
	done := map[[2]int]bool{}
	for _, ri := range w.nmRoutes(p, "road") {
		r := &n.Routes[ri]
		k := n.pos[ri][p]
		for _, step := range []int{1, -1} {
			j := k + step
			if j < 0 || j >= len(r.Parasangs) {
				continue
			}
			q := r.Parasangs[j]
			dx, dy := sgn(q%w.W-px), sgn(q/w.W-py)
			if dx == 0 && dy == 0 || zn.ZX != 1+dx || zn.ZY != 1+dy || done[[2]int{dx, dy}] {
				continue
			}
			// the stroke end (zone.go stroke): the parasang edge toward q, in this zone's coordinates
			ex, ey := 120+dx*121-zn.ZX*MW, 37+dy*38-zn.ZY*MH
			bx, by, bd := -1, -1, 1<<30
			for y := 2; y < MH-2; y++ {
				for x := 2; x < MW-2; x++ {
					if zn.At(x, y) != TRoad {
						continue
					}
					d := (x-ex)*(x-ex) + 4*(y-ey)*(y-ey)
					if d >= 9 && d < bd {
						bx, by, bd = x, y, d
					}
				}
			}
			if bx < 0 {
				continue
			}
			ahead, na, behind, nb := w.nmEnds(ri, p, q)
			text := fmt.Sprintf("{{Y|%s}}\nAhead: {{W|%s}}, %s.", r.Name, ahead, nmParasangs(na))
			if nb > 0 {
				text += fmt.Sprintf("\nBehind: %s, %s.", behind, nmParasangs(nb))
			}
			if _, _, ok := nmSign(zn, bx, by, 2, text); ok {
				done[[2]int{dx, dy}] = true
			}
		}
	}
}

func nmParasangs(n int) string {
	if n <= 1 {
		return "one parasang"
	}
	return fmt.Sprintf("%d parasangs", n)
}

// ---- towns ----

func (g *Game) nmTown(zn *Zone, bd *BurgData) {
	w := g.W
	nb, ok := w.nmBurg(bd.ID)
	if !ok {
		return
	}
	kind, _ := g.districtAt(zn.WX, zn.WY, zn.ZX, zn.ZY)
	if zn.Z != 0 {
		kind = ""
	}
	// harbour: the district takes the harbour's name; a sign at the quay
	if nb.Harbor != "" && zn.Z == 0 && (kind == "harbor" || kind == "" && bd.Port) {
		if kind == "harbor" {
			zn.Name = nb.Harbor
		}
		for i, t := range zn.T {
			if t == TBridge {
				if _, _, ok := nmSign(zn, i%MW, i/MW, 3, fmt.Sprintf("{{Y|%s}}\nThe harbour gate of %s. Ships berth along the quay.", nb.Harbor, bd.Name)); ok {
					break
				}
			}
		}
	}
	if nb.Airdock != "" && zn.Sky {
		nmSign(zn, 40, 12, 14, fmt.Sprintf("{{Y|%s}}\nMooring posts and wind-flags: the air dock of %s.", nb.Airdock, bd.Name))
	}
	if strings.HasSuffix(zn.Name, ", the Arena") {
		zn.Name = bd.Name + ", " + g.nmArenaName(bd)
	}
	if kind == "" || kind == "market" || kind == "plaza" || kind == "harbor" {
		g.nmInn(zn, bd)
	}
	if kind != "" && kind != "farms" && kind != "mines" && kind != "arena" {
		g.nmStreets(zn, bd, kind)
	}
}

var nmArenaForms = map[string][]string{
	"":          {"the %s Arena", "the Colosseum of %s", "the %s Pit", "the Sands of %s"},
	"mythic":    {"the Stadium of %s", "the %s Hippodrome", "the Agon of %s"},
	"cyberpunk": {"the %s Dome", "the %s Cage", "Arena %s"},
	"steampunk": {"the %s Hippodrome", "the %s Pit", "the Brass Ring of %s"},
	"wasteland": {"the %s Thunderpit", "the %s Cage", "the Scrap Ring of %s"},
}

func (g *Game) nmArenaName(bd *BurgData) string {
	forms := nmArenaForms[g.Genre]
	if forms == nil {
		forms = nmArenaForms[""]
	}
	word := bd.Name
	if cu := g.W.cultByID[bd.Culture]; cu != nil && len(cu.Places) > 0 && nmHash(g.W, fmt.Sprint("ar:w:", bd.ID), 2) == 0 {
		word = cu.Places[nmHash(g.W, fmt.Sprint("ar:p:", bd.ID), len(cu.Places))]
	}
	return fmt.Sprintf(forms[nmHash(g.W, fmt.Sprint("ar:", bd.ID), len(forms))], word)
}

// inn sign tables: adjectives and nouns per genre, the house word, and the board's picture
var nmInnAdj = map[string][]string{
	"":           {"Drowned", "Gilded", "Crooked", "Laughing", "Sleeping", "Rusty", "Silver", "Merry", "Blind", "Wandering", "Red", "Black", "Golden", "Three", "Last", "Weary", "Green", "Jolly"},
	"mythic":     {"Wine-Dark", "Golden", "Laughing", "Hero's", "Oracle's", "Drunken", "Winged", "Bronze"},
	"steampunk":  {"Brass", "Smoky", "Gilded", "Crooked", "Copper", "Rusty", "Ticking", "Sooty"},
	"dieselpunk": {"Iron", "Riveted", "Blackout", "Rusty", "Lucky", "Smoky"},
	"atompunk":   {"Atomic", "Rocket", "Orbiting", "Glowing", "Lucky", "Chrome"},
	"cyberpunk":  {"Neon", "Chrome", "Static", "Glitched", "Electric", "Broken", "Ghost"},
	"biopunk":    {"Spore", "Grafted", "Budding", "Twin-Headed", "Green", "Bleeding"},
	"solarpunk":  {"Sunny", "Green", "Blooming", "Windy", "Honeyed", "Bright"},
	"wasteland":  {"Rad", "Scrap", "Last", "Dusty", "Rusty", "Lucky", "Two-Headed"},
	"twilight":   {"Fading", "Last", "Pale", "Gilded", "Sleeping", "Candlelit", "Hollow"},
}
var nmInnNoun = map[string][]string{
	"":           {"Lantern", "Goat", "Crown", "Anchor", "Stag", "Kettle", "Raven", "Barrel", "Wheel", "Fox", "Lamb", "Boar", "Owl", "Key", "Bell", "Harp", "Dragon", "Plough"},
	"mythic":     {"Amphora", "Lyre", "Bull", "Satyr", "Laurel", "Trireme", "Owl"},
	"steampunk":  {"Cog", "Boiler", "Piston", "Gasworks", "Zeppelin", "Kettle", "Spanner"},
	"dieselpunk": {"Rivet", "Tank", "Gasket", "Searchlight", "Flak Gun", "Locomotive"},
	"atompunk":   {"Saucer", "Isotope", "Rocket", "Robot", "Jukebox", "Comet"},
	"cyberpunk":  {"Socket", "Circuit", "Ghost", "Byte", "Jack", "Firewall", "Diode"},
	"biopunk":    {"Vat", "Graft", "Spore", "Leech", "Helix", "Polyp"},
	"solarpunk":  {"Kite", "Hive", "Seedling", "Windmill", "Pollinator", "Lantern"},
	"wasteland":  {"Rat", "Bottle Cap", "Tyre", "Geiger", "Vulture", "Jerrycan"},
	"twilight":   {"Candle", "Relic", "Censer", "Throne", "Swan", "Lantern"},
}
var nmInnHouse = map[string]string{"": "inn", "mythic": "wine-house", "steampunk": "public house", "dieselpunk": "canteen", "atompunk": "diner",
	"cyberpunk": "bar", "biopunk": "tap-house", "solarpunk": "commons kitchen", "wasteland": "saloon", "twilight": "old inn"}

func nmGenreTable[T any](m map[string]T, genre string) T {
	if v, ok := m[genre]; ok {
		return v
	}
	return m[""]
}

// nmInnName: the inn of a town zone (one per zone key), sometimes in the local tongue ("Kalvik's Rest").
func (g *Game) nmInnName(bd *BurgData, key string) string {
	w := g.W
	adj := nmGenreTable(nmInnAdj, g.Genre)
	noun := nmGenreTable(nmInnNoun, g.Genre)
	k := "inn:" + key
	if cu := w.cultByID[bd.Culture]; cu != nil && len(cu.Places) > 0 && nmHash(w, k+":native", 4) == 0 {
		place := cu.Places[nmHash(w, k+":place", len(cu.Places))]
		forms := []string{"%s's Rest", "the %s Arms", "the House of %s"}
		return fmt.Sprintf(forms[nmHash(w, k+":form", len(forms))], place)
	}
	return "The " + adj[nmHash(w, k+":adj", len(adj))] + " " + noun[nmHash(w, k+":noun", len(noun))]
}

func (g *Game) nmInn(zn *Zone, bd *BurgData) {
	var doors []int
	for i, t := range zn.T {
		if t == TDoor {
			doors = append(doors, i)
		}
	}
	if len(doors) == 0 {
		return
	}
	start := nmHash(g.W, "innd:"+zn.Key, len(doors))
	name := g.nmInnName(bd, zn.Key)
	house := nmGenreTable(nmInnHouse, g.Genre)
	text := fmt.Sprintf("{{Y|%s}}\nA sign over the door of the %s: food, drink and a bed for the night.", name, house)
	for k := 0; k < len(doors); k++ {
		d := doors[(start+k)%len(doors)]
		x, y := d%MW, d/MW
		// beside the door along the wall, never in front of it
		for _, o := range [][2]int{{-1, 1}, {1, 1}, {-1, -1}, {1, -1}} {
			if nmSafe(zn, x+o[0], y+o[1]) {
				zn.Set(x+o[0], y+o[1], TInscription)
				zn.Notes[(y+o[1])*MW+x+o[0]] = text
				return
			}
		}
	}
}

var nmStreetTrade = map[string][]string{
	"market": {"Market Street", "Coin Lane", "Chandlers' Row", "Haggle Lane"}, "plaza": {"High Street", "Fountain Row", "Square Lane"},
	"palace": {"Crown Street", "Herald's Way", "Court Lane"}, "temple": {"Chapel Street", "Pilgrim Lane", "Candle Row"},
	"academy": {"Scholars' Walk", "Ink Lane", "Lamp Street"}, "barracks": {"Drill Lane", "Pike Row", "Muster Street"},
	"forge": {"Smiths' Lane", "Anvil Row", "Cinder Street"}, "artisans": {"Weavers' Row", "Tanners' Lane", "Potters' Street", "Dyers' Row"},
	"slums": {"Rat Alley", "Mud Lane", "Crooked Row"}, "homes": {"Elm Street", "Well Lane", "Garden Row"}, "harbor": {"Quay Street", "Rope Walk", "Net Lane", "Tar Row"},
}
var nmStreetWord = map[string][]string{"": {"Street", "Lane", "Row", "Way"}, "cyberpunk": {"Avenue", "Strip", "Boulevard"}, "atompunk": {"Avenue", "Boulevard", "Drive"},
	"dieselpunk": {"Prospekt", "Avenue", "Strasse"}, "steampunk": {"Street", "Terrace", "Arcade"}, "solarpunk": {"Greenway", "Walk", "Lane"}, "wasteland": {"Alley", "Strip", "Row"}}

// nmStreets: one or two street signposts in a city district, named for its trade, the town's rivers and
// local places.
func (g *Game) nmStreets(zn *Zone, bd *BurgData, kind string) {
	w := g.W
	var names []string
	if l := nmStreetTrade[kind]; len(l) > 0 {
		names = append(names, l[nmHash(w, "st:t:"+zn.Key, len(l))])
	}
	word := nmGenreTable(nmStreetWord, g.Genre)
	if rv := w.nmRiver(bd.P, true); rv != nil && nmHash(w, "st:r:"+zn.Key, 2) == 0 {
		names = append(names, rv.Name+" "+word[nmHash(w, "st:rw:"+zn.Key, len(word))])
	} else if cu := w.cultByID[bd.Culture]; cu != nil && len(cu.Places) > 0 {
		names = append(names, cu.Places[nmHash(w, "st:p:"+zn.Key, len(cu.Places))]+" "+word[nmHash(w, "st:pw:"+zn.Key, len(word))])
	}
	if kind != "market" && kind != "plaza" && len(names) > 1 {
		names = names[nmHash(w, "st:one:"+zn.Key, 2):][:1] // a quiet district gets one signpost
	}
	for i, nm := range names {
		// walk the zone from a hashed start and take the first cobble beside a road or a crossing of cobbles
		start := nmHash(w, fmt.Sprint("st:s:", i, ":", zn.Key), MW*MH)
		for k := 0; k < MW*MH; k++ {
			j := (start + k*7919) % (MW * MH)
			x, y := j%MW, j/MW
			if zn.At(x, y) != TCobble || !nmSafe(zn, x, y) || zn.At(x, y-1) != TCobble && zn.At(x, y+1) != TCobble {
				continue
			}
			zn.Set(x, y, TInscription)
			zn.Notes[j] = fmt.Sprintf("{{Y|%s}}\nA signpost of %s.", nm, bd.Name)
			break
		}
	}
}

// ---- entering ----

type nmState struct {
	Last string `json:"last"` // seed:burg of the last town entered
}

func nmEnter(g *Game, zn *Zone) {
	w := g.W
	if w.Names == nil {
		return
	}
	st := g.ext("nm", func() any { return &nmState{} }).(*nmState)
	if zn.Kind != "town" || zn.Burg <= 0 {
		if zn.Z == 0 && zn.Kind != "town" {
			st.Last = ""
		}
		return
	}
	key := fmt.Sprintf("%s:%d", w.Seed, zn.Burg)
	if st.Last != key {
		st.Last = key
		if bd := w.burgByID[zn.Burg]; bd != nil && bd.Alive {
			if nb, ok := w.nmBurg(bd.ID); ok && nb.Tier != "" {
				g.Msg(fmt.Sprintf("{{W|You enter the %s of %s.}}", nb.Tier, bd.Name))
			}
		}
	}
	g.nmRiftSign(zn)
}

// nmRiftSign: a named stone beside a rift gate, in the town's tongue ("the Kalvik Gate").
func (g *Game) nmRiftSign(zn *Zone) {
	for i, t := range zn.T {
		if t != TPortal {
			continue
		}
		x, y := i%MW, i/MW
		for dy := -2; dy <= 2; dy++ {
			for dx := -2; dx <= 2; dx++ {
				if zn.At(x+dx, y+dy) == TInscription && strings.Contains(zn.Notes[(y+dy)*MW+x+dx], "rift gate") {
					return
				}
			}
		}
		bd := g.W.burgByID[zn.Burg]
		if bd == nil {
			return
		}
		name := g.nmGateName(bd)
		text := fmt.Sprintf("{{Y|%s}}\nThe rift gate of %s: %s.", name, bd.Name, nonEmpty(zn.Notes[i], "a tear onto another world"))
		nmSign(zn, x, y, 2, text)
		return
	}
}

func (g *Game) nmGateName(bd *BurgData) string {
	w := g.W
	word := bd.Name
	if cu := w.cultByID[bd.Culture]; cu != nil && len(cu.Places) > 0 {
		word = cu.Places[nmHash(w, fmt.Sprint("gate:p:", bd.ID), len(cu.Places))]
	}
	forms := []string{"the %s Gate", "the Door of %s", "the %s Threshold", "the Veil of %s"}
	return fmt.Sprintf(forms[nmHash(w, fmt.Sprint("gate:", bd.ID), len(forms))], word)
}

// ---- sites ----

// nmSiteName renames a wilderness tower or dungeon after the feature it stands in, half the time
// (dungeon.go siteName passes its own name as def, after all its rng draws).
func (g *Game) nmSiteName(theme string, p int, def string) string {
	w := g.W
	fs := w.nmFeatures(p)
	if len(fs) == 0 || nmHash(w, fmt.Sprint("site:", theme, ":", p), 2) == 0 {
		return def
	}
	f := fs[0]
	if f.Kind == "sea" || f.Kind == "ocean" || f.Kind == "bay" || f.Kind == "gulf" {
		return def
	}
	noun, _, ok := strings.Cut(def, " of ")
	if !ok {
		return def
	}
	return noun + " of " + f.Name
}

// ---- gossip ----

var nmGossipLines = map[string][]string{
	"range":   {"The %s lie %s. Nobody crosses them in winter if they can help it.", "My uncle went up into the %s %s and came back with white hair."},
	"peak":    {"On a clear day you can see %s %s. They say the summit is holy.", "%s %s has never been climbed by anyone sane."},
	"pass":    {"%s %s is the only way over. Pay the tolls and keep moving.", "Snow closes %s %s early most years."},
	"hills":   {"Sheep and old barrows: that's the %s, %s.", "Shepherds in the %s %s have seen lights at night."},
	"lake":    {"The fish from %s %s are the best for a hundred leagues.", "Something big lives in %s, %s. Ask any fisher."},
	"bay":     {"Boats out of %s, %s, bring in the best catch.", "Smugglers like %s %s. Lots of coves."},
	"gulf":    {"Storms come in off %s %s.", "Half the fleets of the coast cross %s, %s."},
	"sea":     {"Ships sail %s %s, when the wind allows.", "My brother drowned in %s, %s."},
	"ocean":   {"Nobody knows what is on the far side of %s, %s.", "Sailors say %s %s has no bottom."},
	"forest":  {"Don't wander %s %s after dark.", "Woodcutters go into %s %s and not all of them come out."},
	"jungle":  {"Fevers and snakes in %s %s. Stay out.", "There are ruins deep in %s %s, they say."},
	"taiga":   {"Wolves in %s %s this year.", "Trappers sell their furs from %s %s here."},
	"fungal":  {"Breathe through cloth in %s %s.", "The glowing caps of %s %s fetch good coin."},
	"desert":  {"Crossing %s %s? Carry twice the water you think.", "Caravans vanish in %s %s."},
	"barrens": {"Nothing grows in %s %s but thorns.", "The wind never stops in %s %s."},
	"plain":   {"Riders sweep across %s %s.", "Good grazing on %s %s."},
	"marsh":   {"%s %s swallow carts whole.", "Marsh-lights in %s %s lead fools astray."},
	"tundra":  {"Out on %s %s the cold kills quietly.", "Herders follow the beasts across %s %s."},
	"ice":     {"%s %s cracks like thunder in spring.", "Things frozen in %s %s are older than kings."},
	"ash":     {"%s %s still smoulder underfoot.", "Ash from %s %s gets into everything."},
	"crystal": {"%s %s sing when the wind blows.", "Prospectors cut themselves to ribbons in %s %s."},
	"lava":    {"%s %s glow at night.", "Only fools and smiths go near %s %s."},
}

// nmDir is a compass phrase from parasang a to b.
func nmDir(w *WorldData, a, b int) string {
	dx, dy := b%w.W-a%w.W, b/w.W-a/w.W
	if dx*dx+dy*dy <= 1 {
		return "close by"
	}
	ns, ew := "", ""
	if dy*3 < -abs(dx) {
		ns = "north"
	} else if dy*3 > abs(dx) {
		ns = "south"
	}
	if dx*3 > abs(dy) {
		ew = "east"
	} else if dx*3 < -abs(dy) {
		ew = "west"
	}
	return "to the " + nonEmpty(ns+ew, "near")
}

// nmGossip: about one time in four, a line about a named feature, river or road within a few parasangs.
func (g *Game) nmGossip(a *Actor) string {
	w := g.W
	if w.Names == nil || g.Cur == nil || a == nil {
		return ""
	}
	key := fmt.Sprintf("gos:%d:%d", a.ID, g.Turn/600)
	if nmHash(w, key, 4) != 0 {
		return ""
	}
	p := w.Idx(g.Cur.WX, g.Cur.WY)
	n := w.nmData()
	// candidates: features within 5 parasangs (by their nearest parasang), rivers and roads here
	type cand struct {
		line string
	}
	var cs []cand
	px, py := p%w.W, p/w.W
	for i := range n.Features {
		f := &n.Features[i]
		lines := nmGossipLines[f.Kind]
		if len(lines) == 0 {
			continue
		}
		best, bd := -1, 1<<30
		for _, q := range f.Parasangs {
			d := (q%w.W-px)*(q%w.W-px) + (q/w.W-py)*(q/w.W-py)
			if d < bd {
				best, bd = q, d
			}
		}
		if bd > 25 {
			continue
		}
		l := lines[nmHash(w, fmt.Sprint(key, ":l:", i), len(lines))]
		cs = append(cs, cand{upperFirst(fmt.Sprintf(l, f.Name, nmDir(w, p, best)))})
	}
	if rv := w.nmRiver(p, true); rv != nil {
		cs = append(cs, cand{fmt.Sprintf("The %s floods every spring. Mind the fords.", rv.Name)})
	}
	if rs := w.nmRoutes(p, "road"); len(rs) > 0 {
		r := &n.Routes[rs[0]]
		other := r.B
		if bd := w.burgByID[r.B]; bd != nil && bd.P == p {
			other = r.A
		}
		if bd := w.burgByID[other]; bd != nil {
			cs = append(cs, cand{fmt.Sprintf("Take %s if you are bound for %s. Safer than the wilds.", r.Name, bd.Name)})
		}
	}
	if len(cs) == 0 {
		return ""
	}
	return "\"" + cs[nmHash(w, key+":pick", len(cs))].line + "\""
}
