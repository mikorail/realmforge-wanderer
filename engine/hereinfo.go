package engine

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// "What can I do here" overview (prefix hi). Key ' (apostrophe; ? is Help) opens mode "here": a screen listing what
// the player can do from the current tile, derived from live state each time it opens (and after every pick):
// the people in view with the real talk menu (talkMenu probed in a sandbox, so every talk hook shows), doors,
// stairs, containers and other interactive terrain (through the context-menu rules in ctxmenu.go), the cook / drink /
// sleep / workbench keys when they apply, items underfoot and nearby, hazards, and live deadlines and quests.
// Rows are ranked by nearness. A pick fires the ctx pseudo key "ctx:<id>:<x>:<y>" or the plain key; both go through
// g.Key so turn costs, crime and replay are exactly the keyboard's. The screen is drawn by the engine; the key column
// holds ENGINE key names and the frame field "here" (HereJSON) lists their cells so the shell can overlay the player's
// rebound keys (src/tips.ts hereLabel, Tooltip.svelte). Nothing here is saved: the state in g.ext is json:"-".

type hiRow struct {
	Group string // heading
	Text  string // plain text
	Tone  string // "hostile" | "warn" | "good" | ""
	Dist  int
	Key   string // engine key shown in the key column ("" none)
	Act   string // what a pick sends through g.Key: a ctx:... pseudo key or a plain key; "" = information only
	Extra string // second line (talk options and the like)
}

type hiPos struct {
	X, Y, W int
	K       string
}

type hiState struct {
	rows []hiRow
	sel  int // index into rows (always an actionable row when one exists)
	pos  []hiPos
}

func init() {
	modeKeys["here"] = hiKeys
	modeDraws["here"] = hiDraw
	mainKeys["'"] = hiOpen
	paEntries = append(paEntries, paEntry{Name: "What can I do here", Keywords: "overview actions nearby interact help options here", Cat: "Screen", Key: "'",
		Cond: func(g *Game) bool { return g.Mode == "main" }})
}

func (g *Game) hi() *hiState {
	return g.ext("hi", func() any { return &hiState{} }).(*hiState)
}

func (st *hiState) MarshalJSON() ([]byte, error) { return []byte("{}"), nil }
func (st *hiState) UnmarshalJSON([]byte) error   { return nil }

func hiOpen(g *Game) {
	if g.P == nil || g.Cur == nil || g.OnWorld {
		g.Msg("{{K|Step into a place first: there is nothing to list on the world map.}}")
		return
	}
	st := g.hi()
	st.rows, st.sel = hiBuild(g), 0
	hiFirstPick(st)
	g.Mode = "here"
}

func hiFirstPick(st *hiState) {
	for i, r := range st.rows {
		if r.Act != "" {
			st.sel = i
			return
		}
	}
}

func hiKeys(g *Game, k string) {
	st := g.hi()
	acts := []int{}
	for i, r := range st.rows {
		if r.Act != "" {
			acts = append(acts, i)
		}
	}
	pos := 0
	for j, i := range acts {
		if i == st.sel {
			pos = j
		}
	}
	step := func(d int) {
		if len(acts) > 0 {
			st.sel = acts[clampi(pos+d, 0, len(acts)-1)]
		}
	}
	switch {
	case k == "Escape" || k == "'" || k == "Enter" && len(acts) == 0:
		g.Mode = "main"
	case k == "ArrowUp" || k == "N8":
		step(-1)
	case k == "ArrowDown" || k == "N2":
		step(1)
	case k == "PageUp":
		step(-8)
	case k == "PageDown":
		step(8)
	case k == "Home":
		step(-1000)
	case k == "End":
		step(1000)
	case k == "Enter":
		hiFire(g, st.rows[st.sel])
	case letterIndex(k) >= 0 && letterIndex(k) < len(acts):
		hiFire(g, st.rows[acts[letterIndex(k)]])
	}
}

func hiFire(g *Game, r hiRow) {
	if r.Act == "" {
		return
	}
	g.Mode = "main"
	g.Key(r.Act)
	if g.Mode == "main" && !g.Dead && g.hiReopen() {
		st := g.hi()
		st.rows, st.sel = hiBuild(g), 0
		hiFirstPick(st)
		g.Mode = "here"
	}
}

// hiReopen: the screen stays up after a pick only when the pick took no turn that changed the place; to keep the
// keyboard feel (act, then the world answers) we return to the map.
func (g *Game) hiReopen() bool { return false }

// ---- building the list ----

