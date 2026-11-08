package engine

import (
	"fmt"
	"sort"
	"strings"
)

// The six table games of games.go (prefix gm). Each fills a gmImpl: start, key, draw. Rounds roll
// with g.R like skill checks do; nothing here runs during world generation.

func init() {
	gmImpls[0] = gmImpl{gmCardStart, gmCardKey, gmCardDraw, "Left/Right: pick  Enter: play  p: palm a trump"}
	gmImpls[1] = gmImpl{gmLieStart, gmLieKey, gmLieDraw, "Up/Down: count  Left/Right: face  Enter: bid  l: liar  r: read  p: peek"}
	gmImpls[2] = gmImpl{gmHazStart, gmHazKey, gmHazDraw, "Enter: roll  d: double  w: weighted dice  p: switch dice"}
	gmImpls[3] = gmImpl{gmTileStart, gmTileKey, gmTileDraw, "Left/Right: pick  Enter: draw/discard  w: win  c: claim  p: palm"}
	gmImpls[4] = gmImpl{gmTaflStart, gmTaflKey, gmTaflDraw, "Arrows: cursor  Enter: select/move  p: slip a piece"}
	gmImpls[5] = gmImpl{gmShellStart, gmShellKey, gmShellDraw, "1/2/3: pick a shell  w: watch hands  c: call the palm"}
}

func gmShuffle(g *Game, n int) []int {
	d := make([]int, n)
	for i := range d {
		d[i] = i
	}
	for i := n - 1; i > 0; i-- {
		j := g.R.Int(0, i)
		d[i], d[j] = d[j], d[i]
	}
	return d
}

func gmHas(a []int, v int) bool {
	for _, x := range a {
		if x == v {
			return true
		}
	}
	return false
}

func gmDel(a []int, i int) []int { return append(append([]int{}, a[:i]...), a[i+1:]...) }

func gmCount(a []int, v int) int {
	n := 0
	for _, x := range a {
		if x == v {
			n++
		}
	}
	return n
}

// ---- 0: trick-taking cards (24 cards, 4 suits x ranks 1-6, trump suit turned up, follow suit) ----

var gmSuitName = [4]string{"Sw", "Cu", "Co", "St"}

func gmCardStr(c, trump int) string {
	s := fmt.Sprintf("%s%d", gmSuitName[c/6], c%6+1)
	if c/6 == trump {
		return "{{Y|" + s + "}}"
	}
	return s
}

// used[c] marks played cards in t.grid
func gmCardStart(g *Game, t *gmTable) {
	d := gmShuffle(g, 24)
	t.a = append([]int{}, d[:6]...)
	t.b = append([]int{}, d[6:12]...)
	sort.Ints(t.a)
	t.n[0] = d[12] / 6
	t.n[2], t.n[3] = -1, -1
	t.n[6] = g.R.Int(0, 1)
	t.say(fmt.Sprintf("Trump is %s.", gmSuitName[t.n[0]]))
	if t.n[6] == 1 {
		gmCardOppLead(g, t)
	}
}

func gmCardLegal(hand []int, led int) []int {
	var out []int
	for i, c := range hand {
		if led < 0 || c/6 == led/6 {
			out = append(out, i)
		}
	}
	if len(out) == 0 {
		for i := range hand {
			out = append(out, i)
		}
	}
	return out
}

func gmCardBeats(led, resp, trump int) bool {
	if resp/6 == led/6 {
		return resp%6 > led%6
	}
	return resp/6 == trump
}

func gmCardAI(g *Game, hand []int, led, trump, lvl int) int {
	legal := gmCardLegal(hand, led)
	if g.R.Chance(maxf(0, 0.25-float64(lvl)*0.012)) {
		return legal[g.R.Int(0, len(legal)-1)]
	}
	val := func(c int) int { // trump outranks
		v := c % 6
		if c/6 == trump {
			v += 10
		}
		return v
	}
	best := legal[0]
	if led < 0 {
		bi := -1
		for _, i := range legal {
			if hand[i]/6 != trump && hand[i]%6 >= 4 && (bi < 0 || val(hand[i]) > val(hand[bi])) {
				bi = i
			}
		}
		if bi >= 0 {
			return bi
		}
		for _, i := range legal {
			if val(hand[i]) < val(hand[best]) {
				best = i
			}
		}
		return best
	}
	win := -1
	for _, i := range legal {
		if gmCardBeats(led, hand[i], trump) && (win < 0 || val(hand[i]) < val(hand[win])) {
			win = i
		}
	}
	if win >= 0 {
		return win
	}
	for _, i := range legal {
		if val(hand[i]) < val(hand[best]) {
			best = i
		}
	}
	return best
}

func gmCardOppLead(g *Game, t *gmTable) {
	i := gmCardAI(g, t.b, -1, t.n[0], t.opp.Level)
	t.n[3] = t.b[i]
	t.n[8] = 1
	t.b = gmDel(t.b, i)
	t.say(fmt.Sprintf("%s leads %s.", t.opp.Name, gmCardStr(t.n[3], t.n[0])))
}

