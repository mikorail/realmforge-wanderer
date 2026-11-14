package engine

// Gentle first-hour tutorial (prefix tu, state g.ext("tu")) and the shared popup menu helper.
// Offered once at the start of a new run (and dismissable forever via the "tu_on" pref and the saved
// Never flag). Steps are detected by polling real state from the turn and sidebar hooks (every render),
// so no other file has to cooperate. While it runs, the hints.go / hints2.go tips it already covers
// are marked shown, so both systems never fire the same tip; skipping the guide gives them back.
// F12, the palette ("Tutorial") or the console command `tutorial` opens the menu: skip step / skip all.

import (
	"fmt"
	"strings"
)

type tuState struct {
	Offered bool     `json:"o,omitempty"`
	Never   bool     `json:"nv,omitempty"`
	On      bool     `json:"on,omitempty"`
	Done    bool     `json:"dn,omitempty"`
	Step    int      `json:"s,omitempty"`
	Q1      []string `json:"q1,omitempty"` // hints.go ids we silenced
	Q2      []string `json:"q2,omitempty"` // hints2.go ids we silenced

	BSet        int    `json:"bs,omitempty"` // step+1 the baseline below was taken for
	BX          int    `json:"bx,omitempty"`
	BY          int    `json:"by,omitempty"`
	BKey        string `json:"bk,omitempty"`
	BEq, BKills int
	BQ, BFood   int
	BHun, BThi  int
	BTurn       int `json:"bt,omitempty"`
	prev        string
	lastTurn    int
	tGold, tInv int
	seenJ       bool
	warned      map[string]bool
	title, text string
	opts        []Option
	sel         int
}

func (g *Game) tu() *tuState {
	return g.ext("tu", func() any { return &tuState{} }).(*tuState)
}

func init() {
	turnHooks = append(turnHooks, tuTurn)
	enterHooks = append(enterHooks, tuEnter)
	sideHooks = append(sideHooks, tuSide)
	// F12 belongs to the Legacy page (aging.go); the guide opens from the palette and console
	modeKeys["tumenu"] = tuMenuKey
	modeDraws["tumenu"] = tuMenuDraw
	paEntries = append(paEntries, paEntry{Name: "Tutorial guide", Keywords: "tutorial guide help skip learn", Cat: "Help", Label: "", Func: tuMenu})
	cnCmds = append(cnCmds, cnCmd{"tutorial", []string{"guide"}, "tutorial [on|off|skip|restart|status]", "the first-hour guide", false, tuCmd})
}

// tuSeen is the optional one-line hook for screens the polling cannot observe (journal). See report.
func tuSeen(g *Game, what string) {
	if what == "journal" {
		g.tu().seenJ = true
	}
}

func tuTurn(g *Game) { tuPoll(g) }

func tuSide(g *Game) []string {
	tuPoll(g)
	st := g.tu()
	if !st.On || st.Done || g.P == nil || st.Step >= len(tuSteps) {
		return nil
	}
	out := []string{fmt.Sprintf("{{c|Guide %d/%d}} {{K|F12 menu}}", st.Step+1, len(tuSteps))}
	for _, l := range Wrap(tuSteps[st.Step].Obj, 29) {
		out = append(out, "{{Y|"+l+"}}")
	}
	return out
}

func tuEnter(g *Game, zn *Zone) {
	st := g.tu()
	tuPoll(g)
	if !st.On || st.Done || g.P == nil {
		return
	}
	if st.warned == nil {
		st.warned = map[string]bool{}
	}
	switch zn.Kind {
	case "town":
		if !st.warned["town"] {
			st.warned["town"] = true
			g.Msg("{{c|Guide:}} " + zn.Name + " is safe ground: guards, merchants and people to talk to. Return here when you are hurt.")
		}
	case "dungeon", "crypt", "tower", "lair", "ruin":
		if g.P.Level < 3 && !st.warned["deep"] {
			st.warned["deep"] = true
			g.Msg("{{c|Guide:}} {{R|Places like this bite hard at low level.}} Fight near the door, and leave with < if it goes badly.")
		}
	}
}

func tuFoodCount(g *Game) int {
	n := 0
	for _, it := range g.P.Inv {
		switch it.Kind {
		case "food", "water", "tonic":
			n += maxi(1, it.Count)
		}
	}
	return n
}

func tuEqCount(g *Game) int {
	n := 0
	for _, bp := range g.P.Body {
		if bp.Item != nil {
			n++
		}
	}
	return n
}

func tuBase(g *Game, st *tuState) {
	st.BSet = st.Step + 1
	st.BX, st.BY, st.BKey = g.P.X, g.P.Y, g.Cur.Key
	st.BEq, st.BKills, st.BQ, st.BFood = tuEqCount(g), g.Kills, len(g.Quests), tuFoodCount(g)
	st.BHun, st.BThi, st.BTurn = g.Hunger, g.Thirst, g.Turn
	st.seenJ = false
}

