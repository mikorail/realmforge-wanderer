package engine

import (
	"fmt"
	"sort"
	"strings"
)

// Social actions in conversation (prefix sa). Five moves per person: persuade, intimidate, bribe, lie, charm.
// Each is a 2d6 check (checks.go) against a difficulty built from the person (level, role, opinion and mood from
// lives.go, temper from their traits, what they know of you from knowledge.go and the nemesis dossier, local
// customs from etiquette.go). The roll breakdown is printed in full. Results spread: everybody who can see you
// remembers it (lvWitness), a rumour record is queued through saQueueRumour (a rumour mill may set saRumourSink),
// townsfolk can turn cold for a few days, and a won persuasion can unlock one local favour (a gate-pass, a free
// night at the inn, a discount for a week, a tip about a hideout, a blessing). Crimes go through crime.go only.
// State g.ext("sa"), planet-local. Gameplay rolls use g.R; nothing here draws g.R to generate content.
//
// Hook outside this file (listed in the report): saPriceMod in talk.go price() (buy side).

// saRumourSink is set by a rumour mill (engine/rumourmill.go) when it exists; nil is fine.
var saRumourSink func(g *Game, kind, subject, text string)

// saQueueRumour hands a rumour record to the rumour mill, or posts a town news line when there is none yet.
func saQueueRumour(g *Game, kind, subject, text string) {
	if g == nil || text == "" {
		return
	}
	if saRumourSink != nil {
		saRumourSink(g, kind, subject, text)
		return
	}
	if g.W != nil && g.Cur != nil {
		lvSt(g).addNews(g.Day(), g.Cur.Burg, "%s", text)
	}
}

func init() {
	unLocalKeys = append(unLocalKeys, "sa")
	talkHooks = append(talkHooks, saTalk)
}

const saDailyCap = 3

type saFavour struct {
	Kind  string `json:"k"` // discount inn
	Burg  int    `json:"b"`
	Until int    `json:"u"` // last day it counts
	Pct   int    `json:"p,omitempty"`
	Who   string `json:"w,omitempty"`
}

type saState struct {
	Last  map[string]int `json:"l,omitempty"` // "actor:move" -> day
	Count map[string]int `json:"c,omitempty"` // "actor:day" -> moves made
	Done  map[string]int `json:"d,omitempty"` // "burg:kind" -> day a favour was given
	Cold  map[int]int    `json:"o,omitempty"` // burg -> last day townsfolk are cold to you
	Favs  []saFavour     `json:"f,omitempty"`
	Won   int            `json:"w,omitempty"`
	Lost  int            `json:"x,omitempty"`
}

func saSt(g *Game) *saState {
	s := g.ext("sa", func() any {
		return &saState{Last: map[string]int{}, Count: map[string]int{}, Done: map[string]int{}, Cold: map[int]int{}}
	}).(*saState)
	if s.Last == nil {
		s.Last = map[string]int{}
	}
	if s.Count == nil {
		s.Count = map[string]int{}
	}
	if s.Done == nil {
		s.Done = map[string]int{}
	}
	if s.Cold == nil {
		s.Cold = map[int]int{}
	}
	day := g.Day()
	if len(s.Count) > 200 || len(s.Last) > 200 {
		s.Count, s.Last = map[string]int{}, map[string]int{}
	}
	keep := s.Favs[:0]
	for _, f := range s.Favs {
		if f.Until >= day {
			keep = append(keep, f)
		}
	}
	s.Favs = keep
	return s
}

// saPriceMod: a discount favour for this town, or a cold town after a failed threat. Buy side only.
func (g *Game) saPriceMod(a *Actor) float64 {
	if g == nil || g.W == nil || g.Cur == nil || g.Cur.Burg <= 0 {
		return 1
	}
	s := saSt(g)
	m := 1.0
	for _, f := range s.Favs {
		if f.Kind == "discount" && f.Burg == g.Cur.Burg && f.Pct > 0 {
			m *= 1 - float64(f.Pct)/100
			break
		}
	}
	if s.Cold[g.Cur.Burg] >= g.Day() {
		m *= 1.15
	}
	return m
}

// ---------------------------------------------------------------- rolls

type saMod struct {
	L string
	V int
}

func saSigned(m []saMod) string {
	var parts []string
	for _, x := range m {
		if x.V == 0 {
			continue
		}
		parts = append(parts, fmt.Sprintf("%s %+d", x.L, x.V))
	}
	if len(parts) == 0 {
		return "no modifiers"
	}
	return strings.Join(parts, ", ")
}

// saRoll rolls 2d6 + skill + mods against the sum of the difficulty parts and returns the visible breakdown.
func (g *Game) saRoll(kind string, dcParts, mods []saMod) (ok bool, margin int, line string) {
	dc := 0
	for _, x := range dcParts {
		dc += x.V
	}
	skill := g.checkMod(kind)
	bonus := 0
	for _, x := range mods {
		bonus += x.V
	}
	d1, d2 := g.R.Dice(1, 6), g.R.Dice(1, 6)
	total := d1 + d2 + skill + bonus
	margin = total - dc
	ok = margin >= 0
	res := "{{R|failure by " + fmt.Sprint(-margin) + "}}"
	if ok {
		res = "{{G|success by " + fmt.Sprint(margin) + "}}"
	}
	line = fmt.Sprintf("{{K|Roll: 2d6 (%d+%d) %+d %s", d1, d2, skill, checkKinds[kind].label)
	if bonus != 0 {
		line += "; " + saSigned(mods)
	}
	line += fmt.Sprintf(" = %d. Difficulty %d (%s).}} %s", total, dc, saSigned(dcParts), res)
	if ok {
		g.gainXP(3 + dc/2)
	}
	return
}

