package engine

import (
	"fmt"
	"sort"
	"strings"
	"sync"
)

// Arcane (ar): spell and ritual research, pacts with patrons, and masters who teach.
//   - Research: combine up to three sources (relic, ingredient, book, owned power, essence) with a focus element at a
//     research site. The result is fixed by a hash of (seed, sources, focus): the same combination always gives the same
//     power (name, form, magnitude, cooldown, flaw). It becomes the ability `ar_<id>`; refine to purify or intensify.
//   - Pacts: a patron (from the world's religions and monsters) grants boons in exchange for terms (tithe, tasks,
//     a taboo, a share of XP). Standing rises with kept terms and falls with breaches. Rituals at holy sites and ley
//     nodes gain from the moon, feasts and solstices.
//   - Masters: 3-6 hermits in remote places teach one power or maneuver after a favour, a test or a standing.
//
// State: g.ext("ar") (global: powers, pacts, lessons) and g.ext("arl") (planet-local: tasks, masters met; in unLocalKeys).

type arPower struct {
	ID      string   `json:"id"`
	Name    string   `json:"n"`
	Elem    string   `json:"el"`
	Form    string   `json:"fm"`
	Flaw    string   `json:"fl,omitempty"`
	Mag     int      `json:"mg"`
	Cool    int      `json:"cd"`
	Rng     int      `json:"rg"`
	DC      int      `json:"dc,omitempty"`
	Qual    int      `json:"q"`
	Refined int      `json:"rf,omitempty"`
	Uses    int      `json:"us,omitempty"`
	Src     []string `json:"src,omitempty"`
	Combo   string   `json:"cb"`
	Year    int      `json:"yr,omitempty"`
}

type arPactRec struct {
	PID      string         `json:"pid"`
	PName    string         `json:"pn"`
	PKind    string         `json:"pk"`
	PElem    string         `json:"pe"`
	TKind    string         `json:"tk"`
	TKey     string         `json:"tky,omitempty"`
	TArg     int            `json:"ta,omitempty"`
	TText    string         `json:"tt"`
	Rank     int            `json:"rk"`
	Standing int            `json:"st"`
	Since    int            `json:"sn"`
	TitheDue int            `json:"td,omitempty"`
	Attr     [6]int         `json:"at"`
	Power    string         `json:"pw,omitempty"`
	PowerLvl int            `json:"pl,omitempty"`
	Breaches int            `json:"br,omitempty"`
	Kept     int            `json:"kp,omitempty"`
	Offered  int            `json:"of,omitempty"`
	CallDay  int            `json:"cdy,omitempty"`
	CallUsed int            `json:"cu,omitempty"`
	RiteDay  int            `json:"rd,omitempty"`
	LastHun  int            `json:"lh,omitempty"`
	Food     map[string]int `json:"fd,omitempty"`
}

type arSum struct {
	ID    int `json:"id"`
	Until int `json:"u"`
}

type arState struct {
	Powers   map[string]*arPower `json:"pw,omitempty"`
	Combos   map[string]string   `json:"cb,omitempty"`
	Pacts    []*arPactRec        `json:"pacts,omitempty"`
	Left     int                 `json:"left,omitempty"`
	Grudge   map[string]bool     `json:"gr,omitempty"`
	Taught   map[string]bool     `json:"tg,omitempty"`
	Deeds    map[string]bool     `json:"dd,omitempty"`
	Sums     []arSum             `json:"sums,omitempty"`
	Attuned  int                 `json:"att,omitempty"` // turn+1 until which you are attuned
	AttDay   int                 `json:"atd,omitempty"`
	LastDay  int                 `json:"ld,omitempty"`
	LastXP   int                 `json:"lx,omitempty"`
	Crafted  int                 `json:"cr,omitempty"`
	Backfire int                 `json:"bf,omitempty"`

	// screen state (never saved)
	tab, sel, stage, focus, term int      `json:"-"`
	picks                        []string `json:"-"`
	cur                          string   `json:"-"`
	note                         string   `json:"-"`
}

type arTask struct {
	Kind     string `json:"k"` // slay offer visit
	Text     string `json:"t"`
	Need     int    `json:"n"`
	Have     int    `json:"h"`
	Target   int    `json:"tp,omitempty"`
	Gold     int    `json:"g,omitempty"`
	Deadline int    `json:"dl"`
	Quest    int    `json:"q"`
}

type arMRec struct {
	Met      bool `json:"met,omitempty"`
	Started  bool `json:"st,omitempty"`
	Done     bool `json:"dn,omitempty"`
	RetryDay int  `json:"rt,omitempty"`
}

type arLocal struct {
	Tasks    map[string]*arTask `json:"tasks,omitempty"`
	NextTask map[string]int     `json:"nt,omitempty"`
	Masters  map[string]*arMRec `json:"ms,omitempty"`
	Heard    map[string]bool    `json:"hd,omitempty"`
}

func (g *Game) ar() *arState {
	st := g.ext("ar", func() any { return &arState{} }).(*arState)
	if st.Powers == nil {
		st.Powers = map[string]*arPower{}
	}
	if st.Combos == nil {
		st.Combos = map[string]string{}
	}
	if st.Grudge == nil {
		st.Grudge = map[string]bool{}
	}
	if st.Taught == nil {
		st.Taught = map[string]bool{}
	}
	if st.Deeds == nil {
		st.Deeds = map[string]bool{}
	}
	return st
}

func (g *Game) arl() *arLocal {
	l := g.ext("arl", func() any { return &arLocal{} }).(*arLocal)
	if l.Tasks == nil {
		l.Tasks = map[string]*arTask{}
	}
	if l.NextTask == nil {
		l.NextTask = map[string]int{}
	}
	if l.Masters == nil {
		l.Masters = map[string]*arMRec{}
	}
	if l.Heard == nil {
		l.Heard = map[string]bool{}
	}
	return l
}

func (l *arLocal) rec(id string) *arMRec {
	r := l.Masters[id]
	if r == nil {
		r = &arMRec{}
		l.Masters[id] = r
	}
	return r
}

func init() {
	unLocalKeys = append(unLocalKeys, "arl")
	modeKeys["ar"] = arKeys
	modeDraws["ar"] = arDraw
	if _, taken := mainKeys["F8"]; !taken {
		mainKeys["F8"] = func(g *Game) { g.Mode, g.ar().tab, g.ar().sel, g.ar().stage = "ar", 0, 0, 0 }
	}
	zoneHooks = append(zoneHooks, arZone)
	talkHooks = append(talkHooks, arTalk)
	turnHooks = append(turnHooks, arTurn)
	killHooks = append(killHooks, arKill)
	enterHooks = append(enterHooks, arEnter)
	sideHooks = append(sideHooks, arSide)
	for _, d := range []*skStatusDef{
		{ID: "arhex", Name: "Hexed", Glyph: 'H', DV: -2, Hit: -2, WIL: -2},
		{ID: "arblind", Name: "Blinded", Glyph: 'B', DV: -4, Hit: -6},
		{ID: "arattuned", Name: "Attuned", Glyph: 'A', WIL: 2},
	} {
		skAddStatus(d)
	}
}

func arH(parts ...string) uint64 { return HashStr(strings.Join(parts, "|")) }

