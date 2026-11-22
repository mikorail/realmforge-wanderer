package engine

// In-game Encyclopedia (prefix enc). F1 or the palette entry "Encyclopedia" opens it (mode "enc").
// Pages are built from live data when the screen opens (attributes, skills, statuses, damage types,
// stances, item classes, species, realms, cooking tags, powers, the world's culture packs and faith
// lore). Places and people you have met are recorded in g.ext("enc") (planet-local) and only those
// get entries; cultures and faiths appear once you have met them (your own always do).
// Type to filter, Tab or Left/Right changes category, Enter opens the links of a page.
// Read-only and cosmetic: it never draws g.R and changes no game state beyond its own record.

import (
	"fmt"
	"sort"
	"strings"
)

type encPlace struct {
	Name  string `json:"n"`
	Kind  string `json:"k"`
	Burg  int    `json:"b"` // burg id, -1 when not a town
	State int    `json:"s"`
	Cult  int    `json:"c"` // culture id, -1 unknown
	Rel   int    `json:"r"` // religion id, -1 unknown
}

type encPerson struct {
	Name  string `json:"n"`
	Title string `json:"t,omitempty"`
	Role  string `json:"ro,omitempty"`
	Caste string `json:"ca,omitempty"`
	House string `json:"h,omitempty"`
	Spec  string `json:"sp,omitempty"`
	Where string `json:"w,omitempty"`
	Sim   int    `json:"sm,omitempty"`
	State int    `json:"s"`
	Cult  int    `json:"c"`
	Rel   int    `json:"r"`
}

type encState struct {
	Places map[string]*encPlace  `json:"pl,omitempty"`
	People map[string]*encPerson `json:"pe,omitempty"`

	// interface state, never saved
	entries []*encEntry
	idx     map[string]int
	cat     int
	buf     string
	sel     int
	focus   int // 0 list, 1 page links
	scroll  int
	lsel    int
	hist    []string
	lastID  string
}

type encEntry struct {
	ID, Cat, Title, Sub string
	Body                []string
	Links               []string
	hay                 string
}

func (g *Game) enc() *encState {
	st := g.ext("enc", func() any { return &encState{} }).(*encState)
	if st.Places == nil {
		st.Places = map[string]*encPlace{}
	}
	if st.People == nil {
		st.People = map[string]*encPerson{}
	}
	return st
}

func init() {
	unLocalKeys = append(unLocalKeys, "enc")
	modeKeys["enc"] = encKeys
	modeDraws["enc"] = encDraw
	if _, taken := mainKeys["F1"]; !taken {
		mainKeys["F1"] = func(g *Game) { encOpen(g) }
	}
	enterHooks = append(enterHooks, func(g *Game, zn *Zone) { g.encNoteHere() })
	turnHooks = append(turnHooks, func(g *Game) { g.encNoteHere() })
	paEntries = append(paEntries, paEntry{Name: "Encyclopedia", Keywords: "codex lore reference wiki help skills statuses cultures", Cat: "Screen",
		Key: "F1", Func: func(g *Game) { encOpen(g) }})
}

func encOpen(g *Game) {
	st := g.enc()
	st.entries = g.encBuild()
	st.idx = map[string]int{}
	for i, e := range st.entries {
		st.idx[e.ID] = i
	}
	st.cat, st.buf, st.sel, st.focus, st.scroll, st.lsel, st.hist, st.lastID = 0, "", 0, 0, 0, 0, nil, ""
	g.Mode = "enc"
}

func encClose(g *Game) {
	g.Mode = "main"
	if g.OnWorld {
		g.Mode = "world"
	}
}

// ---------------------------------------------------------------- discovery

func (g *Game) encRelOfState(id int) int {
	if s := g.W.stateByID[id]; s != nil {
		return s.Religion
	}
	return -1
}

func (g *Game) encNoteHere() {
	if g.Cur == nil || g.P == nil || g.W == nil || g.Dead {
		return
	}
	st := g.enc()
	zn := g.Cur
	curCult, curState, curRel := -1, 0, -1
	switch zn.Kind {
	case "town", "ruin", "lair", "wonder", "crypt", "dungeon", "tower", "camp":
		pl := &encPlace{Name: zn.Name, Kind: zn.Kind, Burg: -1, Cult: -1, Rel: -1}
		key := zn.Kind + ":" + zn.Name
		p := g.W.Idx(zn.WX, zn.WY)
		if p >= 0 && p < len(g.W.State) {
			pl.State = g.W.State[p]
			if p < len(g.W.Culture) {
				pl.Cult = g.W.Culture[p]
			}
			if p < len(g.W.Religion) {
				pl.Rel = g.W.Religion[p]
			}
		}
		if zn.Kind == "town" {
			if bd := g.W.burgByID[zn.Burg]; bd != nil {
				pl.Name, pl.Burg, pl.State, pl.Cult = bd.Name, bd.ID, bd.State, bd.Culture
				pl.Rel = g.encRelOfState(bd.State)
				key = fmt.Sprintf("b%d", bd.ID)
			}
		}
		if _, ok := st.Places[key]; !ok && pl.Name != "" {
			st.Places[key] = pl
		}
		curCult, curState, curRel = pl.Cult, pl.State, pl.Rel
	default:
		p := g.W.Idx(zn.WX, zn.WY)
		if p >= 0 && p < len(g.W.State) {
			curState = g.W.State[p]
			if p < len(g.W.Culture) {
				curCult = g.W.Culture[p]
			}
			if p < len(g.W.Religion) {
				curRel = g.W.Religion[p]
			}
		}
	}
	for _, a := range g.Cur.Actors {
		if a == nil || a == g.P || a.HP <= 0 || a.Name == "" || !(a.Talked || a.Kinship) || g.Hostile(a) {
			continue
		}
		s := a.State
		if s == 0 {
			s = curState
		}
		key := fmt.Sprintf("%s|%d", a.Name, s)
		if _, ok := st.People[key]; ok {
			continue
		}
		r := a.Religion
		if r == 0 {
			r = curRel
		}
		st.People[key] = &encPerson{Name: a.Name, Title: a.Title, Role: a.Role, Caste: a.Caste, House: a.House, Spec: a.Species,
			Where: g.Cur.Name, Sim: a.SimPer, State: s, Cult: curCult, Rel: r}
	}
}

