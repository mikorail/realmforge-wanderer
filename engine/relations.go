package engine

import (
	"fmt"
	"math"
	"sort"
	"strings"
)

// Relationships and consequences screen (prefix rl). Mode `rls`, opened from the palette ("Relationships
// and consequences"); no main key. Read-only: it reads the life simulation (lives.go, knowledge.go),
// favours and quests (npcquests.go), crime (crime.go), karma and queued crossroads cards (karma.go,
// crossroads.go), the town board (townvoice.go), the nemesis (nemesis.go), companions, banks and realm
// standing, and never changes any of it. Nothing is saved.
//
//   tab 0 Web          up to 12 people nearest to you as a small graph (ASCII link lines for spouse, kin, friends,
//                      rivals and bosses, dotted spokes to you), and the list of everyone you know grouped by
//                      how they stand toward you, with bars
//   tab 1 People       the same list with a detail pane: their opinion of you and why (memories), debts and
//                      favours, grudges (crime, war, realm standing), fears, their ties
//   tab 2 Consequences karma ledger, queued crossroads consequences, towns whose fate changed, hunters,
//                      allies and enemies made, loans
//
// Keys: Tab / Left / Right or 1 2 3 switch tabs, Up/Down choose (scroll on tab 2), PageUp/PageDown scroll, Esc closes.

type rlState struct {
	Tab    int `json:"-"`
	Sel    int `json:"-"`
	Scroll int `json:"-"`
	DScr   int `json:"-"`
}

func (g *Game) rl() *rlState {
	return g.ext("rl", func() any { return &rlState{} }).(*rlState)
}

func init() {
	modeKeys["rls"] = rlKeys
	modeDraws["rls"] = func(g *Game) { g.rlDraw() }
	paEntries = append(paEntries, paEntry{Name: "Relationships and consequences", Keywords: "people web graph love hate owe fear debts grudges karma allies enemies ledger what have i done", Cat: "Screen",
		Func: func(g *Game) {
			st := g.rl()
			st.Tab, st.Sel, st.Scroll, st.DScr = 0, 0, 0, 0
			g.Mode = "rls"
		}})
}

// ---------------------------------------------------------------- people

type rlNode struct {
	P    *lvPerson
	Dist int
	Tags string // markup letters: L loves you, H hates you, F fears, $ debt, ! grudge
}

func rlHasMem(p *lvPerson, kinds ...string) bool {
	for _, m := range p.Mem {
		for _, k := range kinds {
			if m.Kind == k {
				return true
			}
		}
	}
	return false
}

func rlQuestsOwed(g *Game, p *lvPerson) []*nqQuest {
	var out []*nqQuest
	for _, q := range nqSt(g).Q {
		if q.Stage == 1 && q.Giver == p.Key {
			out = append(out, q)
		}
	}
	return out
}

func rlDebtful(g *Game, p *lvPerson) bool {
	return len(rlQuestsOwed(g, p)) > 0 || nqSt(g).Favour[p.Key] || rlHasMem(p, "theft")
}

func rlTags(g *Game, p *lvPerson) string {
	t := ""
	if p.Opinion >= 60 {
		t += "{{G|L}}"
	}
	if p.Opinion <= -50 {
		t += "{{R|H}}"
	}
	if p.Opinion < 0 && (rlHasMem(p, "attack", "kill") || p.Trait[4] >= 70 && p.Opinion <= -20) {
		t += "{{M|F}}"
	}
	if rlDebtful(g, p) {
		t += "{{W|$}}"
	}
	if p.Opinion <= -25 || rlHasMem(p, "theft", "attack", "kill") && p.Opinion < 10 {
		t += "{{r|!}}"
	}
	return t
}

