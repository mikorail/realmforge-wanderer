package engine

import (
	"fmt"
	"sort"
	"strings"
)

// camplife.go (prefix cf): when you rest at a fire or a night camp with recruited companions, they come
// alive. A short camp scene (mode "cfc") shows what each does (cook from carried ingredients and feed the
// party, repair and sharpen gear, tend wounds, mend clothes, dice, song, a story from their own life),
// a pair may argue or make up, and someone remarks on the day. You then pick up to two things to do
// (listen, share your own story, give a gift, sharpen together, sing) for small loyalty and mood gains.
// One scene per game day. Opened by the sleep flow (a turn hook sees a camp or tent sleep begin) or from
// the command palette ("Camp scene"). All text is template based; picks are hashes and a side rng
// (seed:cf:day:zoneKey), never g.R. State g.ext("cf") (global: keyed by actor id and day).

type cfState struct {
	Day   int         `json:"d,omitempty"` // game day + 1 of the last scene
	Mood  map[int]int `json:"m,omitempty"` // companion id -> mood 0..100 (50 neutral)
	Told  map[int]int `json:"t,omitempty"` // companion id -> day + 1 of the last story they told
	Scene int         `json:"n,omitempty"` // scenes held in all
	Slept int         `json:"-"`           // minutes of the current camp sleep
	Cur   *cfScene    `json:"-"`           // the open scene
}

type cfScene struct {
	Lines []string
	Acts  int
	Done  map[string]bool
	Foot  []string
}

func (g *Game) cfSt() *cfState {
	st := g.ext("cf", func() any { return &cfState{} }).(*cfState)
	if st.Mood == nil {
		st.Mood = map[int]int{}
	}
	if st.Told == nil {
		st.Told = map[int]int{}
	}
	return st
}

func init() {
	turnHooks = append(turnHooks, cfTurn)
	modeKeys["cfc"] = cfKeys
	modeDraws["cfc"] = func(g *Game) { g.cfDraw() }
	paEntries = append(paEntries, paEntry{Name: "Camp scene", Keywords: "camp fire companions rest story sing share meal evening", Cat: "Life", Label: "ability",
		Func: func(g *Game) { g.cfOpen(true) }, Cond: func(g *Game) bool { return g.cfPossible() == "" }})
}

func (g *Game) cfMood(a *Actor) int {
	if v, ok := g.cfSt().Mood[a.ID]; ok {
		return v
	}
	return 50
}

func (g *Game) cfAddMood(a *Actor, n int) {
	g.cfSt().Mood[a.ID] = clampi(g.cfMood(a)+n, 0, 100)
}

func cfMoodWord(m int) string {
	switch {
	case m >= 80:
		return "glowing"
	case m >= 62:
		return "content"
	case m >= 40:
		return "steady"
	case m >= 22:
		return "edgy"
	}
	return "sour"
}

// cfPossible: "" when a scene may open now, else the reason.
func (g *Game) cfPossible() string {
	if g.Cur == nil || g.P == nil || g.W == nil || g.Dead {
		return "No camp here."
	}
	if len(g.cbTalkers()) == 0 {
		return "No companions are with you."
	}
	if g.nearHostile() {
		return "You can't settle with enemies in sight."
	}
	fire := g.ijTileNear(2, func(x, y int) bool { return g.Cur.At(x, y) == TBrazier || g.Cur.Fire[y*MW+x] > 0 })
	camp := g.ijHasTent() && outdoors(g.Cur)
	if !fire && !camp && g.Cur.Kind != "town" {
		return "You need a fire or a camp."
	}
	if g.cfSt().Day == g.Day()+1 {
		return "The camp has had its evening already."
	}
	return ""
}

func (g *Game) cfOpen(manual bool) {
	if why := g.cfPossible(); why != "" {
		if manual {
			g.Msg(why)
		}
		return
	}
	st := g.cfSt()
	st.Day = g.Day() + 1
	st.Scene++
	st.Cur = g.cfBuild()
	g.Mode = "cfc"
}

