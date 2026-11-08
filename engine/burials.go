package engine

import (
	"fmt"
	"sort"
	"strings"
)

// Burials, bunkers and hoards (prefix bu). Barrows hold the grave goods of a real dead person (taboo, ward,
// curse, heirs); pre-war bunkers hide under ruins in the punk/wasteland genres (hatch found by search, opened
// by code, hack or force); pirate hoards hang on split charts held by different people; the ground itself
// shifts (floods, dunes, quakes, calving ice) and exposes or takes back finds; appraisal, collectors, fences,
// claims and a Finds ledger follow. Sites are a pure function of the world (cached per WorldData), the zone hook
// uses side rngs only. State: g.ext("bu") (planet-local).
//
// Hooks (optional, see report): buDig(x,y) before bhDig, buSearch after `s`, buBump in interact, buRead for
// case "map", buFinds for a key. Without them everything still works by walking up to a site (menu on
// arrival), the turn hook, talk options and Item.Text on the charts.

func init() {
	unLocalKeys = append(unLocalKeys, "bu")
	zoneHooks = append(zoneHooks, buZone)
	enterHooks = append(enterHooks, buEnter)
	turnHooks = append(turnHooks, buTurn)
	talkHooks = append(talkHooks, buTalk)
	killHooks = append(killHooks, buKill)
	modeKeys["bumenu"] = func(g *Game, k string) { g.keyMenu(k, func() { buClose(g) }) }
	modeDraws["bumenu"] = buDrawMenu
	skAddStatus(&skStatusDef{ID: "gravesick", Name: "Grave sickness", Glyph: 'G', AGI: -2, WIL: -2, Speed: -10, Tick: func(g *Game, a *Actor) {
		if g.Turn%45 == 0 {
			g.hurt(a, 1, nil, "grave sickness")
		}
	}})
}

// ---------------------------------------------------------------- sites (pure)

type buHolder struct {
	Kind string // tavern rival ship
	Burg int
	P    int
}

type buSite struct {
	Num     int
	ID      string
	Kind    string // barrow bunker hoard
	P       int
	WX, WY  int
	ZX, ZY  int
	Person  int // PersonData index, -1 none
	Parts   int
	Name    string
	Captain string
	Isle    bool
	State   int
	Burg    int
	Holders []buHolder
}

type buIndex struct {
	sites []*buSite
	byP   map[int]*buSite
	byID  map[string]*buSite
	byNum map[int]*buSite
	dead  map[int][]int
	all   []int
}

var buCache = map[*WorldData]*buIndex{}

func buIdx(w *WorldData) *buIndex {
	if ix := buCache[w]; ix != nil {
		return ix
	}
	ix := &buIndex{byP: map[int]*buSite{}, byID: map[string]*buSite{}, byNum: map[int]*buSite{}, dead: map[int][]int{}}
	buCache[w] = ix
	var ds []int
	for i := range w.People {
		p := &w.People[i]
		if !p.Alive && p.Died > 0 && p.Name != "" {
			ds = append(ds, i)
		}
	}
	sort.Slice(ds, func(a, b int) bool {
		pa, pb := &w.People[ds[a]], &w.People[ds[b]]
		if pa.Renown != pb.Renown {
			return pa.Renown > pb.Renown
		}
		return pa.ID < pb.ID
	})
	for _, i := range ds {
		if st := w.People[i].State; len(ix.dead[st]) < 12 {
			ix.dead[st] = append(ix.dead[st], i)
		}
		if len(ix.all) < 40 {
			ix.all = append(ix.all, i)
		}
	}
	for p := 0; p < len(w.Biome); p++ {
		if w.IsWater(p) {
			continue
		}
		if s := buMake(w, ix, p); s != nil {
			s.Num = len(ix.sites) + 1
			ix.sites = append(ix.sites, s)
			ix.byP[p] = s
			ix.byID[s.ID] = s
			ix.byNum[s.Num] = s
		}
	}
	for _, s := range ix.sites {
		if s.Kind == "hoard" {
			buMakeHolders(w, s)
		}
	}
	return ix
}

func buSphere(w *WorldData, p int) string {
	if p >= 0 && p < len(w.Culture) {
		if cu := w.cultByID[w.Culture[p]]; cu != nil {
			return cu.Sphere
		}
	}
	return ""
}

func buMake(w *WorldData, ix *buIndex, p int) *buSite {
	r := NewRng(fmt.Sprintf("%s:bu:site:%d", w.Seed, p))
	x, y := p%w.W, p/w.W
	burg := w.burgByID[w.Burg[p]]
	if burg != nil && burg.Alive {
		return nil
	}
	s := &buSite{P: p, WX: x, WY: y, ZX: r.Int(0, 2), ZY: r.Int(0, 2), Person: -1, State: w.State[p]}
	rb, rh, rm := r.Float(), r.Float(), r.Float()
	// pre-war bunker
	if buBunkerGenre(w.Genre) {
		pr := 0.006
		if w.Genre == "wasteland" {
			pr = 0.014
		}
		if burg != nil {
			pr = 0.5
		}
		if rb < pr && (burg == nil || burg.Layer == "surface") {
			s.Kind = "bunker"
			s.ID = fmt.Sprintf("bk:%d", p)
			kit := buBunkerKitFor(w.Genre)
			place := w.RegionName(p)
			if burg != nil {
				place = burg.Name
				s.ZX, s.ZY, s.Burg = 1, 1, burg.ID
				if burg.State > 0 {
					s.State = burg.State
				}
			}
			s.Name = strings.ToUpper(kit.Place[:1]) + kit.Place[1:] + " under " + place
			return s
		}
	}
	if burg != nil {
		return nil
	}
	// pirate hoard on a coast or an islet
	land, wet := 0, false
	for _, d := range dirs8 {
		nx, ny := x+d[0], y+d[1]
		if !w.In(nx, ny) {
			continue
		}
		if w.IsWater(w.Idx(nx, ny)) {
			wet = true
		} else {
			land++
		}
	}
	if wet {
		pr := 0.02
		if land <= 3 {
			pr = 0.07
			s.Isle = true
		}
		if rh < pr {
			s.Kind = "hoard"
			s.ID = fmt.Sprintf("hd:%d", p)
			s.Parts = 2 + r.Int(0, 1)
			given := "Morrow"
			if cu := w.cultByID[w.Culture[p]]; cu != nil && len(cu.Given) > 0 {
				given = Pick(r, cu.Given)
			}
			if r.Chance(0.5) {
				s.Captain = given + " " + Pick(r, buPirateEpithets)
			} else {
				s.Captain = Pick(r, buPirateTitles) + " " + Pick(r, buPirateEpithets)
			}
			s.Name = "Hoard of " + s.Captain
			return s
		}
	}
	// barrow
	pr := 0.02
	sph := buSphereGroup(buSphere(w, p))
	if sph == "mound" || sph == "kurgan" {
		pr *= 2.5
	}
	switch w.Genre {
	case "mythic", "fantasy", "twilight":
		pr *= 1.8
	}
	if rm < pr {
		s.Kind = "barrow"
		s.ID = fmt.Sprintf("bw:%d", p)
		list := ix.dead[w.State[p]]
		if len(list) == 0 {
			list = ix.all
		}
		if len(list) > 0 {
			s.Person = list[r.Int(0, len(list)-1)]
			s.State = w.People[s.Person].State
			s.Name = "Barrow of " + w.People[s.Person].Name
		} else {
			s.Name = "Nameless barrow"
		}
		return s
	}
	return nil
}

func buMakeHolders(w *WorldData, s *buSite) {
	type bd struct {
		b *BurgData
		d int
	}
	var ports, others []bd
	for i := range w.Burgs {
		b := &w.Burgs[i]
		if !b.Alive || b.Layer != "surface" {
			continue
		}
		d := maxi(abs(b.P%w.W-s.WX), abs(b.P/w.W-s.WY))
		if d > 30 {
			continue
		}
		if b.Port {
			ports = append(ports, bd{b, d})
		} else {
			others = append(others, bd{b, d})
		}
	}
	less := func(l []bd) {
		sort.Slice(l, func(i, j int) bool {
			if l[i].d != l[j].d {
				return l[i].d < l[j].d
			}
			return l[i].b.ID < l[j].b.ID
		})
	}
	less(ports)
	less(others)
	ports = append(ports, others...)
	r := NewRng(fmt.Sprintf("%s:bu:holders:%s", w.Seed, s.ID))
	for k := 0; k < s.Parts && len(ports) > 0; k++ {
		b := ports[k%len(ports)].b
		h := buHolder{Kind: Pick(r, buHolderKinds), Burg: b.ID, P: b.P}
		if k == 0 && h.Kind == "tavern" && s.Parts == 2 {
			h.Kind = "rival"
		}
		if h.Kind != "tavern" {
			// a land parasang near the burg (rivals) or on its coast (ships)
			bx, by := b.P%w.W, b.P/w.W
			for i := 0; i < 80; i++ {
				px, py := bx+r.Int(-4, 4), by+r.Int(-4, 4)
				if !w.In(px, py) {
					continue
				}
				q := w.Idx(px, py)
				if w.IsWater(q) || w.Burg[q] > 0 || (px == bx && py == by) {
					continue
				}
				if h.Kind == "ship" {
					wet := false
					for _, d := range dirs8 {
						if w.In(px+d[0], py+d[1]) && w.IsWater(w.Idx(px+d[0], py+d[1])) {
							wet = true
						}
					}
					if !wet {
						continue
					}
				}
				h.P = q
				break
			}
		}
		s.Holders = append(s.Holders, h)
	}
}

