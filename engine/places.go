package engine

import (
	"container/heap"
	"fmt"
	"math"
	"strings"
)

// Places and Journal+ (prefix wpl; the hook names the spec asked for keep wp: wpGlyph, wpAbilities, wpUse).
// Your own waypoints and map notes: a named, iconed, annotated parasang per planet. F10 opens the Journal+
// screen with tabs Places and Achievements. Places lists distance, bearing, and an estimated walk in
// km and days (travel.go tvKm/tvMinutes); "Travel there" auto-walks the world map along the cheapest
// land route (Dijkstra over tvMinutes, no water unless you can swim) with repeated WorldMove calls, and halts
// on any event, danger or blocked pass. State g.ext("wp"), global with a planet index per place.

type wpPlace struct {
	Name   string `json:"n"`
	WX     int    `json:"x"`
	WY     int    `json:"y"`
	Note   string `json:"t,omitempty"`
	Icon   string `json:"i,omitempty"`
	Planet int    `json:"p,omitempty"`
	Turn   int    `json:"tn,omitempty"`
}

type wplState struct {
	List []wpPlace `json:"l,omitempty"`

	sel, tab  int
	buf       string
	field     string // new | name | note
	edit      int    // list index being edited
	ret       string // mode to return to after the prompt
	ver       int
	idx       map[int]int
	idxVer    int
	idxPlanet int
	rkey      string
	route     [][2]int
	rmins     int
	rok       bool
}

func (g *Game) wpl() *wplState {
	return g.ext("wp", func() any { return &wplState{} }).(*wplState)
}

var wplIcons = []string{"◆", "★", "⌂", "☼", "♣", "▲", "Ω", "¤", "!", "?"}

const wplMax = 40

func init() {
	mainKeys["F10"] = func(g *Game) { g.Mode = "jplus"; g.wpl().sel = 0 }
	modeKeys["jplus"] = wplKeys
	modeDraws["jplus"] = wplDraw
	modeKeys["wpname"] = wplNameKey
	modeDraws["wpname"] = wplNameDraw
	sideHooks = append(sideHooks, wplSide)
}

// ---- helpers

// wplHere lists indices of this planet's places.
func (g *Game) wplHere() []int {
	st := g.wpl()
	pl := g.PlanetIndex()
	var out []int
	for i, p := range st.List {
		if p.Planet == pl {
			out = append(out, i)
		}
	}
	return out
}

func wplCompass(dx, dy float64) string {
	if dx == 0 && dy == 0 {
		return "here"
	}
	deg := math.Atan2(-dy, dx) * 180 / math.Pi
	k := int(math.Floor((deg+22.5+360)/45)) % 8
	return [8]string{"E", "NE", "N", "NW", "W", "SW", "S", "SE"}[k]
}

func (g *Game) wplKm(p *wpPlace) float64 {
	return math.Hypot(float64(p.WX-g.WX), float64(p.WY-g.WY)) * g.tvKm()
}

func (g *Game) wplDir(p *wpPlace) string {
	return wplCompass(float64(p.WX-g.WX), float64(p.WY-g.WY))
}

func (st *wplState) touch() { st.ver++; st.rkey = "" }

// wpGlyph: world-map marker for one of your places on parasang p.
// Hook: render.go drawWorld, inside `if g.Known[p] {` is too late for unknown cells, so place it right
// after the egOverlay line: `if gl2, f2, ok := g.wpGlyph(p); ok { gl, f = gl2, f2 }`.
func (g *Game) wpGlyph(p int) (rune, byte, bool) {
	st := g.wpl()
	if len(st.List) == 0 {
		return 0, 0, false
	}
	pl := g.PlanetIndex()
	if st.idx == nil || st.idxVer != st.ver || st.idxPlanet != pl {
		st.idx = map[int]int{}
		for i, q := range st.List {
			if q.Planet == pl && g.W.In(q.WX, q.WY) {
				st.idx[g.W.Idx(q.WX, q.WY)] = i
			}
		}
		st.idxVer, st.idxPlanet = st.ver, pl
	}
	i, ok := st.idx[p]
	if !ok {
		return 0, 0, false
	}
	r := []rune(st.List[i].Icon)
	if len(r) == 0 {
		return '◆', CYellow, true
	}
	return r[0], CYellow, true
}

