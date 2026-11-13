package engine

import (
	"fmt"
	"strings"
)

// Genre signature systems, part 2 (prefix gd): the biopunk breeding vat (gen_bio.go), solarpunk gardens
// and energy (gen_solar.go), steam/diesel/atom boilers, fuel and automatons (gen_steam.go) and the
// fantasy/mythic gods that answer and omens (gen_myth.go). This file holds what they share: one saved
// state (g.ext("gd"), planet-local), a small row-menu screen used by every device (modes gdv, gdm, gdx),
// device items placed by zone hooks, and bundle helpers for seeds, fuel, cells and compost.
//
// Devices are items of Kind "gddev" (Effect = device type) sitting on a TMachine tile; bumping one runs
// gdBump (one-line hook in MoveP). All placement uses side rngs (seed:gd:...), never g.R, and only in the
// genres that own the system, so every other genre generates byte-identically.

type gdState struct {
	Vats   map[string]*gdVat  `json:"vt,omitempty"`
	Plots  map[string]*gdPlot `json:"pl,omitempty"`
	Grids  map[string]*gdGrid `json:"gr,omitempty"`
	Towns  map[int]int        `json:"tw,omitempty"` // burg id -> town grid level 0..5
	Auto   *gdAuto            `json:"au,omitempty"`
	Press  map[string]int     `json:"pr,omitempty"` // device key -> turn the press stays warm until
	Faith  map[int]*gdFaith   `json:"fa,omitempty"` // religion id -> favour record
	Patron int                `json:"pa,omitempty"` // religion id you follow
	Done   map[int]bool       `json:"od,omitempty"` // omen days already fulfilled
	Read   map[int]string     `json:"or,omitempty"` // omen day -> the reading you paid for
	Rain   int                `json:"rn,omitempty"` // a miracle's rain lasts until this turn
	Firsts map[string]bool    `json:"fs,omitempty"`
	Cut    map[string]int     `json:"ct,omitempty"` // forage cooldowns: "zone:tile" -> day
	Home   map[string]bool    `json:"hm,omitempty"` // homes already fitted out
	Ctx    map[string]int     `json:"cx,omitempty"` // misc counters (day deeds, grid books)
	Sold   map[string]int     `json:"sl,omitempty"` // seed trade ledger by day
	Extra  map[string]string  `json:"ex,omitempty"`

	ui     *gdUI
	vsel   gdVatSel
	wsel   gdWorkSel
	rainOn bool
}

func (g *Game) gd() *gdState {
	st := g.ext("gd", func() any { return &gdState{} }).(*gdState)
	if st.Vats == nil {
		st.Vats = map[string]*gdVat{}
	}
	if st.Plots == nil {
		st.Plots = map[string]*gdPlot{}
	}
	if st.Grids == nil {
		st.Grids = map[string]*gdGrid{}
	}
	if st.Towns == nil {
		st.Towns = map[int]int{}
	}
	if st.Press == nil {
		st.Press = map[string]int{}
	}
	if st.Faith == nil {
		st.Faith = map[int]*gdFaith{}
	}
	if st.Done == nil {
		st.Done = map[int]bool{}
	}
	if st.Read == nil {
		st.Read = map[int]string{}
	}
	if st.Firsts == nil {
		st.Firsts = map[string]bool{}
	}
	if st.Cut == nil {
		st.Cut = map[string]int{}
	}
	if st.Home == nil {
		st.Home = map[string]bool{}
	}
	if st.Ctx == nil {
		st.Ctx = map[string]int{}
	}
	if st.Sold == nil {
		st.Sold = map[string]int{}
	}
	return st
}

func init() {
	unLocalKeys = append(unLocalKeys, "gd")
	for _, m := range []string{"gdv", "gdm", "gdx"} {
		modeKeys[m] = gdKeys
		modeDraws[m] = gdDraw
	}
	enterHooks = append(enterHooks, gdEnter)
}

// gdFirst records a once-only deed ("firsts") and writes it to the chronicle.
func (g *Game) gdFirst(key, text string) {
	st := g.gd()
	if st.Firsts[key] {
		return
	}
	st.Firsts[key] = true
	g.Msg("{{Y|" + text + "}}")
	if g.Cur != nil && g.W != nil {
		g.deed("discovery", fmt.Sprintf("The wanderer %s: %s", g.P.Name, lowerFirst(text)), 0, 0)
	}
}

