package engine

// Generated campaigns (prefix cmp): multi-act plots built from the world's real notables and burgs, plus questline
// ladders for every order and sect the player joins. State g.ext("cmp"), planet-local. Generation uses side rngs
// only (NewRng(seed+":cmp:"+...)); g.R is used by play-time checks. WorldData, lives and knowledge state are read
// (only remember(), addNews, knowledge facts, reputation and deeds are written). Screen: mode "cmp", ability "cmp_plots".

import (
	"fmt"
	"sort"
	"strings"
)

type cmpCast struct {
	Role string `json:"r"` // boss ally wit vic
	Key  string `json:"k"`
	Name string `json:"n"`
	Burg int    `json:"b"`
	Desc string `json:"d"`
}

type cmpClue struct {
	ID    int    `json:"i"`
	Act   int    `json:"a"`
	Kind  string `json:"k"` // talk rumour doc book dungeon
	Who   string `json:"w,omitempty"`
	WName string `json:"wn,omitempty"`
	Burg  int    `json:"b"`
	P     int    `json:"pp"`
	Zone  string `json:"z,omitempty"`
	Site  string `json:"s,omitempty"`
	Name  string `json:"n"`
	Text  string `json:"t"`
	Chk   string `json:"c,omitempty"`
	DC    int    `json:"dc,omitempty"`
	Found bool   `json:"f,omitempty"`
	Put   bool   `json:"pt,omitempty"`
	TryT  int    `json:"tt,omitempty"`
}

type cmpAct struct {
	Title string `json:"t"`
	Brief string `json:"b"`
}

type cmpPlot struct {
	ID      int        `json:"i"`
	Tpl     string     `json:"tp"`
	Title   string     `json:"t"`
	Hook    string     `json:"h"`
	Home    int        `json:"hm"`
	Town2   int        `json:"t2"`
	State   int        `json:"st"`
	Cast    []cmpCast  `json:"c"`
	Acts    []cmpAct   `json:"a"`
	Clues   []*cmpClue `json:"cl"`
	Cur     int        `json:"cu"`
	Status  string     `json:"s,omitempty"` // "" unknown, active, ready, done
	End     string     `json:"e,omitempty"`
	EndDay  int        `json:"ed,omitempty"`
	EpiDay  int        `json:"ep,omitempty"`
	QID     int        `json:"q,omitempty"`
	Hooked  bool       `json:"hk,omitempty"`
	Pending string     `json:"pd,omitempty"`
	Tried   bool       `json:"tr,omitempty"`
	Realm   string     `json:"rl,omitempty"`
}

type cmpStep struct {
	T       string `json:"t"` // audience tasks slay relic tithe pilgrim rank
	Text    string `json:"x"`
	N       int    `json:"n,omitempty"`
	Base    int    `json:"b,omitempty"`
	Have    int    `json:"h,omitempty"`
	Zone    string `json:"z,omitempty"`
	Burg    int    `json:"bg,omitempty"`
	P       int    `json:"p,omitempty"`
	Name    string `json:"nm,omitempty"`
	Started bool   `json:"s,omitempty"`
	Spawned bool   `json:"sp,omitempty"`
	Ready   bool   `json:"r,omitempty"`
	Gold    int    `json:"g"`
	XP      int    `json:"xp"`
	Rep     int    `json:"rp"`
}

type cmpLad struct {
	Key    string     `json:"k"`
	Ref    string     `json:"rf"`
	Sect   bool       `json:"sc,omitempty"`
	Title  string     `json:"t"`
	Name   string     `json:"n"`
	Kind   string     `json:"kd"`
	Leader string     `json:"l"`
	Tit    string     `json:"lt"`
	RepKey string     `json:"rk"`
	Creed  string     `json:"cr,omitempty"`
	Burg   int        `json:"b"`
	Steps  []*cmpStep `json:"s"`
	Cur    int        `json:"c"`
	QID    int        `json:"q,omitempty"`
	Done   bool       `json:"d,omitempty"`
	Lapsed bool       `json:"lp,omitempty"`
}

type cmpUI struct {
	sel, tab, off int
}

type cmpState struct {
	Gen   bool               `json:"g,omitempty"`
	Plots []*cmpPlot         `json:"p,omitempty"`
	Lads  map[string]*cmpLad `json:"l,omitempty"`
	Scan  int                `json:"sc,omitempty"`

	ui cmpUI
}

func init() {
	unLocalKeys = append(unLocalKeys, "cmp")
	turnHooks = append(turnHooks, cmpTurn)
	enterHooks = append(enterHooks, cmpEnter)
	talkHooks = append(talkHooks, cmpTalk)
	killHooks = append(killHooks, cmpKill)
	modeKeys["cmp"] = cmpKeys
	modeDraws["cmp"] = func(g *Game) { g.cmpDraw() }
}

func cmpSt(g *Game) *cmpState {
	s := g.ext("cmp", func() any { return &cmpState{} }).(*cmpState)
	if s.Lads == nil {
		s.Lads = map[string]*cmpLad{}
	}
	return s
}

func cmpPlotByID(st *cmpState, id int) *cmpPlot {
	for _, p := range st.Plots {
		if p.ID == id {
			return p
		}
	}
	return nil
}

func (p *cmpPlot) clue(id int) *cmpClue {
	for _, c := range p.Clues {
		if c.ID == id {
			return c
		}
	}
	return nil
}

func (p *cmpPlot) boss() *cmpCast { return &p.Cast[0] }

func (p *cmpPlot) fill(s string) string {
	pairs := []string{"{town}", "{town}", "{town2}", "{town2}", "{realm}", p.Realm}
	for _, c := range p.Cast {
		pairs = append(pairs, "{"+c.Role+"}", c.Name)
	}
	return strings.NewReplacer(pairs...).Replace(s)
}

func cmpTplByID(id string) *cmpTpl {
	for _, t := range cmpTpls {
		if t.ID == id {
			return t
		}
	}
	return nil
}

func cmpBurgName(g *Game, b int) string { return g.knBurgName(b) }

func cmpBurgP(g *Game, b int) int {
	if bd := g.W.burgByID[b]; bd != nil {
		return bd.P
	}
	return 0
}

// ---------------------------------------------------------------- generation

// cmpSiteNear finds a dungeon near a parasang and returns its first-level zone key, name and parasang.
func cmpSiteNear(g *Game, wx, wy, maxRad int, r *Rng) (key, name string, p int, ok bool) {
	type cand struct {
		key, name string
		p         int
	}
	var cs []cand
	for rad := 0; rad <= maxRad && len(cs) < 4; rad++ {
		for dy := -rad; dy <= rad; dy++ {
			for dx := -rad; dx <= rad; dx++ {
				if maxi(abs(dx), abs(dy)) != rad {
					continue
				}
				x, y := wx+dx, wy+dy
				if !g.W.In(x, y) {
					continue
				}
				for zy := 0; zy < 3; zy++ {
					for zx := 0; zx < 3; zx++ {
						if s := g.siteAt(x, y, zx, zy); s != nil && s.Kind == "dungeon" && s.Depth >= 1 {
							cs = append(cs, cand{ZoneKey(x, y, zx, zy, 1), s.Name, g.W.Idx(x, y)})
						}
					}
				}
			}
		}
	}
	if len(cs) == 0 {
		return "", "", 0, false
	}
	c := cs[r.Int(0, len(cs)-1)]
	return c.key, c.name, c.p, true
}