func wplSide(g *Game) []string {
	if g.P == nil || g.W == nil {
		return nil
	}
	idx := g.wplHere()
	if len(idx) == 0 {
		return nil
	}
	st := g.wpl()
	best, bd := -1, math.MaxFloat64
	for _, i := range idx {
		p := &st.List[i]
		d := math.Hypot(float64(p.WX-g.WX), float64(p.WY-g.WY))
		if d < bd {
			best, bd = i, d
		}
	}
	p := &st.List[best]
	name := p.Name
	if len([]rune(name)) > 14 {
		name = string([]rune(name)[:14])
	}
	if bd == 0 {
		return []string{"{{c|" + name + "}} {{K|(here)}}"}
	}
	return []string{fmt.Sprintf("{{c|%s}} {{K|%s %.0f km}}", name, g.wplDir(p), g.wplKm(p))}
}

// ---- route

type wplItem struct{ d, i int }
type wplHeap []wplItem

func (h wplHeap) Len() int            { return len(h) }
func (h wplHeap) Less(a, b int) bool  { return h[a].d < h[b].d }
func (h wplHeap) Swap(a, b int)       { h[a], h[b] = h[b], h[a] }
func (h *wplHeap) Push(x interface{}) { *h = append(*h, x.(wplItem)) }
func (h *wplHeap) Pop() interface{} {
	o := *h
	n := len(o)
	it := o[n-1]
	*h = o[:n-1]
	return it
}

// wplRoute finds the cheapest land route from where you stand to (tx, ty): the cells to step through
// (excluding the start), the walking minutes, and whether there is one.
func (g *Game) wplRoute(tx, ty int) ([][2]int, int, bool) {
	w := g.W
	if !w.In(tx, ty) {
		return nil, 0, false
	}
	start, goal := w.Idx(g.WX, g.WY), w.Idx(tx, ty)
	if start == goal {
		return nil, 0, true
	}
	n := w.W * w.H
	dist := make([]int, n)
	prev := make([]int32, n)
	for i := range dist {
		dist[i], prev[i] = math.MaxInt32, -1
	}
	swim := g.canSwim()
	dist[start] = 0
	h := &wplHeap{{0, start}}
	for h.Len() > 0 {
		it := heap.Pop(h).(wplItem)
		if it.d > dist[it.i] {
			continue
		}
		if it.i == goal {
			break
		}
		x, y := it.i%w.W, it.i/w.W
		for _, d := range dirs8 {
			nx, ny := x+d[0], y+d[1]
			if !w.In(nx, ny) {
				continue
			}
			np := w.Idx(nx, ny)
			if w.IsWater(np) && !swim {
				continue
			}
			if k, _ := g.jyBlockWhy(np); k != "" && np != goal {
				continue
			}
			nd := it.d + g.tvMinutes(np) + 1
			if nd < dist[np] {
				dist[np], prev[np] = nd, int32(it.i)
				heap.Push(h, wplItem{nd, np})
			}
		}
	}
	if prev[goal] < 0 {
		return nil, 0, false
	}
	var rev [][2]int
	mins := 0
	for c := goal; c != start; c = int(prev[c]) {
		rev = append(rev, [2]int{c % w.W, c / w.W})
		mins += g.tvMinutes(c)
	}
	for i, j := 0, len(rev)-1; i < j; i, j = i+1, j-1 {
		rev[i], rev[j] = rev[j], rev[i]
	}
	return rev, mins, true
}

func (g *Game) wplRouteFor(p *wpPlace) ([][2]int, int, bool) {
	st := g.wpl()
	key := fmt.Sprintf("%d,%d>%d,%d@%d/%v", g.WX, g.WY, p.WX, p.WY, g.PlanetIndex(), g.canSwim())
	if st.rkey != key {
		st.route, st.rmins, st.rok = g.wplRoute(p.WX, p.WY)
		st.rkey = key
	}
	return st.route, st.rmins, st.rok
}

