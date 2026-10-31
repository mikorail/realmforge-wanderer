package engine

import (
	"fmt"
	"math"
	"sort"
	"strings"
)

// Zone is one 80×25 screen. Each parasang is 3×3 zones on the surface and stacks down (z>0) and up (z<0).
type Zone struct {
	Key        string         `json:"key"`
	WX         int            `json:"wx"`
	WY         int            `json:"wy"`
	ZX         int            `json:"zx"`
	ZY         int            `json:"zy"`
	Z          int            `json:"z"`
	Kind       string         `json:"kind"`
	Name       string         `json:"name"`
	T          []Terrain      `json:"t"`
	Liq        []uint8        `json:"liq"` // 0 none 1 blood 2 oil 3 acid 4 slime
	Fire       []uint8        `json:"fire"`
	Seen       []uint8        `json:"seen"`
	Lit        []bool         `json:"-"`
	Dry        []bool         `json:"dry,omitempty"`
	Items      []*Item        `json:"items"`
	Actors     []*Actor       `json:"actors"`
	Notes      map[int]string `json:"notes"`
	WallF      byte           `json:"wf"`
	Dark       bool           `json:"dark"`
	Underwater bool           `json:"uw"`
	Sky        bool           `json:"sky"`
	Burg       int            `json:"burg"`
	Level      int            `json:"lvl"`
	Visited    bool           `json:"vis"`
	Faction    string         `json:"fac"`
	Biome      int            `json:"biome"`
	Traps      map[int]uint8  `json:"traps,omitempty"`
}

func ZoneKey(wx, wy, zx, zy, z int) string { return fmt.Sprintf("%d.%d.%d.%d.%d", wx, wy, zx, zy, z) }

func (z *Zone) In(x, y int) bool { return x >= 0 && y >= 0 && x < MW && y < MH }
func (z *Zone) At(x, y int) Terrain {
	if !z.In(x, y) {
		return TRock
	}
	return z.T[y*MW+x]
}
func (z *Zone) Set(x, y int, t Terrain) {
	if z.In(x, y) {
		z.T[y*MW+x] = t
	}
}

func (z *Zone) ActorAt(x, y int) *Actor {
	for _, a := range z.Actors {
		if a.X == x && a.Y == y && a.HP > 0 {
			return a
		}
	}
	return nil
}

func (z *Zone) ItemsAt(x, y int) []*Item {
	var out []*Item
	for _, it := range z.Items {
		if it.X == x && it.Y == y {
			out = append(out, it)
		}
	}
	return out
}

func (z *Zone) Walkable(x, y int) bool {
	if !z.In(x, y) {
		return false
	}
	return z.At(x, y).D().Walk
}

func (z *Zone) Opaque(x, y int) bool {
	if !z.In(x, y) {
		return true
	}
	return z.At(x, y).D().Opaque
}

func (z *Zone) WaterAt(x, y int) int {
	if !z.In(x, y) {
		return 0
	}
	if z.Underwater && (z.Dry == nil || !z.Dry[y*MW+x]) {
		return 2
	}
	return z.At(x, y).D().Water
}

func (z *Zone) FreeSpot(r *Rng, near func(x, y int) bool) (int, int, bool) {
	for i := 0; i < 600; i++ {
		x, y := r.Int(1, MW-2), r.Int(1, MH-2)
		if z.Walkable(x, y) && z.ActorAt(x, y) == nil && z.At(x, y).D().Water < 2 && z.At(x, y) != TLava && z.At(x, y) != TVoid && z.At(x, y) != TChasm && (near == nil || near(x, y)) {
			return x, y, true
		}
	}
	return 0, 0, false
}

func newZone(key string, wx, wy, zx, zy, z int) *Zone {
	n := MW * MH
	return &Zone{Key: key, WX: wx, WY: wy, ZX: zx, ZY: zy, Z: z, T: make([]Terrain, n), Liq: make([]uint8, n), Fire: make([]uint8, n),
		Seen: make([]uint8, n), Lit: make([]bool, n), Notes: map[int]string{}, WallF: CGray, Burg: -1}
}

// stairsDown is where a zone's '>' tiles are; computed from the key alone so the zone below can place '<' to match.
func (g *Game) stairsDown(wx, wy, zx, zy, z int) [][2]int {
	if z >= 14 || !g.W.In(wx, wy) {
		return nil
	}
	p := g.W.Idx(wx, wy)
	if z >= 0 {
		if s := g.siteAt(wx, wy, zx, zy); s != nil {
			if s.Kind == "tower" && z == 0 || s.Kind == "dungeon" && z >= s.Depth {
				return nil
			}
			if s.Kind == "dungeon" {
				x, y := posFor(siteKey(wx, wy, zx, zy) + fmt.Sprintf(":down%d", z))
				return [][2]int{{x, y}}
			}
		}
	}
	r := NewRng(g.W.Seed + ":stairs:" + ZoneKey(wx, wy, zx, zy, z))
	chance := 0.6
	if z == 0 {
		chance = 0.22
		h := g.W.Height[p]
		if h >= 50 {
			chance = 0.55
		}
		if h >= 70 {
			chance = 0.9
		}
		if g.W.IsWater(p) {
			chance = 0
			if g.W.seaOpen[p] && zx == 1 && zy == 1 {
				chance = 1
			}
		}
		if zx == 1 && zy == 1 {
			if b := g.W.burgByID[g.W.Burg[p]]; b != nil && !b.Alive {
				chance = 1 // crypt beneath the ruin
			}
			if _, ok := g.W.deepBurg[1][p]; ok {
				chance = 1
			}
		}
	} else if z < 0 {
		return nil
	}
	var out [][2]int
	if r.Chance(chance) {
		out = append(out, [2]int{r.Int(8, MW-9), r.Int(4, MH-5)})
	}
	if z > 0 && r.Chance(0.2) {
		out = append(out, [2]int{r.Int(8, MW-9), r.Int(4, MH-5)})
	}
	return out
}

func (g *Game) updraftAt(wx, wy, zx, zy int) (int, int, bool) {
	if !g.W.In(wx, wy) || zx != 1 || zy != 1 {
		return 0, 0, false
	}
	p := g.W.Idx(wx, wy)
	if !g.W.skyOpen[p] {
		return 0, 0, false
	}
	r := NewRng(g.W.Seed + ":updraft:" + ZoneKey(wx, wy, zx, zy, 0))
	return r.Int(10, MW-11), r.Int(5, MH-6), true
}

func (g *Game) zoneLevel(wx, wy, z int) int {
	p := g.W.Idx(clampi(wx, 0, g.W.W-1), clampi(wy, 0, g.W.H-1))
	d := math.Hypot(float64(wx-g.StartX), float64(wy-g.StartY))
	lvl := 1 + int(d/5)
	if z > 0 {
		lvl += z * 2
	}
	if z < 0 {
		lvl += 3
	}
	switch g.W.Biome[p] {
	case 11, 13, 14, 15, 16:
		lvl += 3
	case 1, 10, 12:
		lvl++
	}
	if g.W.State[p] == 0 {
		lvl++
	}
	return clampi(lvl, 1, 40)
}