func cmpGen(g *Game, st *cmpState) {
	st.Gen = true
	w := g.W
	r := NewRng(fmt.Sprintf("%s:cmp:plots:%d", w.Seed, w.uniIdx))
	byBurg := map[int][]*PersonData{}
	for i := range w.People {
		pp := &w.People[i]
		if !pp.Alive || pp.Burg <= 0 {
			continue
		}
		if bd := w.burgByID[pp.Burg]; bd == nil || !bd.Alive {
			continue
		}
		byBurg[pp.Burg] = append(byBurg[pp.Burg], pp)
	}
	var burgs []int
	for b := range byBurg {
		burgs = append(burgs, b)
	}
	sort.Ints(burgs)
	if len(burgs) < 3 {
		return
	}
	shuffleInts := func(a []int) {
		for i := len(a) - 1; i > 0; i-- {
			j := r.Int(0, i)
			a[i], a[j] = a[j], a[i]
		}
	}
	shuffleInts(burgs)
	order := make([]int, len(cmpTpls))
	for i := range order {
		order[i] = i
	}
	shuffleInts(order)
	want := r.Int(2, 4)
	usedB := map[int]bool{}
	usedP := map[string]bool{}
	for _, ti := range order {
		if len(st.Plots) >= want {
			break
		}
		for _, a := range burgs {
			if usedB[a] {
				continue
			}
			if p := cmpBuild(g, cmpTpls[ti], r, a, byBurg, burgs, usedB, usedP, len(st.Plots)+1); p != nil {
				st.Plots = append(st.Plots, p)
				break
			}
		}
	}
}

func cmpBuild(g *Game, tpl *cmpTpl, r *Rng, a int, byBurg map[int][]*PersonData, burgs []int, usedB map[int]bool, usedP map[string]bool, id int) *cmpPlot {
	w := g.W
	bdA := w.burgByID[a]
	if bdA == nil {
		return nil
	}
	has := func(list []string, role string) int {
		for i, x := range list {
			if x == role {
				return i
			}
		}
		return 99
	}
	pick := func(cands []*PersonData, pref []string) *PersonData {
		var best *PersonData
		bi := 100
		for _, c := range cands {
			if usedP[fmt.Sprint(c.ID)] {
				continue
			}
			if i := has(pref, c.Role); i < bi {
				best, bi = c, i
			}
		}
		return best
	}
	boss := pick(byBurg[a], tpl.Pref)
	if boss == nil {
		return nil
	}
	usedP[fmt.Sprint(boss.ID)] = true
	// nearby burgs by distance from A
	near := append([]int{}, burgs...)
	sort.SliceStable(near, func(i, j int) bool {
		return wsCheb(w, w.burgByID[near[i]].P, bdA.P) < wsCheb(w, w.burgByID[near[j]].P, bdA.P)
	})
	var ally, wit, vic *PersonData
	ally = pick(byBurg[a], nil)
	if ally != nil {
		usedP[fmt.Sprint(ally.ID)] = true
	}
	var others []*PersonData
	n := 0
	for _, b := range near {
		if b == a {
			continue
		}
		if wsCheb(w, w.burgByID[b].P, bdA.P) > 20 || n >= 6 {
			break
		}
		others = append(others, byBurg[b]...)
		n++
	}
	if ally == nil {
		if ally = pick(others, nil); ally == nil {
			return nil
		}
		usedP[fmt.Sprint(ally.ID)] = true
	}
	if wit = pick(others, nil); wit == nil {
		return nil
	}
	usedP[fmt.Sprint(wit.ID)] = true
	if vic = pick(others, tpl.VicPref); vic == nil {
		if vic = pick(byBurg[a], tpl.VicPref); vic == nil {
			return nil
		}
	}
	usedP[fmt.Sprint(vic.ID)] = true
	usedB[a] = true
	p := &cmpPlot{ID: id, Tpl: tpl.ID, Home: a, Town2: wit.Burg, State: bdA.State, Title: tpl.Title}
	if sd := w.stateByID[bdA.State]; sd != nil {
		p.Realm = sd.Name
	} else {
		p.Realm = "the realm"
	}
	for i, pd := range []*PersonData{boss, ally, wit, vic} {
		p.Cast = append(p.Cast, cmpCast{Role: []string{"boss", "ally", "wit", "vic"}[i], Key: fmt.Sprintf("s%d", pd.ID), Name: pd.Name, Burg: pd.Burg, Desc: tpl.Roles[i]})
	}
	p.Title = tpl.Title + " of " + bdA.Name
	p.Hook = p.fill(tpl.Hook)
	// acts
	idx := []int{0, 1, 2, 3, 4}
	switch r.Int(0, 2) {
	case 0:
		idx = []int{0, 1, 2, 3, 4}
	case 1:
		omit := r.Int(1, 3)
		idx = nil
		for i := 0; i < 5; i++ {
			if i != omit {
				idx = append(idx, i)
			}
		}
	default:
		keep := r.Int(1, 3)
		idx = []int{0, keep, 4}
	}
	skey, sname, sp, sok := cmpSiteNear(g, bdA.P%w.W, bdA.P/w.W, 10, r)
	nextClue := 0
	for ai, ti := range idx {
		at := tpl.Acts[ti]
		p.Acts = append(p.Acts, cmpAct{Title: at.Title, Brief: p.fill(at.Brief)})
		for _, ct := range at.Clues {
			nextClue++
			cl := &cmpClue{ID: nextClue, Act: ai, Kind: ct.K, Name: ct.N, Chk: ct.C, DC: ct.D}
			loc := a
			if ct.L == "B" {
				loc = p.Town2
			}
			cl.Burg = loc
			txt := ct.T
			switch ct.K {
			case "talk":
				for _, c := range p.Cast {
					if c.Role == ct.W {
						cl.Who, cl.WName, cl.Burg = c.Key, c.Name, c.Burg
					}
				}
			case "dungeon":
				if sok {
					cl.Zone, cl.Site, cl.P = skey, sname, sp
					cl.Burg = a
				} else {
					cl.Kind, cl.Burg = "doc", p.Town2
				}
			}
			if cl.P == 0 {
				cl.P = cmpBurgP(g, cl.Burg)
			}
			if ct.K == "dungeon" && sok {
				txt = strings.ReplaceAll(txt, "{site}", sname)
			} else {
				txt = strings.ReplaceAll(txt, "{site}", "a hidden cellar")
			}
			cl.Text = p.fill(txt)
			cl.Name = p.fill(cl.Name)
			p.Clues = append(p.Clues, cl)
		}
	}
	return p
}

// ---------------------------------------------------------------- journal mirror

func cmpQuestByID(g *Game, id int) *Quest {
	for _, q := range g.Quests {
		if q.ID == id {
			return q
		}
	}
	return nil
}

