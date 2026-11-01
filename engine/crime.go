package engine

import (
	"fmt"
	"sort"
	"strings"
)

// Crime: laws and contraband per realm, guard searches at the gate, stealing (pickpocketing,
// market counters, locked chests), bounties per realm and jail. State lives in g.ext("uw")
// (underworld.go). Item.Extra["stolen"] = victim realm id + 1.

// ---- laws ----

type uwLaws struct {
	Occult  bool // occult / eldritch / cursed relics
	Heresy  bool // forbidden books
	Augment bool // unlicensed augments
	Drugs   bool // stimulant tonics
	Arms    bool // war-weapons and bows inside cities
	Mutant  bool // open mutation is a crime (fine only)
}

func (g *Game) uwPct(state int, cat string, pct int) bool {
	return int(HashStr(fmt.Sprintf("%s:uwlaw:%d:%s", g.Seed, state, cat))%100) < pct
}

// uwLawsFor derives a realm's statutes from its ideology, religion doctrine and the genre.
func (g *Game) uwLawsFor(state int) uwLaws {
	var l uwLaws
	st := g.W.stateByID[state]
	if st == nil {
		return uwLaws{Arms: g.uwPct(state, "arms", 20)}
	}
	p := func(cat string, pct int) bool { return g.uwPct(state, cat, pct) }
	switch st.Ideology {
	case "theocracy":
		l = uwLaws{Occult: true, Heresy: true, Drugs: p("drugs", 70), Mutant: p("mut", 60), Augment: p("aug", 60)}
	case "technocracy":
		l = uwLaws{Augment: true, Mutant: p("mut", 50), Arms: p("arms", 30), Occult: p("occ", 40)}
	case "communal":
		l = uwLaws{Arms: p("arms", 60), Drugs: p("drugs", 50), Augment: p("aug", 30)}
	case "oligarchy":
		l = uwLaws{Arms: p("arms", 50), Drugs: p("drugs", 30), Occult: p("occ", 30), Augment: p("aug", 25)}
	case "republic":
		l = uwLaws{Arms: p("arms", 20), Drugs: p("drugs", 20), Heresy: p("her", 10)}
	case "autocracy":
		l = uwLaws{Arms: p("arms", 55), Heresy: p("her", 50), Occult: p("occ", 50), Drugs: p("drugs", 40)}
	default: // monarchy and unnamed
		l = uwLaws{Occult: p("occ", 40), Arms: p("arms", 25), Drugs: p("drugs", 30), Heresy: p("her", 30), Mutant: p("mut", 25), Augment: p("aug", 25)}
	}
	if rel := g.W.relByID[st.Religion]; rel != nil {
		switch rel.Doctrine {
		case "ancestral", "sacred_grove":
			l.Augment = true
		case "esoteric", "liberation", "mutation_rites":
			l.Occult, l.Heresy = false, false
			if rel.Doctrine == "mutation_rites" {
				l.Mutant = false
			}
		case "militant":
			l.Arms = false
		case "caste", "compassion":
			l.Drugs = true
		}
		if rel.Theology == "monotheism" {
			l.Heresy = true
		}
	}
	switch g.Genre {
	case "cyberpunk":
		if st.Ideology == "technocracy" || st.Ideology == "oligarchy" {
			l.Augment = true
		}
	case "biopunk", "atompunk":
		if p("mutg", 60) {
			l.Mutant = true
		}
	case "solarpunk":
		l.Drugs = l.Drugs && p("sol", 40)
	}
	return l
}

func (l uwLaws) list(g *Game) []string {
	var out []string
	if l.Occult {
		out = append(out, "occult and cursed relics")
	}
	if l.Heresy {
		out = append(out, "forbidden books")
	}
	if l.Augment {
		out = append(out, "unlicensed "+g.augSkin().Noun+"s")
	}
	if l.Drugs {
		out = append(out, "stimulant "+strings.ToLower(g.Sk.Haste)+"s")
	}
	if l.Arms {
		out = append(out, "war-arms and bows within the walls")
	}
	if l.Mutant {
		out = append(out, "open mutation (fined)")
	}
	out = append(out, "poisons", "forged papers", "stolen goods")
	return out
}