func hiDistOf(g *Game, x, y int) int { return cmCheb(x, y, g.P.X, g.P.Y) }

func hiBuild(g *Game) []hiRow {
	var rows []hiRow
	z, p := g.Cur, g.P
	add := func(r hiRow) { rows = append(rows, r) }
	grp := func(d int) string {
		if d <= 1 {
			return "Right here"
		}
		return "Nearby"
	}
	hintKey := func(h string) string {
		if len(h) == 1 || len(h) >= 2 && len(h) <= 3 && h[0] == 'F' && h[1] >= '0' && h[1] <= '9' {
			return h
		}
		return ""
	}
	ctxRow := func(it CtxItem, x, y, d int, label string) hiRow {
		act := fmt.Sprintf("ctx:%s:%d:%d", it.ID, x, y)
		return hiRow{Group: grp(d), Text: label, Tone: it.Tone, Dist: d, Key: hintKey(it.Hint), Act: act}
	}

	// underfoot
	if its := z.ItemsAt(p.X, p.Y); len(its) > 0 {
		names := []string{}
		for i, it := range its {
			if i >= 3 {
				names = append(names, fmt.Sprintf("and %d more", len(its)-3))
				break
			}
			names = append(names, Plain(it.Display()))
		}
		add(hiRow{Group: "Right here", Text: "Pick up: " + strings.Join(names, ", "), Dist: 0, Key: "g", Act: "g"})
	}
	if t := z.At(p.X, p.Y); t == TDown || t == TUp {
		key, lbl := ">", "Take the stairs down"
		if t == TUp {
			key, lbl = "<", "Take the stairs up"
		}
		add(hiRow{Group: "Right here", Text: lbl, Dist: 0, Key: key, Act: key})
	}
	add(hiRow{Group: "Right here", Text: "Search the area", Dist: 0, Key: "s", Act: "s"})

	// people
	type pr struct {
		a *Actor
		d int
	}
	var ppl []pr
	for _, a := range z.Actors {
		if a == p || a.HP <= 0 || !g.visible(a.X, a.Y) {
			continue
		}
		if d := hiDistOf(g, a.X, a.Y); d <= 8 {
			ppl = append(ppl, pr{a, d})
		}
	}
	sort.SliceStable(ppl, func(i, j int) bool { return ppl[i].d < ppl[j].d })
	if len(ppl) > 8 {
		ppl = ppl[:8]
	}
	probed := 0
	for _, e := range ppl {
		a, d := e.a, e.d
		name := Plain(g.the(a))
		mood := "friendly"
		if g.Hostile(a) {
			mood = "hostile"
		} else if isFollower(a) {
			mood = "companion"
		}
		dist := fmt.Sprintf("%d away", d)
		if d <= 1 {
			dist = "adjacent"
		}
		items := g.CtxMenu(a.X, a.Y)
		made := false
		for _, it := range items {
			if it.ID == "look" || it.ID == "walk" {
				continue
			}
			r := ctxRow(it, a.X, a.Y, d, it.Label)
			if made {
				r.Text = "   " + it.Label
			} else {
				r.Text = fmt.Sprintf("%s (%s, %s): %s", name, mood, dist, it.Label)
			}
			if it.ID == "talk" && !isFollower(a) && probed < 3 && !g.Hostile(a) {
				probed++
				if opts := g.hiProbeTalk(a); len(opts) > 0 {
					r.Extra = "They offer: " + strings.Join(opts, "; ")
				}
			}
			add(r)
			made = true
		}
		if !made { // context menus off: tell the player how
			r := hiRow{Group: grp(d), Text: fmt.Sprintf("%s (%s, %s)", name, mood, dist), Dist: d}
			if d <= 1 {
				if k := cmKeyFor(cmSign(a.X-p.X), cmSign(a.Y-p.Y)); k != "" {
					r.Text += ": move into them to " + map[bool]string{true: "attack", false: "talk"}[g.Hostile(a)]
					r.Act, r.Tone = k, map[bool]string{true: "hostile"}[g.Hostile(a)]
				}
			}
			add(r)
		}
	}

	// interactive terrain: all adjacent cells, plus the nearest of each kind within 6
	seen := map[Terrain]int{}
	type tc struct {
		x, y, d int
		t       Terrain
	}
	var cells []tc
	for y := maxi(0, p.Y-6); y <= mini(MH-1, p.Y+6); y++ {
		for x := maxi(0, p.X-6); x <= mini(MW-1, p.X+6); x++ {
			if x == p.X && y == p.Y || z.Seen[y*MW+x] == 0 {
				continue
			}
			t := z.At(x, y)
			d := t.D()
			if !(d.Interact || t == TDoor || t == TLocked || t == TDown || t == TUp || t == TStall || t == TOre || t == TStatue) {
				continue
			}
			cells = append(cells, tc{x, y, hiDistOf(g, x, y), t})
		}
	}
	sort.SliceStable(cells, func(i, j int) bool { return cells[i].d < cells[j].d })
	for _, c := range cells {
		if seen[c.t] >= 1 && c.d > 1 || seen[c.t] >= 3 {
			continue
		}
		seen[c.t]++
		name := strings.ToLower(c.t.D().Name)
		for _, it := range g.CtxMenu(c.x, c.y) {
			switch it.ID {
			case "stairs", "bump", "steal":
			default:
				continue
			}
			lbl := it.Label
			if it.ID == "bump" && strings.HasPrefix(lbl, "Examine") {
				lbl = "Use the " + name
			}
			if c.d > 1 {
				lbl += fmt.Sprintf(" (%d away)", c.d)
			}
			add(ctxRow(it, c.x, c.y, c.d, lbl))
		}
	}
	nearT := func(ts ...Terrain) (int, bool) {
		best, ok := 99, false
		for _, c := range cells {
			for _, t := range ts {
				if c.t == t && c.d < best {
					best, ok = c.d, true
				}
			}
		}
		return best, ok
	}
	// scans above only keep interactive kinds; beds, benches and fires are found with a direct sweep
	scan := func(ts ...Terrain) (int, bool) {
		best, ok := 99, false
		for y := maxi(0, p.Y-5); y <= mini(MH-1, p.Y+5); y++ {
			for x := maxi(0, p.X-5); x <= mini(MW-1, p.X+5); x++ {
				t := z.At(x, y)
				for _, w := range ts {
					if t == w && z.Seen[y*MW+x] != 0 {
						if d := hiDistOf(g, x, y); d < best {
							best, ok = d, true
						}
					}
				}
			}
		}
		return best, ok
	}
	_ = nearT
	if d, ok := scan(TBed, TAshBed, TTent); ok {
		add(hiRow{Group: grp(d), Text: "Sleep (needs a bed within reach)", Dist: d, Key: "F5", Act: map[bool]string{true: "F5"}[d <= 1]})
	}
	if d, ok := scan(TAnvil, TMachine); ok {
		txt := "Work at the bench"
		if g.tinkRank() < 1 {
			txt += " (needs the Tinkering skill)"
		}
		add(hiRow{Group: grp(d), Text: txt, Dist: d, Key: "T", Act: map[bool]string{true: "T"}[d <= 1 && g.tinkRank() >= 1]})
	}
	if why := g.canCook(); why == "" {
		if d, ok := scan(TBrazier); ok || z.Kind == "town" {
			if !ok {
				d = 1
			}
			add(hiRow{Group: grp(d), Text: "Cook a meal", Dist: d, Key: "C", Act: map[bool]string{true: "C"}[d <= 1 || z.Kind == "town"]})
		}
	}
	if z.WaterAt(p.X, p.Y) > 0 && !z.Underwater || g.adjacent(TWell) || g.adjacent(TShallow) {
		add(hiRow{Group: "Right here", Text: "Drink", Dist: 0, Key: "d", Act: "d"})
	}
	// loose items nearby
	n := 0
	for _, it := range z.Items {
		d := hiDistOf(g, it.X, it.Y)
		if d == 0 || d > 6 || !g.visible(it.X, it.Y) || n >= 4 {
			continue
		}
		n++
		add(hiRow{Group: grp(d), Text: fmt.Sprintf("Go and pick up %s (%d away)", Plain(it.Display()), d), Dist: d, Key: "g", Act: fmt.Sprintf("ctx:pickup:%d:%d", it.X, it.Y)})
	}
	// zone edges
	if !g.OnWorld {
		for _, e := range []struct {
			n    string
			d    int
			x, y int
		}{{"west", p.X, 0, p.Y}, {"east", MW - 1 - p.X, MW - 1, p.Y}, {"north", p.Y, p.X, 0}, {"south", MH - 1 - p.Y, p.X, MH - 1}} {
			if e.d <= 4 && z.Walkable(e.x, e.y) && z.Seen[e.y*MW+e.x] != 0 {
				add(hiRow{Group: grp(e.d), Text: fmt.Sprintf("Leave this area by the %s edge (%d steps)", e.n, e.d), Dist: e.d, Act: fmt.Sprintf("ctx:walk:%d:%d", e.x, e.y)})
			}
		}
	}

	rows = append(rows, hiThreats(g)...)
	rows = append(rows, hiObligations(g)...)
	return hiOrder(rows)
}