func (p *cmpPlot) curClues() []*cmpClue {
	var out []*cmpClue
	for _, c := range p.Clues {
		if c.Act == p.Cur && !c.Found {
			out = append(out, c)
		}
	}
	return out
}

func cmpHint(g *Game, c *cmpClue) string {
	switch c.Kind {
	case "talk":
		return fmt.Sprintf("Ask %s in %s about %s.", c.WName, cmpBurgName(g, c.Burg), c.Name)
	case "rumour":
		return fmt.Sprintf("Listen for gossip in %s.", cmpBurgName(g, c.Burg))
	case "dungeon":
		return fmt.Sprintf("Search %s (near %s) for %s.", c.Site, cmpBurgName(g, c.Burg), c.Name)
	}
	return fmt.Sprintf("Search %s for %s.", cmpBurgName(g, c.Burg), c.Name)
}

func cmpSyncPlot(g *Game, p *cmpPlot) {
	if p.QID == 0 {
		return
	}
	q := cmpQuestByID(g, p.QID)
	if q == nil {
		return
	}
	switch p.Status {
	case "done":
		q.Done, q.Closed = true, true
		q.Text = "Ended: " + p.End + "."
		return
	case "ready":
		q.Text = fmt.Sprintf("You have the proof. Confront %s in %s.", p.boss().Name, cmpBurgName(g, p.boss().Burg))
		q.TP, q.Target = cmpBurgP(g, p.boss().Burg), ""
		return
	}
	if p.Cur < len(p.Acts) {
		cl := p.curClues()
		txt := fmt.Sprintf("Act %d of %d, %s. %s", p.Cur+1, len(p.Acts), p.Acts[p.Cur].Title, p.Acts[p.Cur].Brief)
		if len(cl) > 0 {
			txt += " " + cmpHint(g, cl[0])
			q.TP = cl[0].P
			q.Target = cl[0].Zone
		}
		q.Text = txt
	}
}

func cmpOpenQuest(g *Game, p *cmpPlot) {
	if p.QID != 0 && cmpQuestByID(g, p.QID) != nil {
		return
	}
	p.QID = g.newID()
	g.Quests = append(g.Quests, &Quest{ID: p.QID, Kind: "campaign", Title: p.Title, Text: p.Hook, Giver: "rumour", GiverID: -1, Faction: "player", TP: cmpBurgP(g, p.Home)})
	cmpSyncPlot(g, p)
}

// ---------------------------------------------------------------- clues

func cmpFound(g *Game, p *cmpPlot, cl *cmpClue) {
	if cl.Found {
		return
	}
	cl.Found = true
	g.Msg("{{Y|Clue}} ({{c|" + p.Title + "}}): " + cl.Text)
	g.gainXP(10 + 4*cl.Act)
	if cl.Kind == "rumour" || cl.Kind == "talk" {
		kn := knSt(g)
		subj := p.boss().Key
		f := kn.add(knFact{Kind: "rumour", Subject: subj, Text: cl.Text, True: true, Source: cl.Who, Day: lvDay(g), Burg: cl.Burg})
		if pw := lvGet(g, cl.Who); pw != nil && cl.Who != "" {
			knGive(pw, f.ID)
		}
	}
	cmpAdvance(g, p)
}

func (p *cmpPlot) actDone(a int) bool {
	for _, c := range p.Clues {
		if c.Act == a && !c.Found {
			return false
		}
	}
	return true
}

func cmpAdvance(g *Game, p *cmpPlot) {
	for p.Status == "active" && p.Cur < len(p.Acts) && p.actDone(p.Cur) {
		g.Msg(fmt.Sprintf("{{G|%s: act %d (%s) complete.}}", p.Title, p.Cur+1, p.Acts[p.Cur].Title))
		p.Cur++
	}
	if p.Status == "active" && p.Cur >= len(p.Acts) {
		p.Status = "ready"
		g.Msg(fmt.Sprintf("{{R|You have all you need. Confront %s in %s.}}", p.boss().Name, cmpBurgName(g, p.boss().Burg)))
	}
	cmpSyncPlot(g, p)
}

func cmpStart(g *Game, p *cmpPlot) string {
	p.Status = "active"
	cmpOpenQuest(g, p)
	g.deedOnce("cmpstart:"+fmt.Sprint(p.ID), "campaign", fmt.Sprintf("The wanderer %s took up the matter of \"%s\".", g.P.Name, p.Title))
	for _, c := range p.Clues {
		if c.Act == 0 && c.Kind == "rumour" && !c.Found {
			cmpFound(g, p, c)
			return c.Text
		}
	}
	return p.Hook
}

func (g *Game) deedOnce(key, kind, text string) {
	if g.Cur == nil || g.W == nil {
		return
	}
	st := cmpSt(g)
	_ = st
	for _, d := range g.Deeds {
		if d.Text == text {
			return
		}
	}
	g.deed(kind, text, 0, 0)
}

// ---------------------------------------------------------------- turn, enter, kill

func cmpTurn(g *Game) {
	if g == nil || g.W == nil || g.P == nil || g.Cur == nil {
		return
	}
	st := cmpSt(g)
	if !st.Gen {
		cmpGen(g, st)
	}
	// pick up clue items
	for _, it := range g.P.Inv {
		if it.Extra == nil || it.Extra["cmp"] == 0 {
			continue
		}
		if p := cmpPlotByID(st, it.Extra["cmp"]); p != nil {
			if cl := p.clue(it.Extra["cmpc"]); cl != nil && !cl.Found && p.Status == "active" {
				cmpFound(g, p, cl)
			}
		}
	}
	if st.Scan != 0 && g.Turn-st.Scan < 20 && g.Turn >= st.Scan {
		return
	}
	st.Scan = g.Turn
	day := lvDay(g)
	pp, pb := lsPlayerLoc(g)
	for _, p := range st.Plots {
		if p.Status == "" && !p.Hooked && pp >= 0 {
			hp, tp := cmpBurgP(g, p.Home), cmpBurgP(g, p.Town2)
			if pb == p.Home || pb == p.Town2 || wsCheb(g.W, hp, pp) <= 2 || wsCheb(g.W, tp, pp) <= 2 {
				p.Hooked = true
				lvSt(g).addNews(day, p.Home, "%s", p.Hook)
			}
		}
		if p.Status == "done" && p.EpiDay > 0 && day >= p.EpiDay {
			p.EpiDay = 0
			lvSt(g).addNews(day, p.Home, "%s", cmpEpilogue(g, p))
		}
		if p.Status == "active" || p.Status == "ready" {
			cmpPlaceHere(g, p, g.Cur)
		}
	}
	cmpLadSync(g, st)
}

func cmpEpilogue(g *Game, p *cmpPlot) string {
	b := p.boss().Name
	t := cmpBurgName(g, p.Home)
	switch p.End {
	case "expose":
		return fmt.Sprintf("%s is still the talk of %s after the affair of \"%s\"; few will trust a name they once did.", b, t, p.Title)
	case "join":
		return fmt.Sprintf("In %s, folk say %s has grown bolder since the affair of \"%s\" went quiet.", t, b, p.Title)
	case "blackmail":
		return fmt.Sprintf("%s of %s is said to be paying someone, though nobody knows whom or why.", b, t)
	case "kill":
		return fmt.Sprintf("The death of %s of %s is still debated; some say it was long overdue.", b, t)
	}
	return fmt.Sprintf("In %s the matter of \"%s\" went unanswered, and the consequences are catching up.", t, p.Title)
}