// encKnown returns the culture and faith ids the player has met (their own always).
func (g *Game) encKnown() (map[int]bool, map[int]bool) {
	cu, fa := map[int]bool{}, map[int]bool{}
	st := g.enc()
	if bd := g.W.burgByID[g.StartBurg]; bd != nil {
		cu[bd.Culture] = true
	}
	if s := g.W.stateByID[g.HomeState]; s != nil {
		cu[s.Culture] = true
		fa[s.Religion] = true
	}
	fa[g.HomeFaith] = true
	for _, p := range st.Places {
		if p.Cult >= 0 {
			cu[p.Cult] = true
		}
		if p.Rel >= 0 {
			fa[p.Rel] = true
		}
	}
	for _, p := range st.People {
		if p.Cult >= 0 {
			cu[p.Cult] = true
		}
		if p.Rel >= 0 {
			fa[p.Rel] = true
		}
		if p.Sim > 0 {
			for _, pd := range g.W.People {
				if pd.ID == p.Sim {
					cu[pd.Culture] = true
					fa[pd.Religion] = true
				}
			}
		}
	}
	return cu, fa
}

// ---------------------------------------------------------------- builder

type encB struct {
	e *encEntry
}

func (b *encB) p(s string)  { b.e.Body = append(b.e.Body, s) }
func (b *encB) gap()        { b.e.Body = append(b.e.Body, "") }
func (b *encB) h(s string)  { b.gap(); b.e.Body = append(b.e.Body, "{{W|"+s+"}}") }
func (b *encB) kv(k, v string) {
	if v != "" {
		b.e.Body = append(b.e.Body, "{{K|"+k+":}} "+v)
	}
}
func (b *encB) bullets(l []string, max int) {
	for i, s := range l {
		if max > 0 && i >= max {
			b.e.Body = append(b.e.Body, fmt.Sprintf("{{K|... and %d more}}", len(l)-max))
			break
		}
		b.e.Body = append(b.e.Body, "- "+s)
	}
}
func (b *encB) link(ids ...string) {
	for _, id := range ids {
		if id == "" {
			continue
		}
		dup := false
		for _, x := range b.e.Links {
			if x == id {
				dup = true
			}
		}
		if !dup {
			b.e.Links = append(b.e.Links, id)
		}
	}
}

type encList struct{ all []*encEntry }

func (l *encList) add(id, cat, title, sub string) *encB {
	e := &encEntry{ID: id, Cat: cat, Title: title, Sub: sub}
	l.all = append(l.all, e)
	return &encB{e}
}

func encSigned(n int) string {
	if n > 0 {
		return fmt.Sprintf("+%d", n)
	}
	return fmt.Sprintf("%d", n)
}

func (g *Game) encBuild() []*encEntry {
	var l encList
	g.encAttrs(&l)
	g.encStatuses(&l)
	g.encCombat(&l)
	g.encItems(&l)
	g.encSpecies(&l)
	g.encRealms(&l)
	g.encCooking(&l)
	g.encMagic(&l)
	g.encCultures(&l)
	g.encPlaces(&l)
	for _, e := range l.all {
		e.hay = strings.ToLower(Plain(e.Title + " " + e.Sub + " " + strings.Join(e.Body, " ")))
	}
	return l.all
}

func (g *Game) encAttrs(l *encList) {
	for i := 0; i < 6; i++ {
		b := l.add(fmt.Sprintf("attr:%d", i), "attr", AttrNames[i], AttrShort[i])
		b.p(encAttrText[i])
		if g.P != nil {
			b.gap()
			b.kv("Your score", fmt.Sprintf("%d (modifier %s)", g.P.Score(i), encSigned(g.P.M(i))))
		}
		for _, cd := range callingDefs {
			if cd.Caste {
				continue
			}
			if cd.Attr[i] > 0 {
				b.link("calling:" + cd.ID)
			}
		}
		if i == EGO {
			b.link("rep")
		}
	}
	for _, cd := range callingDefs {
		if cd.Caste {
			continue
		}
		b := l.add("calling:"+cd.ID, "attr", g.Sk.Calling(cd.ID), "calling")
		b.p(cd.Desc)
		var at []string
		for i := 0; i < 6; i++ {
			if cd.Attr[i] != 0 {
				at = append(at, fmt.Sprintf("%s %s", AttrShort[i], encSigned(cd.Attr[i])))
			}
		}
		b.kv("Attributes", strings.Join(at, ", "))
		var sk []string
		ids := make([]string, 0, len(cd.Skills))
		for k := range cd.Skills {
			ids = append(ids, k)
		}
		sort.Strings(ids)
		for _, k := range ids {
			sk = append(sk, fmt.Sprintf("%s %d", skillName(k), cd.Skills[k]))
			b.link("skill:" + k)
		}
		b.kv("Skills", strings.Join(sk, ", "))
		var rq []string
		for i := 0; i < 6; i++ {
			if v := callingReq[cd.ID][i]; v > 0 {
				rq = append(rq, fmt.Sprintf("%s %d", AttrShort[i], v))
			}
		}
		b.kv("Needs", strings.Join(rq, ", "))
	}
	seen := map[string]bool{}
	ids := append([]string{}, SkillIDs...)
	var extra []string
	for k := range wpSkillNames {
		extra = append(extra, k)
	}
	sort.Strings(extra)
	ids = append(ids, extra...)
	for _, id := range ids {
		if seen[id] {
			continue
		}
		seen[id] = true
		b := l.add("skill:"+id, "attr", skillName(id), "skill")
		d := skillDesc[id]
		if d == "" {
			d = encSkillExtra[id]
		}
		if d == "" {
			d = "A weapon or craft skill."
		}
		b.p(d)
		if g.Skills != nil && g.Skills[id] > 0 {
			b.gap()
			b.kv("Your rank", fmt.Sprintf("%d", g.Skills[id]))
		}
		var who []string
		for _, cd := range callingDefs {
			if !cd.Caste && cd.Skills[id] > 0 {
				who = append(who, g.Sk.Calling(cd.ID))
				b.link("calling:" + cd.ID)
			}
		}
		b.kv("Callings that begin with it", strings.Join(who, ", "))
		if t, ok := bwPhysSkill[id]; ok {
			b.kv("Deals", t+" damage")
			b.link("dtype:" + t)
		}
		b.link("item:weapon")
		b.gap()
		b.p("{{K|Train skills with points on the Skills screen (K).}}")
	}
}

func encDV(d *skStatusDef) string {
	var p []string
	add := func(n int, s string) {
		if n != 0 {
			p = append(p, fmt.Sprintf("%s %s", encSigned(n), s))
		}
	}
	add(d.DV, "DV")
	add(d.Hit, "to hit")
	add(d.AV, "AV")
	add(d.MA, "mental armor")
	add(d.AGI, "Agility")
	add(d.INT, "Intelligence")
	add(d.WIL, "Willpower")
	add(d.EGO, "Ego")
	if d.Speed < 0 {
		p = append(p, fmt.Sprintf("slower (%d speed)", d.Speed))
	} else if d.Speed > 0 {
		p = append(p, fmt.Sprintf("faster (+%d speed)", d.Speed))
	}
	add(d.PV, "penetration against it")
	return strings.Join(p, ", ")
}

