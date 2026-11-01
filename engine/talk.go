package engine

import (
	"fmt"
	"sort"
	"strings"
)

type Quest struct {
	ID      int    `json:"id"`
	Kind    string `json:"kind"` // slay fetch deliver
	Title   string `json:"title"`
	Text    string `json:"text"`
	Giver   string `json:"giver"`
	GiverID int    `json:"giverId"`
	Faction string `json:"faction"`
	Target  string `json:"target"` // zone key
	TP      int    `json:"tp"`     // parasang
	Tpl     string `json:"tpl"`
	Name    string `json:"name"`
	Spawned bool   `json:"spawned"`
	Done    bool   `json:"done"`
	Closed  bool   `json:"closed"`
	Reward  int    `json:"reward"`
	BurgTo  int    `json:"burgTo"`
	HomeKey string `json:"homeKey"`
	SimMon  int    `json:"simMon"`
	ItemID  int    `json:"itemId"`
}

func (g *Game) openTalk(a *Actor) {
	g.Talk = a
	g.Mode = "talk"
	g.Sel = 0
	g.Popup = nil
	if a.AI == "follower" {
		g.companionTalk(a, g.compGreeting(a))
		return
	}
	greet := g.greeting(a)
	g.talkMenu(greet)
}

func (g *Game) greeting(a *Actor) string {
	rep := g.Rep[a.Faction]
	switch {
	case a.Kinship:
		return fmt.Sprintf("\"Ah, %s, kin of my %s. Well met again.\"", g.P.Name, strings.ToLower(g.Sk.Ritual))
	case rep >= 250:
		return "\"Friend! Your name is spoken well here.\""
	case rep <= -250:
		return "\"...What do you want?\" They eye you warily."
	}
	if sg := g.swGreeting(a); sg != "" {
		return sg
	}
	if fg := g.feastGreeting(a); fg != "" {
		return fg
	}
	if rm := g.attireRemark(a); rm != "" {
		return rm
	}
	if rm := g.augRemark(a); rm != "" {
		return rm
	}
	if cg := g.casteGreeting(a); cg != "" {
		return cg
	}
	lines := []string{"\"Live and drink, wanderer.\"", "\"Travelers are rare. Speak.\"", "\"Mm? Yes?\"", "\"You're not from here.\""}
	return Pick(NewRng(fmt.Sprint(a.ID)), lines)
}

func (g *Game) talkMenu(text string) {
	a := g.Talk
	g.Popup = Wrap(text, 100)
	if vs := g.innerVoices(a); len(vs) > 0 {
		g.Popup = append(Wrap(strings.Join(vs, "\n\n"), 100), append([]string{""}, g.Popup...)...)
	}
	g.Opts = nil
	add := func(t string, f func()) { g.Opts = append(g.Opts, Option{t, f}) }
	add("Who are you?", func() { g.talkMenu(g.aboutNPC(a)) })
	add("What news is there?", func() { g.talkMenu(g.gossip(a)) })
	if g.canJoinFeast(a) {
		_, f := g.feastToday()
		add("{{W|Join the "+f.Name+".}}", func() { g.joinFeast(a) })
	}
	if !a.Kinship {
		add(fmt.Sprintf("{{C|Perform the %s.}}", g.Sk.Ritual), func() { g.ritual(a) })
	} else if !a.Talked && len(g.Secrets) > 0 {
		add("{{c|Share a secret with them.}}", func() { g.shareSecret(a) })
	}
	if a.Kinship && !a.Has("recipes") && a.AI != "follower" {
		add("{{y|Swap recipes.}}", func() { g.swapRecipes(a) })
	}
	if a.Role == "tailor" {
		add("{{O|Have a garment remade (transmog).}}", func() { g.tailorMenu(a) })
	}
	if a.Role == "augmenter" {
		add(fmt.Sprintf("{{O|Install or remove a %s.}}", g.augSkin().Noun), func() { g.augMenu(a) })
	}
	if a.Teach != "" {
		add("{{O|Teach me.}}", func() { g.trainMenu(a) })
	}
	if a.Role == "arena" {
		if len(g.ArenaFoes) > 0 {
			add("{{K|(A bout is already under way.)}}", func() { g.talkMenu(g.greeting(a)) })
		} else {
			add(fmt.Sprintf("{{R|Fight in the arena.}} {{K|(bout %d)}}", g.ArenaWins+1), func() { g.startArena(a) })
		}
	}
	if a.Role == "merchant" || a.Role == "smith" || a.Role == "apothecary" || a.Role == "bookseller" || a.Role == "exotic" || a.Role == "tailor" || len(a.Inv) > 0 {
		add("{{W|Let's trade.}}", func() { g.openTrade(a) })
	}
	for _, q := range g.Quests {
		if q.Done && !q.Closed && q.GiverID == a.ID {
			q := q
			add("{{G|I have done what you asked.}}", func() { g.turnIn(q) })
		}
		if q.Kind == "deliver" && !q.Done && a.Role == "elder" && g.Cur.Burg == q.BurgTo {
			q := q
			add("{{G|I bring a letter for you.}}", func() { g.deliver(q, a) })
		}
	}
	if (a.Role == "elder" || a.Role == "notable") && a.Quest == 0 {
		add("Do you have any work for me?", func() { g.offerQuest(a) })
	}
	if a.Role == "priest" || a.Role == "apothecary" {
		cost := 10 + g.P.Level*3
		add(fmt.Sprintf("Heal my wounds. (%d %s)", cost, g.Sk.Currency), func() {
			if g.P.Gold < cost {
				g.talkMenu("\"You can't afford it.\"")
				return
			}
			g.P.Gold -= cost
			g.P.HP = g.P.MaxHP
			delete(g.P.Status, "bleeding")
			delete(g.P.Status, "poisoned")
			for _, bp := range g.P.Body {
				bp.Severed = false
			}
			g.talkMenu("Your wounds close. You feel whole.")
		})
	}
	if a.Role == "ferry" {
		add("{{B|I want passage by sea.}}", func() { g.ferryMenu(a) })
	}
	if a.Role == "villager" || a.Role == "guard" {
		add("Travel with me.", func() { g.recruit(a) })
	}
	g.Opts = append(g.Opts, g.checkOptions(a)...)
	for _, h := range talkHooks {
		g.Opts = append(g.Opts, h(g, a)...)
	}
	add("Goodbye.", func() { g.Mode = "main"; g.Talk = nil })
	g.Sel = 0
}