// GetZone returns a visited zone or generates it deterministically.
func (g *Game) GetZone(wx, wy, zx, zy, z int) *Zone {
	key := ZoneKey(wx, wy, zx, zy, z)
	if zn, ok := g.Zones[key]; ok {
		return zn
	}
	zn := g.genZone(wx, wy, zx, zy, z)
	g.Zones[key] = zn
	return zn
}

func (g *Game) genZone(wx, wy, zx, zy, z int) *Zone {
	key := ZoneKey(wx, wy, zx, zy, z)
	zn := newZone(key, wx, wy, zx, zy, z)
	r := NewRng(g.W.Seed + ":zone:" + key)
	p := g.W.Idx(wx, wy)
	zn.Biome = g.W.Biome[p]
	zn.Level = g.zoneLevel(wx, wy, z)
	site := g.siteAt(wx, wy, zx, zy)
	switch {
	case z <= towerBase-1:
		if site == nil || site.Kind != "tower" {
			site = &Site{Kind: "tower", Theme: "magic", Name: "A forgotten tower", Depth: 1}
		}
		g.genTowerFloor(zn, r, site, towerBase-z)
	case z < 0:
		g.genSky(zn, r, p)
	case z == 0:
		g.genSurface(zn, r, p)
	case site != nil && site.Kind == "dungeon" && z <= site.Depth && !(site.Ruin && z == 1):
		g.genDungeon(zn, r, site)
	default:
		g.genUnder(zn, r, p)
	}
	// stairs to the level below, '<' from the level above
	for _, s := range g.stairsDown(wx, wy, zx, zy, z) {
		g.clearAround(zn, s[0], s[1], 1)
		zn.Set(s[0], s[1], TDown)
	}
	if z > 0 {
		for _, s := range g.stairsDown(wx, wy, zx, zy, z-1) {
			g.clearAround(zn, s[0], s[1], 1)
			zn.Set(s[0], s[1], TUp)
		}
	}
	if z == 0 && site != nil {
		if site.Kind == "tower" {
			x, y := g.towerStair(siteKey(wx, wy, zx, zy), 0)
			g.siteEntrance(zn, site, x, y)
		} else if st := g.stairsDown(wx, wy, zx, zy, 0); len(st) > 0 {
			g.siteEntrance(zn, site, st[0][0], st[0][1])
			zn.Set(st[0][0], st[0][1], TDown)
		}
	}
	if z == 0 {
		if x, y, ok := g.updraftAt(wx, wy, zx, zy); ok {
			g.clearAround(zn, x, y, 1)
			zn.Set(x, y, TUpdraft)
			zn.Notes[y*MW+x] = "A " + g.Sk.Updraft + " rises toward the " + g.W.LayerName("sky", "sky") + " here. Press < to ascend."
		}
	}
	if z == -1 {
		if x, y, ok := g.updraftAt(wx, wy, zx, zy); ok {
			g.clearAround(zn, x, y, 2)
			for dy := -1; dy <= 1; dy++ {
				for dx := -1; dx <= 1; dx++ {
					if zn.At(x+dx, y+dy) == TVoid {
						zn.Set(x+dx, y+dy, TCloud)
					}
				}
			}
			zn.Set(x, y, TUp)
			zn.Notes[y*MW+x] = "The " + g.Sk.Updraft + " leads back down. Press > to descend."
		}
	}
	g.connect(zn)
	g.computeLight(zn)
	if zn.Kind != "town" && zn.Kind != "dungeon" && zn.Kind != "tower" {
		g.populate(zn, r)
	}
	return zn
}

func (g *Game) clearAround(zn *Zone, x, y, rad int) {
	for dy := -rad; dy <= rad; dy++ {
		for dx := -rad; dx <= rad; dx++ {
			t := zn.At(x+dx, y+dy)
			if !t.D().Walk || t == TDeep || t == TLava || t == TChasm || (t == TVoid && !zn.Sky) {
				zn.Set(x+dx, y+dy, g.floorFor(zn))
			}
		}
	}
}

func (g *Game) floorFor(zn *Zone) Terrain {
	if zn.Sky {
		return TCloud
	}
	if zn.Z > 0 {
		switch {
		case zn.Underwater:
			return TSand
		case zn.Kind == "underdark" && zn.Z%3 == 0:
			return TCrystalFloor
		case zn.Kind == "underdark":
			return TFungus
		}
		return TFloor
	}
	switch zn.Biome {
	case 1, 2:
		return TSand
	case 10, 11:
		return TSnow
	case 13, 16:
		return TAsh
	case 14:
		return TCrystalFloor
	case 15:
		return TFungus
	case 0:
		return TSand
	}
	return TGrass
}

// global tile coordinates across the whole parasang (240×75), so rivers and coasts line up between zones
func gxy(zn *Zone, x, y int) (float64, float64) {
	return float64(zn.ZX*MW + x), float64(zn.ZY*MH + y)
}

var dirs8 = [8][2]int{{0, -1}, {1, -1}, {1, 0}, {1, 1}, {0, 1}, {-1, 1}, {-1, 0}, {-1, -1}}

