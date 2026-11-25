package engine

import (
	"fmt"
	"math"
	"sort"
	"strings"
)

// Monster ecology (prefix mg). A planet-level simulation of herds and predators on the day clock, at the
// scale of regions of mgCell x mgCell parasangs. Each region keeps six population indexes (grazing herds,
// small game, birds, fish, predators, apex hunters; 100 = a normal year) built from the bestiary read-only
// (bsAll groups, diets, tempers, habitats). Every week: logistic growth against a seasonal carrying capacity
// (biome, climate, a good or lean forage year), predation (predators eat herds and game), and the lag
// "more prey this season, more predators next season". In spring and autumn the herds migrate between
// summer and winter ranges toward the neighbouring region with better seasonal capacity; the window is
// early or late by up to six days each year. Birds pass over and fish run in rivers in fixed windows.
// Players see it as: extra or fewer wild groups placed by a zone hook (side rng seed:mg:zoneKey:season,
// append only; refreshed on entering when the season turned), farmers and herders talking about the herds
// (talk option, mgLine for barks), grain and meat prices (mgPriceMod, mgPriceFor), small raids in lean
// years (a flagged group that is removed afterwards) and DENS: a den in a wild zone holds a named alpha, a
// brood and sometimes pack mates. A den can be fought, poisoned, burned or emptied with bait; the next
// season something else, or a rival pack, moves in, with tracks, remains and a farmers' notice stone.
// Kills of apex predators and den clearings change the numbers. Screen mode "mg" (palette "Wildlife of the
// land"), den menu mode "mgd" (palette "Deal with the den"). Rumours go to mgRumourSink when set.
// State g.ext("mg"), planet-local ("mg" in unLocalKeys). Existing systems (bestiary bs, nests nt, frontier
// ecology ef, hunting hn) are only read or left alone.

func init() {
	unLocalKeys = append(unLocalKeys, "mg")
	zoneHooks = append(zoneHooks, mgZone)
	enterHooks = append(enterHooks, mgEnter)
	turnHooks = append(turnHooks, mgTurn)
	killHooks = append(killHooks, mgKill)
	talkHooks = append(talkHooks, mgTalk)
	modeKeys["mg"] = mgKeys
	modeDraws["mg"] = func(g *Game) { g.mgDraw() }
	modeKeys["mgd"] = mgdKeys
	modeDraws["mgd"] = func(g *Game) { g.mgdDraw() }
	paEntries = append(paEntries,
		paEntry{Name: "Wildlife of the land", Keywords: "herds predators migration dens animals ecology wildlife population seasons", Cat: "Screen", Label: "screen",
			Func: func(g *Game) { g.mg().tab, g.mg().sel = 0, 0; g.Mode = "mg" }},
		paEntry{Name: "Deal with the den", Keywords: "den lair brood alpha poison burn lure bait clear", Cat: "Action", Label: "den",
			Cond: func(g *Game) bool { return g.mgDenNear() != nil },
			Func: func(g *Game) { g.mg().denSel = 0; g.Mode = "mgd" }})
}

// mgRumourSink, when set, hears about den clearings, new tenants, raids and early or late herds.
var mgRumourSink func(g *Game, kind, subject, text string)

const mgCell = 6

type mgSpW struct {
	S *bsSp
	W float64
}

type mgReg struct {
	ID, RX, RY, Land, Biome, Temp, Height, River, Roads, Towns int
	Coast                                                      bool
	CX, CY                                                     int
	Name                                                       string
	Nb                                                         []int
	sp                                                         [mgGuilds][]mgSpW
	spOK                                                       bool
}

type mgIndex struct {
	cols, rows int
	regs       []*mgReg
}

var mgCache = map[*WorldData]*mgIndex{}

func mgIdx(w *WorldData) *mgIndex {
	if ix := mgCache[w]; ix != nil {
		return ix
	}
	cols, rows := (w.W+mgCell-1)/mgCell, (w.H+mgCell-1)/mgCell
	ix := &mgIndex{cols: cols, rows: rows}
	for i := 0; i < cols*rows; i++ {
		ix.regs = append(ix.regs, &mgReg{ID: i, RX: i % cols, RY: i / cols})
	}
	at := func(a []int, p int) int {
		if p >= 0 && p < len(a) {
			return a[p]
		}
		return 0
	}
	bc := make([][17]int, len(ix.regs))
	for y := 0; y < w.H; y++ {
		for x := 0; x < w.W; x++ {
			p := y*w.W + x
			b := at(w.Biome, p)
			if b == 0 {
				continue
			}
			r := ix.regs[(y/mgCell)*cols+x/mgCell]
			r.Land++
			r.Temp += at(w.Temp, p)
			r.Height += at(w.Height, p)
			r.CX += x
			r.CY += y
			bc[r.ID][clampi(b, 0, 16)]++
			if at(w.River, p) > 0 {
				r.River++
			}
			if at(w.Road, p) > 0 {
				r.Roads++
			}
			if at(w.Burg, p) > 0 {
				r.Towns++
			}
			for _, d := range [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
				nx, ny := x+d[0], y+d[1]
				if w.In(nx, ny) && at(w.Biome, ny*w.W+nx) == 0 {
					r.Coast = true
				}
			}
		}
	}
	for _, r := range ix.regs {
		if r.Land > 0 {
			r.Temp /= r.Land
			r.Height /= r.Land
			r.CX /= r.Land
			r.CY /= r.Land
		}
		best := 1
		for b := 1; b < 17; b++ {
			if bc[r.ID][b] > bc[r.ID][best] {
				best = b
			}
		}
		r.Biome = best
		for dy := -1; dy <= 1; dy++ {
			for dx := -1; dx <= 1; dx++ {
				nx, ny := r.RX+dx, r.RY+dy
				if (dx != 0 || dy != 0) && nx >= 0 && ny >= 0 && nx < cols && ny < rows {
					r.Nb = append(r.Nb, ny*cols+nx)
				}
			}
		}
	}
	// names: the largest living town inside, else the world's own region name
	bigPop := map[int]float64{}
	for i := range w.Burgs {
		b := &w.Burgs[i]
		if !b.Alive || b.P < 0 || b.P >= w.W*w.H {
			continue
		}
		r := ix.regs[((b.P/w.W)/mgCell)*cols+(b.P%w.W)/mgCell]
		if b.Pop > bigPop[r.ID] || r.Name == "" {
			bigPop[r.ID] = b.Pop
			r.Name = "the " + b.Name + " country"
		}
	}
	for _, r := range ix.regs {
		if r.Name == "" {
			if r.Land > 0 {
				n := w.RegionName(clampi(r.CY, 0, w.H-1)*w.W + clampi(r.CX, 0, w.W-1))
				if n == "the wilds" {
					n = fmt.Sprintf("the %s reaches (%d,%d)", BiomeName(r.Biome), r.RX, r.RY)
				}
				r.Name = n
			} else {
				r.Name = "open water"
			}
		}
	}
	mgCache[w] = ix
	return ix
}

func (ix *mgIndex) at(w *WorldData, wx, wy int) *mgReg {
	if !w.In(wx, wy) {
		return nil
	}
	i := (wy/mgCell)*ix.cols + wx/mgCell
	if i < 0 || i >= len(ix.regs) {
		return nil
	}
	return ix.regs[i]
}

func mgGuildOf(s *bsSp) int {
	if s.Era != "living" {
		return -1
	}
	if s.Aquatic() || s.Group == "fish" {
		return mgFish
	}
	if s.Group == "bird" && !s.Hostile() {
		return mgBirds
	}
	if s.Hostile() {
		if s.Size >= 5 || s.Rarity >= 4 {
			return mgApex
		}
		return mgPred
	}
	switch s.Group {
	case "mam", "syn", "rept", "dino", "amph", "bird":
	default:
		return -1
	}
	if s.Group == "bird" {
		return mgBirds
	}
	if s.Size >= 3 && (s.Temper == "herd" || s.Temper == "skittish" || s.Temper == "passive" || s.Diet == 'h') {
		return mgHerd
	}
	return mgGame
}

// species lists the bestiary species of each guild that live in the region, heaviest first.
func (ix *mgIndex) species(g *Game, r *mgReg) *[mgGuilds][]mgSpW {
	if r.spOK {
		return &r.sp
	}
	r.spOK = true
	pl := &bsPlace{where: 's', biome: r.Biome, height: r.Height, temp: r.Temp, coast: r.Coast, genre: g.Genre, surface: true}
	if r.River > 0 || r.Coast {
		pl.water = 30
	}
	for _, s := range bsAll {
		gd := mgGuildOf(s)
		if gd < 0 {
			continue
		}
		if gd == mgFish && pl.water == 0 {
			continue
		}
		if w := s.weight(pl); w > 0 {
			r.sp[gd] = append(r.sp[gd], mgSpW{s, w})
		}
	}
	for gd := range r.sp {
		l := r.sp[gd]
		sort.SliceStable(l, func(i, j int) bool {
			if l[i].W != l[j].W {
				return l[i].W > l[j].W
			}
			return l[i].S.ID < l[j].S.ID
		})
		if len(l) > 14 {
			r.sp[gd] = l[:14]
		}
	}
	return &r.sp
}

// ---------------------------------------------------------------- state

type mgSeen struct {
	Day  int           `json:"d"`
	Pop  [mgGuilds]int `json:"p"`
	PDay int           `json:"pd,omitempty"`
	PPop [mgGuilds]int `json:"pp"`
	Via  string        `json:"v,omitempty"`
}

