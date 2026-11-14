package engine

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// World and story systems (prefix wy):
// 1. Planet traits (low_gravity, toxic_air, tidally_locked, dead_ruin), seasonal rift storms,
//    star-map screen (Mode == "starmap"), and off-world trade goods pricing.
// 2. Library chronicles that reveal dungeon sites, twisted/out-of-date rumours, historical dreams
//    on rest, 3-verse start-of-run prophecy with dynamic omen resolution, companion narrative arcs
//    (trust -> crisis -> bond/sacrifice/departure/betrayal) with lethal blow interception, and
//    personalized NPC/companion epilogues.

var wyAllPlanetTraits = []string{"low_gravity", "toxic_air", "tidally_locked", "dead_ruin"}

var wyTraitNames = map[string]string{
	"low_gravity":    "Low Gravity (+2 throw/jump range, -50% fall damage)",
	"toxic_air":      "Toxic Atmosphere (1 poison dmg/10 turns outdoors without filter/mask)",
	"tidally_locked": "Tidally Locked (west permanent scorching day, east permanent freezing night)",
	"dead_ruin":      "Dead Ruin World (ancient relics and construct remnants litter the surface)",
}

type wyProphecy struct {
	Verses    [3]string `json:"verses"`
	OmenBeast string    `json:"omenBeast"` // unfulfilled | slain | spared
	OmenRift  string    `json:"omenRift"`  // unfulfilled | crossed | earthbound
	OmenCrown string    `json:"omenCrown"` // unfulfilled | champion | outcast | wanderer
	Resolved  bool      `json:"resolved,omitempty"`
}

type wyCompArc struct {
	CompID     int      `json:"cid"`
	Name       string   `json:"name"`
	Stage      string   `json:"stage"`   // "" | "trust" | "crisis" | "resolution"
	Outcome    string   `json:"outcome"` // "" | "bond" | "sacrifice" | "departure" | "betrayal"
	Sacrificed bool     `json:"sac,omitempty"`
	Events     []string `json:"ev,omitempty"`
}

type wyMetNPC struct {
	ID      int    `json:"id"`
	Name    string `json:"name"`
	Role    string `json:"role"`
	Title   string `json:"title,omitempty"`
	Faction string `json:"fac"`
	Town    string `json:"town"`
	Alive   bool   `json:"alive"`
}

type wyState struct {
	Prophecy       *wyProphecy        `json:"prophecy,omitempty"`
	RevealedSites  map[int]bool       `json:"revSites,omitempty"`
	BooksRead      map[string]bool    `json:"books,omitempty"`
	TwistedRumours map[string]string  `json:"twisted,omitempty"` // "wx,wy" -> kind ("ambush" | "slain")
	RumourSeq      int                `json:"rseq,omitempty"`
	DreamHint      string             `json:"dreamHint,omitempty"`
	DreamTargetP   int                `json:"dreamP,omitempty"`
	DreamSeq       int                `json:"dseq,omitempty"`
	CompArcs       map[int]*wyCompArc `json:"compArcs,omitempty"`
	MetNPCs        map[int]*wyMetNPC  `json:"metNPCs,omitempty"`
	ClosedRifts    map[int]int        `json:"closedRifts,omitempty"` // riftIdx -> season bucket
	TraitOverride  map[int][]string   `json:"traitOv,omitempty"`
	TwistedMem     map[[2]int]string  `json:"-"` // live mirror of TwistedRumours keyed by [2]int
}

func (g *Game) wy() *wyState {
	st := g.ext("wy", func() any { return &wyState{} }).(*wyState)
	if st.RevealedSites == nil {
		st.RevealedSites = map[int]bool{}
	}
	if st.BooksRead == nil {
		st.BooksRead = map[string]bool{}
	}
	if st.TwistedRumours == nil {
		st.TwistedRumours = map[string]string{}
	}
	if st.TwistedMem == nil {
		st.TwistedMem = map[[2]int]string{}
		for k, v := range st.TwistedRumours {
			var wx, wy int
			if _, err := fmt.Sscanf(k, "%d,%d", &wx, &wy); err == nil {
				st.TwistedMem[[2]int{wx, wy}] = v
			}
		}
	}
	if st.CompArcs == nil {
		st.CompArcs = map[int]*wyCompArc{}
	}
	if st.MetNPCs == nil {
		st.MetNPCs = map[int]*wyMetNPC{}
	}
	if st.ClosedRifts == nil {
		st.ClosedRifts = map[int]int{}
	}
	if st.TraitOverride == nil {
		st.TraitOverride = map[int][]string{}
	}
	return st
}

func init() {
	modeKeys["starmap"] = wyStarMapKey
	modeDraws["starmap"] = wyDrawStarMap
	if _, taken := mainKeys["*"]; !taken {
		mainKeys["*"] = func(g *Game) { g.OpenStarMap() }
	}
	turnHooks = append(turnHooks, func(g *Game) {
		wyPlanetTurn(g)
		if g.Cur != nil && g.Cur.Kind != "ship" && g.Turn%60 == 0 {
			wyRiftStormTurn(g)
		}
		g.wyEnsureProphecy()
		g.wyUpdateProphecyProgress()
	})
	zoneHooks = append(zoneHooks, func(g *Game, zn *Zone, r *Rng) {
		wyZoneHook(g, zn, r)
	})
	enterHooks = append(enterHooks, func(g *Game, zn *Zone) {
		wyOnEnterZone(g, zn)
	})
	killHooks = append(killHooks, func(g *Game, d, src *Actor) {
		wyOnKill(g, d, src)
	})
	talkHooks = append(talkHooks, wyTalkOptions)
}

// ---- 1. Planet Traits, Rift Storms, Star-Map & Off-World Trade ----