func uwIsPoison(it *Item) bool  { return it != nil && it.Extra["uw:poison"] > 0 }
func uwIsForged(it *Item) bool  { return it != nil && it.Extra["uw:forged"] > 0 }
func uwIsStolen(it *Item) bool  { return it != nil && it.Extra["stolen"] > 0 }
func uwStolenFrom(it *Item) int { return it.Extra["stolen"] - 1 }

// uwIllegal: whether carrying the item is a crime in the given realm.
func (g *Game) uwIllegal(it *Item, state int) bool {
	if it == nil {
		return false
	}
	if uwIsStolen(it) && uwStolenFrom(it) == state {
		return true
	}
	if uwIsPoison(it) || uwIsForged(it) {
		return true
	}
	l := g.uwLawsFor(state)
	switch {
	case l.Occult && (it.Cat == "occult" || it.Cat == "eldritch" || it.Cat == "cursed"):
		return true
	case l.Heresy && it.Kind == "book" && (it.Extra["uw:forbidden"] > 0 || it.Cat == "occult" || it.Cat == "eldritch"):
		return true
	case l.Augment && it.Kind == "augment":
		return true
	case l.Drugs && it.Kind == "tonic" && (it.Effect == "haste" || it.Effect == "strength"):
		return true
	case l.Arms && (it.Kind == "ranged" || it.Kind == "weapon" && it.Tier >= 3):
		return true
	}
	return false
}

// uwContraband lists the player's illegal items (pack and worn) for a realm.
func (g *Game) uwContraband(state int) []*Item {
	var out []*Item
	for _, it := range g.P.Inv {
		if g.uwIllegal(it, state) {
			out = append(out, it)
		}
	}
	for _, bp := range g.P.Body {
		if bp.Item != nil && g.uwIllegal(bp.Item, state) {
			out = append(out, bp.Item)
		}
		if bp.Aug != nil && g.uwIllegal(bp.Aug, state) {
			out = append(out, bp.Aug)
		}
	}
	return out
}

func (g *Game) uwMutated() bool {
	return g.Genotype == "mutant" && g.uwHasMutations()
}

func (g *Game) uwHasMutations() bool {
	for id := range powerByID {
		if muPhys(id) && g.mutLevel(id) > 0 {
			return true
		}
	}
	return false
}

// ---- where the player stands ----

// uwStateHere: the realm that owns the current zone (0 = free folk).
func (g *Game) uwStateHere() int {
	if g.Cur == nil {
		return 0
	}
	if bd := g.W.burgByID[g.Cur.Burg]; bd != nil && g.Cur.Kind == "town" {
		return bd.State
	}
	if g.W.In(g.Cur.WX, g.Cur.WY) {
		return g.W.State[g.W.Idx(g.Cur.WX, g.Cur.WY)]
	}
	return 0
}

func uwFac(state int) string { return fmt.Sprintf("state:%d", state) }

func (g *Game) uwRealm(state int) string {
	if state <= 0 {
		return "the free folk"
	}
	return g.FactionName(uwFac(state))
}

// ---- bounty ----

// uwBounty: the player's outstanding bounty with a realm. Bounty hunters read this.
func (g *Game) uwBounty(state int) int { return g.uwSt().Bounty[state] }

func (g *Game) uwWantedHere() int { return g.uwBounty(g.uwStateHere()) }

// uwIsGuardOf: guards (and soldiers) who enforce a realm's law.
func uwIsGuardOf(a *Actor, state int) bool {
	return a.HP > 0 && (a.AI == "guard" || a.Role == "guard") && a.Faction == uwFac(state)
}

// uwWanted: whether this actor would attack the player for a standing bounty (optional hook in Hostile).
func (g *Game) uwWanted(a *Actor) bool {
	if a == nil || a.AI != "guard" {
		return false
	}
	var st int
	if _, err := fmt.Sscanf(a.Faction, "state:%d", &st); err != nil {
		return false
	}
	return g.uwBounty(st) >= 10
}

func (g *Game) uwAlertGuards(state int, zn *Zone) {
	if zn == nil {
		return
	}
	for _, a := range zn.Actors {
		if uwIsGuardOf(a, state) && !a.Angry {
			a.Angry = true
		}
	}
}