func (g *Game) encStatuses(l *encList) {
	ids := make([]string, 0, len(skStatuses))
	for k := range skStatuses {
		ids = append(ids, k)
	}
	sort.Slice(ids, func(i, j int) bool { return skStatuses[ids[i]].Name < skStatuses[ids[j]].Name })
	done := map[string]bool{}
	for _, k := range ids {
		d := skStatuses[k]
		done[k] = true
		b := l.add("status:"+k, "status", d.Name, "status")
		if s := encDV(d); s != "" {
			b.p(s + ".")
		}
		if n := encStatusNote[k]; n != "" {
			b.p(n)
		} else if d.Tick != nil {
			b.p("Does something each turn while it lasts.")
		}
		if d.Glyph != 0 {
			b.kv("Shown as", string(d.Glyph))
		}
		if g.P != nil && g.P.St(k) > 0 {
			b.gap()
			b.p(fmt.Sprintf("{{Y|You have this now (%d turns left).}}", g.P.St(k)))
		}
	}
	for _, o := range encOtherStatuses {
		if done[o.ID] {
			continue
		}
		b := l.add("status:"+o.ID, "status", o.Name, "status")
		b.p(o.Text)
		if g.P != nil && g.P.St(o.ID) > 0 {
			b.gap()
			b.p(fmt.Sprintf("{{Y|You have this now (%d turns left).}}", g.P.St(o.ID)))
		}
		switch o.ID {
		case "burning":
			b.link("dtype:fire")
		case "frozen":
			b.link("dtype:cold", "cook:rime")
		case "poisoned":
			b.link("dtype:poison", "cook:mending")
		case "fed":
			b.link("cook")
		}
	}
}

func (g *Game) encCombat(l *encList) {
	b := l.add("combat", "combat", "How combat works", "overview")
	b.p(encCombatIntro)
	b.gap()
	b.p("Damage types: " + strings.Join(bwTypes, ", ") + ".")
	for _, t := range bwTypes {
		b.link("dtype:" + t)
	}
	b.link("stances", "status:cs_stagger", "status:cs_offbal", "status:cs_pinned", "status:cs_disarmed")
	for _, t := range bwTypes {
		b := l.add("dtype:"+t, "combat", "Damage: "+t, "damage type")
		b.p(encDamageText[t])
		var skills []string
		for sid, ty := range bwPhysSkill {
			if ty == t {
				skills = append(skills, sid)
			}
		}
		sort.Strings(skills)
		var names []string
		for _, s := range skills {
			names = append(names, skillName(s))
			b.link("skill:" + s)
		}
		b.kv("Weapon skills", strings.Join(names, ", "))
		var res, weak []string
		var flags []string
		for f := range bwFlagRes {
			flags = append(flags, f)
		}
		sort.Strings(flags)
		for _, f := range flags {
			v := bwFlagRes[f][t]
			if v >= 50 {
				res = append(res, f)
			} else if v <= -25 {
				weak = append(weak, f)
			}
		}
		b.kv("Strongly resisted by creatures flagged", strings.Join(res, ", "))
		b.kv("Extra damage against creatures flagged", strings.Join(weak, ", "))
		b.link("combat")
	}
	sk := csSkinFor(g.Genre)
	b = l.add("stances", "combat", "Stances", "combat")
	b.p("Hold a stance (abilities, a) to trade accuracy, defence and fatigue. Only one at a time.")
	names := [4]string{}
	keys := []string{"guard", "aggr", "evade", "aim"}
	for i, def := range csStances {
		names[i] = sk.Stance[i]
		b.h(sk.Stance[i])
		b.p(fmt.Sprintf("{{K|Melee hit %s, ranged hit %s, DV %s, fatigue %.0f%%.}}", encSigned(def.Hit), encSigned(def.RHit), encSigned(def.DV), def.Fat*100))
		if t := encStanceText[keys[i]]; t != "" {
			b.p(t)
		}
	}
	b.h("Techniques by weapon class")
	for _, c := range csClassOrder {
		b.p(fmt.Sprintf("%s: {{W|%s}} (active), {{W|%s}} (on hit). %s", csClassNames[c], sk.Tech[c], sk.Passive[c], encClassText[c]))
	}
	b.gap()
	b.p(fmt.Sprintf("Also: %s, %s, %s and %s (a momentum counter that unlocks the finisher).", sk.Finish, sk.Riposte, sk.Bash, sk.Combo))
	b.link("combat")
}

func (g *Game) encItems(l *encList) {
	b := l.add("item", "item", "Item classes", "overview")
	b.p("Everything you carry belongs to a class. Open the inventory with {{W|i}}.")
	b.gap()
	for _, k := range encItemKinds {
		b.p("- " + k.Name)
		b.link("item:" + k.Kind)
	}
	b.h("Tiers and materials")
	b.p("Gear improves by tier. Materials in this genre, lowest to highest: " + strings.Join(g.Sk.Mats[:], ", ") + ".")
	b.link("item:tier")
	for _, k := range encItemKinds {
		e := l.add("item:"+k.Kind, "item", k.Name, "item class")
		e.p(k.Text)
		if g.P != nil {
			n := 0
			for _, it := range g.P.Inv {
				if it != nil && it.Kind == k.Kind {
					n++
				}
			}
			if n > 0 {
				e.gap()
				e.p(fmt.Sprintf("{{Y|You carry %d of these.}}", n))
			}
		}
		switch k.Kind {
		case "weapon", "ranged":
			e.link("combat", "skill:longblade", "dtype:slash", "dtype:blunt", "dtype:pierce")
		case "armor":
			e.link("combat", "stances")
		case "ingredient", "food":
			e.link("cook")
		case "tonic":
			e.link("status:poisoned", "status:bleeding")
		}
		e.link("item")
	}
	e := l.add("item:tier", "item", "Tiers and materials", "items")
	e.p("Weapons and armor come in tiers. Each tier is named for a material in this world's genre:")
	for i, m := range g.Sk.Mats {
		e.p(fmt.Sprintf("- Tier %d: %s", i, m))
	}
	e.gap()
	e.p("Rarer gear carries mods and a provenance; relics are unidentified until studied.")
	e.link("item")
}

