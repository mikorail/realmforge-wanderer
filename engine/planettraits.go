package engine

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"unicode"
)

// Planet traits and star-map extras (prefix pt). Adds ten more planet traits (high gravity,
// aether-rich, split sky, ocean, storm, fungal bloom, frozen, war-ruined, sacred, trade hub, plague)
// chosen deterministically per planet from the universe seed, on top of the four in world_story.go.
// Traits are a pure function of the seed, so the only saved state (g.ext("plt"), global) is what you
// have visited, scanned and written in the planet journal. Effects live in turn/zone/enter/side hooks;
// gear that counters a trait is detected by name/kind/effect tokens (pltCounterTokens).
//
// Hooks other files can call (see the final report): pltStarMapLines, pltRiftLines, pltScanSelected,
// pltTravelMul, pltFallDamage, pltPriceMod, pltSpellBoost, pltJournalLines.

type pltEntry struct {
	Name    string   `json:"n"`
	Day     int      `json:"d"`
	Date    string   `json:"dt,omitempty"`
	Traits  []string `json:"t,omitempty"`
	Places  []string `json:"p,omitempty"`
	Scanned bool     `json:"sc,omitempty"`
}

type pltState struct {
	Scanned map[int]bool      `json:"sc,omitempty"`
	Journal map[int]*pltEntry `json:"jn,omitempty"`
	LastX   int               `json:"lx,omitempty"`
	LastY   int               `json:"ly,omitempty"`
	Idle    int               `json:"idle,omitempty"`
	Signal  map[int]string    `json:"sig,omitempty"`
	Scans   int               `json:"scans,omitempty"`
}

func (g *Game) plt() *pltState {
	s := g.ext("plt", func() any { return &pltState{} }).(*pltState)
	if s.Scanned == nil {
		s.Scanned = map[int]bool{}
	}
	if s.Journal == nil {
		s.Journal = map[int]*pltEntry{}
	}
	if s.Signal == nil {
		s.Signal = map[int]string{}
	}
	return s
}

func init() {
	turnHooks = append(turnHooks, pltTurn)
	zoneHooks = append(zoneHooks, pltZone)
	enterHooks = append(enterHooks, pltEnter)
	sideHooks = append(sideHooks, pltSide)
	modeKeys["pltscan"] = pltKey
	modeDraws["pltscan"] = pltDraw
	if _, taken := mainKeys["F11"]; !taken {
		mainKeys["F11"] = func(g *Game) { g.Mode, g.Sel, g.Tab = "pltscan", g.PlanetIndex(), 0 }
	}
}

// ---- choosing traits ----

var pltCache sync.Map // planet seed -> []string

// pltTraits is the planet's new traits (1-2), a pure function of the universe and planet seed.
func pltTraits(w *WorldData, id int) []string {
	if w == nil {
		return nil
	}
	u := unUni(w)
	if id < 0 || id >= len(u.Planets) || u.Planets[id] == nil {
		id = 0
	}
	seed := unSeed(u, id)
	if v, ok := pltCache.Load(seed + "|" + fmt.Sprint(id)); ok {
		return v.([]string)
	}
	pw := u.Planets[id]
	kind := strings.ToLower(pw.PlanetKind + " " + pw.Name + " " + pw.Planet)
	type sc struct {
		id    string
		score int
	}
	var all []sc
	for _, d := range pltTraitDefs {
		if id == u.Start && d.Harsh {
			continue
		}
		s := int(HashStr(seed+":pt:trait:"+d.ID) % 100)
		for _, k := range d.Kinds {
			if strings.Contains(kind, k) {
				s += 70
				break
			}
		}
		all = append(all, sc{d.ID, s})
	}
	sort.SliceStable(all, func(i, j int) bool {
		if all[i].score != all[j].score {
			return all[i].score > all[j].score
		}
		return all[i].id < all[j].id
	})
	out := []string{all[0].id}
	if len(all) > 1 && HashStr(seed+":pt:second")%100 < 40 {
		out = append(out, all[1].id)
	}
	pltCache.Store(seed+"|"+fmt.Sprint(id), out)
	return out
}