type mgDen struct {
	Key    string `json:"k"`
	Reg    int    `json:"r"`
	WX     int    `json:"wx"`
	WY     int    `json:"wy"`
	ZX     int    `json:"zx"`
	ZY     int    `json:"zy"`
	Name   string `json:"n"`
	Sp     string `json:"sp,omitempty"`
	Alpha  string `json:"al,omitempty"`
	Brood  int    `json:"br,omitempty"`
	Adults int    `json:"ad,omitempty"`
	Epoch  int    `json:"e,omitempty"`
	State  string `json:"st"` // active cleared vacant
	Day    int    `json:"d,omitempty"`
	How    string `json:"h,omitempty"`
	Prev   string `json:"pv,omitempty"`
	Known  bool   `json:"kn,omitempty"`
	Built  bool   `json:"b,omitempty"`
	MX     int    `json:"mx,omitempty"`
	MY     int    `json:"my,omitempty"`
	Lvl    int    `json:"lv,omitempty"`
	Seen   bool   `json:"se,omitempty"` // occupant seen with your own eyes
}

type mgMove struct {
	From int `json:"f"`
	To   int `json:"t"`
	G    int `json:"g"`
	Day  int `json:"d"`
	N    int `json:"n"`
}

type mgRaid struct {
	Zone  string `json:"z"`
	Until int    `json:"u"`
	Day   int    `json:"d"`
}

type mgState struct {
	Day       int                   `json:"d"`
	Init      bool                  `json:"i,omitempty"`
	Pop       map[int][mgGuilds]int `json:"p"`
	Prey      map[int]int           `json:"py"`
	Dens      map[string]*mgDen     `json:"dn"`
	Seen      map[int]*mgSeen       `json:"s"`
	Moves     []mgMove              `json:"mv,omitempty"`
	Zones     map[string]int        `json:"z"`
	Raids     []*mgRaid             `json:"r,omitempty"`
	LastRaid  map[string]int        `json:"lr"`
	Said      map[string]int        `json:"sd"`
	Kills     [mgGuilds]int         `json:"k"`
	Cleared   int                   `json:"cl,omitempty"`
	Displaced int                   `json:"dp,omitempty"`
	Raided    int                   `json:"ra,omitempty"`

	tab, sel, denSel int
}

func (g *Game) mg() *mgState {
	st := g.ext("mg", func() any {
		return &mgState{Pop: map[int][mgGuilds]int{}, Prey: map[int]int{}, Dens: map[string]*mgDen{}, Seen: map[int]*mgSeen{},
			Zones: map[string]int{}, LastRaid: map[string]int{}, Said: map[string]int{}}
	}).(*mgState)
	if st.Pop == nil {
		st.Pop = map[int][mgGuilds]int{}
	}
	if st.Prey == nil {
		st.Prey = map[int]int{}
	}
	if st.Dens == nil {
		st.Dens = map[string]*mgDen{}
	}
	if st.Seen == nil {
		st.Seen = map[int]*mgSeen{}
	}
	if st.Zones == nil {
		st.Zones = map[string]int{}
	}
	if st.LastRaid == nil {
		st.LastRaid = map[string]int{}
	}
	if st.Said == nil {
		st.Said = map[string]int{}
	}
	return st
}

func mgFill(t string, kv ...string) string { return strings.NewReplacer(kv...).Replace(t) }

func (g *Game) mgReady() bool {
	return g != nil && g.W != nil && g.P != nil && len(g.W.Biome) > 0 && g.W.W > 0
}

// pop returns a region's indexes, seeding them from the world seed on first use.
func (st *mgState) pop(g *Game, id int) [mgGuilds]int {
	if p, ok := st.Pop[id]; ok {
		return p
	}
	r := NewRng(fmt.Sprintf("%s:mg:init:%d", g.Seed, id))
	var p [mgGuilds]int
	for i := range p {
		p[i] = 70 + r.Int(0, 55)
	}
	p[mgApex] = p[mgApex] * 45 / 100
	if ix := mgIdx(g.W); id >= 0 && id < len(ix.regs) {
		rg := ix.regs[id]
		if rg.River == 0 && !rg.Coast {
			p[mgFish] = 0
		}
		if rg.Land < 3 {
			p = [mgGuilds]int{}
		}
	}
	st.Pop[id] = p
	return p
}

func (st *mgState) add(g *Game, id, gd, n int) {
	p := st.pop(g, id)
	p[gd] = clampi(p[gd]+n, 0, 260)
	st.Pop[id] = p
}

func mgYear(day int) int { return day / (4 * SeasonDays) }

func (g *Game) mgOffset(year int) int {
	return int(HashStr(fmt.Sprintf("%s:mg:off:%d", g.Seed, year))%13) - 6
}

func (g *Game) mgForage(id, year int) float64 {
	gl := float64(HashStr(fmt.Sprintf("%s:mg:fg:%d", g.Seed, year))%1000) / 1000
	lc := float64(HashStr(fmt.Sprintf("%s:mg:fl:%d:%d", g.Seed, id, year))%1000) / 1000
	return (0.8 + 0.38*gl) * (0.9 + 0.2*lc)
}

func (g *Game) mgCap(r *mgReg, gd, season, year int) int {
	f := mgForageBiome[clampi(r.Biome, 0, 16)]
	swing := [4]float64{0, 6, 0, -8}
	comfort := 1 - math.Abs(float64(r.Temp)+swing[season]-15)/45
	comfort = math.Max(0.2, math.Min(1.1, comfort))
	fg := g.mgForage(r.ID, year)
	hi := r.Height >= 55
	var c float64
	switch gd {
	case mgHerd:
		c = f * comfort * fg * 100
		if hi && season == 1 {
			c *= 1.25
		} else if hi && season == 3 {
			c *= 0.45
		}
		c = math.Max(5, math.Min(180, c))
	case mgGame:
		c = (0.4 + 0.6*f) * (0.7 + 0.3*comfort) * fg * 100
		c = math.Max(10, math.Min(170, c))
	case mgBirds:
		c = (50 + 50*f) * fg
		if season == 3 {
			c *= 0.6
		}
	case mgFish:
		if r.River == 0 && !r.Coast {
			return 0
		}
		c = 90 * (0.7 + 0.3*fg)
	case mgPred:
		c = 170
	default:
		c = 90
	}
	return int(c)
}

func mgClampF(x, lo, hi float64) float64 { return math.Max(lo, math.Min(hi, x)) }

// mgWeek runs one weekly step of the whole planet.
func (g *Game) mgWeek(st *mgState, ix *mgIndex, day int) {
	season := SeasonOfDay(day)
	year := mgYear(day)
	dis := day % SeasonDays
	off := g.mgOffset(year)
	mig := (season == 0 || season == 2) && dis-off >= 0 && dis-off < 14
	boundary := dis < 7
	grow := func(x, c, rate float64) float64 {
		if c <= 0.02 {
			return x * 0.7
		}
		if x < 0.03 && c > 0.3 {
			x = 0.03
		}
		return x + rate*x*(1-x/c)
	}
	for _, r := range ix.regs {
		if r.Land < 3 {
			continue
		}
		p := st.pop(g, r.ID)
		cp := func(i int) float64 { return float64(g.mgCap(r, i, season, year)) / 100 }
		h, s, b, f := float64(p[mgHerd])/100, float64(p[mgGame])/100, float64(p[mgBirds])/100, float64(p[mgFish])/100
		pr, ap := float64(p[mgPred])/100, float64(p[mgApex])/100
		h = grow(h, cp(mgHerd), 0.12)
		s = grow(s, cp(mgGame), 0.18)
		b = grow(b, cp(mgBirds), 0.10)
		f = grow(f, cp(mgFish), 0.12)
		press := 0.045 * (pr + 1.5*ap)
		h -= press * h * 0.7
		s -= press * s * 0.9
		if h > cp(mgHerd)*1.25 {
			h -= 0.12 * (h - cp(mgHerd))
		}
		prey := 0.6*h + 0.4*s
		if boundary {
			tp := mgClampF(0.3+0.8*prey, 0.1, 1.9)
			pr += (tp - pr) * 0.45
			ap += (pr*0.45 - ap) * 0.4
			st.Prey[r.ID] = int(prey * 100)
		}
		if prey < 0.4 {
			pr -= 0.03 * pr
			ap -= 0.03 * ap
		}
		p[mgHerd], p[mgGame], p[mgBirds], p[mgFish] = int(mgClampF(h, 0, 2.6)*100), int(mgClampF(s, 0, 2.6)*100), int(mgClampF(b, 0, 2.6)*100), int(mgClampF(f, 0, 2.6)*100)
		p[mgPred], p[mgApex] = int(mgClampF(pr, 0, 2.6)*100), int(mgClampF(ap, 0, 2.6)*100)
		st.Pop[r.ID] = p
	}
	if !mig {
		return
	}
	ts := (season + 1) % 4 // the range they are heading for
	for _, r := range ix.regs {
		if r.Land < 3 {
			continue
		}
		p := st.pop(g, r.ID)
		if p[mgHerd] < 12 {
			continue
		}
		best, bc := -1, g.mgCap(r, mgHerd, ts, year)
		for _, nb := range r.Nb {
			n := ix.regs[nb]
			if n.Land < 3 {
				continue
			}
			if c := g.mgCap(n, mgHerd, ts, year); c > bc*112/100 {
				best, bc = nb, c
			}
		}
		if best < 0 {
			continue
		}
		amt := p[mgHerd] * 22 / 100
		if amt < 2 {
			continue
		}
		st.add(g, r.ID, mgHerd, -amt)
		st.add(g, best, mgHerd, amt)
		st.Moves = append(st.Moves, mgMove{From: r.ID, To: best, G: mgHerd, Day: day, N: amt})
	}
	if len(st.Moves) > 30 {
		st.Moves = st.Moves[len(st.Moves)-30:]
	}
}