func (g *Game) genSurface(zn *Zone, r *Rng, p int) {
	w := g.W
	b := w.Biome[p]
	h := w.Height[p]
	seed := HashStr(w.Seed + fmt.Sprint(p))
	zn.Kind = "surface"
	zn.Name = titleCase(BiomeName(b))
	if w.IsWater(p) {
		zn.Kind = "sea"
		zn.Name = "Open Sea"
	}
	for y := 0; y < MH; y++ {
		for x := 0; x < MW; x++ {
			gx, gy := gxy(zn, x, y)
			n1 := fbm(seed, gx/9, gy/4)
			n2 := noise2(seed+77, gx/3, gy/1.5)
			zn.T[y*MW+x] = surfTile(b, h, w.Temp[p], n1, n2, r)
		}
	}
	// coasts: water bleeds in from ocean neighbours, land from land neighbours
	for i, d := range dirs8 {
		nx, ny := zn.WX+d[0], zn.WY+d[1]
		if !w.In(nx, ny) {
			continue
		}
		np := w.Idx(nx, ny)
		if w.IsWater(np) == w.IsWater(p) {
			continue
		}
		_ = i
		for y := 0; y < MH; y++ {
			for x := 0; x < MW; x++ {
				gx, gy := gxy(zn, x, y)
				// distance (0..1) from the side facing the neighbour
				tx, ty := 1.0, 1.0
				if d[0] > 0 {
					tx = 1 - gx/240
				} else if d[0] < 0 {
					tx = gx / 240
				}
				if d[1] > 0 {
					ty = 1 - gy/75
				} else if d[1] < 0 {
					ty = gy / 75
				}
				t := math.Min(tx, ty)
				if d[0] != 0 && d[1] != 0 {
					t = math.Max(tx, ty)
				}
				edge := 0.18 + fbm(seed+9, gx/14, gy/5)*0.22
				if t < edge {
					idx := y*MW + x
					if w.IsWater(p) {
						if t < edge-0.05 {
							zn.T[idx] = surfTile(w.Biome[np], w.Height[np], w.Temp[np], fbm(seed, gx/9, gy/4), noise2(seed+77, gx/3, gy/1.5), r)
						} else {
							zn.T[idx] = TSand
						}
					} else {
						if t < edge-0.06 {
							zn.T[idx] = TDeep
						} else if t < edge-0.02 {
							zn.T[idx] = TShallow
						} else {
							zn.T[idx] = TSand
						}
					}
				}
			}
		}
	}
	// rivers and roads run from the parasang centre toward neighbours that share them
	if !w.IsWater(p) {
		if w.River[p] > 0 {
			width := 1.2 + math.Min(float64(w.River[p])/3, 2.5)
			for _, d := range dirs8 {
				nx, ny := zn.WX+d[0], zn.WY+d[1]
				if w.In(nx, ny) && (w.River[w.Idx(nx, ny)] > 0 || w.IsWater(w.Idx(nx, ny))) {
					g.stroke(zn, seed+3, d, width, func(x, y int, core bool) {
						if core {
							zn.Set(x, y, TDeep)
						} else if zn.At(x, y) != TDeep {
							zn.Set(x, y, TShallow)
						}
					})
				}
			}
			zn.Name = zn.Name + " by the river"
		}
		if w.Road[p] > 0 || w.Burg[p] > 0 {
			for _, d := range dirs8 {
				nx, ny := zn.WX+d[0], zn.WY+d[1]
				if w.In(nx, ny) && (w.Road[w.Idx(nx, ny)] > 0 || w.Burg[w.Idx(nx, ny)] > 0) && (w.Road[p] > 0 || w.Road[w.Idx(nx, ny)] > 0) {
					g.stroke(zn, seed+5, d, 0.9, func(x, y int, core bool) {
						t := zn.At(x, y)
						if t == TDeep || t == TShallow {
							zn.Set(x, y, TBridge)
						} else if t != TBridge {
							zn.Set(x, y, TRoad)
						}
					})
				}
			}
		}
	}
	zn.Faction = fmt.Sprintf("state:%d", w.State[p])
	if zn.ZX == 1 && zn.ZY == 1 {
		g.genFeature(zn, r, p)
	} else if bid := w.Burg[p]; bid > 0 {
		if bd := w.burgByID[bid]; bd != nil && bd.Alive {
			if kind, _ := g.districtAt(zn.WX, zn.WY, zn.ZX, zn.ZY); kind != "" {
				g.genDistrict(zn, r, bd, g.cityPlanFor(bd), kind)
			} else {
				g.genOutskirts(zn, r, bd)
				if ol := w.outlawByBrg[bd.ID]; ol != nil && g.outlawZone(bd) == [2]int{zn.ZX, zn.ZY} {
					g.genCamp(zn, r)
					for i := 0; i < ol.Strength/6; i++ {
						a := g.Spawn(Pick(r, []string{"bandit", "archer"}), zn.Level, r)
						if x, y, ok := zn.FreeSpot(r, nil); ok {
							a.X, a.Y = x, y
							a.Desc = "One of " + ol.Name + "."
							zn.Actors = append(zn.Actors, a)
						}
					}
					zn.Name = ol.Name
					zn.Kind = "camp"
				}
			}
		}
	}
	if zn.Kind == "surface" && r.Chance(0.06) {
		g.genCamp(zn, r)
	}
}

func surfTile(b, h, temp int, n1, n2 float64, r *Rng) Terrain {
	if b == 0 {
		if n1 > 0.78 {
			return TShallow
		}
		return TDeep
	}
	if h >= 70 && n1 > 0.42 {
		return TRock
	}
	if h >= 52 && n1 > 0.68 {
		return TRock
	}
	cold := temp < -4
	switch b {
	case 1:
		if n2 > 0.93 {
			return TRock
		}
		if n2 > 0.88 {
			return TDeadTree
		}
		return TSand
	case 2:
		if n2 > 0.9 {
			return TRock
		}
		if n2 > 0.84 {
			return TBush
		}
		if n1 > 0.5 {
			return TDirt
		}
		return TSand
	case 3:
		if n2 > 0.9 {
			return TTree
		}
		if n1 > 0.55 {
			return TTallGrass
		}
		return TGrass
	case 4:
		if n2 > 0.94 {
			return TTree
		}
		if n2 > 0.85 {
			return TBush
		}
		if n1 > 0.6 {
			return TTallGrass
		}
		return TGrass
	case 5, 6, 7, 8:
		if n1 > 0.48 && n2 > 0.35 {
			return TTree
		}
		if n2 > 0.82 {
			return TBush
		}
		if b == 8 || b == 7 {
			return TMoss
		}
		if cold {
			return TSnow
		}
		return TGrass
	case 9:
		if n1 > 0.5 && n2 > 0.4 {
			return TTree
		}
		return TSnow
	case 10:
		if n2 > 0.9 {
			return TRock
		}
		if n1 > 0.6 {
			return TDirt
		}
		return TSnow
	case 11:
		if n1 > 0.7 {
			return TRock
		}
		if n2 > 0.93 {
			return TChasm
		}
		if n1 > 0.45 {
			return TIce
		}
		return TSnow
	case 12:
		if n1 > 0.62 {
			return TShallow
		}
		if n2 > 0.75 {
			return TReeds
		}
		if n2 > 0.92 {
			return TTree
		}
		return TMoss
	case 13:
		if n2 > 0.96 {
			return TLava
		}
		if n2 > 0.88 {
			return TDeadTree
		}
		if n1 > 0.7 {
			return TRubble
		}
		return TAsh
	case 14:
		if n2 > 0.82 {
			return TCrystal
		}
		return TCrystalFloor
	case 15:
		if n1 > 0.5 && n2 > 0.45 {
			return TMushroom
		}
		if n2 > 0.85 {
			return TMoss
		}
		return TFungus
	case 16:
		if n1 > 0.6 {
			return TLava
		}
		if n2 > 0.8 {
			return TRock
		}
		return TAsh
	}
	return TGrass
}

// stroke draws a wobbly band from the parasang centre to the edge in direction d, clipped to this zone.
func (g *Game) stroke(zn *Zone, seed uint64, d [2]int, width float64, paint func(x, y int, core bool)) {
	cx, cy := 120.0, 37.0
	ex, ey := cx+float64(d[0])*121, cy+float64(d[1])*38
	steps := 260
	for i := 0; i <= steps; i++ {
		t := float64(i) / float64(steps)
		px := cx + (ex-cx)*t
		py := cy + (ey-cy)*t
		wob := (noise2(seed, t*6, float64(d[0]*3+d[1])) - 0.5) * 14 * math.Sin(t*math.Pi)
		if d[0] == 0 {
			px += wob
		} else {
			py += wob * 0.4
		}
		lx := int(px) - zn.ZX*MW
		ly := int(py) - zn.ZY*MH
		wr := int(math.Ceil(width))
		for dy := -wr; dy <= wr; dy++ {
			for dx := -wr * 2; dx <= wr*2; dx++ {
				dd := math.Hypot(float64(dx)/2, float64(dy))
				if dd <= width && zn.In(lx+dx, ly+dy) {
					paint(lx+dx, ly+dy, dd <= width*0.5)
				}
			}
		}
	}
}

