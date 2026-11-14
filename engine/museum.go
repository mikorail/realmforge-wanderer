package engine

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// Museums (prefix hm): towns of tier 1 or more get a museum building (in the academy district when there is
// one, else the plaza/palace zone) with a curator (Role "curator"). Donate relics, fossils, finds, grave goods,
// collections, small curios and mounted heads: coin by value x1.5, reputation with the town's realm, fame, a
// plaque on the museum wall ("Donated by <name> in Y<year>"), wing counts and milestones (5/15/40 items), and
// a monthly list of wanted pieces that pay double. State g.ext("hml") (planet-local, in unLocalKeys).
// Zone hook uses the side rng seed:hm:zoneKey only.

type hmDon struct {
	Wing int    `json:"w"`
	Name string `json:"n"`
	Text string `json:"t"`
	N    int    `json:"c"`
	Year int    `json:"y"`
	Cell int    `json:"p"` // plaque tile index, -1 none
	Pay  int    `json:"v,omitempty"`
}

type hmWant struct {
	Kind  string `json:"k"` // fossil relic head
	Key   string `json:"i,omitempty"`
	Label string `json:"l"`
	Done  bool   `json:"d,omitempty"`
}

type hmMus struct {
	Burg    int          `json:"b"`
	Zone    string       `json:"z"`
	Name    string       `json:"n"`
	X       int          `json:"x"`
	Y       int          `json:"y"`
	W       int          `json:"w"`
	H       int          `json:"h"`
	Don     []*hmDon     `json:"d,omitempty"`
	Wants   []hmWant     `json:"wa,omitempty"`
	WantMon int          `json:"wm,omitempty"`
	Mile    map[int]bool `json:"m,omitempty"`
}

type hmLocal struct {
	Mus map[string]*hmMus `json:"m,omitempty"`
}

func hmL(g *Game) *hmLocal {
	l := g.ext(hmLKey, func() any { return &hmLocal{} }).(*hmLocal)
	if l.Mus == nil {
		l.Mus = map[string]*hmMus{}
	}
	return l
}

func hmCount(m *hmMus) int {
	n := 0
	for _, d := range m.Don {
		n += d.N
	}
	return n
}

func hmWingCount(m *hmMus, wing int) int {
	n := 0
	for _, d := range m.Don {
		if d.Wing == wing {
			n += d.N
		}
	}
	return n
}

func hmMuseumHere(g *Game) *hmMus {
	if g.Cur == nil || g.W == nil {
		return nil
	}
	for _, m := range hmL(g).Mus {
		if m.Zone == g.Cur.Key {
			return m
		}
	}
	return nil
}

// ---------------------------------------------------------------- building

func hmMuseumTarget(cp *cityPlan) [2]int {
	var ac [][2]int
	for k, v := range cp.districts {
		if v == "academy" {
			ac = append(ac, k)
		}
	}
	if len(ac) == 0 {
		return [2]int{1, 1}
	}
	sort.Slice(ac, func(i, j int) bool {
		if ac[i][1] != ac[j][1] {
			return ac[i][1] < ac[j][1]
		}
		return ac[i][0] < ac[j][0]
	})
	return ac[0]
}

