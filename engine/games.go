package engine

import (
	"fmt"
	"sort"
	"strings"
)

// Games, arena betting and busking (prefix gm). Table games (games_play.go) start from a "Play a
// game" talk option on villagers, merchants and hustlers (Role "gambler", placed by a zone hook with
// side rngs `seed:gm:zoneKey`, one "Gambler of <town>" boss per planet); bookies (Role "bookie") sit
// in arena districts and take bets on staged bouts; item kind "instrument" gives the ability gm_play
// (busking, calming beasts, soothing guards, feasts). Screens use mode "gm". State g.ext("gm").
// Round play may use g.R like checks do; generation uses hashes and side rngs only.

type gmState struct {
	Won, Lost          int
	BookWon, BookLost  int
	Streak             map[int]int       `json:"streak,omitempty"`
	Cool               map[int]int       `json:"cool,omitempty"`
	Beaten             map[int]bool      `json:"beaten,omitempty"`
	Played             map[int]int       `json:"played,omitempty"`
	Spot               map[string]int    `json:"spot,omitempty"`
	FeastDone          map[string]bool   `json:"feastDone,omitempty"`
	BoutDone           map[string]bool   `json:"boutDone,omitempty"`
	Rec                map[string][2]int `json:"rec,omitempty"`
	Deeds              map[string]bool   `json:"deeds,omitempty"`
	Insulted           map[int]bool      `json:"insulted,omitempty"`
	Fame, Busked, Tips int
	CalmTo, CalmQ      int
	Fixed, Caught      int

	tb *gmTable
}

func (g *Game) gm() *gmState {
	st := g.ext("gm", func() any { return &gmState{} }).(*gmState)
	if st.Streak == nil {
		st.Streak = map[int]int{}
	}
	if st.Cool == nil {
		st.Cool = map[int]int{}
	}
	if st.Beaten == nil {
		st.Beaten = map[int]bool{}
	}
	if st.Played == nil {
		st.Played = map[int]int{}
	}
	if st.Spot == nil {
		st.Spot = map[string]int{}
	}
	if st.FeastDone == nil {
		st.FeastDone = map[string]bool{}
	}
	if st.BoutDone == nil {
		st.BoutDone = map[string]bool{}
	}
	if st.Rec == nil {
		st.Rec = map[string][2]int{}
	}
	if st.Deeds == nil {
		st.Deeds = map[string]bool{}
	}
	if st.Insulted == nil {
		st.Insulted = map[int]bool{}
	}
	return st
}

// gmTable is the on-screen state of a menu or a table game.
type gmTable struct {
	mode   int // 0 menu, 2 playing, 3 round over
	title  string
	text   string
	opts   []Option
	esc    func()
	sel    int
	kind   int
	opp    *Actor
	bet    int
	item   *Item
	log    []string
	res    int
	a, b   []int
	w      []int
	n      [12]int
	grid   [49]int
	cur    int
	boss   bool
	seq    []int
	round  int
	wins   int
	losses int
	more   bool
	replay bool
	caught bool
	cheat  bool
	note   string
}

func (t *gmTable) say(s string) {
	t.log = append(t.log, s)
	if len(t.log) > 6 {
		t.log = t.log[len(t.log)-6:]
	}
}

type gmImpl struct {
	start func(g *Game, t *gmTable)
	key   func(g *Game, t *gmTable, k string)
	draw  func(g *Game, t *gmTable, x, y, w int)
	help  string
}

var gmImpls [gmGameCount]gmImpl

func init() {
	modeKeys["gm"] = gmKeys
	modeDraws["gm"] = gmDraw
	talkHooks = append(talkHooks, gmTalk)
	zoneHooks = append(zoneHooks, gmZone)
}

// ---- small helpers ----

func (g *Game) gmSphere() string {
	if g.Cur != nil && g.W != nil {
		if bd := g.W.burgByID[g.Cur.Burg]; bd != nil {
			if cu := g.W.cultByID[bd.Culture]; cu != nil {
				return cu.Sphere
			}
		}
	}
	return ""
}

func (g *Game) gmGameName(kind int) string { return gmNames(g.Genre, g.gmSphere())[kind] }

func gmAbs(a int) int {
	if a < 0 {
		return -a
	}
	return a
}

func (g *Game) gmStateID() int {
	if g.Cur != nil {
		if bd := g.W.burgByID[g.Cur.Burg]; bd != nil {
			return bd.State
		}
	}
	return g.uwStateHere()
}

func (g *Game) gmMoney(n int) string { return g.cuFormat(n) }

func (g *Game) gmDeed(key, text string) {
	st := g.gm()
	if st.Deeds[key] {
		return
	}
	st.Deeds[key] = true
	g.deed("game", text, 0, 0)
}

// gmMenu shows a text with lettered options; esc runs on Escape.
func (g *Game) gmMenu(title, text string, opts []Option, esc func()) {
	st := g.gm()
	st.tb = &gmTable{mode: 0, title: title, text: text, opts: opts, esc: esc}
	g.Mode = "gm"
}

// gmBack returns to the dialogue (or the map when there was none).
func (g *Game) gmBack() {
	st := g.gm()
	st.tb = nil
	if g.Talk != nil {
		g.Mode = "talk"
		g.talkMenu(g.greeting(g.Talk))
		return
	}
	g.Mode = "main"
}

func (g *Game) gmLeave() {
	g.gm().tb = nil
	g.Mode, g.Talk = "main", nil
}

// ---- screens ----

func gmKeys(g *Game, k string) {
	st := g.gm()
	t := st.tb
	if t == nil {
		g.Mode = "main"
		return
	}
	switch t.mode {
	case 0:
		switch k {
		case "ArrowUp":
			t.sel = maxi(0, t.sel-1)
		case "ArrowDown":
			t.sel = mini(len(t.opts)-1, t.sel+1)
		case "Enter", " ":
			if t.sel >= 0 && t.sel < len(t.opts) {
				t.opts[t.sel].Do()
			}
		case "Escape":
			if t.esc != nil {
				t.esc()
			} else {
				g.gmBack()
			}
		default:
			if len(k) == 1 && k[0] >= 'a' && int(k[0]-'a') < len(t.opts) {
				t.opts[k[0]-'a'].Do()
			}
		}
	case 2:
		if k == "Escape" {
			t.say("You push back from the table and fold.")
			g.gmEndRound(t, -1)
			return
		}
		gmImpls[t.kind].key(g, t, k)
	case 3:
		if k == "Enter" || k == "Escape" || k == " " {
			g.gmAfter(t)
		}
	}
}

