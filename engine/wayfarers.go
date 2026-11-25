package engine

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// Wayfarers (prefix wf): rare encounters on the open road so travel stays surprising. Templates live in
// wayfarers_data.go. When you walk into an untouched wild surface zone, a hash of (seed, zone, day) decides
// whether someone is waiting there (about one zone in thirteen, never twice within two game days, never with
// hostiles in sight). The cast is real actors placed near you with a side rng (people, objects, beasts), flagged
// "wfx"; walking within three tiles of them, or talking to the host, opens a lettered choice screen (mode "wf").
// Choices reuse the crossroads machinery (effect strings, requirements, checks, karma, placeholders from the real
// world via crBuild) plus extra tokens: coin:+n|-n (units of local money, scaled by level through currency.go),
// hostile, foe:<creature>:<n>, clue, tip, rumour:<text>, map, job. Jobs (an escort, a parcel, a message, a ghost's
// last words) are mirrored into g.Quests as Kind "wf" and finish by themselves when you enter the destination
// town. Text uses the encountered region's culture pack voice when there is one.
//
// Nothing in the generation draws g.R (side rngs only); g.R only rolls the checks. State g.ext("wf"), planet-local.
// What already exists and is not duplicated: es_wild.go (static hermit cabins, shrines, camps, trail signs),
// worldsim.go (pilgrims, caravans, minstrels, mercenaries and slavers walking the roads), scenes.go (refugee and
// pilgrim columns, beast fights), caravans.go (trade caravans), crossroads.go (dilemma cards in towns and at
// junctions), nemesis.go and rival.go (hunters). These are one-off living scenes you can talk into a choice.

const wfRate = 75 // chance per thousand per qualifying zone entry

func init() {
	unLocalKeys = append(unLocalKeys, "wf")
	enterHooks = append(enterHooks, wfEnter)
	turnHooks = append(turnHooks, wfTurn)
	talkHooks = append(talkHooks, wfTalk)
	modeKeys["wf"] = wfKeys
	modeDraws["wf"] = func(g *Game) { g.wfDraw() }
}

type wfCast struct {
	Name  string // "" = a name from the local culture; {E} style placeholders allowed
	Title string
	Glyph rune
	F     byte
	Tpl   string // "" = a person; else a creature template id
	N     int
	Obj   bool // a thing, not a person (a stone, a cart)
	Alt   string
	Host  bool // the one who speaks for the scene
}

type wfTpl struct {
	ID       string
	Title    string
	Need     string // "" "person" (a dead notable) "war"
	Cast     []wfCast
	Text     string
	Ch       []crCh
	Job      string // escort parcel message grave
	JobTitle string
	JobText  string
	JobEff   string
	JobDays  int
	Prep     func(g *Game, c map[string]string, r *Rng) bool
	Weight   int
}

type wfOpen struct {
	TID   string            `json:"t"`
	Ctx   map[string]string `json:"c"`
	Day   int               `json:"d"`
	Asked bool              `json:"a,omitempty"`
	Done  bool              `json:"x,omitempty"`
	Zone  string            `json:"z"`
}

type wfJob struct {
	QID      int               `json:"q"`
	Kind     string            `json:"k"`
	Title    string            `json:"ti"`
	Dest     int               `json:"de"`
	DestName string            `json:"dn"`
	Due      int               `json:"du"`
	Eff      string            `json:"e"`
	Ctx      map[string]string `json:"c"`
	Giver    string            `json:"g"`
	Ambush   int               `json:"am,omitempty"`
}

type wfCur struct {
	TID string `json:"t"`
	Z   string `json:"z"`
}

type wfState struct {
	Last int                `json:"l"`
	Seen map[string]int     `json:"s"`
	Open map[string]*wfOpen `json:"o"`
	Jobs []*wfJob           `json:"j,omitempty"`
	Cur  *wfCur             `json:"cu,omitempty"`
	N    int                `json:"n"`
	Kept int                `json:"kp,omitempty"`

	phase int
	res   []string
}

func wfSt(g *Game) *wfState {
	s := g.ext("wf", func() any { return &wfState{Seen: map[string]int{}, Open: map[string]*wfOpen{}} }).(*wfState)
	if s.Seen == nil {
		s.Seen = map[string]int{}
	}
	if s.Open == nil {
		s.Open = map[string]*wfOpen{}
	}
	return s
}

