package engine

import (
	"fmt"
	"strings"
)

// Dungeon mechanics (prefix dr). Room-level puzzle and mechanic set pieces for dungeon levels (and
// a few that fit tower floors): pressure plates (weight, a pushed block, or a timer), levers (one is
// true), rune doors with the answer carved elsewhere on the level, a turning hall, treasure rooms that
// flood or collapse after the treasure is lifted, braziers to light in order, mirror chambers that call
// up shades of you and your companions, gallery statues that move when unseen, idols and altars that
// trade an offering for a boon with a curse risk, arenas with waves, and a plank bridge over a chasm.
// Each is a small state machine in g.ext("dr"), keyed by zone key, placed by a zone hook with the side
// rng seed:dr:zoneKey only (carved chambers are additions; props sit on free floor tiles that cannot
// cut a path; stairs untouched). Interaction: stand next to a prop and a lettered-choice screen opens
// (mode "dr", once per approach), or use the palette "Use nearby mechanism"; drBump(g,x,y) opens the
// same screen from a bump if the coordinator wires it (see the report). Failure never kills: damage is
// capped at HP-1. Boss fight gimmicks live in delvemech_boss.go. State is planet-local ("dr").

func init() {
	unLocalKeys = append(unLocalKeys, "dr")
	zoneHooks = append(zoneHooks, drZone)
	enterHooks = append(enterHooks, drEnter)
	turnHooks = append(turnHooks, drTurn)
	killHooks = append(killHooks, drKill)
	modeKeys["dr"] = drKeys
	modeDraws["dr"] = func(g *Game) { g.drDraw() }
	paEntries = append(paEntries, paEntry{Name: "Use nearby mechanism", Keywords: "lever plate rune brazier idol statue pillar puzzle dungeon switch",
		Cat: "Action", Label: "dungeon",
		Cond: func(g *Game) bool { _, _, ok := g.drNearProp(); return ok },
		Func: func(g *Game) {
			if f, i, ok := g.drNearProp(); ok {
				g.drOpen(f, i)
			}
		}})
}

type drFeat struct {
	ID   string `json:"i"`
	Kind string `json:"k"`
	X    int    `json:"x,omitempty"`
	Y    int    `json:"y,omitempty"`
	GX   int    `json:"gx,omitempty"`
	GY   int    `json:"gy,omitempty"`
	St   int    `json:"s,omitempty"` // 0 waiting, 1 running, 2 solved, 3 spent
	V    int    `json:"v,omitempty"`
	N    int    `json:"n,omitempty"`
	M    int    `json:"m,omitempty"`
	T0   int    `json:"t,omitempty"`
	Dur  int    `json:"d,omitempty"`
	Seq  []int  `json:"q,omitempty"`
	In   []int  `json:"in,omitempty"`
	Pts  []int  `json:"p,omitempty"`
	Aux  []int  `json:"a,omitempty"`
	RX   int    `json:"rx,omitempty"`
	RY   int    `json:"ry,omitempty"`
	RW   int    `json:"rw,omitempty"`
	RH   int    `json:"rh,omitempty"`
	Lv   int    `json:"l,omitempty"`
	Tag  string `json:"g,omitempty"`
}

type drUI struct {
	f   *drFeat
	idx int
	res []string
}

type drState struct {
	F      map[string][]*drFeat `json:"f,omitempty"`
	B      map[string]*drBoss   `json:"b,omitempty"`
	Solved int                  `json:"sv,omitempty"`
	Failed int                  `json:"fl,omitempty"`

	ui   drUI
	near map[string]bool
}

func (g *Game) dr() *drState {
	s := g.ext("dr", func() any { return &drState{F: map[string][]*drFeat{}, B: map[string]*drBoss{}} }).(*drState)
	if s.F == nil {
		s.F = map[string][]*drFeat{}
	}
	if s.B == nil {
		s.B = map[string]*drBoss{}
	}
	if s.near == nil {
		s.near = map[string]bool{}
	}
	return s
}

// ---------------------------------------------------------------- small helpers

func drPack(x, y int) int { return y*MW + x }

func drTier(zn *Zone) int { return clampi(zn.Level/3+1, 1, 5) }

func drDC(zn *Zone, base int) int { return clampi(base+zn.Level/4, base, base+6) }

// drHurt hurts the player without ever killing.
func (g *Game) drHurt(n int, cause string) {
	if n > g.P.HP-1 {
		n = g.P.HP - 1
	}
	if n > 0 {
		g.hurt(g.P, n, nil, cause)
	}
}

func (g *Game) drSay(s string) {
	st := g.dr()
	st.ui.res = append(st.ui.res, s)
	g.Msg(s)
}

func drKindDef(id string) drKindDef2 {
	for _, d := range drKindDefs {
		if d.ID == id {
			return drKindDef2{d.Name, d.Carve, d.Hint}
		}
	}
	return drKindDef2{id, false, ""}
}

type drKindDef2 struct {
	Name  string
	Carve bool
	Hint  string
}

type drRoom struct {
	X, Y, W, H int
	DX, DY     int // door tile in the wall ring
	Side       int // 0 top 1 right 2 bottom 3 left
	Floor      Terrain
}

var drDir4 = [4][2]int{{0, -1}, {1, 0}, {0, 1}, {-1, 0}}

// drCarve cuts a w by h chamber from solid rock, joined to the level by a short corridor that
// starts at a door tile (left as rock for the builder to set). Only additions: the old level is untouched.
func drCarve(zn *Zone, r *Rng, w, h int, sides int) (drRoom, bool) {
	for try := 0; try < 150; try++ {
		x, y := r.Int(3, MW-w-4), r.Int(3, MH-h-4)
		clear := true
		for yy := y - 1; yy <= y+h && clear; yy++ {
			for xx := x - 1; xx <= x+w; xx++ {
				if zn.At(xx, yy).D().Walk {
					clear = false
					break
				}
			}
		}
		if !clear {
			continue
		}
		first := r.Int(0, 3)
		for s := 0; s < 4; s++ {
			side := (first + s) % 4
			if sides&(1<<uint(side)) == 0 {
				continue
			}
			var sx, sy int
			switch side {
			case 0:
				sx, sy = x+w/2, y-1
			case 1:
				sx, sy = x+w, y+h/2
			case 2:
				sx, sy = x+w/2, y+h
			default:
				sx, sy = x-1, y+h/2
			}
			d := drDir4[side]
			var path [][2]int
			cx, cy := sx, sy
			var fl Terrain
			found := false
			for k := 0; k < 8; k++ {
				if cx < 1 || cy < 1 || cx > MW-2 || cy > MH-2 {
					break
				}
				t := zn.At(cx, cy)
				if k > 0 && t.D().Walk {
					if evFloors[t] && t != TRubble && t != TBones {
						fl, found = t, true
					}
					break
				}
				if t.D().Walk {
					break
				}
				path = append(path, [2]int{cx, cy})
				cx += d[0]
				cy += d[1]
			}
			if !found {
				continue
			}
			for yy := y; yy < y+h; yy++ {
				for xx := x; xx < x+w; xx++ {
					zn.Set(xx, yy, fl)
				}
			}
			for _, p := range path[1:] {
				zn.Set(p[0], p[1], fl)
			}
			return drRoom{X: x, Y: y, W: w, H: h, DX: sx, DY: sy, Side: side, Floor: fl}, true
		}
	}
	return drRoom{}, false
}

