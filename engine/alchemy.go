package engine

import (
	"fmt"
	"sort"
	"strings"
)

// Herbalism and alchemy (prefix al). Every ingredient (herb, fungus, fruit, beast material) has 2-3 hidden
// properties, a pure function of its id and the world seed (side rng, never g.R). You learn them by tasting,
// from an herbalist, from books or by brewing. Properties shared by 2+ ingredients become the potion's
// effects. State in g.ext("al"), planet-local ("al" in unLocalKeys). Screen F4 / ability al_brew.

type alState struct {
	Known     map[string][]string `json:"k,omitempty"` // ingredient key -> discovered property ids
	Recipes   map[string]bool     `json:"r,omitempty"` // named recipes known
	Disc      map[string]bool     `json:"d,omitempty"` // experimental combos (sorted property ids joined by +)
	Firsts    map[string]bool     `json:"f,omitempty"`
	BooksDone map[string]bool     `json:"b,omitempty"`
	Rank      int                 `json:"rk,omitempty"` // herbalism 0-5
	XP        int                 `json:"xp,omitempty"`
	Made      int                 `json:"mk,omitempty"`

	stage, sel   int
	method, base int
	picks        []*Item
	npc          bool
	note         string
	loreOff      int
}

func (g *Game) al() *alState {
	st := g.ext("al", func() any { return &alState{} }).(*alState)
	if st.Known == nil {
		st.Known = map[string][]string{}
	}
	if st.Recipes == nil {
		st.Recipes = map[string]bool{}
	}
	if st.Disc == nil {
		st.Disc = map[string]bool{}
	}
	if st.Firsts == nil {
		st.Firsts = map[string]bool{}
	}
	if st.BooksDone == nil {
		st.BooksDone = map[string]bool{}
	}
	return st
}

func init() {
	unLocalKeys = append(unLocalKeys, "al")
	modeKeys["al"] = alKeys
	modeDraws["al"] = alDraw
	mainKeys["F4"] = func(g *Game) { g.alOpen(false) }
	zoneHooks = append(zoneHooks, alZone)
	talkHooks = append(talkHooks, alTalk)
	turnHooks = append(turnHooks, alTurn)
}

// ---- ingredients and properties ---------------------------------------------------

type alPW struct {
	ID string
	W  int
}

// alIng identifies a brewing ingredient: its stable key and cooking/beast tag.
func alIng(it *Item) (key, tag string, ok bool) {
	if it == nil {
		return
	}
	switch it.Kind {
	case "ingredient":
		if it.Effect == "" {
			return
		}
		k := strings.TrimPrefix(strings.ToLower(it.Name), "dried ")
		return k, it.Effect, true
	case "trophy":
		mat, sp, _, t := bcInfo(it)
		if mat == nil || sp == nil {
			return
		}
		return "bc:" + mat.ID + ":" + sp.ID, t, true
	}
	return
}

func (g *Game) alPropsOf(key, tag string) []alPW {
	r := NewRng(g.Seed + ":al:p:" + key)
	n := 2 + r.Int(0, 1)
	var out []alPW
	have := map[string]bool{}
	if p, ok := alTagProp[tag]; ok {
		out = append(out, alPW{p, r.Int(2, 3)})
		have[p] = true
	}
	for len(out) < n {
		p := alProps[r.Int(0, len(alProps)-1)].ID
		if have[p] {
			continue
		}
		have[p] = true
		out = append(out, alPW{p, r.Int(1, 3)})
	}
	return out
}

func alQual(it *Item) int {
	if it.Kind == "trophy" {
		if _, _, q, _ := bcInfo(it); q > 0 {
			return q
		}
	}
	return clampi(1+it.Value/8, 1, 4)
}

func (st *alState) knows(key, p string) bool {
	for _, k := range st.Known[key] {
		if k == p {
			return true
		}
	}
	return false
}

func (st *alState) learn(key, p string) bool {
	if st.knows(key, p) {
		return false
	}
	st.Known[key] = append(st.Known[key], p)
	return true
}

func (g *Game) alIngredients() []*Item {
	var out []*Item
	for _, it := range g.P.Inv {
		if _, _, ok := alIng(it); ok {
			out = append(out, it)
		}
	}
	return out
}

func alPropLabel(id string) string {
	d := alPropByID[id]
	if d == nil {
		return id
	}
	return "{{" + d.Col + "|" + d.Name + "}}"
}

func (g *Game) alPropLine(it *Item) string {
	key, tag, ok := alIng(it)
	if !ok {
		return ""
	}
	st := g.al()
	var parts []string
	for _, pw := range g.alPropsOf(key, tag) {
		if st.knows(key, pw.ID) {
			parts = append(parts, alPropLabel(pw.ID))
		} else {
			parts = append(parts, "{{K|?}}")
		}
	}
	return strings.Join(parts, " ")
}

// ---- brewing core ---------------------------------------------------------------

type alRes struct {
	Eff  map[string]int
	Leak []string
	Side int
}