func (g *Game) pltTraits() []string {
	if g == nil || g.W == nil {
		return nil
	}
	return pltTraits(g.W, g.PlanetIndex())
}

func (g *Game) pltHas(id string) bool {
	for _, t := range g.pltTraits() {
		if t == id {
			return true
		}
	}
	return false
}

func pltName(id string) string {
	if d := pltTraitByID[id]; d != nil {
		return d.Name
	}
	return id
}

// ---- gear that counters a trait ----

func pltWordHit(name, tok string) bool {
	if strings.Contains(tok, " ") || strings.Contains(tok, "-") || len(tok) > 4 {
		if strings.Contains(tok, " ") || strings.Contains(tok, "-") {
			return strings.Contains(name, tok)
		}
		for _, w := range strings.FieldsFunc(name, func(r rune) bool { return !unicode.IsLetter(r) }) {
			if strings.HasPrefix(w, tok) {
				return true
			}
		}
		return false
	}
	for _, w := range strings.FieldsFunc(name, func(r rune) bool { return !unicode.IsLetter(r) }) {
		if w == tok || w == tok+"s" {
			return true
		}
	}
	return false
}

func pltItemHits(it *Item, toks []string, kinds []string) bool {
	if it == nil {
		return false
	}
	text := strings.ToLower(it.Name + " " + it.Effect + " " + it.Cat + " " + it.Look)
	for _, t := range toks {
		if pltWordHit(text, t) {
			return true
		}
		if it.Extra != nil && it.Extra[t] > 0 {
			return true
		}
	}
	for _, k := range kinds {
		if it.Kind == k {
			return true
		}
	}
	return false
}

// pltCounter reports whether the player is protected against trait id, and by what.
func (g *Game) pltCounter(id string) (bool, string) {
	p := g.P
	if p == nil {
		return false, ""
	}
	toks, kinds := pltCounterTokens[id], pltCounterKinds[id]
	for _, bp := range p.Body {
		if bp == nil || bp.Severed {
			continue
		}
		if pltItemHits(bp.Item, toks, kinds) {
			return true, bp.Item.Name
		}
		if pltItemHits(bp.Aug, toks, nil) {
			return true, bp.Aug.Name
		}
	}
	for _, it := range p.Inv {
		if pltItemHits(it, toks, kinds) {
			return true, it.Name
		}
	}
	for _, m := range pltCounterMeals[id] {
		if p.Meal[m] > 0 {
			return true, "a " + m + " meal"
		}
	}
	switch id {
	case "plague_world", "fungal_bloom":
		if p.Has("poison_resist") || p.St("poison_resist") > 0 || p.Has("gasmask") || p.Has("filter") {
			return true, "poison resistance"
		}
		if g.wyHasToxicProtection() && id == "fungal_bloom" {
			return true, "a filter"
		}
	case "frozen":
		if p.Pow("frost") > 0 || p.Has("cold_resist") {
			return true, "cold resistance"
		}
	case "high_gravity":
		if p.Pow("levitate") > 0 || p.Pow("flight") > 0 || p.Has("fly") {
			return true, "flight"
		}
	}
	return false, ""
}

// pltPrepLine describes how ready the player is for one trait (empty for boons).
func (g *Game) pltPrepLine(id string) string {
	d := pltTraitByID[id]
	if d == nil || d.Counter == "" {
		return ""
	}
	if ok, by := g.pltCounter(id); ok {
		return "{{G|ready}} (" + by + ")"
	}
	return "{{R|unprepared}}: bring " + d.Counter
}

// ---- the turn hook: environmental effects ----

func pltOutdoors(g *Game) bool {
	z := g.Cur
	if z == nil || z.Z != 0 || z.Underwater || z.Sky || z.Kind == "ship" || z.Kind == "town" {
		return false
	}
	return !roofed(z.At(g.P.X, g.P.Y))
}