// wyPlanetTrait deterministically assigns 1-2 traits to a planet from its WorldData and planetID.
func wyPlanetTrait(w *WorldData, planetID int) []string {
	if w == nil {
		return []string{"low_gravity"}
	}
	pw := w
	if u := unUni(w); planetID >= 0 && planetID < len(u.Planets) && u.Planets[planetID] != nil {
		pw = u.Planets[planetID]
	}
	baseH := HashStr(w.Seed + ":wy:traitbase")
	primaryIdx := int((baseH + uint64(maxi(0, planetID))) % uint64(len(wyAllPlanetTraits)))
	out := []string{wyAllPlanetTraits[primaryIdx]}
	h := HashStr(fmt.Sprintf("%s:wy:trait:%d:%s", pw.Seed, planetID, pw.PlanetKind))
	kind := strings.ToLower(pw.PlanetKind)
	kindIdx := -1
	switch {
	case strings.Contains(kind, "moon") || strings.Contains(kind, "asteroid") || strings.Contains(kind, "low"):
		kindIdx = 0
	case strings.Contains(kind, "toxic") || strings.Contains(kind, "swamp") || strings.Contains(kind, "acid") || strings.Contains(kind, "ash"):
		kindIdx = 1
	case strings.Contains(kind, "tidal") || strings.Contains(kind, "eyeball") || strings.Contains(kind, "locked"):
		kindIdx = 2
	case strings.Contains(kind, "dead") || strings.Contains(kind, "ruin") || strings.Contains(kind, "tomb") || strings.Contains(kind, "relic"):
		kindIdx = 3
	}
	if kindIdx >= 0 && wyAllPlanetTraits[kindIdx] != out[0] {
		out = append(out, wyAllPlanetTraits[kindIdx])
	} else if (h>>8)%2 == 0 {
		secIdx := (primaryIdx + 1 + int((h>>16)%3)) % len(wyAllPlanetTraits)
		if wyAllPlanetTraits[secIdx] != out[0] {
			out = append(out, wyAllPlanetTraits[secIdx])
		}
	}
	return out
}

func (g *Game) wyPlanetTraits() []string {
	if g == nil || g.W == nil {
		return nil
	}
	pid := g.PlanetIndex()
	if ov, ok := g.wy().TraitOverride[pid]; ok && len(ov) > 0 {
		return ov
	}
	return wyPlanetTrait(g.W, pid)
}

func (g *Game) wySetPlanetTraits(planetID int, traits ...string) {
	g.wy().TraitOverride[planetID] = append([]string{}, traits...)
}

func (g *Game) wyHasPlanetTrait(trait string) bool {
	for _, t := range g.wyPlanetTraits() {
		if t == trait {
			return true
		}
	}
	return false
}

// wyThrowBonus returns +2 throw/jump range when on a low_gravity planet.
func wyThrowBonus(g *Game) int {
	if g != nil && g.wyHasPlanetTrait("low_gravity") {
		return 2
	}
	return 0
}

func (g *Game) wyThrowBonus() int { return wyThrowBonus(g) }

// wyFallDamage halves fall or knockback damage on a low_gravity planet.
func wyFallDamage(g *Game, dmg int) int {
	if dmg <= 0 {
		return 0
	}
	if g != nil && g.wyHasPlanetTrait("low_gravity") {
		return maxi(1, dmg/2)
	}
	return dmg
}

func (g *Game) wyFallDamage(dmg int) int { return wyFallDamage(g, dmg) }

// wyTidallyLockedLight reports whether the current planet is tidally locked and whether
// the player's current longitude (WX < W/2) is in permanent scorching daylight or freezing night.
func wyTidallyLockedLight(g *Game) (locked bool, daylight bool) {
	if g == nil || g.W == nil || !g.wyHasPlanetTrait("tidally_locked") {
		return false, g != nil && g.daylight()
	}
	w := maxi(2, g.W.W)
	return true, g.WX < w/2
}

func (g *Game) wyTidallyLockedLight() (locked bool, daylight bool) {
	return wyTidallyLockedLight(g)
}

func (g *Game) wyHasToxicProtection() bool {
	if g == nil || g.P == nil {
		return false
	}
	p := g.P
	if p.Pow("poison") > 0 || p.Pow("regen") >= 3 || p.Has("poison") || p.Has("poison_resist") || p.Has("gasmask") || p.Has("filter") || p.St("poison_resist") > 0 {
		return true
	}
	matchToken := func(s string) bool {
		s = strings.ToLower(s)
		return strings.Contains(s, "gasmask") || strings.Contains(s, "gas mask") ||
			strings.Contains(s, "filter") || strings.Contains(s, "respirator") ||
			strings.Contains(s, "rebreather") || strings.Contains(s, "poison") ||
			strings.Contains(s, "lung") || strings.Contains(s, "ward")
	}
	checkItem := func(it *Item) bool {
		if it == nil {
			return false
		}
		if matchToken(it.Name) || matchToken(it.Effect) || matchToken(it.Cat) || it.Elem == "poison" {
			return true
		}
		if it.Extra != nil {
			if it.Extra["gasmask"] > 0 || it.Extra["filter"] > 0 || it.Extra["poison"] > 0 || it.Extra["poison_resist"] > 0 {
				return true
			}
		}
		return false
	}
	for _, bp := range p.Body {
		if bp == nil || bp.Severed {
			continue
		}
		if checkItem(bp.Aug) || checkItem(bp.Item) {
			return true
		}
	}
	for _, it := range p.Inv {
		if checkItem(it) {
			return true
		}
	}
	return false
}

// wyPlanetTurn applies periodic planet trait effects (toxic_air poison outdoors every 10 turns,
// tidally_locked lighting/temperature).
func wyPlanetTurn(g *Game) {
	if g == nil || g.P == nil || g.Dead || g.Cur == nil {
		return
	}
	if locked, day := wyTidallyLockedLight(g); locked && g.Cur.Z == 0 {
		g.Cur.Dark = !day
	}
	if g.wyHasPlanetTrait("toxic_air") && g.Cur.Z == 0 && !g.Cur.Underwater {
		if g.Turn%10 == 0 && !g.wyHasToxicProtection() {
			hpBefore := g.P.HP
			g.hurt(g.P, 1, nil, "poison")
			if g.P.HP == hpBefore && g.P.HP > 0 {
				g.P.HP--
				if g.P.HP <= 0 {
					g.die("toxic air", nil)
				}
			}
			g.Msg("{{G|The toxic atmosphere sears your lungs! (-1 HP)}}")
		}
	}
}