func hmMuseumZone(g *Game, zn *Zone) {
	if g.W == nil || zn.Kind != "town" || zn.Z != 0 {
		return
	}
	bd := g.W.burgByID[zn.Burg]
	if bd == nil || !bd.Alive || bd.Layer != "surface" || zn.WX != bd.P%g.W.W || zn.WY != bd.P/g.W.W {
		return
	}
	cp := g.cityPlanFor(bd)
	if cp.tier < 1 {
		return
	}
	if t := hmMuseumTarget(cp); zn.ZX != t[0] || zn.ZY != t[1] {
		return
	}
	bk := strconv.Itoa(bd.ID)
	l := hmL(g)
	if l.Mus[bk] != nil {
		return
	}
	r := NewRng(g.Seed + ":hm:" + zn.Key)
	rc, ok := g.lot(zn, r, 11, 13, 7, 8)
	if !ok {
		rc, ok = g.lot(zn, r, 9, 10, 6, 6)
	}
	if !ok {
		return
	}
	x0, y0, w, h := rc[0], rc[1], rc[2], rc[3]
	for y := y0; y < y0+h; y++ {
		for x := x0; x < x0+w; x++ {
			t := TCarpet
			if y == y0 || y == y0+h-1 || x == x0 || x == x0+w-1 {
				t = TWall
			}
			zn.Set(x, y, t)
			delete(zn.Notes, y*MW+x)
		}
	}
	dx := x0 + w/2
	zn.Set(dx, y0+h-1, TDoor)
	name := fmt.Sprintf(Pick(r, []string{"%s Museum", "The %s Athenaeum", "%s Hall of Antiquities"}), bd.Name)
	if sx, sy := dx+2, y0+h; zn.At(sx, sy) == TCobble || zn.At(sx, sy) == TDirt {
		zn.Set(sx, sy, TInscription)
		zn.Notes[sy*MW+sx] = name + ". Open to all. The curator receives relics, fossils, grave goods, collections and mounted beasts, and pays for them."
	}
	cu := g.W.cultByID[bd.Culture]
	a := g.makeNPC("curator", bd, cu, r)
	a.Title, a.F, a.AI = "curator", CCyan, "still"
	a.Desc = "The curator of the " + name + "."
	a.X, a.Y = dx, y0+h/2
	a.HomeX, a.HomeY = a.X, a.Y
	zn.Actors = append(zn.Actors, a)
	l.Mus[bk] = &hmMus{Burg: bd.ID, Zone: zn.Key, Name: name, X: x0, Y: y0, W: w, H: h, Mile: map[int]bool{}}
}

// hmPlaqueCells: wall cells inside the museum, in the order plaques are hung.
func hmPlaqueCells(m *hmMus) [][2]int {
	var out [][2]int
	for x := m.X + 1; x <= m.X+m.W-2; x++ {
		out = append(out, [2]int{x, m.Y + 1})
	}
	for y := m.Y + 3; y <= m.Y+m.H-4; y++ {
		out = append(out, [2]int{m.X + 1, y}, [2]int{m.X + m.W - 2, y})
	}
	for x := m.X + 2; x <= m.X+m.W-3; x += 2 {
		if x != m.X+m.W/2 {
			out = append(out, [2]int{x, m.Y + 2})
		}
	}
	return out
}

// ---------------------------------------------------------------- wanted list

func hmMonth(g *Game) int { return g.Day() / SeasonDays }

func hmWantsFor(g *Game, m *hmMus) {
	mon := hmMonth(g)
	if m.WantMon == mon+1 && len(m.Wants) > 0 {
		return
	}
	m.WantMon = mon + 1
	m.Wants = nil
	r := NewRng(fmt.Sprintf("%s:hm:want:%d:%d", g.Seed, m.Burg, mon))
	var fos []*bsSp
	for _, s := range bsAll {
		if s.Extinct() && s.Era != "recent" {
			fos = append(fos, s)
		}
	}
	if len(fos) > 0 {
		s := Pick(r, fos)
		m.Wants = append(m.Wants, hmWant{Kind: "fossil", Key: s.ID, Label: "a fossil of " + bsArticle(s.Display())})
	}
	state := 0
	if bd := g.W.burgByID[m.Burg]; bd != nil {
		state = bd.State
	}
	var local, all []*ArtifactData
	for i := range g.W.Artifacts {
		a := &g.W.Artifacts[i]
		all = append(all, a)
		if a.P >= 0 && a.P < len(g.W.State) && g.W.State[a.P] == state {
			local = append(local, a)
		}
	}
	pool := local
	if len(pool) == 0 {
		pool = all
	}
	if len(pool) > 0 {
		a := Pick(r, pool)
		m.Wants = append(m.Wants, hmWant{Kind: "relic", Key: strconv.Itoa(a.ID), Label: "the relic " + a.Name})
	}
	m.Wants = append(m.Wants, hmWant{Kind: "head", Label: "a mounted beast's head"})
}

