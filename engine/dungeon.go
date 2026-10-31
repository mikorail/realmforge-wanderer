package engine

import (
	"fmt"
	"math"
	"sort"
)

// Site is a dungeon column (z = 1..Depth under one surface zone) or a tower (floors above it,
// stored at z = -100-floor). Sites come from Realmforge (ruins, temples, mines, outlaw camps,
// wonders, the Spire of Trials and the Bottomless Abyss) plus seeded wilderness dungeons.
type Site struct {
	Kind   string // dungeon tower
	Theme  string // catacomb sunken arcane chasm barrow mine hideout vault abyss magic trials
	Name   string
	Depth  int // dungeon levels or tower floors
	Bonus  int // difficulty on top of the zone level
	Gold   int
	Outlaw *OutlawData
	Ruin   bool
}

const towerBase = -100

func siteKey(wx, wy, zx, zy int) string { return fmt.Sprintf("%d.%d.%d.%d", wx, wy, zx, zy) }

// siteAt returns the site rooted at a surface zone (cached, deterministic).
func (g *Game) siteAt(wx, wy, zx, zy int) *Site {
	if !g.W.In(wx, wy) {
		return nil
	}
	if g.sites == nil {
		g.sites = map[string]*Site{}
	}
	k := siteKey(wx, wy, zx, zy)
	if s, ok := g.sites[k]; ok {
		return s
	}
	s := g.findSite(wx, wy, zx, zy)
	g.sites[k] = s
	return s
}

func (g *Game) findSite(wx, wy, zx, zy int) *Site {
	w := g.W
	p := w.Idx(wx, wy)
	center := zx == 1 && zy == 1
	r := NewRng(w.Seed + ":site:" + siteKey(wx, wy, zx, zy))
	sk := g.Sk
	if center {
		for _, m := range w.Megas {
			if m.P != p {
				continue
			}
			if m.Kind == "abyss" {
				return &Site{Kind: "dungeon", Theme: "abyss", Name: m.Name, Depth: 12, Bonus: 2}
			}
			return &Site{Kind: "tower", Theme: "trials", Name: m.Name, Depth: 20, Bonus: 1}
		}
		for _, wi := range w.wondersAt[p] {
			wd := &w.Wonders[wi]
			if wd.Kind == "tower_magic" || wd.Kind == "tower_trials" {
				th := "magic"
				if wd.Kind == "tower_trials" {
					th = "trials"
				}
				return &Site{Kind: "tower", Theme: th, Name: wd.Name, Depth: 8 + r.Int(0, 4), Bonus: 1}
			}
		}
		if bd := w.burgByID[w.Burg[p]]; bd != nil && !bd.Alive && bd.Layer == "surface" {
			return &Site{Kind: "dungeon", Theme: "catacomb", Name: "Catacombs of " + bd.Name, Depth: 3, Ruin: true}
		}
	}
	if kind, bd := g.districtAt(wx, wy, zx, zy); bd != nil {
		switch kind {
		case "temple":
			return &Site{Kind: "dungeon", Theme: "catacomb", Name: "Catacombs beneath " + bd.Name, Depth: 2}
		case "mines":
			return &Site{Kind: "dungeon", Theme: "mine", Name: "Mines of " + bd.Name, Depth: 3}
		case "slums":
			if ol := w.outlawByBrg[bd.ID]; ol != nil {
				return &Site{Kind: "dungeon", Theme: "hideout", Name: ol.Name, Depth: 2, Gold: ol.Gold, Outlaw: ol}
			}
			return &Site{Kind: "dungeon", Theme: "hideout", Name: "Thieves' warren of " + bd.Name, Depth: 2}
		}
		if kind != "" {
			return nil
		}
	}
	if bd := w.burgByID[w.Burg[p]]; bd != nil && bd.Alive {
		if ol := w.outlawByBrg[bd.ID]; ol != nil && g.outlawZone(bd) == [2]int{zx, zy} {
			return &Site{Kind: "dungeon", Theme: "hideout", Name: ol.Name, Depth: 2, Gold: ol.Gold, Outlaw: ol}
		}
		return nil
	}
	if w.IsWater(p) || w.Burg[p] > 0 {
		return nil
	}
	// seeded wilderness: about one dungeon per 12 parasangs, one tower per 30
	h := NewRng(w.Seed + ":wild:" + fmt.Sprint(p))
	roll := h.Float()
	pick := [2]int{h.Int(0, 2), h.Int(0, 2)}
	if pick != [2]int{zx, zy} {
		return nil
	}
	b := w.Biome[p]
	theme := "barrow"
	switch {
	case g.Genre != "fantasy" && g.Genre != "mythic" && h.Chance(0.5):
		theme = "vault"
	case b == 12 || b == 8 || b == 7:
		theme = "sunken"
	case b == 14:
		theme = "arcane"
	case w.Height[p] >= 60 || b == 13 || b == 16:
		theme = "chasm"
	case h.Chance(0.4):
		theme = "catacomb"
	}
	if roll < 0.035 {
		return &Site{Kind: "tower", Theme: "magic", Name: g.siteName("tower", p, h), Depth: 5 + h.Int(0, 4)}
	}
	if roll < 0.12 {
		return &Site{Kind: "dungeon", Theme: theme, Name: g.siteName(theme, p, h), Depth: 2 + h.Int(0, 3)}
	}
	_ = sk
	return nil
}