func cmpEnter(g *Game, zn *Zone) {
	if g == nil || zn == nil || g.W == nil {
		return
	}
	st := cmpSt(g)
	if !st.Gen {
		return
	}
	for _, p := range st.Plots {
		if p.Status == "active" || p.Status == "ready" {
			cmpPlaceHere(g, p, zn)
		}
	}
	for _, key := range cmpLadKeys(st) {
		l := st.Lads[key]
		if l.Done || l.Lapsed || l.Cur >= len(l.Steps) {
			continue
		}
		s := l.Steps[l.Cur]
		if !s.Started {
			continue
		}
		switch s.T {
		case "pilgrim":
			if zn.Kind == "town" && zn.Burg == s.Burg && !s.Ready {
				cmpStepReady(g, l, s)
			}
		case "relic":
			if !s.Spawned && zn.Key == s.Zone {
				r := NewRng(fmt.Sprintf("%s:cmp:relic:%s:%d", g.W.Seed, l.Key, l.Cur))
				if x, y, ok := zn.FreeSpot(r, nil); ok {
					s.Spawned = true
					zn.Items = append(zn.Items, &Item{ID: g.newID(), Kind: "trophy", Name: s.Name, Glyph: '♦', F: CYellow, X: x, Y: y, Weight: 1, Value: 60, Ident: true,
						Text: fmt.Sprintf("cmp-relic:%s:%d", l.Key, l.Cur), Lore: "Lost to " + l.Name + " long ago."})
					g.Msg("{{Y|Something that belongs to " + l.Name + " lies in the dark...}}")
				}
			}
		}
	}
}

// cmpPlaceHere puts the current act's clue items into the zone being entered or stood in.
func cmpPlaceHere(g *Game, p *cmpPlot, zn *Zone) {
	if zn == nil {
		return
	}
	for _, c := range p.Clues {
		if c.Put || c.Found || c.Act > p.Cur {
			continue
		}
		switch c.Kind {
		case "doc", "book":
			if zn.Kind != "town" || zn.Burg != c.Burg || zn.Z != 0 {
				continue
			}
		case "dungeon":
			if zn.Key != c.Zone {
				continue
			}
		default:
			continue
		}
		r := NewRng(fmt.Sprintf("%s:cmp:item:%d:%d", g.W.Seed, p.ID, c.ID))
		x, y, ok := zn.FreeSpot(r, func(x, y int) bool { return c.Kind == "dungeon" || mxIndoor(zn.At(x, y)) })
		if !ok {
			x, y, ok = zn.FreeSpot(r, nil)
		}
		if !ok {
			continue
		}
		c.Put = true
		it := &Item{ID: g.newID(), Name: c.Name, Glyph: '?', F: CYellow, Kind: "letter", Text: c.Text, X: x, Y: y,
			Lore: "Part of the matter of " + p.Title + ".", Extra: map[string]int{"cmp": p.ID, "cmpc": c.ID}}
		if c.Kind == "book" {
			it.Kind, it.Glyph, it.F, it.Weight, it.Value = "book", '■', CCyan, 1, 6
		}
		zn.Items = append(zn.Items, it)
		if c.Kind == "dungeon" {
			g.Msg("{{Y|Something here bears on " + p.Title + ".}}")
		}
	}
}

func cmpKill(g *Game, d, src *Actor) {
	if g == nil || d == nil {
		return
	}
	st := cmpSt(g)
	for _, p := range st.Plots {
		if p.Pending == "kill" && d.Has(fmt.Sprintf("cmpk:%d", p.ID)) {
			p.Pending = ""
			cmpResolve(g, p, "kill", nil)
		}
	}
}

// ---------------------------------------------------------------- talk

func cmpStandIn(a *Actor) bool {
	switch a.Role {
	case "innkeeper", "elder", "priest", "scholar", "healer":
		return true
	}
	return false
}

func cmpTalk(g *Game, a *Actor) []Option {
	if g == nil || a == nil || g.W == nil || g.Cur == nil || a.AI == "hostile" || a.AI == "follower" || a.Angry {
		return nil
	}
	st := cmpSt(g)
	if !st.Gen {
		return nil
	}
	var opts []Option
	burg := g.Cur.Burg
	pk := ""
	if burg > 0 {
		if lp := lvPersonOf(g, a); lp != nil {
			pk = lp.Key
		}
	}
	for _, p := range st.Plots {
		p := p
		if burg > 0 && p.Status == "" && (burg == p.Home || burg == p.Town2) {
			opts = append(opts, Option{"{{c|Ask what the whispers are about " + p.boss().Name + ".}}", func() {
				g.talkMenu("\"" + cmpStart(g, p) + "\"")
			}})
		}
		if p.Status == "active" {
			for _, c := range p.curClues() {
				c := c
				switch c.Kind {
				case "rumour":
					if burg == c.Burg {
						opts = append(opts, Option{"{{c|Listen for gossip about " + p.boss().Name + ".}}", func() {
							cmpFound(g, p, c)
							g.talkMenu("\"" + c.Text + "\"")
						}})
					}
				case "talk":
					lp := lvGet(g, c.Who)
					alive := lp != nil && lp.Alive
					ok := pk != "" && pk == c.Who
					if !ok && !alive && lp != nil && burg == c.Burg && cmpStandIn(a) {
						ok = true
					}
					if !ok {
						continue
					}
					label := "Ask " + c.WName + " about " + c.Name + "."
					if c.Chk != "" {
						label = g.checkLabel(c.Chk, c.DC) + " Press " + c.WName + " about " + c.Name + "."
					}
					opts = append(opts, Option{label, func() { g.cmpAsk(p, c) }})
				}
			}
		}
		if p.Status == "ready" && pk != "" && pk == p.boss().Key {
			opts = append(opts, Option{"{{R|Confront " + p.boss().Name + " about the affair of " + cmpBurgName(g, p.Home) + ".}}", func() { g.cmpConfront(a, p, "") }})
		}
	}
	opts = append(opts, cmpLadTalk(g, a, st)...)
	return opts
}

func (g *Game) cmpAsk(p *cmpPlot, c *cmpClue) {
	if c.Chk != "" {
		if c.TryT != 0 && g.Turn-c.TryT < 20 && g.Turn >= c.TryT {
			g.talkMenu("\"I have said all I will for now. Come back later.\"")
			return
		}
		if !g.roll2d6Check(g.checkMod(c.Chk), c.DC) {
			c.TryT = g.Turn
			g.talkMenu("{{R|[" + checkKinds[c.Chk].label + ": failure]}}\n\n\"I would rather not say. It is none of my affair.\"")
			return
		}
		g.talkMenu("{{G|[" + checkKinds[c.Chk].label + ": success]}}\n\n\"" + c.Text + "\"")
	} else {
		g.talkMenu("\"" + c.Text + "\"")
	}
	cmpFound(g, p, c)
}