func (g *Game) aboutNPC(a *Actor) string {
	s := fmt.Sprintf("\"I am %s", a.Name)
	if a.Title != "" {
		s += ", " + a.Title
	}
	s += ".\""
	if a.Desc != "" {
		s += " " + a.Desc
	}
	if l := g.socialLine(a); l != "" {
		s += "\n\n" + l
	}
	if a.SimPer > 0 {
		for _, pp := range g.W.People {
			if pp.ID == a.SimPer {
				for _, e := range g.W.Events {
					if strings.Contains(e.Text, pp.Name) {
						s += fmt.Sprintf("\n\nIn the year %d: %s", e.Y, e.Text)
						break
					}
				}
			}
		}
	}
	s += fmt.Sprintf("\n\nThey belong to {{Y|%s}} (%s).", g.FactionName(a.Faction), RepWord(g.Rep[a.Faction]))
	if rel := g.W.relByID[a.Religion]; rel != nil {
		s += fmt.Sprintf(" They follow the {{C|%s}}.", rel.Name)
	}
	return s
}

// gossip pulls real events from the simulation near this place.
func (g *Game) gossip(a *Actor) string {
	if s := g.mrGossip(a); s != "" {
		return s
	}
	if s := g.feastGossip(a); s != "" {
		return s
	}
	if s := g.swGossip(a); s != "" {
		return s
	}
	r := NewRng(fmt.Sprintf("%d:%d", a.ID, g.Turn/600))
	w := g.W
	p := w.Idx(g.Cur.WX, g.Cur.WY)
	var cands []int
	for dy := -6; dy <= 6; dy++ {
		for dx := -6; dx <= 6; dx++ {
			if w.In(g.Cur.WX+dx, g.Cur.WY+dy) {
				cands = append(cands, w.eventsAt[w.Idx(g.Cur.WX+dx, g.Cur.WY+dy)]...)
			}
		}
	}
	st := w.State[p]
	for i := len(w.Events) - 1; i >= 0 && len(cands) < 30; i-- {
		if w.Events[i].St == st {
			cands = append(cands, i)
		}
	}
	if len(cands) == 0 {
		return "\"Nothing ever happens here.\""
	}
	sort.Ints(cands)
	// favour recent
	i := cands[len(cands)-1-r.Int(0, mini(len(cands)-1, 8))]
	e := w.Events[i]
	ago := w.Year - e.Y
	when := fmt.Sprintf("%d years ago", ago)
	if ago <= 1 {
		when = "just this year"
	}
	return fmt.Sprintf("\"They say that %s — %s.\"", lowerFirst(e.Text), when)
}

func lowerFirst(s string) string {
	s = strings.TrimSuffix(strings.TrimSpace(s), ".")
	if len(s) > 1 && s[1] >= 'a' && s[1] <= 'z' && !strings.HasPrefix(s, "The ") {
		return s
	}
	if strings.HasPrefix(s, "The ") {
		return "the " + s[4:]
	}
	return s
}

