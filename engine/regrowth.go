package engine

import (
	"fmt"
	"strings"
)

// Ruins that grow back (prefix rg). The world changes over the decades of game time (the player ages, see
// aging.go): every town has a FATE, a pure function of (seed, years since the start, what the wanderer did
// for that town) with five stages:
//
//	thriving     the town as generated
//	rebuilt      new timber, scaffolding and fresh floors on the old plan
//	abandoned    the people have left; squatters, camps and hermits move in
//	overgrown    a ruin under moss, vines, saplings and fallen walls
//	resettled    the ruin is lived in again: tents, scaffolds, settlers
//
// Seed only: a share of living towns (small ones, frontier ones, bandit towns) are doomed at some year
// 20..120 after the start (abandoned for 10..25 years, overgrown for 35..85, then often resettled);
// capitals never are. Fallen towns (!Alive) are overgrown at the start and some are resettled later
// (20..120 years in). The player's deeds, recorded for the town in g.ext("rg") by the hourly hook:
// SAVED (the town board in townvoice.go counts enough good deeds there) and RULED (the player holds
// office in the town's realm, politics.go) keep a town thriving for good, with a plaque; BURNED (a large
// blaze in a town zone, or more than six townsfolk killed, or enough bad deeds) empties it at once, and it
// is then rebuilt after 3..8 years (65%, more if it was ever saved) or left to the weeds.
//
// Applied by a zone hook and an enter hook (a zone already generated is repainted when its stage has moved
// on): side rng seed:rg:<zoneKey>:<stage>, painted with the tgPainter safe pattern (tiles away from stairs,
// actors and items; walls only ever fall open, trees keep six open neighbours), then connect + computeLight.
// One inscription tells what happened and when; zone names carry a suffix; squatters are placed; in
// abandoned and overgrown living towns most plain townsfolk are gone. State g.ext("rg"), planet-local.
// Hooks outside this file (see report): rgWorldGlyph and rgInfo in render.go drawWorld.

type rgStage int

const (
	rgThriving rgStage = iota
	rgRebuilt
	rgAbandoned
	rgOvergrown
	rgResettled
)

var rgStageName = [...]string{"thriving", "rebuilt", "abandoned", "overgrown ruin", "resettled"}

func init() {
	unLocalKeys = append(unLocalKeys, "rg")
	zoneHooks = append(zoneHooks, rgZone)
	enterHooks = append(enterHooks, rgEnter)
	turnHooks = append(turnHooks, rgTurn)
	sqFallSink = func(g *Game, burg, to int) { // a besieged town fell (sieges.go)
		s := g.rg()
		s.Burned[burg] = g.Day()
		s.Why[burg] = "blood"
	}
}

type rgState struct {
	Saved   map[int]int    `json:"sv"` // burg -> day the wanderer was credited with saving it
	Burned  map[int]int    `json:"bn"` // burg -> day it was burned or bled
	Why     map[int]string `json:"wy"` // burg -> "fire" or "blood"
	Ruled   map[int]int    `json:"rl"` // burg -> day the wanderer first ruled it
	Applied map[string]int `json:"ap"` // zone key -> stage+1 painted (6: plaque only)
	Base    map[string]string `json:"bs"` // zone key -> the name it was generated with
}

func (g *Game) rg() *rgState {
	s := g.ext("rg", func() any { return &rgState{} }).(*rgState)
	if s.Saved == nil {
		s.Saved = map[int]int{}
	}
	if s.Burned == nil {
		s.Burned = map[int]int{}
	}
	if s.Why == nil {
		s.Why = map[int]string{}
	}
	if s.Ruled == nil {
		s.Ruled = map[int]int{}
	}
	if s.Applied == nil {
		s.Applied = map[string]int{}
	}
	if s.Base == nil {
		s.Base = map[string]string{}
	}
	return s
}

// ---------------------------------------------------------------- the fate function

type rgFate struct {
	Stage rgStage
	Since int    // years after the start when this stage began (negative: before the game)
	Cause string // burned doom ruin saved ruled
	Event string // what happened, in words
}

