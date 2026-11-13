package engine

import (
	"fmt"
	"sort"
	"strings"
	"unicode"
)

// Languages (lx): every culture speaks a pseudo-language derived purely from the names already in
// the world file (its given names, houses, places and towns give a syllable inventory). Cultures of
// one sphere are a family: they share roots and differ by regular sound shifts. You learn by
// hearing, asking locals, paying teachers, phrasebooks, inscriptions; companions of that culture
// and a translator gadget lend fluency. Low fluency garbles NPC speech, limits the talk menu and
// worsens prices. State: g.ext("lx") (global, keyed by culture name) and g.ext("lxl") (planet-local:
// lessons per day, insulted towns, practised texts). Never draws g.R for anything generated.

type lxState struct {
	Flu     map[string]int  `json:"f,omitempty"` // culture name → learned fluency 0..100
	Met     map[string]bool `json:"m,omitempty"` // cultures whose people you have spoken among
	Hear    map[string]int  `json:"h,omitempty"` // minutes spent in their towns
	Study   map[string]int  `json:"s,omitempty"` // "name" → day of the last phrasebook session
	Deeds   map[string]bool `json:"d,omitempty"`
	Origin  string          `json:"o,omitempty"`
	OSphere string          `json:"os,omitempty"`
	Sel     int             `json:"-"`
}

type lxLocal struct {
	Taught map[string]int  `json:"t,omitempty"` // "actor:day" → lessons
	Ins    map[int]int     `json:"i,omitempty"` // burg → turn of the last fumbled greeting
	Read   map[string]bool `json:"r,omitempty"` // texts practised
}

func init() {
	unLocalKeys = append(unLocalKeys, "lxl")
	modeKeys["lx"] = lxKeys
	modeDraws["lx"] = lxDraw
	talkHooks = append(talkHooks, lxTalk)
	turnHooks = append(turnHooks, lxTurn)
	zoneHooks = append(zoneHooks, lxZone)
	enterHooks = append(enterHooks, lxEnter)
	sideHooks = append(sideHooks, lxSide)
	for i, c := range lxConcepts {
		if _, ok := lxIdx[c]; !ok {
			lxIdx[c] = i
		}
	}
}

var lxIdx = map[string]int{}

func (g *Game) lx() *lxState {
	st := g.ext("lx", func() any { return &lxState{} }).(*lxState)
	if st.Flu == nil {
		st.Flu = map[string]int{}
	}
	if st.Met == nil {
		st.Met = map[string]bool{}
	}
	if st.Hear == nil {
		st.Hear = map[string]int{}
	}
	if st.Study == nil {
		st.Study = map[string]int{}
	}
	if st.Deeds == nil {
		st.Deeds = map[string]bool{}
	}
	if st.Origin == "" && g.W != nil {
		cu := g.stCulture()
		if cu == nil && g.Cur != nil {
			cu = g.lxCultureAt()
		}
		if cu != nil {
			st.Origin, st.OSphere = cu.Name, cu.Sphere
			st.Flu[cu.Name] = 100
		}
	}
	return st
}

func (g *Game) lxl() *lxLocal {
	l := g.ext("lxl", func() any { return &lxLocal{} }).(*lxLocal)
	if l.Taught == nil {
		l.Taught = map[string]int{}
	}
	if l.Ins == nil {
		l.Ins = map[int]int{}
	}
	if l.Read == nil {
		l.Read = map[string]bool{}
	}
	return l
}

// ---- the pseudo-languages ----

type lxInv struct{ on, nu, co map[string]int }

func lxNewInv() *lxInv { return &lxInv{map[string]int{}, map[string]int{}, map[string]int{}} }

func lxIsV(r rune) bool {
	return strings.ContainsRune("aeiouyàáâäãåæèéêëìíîïòóôöõøùúûü", r)
}

func lxAnalyse(names []string, inv *lxInv) {
	for _, nm := range names {
		for _, w := range strings.FieldsFunc(strings.ToLower(nm), func(r rune) bool { return !unicode.IsLetter(r) }) {
			var groups []struct {
				v bool
				s string
			}
			for _, r := range w {
				v := lxIsV(r)
				if n := len(groups); n > 0 && groups[n-1].v == v {
					groups[n-1].s += string(r)
				} else {
					groups = append(groups, struct {
						v bool
						s string
					}{v, string(r)})
				}
			}
			carry := ""
			for gi, gr := range groups {
				if !gr.v {
					continue
				}
				on := carry
				carry = ""
				if gi == 1 && !groups[0].v {
					on = groups[0].s
				}
				coda := ""
				if gi+1 < len(groups) {
					c := groups[gi+1].s
					if gi+2 == len(groups) {
						coda = c
					} else if rr := []rune(c); len(rr) >= 2 {
						coda, carry = string(rr[:1]), string(rr[1:])
					} else {
						carry = c
					}
				}
				if len([]rune(on)) <= 3 {
					inv.on[on]++
				}
				inv.nu[gr.s]++
				if len([]rune(coda)) <= 2 {
					inv.co[coda]++
				}
			}
		}
	}
}