func wfTplByID(id string) *wfTpl {
	for i := range wfTpls {
		if wfTpls[i].ID == id {
			return &wfTpls[i]
		}
	}
	return nil
}

func wfCheb(a *Actor, b *Actor) int { return maxi(abs(a.X-b.X), abs(a.Y-b.Y)) }

// ---------------------------------------------------------------- context

// wfNearTown: the nearest living surface town at least two parasangs away (else the nearest at all).
func (g *Game) wfNearTown(wx, wy int) *BurgData {
	var best, any *BurgData
	bd, ad := 1<<30, 1<<30
	for i := range g.W.Burgs {
		b := &g.W.Burgs[i]
		if !b.Alive || b.Layer != "surface" {
			continue
		}
		d := abs(b.P%g.W.W-wx) + abs(b.P/g.W.W-wy)
		if d < ad || d == ad && b.ID < any.ID {
			ad, any = d, b
		}
		if d >= 2 && (d < bd || d == bd && b.ID < best.ID) {
			bd, best = d, b
		}
	}
	if best == nil {
		return any
	}
	return best
}

// wfVoice: greeting, proverb and culture name from the local culture pack (empty without one).
func (g *Game) wfVoice(key string) (gr, pv, cu string) {
	c := g.localCulture()
	if c == nil {
		return "Well met.", "The road teaches what the house cannot.", "the road"
	}
	cu = c.Name
	ci := g.W.etCultIdx(c)
	gr, pv = "Well met.", "The road teaches what the house cannot."
	if pk := g.W.cpOf(ci); pk != nil && pk.Voice != nil {
		if s := g.W.cpChoose(pk.Voice.Greet, "wf", key); s != "" {
			gr = s
		}
		if s := g.W.cpChoose(pk.Voice.Proverbs, "wf", "pv", key); s != "" {
			pv = s
		}
	}
	return
}

func (g *Game) wfCtx(zn *Zone, r *Rng) map[string]string {
	c := g.crBuild(crWhere{zn.WX, zn.WY, 0, zn.Key}, r)
	if b := g.wfNearTown(zn.WX, zn.WY); b != nil {
		c["Dest"], c["DestID"] = b.Name, strconv.Itoa(b.ID)
		c["DestP"] = strconv.Itoa(b.P)
		if s := g.W.stateByID[b.State]; s != nil {
			c["DestR"] = s.Name
		}
	} else {
		c["Dest"], c["DestID"] = "the next town", "0"
	}
	if c["DestR"] == "" {
		c["DestR"] = "the local lord"
	}
	c["Gr"], c["Pv"], c["Cu"] = g.wfVoice(zn.Key + ":" + fmt.Sprint(g.Day()))
	c["Day"] = fmt.Sprint(g.Day())
	return c
}

// ---------------------------------------------------------------- spawning

func (g *Game) wfMake(zn *Zone, cs wfCast, name string, r *Rng) *Actor {
	lvl := maxi(1, g.zoneLevel(zn.WX, zn.WY, 0))
	var a *Actor
	if cs.Tpl != "" && !cs.Obj {
		id := cs.Tpl
		if tplByID[id] == nil {
			id = "boar"
		}
		a = g.Spawn(id, lvl, r)
		a.Faction, a.AI, a.Angry = "wf", "still", false
		if cs.Alt != "" {
			a.AI = cs.Alt
		}
		if name != "" {
			a.Name = name
		}
		if strings.EqualFold(cs.Title, "stag") {
			a.Set("wfstag")
		}
		if cs.Glyph != 0 {
			a.Glyph, a.F = cs.Glyph, cs.F
		}
		a.Set("wfx")
		return a
	}
	plan, sp := "humanoid", "human"
	if cu := g.localCulture(); cu != nil {
		if s := g.speciesOf(cu); s != nil {
			plan, sp = BodyPlanForSpecies(s.ID), s.ID
		}
	}
	a = &Actor{ID: g.newID(), Name: name, Proper: !cs.Obj, Glyph: cs.Glyph, F: cs.F, Tpl: "npc", Faction: "wf", Level: lvl,
		Attr: [6]int{12, 12, 12, 12, 12, 12}, Speed: 100, AI: "still", Role: "wayfarer", Species: sp, Body: MakeBody(plan), Title: cs.Title}
	if cs.Obj {
		a.Species, a.Body = "", MakeBody("humanoid")
		a.Set("wfobj")
		a.MaxHP = 400
	} else {
		a.MaxHP = 12 + lvl*3
	}
	a.HP = a.MaxHP
	a.Set("wfx")
	a.Desc = cs.Title
	return a
}