func (g *Game) uwAddBounty(state, n int, why string) {
	if n <= 0 {
		return
	}
	st := g.uwSt()
	if g.uwGuildMember() {
		n = n * 3 / 4
	}
	st.Bounty[state] = mini(st.Bounty[state]+n, 6000)
	g.Msg(fmt.Sprintf("{{R|Crime witnessed%s! Bounty with %s: %d %s.}}", why, g.uwRealm(state), st.Bounty[state], g.Sk.Currency))
	if state > 0 {
		g.Rep[uwFac(state)] -= mini(n/5+1, 90)
		g.compRep(uwFac(state), -mini(n/5+1, 90))
	}
	if st.Bounty[state] >= 10 {
		g.uwAlertGuards(state, g.Cur)
	}
}

// uwClearBounty pardons the player: guards stand down everywhere, standing recovers a little.
func (g *Game) uwClearBounty(state int) {
	st := g.uwSt()
	delete(st.Bounty, state)
	for _, zn := range g.Zones {
		for _, a := range zn.Actors {
			if uwIsGuardOf(a, state) {
				a.Angry = false
			}
		}
	}
	if state > 0 && g.Rep[uwFac(state)] < -150 {
		g.Rep[uwFac(state)] = -150
	}
}

func (g *Game) uwPayBounty(state int) bool {
	b := g.uwBounty(state)
	if b <= 0 {
		return true
	}
	if g.P.Gold < b {
		g.Msg("You can't cover the bounty.")
		return false
	}
	g.P.Gold -= b
	g.uwClearBounty(state)
	g.Msg(fmt.Sprintf("You pay %d %s. The record is struck.", b, g.Sk.Currency))
	return true
}

// uwGuildMember: member of any thieves' guild (cuts bounties by a quarter, helps stealing).
func (g *Game) uwGuildMember() bool {
	for _, o := range g.Orders() {
		if o.Kind == "thieves" && g.OrderMembership(o.ID) != nil {
			return true
		}
	}
	return false
}

// uwJail: serve time. Illegal and stolen goods are confiscated, some valuables go to the evidence locker.
func (g *Game) uwJail(state int) {
	st := g.uwSt()
	b := g.uwBounty(state)
	hours := clampi(8+b/25, 8, 24*10)
	var lost []string
	for _, it := range g.uwContraband(state) {
		if bp := g.P.Equipped(it); bp != nil {
			bp.Item = nil
		} else {
			g.removeInv(it)
		}
		lost = append(lost, it.Name)
	}
	for _, it := range append([]*Item{}, g.P.Inv...) {
		if uwIsStolen(it) {
			g.removeInv(it)
			lost = append(lost, it.Name)
		}
	}
	var spare []*Item
	for _, it := range g.P.Inv {
		if it.Quest == 0 && it.Kind != "backpack" && it.Kind != "key" && it.Value >= 20 {
			spare = append(spare, it)
		}
	}
	sort.Slice(spare, func(i, j int) bool { return spare[i].Value > spare[j].Value })
	for i := 0; i < len(spare) && i < 2; i++ {
		if g.R.Chance(0.5) {
			g.removeInv(spare[i])
			lost = append(lost, spare[i].Name)
		}
	}
	g.P.Gold -= g.P.Gold / 10
	g.Turn += hours * 60
	g.Hunger = mini(g.Hunger+hours*8, 2000)
	g.Thirst = mini(g.Thirst+hours*8, 1500)
	g.P.HP = mini(g.P.MaxHP, g.P.HP+hours)
	g.uwClearBounty(state)
	st.Jailed++
	g.Msg(fmt.Sprintf("{{Y|You serve %d hours in the cells of %s.}}", hours, g.uwRealm(state)))
	if len(lost) > 0 {
		g.Msg("{{R|Confiscated: " + strings.Join(lost, ", ") + ".}}")
	}
	g.recompute()
}