// alCompute finds the shared properties of the ingredients; onlyKnown ignores undiscovered ones (previews).
func alCompute(g *Game, items []*Item, onlyKnown bool) alRes {
	st := g.al()
	cnt, sum := map[string]int{}, map[string]int{}
	for _, it := range items {
		key, tag, ok := alIng(it)
		if !ok {
			continue
		}
		q := alQual(it)
		for _, pw := range g.alPropsOf(key, tag) {
			if onlyKnown && !st.knows(key, pw.ID) {
				continue
			}
			cnt[pw.ID]++
			sum[pw.ID] += q * pw.W
		}
	}
	res := alRes{Eff: map[string]int{}}
	for p, c := range cnt {
		if c >= 2 {
			res.Eff[p] = clampi(1+sum[p]/(c+2), 1, 8)
		} else if alPropByID[p].Bad {
			res.Leak = append(res.Leak, p)
		}
	}
	for _, c := range alConflicts {
		if res.Eff[c[0]] > 0 && res.Eff[c[1]] > 0 {
			delete(res.Eff, c[0])
			delete(res.Eff, c[1])
			res.Side++
		}
	}
	sort.Strings(res.Leak)
	return res
}

func alSortedEff(eff map[string]int) []string {
	var ids []string
	for p := range eff {
		ids = append(ids, p)
	}
	sort.Slice(ids, func(i, j int) bool {
		if eff[ids[i]] != eff[ids[j]] {
			return eff[ids[i]] > eff[ids[j]]
		}
		return ids[i] < ids[j]
	})
	return ids
}

func (g *Game) alTool(n int) bool {
	if g.al().npc {
		return true
	}
	for _, it := range g.P.Inv {
		if it.Extra["altool"] == n {
			return true
		}
	}
	return false
}

func (g *Game) alMod() int {
	best := -99
	for _, k := range []string{"survival", "tinkering", "lore"} {
		best = maxi(best, g.checkMod(k))
	}
	m := best + g.al().Rank
	if g.alTool(1) {
		m++
	}
	return m
}

func (g *Game) alWhyNot(npc bool) string {
	if npc || g.homeHere() != nil {
		return ""
	}
	if g.Cur == nil || g.P == nil {
		return "Nowhere to brew."
	}
	if why := g.canCook(); why != "" {
		return strings.Replace(why, "cook", "brew", 1)
	}
	return ""
}

func (g *Game) alWaterNear() bool {
	for dy := -1; dy <= 1; dy++ {
		for dx := -1; dx <= 1; dx++ {
			if x, y := g.P.X+dx, g.P.Y+dy; g.Cur.In(x, y) && g.Cur.WaterAt(x, y) > 0 {
				return true
			}
		}
	}
	return false
}

// alBaseSrc finds the liquid for a base: a container with a dram of it, or open water for water.
func (g *Game) alBaseSrc(b alBase) (*Item, bool) {
	for _, it := range g.P.Inv {
		if it.Kind != "water" || it.Charges <= 0 {
			continue
		}
		e := it.Effect
		if e == "fresh" {
			e = ""
		}
		if e == b.Liquid {
			return it, true
		}
	}
	if b.ID == "water" && g.alWaterNear() {
		return nil, true
	}
	return nil, false
}

func (g *Game) alGain(n int) {
	st := g.al()
	st.XP += n
	th := []int{8, 25, 55, 100, 170}
	for st.Rank < 5 && st.XP >= th[st.Rank] {
		st.Rank++
		g.Msg(fmt.Sprintf("{{G|Your herbalism rises to rank %d.}}", st.Rank))
		if st.Rank == 5 {
			g.alFirst("master", "became a master herbalist")
		}
	}
}

func (g *Game) alFirst(key, text string) {
	st := g.al()
	if st.Firsts[key] {
		return
	}
	st.Firsts[key] = true
	if g.W != nil && g.Cur != nil {
		g.deed("alchemy", text, 0, 0)
	}
}

func alDiscKey(eff map[string]int) string {
	var ids []string
	for p := range eff {
		ids = append(ids, p)
	}
	sort.Strings(ids)
	return strings.Join(ids, "+")
}

func alAllIn(props []string, eff map[string]int) bool {
	for _, p := range props {
		if eff[p] <= 0 {
			return false
		}
	}
	return true
}

func (g *Game) alBestRecipe(eff map[string]int) *alRecipeDef {
	st := g.al()
	var best *alRecipeDef
	for i := range alRecipes {
		rc := &alRecipes[i]
		if st.Recipes[rc.ID] && alAllIn(rc.Props, eff) && (best == nil || len(rc.Props) > len(best.Props)) {
			best = rc
		}
	}
	return best
}