// wplTravel auto-walks the world map to a place, stopping on anything out of the ordinary.
func (g *Game) wplTravel(p *wpPlace) {
	if g.ijNoFastTravel() { // a vow against fast travel (challenges.go)
		g.Msg("Not in this vow.")
		return
	}
	if p.Planet != g.PlanetIndex() {
		g.Msg("{{R|That place lies on another world.}}")
		return
	}
	if p.WX == g.WX && p.WY == g.WY {
		g.Msg("You are already at " + p.Name + ".")
		g.Mode = "main"
		return
	}
	path, _, ok := g.wplRoute(p.WX, p.WY)
	if !ok {
		g.Msg("{{R|There is no land route to " + p.Name + ".}}")
		return
	}
	if !g.OnWorld {
		g.openWorldMap()
		if g.Mode != "world" {
			return
		}
	} else {
		g.Mode = "world"
	}
	g.Msg(fmt.Sprintf("You set out for {{C|%s}}.", p.Name))
	for _, c := range path {
		dx, dy := c[0]-g.WX, c[1]-g.WY
		if abs(dx) > 1 || abs(dy) > 1 {
			break
		}
		hp0, n0 := g.P.HP, len(g.Msgs)
		g.WorldMove(dx, dy)
		g.CX, g.CY = g.WX, g.WY
		if g.Dead || g.Mode != "world" || g.WX != c[0] || g.WY != c[1] {
			g.Msg("{{Y|You halt your journey.}}")
			return
		}
		if g.P.HP < hp0 {
			g.Msg("{{Y|You halt: you are hurt.}}")
			return
		}
		nn := len(g.Msgs) - n0
		if nn <= 0 && len(g.Msgs) >= 300 {
			nn = 4
		}
		for i := maxi(0, len(g.Msgs)-nn); i < len(g.Msgs); i++ {
			if !strings.HasPrefix(Plain(g.Msgs[i].T), "You travel ") {
				g.Msg("{{Y|You halt your journey.}}")
				return
			}
		}
	}
	g.Msg(fmt.Sprintf("{{G|You arrive at %s.}} {{K|(Enter steps in)}}", p.Name))
}

// ---- adding and editing

func (g *Game) wplDefaultName() string {
	p := g.W.Idx(g.WX, g.WY)
	if bd := g.W.burgByID[g.W.Burg[p]]; bd != nil {
		return bd.Name
	}
	if g.Cur != nil && !g.OnWorld && g.Cur.WX == g.WX && g.Cur.WY == g.WY && g.Cur.Name != "" {
		return g.Cur.Name
	}
	return fmt.Sprintf("Waypoint %d", len(g.wpl().List)+1)
}

func (g *Game) wplPrompt(field, text string, edit int, ret string) {
	st := g.wpl()
	st.field, st.buf, st.edit, st.ret = field, text, edit, ret
	g.Mode = "wpname"
}

func wplNameKey(g *Game, k string) {
	st := g.wpl()
	limit := 24
	if st.field == "note" {
		limit = 64
	}
	switch k {
	case "Escape":
		wplNameDone(g, false)
	case "Enter":
		wplNameDone(g, true)
	case "Backspace":
		if rs := []rune(st.buf); len(rs) > 0 {
			st.buf = string(rs[:len(rs)-1])
		}
	default:
		if rs := []rune(k); len(rs) == 1 && rs[0] >= 32 && rs[0] < 127 && len([]rune(st.buf)) < limit {
			st.buf += k
		}
	}
}

func wplNameDone(g *Game, ok bool) {
	st := g.wpl()
	g.Mode = st.ret
	if g.Mode == "" {
		g.Mode = "main"
	}
	text := strings.TrimSpace(st.buf)
	if !ok {
		return
	}
	switch st.field {
	case "new":
		if text == "" {
			text = g.wplDefaultName()
		}
		g.wplAdd(text)
	case "name":
		if text != "" && st.edit >= 0 && st.edit < len(st.List) {
			st.List[st.edit].Name = text
			st.touch()
		}
	case "note":
		if st.edit >= 0 && st.edit < len(st.List) {
			st.List[st.edit].Note = text
			st.touch()
		}
	}
}