// uwAssault: optional hook for provoke(): striking a townsperson is a crime if anyone saw it.
func (g *Game) uwAssault(d *Actor) {
	if g.Cur == nil || g.Cur.Kind != "town" || !strings.HasPrefix(d.Faction, "state:") {
		return
	}
	var state int
	fmt.Sscanf(d.Faction, "state:%d", &state)
	if g.atWar(uwFac(g.HomeState), d.Faction) {
		return
	}
	g.uwAddBounty(state, 40+g.P.Level*3, " (assault)")
}

func uwKill(g *Game, d, src *Actor) {
	if src != g.P || g.Cur == nil || g.Cur.Kind != "town" || !strings.HasPrefix(d.Faction, "state:") || d.AI == "follower" {
		return
	}
	var state int
	fmt.Sscanf(d.Faction, "state:%d", &state)
	if g.atWar(uwFac(g.HomeState), d.Faction) {
		return
	}
	n := 150
	if d.AI == "guard" {
		n = 250
	}
	g.uwAddBounty(state, n, " (murder)")
}

// ---- guard searches ----

func (g *Game) uwSearchChance(state, n int) float64 {
	rep := g.Rep[uwFac(state)]
	if rep >= 600 {
		return 0.03
	}
	c := 0.4 + 0.08*float64(mini(n, 5)) - float64(clampi(rep, -500, 600))/1600
	if !g.daylight() {
		c -= 0.1
	}
	if g.uwGuildMember() {
		c -= 0.08
	}
	return fclamp(c, 0.05, 0.9)
}

func fclamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func uwFine(items []*Item) int {
	f := 15
	for _, it := range items {
		f += 8 + it.Value/4
		if uwIsForged(it) || uwIsPoison(it) {
			f += 40
		}
	}
	return f
}

func uwEnter(g *Game, zn *Zone) {
	st := g.uwSt()
	if zn.Kind != "town" {
		st.LastBurg = 0
		return
	}
	bd := g.W.burgByID[zn.Burg]
	if bd == nil || st.LastBurg == zn.Burg+1 {
		return
	}
	st.LastBurg = zn.Burg + 1
	if b := st.Bounty[bd.State]; b >= 10 {
		g.uwAlertGuards(bd.State, zn)
		g.Msg(fmt.Sprintf("{{R|The guards of %s know your face. Bounty: %d %s.}}", g.uwRealm(bd.State), b, g.Sk.Currency))
		return
	}
	if g.Turn < 5 {
		return
	}
	key := fmt.Sprintf("%d:%d", bd.ID, g.Turn/1440)
	if st.Searched[key] {
		return
	}
	items := g.uwContraband(bd.State)
	mutant := g.uwLawsFor(bd.State).Mutant && g.uwMutated()
	n := len(items)
	if mutant {
		n++
	}
	if n == 0 {
		return
	}
	st.Searched[key] = true
	if !g.R.Chance(g.uwSearchChance(bd.State, n)) {
		return
	}
	g.uwSearchMenu(bd.State, items, mutant)
}

