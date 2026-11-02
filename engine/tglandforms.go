package engine

import (
	"fmt"
	"math"
	"strings"
)

// Landforms in zone generation (prefix tg): escarpments at steep height steps between parasangs,
// waterfalls where rivers drop over them (sometimes with a grotto behind), canyons in dry
// uplands, glaciers with crevasses (which drop into ice caves below), oases, hot springs and
// geysers, tar pits, quicksand, salt flats, coral reefs, sinkholes and volcanoes. Everything is
// painted by one zone hook from the side rng `seed:tg:zoneKey`, only over natural tiles and away
// from stairs, actors and items, so zones without a landform come out exactly as before.

const tgCliffDrop = 900.0 // metres between neighbouring parasangs that make an escarpment

func init() {
	zoneHooks = append(zoneHooks, tgZoneHook)
}

// tgNatural: wild ground a landform may paint over.
func tgNatural(t Terrain) bool {
	switch t {
	case TGrass, TTallGrass, TSand, TDirt, TSnow, TIce, TAsh, TMoss, TFungus, TCrystalFloor, TTree,
		TDeadTree, TMushroom, TBush, TReeds, TRock, TCrystal, TRubble, TBones, TFloor:
		return true
	}
	return false
}

func tgWater(t Terrain) bool { return t == TShallow || t == TDeep }

func tgNeighbours(w *WorldData, p int, card bool) []int {
	x, y := p%w.W, p/w.W
	var out []int
	for i, d := range dirs8 {
		if card && i%2 == 1 {
			continue
		}
		if w.In(x+d[0], y+d[1]) {
			out = append(out, w.Idx(x+d[0], y+d[1]))
		}
	}
	return out
}

// tgVolcanoAt reports whether parasang p holds a volcano and whether it is active. When the world
// file carries Realmforge's volcanoes (hazards.go) they decide; older worlds derive them from
// height, biome and planet.
func tgVolcanoAt(w *WorldData, p int) (bool, bool) {
	if w == nil || p < 0 || p >= len(w.Biome) || w.IsWater(p) {
		return false, false
	}
	if w.Hazards != nil && len(w.Hazards.Volcanoes) > 0 {
		v := w.hzVolcanoAt(p)
		return v != nil, v != nil && v.Active
	}
	b, h := w.Biome[p], w.Height[p]
	nmax, sum, n := 0, 0, 0
	for _, q := range tgNeighbours(w, p, false) {
		nh := w.Height[q]
		nmax = maxi(nmax, nh)
		sum += nh
		n++
	}
	if n == 0 {
		return false, false
	}
	mean := float64(sum) / float64(n)
	localMax := h >= nmax
	hs := HashStr(fmt.Sprintf("%s:tg:volc:%d", w.Seed, p))
	switch {
	case b == 16 && localMax:
		return true, true
	case b == 13 && h >= 50 && localMax:
		return true, hs%3 == 0
	case w.PlanetKind == "volcanic" && h >= 62 && localMax && float64(h)-mean >= 6:
		return true, hs%2 == 0
	case (b == 16 || b == 13) && nmax >= 65 && h >= 40 && hs%5 == 0:
		return true, b == 16
	case h >= 70 && localMax && float64(h)-mean >= 8 && hs%3 == 0:
		return true, hs%9 == 0 // a lone stratovolcano on any world
	}
	return false, false
}

// tgVolcanoZone is the sub-zone of parasang p that holds the cone.
func tgVolcanoZone(w *WorldData, p int) (int, int) {
	if w.Burg[p] == 0 && len(w.wondersAt[p]) == 0 && len(w.monstersAt[p]) == 0 {
		return 1, 1
	}
	return 1, 0
}

// tgGlacial: ice sheets and glacier tongues.
func tgGlacial(w *WorldData, p int) bool {
	if w.IsWater(p) {
		return false
	}
	b, h := w.Biome[p], w.Height[p]
	cold := p < len(w.Temp) && w.Temp[p] <= -2
	return b == 11 || (b == 10 || b == 9) && h >= 55 || h >= 72 && cold
}

