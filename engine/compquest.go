package engine

import (
	"fmt"
	"strconv"
	"strings"
)

// Companion personal quests (prefix cq). Once a recruited companion trusts you (loyalty 70+ after
// three days together) they ask for help with something from home, drawn from their kin and home
// town: avenge a relative maimed by a beast or bandit (a named foe placed near their home
// parasang), recover a family heirloom from a ruin (the nearest dead town) or the wilds, make
// peace with an estranged relative in another town, or settle a debt with a creditor at home.
// Each quest is mirrored into g.Quests (Kind "companion", ignored by the core quest code) so it
// shows in the journal and on the world map. Rewards: +30 loyalty, a perk (attributes, HP, AV),
// player XP and a "companion" deed. State lives in g.ext("cq"), keyed by companion actor id; quest
// stories come from NewRng(seed:cq:id), never g.R.

type cqQuest struct {
	Comp   int    `json:"c"`
	Name   string `json:"n"`
	Kind   string `json:"k"`  // avenge heirloom peace debt
	Stage  int    `json:"s"`  // 0 offered, 1 active, 2 ready to report, 3 complete, 4 failed
	QID    int    `json:"q"`  // mirror Quest id in g.Quests
	Title  string `json:"t"`  //
	Text   string `json:"x"`  // the companion's story
	Target string `json:"tk"` // zone key where the foe / item / relative is placed
	TP     int    `json:"tp"` // parasang
	Foe    string `json:"f,omitempty"`
	Tpl    string `json:"tpl,omitempty"`
	NPC    int    `json:"npc,omitempty"` // placed actor id (foe, relative, creditor)
	Item   string `json:"it,omitempty"`  // heirloom name
	Gold   int    `json:"g,omitempty"`   // debt
	Placed bool   `json:"pl,omitempty"`
	Planet int    `json:"pn,omitempty"`
	Perk   string `json:"pk,omitempty"`
}

type cqState struct {
	Q       map[int]*cqQuest `json:"q,omitempty"`
	NextAsk map[int]int      `json:"na,omitempty"`
}

func (g *Game) cq() *cqState {
	st := g.ext("cq", func() any { return &cqState{} }).(*cqState)
	if st.Q == nil {
		st.Q = map[int]*cqQuest{}
	}
	if st.NextAsk == nil {
		st.NextAsk = map[int]int{}
	}
	return st
}

const cqDays = 3

// cqHome parses a companion's home zone key into its parasang.
func cqHome(c *Companion) (int, int, bool) {
	ps := strings.Split(c.HomeKey, ".")
	if len(ps) < 2 {
		return 0, 0, false
	}
	x, e1 := strconv.Atoi(ps[0])
	y, e2 := strconv.Atoi(ps[1])
	return x, y, e1 == nil && e2 == nil
}

func (g *Game) cqEligible(a *Actor) bool {
	c := a.Comp
	if c == nil || c.Summoned || c.Loyalty < 70 || g.Turn-c.Since < cqDays*DayLen || g.Turn < g.cq().NextAsk[a.ID] {
		return false
	}
	st := g.cq()
	if _, ok := st.Q[a.ID]; ok {
		return false
	}
	for _, q := range st.Q {
		if q.Stage < 3 {
			return false // one personal matter at a time
		}
	}
	return true
}