func gmCardKey(g *Game, t *gmTable, k string) {
	switch k {
	case "ArrowLeft":
		t.cur = maxi(0, t.cur-1)
	case "ArrowRight":
		t.cur = mini(len(t.a)-1, t.cur+1)
	case "p":
		if t.cheat {
			return
		}
		t.cheat = true
		if !g.gmCheat(t, 2) {
			return
		}
		for r := 5; r >= 0; r-- {
			c := t.n[0]*6 + r
			if t.grid[c] == 0 && !gmHas(t.a, c) && !gmHas(t.b, c) && t.n[2] != c && t.n[3] != c {
				lo := 0
				for i, h := range t.a {
					if h%6 < t.a[lo]%6 {
						lo = i
					}
				}
				t.a[lo] = c
				sort.Ints(t.a)
				t.say("You palm a high trump into your hand.")
				return
			}
		}
		t.say("There is no better card to palm.")
	case "Enter", " ":
		if t.cur >= len(t.a) {
			return
		}
		legal := gmCardLegal(t.a, t.n[3])
		if !gmHas(legal, t.cur) {
			t.say("{{y|You must follow suit.}}")
			return
		}
		c := t.a[t.cur]
		t.a = gmDel(t.a, t.cur)
		t.cur = mini(t.cur, len(t.a)-1)
		t.n[2] = c
		if t.n[3] < 0 {
			t.n[8] = 0
			i := gmCardAI(g, t.b, c, t.n[0], t.opp.Level)
			t.n[3] = t.b[i]
			t.b = gmDel(t.b, i)
			t.say(fmt.Sprintf("%s answers %s.", t.opp.Name, gmCardStr(t.n[3], t.n[0])))
			t.n[6] = 0
		}
		gmCardResolve(g, t)
	}
}

func gmCardResolve(g *Game, t *gmTable) {
	me, op := t.n[2], t.n[3]
	var meWin bool
	if t.n[8] == 0 {
		meWin = !gmCardBeats(me, op, t.n[0])
	} else {
		meWin = gmCardBeats(op, me, t.n[0])
	}
	t.grid[me], t.grid[op] = 1, 1
	if meWin {
		t.n[4]++
		t.n[6] = 0
		t.say("{{G|You take the trick.}}")
	} else {
		t.n[5]++
		t.n[6] = 1
		t.say("They take the trick.")
	}
	t.n[2], t.n[3] = -1, -1
	t.n[7]++
	if t.n[7] >= 6 {
		switch {
		case t.n[4] > 3:
			g.gmEndRound(t, 1)
		case t.n[4] < 3:
			g.gmEndRound(t, -1)
		default:
			g.gmEndRound(t, 0)
		}
		return
	}
	if t.n[6] == 1 {
		gmCardOppLead(g, t)
	}
}

func gmCardDraw(g *Game, t *gmTable, x, y, w int) {
	s := g.sc
	s.Text(x, y, fmt.Sprintf("Trump {{Y|%s}}   Tricks: you {{G|%d}}  them {{R|%d}}   (trick %d of 6)", gmSuitName[t.n[0]], t.n[4], t.n[5], t.n[7]+1), CGray)
	on := "--"
	if t.n[3] >= 0 {
		on = gmCardStr(t.n[3], t.n[0])
	}
	s.Text(x, y+2, "On the table: "+on, CGray)
	var parts []string
	for i, c := range t.a {
		cs := gmCardStr(c, t.n[0])
		if i == t.cur {
			cs = "{{W|>}}" + cs + "{{W|<}}"
		} else {
			cs = " " + cs + " "
		}
		parts = append(parts, cs)
	}
	s.Text(x, y+4, "Your hand: "+strings.Join(parts, " "), CGray)
	s.Text(x, y+6, fmt.Sprintf("{{K|%s holds %d cards.}}", t.opp.Name, len(t.b)), CGray)
}

// ---- 1: liar's dice ----

func gmTail(k, m int) float64 { // P(X >= k), X ~ Bin(m, 1/6)
	if k <= 0 {
		return 1
	}
	if k > m {
		return 0
	}
	p, tot := 1.0/6, 0.0
	for i := k; i <= m; i++ {
		c := 1.0
		for j := 0; j < i; j++ {
			c = c * float64(m-j) / float64(j+1)
		}
		pw := c
		for j := 0; j < i; j++ {
			pw *= p
		}
		for j := 0; j < m-i; j++ {
			pw *= 1 - p
		}
		tot += pw
	}
	return tot
}

func gmLieRoll(g *Game, t *gmTable) {
	t.a, t.b = nil, nil
	for i := 0; i < t.n[0]; i++ {
		t.a = append(t.a, g.R.Dice(1, 6))
	}
	for i := 0; i < t.n[1]; i++ {
		t.b = append(t.b, g.R.Dice(1, 6))
	}
	sort.Ints(t.a)
	t.n[2], t.n[3], t.n[4], t.n[7], t.n[8] = 0, 0, -1, 0, 0
	t.n[5], t.n[6] = 1, 2
	if t.n[10] == 1 {
		gmLieOpp(g, t)
	}
}

func gmLieStart(g *Game, t *gmTable) {
	t.n[0], t.n[1], t.n[10] = 3, 3, 0
	gmLieRoll(g, t)
	t.say("Three dice each. Bid how many of a face are on the table; call liar to challenge.")
}

func gmLieValid(t *gmTable, q, f int) bool {
	return q <= t.n[0]+t.n[1] && (q > t.n[2] || (q == t.n[2] && f > t.n[3]))
}