var themeNouns = map[string][]string{
	"catacomb": {"Ossuary", "Catacombs", "Bone Vaults"},
	"sunken":   {"Drowned Vault", "Sunken Halls", "Flooded Crypt"},
	"arcane":   {"Labyrinth", "Prism Vaults", "Singing Maze"},
	"chasm":    {"Chasm", "Deep Fissure", "Black Rift"},
	"barrow":   {"Barrow", "Howe", "Mound of Kings"},
	"vault":    {"Vault", "Bunker", "Sealed Complex"},
	"tower":    {"Tower", "Spire", "Observatory"},
}

func (g *Game) siteName(theme string, p int, r *Rng) string {
	cu := g.W.cultByID[g.W.Culture[p]]
	place := g.W.RegionName(p)
	if cu != nil && len(cu.Places) > 0 {
		place = Pick(r, cu.Places)
	}
	noun := Pick(r, themeNouns[theme])
	if theme == "vault" && g.Genre == "cyberpunk" {
		noun = Pick(r, []string{"Data Vault", "Dead Arcology", "Black Site"})
	}
	return fmt.Sprintf("%s of %s", noun, place)
}

// outlawZone: the sub-zone of a town's parasang where its outlaw camp sits (first one outside the city)
func (g *Game) outlawZone(bd *BurgData) [2]int {
	cp := g.cityPlanFor(bd)
	for _, s := range [][2]int{{0, 1}, {2, 1}, {1, 0}, {1, 2}, {0, 0}, {2, 2}, {0, 2}, {2, 0}} {
		if _, ok := cp.districts[s]; !ok {
			return s
		}
	}
	return [2]int{-1, -1}
}

func posFor(seed string) (int, int) {
	r := NewRng(seed)
	return r.Int(10, MW-11), r.Int(4, MH-5)
}