// ritual: Qud's water ritual, re-skinned per genre. Rep with the faction, plus a secret from history.
func (g *Game) ritual(a *Actor) {
	a.Kinship = true
	gain := 50 + g.Skills["persuasion"]*25 + g.P.M(EGO)*5
	if a.SimPer > 0 {
		gain += 50
	}
	text := g.Sk.RitualDesc + " with " + a.Name + ".\n\n"
	g.AddRep(a.Faction, gain, " for the "+g.Sk.Ritual)
	if a.Religion > 0 {
		g.Rep[fmt.Sprintf("faith:%d", a.Religion)] += gain / 2
	}
	sec := g.revealSecret(a)
	text += sec
	g.deed("kinship", fmt.Sprintf("The wanderer %s performed the %s with %s.", g.P.Name, g.Sk.Ritual, a.Name), 0, 0)
	g.gainXP(10)
	g.talkMenu(text)
}

func (g *Game) revealSecret(a *Actor) string {
	w := g.W
	r := NewRng(fmt.Sprintf("secret:%d", a.ID))
	// prefer secrets of places: lairs, relics, ruins, wonders
	type cand struct {
		text, kind string
		p, y       int
	}
	var cs []cand
	cx, cy := g.Cur.WX, g.Cur.WY
	near := func(p int) bool { return abs(p%w.W-cx) <= 14 && abs(p/w.W-cy) <= 10 }
	for _, m := range w.Monsters {
		if near(m.P) && m.Slain {
			cs = append(cs, cand{g.slSecret(&m), "lair", m.P, w.Year})
		} else if near(m.P) && !g.Slain[m.ID] {
			cs = append(cs, cand{fmt.Sprintf("%s, a %s, lairs at %s.", m.Name, strings.ToLower(m.Species), nonEmpty(m.Lair, w.RegionName(m.P))), "lair", m.P, w.Year})
		}
	}
	for _, b := range w.Burgs {
		if !b.Alive && near(b.P) {
			cs = append(cs, cand{fmt.Sprintf("The ruins of %s lie at the edge of %s. Its dead are buried beneath it.", b.Name, w.RegionName(b.P)), "ruin", b.P, w.Year})
		}
	}
	for _, ar := range w.Artifacts {
		if ar.P >= 0 && near(ar.P) && !g.Taken[ar.ID] {
			cs = append(cs, cand{fmt.Sprintf("The relic %s is held by %s, near %s.", ar.Name, nonEmpty(ar.Holder, "the dead"), w.RegionName(ar.P)), "relic", ar.P, ar.Year})
		}
	}
	for dy := -8; dy <= 8; dy++ {
		for dx := -10; dx <= 10; dx++ {
			if !w.In(cx+dx, cy+dy) {
				continue
			}
			p := w.Idx(cx+dx, cy+dy)
			if w.Burg[p] > 0 {
				continue
			}
			if s := g.parasangSite(p); s != nil {
				what := "a dungeon"
				if s.Kind == "tower" {
					what = fmt.Sprintf("a tower of %d floors", s.Depth)
				}
				cs = append(cs, cand{fmt.Sprintf("%s is %s near %s. Few who go in come back rich; fewer come back at all.", s.Name, what, w.RegionName(p)), "dungeon", p, w.Year})
			}
		}
	}
	for _, wd := range w.Wonders {
		if near(wd.P) {
			cs = append(cs, cand{fmt.Sprintf("%s stands at %s.", wd.Name, w.RegionName(wd.P)), "wonder", wd.P, wd.Year})
		}
	}
	for i := len(w.Events) - 1; i >= 0 && len(cs) < 40; i-- {
		e := w.Events[i]
		if e.P >= 0 && near(e.P) && (e.Kind == "war" || e.Kind == "death" || e.Kind == "intrigue" || e.Kind == "fall" || e.Kind == "coup") {
			cs = append(cs, cand{fmt.Sprintf("In the year %d, %s", e.Y, lowerFirst(e.Text)+"."), "history", e.P, e.Y})
		}
	}
	if len(cs) == 0 {
		return "They have no secrets worth sharing."
	}
	for tries := 0; tries < 10; tries++ {
		c := cs[r.Int(0, len(cs)-1)]
		if g.addSecret(c.text, c.kind, c.p, c.y) {
			if g.Skills["lore"] > 0 {
				g.gainXP(5 * g.Skills["lore"])
			}
			return "They lean in and share a secret:\n\n{{W|" + c.text + "}}\n\n(Recorded in your journal; the place is marked on your world map.)"
		}
	}
	return "They have nothing you don't already know."
}