// drSpot finds a free plain floor tile for a prop. safe: the tile may become solid without cutting a path.
func drSpot(zn *Zone, r *Rng, safe bool, away [][2]int, minD int) (int, int, bool) {
	for i := 0; i < 400; i++ {
		x, y := r.Int(3, MW-4), r.Int(3, MH-4)
		if !evPlain(zn, x, y) || len(zn.ItemsAt(x, y)) > 0 || zn.ActorAt(x, y) != nil || evNearStair(zn, x, y, 2) || evWalkN(zn, x, y) < 5 {
			continue
		}
		if safe && !evSafeBlock(zn, x, y) {
			continue
		}
		ok := true
		for _, a := range away {
			if maxi(abs(a[0]-x), abs(a[1]-y)) < minD {
				ok = false
				break
			}
		}
		if ok {
			return x, y, true
		}
	}
	return 0, 0, false
}

// drDist: walking distances from a tile over walkable ground.
func drDist(zn *Zone, sx, sy int) []int {
	d := make([]int, MW*MH)
	for i := range d {
		d[i] = -1
	}
	d[sy*MW+sx] = 0
	q := []int{sy*MW + sx}
	for len(q) > 0 {
		i := q[0]
		q = q[1:]
		x, y := i%MW, i/MW
		for _, dd := range dirs8 {
			nx, ny := x+dd[0], y+dd[1]
			if !zn.In(nx, ny) {
				continue
			}
			ni := ny*MW + nx
			t := zn.At(nx, ny)
			if d[ni] >= 0 || !t.D().Walk || t == TLava || t == TChasm {
				continue
			}
			d[ni] = d[i] + 1
			q = append(q, ni)
		}
	}
	return d
}

func drItemInZone(zn *Zone, id int) bool {
	for _, it := range zn.Items {
		if it.ID == id {
			return true
		}
	}
	return false
}

func (g *Game) drAddItem(zn *Zone, it *Item, x, y int) {
	it.X, it.Y = x, y
	zn.Items = append(zn.Items, it)
}

func (g *Game) drFoe(zn *Zone, r *Rng, theme string, lvl int) *Actor {
	foes := themes[theme].foes
	id := "skeleton"
	if len(foes) > 0 {
		id = Pick(r, foes)
	}
	return g.Spawn(id, lvl, r)
}

// drReward drops loot scaled to depth around (x,y): n finds, coin, sometimes an augment.
func (g *Game) drReward(zn *Zone, f *drFeat, x, y, n int) {
	r := NewRng(g.W.Seed + ":dr:rw:" + f.ID)
	tier := clampi(drTier(zn)+f.Lv/3, 1, 5)
	for i := 0; i < n; i++ {
		it := g.Loot(r, tier)
		bx, by := x, y
		if i > 0 {
			if nx, ny, ok := evBeside(zn, x, y); ok {
				bx, by = nx, ny
			}
		}
		g.drAddItem(zn, it, bx, by)
	}
	g.P.Gold += 20*zn.Level + r.Int(5, 40)
	if au := g.augLoot(zn.Key+":dr:"+f.ID, 0.2); au != nil {
		if nx, ny, ok := evBeside(zn, x, y); ok {
			g.drAddItem(zn, au, nx, ny)
		}
	}
	g.gainXP(25 + 8*zn.Level)
}

func (g *Game) drSolved(f *drFeat, line string) {
	f.St = 2
	st := g.dr()
	st.Solved++
	g.Msg("{{G|" + line + "}}")
	if st.Solved == 1 {
		g.deed("quest", fmt.Sprintf("The wanderer %s solved the first of the old mechanisms under the world.", g.P.Name), 0, 0)
	}
}

func (g *Game) drSetDoor(f *drFeat, open bool) {
	zn := g.Cur
	if open {
		zn.Set(f.GX, f.GY, TDoorOpen)
		return
	}
	if (g.P.X == f.GX && g.P.Y == f.GY) || zn.ActorAt(f.GX, f.GY) != nil {
		return
	}
	zn.Set(f.GX, f.GY, TMetalWall)
}

func drInRect(f *drFeat, x, y int) bool {
	return x >= f.RX && x < f.RX+f.RW && y >= f.RY && y < f.RY+f.RH
}

// ---------------------------------------------------------------- zone hook

func drZone(g *Game, zn *Zone, _ *Rng) {
	if g.W == nil || zn == nil || zn.Underwater || zn.Sky && zn.Z >= 0 {
		return
	}
	s, lv := evSiteLevel(g, zn)
	if s == nil {
		return
	}
	r := NewRng(g.W.Seed + ":dr:" + zn.Key)
	p := minf(0.28+0.02*float64(lv), 0.5)
	if !r.Chance(p) {
		return
	}
	n := 1
	if r.Chance(0.12) {
		n = 2
	}
	carveOK := zn.Kind == "dungeon" && zn.Z > 0
	st := g.dr()
	used := map[string]bool{}
	for i := 0; i < n; i++ {
		kinds := drKindsByTheme[s.Theme]
		if len(kinds) == 0 {
			kinds = []string{"idol", "guard", "light"}
		}
		kind := Pick(r, kinds)
		if drKindDef(kind).Carve && !carveOK {
			kind = Pick(r, []string{"light", "guard", "idol"})
		}
		if used[kind] {
			continue
		}
		used[kind] = true
		id := fmt.Sprintf("%s#%d", zn.Key, i)
		if f := g.drBuild(kind, zn, r, id, lv, s.Theme); f != nil {
			st.F[zn.Key] = append(st.F[zn.Key], f)
		}
	}
	if len(st.F[zn.Key]) > 0 {
		g.computeLight(zn)
	}
}

func (g *Game) drBuild(kind string, zn *Zone, r *Rng, id string, lv int, theme string) *drFeat {
	f := &drFeat{ID: id, Kind: kind, Lv: lv, Tag: theme}
	var ok bool
	switch kind {
	case "plate":
		ok = g.drBuildPlate(zn, r, f)
	case "lever":
		ok = g.drBuildLever(zn, r, f)
	case "rune":
		ok = g.drBuildRune(zn, r, f)
	case "turn":
		ok = g.drBuildTurn(zn, r, f)
	case "flood":
		ok = g.drBuildFlood(zn, r, f)
	case "light":
		ok = g.drBuildLight(zn, r, f)
	case "mirror":
		ok = g.drBuildRoom(zn, r, f, 7, 5, "A silvered mirror-hall. Your own shape waits in every pane.")
	case "guard":
		ok = g.drBuildGuard(zn, r, f)
	case "idol":
		ok = g.drBuildIdol(zn, r, f)
	case "arena":
		ok = g.drBuildRoom(zn, r, f, 9, 5, "A sand-floored arena. The gate is open, for now.")
	case "bridge":
		ok = g.drBuildBridge(zn, r, f)
	}
	if !ok {
		return nil
	}
	return f
}

// drVault carves a sealed 5x3 vault with loot inside, its door tile set to a riveted wall.
func (g *Game) drVault(zn *Zone, r *Rng, f *drFeat, note string) bool {
	rm, ok := drCarve(zn, r, 5, 3, 15)
	if !ok {
		return false
	}
	f.RX, f.RY, f.RW, f.RH = rm.X, rm.Y, rm.W, rm.H
	f.GX, f.GY = rm.DX, rm.DY
	zn.Set(rm.DX, rm.DY, TMetalWall)
	zn.Notes[rm.DY*MW+rm.DX] = note
	tier := clampi(drTier(zn)+1, 1, 5)
	for i := 0; i < 3; i++ {
		g.drAddItem(zn, g.Loot(r, tier), rm.X+r.Int(0, rm.W-1), rm.Y+r.Int(0, rm.H-1))
	}
	g.drAddItem(zn, &Item{ID: g.newID(), Kind: "gold", Name: g.Sk.Currency, Glyph: '$', F: CYellow, Count: 25 * zn.Level}, rm.X+2, rm.Y+1)
	return true
}