func (g *Game) uwSearchMenu(state int, items []*Item, mutant bool) {
	fine := uwFine(items)
	if mutant {
		fine += 25 + g.P.Level*2
	}
	var names []string
	for _, it := range items {
		names = append(names, it.Display())
	}
	if mutant {
		names = append(names, "{{G|your mutations}}")
	}
	text := fmt.Sprintf("A watch-captain of %s steps into your path. \"Search. Open your pack.\"\n\nThey find: %s.\n\nFine: {{W|%d %s}}.", g.uwRealm(state), strings.Join(names, ", "), fine, g.Sk.Currency)
	opts := []Option{{"{{W|Submit.}} They confiscate the goods and take the fine (jail if you can't pay).", func() {
		g.uwSeize(state, items)
		if g.P.Gold >= fine {
			g.P.Gold -= fine
			g.Msg(fmt.Sprintf("You pay a fine of %d %s.", fine, g.Sk.Currency))
			g.uwClose()
			return
		}
		g.P.Gold = 0
		g.uwClose()
		g.Msg("{{R|You can't pay. They haul you in.}}")
		g.uwAddBounty(state, 20, "")
		g.uwJail(state)
	}}}
	bribe := fine * 2
	opts = append(opts, Option{fmt.Sprintf("{{y|Bribe the captain.}} {{K|(%d %s, haggle with Persuasion)}}", bribe, g.Sk.Currency), func() {
		cost := bribe
		if g.roll2d6Check(g.checkMod("persuasion"), 9) {
			cost = bribe * 2 / 3
		}
		if g.P.Gold < cost {
			g.Msg("You can't afford the bribe. The captain's smile fades.")
			g.uwSeize(state, items)
			g.uwClose()
			g.uwAddBounty(state, 20, " (bribery)")
			g.uwJail(state)
			return
		}
		g.P.Gold -= cost
		g.uwClose()
		g.Msg(fmt.Sprintf("The captain pockets %d %s and looks the other way.", cost, g.Sk.Currency))
	}})
	for _, it := range g.P.Inv {
		if uwIsForged(it) {
			it := it
			opts = append(opts, Option{"{{c|Show your forged papers.}}", func() {
				g.removeInv(it)
				g.uwClose()
				if g.roll2d6Check(g.checkMod("bluff"), 8) {
					g.Msg("{{G|The captain squints at the seal, then waves you on.}}")
				} else {
					g.Msg("{{R|The seal is a clumsy fake!}}")
					g.uwSeize(state, items)
					g.uwAddBounty(state, 60, " (forgery)")
					g.uwJail(state)
				}
			}})
			break
		}
	}
	opts = append(opts, Option{"{{R|Run for it.}}", func() {
		g.uwClose()
		g.uwAddBounty(state, 40+fine/2, " (resisting search)")
	}})
	g.uwOpenMenu("Gate search", text, opts, true)
}

func (g *Game) uwSeize(state int, items []*Item) {
	for _, it := range items {
		if bp := g.P.Equipped(it); bp != nil {
			bp.Item = nil
		} else if g.P.Inv != nil {
			removed := false
			for i, o := range g.P.Inv {
				if o == it {
					g.P.Inv = append(g.P.Inv[:i], g.P.Inv[i+1:]...)
					removed = true
					break
				}
			}
			if !removed {
				for _, b := range g.P.Body {
					if b.Aug == it {
						b.Aug = nil
					}
				}
			}
		}
	}
	if len(items) > 0 {
		g.Msg(fmt.Sprintf("{{R|%d item(s) confiscated.}}", len(items)))
	}
	g.recompute()
}

// ---- stealing ----

// uwStealDC: difficulty of lifting an item from a victim (higher is harder).
func (g *Game) uwStealDC(victim *Actor, it *Item, worn, counter bool) int {
	dc := 8 + victim.M(INT) + victim.Level/4
	if victim.AI == "guard" || victim.Role == "guard" {
		dc += 3
	}
	if counter {
		dc += 2
	}
	if it != nil {
		dc += mini(itemWeight(it)/4, 3) + mini(it.Value/80, 4)
	}
	if worn {
		dc += 3
	}
	dc += g.uwDarkness()
	return dc
}

// uwDarkness: night and unlit ground make you harder to notice.
func (g *Game) uwDarkness() int {
	d := 0
	if !g.daylight() {
		d -= 2
		i := g.P.Y*MW + g.P.X
		if g.Cur != nil && i >= 0 && i < len(g.Cur.Lit) && !g.Cur.Lit[i] {
			d -= 1
		}
	}
	return d
}

func (g *Game) uwStealMod() int {
	m := maxi(g.P.M(AGI), g.P.M(INT)-1) + 2*g.Skills["acrobatics"] + g.Skills["shortblade"]
	if g.uwGuildMember() {
		m++
	}
	return m
}

func (g *Game) uwStealChance(victim *Actor, it *Item, worn, counter bool) float64 {
	return chanceFor(g.uwStealMod(), g.uwStealDC(victim, it, worn, counter))
}

// uwWitnesses: townsfolk and guards who might see the crime (excluding the victim).
func (g *Game) uwWitnesses(victim *Actor) []*Actor {
	var out []*Actor
	for _, a := range g.Cur.Actors {
		if a == victim || a == g.P || a.HP <= 0 || !strings.HasPrefix(a.Faction, "state:") || a.AI == "follower" {
			continue
		}
		if abs(a.X-g.P.X) <= 9 && abs(a.Y-g.P.Y) <= 7 && g.visible(a.X, a.Y) {
			out = append(out, a)
		}
	}
	return out
}