func (g *Game) cmpConfront(a *Actor, p *cmpPlot, text string) {
	if text == "" {
		text = p.fill(fmt.Sprintf("{boss} reads what you have laid out, and goes still. \"So. What do you mean to do with it?\" %s", p.Title))
	}
	g.Popup = Wrap(text, 100)
	g.Opts = nil
	add := func(t string, f func()) { g.Opts = append(g.Opts, Option{t, f}) }
	add("{{G|Expose "+p.boss().Name+" before "+p.Realm+".}}", func() { cmpResolve(g, p, "expose", a) })
	add("{{y|Join "+p.boss().Name+" and share in it.}}", func() { cmpResolve(g, p, "join", a) })
	if !p.Tried {
		add(g.checkLabel("intimidate", 9)+" Blackmail "+p.boss().Name+".", func() {
			if g.roll2d6Check(g.checkMod("intimidate"), 9) {
				cmpResolve(g, p, "blackmail", a)
				return
			}
			p.Tried = true
			if lp := lvGet(g, p.boss().Key); lp != nil {
				lp.remember(lvDay(g), "insult", "You tried to blackmail me over "+p.Title+".", -12)
			}
			g.AddRep(uwFac(p.State), -4, " for threatening a notable")
			g.cmpConfront(a, p, "{{R|[Intimidate: failure]}}\n\n\"You are bluffing, and badly. Get out of my sight.\"")
		})
	}
	add("{{R|Strike "+p.boss().Name+" down.}}", func() {
		p.Pending = "kill"
		a.Angry = true
		a.Set(fmt.Sprintf("cmpk:%d", p.ID))
		g.Popup, g.Opts, g.Talk, g.Mode = nil, nil, nil, "main"
		g.Msg("{{R|" + p.boss().Name + " reaches for a weapon. It ends here.}}")
	})
	add("Walk away from it all.", func() { cmpResolve(g, p, "walk", a) })
	add("Say nothing, for now.", func() { g.talkMenu("\"Another time, then.\"") })
}

func cmpResolve(g *Game, p *cmpPlot, end string, a *Actor) {
	if p.Status == "done" {
		return
	}
	tpl := cmpTplByID(p.Tpl)
	day := lvDay(g)
	base := 100 + 25*len(p.Acts) + g.P.Level*12
	boss := lvGet(g, p.boss().Key)
	var notes []string
	repH := func(n int, why string) { g.AddRep(uwFac(p.State), n, why) }
	pay := func(n int) { notes = append(notes, "{{G|"+g.cuEarn(n)+"}}") }
	rem := func(key, kind, text string, wt int) {
		if lp := lvGet(g, key); lp != nil && lp.Alive {
			lp.remember(day, kind, text, wt)
		}
	}
	switch end {
	case "expose":
		repH(14, " for exposing a plot")
		pay(base / 2)
		g.gainXP(120 + 30*len(p.Acts))
		rem(p.Cast[2].Key, "help", "You brought down the one I feared.", 12)
		rem(p.Cast[3].Key, "saved", "You exposed the plot against me.", 15)
		if boss != nil && boss.Alive {
			boss.remember(day, "attack", "You exposed me before everyone.", -25)
		}
		if a != nil && a.HP > 0 {
			a.Angry = false
		}
	case "join":
		repH(-10, " for joining a conspiracy")
		pay(base * 2)
		g.gainXP(50)
		if boss != nil && boss.Alive {
			boss.remember(day, "quest", "You took my side in the matter of "+p.Title+".", 20)
		}
		rem(p.Cast[3].Key, "insult", "You sided with the one who wronged me.", -10)
	case "blackmail":
		repH(-3, " for extortion")
		pay(base * 3 / 2)
		g.gainXP(80)
		if boss != nil && boss.Alive {
			boss.remember(day, "insult", "You hold the matter of "+p.Title+" over me.", -30)
		}
	case "kill":
		repH(-8, " for killing a notable")
		pay(base / 3)
		g.gainXP(160 + 30*len(p.Acts))
		rem(p.Cast[3].Key, "saved", "The one who wronged me is dead, by your hand.", 12)
	default:
		g.gainXP(20)
		rem(p.Cast[2].Key, "insult", "You knew, and walked away.", -6)
	}
	p.Status, p.End, p.EndDay, p.EpiDay = "done", end, day, day+4
	p.Pending = ""
	text := p.fill(tpl.End[end])
	cmpSyncPlot(g, p)
	if g.Cur != nil {
		verbs := map[string]string{"expose": "exposed", "join": "joined the scheme of", "blackmail": "blackmailed", "kill": "slew", "walk": "walked away from"}
		g.deed("campaign", fmt.Sprintf("The wanderer %s %s %s in the matter of \"%s\".", g.P.Name, verbs[end], p.boss().Name, p.Title), 0, 0)
	}
	if end == "expose" || end == "kill" {
		lvSt(g).addNews(day, p.Home, "A wanderer, %s, has settled the matter of \"%s\" in %s.", g.P.Name, p.Title, cmpBurgName(g, p.Home))
	}
	g.Msg("{{Y|" + p.Title + " ends: " + end + ".}} " + strings.Join(notes, " "))
	if a != nil && g.Mode == "talk" && g.Talk == a && end != "kill" {
		g.talkMenu(text + " " + strings.Join(notes, " "))
	} else {
		g.Msg(text)
	}
}

// ---------------------------------------------------------------- ladders

func cmpLadKeys(st *cmpState) []string {
	var k []string
	for key := range st.Lads {
		k = append(k, key)
	}
	sort.Strings(k)
	return k
}

var cmpRelicNoun = map[string]string{
	"holy": "reliquary", "monastic": "psalter", "cult": "idol", "secret": "seal", "thieves": "strongbox", "merchant": "charter", "scholars": "codex",
	"sect": "scroll", "clan": "ancestral blade", "guild": "banner",
}

func cmpLadSync(g *Game, st *cmpState) {
	os := g.orderState()
	var ids []string
	for id := range os.Mem {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		key := "o:" + id
		if st.Lads[key] == nil {
			if o := g.OrderByID(id); o != nil {
				if l := cmpBuildLad(g, key, id, false, o.Name, o.Kind, o.Leader, o.LeaderTit, o.RepKey(), o.Creed, o.Burg, o.WX, o.WY, cmpOrderPilgrims(g, o)); l != nil {
					st.Lads[key] = l
				}
			}
		}
	}
	if s := g.mrMine(); s != nil {
		key := "m:" + s.ID
		if st.Lads[key] == nil {
			var pil []int
			for _, h := range s.Halls {
				pil = append(pil, h.Burg)
			}
			burg := 0
			if len(s.Halls) > 0 {
				burg = s.Halls[0].Burg
			}
			if l := cmpBuildLad(g, key, s.ID, true, s.Short(), s.Kind, s.Leader, s.LeaderTit, s.RepKey(), s.Creed, burg, s.WX, s.WY, pil); l != nil {
				st.Lads[key] = l
			}
		}
	}
	for _, key := range cmpLadKeys(st) {
		l := st.Lads[key]
		if l.Done {
			continue
		}
		member := false
		if l.Sect {
			member = g.mr().Sect == l.Ref
		} else {
			member = os.Mem[l.Ref] != nil
		}
		if !member {
			if !l.Lapsed {
				l.Lapsed = true
				cmpSyncLad(g, l)
			}
			continue
		}
		if l.Lapsed {
			l.Lapsed = false
		}
		if l.Cur >= len(l.Steps) {
			continue
		}
		if l.QID == 0 || cmpQuestByID(g, l.QID) == nil {
			l.QID = g.newID()
			g.Quests = append(g.Quests, &Quest{ID: l.QID, Kind: "campaign", Title: l.Title, Text: "", Giver: l.Leader, GiverID: -1, Faction: "player", TP: cmpBurgP(g, l.Burg)})
		}
		s := l.Steps[l.Cur]
		if !s.Started {
			cmpStepStart(g, l, s)
		}
		cmpStepEval(g, l, s)
		cmpSyncLad(g, l)
	}
}

