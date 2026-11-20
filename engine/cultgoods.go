package engine

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// Culture pack material culture, literature and language (prefix cg). Everything is optional: a culture
// without a pack adds nothing, and every function tolerates a nil pack. Culture arguments are culture ids
// (CultureData.ID); cpOf wants an index, cgPack converts. State g.ext("cg") (planet-local).
//
// Books: cgWorkBook turns a Literature work into a Kind "book" item (Extra cgw, cgw:c culture+1, cgw:i
// work index+1, gl:ev event+1, lxc culture+1); reading runs through cgRead (mode "cgr"), shares the gl shelf
// and claim store (glSt().Claims) so a pack work and a generated book on the same event can contradict.
// Food: cgDishes / cgDishItem / cgStock (dishes, drinks, trade goods, craft goods). Item.Extra "unclean" is a
// bitmask of culture ids < 62 that hold the food unclean, "unclean:<id>" = 1 names every such culture.
// Language: cgPhrase, cgNumeral, cgShout, lexicon patched into lxLang.Words, phrases page (mode "cgl").

type cgState struct {
	Heard  map[string]int  `json:"h,omitempty"` // "culture:use:say" → times heard
	Verse  map[string]bool `json:"v,omitempty"` // "culture:form" → verse heard
	Read   map[string]bool `json:"r,omitempty"` // "culture:work index" → finished
	Bought map[string]bool `json:"b,omitempty"`
	Day    map[string]int  `json:"d,omitempty"` // "actor:what" → day last used

	View   *Item    `json:"-"`
	Lines  []string `json:"-"`
	Scroll int      `json:"-"`
	Minute int      `json:"-"`
}

func (g *Game) cgSt() *cgState {
	st := g.ext("cg", func() any { return &cgState{} }).(*cgState)
	if st.Heard == nil {
		st.Heard = map[string]int{}
	}
	if st.Verse == nil {
		st.Verse = map[string]bool{}
	}
	if st.Read == nil {
		st.Read = map[string]bool{}
	}
	if st.Bought == nil {
		st.Bought = map[string]bool{}
	}
	if st.Day == nil {
		st.Day = map[string]int{}
	}
	return st
}

var cgLexDone = map[string]bool{}
var cgWrapped bool

func init() {
	unLocalKeys = append(unLocalKeys, "cg")
	modeKeys["cgr"] = cgReadKeys
	modeDraws["cgr"] = cgReadDraw
	modeKeys["cgl"] = cgLangKeys
	modeDraws["cgl"] = cgLangDraw
	talkHooks = append(talkHooks, cgTalk)
	enterHooks = append(enterHooks, cgEnter)
}

// ---- culture helpers ----

func cgIdx(w *WorldData, id int) int {
	if w == nil {
		return -1
	}
	for i := range w.Cultures {
		if w.Cultures[i].ID == id {
			return i
		}
	}
	return -1
}

func cgPack(w *WorldData, id int) *CulturePack {
	i := cgIdx(w, id)
	if i < 0 {
		return nil
	}
	return w.cpOf(i)
}

func (g *Game) cgHere() (*CultureData, *CulturePack) {
	if g == nil || g.W == nil || g.Cur == nil {
		return nil, nil
	}
	cu := g.lxCultureAt()
	if cu == nil {
		return nil, nil
	}
	return cu, cgPack(g.W, cu.ID)
}

func cgOK(g *Game, a *Actor) bool {
	return a != nil && g != nil && g.W != nil && g.Cur != nil && a.AI != "follower" && !g.Hostile(a)
}

// ---- books ----

func cgWorks(w *WorldData, culture int) []Work {
	if p := cgPack(w, culture); p != nil && p.Literature != nil {
		return p.Literature.Works
	}
	return nil
}

// cgWorkBook makes a book of one of the culture's works (chosen by a stable hash of key), nil when the
// culture has none.
func (g *Game) cgWorkBook(culture int, key string) *Item {
	ws := cgWorks(g.W, culture)
	if len(ws) == 0 {
		return nil
	}
	return g.cgWorkItem(culture, g.W.cpIdx(len(ws), "cgwork", strconv.Itoa(culture), key))
}

