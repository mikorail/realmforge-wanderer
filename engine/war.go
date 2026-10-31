package engine

import (
	"fmt"
	"strconv"
	"strings"
)

// Joinable wars: frontier skirmishes, enlisting with a realm at war, contracts, ranks and a status screen.
// Everything hangs off the hooks in hooks.go; saved state lives in g.ext("war").

var warRanks = []string{"Recruit", "Soldier", "Sergeant", "Lieutenant", "Captain", "Commander", "Marshal"}
var warMerit = []int{0, 10, 30, 70, 130, 220, 350}

type warContract struct {
	Kind  string `json:"k"` // kill hold dispatch
	Enemy int    `json:"e"`
	Need  int    `json:"n"`
	Have  int    `json:"h"`
	Burg  int    `json:"b,omitempty"` // dispatch destination
	Pay   int    `json:"p"`
}

type warState struct {
	Realm     int          `json:"realm"` // 0 = not enlisted
	Merit     int          `json:"merit"`
	Kills     int          `json:"kills"`
	Done      int          `json:"done"`
	Companion int          `json:"comp"`
	C         *warContract `json:"c,omitempty"`
}

func init() {
	zoneHooks = append(zoneHooks, warZone)
	talkHooks = append(talkHooks, warTalk)
	killHooks = append(killHooks, warKill)
	enterHooks = append(enterHooks, warEnter)
	turnHooks = append(turnHooks, warTurn)
	sideHooks = append(sideHooks, warSide)
	mainKeys["W"] = func(g *Game) { g.Mode = "war" }
	modeKeys["war"] = func(g *Game, k string) {
		if k == "Escape" || k == "Enter" || k == "W" || k == " " {
			g.Mode = "main"
		}
	}
	modeDraws["war"] = func(g *Game) { g.drawWar() }
}

func (g *Game) war() *warState { return g.ext("war", func() any { return &warState{} }).(*warState) }

func stateFaction(id int) string { return "state:" + strconv.Itoa(id) }

func (g *Game) realmName(id int) string {
	if s := g.W.stateByID[id]; s != nil {
		return s.Name
	}
	return "the Free Folk"
}

// warsOf lists the living realms a realm is at war with (either side may have declared it).
func (g *Game) warsOf(id int) []int {
	var out []int
	seen := map[int]bool{}
	add := func(e int) {
		if e > 0 && e != id && !seen[e] {
			if s := g.W.stateByID[e]; s != nil && s.Alive {
				seen[e] = true
				out = append(out, e)
			}
		}
	}
	if s := g.W.stateByID[id]; s != nil {
		for _, w := range s.Wars {
			add(w)
		}
	}
	for i := range g.W.States {
		for _, w := range g.W.States[i].Wars {
			if w == id {
				add(g.W.States[i].ID)
			}
		}
	}
	return out
}

func (g *Game) warName(a, b int) string {
	return fmt.Sprintf("The %s–%s War", g.realmName(a), g.realmName(b))
}

// frontierAt: the parasang's realm and an enemy realm among its 8 neighbours.
func (g *Game) frontierAt(wx, wy int) (own, enemy int, ok bool) {
	w := g.W
	if !w.In(wx, wy) {
		return
	}
	own = w.State[w.Idx(wx, wy)]
	if own <= 0 {
		return
	}
	foes := g.warsOf(own)
	if len(foes) == 0 {
		return
	}
	for dy := -1; dy <= 1; dy++ {
		for dx := -1; dx <= 1; dx++ {
			if (dx != 0 || dy != 0) && w.In(wx+dx, wy+dy) {
				s := w.State[w.Idx(wx+dx, wy+dy)]
				for _, f := range foes {
					if f == s {
						return own, s, true
					}
				}
			}
		}
	}
	return own, 0, false
}

func (g *Game) isFrontier(zn *Zone) bool {
	if zn == nil || zn.Z != 0 || zn.Kind != "surface" {
		return false
	}
	_, _, ok := g.frontierAt(zn.WX, zn.WY)
	return ok
}

func (g *Game) warBurg(realm int) *BurgData {
	if s := g.W.stateByID[realm]; s != nil {
		if b := g.W.burgByID[s.Capital]; b != nil {
			return b
		}
	}
	for i := range g.W.Burgs {
		if g.W.Burgs[i].State == realm && g.W.Burgs[i].Alive {
			return &g.W.Burgs[i]
		}
	}
	return nil
}

func (g *Game) soldier(realm int, r *Rng) *Actor {
	bd := g.warBurg(realm)
	if bd == nil {
		return nil
	}
	a := g.makeNPC("guard", bd, g.W.cultByID[bd.Culture], r)
	a.Faction, a.State = stateFaction(realm), realm
	a.Role, a.Title, a.AI = "", "soldier", "hostile"
	a.Set("soldier")
	a.Desc = fmt.Sprintf("A soldier of %s.", g.realmName(realm))
	return a
}