func (g *Game) drBuildPlate(zn *Zone, r *Rng, f *drFeat) bool {
	f.V = r.Int(0, 2)
	if !g.drVault(zn, r, f, "A riveted slab seals the vault. It has no handle.") {
		return false
	}
	x, y, ok := drSpot(zn, r, f.V == 1, [][2]int{{f.GX, f.GY}}, 8)
	if !ok {
		return false
	}
	f.X, f.Y = x, y
	zn.Set(x, y, TCarpet)
	switch f.V {
	case 0:
		f.N = clampi(5+f.Lv, 6, 12)
		zn.Notes[drPack(x, y)] = fmt.Sprintf("A worn square plate set flush in the floor, rimmed with wear. It would take about %d stone of weight to sink it.", f.N)
	case 1:
		bx, by, ok := drSpot(zn, r, true, [][2]int{{x, y}}, 3)
		if !ok {
			zn.Set(x, y, g.floorFor(zn))
			return false
		}
		zn.Set(bx, by, TCrate)
		zn.Notes[drPack(bx, by)] = "A block of dressed stone. There are drag marks on the floor leading toward a worn plate."
		f.Pts = []int{drPack(bx, by)}
		zn.Notes[drPack(x, y)] = "A worn square plate set flush in the floor. Something heavy has been dragged here before."
	default:
		d := drDist(zn, x, y)
		dd := 20
		for _, dn := range dirs8 {
			if v := d[(f.GY+dn[1])*MW+f.GX+dn[0]]; v >= 0 && v < dd {
				dd = v
			}
		}
		f.Dur = dd + dd/3 + 4
		zn.Notes[drPack(x, y)] = "A worn square plate set flush in the floor. It clicks when stepped on, and something far off begins to grind."
	}
	return true
}

func (g *Game) drBuildLever(zn *Zone, r *Rng, f *drFeat) bool {
	if !g.drVault(zn, r, f, "A riveted slab seals the vault. Rods run into the wall beside it.") {
		return false
	}
	f.N = r.Int(0, 2)
	var away [][2]int
	for i := 0; i < 3; i++ {
		x, y, ok := drSpot(zn, r, true, away, 6)
		if !ok {
			return false
		}
		zn.Set(x, y, TMachine)
		look := drLeverLooks[1+r.Int(0, 1)]
		if i == f.N {
			look = drLeverLooks[0]
		}
		zn.Notes[drPack(x, y)] = "An iron lever set in a stone socket; " + look + "."
		f.Pts = append(f.Pts, drPack(x, y))
		away = append(away, [2]int{x, y})
	}
	return true
}

func (g *Game) drBuildRune(zn *Zone, r *Rng, f *drFeat) bool {
	if !g.drVault(zn, r, f, "Four sigils are cut into the lintel. The slab has no handle.") {
		return false
	}
	perm := []int{0, 1, 2, 3}
	for i := 3; i > 0; i-- {
		j := r.Int(0, i)
		perm[i], perm[j] = perm[j], perm[i]
	}
	syms := make([]int, 0, 4)
	have := map[int]bool{}
	for len(syms) < 4 {
		k := r.Int(0, len(drSyms)-1)
		if !have[k] {
			have[k] = true
			syms = append(syms, k)
		}
	}
	f.Aux = syms
	f.Seq = perm[:3]
	s, _ := evSiteLevel(g, zn)
	who := ""
	if s != nil {
		c := evCtxAt(g, s, zn.WX, zn.WY, zn.ZX, zn.ZY)
		if c.ruler != nil {
			who = " Set down for " + c.ruler.Name + "."
		}
	}
	cx, cy, ok := drSpot(zn, r, true, [][2]int{{f.GX, f.GY}}, 10)
	if !ok {
		return false
	}
	zn.Set(cx, cy, TInscription)
	zn.Notes[drPack(cx, cy)] = fmt.Sprintf(Pick(r, drRuneClue), drSyms[syms[f.Seq[0]]], drSyms[syms[f.Seq[1]]], drSyms[syms[f.Seq[2]]]) + who
	var names []string
	for _, k := range syms {
		names = append(names, drSyms[k])
	}
	zn.Notes[f.GY*MW+f.GX] = "Four sigils are cut into the lintel: " + strings.Join(names, ", ") + ". The slab has no handle."
	return true
}

func (g *Game) drBuildTurn(zn *Zone, r *Rng, f *drFeat) bool {
	rm, ok := drCarve(zn, r, 7, 5, 10)
	if !ok {
		return false
	}
	f.RX, f.RY, f.RW, f.RH = rm.X, rm.Y, rm.W, rm.H
	zn.Set(rm.DX, rm.DY, rm.Floor)
	for xx := rm.X; xx <= rm.X+6; xx++ {
		zn.Set(xx, rm.Y+1, TMetalWall)
		zn.Set(xx, rm.Y+3, TMetalWall)
	}
	f.GX, f.GY = rm.X+3, rm.Y+1 // north door
	f.X, f.Y = rm.X+3, rm.Y+3   // south door
	zn.Set(f.GX, f.GY, TDoorOpen)
	lx := rm.X + 5
	if rm.Side == 1 {
		lx = rm.X + 1
	}
	zn.Set(lx, rm.Y+2, TMachine)
	zn.Notes[drPack(lx, rm.Y+2)] = "A great lever. The hall's inner walls are set on a pivot below the floor."
	f.Pts = []int{drPack(lx, rm.Y+2)}
	tier := drTier(zn)
	g.drAddItem(zn, g.Loot(r, tier), rm.X+2, rm.Y)
	if a := g.drFoe(zn, r, f.Tag, zn.Level); a != nil {
		a.X, a.Y = rm.X+4, rm.Y
		a.Set("drturn")
		zn.Actors = append(zn.Actors, a)
	}
	g.drAddItem(zn, g.Loot(r, tier+1), rm.X+2, rm.Y+4)
	g.drAddItem(zn, g.Loot(r, tier+1), rm.X+4, rm.Y+4)
	g.drAddItem(zn, &Item{ID: g.newID(), Kind: "gold", Name: g.Sk.Currency, Glyph: '$', F: CYellow, Count: 30 * zn.Level}, rm.X+3, rm.Y+4)
	return true
}

func (g *Game) drBuildFlood(zn *Zone, r *Rng, f *drFeat) bool {
	rm, ok := drCarve(zn, r, 9, 5, 15)
	if !ok {
		return false
	}
	f.RX, f.RY, f.RW, f.RH = rm.X, rm.Y, rm.W, rm.H
	zn.Set(rm.DX, rm.DY, rm.Floor)
	f.GX, f.GY = rm.DX, rm.DY
	f.V = r.Int(0, 1)
	if f.Tag == "sunken" {
		f.V = 0
	}
	f.Dur = 14 + r.Int(0, 6)
	it := g.Loot(r, clampi(drTier(zn)+1, 1, 5))
	if it.Extra == nil {
		it.Extra = map[string]int{}
	}
	it.Extra["drf"] = 1
	g.drAddItem(zn, it, rm.X+rm.W/2, rm.Y+rm.H/2)
	f.N = it.ID
	g.drAddItem(zn, &Item{ID: g.newID(), Kind: "gold", Name: g.Sk.Currency, Glyph: '$', F: CYellow, Count: 40 * zn.Level}, rm.X+rm.W/2+1, rm.Y+rm.H/2)
	zn.Notes[drPack(rm.X+rm.W/2, rm.Y+rm.H/2)] = "A plinth worn smooth. Whatever lay here was meant to be lifted."
	return true
}