func gmDraw(g *Game) {
	st := g.gm()
	t := st.tb
	if t == nil {
		g.Mode = "main"
		return
	}
	s := g.sc
	title := t.title
	if t.mode != 0 {
		title = titleCase(g.gmGameName(t.kind))
	}
	x, y, w, h := g.panel(title)
	if t.mode == 0 {
		for _, l := range Wrap(t.text, w) {
			s.TextB(x, y, l, CGray, CBgDark, x+w)
			y++
		}
		y++
		for i, o := range t.opts {
			bg := CBgDark
			if i == t.sel {
				bg = CBgSel
			}
			s.Fill(x, y, w, 1, bg)
			s.TextB(x, y, fmt.Sprintf("{{W|%c)}} %s", 'a'+i, o.Text), CGray, bg, x+w)
			y++
		}
		s.Text(x, y+h-(y-2)-1, "{{K|Up/Down, letter or Enter: choose   Esc: back}}", CGray)
		return
	}
	pot := "no stake"
	if t.bet > 0 {
		pot = "stake " + g.gmMoney(t.bet)
	}
	extra := ""
	if t.boss {
		extra = fmt.Sprintf("  {{Y|series %d-%d, game %d of 3}}", t.wins, t.losses, t.round+1)
	}
	s.Text(x, y, fmt.Sprintf("{{C|vs %s}}  {{K|%s}}%s", t.opp.Name, pot, extra), CGray)
	if t.mode == 2 {
		gmImpls[t.kind].draw(g, t, x, y+2, w)
	}
	ly := y + h - 9
	for i, l := range t.log {
		s.TextB(x, ly+i, l, CGray, CBgDark, x+w)
	}
	if t.mode == 3 {
		s.TextB(x, y+h-2, t.note, CGray, CBgDark, x+w)
		s.Text(x, y+h-1, "{{K|Enter: continue}}", CGray)
	} else {
		s.Text(x, y+h-1, "{{K|"+gmImpls[t.kind].help+"   Esc: fold}}", CGray)
	}
}

// ---- starting a game ----

func (g *Game) gmPurse(a *Actor) int {
	if a.Has("gmboss") {
		return 2000
	}
	return maxi(a.Gold, 10+a.Level*3)
}

func (g *Game) gmStart(a *Actor) {
	st := g.gm()
	g.Talk = a
	if st.Cool[a.ID] > g.Turn {
		g.gmMenu("Table", "\""+Pick(NewRng(fmt.Sprint(a.ID, g.Day())), []string{"Not with you. Not today.", "You have had your luck. Come back tomorrow.", "I know your face and your hands. No."})+"\"", nil, nil)
		return
	}
	if a.Has("gmboss") {
		if g.gm().Beaten[a.ID] {
			g.gmMenu("Table", "\"You took my prize. I have nothing left that you want.\"", nil, nil)
			return
		}
		if st.Won < 60 {
			g.gmMenu("Table", fmt.Sprintf("\"I do not sit with beginners. Win %s at the common tables of this realm, and ask again.\" {{Y|(Won so far: %s)}}", g.gmMoney(60), g.gmMoney(st.Won)), nil, nil)
			return
		}
		g.gmMenu("The Gambler", "\"Three games, three kinds. Best of three takes the pot, and the "+gmPrizeName(g)+" besides.\"", []Option{
			{"{{R|Sit down.}}", func() { g.gmBetMenu(a, 0) }},
			{"Not yet.", func() { g.gmBack() }},
		}, nil)
		return
	}
	names := gmNames(g.Genre, g.gmSphere())
	var opts []Option
	for i := 0; i < gmGameCount; i++ {
		i := i
		opts = append(opts, Option{"{{Y|" + titleCase(names[i]) + "}}", func() { g.gmBetMenu(a, i) }})
	}
	opts = append(opts, Option{"Never mind.", func() { g.gmBack() }})
	g.gmMenu("A game", fmt.Sprintf("%s lays out the stakes. \"What will it be?\" {{K|(Won %s, lost %s at the tables.)}}", g.the(a), g.gmMoney(st.Won), g.gmMoney(st.Lost)), opts, nil)
}

var gmBetSteps = []int{5, 10, 25, 50, 100, 250, 500}

func (g *Game) gmBetMenu(a *Actor, kind int) {
	purse := g.gmPurse(a)
	have := g.cuTotal()
	var opts []Option
	opts = append(opts, Option{"Play for pride (no stake).", func() { g.gmBegin(a, kind, 0, nil) }})
	for _, b := range gmBetSteps {
		b := b
		if b > have || b > purse {
			continue
		}
		opts = append(opts, Option{"Stake " + g.gmMoney(b) + ".", func() { g.gmBegin(a, kind, b, nil) }})
	}
	if !a.Has("gmboss") {
		opts = append(opts, Option{"{{O|Stake an item.}}", func() { g.gmItemMenu(a, kind) }})
	}
	opts = append(opts, Option{"Decline.", func() { g.gmDecline(a) }})
	txt := fmt.Sprintf("%s holds a purse of about %s. How much?", g.the(a), g.gmMoney(purse))
	if a.Has("gmboss") {
		txt = "The pot is yours to name. The prize stays on the table."
	}
	g.gmMenu("The stake", txt, opts, func() { g.gmDecline(a) })
}

func (g *Game) gmDecline(a *Actor) {
	sp := g.gmSphere()
	st := g.gm()
	if a.Role == "gambler" && gmProudSpheres[sp] && !st.Insulted[a.ID] && g.Cur != nil {
		st.Insulted[a.ID] = true
		if bd := g.W.burgByID[g.Cur.Burg]; bd != nil && bd.State > 0 {
			g.AddRep(fmt.Sprintf("state:%d", bd.State), -2, " for refusing a wager")
		}
		line := gmInsultLine[sp]
		if line == "" {
			line = "Declining a wager is an insult here."
		}
		g.Msg("{{y|" + line + "}}")
	}
	g.gmBack()
}