func (g *Game) saChance(kind string, dcParts, mods []saMod) int {
	dc, bonus := 0, 0
	for _, x := range dcParts {
		dc += x.V
	}
	for _, x := range mods {
		bonus += x.V
	}
	return int(chanceFor(g.checkMod(kind)+bonus, dc)*100 + 0.5)
}

func saLabel(g *Game, kind string, dcParts, mods []saMod, text string) string {
	dc := 0
	for _, x := range dcParts {
		dc += x.V
	}
	return fmt.Sprintf("{{c|[%s · %s %d%%]}} %s", checkKinds[kind].label, diffName(dc), g.saChance(kind, dcParts, mods), text)
}

// ---------------------------------------------------------------- who they are

type saWho struct {
	a      *Actor
	p      *lvPerson
	burg   int
	state  int
	ci     int
	cust   *CustomsPack
	op     int
	mood   int
	wealth int
	tr     [5]int
}

func (g *Game) saWho(a *Actor) *saWho {
	w := &saWho{a: a, ci: -1, tr: [5]int{50, 50, 50, 50, 50}}
	w.p = lvPersonOf(g, a)
	if w.p != nil {
		w.op, w.mood, w.wealth, w.tr = w.p.Opinion, w.p.Mood, w.p.Wealth, w.p.Trait
	}
	if g.Cur != nil && g.Cur.Kind == "town" {
		w.burg = g.Cur.Burg
	}
	w.state = g.uwStateHere()
	w.ci, w.cust = g.etHere()
	return w
}

func saArmed(a *Actor) bool {
	if a.Has("armed") || a.Role == "guard" {
		return true
	}
	if p := a.PartFor("hand"); p != nil && p.Item != nil {
		return true
	}
	return false
}

var saRoleDC = map[string]map[string]int{
	"persuade":  {"guard": 2, "elder": 1, "notable": 2, "noble": 2, "governor": 2, "priest": 1, "villager": -1, "innkeeper": -1, "tavern": -1},
	"intimidate": {"guard": 3, "noble": 2, "governor": 3, "priest": 2, "elder": 1, "notable": 2, "villager": -2, "merchant": -1, "innkeeper": -1},
	"bribe":     {"guard": 2, "priest": 4, "elder": 2, "governor": 4, "notable": 1, "merchant": -1, "villager": -2, "innkeeper": -1, "tavern": -1},
	"bluff":     {"guard": 3, "elder": 1, "priest": 1, "notable": 1, "governor": 2, "villager": -1},
	"charm":     {"elder": 1, "noble": 2, "notable": 2, "governor": 2, "priest": 1, "villager": -1, "innkeeper": -1, "tavern": -1},
}

var saBase = map[string]int{"persuade": 9, "intimidate": 9, "bribe": 8, "bluff": 10, "charm": 9}
var saKind = map[string]string{"persuade": "persuasion", "intimidate": "intimidate", "bribe": "streetwise", "bluff": "bluff", "charm": "empathy"}

// saDC builds the visible difficulty parts for a move.
func (g *Game) saDC(w *saWho, move string) []saMod {
	a := w.a
	parts := []saMod{{"base", saBase[move]}}
	if d := clampi((a.Level-g.P.Level)/2, -2, 3); d != 0 {
		parts = append(parts, saMod{"level", d})
	}
	if r := saRoleDC[move][a.Role]; r != 0 {
		parts = append(parts, saMod{"role", r})
	}
	if o := clampi(-w.op/25, -3, 3); o != 0 && move != "intimidate" {
		parts = append(parts, saMod{"their opinion of you", o})
	}
	switch move {
	case "persuade", "charm":
		if m := clampi(-w.mood/40, -2, 2); m != 0 {
			parts = append(parts, saMod{"mood", m})
		}
	case "intimidate":
		if saArmed(a) {
			parts = append(parts, saMod{"armed", 2})
		}
		if w.tr[3] < 35 {
			parts = append(parts, saMod{"proud", 1})
		}
		if w.tr[4] > 60 {
			parts = append(parts, saMod{"timid", -2})
		}
		if a.Level+3 < g.P.Level {
			parts = append(parts, saMod{"weak", -2})
		}
	case "bribe":
		if w.tr[1] >= 65 {
			parts = append(parts, saMod{"honest", 2})
		}
		if w.wealth > 300 {
			parts = append(parts, saMod{"wealthy", 1})
		}
	case "bluff":
		if g.saHeardOfYou(w) {
			parts = append(parts, saMod{"they have heard of you", 2})
		}
		if n := nxSt(g).Nem; nxActive(n) && w.burg > 0 && n.Burg == w.burg {
			parts = append(parts, saMod{"hunter's notices", 2})
		}
	}
	return parts
}