func rlPeople(g *Game) []*rlNode {
	pp, _ := lsPlayerLoc(g)
	var out []*rlNode
	for _, p := range lvSt(g).People {
		if !p.Met && p.Opinion == 0 && len(p.Mem) == 0 {
			continue
		}
		n := &rlNode{P: p, Dist: 999}
		b := nonZero(p.Burg, p.Home)
		if bd := g.W.burgByID[b]; bd != nil && pp >= 0 {
			n.Dist = wsCheb(g.W, bd.P, pp)
		}
		n.Tags = rlTags(g, p)
		out = append(out, n)
	}
	sort.Slice(out, func(i, j int) bool {
		a, b := out[i].P, out[j].P
		if a.Alive != b.Alive {
			return a.Alive
		}
		if a.Opinion != b.Opinion {
			return a.Opinion > b.Opinion
		}
		return a.Key < b.Key
	})
	return out
}

func rlStanceHdr(op int) string {
	switch {
	case op >= 60:
		return "Devoted and fond"
	case op >= 20:
		return "Friendly"
	case op > -20:
		return "Neutral"
	case op > -60:
		return "Wary or cold"
	}
	return "Hostile"
}

func rlBar(op int) string {
	n := abs(op) * 5 / 100
	if op != 0 && n == 0 {
		n = 1
	}
	n = mini(n, 5)
	if op < 0 {
		return "{{K|" + strings.Repeat(".", 5-n) + "}}{{R|" + strings.Repeat("#", n) + "}}{{K||....." + "}}"
	}
	return "{{K|.....|}}{{G|" + strings.Repeat("#", n) + "}}{{K|" + strings.Repeat(".", 5-n) + "}}"
}

func rlOpCol(op int) byte {
	switch {
	case op >= 20:
		return CGreen
	case op <= -20:
		return CRed
	}
	return CGray
}

func rlTrunc(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n-1]) + "."
}

type rlRow struct {
	hdr string
	idx int
}

func rlRows(nodes []*rlNode) []rlRow {
	var rows []rlRow
	last := ""
	for i, n := range nodes {
		h := rlStanceHdr(n.P.Opinion)
		if !n.P.Alive {
			h = "Dead"
		}
		if h != last {
			rows = append(rows, rlRow{hdr: h, idx: -1})
			last = h
		}
		rows = append(rows, rlRow{idx: i})
	}
	return rows
}

func rlListLine(n *rlNode, nameW int, sel bool) string {
	p := n.P
	name := rlTrunc(p.Name, nameW)
	if !p.Alive {
		name = "{{K|" + name + "}}"
	} else if sel {
		name = "{{W|" + name + "}}"
	}
	pad := strings.Repeat(" ", maxi(0, nameW-len([]rune(Plain(name)))))
	return fmt.Sprintf("%s%s %s {{%s|%+4d}} %s", name, pad, rlBar(p.Opinion), string(rlColLetter(p.Opinion)), p.Opinion, n.Tags)
}

func rlColLetter(op int) byte {
	switch {
	case op >= 20:
		return 'G'
	case op <= -20:
		return 'R'
	}
	return 'y'
}

// ---------------------------------------------------------------- graph

type rlGrid struct {
	w, h int
	g    []rune
	f    []byte
	b    []byte
}

func newRlGrid(w, h int) *rlGrid {
	gr := &rlGrid{w: w, h: h, g: make([]rune, w*h), f: make([]byte, w*h), b: make([]byte, w*h)}
	for i := range gr.g {
		gr.g[i], gr.f[i], gr.b[i] = ' ', CGray, CBgDark
	}
	return gr
}

func (gr *rlGrid) put(x, y int, r rune, f, b byte) {
	if x < 0 || y < 0 || x >= gr.w || y >= gr.h {
		return
	}
	i := y*gr.w + x
	gr.g[i], gr.f[i] = r, f
	if b != 255 {
		gr.b[i] = b
	}
}

func (gr *rlGrid) text(x, y int, s string, f, b byte) {
	for _, r := range s {
		gr.put(x, y, r, f, b)
		x++
	}
}