func (g *Game) genCamp(zn *Zone, r *Rng) {
	x, y, ok := zn.FreeSpot(r, nil)
	if !ok {
		return
	}
	for dy := -2; dy <= 2; dy++ {
		for dx := -4; dx <= 4; dx++ {
			if zn.In(x+dx, y+dy) && zn.At(x+dx, y+dy) != TDeep {
				zn.Set(x+dx, y+dy, TDirt)
			}
		}
	}
	zn.Set(x, y, TBrazier)
	zn.Set(x-3, y+1, TBed)
	zn.Set(x+3, y-1, TBed)
	n := r.Int(2, 4)
	for i := 0; i < n; i++ {
		tp := "bandit"
		if i == 0 && r.Chance(0.3) {
			tp = "chief"
		} else if r.Chance(0.35) {
			tp = "archer"
		}
		a := g.Spawn(tp, zn.Level, r)
		a.X, a.Y = clampi(x+r.Int(-3, 3), 1, MW-2), clampi(y+r.Int(-2, 2), 1, MH-2)
		if zn.Walkable(a.X, a.Y) && zn.ActorAt(a.X, a.Y) == nil {
			zn.Actors = append(zn.Actors, a)
		}
	}
	for i := 0; i < 2; i++ {
		it := g.Loot(r, zn.Level/3)
		it.X, it.Y = x+r.Int(-2, 2), y+r.Int(-1, 1)
		zn.Items = append(zn.Items, it)
	}
	zn.Name += ", " + g.Sk.Bandits + " camp"
}

// genFeature fills the centre zone of a parasang with what Realmforge put there: a town, ruin, wonder or lair.
func (g *Game) genFeature(zn *Zone, r *Rng, p int) {
	w := g.W
	for _, wi := range w.wondersAt[p] {
		wd := &w.Wonders[wi]
		if wd.Layer == "surface" || wd.Layer == "" {
			g.genWonder(zn, r, wd)
		}
	}
	if bd := w.burgByID[w.Burg[p]]; bd != nil {
		if bd.Alive {
			if cp := g.cityPlanFor(bd); cp.tier > 0 {
				g.genDistrict(zn, r, bd, cp, cp.districts[[2]int{1, 1}])
			} else {
				g.genTown(zn, r, bd)
			}
		} else {
			g.genRuin(zn, r, bd)
		}
	}
	for _, mi := range w.monstersAt[p] {
		m := &w.Monsters[mi]
		if (m.Layer == "surface" || m.Layer == "sea" && zn.Kind == "sea") && !g.Slain[m.ID] {
			g.placeSimMonster(zn, r, m)
		}
	}
}

func (g *Game) placeSimMonster(zn *Zone, r *Rng, m *MonsterData) {
	x, y, ok := zn.FreeSpot(r, func(x, y int) bool { return abs(x-40) < 20 && abs(y-12) < 8 })
	if !ok {
		return
	}
	lvl := clampi(m.Power/7, 8, 35)
	a := g.Spawn("horror", lvl, r)
	a.Name, a.Proper, a.Boss, a.SimMon, a.Desc = m.Name, true, true, m.ID, m.Lore
	a.Glyph = []rune(strings.ToUpper(m.Species))[0]
	a.F = CRed
	if m.Tier == "cataclysm" || m.Tier == "legendary" {
		a.F = CMagenta
	}
	a.MaxHP = maxi(a.MaxHP, m.HP)
	a.HP = a.MaxHP
	a.AVNat += int(m.Armor * 14)
	a.Faction = "legends"
	a.XP = 400 + m.Power*20
	a.Gold = 100 + m.Power*3
	a.X, a.Y = x, y
	for dy := -3; dy <= 3; dy++ {
		for dx := -6; dx <= 6; dx++ {
			if r.Chance(0.25) && zn.Walkable(x+dx, y+dy) && zn.At(x+dx, y+dy).D().Water == 0 {
				zn.Set(x+dx, y+dy, TBones)
			}
		}
	}
	if m.Trophy != "" {
		a.Inv = append(a.Inv, &Item{ID: g.newID(), Kind: "trophy", Name: m.Trophy, Glyph: '♦', F: CYellow, Value: 200 + m.Power*4,
			Lore: "Proof you slew " + m.Name + "."})
	}
	zn.Actors = append(zn.Actors, a)
	if m.Lair != "" {
		zn.Name = m.Lair
	} else {
		zn.Name = "Lair of " + m.Name
	}
	zn.Kind = "lair"
}

func (g *Game) building(zn *Zone, x0, y0, w, h int, wall, floor Terrain) (int, int) {
	for y := y0; y < y0+h; y++ {
		for x := x0; x < x0+w; x++ {
			if y == y0 || y == y0+h-1 || x == x0 || x == x0+w-1 {
				zn.Set(x, y, wall)
			} else {
				zn.Set(x, y, floor)
			}
		}
	}
	dx, dy := x0+w/2, y0+h-1
	if y0+h-1 >= MH-2 {
		dy = y0
	}
	zn.Set(dx, dy, TDoor)
	return dx, dy
}

