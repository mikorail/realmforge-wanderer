package engine

import (
	"fmt"
	"sort"
	"strings"
)

// Biopunk signature: the breeding vat (prefix gd). Samples (tissue, blood, venom, seed, spore) come from
// kills, forage and flesh-grafters. A vat (device "vat", screen mode "gdv") combines 1-2 samples and a
// nutrient over game hours into a crop, an organ graft, a hatchling companion or a weapon organ. The whole
// outcome (mutation, quality, duration) is fixed by a side-rng hash when you seed the vat; it ripens by
// the clock, so it keeps growing while you are away. Harmful mutations hatch hostile abominations.

type gdSmp struct {
	Type string `json:"t"`
	Src  string `json:"s"`
	Name string `json:"n"`
	Q    int    `json:"q"`
}

type gdVat struct {
	S     []gdSmp `json:"s,omitempty"`
	Nut   string  `json:"nu,omitempty"`
	Tag   string  `json:"tg,omitempty"`
	Goal  string  `json:"g,omitempty"`
	Start int     `json:"st,omitempty"`
	Done  int     `json:"dn,omitempty"`
	Out   int     `json:"o,omitempty"`
	Q     int     `json:"q,omitempty"`
	Seed  string  `json:"sd,omitempty"`
	Told  bool    `json:"td,omitempty"`
}

type gdVatSel struct {
	s1, s2, nut *Item
	goal        string
}

var gdSampleTypes = map[string]struct {
	Name string
	F    byte
}{"tissue": {"tissue", CRed}, "blood": {"blood", CDarkRed}, "venom": {"venom", CGreen}, "spore": {"spore", CMagenta}, "seed": {"seed", CYellow}}

func init() {
	zoneHooks = append(zoneHooks, gdBioZone)
	killHooks = append(killHooks, gdBioKill)
	talkHooks = append(talkHooks, gdBioTalk)
	turnHooks = append(turnHooks, gdBioTurn)
	enterHooks = append(enterHooks, gdBioEnter)
	sideHooks = append(sideHooks, gdBioSide)
	gdDevOpen["vat"] = gdVatOpen
}

func (g *Game) gdMakeSample(typ, src, name string, q int) *Item {
	d := gdSampleTypes[typ]
	q = clampi(q, 1, 5)
	return &Item{ID: g.newID(), Kind: "gdsample", Effect: typ + ":" + src, Name: fmt.Sprintf("%s sample (%s)", d.Name, name), Glyph: '¤', F: d.F,
		Value: 10 + 8*q, Weight: 1, Extra: map[string]int{"gdq": q}, Lore: fmt.Sprintf("A cold-sealed %s sample of %s, quality %d. Feed it to a breeding vat.", d.Name, name, q)}
}

func gdSmpOf(it *Item) gdSmp {
	typ, src, _ := strings.Cut(it.Effect, ":")
	n := it.Name
	if i := strings.Index(n, "("); i >= 0 {
		n = strings.TrimSuffix(n[i+1:], ")")
	}
	return gdSmp{Type: typ, Src: src, Name: n, Q: maxi(1, it.Extra["gdq"])}
}

func (g *Game) gdSamples() []*Item {
	var out []*Item
	for _, it := range g.P.Inv {
		if it.Kind == "gdsample" {
			out = append(out, it)
		}
	}
	return out
}

func (g *Game) gdNutrients() []*Item {
	var out []*Item
	for _, it := range g.P.Inv {
		if it.Kind == "food" || it.Kind == "ingredient" || it.Kind == "dish" {
			out = append(out, it)
		}
	}
	return out
}

// ---- sources of samples ----