func (g *Game) wyPlanetTurn() { wyPlanetTurn(g) }

// wyZoneHook spawns extra ancient relics and construct remnants on dead_ruin worlds.
func wyZoneHook(g *Game, zn *Zone, r *Rng) {
	if g == nil || zn == nil || r == nil {
		return
	}
	r = NewRng(g.Seed + ":wy:zone:" + zn.Key)
	if g.wyHasPlanetTrait("dead_ruin") && zn.Z == 0 && zn.Kind != "town" {
		// Spawn a construct remnant and an ancient relic/oddity
		lvl := maxi(1, zn.Level)
		con := g.Spawn("golem", lvl, r)
		con.Name = "ancient construct remnant"
		con.Set("dead_ruin_remnant")
		if x, y, ok := zn.FreeSpot(r, nil); ok {
			con.X, con.Y = x, y
			zn.Actors = append(zn.Actors, con)
		}
		relic := g.MakeOddity(r, clampi(lvl/2+1, 1, 4))
		relic.Name = "precursor ruin-core"
		if relic.Extra == nil {
			relic.Extra = map[string]int{}
		}
		relic.Extra["dead_ruin_relic"] = 1
		wyTagOffworldItem(relic, g.PlanetIndex())
		if x, y, ok := zn.FreeSpot(r, nil); ok {
			relic.X, relic.Y = x, y
			zn.Items = append(zn.Items, relic)
		}
	}
}

// ---- Rift Storms ----

// wyIsRiftStorm reports whether riftIdx is experiencing a seasonal rift storm (every 15-day window).
func wyIsRiftStorm(g *Game, riftIdx int) bool {
	if g == nil || g.W == nil {
		return false
	}
	seasonBucket := g.Day() / 15
	if s, ok := g.wy().ClosedRifts[riftIdx]; ok && s == seasonBucket {
		return true
	}
	h := HashStr(fmt.Sprintf("%s:wy:storm:%d:%d", g.Seed, riftIdx, seasonBucket))
	return h%3 == 0
}

func (g *Game) wyIsRiftStorm(riftIdx int) bool { return wyIsRiftStorm(g, riftIdx) }

// wyRiftClosed reports whether a rift is sealed for the current 15-day season by a rift storm.
func wyRiftClosed(g *Game, riftIdx int) bool {
	if g == nil || g.W == nil {
		return false
	}
	seasonBucket := g.Day() / 15
	if s, ok := g.wy().ClosedRifts[riftIdx]; ok && s == seasonBucket {
		return true
	}
	if !wyIsRiftStorm(g, riftIdx) {
		return false
	}
	h := HashStr(fmt.Sprintf("%s:wy:seal:%d:%d", g.Seed, riftIdx, seasonBucket))
	return h%2 == 0
}

func (g *Game) wyRiftClosed(riftIdx int) bool { return wyRiftClosed(g, riftIdx) }

// wyRiftStormTurn checks active rift storms and can spawn an off-world creature or seal the rift.
func wyRiftStormTurn(g *Game) {
	if g == nil || g.W == nil || g.Cur == nil || g.Dead {
		return
	}
	u := unUni(g.W)
	nRifts := maxi(1, len(u.Rifts))
	seasonBucket := g.Day() / 15
	for ri := 0; ri < nRifts; ri++ {
		if !wyIsRiftStorm(g, ri) {
			continue
		}
		// Seal rift for the season if storm peaks
		if HashStr(fmt.Sprintf("%s:wy:seal:%d:%d", g.Seed, ri, seasonBucket))%2 == 0 {
			g.wy().ClosedRifts[ri] = seasonBucket
		}
		// Spawn an exotic creature displaced from another planet if none is in the zone yet
		hasDisplaced := false
		for _, a := range g.Cur.Actors {
			if a.Has("rift_displaced") {
				hasDisplaced = true
				break
			}
		}
		if !hasDisplaced {
			g.wySpawnRiftCreature(g.Cur, ri)
		}
	}
}

func (g *Game) wyRiftStormTurn() { wyRiftStormTurn(g) }

func (g *Game) wySpawnRiftCreature(zn *Zone, riftIdx int) *Actor {
	if g == nil || zn == nil {
		return nil
	}
	r := NewRng(fmt.Sprintf("%s:wy:riftmob:%s:%d:%d", g.Seed, zn.Key, riftIdx, g.Day()/15))
	tpls := []string{"stalker", "crystalgolem", "wraith", "moth", "drake", "horror"}
	tpl := Pick(r, tpls)
	a := g.Spawn(tpl, maxi(1, g.P.Level+1), r)
	originPlanet := "another sphere"
	if u := unUni(g.W); len(u.Planets) > 1 {
		otherIdx := (g.PlanetIndex() + 1) % len(u.Planets)
		originPlanet = u.planetLabel(otherIdx)
	}
	a.Name = "rift-displaced " + a.Name
	a.Desc = fmt.Sprintf("Hurl through a howling rift storm from %s.", originPlanet)
	a.Set("rift_displaced")
	if x, y, ok := zn.FreeSpot(r, nil); ok {
		a.X, a.Y = x, y
	} else {
		a.X, a.Y = clampi(g.P.X+2, 1, MW-2), clampi(g.P.Y, 1, MH-2)
	}
	zn.Actors = append(zn.Actors, a)
	g.Msg(fmt.Sprintf("{{M|A rift storm crackles overhead — %s is hurled through from %s!}}", g.the(a), originPlanet))
	return a
}

// ---- Star-Map Screen (Mode == "starmap") ----