// entrance draws the visible structure around a site's way in on the surface.
func (g *Game) siteEntrance(zn *Zone, s *Site, x, y int) {
	if s.Kind == "tower" {
		for dy := -4; dy <= 4; dy++ {
			for dx := -8; dx <= 8; dx++ {
				d := math.Hypot(float64(dx)/2, float64(dy))
				if d < 3.6 {
					zn.Set(x+dx, y+dy, TFloor)
				} else if d < 4.4 {
					zn.Set(x+dx, y+dy, TWall)
				}
			}
		}
		zn.Set(x, y+4, TDoor)
		zn.Set(x, y, TUp)
		zn.Notes[y*MW+x] = fmt.Sprintf("Stairs wind up into {{M|%s}} (%d floors). Press < to climb.", s.Name, s.Depth)
		return
	}
	if zn.Kind == "town" {
		zn.Notes[y*MW+x] = fmt.Sprintf("Steps lead down into {{M|%s}}. Press > to descend.", s.Name)
		return
	}
	for dy := -2; dy <= 2; dy++ {
		for dx := -4; dx <= 4; dx++ {
			edge := abs(dx) == 4 || abs(dy) == 2
			if edge && !(dy == 2 && dx == 0) {
				if (dx+dy)%3 == 0 {
					zn.Set(x+dx, y+dy, TRubble)
				} else {
					zn.Set(x+dx, y+dy, TWall)
				}
			} else {
				zn.Set(x+dx, y+dy, TFloor)
			}
		}
	}
	zn.Set(x-2, y-1, TInscription)
	zn.Notes[(y-1)*MW+x-2] = fmt.Sprintf("Carved over the way down: {{M|%s}}.", s.Name)
	zn.Notes[y*MW+x] = fmt.Sprintf("A way down into {{M|%s}}. Press > to descend.", s.Name)
}

var abyssLayers = []string{"Edge of the Abyss", "Forest of Whispers", "The Great Fault", "Goblet of Giants", "Sea of Corpses", "Capital of the Unreturned"}

type themeStyle struct {
	wall, floor Terrain
	foes        []string
	boss        string
	bossName    []string
}

var themes = map[string]themeStyle{
	"catacomb": {TWall, TFloor, []string{"skeleton", "skeleton", "ghoul", "rat", "spider"}, "wight", []string{"the Bone Abbot", "the Crowned Dead", "the Ossuary King"}},
	"sunken":   {TRock, TShallow, []string{"leech", "ooze", "eel", "crab", "croc"}, "kraken", []string{"the Drowned Thing", "Mother of Silt"}},
	"arcane":   {TCrystal, TCrystalFloor, []string{"golem", "turret", "wisp", "shardmoth"}, "crystalgolem", []string{"the Prism Warden", "the Singing Colossus"}},
	"chasm":    {TRock, TFloor, []string{"crawler", "troglodyte", "spider", "ooze"}, "horror", []string{"the Thing Below", "the Hollow Mouth"}},
	"barrow":   {TRock, TDirt, []string{"skeleton", "ghoul", "wisp", "rat"}, "wight", []string{"the Barrow King", "the Howe-Lord"}},
	"mine":     {TRock, TDirt, []string{"crawler", "troglodyte", "rat", "beetle", "ooze"}, "horror", []string{"What the Miners Woke", "the Deep Worm"}},
	"hideout":  {TWall, TFloor, []string{"bandit", "bandit", "archer"}, "chief", []string{"the Fence-King", "the Red Hand"}},
	"vault":    {TWall, TFloor, []string{"golem", "turret", "rat", "ooze"}, "golem", []string{"the Overseer Unit", "the Last Warden"}},
	"abyss":    {TRock, TFungus, []string{"crawler", "horror", "spider", "wisp", "ghoul"}, "horror", []string{"the Unreturned", "the Abyssal Sovereign"}},
	"magic":    {TWall, TCarpet, []string{"wisp", "golem", "turret", "shardmoth"}, "crystalgolem", []string{"the Mad Archmage's Familiar", "the Tower's Heart"}},
	"trials":   {TWall, TFloor, []string{"golem", "bandit", "wolf", "skeleton", "turret"}, "chief", []string{"the Floor Guardian", "the Trial-Keeper"}},
}