func gdBioKill(g *Game, d, src *Actor) {
	if g.Genre != "biopunk" || d == nil || g.P == nil || g.Cur == nil {
		return
	}
	if src != g.P && (src == nil || src.AI != "follower") {
		return
	}
	if d.Has("gdabom") {
		g.give(g.gdMakeSample("tissue", "tpl:ooze", "salvaged abomination", 3))
		g.Msg("{{G|You cut a salvage sample from the abomination.}}")
		return
	}
	if d.Role != "" || d.Proper || d.Species != "" || d.SimPer > 0 || d.Has("undead") || d.Tpl == "npc" || d.Tpl == "golem" {
		return
	}
	r := g.gdRng("smp", d.ID)
	p := 0.30 + 0.07*float64(g.Skills["survival"]) + 0.03*float64(g.Skills["lore"])
	if !r.Chance(p) {
		return
	}
	srcID, name, q := "tpl:"+d.Tpl, strings.ToLower(d.Name), 1+d.Level/4
	if sp := bsByTpl[d.Tpl]; sp != nil {
		srcID, name, q = "bs:"+sp.ID, strings.ToLower(sp.Display()), q+sp.Rarity/2
	}
	typ := "tissue"
	switch {
	case d.Tpl == "sporefolk" || d.Tpl == "shambler" || d.Tpl == "ooze" || d.Tpl == "mushroom":
		typ = "spore"
	case d.Has("poison"):
		typ = "venom"
		if r.Chance(0.4) {
			typ = "tissue"
		}
	case r.Chance(0.4):
		typ = "blood"
	}
	it := g.gdMakeSample(typ, srcID, name, q)
	g.give(it)
	g.Msg("{{G|You take a " + gdSampleTypes[typ].Name + " sample from the " + name + ".}}")
}

func gdBioForage(g *Game, zn *Zone, x, y int) {
	t := zn.At(x, y)
	if t != TMushroom && t != TTree && t != TTrunk {
		return
	}
	st := g.gd()
	key := fmt.Sprintf("%s:%d", zn.Key, y*MW+x)
	if d, ok := st.Cut[key]; ok && g.Day()-d < 3 {
		return
	}
	st.Cut[key] = g.Day()
	r := g.gdRng("forage", key, g.Day())
	if !r.Chance(0.5 + 0.1*float64(g.Skills["survival"])) {
		return
	}
	typ, nm := "spore", "giant mushroom"
	if t != TMushroom {
		typ, nm = "seed", "tree"
	}
	g.give(g.gdMakeSample(typ, "plant:"+nm, nm, 1+zn.Level/3+r.Int(0, 1)))
	g.Msg("{{G|You scrape a " + typ + " sample from the " + nm + ".}}")
}

func gdBioTalk(g *Game, a *Actor) []Option {
	if g.Genre != "biopunk" || (a.Role != "augmenter" && a.Role != "apothecary") || g.Cur == nil {
		return nil
	}
	return []Option{{"{{O|Buy a vat sample.}}", func() { gdBioShop(g, a) }}}
}

func gdBioShop(g *Game, a *Actor) {
	g.Popup = Wrap("\"Fresh from the cold room. Sealed, labelled, and not stolen.\"", 100)
	g.Opts = nil
	r := g.gdRng("shop", a.ID, g.Day())
	for i := 0; i < 4; i++ {
		typ := []string{"tissue", "blood", "venom", "spore", "seed"}[r.Int(0, 4)]
		q := r.Int(1, 3)
		var src, name string
		if r.Chance(0.5) {
			sp := bsAll[r.Int(0, len(bsAll)-1)]
			src, name = "bs:"+sp.ID, strings.ToLower(sp.Display())
		} else {
			tp := []string{"rat", "snake", "boar", "wolf", "spider", "beetle"}[r.Int(0, 5)]
			src, name = "tpl:"+tp, tp
		}
		if typ == "seed" || typ == "spore" {
			name = []string{"glowcap", "bladereed", "marrowvine", "weeping fig"}[r.Int(0, 3)]
			src = "plant:" + name
		}
		price := 20 + 15*q
		ty, sr, nm, qq := typ, src, name, q
		g.Opts = append(g.Opts, Option{fmt.Sprintf("%s sample (%s), quality %d {{K|(%s)}}", titleCase(typ), name, q, g.gdPrice(price)), func() {
			if g.gdPay(price) {
				g.give(g.gdMakeSample(ty, sr, nm, qq))
				g.talkMenu("\"Keep it cold.\"")
			}
		}})
	}
	g.Opts = append(g.Opts, Option{"Enough.", func() { g.talkMenu(g.greeting(a)) }})
	g.Sel = 0
}