// ---------------------------------------------------------------- windows (pure functions of the date)

func (g *Game) mgTiming(day int) (word string, off int) {
	off = g.mgOffset(mgYear(day))
	switch {
	case off <= -3:
		return "early", off
	case off >= 3:
		return "late", off
	}
	return "on time", off
}

func (g *Game) mgBirdsPass(day int) bool {
	s, d := SeasonOfDay(day), day%SeasonDays
	return (s == 0 || s == 2) && d >= 4 && d < 20
}

func (g *Game) mgFishRun(day int) bool {
	s, d := SeasonOfDay(day), day%SeasonDays
	return (s == 0 && d >= 8 && d < 24) || (s == 2 && d >= 4 && d < 18)
}

func mgWord(v int) string {
	i := v / 25
	return mgPopWords[clampi(i, 0, len(mgPopWords)-1)]
}

func mgBar(v, w int) string {
	n := clampi(v*w/200, 0, w)
	return strings.Repeat("#", n) + strings.Repeat("-", w-n)
}

// ---------------------------------------------------------------- turn

func mgTurn(g *Game) {
	if !g.mgReady() {
		return
	}
	st := g.mg()
	ix := mgIdx(g.W)
	today := g.Day()
	if !st.Init {
		st.Init, st.Day = true, today
	}
	if today > st.Day {
		for d := maxi(st.Day+1, today-120); d <= today; d++ {
			if d%7 == 0 {
				g.mgWeek(st, ix, d)
			}
		}
		st.Day = today
		g.mgDaily(st)
	}
	if g.Turn%10 == 0 {
		g.mgNotice(st)
	}
	if g.Turn%60 == 0 {
		g.mgSee("walked")
		g.mgRaidRoll(st, ix)
	}
	g.mgRaidTick(st)
}

func (g *Game) mgNote(kind, subject, text string) {
	if mgRumourSink != nil {
		mgRumourSink(g, kind, subject, text)
	}
}

// mgDaily: the herds are early or late (once per window).
func (g *Game) mgDaily(st *mgState) {
	day := g.Day()
	s, d := SeasonOfDay(day), day%SeasonDays
	word, off := g.mgTiming(day)
	if (s == 0 || s == 2) && d == maxi(0, off)+1 && word != "on time" {
		key := fmt.Sprintf("tim:%d:%d", mgYear(day), s)
		if st.Said[key] == 0 {
			st.Said[key] = 1
			text := fmt.Sprintf("The herds are %s this year.", word)
			g.mgNote("herds", word, text)
			if g.Cur != nil && g.Cur.Z == 0 && !g.Cur.Underwater {
				g.Msg("{{y|" + text + " Farmers and drovers are talking about it.}}")
			}
		}
	}
}

func (g *Game) mgRegHere() *mgReg {
	if !g.mgReady() || g.Cur == nil {
		return nil
	}
	return mgIdx(g.W).at(g.W, g.Cur.WX, g.Cur.WY)
}

// mgSee keeps what the player has seen of the land, for the trends tab.
func (g *Game) mgSee(via string) {
	r := g.mgRegHere()
	if r == nil || r.Land < 3 || g.Cur.Z != 0 {
		return
	}
	st := g.mg()
	g.mgSeeReg(st, r.ID, via)
}

func (g *Game) mgSeeReg(st *mgState, id int, via string) {
	day := g.Day()
	p := st.pop(g, id)
	s := st.Seen[id]
	if s == nil {
		st.Seen[id] = &mgSeen{Day: day, Pop: p, Via: via}
		return
	}
	if day >= s.Day+7 {
		s.PDay, s.PPop = s.Day, s.Pop
	}
	s.Day, s.Pop, s.Via = day, p, via
}

// ---------------------------------------------------------------- zones: spawns and dens

func (g *Game) mgWild(zn *Zone) bool {
	return g.mgReady() && zn != nil && zn.Z == 0 && zn.Kind == "surface" && !zn.Underwater && !zn.Sky && g.W.In(zn.WX, zn.WY)
}

func mgZone(g *Game, zn *Zone, _ *Rng) { g.mgRefresh(zn) }

func mgEnter(g *Game, zn *Zone) {
	if !g.mgReady() {
		return
	}
	g.mgRefresh(zn)
	g.mgSee("walked")
	g.mgRaidEnter(zn)
}

func (g *Game) mgRefresh(zn *Zone) {
	if !g.mgWild(zn) {
		return
	}
	g.mgPopulate(zn)
	g.mgDenEnsure(zn)
}

func (g *Game) mgDrawFit(r *Rng, list []mgSpW, pl *bsPlace, skip string) *bsSp {
	var pool []*bsSp
	var ws []float64
	for _, e := range list {
		if e.S.ID == skip {
			continue
		}
		if w := e.S.weight(pl); w > 0 {
			pool = append(pool, e.S)
			ws = append(ws, w)
		}
	}
	return bsDraw(r, pool, ws)
}

func mgCount(r *Rng, x float64, capN int) int {
	if x <= 0 {
		return 0
	}
	n := int(x)
	if r.Chance(x - float64(n)) {
		n++
	}
	return mini(n, capN)
}

func (g *Game) mgPopulate(zn *Zone) {
	st := g.mg()
	ix := mgIdx(g.W)
	reg := ix.at(g.W, zn.WX, zn.WY)
	if reg == nil || reg.Land < 3 {
		return
	}
	day := g.Day()
	stamp := day/SeasonDays + 1
	if st.Zones[zn.Key] == stamp {
		return
	}
	if st.Zones[zn.Key] != 0 {
		g.mgDropExtras(zn)
	}
	st.Zones[zn.Key] = stamp
	rr := NewRng(fmt.Sprintf("%s:mg:%s:%d", g.Seed, zn.Key, stamp))
	pop := st.pop(g, reg.ID)
	sp := ix.species(g, reg)
	pl := g.bsPlaceOf(zn)
	put := func(gd, n int, tag string) {
		for i := 0; i < n; i++ {
			s := g.mgDrawFit(rr, sp[gd], pl, "")
			if s == nil {
				return
			}
			before := len(zn.Actors)
			g.bsPutGroup(zn, rr, s, s.groupSize(rr), 0)
			for _, a := range zn.Actors[before:] {
				a.Set("mgx")
				a.Set(tag)
			}
		}
	}
	if pop[mgHerd] >= 45 {
		put(mgHerd, mgCount(rr, float64(pop[mgHerd])/100*0.8, 3), "mgherd")
	}
	if pop[mgGame] >= 45 {
		put(mgGame, mgCount(rr, float64(pop[mgGame])/100*0.6, 2), "mgherd")
	}
	if pop[mgPred] >= 115 {
		put(mgPred, mgCount(rr, float64(pop[mgPred]-100)/100*1.3, 2), "mgpred")
	}
	if g.mgBirdsPass(day) && rr.Chance(0.5) {
		put(mgBirds, 1, "mgbird")
	}
	if g.mgFishRun(day) && pl.water >= 25 && rr.Chance(0.6) {
		put(mgFish, 1, "mgfish")
	}
	if g.mgLean(reg, pop) && rr.Chance(0.35) {
		g.mgRoadPred(zn, rr, sp, pl)
	}
}

// mgLean: lean years push predators toward people.
func (g *Game) mgLean(reg *mgReg, pop [mgGuilds]int) bool {
	prey := (6*pop[mgHerd] + 4*pop[mgGame]) / 10
	return pop[mgPred] > 85 && prey < 70 || g.mgForage(reg.ID, mgYear(g.Day())) < 0.8 && pop[mgPred] > 70
}

func (g *Game) mgRoadPred(zn *Zone, rr *Rng, sp *[mgGuilds][]mgSpW, pl *bsPlace) {
	var road []int
	for i, t := range zn.T {
		if t == TRoad {
			road = append(road, i)
		}
	}
	if len(road) == 0 {
		return
	}
	s := g.mgDrawFit(rr, sp[mgPred], pl, "")
	if s == nil {
		return
	}
	i := road[rr.Int(0, len(road)-1)]
	x, y, ok := bsSpot(zn, rr, s, i%MW, i/MW)
	if !ok {
		return
	}
	a := g.bsSpawn(s, maxi(zn.Level, tplByID[s.Tpl()].Lvl), rr)
	a.X, a.Y, a.HomeX, a.HomeY = x, y, x, y
	a.Set("mgx")
	a.Set("mgroad")
	zn.Actors = append(zn.Actors, a)
}

func (g *Game) mgDropExtras(zn *Zone) {
	keep := zn.Actors[:0]
	for _, a := range zn.Actors {
		if a.Has("mgx") && a.HP > 0 && !a.Angry && !(g.Cur == zn && g.visible(a.X, a.Y)) {
			continue
		}
		keep = append(keep, a)
	}
	zn.Actors = keep
}

// ---------------------------------------------------------------- dens

// mgDenPlan says whether a wild zone holds a den: a pure function of the seed and the zone key.
func (g *Game) mgDenPlan(wx, wy, zx, zy int) bool {
	w := g.W
	if !w.In(wx, wy) || zx == 1 && zy == 1 {
		return false
	}
	p := w.Idx(wx, wy)
	if w.Biome[p] == 0 || p < len(w.Burg) && w.Burg[p] > 0 || p < len(w.Biome) && w.Biome[p] == 11 {
		return false
	}
	if HashStr(fmt.Sprintf("%s:mg:denp:%d.%d.%d.%d", g.Seed, wx, wy, zx, zy))%100 >= 8 {
		return false
	}
	reg := mgIdx(w).at(w, wx, wy)
	if reg == nil {
		return false
	}
	sp := mgIdx(w).species(g, reg)
	return len(sp[mgPred])+len(sp[mgApex]) > 0
}