// genDungeon: rooms and corridors cut from solid rock, stairs linked to the levels above and below.
func (g *Game) genDungeon(zn *Zone, r *Rng, s *Site) {
	st := themes[s.Theme]
	zn.Kind, zn.Dark = "dungeon", true
	zn.Level += s.Bonus * zn.Z
	zn.Name = fmt.Sprintf("%s, level %d", s.Name, zn.Z)
	if s.Theme == "abyss" {
		zn.Name = fmt.Sprintf("%s: %s (%d)", s.Name, abyssLayers[clampi((zn.Z-1)/2, 0, 5)], zn.Z)
	}
	if s.Theme == "mine" || s.Theme == "hideout" {
		zn.WallF = CBrown
	}
	for i := range zn.T {
		zn.T[i] = st.wall
		if st.wall == TCrystal {
			zn.T[i] = TRock
		}
	}
	var rooms [][4]int
	carve := func(rc [4]int) {
		for y := rc[1]; y < rc[1]+rc[3]; y++ {
			for x := rc[0]; x < rc[0]+rc[2]; x++ {
				zn.Set(x, y, st.floor)
			}
		}
	}
	addRoom := func(cx, cy, w, h int) {
		rc := [4]int{clampi(cx-w/2, 1, MW-w-1), clampi(cy-h/2, 1, MH-h-1), w, h}
		carve(rc)
		rooms = append(rooms, rc)
	}
	// rooms around the stairs first
	for _, p := range g.stairsDown(zn.WX, zn.WY, zn.ZX, zn.ZY, zn.Z-1) {
		addRoom(p[0], p[1], r.Int(6, 10), r.Int(4, 6))
	}
	for _, p := range g.stairsDown(zn.WX, zn.WY, zn.ZX, zn.ZY, zn.Z) {
		addRoom(p[0], p[1], r.Int(6, 10), r.Int(4, 6))
	}
	for tries := 0; tries < 80 && len(rooms) < 11; tries++ {
		w, h := r.Int(5, 14), r.Int(3, 7)
		x, y := r.Int(1, MW-w-1), r.Int(1, MH-h-1)
		ok := true
		for _, o := range rooms {
			if x < o[0]+o[2]+1 && x+w+1 > o[0] && y < o[1]+o[3]+1 && y+h+1 > o[1] {
				ok = false
				break
			}
		}
		if ok {
			carve([4]int{x, y, w, h})
			rooms = append(rooms, [4]int{x, y, w, h})
		}
	}
	center := func(rc [4]int) (int, int) { return rc[0] + rc[2]/2, rc[1] + rc[3]/2 }
	order := append([][4]int{}, rooms...)
	sort.Slice(order, func(i, j int) bool { return order[i][0]+order[i][1] < order[j][0]+order[j][1] })
	for i := 1; i < len(order); i++ {
		ax, ay := center(order[i-1])
		bx, by := center(order[i])
		g.corridor(zn, ax, ay, bx, by, st.floor, r)
	}
	// extra loops so it isn't a single line
	for i := 0; i < 2 && len(order) > 3; i++ {
		a, b := order[r.Int(0, len(order)-1)], order[r.Int(0, len(order)-1)]
		ax, ay := center(a)
		bx, by := center(b)
		g.corridor(zn, ax, ay, bx, by, st.floor, r)
	}
	// doors where corridors meet room walls
	for _, rc := range rooms {
		for x := rc[0]; x < rc[0]+rc[2]; x++ {
			for _, y := range []int{rc[1] - 1, rc[1] + rc[3]} {
				if zn.At(x, y) == st.floor && zn.At(x-1, y) != st.floor && zn.At(x+1, y) != st.floor && r.Chance(0.4) && st.wall == TWall {
					zn.Set(x, y, TDoor)
				}
			}
		}
	}
	if st.wall == TCrystal {
		for i := range zn.T {
			if zn.T[i] == TRock && r.Chance(0.35) {
				zn.T[i] = TCrystal
			}
		}
	}
	// theme dressing
	for _, rc := range rooms {
		for k := 0; k < rc[2]*rc[3]/12; k++ {
			x, y := rc[0]+r.Int(0, rc[2]-1), rc[1]+r.Int(0, rc[3]-1)
			if zn.At(x, y) != st.floor {
				continue
			}
			switch s.Theme {
			case "catacomb", "barrow":
				zn.Set(x, y, Pick(r, []Terrain{TBones, TBones, TSarcophagus, TRubble}))
			case "mine":
				zn.Set(x, y, Pick(r, []Terrain{TRubble, TCrate}))
			case "hideout":
				zn.Set(x, y, Pick(r, []Terrain{TCrate, TBed, TBrazier}))
			case "vault":
				zn.Set(x, y, Pick(r, []Terrain{TMachine, TCrate}))
			case "abyss", "chasm":
				zn.Set(x, y, Pick(r, []Terrain{TMushroom, TBones, TChasm}))
			case "sunken":
				zn.Set(x, y, Pick(r, []Terrain{TDeep, TKelp, TCoral}))
			}
		}
	}
	if s.Theme == "mine" {
		for i := range zn.T {
			if zn.T[i] == TRock && r.Chance(0.04) {
				zn.T[i] = TOre
			}
		}
	}
	if s.Theme == "catacomb" || s.Theme == "barrow" {
		for _, rc := range rooms {
			if r.Chance(0.3) {
				x, y := rc[0]+1, rc[1]
				if zn.At(x, y) == st.floor {
					zn.Set(x, y, TInscription)
					zn.Notes[y*MW+x] = g.deadLine(zn, r)
				}
			}
		}
	}
	g.lockedVault(zn, r, rooms, st)
	g.layTraps(zn, r, 2+zn.Z)
	// inhabitants
	n := 3 + zn.Z + r.Int(0, 3)
	for i := 0; i < n; i++ {
		rc := rooms[r.Int(0, len(rooms)-1)]
		x, y := rc[0]+r.Int(0, rc[2]-1), rc[1]+r.Int(0, rc[3]-1)
		if !zn.Walkable(x, y) || zn.ActorAt(x, y) != nil {
			continue
		}
		a := g.Spawn(Pick(r, st.foes), zn.Level, r)
		if a.Faction == "outlaws" && s.Outlaw != nil {
			a.Desc = "One of " + s.Outlaw.Name + "."
		}
		a.X, a.Y = x, y
		zn.Actors = append(zn.Actors, a)
	}
	for i := 0; i < 2+zn.Z/2; i++ {
		rc := rooms[r.Int(0, len(rooms)-1)]
		x, y := rc[0]+r.Int(0, rc[2]-1), rc[1]+r.Int(0, rc[3]-1)
		if zn.Walkable(x, y) {
			it := g.Loot(r, zn.Level/3+1)
			it.X, it.Y = x, y
			zn.Items = append(zn.Items, it)
		}
	}
	if zn.Z >= s.Depth {
		g.placeBoss(zn, r, s, rooms, st)
	}
}