func tuOfferOK(g *Game, st *tuState) bool {
	return !st.Offered && !st.Never && !st.On && g.Pref("tu_on") && g.Turn <= 300 && g.Mode == "main" && !g.OnWorld
}

func tuPoll(g *Game) {
	if g.P == nil || g.Cur == nil || g.Dead || g.W == nil {
		return
	}
	st := g.tu()
	mode, prev := g.Mode, st.prev
	st.prev = mode
	if !st.On || st.Done {
		if tuOfferOK(g, st) {
			tuOffer(g)
		}
		return
	}
	if st.Step >= len(tuSteps) {
		st.Done = true
		return
	}
	if st.BSet != st.Step+1 {
		tuBase(g, st)
	}
	id := tuSteps[st.Step].ID
	// trade bookkeeping
	if mode == "trade" && prev != "trade" {
		st.tGold, st.tInv = g.P.Gold, len(g.P.Inv)
	}
	ok := false
	switch id {
	case "move":
		dx, dy := abs(g.P.X-st.BX), abs(g.P.Y-st.BY)
		ok = g.Cur.Key != st.BKey || maxi(dx, dy) >= 5
	case "look":
		ok = mode == "look"
	case "talk":
		ok = mode == "talk" || mode == "trade"
	case "buy":
		if mode == "trade" {
			ok = g.P.Gold < st.tGold && len(g.P.Inv) >= st.tInv
		} else if prev == "trade" {
			ok = g.P.Gold < st.tGold
		}
	case "equip":
		ok = tuEqCount(g) != st.BEq
	case "fight":
		ok = g.Kills > st.BKills
	case "rest":
		ok = tuFoodCount(g) < st.BFood || g.Hunger < st.BHun-50 || g.Thirst < st.BThi-50 ||
			(st.lastTurn > 0 && g.Turn-st.lastTurn >= 8 && mode == "main")
	case "journal":
		ok = st.seenJ || mode == "journal"
	case "map":
		ok = mode == "world"
	case "quest":
		ok = len(g.Quests) > st.BQ
	case "ability":
		ok = mode == "abilities"
	case "palette":
		ok = mode == "pal"
	}
	if s := tuSteps[st.Step]; !ok && s.Soft > 0 && g.Turn-st.BTurn >= s.Soft {
		ok = true
	}
	st.lastTurn = g.Turn
	if !ok {
		if mode == "main" && st.Step < 5 {
			tuWarnHostile(g, st)
		}
		return
	}
	g.Msg("{{G|Guide: done.}}")
	tuAdvance(g, st)
}

func tuWarnHostile(g *Game, st *tuState) {
	if st.warned == nil {
		st.warned = map[string]bool{}
	}
	if st.warned["hostile"] {
		return
	}
	for _, a := range g.Cur.Actors {
		if a != g.P && a.HP > 0 && g.Hostile(a) && a.AI != "follower" && g.visible(a.X, a.Y) {
			st.warned["hostile"] = true
			g.Msg("{{c|Guide:}} Something hostile is in sight. Bump it to attack, or walk away to avoid it.")
			return
		}
	}
}

func tuAdvance(g *Game, st *tuState) {
	st.Step++
	st.BSet = 0
	if st.Step >= len(tuSteps) {
		st.Done = true
		g.Msg("{{G|Guide: that is the whole first hour. Good luck out there. (F12 reopens the guide menu.)}}")
		return
	}
	tuBase(g, st)
	g.Msg("{{c|Guide:}} " + tuSteps[st.Step].Text)
}

// tuQuiet marks the hint ids the tutorial covers as already shown, remembering which it newly marked.
func tuQuiet(g *Game, st *tuState) {
	h, s2 := g.hintState(), g.h2()
	for _, id := range tuQuiet1 {
		if !h.Shown[id] {
			h.Shown[id] = true
			st.Q1 = append(st.Q1, id)
		}
	}
	for _, id := range tuQuiet2 {
		if !s2.Shown[id] {
			s2.Shown[id] = true
			st.Q2 = append(st.Q2, id)
		}
	}
}

func tuRestoreTips(g *Game, st *tuState) {
	h, s2 := g.hintState(), g.h2()
	for _, id := range st.Q1 {
		delete(h.Shown, id)
	}
	for _, id := range st.Q2 {
		delete(s2.Shown, id)
	}
	st.Q1, st.Q2 = nil, nil
}

func tuStart(g *Game, st *tuState, from int) {
	st.On, st.Done, st.Step, st.BSet = true, false, from, 0
	tuQuiet(g, st)
	tuBase(g, st)
	home := g.Cur.Name
	g.Msg("{{c|Guide:}} Starting the first-hour guide. " + tuSteps[from].Text)
	if g.Cur.Kind == "town" && home != "" {
		g.Msg("{{c|Guide:}} " + home + " is your safe starting area. Nothing here will attack you; take your time.")
	}
}