// ---- placement ----

func gdBioZone(g *Game, zn *Zone, _ *Rng) {
	if g.Genre != "biopunk" || !gdTown(g, zn) {
		return
	}
	r := NewRng(g.Seed + ":gd:vat:" + zn.Key)
	p := 0.35
	if zn.Kind == "outskirts" {
		p = 0.15
	}
	if !r.Chance(p) {
		return
	}
	if x, y, ok := g.gdFreeSpot(zn, r, nil); ok {
		g.gdPlaceDev(zn, "vat", x, y, 1+r.Int(0, 2))
	}
}

// ---- the vat screen ----

func gdVatName(v *gdVat, now int) string {
	switch {
	case v.Goal == "":
		return "breeding vat"
	case now >= v.Done:
		return "breeding vat (ready)"
	}
	return "breeding vat (growing)"
}

func gdVatOpen(g *Game, d *Item) {
	st := g.gd()
	key := gdDevKey(g.Cur, d)
	v := st.Vats[key]
	if v == nil {
		v = &gdVat{}
		st.Vats[key] = v
	}
	st.vsel = gdVatSel{}
	g.gdOpen("gdv", "Breeding vat", func(u *gdUI) { gdVatBuild(g, u, d, v) })
}

func gdVatGoals(ss []*Item) []string {
	var seed, flesh, venom bool
	for _, it := range ss {
		switch gdSmpOf(it).Type {
		case "seed", "spore":
			seed = true
		case "tissue", "blood":
			flesh = true
		case "venom":
			venom = true
		}
	}
	var out []string
	if seed {
		out = append(out, "crop")
	}
	if flesh {
		out = append(out, "graft", "pet")
	}
	if flesh || venom {
		out = append(out, "weapon")
	}
	return out
}

var gdGoalName = map[string]string{"crop": "a crop", "graft": "an organ graft", "pet": "a hatchling companion", "weapon": "a weapon organ"}

type gdPlan struct {
	Out, Q, Mins int
	Risk, Good   float64
	Seed         string
}

func (g *Game) gdVatPlan(ss []*Item, nut *Item, goal string, grade int, tag string) gdPlan {
	sm := g.gdSkill()
	var srcs []string
	qs := 0
	for _, it := range ss {
		srcs = append(srcs, it.Effect)
		qs += gdSmpOf(it).Q
	}
	sort.Strings(srcs)
	seed := fmt.Sprintf("%s:gd:vat:%s:%s:%s:%d", g.Seed, strings.Join(srcs, "+"), nut.Name, goal, g.Turn)
	r := NewRng(seed)
	hybrid := len(ss) == 2 && ss[0].Effect != ss[1].Effect
	avg := qs / len(ss)
	bad := 0.30 - 0.03*float64(sm) - 0.05*float64(grade-1) - 0.03*float64(avg-1)
	good := 0.15 + 0.025*float64(sm) + 0.05*float64(grade-1)
	if hybrid {
		bad += 0.08
		good += 0.10
	}
	bad, good = clampF(bad, 0.05, 0.45), clampF(good, 0.10, 0.55)
	roll := r.Float()
	out := 0
	if roll < bad {
		out = -1
	} else if roll < bad+good {
		out = 1
	}
	hrs := map[string]int{"crop": 12, "graft": 36, "pet": 60, "weapon": 30}[goal]
	hrs += r.Int(0, hrs/2)
	hrs = hrs * (100 - 8*(grade-1) - 3*sm) / 100
	q := clampi(1+sm/3+(grade-1)+avg/3+boolI(out > 0)-boolI(out < 0 && false), 1, 5)
	return gdPlan{Out: out, Q: q, Mins: maxi(hrs, 2) * 60, Risk: bad, Good: good, Seed: seed}
}

