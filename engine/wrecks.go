package engine

import (
	"fmt"
	"sort"
	"strings"
)

// Sunken treasure (prefix wr): the world's real history, lost at sea. A registry derived purely from
// WorldData (wars between port realms, hurricanes and tsunamis, plague ships, embargo runners, spice
// fleets, lost expeditions, sea monsters, drowned towns and fallen lighthouses) puts named wrecks,
// drowned cities, kraken hoards and foundered lighthouses on open-sea parasangs (wrBuild, cached per
// world, never g.R). Sites are rumours until found: harbourmasters and sailors tell of them (talk
// hooks), the shipwright sells charts and a sounding line, lookouts spot shallow ones, and diving
// onto one finds it. The sea-floor zone (z 1, centre zone) is laid out by a zone hook with a side rng.
// Diving needs air: gear items (rig, bell, deep suit; Kind "wrgear") refill the breath meter from their
// own air; deeper sites crush an unprepared diver; salvage lots (Kind "wrcargo") drag you down without
// a bell. Strongboxes (Kind "wrbox", Extra["locked"] = tier) open with the captain's key or tinkering.
// State: g.ext("wr") (planet-local, "wr" in unLocalKeys).

type wrSiteData struct {
	ID      int
	Kind    string // wreck city hoard light
	P       int
	Name    string // with article: "the Gilded Heron", "the drowned city of Orlen"
	Ship    string
	Captain string
	CaptID  int
	Realm   int
	Year    int
	Source  string // war storm tsunami plague kraken expedition embargo spice city light
	Cause   string
	Event   string
	Cargo   string // key of wrCargoDefs
	Art     string // relic name for relic cargo
	Depth   int    // 1..4
	Tier    int    // strongbox lock tier 1..3
	Key     int    // key id of the captain's strongbox
	Hazards []string
	Monster int
	Mate    int // sister wreck id, -1 none
	Burg    int
}

func (s *wrSiteData) has(h string) bool {
	for _, x := range s.Hazards {
		if x == h {
			return true
		}
	}
	return false
}

type wrRegistry struct {
	Sites []*wrSiteData
	ByP   map[int]*wrSiteData
}

var wrCache = map[*WorldData]*wrRegistry{}

func wrReg(w *WorldData) *wrRegistry {
	if r, ok := wrCache[w]; ok {
		return r
	}
	r := wrBuild(w)
	wrCache[w] = r
	return r
}

// wrSite is the site on the sea floor under parasang (wx, wy), or nil.
func (g *Game) wrSite(wx, wy int) *wrSiteData {
	if g.W == nil || !g.W.In(wx, wy) {
		return nil
	}
	return wrReg(g.W).ByP[g.W.Idx(wx, wy)]
}

// ---- state ----

type wrState struct {
	Found  map[int]bool   `json:"f,omitempty"`
	Looted map[int]bool   `json:"l,omitempty"`
	Claim  map[int]bool   `json:"c,omitempty"`
	Paid   map[int]bool   `json:"pd,omitempty"`
	Dived  map[int]int    `json:"d,omitempty"`
	First  map[string]int `json:"fi,omitempty"`
	LastAt int            `json:"la,omitempty"`
}

func wrSt(g *Game) *wrState {
	s := g.ext("wr", func() any { return &wrState{} }).(*wrState)
	if s.Found == nil {
		s.Found = map[int]bool{}
	}
	if s.Looted == nil {
		s.Looted = map[int]bool{}
	}
	if s.Claim == nil {
		s.Claim = map[int]bool{}
	}
	if s.Paid == nil {
		s.Paid = map[int]bool{}
	}
	if s.Dived == nil {
		s.Dived = map[int]int{}
	}
	if s.First == nil {
		s.First = map[string]int{}
	}
	return s
}

func init() {
	unLocalKeys = append(unLocalKeys, "wr")
	zoneHooks = append(zoneHooks, wrZone)
	enterHooks = append(enterHooks, wrEnter)
	turnHooks = append(turnHooks, wrTick)
	killHooks = append(killHooks, wrKill)
	talkHooks = append(talkHooks, wrTalk)
	sideHooks = append(sideHooks, wrSide)
}

func (g *Game) wrFirst(key, text string) {
	st := wrSt(g)
	if st.First[key] == 0 {
		st.First[key] = g.Turn + 1
		g.deed("ship", text, 0, 0)
	}
}

// ---- the registry ----

type wrBuilder struct {
	w     *WorldData
	rg    *wrRegistry
	ports map[int][]*BurgData
	crew  map[int][]*PersonData
}

func wrBuild(w *WorldData) *wrRegistry {
	rg := &wrRegistry{ByP: map[int]*wrSiteData{}}
	if len(w.Biome) == 0 || len(w.seaOpen) != len(w.Biome) {
		return rg
	}
	b := &wrBuilder{w: w, rg: rg, ports: map[int][]*BurgData{}, crew: map[int][]*PersonData{}}
	for i := range w.Burgs {
		if bd := &w.Burgs[i]; bd.Alive && bd.Port && bd.Layer == "surface" {
			b.ports[bd.State] = append(b.ports[bd.State], bd)
		}
	}
	for i := range w.People {
		if p := &w.People[i]; p.Role == "general" || p.Role == "hero" {
			b.crew[p.State] = append(b.crew[p.State], p)
		}
	}
	b.wars()
	b.storms()
	b.plague()
	b.events()
	b.krakens()
	b.cities()
	b.lights()
	return rg
}

func (b *wrBuilder) hash(k string) uint64 { return HashStr(b.w.Seed + ":wr:" + k) }

func (b *wrBuilder) sea(p int) bool {
	w := b.w
	if p < 0 || p >= len(w.Biome) || !w.IsWater(p) || !w.seaOpen[p] {
		return false
	}
	_, has := w.seaBurg[p]
	return !has
}

func (b *wrBuilder) free(p int) bool { return b.sea(p) && b.rg.ByP[p] == nil }

// near picks (by hash) a free open-sea parasang at Chebyshev distance minR..maxR from p0, nearest ring first.
func (b *wrBuilder) near(p0 int, key string, minR, maxR int) int {
	w := b.w
	if p0 < 0 || p0 >= len(w.Biome) {
		return -1
	}
	x0, y0 := p0%w.W, p0/w.W
	for r := minR; r <= maxR; r++ {
		var c []int
		for dy := -r; dy <= r; dy++ {
			for dx := -r; dx <= r; dx++ {
				if maxi(abs(dx), abs(dy)) != r || !w.In(x0+dx, y0+dy) {
					continue
				}
				if q := w.Idx(x0+dx, y0+dy); b.free(q) {
					c = append(c, q)
				}
			}
		}
		if len(c) > 0 {
			return c[int(b.hash(key)%uint64(len(c)))]
		}
	}
	return -1
}

// landDist is the distance to the nearest land (1..max), or max+1.
func (b *wrBuilder) landDist(p, max int) int {
	w := b.w
	x0, y0 := p%w.W, p/w.W
	for r := 1; r <= max; r++ {
		for dy := -r; dy <= r; dy++ {
			for dx := -r; dx <= r; dx++ {
				if maxi(abs(dx), abs(dy)) == r && w.In(x0+dx, y0+dy) && !w.IsWater(w.Idx(x0+dx, y0+dy)) {
					return r
				}
			}
		}
	}
	return max + 1
}

func (b *wrBuilder) add(s *wrSiteData) *wrSiteData {
	if s.P < 0 || b.rg.ByP[s.P] != nil || len(b.rg.Sites) >= 80 {
		return nil
	}
	s.ID = len(b.rg.Sites)
	s.Mate = -1
	h := b.hash(fmt.Sprint("site", s.P))
	s.Depth = clampi(b.landDist(s.P, 4)+int(h%3)-1, 1, 4)
	switch s.Kind {
	case "hoard":
		s.Depth = clampi(s.Depth+1, 3, 4)
	case "city", "light":
		s.Depth = clampi(s.Depth, 1, 2)
	}
	s.Tier = clampi((s.Depth+1)/2+int(h/13%2), 1, 3)
	s.Key = int(h%900000) + 1
	if h%2 == 0 {
		s.Hazards = append(s.Hazards, "eel")
	}
	if (h/2)%3 == 0 || s.Depth >= 3 {
		s.Hazards = append(s.Hazards, "shark")
	}
	if s.Depth >= 2 && (h/7)%3 == 0 {
		s.Hazards = append(s.Hazards, "current")
	}
	if s.Kind != "hoard" && (h/5)%10 < 7 {
		s.Hazards = append(s.Hazards, "ghost")
	}
	if s.Kind == "hoard" || s.Source == "kraken" || s.Depth == 4 && (h/11)%4 == 0 {
		s.Hazards = append(s.Hazards, "kraken")
	}
	b.rg.Sites = append(b.rg.Sites, s)
	b.rg.ByP[s.P] = s
	return s
}