func (g *Game) alBrew(items []*Item) string {
	st := g.al()
	m, b := alMethods[st.method], alBases[st.base]
	if len(items) < 2 || len(items) > 4 {
		return "Choose two to four ingredients."
	}
	if m.Tool > 0 && !g.alTool(m.Tool) {
		return "You need " + map[int]string{1: "a mortar", 2: "an alembic"}[m.Tool] + " for that."
	}
	cont, ok := g.alBaseSrc(b)
	if !ok {
		return "You have no " + b.Name + " to brew with."
	}
	res := alCompute(g, items, false)
	if m.ID == "coating" {
		bad := false
		for p := range res.Eff {
			bad = bad || alPropByID[p].Bad
		}
		if !bad {
			return "These ingredients hold no venom to coat a blade with."
		}
	}
	if cont != nil {
		cont.Charges--
	}
	dc := 6 + len(items) + m.Diff + res.Side
	margin := g.R.Dice(2, 6) + g.alMod() - dc
	// reveal what the brew shows, before the ingredients are used up
	if len(res.Eff) > 0 && margin > -4 {
		for _, it := range items {
			key, tag, _ := alIng(it)
			for _, pw := range g.alPropsOf(key, tag) {
				if res.Eff[pw.ID] > 0 {
					st.learn(key, pw.ID)
				}
			}
		}
	}
	for _, it := range items {
		g.consume(it)
	}
	if len(res.Eff) == 0 || margin <= -4 {
		g.give(&Item{ID: g.newID(), Kind: "scrap", Name: "vial of sludge", Glyph: '!', F: CDarkGreen, Count: 1, Value: 1})
		g.alGain(1)
		if len(res.Eff) == 0 {
			return "The ingredients share nothing. The brew is sludge."
		}
		return "The brew curdles into sludge."
	}
	bonus := m.Pot + st.Rank/2
	if g.alTool(1) {
		bonus++
	}
	if m.ID == "distillation" && g.alTool(2) {
		bonus++
	}
	switch b.ID {
	case "oil":
		if m.ID == "ointment" || m.ID == "coating" {
			bonus++
		} else {
			bonus--
		}
	case "wine", "alcohol":
		if m.ID == "decoction" || m.ID == "distillation" {
			bonus++
		}
	}
	if b.ID == "alcohol" && m.ID != "distillation" {
		bonus--
	}
	if margin >= 4 {
		bonus++
	}
	if margin < 0 {
		bonus--
	}
	eff := map[string]int{}
	for p, l := range res.Eff {
		eff[p] = clampi(l+bonus, 1, 10)
	}
	var leak []string
	if margin < 2 && b.ID != "honey" {
		leak = res.Leak
	}
	msg := ""
	if rc := g.alNewRecipe(eff); rc != "" {
		msg = " " + rc
	}
	it := g.alMakeItem(eff, leak, res.Side, m)
	g.give(it)
	st.Made++
	g.alGain(1 + len(eff))
	g.alFirst("brew", "brewed a first potion from wild herbs")
	if it.Kind == "poison" {
		g.alFirst("poison", "mixed a first weapon poison")
	}
	return "You brew " + it.Name + "." + msg
}

// alNewRecipe records a named recipe or an experimental combination once all its properties are known.
func (g *Game) alNewRecipe(eff map[string]int) string {
	st := g.al()
	for i := range alRecipes {
		rc := &alRecipes[i]
		if !st.Recipes[rc.ID] && alAllIn(rc.Props, eff) {
			st.Recipes[rc.ID] = true
			g.alFirst("recipe", "worked out a potion recipe by experiment")
			return "{{Y|You have worked out the " + rc.Name + " recipe.}}"
		}
	}
	if len(eff) > 0 && g.alBestRecipe(eff) == nil {
		dk := alDiscKey(eff)
		if !st.Disc[dk] {
			st.Disc[dk] = true
			g.alFirst("recipe", "worked out a potion recipe by experiment")
			return "{{Y|A new combination is noted in your herbal.}}"
		}
	}
	return ""
}

func alPropNames(ids []string) string {
	var n []string
	for _, p := range ids {
		n = append(n, alPropByID[p].Name)
	}
	return strings.Join(n, "/")
}

func (g *Game) alMakeItem(eff map[string]int, leak []string, side int, m alMethod) *Item {
	ids := alSortedEff(eff)
	top := 0
	val := 10
	for _, p := range ids {
		top = maxi(top, eff[p])
		val += eff[p] * 6
	}
	name := alPropNames(ids)
	if len(ids) > 2 {
		name = alPropNames(ids[:2]) + "+"
	}
	if rc := g.alBestRecipe(eff); rc != nil {
		name = rc.Name
	}
	it := &Item{ID: g.newID(), Glyph: '!', Extra: map[string]int{}, Count: m.Doses, Power: top, Value: val}
	for _, p := range ids {
		it.Extra["alp:"+p] = eff[p]
	}
	for _, p := range leak {
		it.Extra["alleak:"+p] = 1
	}
	it.Extra["alside"] = side
	var lines []string
	for _, p := range ids {
		lines = append(lines, fmt.Sprintf("%s %d", alPropLabel(p), eff[p]))
	}
	it.Lore = "Effects: " + strings.Join(lines, ", ") + "."
	if len(leak) > 0 {
		it.Lore += " It leaks " + alPropNames(leak) + "."
	}
	if side > 0 {
		it.Lore += " Opposed herbs leave it sour."
	}
	if m.ID == "coating" {
		it.Kind, it.Effect = "poison", "alpoison"
		it.Charges = 3
		it.Count = 0
		it.F = CGreen
		it.Value = val * 2
		it.Name = fmt.Sprintf("%s coating (%d)", name, top)
		bd, bl := "", 0
		for _, p := range ids {
			if alPropByID[p].Bad && eff[p] > bl {
				bd, bl = p, eff[p]
			}
		}
		for i := range alProps {
			if alProps[i].ID == bd {
				it.Extra["alptype"] = i + 1
			}
		}
		it.Power = bl
		return it
	}
	it.Kind, it.Effect = "tonic", "brewed"
	it.F = CMagenta
	if alPropByID[ids[0]].Bad {
		it.F = CRed
	}
	switch m.ID {
	case "ointment":
		it.Extra["alo"] = 1
	case "smoke":
		it.Extra["als"] = 1
	}
	it.Name = fmt.Sprintf("%s %s (%d)", name, m.Noun, top)
	return it
}

// ---- drinking, applying, poisons ---------------------------------------------------

func alItemEffects(it *Item) map[string]int {
	eff := map[string]int{}
	for k, v := range it.Extra {
		if strings.HasPrefix(k, "alp:") && v > 0 {
			eff[strings.TrimPrefix(k, "alp:")] = v
		}
	}
	return eff
}