func (g *Game) deadLine(zn *Zone, r *Rng) string {
	w := g.W
	st := w.State[w.Idx(zn.WX, zn.WY)]
	var dead []*PersonData
	for i := range w.People {
		if !w.People[i].Alive && (w.People[i].State == st || st == 0) {
			dead = append(dead, &w.People[i])
		}
	}
	if len(dead) == 0 {
		return "A burial niche. The name is gone."
	}
	pp := dead[r.Int(0, len(dead)-1)]
	return fmt.Sprintf("A burial niche: {{Y|%s}}, %s, %d–%d%s.", pp.Name, nonEmpty(pp.Title, pp.Role), pp.Born, pp.Died, causeText(pp.Cause))
}

func (g *Game) corridor(zn *Zone, ax, ay, bx, by int, floor Terrain, r *Rng) {
	x, y := ax, ay
	horizFirst := r.Chance(0.5)
	for i := 0; i < 400 && (x != bx || y != by); i++ {
		if horizFirst && x != bx || !horizFirst && y == by {
			x += sign(bx - x)
		} else {
			y += sign(by - y)
		}
		t := zn.At(x, y)
		if !t.D().Walk || t == TChasm || t == TDeep {
			zn.Set(x, y, floor)
		}
	}
}

// lockedVault: a side room behind a locked door; its key lies elsewhere on the level.
func (g *Game) lockedVault(zn *Zone, r *Rng, rooms [][4]int, st themeStyle) {
	if len(rooms) < 4 || !r.Chance(0.5) {
		return
	}
	for tries := 0; tries < 40; tries++ {
		w, h := r.Int(4, 6), r.Int(3, 4)
		x, y := r.Int(2, MW-w-2), r.Int(2, MH-h-2)
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
		// find the nearest open tile below or above to hang the door on
		for dy := 1; dy < 6; dy++ {
			dx := x + w/2
			if zn.At(dx, y+h+dy).D().Walk {
				for yy := y; yy < y+h; yy++ {
					for xx := x; xx < x+w; xx++ {
						zn.Set(xx, yy, st.floor)
					}
				}
				for k := 0; k < dy; k++ {
					zn.Set(dx, y+h+k, st.floor)
				}
				zn.Set(dx, y+h, TLocked)
				keyID := int(HashStr(zn.Key) % 1000000)
				it := &Item{ID: g.newID(), Kind: "key", Name: "iron key", Glyph: '¬', F: CYellow, Extra: map[string]int{"key": keyID}, Value: 1}
				rc := rooms[r.Int(0, len(rooms)-1)]
				it.X, it.Y = rc[0]+rc[2]/2, rc[1]+rc[3]/2
				zn.Items = append(zn.Items, it)
				for k := 0; k < 3; k++ {
					loot := g.Loot(r, zn.Level/3+2)
					loot.X, loot.Y = x+r.Int(0, w-1), y+r.Int(0, h-1)
					zn.Items = append(zn.Items, loot)
				}
				gold := &Item{ID: g.newID(), Kind: "gold", Name: g.Sk.Currency, Glyph: '$', F: CYellow, Count: 30 * zn.Level, X: x, Y: y}
				zn.Items = append(zn.Items, gold)
				return
			}
		}
	}
}