func hmWantMatch(w hmWant, it *Item) bool {
	if w.Done || it == nil {
		return false
	}
	switch w.Kind {
	case "fossil":
		s := bsFossilSpecies(it)
		return s != nil && s.ID == w.Key
	case "relic":
		return it.SimArt > 0 && strconv.Itoa(it.SimArt) == w.Key
	case "head":
		return it.Extra["bctroph"] > 0
	}
	return false
}

// ---------------------------------------------------------------- donating

type hmDonC struct {
	Title string
	Items []*Item
	Wing  int
	Pay   int
	Want  int
	Plaq  string
}

func hmWingOf(it *Item) (int, bool) {
	switch {
	case it.Kind == "fossil", it.Extra["bctroph"] > 0:
		return hmWNature, true
	case it.SimArt > 0, it.Extra["bu:find"] > 0, it.Extra["bu:nom"] > 0:
		return hmWAntiq, true
	case it.Kind == "clutter" && it.Extra["cl:i"] > 0:
		return hmWFolk, true
	case it.Kind == "clutter" && it.Extra["cl:n"] > 0 && it.Extra["cl:rot"] == 0:
		return hmWNature, true
	}
	return 0, false
}

func hmDonCands(g *Game, m *hmMus) (out []hmDonC, refused int) {
	hmWantsFor(g, m)
	held := clHeld(g)
	for _, s := range clSets {
		if clSetHeld(held, s) < len(s.Items) {
			continue
		}
		var its []*Item
		pay := 0
		for _, di := range s.Items {
			for _, it := range g.P.Inv {
				if it.Extra["cl:i"] == di+1 && it.Extra["cl:q"] == 0 && !uwIsStolen(it) {
					its = append(its, it)
					pay += it.Value
					break
				}
			}
		}
		if len(its) == len(s.Items) {
			out = append(out, hmDonC{Title: "the complete set of " + strings.ToLower(s.Title), Items: its, Wing: hmWFolk, Pay: maxi(1, pay*3/2), Want: -1,
				Plaq: "a collection of " + strings.ToLower(s.Title) + " (" + strconv.Itoa(len(its)) + " pieces)"})
		}
	}
	for _, it := range g.P.Inv {
		wing, ok := hmWingOf(it)
		if !ok || it.Quest > 0 || g.P.Equipped(it) != nil {
			continue
		}
		if it.Extra["bu:app"] > 0 || uwIsStolen(it) || it.Extra["bu:claim"] > 0 {
			refused++
			continue
		}
		want := -1
		for i, w := range m.Wants {
			if hmWantMatch(w, it) {
				want = i
			}
		}
		name := Plain(it.Display())
		out = append(out, hmDonC{Title: name, Items: []*Item{it}, Wing: wing, Pay: maxi(1, it.Value*3/2), Want: want, Plaq: name})
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Wing != out[j].Wing {
			return out[i].Wing < out[j].Wing
		}
		return out[i].Pay > out[j].Pay
	})
	if len(out) > 16 {
		out = out[:16]
	}
	return
}

func hmYear(g *Game) int { return g.W.Year + g.Day()/YearDays }

func (g *Game) hmDonateMenu(a *Actor, m *hmMus, note string) {
	cs, refused := hmDonCands(g, m)
	text := fmt.Sprintf("%s\n{{K}}%d items in the rolls of the %s.", note, hmCount(m), m.Name)
	text = strings.ReplaceAll(text, "{{K}}", "")
	if refused > 0 {
		text += " The curator will not take stolen goods, claimed grave goods or things not yet appraised."
	}
	g.Popup = Wrap(strings.TrimSpace(text), 100)
	g.Opts = nil
	g.Sel = 0
	for _, c := range cs {
		c := c
		extra := ""
		if c.Want >= 0 {
			extra = " {{G|WANTED: pays double}}"
		}
		g.Opts = append(g.Opts, Option{fmt.Sprintf("Donate %s (%d). {{K|[%s]}}%s", c.Title, c.Pay, hmWingName[c.Wing], extra), func() { g.hmDonate(a, m, c) }})
	}
	g.Opts = append(g.Opts, Option{"Never mind.", func() { g.talkMenu(g.greeting(a)) }})
}

