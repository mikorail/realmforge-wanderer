package engine

import (
	"fmt"
	"sort"
	"strings"
)

// Per-realm story hooks (prefix rs). Every living realm gets ONE unique ongoing crisis, derived from its
// real data with a side rng (HashStr/NewRng of seed:rs:..., never g.R): wars, famine, persecutions, reform
// movements, genre forces, the ruler and the dead of the dynasty, charters and leagues, outbreaks and the
// chronicle (WorldData.Events). The twelve archetypes (realmstory_data.go) are a succession dispute, a plague
// of unrest, a heresy trial, a cult in the capital, border war weariness, famine and hoarding, a usurper
// general, a trade-guild strangle, a haunted dynasty, a missing heir, ruinous debt and a rebel province.
//
// This is a political and social STATE of the realm with three stages (whisper, open trouble, breaking point)
// that advance with the day clock and with what the player does (Push days), not a plot ladder (campaigns.go)
// or a pacing arc (director.go); politics.go's timed succession/coup/revolution crises are read only. The
// crisis shows in the world as town notices (an inscribed stone in each town's centre, rewritten on entering),
// townsfolk ("Ask about the troubles in X", talk hook), barks through rsLine, and up to three entry points:
// an informant and a document in the capital plus a duel (champion in the capital), a relic in a dungeon near
// it, or a choice card that comes when the trouble peaks. Each opens the choice screen (mode "rs"): side with
// either party, expose, broker, profit, or ignore it (it then worsens, and rsPriceMod raises town prices).
// A resolution changes realm standing, adds a deed and news and leaves an aftermath tag that barks, notices
// and the screen mention. Screen: mode "rs", palette "Realm affairs". State g.ext("rs"), planet-local.

func init() {
	unLocalKeys = append(unLocalKeys, "rs")
	turnHooks = append(turnHooks, rsTurn)
	zoneHooks = append(zoneHooks, rsZone)
	enterHooks = append(enterHooks, rsEnter)
	killHooks = append(killHooks, rsKill)
	talkHooks = append(talkHooks, rsTalk)
	modeKeys["rs"] = rsKeys
	modeDraws["rs"] = func(g *Game) { g.rsDraw() }
	paEntries = append(paEntries, paEntry{Name: "Realm affairs", Keywords: "realm crisis troubles politics story succession unrest heresy cult war famine", Cat: "Screen", Label: "screen",
		Func: func(g *Game) { g.rsOpen("view", 0, "", nil) }})
}

// ---------------------------------------------------------------- state

type rsRealm struct {
	Push    int      `json:"p,omitempty"`  // days the crisis was pushed on (+) or held back (-)
	Known   int      `json:"k,omitempty"`  // 0 nothing, 1 rumour, 2 who, 3 what is hidden
	Res     string   `json:"r,omitempty"`  // resolution: a b x t, empty while open
	ResDay  int      `json:"rd,omitempty"` // day resolved
	Ignored int      `json:"ig,omitempty"`
	Profits int      `json:"pf,omitempty"`
	PStage  int      `json:"ps,omitempty"` // bitmask of stages already profited from
	Seen    int      `json:"sn,omitempty"` // highest stage announced in the news, plus one
	WorseN  bool     `json:"wn,omitempty"`
	Inf     bool     `json:"in,omitempty"`
	Card    bool     `json:"cd,omitempty"`
	Did     []string `json:"d,omitempty"`
	Asked   []int    `json:"as,omitempty"`
}

type rsPending struct {
	ID  int
	Src string
}

type rsState struct {
	Realms  map[int]*rsRealm `json:"r"`
	Notices map[string]int   `json:"n"` // zone key -> tile index of the notice stone

	mode, src, res string
	id             int
	talker         *Actor
	pend           *rsPending
	cycle          int
}

func (g *Game) rs() *rsState {
	s := g.ext("rs", func() any { return &rsState{Realms: map[int]*rsRealm{}, Notices: map[string]int{}} }).(*rsState)
	if s.Realms == nil {
		s.Realms = map[int]*rsRealm{}
	}
	if s.Notices == nil {
		s.Notices = map[string]int{}
	}
	return s
}

func (g *Game) rsRealmOf(id int) *rsRealm {
	s := g.rs()
	r := s.Realms[id]
	if r == nil {
		r = &rsRealm{}
		s.Realms[id] = r
	}
	return r
}

// ---------------------------------------------------------------- derived crisis (never saved)

type rsDef struct {
	A       *rsArch
	St      *StateData
	Cap     *BurgData
	Third   string
	T1, T2  int // day thresholds for stages 1 and 2
	TW      int // day the unresolved crisis counts as worsened
	Duelist string
	RelicZ  string // zone key of the relic's dungeon level
	relicOK bool
	rep     *strings.Replacer
}

var rsDefCache = map[string]*rsDef{}

func (d *rsDef) fill(s string) string { return d.rep.Replace(s) }

func rsLower(s string) string {
	if s == "" {
		return s
	}
	return strings.ToLower(s[:1]) + s[1:]
}