// cqMake builds a companion's personal quest from their home, kin and the world around it.
func (g *Game) cqMake(a *Actor) *cqQuest {
	c := a.Comp
	w := g.W
	hx, hy, ok := cqHome(c)
	if !ok || !w.In(hx, hy) {
		return nil
	}
	r := NewRng(fmt.Sprintf("%s:cq:%d", w.Seed, a.ID))
	home := w.burgByID[w.Burg[w.Idx(hx, hy)]]
	cu := (*CultureData)(nil)
	if home != nil {
		cu = w.cultByID[home.Culture]
	}
	town := nonEmpty(c.Town, "home")
	q := &cqQuest{Comp: a.ID, Name: a.Name, Planet: w.uniIdx}
	kin := ""
	if len(c.Kin) > 0 {
		kin = c.Kin[r.Int(0, len(c.Kin)-1)]
	}
	invent := func() string {
		rel := Pick(r, []string{"brother", "sister", "cousin", "uncle", "aunt"})
		n := givenOf(g.cultureName(cu, r))
		if a.House != "" {
			n += " " + a.House
		}
		return rel + " " + n
	}
	// wild land next to home (not water, not a town)
	wild := func() (int, int, bool) {
		off := r.Int(0, 7)
		for i := 0; i < 8; i++ {
			dx, dy := dirs8[(off+i)%8][0], dirs8[(off+i)%8][1]
			for k := 1; k <= 3; k++ {
				x, y := hx+dx*k, hy+dy*k
				if !w.In(x, y) {
					break
				}
				p := w.Idx(x, y)
				if !w.IsWater(p) && w.Burg[p] == 0 && len(w.wondersAt[p]) == 0 {
					return x, y, true
				}
			}
		}
		return 0, 0, false
	}
	var kinds []string
	if _, _, ok := wild(); ok {
		kinds = append(kinds, "avenge", "heirloom")
	}
	var other *BurgData
	for i := range w.Burgs {
		b := &w.Burgs[i]
		if !b.Alive || b.Layer != "surface" || home != nil && b.ID == home.ID {
			continue
		}
		d := maxi(abs(b.P%w.W-hx), abs(b.P/w.W-hy))
		if d >= 2 && d <= 15 && (other == nil || d < maxi(abs(other.P%w.W-hx), abs(other.P/w.W-hy))) {
			other = b
		}
	}
	if other != nil {
		kinds = append(kinds, "peace")
	}
	if home != nil && home.Alive {
		kinds = append(kinds, "debt")
	}
	if len(kinds) == 0 {
		return nil
	}
	q.Kind = Pick(r, kinds)
	switch q.Kind {
	case "avenge":
		x, y, _ := wild()
		q.Target, q.TP = ZoneKey(x, y, 1, 1, 0), w.Idx(x, y)
		lvl := g.zoneLevel(x, y, 0) + 2
		if r.Chance(0.4) {
			q.Tpl = "bandit"
			q.Foe = givenOf(g.cultureName(cu, r)) + " " + Pick(r, []string{"Redhand", "the Cutter", "Blackgate", "the Toll-Taker", "Ninefingers"})
		} else {
			q.Tpl = pickCreature(r, w.Biome[q.TP], 's', lvl)
			q.Foe = "the " + Pick(r, []string{"Maimer", "Scourge", "Widowmaker", "Grey Terror", "Night-Prowler"}) + " of " + town
		}
		who := kin
		if who == "" {
			who = invent()
		}
		q.Title = fmt.Sprintf("%s: avenge %s %s", givenOf(a.Name), possessiveWord(a), who)
		q.Text = fmt.Sprintf("\"Before I left %s, %s took my %s on the road and left them crippled. They say it still haunts the land %s of %s. I want it dead, and I want to be there when it falls.\"",
			town, q.Foe, who, cqDir(hx, hy, x, y), town)
		q.Perk = "STR"
	case "heirloom":
		item := Pick(r, []string{"silver locket", "grandmother's ring", "carved family seal", "wedding torc", "old house blade", "book of the household"})
		if a.House != "" {
			q.Item = "House " + a.House + "'s " + item
		} else {
			q.Item = givenOf(a.Name) + "'s family " + item
		}
		x, y, _ := wild()
		where := fmt.Sprintf("the wilds %s of %s", cqDir(hx, hy, x, y), town)
		bestD := 99
		for i := range w.Burgs {
			b := &w.Burgs[i]
			if b.Alive || b.Layer != "surface" {
				continue
			}
			if d := maxi(abs(b.P%w.W-hx), abs(b.P/w.W-hy)); d <= 14 && d < bestD {
				bestD, x, y = d, b.P%w.W, b.P/w.W
				where = "the ruins of " + b.Name
			}
		}
		q.Target, q.TP = ZoneKey(x, y, 1, 1, 0), w.Idx(x, y)
		q.Title = fmt.Sprintf("%s: recover the %s", givenOf(a.Name), item)
		q.Text = fmt.Sprintf("\"My family's %s was carried off when I was small. A trader swore he saw it in %s, among the bones of those who took it. Help me bring it back, and my %s will light candles for you.\"",
			item, where, nonEmpty(kin, "kin"))
		q.Perk = "TOU"
	case "peace":
		x, y := other.P%w.W, other.P/w.W
		q.Target, q.TP = ZoneKey(x, y, 1, 1, 0), other.P
		q.Foe = invent()
		q.Title = fmt.Sprintf("%s: make peace with %s %s", givenOf(a.Name), possessiveWord(a), q.Foe)
		q.Text = fmt.Sprintf("\"My %s left %s in anger years ago and settled in %s. Words were said that should not have been. I would set it right before one of us is in the ground. Will you take me there?\"",
			q.Foe, town, other.Name)
		q.Perk = "HP"
	case "debt":
		q.Target, q.TP = c.HomeKey, w.Idx(hx, hy)
		q.Foe = g.cultureName(cu, r)
		q.Gold = 40 + a.Level*15
		q.Title = fmt.Sprintf("%s: settle a debt in %s", givenOf(a.Name), town)
		q.Text = fmt.Sprintf("\"I owe %s of %s more than I can pay; it's half of why I left with you. If they find me, they'll take it out of my %s. Could you settle it? %d %s would clear my name.\"",
			q.Foe, town, nonEmpty(kin, "family"), q.Gold, g.Sk.Currency)
		q.Perk = "AGI"
	}
	return q
}