func pltTurn(g *Game) {
	if g == nil || g.P == nil || g.Dead || g.Cur == nil || g.W == nil || g.Cur.Kind == "ship" {
		return
	}
	tr := g.pltTraits()
	if len(tr) == 0 {
		return
	}
	p := g.P
	out := pltOutdoors(g)
	for _, id := range tr {
		switch id {
		case "high_gravity":
			if g.Cur.Z == 0 {
				if ok, _ := g.pltCounter(id); !ok {
					p.AddSt("sluggish", 2)
					if g.Turn%240 == 0 {
						g.Msg("{{Y|The heavy gravity drags at every limb.}}")
					}
				}
			}
		case "aether_rich":
			if g.Turn%2 == 0 {
				for k, v := range p.Cool {
					if v > 0 {
						p.Cool[k] = v - 1
					}
				}
			}
			if out && g.Turn%90 == 0 && g.R.Chance(0.07) {
				if ok, _ := g.pltCounter(id); !ok {
					p.AddSt("disoriented", 5)
					g.Msg("{{M|Raw aether crawls over your skin; for a moment your shape feels borrowed.}}")
					if g.R.Chance(0.4) {
						g.gainXP(5 + p.Level)
					}
				}
			}
		case "night_day_halves":
			if locked, _ := wyTidallyLockedLight(g); locked || g.Cur.Z != 0 {
				continue
			}
			day := pltDayHalf(g)
			g.Cur.Dark = !day
			if out {
				if ok, _ := g.pltCounter(id); !ok {
					if day && g.Turn%15 == 0 {
						g.Thirst += 4
						if g.Turn%120 == 0 {
							g.Msg("{{Y|The unmoving sun parches your throat.}}")
						}
					}
					if !day && g.Turn%15 == 0 {
						g.Hunger += 4
						if g.Turn%120 == 0 {
							g.Msg("{{C|The endless night chills you to the bone.}}")
						}
					}
				}
			}
		case "ocean_world":
			if g.Cur.Z == 0 && !g.Cur.Underwater && g.Cur.At(p.X, p.Y) == TShallow && g.Turn%10 == 0 {
				if ok, _ := g.pltCounter(id); !ok {
					g.Hunger += 3
					if g.Turn%60 == 0 {
						g.Msg("{{c|Wading is cold, sodden work; a boat would serve you better.}}")
					}
				}
			}
		case "storm_world":
			if out && g.Turn%6 == 0 && g.exposedAt(p.X, p.Y) && g.R.Chance(0.10) {
				if ok, by := g.pltCounter(id); ok {
					g.Msg("{{Y|Lightning arcs down and your " + by + " draws it harmlessly aside.}}")
				} else {
					g.lightning()
				}
			}
		case "fungal_bloom":
			if (out || g.Cur.Dark) && g.Cur.Z == 0 && g.Turn%15 == 0 {
				ch := 0.35
				if g.Cur.At(p.X, p.Y) == TFungus {
					ch = 0.7
				}
				if ok, _ := g.pltCounter(id); !ok && g.R.Chance(ch) {
					p.AddSt("ill", 90)
					g.Msg("{{G|You cough; spores take root in your lungs and a fever starts.}}")
				}
			}
		case "frozen":
			if out && g.Turn%10 == 0 {
				if ok, _ := g.pltCounter(id); !ok {
					g.Msg("{{C|The cold bites deep (-1 HP).}}")
					g.hurt(p, 1, nil, "the cold")
					if g.Dead {
						return
					}
				}
			}
		case "sacred":
			pltSacredRest(g)
		case "plague_world":
			if g.Cur.Z == 0 && g.Turn%60 == 0 && g.R.Chance(0.12) {
				if ok, _ := g.pltCounter(id); !ok {
					p.AddSt("ill", 240)
					g.Msg("{{G|Sickness of the plague-lands settles in your bones.}}")
				}
			}
		}
	}
}

func pltDayHalf(g *Game) bool {
	h := maxi(2, g.W.H)
	return g.WY < h/2
}