// gdSkill is the care score of your hands: tinkering, survival and lore ranks, 0..9.
func (g *Game) gdSkill() int {
	return clampi(g.Skills["tinkering"], 0, 3) + clampi(g.Skills["survival"], 0, 3) + clampi(g.Skills["lore"], 0, 3)
}

func (g *Game) gdRng(tag string, parts ...any) *Rng {
	return NewRng(fmt.Sprintf("%s:gd:%s:%s", g.Seed, tag, fmt.Sprint(parts...)))
}

func gdBar(n, max, w int) string {
	if max <= 0 {
		max = 1
	}
	n = clampi(n, 0, max)
	f := n * w / max
	return strings.Repeat("█", f) + strings.Repeat("░", w-f)
}

// gdHours prints a span of minutes as "2d 5h" or "40 min".
func gdHours(min int) string {
	if min < 60 {
		return fmt.Sprintf("%d min", maxi(min, 0))
	}
	h := min / 60
	if h >= 24 {
		return fmt.Sprintf("%dd %dh", h/24, h%24)
	}
	return fmt.Sprintf("%dh", h)
}

// gdGenre reports whether the genre owns any gd system.
func gdGenre(g *Game) bool {
	switch g.Genre {
	case "biopunk", "solarpunk", "steampunk", "dieselpunk", "atompunk", "fantasy", "mythic":
		return true
	}
	return false
}

func gdSteamy(g *Game) bool {
	return g.Genre == "steampunk" || g.Genre == "dieselpunk" || g.Genre == "atompunk"
}

// ---- the row-menu screen --------------------------------------------------------------

type gdRow struct {
	Label string
	Do    func()
	Off   bool
}

type gdUI struct {
	Title string
	Mode  string
	Lines []string
	Rows  []gdRow
	Sel   int
	build func(u *gdUI)
	back  func()
}

func (u *gdUI) line(s string) { u.Lines = append(u.Lines, s) }
func (u *gdUI) gap()          { u.Lines = append(u.Lines, "") }
func (u *gdUI) row(label string, do func()) {
	u.Rows = append(u.Rows, gdRow{Label: label, Do: do})
}
func (u *gdUI) off(label string) {
	u.Rows = append(u.Rows, gdRow{Label: "{{K|" + Plain(label) + "}}", Off: true})
}

// gdOpen shows a screen built by build; Esc (or the back func) leaves it.
func (g *Game) gdOpen(mode, title string, build func(u *gdUI)) *gdUI {
	st := g.gd()
	u := &gdUI{Title: title, Mode: mode, build: build}
	st.ui = u
	build(u)
	g.Mode = mode
	return u
}

func (g *Game) gdRefresh() {
	u := g.gd().ui
	if u == nil || g.Mode != u.Mode {
		return
	}
	sel := u.Sel
	u.Lines, u.Rows = nil, nil
	u.build(u)
	u.Sel = clampi(sel, 0, maxi(0, len(u.Rows)-1))
}

func (g *Game) gdClose() {
	g.gd().ui = nil
	g.Mode = "main"
}

func gdKeys(g *Game, k string) {
	u := g.gd().ui
	if u == nil {
		g.Mode = "main"
		return
	}
	run := func(i int) {
		if i < 0 || i >= len(u.Rows) || u.Rows[i].Off || u.Rows[i].Do == nil {
			return
		}
		u.Sel = i
		u.Rows[i].Do()
		if g.Dead {
			return
		}
		if g.gd().ui == u && g.Mode == u.Mode {
			g.gdRefresh()
		}
	}
	if i := letterIndex(k); i >= 0 {
		run(i)
		return
	}
	switch k {
	case "Escape":
		if u.back != nil {
			u.back()
			return
		}
		g.gdClose()
	case "ArrowUp":
		u.Sel = maxi(0, u.Sel-1)
	case "ArrowDown":
		u.Sel = clampi(u.Sel+1, 0, maxi(0, len(u.Rows)-1))
	case "Enter", " ":
		run(u.Sel)
	}
}