// layTraps hides traps on floor tiles; they show as ^ once spotted.
func (g *Game) layTraps(zn *Zone, r *Rng, n int) {
	if zn.Traps == nil {
		zn.Traps = map[int]uint8{}
	}
	for i := 0; i < n*3 && len(zn.Traps) < n; i++ {
		x, y := r.Int(1, MW-2), r.Int(1, MH-2)
		t := zn.At(x, y)
		if t.D().Walk && t.D().Water == 0 && t != TDown && t != TUp && t != TDoor && t != TDoorOpen {
			zn.Traps[y*MW+x] = uint8(r.Int(1, 5))
		}
	}
}

func (g *Game) placeBoss(zn *Zone, r *Rng, s *Site, rooms [][4]int, st themeStyle) {
	big := rooms[0]
	for _, rc := range rooms {
		if rc[2]*rc[3] > big[2]*big[3] {
			big = rc
		}
	}
	boss := g.Spawn(st.boss, zn.Level+3, r)
	boss.Boss = true
	boss.Set("boss")
	boss.MaxHP = boss.MaxHP * 2
	boss.HP = boss.MaxHP
	boss.Name, boss.Proper = Pick(r, st.bossName), true
	if s.Outlaw != nil {
		boss.Name = "the chief of " + s.Outlaw.Name
	}
	boss.Desc = "The master of " + s.Name + "."
	boss.XP *= 3
	boss.X, boss.Y = big[0]+big[2]/2, big[1]+big[3]/2
	g.nudge(zn, boss)
	boss.Inv = append(boss.Inv, g.MakeOddity(r, 4), g.bossDrop(r, zn.Level))
	boss.Gold += 50*zn.Level + s.Gold
	zn.Actors = append(zn.Actors, boss)
	it := g.Loot(r, 5)
	it.X, it.Y = boss.X, boss.Y
	zn.Items = append(zn.Items, it)
	zn.Name += " — the deepest hall"
}