func pltSacredRest(g *Game) {
	s := g.plt()
	p := g.P
	if p.X == s.LastX && p.Y == s.LastY {
		s.Idle++
	} else {
		s.Idle = 0
		s.LastX, s.LastY = p.X, p.Y
	}
	if s.Idle < 10 || s.Idle%10 != 0 {
		return
	}
	near := g.Cur.Kind == "town"
	if !near {
		for dy := -3; dy <= 3 && !near; dy++ {
			for dx := -3; dx <= 3; dx++ {
				if g.Cur.At(p.X+dx, p.Y+dy) == TAltar {
					near = true
					break
				}
			}
		}
	}
	if !near {
		return
	}
	healed := false
	if p.HP < p.MaxHP {
		p.HP = mini(p.MaxHP, p.HP+2)
		healed = true
	}
	for _, st := range []string{"ill", "glotrot", "hzsick", "hzfog", "hzpox", "poisoned"} {
		if p.St(st) > 0 {
			delete(p.Status, st)
			healed = true
		}
	}
	if healed && s.Idle%30 == 0 {
		g.Msg("{{Y|A holy calm rests on you here; your hurts ease.}}")
	}
}

// ---- zone hook: terrain and loot dressing ----

func pltZone(g *Game, zn *Zone, _ *Rng) {
	if g == nil || g.W == nil || zn == nil || zn.Z != 0 || zn.Sky || zn.Underwater || zn.Kind == "town" || zn.Kind == "ship" {
		return
	}
	tr := g.pltTraits()
	if len(tr) == 0 {
		return
	}
	r := NewRng(g.Seed + ":pt:zone:" + zn.Key)
	base := func(t Terrain) bool { return t == TGrass || t == TTallGrass || t == TSand || t == TDirt }
	for _, id := range tr {
		switch id {
		case "ocean_world":
			orig := append([]Terrain(nil), zn.T...)
			wet := func(x, y int) bool {
				if !zn.In(x, y) {
					return false
				}
				t := orig[y*MW+x]
				return t == TShallow || t == TDeep
			}
			for y := 1; y < MH-1; y++ {
				for x := 1; x < MW-1; x++ {
					if !base(orig[y*MW+x]) {
						continue
					}
					near := wet(x+1, y) || wet(x-1, y) || wet(x, y+1) || wet(x, y-1)
					if near && r.Chance(0.6) || !near && r.Chance(0.03) {
						zn.T[y*MW+x] = TShallow
					}
				}
			}
			if r.Chance(0.3) {
				for tries := 0; tries < 200; tries++ {
					x, y := r.Int(2, MW-3), r.Int(2, MH-3)
					if zn.T[y*MW+x] != TShallow || zn.ActorAt(x, y) != nil {
						continue
					}
					land := false
					for _, d := range [][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
						if base(zn.At(x+d[0], y+d[1])) {
							land = true
						}
					}
					if land {
						zn.T[y*MW+x] = TBoat
						zn.Notes[y*MW+x] = "A moored skiff, left for whoever needs it."
						break
					}
				}
			}
		case "fungal_bloom":
			for i, t := range zn.T {
				if base(t) && r.Chance(0.12) {
					zn.T[i] = TFungus
				}
			}
		case "frozen":
			for i, t := range zn.T {
				if base(t) && r.Chance(0.45) {
					zn.T[i] = TSnow
				}
			}
		case "war_ruined":
			for i, t := range zn.T {
				if base(t) {
					if r.Chance(0.05) {
						zn.T[i] = TRubble
					} else if r.Chance(0.008) {
						zn.T[i] = TBones
					}
				}
			}
			for n := r.Int(1, 3); n > 0; n-- {
				if x, y, ok := zn.FreeSpot(r, nil); ok {
					names := []string{"battlefield salvage", "spent war-gear", "twisted armour plates", "broken weapon hafts"}
					zn.Items = append(zn.Items, &Item{ID: g.newID(), Kind: "scrap", Name: Pick(r, names), Glyph: '%', F: CDarkGray,
						Count: r.Int(1, 3), Value: 6 + zn.Level*2, Weight: 1, X: x, Y: y,
						Lore: "Scavenged from the wars that ruined this world."})
				}
			}
		case "sacred":
			if r.Chance(0.12) {
				if x, y, ok := zn.FreeSpot(r, nil); ok {
					zn.Set(x, y, TAltar)
					zn.Notes[y*MW+x] = "A wayside shrine; the whole world is hallowed."
				}
			}
		}
	}
}