func (g *Game) drBuildLight(zn *Zone, r *Rng, f *drFeat) bool {
	perm := []int{0, 1, 2, 3}
	for i := 3; i > 0; i-- {
		j := r.Int(0, i)
		perm[i], perm[j] = perm[j], perm[i]
	}
	f.Seq = perm
	var away [][2]int
	var names []string
	start := r.Int(0, len(drBrazierNames)-4)
	for i := 0; i < 4; i++ {
		x, y, ok := drSpot(zn, r, true, away, 3)
		if !ok {
			return false
		}
		if i == 0 {
			// keep the four close together: later ones within 12 of the first
			away = nil
		}
		zn.Set(x, y, TPillar)
		nm := drBrazierNames[start+i]
		names = append(names, nm)
		zn.Notes[drPack(x, y)] = "A cold brazier, its bowl scorched with the sign of " + nm + "."
		f.Pts = append(f.Pts, drPack(x, y))
		away = append(away, [2]int{x, y})
	}
	cx, cy, ok := drSpot(zn, r, true, nil, 0)
	if !ok {
		return false
	}
	zn.Set(cx, cy, TInscription)
	var order []string
	for _, k := range f.Seq {
		order = append(order, names[k])
	}
	zn.Notes[drPack(cx, cy)] = "The flame must be carried from one bowl to the next in this order: " + strings.Join(order, ", then ") + "."
	f.Tag = f.Tag + "|" + strings.Join(names, ",")
	return true
}

// drBuildRoom: a plain chamber for the mirror and the arena.
func (g *Game) drBuildRoom(zn *Zone, r *Rng, f *drFeat, w, h int, note string) bool {
	rm, ok := drCarve(zn, r, w, h, 15)
	if !ok {
		return false
	}
	f.RX, f.RY, f.RW, f.RH = rm.X, rm.Y, rm.W, rm.H
	f.GX, f.GY = rm.DX, rm.DY
	zn.Set(rm.DX, rm.DY, rm.Floor)
	zn.Notes[drPack(rm.X+rm.W/2, rm.Y+rm.H/2)] = note
	return true
}

func (g *Game) drBuildGuard(zn *Zone, r *Rng, f *drFeat) bool {
	var away [][2]int
	n := 3 + r.Int(0, 1)
	for i := 0; i < n; i++ {
		x, y, ok := drSpot(zn, r, true, away, 4)
		if !ok {
			break
		}
		f.Aux = append(f.Aux, int(zn.At(x, y)))
		zn.Set(x, y, TStatue)
		zn.Notes[drPack(x, y)] = "A statue of a robed figure, one hand held out. The face is worn blank. Its feet are not quite where you remember them."
		f.Pts = append(f.Pts, drPack(x, y))
		away = append(away, [2]int{x, y})
	}
	return len(f.Pts) >= 2
}

func (g *Game) drBuildIdol(zn *Zone, r *Rng, f *drFeat) bool {
	x, y, ok := drSpot(zn, r, true, nil, 0)
	if !ok {
		return false
	}
	f.V = r.Int(0, 1)
	f.X, f.Y = x, y
	if f.V == 0 {
		zn.Set(x, y, TStatue)
		zn.Notes[drPack(x, y)] = "You see " + Pick(r, drIdolNames) + ". Its lap is stained dark with old offerings."
	} else {
		zn.Set(x, y, TAltar)
		zn.Notes[drPack(x, y)] = "A black altar with a channel cut for blood. Offerings have been left here for a very long time."
	}
	f.M = 3
	return true
}

func (g *Game) drBuildBridge(zn *Zone, r *Rng, f *drFeat) bool {
	rm, ok := drCarve(zn, r, 13, 5, 10)
	if !ok {
		return false
	}
	f.RX, f.RY, f.RW, f.RH = rm.X, rm.Y, rm.W, rm.H
	f.GX, f.GY = rm.DX, rm.DY
	zn.Set(rm.DX, rm.DY, rm.Floor)
	dir := 1
	if rm.Side == 1 {
		dir = -1
	}
	f.V = dir
	row := rm.Y + 2
	at := func(i int) int { // i = 0 near bank .. 12 far bank
		if dir == 1 {
			return rm.X + i
		}
		return rm.X + rm.W - 1 - i
	}
	for i := 3; i <= 9; i++ {
		for yy := rm.Y; yy < rm.Y+rm.H; yy++ {
			zn.Set(at(i), yy, TChasm)
		}
		zn.Set(at(i), row, TBridge)
		f.Pts = append(f.Pts, drPack(at(i), row))
	}
	rot := 1 + r.Int(0, 1)
	for k := 0; k < rot; k++ {
		f.Aux = append(f.Aux, f.Pts[r.Int(1, len(f.Pts)-2)])
	}
	f.GX, f.GY = rm.DX, rm.DY
	g.drAddItem(zn, g.Loot(r, clampi(drTier(zn)+1, 1, 5)), at(11), row)
	g.drAddItem(zn, g.Loot(r, drTier(zn)), at(12), row-1)
	g.drAddItem(zn, &Item{ID: g.newID(), Kind: "gold", Name: g.Sk.Currency, Glyph: '$', F: CYellow, Count: 35 * zn.Level}, at(12), row+1)
	zn.Notes[drPack(at(3), row)] = "Planks laid over a long drop. Some of them are older than the rest."
	f.M = drPack(at(0), row)
	return true
}

// ---------------------------------------------------------------- arrival hint

func drEnter(g *Game, zn *Zone) {
	st := g.dr()
	seen := map[string]bool{}
	for _, f := range st.F[zn.Key] {
		if f.St >= 2 || seen[f.Kind] {
			continue
		}
		seen[f.Kind] = true
		if h := drKindDef(f.Kind).Hint; h != "" {
			g.Msg("{{c|" + h + "}}")
		}
	}
	g.drBossEnter(zn)
}

func drKill(g *Game, d, src *Actor) {
	g.drBossKill(d)
}

// ---------------------------------------------------------------- props, proximity, bump

type drProp struct{ X, Y, Idx int }

func drProps(f *drFeat) []drProp {
	var out []drProp
	pts := func() {
		for i, p := range f.Pts {
			out = append(out, drProp{p % MW, p / MW, i})
		}
	}
	switch f.Kind {
	case "plate":
		if f.V == 1 && f.St < 2 && len(f.Pts) > 0 {
			out = append(out, drProp{f.Pts[0] % MW, f.Pts[0] / MW, 0})
		}
	case "lever", "light", "guard", "bridge":
		pts()
	case "turn":
		pts()
	case "rune":
		if f.St < 2 {
			out = append(out, drProp{f.GX, f.GY, 0})
		}
	case "idol":
		if f.St < 3 {
			out = append(out, drProp{f.X, f.Y, 0})
		}
	}
	return out
}

func (g *Game) drNearProp() (*drFeat, int, bool) {
	if g.Cur == nil || g.P == nil {
		return nil, 0, false
	}
	for _, f := range g.dr().F[g.Cur.Key] {
		for _, p := range drProps(f) {
			if maxi(abs(p.X-g.P.X), abs(p.Y-g.P.Y)) <= 1 {
				return f, p.Idx, true
			}
		}
	}
	if b := g.dr().B[g.Cur.Key]; b != nil && !b.Done {
		if f, i, ok := g.drBossProp(b); ok {
			return f, i, true
		}
	}
	return nil, 0, false
}

// drBump opens the screen for the prop at (x,y). Wire from MoveP/interact (see the report).
func drBump(g *Game, x, y int) bool {
	if g.Cur == nil {
		return false
	}
	for _, f := range g.dr().F[g.Cur.Key] {
		for _, p := range drProps(f) {
			if p.X == x && p.Y == y {
				g.drOpen(f, p.Idx)
				return true
			}
		}
	}
	if b := g.dr().B[g.Cur.Key]; b != nil && !b.Done {
		for i, p := range b.Pts {
			if p%MW == x && p/MW == y && b.Gim != "lava" && b.Gim != "ice" && b.Gim != "safezone" {
				g.drOpen(g.drBossFeat(b), i)
				return true
			}
		}
	}
	return false
}