// line draws a Bresenham line; ch picks the glyph from the step direction, or dot is used for spokes.
func (gr *rlGrid) line(x0, y0, x1, y1 int, f byte, dot bool) {
	dx, dy := abs(x1-x0), -abs(y1-y0)
	sx, sy := 1, 1
	if x0 > x1 {
		sx = -1
	}
	if y0 > y1 {
		sy = -1
	}
	e := dx + dy
	for n := 0; n < 400; n++ {
		r := '.'
		if !dot {
			switch {
			case y0 == y1 && x0 == x1:
				r = '+'
			case sy == 0 || dx > -dy*3:
				r = '-'
			case dx*3 < -dy:
				r = '|'
			case sx*sy > 0:
				r = '\\'
			default:
				r = '/'
			}
		}
		gr.put(x0, y0, r, f, 255)
		if x0 == x1 && y0 == y1 {
			break
		}
		e2 := 2 * e
		if e2 >= dy {
			e += dy
			x0 += sx
		}
		if e2 <= dx {
			e += dx
			y0 += sy
		}
	}
}

// rlNear picks up to 12 living people nearest to you (the selected one always included).
func rlNear(nodes []*rlNode, sel int) []*rlNode {
	var al []*rlNode
	for _, n := range nodes {
		if n.P.Alive {
			al = append(al, n)
		}
	}
	sort.SliceStable(al, func(i, j int) bool {
		if al[i].Dist != al[j].Dist {
			return al[i].Dist < al[j].Dist
		}
		return abs(al[i].P.Opinion) > abs(al[j].P.Opinion)
	})
	if len(al) > 12 {
		al = al[:12]
		if sel >= 0 && sel < len(nodes) && nodes[sel].P.Alive {
			in := false
			for _, n := range al {
				if n == nodes[sel] {
					in = true
				}
			}
			if !in {
				al[11] = nodes[sel]
			}
		}
	}
	return al
}

func (g *Game) rlGraph(x, y, w, h int, nodes []*rlNode, sel int) {
	s := g.sc
	gr := newRlGrid(w, h)
	near := rlNear(nodes, sel)
	n := len(near)
	cx, cy := w/2, h/2
	rx, ry := float64(w/2-7), float64(h/2-1)
	px := make([]int, n)
	py := make([]int, n)
	for i := range near {
		a := 2*math.Pi*float64(i)/float64(maxi(n, 1)) - math.Pi/2
		px[i] = cx + int(math.Round(rx*math.Cos(a)))
		py[i] = cy + int(math.Round(ry*math.Sin(a)))
	}
	// spokes to you, then links between people
	for i, nd := range near {
		col := CDarkGray
		if nd.P.Opinion >= 20 {
			col = CDarkGreen
		} else if nd.P.Opinion <= -20 {
			col = CDarkRed
		}
		gr.line(cx, cy, px[i], py[i], col, true)
	}
	idx := map[string]int{}
	for i, nd := range near {
		idx[nd.P.Key] = i
	}
	link := func(a, b int, col byte) {
		if a != b {
			gr.line(px[a], py[a], px[b], py[b], col, false)
		}
	}
	for i, nd := range near {
		p := nd.P
		if j, ok := idx[p.Spouse]; ok && p.Spouse != "" {
			link(i, j, CYellow)
		}
		for _, k := range p.Kin {
			if j, ok := idx[k]; ok && j > i {
				link(i, j, CDarkCyan)
			}
		}
		for _, k := range p.Friends {
			if j, ok := idx[k]; ok && j > i {
				link(i, j, CGreen)
			}
		}
		for _, k := range p.Rivals {
			if j, ok := idx[k]; ok && j > i {
				link(i, j, CRed)
			}
		}
		if j, ok := idx[p.Boss]; ok && p.Boss != "" {
			link(i, j, CMagenta)
		}
	}
	gr.text(cx-3, cy, "( YOU )", CWhite, CBgSel)
	for i, nd := range near {
		lab := "[" + rlTrunc(givenOf(nd.P.Name), 9) + "]"
		x0 := clampi(px[i]-len([]rune(lab))/2, 0, w-len([]rune(lab)))
		bg := CBgDark
		if sel >= 0 && sel < len(nodes) && nodes[sel] == nd {
			bg = CBgSel
		}
		gr.text(x0, py[i], lab, rlOpCol(nd.P.Opinion), bg)
	}
	for yy := 0; yy < h; yy++ {
		for xx := 0; xx < w; xx++ {
			i := yy*w + xx
			s.Put(x+xx, y+yy, gr.g[i], gr.f[i], gr.b[i])
		}
	}
}