func lxMerge(a, b *lxInv, wb int) *lxInv {
	o := lxNewInv()
	add := func(dst, src map[string]int, w int) {
		for k, v := range src {
			dst[k] += v * w
		}
	}
	add(o.on, a.on, 1)
	add(o.nu, a.nu, 1)
	add(o.co, a.co, 1)
	if b != nil {
		add(o.on, b.on, wb)
		add(o.nu, b.nu, wb)
		add(o.co, b.co, wb)
	}
	return o
}

func lxPick(r *Rng, m map[string]int, def []string) string {
	if len(m) == 0 {
		return Pick(r, def)
	}
	keys := make([]string, 0, len(m))
	tot := 0
	for k, v := range m {
		keys = append(keys, k)
		tot += v
	}
	sort.Strings(keys)
	x := r.Int(0, maxi(tot-1, 0))
	for _, k := range keys {
		x -= m[k]
		if x < 0 {
			return k
		}
	}
	return keys[len(keys)-1]
}

func lxSyll(r *Rng, inv *lxInv) string {
	on := lxPick(r, inv.on, []string{"k", "t", "m", "n", "s", "r", "l"})
	if r.Chance(0.12) {
		on = ""
	}
	return on + lxPick(r, inv.nu, []string{"a", "e", "i", "o", "u"}) + lxPick(r, inv.co, []string{"", "", "n", "r"})
}

type lxLang struct {
	Key, Name, Sphere string
	Fam               string
	Order             string
	Plural, Polite    string
	Words             map[string]string
	own, pool         *lxInv
	cu                *CultureData
}

var lxCache = map[string]*lxLang{}

func (g *Game) lxLangFor(cu *CultureData) *lxLang {
	if cu == nil || g.W == nil {
		return nil
	}
	ck := fmt.Sprintf("%s|%d|%s", g.W.Seed, g.W.uniIdx, cu.Name)
	if l := lxCache[ck]; l != nil {
		return l
	}
	w := g.W
	names := func(c *CultureData) []string {
		out := append(append(append([]string{}, c.Given...), c.Houses...), c.Places...)
		for i := range w.Burgs {
			if w.Burgs[i].Culture == c.ID {
				out = append(out, w.Burgs[i].Name)
			}
		}
		return out
	}
	own := lxNewInv()
	lxAnalyse(names(cu), own)
	fam := cu.Sphere
	if fam == "" {
		fam = "~" + cu.Name
	}
	pool := lxNewInv()
	for i := range w.Cultures {
		c := &w.Cultures[i]
		if c.ID == cu.ID || (cu.Sphere != "" && c.Sphere == cu.Sphere) {
			lxAnalyse(names(c), pool)
		}
	}
	seed := w.Seed + ":lx:"
	l := &lxLang{Key: cu.Name, Name: cu.Name + " tongue", Sphere: cu.Sphere, Fam: fam, Words: map[string]string{}, own: own, pool: lxMerge(pool, own, 1), cu: cu}
	rg := NewRng(seed + "gram:" + cu.Name)
	l.Order = Pick(rg, lxOrders)
	fr := NewRng(seed + "gramfam:" + fam)
	l.Plural = Pick(fr, lxPluralEnds)
	l.Polite = lxSyll(rg, own)
	if len([]rune(l.Polite)) > 3 {
		l.Polite = string([]rune(l.Polite)[:2])
	}
	// regular sound shifts for this tongue within its family
	sr := NewRng(seed + "shift:" + cu.Name)
	var pairs []string
	used := map[int]bool{}
	for len(pairs) < 6 {
		k := sr.Int(0, len(lxShifts)-1)
		if used[k] {
			continue
		}
		used[k] = true
		pairs = append(pairs, lxShifts[k][0], lxShifts[k][1])
	}
	rep := strings.NewReplacer(pairs...)
	taken := map[string]bool{}
	for i, c := range lxConcepts {
		rr := NewRng(seed + "root:" + fam + ":" + c)
		n := 2
		if i < 40 {
			n = 1
			if rr.Chance(0.4) {
				n = 2
			}
		} else if rr.Chance(0.2) {
			n = 3
		}
		var sb strings.Builder
		for k := 0; k < n; k++ {
			sb.WriteString(lxSyll(rr, l.pool))
		}
		word := rep.Replace(sb.String())
		for taken[word] {
			word += lxSyll(rr, own)
		}
		taken[word] = true
		l.Words[c] = word
	}
	lxCache[ck] = l
	return l
}

func lxCap(s string) string {
	if s == "" {
		return s
	}
	r := []rune(s)
	r[0] = unicode.ToUpper(r[0])
	return string(r)
}

