package engine

import (
	"fmt"
	"sort"
	"strings"
)

// Cities: a Realmforge town grows from one zone to the whole 3×3 parasang. Every zone of the
// footprint is a district chosen from the town's economic sectors (trade → market, faith →
// temple, military → barracks/arena, mining → mines, ...), its port, and whether it is a capital.

type cityPlan struct {
	tier      int               // 0 single zone, 1 plus-shape, 2 full 3×3
	districts map[[2]int]string // sub-zone → district kind
	walled    bool
}

var sideOrder = [][2]int{{1, 0}, {2, 1}, {1, 2}, {0, 1}, {2, 0}, {2, 2}, {0, 2}, {0, 0}}

func (g *Game) cityPlanFor(bd *BurgData) *cityPlan {
	if g.plans == nil {
		g.plans = map[int]*cityPlan{}
	}
	if p, ok := g.plans[bd.ID]; ok {
		return p
	}
	w := g.W
	cp := &cityPlan{districts: map[[2]int]string{}, walled: bd.Capital || bd.Districts["walls"] > 0 || w.cityTier[bd.ID] == 2}
	cp.tier = w.cityTier[bd.ID]
	center := "plaza"
	if bd.Capital {
		center = "palace"
	}
	cp.districts[[2]int{1, 1}] = center
	if cp.tier == 0 {
		g.plans[bd.ID] = cp
		return cp
	}
	type cand struct {
		kind string
		w    float64
	}
	sec := func(n string) float64 { return w.Sector(bd, n) }
	cs := []cand{
		{"market", 0.5 + sec("trade")*3 + float64(bd.Districts["market"])*0.4},
		{"temple", 0.25 + sec("faith")*3 + float64(bd.Districts["temple"])*0.4},
		{"academy", 0.1 + sec("scholarship")*3 + float64(bd.Districts["academy"])*0.4},
		{"barracks", 0.15 + sec("military")*3 + float64(bd.Garrison)/40},
		{"forge", 0.15 + sec("industry")*3},
		{"artisans", 0.1 + sec("crafts")*3},
		{"mines", sec("mining") * 3},
		{"farms", 0.1 + sec("farming")*2},
		{"slums", 0.2},
		{"homes", 0.3},
	}
	if bd.Port {
		cs = append(cs, cand{"harbor", 2 + sec("seafaring")*3})
	}
	if cp.tier == 2 || sec("military") > 0.2 {
		cs = append(cs, cand{"arena", 0.45 + sec("military")*2})
	}
	sort.SliceStable(cs, func(i, j int) bool { return cs[i].w > cs[j].w })
	slots := sideOrder[:4]
	if cp.tier == 2 {
		slots = sideOrder
	}
	// a harbor goes on the side that faces the sea when there is one
	used := map[[2]int]bool{}
	if bd.Port {
		px, py := bd.P%w.W, bd.P/w.W
		for _, s := range slots {
			dx, dy := s[0]-1, s[1]-1
			if w.In(px+dx, py+dy) && w.IsWater(w.Idx(px+dx, py+dy)) {
				cp.districts[s] = "harbor"
				used[s] = true
				break
			}
		}
	}
	i := 0
	for _, s := range slots {
		if used[s] {
			continue
		}
		for i < len(cs) && (cs[i].kind == "harbor" && hasDistrict(cp, "harbor")) {
			i++
		}
		kind := "homes"
		if i < len(cs) {
			kind = cs[i].kind
			i++
		}
		cp.districts[s] = kind
	}
	g.plans[bd.ID] = cp
	return cp
}

func hasDistrict(cp *cityPlan, k string) bool {
	for _, v := range cp.districts {
		if v == k {
			return true
		}
	}
	return false
}

var districtNames = map[string]string{
	"plaza": "Plaza", "palace": "Palace Quarter", "market": "Market District", "temple": "Temple Quarter", "academy": "Scholars' Quarter",
	"barracks": "Barracks", "forge": "Forge Ward", "artisans": "Artisans' Row", "mines": "Mine Heads", "farms": "Town Fields", "slums": "the Warrens",
	"homes": "Residential Ward", "harbor": "Harbor", "arena": "Arena",
}