func alDominant(it *Item) string {
	if t := it.Extra["alptype"]; t > 0 && t <= len(alProps) {
		return alProps[t-1].ID
	}
	ids := alSortedEff(alItemEffects(it))
	for _, p := range ids {
		if alPropByID[p].Bad {
			return p
		}
	}
	return ""
}

// alUseItem is the useItem hook: brewed potions and weapon poisons.
func (g *Game) alUseItem(it *Item) bool {
	switch {
	case it.Kind == "tonic" && it.Effect == "brewed":
		g.alDrink(it)
		return true
	case it.Kind == "poison" && it.Effect == "alpoison":
		g.alCoat(it)
		return true
	}
	return false
}

func (g *Game) alDrink(it *Item) {
	p := g.P
	eff := alItemEffects(it)
	if it.Extra["als"] > 0 {
		hit := 0
		for _, a := range g.Cur.Actors {
			if a != p && a.HP > 0 && g.Hostile(a) && abs(a.X-p.X) <= 3 && abs(a.Y-p.Y) <= 3 {
				for q, l := range eff {
					if alPropByID[q].Bad {
						g.alApply(a, q, l)
					}
				}
				hit++
			}
		}
		for q, l := range eff {
			if !alPropByID[q].Bad {
				g.alApply(p, q, maxi(1, (l+1)/2))
			}
		}
		g.Msg(fmt.Sprintf("You light the bundle. Smoke rolls out; %d creatures breathe it.", hit))
		g.consume(it)
		return
	}
	g.Msg("You use the " + it.Name + ".")
	for q, l := range eff {
		if it.Extra["alo"] > 0 {
			if q == "healing" || q == "fireres" || q == "frostres" || q == "cleansing" || q == "antidote" {
				l = l * 3 / 2
			} else {
				l = maxi(1, l/2)
			}
		}
		g.alApply(p, q, l)
	}
	for k := range it.Extra {
		if strings.HasPrefix(k, "alleak:") {
			g.alApply(p, strings.TrimPrefix(k, "alleak:"), 1)
		}
	}
	if it.Extra["alside"] > 0 {
		p.AddSt("ill", 20)
		g.Msg("{{g|Your stomach turns.}}")
	}
	g.consume(it)
}

// alApply applies one property at a level to an actor (messages only for the player).
func (g *Game) alApply(a *Actor, prop string, lvl int) {
	if a == nil || a.HP <= 0 {
		return
	}
	you := a == g.P
	say := func(s string) {
		if you {
			g.Msg(s)
		}
	}
	dur := 15 + 6*lvl
	switch prop {
	case "healing":
		a.HP = mini(a.MaxHP, a.HP+2+3*lvl)
		a.AddSt("alheal", 5+lvl)
		say("{{G|Warmth knits your wounds.}}")
	case "stamina":
		delete(a.Status, "exhausted")
		a.AddSt("alstam", dur)
		say("{{C|Fatigue lifts.}}")
	case "antidote":
		delete(a.Status, "poisoned")
		delete(a.Status, "ill")
		say("The poison leaves you.")
	case "feverc":
		delete(a.Status, "ill")
		delete(a.Status, "glotrot")
		say("Your fever breaks.")
	case "sleep":
		if !a.Has("undead") {
			a.AddSt("alsleep", 3+lvl*2)
			say("{{b|Drowsiness drags you under.}}")
		}
	case "calm":
		a.AddSt("alcalm", dur)
		say("{{c|A deep calm settles.}}")
	case "rage":
		a.AddSt("alrage", dur)
		say("{{R|Red rage wells up.}}")
	case "fireres":
		delete(a.Status, "burning")
		a.AddSt("alfire", dur*2)
		say("{{O|Heat slides off you.}}")
	case "frostres":
		a.AddSt("alfrost", dur*2)
		say("{{B|The cold cannot touch you.}}")
	case "nightsight":
		a.AddSt("alnight", dur*2)
		say("{{M|The dark turns grey and clear.}}")
	case "strength":
		a.AddSt("mighty", dur)
		a.AddSt("alstrong", dur)
		say("{{O|Your muscles swell.}}")
	case "swiftness":
		a.AddSt("hasted", dur/2)
		a.AddSt("alswift", dur)
		say("{{C|The world slows.}}")
	case "poison":
		if !a.Has("undead") && a.Tpl != "golem" {
			a.AddSt("poisoned", 6+2*lvl)
			say("{{g|Poison burns in your veins.}}")
		}
	case "paralysis":
		a.AddSt("alpara", 2+mini(lvl/2, 6))
		say("{{y|Your limbs lock.}}")
	case "hallucination":
		a.AddSt("alhall", dur)
		say("{{m|The walls breathe.}}")
	case "cleansing":
		for _, s := range []string{"poisoned", "ill", "glotrot", "disoriented", "dazed", "alhall"} {
			delete(a.Status, s)
		}
		say("You feel scoured clean.")
	}
}

func (g *Game) alCoat(it *Item) {
	for _, bp := range g.P.Body {
		w := bp.Item
		if w == nil || w.Kind != "weapon" {
			continue
		}
		if w.Extra == nil {
			w.Extra = map[string]int{}
		}
		w.Extra["alpoison"] = it.Charges
		w.Extra["alppot"] = maxi(it.Power, 1)
		w.Extra["alptype"] = it.Extra["alptype"]
		g.Msg(fmt.Sprintf("You smear the %s on your %s (%d blows).", it.Name, w.Name, it.Charges))
		g.removeInv(it)
		g.alFirst("coat", "coated a blade with a brewed poison")
		return
	}
	g.Msg("Wield a melee weapon first.")
}