func (l *lxLang) pseudo(word string) string {
	r := NewRng("lx:ps:" + l.Key + ":" + word)
	n := clampi((len(word)+2)/3, 1, 3)
	var sb strings.Builder
	for i := 0; i < n; i++ {
		sb.WriteString(lxSyll(r, l.pool))
	}
	return sb.String()
}

// known words, in learning order, at a fluency
func lxKnownN(f int) int { return clampi(f, 0, 100) * len(lxConcepts) / 100 }

// ---- who speaks what ----

func (g *Game) lxCultureAt() *CultureData {
	if g.W == nil || g.Cur == nil {
		return nil
	}
	if bd := g.W.burgByID[g.Cur.Burg]; bd != nil {
		return g.W.cultByID[bd.Culture]
	}
	p := g.W.Idx(g.Cur.WX, g.Cur.WY)
	if p < 0 || p >= len(g.W.Culture) {
		return nil
	}
	return g.W.cultByID[g.W.Culture[p]]
}

// lxHereID: the culture id of the place you stand in (for inscriptions).
func (g *Game) lxHereID() int {
	if cu := g.lxCultureAt(); cu != nil {
		return cu.ID
	}
	return -1
}

func (g *Game) lxCultureByName(name string) *CultureData {
	if g.W == nil {
		return nil
	}
	for i := range g.W.Cultures {
		if g.W.Cultures[i].Name == name {
			return &g.W.Cultures[i]
		}
	}
	return nil
}

func (g *Game) lxActorCulture(a *Actor) *CultureData {
	return g.lxCultureAt()
}

func (g *Game) lxCompCultures() []string {
	var out []string
	for _, c := range g.localCompanions() {
		if c.Comp == nil || c.Comp.Town == "" {
			continue
		}
		for i := range g.W.Burgs {
			if g.W.Burgs[i].Name == c.Comp.Town {
				if cu := g.W.cultByID[g.W.Burgs[i].Culture]; cu != nil {
					out = append(out, cu.Name)
				}
				break
			}
		}
	}
	return out
}

func (g *Game) lxTranslator() int {
	best := 0
	for _, it := range g.P.Inv {
		if it.Kind == "trophy" && it.Effect == "lx:translator" && it.Extra["lxt"] > best {
			best = it.Extra["lxt"]
		}
	}
	return best
}

// lxStored is the fluency you have actually learned (origin 100, sphere-mates at least 20).
func (g *Game) lxStored(cu *CultureData) int {
	st := g.lx()
	if cu.Name == st.Origin {
		return 100
	}
	f := st.Flu[cu.Name]
	if st.OSphere != "" && cu.Sphere == st.OSphere && f < 20 {
		f = 20
	}
	return f
}

// lxEff is your working fluency: learned + a companion of that culture (+25) + translator gadget.
func (g *Game) lxEff(cu *CultureData) int {
	if cu == nil {
		return 100
	}
	f := g.lxStored(cu)
	if f >= 100 {
		return 100
	}
	for _, n := range g.lxCompCultures() {
		if n == cu.Name {
			f += 25
			break
		}
	}
	f += g.lxTranslator()
	return clampi(f, 0, 100)
}

// lxForeign: the speaker's culture, your working fluency, and whether it is a tongue you lack.
func (g *Game) lxForeign(a *Actor) (*CultureData, int, bool) {
	if a == nil || isFollower(a) || g.Cur == nil {
		return nil, 100, false
	}
	cu := g.lxActorCulture(a)
	if cu == nil {
		return nil, 100, false
	}
	f := g.lxEff(cu)
	return cu, f, f < 100
}

func (g *Game) lxGain(cu *CultureData, n, cp int, why string) {
	st := g.lx()
	old := g.lxStored(cu)
	if old >= cp || n <= 0 {
		return
	}
	nw := mini(cp, old+n)
	st.Flu[cu.Name] = nw
	st.Met[cu.Name] = true
	if nw/25 > old/25 && nw < 100 {
		g.Msg(fmt.Sprintf("{{c|You now follow the %s at about %d%%.}}", g.lxLangFor(cu).Name, nw))
	} else if nw >= 100 {
		g.Msg(fmt.Sprintf("{{G|You speak the %s fluently.}}", g.lxLangFor(cu).Name))
	} else if why != "" {
		g.Msg("{{c|" + why + "}}")
	}
	g.lxDeeds()
}

func (g *Game) lxDeed(key, text string) {
	st := g.lx()
	if st.Deeds[key] {
		return
	}
	st.Deeds[key] = true
	g.deed("language", text, 0, 0)
}

func (g *Game) lxDeeds() {
	st := g.lx()
	n, half, full := 0, 0, 0
	for k, v := range st.Flu {
		if k == st.Origin {
			continue
		}
		if v > 0 {
			n++
		}
		if v >= 50 {
			half++
		}
		if v >= 100 {
			full++
		}
	}
	if n > 0 {
		g.lxDeed("first", "Spoke a first word of a foreign tongue.")
	}
	if half > 0 {
		g.lxDeed("half", "Could hold half a conversation in a foreign tongue.")
	}
	if full > 0 {
		g.lxDeed("fluent", "Became fluent in a foreign tongue.")
	}
	if half >= 3 {
		g.lxDeed("poly", "A polyglot: three foreign tongues spoken well.")
	}
}