func (g *Game) encSpecies(l *encList) {
	for i := range g.W.Species {
		sp := &g.W.Species[i]
		b := l.add("species:"+sp.ID, "species", sp.Name, "species")
		b.p(sp.Desc)
		b.gap()
		var at []string
		am := speciesAttr(sp)
		for a := 0; a < 6; a++ {
			if am[a] != 0 {
				at = append(at, fmt.Sprintf("%s %s", AttrShort[a], encSigned(am[a])))
				b.link(fmt.Sprintf("attr:%d", a))
			}
		}
		b.kv("Attribute bonuses", strings.Join(at, ", "))
		b.kv("Body", BodyPlanForSpecies(sp.ID))
		b.kv("Traits", strings.Join(sp.Traits, ", "))
		b.kv("Plural", sp.Plural)
		if g.P != nil && g.P.Species == sp.ID {
			b.p("{{Y|You are of this people.}}")
		}
		var cu []string
		for _, c := range g.W.Cultures {
			if c.Species == sp.ID {
				cu = append(cu, c.Name)
				b.link(fmt.Sprintf("culture:%d", c.ID))
			}
		}
		b.kv("Cultures", strings.Join(cu, ", "))
	}
}

func (g *Game) encRealms(l *encList) {
	b := l.add("rep", "realm", "Reputation", "overview")
	b.p(encRepIntro)
	b.gap()
	b.p("Standings, lowest to highest: " + RepWord(-700) + ", " + RepWord(-400) + ", " + RepWord(0) + ", " + RepWord(300) + ", " + RepWord(700) + ".")
	b.p("{{K|The full list is on the Reputation screen (r).}}")
	var fk []string
	for f := range creatureFactions {
		fk = append(fk, f)
	}
	sort.Strings(fk)
	for _, f := range fk {
		n := g.FactionName(f)
		if n == "" || n == titleCase(f) && creatureFactions[f] == "" {
			continue
		}
		v, ok := g.Rep[f]
		if !ok {
			continue
		}
		b.p(fmt.Sprintf("- %s: %s", n, RepWord(v)))
	}
	for _, s := range g.W.States {
		if !s.Alive || s.ID <= 0 {
			continue
		}
		b.link(fmt.Sprintf("state:%d", s.ID))
		e := l.add(fmt.Sprintf("state:%d", s.ID), "realm", s.FullName, "realm")
		e.kv("Form", s.Form)
		e.kv("Ruler", s.Ruler)
		e.kv("Ideology", s.Ideology)
		e.kv("Species", s.Species)
		if s.Founded != 0 {
			e.kv("Founded", fmt.Sprintf("year %d", s.Founded))
		}
		if v, ok := g.Rep[fmt.Sprintf("state:%d", s.ID)]; ok {
			e.gap()
			e.p(fmt.Sprintf("Your standing: %s {{K|(%d)}}", RepWord(v), v))
		}
		if s.SocialOrder != nil {
			e.kv("Social order", s.SocialOrder.Name)
		}
		var wars []string
		for _, w := range s.Wars {
			if o := g.W.stateByID[w]; o != nil {
				wars = append(wars, o.Name)
				e.link(fmt.Sprintf("state:%d", o.ID))
			}
		}
		e.kv("At war with", strings.Join(wars, ", "))
		if s.Culture >= 0 {
			e.link(fmt.Sprintf("culture:%d", s.Culture))
		}
		e.link(fmt.Sprintf("faith:%d", s.Religion))
		if s.Capital > 0 {
			e.link(fmt.Sprintf("place:b%d", s.Capital))
		}
		e.link("rep")
	}
}

func (g *Game) encCooking(l *encList) {
	b := l.add("cook", "cook", "Cooking", "overview")
	b.p(encCookIntro)
	b.gap()
	for _, t := range cookTagIDs {
		b.p(fmt.Sprintf("- {{%s|%s}}: %s", cookTags[t].Col, t, cookTags[t].Desc(1)))
		b.link("cook:" + t)
	}
	if len(g.Cookbook) > 0 {
		b.h("Your cookbook")
		for _, d := range g.Cookbook {
			b.p(fmt.Sprintf("- %s {{K|(%s)}}", d.Name, strings.Join(d.Tags, ", ")))
		}
	}
	b.link("item:ingredient", "item:food")
	cu, _ := g.encKnown()
	for _, t := range cookTagIDs {
		ct := cookTags[t]
		e := l.add("cook:"+t, "cook", "Cooking tag: "+t, "cooking")
		e.p(fmt.Sprintf("{{%s|%s}}", ct.Col, ct.Name))
		e.kv("Level 1", ct.Desc(1))
		e.kv("Level 2", ct.Desc(2))
		var dishes []string
		for _, c := range g.W.Cultures {
			if !cu[c.ID] {
				continue
			}
			if pk := g.W.cpOf(c.ID); pk != nil && pk.Material != nil && pk.Material.Cuisine != nil {
				for _, d := range pk.Material.Cuisine.Dishes {
					if d.Tag == t {
						dishes = append(dishes, fmt.Sprintf("%s {{K|(%s)}}", d.Name, c.Name))
						e.link(fmt.Sprintf("culture:%d:table", c.ID))
					}
				}
			}
		}
		if len(dishes) > 0 {
			e.h("Dishes of the peoples you know")
			e.bullets(dishes, 10)
		}
		switch t {
		case "ember":
			e.link("dtype:fire")
		case "rime":
			e.link("dtype:cold", "status:frozen")
		case "venom":
			e.link("dtype:poison")
		case "hearty":
			e.link("attr:0")
		case "keen":
			e.link("attr:3", "attr:4")
		case "mending":
			e.link("status:bleeding")
		}
		e.link("cook")
	}
}