func buHolderHintText(g *Game, s *buSite, part int) string {
	if part < 1 || part > len(s.Holders) {
		return ""
	}
	h := s.Holders[part-1]
	where := "the coast"
	if bd := g.W.burgByID[h.Burg]; bd != nil {
		where = bd.Name
	}
	return fmt.Sprintf(buHolderHint[h.Kind], where)
}

// ---------------------------------------------------------------- state

type buRec struct {
	Stage  int    `json:"s,omitempty"`
	Placed bool   `json:"pl,omitempty"`
	Zone   string `json:"z,omitempty"`
	X      int    `json:"x,omitempty"`
	Y      int    `json:"y,omitempty"`
	EX     int    `json:"ex,omitempty"`
	EY     int    `json:"ey,omitempty"`
	Known  bool   `json:"kn,omitempty"`
	Code   bool   `json:"cd,omitempty"`
	Rival  bool   `json:"rv,omitempty"`
	Day    int    `json:"d,omitempty"`
	Tried  int    `json:"tr,omitempty"`
}

type buClaim struct {
	ID      int    `json:"id"`
	Site    string `json:"site"`
	Name    string `json:"n"`
	State   int    `json:"st"`
	Rel     int    `json:"rel,omitempty"`
	Kind    string `json:"k"` // heir hunt
	Due     int    `json:"due"`
	Value   int    `json:"v"`
	Done    bool   `json:"done,omitempty"`
	Spawned bool   `json:"sp,omitempty"`
	Grave   bool   `json:"gr,omitempty"`
}

type buFind struct {
	Day   int    `json:"d"`
	Year  int    `json:"y"`
	Title string `json:"t"`
	Text  string `json:"x"`
	Value int    `json:"v"`
}

type buExp struct {
	Zone string `json:"z"`
	ID   int    `json:"i"`
}

type buState struct {
	N         int               `json:"n"`
	Rec       map[string]*buRec `json:"rec,omitempty"`
	ByZone    map[string]string `json:"bz,omitempty"`
	Parts     map[int]int       `json:"pt,omitempty"` // site num → bitmask of chart pieces ever held
	Tavern    map[string]int    `json:"tv,omitempty"` // "num:part" → actor id holding the piece
	Given     map[string]bool   `json:"gv,omitempty"`
	Claims    []*buClaim        `json:"cl,omitempty"`
	Heirs     map[int]int       `json:"hr,omitempty"` // heir actor id → claim id
	Finds     []buFind          `json:"fd,omitempty"`
	Prov      map[int]string    `json:"pv,omitempty"`
	Shift     map[string]int    `json:"sh,omitempty"`
	Firsts    map[string]bool   `json:"fs,omitempty"`
	Rumor     map[string]bool   `json:"ru,omitempty"`
	LastDay   int               `json:"ld,omitempty"`
	Near      string            `json:"-"`
	title     string
	lastPrint int
}

func buSt(g *Game) *buState {
	s := g.ext("bu", func() any { return &buState{} }).(*buState)
	if s.Rec == nil {
		s.Rec = map[string]*buRec{}
	}
	if s.ByZone == nil {
		s.ByZone = map[string]string{}
	}
	if s.Parts == nil {
		s.Parts = map[int]int{}
	}
	if s.Tavern == nil {
		s.Tavern = map[string]int{}
	}
	if s.Given == nil {
		s.Given = map[string]bool{}
	}
	if s.Heirs == nil {
		s.Heirs = map[int]int{}
	}
	if s.Prov == nil {
		s.Prov = map[int]string{}
	}
	if s.Shift == nil {
		s.Shift = map[string]int{}
	}
	if s.Firsts == nil {
		s.Firsts = map[string]bool{}
	}
	if s.Rumor == nil {
		s.Rumor = map[string]bool{}
	}
	return s
}

func (st *buState) next() int { st.N++; return st.N }

func (st *buState) rec(id string) *buRec {
	r := st.Rec[id]
	if r == nil {
		r = &buRec{}
		st.Rec[id] = r
	}
	return r
}

func buRng(g *Game, parts ...string) *Rng {
	return NewRng(g.Seed + ":bu:" + strings.Join(parts, ":"))
}

// buDigSites: parasangs (index) of live hoards, for the rival adventurers (rival.go) to race you to.
func (g *Game) buDigSites() []int {
	return g.buLiveSites("hoard")
}

// buLiveSites lists parasangs of unlooted sites of a kind ("" = every kind).
func (g *Game) buLiveSites(kind string) []int {
	st := buSt(g)
	var out []int
	for _, s := range buIdx(g.W).sites {
		if kind != "" && s.Kind != kind {
			continue
		}
		if r := st.Rec[s.ID]; r != nil && r.Stage >= 3 {
			continue
		}
		out = append(out, s.P)
	}
	return out
}

// buRivalLoot lets a rival adventurer empty the site at parasang p; returns whether there was one.
func (g *Game) buRivalLoot(p int) bool {
	s := buIdx(g.W).byP[p]
	if s == nil {
		return false
	}
	r := buSt(g).rec(s.ID)
	if r.Stage >= 3 {
		return false
	}
	r.Stage, r.Rival = 3, true
	return true
}

func buPerson(g *Game, s *buSite) *PersonData {
	if s.Person >= 0 && s.Person < len(g.W.People) {
		return &g.W.People[s.Person]
	}
	return nil
}

func buDeadName(g *Game, s *buSite) string {
	if p := buPerson(g, s); p != nil {
		return p.Name
	}
	return "the nameless chief"
}

func buFirst(g *Game, key, text string) {
	st := buSt(g)
	if st.Firsts[key] {
		return
	}
	st.Firsts[key] = true
	g.deed("event", text, 0, 0)
}

func buAddFind(g *Game, title, text string, value int) {
	st := buSt(g)
	st.Finds = append(st.Finds, buFind{Day: g.Day(), Year: g.W.Year + g.Day()/YearDays, Title: title, Text: text, Value: value})
}

// ---------------------------------------------------------------- zone hook

func buPlain(zn *Zone) bool {
	return zn.Z == 0 && zn.Kind != "town" && zn.Kind != "sea" && !zn.Underwater && !zn.Sky
}

func buSpotOK(zn *Zone, x, y, rad int) bool {
	for dy := -rad; dy <= rad; dy++ {
		for dx := -rad; dx <= rad; dx++ {
			t := zn.At(x+dx, y+dy)
			if !zn.Walkable(x+dx, y+dy) || !tgNatural(t) || zn.WaterAt(x+dx, y+dy) > 0 || zn.ActorAt(x+dx, y+dy) != nil || len(zn.ItemsAt(x+dx, y+dy)) > 0 {
				return false
			}
		}
	}
	return true
}

func buSpot(zn *Zone, r *Rng, rad int) (int, int, bool) {
	for i := 0; i < 500; i++ {
		x, y := r.Int(4, MW-5), r.Int(4, MH-5)
		if buSpotOK(zn, x, y, rad) {
			return x, y, true
		}
	}
	x, y, ok := zn.FreeSpot(r, func(x, y int) bool { return x > 2 && y > 2 && x < MW-3 && y < MH-3 })
	return x, y, ok
}

func buZone(g *Game, zn *Zone, _ *Rng) {
	if !buPlain(zn) || !g.W.In(zn.WX, zn.WY) {
		return
	}
	ix := buIdx(g.W)
	p := g.W.Idx(zn.WX, zn.WY)
	if s := ix.byP[p]; s != nil {
		ok := zn.ZX == s.ZX && zn.ZY == s.ZY
		if s.Kind == "bunker" && s.Burg != 0 {
			ok = zn.Kind == "ruin"
		}
		if ok {
			buPlace(g, zn, s)
		}
	}
	buPlaceParts(g, zn, p, ix)
	buPlaceNotes(g, zn, ix)
}

func buPlace(g *Game, zn *Zone, s *buSite) {
	st := buSt(g)
	rec := st.rec(s.ID)
	if rec.Placed {
		return
	}
	r := NewRng(g.Seed + ":bu:spot:" + s.ID)
	switch s.Kind {
	case "barrow":
		x, y, ok := buSpot(zn, r, 2)
		if !ok {
			return
		}
		for dy := -1; dy <= 1; dy++ {
			for dx := -2; dx <= 2; dx++ {
				if zn.Walkable(x+dx, y+dy) {
					zn.Set(x+dx, y+dy, TDirt)
				}
			}
		}
		zn.Set(x, y, TInscription)
		zn.Set(x, y+1, TRubble)
		if zn.Walkable(x+2, y) && r.Chance(0.5) {
			zn.Set(x+2, y, TBones)
		}
		zn.Notes[y*MW+x] = buEpitaph(g, s) + "\n\n{{K|A heap of earth and stones below the stone seals a door. A spade (D) would open it.}}"
		rec.X, rec.Y, rec.EX, rec.EY = x, y, x, y+1
	case "bunker":
		x, y, ok := buSpot(zn, r, 1)
		if !ok {
			return
		}
		t := TRubble
		if b := g.W.Biome[s.P]; b == 1 || b == 2 {
			t = TSand
		}
		for dy := -1; dy <= 1; dy++ {
			for dx := -1; dx <= 1; dx++ {
				if zn.Walkable(x+dx, y+dy) {
					zn.Set(x+dx, y+dy, t)
				}
			}
		}
		rec.X, rec.Y, rec.EX, rec.EY = x, y, x, y
	case "hoard":
		x, y, ok := buSpot(zn, r, 0)
		if !ok {
			return
		}
		rec.X, rec.Y, rec.EX, rec.EY = x, y, x, y
	}
	rec.Placed, rec.Zone = true, zn.Key
	st.ByZone[zn.Key] = s.ID
}

// ---------------------------------------------------------------- epitaphs and goods