// genDistrict paints one city zone.
func (g *Game) genDistrict(zn *Zone, r *Rng, bd *BurgData, cp *cityPlan, kind string) {
	w := g.W
	zn.Kind, zn.Burg = "town", bd.ID
	zn.Faction = fmt.Sprintf("state:%d", bd.State)
	zn.WallF = g.Sk.WallColor
	zn.Name = bd.Name + ", " + districtNames[kind]
	cu := w.cultByID[bd.Culture]
	// ground: cobbles over the whole zone except water and the harbor's sea side
	for y := 0; y < MH; y++ {
		for x := 0; x < MW; x++ {
			if zn.Dry != nil && !zn.Dry[y*MW+x] {
				continue
			}
			t := zn.At(x, y)
			if kind == "farms" || kind == "mines" {
				if t.D().Water > 0 || t == TRock && kind == "mines" {
					continue
				}
				zn.Set(x, y, TDirt)
				continue
			}
			zn.Set(x, y, TCobble)
		}
	}
	for x := 0; x < MW; x++ {
		zn.Set(x, 12, TRoad)
	}
	for y := 0; y < MH; y++ {
		zn.Set(40, y, TRoad)
	}
	// city walls along edges that face open country
	if cp.walled && zn.Z == 0 {
		g.cityWalls(zn, cp)
	}
	add := func(role, title string, x, y int) *Actor {
		a := g.makeNPC(role, bd, cu, r)
		if title != "" {
			a.Title = title
		}
		a.X, a.Y, a.HomeX, a.HomeY = x, y, x, y
		if !zn.Walkable(x, y) || zn.ActorAt(x, y) != nil {
			g.nudge(zn, a)
			a.HomeX, a.HomeY = a.X, a.Y
		}
		zn.Actors = append(zn.Actors, a)
		return a
	}
	shop := func(role, title string) {
		if rc, ok := g.lot(zn, r, 7, 11, 4, 6); ok {
			dx, dy := g.building(zn, rc[0], rc[1], rc[2], rc[3], TWall, TFloor)
			_ = dx
			_ = dy
			zn.Set(rc[0]+1, rc[1]+1, TStall)
			add(role, title, rc[0]+rc[2]/2, rc[1]+rc[3]/2)
		}
	}
	crowd := func(n int) {
		for i := 0; i < n; i++ {
			if x, y, ok := zn.FreeSpot(r, func(x, y int) bool { return zn.At(x, y) == TCobble || zn.At(x, y) == TRoad }); ok {
				add("villager", "", x, y)
			}
		}
	}
	guards := func(n int) {
		for i := 0; i < n; i++ {
			if x, y, ok := zn.FreeSpot(r, nil); ok {
				add("guard", "", x, y)
			}
		}
	}

	switch kind {
	case "plaza", "palace":
		for y := 8; y <= 16; y++ {
			for x := 28; x <= 52; x++ {
				zn.Set(x, y, TCobble)
			}
		}
		zn.Set(40, 12, TWell)
		zn.Set(44, 10, TInscription)
		zn.Notes[10*MW+44] = g.townPlaque(bd)
		hall := [4]int{30, 1, 21, 6}
		if kind == "palace" {
			hall = [4]int{22, 1, 37, 7}
		}
		g.building(zn, hall[0], hall[1], hall[2], hall[3], TWall, TCarpet)
		cx := hall[0] + hall[2]/2
		if kind == "palace" {
			zn.Set(cx, hall[1]+1, TThrone)
			for x := hall[0] + 3; x < hall[0]+hall[2]-3; x += 5 {
				zn.Set(x, hall[1]+3, TPillar)
			}
		}
		// the town's people from the chronicle hold court here
		notables := w.peopleByBrg[bd.ID]
		for i, ni := range notables {
			if i >= 10 {
				break
			}
			a := g.npcFromPerson(&w.People[ni], bd, r)
			if i == 0 {
				a.Role = "elder"
				a.X, a.Y = cx, hall[1]+2
			} else {
				a.X, a.Y = hall[0]+2+(i*3)%(hall[2]-4), hall[1]+3
			}
			a.HomeX, a.HomeY = a.X, a.Y
			g.nudge(zn, a)
			zn.Actors = append(zn.Actors, a)
		}
		if len(notables) == 0 {
			add("elder", "", cx, hall[1]+2)
		}
		if !hasDistrict(cp, "market") {
			zn.Set(34, 14, TStall)
			zn.Set(46, 14, TStall)
			add("merchant", "", 34, 15)
		}
		if !hasDistrict(cp, "temple") {
			if rc, ok := g.lot(zn, r, 9, 11, 5, 6); ok {
				g.building(zn, rc[0], rc[1], rc[2], rc[3], TWall, TCarpet)
				zn.Set(rc[0]+rc[2]/2, rc[1]+1, TAltar)
				add("priest", "", rc[0]+rc[2]/2, rc[1]+2)
			}
		}
		if bd.Port && !hasDistrict(cp, "harbor") {
			add("ferry", "harbourmaster", 36, 15)
		}
		guards(2 + clampi(bd.Garrison/10, 0, 4))
		crowd(4)
	case "market":
		for y := 3; y < MH-3; y += 4 {
			for x := 6; x < MW-6; x += 6 {
				if y != 12 && x != 40 && zn.At(x, y) == TCobble && r.Chance(0.7) {
					zn.Set(x, y, TStall)
				}
			}
		}
		shop("merchant", "")
		shop("smith", "")
		shop("apothecary", "")
		shop("bookseller", "bookseller")
		shop("exotic", "curio dealer")
		shop("tailor", "tailor")
		if bd.Port || r.Chance(0.4) {
			shop("merchant", "spice trader")
		}
		if bd.League != "" {
			if rc, ok := g.lot(zn, r, 11, 13, 5, 6); ok {
				g.building(zn, rc[0], rc[1], rc[2], rc[3], TWall, TCarpet)
				x, y := rc[0]+rc[2]/2, rc[1]+1
				zn.Set(x, y, TInscription)
				zn.Notes[y*MW+x] = fmt.Sprintf("Hall of the {{Y|%s}}. Members trade under its charter.", bd.League)
				add("merchant", "league factor", x, y+1)
			}
		}
		for i := 0; i < 6; i++ {
			x, y := r.Int(3, MW-4), r.Int(2, MH-3)
			if zn.At(x, y) == TCobble {
				zn.Set(x, y, TCrate)
			}
		}
		crowd(8)
		guards(2)
	case "harbor":
		// the sea side: water past a quay, piers and moored ships
		side := g.seaSide(zn, bd)
		for y := 0; y < MH; y++ {
			for x := 0; x < MW; x++ {
				if seaward(side, x, y) > 0 {
					zn.Set(x, y, TDeep)
				} else if seaward(side, x, y) == 0 {
					zn.Set(x, y, TBridge)
				}
			}
		}
		for k := 0; k < 4; k++ {
			g.pier(zn, side, 8+k*18+r.Int(-2, 2), r)
		}
		for i := 0; i < 10; i++ {
			x, y := r.Int(3, MW-4), r.Int(2, MH-3)
			if zn.At(x, y) == TCobble {
				zn.Set(x, y, TCrate)
			}
		}
		if rc, ok := g.lot(zn, r, 12, 16, 5, 7); ok {
			g.building(zn, rc[0], rc[1], rc[2], rc[3], TWall, TFloor)
			for x := rc[0] + 1; x < rc[0]+rc[2]-1; x += 2 {
				zn.Set(x, rc[1]+1, TCrate)
			}
		}
		add("ferry", "harbourmaster", 40, 13)
		shop("merchant", "fishmonger")
		for i := 0; i < 4; i++ {
			if x, y, ok := zn.FreeSpot(r, nil); ok {
				add("villager", Pick(r, []string{"sailor", "dockhand", "fisher", "shipwright"}), x, y)
			}
		}
		guards(1)
	case "temple":
		x0, y0 := 26, 2
		g.building(zn, x0, y0, 29, 9, TWall, TCarpet)
		for x := x0 + 3; x < x0+27; x += 4 {
			zn.Set(x, y0+3, TPillar)
			zn.Set(x, y0+6, TPillar)
		}
		zn.Set(40, y0+1, TAltar)
		rel := w.relByID[w.Religion[clampi(bd.P, 0, len(w.Religion)-1)]]
		if rel != nil {
			zn.Notes[(y0+1)*MW+40] = fmt.Sprintf("The high altar of the %s, sacred to %s.", rel.Name, rel.Deity)
			zn.Name = bd.Name + ", " + "Temple of " + rel.Deity
		}
		add("priest", "high priest", 40, y0+2)
		add("priest", "", 34, y0+4)
		// graveyard of the realm's dead from the chronicle
		var dead []*PersonData
		for i := range w.People {
			pp := &w.People[i]
			if !pp.Alive && pp.State == bd.State {
				dead = append(dead, pp)
			}
		}
		for i := 0; i < 14; i++ {
			x, y := 6+(i%7)*4, 15+(i/7)*3
			if zn.At(x, y) != TCobble && zn.At(x, y) != TRoad {
				continue
			}
			zn.Set(x, y, TInscription)
			if i < len(dead) {
				pp := dead[len(dead)-1-i]
				zn.Notes[y*MW+x] = fmt.Sprintf("A gravestone: {{Y|%s}}, %s. %d–%d%s.", pp.Name, nonEmpty(pp.Title, pp.Role), pp.Born, pp.Died, causeText(pp.Cause))
			} else {
				zn.Notes[y*MW+x] = "A weathered gravestone. The name has worn away."
			}
		}
		for i := 0; i < 4; i++ {
			if x, y, ok := zn.FreeSpot(r, nil); ok {
				add("villager", "pilgrim", x, y)
			}
		}
	case "academy":
		g.building(zn, 24, 2, 33, 9, TWall, TFloor)
		for y := 4; y <= 8; y += 2 {
			for x := 26; x < 55; x++ {
				if x%6 != 0 {
					zn.Set(x, y, TShelf)
				}
			}
		}
		add("trainer", "master of lore", 40, 3).Teach = "lore,tinkering,persuasion"
		shop("bookseller", "archivist")
		for i := 0; i < 4; i++ {
			if x, y, ok := zn.FreeSpot(r, nil); ok {
				add("villager", Pick(r, []string{"student", "scribe", "scholar", "astronomer"}), x, y)
			}
		}
	case "barracks":
		g.building(zn, 6, 2, 26, 8, TWall, TFloor)
		for x := 8; x < 30; x += 3 {
			zn.Set(x, 3, TBed)
		}
		for y := 15; y < 22; y += 3 {
			for x := 46; x < 72; x += 5 {
				zn.Set(x, y, TDummy)
			}
		}
		add("trainer", "drillmaster", 50, 14).Teach = "longblade,axe,cudgel,missile,endurance,acrobatics"
		add("smith", "armorer", 20, 6)
		guards(4 + clampi(bd.Garrison/6, 0, 8))
	case "arena":
		cx, cy := 40, 12
		for y := 1; y < MH-1; y++ {
			for x := 6; x < MW-6; x++ {
				d := sq(float64(x-cx)/2.6) + sq(float64(y-cy))
				switch {
				case d < 64:
					zn.Set(x, y, TSand)
				case d < 100:
					zn.Set(x, y, TSeat)
				case d < 121:
					zn.Set(x, y, TWall)
				}
			}
		}
		zn.Set(cx, cy+10, TDoorOpen)
		zn.Set(cx, cy-10, TDoorOpen)
		zn.Set(cx, cy+11, TCobble)
		a := add("arena", "arena master", cx+2, cy+9)
		a.AI = "still"
		for i := 0; i < 3; i++ {
			add("villager", "gladiator", cx-8+i*8, cy+7)
		}
		zn.Name = bd.Name + ", the Arena"
	case "forge":
		for k := 0; k < 4; k++ {
			if rc, ok := g.lot(zn, r, 8, 11, 5, 6); ok {
				g.building(zn, rc[0], rc[1], rc[2], rc[3], TWall, TFloor)
				zn.Set(rc[0]+1, rc[1]+1, TBrazier)
				zn.Set(rc[0]+3, rc[1]+1, TAnvil)
				if k < 2 {
					add("smith", "master smith", rc[0]+rc[2]/2, rc[1]+2)
				} else if k == 2 {
					add("trainer", "tinker", rc[0]+rc[2]/2, rc[1]+2).Teach = "tinkering"
				}
			}
		}
		shop("exotic", "machinist")
		crowd(3)
	case "artisans":
		shop("tailor", "clothier")
		for k := 0; k < 5; k++ {
			if rc, ok := g.lot(zn, r, 7, 10, 4, 6); ok {
				g.building(zn, rc[0], rc[1], rc[2], rc[3], TWall, TFloor)
				if k < 3 {
					add("exotic", Pick(r, []string{"jeweller", "potter", "weaver", "glassblower", "painter", "carver"}), rc[0]+rc[2]/2, rc[1]+2)
				}
			}
		}
		crowd(5)
	case "mines":
		for i := 0; i < 260; i++ {
			x, y := r.Int(1, MW-2), r.Int(1, MH-2)
			if abs(x-40) > 2 && abs(y-12) > 1 && zn.At(x, y) == TDirt {
				if r.Chance(0.15) {
					zn.Set(x, y, TOre)
				} else {
					zn.Set(x, y, TRock)
				}
			}
		}
		g.building(zn, 50, 3, 12, 5, TWall, TFloor)
		add("merchant", "mine foreman", 56, 5)
		for i := 0; i < 4; i++ {
			if x, y, ok := zn.FreeSpot(r, nil); ok {
				add("villager", "miner", x, y)
			}
		}
	case "farms":
		for y := 2; y < MH-2; y++ {
			for x := 3; x < MW-3; x++ {
				if (y/3)%2 == 0 && x%13 != 0 && zn.At(x, y) == TDirt && y != 12 && x != 40 {
					zn.Set(x, y, TCrop)
				}
			}
		}
		g.building(zn, 46, 14, 12, 6, TWall, TFloor)
		add("merchant", "granary keeper", 52, 16)
		for i := 0; i < 4; i++ {
			if x, y, ok := zn.FreeSpot(r, nil); ok {
				add("villager", "farmer", x, y)
			}
		}
	case "slums", "homes":
		n, wmin, wmax := 10, 6, 9
		if kind == "slums" {
			n, wmin, wmax = 18, 4, 6
		}
		for k := 0; k < n; k++ {
			if rc, ok := g.lot(zn, r, wmin, wmax, 4, 5); ok {
				g.building(zn, rc[0], rc[1], rc[2], rc[3], TWall, TFloor)
				zn.Set(rc[0]+1, rc[1]+1, TBed)
			}
		}
		if kind == "slums" {
			add("trainer", "fence", 20, 13).Teach = "shortblade,acrobatics,persuasion"
			for i := 0; i < 5; i++ {
				if x, y, ok := zn.FreeSpot(r, nil); ok {
					add("villager", Pick(r, []string{"beggar", "urchin", "drunk", "pickpocket", "rag-picker"}), x, y)
				}
			}
		} else {
			crowd(6)
		}
	}
	g.placeLandmarks(zn, r, bd, kind)
	g.assignSociety(zn, r, bd)
}

