package engine

// Gear that grows (prefix gg): heirlooms that level from deeds, item provenance with "look closer",
// and reforging / renaming at a forge. Saved state g.ext("gg"); per-item data in Item.Extra:
//   ggxp ggkills gglv ggfx ggab, ggk:<tag> ggt:<noun> (history counters), ggo ggfp ggw (origin stamps),
//   ggl ggla (look closer), ggrf (reforged). Never draws g.R for generation: side rngs ggRng.

import (
	"fmt"
	"sort"
	"strings"
)

func init() {
	killHooks = append(killHooks, ggKill)
	turnHooks = append(turnHooks, ggTurn)
	enterHooks = append(enterHooks, ggEnter)
	modeKeys["ggf"] = ggfKeys
	modeDraws["ggf"] = ggfDraw
	talkHooks = append(talkHooks, func(g *Game, a *Actor) []Option {
		if a.Role != "smith" {
			return nil
		}
		return []Option{{"{{O|Reforge or rename gear.}}", func() { g.ggOpen(a.ID, "") }}}
	})
}

type ggRec struct {
	Title string   `json:"t"`
	Lore0 string   `json:"l0,omitempty"`
	Hist  []string `json:"h,omitempty"`
	RenLv int      `json:"rl,omitempty"`
}

type ggUI struct {
	stage, sel int
	a, b       *Item
	target     *Item
	smith      int
	buf, back  string
	msg        string
	plan       *ggPlan
}

type ggState struct {
	Rec    map[int]*ggRec  `json:"rec,omitempty"`
	Firsts map[string]bool `json:"fs,omitempty"`
	Sold   map[int]string  `json:"sold,omitempty"`
	LastHP int             `json:"hp,omitempty"`
	ui     ggUI
}

func ggSt(g *Game) *ggState {
	s := g.ext("gg", func() any { return &ggState{} }).(*ggState)
	if s.Rec == nil {
		s.Rec = map[int]*ggRec{}
	}
	if s.Firsts == nil {
		s.Firsts = map[string]bool{}
	}
	if s.Sold == nil {
		s.Sold = map[int]string{}
	}
	return s
}

func (g *Game) ggFirst(key, text string) {
	st := ggSt(g)
	if !st.Firsts[key] {
		st.Firsts[key] = true
		g.deed("gear", text, 0, 0)
	}
}

func ggRng(g *Game, key string, id int) *Rng {
	return NewRng(fmt.Sprintf("%s:gg:%s:%d", g.W.Seed, key, id))
}

func ggEx(it *Item) map[string]int {
	if it.Extra == nil {
		it.Extra = map[string]int{}
	}
	return it.Extra
}

func ggEligible(it *Item) bool {
	if it == nil {
		return false
	}
	switch it.Kind {
	case "weapon", "armor", "ranged":
		return true
	case "artifact":
		return it.DN > 0 || it.AV > 0
	}
	return false
}

func ggIsWeapon(it *Item) bool {
	return it != nil && (it.Kind == "weapon" || it.Kind == "ranged" || it.Kind == "artifact" && it.DN > 0)
}

// ---- levels and stat steps ------------------------------------------------

var ggThr = []int{25, 80, 200, 500}
var ggHitTab = []int{0, 1, 1, 2, 2}
var ggDmgTab = []int{0, 0, 1, 1, 2}
var ggAVTab = []int{0, 1, 1, 2, 2}

func ggLevelFor(xp int) int {
	lv := 0
	for _, t := range ggThr {
		if xp >= t {
			lv++
		}
	}
	return lv
}

func ggLv(it *Item) int {
	if it == nil || it.Extra == nil {
		return 0
	}
	return clampi(it.Extra["gglv"], 0, 4)
}

// ggHitBonus: to-hit step of an heirloom weapon. Hook: swing() `hit += g.ggHitBonus(a, w)`.
func (g *Game) ggHitBonus(a *Actor, w *Item) int {
	if !ggIsWeapon(w) {
		return 0
	}
	return ggHitTab[ggLv(w)]
}

// ggDamageBonus: damage step of an heirloom weapon. Hook: swing() `dmg += g.ggDamageBonus(a, w)`.
func (g *Game) ggDamageBonus(a *Actor, w *Item) int {
	if !ggIsWeapon(w) {
		return 0
	}
	return ggDmgTab[ggLv(w)]
}

func ggWornArmour(a *Actor) []*Item {
	var out []*Item
	if a == nil {
		return nil
	}
	for _, p := range a.Body {
		if p.Item != nil && !p.Severed && p.Item.Kind == "armor" && ggLv(p.Item) > 0 {
			out = append(out, p.Item)
		}
	}
	return out
}

// ggArmourAV needs no Game: Actor.AV() hook `av += ggArmourAV(a)`. Total capped at +3.
func ggArmourAV(a *Actor) int {
	n := 0
	for _, it := range ggWornArmour(a) {
		n += ggAVTab[ggLv(it)]
		if ggLv(it) >= 3 && it.Extra["ggfx"] == 13 {
			n++
		}
	}
	return mini(n, 3)
}

func (g *Game) ggAVBonus(a *Actor) int { return ggArmourAV(a) }

// ggArmourDV: Actor.DV() hook `dv += ggArmourDV(a)`. Level 3+ pieces, capped at +2.
func ggArmourDV(a *Actor) int {
	n := 0
	for _, it := range ggWornArmour(a) {
		if ggLv(it) >= 3 {
			n++
		}
	}
	return mini(n, 2)
}

// ---- victims ---------------------------------------------------------------