func (b *wrBuilder) shipName(key string) string {
	h := b.hash("ship:" + key)
	if l := wrShipWhole[b.w.Genre]; len(l) > 0 && h%2 == 0 {
		return l[int(h/2)%len(l)]
	}
	return wrShipA[int(h/3)%len(wrShipA)] + " " + wrShipB[int(h/101)%len(wrShipB)]
}

func (b *wrBuilder) captain(state, year int, key string) (string, int) {
	var pri, any []*PersonData
	for _, p := range b.crew[state] {
		if p.Born > year || !p.Alive && p.Died > 0 && p.Died < year {
			continue
		}
		any = append(any, p)
		if !p.Alive && p.Died >= year && p.Died <= year+3 {
			pri = append(pri, p)
		}
	}
	l := pri
	if len(l) == 0 {
		l = any
	}
	if len(l) == 0 {
		return "", 0
	}
	p := l[int(b.hash("capt:"+key)%uint64(len(l)))]
	return p.Name, p.ID
}

func (b *wrBuilder) realmAt(p int) int {
	if p >= 0 && p < len(b.w.State) {
		return b.w.State[p]
	}
	return 0
}

// wreck lays a lost ship at sea parasang p.
func (b *wrBuilder) wreck(src, key string, p, realm, year int, event, cause string) *wrSiteData {
	if p < 0 {
		return nil
	}
	order := wrCargoOrder[src]
	if len(order) == 0 {
		order = wrCargoOrder["storm"]
	}
	ship := b.shipName(key)
	s := &wrSiteData{Kind: "wreck", P: p, Ship: ship, Name: "the " + ship, Realm: realm, Year: year, Source: src, Event: event, Cause: cause,
		Cargo: order[int(b.hash("cargo:"+key)%uint64(len(order)))]}
	if realm > 0 {
		s.Captain, s.CaptID = b.captain(realm, year, key)
	}
	if arts := b.w.Artifacts; len(arts) > 0 {
		s.Art = arts[int(b.hash("art:"+key)%uint64(len(arts)))].Name
	}
	return b.add(s)
}

func (b *wrBuilder) sister(s *wrSiteData, realm int, key string) {
	if s == nil {
		return
	}
	if t := b.wreck(s.Source, key+"s", b.near(s.P, key+"sp", 1, 3), realm, s.Year, s.Event, s.Cause); t != nil {
		s.Mate, t.Mate = t.ID, s.ID
	}
}

func (b *wrBuilder) portOf(state int, key string) *BurgData {
	l := b.ports[state]
	if len(l) == 0 {
		return nil
	}
	return l[int(b.hash("port:"+key)%uint64(len(l)))]
}

func (b *wrBuilder) wars() {
	w := b.w
	for i := range w.Events {
		e := &w.Events[i]
		if e.Kind != "war" || e.St <= 0 {
			continue
		}
		h := b.hash(fmt.Sprint("war", i))
		if h%9 != 0 {
			continue
		}
		other := 0
		for j := range w.States {
			st := &w.States[j]
			if st.ID != e.St && (st.FullName != "" && strings.Contains(e.Text, st.FullName) || st.Name != "" && strings.Contains(e.Text, st.Name)) {
				other = st.ID
				break
			}
		}
		key := fmt.Sprint("war", i)
		pa, pb := b.portOf(e.St, key), b.portOf(other, key)
		p0 := -1
		switch {
		case e.P >= 0 && e.P < len(w.Biome) && w.IsWater(e.P):
			p0 = e.P
		case pa != nil && pb != nil:
			p0 = w.Idx((pa.P%w.W+pb.P%w.W)/2, (pa.P/w.W+pb.P/w.W)/2)
		case pa != nil:
			p0 = pa.P
		case pb != nil:
			p0 = pb.P
		}
		if p0 < 0 {
			continue
		}
		realm, foe := e.St, other
		if other > 0 && (h/9)%2 == 1 {
			realm, foe = other, e.St
		}
		s := b.wreck("war", key, b.near(p0, key+"p", 0, 9), realm, e.Y, e.Text, fmt.Sprintf("was sunk in the sea-fighting of the year %d", e.Y))
		if s != nil && h/17%2 == 0 && foe > 0 {
			b.sister(s, foe, key)
		}
	}
}

func (b *wrBuilder) storms() {
	w := b.w
	if w.Hazards == nil {
		return
	}
	for i := range w.Hazards.Recent {
		r := &w.Hazards.Recent[i]
		if r.Kind != "tsunami" && (r.Kind != "storm" || r.Severity < 3) {
			continue
		}
		key := fmt.Sprint("hz", i)
		p := -1
		var c []int
		for _, q := range r.Area {
			if b.free(q) {
				c = append(c, q)
			}
		}
		if len(c) > 0 {
			p = c[int(b.hash(key)%uint64(len(c)))]
		} else {
			p = b.near(r.P, key, 0, 6)
		}
		realm := b.realmAt(r.P)
		for _, id := range r.Towns {
			if bd := w.burgByID[id]; bd != nil {
				realm = bd.State
				break
			}
		}
		what := "the great " + r.Kind
		if r.Name != "" {
			what = "the " + r.Kind + " " + r.Name
		}
		s := b.wreck(r.Kind, key, p, realm, r.Year, fmt.Sprintf("In year %d, %s struck the coast.", r.Year, what), "foundered in "+what)
		if s != nil && b.hash(key+"sis")%5 < 2 {
			b.sister(s, realm, key)
		}
	}
	for i := range w.Events {
		e := &w.Events[i]
		if e.Kind != "disaster" {
			continue
		}
		t := strings.ToLower(e.Text)
		if !strings.Contains(t, "hurricane") && !strings.Contains(t, "typhoon") && !strings.Contains(t, "tsunami") {
			continue
		}
		key := fmt.Sprint("dis", i)
		if b.hash(key)%2 != 0 {
			continue
		}
		realm := e.St
		if realm <= 0 {
			realm = b.realmAt(e.P)
		}
		b.wreck("storm", key, b.near(e.P, key+"p", 1, 8), realm, e.Y, e.Text, "foundered in the great storm of the year "+fmt.Sprint(e.Y))
	}
}

func (b *wrBuilder) plague() {
	w := b.w
	if w.Hazards == nil {
		return
	}
	for i := range w.Hazards.Outbreaks {
		o := &w.Hazards.Outbreaks[i]
		in := map[int]bool{}
		for _, p := range o.Parasangs {
			in[p] = true
		}
		var ports []*BurgData
		for j := range w.Burgs {
			if bd := &w.Burgs[j]; bd.Alive && bd.Port && bd.Layer == "surface" && in[bd.P] {
				ports = append(ports, bd)
			}
		}
		if len(ports) == 0 {
			continue
		}
		key := fmt.Sprint("plague", i)
		bd := ports[int(b.hash(key)%uint64(len(ports)))]
		b.wreck("plague", key, b.near(bd.P, key+"p", 2, 8), bd.State, o.Since, fmt.Sprintf("In year %d the %s came by ship to %s.", o.Since, o.Name, bd.Name), "was turned from every harbour and cast adrift with plague aboard")
	}
}

func (b *wrBuilder) events() {
	w := b.w
	for i := range w.Events {
		e := &w.Events[i]
		src, mod := "", uint64(0)
		switch {
		case e.Kind == "misc" && strings.HasPrefix(e.Text, "SEA EMBARGO"):
			src, mod = "embargo", 6
		case e.Kind == "culture" && strings.Contains(e.Text, "Spice ships fill"):
			src, mod = "spice", 2
		case e.Kind == "culture" && strings.Contains(e.Text, "Navigators of") && strings.Contains(e.Text, "return with"):
			src, mod = "expedition", 2
		default:
			continue
		}
		key := fmt.Sprint("ev", i)
		if b.hash(key)%mod != 0 || e.P < 0 {
			continue
		}
		realm := e.St
		if realm <= 0 {
			realm = b.realmAt(e.P)
		}
		cause := map[string]string{"embargo": "ran the embargo and was never seen again", "spice": "went down overloaded with spice", "expedition": "sailed to find new lands and never returned"}[src]
		b.wreck(src, key, b.near(e.P, key+"p", 1, 8), realm, e.Y, e.Text, cause)
	}
}

func (b *wrBuilder) krakens() {
	w := b.w
	for i := range w.Monsters {
		m := &w.Monsters[i]
		if m.P < 0 || m.P >= len(w.Biome) {
			continue
		}
		nm := strings.ToLower(m.Name + " " + m.Species + " " + m.Lair)
		if !w.IsWater(m.P) && m.Layer != "sea" && !strings.Contains(nm, "kraken") && !strings.Contains(nm, "leviathan") {
			continue
		}
		key := fmt.Sprint("mon", m.ID)
		p := b.near(m.P, key+"p", 0, 8)
		if p < 0 {
			continue
		}
		s := b.add(&wrSiteData{Kind: "hoard", P: p, Name: m.Name + "'s hoard", Ship: m.Name, Year: w.Year, Source: "kraken", Event: m.Lore, Cargo: "treasury", Monster: m.ID,
			Cause: "sleeps on the plunder of a hundred ships"})
		if s == nil {
			continue
		}
		if t := b.wreck("kraken", key, b.near(p, key+"w", 1, 4), b.realmAt(m.P), w.Year-3-int(b.hash(key)%40), m.Lore, "was dragged under by "+m.Name); t != nil {
			t.Monster = m.ID
		}
	}
}