func gdDraw(g *Game) {
	u := g.gd().ui
	if u == nil {
		return
	}
	x, y, w, h := g.panel(u.Title)
	s := g.sc
	y0, bottom := y, y+h-1
	for _, l := range u.Lines {
		for _, wl := range Wrap(l, w) {
			if y < bottom-3 {
				s.TextB(x, y, wl, CGray, CBgDark, x+w)
				y++
			}
		}
	}
	y++
	room := bottom - 1 - y
	if room < 1 {
		room = 1
	}
	top := maxi(0, u.Sel-room+1)
	for i := top; i < len(u.Rows) && i < top+room; i++ {
		bg := CBgDark
		if i == u.Sel {
			bg = CBgSel
		}
		s.Fill(x, y, w, 1, bg)
		letter := '?'
		if i < 26 {
			letter = rune('a' + i)
		}
		s.TextB(x, y, fmt.Sprintf("{{W|%c)}} %s", letter, u.Rows[i].Label), CGray, bg, x+w)
		y++
	}
	_ = y0
	s.TextB(x, bottom, "{{K|letters or Enter choose, Esc leave}}", CGray, CBgDark, x+w)
}

// ---- device items ----------------------------------------------------------------------

type gdDevDef struct {
	Name string
	G    rune
	F    byte
	Desc string
}

var gdDevDefs = map[string]gdDevDef{
	"vat":      {"breeding vat", 'Θ', CGreen, "A glass-walled tank of warm amber nutrient broth, wired for samples."},
	"plot":     {"garden plot", '"', CGreen, "A raised bed of dark soil, edged with salvaged tile."},
	"solar":    {"solar panel", 'Σ', CYellow, "A tilted lattice of blue photovoltaic glass."},
	"wind":     {"wind turbine", 'λ', CCyan, "A slim turbine on a mast, blades turning."},
	"battery":  {"battery bank", 'Π', COrange, "Stacked cells in a weatherproof cabinet, with a charge gauge."},
	"lamp":     {"street lamp", '!', CDarkGray, "A lamp post wired to the grid; it burns when it is dark and the bank has charge."},
	"purifier": {"water purifier", 'Ö', CBlue, "A humming still of sand, reed-bed and ultraviolet tubes."},
	"lift":     {"freight lift", 'Δ', CWhite, "A counterweighted platform on a mast, run by the grid."},
	"bench":    {"workshop bench", 'Ħ', COrange, "A vice, a lathe and a rack of gauges; room to build and mend machines."},
	"press":    {"steam press", 'Ψ', CRed, "A hissing platen press that squeezes metal and wood into better stock."},
}

var gdDevOpen = map[string]func(g *Game, d *Item){}

func gdDevKey(zn *Zone, d *Item) string { return fmt.Sprintf("%s:%d,%d", zn.Key, d.X, d.Y) }

func gdDevAt(zn *Zone, x, y int) *Item {
	for _, it := range zn.Items {
		if it.Kind == "gddev" && it.X == x && it.Y == y {
			return it
		}
	}
	return nil
}

func (g *Game) gdDevs(zn *Zone, kind string) []*Item {
	var out []*Item
	for _, it := range zn.Items {
		if it.Kind == "gddev" && (kind == "" || it.Effect == kind) {
			out = append(out, it)
		}
	}
	return out
}

func (g *Game) gdPlaceDev(zn *Zone, kind string, x, y, q int) *Item {
	def := gdDevDefs[kind]
	it := &Item{ID: g.newID(), Kind: "gddev", Effect: kind, Name: def.Name, Glyph: def.G, F: def.F, X: x, Y: y,
		Lore: def.Desc, Extra: map[string]int{"gdq": q}}
	zn.Items = append(zn.Items, it)
	zn.Set(x, y, TMachine)
	return it
}

var gdBadTiles = map[Terrain]bool{TDoor: true, TDoorOpen: true, TDown: true, TUp: true, TBed: true, TPortal: true, TTrap: true,
	TBridge: true, TUpdraft: true, TLava: true, TChasm: true, TVoid: true, TGateArch: true, TGateClosed: true, TShallow: true, TDeep: true}