func (g *Game) shareSecret(a *Actor) {
	a.Talked = true
	gain := 40 + g.Skills["lore"]*20
	s := g.Secrets[g.R.Int(0, len(g.Secrets)-1)]
	g.AddRep(a.Faction, gain, " for sharing a secret")
	g.talkMenu(fmt.Sprintf("You tell them: \"%s\"\n\nThey nod slowly. \"I will remember this.\"", s.Text))
}

func (g *Game) recruit(a *Actor) {
	if g.companionCount() >= maxCompanions {
		g.talkMenu("\"You've enough company already.\"")
		return
	}
	need := 600
	if a.Kinship {
		need -= 300
	}
	need -= g.Skills["persuasion"] * 150
	if g.Rep[a.Faction] < need {
		g.talkMenu(fmt.Sprintf("\"I don't know you well enough.\" (needs reputation %d with %s, or the %s and Persuasion)", need, g.FactionName(a.Faction), g.Sk.Ritual))
		return
	}
	g.joinCompanion(a)
	g.Mode = "main"
}

func (g *Game) ferryMenu(a *Actor) {
	w := g.W
	type port struct {
		b    *BurgData
		cost int
	}
	var ps []port
	cx, cy := g.Cur.WX, g.Cur.WY
	for i := range w.Burgs {
		b := &w.Burgs[i]
		if !b.Alive || !b.Port || b.ID == g.Cur.Burg || b.Layer != "surface" {
			continue
		}
		d := abs(b.P%w.W-cx) + abs(b.P/w.W-cy)
		ps = append(ps, port{b, 10 + d*3})
	}
	sort.Slice(ps, func(i, j int) bool { return ps[i].cost < ps[j].cost })
	if len(ps) > 12 {
		ps = ps[:12]
	}
	g.Popup = Wrap("\"Where to? I know every harbour on these waters.\"", 100)
	g.Opts = nil
	for _, p := range ps {
		p := p
		cost := p.cost * (100 - g.Skills["persuasion"]*8) / 100
		g.Opts = append(g.Opts, Option{fmt.Sprintf("%s (%s) — %d %s", p.b.Name, w.StateName(p.b.State), cost, g.Sk.Currency), func() {
			if g.P.Gold < cost {
				g.talkMenu("\"Coin first, wanderer.\"")
				return
			}
			g.P.Gold -= cost
			g.Turn += 600
			g.Hunger += 200
			g.Thirst += 200
			g.Mode = "main"
			g.Talk = nil
			g.Known[p.b.P] = true
			g.Msg(fmt.Sprintf("You %s to {{Y|%s}}.", g.Sk.Ferry, p.b.Name))
			g.enterZone(g.GetZone(p.b.P%w.W, p.b.P/w.W, 1, 1, 0), 40, 13, true)
			g.recompute()
		}})
	}
	g.Opts = append(g.Opts, Option{"Never mind.", func() { g.talkMenu(g.greeting(a)) }})
	g.Sel = 0
}

// ---- quests ----