func cqDir(x0, y0, x1, y1 int) string {
	dx, dy := sign(x1-x0), sign(y1-y0)
	ns := map[int]string{-1: "north", 0: "", 1: "south"}[dy]
	ew := map[int]string{-1: "west", 0: "", 1: "east"}[dx]
	if ns+ew == "" {
		return "around"
	}
	return ns + ew
}

// cqTick: hourly, companions who trust you bring up their personal matter; quests whose companion
// is gone fail; held heirlooms are noticed.
func (g *Game) cqTick() {
	if g.Cur == nil {
		return
	}
	st := g.cq()
	for _, q := range st.Q {
		if q.Stage == 1 && q.Kind == "heirloom" && q.Planet == g.W.uniIdx {
			for _, it := range g.P.Inv {
				if it.Quest == q.QID && q.QID != 0 {
					q.Stage = 2
					g.cqMirrorDone(q)
					g.Msg(fmt.Sprintf("{{G|You hold the %s. %s will want to see it.}}", q.Item, q.Name))
				}
			}
		}
	}
	if g.Turn%60 != 0 {
		return
	}
	for id, q := range st.Q {
		if q.Stage >= 3 || g.cqFind(id) != nil {
			continue
		}
		q.Stage = 4
		if mq := g.cqMirror(q); mq != nil {
			mq.Done, mq.Closed = true, true
			mq.Title += " (abandoned)"
		}
	}
	for _, a := range g.localCompanions() {
		if !g.cqEligible(a) {
			continue
		}
		q := g.cqMake(a)
		if q == nil {
			st.NextAsk[a.ID] = g.Turn + 10*DayLen
			continue
		}
		st.Q[a.ID] = q
		g.Msg(fmt.Sprintf("{{Y|%s seems to have something on %s mind.}} {{K|(talk to them)}}", a.Name, possessiveWord(a)))
	}
}

// cqFind: the companion, wherever they wait (nil if they left or died).
func (g *Game) cqFind(id int) *Actor {
	for _, a := range g.localCompanions() {
		if a.ID == id {
			return a
		}
	}
	for _, wc := range g.waitingCompanions() {
		if wc.A.ID == id {
			return wc.A
		}
	}
	return nil
}