func clampF(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func boolI(b bool) int {
	if b {
		return 1
	}
	return 0
}

func gdPick(list []*Item, cur *Item, skip *Item) *Item {
	var c []*Item
	for _, it := range list {
		if it != skip {
			c = append(c, it)
		}
	}
	if len(c) == 0 {
		return nil
	}
	for i, it := range c {
		if it == cur {
			if i+1 < len(c) {
				return c[i+1]
			}
			return nil
		}
	}
	return c[0]
}

func gdVatBuild(g *Game, u *gdUI, d *Item, v *gdVat) {
	st := g.gd()
	sel := &st.vsel
	grade := maxi(1, d.Extra["gdq"])
	u.line(fmt.Sprintf("{{C|Vat grade %d.}} Quality improves with Tinkering, Survival and lore; mutations are a gamble.", grade))
	d.Name = gdVatName(v, g.Turn)
	if v.Goal != "" {
		var nm []string
		for _, s := range v.S {
			nm = append(nm, s.Name+" "+s.Type)
		}
		u.line(fmt.Sprintf("Culture: %s, grown toward %s in %s broth.", strings.Join(nm, " + "), gdGoalName[v.Goal], v.Nut))
		if g.Turn >= v.Done {
			u.line("{{G|The culture is ripe.}} Left too long it overgrows.")
			u.row("{{G|Harvest the vat.}}", func() { g.gdVatHarvest(d, v) })
		} else {
			u.line(fmt.Sprintf("{{Y|%s}} %d%%, ripe in %s.", gdBar(g.Turn-v.Start, v.Done-v.Start, 20), 100*(g.Turn-v.Start)/maxi(1, v.Done-v.Start), gdHours(v.Done-g.Turn)))
		}
		u.row("{{r|Drain the vat and lose it all.}}", func() {
			*v = gdVat{}
			g.Msg("You drain the culture into the sump.")
		})
		return
	}
	var live []*Item
	for _, it := range []*Item{sel.s1, sel.s2, sel.nut} {
		if it != nil {
			found := false
			for _, p := range g.P.Inv {
				found = found || p == it
			}
			if !found {
				*sel = gdVatSel{}
				break
			}
		}
	}
	samples := g.gdSamples()
	name := func(it *Item) string {
		if it == nil {
			return "{{K|none}}"
		}
		return it.Display()
	}
	u.row("First sample: "+name(sel.s1), func() { sel.s1 = gdPick(samples, sel.s1, sel.s2) })
	u.row("Second sample (optional): "+name(sel.s2), func() { sel.s2 = gdPick(samples, sel.s2, sel.s1) })
	u.row("Nutrient: "+name(sel.nut), func() { sel.nut = gdPick(g.gdNutrients(), sel.nut, nil) })
	if sel.s1 != nil {
		live = append(live, sel.s1)
	}
	if sel.s2 != nil {
		live = append(live, sel.s2)
	}
	goals := gdVatGoals(live)
	ok := false
	for _, gl := range goals {
		ok = ok || gl == sel.goal
	}
	if !ok {
		sel.goal = ""
		if len(goals) > 0 {
			sel.goal = goals[0]
		}
	}
	gn := "{{K|(choose a sample)}}"
	if sel.goal != "" {
		gn = gdGoalName[sel.goal]
	}
	u.row("Grow: "+gn, func() {
		for i, gl := range goals {
			if gl == sel.goal {
				sel.goal = goals[(i+1)%len(goals)]
				return
			}
		}
	})
	if sel.s1 == nil || sel.nut == nil || sel.goal == "" {
		u.line("{{K|Pick a sample, a nutrient and what to grow.}}")
		return
	}
	tag := sel.nut.Effect
	pl := g.gdVatPlan(live, sel.nut, sel.goal, grade, tag)
	u.line(fmt.Sprintf("Forecast: about %s. Mutation: {{R|%d%% harmful}}, {{G|%d%% beneficial}}.", gdHours(pl.Mins), int(pl.Risk*100), int(pl.Good*100)))
	u.row("{{W|Seed the vat.}}", func() {
		pl := g.gdVatPlan(live, sel.nut, sel.goal, grade, tag)
		v.S = nil
		for _, it := range live {
			v.S = append(v.S, gdSmpOf(it))
		}
		v.Nut, v.Tag, v.Goal = sel.nut.Name, tag, sel.goal
		v.Start, v.Done, v.Out, v.Q, v.Seed, v.Told = g.Turn, g.Turn+pl.Mins, pl.Out, pl.Q, pl.Seed, false
		for _, it := range live {
			g.removeInv(it)
		}
		g.gdTakeOne(sel.nut)
		*sel = gdVatSel{}
		g.Msg("{{G|You seal the samples into the vat. The broth begins to stir.}}")
		g.spendTurns(5)
	})
}

var gdCropNames = map[string]string{"hearty": "meat-bulb", "spry": "sugar-berry", "thick": "marrow-gourd", "mending": "poultice-moss", "keen": "lumen-fungus",
	"ember": "pepper-bloom", "rime": "frost-lichen", "deep": "kelp-bladder", "lasting": "dry rind", "venom": "barbed pod"}

func (g *Game) gdVatHarvest(d *Item, v *gdVat) {
	r := NewRng(v.Seed + ":harvest")
	late := (g.Turn - v.Done) / DayLen
	q, out := v.Q-late, v.Out
	if late >= 4 && out >= 0 && r.Chance(0.5) {
		out = -1
	}
	q = clampi(q, 1, 5)
	goal := v.Goal
	smp := v.S
	nm := v.Nut
	tag := v.Tag
	*v = gdVat{}
	d.Name = gdVatName(v, g.Turn)
	if out < 0 {
		g.gdAbomination(d, goal, q, r)
		return
	}
	mut := ""
	if out > 0 {
		mut = []string{"hardy", "bright", "swift-growing", "dense", "resonant"}[r.Int(0, 4)]
	} else if r.Chance(0.5) {
		mut = []string{"pale", "odd-scented", "twitchy", "speckled"}[r.Int(0, 3)]
	}
	switch goal {
	case "crop":
		if _, ok := cookTags[tag]; !ok {
			tag = cookTagIDs[HashStr(nm)%uint64(len(cookTagIDs))]
		}
		n := 3 + q + r.Int(0, 2) + 2*boolI(out > 0)
		cn := gdCropNames[tag]
		if mut != "" {
			cn = mut + " " + cn
		}
		g.give(&Item{ID: g.newID(), Kind: "ingredient", Name: "vat-grown " + cn, Effect: tag, Glyph: '%', F: CGreen, Count: n, Value: 6 + 2*q, Power: 120 + 40*q})
		g.give(&Item{ID: g.newID(), Kind: "food", Name: "vat-grown loaf", Glyph: '%', F: CBrown, Count: n/2 + 1, Value: 4, Power: 350 + 50*q})
		g.Msg(fmt.Sprintf("{{G|You harvest %d vat-grown %s and a few loaves.}}", n, cn))
		g.gdFirst("vat_crop", "grew a first crop in a breeding vat.")
	case "graft":
		var pool []string
		for _, ad := range AugDefs {
			if !ad.Rare || (q >= 4 && out > 0) {
				pool = append(pool, ad.ID)
			}
		}
		it := g.MakeAugment(Pick(r, pool))
		it.Value = it.Value * (4 + q) / 5
		it.Lore = "Grown in a breeding vat from " + gdSmpNames(smp) + ". " + it.Lore
		if out > 0 {
			if it.Bonus == nil {
				it.Bonus = map[int]int{}
			}
			it.Bonus[r.Int(0, 5)]++
			it.Name += " (mutant)"
			it.Rarity, it.F = "rare", CBlue
		}
		g.give(it)
		g.Msg("{{G|The vat opens: a living " + it.Name + ", still warm.}}")
		g.gdFirst("vat_graft", "grew a first organ graft in a breeding vat.")
	case "pet":
		g.gdHatch(d, smp, q, out, mut, r, v)
	case "weapon":
		sk := []string{"longblade", "shortblade", "axe", "cudgel"}[r.Int(0, 3)]
		it := g.MakeWeapon(sk, clampi(1+q/2, 1, 4), r)
		it.Name = map[string]string{"longblade": "grown bone-scythe", "shortblade": "barbed stinger", "axe": "chitin cleaver", "cudgel": "knotted gristle-club"}[sk]
		for _, s := range smp {
			if s.Type == "venom" {
				it.Elem = "poison"
				it.Name = "venomous " + it.Name
			}
		}
		if out > 0 {
			it.DN++
			it.PV++
			it.Name = "mutant " + it.Name
			it.F = CMagenta
		}
		it.Lore = "A weapon organ grown from " + gdSmpNames(smp) + ". It twitches in your grip."
		it.Value += 15 * q
		g.give(it)
		g.Msg("{{G|You pull a " + it.Name + " from the vat, slick and warm.}}")
		g.gdFirst("vat_weapon", "grew a first weapon organ in a breeding vat.")
	}
	g.gainXP(10 * q)
}

func gdSmpNames(ss []gdSmp) string {
	var n []string
	for _, s := range ss {
		n = append(n, s.Name)
	}
	return strings.Join(n, " and ")
}

func (g *Game) gdAbomination(d *Item, goal string, q int, r *Rng) {
	tpl := map[string]string{"crop": "shambler", "graft": "crawler", "pet": "spider", "weapon": "beetle"}[goal]
	a := g.Spawn(tpl, maxi(2, g.Cur.Level+1+q/2), r)
	a.Name, a.Proper, a.F = "vat abomination", false, CMagenta
	a.AI, a.Angry = "hostile", true
	a.Set("gdabom")
	a.X, a.Y = d.X, d.Y
	g.nudge(g.Cur, a)
	g.Cur.Actors = append(g.Cur.Actors, a)
	g.Msg("{{R|The glass bursts. Something that was never meant to live drags itself out of the vat!}}")
	g.gdFirst("vat_abom", "hatched a first abomination from a failed culture.")
}

func (g *Game) gdHatch(d *Item, smp []gdSmp, q, out int, mut string, r *Rng, v *gdVat) {
	if !g.tmSlotFree() {
		g.Msg("{{y|A hatchling stirs in the vat, but you already lead as many companions as you can keep. Make room and come back.}}")
		v.S, v.Goal, v.Done, v.Out, v.Q, v.Seed = smp, "pet", g.Turn-1, out, q, "gdlate"+fmt.Sprint(r.Int(0, 99999))
		v.Nut, v.Tag = "late", ""
		return
	}
	var sp *bsSp
	var tplID string
	for _, s := range smp {
		if s.Type != "tissue" && s.Type != "blood" {
			continue
		}
		kind, id, _ := strings.Cut(s.Src, ":")
		if kind == "bs" && bsByID[id] != nil {
			sp = bsByID[id]
			break
		}
		if kind == "tpl" && tplByID[id] != nil {
			tplID = id
			break
		}
	}
	lvl := maxi(1, 1+q/2)
	var a *Actor
	switch {
	case sp != nil:
		a = g.bsSpawn(sp, lvl, r)
	case tplID != "":
		a = g.Spawn(tplID, lvl, r)
	default:
		a = g.Spawn("rat", lvl, r)
	}
	a.X, a.Y = g.P.X, g.P.Y
	g.nudge(g.Cur, a)
	traits := []string{"swift", "tough", "venomous", "armored", "keen-eyed", "brawny"}
	tr := ""
	if out > 0 || r.Chance(0.4) {
		tr = Pick(r, traits)
		switch tr {
		case "swift":
			a.Speed += 20
		case "tough":
			a.MaxHP = a.MaxHP * 3 / 2
		case "armored":
			a.AVNat += 3
		case "keen-eyed":
			a.DVNat += 3
		case "brawny":
			a.Attr[STR] += 4
		case "venomous":
			for _, p := range a.Body {
				if p.Natural != nil {
					p.Natural.Elem = "poison"
				}
			}
		}
	}
	if len(smp) == 2 && smp[0].Src != smp[1].Src {
		a.MaxHP += 6
		a.Attr[TOU] += 2
		tr = strings.TrimSpace(tr + " chimera")
	}
	a.HP = a.MaxHP
	g.Cur.Actors = append(g.Cur.Actors, a)
	if sp != nil {
		g.tmJoin(a, sp, 55+5*g.Skills["survival"], g.Cur.Key)
	} else {
		a.Comp = &Companion{Faction: a.Faction, AI: "wander", F: a.F, HomeKey: g.Cur.Key, HomeX: a.X, HomeY: a.Y, Loyalty: 55, Since: g.Turn}
		a.Order, a.AI, a.Faction, a.Angry, a.F = OrdFollow, "follower", "player", false, CGreen
		a.Set("tm")
		a.Name, a.Proper, a.Title = g.tmPetName(a), true, strings.ToLower(a.Tpl)
		g.tm().Fed[a.ID] = g.Turn
	}
	a.Title = strings.TrimSpace("vat-bred " + tr + " " + strings.TrimPrefix(a.Title, "tamed "))
	g.Msg(fmt.Sprintf("{{G|%s the %s hatches, wet and blinking, and follows you.}}", a.Name, a.Title))
	g.gdFirst("vat_pet", "hatched a first companion in a breeding vat.")
}

// ---- time ----

func gdBioTurn(g *Game) {
	if g.Genre != "biopunk" || g.Turn%10 != 0 || g.Cur == nil {
		return
	}
	st := g.gd()
	for _, d := range g.gdDevs(g.Cur, "vat") {
		v := st.Vats[gdDevKey(g.Cur, d)]
		if v == nil || v.Goal == "" {
			continue
		}
		d.Name = gdVatName(v, g.Turn)
		if g.Turn >= v.Done && !v.Told {
			v.Told = true
			g.Msg("{{G|A vat nearby chimes: the culture is ripe.}}")
		}
	}
}

func gdBioEnter(g *Game, zn *Zone) {
	if g.Genre != "biopunk" {
		return
	}
	st := g.gd()
	for _, d := range g.gdDevs(zn, "vat") {
		v := st.Vats[gdDevKey(zn, d)]
		if v == nil || v.Goal == "" {
			continue
		}
		d.Name = gdVatName(v, g.Turn)
		if g.Turn >= v.Done {
			v.Told = true
			g.Msg(fmt.Sprintf("{{G|While you were away your culture finished: it has been ripe for %s.}}", gdHours(g.Turn-v.Done)))
		} else {
			g.Msg(fmt.Sprintf("{{Y|Your vat has grown %d%% in your absence.}}", 100*(g.Turn-v.Start)/maxi(1, v.Done-v.Start)))
		}
	}
}

func gdBioSide(g *Game) []string {
	if g.Genre != "biopunk" || g.Cur == nil {
		return nil
	}
	st := g.gd()
	for _, d := range g.gdDevs(g.Cur, "vat") {
		if v := st.Vats[gdDevKey(g.Cur, d)]; v != nil && v.Goal != "" && g.Turn >= v.Done {
			return []string{"{{G|Vat: ripe}}"}
		}
	}
	return nil
}
