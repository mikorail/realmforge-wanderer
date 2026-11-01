package engine

import (
	"fmt"
	"strings"
)

// Water ritual v2 (WP5): after the base ritual (talk.go) the person is kin and a "Speak as
// kin" option opens a menu: how they regard you (reputation bands and what each unlocks),
// secrets for gossip, coin, drams of water or reputation, places, a skill / recipe / power from
// their people, and recruiting a legendary. Rituals are counted per faction.

func lgBand(rep int) int {
	switch {
	case rep >= 900:
		return 5
	case rep >= 600:
		return 4
	case rep >= 250:
		return 3
	case rep > -250:
		return 2
	case rep > -600:
		return 1
	}
	return 0
}

var lgBandNames = []string{"Hated", "Disliked", "Indifferent", "Liked", "Loved", "Revered"}
var lgBandCols = []string{"R", "r", "y", "g", "G", "W"}
var lgBandUnlock = []string{
	"they will not speak to you",
	"only the ritual and plain talk",
	"gossip and trade",
	"secrets, places and a lesson in a skill",
	"recipes of their people",
	"a gift of power, and legendary kin will travel with you",
}

func lgBandWord(rep int) string {
	b := lgBand(rep)
	return "{{" + lgBandCols[b] + "|" + lgBandNames[b] + "}}"
}

func lgTalk(g *Game, a *Actor) []Option {
	st := lgSt(g)
	if a.Kinship && !st.Counted[a.ID] {
		st.Counted[a.ID] = true
		st.Rit[a.Faction]++
		if n := st.Rit[a.Faction]; n%3 == 0 {
			g.AddRep(a.Faction, 60, fmt.Sprintf(" (%d rites shared with their people)", n))
		}
	}
	if !a.Kinship || a.AI == "follower" {
		return nil
	}
	return []Option{{"{{C|Speak as kin.}}", func() { g.lgKinMenu(a, "\"Ask, then, as kin.\"") }}}
}

func (g *Game) lgKinMenu(a *Actor, text string) {
	rep := g.Rep[a.Faction]
	b := lgBand(rep)
	g.Popup = Wrap(text, 100)
	g.Opts = nil
	add := func(t string, f func()) { g.Opts = append(g.Opts, Option{t, f}) }
	locked := func(t string, need int) {
		add("{{K|"+t+" (needs "+lgBandNames[need]+")}}", func() { g.lgKinMenu(a, "\"Not yet. You are "+lgBandNames[b]+" to me.\"") })
	}
	add("How do they regard me?", func() { g.lgKinMenu(a, g.lgStanding(a)) })
	if !a.Has("lg:gossip") {
		add(fmt.Sprintf("Tell them what you have heard. {{K|(+%d)}}", 60+10*g.Skills["persuasion"]), func() { g.lgGossip(a) })
	}
	cost := g.lgSecretCost()
	dr := maxi(2, cost/8)
	if b >= 3 {
		add(fmt.Sprintf("Buy a secret. (%d %s)", cost, g.Sk.Currency), func() { g.lgBuy(a, "coin", cost) })
		add(fmt.Sprintf("Buy a secret with water. (%d drams)", dr), func() { g.lgBuy(a, "water", dr) })
		add("Ask for a secret. {{K|(-50 reputation)}}", func() { g.lgBuy(a, "rep", 50) })
		add("Ask where other people live. {{K|(-30 reputation)}}", func() { g.lgPlace(a) })
	} else {
		locked("Buy or ask for secrets", 3)
	}
	if sk := g.lgSkillFor(a); sk != "" && b >= 3 {
		rk := g.Skills[sk]
		c := 150 * (rk + 1)
		if rk >= 2 || a.Has("lg:skill") {
			add("{{K|Learn "+skillName(sk)+" (nothing more to learn from them)}}", func() { g.lgKinMenu(a, "\"I have shown you all I can.\"") })
		} else {
			add(fmt.Sprintf("Learn %s to rank %d. {{K|(-%d reputation)}}", skillName(sk), rk+1, c), func() { g.lgLearnSkill(a, sk, c) })
		}
	} else if b < 3 {
		locked("Learn a skill from them", 3)
	}
	if b >= 4 {
		if rc := g.lgRecipeFor(a); rc != nil && !a.Has("lg:recipe") {
			add(fmt.Sprintf("Learn to make %s. {{K|(-200 reputation)}}", rc.Name), func() { g.lgLearnRecipe(a, rc) })
		}
	} else {
		locked("Learn a recipe", 4)
	}
	if b >= 5 {
		if id := g.lgPowerFor(a); id != "" && !a.Has("lg:power") {
			add(fmt.Sprintf("Receive %s from them. {{K|(-300 reputation)}}", g.lgPowerName(id)), func() { g.lgLearnPower(a, id) })
		}
	} else {
		locked("Receive a gift of power", 5)
	}
	if a.Has("legend") {
		need := maxi(50, a.Level*20)
		if b >= 5 || b >= 3 && rep >= 250+need {
			add(fmt.Sprintf("Ask them to travel with you. {{K|(-%d reputation)}}", need), func() { g.lgRecruit(a, need) })
		} else {
			add(fmt.Sprintf("{{K|Ask them to travel with you (needs %d reputation)}}", 250+need), func() { g.lgKinMenu(a, "\"Earn more of my trust first.\"") })
		}
	}
	add("Back.", func() { g.talkMenu(g.greeting(a)) })
	g.Sel = 0
}