var rgDoomCauses = []string{"a hard famine", "three dry summers", "a sickness that fouled the wells", "the long wars on the border", "the tax collectors", "a fire in the granaries"}

func (g *Game) rgHash(tag string, id int) uint64 {
	return HashStr(fmt.Sprintf("%s:rg:%s:%d", g.Seed, tag, id))
}

func rgYear(day int) int { return day / YearDays }

func (g *Game) rgDoomCause(bd *BurgData) string {
	h := g.rgHash("cause", bd.ID)
	list := append([]string(nil), rgDoomCauses...)
	if bd.Port {
		list = append(list, "a harbour silted up", "pirates who never left")
	}
	if w := g.W; w != nil && w.Sector(bd, "mining") > 0.2 {
		list = append(list, "a mine that ran dry")
	}
	return list[int(h%uint64(len(list)))]
}

// rgFateOf is the pure fate of a town at the current game day.
func (g *Game) rgFateOf(bd *BurgData) rgFate {
	s := g.rg()
	yrs := rgYear(g.Day())
	h := func(tag string) uint64 { return g.rgHash(tag, bd.ID) }
	// the wanderer's deeds first
	if d, ok := s.Burned[bd.ID]; ok && s.Saved[bd.ID] <= d {
		b := rgYear(d)
		why := "the burning"
		if s.Why[bd.ID] == "blood" {
			why = "the killings"
		}
		gap := 3 + int(h("gap")%6)
		rebuild := int(h("rb")%100) < 65
		if _, saved := s.Saved[bd.ID]; saved {
			rebuild = true
		}
		switch {
		case yrs < b+gap:
			return rgFate{rgAbandoned, b, "burned", why}
		case rebuild && yrs < b+gap+30:
			return rgFate{rgRebuilt, b + gap, "burned", why}
		case rebuild:
			return rgFate{rgThriving, b + gap + 30, "burned", why}
		case yrs < b+gap+15:
			return rgFate{rgAbandoned, b, "burned", why}
		case yrs < b+gap+75:
			return rgFate{rgOvergrown, b + gap + 15, "burned", why}
		}
		return rgFate{rgResettled, b + gap + 75, "burned", why}
	}
	if d, ok := s.Saved[bd.ID]; ok {
		return rgFate{rgThriving, rgYear(d), "saved", "a wanderer's help"}
	}
	if d, ok := s.Ruled[bd.ID]; ok {
		return rgFate{rgThriving, rgYear(d), "ruled", "a wanderer's rule"}
	}
	if !bd.Alive {
		// a ruin at the start: some are resettled later, and later still rebuilt
		old := -(20 + int(h("old")%150))
		if int(h("rs")%100) < 45 {
			at := 20 + int(h("rsy")%100)
			switch {
			case yrs >= at+35:
				return rgFate{rgRebuilt, at + 35, "ruin", "settlers who stayed"}
			case yrs >= at:
				return rgFate{rgResettled, at, "ruin", "settlers"}
			}
		}
		return rgFate{rgOvergrown, old, "ruin", "the fall of the town"}
	}
	if bd.Capital {
		return rgFate{Stage: rgThriving}
	}
	risk := 0.30
	switch g.W.cityTier[bd.ID] {
	case 1:
		risk = 0.10
	case 2:
		risk = 0.02
	}
	if bd.Bandit {
		risk += 0.10
	}
	if st := g.W.stateByID[bd.State]; st != nil && len(st.Wars) > 0 {
		risk += 0.08
	}
	if int(h("doom")%1000) >= int(risk*1000) {
		return rgFate{Stage: rgThriving}
	}
	d := 20 + int(h("dy")%100)
	a := 10 + int(h("ay")%16)
	r := 35 + int(h("ry")%51)
	cause := g.rgDoomCause(bd)
	switch {
	case yrs < d:
		return rgFate{Stage: rgThriving}
	case yrs < d+a:
		return rgFate{rgAbandoned, d, "doom", cause}
	case yrs < d+a+r || int(h("rs")%100) >= 65:
		return rgFate{rgOvergrown, d + a, "doom", cause}
	}
	return rgFate{rgResettled, d + a + r, "doom", cause}
}