// alOnHit applies weapon poison on a hit (hook in combat after a successful hit).
func (g *Game) alOnHit(att, def *Actor, w *Item) {
	if w == nil || w.Extra == nil || w.Extra["alpoison"] <= 0 || def == nil || def.HP <= 0 {
		return
	}
	w.Extra["alpoison"]--
	if t := w.Extra["alptype"]; t > 0 && t <= len(alProps) {
		g.alApply(def, alProps[t-1].ID, maxi(1, w.Extra["alppot"]))
		if att == g.P && w.Extra["alpoison"] == 0 {
			g.Msg("The poison on your " + w.Name + " is spent.")
		}
	}
}

// alPoisonFor: for the stealth agent. Given a poison vial or tainted food, returns its property id
// ("poison", "paralysis", "sleep", "hallucination", ...) or "".
func (g *Game) alPoisonFor(it *Item) string {
	if it == nil {
		return ""
	}
	if it.Kind == "poison" && it.Effect == "alpoison" {
		return alDominant(it)
	}
	if t := it.Extra["alfood"]; t > 0 && t <= len(alProps) {
		return alProps[t-1].ID
	}
	return ""
}

// alPoisonPotency is the potency 1..10 of a poison vial or tainted food.
func (g *Game) alPoisonPotency(it *Item) int {
	if it == nil {
		return 0
	}
	if it.Extra["alfood"] > 0 {
		return maxi(1, it.Extra["alfoodp"])
	}
	return maxi(1, it.Power)
}

// alTaint spends a charge of the vial to taint a food or drink item for assassination.
func (g *Game) alTaint(food, vial *Item) bool {
	p := g.alPoisonFor(vial)
	if p == "" || food == nil || (food.Kind != "food" && food.Kind != "dish" && food.Kind != "water" && food.Kind != "ingredient") {
		return false
	}
	if food.Extra == nil {
		food.Extra = map[string]int{}
	}
	for i := range alProps {
		if alProps[i].ID == p {
			food.Extra["alfood"] = i + 1
		}
	}
	food.Extra["alfoodp"] = g.alPoisonPotency(vial)
	vial.Charges--
	if vial.Charges <= 0 {
		g.removeInv(vial)
	}
	g.alFirst("taint", "tainted a meal with a brewed poison")
	return true
}

// alApplyPoison applies a poison property to a target (for ss: after the victim eats).
func (g *Game) alApplyPoison(target *Actor, prop string, pot int) {
	g.alApply(target, prop, maxi(1, pot))
}

// alYield is the herbalist's gathering multiplier (1..3) for the clutter agent.
func (g *Game) alYield() int { return 1 + g.al().Rank/2 }

// ---- tasting ----------------------------------------------------------------------

func (g *Game) alTaste(it *Item) string {
	st := g.al()
	key, tag, ok := alIng(it)
	if !ok {
		return "That is not something to taste."
	}
	var unk []alPW
	for _, pw := range g.alPropsOf(key, tag) {
		if !st.knows(key, pw.ID) {
			unk = append(unk, pw)
		}
	}
	if len(unk) == 0 {
		return "You already know everything it can do."
	}
	pw := unk[g.R.Int(0, len(unk)-1)]
	st.learn(key, pw.ID)
	g.consume(it)
	g.alGain(1)
	g.alFirst("taste", "tasted unknown herbs to learn their powers")
	out := "You taste it carefully: " + alPropLabel(pw.ID) + "."
	if alPropByID[pw.ID].Bad {
		if g.roll2d6Check(g.checkMod("survival"), 8) {
			return out + " You spit it out in time."
		}
		g.alApply(g.P, pw.ID, 1)
	}
	return out
}

// ---- screen -----------------------------------------------------------------------

type alRow struct {
	label string
	do    func()
}

func (g *Game) alOpen(npc bool) {
	if why := g.alWhyNot(npc); why != "" {
		g.Msg(why)
		return
	}
	st := g.al()
	st.stage, st.sel, st.picks, st.npc, st.note = 0, 0, nil, npc, ""
	g.Mode = "al"
}

func (g *Game) alAbilities() []Ability {
	if g.Cur == nil || g.P == nil || g.alWhyNot(false) != "" {
		return nil
	}
	return []Ability{{ID: "al_brew", Name: "Brew", Target: "self"}}
}

func (g *Game) alAbUse(id string) {
	if id == "al_brew" {
		g.alOpen(false)
	}
}

func (st *alState) picked(it *Item) int {
	for i, p := range st.picks {
		if p == it {
			return i
		}
	}
	return -1
}

type alRec struct {
	name  string
	props []string
}

func (g *Game) alKnownRecs() []alRec {
	st := g.al()
	var out []alRec
	for _, rc := range alRecipes {
		if st.Recipes[rc.ID] {
			out = append(out, alRec{rc.Name, rc.Props})
		}
	}
	var dk []string
	for k := range st.Disc {
		dk = append(dk, k)
	}
	sort.Strings(dk)
	for _, k := range dk {
		props := strings.Split(k, "+")
		out = append(out, alRec{alPropNames(props), props})
	}
	return out
}