func (g *Game) encMagic(l *encList) {
	b := l.add("magic", "magic", g.Sk.PowersName+" and magic", "overview")
	b.p(encMagicIntro)
	b.gap()
	b.p(encArcaneIntro)
	ar := arSkinFor(g.Genre)
	b.gap()
	b.kv("This world's word for the craft", ar.School)
	b.link("arcane:elems", "arcane:forms")
	for _, p := range PowerDefs {
		b.link("power:" + p.ID)
	}
	for _, p := range PowerDefs {
		name := g.Sk.PowerName(p.ID)
		if p.Defect {
			name = titleCase(p.ID)
		}
		sub := g.Sk.PowersName
		if p.Defect {
			sub = "defect"
		}
		e := l.add("power:"+p.ID, "magic", name, sub)
		e.p(p.Desc)
		e.gap()
		kind := "passive"
		if p.Active {
			kind = "active"
		}
		e.kv("Kind", kind)
		e.kv("Cost at creation", fmt.Sprintf("%d points", p.Cost))
		if p.Active {
			e.kv("Cooldown", fmt.Sprintf("%d turns", p.Cool))
			e.kv("Target", p.Target)
		}
		if g.P != nil && g.P.Powers != nil && g.P.Powers[p.ID] > 0 {
			e.p(fmt.Sprintf("{{Y|You have this (rank %d).}}", g.P.Powers[p.ID]))
		}
		for el, pw := range arElemPower {
			if pw == p.ID {
				e.link("arcane:elems")
				_ = el
			}
		}
		e.link("magic")
	}
	fam := ar.Fam
	e := l.add("arcane:elems", "magic", "Elements of "+ar.School, "research")
	e.p("Researched powers take an element. Each leans on an attribute and echoes a basic power.")
	for _, el := range arElems {
		pw := arElemPower[el]
		e.p(fmt.Sprintf("- {{W|%s}} (%s): uses %s, like %s", titleCase(el), arElemWord[fam][el], AttrNames[arElemAttr[el]], g.Sk.PowerName(pw)))
		e.link("power:"+pw, fmt.Sprintf("attr:%d", arElemAttr[el]))
		if _, ok := map[string]bool{"fire": true, "frost": true, "lightning": true, "mind": true, "shadow": true, "death": true}[el]; ok {
			t := map[string]string{"fire": "fire", "frost": "cold", "lightning": "shock", "mind": "psychic", "shadow": "dark", "death": "dark"}[el]
			e.link("dtype:" + t)
		}
	}
	e.link("arcane:forms", "magic")
	e = l.add("arcane:forms", "magic", "Forms of "+ar.School, "research")
	e.p("The shape a researched power takes.")
	for _, f := range arForms {
		e.p(fmt.Sprintf("- {{W|%s}} (%s): %s. Cooldown about %d turns.", titleCase(f), arFormNoun[fam][f], arFormDesc[f], arFormCool[f]))
	}
	e.link("arcane:elems", "magic")
}

// ---------------------------------------------------------------- cultures and faiths

func (g *Game) encCultures(l *encList) {
	cu, fa := g.encKnown()
	for i := range g.W.Cultures {
		c := &g.W.Cultures[i]
		if !cu[c.ID] {
			continue
		}
		id := fmt.Sprintf("culture:%d", c.ID)
		b := l.add(id, "culture", c.Name, "culture")
		b.kv("People", c.Species)
		b.kv("Sphere", c.Sphere)
		b.kv("Food", c.Food)
		b.kv("Dress", c.Dress)
		b.kv("Music", c.Music)
		b.kv("Buildings", c.Architecture)
		if len(c.Values) > 0 {
			b.kv("Values", strings.Join(c.Values, ", "))
		}
		if len(c.Given) > 0 {
			b.kv("Given names", strings.Join(c.Given[:mini(6, len(c.Given))], ", "))
		}
		if len(c.Houses) > 0 {
			b.kv("House names", strings.Join(c.Houses[:mini(6, len(c.Houses))], ", "))
		}
		if len(c.Feasts) > 0 {
			b.h("Feasts")
			for _, f := range c.Feasts {
				b.p(fmt.Sprintf("- {{W|%s}} {{K|(%s)}}", f.Name, f.Kind))
			}
		}
		if c.Species != "" {
			b.link("species:" + c.Species)
		}
		for _, s := range g.W.States {
			if s.Alive && s.ID > 0 && s.Culture == c.ID {
				b.link(fmt.Sprintf("state:%d", s.ID))
			}
		}
		pk := g.W.cpOf(c.ID)
		if pk == nil {
			b.gap()
			b.p("{{K|Little more is known of their ways.}}")
			continue
		}
		g.encCulturePack(l, b, c, pk, id)
	}
	for i := range g.W.Religions {
		r := &g.W.Religions[i]
		if !fa[r.ID] {
			continue
		}
		id := fmt.Sprintf("faith:%d", r.ID)
		b := l.add(id, "faith", r.Name, "faith")
		b.kv("Deity", r.Deity)
		b.kv("Kind", r.Type)
		b.kv("Theology", r.Theology)
		if r.Doctrine != "" {
			b.gap()
			b.p(r.Doctrine)
		}
		for _, s := range g.W.States {
			if s.Alive && s.ID > 0 && s.Religion == r.ID {
				b.link(fmt.Sprintf("state:%d", s.ID))
			}
		}
		if r.ID == g.HomeFaith {
			b.gap()
			b.p("{{Y|The faith of your home.}}")
		}
		if r.Lore != nil {
			b.link(id + ":lore")
			g.encFaithLore(l, r)
		}
		b.link(fmt.Sprintf("faith:%d", r.ID))
		b.e.Links = b.e.Links[:len(b.e.Links)-1] // keep list tidy: no self link
	}
}

func (g *Game) encFaithLore(l *encList, r *ReligionData) {
	lo := r.Lore
	id := fmt.Sprintf("faith:%d", r.ID)
	b := l.add(id+":lore", "faith", r.Name+": lore", "faith lore")
	b.link(id)
	if lo.Afterlife != "" {
		b.h("Afterlife")
		b.p(lo.Afterlife)
	}
	if lo.Scripture != nil {
		b.h("Scripture: " + lo.Scripture.Title)
		b.bullets(lo.Scripture.Excerpts, 6)
	}
	if lo.Liturgy != nil {
		b.h("Liturgy: " + lo.Liturgy.Name)
		b.bullets(lo.Liturgy.Steps, 10)
	}
	if len(lo.Prayers) > 0 {
		b.h("Prayers")
		b.bullets(lo.Prayers, 5)
	}
	if len(lo.Saints) > 0 {
		b.h("Saints")
		for _, s := range lo.Saints {
			b.p(fmt.Sprintf("- {{W|%s}}: %s", s.Name, s.Legend))
		}
	}
	if len(lo.HolyDays) > 0 {
		b.h("Holy days")
		for _, d := range lo.HolyDays {
			b.p(fmt.Sprintf("- {{W|%s}}: %s", d.Name, d.Rite))
		}
	}
	if len(lo.Heresies) > 0 {
		b.h("Heresies")
		for _, h := range lo.Heresies {
			b.p(fmt.Sprintf("- {{W|%s}}: %s", h.Name, h.Claim))
		}
	}
	if len(lo.Blessings) > 0 {
		b.h("Blessings")
		b.bullets(lo.Blessings, 5)
	}
	if len(lo.Sins) > 0 {
		b.h("Sins")
		b.bullets(lo.Sins, 8)
	}
	if len(lo.Observances) > 0 {
		b.h("Observances")
		b.bullets(lo.Observances, 8)
	}
}