// tgCanyonAt: gorges cut into dry uplands.
func tgCanyonAt(w *WorldData, p int) bool {
	if w.IsWater(p) {
		return false
	}
	b, h := w.Biome[p], w.Height[p]
	return b == 1 && h >= 45 || b == 2 && h >= 55 && h < 85 || b == 3 && h >= 52
}

// tgCliffsAround: some cardinal neighbour lies an escarpment below p.
func tgCliffsAround(w *WorldData, p int) bool {
	for _, q := range tgNeighbours(w, p, true) {
		if !w.IsWater(q) && w.tgParasangM(p)-w.tgParasangM(q) >= tgCliffDrop {
			return true
		}
	}
	return false
}

// tgPainter paints over natural tiles outside a protected mask (stairs, actors, items).
type tgPainter struct {
	zn      *Zone
	prot    []bool
	changed bool
}

func tgNewPainter(zn *Zone) *tgPainter {
	pt := &tgPainter{zn: zn, prot: make([]bool, MW*MH)}
	mark := func(x, y, rad int) {
		for dy := -rad; dy <= rad; dy++ {
			for dx := -rad; dx <= rad; dx++ {
				if zn.In(x+dx, y+dy) {
					pt.prot[(y+dy)*MW+x+dx] = true
				}
			}
		}
	}
	for i, t := range zn.T {
		switch t {
		case TDown, TUp, TUpdraft, TPortal, TAltar, TInscription, TStatue, TSarcophagus, TBrazier, TBed:
			mark(i%MW, i/MW, 1)
		}
	}
	for _, a := range zn.Actors {
		mark(a.X, a.Y, 0)
	}
	for _, it := range zn.Items {
		mark(it.X, it.Y, 0)
	}
	return pt
}

func (pt *tgPainter) ok(x, y int) bool {
	return pt.zn.In(x, y) && !pt.prot[y*MW+x]
}

// set paints t at (x, y) over natural ground; water too when wet is true.
func (pt *tgPainter) set(x, y int, t Terrain, wet bool) bool {
	if !pt.ok(x, y) {
		return false
	}
	cur := pt.zn.At(x, y)
	if !tgNatural(cur) && !(wet && tgWater(cur)) {
		return false
	}
	if cur != t {
		pt.zn.Set(x, y, t)
		pt.changed = true
	}
	return true
}

func tgZoneHook(g *Game, zn *Zone, _ *Rng) {
	w := g.W
	if w == nil || !w.In(zn.WX, zn.WY) {
		return
	}
	p := w.Idx(zn.WX, zn.WY)
	r := NewRng(w.Seed + ":tg:" + zn.Key)
	pt := tgNewPainter(zn)
	switch {
	case zn.Z == 0 && zn.Kind == "surface":
		g.tgSurface(zn, r, pt, p)
	case zn.Z == 0 && zn.Kind == "sea":
		g.tgReef(zn, r, pt, p)
	case zn.Z == 1 && zn.Kind == "strata" && tgGlacial(w, p):
		g.tgIceCave(zn, pt)
	}
	if pt.changed {
		g.connect(zn)
		g.computeLight(zn)
	}
}