func (g *Game) gmItemMenu(a *Actor, kind int) {
	var its []*Item
	for _, it := range g.P.Inv {
		if it.Value >= 10 && !it.Stack() && it.Quest == 0 {
			its = append(its, it)
		}
	}
	sort.SliceStable(its, func(i, j int) bool { return its[i].Value > its[j].Value })
	if len(its) > 8 {
		its = its[:8]
	}
	purse := g.gmPurse(a)
	var opts []Option
	for _, it := range its {
		it := it
		if it.Value > purse {
			continue
		}
		opts = append(opts, Option{fmt.Sprintf("%s {{K|(worth %s)}}", it.Name, g.gmMoney(it.Value)), func() { g.gmBegin(a, kind, it.Value, it) }})
	}
	opts = append(opts, Option{"Back.", func() { g.gmBetMenu(a, kind) }})
	g.gmMenu("The stake", "They match your item with coin of equal worth.", opts, func() { g.gmBetMenu(a, kind) })
}

func (g *Game) gmBegin(a *Actor, kind, bet int, item *Item) {
	if item == nil && bet > 0 {
		if !g.cuSpend(bet) {
			g.gmMenu("The stake", "You cannot cover that here.", []Option{{"Back.", func() { g.gmBetMenu(a, kind) }}}, nil)
			return
		}
	}
	t := &gmTable{kind: kind, opp: a, bet: bet, item: item}
	if a.Has("gmboss") {
		t.boss = true
		t.bet = bet
		r := NewRng(fmt.Sprint(g.Seed, ":gm:boss:", a.ID))
		perm := []int{0, 1, 2, 3, 4, 5}
		for i := 5; i > 0; i-- {
			j := r.Int(0, i)
			perm[i], perm[j] = perm[j], perm[i]
		}
		t.seq = perm[:3]
		t.kind = t.seq[0]
	}
	g.gm().tb = t
	g.Mode = "gm"
	g.gmStartRound(t)
}

func (g *Game) gmStartRound(t *gmTable) {
	t.mode = 2
	t.log = nil
	t.a, t.b, t.w = nil, nil, nil
	t.n = [12]int{}
	t.grid = [49]int{}
	t.cur, t.cheat, t.caught, t.more, t.replay, t.note = 0, false, false, false, false, ""
	gmImpls[t.kind].start(g, t)
}

// gmCheat: a Streetwise check against the table's eyes. False means you were caught (the round
// ends and the penalty falls); true means the trick passed unnoticed.
func (g *Game) gmCheat(t *gmTable, extra int) bool {
	dc := 7 + t.opp.Level/2 + extra
	if g.roll2d6Check(g.checkMod("streetwise"), dc) {
		return true
	}
	t.caught = true
	t.say("{{R|A hand closes on your wrist. \"Cheat!\"}}")
	g.gmEndRound(t, -1)
	return false
}

func (g *Game) gmEndRound(t *gmTable, res int) {
	st := g.gm()
	st.Played[t.kind]++
	if t.boss {
		switch res {
		case 1:
			t.wins++
		case -1:
			t.losses++
		}
		if t.caught {
			t.losses = 2
		}
		if t.wins < 2 && t.losses < 2 {
			t.more = true
			if res == 0 {
				t.replay = true
				t.note = "A push: the game is replayed."
			} else {
				t.note = fmt.Sprintf("Round %s. The next game is %s.", map[int]string{1: "won", -1: "lost"}[res], g.gmGameName(t.seq[minAt(t.round+1)]))
			}
			t.mode = 3
			return
		}
		if t.wins >= 2 {
			res = 1
		} else {
			res = -1
		}
	}
	g.gmPayout(t, res)
	t.mode = 3
}

func minAt(i int) int { return mini(i, 2) }

func (g *Game) gmPayout(t *gmTable, res int) {
	st := g.gm()
	o := t.opp
	t.res = res
	switch res {
	case 1:
		if t.bet > 0 {
			if t.item == nil {
				g.cuEarn(2 * t.bet)
			} else {
				g.cuEarn(t.bet)
			}
			o.Gold = maxi(0, o.Gold-t.bet)
			st.Won += t.bet
		}
		st.Streak[o.ID]++
		g.gainXP(3 + o.Level/2)
		t.note = "{{G|You win" + map[bool]string{true: " " + g.gmMoney(t.bet), false: ""}[t.bet > 0] + ".}}"
		g.gmDeed("win"+fmt.Sprint(t.kind), fmt.Sprintf("The wanderer %s won at %s in %s.", g.P.Name, g.gmGameName(t.kind), g.Cur.Name))
		if t.bet >= 100 {
			g.gmDeed("big", fmt.Sprintf("The wanderer %s broke a table in %s, winning %d.", g.P.Name, g.Cur.Name, t.bet))
		}
		g.gmStreakAnger(t)
		if t.boss {
			g.gmBossWin(t)
		}
	case 0:
		if t.item == nil && t.bet > 0 {
			g.cuEarn(t.bet)
		}
		t.note = "A push: stakes returned."
	default:
		if t.bet > 0 {
			o.Gold += t.bet
			st.Lost += t.bet
			if t.item != nil {
				g.removeInv(t.item)
			}
		}
		st.Streak[o.ID] = 0
		t.note = "{{R|You lose" + map[bool]string{true: " " + g.gmMoney(t.bet), false: ""}[t.bet > 0] + ".}}"
		if t.caught {
			g.gmPenalty(t)
		}
	}
	g.Turn += 20
	g.cuSync()
}

func (g *Game) gmStreakAnger(t *gmTable) {
	st := g.gm()
	o := t.opp
	n := st.Streak[o.ID]
	if o.Has("gmboss") || o.Role != "gambler" {
		return
	}
	switch {
	case n >= 5:
		o.Angry = true
		t.note += " {{R|\"Nobody wins like that!\" The hustler's friends close in.}}"
		g.Talk = nil
		t.more = false
	case n >= 3:
		st.Cool[o.ID] = g.Turn + 2*DayLen
		t.note += " {{y|\"That is enough. Come back in two days.\"}}"
	}
}

