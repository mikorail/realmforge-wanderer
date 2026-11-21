package engine

import (
	"strconv"
	"strings"
)

// Combat-feel events for the web shell (prefix hx): HitBegin snapshots the actors of the current zone before a
// key press, HitEnd diffs them afterwards (HP, positions, statuses) and reads the new message-log lines, and
// returns structured events (damage, crit, miss, block, parry, heal, death, burn, poison, bleed) for floating
// numbers, flashes and shake. Read-only and deterministic: no g.R, no game state is changed, at most 12 events
// per key. The snapshot lives in g.ext and holds only unexported fields, so it adds nothing to saves.

// HitEvent is one thing that happened to a cell during the last key press.
type HitEvent struct {
	Kind     string `json:"kind"` // hit crit miss block parry heal death burn poison bleed
	X        int    `json:"x"`    // map cell of the target
	Y        int    `json:"y"`
	N        int    `json:"n"`          // damage or heal amount (0 for miss/block/parry)
	Attacker int    `json:"attackerId"` // actor id, 0 when unknown or none
	Target   int    `json:"targetId"`
	Elem     string `json:"elem,omitempty"` // fire frost shock acid poison dark holy, empty for plain
	Boss     bool   `json:"boss"`           // the attacker or the target is a boss
	Player   bool   `json:"player"`         // the target is the player
}

const hxMax = 12

type hxActor struct {
	id, x, y, hp, mhp int
	name              string
	proper, boss, me  bool
	burn, poison, bld int
}

type hxSnap struct {
	zone  string
	list  []hxActor
	dead  bool
	valid bool
}

type hxState struct{ s hxSnap }

func (g *Game) hxTake() hxSnap {
	s := hxSnap{dead: g.Dead}
	if g.Cur == nil || g.P == nil || g.OnWorld {
		return s
	}
	s.valid, s.zone = true, g.Cur.Key
	s.list = make([]hxActor, 0, len(g.Cur.Actors)+1)
	add := func(a *Actor) {
		s.list = append(s.list, hxActor{id: a.ID, x: a.X, y: a.Y, hp: a.HP, mhp: a.MaxHP, name: a.Name, proper: a.Proper, boss: a.Boss, me: a == g.P,
			burn: a.St("burning"), poison: a.St("poisoned"), bld: a.St("bleeding")})
	}
	add(g.P)
	for _, a := range g.Cur.Actors {
		if a != nil && a != g.P {
			add(a)
		}
	}
	return s
}

// HitBegin records the state just before a key is handled.
func (g *Game) HitBegin() {
	st := g.ext("hitfx", func() any { return &hxState{} }).(*hxState)
	st.s = g.hxTake()
	g.ext("hitfx-n", func() any { return &hxLog{} }).(*hxLog).take(g)
}

// hxThe mirrors g.the for a snapshot actor, lower case.
func hxThe(a *hxActor) string {
	if a.me {
		return "you"
	}
	low := strings.ToLower(a.name)
	if a.proper || strings.HasPrefix(low, "the ") {
		return low
	}
	return "the " + low
}

// hxStrip removes {{C|...}} colour markup from a log line and lower-cases it.
func hxStrip(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == '{' && i+1 < len(s) && s[i+1] == '{' {
			j := strings.IndexByte(s[i:], '|')
			if j > 0 && j < 6 {
				i += j
				continue
			}
		}
		if s[i] == '}' && i+1 < len(s) && s[i+1] == '}' {
			i++
			continue
		}
		b.WriteByte(s[i])
	}
	return strings.ToLower(b.String())
}

func hxLetter(c byte) bool { return c >= 'a' && c <= 'z' }

// hxFind returns the first index of w in s that sits on word boundaries from position from, or -1.
func hxFind(s, w string, from int) int {
	for from <= len(s) {
		i := strings.Index(s[from:], w)
		if i < 0 {
			return -1
		}
		i += from
		e := i + len(w)
		if (i == 0 || !hxLetter(s[i-1])) && (e >= len(s) || !hxLetter(s[e])) {
			return i
		}
		from = i + 1
	}
	return -1
}

func hxElem(tail string) string {
	for _, e := range [][2]string{{"fire", "fire"}, {"flame", "fire"}, {"frost", "frost"}, {"freez", "frost"}, {"shock", "shock"}, {"lightning", "shock"},
		{"acid", "acid"}, {"poison", "poison"}, {"venom", "poison"}, {"shadow", "dark"}, {"curse", "dark"}, {"holy", "holy"}, {"radiant", "holy"}} {
		if hxFind(tail, e[0], 0) >= 0 || strings.Contains(tail, e[0]) && len(e[0]) >= 5 {
			return e[1]
		}
	}
	return ""
}