// ---- garbling ----

func lxProud(cu *CultureData) bool {
	s := strings.ToLower(strings.Join(cu.Values, " "))
	for _, w := range lxProudWords {
		if strings.Contains(s, w) {
			return true
		}
	}
	return HashStr("lx:proud:"+cu.Name)%3 == 0
}

func lxStem(w string) (string, bool) {
	if _, ok := lxIdx[w]; ok {
		return w, true
	}
	for _, suf := range []string{"es", "s", "ed", "ing"} {
		if strings.HasSuffix(w, suf) {
			if _, ok := lxIdx[w[:len(w)-len(suf)]]; ok {
				return w[:len(w)-len(suf)], true
			}
		}
	}
	return "", false
}

// lxGarble replaces words you have not learned by foreign ones, more as fluency drops.
func lxGarble(l *lxLang, text string, f int, salt uint64, strict bool) string {
	if f >= 100 || l == nil {
		return text
	}
	pv, pp := 0.0, 0.0
	switch {
	case f < 25:
		pv, pp = 1, 0.9
		if strict {
			pp = 1
		}
	case f < 60:
		pv = 0.7 - float64(f-25)/35*0.45
		pp = 0.15 - float64(f-25)/35*0.1
		if strict {
			pp = 0.5 - float64(f-25)/35*0.25
		}
	case f < 90:
		pv = 0.2 - float64(f-60)/30*0.15
		if strict {
			pp = 0.05
		}
	default:
		pv = 0.05
	}
	nk := lxKnownN(f)
	rs := []rune(text)
	var sb strings.Builder
	start := true
	for i := 0; i < len(rs); {
		c := rs[i]
		if c == '{' && i+1 < len(rs) && rs[i+1] == '{' {
			for i < len(rs) {
				sb.WriteRune(rs[i])
				i++
				if rs[i-1] == '|' {
					break
				}
			}
			continue
		}
		if c == '}' && i+1 < len(rs) && rs[i+1] == '}' {
			sb.WriteString("}}")
			i += 2
			continue
		}
		if c == '&' && i+1 < len(rs) && rs[i+1] < 128 && unicode.IsLetter(rs[i+1]) {
			sb.WriteRune(c)
			sb.WriteRune(rs[i+1])
			i += 2
			continue
		}
		if !unicode.IsLetter(c) {
			sb.WriteRune(c)
			if strings.ContainsRune(".!?\"\n“", c) {
				start = true
			}
			i++
			continue
		}
		j := i
		for j < len(rs) && (unicode.IsLetter(rs[j]) || (rs[j] == '\'' && j+1 < len(rs) && unicode.IsLetter(rs[j+1]))) {
			j++
		}
		word := string(rs[i:j])
		lw := strings.ToLower(word)
		proper := !start && unicode.IsUpper(rs[i])
		wasStart := start
		start = false
		i = j
		if proper || len(lw) < 3 {
			sb.WriteString(word)
			continue
		}
		h := float64(HashStr(fmt.Sprintf("%d:%d:%s:%d", salt, j, lw, f))%1000) / 1000
		out := word
		if c, ok := lxStem(lw); ok {
			if lxIdx[c] >= nk && h < pv {
				fw := l.Words[c]
				if f >= 25 {
					fw = "[" + fw + "]"
				}
				out = fw
			}
		} else if h < pp {
			out = l.pseudo(lw)
		}
		if out != word && (wasStart || unicode.IsUpper(rs[i-len([]rune(word))])) && !strings.HasPrefix(out, "[") {
			out = lxCap(out)
		}
		sb.WriteString(out)
	}
	return sb.String()
}

// lxFilter garbles an NPC line by your fluency in their tongue. Hook: talkMenu popup text.
func (g *Game) lxFilter(a *Actor, text string) string {
	cu, f, ok := g.lxForeign(a)
	if !ok {
		if cu != nil && a != nil && !a.Has("lxj") && !isFollower(a) && g.lxStored(cu) >= 100 && cu.Name != g.lx().Origin &&
			HashStr(fmt.Sprint("lxj", a.ID))%3 == 0 && strings.HasPrefix(text, "\"") {
			a.Set("lxj")
			return text + "\n\n{{K|They grin at a turn of phrase you used, a local one, and say you talk like someone born here.}}"
		}
		return text
	}
	l := g.lxLangFor(cu)
	if l == nil {
		return text
	}
	out := lxGarble(l, text, f, uint64(a.ID), false)
	if !a.Has("lxg") {
		a.Set("lxg")
		g.lx().Met[cu.Name] = true
		pre := ""
		if f < 25 {
			out = "\"" + lxCap(l.Words["hello"]) + "!\" " + out
		}
		if f >= 1 {
			pre = fmt.Sprintf("{{K|You greet them: \"%s\".}}\n\n", lxCap(l.Words["hello"]))
		}
		if f < 40 && lxProud(cu) {
			lc := g.lxl()
			if last, seen := lc.Ins[g.Cur.Burg]; !seen || g.Turn-last > DayLen {
				if g.R.Dice(2, 6)+g.P.M(EGO)+f/10 < 8 {
					lc.Ins[g.Cur.Burg] = g.Turn
					g.AddRep(a.Faction, -1, " for your clumsy greeting")
					pre += "{{R|Your greeting comes out wrong. They bristle.}}\n\n"
				}
			}
		}
		out = pre + out
	}
	return out
}

