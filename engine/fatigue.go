package engine

import (
	"fmt"
	"strings"
)

// fatigue.go (prefix ij): a fatigue meter 0..100 (ijState.Fat), sleeping (bed, inn room, tent, campfire,
// open ground), dreams drawn from the world's real history, and the sidebar line. Main key F5 opens the
// sleep menu. Never draws g.R during generation; in-play rolls only.

const (
	ijTiredAt = 50.0
	ijSpentAt = 80.0
)

func init() {
	turnHooks = append(turnHooks, ijFatTurn)
	sideHooks = append(sideHooks, ijSide)
	mainKeys["F5"] = func(g *Game) { g.ijOpenSleep() }
	modeKeys["ijsleep"] = ijSleepKeys
	modeDraws["ijsleep"] = ijSleepDraw
}

var ijRecover = map[string]float64{"bed": 0.28, "inn": 0.28, "tent": 0.22, "camp": 0.16, "open": 0.10}
var ijHealEvery = map[string]int{"bed": 15, "inn": 15, "tent": 25, "camp": 30}
var ijSleepName = map[string]string{"bed": "in a bed", "inn": "in a rented room", "tent": "in your tent", "camp": "by the campfire", "open": "on the open ground"}

func (g *Game) ijMakeTent() *Item {
	return &Item{ID: g.newID(), Kind: "tent", Name: "travelling tent", Glyph: '^', F: CBrown, Value: 60, Weight: 8,
		Lore: "Oiled canvas and poles. Use it outdoors to pitch camp and sleep."}
}

// ijUseItem handles the tent (hook at the top of useItem).
func (g *Game) ijUseItem(it *Item) bool {
	if it == nil || it.Kind != "tent" {
		return false
	}
	if !outdoors(g.Cur) {
		g.Msg("There is no room to pitch a tent here.")
		return true
	}
	g.ijSleep("tent", 600)
	return true
}

func ijFatTurn(g *Game) {
	st := g.ijSt()
	p := g.P
	if p == nil || g.Cur == nil {
		return
	}
	if st.Down {
		if p.St("asleep") <= 0 || st.Fat <= 45 {
			st.Down = false
			delete(p.Status, "asleep")
			p.Energy = maxi(p.Energy, 1000)
			g.Msg("You wake on the ground, stiff and ashamed.")
		} else {
			p.AddSt("asleep", 2)
		}
	}
	if st.Sleep != "" || st.Down {
		kind := st.Sleep
		if kind == "" {
			kind = "open"
		}
		rate := ijRecover[kind]
		if st.Down {
			rate = 0.15
		}
		if aw := g.activeWeather(); aw != nil && aw.Kind != nil && (aw.Kind.Wet >= 2 || aw.Kind.Hurt > 0) && kind != "bed" && kind != "inn" && kind != "tent" {
			rate /= 2
		}
		st.Fat = maxf(0, st.Fat-rate)
		if n := ijHealEvery[kind]; n > 0 && !g.ijNoRegen() && g.Turn%n == 0 && p.HP < p.MaxHP && p.St("bleeding") == 0 && g.Hunger < 2400 {
			p.HP++
		}
	} else {
		r := 0.05
		if p.St("sprint") > 0 || g.nearHostile() {
			r += 0.04
		}
		switch g.encumbrance() {
		case 1:
			r += 0.03
		case 2:
			r += 0.08
		}
		if aw := g.activeWeather(); aw != nil && aw.Kind != nil && (aw.Kind.Wet >= 2 || aw.Kind.Slow > 0 || aw.Kind.Hurt > 0) {
			r += 0.02
		}
		r *= 1 - 0.06*float64(minInt(g.Skills["endurance"], 5))
		st.Fat = minf(100, st.Fat+r)
	}
	tier := 0
	if st.Fat >= ijSpentAt {
		tier = 2
	} else if st.Fat >= ijTiredAt {
		tier = 1
	}
	if tier > st.Tier {
		if tier == 1 {
			g.Msg("{{W|You are getting tired.}}")
		} else {
			g.Msg("{{R|You are exhausted. Find somewhere to sleep.}}")
		}
	}
	st.Tier = tier
	switch tier {
	case 0:
		delete(p.Status, "ij_tired")
		delete(p.Status, "ij_spent")
	case 1:
		delete(p.Status, "ij_spent")
		if p.St("ij_tired") <= 0 {
			p.AddSt("ij_tired", ijForever)
		}
	case 2:
		delete(p.Status, "ij_tired")
		if p.St("ij_spent") <= 0 {
			p.AddSt("ij_spent", ijForever)
		}
	}
	if st.Sleep != "" || st.Down || p.HP <= 0 {
		return
	}
	if tier == 2 && g.R.Chance(0.003) {
		g.Msg("{{R|You nod off on your feet.}}")
		p.AddSt("asleep", 3)
		p.Energy -= 3000
	}
	if st.Fat >= 100 {
		st.Down = true
		g.Msg("{{R|Exhaustion takes you. You collapse where you stand.}}")
		g.ijFirst("collapse", fmt.Sprintf("%s collapsed from exhaustion in %s.", p.Name, g.Cur.Name))
		if g.Cur.Kind == "town" && p.Gold > 10 && g.R.Chance(0.25) {
			loss := p.Gold / 5
			p.Gold -= loss
			g.Msg(fmt.Sprintf("{{R|You wake lighter: someone took %d %s.}}", loss, g.Sk.Currency))
		}
		p.AddSt("asleep", 360)
		p.Energy -= 360 * 1000
	}
}