func (g *Game) tgSurface(zn *Zone, r *Rng, pt *tgPainter, p int) {
	w := g.W
	b := w.Biome[p]
	h := w.Height[p]
	if v, act := tgVolcanoAt(w, p); v {
		if zx, zy := tgVolcanoZone(w, p); zn.ZX == zx && zn.ZY == zy {
			g.tgVolcano(zn, r, pt, act)
			return
		}
		g.tgVolcanoFlank(zn, r, pt, act)
	}
	g.tgCliffs(zn, r, pt, p)
	switch {
	case tgGlacial(w, p):
		g.tgGlacier(zn, pt, p)
	case tgCanyonAt(w, p) && r.Chance(0.4):
		g.tgCanyon(zn, r, pt, p)
	}
	temp := 10
	if p < len(w.Temp) {
		temp = w.Temp[p]
	}
	// hot springs and geysers: volcanic ground, cold uplands, rarely anywhere
	spring := 0.01
	switch {
	case b == 13 || b == 16:
		spring = 0.15
	case tgGlacial(w, p) || b == 10 || h >= 55 && temp < 5:
		spring = 0.06
	}
	if v, _ := tgVolcanoAt(w, p); v {
		spring = 0.3
	}
	if w.PlanetKind == "volcanic" {
		spring += 0.1
	}
	if r.Chance(spring) {
		g.tgSprings(zn, r, pt)
	}
	switch b {
	case 1, 2:
		if r.Chance(0.12) {
			g.tgOasis(zn, r, pt)
		}
		if h <= 30 && r.Chance(0.3) {
			g.tgSaltFlat(zn, pt, p)
		}
		if r.Chance(0.1) {
			g.tgBlobs(zn, r, pt, TQuicksand, r.Int(2, 4), 1.2, 2.4)
		}
		if b == 1 && r.Chance(0.05) {
			g.tgBlobs(zn, r, pt, TTar, r.Int(1, 3), 1.5, 3)
		}
	case 12:
		if r.Chance(0.1) {
			g.tgBlobs(zn, r, pt, TQuicksand, r.Int(1, 3), 1, 2)
		}
		if r.Chance(0.03) {
			g.tgBlobs(zn, r, pt, TTar, r.Int(1, 2), 1.5, 2.5)
		}
	case 3, 13:
		if r.Chance(0.05) {
			g.tgBlobs(zn, r, pt, TTar, r.Int(1, 3), 1.5, 3)
		}
	}
	if (b >= 3 && b <= 6) && h >= 22 && h <= 55 && r.Chance(0.04) {
		g.tgSinkholes(zn, r, pt)
	}
}

// ---- escarpments and waterfalls ----

func (g *Game) tgCliffs(zn *Zone, r *Rng, pt *tgPainter, p int) {
	w := g.W
	here := w.tgParasangM(p)
	var falls [][4]int // x, y, along dx, along dy
	for di := 0; di < 8; di += 2 {
		d := dirs8[di]
		nx, ny := zn.WX+d[0], zn.WY+d[1]
		if !w.In(nx, ny) {
			continue
		}
		q := w.Idx(nx, ny)
		drop := here - w.tgParasangM(q)
		if w.IsWater(q) || drop < tgCliffDrop {
			continue
		}
		// only the sub-zones on that edge
		if d[0] > 0 && zn.ZX != 2 || d[0] < 0 && zn.ZX != 0 || d[1] > 0 && zn.ZY != 2 || d[1] < 0 && zn.ZY != 0 {
			continue
		}
		thick := 1
		if drop >= 1600 {
			thick = 2
		}
		seed := HashStr(fmt.Sprintf("%s:tg:cliff:%d:%d", w.Seed, p, di))
		vertical := d[0] != 0 // the cliff runs north-south along an east/west edge
		n, span := MH, 6.0
		if !vertical {
			n, span = MW, 3.0
		}
		prev := -1
		for s := 0; s < n; s++ {
			var gs float64
			if vertical {
				gs = float64(zn.ZY*MH + s)
			} else {
				gs = float64(zn.ZX*MW + s)
			}
			off := 2 + int(fbm(seed, gs/9, 0.5)*span)
			pos := off // distance from the zone edge facing the neighbour
			lo, hi := pos, pos
			if prev >= 0 {
				lo, hi = mini(pos, prev), maxi(pos, prev)
			}
			prev = pos
			for o := lo; o <= hi+thick-1; o++ {
				var x, y int
				switch {
				case d[0] > 0:
					x, y = MW-1-o, s
				case d[0] < 0:
					x, y = o, s
				case d[1] > 0:
					x, y = s, MH-1-o
				default:
					x, y = s, o
				}
				if !pt.ok(x, y) {
					continue
				}
				cur := zn.At(x, y)
				if tgWater(cur) {
					zn.Set(x, y, TFalls)
					pt.changed = true
					ax, ay := 0, 1
					if !vertical {
						ax, ay = 1, 0
					}
					falls = append(falls, [4]int{x, y, ax, ay})
					// spray below the falls, toward the low side
					for k := 1; k <= 2; k++ {
						sx, sy := x+d[0]*(o-pos+k), y+d[1]*(o-pos+k)
						if zn.In(sx, sy) && zn.At(sx, sy) != TFalls && zn.At(sx, sy) != TDeep {
							pt.set(sx, sy, TSpray, true)
						}
					}
				} else {
					pt.set(x, y, TCliff, false)
				}
			}
		}
	}
	if len(falls) > 0 {
		tgName(zn, "by the falls")
		if r.Chance(0.4) {
			g.tgGrotto(zn, r, pt, falls[r.Int(0, len(falls)-1)])
		}
	}
}

