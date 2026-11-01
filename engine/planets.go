package engine

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
)

// Planets and rifts (prefix un). A world file is either one world or a "wanderer-universe":
// {kind, start, planets: [world...], rifts: [{id, name, a:{planet,burg}, b:{planet,burg}}]}.
// ParseWorld returns the start planet; every planet points back at the shared Universe.
//
// Per-planet isolation works by swapping: the Game always holds the current planet's state in
// its usual fields (Zones, Known, Slain, Rep, ...) and the planet-local feature stores
// (unLocalKeys). Stepping through a rift moves all of it into an unSnap kept in g.ext("uni")
// and restores (or freshly creates) the destination's. Everything keyed by zone key, parasang,
// burg, realm, monster or person id therefore never sees another planet's ids. The player,
// inventory, skills, companions that follow, deeds, messages and the global feature stores
// travel along. Worlds are never saved: the host passes the world file again on load and
// LoadGame picks the saved planet out of it (unResolve).

type Universe struct {
	Kind    string                 `json:"kind"`
	Start   int                    `json:"start"`
	Planets []*WorldData           `json:"planets"`
	Rifts   []RiftData             `json:"rifts"`
	ends    map[int]map[int][]int  // planet → burg id → rift indices
	gateZ   map[int]map[string]int // planet → zone key of the burg's town → burg id
	gateP   map[int]map[int][]int  // planet → parasang → rift burg ids (world map)
}

// unTownZone is the zone holding a burg's town: surface, deep layer, sea floor or sky isle.
func unTownZone(w *WorldData, bd *BurgData) (wx, wy, z int) {
	wx, wy = bd.P%w.W, bd.P/w.W
	switch bd.Layer {
	case "deep":
		if _, ok := w.deepBurg[bd.Depth][bd.P]; ok {
			z = bd.Depth
		}
	case "sea":
		if _, ok := w.seaBurg[bd.P]; ok {
			z = 1
		}
	case "sky":
		if _, ok := w.skyBurg[bd.P]; ok {
			z = -1
		}
	}
	return
}

type RiftEnd struct {
	Planet int `json:"planet"`
	Burg   int `json:"burg"`
}

type RiftData struct {
	ID   int     `json:"id"`
	Name string  `json:"name"`
	A    RiftEnd `json:"a"`
	B    RiftEnd `json:"b"`
}

// feature stores whose ids belong to one planet (bounties, money of its realms, homes, orders,
// parlors, quest spine, laws, war, world sim, villains)
var unLocalKeys = []string{"bh", "cu", "home", "orders", "pl", "spine", "uw", "war", "ws", "vl"}

func unParse(s string) (*WorldData, error) {
	u := &Universe{}
	if err := json.Unmarshal([]byte(s), u); err != nil {
		return nil, err
	}
	if len(u.Planets) == 0 {
		return nil, errors.New("universe file has no planets")
	}
	for i, w := range u.Planets {
		if w == nil {
			return nil, fmt.Errorf("planet %d is empty", i)
		}
		w.index()
		w.uni, w.uniIdx = u, i
	}
	u.Start = clampi(u.Start, 0, len(u.Planets)-1)
	u.index()
	return u.Planets[u.Start], nil
}

func (u *Universe) index() {
	u.ends = map[int]map[int][]int{}
	u.gateZ = map[int]map[string]int{}
	u.gateP = map[int]map[int][]int{}
	ok := func(e RiftEnd) bool {
		return e.Planet >= 0 && e.Planet < len(u.Planets) && u.Planets[e.Planet].burgByID[e.Burg] != nil
	}
	for i, r := range u.Rifts {
		if !ok(r.A) || !ok(r.B) {
			continue
		}
		for _, e := range []RiftEnd{r.A, r.B} {
			if u.ends[e.Planet] == nil {
				u.ends[e.Planet] = map[int][]int{}
			}
			if len(u.ends[e.Planet][e.Burg]) == 0 {
				w := u.Planets[e.Planet]
				bd := w.burgByID[e.Burg]
				wx, wy, z := unTownZone(w, bd)
				if u.gateZ[e.Planet] == nil {
					u.gateZ[e.Planet], u.gateP[e.Planet] = map[string]int{}, map[int][]int{}
				}
				u.gateZ[e.Planet][ZoneKey(wx, wy, 1, 1, z)] = e.Burg
				u.gateP[e.Planet][bd.P] = append(u.gateP[e.Planet][bd.P], e.Burg)
			}
			u.ends[e.Planet][e.Burg] = append(u.ends[e.Planet][e.Burg], i)
		}
	}
}