// wfPlace puts the cast near the player (or near the first member), returns the placed actors.
func (g *Game) wfPlace(zn *Zone, t *wfTpl, c map[string]string, r *Rng) []*Actor {
	var out []*Actor
	fx, fy := -1, -1
	for _, cs := range t.Cast {
		cs.Title = crFill(cs.Title, c)
		n := maxi(cs.N, 1)
		for i := 0; i < n; i++ {
			name := crFill(cs.Name, c)
			if name == "" && !cs.Obj && cs.Tpl == "" {
				name = g.cultureName(g.localCulture(), r)
			}
			a := g.wfMake(zn, cs, name, r)
			near := func(x, y int) bool {
				if fx >= 0 {
					return maxi(abs(x-fx), abs(y-fy)) <= 2 && maxi(abs(x-g.P.X), abs(y-g.P.Y)) >= 3
				}
				d := maxi(abs(x-g.P.X), abs(y-g.P.Y))
				return d >= 5 && d <= 10
			}
			x, y, ok := zn.FreeSpot(r, near)
			if !ok && fx >= 0 {
				x, y, ok = zn.FreeSpot(r, func(x, y int) bool { return maxi(abs(x-fx), abs(y-fy)) <= 5 })
			}
			if !ok {
				x, y, ok = zn.FreeSpot(r, func(x, y int) bool { return maxi(abs(x-g.P.X), abs(y-g.P.Y)) >= 3 })
			}
			if !ok {
				continue
			}
			a.X, a.Y, a.HomeX, a.HomeY = x, y, x, y
			if cs.Host && i == 0 {
				a.Set("wfhost")
			}
			zn.Actors = append(zn.Actors, a)
			out = append(out, a)
			if fx < 0 {
				fx, fy = x, y
			}
			if c["E"] == "" && !cs.Obj && cs.Tpl == "" {
				c["E"], c["Et"] = a.Name, a.Title
			}
		}
	}
	if len(out) > 0 && c["E"] == "" {
		c["E"], c["Et"] = out[0].Name, out[0].Title
	}
	hosted := false
	for _, a := range out {
		if a.Has("wfhost") {
			hosted = true
		}
	}
	if !hosted && len(out) > 0 {
		out[0].Set("wfhost")
	}
	return out
}

func wfEnter(g *Game, zn *Zone) {
	if g.W == nil || g.P == nil || zn == nil || zn.Z != 0 || g.OnWorld || !g.W.In(zn.WX, zn.WY) {
		return
	}
	st := wfSt(g)
	day := g.Day()
	if zn.Kind == "town" && zn.Burg > 0 {
		g.wfJobsAt(zn.Burg)
	}
	if zn.Kind != "surface" || zn.Underwater || zn.Sky || zn.Dark {
		return
	}
	// stale scenes
	if o := st.Open[zn.Key]; o != nil {
		if o.Done || day-o.Day > 3 {
			g.wfStrip(zn)
			delete(st.Open, zn.Key)
		}
		return
	}
	g.wfStrip(zn)
	g.wfAmbush(zn)
	g.wfTrySpawn(zn)
}

func (g *Game) wfStrip(zn *Zone) {
	keep := zn.Actors[:0]
	for _, a := range zn.Actors {
		if a.Has("wfx") && a.HP > 0 && !a.Angry && a.AI != "hostile" {
			continue
		}
		keep = append(keep, a)
	}
	zn.Actors = keep
}