// rgLabel is the suffix for a town's name on maps and lists ("" while it thrives).
func (g *Game) rgLabel(bd *BurgData) string {
	if g.W == nil || bd == nil {
		return ""
	}
	f := g.rgFateOf(bd)
	switch f.Stage {
	case rgAbandoned:
		return " (abandoned)"
	case rgOvergrown:
		if bd.Alive {
			return " (overgrown ruin)"
		}
	case rgResettled:
		return " (resettled)"
	case rgRebuilt:
		return " (rebuilt)"
	}
	return ""
}

// ---------------------------------------------------------------- world map hooks

var rgGlyphDay = -1
var rgGlyphW *WorldData
var rgGlyphVer int
var rgGlyphCache = map[int]rgFate{}

func (g *Game) rgFateCached(bd *BurgData) rgFate {
	s := g.rg()
	ver := len(s.Saved) + 3*len(s.Burned) + 7*len(s.Ruled)
	if rgGlyphW != g.W || rgGlyphDay != g.Day() || rgGlyphVer != ver {
		rgGlyphW, rgGlyphDay, rgGlyphVer = g.W, g.Day(), ver
		rgGlyphCache = map[int]rgFate{}
	}
	if f, ok := rgGlyphCache[bd.ID]; ok {
		return f
	}
	f := g.rgFateOf(bd)
	rgGlyphCache[bd.ID] = f
	return f
}

// rgWorldGlyph is the world-map glyph hook (inside the Known block, after the town glyph).
func (g *Game) rgWorldGlyph(p int, gl rune, f byte) (rune, byte) {
	if g.W == nil || g.W.Burg[p] <= 0 {
		return gl, f
	}
	bd := g.W.burgByID[g.W.Burg[p]]
	if bd == nil || bd.Layer != "" && bd.Layer != "surface" {
		return gl, f
	}
	switch g.rgFateCached(bd).Stage {
	case rgAbandoned, rgOvergrown:
		return 'Ω', CDarkGray
	case rgResettled:
		if !bd.Alive {
			return 'o', CBrown
		}
	}
	return gl, f
}

// rgInfo is the world-map info hook: what became of the town on the cursor's parasang.
func (g *Game) rgInfo(p int) []string {
	if g.W == nil || g.W.Burg[p] <= 0 || !g.Known[p] {
		return nil
	}
	bd := g.W.burgByID[g.W.Burg[p]]
	if bd == nil || bd.Layer != "" && bd.Layer != "surface" {
		return nil
	}
	f := g.rgFateCached(bd)
	yr := g.W.Year + f.Since
	switch f.Stage {
	case rgAbandoned:
		return []string{fmt.Sprintf("{{K|%s is abandoned: left in year %d after %s}}", bd.Name, yr, f.Event)}
	case rgOvergrown:
		if bd.Alive {
			return []string{fmt.Sprintf("{{K|%s is an overgrown ruin: empty since year %d}}", bd.Name, yr)}
		}
	case rgResettled:
		return []string{fmt.Sprintf("{{y|Settlers have come back to %s (year %d)}}", bd.Name, yr)}
	case rgRebuilt:
		return []string{fmt.Sprintf("{{G|%s has been rebuilt: new timber since year %d}}", bd.Name, yr)}
	}
	return nil
}

// ---------------------------------------------------------------- recording the player's deeds