func (g *Game) encCulturePack(l *encList, main *encB, c *CultureData, pk *CulturePack, id string) {
	sub := func(key, title string) *encB {
		main.link(id + ":" + key)
		b := l.add(id+":"+key, "culture", c.Name+": "+title, "culture")
		b.link(id)
		return b
	}
	if cp := pk.Customs; cp != nil {
		b := sub("customs", "customs and taboos")
		if cp.Ideals != nil {
			b.kv("On death", cp.Ideals.Death)
			b.kv("On hosts", cp.Ideals.Host)
			b.kv("On names", cp.Ideals.Name)
			b.kv("On wealth", cp.Ideals.Wealth)
		}
		if h := cp.Hospitality; h != nil {
			b.h("Hospitality")
			b.p(fmt.Sprintf("A guest is protected for %d days.", h.GuestRight))
			b.kv("Offered", strings.Join(h.Offers, ", "))
			b.kv("Duties", strings.Join(h.Duties, ", "))
			b.kv("Breach", h.Breach)
		}
		if len(cp.Etiquette) > 0 {
			b.h("Etiquette")
			for _, e := range cp.Etiquette {
				b.p(fmt.Sprintf("- {{K|%s:}} %s", e.Kind, e.Text))
			}
		}
		if len(cp.Taboos) > 0 {
			b.h("Taboos")
			for _, t := range cp.Taboos {
				pen := t.Penalty
				if pen == "" || pen == "none" {
					pen = "scorn"
				}
				b.p(fmt.Sprintf("- %s {{K|(%s)}}", t.Text, pen))
			}
		}
		if len(cp.Rites) > 0 {
			b.h("Rites")
			for _, r := range cp.Rites {
				b.p(fmt.Sprintf("- {{W|%s}}: %s", r.Name, strings.Join(r.Steps, " ")))
			}
		}
		if lw := cp.Law; lw != nil {
			b.h("Law")
			b.kv("Judge", lw.Judge)
			b.kv("Ordeal", lw.Ordeal)
			for _, e := range lw.Entries {
				b.p(fmt.Sprintf("- {{W|%s}}: %s. %s", e.Crime, e.Punish, e.Note))
			}
		}
		if len(cp.Festivals) > 0 {
			b.h("Festival rites")
			for _, f := range cp.Festivals {
				b.p(fmt.Sprintf("- {{W|%s}}: %s", f.Feast, strings.Join(f.Steps, " ")))
			}
		}
	}
	if vp := pk.Voice; vp != nil {
		b := sub("speech", "proverbs and speech")
		b.kv("Register", vp.Register)
		b.kv("Humour", vp.Humour)
		if a := vp.Address; a != nil {
			b.kv("Formal address", a.Formal)
			b.kv("Familiar address", a.Familiar)
		}
		if len(vp.Proverbs) > 0 {
			b.h("Proverbs")
			b.bullets(vp.Proverbs, 12)
		}
		if len(vp.Idioms) > 0 {
			b.h("Idioms")
			for _, i := range vp.Idioms {
				b.p(fmt.Sprintf("- \"%s\": %s", i.Say, i.Meaning))
			}
		}
		if len(vp.Honorifics) > 0 {
			b.h("Honorifics")
			var ks []string
			for k := range vp.Honorifics {
				ks = append(ks, k)
			}
			sort.Strings(ks)
			for _, k := range ks {
				b.p(fmt.Sprintf("- {{K|%s:}} %s", k, vp.Honorifics[k]))
			}
		}
		if len(vp.Greet) > 0 {
			b.h("Greetings")
			b.bullets(vp.Greet, 5)
		}
	}
	if lp := pk.Lang; lp != nil {
		b := sub("lexicon", "language and lexicon")
		if ph := lp.Phonology; ph != nil {
			b.kv("Consonants", ph.Consonants)
			b.kv("Vowels", ph.Vowels)
			b.kv("Word shape", ph.Shape)
		}
		if len(lp.Numerals) > 0 {
			b.kv("Numerals", strings.Join(lp.Numerals, ", "))
		}
		if len(lp.Phrases) > 0 {
			b.h("Phrases")
			for _, p := range lp.Phrases {
				b.p(fmt.Sprintf("- {{W|%s}}: %s {{K|(%s)}}", p.Say, p.Gloss, p.Use))
			}
		}
		if len(lp.Lexicon) > 0 {
			b.h("Lexicon")
			for i, w := range lp.Lexicon {
				if i >= 60 {
					b.p(fmt.Sprintf("{{K|... and %d more}}", len(lp.Lexicon)-60))
					break
				}
				b.p(fmt.Sprintf("- {{W|%s}}: %s", w[0], w[1]))
			}
		}
	}
	if mp := pk.Material; mp != nil {
		b := sub("table", "food, craft and dress")
		if cz := mp.Cuisine; cz != nil {
			b.kv("Staples", strings.Join(cz.Staples, ", "))
			b.kv("Drinks", strings.Join(cz.Drinks, ", "))
			b.kv("Never eaten", strings.Join(cz.Unclean, ", "))
			if len(cz.Dishes) > 0 {
				b.h("Dishes")
				for _, d := range cz.Dishes {
					b.p(fmt.Sprintf("- {{W|%s}} ({{%s|%s}}): %s", d.Name, cookTagCol(d.Tag), d.Tag, d.Note))
					b.link("cook:" + d.Tag)
				}
			}
		}
		if len(mp.Crafts) > 0 {
			b.h("Crafts")
			for _, cr := range mp.Crafts {
				b.p(fmt.Sprintf("- {{W|%s}}: %s", cr.Name, strings.Join(cr.Goods, ", ")))
			}
		}
		if len(mp.Dress) > 0 {
			b.h("Dress")
			var ks []string
			for k := range mp.Dress {
				ks = append(ks, k)
			}
			sort.Strings(ks)
			for _, k := range ks {
				b.p(fmt.Sprintf("- {{K|%s:}} %s", k, mp.Dress[k]))
			}
		}
		if bs := mp.Buildings; bs != nil {
			b.h("Buildings")
			b.kv("Materials", strings.Join(bs.Materials, ", "))
			b.kv("House", bs.House)
			b.kv("Hall", bs.Hall)
			b.kv("Temple", bs.Temple)
		}
		if ms := mp.Music; ms != nil {
			b.h("Music")
			b.kv("Instruments", strings.Join(ms.Instruments, ", "))
			b.kv("Forms", strings.Join(ms.Forms, ", "))
		}
		if len(mp.Games) > 0 {
			b.h("Games")
			for _, gm := range mp.Games {
				b.p(fmt.Sprintf("- {{W|%s}}: %s", gm.Name, gm.Rule))
			}
		}
		b.kv("The land", mp.Land)
	}
	if lt := pk.Literature; lt != nil && (len(lt.Works) > 0 || lt.Taste != "") {
		b := sub("lit", "literature")
		b.kv("Taste", lt.Taste)
		for _, f := range lt.Forms {
			b.p(fmt.Sprintf("- {{W|%s}}: %s", f.Name, f.Rule))
		}
		if len(lt.Works) > 0 {
			b.h("Works")
			for i, w := range lt.Works {
				if i >= 15 {
					b.p(fmt.Sprintf("{{K|... and %d more}}", len(lt.Works)-15))
					break
				}
				b.p(fmt.Sprintf("- {{W|%s}} by %s, year %d {{K|(%s)}}", w.Title, w.Author, w.Year, w.Form))
			}
		}
	}
}