// ---- enter hook: journal and arrival warnings ----

func pltEnter(g *Game, zn *Zone) {
	if g == nil || g.W == nil || zn == nil || g.P == nil {
		return
	}
	s := g.plt()
	id := g.PlanetIndex()
	e := s.Journal[id]
	if e == nil {
		u := unUni(g.W)
		e = &pltEntry{Name: u.planetLabel(clampi(id, 0, len(u.Planets)-1)), Day: g.Day(), Date: g.dateLine(),
			Traits: append(append([]string{}, g.wyPlanetTraits()...), g.pltTraits()...), Scanned: s.Scanned[id]}
		s.Journal[id] = e
		var warn []string
		for _, t := range g.pltTraits() {
			if l := g.pltPrepLine(t); strings.Contains(l, "unprepared") {
				warn = append(warn, pltName(t))
			}
		}
		g.Msg("{{M|You arrive on " + e.Name + ".}} Traits: " + pltTraitList(g.pltTraits()) + ".")
		if len(warn) > 0 {
			g.Msg("{{R|Unprepared for: " + strings.Join(warn, ", ") + ".}}")
		}
	}
	if zn.Name != "" && zn.Kind != "" && zn.Kind != "wild" && zn.Kind != "ship" && len(e.Places) < 10 {
		for _, p := range e.Places {
			if p == zn.Name {
				return
			}
		}
		e.Places = append(e.Places, zn.Name)
	}
}

func pltTraitList(ids []string) string {
	var n []string
	for _, t := range ids {
		n = append(n, pltName(t))
	}
	return strings.Join(n, ", ")
}

// ---- sidebar ----

func pltSide(g *Game) []string {
	tr := g.pltTraits()
	if g == nil || g.W == nil || len(tr) == 0 {
		return nil
	}
	var tags []string
	bad := false
	for _, t := range tr {
		d := pltTraitByID[t]
		if d == nil {
			continue
		}
		tags = append(tags, d.Tag)
		if d.Counter != "" {
			if ok, _ := g.pltCounter(t); !ok {
				bad = true
			}
		}
	}
	line := "{{Y|World:}} " + strings.Join(tags, ", ")
	if bad {
		line += " {{R|(!)}}"
	}
	return []string{line}
}

// ---- hook helpers for shared files ----

// pltTravelMul scales overland travel time (>1 = slower) on this planet (tvPace/tvMinutes).
func pltTravelMul(g *Game) float64 {
	if g == nil || g.W == nil {
		return 1
	}
	m := 1.0
	if g.pltHas("high_gravity") {
		if ok, _ := g.pltCounter("high_gravity"); !ok {
			m *= 1.35
		}
	}
	if g.pltHas("frozen") {
		m *= 1.1
	}
	return m
}

// pltFallDamage doubles fall damage under high gravity (unless countered).
func pltFallDamage(g *Game, dmg int) int {
	if g != nil && dmg > 0 && g.pltHas("high_gravity") {
		if ok, _ := g.pltCounter("high_gravity"); !ok {
			return dmg * 2
		}
	}
	return dmg
}

// pltPriceMod is a multiplier for price() (buy: what the merchant charges; sell: what they pay).
func pltPriceMod(g *Game, buy bool) float64 {
	if g != nil && g.pltHas("trade_hub") {
		if buy {
			return 0.9
		}
		return 1.1
	}
	return 1
}

// pltSpellBoost is a percentage bonus to spell/psi power on aether-rich worlds.
func pltSpellBoost(g *Game) int {
	if g != nil && g.pltHas("aether_rich") {
		return 25
	}
	return 0
}

// pltScavengeBonus is extra salvage found when scavenging on war-ruined worlds.
func pltScavengeBonus(g *Game) int {
	if g != nil && g.pltHas("war_ruined") {
		return 1
	}
	return 0
}

// ---- star map: knowledge, warnings, scan ----

func (g *Game) pltKnown(i int) bool {
	s := g.plt()
	return i == g.PlanetIndex() || s.Scanned[i] || s.Journal[i] != nil
}