func (g *Game) drOpen(f *drFeat, idx int) {
	st := g.dr()
	st.ui = drUI{f: f, idx: idx}
	g.Mode = "dr"
}

func (g *Game) drProximity() {
	if g.Mode != "main" || g.Cur == nil {
		return
	}
	st := g.dr()
	f, i, ok := g.drNearProp()
	if !ok {
		if len(st.near) > 0 {
			st.near = map[string]bool{}
		}
		return
	}
	if st.near[f.ID] {
		return
	}
	st.near[f.ID] = true
	if f.Kind != "bossmech" {
		for _, a := range g.Cur.Actors {
			if a.HP > 0 && a != g.P && g.Hostile(a) && dist(a, g.P) <= 4 && g.visible(a.X, a.Y) {
				return
			}
		}
	}
	g.drOpen(f, i)
}

// ---------------------------------------------------------------- turn

func drTurn(g *Game) {
	if g.P == nil || g.Cur == nil || g.W == nil || g.Dead {
		return
	}
	st := g.dr()
	for _, f := range st.F[g.Cur.Key] {
		g.drTick(f)
	}
	g.drBossTick()
	g.drProximity()
}

func (g *Game) drTick(f *drFeat) {
	zn := g.Cur
	p := g.P
	switch f.Kind {
	case "plate":
		g.drTickPlate(f)
	case "flood":
		if f.St == 0 && !drItemInZone(zn, f.N) {
			f.St, f.T0 = 1, g.Turn
			g.Msg("{{R|" + Pick(NewRng(f.ID), drFloodTell) + " The way out is behind you. Run!}}")
			if f.V == 1 {
				g.Msg("{{R|" + Pick(NewRng(f.ID+"c"), drCollapseTell) + "}}")
			}
		}
		if f.St == 1 {
			g.drTickFlood(f)
		}
	case "mirror":
		if f.St == 0 && drInRect(f, p.X, p.Y) {
			f.St = 1
			g.drMirrorShades(f)
		} else if f.St == 1 && g.drFlagged("drm:"+f.ID) == 0 {
			g.drSolved(f, "The last shade falls and the mirrors go dark. A silvered coffer is left behind.")
			g.drReward(zn, f, f.RX+f.RW/2, f.RY+f.RH/2, 2)
		}
	case "arena":
		g.drTickArena(f)
	case "guard":
		g.drTickGuard(f)
	case "bridge":
		g.drTickBridge(f)
	}
}

func (g *Game) drFlagged(flag string) int {
	n := 0
	for _, a := range g.Cur.Actors {
		if a.HP > 0 && a.Has(flag) {
			n++
		}
	}
	return n
}

func (g *Game) drTickPlate(f *drFeat) {
	zn, p := g.Cur, g.P
	switch f.V {
	case 0:
		w := 0
		for _, it := range zn.ItemsAt(f.X, f.Y) {
			w += it.Weight * maxi(1, it.Count)
		}
		if p.X == f.X && p.Y == f.Y {
			w += 4
		}
		if w >= f.N {
			if f.St < 2 {
				g.drSolved(f, "The plate sinks with a deep note. Somewhere a slab grinds aside.")
				g.drSetDoor(f, true)
			} else if zn.At(f.GX, f.GY) != TDoorOpen {
				g.drSetDoor(f, true)
			}
		} else if f.St == 2 && zn.At(f.GX, f.GY) == TDoorOpen && !drInRect(f, p.X, p.Y) {
			g.drSetDoor(f, false)
		}
	case 2:
		if f.St == 0 && p.X == f.X && p.Y == f.Y {
			f.St, f.T0 = 1, g.Turn
			g.drSetDoor(f, true)
			g.Msg(fmt.Sprintf("{{Y|Click. A slab grinds open far off, and a clock of stone begins to run (about %d minutes).}}", f.Dur))
		}
		if f.St == 1 {
			left := f.Dur - (g.Turn - f.T0)
			if left == 3 {
				g.Msg("{{O|The grinding changes pitch: the slab is about to close.}}")
			}
			if left <= 0 {
				if drInRect(f, p.X, p.Y) || zn.ActorAt(f.GX, f.GY) != nil {
					f.St = 2
					g.Msg("{{G|The slab jams open with you inside.}}")
				} else {
					f.St = 0
					g.drSetDoor(f, false)
					g.Msg("The slab grinds shut. The plate rises again.")
				}
			} else if f.St == 1 && drInRect(f, p.X, p.Y) && f.N == 0 {
				f.N = 1
				g.drSolved(f, "You are in the vault before the slab closes.")
				f.St = 1
			}
		}
	}
}

func (g *Game) drTickFlood(f *drFeat) {
	zn := g.Cur
	el := g.Turn - f.T0
	var tiles [][2]int
	far := f.RX
	if f.GX < f.RX+f.RW/2 {
		far = f.RX + f.RW - 1
	}
	for x := f.RX; x < f.RX+f.RW; x++ {
		for y := f.RY; y < f.RY+f.RH; y++ {
			tiles = append(tiles, [2]int{x, y})
		}
	}
	// order by distance from the far wall
	for i := 1; i < len(tiles); i++ {
		for j := i; j > 0 && abs(tiles[j][0]-far) < abs(tiles[j-1][0]-far); j-- {
			tiles[j], tiles[j-1] = tiles[j-1], tiles[j]
		}
	}
	per := maxi(1, len(tiles)/f.Dur+1)
	for i, t := range tiles {
		stage := el*per - i
		if f.V == 0 {
			switch {
			case stage >= per:
				zn.Set(t[0], t[1], TDeep)
			case stage > 0:
				if zn.At(t[0], t[1]) != TDeep {
					zn.Set(t[0], t[1], TShallow)
				}
			}
		} else if stage > 0 && zn.At(t[0], t[1]) != TRubble {
			zn.Set(t[0], t[1], TRubble)
			if g.P.X == t[0] && g.P.Y == t[1] {
				g.Msg("{{R|A slab of ceiling comes down on you!}}")
				g.drHurt(3+zn.Level/2, "falling stone")
			}
		}
	}
	if el >= f.Dur+4 {
		f.St = 3
		if f.V == 0 {
			g.Msg("The water settles.")
		} else {
			g.Msg("The dust settles.")
		}
	}
}

func (g *Game) drMirrorShades(f *drFeat) {
	zn := g.Cur
	r := NewRng(g.W.Seed + ":dr:shade:" + f.ID)
	mk := func(name string, src *Actor, glyph rune) {
		a := g.Spawn("skeleton", maxi(src.Level, 2), r)
		a.Name, a.Proper, a.Glyph, a.F = "shade of "+name, true, glyph, CDarkGray
		a.Attr = src.Attr
		a.MaxHP = maxi(10, src.MaxHP*55/100)
		a.HP = a.MaxHP
		a.XP = 20 + src.Level*8
		a.Desc = "A reflection that stepped out of the glass."
		a.Set("drm:" + f.ID)
		a.AI = "hostile"
		a.Faction = "undead"
		for i := 0; i < 20; i++ {
			a.X, a.Y = f.RX+r.Int(0, f.RW-1), f.RY+r.Int(0, f.RH-1)
			if zn.Walkable(a.X, a.Y) && zn.ActorAt(a.X, a.Y) == nil && !(a.X == g.P.X && a.Y == g.P.Y) {
				break
			}
		}
		g.nudge(zn, a)
		zn.Actors = append(zn.Actors, a)
	}
	mk(g.P.Name, g.P, '@')
	n := 0
	for _, o := range append([]*Actor{}, zn.Actors...) {
		if o.AI == "follower" && o.HP > 0 && n < 3 {
			mk(o.Name, o, o.Glyph)
			n++
		}
	}
	g.Msg("{{M|The panes ripple. Something that looks like you steps out" + map[bool]string{true: ", and others behind it", false: ""}[n > 0] + ".}}")
}