func unUni(w *WorldData) *Universe {
	if w.uni == nil {
		w.uni = &Universe{Kind: "single", Planets: []*WorldData{w}}
		w.uniIdx = 0
	}
	if w.uni.ends == nil {
		w.uni.index()
	}
	return w.uni
}

// unSeed is the game seed on planet i: the planet's own world seed (kept distinct from home's).
func unSeed(u *Universe, i int) string {
	s := u.Planets[i].Seed
	if i != u.Start && s == u.Planets[u.Start].Seed {
		s += fmt.Sprintf("~p%d", i)
	}
	return s
}

// ---- saved state ----

type unSnap struct {
	Zones     map[string]*Zone           `json:"zones"`
	Known     map[int]bool               `json:"known,omitempty"`
	Slain     map[int]bool               `json:"slain,omitempty"`
	Taken     map[int]bool               `json:"taken,omitempty"`
	Visited   map[string]bool            `json:"visited,omitempty"`
	Read      map[string]bool            `json:"read,omitempty"`
	FeastSeen map[string]bool            `json:"feastSeen,omitempty"`
	Rep       map[string]int             `json:"rep,omitempty"`
	HomeState int                        `json:"hs"`
	HomeFaith int                        `json:"hf"`
	StartX    int                        `json:"sx"`
	StartY    int                        `json:"sy"`
	StartBurg int                        `json:"sb"`
	FeastNote string                     `json:"fn,omitempty"`
	WxLast    string                     `json:"wx,omitempty"`
	ArenaKey  string                     `json:"ak,omitempty"`
	ArenaFoes []int                      `json:"af,omitempty"`
	Quests    []*Quest                   `json:"quests,omitempty"`
	Store     map[string]json.RawMessage `json:"store,omitempty"`
	InSpots   map[string]*inSpot         `json:"inSp,omitempty"` // zone-keyed parts of the global "in" store
	InZones   map[string]bool            `json:"inZn,omitempty"`
	InEnc     map[string]bool            `json:"inEn,omitempty"`
	LqFor     map[string]int             `json:"lqFg,omitempty"`
	Murim     *unMr                      `json:"mr,omitempty"` // sect standing on this planet (murim.go)
}

// unMr: the planet-local half of murim's mrState. Sects are generated per world, so membership,
// missions, raids, tournament and the sect rep baselines stay on the planet they belong to;
// cultivation (realm, qi, techniques, applied stats) travels with you.
type unMr struct {
	Sect    string          `json:"sect,omitempty"`
	Rank    int             `json:"rank,omitempty"`
	Contrib int             `json:"con,omitempty"`
	Mission *mrMission      `json:"mis,omitempty"`
	Raids   map[string]bool `json:"raids,omitempty"`
	Tourn   mrTourn         `json:"tourn"`
	PayDay  int             `json:"pay,omitempty"`
	Seen    map[string]bool `json:"seen,omitempty"`
	Lock    map[string]int  `json:"lock,omitempty"`
	Base    map[string]int  `json:"base,omitempty"`
	Coach   map[string]int  `json:"coach,omitempty"`
}

type unState struct {
	Init   bool            `json:"i"`
	Planet int             `json:"p"`
	Home   string          `json:"h"` // seed of the home planet: the host files the run under it
	Snaps  map[int]*unSnap `json:"s,omitempty"`
	Trips  int             `json:"n,omitempty"`
	Seen   map[int]bool    `json:"seen,omitempty"` // planets visited
	opts   []int           // rift indices offered at the gate
	from   int             // burg of the gate being used
}