// saMods: the player's side besides the skill itself.
func (g *Game) saMods(w *saWho, move string) []saMod {
	var m []saMod
	switch move {
	case "persuade", "charm", "bribe":
		if w.cust != nil {
			if n := g.etKnownEtiquette(w.ci, w.cust); n > 0 {
				m = append(m, saMod{"local manners", mini(2, 1+n/3)})
			}
		}
		if rep := g.Rep[uwFac(w.state)]; w.state > 0 && rep != 0 {
			m = append(m, saMod{"standing", clampi(rep/200, -2, 2)})
		}
	case "intimidate":
		if d := g.P.Level - w.a.Level; d != 0 {
			m = append(m, saMod{"level edge", clampi(d/2, -2, 3)})
		}
		if g.etWielding() {
			m = append(m, saMod{"drawn steel", 1})
		}
	}
	return m
}

// saHeardOfYou: this person knows a fact about you (knowledge.go).
func (g *Game) saHeardOfYou(w *saWho) bool {
	if w.p == nil {
		return false
	}
	for id, f := range knSt(g).Facts {
		if f.Subject == "player" && knKnown(w.p, id) {
			return true
		}
	}
	return false
}

func (g *Game) saGuardsSee() bool {
	if g.Cur == nil {
		return false
	}
	state := g.uwStateHere()
	for _, a := range g.Cur.Actors {
		if a.HP > 0 && uwIsGuardOf(a, state) && maxi(abs(a.X-g.P.X), abs(a.Y-g.P.Y)) <= 14 && g.losClear(a.X, a.Y, g.P.X, g.P.Y) {
			return true
		}
	}
	return false
}

func (g *Game) saOthersSee(a *Actor) bool {
	for _, o := range g.etWitnesses() {
		if o != a {
			return true
		}
	}
	return false
}

// ---------------------------------------------------------------- bookkeeping

func (g *Game) saUsed(a *Actor) int  { return saSt(g).Count[fmt.Sprintf("%d:%d", a.ID, g.Day())] }
func (g *Game) saDone(a *Actor, move string) bool {
	d, ok := saSt(g).Last[fmt.Sprintf("%d:%s", a.ID, move)]
	return ok && d == g.Day()
}

func (g *Game) saSpend(a *Actor, move string) {
	s := saSt(g)
	s.Count[fmt.Sprintf("%d:%d", a.ID, g.Day())]++
	s.Last[fmt.Sprintf("%d:%s", a.ID, move)] = g.Day()
}

func (g *Game) saRemember(w *saWho, kind, text string, weight int) {
	if w.p != nil {
		g.lvMemoryAdd(w.p, kind, text, weight)
	}
}

// saSpread: everyone who can see you remembers, and a rumour record goes out.
func (g *Game) saSpread(w *saWho, kind, text string, weight int, rumour string) {
	g.lvWitness(kind, text, weight)
	if rumour != "" {
		subject := "player"
		if w.p != nil {
			subject = w.p.Key
		}
		saQueueRumour(g, kind, subject, rumour)
	}
}

func (g *Game) saCold(w *saWho, days int, why string) {
	if w.burg <= 0 {
		return
	}
	s := saSt(g)
	s.Cold[w.burg] = g.Day() + days
	g.Msg("{{R|" + why + " The people of " + g.knBurgName(w.burg) + " turn cold toward you for a few days.}}")
	if w.state > 0 {
		g.AddRep(uwFac(w.state), -10, " for troubling the town")
	}
}

// ---------------------------------------------------------------- the talk hook

func saTalk(g *Game, a *Actor) []Option {
	if g == nil || g.W == nil || g.Cur == nil || g.P == nil || a == nil || a.HP <= 0 || a.Tpl != "npc" || a.AI == "hostile" || a.AI == "follower" || a.Has("wfobj") {
		return nil
	}
	if g.Hostile(a) {
		return nil
	}
	var out []Option
	w := g.saWho(a)
	if g.saUsed(a) >= saDailyCap {
		return []Option{{"{{K|(They are tired of your few words today.)}}", func() { g.talkMenu("\"Enough. Come back tomorrow.\"") }}}
	}
	out = append(out, Option{"Speak a few words to win them round.", func() { g.saPersuadeMenu(w) }})
	if a.Role != "priest" {
		out = append(out, Option{"Gesture at your blade and lean in close.", func() { g.saIntimidateMenu(w) }})
		out = append(out, Option{"Gesture at your purse and make an offer.", func() { g.saBribeMenu(w) }})
	}
	if g.crVow("tr") != nil {
		out = append(out, Option{"{{K|(Make up a story: your vow of Truth forbids it.)}}", func() { g.talkMenu("\"Yes?\"") }})
	} else {
		out = append(out, Option{"Make up a story to tell them.", func() { g.saBluffMenu(w) }})
	}
	out = append(out, Option{"Trade news and pay them a compliment.", func() { g.saCharmMenu(w) }})
	out = append(out, saRedeem(g, a)...)
	return out
}

func (g *Game) saShow(text string, opts []Option) {
	g.Popup = Wrap(text, 100)
	g.Opts = append(opts, Option{"Back.", func() { g.talkMenu("\"Well?\"") }})
	g.Sel = 0
}

