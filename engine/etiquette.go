package engine

import (
	"fmt"
	"sort"
	"strings"
)

// Etiquette, taboos, hospitality and local law (et), all from Pack.Customs. Optional: without a pack
// the module only offers the primer page (built from CultureData.Values/Food/Dress/Feasts).
//
//   - Primer (mode "etp", ability et_customs): customs of the land you stand in; taboos and courtesies show "?"
//     until learned (a rebuke, an elder via etTell, living there, a first greeting).
//   - Detectors poll once per game minute (etTurn): new anger, entering homes/halls, bare head/arms, eating,
//     drinking, wasting food, broken vows; optional hook functions cover trade, work and target mode.
//   - Reactions by severity (etReact): frown, insult (+ surcharge a day), violence/ejection/penalty.
//   - Hospitality (etHosp): at night or when tired a local offers food and a roof; guest right and duties.
//   - Law: etLaw, etLawLine, and the "Stand before the judge" talk option.
//
// State g.ext("et"), planet-local. Text choice never draws g.R (cpChoose).

type etGuest struct {
	Burg   int               `json:"b"`
	Cult   int               `json:"c"`
	Host   int               `json:"h"`
	Name   string            `json:"n"`
	Until  int               `json:"u"`
	Duties []string          `json:"d,omitempty"`
	Text   map[string]string `json:"t,omitempty"`
	Bounty int               `json:"bo,omitempty"`
}

type etOffer struct {
	Burg int    `json:"b"`
	Cult int    `json:"c"`
	Host int    `json:"h"`
	Name string `json:"n"`
	Day  int    `json:"d"`
}

type etState struct {
	KT       map[string]bool `json:"kt,omitempty"`
	KE       map[string]bool `json:"ke,omitempty"`
	Live     map[int]int     `json:"lv,omitempty"`
	Reward   map[string]bool `json:"rw,omitempty"`
	Done     map[string]int  `json:"dn,omitempty"`
	Ban      map[int]int     `json:"bn,omitempty"`
	Sur      map[int]int     `json:"sr,omitempty"`
	OfferDay map[int]int     `json:"od,omitempty"`
	Met      map[string]bool `json:"mt,omitempty"`
	Guest    *etGuest        `json:"g,omitempty"`
	Offer    *etOffer        `json:"o,omitempty"`
	OutKey   string          `json:"ok,omitempty"`
	OutX     int             `json:"ox,omitempty"`
	OutY     int             `json:"oy,omitempty"`
	Broke    int             `json:"br,omitempty"`
	DrinkDay int             `json:"dd,omitempty"`
	DrinkN   int             `json:"dk,omitempty"`

	primed     bool
	prevKey    string
	prevSet    string
	prevAngry  map[int]bool
	prevInv    map[int]etItemSnap
	prevHunger int
	scroll     int
	sel        int
}

type etItemSnap struct {
	Name  string
	Kind  string
	Count int
}

func init() {
	unLocalKeys = append(unLocalKeys, "et")
	modeKeys["etp"] = etKeys
	modeDraws["etp"] = etDraw
	talkHooks = append(talkHooks, etTalk)
	turnHooks = append(turnHooks, etTurn)
	enterHooks = append(enterHooks, etEnter)
}

func (g *Game) etSt() *etState {
	s := g.ext("et", func() any { return &etState{} }).(*etState)
	if s.KT == nil {
		s.KT = map[string]bool{}
	}
	if s.KE == nil {
		s.KE = map[string]bool{}
	}
	if s.Live == nil {
		s.Live = map[int]int{}
	}
	if s.Reward == nil {
		s.Reward = map[string]bool{}
	}
	if s.Done == nil {
		s.Done = map[string]int{}
	}
	if s.Ban == nil {
		s.Ban = map[int]int{}
	}
	if s.Sur == nil {
		s.Sur = map[int]int{}
	}
	if s.OfferDay == nil {
		s.OfferDay = map[int]int{}
	}
	if s.Met == nil {
		s.Met = map[string]bool{}
	}
	return s
}

// ---------------------------------------------------------------- where and who

func (w *WorldData) etCultIdx(cu *CultureData) int {
	if w == nil || cu == nil {
		return -1
	}
	for i := range w.Cultures {
		if &w.Cultures[i] == cu {
			return i
		}
	}
	return -1
}

func (g *Game) etBurg() int {
	if g.Cur != nil && g.Cur.Kind == "town" && g.Cur.Burg > 0 {
		return g.Cur.Burg
	}
	return 0
}

// etHere: culture index of the place you stand in (-1 none) and its customs (nil without a pack).
func (g *Game) etHere() (int, *CustomsPack) {
	if g.W == nil || g.Cur == nil || g.OnWorld {
		return -1, nil
	}
	var cu *CultureData
	if b := g.W.burgByID[g.etBurg()]; b != nil {
		cu = g.W.cultByID[b.Culture]
	} else {
		cu = g.localCulture()
	}
	ci := g.W.etCultIdx(cu)
	if ci < 0 {
		return -1, nil
	}
	if p := g.W.cpOf(ci); p != nil {
		return ci, p.Customs
	}
	return ci, nil
}

func (g *Game) etOwnCulture() int {
	if s := g.W.stateByID[g.HomeState]; s != nil {
		return g.W.etCultIdx(g.W.cultByID[s.Culture])
	}
	return -1
}

func (g *Game) etState() int {
	if g.W == nil || g.Cur == nil {
		return 0
	}
	return g.uwStateHere()
}

func (g *Game) etWielding() bool {
	for _, p := range g.P.Body {
		if p.Slot == "hand" && !p.Severed && p.Item != nil {
			k := p.Item.Kind
			if k == "weapon" || k == "ranged" || k == "artifact" && p.Item.DN > 0 {
				return true
			}
		}
	}
	return false
}

// etSetting reports the kind of place the player stands in and whether it is a roofed or sacred interior.
func (g *Game) etSetting() (map[string]bool, bool) {
	set := map[string]bool{}
	z := g.Cur
	if z == nil || g.P == nil {
		return set, false
	}
	if z.Kind != "town" {
		set["road"] = true
		return set, false
	}
	dist := ""
	if g.W != nil {
		dist, _ = g.districtAt(z.WX, z.WY, z.ZX, z.ZY)
	}
	roof := roofed(z.At(g.P.X, g.P.Y))
	indoors := roof
	switch dist {
	case "temple":
		set["temple"] = true
		indoors = true
	case "palace", "academy":
		set["hall"] = true
		indoors = true
	case "market":
		set["market"] = true
		if roof {
			set["home"] = false
		}
	case "homes", "slums":
		if roof {
			set["home"] = true
		}
	default:
		if roof {
			near := ""
			bd := 99
			for _, a := range z.Actors {
				if d := maxi(abs(a.X-g.P.X), abs(a.Y-g.P.Y)); d < bd && d <= 6 && a.Tpl == "npc" {
					bd, near = d, a.Role
				}
			}
			switch near {
			case "priest":
				set["temple"] = true
			case "elder", "noble", "governor":
				set["hall"] = true
			default:
				set["home"] = true
			}
		}
	}
	if b := g.feastTown(); b != nil && g.W.FeastOn(b, g.Day()) != nil {
		set["feast"] = true
	}
	if st := g.etSt(); set["feast"] || st.Guest != nil && st.Guest.Burg == z.Burg {
		set["table"] = true
	}
	return set, indoors
}