func hiGroupRank(g string) int {
	switch g {
	case "Right here":
		return 0
	case "Nearby":
		return 1
	case "Hazards":
		return 2
	}
	return 3
}

// hiOrder sorts by group, then nearness, keeping a person's follow-up actions (their "   " lines) glued to them.
func hiOrder(rows []hiRow) []hiRow {
	type blk struct {
		rows []hiRow
		g, d int
	}
	var bl []blk
	for _, r := range rows {
		if strings.HasPrefix(r.Text, "   ") && len(bl) > 0 {
			bl[len(bl)-1].rows = append(bl[len(bl)-1].rows, r)
			continue
		}
		bl = append(bl, blk{[]hiRow{r}, hiGroupRank(r.Group), r.Dist})
	}
	sort.SliceStable(bl, func(i, j int) bool {
		if bl[i].g != bl[j].g {
			return bl[i].g < bl[j].g
		}
		return bl[i].d < bl[j].d
	})
	var out []hiRow
	for _, b := range bl {
		out = append(out, b.rows...)
	}
	return out
}

// hiProbeTalk lists the talk options the person would offer, by building the real menu and rolling every other
// piece of state back (rng, log, popup, options, selection, who is talked to).
func (g *Game) hiProbeTalk(a *Actor) (out []string) {
	rs, nm, popup, opts, sel, talk, mode := g.R.S, len(g.Msgs), g.Popup, g.Opts, g.Sel, g.Talk, g.Mode
	defer func() {
		recover()
		g.R.S = rs
		if len(g.Msgs) > nm {
			g.Msgs = g.Msgs[:nm]
		}
		g.Popup, g.Opts, g.Sel, g.Talk, g.Mode = popup, opts, sel, talk, mode
	}()
	g.Talk = a
	g.talkMenu("")
	for _, o := range g.Opts {
		t := strings.TrimSpace(Plain(o.Text))
		if t == "" || strings.HasPrefix(t, "Goodbye") || strings.HasPrefix(t, "Who are you") || strings.HasPrefix(t, "What news") || strings.HasPrefix(t, "(") {
			continue
		}
		if len(out) >= 6 {
			out = append(out, "...")
			break
		}
		out = append(out, strings.TrimRight(t, "."))
	}
	return out
}