func (g *Game) cgWorkItem(culture, i int) *Item {
	ws := cgWorks(g.W, culture)
	if i < 0 || i >= len(ws) {
		return nil
	}
	wk := ws[i]
	cu := g.W.cultByID[culture]
	var L []string
	L = append(L, "{{Y|"+wk.Title+"}}")
	by := wk.Form
	if wk.Author != "" {
		by += " by " + wk.Author
	}
	if wk.Year != 0 {
		by += ", year " + strconv.Itoa(wk.Year)
	}
	if cu != nil {
		by += " (" + cu.Name + ")"
	}
	L = append(L, "{{K|"+by+"}}", "")
	for _, ln := range wk.Lines {
		L = append(L, ln)
	}
	text := strings.Join(L, "\n")
	it := g.MakeBook(wk.Title, text)
	it.Lore = wk.Form + " by " + wk.Author
	it.Value = clampi(14+len(strings.Fields(text))/6, 14, 80)
	it.Extra = map[string]int{"cgw": 1, "cgw:c": culture + 1, "cgw:i": i + 1, "lxc": culture + 1,
		"gl:min": clampi(len(strings.Fields(text))/18, 3, 30)}
	if wk.Event != nil && *wk.Event >= 0 && *wk.Event < len(g.W.Events) {
		it.Extra["gl:ev"] = *wk.Event + 1
	}
	return it
}

// cgPickBook is the hook for GenBook: about 60% of the books of a culture with works are one of its works.
func (g *Game) cgPickBook(r *Rng, p int) *Item {
	if g == nil || g.W == nil || r == nil || p < 0 || p >= len(g.W.Culture) {
		return nil
	}
	cid := g.W.Culture[p]
	if len(cgWorks(g.W, cid)) == 0 || r.Int(0, 99) >= 60 {
		return nil
	}
	return g.cgWorkBook(cid, strconv.Itoa(r.Int(1, 1<<30)))
}

func (g *Game) cgWorkOf(it *Item) (Work, int, bool) {
	if it == nil || it.Extra["cgw"] == 0 || g.W == nil {
		return Work{}, 0, false
	}
	c := it.Extra["cgw:c"] - 1
	ws := cgWorks(g.W, c)
	i := it.Extra["cgw:i"] - 1
	if i < 0 || i >= len(ws) {
		return Work{}, 0, false
	}
	return ws[i], c, true
}

// cgRead opens the reading screen for a pack work (call from useItem case "book", before glRead).
func (g *Game) cgRead(it *Item) bool {
	wk, c, ok := g.cgWorkOf(it)
	if !ok {
		return false
	}
	st := g.cgSt()
	mins := 2
	if !st.Read[fmt.Sprintf("%d:%d", c, it.Extra["cgw:i"])] {
		mins = maxi(3, it.Extra["gl:min"]-g.Skills["lore"]*2)
	}
	g.Turn += mins
	st.Minute = mins
	st.View, st.Scroll = it, 0
	st.Lines = Wrap(g.lxBookText(it), MW-14)
	if g.Skills["lore"] >= 2 && wk.Bias != "" {
		st.Lines = append(st.Lines, "", "{{K|(Your learning tells you this author leans "+wk.Bias+".)}}")
	}
	g.Mode = "cgr"
	if len(st.Lines) <= glVis {
		g.cgFinish(it)
	}
	return true
}

func (g *Game) cgFinish(it *Item) {
	wk, c, ok := g.cgWorkOf(it)
	if !ok {
		return
	}
	st := g.cgSt()
	key := fmt.Sprintf("%d:%d", c, it.Extra["cgw:i"])
	if st.Read[key] {
		return
	}
	st.Read[key] = true
	g.Read["book:"+it.Name] = true
	g.gainXP(10 + g.Skills["lore"]*5)
	gs := g.glSt()
	if len(gs.Shelf) < 80 {
		gs.Shelf = append(gs.Shelf, glShelfRow{wk.Title, wk.Author, wk.Form, wk.Bias})
	}
	p, yr := -1, wk.Year
	if wk.Event != nil && *wk.Event >= 0 && *wk.Event < len(g.W.Events) {
		e := g.W.Events[*wk.Event]
		p, yr = e.P, e.Y
		// the same claim store the generated books use: two works on one event can contradict
		ek := "e" + strconv.Itoa(*wk.Event)
		cl := glClaim{Title: wk.Title, Author: wk.Author, Bias: wk.Bias, Year: e.Y}
		for _, o := range gs.Claims[ek] {
			g.glCompare(o, cl)
			g.cgSlant(o, cl, p)
		}
		if len(gs.Claims[ek]) < 6 {
			gs.Claims[ek] = append(gs.Claims[ek], cl)
		}
	}
	if wk.Reveals != "" && g.lxEff(g.W.cultByID[c]) >= 50 {
		if g.addSecret(fmt.Sprintf("%s (per \"%s\")", wk.Reveals, wk.Title), "history", p, yr) {
			g.Msg("{{W|(Recorded in your journal.)}}")
		}
	}
	g.Msg("{{c|You finish \"" + it.Name + "\".}}")
}