func saRedeem(g *Game, a *Actor) []Option {
	if g.Cur == nil || g.Cur.Burg <= 0 {
		return nil
	}
	switch a.Role {
	case "innkeeper", "tavern", "lw_innkeeper":
	default:
		return nil
	}
	s := saSt(g)
	for i, f := range s.Favs {
		if f.Kind == "inn" && f.Burg == g.Cur.Burg {
			i := i
			return []Option{{"{{G|Claim the free bed you were promised.}}", func() {
				if g.nearHostile() {
					g.talkMenu("\"Not with trouble at the door.\"")
					return
				}
				st := saSt(g)
				if i < len(st.Favs) {
					st.Favs = append(st.Favs[:i], st.Favs[i+1:]...)
				}
				g.Turn += 480
				g.P.HP = g.P.MaxHP
				delete(g.P.Status, "bleeding")
				g.Mode = "main"
				g.Talk = nil
				g.Msg("{{G|You sleep the night in a good bed, and nobody asks for coin.}}")
			}}}
		}
	}
	return nil
}

// ---------------------------------------------------------------- persuade

func (g *Game) saPersuadeMenu(w *saWho) {
	a := w.a
	dc := g.saDC(w, "persuade")
	mods := g.saMods(w, "persuade")
	var opts []Option
	if !g.saDone(a, "pfav") {
		opts = append(opts, Option{saLabel(g, "persuasion", dc, mods, "Ask a favour. "+saFavourHint(a)), func() { g.saPersuade(w, "pfav", 1) }})
	}
	if !g.saDone(a, "pinfo") {
		d2 := append(append([]saMod{}, dc...), saMod{"a small thing", -2})
		opts = append(opts, Option{saLabel(g, "persuasion", d2, mods, "Ask what they know."), func() { g.saPersuade(w, "pinfo", -2) }})
	}
	if saShopRole(a.Role) && !g.saDone(a, "pdisc") {
		d3 := append(append([]saMod{}, dc...), saMod{"haggling", 1})
		opts = append(opts, Option{saLabel(g, "persuasion", d3, mods, "Ask for a better price."), func() { g.saPersuade(w, "pdisc", 1) }})
	}
	if len(opts) == 0 {
		g.talkMenu("\"We have said all there is to say today.\"")
		return
	}
	g.saShow("\"What do you want of me?\"", opts)
}

func saShopRole(r string) bool {
	switch r {
	case "merchant", "smith", "apothecary", "bookseller", "exotic", "tailor", "innkeeper", "tavern", "lw_innkeeper":
		return true
	}
	return false
}

func saFavourHint(a *Actor) string {
	switch a.Role {
	case "guard":
		return "{{K|(a pass)}}"
	case "innkeeper", "tavern", "lw_innkeeper":
		return "{{K|(a night's bed)}}"
	case "priest":
		return "{{K|(a blessing)}}"
	case "merchant", "smith", "apothecary", "bookseller", "exotic", "tailor":
		return "{{K|(a week's discount)}}"
	}
	return "{{K|(a tip)}}"
}

func (g *Game) saPersuade(w *saWho, move string, tweak int) {
	a := w.a
	g.saSpend(a, move)
	dc := g.saDC(w, "persuade")
	if tweak != 0 {
		dc = append(dc, saMod{"the ask", tweak})
	}
	if move == "pinfo" {
		dc[len(dc)-1].L = "a small thing"
	}
	ok, margin, line := g.saRoll("persuasion", dc, g.saMods(w, "persuade"))
	if !ok {
		saSt(g).Lost++
		g.saRemember(w, "refuse", "You pressed them and they refused.", -1)
		txt := Pick(NewRng(fmt.Sprint(a.ID, ":", g.Day())), saRefusals)
		if s := g.dvSayQ(a, "refuse", nil); s != "" {
			txt = s
		}
		if margin <= -5 {
			g.saRemember(w, "insult", "You pestered them.", -3)
			txt += " They turn away, annoyed."
		}
		g.talkMenu(line + "\n\n" + txt)
		return
	}
	saSt(g).Won++
	g.saRemember(w, "help", "You talked them round.", 2)
	switch move {
	case "pinfo":
		g.talkMenu(line + "\n\n{{G|They lean in.}} " + g.loGossipOpen(a) + g.gossip(a))
	case "pdisc":
		txt := g.saFavour(w, "discount", 10, 3)
		g.talkMenu(line + "\n\n" + txt)
	default:
		g.talkMenu(line + "\n\n" + g.saFavour(w, "", 0, 0))
	}
	g.saSpread(w, "help", "You charmed a favour from "+a.Name+".", 1, "")
}