// alBrewAgain picks the best ingredients in the pack for a recipe and brews.
func (g *Game) alBrewAgain(props []string) string {
	st := g.al()
	var cands []*Item
	for _, it := range g.alIngredients() {
		key, tag, _ := alIng(it)
		for _, pw := range g.alPropsOf(key, tag) {
			if st.knows(key, pw.ID) && containsStr(props, pw.ID) {
				cands = append(cands, it)
				break
			}
		}
	}
	if len(cands) > 10 {
		cands = cands[:10]
	}
	bestScore := -1 << 30
	var best []*Item
	for mask := 1; mask < 1<<len(cands); mask++ {
		var sub []*Item
		for i := range cands {
			if mask&(1<<i) != 0 {
				sub = append(sub, cands[i])
			}
		}
		if len(sub) < 2 || len(sub) > 4 {
			continue
		}
		res := alCompute(g, sub, false)
		if !alAllIn(props, res.Eff) {
			continue
		}
		sc := -2*res.Side - len(res.Leak) - len(sub)
		for _, l := range res.Eff {
			sc += l
		}
		if sc > bestScore {
			bestScore, best = sc, sub
		}
	}
	if best == nil {
		return "You lack the right ingredients for that."
	}
	return g.alBrew(best)
}

func containsStr(a []string, s string) bool {
	for _, x := range a {
		if x == s {
			return true
		}
	}
	return false
}

func (g *Game) alRows() []alRow {
	st := g.al()
	var out []alRow
	switch st.stage {
	case 0:
		out = append(out,
			alRow{"Base: {{Y|" + alBases[st.base].Name + "}}", func() { st.base = (st.base + 1) % len(alBases) }},
			alRow{"Method: {{Y|" + alMethods[st.method].Name + "}}", func() { st.method = (st.method + 1) % len(alMethods) }},
			alRow{"{{G|Brew now}} {{K|(" + fmt.Sprint(len(st.picks)) + " chosen)}}", func() {
				st.note = g.alBrew(st.picks)
				g.Msg(st.note)
				st.picks = nil
			}},
			alRow{"{{C|Brew again...}}", func() { st.stage, st.sel = 1, 0 }},
			alRow{"{{C|Taste an ingredient...}}", func() { st.stage, st.sel = 2, 0 }},
			alRow{"{{C|What I know...}}", func() { st.stage, st.sel, st.loreOff = 3, 0, 0 }},
		)
		for _, it := range g.alIngredients() {
			it := it
			mark := "[ ]"
			if st.picked(it) >= 0 {
				mark = "{{Y|[x]}}"
			}
			out = append(out, alRow{fmt.Sprintf("%s %s  %s", mark, it.Display(), g.alPropLine(it)), func() {
				if i := st.picked(it); i >= 0 {
					st.picks = append(st.picks[:i], st.picks[i+1:]...)
				} else if len(st.picks) < 4 {
					st.picks = append(st.picks, it)
				}
			}})
		}
	case 1:
		for _, rc := range g.alKnownRecs() {
			rc := rc
			out = append(out, alRow{fmt.Sprintf("%s {{K|(%s)}}", rc.name, alPropNames(rc.props)), func() {
				st.note = g.alBrewAgain(rc.props)
				g.Msg(st.note)
				st.picks, st.stage, st.sel = nil, 0, 0
			}})
		}
		out = append(out, alRow{"{{K|Back}}", func() { st.stage, st.sel = 0, 0 }})
	case 2:
		for _, it := range g.alIngredients() {
			it := it
			out = append(out, alRow{fmt.Sprintf("%s  %s", it.Display(), g.alPropLine(it)), func() {
				st.note = g.alTaste(it)
				g.Msg(st.note)
				st.stage, st.sel = 0, 0
			}})
		}
		out = append(out, alRow{"{{K|Back}}", func() { st.stage, st.sel = 0, 0 }})
	case 3:
		out = append(out, alRow{"{{K|Back}}", func() { st.stage, st.sel = 0, 0 }})
	}
	return out
}

func alKeys(g *Game, k string) {
	st := g.al()
	rows := g.alRows()
	switch k {
	case "Escape":
		if st.stage != 0 {
			st.stage, st.sel = 0, 0
			return
		}
		npc := st.npc
		st.npc = false
		if npc && g.Talk != nil {
			g.Mode = "talk"
			g.talkMenu("\"Anything else?\"")
		} else {
			g.Mode = "main"
		}
	case "ArrowUp":
		st.sel = maxi(0, st.sel-1)
	case "ArrowDown":
		st.sel = mini(maxi(0, len(rows)-1), st.sel+1)
	case "Enter", " ":
		if st.sel < len(rows) {
			rows[st.sel].do()
		}
	}
}