func buEpitaph(g *Game, s *buSite) string {
	r := NewRng(g.Seed + ":bu:epi:" + s.ID)
	grp := buSphereGroup(buSphere(g.W, s.P))
	flour := Pick(r, buEpitaphFlourish[grp])
	name := buDeadName(g, s)
	if p := buPerson(g, s); p != nil {
		realm := g.W.StateName(p.State)
		title := nonEmpty(p.Title, titleCase(p.Role))
		cause := ""
		if p.Cause != "" {
			cause = ", " + p.Cause
		}
		return fmt.Sprintf("%s, %s of %s.\nBorn in the year %d, dead in the year %d%s.\n\n%s", p.Name, title, realm, p.Born, p.Died, cause, strings.ReplaceAll(flour, "{n}", p.Name))
	}
	return strings.ReplaceAll(flour, "{n}", name)
}

func buTraits(g *Game, s *buSite) (ward, curse string) {
	r := NewRng(g.Seed + ":bu:traits:" + s.ID)
	ward = Pick(r, []string{"poison", "collapse", "none", "none"})
	curse = Pick(r, []string{"ghost", "sick", "both", "quiet", "omen", "hexword", "sick"})
	return
}

func buNominal(it *Item, r *Rng) {
	if it.Extra == nil {
		it.Extra = map[string]int{}
	}
	it.Extra["bu:nom"] = maxi(it.Value, 10)
	it.Extra["bu:tv"] = 50 + r.Int(0, 250)
	it.Extra["bu:app"] = 1
}

func buOdd(g *Game, r *Rng, tier int, story string) *Item {
	var it *Item
	if r.Chance(0.3) {
		it = g.MakeOddity(r, tier)
		it.Name = Pick(r, buOddNames)
	} else {
		it = &Item{ID: g.newID(), Kind: "trophy", Name: Pick(r, buOddNames), Glyph: '♦', F: CYellow, Value: 30 + tier*25 + r.Int(0, 40), Weight: 1}
	}
	it.Lore = "Unusual. A scholar or collector could say what it is worth."
	buNominal(it, r)
	buSt(g).Prov[it.ID] = story + " Made by " + Pick(r, buOddMakers) + "."
	return it
}

// ---------------------------------------------------------------- menus

func buOpen(g *Game, title, text string, opts []Option) {
	st := buSt(g)
	st.title = title
	g.Popup = Wrap(text, 100)
	g.Opts = opts
	g.Sel = 0
	g.Mode = "bumenu"
}

func buClose(g *Game) {
	g.Mode, g.Talk, g.Popup, g.Opts = "main", nil, nil, nil
}

func buDrawMenu(g *Game) {
	s := g.sc
	h := len(g.Popup) + len(g.Opts) + 4
	y0 := maxi(1, (MH-h)/2)
	s.Box(2, y0, SW-4, mini(h, SH-2), CDarkMagenta, buSt(g).title)
	y := y0 + 1
	for _, l := range g.Popup {
		s.TextB(6, y, l, CGray, CBgDark, SW-3)
		y++
	}
	y++
	for i, o := range g.Opts {
		bg := CBgDark
		if i == g.Sel {
			bg = CBgSel
		}
		s.Fill(4, y, SW-8, 1, bg)
		s.TextB(4, y, fmt.Sprintf("{{W|%c)}} %s", 'a'+i, o.Text), CGray, bg, SW-3)
		y++
	}
}

func buHostileNear(g *Game) bool {
	for _, a := range g.Cur.Actors {
		if a.HP > 0 && g.Hostile(a) && g.visible(a.X, a.Y) {
			return true
		}
	}
	return false
}

// buDigTime spends minutes digging; false when interrupted.
func buDigTime(g *Game, need int) bool {
	if bhTool(g) {
		need /= 4
	}
	for i := 0; i < need; i++ {
		if buHostileNear(g) {
			g.Msg("{{R|You stop digging: something is near!}}")
			return false
		}
		g.spend(1000)
		if g.Dead {
			return false
		}
	}
	return true
}

func buWitnessed(g *Game, value int, why string) {
	w := g.uwWitnesses(nil)
	for range w {
		if g.R.Chance(0.45) {
			g.uwAddBounty(g.uwStateHere(), 40+value/10, why)
			return
		}
	}
}

// ---- proximity: step up to a site

func buTurn(g *Game) {
	if g.Dead || g.Cur == nil {
		return
	}
	st := buSt(g)
	buScanInv(g, st)
	if day := g.Day(); day != st.LastDay {
		st.LastDay = day
		buDaily(g, st)
	}
	zn := g.Cur
	id := st.ByZone[zn.Key]
	if id == "" || zn.Z != 0 {
		st.Near = ""
		return
	}
	s := buIdx(g.W).byID[id]
	rec := st.Rec[id]
	if s == nil || rec == nil || !rec.Placed || rec.Stage >= 3 {
		return
	}
	d := maxi(abs(g.P.X-rec.X), abs(g.P.Y-rec.Y))
	// hidden hatch: passive notice
	if s.Kind == "bunker" && rec.Stage == 0 {
		if rec.Known && d <= 2 {
			buReveal(g, s, rec, "Your recording's directions match this rubble. A hatch plate shows under the debris!")
		} else if d <= 1 && rec.Tried != g.Day()+1 {
			rec.Tried = g.Day() + 1
			if g.roll2d6Check(g.checkMod("perception"), dcMedium+3) {
				buReveal(g, s, rec, "Something flat and regular shows under the rubble: a hatch plate!")
			}
		}
		return
	}
	if d > 3 {
		if st.Near == id {
			st.Near = ""
		}
		return
	}
	if st.Near == id || g.Mode != "main" || buHostileNear(g) {
		return
	}
	switch s.Kind {
	case "barrow":
		if d <= 1 || maxi(abs(g.P.X-rec.EX), abs(g.P.Y-rec.EY)) <= 1 {
			st.Near = id
			buBarrowMenu(g, s, rec)
		}
	case "bunker":
		if d <= 1 {
			st.Near = id
			buBunkerMenu(g, s, rec)
		}
	case "hoard":
		if d == 0 && buMaskFull(st, s) {
			st.Near = id
			buHoardMenu(g, s, rec)
		}
	}
}

func buScanInv(g *Game, st *buState) {
	for _, it := range g.P.Inv {
		if it.Extra == nil {
			continue
		}
		if n := it.Extra["buc"]; n > 0 && it.Kind == "map" {
			if st.Parts[n]&(1<<uint(it.Extra["buh"])) == 0 {
				st.Parts[n] |= 1 << uint(it.Extra["buh"])
				if s := buIdx(g.W).byNum[n]; s != nil {
					it.Text = buMapText(g, s, it.Extra["buh"])
					if buMaskFull(st, s) {
						g.Msg("{{c|The pieces of " + s.Captain + "'s chart fit together: you know where it lies.}}")
						buFirst(g, "chart", fmt.Sprintf("%s pieced together the split chart of %s.", g.P.Name, s.Captain))
					}
				}
			}
		}
		if n := it.Extra["bu:code"]; n > 0 {
			if s := buIdx(g.W).byNum[n]; s != nil {
				if rec := st.rec(s.ID); !rec.Code {
					rec.Code = true
					g.Msg("{{c|The note's code is for the " + strings.ToLower(buBunkerKitFor(g.Genre).Place) + " at " + g.W.RegionName(s.P) + ".}}")
				}
			}
		}
		if n := it.Extra["bu:hatch"]; n > 0 {
			if s := buIdx(g.W).byNum[n]; s != nil {
				if rec := st.rec(s.ID); !rec.Known {
					rec.Known = true
					g.Msg("{{c|The recording tells you where a sealed cache lies: parasang (" + fmt.Sprint(s.WX) + "," + fmt.Sprint(s.WY) + ").}}")
				}
			}
		}
	}
}

func buMaskFull(st *buState, s *buSite) bool {
	if s.Kind != "hoard" {
		return false
	}
	m := st.Parts[s.Num]
	for k := 1; k <= s.Parts; k++ {
		if m&(1<<uint(k)) == 0 {
			return false
		}
	}
	return true
}

// ---------------------------------------------------------------- barrows

func buBarrowMenu(g *Game, s *buSite, rec *buRec) {
	grp := buSphereGroup(buSphere(g.W, s.P))
	word := Pick(buRng(g, "word", s.ID), buMoundWord[grp])
	text := fmt.Sprintf("A %s rises here. A standing stone bears an inscription.", word)
	var opts []Option
	switch rec.Stage {
	case 0:
		opts = append(opts, Option{"Read the inscription.", func() {
			g.gainXP(5 + g.Skills["lore"]*5)
			buOpen(g, s.Name, buEpitaph(g, s)+"\n\n{{K|The sealed door lies below the heap of stones.}}", []Option{{"Back.", func() { buBarrowMenu(g, s, rec) }}})
		}})
		opts = append(opts, Option{"Dig through the sealed door.", func() { buBarrowBreak(g, s, rec) }})
	case 1:
		text += " The door stands open."
		opts = append(opts, Option{"Enter the burial chamber.", func() { buChamber(g, s, rec) }})
	}
	opts = append(opts, Option{"Leave it be.", func() { buClose(g) }})
	buOpen(g, s.Name, text, opts)
}