// cfTurn: a camp or tent sleep that has run 20 minutes opens the scene (shown once the sleep ends).
func cfTurn(g *Game) {
	if g.P == nil || g.Cur == nil || g.Dead {
		return
	}
	st := g.cfSt()
	sl := g.ijSt().Sleep
	if sl != "camp" && sl != "tent" {
		st.Slept = 0
		return
	}
	st.Slept++
	if st.Slept == 20 {
		g.cfOpen(false)
	}
}

// ---- building the scene ----

var cfActs = []string{"cook", "repair", "tend", "sharpen", "mend", "dice", "song", "story"}

func cfIngredients(g *Game) []*Item {
	var out []*Item
	for _, it := range g.P.Inv {
		if it.Kind == "ingredient" && it.Effect != "" && it.Extra["cl:rot"] == 0 {
			out = append(out, it)
		}
	}
	return out
}

func (g *Game) cfBuild() *cfScene {
	r := NewRng(g.W.Seed + ":cf:" + fmt.Sprint(g.Day()) + ":" + g.Cur.Key)
	sc := &cfScene{Done: map[string]bool{}}
	comps := g.cbTalkers()
	if len(comps) > maxCompanions {
		comps = comps[:maxCompanions]
	}
	sc.Lines = append(sc.Lines, "{{K|"+g.cfSetting()+"}}", "")
	cooked := false
	used := map[string]bool{}
	hurt := g.P.HP*100 < g.P.MaxHP*70
	for _, a := range comps {
		for _, c := range comps {
			if c.HP*100 < c.MaxHP*70 {
				hurt = true
			}
		}
		var pool []string
		for _, k := range cfActs {
			switch {
			case k == "cook" && (cooked || len(cfIngredients(g)) == 0):
				continue
			case used[k]:
				continue
			}
			pool = append(pool, k)
			if k == "tend" && hurt {
				pool = append(pool, k, k)
			}
		}
		if len(pool) == 0 {
			pool = []string{"song"}
		}
		act := Pick(r, pool)
		used[act] = true
		if act == "cook" {
			cooked = true
		}
		sc.Lines = append(sc.Lines, g.cfAct(a, act, r)...)
		sc.Lines = append(sc.Lines, "")
	}
	if len(comps) >= 2 {
		sc.Lines = append(sc.Lines, g.cfPair(comps[0], comps[1], r)...)
		sc.Lines = append(sc.Lines, "")
	}
	if l := g.cfReact(comps[r.Int(0, len(comps)-1)], r); l != "" {
		sc.Lines = append(sc.Lines, l)
	}
	return sc
}

func (g *Game) cfSetting() string {
	where := "the fire"
	if g.Cur.Kind == "town" {
		where = "a public hearth"
	} else if !g.ijTileNear(2, func(x, y int) bool { return g.Cur.At(x, y) == TBrazier || g.Cur.Fire[y*MW+x] > 0 }) {
		where = "your camp"
	}
	return fmt.Sprintf("The party settles at %s in %s.", where, nonEmpty(g.Cur.Name, "the wild"))
}

func (g *Game) cfName(a *Actor) string { return "{{c|" + givenOf(a.Name) + "}}" }

// cfSay: a spoken line in the companion's voice (culture pack or banks), "" when none.
func (g *Game) cfVoice(a *Actor, kind string) string {
	s := g.dvSay(a, kind, nil)
	if s == "" {
		return ""
	}
	return fmt.Sprintf("%s says: \"%s\"", givenOf(a.Name), s)
}