func (g *Game) lgStanding(a *Actor) string {
	st := lgSt(g)
	rep := g.Rep[a.Faction]
	var sb strings.Builder
	fmt.Fprintf(&sb, "Among %s you are %s (%d).\n", g.FactionName(a.Faction), lgBandWord(rep), rep)
	if n := st.Rit[a.Faction]; n > 0 {
		fmt.Fprintf(&sb, "You have shared the %s with their people %d time(s).\n", g.Sk.Ritual, n)
	}
	if w, ok := st.Opin[a.ID]; ok {
		fmt.Fprintf(&sb, "They themselves %s you.\n", map[string]string{"love": "dearly love", "like": "like", "dislike": "distrust", "hate": "despise"}[w])
	}
	sb.WriteString("\n")
	for i, u := range lgBandUnlock {
		mark := "{{K|"
		if i <= lgBand(rep) {
			mark = "{{" + lgBandCols[i] + "|"
		}
		fmt.Fprintf(&sb, "%s%s: %s}}\n", mark, lgBandNames[i], u)
	}
	return strings.TrimRight(sb.String(), "\n")
}

func (g *Game) lgSecretCost() int {
	c := 30 + 12*g.P.Level
	return c * (100 - g.Skills["persuasion"]*8) / 100
}

func (g *Game) lgGossip(a *Actor) {
	a.Set("lg:gossip")
	gain := 60 + 10*g.Skills["persuasion"]
	g.AddRep(a.Faction, gain, " for news")
	g.gainXP(5)
	g.lgKinMenu(a, "You trade what you have heard on the road. They listen closely.\n\n"+g.gossip(a))
}

// lgDrams is the water you carry, in drams (charges across your water containers).
func (g *Game) lgDrams() int {
	n := 0
	for _, it := range g.P.Inv {
		if it.Kind == "water" {
			n += it.Charges
		}
	}
	return n
}

func (g *Game) lgSpendDrams(n int) bool {
	if g.lgDrams() < n {
		return false
	}
	for _, it := range g.P.Inv {
		if it.Kind != "water" {
			continue
		}
		t := mini(it.Charges, n)
		it.Charges -= t
		n -= t
		if n == 0 {
			break
		}
	}
	return true
}

func (g *Game) lgBuy(a *Actor, how string, price int) {
	switch how {
	case "coin":
		if g.P.Gold < price {
			g.lgKinMenu(a, "\"Come back with coin.\"")
			return
		}
	case "water":
		if g.lgDrams() < price {
			g.lgKinMenu(a, "\"I am thirsty, not charitable. You haven't the water.\"")
			return
		}
	}
	st := lgSt(g)
	text, ok := g.lgSecret(a, st.Buys[a.ID])
	if !ok {
		g.lgKinMenu(a, "\"I have told you everything I know that you do not.\"")
		return
	}
	switch how {
	case "coin":
		g.P.Gold -= price
		a.Gold += price
	case "water":
		g.lgSpendDrams(price)
	case "rep":
		g.AddRep(a.Faction, -price, " for a secret")
	}
	st.Buys[a.ID]++
	if g.Skills["lore"] > 0 {
		g.gainXP(5 * g.Skills["lore"])
	}
	g.lgKinMenu(a, "They lean in and share a secret:\n\n{{W|"+text+"}}\n\n(Recorded in your journal; the place is marked on your world map.)")
}