// saFavour grants one local favour by the person's role. kind "" picks by role.
func (g *Game) saFavour(w *saWho, kind string, pct, days int) string {
	a := w.a
	s := saSt(g)
	day := g.Day()
	burg := w.burg
	if kind == "" {
		switch a.Role {
		case "guard":
			kind = "pass"
		case "innkeeper", "tavern", "lw_innkeeper":
			kind = "inn"
		case "priest":
			kind = "bless"
		case "merchant", "smith", "apothecary", "bookseller", "exotic", "tailor":
			kind, pct, days = "discount", 20, 7
		default:
			kind = "tip"
		}
		if burg <= 0 && (kind == "pass" || kind == "inn" || kind == "discount") {
			kind = "tip"
		}
	}
	key := fmt.Sprintf("%d:%s", burg, kind)
	if d, ok := s.Done[key]; ok && day-d < 10 && kind != "tip" {
		return "\"I did what I could for you already. Not again so soon.\""
	}
	s.Done[key] = day
	quote := ""
	if q := g.dvSayQ(a, "agree", nil); q != "" {
		quote = q + " "
	}
	town := g.knBurgName(burg)
	switch kind {
	case "pass":
		if bn := g.etSt().Ban; bn != nil {
			delete(bn, burg)
		}
		if b := g.uwBounty(w.state); b > 0 {
			g.uwSt().Bounty[w.state] = b * 3 / 4
		}
		it := &Item{ID: g.newID(), Kind: "letter", Name: "gate-pass of " + town, Glyph: '=', F: CWhite, Value: 1,
			Text: "Let the bearer pass the gates of " + town + " unquestioned. Signed by " + a.Name + ", " + nonEmptyCr(a.Title, "of the watch") + "."}
		g.give(it)
		saQueueRumour(g, "favour", w.keyOr(), a.Name+" gave "+g.P.Name+" a gate-pass of "+town+".")
		return quote + "{{G|" + a.Name + " scratches a gate-pass for you. Bans and the guards' suspicion ease in " + town + ".}}"
	case "inn":
		s.Favs = append(s.Favs, saFavour{Kind: "inn", Burg: burg, Until: day + 7, Who: a.Name})
		return quote + "{{G|\"One night, on the house, in the next week. Ask me at the bar.\"}}"
	case "bless":
		g.P.HP = g.P.MaxHP
		delete(g.P.Status, "poisoned")
		delete(g.P.Status, "bleeding")
		return quote + "{{G|" + a.Name + " lays a hand on you and speaks a blessing. Your wounds close.}}"
	case "discount":
		pct = clampi(pct, 5, 30)
		s.Favs = append(s.Favs, saFavour{Kind: "discount", Burg: burg, Until: day + maxi(days, 1), Pct: pct, Who: a.Name})
		return quote + fmt.Sprintf("{{G|\"%d%% off, for %d days, in my name.\" Merchants of %s will honour it.}}", pct, maxi(days, 1), town)
	default:
		text, p, ok := g.nearestSite()
		if ok {
			g.addSecret(text, "dungeon", p, g.W.Year)
			return quote + "{{G|\"Keep it to yourself: " + text + "\" It goes into your journal.}}"
		}
		return quote + g.loGossipOpen(a) + g.gossip(a)
	}
}

func (w *saWho) keyOr() string {
	if w.p != nil {
		return w.p.Key
	}
	return "player"
}

// ---------------------------------------------------------------- intimidate

func (g *Game) saIntimidateMenu(w *saWho) {
	a := w.a
	if g.Rep[a.Faction] >= 250 {
		g.talkMenu("\"You would threaten me, after everything? No.\"")
		return
	}
	edge := g.P.Level+1 >= a.Level || g.P.Score(STR) >= a.Score(STR)-2
	if !edge {
		g.talkMenu("They look you up and down and almost laugh. You do not have the weight to frighten this one.")
		return
	}
	dc := g.saDC(w, "intimidate")
	mods := g.saMods(w, "intimidate")
	var opts []Option
	if !g.saDone(a, "ainfo") {
		opts = append(opts, Option{saLabel(g, "intimidate", dc, mods, "{{R|Make them tell you what they know.}}"), func() { g.saIntimidate(w, "ainfo") }})
	}
	if saShopRole(a.Role) && !g.saDone(a, "adisc") {
		opts = append(opts, Option{saLabel(g, "intimidate", dc, mods, "{{R|Lean on them for a better price.}}"), func() { g.saIntimidate(w, "adisc") }})
	}
	if a.Gold > 0 && a.Role != "guard" && !g.saDone(a, "acoin") {
		opts = append(opts, Option{saLabel(g, "intimidate", dc, mods, "{{R|Make them hand over coin.}} {{K|(a crime)}}"), func() { g.saIntimidate(w, "acoin") }})
	}
	if len(opts) == 0 {
		g.talkMenu("They have nothing you could squeeze out of them today.")
		return
	}
	g.saShow("You stand a little closer than is polite.", opts)
}