// ---------------------------------------------------------------- detail

func rlDays(g *Game, day int) string {
	d := lvDay(g) - day
	switch {
	case d <= 0:
		return "today"
	case d == 1:
		return "yesterday"
	case d < 60:
		return fmt.Sprintf("%d days ago", d)
	}
	return fmt.Sprintf("%d months ago", d/28)
}

func rlDetail(g *Game, n *rlNode) []string {
	p := n.P
	var L []string
	add := func(s string) { L = append(L, s) }
	home := ct2Town(g, p.Home)
	now := home
	if p.Burg > 0 {
		now = ct2Town(g, p.Burg)
	}
	add("{{W|" + p.Name + "}} {{K|" + strings.ToLower(nonEmpty(p.Job, p.Role)) + ", " + p.Species + "}}")
	if p.Alive {
		where := "in " + now
		if p.Burg == 0 {
			where = "on the road"
		}
		add(fmt.Sprintf("{{K|Age %d, %s. Home: %s.}}", ct2Age(g, p), where, home))
	} else {
		add(fmt.Sprintf("{{K|Dead in year %d. %s}}", p.Died, titleFirst(p.Cause)))
	}
	add(fmt.Sprintf("Toward you: {{%s|%s}} %s {{y|%+d}}", string(rlColLetter(p.Opinion)), lvOpinionWord(p), rlBar(p.Opinion), p.Opinion))
	if p.Alive {
		add("{{K|Mood: " + lvMoodWord(p) + "}}")
	}
	add("")
	add("{{Y|Why they feel this way}}")
	mems := append([]lvMemory(nil), p.Mem...)
	sort.SliceStable(mems, func(i, j int) bool { return abs(mems[i].Weight) > abs(mems[j].Weight) })
	if len(mems) == 0 {
		add("{{K|  They have no particular memory of you.}}")
	}
	for i, m := range mems {
		if i >= 6 {
			break
		}
		c := "G"
		if m.Weight < 0 {
			c = "R"
		}
		add(fmt.Sprintf("  {{%s|%+d}} %s {{K|(%s)}}", c, m.Weight, strings.TrimSuffix(m.Text, "."), rlDays(g, m.Day)))
	}
	// debts and favours
	var debts []string
	for _, q := range rlQuestsOwed(g, p) {
		debts = append(debts, fmt.Sprintf("{{W|You owe them:}} %s, due in %d days.", q.Title, maxi(0, q.Due-lvDay(g))))
	}
	if nqSt(g).Favour[p.Key] {
		debts = append(debts, "{{G|They owe you:}} a favour, to ask any time.")
	}
	for _, m := range p.Mem {
		switch {
		case m.Kind == "theft":
			debts = append(debts, "{{W|You owe them:}} "+strings.TrimSuffix(m.Text, ".")+".")
		case (m.Kind == "gift" || m.Kind == "help" || m.Kind == "saved") && m.Weight >= 8:
			debts = append(debts, "{{G|They owe you:}} "+strings.TrimSuffix(m.Text, ".")+".")
		}
	}
	if sp := g.fm().Spouse; sp != nil && sp.Name == p.Name {
		debts = append(debts, "{{M|They are your spouse.}}")
	}
	for _, k := range g.fm().Kids {
		if k.Name == p.Name {
			debts = append(debts, "{{M|They are your child.}}")
		}
	}
	add("")
	add("{{Y|Debts and favours}}")
	if len(debts) == 0 {
		add("{{K|  None between you.}}")
	}
	for i, d := range debts {
		if i >= 6 {
			break
		}
		add("  " + d)
	}
	// grudges
	var gr []string
	for _, m := range p.Mem {
		if m.Weight < 0 && (m.Kind == "attack" || m.Kind == "kill" || m.Kind == "insult" || m.Kind == "witnessed") {
			gr = append(gr, strings.TrimSuffix(m.Text, ".")+".")
		}
	}
	if p.State > 0 {
		sk := fmt.Sprintf("state:%d", p.State)
		if rep := g.Rep[sk]; rep <= -50 {
			gr = append(gr, fmt.Sprintf("Their realm, %s, holds you in low standing (%d).", g.realmName(p.State), rep))
		}
		if b := g.uwBounty(p.State); b > 0 {
			gr = append(gr, fmt.Sprintf("%s has a bounty of %d on you.", g.realmName(p.State), b))
		}
		if h := g.W.stateByID[g.HomeState]; h != nil && g.HomeState > 0 && g.HomeState != p.State {
			for _, wr := range h.Wars {
				if wr == p.State {
					gr = append(gr, "Their realm is at war with yours.")
				}
			}
		}
	}
	if nem := nxSt(g).Nem; nem != nil && nem.Name == p.Name && nxActive(nem) {
		gr = append(gr, "They hunt you: "+strings.TrimSuffix(nem.Reason, ".")+".")
	}
	add("")
	add("{{Y|Grudges}}")
	if len(gr) == 0 {
		add("{{K|  None.}}")
	}
	for i, d := range gr {
		if i >= 6 {
			break
		}
		add("  {{R|" + d + "}}")
	}
	// fears
	if p.Fear != "" || strings.Contains(n.Tags, "F") {
		add("")
		add("{{Y|Fears}}")
		if p.Fear != "" {
			add("  " + strings.TrimSuffix(p.Fear, ".") + ".")
		}
		if strings.Contains(n.Tags, "F") {
			add("  {{M|They are afraid of you.}}")
		}
	}
	// ties
	ls := lvSt(g)
	tie := func(label string, keys ...string) {
		var names []string
		for _, k := range keys {
			if q := ls.People[k]; q != nil && k != "" {
				names = append(names, fmt.Sprintf("%s (%s)", q.Name, lvOpinionWord(q)))
			}
		}
		if len(names) > 0 {
			add("  " + label + " " + strings.Join(names, ", "))
		}
	}
	add("")
	add("{{Y|Ties, and what they think of you}}")
	tie("{{Y|Spouse:}}", p.Spouse)
	tie("{{c|Kin:}}", p.Kin...)
	tie("{{G|Friends:}}", p.Friends...)
	tie("{{R|Rivals:}}", p.Rivals...)
	tie("{{M|Patron:}}", p.Boss)
	if len(p.Life) > 0 {
		add("")
		add("{{Y|Their life, lately}}")
		for i := maxi(0, len(p.Life)-3); i < len(p.Life); i++ {
			add(fmt.Sprintf("  {{K|Year %d:}} %s", p.Life[i].Y, strings.TrimSuffix(p.Life[i].Text, ".")))
		}
	}
	return L
}