func (g *Game) cqMirror(q *cqQuest) *Quest {
	for _, mq := range g.Quests {
		if mq.ID == q.QID && q.QID != 0 {
			return mq
		}
	}
	return nil
}

func (g *Game) cqMirrorDone(q *cqQuest) {
	if mq := g.cqMirror(q); mq != nil {
		mq.Done = true
	}
}

// cqTalkOptions: extra lines in companion talk (hook in companionTalk).
func (g *Game) cqTalkOptions(a *Actor) []Option {
	q := g.cq().Q[a.ID]
	if q == nil {
		return nil
	}
	switch q.Stage {
	case 0:
		return []Option{{"{{Y|Something is on your mind.}}", func() { g.cqOffer(a, q) }}}
	case 1:
		return []Option{{"{{K|About your trouble at home…}}", func() { g.companionTalk(a, q.Text+"\n\n{{K|(Quest: "+q.Title+")}}") }}}
	case 2:
		return []Option{{"{{G|It is done.}}", func() { g.cqComplete(a, q) }}}
	}
	return nil
}

func (g *Game) cqOffer(a *Actor, q *cqQuest) {
	g.Talk, g.Mode, g.Sel = a, "talk", 0
	g.Popup = Wrap(q.Text, 100)
	g.Opts = []Option{
		{"{{G|I'll help you.}}", func() { g.cqAccept(a, q) }},
		{"Not now.", func() {
			delete(g.cq().Q, a.ID)
			g.cq().NextAsk[a.ID] = g.Turn + cqDays*DayLen
			g.addLoyalty(a, -3)
			g.companionTalk(a, "\"I see.\" They look away. \"Another time, then.\"")
		}},
	}
}

func (g *Game) cqAccept(a *Actor, q *cqQuest) {
	q.Stage = 1
	mq := &Quest{ID: g.newID(), Kind: "companion", Title: q.Title, Text: q.Text, Giver: a.Name, GiverID: a.ID, Faction: "player", Target: q.Target, TP: q.TP}
	q.QID = mq.ID
	g.Quests = append(g.Quests, mq)
	g.Known[q.TP] = true
	g.addLoyalty(a, 5)
	g.Msg("{{W|New quest: " + q.Title + "}}")
	g.companionTalk(a, "\"Thank you. I won't forget this.\"\n\n{{K|(Quest added to your journal; destination marked on the world map.)}}")
	if g.Cur != nil && g.Cur.Key == q.Target {
		g.cqEnter(g.Cur)
	}
}

// cqEnter places the quest's foe, heirloom, relative or creditor on arrival (enter hook).
func (g *Game) cqEnter(zn *Zone) {
	for _, q := range g.cq().Q {
		if q.Stage != 1 || q.Placed || q.Target != zn.Key || q.Planet != g.W.uniIdx {
			continue
		}
		q.Placed = true
		r := NewRng(fmt.Sprintf("%s:cqp:%d", g.W.Seed, q.Comp))
		spot := func(a *Actor) {
			if x, y, ok := zn.FreeSpot(r, func(x, y int) bool { return abs(x-g.P.X) > 6 || abs(y-g.P.Y) > 4 }); ok {
				a.X, a.Y = x, y
			} else if x, y, ok := zn.FreeSpot(r, nil); ok {
				a.X, a.Y = x, y
			}
			a.HomeX, a.HomeY = a.X, a.Y
			zn.Actors = append(zn.Actors, a)
		}
		switch q.Kind {
		case "avenge":
			f := g.Spawn(q.Tpl, zn.Level+2, r)
			f.Name, f.Proper, f.Boss = q.Foe, true, true
			f.MaxHP += 10
			f.HP = f.MaxHP
			f.Quest = q.QID
			spot(f)
			q.NPC = f.ID
		case "heirloom":
			it := &Item{ID: g.newID(), Kind: "trophy", Name: q.Item, Glyph: '♦', F: CYellow, Value: 60, Weight: 1, Quest: q.QID, Ident: true,
				Lore: "An old family treasure, worn smooth by hands long dead."}
			if x, y, ok := zn.FreeSpot(r, nil); ok {
				it.X, it.Y = x, y
			}
			zn.Items = append(zn.Items, it)
			for i := 0; i < 2; i++ {
				m := g.Spawn(pickCreature(r, zn.Biome, 's', zn.Level+1), zn.Level+1, r)
				m.X, m.Y = it.X+r.Int(-3, 3), it.Y+r.Int(-2, 2)
				if !zn.Walkable(m.X, m.Y) || zn.ActorAt(m.X, m.Y) != nil {
					if x, y, ok := zn.FreeSpot(r, nil); ok {
						m.X, m.Y = x, y
					}
				}
				zn.Actors = append(zn.Actors, m)
			}
		case "peace", "debt":
			bd := g.W.burgByID[zn.Burg]
			if bd == nil {
				bd = g.W.burgByID[g.W.Burg[q.TP]]
			}
			if bd == nil {
				q.Placed = false
				continue
			}
			n := g.makeNPC("villager", bd, g.W.cultByID[bd.Culture], r)
			if q.Kind == "peace" {
				n.Name = q.Foe[strings.Index(q.Foe, " ")+1:]
				n.Title = "estranged " + q.Foe[:strings.Index(q.Foe, " ")] + " of " + q.Name
			} else {
				n.Name = q.Foe
				n.Title = "moneylender"
			}
			n.Quest = q.QID
			spot(n)
			q.NPC = n.ID
		}
		if q.NPC != 0 || q.Kind == "heirloom" {
			g.Msg(fmt.Sprintf("{{Y|%s}}", g.cqArrive(q)))
		}
	}
}