func (g *Game) saIntimidate(w *saWho, move string) {
	a := w.a
	g.saSpend(a, move)
	// breaking the local peace is itself a taboo in some cultures (etiquette.go)
	g.etAct("public_anger", "")
	ok, margin, line := g.saRoll("intimidate", g.saDC(w, "intimidate"), g.saMods(w, "intimidate"))
	if ok {
		saSt(g).Won++
		g.crKarma(0, -1, "frightened "+a.Name)
		g.crKarma(2, 1, "faced down "+a.Name)
		g.saRemember(w, "threat", "You frightened them into it.", -6)
		g.saSpread(w, "insult", "You bullied "+a.Name+" in public.", -2, g.P.Name+" was seen bullying "+a.Name+".")
		switch move {
		case "ainfo":
			g.talkMenu(line + "\n\n\"All right! All right. Listen.\" " + g.loGossipOpen(a) + g.gossip(a))
		case "adisc":
			g.talkMenu(line + "\n\n" + g.saFavour(w, "discount", 25, 3))
		default:
			take := mini(a.Gold, 6+3*a.Level)
			a.Gold -= take
			g.P.Gold += take
			txt := fmt.Sprintf("They count %s into your hand with shaking fingers.", g.cuFormat(take))
			if g.saOthersSee(a) && w.state > 0 {
				g.uwAddBounty(w.state, 10+take/2, " for extortion")
			}
			g.talkMenu(line + "\n\n" + txt)
		}
		return
	}
	saSt(g).Lost++
	g.crKarma(2, -1, "failed to frighten "+a.Name)
	g.saRemember(w, "insult", "You tried to frighten them.", -8)
	proud := w.tr[3] < 35
	if margin <= -3 || saArmed(a) || proud {
		g.saSpread(w, "insult", "You threatened "+a.Name+" and it went badly.", -4, g.P.Name+" threatened "+a.Name+" and was laughed off.")
		if saArmed(a) || margin <= -5 {
			a.Angry = true
			g.Msg("{{R|" + a.Name + " draws steel: they will not be threatened.}}")
			g.saCold(w, 3, "Blood is up.")
			g.Mode = "main"
			g.Talk = nil
			return
		}
		if w.state > 0 && g.saGuardsSee() {
			g.uwAddBounty(w.state, 12, " for menacing a citizen")
		}
		g.saCold(w, 3, a.Name+" shouts that you threatened them.")
		g.talkMenu(line + "\n\n\"Get away from me, or I call the watch!\"")
		return
	}
	g.talkMenu(line + "\n\n\"Is that supposed to scare me? Go on, leave.\"")
}

// ---------------------------------------------------------------- bribe

func (g *Game) saBribeMenu(w *saWho) {
	a := w.a
	if g.saDone(a, "bribe") {
		g.talkMenu("\"You already tried that today. Don't make it worse.\"")
		return
	}
	base := 4 + 3*a.Level
	if w.wealth > 0 {
		base += mini(w.wealth/20, 60)
	}
	tiers := []struct {
		name string
		mul  float64
		dc   int
	}{{"a small purse", 1, 0}, {"a fair purse", 2.5, -2}, {"a heavy purse", 5, -4}}
	dc := g.saDC(w, "bribe")
	mods := g.saMods(w, "bribe")
	var opts []Option
	for _, t := range tiers {
		t := t
		cost := int(float64(base) * t.mul)
		d := append(append([]saMod{}, dc...), saMod{"size of purse", t.dc})
		purpose := "a favour"
		if a.Role == "guard" {
			purpose = "a blind eye"
		}
		opts = append(opts, Option{saLabel(g, "streetwise", d, mods, fmt.Sprintf("Offer %s for %s. {{K|(%s)}}", t.name, purpose, g.cuFormat(cost))),
			func() { g.saBribe(w, cost, t.dc) }})
	}
	g.saShow("You let a few coins show.", opts)
}

func (g *Game) saBribe(w *saWho, cost, tweak int) {
	a := w.a
	if g.P.Gold < cost {
		g.talkMenu("You cannot cover that. " + g.cuFormat(cost) + " would be needed.")
		return
	}
	g.saSpend(a, "bribe")
	dc := append(g.saDC(w, "bribe"), saMod{"size of purse", tweak})
	ok, margin, line := g.saRoll("streetwise", dc, g.saMods(w, "bribe"))
	watchful := a.Role == "guard" || w.tr[1] >= 65 || g.saGuardsSee()
	if ok {
		if !g.cuSpend(cost) {
			g.talkMenu("You cannot cover that.")
			return
		}
		a.Gold += cost
		saSt(g).Won++
		g.crKarma(1, -2, "bribed "+a.Name)
		g.saRemember(w, "bribe", "You bought their help with coin.", 1)
		txt := ""
		if a.Role == "guard" && w.state > 0 {
			if b := g.uwBounty(w.state); b > 0 {
				g.uwSt().Bounty[w.state] = b * 7 / 10
				txt = "\n\n{{G|The guard studies the far wall while the notices on you are quietly lost.}}"
			}
		}
		fav := g.saFavour(w, "", 0, 0)
		g.talkMenu(line + "\n\n\"Coin well met.\" " + fav + txt)
		if g.saOthersSee(a) {
			g.saSpread(w, "insult", "You were seen passing coin to "+a.Name+".", -1, "Folk whisper that "+g.P.Name+" buys officials in "+nonEmptyCr(g.knBurgName(w.burg), "these parts")+".")
		}
		return
	}
	saSt(g).Lost++
	g.saRemember(w, "insult", "You tried to buy them.", -6)
	if watchful {
		g.crKarma(1, -1, "was caught offering a bribe")
		if w.state > 0 {
			g.uwAddBounty(w.state, 10+cost/3, " for attempted bribery")
		}
		g.saSpread(w, "insult", "You tried to bribe "+a.Name+" and were reported.", -3, g.P.Name+" was reported for offering a bribe.")
		g.talkMenu(line + "\n\n\"Put that away! I'll have the watch on you.\"")
		return
	}
	g.talkMenu(line + "\n\n" + fmt.Sprintf("They push your hand away. \"I'm not for sale.\" You keep your coin (%s).", g.cuFormat(cost)) + map[bool]string{true: " They seem offended.", false: ""}[margin <= -4])
}

// ---------------------------------------------------------------- lies