// lgSecret picks a secret of a place near you that you do not know yet, varying with salt.
func (g *Game) lgSecret(a *Actor, salt int) (string, bool) {
	w := g.W
	type cand struct {
		text, kind string
		p, y       int
	}
	var cs []cand
	cx, cy := g.Cur.WX, g.Cur.WY
	near := func(p int) bool { return abs(p%w.W-cx) <= 22 && abs(p/w.W-cy) <= 16 }
	for _, m := range w.Monsters {
		if near(m.P) && m.Slain {
			cs = append(cs, cand{g.slSecret(&m), "lair", m.P, w.Year})
		} else if near(m.P) && !g.Slain[m.ID] {
			cs = append(cs, cand{fmt.Sprintf("%s, a %s, lairs at %s.", m.Name, strings.ToLower(m.Species), nonEmpty(m.Lair, w.RegionName(m.P))), "lair", m.P, w.Year})
		}
	}
	for _, b := range w.Burgs {
		if !b.Alive && near(b.P) {
			cs = append(cs, cand{fmt.Sprintf("The ruins of %s lie at the edge of %s.", b.Name, w.RegionName(b.P)), "ruin", b.P, w.Year})
		}
	}
	for _, ar := range w.Artifacts {
		if ar.P >= 0 && near(ar.P) && !g.Taken[ar.ID] {
			cs = append(cs, cand{fmt.Sprintf("The relic %s is held by %s, near %s.", ar.Name, nonEmpty(ar.Holder, "the dead"), w.RegionName(ar.P)), "relic", ar.P, ar.Year})
		}
	}
	for _, wd := range w.Wonders {
		if near(wd.P) {
			cs = append(cs, cand{fmt.Sprintf("%s stands at %s.", wd.Name, w.RegionName(wd.P)), "wonder", wd.P, wd.Year})
		}
	}
	for dy := -12; dy <= 12; dy++ {
		for dx := -16; dx <= 16; dx++ {
			if !w.In(cx+dx, cy+dy) {
				continue
			}
			p := w.Idx(cx+dx, cy+dy)
			if w.Burg[p] > 0 {
				continue
			}
			if s := g.parasangSite(p); s != nil {
				cs = append(cs, cand{fmt.Sprintf("%s lies near %s. Few who go in come back rich.", s.Name, w.RegionName(p)), "dungeon", p, w.Year})
			}
		}
	}
	if len(cs) == 0 {
		return "", false
	}
	r := NewRng(fmt.Sprintf("%s:lgsec:%d:%d", g.Seed, a.ID, salt))
	start := r.Int(0, len(cs)-1)
	for i := range cs {
		c := cs[(start+i)%len(cs)]
		if g.addSecret(c.text, c.kind, c.p, c.y) {
			return c.text, true
		}
	}
	return "", false
}

// lgPlace: ask where other people live; marks a living settlement you have not heard of.
func (g *Game) lgPlace(a *Actor) {
	w := g.W
	cx, cy := g.Cur.WX, g.Cur.WY
	type pl struct {
		b *BurgData
		d int
	}
	var ps []pl
	for i := range w.Burgs {
		b := &w.Burgs[i]
		if !b.Alive || b.Layer != "surface" || g.Known[b.P] || b.ID == g.Cur.Burg {
			continue
		}
		ps = append(ps, pl{b, abs(b.P%w.W-cx) + abs(b.P/w.W-cy)})
	}
	if len(ps) == 0 {
		g.lgKinMenu(a, "\"You already know every hearth I know.\"")
		return
	}
	best := ps[0]
	for _, p := range ps {
		if p.d < best.d || p.d == best.d && p.b.ID < best.b.ID {
			best = p
		}
	}
	b := best.b
	g.AddRep(a.Faction, -30, " for directions")
	g.Known[b.P] = true
	g.addSecret(fmt.Sprintf("%s, of %s, lies near %s.", b.Name, w.StateName(b.State), w.RegionName(b.P)), "town", b.P, w.Year)
	g.lgKinMenu(a, fmt.Sprintf("\"Follow the roads toward %s. They keep a hearth of %s.\"\n\n(%s is marked on your world map.)", w.RegionName(b.P), w.StateName(b.State), b.Name))
}

var lgRoleSkill = map[string]string{
	"guard": "longblade", "merchant": "persuasion", "smith": "tinkering", "apothecary": "lore", "priest": "lore", "scholar": "lore",
	"elder": "persuasion", "tailor": "persuasion", "villager": "endurance",
}
var lgSkillPool = []string{"longblade", "shortblade", "axe", "cudgel", "missile", "endurance", "acrobatics", "lore", "persuasion", "tinkering"}