// ---------------------------------------------------------------- consequences

func rlAxisBar(v int) string {
	pos := clampi((v+100)*24/200, 0, 24)
	bar := []rune(strings.Repeat("-", 25))
	bar[12] = '|'
	lo, hi := 12, pos
	if pos < 12 {
		lo, hi = pos, 12
	}
	for j := lo; j <= hi; j++ {
		if j != 12 {
			bar[j] = '='
		}
	}
	return string(bar)
}

func rlConsequences(g *Game) []string {
	var L []string
	add := func(s string) { L = append(L, s) }
	ks := crKs(g)
	add("{{Y|Karma ledger}}  They call you {{M|" + g.crEpithet() + "}}.")
	for i := 0; i < 5; i++ {
		v := ks.Ax[i]
		c := "G"
		if v < 0 {
			c = "R"
		}
		add(fmt.Sprintf("  {{G|%-10s}} %s {{R|%-10s}} {{%s|%+d}}", crAxisNames[i][0], rlAxisBar(v), crAxisNames[i][1], c, v))
	}
	add(fmt.Sprintf("  Vows sworn %d, kept {{G|%d}}, broken {{R|%d}}.", len(ks.Vows), ks.Kept, ks.Broke))
	if n := len(ks.Log); n > 0 {
		add("  {{K|Recent weighing:}}")
		for i := maxi(0, n-6); i < n; i++ {
			add("   {{K|" + rlDays(g, ks.Log[i].Day) + "}} " + ks.Log[i].Text)
		}
	}
	add("")
	add("{{Y|Consequences on their way}}")
	pend := crSt(g).Pend
	if len(pend) == 0 {
		add("{{K|  Nothing is queued. The world has not caught up with you, or has already.}}")
	}
	for _, pd := range pend {
		title := pd.ID
		if t := crTplByID(pd.ID); t != nil {
			title = crFill(t.Title, pd.Ctx)
		}
		in := pd.Due - g.Day()
		when := fmt.Sprintf("in about %d days", maxi(in, 0))
		if in <= 0 {
			when = "any day now"
		}
		add(fmt.Sprintf("  {{W|%s}} {{K|%s}}", title, when))
	}
	add("")
	add("{{Y|Towns whose fate changed}}")
	rb := g.rb()
	var ids []int
	for id := range rb.Names {
		ids = append(ids, id)
	}
	for id, n := range rb.Kills {
		if n > 0 && rb.Names[id] == nil {
			ids = append(ids, id)
		}
	}
	sort.Ints(ids)
	if len(ids) == 0 {
		add("{{K|  No town has changed because of you yet.}}")
	}
	for _, id := range ids {
		bd := g.W.burgByID[id]
		if bd == nil {
			continue
		}
		line := "  {{Y|" + bd.Name + "}}"
		if n := rb.Names[id]; n != nil {
			if n.Mood > 0 {
				line += " {{G|saved}}"
			} else if n.Mood < 0 {
				line += " {{R|ruined}}"
			}
			if n.Inn != "" {
				line += fmt.Sprintf(": the inn is now %s", n.Inn)
				if n.InnOld != "" {
					line += " (once " + n.InnOld + ")"
				}
			}
		}
		if k := rb.Kills[id]; k > 0 {
			line += fmt.Sprintf(" {{R|%d townsfolk died by your hand}}", k)
		}
		add(line)
	}
	add("")
	add("{{Y|Hunters and bounties}}")
	found := false
	if nem := nxSt(g).Nem; nem != nil {
		found = true
		add(fmt.Sprintf("  {{R|%s}} %s: %s {{K|(%s)}}", nem.Name, nem.Epithet, strings.TrimSuffix(nem.Reason, "."), nem.Status))
	}
	var bst []int
	for s, b := range g.uwSt().Bounty {
		if b > 0 {
			bst = append(bst, s)
		}
	}
	sort.Ints(bst)
	for _, s := range bst {
		found = true
		add(fmt.Sprintf("  {{R|%d %s}} bounty in %s", g.uwBounty(s), g.Sk.Currency, g.realmName(s)))
	}
	if !found {
		add("{{K|  No one is hunting you.}}")
	}
	add("")
	add("{{Y|Allies and enemies made}}")
	var allies, foes []string
	nodes := rlPeople(g)
	for _, n := range nodes {
		if !n.P.Alive {
			continue
		}
		switch {
		case n.P.Opinion >= 50:
			allies = append(allies, fmt.Sprintf("%s (%+d)", n.P.Name, n.P.Opinion))
		case n.P.Opinion <= -50:
			foes = append(foes, fmt.Sprintf("%s (%+d)", n.P.Name, n.P.Opinion))
		}
	}
	for _, a := range g.localCompanions() {
		allies = append(allies, a.Name+" (companion)")
	}
	var fk []string
	for f := range g.Rep {
		fk = append(fk, f)
	}
	sort.Strings(fk)
	for _, f := range fk {
		v := g.Rep[f]
		switch {
		case v >= 150:
			allies = append(allies, g.FactionName(f)+fmt.Sprintf(" (standing %+d)", v))
		case v <= -150:
			foes = append(foes, g.FactionName(f)+fmt.Sprintf(" (standing %+d)", v))
		}
	}
	if len(allies) == 0 {
		add("  {{G|Allies:}} {{K|none yet.}}")
	} else {
		for _, l := range Wrap("  {{G|Allies:}} "+strings.Join(allies, ", ")+".", 66) {
			add(l)
		}
	}
	if len(foes) == 0 {
		add("  {{R|Enemies:}} {{K|none yet.}}")
	} else {
		for _, l := range Wrap("  {{R|Enemies:}} "+strings.Join(foes, ", ")+".", 66) {
			add(l)
		}
	}
	if loans := bkSt(g).Loans; len(loans) > 0 {
		add("")
		add("{{Y|Loans}}")
		for _, l := range loans {
			st := ""
			if l.Defaulted {
				st = " {{R|defaulted}}"
			}
			add(fmt.Sprintf("  You owe %s %d, due day %d.%s", l.BankName, l.Principal, l.DueDay, st))
		}
	}
	return L
}