func (g *Game) drTickArena(f *drFeat) {
	zn, p := g.Cur, g.P
	switch f.St {
	case 0:
		if drInRect(f, p.X, p.Y) {
			f.St, f.N, f.T0 = 1, 0, g.Turn
			g.drSetDoor(f, false)
			zn.Set(f.GX, f.GY, TMetalWall)
			g.Msg("{{R|" + Pick(NewRng(f.ID), drArenaTaunts) + "}}")
			g.drWave(f)
		}
	case 1:
		if g.drFlagged("dra:"+f.ID) == 0 {
			if f.N >= 3 {
				zn.Set(f.GX, f.GY, TDoorOpen)
				g.drSolved(f, "The last of them falls. The gate lifts, and a prize is set on the sand.")
				g.drReward(zn, f, f.RX+f.RW/2, f.RY+f.RH/2, 3)
				f.St = 2
			} else {
				g.drWave(f)
			}
		} else if p.HP*4 < p.MaxHP || g.Turn-f.T0 > 150 {
			zn.Set(f.GX, f.GY, TDoorOpen)
			f.St = 3
			st := g.dr()
			st.Failed++
			g.Msg("{{O|The gate lifts. The crowd that is not there groans. No prize for the one who yields.}}")
		}
	}
}

func (g *Game) drWave(f *drFeat) {
	zn := g.Cur
	f.N++
	r := NewRng(fmt.Sprintf("%s:dr:wave:%s:%d", g.W.Seed, f.ID, f.N))
	cnt := 2 + f.N/2 + zn.Level/8
	for i := 0; i < cnt; i++ {
		a := g.drFoe(zn, r, f.Tag, zn.Level+f.N/2)
		a.Set("dra:" + f.ID)
		for k := 0; k < 20; k++ {
			a.X, a.Y = f.RX+r.Int(0, f.RW-1), f.RY+r.Int(0, f.RH-1)
			if zn.Walkable(a.X, a.Y) && zn.ActorAt(a.X, a.Y) == nil && !(a.X == g.P.X && a.Y == g.P.Y) {
				break
			}
		}
		g.nudge(zn, a)
		zn.Actors = append(zn.Actors, a)
	}
	g.Msg(fmt.Sprintf("{{O|Wave %d of 3.}}", f.N))
}

func (g *Game) drTickGuard(f *drFeat) {
	zn, p := g.Cur, g.P
	if g.Turn%2 != 0 {
		return
	}
	for i, pk := range f.Pts {
		x, y := pk%MW, pk/MW
		if g.visible(x, y) {
			continue
		}
		d := maxi(abs(x-p.X), abs(y-p.Y))
		if d > 11 {
			continue
		}
		if d <= 1 {
			if g.Turn%6 == 0 {
				g.Msg("{{R|Stone fingers close on your arm. You never saw it move.}}")
				g.drHurt(2+zn.Level/3, "a stone hand")
				p.AddSt("dazed", 2)
			}
			continue
		}
		bx, by, bd := x, y, d
		for _, dd := range dirs8 {
			nx, ny := x+dd[0], y+dd[1]
			if !evFloors[zn.At(nx, ny)] || zn.At(nx, ny) == TRubble || zn.ActorAt(nx, ny) != nil || len(zn.ItemsAt(nx, ny)) > 0 || (nx == p.X && ny == p.Y) || evNearStair(zn, nx, ny, 0) {
				continue
			}
			if nd := maxi(abs(nx-p.X), abs(ny-p.Y)); nd < bd {
				bx, by, bd = nx, ny, nd
			}
		}
		if bx != x || by != y {
			zn.Set(x, y, Terrain(f.Aux[i]))
			f.Aux[i] = int(zn.At(bx, by))
			zn.Set(bx, by, TStatue)
			zn.Notes[drPack(bx, by)] = zn.Notes[pk]
			delete(zn.Notes, pk)
			f.Pts[i] = drPack(bx, by)
		}
	}
}

func (g *Game) drTickBridge(f *drFeat) {
	zn, p := g.Cur, g.P
	here := drPack(p.X, p.Y)
	for k, rp := range f.Aux {
		if rp == here && zn.At(p.X, p.Y) == TBridge {
			zn.Set(p.X, p.Y, TChasm)
			f.Aux = append(f.Aux[:k], f.Aux[k+1:]...)
			bx, by := f.M%MW, f.M/MW
			p.X, p.Y = bx, by
			g.Msg("{{R|The plank snaps under you! You throw yourself back onto the bank.}}")
			g.drHurt(2+zn.Level/3, "a broken plank")
			g.Msg("{{c|(Use the mechanism at the gap to leap it or lay a plank.)}}")
			return
		}
	}
	for _, pk := range f.Pts {
		if pk == here && zn.At(p.X, p.Y) == TBridge {
			f.M = here
		}
	}
	if f.M == 0 {
		f.M = drPack(f.RX, f.RY)
	}
}

// ---------------------------------------------------------------- the lettered screen

type drOpt struct {
	T  string
	Do func() bool
}

func drKeys(g *Game, k string) {
	st := g.dr()
	f := st.ui.f
	if k == "Escape" || k == "q" || f == nil || g.Dead {
		g.Mode = "main"
		return
	}
	i := letterIndex(k)
	if i < 0 {
		return
	}
	_, _, opts := g.drMenu(f, st.ui.idx)
	if i >= len(opts) {
		return
	}
	st.ui.res = nil
	if !opts[i].Do() || g.Dead {
		g.Mode = "main"
	}
}

func (g *Game) drDraw() {
	st := g.dr()
	sc := g.sc
	if st.ui.f == nil {
		g.Mode = "main"
		return
	}
	title, lines, opts := g.drMenu(st.ui.f, st.ui.idx)
	x, y, w, h := g.panel(title)
	bottom := y + h - 1
	put := func(s string) {
		for _, ln := range Wrap(s, w) {
			if y < bottom {
				sc.TextB(x, y, ln, CGray, CBgDark, x+w)
				y++
			}
		}
	}
	for _, l := range lines {
		put(l)
	}
	y++
	for i, o := range opts {
		put(fmt.Sprintf("{{W|%c}}) %s", 'a'+i, o.T))
	}
	if len(st.ui.res) > 0 {
		y++
		for _, s := range st.ui.res {
			put(s)
		}
	}
	sc.TextB(x, bottom, "{{K|Press a letter to choose   Esc step away}}", CGray, CBgDark, x+w)
}

