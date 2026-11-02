package engine

import (
	"fmt"
	"math"
	"strings"
)

// Plant species on map tiles (prefix fl). Zone.T is untouched: a tile's species is a pure function of
// (world seed, zone, tile, its terrain, the parasang's biome/temperature/height, genre), so it is cached
// per zone at run time (`flZone.idx`, rebuilt after load or when a tile's terrain changes) and only what
// the player changed is saved: felled trees, harvests and species forced by other systems.
// Rendering, look text, felling with an axe (bump), and gathering fruit (bump a tree, `g` on a plant).

func init() {
	zoneHooks = append(zoneHooks, flZoneHook)
}

const flUnset = 0xFFFF

// flZone is the saved per-zone flora state (Zone.Flora).
type flZone struct {
	Felled map[int]bool   `json:"f,omitempty"` // tile → stump
	Picked map[int]int    `json:"p,omitempty"` // tile → year+1 of the last harvest
	Forced map[int]uint16 `json:"s,omitempty"` // tile → species id+1 set by another system

	idx   []uint16 // runtime cache: species id+1 per tile, 0 none, flUnset not computed
	tt    []Terrain
	cands map[uint8][]uint16
	key   string
}

var flByName = func() map[string]uint16 {
	m := map[string]uint16{}
	for i, s := range flCat {
		m[strings.ToLower(s.Name)] = uint16(i + 1)
	}
	return m
}()

func flState(zn *Zone) *flZone {
	if zn.Flora == nil {
		zn.Flora = &flZone{}
	}
	return zn.Flora
}

func flLayerOf(zn *Zone) uint8 {
	switch {
	case zn.Underwater:
		return flSea
	case zn.Sky:
		return flSky
	case zn.Z > 0:
		return flDeep
	}
	return flSurf
}

func flForest(b int) bool { return b >= 5 && b <= 9 }

// flClimate: the parasang's base temperature and elevation.
func (g *Game) flClimate(zn *Zone) (int, int) {
	w := g.W
	p := w.Idx(zn.WX, zn.WY)
	t := biomeTemp[w.biomeAt(p)]
	if p >= 0 && p < len(w.Temp) {
		t = w.Temp[p]
	}
	h := 30
	if p >= 0 && p < len(w.Height) {
		h = w.Height[p]
	}
	return t, h
}

// flClassAt is the plant slot of a tile, or flNone. h is the tile hash (sparse classes).
func flClassAt(zn *Zone, t Terrain, h uint64) uint8 {
	r := h % 100
	switch t {
	case TTree:
		return flTree
	case TDeadTree:
		return flSnag
	case TBush:
		return flShrub
	case TTallGrass:
		if r < 12 {
			return flFlower
		}
		return flGrass
	case TGrass:
		switch {
		case r < 5:
			return flFlower
		case r < 8:
			return flGrass
		case r < 10 && flForest(zn.Biome) && zn.Z == 0:
			return flMat
		}
	case TMoss:
		if r < 5 && flForest(zn.Biome) && zn.Z == 0 {
			return flMat
		}
		return flMoss
	case TReeds:
		return flReed
	case TMushroom:
		return flGiant
	case TFungus:
		if r < 35 {
			return flMat
		}
	case TKelp:
		return flKelp
	case TShallow:
		if zn.Underwater || zn.Z != 0 {
			return flNone
		}
		if zn.Kind == "sea" || zn.Biome == 0 {
			if r < 6 {
				return flSeagrass
			}
		} else if r < 5 {
			return flLily
		}
	case TCrop:
		return flCrop
	case TCrystal:
		return flCrystal
	}
	return flNone
}

func flGenreOK(sp *flSp, genre string) bool {
	if len(sp.Gen) == 0 {
		return true
	}
	for _, x := range sp.Gen {
		if x == genre {
			return true
		}
	}
	return false
}

func flBioOK(sp *flSp, b int) bool {
	if sp.Bio == nil {
		return true
	}
	for _, x := range sp.Bio {
		if x == b {
			return true
		}
	}
	return false
}

