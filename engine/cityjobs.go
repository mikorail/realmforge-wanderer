package engine

import (
	"fmt"
	"strings"
)

// City jobs for coin (prefix cj). Foremen and shopkeepers offer a shift ("Sign on for a shift as
// a ..."): hours pass with hunger and thirst (tvPass), a mid-shift scene with lettered choices
// (screen mode cjw), a wage in local money scaled by town tier, rank and skill, tips, strains,
// promotions (apprentice, journeyman, master), perks and slow skill practice. One shift per job
// line per day. State g.ext("cj"), planet-local. Night life is in nightlife.go.

type cjState struct {
	Shifts   map[string]int  `json:"shifts,omitempty"`
	Rank     map[string]int  `json:"rank,omitempty"`
	Hired    map[string]bool `json:"hired,omitempty"`
	Last     map[string]int  `json:"last,omitempty"`  // line -> day+1 of last shift
	Practice map[string]int  `json:"prac,omitempty"`  // skill -> shifts of practice
	Perk     map[string]int  `json:"perk,omitempty"`  // perk key -> day+1
	Seen     map[string]bool `json:"seen,omitempty"`  // firsts
	Spawned  map[string]int  `json:"spawn,omitempty"` // zone key -> night id
	Fined    map[string]int  `json:"fined,omitempty"` // curfew: burg+night
	Earned   int             `json:"earned,omitempty"`

	run *cjRun
}

type cjRun struct {
	Line  *cjLine
	Boss  string
	Fac   string
	Scene *cjScene
	Phase int // 0 scene, 1 result
	Out   []string
	Pay   int // percent change so far
	Hurt  int
	Fx    string
}

func (g *Game) cj() *cjState {
	st := g.ext("cj", func() any { return &cjState{} }).(*cjState)
	if st.Shifts == nil {
		st.Shifts = map[string]int{}
		st.Rank = map[string]int{}
		st.Hired = map[string]bool{}
		st.Last = map[string]int{}
		st.Practice = map[string]int{}
		st.Perk = map[string]int{}
		st.Seen = map[string]bool{}
		st.Spawned = map[string]int{}
		st.Fined = map[string]int{}
	}
	return st
}

func init() {
	unLocalKeys = append(unLocalKeys, "cj")
	modeKeys["cjw"] = cjKeys
	modeDraws["cjw"] = cjDraw
	talkHooks = append(talkHooks, cjTalk, cjNightTalk)
	enterHooks = append(enterHooks, cjEnter)
	turnHooks = append(turnHooks, cjTurn)
}

func (g *Game) cjFirst(key, text string) {
	st := g.cj()
	if st.Seen[key] {
		return
	}
	st.Seen[key] = true
	g.deed("event", text, 0, 0)
}

// ---- which job a person offers ----

func cjFind(id string) *cjLine {
	for _, l := range cjLines {
		if l.ID == id {
			return l
		}
	}
	return nil
}

func cjLineFor(a *Actor) *cjLine {
	if a == nil || a.HP <= 0 || a.Has("cj") || a.AI == "follower" || a.Angry {
		return nil
	}
	t := strings.ToLower(a.Title)
	has := func(ws ...string) bool {
		for _, w := range ws {
			if strings.Contains(t, w) {
				return true
			}
		}
		return false
	}
	switch {
	case has("foreman", "miner"):
		return cjFind("mine")
	case a.Role == "innkeeper" || a.Role == "tavern" || a.Role == "lw_innkeeper" || has("innkeeper", "tapster", "tavern"):
		return cjFind("tavern")
	case a.Role == "ostler" || has("stablehand", "ostler"):
		return cjFind("stable")
	case a.Role == "ferry" || a.Role == "dockmaster" || has("dockhand", "sailor", "harbourmaster", "fishmonger"):
		return cjFind("dock")
	case a.Role == "smith":
		return cjFind("forge")
	case a.Role == "apothecary":
		return cjFind("apoth")
	case a.Role == "scholar" || a.Role == "bookseller" || has("scribe", "clerk"):
		return cjFind("clerk")
	case a.Role == "guard" || a.Role == "sergeant" || has("captain"):
		return cjFind("patrol")
	case a.Role == "fence" || a.Role == "bmdealer":
		return cjFind("back")
	case a.Role == "lw_farmer" || (a.Role == "villager" && has("farmer", "herder")):
		return cjFind("farm")
	case a.Role == "merchant":
		return cjFind("stall")
	}
	return nil
}