func rgTurn(g *Game) {
	if g.Turn%60 != 0 || g.W == nil || g.P == nil || g.Cur == nil || g.Sandboxed() {
		return
	}
	s := g.rg()
	day := g.Day()
	// what the town board knows (townvoice.go)
	rb := g.rb()
	pos, neg := map[int]int{}, map[int]int{}
	lastPos, lastNeg := map[int]int{}, map[int]int{}
	for _, d := range rb.Deeds {
		if d.Town <= 0 {
			continue
		}
		if d.W > 0 {
			pos[d.Town] += d.W
			lastPos[d.Town] = maxi(lastPos[d.Town], d.Day)
		} else {
			neg[d.Town] += -d.W
			lastNeg[d.Town] = maxi(lastNeg[d.Town], d.Day)
		}
	}
	for id, p := range pos {
		if _, ok := s.Saved[id]; !ok && p >= 5 && p >= neg[id] {
			s.Saved[id] = lastPos[id]
		}
	}
	for id, n := range neg {
		if _, ok := s.Burned[id]; !ok && n >= 6 && n > pos[id] {
			s.Burned[id], s.Why[id] = lastNeg[id], "blood"
		}
	}
	for id, k := range rb.Kills {
		if _, ok := s.Burned[id]; !ok && k >= 6 {
			s.Burned[id], s.Why[id] = day, "blood"
		}
	}
	// a realm you hold office in: its towns flourish
	for realm := range g.po().Office {
		for i := range g.W.Burgs {
			if b := &g.W.Burgs[i]; b.State == realm && b.Alive {
				if _, ok := s.Ruled[b.ID]; !ok {
					s.Ruled[b.ID] = day
				}
			}
		}
	}
	// a great blaze in a town zone
	if zn := g.Cur; zn.Kind == "town" && zn.Burg > 0 && !g.OnWorld {
		n := 0
		for _, f := range zn.Fire {
			if f > 0 {
				n++
			}
		}
		if n >= 80 {
			if _, ok := s.Burned[zn.Burg]; !ok {
				s.Burned[zn.Burg], s.Why[zn.Burg] = day, "fire"
				g.Msg("{{R|The fire is past saving. People will not come back to a town like this soon.}}")
			}
		}
	}
}

// ---------------------------------------------------------------- applying a fate to a zone

func rgZone(g *Game, zn *Zone, _ *Rng) { g.rgVisit(zn) }
func rgEnter(g *Game, zn *Zone)       { g.rgVisit(zn) }

func (g *Game) rgVisit(zn *Zone) {
	w := g.W
	if w == nil || zn.Z != 0 || zn.Sky || zn.Underwater || !w.In(zn.WX, zn.WY) {
		return
	}
	if zn.Kind != "town" && zn.Kind != "outskirts" && zn.Kind != "ruin" {
		return
	}
	p := w.Idx(zn.WX, zn.WY)
	bd := w.burgByID[w.Burg[p]]
	if bd == nil || bd.Layer != "" && bd.Layer != "surface" {
		return
	}
	f := g.rgFateOf(bd)
	s := g.rg()
	code := int(f.Stage) + 1
	plaque := f.Stage == rgThriving && (f.Cause == "saved" || f.Cause == "ruled") && zn.ZX == 1 && zn.ZY == 1
	if plaque {
		code = 6
	}
	if f.Stage == rgThriving && !plaque {
		return
	}
	if s.Applied[zn.Key] == code {
		return
	}
	if _, ok := s.Base[zn.Key]; !ok {
		s.Base[zn.Key] = zn.Name
	}
	s.Applied[zn.Key] = code
	g.rgPaint(zn, bd, f, plaque)
}

// rgMix holds per-tile odds for a stage.
type rgMix struct {
	wallRubble, wallVine, wallTimber float64
	floorRubble, floorMoss, floorBush float64
	streetGreen, sapling, dead        float64
	stall, timberFloor, scaffold      float64
}

var rgMixes = map[rgStage]rgMix{
	rgAbandoned: {wallRubble: 0.07, wallVine: 0.06, floorRubble: 0.06, floorMoss: 0.04, streetGreen: 0.12, sapling: 0.004, dead: 0.003, stall: 0.25},
	rgOvergrown: {wallRubble: 0.17, wallVine: 0.30, floorRubble: 0.22, floorMoss: 0.22, floorBush: 0.08, streetGreen: 0.50, sapling: 0.028, dead: 0.010, stall: 0.80},
	rgRebuilt:   {wallTimber: 0.14, timberFloor: 0.55, scaffold: 0.05},
	rgResettled: {wallRubble: 0.08, wallVine: 0.12, wallTimber: 0.06, floorRubble: 0.10, floorMoss: 0.10, streetGreen: 0.22, sapling: 0.010, dead: 0.004, stall: 0.40, timberFloor: 0.30, scaffold: 0.04},
}