func OpenStarMap(g *Game) {
	if g != nil {
		g.OpenStarMap()
	}
}

func (g *Game) OpenStarMap() {
	g.Mode = "starmap"
	g.Sel = g.PlanetIndex()
}

func (g *Game) wyStarMapLines() []string {
	if g == nil || g.W == nil {
		return nil
	}
	u := unUni(g.W)
	var lines []string
	lines = append(lines, "{{Y|=== Star-Map of the Spheres ===}}")
	for i, pw := range u.Planets {
		marker := "  "
		if i == g.PlanetIndex() {
			marker = "{{G|[HERE]}} "
		}
		traits := wyPlanetTrait(g.W, i)
		if ov, ok := g.wy().TraitOverride[i]; ok && len(ov) > 0 {
			traits = ov
		}
		pName := nonEmpty(pw.Name, pw.Planet)
		if pName == "" {
			pName = fmt.Sprintf("Planet %d", i+1)
		}
		pKind := nonEmpty(pw.PlanetKind, "terrestrial")
		lines = append(lines, fmt.Sprintf("%s%d. {{C|%s}} (%s) — Traits: {{W|%s}}", marker, i+1, pName, pKind, strings.Join(traits, ", ")))
	}
	lines = append(lines, "")
	lines = append(lines, "{{M|Active Rift Links & Seasonal Weather:}}")
	if len(u.Rifts) == 0 {
		if wyIsRiftStorm(g, 0) {
			lines = append(lines, "  Local Astral Tide: {{M|[RIFT STORM ACTIVE]}}")
		} else {
			lines = append(lines, "  No charted interplanetary gates on this sphere.")
		}
	} else {
		for ri, rf := range u.Rifts {
			status := "{{G|[OPEN]}}"
			if wyRiftClosed(g, ri) {
				status = "{{R|[SEALED BY STORM]}}"
			} else if wyIsRiftStorm(g, ri) {
				status = "{{M|[RIFT STORM]}}"
			}
			aName := u.planetLabel(clampi(rf.A.Planet, 0, len(u.Planets)-1))
			bName := u.planetLabel(clampi(rf.B.Planet, 0, len(u.Planets)-1))
			lines = append(lines, fmt.Sprintf("  Ø %s: %s <-> %s %s", rf.Name, aName, bName, status))
		}
	}
	lines = append(lines, pltStarMapLines(g)...) // planet traits (planettraits.go)
	return lines
}

func wyStarMapKey(g *Game, k string) {
	u := unUni(g.W)
	n := maxi(1, len(u.Planets))
	switch k {
	case "Escape", "q", "Q", "Enter", " ":
		g.Mode = "main"
	case "Up", "N8", "k":
		g.Sel = clampi(g.Sel-1, 0, n-1)
	case "Down", "N2", "j":
		g.Sel = clampi(g.Sel+1, 0, n-1)
	case "s":
		pltScanSelected(g)
	}
}

func wyDrawStarMap(g *Game) {
	s := g.sc
	lines := g.wyStarMapLines()
	h := clampi(len(lines)+4, 10, MH-2)
	y0 := maxi(1, (MH-h)/2)
	s.Box(4, y0, 72, h, CMagenta, "Star-Map of the Spheres")
	for i, ln := range lines {
		if i+2 >= h-1 {
			break
		}
		s.TextB(6, y0+1+i, ln, CGray, CBgDark, 74)
	}
	s.TextB(6, y0+h-1, "[Up/Down inspect spheres, Esc/q close]", CYellow, CBgDark, 74)
}

// ---- Off-World Trade Goods ----

// wyTagOffworldItem marks an item with its origin planet ("planet_origin:<id>").
func wyTagOffworldItem(it *Item, planetID int) {
	if it == nil {
		return
	}
	if it.Extra == nil {
		it.Extra = map[string]int{}
	}
	// Clean any old planet_origin:<id> tag first
	for k := range it.Extra {
		if strings.HasPrefix(k, "planet_origin:") {
			delete(it.Extra, k)
		}
	}
	tag := fmt.Sprintf("planet_origin:%d", planetID)
	it.Extra[tag] = 1
	it.Extra["planet_origin"] = planetID + 1
	if !strings.Contains(it.Lore, tag) {
		if it.Lore == "" {
			it.Lore = "Origin marker: " + tag
		} else {
			it.Lore += " (" + tag + ")"
		}
	}
}

func wyItemOriginPlanet(it *Item) (int, bool) {
	if it == nil {
		return 0, false
	}
	if it.Extra != nil {
		for k, v := range it.Extra {
			if v > 0 && strings.HasPrefix(k, "planet_origin:") {
				if id, err := strconv.Atoi(strings.TrimPrefix(k, "planet_origin:")); err == nil {
					return id, true
				}
			}
		}
		if v := it.Extra["planet_origin"]; v > 0 {
			return v - 1, true
		}
	}
	if idx := strings.Index(it.Lore, "planet_origin:"); idx >= 0 {
		rest := it.Lore[idx+len("planet_origin:"):]
		end := 0
		for end < len(rest) && (rest[end] == '-' || (rest[end] >= '0' && rest[end] <= '9')) {
			end++
		}
		if end > 0 {
			if id, err := strconv.Atoi(rest[:end]); err == nil {
				return id, true
			}
		}
	}
	return 0, false
}

// wyOffworldPriceMul returns 2.0 when an item originates from another planet than g.PlanetIndex().
func (g *Game) wyOffworldPriceMul(it *Item) float64 {
	if g == nil || it == nil {
		return 1.0
	}
	orig, ok := wyItemOriginPlanet(it)
	if !ok {
		return 1.0
	}
	if orig != g.PlanetIndex() {
		return 2.0
	}
	return 1.0
}

func wyOffworldPriceMul(g *Game, it *Item) float64 { return g.wyOffworldPriceMul(it) }

// ---- 2. Libraries, Rumours, Dreams, Prophecy, Companion Arcs & Epilogues ----