func (g *Game) cfAct(a *Actor, act string, r *Rng) []string {
	n := g.cfName(a)
	switch act {
	case "cook":
		ings := cfIngredients(g)
		sort.SliceStable(ings, func(i, j int) bool { return ings[i].ID < ings[j].ID })
		take := mini(2+boolInt(g.Skills["survival"] >= 2), len(ings))
		pickd := ings[:take]
		tags := map[string]int{}
		var list []string
		for _, it := range pickd {
			tags[it.Effect] = mini(2, tags[it.Effect]+1)
			list = append(list, it.Effect)
		}
		sort.Strings(list)
		key := strings.Join(list, "+")
		name := ""
		for i := range g.Cookbook {
			if g.Cookbook[i].Key == key {
				name = g.Cookbook[i].Name
				g.Cookbook[i].Cooked++
			}
		}
		if name == "" {
			name = g.dishName(key, pickd)
			where := ""
			if cu := g.localCulture(); cu != nil {
				where = cu.Name
			}
			g.Cookbook = append(g.Cookbook, Dish{Key: key, Name: name, Tags: list, Where: where, Cooked: 1, FirstAt: g.Turn / 1440})
		}
		for _, it := range pickd {
			g.useUp(it)
		}
		g.applyMeal(name, tags, 1440+360*g.Skills["survival"], true)
		g.cfAddMood(a, 6)
		return []string{n + " hangs a pot over the coals and cooks " + name + " from your stores. Everyone eats.", "  " + g.cfBlurb(a, r)}
	case "repair":
		g.ijSt().Fat = maxf(0, g.ijSt().Fat-4)
		g.cfAddMood(a, 3)
		return []string{n + " spreads a cloth and repairs a worn strap and a loose buckle on your kit, then oils the rest. " + g.cfBlurb(a, r)}
	case "tend":
		heal := func(x *Actor) {
			x.HP = mini(x.MaxHP, x.HP+maxi(2, x.MaxHP/8))
		}
		heal(g.P)
		for _, c := range g.localCompanions() {
			heal(c)
		}
		g.cfAddMood(a, 3)
		return []string{n + " boils water, washes and binds everyone's hurts, and mutters about people who never ask for help. The party feels better."}
	case "sharpen":
		g.cfAddMood(a, 2)
		return []string{n + " works a whetstone along a blade with long, even strokes until it sings.", "  " + g.cfBlurb(a, r)}
	case "mend":
		g.cfAddMood(a, 2)
		return []string{n + " threads a needle and mends a torn cloak by the firelight, tongue between the teeth."}
	case "dice":
		g.cfAddMood(a, 4)
		coin := g.Sk.Currency
		return []string{fmt.Sprintf("%s shakes out a set of bone dice and plays against the dark, staking a single %s that never changes hands.", n, coin)}
	case "song":
		g.cfAddMood(a, 5)
		return []string{n + " hums an old tune, then sings a verse low enough not to wake the night.", "  " + g.cfBlurb(a, r)}
	}
	return g.cfStory(a, r, "")
}

// cfBlurb: a line spoken in the culture's own voice, or a plain fallback.
func (g *Game) cfBlurb(a *Actor, r *Rng) string {
	for _, k := range []string{"praise", "thanks", "agree"} {
		if r.Chance(0.4) {
			continue
		}
		if s := g.cfVoice(a, k); s != "" {
			return s
		}
	}
	return ""
}

func (g *Game) cfStory(a *Actor, r *Rng, lead string) []string {
	st := g.cfSt()
	c := a.Comp
	st.Told[a.ID] = g.Day() + 1
	g.cfAddMood(a, 3)
	var opts []string
	town := ""
	if c != nil {
		town = c.Town
	}
	if town != "" {
		opts = append(opts,
			"tells how the bakers of {town} rise in the dark, and how the whole street smells of it by dawn.",
			"tells of a flood year in {town}, and the night everyone slept on the roofs and sang.",
			"remembers a festival in {town} where a goat got into the procession and no one dared laugh.")
	}
	if c != nil && len(c.Kin) > 0 {
		opts = append(opts,
			"talks about {kin}, and the things they used to say that sounded like nonsense until later.",
			"tells of the winter {kin} taught them to read the weather from a cat's whiskers.")
	}
	if c != nil && strings.HasPrefix(c.Faction, "state:") {
		opts = append(opts, "tells of the tax collector of {realm} who once got lost for three days and was never rude again.")
	}
	if a.Religion > 0 {
		opts = append(opts, "tells a story of {faith}: a stranger who asked for water, and the household that learned what they had turned away.")
	}
	if a.Species != "" {
		opts = append(opts, "tells how their kind first crossed a river that was meant to stop them, as the elders tell it.")
	}
	opts = append(opts,
		"tells of the first time they were truly afraid, and the odd small thing that got them through it.",
		"tells a short story about a road, a bargain and a mule with strong views.")
	s := strings.NewReplacer("{town}", nonEmpty(town, "home")).Replace(Pick(r, opts))
	s = g.cbFill(a, s, nil)
	out := []string{g.cfName(a) + " " + s}
	if lead != "" {
		out[0] = lead + out[0]
	}
	if b := g.cfBlurb(a, r); b != "" {
		out = append(out, "  "+b)
	}
	return out
}