func (g *Game) genTown(zn *Zone, r *Rng, bd *BurgData) {
	w := g.W
	zn.Kind, zn.Name, zn.Burg = "town", bd.Name, bd.ID
	zn.Faction = fmt.Sprintf("state:%d", bd.State)
	zn.WallF = g.Sk.WallColor
	size := clampi(int(bd.Pop*2)+6, 6, 14)
	if bd.Capital {
		size += 3
	}
	x0, x1 := 40-size*2, 40+size*2
	y0, y1 := 12-size/2-2, 12+size/2+2
	x0, x1, y0, y1 = clampi(x0, 2, MW-3), clampi(x1, 2, MW-3), clampi(y0, 1, MH-2), clampi(y1, 1, MH-2)
	for y := y0 - 1; y <= y1+1; y++ {
		for x := x0 - 1; x <= x1+1; x++ {
			if zn.Dry != nil && !zn.Dry[y*MW+x] {
				continue
			}
			zn.Set(x, y, TDirt)
		}
	}
	if bd.Port {
		// a quay on the east side, water beyond it
		for y := y0; y <= y1; y++ {
			for x := x1 + 2; x < MW && x <= x1+8; x++ {
				if x == x1+2 {
					zn.Set(x, y, TBridge)
				} else if zn.At(x, y).D().Water == 0 {
					zn.Set(x, y, TShallow)
				}
			}
		}
	}
	walls := bd.Capital || bd.Districts["walls"] > 0
	if walls {
		for x := x0 - 1; x <= x1+1; x++ {
			zn.Set(x, y0-1, TWall)
			zn.Set(x, y1+1, TWall)
		}
		for y := y0 - 1; y <= y1+1; y++ {
			zn.Set(x0-1, y, TWall)
			zn.Set(x1+1, y, TWall)
		}
		zn.Set(40, y0-1, TDoorOpen)
		zn.Set(40, y1+1, TDoorOpen)
		zn.Set(x0-1, 12, TDoorOpen)
		zn.Set(x1+1, 12, TDoorOpen)
	}
	for x := x0; x <= x1; x++ {
		zn.Set(x, 12, TRoad)
	}
	for y := y0; y <= y1; y++ {
		zn.Set(40, y, TRoad)
	}
	zn.Set(42, 13, TWell)
	type room struct {
		x, y, w, h int
		role       string
	}
	roles := []string{"elder", "merchant", "smith", "apothecary", "home", "home", "home", "home"}
	if bd.Districts["temple"] > 0 || r.Chance(0.6) {
		roles = append([]string{"temple"}, roles...)
	}
	if bd.Districts["academy"] > 0 {
		roles = append(roles, "academy")
	}
	if bd.Districts["market"] > 0 || bd.Pop > 8 {
		roles = append(roles, "merchant")
	}
	var rooms []room
	tries := 0
	for len(rooms) < len(roles) && tries < 400 {
		tries++
		rw, rh := r.Int(6, 10), r.Int(4, 6)
		rx, ry := r.Int(x0, x1-rw), r.Int(y0, y1-rh)
		if rx < x0 || ry < y0 {
			continue
		}
		ok := true
		for _, o := range rooms {
			if rx < o.x+o.w+1 && rx+rw+1 > o.x && ry < o.y+o.h+1 && ry+rh+1 > o.y {
				ok = false
				break
			}
		}
		if ry <= 12 && ry+rh >= 12 || rx <= 40 && rx+rw >= 40 {
			ok = false
		}
		for y := ry; y < ry+rh && ok; y++ {
			for x := rx; x < rx+rw; x++ {
				if zn.At(x, y).D().Water > 0 {
					ok = false
					break
				}
			}
		}
		if !ok {
			continue
		}
		rooms = append(rooms, room{rx, ry, rw, rh, roles[len(rooms)]})
	}
	cu := w.cultByID[bd.Culture]
	st := w.stateByID[bd.State]
	used := map[int]bool{}
	notables := w.peopleByBrg[bd.ID]
	for _, rm := range rooms {
		floor := TFloor
		if rm.role == "temple" || rm.role == "elder" {
			floor = TCarpet
		}
		g.building(zn, rm.x, rm.y, rm.w, rm.h, TWall, floor)
		cx, cy := rm.x+rm.w/2, rm.y+rm.h/2
		switch rm.role {
		case "temple":
			zn.Set(cx, rm.y+1, TAltar)
			if rel := w.relByID[w.Religion[g.W.Idx(zn.WX, zn.WY)]]; rel != nil {
				zn.Notes[(rm.y+1)*MW+cx] = fmt.Sprintf("An altar of the %s, sacred to %s.", rel.Name, rel.Deity)
			}
		case "academy":
			for x := rm.x + 1; x < rm.x+rm.w-1; x += 2 {
				zn.Set(x, rm.y+1, TShelf)
			}
		case "merchant":
			zn.Set(cx-1, rm.y+1, TStall)
			zn.Set(cx+1, rm.y+1, TStall)
		case "smith":
			zn.Set(cx, rm.y+1, TBrazier)
		case "home":
			zn.Set(rm.x+1, rm.y+1, TBed)
		}
		if rm.role == "home" && r.Chance(0.5) {
			continue
		}
		role := rm.role
		if role == "home" {
			role = "villager"
		}
		var a *Actor
		if role == "elder" && len(notables) > 0 {
			a = g.npcFromPerson(&w.People[notables[0]], bd, r)
			used[notables[0]] = true
			a.Role = "elder"
		} else {
			a = g.makeNPC(role, bd, cu, r)
		}
		a.X, a.Y = cx, cy
		a.HomeX, a.HomeY = cx, cy
		if !zn.Walkable(cx, cy) {
			a.X, a.Y = cx, rm.y+rm.h-2
		}
		zn.Actors = append(zn.Actors, a)
	}
	for _, ni := range notables {
		if used[ni] {
			continue
		}
		a := g.npcFromPerson(&w.People[ni], bd, r)
		if x, y, ok := zn.FreeSpot(r, func(x, y int) bool { return x >= x0 && x <= x1 && y >= y0 && y <= y1 }); ok {
			a.X, a.Y, a.HomeX, a.HomeY = x, y, x, y
			zn.Actors = append(zn.Actors, a)
		}
		if len(zn.Actors) > 24 {
			break
		}
	}
	guards := 2 + int(bd.Pop/4)
	if bd.Capital {
		guards += 3
	}
	for i := 0; i < clampi(guards, 2, 8); i++ {
		a := g.makeNPC("guard", bd, cu, r)
		if x, y, ok := zn.FreeSpot(r, func(x, y int) bool { return x >= x0-2 && x <= x1+2 && y >= y0-2 && y <= y1+2 }); ok {
			a.X, a.Y, a.HomeX, a.HomeY = x, y, x, y
			zn.Actors = append(zn.Actors, a)
		}
	}
	for i := 0; i < clampi(int(bd.Pop), 2, 6); i++ {
		a := g.makeNPC("villager", bd, cu, r)
		if x, y, ok := zn.FreeSpot(r, func(x, y int) bool { return x >= x0 && x <= x1 && y >= y0 && y <= y1 }); ok {
			a.X, a.Y, a.HomeX, a.HomeY = x, y, x, y
			zn.Actors = append(zn.Actors, a)
		}
	}
	if bd.Port {
		for _, a := range zn.Actors {
			if a.Role == "villager" {
				a.Role = "ferry"
				a.Title = "harbourmaster"
				break
			}
		}
	}
	title := bd.Kind
	if st != nil && st.ID > 0 {
		title += " of " + st.Name
	}
	zn.Name = bd.Name + ", " + title
	sx, sy := 40+3, 12-1
	zn.Set(sx, sy, TInscription)
	zn.Notes[sy*MW+sx] = g.townPlaque(bd)
	g.assignSociety(zn, r, bd)
}