// ---------------------------------------------------------------- keys and drawing

func rlKeys(g *Game, k string) {
	st := g.rl()
	nodes := rlPeople(g)
	switch k {
	case "Escape", "q":
		g.Mode = "main"
		return
	case "Tab", "ArrowRight", "N6":
		st.Tab = (st.Tab + 1) % 3
		st.Scroll, st.DScr = 0, 0
		return
	case "ArrowLeft", "N4":
		st.Tab = (st.Tab + 2) % 3
		st.Scroll, st.DScr = 0, 0
		return
	case "1", "2", "3":
		st.Tab = int(k[0] - '1')
		st.Scroll, st.DScr = 0, 0
		return
	}
	if st.Tab == 2 {
		switch k {
		case "ArrowDown", "N2", "j":
			st.Scroll++
		case "ArrowUp", "N8", "k":
			st.Scroll = maxi(0, st.Scroll-1)
		case "PageDown":
			st.Scroll += 12
		case "PageUp":
			st.Scroll = maxi(0, st.Scroll-12)
		case "Home":
			st.Scroll = 0
		}
		return
	}
	switch k {
	case "ArrowDown", "N2", "j":
		st.Sel = mini(st.Sel+1, maxi(0, len(nodes)-1))
		st.DScr = 0
	case "ArrowUp", "N8", "k":
		st.Sel = maxi(0, st.Sel-1)
		st.DScr = 0
	case "PageDown":
		if st.Tab == 1 {
			st.DScr += 10
		} else {
			st.Sel = mini(st.Sel+8, maxi(0, len(nodes)-1))
		}
	case "PageUp":
		if st.Tab == 1 {
			st.DScr = maxi(0, st.DScr-10)
		} else {
			st.Sel = maxi(0, st.Sel-8)
		}
	}
}