// cgSlant notes two books on one event that lean opposite ways.
func (g *Game) cgSlant(a, b glClaim, p int) {
	if a.Title == b.Title {
		return
	}
	sa, sb := cgBiasSlant[a.Bias], cgBiasSlant[b.Bias]
	if sa*sb >= 0 { // only a flatterer against a cynic
		return
	}
	line := fmt.Sprintf("Contradiction between two books on the same event: \"%s\" (%s, by %s) tells it as the other \"%s\" (%s, by %s) does not. One of them bent the telling.",
		a.Title, a.Bias, a.Author, b.Title, b.Bias, b.Author)
	if g.addSecret(line, "contradiction", p, a.Year) {
		g.Msg("{{Y|Contradiction noted: the " + a.Bias + " \"" + a.Title + "\" and the " + b.Bias + " \"" + b.Title + "\" do not tell it alike.}}")
	}
}

func cgReadClose(g *Game) {
	st := g.cgSt()
	st.View, st.Lines = nil, nil
	if g.Talk != nil {
		g.Mode = "talk"
	} else if g.ItemSel != nil {
		g.Mode = "inv"
	} else {
		g.Mode = "main"
	}
	g.ItemSel = nil
}

func cgReadKeys(g *Game, k string) {
	st := g.cgSt()
	max := maxi(0, len(st.Lines)-glVis)
	switch k {
	case "ArrowDown":
		st.Scroll++
	case "ArrowUp":
		st.Scroll--
	case "PageDown", " ":
		st.Scroll += glVis - 2
	case "PageUp":
		st.Scroll -= glVis - 2
	case "Home":
		st.Scroll = 0
	case "End":
		st.Scroll = max
	case "Escape", "q", "Enter":
		cgReadClose(g)
		return
	}
	st.Scroll = clampi(st.Scroll, 0, max)
	if st.Scroll >= max && st.View != nil {
		g.cgFinish(st.View)
	}
}

func cgReadDraw(g *Game) {
	s := g.sc
	st := g.cgSt()
	x, y, w, h := g.panel("Reading")
	if st.View == nil {
		return
	}
	s.TextB(x, y, fmt.Sprintf("{{K|Reading took about %d minutes. Line %d of %d.}}", st.Minute, mini(st.Scroll+glVis, len(st.Lines)), len(st.Lines)), CGray, CBgDark, x+w)
	for i := 0; i < glVis && st.Scroll+i < len(st.Lines); i++ {
		s.TextB(x, y+2+i, st.Lines[st.Scroll+i], CGray, CBgDark, x+w)
	}
	s.Text(x, y+h-1, "{{K|Up/Down/PgUp/PgDn: scroll   Esc: close}}", CGray)
}

// ---- food ----

// cgDishes: the culture's cuisine dishes (nil without a pack).
func (g *Game) cgDishes(culture int) []CpDish {
	p := cgPack(g.W, culture)
	if p == nil || p.Material == nil || p.Material.Cuisine == nil {
		return nil
	}
	return p.Material.Cuisine.Dishes
}

func cgTagOf(d CpDish) string {
	if _, ok := cookTags[d.Tag]; ok {
		return d.Tag
	}
	for _, w := range d.With {
		if t := stapleTag(w); t != "" {
			return t
		}
	}
	return "hearty"
}

func cgHasWord(hay []string, d CpDish) bool {
	for _, u := range hay {
		u = strings.ToLower(strings.TrimSpace(u))
		if u == "" {
			continue
		}
		if strings.Contains(strings.ToLower(d.Name), u) {
			return true
		}
		for _, w := range d.With {
			if strings.ToLower(w) == u {
				return true
			}
		}
	}
	return false
}

// cgDishItem makes a ready-cooked dish item of the culture (feast dishes carry their feast tag twice).
func (g *Game) cgDishItem(culture int, d CpDish, n int) *Item {
	tag := cgTagOf(d)
	eff := tag
	val := 16
	if d.Feast != "" {
		eff = tag + "+" + tag
		val = 24
	}
	it := &Item{ID: g.newID(), Kind: "dish", Name: d.Name, Effect: eff, Glyph: '%', F: CYellow, Count: n, Value: val, Power: 500, Weight: 1,
		Extra: map[string]int{"cg": 1, "cgd:c": culture + 1}}
	lore := d.Note
	if len(d.With) > 0 {
		lore = strings.TrimSpace(lore + " Made with " + strings.Join(d.With, ", ") + ".")
	}
	var mask int
	var names []string
	for i := range g.W.Cultures {
		c := &g.W.Cultures[i]
		p := cgPack(g.W, c.ID)
		if p == nil || p.Material == nil || p.Material.Cuisine == nil || !cgHasWord(p.Material.Cuisine.Unclean, d) {
			continue
		}
		it.Extra["unclean:"+strconv.Itoa(c.ID)] = 1
		if c.ID >= 0 && c.ID < 62 {
			mask |= 1 << uint(c.ID)
		}
		names = append(names, c.Name)
	}
	if mask != 0 {
		it.Extra["unclean"] = mask
	} else if len(names) > 0 {
		it.Extra["unclean"] = 1 << 62
	}
	if len(names) > 0 {
		sort.Strings(names)
		lore = strings.TrimSpace(lore + " Unclean food to the " + strings.Join(names, ", the ") + ".")
	}
	it.Lore = lore
	return it
}