func rgNearGate(zn *Zone, x, y int) bool {
	for _, d := range dirs8 {
		switch zn.At(x+d[0], y+d[1]) {
		case TGateArch, TGateClosed, TDoor, TDoorOpen, TDown, TUp:
			return true
		}
	}
	return false
}

func rgOpenAround(zn *Zone, x, y int) int {
	n := 0
	for _, d := range dirs8 {
		if zn.Walkable(x+d[0], y+d[1]) {
			n++
		}
	}
	return n
}

func (g *Game) rgPaint(zn *Zone, bd *BurgData, f rgFate, plaque bool) {
	w := g.W
	r := NewRng(fmt.Sprintf("%s:rg:%s:%d", w.Seed, zn.Key, int(f.Stage)))
	pt := tgNewPainter(zn)
	m := rgMixes[f.Stage]
	changed := false
	set := func(x, y int, t Terrain) {
		if !pt.ok(x, y) || zn.At(x, y) == t {
			return
		}
		zn.Set(x, y, t)
		changed = true
	}
	if !plaque {
		for y := 1; y < MH-1; y++ {
			for x := 1; x < MW-1; x++ {
				t := zn.At(x, y)
				u := r.Float()
				edge := x < 3 || y < 3 || x > MW-4 || y > MH-4
				switch {
				case isWallTerrain(t):
					if rgNearGate(zn, x, y) {
						continue
					}
					switch {
					case u < m.wallRubble:
						set(x, y, TRubble)
					case u < m.wallRubble+m.wallVine:
						set(x, y, THedgeWall) // walls under vines
					case u < m.wallRubble+m.wallVine+m.wallTimber:
						set(x, y, TWoodWall) // patched with fresh timber
					}
				case isFloorTerrain(t):
					switch {
					case u < m.floorRubble:
						set(x, y, TRubble) // fallen roof
					case u < m.floorRubble+m.floorMoss:
						set(x, y, TMoss)
					case u < m.floorRubble+m.floorMoss+m.floorBush:
						set(x, y, TBush)
					case t != TWoodFloor && u < m.floorRubble+m.floorMoss+m.floorBush+m.timberFloor*0.3:
						set(x, y, TWoodFloor)
					}
					if m.scaffold > 0 && r.Chance(m.scaffold) && rgBesideWall(zn, x, y) {
						set(x, y, TScaffold)
					}
				case t == TStall:
					if u < m.stall {
						set(x, y, TRubble)
					}
				case t == TCobble || t == TRoad || t == TDirt || t == TGrass:
					if u < m.streetGreen {
						switch r.Int(0, 2) {
						case 0:
							set(x, y, TMoss)
						case 1:
							set(x, y, TTallGrass)
						default:
							if t == TGrass || t == TDirt {
								set(x, y, TBush)
							} else {
								set(x, y, TMoss)
							}
						}
					} else if !edge && !rgNearGate(zn, x, y) && (t == TGrass || t == TDirt) && rgOpenAround(zn, x, y) >= 6 {
						switch v := r.Float(); {
						case v < m.sapling:
							set(x, y, TTree)
						case v < m.sapling+m.dead:
							set(x, y, TDeadTree)
						}
					}
					if m.scaffold > 0 && (t == TDirt || t == TGrass) && r.Chance(m.scaffold*0.6) && rgBesideWall(zn, x, y) {
						set(x, y, TScaffold)
					}
				case t == TRubble:
					if u < m.timberFloor {
						set(x, y, TWoodFloor) // cleared and floored again
					} else if m.sapling > 0.02 && !edge && u < 0.12 && rgOpenAround(zn, x, y) >= 6 {
						set(x, y, TTree)
					}
				}
			}
		}
		g.rgSquatters(zn, bd, f, r, pt, set)
		if f.Stage == rgOvergrown || f.Stage == rgAbandoned || f.Stage == rgResettled {
			g.rgThin(zn, bd, f, r)
		}
	}
	// the sign: centre zone of the town only (or the ruin's)
	if zn.ZX == 1 && zn.ZY == 1 || zn.Kind == "ruin" && zn.ZX == 1 && zn.ZY == 1 {
		if txt := g.rgSign(bd, f, plaque); txt != "" {
			if x, y, ok := zn.FreeSpot(r, func(x, y int) bool { return x > 3 && y > 3 && x < MW-4 && y < MH-4 && pt.ok(x, y) && cvOpenTile(zn.At(x, y)) }); ok {
				zn.Set(x, y, TInscription)
				zn.Notes[y*MW+x] = txt
				changed = true
			}
		}
	}
	g.rgName(zn, bd, f, plaque)
	if changed {
		g.connect(zn)
		g.computeLight(zn)
	}
}