func buBarrowBreak(g *Game, s *buSite, rec *buRec) {
	buClose(g)
	if !buDigTime(g, 40) {
		return
	}
	rec.Stage = 1
	ward, _ := buTraits(g, s)
	g.Msg("{{Y|The seal breaks. " + buWardText[ward] + "}}")
	lvl := g.Cur.Level
	switch ward {
	case "poison":
		if g.roll2d6Check(g.checkMod("perception"), dcMedium) {
			g.Msg("{{G|You hold your breath and step back until it clears.}}")
		} else {
			g.P.AddSt("poisoned", 25)
			g.Msg("{{R|The grave-air burns your lungs. You are poisoned.}}")
		}
	case "collapse":
		if g.roll2d6Check(g.checkMod("acrobatics"), dcMedium) {
			g.Msg("{{G|You throw yourself clear.}}")
		} else {
			g.hurt(g.P, 5+lvl, nil, "a barrow collapse")
		}
	}
	buFirst(g, "barrow", fmt.Sprintf("%s broke into the barrow of %s.", g.P.Name, buDeadName(g, s)))
	buBarrowMenu(g, s, rec)
}

func buChamber(g *Game, s *buSite, rec *buRec) {
	name := buDeadName(g, s)
	kit := buKitFor(g.Genre)
	text := fmt.Sprintf("The chamber is dry and cold. %s lies on a stone bier, surrounded by their goods: a blade, an ornament, a vessel, and a heap of %ss.", name, kit.Coin)
	if p := buPerson(g, s); p != nil {
		text += fmt.Sprintf("\n\n{{R|Taking them will anger %s%s.}}", g.realmName(p.State), buFaithBit(g, p))
	}
	buOpen(g, s.Name, text, []Option{
		{"Take everything.", func() { buLootBarrow(g, s, rec, true) }},
		{"Take only the coins.", func() { buLootBarrow(g, s, rec, false) }},
		{"Leave the dead in peace.", func() { buClose(g) }},
	})
}

func buFaithBit(g *Game, p *PersonData) string {
	if rel := g.W.relByID[p.Religion]; rel != nil {
		return " and the faith of " + rel.Name
	}
	return ""
}

func buLootBarrow(g *Game, s *buSite, rec *buRec, full bool) {
	buClose(g)
	zn := g.Cur
	rec.Stage, rec.Day = 3, g.Day()
	r := buRng(g, "goods", s.ID)
	kit := buKitFor(g.Genre)
	name := buDeadName(g, s)
	tier := clampi(zn.Level/3+1, 1, 5)
	_, curse := buTraits(g, s)
	state, rel := 0, 0
	if p := buPerson(g, s); p != nil {
		state, rel = p.State, p.Religion
	}
	coin := 30 + r.Int(0, 60) + zn.Level*8
	if full {
		coin += 40
	}
	g.P.Gold += coin
	g.Msg(fmt.Sprintf("{{Y|%d %s in %ss.}}", coin, g.Sk.Currency, kit.Coin))
	var items []*Item
	val := coin
	if full {
		wn := strings.ReplaceAll(Pick(r, kit.Weapon), "{n}", name)
		var base *Item
		if strings.Contains(wn, "pistol") || strings.Contains(wn, "rifle") || strings.Contains(wn, "revolver") || strings.Contains(wn, "laser") || strings.Contains(wn, "handgun") {
			base = g.MakeRanged(tier, r)
		} else {
			base = g.MakeWeapon("longblade", tier, r)
		}
		w := g.MakeUnique(base, wn, fmt.Sprintf("Laid in the grave of %s.", name))
		items = append(items, w)
		orn := buOdd(g, r, tier, "A grave ornament of "+name+".")
		orn.Name = strings.ReplaceAll(Pick(r, kit.Ornament), "{n}", name)
		items = append(items, orn)
		ves := &Item{ID: g.newID(), Kind: "trophy", Name: strings.ReplaceAll(Pick(r, kit.Vessel), "{n}", name), Glyph: '♦', F: CYellow, Value: 45 + tier*25 + r.Int(0, 40), Weight: 2,
			Lore: "A funerary vessel. Scholars pay for these; the law does not like them."}
		buNominal(ves, r)
		buSt(g).Prov[ves.ID] = "Placed at the foot of " + name + "'s bier by the mourners."
		items = append(items, ves)
	}
	if len(items) > 0 {
		cl := buNewClaim(g, s, name, state, rel, true, items)
		for _, it := range items {
			val += it.Value
			g.give(it)
			g.Msg("{{M|Grave goods: " + it.Display() + ".}}")
			_ = cl
		}
	}
	if state > 0 || rel > 0 {
		pen, fpen := 15, 10
		if full {
			pen, fpen = 45, 30
		}
		if state > 0 {
			g.Rep[uwFac(state)] -= pen
			g.compRep(uwFac(state), -pen)
		}
		if rel > 0 {
			g.Rep[fmt.Sprintf("faith:%d", rel)] -= fpen
			g.compRep(fmt.Sprintf("faith:%d", rel), -fpen)
		}
		g.Msg("{{R|Grave robbing: the dead keep faith with the living, and the living remember.}}")
	}
	if full {
		g.Msg("{{R|" + buCurseText[curse] + "}}")
		switch curse {
		case "ghost", "both":
			a := g.Spawn("wight", zn.Level+2, r)
			a.Name, a.Proper = name+"'s shade", true
			a.Desc = "The restless dead of this barrow, come to take back what is theirs."
			a.X, a.Y = g.P.X, g.P.Y
			g.nudge(zn, a)
			zn.Actors = append(zn.Actors, a)
		}
		switch curse {
		case "sick", "both":
			g.P.AddSt("gravesick", 2880)
			g.Msg("{{R|Grave sickness takes hold of you.}}")
		}
	}
	g.gainXP(40 + tier*15)
	buWitnessed(g, val, " (grave robbery)")
	buFirst(g, "robbed", fmt.Sprintf("%s robbed the grave of %s.", g.P.Name, name))
	buAddFind(g, s.Name, fmt.Sprintf("Opened the barrow of %s in %s.", name, g.W.RegionName(s.P)), val)
	g.spend(1000)
}

// ---------------------------------------------------------------- bunkers

func buCodeOf(w *WorldData, s *buSite) string {
	h := HashStr(w.Seed + ":bu:code:" + s.ID)
	return fmt.Sprintf("%04d", h%10000)
}

func buHasCode(g *Game, s *buSite) bool {
	if buSt(g).rec(s.ID).Code {
		return true
	}
	for _, it := range g.P.Inv {
		if it.Extra != nil && it.Extra["bu:code"] == s.Num {
			return true
		}
	}
	return false
}

func buReveal(g *Game, s *buSite, rec *buRec, msg string) {
	rec.Stage = 1
	zn := g.Cur
	zn.Set(rec.X, rec.Y, TInscription)
	kit := buBunkerKitFor(g.Genre)
	zn.Notes[rec.Y*MW+rec.X] = "Under the debris: " + kit.Hatch + ", and " + kit.Lock + ". Step up to it."
	g.Msg("{{Y|" + msg + "}}")
	buFirst(g, "hatch", fmt.Sprintf("%s uncovered a sealed hatch under the rubble.", g.P.Name))
}

// buSearch is called after the `s` key; reveals a hidden hatch nearby.
func buSearch(g *Game) {
	if g.Cur == nil {
		return
	}
	st := buSt(g)
	id := st.ByZone[g.Cur.Key]
	s := buIdx(g.W).byID[id]
	if s == nil || s.Kind != "bunker" {
		return
	}
	rec := st.rec(id)
	if rec.Stage != 0 || !rec.Placed {
		return
	}
	d := maxi(abs(g.P.X-rec.X), abs(g.P.Y-rec.Y))
	dc := dcMedium + 2
	if rec.Known {
		dc -= 4
	}
	if d <= 4 && g.roll2d6Check(g.checkMod("perception")+(4-d), dc) {
		buReveal(g, s, rec, "Searching, you find a hatch plate under the debris!")
	}
}

func buBunkerMenu(g *Game, s *buSite, rec *buRec) {
	kit := buBunkerKitFor(g.Genre)
	var opts []Option
	text := ""
	switch rec.Stage {
	case 1:
		text = fmt.Sprintf("%s, set flush into the ground. %s.", strings.ToUpper(kit.Hatch[:1])+kit.Hatch[1:], strings.ToUpper(kit.Lock[:1])+kit.Lock[1:])
		if buHasCode(g, s) {
			opts = append(opts, Option{"Enter the code from the note.", func() { buBunkerOpen(g, s, rec, "You key in the code. Old relays clunk.") }})
		}
		opts = append(opts, Option{"Hack the lock (Tinkering).", func() {
			buClose(g)
			g.spend(5000)
			if g.roll2d6Check(g.checkMod("tinkering"), dcMedium+1) {
				buBunkerOpen(g, s, rec, "The lock yields to your fingers.")
				return
			}
			g.hurt(g.P, 3+g.Cur.Level/2, nil, "a lock's shock")
			g.Msg("{{R|The lock bites back with a surge, and a distant alarm whines.}}")
		}})
		opts = append(opts, Option{"Force the hatch (Endurance, takes time).", func() {
			buClose(g)
			if !buDigTime(g, 20) {
				return
			}
			if g.roll2d6Check(g.checkMod("endurance"), dcMedium+2) {
				buBunkerOpen(g, s, rec, "The hatch gives with a scream of metal.")
				return
			}
			g.hurt(g.P, 3, nil, "a slipped pry-bar")
			g.Msg("{{R|The bar slips. The hatch holds.}}")
		}})
	case 2:
		text = "The hatch stands open on a ladder shaft and a dark passage."
		opts = append(opts, Option{"Climb down.", func() { buBunkerVault(g, s, rec) }})
	}
	opts = append(opts, Option{"Leave it.", func() { buClose(g) }})
	buOpen(g, s.Name, text, opts)
}