func (g *Game) unRaw() *unState {
	return g.ext("uni", func() any { return &unState{} }).(*unState)
}

func (g *Game) unSt() *unState {
	st := g.unRaw()
	if !st.Init && g.W != nil {
		st.Init, st.Planet, st.Home = true, g.W.uniIdx, g.Seed
	}
	if st.Snaps == nil {
		st.Snaps = map[int]*unSnap{}
	}
	if st.Seen == nil {
		st.Seen = map[int]bool{}
	}
	st.Seen[st.Planet] = true
	return st
}

// unResolve picks the planet a loaded save stands on (old saves: the start planet).
func (g *Game) unResolve(w *WorldData) *WorldData {
	st := g.unRaw()
	u := unUni(w)
	if !st.Init {
		return w
	}
	if st.Planet >= 0 && st.Planet < len(u.Planets) {
		return u.Planets[st.Planet]
	}
	st.Planet = w.uniIdx
	return w
}

// unHome is the seed the host files this run under (constant across rift travel).
func (g *Game) unHome() string {
	if st := g.unSt(); st.Home != "" {
		return st.Home
	}
	return g.Seed
}

func (g *Game) unPlanetName() string {
	if g.W == nil {
		return ""
	}
	return nonEmpty(g.W.Name, g.W.Planet)
}

// UniversePlanets is how many planets the loaded world file holds (1 for a single world).
func (g *Game) UniversePlanets() int { return len(unUni(g.W).Planets) }

// PlanetIndex is the planet the player stands on.
func (g *Game) PlanetIndex() int { return g.W.uniIdx }

func unRiftBurgAt(w *WorldData, p int) bool {
	return len(unUni(w).gateP[w.uniIdx][p]) > 0
}

// unOther is the far end of rift ri seen from burg on planet pl.
func unOther(r RiftData, pl, burg int) RiftEnd {
	if r.A.Planet == pl && r.A.Burg == burg {
		return r.B
	}
	return r.A
}

func (u *Universe) planetLabel(i int) string {
	w := u.Planets[i]
	n := nonEmpty(w.Name, fmt.Sprintf("planet %d", i+1))
	if w.Planet != "" && w.Planet != w.Name {
		n += " (" + w.Planet + ")"
	}
	return n
}

func (g *Game) unRiftInfo(p int) []string {
	w := g.W
	if !unRiftBurgAt(w, p) {
		return nil
	}
	u := unUni(w)
	var out []string
	for _, burg := range u.gateP[w.uniIdx][p] {
		for _, ri := range u.ends[w.uniIdx][burg] {
			r := u.Rifts[ri]
			to := unOther(r, w.uniIdx, burg)
			out = append(out, fmt.Sprintf("{{M|Ø %s}} to %s", r.Name, u.Planets[to.Planet].burgByID[to.Burg].Name+", "+u.planetLabel(to.Planet)))
		}
	}
	return out
}

// ---- the gate tile ----

func init() {
	zoneHooks = append(zoneHooks, func(g *Game, zn *Zone, _ *Rng) { g.unGate(zn) })
	enterHooks = append(enterHooks, func(g *Game, zn *Zone) { g.unGate(zn); unSeenNote(g) })
	sideHooks = append(sideHooks, unSide)
	modeKeys["rift"] = unKey
	modeDraws["rift"] = unDraw
}

func unSeenNote(g *Game) { g.unSt() }

