package engine

import (
	"fmt"
	"sort"
	"strings"
)

// Hazards (prefix hz): Realmforge's natural disasters and diseases, read from WorldData.Hazards
// (faults, volcanoes, recent disasters, live outbreaks, disease profiles). Recent disasters mark the
// land (quake rubble and homeless townsfolk, flood mud and debris, ash around erupted volcanoes, burnt
// forests, locust-stripped fields); outbreaks bring quarantined towns with guarded gates, sick townsfolk,
// physicians and pyres; the player can catch a disease (incubation, symptoms as statuses, contagion to
// companions, recovery with immunity) and be cured by physicians, priests, herbs or medicine. Fault
// zones tremble and active volcanoes rumble. Zone generation uses only the side rng seed:hz:zoneKey,
// turn effects use hashes of seed+turn, so a world without hazards is unchanged.
// State: g.ext("hz") (body: illness, immunity) and g.ext("hzl") (planet-local: passes, last zone).

type HazardsData struct {
	Faults    []HzFault    `json:"faults,omitempty"`
	Volcanoes []HzVolcano  `json:"volcanoes,omitempty"`
	Recent    []HzRecent   `json:"recent,omitempty"`
	Outbreaks []HzOutbreak `json:"outbreaks,omitempty"`
	Diseases  []HzDisease  `json:"diseases,omitempty"`
}

type HzFault struct {
	Name      string `json:"name"`
	Parasangs []int  `json:"parasangs"`
	Last      int    `json:"last,omitempty"`
}

type HzVolcano struct {
	P      int    `json:"p"`
	Name   string `json:"name"`
	Active bool   `json:"active"`
	Last   int    `json:"last"`
}

type HzRecent struct {
	Kind     string  `json:"kind"` // quake tsunami eruption storm tornado flood landslide avalanche locust wildfire
	Name     string  `json:"name,omitempty"`
	Year     int     `json:"year"`
	P        int     `json:"p"`
	Severity float64 `json:"severity"`
	Towns    []int   `json:"towns"`
	Area     []int   `json:"area,omitempty"`
}

type HzOutbreak struct {
	Disease     string `json:"disease"`
	Name        string `json:"name"`
	Since       int    `json:"since"`
	Parasangs   []int  `json:"parasangs"`
	Quarantined []int  `json:"quarantined"`
}

type HzDisease struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	Vector     string   `json:"vector"`
	Trans      float64  `json:"trans"`
	Lethal     float64  `json:"lethal"`
	Immunity   float64  `json:"immunity"`
	Incubation int      `json:"incubation"`
	Duration   int      `json:"duration"`
	Hits       []string `json:"hits"`
	Symptoms   []string `json:"symptoms"`
	Cures      []string `json:"cures"`
	Desc       string   `json:"desc"`
}

func init() {
	zoneHooks = append(zoneHooks, hzZone)
	enterHooks = append(enterHooks, hzEnter)
	turnHooks = append(turnHooks, hzTurn)
	talkHooks = append(talkHooks, hzTalk)
	sideHooks = append(sideHooks, hzSide)
	unLocalKeys = append(unLocalKeys, "hzl")
	skAddStatus(&skStatusDef{ID: "hzsick", Name: "Sick", Glyph: 'ж', AGI: -2, Speed: -10})
	skAddStatus(&skStatusDef{ID: "hzfog", Name: "Fevered mind", Glyph: 'ж', INT: -2, WIL: -2})
	skAddStatus(&skStatusDef{ID: "hzpox", Name: "Disfigured", Glyph: 'ж', EGO: -3})
}

// ---- state ----

type hzIll struct {
	ID     string `json:"id"`
	Name   string `json:"n"` // outbreak name ("the Black Death")
	Caught int    `json:"c"` // turn caught
	Onset  int    `json:"o"` // turn symptoms begin
	Until  int    `json:"u"` // turn it ends
	Where  string `json:"w"` // town or place
	Cured  string `json:"cu,omitempty"`
	Said   int    `json:"s,omitempty"` // last symptom message turn
}

type hzState struct {
	Ill    *hzIll            `json:"ill,omitempty"`
	Immune map[string]int    `json:"im,omitempty"` // disease id → turn recovered
	Comp   map[int]*hzIll    `json:"cp,omitempty"` // companion actor id → illness
	CImm   map[int][]string  `json:"ci,omitempty"` // companion actor id → immune disease ids
	Had    int               `json:"h,omitempty"`  // diseases survived
	Seen   map[string]bool   `json:"se,omitempty"` // journal / once-only notes
	Extra  map[string]string `json:"x,omitempty"`
}

type hzLocal struct {
	Pass map[int]bool `json:"ps,omitempty"` // burg ids the player may enter during quarantine
	Last string       `json:"l,omitempty"`  // last zone outside a quarantined town
	LX   int          `json:"lx,omitempty"`
	LY   int          `json:"ly,omitempty"`
}