func (g *Game) wfTrySpawn(zn *Zone) {
	st := wfSt(g)
	day := g.Day()
	if st.Last > 0 && g.Turn-st.Last < 2*DayLen {
		return
	}
	if HashStr(fmt.Sprint(g.Seed, ":wf:roll:", zn.Key, ":", day))%1000 >= wfRate {
		return
	}
	if g.nearHostile() {
		return
	}
	var pool []*wfTpl
	for i := range wfTpls {
		t := &wfTpls[i]
		if d, ok := st.Seen[t.ID]; ok && day-d < 30 {
			continue
		}
		for w := 0; w < maxi(t.Weight, 1); w++ {
			pool = append(pool, t)
		}
	}
	if len(pool) == 0 {
		return
	}
	start := int(HashStr(fmt.Sprint(g.Seed, ":wf:pick:", zn.Key, ":", day)) % uint64(len(pool)))
	for k := 0; k < len(pool); k++ {
		t := pool[(start+k)%len(pool)]
		r := NewRng(fmt.Sprint(g.Seed, ":wf:spawn:", t.ID, ":", zn.Key, ":", day))
		c := g.wfCtx(zn, r)
		if t.Prep != nil && !t.Prep(g, c, r) {
			continue
		}
		actors := g.wfPlace(zn, t, c, r)
		if len(actors) == 0 {
			continue
		}
		st.Last = g.Turn
		st.N++
		st.Seen[t.ID] = day
		st.Open[zn.Key] = &wfOpen{TID: t.ID, Ctx: c, Day: day, Zone: zn.Key}
		if len(st.Open) > 40 {
			for k2, o := range st.Open {
				if o.Done || day-o.Day > 6 {
					delete(st.Open, k2)
				}
			}
		}
		head := actors[0]
		for _, a := range actors {
			if a.Has("wfhost") {
				head = a
			}
		}
		g.Msg("{{Y|You notice " + head.Name + nonEmptyWf(", "+head.Title, head.Title != "" && !strings.EqualFold(head.Title, head.Name)) + " ahead.}}")
		return
	}
}

func nonEmptyWf(s string, ok bool) string {
	if ok {
		return s
	}
	return ""
}

// ---------------------------------------------------------------- opening the scene

func wfTurn(g *Game) {
	if g.W == nil || g.Cur == nil || g.P == nil || g.OnWorld || g.Mode != "main" {
		return
	}
	st := wfSt(g)
	if g.Turn%60 == 0 {
		g.wfJobsTick()
	}
	o := st.Open[g.Cur.Key]
	if o == nil || o.Asked || o.Done {
		return
	}
	for _, a := range g.Cur.Actors {
		if a.Has("wfx") && a.HP > 0 && wfCheb(a, g.P) <= 3 {
			if g.nearHostile() {
				return
			}
			g.wfOpenScreen(o)
			return
		}
	}
}

func wfTalk(g *Game, a *Actor) []Option {
	if g == nil || g.Cur == nil || a == nil || !a.Has("wfx") || !a.Has("wfhost") || a.HP <= 0 {
		return nil
	}
	o := wfSt(g).Open[g.Cur.Key]
	if o == nil || o.Done {
		return nil
	}
	return []Option{{"{{W|Hear what they have to say.}}", func() { g.wfOpenScreen(o) }}}
}

func (g *Game) wfOpenScreen(o *wfOpen) {
	st := wfSt(g)
	if wfTplByID(o.TID) == nil {
		return
	}
	st.Cur = &wfCur{TID: o.TID, Z: o.Zone}
	st.phase, st.res = 0, nil
	g.Mode = "wf"
	g.Talk = nil
	g.Sel = 0
	o.Asked = true
}

func (g *Game) wfCurrent() (*wfTpl, *wfOpen) {
	st := wfSt(g)
	if st.Cur == nil {
		return nil, nil
	}
	o := st.Open[st.Cur.Z]
	t := wfTplByID(st.Cur.TID)
	if o == nil || t == nil {
		return nil, nil
	}
	return t, o
}

type wfOpt struct {
	label string
	ok    bool
	why   string
}