// uwSeen: whether any bystander notices a successful theft.
func (g *Game) uwSeen(victim *Actor) bool {
	p := 0.22
	if !g.daylight() {
		p = 0.09
	}
	if g.uwDarkness() <= -3 {
		p /= 2
	}
	for _, w := range g.uwWitnesses(victim) {
		q := p
		if w.AI == "guard" {
			q *= 2
		}
		if g.R.Chance(q) {
			return true
		}
	}
	return false
}

func (g *Game) uwMarkStolen(it *Item, state int) {
	if it.Extra == nil {
		it.Extra = map[string]int{}
	}
	it.Extra["stolen"] = state + 1
}

// uwTake pulls one unit of an item out of an inventory (stacks split).
func (g *Game) uwTake(from *[]*Item, it *Item) *Item {
	if it.Stack() && it.Count > 1 {
		it.Count--
		cp := *it
		cp.Count = 1
		cp.ID = g.newID()
		if it.Extra != nil {
			cp.Extra = map[string]int{}
			for k, v := range it.Extra {
				cp.Extra[k] = v
			}
		}
		return &cp
	}
	for i, o := range *from {
		if o == it {
			*from = append((*from)[:i], (*from)[i+1:]...)
			break
		}
	}
	return it
}

type uwLoot struct {
	label  string
	it     *Item
	worn   *BodyPart
	gold   int
	chance float64
}

func (g *Game) uwLootOf(v *Actor, counter bool) []uwLoot {
	var out []uwLoot
	for _, it := range v.Inv {
		if it.Quest > 0 {
			continue
		}
		out = append(out, uwLoot{label: it.Display(), it: it, chance: g.uwStealChance(v, it, false, counter)})
	}
	if !counter {
		for _, bp := range v.Body {
			if bp.Item != nil && bp.Item.Quest == 0 {
				out = append(out, uwLoot{label: bp.Item.Display() + " {{K|(worn)}}", it: bp.Item, worn: bp, chance: g.uwStealChance(v, bp.Item, true, false)})
			}
		}
		if v.Gold >= 8 {
			n := maxi(4, v.Gold/4)
			out = append(out, uwLoot{label: fmt.Sprintf("{{W|their purse}} {{K|(about %d)}}", n), gold: n, chance: chanceFor(g.uwStealMod(), g.uwStealDC(v, nil, false, false)-1)})
		}
	}
	return out
}

// uwAttempt rolls a theft. Returns whether the item was taken.
func (g *Game) uwAttempt(v *Actor, l uwLoot, counter bool) bool {
	st := g.uwSt()
	state := v.State
	if state == 0 {
		fmt.Sscanf(v.Faction, "state:%d", &state)
	}
	mod := g.uwStealMod()
	dc := g.uwStealDC(v, l.it, l.worn != nil, counter)
	if l.it == nil {
		dc--
	}
	roll := g.R.Dice(2, 6) + mod
	g.spend(1000)
	if roll >= dc {
		var got *Item
		switch {
		case l.worn != nil:
			got = l.worn.Item
			l.worn.Item = nil
		case l.it != nil:
			got = g.uwTake(&v.Inv, l.it)
		default:
			n := mini(l.gold, v.Gold)
			v.Gold -= n
			g.P.Gold += n
			got = &Item{Kind: "gold", Value: n}
			g.Msg(fmt.Sprintf("{{G|You lift %d %s from %s.}}", n, g.Sk.Currency, g.the(v)))
		}
		if got.Kind != "gold" {
			g.uwMarkStolen(got, state)
			g.give(got)
			g.Msg("{{G|You lift " + got.Display() + " from " + g.the(v) + ".}}")
		}
		st.Stolen++
		g.gainXP(4 + mini(got.Value/10, 20))
		if g.uwSeen(v) {
			g.uwAddBounty(state, 25+got.Value/2, " (theft)")
		}
		return true
	}
	g.Msg("{{R|" + strings.ToUpper(g.the(v)[:1]) + g.the(v)[1:] + " catches your hand!}}")
	v.Angry = true
	if v.AI == "guard" {
		g.uwAlertGuards(state, g.Cur)
	}
	val := 10
	if l.it != nil {
		val = l.it.Value
	}
	g.uwAddBounty(state, 20+val/3, " (attempted theft)")
	return false
}