func cookTagCol(t string) string {
	if c, ok := cookTags[t]; ok {
		return c.Col
	}
	return "y"
}

// ---------------------------------------------------------------- places and people

func (g *Game) encPlaces(l *encList) {
	st := g.enc()
	var keys []string
	for k := range st.Places {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		p := st.Places[k]
		if p == nil {
			continue
		}
		b := l.add("place:"+k, "place", p.Name, p.Kind)
		b.kv("Kind", p.Kind)
		if s := g.W.stateByID[p.State]; s != nil && p.State > 0 {
			b.kv("Realm", s.FullName)
			b.link(fmt.Sprintf("state:%d", s.ID))
		}
		if bd := g.W.burgByID[p.Burg]; bd != nil {
			b.kv("Population", fmt.Sprintf("about %d", int(bd.Pop*1000)))
			b.kv("Governor", bd.Governor)
			if bd.Capital {
				b.p("A capital.")
			}
			if bd.Port {
				b.p("A port.")
			}
			b.kv("Layer", bd.Layer)
		}
		if c := g.W.cultByID[p.Cult]; c != nil && p.Cult >= 0 {
			b.kv("People", c.Name)
			b.link(fmt.Sprintf("culture:%d", c.ID))
		}
		if r := g.W.relByID[p.Rel]; r != nil && p.Rel >= 0 {
			b.kv("Faith", r.Name)
			b.link(fmt.Sprintf("faith:%d", r.ID))
		}
		var who []string
		for pk, pe := range st.People {
			if pe != nil && pe.Where == p.Name {
				who = append(who, pe.Name)
				b.link("person:" + pk)
			}
		}
		sort.Strings(who)
		b.kv("People met here", strings.Join(who, ", "))
	}
	keys = keys[:0]
	for k := range st.People {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		p := st.People[k]
		if p == nil {
			continue
		}
		title := p.Name
		sub := "person"
		if p.Title != "" {
			sub = p.Title
		}
		b := l.add("person:"+k, "person", title, sub)
		b.kv("Role", p.Role)
		b.kv("Caste", p.Caste)
		b.kv("House", p.House)
		if s := g.W.stateByID[p.State]; s != nil && p.State > 0 {
			b.kv("Realm", s.FullName)
			b.link(fmt.Sprintf("state:%d", s.ID))
		}
		if p.Spec != "" {
			if sp := g.W.specByID[p.Spec]; sp != nil {
				b.kv("Species", sp.Name)
				b.link("species:" + sp.ID)
			}
		}
		b.kv("Met at", p.Where)
		if p.Sim > 0 {
			for _, pd := range g.W.People {
				if pd.ID == p.Sim {
					b.gap()
					b.p(fmt.Sprintf("%s, born year %d. Renown %d.", titleCase(pd.Role), pd.Born, pd.Renown))
					b.kv("Traits", strings.Join(pd.Traits, ", "))
					if pd.Burg > 0 {
						b.link(fmt.Sprintf("place:b%d", pd.Burg))
					}
				}
			}
		}
		if c := g.W.cultByID[p.Cult]; c != nil && p.Cult >= 0 {
			b.link(fmt.Sprintf("culture:%d", c.ID))
		}
		if r := g.W.relByID[p.Rel]; r != nil && p.Rel >= 0 {
			b.link(fmt.Sprintf("faith:%d", r.ID))
		}
		for pk, pl := range st.Places {
			if pl != nil && pl.Name == p.Where {
				b.link("place:" + pk)
			}
		}
	}
}

// ---------------------------------------------------------------- screen

// encVisible lists entry indices that pass the category and search filter, best match first.
func (g *Game) encVisible() []int {
	st := g.enc()
	cat := encCats[st.cat].ID
	toks := strings.Fields(strings.ToLower(st.buf))
	type sc struct{ i, s int }
	var m []sc
	for i, e := range st.entries {
		if cat != "" && e.Cat != cat {
			continue
		}
		score := 1
		if len(toks) > 0 {
			title := strings.ToLower(Plain(e.Title))
			score = 0
			ok := true
			for _, t := range toks {
				switch {
				case strings.HasPrefix(title, t):
					score += 100
				case strings.Contains(title, t):
					score += 60
				case strings.Contains(e.hay, t):
					score += 10
				default:
					ok = false
				}
			}
			if !ok {
				continue
			}
		}
		m = append(m, sc{i, score})
	}
	if len(toks) > 0 {
		sort.SliceStable(m, func(a, b int) bool { return m[a].s > m[b].s })
	}
	out := make([]int, len(m))
	for i, x := range m {
		out[i] = x.i
	}
	return out
}

// encCur returns the selected entry, resetting the page scroll when the selection changed.
func (g *Game) encCur(vis []int) *encEntry {
	st := g.enc()
	st.sel = clampi(st.sel, 0, maxi(0, len(vis)-1))
	if len(vis) == 0 {
		return nil
	}
	e := st.entries[vis[st.sel]]
	if e.ID != st.lastID {
		st.lastID, st.scroll, st.lsel = e.ID, 0, 0
	}
	return e
}

func (g *Game) encLinks(e *encEntry) []*encEntry {
	st := g.enc()
	var out []*encEntry
	for _, id := range e.Links {
		if i, ok := st.idx[id]; ok && id != e.ID {
			out = append(out, st.entries[i])
		}
	}
	return out
}

func (g *Game) encGo(id string) {
	st := g.enc()
	if st.buf != "" || st.cat != 0 {
		st.buf, st.cat = "", 0
	}
	vis := g.encVisible()
	for n, i := range vis {
		if st.entries[i].ID == id {
			st.sel = n
			return
		}
	}
}