func (g *Game) offerQuest(a *Actor) {
	w := g.W
	r := NewRng(fmt.Sprintf("quest:%d", a.ID))
	q := &Quest{ID: g.newID(), Giver: a.Name, GiverID: a.ID, Faction: a.Faction, HomeKey: g.Cur.Key, Reward: 60 + g.P.Level*25}
	cx, cy := g.Cur.WX, g.Cur.WY
	// 1. a real monster from the simulation nearby
	for _, m := range w.Monsters {
		if !g.Slain[m.ID] && !m.Slain && abs(m.P%w.W-cx) <= 10 && abs(m.P/w.W-cy) <= 8 && m.Layer == "surface" && r.Chance(0.7) {
			q.Kind, q.SimMon, q.TP = "slay", m.ID, m.P
			q.Title = "Slay " + m.Name
			q.Text = fmt.Sprintf("\"%s has plagued us for years. It lairs at %s. Bring us proof of its death.\"", m.Name, nonEmpty(m.Lair, w.RegionName(m.P)))
			q.Reward = 300 + m.Power*4
			break
		}
	}
	if q.Kind == "" {
		switch r.Int(0, 2) {
		case 0, 1:
			// bandits or a beast in the wilds a few parasangs away
			tx, ty := clampi(cx+r.Int(-4, 4), 0, w.W-1), clampi(cy+r.Int(-3, 3), 0, w.H-1)
			for k := 0; k < 20 && (w.IsWater(w.Idx(tx, ty)) || w.Burg[w.Idx(tx, ty)] > 0); k++ {
				tx, ty = clampi(cx+r.Int(-5, 5), 0, w.W-1), clampi(cy+r.Int(-4, 4), 0, w.H-1)
			}
			q.TP = w.Idx(tx, ty)
			zx, zy := r.Int(0, 2), r.Int(0, 2)
			q.Target = ZoneKey(tx, ty, zx, zy, 0)
			if r.Chance(0.5) {
				q.Kind, q.Tpl = "slay", "chief"
				q.Name = g.cultureName(w.cultByID[w.Culture[q.TP]], r) + " the " + Pick(r, []string{"Red", "Cruel", "One-Eyed", "Butcher", "Grinning", "Wolf"})
				q.Title = "Kill " + q.Name
				q.Text = fmt.Sprintf("\"%s and their %s have been raiding the roads near %s. End them.\"", q.Name, strings.ToLower(g.Sk.Bandits), w.RegionName(q.TP))
			} else {
				q.Kind = "fetch"
				q.Name = "heirloom of " + a.Name
				q.Title = "Recover the " + q.Name
				q.Text = fmt.Sprintf("\"My family's heirloom was lost in the %s near %s. Find it.\"", BiomeName(w.Biome[q.TP]), w.RegionName(q.TP))
			}
		default:
			// deliver a letter to another town's elder
			var bs []*BurgData
			for i := range w.Burgs {
				b := &w.Burgs[i]
				if b.Alive && b.Layer == "surface" && b.ID != g.Cur.Burg && abs(b.P%w.W-cx) <= 12 && abs(b.P/w.W-cy) <= 9 {
					bs = append(bs, b)
				}
			}
			if len(bs) == 0 {
				g.talkMenu("\"Nothing for now.\"")
				return
			}
			b := Pick(r, bs)
			q.Kind, q.BurgTo, q.TP = "deliver", b.ID, b.P
			q.Title = "Carry a letter to " + b.Name
			q.Text = fmt.Sprintf("\"Take this sealed letter to the elder of %s, in %s. Tell no one.\"", b.Name, w.StateName(b.State))
			g.give(&Item{ID: g.newID(), Kind: "letter", Name: "sealed letter for " + b.Name, Glyph: '■', F: CWhite, Quest: q.ID})
		}
	}
	a.Quest = q.ID
	g.Quests = append(g.Quests, q)
	g.Known[q.TP] = true
	g.Msg("{{W|New quest: " + q.Title + "}}")
	g.talkMenu(q.Text + "\n\n{{K|(Quest added to your journal; destination marked on the world map.)}}")
}

func (g *Game) questSpawn(zn *Zone) {
	for _, q := range g.Quests {
		if q.Done || q.Spawned || q.Target != zn.Key {
			continue
		}
		q.Spawned = true
		r := NewRng(fmt.Sprint(q.ID))
		switch q.Kind {
		case "slay":
			boss := g.Spawn(q.Tpl, zn.Level+2, r)
			boss.Name, boss.Proper, boss.Quest = q.Name, true, q.ID
			g.nudge(zn, boss)
			if x, y, ok := zn.FreeSpot(r, nil); ok {
				boss.X, boss.Y = x, y
			}
			zn.Actors = append(zn.Actors, boss)
			for i := 0; i < r.Int(2, 4); i++ {
				m := g.Spawn(Pick(r, []string{"bandit", "archer"}), zn.Level, r)
				if x, y, ok := zn.FreeSpot(r, func(x, y int) bool { return abs(x-boss.X) < 8 && abs(y-boss.Y) < 5 }); ok {
					m.X, m.Y = x, y
					zn.Actors = append(zn.Actors, m)
				}
			}
		case "fetch":
			it := g.Loot(r, zn.Level/3+2)
			it.Name = q.Name
			it.Quest = q.ID
			it.F = CYellow
			if x, y, ok := zn.FreeSpot(r, nil); ok {
				it.X, it.Y = x, y
				zn.Items = append(zn.Items, it)
			}
			for i := 0; i < 2; i++ {
				m := g.Spawn(pickCreature(r, zn.Biome, 's', zn.Level+1), zn.Level+1, r)
				if x, y, ok := zn.FreeSpot(r, nil); ok {
					m.X, m.Y = x, y
					zn.Actors = append(zn.Actors, m)
				}
			}
		}
	}
}

func (g *Game) questKill(d *Actor) {
	for _, q := range g.Quests {
		if q.Done {
			continue
		}
		if q.Kind == "slay" && (d.Quest == q.ID || q.SimMon > 0 && d.SimMon == q.SimMon) {
			q.Done = true
			g.Msg("{{G|Quest complete: " + q.Title + ". Return to " + q.Giver + ".}}")
		}
	}
}