func (g *Game) mgAlphaName(key string, epoch int) string {
	r := NewRng(fmt.Sprintf("%s:mg:an:%s:%d", g.Seed, key, epoch))
	return Pick(r, mgAlphaFirst) + Pick(r, mgAlphaLast)
}

func (g *Game) mgDenStub(wx, wy, zx, zy int) *mgDen {
	st := g.mg()
	key := ZoneKey(wx, wy, zx, zy, 0)
	if d := st.Dens[key]; d != nil {
		return d
	}
	reg := mgIdx(g.W).at(g.W, wx, wy)
	r := NewRng(fmt.Sprintf("%s:mg:dn:%s", g.Seed, key))
	d := &mgDen{Key: key, Reg: reg.ID, WX: wx, WY: wy, ZX: zx, ZY: zy, State: "active"}
	d.Name = g.mgAlphaName(key, 0) + "'s " + Pick(r, mgDenPlace)
	st.Dens[key] = d
	return d
}

func mgSeasonCount(day int) int { return day / SeasonDays }

func (g *Game) mgDenEnsure(zn *Zone) {
	if !g.mgDenPlan(zn.WX, zn.WY, zn.ZX, zn.ZY) {
		return
	}
	d := g.mgDenStub(zn.WX, zn.WY, zn.ZX, zn.ZY)
	reg := mgIdx(g.W).at(g.W, zn.WX, zn.WY)
	if !d.Built {
		rg := NewRng(fmt.Sprintf("%s:mg:build:%s", g.Seed, zn.Key))
		g.mgPickOccupant(zn, reg, d, rg, "")
		g.mgBuildDen(zn, reg, d, rg, false)
		return
	}
	if (d.State == "cleared" || d.State == "vacant") && mgSeasonCount(g.Day()) > mgSeasonCount(d.Day) {
		g.mgMoveIn(zn, reg, d)
	}
}

func (g *Game) mgPickOccupant(zn *Zone, reg *mgReg, d *mgDen, rg *Rng, skip string) *bsSp {
	st := g.mg()
	sp := mgIdx(g.W).species(g, reg)
	pl := g.bsPlaceOf(zn)
	pop := st.pop(g, reg.ID)
	list := sp[mgPred]
	if (pop[mgApex] > 60 && rg.Chance(0.3) || len(list) == 0) && len(sp[mgApex]) > 0 {
		list = sp[mgApex]
	}
	s := g.mgDrawFit(rg, list, pl, skip)
	if s == nil {
		s = g.mgDrawFit(rg, append(append([]mgSpW{}, sp[mgPred]...), sp[mgApex]...), pl, skip)
	}
	if s == nil {
		s = g.mgDrawFit(rg, append(append([]mgSpW{}, sp[mgPred]...), sp[mgApex]...), pl, "")
	}
	if s != nil {
		d.Sp = s.ID
		d.Brood = rg.Int(1, 3)
		d.Adults = 0
		if s.Temper == "pack" || s.Has("pack") {
			d.Adults = rg.Int(1, 2)
		}
	}
	return s
}

func (g *Game) mgBuildDen(zn *Zone, reg *mgReg, d *mgDen, rg *Rng, moved bool) {
	s := bsByID[d.Sp]
	if s == nil {
		return
	}
	if !d.Built {
		var x, y int
		ok := false
		for i := 0; i < 30 && !ok; i++ {
			if x, y, ok = zn.FreeSpot(rg, nil); ok && (abs(x-40) < 8 && abs(y-12) < 4 || x < 4 || y < 4 || x > MW-5 || y > MH-5) {
				ok = false
			}
		}
		if !ok {
			return
		}
		d.MX, d.MY, d.Built = x, y, true
		for dy := -2; dy <= 2; dy++ {
			for dx := -3; dx <= 3; dx++ {
				switch zn.At(x+dx, y+dy) {
				case TGrass, TTallGrass, TDirt, TSand, TSnow, TMoss, TAsh:
					if rg.Chance(0.7) {
						zn.Set(x+dx, y+dy, TDirt)
					}
				}
			}
		}
		if zn.Notes == nil {
			zn.Notes = map[int]string{}
		}
		zn.Set(x, y, TRubble)
		zn.Notes[y*MW+x] = Pick(rg, mgDenSite)
	}
	base := maxi(zn.Level, tplByID[s.Tpl()].Lvl)
	d.Lvl = base
	aname := d.Alpha
	if aname == "" || d.Epoch > 0 && moved {
		aname = g.mgAlphaName(d.Key, d.Epoch)
	}
	d.Alpha = aname
	mk := func(lvl int, tag string) *Actor {
		x, y, ok := bsSpot(zn, rg, s, d.MX, d.MY)
		if !ok {
			return nil
		}
		a := g.bsSpawn(s, lvl, rg)
		a.X, a.Y, a.HomeX, a.HomeY = x, y, d.MX, d.MY
		a.Set("mgden")
		a.Set(tag)
		a.AI = "hostile"
		zn.Actors = append(zn.Actors, a)
		return a
	}
	if a := mk(base+2, "mgalpha"); a != nil {
		if s.Rarity >= 4 || s.Size >= 5 {
			a.Level++
		}
		a.Name, a.Proper = aname+" the "+s.Name, true
		a.MaxHP = a.MaxHP * 130 / 100
		a.HP = a.MaxHP
		a.Desc = fmt.Sprintf("%s, the alpha of %s. %s", aname, d.Name, a.Desc)
	}
	bw := mgBroodWord[s.Group]
	if bw == "" {
		bw = "young"
	}
	for i := 0; i < d.Adults; i++ {
		mk(base, "mgadult")
	}
	for i := 0; i < d.Brood; i++ {
		if a := mk(1, "mgbrood"); a != nil {
			a.Name = s.Name + " " + bw
			a.Level = maxi(1, a.Level-2)
			a.MaxHP = maxi(3, a.MaxHP*55/100)
			a.HP = a.MaxHP
		}
	}
	d.State = "active"
	if moved {
		g.mgTraces(zn, d, s, rg)
	}
}

func (g *Game) mgTraces(zn *Zone, d *mgDen, s *bsSp, rg *Rng) {
	if zn.Notes == nil {
		zn.Notes = map[int]string{}
	}
	for i := 0; i < 3; i++ {
		x, y := clampi(d.MX+rg.Int(-5, 5), 2, MW-3), clampi(d.MY+rg.Int(-3, 3), 2, MH-3)
		if !zn.Walkable(x, y) || zn.At(x, y) == TInscription || zn.At(x, y) == TRubble {
			continue
		}
		if i == 0 {
			zn.Items = append(zn.Items, &Item{ID: g.newID(), Kind: "scrap", Name: mgFill(Pick(rg, mgTrackWords), "{sp}", strings.ToLower(s.Name)), Glyph: ',', F: CBrown, Value: 0, Count: 1, X: x, Y: y})
		} else {
			zn.Set(x, y, TBones)
		}
	}
	rx, ry := clampi(d.MX+rg.Int(-2, 2), 2, MW-3), clampi(d.MY+2, 2, MH-3)
	if zn.Walkable(rx, ry) && zn.At(rx, ry) != TRubble {
		zn.Items = append(zn.Items, &Item{ID: g.newID(), Kind: "scrap", Name: Pick(rg, mgRemainWords), Glyph: '%', F: CDarkGray, Value: 0, Count: 1, X: rx, Y: ry})
	}
	// the farmers' notice stone, away from the mound
	for i := 0; i < 30; i++ {
		x, y, ok := zn.FreeSpot(rg, nil)
		if !ok {
			return
		}
		if abs(x-d.MX)+abs(y-d.MY) < 8 || abs(x-d.MX)+abs(y-d.MY) > 22 || abs(x-40) < 3 && abs(y-12) < 2 {
			continue
		}
		town := "the nearby farms"
		if b := g.rmNearestBurg(g.W.Idx(zn.WX, zn.WY)); b != nil {
			town = b.Name
		}
		oldN := "old tenants"
		if o := bsByID[d.Prev]; o != nil {
			oldN = strings.ToLower(o.Name)
		}
		tpl := Pick(rg, mgNoticeSelf)
		zn.Set(x, y, TInscription)
		zn.Notes[y*MW+x] = mgFill(tpl, "{town}", town, "{old}", oldN, "{sp}", strings.ToLower(s.Name), "{season}", mgSeasonNames[SeasonOfDay(g.Day())])
		return
	}
}

func (g *Game) mgMoveIn(zn *Zone, reg *mgReg, d *mgDen) {
	st := g.mg()
	rg := NewRng(fmt.Sprintf("%s:mg:in:%s:%d", g.Seed, zn.Key, d.Epoch+1))
	old := d.Sp
	var s *bsSp
	if rg.Chance(0.45) {
		s = bsByID[old]
		d.Sp = old
		d.Brood, d.Adults = rg.Int(1, 3), rg.Int(1, 2)
	} else {
		s = g.mgPickOccupant(zn, reg, d, rg, old)
	}
	if s == nil {
		return
	}
	d.Prev = old
	d.Epoch++
	d.Alpha = ""
	d.State, d.Day, d.How = "active", g.Day(), ""
	d.Seen = false
	g.mgBuildDen(zn, reg, d, rg, true)
	st.Displaced++
	oldS := "the old tenants"
	if o := bsByID[old]; o != nil {
		oldS = "the " + strings.ToLower(o.Name)
	}
	g.mgNote("den", d.Key, fmt.Sprintf("Something new has moved into %s: %s replace %s.", d.Name, strings.ToLower(s.Name), oldS))
}

