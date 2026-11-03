package engine

import (
	"fmt"
	"sort"
	"strings"
)

// Politics (prefix po): the court of each realm (offices and political factions derived from the
// world file's people, ideology and social order), audiences with the ruler, office tasks for the
// player, backing factions, intrigue (bribes, blackmail, assassination contracts), and succession
// crises / coups / revolutions that break out in the living world (tied to socialworld.go movements).
// The player never rules. Screen on `A`; saved state in g.ext("po"), planet-local ("po" in unLocalKeys).

func init() {
	turnHooks = append(turnHooks, poTurn)
	sideHooks = append(sideHooks, poSide)
	talkHooks = append(talkHooks, poTalk)
	mainKeys["A"] = func(g *Game) { g.Mode, g.Sel, g.Tab = "politics", 0, 0; g.po().Ask = nil }
	modeKeys["politics"] = poKeys
	modeDraws["politics"] = func(g *Game) { g.poDraw() }
	unLocalKeys = append(unLocalKeys, "po")
}

// ---- saved state ----

type poTask struct {
	Office int    `json:"o"`
	Kind   string `json:"k"` // levy embassy parley probe muster mediate
	Realm  int    `json:"r"`
	Burg   int    `json:"b"`
	Foe    int    `json:"f,omitempty"`
	Stage  int    `json:"s"`
	Pay    int    `json:"p"`
	Cost   int    `json:"c,omitempty"`
	Left   int    `json:"l,omitempty"`
	Due    int    `json:"d"` // day
	Plot   string `json:"pl,omitempty"`
	PlotID int    `json:"pi,omitempty"`
}

type poCrisis struct {
	Kind   string `json:"k"` // succession coup revolution
	Start  int    `json:"s"` // day
	End    int    `json:"e"`
	A      string `json:"a"` // faction ids
	B      string `json:"b"`
	Bias   int    `json:"bi,omitempty"` // + favours A, - favours B (the player's push)
	Done   bool   `json:"dn,omitempty"`
	Winner string `json:"w,omitempty"`
}

type poContract struct {
	Target int    `json:"t"`
	Name   string `json:"n"`
	Realm  int    `json:"r"`
	Due    int    `json:"d"` // day
	Cost   int    `json:"c"`
	Cover  bool   `json:"cv,omitempty"`
}

type poAsk struct {
	Text string
	Opts []Option
}

type poState struct {
	Last      int               `json:"last,omitempty"`
	LastDay   int               `json:"ld,omitempty"`
	Office    map[int]int       `json:"off,omitempty"`  // realm → office index held
	Done      map[int]int       `json:"done,omitempty"` // realm → tasks finished
	Stand     map[int]int       `json:"st,omitempty"`   // realm → court standing
	Fav       map[string]int    `json:"fav,omitempty"`  // faction id → favour -100..100
	Hit       map[string]int    `json:"hit,omitempty"`  // faction id → strength change from events
	Back      map[int]string    `json:"back,omitempty"` // realm → backed faction id
	Dirt      map[int]int       `json:"dirt,omitempty"` // person id → secrets gathered
	Dead      map[int]int       `json:"dead,omitempty"` // person id → day+1 they died
	Ruler     map[int]string    `json:"ruler,omitempty"`
	Crises    map[int]*poCrisis `json:"cr,omitempty"`
	Peace     map[string]int    `json:"peace,omitempty"` // "a-b" → day the truce ends
	Aud       map[int]int       `json:"aud,omitempty"`   // realm → day+1 of the last audience
	Paid      map[int]int       `json:"paid,omitempty"`  // realm → day of the last stipend
	Contracts []*poContract     `json:"con,omitempty"`
	Task      *poTask           `json:"task,omitempty"`
	Log       []string          `json:"log,omitempty"`
	Ask       *poAsk            `json:"-"`
}

func (g *Game) po() *poState {
	s := g.ext("po", func() any { return &poState{} }).(*poState)
	if s.Office == nil {
		s.Office, s.Done, s.Stand = map[int]int{}, map[int]int{}, map[int]int{}
		s.Fav, s.Hit, s.Back = map[string]int{}, map[string]int{}, map[int]string{}
		s.Dirt, s.Dead, s.Ruler = map[int]int{}, map[int]int{}, map[int]string{}
		s.Crises, s.Peace, s.Aud, s.Paid = map[int]*poCrisis{}, map[string]int{}, map[int]int{}, map[int]int{}
	}
	if s.Office == nil || s.Dirt == nil || s.Crises == nil || s.Peace == nil || s.Hit == nil || s.Back == nil || s.Ruler == nil || s.Dead == nil || s.Aud == nil || s.Paid == nil || s.Fav == nil || s.Done == nil || s.Stand == nil {
		if s.Office == nil {
			s.Office = map[int]int{}
		}
		if s.Done == nil {
			s.Done = map[int]int{}
		}
		if s.Stand == nil {
			s.Stand = map[int]int{}
		}
		if s.Fav == nil {
			s.Fav = map[string]int{}
		}
		if s.Hit == nil {
			s.Hit = map[string]int{}
		}
		if s.Back == nil {
			s.Back = map[int]string{}
		}
		if s.Dirt == nil {
			s.Dirt = map[int]int{}
		}
		if s.Dead == nil {
			s.Dead = map[int]int{}
		}
		if s.Ruler == nil {
			s.Ruler = map[int]string{}
		}
		if s.Crises == nil {
			s.Crises = map[int]*poCrisis{}
		}
		if s.Peace == nil {
			s.Peace = map[string]int{}
		}
		if s.Aud == nil {
			s.Aud = map[int]int{}
		}
		if s.Paid == nil {
			s.Paid = map[int]int{}
		}
	}
	return s
}

func (g *Game) poDay() int { return g.Turn / DayLen }

func (g *Game) poYear() int { return g.W.Year + g.poDay()/YearDays }

// poRealm: the realm whose court the screen shows (where you stand, else the last one seen).
func (g *Game) poRealm() int {
	po := g.po()
	if g.Cur != nil && g.W.In(g.Cur.WX, g.Cur.WY) {
		if s := g.W.State[g.W.Idx(g.Cur.WX, g.Cur.WY)]; s > 0 {
			if st := g.W.stateByID[s]; st != nil && st.Alive {
				po.Last = s
				return s
			}
		}
	}
	if po.Last == 0 {
		po.Last = g.HomeState
	}
	return po.Last
}

func (g *Game) poLog(text string) {
	po := g.po()
	po.Log = append(po.Log, fmt.Sprintf("Y%d: %s", g.poYear(), text))
	if len(po.Log) > 24 {
		po.Log = po.Log[len(po.Log)-24:]
	}
}

func poPeaceKey(a, b int) string {
	if a > b {
		a, b = b, a
	}
	return fmt.Sprintf("%d-%d", a, b)
}

// poPeace: a truce the player negotiated is in force (optional hook for warsOf / atWar).
func (g *Game) poPeace(a, b int) bool {
	po := g.po()
	d, ok := po.Peace[poPeaceKey(a, b)]
	return ok && d > g.poDay()
}

// poCrisisAt: the unresolved crisis in a realm, if any.
func (g *Game) poCrisisAt(realm int) *poCrisis {
	if c := g.po().Crises[realm]; c != nil && !c.Done {
		return c
	}
	return nil
}

// poPriceMod: optional price factor: markets tighten in a realm in upheaval.
func (g *Game) poPriceMod(a *Actor) float64 {
	if a != nil && a.State > 0 && g.poCrisisAt(a.State) != nil {
		return 1.12
	}
	return 1
}

// poLawless: the watch is stretched thin while a coup or revolution runs (optional hook for crime.go).
func (g *Game) poLawless(state int) bool {
	c := g.poCrisisAt(state)
	return c != nil && c.Kind != "succession"
}

// ---- the court, derived from the world ----

type poMember struct {
	ID     int
	Name   string
	Title  string
	Office int
	Mar    int
	Dip    int
	Ste    int
	Int    int
	Gen    bool
}

var poPoolCache = map[*WorldData]map[int][]*PersonData{}