// ---- chests and counters ----

func (g *Game) uwChestKey(x, y int) string { return fmt.Sprintf("%s:%d,%d", g.Cur.Key, x, y) }

// uwChestLoot is deterministic per chest; every item is hot in the owner's realm.
func (g *Game) uwChestLoot(x, y int) ([]*Item, int) {
	r := NewRng(g.Seed + ":uw:chest:" + g.uwChestKey(x, y))
	tier := 0
	if bd := g.W.burgByID[g.Cur.Burg]; bd != nil {
		tier = g.W.cityTier[bd.ID]
	}
	var items []*Item
	for i := 0; i < r.Int(1, 3); i++ {
		items = append(items, g.Loot(r, clampi(tier+r.Int(0, 2), 0, 5)))
	}
	return items, r.Int(5, 30) * (tier + 1)
}

func (g *Game) uwPickChest(x, y int) {
	st := g.uwSt()
	k := g.uwChestKey(x, y)
	if st.Looted[k] {
		g.Msg("You've already emptied this chest.")
		return
	}
	if g.Skills["tinkering"] < 1 {
		g.Msg("The lock is beyond you. You need the Tinkering skill and a pick.")
		return
	}
	state := g.uwStateHere()
	tier := 0
	if bd := g.W.burgByID[g.Cur.Burg]; bd != nil {
		tier = g.W.cityTier[bd.ID]
	}
	mod := maxi(g.P.M(AGI), g.P.M(INT)) + 2*g.Skills["tinkering"] + g.uwGuildBonus()
	dc := 9 + tier + g.uwDarkness()
	roll := g.R.Dice(2, 6)
	g.spend(2000)
	if roll+mod < dc {
		g.Msg("{{R|The pick slips.}}")
		if roll <= 3 {
			g.Msg("{{R|The lock snaps with a crack that carries!}}")
			if g.uwSeen(nil) {
				g.uwAddBounty(state, 40, " (burglary)")
			}
		}
		return
	}
	st.Looted[k] = true
	items, gold := g.uwChestLoot(x, y)
	for _, it := range items {
		g.uwMarkStolen(it, state)
		g.give(it)
		g.Msg("{{G|You find " + it.Display() + ".}}")
	}
	g.P.Gold += gold
	g.Msg(fmt.Sprintf("{{G|You pocket %d %s.}}", gold, g.Sk.Currency))
	st.Stolen++
	g.gainXP(8 + tier*4)
	if g.uwSeen(nil) {
		g.uwAddBounty(state, 60+gold, " (burglary)")
	}
}

func (g *Game) uwGuildBonus() int {
	if g.uwGuildMember() {
		return 1
	}
	return 0
}

// uwKeeperFor: the shopkeeper nearest a stall or counter.
func (g *Game) uwKeeperFor(x, y int) *Actor {
	var best *Actor
	bd := 99
	for _, a := range g.Cur.Actors {
		if a.HP <= 0 || len(a.Inv) == 0 || a.AI == "follower" || a == g.P {
			continue
		}
		if a.Role != "merchant" && a.Role != "smith" && a.Role != "apothecary" && a.Role != "bookseller" && a.Role != "exotic" && a.Role != "tailor" {
			continue
		}
		if d := maxi(abs(a.X-x), abs(a.Y-y)); d < bd && d <= 6 {
			best, bd = a, d
		}
	}
	return best
}

// uwSeen with a nil victim is allowed: witnesses are anyone in sight.
func (g *Game) uwLockedAt(x, y int) bool { return g.Cur.At(x, y) == TCrate && g.Cur.Kind == "town" }

// ---- the steal menu ----