func (g *Game) wfOptions(t *wfTpl, o *wfOpen) []wfOpt {
	var out []wfOpt
	for i := range t.Ch {
		ch := &t.Ch[i]
		ok, why := g.crReqOK(ch.Req, o.Ctx)
		if strings.Contains(ch.Fl, "lie") && g.crVow("tr") != nil {
			ok, why = false, "vow of Truth"
		}
		lab := crFill(ch.T, o.Ctx)
		if ch.Chk != "" {
			f := strings.Split(ch.Chk, ":")
			dc, _ := strconv.Atoi(f[1])
			lab = g.checkLabel(f[0], dc) + " " + lab
		}
		out = append(out, wfOpt{lab, ok, why})
	}
	out = append(out, wfOpt{"{{K|Walk on.}}", true, ""})
	return out
}

func wfKeys(g *Game, k string) {
	st := wfSt(g)
	t, o := g.wfCurrent()
	if t == nil {
		st.Cur = nil
		g.Mode = "main"
		return
	}
	if st.phase == 1 {
		if k == "Enter" || k == " " || k == "Escape" {
			st.Cur, st.res = nil, nil
			g.Mode = "main"
		}
		return
	}
	opts := g.wfOptions(t, o)
	pick := func(i int) {
		if i < 0 || i >= len(opts) {
			return
		}
		if !opts[i].ok {
			g.Msg("{{K|You cannot choose that: " + opts[i].why + ".}}")
			return
		}
		g.wfChoose(t, o, i)
	}
	if i := letterIndex(k); i >= 0 {
		pick(i)
		return
	}
	switch k {
	case "Escape":
		st.Cur = nil
		g.Mode = "main"
	case "ArrowUp":
		g.Sel = maxi(0, g.Sel-1)
	case "ArrowDown":
		g.Sel = mini(len(opts)-1, g.Sel+1)
	case "Enter", " ":
		pick(g.Sel)
	}
}

func (g *Game) wfChoose(t *wfTpl, o *wfOpen, i int) {
	st := wfSt(g)
	c := o.Ctx
	out, eff := "You nod, and walk on. Whatever it was, it is behind you now.", ""
	if i < len(t.Ch) {
		ch := &t.Ch[i]
		eff, out = ch.Win, ch.WT
		if ch.Chk != "" {
			f := strings.Split(ch.Chk, ":")
			dc, _ := strconv.Atoi(f[1])
			if g.roll2d6Check(g.checkMod(f[0]), dc) {
				g.gainXP(4 + dc)
				out = "{{G|[" + checkKinds[f[0]].label + ": Success]}} " + ch.WT
			} else {
				eff, out = ch.Lose, "{{R|["+checkKinds[f[0]].label+": Failure]}} "+ch.LT
			}
		}
	}
	lines, keep := g.wfApply(t, o, eff)
	st.res = append(Wrap(crFill(out, c), 96), "")
	st.res = append(st.res, lines...)
	o.Done = true
	if !keep {
		g.wfRemove(o)
	}
	st.phase = 1
}

func (g *Game) wfZone(o *wfOpen) *Zone {
	if zn := g.Zones[o.Zone]; zn != nil {
		return zn
	}
	return g.Cur
}

func (g *Game) wfRemove(o *wfOpen) {
	zn := g.wfZone(o)
	if zn == nil {
		return
	}
	keep := zn.Actors[:0]
	for _, a := range zn.Actors {
		if a.Has("wfx") && !a.Angry && a.AI != "hostile" {
			continue
		}
		keep = append(keep, a)
	}
	zn.Actors = keep
}

func (g *Game) wfHostile(o *wfOpen) {
	zn := g.wfZone(o)
	if zn == nil {
		return
	}
	for _, a := range zn.Actors {
		if a.Has("wfx") && !a.Has("wfobj") && !a.Has("wfstag") && a.HP > 0 {
			a.Angry = true
			a.AI = "hostile"
			if a.Tpl != "npc" {
				a.Faction = map[bool]string{true: "outlaws", false: "beasts"}[a.Tpl == "bandit" || a.Tpl == "archer" || a.Tpl == "chief"]
			}
		}
	}
}