// mgDenNear finds an active den in the current zone with live occupants within reach.
func (g *Game) mgDenNear() *mgDen {
	if !g.mgReady() || g.Cur == nil || g.P == nil {
		return nil
	}
	d := g.mg().Dens[g.Cur.Key]
	if d == nil || !d.Built || d.State != "active" || abs(g.P.X-d.MX) > 9 || abs(g.P.Y-d.MY) > 7 {
		return nil
	}
	if len(g.mgDenActors(g.Cur)) == 0 {
		return nil
	}
	return d
}

func (g *Game) mgDenActors(zn *Zone) []*Actor {
	var out []*Actor
	for _, a := range zn.Actors {
		if a.Has("mgden") && a.HP > 0 {
			out = append(out, a)
		}
	}
	return out
}

// mgNotice marks a den as known once you are close to the mound.
func (g *Game) mgNotice(st *mgState) {
	if !g.mgReady() || g.Cur == nil {
		return
	}
	d := st.Dens[g.Cur.Key]
	if d == nil || !d.Built || abs(g.P.X-d.MX) > 14 || abs(g.P.Y-d.MY) > 9 {
		return
	}
	if !d.Known {
		d.Known = true
		g.Msg("{{O|You find the signs of a den here: " + d.Name + ".}}")
	}
	if d.State == "active" && !d.Seen {
		for _, a := range g.mgDenActors(g.Cur) {
			if g.visible(a.X, a.Y) {
				d.Seen = true
				if s := bsByID[d.Sp]; s != nil {
					g.Msg(fmt.Sprintf("{{O|%s holds this den: %s, with %d young.}}", d.Alpha, strings.ToLower(s.Name), d.Brood))
				}
				break
			}
		}
	}
}

// ---------------------------------------------------------------- clearing a den

type mgdRow struct {
	ID, Text string
	OK       bool
	Why      string
}

func (g *Game) mgFind(f func(it *Item) bool) *Item {
	for _, it := range g.P.Inv {
		if f(it) {
			return it
		}
	}
	return nil
}

func (g *Game) mgPoisonItem() *Item {
	return g.mgFind(func(it *Item) bool {
		if it.Kind == "poison" {
			return true
		}
		return it.Kind == "tonic" && it.Effect == "brewed" && (it.Extra["alp:poison"] > 0 || it.Extra["alp:paralysis"] > 0 || it.Extra["alp:sleep"] > 0)
	})
}

func (g *Game) mgFireItem() *Item {
	return g.mgFind(func(it *Item) bool {
		return it.Kind == "grenade" && it.Effect == "fire" || it.Kind == "light" || it.Elem == "fire" && it.Kind != "weapon" && it.Kind != "ranged"
	})
}

func (g *Game) mgBaitItem() *Item {
	return g.mgFind(func(it *Item) bool { return it.Kind == "food" || it.Kind == "corpse" })
}

func (g *Game) mgDC(d *mgDen, adj int) int { return clampi(9+d.Lvl/4+adj, 7, 15) }

func (g *Game) mgdRows(d *mgDen) []mgdRow {
	chk := func(adj int) string {
		return g.checkLabel("survival", g.mgDC(d, adj))
	}
	rows := []mgdRow{{ID: "read", Text: "Read the signs around the den " + chk(-3), OK: true}}
	row := func(id, text string, it *Item, need string, adj int) {
		r := mgdRow{ID: id, Text: text + " " + chk(adj), OK: it != nil}
		if it == nil {
			r.Why = need
		}
		rows = append(rows, r)
	}
	row("poison", "Poison the den (bait and brew)", g.mgPoisonItem(), "You need a poison or a harmful brew.", 0)
	row("burn", "Burn the den out", g.mgFireItem(), "You need fire: a torch or a fire grenade.", -1)
	row("lure", "Lure them away with bait", g.mgBaitItem(), "You need meat or food for bait.", 1)
	rows = append(rows, mgdRow{ID: "back", Text: "Leave it for now. (You can still fight your way in.)", OK: true})
	return rows
}

func mgdKeys(g *Game, k string) {
	d := g.mgDenNear()
	st := g.mg()
	if d == nil {
		g.Mode = "main"
		return
	}
	rows := g.mgdRows(d)
	switch k {
	case "Escape", "q":
		g.Mode = "main"
	case "ArrowUp", "N8", "k":
		st.denSel = maxi(0, st.denSel-1)
	case "ArrowDown", "N2", "j":
		st.denSel = mini(len(rows)-1, st.denSel+1)
	case "Enter", " ":
		if st.denSel >= len(rows) {
			return
		}
		r := rows[st.denSel]
		if !r.OK {
			g.Msg("{{K|" + r.Why + "}}")
			return
		}
		g.Mode = "main"
		g.mgDenAct(d, r.ID)
	}
}

func (g *Game) mgdDraw() {
	d := g.mgDenNear()
	sc := g.sc
	x, y, w, h := g.panel("The den")
	if d == nil {
		sc.TextB(x, y, "There is no den here.", CGray, CBgDark, x+w)
		return
	}
	st := g.mg()
	sc.TextB(x, y, "{{K|Up/Down choose  Enter do it  Esc close}}", CGray, CBgDark, x+w)
	y += 2
	s := bsByID[d.Sp]
	head := "{{Y|" + d.Name + "}}"
	if d.Seen && s != nil {
		head += fmt.Sprintf("  {{W|%s}}, %s, %d young, %d pack mates", d.Alpha, strings.ToLower(s.Name), d.Brood, d.Adults)
	} else {
		head += "  {{K|You have not seen what lives in it.}}"
	}
	sc.TextB(x, y, head, CGray, CBgDark, x+w)
	y += 2
	rows := g.mgdRows(d)
	st.denSel = clampi(st.denSel, 0, len(rows)-1)
	for i, r := range rows {
		bg := CBgDark
		if i == st.denSel {
			bg = CBgSel
		}
		sc.Fill(x, y, w, 1, bg)
		c := "W"
		if !r.OK {
			c = "K"
		}
		sc.TextB(x, y, fmt.Sprintf("{{%s|%s}}", c, r.Text), CGray, bg, x+w)
		y++
	}
	y++
	notes := []string{"Poison kills the young and the pack mates outright and weakens the alpha. Fire drives out or burns what is inside. Bait pulls the family away to somewhere else (other farmers will meet them). A failed attempt wakes the den, and a fight starts.",
		"A cleared den stays empty until the next season, then something else, or a rival pack, moves in."}
	for _, n := range notes {
		for _, ln := range Wrap(n, w) {
			if y < 2+h {
				sc.TextB(x, y, "{{K|"+ln+"}}", CGray, CBgDark, x+w)
				y++
			}
		}
	}
}

func (g *Game) mgDenAct(d *mgDen, id string) {
	zn := g.Cur
	rg := g.R
	switch id {
	case "read":
		ok := g.roll2d6Check(g.checkMod("survival"), g.mgDC(d, -3))
		d.Known = true
		if s := bsByID[d.Sp]; s != nil && ok {
			d.Seen = true
			g.Msg(fmt.Sprintf("{{O|You read the signs: %s's den, %s, %d young and %d pack mates.}}", d.Alpha, strings.ToLower(s.Name), d.Brood, d.Adults))
		} else {
			g.Msg("The tracks overlap and tell you little.")
		}
		g.spend(300)
		return
	case "poison":
		it := g.mgPoisonItem()
		if it == nil {
			return
		}
		g.consume(it)
		g.spend(600)
		if g.roll2d6Check(g.checkMod("survival"), g.mgDC(d, 0)) {
			var alpha *Actor
			for _, a := range g.mgDenActors(zn) {
				if a.Has("mgalpha") {
					alpha = a
					continue
				}
				g.removeActor(zn, a)
			}
			if alpha != nil && rg.Chance(0.6) {
				g.removeActor(zn, alpha) // crawls off to die
				g.Msg("{{G|The den falls quiet. Later you find the alpha dead in the grass, foam at the mouth.}}")
			} else if alpha != nil {
				alpha.HP = maxi(1, alpha.HP*40/100)
				alpha.AddSt("poisoned", 40)
				alpha.Angry = true
				g.Msg("{{G|The young and the pack mates die quietly. The alpha staggers out, sick and furious.}}")
				return
			}
			g.mgCleared(zn, d, "poison")
			return
		}
		g.mgWake(zn, "The bait is refused. Growls rise from the den.")
	case "burn":
		it := g.mgFireItem()
		if it == nil {
			return
		}
		if it.Kind == "grenade" {
			g.consume(it)
		}
		g.spend(500)
		for dy := -2; dy <= 2; dy++ {
			for dx := -3; dx <= 3; dx++ {
				switch zn.At(d.MX+dx, d.MY+dy) {
				case TGrass, TTallGrass, TDirt, TMoss, TBush:
					zn.Set(d.MX+dx, d.MY+dy, TAshBed)
				}
			}
		}
		if g.roll2d6Check(g.checkMod("survival"), g.mgDC(d, -1)) {
			var alpha *Actor
			for _, a := range g.mgDenActors(zn) {
				if a.Has("mgalpha") {
					alpha = a
					continue
				}
				g.removeActor(zn, a)
			}
			if alpha != nil && rg.Chance(0.55) {
				alpha.AddSt("burning", 10)
				alpha.Angry = true
				g.Msg("{{R|Smoke pours out. The young and the pack mates do not come out. The alpha does, burning, and goes for you.}}")
				return
			}
			g.Msg("{{O|The den burns out. Nothing comes out alive.}}")
			if alpha != nil {
				g.removeActor(zn, alpha)
			}
			g.mgCleared(zn, d, "fire")
			return
		}
		g.P.AddSt("burning", 3)
		g.mgWake(zn, "The fire catches the wrong way and the den boils over.")
	case "lure":
		it := g.mgBaitItem()
		if it == nil {
			return
		}
		g.consume(it)
		g.spend(900)
		if g.roll2d6Check(g.checkMod("survival"), g.mgDC(d, 1)) {
			for _, a := range g.mgDenActors(zn) {
				g.removeActor(zn, a)
			}
			g.Msg("{{G|You lay a trail of bait away from the den. The family follows it over the ridge, and does not come back.}}")
			if reg := mgIdx(g.W).regs[d.Reg]; reg != nil {
				for _, nb := range reg.Nb {
					if mgIdx(g.W).regs[nb].Land >= 3 {
						g.mg().add(g, nb, mgPred, 4)
						break
					}
				}
			}
			g.mgCleared(zn, d, "lured")
			return
		}
		g.mgWake(zn, "They follow the bait right back to you.")
	}
}