// lgSkillFor: the skill this person can teach (their trade, or one their people favour).
func (g *Game) lgSkillFor(a *Actor) string {
	for _, sk := range strings.Split(a.Teach, ",") {
		if sk != "" {
			return sk
		}
	}
	if sk := lgRoleSkill[a.Role]; sk != "" {
		return sk
	}
	return lgSkillPool[int(HashStr(g.Seed+a.Faction)%uint64(len(lgSkillPool)))]
}

func (g *Game) lgLearnSkill(a *Actor, sk string, cost int) {
	if g.Rep[a.Faction]-cost < 250 {
		g.lgKinMenu(a, fmt.Sprintf("\"You must stand higher with us to ask this. (needs %d reputation)\"", 250+cost))
		return
	}
	g.AddRep(a.Faction, -cost, " for a lesson")
	g.Skills[sk]++
	a.Set("lg:skill")
	g.Turn += 240
	g.Msg(fmt.Sprintf("{{G|%s teaches you; %s rises to rank %d.}}", g.The(a), skillName(sk), g.Skills[sk]))
	g.lgKinMenu(a, fmt.Sprintf("Hours later, %s rises to rank %d.", skillName(sk), g.Skills[sk]))
}

// lgRecipeFor: a workbench recipe of their people you do not know yet.
func (g *Game) lgRecipeFor(a *Actor) *cRecipe {
	var opts []*cRecipe
	for _, rc := range g.craftRecipes() {
		if !rc.Basic && !g.craftSt().Known[rc.ID] && rc.Rank <= 3 {
			opts = append(opts, rc)
		}
	}
	if len(opts) == 0 {
		return nil
	}
	return opts[int(HashStr(g.Seed+a.Faction+"recipe")%uint64(len(opts)))]
}

func (g *Game) lgLearnRecipe(a *Actor, rc *cRecipe) {
	if g.Rep[a.Faction] < 800 {
		g.lgKinMenu(a, "\"Some knowledge is not given cheaply. (needs 800 reputation)\"")
		return
	}
	g.AddRep(a.Faction, -200, " for a craft")
	g.craftSt().Known[rc.ID] = true
	a.Set("lg:recipe")
	g.gainXP(20)
	g.Msg(fmt.Sprintf("{{G|You learn to make %s (workbench: T).}}", rc.Name))
	g.lgKinMenu(a, fmt.Sprintf("They show you how %s is made, step by step.", rc.Name))
}

var lgGiftPool = []string{"sprint", "heal", "darkvision", "sense", "regen", "carapace", "spikes", "swimming", "diving", "climbing", "glimmer", "forcebubble", "arcanebolt", "slumber", "forcepush"}

// lgPowerFor: a power their people are known for that you do not have.
func (g *Game) lgPowerFor(a *Actor) string {
	start := int(HashStr(g.Seed+a.Faction+"power") % uint64(len(lgGiftPool)))
	for i := range lgGiftPool {
		id := lgGiftPool[(start+i)%len(lgGiftPool)]
		if g.P.Powers[id] == 0 {
			return id
		}
	}
	return ""
}

func (g *Game) lgPowerName(id string) string {
	if n := g.Sk.PowerName(id); n != "" {
		return n
	}
	return id
}

func (g *Game) lgLearnPower(a *Actor, id string) {
	if g.Rep[a.Faction] < 900 {
		g.lgKinMenu(a, "\"Not yet.\"")
		return
	}
	g.AddRep(a.Faction, -300, " for a gift")
	if g.P.Powers == nil {
		g.P.Powers = map[string]int{}
	}
	g.P.Powers[id] = 1
	a.Set("lg:power")
	g.gainXP(40)
	g.deed("kinship", fmt.Sprintf("%s was taught %s by %s.", g.P.Name, g.lgPowerName(id), a.Name), 0, 0)
	g.Msg(fmt.Sprintf("{{W|%s gives you %s.}}", g.The(a), g.lgPowerName(id)))
	g.lgKinMenu(a, fmt.Sprintf("They lay a hand on your brow. %s wakes in you.", g.lgPowerName(id)))
}

func (g *Game) lgRecruit(a *Actor, need int) {
	if g.companionCount() >= maxCompanions {
		g.lgKinMenu(a, "\"You've enough company already.\"")
		return
	}
	if g.Rep[a.Faction] < 250+need && lgBand(g.Rep[a.Faction]) < 5 {
		g.lgKinMenu(a, fmt.Sprintf("\"Not yet.\" (needs %d reputation)", 250+need))
		return
	}
	g.AddRep(a.Faction, -need, " to bind a legend to your road")
	g.joinCompanion(a)
	g.Mode, g.Talk = "main", nil
}