func (g *Game) drMenu(f *drFeat, idx int) (string, []string, []drOpt) {
	zn := g.Cur
	name := drKindDef(f.Kind).Name
	if f.Kind == "bossmech" {
		return g.drBossMenu(f, idx)
	}
	var lines []string
	var opts []drOpt
	leave := drOpt{"Step away.", func() bool { return false }}
	switch f.Kind {
	case "plate":
		lines = []string{"A block of dressed stone sits in the room, with drag marks leading to a worn plate."}
		if f.St >= 2 {
			lines = append(lines, "The plate is sunk. The way is open.")
			break
		}
		dc := drDC(zn, 9)
		opts = append(opts, drOpt{fmt.Sprintf("Heave the block onto the plate. %s", g.checkLabel("endurance", dc)), func() bool {
			if g.roll2d6Check(g.checkMod("endurance"), dc) {
				bx, by := f.Pts[0]%MW, f.Pts[0]/MW
				zn.Set(bx, by, g.floorFor(zn))
				delete(zn.Notes, f.Pts[0])
				zn.Set(f.X, f.Y, TCrate)
				g.drSolved(f, "The block grinds onto the plate and settles. A slab grinds aside far off.")
				g.drSetDoor(f, true)
				g.spend(3000)
				return false
			}
			g.drSay("It will not budge. You strain a muscle.")
			g.drHurt(2, "a strained back")
			g.spend(2000)
			return true
		}})
	case "lever":
		which := idx
		lines = []string{"An iron lever in a stone socket. " + strings.TrimPrefix(zn.Notes[f.Pts[idx]], "An iron lever set in a stone socket; ")}
		if f.St >= 2 {
			lines = append(lines, "It is spent.")
			break
		}
		dc := drDC(zn, 9)
		opts = append(opts, drOpt{fmt.Sprintf("Look it over for signs of use. %s", g.checkLabel("perception", dc)), func() bool {
			if g.roll2d6Check(g.checkMod("perception"), dc) {
				if which == f.N {
					g.drSay("The grip is polished by long use. This is the one that was meant to be pulled.")
				} else {
					g.drSay("Rust in the socket, a seized pin. Nobody pulled this one on purpose.")
				}
			} else {
				g.drSay("You cannot tell.")
			}
			g.spend(1000)
			return true
		}})
		opts = append(opts, drOpt{"Pull it.", func() bool {
			g.spend(1000)
			if which == f.N {
				g.drSolved(f, "The lever drops with a heavy clack. A slab grinds aside.")
				g.drSetDoor(f, true)
				return false
			}
			g.drSay(Pick(g.R, drTrapLines))
			switch g.R.Int(0, 2) {
			case 0:
				g.drHurt(3+zn.Level/2, "a dart trap")
				g.P.AddSt("poisoned", 8)
			case 1:
				g.P.AddSt("poisoned", 12)
				g.P.AddSt("dazed", 3)
			default:
				for i := 0; i < 2; i++ {
					a := g.drFoe(zn, g.R, f.Tag, zn.Level)
					a.X, a.Y = g.P.X+g.R.Int(-4, 4), g.P.Y+g.R.Int(-3, 3)
					g.nudge(zn, a)
					zn.Actors = append(zn.Actors, a)
				}
			}
			return true
		}})
	case "rune":
		lines = []string{zn.Notes[drPack(f.GX, f.GY)], ""}
		if len(f.In) > 0 {
			var got []string
			for _, k := range f.In {
				got = append(got, drSyms[f.Aux[k]])
			}
			lines = append(lines, "Pressed so far: "+strings.Join(got, ", ")+".")
		}
		for i := range f.Aux {
			i := i
			opts = append(opts, drOpt{"Press the " + drSyms[f.Aux[i]] + ".", func() bool { return g.drRunePress(f, i) }})
		}
		if f.N >= 3 {
			dc := drDC(zn, 12)
			opts = append(opts, drOpt{fmt.Sprintf("Pry at the seal. %s", g.checkLabel("tinkering", dc)), func() bool {
				g.spend(3000)
				if g.roll2d6Check(g.checkMod("tinkering"), dc) {
					g.drSolved(f, "The seal gives with a crack. The slab swings.")
					g.drSetDoor(f, true)
					return false
				}
				g.drSay("The seal bites back.")
				g.drHurt(3+zn.Level/3, "a rune shock")
				return true
			}})
		}
	case "turn":
		lines = []string{"A great lever in a hall whose inner walls turn with it. Two shut chambers lie north and south."}
		opts = append(opts, drOpt{"Throw the lever.", func() bool {
			north := zn.At(f.GX, f.GY) == TDoorOpen
			g.spend(2000)
			if north {
				zn.Set(f.GX, f.GY, TMetalWall)
				zn.Set(f.X, f.Y, TDoorOpen)
			} else {
				zn.Set(f.GX, f.GY, TDoorOpen)
				zn.Set(f.X, f.Y, TMetalWall)
			}
			if f.St == 0 {
				f.St = 2
				g.dr().Solved++
			}
			g.drSay("Stone grinds under the floor and the inner walls swing round.")
			return true
		}})
	case "light":
		lines = []string{zn.Notes[f.Pts[idx]]}
		if f.St >= 2 {
			lines = append(lines, "All four burn.")
			break
		}
		lines = append(lines, fmt.Sprintf("Lit in the right order so far: %d of 4.", f.N))
		names := strings.Split(strings.SplitN(f.Tag, "|", 2)[1], ",")
		i := idx
		light := false
		for _, it := range g.P.Inv {
			if it.Kind == "light" || it.Elem == "fire" {
				light = true
			}
		}
		dc := drDC(zn, 9)
		opts = append(opts, drOpt{fmt.Sprintf("Light the %s brazier%s.", names[i], map[bool]string{true: "", false: " by striking sparks " + g.checkLabel("tinkering", dc)}[light]), func() bool {
			g.spend(1000)
			if !light && !g.roll2d6Check(g.checkMod("tinkering"), dc) {
				g.drSay("The sparks die.")
				return true
			}
			return g.drLightBrazier(f, i)
		}})
	case "mirror":
		lines = []string{"Silvered panes. Nothing here to use."}
	case "guard":
		lines = []string{"A stone figure, hand held out. It is cold, and it is closer to you than it was."}
		dc := drDC(zn, 10)
		i := idx
		opts = append(opts, drOpt{fmt.Sprintf("Hammer it to pieces with all you have. %s", g.checkLabel("endurance", dc)), func() bool {
			g.spend(2000)
			if g.roll2d6Check(g.checkMod("endurance"), dc) {
				pk := f.Pts[i]
				x, y := pk%MW, pk/MW
				zn.Set(x, y, Terrain(f.Aux[i]))
				delete(zn.Notes, pk)
				f.Pts = append(f.Pts[:i], f.Pts[i+1:]...)
				f.Aux = append(f.Aux[:i], f.Aux[i+1:]...)
				g.drSay("The figure bursts. A heartstone rolls out of the dust.")
				g.drAddItem(zn, &Item{ID: g.newID(), Kind: "trophy", Name: "sculptor's heartstone", Glyph: '*', F: CMagenta, Value: 40 + 12*zn.Level, Lore: "A gem packed in the chest of a walking statue."}, x, y)
				g.gainXP(20 + zn.Level*5)
				if len(f.Pts) == 0 {
					f.St = 2
				}
				return false
			}
			g.drSay("Your blow rings off it. Stone fingers close on your wrist.")
			g.drHurt(2+zn.Level/3, "a stone hand")
			g.P.AddSt("dazed", 2)
			return true
		}})
	case "idol":
		return g.drIdolMenu(f)
	case "bridge":
		lines = []string{"A plank bridge over a long, dark drop. Some planks are older than the rest."}
		p := g.P
		row := f.Pts[0] / MW
		ahead := func(n int) Terrain { return zn.At(p.X+f.V*n, row) }
		if p.Y == row && ahead(1) == TChasm {
			dc := drDC(zn, 9)
			opts = append(opts, drOpt{fmt.Sprintf("Leap the gap. %s", g.checkLabel("acrobatics", dc)), func() bool {
				g.spend(1000)
				if g.roll2d6Check(g.checkMod("acrobatics"), dc) && zn.Walkable(p.X+2*f.V, row) && ahead(2) != TChasm {
					p.X += 2 * f.V
					g.drSay("You clear the gap and land on the far plank.")
					return false
				}
				g.drSay("You fall short and catch the edge, and haul yourself back.")
				g.drHurt(2+zn.Level/3, "a hard landing")
				return true
			}})
			opts = append(opts, drOpt{"Lay a plank across the gap (uses one " + g.Sk.Scrap + ").", func() bool {
				for _, it := range g.P.Inv {
					if it.Kind == "scrap" {
						if it.Count > 1 {
							it.Count--
						} else {
							g.removeInv(it)
						}
						zn.Set(p.X+f.V, row, TBridge)
						g.drSay("You lay a plank and test it. It holds.")
						g.spend(1000)
						return false
					}
				}
				g.drSay("You have nothing to lay.")
				return true
			}})
		} else {
			dc := drDC(zn, 9)
			opts = append(opts, drOpt{fmt.Sprintf("Test the planks with your weight and your ears. %s", g.checkLabel("perception", dc)), func() bool {
				g.spend(2000)
				if g.roll2d6Check(g.checkMod("perception"), dc) {
					for _, rp := range f.Aux {
						zn.Set(rp%MW, rp/MW, TChasm)
					}
					g.drSay(fmt.Sprintf("You find %d rotten plank(s) and pull them out. Leap the gaps or lay a plank.", len(f.Aux)))
					f.Aux = nil
				} else {
					g.drSay("They all look about as bad as each other.")
				}
				return true
			}})
		}
	}
	opts = append(opts, leave)
	return name, lines, opts
}