func (g *Game) cjTitle(l *cjLine) string {
	if m, ok := cjTitles[g.Genre]; ok {
		if t, ok := m[l.ID]; ok {
			return t
		}
	}
	return l.Title
}

func (g *Game) cjRank(l *cjLine) int { return clampi(g.cj().Rank[l.ID], 0, 2) }

func (g *Game) cjTier() int {
	if g.Cur == nil || g.W == nil {
		return 0
	}
	return g.W.cityTier[g.Cur.Burg]
}

func (g *Game) cjFac(a *Actor) string {
	if a != nil && strings.HasPrefix(a.Faction, "state:") {
		return a.Faction
	}
	return uwFac(g.uwStateHere())
}

func (g *Game) cjWhenOK(l *cjLine) (bool, string) {
	h := g.Hour()
	switch l.When {
	case "day":
		if h < 6 || h > 16 {
			return false, "Come back between six and four, when the day's work starts."
		}
	case "eve":
		if h < 16 && h >= 5 {
			return false, "Come back in the evening, when the trade picks up."
		}
	case "night":
		if h < 17 && h >= 3 {
			return false, "Not now. Come after dark."
		}
	}
	return true, ""
}

// ---- talk ----

func cjTalk(g *Game, a *Actor) []Option {
	if g == nil || a == nil || g.Cur == nil || g.Cur.Kind != "town" {
		return nil
	}
	l := cjLineFor(a)
	if l == nil {
		return nil
	}
	st := g.cj()
	rank := g.cjRank(l)
	title := g.cjTitle(l)
	var opts []Option
	text := fmt.Sprintf("Sign on for a shift as a %s {{K|(%dh, %s)}}", title, l.Hours, cjRankNames[rank])
	if l.Shady {
		text = fmt.Sprintf("Sign on for quiet work in the back room {{K|(%dh)}}", l.Hours)
	}
	opts = append(opts, Option{"{{w|" + text + "}}", func() { g.cjStart(a, l) }})
	if st.Hired[l.ID] {
		opts = append(opts, Option{fmt.Sprintf("Hand in your notice as a %s.", title), func() {
			st.Hired[l.ID] = false
			g.Talk = a
			g.talkMenu("\"Fair enough. The door is open if you want it back.\"")
		}})
		if rank >= 1 && l.Perk == "room" {
			opts = append(opts, Option{"Ask for the staff cot for the night.", func() { g.cjCot(a, l) }})
		}
		if rank >= 2 {
			opts = append(opts, Option{"Ask your old guild contact what is being said.", func() { g.cjContact(a, l) }})
		}
	}
	return opts
}

func (g *Game) cjCot(a *Actor, l *cjLine) {
	st := g.cj()
	g.Talk = a
	if st.Perk["cot"] == g.Day()+1 {
		g.talkMenu("\"You had the cot last night. Give another soul a turn.\"")
		return
	}
	if g.nearHostile() {
		g.Msg("You can't rest with enemies in sight.")
		return
	}
	h := g.Hour()
	if h >= 5 && h < 19 {
		g.talkMenu("\"It's day. The cot is for the night.\"")
		return
	}
	st.Perk["cot"] = g.Day() + 1
	mins := ((6 - h + 24) % 24) * 60
	if mins == 0 || mins > 600 {
		mins = 480
	}
	g.tvPass(mins)
	g.P.HP = mini(g.P.MaxHP, g.P.HP+g.P.MaxHP/4)
	g.Msg("{{G|You sleep on the staff cot and wake rested, with a heel of bread on the sill.}}")
	g.talkMenu("\"Morning. Ate your bread? Good.\"")
}

func (g *Game) cjContact(a *Actor, l *cjLine) {
	st := g.cj()
	g.Talk = a
	if st.Perk["contact:"+l.ID] == g.Day()+1 {
		g.talkMenu("\"Nothing new since you asked this morning.\"")
		return
	}
	st.Perk["contact:"+l.ID] = g.Day() + 1
	r := NewRng(fmt.Sprintf("%s:cj:contact:%s:%d", g.Seed, l.ID, g.Day()))
	txt := Pick(r, cjRumours)
	if site, p, ok := g.nearestSite(); ok && r.Chance(0.4) {
		g.addSecret(site, "dungeon", p, g.W.Year)
		txt = site
	} else {
		g.addSecret(txt, "rumour", -1, g.W.Year)
	}
	g.talkMenu("Your old mates in the trade lean in: " + txt)
}