func (g *Game) cqArrive(q *cqQuest) string {
	switch q.Kind {
	case "avenge":
		return q.Foe + " is somewhere near. " + q.Name + "'s quarry."
	case "heirloom":
		return "Somewhere here lies the " + q.Item + "."
	case "peace":
		return q.Name + "'s " + q.Foe + " lives here."
	}
	return q.Foe + ", " + q.Name + "'s creditor, keeps a table here."
}

// cqKill: the avenged foe falls (kill hook).
func (g *Game) cqKill(d, src *Actor) {
	for _, q := range g.cq().Q {
		if q.Stage == 1 && q.Kind == "avenge" && q.NPC != 0 && d.ID == q.NPC {
			q.Stage = 2
			g.cqMirrorDone(q)
			g.Msg(fmt.Sprintf("{{G|%s is dead. %s will want to hear it from you.}}", q.Foe, q.Name))
			if a := g.findActor(q.Comp); a != nil && isFollower(a) {
				g.cqComplete(a, q)
			}
		}
	}
}

// cqTalk: options on the relative or the creditor (talk hook).
func (g *Game) cqTalk(n *Actor) []Option {
	var out []Option
	for _, q := range g.cq().Q {
		if q.Stage != 1 || q.NPC == 0 || n.ID != q.NPC {
			continue
		}
		q := q
		comp := g.findActor(q.Comp)
		here := comp != nil && isFollower(comp)
		switch q.Kind {
		case "peace":
			if !here {
				out = append(out, Option{fmt.Sprintf("{{K|%s sent me.}}", q.Name), func() {
					g.talkMenu(fmt.Sprintf("\"%s? Then let them come and say it to my face.\" {{K|(Bring %s here.)}}", givenOf(q.Name), q.Name))
				}})
				continue
			}
			out = append(out, Option{fmt.Sprintf("{{G|%s has come to make peace.}}", q.Name), func() {
				q.Stage = 2
				g.cqMirrorDone(q)
				g.Mode, g.Talk = "main", nil
				g.Msg(fmt.Sprintf("{{G|%s and %s talk long into the evening. Old words are taken back; they part with an embrace.}}", q.Name, n.Name))
				g.cqComplete(comp, q)
			}})
		case "debt":
			out = append(out, Option{fmt.Sprintf("{{W|Settle %s's debt.}} {{K|(%s)}}", givenOf(q.Name), g.cuFormat(q.Gold)), func() {
				if g.P.Gold < q.Gold || !g.cuSpend(q.Gold) {
					g.talkMenu("\"That's not enough. Come back with the full sum.\"")
					return
				}
				q.Stage = 2
				g.cqMirrorDone(q)
				g.Mode, g.Talk = "main", nil
				g.Msg(fmt.Sprintf("{{G|%s counts the coin twice and strikes %s's name from the ledger.}}", n.Name, q.Name))
				if here {
					g.cqComplete(comp, q)
				}
			}})
		}
	}
	return out
}