func gmLieOpp(g *Game, t *gmTable) {
	own := func(f int) int { return gmCount(t.b, f) }
	prob := func(q, f int) float64 { return gmTail(q-own(f), t.n[0]) }
	if t.n[4] == 0 {
		thr := 0.30 + 0.01*float64(t.opp.Level) - 0.012*float64(g.checkMod("bluff")) + (g.R.Float()-0.5)*0.16
		thr = maxf(0.12, thr)
		if thr > 0.5 {
			thr = 0.5
		}
		if prob(t.n[2], t.n[3]) < thr {
			gmLieCall(g, t, 1)
			return
		}
	}
	bq, bf, bp := 0, 0, -1.0
	for f := 1; f <= 6; f++ {
		q := t.n[2] + 1
		if t.n[2] == 0 {
			q = 1
		} else if f > t.n[3] {
			q = t.n[2]
		}
		if !gmLieValid(t, q, f) {
			continue
		}
		if p := prob(q, f); p > bp {
			bq, bf, bp = q, f, p
		}
	}
	if bp < 0 || (t.n[4] == 0 && bp < 0.15) {
		gmLieCall(g, t, 1)
		return
	}
	if g.R.Chance(0.2) { // bluff: push the face they hold most of one higher
		mf := 1
		for f := 1; f <= 6; f++ {
			if own(f) > own(mf) {
				mf = f
			}
		}
		if gmLieValid(t, t.n[2]+1, mf) {
			bq, bf = t.n[2]+1, mf
		}
	}
	t.n[2], t.n[3], t.n[4] = bq, bf, 1
	t.n[5], t.n[6] = bq, mini(6, bf+1)
	if !gmLieValid(t, t.n[5], t.n[6]) {
		t.n[5], t.n[6] = bq+1, bf
	}
	t.say(fmt.Sprintf("%s bids {{Y|%d x %d}}.", t.opp.Name, bq, bf))
}

func gmLieCall(g *Game, t *gmTable, caller int) {
	cnt := gmCount(t.a, t.n[3]) + gmCount(t.b, t.n[3])
	truth := cnt >= t.n[2]
	loser := caller
	if !truth {
		loser = 1 - caller
	}
	who := "You call"
	if caller == 1 {
		who = t.opp.Name + " calls"
	}
	t.say(fmt.Sprintf("%s liar on %dx%d. Dice: you %v, them %v: %d on the table.", who, t.n[2], t.n[3], t.a, t.b, cnt))
	if loser == 0 {
		t.n[0]--
		t.say("{{R|You lose a die.}}")
	} else {
		t.n[1]--
		t.say("{{G|They lose a die.}}")
	}
	switch {
	case t.n[0] <= 0:
		g.gmEndRound(t, -1)
	case t.n[1] <= 0:
		g.gmEndRound(t, 1)
	default:
		t.n[10] = loser
		gmLieRoll(g, t)
	}
}

func gmLieKey(g *Game, t *gmTable, k string) {
	switch k {
	case "ArrowUp":
		t.n[5] = mini(t.n[0]+t.n[1], t.n[5]+1)
	case "ArrowDown":
		t.n[5] = maxi(1, t.n[5]-1)
	case "ArrowLeft":
		t.n[6] = maxi(1, t.n[6]-1)
	case "ArrowRight":
		t.n[6] = mini(6, t.n[6]+1)
	case "Enter", " ":
		if !gmLieValid(t, t.n[5], t.n[6]) {
			t.say("{{y|Your bid must beat the last one.}}")
			return
		}
		t.n[2], t.n[3], t.n[4] = t.n[5], t.n[6], 0
		t.say(fmt.Sprintf("You bid {{Y|%d x %d}}.", t.n[2], t.n[3]))
		gmLieOpp(g, t)
	case "l":
		if t.n[4] != 1 {
			t.say("There is nothing to call.")
			return
		}
		gmLieCall(g, t, 0)
	case "r":
		if t.n[8] > 0 {
			t.say("You have already studied them this round.")
			return
		}
		if g.roll2d6Check(g.checkMod("bluff"), 7+t.opp.Level/2) {
			i := g.R.Int(0, len(t.b)-1)
			t.n[8] = i + 1
			t.say(fmt.Sprintf("{{G|A flicker in their eyes: one of their dice is a %d.}}", t.b[i]))
		} else {
			t.n[8] = 99
			t.say("Their face gives away nothing.")
		}
	case "p":
		if t.cheat {
			return
		}
		t.cheat = true
		if g.gmCheat(t, 2) {
			t.n[7] = 1
			t.say(fmt.Sprintf("{{G|You lift the corner of their cup: %v.}}", t.b))
		}
	}
}