func buBunkerOpen(g *Game, s *buSite, rec *buRec, msg string) {
	buClose(g)
	rec.Stage = 2
	kit := buBunkerKitFor(g.Genre)
	g.Msg("{{Y|" + msg + "}}")
	r := buRng(g, "guard", s.ID)
	lvl := g.Cur.Level + 2
	switch Pick(r, []string{"rad", "gas", "turret", "rad"}) {
	case "rad":
		g.Msg("{{R|" + strings.ToUpper(kit.Hazard[:1]) + kit.Hazard[1:] + ".}}")
		if !g.roll2d6Check(g.checkMod("endurance"), dcMedium) {
			g.hurt(g.P, 3+lvl, nil, "leaking radiation")
			g.P.AddSt("poisoned", 20)
		}
	case "gas":
		g.Msg("{{R|" + strings.ToUpper(kit.Hazard[:1]) + kit.Hazard[1:] + ".}}")
		if !g.roll2d6Check(g.checkMod("endurance"), dcMedium) {
			g.P.AddSt("choking", 6)
			g.hurt(g.P, 2+lvl/2, nil, "a gas leak")
		}
	case "turret":
		a := g.Spawn("turret", lvl+1, r)
		a.Name, a.Proper = kit.Turret, false
		a.X, a.Y = g.P.X, g.P.Y
		g.nudge(g.Cur, a)
		g.Cur.Actors = append(g.Cur.Actors, a)
		g.Msg("{{R|A " + kit.Turret + " swivels toward you out of the dark!}}")
	}
	buFirst(g, "bunker", fmt.Sprintf("%s opened a sealed pre-war cache.", g.P.Name))
}

func buBunkerVault(g *Game, s *buSite, rec *buRec) {
	r := buRng(g, "vault", s.ID)
	kind := Pick(r, buVaultTypes)
	buOpen(g, s.Name, "Below: "+buVaultDesc[kind]+". Dust lies thick; nothing has stirred in generations.", []Option{
		{"Take what you can carry.", func() { buLootBunker(g, s, rec, kind) }},
		{"Leave it.", func() { buClose(g) }},
	})
}

func buLootBunker(g *Game, s *buSite, rec *buRec, kind string) {
	buClose(g)
	rec.Stage, rec.Day = 3, g.Day()
	r := buRng(g, "vloot", s.ID)
	kit := buBunkerKitFor(g.Genre)
	zn := g.Cur
	tier := clampi(zn.Level/3+2, 2, 5)
	var items []*Item
	switch kind {
	case "stores":
		f := g.MakeFood(r, r.Int(4, 8))
		f.Name = kit.Crate
		f.Value = 8
		items = append(items, f, g.MakeWater())
		items = append(items, g.MakeTonic("heal", 1))
	case "armoury":
		for i := 0; i < 2; i++ {
			var it *Item
			if r.Chance(0.6) {
				it = g.MakeRanged(tier, r)
			} else {
				it = g.MakeWeapon(Pick(r, []string{"longblade", "axe", "cudgel"}), tier, r)
			}
			it = g.MakeUnique(it, Pick(r, kit.Arms), "Sealed away before the old collapse; still oiled.")
			items = append(items, it)
		}
		items = append(items, g.MakeAmmo(r.Int(15, 30)))
	case "medbay":
		items = append(items, g.MakeTonic("heal", 3), g.MakeTonic("salve", 2), g.MakeTonic("antidote", 2))
		items[0].Name = kit.Med
	case "archive":
		for i := 0; i < 2; i++ {
			txt := "Rows of dead columns, then a fragment: "
			if other := buNearestOther(g, s); other != nil && i == 0 {
				txt += fmt.Sprintf("\"A second cache lies near %s, parasang (%d,%d).\"", g.W.RegionName(other.P), other.WX, other.WY)
				items = append(items, nil)
				c := g.MakeBook(kit.Core, txt)
				c.Value = 80 + tier*40
				c.Extra = map[string]int{"bu:hatch": other.Num}
				buNominal(c, r)
				buSt(g).Prov[c.ID] = "Recovered from the " + kit.Place + " at " + g.W.RegionName(s.P) + "."
				items[len(items)-1] = c
				continue
			}
			c := g.MakeBook(kit.Core, txt+"personnel lists, ration tallies, a lullaby recorded twice.")
			c.Value = 60 + tier*30
			buNominal(c, r)
			buSt(g).Prov[c.ID] = "Recovered from the " + kit.Place + " at " + g.W.RegionName(s.P) + "."
			items = append(items, c)
		}
	case "relicvault":
		for i := 0; i < 2; i++ {
			o := g.MakeOddity(r, tier+1)
			o.Name = kit.Relic
			o.Lore = "Relic technology. A scholar might know what it does and what it is worth."
			buNominal(o, r)
			buSt(g).Prov[o.ID] = "Cached in the " + kit.Place + " at " + g.W.RegionName(s.P) + " before the fall."
			items = append(items, o)
		}
	}
	coin := 60 + r.Int(0, 80) + zn.Level*10
	g.P.Gold += coin
	g.Msg(fmt.Sprintf("{{Y|%d %s from a strongbox.}}", coin, g.Sk.Currency))
	val := coin
	for _, it := range items {
		val += it.Value * maxi(it.Count, 1)
		g.give(it)
		g.Msg("{{M|Found: " + it.Display() + ".}}")
	}
	if burg := g.W.burgByID[s.Burg]; burg != nil && burg.State > 0 {
		if sd := g.W.stateByID[burg.State]; sd != nil && sd.Alive && r.Chance(0.35) {
			var tag []*Item
			for _, it := range items {
				if it.Kind != "food" && it.Kind != "water" && it.Kind != "tonic" && it.Kind != "ammo" {
					tag = append(tag, it)
				}
			}
			if len(tag) > 0 {
				buNewClaim(g, s, s.Name, burg.State, 0, false, tag)
			}
		}
	}
	g.gainXP(60 + tier*20)
	buAddFind(g, s.Name, "Looted the "+kit.Place+": "+buVaultDesc[kind]+".", val)
	buFirst(g, "vault", fmt.Sprintf("%s emptied a pre-war vault at %s.", g.P.Name, g.W.RegionName(s.P)))
	g.spend(1000)
}

func buNearestOther(g *Game, s *buSite) *buSite {
	var best *buSite
	bd := 1 << 30
	st := buSt(g)
	for _, o := range buIdx(g.W).sites {
		if o == s || o.Kind != "bunker" {
			continue
		}
		if r := st.Rec[o.ID]; r != nil && r.Stage >= 3 {
			continue
		}
		if d := maxi(abs(o.WX-s.WX), abs(o.WY-s.WY)); d < bd {
			best, bd = o, d
		}
	}
	return best
}

// notes and recordings lying in ruins near a bunker
func buPlaceNotes(g *Game, zn *Zone, ix *buIndex) {
	if !buBunkerGenre(g.Genre) || zn.Kind != "ruin" {
		return
	}
	var best *buSite
	bd := 9
	for _, s := range ix.sites {
		if s.Kind != "bunker" {
			continue
		}
		if d := maxi(abs(s.WX-zn.WX), abs(s.WY-zn.WY)); d < bd && d > 0 || d == 0 && best == nil && bd == 9 {
			best, bd = s, d
		}
	}
	if best == nil {
		return
	}
	r := NewRng(g.Seed + ":bu:note:" + zn.Key)
	kit := buBunkerKitFor(g.Genre)
	if r.Chance(0.45) {
		if x, y, ok := zn.FreeSpot(r, nil); ok {
			txt := fmt.Sprintf(Pick(r, kit.NoteTxt), buCodeOf(g.W, best))
			it := &Item{ID: g.newID(), Kind: "letter", Name: kit.Note, Glyph: '?', F: CYellow, Value: 10, X: x, Y: y, Ident: true,
				Text: txt + fmt.Sprintf("\n\n(It refers to the %s at %s.)", kit.Place, g.W.RegionName(best.P)), Extra: map[string]int{"bu:code": best.Num}}
			zn.Items = append(zn.Items, it)
		}
	}
	if r.Chance(0.25) {
		if x, y, ok := zn.FreeSpot(r, nil); ok {
			it := g.MakeBook("scratched recording", fmt.Sprintf("A voice, half-eaten by noise: \"...under the rubble, %s of the old square, near %s. Hatch's flush. Count three steps from the broken pillar.\"", bhDir(best.WX-zn.WX, best.WY-zn.WY), g.W.RegionName(best.P)))
			it.X, it.Y, it.Ident = x, y, true
			it.Extra = map[string]int{"bu:hatch": best.Num}
			zn.Items = append(zn.Items, it)
		}
	}
}

// ---------------------------------------------------------------- pirate hoards and charts

func buMapText(g *Game, s *buSite, part int) string {
	st := buSt(g)
	out := fmt.Sprintf("Half of a chart, torn through by an angry hand. Piece %d of %d. A name is inked in the corner: %s.", part, s.Parts, s.Captain)
	lvl := part
	if s.Parts == 2 && part == 2 {
		lvl = 3
	}
	if s.Parts == 2 && part == 1 {
		lvl = 1
	}
	region := g.W.RegionName(s.P)
	switch lvl {
	case 1:
		out += fmt.Sprintf("\n\nA red cross rides the coast of %s.", region)
		if s.Parts == 2 {
			lvl = 2
		}
	}
	if lvl >= 2 && (s.Parts == 2 && part == 1 || s.Parts == 3 && part == 2) {
		b, d := bhNearBurg(g, s.WX, s.WY)
		if b != nil {
			bx, by := bhPxy(g, b.P)
			out += fmt.Sprintf("\n\nLandmark: about %d parasangs %s of %s.", d, bhDir(s.WX-bx, s.WY-by), b.Name)
		}
		if s.Isle {
			out += " The mark sits on a small island."
		}
	}
	if lvl == 3 {
		out += fmt.Sprintf("\n\nExact: parasang (%d,%d), in %s of it, %s.", s.WX, s.WY, bhThird(s.ZX, s.ZY), Pick(NewRng(g.Seed+":bu:where:"+s.ID), buHoardWhere))
		if s.Parts == 2 {
			out += " Stand on the spot and press D to dig."
		}
	}
	if next := buHolderHintText(g, s, part%s.Parts+1); next != "" {
		out += "\n\nMargin: the other piece " + next + "."
	}
	if buMaskFull(st, s) {
		out += fmt.Sprintf("\n\n{{G|Together the pieces point to parasang (%d,%d), %s. Stand on the spot and dig (D).}}", s.WX, s.WY, bhThird(s.ZX, s.ZY))
	}
	return out
}