// hxParty finds who a log line names, in order of appearance: returns snapshot indexes of the first two entities.
// Entities with the same name are told apart by pick(candidates).
func hxParty(line string, byName map[string][]int, names []string, pick func(c []int, first int) int) (a, t int) {
	a, t = -1, -1
	type hit struct {
		i    int
		name string
	}
	var found []hit
	for _, nm := range names {
		if i := hxFind(line, nm, 0); i >= 0 {
			found = append(found, hit{i, nm})
		}
	}
	// order by position, longer names first on a tie
	for i := 1; i < len(found); i++ {
		for j := i; j > 0 && (found[j].i < found[j-1].i || found[j].i == found[j-1].i && len(found[j].name) > len(found[j-1].name)); j-- {
			found[j], found[j-1] = found[j-1], found[j]
		}
	}
	// drop names swallowed by a longer one at the same spot
	var f2 []hit
	for _, h := range found {
		if len(f2) > 0 && h.i < f2[len(f2)-1].i+len(f2[len(f2)-1].name) {
			continue
		}
		f2 = append(f2, h)
	}
	if len(f2) > 0 {
		a = pick(byName[f2[0].name], -1)
	}
	if len(f2) > 1 {
		t = pick(byName[f2[1].name], a)
	}
	return
}

// HitEnd returns the combat events caused since HitBegin.
func (g *Game) HitEnd() []HitEvent {
	st := g.ext("hitfx", func() any { return &hxState{} }).(*hxState)
	old, now := st.s, g.hxTake()
	if !old.valid || !now.valid || old.zone != now.zone {
		return nil
	}
	// fresh log lines (same diffing as sfxinfo, using the snapshot count kept there)
	sst := g.ext("hitfx-n", func() any { return &hxLog{} }).(*hxLog)
	k := len(g.Msgs) - sst.n
	if k < 0 || k > len(g.Msgs) {
		k = 0
	}
	var lines []string
	for _, m := range g.Msgs[len(g.Msgs)-k:] {
		lines = append(lines, hxStrip(m.T))
	}
	if k == 0 && len(g.Msgs) > 0 && sst.n > 0 {
		m := g.Msgs[len(g.Msgs)-1]
		if m.Turn != sst.turn || m.Rep != sst.rep || m.T != sst.last {
			lines = append(lines, hxStrip(m.T))
		}
	}
	sst.take(g)

	byID := map[int]int{} // id -> index in old.list
	byNow := map[int]*hxActor{}
	for i := range old.list {
		byID[old.list[i].id] = i
	}
	for i := range now.list {
		byNow[now.list[i].id] = &now.list[i]
	}
	byName := map[string][]int{}
	var names []string
	for i := range old.list {
		nm := hxThe(&old.list[i])
		if _, ok := byName[nm]; !ok {
			names = append(names, nm)
		}
		byName[nm] = append(byName[nm], i)
	}
	// longest names first so "the cave rat" wins over "the rat"
	for i := 1; i < len(names); i++ {
		for j := i; j > 0 && len(names[j]) > len(names[j-1]); j-- {
			names[j], names[j-1] = names[j-1], names[j]
		}
	}

	var out []HitEvent
	push := func(e HitEvent) {
		if len(out) < hxMax {
			out = append(out, e)
		}
	}
	posOf := func(i int) (int, int) {
		if c := byNow[old.list[i].id]; c != nil {
			return c.x, c.y
		}
		return old.list[i].x, old.list[i].y
	}
	dist := func(i, j int) int {
		if i < 0 || j < 0 {
			return 0
		}
		dx, dy := old.list[i].x-old.list[j].x, old.list[i].y-old.list[j].y
		if dx < 0 {
			dx = -dx
		}
		if dy < 0 {
			dy = -dy
		}
		if dy > dx {
			return dy
		}
		return dx
	}
	lost := map[int]int{} // old index -> HP lost this key
	for i := range old.list {
		if c := byNow[old.list[i].id]; c != nil && c.hp < old.list[i].hp {
			lost[i] = old.list[i].hp - c.hp
		}
	}
	accounted := map[int]int{}
	killed := map[int]bool{}
	pick := func(c []int, other int) int {
		best, bs := -1, 1<<30
		for _, i := range c {
			s := dist(i, other)
			if _, ok := lost[i]; ok && accounted[i] < lost[i] {
				s -= 50
			}
			if s < bs {
				best, bs = i, s
			}
		}
		return best
	}
	boss := func(a, t int) bool {
		return a >= 0 && old.list[a].boss || t >= 0 && old.list[t].boss
	}
	id := func(i int) int {
		if i < 0 {
			return 0
		}
		return old.list[i].id
	}
	for _, ln := range lines {
		switch {
		case strings.Contains(ln, " damage") && strings.Contains(ln, " for "):
			a, t := hxParty(ln, byName, names, pick)
			if t < 0 {
				continue
			}
			n := 0
			if i := strings.Index(ln, " for "); i >= 0 {
				j := i + 5
				e := j
				for e < len(ln) && ln[e] >= '0' && ln[e] <= '9' {
					e++
				}
				n, _ = strconv.Atoi(ln[j:e])
			}
			kind := "hit"
			if strings.Contains(ln, "critical!") {
				kind = "crit"
			}
			tail := ln
			if i := strings.Index(ln, " damage"); i >= 0 {
				tail = ln[i:]
			}
			x, y := posOf(t)
			accounted[t] += n
			push(HitEvent{Kind: kind, X: x, Y: y, N: n, Attacker: id(a), Target: id(t), Elem: hxElem(tail), Boss: boss(a, t), Player: old.list[t].me})
		case strings.Contains(ln, "fail to penetrate"):
			a, t := hxParty(ln, byName, names, pick)
			if t >= 0 {
				x, y := posOf(t)
				push(HitEvent{Kind: "block", X: x, Y: y, Attacker: id(a), Target: id(t), Boss: boss(a, t), Player: old.list[t].me})
			}
		case strings.Contains(ln, "parr"):
			a, t := hxParty(ln, byName, names, pick)
			if t >= 0 {
				// "X parries Y": the parrier is the first name, the attacker the second
				x, y := posOf(a)
				push(HitEvent{Kind: "parry", X: x, Y: y, Attacker: id(t), Target: id(a), Boss: boss(a, t), Player: old.list[a].me})
			}
		case strings.Contains(ln, " miss"):
			a, t := hxParty(ln, byName, names, pick)
			if t >= 0 {
				x, y := posOf(t)
				push(HitEvent{Kind: "miss", X: x, Y: y, Attacker: id(a), Target: id(t), Boss: boss(a, t), Player: old.list[t].me})
			}
		case strings.HasSuffix(ln, " dies.") || strings.HasSuffix(ln, " is destroyed.") || strings.HasSuffix(ln, " crumbles to dust."):
			for _, nm := range names {
				if hxFind(ln, nm, 0) == 0 {
					for _, i := range byName[nm] {
						if byNow[old.list[i].id] == nil && !killed[i] && !old.list[i].me {
							killed[i] = true
							break
						}
					}
					break
				}
			}
		}
	}
	heal := false
	for _, ln := range lines {
		if strings.Contains(ln, "heal") || strings.Contains(ln, "restor") || strings.Contains(ln, "mend") || strings.Contains(ln, "bandage") || strings.Contains(ln, "regenerat") {
			heal = true
		}
	}
	// residual HP changes with no damage line: statuses, spells, falls, traps, and heals
	for i := range old.list {
		o := &old.list[i]
		c := byNow[o.id]
		if c == nil {
			continue
		}
		if !o.me && !g.visible(c.x, c.y) {
			continue
		}
		if rest := lost[i] - accounted[i]; rest > 0 {
			kind, el := "hit", ""
			switch {
			case o.burn > 0 || c.burn > 0:
				kind, el = "burn", "fire"
			case o.poison > 0 || c.poison > 0:
				kind, el = "poison", "poison"
			case o.bld > 0 || c.bld > 0:
				kind = "bleed"
			}
			push(HitEvent{Kind: kind, X: c.x, Y: c.y, N: rest, Target: o.id, Elem: el, Boss: o.boss, Player: o.me})
		} else if gain := c.hp - o.hp; gain > 0 && (heal && gain >= 2 || gain >= 3 && gain*10 >= o.mhp) {
			push(HitEvent{Kind: "heal", X: c.x, Y: c.y, N: gain, Target: o.id, Boss: o.boss, Player: o.me})
		}
	}
	// deaths: the player, and actors that vanished after a death line
	if now.dead && !old.dead {
		me := &old.list[0]
		push(HitEvent{Kind: "death", X: now.list[0].x, Y: now.list[0].y, Target: me.id, Player: true})
	}
	for i := range old.list {
		if killed[i] {
			o := &old.list[i]
			push(HitEvent{Kind: "death", X: o.x, Y: o.y, Target: o.id, Boss: o.boss})
		}
	}
	return out
}

// hxLog remembers the message-log position so HitEnd reads only lines added by this key.
type hxLog struct {
	n, turn, rep int
	last         string
}

func (l *hxLog) take(g *Game) {
	l.n = len(g.Msgs)
	if l.n > 0 {
		m := g.Msgs[l.n-1]
		l.last, l.turn, l.rep = m.T, m.Turn, m.Rep
	}
}

// HitJS converts events for the wasm frame (syscall/js cannot take structs).
func HitJS(ev []HitEvent) []any {
	out := make([]any, len(ev))
	for i, e := range ev {
		m := map[string]any{"kind": e.Kind, "x": e.X, "y": e.Y, "n": e.N, "attackerId": e.Attacker, "targetId": e.Target, "boss": e.Boss, "player": e.Player}
		if e.Elem != "" {
			m["elem"] = e.Elem
		}
		out[i] = m
	}
	return out
}