func isSoldier(a *Actor) bool { return a != nil && (a.Has("soldier") || a.Role == "guard") }

// warZone: skirmishes and banners on frontier zones.
func warZone(g *Game, zn *Zone, _ *Rng) {
	if !g.isFrontier(zn) {
		return
	}
	own, foe, _ := g.frontierAt(zn.WX, zn.WY)
	r := NewRng(g.Seed + ":skirmish:" + zn.Key)
	if !r.Chance(0.4) {
		return
	}
	cx, cy, ok := zn.FreeSpot(r, func(x, y int) bool { return x > 12 && x < MW-12 && y > 5 && y < MH-5 })
	if !ok {
		return
	}
	side := func(realm, ox int) {
		n := r.Int(2, 4)
		for i := 0; i < n; i++ {
			x, y, ok := zn.FreeSpot(r, func(x, y int) bool { return abs(x-(cx+ox)) <= 3 && abs(y-cy) <= 3 })
			if !ok {
				continue
			}
			a := g.soldier(realm, r)
			if a == nil {
				return
			}
			a.X, a.Y, a.HomeX, a.HomeY = x, y, x, y
			zn.Actors = append(zn.Actors, a)
		}
	}
	side(own, -4)
	side(foe, 4)
	name := g.warName(own, foe)
	for _, ox := range []int{-7, 7} {
		if x, y, ok := zn.FreeSpot(r, func(x, y int) bool { return abs(x-(cx+ox)) <= 2 && abs(y-cy) <= 3 }); ok {
			zn.Set(x, y, TStatue)
			who := own
			if ox > 0 {
				who = foe
			}
			zn.Notes[y*MW+x] = fmt.Sprintf("A torn banner of %s hangs from a broken pike. Scratched below: \"%s.\"", g.realmName(who), name)
		}
	}
	if zn.Faction == "" {
		zn.Faction = stateFaction(own)
	}
}

func (g *Game) warRank(es *warState) int {
	r := 0
	for i, m := range warMerit {
		if es.Merit >= m {
			r = i
		}
	}
	return r
}

// warStanding: bonus for society standing (society.go socialStanding): full in your realm's lands, half elsewhere.
func (g *Game) warStanding() int {
	es := g.war()
	if es.Realm == 0 {
		return 0
	}
	b := 1 + g.warRank(es)
	if g.Cur != nil && g.W.In(g.Cur.WX, g.Cur.WY) && g.W.State[g.W.Idx(g.Cur.WX, g.Cur.WY)] != es.Realm {
		b /= 2
	}
	return b
}

func (g *Game) warAddMerit(n int) {
	es := g.war()
	before := g.warRank(es)
	es.Merit += n
	if after := g.warRank(es); after > before {
		g.Msg(fmt.Sprintf("{{G|Promoted to %s in the army of %s!}}", warRanks[after], g.realmName(es.Realm)))
		g.deed("quest", fmt.Sprintf("%s was raised to %s in the army of %s.", g.P.Name, warRanks[after], g.realmName(es.Realm)), 0, 0)
	}
}

func (g *Game) warEnlist(realm int, from *Actor) {
	es := g.war()
	*es = warState{Realm: realm}
	g.Msg(fmt.Sprintf("{{G|You take the oath and enlist with %s.}}", g.realmName(realm)))
	g.deed("quest", fmt.Sprintf("%s enlisted in the army of %s.", g.P.Name, g.realmName(realm)), 0, 0)
	g.talkMenu("\"Welcome to the colours, recruit. Ask me for a contract when you are ready.\"")
}

func (g *Game) warDischarge(why string) {
	es := g.war()
	if es.Realm == 0 {
		return
	}
	g.Msg("{{R|You are discharged in disgrace (" + why + ").}}")
	g.AddRep(stateFaction(es.Realm), -150, " for your desertion")
	g.deed("quest", fmt.Sprintf("%s was struck from the rolls of %s.", g.P.Name, g.realmName(es.Realm)), 0, 0)
	*es = warState{}
}

func (g *Game) warPay(c *warContract, es *warState) {
	g.P.Gold += c.Pay
	g.gainXP(c.Pay)
	g.AddRep(stateFaction(es.Realm), 60, " for service")
	if c.Enemy > 0 {
		g.AddRep(stateFaction(c.Enemy), -40, "")
	}
	es.Done++
	g.warAddMerit(10 + c.Need/2)
	g.Msg(fmt.Sprintf("{{G|Contract complete: %d %s.}}", c.Pay, g.Sk.Currency))
	g.deed("quest", fmt.Sprintf("%s fulfilled a contract for %s against %s.", g.P.Name, g.realmName(es.Realm), g.realmName(c.Enemy)), 0, 0)
	es.C = nil
}