// wyReadLibraryBook reads a historical event from g.W.Events, logs it, awards +15 XP, and
// reveals an unvisited ruin/dungeon parasang on the world map.
func (g *Game) wyReadLibraryBook(shelfX, shelfY int) string {
	if g == nil || g.W == nil {
		return ""
	}
	st := g.wy()
	zKey := "world"
	if g.Cur != nil {
		zKey = g.Cur.Key
	}
	shelfKey := fmt.Sprintf("%s:%d,%d", zKey, shelfX, shelfY)
	r := NewRng(fmt.Sprintf("%s:wy:lib:%s:%d", g.Seed, shelfKey, len(st.BooksRead)))

	var evText string
	var evYear int
	if len(g.W.Events) > 0 {
		ev := g.W.Events[r.Int(0, len(g.W.Events)-1)]
		evYear = ev.Y
		evText = ev.Text
	} else {
		evYear = maxi(1, g.W.Year-r.Int(10, 120))
		evText = fmt.Sprintf("The chroniclers of %s recorded the fall of an old watchtower in the border marches.", g.unPlanetName())
	}

	// Pick an unvisited parasang to reveal (preferring dead burgs / ruins / monsters / wonders)
	targetP := -1
	for _, b := range g.W.Burgs {
		if !b.Alive && !g.Known[b.P] && !st.RevealedSites[b.P] {
			targetP = b.P
			break
		}
	}
	if targetP < 0 {
		for _, m := range g.W.Monsters {
			if m.P >= 0 && m.P < g.W.W*g.W.H && !g.Known[m.P] && !st.RevealedSites[m.P] {
				targetP = m.P
				break
			}
		}
	}
	if targetP < 0 {
		for tries := 0; tries < 50; tries++ {
			cand := r.Int(0, maxi(1, g.W.W*g.W.H-1))
			if !st.RevealedSites[cand] {
				targetP = cand
				break
			}
		}
	}
	if targetP < 0 {
		targetP = clampi((g.WY*g.W.W+g.WX+7)%(maxi(1, g.W.W*g.W.H)), 0, maxi(0, g.W.W*g.W.H-1))
	}

	g.Known[targetP] = true
	st.RevealedSites[targetP] = true
	st.BooksRead[shelfKey] = true
	g.gainXP(15)

	rx, ry := targetP%maxi(1, g.W.W), targetP/maxi(1, g.W.W)
	passage := fmt.Sprintf("In a dusty tome you read: \"In the year %d, %s\" Margin notes mark a forgotten ruin at (%d, %d).", evYear, evText, rx, ry)
	g.Msg("{{c|" + passage + " (+15 XP)}}")
	return passage
}

func (g *Game) wyRevealedSites() map[int]bool {
	return g.wy().RevealedSites
}

// wyRumour generates a rumour from an NPC. ~35% of rumours are twisted or out-of-date
// (accurate == false), flagging that parasang in TwistedRumours for a rival-slain quarry or ambush.
func (g *Game) wyRumour(npc *Actor) (text string, accurate bool) {
	if g == nil || g.W == nil {
		return "", true
	}
	st := g.wy()
	st.RumourSeq++
	npcID := 0
	npcName := "a local traveller"
	if npc != nil {
		npcID = npc.ID
		npcName = npc.Name
		g.wyRecordMetNPC(npc)
	}
	r := NewRng(fmt.Sprintf("%s:wy:rumour:%d:%d", g.Seed, npcID, st.RumourSeq))
	// Deterministic ~35% twisted rate (guaranteed to hit both true and false across small sequences)
	twisted := (st.RumourSeq%3 == 0) || (st.RumourSeq%3 != 1 && r.Chance(0.35))
	accurate = !twisted

	// Pick a nearby parasang
	wx := clampi(g.WX+r.Int(-4, 4), 0, maxi(0, g.W.W-1))
	wy := clampi(g.WY+r.Int(-4, 4), 0, maxi(0, g.W.H-1))
	if wx == g.WX && wy == g.WY && g.W.W > 2 {
		wx = (wx + 1) % g.W.W
	}

	subject := "a fierce beast's lair"
	if len(g.W.Monsters) > 0 {
		m := g.W.Monsters[r.Int(0, len(g.W.Monsters)-1)]
		subject = m.Name
		if m.P >= 0 && m.P < g.W.W*g.W.H {
			wx, wy = m.P%g.W.W, m.P/g.W.W
		}
	} else if len(g.W.Outlaws) > 0 {
		o := g.W.Outlaws[r.Int(0, len(g.W.Outlaws)-1)]
		subject = o.Name
		if o.P >= 0 && o.P < g.W.W*g.W.H {
			wx, wy = o.P%g.W.W, o.P/g.W.W
		}
	}

	if accurate {
		p := g.W.Idx(wx, wy)
		g.Known[p] = true
		text = fmt.Sprintf("%s whispers: \"Word is true — %s still lurks at (%d, %d) bearing rich spoils.\"", npcName, subject, wx, wy)
	} else {
		kind := "ambush"
		if st.RumourSeq%2 == 0 {
			kind = "slain"
		}
		key := fmt.Sprintf("%d,%d", wx, wy)
		st.TwistedRumours[key] = kind
		st.TwistedMem[[2]int{wx, wy}] = kind
		text = fmt.Sprintf("%s mutters: \"I heard %s is unguarded at (%d, %d)... though the tale may be old.\"", npcName, subject, wx, wy)
	}
	g.Msg(text)
	return text, accurate
}

func (g *Game) wyTwistedRumours() map[[2]int]string {
	return g.wy().TwistedMem
}