func (g *Game) etWitnesses() []*Actor {
	var out []*Actor
	if g.Cur == nil {
		return nil
	}
	for _, a := range g.Cur.Actors {
		if a.Tpl != "npc" || a.HP <= 0 || a.AI == "hostile" || a.AI == "follower" || a.Faction == "player" {
			continue
		}
		if maxi(abs(a.X-g.P.X), abs(a.Y-g.P.Y)) > 14 || !g.visible(a.X, a.Y) {
			continue
		}
		out = append(out, a)
	}
	sort.Slice(out, func(i, j int) bool {
		di := maxi(abs(out[i].X-g.P.X), abs(out[i].Y-g.P.Y))
		dj := maxi(abs(out[j].X-g.P.X), abs(out[j].Y-g.P.Y))
		if di != dj {
			return di < dj
		}
		return out[i].ID < out[j].ID
	})
	if len(out) > 3 {
		out = out[:3]
	}
	return out
}

func (g *Game) etVars(a *Actor, ci int, item string) map[string]string {
	v := map[string]string{"item": item, "time": g.timeOfDay()}
	if a != nil {
		v["name"], v["title"] = a.Name, a.Title
	}
	if ci >= 0 && ci < len(g.W.Cultures) {
		v["culture"] = g.W.Cultures[ci].Name
	}
	if b := g.W.burgByID[g.etBurg()]; b != nil {
		v["town"] = b.Name
		if s := g.W.stateByID[b.State]; s != nil {
			v["realm"] = s.Name
			if r := g.W.relByID[s.Religion]; r != nil {
				v["faith"], v["deity"] = r.Name, r.Deity
			}
		}
	}
	return v
}

// ---------------------------------------------------------------- knowledge

func etKey(ci int, id string) string { return fmt.Sprintf("%d:%s", ci, id) }

func (g *Game) etKnowT(ci int, id string) bool { return g.etSt().KT[etKey(ci, id)] }
func (g *Game) etKnowE(ci int, id string) bool { return g.etSt().KE[etKey(ci, id)] }

func (g *Game) etLearnT(ci int, id string) { g.etSt().KT[etKey(ci, id)] = true }

// etTell is the hook for elders, books and rumours: it teaches up to n unknown taboos and courtesies of a
// culture (kind "taboo", "etiquette" or "" for both) and returns the lines told ("" when nothing is new).
func (g *Game) etTell(ci, n int, kind string) string {
	p := g.W.cpOf(ci)
	if p == nil || p.Customs == nil || n <= 0 {
		return ""
	}
	st := g.etSt()
	var out []string
	if kind != "etiquette" {
		for _, t := range p.Customs.Taboos {
			if n > 0 && !st.KT[etKey(ci, t.ID)] {
				st.KT[etKey(ci, t.ID)] = true
				out = append(out, "It is forbidden to this people: "+t.Text)
				n--
			}
		}
	}
	if kind != "taboo" {
		for _, e := range p.Customs.Etiquette {
			if n > 0 && !st.KE[etKey(ci, e.ID)] {
				st.KE[etKey(ci, e.ID)] = true
				out = append(out, "Good manners here: "+e.Text)
				n--
			}
		}
	}
	return strings.Join(out, "\n")
}

func (g *Game) etLearnOne(ci int, cust *CustomsPack, parts ...string) string {
	st := g.etSt()
	var cand []string
	for _, e := range cust.Etiquette {
		if !st.KE[etKey(ci, e.ID)] {
			cand = append(cand, "e:"+e.ID)
		}
	}
	for _, t := range cust.Taboos {
		if !st.KT[etKey(ci, t.ID)] && t.Severity <= 1 {
			cand = append(cand, "t:"+t.ID)
		}
	}
	if len(cand) == 0 {
		return ""
	}
	pick := g.W.cpChoose(cand, append([]string{"etlearn", fmt.Sprint(ci)}, parts...)...)
	if strings.HasPrefix(pick, "e:") {
		st.KE[etKey(ci, pick[2:])] = true
		for _, e := range cust.Etiquette {
			if e.ID == pick[2:] {
				return e.Text
			}
		}
	}
	st.KT[etKey(ci, pick[2:])] = true
	for _, t := range cust.Taboos {
		if t.ID == pick[2:] {
			return t.Text
		}
	}
	return ""
}

// ---------------------------------------------------------------- acts and reactions

// etAct: the player did act here. Witnesses react per matching taboo. Returns true if a taboo fired.
func (g *Game) etAct(act string, item string, extra ...string) bool {
	ci, cust := g.etHere()
	if cust == nil || g.P == nil {
		return false
	}
	set, _ := g.etSetting()
	for _, x := range extra {
		set[x] = true
	}
	wit := g.etWitnesses()
	if len(wit) == 0 {
		return false
	}
	st := g.etSt()
	day := g.Day()
	fired := false
	for _, t := range cust.Taboos {
		if t.Act != act {
			continue
		}
		if t.Where != "" && !set[t.Where] {
			continue
		}
		if t.Where == "" && act == "draw_weapon_indoors" && !(set["temple"] || set["hall"] || set["home"] || set["feast"]) {
			continue
		}
		key := fmt.Sprintf("%d:%s:%d", g.etBurg(), t.ID, day)
		if _, done := st.Done[key]; done {
			continue
		}
		st.Done[key] = day
		g.etReact(ci, t, wit, item)
		fired = true
	}
	return fired
}

func (g *Game) etRebukeText(ci int, t Taboo, a *Actor, item string) string {
	txt := t.Rebuke
	if txt == "" {
		sv := clampi(t.Severity, 1, 3)
		txt = g.W.cpChoose(etRebuke[sv], "rebuke", t.ID, fmt.Sprint(g.Day()))
	}
	return cpFill(txt, g.etVars(a, ci, item))
}