func (b *wrBuilder) hzFor(bd *BurgData) *HzRecent {
	w := b.w
	if w.Hazards == nil {
		return nil
	}
	for i := range w.Hazards.Recent {
		r := &w.Hazards.Recent[i]
		if r.Kind != "tsunami" && r.Kind != "flood" {
			continue
		}
		for _, id := range r.Towns {
			if id == bd.ID {
				return r
			}
		}
		for _, q := range r.Area {
			if q == bd.P {
				return r
			}
		}
	}
	return nil
}

func (b *wrBuilder) cities() {
	w := b.w
	for i := range w.Burgs {
		bd := &w.Burgs[i]
		if bd.Alive || bd.Layer != "surface" {
			continue
		}
		hz := b.hzFor(bd)
		if hz == nil && !bd.Port {
			continue
		}
		key := fmt.Sprint("dead", bd.ID)
		p := b.near(bd.P, key, 0, 3)
		if p < 0 {
			continue
		}
		year, ev, cause := w.Year-20, "The sea took "+bd.Name+".", "was taken by the sea"
		if hz != nil {
			year, cause = hz.Year, fmt.Sprintf("was drowned by the %s of the year %d", hz.Kind, hz.Year)
			ev = fmt.Sprintf("In the year %d a %s drowned %s and its people.", hz.Year, hz.Kind, bd.Name)
		}
		if evs := w.eventsAt[bd.P]; len(evs) > 0 && hz == nil {
			e := w.Events[evs[len(evs)-1]]
			year, ev = e.Y, e.Text
		}
		b.add(&wrSiteData{Kind: "city", P: p, Name: "the drowned city of " + bd.Name, Ship: bd.Name, Realm: bd.State, Year: year, Source: "city", Event: ev, Cause: cause, Cargo: "treasury", Burg: bd.ID})
	}
	for i := range w.Wonders {
		wd := &w.Wonders[i]
		if wd.Kind != "sunken_citadel" && !(wd.Kind == "giant_harbor" && wd.Destroyed) {
			continue
		}
		p := b.near(wd.P, fmt.Sprint("won", wd.ID), 0, 5)
		b.add(&wrSiteData{Kind: "city", P: p, Name: "the drowned " + strings.ToLower(wd.Name), Ship: wd.Name, Year: wd.Year, Source: "city", Event: wd.Lore, Cause: "lies drowned beneath the waves", Cargo: "treasury"})
	}
	if w.Hazards == nil {
		return
	}
	type cand struct {
		r *HzRecent
		i int
	}
	var cs []cand
	for i := range w.Hazards.Recent {
		if r := &w.Hazards.Recent[i]; r.Kind == "tsunami" && r.Severity >= 3 || r.Kind == "flood" && r.Severity >= 5 {
			cs = append(cs, cand{r, i})
		}
	}
	sort.SliceStable(cs, func(a, c int) bool { return cs[a].r.Severity > cs[c].r.Severity })
	n := 0
	for _, c := range cs {
		if n >= 3 {
			break
		}
		for _, id := range c.r.Towns {
			bd := w.burgByID[id]
			if bd == nil || !bd.Alive || !bd.Port {
				continue
			}
			key := fmt.Sprint("quarter", c.i, bd.ID)
			if p := b.near(bd.P, key, 1, 4); p >= 0 {
				if b.add(&wrSiteData{Kind: "city", P: p, Name: "the drowned quarter of " + bd.Name, Ship: bd.Name, Realm: bd.State, Year: c.r.Year, Source: "city", Burg: bd.ID,
					Event: fmt.Sprintf("In the year %d a %s took the old harbour quarter of %s.", c.r.Year, c.r.Kind, bd.Name), Cause: "was drowned by the " + c.r.Kind, Cargo: "treasury"}) != nil {
					n++
				}
				break
			}
		}
	}
}

func (b *wrBuilder) lights() {
	w := b.w
	for i := range w.Burgs {
		bd := &w.Burgs[i]
		if !bd.Alive || !bd.Port || bd.Capital || bd.Layer != "surface" {
			continue
		}
		key := fmt.Sprint("light", bd.ID)
		h := b.hash(key)
		if h%38 != 0 {
			continue
		}
		b.add(&wrSiteData{Kind: "light", P: b.near(bd.P, key, 1, 4), Name: "the drowned lighthouse of " + bd.Name, Ship: bd.Name, Realm: bd.State, Year: w.Year - 30 - int(h/38%120), Source: "light", Cargo: "treasury",
			Event: "The lamp-tower of " + bd.Name + " fell into the sea in a winter gale; the harbour dues in its lamp-room went with it.", Cause: "was thrown into the sea by a gale"})
	}
}

// ---- text ----

func wrDepthWord(d int) string {
	return [...]string{"", "shallows", "mid-water", "deep water", "abyssal dark"}[clampi(d, 1, 4)]
}

func (g *Game) wrWhere(p int) string {
	w := g.W
	best, bd := (*BurgData)(nil), 15
	for i := range w.Burgs {
		b := &w.Burgs[i]
		if b.Alive && b.Layer == "surface" {
			if d := maxi(abs(b.P%w.W-p%w.W), abs(b.P/w.W-p/w.W)); d < bd {
				best, bd = b, d
			}
		}
	}
	if best == nil {
		return "somewhere in the open sea off " + w.RegionName(p)
	}
	dx, dy := p%w.W-best.P%w.W, p/w.W-best.P/w.W
	return fmt.Sprintf("%d parasangs %s of %s", bd, wrDirNames[shDirNames[shHeading(dx, dy)]], best.Name)
}

func (g *Game) wrCargoName(s *wrSiteData) string {
	if s.Cargo == "relics" && s.Art != "" {
		return "the reliquary of " + s.Art
	}
	d := wrCargoDefs[s.Cargo]
	if n := d.Name[g.Genre]; n != "" {
		return n
	}
	return d.Name[""]
}

func (g *Game) wrRealm(s *wrSiteData) string {
	if s.Realm > 0 {
		return g.W.StateName(s.Realm)
	}
	return "the sea-lanes"
}

func (g *Game) wrHazardText(s *wrSiteData) string {
	var l []string
	for _, h := range s.Hazards {
		l = append(l, wrHazardText[h])
	}
	if len(l) == 0 {
		return ""
	}
	return " Divers speak of " + strings.Join(l, ", ") + "."
}

func (g *Game) wrTale(s *wrSiteData) string {
	where := g.wrWhere(s.P)
	switch s.Kind {
	case "city":
		return fmt.Sprintf("%s %s in the year %d. The streets are said to lie %s, in the %s.%s", trCap(s.Name), s.Cause, s.Year, where, wrDepthWord(s.Depth), g.wrHazardText(s))
	case "hoard":
		return fmt.Sprintf("%s %s. Its den lies %s, deep in the %s.%s", trCap(s.Name), s.Cause, where, wrDepthWord(s.Depth), g.wrHazardText(s))
	case "light":
		return fmt.Sprintf("%s %s in the year %d. The lamp-room gold lies %s.%s", trCap(s.Name), s.Cause, s.Year, where, g.wrHazardText(s))
	}
	capt := ""
	if s.Captain != "" {
		capt = ", under Captain " + s.Captain
	}
	return fmt.Sprintf("%s of %s%s %s in the year %d. She carried %s. They say she lies %s, in the %s.%s", trCap(s.Name), g.wrRealm(s), capt, s.Cause, s.Year, g.wrCargoName(s), where, wrDepthWord(s.Depth), g.wrHazardText(s))
}

func (g *Game) wrLog(s *wrSiteData) string {
	src := s.Source
	l := wrLogs[src]
	if len(l) == 0 {
		l = wrLogs["lost"]
	}
	t := l[int(HashStr(g.W.Seed+":wr:log:"+fmt.Sprint(s.ID))%uint64(len(l)))]
	capt := s.Captain
	if capt == "" {
		capt = "the master"
	}
	t = strings.NewReplacer("{captain}", capt, "{ship}", s.Ship, "{realm}", g.wrRealm(s), "{year}", fmt.Sprint(s.Year), "{place}", g.W.RegionName(s.P), "{cargo}", g.wrCargoName(s)).Replace(t)
	t = fmt.Sprintf("Logbook of the %s. %s\n\n\"%s\"", s.Ship, "Last entry, in a hand that grows unsteady:", t)
	if s.Event != "" {
		ev := s.Event
		if len(ev) > 230 {
			ev = ev[:230] + "..."
		}
		t += fmt.Sprintf("\n\nThe chronicle of %s records: %s", g.wrRealm(s), ev)
	}
	if s.Captain != "" {
		t += fmt.Sprintf("\n\nSigned: %s.", s.Captain)
	}
	return t
}