func (g *Game) ijFatigueWord() string {
	f := g.ijSt().Fat
	switch {
	case f >= ijSpentAt:
		return "{{R|exhausted}}"
	case f >= ijTiredAt:
		return "{{W|tired}}"
	case g.P != nil && g.P.St("rested") > 0:
		return "{{G|rested}}"
	case f < 20:
		return "{{g|fresh}}"
	}
	return "{{y|fine}}"
}

func ijSide(g *Game) []string {
	if g.P == nil {
		return nil
	}
	out := []string{"Fatigue: " + g.ijFatigueWord()}
	if len(g.ijSt().Inj) > 0 {
		var n []string
		for _, in := range g.ijSt().Inj {
			n = append(n, strings.ToLower(in.Name))
		}
		out = append(out, "{{r|"+strings.Join(n, ", ")+"}}")
	}
	return out
}

// ---- sleeping ----

func (g *Game) ijTileNear(r int, f func(x, y int) bool) bool {
	z := g.Cur
	for dy := -r; dy <= r; dy++ {
		for dx := -r; dx <= r; dx++ {
			x, y := g.P.X+dx, g.P.Y+dy
			if z.In(x, y) && f(x, y) {
				return true
			}
		}
	}
	return false
}

func (g *Game) ijHasTent() bool {
	for _, it := range g.P.Inv {
		if it.Kind == "tent" {
			return true
		}
	}
	return false
}

// ijSleepKinds are the ways to sleep here.
func (g *Game) ijSleepKinds() []string {
	var k []string
	if g.ijTileNear(1, func(x, y int) bool { return g.Cur.At(x, y) == TBed }) && (g.homeHere() != nil || g.Cur.Kind != "town") {
		k = append(k, "bed")
	}
	if g.ijHasTent() && outdoors(g.Cur) {
		k = append(k, "tent")
	}
	if g.ijTileNear(2, func(x, y int) bool { return g.Cur.At(x, y) == TBrazier || g.Cur.Fire[y*MW+x] > 0 }) {
		k = append(k, "camp")
	}
	return append(k, "open")
}

func (g *Game) ijOpenSleep() {
	if g.Cur == nil || g.P == nil {
		return
	}
	g.Mode = "ijsleep"
}

func ijSleepKeys(g *Game, k string) {
	kinds := g.ijSleepKinds()
	if k == "Escape" || k == "q" {
		g.Mode = "main"
		return
	}
	if len(k) == 1 && k[0] >= '1' && k[0] <= '9' {
		if i := int(k[0] - '1'); i < len(kinds) {
			g.Mode = "main"
			g.ijSleep(kinds[i], 600)
		}
	}
}

func ijSleepDraw(g *Game) {
	s := g.sc
	kinds := g.ijSleepKinds()
	lines := []string{fmt.Sprintf("Fatigue: %s ({{K|%d/100}})", g.ijFatigueWord(), int(g.ijSt().Fat)), ""}
	for i, k := range kinds {
		lines = append(lines, fmt.Sprintf("{{W|[%d]}} Sleep %s", i+1, ijSleepName[k]))
	}
	lines = append(lines, "", "{{K|[Esc] cancel}}")
	h := len(lines) + 4
	y0 := (MH - h) / 2
	s.Box(20, y0, 50, h, CCyan, "Sleep")
	for i, l := range lines {
		s.TextB(22, y0+2+i, l, CGray, CBgDark, 69)
	}
}