func (g *Game) etReact(ci int, t Taboo, wit []*Actor, item string) {
	st := g.etSt()
	sev := clampi(t.Severity, 1, 3)
	st.KT[etKey(ci, t.ID)] = true
	w0 := wit[0]
	line := g.etRebukeText(ci, t, w0, item)
	col := []string{"", "y", "O", "R"}[sev]
	who := w0.Name
	if who == "" {
		who = g.The(w0)
	}
	verb := []string{"", "frowns", "snaps", "shouts"}[sev]
	g.Msg(fmt.Sprintf("{{%s|%s %s: \"%s\"}}", col, who, verb, line))
	g.Msg("{{K|You have learned a local taboo: " + t.Text + "}}")
	dOp := []int{0, -4, -10, -25}[sev]
	why := "You " + etActLabel[t.Act] + "."
	for _, a := range wit {
		if p := lvPersonFor(g, a); p != nil {
			g.lvMemoryAdd(p, "taboo", why, dOp)
		}
	}
	state := g.etState()
	burg := g.etBurg()
	if sev >= 2 && state > 0 {
		g.AddRep(uwFac(state), -30*(sev-1)-(sev-2)*20, " for breaking local custom")
	}
	if sev == 2 && burg > 0 {
		st.Sur[burg] = g.Day() + 1
		g.Msg("{{y|Word spreads in the market: merchants will charge you extra for a day.}}")
	}
	if sev == 3 {
		g.etPenalty(ci, t, wit, state, burg)
	}
}

func (g *Game) etPenalty(ci int, t Taboo, wit []*Actor, state, burg int) {
	st := g.etSt()
	p := g.P
	base := 30 + p.Level*8
	switch t.Penalty {
	case "fine", "weregild":
		if t.Penalty == "weregild" {
			base *= 2
		}
		if g.cuSpend(base) {
			g.Msg("{{R|You pay " + g.cuFormat(base) + " as the custom demands.}}")
		} else {
			g.Msg("{{R|You cannot pay " + g.cuFormat(base) + ". The town will not forget the debt.}}")
			if state > 0 {
				g.AddRep(uwFac(state), -40, " for an unpaid fine")
			}
		}
	case "scorn":
		if state > 0 {
			g.AddRep(uwFac(state), -60, " in public scorn")
		}
		if burg > 0 {
			st.Sur[burg] = g.Day() + 3
		}
		g.Msg("{{R|The people turn their backs on you. No one wants your trade.}}")
	case "exile":
		if burg > 0 {
			st.Ban[burg] = g.Day() + 30
		}
		g.Msg("{{R|You are cast out. You may not enter this town for thirty days.}}")
		g.etExpel()
	case "lash":
		n := mini(clampi(p.MaxHP/5, 2, 40), p.HP-1)
		g.Msg("{{R|You are lashed before the townsfolk.}}")
		if n > 0 {
			g.hurt(p, n, nil, "the lash")
		}
	case "ordeal":
		if g.roll2d6Check(g.checkMod("endurance"), 10) {
			g.Msg("{{G|You endure the ordeal. Honour is satisfied.}}")
			g.gainXP(10)
		} else {
			n := mini(clampi(p.MaxHP/4, 2, 50), p.HP-1)
			g.Msg("{{R|The ordeal breaks you. The town is satisfied.}}")
			if n > 0 {
				g.hurt(p, n, nil, "an ordeal")
			}
		}
	case "duel":
		wit[0].Angry = true
		g.Msg("{{R|" + wit[0].Name + " demands satisfaction and draws steel.}}")
	case "hang", "mutilation", "enslave":
		if burg > 0 {
			st.Ban[burg] = g.Day() + 60
		}
		if state > 0 {
			g.uwAlertGuards(state, g.Cur)
		}
		g.Msg("{{R|The guards close on you: the custom calls for " + etPenaltyText[t.Penalty] + ".}}")
	default:
		if burg > 0 {
			st.Ban[burg] = g.Day() + 7
		}
		g.Msg("{{R|You are ordered out of the town.}}")
		g.etExpel()
	}
}

// etExpel walks the player out to the last open zone they stood in.
func (g *Game) etExpel() {
	st := g.etSt()
	if z := g.Zones[st.OutKey]; z != nil && st.OutKey != "" && g.Cur != nil && z.Key != g.Cur.Key {
		g.enterZone(z, st.OutX, st.OutY, true)
	}
}

// etRemote: a taboo broken out of anyone's sight (a vow, a duty, an offer left hanging): rep and the host only.
func (g *Game) etRemote(ci int, act string, host int, hostName string) {
	cust := (*CustomsPack)(nil)
	if p := g.W.cpOf(ci); p != nil {
		cust = p.Customs
	}
	if cust == nil {
		return
	}
	for _, t := range cust.Taboos {
		if t.Act != act {
			continue
		}
		g.etSt().KT[etKey(ci, t.ID)] = true
		sev := clampi(t.Severity, 1, 3)
		txt := g.etRebukeText(ci, t, nil, "")
		g.Msg(fmt.Sprintf("{{O|Word of it reaches the %s: \"%s\"}}", g.W.Cultures[ci].Name, txt))
		if state := g.etState(); state > 0 && sev >= 2 {
			g.AddRep(uwFac(state), -25*sev, " for breaking custom")
		}
		g.lvWitness("taboo", "You "+etActLabel[act]+".", []int{0, -3, -8, -18}[sev])
		return
	}
}

// ---------------------------------------------------------------- per-minute polling