// wyOnEnterZone triggers twisted rumour surprises when arriving at a flagged parasang,
// and tracks notables met and rift trips.
func wyOnEnterZone(g *Game, zn *Zone) {
	if g == nil || zn == nil {
		return
	}
	g.wyEnsureProphecy()
	g.wyUpdateProphecyProgress()
	for _, a := range zn.Actors {
		if a.Proper || a.SimPer > 0 || a.Role == "elder" || a.Role == "notable" {
			g.wyRecordMetNPC(a)
		}
	}
	st := g.wy()
	coord := [2]int{zn.WX, zn.WY}
	key := fmt.Sprintf("%d,%d", zn.WX, zn.WY)
	kind, ok := st.TwistedMem[coord]
	if !ok {
		kind, ok = st.TwistedRumours[key]
	}
	if ok && zn.Z == 0 {
		delete(st.TwistedMem, coord)
		delete(st.TwistedRumours, key)
		r := NewRng(fmt.Sprintf("%s:wy:twist:%s", g.Seed, zn.Key))
		if kind == "slain" {
			corpse := &Item{
				ID:     g.newID(),
				Kind:   "corpse",
				Name:   "slain quarry carcass",
				Glyph:  '%',
				F:      CDarkRed,
				X:      clampi(g.P.X+1, 1, MW-2),
				Y:      clampi(g.P.Y, 1, MH-2),
				Weight: 10,
				Lore:   "Slain days ago by a rival adventurer; the rumour was out of date.",
			}
			zn.Items = append(zn.Items, corpse)
			g.Msg("{{y|The rumour was out of date — a rival adventurer already slew the quarry here!}}")
		} else {
			foe := g.Spawn("bandit", maxi(1, g.P.Level), r)
			foe.Name = "rumour-bait ambusher"
			foe.AI = "hostile"
			foe.Angry = true
			foe.Set("twisted_ambush")
			foe.X, foe.Y = clampi(g.P.X+2, 1, MW-2), clampi(g.P.Y, 1, MH-2)
			zn.Actors = append(zn.Actors, foe)
			g.Msg("{{R|The rumour was a trap! Ambushers spring from hiding!}}")
		}
	}
}

// ---- Historical Dreams ----

// wyDream replays a vivid historical dream from g.W.Events and hints at the compass direction
// and coordinates of a relic or burial site.
func (g *Game) wyDream() string {
	if g == nil || g.W == nil {
		return ""
	}
	st := g.wy()
	st.DreamSeq++
	r := NewRng(fmt.Sprintf("%s:wy:dream:%d", g.Seed, st.DreamSeq))

	var evPassage string
	if len(g.W.Events) > 0 {
		ev := g.W.Events[r.Int(0, len(g.W.Events)-1)]
		evPassage = fmt.Sprintf("In a dream of year %d, you witness: %s", ev.Y, ev.Text)
	} else {
		evPassage = fmt.Sprintf("In a vivid dream, ancient banners fall beneath the moons of %s.", g.unPlanetName())
	}

	targetP := -1
	targetName := "an ancient burial relic"
	if len(g.W.Artifacts) > 0 {
		art := g.W.Artifacts[r.Int(0, len(g.W.Artifacts)-1)]
		targetName = art.Name
		if art.P >= 0 && art.P < g.W.W*g.W.H {
			targetP = art.P
		}
	}
	if targetP < 0 {
		if ix := buIdx(g.W); ix != nil && len(ix.sites) > 0 {
			s := ix.sites[r.Int(0, len(ix.sites)-1)]
			targetName = s.Name
			targetP = s.P
		}
	}
	if targetP < 0 {
		targetP = r.Int(0, maxi(1, g.W.W*g.W.H-1))
	}

	tx, ty := targetP%maxi(1, g.W.W), targetP/maxi(1, g.W.W)
	dir := wyCompassDir(g.WX, g.WY, tx, ty)
	hint := fmt.Sprintf("%s A spectral voice whispers that %s lies to the %s at (%d, %d).", evPassage, targetName, dir, tx, ty)
	st.DreamHint = hint
	st.DreamTargetP = targetP
	g.Known[targetP] = true
	g.Msg("{{m|" + hint + "}}")
	return hint
}

func (g *Game) wyDreamHint() string { return g.wy().DreamHint }

func wyOnRest(g *Game) string {
	if g == nil {
		return ""
	}
	return g.wyDream()
}

func wyCompassDir(x0, y0, x1, y1 int) string {
	dx, dy := x1-x0, y1-y0
	if dx == 0 && dy == 0 {
		return "here"
	}
	var ns, ew string
	if dy < 0 {
		ns = "north"
	} else if dy > 0 {
		ns = "south"
	}
	if dx > 0 {
		ew = "east"
	} else if dx < 0 {
		ew = "west"
	}
	if ns != "" && ew != "" {
		return ns + "-" + ew
	}
	return ns + ew
}

// ---- Start-of-Run Prophecy ----

// wyEnsureProphecy initializes the 3-verse prophecy at the start of the run.
func (g *Game) wyEnsureProphecy() *wyProphecy {
	if g == nil {
		return nil
	}
	st := g.wy()
	if st.Prophecy != nil {
		return st.Prophecy
	}
	planet := g.unPlanetName()
	if planet == "" {
		planet = "the mortal sphere"
	}
	st.Prophecy = &wyProphecy{
		Verses: [3]string{
			fmt.Sprintf("Verse I (The Beast): Beneath the skies of %s, the Wanderer shall face the Great Quarry in blood or mercy.", planet),
			"Verse II (The Rift): When the astral gates shimmer, the Wanderer shall choose between the home-soil and the outer spheres.",
			"Verse III (The Crown): Before the final reckoning, thrones shall weigh the Wanderer as champion, outcast, or sovereign.",
		},
		OmenBeast: "unfulfilled",
		OmenRift:  "unfulfilled",
		OmenCrown: "unfulfilled",
	}
	return st.Prophecy
}