func rsUpper(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func (g *Game) rsDef(id int) *rsDef {
	w := g.W
	if w == nil || id <= 0 {
		return nil
	}
	st := w.stateByID[id]
	if st == nil || !st.Alive {
		return nil
	}
	key := fmt.Sprint(w.Seed, ":", id)
	if d, ok := rsDefCache[key]; ok {
		return d
	}
	d := g.rsBuild(st)
	rsDefCache[key] = d
	return d
}

func (g *Game) rsCapital(st *StateData) *BurgData {
	w := g.W
	if b := w.burgByID[st.Capital]; b != nil && b.Alive {
		return b
	}
	var best *BurgData
	for i := range w.Burgs {
		b := &w.Burgs[i]
		if b.State == st.ID && b.Alive && (best == nil || b.Pop > best.Pop) {
			best = b
		}
	}
	return best
}

func (g *Game) rsBuild(st *StateData) *rsDef {
	w := g.W
	cp := g.rsCapital(st)
	if cp == nil {
		return nil
	}
	ar := rsPickArch(w, st)
	d := &rsDef{A: ar, St: st, Cap: cp, Third: ar.Third}
	h := HashStr(fmt.Sprint(w.Seed, ":rs:time:", st.ID))
	d.T1 = 12 + int(h%14)
	d.T2 = d.T1 + 14 + int((h>>8)%14)
	d.TW = d.T2 + 24 + int((h>>16)%12)
	cu := w.cultByID[st.Culture]
	r := NewRng(fmt.Sprint(w.Seed, ":rs:names:", st.ID))
	ruler := st.Ruler
	if ruler == "" {
		ruler = "the ruler"
	}
	house := st.Name
	if f := strings.Fields(st.Ruler); len(f) > 1 {
		house = f[len(f)-1]
	} else if cu != nil && len(cu.Houses) > 0 {
		house = Pick(r, cu.Houses)
	}
	var pool []*PersonData
	for i := range w.People {
		p := &w.People[i]
		if p.Alive && p.State == st.ID && p.Name != st.Ruler && p.Name != "" {
			pool = append(pool, p)
		}
	}
	sort.Slice(pool, func(i, j int) bool { return pool[i].ID < pool[j].ID })
	pickName := func(skip string) string {
		if len(pool) > 0 {
			for k := 0; k < 8; k++ {
				p := pool[r.Int(0, len(pool)-1)]
				if p.Name != skip {
					return p.Name
				}
			}
		}
		for k := 0; k < 8; k++ {
			n := g.cultureName(cu, r)
			if n != skip {
				return n
			}
		}
		return "a rival"
	}
	a := pickName("")
	b := pickName(a)
	faith := "the temple"
	if rl := w.relByID[st.Religion]; rl != nil && rl.Name != "" {
		faith = rl.Name
	}
	cul := "the people"
	if cu != nil {
		cul = cu.Name
	}
	foe := "a neighbouring realm"
	for _, id := range st.Wars {
		if o := w.stateByID[id]; o != nil && o.Alive {
			foe = o.Name
			break
		}
	}
	if foe == "a neighbouring realm" {
		low := 1 << 30
		ids := make([]int, 0, len(st.Relations))
		for id := range st.Relations {
			ids = append(ids, id)
		}
		sort.Ints(ids)
		for _, id := range ids {
			if o := w.stateByID[id]; o != nil && o.Alive && o.ID != st.ID && st.Relations[id] < low {
				low, foe = st.Relations[id], o.Name
			}
		}
	}
	d.Duelist = b
	d.rep = strings.NewReplacer("{realm}", st.Name, "{ruler}", ruler, "{capital}", cp.Name, "{faith}", faith, "{culture}", cul,
		"{a}", a, "{b}", b, "{foe}", foe, "{house}", house)
	return d
}

func rsPickArch(w *WorldData, st *StateData) *rsArch {
	wt := map[string]int{}
	for _, a := range rsArchs {
		wt[a.ID] = 2
	}
	add := func(id string, n int) { wt[id] += n }
	if len(st.Wars) > 0 {
		add("warweary", 5)
		add("usurper", 2)
		add("debt", 2)
		add("rebel", 1)
	}
	if st.Famine != nil {
		add("famine", 4)
		if st.Famine.Choice == "hoard" || st.Famine.Choice == "requisition" {
			add("famine", 2)
		}
		add("unrest", 1)
	}
	for _, p := range st.Persecutions {
		if p.Ended == 0 {
			if p.Axis == "faith" {
				add("heresy", 3)
			} else {
				add("rebel", 3)
			}
		}
	}
	if st.Reforms != nil && len(st.Reforms.Movements) > 0 {
		add("unrest", 4)
		add("guild", 1)
	}
	if so := st.SocialOrder; so != nil {
		if so.Movement {
			add("unrest", 3)
		}
		if so.Pressure >= 50 {
			add("unrest", 2)
		}
	}
	for _, f := range st.Forces {
		switch f.Kind {
		case "strikes":
			add("unrest", 3)
		case "megacorps":
			add("guild", 3)
		case "wareconomy", "armsrace":
			add("warweary", 2)
		case "leylines":
			add("cult", 2)
		case "fallout", "smog":
			add("debt", 1)
		}
	}
	if rp := w.swPerson(st.RulerID); rp != nil {
		if !rp.Alive {
			add("succession", 4)
			add("missingheir", 2)
		} else if rp.Born > 0 && w.Year-rp.Born >= 60 {
			add("succession", 3)
		}
	} else {
		add("succession", 1)
	}
	if rl := w.relByID[st.Religion]; rl != nil {
		t := strings.ToLower(rl.Type)
		if strings.Contains(t, "esoter") || strings.Contains(t, "occult") || strings.Contains(t, "myst") {
			add("cult", 2)
			add("heresy", 1)
		}
		if strings.Contains(t, "orthod") || strings.Contains(t, "dogma") || strings.Contains(t, "church") {
			add("heresy", 2)
		}
	}
	if cp := w.burgByID[st.Capital]; cp != nil && (cp.Charter || cp.League != "") {
		add("guild", 3)
	}
	if w.Hazards != nil {
		hit := false
		for _, o := range w.Hazards.Outbreaks {
			for i := range w.Burgs {
				if b := &w.Burgs[i]; b.State == st.ID && b.Alive {
					for _, p := range o.Parasangs {
						if p == b.P {
							hit = true
						}
					}
				}
			}
		}
		if hit {
			add("debt", 2)
			add("unrest", 2)
		}
	}
	murdered := 0
	for i := range w.People {
		p := &w.People[i]
		if !p.Alive && p.State == st.ID {
			c := strings.ToLower(p.Cause)
			if strings.Contains(c, "murder") || strings.Contains(c, "poison") || strings.Contains(c, "assassin") || strings.Contains(c, "slain") {
				murdered++
			}
		}
	}
	if murdered >= 2 {
		add("haunted", 3)
		add("usurper", 1)
	}
	for _, e := range w.Events {
		if e.St != st.ID {
			continue
		}
		t := strings.ToLower(e.Kind + " " + e.Text)
		for _, kv := range [][2]string{{"coup", "usurper"}, {"revolt", "rebel"}, {"rebel", "rebel"}, {"famine", "famine"}, {"plague", "unrest"}, {"succession", "succession"},
			{"heir", "missingheir"}, {"debt", "debt"}, {"bankrupt", "debt"}, {"heresy", "heresy"}, {"schism", "heresy"}, {"cult", "cult"}, {"strike", "unrest"}, {"curse", "haunted"}} {
			if strings.Contains(t, kv[0]) {
				add(kv[1], 1)
			}
		}
	}
	r := NewRng(fmt.Sprint(w.Seed, ":rs:pick:", st.ID))
	tot := 0
	for _, a := range rsArchs {
		tot += wt[a.ID]
	}
	n := r.Int(0, tot-1)
	for i := range rsArchs {
		n -= wt[rsArchs[i].ID]
		if n < 0 {
			return &rsArchs[i]
		}
	}
	return &rsArchs[0]
}

// ---------------------------------------------------------------- stage logic

// rsEff is the day the crisis has really reached, with the player's pushes.
func (g *Game) rsEff(rr *rsRealm) int {
	e := g.Day() + rr.Push
	if rr.Res != "" && rr.ResDay > 0 {
		e = rr.ResDay + rr.Push
	}
	return e
}

func (g *Game) rsStage(d *rsDef, rr *rsRealm) int {
	e := g.rsEff(rr)
	switch {
	case e >= d.T2:
		return 2
	case e >= d.T1:
		return 1
	}
	return 0
}

func (g *Game) rsWorse(d *rsDef, rr *rsRealm) bool {
	return rr.Res == "" && g.rsEff(rr) >= d.TW
}

func rsStageName(s int) string {
	return []string{"a whisper", "open trouble", "the breaking point"}[clampi(s, 0, 2)]
}

func rsResIdx(k string) int { return strings.Index("abxt", k) }

func (d *rsDef) tag(rr *rsRealm) string {
	i := rsResIdx(rr.Res)
	if i < 0 {
		return ""
	}
	return d.fill(d.A.Tag[i])
}

func (g *Game) rsLog(rr *rsRealm, s string) {
	rr.Did = append(rr.Did, fmt.Sprintf("Day %d: %s", g.Day()+1, s))
	if len(rr.Did) > 10 {
		rr.Did = rr.Did[len(rr.Did)-10:]
	}
}

func (g *Game) rsNews(d *rsDef, format string, a ...any) {
	lvSt(g).addNews(lvDay(g), d.Cap.ID, "%s", fmt.Sprintf(format, a...))
}

// rsHere is the realm id of the place you stand in (0 = none).
func (g *Game) rsHere() int {
	if g.W == nil || g.Cur == nil || g.OnWorld {
		return 0
	}
	if g.Cur.Burg > 0 {
		if b := g.W.burgByID[g.Cur.Burg]; b != nil {
			return b.State
		}
	}
	if g.W.In(g.Cur.WX, g.Cur.WY) {
		if p := g.W.Idx(g.Cur.WX, g.Cur.WY); p >= 0 && p < len(g.W.State) {
			return g.W.State[p]
		}
	}
	return 0
}

func (g *Game) rsNoticeText(d *rsDef, rr *rsRealm) string {
	if rr.Res != "" {
		return "NOTICE: " + rsUpper(d.tag(rr)) + ". The court of " + d.St.Name + " asks all to keep the peace."
	}
	s := d.fill(d.A.Notice[g.rsStage(d, rr)])
	if g.rsWorse(d, rr) {
		s += " The trouble has been left to fester."
	}
	return s
}

// ---------------------------------------------------------------- price effect

// rsPriceMod is a small buying-price factor for the realm you trade in: trouble raises prices with its
// stage and when it is ignored, a settled crisis eases them a little. Returns 1 when nothing applies.
func rsPriceMod(g *Game) float64 {
	if g == nil || g.W == nil {
		return 1
	}
	id := g.rsHere()
	d := g.rsDef(id)
	if d == nil {
		return 1
	}
	rr := g.rsRealmOf(id)
	if rr.Res != "" {
		switch rr.Res {
		case "a":
			return 0.98
		case "b":
			return 1.02
		case "x":
			return 0.96
		}
		return 0.97
	}
	n := g.rsStage(d, rr) * d.A.Price
	if g.rsWorse(d, rr) {
		n += d.A.Price*2 + rr.Ignored*2
	}
	n += rr.Profits * 2
	if n > 25 {
		n = 25
	}
	return 1 + float64(n)/100
}

// ---------------------------------------------------------------- barks

// rsLine is a bark about the realm's crisis for the actor (barks.go can call it): the stage's talk, a worse
// line when ignored, or the aftermath once settled. "" when the actor's realm has none.
func rsLine(g *Game, a *Actor) string {
	if g == nil || g.W == nil || a == nil {
		return ""
	}
	id := a.State
	if id <= 0 {
		id = g.rsHere()
	}
	d := g.rsDef(id)
	if d == nil {
		return ""
	}
	rr := g.rsRealmOf(id)
	h := HashStr(fmt.Sprint(g.Seed, ":rs:bark:", id, ":", a.ID, ":", g.Turn/120))
	if rr.Res != "" {
		tag := d.tag(rr)
		return []string{"They still talk of how " + tag + ".", "Since " + tag + ", the market is calmer.", "Say what you like, " + tag + "."}[h%3]
	}
	if g.rsWorse(d, rr) && h%2 == 0 {
		return rsWorseLines[h/2%uint64(len(rsWorseLines))]
	}
	return d.fill(d.A.Say[g.rsStage(d, rr)])
}

// ---------------------------------------------------------------- generation hooks

func rsZone(g *Game, zn *Zone, _ *Rng) {
	w := g.W
	if w == nil || zn == nil {
		return
	}
	if zn.Z == 0 && zn.Kind == "town" && zn.Burg > 0 {
		bd := w.burgByID[zn.Burg]
		if bd == nil || !bd.Alive {
			return
		}
		d := g.rsDef(bd.State)
		if d == nil {
			return
		}
		rr := g.rsRealmOf(bd.State)
		r := NewRng(w.Seed + ":rs:zone:" + zn.Key)
		if zn.ZX == 1 && zn.ZY == 1 {
			x, y, ok := zn.FreeSpot(r, swNear(40, 12, 16, 8))
			if ok {
				zn.Set(x, y, TInscription)
				zn.Notes[y*MW+x] = g.rsNoticeText(d, rr)
				g.rs().Notices[zn.Key] = y*MW + x
			}
			if bd.ID == d.Cap.ID {
				g.rsPlaceCapital(zn, d, bd, r)
			}
		}
		return
	}
	if zn.Z == 1 && w.In(zn.WX, zn.WY) {
		p := w.Idx(zn.WX, zn.WY)
		if p < 0 || p >= len(w.State) {
			return
		}
		d := g.rsDef(w.State[p])
		if d == nil || d.Third != "relic" {
			return
		}
		g.rsRelicSite(d)
		if d.relicOK && d.RelicZ == zn.Key {
			r := NewRng(w.Seed + ":rs:relic:" + zn.Key)
			if x, y, ok := zn.FreeSpot(r, nil); ok {
				it := g.rsItem(d, "relic")
				it.X, it.Y = x, y
				zn.Items = append(zn.Items, it)
			}
		}
	}
}

// rsRelicSite picks, once, the nearest dungeon level to the capital that can hold the relic.
func (g *Game) rsRelicSite(d *rsDef) {
	if d.RelicZ != "" || d.relicOK {
		return
	}
	sites := g.jbSites(d.Cap)
	if len(sites) == 0 {
		d.Third = "card"
		d.relicOK = false
		d.RelicZ = "-"
		return
	}
	bx, by := d.Cap.P%g.W.W, d.Cap.P/g.W.W
	best, bd := 0, 1<<30
	for i, s := range sites {
		dist := abs(s.WX-bx) + abs(s.WY-by)
		if dist < bd {
			best, bd = i, dist
		}
	}
	s := sites[best]
	d.RelicZ = ZoneKey(s.WX, s.WY, s.ZX, s.ZY, 1)
	d.relicOK = true
}

func (g *Game) rsItem(d *rsDef, kind string) *Item {
	id := d.St.ID
	if kind == "relic" {
		return &Item{ID: g.newID(), Kind: "trophy", Name: d.A.Items[1], Glyph: '&', F: CYellow, Weight: 2, Value: 6, Ident: true,
			Effect: "rs:relic", Extra: map[string]int{"rs": id},
			Lore: d.fill("It bears on the trouble in {realm}: " + rsLower(d.A.Brief) + " Reading it will show you what to do.")}
	}
	return &Item{ID: g.newID(), Kind: "trophy", Name: d.A.Items[0], Glyph: '?', F: CWhite, Weight: 1, Value: 3, Ident: true,
		Effect: "rs:doc", Extra: map[string]int{"rs": id},
		Lore: d.fill(d.A.Doc)}
}

var rsInfTitles = map[string]string{"succession": "court copyist", "unrest": "street speaker", "heresy": "temple clerk", "cult": "lodge courier",
	"warweary": "quartermaster's clerk", "famine": "customs tally-man", "usurper": "general's former page", "guild": "guild bookkeeper",
	"haunted": "palace servant", "missingheir": "old household page", "debt": "counting-house clerk", "rebel": "border tax clerk"}

func (g *Game) rsPlaceCapital(zn *Zone, d *rsDef, bd *BurgData, r *Rng) {
	cu := g.W.cultByID[bd.Culture]
	inf := g.makeNPC("villager", bd, cu, r)
	inf.Title = rsInfTitles[d.A.ID]
	inf.Desc = fmt.Sprintf("A %s of %s, who looks like someone with something to say and nobody to say it to.", inf.Title, bd.Name)
	inf.Set("rsinf")
	inf.AI = "still"
	swPut(zn, r, inf, swNear(40, 12, 18, 9))
	if x, y, ok := zn.FreeSpot(r, swNear(40, 12, 18, 9)); ok {
		it := g.rsItem(d, "doc")
		it.X, it.Y = x, y
		zn.Items = append(zn.Items, it)
	}
	if d.Third == "duel" {
		duel := g.Spawn("bandit", maxi(3, zn.Level+2), r)
		duel.Name = d.Duelist
		duel.Proper = true
		duel.Title = "champion"
		duel.Role = "rs_duelist"
		duel.Faction = zn.Faction
		duel.AI = "still"
		duel.F = CRed
		duel.Angry = false
		duel.Desc = d.fill("A hard-faced fighter wearing the colours of {b}, who is looking for someone to challenge in the street.")
		duel.Set("rsduel")
		swPut(zn, r, duel, swNear(40, 12, 18, 9))
	}
}

func rsEnter(g *Game, zn *Zone) {
	if g.W == nil || zn == nil || zn.Kind != "town" || zn.Burg <= 0 {
		return
	}
	idx, ok := g.rs().Notices[zn.Key]
	if !ok {
		return
	}
	bd := g.W.burgByID[zn.Burg]
	if bd == nil {
		return
	}
	d := g.rsDef(bd.State)
	if d == nil {
		return
	}
	if zn.Notes != nil && idx >= 0 && idx < len(zn.T) && zn.T[idx] == TInscription {
		zn.Notes[idx] = g.rsNoticeText(d, g.rsRealmOf(bd.State))
	}
}

// ---------------------------------------------------------------- turn, kill

func rsTurn(g *Game) {
	if g.W == nil || g.P == nil || g.Cur == nil {
		return
	}
	s := g.rs()
	if g.Turn%360 == 0 {
		g.rsStageNews()
	}
	if g.Mode != "main" {
		return
	}
	if s.pend != nil {
		p := s.pend
		s.pend = nil
		g.rsOpen("choice", p.ID, p.Src, nil)
		return
	}
	// documents and relics: reading one opens the choice screen
	for _, it := range g.P.Inv {
		if it.Extra == nil || it.Extra["rs"] <= 0 || !strings.HasPrefix(it.Effect, "rs:") {
			continue
		}
		id := it.Extra["rs"]
		src := strings.TrimPrefix(it.Effect, "rs:")
		g.removeInv(it)
		d := g.rsDef(id)
		if d == nil {
			return
		}
		rr := g.rsRealmOf(id)
		if rr.Res != "" {
			g.Msg("You read " + it.Name + ". It is old news now: " + d.tag(rr) + ".")
			return
		}
		if rr.Known < 3 {
			rr.Known = 3
		}
		g.rsLog(rr, "Read "+it.Name+".")
		g.Msg("{{W|You read " + it.Name + ".}} " + d.fill(d.A.Facts[1]))
		g.rsOpen("choice", id, src, nil)
		return
	}
	// a duel under way: the champion yields when beaten, stands down when you are
	for _, a := range g.Cur.Actors {
		if a.HP <= 0 || !a.Has("rsduel") || !a.Angry {
			continue
		}
		id := g.rsHere()
		if id <= 0 {
			continue
		}
		switch {
		case a.HP*100 <= a.MaxHP*35:
			a.Angry, a.AI = false, "still"
			a.HP = maxi(a.HP, a.MaxHP/2)
			a.Flags["rsduel"] = false
			g.Msg("{{G|" + a.Name + " lowers their blade: 'Enough. You have the better of me.'}}")
			rr := g.rsRealmOf(id)
			rr.Known = maxi(rr.Known, 2)
			g.rsLog(rr, "Beat "+a.Name+" in a duel.")
			g.rs().pend = &rsPending{id, "duel"}
		case g.P.HP*100 <= g.P.MaxHP*30:
			a.Angry, a.AI = false, "still"
			g.Msg("{{y|" + a.Name + " steps back: 'That will do. Go and have your wounds seen to.'}}")
			rr := g.rsRealmOf(id)
			rr.Push += 4
			g.rsLog(rr, "Lost a duel to "+a.Name+".")
		}
	}
	// the choice card: the streets force a decision when the trouble peaks
	if g.Turn%60 == 17 && g.Cur.Kind == "town" && !g.rsHostileNear(12) {
		id := g.rsHere()
		d := g.rsDef(id)
		if d == nil || d.Third != "card" {
			return
		}
		rr := g.rsRealmOf(id)
		if rr.Res == "" && !rr.Card && g.rsStage(d, rr) >= 1 {
			rr.Card = true
			g.rsLog(rr, "The trouble in the streets forced a choice.")
			g.Msg("{{W|The trouble in " + d.St.Name + " reaches you in the street.}}")
			g.rsOpen("choice", id, "card", nil)
		}
	}
}

func (g *Game) rsHostileNear(n int) bool {
	for _, a := range g.Cur.Actors {
		if a != g.P && a.HP > 0 && g.Hostile(a) && abs(a.X-g.P.X) <= n && abs(a.Y-g.P.Y) <= n {
			return true
		}
	}
	return false
}

// rsStageNews posts the realm's stage changes to the town news.
func (g *Game) rsStageNews() {
	ids := make([]int, 0, len(g.W.States))
	for i := range g.W.States {
		ids = append(ids, g.W.States[i].ID)
	}
	sort.Ints(ids)
	for _, id := range ids {
		d := g.rsDef(id)
		if d == nil {
			continue
		}
		rr := g.rsRealmOf(id)
		if rr.Res != "" {
			continue
		}
		st := g.rsStage(d, rr)
		if st+1 > rr.Seen && st >= 1 {
			rr.Seen = st + 1
			if st == 1 {
				g.rsNews(d, "Trouble comes into the open in %s: %s.", d.St.Name, d.A.Noun)
			} else {
				g.rsNews(d, "%s in %s has reached its breaking point.", rsUpper(d.A.Noun), d.St.Name)
			}
		}
		if g.rsWorse(d, rr) && !rr.WorseN {
			rr.WorseN = true
			g.rsNews(d, "%s in %s has been left to fester and prices are rising.", rsUpper(d.A.Noun), d.St.Name)
		}
	}
}

func rsKill(g *Game, dd, src *Actor) {
	if dd == nil || !dd.Has("rsduel") || g.W == nil {
		return
	}
	id := g.rsHere()
	if id <= 0 {
		return
	}
	rr := g.rsRealmOf(id)
	rr.Known = maxi(rr.Known, 2)
	g.rsLog(rr, "Killed "+dd.Name+" in a duel.")
	g.rs().pend = &rsPending{id, "duel"}
}

// ---------------------------------------------------------------- talk

func rsTalk(g *Game, a *Actor) []Option {
	if g.W == nil || g.Cur == nil || g.P == nil || a == nil || a.HP <= 0 || a.Angry || a.AI == "hostile" || a.AI == "follower" {
		return nil
	}
	if g.Cur.Kind != "town" || g.Cur.Burg <= 0 {
		return nil
	}
	id := g.rsHere()
	d := g.rsDef(id)
	if d == nil {
		return nil
	}
	rr := g.rsRealmOf(id)
	var out []Option
	if a.Has("rsinf") {
		if !rr.Inf && rr.Res == "" {
			out = append(out, Option{"Ask the " + a.Title + " what they know", func() {
				rr.Inf = true
				rr.Known = maxi(rr.Known, 3)
				g.rsLog(rr, "Heard the "+a.Title+" in "+d.Cap.Name+".")
				g.talkMenu("\"" + d.fill(d.A.Info) + "\"\n\n{{K|(" + d.fill(d.A.Facts[0]) + " " + d.fill(d.A.Facts[1]) + ")}}")
			}})
		}
		if rr.Inf && rr.Res == "" {
			out = append(out, Option{"Decide what to do about the trouble", func() { g.rsOpen("choice", id, "informant", a) }})
		}
	}
	if a.Has("rsduel") && rr.Res == "" && !a.Angry {
		out = append(out, Option{"Accept " + a.Name + "'s challenge to a duel", func() {
			a.Angry, a.AI = true, "hostile"
			g.Msg("{{R|" + a.Name + " draws. 'Then show me.'}} Beat them and they will yield.")
			g.Mode = "main"
		}})
	}
	out = append(out, Option{"Ask about the troubles in " + d.St.Name, func() {
		n := 0
		known := false
		for _, v := range rr.Asked {
			if v == a.ID {
				known = true
			}
		}
		if !known {
			rr.Asked = append(rr.Asked, a.ID)
			if rr.Known < 2 {
				rr.Known++
			}
			n = rr.Known
		}
		_ = n
		line := rsLine(g, a)
		txt := "\"" + line + "\""
		if rr.Res == "" && rr.Known >= 1 {
			txt += "\n\n{{K|" + d.fill(d.A.Brief) + "}}"
		}
		if rr.Known >= 2 && rr.Res == "" {
			txt += "\n{{K|" + d.fill(d.A.Facts[0]) + "}}"
		}
		g.talkMenu(txt)
	}})
	return out
}

// ---------------------------------------------------------------- the choice

func (g *Game) rsOpen(mode string, id int, src string, talker *Actor) {
	s := g.rs()
	if mode == "view" && id <= 0 {
		id = g.rsHere()
		if g.rsDef(id) == nil {
			id = g.HomeState
		}
		if g.rsDef(id) == nil {
			if ids := g.rsRealms(); len(ids) > 0 {
				id = ids[0]
			}
		}
	}
	s.mode, s.id, s.src, s.talker, s.res = mode, id, src, talker, ""
	g.Mode = "rs"
}

// rsRealms lists the realms worth showing: the one here, home, and any you know something about.
func (g *Game) rsRealms() []int {
	set := map[int]bool{}
	if id := g.rsHere(); g.rsDef(id) != nil {
		set[id] = true
	}
	if g.rsDef(g.HomeState) != nil {
		set[g.HomeState] = true
	}
	for id, rr := range g.rs().Realms {
		if (rr.Known > 0 || rr.Res != "") && g.rsDef(id) != nil {
			set[id] = true
		}
	}
	out := make([]int, 0, len(set))
	for id := range set {
		out = append(out, id)
	}
	sort.Ints(out)
	return out
}

type rsOptT struct {
	Key, Label string
	Kind       string // check kind or ""
	Dc         int
	Locked     string
}

func (g *Game) rsOptions(d *rsDef, rr *rsRealm) []rsOptT {
	s := g.rsStage(d, rr)
	opts := []rsOptT{
		{"a", d.fill(d.A.Lab[0]), "", 0, ""},
		{"b", d.fill(d.A.Lab[1]), "", 0, ""},
		{"x", d.fill(d.A.Lab[2]), d.A.Chk[0], 9 + s, ""},
		{"t", d.fill(d.A.Lab[3]), d.A.Chk[1], 9 + s*2, ""},
		{"p", d.fill(d.A.Lab[4]), d.A.Chk[2], 8 + s, ""},
		{"i", "Walk away and let it run its course", "", 0, ""},
	}
	if rr.Known < 3 {
		opts[2].Locked = "needs proof"
	}
	if rr.PStage&(1<<uint(s)) != 0 {
		opts[4].Locked = "done this stage"
	}
	return opts
}

func rsKeys(g *Game, k string) {
	s := g.rs()
	close := func() {
		tk := s.talker
		s.talker = nil
		s.mode = ""
		if tk != nil && g.Talk != nil && tk.HP > 0 {
			g.Mode = "talk"
			g.talkMenu("\"Anything else?\"")
			return
		}
		g.Mode = "main"
	}
	switch s.mode {
	case "result":
		close()
		return
	case "choice":
		if k == "Escape" || k == "q" {
			close()
			return
		}
		d := g.rsDef(s.id)
		if d == nil || len(k) != 1 {
			return
		}
		rr := g.rsRealmOf(s.id)
		if rr.Res != "" {
			close()
			return
		}
		for _, o := range g.rsOptions(d, rr) {
			if o.Key == k || (k == "f" && o.Key == "i") || (k == "c" && o.Key == "x") || (k == "d" && o.Key == "t") || (k == "e" && o.Key == "p") {
				if o.Locked != "" {
					g.Msg("{{y|Not now: " + o.Locked + ".}}")
					return
				}
				s.res = g.rsDo(d, rr, o.Key)
				s.mode = "result"
				return
			}
		}
	default:
		switch k {
		case "Escape", "q":
			close()
		case "ArrowRight", "Tab", "n", "ArrowDown":
			g.rsCycle(1)
		case "ArrowLeft", "p", "ArrowUp":
			g.rsCycle(-1)
		}
	}
}

func (g *Game) rsCycle(dir int) {
	s := g.rs()
	ids := g.rsRealms()
	if len(ids) == 0 {
		return
	}
	i := 0
	for j, id := range ids {
		if id == s.id {
			i = j
		}
	}
	s.id = ids[(i+dir+len(ids))%len(ids)]
}

// rsRoll rolls a named check and returns whether it passed.
func (g *Game) rsRoll(kind string, dc int) bool {
	if kind == "" {
		return true
	}
	return g.roll2d6Check(g.checkMod(kind), dc)
}

func (g *Game) rsResolve(d *rsDef, rr *rsRealm, key, did string) {
	rr.Res = key
	rr.ResDay = g.Day()
	g.rsLog(rr, did)
	g.rsNews(d, "%s in %s is over: %s.", rsUpper(d.A.Noun), d.St.Name, d.tag(rr))
	g.deed("rs", fmt.Sprintf("%s settled %s in %s: %s.", g.P.Name, d.A.Noun, d.St.Name, d.tag(rr)), 0, 0)
}

func (g *Game) rsDo(d *rsDef, rr *rsRealm, key string) string {
	fac := fmt.Sprintf("state:%d", d.St.ID)
	lvl := maxi(1, g.P.Level)
	s := g.rsStage(d, rr)
	h := HashStr(fmt.Sprint(g.Seed, ":rs:txt:", d.St.ID, ":", g.Day()))
	switch key {
	case "a":
		g.rsResolve(d, rr, "a", "Chose to "+rsLower(d.fill(d.A.Lab[0]))+".")
		g.AddRep(fac, 10, " for standing with the order")
		g.gainXP(25 + lvl*2)
		return d.fill(d.A.Out[0])
	case "b":
		g.rsResolve(d, rr, "b", "Sided against the order")
		g.AddRep(fac, -10, " for backing its challengers")
		pay := g.cuEarn(60 + lvl*15)
		g.gainXP(25 + lvl*2)
		return d.fill(d.A.Out[1]) + " {{K|Their backers pay you " + pay + ".}}"
	case "x":
		if g.rsRoll(d.A.Chk[0], 9+s) {
			g.rsResolve(d, rr, "x", "Exposed the truth")
			g.AddRep(fac, 16, " for exposing the truth")
			pay := g.cuEarn(40 + lvl*10)
			g.gainXP(45 + lvl*3)
			return d.fill(d.A.Out[2]) + " {{K|The court pays you " + pay + ".}}"
		}
		rr.Push += 8
		g.rsLog(rr, "A failed exposure made it worse.")
		g.AddRep(fac, -4, " for a failed accusation")
		return "You make your case and it does not hold. The accused are warned and the trouble moves faster."
	case "t":
		if g.rsRoll(d.A.Chk[1], 9+s*2) {
			g.rsResolve(d, rr, "t", "Brokered a settlement")
			g.AddRep(fac, 10, " for brokering peace")
			pay := g.cuEarn(30 + lvl*10)
			g.gainXP(40 + lvl*3)
			return d.fill(d.A.Out[3]) + " {{K|Both sides send you " + pay + ".}}"
		}
		rr.Push += 6
		g.rsLog(rr, "A failed parley hardened both sides.")
		g.AddRep(fac, -3, " for a failed parley")
		return "Neither side will sit with the other. Your parley breaks up, and tempers are worse for it."
	case "p":
		rr.PStage |= 1 << uint(s)
		if g.rsRoll(d.A.Chk[2], 8+s) {
			rr.Profits++
			rr.Push += 10
			g.rsLog(rr, "Profited from the trouble.")
			pay := g.cuEarn(70 + lvl*25 + s*30)
			g.AddRep(fac, -6, " for profiting from the trouble")
			return rsProfitLines[h%uint64(len(rsProfitLines))] + " {{K|You take " + pay + ".}}"
		}
		g.AddRep(fac, -4, " for dealing on both sides")
		g.rsLog(rr, "Tried to profit and was noticed.")
		return "A buyer talks. You are seen dealing on both sides and leave with less than you came with."
	}
	rr.Ignored++
	rr.Push += 12
	g.rsLog(rr, "Walked away.")
	return rsIgnoreLines[h%uint64(len(rsIgnoreLines))] + " {{K|It will move faster now, and prices will follow.}}"
}

// ---------------------------------------------------------------- screen

func (g *Game) rsDraw() {
	sc := g.sc
	s := g.rs()
	d := g.rsDef(s.id)
	title := "Realm affairs"
	if s.mode == "choice" || s.mode == "result" {
		title = "A crisis in " + nonEmpty(func() string {
			if d != nil {
				return d.St.Name
			}
			return ""
		}(), "the realm")
	}
	x, y, w, h := g.panel(title)
	bottom := 2 + 1 + h - 1
	put := func(txt string) {
		for _, ln := range Wrap(txt, w) {
			if y < bottom {
				sc.TextB(x, y, ln, CGray, CBgDark, x+w)
				y++
			}
		}
	}
	if d == nil {
		put("{{K|No realm here has a crisis you can read. Visit a town of a living realm.}}")
		sc.TextB(x, bottom, "{{K|Esc close}}", CGray, CBgDark, x+w)
		return
	}
	rr := g.rsRealmOf(s.id)
	stage := g.rsStage(d, rr)
	if s.mode == "result" {
		put("{{W|" + d.fill(d.A.Name) + "}}")
		y++
		put(s.res)
		y++
		if rr.Res != "" {
			put("{{G|Aftermath: " + d.tag(rr) + ".}}")
		}
		sc.TextB(x, bottom, "{{K|Press any key}}", CGray, CBgDark, x+w)
		return
	}
	if s.mode == "choice" {
		put("{{W|" + d.fill(d.A.Name) + "}}  {{K|" + d.A.Noun + ", stage " + fmt.Sprint(stage+1) + " of 3: " + rsStageName(stage) + "}}")
		y++
		put(d.fill(d.A.Brief))
		srcTxt := map[string]string{"informant": "Your informant has told you what they know.", "doc": "The document is in your hands.", "relic": "What you found in the dark tells the story.",
			"duel": "The champion has yielded and the crowd waits for your word.", "card": "The trouble has reached you and will not wait."}[s.src]
		if srcTxt != "" {
			put("{{K|" + srcTxt + "}}")
		}
		y++
		if rr.Res != "" {
			put("{{G|It is settled: " + d.tag(rr) + ".}}")
		} else {
			put("What will you do?")
			y++
			for _, o := range g.rsOptions(d, rr) {
				tail := ""
				if o.Kind != "" {
					tail = fmt.Sprintf(" {{c|[%s %d%%]}}", checkKinds[o.Kind].label, int(chanceFor(g.checkMod(o.Kind), o.Dc)*100+0.5))
				}
				if o.Locked != "" {
					tail = " {{K|(" + o.Locked + ")}}"
				}
				put(fmt.Sprintf("{{W|%s}}) %s%s", o.Key, o.Label, tail))
			}
		}
		sc.TextB(x, bottom, "{{K|Letter choose  Esc leave it for now}}", CGray, CBgDark, x+w)
		return
	}
	// overview
	put("{{W|" + d.St.FullName + "}}  {{K|(left/right: other realms)}}")
	y++
	put("{{Y|" + d.fill(d.A.Name) + "}} ({{K|" + d.A.Noun + "}})")
	if rr.Res != "" {
		put("{{G|Settled: " + d.tag(rr) + ".}}")
	} else {
		pips := strings.Repeat("#", stage+1) + strings.Repeat("-", 2-stage)
		put(fmt.Sprintf("Stage [%s] %s", pips, rsStageName(stage)))
		if g.rsWorse(d, rr) {
			put("{{R|It has been left to fester.}}")
		}
	}
	y++
	put("{{W|What you know}}")
	switch {
	case rr.Known <= 0 && rr.Res == "":
		put("{{K|Nothing yet. Ask townsfolk about the troubles.}}")
	default:
		put(d.fill(d.A.Brief))
		if rr.Known >= 2 {
			put(d.fill(d.A.Facts[0]))
		}
		if rr.Known >= 3 {
			put(d.fill(d.A.Facts[1]))
		}
	}
	if cr := g.poCrisisAt(s.id); cr != nil {
		put("{{R|The court is also in " + poCrisisWord[cr.Kind] + ".}}")
	}
	if rr.Known >= 1 && rr.Res == "" {
		y++
		put("{{W|Ways in}}")
		put(d.fill("An informant is said to be in {capital}, and a paper was dropped in its streets."))
		switch d.Third {
		case "duel":
			put(d.fill("A champion of {b} in {capital} is looking for someone to fight."))
		case "relic":
			put(d.fill("Something that bears on this lies in a dungeon near {capital}."))
		default:
			put("Sooner or later the trouble will reach you in the street.")
		}
	}
	y++
	pm := int(rsPriceMod(g)*100+0.5) - 100
	if g.rsDef(g.rsHere()) != nil && g.rsHere() == s.id {
		put(fmt.Sprintf("{{W|Prices here}} %+d%%", pm))
	}
	if len(rr.Did) > 0 {
		y++
		put("{{W|What you did}}")
		for _, ln := range rr.Did {
			put("{{K|" + ln + "}}")
		}
	}
	sc.TextB(x, bottom, "{{K|Left/right other realms  Esc close}}", CGray, CBgDark, x+w)
}