func gmLieDraw(g *Game, t *gmTable, x, y, w int) {
	s := g.sc
	s.Text(x, y, fmt.Sprintf("Your dice: {{W|%v}}", t.a), CGray)
	them := strings.Repeat("[?]", len(t.b))
	if t.n[7] == 1 {
		them = fmt.Sprint(t.b)
	} else if t.n[8] > 0 && t.n[8] < 99 {
		parts := make([]string, len(t.b))
		for i := range parts {
			parts[i] = "[?]"
		}
		parts[t.n[8]-1] = fmt.Sprintf("[%d]", t.b[t.n[8]-1])
		them = strings.Join(parts, "")
	}
	s.Text(x, y+1, "Their dice: "+them, CGray)
	bid := "none yet"
	if t.n[4] >= 0 && t.n[2] > 0 {
		bid = fmt.Sprintf("%d x %d", t.n[2], t.n[3])
	}
	s.Text(x, y+3, "Current bid: {{Y|"+bid+"}}", CGray)
	ok := "{{G|"
	if !gmLieValid(t, t.n[5], t.n[6]) {
		ok = "{{R|"
	}
	s.Text(x, y+4, fmt.Sprintf("Your bid:    %s%d x %d}}", ok, t.n[5], t.n[6]), CGray)
	s.Text(x, y+6, fmt.Sprintf("{{K|You hold %d dice, they hold %d.  Bluff skill %+d}}", t.n[0], t.n[1], g.checkMod("bluff")), CGray)
}

// ---- 2: hazard dice (come-out 7/11 wins, 2/3/12 loses, else the point) ----

func gmHazFindDice(g *Game) *Item {
	for _, it := range g.P.Inv {
		if it.Kind == "gmdice" && it.Charges > 0 {
			return it
		}
	}
	return nil
}

func gmHazStart(g *Game, t *gmTable) {
	t.say("Come-out roll: 7 or 11 wins, 2, 3 or 12 loses, anything else is your point.")
}

func gmHazRoll(g *Game, t *gmTable, loaded bool) {
	d1, d2 := g.R.Dice(1, 6), g.R.Dice(1, 6)
	if loaded && g.R.Chance(0.5) {
		target := 7
		if t.n[0] > 0 {
			target = t.n[0]
		}
		d1 = g.R.Int(maxi(1, target-6), mini(6, target-1))
		d2 = target - d1
	}
	tot := d1 + d2
	t.a = []int{d1, d2}
	t.n[1] = tot
	t.n[2]++
	t.say(fmt.Sprintf("You throw {{W|%d + %d = %d}}.", d1, d2, tot))
	if t.n[0] == 0 {
		switch {
		case tot == 7 || tot == 11:
			g.gmEndRound(t, 1)
		case tot == 2 || tot == 3 || tot == 12:
			g.gmEndRound(t, -1)
		default:
			t.n[0] = tot
			t.say(fmt.Sprintf("The point is {{Y|%d}}. Roll it again before a 7.", tot))
		}
		return
	}
	switch {
	case tot == t.n[0]:
		g.gmEndRound(t, 1)
	case tot == 7:
		g.gmEndRound(t, -1)
	case t.n[2] >= 14:
		g.gmEndRound(t, 0)
	}
}

func gmHazKey(g *Game, t *gmTable, k string) {
	switch k {
	case "Enter", " ":
		gmHazRoll(g, t, false)
	case "d":
		if t.n[0] == 0 || t.n[3] > 0 || t.item != nil || t.bet == 0 {
			t.say("You can only double once, on a point, with coin down.")
			return
		}
		if !g.cuSpend(t.bet) {
			t.say("{{y|You cannot cover a double.}}")
			return
		}
		t.n[3] = 1
		t.bet *= 2
		t.say("You double the stake.")
	case "w":
		it := gmHazFindDice(g)
		if it == nil {
			t.say("You have no weighted dice.")
			return
		}
		it.Charges--
		if g.gmCheat(t, -1) {
			gmHazRoll(g, t, true)
		}
	case "p":
		if t.cheat {
			return
		}
		t.cheat = true
		if g.gmCheat(t, 3) {
			t.say("You swap in your own bones.")
			gmHazRoll(g, t, true)
		}
	}
}

func gmHazDraw(g *Game, t *gmTable, x, y, w int) {
	s := g.sc
	d := "[ ] [ ]"
	if len(t.a) == 2 {
		d = fmt.Sprintf("[%d] [%d]", t.a[0], t.a[1])
	}
	pt := "come-out"
	if t.n[0] > 0 {
		pt = fmt.Sprintf("point {{Y|%d}}", t.n[0])
	}
	s.Text(x, y, "Dice: {{W|"+d+"}}   Phase: "+pt, CGray)
	s.Text(x, y+2, fmt.Sprintf("Rolls so far: %d (a push after 14)", t.n[2]), CGray)
	if it := gmHazFindDice(g); it != nil {
		s.Text(x, y+3, fmt.Sprintf("{{K|%s: %d throws left}}", it.Name, it.Charges), CGray)
	}
}

// ---- 3: tile game (27 kinds x2: win with two melds and a pair in 8 tiles) ----

func gmTileStr(i int) string { return string("BOK"[i/9]) + fmt.Sprint(i%9+1) }

func gmTileMelds(c *[27]int, n int) bool {
	if n == 0 {
		return true
	}
	i := 0
	for i < 27 && c[i] == 0 {
		i++
	}
	if i == 27 {
		return n == 0
	}
	if c[i] >= 3 {
		c[i] -= 3
		ok := gmTileMelds(c, n-1)
		c[i] += 3
		if ok {
			return true
		}
	}
	if i%9 <= 6 && c[i+1] > 0 && c[i+2] > 0 {
		c[i]--
		c[i+1]--
		c[i+2]--
		ok := gmTileMelds(c, n-1)
		c[i]++
		c[i+1]++
		c[i+2]++
		return ok
	}
	return false
}