func (g *Game) mgWake(zn *Zone, why string) {
	for _, a := range g.mgDenActors(zn) {
		a.Angry = true
		a.AddSt("frenzied", 12)
	}
	g.Msg("{{R|" + why + " The whole den comes for you.}}")
}

func (g *Game) mgCleared(zn *Zone, d *mgDen, how string) {
	st := g.mg()
	d.State, d.Day, d.How = "cleared", g.Day(), how
	if how == "lured" {
		d.State = "vacant"
	}
	st.Cleared++
	s := bsByID[d.Sp]
	pen, xp := 6, 40+d.Lvl*8
	if s != nil && (s.Size >= 5 || s.Rarity >= 4) {
		pen, xp = 10, xp+40
		st.Kills[mgApex]++
	}
	if how != "lured" {
		st.add(g, d.Reg, mgPred, -pen)
	}
	st.add(g, d.Reg, mgHerd, 4)
	g.gainXP(xp)
	words := map[string]string{"fought": "fought out", "poison": "poisoned out", "fire": "burned out", "lured": "lured away"}
	text := fmt.Sprintf("%s was %s of %s.", d.Alpha+"'s family", words[how], d.Name)
	if s != nil {
		text = fmt.Sprintf("The %s of %s were %s.", strings.ToLower(s.Name), d.Name, words[how])
	}
	g.Msg("{{G|" + text + " The farmers will be glad. Next season something else may move in.}}")
	g.deed("hunt", "The wanderer "+g.P.Name+": "+text, 0, 0)
	g.mgNote("den", d.Key, text)
}

// ---------------------------------------------------------------- kills

func mgKill(g *Game, d, src *Actor) {
	if !g.mgReady() || d == nil || g.Cur == nil {
		return
	}
	st := g.mg()
	byP := src == g.P || src != nil && src.AI == "follower"
	reg := g.mgRegHere()
	if d.Has("mgden") {
		if den := st.Dens[g.Cur.Key]; den != nil && den.State == "active" {
			left := 0
			for _, a := range g.mgDenActors(g.Cur) {
				if a != d {
					left++
				}
			}
			if d.Has("mgalpha") && byP {
				name := d.Name
				g.Cur.Items = append(g.Cur.Items, &Item{ID: g.newID(), Kind: "trophy", Name: strings.TrimSuffix(strings.Split(name, " the ")[0], ",") + "'s skull", Glyph: '♦', F: CYellow, Value: 40 + den.Lvl*10, Weight: 2, X: d.X, Y: d.Y,
					Lore: "The alpha of " + den.Name + ". Farmers will pay to see it."})
			}
			if left == 0 {
				g.mgCleared(g.Cur, den, "fought")
			}
		}
	}
	if !byP || reg == nil || reg.Land < 3 {
		return
	}
	s := bsByTpl[d.Tpl]
	if s == nil {
		return
	}
	gd := mgGuildOf(s)
	if gd < 0 {
		return
	}
	cost := map[int]int{mgHerd: 1, mgGame: 1, mgBirds: 1, mgFish: 1, mgPred: 3, mgApex: 8}[gd]
	st.add(g, reg.ID, gd, -cost)
	st.Kills[gd]++
}

// ---------------------------------------------------------------- raids

func (g *Game) mgFarmZone(zn *Zone) bool {
	if zn == nil || zn.Z != 0 {
		return false
	}
	if zn.Kind == "outskirts" {
		return true
	}
	if zn.Kind == "town" && zn.Burg > 0 {
		dist, _ := g.districtAt(zn.WX, zn.WY, zn.ZX, zn.ZY)
		return dist == "farms"
	}
	return false
}

func (g *Game) mgRaidRoll(st *mgState, ix *mgIndex) {
	zn := g.Cur
	if !g.mgReady() || !g.mgFarmZone(zn) {
		return
	}
	reg := ix.at(g.W, zn.WX, zn.WY)
	if reg == nil || reg.Land < 3 {
		return
	}
	day := g.Day()
	if last, ok := st.LastRaid[zn.Key]; ok && day-last < 3 {
		return
	}
	pop := st.pop(g, reg.ID)
	if !g.mgLean(reg, pop) {
		return
	}
	ch := 0.06
	if h := g.Hour(); h >= 20 || h < 5 {
		ch = 0.14
	}
	if !g.R.Chance(ch) {
		return
	}
	sp := ix.species(g, reg)
	pl := g.bsPlaceOf(zn)
	s := g.mgDrawFit(g.R, sp[mgPred], pl, "")
	if s == nil {
		return
	}
	st.LastRaid[zn.Key] = day
	if len(st.LastRaid) > 200 {
		st.LastRaid = map[string]int{zn.Key: day}
	}
	n := g.R.Int(1, 3)
	ex, ey := 2, g.R.Int(4, MH-5)
	if g.R.Chance(0.5) {
		ex = MW - 3
	}
	put := 0
	for i := 0; i < n; i++ {
		x, y, ok := bsSpot(zn, g.R, s, ex, ey)
		if !ok {
			continue
		}
		a := g.bsSpawn(s, maxi(zn.Level, tplByID[s.Tpl()].Lvl), g.R)
		a.X, a.Y, a.HomeX, a.HomeY = x, y, x, y
		a.Set("mgraid")
		a.AI = "hostile"
		zn.Actors = append(zn.Actors, a)
		put++
	}
	if put == 0 {
		return
	}
	for i := 0; i < 2; i++ {
		x, y, ok := bsSpot(zn, g.R, s, ex, ey)
		if ok && zn.At(x, y) != TInscription {
			zn.Items = append(zn.Items, &Item{ID: g.newID(), Kind: "scrap", Name: "savaged sheep", Glyph: '%', F: CRed, Value: 0, Count: 1, X: x, Y: y})
		}
	}
	st.Raids = append(st.Raids, &mgRaid{Zone: zn.Key, Until: g.Turn + 150, Day: day})
	st.Raided++
	g.Msg("{{R|" + mgFill(Pick(g.R, mgRaidText), "{sp}", strings.ToLower(s.Name)) + "}}")
	g.mgNote("raid", zn.Key, fmt.Sprintf("%s came down from the wild and raided the farms near %s.", titleCase(s.Name), zn.Name))
}

func (g *Game) mgRaidEnter(zn *Zone) {}

func (g *Game) mgRaidTick(st *mgState) {
	if len(st.Raids) == 0 {
		return
	}
	keep := st.Raids[:0]
	for _, r := range st.Raids {
		if g.Turn < r.Until {
			keep = append(keep, r)
			continue
		}
		if zn := g.Zones[r.Zone]; zn != nil {
			shown := false
			rest := zn.Actors[:0]
			for _, a := range zn.Actors {
				if a.Has("mgraid") && a.HP > 0 {
					if g.Cur == zn && g.visible(a.X, a.Y) {
						shown = true
					}
					continue
				}
				rest = append(rest, a)
			}
			zn.Actors = rest
			if shown {
				g.Msg("{{K|" + mgRaidWithdraw + "}}")
			}
		}
	}
	st.Raids = keep
}

// ---------------------------------------------------------------- talk, lines, prices

func mgHerdFolk(a *Actor) bool {
	t := strings.ToLower(a.Title)
	switch a.Role {
	case "huntsman", "warden", "ef_shepherd", "ef_lumberjack", "gardener":
		return true
	}
	for _, k := range []string{"farmer", "herder", "shepherd", "drover", "fisher", "hunter", "forester", "ranger"} {
		if strings.Contains(t, k) {
			return true
		}
	}
	return false
}

func mgTalk(g *Game, a *Actor) []Option {
	if !g.mgReady() || g.Cur == nil || a == nil || a.HP <= 0 || a.Angry || a.AI == "hostile" || a.AI == "follower" || a.Tpl != "npc" || !mgHerdFolk(a) {
		return nil
	}
	return []Option{
		{"{{O|Ask about the herds.}}", func() { g.talkMenu(g.mgAsk(a)) }},
		{"{{O|Ask where the beasts den.}}", func() { g.talkMenu(g.mgAskDens(a)) }},
	}
}