func (g *Game) warContractText(c *warContract) string {
	switch c.Kind {
	case "kill":
		return fmt.Sprintf("Slay %d soldiers of %s (%d/%d)", c.Need, g.realmName(c.Enemy), c.Have, c.Need)
	case "hold":
		return fmt.Sprintf("Hold a frontier zone %d turns (%d/%d)", c.Need, c.Have, c.Need)
	}
	bn := "a city"
	if b := g.W.burgByID[c.Burg]; b != nil {
		bn = b.Name
	}
	return "Carry a dispatch to " + bn
}

func (g *Game) warOffer(a *Actor) {
	es := g.war()
	foes := g.warsOf(es.Realm)
	if len(foes) == 0 {
		g.talkMenu("\"The war is quiet. Nothing for you.\"")
		return
	}
	foe := foes[g.R.Int(0, len(foes)-1)]
	c := &warContract{Enemy: foe}
	switch g.R.Int(0, 2) {
	case 0:
		c.Kind, c.Need = "kill", g.R.Int(3, 6)
		c.Pay = 40 + c.Need*25
	case 1:
		c.Kind, c.Need = "hold", g.R.Int(20, 40)
		c.Pay = 60 + c.Need*5
	default:
		var best *BurgData
		bd := -1
		for i := range g.W.Burgs {
			b := &g.W.Burgs[i]
			if b.State != es.Realm || !b.Alive || b.Layer != "surface" || b.ID == g.Cur.Burg {
				continue
			}
			d := abs(b.P%g.W.W-g.Cur.WX) + abs(b.P/g.W.W-g.Cur.WY)
			if best == nil || d < bd && d > 2 || g.R.Chance(0.15) {
				best, bd = b, d
			}
		}
		if best == nil {
			c.Kind, c.Need, c.Pay = "kill", 4, 140
		} else {
			c.Kind, c.Burg, c.Need, c.Pay = "dispatch", best.ID, 1, 120+bd*8
		}
	}
	es.C = c
	g.talkMenu("\"Here is your task: " + g.warContractText(c) + ". " + fmt.Sprintf("%d %s on completion.\"", c.Pay, g.Sk.Currency))
}

func warTalk(g *Game, a *Actor) []Option {
	if a == nil || a.State <= 0 || !strings.HasPrefix(a.Faction, "state:") {
		return nil
	}
	es := g.war()
	var out []Option
	realm := a.State
	if es.Realm == 0 {
		barracks := g.Cur != nil && strings.HasSuffix(g.Cur.Name, districtNames["barracks"])
		if !(a.Role == "elder" || barracks && (a.Role == "trainer" || a.Role == "guard")) {
			return nil
		}
		foes := g.warsOf(realm)
		if len(foes) == 0 {
			return nil
		}
		if g.Rep[a.Faction] < 0 {
			return []Option{{"{{K|Enlist with " + g.realmName(realm) + ". (needs reputation 0)}}", func() {
				g.talkMenu("\"We do not take those the realm distrusts.\"")
			}}}
		}
		return []Option{{"{{R|Enlist with " + g.realmName(realm) + ".}} {{K|(at war with " + g.realmName(foes[0]) + ")}}", func() { g.warEnlist(realm, a) }}}
	}
	if es.Realm != realm {
		return nil
	}
	if es.C == nil {
		out = append(out, Option{"{{R|Request a war contract.}}", func() { g.warOffer(a) }})
	} else if es.C.Kind == "dispatch" && g.Cur.Burg == es.C.Burg && (a.Role == "elder" || a.Role == "guard" || a.Role == "notable") {
		out = append(out, Option{"{{G|Present the dispatch.}}", func() {
			c := es.C
			g.warPay(c, es)
			g.talkMenu("\"Received. The realm thanks you.\"")
		}})
	} else {
		c := es.C
		out = append(out, Option{"{{K|Report on my contract.}}", func() {
			g.talkMenu("\"" + g.warContractText(c) + ". Get to it.\"")
		}})
	}
	if rk := g.warRank(es); rk >= 4 && es.Companion < rk-3 {
		out = append(out, Option{"{{C|Request a soldier to serve at my side.}}", func() { g.warCompanion(es) }})
	}
	return out
}

func (g *Game) warCompanion(es *warState) {
	n := 0
	for _, o := range g.Cur.Actors {
		if o.AI == "follower" {
			n++
		}
	}
	if n >= 3 {
		g.talkMenu("\"You've enough company already.\"")
		return
	}
	a := g.soldier(es.Realm, g.R)
	if a == nil {
		return
	}
	a.AI, a.Faction, a.F, a.Angry = "follower", "player", CGreen, false
	a.Level += 2
	a.X, a.Y = g.P.X, g.P.Y
	g.Cur.Actors = append(g.Cur.Actors, a)
	g.nudge(g.Cur, a)
	es.Companion++
	g.Msg(fmt.Sprintf("{{G|%s of the realm's army joins you.}}", a.Name))
	g.Mode = "main"
	g.Talk = nil
}