// flCandidates lists the species (id+1) that fit a class in this zone, relaxing the climate,
// then the biome, then the layer when nothing fits.
func (g *Game) flCandidates(zn *Zone, cl uint8) []uint16 {
	st := flState(zn)
	if st.cands == nil || st.key != g.Genre {
		st.cands, st.key = map[uint8][]uint16{}, g.Genre
	}
	if c, ok := st.cands[cl]; ok {
		return c
	}
	lay := flLayerOf(zn)
	temp, ht := g.flClimate(zn)
	exotic := func(sp *flSp) bool { // only grows in an exotic biome (ashland, crystal, fungal, lava)
		if sp.Bio == nil {
			return false
		}
		for _, b := range sp.Bio {
			if b < 13 {
				return false
			}
		}
		return true
	}
	fits := func(sp *flSp) bool { return temp >= sp.TLo && temp <= sp.THi && ht >= sp.HLo && ht <= sp.HHi }
	try := func(stage int) []uint16 {
		var out []uint16
		for i := range flCat {
			sp := &flCat[i]
			if sp.Cl != cl || !flGenreOK(sp, g.Genre) {
				continue
			}
			ok := false
			switch stage {
			case 1: // strict
				ok = sp.Lay == lay && flBioOK(sp, zn.Biome) && fits(sp)
			case 2: // biome, any climate
				ok = sp.Lay == lay && flBioOK(sp, zn.Biome)
			case 3: // a neighbour's plant bled in: same layer, climate fits
				ok = sp.Lay == lay && !exotic(sp) && fits(sp)
			case 4:
				ok = sp.Lay == lay && !exotic(sp)
			case 5: // any layer but the sky's
				ok = (sp.Lay != flSky || lay == flSky) && !exotic(sp)
			default:
				ok = true
			}
			if ok {
				out = append(out, uint16(i+1))
			}
		}
		return out
	}
	var c []uint16
	for stage := 1; stage <= 6 && len(c) == 0; stage++ {
		c = try(stage)
	}
	c = flStandPool(c, cl, HashStr(fmt.Sprint(g.W.Seed, ":flpool:", zn.WX, ".", zn.WY, ".", zn.Z, ":", cl)))
	st.cands[cl] = c
	return c
}

// flStandPool keeps a handful of species per parasang and class, so a forest has a character
// (a beech-oak wood, a spruce-birch taiga) and the next parasang a different mix.
func flStandPool(c []uint16, cl uint8, h uint64) []uint16 {
	max := 4
	switch cl {
	case flTree, flFlower:
		max = 6
	case flShrub:
		max = 5
	}
	if len(c) <= max {
		return c
	}
	rest := append([]uint16(nil), c...)
	var out []uint16
	for len(out) < max && len(rest) > 0 {
		h = wxMix(h + 0x9e3779b97f4a7c15)
		id := flWeighted(rest, h)
		out = append(out, id)
		for k, v := range rest {
			if v == id {
				rest = append(rest[:k], rest[k+1:]...)
				break
			}
		}
	}
	// keep catalogue order so results do not depend on draw order
	var ord []uint16
	for _, v := range c {
		for _, o := range out {
			if o == v {
				ord = append(ord, v)
			}
		}
	}
	return ord
}

func flWeighted(c []uint16, h uint64) uint16 {
	tot := 0
	for _, id := range c {
		tot += flCat[id-1].Wt
	}
	if tot <= 0 {
		return 0
	}
	v := int(h % uint64(tot))
	for _, id := range c {
		v -= flCat[id-1].Wt
		if v < 0 {
			return id
		}
	}
	return c[len(c)-1]
}

func (g *Game) flSeed() uint64 { return HashStr(g.W.Seed + ":fl") }

func flTileHash(seed uint64, zn *Zone, i int) uint64 {
	return wxMix(seed ^ HashStr(zn.Key)*0x9e3779b97f4a7c15 ^ uint64(i+1)*0xc2b2ae3d27d4eb4f)
}