func (g *Game) questPickup(it *Item) {
	for _, q := range g.Quests {
		if !q.Done && q.Kind == "fetch" && it.Quest == q.ID {
			q.Done = true
			g.Msg("{{G|You found the " + q.Name + ". Return to " + q.Giver + ".}}")
		}
	}
}

func (g *Game) deliver(q *Quest, a *Actor) {
	for _, it := range g.P.Inv {
		if it.Quest == q.ID {
			g.removeInv(it)
			break
		}
	}
	q.Done, q.Closed = true, true
	g.P.Gold += q.Reward
	g.gainXP(q.Reward / 2)
	g.AddRep(a.Faction, 75, " for your discretion")
	g.AddRep(q.Faction, 75, " for the delivered letter")
	g.deed("quest", fmt.Sprintf("The wanderer %s carried a sealed letter from %s to %s.", g.P.Name, q.Giver, a.Name), 0, 0)
	g.talkMenu(fmt.Sprintf("They break the seal, read, and burn it. \"Here. %d %s for your trouble.\"", q.Reward, g.Sk.Currency))
}

func (g *Game) turnIn(q *Quest) {
	q.Closed = true
	if q.Kind == "fetch" {
		for _, it := range g.P.Inv {
			if it.Quest == q.ID {
				g.removeInv(it)
				break
			}
		}
	}
	g.P.Gold += q.Reward
	g.gainXP(q.Reward)
	g.AddRep(q.Faction, 150, " for completing a quest")
	reward := g.Loot(g.R, g.P.Level/3+2)
	g.give(reward)
	g.deed("quest", fmt.Sprintf("The wanderer %s completed a task for %s: %s.", g.P.Name, q.Giver, q.Title), 0, 0)
	g.talkMenu(fmt.Sprintf("\"You have done us a great service.\" They give you %d %s and %s.", q.Reward, g.Sk.Currency, reward.Display()))
}

// ---- lore ----

func (g *Game) annalsBook(state int) *Item {
	w := g.W
	st := w.stateByID[state]
	name := "the wilds"
	if st != nil {
		name = st.Name
	}
	var lines []string
	for i := len(w.Events) - 1; i >= 0 && len(lines) < 14; i-- {
		e := w.Events[i]
		if e.St == state {
			lines = append(lines, fmt.Sprintf("{{K|Y%d}} %s", e.Y, e.Text))
		}
	}
	for i, j := 0, len(lines)-1; i < j; i, j = i+1, j-1 {
		lines[i], lines[j] = lines[j], lines[i]
	}
	text := fmt.Sprintf("{{Y|The Annals of %s}}\n\n%s", name, strings.Join(lines, "\n"))
	return g.MakeBook("Annals of "+name, text)
}

func (g *Game) interact(x, y int) {
	if inBump(g, x, y) { // stat gates, shrines, glyphs (gates.go / insight.go)
		return
	}
	z := g.Cur
	t := z.At(x, y)
	i := y*MW + x
	note := z.Notes[i]
	switch t {
	case TPortal:
		g.unBump(x, y) // planets.go
	case TInscription, TSarcophagus, TStatue:
		if note == "" {
			note = "Worn smooth; you can't read it."
		}
		k := z.Key + fmt.Sprint(i)
		if !g.Read[k] {
			g.Read[k] = true
			xp := 5 + g.Skills["lore"]*10
			g.gainXP(xp)
			if g.Skills["lore"] > 0 {
				note += fmt.Sprintf("\n\n{{c|Your study of lore yields insight. +%d XP}}", xp)
			}
			g.addSecret(Plain(note), "inscription", g.W.Idx(z.WX, z.WY), g.W.Year)
		}
		g.showPopup(t.D().Name, note)
		if t == TSarcophagus && z.Kind == "crypt" && g.R.Chance(0.25) && !g.Read[k+"risen"] {
			g.Read[k+"risen"] = true
			a := g.Spawn("wight", z.Level, g.R)
			a.X, a.Y = x, y+1
			g.nudge(z, a)
			z.Actors = append(z.Actors, a)
			g.Msg("{{R|The lid grinds aside. Something climbs out!}}")
		}
	case TAltar:
		g.showPopup("Altar", nonEmpty(note, "A small shrine.")+"\n\nYou pray a while.")
		if g.P.HP < g.P.MaxHP {
			g.P.HP = mini(g.P.MaxHP, g.P.HP+5)
		}
	case TWell, TShallow:
		g.drinkHere()
	case TShelf:
		k := z.Key + fmt.Sprint(i)
		if g.Read[k] {
			g.Msg("You've already searched this shelf.")
			return
		}
		g.Read[k] = true
		st := g.W.State[g.W.Idx(z.WX, z.WY)]
		_ = st
		b := g.RandomBook(g.R, g.W.Idx(z.WX, z.WY))
		g.give(b)
		g.Msg("You find a " + g.Sk.Book + ": {{C|" + b.Name + "}}.")
	}
	g.spend(1000)
}

