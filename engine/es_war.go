package engine

// Battlefields, sieges, fallen towns, last stands and disaster sites, told by what is left (prefix ew,
// state g.ext("esw"), planet-local). Each planet's ~25 most significant chronicle entries (WorldData.Events,
// dead notables of PersonData killed in battle, Hazards.Recent and Hazards.Outbreaks) are ranked once
// (ewReg, deterministic) and given one outskirts or wild sub-zone of their parasang, at most two per
// parasang. A zone hook paints the scene with a side rng seed:esw:<kind>:<id> on free natural tiles only,
// away from stairs, actors and items (tgPainter), then reconnects the zone; zones without a scene are never
// touched. Every scene has three layers: what you see (tile notes, item names and lore), one readable note
// (Kind book), and a payoff when you find it and read it: a journal secret, a fact in the case notes
// (knowledge.go), a lead to a real place, a little loot, and a Places entry (places.go).
// Disaster sites only add items (remains, belongings, a tally): the ground marks are hazards_zone.go's.

import (
	"fmt"
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

type ewScene struct {
	ID     string
	Kind   string // battle siege fall stand disaster plague famine
	E      int    // WorldData.Events index, -1 none
	Per    int    // WorldData.People index (stand), -1 none
	Hz     int    // Hazards.Recent index, -1 none
	Ob     int    // Hazards.Outbreaks index, -1 none
	Dis    string // disaster kind (quake flood ...) for disaster scenes
	P      int
	ZX, ZY int
	Score  int
	Year   int
}

type ewRegistry struct {
	list   []*ewScene
	byZone map[[4]int][]*ewScene
}

var ewCache = map[*WorldData]*ewRegistry{}

// esRec is the saved record of a built scene: the same text on every read, after save and load.
type esRec struct {
	ID   string `json:"id"`
	K    string `json:"k"`
	Z    string `json:"z"`
	E    int    `json:"e"`
	P    int    `json:"p"`
	Y    int    `json:"y,omitempty"`
	CX   int    `json:"cx"`
	CY   int    `json:"cy"`
	T    string `json:"t"`            // title (Places entry)
	N    string `json:"n,omitempty"`  // the readable note's name
	L    string `json:"l"`            // what you see on finding it
	H    string `json:"h,omitempty"`  // a hint on entering the zone
	F    string `json:"f,omitempty"`  // the fact it proves
	S    string `json:"s,omitempty"`  // journal secret
	Src  string `json:"sr,omitempty"` // who "told" the fact in the case notes
	B    int    `json:"b,omitempty"`  // burg the fact is about
	Lead int    `json:"ld"`           // parasang the note points at, -1 none
	LT   string `json:"lt,omitempty"`
	D    bool   `json:"d,omitempty"`  // discovered
	Pd   bool   `json:"pd,omitempty"` // payoff given
	Hn   bool   `json:"hn,omitempty"` // hint shown
}

type esState struct {
	S map[string]*esRec `json:"s,omitempty"`
}

func ewSt(g *Game) *esState {
	s := g.ext("esw", func() any { return &esState{} }).(*esState)
	if s.S == nil {
		s.S = map[string]*esRec{}
	}
	return s
}

func init() {
	zoneHooks = append(zoneHooks, ewZone)
	enterHooks = append(enterHooks, ewEnter)
	turnHooks = append(turnHooks, ewTurn)
	unLocalKeys = append(unLocalKeys, "esw")
}

const ewMaxScenes = 25

var (
	ewRepulseRe = regexp.MustCompile(`(?i)repuls|repel|crushed|beaten back|routed|turned back|thrown back`)
	ewClashRe   = regexp.MustCompile(`(?i)storming|invasion|marches an army|overruns|clash|battle`)
	ewNavalRe   = regexp.MustCompile(`(?i)launches the|flagship|squadron|naval|docks of`)
	ewRoyalRe   = regexp.MustCompile(`(?i)royal seat|capital|citadel|seat of power|palace`)
	ewSackRe    = regexp.MustCompile(`(?i)siege|sack|plunder|flames|storm`)
	ewSickRe    = regexp.MustCompile(`(?i)breaks out|comes to|appears in|outbreak`)
	ewSickName  = []*regexp.Regexp{
		regexp.MustCompile(`call the ([\p{Lu}][\p{L}'’\- ]*?) breaks out`),
		regexp.MustCompile(`^The ([\p{Lu}][\p{L}'’\- ]*?) comes to`),
		regexp.MustCompile(`OUTBREAK: the ([\p{Lu}][\p{L}'’\- ]*?) appears`),
	}
	ewWarRe1  = regexp.MustCompile(`((?:Conquest|Invasion|War) of (?:the )?[\p{Lu}][\p{L}'’\-]*(?: [\p{Lu}][\p{L}'’\-]*)*)`)
	ewWarRe2  = regexp.MustCompile(`((?:[\p{Lu}][\p{L}'’\-]*[ –-])+(?:War|Campaign))\b`)
	ewFleeRe  = regexp.MustCompile(`([\p{Lu}][\p{L}'’\-]*(?: [\p{Lu}][\p{L}'’\-]*)?) (?:escapes|flees)`)
	ewHouseRe = regexp.MustCompile(`House ([\p{Lu}][\p{L}'’\-]+) (?:seizes|claims|storm|usurp)`)
	ewHeirRe  = []*regexp.Regexp{
		regexp.MustCompile(`(?:child|youth) ([\p{Lu}][\p{L}'’\-]+)`),
		regexp.MustCompile(`passes to ([\p{Lu}][\p{L}'’\-]+)`),
		regexp.MustCompile(`([\p{Lu}][\p{L}'’\-]+(?: [\p{Lu}][\p{L}'’\-]+)?) (?:ascends|is crowned|takes up)`),
	}
	ewBeastWonRe = regexp.MustCompile(`(?i)triumph|defeat|banish|slain|slay`)
	ewDisCue     = [][2]string{{"quake", "quake"}, {"earth heaves", "quake"}, {"ground trembles", "quake"}, {"flood", "flood"}, {"bursts its banks", "flood"}, {"swell", "flood"},
		{"hurricane", "storm"}, {"typhoon", "storm"}, {"twisting storm", "tornado"}, {"eruption", "eruption"}, {"wakes after long sleep", "eruption"}, {"mountainside gives way", "landslide"},
		{"wildfire", "wildfire"}, {"swarm", "locust"}}
)

// ---- registry: which events get a scene, and where ----

func (g *Game) ewReg() *ewRegistry {
	w := g.W
	if r, ok := ewCache[w]; ok {
		return r
	}
	r := g.ewBuildReg()
	ewCache[w] = r
	return r
}

func (w *WorldData) ewLand(p int) bool { return p >= 0 && p < len(w.Biome) && !w.IsWater(p) }

func (g *Game) ewBuildReg() *ewRegistry {
	w := g.W
	var cs []*ewScene
	add := func(s *ewScene) {
		if s.Per == 0 && s.Kind != "stand" {
			s.Per = -1
		}
		cs = append(cs, s)
	}
	y0, y1 := 0, 0
	if len(w.Events) > 0 {
		y0, y1 = w.Events[0].Y, w.Events[len(w.Events)-1].Y
	}
	rec := func(y int) int {
		if y1 <= y0 {
			return 0
		}
		return clampi(10*(y-y0)/(y1-y0), 0, 10)
	}
	hasHz := w.Hazards != nil && len(w.Hazards.Recent) > 0
	obNames := map[string]bool{}
	if w.Hazards != nil {
		for _, ob := range w.Hazards.Outbreaks {
			obNames[strings.ToLower(strings.TrimPrefix(ob.Name, "the "))] = true
		}
	}
	for i, e := range w.Events {
		if !w.ewLand(e.P) {
			continue
		}
		t := e.Text
		base := &ewScene{ID: "e" + strconv.Itoa(i), E: i, Per: -1, Hz: -1, Ob: -1, P: e.P, Year: e.Y}
		switch e.Kind {
		case "war":
			if ewNavalRe.MatchString(t) {
				continue
			}
			base.Kind, base.Score = "battle", 34+rec(e.Y)
			if ewRepulseRe.MatchString(t) || ewClashRe.MatchString(t) {
				base.Score += 15
			}
			add(base)
		case "monster":
			if w.ewBeast(t) == "" || !(strings.Contains(t, "CLASH") || strings.Contains(t, "SACKED") || strings.Contains(t, "TRIUMPH") || strings.Contains(t, "armies")) {
				continue
			}
			base.Kind, base.Score = "battle", 42+rec(e.Y)
			add(base)
		case "capture", "siege":
			bd := w.burgByID[w.Burg[e.P]]
			if bd == nil || bd.Layer != "surface" && bd.Layer != "" {
				continue
			}
			base.Kind, base.Score = "siege", 38+rec(e.Y)
			if ewRoyalRe.MatchString(t) {
				base.Score += 20
			}
			if ewSackRe.MatchString(t) {
				base.Score += 10
			}
			add(base)
		case "fall":
			bd := w.burgByID[w.Burg[e.P]]
			if bd == nil {
				if p := w.ewCapP(e.St); w.ewLand(p) {
					bd = w.burgByID[w.Burg[p]]
					base.P = p
				}
			}
			if bd == nil || bd.Layer != "surface" && bd.Layer != "" {
				continue
			}
			base.Kind, base.Score = "fall", 55+rec(e.Y)
			add(base)
		case "disaster", "eco_disaster", "famine", "plague":
			low := strings.ToLower(t)
			switch {
			case e.Kind == "famine" || strings.Contains(low, "famine") || strings.Contains(low, "dust bowl"):
				base.Kind, base.Score = "famine", 40+rec(e.Y)
				add(base)
			case e.Kind == "plague" || ewSickRe.MatchString(t) && ewSickName1(t) != "":
				if obNames[strings.ToLower(ewSickName1(t))] {
					continue // the outbreak record below covers it
				}
				base.Kind, base.Score = "plague", 32+rec(e.Y)
				add(base)
			case !hasHz && e.Kind == "disaster":
				for _, c := range ewDisCue {
					if strings.Contains(low, c[0]) {
						base.Kind, base.Dis, base.Score = "disaster", c[1], 22+rec(e.Y)
						add(base)
						break
					}
				}
			}
		}
	}
	for i := range w.People {
		pd := &w.People[i]
		c := strings.ToLower(pd.Cause)
		if pd.Alive || pd.Died == 0 || !strings.Contains(c, "battle") && !strings.Contains(c, "titan") {
			continue
		}
		ei, p := w.ewStandPlace(pd)
		if !w.ewLand(p) {
			continue
		}
		s := 30 + mini(30, pd.Renown/3)
		if pd.Role == "ruler" {
			s += 12
		}
		add(&ewScene{ID: "p" + strconv.Itoa(pd.ID), Kind: "stand", E: ei, Per: i, Hz: -1, Ob: -1, P: p, Year: pd.Died, Score: s})
	}
	if w.Hazards != nil {
		for i := range w.Hazards.Recent {
			he := &w.Hazards.Recent[i]
			if w.Year-he.Year > 60 {
				continue
			}
			p := he.P
			if !w.ewLand(p) {
				p = -1
				for _, t := range he.Towns {
					if bd := w.burgByID[t]; bd != nil && w.ewLand(bd.P) && bd.Layer == "surface" {
						p = bd.P
						break
					}
				}
			}
			if p < 0 {
				continue
			}
			add(&ewScene{ID: "h" + strconv.Itoa(i), Kind: "disaster", Dis: he.Kind, E: w.ewHzEvent(he), Per: -1, Hz: i, Ob: -1, P: p, Year: he.Year,
				Score: 20 + mini(16, int(he.Severity*2)) + 2*mini(6, len(he.Towns)) + mini(10, 60-(w.Year-he.Year))/2})
		}
		for i := range w.Hazards.Outbreaks {
			ob := &w.Hazards.Outbreaks[i]
			p := -1
			for _, q := range ob.Parasangs {
				if !w.ewLand(q) {
					continue
				}
				if p < 0 {
					p = q
				}
				if bd := w.burgByID[w.Burg[q]]; bd != nil && bd.Alive {
					p = q
					break
				}
			}
			if p < 0 {
				continue
			}
			add(&ewScene{ID: "o" + strconv.Itoa(i), Kind: "plague", E: -1, Per: -1, Hz: -1, Ob: i, P: p, Year: ob.Since, Score: 34 + len(ob.Parasangs)})
		}
	}
	sort.SliceStable(cs, func(a, b int) bool {
		if cs[a].Score != cs[b].Score {
			return cs[a].Score > cs[b].Score
		}
		return cs[a].ID < cs[b].ID
	})
	reg := &ewRegistry{byZone: map[[4]int][]*ewScene{}}
	per := map[int]int{}
	used := map[[4]int]bool{}
	taken := map[string]bool{}
	try := func(sc *ewScene) bool {
		kp := sc.Kind + ":" + strconv.Itoa(sc.P)
		if taken[sc.ID] || taken[kp] || per[sc.P] >= 2 {
			return false
		}
		zx, zy, ok := g.ewSub(sc, used)
		if !ok {
			return false
		}
		sc.ZX, sc.ZY = zx, zy
		taken[sc.ID], taken[kp] = true, true
		per[sc.P]++
		reg.list = append(reg.list, sc)
		k := [4]int{sc.P % w.W, sc.P / w.W, zx, zy}
		reg.byZone[k] = append(reg.byZone[k], sc)
		return true
	}
	for _, k := range []string{"fall", "siege", "battle", "stand", "disaster", "plague", "famine"} {
		n := 0
		for _, sc := range cs {
			if n >= 3 {
				break
			}
			if sc.Kind == k && try(sc) {
				n++
			}
		}
	}
	kinds := map[string]int{}
	for _, sc := range reg.list {
		kinds[sc.Kind]++
	}
	for _, sc := range cs {
		if len(reg.list) >= ewMaxScenes {
			break
		}
		if kinds[sc.Kind] < 6 && try(sc) {
			kinds[sc.Kind]++
		}
	}
	return reg
}

// ewSub picks the scene's sub-zone: a hash order over the eight outer zones of the parasang (sieges try
// the four beside the town first), skipping city districts, outer wards and an outlaw camp; a fallen town
// that is now a ruin takes its own centre zone.
func (g *Game) ewSub(sc *ewScene, used map[[4]int]bool) (int, int, bool) {
	w := g.W
	wx, wy := sc.P%w.W, sc.P/w.W
	bd := w.burgByID[w.Burg[sc.P]]
	type cand struct {
		x, y int
		h    uint64
	}
	var cs []cand
	for zy := 0; zy < 3; zy++ {
		for zx := 0; zx < 3; zx++ {
			h := HashStr(fmt.Sprintf("%s:esw:z:%s:%d:%d", w.Seed, sc.ID, zx, zy)) >> 2
			if zx == 1 && zy == 1 {
				if !(sc.Kind == "fall" && bd != nil && !bd.Alive) {
					continue
				}
				h = 0
			} else {
				if sc.Kind == "siege" && zx != 1 && zy != 1 {
					h |= 1 << 60
				}
				// the edge facing sea or high mountains is likely water or rock: try it last
				nx, ny := wx+zx-1, wy+zy-1
				if !w.In(nx, ny) || w.IsWater(w.Idx(nx, ny)) {
					h |= 1 << 61
				} else if w.Height[w.Idx(nx, ny)] >= 70 {
					h |= 1 << 61
				}
			}
			cs = append(cs, cand{zx, zy, h})
		}
	}
	sort.Slice(cs, func(a, b int) bool { return cs[a].h < cs[b].h })
	for _, c := range cs {
		k := [4]int{wx, wy, c.x, c.y}
		if used[k] {
			continue
		}
		if !(c.x == 1 && c.y == 1) {
			if kind, _ := g.districtAt(wx, wy, c.x, c.y); kind != "" {
				continue
			}
			if bd != nil && bd.Alive && w.outlawByBrg[bd.ID] != nil && g.outlawZone(bd) == [2]int{c.x, c.y} {
				continue
			}
		}
		used[k] = true
		return c.x, c.y, true
	}
	return 0, 0, false
}

// ewStandPlace: the battle a notable died in, the war or capture of their realm nearest their death year.
func (w *WorldData) ewStandPlace(pd *PersonData) (int, int) {
	sn := ""
	if s := w.stateByID[pd.State]; s != nil && pd.State > 0 {
		sn = s.Name
	}
	best, bp, bs := -1, pd.P, 1<<30
	for i, e := range w.Events {
		if e.Y < pd.Died-4 || e.Y > pd.Died || !w.ewLand(e.P) {
			continue
		}
		titan := strings.Contains(strings.ToLower(pd.Cause), "titan")
		if titan && e.Kind != "monster" || !titan && e.Kind != "war" && e.Kind != "capture" {
			continue
		}
		if e.St != pd.State && (sn == "" || !strings.Contains(e.Text, sn)) {
			continue
		}
		d := (pd.Died - e.Y) * 1000
		if pd.P >= 0 {
			d += abs(e.P%w.W-pd.P%w.W) + abs(e.P/w.W-pd.P/w.W)
		}
		if d < bs {
			best, bp, bs = i, e.P, d
		}
	}
	return best, bp
}

// ewHzEvent ties a hazard record to its chronicle line.
func (w *WorldData) ewHzEvent(he *HzRecent) int {
	for i, e := range w.Events {
		if e.Kind != "disaster" || e.Y != he.Year {
			continue
		}
		if e.P == he.P || he.Name != "" && strings.Contains(e.Text, strings.TrimPrefix(he.Name, "the ")) {
			return i
		}
	}
	return -1
}

func ewSickName1(t string) string {
	for _, re := range ewSickName {
		if m := re.FindStringSubmatch(t); m != nil {
			return strings.TrimSpace(m[1])
		}
	}
	return ""
}

// ---- world readers ----

func (w *WorldData) ewBeast(t string) string {
	best := ""
	for _, m := range w.Monsters {
		if len(m.Name) > len(best) && strings.Contains(t, m.Name) {
			best = m.Name
		}
	}
	return best
}

// ewNamed lists the realms named in a text, in order of appearance.
func (w *WorldData) ewNamed(text string) []int {
	type hit struct{ at, id int }
	var hs []hit
	for _, s := range w.States {
		if s.ID <= 0 || len(s.Name) < 3 {
			continue
		}
		if i := strings.Index(text, s.Name); i >= 0 {
			hs = append(hs, hit{i, s.ID})
		}
	}
	sort.Slice(hs, func(a, b int) bool {
		if hs[a].at != hs[b].at {
			return hs[a].at < hs[b].at
		}
		return hs[a].id < hs[b].id
	})
	var out []int
	for _, h := range hs {
		out = append(out, h.id)
	}
	return out
}

func (w *WorldData) ewSides(e EventData) (att, def int) {
	att = e.St
	ids := w.ewNamed(e.Text)
	if w.stateByID[att] == nil && len(ids) > 0 {
		att = ids[0]
	}
	for _, id := range ids {
		if id != att {
			def = id
			break
		}
	}
	if def == 0 && w.ewLand(e.P) {
		if o := w.State[e.P]; o != att && o > 0 {
			def = o
		}
	}
	return
}

func (w *WorldData) ewRealm(id int) string {
	if s := w.stateByID[id]; s != nil && id > 0 {
		if s.FullName != "" {
			return s.FullName
		}
		return s.Name
	}
	return ""
}

func (w *WorldData) ewShort(id int) string {
	if s := w.stateByID[id]; s != nil && id > 0 {
		return s.Name
	}
	return ""
}

func (w *WorldData) ewCapP(id int) int {
	if s := w.stateByID[id]; s != nil {
		if b := w.burgByID[s.Capital]; b != nil {
			return b.P
		}
	}
	return -1
}

func (w *WorldData) ewCapName(id int) string {
	if s := w.stateByID[id]; s != nil {
		if b := w.burgByID[s.Capital]; b != nil {
			return b.Name
		}
	}
	return ""
}

// ewNear: the nearest surface town to p within max parasangs (alive only when alive), skipping skip.
func (w *WorldData) ewNear(p, max int, alive bool, skip int) *BurgData {
	px, py := p%w.W, p/w.W
	var best *BurgData
	bd := 1 << 30
	for i := range w.Burgs {
		b := &w.Burgs[i]
		if b.ID == skip || b.ID <= 0 || alive && !b.Alive || b.Layer != "surface" && b.Layer != "" {
			continue
		}
		d := (b.P%w.W-px)*(b.P%w.W-px) + (b.P/w.W-py)*(b.P/w.W-py)
		if d <= max*max && (d < bd || d == bd && b.ID < best.ID) {
			best, bd = b, d
		}
	}
	return best
}

func (w *WorldData) ewDir(from, to int) (int, int) {
	return sign(to%w.W - from%w.W), sign(to/w.W - from/w.W)
}

// ewWarName finds the name the chronicle gives a war, from this line or an earlier declaration.
func (w *WorldData) ewWarName(idx, a, b int) string {
	parse := func(t string) string {
		if m := ewWarRe1.FindStringSubmatch(t); m != nil {
			return "the " + m[1]
		}
		if m := ewWarRe2.FindStringSubmatch(t); m != nil {
			return "the " + strings.TrimPrefix(m[1], "The ")
		}
		return ""
	}
	if idx >= 0 && idx < len(w.Events) {
		if n := parse(w.Events[idx].Text); n != "" {
			return n
		}
		an, bn := w.ewShort(a), w.ewShort(b)
		y := w.Events[idx].Y
		for j := idx - 1; j >= 0 && w.Events[j].Y >= y-25; j-- {
			e := w.Events[j]
			if e.Kind != "war" || an != "" && !strings.Contains(e.Text, an) || bn != "" && !strings.Contains(e.Text, bn) {
				continue
			}
			if n := parse(e.Text); n != "" {
				return n
			}
		}
	}
	if an, bn := w.ewShort(a), w.ewShort(b); an != "" && bn != "" {
		return fmt.Sprintf("the war between %s and %s", an, bn)
	}
	return "a war no one bothered to name"
}

func (w *WorldData) ewGeneral(state, year int) *PersonData {
	var best *PersonData
	for i := range w.People {
		pd := &w.People[i]
		if pd.State != state || pd.Role != "general" || pd.Born > year-16 || !pd.Alive && pd.Died < year {
			continue
		}
		if best == nil || pd.Martial > best.Martial || pd.Martial == best.Martial && pd.ID < best.ID {
			best = pd
		}
	}
	return best
}

// ewColour turns a state colour ("#a03020" or "hsl(170 59% 50%)") into a word for its standard.
func ewColour(c string) string {
	h, l := -1.0, 50.0
	if strings.HasPrefix(c, "hsl(") {
		f := strings.FieldsFunc(strings.TrimSuffix(strings.TrimPrefix(c, "hsl("), ")"), func(r rune) bool { return r == ' ' || r == ',' || r == '%' })
		if len(f) >= 3 {
			h, _ = strconv.ParseFloat(f[0], 64)
			l, _ = strconv.ParseFloat(f[2], 64)
		}
	} else if strings.HasPrefix(c, "#") && len(c) == 7 {
		v, err := strconv.ParseUint(c[1:], 16, 32)
		if err == nil {
			r, g, b := float64(v>>16&255)/255, float64(v>>8&255)/255, float64(v&255)/255
			mx, mn := math.Max(r, math.Max(g, b)), math.Min(r, math.Min(g, b))
			l = (mx + mn) * 50
			if mx-mn < 0.08 {
				h = -2
			} else {
				switch mx {
				case r:
					h = math.Mod((g-b)/(mx-mn), 6) * 60
				case g:
					h = ((b-r)/(mx-mn) + 2) * 60
				default:
					h = ((r-g)/(mx-mn) + 4) * 60
				}
				if h < 0 {
					h += 360
				}
			}
		}
	}
	word := "faded"
	switch {
	case h == -2 && l < 30:
		return "black"
	case h == -2 && l > 70:
		return "white"
	case h == -2:
		return "grey"
	case h < 0:
		return word
	case h < 15 || h >= 340:
		word = "red"
	case h < 45:
		word = "orange"
	case h < 70:
		word = "gold"
	case h < 165:
		word = "green"
	case h < 200:
		word = "sea-green"
	case h < 255:
		word = "blue"
	case h < 290:
		word = "purple"
	default:
		word = "crimson"
	}
	if l < 30 {
		word = "dark " + word
	}
	return word
}

// ---- painting ----

type ewPaint struct {
	zn      *Zone
	pt      *tgPainter
	floor   bool
	changed bool
}

func (p *ewPaint) ok(x, y int) bool {
	if x < 2 || y < 2 || x > MW-3 || y > MH-3 || !p.pt.ok(x, y) {
		return false
	}
	switch p.zn.At(x, y) {
	case TGrass, TTallGrass, TSand, TDirt, TSnow, TAsh, TMoss, TFungus, TCrystalFloor, TBush, TRubble, TBones, TDeadTree:
		return true
	case TFloor:
		return p.floor
	}
	return false
}

func (p *ewPaint) set(x, y int, t Terrain) bool {
	if !p.ok(x, y) {
		return false
	}
	if p.zn.At(x, y) != t {
		p.zn.Set(x, y, t)
		p.changed = true
	}
	return true
}

func (p *ewPaint) guard(x, y int) {
	if p.zn.In(x, y) {
		p.pt.prot[y*MW+x] = true
	}
}

type ewCtx struct {
	g   *Game
	w   *WorldData
	zn  *Zone
	r   *Rng
	p   *ewPaint
	sc  *ewScene
	rec *esRec
	era string
}

// centre finds an open stretch (rx by ry each way) of paintable ground.
func (c *ewCtx) centre(rx, ry int) (int, int, bool) {
	best, bx, by := -1, 0, 0
	for i := 0; i < 60; i++ {
		x, y := c.r.Int(rx+3, MW-rx-4), c.r.Int(ry+3, MH-ry-4)
		n := 0
		for dy := -ry; dy <= ry; dy++ {
			for dx := -rx; dx <= rx; dx++ {
				if c.p.ok(x+dx, y+dy) {
					n++
				}
			}
		}
		if n > best {
			best, bx, by = n, x, y
		}
	}
	return bx, by, best*2 >= (2*rx+1)*(2*ry+1)
}

// drop lays an item on the nearest free walkable dry tile within 3 of (x, y).
func (c *ewCtx) drop(it *Item, x, y int) bool {
	if it == nil {
		return false
	}
	zn := c.zn
	for rad := 0; rad <= 3; rad++ {
		for dy := -rad; dy <= rad; dy++ {
			for dx := -rad; dx <= rad; dx++ {
				if maxi(abs(dx), abs(dy)) != rad {
					continue
				}
				nx, ny := x+dx, y+dy
				if nx < 1 || ny < 1 || nx > MW-2 || ny > MH-2 || !zn.Walkable(nx, ny) || zn.WaterAt(nx, ny) > 0 || zn.ActorAt(nx, ny) != nil {
					continue
				}
				switch zn.At(nx, ny) {
				case TDown, TUp, TUpdraft, TPortal, TLava, TChasm, TVoid, TTrap:
					continue
				}
				it.X, it.Y = nx, ny
				zn.Items = append(zn.Items, it)
				c.p.guard(nx, ny)
				return true
			}
		}
	}
	return false
}

// stone carves an inscription on the nearest paintable tile.
func (c *ewCtx) stone(x, y int, text string) bool {
	for rad := 0; rad <= 3; rad++ {
		for dy := -rad; dy <= rad; dy++ {
			for dx := -rad; dx <= rad; dx++ {
				if maxi(abs(dx), abs(dy)) != rad || !c.p.ok(x+dx, y+dy) {
					continue
				}
				c.p.set(x+dx, y+dy, TInscription)
				c.zn.Notes[(y+dy)*MW+x+dx] = text
				c.p.guard(x+dx, y+dy)
				return true
			}
		}
	}
	return false
}

func (c *ewCtx) thing(name, lore string, glyph rune, f byte, weight int) *Item {
	return &Item{ID: c.g.newID(), Kind: "scrap", Name: name, Glyph: glyph, F: f, Value: 1, Weight: weight, Lore: lore, Ident: true}
}

func (c *ewCtx) remains(name, lore string) *Item {
	return c.thing(name, lore, '%', CWhite, 25)
}

func (c *ewCtx) note(name string, lines []string) *Item {
	it := c.g.MakeBook(name, strings.Join(lines, "\n"))
	it.Ident, it.Glyph, it.F, it.Value = true, '?', CYellow, 3
	c.rec.N = name
	return it
}

func (c *ewCtx) gold(n int) *Item {
	return &Item{ID: c.g.newID(), Kind: "gold", Name: c.g.Sk.Currency, Glyph: '$', F: CYellow, Count: maxi(1, n)}
}

func (c *ewCtx) sphereOf(state, culture int) string {
	if cu := c.w.cultByID[culture]; cu != nil && culture > 0 && cu.Sphere != "" {
		return cu.Sphere
	}
	return c.g.stateSphere(state)
}

func (c *ewCtx) cultOf(state int) int {
	if s := c.w.stateByID[state]; s != nil {
		return s.Culture
	}
	if c.w.ewLand(c.sc.P) {
		return c.w.Culture[c.sc.P]
	}
	return 0
}

func (c *ewCtx) cultName(cu int) string {
	if cd := c.w.cultByID[cu]; cd != nil {
		return cd.Name
	}
	return "local"
}

func (c *ewCtx) weaponName(sphere string) (string, string) {
	if k := wpSpheres[sphere]; k != nil && len(k.Weapons) > 0 {
		e := Pick(c.r, k.Weapons)
		return e.Name, e.Desc
	}
	if ns := c.g.Sk.Nouns["longblade"]; len(ns) > 0 {
		return Pick(c.r, ns), "A soldier's weapon."
	}
	return "blade", "A soldier's weapon."
}

func (c *ewCtx) armourName(sphere string) string {
	if k := wpSpheres[sphere]; k != nil && len(k.Armor) > 0 {
		return Pick(c.r, k.Armor).Name
	}
	return Pick(c.r, []string{"rotted leathers", "a rusted coat", "a padded jack"})
}

// broken: a culture's weapon, broken on the field.
func (c *ewCtx) broken(state int) *Item {
	name, desc := c.weaponName(c.sphereOf(state, 0))
	how := Pick(c.r, []string{"snapped at the hilt", "bent and notched", "broken in two", "rusted to the guard", "split along the haft"})
	side := c.w.ewShort(state)
	lore := fmt.Sprintf("%s This one is %s.", desc, how)
	if side != "" {
		lore += fmt.Sprintf(" %s work, carried for %s.", titleCase(c.cultName(c.cultOf(state))), side)
	}
	return c.thing("broken "+name, lore, '/', CDarkGray, 2)
}

func (c *ewCtx) person(cu int) string {
	cd := c.w.cultByID[cu]
	if cd == nil || len(cd.Given) == 0 {
		return Pick(c.r, []string{"Ashe", "Bren", "Coll", "Dara", "Edrin", "Fen", "Gave", "Hollis"})
	}
	n := Pick(c.r, cd.Given)
	if len(cd.Houses) > 0 && !strings.Contains(n, " ") && c.r.Chance(0.6) {
		n += " " + Pick(c.r, cd.Houses)
	}
	return n
}

// say speaks one line of a culture's voice (pack when present, else the built-in bank), placeholders filled.
func (c *ewCtx) say(cu int, kind string, vars map[string]string) string {
	w := c.w
	var list []string
	cd := w.cultByID[cu]
	if cd != nil && cd.Pack != nil && cd.Pack.Voice != nil {
		v := cd.Pack.Voice
		switch kind {
		case "oath":
			list = v.Oaths
		case "farewell":
			list = v.Farewell
		case "proverb":
			list = v.Proverbs
		case "curse":
			list = v.Curses
		}
	}
	rel := 0
	if w.ewLand(c.sc.P) {
		rel = w.Religion[c.sc.P]
	}
	if rd := w.relByID[rel]; rd != nil {
		if rd.Lore != nil && kind == "prayer" {
			list = rd.Lore.Prayers
		} else if rd.Lore != nil && kind == "afterlife" && rd.Lore.Afterlife != "" {
			list = []string{rd.Lore.Afterlife}
		}
		if vars["faith"] == "" {
			vars["faith"] = rd.Name
		}
		if vars["deity"] == "" {
			vars["deity"] = nonEmpty(rd.Deity, rd.Name)
		}
	}
	if vars["culture"] == "" && cd != nil {
		vars["culture"] = cd.Name
	}
	if vars["time"] == "" {
		vars["time"] = "night"
	}
	if len(list) > 0 {
		raw := w.cpChoose(list, "esw", kind, c.sc.ID)
		s := ewTidy(cpFill(raw, vars))
		if !strings.Contains(s, "{") && s != "" && !strings.Contains(s, "By,") && !strings.Contains(s, " ,") {
			return s
		}
	}
	return cpFill(w.cpChoose(ewVoiceBank[kind], "esw", kind, c.sc.ID), vars)
}

// ewTidy mends the gaps a dropped placeholder leaves ("then,, as" / "you , and").
func ewTidy(s string) string {
	for _, f := range [][2]string{{",,", ","}, {" ,", ","}, {"  ", " "}, {", .", "."}, {",.", "."}} {
		for strings.Contains(s, f[0]) {
			s = strings.ReplaceAll(s, f[0], f[1])
		}
	}
	return strings.TrimSpace(s)
}

func ewAgeLine(age int) string {
	switch {
	case age > 120:
		return "The letters are nearly gone; you trace them with a finger. "
	case age > 50:
		return "Moss has crept into the letters. "
	}
	return ""
}

// ---- the zone hook ----

func ewZone(g *Game, zn *Zone, _ *Rng) {
	w := g.W
	if w == nil || zn.Z != 0 || zn.Sky || zn.Underwater || !w.In(zn.WX, zn.WY) {
		return
	}
	scs := g.ewReg().byZone[[4]int{zn.WX, zn.WY, zn.ZX, zn.ZY}]
	for _, sc := range scs {
		if zn.Kind == "surface" || zn.Kind == "outskirts" || zn.Kind == "ruin" && sc.Kind == "fall" {
			g.ewBuildScene(zn, sc)
		}
	}
}

func (g *Game) ewBuildScene(zn *Zone, sc *ewScene) {
	w := g.W
	c := &ewCtx{g: g, w: w, zn: zn, r: NewRng(w.Seed + ":esw:" + sc.Kind + ":" + sc.ID), sc: sc, era: ewEra(g.Genre),
		p:   &ewPaint{zn: zn, pt: tgNewPainter(zn), floor: zn.Kind == "ruin"},
		rec: &esRec{ID: sc.ID, K: sc.Kind, Z: zn.Key, E: sc.E, P: sc.P, Y: sc.Year, Lead: -1}}
	ok := false
	switch sc.Kind {
	case "battle":
		ok = c.battle()
	case "siege":
		ok = c.siege()
	case "fall":
		ok = c.fall()
	case "stand":
		ok = c.stand()
	case "disaster":
		ok = c.disaster()
	case "plague":
		ok = c.plague()
	case "famine":
		ok = c.famine()
	}
	if c.p.changed {
		g.connect(zn)
		g.computeLight(zn)
	}
	if ok {
		ewSt(g).S[sc.ID] = c.rec
	}
}

// placeName: where a fight is remembered as having been.
func (c *ewCtx) placeName() (string, *BurgData) {
	w := c.w
	if bd := w.burgByID[w.Burg[c.sc.P]]; bd != nil {
		return bd.Name, bd
	}
	if bd := w.ewNear(c.sc.P, 4, false, 0); bd != nil {
		return bd.Name, bd
	}
	return titleCase(BiomeName(w.Biome[c.sc.P])) + " of " + w.StateName(w.State[c.sc.P]), nil
}

// ---- 1. battlefields ----

func (c *ewCtx) battle() bool {
	w, g, r := c.w, c.g, c.r
	e := w.Events[c.sc.E]
	att, def := w.ewSides(e)
	beast := ""
	win, lose := att, def
	if e.Kind == "monster" {
		beast = w.ewBeast(e.Text)
		win, lose = 0, att
		if ewBeastWonRe.MatchString(e.Text) {
			win, lose = att, 0
		}
	} else if def > 0 && ewRepulseRe.MatchString(e.Text) {
		win, lose = def, att
	}
	place, bd := c.placeName()
	battle := "the Battle of " + place
	if beast != "" {
		battle = "the fight with " + beast + " at " + place
	}
	war := ""
	if beast == "" {
		war = w.ewWarName(c.sc.E, att, def)
	}
	cx, cy, ok := c.centre(12, 5)
	if !ok {
		if cx, cy, ok = c.centre(8, 3); !ok {
			return false
		}
	}
	// the losers ran home: toward their capital, or away from the winners'
	fx, fy := 0, 0
	if lp := w.ewCapP(lose); lp >= 0 && lp != c.sc.P {
		fx, fy = w.ewDir(c.sc.P, lp)
	} else if wp := w.ewCapP(win); wp >= 0 && wp != c.sc.P {
		fx, fy = w.ewDir(wp, c.sc.P)
	}
	if fx == 0 && fy == 0 {
		fx, fy = dirs8[r.Int(0, 7)][0], dirs8[r.Int(0, 7)][1]
		if fx == 0 && fy == 0 {
			fx = 1
		}
	}
	dir := bhDir(fx, fy)
	// the line where they stood, across the line of flight
	px, py := -fy, fx
	L := 13
	if py != 0 {
		L = 6
	}
	var line [][2]int
	for t := -L; t <= L; t++ {
		j := r.Int(-1, 1)
		x, y := cx+px*t+fx*j, cy+py*t+fy*j
		line = append(line, [2]int{x, y})
		if r.Chance(0.6) {
			c.p.set(x, y, TBones)
		} else if r.Chance(0.4) {
			c.p.set(x, y, TRubble)
		}
		if r.Chance(0.3) {
			c.p.set(x-fx, y-fy, TBones)
		}
	}
	// the rout: bones thinning out the way they ran
	var rout [][2]int
	for s := 1; s <= 22; s++ {
		bx, by := cx+fx*s, cy+fy*s/2
		spread := 1 + s/5
		for k := 0; k < 2; k++ {
			o := r.Int(-spread, spread)
			x, y := bx+px*o, by+py*o
			if r.Chance(0.55 - float64(s)*0.02) {
				c.p.set(x, y, TBones)
			}
		}
		rout = append(rout, [2]int{bx, by})
	}
	at := func(pts [][2]int) (int, int) { q := Pick(r, pts); return q[0], q[1] }
	// broken gear of both sides on the line, the losers' strewn along the rout
	for k := 0; k < 5; k++ {
		side := lose
		if k < 2 && win > 0 {
			side = win
		}
		if side <= 0 {
			side = att
		}
		x, y := at(line)
		c.drop(c.broken(side), x, y)
	}
	for k := 0; k < 2; k++ {
		x, y := at(line)
		c.drop(c.thing(Pick(r, ewDebris[c.era]), fmt.Sprintf("Left where it fell in %s, Y%d.", battle, e.Y), '\\', CDarkGray, 3), x, y)
	}
	loserN := w.ewShort(lose)
	if loserN == "" {
		loserN = "the defenders"
	}
	winN := w.ewShort(win)
	if beast != "" && win == 0 {
		winN = beast
	}
	if winN == "" {
		winN = "the attackers"
	}
	lsph := c.sphereOf(lose, 0)
	for _, s := range []int{6, 13, 19} {
		if s-1 < len(rout) {
			x, y := rout[s-1][0], rout[s-1][1]
			c.drop(c.remains("a skeleton face down, still in "+c.armourName(lsph), fmt.Sprintf("Struck from behind while running %s. One of %s, by the gear.", dir, loserN)), x, y)
		}
	}
	if k := wpSpheres[lsph]; k != nil && k.Shield != nil && len(rout) > 9 {
		c.drop(c.thing(ewA(k.Shield.Name)+", thrown away", k.Shield.Desc+" Dropped by someone who wanted to run faster.", ')', CBrown, 4), rout[9][0], rout[9][1])
	}
	x, y := at(line)
	c.drop(c.thing(Pick(r, ewScav), "Scavengers had their turn after the burial parties.", ',', CDarkGray, 1), x, y)
	// the standard, planted where the line held (the winners') or trampled (the beast's victims)
	std := ""
	if s := w.stateByID[win]; s != nil && win > 0 {
		std = fmt.Sprintf("%s standard of %s", ewColour(s.Color), w.ewRealm(win))
		c.drop(&Item{ID: g.newID(), Kind: "trophy", Name: std, Glyph: '|', F: CYellow, Value: 30 + r.Int(0, 20), Weight: 6, Ident: true,
			Lore: fmt.Sprintf("Its pole was driven into the ground where the line held, in %s, Y%d. The cloth still bears the name of %s, faded by %d years of weather.", battle, e.Y, w.ewRealm(win), maxi(1, w.Year-e.Y))}, cx, cy)
	} else if s := w.stateByID[lose]; s != nil && lose > 0 {
		std = fmt.Sprintf("trampled %s standard of %s", ewColour(s.Color), w.ewRealm(lose))
		c.drop(&Item{ID: g.newID(), Kind: "trophy", Name: std, Glyph: '|', F: CYellow, Value: 25, Weight: 6, Ident: true,
			Lore: fmt.Sprintf("Pressed into the mud and left. The host of %s carried it against %s in Y%d.", w.ewRealm(lose), winN, e.Y)}, cx, cy)
	}
	// a cairn raised by the winners on home ground, else a grave dug by the locals
	gx, gy := cx-fx*5, cy-fy*3
	ep := ewAgeLine(w.Year-e.Y) + fmt.Sprintf("Here lie the dead of %s, Y%d.", battle, e.Y)
	if strings.HasPrefix(war, "the war between") {
		ep += fmt.Sprintf(" They fell in %s.", war)
	} else if war != "" {
		ep += fmt.Sprintf(" They fell in %s, %s against %s.", war, nonEmpty(w.ewShort(att), "the attackers"), nonEmpty(w.ewShort(def), "the defenders"))
	}
	local := c.cultOf(w.State[c.sc.P])
	if win > 0 && w.State[c.sc.P] == win {
		for _, d := range dirs8 {
			c.p.set(gx+d[0], gy+d[1], TRubble)
		}
		ep += fmt.Sprintf(" Raised by %s, who held the field.", w.ewShort(win))
	} else {
		for dx := -3; dx <= 3; dx++ {
			c.p.set(gx+dx, gy, TDirt)
			if dx%2 == 0 {
				c.p.set(gx+dx, gy+1, TBones)
			}
		}
		who := "the farmers hereabouts"
		if bd != nil {
			who = "the people of " + bd.Name
		}
		ep += fmt.Sprintf(" Dug by %s, for both sides alike.", who)
	}
	ep += " " + c.say(local, "prayer", map[string]string{"town": place})
	c.stone(gx, gy-1, ep)
	// a trophy someone missed
	var trophy *Item
	if beast != "" {
		trophy = &Item{ID: g.newID(), Kind: "trophy", Name: "a scale of " + beast, Glyph: '♦', F: CGreen, Value: 60, Weight: 3, Ident: true,
			Lore: fmt.Sprintf("Torn off in %s, Y%d, and trodden into the ground.", battle, e.Y)}
	} else {
		trophy = g.MakeSphereWeapon(lsph, -1, 1+r.Int(0, 1), r)
		trophy.Ident = true
		trophy.Lore = strings.TrimSpace(trophy.Lore + fmt.Sprintf(" Found on the field of %s, where %s broke in Y%d.", place, loserN, e.Y))
	}
	x, y = at(rout[:12])
	c.drop(trophy, x, y)
	// the roll of a company
	side := lose
	if side <= 0 {
		side = att
	}
	cu := c.cultOf(side)
	ord := ewOrdinal[r.Int(0, len(ewOrdinal)-1)]
	roll := fmt.Sprintf("%s of the %s company, %s", ewRoll[c.era], ord, place)
	n0 := 60 + r.Int(0, 140)
	n1 := n0 * (20 + r.Int(0, 35)) / 100
	names := []string{c.person(cu), c.person(cu), c.person(cu), c.person(cu)}
	lines := []string{
		fmt.Sprintf("Roll of the %s company of %s at %s, Y%d.", ord, nonEmpty(w.ewShort(side), "the host"), place, e.Y),
		fmt.Sprintf("Mustered at dawn: %d. Answering at dusk: %d.", n0, n1),
		fmt.Sprintf("Lost: %s, %s, %s and %s.", names[0], names[1], names[2], names[3]),
	}
	capN := nonEmpty(w.ewCapName(lose), "home")
	switch {
	case beast != "" && win == 0:
		lines = append(lines, fmt.Sprintf("%s came over the ridge at noon. Spears did nothing to it. We go back %s to %s.", beast, dir, capN))
	case beast != "":
		lines = append(lines, fmt.Sprintf("%s is down. Half of us are with it.", beast))
	case side == lose:
		lines = append(lines, fmt.Sprintf("%s holds the field. We fall back %s, toward %s.", winN, dir, capN))
	default:
		lines = append(lines, fmt.Sprintf("%s broke and ran %s. The field is ours.", ewCap(loserN), dir))
	}
	lines = append(lines, c.say(cu, "oath", map[string]string{"realm": w.ewShort(side), "town": place}))
	x, y = at(line)
	c.drop(c.note(roll, lines), x, y)

	rc := c.rec
	rc.CX, rc.CY = cx, cy
	rc.T = "Field of " + place
	if beast != "" {
		rc.T = "Where " + beast + " was fought"
	}
	rc.L = fmt.Sprintf("Bones lie in a ragged line across the ground and thin out to the %s: a line broke here and ran.", dir)
	if std != "" {
		rc.L += " " + ewCap(ewA(std)) + " still marks the spot."
	}
	rc.H = "Crows lift from the ground somewhere nearby. Something was fought over here."
	if war != "" {
		rc.F = fmt.Sprintf("%s, Y%d, in %s: %s held the field and %s fled %s", ewCap(battle), e.Y, war, winN, loserN, dir)
		if capN != "home" {
			rc.F += " toward " + capN
		}
		rc.F += "."
	} else {
		rc.F = fmt.Sprintf("%s, Y%d: %s held the field and %s fled %s.", ewCap(battle), e.Y, winN, loserN, dir)
	}
	rc.S = rc.F
	rc.Src = "the field of " + place
	if bd != nil {
		rc.B = bd.ID
	}
	// a lead: the next field of the same war, else where the losers went
	if q := g.ewSibling(c.sc, att, def); q != nil {
		rc.Lead = q.P
		rc.LT = fmt.Sprintf("The same war was fought again near %s.", w.RegionName(q.P))
	} else if lp := w.ewCapP(lose); lp >= 0 && lp != c.sc.P {
		rc.Lead = lp
		rc.LT = fmt.Sprintf("The roll names %s as the place the survivors fell back to.", capN)
	}
	return true
}

// ewA puts "a" or "an" before a noun phrase.
func ewA(s string) string {
	if s != "" && strings.ContainsRune("aeiouAEIOU", rune(s[0])) {
		return "an " + s
	}
	return "a " + s
}

func ewCap(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// ewSibling: the nearest other battle or siege scene between the same realms.
func (g *Game) ewSibling(sc *ewScene, a, b int) *ewScene {
	w := g.W
	an, bn := w.ewShort(a), w.ewShort(b)
	if an == "" {
		return nil
	}
	var best *ewScene
	bd := 1 << 30
	for _, o := range g.ewReg().list {
		if o == sc || o.E < 0 || o.P == sc.P || o.Kind != "battle" && o.Kind != "siege" {
			continue
		}
		t := w.Events[o.E].Text
		if !strings.Contains(t, an) || bn != "" && !strings.Contains(t, bn) {
			continue
		}
		d := abs(o.P%w.W-sc.P%w.W) + abs(o.P/w.W-sc.P/w.W)
		if d < bd {
			best, bd = o, d
		}
	}
	return best
}

// ---- 2. sieges ----

func (c *ewCtx) siege() bool {
	w, g, r := c.w, c.g, c.r
	e := w.Events[c.sc.E]
	bd := w.burgByID[w.Burg[c.sc.P]]
	if bd == nil {
		return false
	}
	att, def := w.ewSides(e)
	attN := nonEmpty(w.ewShort(att), "the besiegers")
	defN := w.ewShort(def)
	if defN == "" || defN == bd.Name {
		defN = "the garrison of " + bd.Name
	}
	tdx, tdy := sign(1-c.zn.ZX), sign(1-c.zn.ZY)
	cx, cy, ok := c.centre(9, 4)
	if !ok {
		return false
	}
	walled := bd.Alive && g.cityPlanFor(bd).walled
	wallT, wallW := TWoodWall, "palisade"
	if walled {
		wallT, wallW = TStoneWall, "outer wall"
	}
	// the outer works facing the town, breached in the middle
	wx := clampi(cx+tdx*18, 6, MW-7)
	wy := clampi(cy+tdy*7, 4, MH-5)
	var breach [][2]int
	if tdx != 0 && tdy == 0 {
		for y := wy - 4; y <= wy+4; y++ {
			if abs(y-wy) <= 1 {
				c.p.set(wx, y, TRubble)
				breach = append(breach, [2]int{wx, y})
			} else if r.Chance(0.85) {
				c.p.set(wx, y, wallT)
			}
		}
	} else {
		for x := wx - 8; x <= wx+8; x++ {
			if abs(x-wx) <= 2 {
				c.p.set(x, wy, TRubble)
				breach = append(breach, [2]int{x, wy})
			} else if r.Chance(0.85) {
				c.p.set(x, wy, wallT)
			}
		}
	}
	for _, b := range breach {
		for k := 0; k < 3; k++ {
			c.p.set(b[0]-tdx*r.Int(1, 3)+r.Int(-1, 1), b[1]-tdy*r.Int(1, 2), TRubble)
		}
	}
	c.drop(c.thing(ewMissile[c.era], fmt.Sprintf("From the %s before %s, Y%d.", wallW, bd.Name, e.Y), ',', CGray, 1), breach[0][0]-tdx, breach[0][1]-tdy)
	// the besiegers' camp: a ring of tent scars around cold fires
	for k := 0; k < 36; k++ {
		a := float64(k) * math.Pi * 2 / 36
		x, y := cx+int(math.Round(math.Cos(a)*8)), cy+int(math.Round(math.Sin(a)*4))
		if k%6 < 3 {
			c.p.set(x, y, TTent)
		}
	}
	for k := 0; k < 3; k++ {
		fx, fy := cx+r.Int(-5, 5), cy+r.Int(-2, 2)
		c.p.set(fx, fy, TAsh)
		c.p.set(fx+1, fy, TAsh)
		c.drop(c.thing(Pick(r, ewCamp[c.era]), fmt.Sprintf("Left in the camp of %s when the siege of %s ended.", attN, bd.Name), '&', CBrown, 2), fx, fy+1)
	}
	// the engine, burnt where it stood, between camp and wall
	ex, ey := (cx+wx)/2, (cy+wy)/2
	eng := Pick(r, ewEngine[c.era])
	for dy := -2; dy <= 2; dy++ {
		for dx := -4; dx <= 4; dx++ {
			if r.Chance(0.6) {
				c.p.set(ex+dx, ey+dy, TAsh)
			}
		}
	}
	c.p.set(ex, ey, TRubble)
	c.drop(c.thing(eng, fmt.Sprintf("It battered the %s of %s in Y%d. Someone burned it after, so it would not be used again.", wallW, bd.Name, e.Y), 'Ψ', CDarkGray, 40), ex, ey)
	// orders of the day
	cmd := w.ewGeneral(att, e.Y)
	cmdN := "the captain"
	if cmd != nil {
		cmdN = cmd.Name
		if cmd.Title != "" {
			cmdN += ", " + cmd.Title
		}
	}
	side := bhDir(tdx, tdy)
	cu := c.cultOf(att)
	lines := []string{
		fmt.Sprintf("Camp before %s, Y%d. Orders of %s.", bd.Name, e.Y, cmdN),
		fmt.Sprintf("The %s goes forward at first light against the %s %s.", ewShortEngine(eng), side, wallW),
		fmt.Sprintf("%s will not yield, so no quarter until the gate is ours.", ewCap(defN)),
	}
	if ewSackRe.MatchString(e.Text) && strings.Contains(strings.ToLower(e.Text), "vault") || strings.Contains(strings.ToLower(e.Text), "plunder") {
		lines = append(lines, "The vaults belong to the crown. Anyone caught with plate in their pack hangs at the gate.")
	}
	lines = append(lines, c.say(cu, "oath", map[string]string{"realm": attN, "town": bd.Name}))
	c.drop(c.note("orders of the day before "+bd.Name, lines), cx, cy)
	loot := g.Loot(r, 1+r.Int(0, 1))
	loot.Lore = strings.TrimSpace(loot.Lore + fmt.Sprintf(" Looted from %s in Y%d and dropped in the rush.", bd.Name, e.Y))
	c.drop(loot, breach[len(breach)/2][0]-tdx*2, breach[len(breach)/2][1]-tdy)
	c.drop(c.gold(12+r.Int(0, 30)), ex+r.Int(-2, 2), ey)

	rc := c.rec
	rc.CX, rc.CY = cx, cy
	rc.T = "Siege lines of " + bd.Name
	rc.L = fmt.Sprintf("A ring of tent scars and cold fires, %s, and a broken stretch of %s facing %s.", eng, wallW, bd.Name)
	rc.H = "Old earthworks cross the ground here, and the grass grows strangely over a ring of scars."
	rc.F = fmt.Sprintf("%s took %s from %s in Y%d after a siege; the breach was made in the %s %s.", ewCap(attN), bd.Name, defN, e.Y, side, wallW)
	rc.S = rc.F
	rc.Src = "the siege lines of " + bd.Name
	rc.B = bd.ID
	return true
}

func ewShortEngine(s string) string {
	for _, k := range []string{"trebuchet", "ram", "mangonel", "mortar", "cannon", "engine", "rocket rail", "reactor", "walker", "railgun", "drone", "truck", "catapult"} {
		if strings.Contains(s, k) {
			return k
		}
	}
	return "engine"
}

// ---- 3. fallen cities ----

func (c *ewCtx) fall() bool {
	w, g, r := c.w, c.g, c.r
	e := w.Events[c.sc.E]
	bd := w.burgByID[w.Burg[c.sc.P]]
	city := w.RegionName(c.sc.P)
	if bd != nil {
		city = bd.Name
	}
	fallen := e.St
	// who sacked it: the capture of this town shortly before, else the other realm named
	sacker, sackY := 0, e.Y
	for j := c.sc.E - 1; j >= 0 && w.Events[j].Y >= e.Y-12; j-- {
		x := w.Events[j]
		if x.Kind == "capture" && (x.P == c.sc.P || strings.Contains(x.Text, city)) {
			sacker, sackY = x.St, x.Y
			break
		}
	}
	if sacker == 0 {
		for _, id := range w.ewNamed(e.Text) {
			if id != fallen {
				sacker = id
				break
			}
		}
	}
	sackN := nonEmpty(w.ewShort(sacker), "their enemies")
	fug := ""
	if m := ewFleeRe.FindStringSubmatch(e.Text); m != nil {
		fug = m[1]
	}
	// a few burnt houses
	type rect struct{ x, y, w, h int }
	var hs []rect
	for k := 0; k < 120 && len(hs) < 3; k++ {
		rw, rh := r.Int(8, 11), r.Int(5, 6)
		if k >= 60 {
			rw, rh = r.Int(5, 7), 4
		}
		x, y := r.Int(4, MW-rw-5), r.Int(3, MH-rh-4)
		good := true
		for _, h := range hs {
			if x < h.x+h.w+2 && h.x < x+rw+2 && y < h.y+h.h+2 && h.y < y+rh+2 {
				good = false
			}
		}
		for yy := y; yy < y+rh && good; yy++ {
			for xx := x; xx < x+rw; xx++ {
				if !c.p.ok(xx, yy) {
					good = false
					break
				}
			}
		}
		if good {
			hs = append(hs, rect{x, y, rw, rh})
		}
	}
	if len(hs) == 0 {
		return false
	}
	for _, h := range hs {
		door := [2]int{h.x + h.w/2, h.y + h.h - 1}
		if r.Chance(0.5) {
			door = [2]int{h.x, h.y + h.h/2}
		}
		for y := h.y; y < h.y+h.h; y++ {
			for x := h.x; x < h.x+h.w; x++ {
				edge := y == h.y || y == h.y+h.h-1 || x == h.x || x == h.x+h.w-1
				switch {
				case x == door[0] && y == door[1]:
					c.p.set(x, y, TRubble)
				case edge && r.Chance(0.6):
					c.p.set(x, y, TStoneWall)
				case edge:
					c.p.set(x, y, TRubble)
				case r.Chance(0.35):
					c.p.set(x, y, TRubble)
				case r.Chance(0.4):
					c.p.set(x, y, TAsh)
				default:
					c.p.set(x, y, TDirt)
				}
			}
		}
	}
	cu := w.Culture[c.sc.P]
	if bd != nil {
		cu = bd.Culture
	}
	fam := c.person(cu)
	if cd := w.cultByID[cu]; cd != nil && len(cd.Houses) > 0 {
		fam = Pick(r, cd.Houses)
	} else if i := strings.LastIndex(fam, " "); i > 0 {
		fam = fam[i+1:]
	}
	h0 := hs[0]
	c.stone(h0.x+1, h0.y+1, ewAgeLine(w.Year-sackY)+fmt.Sprintf("Here stood the house of the %s family of %s. It burned in Y%d, when %s took %s.", fam, city, sackY, sackN, city))
	// the stash, in a corner of the last house
	hl := hs[len(hs)-1]
	sx, sy := hl.x+hl.w-2, hl.y+1
	refuge := w.ewNear(c.sc.P, 12, true, func() int {
		if bd != nil {
			return bd.ID
		}
		return 0
	}())
	refN, dir := "the hills", "away"
	if refuge != nil {
		refN = refuge.Name
		dir = bhDir(refuge.P%w.W-c.sc.P%w.W, refuge.P/w.W-c.sc.P/w.W)
	}
	writer := c.person(cu)
	lines := []string{fmt.Sprintf("We leave tonight with what we can carry. The soldiers of %s are in the streets of %s.", sackN, city)}
	if fug != "" {
		lines = append(lines, fmt.Sprintf("They say %s has fled into the wilds, and the court with them.", fug))
	}
	lines = append(lines,
		fmt.Sprintf("We go %s to %s, where my mother's people are.", dir, refN),
		"The coin is for whoever buries what is left of our dead.",
		c.say(cu, "proverb", map[string]string{"town": city, "realm": w.ewShort(fallen)}),
		fmt.Sprintf("%s, of the %s house, Y%d.", writer, fam, sackY))
	c.drop(c.note(fmt.Sprintf("note in the %s family's hiding place", fam), lines), sx, sy)
	c.drop(c.gold(20+r.Int(0, 45)), sx, sy)
	heir := c.thing(Pick(r, []string{"a silver spoon wrapped in cloth", "a family seal wrapped in wool", "a string of amber beads", "a small brass mirror"}),
		fmt.Sprintf("Hidden by the %s family of %s when %s came, Y%d.", fam, city, sackN, sackY), '*', CYellow, 1)
	heir.Value = 15 + r.Int(0, 20)
	c.drop(heir, sx, sy)
	c.drop(g.Loot(r, 1+r.Int(0, 1)), sx, sy)

	rc := c.rec
	rc.CX, rc.CY = h0.x+h0.w/2, h0.y+h0.h/2
	rc.T = "Burnt houses of " + city
	rc.L = fmt.Sprintf("Burnt walls stand open to the sky; ash and broken roof tiles fill the rooms. This is what %s left of %s.", sackN, city)
	rc.H = "Blackened walls show above the grass nearby."
	realm := nonEmpty(w.ewRealm(fallen), city)
	rc.F = fmt.Sprintf("%s fell in Y%d, after %s took %s.", realm, e.Y, sackN, city)
	if fug != "" {
		rc.F += fmt.Sprintf(" %s fled into the wilds.", fug)
	}
	rc.F += fmt.Sprintf(" Families of %s fled %s to %s.", city, dir, refN)
	rc.S = fmt.Sprintf("The %s family hid their coin in their burnt house outside %s before fleeing to %s.", fam, city, refN)
	rc.Src = "a note in the ruins of " + city
	if bd != nil {
		rc.B = bd.ID
	}
	if refuge != nil {
		rc.Lead = refuge.P
		rc.LT = fmt.Sprintf("The %s family went to %s. Someone there may remember them.", fam, refN)
	}
	return true
}

// ---- 4. last stands ----

func (c *ewCtx) stand() bool {
	w, g, r := c.w, c.g, c.r
	pd := &w.People[c.sc.Per]
	place, bd := c.placeName()
	foe, beast := "", ""
	if c.sc.E >= 0 {
		e := w.Events[c.sc.E]
		if e.Kind == "monster" {
			beast = w.ewBeast(e.Text)
			foe = beast
		} else {
			att, def := w.ewSides(e)
			if att == pd.State {
				foe = w.ewShort(def)
			} else {
				foe = w.ewShort(att)
			}
		}
	}
	if foe == "" {
		foe = "the enemy"
	}
	cx, cy, ok := c.centre(7, 3)
	if !ok {
		return false
	}
	title := nonEmpty(pd.Title, pd.Role)
	// the guard who did not run, in a ring around the body; the enemy dead further out
	for k := 0; k < 10; k++ {
		a := float64(k) * math.Pi * 2 / 10
		c.p.set(cx+int(math.Round(math.Cos(a)*3)), cy+int(math.Round(math.Sin(a)*2)), TBones)
	}
	for k := 0; k < 14; k++ {
		a := float64(k)*math.Pi*2/14 + r.Float()*0.3
		if r.Chance(0.7) {
			c.p.set(cx+int(math.Round(math.Cos(a)*float64(6+r.Int(0, 2)))), cy+int(math.Round(math.Sin(a)*3)), TBones)
		}
	}
	c.p.set(cx, cy, TBones)
	sph := c.sphereOf(pd.State, pd.Culture)
	c.drop(c.remains(fmt.Sprintf("the bones of %s, %s", pd.Name, title), fmt.Sprintf("Killed at %s in Y%d, at %d. The ring of guards around them faced outward.", place, pd.Died, maxi(0, pd.Died-pd.Born))), cx, cy)
	wpn := g.MakeSphereWeapon(sph, -1, 2+r.Int(0, 1), r)
	wpn.Ident = true
	wpn.Lore = strings.TrimSpace(wpn.Lore + fmt.Sprintf(" It belonged to %s, %s, who died with it in hand at %s in Y%d.", pd.Name, title, place, pd.Died))
	c.drop(wpn, cx+1, cy)
	if arm := g.MakeSphereArmor(sph, false, 2, r); arm != nil {
		arm.Ident = true
		arm.Lore = strings.TrimSpace(arm.Lore + fmt.Sprintf(" Worn by %s at the end. The back is unmarked.", pd.Name))
		c.drop(arm, cx-1, cy)
	}
	c.drop(&Item{ID: g.newID(), Kind: "trophy", Name: "signet of " + pd.Name, Glyph: 'o', F: CYellow, Value: 50 + pd.Renown, Weight: 1, Ident: true,
		Lore: fmt.Sprintf("The seal of %s, %s. It is worth more to their house than to a jeweller.", pd.Name, title)}, cx, cy+1)
	for k := 0; k < 2; k++ {
		a := float64(k*5) * math.Pi * 2 / 10
		c.drop(c.remains("a guard's skeleton, facing outward", fmt.Sprintf("Sworn to %s, and kept to it.", pd.Name)), cx+int(math.Round(math.Cos(a)*3)), cy+int(math.Round(math.Sin(a)*2)))
	}
	// what the letter points at: the traitor house, the heir, or where the realm's treasure lies
	heir, house := "", ""
	for i, e := range w.Events {
		if e.Kind != "death" || abs(e.Y-pd.Died) > 1 || !strings.Contains(e.Text, pd.Name) {
			continue
		}
		_ = i
		if m := ewHouseRe.FindStringSubmatch(e.Text); m != nil {
			house = m[1]
		}
		for _, re := range ewHeirRe {
			if m := re.FindStringSubmatch(e.Text); m != nil && m[1] != "The" {
				heir = m[1]
				break
			}
		}
		break
	}
	var art *ArtifactData
	for i := range w.Artifacts {
		a := &w.Artifacts[i]
		if a.HolderType == "burg" {
			if hb := w.burgByID[a.HolderID]; hb != nil && hb.State == pd.State && hb.Alive {
				art = a
				break
			}
		}
	}
	capN := nonEmpty(w.ewCapName(pd.State), "home")
	to := "the council of " + capN
	if heir != "" && heir != pd.Name {
		to = heir
	}
	cu := pd.Culture
	lines := []string{
		fmt.Sprintf("To %s,", to),
		fmt.Sprintf("We are cut off at %s. %s are on three sides of us, and the fourth is no better.", place, ewCap(foe)),
	}
	if beast != "" {
		lines[1] = fmt.Sprintf("We are cut off at %s. %s came down on the camp at dusk, and there is no outrunning it.", place, beast)
	}
	rc := c.rec
	switch {
	case house != "":
		lines = append(lines, fmt.Sprintf("If I do not come back, do not trust House %s. Their men were held back today, and they knew why.", house))
		rc.F = fmt.Sprintf("%s, %s, wrote before dying at %s (Y%d) that House %s held back its men. House %s took the throne after.", pd.Name, title, place, pd.Died, house, house)
	case heir != "" && heir != pd.Name:
		lines = append(lines, fmt.Sprintf("%s must take the seal at once. Do not let the council wait for my body.", heir))
		rc.F = fmt.Sprintf("%s, %s, named %s heir in a last letter from %s, Y%d.", pd.Name, title, heir, place, pd.Died)
	case art != nil:
		hb := w.burgByID[art.HolderID]
		lines = append(lines, fmt.Sprintf("The %s is safe in %s. Whatever happens here, do not let them move it.", art.Name, hb.Name))
		rc.F = fmt.Sprintf("%s, %s, wrote before dying in Y%d that the %s is kept in %s.", pd.Name, title, pd.Died, art.Name, hb.Name)
		rc.Lead = hb.P
		rc.LT = fmt.Sprintf("The letter says the %s is kept in %s.", art.Name, hb.Name)
	default:
		lines = append(lines, fmt.Sprintf("Tell %s we held as long as we could.", capN))
		rc.F = fmt.Sprintf("%s, %s, fell at %s in Y%d, fighting %s.", pd.Name, title, place, pd.Died, foe)
	}
	lines = append(lines, "If this reaches you, I did not.",
		c.say(cu, "farewell", map[string]string{"name": heir, "realm": w.ewShort(pd.State)}),
		fmt.Sprintf("%s, %s", pd.Name, title))
	c.drop(c.note("last letter of "+pd.Name, lines), cx, cy-1)

	rc.CX, rc.CY = cx, cy
	rc.T = "Last stand of " + pd.Name
	rc.L = fmt.Sprintf("A tight ring of bones around one body: guards who did not run, and the one they guarded. Further out lie the dead of %s.", foe)
	if beast != "" {
		rc.L = fmt.Sprintf("A tight ring of bones around one body: guards who did not run, and the one they guarded. The ground around them is torn up by something far bigger than a horse: %s.", beast)
	}
	rc.H = "Crows circle one spot nearby, and keep coming back to it."
	rc.S = fmt.Sprintf("%s, %s, died at %s in Y%d (%s).", pd.Name, title, place, pd.Died, pd.Cause)
	rc.Src = "the last letter of " + pd.Name
	if bd != nil {
		rc.B = bd.ID
	} else {
		rc.B = pd.Burg
	}
	if rc.Lead < 0 {
		if cp := w.ewCapP(pd.State); cp >= 0 && cp != c.sc.P {
			rc.Lead = cp
			rc.LT = fmt.Sprintf("The letter was meant for %s.", capN)
		}
	}
	return true
}

// ---- 5. disasters, plague pits and famine roads ----

func (c *ewCtx) disaster() bool {
	w, r := c.w, c.r
	kind, label, sev := c.sc.Dis, "", 3.0
	var towns []int
	if c.sc.Hz >= 0 {
		he := &w.Hazards.Recent[c.sc.Hz]
		label, sev, towns = hzLabel(he), he.Severity, he.Towns
	} else {
		label = fmt.Sprintf("the %s in Y%d", hzKindName[kind], c.sc.Year)
		if bd := w.burgByID[w.Burg[c.sc.P]]; bd != nil {
			towns = []int{bd.ID}
		}
	}
	if len(towns) == 0 {
		if bd := w.ewNear(c.sc.P, 4, false, 0); bd != nil {
			towns = []int{bd.ID}
		}
	}
	if len(towns) == 0 || ewDisRemains[kind] == nil {
		return false
	}
	t0 := w.burgByID[towns[0]]
	if t0 == nil {
		return false
	}
	cx, cy, ok := c.centre(5, 2)
	if !ok {
		return false
	}
	// the dead in a row, the things they were carrying
	rem := ewDisRemains[kind]
	for k := 0; k < 3; k++ {
		c.drop(c.remains(rem[k%len(rem)], fmt.Sprintf("Laid out here after %s and never collected.", label)), cx-3+k*3, cy)
	}
	c.drop(c.thing(ewCart[c.era], "Loaded in a hurry and left in a hurry.", '&', CBrown, 30), cx+5, cy+1)
	c.drop(c.thing(Pick(r, ewToy[c.era]), "Small enough to hold in one hand.", '*', CBrown, 1), cx-5, cy+1)
	// the tally
	cu := t0.Culture
	reeve := c.person(cu)
	lines := []string{fmt.Sprintf("Cut by %s, reeve of %s, after %s.", reeve, t0.Name, label)}
	total := 0
	for i, t := range towns {
		if i >= 3 {
			break
		}
		b := w.burgByID[t]
		if b == nil {
			continue
		}
		n := int(sev*2) + 3 + r.Int(0, 9) + mini(80, int(b.Pop/12))
		if kind == "locust" {
			n = 2 + r.Int(0, 6) + mini(20, int(b.Pop/40))
		}
		total += n
		lines = append(lines, fmt.Sprintf("%s: %d notches.", b.Name, n))
	}
	miss := r.Int(2, 12)
	lines = append(lines, fmt.Sprintf("Missing, not counted: %d. %s took them and has not given them back.", miss, ewCap(ewDisWhat[kind])),
		c.say(cu, "prayer", map[string]string{"town": t0.Name}))
	c.drop(c.note("notched tally of the dead of "+t0.Name, lines), cx, cy+1)
	c.drop(c.gold(4+r.Int(0, 14)), cx+1, cy-1)

	rc := c.rec
	rc.CX, rc.CY = cx, cy
	rc.T = titleCase(hzKindName[kind]) + " dead of " + t0.Name
	rc.L = fmt.Sprintf("Someone laid out the dead here after %s and covered them with whatever came to hand. Nobody came back for them.", label)
	rc.H = "There is a smell on the wind here, old and sweet."
	rc.F = fmt.Sprintf("%s killed at least %d in and around %s; %d more were never found.", ewCap(label), total, t0.Name, miss)
	rc.S = rc.F
	rc.Src = "the reeve's tally of " + t0.Name
	rc.B = t0.ID
	return true
}

func (c *ewCtx) plague() bool {
	w, g, r := c.w, c.g, c.r
	name, year := "the sickness", c.sc.Year
	var dz *HzDisease
	if c.sc.Ob >= 0 {
		ob := &w.Hazards.Outbreaks[c.sc.Ob]
		name = ob.Name
		dz = w.hzDisease(ob.Disease)
	} else if c.sc.E >= 0 {
		if n := ewSickName1(w.Events[c.sc.E].Text); n != "" {
			name = "the " + n
		}
	}
	bd := w.burgByID[w.Burg[c.sc.P]]
	if bd == nil {
		bd = w.ewNear(c.sc.P, 4, false, 0)
	}
	if bd == nil {
		return false
	}
	cx, cy, ok := c.centre(5, 2)
	if !ok {
		return false
	}
	for dx := -4; dx <= 4; dx++ {
		for dy := -1; dy <= 1; dy++ {
			if dy == 0 && abs(dx) <= 3 {
				c.p.set(cx+dx, cy+dy, TBones)
			} else {
				c.p.set(cx+dx, cy+dy, TDirt)
			}
		}
	}
	c.drop(c.thing("a split sack of quicklime", "White powder has spilled over the edge of the pit. It burns the fingers.", '%', CWhite, 8), cx-5, cy)
	c.drop(c.thing("a split sack of quicklime", "Half used. The rest has set hard in the rain.", '%', CWhite, 8), cx+5, cy)
	c.drop(c.thing("a long-handled hook", "For pulling the dead off the cart without touching them.", '/', CBrown, 4), cx+4, cy+2)
	cu := bd.Culture
	clergy := "priest"
	faith := ""
	if rd := w.relByID[w.Religion[c.sc.P]]; rd != nil {
		faith = rd.Name
		if rd.Lore != nil && rd.Lore.Clergy != nil && rd.Lore.Clergy.Title != "" {
			clergy = rd.Lore.Clergy.Title
		}
	}
	c.drop(c.thing(ewA(clergy+"'s stole folded on a stone"), "Taken off to work, and never taken up again.", '~', CDarkRed, 1), cx-4, cy+2)
	n := 20 + r.Int(0, 60)
	advice := "Burn the bedding. Boil the water. Keep the sick apart."
	if dz != nil {
		switch v := strings.ToLower(dz.Vector); {
		case strings.Contains(v, "water"):
			advice = "Do not drink from the river below the town. Boil everything."
		case strings.Contains(v, "air"), strings.Contains(v, "breath"):
			advice = "Keep the sick apart; it rides on their breath. Wrap your face."
		case strings.Contains(v, "flea"), strings.Contains(v, "rat"), strings.Contains(v, "vermin"):
			advice = "Kill the rats. Burn the straw they nest in."
		case strings.Contains(v, "touch"), strings.Contains(v, "contact"), strings.Contains(v, "blood"):
			advice = "Do not touch the dead without waxed gloves. Burn the gloves after."
		}
	}
	priest := c.person(cu)
	lines := []string{
		fmt.Sprintf("%d laid here in lime, Y%d, dead of %s.", n, year, name),
		advice,
		c.say(cu, "prayer", map[string]string{"town": bd.Name}),
		c.say(cu, "afterlife", map[string]string{"town": bd.Name}),
		fmt.Sprintf("%s, %s%s, at %s.", priest, clergy, func() string {
			if faith != "" {
				return " of " + faith
			}
			return ""
		}(), bd.Name),
	}
	c.drop(c.note(clergy+"'s note at the lime pit of "+bd.Name, lines), cx, cy+2)

	rc := c.rec
	rc.CX, rc.CY = cx, cy
	rc.T = "Plague pit of " + bd.Name
	rc.L = fmt.Sprintf("A long pit, filled and sunk again, white with lime at the edges. The dead of %s went in here.", name)
	rc.H = "A white crust shows through the grass nearby, and nothing grows on it."
	rc.F = fmt.Sprintf("%s came to %s in Y%d; %d dead went into a lime pit outside the town.", ewCap(name), bd.Name, year, n)
	rc.S = fmt.Sprintf("The %s at %s's lime pit wrote: %s", clergy, bd.Name, advice)
	rc.Src = "the " + clergy + " at the lime pit"
	rc.B = bd.ID
	_ = g
	return true
}

func (c *ewCtx) famine() bool {
	w, r := c.w, c.r
	e := w.Events[c.sc.E]
	bd := w.burgByID[w.Burg[c.sc.P]]
	if bd == nil {
		bd = w.ewNear(c.sc.P, 4, false, 0)
	}
	if bd == nil {
		return false
	}
	st := e.St
	if w.stateByID[st] == nil {
		st = w.State[c.sc.P]
	}
	choice := ""
	if s := w.stateByID[st]; s != nil && s.Famine != nil && abs(s.Famine.Year-e.Y) <= 4 && s.Famine.Text != "" {
		choice = s.Famine.Text
		if s.Famine.Ruler != "" {
			choice += " by order of " + s.Famine.Ruler
		}
	}
	// the road out: the zone's own road if it has one, else a trodden track
	var road [][2]int
	for y := 3; y < MH-3; y++ {
		for x := 3; x < MW-3; x++ {
			if c.zn.At(x, y) == TRoad && c.p.pt.ok(x, y) {
				road = append(road, [2]int{x, y})
			}
		}
	}
	if len(road) < 10 {
		cx, cy, ok := c.centre(14, 1)
		if !ok {
			return false
		}
		road = nil
		for x := cx - 14; x <= cx+14; x++ {
			y := cy
			if (x/5)%2 == 0 {
				y++
			}
			if c.p.set(x, y, TDirt) {
				road = append(road, [2]int{x, y})
			}
		}
		if len(road) < 10 {
			return false
		}
	}
	at := func(i int) (int, int) { q := road[clampi(i, 0, len(road)-1)]; return q[0], q[1] }
	n := len(road)
	cx, cy := at(n / 2)
	cu := bd.Culture
	x, y := at(n/2 - 3)
	c.drop(c.thing(ewCart[c.era], "Pushed this far, then left. There is nothing in it.", '&', CBrown, 30), x, y+1)
	x, y = at(n/2 - 2)
	c.drop(c.thing(Pick(r, ewToy[c.era]), "Dropped beside the cart, or set down there on purpose.", '*', CBrown, 1), x, y+1)
	x, y = at(n/2 + 4)
	c.drop(c.thing("a cooking pot with nothing in it", "Scraped clean, and then scraped again.", '&', CGray, 3), x, y-1)
	x, y = at(n / 4)
	c.drop(c.remains("a body by the roadside, very thin", "Sat down to rest and did not get up."), x, y+1)
	x, y = at(n * 3 / 4)
	c.drop(c.remains("a body curled under a coat", "The coat was put over them by someone else."), x, y-1)
	x, y = at(n/2 + 1)
	c.stone(x, y-2, fmt.Sprintf("A child. No name. The hunger of Y%d.", e.Y))
	refuge := w.ewNear(c.sc.P, 12, true, bd.ID)
	refN := "the coast"
	if refuge != nil {
		refN = refuge.Name
	}
	names := []string{c.person(cu), c.person(cu), c.person(cu)}
	gran := "The granaries were empty by midwinter."
	if choice != "" {
		gran = fmt.Sprintf("What was in the granaries was %s.", choice)
	}
	list := map[string]string{"old": "list of names on a strip of bark", "powder": "list of names on a ration card", "atom": "list of names on a ration card", "tech": "list of names on a ration chit", "ruin": "list of names on a cigarette packet"}[c.era]
	lines := []string{
		fmt.Sprintf("We left %s in the hunger of Y%d, walking to %s, where they say there is bread.", bd.Name, e.Y, refN),
		gran,
		fmt.Sprintf("%s, %s, %s. %s stopped here.", names[0], names[1], names[2], names[2]),
		c.say(cu, "proverb", map[string]string{"town": bd.Name}),
	}
	c.drop(c.note(list+" from "+bd.Name, lines), cx, cy+1)

	rc := c.rec
	rc.CX, rc.CY = cx, cy
	rc.T = "Famine road from " + bd.Name
	rc.L = "An empty cart, a child's toy beside it, and further on the people who were pushing it. This was the road out of the hunger."
	rc.H = "Something lies along the road ahead, very still."
	rc.F = fmt.Sprintf("In the famine of Y%d around %s, people walked to %s for bread.", e.Y, bd.Name, refN)
	if choice != "" {
		rc.F += fmt.Sprintf(" The grain was %s.", choice)
	}
	rc.S = rc.F
	rc.Src = "a list of names on the famine road"
	rc.B = bd.ID
	if refuge != nil {
		rc.Lead = refuge.P
		rc.LT = fmt.Sprintf("The walkers were going to %s.", refN)
	}
	return true
}

// ---- play: hints, discovery, payoff ----

func ewEnter(g *Game, zn *Zone) {
	st := ewSt(g)
	for _, k := range ewKeys(st) {
		rc := st.S[k]
		if rc.Z == zn.Key && !rc.D && !rc.Hn && rc.H != "" {
			rc.Hn = true
			g.Msg("{{K|" + rc.H + "}}")
		}
	}
}

func ewKeys(st *esState) []string {
	ks := make([]string, 0, len(st.S))
	for k := range st.S {
		ks = append(ks, k)
	}
	sort.Strings(ks)
	return ks
}

func ewTurn(g *Game) {
	if g.Cur == nil || g.W == nil || g.P == nil {
		return
	}
	st := ewSt(g)
	if len(st.S) == 0 {
		return
	}
	for _, k := range ewKeys(st) {
		rc := st.S[k]
		if rc.Pd {
			continue
		}
		if !rc.D && rc.Z == g.Cur.Key && maxi(abs(g.P.X-rc.CX), abs(g.P.Y-rc.CY)) <= 6 {
			rc.D = true
			g.Msg("{{y|" + rc.L + "}}")
			if rc.S != "" {
				g.addSecret(rc.S, "es", rc.P, rc.Y)
			}
			g.ewPlace(rc)
		}
		if rc.D && (rc.N == "" || g.Read["book:"+rc.N]) {
			g.ewPay(rc)
		}
	}
}

// ewPlace adds a Places entry for the scene's parasang when you have none there (places.go).
func (g *Game) ewPlace(rc *esRec) {
	st := g.wpl()
	pl := g.PlanetIndex()
	for _, p := range st.List {
		if p.Planet == pl && p.WX == g.WX && p.WY == g.WY {
			return
		}
	}
	if len(st.List) >= wplMax-8 {
		return
	}
	g.wplAdd(rc.T)
	if n := len(st.List); n > 0 {
		st.List[n-1].Note = fmt.Sprintf("Y%d. %s", rc.Y, knShort(rc.L, 90))
		st.touch()
	}
}

func (g *Game) ewPay(rc *esRec) {
	rc.Pd = true
	if rc.F != "" {
		subj := "burg:" + strconv.Itoa(rc.B)
		f := knSt(g).add(knFact{Kind: "event", Subject: subj, Text: rc.F, True: true, Source: "es:" + rc.ID, Day: lvDay(g), Burg: rc.B, Lvl: 1, Ver: 1})
		g.knLearn(f, &lvPerson{Key: "es:" + rc.ID, Name: nonEmpty(rc.Src, rc.T)}, 2, "found")
		g.Msg("{{c|What you read here goes into your case notes.}}")
	}
	if rc.Lead >= 0 && rc.Lead < len(g.W.Biome) {
		g.Known[rc.Lead] = true
		g.addSecret(rc.LT, "lead", rc.Lead, rc.Y)
		g.Msg("{{c|" + rc.LT + " (marked on your map)}}")
	}
}