// flPick is the species (id+1) of a tile: mixed stands from patch dominants in global coordinates
// (so stands run across zone edges), a secondary from the neighbouring patch, and a few strays.
func (g *Game) flPick(zn *Zone, x, y int, t Terrain) uint16 {
	i := y*MW + x
	seed := g.flSeed()
	h := flTileHash(seed, zn, i)
	cl := flClassAt(zn, t, h)
	if cl == flNone {
		return 0
	}
	c := g.flCandidates(zn, cl)
	if len(c) == 0 {
		return 0
	}
	gx, gy := gxy(zn, x, y)
	ns := seed ^ uint64(zn.Z+7)*0x51ed27
	px := int(math.Floor((gx + 4*noise2(ns, gx/6, gy/3)) / 7))
	py := int(math.Floor((gy + 2*noise2(ns+1, gx/6, gy/3)) / 3))
	patch := func(px, py int) uint64 {
		return wxMix(seed ^ uint64(px)*0x9e3779b97f4a7c15 ^ uint64(py)*0xc2b2ae3d27d4eb4f ^ uint64(cl)*0x165667b19e3779f9 ^ HashStr(fmt.Sprint(zn.WX, zn.WY, zn.Z)))
	}
	roll := (h >> 20) % 100
	switch {
	case roll < 60:
		return flWeighted(c, patch(px, py))
	case roll < 88:
		return flWeighted(c, patch(px+1, py+(int(h>>40)%3)-1))
	}
	return flWeighted(c, h>>8)
}

// flID returns the cached species (id+1) of a tile, recomputing it when the terrain changed.
func (g *Game) flID(zn *Zone, i int) uint16 {
	if zn == nil || i < 0 || i >= len(zn.T) {
		return 0
	}
	st := flState(zn)
	if len(st.idx) != len(zn.T) {
		st.idx = make([]uint16, len(zn.T))
		st.tt = make([]Terrain, len(zn.T))
		for k := range st.idx {
			st.idx[k] = flUnset
		}
	}
	t := zn.T[i]
	if st.idx[i] != flUnset && st.tt[i] == t {
		return st.idx[i]
	}
	var id uint16
	if f, ok := st.Forced[i]; ok && int(f) <= len(flCat) && f > 0 && flSlotFits(flCat[f-1].Cl, t) {
		id = f
	} else {
		id = g.flPick(zn, i%MW, i/MW, t)
	}
	st.idx[i], st.tt[i] = id, t
	return id
}

// flSlotFits: can a species of class cl stand on terrain t?
func flSlotFits(cl uint8, t Terrain) bool {
	switch cl {
	case flTree:
		return t == TTree
	case flSnag:
		return t == TDeadTree
	case flShrub:
		return t == TBush
	case flGrass, flFlower:
		return t == TTallGrass || t == TGrass
	case flReed:
		return t == TReeds
	case flMoss:
		return t == TMoss
	case flGiant:
		return t == TMushroom
	case flMat:
		return t == TFungus || t == TGrass || t == TMoss
	case flKelp:
		return t == TKelp
	case flLily, flSeagrass:
		return t == TShallow
	case flCrop:
		return t == TCrop
	case flCrystal:
		return t == TCrystal
	}
	return false
}

// flSpeciesFor is the species on a tile (nil if none). Other systems may use it.
func flSpeciesFor(g *Game, zn *Zone, x, y int) *flSp {
	if zn == nil || !zn.In(x, y) {
		return nil
	}
	if id := g.flID(zn, y*MW+x); id > 0 {
		return &flCat[id-1]
	}
	return nil
}

// flSetSpecies forces a named species onto a tile (e.g. oasis palms); it is saved with the zone.
func flSetSpecies(g *Game, zn *Zone, x, y int, name string) bool {
	id, ok := flByName[strings.ToLower(name)]
	if !ok || zn == nil || !zn.In(x, y) {
		return false
	}
	st := flState(zn)
	if st.Forced == nil {
		st.Forced = map[int]uint16{}
	}
	i := y*MW + x
	st.Forced[i] = id
	if len(st.idx) == len(zn.T) {
		st.idx[i] = flUnset
	}
	return true
}

func flSeasonBit(season int) uint8 { return 1 << uint(season&3) }