func encKeys(g *Game, k string) {
	st := g.enc()
	if st.entries == nil {
		encOpen(g)
	}
	vis := g.encVisible()
	cur := g.encCur(vis)
	if k == "F1" {
		encClose(g)
		return
	}
	if st.focus == 1 && cur != nil {
		links := g.encLinks(cur)
		switch k {
		case "Escape", "Backspace":
			if n := len(st.hist); n > 0 {
				id := st.hist[n-1]
				st.hist = st.hist[:n-1]
				g.encGo(id)
			} else {
				st.focus = 0
			}
		case "ArrowUp":
			st.lsel = maxi(0, st.lsel-1)
		case "ArrowDown":
			st.lsel = mini(maxi(0, len(links)-1), st.lsel+1)
		case "PageUp":
			st.scroll = maxi(0, st.scroll-10)
		case "PageDown":
			st.scroll += 10
		case "Enter":
			if st.lsel >= 0 && st.lsel < len(links) {
				st.hist = append(st.hist, cur.ID)
				if len(st.hist) > 30 {
					st.hist = st.hist[1:]
				}
				g.encGo(links[st.lsel].ID)
			}
		}
		return
	}
	switch k {
	case "Escape":
		if st.buf != "" {
			st.buf, st.sel = "", 0
		} else {
			encClose(g)
		}
	case "Enter":
		if cur != nil {
			st.focus, st.lsel = 1, 0
		}
	case "Backspace":
		if rs := []rune(st.buf); len(rs) > 0 {
			st.buf = string(rs[:len(rs)-1])
			st.sel = 0
		}
	case "ArrowUp":
		st.sel = maxi(0, st.sel-1)
	case "ArrowDown":
		st.sel = mini(maxi(0, len(vis)-1), st.sel+1)
	case "PageUp":
		st.sel = maxi(0, st.sel-10)
	case "PageDown":
		st.sel = mini(maxi(0, len(vis)-1), st.sel+10)
	case "Home":
		st.sel = 0
	case "End":
		st.sel = maxi(0, len(vis)-1)
	case "Tab", "ArrowRight":
		st.cat, st.sel = (st.cat+1)%len(encCats), 0
	case "ArrowLeft":
		st.cat, st.sel = (st.cat+len(encCats)-1)%len(encCats), 0
	default:
		if rs := []rune(k); len(rs) == 1 && rs[0] >= 32 && rs[0] < 127 && len(st.buf) < 40 {
			st.buf += k
			st.sel = 0
		}
	}
}

func encDraw(g *Game) {
	s := g.sc
	st := g.enc()
	if st.entries == nil {
		encOpen(g)
	}
	x, y, w, h := g.panel("Encyclopedia")
	right := x + w
	// category strip
	cx := x
	for i, c := range encCats {
		col := "K"
		if i == st.cat {
			col = "Y"
		}
		cx = s.TextB(cx, y, "{{"+col+"|"+c.Name+"}} ", CGray, CBgDark, right)
	}
	s.TextB(x, y+1, "{{W|> }}{{W|"+st.buf+"}}{{Y|_}}", CGray, CBgDark, right)
	for xx := x; xx < right; xx++ {
		s.Put(xx, y+2, '─', CDarkGray, CBgDark)
	}
	vis := g.encVisible()
	cur := g.encCur(vis)
	s.TextB(right-14, y+1, fmt.Sprintf("{{K|%d entries}}", len(vis)), CGray, CBgDark, right)
	top := y + 3
	rows := h - 5
	listW := 30
	for yy := top; yy < top+rows; yy++ {
		s.Put(x+listW+1, yy, '│', CDarkGray, CBgDark)
	}
	first := 0
	if st.sel >= rows {
		first = st.sel - rows + 1
	}
	if len(vis) == 0 {
		s.TextB(x, top, "{{K|Nothing matches.}}", CGray, CBgDark, x+listW)
	}
	for r := 0; r < rows && first+r < len(vis); r++ {
		e := st.entries[vis[first+r]]
		bg, col := CBgDark, "y"
		if first+r == st.sel {
			bg, col = CBgSel, "W"
		}
		s.Fill(x, top+r, listW, 1, bg)
		t := []rune(Plain(e.Title))
		if len(t) > listW-2 {
			t = append(t[:listW-3], '~')
		}
		s.TextB(x, top+r, " {{"+col+"|"+string(t)+"}}", CGray, bg, x+listW)
	}
	// detail
	dx := x + listW + 3
	dw := right - dx
	if cur != nil {
		s.TextB(dx, top, "{{Y|"+cur.Title+"}}  {{K|"+cur.Sub+"}}", CGray, CBgDark, right)
		lines := []string{}
		for _, p := range cur.Body {
			lines = append(lines, Wrap(p, dw)...)
		}
		links := g.encLinks(cur)
		linkRows := 0
		if len(links) > 0 {
			linkRows = mini(len(links), 4) + 1
		}
		view := rows - 2 - linkRows
		if view < 3 {
			view = 3
		}
		maxScroll := maxi(0, len(lines)-view)
		st.scroll = clampi(st.scroll, 0, maxScroll)
		for r := 0; r < view && st.scroll+r < len(lines); r++ {
			s.TextB(dx, top+2+r, lines[st.scroll+r], CGray, CBgDark, right)
		}
		if maxScroll > 0 {
			s.TextB(right-8, top, fmt.Sprintf("{{K|%d/%d}}", st.scroll+1, maxScroll+1), CGray, CBgDark, right)
		}
		if linkRows > 0 {
			ly := top + rows - linkRows
			hdr := "See also"
			if st.focus == 1 {
				hdr = "See also (Enter follows)"
			}
			s.TextB(dx, ly, "{{K|"+hdr+"}}", CGray, CBgDark, right)
			f := 0
			if st.lsel >= 4 {
				f = st.lsel - 3
			}
			for r := 0; r < 4 && f+r < len(links); r++ {
				col, mark := "C", "  "
				if st.focus == 1 && f+r == st.lsel {
					col, mark = "W", "> "
				}
				s.TextB(dx, ly+1+r, mark+"{{"+col+"|"+Plain(links[f+r].Title)+"}} {{K|"+links[f+r].Sub+"}}", CGray, CBgDark, right)
			}
		}
	}
	foot := encKeysList
	if st.focus == 1 {
		foot = encKeysDetail
	}
	s.TextB(x, y+h-1, "{{K|"+foot+"}}", CGray, CBgDark, right)
}