// tgGrotto hollows a little cave into the cliff right beside a waterfall, reached through the curtain.
func (g *Game) tgGrotto(zn *Zone, r *Rng, pt *tgPainter, f [4]int) {
	x, y, ax, ay := f[0], f[1], f[2], f[3]
	cx, cy := ay, ax // across the cliff line
	dir := 1
	if r.Chance(0.5) {
		dir = -1
	}
	var cells [][2]int
	for k := 1; k <= 2; k++ {
		gx, gy := x+ax*k*dir, y+ay*k*dir
		if !pt.ok(gx, gy) || zn.At(gx, gy) == TFalls {
			return
		}
		cells = append(cells, [2]int{gx, gy})
	}
	for _, c := range cells {
		zn.Set(c[0], c[1], TFloor)
		for _, s := range []int{-1, 1} {
			if px, py := c[0]+cx*s, c[1]+cy*s; pt.ok(px, py) && zn.At(px, py) != TFalls {
				zn.Set(px, py, TCliff)
			}
		}
	}
	if ex, ey := x+ax*3*dir, y+ay*3*dir; pt.ok(ex, ey) && zn.At(ex, ey) != TFalls {
		zn.Set(ex, ey, TCliff)
	}
	end := cells[len(cells)-1]
	it := g.Loot(r, zn.Level/3+1)
	it.X, it.Y = end[0], end[1]
	zn.Items = append(zn.Items, it)
	zn.Notes[end[1]*MW+end[0]] = "A dry hollow behind the falls."
	pt.changed = true
}

// ---- canyon, glacier, ice caves ----

func (g *Game) tgCanyon(zn *Zone, r *Rng, pt *tgPainter, p int) {
	seed := HashStr(fmt.Sprintf("%s:tg:canyon:%d", g.W.Seed, p))
	floorT := TSand
	if g.W.Biome[p] != 1 {
		floorT = TDirt
	}
	any := false
	for y := 0; y < MH; y++ {
		for x := 0; x < MW; x++ {
			gx, gy := gxy(zn, x, y)
			d := math.Abs(fbm(seed, gx/40, gy/14) - 0.5)
			switch {
			case d < 0.03:
				t := floorT
				if noise2(seed+3, gx/2, gy) > 0.9 {
					t = TRubble
				}
				if pt.set(x, y, t, false) {
					any = true
				}
			case d < 0.05:
				if noise2(seed+5, gx/6, gy/3) > 0.85 {
					pt.set(x, y, TRubble, false) // a scree path down the wall
				} else if pt.set(x, y, TCliff, false) {
					any = true
				}
			}
		}
	}
	if any {
		tgName(zn, "a canyon")
	}
}

func (g *Game) tgGlacier(zn *Zone, pt *tgPainter, p int) {
	seed := HashStr(fmt.Sprintf("%s:tg:glacier:%d", g.W.Seed, p))
	for y := 0; y < MH; y++ {
		for x := 0; x < MW; x++ {
			gx, gy := gxy(zn, x, y)
			if fbm(seed, gx/22, gy/8) <= 0.42 {
				continue
			}
			t := zn.At(x, y)
			if t == TRock || !t.D().Walk {
				continue
			}
			if math.Abs(noise2(seed+5, gx/6, gy/1.6)-0.5) < 0.022 && noise2(seed+6, gx/10, gy/4) > 0.45 {
				pt.set(x, y, TChasm, false) // crevasse
			} else {
				pt.set(x, y, TGlacier, false)
			}
		}
	}
}