func poPool(w *WorldData, realm int) []*PersonData {
	m := poPoolCache[w]
	if m == nil {
		m = map[int][]*PersonData{}
		for i := range w.People {
			p := &w.People[i]
			if p.Alive && p.State > 0 {
				m[p.State] = append(m[p.State], p)
			}
		}
		poPoolCache[w] = m
	}
	st := w.stateByID[realm]
	var out []*PersonData
	for _, p := range m[realm] {
		if st != nil && p.ID == st.RulerID {
			continue
		}
		out = append(out, p)
	}
	return out
}

func (g *Game) poGenName(realm int, key string) string {
	r := NewRng(fmt.Sprintf("%s:po:name:%d:%s", g.Seed, realm, key))
	given, house := "Aren", "Vael"
	if st := g.W.stateByID[realm]; st != nil {
		if cu := g.W.cultByID[st.Culture]; cu != nil {
			if len(cu.Given) > 0 {
				given = cu.Given[r.Int(0, len(cu.Given)-1)]
			}
			if len(cu.Houses) > 0 {
				house = cu.Houses[r.Int(0, len(cu.Houses)-1)]
			}
		}
	}
	return given + " " + house
}

func poOfficeScore(oi int, p *PersonData) int {
	switch oi {
	case 0:
		return p.Diplo*2 + p.Steward/2
	case 1:
		return p.Steward*2 + p.Diplo/2
	case 2:
		s := p.Martial * 2
		if p.Role == "general" {
			s += 6
		}
		return s
	case 3:
		return p.Intrigue*2 + p.Diplo/2
	}
	s := p.Diplo + p.Intrigue/2
	if p.Role == "priest" || p.Role == "prophet" || p.Role == "magician" {
		s += 10
	}
	return s
}

func (g *Game) poCourt(realm int) [5]poMember {
	po := g.po()
	sk := poSkinFor(g.Genre)
	var out [5]poMember
	used := map[int]bool{}
	for _, oi := range []int{4, 2, 0, 1, 3} {
		var best *PersonData
		bs := -1 << 30
		for _, p := range poPool(g.W, realm) {
			if used[p.ID] || po.Dead[p.ID] > 0 || p.Role == "heir" {
				continue
			}
			if sc := poOfficeScore(oi, p); sc > bs || (sc == bs && best != nil && p.ID < best.ID) {
				best, bs = p, sc
			}
		}
		if best != nil {
			used[best.ID] = true
			out[oi] = poMember{ID: best.ID, Name: best.Name, Title: sk.Offices[oi], Office: oi, Mar: best.Martial, Dip: best.Diplo, Ste: best.Steward, Int: best.Intrigue}
			continue
		}
		n := 0
		id := -(realm*1000 + oi*20)
		for po.Dead[id-n] > 0 && n < 19 {
			n++
		}
		r := NewRng(fmt.Sprintf("%s:po:gen:%d:%d:%d", g.Seed, realm, oi, n))
		out[oi] = poMember{ID: id - n, Name: g.poGenName(realm, fmt.Sprintf("o%d.%d", oi, n)), Title: sk.Offices[oi], Office: oi,
			Mar: r.Int(3, 9), Dip: r.Int(3, 9), Ste: r.Int(3, 9), Int: r.Int(3, 9), Gen: true}
	}
	return out
}

func (g *Game) poRulerName(realm int) string {
	if n := g.po().Ruler[realm]; n != "" {
		return n
	}
	if st := g.W.stateByID[realm]; st != nil && st.Ruler != "" {
		return st.Ruler
	}
	return g.poGenName(realm, "ruler")
}

func (g *Game) poCouncil(realm int) string {
	sk := poSkinFor(g.Genre)
	st := g.W.stateByID[realm]
	if st != nil && (g.Genre == "cyberpunk" || g.Genre == "wasteland" || g.Genre == "solarpunk" || g.Genre == "biopunk") {
		return sk.Council
	}
	if st != nil {
		if c, ok := poCouncilByIdeology[st.Ideology]; ok {
			return c
		}
	}
	return sk.Council
}

// ---- political factions ----

type poFac struct {
	ID     string
	Name   string
	Kind   string
	Leader string
	Person int
	Str    int
}

func (g *Game) poFacs(realm int) []poFac {
	st := g.W.stateByID[realm]
	if st == nil {
		return nil
	}
	po := g.po()
	sk := poSkinFor(g.Genre)
	court := g.poCourt(realm)
	rs := NewRng(fmt.Sprintf("%s:po:fac:%d", g.Seed, realm))
	var noise [5]int
	for i := range noise {
		noise[i] = rs.Int(-8, 8)
	}
	war := len(g.warsOf(realm)) > 0
	var gens, merch, priests int
	var marSum int
	for _, p := range poPool(g.W, realm) {
		if po.Dead[p.ID] > 0 {
			continue
		}
		switch p.Role {
		case "general":
			gens++
			marSum += p.Martial
		case "merchant":
			merch++
		case "priest", "prophet", "magician":
			priests++
		}
	}
	var out []poFac
	add := func(id, name, kind, leader string, person, str int) {
		str += po.Hit[id]
		out = append(out, poFac{ID: id, Name: name, Kind: kind, Leader: leader, Person: person, Str: clampi(str, 3, 95)})
	}
	fid := func(k string) string { return fmt.Sprintf("po:%d:%s", realm, k) }
	cs := 35
	if rp := g.W.swPerson(st.RulerID); rp != nil && po.Ruler[realm] == "" {
		cs += (rp.Diplo + rp.Steward) / 2
	}
	add(fid("crown"), sk.Fac[0], "crown", g.poRulerName(realm), 0, cs+noise[0])
	ss := 18 + poMin(40, marSum+gens*3) + noise[1]
	if war {
		ss += 8
	}
	add(fid("sword"), sk.Fac[1], "sword", court[2].Name, court[2].ID, ss)
	ps := 22 + poMin(32, merch*8+court[1].Ste*2) + noise[2]
	if war {
		ps -= 4
	}
	add(fid("purse"), sk.Fac[2], "purse", court[1].Name, court[1].ID, ps)
	add(fid("altar"), sk.Fac[3], "altar", court[4].Name, court[4].ID, 22+poMin(30, priests*9)+noise[3])
	if mv := swMovementOf(st); mv != "" {
		leader := "a firebrand of the lower streets"
		press, sup := 30, 30
		if so := st.SocialOrder; so != nil {
			press = clampi(so.Pressure, 0, 100)
		}
		if m := swMove(st); m != nil {
			if m.Leader != "" {
				leader = m.Leader
			}
			sup = clampi(m.Support, 0, 100)
		}
		add(fid("street"), mv, "street", leader, 0, 15+press/3+sup/3+noise[4])
	}
	n := 0
	for _, p := range poPool(g.W, realm) {
		if p.Role != "heir" || po.Dead[p.ID] > 0 || n >= 2 {
			continue
		}
		n++
		hr := NewRng(fmt.Sprintf("%s:po:heir:%d", g.Seed, p.ID))
		add(fid(fmt.Sprintf("h%d", p.ID)), "Partisans of "+p.Name, "heir", p.Name, p.ID, 18+clampi(p.Renown/10, 0, 20)+hr.Int(-8, 8))
	}
	return out
}