func (g *Game) showPopup(title, text string) {
	g.PopT = title
	g.Popup = Wrap(text, 70)
	g.Mode = "popup"
}

func (g *Game) drinkHere() {
	if g.lqDrinkHere() {
		return
	}
	g.Thirst = 0
	filled := false
	for _, it := range g.P.Inv {
		if it.Kind == "water" && it.Charges < it.Power {
			it.Charges = it.Power
			filled = true
		}
	}
	if filled {
		g.Msg("You drink deeply and fill your " + g.Sk.Water + ".")
	} else {
		g.Msg("You drink deeply.")
	}
}

func (g *Game) dig(x, y int) {
	g.Cur.Set(x, y, TRubble)
	g.Msg("You dig through.")
	g.spend(2000)
}

// ---- trade ----

func (g *Game) price(it *Item, buy bool) int {
	v := maxi(it.Value, 1)
	v = maxi(1, int(float64(v)*g.wsItemMul(it))) // regional market (worldsim.go)
	if g.Talk != nil {
		tr := thTraitRep(g, g.Talk) // how your traits sit with them (traits.go)
		if buy {
			v = maxi(1, v*(100-3*tr)/100)
		} else {
			v = maxi(1, v*(100+3*tr)/100)
		}
	}
	if it.Count > 1 && it.Stack() {
		v = maxi(it.Value, 1)
	}
	ego := float64(g.P.M(EGO)) * 0.03
	per := float64(g.Skills["persuasion"]) * 0.07
	rep := float64(clampi(g.Rep[g.Talk.Faction], -500, 800)) / 8000
	if buy {
		f := 1.4 - ego - per - rep
		if g.Talk.Has("disc") {
			f -= 0.15
		}
		if g.Talk.Has("markup") {
			f += 0.15
		}
		f *= g.castePriceMod(g.Talk) * g.augPriceMod(g.Talk) * g.swPriceMod(g.Talk, it, true)
		if f < 0.75 {
			f = 0.75
		}
		return maxi(1, int(float64(v)*f*g.lqPriceFactor(true)))
	}
	f := 0.4 + ego + per + rep
	f *= g.uwStolenFactor(it, buy) // honest merchants pay half for stolen goods (crime.go)
	f /= g.castePriceMod(g.Talk) * g.augPriceMod(g.Talk)
	f *= g.swPriceMod(g.Talk, it, false)
	if f > 0.8 {
		f = 0.8
	}
	return maxi(1, int(float64(v)*f*g.lqPriceFactor(false)))
}

func (g *Game) openTrade(a *Actor) {
	g.feastStock(a)
	g.Talk = a
	g.Mode = "trade"
	g.Tab = 0
	g.Sel = 0
}

func (g *Game) tradeAct() {
	a := g.Talk
	if g.Tab == 0 {
		if g.Sel >= len(a.Inv) {
			return
		}
		it := a.Inv[g.Sel]
		one := it
		pr := g.price(it, true)
		if !g.cuCanSpend(pr) { // currency.go: local money first, foreign coin only where accepted
			g.Msg("You can't afford " + it.Display() + ".")
			return
		}
		if !g.cuSpend(pr) {
			return
		}
		a.Gold += pr
		if it.Stack() && it.Count > 1 {
			it.Count--
			cp := *it
			cp.Count = 1
			cp.ID = g.newID()
			one = &cp
		} else {
			a.Inv = append(a.Inv[:g.Sel], a.Inv[g.Sel+1:]...)
		}
		g.give(one)
		g.Msg(fmt.Sprintf("You buy %s for %s.", one.Display(), g.cuFormat(pr)))
	} else {
		if g.Sel >= len(g.P.Inv) {
			return
		}
		it := g.P.Inv[g.Sel]
		if g.uwNoSale(it, a) { // honest merchants refuse stolen or banned goods
			return
		}
		if it.Quest > 0 {
			g.Msg("You can't sell that.")
			return
		}
		pr := g.price(it, false)
		if a.Gold < pr {
			g.Msg("They can't afford it.")
			return
		}
		a.Gold -= pr
		g.cuEarn(pr)
		one := it
		if it.Stack() && it.Count > 1 {
			it.Count--
			cp := *it
			cp.Count = 1
			one = &cp
		} else {
			g.removeInv(it)
		}
		a.Inv = append(a.Inv, one)
		if one.SimArt > 0 {
			g.deed("relic", fmt.Sprintf("The relic %s was sold by the wanderer %s to %s.", one.Name, g.P.Name, a.Name), 0, one.SimArt)
		}
		g.Msg(fmt.Sprintf("You sell %s for %s.", one.Display(), g.cuFormat(pr)))
	}
}