func (g *Game) hmDonate(a *Actor, m *hmMus, c hmDonC) {
	bd := g.W.burgByID[m.Burg]
	pay := c.Pay
	want := c.Want >= 0 && c.Want < len(m.Wants) && !m.Wants[c.Want].Done
	if want {
		pay *= 2
	}
	var first *Item
	for _, it := range c.Items {
		o := hmSplit(g, it)
		if first == nil {
			first = o
		}
	}
	got := g.cuEarn(pay)
	st := hmSt(g)
	st.Gave += len(c.Items)
	n := len(c.Items)
	rep := clampi(2+pay/120, 2, 12)
	if want {
		rep += 8
		m.Wants[c.Want].Done = true
		g.gainXP(30)
		g.hmFirst("want", fmt.Sprintf("The wanderer %s answered the %s's call for %s.", g.P.Name, m.Name, m.Wants[c.Want].Label))
	}
	if bd != nil {
		g.AddRep(fmt.Sprintf("state:%d", bd.State), rep, " for a museum donation")
		g.fmFame(bd.State, 1)
	}
	g.gainXP(5 + pay/40)
	d := &hmDon{Wing: c.Wing, Name: c.Plaq, N: n, Year: hmYear(g), Cell: -1, Pay: pay}
	d.Text = fmt.Sprintf(hmPlaqueFmt[len(m.Don)%len(hmPlaqueFmt)], g.P.Name, d.Year, c.Plaq)
	cells := hmPlaqueCells(m)
	if zn := g.Zones[m.Zone]; zn != nil {
		used := map[int]bool{}
		for _, o := range m.Don {
			if o.Cell >= 0 {
				used[o.Cell] = true
			}
		}
		for _, cl := range cells {
			i := cl[1]*MW + cl[0]
			if used[i] || zn.At(cl[0], cl[1]) != TCarpet || zn.ActorAt(cl[0], cl[1]) != nil {
				continue
			}
			zn.Set(cl[0], cl[1], TInscription)
			zn.Notes[i] = d.Text
			d.Cell = i
			break
		}
	}
	old := hmCount(m)
	m.Don = append(m.Don, d)
	g.Msg(fmt.Sprintf("{{G|The museum pays %s for %s.}}", got, c.Title))
	g.hmFirst("don", fmt.Sprintf("The wanderer %s made a first gift to a museum.", g.P.Name))
	switch c.Wing {
	case hmWNature:
		g.hmFirst("don0", fmt.Sprintf("The wanderer %s gave the natural history wing of the %s a specimen.", g.P.Name, m.Name))
	case hmWAntiq:
		if first != nil && first.SimArt > 0 {
			g.deed("relic", fmt.Sprintf("The wanderer %s gave the relic %s to the %s.", g.P.Name, first.Name, m.Name), 0, first.SimArt)
		}
		g.hmFirst("don1", fmt.Sprintf("The wanderer %s gave the antiquities of the %s a new piece.", g.P.Name, m.Name))
	case hmWFolk:
		g.hmFirst("don2", fmt.Sprintf("The wanderer %s gave the %s a piece of folk life.", g.P.Name, m.Name))
	}
	g.hmMilestones(m, old, hmCount(m))
	g.hmDonateMenu(a, m, fmt.Sprintf("\"Gratefully received.\" %s hangs the plaque: %s", a.Name, d.Text))
}