func (g *Game) wyUpdateProphecyProgress() {
	if g == nil || g.P == nil {
		return
	}
	pr := g.wyEnsureProphecy()
	if g.Kills > 0 && pr.OmenBeast == "unfulfilled" {
		pr.OmenBeast = "slain"
	}
	if g.W != nil {
		u := unUni(g.W)
		if g.unSt().Trips > 0 || g.PlanetIndex() != u.Start {
			pr.OmenRift = "crossed"
		}
	}
	bestRep, worstRep := 0, 0
	for k, v := range g.Rep {
		if strings.HasPrefix(k, "state:") {
			if v > bestRep {
				bestRep = v
			}
			if v < worstRep {
				worstRep = v
			}
		}
	}
	if bestRep >= 100 {
		pr.OmenCrown = "champion"
	} else if worstRep <= -100 {
		pr.OmenCrown = "outcast"
	}
}

func wyOnKill(g *Game, d, src *Actor) {
	if g == nil || d == nil {
		return
	}
	if src == g.P || (src != nil && src.AI == "follower") {
		pr := g.wyEnsureProphecy()
		if d.Boss || d.SimMon > 0 || d.Level >= 1 {
			pr.OmenBeast = "slain"
		}
	}
}

// wyProphecyResolution resolves the 3 omens dynamically from the player's deeds and feeds the endgame.
func (g *Game) wyProphecyResolution() string {
	if g == nil {
		return ""
	}
	pr := g.wyEnsureProphecy()
	g.wyUpdateProphecyProgress()

	var parts []string
	switch pr.OmenBeast {
	case "slain":
		parts = append(parts, "The Omen of the Beast was fulfilled in blood: the great quarry fell before your blade.")
	default:
		pr.OmenBeast = "spared"
		parts = append(parts, "The Omen of the Beast resolved in stillness: you walked the wilds without slaying the ancient quarry.")
	}

	switch pr.OmenRift {
	case "crossed":
		parts = append(parts, "The Omen of the Rift was fulfilled across the stars: you stepped through the astral gates onto foreign spheres.")
	default:
		pr.OmenRift = "earthbound"
		parts = append(parts, "The Omen of the Rift remained earthbound: you kept faithful vigil upon your home world.")
	}

	switch pr.OmenCrown {
	case "champion":
		parts = append(parts, "The Omen of the Crown hailed you as Champion of the Realms, honoured in the courts of the living.")
	case "outcast":
		parts = append(parts, "The Omen of the Crown marked you as the Dread Outcast, feared by magistrate and monarch alike.")
	default:
		pr.OmenCrown = "wanderer"
		parts = append(parts, "The Omen of the Crown named you the Unbound Wanderer, beholden to neither throne nor banner.")
	}
	pr.Resolved = true
	return strings.Join(parts, " ")
}

// ---- Companion Arcs & Lethal Blow Sacrifice ----

func (g *Game) wyCompanionArc(c *Actor) *wyCompArc {
	if g == nil || c == nil {
		return nil
	}
	st := g.wy()
	arc, ok := st.CompArcs[c.ID]
	if !ok {
		arc = &wyCompArc{CompID: c.ID, Name: c.Name}
		st.CompArcs[c.ID] = arc
	}
	return arc
}

// wyAdvanceCompanionArc progresses a companion through "trust", "crisis", and "resolution"
// ("bond", "sacrifice", "departure", or "betrayal" if Loyalty < 15).
func (g *Game) wyAdvanceCompanionArc(c *Actor, event string) string {
	if g == nil || c == nil {
		return ""
	}
	arc := g.wyCompanionArc(c)
	arc.Name = c.Name
	if event != "" {
		arc.Events = append(arc.Events, event)
	}
	loyalty := 60
	if c.Comp != nil {
		loyalty = c.Comp.Loyalty
	}

	// Low loyalty triggers betrayal
	if loyalty < 15 || event == "betrayal" {
		arc.Stage = "resolution"
		arc.Outcome = "betrayal"
		c.AI = "hostile"
		c.Faction = "outlaws"
		c.Angry = true
		c.Order = ""
		g.Msg(fmt.Sprintf("{{R|%s turns on you in bitter betrayal! \"I follow you no longer!\"}}", c.Name))
		return "betrayal"
	}

	switch event {
	case "trust":
		arc.Stage = "trust"
		g.addLoyalty(c, 10)
		g.Msg(fmt.Sprintf("{{G|%s confides in you, forging a bond of trust.}}", c.Name))
		return arc.Stage
	case "crisis":
		arc.Stage = "crisis"
		g.Msg(fmt.Sprintf("{{Y|%s faces a moment of crisis from their past.}}", c.Name))
		return arc.Stage
	case "bond", "sacrifice", "departure":
		arc.Stage = "resolution"
		arc.Outcome = event
		if event == "departure" {
			c.AI = "town"
			c.Order = ""
			g.Msg(fmt.Sprintf("{{y|%s bids you farewell to walk their own road.}}", c.Name))
		} else {
			g.addLoyalty(c, 15)
			g.Msg(fmt.Sprintf("{{G|%s swears an unbreakable %s with you.}}", c.Name, event))
		}
		return arc.Outcome
	}

	// Automatic stage progression when event is generic
	switch arc.Stage {
	case "":
		arc.Stage = "trust"
		g.addLoyalty(c, 10)
		return "trust"
	case "trust":
		arc.Stage = "crisis"
		return "crisis"
	default:
		arc.Stage = "resolution"
		if loyalty < 35 {
			arc.Outcome = "departure"
			c.AI = "town"
			c.Order = ""
		} else if strings.Contains(strings.ToLower(event), "sac") {
			arc.Outcome = "sacrifice"
		} else {
			arc.Outcome = "bond"
		}
		return arc.Outcome
	}
}