func hzSt(g *Game) *hzState {
	s := g.ext("hz", func() any { return &hzState{} }).(*hzState)
	if s.Immune == nil {
		s.Immune = map[string]int{}
	}
	if s.Comp == nil {
		s.Comp = map[int]*hzIll{}
	}
	if s.CImm == nil {
		s.CImm = map[int][]string{}
	}
	if s.Seen == nil {
		s.Seen = map[string]bool{}
	}
	return s
}

func hzLoc(g *Game) *hzLocal {
	s := g.ext("hzl", func() any { return &hzLocal{} }).(*hzLocal)
	if s.Pass == nil {
		s.Pass = map[int]bool{}
	}
	return s
}

// ---- world index ----

type hzIndex struct {
	fault    map[int]int   // parasang → fault index
	volc     map[int]int   // parasang → volcano index
	recentAt map[int][]int // parasang → recent indices (epicentre, area, towns)
	townHit  map[int][]int // burg id → recent indices
	outAt    map[int]int   // parasang → outbreak index
	quar     map[int]int   // burg id → outbreak index
	dis      map[string]*HzDisease
}

var hzIdxCache = map[*WorldData]*hzIndex{}

func (w *WorldData) hz() *hzIndex {
	if w == nil || w.Hazards == nil {
		return nil
	}
	if ix := hzIdxCache[w]; ix != nil {
		return ix
	}
	h := w.Hazards
	ix := &hzIndex{fault: map[int]int{}, volc: map[int]int{}, recentAt: map[int][]int{}, townHit: map[int][]int{}, outAt: map[int]int{}, quar: map[int]int{}, dis: map[string]*HzDisease{}}
	for i := range h.Faults {
		for _, p := range h.Faults[i].Parasangs {
			ix.fault[p] = i
		}
	}
	for i := range h.Volcanoes {
		ix.volc[h.Volcanoes[i].P] = i
	}
	for i := range h.Recent {
		e := &h.Recent[i]
		seen := map[int]bool{}
		add := func(p int) {
			if !seen[p] {
				seen[p] = true
				ix.recentAt[p] = append(ix.recentAt[p], i)
			}
		}
		add(e.P)
		for _, p := range e.Area {
			add(p)
		}
		for _, b := range e.Towns {
			ix.townHit[b] = append(ix.townHit[b], i)
			if bd := w.burgByID[b]; bd != nil {
				add(bd.P)
			}
		}
	}
	for i := range h.Outbreaks {
		for _, p := range h.Outbreaks[i].Parasangs {
			ix.outAt[p] = i
		}
		for _, b := range h.Outbreaks[i].Quarantined {
			ix.quar[b] = i
		}
	}
	for i := range h.Diseases {
		ix.dis[h.Diseases[i].ID] = &h.Diseases[i]
	}
	hzIdxCache[w] = ix
	return ix
}

// hzDisease: the profile for an id (falls back to a mild generic sickness).
func (w *WorldData) hzDisease(id string) *HzDisease {
	if ix := w.hz(); ix != nil {
		if d := ix.dis[id]; d != nil {
			return d
		}
	}
	return &HzDisease{ID: id, Name: id, Vector: "contact", Trans: 0.2, Lethal: 0.02, Immunity: 0.5, Incubation: 2, Duration: 5, Hits: []string{"tou"}, Symptoms: []string{"a fever"}, Cures: []string{"physician", "rest"}}
}

// hzVolcanoAt: the volcano in parasang p, if any (terrain agents may add tgVolcanoAt; this reads the export).
func (w *WorldData) hzVolcanoAt(p int) *HzVolcano {
	if ix := w.hz(); ix != nil {
		if i, ok := ix.volc[p]; ok {
			return &w.Hazards.Volcanoes[i]
		}
	}
	return nil
}

func (w *WorldData) hzFaultAt(p int) *HzFault {
	if ix := w.hz(); ix != nil {
		if i, ok := ix.fault[p]; ok {
			return &w.Hazards.Faults[i]
		}
	}
	return nil
}

func (w *WorldData) hzOutbreakAt(p int) *HzOutbreak {
	if ix := w.hz(); ix != nil {
		if i, ok := ix.outAt[p]; ok {
			return &w.Hazards.Outbreaks[i]
		}
	}
	return nil
}

// hzQuarantine: the outbreak that keeps burg b under quarantine, if any.
func (w *WorldData) hzQuarantine(b int) *HzOutbreak {
	if ix := w.hz(); ix != nil {
		if i, ok := ix.quar[b]; ok {
			return &w.Hazards.Outbreaks[i]
		}
	}
	return nil
}

// how long marks stay on the land, in years
var hzMarkYears = map[string]int{"quake": 30, "tsunami": 15, "eruption": 40, "storm": 8, "tornado": 5, "flood": 10, "landslide": 25, "avalanche": 6, "locust": 3, "wildfire": 15}