func gmTileWin(h []int) bool {
	if len(h) != 8 {
		return false
	}
	var c [27]int
	for _, v := range h {
		c[v]++
	}
	for p := 0; p < 27; p++ {
		if c[p] >= 2 {
			c[p] -= 2
			ok := gmTileMelds(&c, 2)
			c[p] += 2
			if ok {
				return true
			}
		}
	}
	return false
}

func gmTileScore(h []int, i int) int {
	var c [27]int
	for _, v := range h {
		c[v]++
	}
	v := h[i]
	sc := (c[v] - 1) * 4
	r := v % 9
	if r > 0 {
		sc += c[v-1] * 2
	}
	if r < 8 {
		sc += c[v+1] * 2
	}
	if r > 1 {
		sc += c[v-2]
	}
	if r < 7 {
		sc += c[v+2]
	}
	return sc
}

func gmTileStart(g *Game, t *gmTable) {
	d := gmShuffle(g, 54)
	for i := range d {
		d[i] %= 27
	}
	t.a = append([]int{}, d[:7]...)
	t.b = append([]int{}, d[7:14]...)
	t.w = append([]int{}, d[14:]...)
	sort.Ints(t.a)
	t.n[0], t.n[1], t.n[2] = 0, -1, -1
	t.say("Draw a tile, discard a tile. Two sets (triple or run) and a pair win.")
}

func gmTileOpp(g *Game, t *gmTable) {
	if t.n[2] >= 0 {
		if gmTileWin(append(append([]int{}, t.b...), t.n[2])) {
			t.say(fmt.Sprintf("{{R|%s claims your %s: \"Ron!\"}}", t.opp.Name, gmTileStr(t.n[2])))
			g.gmEndRound(t, -1)
			return
		}
	}
	if len(t.w) == 0 {
		t.say("The wall is empty: a draw.")
		g.gmEndRound(t, 0)
		return
	}
	t.b = append(t.b, t.w[0])
	t.w = t.w[1:]
	if gmTileWin(t.b) {
		t.say("{{R|" + t.opp.Name + " lays down the hand: \"Tsumo!\"}}")
		g.gmEndRound(t, -1)
		return
	}
	best, bs := 0, -1<<30
	for i := range t.b {
		rest := gmDel(t.b, i)
		sc := gmTileScore(t.b, i) * -1
		for tile := 0; tile < 27; tile++ {
			if gmTileWin(append(append([]int{}, rest...), tile)) {
				sc += 100
				break
			}
		}
		sc += g.R.Int(0, maxi(0, 3-t.opp.Level/4))
		if sc > bs {
			best, bs = i, sc
		}
	}
	t.n[1] = t.b[best]
	t.b = gmDel(t.b, best)
	t.n[2] = -1
	t.n[0] = 0
	t.say(fmt.Sprintf("%s discards %s.", t.opp.Name, gmTileStr(t.n[1])))
}

func gmTileKey(g *Game, t *gmTable, k string) {
	switch k {
	case "ArrowLeft":
		t.cur = maxi(0, t.cur-1)
	case "ArrowRight":
		t.cur = mini(len(t.a)-1, t.cur+1)
	case "c":
		if t.n[0] == 0 && t.n[1] >= 0 && gmTileWin(append(append([]int{}, t.a...), t.n[1])) {
			t.say("{{G|You claim their discard and show the hand!}}")
			g.gmEndRound(t, 1)
		} else {
			t.say("That tile does not finish your hand.")
		}
	case "w":
		if t.n[0] == 1 && gmTileWin(t.a) {
			t.say("{{G|You lay down a winning hand!}}")
			g.gmEndRound(t, 1)
		} else {
			t.say("You have no winning hand.")
		}
	case "p":
		if t.n[0] != 1 || t.cheat {
			return
		}
		t.cheat = true
		if !g.gmCheat(t, 2) {
			return
		}
		rest := gmDel(t.a, t.cur)
		for tile := 0; tile < 27; tile++ {
			if gmTileWin(append(append([]int{}, rest...), tile)) {
				t.a = append(rest, tile)
				sort.Ints(t.a)
				t.say("You palm the tile you needed.")
				return
			}
		}
		t.say("You palm a tile but it does not help.")
	case "Enter", " ", "d":
		if t.n[0] == 0 {
			if len(t.w) == 0 {
				g.gmEndRound(t, 0)
				return
			}
			t.a = append(t.a, t.w[0])
			t.w = t.w[1:]
			sort.Ints(t.a)
			t.n[0] = 1
			t.n[1] = -1
			t.say("You draw " + gmTileStr(t.a[len(t.a)-1]) + ".")
			if gmTileWin(t.a) {
				t.say("{{G|Your hand is complete: press w!}}")
			}
			return
		}
		if t.cur >= len(t.a) {
			return
		}
		t.n[2] = t.a[t.cur]
		t.a = gmDel(t.a, t.cur)
		t.cur = mini(t.cur, len(t.a)-1)
		t.say("You discard " + gmTileStr(t.n[2]) + ".")
		gmTileOpp(g, t)
	}
}