// genTowerFloor: a round chamber per floor, stairs offset so you cross each floor.
func (g *Game) genTowerFloor(zn *Zone, r *Rng, s *Site, floor int) {
	st := themes[s.Theme]
	zn.Kind, zn.Dark = "tower", true
	zn.Level = g.zoneLevel(zn.WX, zn.WY, 0) + 1 + floor*(1+s.Bonus)/2
	zn.Name = fmt.Sprintf("%s, floor %d", s.Name, floor)
	for i := range zn.T {
		zn.T[i] = TVoid
	}
	zn.Sky = true
	for y := 0; y < MH; y++ {
		for x := 0; x < MW; x++ {
			d := math.Hypot(float64(x-40)/2.2, float64(y-12))
			switch {
			case d < 10.5:
				zn.Set(x, y, st.floor)
			case d < 11.5:
				zn.Set(x, y, st.wall)
			}
		}
	}
	key := siteKey(zn.WX, zn.WY, zn.ZX, zn.ZY)
	ux, uy := g.towerStair(key, floor)
	dx, dy := g.towerStair(key, floor-1)
	if floor < s.Depth {
		zn.Set(ux, uy, TUp)
	}
	zn.Set(dx, dy, TDown)
	for i := 0; i < 6; i++ {
		x, y := r.Int(26, 54), r.Int(4, 20)
		if zn.At(x, y) == st.floor && r.Chance(0.5) {
			zn.Set(x, y, Pick(r, []Terrain{TPillar, TBrazier, TShelf, TStatue}))
		}
	}
	g.layTraps(zn, r, 1+floor/3)
	n := 2 + floor/2 + r.Int(0, 2)
	for i := 0; i < n; i++ {
		x, y, ok := zn.FreeSpot(r, func(x, y int) bool { return zn.At(x, y) == st.floor })
		if ok {
			a := g.Spawn(Pick(r, st.foes), zn.Level, r)
			a.X, a.Y = x, y
			zn.Actors = append(zn.Actors, a)
		}
	}
	if floor%5 == 0 || floor == s.Depth {
		g.placeBoss(zn, r, s, [][4]int{{30, 7, 20, 11}}, st)
		if floor != s.Depth {
			zn.Name = fmt.Sprintf("%s, floor %d — guardian's floor", s.Name, floor)
		}
	}
	if x, y, ok := zn.FreeSpot(r, nil); ok {
		it := g.Loot(r, zn.Level/3+1)
		it.X, it.Y = x, y
		zn.Items = append(zn.Items, it)
	}
}

// inRing pulls a point inside a tower chamber.
func (g *Game) inRing(x, y int) (int, int) {
	for math.Hypot(float64(x-40)/2.2, float64(y-12)) >= 9.5 {
		x += sign(40 - x)
		y += sign(12 - y)
	}
	return x, y
}

// ---- traps, keys, ore ----

func (g *Game) springTrap(x, y int) {
	z := g.Cur
	if z.Traps == nil {
		return
	}
	i := y*MW + x
	k, ok := z.Traps[i]
	if !ok {
		return
	}
	delete(z.Traps, i)
	if z.At(x, y) == TTrap {
		z.Set(x, y, TFloor)
	}
	if g.R.Int(1, 20)+g.P.M(AGI)+g.Skills["acrobatics"]*2 >= 17 {
		g.Msg("{{W|You leap clear of a trap!}}")
		return
	}
	p := g.P
	switch k {
	case 1:
		g.Msg("{{R|Spikes stab up from the floor!}}")
		g.hurt(p, g.R.Dice(2, 6), nil, "a spike trap")
		p.AddSt("bleeding", 4)
	case 2:
		g.Msg("{{G|A poisoned dart strikes you!}}")
		g.hurt(p, g.R.Dice(1, 4), nil, "a dart trap")
		p.AddSt("poisoned", 10)
	case 3:
		g.Msg("{{R|Flames roar from the walls!}}")
		g.explode(x, y, "fire", z.Level/3, nil)
	case 4:
		if z.Z > 0 {
			g.Msg("{{R|The floor gives way!}}")
			g.fall()
			return
		}
		g.Msg("{{R|A pit! You twist your ankle.}}")
		g.hurt(p, g.R.Dice(1, 6), nil, "a pit")
	case 5:
		g.Msg("{{O|A bell clangs somewhere. Something heard that.}}")
		for i := 0; i < 2; i++ {
			a := g.Spawn(pickCreature(g.R, z.Biome, 'd', z.Level), z.Level, g.R)
			a.X, a.Y = x+g.R.Int(-6, 6), y+g.R.Int(-3, 3)
			g.nudge(z, a)
			z.Actors = append(z.Actors, a)
		}
	}
}