func (g *Game) wplAdd(name string) {
	st := g.wpl()
	pl := g.PlanetIndex()
	for i := range st.List {
		if p := &st.List[i]; p.Planet == pl && p.WX == g.WX && p.WY == g.WY {
			p.Name = name
			st.touch()
			g.Msg(fmt.Sprintf("You rename this place {{C|%s}}.", name))
			return
		}
	}
	if len(st.List) >= wplMax {
		g.Msg("{{R|Your map has no room for more marks.}}")
		return
	}
	st.List = append(st.List, wpPlace{Name: name, WX: g.WX, WY: g.WY, Icon: wplIcons[len(st.List)%len(wplIcons)], Planet: pl, Turn: g.Turn})
	st.touch()
	g.Known[g.W.Idx(g.WX, g.WY)] = true
	g.Msg(fmt.Sprintf("You mark {{C|%s}} on your map.", name))
}

func wplNameDraw(g *Game) {
	s := g.sc
	st := g.wpl()
	title := map[string]string{"new": "Name this place", "name": "Rename this place", "note": "Note for this place"}[st.field]
	h := 7
	y0 := (MH - h) / 2
	s.Box(14, y0, 68, h, CYellow, title)
	s.TextB(16, y0+2, "{{W|"+st.buf+"}}{{Y|_}}", CGray, CBgDark, 80)
	s.TextB(16, y0+4, "{{K|type · Enter confirm · Esc cancel}}", CGray, CBgDark, 80)
}

// ---- abilities (hook in Abilities / usePower)

// wpAbilities: mark the spot you stand on, or open your places.
// Hook: powers.go Abilities(): `out = append(out, g.wpAbilities()...)`.
func (g *Game) wpAbilities() []Ability {
	if g.P == nil || g.W == nil {
		return nil
	}
	return []Ability{{ID: "wp_mark", Name: "Mark this place", Target: "self"}, {ID: "wp_places", Name: "Open your places (F10)", Target: "self"}}
}

// wpUse runs wp_ abilities. Hook: powers.go usePower(): `if strings.HasPrefix(id, "wp_") { g.wpUse(id, tx, ty); return }`.
func (g *Game) wpUse(id string, tx, ty int) {
	switch id {
	case "wp_mark":
		g.wplPrompt("new", g.wplDefaultName(), -1, "main")
	case "wp_places":
		g.Mode = "jplus"
		g.wpl().tab = 0
	}
}

// ---- Journal+ screen (F10)

func wplKeys(g *Game, k string) {
	st := g.wpl()
	switch k {
	case "Escape", "F10", "q":
		g.Mode = "main"
		return
	case "Tab", "ArrowRight":
		st.tab = (st.tab + 1) % 2
		return
	case "ArrowLeft":
		st.tab = (st.tab + 1) % 2
		return
	case "1":
		st.tab = 0
		return
	case "2":
		st.tab = 1
		return
	}
	if st.tab == 1 {
		g.achKey(k)
		return
	}
	idx := g.wplHere()
	if st.sel >= len(idx) {
		st.sel = maxi(0, len(idx)-1)
	}
	cur := -1
	if len(idx) > 0 {
		cur = idx[st.sel]
	}
	switch k {
	case "ArrowDown", "j":
		st.sel = mini(st.sel+1, maxi(0, len(idx)-1))
	case "ArrowUp", "k":
		st.sel = maxi(st.sel-1, 0)
	case "m", "M":
		g.wplPrompt("new", g.wplDefaultName(), -1, "jplus")
	case "r":
		if cur >= 0 {
			g.wplPrompt("name", st.List[cur].Name, cur, "jplus")
		}
	case "e", "n":
		if cur >= 0 {
			g.wplPrompt("note", st.List[cur].Note, cur, "jplus")
		}
	case "d", "Delete":
		if cur >= 0 {
			g.Msg("You scratch " + st.List[cur].Name + " off your map.")
			st.List = append(st.List[:cur], st.List[cur+1:]...)
			st.touch()
		}
	case "i":
		if cur >= 0 {
			p := &st.List[cur]
			j := 0
			for n, ic := range wplIcons {
				if ic == p.Icon {
					j = n + 1
				}
			}
			p.Icon = wplIcons[j%len(wplIcons)]
			st.touch()
		}
	case "Enter", "t", "g":
		if cur >= 0 {
			p := st.List[cur]
			g.Mode = "main"
			g.wplTravel(&p)
		}
	}
}