func (g *Game) lxGossipOK(a *Actor) bool {
	_, f, ok := g.lxForeign(a)
	return !ok || f >= 40
}

// lxGossipGarble is the gossip you get from someone you barely understand.
func (g *Game) lxGossipGarble(a *Actor) string {
	cu, f, ok := g.lxForeign(a)
	if !ok {
		return ""
	}
	l := g.lxLangFor(cu)
	topics := []string{"war", "king", "gold", "danger", "death", "road", "market", "ship", "storm", "thief", "god", "town"}
	r := NewRng(fmt.Sprintf("lx:gos:%d:%d", a.ID, g.Day()))
	var bits []string
	for k := 0; k < 3; k++ {
		c := Pick(r, topics)
		if lxIdx[c] < lxKnownN(f) {
			bits = append(bits, c)
		} else {
			bits = append(bits, "["+l.Words[c]+"]")
		}
	}
	g.lxGain(cu, 1, 30, "")
	return fmt.Sprintf("They pour out news in the %s, too fast to follow. You catch only: %s.", l.Name, strings.Join(bits, ", "))
}

// lxText shows an inscription, sign or book in the tongue of a culture, as far as you can read it.
func (g *Game) lxText(culture int, text string) string {
	if g.W == nil || text == "" {
		return text
	}
	cu := g.W.cultByID[culture]
	if cu == nil {
		return text
	}
	f := g.lxEff(cu)
	if f >= 90 {
		return text
	}
	l := g.lxLangFor(cu)
	if l == nil {
		return text
	}
	key := fmt.Sprintf("%d", HashStr(text))
	lc := g.lxl()
	if !lc.Read[key] {
		lc.Read[key] = true
		g.lxGain(cu, 1, 50, "")
	}
	out := lxGarble(l, text, f, HashStr(text), true)
	if f < 60 {
		out += fmt.Sprintf("\n\n{{K|(Written in the %s. You read it at %d%%.)}}", l.Name, f)
	}
	return out
}

// lxBookText: books bought in foreign towns carry Extra["lxc"] = culture id + 1.
func (g *Game) lxBookText(it *Item) string {
	if it == nil || it.Extra["lxc"] <= 0 {
		return it.Text
	}
	return g.lxText(it.Extra["lxc"]-1, it.Text)
}

// ---- trade ----

// lxPriceMod multiplies what you pay (buy) or what you are paid (sell).
func (g *Game) lxPriceMod(a *Actor, buy bool) float64 {
	_, f, ok := g.lxForeign(a)
	if !ok {
		if a != nil && !isFollower(a) && f >= 100 && g.lxCultureAt() != nil && g.lxCultureAt().Name != g.lx().Origin {
			if buy {
				return 0.9
			}
			return 1.1
		}
		return 1
	}
	if f < 25 {
		if buy {
			return 1.2
		}
		return 0.8
	}
	b := 0.10 * float64(f) / 100
	if buy {
		return 1 - b
	}
	return 1 + b
}

// ---- talk ----

func lxTeacher(a *Actor) bool {
	return a.Role == "scholar" || a.Role == "priest" || a.Role == "elder" || a.Role == "bookseller"
}

func lxTalk(g *Game, a *Actor) []Option {
	var out []Option
	cu, f, ok := g.lxForeign(a)
	if ok {
		l := g.lxLangFor(cu)
		if f < 25 {
			out = append(out,
				Option{"{{c|Gesture a greeting.}}", func() { g.lxGesture(a, cu) }},
				Option{"{{c|Ask for directions by pointing.}}", func() { g.talkMenu(g.lxDirections()) }})
		}
		cp := 60
		if lxTeacher(a) {
			cp = 100
		}
		if l != nil && g.lxStored(cu) < cp && !a.Has("hostile") && a.AI != "hostile" {
			cost := 3 + g.P.Level
			if lxTeacher(a) {
				cost = 10 + g.P.Level*2
			}
			out = append(out, Option{fmt.Sprintf("{{O|Teach me a few words of %s.}} {{K|(%s)}}", l.Name, g.cuFormat(cost)), func() { g.lxTeach(a, cu, cost) }})
		}
	}
	if lxTeacher(a) {
		out = append(out, Option{"{{c|Ask about the tongues of other peoples.}}", func() { g.lx().Sel = 0; g.Mode = "lx" }})
	}
	return out
}