func (g *Game) wfFoes(o *wfOpen, tpl string, n int) {
	zn := g.wfZone(o)
	if zn == nil || tplByID[tpl] == nil {
		return
	}
	r := NewRng(fmt.Sprint(g.Seed, ":wf:foe:", o.Zone, ":", g.Turn))
	lvl := maxi(1, g.zoneLevel(zn.WX, zn.WY, 0))
	for i := 0; i < n; i++ {
		x, y, ok := zn.FreeSpot(r, func(x, y int) bool {
			d := maxi(abs(x-g.P.X), abs(y-g.P.Y))
			return d >= 4 && d <= 9
		})
		if !ok {
			return
		}
		a := g.Spawn(tpl, lvl, r)
		a.X, a.Y, a.HomeX, a.HomeY = x, y, x, y
		a.AI = "hostile"
		a.Set("wfx")
		a.Angry = true
		zn.Actors = append(zn.Actors, a)
	}
}

// ---------------------------------------------------------------- effects

// wfApply runs an effect string: extra tokens here, everything else through crApply. keep = the cast stays.
func (g *Game) wfApply(t *wfTpl, o *wfOpen, eff string) ([]string, bool) {
	var lines, rest []string
	keep := false
	c := o.Ctx
	u := 5 + 3*g.P.Level
	flush := func() {
		if len(rest) > 0 {
			lines = append(lines, g.crApply(strings.Join(rest, ";"), c)...)
			rest = nil
		}
	}
loop:
	for _, tk := range strings.Split(eff, ";") {
		tk = strings.TrimSpace(tk)
		if tk == "" {
			continue
		}
		f := strings.SplitN(tk, ":", 3)
		num := func(i int) int {
			if i < len(f) {
				n, _ := strconv.Atoi(strings.TrimPrefix(f[i], "+"))
				return n
			}
			return 0
		}
		switch f[0] {
		case "coin":
			flush()
			n := num(1) * u
			if n >= 0 {
				lines = append(lines, "{{y|+"+g.cuEarn(n)+"}}")
			} else if g.cuSpend(-n) {
				lines = append(lines, "{{y|-"+g.cuFormat(-n)+"}}")
			} else {
				lines = append(lines, "{{y|You cannot pay "+g.cuFormat(-n)+".}}")
				break loop
			}
		case "hostile":
			g.wfHostile(o)
			keep = true
			lines = append(lines, "{{R|Steel comes out.}}")
		case "foe":
			g.wfFoes(o, f[1], maxi(1, num(2)))
			keep = true
			lines = append(lines, "{{R|Trouble steps out of the brush.}}")
		case "clue", "tip":
			flush()
			lines = append(lines, g.wfClue(o, f[0] == "clue")...)
		case "rumour":
			saQueueRumour(g, "wayfarer", "player", crFill(strings.Join(f[1:], ":"), c))
		case "map":
			flush()
			lines = append(lines, g.wfMap(o)...)
		case "job":
			flush()
			lines = append(lines, g.wfJobNew(t, o)...)
		default:
			rest = append(rest, tk)
		}
	}
	flush()
	return lines, keep
}

// wfClue: a hint about the nearest dungeon, tower or ruin; also noted as a heard fact in the case notes.
func (g *Game) wfClue(o *wfOpen, fact bool) []string {
	text, p, ok := g.nearestSite()
	if !ok {
		return []string{"{{K|They know nothing you do not.}}"}
	}
	g.addSecret(text, "dungeon", p, g.W.Year)
	lines := []string{"{{C|A place goes into your journal: " + text + "}}"}
	if fact {
		st := knSt(g)
		f := st.add(knFact{Kind: "rumour", Subject: "burg:0", Text: text, True: true, Day: lvDay(g), Lvl: 0})
		from := &lvPerson{Key: "wf:" + o.TID, Name: nonEmptyCr(o.Ctx["E"], "a wayfarer"), Role: "wayfarer", Alive: true}
		g.knLearn(f, from, 2, "ask")
	}
	return lines
}

func (g *Game) wfMap(o *wfOpen) []string {
	text, _, ok := g.nearestSite()
	body := "A hand-drawn map, smudged by thumbs."
	if ok {
		body += " It marks a place: " + text
	}
	it := g.MakeBook("hand-drawn map", body)
	g.give(it)
	lines := []string{"{{G|You gain a hand-drawn map.}}"}
	lines = append(lines, g.wfClue(o, false)...)
	return lines
}