// unGate makes sure the town zone of a rift burg has its gate; returns its position.
func (g *Game) unGate(zn *Zone) (int, int, bool) {
	u := unUni(g.W)
	burg, ok := u.gateZ[g.W.uniIdx][zn.Key]
	if !ok {
		return 0, 0, false
	}
	for i, t := range zn.T {
		if t == TPortal {
			return i % MW, i / MW, true
		}
	}
	var names []string
	for _, ri := range u.ends[g.W.uniIdx][burg] {
		names = append(names, u.Rifts[ri].Name)
	}
	r := NewRng(g.W.Seed + ":rift:" + zn.Key)
	x, y, ok := zn.FreeSpot(r, func(x, y int) bool {
		if abs(x-40) > 16 || abs(y-12) > 7 || zn.At(x, y) == TDoor || zn.At(x, y) == TDoorOpen {
			return false
		}
		free := 0
		for _, d := range [][2]int{{0, 1}, {0, -1}, {1, 0}, {-1, 0}} {
			if zn.Walkable(x+d[0], y+d[1]) {
				free++
			}
		}
		return free >= 3
	})
	if !ok {
		x, y = 40, 12
	}
	zn.Set(x, y, TPortal)
	if zn.Notes == nil {
		zn.Notes = map[int]string{}
	}
	zn.Notes[y*MW+x] = strings.Join(names, ", ")
	return x, y, true
}

// unBump: bumping the gate asks where to go.
func (g *Game) unBump(x, y int) {
	w := g.W
	zn := g.Cur
	u := unUni(w)
	burg := u.gateZ[w.uniIdx][zn.Key]
	st := g.unSt()
	st.opts, st.from = append([]int{}, u.ends[w.uniIdx][burg]...), burg
	if len(st.opts) == 0 {
		g.Msg("The rift gate is dark. Nothing answers on the other side.")
		return
	}
	g.Mode = "rift"
	g.Sel = 0
}

func unKey(g *Game, k string) {
	st := g.unSt()
	switch {
	case k == "Escape" || k == "n" || k == "N":
		g.Mode = "main"
		g.Msg("You step back from the shimmer.")
	case (k == "y" || k == "Y" || k == "Enter") && len(st.opts) > 0:
		g.unTravel(st.opts[clampi(g.Sel, 0, len(st.opts)-1)], st.from)
	case k == "Up" || k == "N8":
		g.Sel = maxi(0, g.Sel-1)
	case k == "Down" || k == "N2":
		g.Sel = mini(len(st.opts)-1, g.Sel+1)
	default:
		if i := letterIndex(k); i >= 0 && i < len(st.opts) {
			g.unTravel(st.opts[i], st.from)
		}
	}
}

func unDraw(g *Game) {
	s := g.sc
	st := g.unSt()
	u := unUni(g.W)
	h := len(st.opts) + 5
	y0 := maxi(1, (MH-h)/2)
	s.Box(6, y0, 68, h, CMagenta, "Rift gate")
	s.TextB(8, y0+1, "{{M|The air folds open onto another sky.}} Step through?", CGray, CBgDark, 72)
	for i, ri := range st.opts {
		r := u.Rifts[ri]
		to := unOther(r, g.W.uniIdx, st.from)
		bg := byte(CBgDark)
		if i == g.Sel {
			bg = CBgSel
		}
		dest := u.Planets[to.Planet].burgByID[to.Burg].Name + ", " + u.planetLabel(to.Planet)
		s.TextB(8, y0+3+i, fmt.Sprintf("{{W|%c)}} %s {{K|→}} %s", 'a'+i, r.Name, dest), CGray, bg, 72)
	}
	s.TextB(8, y0+h-1, "[y/Enter step through, letter picks, n/Esc stay]", CYellow, CBgDark, 72)
}

func unSide(g *Game) []string {
	if g.UniversePlanets() < 2 {
		return nil
	}
	return []string{"{{M|Ø}} " + g.unPlanetName()}
}

// ---- travel ----

func (g *Game) unTravel(ri, fromBurg int) {
	u := unUni(g.W)
	if ri < 0 || ri >= len(u.Rifts) {
		g.Mode = "main"
		return
	}
	r := u.Rifts[ri]
	to := unOther(r, g.W.uniIdx, fromBurg)
	if !g.UnGo(to.Planet, to.Burg) {
		g.Mode = "main"
		g.Msg("The rift gutters and closes. Nothing happens.")
		return
	}
	g.Msg(fmt.Sprintf("{{M|You step through %s onto %s.}}", r.Name, g.unPlanetName()))
}