func (g *Game) mgAsk(a *Actor) string {
	if r := g.mgRegHere(); r != nil {
		st := g.mg()
		g.mgSeeReg(st, r.ID, "told")
		for _, nb := range r.Nb {
			if st.Seen[nb] == nil && mgIdx(g.W).regs[nb].Land >= 3 && HashStr(fmt.Sprintf("%s:mg:tell:%d:%d", g.Seed, a.ID, nb))%3 == 0 {
				g.mgSeeReg(st, nb, "told")
			}
		}
	}
	if s := g.mgLine(a); s != "" {
		return s
	}
	return "\"Nothing out of the ordinary.\""
}

// mgLine is what a farmer or herder says about the herds right now ("" outside the wild country).
func (g *Game) mgLine(a *Actor) string {
	if !g.mgReady() {
		return ""
	}
	r := g.mgRegHere()
	if r == nil || r.Land < 3 {
		return ""
	}
	return mgLine(g, a)
}

func mgLine(g *Game, a *Actor) string {
	r := g.mgRegHere()
	if r == nil || r.Land < 3 {
		return ""
	}
	st := g.mg()
	ix := mgIdx(g.W)
	pop := st.pop(g, r.ID)
	day := g.Day()
	id := 0
	if a != nil {
		id = a.ID
	}
	h := NewRng(fmt.Sprintf("%s:mg:line:%d:%d", g.Seed, id, day))
	sp := ix.species(g, r)
	name := func(gd int) string {
		if len(sp[gd]) == 0 {
			return "beasts"
		}
		return strings.ToLower(sp[gd][0].S.Name) + "s"
	}
	kv := []string{"{place}", r.Name, "{sp}", name(mgHerd)}
	// dens first: news of tenants
	var news *mgDen
	keys := make([]string, 0, len(st.Dens))
	for k := range st.Dens {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		d := st.Dens[k]
		if (d.Reg == r.ID || g.mgNear(ix, d.Reg, r.ID)) && d.Built && day-d.Day <= 2*SeasonDays && (d.Epoch > 0 || d.State != "active") {
			news = d
			break
		}
	}
	if news != nil {
		kv = []string{"{place}", news.Name, "{sp}", "something you would not want to meet"}
		if o := bsByID[news.Sp]; o != nil && news.State == "active" {
			kv[3] = strings.ToLower(o.Name)
		}
		if news.State == "active" {
			return mgFill(Pick(h, mgLineDenNew), kv...)
		}
		return mgFill(Pick(h, mgLineDenOld), kv...)
	}
	word, off := g.mgTiming(day)
	d := day % SeasonDays
	if s := SeasonOfDay(day); (s == 0 || s == 2) && d < 30 && word != "on time" && abs(off) >= 3 && h.Chance(0.6) {
		if word == "early" {
			return mgFill(Pick(h, mgLineEarly), kv...)
		}
		return mgFill(Pick(h, mgLineLate), kv...)
	}
	year := mgYear(day)
	season := SeasonOfDay(day)
	if pop[mgHerd] > g.mgCap(r, mgHerd, season, year)*115/100 && pop[mgHerd] > 100 {
		return mgFill(Pick(h, mgLineOver), kv...)
	}
	if g.mgLean(r, pop) {
		if h.Chance(0.5) {
			return mgFill(Pick(h, mgLineLean), kv...)
		}
		kv[3] = name(mgPred)
		return mgFill(Pick(h, mgLinePred), kv...)
	}
	if pop[mgPred] > 125 {
		kv[3] = name(mgPred)
		return mgFill(Pick(h, mgLinePred), kv...)
	}
	if g.mgBirdsPass(day) && h.Chance(0.4) {
		return mgFill(Pick(h, mgLineBirds), kv...)
	}
	if g.mgFishRun(day) && r.River > 0 && h.Chance(0.5) {
		return mgFill(Pick(h, mgLineFish), kv...)
	}
	if pop[mgHerd] < 55 {
		return mgFill(Pick(h, mgLineThin), kv...)
	}
	if pop[mgHerd] > 120 {
		return mgFill(Pick(h, mgLineGood), kv...)
	}
	return mgFill(Pick(h, mgLineQuiet), kv...)
}

func (g *Game) mgNear(ix *mgIndex, a, b int) bool {
	if a == b {
		return true
	}
	for _, n := range ix.regs[a].Nb {
		if n == b {
			return true
		}
	}
	return false
}

func (g *Game) mgAskDens(a *Actor) string {
	r := g.mgRegHere()
	if r == nil || g.Cur == nil {
		return "\"I stay on my own land. Ask a huntsman.\""
	}
	ix := mgIdx(g.W)
	type cand struct {
		d    *mgDen
		dist int
	}
	var cs []cand
	for _, id := range append([]int{r.ID}, r.Nb...) {
		rg := ix.regs[id]
		if rg.Land < 3 {
			continue
		}
		for ry := 0; ry < mgCell; ry++ {
			for rx := 0; rx < mgCell; rx++ {
				wx, wy := rg.RX*mgCell+rx, rg.RY*mgCell+ry
				if !g.W.In(wx, wy) {
					continue
				}
				for zy := 0; zy < 3; zy++ {
					for zx := 0; zx < 3; zx++ {
						if g.mgDenPlan(wx, wy, zx, zy) {
							d := g.mgDenStub(wx, wy, zx, zy)
							cs = append(cs, cand{d, abs(wx-g.Cur.WX) + abs(wy-g.Cur.WY)})
						}
					}
				}
			}
		}
	}
	sort.SliceStable(cs, func(i, j int) bool {
		if cs[i].d.Known != cs[j].d.Known {
			return !cs[i].d.Known
		}
		if cs[i].dist != cs[j].dist {
			return cs[i].dist < cs[j].dist
		}
		return cs[i].d.Key < cs[j].d.Key
	})
	if len(cs) == 0 {
		return "\"No dens that I know of. The beasts keep to the high ground here.\""
	}
	c := cs[0].d
	c.Known = true
	g.ceSetQuestTarget(c.WX, c.WY)
	dir := rmDir(c.WX-g.Cur.WX, c.WY-g.Cur.WY)
	reg := ix.regs[c.Reg]
	return fmt.Sprintf("\"There is a den they call %s, %s of here in %s, about %d parasangs off. Something bad lives in it. I would not go with less than three good blades.\" {{c|Your quest arrow points at (%d,%d).}}", c.Name, dir, reg.Name, maxi(1, cs[0].dist), c.WX, c.WY)
}

// mgPriceMod is the food price multiplier of the land you stand in (1.0 in a normal year).
func mgPriceMod(g *Game) float64 {
	if !g.mgReady() || g.Cur == nil {
		return 1
	}
	r := g.mgRegHere()
	if r == nil || r.Land < 3 {
		return 1
	}
	pop := g.mg().pop(g, r.ID)
	m := 1.0
	day := g.Day()
	if pop[mgHerd] > g.mgCap(r, mgHerd, SeasonOfDay(day), mgYear(day))*115/100 && pop[mgHerd] > 100 {
		m += 0.12
	}
	if pop[mgHerd] < 70 {
		m += mgClampF(float64(70-pop[mgHerd])*0.004, 0, 0.2)
	}
	if g.mgLean(r, pop) {
		m += 0.08
	}
	return math.Min(m, 1.35)
}

func mgFoodItem(it *Item) bool {
	if it == nil {
		return false
	}
	switch it.Kind {
	case "food", "dish", "corpse":
		return true
	}
	n := strings.ToLower(it.Name)
	for _, k := range []string{"grain", "bread", "flour", "barley", "wheat", "rice", "meat", "mutton", "fleece", "wool", "hide", "pelt"} {
		if strings.Contains(n, k) {
			return true
		}
	}
	return false
}

// mgPriceFor is mgPriceMod for food, meat and herd goods only.
func mgPriceFor(g *Game, it *Item) float64 {
	if mgFoodItem(it) {
		return mgPriceMod(g)
	}
	return 1
}

// ---------------------------------------------------------------- screen

func (g *Game) mgDenList() []*mgDen {
	var out []*mgDen
	for _, d := range g.mg().Dens {
		if d.Known {
			out = append(out, d)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Key < out[j].Key })
	return out
}

func (g *Game) mgSeenList() []int {
	st := g.mg()
	var ids []int
	for id := range st.Seen {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool {
		a, b := st.Seen[ids[i]], st.Seen[ids[j]]
		if a.Day != b.Day {
			return a.Day > b.Day
		}
		return ids[i] < ids[j]
	})
	return ids
}

func mgKeys(g *Game, k string) {
	st := g.mg()
	n := 0
	switch st.tab {
	case 2:
		n = len(g.mgDenList())
	case 3:
		n = len(g.mgSeenList())
	}
	switch k {
	case "Escape", "q":
		g.Mode = "main"
	case "ArrowLeft", "N4", "h":
		st.tab, st.sel = (st.tab+3)%4, 0
	case "ArrowRight", "N6", "l", "Tab":
		st.tab, st.sel = (st.tab+1)%4, 0
	case "1", "2", "3", "4":
		st.tab, st.sel = int(k[0]-'1'), 0
	case "ArrowUp", "N8", "k":
		st.sel = maxi(0, st.sel-1)
	case "ArrowDown", "N2", "j":
		st.sel = mini(maxi(0, n-1), st.sel+1)
	case "t":
		switch st.tab {
		case 2:
			if l := g.mgDenList(); st.sel < len(l) {
				g.ceSetQuestTarget(l[st.sel].WX, l[st.sel].WY)
				g.Msg(fmt.Sprintf("{{c|The quest arrow now points to (%d,%d).}}", l[st.sel].WX, l[st.sel].WY))
			}
		case 3:
			if l := g.mgSeenList(); st.sel < len(l) {
				r := mgIdx(g.W).regs[l[st.sel]]
				g.ceSetQuestTarget(r.CX, r.CY)
				g.Msg(fmt.Sprintf("{{c|The quest arrow now points to (%d,%d).}}", r.CX, r.CY))
			}
		}
	}
}