func hiThreats(g *Game) []hiRow {
	var rows []hiRow
	z, p := g.Cur, g.P
	add := func(t, tone string, d int) {
		rows = append(rows, hiRow{Group: "Hazards", Text: t, Tone: tone, Dist: d})
	}
	hostile, nearest := 0, 99
	for _, a := range z.Actors {
		if a.HP > 0 && a != p && g.Hostile(a) && g.visible(a.X, a.Y) {
			hostile++
			nearest = mini(nearest, hiDistOf(g, a.X, a.Y))
		}
	}
	if hostile > 0 {
		add(fmt.Sprintf("%d hostile in sight, nearest %d away", hostile, nearest), "hostile", nearest)
	}
	haz := map[Terrain]string{TLava: "Lava", TDeep: "Deep water", TQuicksand: "Quicksand", TTar: "Tar pit", TFumarole: "Fumarole vent",
		TGeyser: "Geyser", TChasm: "Chasm", TRift: "Rift", TCliff: "Cliff edge", TVoid: "Open sky (a long fall)", TSulfur: "Sulfur vents"}
	bestHaz := map[Terrain]int{}
	fireD, acidD := 0, 0
	for y := maxi(0, p.Y-3); y <= mini(MH-1, p.Y+3); y++ {
		for x := maxi(0, p.X-3); x <= mini(MW-1, p.X+3); x++ {
			if z.Seen[y*MW+x] == 0 {
				continue
			}
			if _, ok := haz[z.At(x, y)]; ok {
				if d := hiDistOf(g, x, y); bestHaz[z.At(x, y)] == 0 || d < bestHaz[z.At(x, y)] {
					bestHaz[z.At(x, y)] = maxi(d, 1)
				}
			}
			if z.Fire[y*MW+x] > 0 {
				if d := hiDistOf(g, x, y); fireD == 0 || d < fireD {
					fireD = maxi(d, 1)
				}
			}
			if z.Liq[y*MW+x] == 3 {
				if d := hiDistOf(g, x, y); acidD == 0 || d < acidD {
					acidD = maxi(d, 1)
				}
			}
		}
	}
	var ks []int
	for t := range bestHaz {
		ks = append(ks, int(t))
	}
	sort.Ints(ks)
	for _, k := range ks {
		add(fmt.Sprintf("%s, %d away", haz[Terrain(k)], bestHaz[Terrain(k)]), "warn", bestHaz[Terrain(k)])
	}
	if fireD > 0 {
		add(fmt.Sprintf("Open flame, %d away", fireD), "warn", fireD)
	}
	if acidD > 0 {
		add(fmt.Sprintf("Acid pooled on the ground, %d away", acidD), "warn", acidD)
	}
	if aw := g.activeWeather(); aw != nil && aw.Kind != nil {
		add(fmt.Sprintf("Weather: %s, %d degrees", strings.ToLower(aw.Kind.Name), aw.Temp), "", 0)
	}
	if ob, ch := g.hzExposure(); ob != nil {
		add(fmt.Sprintf("Outbreak here: %s (about %d%% per check of catching it)", g.W.hzDisease(ob.Disease).Name, int(ch*100+0.5)), "warn", 0)
	}
	if z.Dark {
		add("This place is dark: you see only what your light reaches", "", 0)
	}
	if z.Underwater {
		add("Underwater: you can hold your breath only so long", "warn", 0)
	}
	if g.Hunger > 1200 || g.Thirst > 900 {
		add("You need food or drink", "warn", 0)
	}
	return rows
}