// cjPriceMod: a journeyman or master of a shop's trade buys cheaper there (one line in price()).
func (g *Game) cjPriceMod(a *Actor) float64 {
	if g == nil || a == nil {
		return 1
	}
	l := cjLineFor(a)
	if l == nil || l.Perk != "shop" {
		return 1
	}
	st := g.cj()
	if !st.Hired[l.ID] && st.Shifts[l.ID] == 0 {
		return 1
	}
	return 1 - 0.05*float64(g.cjRank(l))
}

// ---- running a shift ----

func (g *Game) cjStart(a *Actor, l *cjLine) {
	st := g.cj()
	g.Talk = a
	say := func(s string) { g.talkMenu(s) }
	if ok, why := g.cjWhenOK(l); !ok {
		say("\"" + why + "\"")
		return
	}
	if st.Last[l.ID] == g.Day()+1 {
		say("\"You've worked today. One shift a day, no more.\"")
		return
	}
	if g.nearHostile() {
		g.Msg("You can't work with enemies in sight.")
		return
	}
	if g.uwWantedHere() >= 10 && !l.Shady {
		say("\"We don't take wanted folk on the books. Settle with the law first.\"")
		return
	}
	if !st.Hired[l.ID] {
		st.Hired[l.ID] = true
		g.Msg(fmt.Sprintf("{{G|You sign on as a %s %s.}}", cjRankNames[g.cjRank(l)], g.cjTitle(l)))
	}
	st.Last[l.ID] = g.Day() + 1
	r := NewRng(fmt.Sprintf("%s:cj:scene:%s:%d:%s", g.Seed, l.ID, g.Day(), g.Cur.Key))
	sc := &l.Scenes[r.Int(0, len(l.Scenes)-1)]
	g.tvPass(l.Hours * 30)
	run := &cjRun{Line: l, Boss: a.Name, Fac: g.cjFac(a), Scene: sc}
	st.run = run
	run.Out = Wrap(strings.ReplaceAll(l.Intro, "{title}", g.cjTitle(l)), 70)
	g.Popup, g.Opts = nil, nil
	g.Mode = "cjw"
	g.Sel = 0
}

// cjMod is the modifier for a mid-shift check kind.
func (g *Game) cjMod(kind string) int {
	p := g.P
	s := g.Skills
	switch kind {
	case "lift":
		return p.M(STR) + 2*s["endurance"]
	case "grit":
		return p.M(TOU) + 2*s["endurance"]
	case "count":
		return p.M(INT) + 2*s["lore"]
	case "brew":
		return maxi(p.M(INT), p.M(WIL)) + 2*s["lore"]
	case "tend":
		return maxi(p.M(WIL), p.M(AGI)) + 2*s["survival"]
	case "patrol":
		return g.checkMod("perception") + s["cudgel"] + s["shortblade"]
	case "street":
		return g.checkMod("streetwise")
	case "haggle":
		return g.checkMod("persuasion")
	case "mend":
		return g.checkMod("tinkering")
	}
	return 0
}

func cjKindName(k string) string {
	return map[string]string{"lift": "Lift", "grit": "Grit", "count": "Count", "brew": "Brew", "tend": "Tend", "patrol": "Watch", "street": "Streetwise", "haggle": "Haggle", "mend": "Mend"}[k]
}

func cjKeys(g *Game, k string) {
	st := g.cj()
	run := st.run
	if run == nil {
		g.Mode = "main"
		return
	}
	if run.Phase == 1 {
		if k == "Enter" || k == " " || k == "Escape" {
			g.Mode = "main"
			g.Talk = nil
			st.run = nil
		}
		return
	}
	n := len(run.Scene.Ch) + 1
	if i := letterIndex(k); i >= 0 && i < n {
		g.cjChoose(i)
		return
	}
	switch k {
	case "ArrowUp":
		g.Sel = maxi(0, g.Sel-1)
	case "ArrowDown":
		g.Sel = mini(n-1, g.Sel+1)
	case "Enter", " ":
		g.cjChoose(g.Sel)
	case "Escape":
		g.cjChoose(n - 1)
	}
}