// cgUncleanFor: does the item count as unclean to this culture (see cgDishItem)?
func cgUncleanFor(it *Item, culture int) bool {
	return it != nil && it.Extra != nil && it.Extra["unclean:"+strconv.Itoa(culture)] > 0
}

func cgDrinkItem(g *Game, culture int, drink string) *Item {
	liq := "fresh"
	ld := strings.ToLower(drink)
	for _, m := range cgDrinkLiquid {
		if strings.Contains(ld, m.Word) {
			liq = m.Liquid
			break
		}
	}
	it := &Item{ID: g.newID(), Kind: "water", Name: drink + " flask", Glyph: 'δ', F: CYellow, Power: 16, Value: 9,
		Lore: "A local drink: " + drink + ".", Extra: map[string]int{"cg": 1, "cgd:c": culture + 1}}
	g.lqSet(it, liq, 6)
	return it
}

// ---- trade and craft goods ----

// cgGoods: the culture's local trade goods as Kind "trade" items (Value from the pack).
func (g *Game) cgGoods(culture int) []*Item {
	p := cgPack(g.W, culture)
	if p == nil || p.Material == nil {
		return nil
	}
	cu := g.W.cultByID[culture]
	var out []*Item
	for _, tg := range p.Material.TradeGoods {
		if tg.Name == "" {
			continue
		}
		wt := cgGoodWeight[tg.Kind]
		if wt == 0 {
			wt = 2
		}
		it := &Item{ID: g.newID(), Kind: "trade", Name: tg.Name, Glyph: '%', F: CWhite, Value: maxi(1, tg.Value), Weight: wt, Count: 1,
			Extra: map[string]int{"cgg": 1, "cgg:c": culture + 1}}
		it.Lore = "A " + tg.Kind + " the " + map[bool]string{true: cu.Name, false: "local"}[cu != nil] + " trade in."
		out = append(out, it)
	}
	return out
}

// cgCraftGoods: goods of the culture's crafts, each described by the craft's mark.
func (g *Game) cgCraftGoods(culture int) []*Item {
	p := cgPack(g.W, culture)
	if p == nil || p.Material == nil {
		return nil
	}
	var out []*Item
	for _, cr := range p.Material.Crafts {
		for _, gd := range cr.Goods {
			h := int(HashStr(g.W.Seed+":cgcraft:"+cr.Name+":"+gd) % 31)
			it := &Item{ID: g.newID(), Kind: "trade", Name: gd, Glyph: '%', F: CCyan, Value: 10 + h, Weight: 2, Count: 1,
				Extra: map[string]int{"cgg": 1, "cgg:c": culture + 1, "cgc": 1}}
			it.Lore = "From the " + strings.ToLower(cr.Name) + " trade."
			if cr.Mark != "" {
				it.Lore += " Its mark: " + cr.Mark + "."
			}
			out = append(out, it)
		}
	}
	return out
}

func cgMatchFeast(f *FeastData, key string) bool {
	if f == nil || key == "" {
		return false
	}
	k := strings.ToLower(key)
	return strings.ToLower(f.Kind) == k || strings.Contains(strings.ToLower(f.Name), k)
}