func gmTileDraw(g *Game, t *gmTable, x, y, w int) {
	s := g.sc
	var parts []string
	for i, v := range t.a {
		c := gmTileStr(v)
		if i == t.cur && t.n[0] == 1 {
			c = "{{W|>" + c + "<}}"
		} else {
			c = " " + c + " "
		}
		parts = append(parts, c)
	}
	s.Text(x, y, "Your tiles: "+strings.Join(parts, ""), CGray)
	disc := "--"
	if t.n[1] >= 0 {
		disc = "{{Y|" + gmTileStr(t.n[1]) + "}}"
	}
	s.Text(x, y+2, fmt.Sprintf("Their last discard: %s    Wall: %d tiles    Their hand: %d", disc, len(t.w), len(t.b)), CGray)
	if t.n[0] == 0 {
		s.Text(x, y+4, "{{K|Enter to draw, or c to claim their discard if it completes your hand.}}", CGray)
	} else {
		s.Text(x, y+4, "{{K|Choose a tile to discard (Enter), or declare the win with w.}}", CGray)
	}
}

// ---- 4: siege board game (brandubh-like, 7x7; attackers move first) ----

func gmTown(p, side int) bool { return (side == 1 && p == 1) || (side == 2 && (p == 2 || p == 3)) }

func gmTRestricted(i int) bool { return i == 24 || i == 0 || i == 6 || i == 42 || i == 48 }

var gmTDirs = [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}}

func gmTMoves(b *[49]int, side int) [][2]int {
	var out [][2]int
	for i, p := range b {
		if !gmTown(p, side) {
			continue
		}
		x, y := i%7, i/7
		for _, d := range gmTDirs {
			for s := 1; s < 7; s++ {
				nx, ny := x+d[0]*s, y+d[1]*s
				if nx < 0 || nx > 6 || ny < 0 || ny > 6 {
					break
				}
				j := ny*7 + nx
				if b[j] != 0 {
					break
				}
				if gmTRestricted(j) && p != 3 {
					continue // may pass an empty throne, never stop on it or a corner
				}
				out = append(out, [2]int{i, j})
			}
		}
	}
	return out
}

// gmTApply moves and captures; it returns how many were taken.
func gmTApply(b *[49]int, from, to int) int {
	p := b[from]
	side := 1
	if p != 1 {
		side = 2
	}
	b[from], b[to] = 0, p
	x, y := to%7, to/7
	n := 0
	for _, d := range gmTDirs {
		nx, ny := x+d[0], y+d[1]
		fx, fy := x+2*d[0], y+2*d[1]
		if nx < 0 || nx > 6 || ny < 0 || ny > 6 {
			continue
		}
		j := ny*7 + nx
		if b[j] == 0 || gmTown(b[j], side) {
			continue
		}
		if fx < 0 || fx > 6 || fy < 0 || fy > 6 {
			continue
		}
		k := fy*7 + fx
		hostile := (k == 0 || k == 6 || k == 42 || k == 48) || (k == 24 && b[24] == 0)
		if gmTown(b[k], side) || hostile {
			b[j] = 0
			n++
		}
	}
	return n
}

func gmTKing(b *[49]int) int {
	for i, p := range b {
		if p == 3 {
			return i
		}
	}
	return -1
}

func gmTLanes(b *[49]int, k int) int {
	n := 0
	x, y := k%7, k/7
	for _, d := range gmTDirs {
		for s := 1; s < 7; s++ {
			nx, ny := x+d[0]*s, y+d[1]*s
			if nx < 0 || nx > 6 || ny < 0 || ny > 6 {
				break
			}
			j := ny*7 + nx
			if (j == 0 || j == 6 || j == 42 || j == 48) && b[j] == 0 {
				n++
				break
			}
			if b[j] != 0 {
				break
			}
		}
	}
	return n
}

// gmTEval is from the defenders' point of view.
func gmTEval(b *[49]int, toMove int) int {
	k := gmTKing(b)
	if k < 0 {
		return -10000
	}
	if k == 0 || k == 6 || k == 42 || k == 48 {
		return 10000
	}
	att, def := 0, 0
	for _, p := range b {
		if p == 1 {
			att++
		} else if p == 2 {
			def++
		}
	}
	sc := def*14 - att*10
	lanes := gmTLanes(b, k)
	if (toMove == 2 && lanes >= 1) || (toMove == 1 && lanes >= 2) {
		return 9000
	}
	sc += lanes * 80
	kx, ky := k%7, k/7
	dist := 6
	for _, c := range [4][2]int{{0, 0}, {6, 0}, {0, 6}, {6, 6}} {
		d := maxi(gmAbs(kx-c[0]), gmAbs(ky-c[1]))
		if d < dist {
			dist = d
		}
	}
	sc += (6 - dist) * 6
	for _, d := range gmTDirs {
		nx, ny := kx+d[0], ky+d[1]
		if nx >= 0 && nx < 7 && ny >= 0 && ny < 7 && b[ny*7+nx] == 1 {
			sc -= 8
		}
	}
	return sc
}