func (g *Game) tgIceCave(zn *Zone, pt *tgPainter) {
	seed := HashStr(g.W.Seed + ":tg:icecave:" + zn.Key)
	for i, t := range zn.T {
		x, y := i%MW, i/MW
		switch t {
		case TRock:
			zn.T[i] = TIceWall
		case TFloor:
			if noise2(seed, float64(x)/4, float64(y)/2) > 0.45 {
				zn.T[i] = TIce
			}
		case TShallow:
			zn.T[i] = TIce
		default:
			continue
		}
		pt.changed = true
	}
	zn.Name = "Ice Caves beneath the glacier"
}

// ---- small features ----

func (g *Game) tgSpot(r *Rng, mx, my int) (int, int) {
	return r.Int(mx, MW-1-mx), r.Int(my, MH-1-my)
}

// tgBlobs paints n noisy ellipses of t with radii between lo and hi.
func (g *Game) tgBlobs(zn *Zone, r *Rng, pt *tgPainter, t Terrain, n int, lo, hi float64) {
	seed := uint64(r.Int(1, 1<<30))
	for k := 0; k < n; k++ {
		cx, cy := g.tgSpot(r, 6, 3)
		rad := lo + r.Float()*(hi-lo)
		for dy := -4; dy <= 4; dy++ {
			for dx := -9; dx <= 9; dx++ {
				d := math.Hypot(float64(dx)/2, float64(dy))
				if d < rad+noise2(seed, float64(cx+dx)/2, float64(cy+dy))*0.8-0.4 {
					if tt := zn.At(cx+dx, cy+dy); tt.D().Walk && tt != TRock {
						pt.set(cx+dx, cy+dy, t, false)
					}
				}
			}
		}
	}
	if t == TTar {
		tgName(zn, "tar pits")
	}
}

func (g *Game) tgOasis(zn *Zone, r *Rng, pt *tgPainter) {
	cx, cy := g.tgSpot(r, 14, 6)
	for dy := -6; dy <= 6; dy++ {
		for dx := -13; dx <= 13; dx++ {
			x, y := cx+dx, cy+dy
			d := math.Hypot(float64(dx)/2.2, float64(dy))
			switch {
			case d < 1.2:
				pt.set(x, y, TDeep, true)
			case d < 2.6:
				pt.set(x, y, TShallow, true)
			case d < 3.4:
				pt.set(x, y, TTallGrass, false)
			case d < 5.2:
				if r.Chance(0.4) {
					pt.set(x, y, TPalm, false)
				} else {
					pt.set(x, y, TGrass, false)
				}
			case d < 6 && r.Chance(0.3):
				pt.set(x, y, TBush, false)
			}
		}
	}
	tgName(zn, "an oasis")
}

func (g *Game) tgSaltFlat(zn *Zone, pt *tgPainter, p int) {
	seed := HashStr(fmt.Sprintf("%s:tg:salt:%d", g.W.Seed, p))
	for y := 0; y < MH; y++ {
		for x := 0; x < MW; x++ {
			gx, gy := gxy(zn, x, y)
			if fbm(seed, gx/18, gy/6) > 0.55 {
				if t := zn.At(x, y); t == TSand || t == TDirt || t == TRubble || t == TDeadTree || t == TBush {
					pt.set(x, y, TSalt, false)
				}
			}
		}
	}
	if pt.changed {
		tgName(zn, "salt flats")
	}
}