func (g *Game) townPlaque(bd *BurgData) string {
	w := g.W
	s := fmt.Sprintf("{{Y|%s}} — a %s", bd.Name, strings.ToLower(bd.Kind))
	if st := w.stateByID[bd.State]; st != nil && st.ID > 0 {
		s += " of " + st.FullName
		if st.Ruler != "" {
			s += ", under " + st.Ruler
		}
	}
	s += "."
	if bd.Original != "" && bd.Original != bd.Name {
		s += fmt.Sprintf(" In older days it was called %s.", bd.Original)
	}
	if bd.Governor != "" {
		s += " Its governor is " + bd.Governor + "."
	}
	evs := w.eventsAt[bd.P]
	if len(evs) > 0 {
		e := w.Events[evs[len(evs)-1]]
		s += fmt.Sprintf("\n\nRecent news, year %d: %s", e.Y, e.Text)
	}
	return s
}

func (g *Game) genOutskirts(zn *Zone, r *Rng, bd *BurgData) {
	// farm plots and a hamlet or two
	n := r.Int(1, 3)
	for i := 0; i < n; i++ {
		x, y := r.Int(4, MW-16), r.Int(3, MH-9)
		for dy := 0; dy < 5; dy++ {
			for dx := 0; dx < 12; dx++ {
				if zn.At(x+dx, y+dy).D().Water == 0 && zn.At(x+dx, y+dy) != TRock {
					if dy%2 == 0 {
						zn.Set(x+dx, y+dy, TTallGrass)
					} else {
						zn.Set(x+dx, y+dy, TDirt)
					}
				}
			}
		}
		if r.Chance(0.5) {
			g.building(zn, x+2, y+6, 6, 4, TWall, TFloor)
			a := g.makeNPC("villager", bd, g.W.cultByID[bd.Culture], r)
			a.X, a.Y, a.HomeX, a.HomeY = x+5, y+7, x+5, y+7
			a.Title = "farmer"
			zn.Actors = append(zn.Actors, a)
		}
	}
	zn.Name = "Outskirts of " + bd.Name
	zn.Kind = "outskirts"
}

func (g *Game) genRuin(zn *Zone, r *Rng, bd *BurgData) {
	w := g.W
	zn.Kind = "ruin"
	zn.Name = "Ruins of " + bd.Name
	for i := 0; i < 9; i++ {
		rw, rh := r.Int(5, 11), r.Int(4, 7)
		rx, ry := r.Int(10, MW-rw-10), r.Int(2, MH-rh-2)
		for y := ry; y < ry+rh; y++ {
			for x := rx; x < rx+rw; x++ {
				edge := y == ry || y == ry+rh-1 || x == rx || x == rx+rw-1
				if edge && r.Chance(0.7) {
					zn.Set(x, y, TWall)
				} else if edge {
					zn.Set(x, y, TRubble)
				} else {
					zn.Set(x, y, Pick(r, []Terrain{TFloor, TFloor, TRubble, TDirt}))
				}
			}
		}
	}
	// inscriptions tell what Realmforge's chronicle remembers of this place
	evs := w.eventsAt[bd.P]
	k := 0
	for i := len(evs) - 1; i >= 0 && k < 4; i-- {
		e := w.Events[evs[i]]
		x, y, ok := zn.FreeSpot(r, nil)
		if !ok {
			break
		}
		zn.Set(x, y, TInscription)
		zn.Notes[y*MW+x] = fmt.Sprintf("Weathered letters, year %d: \"%s\"", e.Y, e.Text)
		k++
	}
	if k == 0 {
		x, y, ok := zn.FreeSpot(r, nil)
		if ok {
			zn.Set(x, y, TInscription)
			zn.Notes[y*MW+x] = fmt.Sprintf("Here stood %s. No one remembers who broke it.", bd.Name)
		}
	}
	// relics held by the dead town lie in the rubble
	for i := range w.Artifacts {
		a := &w.Artifacts[i]
		if a.HolderType == "burg" && a.HolderID == bd.ID && !g.Taken[a.ID] {
			if x, y, ok := zn.FreeSpot(r, nil); ok {
				it := g.RelicItem(a)
				it.X, it.Y = x, y
				zn.Items = append(zn.Items, it)
			}
		}
	}
	for i := 0; i < r.Int(2, 4); i++ {
		if x, y, ok := zn.FreeSpot(r, nil); ok {
			it := g.Loot(r, zn.Level/3+1)
			it.X, it.Y = x, y
			zn.Items = append(zn.Items, it)
		}
	}
	n := r.Int(3, 6)
	for i := 0; i < n; i++ {
		tp := pickCreature(r, zn.Biome, 'r', zn.Level)
		a := g.Spawn(tp, zn.Level, r)
		if x, y, ok := zn.FreeSpot(r, nil); ok {
			a.X, a.Y = x, y
			zn.Actors = append(zn.Actors, a)
		}
	}
	zn.Level++
}

func (g *Game) genWonder(zn *Zone, r *Rng, wd *WonderData) {
	zn.Kind = "wonder"
	zn.Name = wd.Name
	cx, cy := 40, 12
	switch wd.Kind {
	case "world_tree":
		for y := 0; y < MH; y++ {
			for x := 0; x < MW; x++ {
				d := math.Hypot(float64(x-cx)/2, float64(y-cy))
				if d < 4 {
					zn.Set(x, y, TTrunk)
				} else if d < 11 && r.Chance(0.5) {
					zn.Set(x, y, TTree)
				} else if d < 13 {
					zn.Set(x, y, TMoss)
				}
			}
		}
	case "tower_magic", "tower_trials":
		for y := 0; y < MH; y++ {
			for x := 0; x < MW; x++ {
				d := math.Hypot(float64(x-cx)/2, float64(y-cy))
				if d < 6 && d >= 5 {
					zn.Set(x, y, TWall)
				} else if d < 5 {
					zn.Set(x, y, TFloor)
				}
			}
		}
		zn.Set(cx, cy+6, TDoor)
	default:
		g.building(zn, cx-12, cy-5, 25, 11, TWall, TFloor)
		for x := cx - 9; x <= cx+9; x += 3 {
			zn.Set(x, cy-2, TPillar)
			zn.Set(x, cy+2, TPillar)
		}
		zn.Set(cx, cy, TStatue)
		zn.Notes[cy*MW+cx] = fmt.Sprintf("A statue of %s.", nonEmpty(wd.Builder, "the builders"))
	}
	if wd.Destroyed {
		for i := 0; i < MW*MH/6; i++ {
			x, y := r.Int(0, MW-1), r.Int(0, MH-1)
			if zn.At(x, y) == TWall || zn.At(x, y) == TPillar {
				zn.Set(x, y, TRubble)
			}
		}
		zn.Name = "Ruins of " + wd.Name
	}
	px, py := cx-4, cy+7
	if wd.Kind == "world_tree" {
		px, py = cx-15, cy
	}
	zn.Set(px, py, TInscription)
	zn.Notes[py*MW+px] = fmt.Sprintf("{{Y|%s}}. Raised in the year %d by %s. %s %s", wd.Name, wd.Year, nonEmpty(wd.Builder, "unknown hands"), wd.Lore, wd.Effect)
	it := g.Loot(r, 4)
	it.X, it.Y = cx+1, cy+1
	if zn.Walkable(it.X, it.Y) {
		zn.Items = append(zn.Items, it)
	}
}