func (g *Game) ggTags(d *Actor) []string {
	var tags []string
	t := tplByID[d.Tpl]
	undead := d.Has("undead") || t != nil && t.Has("undead")
	if undead {
		tags = append(tags, "undead")
	}
	if d.SimMon > 0 {
		tags = append(tags, "legend")
		for i := range g.W.Monsters {
			m := &g.W.Monsters[i]
			if m.ID == d.SimMon {
				ln := strings.ToLower(m.Name + " " + m.Species)
				if m.Cat == "dragon" || strings.Contains(ln, "dragon") || strings.Contains(ln, "wyrm") || strings.Contains(ln, "drake") {
					tags = append(tags, "dragon")
				}
			}
		}
	} else if d.Has("legend") {
		tags = append(tags, "legend")
	}
	if d.SimPer > 0 {
		for _, p := range g.W.People {
			if p.ID == d.SimPer && (p.Role == "ruler" || p.Role == "general") {
				tags = append(tags, "ruler")
			}
		}
	}
	if d.Has("fire") || t != nil && t.Has("fire") || d.St("burning") > 0 {
		tags = append(tags, "fire")
	}
	if d.Has("frost") || t != nil && t.Has("frost") {
		tags = append(tags, "ice")
	}
	isMan := d.Species != "" || d.SimPer > 0 || d.Role != "" || t != nil && t.Body == "humanoid" && !undead && t.Faction != "beasts"
	if isMan && !undead {
		tags = append(tags, "man")
	}
	if t != nil && (t.Faction == "beasts" || t.Faction == "vermin" || t.Faction == "wildlife") && !undead && !isMan {
		tags = append(tags, "beast")
	}
	return tags
}