func (g *Game) cfPair(a, b *Actor, r *Rng) []string {
	ca, cb := a.Comp, b.Comp
	na, nb := g.cfName(a), g.cfName(b)
	kin := ca != nil && cb != nil && (ca.HomeKey == cb.HomeKey && ca.HomeKey != "" || a.House != "" && a.House == b.House)
	rival := ca != nil && cb != nil && strings.HasPrefix(ca.Faction, "state:") && strings.HasPrefix(cb.Faction, "state:") && g.atWar(ca.Faction, cb.Faction)
	argue := r.Int(0, 3) == 0 || rival && r.Chance(0.5) || g.cfMood(a)+g.cfMood(b) < 80 && r.Chance(0.5)
	if argue {
		g.cfAddMood(a, -4)
		g.cfAddMood(b, -4)
		switch {
		case rival:
			return []string{na + " and " + nb + " go at each other about whose realm started the war, until the fire pops and they both stop to glare at it."}
		case kin:
			return []string{na + " and " + nb + " bicker the way only people from one street can, over a thing that happened years ago and was not even important."}
		}
		return []string{na + " and " + nb + " fall out over who takes the next watch. It ends with two folded arms and a long silence."}
	}
	g.cfAddMood(a, 3)
	g.cfAddMood(b, 3)
	switch {
	case rival:
		return []string{na + " and " + nb + " trade a grudging compliment across the fire, the first in a while. Neither mentions the war."}
	case kin:
		return []string{na + " and " + nb + " fall into the old shorthand of people who grew up close, and laugh at something nobody else would get."}
	}
	return []string{na + " passes " + nb + " the best seat by the fire without a word, and " + nb + " nods back. That settles it."}
}

func (g *Game) cfReact(a *Actor, r *Rng) string {
	n := g.cfName(a)
	st := g.cb()
	switch {
	case g.P.HP*100 < g.P.MaxHP*50:
		return n + " eyes your wounds and says you should be more careful. They say it kindly."
	case g.Hunger > 1200:
		return n + " pats their stomach and wonders aloud if anyone remembers what a full meal is."
	case st.Wx != "" && r.Chance(0.6):
		return n + " glances at the sky and has something to say about today's " + strings.ToLower(st.Wx) + "."
	}
	return Pick(r, []string{
		n + " looks over the day's road and says it went better than the last one.",
		n + " leans back and says the day had a good shape to it, whatever that means.",
		n + " stares into the embers and says they will remember today, for one reason or another."})
}

// ---- the choices ----

func (sc *cfScene) feet(g *Game) []string {
	var out []string
	add := func(key, lbl string) {
		mark := "{{W|[" + key + "]}}"
		if sc.Done[key] {
			mark = "{{K|[" + key + "]}}"
		}
		out = append(out, mark+" "+lbl)
	}
	add("1", "Sit and listen")
	add("2", "Share your own story")
	add("3", "Give a gift (food)")
	add("4", "Sharpen your weapon together")
	add("5", "Sing")
	return out
}

func (g *Game) cfLow(comps []*Actor) *Actor {
	var best *Actor
	for _, a := range comps {
		if best == nil || g.cfMood(a) < g.cfMood(best) {
			best = a
		}
	}
	return best
}