func gmTSearch(b *[49]int, side, depth int) (int, [2]int) {
	mv := gmTMoves(b, side)
	best := [2]int{-1, -1}
	var bv int
	if side == 2 {
		bv = -1 << 30
	} else {
		bv = 1 << 30
	}
	if len(mv) == 0 {
		if side == 2 {
			return -9000, best
		}
		return 9000, best
	}
	for _, m := range mv {
		nb := *b
		gmTApply(&nb, m[0], m[1])
		var v int
		if depth <= 1 || gmTKing(&nb) < 0 || gmTEval(&nb, 3-side) >= 9000 {
			v = gmTEval(&nb, 3-side)
		} else {
			v, _ = gmTSearch(&nb, 3-side, depth-1)
		}
		if (side == 2 && v > bv) || (side == 1 && v < bv) {
			bv, best = v, m
		}
	}
	return bv, best
}

func gmTaflStart(g *Game, t *gmTable) {
	for _, i := range []int{3, 10, 38, 45, 21, 22, 26, 27} {
		t.grid[i] = 1
	}
	for _, i := range []int{17, 23, 25, 31} {
		t.grid[i] = 2
	}
	t.grid[24] = 3
	t.n[3], t.n[4] = 3, 3
	t.n[2] = -1
	t.say("Choose your side: d to defend the king, a to attack. Defenders win at a corner.")
}

func gmTEnd(g *Game, t *gmTable) bool {
	k := gmTKing(&t.grid)
	me := t.n[0]
	if k < 0 {
		t.say("The king is taken.")
		if me == 1 {
			g.gmEndRound(t, 1)
		} else {
			g.gmEndRound(t, -1)
		}
		return true
	}
	if k == 0 || k == 6 || k == 42 || k == 48 {
		t.say("The king reaches a corner!")
		if me == 2 {
			g.gmEndRound(t, 1)
		} else {
			g.gmEndRound(t, -1)
		}
		return true
	}
	if t.n[5] >= 70 {
		t.say("Neither side can break through: a draw.")
		g.gmEndRound(t, 0)
		return true
	}
	return false
}

func gmTAI(g *Game, t *gmTable) {
	side := 3 - t.n[0]
	mv := gmTMoves(&t.grid, side)
	if len(mv) == 0 {
		t.say(t.opp.Name + " has no move.")
		if t.n[0] == 2 {
			g.gmEndRound(t, 1)
		} else {
			g.gmEndRound(t, -1)
		}
		return
	}
	noise := maxi(0, 30-t.opp.Level*3)
	bestM, bestV := mv[0], 1<<30*(1-2*boolInt(side == 2)) // minimise for attackers
	if side == 2 {
		bestV = -1 << 30
	}
	for _, m := range mv {
		nb := t.grid
		gmTApply(&nb, m[0], m[1])
		var v int
		if gmTKing(&nb) < 0 || gmTEval(&nb, 3-side) >= 9000 {
			v = gmTEval(&nb, 3-side)
		} else {
			v, _ = gmTSearch(&nb, 3-side, 1)
		}
		if noise > 0 {
			if side == 2 {
				v += g.R.Int(0, noise)
			} else {
				v -= g.R.Int(0, noise)
			}
		}
		if (side == 2 && v > bestV) || (side == 1 && v < bestV) {
			bestM, bestV = m, v
		}
	}
	n := gmTApply(&t.grid, bestM[0], bestM[1])
	t.n[5]++
	t.say(fmt.Sprintf("%s moves %s%d to %s%d.", t.opp.Name, string('a'+rune(bestM[0]%7)), bestM[0]/7+1, string('a'+rune(bestM[1]%7)), bestM[1]/7+1) + map[bool]string{true: fmt.Sprintf(" {{R|%d of yours taken.}}", n), false: ""}[n > 0])
	gmTEnd(g, t)
}

func gmTaflKey(g *Game, t *gmTable, k string) {
	if t.n[1] == 0 {
		switch k {
		case "d":
			t.n[0], t.n[1] = 2, 1
			t.say("You hold the king. They attack first.")
			gmTAI(g, t)
		case "a":
			t.n[0], t.n[1] = 1, 1
			t.say("You are the attackers: move first.")
		}
		return
	}
	cx, cy := t.n[3], t.n[4]
	switch k {
	case "ArrowLeft":
		t.n[3] = maxi(0, cx-1)
	case "ArrowRight":
		t.n[3] = mini(6, cx+1)
	case "ArrowUp":
		t.n[4] = maxi(0, cy-1)
	case "ArrowDown":
		t.n[4] = mini(6, cy+1)
	case "p":
		if t.cheat || t.n[2] < 0 {
			t.say("Select a piece first, then slip an enemy one off the board.")
			return
		}
		t.cheat = true
		if !g.gmCheat(t, 3) {
			return
		}
		for i, p := range t.grid {
			if p != 0 && p != 3 && !gmTown(p, t.n[0]) {
				t.grid[i] = 0
				t.say("An enemy piece slips into your sleeve.")
				break
			}
		}
	case "Enter", " ":
		i := cy*7 + cx
		me := t.n[0]
		if t.n[2] >= 0 && gmTown(t.grid[i], me) {
			t.n[2] = -1
		}
		if t.n[2] < 0 {
			if gmTown(t.grid[i], me) {
				t.n[2] = i
			}
			return
		}
		ok := false
		for _, m := range gmTMoves(&t.grid, me) {
			if m[0] == t.n[2] && m[1] == i {
				ok = true
				break
			}
		}
		if !ok {
			t.n[2] = -1
			t.say("{{y|That piece cannot go there.}}")
			return
		}
		n := gmTApply(&t.grid, t.n[2], i)
		t.n[2] = -1
		t.n[5]++
		if n > 0 {
			t.say(fmt.Sprintf("{{G|You take %d piece(s).}}", n))
		}
		if gmTEnd(g, t) {
			return
		}
		gmTAI(g, t)
	}
}