func etTurn(g *Game) {
	if g.W == nil || g.Cur == nil || g.P == nil || g.OnWorld {
		return
	}
	st := g.etSt()
	ci, cust := g.etHere()
	day := g.Day()
	// live among them
	if cust != nil && ci >= 0 {
		st.Live[ci]++
		if n := st.Live[ci]; n%720 == 360 {
			if s := g.etLearnOne(ci, cust, "live", fmt.Sprint(n)); s != "" {
				g.Msg("{{K|Living among the " + g.W.Cultures[ci].Name + ", you pick up a custom: " + s + "}}")
			}
		}
	}
	// angry transitions
	angry := map[int]bool{}
	for _, a := range g.Cur.Actors {
		if a.Angry {
			angry[a.ID] = true
		}
	}
	fresh := st.primed && st.prevKey == g.Cur.Key
	if fresh && cust != nil {
		for _, a := range g.Cur.Actors {
			if !a.Angry || st.prevAngry[a.ID] || a.Tpl != "npc" || a.AI == "guard" || a.Role == "guard" || a.AI == "hostile" || a.AI == "follower" {
				continue
			}
			g.etAct("public_anger", "")
			if _, in := g.etSetting(); g.etWielding() && in {
				g.etAct("draw_weapon_indoors", "")
			}
			break
		}
	}
	if g.Cur != nil && st.Guest != nil && fresh {
		g.etGuestWatch(angry)
	}
	st.prevAngry, st.prevKey = angry, g.Cur.Key
	// setting transitions
	set, indoors := g.etSetting()
	now := ""
	if set["home"] {
		now = "home"
	} else if set["hall"] {
		now = "hall"
	}
	if cust != nil && fresh && st.prevSet == "" && now != "" && ci != g.etOwnCulture() &&
		!(st.Guest != nil && st.Guest.Burg == g.etBurg()) {
		g.etAct("enter_unbidden", "")
	}
	st.prevSet = now
	// attire
	if cust != nil && g.Cur.Kind == "town" && g.Turn%10 == 0 && (indoors || set["feast"] || set["market"]) {
		if h := g.P.PartFor("head"); h != nil && h.Item == nil {
			g.etAct("bare_head", "")
		}
		if b := g.P.PartFor("body"); b != nil && b.Item == nil {
			g.etAct("bare_arms", "")
		}
	}
	etInvPoll(g, st, ci, cust, set)
	// vows
	if b := crKs(g).Broke; b > st.Broke {
		st.Broke = b
		if cust != nil {
			g.etRemote(ci, "break_oath", 0, "")
		}
	} else if b < st.Broke {
		st.Broke = b
	}
	st.primed = true
	etHospTurn(g, st, ci, cust, day)
}

func etInvPoll(g *Game, st *etState, ci int, cust *CustomsPack, set map[string]bool) {
	cur := map[int]etItemSnap{}
	for _, it := range g.P.Inv {
		switch it.Kind {
		case "food", "dish", "ingredient", "corpse", "water", "tonic":
			cur[it.ID] = etItemSnap{it.Name, it.Kind, maxi(it.Count, 1)}
		}
	}
	defer func() { st.prevInv, st.prevHunger = cur, g.Hunger }()
	if st.prevInv == nil || cust == nil || g.Cur == nil || g.Cur.Kind != "town" {
		return
	}
	ate := g.Hunger < st.prevHunger-50
	for id, old := range st.prevInv {
		now := cur[id].Count
		if now >= old.Count {
			continue
		}
		lname := strings.ToLower(old.Name)
		drink := false
		for _, w := range etDrinkWords {
			if strings.Contains(lname, w) {
				drink = true
				break
			}
		}
		switch {
		case drink:
			d := g.Day()
			if st.DrinkDay != d {
				st.DrinkDay, st.DrinkN = d, 0
			}
			st.DrinkN++
			if st.DrinkN >= 3 {
				g.etAct("drink_to_excess", old.Name)
			}
		case ate && (old.Kind == "food" || old.Kind == "dish" || old.Kind == "ingredient" || old.Kind == "corpse"):
			etEaten(g, ci, cust, old.Name, set)
		case !ate && (old.Kind == "food" || old.Kind == "dish"):
			for _, fl := range g.Cur.Items {
				if fl.Name == old.Name && fl.X == g.P.X && fl.Y == g.P.Y {
					g.etAct("waste_food", old.Name)
					break
				}
			}
		}
	}
}

func etEaten(g *Game, ci int, cust *CustomsPack, name string, set map[string]bool) {
	if p := g.W.cpOf(ci); p != nil && p.Material != nil && p.Material.Cuisine != nil {
		ln := strings.ToLower(name)
		for _, u := range p.Material.Cuisine.Unclean {
			if u != "" && strings.Contains(ln, strings.ToLower(u)) {
				g.etAct("eat_unclean", name)
				break
			}
		}
	}
	if set["feast"] || set["table"] {
		near := false
		for _, a := range g.Cur.Actors {
			if a.Tpl == "npc" && a.HP > 0 && a.AI != "hostile" && maxi(abs(a.X-g.P.X), abs(a.Y-g.P.Y)) <= 3 {
				near = true
				break
			}
		}
		if !near {
			g.etAct("eat_alone", name, "table")
		}
	}
}

// ---------------------------------------------------------------- optional hook functions

// etTrade is called from openTrade: haggling at a feast or in a temple, trading on a holy day.
func (g *Game) etTrade(a *Actor) {
	if g.W == nil {
		return
	}
	set, _ := g.etSetting()
	if set["feast"] {
		g.etAct("work_on_holy_day", "")
	}
	if set["feast"] || set["temple"] {
		g.etAct("haggle_sacred", "")
	}
}

// etWork is called when the player crafts or digs (kind "craft" or "dig"): work on a holy day.
func (g *Game) etWork(kind string) {
	if g.W == nil {
		return
	}
	if set, _ := g.etSetting(); set["feast"] {
		g.etAct("work_on_holy_day", kind)
	}
}

// etTarget is called when target mode opens: a drawn weapon under a roof.
func (g *Game) etTarget() {
	if g.W == nil || g.P == nil || !g.etWielding() {
		return
	}
	if _, in := g.etSetting(); in {
		g.etAct("draw_weapon_indoors", "")
	}
}

// etPriceMod is the merchant surcharge after an insult or scorn (1 when none).
func (g *Game) etPriceMod() float64 {
	if g.W == nil || g.Cur == nil {
		return 1
	}
	if until, ok := g.etSt().Sur[g.etBurg()]; ok && g.etBurg() > 0 && g.Day() <= until {
		return 1.2
	}
	return 1
}

// ---------------------------------------------------------------- enter hook

func etEnter(g *Game, zn *Zone) {
	if g.W == nil || g.P == nil {
		return
	}
	st := g.etSt()
	day := g.Day()
	if zn.Kind == "town" && zn.Burg > 0 {
		if until, ok := st.Ban[zn.Burg]; ok && day <= until && st.OutKey != "" && st.OutKey != zn.Key {
			if z := g.Zones[st.OutKey]; z != nil && z.Kind != "town" {
				b := g.W.burgByID[zn.Burg]
				name := "the town"
				if b != nil {
					name = b.Name
				}
				g.Msg("{{R|You are banned from " + name + ". Guards at the gate turn you back.}}")
				g.enterZone(z, st.OutX, st.OutY, true)
				return
			}
		}
	} else {
		st.OutKey, st.OutX, st.OutY = zn.Key, g.P.X, g.P.Y
	}
	if o := st.Offer; o != nil && g.Cur != nil && (zn.Burg != o.Burg || zn.Kind != "town") {
		st.Offer = nil
		g.etRemote(o.Cult, "turn_back", o.Host, o.Name)
	}
	st.prevKey = zn.Key
	st.prevAngry = map[int]bool{}
	for _, a := range zn.Actors {
		if a.Angry {
			st.prevAngry[a.ID] = true
		}
	}
	st.prevSet = ""
}