func (g *Game) gmPenalty(t *gmTable) {
	st := g.gm()
	st.Caught++
	fine := t.bet*2 + 10
	if g.cuSpend(fine) {
		t.note += " {{R|Fined " + g.gmMoney(fine) + ".}}"
	} else {
		g.uwAddBounty(g.gmStateID(), 15+t.bet/2, " for cheating at "+g.gmGameName(t.kind))
	}
	st.Cool[t.opp.ID] = g.Turn + 3*DayLen
	if g.R.Chance(0.35) {
		t.opp.Angry = true
		t.note += " {{R|They come at you.}}"
	}
}

func (g *Game) gmAfter(t *gmTable) {
	if t.more {
		if t.replay {
			t.replay = false
		} else {
			t.round++
			t.kind = t.seq[mini(t.round, 2)]
		}
		g.gmStartRound(t)
		return
	}
	g.gm().tb = nil
	if g.R != nil && t.opp.Angry {
		g.Mode, g.Talk = "main", nil
		return
	}
	g.gmBack()
}

// ---- the boss gambler ----

var gmBossCache = map[*WorldData]int{}

func (g *Game) gmBossTown() int {
	if id, ok := gmBossCache[g.W]; ok {
		return id
	}
	best, bs := 0, -1.0
	for i := range g.W.Burgs {
		b := &g.W.Burgs[i]
		if !b.Alive || b.Layer != "surface" {
			continue
		}
		sc := float64(g.W.cityTier[b.ID])*1e6 + b.Pop
		if sc > bs || (sc == bs && b.ID < best) {
			best, bs = b.ID, sc
		}
	}
	gmBossCache[g.W] = best
	return best
}

func gmPrizeName(g *Game) string {
	if bd := g.W.burgByID[g.gmBossTown()]; bd != nil {
		return "Dice-Edge of " + bd.Name
	}
	return "Dice-Edge"
}

func (g *Game) gmPrize() *Item {
	bd := g.W.burgByID[g.gmBossTown()]
	name := "Dice-Edge"
	town := "the old city"
	if bd != nil {
		name, town = "Dice-Edge of "+bd.Name, bd.Name
	}
	r := NewRng(fmt.Sprint(g.Seed, ":gm:prize"))
	base := g.MakeWeapon(Pick(r, []string{"longblade", "axe", "cudgel"}), 4, r)
	return g.MakeUnique(base, name, "Staked on the last throw by the Gambler of "+town+", who never thought to lose it.")
}

func (g *Game) gmBossWin(t *gmTable) {
	st := g.gm()
	if st.Beaten[t.opp.ID] {
		return
	}
	st.Beaten[t.opp.ID] = true
	it := g.gmPrize()
	g.give(it)
	t.note += " {{Y|The Gambler slides the " + it.Name + " across the table.}}"
	g.gainXP(60)
	g.gmDeed("boss", fmt.Sprintf("The wanderer %s beat %s, the Gambler, in %s.", g.P.Name, t.opp.Name, g.Cur.Name))
}

// ---- talk ----

var gmPlayRoles = map[string]bool{"villager": true, "merchant": true, "exotic": true, "smith": true, "gambler": true, "notable": true}

func gmTalk(g *Game, a *Actor) []Option {
	var out []Option
	if g.Cur == nil || g.Cur.Kind != "town" || a.AI == "follower" || g.Hostile(a) {
		return nil
	}
	switch {
	case a.Role == "bookie":
		out = append(out, Option{"{{W|Look at the card and place a bet.}}", func() { g.gmBookie(a) }})
	case a.Has("gmboss"):
		out = append(out, Option{"{{R|Challenge the Gambler.}}", func() { g.gmStart(a) }})
	case gmPlayRoles[a.Role]:
		out = append(out, Option{"{{Y|Play a game.}}", func() { g.gmStart(a) }})
	}
	if a.Role == "gambler" {
		out = append(out, Option{"{{O|Ask about weighted dice.}}", func() { g.gmBuyDice(a) }})
	}
	if a.Role == "merchant" || a.Role == "exotic" || a.Role == "bard" || a.Role == "minstrel" {
		out = append(out, Option{"{{O|Look at instruments.}}", func() { g.gmShop(a) }})
	}
	return out
}

func (g *Game) gmBuyDice(a *Actor) {
	price := 60 + a.Level*4
	nm := gmDiceName[g.Genre]
	if nm == "" {
		nm = "weighted dice"
	}
	g.gmMenu("Weighted dice", fmt.Sprintf("\"Two bones that fall kind to the right hand. Six throws before the wear shows. %s, no questions.\"", g.gmMoney(price)), []Option{
		{"Buy them.", func() {
			if g.cuSpend(price) {
				g.give(&Item{ID: g.newID(), Kind: "gmdice", Name: nm, Glyph: ':', F: CYellow, Charges: 6, Value: price / 2, Weight: 1, Ident: true, Lore: "Loaded for a right-handed thrower."})
				g.gmBack()
			} else {
				g.Msg("You cannot afford them.")
				g.gmBack()
			}
		}},
		{"No.", func() { g.gmBack() }},
	}, nil)
}

// ---- zone placement (side rngs only) ----