// gdOpenSpot: a walkable, empty tile with open ground around it (never blocks a corridor).
func gdOpenSpot(zn *Zone, x, y int) bool {
	if !zn.In(x, y) || !zn.Walkable(x, y) || gdBadTiles[zn.At(x, y)] || zn.ActorAt(x, y) != nil || len(zn.ItemsAt(x, y)) > 0 || zn.Notes[y*MW+x] != "" {
		return false
	}
	open := 0
	for _, d := range dirs8 {
		nx, ny := x+d[0], y+d[1]
		if !zn.In(nx, ny) {
			return false
		}
		t := zn.At(nx, ny)
		if t == TDoor || t == TDoorOpen || t == TDown || t == TUp {
			return false
		}
		if zn.Walkable(nx, ny) && zn.WaterAt(nx, ny) < 2 {
			open++
		}
	}
	return open >= 6
}

// gdFreeSpot picks a tile for a device from a side rng; near limits the candidates.
func (g *Game) gdFreeSpot(zn *Zone, r *Rng, near func(x, y int) bool) (int, int, bool) {
	for i := 0; i < 500; i++ {
		x, y := r.Int(2, MW-3), r.Int(2, MH-3)
		if gdOpenSpot(zn, x, y) && (near == nil || near(x, y)) {
			return x, y, true
		}
	}
	return 0, 0, false
}

// gdFreeNear finds the closest open tile to (cx,cy) without drawing any rng.
func (g *Game) gdFreeNear(zn *Zone, cx, cy, maxR int) (int, int, bool) {
	for r := 1; r <= maxR; r++ {
		for dy := -r; dy <= r; dy++ {
			for dx := -r; dx <= r; dx++ {
				if abs(dx) != r && abs(dy) != r {
					continue
				}
				x, y := cx+dx, cy+dy
				if x == g.P.X && y == g.P.Y {
					continue
				}
				if h := g.homeHere(); h != nil && x == h.SX && y == h.SY {
					continue
				}
				if gdOpenSpot(zn, x, y) {
					return x, y, true
				}
			}
		}
	}
	return 0, 0, false
}

// gdTown: a town or outskirts surface zone of a living burg (where gd zone hooks place things).
func gdTown(g *Game, zn *Zone) bool {
	return g.W != nil && zn.Z == 0 && zn.Burg >= 0 && (zn.Kind == "town" || zn.Kind == "outskirts") && g.W.burgByID[zn.Burg] != nil
}

func (g *Game) gdHomeEquip(zn *Zone) {
	h := g.homeHere()
	if h == nil {
		return
	}
	st := g.gd()
	if st.Home[zn.Key] {
		return
	}
	var kinds []string
	switch g.Genre {
	case "biopunk":
		kinds = []string{"vat"}
	case "solarpunk":
		kinds = []string{"plot", "plot", "solar", "battery", "lamp"}
	case "steampunk", "dieselpunk", "atompunk":
		kinds = []string{"bench", "press"}
	}
	if len(kinds) == 0 {
		return
	}
	st.Home[zn.Key] = true
	for _, k := range kinds {
		if x, y, ok := g.gdFreeNear(zn, h.X, h.Y, 8); ok {
			g.gdPlaceDev(zn, k, x, y, 2)
		}
	}
}

// gdBump is the one hook for MoveP: devices, forage and altars. True when the bump was used.
func gdBump(g *Game, x, y int) bool {
	if g == nil || g.Cur == nil || g.W == nil || g.Dead || !gdGenre(g) {
		return false
	}
	zn := g.Cur
	if d := gdDevAt(zn, x, y); d != nil {
		if f := gdDevOpen[d.Effect]; f != nil {
			f(g, d)
			return true
		}
		return false
	}
	switch g.Genre {
	case "biopunk":
		gdBioForage(g, zn, x, y)
	case "solarpunk":
		gdSolarForage(g, zn, x, y)
	case "fantasy", "mythic":
		if zn.At(x, y) == TAltar {
			g.gdAltar(zn, x, y)
			return true
		}
	}
	return false
}

func gdEnter(g *Game, zn *Zone) {
	if !gdGenre(g) || g.W == nil {
		return
	}
	g.gdHomeEquip(zn)
}