func nonEmpty(s, d string) string {
	if s == "" {
		return d
	}
	return s
}

// caves: cellular automata, then the deep layer's settlement or lair if Realmforge has one here
func (g *Game) genUnder(zn *Zone, r *Rng, p int) {
	w := g.W
	z := zn.Z
	zn.Dark = true
	ocean := w.IsWater(p)
	under := w.DeepOpen(p, z)
	sea := ocean && z == 1 && w.seaOpen[p]
	fill := 0.5
	zn.Kind, zn.Name = "strata", fmt.Sprintf("Strata, %d below", z)
	if under {
		fill = 0.4
		zn.Kind = "underdark"
		zn.Name = fmt.Sprintf("%s, depth %d", w.LayerName("deep", "Underdark"), z)
	}
	if sea {
		fill = 0.3
		zn.Kind, zn.Underwater = "seafloor", true
		zn.Name = w.LayerName("sea", "Sea Floor")
	}
	for i := range zn.T {
		if r.Chance(fill) {
			zn.T[i] = TRock
		} else {
			zn.T[i] = g.floorFor(zn)
		}
	}
	for it := 0; it < 4; it++ {
		nt := make([]Terrain, len(zn.T))
		copy(nt, zn.T)
		for y := 0; y < MH; y++ {
			for x := 0; x < MW; x++ {
				c := 0
				for dy := -1; dy <= 1; dy++ {
					for dx := -1; dx <= 1; dx++ {
						if zn.At(x+dx, y+dy) == TRock {
							c++
						}
					}
				}
				if c >= 5 {
					nt[y*MW+x] = TRock
				} else {
					nt[y*MW+x] = g.floorFor(zn)
				}
			}
		}
		zn.T = nt
	}
	seed := HashStr(w.Seed + zn.Key)
	for y := 0; y < MH; y++ {
		for x := 0; x < MW; x++ {
			if zn.At(x, y) == TRock {
				continue
			}
			n := noise2(seed, float64(x)/5, float64(y)/3)
			switch {
			case sea && n > 0.75:
				zn.Set(x, y, TCoral)
			case sea && n > 0.6:
				zn.Set(x, y, TKelp)
			case !sea && z >= 5 && n > 0.86:
				zn.Set(x, y, TLava)
			case !sea && n > 0.82:
				zn.Set(x, y, TShallow)
			case under && n < 0.1:
				zn.Set(x, y, TMushroom)
			case under && z%3 == 0 && n < 0.16:
				zn.Set(x, y, TCrystal)
			case n < 0.05:
				zn.Set(x, y, TBones)
			}
		}
	}
	if zn.ZX == 1 && zn.ZY == 1 {
		// Realmforge deep/sea settlements and lairs
		if bid, ok := w.deepBurg[z][p]; ok && !sea {
			if bd := w.burgByID[bid]; bd != nil {
				g.carveHall(zn, r)
				if bd.Alive {
					g.genTown(zn, r, bd)
				} else {
					g.genRuin(zn, r, bd)
				}
				zn.Dark = true
			}
		}
		if sea {
			if bid, ok := w.seaBurg[p]; ok {
				if bd := w.burgByID[bid]; bd != nil && bd.Alive {
					g.carveHall(zn, r)
					zn.Dry = make([]bool, MW*MH)
					for y := 0; y < MH; y++ {
						for x := 0; x < MW; x++ {
							if math.Hypot(float64(x-40)/2.2, float64(y-12)) < 10 {
								zn.Dry[y*MW+x] = true
							}
						}
					}
					g.genTown(zn, r, bd)
				}
			}
		}
		for _, mi := range w.monstersAt[p] {
			m := &w.Monsters[mi]
			if (m.Layer == "deep" && m.Depth == z || m.Layer == "sea" && sea) && !g.Slain[m.ID] {
				g.carveHall(zn, r)
				g.placeSimMonster(zn, r, m)
			}
		}
		if z == 1 && !ocean {
			if bd := w.burgByID[w.Burg[p]]; bd != nil && !bd.Alive {
				g.genCrypt(zn, r, bd)
			}
		}
	}
}

func (g *Game) carveHall(zn *Zone, r *Rng) {
	for y := 2; y < MH-2; y++ {
		for x := 6; x < MW-6; x++ {
			if math.Hypot(float64(x-40)/2.4, float64(y-12)) < 10+noise2(uint64(r.Int(0, 1<<30)), float64(x)/4, float64(y)/4)*2 {
				zn.Set(x, y, g.floorFor(zn))
			}
		}
	}
}

// genCrypt lays out the tombs of the dead rulers of the realm that held this ruin.
func (g *Game) genCrypt(zn *Zone, r *Rng, bd *BurgData) {
	w := g.W
	zn.Kind = "crypt"
	zn.Name = "Crypt beneath " + bd.Name
	for y := 3; y < MH-3; y++ {
		for x := 14; x < MW-14; x++ {
			if y == 3 || y == MH-4 || x == 14 || x == MW-15 {
				zn.Set(x, y, TWall)
			} else {
				zn.Set(x, y, TFloor)
			}
		}
	}
	for x := 18; x < MW-18; x += 4 {
		zn.Set(x, 6, TPillar)
		zn.Set(x, MH-7, TPillar)
	}
	var dead []*PersonData
	for i := range w.People {
		pp := &w.People[i]
		if !pp.Alive && (pp.State == bd.State || pp.Burg == bd.ID) && (pp.Role == "ruler" || pp.Title != "") {
			dead = append(dead, pp)
		}
	}
	sort.Slice(dead, func(i, j int) bool { return dead[i].Died < dead[j].Died })
	if len(dead) > 8 {
		dead = dead[len(dead)-8:]
	}
	for i, pp := range dead {
		x := 20 + (i%4)*11
		y := 9 + (i/4)*6
		zn.Set(x, y, TSarcophagus)
		title := nonEmpty(pp.Title, "ruler")
		zn.Notes[y*MW+x] = fmt.Sprintf("Here lies {{Y|%s}}, %s. Born %d, died %d%s.", pp.Name, title, pp.Born, pp.Died, causeText(pp.Cause))
		for _, a := range w.Artifacts {
			if a.HolderType == "person" && a.HolderID == pp.ID && !g.Taken[a.ID] {
				it := g.RelicItem(&a)
				it.X, it.Y = x, y+1
				zn.Items = append(zn.Items, it)
			}
		}
	}
	for i := 0; i < r.Int(2, 5); i++ {
		a := g.Spawn(Pick(r, []string{"skeleton", "skeleton", "ghoul", "wight"}), zn.Level, r)
		if x, y, ok := zn.FreeSpot(r, func(x, y int) bool { return x > 14 && x < MW-15 && y > 3 && y < MH-4 }); ok {
			a.X, a.Y = x, y
			zn.Actors = append(zn.Actors, a)
		}
	}
	zn.Set(40, MH-4, TDoorOpen)
}