func gmZone(g *Game, zn *Zone, _ *Rng) {
	if g.W == nil || zn.Kind != "town" || zn.Z != 0 || zn.Burg <= 0 {
		return
	}
	bd := g.W.burgByID[zn.Burg]
	if bd == nil || !bd.Alive || bd.Layer != "surface" {
		return
	}
	kind, _ := g.districtAt(zn.WX, zn.WY, zn.ZX, zn.ZY)
	r := NewRng(g.Seed + ":gm:" + zn.Key)
	cu := g.W.cultByID[bd.Culture]
	tier := g.W.cityTier[bd.ID]
	if kind == "arena" {
		a := g.makeNPC("bookie", bd, cu, r)
		a.Title = Pick(r, []string{"bookmaker", "odds-man", "tout"})
		a.Desc = fmt.Sprintf("A %s who takes bets on the bouts of %s.", a.Title, bd.Name)
		a.Gold = r.Int(300, 800)
		a.F = CYellow
		plPlaceNPC(g, zn, a, 38, 21)
		return
	}
	here := kind == "market" || kind == "slums" || (tier == 0 && zn.ZX == 1 && zn.ZY == 1)
	if !here || !r.Chance(0.85) {
		return
	}
	cx, cy, ok := 0, 0, false
	if r.Chance(0.6) {
		house := gmHouseName[g.Genre]
		if house == "" {
			house = "dicing hall"
		}
		cx, cy, ok = plBuild(g, zn, r, "The "+titleCase(house)+" of "+bd.Name)
	}
	if !ok {
		cx, cy, ok = zn.FreeSpot(r, func(x, y int) bool { return zn.At(x, y) == TCobble || zn.At(x, y) == TFloor })
	}
	if !ok {
		return
	}
	a := g.makeNPC("gambler", bd, cu, r)
	a.Title = Pick(r, gmHustlerTitleFor(g.Genre))
	a.Desc = fmt.Sprintf("A %s of %s with quick hands and a purse of other people's money.", a.Title, bd.Name)
	a.Gold = r.Int(80, 300) + a.Level*25
	a.F = CYellow
	plPlaceNPC(g, zn, a, cx, cy)
	if bd.ID == g.gmBossTown() && kind == "market" || (bd.ID == g.gmBossTown() && tier == 0) {
		if bx, by, ok2 := zn.FreeSpot(r, func(x, y int) bool { return zn.At(x, y) == TCobble || zn.At(x, y) == TFloor }); ok2 {
			b := g.makeNPC("gambler", bd, cu, r)
			b.Title = "the Gambler of " + bd.Name
			b.Level += 6
			b.MaxHP += 30
			b.Gold = 2000
			b.F = CMagenta
			b.Desc = "The Gambler of " + bd.Name + ": nobody remembers a game they did not win. A unique prize rides on their table."
			b.Set("gmboss")
			plPlaceNPC(g, zn, b, bx, by)
		}
	}
}

func gmHustlerTitleFor(genre string) []string {
	if t, ok := gmHustlerTitle[genre]; ok {
		return t
	}
	return gmHustlerTitle["fantasy"]
}

// ---- arena bookie ----

type gmFighter struct {
	Name     string
	Kind     string
	Lvl      int
	HP, Dmg  int
	Acc, Def int
}

func (g *Game) gmMakeFighter(r *Rng, cu *CultureData, state int) gmFighter {
	lvl := maxi(1, g.zoneLevel(g.Cur.WX, g.Cur.WY, 0)+r.Int(-1, 4))
	f := gmFighter{Lvl: lvl}
	switch r.Int(0, 3) {
	case 0:
		f.Kind, f.Name = "blade", g.cultureName(cu, r)
		f.HP, f.Dmg, f.Acc, f.Def = 24+lvl*7, 4+lvl*13/10, 8+lvl, 9+lvl/2
	case 1:
		f.Kind, f.Name = "guard", g.cultureName(cu, r)+" the champion"
		f.HP, f.Dmg, f.Acc, f.Def = 30+lvl*7, 3+lvl, 7+lvl, 11+lvl/2
	case 2:
		var pool []*PersonData
		for i := range g.W.People {
			p := &g.W.People[i]
			if p.Alive && p.State == state && p.Martial >= 6 {
				pool = append(pool, p)
			}
		}
		if len(pool) > 0 {
			p := pool[r.Int(0, len(pool)-1)]
			f.Kind, f.Name = "sim", p.Name
			f.Lvl = maxi(1, p.Martial+lvl/2)
			f.HP, f.Dmg, f.Acc, f.Def = 26+f.Lvl*6, 4+f.Lvl, 8+f.Lvl, 9+f.Lvl/2
			break
		}
		f.Kind, f.Name = "blade", g.cultureName(cu, r)
		f.HP, f.Dmg, f.Acc, f.Def = 24+lvl*7, 4+lvl*13/10, 8+lvl, 9+lvl/2
	default:
		tpl := tplByID[Pick(r, arenaFoes)]
		f.Kind, f.Name = "beast", "the "+tpl.Name(g.Genre)
		f.HP, f.Dmg, f.Acc, f.Def = tpl.HP+lvl*4, 3+tpl.Lvl+lvl/2, 8+tpl.Lvl+lvl/2, 8+tpl.DV/2
	}
	return f
}

func (g *Game) gmBouts() [2][2]gmFighter {
	key := fmt.Sprintf("%s:gm:card:%s:%d", g.Seed, g.Cur.Key, g.Turn/(DayLen/2))
	var cu *CultureData
	state := 0
	if bd := g.W.burgByID[g.Cur.Burg]; bd != nil {
		cu, state = g.W.cultByID[bd.Culture], bd.State
	}
	var out [2][2]gmFighter
	for i := 0; i < 2; i++ {
		r := NewRng(fmt.Sprint(key, ":", i))
		out[i][0] = g.gmMakeFighter(r, cu, state)
		out[i][1] = g.gmMakeFighter(r, cu, state)
		if out[i][0].Name == out[i][1].Name {
			out[i][1].Name += " the younger"
		}
	}
	return out
}

func (g *Game) gmBoutKey(i int) string {
	return fmt.Sprintf("%s:%d:%d", g.Cur.Key, g.Turn/(DayLen/2), i)
}