func cmpOrderPilgrims(g *Game, o *Order) []int {
	var out []int
	if rv := g.OrderByID(o.Rival); rv != nil && rv.Burg != o.Burg {
		out = append(out, rv.Burg)
	}
	return out
}

func cmpBuildLad(g *Game, key, ref string, sect bool, name, kind, leader, tit, repKey, creed string, burg, wx, wy int, pilgrims []int) *cmpLad {
	r := NewRng(g.W.Seed + ":cmp:lad:" + key)
	name = strings.TrimPrefix(name, "The ")
	l := &cmpLad{Key: key, Ref: ref, Sect: sect, Name: name, Kind: kind, Leader: leader, Tit: tit, RepKey: repKey, Creed: creed, Burg: burg}
	l.Title = "The Oath of " + name
	n := r.Int(3, 5)
	pool := cmpLadPool[kind]
	if pool == nil {
		pool = cmpLadPool[""]
	}
	off := r.Int(0, len(pool)-1)
	types := []string{"audience"}
	for i := 0; i < n-2; i++ {
		types = append(types, pool[(off+i)%len(pool)])
	}
	fin := cmpLadFinal[kind]
	if fin == "" {
		fin = "rank"
	}
	types = append(types, fin)
	noun := cmpRelicNoun[kind]
	if noun == "" {
		noun = "relic"
	}
	siteKey, siteName, _, siteOK := cmpSiteNear(g, wx, wy, 12, r)
	for i, t := range types {
		s := &cmpStep{T: t, Gold: 40 + 45*i, XP: 50 + 70*i, Rep: 4 + 3*i, N: 0}
		if i == len(types)-1 {
			s.Gold, s.XP, s.Rep = s.Gold*2, s.XP*2, s.Rep+6
		}
		switch t {
		case "audience":
			s.Text = fmt.Sprintf("Present yourself to %s, %s of %s, and learn what the society expects of you.", leader, strings.ToLower(tit), name)
		case "tasks":
			s.N = 2 + i
			s.Text = fmt.Sprintf(Pick(r, cmpLadVerbs["tasks"]), s.N, name)
		case "slay":
			s.N = 4 + 3*i
			s.Text = fmt.Sprintf(Pick(r, cmpLadVerbs["slay"]), s.N, name)
		case "relic":
			if !siteOK {
				s.T, s.N = "tasks", 2+i
				s.Text = fmt.Sprintf(Pick(r, cmpLadVerbs["tasks"]), s.N, name)
				break
			}
			s.Zone, s.Name = siteKey, fmt.Sprintf("the lost %s of %s", noun, name)
			s.Text = fmt.Sprintf("Recover %s from the depths of %s.", s.Name, siteName)
		case "pilgrim":
			if len(pilgrims) == 0 {
				s.T, s.N = "tasks", 2+i
				s.Text = fmt.Sprintf(Pick(r, cmpLadVerbs["tasks"]), s.N, name)
				break
			}
			s.Burg = pilgrims[r.Int(0, len(pilgrims)-1)]
			s.Text = fmt.Sprintf("Travel to %s and show the sign of %s there.", cmpBurgName(g, s.Burg), name)
		case "tithe":
			s.N = 40*(i+1) + 20
			s.Text = fmt.Sprintf("Give %s to the coffers of %s.", g.cuFormat(s.N), name)
		case "rank":
			s.Text = "Rise a rank within " + name + "."
		}
		l.Steps = append(l.Steps, s)
	}
	return l
}

func cmpLadCounters(g *Game, l *cmpLad) (done, rank, max int) {
	if l.Sect {
		st := g.mr()
		return st.Done, st.Rank, 4
	}
	if m := g.OrderMembership(l.Ref); m != nil {
		return m.Done, m.Rank, 5
	}
	return 0, 0, 5
}

func cmpRankName(g *Game, l *cmpLad, rank int) string {
	if l.Sect {
		return g.mrSk().Ranks[clampi(rank, 0, 4)]
	}
	if o := g.OrderByID(l.Ref); o != nil && rank >= 1 {
		return o.Ranks[clampi(rank-1, 0, 4)]
	}
	return "the next rank"
}

func cmpStepStart(g *Game, l *cmpLad, s *cmpStep) {
	s.Started = true
	done, rank, max := cmpLadCounters(g, l)
	switch s.T {
	case "tasks":
		s.Base = done
	case "slay":
		s.Base = g.Kills
	case "rank":
		s.N = mini(rank+1, max)
		s.Text = "Rise to the rank of " + cmpRankName(g, l, s.N) + " within " + l.Name + "."
	}
	if s.T == "relic" {
		cmpEnterNow(g)
	}
}

// cmpEnterNow places the relic at once if the player is already in its dungeon.
func cmpEnterNow(g *Game) {
	if g.Cur != nil {
		cmpEnter(g, g.Cur)
	}
}

func cmpStepReady0(s *cmpStep) bool { return s.Ready }

func cmpStepReady(g *Game, l *cmpLad, s *cmpStep) {
	if s.Ready {
		return
	}
	s.Ready = true
	g.Msg(fmt.Sprintf("{{G|%s: ready to report to %s.}}", l.Title, l.Leader))
}

func cmpStepEval(g *Game, l *cmpLad, s *cmpStep) {
	if s.Ready {
		return
	}
	done, rank, _ := cmpLadCounters(g, l)
	switch s.T {
	case "audience":
		cmpStepReady(g, l, s)
	case "tasks":
		s.Have = done - s.Base
		if s.Have >= s.N {
			cmpStepReady(g, l, s)
		}
	case "slay":
		s.Have = g.Kills - s.Base
		if s.Have >= s.N {
			cmpStepReady(g, l, s)
		}
	case "relic":
		if cmpRelicItem(g, l, s) != nil {
			cmpStepReady(g, l, s)
		}
	case "tithe":
		if g.P.Gold >= s.N {
			cmpStepReady(g, l, s)
		}
	case "rank":
		if rank >= s.N {
			cmpStepReady(g, l, s)
		}
	}
}

func cmpRelicItem(g *Game, l *cmpLad, s *cmpStep) *Item {
	tag := fmt.Sprintf("cmp-relic:%s:%d", l.Key, l.Cur)
	for _, it := range g.P.Inv {
		if it.Text == tag {
			return it
		}
	}
	return nil
}