func buMapItem(g *Game, s *buSite, part int) *Item {
	r := buRng(g, "mapname", s.ID, fmt.Sprint(part))
	it := &Item{ID: g.newID(), Kind: "map", Name: fmt.Sprintf("%s (piece %d of %d)", Pick(r, bhMapNames), part, s.Parts), Glyph: '?', F: CYellow, Value: 60 + 30*part, Ident: true,
		Extra: map[string]int{"buc": s.Num, "buh": part}, Lore: "Half a pirate's chart. Read it, and find the other pieces."}
	it.Text = buMapText(g, s, part)
	return it
}

// buRead reads a split chart: call from useItem case "map" before bhRead.
func buRead(g *Game, it *Item) bool {
	if it == nil || it.Extra == nil || it.Extra["buc"] == 0 {
		return false
	}
	s := buIdx(g.W).byNum[it.Extra["buc"]]
	if s == nil {
		return false
	}
	it.Text = buMapText(g, s, it.Extra["buh"])
	g.showPopup(it.Name, it.Text)
	return true
}

func buPlaceParts(g *Game, zn *Zone, p int, ix *buIndex) {
	if zn.ZX != 1 || zn.ZY != 1 {
		return
	}
	for _, s := range ix.sites {
		if s.Kind != "hoard" {
			continue
		}
		for k, h := range s.Holders {
			if h.P != p || h.Kind == "tavern" {
				continue
			}
			part := k + 1
			r := NewRng(fmt.Sprintf("%s:bu:part:%s:%d", g.Seed, s.ID, part))
			x, y, ok := zn.FreeSpot(r, nil)
			if !ok {
				continue
			}
			it := buMapItem(g, s, part)
			if h.Kind == "rival" {
				a := g.Spawn("bandit", zn.Level+3, r)
				a.Name, a.Proper, a.Title = Pick(r, []string{"Dagget", "Moll", "Tarn", "Orsa", "Bram", "Kessa"})+" "+Pick(r, buPirateEpithets), true, "first mate"
				a.Desc = "Once " + s.Captain + "'s first mate. They kept the chart."
				a.X, a.Y, a.HomeX, a.HomeY = x, y, x, y
				a.Inv = append(a.Inv, it)
				a.Set("buhold")
				zn.Actors = append(zn.Actors, a)
			} else {
				it.X, it.Y = x, y
				it.Name = "ship's logbook with " + it.Name
				zn.Items = append(zn.Items, it)
				zn.Set(x, y, TBones)
			}
		}
	}
}

func buHoardMenu(g *Game, s *buSite, rec *buRec) {
	buOpen(g, s.Name, fmt.Sprintf("The joined chart says this is the place: %s. The ground here has been disturbed, once.", Pick(NewRng(g.Seed+":bu:where:"+s.ID), buHoardWhere)), []Option{
		{"Dig.", func() { buHoardDig(g, s, rec) }},
		{"Leave it.", func() { buClose(g) }},
	})
}

func buHoardDig(g *Game, s *buSite, rec *buRec) {
	buClose(g)
	if !buDigTime(g, 40) {
		return
	}
	rec.Stage, rec.Day = 3, g.Day()
	r := buRng(g, "hoard", s.ID)
	zn := g.Cur
	lvl := zn.Level + 3
	tier := clampi(lvl/3+1, 1, 5)
	g.Msg("{{Y|Your spade rings on iron: " + s.Captain + "'s strongbox!}}")
	coin := 300 + r.Int(0, 400) + lvl*30
	g.P.Gold += coin
	g.Msg(fmt.Sprintf("{{Y|%d %s.}}", coin, g.Sk.Currency))
	val := coin
	var items []*Item
	base := g.MakeWeapon("longblade", tier, r)
	items = append(items, g.MakeUnique(base, s.Captain+"'s "+Pick(r, []string{"cutlass", "boarding blade", "last word"}), "Buried with "+s.Captain+"'s plunder."))
	for i := 0; i < 2; i++ {
		items = append(items, buOdd(g, r, tier, "Plundered by "+s.Captain+" from a merchantman."))
	}
	items = append(items, g.Loot(r, tier))
	for _, it := range items {
		val += it.Value
		g.give(it)
		g.Msg("{{M|Plunder: " + it.Display() + ".}}")
	}
	if s.State > 0 && r.Chance(0.6) {
		buNewClaim(g, s, "the plunder of "+s.Captain, s.State, 0, false, items)
	}
	switch Pick(r, buHoardGuards) {
	case "crabs":
		for i := 0; i < 2; i++ {
			a := g.Spawn("crab", lvl, r)
			a.X, a.Y = g.P.X, g.P.Y
			g.nudge(zn, a)
			zn.Actors = append(zn.Actors, a)
		}
		g.Msg("{{R|Giant crabs, nesting in the hoard, swarm out!}}")
	case "trap":
		if g.roll2d6Check(g.checkMod("perception"), dcMedium) {
			g.Msg("{{G|You notice the tripwire to the powder keg and cut it.}}")
		} else {
			g.hurt(g.P, 6+lvl, nil, "a booby-trapped hoard")
		}
	case "tide":
		if !g.roll2d6Check(g.checkMod("acrobatics"), dcMedium) {
			g.hurt(g.P, 3+lvl/2, nil, "the tide")
			g.Msg("{{R|The sea rushes into the hole and drags at you.}}")
		}
	case "ghost":
		a := g.Spawn("skeleton", lvl, r)
		a.Name, a.Proper = s.Captain+"'s ghost", true
		a.X, a.Y = g.P.X, g.P.Y
		g.nudge(zn, a)
		zn.Actors = append(zn.Actors, a)
		g.Msg("{{R|The captain's ghost rises to guard the hoard.}}")
	}
	g.gainXP(80 + tier*20)
	buAddFind(g, s.Name, "Dug up the hoard of "+s.Captain+" near "+g.W.RegionName(s.P)+".", val)
	buFirst(g, "hoard", fmt.Sprintf("%s unearthed the hoard of %s.", g.P.Name, s.Captain))
	g.spend(1000)
}

// ---------------------------------------------------------------- dig hook

// buDig is the D handler's first stop: it returns true when it handled the dig.
func (g *Game) buDig(x, y int) bool {
	zn := g.Cur
	if zn == nil || g.Dead || zn.Z != 0 {
		return false
	}
	st := buSt(g)
	id := st.ByZone[zn.Key]
	s := buIdx(g.W).byID[id]
	if s == nil {
		return false
	}
	rec := st.rec(id)
	if !rec.Placed || rec.Stage >= 3 {
		return false
	}
	d := maxi(abs(x-rec.X), abs(y-rec.Y))
	switch s.Kind {
	case "hoard":
		if buMaskFull(st, s) && d == 0 {
			buHoardDig(g, s, rec)
			return true
		}
		if buMaskFull(st, s) && d <= 12 {
			if !buDigTime(g, 15) {
				return true
			}
			g.Msg("You dig a hole and find only sand and shell.")
			g.Msg(fmt.Sprintf("{{c|The joined chart seems to point about %d paces %s.}}", d, bhDir(rec.X-x, rec.Y-y)))
			return true
		}
	case "barrow":
		if rec.Stage == 0 && (maxi(abs(x-rec.EX), abs(y-rec.EY)) <= 1 || d <= 1) {
			buBarrowBreak(g, s, rec)
			return true
		}
	case "bunker":
		if rec.Stage == 0 && d <= 2 {
			if !buDigTime(g, 25) {
				return true
			}
			buReveal(g, s, rec, "Clearing the rubble, you uncover a hatch plate!")
			return true
		}
	}
	return false
}

// buBump is for interact(): bumping the revealed hatch or the barrow stone opens the site menu.
func (g *Game) buBump(x, y int) bool {
	zn := g.Cur
	if zn == nil || zn.Z != 0 {
		return false
	}
	st := buSt(g)
	id := st.ByZone[zn.Key]
	s := buIdx(g.W).byID[id]
	if s == nil {
		return false
	}
	rec := st.Rec[id]
	if rec == nil || rec.Stage >= 3 || x != rec.X || y != rec.Y {
		return false
	}
	switch s.Kind {
	case "barrow":
		buBarrowMenu(g, s, rec)
	case "bunker":
		if rec.Stage == 0 {
			return false
		}
		buBunkerMenu(g, s, rec)
	default:
		return false
	}
	return true
}

// ---------------------------------------------------------------- the ground shifts

func buEnter(g *Game, zn *Zone) {
	if !buPlain(zn) || !g.W.In(zn.WX, zn.WY) {
		return
	}
	buPurge(g, zn)
	buShift(g, zn)
	buHeirs(g, zn)
}