func (g *Game) rlDraw() {
	s := g.sc
	st := g.rl()
	x, y, w, h := g.panel("Relationships and consequences")
	names := []string{"Web", "People", "Consequences"}
	tx := x
	for i, n := range names {
		c := "K"
		if st.Tab == i {
			c = "Y"
		}
		tx = s.TextB(tx, y, fmt.Sprintf("{{%s|[ %d %s ]}} ", c, i+1, n), CGray, CBgDark, x+w)
	}
	y += 2
	h -= 2
	foot := func(t string) { s.TextB(x, y+h-1, "{{K|"+t+"}}", CGray, CBgDark, x+w) }
	if st.Tab == 2 {
		L := rlConsequences(g)
		view := h - 1
		st.Scroll = clampi(st.Scroll, 0, maxi(0, len(L)-view))
		for i := 0; i < view && st.Scroll+i < len(L); i++ {
			s.TextB(x, y+i, L[st.Scroll+i], CGray, CBgDark, x+w)
		}
		foot("Tab: next page. Up/Down/PageUp/PageDown: scroll. Esc: close.")
		return
	}
	nodes := rlPeople(g)
	if len(nodes) == 0 {
		s.TextB(x, y, "{{K|You do not know anyone yet. Talk to people and they will appear here.}}", CGray, CBgDark, x+w)
		foot("Esc: close.")
		return
	}
	st.Sel = clampi(st.Sel, 0, len(nodes)-1)
	rows := rlRows(nodes)
	selRow := 0
	for i, r := range rows {
		if r.idx == st.Sel {
			selRow = i
		}
	}
	if st.Tab == 0 {
		gh := 14
		g.rlGraph(x, y, w, gh, nodes, st.Sel)
		s.TextB(x, y+gh, "{{K|Links: }}{{Y|spouse}} {{c|kin}} {{G|friend}} {{R|rival}} {{M|patron}}  {{K|Tags: L loves, H hates, F fears, $ debt, ! grudge}}", CGray, CBgDark, x+w)
		ly := y + gh + 1
		view := h - gh - 2
		g.rlList(x, ly, w, view, rows, nodes, st, selRow, 24)
		foot("Up/Down: choose. Tab: next page. Esc: close.")
		return
	}
	lw := 36
	g.rlList(x, y, lw, h-1, rows, nodes, st, selRow, 16)
	dx := x + lw + 1
	dw := w - lw - 1
	for yy := 0; yy < h-1; yy++ {
		s.Put(dx-1, y+yy, '|', CDarkGray, CBgDark)
	}
	var L []string
	for _, l := range rlDetail(g, nodes[st.Sel]) {
		if l == "" {
			L = append(L, "")
			continue
		}
		L = append(L, Wrap(l, dw)...)
	}
	view := h - 1
	st.DScr = clampi(st.DScr, 0, maxi(0, len(L)-view))
	for i := 0; i < view && st.DScr+i < len(L); i++ {
		s.TextB(dx, y+i, L[st.DScr+i], CGray, CBgDark, x+w)
	}
	foot("Up/Down: person. PageUp/PageDown: scroll detail. Tab: next page. Esc: close.")
}

func (g *Game) rlList(x, y, w, view int, rows []rlRow, nodes []*rlNode, st *rlState, selRow, nameW int) {
	s := g.sc
	if view < 1 {
		return
	}
	if selRow < st.Scroll {
		st.Scroll = selRow
	}
	if selRow >= st.Scroll+view {
		st.Scroll = selRow - view + 1
	}
	st.Scroll = clampi(st.Scroll, 0, maxi(0, len(rows)-view))
	for i := 0; i < view && st.Scroll+i < len(rows); i++ {
		r := rows[st.Scroll+i]
		yy := y + i
		if r.idx < 0 {
			s.TextB(x, yy, "{{Y|"+r.hdr+"}}", CGray, CBgDark, x+w)
			continue
		}
		bg := CBgDark
		if r.idx == st.Sel {
			bg = CBgSel
		}
		s.Fill(x, yy, w, 1, bg)
		s.TextB(x, yy, rlListLine(nodes[r.idx], nameW, r.idx == st.Sel), CGray, bg, x+w)
	}
}