func rgBesideWall(zn *Zone, x, y int) bool {
	for _, d := range [][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
		if isWallTerrain(zn.At(x+d[0], y+d[1])) {
			return true
		}
	}
	return false
}

// rgName puts the stage into the zone's name, always built from the generated name.
func (g *Game) rgName(zn *Zone, bd *BurgData, f rgFate, plaque bool) {
	base := g.rg().Base[zn.Key]
	if base == "" {
		return
	}
	name := base
	if !plaque {
		switch f.Stage {
		case rgAbandoned:
			name = base + " (abandoned)"
		case rgOvergrown:
			if bd.Alive {
				name = base + " (overgrown ruin)"
			}
		case rgResettled:
			if strings.HasPrefix(base, "Ruins of") {
				name = "Resettled ruins of" + strings.TrimPrefix(base, "Ruins of")
			} else {
				name = base + " (resettled)"
			}
		case rgRebuilt:
			if strings.HasPrefix(base, "Ruins of") {
				name = "New " + strings.TrimPrefix(base, "Ruins of ")
			} else {
				name = base + " (rebuilt)"
			}
		}
	}
	zn.Name = name
}

// rgSign is the one-line inscription: what happened, and when.
func (g *Game) rgSign(bd *BurgData, f rgFate, plaque bool) string {
	yr := g.W.Year + f.Since
	pick := func(tag string, l []string) string { return l[int(g.rgHash(tag, bd.ID)%uint64(len(l)))] }
	switch {
	case plaque && f.Cause == "saved":
		return pick("pl", []string{
			fmt.Sprintf("A brass plaque: %s stood on in the year %d because a wanderer would not look away.", bd.Name, yr),
			fmt.Sprintf("Cut into the lintel: in the year %d the people of %s were helped when nobody else came.", yr, bd.Name),
		})
	case plaque:
		return fmt.Sprintf("Carved in the market stone: in the year %d a wanderer took the reins of %s. The streets are swept and the granaries are full.", yr, bd.Name)
	}
	who := ""
	if f.Cause == "burned" {
		who = fmt.Sprintf(" The survivors say %s set the fire, and they do not say it kindly.", g.P.Name)
	}
	switch f.Stage {
	case rgAbandoned:
		return fmt.Sprintf("Chalked on the door of %s: left in the year %d after %s. The lamps are out and the doors are nailed shut.%s", bd.Name, yr, f.Event, who)
	case rgOvergrown:
		if !bd.Alive {
			return pick("old", []string{
				fmt.Sprintf("Cut into a fallen lintel, half under moss: here stood %s. The roots hold the stones now.", bd.Name),
				fmt.Sprintf("A cairn of fallen stones and one line scratched on it: %s was here, and then it was not.", bd.Name),
			})
		}
		return fmt.Sprintf("Cut deep before the roots took the stone: %s, empty since the year %d, after %s. Nobody has come back.%s", bd.Name, yr, f.Event, who)
	case rgRebuilt:
		return fmt.Sprintf("Fresh paint on a new beam: %s was raised again in the year %d after %s. The scaffolds are still up.%s", bd.Name, yr, f.Event, who)
	case rgResettled:
		return fmt.Sprintf("Chalked on a beam in a hurry: settlers came back to %s in the year %d. They keep to the old quarter and ask few questions.", bd.Name, yr)
	}
	return ""
}