// gmSimBout is a pure bout simulation: it returns the winner (0 or 1) and 3-6 narrated lines.
func gmSimBout(f [2]gmFighter, r *Rng, narrate bool) (int, []string) {
	hp := [2]int{f[0].HP, f[1].HP}
	type ev struct {
		text string
		big  bool
	}
	var evs []ev
	first := 0
	if f[1].Acc > f[0].Acc {
		first = 1
	}
	for round := 0; round < 40 && hp[0] > 0 && hp[1] > 0; round++ {
		for k := 0; k < 2 && hp[0] > 0 && hp[1] > 0; k++ {
			a := (first + k) % 2
			d := 1 - a
			p := 0.5 + float64(f[a].Acc-f[d].Def)*0.04
			if p < 0.1 {
				p = 0.1
			}
			if p > 0.9 {
				p = 0.9
			}
			if r.Float() < p {
				dm := maxi(1, f[a].Dmg*(70+r.Int(0, 60))/100)
				hp[d] -= dm
				if narrate {
					tx := Pick(r, gmBlowHit[f[a].Kind])
					evs = append(evs, ev{fmt.Sprintf(tx, f[a].Name, f[d].Name), dm*10 >= f[a].Dmg*11})
				}
			} else if narrate && r.Chance(0.3) {
				evs = append(evs, ev{fmt.Sprintf(Pick(r, gmBlowMiss), f[a].Name), false})
			}
		}
	}
	w := 0
	if hp[0] <= 0 {
		w = 1
	}
	if !narrate {
		return w, nil
	}
	lines := []string{fmt.Sprintf("%s and %s are driven onto the sand. %s", f[0].Name, f[1].Name, Pick(r, gmCrowdLine))}
	var mid []int
	for i := range evs {
		if evs[i].big {
			mid = append(mid, i)
		}
	}
	for i := 0; i < len(evs) && len(mid) < 3; i++ {
		if !evs[i].big {
			mid = append(mid, i)
		}
	}
	sort.Ints(mid)
	if len(mid) > 4 {
		mid = mid[:4]
	}
	for _, i := range mid {
		lines = append(lines, evs[i].text+".")
	}
	lines = append(lines, fmt.Sprintf(Pick(r, gmFinishLine), f[1-w].Name)+fmt.Sprintf(" {{Y|%s wins.}}", f[w].Name))
	return w, lines
}

func gmOdds(f [2]gmFighter, key string) [2]float64 {
	r := NewRng("gm:odds:" + key)
	wins := 0
	for i := 0; i < 160; i++ {
		if w, _ := gmSimBout(f, r, false); w == 0 {
			wins++
		}
	}
	p := float64(wins+4) / 168
	return [2]float64{maxf(1.1, 0.92/p), maxf(1.1, 0.92/(1-p))}
}

func (g *Game) gmBookie(a *Actor) {
	st := g.gm()
	g.Talk = a
	if st.Cool[a.ID] > g.Turn {
		g.gmMenu("The card", "\"I do not take your money any more. Ask about the fix and ask the law.\"", nil, nil)
		return
	}
	bouts := g.gmBouts()
	var opts []Option
	for i := 0; i < 2; i++ {
		i := i
		key := g.gmBoutKey(i)
		f := bouts[i]
		if st.BoutDone[key] {
			opts = append(opts, Option{fmt.Sprintf("{{K|Bout %d: %s v %s (run)}}", i+1, f[0].Name, f[1].Name), func() { g.gmBookie(a) }})
			continue
		}
		od := gmOdds(f, key)
		opts = append(opts, Option{fmt.Sprintf("Bout %d: {{W|%s}} (lv %d%s, x%.2f) v {{W|%s}} (lv %d%s, x%.2f)", i+1,
			f[0].Name, f[0].Lvl, gmRecTag(st, f[0].Name), od[0], f[1].Name, f[1].Lvl, gmRecTag(st, f[1].Name), od[1]),
			func() { g.gmBoutMenu(a, i, f, od, key) }})
	}
	for _, m := range g.Cur.Actors {
		m := m
		if m.Role == "arena" {
			opts = append(opts, Option{"{{R|Enter the arena yourself.}}", func() { g.gmLeave(); g.Talk = m; g.Mode = "talk"; g.startArena(m) }})
			break
		}
	}
	opts = append(opts, Option{"Leave.", func() { g.gmBack() }})
	g.gmMenu("The card", fmt.Sprintf("\"Two bouts today, odds as chalked.\" {{K|Ledger: won %s, lost %s.}}", g.gmMoney(st.BookWon), g.gmMoney(st.BookLost)), opts, nil)
}

func gmRecTag(st *gmState, name string) string {
	if r, ok := st.Rec[name]; ok {
		return fmt.Sprintf(", %d-%d", r[0], r[1])
	}
	return ""
}

func (g *Game) gmBoutMenu(a *Actor, i int, f [2]gmFighter, od [2]float64, key string) {
	var opts []Option
	for s := 0; s < 2; s++ {
		s := s
		opts = append(opts, Option{fmt.Sprintf("Bet on %s.", f[s].Name), func() { g.gmBoutStake(a, i, f, od, key, s, false) }})
	}
	for s := 0; s < 2; s++ {
		s := s
		cost := 20 + f[1-s].Lvl*12
		opts = append(opts, Option{fmt.Sprintf("{{R|Pay %s to make %s throw the bout.}}", g.gmMoney(cost), f[1-s].Name), func() { g.gmFix(a, i, f, od, key, s, cost) }})
	}
	opts = append(opts, Option{"Back.", func() { g.gmBookie(a) }})
	g.gmMenu("The card", fmt.Sprintf("%s v %s", f[0].Name, f[1].Name), opts, func() { g.gmBookie(a) })
}

func (g *Game) gmBoutStake(a *Actor, i int, f [2]gmFighter, od [2]float64, key string, side int, fixed bool) {
	var opts []Option
	for _, b := range gmBetSteps {
		b := b
		if b > g.cuTotal() {
			continue
		}
		opts = append(opts, Option{"Stake " + g.gmMoney(b) + ".", func() { g.gmRunBout(a, i, f, od, key, side, b, fixed) }})
	}
	opts = append(opts, Option{"Back.", func() { g.gmBoutMenu(a, i, f, od, key) }})
	o := od[side]
	if fixed {
		o = maxf(1.15, o*0.7)
	}
	g.gmMenu("The card", fmt.Sprintf("How much on %s? Pays x%.2f.", f[side].Name, o), opts, func() { g.gmBoutMenu(a, i, f, od, key) })
}

func (g *Game) gmFix(a *Actor, i int, f [2]gmFighter, od [2]float64, key string, side, cost int) {
	st := g.gm()
	if !g.cuSpend(cost) {
		g.gmMenu("The card", "You cannot pay the fighter's price.", []Option{{"Back.", func() { g.gmBoutMenu(a, i, f, od, key) }}}, nil)
		return
	}
	if !g.roll2d6Check(g.checkMod("persuasion"), 8+f[1-side].Lvl/3) {
		g.gmMenu("The card", f[1-side].Name+" pockets the coin and laughs at you. The bout is not fixed.", []Option{{"Back.", func() { g.gmBoutMenu(a, i, f, od, key) }}}, nil)
		return
	}
	st.Fixed++
	g.gmBoutStake(a, i, f, od, key, side, true)
}