func warKill(g *Game, d, src *Actor) {
	es := g.war()
	if es.Realm == 0 || d == nil || (src != g.P && (src == nil || src.AI != "follower")) {
		return
	}
	if d.Faction == stateFaction(es.Realm) && isSoldier(d) {
		g.warDischarge("you killed a soldier of your own side")
		return
	}
	if !isSoldier(d) || !strings.HasPrefix(d.Faction, "state:") {
		return
	}
	id, _ := strconv.Atoi(d.Faction[6:])
	foe := false
	for _, f := range g.warsOf(es.Realm) {
		foe = foe || f == id
	}
	if !foe {
		return
	}
	es.Kills++
	g.warAddMerit(2)
	if c := es.C; c != nil && c.Kind == "kill" && c.Enemy == id {
		c.Have++
		if c.Have >= c.Need {
			g.warPay(c, es)
		} else {
			g.Msg(fmt.Sprintf("{{K|Contract: %d/%d.}}", c.Have, c.Need))
		}
	}
}

// warAngerFoes makes enemy soldiers in the zone attack an enlisted player on sight.
func warAngerFoes(g *Game, zn *Zone) {
	es := g.war()
	if es.Realm == 0 || zn == nil {
		return
	}
	foes := map[string]bool{}
	for _, f := range g.warsOf(es.Realm) {
		foes[stateFaction(f)] = true
	}
	for _, a := range zn.Actors {
		if a.HP > 0 && foes[a.Faction] && a.Has("soldier") {
			a.Angry = true
		}
	}
}

func warEnter(g *Game, zn *Zone) { warAngerFoes(g, zn) }

func warTurn(g *Game) {
	es := g.war()
	if es.Realm == 0 || g.Cur == nil {
		return
	}
	warAngerFoes(g, g.Cur)
	for _, a := range g.Cur.Actors {
		if a.Angry && a.HP > 0 && a.Faction == stateFaction(es.Realm) && isSoldier(a) {
			g.warDischarge("you raised your hand against your own side")
			return
		}
	}
	if c := es.C; c != nil && c.Kind == "hold" && g.isFrontier(g.Cur) {
		if own, _, _ := g.frontierAt(g.Cur.WX, g.Cur.WY); own == es.Realm {
			c.Have++
			if c.Have >= c.Need {
				g.warPay(c, es)
			}
		}
	}
}

func warSide(g *Game) []string {
	es := g.war()
	if es.Realm == 0 {
		return nil
	}
	out := []string{"{{Y|" + warRanks[g.warRank(es)] + "}} {{K|" + g.realmName(es.Realm) + "}}"}
	if c := es.C; c != nil {
		out = append(out, fmt.Sprintf("{{K|War: %s %d/%d}}", c.Kind, c.Have, c.Need))
	}
	return out
}

func (g *Game) drawWar() {
	s := g.sc
	x, y, w, _ := g.panel("War")
	es := g.war()
	t := func(str string) {
		s.TextB(x, y, str, CGray, CBgDark, x+w)
		y++
	}
	if es.Realm == 0 {
		t("{{K|You serve no army. Speak to a drillmaster, warden or elder of a realm at war to enlist.}}")
	} else {
		rk := g.warRank(es)
		t(fmt.Sprintf("{{W|Rank:}} {{Y|%s}} of %s", warRanks[rk], g.realmName(es.Realm)))
		next := "{{K|(highest rank)}}"
		if rk+1 < len(warMerit) {
			next = fmt.Sprintf("{{K|(next at %d)}}", warMerit[rk+1])
		}
		t(fmt.Sprintf("{{W|Merit:}} %d %s   {{W|Kills:}} %d   {{W|Contracts:}} %d", es.Merit, next, es.Kills, es.Done))
		if es.C != nil {
			t("{{W|Contract:}} " + g.warContractText(es.C))
		} else {
			t("{{W|Contract:}} {{K|none (ask a drillmaster or elder)}}")
		}
		y++
	}
	realm := es.Realm
	if realm == 0 && g.Cur != nil && g.W.In(g.Cur.WX, g.Cur.WY) {
		realm = g.W.State[g.W.Idx(g.Cur.WX, g.Cur.WY)]
	}
	t("{{W|Wars of " + g.realmName(realm) + ":}}")
	foes := g.warsOf(realm)
	if len(foes) == 0 {
		t("{{K|  At peace.}}")
	}
	for _, f := range foes {
		t("  {{R|" + g.warName(realm, f) + "}}")
	}
	y++
	t("{{K|Esc closes}}")
}