func hiObligations(g *Game) []hiRow {
	var rows []hiRow
	z := g.Cur
	here := g.hzHere()
	if g.W != nil && g.dl() != nil {
		evs := g.dlActive()
		sort.SliceStable(evs, func(i, j int) bool {
			_, a := g.dlCount(evs[i])
			_, b := g.dlCount(evs[j])
			return a < b
		})
		n := 0
		for _, e := range evs {
			if !e.Known || n >= 4 {
				continue
			}
			local := e.Burg != 0 && e.Burg == z.Burg || e.Site == here
			lbl, left := g.dlCount(e)
			tone := ""
			if left < DayLen {
				tone = "warn"
			}
			where := "elsewhere"
			if local {
				where = "HERE"
			}
			if !local && n >= 2 {
				continue
			}
			n++
			rows = append(rows, hiRow{Group: "Deadlines and quests", Text: fmt.Sprintf("%s [%s]: %s %s", Plain(g.dlTitle(e)), where, lbl, dlClock(left)), Tone: tone, Extra: Plain(g.dlGoal(e))})
		}
	}
	for _, q := range g.Quests {
		if q == nil || q.Closed {
			continue
		}
		var giver *Actor
		for _, a := range z.Actors {
			if a.ID == q.GiverID && a.HP > 0 {
				giver = a
			}
		}
		atTarget := q.Target != "" && q.Target == z.Key
		switch {
		case q.Done && giver != nil:
			r := hiRow{Group: "Deadlines and quests", Text: fmt.Sprintf("Quest done: %s. Report to %s", Plain(q.Title), Plain(g.the(giver))), Tone: "good", Dist: hiDistOf(g, giver.X, giver.Y)}
			if g.visible(giver.X, giver.Y) {
				r.Act = fmt.Sprintf("ctx:talk:%d:%d", giver.X, giver.Y)
			}
			rows = append(rows, r)
		case !q.Done && atTarget:
			rows = append(rows, hiRow{Group: "Deadlines and quests", Text: fmt.Sprintf("Quest target is in this area: %s", Plain(q.Title)), Tone: "good"})
		case !q.Done && giver != nil:
			rows = append(rows, hiRow{Group: "Deadlines and quests", Text: fmt.Sprintf("Open quest from %s: %s", Plain(g.the(giver)), Plain(q.Title))})
		}
	}
	return rows
}

// ---- drawing ----

