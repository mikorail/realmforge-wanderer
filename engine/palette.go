package engine

// Command palette (prefix pa). Press ; to open a type-to-find list of every screen, action and live
// ability. Typing filters by name and keyword, Up/Down move, Enter runs, Esc closes. Running an entry
// returns to the main screen and presses its key, so it behaves exactly like the real binding.

import (
	"sort"
	"strings"
)

const paRecentMax = 5

type paState struct {
	Rec []string `json:"r,omitempty"` // names of the last chosen entries, newest first
	buf string
	sel int
	out Out
}

func (g *Game) pa() *paState {
	return g.ext("pa", func() any { return &paState{} }).(*paState)
}

func init() {
	mainKeys[";"] = func(g *Game) {
		st := g.pa()
		st.buf, st.sel = "", 0
		g.Mode = "pal"
	}
	modeKeys["pal"] = paKey
	modeDraws["pal"] = paDraw
}

// paTakeOut merges the Out a palette-run key produced (Save) into the one Key is returning.
func paTakeOut(g *Game, out Out) Out {
	if st := g.pa(); st.out.Save || st.out.Exit {
		out.Save = out.Save || st.out.Save
		out.Exit = out.Exit || st.out.Exit
		st.out = Out{}
	}
	return out
}

// paList builds the visible entries: the static table plus the live abilities.
func paList(g *Game) []paEntry {
	var l []paEntry
	for _, e := range paEntries {
		if e.Cond == nil || e.Cond(g) {
			l = append(l, e)
		}
	}
	for _, ab := range g.Abilities() {
		ab := ab
		l = append(l, paEntry{Name: ab.Name, Keywords: "ability power use cast", Cat: "Ability",
			Func: func(g *Game) { g.startAbility(ab) }})
	}
	return l
}

// paScore rates how well q matches e; 0 means no match. Every space-separated token must hit.
func paScore(e paEntry, q string) int {
	name := strings.ToLower(e.Name)
	kw := strings.ToLower(e.Keywords + " " + e.Cat + " " + e.Key + " " + e.Label)
	total := 0
	for _, t := range strings.Fields(strings.ToLower(q)) {
		s := 0
		switch {
		case strings.HasPrefix(name, t):
			s = 100
		case paWordPrefix(name, t):
			s = 80
		case strings.Contains(name, t):
			s = 60
		case paWordPrefix(kw, t):
			s = 40
		case strings.Contains(kw, t):
			s = 25
		case paSubseq(name, t):
			s = 10
		}
		if s == 0 {
			return 0
		}
		total += s
	}
	return total
}

func paWordPrefix(s, t string) bool {
	for _, w := range strings.FieldsFunc(s, func(r rune) bool { return r == ' ' || r == '/' }) {
		if strings.HasPrefix(w, t) {
			return true
		}
	}
	return false
}

func paSubseq(s, t string) bool {
	i := 0
	rt := []rune(t)
	for _, r := range s {
		if i < len(rt) && r == rt[i] {
			i++
		}
	}
	return i == len(rt)
}

// paMatches returns the filtered, ordered rows for the current input.
func paMatches(g *Game) []paEntry {
	st := g.pa()
	all := paList(g)
	if strings.TrimSpace(st.buf) == "" {
		var out []paEntry
		seen := map[string]bool{}
		for _, n := range st.Rec {
			for _, e := range all {
				if e.Name == n && !seen[n] {
					seen[n] = true
					out = append(out, e)
				}
			}
		}
		for _, e := range all {
			if !seen[e.Name] {
				out = append(out, e)
			}
		}
		return out
	}
	type sc struct {
		e paEntry
		s int
		i int
	}
	var m []sc
	for i, e := range all {
		if s := paScore(e, st.buf); s > 0 {
			m = append(m, sc{e, s, i})
		}
	}
	sort.SliceStable(m, func(a, b int) bool {
		if m[a].s != m[b].s {
			return m[a].s > m[b].s
		}
		return m[a].i < m[b].i
	})
	out := make([]paEntry, len(m))
	for i, x := range m {
		out[i] = x.e
	}
	return out
}