// ijCanSleep checks the common refusals (message printed).
func (g *Game) ijCanSleep() bool {
	if g.nearHostile() {
		g.Msg("You can't sleep with enemies in sight.")
		return false
	}
	if g.ijSt().Fat < 12 {
		g.Msg("You are not tired enough to sleep.")
		return false
	}
	return true
}

// ijSleep sleeps up to maxMin minutes (until rested), passing time through spend() like rest().
func (g *Game) ijSleep(kind string, maxMin int) {
	if !g.ijCanSleep() {
		return
	}
	st := g.ijSt()
	p := g.P
	st.Sleep = kind
	n := 0
	interrupted := false
	for n < maxMin && !g.Dead {
		if n >= 60 && st.Fat <= 2 {
			break
		}
		p.AddSt("asleep", 2)
		g.spend(1000)
		n++
		if (kind == "open" || kind == "camp") && n%60 == 0 && g.R.Chance(0.12) && !g.Dead {
			g.hurt(p, g.R.Int(1, 3), nil, "exposure")
			g.Msg("{{W|The cold ground wakes you for a moment.}}")
		}
		if g.nearHostile() {
			g.Msg("{{R|You are woken!}}")
			interrupted = true
			break
		}
	}
	st.Sleep = ""
	if g.Dead {
		return
	}
	delete(p.Status, "asleep")
	hrs := n / 60
	where := ijSleepName[kind]
	if kind == "bed" && g.homeHere() != nil {
		where = "in your own bed"
	}
	g.Msg(fmt.Sprintf("You sleep %d hours %s.", hrs, where))
	if (kind == "bed" || kind == "inn") && n >= 240 && !interrupted {
		p.AddSt("rested", DayLen)
		g.Msg("{{G|You wake rested.}}")
	} else if !interrupted && kind == "open" {
		g.Msg("You wake stiff and sore.")
	}
	g.ijFirst("slept_"+kind, fmt.Sprintf("%s slept %s for the first time.", p.Name, ijSleepName[kind]))
	if n >= 120 && g.R.Chance(0.2) {
		if d := g.ijDream(); d != "" {
			g.Msg("{{m|" + d + "}}")
		}
	}
}

func ijTrim(s string, n int) string {
	r := []rune(s)
	if len(r) > n {
		return string(r[:n-1]) + "..."
	}
	return s
}

// ijDream is a short dream from the home realm's real history or a hint from the world's secrets.
func (g *Game) ijDream() string {
	var evs []EventData
	for _, e := range g.W.Events {
		if e.St == g.HomeState && e.Text != "" {
			evs = append(evs, e)
		}
	}
	if len(evs) > 0 && (len(g.Secrets) == 0 || g.R.Chance(0.65)) {
		e := evs[g.R.Int(0, len(evs)-1)]
		return fmt.Sprintf("You dream of the year %d: %s", e.Y, ijTrim(e.Text, 150))
	}
	if len(g.Secrets) > 0 {
		s := g.Secrets[g.R.Int(0, len(g.Secrets)-1)]
		return "In your sleep a voice whispers: " + ijTrim(s.Text, 150)
	}
	return ""
}

// ijRoomOption is the innkeeper's "Rent a room" talk option.
func (g *Game) ijRoomOption(a *Actor) Option {
	fee := maxi(1, int(float64(6+g.P.Level*2)*g.ijPriceMod()))
	return Option{fmt.Sprintf("{{Y|Rent a room.}} {{K|(%d %s, a night)}}", fee, g.Sk.Currency), func() {
		if !g.ijCanSleep() {
			g.talkMenu("\"You look wide awake to me.\"")
			return
		}
		if g.P.Gold < fee {
			g.talkMenu("\"A bed costs coin, friend.\"")
			return
		}
		g.P.Gold -= fee
		g.ijSleep("inn", 600)
		if g.Dead {
			return
		}
		g.talkMenu("\"Sleep well?\"")
	}}
}

// ijOnHomeSleep: sleepHome() (home.go) clears fatigue.
func (g *Game) ijOnHomeSleep() {
	st := g.ijSt()
	st.Fat = 0
	g.P.AddSt("rested", DayLen)
	g.ijFirst("slept_bed", g.P.Name+" slept in a bed for the first time on the road.")
}