func gmTaflDraw(g *Game, t *gmTable, x, y, w int) {
	s := g.sc
	if t.n[1] == 0 {
		s.Text(x, y, "{{Y|d}} defend (king and 4 guards)   {{Y|a}} attack (8 raiders, you move first)", CGray)
		return
	}
	var legal map[int]bool
	if t.n[2] >= 0 {
		legal = map[int]bool{}
		for _, m := range gmTMoves(&t.grid, t.n[0]) {
			if m[0] == t.n[2] {
				legal[m[1]] = true
			}
		}
	}
	for r := 0; r < 7; r++ {
		line := ""
		for c := 0; c < 7; c++ {
			i := r*7 + c
			ch := "{{K|.}}"
			switch t.grid[i] {
			case 1:
				ch = "{{R|A}}"
			case 2:
				ch = "{{C|D}}"
			case 3:
				ch = "{{Y|K}}"
			default:
				switch {
				case legal[i]:
					ch = "{{G|*}}"
				case gmTRestricted(i) && i != 24:
					ch = "{{M|#}}"
				case i == 24:
					ch = "{{M|x}}"
				}
			}
			l, rr := " ", " "
			if c == t.n[3] && r == t.n[4] {
				l, rr = "{{W|[}}", "{{W|]}}"
			}
			if i == t.n[2] {
				l, rr = "{{G|<}}", "{{G|>}}"
			}
			line += l + ch + rr
		}
		s.Text(x+4, y+r, line, CGray)
	}
	side := "defenders (king)"
	if t.n[0] == 1 {
		side = "attackers"
	}
	s.Text(x+4, y+8, "You are the "+side+".  A raiders, D guards, K king, # corner, x throne", CGray)
}

// ---- 5: the shell game (a street scam: the hustler may palm the pea) ----

func gmShellStart(g *Game, t *gmTable) {
	t.n[0] = g.R.Int(0, 2)
	pos := t.n[0]
	n := 6 + g.R.Int(0, 3)
	for i := 0; i < n; i++ {
		a := g.R.Int(0, 2)
		b := (a + g.R.Int(1, 2)) % 3
		t.a = append(t.a, a*3+b)
		if pos == a {
			pos = b
		} else if pos == b {
			pos = a
		}
	}
	t.n[1] = pos
	pc := minf(0.6, 0.3+float64(t.opp.Level)*0.02)
	if g.R.Chance(pc) {
		t.n[2] = 1
	}
	t.say("Keep your eye on the pea.")
}

func gmShellKey(g *Game, t *gmTable, k string) {
	switch k {
	case "w":
		if t.n[3] > 0 {
			t.say("You already watched.")
			return
		}
		t.n[3] = 1
		if g.roll2d6Check(g.checkMod("perception"), 8+t.opp.Level/2) {
			if t.n[2] == 1 {
				t.n[4] = 1
				t.say("{{G|His thumb flicks: the pea left the shells mid-shuffle! It is in his palm.}}")
			} else {
				t.n[4] = 2
				t.say("{{G|His hands are clean: the pea is under a shell.}}")
			}
		} else {
			t.say("Too quick; you see nothing.")
		}
	case "c":
		if t.n[2] == 1 {
			t.say("{{G|You seize his wrist and open his hand: the pea!}}")
			g.gmEndRound(t, 1)
		} else {
			t.say("{{R|His hand is empty. \"You insult me.\"}}")
			g.gmEndRound(t, -1)
		}
	case "1", "2", "3":
		pick := int(k[0] - '1')
		switch {
		case t.n[2] == 1:
			t.say(fmt.Sprintf("{{R|Shell %d is empty. So are the others: the pea was in his palm.}}", pick+1))
			g.gmEndRound(t, -1)
		case pick == t.n[1]:
			t.say(fmt.Sprintf("{{G|The pea is under shell %d.}}", pick+1))
			g.gmEndRound(t, 1)
		default:
			t.say(fmt.Sprintf("{{R|Nothing under shell %d. It was under %d.}}", pick+1, t.n[1]+1))
			g.gmEndRound(t, -1)
		}
	}
}

func gmShellDraw(g *Game, t *gmTable, x, y, w int) {
	s := g.sc
	s.Text(x, y, fmt.Sprintf("The pea starts under shell {{Y|%d}}.", t.n[0]+1), CGray)
	var sw []string
	for _, v := range t.a {
		sw = append(sw, fmt.Sprintf("%d-%d", v/3+1, v%3+1))
	}
	s.Text(x, y+2, "Swaps: "+strings.Join(sw, "  "), CGray)
	s.Text(x, y+4, "  [1]    [2]    [3]", CGray)
	if t.n[4] == 2 {
		s.Text(x, y+6, "{{K|You are sure his hands are clean.}}", CGray)
	}
}