// wrNear lists sites by distance from parasang p (within 45), skipping found ones when unfound is set.
func (g *Game) wrNear(p int, unfound bool, n int) []*wrSiteData {
	w, st := g.W, wrSt(g)
	type ent struct {
		s *wrSiteData
		d int
	}
	var es []ent
	for _, s := range wrReg(w).Sites {
		if unfound && st.Found[s.P] {
			continue
		}
		d := maxi(abs(s.P%w.W-p%w.W), abs(s.P/w.W-p/w.W))
		if d > 45 {
			continue
		}
		es = append(es, ent{s, d*4 + int(HashStr(w.Seed+fmt.Sprint(":wr:j:", s.ID, ":", p))%12)})
	}
	sort.SliceStable(es, func(i, j int) bool { return es[i].d < es[j].d })
	var out []*wrSiteData
	for i := 0; i < len(es) && i < n; i++ {
		out = append(out, es[i].s)
	}
	return out
}

func (g *Game) wrFind(s *wrSiteData, how string) {
	st := wrSt(g)
	if st.Found[s.P] {
		return
	}
	st.Found[s.P] = true
	g.Msg(fmt.Sprintf("{{C|Charted: %s %s.}}", s.Name, how))
	g.wrFirst("found", fmt.Sprintf("%s learned where %s lies below the waves.", g.P.Name, s.Name))
}

// ---- sounding line, ship screen, world map ----

func (g *Game) wrHasLine() bool {
	for _, it := range g.P.Inv {
		if it.Kind == "wrgear" && it.Extra["sound"] > 0 {
			return true
		}
	}
	return false
}

// wrSound casts the sounding line at parasang (wx, wy): "" when you carry none.
func (g *Game) wrSound(wx, wy int) string {
	if g.W == nil || !g.wrHasLine() || !g.W.In(wx, wy) {
		return ""
	}
	p := g.W.Idx(wx, wy)
	if !g.W.IsWater(p) {
		return "The lead finds no water here."
	}
	s := wrReg(g.W).ByP[p]
	if s == nil {
		return "The lead goes down and comes up with grey silt and cold water. Nothing lies below."
	}
	g.wrFind(s, "by the sounding line")
	g.wrEnsureShipWreck(s)
	return fmt.Sprintf("{{Y|The lead strikes something at %s.}} It is %s: %s Dive from the ship screen.", wrDepthWord(s.Depth), s.Name, strings.TrimSpace(g.wrSounding(s)))
}

func (g *Game) wrSounding(s *wrSiteData) string {
	switch s.Kind {
	case "city":
		return "straight lines, courses of cut stone, a drowned city."
	case "hoard":
		return "a mound of metal and bone, and something breathing slowly."
	case "light":
		return "a long fallen tower, hollow."
	}
	return "timbers, ribs of a hull, a ship."
}

// wrShipOpts are ship-screen options (hook in the ship menu's sea block).
func (g *Game) wrShipOpts() []Option {
	sp := g.sh().Ship
	if sp == nil || !sp.Aboard || !g.W.IsWater(sp.At) || !g.wrHasLine() {
		return nil
	}
	return []Option{{"Cast the sounding line.", func() {
		g.tvPass(10)
		g.shMenuMsg(g.wrSound(sp.At%g.W.W, sp.At/g.W.W))
	}}}
}

func (g *Game) wrWorldGlyph(p int, gl rune, f byte) (rune, byte) {
	if g.W == nil || !wrSt(g).Found[p] {
		return gl, f
	}
	s := wrReg(g.W).ByP[p]
	if s == nil {
		return gl, f
	}
	if wrSt(g).Looted[p] {
		return '×', CDarkGray
	}
	switch s.Kind {
	case "city":
		return '◊', CCyan
	case "hoard":
		return '×', CRed
	}
	return '×', CYellow
}

func (g *Game) wrInfo(p int) []string {
	if g.W == nil || !wrSt(g).Found[p] {
		return nil
	}
	s := wrReg(g.W).ByP[p]
	if s == nil {
		return nil
	}
	return []string{fmt.Sprintf("{{Y|%s}} {{K|lies below, %s (year %d)}}", trCap(s.Name), wrDepthWord(s.Depth), s.Year)}
}

// wrEnsureShipWreck makes a found site diveable from the ship screen (the shsea.go dive flow).
func (g *Game) wrEnsureShipWreck(s *wrSiteData) {
	st := g.sh()
	for _, w := range st.Wrecks {
		if w.P == s.P {
			return
		}
	}
	st.Wrecks = append(st.Wrecks, shWreck{P: s.P, Name: s.Name, Placed: true})
}

// ---- gear ----

func (g *Game) wrGearItem(d *wrGearDef) *Item {
	nm := d.Name[g.Genre]
	if nm == "" {
		nm = d.Name[""]
	}
	it := &Item{ID: g.newID(), Kind: "wrgear", Name: nm, Text: nm, Glyph: 'Θ', F: CCyan, Value: d.Price / 2, Weight: 6, Lore: d.Desc,
		Extra: map[string]int{"tier": d.Tier, "air": d.Air, "max": d.Air}}
	if d.Tier == 0 {
		it.Extra["sound"] = 1
	}
	wrGearTag(it)
	return it
}

func wrGearTag(it *Item) {
	if it.Extra["max"] > 0 {
		it.Name = fmt.Sprintf("%s [air %d/%d]", it.Text, it.Extra["air"], it.Extra["max"])
	} else {
		it.Name = it.Text
	}
}

func (g *Game) wrBestGear() *Item {
	var best *Item
	for _, it := range g.P.Inv {
		if it.Kind == "wrgear" && it.Extra["tier"] > 0 && it.Extra["air"] > 0 && (best == nil || it.Extra["tier"] > best.Extra["tier"]) {
			best = it
		}
	}
	return best
}

func (g *Game) wrAllow() int {
	a := 1
	if g.P.Pow("diving") > 0 {
		a = 2
	}
	if it := g.wrBestGear(); it != nil {
		a = maxi(a, it.Extra["tier"]+1)
	}
	return a
}

func (g *Game) wrHasHoist() bool {
	for _, it := range g.P.Inv {
		if it.Kind == "wrgear" && it.Extra["tier"] >= 2 {
			return true
		}
	}
	return false
}

// wrUse is the useItem hook for wreck items (call from the default case of useItem).
func (g *Game) wrUse(it *Item) bool {
	switch it.Kind {
	case "wrbox":
		g.wrTryOpen(it, false)
	case "wrgear":
		if it.Extra["sound"] > 0 {
			g.Msg("Cast it from the ship screen, over the side.")
		} else {
			g.Msg(fmt.Sprintf("It feeds you air by itself while you are under the sea. (%d of %d minutes left; refill at a shipwright.)", it.Extra["air"], it.Extra["max"]))
		}
	case "wrcargo":
		g.Msg("A salvage lot: sell it in a port. A bell or deep suit lets you haul it from the bottom.")
	default:
		return false
	}
	g.Mode = "main"
	g.spend(1000)
	return true
}

// ---- strongboxes ----

func (g *Game) wrBox(s *wrSiteData, tier int) *Item {
	id, key := 0, 0
	if s != nil {
		id, key = s.ID+1, s.Key
	}
	return &Item{ID: g.newID(), Kind: "wrbox", Name: "captain's strongbox", Glyph: '▄', F: CYellow, Weight: 12, Value: 60,
		Lore: "Iron-banded and locked. The captain's key opens it, or a patient hand with tools.", Extra: map[string]int{"locked": tier, "wrb": id, "bkey": key}}
}

func (g *Game) wrKeyItem(s *wrSiteData) *Item {
	return &Item{ID: g.newID(), Kind: "key", Name: "captain's key", Glyph: '¬', F: CYellow, Value: 5, Extra: map[string]int{"key": s.Key}}
}

func (g *Game) wrTryOpen(it *Item, auto bool) bool {
	tier := it.Extra["locked"]
	var key *Item
	for _, k := range g.P.Inv {
		if k.Kind == "key" && k.Extra != nil && it.Extra["bkey"] != 0 && k.Extra["key"] == it.Extra["bkey"] {
			key = k
			break
		}
	}
	tk := g.Skills["tinkering"]
	switch {
	case key != nil:
		g.removeInv(key)
		g.Msg("{{Y|The captain's key turns.}}")
	case auto && tk > tier:
		g.Msg("{{Y|You work the strongbox's lock open with your tools.}}")
	case !auto && tk >= 1 && g.R.Int(1, 20)+g.P.M(AGI)+tk*2 >= 10+tier*3:
		g.Msg("{{Y|You pick the strongbox's lock.}}")
	default:
		if !auto {
			g.Msg(fmt.Sprintf("{{R|The strongbox is locked (tier %d).}} It needs the captain's key, or Tinkering and luck.", tier))
		}
		return false
	}
	g.wrBoxLoot(it)
	g.removeInv(it)
	return true
}