func mgTrend(now, prev [mgGuilds]int, i int, has bool) string {
	if !has {
		return " "
	}
	switch d := now[i] - prev[i]; {
	case d >= 8:
		return "{{G|+}}"
	case d <= -8:
		return "{{R|-}}"
	}
	return "="
}

func (g *Game) mgDraw() {
	sc := g.sc
	st := g.mg()
	x, y, w, h := g.panel("Wildlife of the land")
	bottom := y + h - 1
	tabs := []string{"This land", "Movements", "Dens", "Lands seen"}
	tx := x
	for i, t := range tabs {
		c := "K"
		if i == st.tab {
			c = "Y"
		}
		tx = sc.TextB(tx, y, fmt.Sprintf("{{%s|[ %d %s ]}} ", c, i+1, t), CGray, CBgDark, x+w)
	}
	y += 1
	sc.TextB(x, y, "{{K|Left/Right tab  Up/Down choose  t point the quest arrow  Esc close}}", CGray, CBgDark, x+w)
	y += 2
	if !g.mgReady() || g.Cur == nil {
		return
	}
	line := func(s string) {
		for _, ln := range Wrap(s, w) {
			if y < bottom {
				sc.TextB(x, y, ln, CGray, CBgDark, x+w)
				y++
			}
		}
	}
	ix := mgIdx(g.W)
	day := g.Day()
	season, year := SeasonOfDay(day), mgYear(day)
	switch st.tab {
	case 0:
		r := g.mgRegHere()
		if r == nil || r.Land < 3 {
			line("{{K|There is nothing to count out here.}}")
			return
		}
		pop := st.pop(g, r.ID)
		seen := st.Seen[r.ID]
		word, off := g.mgTiming(day)
		fg := g.mgForage(r.ID, year)
		fw := "an ordinary year"
		switch {
		case fg < 0.8:
			fw = "a lean year"
		case fg > 1.05:
			fw = "a rich year"
		}
		line(fmt.Sprintf("{{Y|%s}}  %s, %s of year %d: %s for grazing.", titleCase(r.Name), BiomeName(r.Biome), mgSeasonNames[season], year+1, fw))
		line(fmt.Sprintf("The herds are %s this year (%+d days).", word, off))
		y++
		sp := ix.species(g, r)
		for i := 0; i < mgGuilds; i++ {
			if i == mgFish && pop[i] == 0 {
				continue
			}
			has := seen != nil && seen.PDay > 0
			var prev [mgGuilds]int
			if seen != nil {
				prev = seen.PPop
			}
			top := ""
			for j, e := range sp[i] {
				if j >= 3 {
					break
				}
				if j > 0 {
					top += ", "
				}
				top += strings.ToLower(e.S.Name)
			}
			words := mgWord(pop[i])
			if i >= mgPred {
				words = mgPredWords[clampi(pop[i]/25, 0, len(mgPredWords)-1)]
			}
			sc.TextB(x, y, fmt.Sprintf("{{W|%-13s}} [%s] %-14s %s {{K|%s}}", mgGuildName[i], mgBar(pop[i], 14), words, mgTrend(pop, prev, i, has), top), CGray, CBgDark, x+w)
			y++
		}
		y++
		if pop[mgHerd] > g.mgCap(r, mgHerd, season, year)*115/100 && pop[mgHerd] > 100 {
			line("{{O|The pasture is overgrazed. Grain is dearer here.}}")
		}
		if g.mgLean(r, pop) {
			line("{{R|A lean year: hungry predators are pushing toward roads and farms.}}")
		}
		if m := mgPriceMod(g); m > 1.02 {
			line(fmt.Sprintf("{{y|Food and meat cost about %d%% more here because of the herds.}}", int((m-1)*100+0.5)))
		}
		if g.mgBirdsPass(day) {
			line("{{c|Birds are passing overhead.}}")
		}
		if g.mgFishRun(day) && r.River > 0 {
			line("{{c|The fish are running in the rivers.}}")
		}
		line(fmt.Sprintf("{{K|You have killed %d herd animals, %d predators and %d apex hunters in this world; %d dens cleared, %d raids met.}}", st.Kills[mgHerd]+st.Kills[mgGame], st.Kills[mgPred], st.Kills[mgApex], st.Cleared, st.Raided))
	case 1:
		word, off := g.mgTiming(day)
		line(fmt.Sprintf("{{Y|Seasonal movements}}  The herds are %s this year (%+d days).", word, off))
		line("Herds go to their summer range in spring and their winter range in autumn. Birds pass over in spring and autumn. Fish run in the rivers in spring and autumn.")
		line(fmt.Sprintf("Herd window this year: spring day %d to %d, autumn day %d to %d.", maxi(1, off+1), off+14, maxi(1, off+1), off+14))
		if g.mgBirdsPass(day) {
			line("{{c|Right now: birds are passing.}}")
		}
		if g.mgFishRun(day) {
			line("{{c|Right now: the fish are running.}}")
		}
		y++
		n := 0
		here := g.mgRegHere()
		for i := len(st.Moves) - 1; i >= 0 && n < 14; i-- {
			m := st.Moves[i]
			from, to := ix.regs[m.From], ix.regs[m.To]
			known := st.Seen[m.From] != nil || st.Seen[m.To] != nil || here != nil && (here.ID == m.From || here.ID == m.To || g.mgNear(ix, here.ID, m.From))
			if !known {
				continue
			}
			line(fmt.Sprintf("Day %d: herds move from %s to %s, %s (about %d%% of a normal herd).", m.Day+1, from.Name, to.Name, rmDir(to.RX-from.RX, to.RY-from.RY), m.N))
			n++
		}
		if n == 0 {
			line("{{K|You have not seen any herd on the move yet. Walk the country in spring or autumn, or ask the herders.}}")
		}
	case 2:
		l := g.mgDenList()
		if len(l) == 0 {
			line("{{K|You know of no dens. Ask farmers and huntsmen where the beasts den, or find the signs in the wild.}}")
			return
		}
		st.sel = clampi(st.sel, 0, len(l)-1)
		for i, d := range l {
			if y >= bottom-9 {
				break
			}
			mark := "  "
			if i == st.sel {
				mark = "> "
			}
			sc.TextB(x, y, fmt.Sprintf("%s{{W|%s}} ({{c|%d,%d}}) %s", mark, d.Name, d.WX, d.WY, mgDenState(d)), CGray, CBgDark, x+w)
			y++
		}
		y++
		d := l[st.sel]
		reg := ix.regs[d.Reg]
		line(fmt.Sprintf("In %s, %s of the zone grid (%d,%d).", reg.Name, BiomeName(reg.Biome), d.ZX, d.ZY))
		if s := bsByID[d.Sp]; s != nil && d.Seen {
			line(fmt.Sprintf("Held by %s: %s, %d young, %d pack mates. Level about %d.", d.Alpha, strings.ToLower(s.Name), d.Brood, d.Adults, d.Lvl))
		} else {
			line("{{K|You have not seen what lives there.}}")
		}
		if d.Epoch > 0 {
			if o := bsByID[d.Prev]; o != nil {
				line(fmt.Sprintf("{{O|A new tenant: it used to hold the %s. This is the %s holder.}}", strings.ToLower(o.Name), mgOrd(d.Epoch+1)))
			}
		}
		if d.State != "active" {
			line(fmt.Sprintf("{{G|Cleared on day %d (%s). Something may move in by the next season.}}", d.Day+1, d.How))
		}
	case 3:
		l := g.mgSeenList()
		if len(l) == 0 {
			line("{{K|You have not taken note of any land yet.}}")
			return
		}
		st.sel = clampi(st.sel, 0, len(l)-1)
		for i, id := range l {
			if y >= bottom-3 {
				break
			}
			s := st.Seen[id]
			r := ix.regs[id]
			mark := "  "
			if i == st.sel {
				mark = "> "
			}
			var cols []string
			for _, gd := range []int{mgHerd, mgPred, mgApex} {
				t := mgTrend(s.Pop, s.PPop, gd, s.PDay > 0)
				w := mgWord(s.Pop[gd])
				if gd >= mgPred {
					w = mgPredWords[clampi(s.Pop[gd]/25, 0, len(mgPredWords)-1)]
				}
				cols = append(cols, fmt.Sprintf("%s %s%s", mgGuildShort[gd], w, t))
			}
			sc.TextB(x, y, fmt.Sprintf("%s{{W|%-26s}} {{K|day %d, %s}}  %s", mark, titleCase(r.Name), s.Day+1, s.Via, strings.Join(cols, "; ")), CGray, CBgDark, x+w)
			y++
		}
		line("{{K|+ and - show change since the visit before. Counts are the state on the day you saw or heard it.}}")
	}
}

func mgOrd(n int) string {
	switch n {
	case 1:
		return "first"
	case 2:
		return "second"
	case 3:
		return "third"
	}
	return fmt.Sprintf("%dth", n)
}

func mgDenState(d *mgDen) string {
	switch d.State {
	case "cleared":
		return "{{G|cleared}}"
	case "vacant":
		return "{{c|vacant}}"
	}
	if d.Epoch > 0 {
		return "{{O|new tenants}}"
	}
	return "{{R|occupied}}"
}