// flSkinOf picks a genre reskin for a tile (nil for most).
func flSkinOf(genre string, sp *flSp, h uint64) *flSkin {
	sk := flGenreSkins[genre]
	if len(sk) == 0 || len(sp.Gen) > 0 || sp.Lay == flSea || sp.Cl == flCrystal {
		return nil
	}
	r := int((h >> 33) % 100)
	for k := range sk {
		if r < sk[k].Chance {
			return &sk[k]
		}
		r -= sk[k].Chance
	}
	return nil
}

func (g *Game) flWarm(zn *Zone) bool {
	t, _ := g.flClimate(zn)
	return t >= 18
}

// flTile is the drawMap hook: species glyph and colour, by season and genre.
func (g *Game) flTile(zn *Zone, i int, t Terrain, gl rune, f byte) (rune, byte) {
	if zn.Flora != nil && zn.Flora.Felled[i] && t.D().Walk {
		return 'ⁿ', CBrown
	}
	id := g.flID(zn, i)
	if id == 0 {
		return gl, f
	}
	sp := &flCat[id-1]
	season := g.Season()
	bit := flSeasonBit(season)
	ng, nf := sp.G, sp.F
	switch sp.Cl {
	case flFlower, flMat:
		if sp.Bloom != 0 && sp.Bloom&bit == 0 {
			return gl, f // not in flower / not fruiting: plain ground
		}
		if sp.Cl == flFlower {
			nf = sp.Blo
		} else if sp.Blo != 0 {
			nf = sp.Blo
		}
	case flLily:
		if sp.Bloom&bit != 0 {
			nf = sp.Blo
		}
	default:
		switch {
		case sp.Decid && season == 3:
			nf = CBrown
			if sp.Cl == flTree {
				ng = 'τ'
			}
		case sp.Bloom&bit != 0 && sp.Blo != 0:
			nf = sp.Blo
		case season == 2 && sp.Aut != 0:
			nf = sp.Aut
		case season == 3 && (sp.Cl == flGrass || sp.Cl == flReed) && !g.flWarm(zn):
			nf = CBrown
		}
	}
	if sk := flSkinOf(g.Genre, sp, flTileHash(g.flSeed(), zn, i)); sk != nil && sk.F != 0 {
		nf = sk.F
	}
	return ng, nf
}

var flColName = map[byte]string{CRed: "scarlet", CDarkRed: "crimson", COrange: "orange", CYellow: "gold", CBrown: "russet", CMagenta: "pink", CWhite: "white", CBlue: "blue"}

func flAn(s string) string {
	if s == "" {
		return s
	}
	switch strings.ToLower(s[:1]) {
	case "a", "e", "i", "o", "u":
		return "an " + s
	}
	return "a " + s
}

// flCount: "an old sessile oak", "a stand of feather grass", "a field of wheat".
func flCount(sp *flSp, s string) string {
	of := map[uint8]string{flGrass: "a stand of ", flMoss: "a carpet of ", flFlower: "a clump of ", flReed: "a bed of ",
		flKelp: "a stand of ", flSeagrass: "a meadow of ", flCrop: "a field of ", flLily: "a raft of "}[sp.Cl]
	if sp.Cl == flMat && sp.G == flGMoss {
		of = "a patch of "
	}
	if of != "" {
		return of + s
	}
	return flAn(s)
}