func (g *Game) wrBoxLoot(it *Item) {
	reg := wrReg(g.W)
	var s *wrSiteData
	if id := it.Extra["wrb"] - 1; id >= 0 && id < len(reg.Sites) {
		s = reg.Sites[id]
	}
	depth, tier, name := 4, 3, "the Abyssal Trench"
	if s != nil {
		depth, tier, name = s.Depth, s.Tier, s.Name
		wrSt(g).Looted[s.P] = true
	}
	r := NewRng(fmt.Sprintf("%s:wr:box:%d:%d", g.W.Seed, it.Extra["wrb"], it.Extra["bkey"]))
	gold := (140 + 90*depth) * (2 + tier) / 2
	earned := g.cuEarn(gold)
	n := 1 + tier
	for i := 0; i < n; i++ {
		g.give(g.Loot(r, tier+1+depth/2))
	}
	g.Msg(fmt.Sprintf("{{G|The strongbox of %s holds %s and %d finds.}}", name, earned, n))
	g.gainXP(30 + depth*20)
	if s != nil {
		// the captain's chart: another place the same history drowned
		var o *wrSiteData
		if s.Mate >= 0 && s.Mate < len(reg.Sites) {
			o = reg.Sites[s.Mate]
		}
		if o == nil || wrSt(g).Found[o.P] {
			if l := g.wrNear(s.P, true, 1); len(l) > 0 {
				o = l[0]
			}
		}
		if o != nil && !wrSt(g).Found[o.P] {
			g.wrFind(o, "from the captain's chart")
		}
	}
	g.wrFirst("box", fmt.Sprintf("%s opened a drowned captain's strongbox.", g.P.Name))
}

func wrAutoOpen(g *Game) {
	for _, it := range append([]*Item(nil), g.P.Inv...) {
		if it.Kind == "wrbox" {
			g.wrTryOpen(it, true)
		}
	}
}

// ---- zone generation (side rng only) ----

func wrZone(g *Game, zn *Zone, _ *Rng) {
	if g.W == nil || zn.Z != 1 || !zn.Underwater || zn.Kind != "seafloor" {
		return
	}
	if g.bsLostAt(zn.WX, zn.WY, zn.ZX, zn.ZY, zn.Z) == "trench" {
		g.wrTrench(zn)
		return
	}
	if zn.ZX != 1 || zn.ZY != 1 {
		return
	}
	s := g.wrSite(zn.WX, zn.WY)
	if s == nil {
		return
	}
	r := NewRng(g.W.Seed + ":wr:zone:" + zn.Key)
	var stairs [][3]int
	for y := 0; y < MH; y++ {
		for x := 0; x < MW; x++ {
			if t := zn.At(x, y); t == TUp || t == TDown {
				stairs = append(stairs, [3]int{x, y, int(t)})
			}
		}
	}
	zn.Actors, zn.Items = nil, nil
	switch s.Kind {
	case "city":
		g.wrLayCity(zn, s, r)
	case "light":
		g.wrLayLight(zn, s, r)
	case "hoard":
		g.wrLayHoard(zn, s, r)
	default:
		g.wrLayWreck(zn, s, r)
	}
	for _, st := range stairs {
		g.clearAround(zn, st[0], st[1], 1)
		zn.Set(st[0], st[1], Terrain(st[2]))
	}
	zn.Name = trCap(s.Name)
	g.connect(zn)
	g.computeLight(zn)
	g.wrPeople(zn, s, r)
}

func (g *Game) wrAnchor(zn *Zone) {
	g.clearAround(zn, 31, 12, 2)
	zn.Set(31, 12, TSand)
	zn.Set(30, 12, TUp)
	zn.Notes[12*MW+30] = "The anchor line, taut and green with weed. Climb it (<) to return to the surface."
}

func (zn *Zone) wrDry(x, y int) {
	if zn.Dry == nil {
		zn.Dry = make([]bool, MW*MH)
	}
	if zn.In(x, y) {
		zn.Dry[y*MW+x] = true
	}
}

func (g *Game) wrClearOval(zn *Zone, r *Rng, cx, cy int, rx, ry float64) {
	for y := 0; y < MH; y++ {
		for x := 0; x < MW; x++ {
			dx, dy := float64(x-cx)/rx, float64(y-cy)/ry
			if dx*dx+dy*dy <= 1 {
				zn.Set(x, y, TSand)
			}
		}
	}
	for i := 0; i < int(rx*ry/6); i++ {
		x, y := cx+r.Int(-int(rx), int(rx)), cy+r.Int(-int(ry), int(ry))
		if zn.At(x, y) == TSand {
			zn.Set(x, y, Pick(r, []Terrain{TRubble, TBones, TKelp, TCoral}))
		}
	}
}

func (g *Game) wrPut(zn *Zone, it *Item, x, y int) {
	it.X, it.Y = x, y
	zn.Items = append(zn.Items, it)
}

func (g *Game) wrGold(amount int) *Item {
	return &Item{ID: g.newID(), Kind: "gold", Name: g.Sk.Currency, Glyph: '$', F: CYellow, Count: amount}
}

func (g *Game) wrNote(zn *Zone, x, y int, text string) {
	zn.Set(x, y, TInscription)
	zn.Notes[y*MW+x] = text
}

func (g *Game) wrLots(zn *Zone, s *wrSiteData, r *Rng, n int, spots [][2]int) {
	d := wrCargoDefs[s.Cargo]
	for i := 0; i < n && i < len(spots); i++ {
		j := r.Int(i, len(spots)-1)
		spots[i], spots[j] = spots[j], spots[i]
		it := &Item{ID: g.newID(), Kind: "wrcargo", Name: g.wrCargoName(s) + " (salvage lot)", Glyph: '■', F: CBrown, Weight: 14,
			Value: d.Value*(1+s.Depth/2) + r.Int(0, 20), Lore: fmt.Sprintf("Cargo lost with %s in the year %d. Heavy: without a bell or deep suit it drags you down.", s.Name, s.Year)}
		g.wrPut(zn, it, spots[i][0], spots[i][1])
	}
}

func (g *Game) wrLayWreck(zn *Zone, s *wrSiteData, r *Rng) {
	g.wrClearOval(zn, r, 42, 12, 18, 9)
	g.wrAnchor(zn)
	wall := func() Terrain {
		if r.Chance(0.75) {
			return TWall
		}
		return TRubble
	}
	for x := 33; x <= 53; x++ {
		hw := 3
		if x > 47 {
			hw = 3 - (x-47)/2
		}
		if x < 35 {
			hw = 3 - (35 - x)
		}
		for y := 12 - hw; y <= 12+hw; y++ {
			if y == 12-hw || y == 12+hw || x == 33 || x == 53 {
				zn.Set(x, y, wall())
			} else if r.Chance(0.12) {
				zn.Set(x, y, TRubble)
			} else {
				zn.Set(x, y, TBoat)
			}
		}
	}
	zn.Set(33, 12, TRubble)
	zn.Set(34, 12, TBoat)
	// cabin at the bow, with an air pocket
	for y := 9; y <= 15; y++ {
		for x := 45; x <= 51; x++ {
			if y == 9 || y == 15 || x == 45 || x == 51 {
				zn.Set(x, y, TWall)
			} else {
				zn.Set(x, y, TBoat)
				zn.wrDry(x, y)
			}
		}
	}
	zn.Set(45, 12, TDoorOpen)
	zn.Set(40, 12, TPillar)
	var spots [][2]int
	for x := 37; x <= 43; x++ {
		for _, y := range []int{10, 11, 13, 14} {
			if zn.At(x, y) == TBoat {
				spots = append(spots, [2]int{x, y})
			}
		}
	}
	for _, p := range spots[:minInt(len(spots), 3)] {
		if r.Chance(0.6) {
			zn.Set(p[0], p[1], TCrate)
		}
	}
	spots = spots[minInt(len(spots), 3):]
	g.wrLots(zn, s, r, 3+s.Depth, spots)
	g.wrNote(zn, 49, 11, g.wrLog(s))
	g.wrPut(zn, g.wrBox(s, s.Tier), 49, 13)
	g.wrNote(zn, 32, 10, fmt.Sprintf("A name board, barnacled and torn free: %s, of %s. Year %d.", trCap(s.Name), g.wrRealm(s), s.Year))
	if !s.has("ghost") {
		g.wrPut(zn, g.wrKeyItem(s), 47, 12)
		zn.Set(47, 12, TBones)
	}
	g.wrPut(zn, g.Loot(r, s.Tier+s.Depth/2), 38, 12)
	g.wrPut(zn, g.wrGold(30+s.Depth*40+r.Int(0, 50)), 44, 12)
}