func (g *Game) hmMilestones(m *hmMus, old, now int) {
	bd := g.W.burgByID[m.Burg]
	if m.Mile == nil {
		m.Mile = map[int]bool{}
	}
	for i, n := range hmMileN {
		if old >= n || now < n || m.Mile[n] {
			continue
		}
		m.Mile[n] = true
		st := hmSt(g)
		title := fmt.Sprintf(hmTitleNames[i], m.Name)
		switch i {
		case 0:
			st.Titles = append(st.Titles, title)
			g.Msg(fmt.Sprintf("{{Y|%d gifts: you are named %s.}}", n, title))
			g.hmLecture(m, bd)
		case 1:
			it := g.hmReward(m, bd)
			g.give(it)
			g.Msg(fmt.Sprintf("{{Y|%d gifts: the curator presses a %s into your hand.}}", n, it.Name))
		default:
			st.Titles = append(st.Titles, title)
			g.Msg(fmt.Sprintf("{{Y|%d gifts: you are named %s. Merchants of the town give you better prices.}}", n, title))
		}
		g.deed("event", fmt.Sprintf("The wanderer %s reached %d gifts to the %s and was named %s.", g.P.Name, n, m.Name, title), 0, 0)
		g.gainXP(20 * (i + 1))
	}
}

func (g *Game) hmReward(m *hmMus, bd *BurgData) *Item {
	r := NewRng(fmt.Sprintf("%s:hm:reward:%d", g.Seed, m.Burg))
	nm := Pick(r, hmRewardNames)
	return &Item{ID: g.newID(), Kind: "artifact", Slot: "float", Name: nm + " of the " + m.Name, Glyph: 'o', F: CCyan, Tier: 2, Value: 220, Weight: 1,
		Ident: true, Rarity: "unique", Bonus: map[int]int{INT: 1},
		Lore: "A small gift from the curators: it sharpens the eye for old things."}
}

// hmLecture: a free lecture; the curator talks about a relic and where it last was (a secret).
func (g *Game) hmLecture(m *hmMus, bd *BurgData) {
	r := NewRng(fmt.Sprintf("%s:hm:lecture:%d", g.Seed, m.Burg))
	var text string
	p := -1
	if n := len(g.W.Artifacts); n > 0 {
		a := g.W.Artifacts[r.Int(0, n-1)]
		text = fmt.Sprintf("At the %s the curator lectured on the relic %s, forged in the year %d.", m.Name, a.Name, a.Year)
		if a.Holder != "" {
			text += " It was last recorded with " + a.Holder + "."
		}
		if a.P >= 0 && a.P < len(g.W.Cell) {
			text += " The trail leads toward " + g.W.RegionName(a.P) + "."
			p = a.P
		}
	} else {
		text = "At the " + m.Name + " the curator lectured on how old bones are found in strata and badlands."
	}
	if g.addSecret(text, "lecture", p, g.W.Year) {
		g.Msg("{{c|The curator gives you a private lecture. A new secret is noted.}}")
	}
}

// ---------------------------------------------------------------- talk

func hmCuratorTalk(g *Game, a *Actor) []Option {
	m := hmMuseumHere(g)
	if m == nil {
		return nil
	}
	return []Option{
		{"Donate to the museum.", func() {
			g.Talk = a
			g.hmDonateMenu(a, m, hmCuratorGreet[int(HashStr(m.Name+g.P.Name)%uint64(len(hmCuratorGreet)))])
		}},
		{"Ask to see the museum's rolls and plaques.", func() { g.hmOpenMuseum(m, true) }},
		{"Ask what the museum is seeking.", func() {
			hmWantsFor(g, m)
			var b strings.Builder
			b.WriteString("\"This month we seek:\"\n")
			for _, w := range m.Wants {
				if w.Done {
					fmt.Fprintf(&b, "\n{{K|%s (found)}}", w.Label)
				} else {
					fmt.Fprintf(&b, "\n{{Y|%s}}", w.Label)
				}
			}
			b.WriteString("\n\n\"Bring one of these and the museum pays double.\"")
			g.talkMenu(b.String())
		}},
	}
}

// ---------------------------------------------------------------- screen