func cfKeys(g *Game, k string) {
	st := g.cfSt()
	sc := st.Cur
	if sc == nil || k == "Escape" || k == "q" || k == "Enter" {
		st.Cur = nil
		g.Mode = "main"
		return
	}
	if len(k) != 1 || k[0] < '1' || k[0] > '5' {
		return
	}
	if sc.Done[k] {
		return
	}
	if sc.Acts >= 2 {
		g.Msg("The evening is winding down.")
		return
	}
	comps := g.cbTalkers()
	if len(comps) == 0 {
		st.Cur = nil
		g.Mode = "main"
		return
	}
	if g.nearHostile() {
		g.Msg("Not with enemies near.")
		return
	}
	r := NewRng(g.W.Seed + ":cf:c:" + fmt.Sprint(g.Day(), ":", k, ":", sc.Acts))
	var out []string
	switch k {
	case "1":
		a := g.cfLow(comps)
		for _, c := range comps {
			if st.Told[c.ID] != g.Day()+1 {
				a = c
				break
			}
		}
		out = append(out, "{{G|You sit and listen.}}")
		g.cfAddMood(a, 6)
		g.addLoyalty(a, 3)
		out = append(out, g.cfStory(a, r, "")...)
	case "2":
		opts := []string{
			"You tell of the strangest place you have slept, and the shorter story of how you left it.",
			"You tell of a road that went nowhere, and of the thing you found at the end of it.",
			"You tell how you learned to be careful, which is a longer story than it should be."}
		out = append(out, "{{G|"+Pick(r, opts)+"}}")
		for _, c := range comps {
			g.cfAddMood(c, 4)
			g.addLoyalty(c, 1)
		}
		out = append(out, "They listen closely, and someone laughs at the right moment.")
	case "3":
		var food *Item
		for _, it := range g.P.Inv {
			if it.Kind == "food" {
				food = it
				break
			}
		}
		if food == nil {
			g.Msg("You have no food to give.")
			return
		}
		a := g.cfLow(comps)
		nm := food.Name
		g.consume(food)
		g.cfAddMood(a, 8)
		g.addLoyalty(a, 4)
		out = append(out, fmt.Sprintf("{{G|You give %s the %s.}} They did not expect it.", givenOf(a.Name), nm))
		if s := g.cfVoice(a, "thanks"); s != "" {
			out = append(out, s)
		}
	case "4":
		if len(g.P.Weapons()) == 0 {
			g.Msg("You have no weapon to sharpen.")
			return
		}
		g.ijSt().Fat = maxf(0, g.ijSt().Fat-8)
		for _, c := range comps {
			g.cfAddMood(c, 3)
			g.addLoyalty(c, 1)
		}
		out = append(out, "{{G|You sit with the whetstone beside your companions.}} For a while the only sound is steel and breathing. Somehow you all rest a little.")
	case "5":
		opts := []string{"You start a verse and they come in on the chorus, a little off and entirely sincere.", "Someone starts a tune; you add a second line; the dark pulls back an arm's length."}
		out = append(out, "{{G|"+Pick(r, opts)+"}}")
		for _, c := range comps {
			g.cfAddMood(c, 5)
			g.addLoyalty(c, 1)
		}
	}
	sc.Done[k] = true
	sc.Acts++
	sc.Lines = append(sc.Lines, "")
	sc.Lines = append(sc.Lines, out...)
	g.spendTurns(12)
}

// ---- drawing ----

func (g *Game) cfDraw() {
	s := g.sc
	st := g.cfSt()
	x, y, w, h := g.panel("Camp")
	sc := st.Cur
	if sc == nil {
		s.TextB(x, y, "{{K|The fire has burned down. Press Esc.}}", CGray, CBgDark, x+w)
		return
	}
	feet := sc.feet(g)
	budget := h - len(feet) - 6
	var lines []string
	for _, l := range sc.Lines {
		if l == "" {
			lines = append(lines, "")
			continue
		}
		lines = append(lines, Wrap(l, w)...)
	}
	if len(lines) > budget {
		lines = lines[len(lines)-budget:]
	}
	for _, l := range lines {
		s.TextB(x, y, l, CGray, CBgDark, x+w)
		y++
	}
	y = maxi(y+1, h-len(feet)-3)
	moods := []string{}
	for _, a := range g.cbTalkers() {
		moods = append(moods, fmt.Sprintf("%s %s", givenOf(a.Name), cfMoodWord(g.cfMood(a))))
	}
	s.TextB(x, y, "{{K|Mood: "+strings.Join(moods, ", ")+"}}", CGray, CBgDark, x+w)
	y++
	for _, f := range feet {
		s.TextB(x, y, f, CGray, CBgDark, x+w)
		y++
	}
	s.TextB(x, y, fmt.Sprintf("{{K|[Esc] break camp   (%d of 2 choices used)}}", sc.Acts), CGray, CBgDark, x+w)
}