// wrBuilding walls a rectangle (border x..x+w-1) with a floor; door at (dx, dy) or -1 for random on the bottom edge.
func (g *Game) wrBuilding(zn *Zone, r *Rng, x, y, w, h int, floor Terrain, dx, dy int, dry bool) bool {
	if x < 1 || y < 1 || x+w >= MW-1 || y+h >= MH-1 || x+w > 26 && x < 35 && y+h > 8 && y < 16 {
		return false
	}
	for yy := y - 1; yy <= y+h; yy++ {
		for xx := x - 1; xx <= x+w; xx++ {
			if zn.At(xx, yy) != TSand && zn.At(xx, yy) != TKelp {
				return false
			}
		}
	}
	for yy := y; yy < y+h; yy++ {
		for xx := x; xx < x+w; xx++ {
			if yy == y || yy == y+h-1 || xx == x || xx == x+w-1 {
				if r.Chance(0.72) {
					zn.Set(xx, yy, TWall)
				} else {
					zn.Set(xx, yy, TRubble)
				}
			} else {
				zn.Set(xx, yy, floor)
				if dry {
					zn.wrDry(xx, yy)
				}
			}
		}
	}
	if dx < 0 {
		dx, dy = x+r.Int(1, w-2), y+h-1
	}
	zn.Set(dx, dy, TDoorOpen)
	return true
}

func (g *Game) wrLayCity(zn *Zone, s *wrSiteData, r *Rng) {
	for i := range zn.T {
		zn.T[i] = TSand
	}
	for i := 0; i < 110; i++ {
		zn.Set(r.Int(1, MW-2), r.Int(1, MH-2), Pick(r, []Terrain{TKelp, TCoral, TBones, TRubble}))
	}
	g.wrAnchor(zn)
	for x := 6; x <= 73; x++ {
		for _, y := range []int{2, 22} {
			zn.Set(x, y, TSand)
			if r.Chance(0.75) {
				zn.Set(x, y, TWall)
			}
		}
	}
	for y := 2; y <= 22; y++ {
		for _, x := range []int{6, 73} {
			zn.Set(x, y, TSand)
			if r.Chance(0.75) {
				zn.Set(x, y, TWall)
			}
		}
	}
	for x := 0; x < MW; x++ {
		zn.Set(x, 12, TCobble)
	}
	for y := 3; y <= 21; y++ {
		zn.Set(40, y, TCobble)
	}
	for y := 9; y <= 15; y++ {
		for x := 34; x <= 46; x++ {
			zn.Set(x, y, TCobble)
		}
	}
	zn.Set(30, 12, TUp)
	zn.Set(31, 12, TCobble)
	tale := Pick(r, wrCityTales)
	g.wrNote(zn, 40, 10, fmt.Sprintf("The chronicle stone of %s.\n\n%s\n\n%s", s.Ship, s.Event, tale))
	for _, p := range [][2]int{{36, 10}, {44, 10}, {36, 14}, {44, 14}} {
		zn.Set(p[0], p[1], TStatue)
		zn.Notes[p[1]*MW+p[0]] = "A civic hero in green bronze, one arm still raised to the surface it will never see again."
	}
	if g.wrBuilding(zn, r, 48, 4, 13, 5, TCarpet, 54, 8, true) {
		zn.Set(54, 5, TAltar)
		zn.Notes[5*MW+54] = "An altar. Air has been trapped under the vault of the roof; you can breathe here."
		g.wrNote(zn, 51, 6, "Scratched into the lintel by a hand that did not wait: \"The water is at the second step. We pray in the dry while we can.\"")
	}
	keyID := int(HashStr(zn.Key) % 1000000)
	vault := g.wrBuilding(zn, r, 50, 16, 11, 5, TFloor, 55, 16, false)
	if vault {
		zn.Set(55, 16, TLocked)
		g.wrPut(zn, g.wrBox(s, s.Tier), 55, 18)
		for i := 0; i < 3; i++ {
			g.wrPut(zn, g.Loot(r, s.Tier+1+i/2), 52+i*3, 19)
		}
		g.wrPut(zn, g.wrGold(120+90*s.Depth+r.Int(0, 80)), 53, 17)
		g.wrPut(zn, g.wrGold(120+60*s.Depth), 58, 17)
	}
	for i := 0; i < 16; i++ {
		w, h := r.Int(5, 9), r.Int(4, 5)
		y := r.Int(3, 8)
		if i%2 == 1 {
			y = r.Int(14, 18)
		}
		x := r.Int(8, 70-w)
		if !g.wrBuilding(zn, r, x, y, w, h, TFloor, -1, 0, false) {
			continue
		}
		for k := 0; k < r.Int(1, 3); k++ {
			zn.Set(x+r.Int(1, w-2), y+r.Int(1, h-2), Pick(r, []Terrain{TCrate, TBed, TShelf, TBones}))
		}
		if r.Chance(0.5) {
			g.wrPut(zn, g.Loot(r, 1+s.Depth/2), x+r.Int(1, w-2), y+h-2)
		}
	}
	g.wrPut(zn, g.wrGold(40+r.Int(0, 60)), 38, 12)
	g.wrPut(zn, g.wrGold(40+r.Int(0, 60)), 42, 13)
	var spots [][2]int
	for i := 0; i < 8; i++ {
		spots = append(spots, [2]int{8 + r.Int(0, 60), 12})
	}
	g.wrLots(zn, s, r, 2, spots)
	zn.Level++
	_ = keyID
}

func (g *Game) wrLayLight(zn *Zone, s *wrSiteData, r *Rng) {
	g.wrClearOval(zn, r, 46, 12, 19, 8)
	g.wrAnchor(zn)
	for x := 33; x <= 52; x++ {
		for _, y := range []int{10, 14} {
			if r.Chance(0.8) {
				zn.Set(x, y, TWall)
			} else {
				zn.Set(x, y, TRubble)
			}
		}
		for y := 11; y <= 13; y++ {
			if r.Chance(0.15) {
				zn.Set(x, y, TRubble)
			} else {
				zn.Set(x, y, TFloor)
			}
		}
	}
	// the lamp-room, with its air
	for y := 8; y <= 16; y++ {
		for x := 52; x <= 60; x++ {
			if y == 8 || y == 16 || x == 52 || x == 60 {
				zn.Set(x, y, TWall)
			} else {
				zn.Set(x, y, TFloor)
				zn.wrDry(x, y)
			}
		}
	}
	zn.Set(52, 12, TDoorOpen)
	zn.Set(56, 12, TBrazier)
	g.wrNote(zn, 57, 9, g.wrLog(s))
	g.wrPut(zn, g.wrBox(s, s.Tier), 58, 13)
	g.wrPut(zn, g.wrKeyItem(s), 54, 14)
	for i := 0; i < 5; i++ {
		g.wrPut(zn, g.wrGold(60+s.Depth*40+r.Int(0, 40)), 34+i*4, 12)
	}
	g.wrPut(zn, g.Loot(r, s.Tier+1), 55, 10)
}

func (g *Game) wrLayHoard(zn *Zone, s *wrSiteData, r *Rng) {
	g.wrClearOval(zn, r, 48, 12, 22, 10)
	g.wrAnchor(zn)
	var spots [][2]int
	for dy := -3; dy <= 3; dy++ {
		for dx := -5; dx <= 5; dx++ {
			x, y := 54+dx, 12+dy
			zn.Set(x, y, TSand)
			switch {
			case r.Chance(0.12):
				zn.Set(x, y, TBones)
			case r.Chance(0.1):
				zn.Set(x, y, TRubble)
			default:
				spots = append(spots, [2]int{x, y})
				if r.Chance(0.35) {
					g.wrPut(zn, g.wrGold(40+r.Int(0, 80)+s.Depth*30), x, y)
				}
			}
		}
	}
	g.wrPut(zn, g.wrBox(s, 3), 53, 11)
	g.wrPut(zn, g.wrBox(s, s.Tier), 55, 13)
	g.wrLots(zn, s, r, 4, spots)
	g.wrPut(zn, g.Loot(r, s.Tier+2), 54, 12)
	g.wrNote(zn, 45, 9, fmt.Sprintf("Scratches in the stone, deep as a man's arm: the tally of ships taken by %s. There are far too many marks to count.", s.Ship))
}

// wrTrench adds a sleeping kraken's hoard to an Abyssal Trench zone (beside the bestiary's own population).
func (g *Game) wrTrench(zn *Zone) {
	r := NewRng(g.W.Seed + ":wr:trench:" + zn.Key)
	lvl := zn.Level + 5
	k := g.Spawn("kraken", lvl, r)
	k.Name, k.Boss, k.AI = "trench kraken", true, "still"
	k.Set("wrsleep")
	k.Set("wrkraken")
	s := &wrSiteData{ID: -1, Key: int(HashStr(zn.Key)%900000) + 1}
	key := g.wrKeyItem(s)
	k.Inv = append(k.Inv, key)
	cx, cy := 20+r.Int(0, 40), 6+r.Int(0, 12)
	g.wrClearOval(zn, r, cx, cy, 7, 4)
	k.X, k.Y = cx, cy
	zn.Actors = append(zn.Actors, k)
	for i := 0; i < 6; i++ {
		if x, y, ok := zn.FreeSpot(r, func(x, y int) bool { return abs(x-cx) <= 6 && abs(y-cy) <= 3 }); ok {
			g.wrPut(zn, g.wrGold(80+r.Int(0, 120)), x, y)
		}
	}
	if x, y, ok := zn.FreeSpot(r, func(x, y int) bool { return abs(x-cx) <= 5 && abs(y-cy) <= 3 }); ok {
		b := g.wrBox(nil, 3)
		b.Extra["bkey"] = s.Key
		g.wrPut(zn, b, x, y)
	}
	if x, y, ok := zn.FreeSpot(r, func(x, y int) bool { return abs(x-cx) <= 5 && abs(y-cy) <= 3 }); ok {
		g.wrPut(zn, g.Loot(r, 5), x, y)
	}
	g.computeLight(zn)
}