// ---------------------------------------------------------------- jobs (escort, parcel, message, last words)

func (g *Game) wfJobNew(t *wfTpl, o *wfOpen) []string {
	if t == nil || t.Job == "" {
		return nil
	}
	st := wfSt(g)
	c := o.Ctx
	dest, _ := strconv.Atoi(c["DestID"])
	if dest <= 0 {
		return nil
	}
	if t.Job == "grave" {
		if id, _ := strconv.Atoi(c["Pgb"]); id > 0 {
			if b := g.W.burgByID[id]; b != nil && b.Alive {
				dest, c["Dest"] = id, b.Name
			}
		}
	}
	bd := g.W.burgByID[dest]
	if bd == nil {
		return nil
	}
	days := maxi(t.JobDays, 3)
	j := &wfJob{QID: g.newID(), Kind: t.Job, Title: crFill(t.JobTitle, c), Dest: dest, DestName: bd.Name, Due: g.Day() + days, Eff: t.JobEff, Giver: nonEmptyCr(c["E"], "a wayfarer")}
	j.Ctx = map[string]string{}
	for k, v := range c {
		j.Ctx[k] = v
	}
	j.Ctx["Dest"] = bd.Name
	st.Jobs = append(st.Jobs, j)
	g.Quests = append(g.Quests, &Quest{ID: j.QID, Kind: "wf", Title: j.Title, Text: crFill(t.JobText, j.Ctx), Giver: j.Giver, GiverID: -1, Faction: "player", TP: bd.P})
	lines := []string{fmt.Sprintf("{{C|New errand: %s. Be in %s within %d days.}}", j.Title, bd.Name, days)}
	if t.Job == "parcel" {
		it := &Item{ID: g.newID(), Kind: "letter", Name: "sealed parcel for " + bd.Name, Glyph: '=', F: CBrown, Value: 1,
			Text: "A parcel wrapped in oilcloth and tied with a dead man's knots. It is addressed to someone in " + bd.Name + ".", Extra: map[string]int{"wf": j.QID}}
		g.give(it)
		lines = append(lines, "{{G|You take the parcel.}}")
	}
	return lines
}

func (g *Game) wfQuest(id int) *Quest {
	for _, q := range g.Quests {
		if q.ID == id {
			return q
		}
	}
	return nil
}

func (g *Game) wfJobsTick() {
	st := wfSt(g)
	day := g.Day()
	keep := st.Jobs[:0]
	for _, j := range st.Jobs {
		if day > j.Due {
			if q := g.wfQuest(j.QID); q != nil {
				q.Closed = true
			}
			g.Msg("{{R|You let the errand lapse: " + j.Title + ". " + j.Giver + " will have found someone else, or no one.}}")
			g.removeParcel(j.QID)
			continue
		}
		keep = append(keep, j)
	}
	st.Jobs = keep
}

func (g *Game) removeParcel(qid int) bool {
	for i, it := range g.P.Inv {
		if it.Extra != nil && it.Extra["wf"] == qid {
			g.P.Inv = append(g.P.Inv[:i], g.P.Inv[i+1:]...)
			return true
		}
	}
	return false
}

func (g *Game) hasParcel(qid int) bool {
	for _, it := range g.P.Inv {
		if it.Extra != nil && it.Extra["wf"] == qid {
			return true
		}
	}
	return false
}

// wfJobsAt finishes every errand bound for this town.
func (g *Game) wfJobsAt(burg int) {
	st := wfSt(g)
	keep := st.Jobs[:0]
	var done []*wfJob
	for _, j := range st.Jobs {
		if j.Dest == burg && (j.Kind != "parcel" || g.hasParcel(j.QID)) {
			done = append(done, j)
			continue
		}
		keep = append(keep, j)
	}
	st.Jobs = keep
	for _, j := range done {
		if j.Kind == "parcel" {
			g.removeParcel(j.QID)
		}
		if q := g.wfQuest(j.QID); q != nil {
			q.Done, q.Closed = true, true
		}
		g.Msg("{{G|Errand done: " + j.Title + ".}}")
		o := &wfOpen{TID: "job", Ctx: j.Ctx}
		lines, _ := g.wfApply(nil, o, j.Eff)
		for _, l := range lines {
			g.Msg(l)
		}
		st.Kept++
		g.deed("event", fmt.Sprintf("%s kept a promise to %s and brought it to %s.", g.P.Name, j.Giver, j.DestName), 0, 0)
	}
}