func tuStop(g *Game, st *tuState, never bool) {
	st.On = false
	tuRestoreTips(g, st)
	if never {
		st.Never = true
		g.SetPref("tu_on", false)
	}
}

func tuOffer(g *Game) {
	st := g.tu()
	st.Offered = true
	tuOpen(g, "Guide",
		"Would you like a gentle guided first hour? A short list of small tasks (move, look, talk, buy, equip, fight, rest, journal, map, quests, powers, palette) shown in the sidebar, each detected as you do it. You can skip any step, or stop at any time with F12.",
		[]Option{
			{"{{G|Yes, guide me.}}", func() { g.Mode = "main"; tuStart(g, g.tu(), 0) }},
			{"{{K|Not now.}}", func() { g.Mode = "main" }},
			{"{{R|No, and never ask again.}}", func() { g.Mode = "main"; tuStop(g, g.tu(), true) }},
		})
}

func tuMenu(g *Game) {
	st := g.tu()
	var opts []Option
	text := "The guide is off."
	if st.On && !st.Done && st.Step < len(tuSteps) {
		text = fmt.Sprintf("Step %d of %d: %s", st.Step+1, len(tuSteps), tuSteps[st.Step].Obj)
		opts = append(opts, Option{"{{Y|Skip this step.}}", func() {
			g.Mode = "main"
			tuAdvance(g, st)
		}})
		opts = append(opts, Option{"{{R|Skip the whole tutorial.}}", func() { g.Mode = "main"; tuStop(g, st, false); g.Msg("Guide: stopped.") }})
	} else {
		if st.Done {
			text = "You finished the guide."
		}
		opts = append(opts, Option{"{{G|Start (or restart) the guide.}}", func() { g.Mode = "main"; tuStart(g, st, 0) }})
	}
	if !st.Never {
		opts = append(opts, Option{"{{R|Never offer the tutorial again.}}", func() { g.Mode = "main"; tuStop(g, st, true) }})
	}
	opts = append(opts, Option{"{{K|Close.}}", func() { g.Mode = "main" }})
	tuOpen(g, "Guide", text, opts)
}

func tuOpen(g *Game, title, text string, opts []Option) {
	st := g.tu()
	st.title, st.text, st.opts, st.sel = title, text, opts, 0
	g.Mode = "tumenu"
}

func tuMenuKey(g *Game, k string) {
	st := g.tu()
	n := len(st.opts)
	switch k {
	case "Escape":
		g.Mode = "main"
	case "ArrowUp", "N8":
		st.sel = (st.sel + n - 1) % maxi(n, 1)
	case "ArrowDown", "N2":
		st.sel = (st.sel + 1) % maxi(n, 1)
	case "Enter", " ":
		if st.sel < n {
			st.opts[st.sel].Do()
		}
	default:
		if i := letterIndex(k); i >= 0 && i < n {
			st.opts[i].Do()
		}
	}
}

func tuMenuDraw(g *Game) {
	s, st := g.sc, g.tu()
	lines := Wrap(st.text, 90)
	h := len(lines) + len(st.opts) + 5
	y0 := maxi(1, (MH-h)/2)
	s.Box(8, y0, SW-16, h, CDarkCyan, st.title)
	y := y0 + 1
	for _, l := range lines {
		s.TextB(11, y, l, CGray, CBgDark, SW-10)
		y++
	}
	y++
	for i, o := range st.opts {
		bg := CBgDark
		if i == st.sel {
			bg = CBgSel
		}
		s.Fill(10, y, SW-20, 1, bg)
		s.TextB(10, y, fmt.Sprintf("{{W|%c)}} %s", 'a'+i, o.Text), CGray, bg, SW-10)
		y++
	}
}

func tuCmd(g *Game, a []string) string {
	st := g.tu()
	sub := ""
	if len(a) > 0 {
		sub = strings.ToLower(a[0])
	}
	switch sub {
	case "on", "start", "restart":
		st.Never = false
		g.SetPref("tu_on", true)
		tuStart(g, st, 0)
		return "Guide started."
	case "off", "stop":
		tuStop(g, st, false)
		return "Guide stopped."
	case "never":
		tuStop(g, st, true)
		return "Guide stopped; it will not be offered again."
	case "skip", "next":
		if st.On && !st.Done {
			tuAdvance(g, st)
			return "Step skipped."
		}
		return "The guide is not running."
	}
	if st.On && !st.Done && st.Step < len(tuSteps) {
		return fmt.Sprintf("Guide step %d/%d: %s", st.Step+1, len(tuSteps), tuSteps[st.Step].Obj)
	}
	return "The guide is off. `tutorial on` starts it."
}