func (g *Game) wrPlace(zn *Zone, r *Rng, a *Actor, near func(x, y int) bool) {
	if x, y, ok := zn.FreeSpot(r, near); ok {
		a.X, a.Y, a.HomeX, a.HomeY = x, y, x, y
		zn.Actors = append(zn.Actors, a)
	}
}

func (g *Game) wrPeople(zn *Zone, s *wrSiteData, r *Rng) {
	lvl := zn.Level + s.Depth
	any := func(x, y int) bool { return true }
	core := func(x, y int) bool { return abs(x-42) <= 12 && abs(y-12) <= 4 }
	if s.Kind == "city" {
		core = func(x, y int) bool { return x > 8 && x < 72 }
	}
	if s.has("ghost") {
		tp, nm := "skeleton", "drowned sailor"
		switch {
		case s.Source == "plague":
			tp, nm = "ghoul", "plague-struck sailor"
		case s.Kind == "city":
			nm = "drowned citizen"
		case s.Kind == "light":
			nm = "drowned keeper"
		}
		n := 2 + r.Int(0, 2)
		if s.Kind == "city" {
			n = 6
		}
		if s.Kind == "light" {
			n = 1
		}
		for i := 0; i < n; i++ {
			a := g.Spawn(tp, lvl, r)
			a.Set("undead")
			g.wrPlace(zn, r, a, core)
			a.Name = nm
		}
		if s.Kind == "wreck" || s.Kind == "city" {
			c := g.Spawn("wight", lvl+1, r)
			c.Boss, c.Proper = true, true
			c.Set("wrcapt")
			if s.Kind == "city" {
				c.Name, c.Title = "the drowned magistrate", "keeper of the vault key"
				c.Inv = append(c.Inv, &Item{ID: g.newID(), Kind: "key", Name: "vault key", Glyph: '¬', F: CYellow, Value: 5, Extra: map[string]int{"key": int(HashStr(zn.Key) % 1000000)}})
			} else {
				c.Name = "Captain " + s.Captain
				if s.Captain == "" {
					c.Name = "the drowned captain"
				}
				c.Title = "master of " + s.Name
				c.Inv = append(c.Inv, g.wrKeyItem(s))
			}
			c.Desc = "The sea has not been kind to the face, but the hat is still on straight."
			if s.Kind == "wreck" {
				c.X, c.Y, c.HomeX, c.HomeY = 48, 12, 48, 12
				zn.Actors = append(zn.Actors, c)
			} else {
				g.wrPlace(zn, r, c, func(x, y int) bool { return x > 48 && y < 10 })
			}
		}
	} else if s.Kind == "city" {
		// no ghosts: the vault key lies in the temple dry
		g.wrPut(zn, &Item{ID: g.newID(), Kind: "key", Name: "vault key", Glyph: '¬', F: CYellow, Value: 5, Extra: map[string]int{"key": int(HashStr(zn.Key) % 1000000)}}, 54, 6)
	}
	if s.has("eel") {
		for i := 0; i < 2; i++ {
			g.wrPlace(zn, r, g.Spawn("eel", lvl, r), any)
		}
	}
	if s.has("shark") {
		g.wrPlace(zn, r, g.Spawn("shark", lvl, r), any)
	}
	if s.has("kraken") {
		k := g.Spawn("kraken", lvl+3, r)
		k.Boss = true
		k.AI = "still"
		k.Set("wrsleep")
		k.Set("wrkraken")
		if s.Monster > 0 {
			for i := range g.W.Monsters {
				if m := &g.W.Monsters[i]; m.ID == s.Monster {
					k.Name, k.Proper, k.Desc = m.Name, true, m.Lore
				}
			}
		} else {
			k.Name = "sleeping kraken"
		}
		if s.Kind == "hoard" {
			k.X, k.Y, k.HomeX, k.HomeY = 60, 12, 60, 12
			zn.Actors = append(zn.Actors, k)
		} else {
			g.wrPlace(zn, r, k, func(x, y int) bool { return x > 56 || y < 5 || y > 19 })
		}
	}
}

// ---- hooks ----

func wrEnter(g *Game, zn *Zone) {
	if g.W == nil || zn.Z != 1 || !zn.Underwater {
		return
	}
	s := g.wrSite(zn.WX, zn.WY)
	if s == nil {
		return
	}
	st := wrSt(g)
	g.wrFind(s, "by diving onto it")
	g.wrEnsureShipWreck(s)
	if zn.ZX != 1 || zn.ZY != 1 {
		return
	}
	if st.Dived[s.P] == 0 {
		st.Dived[s.P] = g.Turn + 1
		g.wrFirst("dive", fmt.Sprintf("%s dived to %s.", g.P.Name, s.Name))
		if s.Kind == "city" {
			g.wrFirst("city", fmt.Sprintf("%s walked the streets of %s.", g.P.Name, s.Name))
		}
		if s.Mate >= 0 {
			if o := wrReg(g.W).Sites[s.Mate]; !st.Found[o.P] {
				g.wrFind(o, "from the wreckage here")
			}
		}
	}
	if s.Depth >= 4 {
		g.wrFirst("abyss", fmt.Sprintf("%s reached the abyssal dark of %s.", g.P.Name, s.Name))
	}
	if over := s.Depth - g.wrAllow(); over > 0 {
		g.Msg(fmt.Sprintf("{{R|%s: the pressure at this depth will crush you without better gear (bell, deep suit). Go back up!}}", wrDepthWord(s.Depth)))
	}
}

func wrTick(g *Game) {
	if g.W == nil || g.P == nil || g.Cur == nil {
		return
	}
	st := wrSt(g)
	wrShipWatch(g, st)
	z := g.Cur
	if !z.Underwater || z.Z != 1 {
		if len(g.P.Inv) > 0 {
			wrAutoOpen(g)
		}
		return
	}
	p := g.P
	inWater := z.WaterAt(p.X, p.Y) >= 2 && p.Pow("gills") == 0 && !p.Has("swimmer")
	s := g.wrSite(z.WX, z.WY)
	depth := 1
	if s != nil {
		depth = s.Depth
	}
	if inWater {
		if gear := g.wrBestGear(); gear != nil {
			g.Breath = mini(g.Breath+1, g.breathCap())
			gear.Extra["air"]--
			wrGearTag(gear)
			if gear.Extra["air"] == 0 {
				g.Msg("{{R|Your " + gear.Text + " runs out of air!}}")
			} else if gear.Extra["air"] == 10 {
				g.Msg("{{Y|Ten minutes of air left.}}")
			}
		} else if depth > 1 {
			g.Breath = maxi(-5, g.Breath-(depth-1))
		}
		if !g.wrHasHoist() {
			n := 0
			for _, it := range p.Inv {
				if it.Kind == "wrcargo" {
					n++
				}
			}
			if n > 0 {
				g.Breath = maxi(-5, g.Breath-n)
				if g.Turn%10 == 0 {
					g.Msg("{{Y|The salvage drags at you; without a bell you cannot haul it far.}}")
				}
			}
		}
		if over := depth - g.wrAllow(); over > 0 && s != nil {
			g.hurt(p, over*2, nil, "pressure")
			if g.Turn%4 == 0 {
				g.Msg("{{R|The pressure crushes your chest.}}")
			}
		}
		if s != nil && s.has("current") && g.Turn%5 == 0 {
			dx := 1
			if s.ID%2 == 0 {
				dx = -1
			}
			nx := p.X + dx
			if z.Walkable(nx, p.Y) && z.ActorAt(nx, p.Y) == nil && z.WaterAt(nx, p.Y) >= 2 {
				p.X = nx
				g.Msg("{{B|The current drags you sideways.}}")
				g.recompute()
			}
		}
	}
	for _, a := range z.Actors {
		if a.HP > 0 && a.Has("wrsleep") && dist(a, p) <= 6 {
			delete(a.Flags, "wrsleep")
			a.AI, a.Angry = "hostile", true
			g.Msg("{{R|" + trCap(a.Name) + " stirs in its hoard and opens an eye the size of a shield.}}")
		}
	}
}