// cgStock returns the pack items a vendor should carry now, by role: merchants (dishes, a drink, three
// trade goods), innkeepers (dishes, drinks), smiths/tailors/artisans (craft goods). Each item carries
// Extra "cgk" (identity), "cgS" (season slot + 1, rotates) and for feast dishes "cgF" (last day + 1).
// A feast dish appears only in its feast window. Nil without a pack. cgApplyStock adds them once.
func cgStock(g *Game, a *Actor) []*Item {
	if g == nil || g.W == nil || a == nil || a.AI == "follower" {
		return nil
	}
	cu, p := g.cgHere()
	if cu == nil || p == nil || p.Material == nil {
		return nil
	}
	slot := g.Day() / SeasonDays
	w := g.W
	mk := func(it *Item, id string, feastEnd int) *Item {
		if it == nil {
			return nil
		}
		it.Extra["cgk"] = int(HashStr(id)&0x3fffffff) + 1
		it.Extra["cgS"] = slot + 1
		if feastEnd > 0 {
			it.Extra["cgF"] = feastEnd + 1
		}
		return it
	}
	var out []*Item
	add := func(it *Item) {
		if it != nil {
			out = append(out, it)
		}
	}
	if cgDishRoles[a.Role] {
		dishes := g.cgDishes(cu.ID)
		var plain []CpDish
		for _, d := range dishes {
			if d.Feast == "" {
				plain = append(plain, d)
			}
		}
		for i := 0; i < 2 && len(plain) > 0; i++ {
			d := plain[w.cpIdx(len(plain), "cgdish", strconv.Itoa(a.ID), strconv.Itoa(slot), strconv.Itoa(i))]
			add(mk(g.cgDishItem(cu.ID, d, 1+i), fmt.Sprintf("dish:%d:%s:%d", cu.ID, d.Name, slot), 0))
		}
		if _, f, fday := g.snFeastWindow(); f != nil {
			for _, d := range dishes {
				if cgMatchFeast(f, d.Feast) {
					add(mk(g.cgDishItem(cu.ID, d, 2), fmt.Sprintf("feast:%d:%s:%d", cu.ID, d.Name, fday), fday))
				}
			}
		}
		if p.Material.Cuisine != nil && len(p.Material.Cuisine.Drinks) > 0 {
			dr := p.Material.Cuisine.Drinks
			d := dr[w.cpIdx(len(dr), "cgdrink", strconv.Itoa(a.ID), strconv.Itoa(slot))]
			add(mk(cgDrinkItem(g, cu.ID, d), fmt.Sprintf("drink:%d:%s:%d", cu.ID, d, slot), 0))
		}
	}
	if cgGoodsRoles[a.Role] {
		gs := g.cgGoods(cu.ID)
		seen := map[int]bool{}
		for i := 0; i < 3 && len(gs) > 0 && len(seen) < len(gs); i++ {
			k := w.cpIdx(len(gs), "cggood", strconv.Itoa(a.ID), strconv.Itoa(slot), strconv.Itoa(i))
			if seen[k] {
				continue
			}
			seen[k] = true
			it := gs[k]
			it.Count = 1 + w.cpIdx(3, "cggoodn", strconv.Itoa(a.ID), it.Name)
			add(mk(it, fmt.Sprintf("good:%d:%s:%d", cu.ID, it.Name, slot), 0))
		}
	}
	if cgCraftRoles[a.Role] {
		gs := g.cgCraftGoods(cu.ID)
		seen := map[int]bool{}
		for i := 0; i < 3 && len(gs) > 0 && len(seen) < len(gs); i++ {
			k := w.cpIdx(len(gs), "cgcraft", strconv.Itoa(a.ID), strconv.Itoa(slot), strconv.Itoa(i))
			if seen[k] {
				continue
			}
			seen[k] = true
			it := gs[k]
			it.Count = 1 + w.cpIdx(2, "cgcraftn", strconv.Itoa(a.ID), it.Name)
			add(mk(it, fmt.Sprintf("craft:%d:%s:%d", cu.ID, it.Name, slot), 0))
		}
	}
	return out
}

// cgApplyStock puts cgStock on the vendor once per item (an actor flag remembers it, so sold goods stay
// sold) and removes pack items of an earlier season or past feast.
func (g *Game) cgApplyStock(a *Actor) {
	if a == nil || (!cgDishRoles[a.Role] && !cgGoodsRoles[a.Role] && !cgCraftRoles[a.Role]) {
		return
	}
	day, slot := g.Day(), g.Day()/SeasonDays
	keep := a.Inv[:0]
	for _, it := range a.Inv {
		if it.Extra != nil && it.Extra["cgk"] > 0 {
			if s := it.Extra["cgS"]; s > 0 && s != slot+1 {
				continue
			}
			if f := it.Extra["cgF"]; f > 0 && day > f-1 {
				continue
			}
		}
		keep = append(keep, it)
	}
	a.Inv = keep
	for _, it := range cgStock(g, a) {
		fl := "cgk:" + strconv.Itoa(it.Extra["cgk"])
		if a.Has(fl) {
			continue
		}
		a.Set(fl)
		a.Inv = append(a.Inv, it)
	}
}

// ---- talk: verses, shelf, greetings, stocking ----

func cgTalk(g *Game, a *Actor) []Option {
	if !cgOK(g, a) {
		return nil
	}
	g.cgApplyStock(a) // vendors get their local wares before the trade window opens
	cu, p := g.cgHere()
	if cu == nil || p == nil {
		return nil
	}
	var out []Option
	if cgVerseRoles[a.Role] && p.Literature != nil && len(p.Literature.Forms) > 0 {
		out = append(out, Option{"Ask for a verse.", func() { g.cgVerse(a) }})
	}
	if a.Role == "bookseller" && len(cgWorks(g.W, cu.ID)) > 0 {
		out = append(out, Option{"Browse the local writers.", func() { g.cgShelf(a) }})
	}
	if p.Lang != nil && len(cgPhrases(p, "greet")) > 0 && a.Role != "guard" {
		out = append(out, Option{"Greet them in their own tongue.", func() { g.cgGreet(a) }})
	}
	return out
}