func (g *Game) gmRunBout(a *Actor, i int, f [2]gmFighter, od [2]float64, key string, side, bet int, fixed bool) {
	st := g.gm()
	if !g.cuSpend(bet) {
		g.gmMenu("The card", "You cannot cover that.", []Option{{"Back.", func() { g.gmBoutMenu(a, i, f, od, key) }}}, nil)
		return
	}
	w, lines := gmSimBout(f, g.R, true)
	text := strings.Join(lines, "\n")
	if fixed {
		if w != side {
			// the thrower goes down on cue: re-narrate in the chosen winner's favour
			w = side
			lines = append(lines[:len(lines)-1], fmt.Sprintf("%s drops his guard at the right moment. {{Y|%s wins.}}", f[1-w].Name, f[w].Name))
			text = strings.Join(lines, "\n")
		}
	}
	for s := 0; s < 2; s++ {
		r := st.Rec[f[s].Name]
		if s == w {
			r[0]++
		} else {
			r[1]++
		}
		st.Rec[f[s].Name] = r
	}
	st.BoutDone[key] = true
	o := od[side]
	if fixed {
		o = maxf(1.15, o*0.7)
	}
	if w == side {
		pay := int(float64(bet) * o)
		g.cuEarn(pay)
		st.BookWon += pay - bet
		text += fmt.Sprintf("\n\n{{G|Your %s wins. You collect %s.}}", f[side].Name, g.gmMoney(pay))
		g.gmDeed("bookwin", fmt.Sprintf("The wanderer %s won on a bout in the arena of %s.", g.P.Name, g.Cur.Name))
	} else {
		st.BookLost += bet
		a.Gold += bet
		text += fmt.Sprintf("\n\n{{R|The book keeps your %s.}}", g.gmMoney(bet))
	}
	if fixed && (g.R.Chance(0.25+0.1*float64(mini(st.Fixed-1, 4))) || w != side) {
		g.uwAddBounty(g.gmStateID(), 40+bet/3, " for fixing a bout")
		st.Cool[a.ID] = g.Turn + 5*DayLen
		text += "\n{{R|The stewards found the fighter's purse. They know who paid.}}"
	}
	g.Turn += 15
	g.cuSync()
	g.gmMenu("The bout", text, []Option{{"Back to the card.", func() { g.gmBookie(a) }}, {"Leave.", func() { g.gmBack() }}}, func() { g.gmBack() })
}

// ---- instruments, busking ----

func (g *Game) gmMakeInstrument(set [4]string, kind, q int) *Item {
	return &Item{ID: g.newID(), Kind: "instrument", Name: gmQualityName[q] + " " + set[kind], Glyph: '&', F: CYellow,
		Value: 30 + q*45, Weight: 2, Ident: true, Extra: map[string]int{"gmq": q, "gmk": kind},
		Lore: "An instrument for playing in squares, taverns and at feasts."}
}

func (g *Game) gmShop(a *Actor) {
	set := gmInstSet(g.Genre, g.gmSphere())
	var opts []Option
	for k := 0; k < 4; k++ {
		for _, q := range []int{1, 2, 3} {
			if (k+q+a.ID)%3 == 0 {
				continue
			}
			k, q := k, q
			price := 25 + q*q*30 + g.plAdj(a, 0)
			opts = append(opts, Option{fmt.Sprintf("%s %s: %s", titleCase(gmQualityName[q]), set[k], g.gmMoney(price)), func() {
				if g.cuSpend(price) {
					g.give(g.gmMakeInstrument(set, k, q))
					g.Msg("You buy a " + gmQualityName[q] + " " + set[k] + ".")
					g.gmDeed("inst", fmt.Sprintf("The wanderer %s took up a %s.", g.P.Name, set[k]))
					g.gmBack()
				} else {
					g.Msg("You cannot afford it.")
				}
			}})
		}
	}
	opts = append(opts, Option{"Back.", func() { g.gmBack() }})
	g.gmMenu("Instruments", "Instruments hang on the wall. Better ones sing longer.", opts, nil)
}

func (g *Game) gmInstrument() *Item {
	var best *Item
	if g.P == nil {
		return nil
	}
	for _, it := range g.P.Inv {
		if it.Kind == "instrument" && (best == nil || it.Extra["gmq"] > best.Extra["gmq"]) {
			best = it
		}
	}
	return best
}

// gmAbilities: hook for Abilities().
func (g *Game) gmAbilities() []Ability {
	if it := g.gmInstrument(); it != nil && g.Cur != nil {
		return []Ability{{ID: "gm_play", Name: "Play " + it.Name, Target: "self"}}
	}
	return nil
}

// gmUse: hook for usePower (ids starting "gm_").
func (g *Game) gmUse(id string, tx, ty int) {
	if id != "gm_play" || g.gmInstrument() == nil {
		return
	}
	g.Talk = nil
	g.gmPlayMenu()
}

// gmFame: renown as a player in 0..100 (read by the family agent).
func (g *Game) gmFame() int { return g.gm().Fame }

// gmCalmBonus: bonus to calming and taming rolls while a calming tune lasts.
func (g *Game) gmCalmBonus() int {
	st := g.gm()
	if st.CalmTo > g.Turn {
		return 1 + st.CalmQ
	}
	return 0
}

func (g *Game) gmCrowd() int {
	if g.Cur == nil || g.Cur.Kind != "town" {
		return 0
	}
	bd := g.W.burgByID[g.Cur.Burg]
	if bd == nil {
		return 0
	}
	n := 4 + g.W.cityTier[bd.ID]*5
	if k, _ := g.districtAt(g.Cur.WX, g.Cur.WY, g.Cur.ZX, g.Cur.ZY); k == "market" || k == "harbor" {
		n += 4
	}
	if _, f := g.feastToday(); f != nil {
		n += 8
	}
	hour := (g.Turn + startMinutes) % DayLen / 60
	if hour < 6 || hour >= 23 {
		n /= 3
	}
	k := mini(g.gm().Spot[fmt.Sprintf("%s:%d", g.Cur.Key, g.Day())], 3)
	return n * (10 - 3*k) / 10
}