// ---- bundles: seeds, fuel, cells, compost --------------------------------------------

var gdBundleKinds = map[string]bool{"gdseed": true, "gdfuel": true, "gdcell": true, "gdcomp": true}

// gdGive adds a bundle item (Count units, Value total) to the pack, merging with the same kind.
func (g *Game) gdGive(it *Item) {
	if it.Count < 1 {
		it.Count = 1
	}
	if it.Weight == 0 {
		it.Weight = 1
	}
	for _, o := range g.P.Inv {
		if o.Kind == it.Kind && o.Effect == it.Effect && o.Name == it.Name {
			o.Count = maxi(o.Count, 1) + it.Count
			o.Value += it.Value
			return
		}
	}
	g.P.Inv = append(g.P.Inv, it)
}

func (g *Game) gdCount(kind, eff string) int {
	n := 0
	for _, it := range g.P.Inv {
		if it.Kind == kind && (eff == "" || it.Effect == eff) {
			n += maxi(it.Count, 1)
		}
	}
	return n
}

// gdTake removes n units of kind/effect from the pack; false and nothing taken when short.
func (g *Game) gdTake(kind, eff string, n int) bool {
	if g.gdCount(kind, eff) < n {
		return false
	}
	for i := 0; i < len(g.P.Inv) && n > 0; {
		it := g.P.Inv[i]
		if it.Kind != kind || (eff != "" && it.Effect != eff) {
			i++
			continue
		}
		c := maxi(it.Count, 1)
		take := mini(c, n)
		unit := it.Value / c
		it.Count = c - take
		it.Value -= unit * take
		n -= take
		if it.Count <= 0 {
			g.P.Inv = append(g.P.Inv[:i], g.P.Inv[i+1:]...)
			continue
		}
		i++
	}
	return true
}

// gdTakeOne removes one unit of a stackable pack item (food, ingredient) or the item itself.
func (g *Game) gdTakeOne(it *Item) {
	if it.Count > 1 {
		it.Count--
		return
	}
	g.removeInv(it)
}

// gdBitsLine prints a bit cost like "3A 2B".
func gdCostLine(cost map[int]int) string { return costText(cost, 0) }

// gdPayBits spends crafting bits when you hold them all.
func (g *Game) gdPayBits(cost map[int]int) bool {
	cs := g.craftSt()
	if !cs.canPay(cost) {
		return false
	}
	cs.pay(cost)
	return true
}

func (g *Game) gdHasBits(cost map[int]int) bool { return g.craftSt().canPay(cost) }

func (g *Game) gdAddBits(cost map[int]int, pct int) {
	cs := g.craftSt()
	for t, n := range cost {
		cs.Bits[bitKey(t)] += n * pct / 100
	}
}

// gdFindActor looks for an actor by id in every zone.
func (g *Game) gdFindActor(id int) (*Actor, *Zone) {
	if g.Cur != nil {
		for _, a := range g.Cur.Actors {
			if a.ID == id && a.HP > 0 {
				return a, g.Cur
			}
		}
	}
	for _, z := range g.Zones {
		for _, a := range z.Actors {
			if a.ID == id && a.HP > 0 {
				return a, z
			}
		}
	}
	return nil, nil
}

// gdSeasonName is the current season of the calendar.
func (g *Game) gdSeasonName() string { return SeasonNames[g.Season()] }

// gdParasang is the world parasang of the current zone (-1 when off the map).
func (g *Game) gdParasang() int {
	if g.W == nil || g.Cur == nil || !g.W.In(g.Cur.WX, g.Cur.WY) {
		return -1
	}
	return g.W.Idx(g.Cur.WX, g.Cur.WY)
}

// gdCurrencyOpt prices an option with the local money.
func (g *Game) gdPrice(base int) string { return g.cuFormat(base) }

// gdPay spends base money units; it says so when you cannot afford it.
func (g *Game) gdPay(base int) bool {
	if base <= 0 {
		return true
	}
	if !g.cuCanSpend(base) {
		g.Msg("{{y|You cannot afford " + g.cuFormat(base) + ".}}")
		return false
	}
	return g.cuSpend(base)
}