func (g *Game) cgVerse(a *Actor) {
	cu, p := g.cgHere()
	if cu == nil || p == nil || p.Literature == nil || len(p.Literature.Forms) == 0 {
		g.talkMenu("They shake their head. \"Nothing comes to mind.\"")
		return
	}
	st := g.cgSt()
	day := g.Day()
	forms := p.Literature.Forms
	idx := g.W.cpIdx(len(forms), "cgverse", strconv.Itoa(a.ID), strconv.Itoa(day/3))
	f := forms[idx]
	vars := map[string]string{"name": g.the(a), "culture": cu.Name}
	intro := cpFill(g.W.cpChoose(cgVerseIntro, "cgvi", strconv.Itoa(a.ID), strconv.Itoa(day)), vars)
	lines := []string{intro, "", "{{Y|" + f.Name + "}}"}
	for _, ln := range f.Example {
		lines = append(lines, "  "+ln)
	}
	if f.Rule != "" {
		lines = append(lines, "", "{{K|("+f.Rule+")}}")
	}
	lines = append(lines, "", g.W.cpChoose(cgVerseThanks, "cgvt", strconv.Itoa(a.ID), strconv.Itoa(day)))
	dk := fmt.Sprintf("%d:verse", a.ID)
	vk := fmt.Sprintf("%d:%s", cu.ID, f.Name)
	if st.Day[dk] != day+1 {
		st.Day[dk] = day + 1
		if !st.Verse[vk] {
			st.Verse[vk] = true
			g.gainXP(5)
			g.AddRep(a.Faction, 1, " for listening to a verse")
			if a.Role == "bard" {
				lines = append(lines, "{{G|They press a small coin on you for a good ear: "+g.cuEarn(1+g.P.Level/3)+".}}")
			}
			lines = append(lines, "{{W|You could play this at a crowd now.}}")
		}
	}
	g.talkMenu(strings.Join(lines, "\n"))
}

// cgBuskTip is a bonus to busking tips in a culture whose verses you have heard (games.go gmBusk mode 0).
func (g *Game) cgBuskTip() int {
	cu, p := g.cgHere()
	if cu == nil || p == nil || p.Literature == nil {
		return 0
	}
	n := 0
	for _, f := range p.Literature.Forms {
		if g.cgSt().Verse[fmt.Sprintf("%d:%s", cu.ID, f.Name)] {
			n++
		}
	}
	return mini(n, 4)
}

func (g *Game) cgShelf(a *Actor) {
	cu, _ := g.cgHere()
	if cu == nil {
		return
	}
	ws := cgWorks(g.W, cu.ID)
	if len(ws) == 0 {
		g.talkMenu("\"Nothing local on the shelf just now.\"")
		return
	}
	st := g.cgSt()
	slot := g.Day() / 20
	var opts []Option
	lines := []string{"\"Local hands, all of them,\" the bookseller says, tapping the spines."}
	seen := map[int]bool{}
	for i := 0; i < 3 && len(seen) < len(ws); i++ {
		k := g.W.cpIdx(len(ws), "cgshelf", strconv.Itoa(a.ID), strconv.Itoa(slot), strconv.Itoa(i))
		if seen[k] {
			continue
		}
		seen[k] = true
		bk := fmt.Sprintf("%d:%d:%d:%d", a.ID, slot, cu.ID, k)
		if st.Bought[bk] {
			continue
		}
		it := g.cgWorkItem(cu.ID, k)
		if it == nil {
			continue
		}
		wk := ws[k]
		line := fmt.Sprintf("\"%s\": a %s by %s.", it.Name, wk.Form, wk.Author)
		if g.Skills["lore"] >= 2 && wk.Bias != "" {
			line += " (" + wk.Bias + ")"
		}
		price := it.Value
		line += " Asking " + g.cuFormat(price) + "."
		lines = append(lines, line)
		itc, kk, pr := it, bk, price
		opts = append(opts, Option{"Buy \"" + it.Name + "\".", func() {
			if !g.cuSpend(pr) {
				g.talkMenu("\"Come back with the coin.\"")
				return
			}
			st.Bought[kk] = true
			g.give(itc)
			g.talkMenu("They wrap it carefully and hand it over.")
		}})
	}
	if len(opts) == 0 {
		g.talkMenu("\"You have bought what I had. Ask again when I have restocked.\"")
		return
	}
	g.talkMenu(strings.Join(lines, "\n\n"))
	opts = append(opts, Option{"Never mind.", func() { g.talkMenu(g.greeting(a)) }})
	g.Opts = opts
}

// ---- language ----