func (g *Game) gmPerfMod(it *Item) int {
	return maxi(g.P.M(EGO), g.P.M(AGI)) + it.Extra["gmq"] + g.gm().Fame/25
}

func (g *Game) gmPlayMenu() {
	it := g.gmInstrument()
	st := g.gm()
	crowd := g.gmCrowd()
	var opts []Option
	opts = append(opts, Option{fmt.Sprintf("Play for coin. {{K|(crowd about %d)}}", crowd), func() { g.gmBusk(it, 0) }})
	opts = append(opts, Option{"Play to calm the beasts near you.", func() { g.gmBusk(it, 1) }})
	opts = append(opts, Option{"Play for the guards.", func() { g.gmBusk(it, 2) }})
	if b, f := g.feastToday(); f != nil {
		_ = b
		opts = append(opts, Option{"{{W|Play at the " + f.Name + ".}}", func() { g.gmBusk(it, 3) }})
	}
	opts = append(opts, Option{"Put it away.", func() { g.gmLeave() }})
	g.gmMenu("Play", fmt.Sprintf("You tune your %s. {{K|Fame %d, %d performances, %s in tips.}}", it.Name, st.Fame, st.Busked, g.gmMoney(st.Tips)), opts, func() { g.gmLeave() })
}

func (g *Game) gmBusk(it *Item, mode int) {
	st := g.gm()
	back := []Option{{"Play on.", func() { g.gmPlayMenu() }}, {"Enough.", func() { g.gmLeave() }}}
	finish := func(text string) {
		g.Turn += 30
		g.gmMenu("Play", text, back, func() { g.gmLeave() })
	}
	margin := g.R.Dice(2, 6) + g.gmPerfMod(it) - 8
	switch mode {
	case 0:
		crowd := g.gmCrowd()
		if crowd <= 0 {
			finish("No one is near to listen but the wind.")
			return
		}
		st.Busked++
		st.Spot[fmt.Sprintf("%s:%d", g.Cur.Key, g.Day())]++
		if margin < 0 {
			tip := crowd / 5
			if tip > 0 {
				g.cuEarn(tip)
				st.Tips += tip
			}
			finish(fmt.Sprintf("You miss notes and the crowd drifts off. %s thrown at your feet.", map[bool]string{true: g.gmMoney(tip), false: "Nothing is"}[tip > 0]))
			return
		}
		tip := crowd*(2+margin)/3 + 1 + st.Fame/10
		g.cuEarn(tip)
		st.Tips += tip
		gain := 1
		if margin >= 4 {
			gain = 2
		}
		st.Fame = mini(100, st.Fame+gain)
		g.gainXP(2 + margin)
		g.gmDeed("busk", fmt.Sprintf("The wanderer %s played for coin in %s.", g.P.Name, g.Cur.Name))
		finish(fmt.Sprintf("A crowd of %d gathers. You play well; they toss %s into your case. {{K|(Fame %d)}}", crowd, g.gmMoney(tip), st.Fame))
	case 1:
		n := 0
		for _, a := range g.Cur.Actors {
			if a != g.P && a.Tpl != "npc" && (a.Faction == "beasts" || a.Faction == "wildlife") && gmAbs(a.X-g.P.X) <= 12 && gmAbs(a.Y-g.P.Y) <= 12 {
				n++
			}
		}
		if margin < -1 {
			finish("The tune sours and the animals pay it no mind.")
			return
		}
		st.CalmTo, st.CalmQ = g.Turn+60+it.Extra["gmq"]*20, it.Extra["gmq"]/2
		if margin >= 2 {
			for _, a := range g.Cur.Actors {
				if a != g.P && a.AI == "hostile" && !a.Boss && a.Tpl != "npc" && (a.Faction == "beasts" || a.Faction == "wildlife") && a.Level <= g.P.Level+3 && gmAbs(a.X-g.P.X) <= 12 && gmAbs(a.Y-g.P.Y) <= 12 {
					a.AI = "wander"
				}
			}
		}
		g.gmDeed("calm", fmt.Sprintf("The wanderer %s calmed beasts with a %s.", g.P.Name, it.Name))
		finish(fmt.Sprintf("The tune settles %d creature(s) nearby. {{K}Taming and calming go easier for a while.}}", n))
	case 2:
		state := g.gmStateID()
		b := g.uwBounty(state)
		if b > 0 {
			if margin >= 0 {
				cut := mini(b, 10+it.Extra["gmq"]*5+margin*3)
				g.uwSt().Bounty[state] -= cut
				if g.uwSt().Bounty[state] <= 0 {
					g.uwClearBounty(state)
				}
				g.gainXP(4)
				finish(fmt.Sprintf("The guards listen, and one admits your bounty might be forgotten. It falls by %s.", g.gmMoney(cut)))
				return
			}
			finish("The guards listen, unmoved. \"Pretty. Still wanted.\"")
			return
		}
		if margin >= 0 {
			g.AddRep(fmt.Sprintf("state:%d", state), 3, " for playing for the watch")
		}
		finish("The guards nod along and one tosses a coin. No harm done.")
	case 3:
		bd, f := g.feastToday()
		if f == nil {
			finish("The feast is over.")
			return
		}
		key := feastKey(bd, f, g.W.DateOf(nil, g.Day()).Year)
		if st.FeastDone[key] {
			finish("You have already played for this feast.")
			return
		}
		if margin < 0 {
			finish("You fumble the festival tune. Someone laughs, kindly.")
			return
		}
		st.FeastDone[key] = true
		tip := 15 + margin*4 + g.gmCrowd()
		g.cuEarn(tip)
		st.Tips += tip
		st.Fame = mini(100, st.Fame+3)
		g.AddRep(fmt.Sprintf("state:%d", bd.State), 6+margin, " for playing at the "+f.Name)
		g.gainXP(15)
		g.gmDeed("feast", fmt.Sprintf("The wanderer %s played at the %s of %s.", g.P.Name, f.Name, bd.Name))
		finish(fmt.Sprintf("The whole square dances. They pay you %s and cheer your name. {{K|(Fame %d)}}", g.gmMoney(tip), st.Fame))
	}
}