// ---------------------------------------------------------------- hospitality

func etHospTurn(g *Game, st *etState, ci int, cust *CustomsPack, day int) {
	if gu := st.Guest; gu != nil && day > gu.Until {
		if len(gu.Duties) > 0 {
			g.etRemote(gu.Cult, "shirk_duty", gu.Host, gu.Name)
		}
		st.Guest = nil
	}
	if cust == nil || cust.Hospitality == nil || len(cust.Hospitality.Offers) == 0 {
		return
	}
	burg := g.etBurg()
	if burg == 0 || g.Mode != "main" || g.P.St("asleep") > 0 || g.nearHostile() || st.Guest != nil || st.Offer != nil {
		return
	}
	if st.OfferDay[burg] == day+1 {
		return
	}
	tired := g.ijSt().Fat >= ijTiredAt
	if g.daylight() && !tired {
		return
	}
	state := g.etState()
	if state > 0 && (g.uwBounty(state) > 0 || g.Rep[uwFac(state)] <= -250) {
		return
	}
	if until, ok := st.Ban[burg]; ok && day <= until {
		return
	}
	var host *Actor
	for _, a := range g.etWitnesses() {
		if a.AI != "guard" && a.Role != "guard" && a.Role != "merchant" {
			host = a
			break
		}
	}
	if host == nil {
		return
	}
	st.OfferDay[burg] = day + 1
	if HashStr(g.W.Seed+fmt.Sprintf(":ethosp:%d:%d", burg, day))%100 >= 75 {
		return
	}
	offer := g.W.cpChoose(cust.Hospitality.Offers, "offer", host.Name, fmt.Sprint(day))
	st.Offer = &etOffer{Burg: burg, Cult: ci, Host: host.ID, Name: host.Name, Day: day}
	vars := g.etVars(host, ci, offer)
	line := cpFill(g.W.cpChoose(etOfferLines, "offerline", host.Name, fmt.Sprint(day)), vars)
	who := host.Name
	if host.Title != "" {
		who = host.Title + " " + who
	}
	text := fmt.Sprintf("%s calls to you: \"%s\"", who, line)
	opts := []Option{
		{"Accept the offer of " + offer, func() { g.etAccept(host, ci, cust, offer) }},
		{"Decline with thanks", func() {
			st.Offer = nil
			g.uwClose()
			g.etAct("refuse_hospitality", offer, "table")
		}},
	}
	g.uwOpenMenu("Hospitality", text, opts, false)
}

func etDutyKind(txt string) string {
	l := strings.ToLower(txt)
	for _, k := range etDutyKinds {
		for _, w := range k.Words {
			if strings.Contains(l, w) {
				return k.Key
			}
		}
	}
	return "chore"
}

func (g *Game) etAccept(host *Actor, ci int, cust *CustomsPack, offer string) {
	st := g.etSt()
	h := cust.Hospitality
	st.Offer = nil
	g.uwClose()
	day := g.Day()
	gr := maxi(h.GuestRight, 1)
	gu := &etGuest{Burg: g.etBurg(), Cult: ci, Host: host.ID, Name: host.Name, Until: day + gr, Text: map[string]string{}, Bounty: g.uwBounty(g.etState())}
	for _, d := range h.Duties {
		k := etDutyKind(d)
		if _, dup := gu.Text[k]; dup || len(gu.Duties) >= 3 {
			continue
		}
		gu.Duties = append(gu.Duties, k)
		gu.Text[k] = cpFill(d, g.etVars(host, ci, offer))
	}
	st.Guest = gu
	g.Hunger = maxi(0, g.Hunger-900)
	g.Thirst = 0
	g.Msg("{{G|You eat " + offer + " at " + host.Name + "'s table, a guest by the custom of the " + g.W.Cultures[ci].Name + ".}}")
	if p := lvPersonFor(g, host); p != nil {
		g.lvMemoryAdd(p, "hospitality", "You accepted their hospitality.", 8)
	}
	if g.etHasTaboo(cust, "refuse_gift") {
		parcel := "parcel of food"
		if p := g.W.cpOf(ci); p != nil && p.Material != nil && p.Material.Cuisine != nil && len(p.Material.Cuisine.Staples) > 0 {
			parcel = g.W.cpChoose(p.Material.Cuisine.Staples, "parcel", host.Name) + " parcel"
		}
		txt := host.Name + " presses a " + parcel + " into your hands for the road."
		opts := []Option{
			{"Take the gift with both hands", func() {
				g.uwClose()
				g.give(&Item{ID: g.newID(), Kind: "food", Name: parcel, Glyph: '%', F: CBrown, Count: 1, Value: 4, Power: 400})
				g.etRest(true)
			}},
			{"Refuse the gift", func() {
				g.uwClose()
				g.etAct("refuse_gift", parcel, "table")
				g.etRest(true)
			}},
		}
		g.uwOpenMenu("Hospitality", txt, opts, false)
		return
	}
	g.etRest(false)
}

func (g *Game) etRest(gift bool) {
	g.Msg(fmt.Sprintf("{{c|Guest right: no harm may come to you or from you here for %d day(s). You owe your host a courtesy (talk to them).}}", maxi(g.etSt().Guest.Until-g.Day(), 1)))
	if g.ijSt().Fat >= 12 && !g.nearHostile() {
		g.ijSleep("inn", 480)
	}
}

func (g *Game) etHasTaboo(c *CustomsPack, act string) bool {
	for _, t := range c.Taboos {
		if t.Act == act {
			return true
		}
	}
	return false
}

// etGuestWatch breaks guest right on violence or a new bounty in the guest's town.
func (g *Game) etGuestWatch(angry map[int]bool) {
	st := g.etSt()
	gu := st.Guest
	if gu == nil || g.etBurg() != gu.Burg {
		return
	}
	broke := false
	for _, a := range g.Cur.Actors {
		if a.Angry && !st.prevAngry[a.ID] && a.Tpl == "npc" && a.AI != "guard" && a.Role != "guard" && a.AI != "hostile" && a.AI != "follower" {
			broke = true
		}
	}
	if state := g.etState(); state > 0 && g.uwBounty(state) > gu.Bounty {
		broke = true
	}
	if !broke {
		return
	}
	st.Guest = nil
	breach := "You have broken guest right."
	if p := g.W.cpOf(gu.Cult); p != nil && p.Customs != nil && p.Customs.Hospitality != nil && p.Customs.Hospitality.Breach != "" {
		breach = cpFill(p.Customs.Hospitality.Breach, g.etVars(nil, gu.Cult, ""))
	}
	g.Msg("{{R|Guest right broken: " + breach + "}}")
	if state := g.etState(); state > 0 {
		g.AddRep(uwFac(state), -150, " for breaking guest right")
	}
	st.Ban[gu.Burg] = g.Day() + 30
	g.lvWitness("taboo", "You broke guest right.", -30)
}