func sq(v float64) float64 { return v * v }

// lot finds a free rectangle of cobbles for a building, away from the main roads.
func (g *Game) lot(zn *Zone, r *Rng, wmin, wmax, hmin, hmax int) ([4]int, bool) {
	for tries := 0; tries < 200; tries++ {
		w, h := r.Int(wmin, wmax), r.Int(hmin, hmax)
		x, y := r.Int(2, MW-w-2), r.Int(1, MH-h-1)
		if x <= 41 && x+w >= 39 || y <= 13 && y+h >= 11 {
			continue
		}
		ok := true
		for yy := y - 1; yy <= y+h && ok; yy++ {
			for xx := x - 1; xx <= x+w; xx++ {
				t := zn.At(xx, yy)
				if t != TCobble && t != TDirt && t != TRoad {
					ok = false
					break
				}
			}
		}
		if ok {
			return [4]int{x, y, w, h}, true
		}
	}
	return [4]int{}, false
}

// cityWalls rings the edges of a city zone that border open country, with gates on the roads.
func (g *Game) cityWalls(zn *Zone, cp *cityPlan) {
	in := func(x, y int) bool {
		_, ok := cp.districts[[2]int{x, y}]
		return ok && x >= 0 && y >= 0 && x < 3 && y < 3
	}
	if !in(zn.ZX, zn.ZY-1) {
		for x := 0; x < MW; x++ {
			zn.Set(x, 0, TWall)
		}
		zn.Set(40, 0, TDoorOpen)
	}
	if !in(zn.ZX, zn.ZY+1) {
		for x := 0; x < MW; x++ {
			zn.Set(x, MH-1, TWall)
		}
		zn.Set(40, MH-1, TDoorOpen)
	}
	if !in(zn.ZX-1, zn.ZY) {
		for y := 0; y < MH; y++ {
			zn.Set(0, y, TWall)
		}
		zn.Set(0, 12, TDoorOpen)
	}
	if !in(zn.ZX+1, zn.ZY) {
		for y := 0; y < MH; y++ {
			zn.Set(MW-1, y, TWall)
		}
		zn.Set(MW-1, 12, TDoorOpen)
	}
}