// noticeTraps: each step, a chance to spot nearby hidden traps (INT, Lore, Survival); search = big bonus.
func (g *Game) noticeTraps(bonus int) {
	z := g.Cur
	if len(z.Traps) == 0 {
		return
	}
	chance := 0.06 + 0.04*float64(g.P.M(INT)+g.Skills["lore"]+g.Skills["survival"]) + 0.1*float64(bonus)
	for i := range z.Traps {
		x, y := i%MW, i/MW
		if z.At(x, y) == TTrap || abs(x-g.P.X) > 3 || abs(y-g.P.Y) > 2 {
			continue
		}
		if g.R.Chance(chance) {
			z.Set(x, y, TTrap)
			g.Msg("{{M|You spot a trap.}}")
		}
	}
}

func (g *Game) unlock(x, y int) {
	keyID := int(HashStr(g.Cur.Key) % 1000000)
	for _, it := range g.P.Inv {
		if it.Kind == "key" && it.Extra != nil && it.Extra["key"] == keyID {
			g.removeInv(it)
			g.Cur.Set(x, y, TDoorOpen)
			g.Msg("{{Y|The key turns. The door swings open.}}")
			g.spend(1000)
			return
		}
	}
	if g.Skills["tinkering"] >= 2 && g.R.Int(1, 20)+g.P.M(AGI)+g.Skills["tinkering"]*2 >= 18 {
		g.Cur.Set(x, y, TDoorOpen)
		g.Msg("{{Y|You pick the lock.}}")
	} else {
		g.Msg("It's locked. The key must be somewhere on this level.")
	}
	g.spend(1000)
}

func (g *Game) mineOre(x, y int) {
	g.Cur.Set(x, y, TRubble)
	n := 2 + g.R.Int(0, 3)
	g.give(g.MakeScrap(g.R, n))
	g.Msg(fmt.Sprintf("You hack out ore and gain %d %s.", n, g.Sk.Scrap))
	if g.R.Chance(0.08) {
		g.give(&Item{ID: g.newID(), Kind: "trophy", Name: "raw gemstone", Glyph: '*', F: CMagenta, Value: 80})
		g.Msg("{{M|A gemstone glints in the rubble!}}")
	}
	g.spend(3000)
}

// parasangSite: the first site in any of a parasang's surface zones.
func (g *Game) parasangSite(p int) *Site {
	wx, wy := p%g.W.W, p/g.W.W
	for zy := 0; zy < 3; zy++ {
		for zx := 0; zx < 3; zx++ {
			if s := g.siteAt(wx, wy, zx, zy); s != nil {
				return s
			}
		}
	}
	return nil
}

// towerStair: where floor f's way up sits (f = 0 is the stair on the surface). Floor f's way
// down is towerStair(f-1), so consecutive floors always agree; the two stairs of one floor are
// kept apart so every floor has to be crossed.
func (g *Game) towerStair(key string, f int) (int, int) {
	x, y := g.inRing(posFor(fmt.Sprintf("%s:floor%d", key, f)))
	if f == 0 {
		return x, y
	}
	dx, dy := g.towerStair(key, f-1)
	if abs(x-dx) < 6 && abs(y-dy) < 3 {
		x, y = g.inRing(80-dx, 24-dy)
		if abs(x-dx) < 6 && abs(y-dy) < 3 {
			x, y = g.inRing(dx+12*sign(40-dx+1), dy)
		}
	}
	return x, y
}