// etDutyOptions: talk options to fulfil a guest's duties, on the host (or an elder when the host is away).
func (g *Game) etDutyOptions(a *Actor) []Option {
	st := g.etSt()
	gu := st.Guest
	if gu == nil || gu.Burg != g.etBurg() || len(gu.Duties) == 0 {
		return nil
	}
	if a.ID != gu.Host && a.Role != "elder" {
		return nil
	}
	var out []Option
	for _, k := range gu.Duties {
		k := k
		out = append(out, Option{etDutyOption[k], func() { g.etDoDuty(a, k) }})
	}
	return out
}

func (g *Game) etDoDuty(a *Actor, k string) {
	st := g.etSt()
	gu := st.Guest
	if gu == nil {
		return
	}
	var msg string
	switch k {
	case "tale":
		msg = "You tell a tale by the fire, and the household listens."
	case "news":
		msg = "You share what you have seen on the road. They ask after every detail."
	case "gift":
		cost := 8 + g.P.Level
		if !g.cuSpend(cost) {
			g.talkMenu("You cannot spare a gift worth " + g.cuFormat(cost) + " just now.")
			return
		}
		msg = "You leave a gift worth " + g.cuFormat(cost) + " on the shelf."
	default:
		for i := 0; i < 60 && !g.Dead && !g.nearHostile(); i++ {
			g.spend(1000)
		}
		msg = "You spend an hour at the host's chores: wood, water and mending."
	}
	var rest []string
	for _, d := range gu.Duties {
		if d != k {
			rest = append(rest, d)
		}
	}
	gu.Duties = rest
	if p := lvPersonFor(g, a); p != nil {
		g.lvMemoryAdd(p, "hospitality", "You honoured a guest's duty.", 6)
	}
	if t := gu.Text[k]; t != "" {
		msg += " {{K|(" + t + ")}}"
	}
	if len(rest) == 0 {
		msg += "\n\n{{G|Your duties as a guest are done.}}"
		if state := g.etState(); state > 0 {
			g.AddRep(uwFac(state), 15, " for a guest well kept")
		}
	}
	g.talkMenu(msg)
}

// ---------------------------------------------------------------- etiquette in talk

func etTalk(g *Game, a *Actor) []Option {
	if g.W == nil || g.Cur == nil || g.P == nil || a == nil {
		return nil
	}
	ci, cust := g.etHere()
	if cust == nil || a.Tpl != "npc" || a.AI == "hostile" || a.AI == "follower" {
		return nil
	}
	st := g.etSt()
	// first meeting teaches one custom
	mk := fmt.Sprintf("%d:%d", g.etBurg(), g.Day())
	if !st.Met[mk] && a.AI != "guard" {
		st.Met[mk] = true
		if s := g.etLearnOne(ci, cust, "meet", a.Name); s != "" {
			g.Msg("{{K|The way " + a.Name + " carries themselves teaches you something: " + s + "}}")
		}
	}
	var out []Option
	if g.etKnownEtiquette(ci, cust) > 0 {
		out = append(out, Option{"Offer the local courtesy", func() { g.etCourtesyMenu(a, ci, cust) }})
	}
	out = append(out, g.etDutyOptions(a)...)
	if cust.Law != nil && g.etIsJudge(a) {
		if state := g.etState(); state > 0 && g.uwBounty(state) > 0 {
			out = append(out, Option{"Stand before the judge", func() { g.etJudgeScene(a, ci, cust) }})
		}
	}
	return out
}

func (g *Game) etIsJudge(a *Actor) bool {
	switch a.Role {
	case "guard", "elder", "governor", "noble", "magistrate", "judge":
		return true
	}
	return a.AI == "guard"
}

func (g *Game) etKnownEtiquette(ci int, cust *CustomsPack) int {
	n := 0
	for _, e := range cust.Etiquette {
		if g.etKnowE(ci, e.ID) {
			n++
		}
	}
	return n
}

var etKindVerb = map[string]string{
	"greeting": "Greet %s the way it is done here", "gesture": "Make the local gesture of respect",
	"gift": "Press a small gift into their hand", "speech": "Speak with the proper words",
	"food": "Praise the food as custom asks", "dress": "Straighten your dress to local custom",
	"space": "Keep the proper distance", "time": "Pay your respects at the proper hour",
}

func (g *Game) etCourtesyMenu(a *Actor, ci int, cust *CustomsPack) {
	var opts []Option
	for _, e := range cust.Etiquette {
		e := e
		if !g.etKnowE(ci, e.ID) {
			continue
		}
		v := etKindVerb[e.Kind]
		if v == "" {
			v = "Observe the courtesy: " + e.Text
		}
		if strings.Contains(v, "%s") {
			v = fmt.Sprintf(v, a.Name)
		}
		opts = append(opts, Option{v, func() { g.etDoCourtesy(a, ci, e) }})
	}
	opts = append(opts, Option{"Never mind.", func() { g.talkMenu(g.greeting(a)) }})
	g.Popup = Wrap("Which courtesy will you show?", 100)
	g.Opts = opts
	g.Sel = 0
}

func (g *Game) etDoCourtesy(a *Actor, ci int, e Etiquette) {
	st := g.etSt()
	text := cpFill(e.Text, g.etVars(a, ci, ""))
	switch e.Kind {
	case "gift":
		cost := 3 + g.P.Level/2
		if !g.cuSpend(cost) {
			g.talkMenu("You have no coin for a gift. (" + g.cuFormat(cost) + ")")
			return
		}
	case "time":
		if !g.daylight() {
			g.talkMenu("It is the wrong hour for that courtesy. " + text)
			return
		}
	case "dress":
		if b := g.P.PartFor("body"); b != nil && b.Item == nil {
			g.talkMenu("You have little to straighten. " + text)
			return
		}
	case "speech":
		if a.Role != "elder" && a.Role != "priest" && a.Role != "governor" && a.Role != "noble" {
			g.talkMenu("They nod politely, but it is not for them that these words are kept. " + text)
			return
		}
	}
	key := fmt.Sprintf("%d:%s:%d", a.ID, e.ID, g.Day())
	if st.Reward[key] {
		g.talkMenu(a.Name + " has already taken note of your manners today.")
		return
	}
	st.Reward[key] = true
	rw := maxi(e.Reward, 1)
	if p := lvPersonFor(g, a); p != nil {
		g.lvMemoryAdd(p, "etiquette", "You showed the proper courtesy.", mini(rw*3, 15))
	}
	if state := g.etState(); state > 0 {
		g.AddRep(uwFac(state), rw*4, " for good manners")
	}
	g.talkMenu(fmt.Sprintf("You follow the custom: %s\n\n{{G|%s approves of you.}}", text, a.Name))
}