func (g *Game) pltPlanetLines(i int) []string {
	u := unUni(g.W)
	if i < 0 || i >= len(u.Planets) {
		return nil
	}
	if !g.pltKnown(i) {
		return []string{"{{K|  unsurveyed: scan it (star map s / F11) before you go}}"}
	}
	var out []string
	for _, t := range pltTraits(g.W, i) {
		d := pltTraitByID[t]
		out = append(out, "  {{W|"+d.Name+"}}: "+d.Desc)
		if l := g.pltPrepLine(t); l != "" {
			out = append(out, "    "+l)
		}
	}
	if sig := g.plt().Signal[i]; sig != "" {
		out = append(out, "  {{c|signal}}: "+sig)
	}
	return out
}

// pltStarMapLines: extra lines for the star-map screen (append to wyStarMapLines).
func pltStarMapLines(g *Game) []string {
	if g == nil || g.W == nil {
		return nil
	}
	u := unUni(g.W)
	out := []string{"", "{{Y|Survey & preparation (s scans the selected sphere):}}"}
	for i := range u.Planets {
		mark := " "
		if i == g.Sel {
			mark = ">"
		}
		out = append(out, fmt.Sprintf("%s{{C|%s}}", mark, u.planetLabel(i)))
		out = append(out, g.pltPlanetLines(i)...)
	}
	for ri, rf := range u.Rifts {
		_ = rf
		if l := pltRiftLines(g, ri); len(l) > 0 {
			out = append(out, l...)
		}
	}
	return out
}

// pltRiftLines: warnings for travelling through rift ri toward the far planet (show on the rift
// gate screen and in the star map).
func pltRiftLines(g *Game, ri int) []string {
	if g == nil || g.W == nil {
		return nil
	}
	u := unUni(g.W)
	if ri < 0 || ri >= len(u.Rifts) {
		return nil
	}
	rf := u.Rifts[ri]
	to := rf.B.Planet
	if rf.B.Planet == g.PlanetIndex() {
		to = rf.A.Planet
	}
	if to < 0 || to >= len(u.Planets) {
		return nil
	}
	head := "{{M|Ø " + rf.Name + "}} leads to " + u.planetLabel(to) + ":"
	if !g.pltKnown(to) {
		return []string{head + " {{K|unknown conditions, scan first}}"}
	}
	out := []string{head}
	for _, t := range pltTraits(g.W, to) {
		d := pltTraitByID[t]
		l := "  " + d.Name
		if p := g.pltPrepLine(t); p != "" {
			l += ": " + p
		}
		out = append(out, l)
	}
	return out
}

// pltRiftWarnMsg posts the rift warnings to the message log (call when a rift is chosen).
func pltRiftWarnMsg(g *Game, ri int) {
	for _, l := range pltRiftLines(g, ri) {
		g.Msg(l)
	}
}

func (g *Game) pltScanGear() *Item {
	for _, it := range g.P.Inv {
		if pltItemHits(it, pltScanTokens, nil) {
			return it
		}
	}
	return nil
}

func (g *Game) pltScrap() (*Item, int) {
	n := 0
	var first *Item
	for _, it := range g.P.Inv {
		if it.Kind == "scrap" {
			if first == nil {
				first = it
			}
			n += maxi(it.Count, 1)
		}
	}
	return first, n
}

func (g *Game) pltDropScrap(n int) {
	for n > 0 {
		var it *Item
		for _, o := range g.P.Inv {
			if o.Kind == "scrap" {
				it = o
				break
			}
		}
		if it == nil {
			return
		}
		c := maxi(it.Count, 1)
		if c > n {
			it.Count = c - n
			return
		}
		n -= c
		for i, o := range g.P.Inv {
			if o == it {
				g.P.Inv = append(g.P.Inv[:i], g.P.Inv[i+1:]...)
				break
			}
		}
	}
}