func (g *Game) cjChoose(i int) {
	st := g.cj()
	run := st.run
	l := run.Line
	if i >= len(run.Scene.Ch) { // walk off
		g.tvPass(l.Hours * 5)
		st.Hired[l.ID] = false
		if st.Rank[l.ID] > 0 {
			st.Rank[l.ID]--
			if st.Rank[l.ID] == 0 {
				st.Shifts[l.ID] = 0
			} else {
				st.Shifts[l.ID] = cjRankAt[0]
			}
		}
		g.AddRep(run.Fac, -3, " for walking off the job")
		run.Out = []string{"You throw down your tools and leave. No pay today, and the word will get around."}
		run.Phase = 1
		g.Msg("{{R|You walk off the job.}}")
		return
	}
	c := run.Scene.Ch[i]
	win := true
	if c.Kind != "" {
		win = g.roll2d6Check(g.cjMod(c.Kind), c.DC)
	}
	var out []string
	if win {
		run.Pay += c.Pay
		out = append(out, "{{G|"+c.Win+"}}")
		if c.Kind != "" {
			g.gainXP(3 + c.DC/2)
		}
	} else {
		run.Pay += c.LosePay
		out = append(out, "{{R|"+c.Lose+"}}")
		if c.Hurt > 0 {
			g.P.HP = maxi(1, g.P.HP-c.Hurt)
			out = append(out, fmt.Sprintf("{{R|You lose %d HP.}}", c.Hurt))
		}
	}
	run.Fx = ""
	g.cjEffect(run, c, win, &out)
	g.cjPayoff(run, win, &out)
	run.Out = out
	run.Phase = 1
}

func (g *Game) cjEffect(run *cjRun, c cjChoice, win bool, out *[]string) {
	l := run.Line
	r := NewRng(fmt.Sprintf("%s:cj:fx:%s:%d:%d", g.Seed, l.ID, g.Day(), g.cj().Shifts[l.ID]))
	state := g.uwStateHere()
	switch c.Fx {
	case "catch":
		if win {
			g.AddRep(run.Fac, 4, " for catching a thief")
			g.crKarma(0, 1, "caught a thief fairly")
			g.cjFirst("catch", "Caught a pickpocket on a watch shift.")
		}
	case "let":
		g.crKarma(0, 1, "let a thief go")
		g.crKarma(1, -1, "looked away on duty")
		*out = append(*out, "{{K|It sits oddly with you, but a hungry thief eats today.}}")
	case "stop":
		if win {
			g.AddRep(run.Fac, 3, " for stopping a brawl")
			g.crKarma(2, 1, "waded into a brawl")
		}
	case "rumour":
		if win {
			txt := Pick(r, cjRumours)
			g.addSecret(txt, "rumour", -1, g.W.Year)
			*out = append(*out, "{{c|You overhear: "+txt+"}}")
		}
	case "patron":
		if win && r.Chance(0.6) {
			if site, p, ok := g.nearestSite(); ok {
				g.addSecret(site, "dungeon", p, g.W.Year)
				*out = append(*out, "{{Y|A well-dressed patron tips you a hint: "+site+"}}")
				g.cjFirst("patron", "A rich patron gave a tip while the wanderer was working a shift.")
			}
		}
	case "run":
		if !win && state > 0 {
			g.uwAddBounty(state, 8, " for hauling stolen goods")
		}
		g.crKarma(1, -1, "ran contraband for a fence")
	case "skim":
		g.crKarma(3, -2, "skimmed from a fence's till")
	}
}