func (g *Game) tgSprings(zn *Zone, r *Rng, pt *tgPainter) {
	cx, cy := g.tgSpot(r, 12, 5)
	pools := r.Int(1, 3)
	for k := 0; k < pools; k++ {
		px, py := cx+r.Int(-8, 8), cy+r.Int(-3, 3)
		rad := 1 + r.Float()*1.2
		for dy := -3; dy <= 3; dy++ {
			for dx := -6; dx <= 6; dx++ {
				d := math.Hypot(float64(dx)/2, float64(dy))
				if d < rad {
					pt.set(px+dx, py+dy, THotSpring, true)
				} else if d < rad+1 {
					if t := zn.At(px+dx, py+dy); t != THotSpring && t != TGeyser {
						pt.set(px+dx, py+dy, TRubble, false)
					}
				}
			}
		}
	}
	for k := r.Int(0, 2); k > 0; k-- {
		gx, gy := clampi(cx+r.Int(-12, 12), 2, MW-3), clampi(cy+r.Int(-5, 5), 2, MH-3)
		if pt.set(gx, gy, TGeyser, false) {
			for _, d := range dirs8 {
				if t := zn.At(gx+d[0], gy+d[1]); t != TGeyser && t != THotSpring {
					pt.set(gx+d[0], gy+d[1], TDirt, false)
				}
			}
		}
	}
	tgName(zn, "hot springs")
}

func (g *Game) tgSinkholes(zn *Zone, r *Rng, pt *tgPainter) {
	for k := r.Int(1, 2); k > 0; k-- {
		cx, cy := g.tgSpot(r, 8, 4)
		for dy := -3; dy <= 3; dy++ {
			for dx := -6; dx <= 6; dx++ {
				d := math.Hypot(float64(dx)/2, float64(dy))
				if d < 1.2 {
					pt.set(cx+dx, cy+dy, TChasm, false)
				} else if d < 2.4 && r.Chance(0.7) {
					pt.set(cx+dx, cy+dy, TRubble, false)
				}
			}
		}
	}
}

func (g *Game) tgReef(zn *Zone, r *Rng, pt *tgPainter, p int) {
	w := g.W
	if p >= len(w.Temp) || w.Temp[p] < 20 {
		return
	}
	seed := HashStr(fmt.Sprintf("%s:tg:reef:%d", w.Seed, p))
	shallow := 0
	for i, t := range zn.T {
		if t == TShallow {
			shallow++
			gx, gy := gxy(zn, i%MW, i/MW)
			if noise2(seed, gx/4, gy/2) > 0.55 {
				pt.set(i%MW, i/MW, TCoral, true)
			}
		}
	}
	if shallow < 40 && r.Chance(0.12) {
		// an atoll: a ring of reef round a lagoon
		cx, cy := g.tgSpot(r, 16, 7)
		for dy := -7; dy <= 7; dy++ {
			for dx := -15; dx <= 15; dx++ {
				d := math.Hypot(float64(dx)/2.2, float64(dy))
				switch {
				case d < 3.8:
					pt.set(cx+dx, cy+dy, TShallow, true)
				case d < 5.6:
					if r.Chance(0.55) {
						pt.set(cx+dx, cy+dy, TCoral, true)
					} else {
						pt.set(cx+dx, cy+dy, TShallow, true)
					}
				}
			}
		}
		zn.Name = "Atoll Reef"
	} else if pt.changed {
		zn.Name = "Coral Shallows"
	}
}

// ---- volcanoes ----