func (g *Game) drRunePress(f *drFeat, i int) bool {
	zn := g.Cur
	f.In = append(f.In, i)
	g.spend(500)
	if len(f.In) < 3 {
		return true
	}
	ok := true
	for k := range f.Seq {
		if f.In[k] != f.Seq[k] {
			ok = false
		}
	}
	f.In = nil
	if ok {
		g.drSolved(f, "The sigils flare in turn. The slab swings open.")
		g.drSetDoor(f, true)
		return false
	}
	f.N++
	g.drSay("The sigils flash white. Raw power jolts through your hands.")
	g.drHurt(3+zn.Level/3, "a rune shock")
	if f.N == 3 {
		g.drSay("The sigils go dull and cold. The seal will only yield to force now.")
	}
	return true
}

func (g *Game) drLightBrazier(f *drFeat, i int) bool {
	zn := g.Cur
	if i == f.Seq[f.N] {
		zn.Set(f.Pts[i]%MW, f.Pts[i]/MW, TBrazier)
		f.N++
		g.computeLight(zn)
		if f.N >= 4 {
			g.drSolved(f, "The fourth bowl catches. The flames lean together and a hidden panel opens.")
			g.drReward(zn, f, f.Pts[3]%MW, f.Pts[3]/MW, 2)
			return false
		}
		g.drSay("It catches and burns steadily.")
		return true
	}
	for _, pk := range f.Pts {
		zn.Set(pk%MW, pk/MW, TPillar)
	}
	f.N = 0
	g.computeLight(zn)
	g.drSay("The flame gutters, and every bowl goes dark at once. Things move in the dark.")
	r := NewRng(fmt.Sprintf("%s:dr:dark:%d", g.W.Seed, g.Turn))
	for k := 0; k < 2; k++ {
		a := g.Spawn("wisp", zn.Level, r)
		a.X, a.Y = g.P.X+r.Int(-3, 3), g.P.Y+r.Int(-3, 3)
		g.nudge(zn, a)
		zn.Actors = append(zn.Actors, a)
	}
	return false
}

func (g *Game) drIdolMenu(f *drFeat) (string, []string, []drOpt) {
	zn := g.Cur
	name := "Idol"
	if f.V == 1 {
		name = "Altar of offerings"
	}
	lines := []string{zn.Notes[drPack(f.X, f.Y)], fmt.Sprintf("It will answer %d more time(s). What it gives is a boon, or a curse; bigger offerings tip it toward the boon.", f.M)}
	thr := 30 * (zn.Level/3 + 1)
	offer := func(score int, what string, pay func()) bool {
		pay()
		g.spend(1000)
		f.M--
		if f.M <= 0 {
			f.St = 3
		}
		chance := clampi(25+60*score/thr, 25, 90)
		if g.R.Int(1, 100) <= chance && !g.R.Chance(0.12) {
			g.drBoon(f)
		} else {
			g.drCurse(f)
		}
		_ = what
		return f.M > 0
	}
	var opts []drOpt
	gold := maxi(10, thr/2)
	if g.P.Gold >= gold {
		opts = append(opts, drOpt{fmt.Sprintf("Leave %d %s.", gold, g.Sk.Currency), func() bool {
			return offer(gold, "gold", func() { g.P.Gold -= gold })
		}})
	}
	hp := maxi(3, g.P.MaxHP/6)
	if g.P.HP > hp+2 {
		opts = append(opts, drOpt{fmt.Sprintf("Give blood (%d HP).", hp), func() bool {
			return offer(hp*thr/(maxi(4, g.P.MaxHP/5)), "blood", func() { g.drHurt(hp, "a blood offering") })
		}})
	}
	n := 0
	for _, it := range g.P.Inv {
		if n >= 5 {
			break
		}
		if it.Kind == "key" || it.Quest != 0 || it.Value <= 0 {
			continue
		}
		it := it
		n++
		opts = append(opts, drOpt{fmt.Sprintf("Offer %s (worth %d).", it.Display(), it.Value), func() bool {
			return offer(it.Value, "item", func() { g.removeInv(it) })
		}})
	}
	return name, lines, append(opts, drOpt{"Step away.", func() bool { return false }})
}

func (g *Game) drBoon(f *drFeat) {
	zn := g.Cur
	p := g.P
	b := drIdolBoons[g.R.Int(0, len(drIdolBoons)-1)]
	switch b {
	case "heal":
		p.HP = p.MaxHP
		g.drSay("{{G|Warmth floods you. Your wounds close.}}")
	case "might":
		p.AddSt("mighty", 300)
		g.drSay("{{G|Your arms swell with borrowed strength.}}")
	case "swift":
		p.AddSt("hasted", 200)
		g.drSay("{{G|The world slows around you.}}")
	case "ward":
		p.AddSt("shielded", 300)
		g.drSay("{{G|A hush settles over you like a second skin.}}")
	case "gold":
		n := 40 * zn.Level
		p.Gold += n
		g.drSay(fmt.Sprintf("{{G|Coin runs out of the idol's mouth: %d %s.}}", n, g.Sk.Currency))
	case "find":
		r := NewRng(fmt.Sprintf("%s:dr:boon:%s:%d", g.W.Seed, f.ID, f.M))
		g.give(g.Loot(r, clampi(drTier(zn)+1, 1, 5)))
		g.drSay("{{G|Something heavy drops into your hand.}}")
	default:
		g.gainXP(60 + 10*zn.Level)
		g.drSay("{{G|You see, for one moment, how the whole place was made.}}")
	}
}

func (g *Game) drCurse(f *drFeat) {
	zn := g.Cur
	p := g.P
	c := drIdolCurses[g.R.Int(0, len(drIdolCurses)-1)]
	switch c {
	case "weak":
		p.AddSt("dazed", 6)
		p.AddSt("slowed", 120)
		g.drSay("{{R|Your limbs turn to lead.}}")
	case "slow":
		p.AddSt("slowed", 200)
		g.drSay("{{R|Time drags at your heels.}}")
	case "hunt":
		g.drSay("{{R|Cold eyes open in the dark. Something was sent after you.}}")
		r := NewRng(fmt.Sprintf("%s:dr:curse:%s:%d", g.W.Seed, f.ID, f.M))
		for k := 0; k < 2; k++ {
			a := g.drFoe(zn, r, f.Tag, zn.Level)
			a.X, a.Y = p.X+r.Int(-4, 4), p.Y+r.Int(-3, 3)
			g.nudge(zn, a)
			zn.Actors = append(zn.Actors, a)
		}
	default:
		lose := mini(p.Gold, maxi(10, p.Gold/8))
		p.Gold -= lose
		p.AddSt("poisoned", 10)
		g.drSay(fmt.Sprintf("{{R|The idol takes %d more than you gave, and your blood turns bitter.}}", lose))
	}
}