// cjPayoff: wage, tips, rank, perks, skill practice, rare events.
func (g *Game) cjPayoff(run *cjRun, win bool, out *[]string) {
	st := g.cj()
	l := run.Line
	rank := g.cjRank(l)
	pay := l.Base[rank]
	pay = pay * (100 + 30*g.cjTier()) / 100
	pay = pay * (100 + 8*g.Skills[l.Skill]) / 100
	pay = pay * clampi(100+run.Pay, 20, 220) / 100
	pay = maxi(1, pay)
	s := g.cuEarn(pay)
	*out = append(*out, fmt.Sprintf("{{G|You are paid %s.}}", s))
	st.Earned += pay
	st.Shifts[l.ID]++
	n := st.Shifts[l.ID]
	g.gainXP(8 + 4*rank)
	if !l.Shady {
		g.AddRep(run.Fac, 2+rank, " for honest work")
	}
	g.cjFirst("shift", "Earned honest coin on a first shift.")
	r := NewRng(fmt.Sprintf("%s:cj:rare:%s:%d", g.Seed, l.ID, n))
	notice := win && rank < 2 && r.Chance(0.07)
	nr := 0
	switch {
	case n >= cjRankAt[1]:
		nr = 2
	case n >= cjRankAt[0]:
		nr = 1
	}
	if notice && nr <= rank {
		n = cjRankAt[rank]
		st.Shifts[l.ID] = n
		nr = rank + 1
		*out = append(*out, "{{Y|"+run.Boss+" has noticed your skill.}}")
	}
	if nr > rank {
		st.Rank[l.ID] = nr
		*out = append(*out, fmt.Sprintf("{{W|Promoted: you are now a %s %s!}}", cjRankNames[nr], g.cjTitle(l)))
		g.AddRep(run.Fac, 8, " for your promotion")
		g.cjFirst("rank"+fmt.Sprint(nr), fmt.Sprintf("Became a %s of a trade in a town.", cjRankNames[nr]))
		switch l.Perk {
		case "meal":
			*out = append(*out, "{{K|Perk: a free meal after every shift.}}")
		case "room":
			*out = append(*out, "{{K|Perk: you may ask for the staff cot at night.}}")
		case "shop":
			*out = append(*out, "{{K|Perk: a trade discount in shops of your craft.}}")
		}
		if nr == 2 {
			*out = append(*out, "{{K|Perk: a guild contact who shares what the trade hears.}}")
		}
	}
	if rank >= 1 && l.Perk == "meal" || rank >= 2 {
		f := g.MakeFood(r, 1)
		g.give(f)
		*out = append(*out, "{{K|The cook hands you a meal for the road.}}")
	}
	// slow skill practice: after N shifts, +1 rank up to 3
	sk := l.Skill
	st.Practice[sk]++
	if cur := g.Skills[sk]; cur < 3 && st.Practice[sk] >= cjSkillAt[cur] {
		g.Skills[sk]++
		*out = append(*out, fmt.Sprintf("{{G|Practice tells: %s rises to rank %d.}}", skillName(sk), g.Skills[sk]))
	}
	for _, o := range *out {
		g.Msg(o)
	}
}

func cjDraw(g *Game) {
	st := g.cj()
	run := st.run
	if run == nil {
		return
	}
	sc := g.sc
	x, y, w, _ := g.panel("A day's work")
	l := run.Line
	sc.TextB(x, y, fmt.Sprintf("{{M|%s}} {{K|(%s, %s)}}", titleCase(g.cjTitle(l)), cjRankNames[g.cjRank(l)], l.Where), CGray, CBgDark, x+w)
	y += 2
	if run.Phase == 1 {
		for _, s := range run.Out {
			for _, ln := range Wrap(s, w-2) {
				sc.TextB(x, y, ln, CGray, CBgDark, x+w)
				y++
			}
		}
		sc.TextB(x, y+1, "{{K|Enter to leave.}}", CGray, CBgDark, x+w)
		return
	}
	for _, ln := range Wrap(strings.Join(run.Out, " "), w-2) {
		sc.TextB(x, y, "{{K|"+ln+"}}", CGray, CBgDark, x+w)
		y++
	}
	y++
	for _, ln := range Wrap(run.Scene.Intro, w-2) {
		sc.TextB(x, y, ln, CGray, CBgDark, x+w)
		y++
	}
	y++
	g.Sel = clampi(g.Sel, 0, len(run.Scene.Ch))
	for i := 0; i <= len(run.Scene.Ch); i++ {
		label := "Walk off the job. {{K|(no pay, a mark against you)}}"
		if i < len(run.Scene.Ch) {
			c := run.Scene.Ch[i]
			label = c.Label
			if c.Kind != "" {
				pct := int(chanceFor(g.cjMod(c.Kind), c.DC)*100 + 0.5)
				label = fmt.Sprintf("{{c|[%s · %s %d%%]}} %s", cjKindName(c.Kind), diffName(c.DC), pct, c.Label)
			}
		}
		bg := CBgDark
		if i == g.Sel {
			bg = CBgSel
		}
		for j, ln := range Wrap(fmt.Sprintf("{{W|%c)}} %s", 'a'+i, label), w-4) {
			sc.Fill(x, y, w, 1, bg)
			if j > 0 {
				ln = "   " + ln
			}
			sc.TextB(x, y, ln, CGray, bg, x+w)
			y++
		}
	}
	sc.TextB(x, y+1, "{{K|Letter or Enter to choose. Esc walks off.}}", CGray, CBgDark, x+w)
}