func arCap(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func (g *Game) arFam() string { return arSkinFor(g.Genre).Fam }

func (g *Game) arDeed(key, text string) {
	st := g.ar()
	if st.Deeds[key] {
		return
	}
	st.Deeds[key] = true
	g.deed("arcane", text, 0, 0)
}

func (g *Game) arHostileNear() bool {
	if g.Cur == nil {
		return false
	}
	for _, a := range g.Cur.Actors {
		if a.HP > 0 && a.AI == "hostile" && dist(a, g.P) <= 12 && g.visible(a.X, a.Y) {
			return true
		}
	}
	return false
}

func (g *Game) arLore() int { return maxi(g.checkMod("lore"), g.P.M(WIL)) }

// ---- calendar ----

func (g *Game) arMoonFull() bool { d := g.Day() % 28; return d >= 13 && d <= 15 }

func (g *Game) arMoonName() string {
	d := g.Day() % 28
	switch {
	case d >= 13 && d <= 15:
		return "full moon"
	case d == 27 || d <= 1:
		return "new moon"
	case d < 13:
		return "waxing moon"
	}
	return "waning moon"
}

// arRiteBonus: bonus points for a ceremony from the sky and the calendar, with the reasons.
func (g *Game) arRiteBonus() (int, []string) {
	n := 0
	var why []string
	if g.arMoonFull() {
		n++
		why = append(why, "full moon")
	}
	if g.Day()%SeasonDays == 0 {
		n++
		why = append(why, "turn of the season")
	}
	if _, f := g.feastToday(); f != nil {
		n++
		why = append(why, f.Name)
	}
	if k, _ := g.arSite(); k == "ley" {
		n++
		why = append(why, "ley node")
	}
	return n, why
}

// ---- sites ----

func arHas(s string, words ...string) bool {
	for _, w := range words {
		if strings.Contains(s, w) {
			return true
		}
	}
	return false
}

// arSite classifies the current place: home library workshop shrine ley ruin ("" none).
func (g *Game) arSite() (string, string) {
	zn := g.Cur
	if zn == nil {
		return "", "nowhere in particular"
	}
	if g.homeHere() != nil {
		return "home", "your home"
	}
	if zn.Z == 0 && g.W.In(zn.WX, zn.WY) {
		if l, _ := swLeyAt(g.W, g.W.Idx(zn.WX, zn.WY)); l != nil {
			return "ley", "a ley node"
		}
	}
	n := strings.ToLower(zn.Name)
	switch {
	case arHas(n, "academy", "universit", "librar", "college", "scriptor", "observator"):
		return "library", "a library"
	case arHas(n, "temple", "shrine", "cathedral", "church", "sanctum", "monastery", "pagoda", "abbey"):
		return "shrine", "a shrine"
	case arHas(n, "forge", "artisan", "workshop", "smith", "tinker", "laborator", "alchem"):
		return "workshop", "a workshop"
	case zn.Kind == "wonder" || zn.Kind == "ruin" || zn.Kind == "crypt":
		return "ruin", "old stones"
	}
	return "", "no place for it"
}

func arHoly(kind string) bool { return kind == "shrine" || kind == "ley" || kind == "ruin" }

func arSiteMul(kind, elem string) int {
	switch kind {
	case "library":
		return 75
	case "ley":
		return 70
	case "workshop":
		if elem == "fire" || elem == "force" || elem == "lightning" {
			return 80
		}
		return 90
	case "shrine":
		if elem == "life" || elem == "mind" || elem == "death" {
			return 80
		}
		return 90
	}
	return 100
}

// ---- research: sources and design ----

type arSrc struct {
	Key, Label, Kind, Elem, Hint string
	Pot                          int
	Item                         *Item
	Ess                          string
}

func (g *Game) arElemOf(key, effect string) string {
	if e, ok := arPowElem[effect]; ok {
		return e
	}
	return arElems[arH(g.Seed, "elem", key)%uint64(len(arElems))]
}

func (g *Game) arSources() []arSrc {
	var out []arSrc
	seen := map[string]bool{}
	add := func(s arSrc) {
		if !seen[s.Key] {
			seen[s.Key] = true
			out = append(out, s)
		}
	}
	items := append([]*Item{}, g.P.Inv...)
	for _, bp := range g.P.Body {
		if bp.Item != nil {
			items = append(items, bp.Item)
		}
	}
	for _, it := range items {
		switch it.Kind {
		case "artifact":
			key := "art:" + strings.ToLower(it.Name)
			add(arSrc{Key: key, Label: "relic: " + it.Name, Kind: "relic", Elem: g.arElemOf(key, it.Effect), Hint: arPowForm[it.Effect], Pot: clampi(3+it.Tier+it.Power/2, 3, 9), Item: it})
		case "ingredient":
			key := "ing:" + strings.ToLower(it.Name)
			add(arSrc{Key: key, Label: "ingredient: " + it.Name, Kind: "ingredient", Elem: g.arElemOf(key, ""), Pot: clampi(1+it.Value/12, 1, 4), Item: it})
		case "book":
			key := "book:" + strings.ToLower(it.Name)
			eff := ""
			if strings.HasPrefix(it.Effect, "pow:") {
				eff = strings.TrimPrefix(it.Effect, "pow:")
			}
			add(arSrc{Key: key, Label: "book: " + it.Name, Kind: "book", Elem: g.arElemOf(key, eff), Hint: arPowForm[eff], Pot: clampi(1+it.Value/15, 1, 4), Item: it})
		}
	}
	var ids []string
	for id, lv := range g.P.Powers {
		if pd := powerByID[id]; lv > 0 && pd != nil && !pd.Defect {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	for _, id := range ids {
		add(arSrc{Key: "pow:" + id, Label: "power: " + g.Sk.PowerName(id), Kind: "power", Elem: g.arElemOf("pow:"+id, id), Hint: arPowForm[id], Pot: 2 + g.P.Powers[id]})
	}
	var en []string
	for name, n := range g.craftSt().Ess {
		if n > 0 {
			en = append(en, name)
		}
	}
	sort.Strings(en)
	for _, name := range en {
		add(arSrc{Key: "ess:" + name, Label: name, Kind: "essence", Elem: g.arElemOf("ess:"+name, ""), Pot: 3, Ess: name})
	}
	return out
}

func arCombo(srcs []arSrc, focus string) string {
	keys := make([]string, 0, len(srcs))
	for _, s := range srcs {
		keys = append(keys, s.Key)
	}
	sort.Strings(keys)
	return strings.Join(keys, "|") + "#" + focus
}

// arDesign is pure: the power depends only on the seed, the sources and the focus.
func (g *Game) arDesign(srcs []arSrc, focus string) *arPower {
	combo := arCombo(srcs, focus)
	r := NewRng(g.Seed + ":ar:" + combo)
	pot := 0
	var keys []string
	w := map[string]int{}
	for f, n := range arFormW[focus] {
		w[f] = n
	}
	for _, s := range srcs {
		pot += s.Pot
		keys = append(keys, s.Key)
		if s.Hint != "" {
			w[s.Hint] += 7
		}
		if s.Elem == focus {
			pot++
		}
	}
	sort.Strings(keys)
	total := 0
	for _, f := range arForms {
		total += w[f]
	}
	form := "bolt"
	if total > 0 {
		roll := r.Int(1, total)
		for _, f := range arForms {
			roll -= w[f]
			if roll <= 0 {
				form = f
				break
			}
		}
	}
	mag := clampi(1+pot/2+r.Int(0, 2)+boolInt(len(srcs) == 3), 1, 10)
	cool := (arFormCool[form] + mag*2) * (80 + r.Int(0, 40)) / 100
	if cool < 5 {
		cool = 5
	}
	rng := clampi(5+mag/2, 5, 14)
	if form == "teleport" {
		rng = clampi(4+mag*2, 6, 20)
	}
	pw := &arPower{Elem: focus, Form: form, Mag: mag, Cool: cool, Rng: rng, Src: keys, Combo: combo}
	pFlaw := clampi([]int{0, 85, 65, 45}[clampi(len(srcs), 1, 3)]-pot*2, 10, 90)
	if r.Int(1, 100) <= pFlaw {
		fl := arElemFlaws[focus]
		pw.Flaw = fl[r.Int(0, len(fl)-1)]
		pw.DC = 7 + r.Int(0, 4) + mag/3
	}
	pw.Qual = clampi(pot/4+r.Int(0, 1), 0, 4)
	fam := g.arFam()
	pw.Name = Pick(r, arEpithets) + " " + arElemWord[fam][focus] + " " + arFormNoun[fam][form]
	pw.ID = fmt.Sprintf("%x", arH(g.Seed, "arid", combo)&0xffffff)
	return pw
}

func (g *Game) arHours(srcs []arSrc, pw *arPower, kind string) int {
	h := 6 + 8*len(srcs) + pw.Mag*2
	h = h * arSiteMul(kind, pw.Elem) / 100
	h = h * (100 - clampi(g.arLore()*4, 0, 30)) / 100
	if g.ar().Attuned > g.Turn {
		h = h * 85 / 100
	}
	return maxi(h, 4)
}

func (g *Game) arFee(n, mag int) int { return 20*n + 10*mag }

func arDescribe(pw *arPower) string {
	s := fmt.Sprintf("%s %s: %s", arCap(pw.Elem), pw.Form, arFormDesc[pw.Form])
	switch pw.Form {
	case "bolt", "cone", "ball", "line", "drain":
		s += fmt.Sprintf(" (about %dd6+%d)", arDiceN(pw), pw.Mag/2)
	case "heal":
		s += fmt.Sprintf(" (about %dd6+%d)", pw.Mag+1, pw.Mag*2)
	}
	return s
}

func arDiceN(pw *arPower) int {
	switch pw.Form {
	case "bolt":
		return 1 + pw.Mag/2
	case "line":
		return 1 + pw.Mag*2/5
	}
	return 1 + pw.Mag/3
}

func arTarget(form string) string {
	switch form {
	case "heal", "ward", "summon":
		return "self"
	}
	return "ranged"
}

// arDoResearch spends the time and materials. Returns the log text.
func (g *Game) arDoResearch(srcs []arSrc, focus string) string {
	st := g.ar()
	if len(srcs) == 0 {
		return "Choose at least one source."
	}
	kind, label := g.arSite()
	if kind == "" {
		return "You need a library, workshop, shrine, ley node, old stones or your own home to work."
	}
	if g.arHostileNear() {
		return "Not with hostile creatures in sight."
	}
	combo := arCombo(srcs, focus)
	if id, ok := st.Combos[combo]; ok {
		return "You already know this working: " + st.Powers[id].Name + "."
	}
	pw := g.arDesign(srcs, focus)
	if _, dup := st.Powers[pw.ID]; dup {
		return "A power with this signature is already known."
	}
	hours := g.arHours(srcs, pw, kind)
	fee := g.arFee(len(srcs), pw.Mag)
	if !g.cuSpend(fee) {
		return fmt.Sprintf("Reagents and fees cost %s; you cannot pay.", g.cuFormat(fee))
	}
	for _, s := range srcs {
		switch s.Kind {
		case "ingredient":
			if s.Item != nil {
				if s.Item.Count > 1 {
					s.Item.Count--
				} else {
					g.removeInv(s.Item)
				}
			}
		case "essence":
			es := g.craftSt().Ess
			es[s.Ess]--
			if es[s.Ess] <= 0 {
				delete(es, s.Ess)
			}
		case "relic":
			if s.Item != nil && s.Item.Charges > 0 {
				s.Item.Charges--
			}
		}
	}
	g.tvPass(hours * 60)
	pw.Year = g.W.Year
	st.Powers[pw.ID] = pw
	st.Combos[combo] = pw.ID
	st.Crafted++
	g.gainXP(20 + pw.Mag*10)
	g.arDeed("first", fmt.Sprintf("Devised the %s %s %q at %s.", arSkinFor(g.Genre).Power, pw.Form, pw.Name, label))
	if pw.Qual >= 4 {
		g.arDeed("masterwork", fmt.Sprintf("Perfected a masterful working, %q.", pw.Name))
	}
	out := fmt.Sprintf("After %d hours at %s you have devised {{M|%s}} (%s).", hours, label, pw.Name, arQualNames[pw.Qual])
	if pw.Flaw != "" {
		out += " It is flawed: " + arFlaws[pw.Flaw].Label + "."
	}
	g.Msg("{{G|" + out + "}}")
	return out
}

// arRefine: mode "purify" removes or eases the flaw; "boost" adds magnitude. Costs time, gold and a reagent.
func (g *Game) arRefine(pw *arPower, mode string) string {
	st := g.ar()
	kind, label := g.arSite()
	if kind == "" {
		return "You need a proper place to work."
	}
	if g.arHostileNear() {
		return "Not with hostile creatures in sight."
	}
	if mode == "purify" && pw.Flaw == "" {
		return "There is no flaw to remove."
	}
	if mode == "boost" && (pw.Refined >= 3 || pw.Mag >= 12) {
		return "It cannot be pushed further."
	}
	// reagent: an essence, else an ingredient
	var ess string
	var ing *Item
	for name, n := range g.craftSt().Ess {
		if n > 0 && (ess == "" || name < ess) {
			ess = name
		}
	}
	if ess == "" {
		for _, it := range g.P.Inv {
			if it.Kind == "ingredient" {
				ing = it
				break
			}
		}
	}
	if ess == "" && ing == nil {
		return "You need an essence or an ingredient as reagent."
	}
	fee := 40 + pw.Mag*10
	hours := 12
	if mode == "boost" {
		fee = 60 + pw.Mag*15
		hours = 16
	}
	hours = hours * arSiteMul(kind, pw.Elem) / 100
	if !g.cuSpend(fee) {
		return fmt.Sprintf("The work costs %s; you cannot pay.", g.cuFormat(fee))
	}
	if ess != "" {
		es := g.craftSt().Ess
		es[ess]--
		if es[ess] <= 0 {
			delete(es, ess)
		}
	} else if ing.Count > 1 {
		ing.Count--
	} else {
		g.removeInv(ing)
	}
	g.tvPass(hours * 60)
	if mode == "purify" {
		if g.R.Dice(2, 6)+g.arLore() >= pw.DC {
			pw.Flaw, pw.DC = "", 0
			pw.Refined++
			g.Msg(fmt.Sprintf("{{G|At %s you purge the flaw from %s.}}", label, pw.Name))
			g.arDeed("purify", fmt.Sprintf("Purged the flaw from %q.", pw.Name))
			return "The flaw is gone."
		}
		pw.DC = maxi(5, pw.DC-2)
		g.Msg("{{y|The flaw resists, but weakens (the backlash check is easier).}}")
		return "The flaw weakens but remains."
	}
	pw.Mag++
	pw.Refined++
	pw.Cool = maxi(5, pw.Cool-2)
	if pw.Refined%2 == 0 {
		pw.Qual = mini(4, pw.Qual+1)
	}
	st.Powers[pw.ID] = pw
	g.Msg(fmt.Sprintf("{{G|%s grows stronger (magnitude %d).}}", pw.Name, pw.Mag))
	return "It grows stronger."
}

// ---- casting ----

func (g *Game) arSandboxGrant() *arPower {
	st := g.ar()
	n := len(st.Powers) + 1
	elems := []string{"fire", "frost", "shock", "void", "light", "blood"}
	forms := []string{"bolt", "beam", "cone", "nova", "mend"}
	el := elems[(n-1)%len(elems)]
	fm := forms[(n-1)%len(forms)]
	id := fmt.Sprintf("sb%d", n)
	pw := &arPower{
		ID:   id,
		Name: fmt.Sprintf("Synthesized %s %s", titleCase(el), titleCase(fm)),
		Elem: el,
		Form: fm,
		Mag:  3 + g.P.Level/3,
		Cool: 14,
		Qual: 3,
	}
	st.Powers[id] = pw
	if g.P.Powers == nil {
		g.P.Powers = map[string]int{}
	}
	g.P.Powers["ar_"+id] = 1
	if g.Cur != nil {
		for _, a := range g.Cur.Actors {
			if a.AI == "follower" {
				if a.Powers == nil {
					a.Powers = map[string]int{}
				}
				a.Powers["ar_"+id] = 1
			}
		}
	}
	return pw
}

func (g *Game) arAbilities() []Ability {
	st := g.ar()
	var out []Ability
	var ids []string
	for id := range st.Powers {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		pw := st.Powers[id]
		out = append(out, Ability{ID: "ar_" + id, Name: pw.Name, Target: arTarget(pw.Form), Cool: g.P.Cool["ar_"+id]})
	}
	for i, pr := range st.Pacts {
		if pr.Standing >= 20 {
			out = append(out, Ability{ID: fmt.Sprintf("ar_call%d", i), Name: fmt.Sprintf("Call %s's emissary (%d)", arShort(pr.PName), g.arCallsLeft(pr)), Target: "self"})
		}
	}
	return out
}

func arShort(name string) string {
	name = strings.TrimPrefix(name, "the ")
	if i := strings.IndexAny(name, ",("); i > 0 {
		name = name[:i]
	}
	return strings.TrimSpace(name)
}

func (g *Game) arFoe(a *Actor) bool {
	return a != nil && a != g.P && a.HP > 0 && a.AI != "follower" && !a.Has("armaster")
}

func (g *Game) arDmg(pw *arPower) int {
	d := g.R.Dice(arDiceN(pw), 6) + pw.Mag/2
	if g.ar().Attuned > g.Turn {
		d = d * 5 / 4
	}
	return maxi(d, 1)
}

var arCause = map[string]string{"fire": "fire", "frost": "frost", "lightning": "lightning", "force": "a force blast", "life": "thorned growth", "death": "a withering curse", "mind": "a psychic lance", "shadow": "shadow"}

func (g *Game) arHit(pw *arPower, d *Actor, scale int) {
	dmg := g.arDmg(pw) * scale / 100
	g.msgHit(pw, d, dmg)
	g.hurt(d, dmg, g.P, arCause[pw.Elem])
	if d.HP <= 0 {
		return
	}
	if !g.R.Chance(0.4 + float64(pw.Mag)*0.04) {
		return
	}
	switch pw.Elem {
	case "fire":
		d.AddSt("burning", 4)
	case "frost":
		d.AddSt("frozen", 2)
		d.AddSt("sluggish", 6)
	case "lightning":
		d.AddSt("stunned", 1)
	case "force":
		d.AddSt("prone", 2)
	case "life":
		d.AddSt("poisoned", 5)
	case "death":
		d.AddSt("bleeding", 5)
	case "mind":
		d.AddSt("dazed", 4)
	case "shadow":
		d.AddSt("arblind", 4)
	}
}

func (g *Game) msgHit(pw *arPower, d *Actor, dmg int) {
	g.Msg(fmt.Sprintf("{{M|%s}} strikes %s for {{R|%d}}.", pw.Name, g.the(d), dmg))
}

// arLineFoes returns foes along the line from the player to (tx,ty) up to rng tiles, stopping at walls.
func (g *Game) arLineFoes(tx, ty, rng int, all bool) []*Actor {
	var out []*Actor
	for i, pt := range line(g.P.X, g.P.Y, tx, ty) {
		if i >= rng || g.Cur.Opaque(pt[0], pt[1]) {
			break
		}
		if a := g.Cur.ActorAt(pt[0], pt[1]); g.arFoe(a) {
			out = append(out, a)
			if !all {
				break
			}
		}
	}
	return out
}

func (g *Game) arCone(tx, ty, rng int) []*Actor {
	vx, vy := tx-g.P.X, ty-g.P.Y
	var out []*Actor
	if vx == 0 && vy == 0 {
		return out
	}
	for _, a := range g.Cur.Actors {
		if !g.arFoe(a) || !g.visible(a.X, a.Y) {
			continue
		}
		ax, ay := a.X-g.P.X, a.Y-g.P.Y
		d := maxi(abs(ax), abs(ay))
		if d == 0 || d > rng {
			continue
		}
		dot := ax*vx + ay*vy
		if dot > 0 && dot*dot*2 >= (ax*ax+ay*ay)*(vx*vx+vy*vy) {
			out = append(out, a)
		}
	}
	return out
}

func (g *Game) arUse(id string, tx, ty int) {
	if strings.HasPrefix(id, "ar_call") {
		g.arCall(id)
		return
	}
	st := g.ar()
	pw := st.Powers[strings.TrimPrefix(id, "ar_")]
	if pw == nil {
		return
	}
	if g.P.Cool[id] > 0 {
		g.Msg(fmt.Sprintf("%s is recharging (%d turns).", pw.Name, g.P.Cool[id]))
		return
	}
	hp0 := g.P.HP
	if pw.Flaw != "" {
		bonus := 0
		if st.Attuned > g.Turn {
			bonus = 2
		}
		if g.R.Dice(2, 6)+g.arLore()+bonus < pw.DC {
			st.Backfire++
			g.Msg("{{R|Your control of " + pw.Name + " slips: " + arFlaws[pw.Flaw].Label + "!}}")
			g.arBacklash(pw)
			if g.P.HP <= 0 || hp0 > 0 && g.P.HP <= 0 {
				return
			}
		}
	}
	if !g.arCast(pw, tx, ty) {
		return
	}
	if g.P.Cool == nil {
		g.P.Cool = map[string]int{}
	}
	g.P.Cool[id] = pw.Cool
	pw.Uses++
	g.spend(1000)
}

func (g *Game) arCast(pw *arPower, tx, ty int) bool {
	p := g.P
	scale := 100
	switch pw.Form {
	case "bolt":
		var t *Actor
		if a := g.Cur.ActorAt(tx, ty); g.arFoe(a) && maxi(abs(tx-p.X), abs(ty-p.Y)) <= pw.Rng {
			t = a
		} else if fs := g.arLineFoes(tx, ty, pw.Rng, false); len(fs) > 0 {
			t = fs[0]
		}
		if t == nil {
			g.Msg("{{K|There is nothing there to strike.}}")
			return false
		}
		g.arHit(pw, t, scale)
	case "line":
		fs := g.arLineFoes(tx, ty, pw.Rng, true)
		if len(fs) == 0 {
			g.Msg("{{K|The beam finds nothing.}}")
		}
		g.Msg("{{M|" + pw.Name + " lances out.}}")
		for _, a := range fs {
			g.arHit(pw, a, scale)
		}
	case "cone":
		fs := g.arCone(tx, ty, clampi(3+pw.Mag/2, 4, 8))
		g.Msg("{{M|" + pw.Name + " sweeps the ground ahead.}}")
		for _, a := range fs {
			g.arHit(pw, a, scale)
		}
	case "ball":
		if maxi(abs(tx-p.X), abs(ty-p.Y)) > pw.Rng || !g.visible(tx, ty) {
			g.Msg("{{K|You can't reach that far.}}")
			return false
		}
		rad := 1 + boolInt(pw.Mag >= 6)
		g.Msg("{{M|" + pw.Name + " bursts open.}}")
		for _, a := range append([]*Actor{}, g.Cur.Actors...) {
			if g.arFoe(a) && maxi(abs(a.X-tx), abs(a.Y-ty)) <= rad {
				g.arHit(pw, a, scale)
			}
		}
		if pw.Elem == "fire" && g.Cur.In(tx, ty) && g.Cur.At(tx, ty).D().Burn {
			g.Cur.Fire[ty*MW+tx] = uint8(g.R.Int(3, 6))
		}
	case "drain":
		t := g.Cur.ActorAt(tx, ty)
		if !g.arFoe(t) || maxi(abs(tx-p.X), abs(ty-p.Y)) > pw.Rng {
			if fs := g.arLineFoes(tx, ty, pw.Rng, false); len(fs) > 0 {
				t = fs[0]
			} else {
				g.Msg("{{K|Nothing there to drain.}}")
				return false
			}
		}
		dmg := g.arDmg(pw)
		g.Msg(fmt.Sprintf("{{M|You drain {{R|%d}} life from %s.}}", dmg, g.the(t)))
		g.hurt(t, dmg, p, "life drain")
		p.HP = mini(p.MaxHP, p.HP+dmg/2+1)
	case "slow", "blind":
		t := g.Cur.ActorAt(tx, ty)
		if !g.arFoe(t) || maxi(abs(tx-p.X), abs(ty-p.Y)) > pw.Rng {
			if fs := g.arLineFoes(tx, ty, pw.Rng, false); len(fs) > 0 {
				t = fs[0]
			} else {
				g.Msg("{{K|Nothing there to affect.}}")
				return false
			}
		}
		if pw.Form == "slow" {
			t.AddSt("sluggish", 8+pw.Mag*2)
			t.AddSt("slowed", 6+pw.Mag)
			g.Msg(g.The(t) + " is dragged to a crawl.")
		} else {
			t.AddSt("arblind", 6+pw.Mag*2)
			g.Msg(g.The(t) + " reels, blinded.")
		}
		g.provoke(t)
	case "heal":
		amt := g.R.Dice(pw.Mag+1, 6) + pw.Mag*2
		p.HP = mini(p.MaxHP, p.HP+amt)
		delete(p.Status, "bleeding")
		if pw.Mag >= 3 {
			delete(p.Status, "poisoned")
		}
		g.Msg(fmt.Sprintf("{{G|%s mends you for %d.}}", pw.Name, amt))
		for _, a := range g.localCompanions() {
			if dist(a, p) <= 3 {
				a.HP = mini(a.MaxHP, a.HP+amt/2)
			}
		}
	case "ward":
		p.AddSt("shielded", 15+pw.Mag*5)
		g.Msg("{{C|" + pw.Name + " closes around you.}}")
		for _, a := range g.localCompanions() {
			if dist(a, p) <= 3 {
				a.AddSt("shielded", 10+pw.Mag*3)
			}
		}
	case "summon":
		tpl := arElemServant[pw.Elem]
		name := strings.ToLower(arElemWord[g.arFam()][pw.Elem]) + " servant"
		if a := g.arSummonFollower(tpl, maxi(1, p.Level/2+pw.Mag/2), name, 60+pw.Mag*20); a == nil {
			g.Msg("There is no room for it to appear.")
			return false
		}
		g.Msg("{{M|" + pw.Name + " calls a servant out of nothing.}}")
	case "teleport":
		if !g.Cur.Walkable(tx, ty) || g.Cur.ActorAt(tx, ty) != nil || !g.visible(tx, ty) || maxi(abs(tx-p.X), abs(ty-p.Y)) > pw.Rng {
			g.Msg("You can't go there.")
			return false
		}
		p.X, p.Y = tx, ty
		g.Msg("{{M|" + pw.Name + " carries you across the gap.}}")
	}
	g.recompute()
	return true
}

func (g *Game) arBacklash(pw *arPower) {
	p := g.P
	switch pw.Flaw {
	case "scorch":
		g.hurt(p, g.R.Dice(1+pw.Mag/3, 6), nil, "a backlash")
	case "numb":
		p.AddSt("stunned", 1+pw.Mag/5)
		p.AddSt("dazed", 4)
	case "wild":
		g.arWild(pw)
	case "hex":
		p.AddSt("arhex", 150+pw.Mag*30)
	case "thirst":
		g.Thirst += 150 + pw.Mag*40
		g.hurt(p, g.R.Dice(1, 4), nil, "a parching draw")
	case "summon":
		g.arHostile(arElemServant[pw.Elem], maxi(1, p.Level/2), "unruly "+strings.ToLower(arElemWord[g.arFam()][pw.Elem])+" echo")
	}
}

func (g *Game) arWild(pw *arPower) {
	p := g.P
	switch g.R.Int(1, 8) {
	case 1:
		g.Msg("{{R|Wild magic: fire blooms around you!}}")
		g.explode(p.X, p.Y, "fire", pw.Mag/2, p)
	case 2:
		if x, y, ok := g.nearestFree(g.Cur, p.X+g.R.Int(-5, 5), p.Y+g.R.Int(-4, 4)); ok {
			p.X, p.Y = x, y
		}
		g.Msg("{{M|Wild magic: the world lurches and you land elsewhere.}}")
	case 3:
		g.Msg("{{R|Wild magic: something steps through.}}")
		g.arHostile("wisp", maxi(1, p.Level/2), "wild echo")
	case 4:
		p.AddSt("sluggish", 10)
		g.Msg("{{y|Wild magic: your limbs turn to syrup.}}")
	case 5:
		p.AddSt("hasted", 15)
		g.Msg("{{G|Wild magic: time hurries for you.}}")
	case 6:
		p.AddSt("stunned", 1)
		for _, a := range g.Cur.Actors {
			if a.HP > 0 && dist(a, p) <= 3 {
				a.AddSt("stunned", 1)
			}
		}
		g.Msg("{{Y|Wild magic: a silent shockwave stops everything near you.}}")
	case 7:
		p.AddSt("shielded", 20)
		g.Msg("{{C|Wild magic: a lucky ward snaps into place.}}")
	default:
		p.AddSt("dazed", 5)
		p.AddSt("confused", 3)
		g.Msg("{{y|Wild magic: your thoughts scatter.}}")
	}
}

// ---- summons ----

func (g *Game) arSummonFollower(tpl string, lvl int, name string, dur int) *Actor {
	p := g.P
	x, y, ok := g.nearestFree(g.Cur, p.X+1, p.Y)
	if !ok || x == p.X && y == p.Y {
		return nil
	}
	a := g.Spawn(tpl, lvl, g.R)
	a.Name, a.Proper = name, false
	a.X, a.Y, a.HomeX, a.HomeY = x, y, x, y
	a.AI, a.Faction, a.F, a.Order = "follower", "player", CMagenta, OrdFollow
	g.comp(a)
	g.Cur.Actors = append(g.Cur.Actors, a)
	st := g.ar()
	st.Sums = append(st.Sums, arSum{ID: a.ID, Until: g.Turn + dur})
	for len(st.Sums) > 3 {
		g.arDismiss(st.Sums[0].ID)
		st.Sums = st.Sums[1:]
	}
	g.recompute()
	return a
}

func (g *Game) arHostile(tpl string, lvl int, name string) *Actor {
	x, y, ok := g.nearestFree(g.Cur, g.P.X+1, g.P.Y+1)
	if !ok {
		return nil
	}
	a := g.Spawn(tpl, lvl, g.R)
	a.Name = name
	a.X, a.Y, a.HomeX, a.HomeY = x, y, x, y
	a.AI = "hostile"
	g.Cur.Actors = append(g.Cur.Actors, a)
	return a
}

func (g *Game) arDismiss(id int) {
	for _, z := range g.Zones {
		for i, a := range z.Actors {
			if a.ID == id {
				z.Actors = append(z.Actors[:i], z.Actors[i+1:]...)
				return
			}
		}
	}
}

func (g *Game) arSummonTick() {
	st := g.ar()
	if len(st.Sums) == 0 {
		return
	}
	keep := st.Sums[:0]
	for _, s := range st.Sums {
		a := g.findActor(s.ID)
		if a == nil || a.HP <= 0 {
			continue
		}
		if g.Turn >= s.Until {
			g.Msg(g.The(a) + " fades away.")
			g.arDismiss(s.ID)
			continue
		}
		keep = append(keep, s)
	}
	st.Sums = keep
}

// ---- patrons and pacts ----

type arPatron struct {
	ID, Name, Kind, Elem, Lore string
}

type arTerm struct {
	Kind string
	Key  string
	Arg  int
	Text string
}

var (
	arPatMu    sync.Mutex
	arPatCache = map[*WorldData][]arPatron{}
)

func (g *Game) arKindName(kind string) string { return arKindLabel[g.arFam()][kind] }

func (g *Game) arPatrons() []arPatron {
	arPatMu.Lock()
	defer arPatMu.Unlock()
	if ps, ok := arPatCache[g.W]; ok {
		return ps
	}
	var ps []arPatron
	seen := map[string]bool{}
	for i, rel := range g.W.Religions {
		if len(ps) >= 5 {
			break
		}
		name := rel.Deity
		if name == "" {
			name = rel.Name
		}
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		h := arH(g.Seed, "patron", fmt.Sprint(i), name)
		kind := arKinds[h%uint64(len(arKinds))]
		el := arKindElems[kind]
		ps = append(ps, arPatron{ID: fmt.Sprintf("r%d", rel.ID), Name: name, Kind: kind, Elem: el[(h>>8)%uint64(len(el))],
			Lore: "Worshipped in the faith of " + rel.Name + "; it answers private bargains too."})
	}
	var mons []int
	for i, m := range g.W.Monsters {
		if !m.Slain && !m.Immortal && m.Name != "" {
			mons = append(mons, i)
		}
	}
	sort.Slice(mons, func(a, b int) bool {
		ma, mb := g.W.Monsters[mons[a]], g.W.Monsters[mons[b]]
		if ma.Power != mb.Power {
			return ma.Power > mb.Power
		}
		return ma.ID < mb.ID
	})
	if len(mons) > 8 {
		mons = mons[:8]
	}
	for _, i := range mons {
		if len(ps) >= 8 {
			break
		}
		m := g.W.Monsters[i]
		if seen[m.Name] {
			continue
		}
		seen[m.Name] = true
		h := arH(g.Seed, "patronm", fmt.Sprint(m.ID))
		if h%3 == 0 {
			continue
		}
		kind := []string{"elder", "demon", "spirit"}[(h>>4)%3]
		el := arKindElems[kind]
		ps = append(ps, arPatron{ID: fmt.Sprintf("m%d", m.ID), Name: m.Name, Kind: kind, Elem: el[(h>>8)%uint64(len(el))],
			Lore: "A power older than the kingdoms that fear it."})
	}
	for i := 0; len(ps) < 3 && i < len(arFallbackPatrons); i++ {
		n := arFallbackPatrons[i]
		if seen[n] {
			continue
		}
		h := arH(g.Seed, "patronf", n)
		kind := arKinds[h%uint64(len(arKinds))]
		el := arKindElems[kind]
		ps = append(ps, arPatron{ID: fmt.Sprintf("f%d", i), Name: n, Kind: kind, Elem: el[(h>>8)%uint64(len(el))], Lore: "Known only from whispers."})
	}
	arPatCache[g.W] = ps
	return ps
}

func (g *Game) arTermsFor(pat arPatron) []arTerm {
	r := NewRng(g.Seed + ":ar:terms:" + pat.ID)
	lvl := g.P.Level
	tithe := arTerm{Kind: "tithe", Arg: 30 + lvl*12}
	tithe.Text = "a tithe of " + g.cuFormat(tithe.Arg) + " every month"
	task := arTerm{Kind: "task", Arg: 4 + r.Int(0, 3)}
	task.Text = fmt.Sprintf("a task every %d days", task.Arg)
	xp := arTerm{Kind: "xp", Arg: 10}
	xp.Text = "a tenth of all the experience you earn"
	tabs := []string{"night", "temple", "beasts", "meat", "hoard"}
	k := Pick(r, tabs)
	tab := arTerm{Kind: "taboo", Key: k}
	if k == "hoard" {
		tab.Arg = 400 + lvl*100
		tab.Text = fmt.Sprintf(arTabooText[k], g.cuFormat(tab.Arg))
	} else {
		tab.Text = arTabooText[k]
	}
	first := tithe
	if r.Chance(0.5) {
		first = xp
	}
	return []arTerm{first, task, tab}
}

func (st *arState) maxPacts() int {
	for _, p := range st.Pacts {
		if p.Rank >= 3 {
			return 2
		}
	}
	return 1
}

func (g *Game) arCallsMax(pr *arPactRec) int { return 1 + pr.Rank }

func (g *Game) arCallsLeft(pr *arPactRec) int {
	if pr.CallDay != g.Day() {
		return g.arCallsMax(pr)
	}
	return maxi(0, g.arCallsMax(pr)-pr.CallUsed)
}

func (g *Game) arCall(id string) {
	st := g.ar()
	var i int
	fmt.Sscanf(strings.TrimPrefix(id, "ar_call"), "%d", &i)
	if i < 0 || i >= len(st.Pacts) {
		return
	}
	pr := st.Pacts[i]
	if pr.Standing < 20 {
		g.Msg("{{K|" + pr.PName + " does not answer.}}")
		return
	}
	if pr.CallDay != g.Day() {
		pr.CallDay, pr.CallUsed = g.Day(), 0
	}
	if pr.CallUsed >= g.arCallsMax(pr) {
		g.Msg("{{K|" + pr.PName + " will send no more emissaries today.}}")
		return
	}
	tpls := arKindServants[pr.PKind]
	tpl := tpls[clampi(pr.Rank, 1, 4)-1]
	name := "emissary of " + arShort(pr.PName)
	if a := g.arSummonFollower(tpl, maxi(1, g.P.Level/2+pr.Rank*2), name, 120+pr.Rank*60); a == nil {
		g.Msg("There is no room for it to appear.")
		return
	}
	pr.CallUsed++
	g.Msg(fmt.Sprintf("{{M|%s sends an emissary to your side.}} {{K|(%d left today)}}", pr.PName, g.arCallsLeft(pr)))
	g.spend(1000)
}

// arApply recomputes boons: attribute bonuses and the granted power, suspended while standing is below 20.
func (g *Game) arApply() {
	for _, pr := range g.ar().Pacts {
		var want [6]int
		wantPow, lvl := "", 0
		if pr.Standing >= 20 {
			if a, ok := arElemAttr[pr.PElem]; ok {
				want[a] = 1 + boolInt(pr.Rank >= 3) + boolInt(pr.Rank >= 4)
			}
			if pr.Rank >= 2 {
				wantPow, lvl = arElemPower[pr.PElem], pr.Rank-1
			}
		}
		for i := range want {
			g.P.Attr[i] += want[i] - pr.Attr[i]
		}
		pr.Attr = want
		if pr.Power != wantPow || pr.PowerLvl != lvl {
			g.arUngrant(pr)
			if wantPow != "" {
				if g.P.Powers == nil {
					g.P.Powers = map[string]int{}
				}
				g.P.Powers[wantPow] += lvl
				pr.Power, pr.PowerLvl = wantPow, lvl
			}
		}
	}
}

func (g *Game) arUngrant(pr *arPactRec) {
	if pr.Power != "" {
		g.P.Powers[pr.Power] -= pr.PowerLvl
		if g.P.Powers[pr.Power] <= 0 {
			delete(g.P.Powers, pr.Power)
		}
	}
	pr.Power, pr.PowerLvl = "", 0
}

func (g *Game) arUnapplyAll(pr *arPactRec) {
	for i := range pr.Attr {
		g.P.Attr[i] -= pr.Attr[i]
		pr.Attr[i] = 0
	}
	g.arUngrant(pr)
}

func (g *Game) arSealCost() int { return 50 + g.P.Level*10 }

// arSeal performs the sealing rite. Returns the log text.
func (g *Game) arSeal(pat arPatron, tm arTerm) string {
	st := g.ar()
	kind, label := g.arSite()
	if !arHoly(kind) {
		return "The rite needs a holy place: a shrine, old stones or a ley node."
	}
	if len(st.Pacts) >= st.maxPacts() {
		return "You can bind yourself to no more patrons yet."
	}
	for _, pr := range st.Pacts {
		if pr.PID == pat.ID {
			return "You are already bound to " + pat.Name + "."
		}
	}
	if st.Grudge[pat.ID] {
		return pat.Name + " will not hear you again."
	}
	if g.arHostileNear() {
		return "Not with hostile creatures in sight."
	}
	cost := g.arSealCost()
	if !g.cuSpend(cost) {
		return fmt.Sprintf("The offering is %s; you cannot pay.", g.cuFormat(cost))
	}
	bonus, why := g.arRiteBonus()
	g.tvPass(180)
	dc := 9 - bonus
	if g.R.Dice(2, 6)+maxi(g.P.M(WIL), g.P.M(EGO)) < dc {
		g.Msg("{{y|The rite at " + label + " is met with silence. The offering is lost.}}")
		return pat.Name + " is unmoved. The offering is lost."
	}
	pr := &arPactRec{PID: pat.ID, PName: pat.Name, PKind: pat.Kind, PElem: pat.Elem, TKind: tm.Kind, TKey: tm.Key, TArg: tm.Arg, TText: tm.Text,
		Rank: 1, Standing: 40, Since: g.Turn, LastHun: g.Hunger}
	if tm.Kind == "tithe" {
		pr.TitheDue = g.Day() + SeasonDays
	}
	st.Pacts = append(st.Pacts, pr)
	st.LastXP = g.XP
	g.arSnapFood(pr)
	g.arApply()
	g.arl().NextTask[pr.PID] = g.Day() + 2
	g.arDeed("pact:"+pat.ID, fmt.Sprintf("Sealed a %s with %s, %s.", arSkinFor(g.Genre).Pact, pat.Name, g.arKindName(pat.Kind)))
	msg := fmt.Sprintf("{{M|%s accepts your %s: %s.}}", pat.Name, arSkinFor(g.Genre).Pact, tm.Text)
	if len(why) > 0 {
		msg += " {{K|(" + strings.Join(why, ", ") + ")}}"
	}
	g.Msg(msg)
	return "The pact is sealed."
}

func (g *Game) arBreach(pr *arPactRec, why string, cost int) {
	pr.Standing -= cost
	pr.Breaches++
	g.Msg(fmt.Sprintf("{{R|You have broken your terms with %s: %s.}} {{K|(standing %d)}}", pr.PName, why, pr.Standing))
	g.P.AddSt("arhex", 120)
	if pr.Standing <= 0 {
		g.arWrath(pr)
	}
}

func (g *Game) arRemovePact(pr *arPactRec) {
	st := g.ar()
	g.arUnapplyAll(pr)
	for i, o := range st.Pacts {
		if o == pr {
			st.Pacts = append(st.Pacts[:i], st.Pacts[i+1:]...)
			break
		}
	}
	loc := g.arl()
	if t := loc.Tasks[pr.PID]; t != nil {
		g.arCloseQuest(t, false)
		delete(loc.Tasks, pr.PID)
	}
	st.Left++
}

func (g *Game) arWrath(pr *arPactRec) {
	st := g.ar()
	st.Grudge[pr.PID] = true
	g.arRemovePact(pr)
	g.P.AddSt("arhex", 800)
	tpls := arKindServants[pr.PKind]
	n := 1 + pr.Rank
	for i := 0; i < n; i++ {
		g.arHostile(tpls[clampi(pr.Rank, 1, 4)-1], maxi(1, g.P.Level-1), "wrathful emissary of "+arShort(pr.PName))
	}
	g.Msg(fmt.Sprintf("{{R|%s casts you out in wrath! Its emissaries come for you.}}", pr.PName))
	g.arDeed("wrath:"+pr.PID, fmt.Sprintf("Cast out in wrath by %s.", pr.PName))
}

func (g *Game) arLeaveCost(pr *arPactRec) int { return (40 + g.P.Level*12) * pr.Rank }

// arLeave ends a pact. pay=true settles the price; pay=false just walks away and is cursed.
func (g *Game) arLeave(pr *arPactRec, pay bool) string {
	if pay {
		cost := g.arLeaveCost(pr)
		if !g.cuSpend(cost) {
			return fmt.Sprintf("The parting price is %s; you cannot pay.", g.cuFormat(cost))
		}
		g.arRemovePact(pr)
		g.Msg("{{y|You settle your account and part from " + pr.PName + ".}}")
		return "You part on fair terms."
	}
	g.arRemovePact(pr)
	g.P.AddSt("arhex", 500)
	g.ar().Grudge[pr.PID] = true
	g.Msg("{{R|You walk away from " + pr.PName + " without paying. A cold mark settles on you.}}")
	g.arDeed("walked:"+pr.PID, fmt.Sprintf("Broke faith with %s.", pr.PName))
	return "You walk away. The mark will fade, but the patron will not forget."
}

// arOffering: a rite at a holy place that raises standing. Once a day per pact.
func (g *Game) arOffering(pr *arPactRec) string {
	kind, _ := g.arSite()
	if !arHoly(kind) {
		return "The rite needs a shrine, old stones or a ley node."
	}
	if pr.RiteDay == g.Day()+1 {
		return "You have already made an offering today."
	}
	if g.arHostileNear() {
		return "Not with hostile creatures in sight."
	}
	cost := 40*pr.Rank + g.P.Level*5
	if !g.cuSpend(cost) {
		return fmt.Sprintf("The offering is %s; you cannot pay.", g.cuFormat(cost))
	}
	bonus, why := g.arRiteBonus()
	g.tvPass(120)
	pr.RiteDay = g.Day() + 1
	gain := 8 + 4*bonus
	pr.Standing = mini(100, pr.Standing+gain)
	g.arApply()
	txt := fmt.Sprintf("%s accepts the offering (+%d standing).", pr.PName, gain)
	if len(why) > 0 {
		txt += " Favoured by: " + strings.Join(why, ", ") + "."
	}
	g.Msg("{{M|" + txt + "}}")
	return txt
}

// arAttune: a ceremony of focus; for a day research is quicker and workings less likely to slip.
func (g *Game) arAttune() string {
	st := g.ar()
	kind, _ := g.arSite()
	if !arHoly(kind) && kind != "library" {
		return "Attunement needs a holy place, a library or a ley node."
	}
	if st.AttDay == g.Day()+1 {
		return "You are already attuned today."
	}
	if g.arHostileNear() {
		return "Not with hostile creatures in sight."
	}
	bonus, why := g.arRiteBonus()
	g.tvPass(180)
	st.AttDay = g.Day() + 1
	dur := DayLen + bonus*360
	st.Attuned = g.Turn + dur
	g.P.AddSt("arattuned", dur)
	txt := fmt.Sprintf("You attune yourself for %d hours: workings strike harder and slip less.", dur/60)
	if len(why) > 0 {
		txt += " (" + strings.Join(why, ", ") + ")"
	}
	g.Msg("{{C|" + txt + "}}")
	return txt
}

// ---- pact tasks (mirrored as g.Quests Kind "pact") ----

func (g *Game) arNewTask(pr *arPactRec) {
	loc := g.arl()
	if loc.Tasks[pr.PID] != nil {
		return
	}
	h := arH(g.Seed, "task", pr.PID, fmt.Sprint(g.Day()))
	kind := []string{"slay", "slay", "offer", "visit"}[h%4]
	t := &arTask{Kind: kind, Deadline: g.Turn + 6*DayLen}
	var wonders []int
	for i, wd := range g.W.Wonders {
		if !wd.Destroyed && g.W.In(wd.P%maxi(g.W.W, 1), wd.P/maxi(g.W.W, 1)) {
			wonders = append(wonders, i)
		}
	}
	if kind == "visit" && len(wonders) == 0 {
		kind, t.Kind = "slay", "slay"
	}
	switch kind {
	case "slay":
		t.Need = 3 + int((h>>8)%3)
		t.Text = fmt.Sprintf("%s demands blood: slay %d hostile creatures within 6 days.", pr.PName, t.Need)
	case "offer":
		t.Gold = 40 + g.P.Level*12
		t.Text = fmt.Sprintf("%s demands an offering of %s within 6 days.", pr.PName, g.cuFormat(t.Gold))
	case "visit":
		best, bd := -1, 1<<30
		for _, i := range wonders {
			wd := g.W.Wonders[i]
			d := arCheb(g.W.W, wd.P, g.W.Idx(g.Cur.WX, g.Cur.WY))
			if d < bd || d == bd && (h>>4)&1 == 1 {
				best, bd = i, d
			}
		}
		wd := g.W.Wonders[best]
		t.Target = wd.P
		t.Need = 1
		t.Text = fmt.Sprintf("%s bids you walk to %s and stand in its shadow within 6 days.", pr.PName, wd.Name)
		t.Deadline = g.Turn + (6+mini(bd, 14))*DayLen
	}
	q := &Quest{ID: g.newID(), Kind: "pact", Title: "Task for " + arShort(pr.PName), Text: t.Text, Giver: pr.PName, TP: t.Target}
	g.Quests = append(g.Quests, q)
	t.Quest = q.ID
	loc.Tasks[pr.PID] = t
	g.Msg("{{M|" + t.Text + "}} {{K|(F8)}}")
}

func arCheb(w, a, b int) int {
	if w <= 0 {
		return 0
	}
	return maxi(abs(a%w-b%w), abs(a/w-b/w))
}

func (g *Game) arCloseQuest(t *arTask, done bool) {
	for _, q := range g.Quests {
		if q.ID == t.Quest {
			q.Done, q.Closed = done, true
		}
	}
}

func (g *Game) arTaskDone(pr *arPactRec, t *arTask) {
	loc := g.arl()
	g.arCloseQuest(t, true)
	delete(loc.Tasks, pr.PID)
	pr.Standing = mini(100, pr.Standing+10)
	loc.NextTask[pr.PID] = g.Day() + maxi(2, pr.TArg)
	g.gainXP(30 + g.P.Level*8)
	g.Msg(fmt.Sprintf("{{G|%s is pleased with your service (+10 standing).}}", pr.PName))
	g.arApply()
}

func (g *Game) arTaskFail(pr *arPactRec, t *arTask, why string) {
	loc := g.arl()
	g.arCloseQuest(t, false)
	delete(loc.Tasks, pr.PID)
	loc.NextTask[pr.PID] = g.Day() + maxi(2, pr.TArg)
	g.arBreach(pr, why, 18)
}

func (g *Game) arPactByID(id string) *arPactRec {
	for _, p := range g.ar().Pacts {
		if p.PID == id {
			return p
		}
	}
	return nil
}

// arPayOffering completes an "offer" task by paying its gold.
func (g *Game) arPayOffering(pr *arPactRec) string {
	t := g.arl().Tasks[pr.PID]
	if t == nil || t.Kind != "offer" {
		return "There is no offering to make."
	}
	if !g.cuSpend(t.Gold) {
		return fmt.Sprintf("The offering is %s; you cannot pay.", g.cuFormat(t.Gold))
	}
	g.arTaskDone(pr, t)
	return "The offering is made."
}

func (g *Game) arRefuseTask(pr *arPactRec) string {
	t := g.arl().Tasks[pr.PID]
	if t == nil {
		return "No task is set."
	}
	g.arCloseQuest(t, false)
	delete(g.arl().Tasks, pr.PID)
	g.arl().NextTask[pr.PID] = g.Day() + maxi(2, pr.TArg)
	g.arBreach(pr, "you refused the task", 12)
	return "You refuse the task."
}

// ---- upkeep ----

func (g *Game) arSnapFood(pr *arPactRec) {
	pr.Food = map[string]int{}
	for _, it := range g.P.Inv {
		if it.Kind == "food" || it.Kind == "ingredient" || it.Kind == "corpse" {
			pr.Food[strings.ToLower(it.Name)] += maxi(it.Count, 1)
		}
	}
	pr.LastHun = g.Hunger
}

func arMeaty(name string) bool {
	for _, w := range arMeatWords {
		if strings.Contains(name, w) {
			return true
		}
	}
	return false
}

func arTurn(g *Game) {
	st := g.ar()
	if len(st.Sums) > 0 {
		g.arSummonTick()
	}
	if st.Attuned != 0 && st.Attuned <= g.Turn {
		st.Attuned = 0
	}
	if len(st.Pacts) == 0 {
		st.LastDay = g.Day()
		return
	}
	for _, pr := range append([]*arPactRec{}, st.Pacts...) {
		switch {
		case pr.TKind == "taboo" && pr.TKey == "meat":
			if g.Hunger < pr.LastHun-80 {
				eaten := ""
				now := map[string]int{}
				for _, it := range g.P.Inv {
					if it.Kind == "food" || it.Kind == "ingredient" || it.Kind == "corpse" {
						now[strings.ToLower(it.Name)] += maxi(it.Count, 1)
					}
				}
				for n, c := range pr.Food {
					if now[n] < c && arMeaty(n) {
						eaten = n
					}
				}
				if eaten != "" {
					g.arBreach(pr, "you ate "+eaten, 15)
				}
				g.arSnapFood(pr)
			} else if g.Turn%30 == 0 {
				g.arSnapFood(pr)
			} else {
				pr.LastHun = g.Hunger
			}
		case pr.TKind == "xp":
			if g.XP > st.LastXP {
				take := (g.XP - st.LastXP) * pr.TArg / 100
				g.XP -= take
				pr.Offered += take
				if pr.Offered >= 60 {
					pr.Offered -= 60
					pr.Standing = mini(100, pr.Standing+1)
				}
			}
		}
	}
	st.LastXP = g.XP
	if g.Turn%60 == 0 {
		g.arApply()
	}
	if d := g.Day(); d != st.LastDay {
		for i := 0; i < mini(d-st.LastDay, 4) && len(st.Pacts) > 0; i++ {
			g.arDaily()
		}
		st.LastDay = d
	}
	// task deadlines
	loc := g.arl()
	for id, t := range loc.Tasks {
		if g.Turn > t.Deadline {
			if pr := g.arPactByID(id); pr != nil {
				g.arTaskFail(pr, t, "the task went undone")
			} else {
				delete(loc.Tasks, id)
			}
		}
	}
}

func (g *Game) arDaily() {
	st := g.ar()
	loc := g.arl()
	for _, pr := range append([]*arPactRec{}, st.Pacts...) {
		if g.arPactByID(pr.PID) == nil {
			continue
		}
		pr.Kept++
		pr.Standing = mini(100, pr.Standing+1)
		switch pr.TKind {
		case "tithe":
			if g.Day() >= pr.TitheDue {
				if g.cuSpend(pr.TArg) {
					pr.TitheDue = g.Day() + SeasonDays
					pr.Standing = mini(100, pr.Standing+3)
					g.Msg(fmt.Sprintf("{{M|You pay %s its monthly tithe.}}", pr.PName))
				} else {
					pr.TitheDue = g.Day() + 7
					g.arBreach(pr, "the tithe is unpaid", 12)
				}
			}
		case "task":
			if loc.Tasks[pr.PID] == nil && g.Day() >= loc.NextTask[pr.PID] {
				g.arNewTask(pr)
			}
		case "taboo":
			if pr.TKey == "hoard" && g.P.Gold > pr.TArg {
				g.arBreach(pr, "you hoard more than you may hold", 6)
			}
		}
		if g.arPactByID(pr.PID) == nil {
			continue
		}
		days := (g.Turn - pr.Since) / DayLen
		need := [5][2]int{{0, 0}, {0, 0}, {55, 10}, {75, 30}, {90, 70}}
		if pr.Rank < 4 && pr.Standing >= need[pr.Rank+1][0] && days >= need[pr.Rank+1][1] {
			pr.Rank++
			g.Msg(fmt.Sprintf("{{G|%s raises you to rank %d.}}", pr.PName, pr.Rank))
			g.arDeed(fmt.Sprintf("rank:%s:%d", pr.PID, pr.Rank), fmt.Sprintf("Rose to rank %d in the service of %s.", pr.Rank, pr.PName))
		}
		if pr.Standing < 20 && pr.Standing > 0 && g.Day()%3 == 0 {
			g.Msg("{{y|" + pr.PName + " is displeased; its gifts are withheld until you mend your standing.}}")
		}
	}
	g.arApply()
}

func arKill(g *Game, d, src *Actor) {
	st := g.ar()
	if len(st.Pacts) == 0 {
		return
	}
	byPlayer := src == g.P
	loc := g.arl()
	for _, pr := range append([]*arPactRec{}, st.Pacts...) {
		if byPlayer && pr.TKind == "taboo" {
			hour := (g.Turn + startMinutes) % DayLen / 60
			switch pr.TKey {
			case "night":
				if hour >= 20 || hour < 5 {
					g.arBreach(pr, "you killed after dark", 15)
				}
			case "beasts":
				if d.Faction == "beasts" || d.Faction == "wildlife" || d.Faction == "vermin" {
					g.arBreach(pr, "you slew a wild beast", 12)
				}
			}
		}
		if t := loc.Tasks[pr.PID]; t != nil && t.Kind == "slay" && (byPlayer || src != nil && src.AI == "follower") && d.AI == "hostile" {
			t.Have++
			if t.Have >= t.Need {
				g.arTaskDone(pr, t)
			}
		}
	}
}

func arEnter(g *Game, zn *Zone) {
	st := g.ar()
	loc := g.arl()
	if len(st.Pacts) > 0 {
		n := strings.ToLower(zn.Name)
		for _, pr := range append([]*arPactRec{}, st.Pacts...) {
			if pr.TKind == "taboo" && pr.TKey == "temple" && arHas(n, "temple", "cathedral", "church", "pagoda", "abbey") {
				g.arBreach(pr, "you set foot in a temple", 15)
			}
			if t := loc.Tasks[pr.PID]; t != nil && t.Kind == "visit" && zn.Z == 0 && g.W.In(zn.WX, zn.WY) && g.W.Idx(zn.WX, zn.WY) == t.Target {
				g.arTaskDone(pr, t)
			}
		}
	}
	for _, m := range g.arMasters() {
		if zn.Z == 0 && g.W.In(zn.WX, zn.WY) && g.W.Idx(zn.WX, zn.WY) == m.P && !loc.Heard[m.ID] {
			loc.Heard[m.ID] = true
			g.Msg("{{M|You sense a presence in this land: someone keeps a " + m.Place + " nearby.}}")
		}
	}
}

func arSide(g *Game) []string {
	st := g.ar()
	var out []string
	for _, pr := range st.Pacts {
		out = append(out, fmt.Sprintf("{{M|%s}} r%d {{K|%d}}", arShort(pr.PName), pr.Rank, pr.Standing))
	}
	if st.Attuned > g.Turn {
		out = append(out, "{{C|Attuned}}")
	}
	return out
}

// ---- masters ----

type arTeach struct{ Kind, ID, Name string }

type arReq struct {
	Kind   string // letter lair fetch test faction
	Skill  string // test: riddle endurance resolve
	DC     int
	Target int // letter: burg id; lair: monster id
	N      int
	Key    string // faction rep key / fetch kind
	Text   string
	Who    string
}

type arMaster struct {
	ID, Name, Title, Story, Place string
	P, ZX, ZY, Burg, Person       int
	Teach                         arTeach
	Req                           arReq
}

var (
	arMasterMu    sync.Mutex
	arMasterCache = map[*WorldData][]*arMaster{}
)

func arCompass(dx, dy int) string {
	s := ""
	switch {
	case dy < -abs(dx)/2:
		s = "north"
	case dy > abs(dx)/2:
		s = "south"
	}
	switch {
	case dx > abs(dy)/2:
		s += "east"
	case dx < -abs(dy)/2:
		s += "west"
	}
	if s == "" {
		return "here"
	}
	return s
}

func (g *Game) arWhere(p int) string {
	w := g.W
	cx, cy := g.Cur.WX, g.Cur.WY
	dx, dy := p%w.W-cx, p/w.W-cy
	return fmt.Sprintf("%d parasangs %s", maxi(abs(dx), abs(dy)), arCompass(dx, dy))
}

func (g *Game) arMasters() []*arMaster {
	arMasterMu.Lock()
	defer arMasterMu.Unlock()
	if ms, ok := arMasterCache[g.W]; ok {
		return ms
	}
	w := g.W
	var burgs []*BurgData
	for i := range w.Burgs {
		if w.Burgs[i].Alive {
			burgs = append(burgs, &w.Burgs[i])
		}
	}
	var ms []*arMaster
	if len(burgs) == 0 || w.W <= 0 {
		arMasterCache[w] = ms
		return ms
	}
	type cand struct{ p, score int }
	var cs []cand
	for p := range w.Biome {
		if w.Biome[p] == 0 || (p < len(w.Burg) && w.Burg[p] != 0) {
			continue
		}
		x, y := p%w.W, p/w.W
		rem := 12
		for _, b := range burgs {
			if d := maxi(abs(b.P%w.W-x), abs(b.P/w.W-y)); d < rem {
				rem = d
			}
		}
		sc := rem*100 + int(arH(w.Seed, "master", fmt.Sprint(p))%97)
		if p < len(w.Height) && w.Height[p] >= 60 {
			sc += 40
		}
		cs = append(cs, cand{p, sc})
	}
	sort.Slice(cs, func(a, b int) bool {
		if cs[a].score != cs[b].score {
			return cs[a].score > cs[b].score
		}
		return cs[a].p < cs[b].p
	})
	n := 3 + int(arH(w.Seed, "master-count")%4)
	var chosen []int
	for _, c := range cs {
		if len(chosen) >= n {
			break
		}
		ok := true
		x, y := c.p%w.W, c.p/w.W
		for _, o := range chosen {
			if maxi(abs(o%w.W-x), abs(o/w.W-y)) < 7 {
				ok = false
				break
			}
		}
		if ok {
			chosen = append(chosen, c.p)
		}
	}
	var people []int
	for i, pe := range w.People {
		if arMasterRoles[strings.ToLower(pe.Role)] {
			people = append(people, i)
		}
	}
	sort.SliceStable(people, func(a, b int) bool {
		da, db := w.People[people[a]].Died > 0, w.People[people[b]].Died > 0
		return da && !db
	})
	// teachings
	var powers []string
	for _, pd := range PowerDefs {
		if !pd.Defect {
			powers = append(powers, pd.ID)
		}
	}
	var mans []string
	for _, t := range skTrees {
		for _, nd := range t.Nodes {
			if nd.Kind == "active" {
				mans = append(mans, nd.ID)
			}
		}
	}
	taken := map[string]bool{}
	fam := arSkinFor(g.Genre)
	for i, p := range chosen {
		h := arH(w.Seed, "master", fmt.Sprint(p))
		m := &arMaster{ID: fmt.Sprintf("ms%d", p), P: p, ZX: int(h % 3), ZY: int((h >> 4) % 3)}
		// nearest living burg
		bb, bd := burgs[0], 1<<30
		for _, b := range burgs {
			if d := maxi(abs(b.P%w.W-p%w.W), abs(b.P/w.W-p/w.W)); d < bd {
				bb, bd = b, d
			}
		}
		m.Burg = bb.ID
		terrain := "wild"
		if p < len(w.Height) && w.Height[p] >= 60 {
			terrain = "mountain"
		} else if p < len(w.Biome) {
			for _, f := range forests {
				if w.Biome[p] == f {
					terrain = "forest"
				}
			}
		}
		m.Place = Pick(NewRng(w.Seed+":ar:place:"+m.ID), arPlaces[terrain])
		m.Title = arCap(fam.Master)
		if len(people) > 0 {
			pe := w.People[people[(i*3+int(h%5))%len(people)]]
			m.Person = pe.ID
			m.Name = pe.Name
			town := "the lowlands"
			if b := w.burgByID[pe.Burg]; b != nil {
				town = b.Name
			}
			if pe.Died > 0 {
				m.Story = fmt.Sprintf("%s was a %s of %s, recorded dead in year %d. The records lie: they walked away and took a %s in the %s, and teach only those who earn it.", pe.Name, strings.ToLower(pe.Role), town, pe.Died, m.Place, terrain)
			} else {
				m.Story = fmt.Sprintf("%s, once a %s of %s, gave up the world for a %s in the %s, and teaches only those who earn it.", pe.Name, strings.ToLower(pe.Role), town, m.Place, terrain)
			}
		} else {
			m.Name = arMasterNames[int(h>>8)%len(arMasterNames)]
			m.Story = fmt.Sprintf("%s keeps a %s far from anyone, and teaches only those who earn it.", m.Name, m.Place)
		}
		// teaching: distinct
		for k := 0; k < 40; k++ {
			hh := arH(w.Seed, "teach", m.ID, fmt.Sprint(k))
			var t arTeach
			if hh%100 < 55 && len(powers) > 0 {
				id := powers[int(hh>>8)%len(powers)]
				t = arTeach{"power", id, g.Sk.PowerName(id)}
			} else if len(mans) > 0 {
				id := mans[int(hh>>8)%len(mans)]
				t = arTeach{"maneuver", id, skNodeByID[id].Name}
			} else {
				continue
			}
			if !taken[t.Kind+t.ID] {
				taken[t.Kind+t.ID] = true
				m.Teach = t
				break
			}
		}
		if m.Teach.ID == "" {
			continue
		}
		m.Req = g.arMakeReq(m, burgs, h)
		ms = append(ms, m)
	}
	arMasterCache[w] = ms
	return ms
}

func (g *Game) arMakeReq(m *arMaster, burgs []*BurgData, h uint64) arReq {
	w := g.W
	kinds := []string{"letter", "lair", "fetch", "test", "faction"}
	start := int(h % 5)
	for k := 0; k < 5; k++ {
		switch kinds[(start+k)%5] {
		case "letter":
			var order []*BurgData
			order = append(order, burgs...)
			sort.Slice(order, func(a, b int) bool {
				da := maxi(abs(order[a].P%w.W-m.P%w.W), abs(order[a].P/w.W-m.P/w.W))
				db := maxi(abs(order[b].P%w.W-m.P%w.W), abs(order[b].P/w.W-m.P/w.W))
				if da != db {
					return da < db
				}
				return order[a].ID < order[b].ID
			})
			if len(order) < 2 {
				continue
			}
			t := order[mini(1+int((h>>6)%3), len(order)-1)]
			return arReq{Kind: "letter", Target: t.ID, Who: t.Name, Text: fmt.Sprintf("carry a sealed letter to an elder in %s", t.Name)}
		case "lair":
			var best *MonsterData
			bd := 1 << 30
			for i := range w.Monsters {
				mo := &w.Monsters[i]
				if mo.Slain || mo.Immortal || mo.Name == "" {
					continue
				}
				d := maxi(abs(mo.P%w.W-m.P%w.W), abs(mo.P/w.W-m.P/w.W))
				if d < bd || d == bd && mo.ID < best.ID {
					best, bd = mo, d
				}
			}
			if best == nil || bd > 18 {
				continue
			}
			return arReq{Kind: "lair", Target: best.ID, Who: best.Name, Text: fmt.Sprintf("slay %s in its lair near %s", best.Name, w.RegionName(best.P))}
		case "fetch":
			if (h>>9)&1 == 0 {
				return arReq{Kind: "fetch", Key: "essence", Text: "bring an essence drawn from a mighty fallen foe"}
			}
			return arReq{Kind: "fetch", Key: "ingredients", N: 3, Text: "bring three rare ingredients as proof of your craft"}
		case "test":
			skill := []string{"riddle", "endurance", "resolve"}[(h>>12)%3]
			text := map[string]string{"riddle": "answer a riddle (a test of Lore)", "endurance": "endure the master's trial of cold and fasting (Endurance)", "resolve": "stand unmoved before a trial of the mind (Willpower)"}[skill]
			return arReq{Kind: "test", Skill: skill, DC: 9 + int((h>>16)%4), Text: text}
		case "faction":
			var key, name string
			if b := w.burgByID[m.Burg]; b != nil && w.stateByID[b.State] != nil {
				key, name = fmt.Sprintf("state:%d", b.State), w.stateByID[b.State].Name
			}
			if os := g.Orders(); len(os) > 0 && (h>>20)%3 == 0 {
				for _, o := range os {
					if !o.Secret {
						return arReq{Kind: "faction", Key: "order:" + o.ID, N: 1, Who: o.Name, Text: "be a sworn member of " + o.Name}
					}
				}
			}
			if key != "" {
				return arReq{Kind: "faction", Key: key, N: 30, Who: name, Text: "have won good standing with " + name + " (reputation 30)"}
			}
		}
	}
	return arReq{Kind: "test", Skill: "riddle", DC: 10, Text: "answer a riddle (a test of Lore)"}
}

func (g *Game) arMasterByID(id string) *arMaster {
	for _, m := range g.arMasters() {
		if m.ID == id {
			return m
		}
	}
	return nil
}

func arMasterOf(g *Game, a *Actor) *arMaster {
	for f := range a.Flags {
		if strings.HasPrefix(f, "arm:") {
			return g.arMasterByID(strings.TrimPrefix(f, "arm:"))
		}
	}
	return nil
}

func arZone(g *Game, zn *Zone, _ *Rng) {
	if zn.Z != 0 || zn.Kind == "town" || zn.Dark || zn.Underwater || zn.Sky {
		return
	}
	for _, m := range g.arMasters() {
		if m.P != g.W.Idx(zn.WX, zn.WY) || m.ZX != zn.ZX || m.ZY != zn.ZY {
			continue
		}
		bd := g.W.burgByID[m.Burg]
		if bd == nil {
			return
		}
		for _, a := range zn.Actors {
			if a.Role == "master" {
				return
			}
		}
		r := NewRng(g.Seed + ":ar:placemaster:" + zn.Key)
		x, y, ok := zn.FreeSpot(r, func(x, y int) bool {
			if x < 4 || x > MW-5 || y < 4 || y > MH-5 {
				return false
			}
			for dy := -2; dy <= 2; dy++ {
				for dx := -2; dx <= 2; dx++ {
					if !zn.Walkable(x+dx, y+dy) || zn.At(x+dx, y+dy).D().Water > 0 {
						return false
					}
				}
			}
			return true
		})
		if !ok {
			if x, y, ok = zn.FreeSpot(r, nil); !ok {
				return
			}
		}
		a := g.makeNPC("scholar", bd, g.W.cultByID[bd.Culture], r)
		a.Role, a.Name, a.Proper, a.Title, a.Desc = "master", m.Name, true, m.Title, m.Story
		a.AI, a.F = "still", CMagenta
		a.X, a.Y, a.HomeX, a.HomeY = x, y, x, y
		a.Level = maxi(a.Level, 10)
		a.Set("armaster")
		a.Set("arm:" + m.ID)
		if m.Person > 0 {
			if pe := g.arPerson(m.Person); pe != nil && pe.Species != "" {
				a.Species = pe.Species
			}
		}
		zn.Actors = append(zn.Actors, a)
		for _, d := range [][2]int{{-2, -2}, {2, -2}, {-2, 2}, {2, 2}} {
			px, py := x+d[0], y+d[1]
			if zn.ActorAt(px, py) == nil && zn.Walkable(px, py) {
				zn.Set(px, py, TPillar)
			}
		}
		if zn.ActorAt(x, y+1) == nil && zn.Walkable(x, y+1) {
			zn.Set(x, y+1, TInscription)
			if zn.Notes == nil {
				zn.Notes = map[int]string{}
			}
			zn.Notes[(y+1)*MW+x] = fmt.Sprintf("A weathered board at the door of a %s: %s. Those who would learn must first prove themselves. Knock, and wait.", m.Place, m.Name)
		}
		return
	}
}

func (g *Game) arPerson(id int) *PersonData {
	for i := range g.W.People {
		if g.W.People[i].ID == id {
			return &g.W.People[i]
		}
	}
	return nil
}

// arReqDone: has the master's condition been met?
func (g *Game) arReqDone(m *arMaster, rec *arMRec) bool {
	switch m.Req.Kind {
	case "lair":
		return rec.Done || g.Slain[m.Req.Target]
	case "faction":
		if strings.HasPrefix(m.Req.Key, "order:") {
			return g.OrderMembership(strings.TrimPrefix(m.Req.Key, "order:")) != nil
		}
		return g.Rep[m.Req.Key] >= m.Req.N
	}
	return rec.Done
}

func (g *Game) arReqProgress(m *arMaster, rec *arMRec) string {
	if g.arReqDone(m, rec) {
		return "{{G|Done.}}"
	}
	switch m.Req.Kind {
	case "letter":
		if !rec.Started {
			return "not begun"
		}
		return "the letter is not yet delivered"
	case "lair":
		return "the beast still lives"
	case "faction":
		if strings.HasPrefix(m.Req.Key, "state:") || !strings.HasPrefix(m.Req.Key, "order:") {
			return fmt.Sprintf("reputation %d of %d", g.Rep[m.Req.Key], m.Req.N)
		}
		return "not a member"
	case "test":
		if rec.RetryDay > g.Day() {
			return "they bid you return in a day"
		}
	}
	return "unfinished"
}

func (g *Game) arPopup(text string, opts ...Option) {
	g.Popup = Wrap(text, 100)
	g.Opts = opts
}

func (g *Game) arMasterMenu(a *Actor, m *arMaster) {
	st := g.ar()
	loc := g.arl()
	rec := loc.rec(m.ID)
	rec.Met = true
	loc.Heard[m.ID] = true
	farewell := Option{"{{K|Farewell.}}", func() { g.talkMenu(g.greeting(a)) }}
	if st.Taught[m.ID] {
		g.arPopup("\"I have given you what I had to give. Go and use it well.\"", farewell)
		return
	}
	var opts []Option
	text := fmt.Sprintf("%s regards you for a long moment. \"I know %s. I do not give it away. To learn it you must %s.\"", m.Name, m.Teach.Name, m.Req.Text)
	if !rec.Started {
		opts = append(opts, Option{"{{Y|I accept your price.}}", func() {
			rec.Started = true
			if m.Req.Kind == "letter" {
				g.give(&Item{ID: g.newID(), Kind: "letter", Name: "sealed letter from " + m.Name, Glyph: '?', F: CMagenta, Value: 1,
					Text: fmt.Sprintf("A letter in %s's hand, sealed with grey wax, for an elder in %s.", m.Name, m.Req.Who), Extra: map[string]int{"ar:letter": 1 + g.arMasterIndex(m)}})
			}
			g.arMasterMenu(a, m)
		}})
		opts = append(opts, farewell)
		g.arPopup(text, opts...)
		return
	}
	if g.arReqDone(m, rec) {
		g.arPopup("\"You have done what I asked. Sit. Watch my hands.\"", Option{"{{G|Learn " + m.Teach.Name + ".}}", func() {
			g.arTeachNow(m)
			g.arMasterMenu(a, m)
		}}, farewell)
		return
	}
	text += " (" + g.arReqProgress(m, rec) + ")"
	switch m.Req.Kind {
	case "fetch":
		opts = append(opts, Option{"{{Y|Give what you asked for.}}", func() {
			if g.arGiveFetch(m) {
				rec.Done = true
			} else {
				g.Msg("{{K|You do not have it yet.}}")
			}
			g.arMasterMenu(a, m)
		}})
	case "test":
		if rec.RetryDay > g.Day() {
			opts = append(opts, Option{"{{K|(They bid you return another day.)}}", func() { g.arMasterMenu(a, m) }})
		} else {
			opts = append(opts, Option{fmt.Sprintf("{{Y|Take the test.}} {{K|(%s)}}", g.arTestOdds(m)), func() { g.arTest(a, m, rec) }})
		}
	}
	opts = append(opts, farewell)
	g.arPopup(text, opts...)
}

func (g *Game) arMasterIndex(m *arMaster) int {
	for i, o := range g.arMasters() {
		if o == m {
			return i
		}
	}
	return 0
}

func (g *Game) arTestMod(m *arMaster) int {
	switch m.Req.Skill {
	case "endurance":
		return g.checkMod("endurance")
	case "resolve":
		return g.P.M(WIL) + g.checkMod("empathy")/2
	}
	return g.checkMod("lore")
}

func (g *Game) arTestOdds(m *arMaster) string {
	return fmt.Sprintf("%d%% to pass, DC %d", int(chanceFor(g.arTestMod(m), m.Req.DC)*100+0.5), m.Req.DC)
}

func (g *Game) arTest(a *Actor, m *arMaster, rec *arMRec) {
	pre := ""
	if m.Req.Skill == "riddle" {
		pre = "\"" + arRiddles[int(arH(g.Seed, "riddle", m.ID)%uint64(len(arRiddles)))] + "\" "
	}
	if m.Req.Skill == "endurance" {
		g.tvPass(360)
		g.hurt(g.P, maxi(1, g.P.MaxHP/10), nil, "the master's trial")
	}
	if g.R.Dice(2, 6)+g.arTestMod(m) >= m.Req.DC {
		rec.Done = true
		g.arPopup(pre+"\n\n\"...Yes. That is the answer. You have passed.\"", Option{"{{G|Continue.}}", func() { g.arMasterMenu(a, m) }})
		return
	}
	rec.RetryDay = g.Day() + 1
	g.arPopup(pre+"\n\n\"No. Come back tomorrow, when you have thought about why.\"", Option{"{{K|Leave.}}", func() { g.talkMenu(g.greeting(a)) }})
}

func (g *Game) arGiveFetch(m *arMaster) bool {
	if m.Req.Key == "essence" {
		es := g.craftSt().Ess
		var names []string
		for n, c := range es {
			if c > 0 {
				names = append(names, n)
			}
		}
		if len(names) == 0 {
			return false
		}
		sort.Strings(names)
		es[names[0]]--
		if es[names[0]] <= 0 {
			delete(es, names[0])
		}
		return true
	}
	need := m.Req.N
	var have []*Item
	for _, it := range g.P.Inv {
		if it.Kind == "ingredient" {
			have = append(have, it)
		}
	}
	n := 0
	for _, it := range have {
		n += maxi(it.Count, 1)
	}
	if n < need {
		return false
	}
	for _, it := range have {
		for need > 0 && maxi(it.Count, 1) > 0 {
			if it.Count > 1 {
				it.Count--
			} else {
				g.removeInv(it)
				need--
				break
			}
			need--
		}
		if need <= 0 {
			break
		}
	}
	return true
}

func (g *Game) arTeachNow(m *arMaster) {
	st := g.ar()
	switch m.Teach.Kind {
	case "power":
		if g.P.Powers == nil {
			g.P.Powers = map[string]int{}
		}
		g.P.Powers[m.Teach.ID] = maxi(g.P.Powers[m.Teach.ID]+1, 2)
	default:
		if nd := skNodeByID[m.Teach.ID]; nd != nil {
			t := skNodeTree[nd.ID]
			s := g.skState()
			s.Open[t.ID] = true
			for i := range t.Nodes {
				if skCost(t.Nodes[i].Q) == 0 {
					g.skGrant(&t.Nodes[i], true)
				}
			}
			for n := nd; n != nil; {
				g.skGrant(n, true)
				if n.Req == "" {
					break
				}
				n = skNodeByID[n.Req]
			}
			g.skSync()
		}
	}
	st.Taught[m.ID] = true
	g.gainXP(100 + g.P.Level*10)
	g.Msg(fmt.Sprintf("{{G|%s teaches you %s.}}", m.Name, m.Teach.Name))
	g.arDeed("taught:"+m.ID, fmt.Sprintf("Learned %s from %s, who keeps a %s.", m.Teach.Name, m.Name, m.Place))
}

func arTalk(g *Game, a *Actor) []Option {
	var out []Option
	if a.Role == "master" {
		if m := arMasterOf(g, a); m != nil {
			out = append(out, Option{"{{M|Ask to be taught.}}", func() { g.arMasterMenu(a, m) }})
		}
	}
	if g.Cur != nil && g.Cur.Burg > 0 && a.AI != "hostile" {
		for _, it := range g.P.Inv {
			idx := it.Extra["ar:letter"] - 1
			if it.Kind != "letter" || idx < 0 || idx >= len(g.arMasters()) {
				continue
			}
			m := g.arMasters()[idx]
			if m.Req.Kind != "letter" || m.Req.Target != g.Cur.Burg {
				continue
			}
			switch a.Role {
			case "elder", "scholar", "priest", "governor", "noble", "merchant":
				it := it
				out = append(out, Option{"{{M|Hand over " + m.Name + "'s letter.}}", func() {
					g.removeInv(it)
					g.arl().rec(m.ID).Done = true
					g.gainXP(20 + g.P.Level*3)
					g.Msg("{{G|The letter is delivered. Return to " + m.Name + ".}}")
					g.talkMenu("\"Ah. His hand, and his grey wax. I will see that he hears of it. Go back to him.\"")
				}})
			}
			break
		}
	}
	return out
}

// arGossip: a rumour of a master, or of a patron. Returns "" most of the time.
func (g *Game) arGossip(a *Actor) string {
	if g.Cur == nil || a == nil {
		return ""
	}
	h := arH(fmt.Sprint(a.ID), fmt.Sprint(g.Turn/600), g.Seed, "argossip")
	if h%100 >= 24 {
		return ""
	}
	loc := g.arl()
	ms := append([]*arMaster{}, g.arMasters()...)
	if h%100 >= 18 {
		if ps := g.arPatrons(); len(ps) > 0 {
			p := ps[int(h>>8)%len(ps)]
			return fmt.Sprintf("\"They say %s, a %s, answers those who strike a bargain at a shrine on a moonlit night. It always asks a price.\"", p.Name, g.arKindName(p.Kind))
		}
	}
	if len(ms) == 0 {
		return ""
	}
	cx, cy, w := g.Cur.WX, g.Cur.WY, g.W.W
	sort.Slice(ms, func(i, j int) bool {
		di := maxi(abs(ms[i].P%w-cx), abs(ms[i].P/w-cy))
		dj := maxi(abs(ms[j].P%w-cx), abs(ms[j].P/w-cy))
		if di != dj {
			return di < dj
		}
		return ms[i].ID < ms[j].ID
	})
	m := ms[int(h>>8)%mini(3, len(ms))]
	loc.Heard[m.ID] = true
	return fmt.Sprintf("\"A %s called %s keeps a %s %s of here. %s teaches only those who prove themselves. Go and see, if you dare.\"", strings.ToLower(m.Title), m.Name, m.Place, g.arWhere(m.P), m.Name)
}