// lxTrimOpts cuts the talk menu to gestures, trade and farewell when you understand nothing.
func (g *Game) lxTrimOpts(a *Actor) {
	_, f, ok := g.lxForeign(a)
	if !ok || f >= 25 {
		return
	}
	var keep []Option
	for _, o := range g.Opts {
		t := strings.ToLower(Plain(o.Text))
		if strings.Contains(t, "trade") || strings.Contains(t, "goodbye") || strings.Contains(t, "gesture") ||
			strings.Contains(t, "directions") || strings.Contains(t, "few words") || strings.Contains(t, "tongues") {
			keep = append(keep, o)
		}
	}
	g.Opts = keep
}

func (g *Game) lxGesture(a *Actor, cu *CultureData) {
	g.Turn += 5
	if g.R.Dice(2, 6)+g.P.M(EGO) >= 8 {
		g.lxGain(cu, 1, 20, "")
		g.talkMenu("You point, mime, bow. They laugh, and nod, and repeat a word slowly for you.")
		return
	}
	g.talkMenu("You wave your hands. They stare, then shrug.")
}

func (g *Game) lxDirections() string {
	if g.W == nil || g.Cur == nil {
		return "They shrug."
	}
	best, bd := 1<<30, (*BurgData)(nil)
	for i := range g.W.Burgs {
		b := &g.W.Burgs[i]
		if !b.Alive || b.Layer != "surface" || b.ID == g.Cur.Burg {
			continue
		}
		dx, dy := b.P%g.W.W-g.Cur.WX, b.P/g.W.W-g.Cur.WY
		if d := dx*dx + dy*dy; d < best {
			best, bd = d, b
		}
	}
	if bd == nil {
		return "They shrug."
	}
	dx, dy := bd.P%g.W.W-g.Cur.WX, bd.P/g.W.W-g.Cur.WY
	dir := ""
	if dy < -abs(dx)/2 {
		dir += "north"
	} else if dy > abs(dx)/2 {
		dir += "south"
	}
	if dx > abs(dy)/2 {
		dir += "east"
	} else if dx < -abs(dy)/2 {
		dir += "west"
	}
	return fmt.Sprintf("They point firmly %s and say a name over and over: %s. About %d parasangs.", dir, bd.Name, mini(99, int(sqrtI(best))))
}

func sqrtI(n int) int {
	r := 0
	for (r+1)*(r+1) <= n {
		r++
	}
	return r
}

func (g *Game) lxTeach(a *Actor, cu *CultureData, cost int) {
	l := g.lxLangFor(cu)
	teacher := lxTeacher(a)
	lc := g.lxl()
	k := fmt.Sprintf("%d:%d", a.ID, g.Day())
	lim := 1
	if teacher {
		lim = 2
	}
	if lc.Taught[k] >= lim {
		g.talkMenu("\"Enough for today. Come back tomorrow, my head aches.\"")
		return
	}
	if g.P.Gold < cost {
		g.talkMenu("They hold out a hand and rub two fingers together. You lack the coin.")
		return
	}
	g.P.Gold -= cost
	lc.Taught[k]++
	g.Turn += map[bool]int{true: 60, false: 30}[teacher]
	dc := 9
	if teacher {
		dc = 7
	}
	mod := maxi(g.checkMod("persuasion"), g.P.M(INT))
	old := g.lxStored(cu)
	cp := 60
	if teacher {
		cp = 100
	}
	gain := 1
	good := g.roll2d6Check(mod, dc)
	if good {
		gain = g.R.Int(5, 10)
		if teacher {
			gain = g.R.Int(8, 15)
		}
	}
	g.lxGain(cu, gain, cp, "")
	g.gainXP(3)
	nw := g.lxStored(cu)
	var words []string
	for i := lxKnownN(old); i < lxKnownN(nw) && len(words) < 4; i++ {
		words = append(words, fmt.Sprintf("%s = %s", l.Words[lxConcepts[i]], lxConcepts[i]))
	}
	txt := "You sit and repeat after them until your tongue tires."
	if !good {
		txt = "You stumble over the sounds. They correct you patiently, to little effect."
	}
	if len(words) > 0 {
		txt += "\n\nYou write down: " + strings.Join(words, "; ") + "."
	}
	g.talkMenu(txt)
}

// ---- hearing, meeting, sidebar ----

func lxTurn(g *Game) {
	if g.Cur == nil || g.Cur.Kind != "town" || g.W == nil {
		return
	}
	cu := g.lxCultureAt()
	if cu == nil {
		return
	}
	st := g.lx()
	if cu.Name == st.Origin {
		return
	}
	st.Hear[cu.Name]++
	if st.Hear[cu.Name]%240 == 0 {
		g.lxGain(cu, 1, 35, "")
	}
}