// seaSide: which edge of a harbor zone faces the water (0 N, 1 E, 2 S, 3 W)
func (g *Game) seaSide(zn *Zone, bd *BurgData) int {
	w := g.W
	dx, dy := zn.ZX-1, zn.ZY-1
	px, py := bd.P%w.W, bd.P/w.W
	if w.In(px+dx, py+dy) && w.IsWater(w.Idx(px+dx, py+dy)) {
		switch {
		case dx > 0:
			return 1
		case dx < 0:
			return 3
		case dy < 0:
			return 0
		}
		return 2
	}
	switch {
	case dx > 0:
		return 1
	case dx < 0:
		return 3
	case dy < 0:
		return 0
	}
	return 2
}

// seaward: >0 water, 0 quay, <0 land
func seaward(side, x, y int) int {
	switch side {
	case 0:
		return 7 - y
	case 1:
		return x - (MW - 24)
	case 2:
		return y - (MH - 8)
	}
	return 23 - x
}

func (g *Game) pier(zn *Zone, side, at int, r *Rng) {
	for k := 0; k < 30; k++ {
		var x, y int
		switch side {
		case 0:
			x, y = at, 7-k
		case 1:
			x, y = MW-24+k, clampi(at/3+2, 2, MH-3)
		case 2:
			x, y = at, MH-8+k
		default:
			x, y = 23-k, clampi(at/3+2, 2, MH-3)
		}
		if !zn.In(x, y) {
			return
		}
		if zn.At(x, y) == TDeep {
			zn.Set(x, y, TBridge)
			if k == 6 && r.Chance(0.7) {
				for dy := -1; dy <= 1; dy++ {
					for dx := 1; dx <= 6; dx++ {
						if side%2 == 0 {
							zn.Set(x+dx, y+dy*2, TBoat)
						} else if zn.At(x+dx-3, y+dy+2) == TDeep {
							zn.Set(x+dx-3, y+dy+2, TBoat)
						}
					}
				}
			}
		}
	}
}

func (g *Game) districtAt(wx, wy, zx, zy int) (string, *BurgData) {
	w := g.W
	if !w.In(wx, wy) {
		return "", nil
	}
	bd := w.burgByID[w.Burg[w.Idx(wx, wy)]]
	if bd == nil || !bd.Alive || bd.Layer != "surface" {
		return "", nil
	}
	return g.cityPlanFor(bd).districts[[2]int{zx, zy}], bd
}

func districtTitle(kind string) string { return strings.ToLower(districtNames[kind]) }