func hiDraw(g *Game) {
	s := g.sc
	st := g.hi()
	st.pos = st.pos[:0]
	s.Fill(0, 0, SW, SH, CBgDark)
	s.Box(0, 0, SW, SH, CDarkCyan, "What can I do here")
	z := g.Cur
	if z == nil || g.P == nil {
		return
	}
	s.TextB(2, 1, fmt.Sprintf("{{W|%s}} {{K|· standing on %s · turn %d}}", Plain(z.Name), strings.ToLower(z.At(g.P.X, g.P.Y).D().Name), g.Turn), CGray, CBgDark, SW-2)
	// lay out lines
	type line struct {
		head bool
		row  int // index into rows, -1 for heads and extras
		text string
		ex   bool
	}
	var lines []line
	last := ""
	for i, r := range st.rows {
		if r.Group != last {
			if last != "" {
				lines = append(lines, line{row: -1, text: "", head: false})
			}
			lines = append(lines, line{head: true, row: -1, text: r.Group})
			last = r.Group
		}
		lines = append(lines, line{row: i})
		if r.Extra != "" {
			for _, w := range Wrap(r.Extra, SW-22) {
				lines = append(lines, line{row: -1, text: w, ex: true})
			}
		}
	}
	if len(st.rows) == 0 {
		s.TextB(3, 4, "{{K|Nothing stands out here.}}", CGray, CBgDark, SW-2)
	}
	const y0, rowsMax = 3, SH - 7
	selLine := 0
	for i, l := range lines {
		if l.row == st.sel && !l.ex {
			selLine = i
		}
	}
	top := 0
	if selLine >= rowsMax-2 {
		top = selLine - rowsMax + 3
	}
	letters := map[int]int{}
	n := 0
	for i, r := range st.rows {
		if r.Act != "" {
			letters[i] = n
			n++
		}
	}
	y := y0
	for i := top; i < len(lines) && y < y0+rowsMax; i++ {
		l := lines[i]
		switch {
		case l.head:
			s.TextB(2, y, "{{Y|"+l.text+"}}", CGray, CBgDark, SW-2)
		case l.ex:
			s.TextB(20, y, "{{K|"+l.text+"}}", CGray, CBgDark, SW-2)
		case l.row >= 0:
			r := st.rows[l.row]
			bg := CBgDark
			if l.row == st.sel && r.Act != "" {
				bg = CBgSel
			}
			s.Fill(2, y, SW-4, 1, bg)
			lt := " "
			if ix, ok := letters[l.row]; ok && ix < 26 {
				lt = string(rune('a' + ix))
			}
			col := "w"
			switch r.Tone {
			case "hostile":
				col = "R"
			case "warn":
				col = "Y"
			case "good":
				col = "G"
			}
			if r.Act == "" {
				col = "K"
				if r.Tone == "warn" {
					col = "Y"
				} else if r.Tone == "hostile" {
					col = "R"
				}
			}
			s.TextB(3, y, "{{Y|"+lt+"}}", CGray, bg, SW-2)
			if r.Key != "" {
				s.TextB(6, y, fmt.Sprintf("{{K|[%-5s]}}", r.Key), CGray, bg, SW-2)
				st.pos = append(st.pos, hiPos{X: 6, Y: y, W: 7, K: r.Key})
			}
			s.TextB(14, y, fmt.Sprintf("{{%s|%s}}", col, strings.ReplaceAll(r.Text, "{", "(")), CGray, bg, SW-2)
		}
		y++
	}
	if top > 0 {
		s.TextB(SW-10, y0-1, "{{K|▲ more}}", CGray, CBgDark, SW-2)
	}
	if top+rowsMax < len(lines) {
		s.TextB(SW-10, y0+rowsMax, "{{K|▼ more}}", CGray, CBgDark, SW-2)
	}
	s.TextB(2, SH-2, "{{K|↑↓ select · Enter or letter does it · ' or Esc closes · rows without a letter are information}}", CGray, CBgDark, SW-2)
}

// HereInfo is the frame field "here": the cells where an engine key name is drawn, for the shell to overlay the
// player's own binding. Nil outside mode "here".
type HereInfo struct {
	Cells []HereCell `json:"cells"`
}
type HereCell struct {
	X int    `json:"x"`
	Y int    `json:"y"`
	W int    `json:"w"`
	K string `json:"k"` // engine key name
}

func (g *Game) HereInfo() *HereInfo {
	if g.Mode != "here" || g.P == nil {
		return nil
	}
	st := g.hi()
	out := &HereInfo{}
	for _, c := range st.pos {
		out.Cells = append(out.Cells, HereCell{X: c.X, Y: c.Y, W: c.W, K: c.K})
	}
	return out
}

// HereJSON is HereInfo as a JSON string for the wasm bridge ("" when not in the mode).
func (g *Game) HereJSON() string {
	h := g.HereInfo()
	if h == nil {
		return ""
	}
	b, err := json.Marshal(h)
	if err != nil {
		return ""
	}
	return string(b)
}