// ---------------------------------------------------------------- law

// etLaw returns the punishment and note a culture's law gives a crime ("" "" without a pack or entry).
func (g *Game) etLaw(culture int, crime string) (punish, note string) {
	if g.W == nil {
		return "", ""
	}
	p := g.W.cpOf(culture)
	if p == nil || p.Customs == nil || p.Customs.Law == nil {
		return "", ""
	}
	for _, e := range p.Customs.Law.Entries {
		if e.Crime == crime {
			return e.Punish, e.Note
		}
	}
	return "", ""
}

// etLawLine flavours crime.go/underworld.go text: "Under the law of the X, theft is met with Y." ("" without a pack).
func (g *Game) etLawLine(state int, crime string) string {
	if g.W == nil || g.Cur == nil {
		return ""
	}
	ci, cust := g.etHere()
	if cust == nil || cust.Law == nil {
		return ""
	}
	punish, note := g.etLaw(ci, crime)
	if punish == "" {
		return ""
	}
	s := fmt.Sprintf("By the law of the %s, %s is met with %s.", g.W.Cultures[ci].Name, crime, punish)
	if note != "" {
		s += " " + note
	}
	return s
}

func etCrimeFor(bounty int) string {
	switch {
	case bounty >= 250:
		return "murder"
	case bounty >= 150:
		return "assault"
	case bounty >= 60:
		return "theft"
	}
	return "trespass"
}

func (g *Game) etJudgeScene(a *Actor, ci int, cust *CustomsPack) {
	law := cust.Law
	state := g.etState()
	bounty := g.uwBounty(state)
	if law == nil || bounty <= 0 {
		return
	}
	crime := etCrimeFor(bounty)
	punish, note := g.etLaw(ci, crime)
	dc, fee, flavour := etJudge(law.Judge)
	text := fmt.Sprintf("You are brought before %s.\n\n%s\n\nYour record: %d %s (%s).", law.Judge, flavour, bounty, g.Sk.Currency, crime)
	if punish != "" {
		text += fmt.Sprintf("\nThe law of the %s says: %s.", g.W.Cultures[ci].Name, punish)
	}
	if note != "" {
		text += " " + note
	}
	cost := bounty * fee / 100
	pl := strings.ToLower(punish)
	if strings.Contains(pl, "weregild") || strings.Contains(pl, "fine") || strings.Contains(pl, "compens") {
		cost = cost * 7 / 10
	}
	cost = maxi(cost, 1)
	opts := []Option{
		{fmt.Sprintf("Pay the weregild: %d %s", cost, g.Sk.Currency), func() {
			if g.P.Gold < cost {
				g.talkMenu("You cannot pay " + fmt.Sprint(cost) + " " + g.Sk.Currency + ".")
				return
			}
			g.P.Gold -= cost
			g.uwClearBounty(state)
			g.talkMenu("The judge strikes your name from the record. The debt is paid.")
		}},
		{"Submit to the punishment", func() {
			g.etSubmit(state, pl)
		}},
	}
	if law.Ordeal != "" {
		opts = append(opts, Option{"Undergo the ordeal: " + law.Ordeal, func() {
			if g.roll2d6Check(g.checkMod("endurance"), 10+dc) {
				g.uwClearBounty(state)
				g.gainXP(15)
				g.talkMenu("{{G|You come through the ordeal unharmed. The gods, or the judge, are satisfied. Your record is cleared.}}")
				return
			}
			n := mini(clampi(g.P.MaxHP/4, 2, 50), g.P.HP-1)
			if n > 0 {
				g.hurt(g.P, n, nil, "an ordeal")
			}
			g.talkMenu("{{R|You fail the ordeal and are hurt. The judge shakes their head. The record stands.}}")
		}})
	}
	opts = append(opts, Option{"Plead your case before the judge", func() {
		if g.roll2d6Check(g.checkMod("persuasion"), 9+dc) {
			st := g.uwSt()
			st.Bounty[state] = bounty / 2
			g.talkMenu("{{G|The judge is moved. Your record is halved.}}")
			return
		}
		st := g.uwSt()
		st.Bounty[state] = bounty * 6 / 5
		g.talkMenu("{{R|The judge takes your plea as insolence. The record grows by a fifth.}}")
	}})
	opts = append(opts, Option{"Walk away.", func() { g.talkMenu(g.greeting(a)) }})
	g.Popup = Wrap(text, 100)
	g.Opts = opts
	g.Sel = 0
}

func (g *Game) etSubmit(state int, pl string) {
	switch {
	case strings.Contains(pl, "lash") || strings.Contains(pl, "whip") || strings.Contains(pl, "flog"):
		n := mini(clampi(g.P.MaxHP/5, 2, 40), g.P.HP-1)
		if n > 0 {
			g.hurt(g.P, n, nil, "the lash")
		}
		g.uwClearBounty(state)
		g.talkMenu("You take the lash. The record is struck.")
	case strings.Contains(pl, "exile") || strings.Contains(pl, "banish"):
		g.uwClearBounty(state)
		if b := g.etBurg(); b > 0 {
			g.etSt().Ban[b] = g.Day() + 30
		}
		g.Mode, g.Talk, g.Popup, g.Opts = "main", nil, nil, nil
		g.Msg("{{R|You are sent into exile.}}")
		g.etExpel()
	default:
		g.Mode, g.Talk, g.Popup, g.Opts = "main", nil, nil, nil
		g.uwJail(state)
	}
}

// ---------------------------------------------------------------- primer page

func (g *Game) etAbilities() []Ability {
	return []Ability{{ID: "et_customs", Name: "Local customs", Target: "self"}}
}

func (g *Game) etUse(id string) {
	if id == "et_customs" {
		st := g.etSt()
		st.scroll, st.sel = 0, 0
		g.Mode = "etp"
	}
}

func (g *Game) etCultList() []int {
	st := g.etSt()
	var out []int
	ci, _ := g.etHere()
	if ci >= 0 {
		out = append(out, ci)
	}
	var more []int
	for c, n := range st.Live {
		if n > 0 && c != ci {
			more = append(more, c)
		}
	}
	sort.Ints(more)
	return append(out, more...)
}