func cgPhrases(p *CulturePack, use string) []Phrase {
	if p == nil || p.Lang == nil {
		return nil
	}
	var out []Phrase
	for _, ph := range p.Lang.Phrases {
		if ph.Use == use && ph.Say != "" {
			out = append(out, ph)
		}
	}
	return out
}

func (g *Game) cgPhraseKnown(cu *CultureData, ph Phrase, i int) bool {
	if cu == nil {
		return false
	}
	f := g.lxEff(cu)
	if f >= 100 {
		return true
	}
	if g.cgSt().Heard[fmt.Sprintf("%d:%s", cu.ID, ph.Say)] >= 2 {
		return true
	}
	return f >= 30+(i%6)*10
}

// cgPhrase gives a greeting (or any use: greet toast thanks farewell curse oath blessing) in the culture's
// own tongue. The gloss is empty until you know it: you have heard it twice, or your fluency is enough.
// Each call counts as hearing the phrase once per game day. "", "" without a pack.
func (g *Game) cgPhrase(culture int, use string) (say, gloss string) {
	if g == nil || g.W == nil {
		return "", ""
	}
	p := cgPack(g.W, culture)
	list := cgPhrases(p, use)
	if len(list) == 0 {
		return "", ""
	}
	cu := g.W.cultByID[culture]
	i := g.W.cpIdx(len(list), "cgphrase", strconv.Itoa(culture), use, strconv.Itoa(g.Day()/7))
	ph := list[i]
	st := g.cgSt()
	hk := fmt.Sprintf("%d:%s", culture, ph.Say)
	dk := "heard:" + hk
	if st.Day[dk] != g.Day()+1 {
		st.Day[dk] = g.Day() + 1
		st.Heard[hk]++
	}
	if g.cgPhraseKnown(cu, ph, i) {
		return ph.Say, ph.Gloss
	}
	return ph.Say, ""
}

func (g *Game) cgGreet(a *Actor) {
	cu, _ := g.cgHere()
	if cu == nil {
		return
	}
	mine, mg := g.cgPhrase(cu.ID, "greet")
	if mine == "" {
		g.talkMenu("You cannot recall how they greet.")
		return
	}
	text := fmt.Sprintf("You greet them: \"%s.\"", mine)
	if mg != "" {
		text += " {{K|(" + mg + ")}}"
	}
	st := g.cgSt()
	dk := fmt.Sprintf("%d:greet", a.ID)
	if st.Day[dk] != g.Day()+1 {
		st.Day[dk] = g.Day() + 1
		g.lxGain(cu, 2, 30, "")
		g.AddRep(a.Faction, 1, " for greeting them in their own tongue")
		text += "\n\nThey smile at hearing it from a stranger."
	}
	if r, rg := g.cgPhrase(cu.ID, "thanks"); r != "" {
		text += fmt.Sprintf("\n\nThey answer: \"%s.\"", r)
		if rg != "" {
			text += " {{K|(" + rg + ")}}"
		}
	}
	g.talkMenu(text)
	g.Popup = Wrap(text, 100) // already in their tongue; the fluency filter would garble it twice
}

// cgNumeral writes n (1..99) in the culture's numerals; "" when the pack has none or n is out of range.
func (g *Game) cgNumeral(culture, n int) string {
	p := cgPack(g.W, culture)
	if p == nil || p.Lang == nil || len(p.Lang.Numerals) < 10 || n < 1 || n > 99 {
		return ""
	}
	nu := p.Lang.Numerals
	if n <= 10 {
		return nu[n-1]
	}
	t, u := n/10, n%10
	parts := []string{}
	if t > 1 {
		parts = append(parts, nu[t-1])
	}
	parts = append(parts, nu[9])
	if u > 0 {
		parts = append(parts, nu[u-1])
	}
	return strings.Join(parts, " ")
}

// cgShout is a vendor's price call in the local numerals ("" without numerals or above 99). The figure
// in digits follows once your fluency is 30%.
func (g *Game) cgShout(a *Actor, price int) string {
	cu, p := g.cgHere()
	if cu == nil || p == nil || a == nil {
		return ""
	}
	s := g.cgNumeral(cu.ID, price)
	if s == "" {
		return ""
	}
	s = lxCap(s)
	if g.lxEff(cu) >= 30 {
		return fmt.Sprintf("\"%s!\" {{K|(%d)}}", s, price)
	}
	return fmt.Sprintf("\"%s!\"", s)
}