// ---- trainers ----

func trainCost(rank int) int { return 80 * (rank + 1) * (rank + 1) }

func (g *Game) trainMenu(a *Actor) {
	g.Popup = Wrap("\"I can teach you, for a price. Past the second rank you must find your own way.\"", 100)
	g.Opts = nil
	for _, sk := range strings.Split(a.Teach, ",") {
		sk := sk
		rk := g.Skills[sk]
		if rk >= 2 {
			g.Opts = append(g.Opts, Option{fmt.Sprintf("{{K|%s — rank %d, nothing more to learn here}}", skillName(sk), rk), func() { g.trainMenu(a) }})
			continue
		}
		cost := trainCost(rk) * (100 - g.Skills["persuasion"]*10) / 100
		g.Opts = append(g.Opts, Option{fmt.Sprintf("%s to rank %d — %d %s", skillName(sk), rk+1, cost, g.Sk.Currency), func() {
			if g.P.Gold < cost {
				g.talkMenu("\"Come back with coin.\"")
				return
			}
			g.P.Gold -= cost
			g.Skills[sk]++
			g.Turn += 240
			g.Msg(fmt.Sprintf("{{G|Hours of drills later, %s rises to rank %d.}}", skillName(sk), g.Skills[sk]))
			g.trainMenu(a)
		}})
	}
	g.Opts = append(g.Opts, Option{"Enough for now.", func() { g.talkMenu(g.greeting(a)) }})
	g.Sel = 0
}

// ---- arena ----

var arenaFoes = []string{"wolf", "bandit", "boar", "scorpion", "jaguar", "bear", "golem", "yeti", "salamander", "chief", "crystalgolem", "horror"}

func (g *Game) startArena(master *Actor) {
	z := g.Cur
	n := mini(1+g.ArenaWins/2, 4)
	lvl := g.P.Level + g.ArenaWins/2
	tpl := arenaFoes[mini(g.ArenaWins, len(arenaFoes)-1)]
	g.ArenaFoes = nil
	for i := 0; i < n; i++ {
		a := g.Spawn(tpl, lvl, g.R)
		a.Faction, a.Angry = "arena", true
		a.X, a.Y = 36+i*3, 6
		g.nudge(z, a)
		z.Actors = append(z.Actors, a)
		g.ArenaFoes = append(g.ArenaFoes, a.ID)
	}
	g.ArenaKey = z.Key
	g.P.X, g.P.Y = 40, 17
	if !z.Walkable(40, 17) {
		if x, y, ok := g.nearestFree(z, 40, 17); ok {
			g.P.X, g.P.Y = x, y
		}
	}
	g.Mode, g.Talk = "main", nil
	g.Msg(fmt.Sprintf("{{R|The crowd roars! Bout %d: %d %s.}}", g.ArenaWins+1, n, tplByID[tpl].Name(g.Genre)))
	g.recompute()
}

func (g *Game) arenaCheck() {
	if len(g.ArenaFoes) == 0 {
		return
	}
	z := g.Zones[g.ArenaKey]
	if z == nil {
		g.ArenaFoes = nil
		return
	}
	alive := 0
	for _, a := range z.Actors {
		for _, id := range g.ArenaFoes {
			if a.ID == id && a.HP > 0 {
				alive++
			}
		}
	}
	if g.Cur != z && alive > 0 {
		keep := z.Actors[:0]
		for _, a := range z.Actors {
			if a.Faction != "arena" {
				keep = append(keep, a)
			}
		}
		z.Actors = keep
		g.ArenaFoes = nil
		g.Msg("{{K|You fled the arena. The crowd jeers.}}")
		return
	}
	if alive > 0 {
		return
	}
	g.ArenaWins++
	purse := 40 * g.ArenaWins * (g.ArenaWins + 1)
	g.P.Gold += purse
	g.gainXP(30 * g.ArenaWins)
	g.ArenaFoes = nil
	g.Msg(fmt.Sprintf("{{W|Victory! The crowd showers you with %d %s.}}", purse, g.Sk.Currency))
	g.AddRep(z.Faction, 25, " for your arena victory")
	if g.ArenaWins == 5 {
		g.Msg("{{Y|You are hailed as Champion of the Arena!}}")
		g.deed("quest", fmt.Sprintf("The wanderer %s became champion of the arena in %s.", g.P.Name, z.Name), 0, 0)
	}
}