func lxEnter(g *Game, zn *Zone) {
	if zn == nil || zn.Kind != "town" || g.W == nil {
		return
	}
	if cu := g.lxCultureAt(); cu != nil {
		st := g.lx()
		if cu.Name != st.Origin && !st.Met[cu.Name] {
			st.Met[cu.Name] = true
			if l := g.lxLangFor(cu); l != nil {
				g.Msg(fmt.Sprintf("The people here speak the {{Y|%s}}.", l.Name))
			}
		}
	}
}

func lxSide(g *Game) []string {
	if g.Cur == nil || g.Cur.Kind != "town" || g.W == nil {
		return nil
	}
	cu := g.lxCultureAt()
	if cu == nil || cu.Name == g.lx().Origin {
		return nil
	}
	l := g.lxLangFor(cu)
	if l == nil {
		return nil
	}
	return []string{fmt.Sprintf("{{K|Tongue:}} {{c|%s}} %d%%", l.Name, g.lxEff(cu))}
}

// ---- stock: phrasebooks, translators, foreign books ----

func (g *Game) lxPrimer(cu *CultureData) *Item {
	l := g.lxLangFor(cu)
	if l == nil {
		return nil
	}
	noun := lxPrimerNames[g.Genre]
	if noun == "" {
		noun = "phrasebook"
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("A primer of the %s. Family: %s. Word order: %s. Plural: -%s. Polite particle: %s.\n\n", l.Name, lxFam(l), lxOrderText[l.Order], l.Plural, l.Polite))
	for i := 0; i < 20; i++ {
		sb.WriteString(fmt.Sprintf("%s = %s\n", l.Words[lxConcepts[i]], lxConcepts[i]))
	}
	it := g.MakeBook(cu.Name+" "+noun, sb.String())
	it.Extra = map[string]int{"lxp": 1}
	it.Lore = cu.Name
	it.Value = 25
	return it
}

func lxFam(l *lxLang) string {
	if strings.HasPrefix(l.Fam, "~") {
		return "isolate"
	}
	return "the " + l.Fam + " tongues"
}

func (g *Game) lxMakeTranslator(r *Rng) *Item {
	name := lxTranslators[g.Genre]
	if name == "" {
		name = "translator"
	}
	n := 30 + r.Int(0, 20)
	return &Item{ID: g.newID(), Kind: "trophy", Name: name, Glyph: '♦', F: CCyan, Value: 100 + n*4, Weight: 1, Effect: "lx:translator",
		Extra: map[string]int{"lxt": n}, Lore: fmt.Sprintf("While carried it lends you %d%% fluency in any tongue.", n)}
}

func lxZone(g *Game, zn *Zone, _ *Rng) {
	if g.W == nil || zn.Kind != "town" || zn.Z != 0 || zn.Burg <= 0 {
		return
	}
	bd := g.W.burgByID[zn.Burg]
	if bd == nil {
		return
	}
	cu := g.W.cultByID[bd.Culture]
	if cu == nil {
		return
	}
	for _, a := range zn.Actors {
		r := NewRng(fmt.Sprintf("%s:lx:stock:%s:%d", g.W.Seed, zn.Key, a.ID))
		switch a.Role {
		case "bookseller":
			if r.Chance(0.8) {
				if p := g.lxPrimer(cu); p != nil {
					a.Inv = append(a.Inv, p)
				}
			}
			if cu.Name != g.lx().Origin {
				for _, it := range a.Inv {
					if it.Kind == "book" && it.Extra["lxp"] == 0 && r.Chance(0.4) {
						if it.Extra == nil {
							it.Extra = map[string]int{}
						}
						it.Extra["lxc"] = cu.ID + 1
					}
				}
			}
		case "exotic":
			if r.Chance(0.35) {
				a.Inv = append(a.Inv, g.lxMakeTranslator(r))
			}
		}
	}
}

// lxRead handles reading a phrasebook; true when the item was one. Hook: useItem case "book".
func (g *Game) lxRead(it *Item) bool {
	if it == nil || it.Kind != "book" || it.Extra["lxp"] == 0 {
		return false
	}
	st := g.lx()
	cu := g.lxCultureByName(it.Lore)
	txt := it.Text
	if cu != nil {
		last, seen := st.Study[cu.Name]
		if !seen || last != g.Day() {
			gain := 2
			if !seen {
				gain = 8
			}
			st.Study[cu.Name] = g.Day()
			g.Turn += 30
			g.lxGain(cu, gain, 70, "")
		} else {
			txt += "\n\n{{K|You have studied this today already.}}"
		}
	}
	g.showPopup(it.Name, txt)
	return true
}

// ---- abilities ----

func (g *Game) lxAbilities() []Ability {
	return []Ability{{ID: "lx_tongues", Name: "Languages", Target: "self"}}
}