func buPurge(g *Game, zn *Zone) {
	day := g.Day()
	n := 0
	out := zn.Items[:0]
	for _, it := range zn.Items {
		if it.Extra != nil && it.Extra["bu:exp"] > 0 && day >= it.Extra["bu:exp"] {
			n++
			continue
		}
		out = append(out, it)
	}
	zn.Items = out
	if n > 0 {
		g.Msg("{{K|The shifting ground has taken something back since you were last here.}}")
	}
}

func buRecentSand(g *Game, p int) bool {
	for k := 0; k < 18; k++ {
		t := g.Turn - k*wxBucket
		if t >= 0 {
			if wk := g.W.WeatherAt(p, t).Kind; wk != nil && wk.ID == "sandstorm" {
				return true
			}
		}
	}
	return false
}

func buShift(g *Game, zn *Zone) {
	w := g.W
	p := w.Idx(zn.WX, zn.WY)
	b := w.Biome[p]
	yr := w.Year + g.Day()/YearDays
	sea := g.Season()
	code := yr*4 + sea + 1
	st := buSt(g)
	if st.Shift[zn.Key] == code {
		return
	}
	st.Shift[zn.Key] = code
	r := NewRng(fmt.Sprintf("%s:bu:shift:%s:%d", g.Seed, zn.Key, code))
	wet := 0
	for i := range zn.T {
		if zn.T[i] == TShallow || zn.T[i] == TDeep {
			wet++
		}
	}
	kind := ""
	switch {
	case (b == 12 || wet >= 40) && (sea == 0 || sea == 2) && r.Chance(0.22):
		kind = "flood"
	case (b == 1 || b == 2) && buRecentSand(g, p) && r.Chance(0.4):
		kind = "dune"
	case tgGlacial(w, p) && sea <= 1 && r.Chance(0.25):
		kind = "calve"
	case r.Chance(0.012):
		kind = "quake"
	}
	if kind == "" {
		return
	}
	x, y, ok := zn.FreeSpot(r, func(x, y int) bool {
		switch kind {
		case "flood":
			for _, d := range dirs8 {
				if zn.WaterAt(x+d[0], y+d[1]) > 0 {
					return true
				}
			}
			return false
		case "dune":
			return zn.At(x, y) == TSand
		case "calve":
			t := zn.At(x, y)
			return t == TGlacier || t == TSnow || t == TIce
		}
		return true
	})
	if !ok {
		x, y, ok = zn.FreeSpot(r, nil)
		if !ok {
			return
		}
	}
	tier := clampi(zn.Level/3+1, 1, 4)
	it := buOdd(g, r, tier, "Uncovered by the "+map[string]string{"flood": "flood", "dune": "moving dunes", "quake": "quake", "calve": "calving ice"}[kind]+" near "+w.RegionName(p)+".")
	it.Name = Pick(r, buShiftItemNames[kind])
	it.X, it.Y = x, y
	it.Extra["bu:exp"] = g.Day() + (1+r.Int(0, 2))*YearDays
	zn.Items = append(zn.Items, it)
	var tab []string
	switch kind {
	case "flood":
		tab = buFloodMsg
	case "dune":
		tab = buDuneMsg
	case "calve":
		tab = buCalveMsg
	default:
		tab = buQuakeMsg
	}
	g.Msg("{{c|" + Pick(r, tab) + "}}")
}

// ---------------------------------------------------------------- claims, heirs and returning goods

func buNewClaim(g *Game, s *buSite, name string, state, rel int, grave bool, items []*Item) *buClaim {
	st := buSt(g)
	r := buRng(g, "claim", s.ID)
	val := 0
	cl := &buClaim{ID: st.next(), Site: s.ID, Name: name, State: state, Rel: rel, Grave: grave, Due: g.Day() + 4 + r.Int(0, 8)}
	if r.Chance(0.5) {
		cl.Kind = "heir"
	} else {
		cl.Kind = "hunt"
	}
	for _, it := range items {
		if it.Extra == nil {
			it.Extra = map[string]int{}
		}
		it.Extra["bu:claim"] = cl.ID
		g.uwMarkStolen(it, state)
		val += maxi(it.Extra["bu:nom"], it.Value)
	}
	cl.Value = val
	if state > 0 {
		st.Claims = append(st.Claims, cl)
	}
	return cl
}

func buClaimByID(st *buState, id int) *buClaim {
	for _, c := range st.Claims {
		if c.ID == id {
			return c
		}
	}
	return nil
}

func buDaily(g *Game, st *buState) {
	day := g.Day()
	for _, c := range st.Claims {
		if c.Done || day < c.Due {
			continue
		}
		if c.Kind == "hunt" || c.Spawned && day > c.Due+20 {
			c.Done = true
			g.uwAddBounty(c.State, 60+c.Value/6, " (heirs of "+c.Name+")")
			g.Msg("{{R|Word has spread: the heirs of " + c.Name + " have set a price on your head.}}")
			bhMaybeHunters(g, g.Cur, false)
		}
	}
	// sicknesses and finds expire on their own
}

func buHeirs(g *Game, zn *Zone) {
	if zn.Kind != "town" {
		return
	}
	bd := g.W.burgByID[zn.Burg]
	if bd == nil {
		return
	}
	st := buSt(g)
	day := g.Day()
	for _, c := range st.Claims {
		if c.Done || c.Spawned || c.Kind != "heir" || day < c.Due || c.State != bd.State {
			continue
		}
		r := NewRng(fmt.Sprintf("%s:bu:heir:%d", g.Seed, c.ID))
		x, y, ok := zn.FreeSpot(r, func(x, y int) bool {
			return abs(x-g.P.X) < 12 && abs(y-g.P.Y) < 7 && (abs(x-g.P.X) > 3 || abs(y-g.P.Y) > 3)
		})
		if !ok {
			continue
		}
		a := g.makeNPC("notable", bd, g.W.cultByID[bd.Culture], r)
		a.Title = "advocate of the heirs"
		a.Desc = "A grim person in a good coat, carrying a writ about " + c.Name + "."
		a.AI, a.X, a.Y, a.HomeX, a.HomeY = "still", x, y, x, y
		a.Set("buheir")
		zn.Actors = append(zn.Actors, a)
		st.Heirs[a.ID] = c.ID
		c.Spawned = true
		g.Msg("{{R|A stranger in a good coat watches you from across the street. They know what you took.}}")
		return
	}
}

// buGoods lists the claimed items in the pack for a claim id (0 = any).
func buGoods(g *Game, claim int) []*Item {
	var out []*Item
	for _, it := range g.P.Inv {
		if it.Extra != nil && it.Extra["bu:claim"] > 0 && (claim == 0 || it.Extra["bu:claim"] == claim) && g.P.Equipped(it) == nil {
			out = append(out, it)
		}
	}
	return out
}

func buRemove(g *Game, its []*Item) {
	for _, it := range its {
		for i, o := range g.P.Inv {
			if o == it {
				g.P.Inv = append(g.P.Inv[:i], g.P.Inv[i+1:]...)
				break
			}
		}
	}
}

func buReturn(g *Game, a *Actor, c *buClaim) {
	its := buGoods(g, c.ID)
	if len(its) == 0 {
		g.talkMenu("\"You carry nothing of theirs.\"")
		return
	}
	val := 0
	for _, it := range its {
		val += maxi(it.Extra["bu:nom"], it.Value)
	}
	buRemove(g, its)
	c.Done = true
	reward := val / 4
	g.P.Gold += reward
	g.Rep[uwFac(c.State)] += 60
	g.compRep(uwFac(c.State), 60)
	if c.Rel > 0 {
		g.Rep[fmt.Sprintf("faith:%d", c.Rel)] += 30
	}
	g.gainXP(50)
	buFirst(g, "returned", fmt.Sprintf("%s returned stolen grave goods to their rightful heirs.", g.P.Name))
	buAddFind(g, "Returned: "+c.Name, "Returned to "+g.realmName(c.State)+" for a reward of "+fmt.Sprint(reward)+".", -val)
	g.talkMenu(fmt.Sprintf("\"Honest of you. The realm remembers. Here is %d %s for your trouble.\"", reward, g.Sk.Currency))
}

// ---------------------------------------------------------------- appraisal, collectors, rumours, talk

func buAppraiser(a *Actor) bool {
	switch a.Role {
	case "bookseller", "exotic", "chronicler", "naturalist", "geneticist", "fence":
		return true
	}
	return uwIsFence(a)
}