func (g *Game) ggNoun(d *Actor) string {
	if d.Proper || d.SimPer > 0 {
		return ""
	}
	n := strings.ToLower(d.Name)
	if t := tplByID[d.Tpl]; t != nil {
		n = strings.ToLower(t.Name(g.Genre))
	}
	f := strings.Fields(n)
	if len(f) == 0 {
		return ""
	}
	var b strings.Builder
	for _, r := range f[len(f)-1] {
		if r >= 'a' && r <= 'z' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func ggCap(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

var ggPrio = map[string]int{"dragon": 6, "ruler": 6, "legend": 3, "undead": 2, "fire": 1, "ice": 1, "man": 0, "beast": 0}

func ggDominant(it *Item) string {
	best, bs := "", 0
	tags := make([]string, 0, len(ggPrio))
	for t := range ggPrio {
		tags = append(tags, t)
	}
	sort.Strings(tags)
	for _, t := range tags {
		c := it.Extra["ggk:"+t]
		need := 3
		if ggPrio[t] >= 3 {
			need = 1
		}
		if c < need {
			continue
		}
		if s := c*2 + ggPrio[t]; s > bs {
			best, bs = t, s
		}
	}
	return best
}

func ggTopNoun(it *Item) string {
	var keys []string
	for k := range it.Extra {
		if strings.HasPrefix(k, "ggt:") {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	best, bc := "", 0
	for _, k := range keys {
		if it.Extra[k] > bc {
			best, bc = strings.TrimPrefix(k, "ggt:"), it.Extra[k]
		}
	}
	return best
}

// ---- records, titles, history -----------------------------------------------

func (g *Game) ggRecFor(it *Item) *ggRec {
	st := ggSt(g)
	r := st.Rec[it.ID]
	if r == nil {
		r = &ggRec{Lore0: it.Lore}
		st.Rec[it.ID] = r
	}
	return r
}

func ggStrip(it *Item, r *ggRec) string {
	n := it.Name
	if r != nil && r.Title != "" && strings.HasPrefix(n, r.Title+", ") {
		n = strings.TrimPrefix(n, r.Title+", ")
	}
	return n
}

func ggSetTitle(it *Item, r *ggRec, title string) {
	base := ggStrip(it, r)
	r.Title = title
	it.Name = title + ", " + base
}

func ggRefreshLore(it *Item, r *ggRec) {
	it.Lore = strings.TrimSpace(r.Lore0 + " " + strings.Join(r.Hist, " "))
}

func (g *Game) ggPlace() string {
	if g.Cur == nil {
		return "the wilds"
	}
	return g.W.RegionName(g.W.Idx(g.Cur.WX, g.Cur.WY))
}

func (g *Game) ggAddHist(it *Item, line string) {
	r := g.ggRecFor(it)
	r.Hist = append(r.Hist, line)
	if len(r.Hist) > 7 {
		r.Hist = append(r.Hist[:2], r.Hist[3:]...)
	}
	if ggLv(it) > 0 {
		ggRefreshLore(it, r)
	}
}

func (g *Game) ggMakeTitle(it *Item) string {
	r := ggRng(g, "title", it.ID)
	dom := ggDominant(it)
	if dom == "" {
		pool := ggDefaultTitles[g.Genre]
		if len(pool) == 0 || r.Chance(0.4) {
			pool = append(append([]string{}, pool...), ggDefaultTitles[""]...)
		}
		return Pick(r, pool)
	}
	noun := ggTopNoun(it)
	if dom == "beast" && noun != "" {
		return ggCap(noun) + "bane"
	}
	words := ggTagWord[dom]
	stem := words[g.Genre]
	if stem == "" {
		stem = words[""]
	}
	if !ggIsWeapon(it) {
		return stem + Pick(r, ggArmorTail)
	}
	switch dom {
	case "dragon":
		return stem + "sbane"
	case "ruler":
		return stem + "breaker"
	case "legend":
		return stem + "bane"
	}
	tails := ggWeaponTail[it.Skill]
	if len(tails) == 0 {
		tails = ggWeaponTail["longblade"]
	}
	return stem + Pick(r, tails)
}

func ggPickFx(it *Item) int {
	dom := ggDominant(it)
	if !ggIsWeapon(it) {
		switch dom {
		case "dragon", "ruler", "legend":
			return 13
		case "undead", "fire", "ice":
			return 12
		}
		return 11
	}
	switch dom {
	case "dragon", "legend":
		return 6
	case "ruler":
		return 7
	case "undead":
		return 3
	case "fire":
		return 1
	case "ice":
		return 2
	case "man":
		return 4
	case "beast":
		return 5
	}
	return 8
}

func (g *Game) ggAddXP(it *Item, n int) {
	if n <= 0 || !ggEligible(it) {
		return
	}
	ex := ggEx(it)
	old := ggLevelFor(ex["ggxp"])
	ex["ggxp"] += n
	nl := ggLevelFor(ex["ggxp"])
	for lv := old + 1; lv <= nl; lv++ {
		g.ggLevelUp(it, lv)
	}
}

func (g *Game) ggLevelUp(it *Item, lv int) {
	ex := ggEx(it)
	ex["gglv"] = lv
	r := g.ggRecFor(it)
	yr, place := g.W.Year, g.ggPlace()
	kills := ex["ggkills"]
	switch lv {
	case 1:
		ggSetTitle(it, r, g.ggMakeTitle(it))
		r.Hist = append(r.Hist, fmt.Sprintf("Y%d: won the name %s near %s.", yr, r.Title, place))
		g.Msg(fmt.Sprintf("{{Y|%s has become an heirloom: %s.}}", ggStrip(it, r), r.Title))
		g.ggFirst("heirloom", fmt.Sprintf("The wanderer %s made an heirloom of %s.", g.P.Name, r.Title))
	case 2:
		r.Hist = append(r.Hist, fmt.Sprintf("Y%d: still at hand after %d kills, at %s.", yr, kills, place))
		g.Msg(fmt.Sprintf("{{Y|%s grows in your hand (level 2).}}", r.Title))
	case 3:
		fx := ggPickFx(it)
		ex["ggfx"] = fx
		if r.RenLv == 0 {
			ggSetTitle(it, r, strings.TrimSuffix(r.Title, " of "+place)+" of "+place)
		}
		r.Hist = append(r.Hist, fmt.Sprintf("Y%d: woke %s at %s.", yr, ggEffectNames[fx], place))
		g.Msg(fmt.Sprintf("{{Y|%s awakens: %s.}}", r.Title, ggEffectNames[fx]))
		g.ggFirst("lv3", fmt.Sprintf("%s woke %s in the hands of %s.", r.Title, ggEffectNames[fx], g.P.Name))
	case 4:
		r.Hist = append(r.Hist, fmt.Sprintf("Y%d: a legend at %s.", yr, place))
		g.Msg(fmt.Sprintf("{{Y|%s is a legend (level 4).}}", r.Title))
		g.ggFirst("lv4", fmt.Sprintf("%s became a legend in the hands of %s.", r.Title, g.P.Name))
	}
	if len(r.Hist) > 7 {
		r.Hist = append(r.Hist[:2], r.Hist[len(r.Hist)-5:]...)
	}
	ggRefreshLore(it, r)
}

// ---- earning XP ----------------------------------------------------------

func ggKill(g *Game, d, src *Actor) {
	if g.P == nil || d == nil || src != g.P {
		return
	}
	if strings.HasPrefix(d.Faction, "state:") && d.SimPer == 0 {
		return
	}
	tags, noun := g.ggTags(d), g.ggNoun(d)
	notable := d.Boss || d.SimMon > 0 || d.SimPer > 0 || d.Has("legend")
	xp := 1 + d.Level/2
	if notable {
		xp += 12 + d.Level
	}
	melee := dist(g.P, d) <= 1
	for _, p := range g.P.Body {
		it := p.Item
		if it == nil || p.Severed || !ggEligible(it) {
			continue
		}
		w := ggIsWeapon(it)
		if w {
			if p.Slot != "hand" || (it.Kind == "ranged") == melee {
				continue
			}
		}
		ex := ggEx(it)
		ex["ggkills"]++
		for _, t := range tags {
			ex["ggk:"+t]++
		}
		if noun != "" {
			k := "ggt:" + noun
			n := 0
			for key := range ex {
				if strings.HasPrefix(key, "ggt:") {
					n++
				}
			}
			if ex[k] > 0 || n < 12 {
				ex[k]++
			}
		}
		if w {
			if notable {
				g.ggAddHist(it, fmt.Sprintf("Slew %s of %s in Y%d.", d.Name, g.ggPlace(), g.W.Year))
			}
			g.ggAddXP(it, xp)
		} else if notable {
			g.ggAddXP(it, 2)
		}
	}
}

// ggOnHit: heirloom hit effects and big-hit XP. Hook: onHit() next to wpOnHit.
func (g *Game) ggOnHit(att, def *Actor, w *Item) {
	if att == nil || def == nil || !ggEligible(w) || def.HP <= 0 {
		return
	}
	if att == g.P && (def.Boss || def.Level >= att.Level+3) {
		g.ggAddXP(w, 1)
	}
	if ggLv(w) < 3 {
		return
	}
	vis := att == g.P || def == g.P || g.visible(def.X, def.Y)
	lv := ggLv(w)
	switch w.Extra["ggfx"] {
	case 1:
		if g.R.Chance(0.25) {
			def.AddSt("burning", 4)
			if vis {
				g.Msg(g.The(def) + " is licked by flame!")
			}
		}
	case 2:
		if g.R.Chance(0.2) {
			def.AddSt("frozen", 2)
		}
	case 3:
		if def.Has("undead") {
			g.hurt(def, 2, att, "a ward")
		}
		if g.R.Chance(0.15) {
			for _, s := range []string{"poisoned", "burning", "bleeding"} {
				if att.St(s) > 0 {
					att.Status[s] = 0
					break
				}
			}
		}
	case 4:
		if g.R.Chance(0.3) && !def.Has("undead") && att.HP < att.MaxHP {
			att.HP = mini(att.MaxHP, att.HP+1+lv/2)
		}
	case 5:
		if g.R.Chance(0.3) && !def.Has("undead") && def.Tpl != "golem" {
			def.AddSt("bleeding", 6)
		}
	case 6:
		if def.Boss || def.SimMon > 0 || def.Has("legend") {
			g.hurt(def, 2+lv/2, att, w.Name)
		}
	case 7:
		if !def.Boss && g.R.Chance(0.2) {
			def.AddSt("stunned", 2)
		}
	case 8:
		if g.R.Chance(0.12) {
			g.hurt(def, g.R.Int(1, 4), att, w.Name)
		}
	}
}

func ggTurn(g *Game) {
	if g.P == nil || g.Dead || g.W == nil || g.Cur == nil {
		return
	}
	st := ggSt(g)
	if st.LastHP > 0 && g.P.HP < st.LastHP {
		lost := st.LastHP - g.P.HP
		for _, it := range ggWornArmour(g.P) {
			ggEx(it)["ggab"]++
			g.ggAddXP(it, 1+lost/8)
		}
		// pieces not yet heirlooms also earn XP
		for _, p := range g.P.Body {
			if p.Item != nil && !p.Severed && p.Item.Kind == "armor" && ggLv(p.Item) == 0 {
				ggEx(p.Item)["ggab"]++
				g.ggAddXP(p.Item, 1+lost/8)
			}
		}
	}
	st.LastHP = g.P.HP
	for _, it := range ggWornArmour(g.P) {
		if ggLv(it) < 3 {
			continue
		}
		switch it.Extra["ggfx"] {
		case 11:
			if g.Turn%10 == 0 && g.P.HP < g.P.MaxHP {
				g.P.HP++
			}
		case 12:
			for _, s := range []string{"burning", "poisoned", "frozen", "bleeding"} {
				if g.P.St(s) > 1 {
					g.P.Status[s]--
				}
			}
		}
	}
	p := g.W.Idx(g.Cur.WX, g.Cur.WY)
	stamp := func(it *Item) {
		if it == nil || !isGear(it) {
			return
		}
		ex := ggEx(it)
		if ex["ggo"] == 0 {
			ex["ggo"] = 1
			if g.Turn < 5 {
				ex["ggo"] = 8
			}
			ex["ggfp"] = p + 1
		}
		ex["ggw"] = 1
	}
	for _, it := range g.P.Inv {
		stamp(it)
	}
	for _, bp := range g.P.Body {
		stamp(bp.Item)
	}
	for _, a := range g.Cur.Actors {
		if a.Role != "merchant" && a.Role != "smith" && a.Role != "apothecary" && a.Role != "exotic" && a.Role != "tailor" {
			continue
		}
		for _, it := range a.Inv {
			if it.Extra != nil && it.Extra["ggw"] == 1 && st.Sold[it.ID] == "" {
				st.Sold[it.ID] = fmt.Sprintf("Sold to %s in %s, Y%d.", a.Name, g.W.RegionName(p), g.W.Year)
			}
		}
	}
}

func ggEnter(g *Game, zn *Zone) {
	if g.W == nil || zn == nil {
		return
	}
	p := g.W.Idx(zn.WX, zn.WY)
	n := strings.ToLower(zn.Name)
	code := 1
	switch {
	case strings.Contains(n, "hoard"):
		code = 2
	case zn.Kind == "crypt" || strings.Contains(n, "barrow") || strings.Contains(n, "tomb") || strings.Contains(n, "grave") || strings.Contains(n, "crypt"):
		code = 3
	case zn.Underwater || zn.Kind == "sea" || zn.Kind == "seafloor" || strings.Contains(n, "wreck") || strings.Contains(n, "drowned"):
		code = 4
	case zn.Kind == "lair" || strings.Contains(n, "lair"):
		code = 5
	}
	mark := func(it *Item, c int, a *Actor) {
		if it == nil || !isGear(it) {
			return
		}
		ex := ggEx(it)
		if ex["ggo"] != 0 {
			return
		}
		ex["ggo"], ex["ggfp"] = c, p+1
		if a != nil {
			switch {
			case a.SimMon > 0:
				ex["ggdm"] = a.SimMon
			case a.SimPer > 0:
				ex["ggdp"] = a.SimPer
			default:
				ex["ggdk"] = 1
			}
		}
	}
	for _, it := range zn.Items {
		mark(it, code, nil)
	}
	for _, a := range zn.Actors {
		c := 7
		if a.Role == "merchant" || a.Role == "smith" || a.Role == "apothecary" || a.Role == "exotic" || a.Role == "tailor" {
			c = 6
		}
		var who *Actor
		if c == 7 {
			who = a
		}
		for _, it := range a.Inv {
			mark(it, c, who)
		}
		for _, bp := range a.Body {
			mark(bp.Item, c, who)
		}
	}
}

// ---- provenance ----------------------------------------------------------

func (g *Game) ggCulture(it *Item, r *Rng) *CultureData {
	cs := g.W.Cultures
	if len(cs) == 0 {
		return nil
	}
	if it.Region != "" {
		var m []int
		for i := range cs {
			if cs[i].Sphere == it.Region {
				m = append(m, i)
			}
		}
		if len(m) > 0 {
			return &cs[Pick(r, m)]
		}
	}
	if p := it.Extra["ggfp"] - 1; p >= 0 && p < len(g.W.Culture) && r.Chance(0.65) {
		if c := g.W.cultByID[g.W.Culture[p]]; c != nil {
			return c
		}
	}
	return &cs[r.Int(0, len(cs)-1)]
}

func (g *Game) ggBurg(c *CultureData, r *Rng) *BurgData {
	var m []int
	for i := range g.W.Burgs {
		b := &g.W.Burgs[i]
		if b.Alive && (c == nil || b.Culture == c.ID) {
			m = append(m, i)
		}
	}
	if len(m) == 0 {
		for i := range g.W.Burgs {
			if g.W.Burgs[i].Alive {
				m = append(m, i)
			}
		}
	}
	if len(m) == 0 {
		return nil
	}
	return &g.W.Burgs[Pick(r, m)]
}

func ggTrade(it *Item) string {
	switch it.Kind {
	case "armor":
		return "armourer"
	case "light":
		return "lampwright"
	case "artifact":
		return "artificer"
	}
	return "smith"
}

func (g *Game) ggProvenance(it *Item) string {
	if g.W == nil || it == nil {
		return ""
	}
	r := ggRng(g, "prov", it.ID)
	c := g.ggCulture(it, r)
	b := g.ggBurg(c, r)
	var parts []string
	if it.SimArt > 0 {
		s := "A relic of the chronicles."
		for _, a := range g.W.Artifacts {
			if a.ID == it.SimArt && a.Holder != "" {
				s = fmt.Sprintf("A relic of the chronicles, forged in Y%d. Last recorded with %s.", a.Year, a.Holder)
			}
		}
		parts = append(parts, s)
	} else {
		who := "an unnamed " + ggTrade(it)
		if c != nil && len(c.Given) > 0 {
			nm := Pick(r, c.Given)
			if len(c.Houses) > 0 {
				nm += " " + Pick(r, c.Houses)
			}
			who = nm + ", a " + c.Name + " " + ggTrade(it)
		}
		s := "Made by " + who
		if b != nil {
			s += " of " + b.Name
		}
		parts = append(parts, s+".")
	}
	ex := it.Extra
	at := g.W.RegionName(maxi(ex["ggfp"]-1, 0))
	switch ex["ggo"] {
	case 2:
		parts = append(parts, "Lifted from a hoard near "+at+".")
	case 3:
		parts = append(parts, "Taken from a barrow near "+at+".")
	case 4:
		parts = append(parts, "Salvaged from the water off "+at+".")
	case 5:
		parts = append(parts, "Carried out of a lair near "+at+".")
	case 6:
		parts = append(parts, "Bought from a shop in "+at+".")
	case 7:
		s := "Taken from a fallen foe near " + at + "."
		for _, m := range g.W.Monsters {
			if m.ID == ex["ggdm"] && ex["ggdm"] > 0 {
				s = "Dropped by " + m.Name + "."
			}
		}
		for _, p := range g.W.People {
			if p.ID == ex["ggdp"] && ex["ggdp"] > 0 {
				s = "Taken from " + p.Name + "."
			}
		}
		parts = append(parts, s)
	case 8:
		parts = append(parts, "Part of the kit you set out with.")
	case 1:
		parts = append(parts, "Found by you near "+at+".")
	}
	if s := buSt(g).Prov[it.ID]; s != "" {
		if len(s) > 160 {
			s = s[:160] + "..."
		}
		parts = append(parts, "Grave goods: "+s)
	}
	if ex["ggrf"] > 0 {
		parts = append(parts, "Reforged from two pieces.")
	}
	if s := ggSt(g).Sold[it.ID]; s != "" {
		parts = append(parts, s)
	}
	return strings.Join(parts, " ")
}

// ggDeep: previous owner and mark (shown after a successful look closer).
func (g *Game) ggDeep(it *Item) []string {
	r := ggRng(g, "deep", it.ID)
	c := g.ggCulture(it, ggRng(g, "prov", it.ID))
	var pool []*PersonData
	for i := range g.W.People {
		p := &g.W.People[i]
		if c == nil || p.Culture == c.ID {
			pool = append(pool, p)
		}
	}
	if len(pool) == 0 {
		for i := range g.W.People {
			pool = append(pool, &g.W.People[i])
		}
	}
	var out []string
	if len(pool) > 0 {
		p := Pick(r, pool)
		s := "A previous owner: " + p.Name
		if p.Title != "" {
			s += ", " + p.Title
		}
		if bd := g.W.burgByID[p.Burg]; bd != nil {
			s += " of " + bd.Name
		}
		if cl := ggRoleClause[p.Role]; cl != "" {
			s += ", " + cl
		}
		s += ". "
		if len(p.Traits) > 0 {
			s += "Said to be " + strings.Join(p.Traits[:mini(2, len(p.Traits))], " and ") + ". "
		}
		if p.Alive {
			s += "Alive when last the chronicles looked, though not carrying this."
		} else {
			s += fmt.Sprintf("Died in Y%d (%s).", p.Died, p.Cause)
		}
		out = append(out, s)
	}
	switch x := r.Int(0, 99); {
	case x < 14:
		out = append(out, "A curse-mark: "+Pick(r, ggCurseMarks)+".")
	case x < 28:
		out = append(out, "A blessing-mark: "+Pick(r, ggBlessMarks)+".")
	default:
		out = append(out, "No mark of curse or blessing.")
	}
	return out
}

// ggDescribe: heirloom block, provenance and hints. Hook: Item.Describe after the Lore lines.
func (g *Game) ggDescribe(it *Item) []string {
	if g.W == nil || it == nil || !isGear(it) {
		return nil
	}
	var l []string
	if lv := ggLv(it); lv > 0 {
		s := fmt.Sprintf("{{Y|Heirloom, level %d}} {{K|(%d XP", lv, it.Extra["ggxp"])
		if lv < len(ggThr) {
			s += fmt.Sprintf(", next at %d", ggThr[lv])
		}
		s += ")}}"
		l = append(l, "", s)
		if ggIsWeapon(it) {
			l = append(l, fmt.Sprintf("{{G|+%d to hit, +%d damage}}", ggHitTab[lv], ggDmgTab[lv]))
		} else {
			l = append(l, fmt.Sprintf("{{G|+%d AV}}", ggAVTab[lv]))
		}
		if lv >= 3 {
			l = append(l, "{{G|Power: "+ggEffectNames[it.Extra["ggfx"]]+"}}")
		}
		if d := ggDominant(it); d != "" {
			l = append(l, "{{K|Known for "+ggTagVictims[d]+".}}")
		}
	}
	l = append(l, "", "{{K|"+g.ggProvenance(it)+"}}")
	if it.Extra["ggl"] >= 2 {
		for _, s := range g.ggDeep(it) {
			l = append(l, "{{c|"+s+"}}")
		}
	}
	hint := ""
	if it.Extra["ggl"] < 2 {
		hint = "{{W|l}} look closer  "
	}
	if g.ggCanRename(it) {
		hint += "{{W|n}} rename heirloom"
	}
	if hint != "" {
		l = append(l, hint)
	}
	return l
}

// ggLook spends a turn studying an item. Hook: keyItem `case "l": g.ggLook(it)`.
func (g *Game) ggLook(it *Item) {
	if !isGear(it) {
		g.Msg("There is nothing more to learn from it.")
		return
	}
	ex := ggEx(it)
	if ex["ggl"] >= 2 {
		g.Msg("You have already read all the history it will give.")
		return
	}
	g.spend(1000)
	dc := 7 + rarityIndex(it.Rarity)
	if g.R.Dice(2, 6)+g.checkMod("lore")+ex["ggla"] >= dc {
		ex["ggl"] = 2
		g.Msg("{{c|You turn it in the light and the history comes clear. (see the item's description)}}")
		return
	}
	ex["ggla"]++
	g.Msg("{{K|You find nothing new. (Lore check failed; another try will go easier.)}}")
}

// ---- rename -------------------------------------------------------------

func (g *Game) ggCanRename(it *Item) bool {
	r := ggSt(g).Rec[it.ID]
	return r != nil && ggLv(it) >= 1 && r.RenLv < ggLv(it)
}

// ggRenameStart: Hook: keyItem `case "n": g.ggRenameStart(it)`.
func (g *Game) ggRenameStart(it *Item) {
	if !g.ggCanRename(it) {
		g.Msg("It can only be renamed once for each level it has earned.")
		return
	}
	u := &ggSt(g).ui
	*u = ggUI{stage: 5, target: it, back: "inv"}
	g.ItemSel = nil
	g.Mode = "ggf"
}

// ---- reforging -----------------------------------------------------------

func ggFamily(a, b *Item) bool {
	if a == b || a == nil || b == nil || a.Kind != b.Kind {
		return false
	}
	switch a.Kind {
	case "weapon", "ranged":
		return a.Skill == b.Skill
	case "armor":
		return a.Slot == b.Slot
	}
	return false
}

func ggScore(it *Item) int {
	return rarityIndex(it.Rarity)*100 + it.Tier*10 + it.PV + it.AV + it.DN*it.DS/2 + it.DB + 2*len(modsOf(it)) + ggLv(it)
}

func ggProtected(it *Item) bool {
	return rarityIndex(it.Rarity) >= 4 || it.SimArt > 0 || it.Quest > 0
}

func modCostSum(md *modDef) int {
	n := 0
	for t, c := range md.Cost {
		n += t * c
	}
	return n
}

type ggPlan struct {
	base, fod *Item
	err       string
	chance    float64
	dc, mod   int
	gain      []string
	lv, xp    int
	src       *Item
	fee       int
	cost      map[int]int
	smith     bool
}

func (g *Game) ggMakePlan(a, b *Item, smith bool) *ggPlan {
	p := &ggPlan{smith: smith}
	if ggProtected(a) && ggProtected(b) {
		p.err = "Neither of these can be given up."
		return p
	}
	p.base, p.fod = a, b
	if ggScore(b) > ggScore(a) {
		p.base, p.fod = b, a
	}
	if ggProtected(p.fod) && !ggProtected(p.base) {
		p.base, p.fod = p.fod, p.base
	}
	ri := maxi(rarityIndex(a.Rarity), rarityIndex(b.Rarity))
	tier := maxi(a.Tier, b.Tier)
	p.dc = 7 + ri + len(modsOf(p.fod))/2
	p.mod = g.checkMod("tinkering")
	if smith {
		p.mod += 3
		p.fee = maxi(10, 25*(1+tier)*(1+ri)*(100-g.Skills["persuasion"]*10)/100)
	} else {
		p.cost = map[int]int{clampi(tier+1, 2, 5): 2 + ri}
	}
	p.chance = chanceFor(p.mod, p.dc)
	have := map[string]bool{}
	for _, m := range modsOf(p.base) {
		have[m] = true
	}
	var cand []*modDef
	for _, m := range modsOf(p.fod) {
		if md := modByID(m); md != nil && !have[m] && isKind(md.Kinds, p.base.Kind) {
			cand = append(cand, md)
		}
	}
	sort.Slice(cand, func(i, j int) bool { return modCostSum(cand[i]) > modCostSum(cand[j]) })
	free := []int{1, 1, 2, 2, 3, 4, 4}[rarityIndex(p.base.Rarity)] - len(modsOf(p.base))
	for i := 0; i < len(cand) && i < free; i++ {
		p.gain = append(p.gain, cand[i].ID)
	}
	p.src = p.base
	if ggLv(p.fod) > ggLv(p.base) || ggLv(p.fod) == ggLv(p.base) && p.fod.Extra["ggxp"] > p.base.Extra["ggxp"] {
		p.src = p.fod
	}
	if ggLv(p.src) > 0 {
		p.xp = p.src.Extra["ggxp"] * 80 / 100
		p.lv = maxi(1, ggLevelFor(p.xp))
	}
	return p
}

func ggUnmod(it *Item, id string) {
	switch id {
	case "sharp":
		it.PV -= 2
	case "counterweighted":
		addBonus(it, AGI, -1)
	case "flaming", "freezing", "electrified":
		it.Elem = ""
	case "reinforced":
		it.AV--
	case "lightweight":
		it.Weight += 2
	case "scoped":
		it.Range -= 3
	case "masterwork":
		it.PV--
		it.DB--
	}
	delete(it.Extra, "mod:"+id)
}

func (g *Game) ggUnequip(it *Item) bool {
	for _, bp := range g.P.Body {
		if bp.Item == it {
			bp.Item = nil
			return true
		}
	}
	return false
}

func (g *Game) ggReforge(u *ggUI) string {
	st := ggSt(g)
	p := u.plan
	if p == nil || p.err != "" {
		return "That cannot be done."
	}
	if p.smith {
		if g.P.Gold < p.fee {
			return "You cannot pay the fee."
		}
		g.P.Gold -= p.fee
	} else {
		cs := g.craftSt()
		if !cs.canPay(p.cost) {
			return "Not enough bits (" + costText(p.cost, 0) + ")."
		}
		cs.pay(p.cost)
	}
	g.spend(3000)
	base, fod := p.base, p.fod
	if g.R.Dice(2, 6)+p.mod < p.dc {
		ex := ggEx(base)
		if ms := modsOf(base); len(ms) > 0 {
			id := ms[len(ms)-1]
			ggUnmod(base, id)
			g.refreshModName(base)
			return "The work goes wrong: " + base.Name + " loses its " + id + " mod. The other piece is returned."
		}
		if ggIsWeapon(base) && base.DS > 2 {
			base.DS--
		} else if base.AV > 0 {
			base.AV--
		} else if base.PV > 0 {
			base.PV--
		}
		ex["ggrf"] += 0
		return "The work goes wrong: " + base.Name + " is damaged. The other piece is returned."
	}
	ex := ggEx(base)
	for _, id := range p.gain {
		if md := modByID(id); md != nil {
			md.Apply(base)
			ex["mod:"+id] = 1
			base.Value += 15 * (len(md.Cost) + 1)
		}
	}
	base.Value += fod.Value / 4
	if ggLv(p.src) > 0 {
		old := st.Rec[base.ID]
		if p.src == fod {
			fr := st.Rec[fod.ID]
			nr := &ggRec{}
			if fr != nil {
				*nr = *fr
				nr.Hist = append([]string{}, fr.Hist...)
			}
			nr.Lore0 = base.Lore
			if old != nil {
				nr.Lore0 = old.Lore0
			}
			base.Name = ggStrip(base, old)
			for k := range ex {
				if strings.HasPrefix(k, "ggk:") || strings.HasPrefix(k, "ggt:") {
					delete(ex, k)
				}
			}
			for k, v := range fod.Extra {
				if strings.HasPrefix(k, "ggk:") || strings.HasPrefix(k, "ggt:") || k == "ggkills" || k == "ggfx" || k == "ggab" {
					ex[k] = v
				}
			}
			st.Rec[base.ID] = nr
			ggSetTitle(base, &ggRec{}, nr.Title)
			// ggSetTitle with an empty record prefixes the title; fix the stored title
		}
		r := st.Rec[base.ID]
		ex["ggxp"], ex["gglv"] = p.xp, p.lv
		r.Hist = append(r.Hist, fmt.Sprintf("Y%d: reforged at %s, and lost a fifth of its fame.", g.W.Year, g.ggPlace()))
		if len(r.Hist) > 7 {
			r.Hist = append(r.Hist[:2], r.Hist[len(r.Hist)-5:]...)
		}
		ggRefreshLore(base, r)
	}
	ex["ggrf"]++
	g.refreshModName(base)
	// remove the fodder; if it was worn and the base was not, the base takes its place
	for _, bp := range g.P.Body {
		if bp.Item == fod {
			if g.P.Equipped(base) == nil {
				g.removeInv(base)
				bp.Item = base
			} else {
				bp.Item = nil
			}
		}
	}
	g.removeInv(fod)
	delete(st.Rec, fod.ID)
	g.recompute()
	g.ggFirst("reforge", fmt.Sprintf("The wanderer %s reforged two pieces into %s.", g.P.Name, base.Name))
	return "You reforge the pieces into " + base.Name + "."
}

// ---- screen --------------------------------------------------------------

func (g *Game) ggOpen(smith int, back string) {
	st := ggSt(g)
	st.ui = ggUI{smith: smith, back: back}
	g.Mode = "ggf"
}

// ggUse runs gg_ abilities. Hook: usePower() `if id == "gg_forge" { g.ggUse(id); return }`.
func (g *Game) ggUse(id string) { g.ggOpen(0, "") }

// ggAbilities: Hook: Abilities() `out = append(out, g.ggAbilities()...)`.
func (g *Game) ggAbilities() []Ability {
	if g.P == nil || g.tinkRank() < 1 {
		return nil
	}
	return []Ability{{ID: "gg_forge", Name: "Reforge or rename gear at the workbench", Target: "self"}}
}

func (g *Game) ggLabel(it *Item) string {
	s := it.Display()
	if g.P.Equipped(it) != nil {
		s += " {{K|(worn)}}"
	}
	if lv := ggLv(it); lv > 0 {
		s += fmt.Sprintf(" {{Y|[heirloom %d]}}", lv)
	}
	return s + " {{K|" + RarityLabel(it.Rarity) + "}}"
}

func (g *Game) ggList(u *ggUI) []*Item {
	var out []*Item
	for _, it := range g.ownedGear() {
		switch u.stage {
		case 1:
			out = append(out, it)
		case 2:
			if ggFamily(u.a, it) {
				out = append(out, it)
			}
		case 4:
			if ggSt(g).Rec[it.ID] != nil && ggLv(it) > 0 {
				out = append(out, it)
			}
		}
	}
	return out
}

func ggLeave(g *Game) {
	u := &ggSt(g).ui
	g.Mode = "main"
	if u.back == "inv" {
		g.Mode = "inv"
		return
	}
	if u.smith != 0 {
		if a := g.findActor(u.smith); a != nil {
			g.openTalk(a)
		}
	}
}

func ggfKeys(g *Game, k string) {
	u := &ggSt(g).ui
	if u.stage == 5 {
		switch k {
		case "Escape":
			ggLeave(g)
		case "Enter":
			name := strings.TrimSpace(u.buf)
			if name != "" && u.target != nil {
				r := ggSt(g).Rec[u.target.ID]
				if r != nil {
					ggSetTitle(u.target, r, name)
					r.RenLv = ggLv(u.target)
					r.Hist = append(r.Hist, fmt.Sprintf("Y%d: renamed %s by %s.", g.W.Year, name, g.P.Name))
					ggRefreshLore(u.target, r)
					g.Msg("It is called {{Y|" + name + "}} now.")
					g.ggFirst("rename", fmt.Sprintf("The wanderer %s named a blade or mail %s.", g.P.Name, name))
				}
			}
			ggLeave(g)
		case "Backspace":
			if rs := []rune(u.buf); len(rs) > 0 {
				u.buf = string(rs[:len(rs)-1])
			}
		default:
			if rs := []rune(k); len(rs) == 1 && rs[0] >= 32 && rs[0] < 127 && len(u.buf) < 24 {
				u.buf += k
			}
		}
		return
	}
	var list []*Item
	n := 3
	if u.stage == 1 || u.stage == 2 || u.stage == 4 {
		list = g.ggList(u)
		n = len(list)
	}
	switch k {
	case "ArrowUp":
		u.sel = maxi(0, u.sel-1)
	case "ArrowDown":
		u.sel = mini(maxi(0, n-1), u.sel+1)
	case "Escape":
		u.msg = ""
		switch u.stage {
		case 0:
			ggLeave(g)
		case 3:
			u.stage, u.sel = 2, 0
		default:
			u.stage, u.sel = 0, 0
		}
	case "Enter", " ":
		u.msg = ""
		switch u.stage {
		case 0:
			switch u.sel {
			case 0:
				u.stage, u.sel = 1, 0
			case 1:
				u.stage, u.sel = 4, 0
			default:
				ggLeave(g)
			}
		case 1:
			if u.sel < len(list) {
				u.a, u.stage, u.sel = list[u.sel], 2, 0
				if len(g.ggList(u)) == 0 {
					u.msg, u.stage = "Nothing else of that kind to melt in.", 1
				}
			}
		case 2:
			if u.sel < len(list) {
				u.b = list[u.sel]
				u.plan = g.ggMakePlan(u.a, u.b, u.smith != 0)
				u.stage = 3
			}
		case 3:
			if u.plan != nil && u.plan.err == "" {
				u.msg = g.ggReforge(u)
				u.stage, u.sel, u.a, u.b, u.plan = 0, 0, nil, nil, nil
			}
		case 4:
			if u.sel < len(list) {
				if !g.ggCanRename(list[u.sel]) {
					u.msg = "It has earned no new name yet (once per level)."
				} else {
					u.target, u.stage, u.buf = list[u.sel], 5, ""
				}
			}
		}
	}
}

func ggfDraw(g *Game) {
	u := &ggSt(g).ui
	x, y, _, h := g.panel("Forge")
	s := g.sc
	row := y
	put := func(t string) {
		for _, ln := range Wrap(t, 90) {
			s.Text(x, row, ln, CGray)
			row++
		}
	}
	if u.smith != 0 {
		put("{{K|The smith works for gold.}}")
	} else {
		put("{{K|The workbench takes bits. " + g.craftSt().bitsLine() + "}}")
	}
	row++
	mark := func(i int) string {
		if i == u.sel {
			return "{{Y|>}} "
		}
		return "  "
	}
	switch u.stage {
	case 0:
		for i, t := range []string{"Reforge two items into one", "Rename an heirloom", "Leave"} {
			s.Text(x, row, mark(i)+t, CGray)
			row++
		}
	case 1, 2, 4:
		title := map[int]string{1: "Choose the first item:", 2: "Choose the second (same kind) to melt in:", 4: "Choose an heirloom to rename:"}[u.stage]
		put(title)
		list := g.ggList(u)
		top := maxi(0, u.sel-(h-10))
		for i := top; i < len(list) && row < y+h-3; i++ {
			s.Text(x, row, mark(i)+g.ggLabel(list[i]), CGray)
			row++
		}
		if len(list) == 0 {
			put("{{K|(nothing suitable)}}")
		}
	case 3:
		p := u.plan
		if p.err != "" {
			put("{{R|" + p.err + "}}")
			break
		}
		put("{{W|Base kept:}} " + g.ggLabel(p.base))
		put("{{R|Melted in:}} " + g.ggLabel(p.fod))
		if len(p.gain) > 0 {
			put("{{G|Mods gained: " + strings.Join(p.gain, ", ") + "}}")
		} else {
			put("{{K|No mods to add (none fit, or no free slot for its rarity).}}")
		}
		if ggLv(p.src) > 0 {
			r := ggSt(g).Rec[p.src.ID]
			t := ""
			if r != nil {
				t = r.Title
			}
			put(fmt.Sprintf("{{Y|Heirloom %s kept at level %d (a fifth of its fame is lost).}}", t, p.lv))
		}
		if p.smith {
			put(fmt.Sprintf("Fee: %d %s", p.fee, g.Sk.Currency))
		} else {
			put("Cost: " + costText(p.cost, 0))
		}
		put(fmt.Sprintf("Chance of success: %d%%. Failure lowers a mod or a stat of the base; the other piece is returned.", int(p.chance*100+0.5)))
		put("{{K|Enter: reforge   Esc: back}}")
	case 5:
		put("{{Y|Name this heirloom:}}")
		put("{{W|" + u.buf + "}}{{Y|_}}")
		put("{{K|type a name, Enter confirm, Esc cancel}}")
	}
	if u.msg != "" {
		s.Text(x, y+h-2, u.msg, CGray)
	}
	if u.stage != 3 && u.stage != 5 {
		s.Text(x, y+h-1, "{{K|Up/Down: select   Enter: choose   Esc: back}}", CGray)
	}
}