func (g *Game) lxUse(id string, tx, ty int) {
	if id == "lx_tongues" {
		g.lx().Sel = 0
		g.Mode = "lx"
	}
}

// ---- screen ----

func (g *Game) lxRows() []*CultureData {
	st := g.lx()
	var out []*CultureData
	if g.W == nil {
		return out
	}
	for i := range g.W.Cultures {
		c := &g.W.Cultures[i]
		if c.Name == st.Origin || st.Met[c.Name] || st.Flu[c.Name] > 0 {
			out = append(out, c)
		}
	}
	return out
}

func lxKeys(g *Game, k string) {
	st := g.lx()
	rows := g.lxRows()
	switch k {
	case "ArrowUp":
		st.Sel = maxi(0, st.Sel-1)
	case "ArrowDown":
		st.Sel = mini(len(rows)-1, st.Sel+1)
	case "s":
		if st.Sel >= 0 && st.Sel < len(rows) {
			for _, it := range g.P.Inv {
				if it.Extra["lxp"] > 0 && it.Lore == rows[st.Sel].Name {
					g.Mode = "main"
					g.lxRead(it)
					return
				}
			}
			g.Msg("You carry no phrasebook of that tongue.")
		}
	case "Escape", "q":
		if g.Talk != nil {
			g.Mode = "talk"
			g.talkMenu(g.greeting(g.Talk))
			return
		}
		g.Mode = "main"
	}
}

func lxBar(f int) string {
	n := clampi(f/10, 0, 10)
	return "{{G|" + strings.Repeat("█", n) + "}}{{K|" + strings.Repeat("░", 10-n) + "}}"
}

func lxDraw(g *Game) {
	s := g.sc
	st := g.lx()
	x, y, w, h := g.panel("Languages")
	rows := g.lxRows()
	if len(rows) == 0 {
		s.Text(x, y, "{{K|You know no tongue but your own.}}", CGray)
		return
	}
	st.Sel = clampi(st.Sel, 0, len(rows)-1)
	lw := 36
	for i, cu := range rows {
		l := g.lxLangFor(cu)
		bg := CBgDark
		if i == st.Sel {
			bg = CBgSel
		}
		s.Fill(x, y+i, lw, 1, bg)
		nm := l.Name
		if r := []rune(nm); len(r) > 19 {
			nm = string(r[:19])
		}
		s.TextB(x, y+i, fmt.Sprintf("%-19s %s %3d%%", nm, lxBar(g.lxEff(cu)), g.lxEff(cu)), CGray, bg, x+lw)
	}
	cu := rows[st.Sel]
	l := g.lxLangFor(cu)
	rx, rw := x+lw+2, w-lw-2
	ry := y
	line := func(t string) {
		for _, ln := range Wrap(t, rw) {
			if ry < y+h-2 {
				s.TextB(rx, ry, ln, CGray, CBgDark, rx+rw)
				ry++
			}
		}
	}
	stored, eff := g.lxStored(cu), g.lxEff(cu)
	line("{{Y|" + lxCap(l.Name) + "}}  {{K|of " + cu.Name + "}}")
	var kin []string
	for i := range g.W.Cultures {
		c := &g.W.Cultures[i]
		if c.ID != cu.ID && c.Sphere != "" && c.Sphere == cu.Sphere {
			kin = append(kin, c.Name)
		}
	}
	fam := lxFam(l)
	if len(kin) > 0 {
		if len(kin) > 4 {
			kin = kin[:4]
		}
		fam += ", kin of " + strings.Join(kin, ", ")
	}
	line("Family: " + fam)
	extra := ""
	if eff > stored {
		extra = fmt.Sprintf(" (working %d%% with companions or gadget)", eff)
	}
	line(fmt.Sprintf("Learned: %d%%%s", stored, extra))
	line(fmt.Sprintf("Grammar: %s order; plural -%s; polite particle \"%s\".", lxOrderText[l.Order], l.Plural, l.Polite))
	if eff >= 25 {
		line(fmt.Sprintf("Polite thanks: \"%s %s\".", lxCap(l.Words["thanks"]), l.Polite))
	}
	line("Teachers: scholars, priests, elders and booksellers teach fastest for a fee; any local gives a few words for coin. Phrasebooks, signs and a companion of that people help.")
	ry++
	nk := lxKnownN(eff)
	line(fmt.Sprintf("{{c|Dictionary}} {{K|(%d of %d words)}}", nk, len(lxConcepts)))
	colW := maxi(1, rw/2)
	for i := 0; i < nk; i++ {
		cx := rx + (i%2)*colW
		cy := ry + i/2
		if cy >= y+h-3 {
			break
		}
		s.TextB(cx, cy, fmt.Sprintf("{{W|%s}} {{K|=}} %s", l.Words[lxConcepts[i]], lxConcepts[i]), CGray, CBgDark, cx+colW)
	}
	s.Text(x, y+h-2, "{{K|Up/Down: tongue   s: study a phrasebook   Esc: back}}", CGray)
}