// rgThin empties a living town: most plain townsfolk and guards are gone; named and special people stay.
func (g *Game) rgThin(zn *Zone, bd *BurgData, f rgFate, r *Rng) {
	if !bd.Alive || zn.Kind == "ruin" {
		return
	}
	cut := 0.7
	switch f.Stage {
	case rgOvergrown:
		cut = 0.97
	case rgResettled:
		cut = 0.5
	}
	keep := zn.Actors[:0]
	for _, a := range zn.Actors {
		plain := (a.Role == "villager" || a.Role == "" || a.Role == "guard" || a.Role == "merchant") &&
			a.Quest == 0 && a.Secret == 0 && a.SimPer == 0 && a.Teach == "" && len(a.Flags) == 0 && a.Comp == nil && a.AI != "follower" && a.Level < 100
		if plain && r.Chance(cut) {
			continue
		}
		keep = append(keep, a)
	}
	zn.Actors = keep
}

// rgSquatters moves people into the empty quarter: a camp, a hermit or a few squatters.
func (g *Game) rgSquatters(zn *Zone, bd *BurgData, f rgFate, r *Rng, pt *tgPainter, set func(x, y int, t Terrain)) {
	w := g.W
	chance := 0.5
	if f.Stage == rgRebuilt {
		return
	}
	if f.Stage == rgOvergrown {
		chance = 0.7
	}
	if !r.Chance(chance) {
		return
	}
	cu := w.cultByID[bd.Culture]
	spot := func() (int, int, bool) {
		return zn.FreeSpot(r, func(x, y int) bool { return x > 4 && y > 3 && x < MW-5 && y < MH-4 && pt.ok(x, y) })
	}
	cx, cy, ok := spot()
	if !ok {
		return
	}
	add := func(a *Actor, rad int) {
		x, y, ok := zn.FreeSpot(r, func(x, y int) bool { return abs(x-cx) <= rad && abs(y-cy) <= rad })
		if !ok {
			return
		}
		a.X, a.Y, a.HomeX, a.HomeY = x, y, x, y
		zn.Actors = append(zn.Actors, a)
	}
	switch kind := r.Int(0, 2); {
	case kind == 0 && f.Stage != rgResettled: // a bandit camp
		set(cx, cy, TBrazier)
		for _, d := range dirs8 {
			if r.Chance(0.3) {
				set(cx+d[0]*2, cy+d[1], TBones)
			}
		}
		for i := 0; i < r.Int(2, 3); i++ {
			a := g.Spawn("bandit", maxi(1, zn.Level), r)
			a.Desc = "A bandit camped in the empty town, wary of anyone who comes through the broken gate."
			add(a, 4)
		}
	case kind == 1: // a hermit
		a := g.makeNPC("villager", bd, cu, r)
		a.Title, a.AI, a.Gold = "hermit", "still", 0
		a.Desc = fmt.Sprintf("A hermit who sleeps in what is left of %s and talks to the walls.", bd.Name)
		add(a, 2)
		set(cx, cy+1, TMoss)
	default: // squatters or settlers
		title := "squatter"
		if f.Stage == rgResettled {
			title = "settler"
			set(cx, cy, TBrazier)
			set(cx-3, cy, TTent)
			set(cx+3, cy, TTent)
		}
		for i := 0; i < r.Int(2, 3); i++ {
			a := g.makeNPC("villager", bd, cu, r)
			a.Title, a.AI, a.Gold = title, "wander", r.Int(0, 6)
			a.Desc = fmt.Sprintf("A %s making do in the old streets of %s.", title, bd.Name)
			add(a, 4)
		}
	}
}