func alDraw(g *Game) {
	st := g.al()
	x, y, _, h := g.panel("Herbalism")
	s := g.sc
	where := "at the fire"
	if st.npc {
		where = "at the herbalist's still (+tools)"
	} else if g.homeHere() != nil {
		where = "at home"
	}
	s.Text(x, y, fmt.Sprintf("{{C|Herbalism rank %d}}  %s  {{K|%d known recipes}}", st.Rank, where, len(g.alKnownRecs())), CGray)
	if st.stage == 3 {
		var keys []string
		for k := range st.Known {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		row := 2
		for _, k := range keys {
			if row >= h-4 {
				break
			}
			s.Text(x, y+row, fmt.Sprintf("%-26s %s", strings.TrimPrefix(k, "bc:"), strings.Join(mapStr(st.Known[k], alPropLabel), " ")), CGray)
			row++
		}
		if len(keys) == 0 {
			s.Text(x, y+2, "{{K|Nothing yet. Taste herbs, brew, read herbals or pay an herbalist.}}", CGray)
		}
		s.Text(x, y+h-3, "{{K|Enter on Back, or Esc, to return}}", CGray)
		s.Text(x, y+h-2, "{{Y|> }}{{K|Back}}", CGray)
		return
	}
	titles := []string{"Choose ingredients (2-4), a base and a method", "Brew again: picks the best ingredients in your pack", "Taste: reveals one property, with some risk"}
	s.Text(x, y+2, "{{W|"+titles[st.stage]+"}}", CGray)
	if st.stage == 0 {
		res := alCompute(g, st.picks, true)
		m := alMethods[st.method]
		dc := 6 + len(st.picks) + m.Diff + res.Side
		var eff []string
		for _, p := range alSortedEff(res.Eff) {
			eff = append(eff, fmt.Sprintf("%s %d", alPropLabel(p), res.Eff[p]))
		}
		if len(eff) == 0 {
			eff = []string{"{{K|nothing you know of}}"}
		}
		s.Text(x, y+3, "Known shared effects: "+strings.Join(eff, ", "), CGray)
		s.Text(x, y+4, fmt.Sprintf("Clean brew chance %d%%{{K|; tools: mortar %v, alembic %v}}", int(chanceFor(g.alMod(), dc)*100+0.5), g.alTool(1), g.alTool(2)), CGray)
	}
	rows := g.alRows()
	n := h - 9
	top := maxi(0, st.sel-n+1)
	for i := top; i < len(rows) && i < top+n; i++ {
		mark := "  "
		if i == st.sel {
			mark = "{{Y|>}} "
		}
		s.Text(x, y+6+i-top, mark+rows[i].label, CGray)
	}
	if st.note != "" {
		s.Text(x, y+h-2, st.note, CGray)
	}
	s.Text(x, y+h-1, "{{K|Up/Down: select   Enter: choose   Esc: back}}", CGray)
}

func mapStr(a []string, f func(string) string) []string {
	out := make([]string, len(a))
	for i, s := range a {
		out[i] = f(s)
	}
	return out
}

// ---- herbalist NPC, books, talk -----------------------------------------------------

func alZone(g *Game, zn *Zone, _ *Rng) {
	if g.W == nil || zn.Z != 0 || zn.Burg < 0 || (zn.Kind != "town" && zn.Kind != "outskirts") {
		return
	}
	bd := g.W.burgByID[zn.Burg]
	if bd == nil {
		return
	}
	r := NewRng(g.Seed + ":al:herbalist:" + zn.Key)
	p := 0.18
	if zn.Kind == "outskirts" {
		p = 0.35
	}
	if !r.Chance(p) {
		return
	}
	x, y, ok := zn.FreeSpot(r, nil)
	if !ok {
		return
	}
	cu := g.W.cultByID[g.W.Culture[clampi(bd.P, 0, len(g.W.Culture)-1)]]
	a := g.makeNPC("herbalist", bd, cu, r)
	a.Title, a.F = "herbalist", CGreen
	a.X, a.Y, a.HomeX, a.HomeY = x, y, x, y
	a.Gold = 120 + 30*a.Level
	a.Desc = fmt.Sprintf("An herbalist of %s, fingers stained green, with bundles of drying leaves hung from a pole.", bd.Name)
	var pool []herb
	for _, h := range herbs {
		if !h.Under {
			pool = append(pool, h)
		}
	}
	for i := 0; i < 5 && len(pool) > 0; i++ {
		j := r.Int(0, len(pool)-1)
		h := pool[j]
		pool = append(pool[:j], pool[j+1:]...)
		a.Inv = append(a.Inv, &Item{ID: g.newID(), Kind: "ingredient", Name: "dried " + h.Name, Effect: h.Tag, Glyph: '♣', F: CGreen, Count: 3, Value: 8, Power: 60})
	}
	a.Inv = append(a.Inv, g.MakeTonic("heal", 2), g.MakeTonic("antidote", 2),
		&Item{ID: g.newID(), Kind: "altool", Name: "mortar and pestle", Glyph: 'u', F: CGray, Value: 25, Extra: map[string]int{"altool": 1}},
		g.alBookItem(r.Int(0, len(alBooks)-1)))
	if r.Chance(0.5) {
		a.Inv = append(a.Inv, &Item{ID: g.newID(), Kind: "altool", Name: "copper alembic", Glyph: 'Y', F: COrange, Value: 80, Extra: map[string]int{"altool": 2}})
	}
	zn.Actors = append(zn.Actors, a)
}

func (g *Game) alBookHerbs(idx int) []herb {
	var out []herb
	for i := 0; i < 3; i++ {
		out = append(out, herbs[(idx*3+i)%len(herbs)])
	}
	return out
}

func (g *Game) alBookItem(idx int) *Item {
	b := alBooks[idx]
	var lines []string
	lines = append(lines, "An herbal. It records:")
	for _, h := range g.alBookHerbs(idx) {
		var ps []string
		for _, pw := range g.alPropsOf(strings.ToLower(h.Name), h.Tag) {
			ps = append(ps, alPropByID[pw.ID].Name)
		}
		lines = append(lines, fmt.Sprintf("%s: %s", h.Name, strings.Join(ps, ", ")))
	}
	for _, id := range b.Recipes {
		rc := alRecipeByID[id]
		lines = append(lines, fmt.Sprintf("Recipe, %s: %s.", rc.Name, alPropNames(rc.Props)))
	}
	it := g.MakeBook(b.Title, strings.Join(lines, "\n"))
	it.Value = 45
	it.Extra = map[string]int{"alb": idx + 1}
	return it
}

// alTurn applies herbals once read (the book popup is shown by the core).
func alTurn(g *Game) {
	if g.P == nil {
		return
	}
	for _, it := range g.P.Inv {
		if it.Kind != "book" || it.Extra["alb"] <= 0 {
			continue
		}
		st := g.al()
		idx := it.Extra["alb"] - 1
		if idx >= len(alBooks) || !g.Read["book:"+it.Name] || st.BooksDone[it.Name] {
			continue
		}
		st.BooksDone[it.Name] = true
		for _, h := range g.alBookHerbs(idx) {
			for _, pw := range g.alPropsOf(strings.ToLower(h.Name), h.Tag) {
				st.learn(strings.ToLower(h.Name), pw.ID)
			}
		}
		for _, id := range alBooks[idx].Recipes {
			st.Recipes[id] = true
		}
		g.alGain(6)
		g.alFirst("book", "learned herbalism from a written herbal")
		g.Msg("{{G|You commit the herbal to memory: new properties and recipes.}}")
	}
}

func (g *Game) alTeach(a *Actor) []*alRecipeDef {
	r := NewRng(g.Seed + ":al:teach:" + fmt.Sprint(a.ID))
	idx := make([]int, len(alRecipes))
	for i := range idx {
		idx[i] = i
	}
	var out []*alRecipeDef
	for len(out) < 3 {
		j := r.Int(0, len(idx)-1)
		rc := &alRecipes[idx[j]]
		idx = append(idx[:j], idx[j+1:]...)
		if a.Role == "apothecary" && (rc.ID == "deathcap" || rc.ID == "dreamsmoke") {
			continue
		}
		out = append(out, rc)
	}
	return out
}

func (g *Game) alSub(text string, opts []Option) {
	g.talkMenu(text)
	g.Opts = append(opts, Option{"Back.", func() { g.talkMenu("\"Anything else?\"") }})
}

func (g *Game) alPotions() []*Item {
	var out []*Item
	for _, it := range g.P.Inv {
		if (it.Kind == "tonic" && it.Effect == "brewed") || (it.Kind == "poison" && it.Effect == "alpoison") {
			out = append(out, it)
		}
	}
	return out
}

func alTalk(g *Game, a *Actor) []Option {
	if a.Role != "herbalist" && a.Role != "apothecary" {
		return nil
	}
	cur := g.Sk.Currency
	st := g.al()
	var out []Option
	out = append(out, Option{"{{O|Use your hearth and still to brew.}}", func() { g.alOpen(true) }})
	out = append(out, Option{"{{C|Tell me what my herbs can do.}}", func() { g.alIdentifyMenu(a) }})
	for _, rc := range g.alTeach(a) {
		rc := rc
		if st.Recipes[rc.ID] {
			continue
		}
		cost := maxi(10, 25+8*len(rc.Props)-st.Rank*3)
		out = append(out, Option{fmt.Sprintf("{{O|Teach me the %s.}} {{K|(%d %s)}}", rc.Name, cost, cur), func() {
			if g.P.Gold < cost {
				g.talkMenu("\"Knowledge feeds me too.\"")
				return
			}
			g.P.Gold -= cost
			st.Recipes[rc.ID] = true
			g.alGain(3)
			g.alFirst("recipe", "learned a potion recipe from an herbalist")
			g.talkMenu(fmt.Sprintf("\"Mind the measures.\" You learn the %s (%s).", rc.Name, alPropNames(rc.Props)))
		}})
	}
	if pots := g.alPotions(); len(pots) > 0 {
		mul := 6
		if a.Role == "herbalist" {
			mul = 8
		}
		total := 0
		for _, it := range pots {
			total += it.Value * maxi(it.Count, 1) * mul / 5
		}
		out = append(out, Option{fmt.Sprintf("{{W|Sell my potions and poisons.}} {{K|(%d, %d %s)}}", len(pots), total, cur), func() {
			for _, it := range pots {
				g.removeInv(it)
			}
			g.P.Gold += total
			g.talkMenu(fmt.Sprintf("\"Good work.\" You are paid %d %s.", total, cur))
		}})
	}
	return out
}

func (g *Game) alIdentifyMenu(a *Actor) {
	st := g.al()
	cur := g.Sk.Currency
	var opts []Option
	seen := map[string]bool{}
	for _, it := range g.alIngredients() {
		it := it
		key, tag, _ := alIng(it)
		if seen[key] {
			continue
		}
		seen[key] = true
		unk := 0
		for _, pw := range g.alPropsOf(key, tag) {
			if !st.knows(key, pw.ID) {
				unk++
			}
		}
		if unk == 0 {
			continue
		}
		fee := 5 + 3*alQual(it)
		if a.Role == "apothecary" {
			fee += 4
		}
		opts = append(opts, Option{fmt.Sprintf("%s {{K|(%d %s)}}", it.Display(), fee, cur), func() {
			if g.P.Gold < fee {
				g.talkMenu("\"Come back with coin.\"")
				return
			}
			g.P.Gold -= fee
			var ps []string
			for _, pw := range g.alPropsOf(key, tag) {
				st.learn(key, pw.ID)
				ps = append(ps, alPropLabel(pw.ID))
			}
			g.alGain(1)
			g.alFirst("ident", "had an herbalist name the powers of a plant")
			g.talkMenu("\"Ah. " + it.Name + ": " + strings.Join(ps, ", ") + ".\"")
		}})
	}
	if len(opts) == 0 {
		g.talkMenu("\"You carry nothing I can teach you about.\"")
		return
	}
	g.alSub("\"Which?\"", opts)
}