func buTalk(g *Game, a *Actor) []Option {
	if a == nil || g.Cur == nil {
		return nil
	}
	st := buSt(g)
	var out []Option
	// heir
	if a.Has("buheir") {
		if c := buClaimByID(st, st.Heirs[a.ID]); c != nil && !c.Done {
			share := maxi(20, c.Value*35/100)
			out = append(out, Option{fmt.Sprintf("{{y|Settle the claim: pay the heirs' share. (%d %s)}}", share, g.Sk.Currency), func() {
				if g.P.Gold < share {
					g.talkMenu("\"The writ names a sum. Come back with it.\"")
					return
				}
				g.P.Gold -= share
				c.Done = true
				g.Rep[uwFac(c.State)] += 10
				g.talkMenu("\"The heirs accept. The matter is closed.\" They fold the writ away.")
			}})
			if len(buGoods(g, c.ID)) > 0 {
				out = append(out, Option{"{{G|Return the goods you took.}}", func() { buReturn(g, a, c) }})
			}
			out = append(out, Option{"{{R|Refuse. Tell them to take it to the courts.}}", func() {
				c.Done = true
				g.uwAddBounty(c.State, 60+c.Value/6, " (refused the heirs of "+c.Name+")")
				g.talkMenu("\"Then you shall meet the court's hunters.\" They walk away.")
				a.AI = "town"
			}})
		}
	}
	// restitution at elders and priests of the realm
	if a.Role == "elder" || a.Role == "priest" || a.Role == "notable" && !a.Has("buheir") {
		seen := map[int]bool{}
		for _, it := range buGoods(g, 0) {
			id := it.Extra["bu:claim"]
			if seen[id] {
				continue
			}
			seen[id] = true
			if c := buClaimByID(st, id); c != nil && !c.Done && c.State == a.State {
				out = append(out, Option{"{{G|Return what I took from " + c.Name + ".}}", func() { buReturn(g, a, c) }})
			}
		}
	}
	// tavern holder of a chart piece
	if a.Role == "villager" && g.Cur.Kind == "town" {
		if o := buTavernOffer(g, a, st); o != nil {
			out = append(out, *o)
		}
	}
	// rumours of hoards
	if g.Cur.Kind == "town" && (a.Role == "elder" || a.Role == "merchant" || a.Role == "exotic" || a.Role == "notable" || a.Role == "villager") {
		if bd := g.W.burgByID[g.Cur.Burg]; bd != nil && bd.Port {
			out = append(out, Option{"{{y|Ask about sea-captains' lost hoards.}}", func() { buRumor(g, a, bd) }})
		}
	}
	if buAppraiser(a) {
		out = append(out, Option{"{{O|Appraise my finds.}}", func() { buAppraiseMenu(g, a) }})
		out = append(out, Option{"{{O|Sell relics and finds to the collector.}}", func() { buSellMenu(g, a) }})
		if len(st.Finds) > 0 {
			out = append(out, Option{"{{c|Show my ledger of finds.}}", func() { buFinds(g) }})
		}
	}
	return out
}

func buTavernOffer(g *Game, a *Actor, st *buState) *Option {
	bd := g.W.burgByID[g.Cur.Burg]
	if bd == nil {
		return nil
	}
	ix := buIdx(g.W)
	for _, s := range ix.sites {
		if s.Kind != "hoard" {
			continue
		}
		for k, h := range s.Holders {
			if h.Kind != "tavern" || h.Burg != bd.ID {
				continue
			}
			key := fmt.Sprintf("%d:%d", s.Num, k+1)
			if st.Given[key] {
				continue
			}
			if id, ok := st.Tavern[key]; ok && id != a.ID {
				continue
			}
			st.Tavern[key] = a.ID
			part := k + 1
			price := 90 + g.P.Level*12
			return &Option{fmt.Sprintf("{{y|\"They say you drink with half a sea-chart.\" (%d %s)}}", price, g.Sk.Currency), func() {
				if g.P.Gold < price && !g.roll2d6Check(g.checkMod("persuasion"), dcMedium+2) {
					g.talkMenu("\"Not for talk. Not for less.\"")
					return
				}
				if g.P.Gold >= price {
					g.P.Gold -= price
				}
				st.Given[key] = true
				it := buMapItem(g, s, part)
				g.give(it)
				g.talkMenu("They slide you a greasy scrap: " + it.Name + ". \"Never saw you.\"")
			}}
		}
	}
	return nil
}

func buRumor(g *Game, a *Actor, bd *BurgData) {
	st := buSt(g)
	ix := buIdx(g.W)
	var best *buSite
	bdist := 1 << 30
	for _, s := range ix.sites {
		if s.Kind != "hoard" || st.rec(s.ID).Stage >= 3 {
			continue
		}
		if d := maxi(abs(s.WX-bd.P%g.W.W), abs(s.WY-bd.P/g.W.W)); d < bdist {
			best, bdist = s, d
		}
	}
	if best == nil || bdist > 40 {
		g.talkMenu("\"Tall tales, mostly. Nothing that floats.\"")
		return
	}
	text := fmt.Sprintf("\"%s, they say, buried a hoard near %s before the tide took them. The chart was torn in %d. ", best.Captain, g.W.RegionName(best.P), best.Parts)
	if h := buHolderHintText(g, best, 1); h != "" {
		text += "The first piece " + h + ". "
	}
	text += "Find all the pieces and a spade will do the rest.\""
	st.Rumor[best.ID] = true
	g.talkMenu(text)
}

func buAppraiseMenu(g *Game, a *Actor) {
	var opts []Option
	for _, it := range g.P.Inv {
		if it.Extra == nil || it.Extra["bu:app"] == 0 {
			continue
		}
		it := it
		grave := it.Extra["bu:claim"] > 0 || it.Extra["stolen"] > 0
		fee := maxi(5, it.Extra["bu:nom"]/20)
		opts = append(opts, Option{fmt.Sprintf("%s (%d %s)", it.Display(), fee, g.Sk.Currency), func() {
			if grave && a.Role != "exotic" && !uwIsFence(a) {
				buAppraiseMenu2(g, a, "\"Where did this come from? I will not put my name to a grave-robber's goods. Try someone less careful.\"")
				return
			}
			if g.P.Gold < fee {
				buAppraiseMenu2(g, a, "\"My time is not free.\"")
				return
			}
			g.P.Gold -= fee
			r := buRng(g, "app", fmt.Sprint(it.ID))
			nom, tv := it.Extra["bu:nom"], it.Extra["bu:tv"]
			it.Value = maxi(1, nom*tv/100)
			it.Ident = true
			delete(it.Extra, "bu:app")
			story := buSt(g).Prov[it.ID]
			it.Lore = strings.TrimSpace(it.Lore + " " + story)
			it.Extra["bu:find"] = 1
			verdict := "about what you would hope"
			switch {
			case tv < 90:
				verdict = "less than you hoped"
			case tv > 200:
				verdict = "a remarkable piece, worth far more than it looks"
			case tv > 130:
				verdict = "better than it looks"
			}
			buAddFind(g, it.Name, story, it.Value)
			g.gainXP(15)
			buAppraiseMenu2(g, a, fmt.Sprintf("%s %s\n\n\"%s. I put it at %d %s.\"\n%s", a.Name, Pick(r, buAppraiseLine), strings.ToUpper(verdict[:1])+verdict[1:], it.Value, g.Sk.Currency, story))
		}})
	}
	if len(opts) == 0 {
		g.talkMenu("\"You have nothing I could put a number on.\"")
		return
	}
	opts = append(opts, Option{"Done.", func() { g.talkMenu("\"Come again.\"") }})
	buOpen(g, "Appraisal", "Unusual items: their worth is hidden until a scholar, collector or fence has looked.", opts)
	g.Talk = a
}

func buAppraiseMenu2(g *Game, a *Actor, msg string) {
	g.Msg(Plain(strings.SplitN(msg, "\n", 2)[0]))
	g.showPopup("Appraisal", msg)
}

func buSellMenu(g *Game, a *Actor) {
	var opts []Option
	n := 0
	for _, it := range g.P.Inv {
		if it.Extra != nil && it.Extra["bu:app"] > 0 {
			continue
		}
		relic := it.SimArt > 0
		find := it.Extra != nil && it.Extra["bu:find"] > 0
		if !relic && !find {
			continue
		}
		if it.Extra != nil && (it.Extra["bu:claim"] > 0 || it.Extra["stolen"] > 0) && a.Role != "exotic" && !uwIsFence(a) {
			continue
		}
		if g.P.Equipped(it) != nil || n >= 12 {
			continue
		}
		n++
		it := it
		pay := it.Value * 6 / 5
		if relic {
			pay = it.Value * 3 / 2
		}
		pay = maxi(1, pay)
		opts = append(opts, Option{fmt.Sprintf("%s: %d %s", it.Display(), pay, g.Sk.Currency), func() {
			if a.Gold < pay {
				g.talkMenu("\"I do not have that much by me today.\"")
				return
			}
			a.Gold -= pay
			g.P.Gold += pay
			buRemove(g, []*Item{it})
			a.Inv = append(a.Inv, it)
			g.gainXP(10)
			g.talkMenu(fmt.Sprintf("\"A fine addition.\" They count out %d %s.", pay, g.Sk.Currency))
		}})
	}
	if len(opts) == 0 {
		g.talkMenu("\"Bring me relics of the old kings, or things a scholar has already named.\"")
		return
	}
	opts = append(opts, Option{"Done.", func() { g.talkMenu("\"Another time.\"") }})
	buOpen(g, "Collector", fmt.Sprintf("Their purse: %d %s. Relics and appraised finds fetch more here than at an honest market.", a.Gold, g.Sk.Currency), opts)
	g.Talk = a
}

// buFinds shows the ledger of great finds (hook it to a key if wanted; also offered by appraisers).
func buFinds(g *Game) {
	st := buSt(g)
	if len(st.Finds) == 0 {
		g.showPopup("Finds", "Nothing yet. Barrows, bunkers, hoards and odd things the land gives up will be written here.")
		return
	}
	var b strings.Builder
	total := 0
	for i := len(st.Finds) - 1; i >= 0 && i > len(st.Finds)-16; i-- {
		f := st.Finds[i]
		fmt.Fprintf(&b, "{{Y|%s}} (day %d, year %d)\n%s", f.Title, f.Day, f.Year, f.Text)
		if f.Value > 0 {
			fmt.Fprintf(&b, " Worth %d.", f.Value)
			total += f.Value
		}
		b.WriteString("\n\n")
	}
	b.WriteString(fmt.Sprintf("{{c|Great finds: %d. Total recorded value: %d %s.}}", len(st.Finds), total, g.Sk.Currency))
	g.showPopup("Finds", b.String())
}

func buKill(g *Game, d, src *Actor) {
	if d == nil || !d.Has("buhold") {
		return
	}
	for _, it := range d.Inv {
		if it.Kind == "map" && it.Extra != nil && it.Extra["buc"] > 0 {
			g.Msg("{{Y|The first mate's chart lies in the dust.}}")
		}
	}
}