// pltScan surveys planet i: pltScanMinutes of time plus a charge of survey gear (reusable
// instruments cost time only) or pltScanScrapCost scrap. It reveals the planet's traits.
func pltScan(g *Game, i int) bool {
	u := unUni(g.W)
	if i < 0 || i >= len(u.Planets) {
		return false
	}
	if g.pltKnown(i) {
		g.Msg("You already know that sphere.")
		return false
	}
	gear := g.pltScanGear()
	_, scrap := g.pltScrap()
	switch {
	case gear != nil && gear.Charges > 0:
		gear.Charges--
	case gear != nil && gear.Charges == 0 && (gear.Stack() || gear.Kind == "tonic"):
		gear.Count--
		if gear.Count <= 0 {
			for k, o := range g.P.Inv {
				if o == gear {
					g.P.Inv = append(g.P.Inv[:k], g.P.Inv[k+1:]...)
					break
				}
			}
		}
	case gear != nil:
		// reusable instrument: time only
	case scrap >= pltScanScrapCost:
		g.pltDropScrap(pltScanScrapCost)
		g.Msg("You wire " + fmt.Sprint(pltScanScrapCost) + " pieces of scrap into a crude antenna.")
	default:
		g.Msg("You need survey gear (scanner, telescope, star chart...) or " + fmt.Sprint(pltScanScrapCost) + " pieces of scrap to listen to the sky.")
		return false
	}
	g.tvPass(pltScanMinutes)
	s := g.plt()
	s.Scanned[i] = true
	s.Scans++
	tr := pltTraits(g.W, i)
	h := HashStr(unSeed(u, i) + ":pt:signal")
	sig := []string{
		"a steady beacon, rich in traffic",
		"faint voices in an old tongue",
		"bursts of static from vast storms",
		"a slow pulse like a drowned bell",
		"a hymn carried on the carrier wave",
		"nothing but wind and distance",
	}[h%6]
	s.Signal[i] = sig
	g.Msg(fmt.Sprintf("{{G|Survey complete: %s shows %s.}}", u.planetLabel(i), pltTraitList(tr)))
	return true
}

// pltScanSelected scans the planet under the star-map cursor (hook: key "s" on the star map).
func pltScanSelected(g *Game) {
	pltScan(g, g.Sel)
}

// ---- own screen (F11): survey list and planet journal ----

// pltJournalLines: one line per visited planet (traits, notable places, arrival date).
func pltJournalLines(g *Game) []string {
	s := g.plt()
	var ids []int
	for i := range s.Journal {
		ids = append(ids, i)
	}
	sort.Ints(ids)
	var out []string
	for _, i := range ids {
		e := s.Journal[i]
		l := fmt.Sprintf("{{C|%s}}: %s; arrived day %d (%s)", e.Name, pltTraitList(pltTraits(g.W, i)), e.Day, e.Date)
		if len(e.Places) > 0 {
			l += "; seen: " + strings.Join(e.Places, ", ")
		}
		out = append(out, l)
	}
	return out
}

func pltKey(g *Game, k string) {
	u := unUni(g.W)
	n := maxi(1, len(u.Planets))
	switch k {
	case "Escape", "q", "Q":
		g.Mode = "main"
	case "Tab":
		g.Tab = (g.Tab + 1) % 2
	case "Up", "N8", "k":
		g.Sel = clampi(g.Sel-1, 0, n-1)
	case "Down", "N2", "j":
		g.Sel = clampi(g.Sel+1, 0, n-1)
	case "Enter", "s", "S":
		if g.Tab == 0 {
			pltScan(g, g.Sel)
		}
	}
}

func pltDraw(g *Game) {
	s := g.sc
	var lines []string
	title := "Planet survey"
	if g.Tab == 0 {
		lines = pltStarMapLines(g)
	} else {
		title = "Planet journal"
		lines = pltJournalLines(g)
		if len(lines) == 0 {
			lines = []string{"No planets visited yet."}
		}
	}
	h := clampi(len(lines)+4, 10, MH-2)
	y0 := maxi(1, (MH-h)/2)
	s.Box(4, y0, 100, h, CMagenta, title)
	for i, ln := range lines {
		if i+2 >= h-1 {
			break
		}
		s.TextB(6, y0+1+i, ln, CGray, CBgDark, 102)
	}
	s.TextB(6, y0+h-1, "[Up/Down select, Enter/s scan, Tab survey/journal, Esc close]", CYellow, CBgDark, 102)
}