// UnGo moves the player (and following companions) to the town of burg on planet pl.
func (g *Game) UnGo(pl, burg int) bool {
	u := unUni(g.W)
	if pl < 0 || pl >= len(u.Planets) {
		return false
	}
	tw := u.Planets[pl]
	bd := tw.burgByID[burg]
	if bd == nil || bd.P < 0 || bd.P >= tw.W*tw.H {
		return false
	}
	st := g.unSt()
	if pl != st.Planet {
		if g.Cur != nil {
			g.cuSync()
		}
		st.Snaps[st.Planet] = g.unCapture()
		g.W, g.Genre, g.Sk, g.Seed = tw, tw.Genre, SkinFor(tw.Genre), unSeed(u, pl)
		if s := st.Snaps[pl]; s != nil {
			g.unRestore(s)
			delete(st.Snaps, pl)
		} else {
			g.unFresh(bd)
		}
		st.Planet = pl
		st.Seen[pl] = true
		st.Trips++
		g.plans, g.sites, g.wmInfo = nil, nil, nil
	}
	wx, wy, z := unTownZone(tw, bd)
	zn := g.GetZone(wx, wy, 1, 1, z)
	x, y, ok := g.unGate(zn)
	if !ok {
		x, y = 40, 12
	} else {
		y++
	}
	g.enterZone(zn, x, y, true)
	g.Mode = "main"
	g.recompute()
	return true
}

// unCapture moves the current planet's state out of the Game.
func (g *Game) unCapture() *unSnap {
	s := &unSnap{Store: map[string]json.RawMessage{}}
	in := inSt(g)
	s.InSpots, s.InZones, s.InEnc = in.Spots, in.Zones, in.Enc
	in.Spots, in.Zones, in.Enc = map[string]*inSpot{}, map[string]bool{}, map[string]bool{}
	lq := g.lq()
	s.LqFor, lq.Foraged = lq.Foraged, map[string]int{}
	mr := g.mr()
	s.Murim = &unMr{mr.Sect, mr.Rank, mr.Contrib, mr.Mission, mr.Raids, mr.Tourn, mr.PayDay, mr.Seen, mr.Lock, mr.Base, mr.Coach}
	mr.Sect, mr.Rank, mr.Contrib, mr.Mission, mr.Tourn, mr.PayDay = "", 0, 0, nil, mrTourn{}, 0
	mr.Raids, mr.Seen, mr.Lock, mr.Base, mr.Coach = nil, nil, nil, nil, nil
	g.flushExt()
	for _, k := range unLocalKeys {
		if raw, ok := g.Store[k]; ok {
			s.Store[k] = raw
			delete(g.Store, k)
		}
		delete(g.extLive, k)
	}
	s.Zones, s.Known, s.Slain, s.Taken = g.Zones, g.Known, g.Slain, g.Taken
	s.Visited, s.Read, s.FeastSeen, s.Rep = g.Visited, g.Read, g.FeastSeen, g.Rep
	s.HomeState, s.HomeFaith, s.StartX, s.StartY, s.StartBurg = g.HomeState, g.HomeFaith, g.StartX, g.StartY, g.StartBurg
	s.FeastNote, s.WxLast, s.ArenaKey, s.ArenaFoes, s.Quests = g.FeastNote, g.WxLast, g.ArenaKey, g.ArenaFoes, g.Quests
	return s
}