func paRun(g *Game, e paEntry) {
	st := g.pa()
	rec := []string{e.Name}
	for _, n := range st.Rec {
		if n != e.Name && len(rec) < paRecentMax {
			rec = append(rec, n)
		}
	}
	st.Rec = rec
	g.Mode = "main"
	if g.OnWorld {
		g.Mode = "world"
	}
	if e.Func != nil {
		e.Func(g)
		return
	}
	if e.Key != "" {
		o := g.Key(e.Key)
		st.out.Save = st.out.Save || o.Save
		st.out.Exit = st.out.Exit || o.Exit
	}
}

func paKey(g *Game, k string) {
	st := g.pa()
	switch k {
	case "Escape":
		g.Mode = "main"
	case "Enter":
		if m := paMatches(g); st.sel >= 0 && st.sel < len(m) {
			paRun(g, m[st.sel])
		}
	case "Backspace":
		if rs := []rune(st.buf); len(rs) > 0 {
			st.buf = string(rs[:len(rs)-1])
			st.sel = 0
		}
	case "ArrowUp":
		st.sel = maxi(0, st.sel-1)
	case "ArrowDown":
		st.sel = mini(maxi(0, len(paMatches(g))-1), st.sel+1)
	case "PageUp":
		st.sel = maxi(0, st.sel-10)
	case "PageDown":
		st.sel = mini(maxi(0, len(paMatches(g))-1), st.sel+10)
	case "Tab":
		if m := paMatches(g); len(m) > 0 && st.sel < len(m) {
			st.buf = strings.ToLower(m[st.sel].Name)
		}
	default:
		if rs := []rune(k); len(rs) == 1 && rs[0] >= 32 && rs[0] < 127 && len(st.buf) < 60 {
			st.buf += k
			st.sel = 0
		}
	}
}

func paDraw(g *Game) {
	s := g.sc
	st := g.pa()
	m := paMatches(g)
	x0, y0, w, h := 10, 3, SW-20, 26
	s.Box(x0, y0, w, h, CDarkCyan, "Command palette")
	s.TextB(x0+2, y0+1, "{{W|> }}{{W|"+st.buf+"}}{{Y|_}}", CGray, CBgDark, x0+w-2)
	for x := x0 + 1; x < x0+w-1; x++ {
		s.Put(x, y0+2, '─', CDarkGray, CBgDark)
	}
	rows := h - 5
	if st.sel >= len(m) {
		st.sel = maxi(0, len(m)-1)
	}
	top := 0
	if st.sel >= rows {
		top = st.sel - rows + 1
	}
	if len(m) == 0 {
		s.TextB(x0+2, y0+3, "{{K|Nothing matches.}}", CGray, CBgDark, x0+w-2)
	}
	recent := strings.TrimSpace(st.buf) == ""
	for r := 0; r < rows && top+r < len(m); r++ {
		e := m[top+r]
		y := y0 + 3 + r
		mark, col := "  ", "y"
		if top+r == st.sel {
			mark, col = "{{Y|>}} ", "W"
		}
		s.TextB(x0+2, y, mark+"{{"+col+"|"+e.Name+"}}", CGray, CBgDark, x0+w-2)
		tag := e.Cat
		if recent && top+r < len(st.Rec) && top+r < paRecentMax {
			tag = "recent"
		}
		s.TextB(x0+w-30, y, "{{K|"+tag+"}}", CGray, CBgDark, x0+w-2)
		lab := e.Label
		if lab == "" {
			lab = e.Key
		}
		if lab != "" {
			s.TextB(x0+w-3-len([]rune(lab)), y, "{{C|"+lab+"}}", CGray, CBgDark, x0+w-2)
		}
	}
	s.TextB(x0+2, y0+h-2, "{{K|type to filter · Up/Down choose · Enter run · Tab complete · Esc close}}", CGray, CBgDark, x0+w-2)
}