// wrShipWatch: lookouts and sounding lines find sites as the ship arrives; found ones become diveable.
func wrShipWatch(g *Game, st *wrState) {
	sp := g.sh().Ship
	if sp == nil || !sp.Aboard || !g.W.IsWater(sp.At) {
		return
	}
	s := wrReg(g.W).ByP[sp.At]
	if st.LastAt != sp.At+1 {
		st.LastAt = sp.At + 1
		if s != nil && !st.Found[s.P] {
			if g.wrHasLine() {
				g.Msg(g.wrSound(sp.At%g.W.W, sp.At/g.W.W))
			} else if HashStr(fmt.Sprintf("%s:wr:look:%d:%d", g.W.Seed, s.P, g.Turn/DayLen))%100 < uint64(44-s.Depth*9) {
				g.wrFind(s, "by your lookout")
				g.Msg("{{C|A lookout sights wreckage below the clear water.}} (Z: dive)")
			}
		}
	}
	if s != nil && st.Found[s.P] {
		g.wrEnsureShipWreck(s)
		inside := g.Cur != nil && g.Cur.Underwater && g.Cur.WX == s.P%g.W.W && g.Cur.WY == s.P/g.W.W
		if !inside {
			sh := g.sh()
			for i := range sh.Wrecks {
				if sh.Wrecks[i].P == s.P && sh.Wrecks[i].Salvaged {
					sh.Wrecks[i].Salvaged = false
				}
			}
		}
	}
}

func wrKill(g *Game, d, src *Actor) {
	if g.W == nil || src != g.P {
		return
	}
	if d.Has("wrcapt") {
		g.wrFirst("capt", fmt.Sprintf("%s laid %s to rest beneath the sea.", g.P.Name, d.Name))
	}
	if d.Has("wrkraken") {
		g.wrFirst("kraken", fmt.Sprintf("%s slew %s in its own hoard.", g.P.Name, d.Name))
	}
}

func wrSide(g *Game) []string {
	if g.W == nil || g.Cur == nil || g.P == nil || !g.Cur.Underwater || g.Cur.Z != 1 {
		return nil
	}
	var out []string
	if s := g.wrSite(g.Cur.WX, g.Cur.WY); s != nil {
		col := "C"
		if s.Depth > g.wrAllow() {
			col = "R"
		}
		out = append(out, fmt.Sprintf("{{%s|Depth %d: %s}}", col, s.Depth, wrDepthWord(s.Depth)))
	}
	if gear := g.wrBestGear(); gear != nil {
		out = append(out, fmt.Sprintf("{{B|Air: %d min (%s)}}", gear.Extra["air"], gear.Text))
	}
	return out
}

// ---- talk ----

func wrTalk(g *Game, a *Actor) []Option {
	if g.W == nil || g.Cur == nil {
		return nil
	}
	var out []Option
	switch {
	case a.Role == "ferry":
		out = append(out,
			Option{"Ask the harbour rolls about lost ships.", func() { g.wrRumour(a, 0) }},
			Option{fmt.Sprintf("Buy a chart of drowned places hereabouts. — %s", g.cuFormat(120)), func() { g.wrBuyChart(a) }},
			Option{"Salvage claims.", func() { g.wrClaims(a) }})
	case a.Role == "villager" && (a.Title == "sailor" || a.Title == "dockhand" || a.Title == "fisher"):
		out = append(out, Option{fmt.Sprintf("Buy a round and ask after lost ships. — %s", g.cuFormat(8)), func() { g.wrRumour(a, 8) }})
	case a.Role == "shipwright":
		out = append(out, Option{"{{B|Diving gear, bells and sounding lines.}}", func() { g.wrGearMenu(a) }})
	}
	return out
}

func (g *Game) wrRumour(a *Actor, cost int) {
	p := g.W.Idx(g.Cur.WX, g.Cur.WY)
	l := g.wrNear(p, true, 1)
	if len(l) == 0 {
		g.talkMenu("\"Nothing more I can tell you about wrecks in these waters.\"")
		return
	}
	if cost > 0 && !g.cuSpend(cost) {
		g.talkMenu("\"Empty cup, empty mouth.\"")
		return
	}
	s := l[0]
	st := wrSt(g)
	st.Found[s.P] = true
	op := fmt.Sprintf(Pick(NewRng(g.W.Seed+":wr:op:"+fmt.Sprint(s.ID)), wrRumourOpen), s.Name)
	g.wrFirst("found", fmt.Sprintf("%s learned where %s lies below the waves.", g.P.Name, s.Name))
	g.talkMenu(op + " " + g.wrTale(s) + "\"\n\n{{C|It is marked on your world map.}}")
}

func (g *Game) wrBuyChart(a *Actor) {
	p := g.W.Idx(g.Cur.WX, g.Cur.WY)
	l := g.wrNear(p, true, 3)
	if len(l) == 0 {
		g.talkMenu("\"I've sold you every drowned place my charts hold.\"")
		return
	}
	if !g.cuSpend(120) {
		g.talkMenu("\"Charts are inked on good vellum, friend.\"")
		return
	}
	st := wrSt(g)
	text := "Notes in a harbour clerk's hand.\n"
	for _, s := range l {
		st.Found[s.P] = true
		text += fmt.Sprintf("\n%s\n", g.wrTale(s))
	}
	g.give(&Item{ID: g.newID(), Kind: "book", Name: "chart of drowned places", Glyph: '?', F: CCyan, Value: 40, Text: text})
	g.wrFirst("found", fmt.Sprintf("%s bought a chart of drowned places.", g.P.Name))
	g.talkMenu(fmt.Sprintf("\"Three marks, inked in. Mind the depths.\" {{C|%d places are marked on your world map.}}", len(l)))
}

func (g *Game) wrClaims(a *Actor) {
	st := wrSt(g)
	g.Talk = a
	g.Popup = Wrap("\"A claim filed here keeps the harbour office on your side, and finders are paid for what they report. Salvage taken without a claim is just theft with wet hands.\"", 100)
	g.Opts = nil
	back := func() { g.talkMenu(g.greeting(a)) }
	n := 0
	for _, s := range wrReg(g.W).Sites {
		if !st.Found[s.P] || n >= 8 {
			continue
		}
		s := s
		fee := 30 + 25*s.Depth
		switch {
		case !st.Claim[s.P]:
			n++
			g.Opts = append(g.Opts, Option{fmt.Sprintf("File a claim on %s. — %s", s.Name, g.cuFormat(fee)), func() {
				if !g.cuSpend(fee) {
					g.talkMenu("\"The clerk wants his fee first.\"")
					return
				}
				st.Claim[s.P] = true
				g.talkMenu("\"Stamped and filed. Nobody else may lawfully touch her.\"")
			}})
		case st.Looted[s.P] && !st.Paid[s.P]:
			n++
			reward := 80 + 70*s.Depth*s.Tier
			g.Opts = append(g.Opts, Option{fmt.Sprintf("Report your salvage from %s. — they pay %s", s.Name, g.cuFormat(reward)), func() {
				st.Paid[s.P] = true
				g.cuEarn(reward)
				if s.Realm > 0 {
					g.AddRep(fmt.Sprintf("state:%d", s.Realm), 40, " for returning a lost ship's papers")
				}
				g.gainXP(40 + s.Depth*15)
				g.wrFirst("claim", fmt.Sprintf("%s was paid a finder's share for the salvage of %s.", g.P.Name, s.Name))
				g.talkMenu("\"A finder's share, as the law says. The owners will be told who brought them home.\"")
			}})
		}
	}
	g.Opts = append(g.Opts, Option{"Never mind.", back})
	g.Sel = 0
}

func (g *Game) wrGearMenu(a *Actor) {
	g.Talk = a
	g.Popup = Wrap("\"Everything a man needs to go down and, with luck, come back up. Gear feeds on its own air; I refill it for a fee.\"", 100)
	g.Opts = nil
	back := func() { g.talkMenu(g.greeting(a)) }
	for i := range wrGearDefs {
		d := &wrGearDefs[i]
		price := d.Price * (100 - g.Skills["persuasion"]*5) / 100
		it := g.wrGearItem(d)
		g.Opts = append(g.Opts, Option{fmt.Sprintf("{{Y|%s}} {{K|(%s)}} — %s", trCap(it.Text), d.Desc, g.cuFormat(price)), func() {
			if !g.cuSpend(price) {
				g.talkMenu("\"Not on credit. Not for something that keeps you alive.\"")
				return
			}
			g.give(g.wrGearItem(d))
			g.talkMenu("\"Check the seals twice.\"")
		}})
	}
	need := 0
	for _, it := range g.P.Inv {
		if it.Kind == "wrgear" && it.Extra["max"] > it.Extra["air"] {
			need += it.Extra["max"] - it.Extra["air"]
		}
	}
	if need > 0 {
		fee := 10 + need/3
		g.Opts = append(g.Opts, Option{fmt.Sprintf("Refill your air. — %s", g.cuFormat(fee)), func() {
			if !g.cuSpend(fee) {
				g.talkMenu("\"Air is cheap. Not free.\"")
				return
			}
			for _, it := range g.P.Inv {
				if it.Kind == "wrgear" && it.Extra["max"] > 0 {
					it.Extra["air"] = it.Extra["max"]
					wrGearTag(it)
				}
			}
			g.talkMenu("\"Full and sweet.\"")
		}})
	}
	g.Opts = append(g.Opts, Option{"Never mind.", back})
	g.Sel = 0
}