func wplDraw(g *Game) {
	s := g.sc
	st := g.wpl()
	s.Fill(0, 0, SW, SH, CBgDark)
	s.Box(0, 0, SW, SH, CDarkCyan, "Journal+")
	tx := 2
	for i, t := range []string{"Places", "Achievements"} {
		c := "K"
		if st.tab == i {
			c = "Y"
		}
		tx = s.TextB(tx, 1, fmt.Sprintf("{{%s|[ %s ]}} ", c, t), CGray, CBgDark, SW-1)
	}
	if st.tab == 1 {
		g.achDraw(3, 3, SW-6, SH-4)
		return
	}
	idx := g.wplHere()
	if st.sel >= len(idx) {
		st.sel = maxi(0, len(idx)-1)
	}
	left := 46
	s.TextB(3, 3, fmt.Sprintf("{{W|%d}} marked on {{c|%s}}   {{K|you stand at %d,%d}}", len(idx), g.unPlanetName(), g.WX, g.WY), CGray, CBgDark, 2+left)
	y := 5
	if len(idx) == 0 {
		s.TextB(3, y, "{{K|No places marked yet. Press m to mark where you stand,}}", CGray, CBgDark, 2+left)
		s.TextB(3, y+1, "{{K|or use the ability Mark this place.}}", CGray, CBgDark, 2+left)
	}
	for n, i := range idx {
		if y >= SH-4 {
			break
		}
		p := &st.List[i]
		bg := CBgDark
		if n == st.sel {
			bg = CBgSel
		}
		s.Fill(3, y, left, 1, bg)
		ic := []rune(p.Icon + "◆")[0]
		s.Put(3, y, ic, CYellow, bg)
		nm := p.Name
		if len([]rune(nm)) > 22 {
			nm = string([]rune(nm)[:22])
		}
		d := fmt.Sprintf("%s %.0f km", g.wplDir(p), g.wplKm(p))
		if p.WX == g.WX && p.WY == g.WY {
			d = "here"
		}
		s.TextB(5, y, fmt.Sprintf("{{W|%-22s}} {{K|%s}}", nm, d), CGray, bg, 3+left)
		y++
	}
	// details
	dx := 3 + left + 3
	dw := SW - dx - 3
	if len(idx) > 0 {
		p := &st.List[idx[st.sel]]
		y = 5
		s.TextB(dx, y, "{{Y|"+p.Name+"}}", CGray, CBgDark, dx+dw)
		y += 2
		s.TextB(dx, y, fmt.Sprintf("{{K|Region}} %d,%d  {{K|Icon}} {{Y|%s}}", p.WX, p.WY, p.Icon), CGray, CBgDark, dx+dw)
		y++
		if bd := g.W.burgByID[g.W.Burg[g.W.Idx(p.WX, p.WY)]]; bd != nil {
			s.TextB(dx, y, "{{K|Settlement}} "+bd.Name, CGray, CBgDark, dx+dw)
			y++
		}
		y++
		if p.WX == g.WX && p.WY == g.WY {
			s.TextB(dx, y, "{{G|You are here.}}", CGray, CBgDark, dx+dw)
		} else {
			km := g.wplKm(p)
			s.TextB(dx, y, fmt.Sprintf("{{K|Bearing}} %s  {{K|distance}} %.0f km", g.wplDir(p), km), CGray, CBgDark, dx+dw)
			y++
			if path, mins, ok := g.wplRouteFor(p); ok {
				s.TextB(dx, y, fmt.Sprintf("{{K|On foot}} about {{W|%s}} {{K|over}} %d regions {{K|%s}}", tvDuration(mins), len(path), g.trHow()), CGray, CBgDark, dx+dw)
			} else {
				s.TextB(dx, y, "{{R|No land route from here.}}", CGray, CBgDark, dx+dw)
			}
		}
		y += 2
		if p.Note != "" {
			for _, l := range Wrap("{{c|"+p.Note+"}}", dw) {
				s.TextB(dx, y, l, CGray, CBgDark, dx+dw)
				y++
			}
		} else {
			s.TextB(dx, y, "{{K|(no note: press e)}}", CGray, CBgDark, dx+dw)
		}
	}
	s.TextB(3, SH-3, "{{K|↑↓ choose · m mark here · r rename · e note · i icon · d delete · Enter travel there · Tab achievements · Esc}}", CGray, CBgDark, SW-2)
}