func cmpSyncLad(g *Game, l *cmpLad) {
	q := cmpQuestByID(g, l.QID)
	if q == nil {
		return
	}
	switch {
	case l.Done:
		q.Done, q.Closed = true, true
		q.Text = "You have taken every step of the oath."
	case l.Lapsed:
		q.Done, q.Closed = true, true
		q.Text = "You left the society; the oath lapsed."
	case l.Cur < len(l.Steps):
		s := l.Steps[l.Cur]
		t := fmt.Sprintf("Step %d of %d. %s", l.Cur+1, len(l.Steps), s.Text)
		if s.N > 0 && (s.T == "tasks" || s.T == "slay") {
			t += fmt.Sprintf(" (%d/%d)", mini(s.Have, s.N), s.N)
		}
		if s.Ready {
			t += " Report to " + l.Leader + "."
		}
		q.Text = t
		switch s.T {
		case "pilgrim":
			q.TP = cmpBurgP(g, s.Burg)
		case "relic":
			q.Target = s.Zone
		default:
			q.TP, q.Target = cmpBurgP(g, l.Burg), ""
		}
	}
}

func cmpLadTalk(g *Game, a *Actor, st *cmpState) []Option {
	var opts []Option
	for _, key := range cmpLadKeys(st) {
		l := st.Lads[key]
		if l.Done || l.Lapsed || l.Cur >= len(l.Steps) {
			continue
		}
		if l.Sect {
			if mrFlag(a, "mrlead:") != l.Ref {
				continue
			}
		} else if !a.Has("ord:" + l.Ref) {
			continue
		}
		s := l.Steps[l.Cur]
		cmpStepEval(g, l, s)
		if s.Ready {
			label := "{{G|Report that you have done as asked.}}"
			if s.T == "audience" {
				label = "{{G|Present yourself and ask what the society expects.}}"
			}
			opts = append(opts, Option{label, func() { g.cmpClaim(a, l) }})
		} else {
			opts = append(opts, Option{"{{c|Ask what the society expects of you next.}}", func() {
				extra := ""
				if s.N > 0 && (s.T == "tasks" || s.T == "slay") {
					extra = fmt.Sprintf(" (%d of %d done)", mini(s.Have, s.N), s.N)
				}
				g.talkMenu("\"" + s.Text + extra + "\"")
			}})
		}
	}
	return opts
}

func (g *Game) cmpClaim(a *Actor, l *cmpLad) {
	s := l.Steps[l.Cur]
	switch s.T {
	case "tithe":
		if g.P.Gold < s.N {
			g.talkMenu("\"Your purse is lighter than your word.\"")
			return
		}
		g.P.Gold -= s.N
	case "relic":
		it := cmpRelicItem(g, l, s)
		if it == nil {
			g.talkMenu("\"Where is it? Do not come with empty hands.\"")
			return
		}
		for i, x := range g.P.Inv {
			if x == it {
				g.P.Inv = append(g.P.Inv[:i], g.P.Inv[i+1:]...)
				break
			}
		}
	}
	step := l.Cur
	gold := s.Gold + g.P.Level*5
	notes := []string{"{{G|" + g.cuEarn(gold) + "}}"}
	g.gainXP(s.XP)
	g.AddRep(l.RepKey, s.Rep, " for your service")
	last := step == len(l.Steps)-1
	if l.Sect && !last && step > 0 {
		it := g.mrPill(clampi(step/2, 0, 2))
		g.give(it)
		notes = append(notes, "You receive "+it.Name+".")
	} else if !l.Sect && !last && step > 0 {
		g.give(cmpTreatise(g, l, step))
		notes = append(notes, "You are given a treatise to keep.")
	}
	l.Cur++
	if last {
		l.Done = true
		g.give(cmpTestament(g, l))
		if l.Sect {
			if sc := g.SectByID(l.Ref); sc != nil {
				if m := g.mrManual(sc.Art.Tech[2].ID); m != nil {
					g.give(m)
					notes = append(notes, "You are trusted with a secret manual.")
				}
			}
		}
		if g.Cur != nil {
			g.deed("campaign", fmt.Sprintf("The wanderer %s completed the oath of %s.", g.P.Name, l.Name), 0, 0)
		}
		lvSt(g).addNews(lvDay(g), l.Burg, "%s has risen high in %s; the old hands speak of the oath they took.", g.P.Name, l.Name)
	} else {
		s2 := l.Steps[l.Cur]
		cmpStepStart(g, l, s2)
		cmpStepEval(g, l, s2)
	}
	cmpSyncLad(g, l)
	line := cmpLadReport[step%len(cmpLadReport)]
	next := ""
	if !last {
		next = " Next: " + l.Steps[l.Cur].Text
	} else {
		next = " The oath is complete."
	}
	g.talkMenu(line + " " + strings.Join(notes, " ") + next)
}

func cmpTreatise(g *Game, l *cmpLad, step int) *Item {
	text := fmt.Sprintf("From the sayings of %s, %s of %s.\n\n%s\n\nA member who has done step %d of the oath may keep this.", l.Leader, strings.ToLower(l.Tit), l.Name, l.Creed, step+1)
	return &Item{ID: g.newID(), Kind: "book", Name: "treatise of " + l.Name, Glyph: '■', F: CCyan, Text: text, Value: 20 + 10*step, Weight: 1, Ident: true}
}

func cmpTestament(g *Game, l *cmpLad) *Item {
	text := fmt.Sprintf("The Testament of %s.\n\n%s\n\nIssued to %s, who took every step of the oath, by %s, %s.", l.Name, l.Creed, g.P.Name, l.Leader, strings.ToLower(l.Tit))
	return &Item{ID: g.newID(), Kind: "book", Name: "testament of " + l.Name, Glyph: '■', F: CYellow, Text: text, Value: 150, Weight: 1, Ident: true, Rarity: "rare"}
}

// ---------------------------------------------------------------- abilities and screen

func (g *Game) cmpAbilities() []Ability {
	if g.Cur == nil || g.P == nil || g.W == nil {
		return nil
	}
	st := cmpSt(g)
	for _, p := range st.Plots {
		if p.Status != "" {
			return []Ability{{ID: "cmp_plots", Name: "Plots and oaths", Target: "self"}}
		}
	}
	if len(st.Lads) > 0 {
		return []Ability{{ID: "cmp_plots", Name: "Plots and oaths", Target: "self"}}
	}
	return nil
}

func (g *Game) cmpAbUse(id string) {
	if id == "cmp_plots" {
		st := cmpSt(g)
		st.ui = cmpUI{}
		g.Mode = "cmp"
	}
}

type cmpEntry struct {
	plot *cmpPlot
	lad  *cmpLad
}

func cmpEntries(g *Game) []cmpEntry {
	st := cmpSt(g)
	var out []cmpEntry
	for _, p := range st.Plots {
		if p.Status != "" {
			out = append(out, cmpEntry{plot: p})
		}
	}
	for _, k := range cmpLadKeys(st) {
		out = append(out, cmpEntry{lad: st.Lads[k]})
	}
	return out
}