// cgLexicon puts the pack's lexicon over the generated words (lexicon glosses that name a concept of
// languages.go). Idempotent; cgEnter runs it for every culture once per world.
func (g *Game) cgLexicon(cu *CultureData) {
	if cu == nil || g.W == nil {
		return
	}
	p := cgPack(g.W, cu.ID)
	if p == nil || p.Lang == nil || len(p.Lang.Lexicon) == 0 {
		return
	}
	l := g.lxLangFor(cu)
	if l == nil {
		return
	}
	for _, e := range p.Lang.Lexicon {
		gl := strings.ToLower(strings.TrimSpace(e[1]))
		if _, ok := lxIdx[gl]; ok && e[0] != "" {
			l.Words[gl] = e[0]
		}
	}
}

func cgEnter(g *Game, zn *Zone) {
	if g.W == nil {
		return
	}
	if !cgWrapped {
		cgWrapped = true
		prev := modeKeys["lx"]
		if prev != nil {
			modeKeys["lx"] = func(g *Game, k string) {
				if k == "p" && g.W != nil {
					g.Mode = "cgl"
					return
				}
				prev(g, k)
			}
		}
	}
	key := fmt.Sprintf("%s|%d", g.W.Seed, g.W.uniIdx)
	if cgLexDone[key] {
		return
	}
	cgLexDone[key] = true
	for i := range g.W.Cultures {
		g.cgLexicon(&g.W.Cultures[i])
	}
}

// cgLangLines: what you know of a culture's pack language, for a page: phrases, words, numerals.
func (g *Game) cgLangLines(cu *CultureData) []string {
	p := cgPack(g.W, cu.ID)
	if p == nil || p.Lang == nil {
		return nil
	}
	eff := g.lxEff(cu)
	var out []string
	var ph []string
	for i, x := range p.Lang.Phrases {
		if g.cgPhraseKnown(cu, x, i) {
			ph = append(ph, fmt.Sprintf("{{W|%s}} {{K|(%s: %s)}}", x.Say, x.Use, x.Gloss))
		}
	}
	if len(ph) > 0 {
		out = append(out, "{{c|Phrases}} {{K|("+strconv.Itoa(len(ph))+" of "+strconv.Itoa(len(p.Lang.Phrases))+")}}")
		out = append(out, ph...)
	}
	if n := len(p.Lang.Lexicon); n > 0 {
		k := lxKnownN(eff) * n / len(lxConcepts)
		if eff >= 100 {
			k = n
		}
		if k > 0 {
			out = append(out, "", "{{c|Words}} {{K|("+strconv.Itoa(k)+" of "+strconv.Itoa(n)+")}}")
			for _, e := range p.Lang.Lexicon[:k] {
				out = append(out, fmt.Sprintf("{{W|%s}} {{K|=}} %s", e[0], e[1]))
			}
		}
	}
	if len(p.Lang.Numerals) > 0 && eff >= 30 {
		out = append(out, "", "{{c|Numerals}} {{K|"+strings.Join(p.Lang.Numerals, " ")+"}}")
	}
	return out
}

func cgLangKeys(g *Game, k string) {
	st := g.lx()
	rows := g.lxRows()
	switch k {
	case "ArrowUp":
		st.Sel = maxi(0, st.Sel-1)
		g.cgSt().Scroll = 0
	case "ArrowDown":
		st.Sel = mini(len(rows)-1, st.Sel+1)
		g.cgSt().Scroll = 0
	case "PageDown":
		g.cgSt().Scroll += 10
	case "PageUp":
		g.cgSt().Scroll = maxi(0, g.cgSt().Scroll-10)
	case "Escape", "q", "p":
		g.Mode = "lx"
	}
}

func cgLangDraw(g *Game) {
	s := g.sc
	st := g.lx()
	cs := g.cgSt()
	x, y, w, h := g.panel("Phrases and words")
	rows := g.lxRows()
	if len(rows) == 0 {
		s.Text(x, y, "{{K|You know no tongue but your own.}}", CGray)
		return
	}
	st.Sel = clampi(st.Sel, 0, len(rows)-1)
	cu := rows[st.Sel]
	l := g.lxLangFor(cu)
	name := cu.Name
	if l != nil {
		name = l.Name
	}
	s.TextB(x, y, "{{Y|"+lxCap(name)+"}} {{K|of "+cu.Name+"}}", CGray, CBgDark, x+w)
	lines := g.cgLangLines(cu)
	if len(lines) == 0 {
		s.TextB(x, y+2, "{{K|Nothing of this tongue is written down yet. Greet its speakers, hear their toasts, buy a phrasebook.}}", CGray, CBgDark, x+w)
	}
	vis := h - 4
	cs.Scroll = clampi(cs.Scroll, 0, maxi(0, len(lines)-vis))
	for i := 0; i < vis && cs.Scroll+i < len(lines); i++ {
		s.TextB(x, y+2+i, lines[cs.Scroll+i], CGray, CBgDark, x+w)
	}
	s.Text(x, y+h-1, "{{K|Up/Down: tongue   PgUp/PgDn: scroll   Esc: back}}", CGray)
}