func (g *Game) saBluffMenu(w *saWho) {
	a := w.a
	dc := g.saDC(w, "bluff")
	mods := g.saMods(w, "bluff")
	var opts []Option
	if !g.saDone(a, "lrank") {
		realm := nonEmptyCr(g.uwRealm(w.state), "the crown")
		opts = append(opts, Option{saLabel(g, "bluff", dc, mods, "Claim to act on the authority of "+realm+"."), func() { g.saBluff(w, "lrank") }})
	}
	if w.state > 0 && g.uwBounty(w.state) > 0 && !g.saDone(a, "lali") {
		d := append(append([]saMod{}, dc...), saMod{"the law is looking", 1})
		opts = append(opts, Option{saLabel(g, "bluff", d, mods, "Claim an alibi for the night of the crime."), func() { g.saBluff(w, "lali") }})
	}
	if k := g.saKinTarget(w); k != nil && !g.saDone(a, "lkin") {
		opts = append(opts, Option{saLabel(g, "bluff", dc, mods, "Claim to be kin of "+k.Name+"."), func() { g.saBluff(w, "lkin") }})
	}
	if len(opts) == 0 {
		g.talkMenu("You cannot think of a lie that would fit here.")
		return
	}
	g.saShow("You settle your face into something honest-looking.", opts)
}

// saKinTarget picks a local person to claim kinship with, by hash of seed, day and person.
func (g *Game) saKinTarget(w *saWho) *lvPerson {
	if w.burg <= 0 {
		return nil
	}
	lvEnsure(g, w.burg)
	var ps []*lvPerson
	for _, p := range lvPresent(g, w.burg) {
		if p.Alive && (w.p == nil || p.Key != w.p.Key) && p.Role != "" {
			ps = append(ps, p)
		}
	}
	if len(ps) == 0 {
		return nil
	}
	sort.Slice(ps, func(i, j int) bool { return ps[i].Key < ps[j].Key })
	return ps[int(HashStr(fmt.Sprint(g.Seed, ":sa:kin:", w.a.ID, g.Day()))%uint64(len(ps)))]
}

func (g *Game) saBluff(w *saWho, move string) {
	a := w.a
	g.saSpend(a, move)
	dc := g.saDC(w, "bluff")
	var kin *lvPerson
	switch move {
	case "lali":
		dc = append(dc, saMod{"the law is looking", 1})
	case "lkin":
		kin = g.saKinTarget(w)
		if kin != nil && w.p != nil {
			for _, k := range append(append([]string{}, w.p.Friends...), w.p.Kin...) {
				if k == kin.Key {
					dc = append(dc, saMod{"they know them well", 3})
					break
				}
			}
		}
	}
	ok, margin, line := g.saRoll("bluff", dc, g.saMods(w, "bluff"))
	g.crKarma(1, -2, "told "+a.Name+" a lie")
	if ok {
		saSt(g).Won++
		g.saRemember(w, "lie", "They believed what you told them.", 1)
		switch move {
		case "lrank":
			g.talkMenu(line + "\n\n\"Forgive me. I did not know. Whatever you need.\" " + g.saFavour(w, "", 0, 0))
		case "lali":
			if b := g.uwBounty(w.state); b > 0 {
				g.uwSt().Bounty[w.state] = b * 6 / 10
			}
			g.talkMenu(line + "\n\n\"Then it was someone else, and the notices are wrong. I'll say so.\" {{G|The bounty on you shrinks.}}")
		default:
			txt := "\"A cousin of " + nonEmptyCr(kinName(kin), "theirs") + "? Then you are welcome.\""
			if kin != nil {
				g.saRemember(w, "help", "You claimed kinship with "+kin.Name+".", 3)
			}
			g.talkMenu(line + "\n\n" + txt + "\n\n" + g.saFavour(w, "discount", 10, 3))
		}
		return
	}
	saSt(g).Lost++
	g.saRemember(w, "lie", "They caught you in a lie.", -8)
	text := "You were caught lying to " + a.Name + "."
	switch move {
	case "lrank":
		if a.Role == "guard" && w.state > 0 {
			g.uwAddBounty(w.state, 15, " for false authority")
		} else if w.state > 0 {
			g.AddRep(uwFac(w.state), -10, " for false claims")
		}
		g.talkMenu(line + "\n\n\"Authority? Show me the seal. No? Then get out.\"")
	case "lali":
		if w.state > 0 {
			g.uwSt().Bounty[w.state] += 8
		}
		g.talkMenu(line + "\n\n\"That is not what I heard. Guards! This one is lying!\"")
		if w.state > 0 && g.saGuardsSee() {
			g.uwAlertGuards(w.state, g.Cur)
		}
	default:
		if kin != nil {
			g.lvMemoryAdd(kin, "lie", "Someone falsely claimed to be your kin.", -4)
		}
		g.talkMenu(line + "\n\n\"I know " + nonEmptyCr(kinName(kin), "them") + ", and you are nothing like them.\"")
	}
	g.saSpread(w, "insult", text, -2, g.P.Name+" was caught in a lie in "+nonEmptyCr(g.knBurgName(w.burg), "the road")+".")
	_ = margin
}

func kinName(p *lvPerson) string {
	if p == nil {
		return ""
	}
	return p.Name
}

// ---------------------------------------------------------------- charm and flattery