func (g *Game) uwStealKey() {
	if g.Cur == nil || g.Mode != "main" {
		return
	}
	var opts []Option
	for _, d := range dirs8 {
		x, y := g.P.X+d[0], g.P.Y+d[1]
		if a := g.Cur.ActorAt(x, y); a != nil && a.AI != "follower" && !g.Hostile(a) {
			a := a
			opts = append(opts, Option{"{{y|Pick the pockets of}} " + g.the(a), func() { g.uwActorMenu(a, false) }})
			continue
		}
		switch g.Cur.At(x, y) {
		case TStall:
			if k := g.uwKeeperFor(x, y); k != nil {
				k := k
				opts = append(opts, Option{"{{y|Lift something from the counter of}} " + g.the(k), func() { g.uwActorMenu(k, true) }})
			} else {
				x, y := x, y
				opts = append(opts, Option{"{{y|Take from the unattended stall.}}", func() { g.uwUnattended(x, y) }})
			}
		case TCrate:
			if g.Cur.Kind == "town" {
				x, y := x, y
				done := g.uwSt().Looted[g.uwChestKey(x, y)]
				label := "{{y|Pick the lock of the chest.}}"
				if done {
					label = "{{K|The chest (already emptied).}}"
				}
				opts = append(opts, Option{label, func() { g.uwClose(); g.uwPickChest(x, y) }})
			}
		}
	}
	if len(opts) == 0 {
		g.Msg("There is nothing to steal from next to you.")
		return
	}
	opts = append(opts, Option{"Never mind.", func() { g.uwClose() }})
	g.uwOpenMenu("Light fingers", "Mind the watch. At night, in unlit corners, you are harder to see.", opts, false)
}

func (g *Game) uwActorMenu(v *Actor, counter bool) {
	loot := g.uwLootOf(v, counter)
	sort.SliceStable(loot, func(i, j int) bool { return loot[i].chance > loot[j].chance })
	if len(loot) == 0 {
		g.uwOpenMenu("Light fingers", "Nothing worth taking.", []Option{{"Back.", func() { g.uwClose() }}}, false)
		return
	}
	var opts []Option
	for i, l := range loot {
		if i >= 18 {
			break
		}
		l := l
		opts = append(opts, Option{fmt.Sprintf("%s {{c|[%d%%]}}", l.label, int(l.chance*100+0.5)), func() {
			g.uwClose()
			g.uwAttempt(v, l, counter)
		}})
	}
	opts = append(opts, Option{"Back.", func() { g.uwClose() }})
	who := g.the(v)
	g.uwOpenMenu("Light fingers", "Choose what to lift from "+who+". Chance includes your Agility, Acrobatics and Shortblade against their awareness.", opts, false)
}

func (g *Game) uwUnattended(x, y int) {
	st := g.uwSt()
	k := g.uwChestKey(x, y)
	g.uwClose()
	if st.Looted[k] {
		g.Msg("The stall has nothing left worth taking.")
		return
	}
	st.Looted[k] = true
	r := NewRng(g.Seed + ":uw:stall:" + k)
	it := g.Loot(r, 1)
	state := g.uwStateHere()
	g.uwMarkStolen(it, state)
	g.give(it)
	g.spend(1000)
	g.Msg("{{G|You slip " + it.Display() + " off the stall.}}")
	st.Stolen++
	if g.uwSeen(nil) {
		g.uwAddBounty(state, 20+it.Value/2, " (theft)")
	}
}

// uwOwned marks loose items lying inside town buildings as someone's property.
func uwZoneOwned(g *Game, zn *Zone, _ *Rng) {
	if zn.Kind != "town" {
		return
	}
	for _, it := range zn.Items {
		if it.Kind == "gold" || it.Quest > 0 {
			continue
		}
		if zn.At(it.X, it.Y) == TFloor {
			if it.Extra == nil {
				it.Extra = map[string]int{}
			}
			it.Extra["uw:own"] = 1
		}
	}
}

// uwPickup: optional hook in take(): picking up property inside a house is theft.
func (g *Game) uwPickup(it *Item) {
	if it.Extra["uw:own"] == 0 {
		return
	}
	delete(it.Extra, "uw:own")
	if g.Cur == nil || g.Cur.Kind != "town" || g.homeHere() != nil {
		return
	}
	state := g.uwStateHere()
	g.uwMarkStolen(it, state)
	g.uwSt().Stolen++
	if g.uwSeen(nil) {
		g.uwAddBounty(state, 15+it.Value/2, " (looting a home)")
	}
}