var cqPerkText = map[string]string{"STR": "+2 Strength", "TOU": "+2 Toughness, +1 AV", "HP": "+10 HP, +2 Ego", "AGI": "+2 Agility, +1 DV"}

// cqComplete: the companion's matter is settled; loyalty, a perk, XP and a deed.
func (g *Game) cqComplete(a *Actor, q *cqQuest) {
	if q.Stage >= 3 {
		return
	}
	q.Stage = 3
	if mq := g.cqMirror(q); mq != nil {
		mq.Done, mq.Closed = true, true
	}
	if q.Kind == "heirloom" {
		for _, it := range g.P.Inv {
			if it.Quest == q.QID {
				g.removeInv(it)
				break
			}
		}
	}
	g.addLoyalty(a, 30)
	switch q.Perk {
	case "STR":
		a.Attr[STR] += 2
	case "TOU":
		a.Attr[TOU] += 2
		a.AVNat++
	case "HP":
		a.MaxHP += 10
		a.HP += 10
		a.Attr[EGO] += 2
	case "AGI":
		a.Attr[AGI] += 2
		a.DVNat++
	}
	g.gainXP(80 + g.P.Level*15)
	line := map[string]string{
		"avenge":   "\"It's over. My %s can sleep now. I owe you more than I can say.\"",
		"heirloom": "\"This was my %s's, and theirs before that. You've given my family back its past.\"",
		"peace":    "\"I've carried that quarrel for years. It's lighter now. Thank you.\"",
		"debt":     "\"My name is clean again in %s. I'll repay you in blood if need be.\"",
	}[q.Kind]
	arg := nonEmpty(firstKin(a.Comp), "family")
	if q.Kind == "debt" {
		arg = nonEmpty(a.Comp.Town, "home")
	}
	if strings.Contains(line, "%s") {
		line = fmt.Sprintf(line, arg)
	}
	g.Msg(fmt.Sprintf("{{G|Personal quest complete: %s.}} {{K|(%s: %s, loyalty +30)}}", q.Title, a.Name, cqPerkText[q.Perk]))
	deed := map[string]string{
		"avenge":   "%s, companion of the wanderer %s, avenged %s kin upon %s.",
		"heirloom": "%s, companion of the wanderer %s, recovered %s family's %s.",
		"peace":    "%s, companion of the wanderer %s, made peace with %s %s.",
		"debt":     "The wanderer %[2]s paid the debt of %[1]s to %[4]s.",
	}[q.Kind]
	obj := q.Foe
	if q.Kind == "heirloom" {
		obj = q.Item
	}
	g.deed("companion", fmt.Sprintf(deed, a.Name, g.P.Name, possessiveWord(a), obj), 0, 0)
	if g.Mode == "talk" && g.Talk == a {
		g.companionTalk(a, line)
	} else {
		g.Msg(fmt.Sprintf("{{c|%s:}} %s", givenOf(a.Name), line))
	}
}

func firstKin(c *Companion) string {
	if c == nil || len(c.Kin) == 0 {
		return ""
	}
	return c.Kin[0]
}

func init() {
	turnHooks = append(turnHooks, func(g *Game) { g.cqTick() })
	enterHooks = append(enterHooks, func(g *Game, zn *Zone) { g.cqEnter(zn) })
	killHooks = append(killHooks, func(g *Game, d, src *Actor) { g.cqKill(d, src) })
	talkHooks = append(talkHooks, func(g *Game, a *Actor) []Option { return g.cqTalk(a) })
}