func flCap(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func flAge(sp *flSp, h uint64) string {
	r := (h >> 50) % 10
	switch sp.Cl {
	case flTree:
		if sp.Size >= 3 {
			return []string{"towering", "colossal", "vast", "ancient", "towering", "great", "colossal", "vast", "ancient", "great"}[r]
		}
		return []string{"young", "mature", "old", "", "", "", "tall", "ancient", "gnarled", "slender"}[r]
	case flShrub:
		return []string{"dense", "straggling", "", "", "low", "", "thick", "", "wind-cut", ""}[r]
	case flGiant:
		return []string{"towering", "", "ancient", "", "swollen", "", "", "pale", "", ""}[r]
	case flSnag:
		return []string{"old", "", "", "weathered", "", "tall", "", "", "", ""}[r]
	}
	return ""
}

func flSeasonText(g *Game, zn *Zone, sp *flSp) string {
	season := g.Season()
	bit := flSeasonBit(season)
	switch sp.Cl {
	case flFlower:
		if sp.Bloom&bit == 0 {
			return "It is not in flower now."
		}
		return "It is in flower."
	case flMat:
		if sp.Bloom != 0 && sp.Bloom&bit == 0 {
			return "Only a trace of it shows out of season."
		}
		return ""
	}
	switch {
	case sp.Decid && season == 3:
		return "Bare for the winter."
	case sp.Bloom&bit != 0 && sp.Blo != 0:
		return "It is in " + flColName[sp.Blo] + " blossom."
	case season == 2 && sp.Aut != 0:
		return "Autumn has turned it " + flColName[sp.Aut] + "."
	case season == 0 && sp.Decid:
		return "New leaves are unfurling."
	case season == 3 && (sp.Cl == flGrass || sp.Cl == flReed) && !g.flWarm(zn):
		return "Dry and dormant in the cold."
	}
	return ""
}

func flSeasonsText(m uint8) string {
	if m == sAll {
		return "all year"
	}
	var p []string
	for s := 0; s < 4; s++ {
		if m&flSeasonBit(s) != 0 {
			p = append(p, strings.ToLower(SeasonNames[s]))
		}
	}
	return strings.Join(p, ", ")
}

// flDisplayName: the species name with its genre reskin, if any.
func (g *Game) flDisplayName(zn *Zone, i int, sp *flSp) string {
	if sk := flSkinOf(g.Genre, sp, flTileHash(g.flSeed(), zn, i)); sk != nil {
		return sk.Adj + " " + sp.Name
	}
	return sp.Name
}

// flLook is the look-mode line for a plant tile ("" when it has no species).
func (g *Game) flLook(zn *Zone, x, y int) string {
	i := y*MW + x
	if zn.Flora != nil && zn.Flora.Felled[i] && zn.T[i].D().Walk {
		if id := zn.Flora.idxOld(i); id > 0 {
			return "{{y|stump}}. The stump of a felled " + flCat[id-1].Name + "."
		}
		return "{{y|stump}}. The stump of a felled tree."
	}
	id := g.flID(zn, i)
	if id == 0 {
		return ""
	}
	sp := &flCat[id-1]
	h := flTileHash(g.flSeed(), zn, i)
	name := g.flDisplayName(zn, i, sp)
	adj := flAge(sp, h)
	full := name
	if adj != "" {
		full = adj + " " + name
	}
	head := fmt.Sprintf("{{y|%s}} {{K|(%s)}}. %s, %s.", name, flClassName[sp.Cl], flCap(flCount(sp, full)), sp.Desc)
	if sp.Cl == flSnag && sp.G == flGSnag {
		head = fmt.Sprintf("{{y|%s}} {{K|(dead tree)}}. %s, %s.", name, flCap(flAn(full)), sp.Desc)
	}
	parts := []string{head}
	if s := flSeasonText(g, zn, sp); s != "" {
		parts = append(parts, s)
	}
	if sk := flSkinOf(g.Genre, sp, h); sk != nil {
		parts = append(parts, sk.Note)
	}
	var info []string
	if sp.Wood != "" {
		info = append(info, "wood: "+flWoodName(sp))
	}
	if sp.Prod != "" {
		info = append(info, sp.Prod+" ("+flSeasonsText(sp.PSeas)+")")
	}
	if len(info) > 0 {
		parts = append(parts, "{{K|"+flCap(strings.Join(info, "; "))+".}}")
	}
	return strings.Join(parts, " ")
}

// idxOld recovers the species a felled tile had before its terrain changed (kept in the cache).
func (st *flZone) idxOld(i int) uint16 {
	if st.Forced != nil {
		if f, ok := st.Forced[i]; ok {
			return f
		}
	}
	return 0
}

func flWoodName(sp *flSp) string {
	w := sp.Wood
	switch {
	case strings.HasSuffix(w, "ribs"), strings.HasSuffix(w, "shards"), w == "charcoal",
		strings.HasSuffix(w, "wood"), strings.HasSuffix(w, "trunk"), strings.HasSuffix(w, "canes"):
		return w
	case w == "bamboo":
		return "bamboo canes"
	case sp.Cl == flGiant:
		return w + " stalk"
	}
	return w + " logs"
}

func (g *Game) flAxe() *Item {
	for _, w := range g.P.Weapons() {
		if w.Skill == "axe" {
			return w
		}
	}
	return nil
}

func (g *Game) flStumpGround(zn *Zone) Terrain {
	if zn.Z != 0 || zn.Sky {
		return g.floorFor(zn)
	}
	switch zn.Biome {
	case 13, 16:
		return TAsh
	case 14:
		return TCrystalFloor
	case 15:
		return TFungus
	case 9, 10, 11:
		return TSnow
	case 1:
		return TSand
	}
	return TDirt
}

// flBump runs when the player walks into a blocking plant: fell it with an axe, else gather its
// fruit in season. False leaves the bump to the old code (climbing, nothing).
func (g *Game) flBump(x, y int) bool {
	zn := g.Cur
	t := zn.At(x, y)
	if t != TTree && t != TDeadTree && t != TMushroom && t != TCrystal {
		return false
	}
	sp := flSpeciesFor(g, zn, x, y)
	if sp == nil {
		return false
	}
	if ax := g.flAxe(); ax != nil && sp.Wood != "" {
		g.flFell(x, y, sp, ax)
		return true
	}
	if g.flHarvest(zn, x, y, sp, false) {
		return true
	}
	if !g.canClimb(t) {
		name := g.flDisplayName(zn, y*MW+x, sp)
		if sp.Wood != "" {
			g.Msg(flCap(flAn(name)) + " blocks the way. {{K|(Wield an axe to fell it.)}}")
		} else {
			g.Msg(flCap(flAn(name)) + " blocks the way.")
		}
	}
	return false
}

// flFell chops a tree down: several turns, named wood, a stump.
func (g *Game) flFell(x, y int, sp *flSp, ax *Item) {
	zn := g.Cur
	i := y*MW + x
	need := maxi(2, 3+int(sp.Size)*2-ax.Tier/2)
	for k := 0; k < need; k++ {
		if g.nearHostile() {
			g.Msg("{{R|You stop chopping: danger is near.}}")
			return
		}
		g.spend(1000)
		if g.Dead {
			return
		}
	}
	n := 1 + int(sp.Size)
	if g.Skills["survival"] > 0 {
		n++
	}
	name := g.flDisplayName(zn, i, sp)
	wn := flWoodName(sp)
	burn := "quickly"
	switch {
	case sp.WoodV >= 14:
		burn = "hot and long"
	case sp.WoodV >= 8:
		burn = "steadily"
	}
	gl, col := '=', byte(CBrown)
	if sp.Cl == flCrystal {
		gl, col = '*', sp.F
	}
	it := &Item{ID: g.newID(), Kind: "scrap", Name: wn, Glyph: gl, F: col, Count: n, Value: maxi(1, sp.WoodV), Weight: 3,
		Lore: fmt.Sprintf("Cut from %s. It burns %s; a tinker can work it.", flAn(sp.Name), burn)}
	st := flState(zn)
	if st.Felled == nil {
		st.Felled = map[int]bool{}
	}
	if st.Forced == nil {
		st.Forced = map[int]uint16{}
	}
	st.Forced[i] = flByName[strings.ToLower(sp.Name)] // remembered for the stump's look text
	st.Felled[i] = true
	zn.Set(x, y, g.flStumpGround(zn))
	g.give(it)
	verb := "fell"
	if sp.Cl == flCrystal {
		verb = "break down"
	} else if sp.Cl == flGiant {
		verb = "hew down"
	}
	g.Msg(fmt.Sprintf("You %s the %s and take %d %s.", verb, name, n, wn))
	g.recompute()
}

// flHarvest gathers a plant's product when in season, once a year per tile.
func (g *Game) flHarvest(zn *Zone, x, y int, sp *flSp, loud bool) bool {
	if sp.Prod == "" {
		if loud {
			g.Msg("There is nothing to gather from the " + sp.Name + ".")
		}
		return loud
	}
	i := y*MW + x
	if sp.PSeas&flSeasonBit(g.Season()) == 0 {
		if loud {
			g.Msg(fmt.Sprintf("The %s has no %s this season ({{K|%s}}).", sp.Name, sp.Prod, flSeasonsText(sp.PSeas)))
		}
		return loud
	}
	yr := g.Day()/YearDays + 1
	st := flState(zn)
	if st.Picked[i] == yr {
		if loud {
			g.Msg("This " + sp.Name + " has already been gathered this year.")
		}
		return loud
	}
	if st.Picked == nil {
		st.Picked = map[int]int{}
	}
	st.Picked[i] = yr
	n := maxi(1, sp.PN)
	if g.Skills["survival"] > 0 {
		n++
	}
	col := sp.F
	if sp.Blo != 0 {
		col = sp.Blo
	}
	it := g.makeIngredient(sp.Prod, sp.PTag, n, col)
	it.Lore = fmt.Sprintf("Gathered from %s.", flAn(sp.Name))
	g.give(it)
	g.Msg(fmt.Sprintf("You gather %d %s from the %s.", n, sp.Prod, sp.Name))
	g.spend(1000)
	return true
}

// flGather is the pickup hook: on a bare tile, `g` gathers from the plant underfoot.
func (g *Game) flGather() bool {
	zn := g.Cur
	sp := flSpeciesFor(g, zn, g.P.X, g.P.Y)
	if sp == nil {
		return false
	}
	return g.flHarvest(zn, g.P.X, g.P.Y, sp, true)
}

// flZoneHook fills the cache and, in wild forests, opens clearings and adds undergrowth
// (side rng seed:fl:zoneKey, walkable tiles only ever stay walkable).
func flZoneHook(g *Game, zn *Zone, _ *Rng) {
	if zn == nil {
		return
	}
	if zn.Kind == "surface" && zn.Z == 0 && !zn.Sky && flForest(zn.Biome) {
		g.flForestDetail(zn)
	}
	for i := range zn.T {
		g.flID(zn, i)
	}
}

func (g *Game) flForestDetail(zn *Zone) {
	r := NewRng(g.W.Seed + ":fl:" + zn.Key)
	floor := TGrass
	if zn.Biome == 7 || zn.Biome == 8 {
		floor = TMoss
	} else if zn.Biome == 9 {
		floor = TMoss
	}
	for n := r.Int(0, 2); n > 0; n-- {
		cx, cy := r.Int(10, MW-11), r.Int(5, MH-6)
		rx, ry := float64(r.Int(3, 7)), float64(r.Int(2, 3))
		for y := cy - 4; y <= cy+4; y++ {
			for x := cx - 8; x <= cx+8; x++ {
				if !zn.In(x, y) {
					continue
				}
				dx, dy := float64(x-cx)/rx, float64(y-cy)/ry
				d := dx*dx + dy*dy
				if d > 1 || zn.At(x, y) != TTree {
					continue
				}
				switch {
				case d < 0.6 && r.Chance(0.9):
					zn.Set(x, y, floor)
				case d >= 0.6 && r.Chance(0.5):
					zn.Set(x, y, Pick(r, []Terrain{TTallGrass, TBush, floor}))
				}
			}
		}
	}
	for y := 1; y < MH-1; y++ {
		for x := 1; x < MW-1; x++ {
			t := zn.At(x, y)
			if t != TGrass && t != TMoss && t != TSnow {
				continue
			}
			trees := 0
			for _, d := range dirs8 {
				if zn.At(x+d[0], y+d[1]) == TTree {
					trees++
				}
			}
			if trees < 2 {
				continue
			}
			v := r.Float()
			switch {
			case v < 0.08:
				zn.Set(x, y, TBush)
			case v < 0.14 && t != TSnow:
				zn.Set(x, y, TTallGrass)
			}
		}
	}
}