func etKeys(g *Game, k string) {
	st := g.etSt()
	switch k {
	case "ArrowUp":
		st.scroll = maxi(0, st.scroll-1)
	case "ArrowDown":
		st.scroll++
	case "PageUp":
		st.scroll = maxi(0, st.scroll-12)
	case "PageDown":
		st.scroll += 12
	case "ArrowLeft":
		st.sel, st.scroll = maxi(0, st.sel-1), 0
	case "ArrowRight":
		st.sel, st.scroll = st.sel+1, 0
	case "Escape", "q":
		g.Mode = "main"
	}
}

func (g *Game) etPrimerLines(ci int) []string {
	w := g.W
	cu := &w.Cultures[ci]
	st := g.etSt()
	var L []string
	add := func(f string, a ...any) { L = append(L, fmt.Sprintf(f, a...)) }
	add("{{Y|The %s}}  {{K|(%s)}}", cu.Name, cu.Sphere)
	var cust *CustomsPack
	if p := w.cpOf(ci); p != nil {
		cust = p.Customs
	}
	if cust != nil {
		kt, ke := 0, 0
		for _, t := range cust.Taboos {
			if st.KT[etKey(ci, t.ID)] {
				kt++
			}
		}
		for _, e := range cust.Etiquette {
			if st.KE[etKey(ci, e.ID)] {
				ke++
			}
		}
		add("{{K|You know %d of %d taboos and %d of %d courtesies. Strangers learn by rebuke, by elders, by living among them.}}", kt, len(cust.Taboos), ke, len(cust.Etiquette))
	}
	L = append(L, "")
	if cust != nil && cust.Ideals != nil {
		id := cust.Ideals
		L = append(L, "{{c|Ideals}}")
		for _, p := range [][2]string{{"Death", id.Death}, {"The host", id.Host}, {"Name", id.Name}, {"Wealth", id.Wealth}} {
			if p[1] != "" {
				add("  %s: %s", p[0], p[1])
			}
		}
		L = append(L, "")
	}
	if len(cu.Values) > 0 {
		add("{{c|Values}}  %s", strings.Join(cu.Values, "; "))
		L = append(L, "")
	}
	if cust != nil && len(cust.Etiquette) > 0 {
		L = append(L, "{{c|Etiquette}}")
		for _, e := range cust.Etiquette {
			if st.KE[etKey(ci, e.ID)] {
				kw := etKindWord[e.Kind]
				add("  {{G|+}} [%s] %s {{K|(goodwill +%d)}}", kw, e.Text, maxi(e.Reward, 1))
			} else {
				L = append(L, "  {{K|? Something is expected of strangers here.}}")
			}
		}
		L = append(L, "")
	}
	if cust != nil && len(cust.Taboos) > 0 {
		L = append(L, "{{c|Taboos}}")
		for _, t := range cust.Taboos {
			if st.KT[etKey(ci, t.ID)] {
				pen := etPenaltyText[t.Penalty]
				if pen == "" {
					pen = "unknown"
				}
				add("  {{R|!}} %s {{K|(%s; price: %s)}}", t.Text, etSevWord[clampi(t.Severity, 1, 3)], pen)
			} else {
				L = append(L, "  {{K|? A thing not done here.}}")
			}
		}
		L = append(L, "")
	}
	if cust != nil && cust.Hospitality != nil {
		h := cust.Hospitality
		L = append(L, "{{c|Hospitality}}")
		add("  Guest right: %d day(s).", h.GuestRight)
		if len(h.Offers) > 0 {
			add("  Offered: %s.", strings.Join(h.Offers, "; "))
		}
		if len(h.Duties) > 0 {
			add("  A guest owes: %s.", strings.Join(h.Duties, "; "))
		}
		if h.Breach != "" {
			add("  Breach: %s", h.Breach)
		}
		L = append(L, "")
	}
	if cust != nil && cust.Law != nil {
		l := cust.Law
		L = append(L, "{{c|Law}}")
		add("  Judge: %s.", l.Judge)
		if l.Ordeal != "" {
			add("  Ordeal: %s.", l.Ordeal)
		}
		for _, e := range l.Entries {
			s := fmt.Sprintf("  %s: %s", e.Crime, e.Punish)
			if e.Note != "" {
				s += " {{K|(" + e.Note + ")}}"
			}
			L = append(L, s)
		}
		L = append(L, "")
	}
	if cust != nil && len(cust.Rites) > 0 {
		L = append(L, "{{c|Rites}}")
		for _, r := range cust.Rites {
			s := "  " + r.Name
			if len(r.Steps) > 0 {
				s += ": " + strings.Join(r.Steps, ", ")
			}
			L = append(L, s)
		}
		L = append(L, "")
	}
	if cust != nil && len(cust.Festivals) > 0 {
		L = append(L, "{{c|Festival customs}}")
		for _, f := range cust.Festivals {
			s := "  " + f.Feast
			if f.Custom != "" {
				s += ": " + f.Custom
			}
			if f.Taboo != "" {
				s += " {{K|Taboo: " + f.Taboo + "}}"
			}
			L = append(L, s)
		}
		L = append(L, "")
	}
	if cust == nil {
		L = append(L, "{{K|No detailed customs are recorded for this people.}}")
	}
	if cu.Food != "" {
		add("{{c|Food}}  %s", cu.Food)
	}
	if cu.Dress != "" {
		add("{{c|Dress}}  %s", cu.Dress)
	}
	if len(cu.Festivals) > 0 {
		add("{{c|Festivals}}  %s", cu.Festivals[0])
	}
	return L
}

func etDraw(g *Game) {
	s := g.sc
	st := g.etSt()
	x, y, w, h := g.panel("Local Customs")
	list := g.etCultList()
	if len(list) == 0 {
		s.Text(x, y, "{{K|You are far from any settled people.}}", CGray)
		return
	}
	st.sel = clampi(st.sel, 0, len(list)-1)
	var lines []string
	for _, l := range g.etPrimerLines(list[st.sel]) {
		if l == "" {
			lines = append(lines, "")
			continue
		}
		lines = append(lines, Wrap(l, w)...)
	}
	vis := h - 3
	st.scroll = clampi(st.scroll, 0, maxi(0, len(lines)-vis))
	for i := 0; i < vis && st.scroll+i < len(lines); i++ {
		s.TextB(x, y+i, lines[st.scroll+i], CGray, CBgDark, x+w)
	}
	s.Text(x, y+h-2, fmt.Sprintf("{{K|Up/Down PgUp/PgDn: scroll   Left/Right: people (%d/%d)   Esc: back}}", st.sel+1, len(list)), CGray)
}