func (g *Game) tgVolcano(zn *Zone, r *Rng, pt *tgPainter, active bool) {
	seed := uint64(r.Int(1, 1<<30))
	breach := r.Float() * 2 * math.Pi
	cx, cy := 40.0, 12.0
	for y := 0; y < MH; y++ {
		for x := 0; x < MW; x++ {
			dx, dy := (float64(x)-cx)/2.2, float64(y)-cy
			d := math.Hypot(dx, dy) + (noise2(seed, float64(x)/3, float64(y)/2)-0.5)*0.8
			ang := math.Atan2(dy, dx)
			da := math.Abs(math.Remainder(ang-breach, 2*math.Pi))
			n := noise2(seed+7, float64(x)/2, float64(y))
			switch {
			case d < 2.8:
				if active {
					pt.set(x, y, TLava, true)
				} else if d < 1.8 {
					pt.set(x, y, TShallow, true) // crater lake
				} else {
					pt.set(x, y, TAsh, false)
				}
			case d < 4:
				if active || n > 0.85 {
					if n > 0.75 {
						pt.set(x, y, TFumarole, false)
					} else {
						pt.set(x, y, TAsh, false)
					}
				} else {
					pt.set(x, y, TAsh, false)
				}
			case d < 5.4:
				if da < 0.32 {
					pt.set(x, y, TRubble, false) // the breach
				} else if n > 0.9 {
					pt.set(x, y, TSulfur, false)
				} else {
					pt.set(x, y, TCliff, false) // crater wall
				}
			case d < 11:
				switch {
				case n > 0.93 && d < 7:
					pt.set(x, y, TObsidian, false)
				case n < 0.04:
					pt.set(x, y, TFumarole, false)
				case n < 0.07:
					pt.set(x, y, TSulfur, false)
				case n > 0.88:
					pt.set(x, y, TRock, false)
				case n > 0.76:
					pt.set(x, y, TRubble, false)
				default:
					pt.set(x, y, TAsh, false)
				}
			}
		}
	}
	// dormant craters keep a glowing vent; active ones pour a lava flow out of the breach
	if !active {
		var ash [][2]int
		for y := 4; y <= 20; y++ {
			for x := 28; x <= 52; x++ {
				if zn.At(x, y) == TAsh && math.Hypot((float64(x)-cx)/2.2, float64(y)-cy) < 4 {
					ash = append(ash, [2]int{x, y})
				}
			}
		}
		for k := 0; k < 3 && len(ash) > 0; k++ {
			c := ash[r.Int(0, len(ash)-1)]
			pt.set(c[0], c[1], TLava, false)
		}
	} else {
		x, y := cx+math.Cos(breach)*4*2.2, cy+math.Sin(breach)*4
		for s := 0; s < 80; s++ {
			ix, iy := int(math.Round(x)), int(math.Round(y))
			if !zn.In(ix, iy) {
				break
			}
			pt.set(ix, iy, TLava, true)
			for _, d := range dirs8 {
				if zn.At(ix+d[0], iy+d[1]) != TLava {
					pt.set(ix+d[0], iy+d[1], TAsh, false)
				}
			}
			wob := (noise2(seed+9, float64(s)/6, 0.3) - 0.5) * 1.4
			x += math.Cos(breach+wob) * 1.4
			y += math.Sin(breach+wob) * 0.6
		}
	}
	if active {
		zn.Name = "Burning Crater"
	} else {
		zn.Name = "Dormant Crater"
	}
}

func (g *Game) tgVolcanoFlank(zn *Zone, r *Rng, pt *tgPainter, active bool) {
	for k := r.Int(2, 5); k > 0; k-- {
		x, y := g.tgSpot(r, 2, 2)
		pt.set(x, y, TFumarole, false)
		if r.Chance(0.4) {
			pt.set(x+1, y, TSulfur, false)
		}
	}
	for k := r.Int(1, 3); k > 0; k-- {
		x, y := g.tgSpot(r, 2, 2)
		pt.set(x, y, TObsidian, false)
	}
	if active && r.Chance(0.6) {
		x, y := float64(r.Int(5, MW-6)), float64(r.Int(3, MH-4))
		ang := r.Float() * 2 * math.Pi
		for s := 0; s < r.Int(8, 22); s++ {
			pt.set(int(x), int(y), TLava, false)
			x += math.Cos(ang) * 1.2
			y += math.Sin(ang) * 0.5
			ang += (r.Float() - 0.5) * 0.6
		}
	}
}

// tgName adds one landform suffix to a zone name (only the first landform names the zone).
func tgName(zn *Zone, s string) {
	if !strings.Contains(zn.Name, ", ") {
		zn.Name += ", " + s
	}
}