func (g *Game) hmOpenMuseum(m *hmMus, fromTalk bool) {
	st := hmSt(g)
	st.mus, st.fromTalk = m, fromTalk
	hmWantsFor(g, m)
	g.Mode = "hmm"
	g.Tab, g.Sel = 0, 0
}

func hmmClose(g *Game) {
	st := hmSt(g)
	if st.fromTalk && g.Talk != nil {
		st.fromTalk = false
		g.Mode = "talk"
		g.talkMenu(g.greeting(g.Talk))
		return
	}
	g.Mode = "main"
}

func hmmKeys(g *Game, k string) {
	switch k {
	case "Escape":
		hmmClose(g)
	case "Tab", "ArrowRight":
		g.Tab = (g.Tab + 1) % 4
		g.Sel = 0
	case "ArrowLeft":
		g.Tab = (g.Tab + 3) % 4
		g.Sel = 0
	case "ArrowUp":
		g.Sel = maxi(0, g.Sel-1)
	case "ArrowDown":
		g.Sel++
	case "PageUp":
		g.Sel = maxi(0, g.Sel-10)
	case "PageDown":
		g.Sel += 10
	}
}

func hmmDraw(g *Game) {
	m := hmSt(g).mus
	if m == nil {
		return
	}
	s := g.sc
	x, y, w, ph := g.panel(m.Name)
	tx := x
	for i := 0; i < 4; i++ {
		label := "Wanted"
		if i < 3 {
			label = fmt.Sprintf("%s (%d)", hmWingName[i], hmWingCount(m, i))
		}
		c := "K"
		if g.Tab == i {
			c = "Y"
		}
		tx = s.TextB(tx, y, fmt.Sprintf("{{%s|[ %s ]}} ", c, label), CGray, CBgDark, x+w)
	}
	var lines []string
	if g.Tab < 3 {
		lines = append(lines, "{{K|"+hmWingName[g.Tab]+": "+hmWingDesc[g.Tab]+"}}", "")
		for _, d := range m.Don {
			if d.Wing != g.Tab {
				continue
			}
			mark := "{{K|plaque unread}}"
			if d.Cell >= 0 && g.Read[m.Zone+fmt.Sprint(d.Cell)] {
				mark = "{{G|plaque read}}"
			} else if d.Cell < 0 {
				mark = "{{K|no plaque (wall full)}}"
			}
			lines = append(lines, fmt.Sprintf("{{Y|Y%d}} %s {{K|x%d}} %s", d.Year, d.Name, d.N, mark))
			lines = append(lines, "   {{K|"+d.Text+"}}")
		}
		if len(lines) == 2 {
			lines = append(lines, "Nothing in this wing yet.")
		}
	} else {
		lines = append(lines, "{{Y|This month the curator seeks}} {{K|(pays double)}}", "")
		for _, wn := range m.Wants {
			if wn.Done {
				lines = append(lines, "{{K|"+wn.Label+" (found)}}")
			} else {
				lines = append(lines, "{{W|"+wn.Label+"}}")
			}
		}
		lines = append(lines, "", fmt.Sprintf("{{Y|Gifts: %d}}  Milestones: 5 (title, a lecture), 15 (a unique gift), 40 (title, merchants' discount).", hmCount(m)))
		for _, n := range hmMileN {
			if m.Mile[n] {
				lines = append(lines, fmt.Sprintf("{{G|%d reached.}}", n))
			}
		}
		for _, t := range hmSt(g).Titles {
			lines = append(lines, "{{c|Title: "+t+"}}")
		}
	}
	rows := ph - 5
	if g.Sel > maxi(0, len(lines)-rows) {
		g.Sel = maxi(0, len(lines)-rows)
	}
	for i := g.Sel; i < len(lines) && i-g.Sel < rows; i++ {
		s.TextB(x, y+2+i-g.Sel, lines[i], CGray, CBgDark, x+w)
	}
	s.TextB(x, y+ph-2, "{{K|Tab wing  Up/Down scroll  Esc leave}}", CGray, CBgDark, x+w)
}