// unRestore brings a planet's saved state back.
func (g *Game) unRestore(s *unSnap) {
	g.Zones, g.Known, g.Slain, g.Taken = s.Zones, s.Known, s.Slain, s.Taken
	g.Visited, g.Read, g.FeastSeen, g.Rep = s.Visited, s.Read, s.FeastSeen, s.Rep
	g.HomeState, g.HomeFaith, g.StartX, g.StartY, g.StartBurg = s.HomeState, s.HomeFaith, s.StartX, s.StartY, s.StartBurg
	g.FeastNote, g.WxLast, g.ArenaKey, g.ArenaFoes, g.Quests = s.FeastNote, s.WxLast, s.ArenaKey, s.ArenaFoes, s.Quests
	g.unFixMaps()
	for _, z := range g.Zones {
		if z.Lit == nil {
			z.Lit = make([]bool, MW*MH)
		}
		if z.Notes == nil {
			z.Notes = map[int]string{}
		}
		g.computeLight(z)
	}
	if g.Store == nil {
		g.Store = map[string]json.RawMessage{}
	}
	for k, raw := range s.Store {
		g.Store[k] = raw
	}
	in := inSt(g)
	if s.InSpots != nil {
		in.Spots = s.InSpots
	}
	if s.InZones != nil {
		in.Zones = s.InZones
	}
	if s.InEnc != nil {
		in.Enc = s.InEnc
	}
	if m := s.Murim; m != nil {
		mr := g.mr()
		mr.Sect, mr.Rank, mr.Contrib, mr.Mission, mr.Tourn, mr.PayDay = m.Sect, m.Rank, m.Contrib, m.Mission, m.Tourn, m.PayDay
		mr.Raids, mr.Seen, mr.Lock, mr.Base, mr.Coach = m.Raids, m.Seen, m.Lock, m.Base, m.Coach
		g.mr() // re-make nil maps
	}
	if s.LqFor != nil {
		g.lq().Foraged = s.LqFor
	}
}

// unFresh: first arrival on a planet. Nobody knows you; beasts and the dead hate you as before.
func (g *Game) unFresh(bd *BurgData) {
	old := g.Rep
	g.Zones, g.Known, g.Slain, g.Taken = map[string]*Zone{}, map[int]bool{bd.P: true}, map[int]bool{}, map[int]bool{}
	g.Visited, g.Read, g.FeastSeen = map[string]bool{}, map[string]bool{}, nil
	g.HomeState, g.HomeFaith = -1, -1
	g.initRep()
	for k, v := range old {
		if !strings.HasPrefix(k, "state:") && !strings.HasPrefix(k, "faith:") && !strings.HasPrefix(k, "mr:") {
			g.Rep[k] = v
		}
	}
	g.StartBurg, g.StartX, g.StartY = bd.ID, bd.P%g.W.W, bd.P/g.W.W
	g.FeastNote, g.WxLast, g.ArenaKey, g.ArenaFoes, g.Quests = "", "", "", nil, nil
}

func (g *Game) unFixMaps() {
	if g.Zones == nil {
		g.Zones = map[string]*Zone{}
	}
	if g.Known == nil {
		g.Known = map[int]bool{}
	}
	if g.Slain == nil {
		g.Slain = map[int]bool{}
	}
	if g.Taken == nil {
		g.Taken = map[int]bool{}
	}
	if g.Visited == nil {
		g.Visited = map[string]bool{}
	}
	if g.Read == nil {
		g.Read = map[string]bool{}
	}
	if g.Rep == nil {
		g.Rep = map[string]int{}
	}
}

// unPlanetsLine lists visited planets (journal / tests).
func (g *Game) unPlanetsLine() string {
	u := unUni(g.W)
	st := g.unSt()
	var ids []int
	for i := range st.Seen {
		ids = append(ids, i)
	}
	sort.Ints(ids)
	var names []string
	for _, i := range ids {
		if i < len(u.Planets) {
			names = append(names, u.planetLabel(i))
		}
	}
	return strings.Join(names, ", ")
}

// FuzzRift travels through rift n (mod the rift count) to the end not on this planet (dev tools).
func (g *Game) FuzzRift(n int) bool {
	u := unUni(g.W)
	if len(u.Rifts) == 0 {
		return false
	}
	r := u.Rifts[n%len(u.Rifts)]
	to := r.A
	if to.Planet == g.W.uniIdx {
		to = r.B
	}
	return g.UnGo(to.Planet, to.Burg)
}