func poMin(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func poFacByID(fs []poFac, id string) *poFac {
	for i := range fs {
		if fs[i].ID == id {
			return &fs[i]
		}
	}
	return nil
}

func poFacKind(fs []poFac, kind string) *poFac {
	for i := range fs {
		if fs[i].Kind == kind {
			return &fs[i]
		}
	}
	return nil
}

func poFavWord(v int) string { return RepWord(v * 7) }

// ---- crises: succession, coups, revolutions ----

func (g *Game) poSides(realm int, kind string) (a, b *poFac) {
	fs := g.poFacs(realm)
	crown, sword := poFacKind(fs, "crown"), poFacKind(fs, "sword")
	switch kind {
	case "succession":
		var h []*poFac
		for i := range fs {
			if fs[i].Kind == "heir" {
				h = append(h, &fs[i])
			}
		}
		switch {
		case len(h) >= 2:
			return h[0], h[1]
		case len(h) == 1:
			return h[0], crown
		}
		return crown, sword
	case "coup":
		return crown, sword
	}
	return crown, poFacKind(fs, "street")
}

func (g *Game) poInvolved(realm int, cr *poCrisis) bool {
	po := g.po()
	if cr != nil && (cr.Bias != 0 || po.Back[realm] == cr.A || po.Back[realm] == cr.B) {
		return true
	}
	_, has := po.Office[realm]
	return has || g.poRealmQuiet() == realm
}

// poRealmQuiet: the realm the player is standing in (no side effects), 0 elsewhere.
func (g *Game) poRealmQuiet() int {
	if g.Cur != nil && g.W.In(g.Cur.WX, g.Cur.WY) {
		return g.W.State[g.W.Idx(g.Cur.WX, g.Cur.WY)]
	}
	return 0
}

func (g *Game) poAnnounce(realm int, cr *poCrisis, text string) {
	g.poLog(text)
	if g.poInvolved(realm, cr) {
		g.Msg("{{O|" + text + "}}")
		g.deed("event", text, 0, 0)
	}
}

var poCrisisWord = map[string]string{"succession": "a succession crisis", "coup": "a coup", "revolution": "a revolution"}

func (g *Game) poStart(realm int, kind string, day int) {
	st := g.W.stateByID[realm]
	a, b := g.poSides(realm, kind)
	if st == nil || a == nil || b == nil {
		return
	}
	dur := map[string]int{"succession": 20, "coup": 14, "revolution": 28}[kind]
	cr := &poCrisis{Kind: kind, Start: day, End: day + dur, A: a.ID, B: b.ID}
	g.po().Crises[realm] = cr
	var txt string
	switch kind {
	case "succession":
		txt = fmt.Sprintf("%s dies. %s and %s each claim the seat of %s.", g.poRulerName(realm), a.Leader, b.Leader, st.Name)
	case "coup":
		txt = fmt.Sprintf("%s plots against %s: the officers of %s march.", b.Leader, g.poRulerName(realm), st.Name)
	default:
		txt = fmt.Sprintf("Revolution in %s: %s rise against %s.", st.Name, b.Name, g.poRulerName(realm))
	}
	g.poAnnounce(realm, cr, txt)
}

func (g *Game) poResolve(realm int, cr *poCrisis) {
	po := g.po()
	st := g.W.stateByID[realm]
	if st == nil {
		cr.Done = true
		return
	}
	fs := g.poFacs(realm)
	a, b := poFacByID(fs, cr.A), poFacByID(fs, cr.B)
	cr.Done = true
	if a == nil || b == nil {
		return
	}
	rs := NewRng(fmt.Sprintf("%s:po:res:%d:%d", g.Seed, realm, cr.Start))
	sa, sb := a.Str+rs.Int(-12, 12), b.Str+rs.Int(-12, 12)
	if cr.Bias > 0 {
		sa += cr.Bias
	} else {
		sb -= cr.Bias
	}
	if po.Back[realm] == a.ID {
		sa += 6
	} else if po.Back[realm] == b.ID {
		sb += 6
	}
	win, lose := a, b
	if sb > sa {
		win, lose = b, a
	}
	cr.Winner = win.ID
	po.Hit[win.ID] += 8
	po.Hit[lose.ID] -= 15
	var txt string
	switch cr.Kind {
	case "succession":
		name := win.Leader
		if win.Kind != "heir" {
			name = g.poGenName(realm, fmt.Sprintf("reg%d", cr.Start))
		}
		po.Ruler[realm] = name
		txt = fmt.Sprintf("%s takes the seat of %s; %s submits.", name, st.Name, lose.Leader)
	case "coup":
		if win.Kind == "sword" {
			po.Ruler[realm] = win.Leader
			txt = fmt.Sprintf("The coup succeeds: %s seizes %s.", win.Leader, st.Name)
		} else {
			if lose.Person != 0 {
				po.Dead[lose.Person] = cr.End + 1
			}
			txt = fmt.Sprintf("The coup fails: %s is purged and %s holds %s.", lose.Leader, g.poRulerName(realm), st.Name)
		}
	default:
		if win.Kind == "street" {
			po.Ruler[realm] = g.poGenName(realm, fmt.Sprintf("rev%d", cr.Start))
			txt = fmt.Sprintf("The revolution wins: %s raise %s to the seat of %s.", win.Name, po.Ruler[realm], st.Name)
		} else {
			txt = fmt.Sprintf("The rising in %s is broken; %s is hunted down.", st.Name, lose.Name)
		}
	}
	// the player's stake
	side := ""
	switch {
	case cr.Bias > 0:
		side = a.ID
	case cr.Bias < 0:
		side = b.ID
	default:
		side = po.Back[realm]
	}
	if side != "" && (side == a.ID || side == b.ID) {
		if side == win.ID {
			reward := 80 + g.P.Level*15
			g.P.Gold += reward
			po.Fav[win.ID] = clampi(po.Fav[win.ID]+25, -100, 100)
			g.Msg(fmt.Sprintf("{{G|Your side prevailed in %s: %d %s as thanks.}}", st.Name, reward, g.Sk.Currency))
			txt += " Those who backed the winners are rewarded."
		} else {
			po.Fav[win.ID] = clampi(po.Fav[win.ID]-25, -100, 100)
			g.Rep[stateFaction(realm)] -= 60
			if _, ok := po.Office[realm]; ok {
				delete(po.Office, realm)
				if po.Task != nil && po.Task.Realm == realm {
					po.Task = nil
				}
				txt += " Your post is revoked."
			}
		}
	}
	g.poAnnounce(realm, cr, txt)
}

// poDaily advances the political world one day (deterministic from the seed, realm and day).
func (g *Game) poDaily(day int) {
	po := g.po()
	for i := range g.W.States {
		st := &g.W.States[i]
		if !st.Alive {
			continue
		}
		cr := po.Crises[st.ID]
		if cr != nil && !cr.Done {
			if day >= cr.End {
				g.poResolve(st.ID, cr)
			}
			continue
		}
		if cr != nil && day < cr.End+SeasonDays {
			continue
		}
		rs := NewRng(fmt.Sprintf("%s:po:day:%d:%d", g.Seed, st.ID, day))
		r := rs.Float()
		age := 40
		if rp := g.W.swPerson(st.RulerID); rp != nil && po.Ruler[st.ID] == "" {
			age = g.poYear() - rp.Born
		}
		pSucc := 1.0 / 3500
		if age > 55 {
			pSucc = 1.0 / 500
		}
		pCoup := 0.0
		switch st.Ideology {
		case "autocracy", "monarchy", "oligarchy", "technocracy", "theocracy":
			pCoup = 1.0 / 1800
		default:
			pCoup = 1.0 / 6000
		}
		if len(g.warsOf(st.ID)) > 0 {
			pCoup *= 2
		}
		pRev := 0.0
		if so := st.SocialOrder; so != nil && so.Movement {
			pRev = 1.0 / float64(300+(100-clampi(so.Pressure, 0, 100))*4)
		}
		switch {
		case r < pSucc:
			g.poStart(st.ID, "succession", day)
		case r < pSucc+pCoup:
			g.poStart(st.ID, "coup", day)
		case r < pSucc+pCoup+pRev:
			g.poStart(st.ID, "revolution", day)
		}
	}
	g.poContractsDue(day)
	g.poTaskDue(day)
	g.poStipends(day)
}

func poTurn(g *Game) {
	if g.W == nil || g.Cur == nil {
		return
	}
	po := g.po()
	day := g.poDay()
	if po.LastDay == 0 {
		po.LastDay = day
		return
	}
	if day <= po.LastDay {
		return
	}
	if day-po.LastDay > 120 {
		po.LastDay = day - 120
	}
	for d := po.LastDay + 1; d <= day; d++ {
		g.poDaily(d)
	}
	po.LastDay = day
}

func (g *Game) poStipends(day int) {
	po := g.po()
	for realm, oi := range po.Office {
		if day-po.Paid[realm] < 7 {
			continue
		}
		if po.Paid[realm] == 0 {
			po.Paid[realm] = day
			continue
		}
		po.Paid[realm] = day
		pay := 12 + 8*g.poRank(realm)
		if fid := po.Back[realm]; fid != "" && po.Fav[fid] >= 30 {
			pay += 10
		}
		g.P.Gold += pay
		g.Msg(fmt.Sprintf("{{G|Stipend as %s of %s: %d %s.}}", poSkinFor(g.Genre).Offices[oi], g.realmName(realm), pay, g.Sk.Currency))
	}
}

func (g *Game) poRank(realm int) int {
	d := g.po().Done[realm]
	switch {
	case d >= 10:
		return 2
	case d >= 4:
		return 1
	}
	return 0
}

func (g *Game) poPostName(realm int) string {
	oi, ok := g.po().Office[realm]
	if !ok {
		return ""
	}
	return poRankName[g.poRank(realm)] + " " + poSkinFor(g.Genre).Offices[oi]
}

func poSide(g *Game) []string {
	if g.W == nil {
		return nil
	}
	var out []string
	if r := g.poRealmQuiet(); r > 0 {
		if cr := g.poCrisisAt(r); cr != nil {
			out = append(out, "{{R|Unrest: "+cr.Kind+"}}")
		}
		if n := g.poPostName(r); n != "" {
			out = append(out, "{{K|"+n+"}}")
		}
	}
	return out
}

// poTalk: townsfolk of a realm will tell you how the court stands.
func poTalk(g *Game, a *Actor) []Option {
	if a == nil || a.State <= 0 || a.ID%3 != 0 || g.W.stateByID[a.State] == nil {
		return nil
	}
	return []Option{{Text: "{{y|What of the court?}}", Do: func() { g.talkMenu(strings.Join(g.poNews(a.State), " ")) }}}
}

func (g *Game) poNews(realm int) []string {
	st := g.W.stateByID[realm]
	if st == nil {
		return []string{"Nobody speaks of the court."}
	}
	sk := poSkinFor(g.Genre)
	out := []string{fmt.Sprintf("%s sits as %s over %s.", g.poRulerName(realm), strings.ToLower(sk.Ruler), g.poCouncil(realm))}
	if cr := g.poCrisisAt(realm); cr != nil {
		out = append(out, "The realm is in "+poCrisisWord[cr.Kind]+".")
	}
	for _, f := range g.warsOf(realm) {
		if g.poPeace(realm, f) {
			out = append(out, "A truce holds with "+g.realmName(f)+".")
		} else {
			out = append(out, "The war with "+g.realmName(f)+" drags on.")
		}
	}
	if mv := swMovementOf(st); mv != "" {
		out = append(out, "The "+mv+" are loud in the streets.")
	}
	return out
}

// ---- tasks ----

func (g *Game) poRealmBurgs(realm int) []*BurgData {
	var out []*BurgData
	for i := range g.W.Burgs {
		b := &g.W.Burgs[i]
		if b.State == realm && b.Alive && b.Layer == "surface" {
			out = append(out, b)
		}
	}
	return out
}

func (g *Game) poPickBurg(realm int, r *Rng, notCapital bool) *BurgData {
	st := g.W.stateByID[realm]
	bs := g.poRealmBurgs(realm)
	px, py := 0, 0
	if g.Cur != nil {
		px, py = g.Cur.WX, g.Cur.WY
	}
	dist := func(b *BurgData) int { return abs(b.P%g.W.W-px) + abs(b.P/g.W.W-py) }
	var c []*BurgData
	for _, b := range bs {
		if notCapital && st != nil && b.ID == st.Capital {
			continue
		}
		c = append(c, b)
	}
	if len(c) == 0 {
		return g.warBurg(realm)
	}
	sort.SliceStable(c, func(i, j int) bool { return dist(c[i]) < dist(c[j]) })
	return c[r.Int(0, poMin(len(c), 5)-1)]
}

func (g *Game) poNewTask(realm, oi int) *poTask {
	st := g.W.stateByID[realm]
	r := NewRng(fmt.Sprintf("%s:po:task:%d:%d:%d", g.Seed, realm, g.po().Done[realm], g.poDay()))
	t := &poTask{Office: oi, Realm: realm, Due: g.poDay() + 12}
	capital := 0
	if st != nil {
		capital = st.Capital
	}
	switch oi {
	case 1:
		bd := g.poPickBurg(realm, r, true)
		t.Kind, t.Burg, t.Pay = "levy", bd.ID, 50+clampi(int(bd.Pop/100), 0, 120)
	case 0:
		foes := g.warsOf(realm)
		var live []int
		for _, f := range foes {
			if !g.poPeace(realm, f) {
				live = append(live, f)
			}
		}
		if len(live) > 0 {
			f := live[r.Int(0, len(live)-1)]
			t.Kind, t.Foe, t.Left, t.Pay = "parley", f, 2, 160+g.P.Level*10
			if b := g.warBurg(f); b != nil {
				t.Burg = b.ID
			}
		} else {
			best, br := 0, -1000
			if st != nil {
				for id, rel := range st.Relations {
					if s := g.W.stateByID[id]; s != nil && s.Alive && id != realm && rel > br {
						best, br = id, rel
					}
				}
			}
			if best == 0 {
				t.Kind, t.Burg, t.Pay = "mediate", capital, 70
			} else {
				t.Kind, t.Foe, t.Pay = "embassy", best, 100
				if b := g.warBurg(best); b != nil {
					t.Burg = b.ID
				}
			}
		}
	case 2:
		bd := g.poPickBurg(realm, r, true)
		need := r.Int(20, 50)
		t.Kind, t.Burg, t.Cost, t.Pay = "muster", bd.ID, need*2, need*5+60
		t.Foe = need
	case 3:
		bd := g.poPickBurg(realm, r, true)
		plotter := g.poGenName(realm, fmt.Sprintf("plot%d", g.po().Done[realm]))
		t.Kind, t.Burg, t.Left, t.Pay, t.Plot, t.PlotID = "probe", bd.ID, 2, 120+g.P.Level*10, plotter, -(realm*1000 + 500 + g.po().Done[realm])
	default:
		bd := g.poPickBurg(realm, r, true)
		t.Kind, t.Burg, t.Pay, t.Left = "mediate", bd.ID, 80+g.P.Level*5, 2
	}
	return t
}

func (g *Game) poBurgName(id int) string {
	if b := g.W.burgByID[id]; b != nil {
		return b.Name
	}
	return "a town"
}

func (g *Game) poTaskText(t *poTask) string {
	r := g.realmName(t.Realm)
	switch t.Kind {
	case "levy":
		if t.Stage == 0 {
			return fmt.Sprintf("Collect the levy in %s, then bring it to %s.", g.poBurgName(t.Burg), g.poBurgName(t.Burg))
		}
		return "Deliver the levy to the treasury at " + g.poBurgName(t.Burg) + "."
	case "parley":
		return fmt.Sprintf("Negotiate a treaty with %s at %s (%d tries left).", g.realmName(t.Foe), g.poBurgName(t.Burg), t.Left)
	case "embassy":
		return fmt.Sprintf("Open an embassy of %s at %s.", r, g.poBurgName(t.Burg))
	case "probe":
		return fmt.Sprintf("Uncover the conspiracy around %s in %s (%d tries left).", t.Plot, g.poBurgName(t.Burg), t.Left)
	case "muster":
		if t.Stage == 0 {
			return fmt.Sprintf("Raise %d soldiers at %s (costs %d %s).", t.Foe, g.poBurgName(t.Burg), t.Cost, g.Sk.Currency)
		}
		if len(g.warsOf(t.Realm)) > 0 {
			return fmt.Sprintf("Bring the %d soldiers to a frontier zone of %s.", t.Foe, r)
		}
		return fmt.Sprintf("Present the %d soldiers at %s for review.", t.Foe, g.poBurgName(t.Burg))
	}
	return fmt.Sprintf("Settle the quarrel of the faithful in %s (%d tries left).", g.poBurgName(t.Burg), t.Left)
}

func (g *Game) poAtBurg(id int) bool {
	return g.Cur != nil && g.Cur.Kind == "town" && g.Cur.Burg == id
}

func (g *Game) poTaskDone(t *poTask, note string) {
	po := g.po()
	po.Done[t.Realm]++
	po.Stand[t.Realm] = clampi(po.Stand[t.Realm]+4, -50, 100)
	g.P.Gold += t.Pay
	g.gainXP(t.Pay / 2)
	g.AddRep(stateFaction(t.Realm), 25, " for service at court")
	g.Msg(fmt.Sprintf("{{G|%s %d %s.}}", note, t.Pay, g.Sk.Currency))
	g.deed("quest", fmt.Sprintf("%s served %s as %s: %s", g.P.Name, g.realmName(t.Realm), g.poPostName(t.Realm), strings.ToLower(note)), 0, 0)
	g.poLog(fmt.Sprintf("%s: %s", g.realmName(t.Realm), note))
	po.Task = nil
}

func (g *Game) poTaskFail(t *poTask, why string) {
	po := g.po()
	po.Stand[t.Realm] = clampi(po.Stand[t.Realm]-3, -50, 100)
	g.Rep[stateFaction(t.Realm)] -= 20
	g.Msg("{{R|Task failed: " + why + "}}")
	g.deed("quest", fmt.Sprintf("%s failed a task for %s: %s", g.P.Name, g.realmName(t.Realm), why), 0, 0)
	po.Task = nil
}

func (g *Game) poTaskDue(day int) {
	po := g.po()
	if t := po.Task; t != nil && day > t.Due {
		g.poTaskFail(t, "the time allowed ran out.")
	}
}

// poStep acts on the current task at the player's location.
func (g *Game) poStep() {
	po := g.po()
	t := po.Task
	if t == nil {
		return
	}
	rs := NewRng(fmt.Sprintf("%s:po:step:%d:%d:%d", g.Seed, t.Realm, t.Stage, g.Turn))
	capital := 0
	if st := g.W.stateByID[t.Realm]; st != nil {
		capital = st.Capital
	}
	chance := func(base float64, stat int) float64 {
		return poClampf(base+0.05*float64(g.P.M(stat)), 0.1, 0.92)
	}
	switch t.Kind {
	case "levy":
		if !g.poAtBurg(t.Burg) {
			g.Msg("You must be in " + g.poBurgName(t.Burg) + ".")
			return
		}
		if t.Stage == 0 {
			t.Stage, t.Burg = 1, capital
			g.Msg("The levy chest is loaded. Carry it to " + g.poBurgName(capital) + ".")
			return
		}
		g.poAsk("You stand before the treasury with the chest.", Option{"Hand it over in full.", func() { g.poTaskDone(t, "The treasury is satisfied; you are paid") }},
			Option{"{{R|Skim a third.}}", func() {
				cut := t.Pay
				if rs.Chance(0.35 + float64(g.poCourt(t.Realm)[3].Int)*0.03) {
					g.Rep[stateFaction(t.Realm)] -= 80
					g.uwAddBounty(t.Realm, 80, " (embezzlement)")
					delete(po.Office, t.Realm)
					g.deed("crime", fmt.Sprintf("%s was caught skimming the levy of %s and lost the post.", g.P.Name, g.realmName(t.Realm)), 0, 0)
					po.Task = nil
					g.Msg("{{R|The spymaster's clerks count the chest. You are dismissed.}}")
					return
				}
				t.Pay += cut / 2
				g.poTaskDone(t, "The theft goes unnoticed; you pocket")
			}})
	case "embassy":
		if !g.poAtBurg(t.Burg) {
			g.Msg("You must be in " + g.poBurgName(t.Burg) + ".")
			return
		}
		g.AddRep(stateFaction(t.Foe), 25, " for the embassy")
		g.poTaskDone(t, "The embassy opens; you are paid")
	case "parley":
		if !g.poAtBurg(t.Burg) {
			g.Msg("You must be in " + g.poBurgName(t.Burg) + ".")
			return
		}
		try := func(bonus float64, gift int) {
			if gift > 0 {
				if g.P.Gold < gift {
					g.Msg("You cannot afford the gift.")
					return
				}
				g.P.Gold -= gift
			}
			p := chance(0.30, EGO) + bonus + float64(g.Rep[stateFaction(t.Foe)])/1500 + float64(g.poCourt(t.Realm)[0].Dip)*0.015
			if rs.Chance(poClampf(p, 0.1, 0.9)) {
				po.Peace[poPeaceKey(t.Realm, t.Foe)] = g.poDay() + 2*YearDays
				g.AddRep(stateFaction(t.Foe), 30, " for the peace")
				g.poLog(fmt.Sprintf("A truce between %s and %s, by %s.", g.realmName(t.Realm), g.realmName(t.Foe), g.P.Name))
				g.poTaskDone(t, fmt.Sprintf("The treaty is sealed between %s and %s; you are paid", g.realmName(t.Realm), g.realmName(t.Foe)))
				return
			}
			t.Left--
			if t.Left <= 0 {
				g.poTaskFail(t, "the envoys walked out.")
				return
			}
			g.Msg("{{O|The envoys refuse. You may try once more.}}")
		}
		g.poAsk(fmt.Sprintf("The envoys of %s will hear your terms.", g.realmName(t.Foe)),
			Option{"Press the terms with words alone.", func() { try(0, 0) }},
			Option{"Sweeten them with a gift (100).", func() { try(0.15, 100) }})
	case "probe":
		if !g.poAtBurg(t.Burg) {
			g.Msg("You must be in " + g.poBurgName(t.Burg) + ".")
			return
		}
		if !rs.Chance(chance(0.35, INT)) {
			t.Left--
			if t.Left <= 0 {
				g.poTaskFail(t, "the trail went cold.")
				return
			}
			g.Msg("{{O|You find nothing yet. The trail is warm; try again.}}")
			return
		}
		g.poAsk(fmt.Sprintf("Letters and witnesses name %s as the head of a plot against the seat.", t.Plot),
			Option{"Expose the plot to the court.", func() {
				g.poFaction(t.Realm, "sword", -5)
				g.poTaskDone(t, "The conspirators are taken; you are rewarded with")
			}},
			Option{"{{R|Blackmail the plotter.}}", func() {
				t.Pay += 120
				g.Msg("{{O|" + t.Plot + " pays for your silence.}}")
				g.poFaction(t.Realm, "sword", 10)
				po.Task = nil
				po.Done[t.Realm]++
				g.P.Gold += t.Pay
				g.deed("crime", fmt.Sprintf("%s sold their silence about %s's plot in %s.", g.P.Name, t.Plot, g.realmName(t.Realm)), 0, 0)
			}},
			Option{"{{R|Join the plot.}}", func() {
				g.Rep[stateFaction(t.Realm)] -= 60
				g.poFaction(t.Realm, "sword", 25)
				po.Task = nil
				if g.poCrisisAt(t.Realm) == nil {
					g.poStart(t.Realm, "coup", g.poDay())
					if cr := g.poCrisisAt(t.Realm); cr != nil {
						cr.Bias -= 20
					}
				}
				g.deed("crime", fmt.Sprintf("%s joined %s's conspiracy against the seat of %s.", g.P.Name, t.Plot, g.realmName(t.Realm)), 0, 0)
			}})
	case "muster":
		if t.Stage == 0 {
			if !g.poAtBurg(t.Burg) {
				g.Msg("You must be in " + g.poBurgName(t.Burg) + ".")
				return
			}
			if g.P.Gold < t.Cost {
				g.Msg(fmt.Sprintf("You need %d %s to equip them.", t.Cost, g.Sk.Currency))
				return
			}
			g.P.Gold -= t.Cost
			t.Stage = 1
			t.Pay += t.Cost
			g.Msg(fmt.Sprintf("{{G|%d soldiers answer the call and fall in behind you.}}", t.Foe))
			return
		}
		if len(g.warsOf(t.Realm)) > 0 {
			if own, _, ok := g.frontierAt(g.Cur.WX, g.Cur.WY); !ok || own != t.Realm || !g.isFrontier(g.Cur) {
				g.Msg("The contingent must march to a frontier zone of " + g.realmName(t.Realm) + ".")
				return
			}
			g.poTaskDone(t, "The contingent reinforces the line; you are paid")
			return
		}
		if !g.poAtBurg(capital) {
			g.Msg("You must be in " + g.poBurgName(capital) + ".")
			return
		}
		g.poTaskDone(t, "The marshal reviews the troops; you are paid")
	default:
		if !g.poAtBurg(t.Burg) {
			g.Msg("You must be in " + g.poBurgName(t.Burg) + ".")
			return
		}
		verdict := func(for_ string) {
			if rs.Chance(chance(0.45, EGO)) {
				g.poFaction(t.Realm, "altar", map[bool]int{true: 8, false: -4}[for_ == "orthodox"])
				g.poTaskDone(t, "The quarrel is settled; you are paid")
				return
			}
			t.Left--
			if t.Left <= 0 {
				g.poTaskFail(t, "both sides left angry.")
				return
			}
			g.Msg("{{O|Neither side accepts the ruling. Try again.}}")
		}
		g.poAsk("Two sects of the faithful quarrel here over rites.", Option{"Rule for the orthodox.", func() { verdict("orthodox") }},
			Option{"Rule for the reformers.", func() { verdict("reform") }})
	}
}

func poClampf(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// poFaction shifts favour with a realm's faction of a kind.
func (g *Game) poFaction(realm int, kind string, d int) {
	if f := poFacKind(g.poFacs(realm), kind); f != nil {
		po := g.po()
		po.Fav[f.ID] = clampi(po.Fav[f.ID]+d, -100, 100)
	}
}

func (g *Game) poAsk(text string, opts ...Option) {
	g.po().Ask = &poAsk{Text: text, Opts: opts}
}

// ---- audience, offices, intrigue ----

func (g *Game) poCapitalHere(realm int) bool {
	st := g.W.stateByID[realm]
	return st != nil && g.poAtBurg(st.Capital)
}

func (g *Game) poAudience(realm int) {
	po := g.po()
	st := g.W.stateByID[realm]
	if st == nil {
		return
	}
	switch {
	case !g.poCapitalHere(realm):
		g.Msg("The ruler holds court in " + g.poBurgName(st.Capital) + ".")
		return
	case g.poCrisisAt(realm) != nil && g.poCrisisAt(realm).Kind != "succession":
		g.Msg("{{R|The palace gates are barred; the realm is in upheaval.}}")
		return
	case g.poCrisisAt(realm) != nil:
		g.Msg("{{O|The seat is empty; the chamberlain turns petitioners away.}}")
		return
	case g.uwBounty(realm) > 0:
		g.Msg("{{R|The guards will not admit a wanted " + g.P.Name + ".}}")
		return
	case g.Rep[stateFaction(realm)] < -50:
		g.Msg("{{R|The ruler will not see you.}}")
		return
	case po.Aud[realm] > 0 && g.poDay()+1-po.Aud[realm] < 1:
		g.Msg("The ruler has heard you today.")
		return
	}
	first := po.Aud[realm] == 0
	po.Aud[realm] = g.poDay() + 1
	po.Stand[realm] = clampi(po.Stand[realm]+1, -50, 100)
	if first {
		g.Rep[stateFaction(realm)] += 5
	}
	sk := poSkinFor(g.Genre)
	ruler := g.poRulerName(realm)
	g.deed("event", fmt.Sprintf("%s was received by %s %s of %s.", g.P.Name, strings.ToLower(sk.Ruler), ruler, st.Name), 0, 0)
	g.poAsk(fmt.Sprintf("%s, %s of %s, receives you in %s. %s", ruler, strings.ToLower(sk.Ruler), st.Name, g.poCouncil(realm), poMood(g.Rep[stateFaction(realm)])),
		Option{"Ask what the court needs.", func() { g.poSay(strings.Join(g.poNews(realm), " ")) }},
		Option{"Present a gift (100).", func() {
			if g.P.Gold < 100 {
				g.Msg("You cannot afford the gift.")
				return
			}
			g.P.Gold -= 100
			g.AddRep(stateFaction(realm), 25, " for the gift")
			g.po().Stand[realm] = clampi(g.po().Stand[realm]+2, -50, 100)
			g.deed("event", fmt.Sprintf("%s laid a gift before %s.", g.P.Name, ruler), 0, 0)
		}},
		Option{"Withdraw.", func() {}})
}

func poMood(rep int) string {
	switch {
	case rep >= 250:
		return "The ruler smiles; you are well spoken of."
	case rep >= 0:
		return "The ruler regards you with polite interest."
	}
	return "The ruler's eyes are cold."
}

func (g *Game) poSay(text string) {
	g.po().Ask = &poAsk{Text: text, Opts: []Option{{Text: "Go on.", Do: func() {}}}}
}

func (g *Game) poPetition(realm, oi int) {
	po := g.po()
	if _, ok := po.Office[realm]; ok {
		g.Msg("You already hold a post in " + g.realmName(realm) + ".")
		return
	}
	if po.Aud[realm] == 0 || g.poDay()+1-po.Aud[realm] > 3 {
		g.Msg("A petition must follow an audience with the ruler (within three days).")
		return
	}
	fee := 80 + g.P.Level*10
	best := -100
	for _, f := range g.poFacs(realm) {
		if po.Fav[f.ID] > best {
			best = po.Fav[f.ID]
		}
	}
	if g.Rep[stateFaction(realm)] < 40 && best < 50 {
		g.Msg("The court does not know you well enough. Earn the realm's regard or a faction's favour first.")
		return
	}
	if g.P.Gold < fee {
		g.Msg(fmt.Sprintf("The chancery fee is %d %s.", fee, g.Sk.Currency))
		return
	}
	g.P.Gold -= fee
	po.Office[realm] = oi
	po.Paid[realm] = g.poDay()
	t := poSkinFor(g.Genre).Offices[oi]
	g.Msg(fmt.Sprintf("{{G|You are sworn in as Deputy %s of %s.}}", t, g.realmName(realm)))
	g.deed("quest", fmt.Sprintf("%s became Deputy %s of %s.", g.P.Name, t, g.realmName(realm)), 0, 0)
	g.poLog(fmt.Sprintf("%s took a post as Deputy %s in %s.", g.P.Name, t, g.realmName(realm)))
}

func (g *Game) poHolder(realm, oi int) poMember { return g.poCourt(realm)[oi] }

func (g *Game) poBribe(realm, oi int) {
	po := g.po()
	h := g.poHolder(realm, oi)
	cost := 50 + h.Ste*10 + g.P.Level*8
	if g.P.Gold < cost {
		g.Msg(fmt.Sprintf("%s's price is %d %s.", h.Name, cost, g.Sk.Currency))
		return
	}
	g.P.Gold -= cost
	rs := NewRng(fmt.Sprintf("%s:po:bribe:%d:%d:%d", g.Seed, h.ID, g.Turn, g.P.Gold))
	fid := fmt.Sprintf("po:%d:%s", realm, poFactionKinds[poOfficeFac[oi]])
	switch {
	case rs.Chance(0.15 + 0.03*float64(h.Int)):
		g.AddRep(stateFaction(realm), -60, " (the bribe was exposed)")
		g.uwAddBounty(realm, 50, " (bribery)")
		po.Fav[fid] = clampi(po.Fav[fid]-15, -100, 100)
		g.deed("crime", fmt.Sprintf("%s was caught bribing %s of %s.", g.P.Name, h.Name, g.realmName(realm)), 0, 0)
	case rs.Chance(poClampf(0.6+0.04*float64(g.P.M(EGO))-0.03*float64(h.Int), 0.2, 0.9)):
		po.Fav[fid] = clampi(po.Fav[fid]+15, -100, 100)
		po.Stand[realm] = clampi(po.Stand[realm]+2, -50, 100)
		g.Msg(fmt.Sprintf("{{G|%s pockets the purse and is your friend.}}", h.Name))
		g.deed("crime", fmt.Sprintf("%s bribed %s, %s of %s.", g.P.Name, h.Name, h.Title, g.realmName(realm)), 0, 0)
	default:
		g.Rep[stateFaction(realm)] -= 10
		g.Msg(fmt.Sprintf("{{O|%s keeps the purse and promises nothing.}}", h.Name))
	}
}

func (g *Game) poDig(realm, oi int) {
	po := g.po()
	h := g.poHolder(realm, oi)
	for i := 0; i < 30 && !g.Dead; i++ {
		g.spend(1000)
	}
	rs := NewRng(fmt.Sprintf("%s:po:dig:%d:%d", g.Seed, h.ID, g.Turn))
	if rs.Chance(poClampf(0.45+0.05*float64(g.P.M(INT))-0.03*float64(h.Int), 0.1, 0.9)) {
		po.Dirt[h.ID]++
		line := fmt.Sprintf(poSecrets[rs.Int(0, len(poSecrets)-1)], h.Name)
		g.Msg("{{G|You learn: " + line + "}}")
		g.deed("crime", fmt.Sprintf("%s dug up a secret about %s.", g.P.Name, h.Name), 0, 0)
		return
	}
	g.Msg(fmt.Sprintf("{{O|Hours of whispers turn up nothing on %s.}}", h.Name))
}

func (g *Game) poBlackmail(realm, oi int) {
	po := g.po()
	h := g.poHolder(realm, oi)
	if po.Dirt[h.ID] <= 0 {
		g.Msg("You have nothing on " + h.Name + ". Dig first.")
		return
	}
	po.Dirt[h.ID]--
	rs := NewRng(fmt.Sprintf("%s:po:bm:%d:%d", g.Seed, h.ID, g.Turn))
	fid := fmt.Sprintf("po:%d:%s", realm, poFactionKinds[poOfficeFac[oi]])
	if rs.Chance(poClampf(0.6+0.05*float64(g.P.M(EGO))-0.04*float64(h.Int), 0.2, 0.9)) {
		gold := 80 + h.Ste*12
		g.P.Gold += gold
		po.Fav[fid] = clampi(po.Fav[fid]-10, -100, 100)
		po.Stand[realm] = clampi(po.Stand[realm]+2, -50, 100)
		g.Msg(fmt.Sprintf("{{G|%s pays %d %s to keep the secret.}}", h.Name, gold, g.Sk.Currency))
		g.deed("crime", fmt.Sprintf("%s blackmailed %s of %s.", g.P.Name, h.Name, g.realmName(realm)), 0, 0)
		return
	}
	g.AddRep(stateFaction(realm), -50, " (the blackmail was reported)")
	g.uwAddBounty(realm, 100, " (blackmail)")
	g.deed("crime", fmt.Sprintf("%s's blackmail of %s backfired.", g.P.Name, h.Name), 0, 0)
}

func (g *Game) poContract(realm, oi int, cover bool) {
	po := g.po()
	h := g.poHolder(realm, oi)
	for _, c := range po.Contracts {
		if c.Target == h.ID {
			g.Msg("A knife is already out for " + h.Name + ".")
			return
		}
	}
	cost := 250 + (h.Mar+h.Int)*8
	if cover {
		cost = cost * 3 / 2
	}
	if g.P.Gold < cost {
		g.Msg(fmt.Sprintf("The contract on %s costs %d %s.", h.Name, cost, g.Sk.Currency))
		return
	}
	g.P.Gold -= cost
	rs := NewRng(fmt.Sprintf("%s:po:con:%d:%d", g.Seed, h.ID, g.Turn))
	po.Contracts = append(po.Contracts, &poContract{Target: h.ID, Name: h.Name, Realm: realm, Due: g.poDay() + rs.Int(2, 5), Cost: cost, Cover: cover})
	g.Msg(fmt.Sprintf("{{R|A contract is let on %s. It will be done in a few days, or not.}}", h.Name))
	g.deed("crime", fmt.Sprintf("%s paid for the death of %s, %s of %s.", g.P.Name, h.Name, h.Title, g.realmName(realm)), 0, 0)
}

func (g *Game) poContractsDue(day int) {
	po := g.po()
	var keep []*poContract
	for _, c := range po.Contracts {
		if c.Due > day {
			keep = append(keep, c)
			continue
		}
		rs := NewRng(fmt.Sprintf("%s:po:hit:%d:%d", g.Seed, c.Target, c.Due))
		var intr int
		for _, m := range g.poCourt(c.Realm) {
			if m.ID == c.Target {
				intr = m.Int
			}
		}
		ok := rs.Chance(poClampf(0.35+float64(c.Cost)/1000-0.02*float64(intr), 0.15, 0.9))
		traced := rs.Chance(0.30)
		if c.Cover {
			traced = rs.Chance(0.08)
		}
		if ok {
			po.Dead[c.Target] = day + 1
			po.Hit[fmt.Sprintf("po:%d:sword", c.Realm)] -= 0
			g.poLog(fmt.Sprintf("%s of %s is found dead.", c.Name, g.realmName(c.Realm)))
			g.Msg(fmt.Sprintf("{{O|%s of %s is dead. The contract is fulfilled.}}", c.Name, g.realmName(c.Realm)))
			g.deed("crime", fmt.Sprintf("%s of %s was murdered by the contract of %s.", c.Name, g.realmName(c.Realm), g.P.Name), 0, 0)
		} else {
			g.Msg(fmt.Sprintf("{{O|The attempt on %s failed.}}", c.Name))
			traced = traced && rs.Chance(0.6)
		}
		if traced {
			g.Rep[stateFaction(c.Realm)] -= 200
			g.uwAddBounty(c.Realm, 400, " (murder of "+c.Name+")")
			g.deed("crime", fmt.Sprintf("The killing of %s was traced to %s.", c.Name, g.P.Name), 0, 0)
		}
	}
	po.Contracts = keep
}

// ---- the screen ----

func poKeys(g *Game, k string) {
	po := g.po()
	if po.Ask != nil {
		ask := po.Ask
		if i := letterIndex(k); i >= 0 && i < len(ask.Opts) {
			po.Ask = nil
			ask.Opts[i].Do()
			return
		}
		if k == "Escape" {
			po.Ask = nil
		}
		return
	}
	realm := g.poRealm()
	switch k {
	case "Escape", "A", "q":
		g.Mode = "main"
		return
	case "Tab", "ArrowRight":
		g.Tab, g.Sel = (g.Tab+1)%3, 0
		return
	case "ArrowLeft":
		g.Tab, g.Sel = (g.Tab+2)%3, 0
		return
	case "ArrowUp", "N8", "k":
		g.Sel = maxi(0, g.Sel-1)
		return
	case "ArrowDown", "N2", "j":
		g.Sel++
		return
	}
	if g.W.stateByID[realm] == nil {
		return
	}
	switch g.Tab {
	case 0:
		g.Sel = clampi(g.Sel, 0, 5)
		if g.Sel == 0 {
			if k == "Enter" || k == "e" {
				g.poAudience(realm)
			}
			return
		}
		oi := g.Sel - 1
		switch k {
		case "Enter", "e":
			if g.P.Gold < 30 {
				g.Msg("A private word costs 30 " + g.Sk.Currency + ".")
				return
			}
			g.P.Gold -= 30
			fid := fmt.Sprintf("po:%d:%s", realm, poFactionKinds[poOfficeFac[oi]])
			po.Fav[fid] = clampi(po.Fav[fid]+5, -100, 100)
			g.Msg("{{G|A courteous word, and a purse, buy a little warmth.}}")
		case "b":
			g.poBribe(realm, oi)
		case "d":
			g.poDig(realm, oi)
		case "m":
			g.poBlackmail(realm, oi)
		case "x":
			g.poContract(realm, oi, false)
		case "X":
			g.poContract(realm, oi, true)
		}
	case 1:
		fs := g.poFacs(realm)
		if len(fs) == 0 {
			return
		}
		g.Sel = clampi(g.Sel, 0, len(fs)-1)
		f := fs[g.Sel]
		switch k {
		case "Enter", "e":
			cost := 40 + g.P.Level*5
			if g.P.Gold < cost {
				g.Msg(fmt.Sprintf("Winning favour costs %d %s.", cost, g.Sk.Currency))
				return
			}
			g.P.Gold -= cost
			po.Fav[f.ID] = clampi(po.Fav[f.ID]+8, -100, 100)
			g.Msg("{{G|Gifts and good words raise your name among " + f.Name + ".}}")
		case "b":
			if po.Back[realm] == f.ID {
				delete(po.Back, realm)
				g.Msg("You withdraw your backing.")
				return
			}
			po.Back[realm] = f.ID
			for _, o := range fs {
				if o.ID != f.ID {
					po.Fav[o.ID] = clampi(po.Fav[o.ID]-5, -100, 100)
				}
			}
			po.Fav[f.ID] = clampi(po.Fav[f.ID]+10, -100, 100)
			g.Msg("{{Y|You declare for " + f.Name + ".}}")
			g.deed("event", fmt.Sprintf("%s threw in with %s in %s.", g.P.Name, f.Name, g.realmName(realm)), 0, 0)
		case "p":
			cr := g.poCrisisAt(realm)
			if cr == nil || (f.ID != cr.A && f.ID != cr.B) {
				g.Msg("That faction is not in the contest.")
				return
			}
			cost := 100 + g.P.Level*10
			if g.P.Gold < cost {
				g.Msg(fmt.Sprintf("Funding the cause costs %d %s.", cost, g.Sk.Currency))
				return
			}
			g.P.Gold -= cost
			if f.ID == cr.A {
				cr.Bias = poMin(cr.Bias+8, 40)
			} else {
				cr.Bias = maxi(cr.Bias-8, -40)
			}
			po.Fav[f.ID] = clampi(po.Fav[f.ID]+20, -100, 100)
			g.Msg("{{Y|Your coin buys swords and voices for " + f.Name + ".}}")
			g.deed("event", fmt.Sprintf("%s funded %s in the %s of %s.", g.P.Name, f.Name, cr.Kind, g.realmName(realm)), 0, 0)
		}
	default:
		g.Sel = clampi(g.Sel, 0, 4)
		switch k {
		case "Enter", "e":
			if po.Task != nil {
				g.poStep()
			} else if oi, ok := po.Office[realm]; ok {
				po.Task = g.poNewTask(realm, oi)
				g.Msg("{{Y|Task: " + g.poTaskText(po.Task) + "}}")
			} else {
				g.poPetition(realm, g.Sel)
			}
		case "c":
			if t := po.Task; t != nil {
				g.poTaskFail(t, "you gave it up.")
			}
		case "R":
			if _, ok := po.Office[realm]; ok {
				delete(po.Office, realm)
				if po.Task != nil && po.Task.Realm == realm {
					po.Task = nil
				}
				g.Msg("You resign your post.")
				g.deed("quest", fmt.Sprintf("%s resigned a post at the court of %s.", g.P.Name, g.realmName(realm)), 0, 0)
			}
		}
	}
}

func (g *Game) poDraw() {
	s := g.sc
	po := g.po()
	x, y, w, h := g.panel("Court & Politics")
	bottom := y + h - 1
	tabs := []string{"Court", "Factions", "Offices & Tasks"}
	tx := x
	for i, t := range tabs {
		c := "K"
		if g.Tab == i {
			c = "Y"
		}
		tx = s.TextB(tx, y, fmt.Sprintf("{{%s|[ %s ]}} ", c, t), CGray, CBgDark, x+w)
	}
	y += 2
	put := func(str string) {
		for _, l := range Wrap(str, w) {
			if y < bottom {
				s.TextB(x, y, l, CGray, CBgDark, x+w)
				y++
			}
		}
	}
	row := func(sel bool, str string) {
		if y >= bottom {
			return
		}
		bg := CBgDark
		if sel {
			bg = CBgSel
		}
		s.Fill(x, y, w, 1, bg)
		s.TextB(x, y, str, CGray, bg, x+w)
		y++
	}
	realm := g.poRealm()
	st := g.W.stateByID[realm]
	if st == nil {
		put("{{K|You know no court here.}}")
		return
	}
	if a := po.Ask; a != nil {
		put("{{W|" + st.FullName + "}}")
		y++
		put(a.Text)
		y++
		for i, o := range a.Opts {
			put(fmt.Sprintf("{{Y|%c)}} %s", 'a'+i, o.Text))
		}
		y++
		put("{{K|Esc leaves}}")
		return
	}
	sk := poSkinFor(g.Genre)
	put(fmt.Sprintf("{{W|%s}} {{K|(%s, %s)}}", st.FullName, nonEmpty(st.Form, "realm"), nonEmpty(st.Ideology, "unnamed rule")))
	if cr := g.poCrisisAt(realm); cr != nil {
		put(fmt.Sprintf("{{R|In %s, ends in %d days.}}", poCrisisWord[cr.Kind], maxi(0, cr.End-g.poDay())))
	}
	if n := g.poPostName(realm); n != "" {
		put(fmt.Sprintf("{{G|You are %s.}} {{K|Standing %d, %d tasks done.}}", n, po.Stand[realm], po.Done[realm]))
	}
	switch g.Tab {
	case 0:
		put(fmt.Sprintf("{{K|%s}}", g.poCouncil(realm)))
		court := g.poCourt(realm)
		g.Sel = clampi(g.Sel, 0, 5)
		row(g.Sel == 0, fmt.Sprintf("{{Y|%-20s}} %s", sk.Ruler, g.poRulerName(realm)))
		for i, m := range court {
			extra := ""
			if po.Dirt[m.ID] > 0 {
				extra = " {{O|[dirt]}}"
			}
			for _, c := range po.Contracts {
				if c.Target == m.ID {
					extra += " {{R|[contract]}}"
				}
			}
			row(g.Sel == i+1, fmt.Sprintf("{{C|%-20s}} %s {{K|D%d M%d S%d I%d}}%s", m.Title, m.Name, m.Dip, m.Mar, m.Ste, m.Int, extra))
		}
		y++
		if g.Sel == 0 {
			put("{{K|Enter: seek an audience (in the capital).}}")
		} else {
			put("{{K|" + poOfficeAbout[g.Sel-1] + ".}}")
			put("{{K|Enter: a private word (30). b bribe  d dig for dirt  m blackmail  x hire a knife  X hire it with cover.}}")
		}
	case 1:
		fs := g.poFacs(realm)
		g.Sel = clampi(g.Sel, 0, maxi(0, len(fs)-1))
		for i, f := range fs {
			mark := " "
			if po.Back[realm] == f.ID {
				mark = "*"
			}
			bar := strings.Repeat("#", f.Str/10) + strings.Repeat(".", 10-f.Str/10)
			row(g.Sel == i, fmt.Sprintf("%s{{C|%-26s}} {{K|%s}} %s", mark, shorten(f.Name, 26), bar, poFavWord(po.Fav[f.ID])))
		}
		y++
		if g.Sel < len(fs) {
			f := fs[g.Sel]
			put(fmt.Sprintf("{{W|Leader:}} %s. This faction %s.", f.Leader, poFactionBent[f.Kind]))
		}
		if cr := g.poCrisisAt(realm); cr != nil {
			a, b := poFacByID(fs, cr.A), poFacByID(fs, cr.B)
			if a != nil && b != nil {
				put(fmt.Sprintf("{{R|The contest:}} %s against %s (your push %+d).", a.Name, b.Name, cr.Bias))
			}
		}
		put("{{K|Enter: win favour. b: back (*) or withdraw. p: fund a side in the contest.}}")
	default:
		oi, held := po.Office[realm]
		g.Sel = clampi(g.Sel, 0, 4)
		if t := po.Task; t != nil {
			put("{{W|Task:}} " + g.poTaskText(t))
			put(fmt.Sprintf("{{K|Pay %d %s; %d days left. Enter acts here, c gives it up.}}", t.Pay, g.Sk.Currency, maxi(0, t.Due-g.poDay())))
		} else if held {
			put(fmt.Sprintf("{{K|No task. Enter takes one from the %s's desk. R resigns.}}", strings.ToLower(sk.Offices[oi])))
		} else {
			put("{{K|Hold a post by petitioning after an audience: Enter on an office (fee, regard needed).}}")
		}
		y++
		court := g.poCourt(realm)
		for i, m := range court {
			tag := ""
			if held && oi == i {
				tag = " {{G|[your post]}}"
			}
			row(g.Sel == i, fmt.Sprintf("{{C|%-20s}} %s%s", m.Title, m.Name, tag))
		}
		y++
		put("{{K|" + poOfficeAbout[g.Sel] + ".}}")
	}
	// court news
	y = maxi(y+1, bottom-5)
	s.TextB(x, y-1, "{{K|Court news}}", CGray, CBgDark, x+w)
	start := maxi(0, len(po.Log)-4)
	for _, l := range po.Log[start:] {
		if y < bottom {
			s.TextB(x, y, "{{K|"+shorten(l, w)+"}}", CGray, CBgDark, x+w)
			y++
		}
	}
	s.TextB(x, bottom, "{{K|Tab switches, arrows select, Esc closes}}", CGray, CBgDark, x+w)
}