// wyCheckCompanionSacrifice checks if lethalDmg would kill g.P and a bonded/sacrificial companion
// intercepts the fatal blow, saving the player (HP = max(1, MaxHP/3)) at the cost of their own life.
func (g *Game) wyCheckCompanionSacrifice(lethalDmg int) bool {
	if g == nil || g.P == nil || g.P.HP-lethalDmg > 0 {
		return false
	}
	var cands []*Actor
	if g.Cur != nil {
		for _, a := range g.Cur.Actors {
			if a != nil && a != g.P && a.HP > 0 && (a.AI == "follower" || a.Comp != nil) {
				cands = append(cands, a)
			}
		}
	}
	for _, c := range cands {
		arc := g.wyCompanionArc(c)
		if arc.Outcome == "bond" || arc.Outcome == "sacrifice" {
			arc.Stage = "resolution"
			arc.Outcome = "sacrifice"
			arc.Sacrificed = true
			c.HP = 0
			g.P.HP = maxi(1, g.P.MaxHP/3)
			g.Dead = false
			g.Msg(fmt.Sprintf("{{R|%s leaps in front of the fatal blow, sacrificing their life to save you!}}", c.Name))
			g.deed("companion", fmt.Sprintf("%s gave their life to shield %s from a mortal blow.", c.Name, g.P.Name), 0, 0)
			return true
		}
	}
	return false
}

// ---- NPC Tracking & Epilogues ----

func (g *Game) wyRecordMetNPC(a *Actor) {
	if g == nil || a == nil || a == g.P || a.ID <= 0 {
		return
	}
	st := g.wy()
	town := ""
	if g.Cur != nil {
		town = g.Cur.Name
	}
	st.MetNPCs[a.ID] = &wyMetNPC{
		ID:      a.ID,
		Name:    a.Name,
		Role:    a.Role,
		Title:   a.Title,
		Faction: a.Faction,
		Town:    town,
		Alive:   a.HP > 0,
	}
}

// wyBuildEpilogues constructs a personalized epilogue line for every notable NPC met and every companion.
func (g *Game) wyBuildEpilogues() []string {
	if g == nil {
		return nil
	}
	st := g.wy()
	for _, a := range g.localCompanions() {
		g.wyCompanionArc(a)
	}
	for _, wc := range g.waitingCompanions() {
		g.wyCompanionArc(wc.A)
	}
	if g.Cur != nil {
		for _, a := range g.Cur.Actors {
			if a.Proper || a.SimPer > 0 || a.Role == "elder" || a.Role == "notable" {
				g.wyRecordMetNPC(a)
			}
		}
	}

	var out []string
	// Companion epilogues first, sorted by ID
	var compIDs []int
	for id := range st.CompArcs {
		compIDs = append(compIDs, id)
	}
	sort.Ints(compIDs)
	seenNames := map[string]bool{}
	for _, id := range compIDs {
		arc := st.CompArcs[id]
		if arc == nil || arc.Name == "" {
			continue
		}
		seenNames[arc.Name] = true
		switch {
		case arc.Sacrificed || arc.Outcome == "sacrifice":
			out = append(out, fmt.Sprintf("%s is remembered in song for laying down their life so that %s might live.", arc.Name, g.P.Name))
		case arc.Outcome == "betrayal":
			out = append(out, fmt.Sprintf("%s vanished into the borderlands after betraying %s, hunted by old oaths.", arc.Name, g.P.Name))
		case arc.Outcome == "departure":
			out = append(out, fmt.Sprintf("%s returned home after parting ways with %s and kept a quiet hearth.", arc.Name, g.P.Name))
		case arc.Outcome == "bond":
			out = append(out, fmt.Sprintf("%s stood beside %s to the very end, their sworn bond unbroken.", arc.Name, g.P.Name))
		default:
			out = append(out, fmt.Sprintf("%s carried the memory of their travels with %s through the years that followed.", arc.Name, g.P.Name))
		}
	}

	// Met NPCs epilogues, sorted by ID
	var npcIDs []int
	for id := range st.MetNPCs {
		npcIDs = append(npcIDs, id)
	}
	sort.Ints(npcIDs)
	for _, id := range npcIDs {
		m := st.MetNPCs[id]
		if m == nil || m.Name == "" || seenNames[m.Name] {
			continue
		}
		seenNames[m.Name] = true
		place := nonEmpty(m.Town, "their settlement")
		if !m.Alive || g.Slain[m.ID] {
			out = append(out, fmt.Sprintf("%s of %s did not survive the tumultuous years of %s's journey.", m.Name, place, g.P.Name))
			continue
		}
		rep := g.Rep[m.Faction]
		switch {
		case rep >= 100:
			out = append(out, fmt.Sprintf("%s in %s prospered and often spoke warmly of %s's deeds.", m.Name, place, g.P.Name))
		case rep <= -100:
			out = append(out, fmt.Sprintf("%s in %s barred their shutters whenever %s's name was spoken.", m.Name, place, g.P.Name))
		default:
			out = append(out, fmt.Sprintf("%s continued their work as %s in %s, telling travellers of the day %s passed through.", m.Name, nonEmpty(m.Role, "citizen"), place, g.P.Name))
		}
	}

	if pr := g.wyProphecyResolution(); pr != "" {
		out = append(out, "Prophecy: "+pr)
	}
	return out
}

// ---- Talk Hooks ----

func wyTalkOptions(g *Game, a *Actor) []Option {
	if g == nil || a == nil {
		return nil
	}
	g.wyRecordMetNPC(a)
	var opts []Option
	if a.Role == "scholar" || a.Role == "bookseller" || a.Role == "elder" || a.Role == "priest" || a.Role == "astrologer" {
		opts = append(opts, Option{
			Text: "{{M|Consult the Star-Map of the Spheres.}}",
			Do:   func() { g.OpenStarMap() },
		})
		opts = append(opts, Option{
			Text: "{{c|Study a historical chronicle from the shelves.}}",
			Do: func() {
				msg := g.wyReadLibraryBook(a.X, a.Y)
				if g.Talk == nil {
					g.Talk = a
				}
				g.talkMenu(msg)
			},
		})
	}
	opts = append(opts, Option{
		Text: "Have you heard any rumours of nearby sites?",
		Do: func() {
			txt, _ := g.wyRumour(a)
			if g.Talk == nil {
				g.Talk = a
			}
			g.talkMenu(txt)
		},
	})
	return opts
}