var saFauxPas = map[string]bool{"touch_head": true, "stare": true, "left_hand": true, "bare_head": true, "turn_back": true, "speak_before_elder": true, "public_anger": true}

func (g *Game) saCharmMenu(w *saWho) {
	a := w.a
	if g.saDone(a, "charm") {
		g.talkMenu("\"You have been very kind already today.\"")
		return
	}
	dc := g.saDC(w, "charm")
	mods := g.saMods(w, "charm")
	var opts []Option
	cu := ""
	if w.ci >= 0 && w.ci < len(g.W.Cultures) {
		cu = g.W.Cultures[w.ci].Name
	}
	// the local manner: one etiquette entry with a greeting, gesture or speech
	var et *Etiquette
	if w.cust != nil {
		for i := range w.cust.Etiquette {
			e := &w.cust.Etiquette[i]
			if e.Kind == "greeting" || e.Kind == "gesture" || e.Kind == "speech" {
				et = e
				break
			}
		}
	}
	if et != nil {
		e := *et
		m := append(append([]saMod{}, mods...), saMod{"local custom", 2 + e.Reward})
		opts = append(opts, Option{saLabel(g, "empathy", dc, m, "Greet them the way it is done among the "+nonEmptyCr(cu, "locals")+"."), func() { g.saCharm(w, "custom", &e) }})
	}
	if w.ci >= 0 {
		if pk := g.W.cpOf(w.ci); pk != nil && pk.Voice != nil && len(pk.Voice.Proverbs) > 0 {
			pv := g.W.cpChoose(pk.Voice.Proverbs, "sa", fmt.Sprint(a.ID), fmt.Sprint(g.Day()))
			m := append(append([]saMod{}, mods...), saMod{"their own proverb", 1})
			opts = append(opts, Option{saLabel(g, "empathy", dc, m, "Offer a saying of their people: \""+knShort(pv, 60)+"\""), func() { g.saCharm(w, "proverb:"+pv, nil) }})
		}
	}
	var faux *Taboo
	if w.cust != nil {
		for i := range w.cust.Taboos {
			if saFauxPas[w.cust.Taboos[i].Act] {
				faux = &w.cust.Taboos[i]
				break
			}
		}
	}
	m := append([]saMod{}, mods...)
	if faux != nil {
		m = append(m, saMod{"unfamiliar manners", -2})
	}
	opts = append(opts, Option{saLabel(g, "empathy", dc, m, "Flatter them in your own way."), func() { g.saCharm(w, "flatter", nil); _ = faux }})
	g.saShow("You look for something true to praise.", opts)
}

func (g *Game) saCharm(w *saWho, how string, e *Etiquette) {
	a := w.a
	g.saSpend(a, "charm")
	dc := g.saDC(w, "charm")
	mods := g.saMods(w, "charm")
	var faux *Taboo
	switch {
	case e != nil:
		mods = append(mods, saMod{"local custom", 2 + e.Reward})
	case strings.HasPrefix(how, "proverb:"):
		mods = append(mods, saMod{"their own proverb", 1})
	default:
		if w.cust != nil {
			for i := range w.cust.Taboos {
				if saFauxPas[w.cust.Taboos[i].Act] {
					faux = &w.cust.Taboos[i]
					mods = append(mods, saMod{"unfamiliar manners", -2})
					break
				}
			}
		}
	}
	ok, margin, line := g.saRoll("empathy", dc, mods)
	if ok {
		saSt(g).Won++
		g.saRemember(w, "charm", "You were gracious and they liked it.", 4)
		if w.p != nil {
			w.p.Mood = clampi(w.p.Mood+10, -100, 100)
		}
		if w.state > 0 {
			g.AddRep(uwFac(w.state), 2, " for good company")
		}
		g.saSpread(w, "help", "You were gracious to "+a.Name+".", 1, "")
		txt := "They smile for the first time. \"You have manners, for a wanderer.\""
		if e != nil {
			txt = cpFill(e.Text, g.etVars(a, w.ci, "")) + "\n\n" + txt
			g.etLearnE(w.ci, e.ID)
		}
		if margin >= 4 {
			txt += "\n\n" + g.loGossipOpen(a) + g.gossip(a)
		} else if margin >= 2 && saShopRole(a.Role) {
			txt += "\n\n" + g.saFavour(w, "discount", 10, 2)
		}
		g.talkMenu(line + "\n\n" + txt)
		return
	}
	saSt(g).Lost++
	g.saRemember(w, "insult", "Your flattery rang false.", -3)
	txt := "\"Hm. Kind of you.\" They do not mean it."
	if faux != nil && margin <= -3 {
		g.etAct(faux.Act, "")
		txt = "You have put a foot wrong: " + cpFill(faux.Text, g.etVars(a, w.ci, "")) + " {{R|Everyone saw it.}}"
		g.saSpread(w, "insult", "You broke a local custom while flattering "+a.Name+".", -3, g.P.Name+" shamed themselves before "+a.Name+" with bad manners.")
	}
	g.talkMenu(line + "\n\n" + txt)
}

// etLearnE marks a courtesy as known (etiquette.go keeps the learn flags private to its own helpers).
func (g *Game) etLearnE(ci int, id string) {
	if ci >= 0 {
		g.etSt().KE[etKey(ci, id)] = true
	}
}