// wfAmbush: an escort on the road may be set upon in a wild zone (hash of seed, zone, day).
func (g *Game) wfAmbush(zn *Zone) {
	st := wfSt(g)
	for _, j := range st.Jobs {
		if j.Kind != "escort" && j.Kind != "message" || j.Ambush >= 2 {
			continue
		}
		if HashStr(fmt.Sprint(g.Seed, ":wf:amb:", zn.Key, ":", g.Day(), ":", j.QID))%100 >= 14 {
			continue
		}
		j.Ambush++
		g.Msg("{{R|Word of your errand has travelled faster than you.}}")
		o := &wfOpen{TID: "amb", Ctx: map[string]string{}, Zone: zn.Key}
		g.wfFoes(o, "bandit", 2)
		return
	}
}

// ---------------------------------------------------------------- screen

func (g *Game) wfDraw() {
	sc := g.sc
	st := wfSt(g)
	t, o := g.wfCurrent()
	if t == nil {
		return
	}
	c := o.Ctx
	x, y, w, _ := g.panel("Wayfarers")
	sc.TextB(x, y, "{{M|"+crFill(t.Title, c)+"}}", CGray, CBgDark, x+w)
	y += 2
	if st.phase == 1 {
		for _, l := range st.res {
			sc.TextB(x, y, l, CGray, CBgDark, x+w)
			y++
		}
		y++
		sc.TextB(x, y, "{{K|Enter to go on.}}", CGray, CBgDark, x+w)
		return
	}
	for _, l := range Wrap(crFill(t.Text, c), w-2) {
		sc.TextB(x, y, l, CGray, CBgDark, x+w)
		y++
	}
	y++
	opts := g.wfOptions(t, o)
	g.Sel = clampi(g.Sel, 0, len(opts)-1)
	for i, op := range opts {
		bg := CBgDark
		if i == g.Sel {
			bg = CBgSel
		}
		col := "W"
		if !op.ok {
			col = "K"
		}
		ls := Wrap(fmt.Sprintf("{{%s|%c)}} %s", col, 'a'+i, op.label), w-4)
		if !op.ok {
			ls = Wrap(fmt.Sprintf("{{K|%c) %s (%s)}}", 'a'+i, Plain(op.label), op.why), w-4)
		}
		for j, l := range ls {
			sc.Fill(x, y, w, 1, bg)
			if j > 0 {
				l = "   " + l
			}
			sc.TextB(x, y, l, CGray, bg, x+w)
			y++
		}
	}
	sc.TextB(x, y+1, "{{K|Letter or Enter to choose. Escape to think about it; they will wait.}}", CGray, CBgDark, x+w)
}

// ---------------------------------------------------------------- helpers for the data file

// wfDead picks a real dead notable of this world for a ghost (hash of seed and day), filling c.
func wfDead(g *Game, c map[string]string, r *Rng) bool {
	var ids []int
	for i := range g.W.People {
		p := &g.W.People[i]
		if !p.Alive && p.Name != "" && p.Renown >= 10 && p.Died > 0 {
			ids = append(ids, i)
		}
	}
	if len(ids) == 0 {
		return false
	}
	sort.Ints(ids)
	p := &g.W.People[ids[r.Int(0, len(ids)-1)]]
	c["Pg"], c["Pgr"] = p.Name, nonEmptyCr(p.Title, p.Role)
	c["Pgc"] = nonEmptyCr(p.Cause, "an unmarked death")
	c["Pgy"] = fmt.Sprint(p.Died)
	c["E"], c["Et"] = p.Name, "ghost of the "+nonEmptyCr(p.Title, nonEmptyCr(p.Role, "dead"))
	if b := g.W.burgByID[p.Burg]; b != nil {
		c["Pgb"], c["PgT"] = strconv.Itoa(b.ID), b.Name
	} else {
		c["Pgb"], c["PgT"] = "0", c["T"]
	}
	return true
}