// hzRecentAt: disasters still marking parasang p (newest first).
func (w *WorldData) hzRecentAt(p int) []*HzRecent {
	ix := w.hz()
	if ix == nil {
		return nil
	}
	var out []*HzRecent
	for _, i := range ix.recentAt[p] {
		e := &w.Hazards.Recent[i]
		if w.Year-e.Year <= hzMarkYears[e.Kind] {
			out = append(out, e)
		}
	}
	sort.SliceStable(out, func(a, b int) bool { return out[a].Year > out[b].Year })
	return out
}

func hzHas(xs []int, v int) bool {
	for _, x := range xs {
		if x == v {
			return true
		}
	}
	return false
}

func hzHasS(xs []string, v string) bool {
	for _, x := range xs {
		if x == v {
			return true
		}
	}
	return false
}

func hzCap(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

var hzKindName = map[string]string{"quake": "earthquake", "tsunami": "great wave", "eruption": "eruption", "storm": "great storm", "tornado": "whirlwind", "flood": "flood", "landslide": "landslide", "avalanche": "avalanche", "locust": "locust swarm", "wildfire": "wildfire"}

// hzLabel: "the earthquake of the Kadu Fault in Y201"
func hzLabel(e *HzRecent) string {
	s := "the " + hzKindName[e.Kind]
	if e.Name != "" {
		switch e.Kind {
		case "quake", "tsunami":
			s += " on " + e.Name
		case "eruption":
			s += " of " + e.Name
		case "storm":
			s += " “" + e.Name + "”"
		case "flood":
			s += " of the " + e.Name
		}
	}
	return fmt.Sprintf("%s in Y%d", s, e.Year)
}

// ---- genre words ----

func hzPick(genre string, m map[string]string) string {
	if s, ok := m[genre]; ok {
		return s
	}
	return m[""]
}

func (g *Game) hzPhysicianTitle() string {
	return hzPick(g.Genre, map[string]string{"": "plague doctor", "steampunk": "physician", "dieselpunk": "army medic", "atompunk": "civil-defence medic", "wasteland": "sawbones", "cyberpunk": "street doc", "biopunk": "vat medic", "solarpunk": "health warden"})
}

func (g *Game) hzMaskName() string {
	return hzPick(g.Genre, map[string]string{"": "beaked plague mask", "steampunk": "carbolic gauze mask", "dieselpunk": "gas mask", "atompunk": "filter mask", "wasteland": "rag respirator", "cyberpunk": "filter respirator", "biopunk": "membrane mask", "solarpunk": "woven filter mask"})
}

// medicine: late genres sell real cures and vaccines; old ages sell herbal tonics
func (g *Game) hzLate() bool {
	switch g.Genre {
	case "fantasy", "mythic", "twilight":
		return false
	}
	return true
}

func (g *Game) hzCureName() string {
	return hzPick(g.Genre, map[string]string{"": "fever-bark theriac", "steampunk": "carbolic tincture", "dieselpunk": "sulfa tablets", "atompunk": "penicillin shot", "wasteland": "scavenged antibiotics", "cyberpunk": "antiviral injector", "biopunk": "counter-strain ampoule", "solarpunk": "phage drops"})
}

func (g *Game) hzVaxName() string {
	return hzPick(g.Genre, map[string]string{"": "", "steampunk": "cowpox lancet", "dieselpunk": "vaccine shot", "atompunk": "vaccine shot", "wasteland": "old-world vaccine", "cyberpunk": "immune patch", "biopunk": "vaccine ampoule", "solarpunk": "vaccine drops"})
}

func (g *Game) hzMakeMask() *Item {
	return &Item{ID: g.newID(), Kind: "armor", Slot: "face", Name: g.hzMaskName(), Glyph: '&', F: CWhite, Value: 25, Weight: 1, Extra: map[string]int{"hzmask": 1},
		Lore: "Worn against bad air and sick breath. Halves your chance of catching a contagion."}
}

func (g *Game) hzMakeCure(vax bool) *Item {
	if vax {
		return &Item{ID: g.newID(), Kind: "tonic", Effect: "hzvax", Name: g.hzVaxName(), Glyph: '!', F: CCyan, Count: 1, Value: 40, Lore: "Protects against a contagion that leaves the survivors immune."}
	}
	return &Item{ID: g.newID(), Kind: "tonic", Effect: "hzcure", Name: g.hzCureName(), Glyph: '!', F: CGreen, Count: 1, Value: 35, Lore: "Breaks a fever and cleanses a sickness."}
}

func (g *Game) hzMasked(a *Actor) bool {
	for _, bp := range a.Body {
		if bp.Item != nil && bp.Item.Extra["hzmask"] > 0 && !bp.Severed {
			return true
		}
	}
	return false
}