func causeText(c string) string {
	if c == "" {
		return ""
	}
	return " (" + c + ")"
}

func (g *Game) genSky(zn *Zone, r *Rng, p int) {
	w := g.W
	zn.Kind, zn.Sky = "sky", true
	zn.Name = w.LayerName("sky", "Sky")
	open := w.skyOpen[p]
	seed := HashStr(w.Seed + "sky" + fmt.Sprint(p))
	for y := 0; y < MH; y++ {
		for x := 0; x < MW; x++ {
			gx, gy := gxy(zn, x, y)
			n := fbm(seed, gx/16, gy/6)
			t := TVoid
			if open && n > 0.52 {
				t = TGrass
				if n > 0.7 && noise2(seed+3, gx/2, gy) > 0.6 {
					t = TTree
				}
			} else if n > 0.47 && (open || n > 0.62) {
				t = TCloud
			}
			zn.T[y*MW+x] = t
		}
	}
	if !open {
		zn.Name = "Open sky"
	}
	if open && zn.ZX == 1 && zn.ZY == 1 {
		if bid, ok := w.skyBurg[p]; ok {
			if bd := w.burgByID[bid]; bd != nil {
				for y := 2; y < MH-2; y++ {
					for x := 10; x < MW-10; x++ {
						if math.Hypot(float64(x-40)/2.2, float64(y-12)) < 10 {
							zn.Set(x, y, TGrass)
						}
					}
				}
				if bd.Alive {
					g.genTown(zn, r, bd)
				} else {
					g.genRuin(zn, r, bd)
				}
			}
		}
		for _, mi := range w.monstersAt[p] {
			m := &w.Monsters[mi]
			if m.Layer == "sky" && !g.Slain[m.ID] {
				g.placeSimMonster(zn, r, m)
			}
		}
	}
}

// connect guarantees every walkable region reaches the zone's edges and stairs (tunnels through rock).
func (g *Game) connect(zn *Zone) {
	if zn.Sky {
		return
	}
	floor := g.floorFor(zn)
	targets := [][2]int{{40, 0}, {40, MH - 1}, {0, 12}, {MW - 1, 12}}
	if zn.Kind == "dungeon" || zn.Kind == "crypt" {
		targets = nil // sealed levels: only the stairs need linking
	}
	for y := 0; y < MH; y++ {
		for x := 0; x < MW; x++ {
			t := zn.At(x, y)
			if t == TDown || t == TUp || t == TUpdraft {
				targets = append(targets, [2]int{x, y})
			}
		}
	}
	cx, cy := 40, 12
	// find a walkable tile near the centre
	best := 1 << 30
	for y := 0; y < MH; y++ {
		for x := 0; x < MW; x++ {
			if zn.Walkable(x, y) && zn.At(x, y) != TDeep && zn.At(x, y) != TLava {
				d := abs(x-40) + abs(y-12)*2
				if d < best {
					best, cx, cy = d, x, y
				}
			}
		}
	}
	for _, tg := range targets {
		x, y := tg[0], tg[1]
		reach := g.reachable(zn, cx, cy)
		if reach[y*MW+x] {
			continue
		}
		// walk from the target toward the centre carving until we touch the reached region
		for i := 0; i < 400 && !(x == cx && y == cy); i++ {
			if reach[y*MW+x] {
				break
			}
			t := zn.At(x, y)
			if !t.D().Walk || t == TLava || t == TChasm {
				if zn.Kind == "town" && t == TWall {
					zn.Set(x, y, TDoorOpen)
				} else {
					zn.Set(x, y, floor)
				}
			}
			if abs(cx-x) > abs(cy-y)*2 {
				x += sign(cx - x)
			} else {
				y += sign(cy - y)
			}
		}
	}
}

func (g *Game) reachable(zn *Zone, sx, sy int) []bool {
	seen := make([]bool, MW*MH)
	q := []int{sy*MW + sx}
	seen[sy*MW+sx] = true
	for len(q) > 0 {
		i := q[0]
		q = q[1:]
		x, y := i%MW, i/MW
		for _, d := range dirs8 {
			nx, ny := x+d[0], y+d[1]
			if !zn.In(nx, ny) {
				continue
			}
			ni := ny*MW + nx
			t := zn.At(nx, ny)
			if seen[ni] || !t.D().Walk || t == TLava || t == TChasm {
				continue
			}
			seen[ni] = true
			q = append(q, ni)
		}
	}
	return seen
}

func (g *Game) computeLight(zn *Zone) {
	for i := range zn.Lit {
		zn.Lit[i] = false
	}
	for y := 0; y < MH; y++ {
		for x := 0; x < MW; x++ {
			rad := 0
			switch zn.At(x, y) {
			case TBrazier:
				rad = 6
			case TLava:
				rad = 2
			case TCrystal, TMushroom, TAltar:
				rad = 2
			case TRift:
				rad = 3
			}
			if zn.Fire[y*MW+x] > 0 {
				rad = 3
			}
			if rad == 0 {
				continue
			}
			for dy := -rad; dy <= rad; dy++ {
				for dx := -rad; dx <= rad; dx++ {
					if dx*dx+dy*dy <= rad*rad && zn.In(x+dx, y+dy) {
						zn.Lit[(y+dy)*MW+x+dx] = true
					}
				}
			}
		}
	}
}

// populate seeds wildlife for the zone's biome and depth.
func (g *Game) populate(zn *Zone, r *Rng) {
	where := byte('s')
	switch {
	case zn.Underwater:
		where = 'e'
	case zn.Sky:
		where = 'k'
	case zn.Z > 0:
		where = 'd'
	}
	n := r.Int(1, 4)
	if zn.Kind == "outskirts" {
		n = r.Int(0, 1)
	}
	if zn.Kind == "sea" {
		n = 0
	}
	if zn.Z > 0 {
		n += zn.Z / 2
	}
	for i := 0; i < n; i++ {
		tp := pickCreature(r, zn.Biome, where, zn.Level)
		count := 1
		if t := tplByID[tp]; t != nil && t.Has("pack") {
			count = r.Int(2, 4)
		}
		x, y, ok := zn.FreeSpot(r, nil)
		if !ok {
			continue
		}
		for k := 0; k < count; k++ {
			a := g.Spawn(tp, zn.Level+r.Int(-1, 1), r)
			ax, ay := x+r.Int(-2, 2), y+r.Int(-1, 1)
			if zn.Walkable(ax, ay) && zn.ActorAt(ax, ay) == nil && (zn.WaterAt(ax, ay) < 2 || a.Has("aquatic")) {
				a.X, a.Y = ax, ay
				zn.Actors = append(zn.Actors, a)
			}
		}
	}
	if r.Chance(0.25) || zn.Z > 0 && r.Chance(0.4) {
		if x, y, ok := zn.FreeSpot(r, nil); ok {
			it := g.Loot(r, zn.Level/3)
			it.X, it.Y = x, y
			zn.Items = append(zn.Items, it)
		}
	}
}