func cmpKeys(g *Game, k string) {
	st := cmpSt(g)
	ui := &st.ui
	es := cmpEntries(g)
	switch k {
	case "Escape":
		g.Mode = "main"
	case "ArrowUp":
		if len(es) > 0 {
			ui.sel, ui.off = (ui.sel+len(es)-1)%len(es), 0
		}
	case "ArrowDown":
		if len(es) > 0 {
			ui.sel, ui.off = (ui.sel+1)%len(es), 0
		}
	case "Tab", "ArrowRight":
		ui.tab, ui.off = (ui.tab+1)%3, 0
	case "ArrowLeft":
		ui.tab, ui.off = (ui.tab+2)%3, 0
	case "PageUp":
		ui.off = maxi(0, ui.off-8)
	case "PageDown":
		ui.off += 8
	case "r":
		if ui.sel < len(es) && es[ui.sel].plot != nil {
			p := es[ui.sel].plot
			if lp := lvGet(g, p.boss().Key); p.Status == "ready" && lp != nil && !lp.Alive {
				cmpResolve(g, p, "expose", nil)
			}
		}
	}
}

func (g *Game) cmpLines(e cmpEntry, tab int) []string {
	var l []string
	if e.plot != nil {
		p := e.plot
		switch tab {
		case 0:
			l = append(l, "{{Y|"+p.Title+"}}", "{{K|"+p.Hook+"}}", "")
			if p.Status == "done" {
				l = append(l, "{{G|Resolved: "+p.End+".}}", p.fill(cmpTplByID(p.Tpl).End[p.End]))
				break
			}
			for i, a := range p.Acts {
				switch {
				case i < p.Cur:
					l = append(l, fmt.Sprintf("{{G|[x] Act %d: %s}}", i+1, a.Title))
				case i == p.Cur && p.Status == "active":
					l = append(l, fmt.Sprintf("{{W|[>] Act %d: %s}}", i+1, a.Title), "    "+a.Brief)
					for _, c := range p.curClues() {
						l = append(l, "    {{c|- "+cmpHint(g, c)+"}}")
					}
				default:
					l = append(l, fmt.Sprintf("{{K|[ ] Act %d: ???}}", i+1))
				}
			}
			if p.Status == "ready" {
				l = append(l, "", fmt.Sprintf("{{R|Confront %s in %s.}}", p.boss().Name, cmpBurgName(g, p.boss().Burg)))
				if lp := lvGet(g, p.boss().Key); lp != nil && !lp.Alive {
					l = append(l, "{{K|They are dead. Press r to lay the proof before the authorities.}}")
				}
			}
			if p.Pending == "kill" {
				l = append(l, "{{R|The fight is on.}}")
			}
		case 1:
			any := false
			for _, c := range p.Clues {
				if c.Found {
					any = true
					l = append(l, fmt.Sprintf("{{G|Act %d}} {{c|%s}}: %s", c.Act+1, c.Name, c.Text), "")
				}
			}
			if !any {
				l = append(l, "{{K|No clues yet.}}")
			}
		default:
			for _, c := range p.Cast {
				state := "{{K|unknown}}"
				if lp := lvGet(g, c.Key); lp != nil {
					if !lp.Alive {
						state = "{{R|dead}}"
					} else if lp.Burg > 0 {
						state = "in " + cmpBurgName(g, lp.Burg)
					} else {
						state = "on the road"
					}
				} else {
					state = "at " + cmpBurgName(g, c.Burg)
				}
				l = append(l, fmt.Sprintf("{{Y|%s}}, %s. %s", c.Name, c.Desc, state))
			}
		}
		return l
	}
	d := e.lad
	l = append(l, "{{Y|"+d.Title+"}}", fmt.Sprintf("{{K|%s, %s of %s}}", d.Leader, strings.ToLower(d.Tit), d.Name), "")
	if d.Lapsed {
		l = append(l, "{{R|You are no longer a member. Progress is kept if you rejoin.}}", "")
	}
	for i, s := range d.Steps {
		switch {
		case i < d.Cur:
			l = append(l, "{{G|[x] "+s.Text+"}}")
		case i == d.Cur && !d.Done:
			extra := ""
			if s.Ready {
				extra = " {{G|(report to " + d.Leader + ")}}"
			} else if s.N > 0 && (s.T == "tasks" || s.T == "slay") {
				extra = fmt.Sprintf(" (%d/%d)", mini(s.Have, s.N), s.N)
			}
			l = append(l, "{{W|[>] "+s.Text+"}}"+extra, fmt.Sprintf("    {{K|Reward: about %s, %d xp, %d standing}}", g.cuFormat(s.Gold), s.XP, s.Rep))
		default:
			l = append(l, "{{K|[ ] ...}}")
		}
	}
	if d.Done {
		l = append(l, "", "{{G|The oath is complete.}}")
	}
	return l
}

func (g *Game) cmpDraw() {
	st := cmpSt(g)
	ui := &st.ui
	x, y, w, h := g.panel("Plots and oaths")
	s := g.sc
	es := cmpEntries(g)
	if len(es) == 0 {
		s.Text(x, y, "{{K|Nothing yet. Listen to the talk in the towns, and join an order or a sect.}}", CGray)
		s.Text(x, y+h-1, "{{K|Esc: close}}", CGray)
		return
	}
	ui.sel = clampi(ui.sel, 0, len(es)-1)
	lw := 36
	for i, e := range es {
		name, tag := "", ""
		if e.plot != nil {
			name = e.plot.Title
			tag = map[string]string{"active": "open", "ready": "ready", "done": "done"}[e.plot.Status]
		} else {
			name = e.lad.Title
			tag = fmt.Sprintf("%d/%d", mini(e.lad.Cur, len(e.lad.Steps)), len(e.lad.Steps))
			if e.lad.Lapsed {
				tag = "lapsed"
			}
		}
		col := "K"
		if i == ui.sel {
			col = "W"
		}
		if len(name) > lw-9 {
			name = name[:lw-9]
		}
		if y+i < y+h-2 {
			s.Text(x, y+i, fmt.Sprintf("{{%s|%-*s}}{{K|%s}}", col, lw-8, name, tag), CGray)
		}
	}
	dx := x + lw + 1
	dw := w - lw - 1
	e := es[ui.sel]
	if e.plot != nil {
		tabs := []string{"Story", "Clues", "Cast"}
		tx := dx
		for i, t := range tabs {
			col := "K"
			if i == ui.tab%3 {
				col = "W"
			}
			tx = s.Text(tx, y, "{{"+col+"|"+t+"}}  ", CGray)
		}
	}
	var lines []string
	for _, l := range g.cmpLines(e, ui.tab%3) {
		lines = append(lines, Wrap(l, dw)...)
	}
	room := h - 4
	if ui.off > maxi(0, len(lines)-room) {
		ui.off = maxi(0, len(lines)-room)
	}
	for i := 0; i < room && ui.off+i < len(lines); i++ {
		s.Text(dx, y+2+i, lines[ui.off+i], CGray)
	}
	s.Text(x, y+h-1, "{{K|Up/Down: choose · Tab: pages · PgUp/PgDn: scroll · Esc: close}}", CGray)
}
