package engine

// Who people are (prefix lv, agent A). Generates the people of each burg (Realmforge notables and townsfolk with
// families, traits, goals, fears, secrets and life lines), binds the actors of town zones to them, renders bios,
// gives talk topics, remembers what you did and shows the People screen. Pure side rngs: never g.R.
// State: lvSt (lives_types.go, "lv") and g.ext("lvb") for actor bindings and talk counters (both planet-local).

import (
	"fmt"
	"math"
	"sort"
	"strings"
)

func init() {
	unLocalKeys = append(unLocalKeys, "lvb")
	lvEnsureTown = lvEnsure
	lvPersonOf = lvPersonFor
	lvBioText = lvBio
	enterHooks = append(enterHooks, lvEnter)
	talkHooks = append(talkHooks, lvTalk)
	killHooks = append(killHooks, lvKill)
	modeKeys["lv"] = lvKeys
	modeDraws["lv"] = func(g *Game) { g.lvDraw() }
}

// lvbState: actor bindings and counters. UI fields are not saved.
type lvbState struct {
	Bind     map[int]string  `json:"b,omitempty"` // actor id -> person key
	Talks    map[string]int  `json:"t,omitempty"` // person key -> days talked
	Last     map[string]int  `json:"l,omitempty"` // person key -> last day talked
	Firsts   map[string]bool `json:"f,omitempty"`
	sel      int
	bio      string
	scroll   int
	fromTalk bool
}

func lvb(g *Game) *lvbState {
	s := g.ext("lvb", func() any { return &lvbState{} }).(*lvbState)
	if s.Bind == nil {
		s.Bind = map[int]string{}
	}
	if s.Talks == nil {
		s.Talks = map[string]int{}
	}
	if s.Last == nil {
		s.Last = map[string]int{}
	}
	if s.Firsts == nil {
		s.Firsts = map[string]bool{}
	}
	return s
}

func lvRng(g *Game, key string) *Rng { return NewRng(g.W.Seed + ":lv:" + key) }

func lvBurgName(g *Game, id int) string {
	if b := g.W.burgByID[id]; b != nil {
		return b.Name
	}
	return "the road"
}

func lvName(g *Game, key string) string {
	if p := lvGet(g, key); p != nil {
		return p.Name
	}
	return "someone"
}

func lvLower(s string) string {
	if s == "" {
		return s
	}
	return strings.ToLower(s[:1]) + s[1:]
}

func lvWord(table [][2]interface{}, v int) string {
	for _, e := range table {
		if v < e[0].(int) {
			return e[1].(string)
		}
	}
	return table[len(table)-1][1].(string)
}

func lvMoodWord(p *lvPerson) string    { return lvWord(lvMoodWords, p.Mood) }
func lvOpinionWord(p *lvPerson) string { return lvWord(lvOpinionWords, p.Opinion) }

func lvTrunc(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n-1]) + "."
}

// ---------------------------------------------------------------- generation

func lvTraitsOf(r *Rng, words []string) [5]int {
	var t [5]int
	for i := range t {
		t[i] = 50 + r.Int(-12, 12)
	}
	for _, w := range words {
		w = strings.ToLower(w)
		if d, ok := lvTraitWords[w]; ok {
			for i := range t {
				t[i] += d[i]
			}
		} else {
			h := int(HashStr(w) % 5)
			t[h] += 10 + int(HashStr(w+"x")%15)
		}
	}
	for i := range t {
		t[i] = clampi(t[i], 3, 97)
	}
	return t
}

func lvBase(g *Game, bd *BurgData, cu *CultureData, key, name, trade string, born int, female bool, r *Rng) *lvPerson {
	sp := "human"
	if s := g.speciesOf(cu); s != nil {
		sp = s.ID
	}
	faith := 0
	if n := len(g.W.Religion); n > 0 {
		faith = g.W.Religion[clampi(bd.P, 0, n-1)]
	}
	cid := 0
	if cu != nil {
		cid = cu.ID
	}
	p := &lvPerson{Key: key, Name: name, Role: trade, Job: trade, Species: sp, Culture: cid, State: bd.State, Faith: faith, Female: female,
		Born: born, Alive: true, Home: bd.ID, Burg: bd.ID, Mood: r.Int(-20, 40)}
	p.Trait = lvTraitsOf(r, nil)
	w := lvTradeWealth[trade]
	p.Wealth = w * (50 + r.Int(0, 150)) / 100
	return p
}

func lvGiven(g *Game, cu *CultureData, r *Rng) string {
	if cu != nil && len(cu.Given) > 0 {
		return Pick(r, cu.Given)
	}
	return Pick(r, []string{"Ash", "Bren", "Cael", "Doru", "Esk", "Fen", "Garo", "Hale", "Ilm", "Joru"})
}

func lvLink(a, b *lvPerson, relA, relB string) {
	a.Kin = lvAddKey(a.Kin, b.Key)
	b.Kin = lvAddKey(b.Kin, a.Key)
	if a.Rel == nil {
		a.Rel = map[string]string{}
	}
	if b.Rel == nil {
		b.Rel = map[string]string{}
	}
	a.Rel[b.Key] = relA
	b.Rel[a.Key] = relB
}

func lvAddKey(l []string, k string) []string {
	for _, x := range l {
		if x == k {
			return l
		}
	}
	return append(l, k)
}

func lvHas(l []string, k string) bool {
	for _, x := range l {
		if x == k {
			return true
		}
	}
	return false
}

func lvSexWord(p *lvPerson, m, f string) string {
	if p.Female {
		return f
	}
	return m
}

// lvEnsure creates the people of a burg once.
func lvEnsure(g *Game, burg int) {
	st := lvSt(g)
	if st.Seeded[burg] || g.W == nil {
		return
	}
	bd := g.W.burgByID[burg]
	if bd == nil {
		return
	}
	st.Seeded[burg] = true
	cu := g.W.cultByID[bd.Culture]
	y0 := g.W.Year
	r := lvRng(g, fmt.Sprintf("town:%d", burg))
	var town []*lvPerson

	// Realmforge notables who live here.
	for i := range g.W.People {
		pp := &g.W.People[i]
		if pp.Burg != burg || !pp.Alive {
			continue
		}
		key := fmt.Sprintf("s%d", pp.ID)
		if st.People[key] != nil {
			continue
		}
		pr := lvRng(g, "n:"+key)
		p := &lvPerson{Key: key, Name: pp.Name, Role: pp.Role, Species: pp.Species, Culture: pp.Culture, State: pp.State, Faith: pp.Religion,
			Female: HashStr(key)%2 == 0, Born: pp.Born, Alive: true, Home: burg, Burg: burg, Job: nonEmpty(pp.Title, pp.Role), Mood: pr.Int(-10, 40)}
		if p.Species == "" {
			p.Species = "human"
		}
		p.Trait = lvTraitsOf(pr, pp.Traits)
		switch pp.Role {
		case "ruler":
			p.Wealth = 3000 + pp.Renown*20
		case "general", "hero":
			p.Wealth = 500 + pp.Renown*5
		default:
			p.Wealth = 150 + pp.Renown*3
		}
		p.Goal, p.GoalID = lvPickGoal(pr, p, pp.Traits, nil, g)
		p.GoalPct = pr.Int(5, 70)
		lvPickFear(pr, p)
		p.Secret = lvPickSecret(g, pr, p, nil)
		p.addLife(pp.Born, "birth", "Born in the year "+fmt.Sprint(pp.Born)+".")
		if pp.Title != "" {
			p.addLife(clampi(pp.Born+pr.Int(20, 40), pp.Born, y0), "job", "Rose to be "+pp.Title+" at "+bd.Name+".")
		}
		st.People[key] = p
		town = append(town, p)
	}

	// Townsfolk.
	size := clampi(int(math.Log2(bd.Pop+1)), 0, 8)
	target := 8 + size
	forced := []string{"innkeeper", "baker", "smith", "healer", "priest", "guard", "merchant", "guard"}
	for n := clampi(int(bd.Pop/20), 0, 4); n > 0; n-- {
		forced = append(forced, "guard")
	}
	pool := lvTradePool(bd, g)
	used := map[string]bool{}
	next := 0
	nextKey := func() string { next++; return fmt.Sprintf("g%d:%d", burg, next) }
	adults := 0
	var gen []*lvPerson
	trade := func(age int) string {
		if age < 14 {
			return "child"
		}
		if len(forced) > 0 {
			t := forced[0]
			forced = forced[1:]
			return t
		}
		return Pick(r, pool)
	}
	mk := func(house string, age int, female bool, tr string) *lvPerson {
		nm := lvGiven(g, cu, r)
		if house != "" {
			nm += " " + house
		}
		p := lvBase(g, bd, cu, nextKey(), nm, tr, y0-age, female, r)
		st.People[p.Key] = p
		gen = append(gen, p)
		town = append(town, p)
		if age >= 14 {
			adults++
		}
		return p
	}
	for adults < target {
		house := g.pickHouse(cu, &Actor{Name: "X"}, used, r)
		k := r.Int(0, 99)
		switch {
		case k < 45: // couple with children
			age := r.Int(20, 66)
			hf := r.Chance(0.5)
			h := mk(house, age, hf, trade(age))
			s := mk(house, clampi(age+r.Int(-6, 6), 18, 80), !hf, trade(30))
			h.Spouse, s.Spouse = s.Key, h.Key
			lvLink(h, s, lvSexWord(s, "husband", "wife"), lvSexWord(h, "husband", "wife"))
			var kids []*lvPerson
			if age >= 24 {
				for n := r.Int(0, 3); n > 0; n-- {
					ka := r.Int(0, lvMin(age-18, 32))
					f := r.Chance(0.5)
					c := mk(house, ka, f, trade(ka))
					kids = append(kids, c)
					for _, par := range []*lvPerson{h, s} {
						lvLink(par, c, lvSexWord(c, "son", "daughter"), lvSexWord(par, "father", "mother"))
					}
					for _, o := range kids[:len(kids)-1] {
						lvLink(c, o, lvSexWord(o, "brother", "sister"), lvSexWord(c, "brother", "sister"))
					}
				}
			}
		case k < 65: // single adult
			age := r.Int(18, 70)
			mk(house, age, r.Chance(0.5), trade(age))
		case k < 80: // widowed with a grown child
			age := r.Int(48, 78)
			w := mk(house, age, r.Chance(0.55), trade(age))
			c := mk(house, age-r.Int(20, 30), r.Chance(0.5), trade(30))
			lvLink(w, c, lvSexWord(c, "son", "daughter"), lvSexWord(w, "father", "mother"))
			w.addLife(clampi(c.Born+r.Int(5, 20), c.Born, y0), "death", "Lost a spouse and raised on alone.")
		case k < 92: // siblings
			age := r.Int(20, 60)
			a := mk(house, age, r.Chance(0.5), trade(age))
			b := mk(house, clampi(age+r.Int(-8, 8), 16, 75), r.Chance(0.5), trade(30))
			lvLink(a, b, lvSexWord(b, "brother", "sister"), lvSexWord(a, "brother", "sister"))
		default: // old hand
			mk(house, r.Int(62, 80), r.Chance(0.5), trade(65))
		}
	}

	// Friends, rivals, bosses among adults.
	var ad []*lvPerson
	for _, p := range town {
		if lvAgeNow(p, y0) >= 14 {
			ad = append(ad, p)
		}
	}
	for _, p := range gen {
		if lvAgeNow(p, y0) < 14 || len(ad) < 3 {
			continue
		}
		if r.Chance(0.65) {
			var best *lvPerson
			for t := 0; t < 4; t++ {
				c := Pick(r, ad)
				if c == p || lvHas(p.Kin, c.Key) || lvHas(p.Friends, c.Key) || lvHas(p.Rivals, c.Key) {
					continue
				}
				if best == nil || abs(c.Trait[3]-p.Trait[3]) < abs(best.Trait[3]-p.Trait[3]) {
					best = c
				}
			}
			if best != nil {
				p.Friends = append(p.Friends, best.Key)
				best.Friends = lvAddKey(best.Friends, p.Key)
			}
		}
		if r.Chance(0.3) {
			var cand *lvPerson
			for t := 0; t < 8; t++ {
				c := Pick(r, ad)
				if c == p || lvHas(p.Kin, c.Key) || lvHas(p.Friends, c.Key) || lvHas(p.Rivals, c.Key) {
					continue
				}
				if c.Role == p.Role || cand == nil {
					cand = c
					if c.Role == p.Role {
						break
					}
				}
			}
			if cand != nil {
				p.Rivals = append(p.Rivals, cand.Key)
				cand.Rivals = lvAddKey(cand.Rivals, p.Key)
			}
		}
	}
	for _, p := range gen {
		age := lvAgeNow(p, y0)
		if age < 14 || p.Role == "child" {
			continue
		}
		if age < 24 {
			for _, c := range gen {
				if c != p && c.Role == p.Role && lvAgeNow(c, y0) > age+10 {
					p.Boss = c.Key
					break
				}
			}
		}
		if p.Boss == "" && p.Role == "guard" {
			for _, c := range gen {
				if c.Role == "sergeant" || c.Role == "guard" && c != p && lvAgeNow(c, y0) > age+5 {
					p.Boss = c.Key
					break
				}
			}
		}
		if p.Boss == "" && r.Chance(0.2) {
			for _, c := range town {
				if strings.HasPrefix(c.Key, "s") && (c.Role == "ruler" || c.Role == "general") {
					p.Boss = c.Key
					break
				}
			}
		}
	}
	// Inner life: goals, fears, secrets, life lines.
	for _, p := range gen {
		age := lvAgeNow(p, y0)
		if age < 14 {
			p.Goal, p.GoalID = Pick(r, []string{"play out until dark", "be allowed to stay up late", "see the ships come in", "have a pet of their own"}), "family"
			p.Fear = Pick(r, []string{"the dark", "being left alone", "thunder", "the old well"})
			p.addLife(p.Born, "birth", "Born in "+bd.Name+".")
			lvKinLife(g, p, y0)
			continue
		}
		p.Goal, p.GoalID = lvPickGoal(r, p, nil, ad, g)
		p.GoalPct = r.Int(0, 60)
		lvPickFear(r, p)
		if r.Chance(0.45) {
			p.Secret = lvPickSecret(g, r, p, ad)
		}
		lvLifeLines(g, p, bd, r, y0)
	}
	lvRelink(g, burg)
	if lvKnowledgeOf != nil {
		for _, p := range town {
			if k := lvKnowledgeOf(g, p); len(k) > 0 {
				p.Knows = append(p.Knows, k...)
			}
		}
	}
}

func lvMin(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func lvAgeNow(p *lvPerson, year int) int { return maxi(0, year-p.Born) }

// lvRelink is a hook point for later: friends and rivals are already mutual.
func lvRelink(g *Game, burg int) {}

func lvTradePool(bd *BurgData, g *Game) []string {
	var pool []string
	for i, n := range g.W.SectorNames {
		if i >= len(bd.Sectors) {
			break
		}
		tr := lvSectorTrades[n]
		cnt := int(bd.Sectors[i]*40 + 0.5)
		for j := 0; j < cnt && len(tr) > 0; j++ {
			pool = append(pool, tr[j%len(tr)])
		}
	}
	pool = append(pool, lvCommonTrades...)
	return pool
}

func lvPickGoal(r *Rng, p *lvPerson, traits []string, others []*lvPerson, g *Game) (string, string) {
	var opts []string
	for _, t := range traits {
		opts = append(opts, lvTraitGoals[strings.ToLower(t)]...)
	}
	for i := 0; i < 3; i++ {
		opts = append(opts, lvRoleGoals[p.Role]...)
	}
	if p.Trait[4] >= 65 {
		opts = append(opts, "safety", "safety")
	}
	if p.Trait[3] >= 70 {
		opts = append(opts, "family", "love")
	}
	if p.Trait[0] >= 70 {
		opts = append(opts, "knowledge")
	}
	if p.Trait[3] <= 30 {
		opts = append(opts, "revenge", "wealth")
	}
	opts = append(opts, lvAllGoals...)
	gid := Pick(r, opts)
	txt := Pick(r, lvGoalText[gid])
	if strings.Contains(txt, "%s") {
		who := "someone in town"
		if gid == "revenge" && len(p.Rivals) > 0 {
			who = lvName(g, p.Rivals[0])
		} else if len(others) > 0 {
			for t := 0; t < 6; t++ {
				c := Pick(r, others)
				if c != p && c.Key != p.Spouse && !lvHas(p.Kin, c.Key) {
					who = c.Name
					break
				}
			}
		}
		txt = fmt.Sprintf(txt, who)
	}
	return txt, gid
}

func lvPickFear(r *Rng, p *lvPerson) {
	opts := append([]string{}, lvFearByGoal[p.GoalID]...)
	opts = append(opts, lvFearByGoal[p.GoalID]...)
	opts = append(opts, lvFearsGeneric...)
	p.Fear = Pick(r, opts)
}

func lvPickSecret(g *Game, r *Rng, p *lvPerson, others []*lvPerson) string {
	if len(p.Rivals) > 0 && r.Chance(0.2) {
		return "Is quietly working to ruin " + lvName(g, p.Rivals[0]) + "."
	}
	d := Pick(r, lvSecrets)
	if d.Name {
		who := "a neighbour"
		for t := 0; t < 8 && len(others) > 0; t++ {
			c := Pick(r, others)
			if c != p && !lvHas(p.Kin, c.Key) {
				who = c.Name
				break
			}
		}
		return fmt.Sprintf(d.Text, who)
	}
	return d.Text
}

func lvPlaceYear(r *Rng, p *lvPerson, lo, hi int) int {
	if hi < lo {
		hi = lo
	}
	return clampi(r.Int(lo, hi), p.Born, hi)
}

func lvKinLife(g *Game, p *lvPerson, y0 int) {
	for _, k := range p.Kin {
		q := lvGet(g, k)
		if q == nil {
			continue
		}
		switch p.Rel[k] {
		case "son", "daughter":
			p.addLife(q.Born, "child", fmt.Sprintf("A %s, %s, was born.", p.Rel[k], q.Name))
		case "father", "mother":
			if q.Born < p.Born {
				for i := range p.Life {
					if p.Life[i].Kind == "birth" && !strings.Contains(p.Life[i].Text, "child of") {
						p.Life[i].Text = strings.TrimSuffix(p.Life[i].Text, ".") + ", child of " + q.Name + "."
						break
					}
				}
			}
		}
	}
	sort.SliceStable(p.Life, func(i, j int) bool { return p.Life[i].Y < p.Life[j].Y })
}

func lvLifeLines(g *Game, p *lvPerson, bd *BurgData, r *Rng, y0 int) {
	age := lvAgeNow(p, y0)
	from := ""
	if age > 25 && r.Chance(0.25) {
		var cands []*BurgData
		for i := range g.W.Burgs {
			b := &g.W.Burgs[i]
			if b.ID != bd.ID && b.State == bd.State {
				cands = append(cands, b)
			}
		}
		if len(cands) > 0 {
			from = Pick(r, cands).Name
		}
	}
	birth := "Born in " + nonEmpty(from, bd.Name) + "."
	p.addLife(p.Born, "birth", birth)
	if from != "" {
		t := Pick(r, lvMoveLines)
		if strings.HasPrefix(t, "Followed") {
			p.addLife(lvPlaceYear(r, p, p.Born+16, y0-2), "move", fmt.Sprintf(t, from, bd.Name))
		} else {
			p.addLife(lvPlaceYear(r, p, p.Born+16, y0-2), "move", fmt.Sprintf(t, bd.Name, from))
		}
	}
	if p.Role != "child" {
		master := "a relative"
		if p.Boss != "" {
			master = lvName(g, p.Boss)
		}
		p.addLife(p.Born+r.Int(12, 16), "job", fmt.Sprintf(Pick(r, lvApprentice), p.Role, master))
	}
	if p.Spouse != "" {
		if s := lvGet(g, p.Spouse); s != nil {
			y := maxi(p.Born, s.Born) + r.Int(18, 26)
			p.addLife(lvMin(y, y0), "marriage", "Married "+s.Name+".")
		}
	}
	if age >= 25 && r.Chance(0.55) {
		p.addLife(lvPlaceYear(r, p, p.Born+16, y0-1), "illness", Pick(r, lvLossLines))
	}
	if age >= 30 && r.Chance(0.55) {
		p.addLife(lvPlaceYear(r, p, p.Born+20, y0-1), "job", Pick(r, lvWorkLines))
	}
	if len(p.Friends) > 0 {
		p.addLife(lvPlaceYear(r, p, p.Born+16, y0-1), "friend", "Became close friends with "+lvName(g, p.Friends[0])+".")
	}
	if len(p.Rivals) > 0 {
		reason := Pick(r, []string{"over a debt", "over a customer", "over a slight at a wedding", "over a boundary", "over something said in drink"})
		p.addLife(lvPlaceYear(r, p, p.Born+16, y0-1), "feud", "Fell out with "+lvName(g, p.Rivals[0])+" "+reason+".")
	}
	lvKinLife(g, p, y0)
}

// ---------------------------------------------------------------- real events

func lvRealEvents(g *Game, p *lvPerson) []lvEvent {
	if g.W == nil {
		return nil
	}
	end := lvYear(g)
	if !p.Alive && p.Died > 0 {
		end = p.Died
	}
	var out []lvEvent
	short := func(s string) string {
		if len(s) > 120 {
			s = s[:117] + "..."
		}
		return s
	}
	skip := map[string]bool{"misc": true, "god": true, "quest": true, "luminary": true}
	bd := g.W.burgByID[p.Home]
	seen := map[int]bool{}
	if strings.HasPrefix(p.Key, "s") {
		for i, e := range g.W.Events {
			if e.Y >= p.Born && e.Y <= end && p.Name != "" && strings.Contains(e.Text, p.Name) {
				out = append(out, lvEvent{Y: e.Y, Kind: "real", Text: short(e.Text)})
				seen[i] = true
				if len(out) >= 8 {
					break
				}
			}
		}
	}
	h := HashStr(p.Key)
	if bd != nil {
		var ids []int
		for _, i := range g.W.eventsAt[bd.P] {
			e := g.W.Events[i]
			if !seen[i] && e.Y >= p.Born+3 && e.Y <= end && !skip[e.Kind] {
				ids = append(ids, i)
			}
		}
		for n, i := range ids {
			if len(ids) > 4 && (HashStr(fmt.Sprint(h, i))+uint64(n))%uint64(len(ids)) >= 4 {
				continue
			}
			e := g.W.Events[i]
			out = append(out, lvEvent{Y: e.Y, Kind: "real", Text: short(e.Text)})
			if len(out) >= 12 {
				break
			}
		}
	}
	// a couple of realm-wide events
	cnt := 0
	for i, e := range g.W.Events {
		if cnt >= 2 {
			break
		}
		if e.St == p.State && p.State > 0 && e.Y >= p.Born+5 && e.Y <= end && (e.Kind == "war" || e.Kind == "peace" || e.Kind == "coup" || e.Kind == "disaster") && !seen[i] {
			if HashStr(fmt.Sprint(h, "st", i))%7 == 0 {
				out = append(out, lvEvent{Y: e.Y, Kind: "real", Text: "In the realm: " + short(e.Text)})
				cnt++
			}
		}
	}
	return out
}

// ---------------------------------------------------------------- bio

func lvPersonality(p *lvPerson) string {
	var w []string
	for i, v := range p.Trait {
		if v >= 66 {
			w = append(w, lvTraitWordsLowHigh[i][1])
		} else if v <= 34 {
			w = append(w, lvTraitWordsLowHigh[i][0])
		}
	}
	if len(w) == 0 {
		return "even-tempered and hard to read"
	}
	return strings.Join(w, ", ")
}

func lvSecretKnown(p *lvPerson) bool {
	if p.Secret == "" {
		return false
	}
	if p.Opinion >= 60 {
		return true
	}
	for _, e := range p.Life {
		if e.Kind == "secret" {
			return true
		}
	}
	return false
}

// lvRevealSecret records that you learned their secret (shown on the bio page from then on).
func lvRevealSecret(g *Game, p *lvPerson) {
	if p == nil || p.Secret == "" {
		return
	}
	for _, e := range p.Life {
		if e.Kind == "secret" {
			return
		}
	}
	p.addLife(lvYear(g), "secret", "Confided a secret to the wanderer.")
}

func lvRelName(g *Game, p *lvPerson, key string) string {
	q := lvGet(g, key)
	if q == nil {
		return "someone"
	}
	s := q.Name
	if q.Role != "" {
		s += ", " + q.Role
	}
	if !q.Alive {
		s += " (dead)"
	}
	return s
}

func lvKinLabel(p *lvPerson, q *lvPerson) string {
	if l := p.Rel[q.Key]; l != "" {
		return l
	}
	d := q.Born - p.Born
	switch {
	case d <= -15:
		return lvSexWord(q, "elder kinsman", "elder kinswoman")
	case d >= 15:
		return lvSexWord(q, "younger kinsman", "younger kinswoman")
	}
	return lvSexWord(q, "brother", "sister")
}

func lvBio(g *Game, p *lvPerson) []string {
	year := lvYear(g)
	var L []string
	age := p.lvAge(year)
	title := p.Role
	if p.Job != "" && p.Job != p.Role {
		title = p.Job
	}
	L = append(L, fmt.Sprintf("{{Y|%s}}  {{K|%s}}", p.Name, title))
	who := ""
	if sp := g.W.specByID[p.Species]; sp != nil {
		who = strings.ToLower(sp.Name)
	}
	if c := g.W.cultByID[p.Culture]; c != nil {
		who = strings.TrimSpace(who + " of the " + c.Name)
	}
	if p.Alive {
		L = append(L, fmt.Sprintf("%s, %d years old (born in the year %d).", titleCase(nonEmpty(who, "person")), age, p.Born))
	} else {
		L = append(L, fmt.Sprintf("{{R|Dead}} in the year %d at %d. %s", p.Died, age, nonEmpty(p.Cause, "")))
	}
	where := "lives in " + lvBurgName(g, p.Home)
	if p.Alive && p.Burg != p.Home {
		if p.Burg == 0 {
			where += ", now on the road"
		} else {
			where += ", now in " + lvBurgName(g, p.Burg)
		}
	}
	line := "They " + where + "."
	if st := g.W.stateByID[p.State]; st != nil && st.ID > 0 {
		line += " Subject of " + st.Name + "."
	}
	if rel := g.W.relByID[p.Faith]; rel != nil {
		line += " Follows the " + rel.Name + "."
	}
	L = append(L, line)
	if p.Alive {
		L = append(L, fmt.Sprintf("Mood: {{C|%s}}.  Toward you: {{C|%s}} (%+d).", lvMoodWord(p), lvOpinionWord(p), p.Opinion))
	}
	L = append(L, "Personality: "+lvPersonality(p)+".")
	if p.Spouse != "" {
		L = append(L, "Spouse: "+lvRelName(g, p, p.Spouse)+".")
	}
	if len(p.Kin) > 0 {
		var ks []string
		for _, k := range p.Kin {
			if q := lvGet(g, k); q != nil && k != p.Spouse {
				ks = append(ks, fmt.Sprintf("%s (%s)", lvRelName(g, p, k), lvKinLabel(p, q)))
			}
		}
		if len(ks) > 0 {
			L = append(L, "Family: "+strings.Join(ks, "; ")+".")
		}
	}
	if len(p.Friends) > 0 {
		var fs []string
		for _, k := range p.Friends {
			fs = append(fs, lvRelName(g, p, k))
		}
		L = append(L, "Friends: "+strings.Join(fs, "; ")+".")
	}
	if len(p.Rivals) > 0 {
		var fs []string
		for _, k := range p.Rivals {
			fs = append(fs, lvRelName(g, p, k))
		}
		L = append(L, "Rivals: "+strings.Join(fs, "; ")+".")
	}
	if p.Boss != "" {
		L = append(L, "Answers to: "+lvRelName(g, p, p.Boss)+".")
	}
	if p.Goal != "" && p.Alive {
		L = append(L, fmt.Sprintf("Wants to: %s {{K|(%d%% there)}}.", p.Goal, p.GoalPct))
	}
	if p.Fear != "" && p.Alive {
		L = append(L, "Fears: "+p.Fear+".")
	}
	if lvSecretKnown(p) {
		L = append(L, "{{R|Secret:}} "+p.Secret)
	}
	L = append(L, "", "{{W|Life}}")
	ev := append([]lvEvent{}, p.Life...)
	ev = append(ev, lvRealEvents(g, p)...)
	sort.SliceStable(ev, func(i, j int) bool { return ev[i].Y < ev[j].Y })
	for _, e := range ev {
		if e.Kind == "secret" {
			continue
		}
		c := "K"
		if e.Kind == "real" {
			c = "c"
		}
		L = append(L, fmt.Sprintf(" {{%s|%4d}}  %s", c, e.Y, e.Text))
	}
	if len(p.Mem) > 0 {
		L = append(L, "", "{{W|What they remember of you}}")
		start := maxi(0, len(p.Mem)-6)
		for _, m := range p.Mem[start:] {
			c := "G"
			if m.Weight < 0 {
				c = "R"
			}
			L = append(L, fmt.Sprintf(" {{%s|%+d}}  %s", c, m.Weight, m.Text))
		}
	}
	return L
}

// ---------------------------------------------------------------- binding actors to people

func lvEligible(a *Actor) bool {
	if a == nil || a.HP <= 0 || a.Tpl != "npc" || a.Boss || a.SimMon != 0 || a.Angry || a.Order != "" || a.Role == "construct" {
		return false
	}
	switch a.AI {
	case "town", "guard", "still", "wander", "":
		return true
	}
	return false
}

func lvTitleOf(p *lvPerson) string {
	if strings.HasPrefix(p.Key, "s") {
		return nonEmpty(p.Job, p.Role)
	}
	if p.Role == "sergeant" {
		return "guard sergeant"
	}
	return p.Role
}

func lvApply(g *Game, a *Actor, p *lvPerson, zn *Zone) {
	b := lvb(g)
	b.Bind[a.ID] = p.Key
	p.Zone = zn.Key
	if strings.HasPrefix(p.Key, "s") {
		return
	}
	a.Name = p.Name
	a.Proper = true
	a.Title = lvTitleOf(p)
	age := p.lvAge(lvYear(g))
	a.Desc = fmt.Sprintf("Aged %d, %s of %s. %s Seems %s.", age, lvAn(a.Title), lvBurgName(g, p.Home), lvWants(p), lvMoodWord(p))
	if p.Species != "" && p.Species != "human" && g.W.specByID[p.Species] != nil {
		a.Desc = fmt.Sprintf("A %s. ", strings.ToLower(g.W.specByID[p.Species].Adj)) + a.Desc
	}
}

func lvAn(role string) string {
	if role == "" {
		return "a townsperson"
	}
	if strings.ContainsAny(role[:1], "aeiou") {
		return "an " + role
	}
	return "a " + role
}

func lvWants(p *lvPerson) string {
	if p.Goal == "" {
		return ""
	}
	return "Wants to " + p.Goal + "."
}

func lvActorMatches(a *Actor, p *lvPerson) bool {
	if a.Role == "elder" {
		return p.Role != "child"
	}
	if tr, ok := lvActorTrades[a.Role]; ok {
		for _, t := range tr {
			if p.Role == t {
				return true
			}
		}
		return false
	}
	if a.Role == "villager" || a.Role == "" {
		return lvCivilian[p.Role]
	}
	return false
}

func lvEnter(g *Game, zn *Zone) {
	if zn == nil || zn.Kind != "town" || zn.Burg <= 0 || g.W == nil {
		return
	}
	lvEnsure(g, zn.Burg)
	lvBindZone(g, zn)
}

func lvBindZone(g *Game, zn *Zone) {
	st := lvSt(g)
	b := lvb(g)
	burg := zn.Burg
	bd := g.W.burgByID[burg]
	if bd == nil {
		return
	}
	year := lvYear(g)
	taken := map[string]bool{}
	for _, k := range b.Bind {
		taken[k] = true
	}
	keep := zn.Actors[:0:0]
	var unbound []*Actor
	for _, a := range zn.Actors {
		if k, ok := b.Bind[a.ID]; ok {
			p := st.People[k]
			if p == nil || !p.Alive || p.Burg != burg {
				delete(b.Bind, a.ID)
				continue
			}
			lvApply(g, a, p, zn)
			keep = append(keep, a)
			continue
		}
		keep = append(keep, a)
		if lvEligible(a) {
			unbound = append(unbound, a)
		}
	}
	zn.Actors = keep
	present := lvPresent(g, burg)
	var remove map[*Actor]bool
	for _, a := range unbound {
		if a.SimPer != 0 {
			k := fmt.Sprintf("s%d", a.SimPer)
			if p := st.People[k]; p != nil {
				if p.Alive && p.Burg == burg && !taken[k] {
					taken[k] = true
					lvApply(g, a, p, zn)
				} else if !p.Alive {
					if remove == nil {
						remove = map[*Actor]bool{}
					}
					remove[a] = true
				}
				continue
			}
			lvAdopt(g, a, burg, zn)
			continue
		}
		rename := !(a.Talked || a.Quest != 0 || a.Teach != "" || a.Kinship)
		var pick *lvPerson
		if rename {
			for pass := 0; pass < 2 && pick == nil; pass++ {
				for _, p := range present {
					if taken[p.Key] || strings.HasPrefix(p.Key, "s") || p.lvAge(year) < 14 || !lvActorMatches(a, p) {
						continue
					}
					if pass == 0 && p.Zone != zn.Key || pass == 1 && p.Zone != "" {
						continue
					}
					pick = p
					break
				}
			}
		}
		if pick != nil {
			taken[pick.Key] = true
			lvApply(g, a, pick, zn)
			if a.Role == "villager" {
				a.Title = lvTitleOf(pick)
			}
		} else {
			lvAdopt(g, a, burg, zn)
		}
	}
	if remove != nil {
		keep = zn.Actors[:0:0]
		for _, a := range zn.Actors {
			if !remove[a] {
				keep = append(keep, a)
			}
		}
		zn.Actors = keep
	}
	// spawn people who have no actor yet
	spawned := 0
	cu0 := g.W.cultByID[bd.Culture]
	for _, p := range present {
		if spawned >= 4 || len(zn.Actors) > 28 {
			break
		}
		if taken[p.Key] || strings.HasPrefix(p.Key, "s") || p.lvAge(year) < 14 || p.Zone != "" && p.Zone != zn.Key {
			continue
		}
		role := lvTradeActorRole[p.Role]
		if role == "" {
			role = "villager"
		}
		cu := g.W.cultByID[p.Culture]
		if cu == nil {
			cu = cu0
		}
		r := lvRng(g, "spawn:"+p.Key)
		a := g.makeNPC(role, bd, cu, r)
		x, y, ok := zn.FreeSpot(r, nil)
		if !ok {
			break
		}
		a.X, a.Y, a.HomeX, a.HomeY = x, y, x, y
		a.Species = p.Species
		zn.Actors = append(zn.Actors, a)
		taken[p.Key] = true
		lvApply(g, a, p, zn)
		spawned++
	}
}

// lvAdopt makes a person out of an actor that has none (specials, extra shopkeepers, strangers).
func lvAdopt(g *Game, a *Actor, burg int, zn *Zone) *lvPerson {
	st := lvSt(g)
	var key string
	for {
		st.Next++
		key = fmt.Sprintf("g%d:a%d", burg, st.Next)
		if st.People[key] == nil {
			break
		}
	}
	bd := g.W.burgByID[burg]
	if bd == nil {
		return nil
	}
	r := lvRng(g, "adopt:"+key)
	cu := g.W.cultByID[bd.Culture]
	trade := strings.ToLower(a.Title)
	if _, ok := lvTradeWealth[trade]; !ok {
		switch a.Role {
		case "apothecary":
			trade = "healer"
		case "bookseller":
			trade = "scribe"
		case "villager", "":
			trade = Pick(r, []string{"farmer", "carter", "weaver", "potter", "fisher"})
		default:
			trade = nonEmpty(trade, a.Role)
		}
	}
	age := r.Int(18, 66)
	if a.Role == "elder" {
		age = r.Int(55, 78)
	}
	y0 := g.W.Year
	p := lvBase(g, bd, cu, key, a.Name, trade, y0-age, r.Chance(0.5), r)
	if a.Species != "" {
		p.Species = a.Species
	}
	p.Role = trade
	if a.Title != "" {
		p.Job = a.Title
	}
	if a.Role == "guard" {
		p.Role = "guard"
		p.Job = "guard"
	}
	p.Goal, p.GoalID = lvPickGoal(r, p, nil, nil, g)
	p.GoalPct = r.Int(0, 60)
	lvPickFear(r, p)
	if r.Chance(0.4) {
		p.Secret = lvPickSecret(g, r, p, nil)
	}
	lvLifeLines(g, p, bd, r, y0)
	st.People[key] = p
	lvApply(g, a, p, zn)
	a.Name = p.Name // unchanged, but keeps Desc consistent
	return p
}

// lvPersonFor is lvPersonOf: the person behind a talking actor.
func lvPersonFor(g *Game, a *Actor) *lvPerson {
	if a == nil || g.W == nil || g.Cur == nil || a.Tpl != "npc" || a.AI == "hostile" || a.AI == "follower" {
		return nil
	}
	b := lvb(g)
	st := lvSt(g)
	if k, ok := b.Bind[a.ID]; ok {
		if p := st.People[k]; p != nil {
			return p
		}
	}
	burg := g.Cur.Burg
	if burg <= 0 {
		return nil
	}
	lvEnsure(g, burg)
	if a.SimPer != 0 {
		if p := st.People[fmt.Sprintf("s%d", a.SimPer)]; p != nil {
			b.Bind[a.ID] = p.Key
			return p
		}
	}
	return lvAdopt(g, a, burg, g.Cur)
}

// ---------------------------------------------------------------- talk

func lvDepth(p *lvPerson) int {
	switch {
	case p.Opinion < -25:
		return 0
	case p.Opinion < 20:
		return 1
	case p.Opinion < 60:
		return 2
	}
	return 3
}

func lvTalk(g *Game, a *Actor) []Option {
	p := lvPersonFor(g, a)
	if p == nil || !p.Alive {
		return nil
	}
	lvTouch(g, p)
	opts := []Option{
		{"Ask about their life.", func() { g.talkMenu(lvLifeTalk(g, p)) }},
		{"Ask what they want.", func() { g.talkMenu(lvWantTalk(g, p)) }},
		{"Ask about their family.", func() { g.talkMenu(lvFamilyTalk(g, p)) }},
	}
	var who []string
	if p.Boss != "" {
		who = append(who, p.Boss)
	}
	for i, k := range p.Friends {
		if i < 2 {
			who = append(who, k)
		}
	}
	if len(p.Rivals) > 0 {
		who = append(who, p.Rivals[0])
	}
	for _, k := range p.Kin {
		if len(who) >= 5 {
			break
		}
		if k != p.Spouse {
			who = append(who, k)
			break
		}
	}
	seen := map[string]bool{}
	for _, k := range who {
		q := lvGet(g, k)
		if q == nil || seen[k] {
			continue
		}
		seen[k] = true
		opts = append(opts, Option{"Ask about " + q.Name + ".", func() { g.talkMenu(lvAboutTalk(g, p, q)) }})
	}
	opts = append(opts, Option{"{{K|Who do I know?}}", func() { g.lvOpenPeople(true) }})
	return opts
}

// lvTouch marks a person as met and records a first meeting and a talk a day.
func lvTouch(g *Game, p *lvPerson) {
	b := lvb(g)
	day := lvDay(g)
	if !p.Met {
		p.Met = true
		p.remember(day, "met", "You introduced yourself.", 2)
		if !b.Firsts["met"] {
			b.Firsts["met"] = true
			g.deed("life", fmt.Sprintf("The wanderer %s spoke with %s, %s of %s.", g.P.Name, p.Name, lvAn(p.Role), lvBurgName(g, p.Home)), 0, 0)
		}
		if strings.HasPrefix(p.Key, "s") && !b.Firsts["notable"] {
			b.Firsts["notable"] = true
			g.deed("life", fmt.Sprintf("The wanderer %s met %s, %s, in %s.", g.P.Name, p.Name, nonEmpty(p.Job, p.Role), lvBurgName(g, p.Home)), 0, 0)
		}
	}
	if b.Last[p.Key] != day+1 {
		b.Last[p.Key] = day + 1
		b.Talks[p.Key]++
		if b.Talks[p.Key] <= 15 && p.Opinion < 70 {
			found := false
			for i := range p.Mem {
				if p.Mem[i].Kind == "talk" {
					p.Mem[i].Day = day
					p.Mem[i].Text = fmt.Sprintf("You have talked with them on %d days.", b.Talks[p.Key])
					p.Opinion = clampi(p.Opinion+1, -100, 100)
					found = true
					break
				}
			}
			if !found {
				p.remember(day, "talk", "You have talked with them on 1 day.", 1)
			}
		}
	}
	lvCheckFriend(g, p)
}

func lvCheckFriend(g *Game, p *lvPerson) {
	b := lvb(g)
	if p.Opinion >= 60 && !b.Firsts["friend"] {
		b.Firsts["friend"] = true
		g.deed("life", fmt.Sprintf("The wanderer %s befriended %s of %s.", g.P.Name, p.Name, lvBurgName(g, p.Home)), 0, 0)
	}
}

func lvAgeWord(g *Game, p *lvPerson) int { return p.lvAge(lvYear(g)) }

func lvLifeTalk(g *Game, p *lvPerson) string {
	d := lvDepth(p)
	if d == 0 {
		return "\"I am " + p.Name + ". That is all you need to know.\"\n\n\"I do not tell my life to people like you.\""
	}
	var L []string
	L = append(L, fmt.Sprintf("\"I am %s, %s of %s. %d years behind me now.\"", p.Name, lvAn(lvTitleOf(p)), lvBurgName(g, p.Home), lvAgeWord(g, p)))
	for _, e := range p.Life {
		if e.Kind == "birth" && strings.HasPrefix(e.Text, "Born in ") {
			L = append(L, fmt.Sprintf("\"I was born in %s, in the year %d.\"", strings.TrimSuffix(strings.TrimPrefix(e.Text, "Born in "), "."), p.Born))
			break
		}
	}
	for _, e := range p.Life {
		if e.Kind == "job" {
			L = append(L, fmt.Sprintf("\"I took up the work in the year %d, and I have kept at it since.\"", e.Y))
			break
		}
	}
	if p.Spouse != "" {
		L = append(L, "\"I married "+lvName(g, p.Spouse)+".\"")
	}
	kids := 0
	for _, k := range p.Kin {
		if r := p.Rel[k]; r == "son" || r == "daughter" {
			kids++
		}
	}
	if kids > 0 {
		L = append(L, fmt.Sprintf("\"We have %d children.\"", kids))
	}
	for _, e := range p.Life {
		if e.Kind == "illness" {
			L = append(L, "\"Hard times: "+lvLower(e.Text)+"\"")
			break
		}
	}
	if ev := lvRealEvents(g, p); len(ev) > 0 {
		e := ev[int(HashStr(p.Key))%len(ev)]
		L = append(L, fmt.Sprintf("\"I remember the year %d. %s\"", e.Y, e.Text))
	}
	if d >= 2 && p.Goal != "" {
		L = append(L, "\"Mostly, I want to "+p.Goal+".\"")
	}
	if d >= 2 && p.Fear != "" {
		L = append(L, "\"What frightens me: "+p.Fear+".\"")
	}
	if d >= 3 && p.Secret != "" {
		L = append(L, "\"You have earned this, so I will say it. "+p.Secret+"\"")
		lvRevealSecret(g, p)
	}
	if len(L) > 8 {
		L = L[:8]
	}
	return strings.Join(L, "\n\n")
}

func lvWantTalk(g *Game, p *lvPerson) string {
	if p.Goal == "" {
		return "\"Want? A quiet day.\""
	}
	s := "\"I want to " + p.Goal + ".\""
	switch {
	case p.GoalPct >= 70:
		s += " They look almost there."
	case p.GoalPct >= 35:
		s += " They have made some headway."
	default:
		s += " They have barely begun."
	}
	if p.Opinion >= 20 && p.Fear != "" {
		s += "\n\n\"And what keeps me up at night is " + p.Fear + ".\""
	}
	return s
}

func lvFamilyTalk(g *Game, p *lvPerson) string {
	var L []string
	if p.Spouse != "" {
		if q := lvGet(g, p.Spouse); q != nil {
			L = append(L, fmt.Sprintf("\"My %s is %s, %s.\"%s", lvSexWord(q, "husband", "wife"), q.Name, lvAn(q.Role), lvDead(q)))
		}
	}
	for _, k := range p.Kin {
		if k == p.Spouse || len(L) >= 7 {
			continue
		}
		if q := lvGet(g, k); q != nil {
			L = append(L, fmt.Sprintf("\"My %s %s, %s.\"%s", lvKinLabel(p, q), q.Name, lvAn(q.Role), lvDead(q)))
		}
	}
	if len(L) == 0 {
		return "\"I have no family left in this town.\""
	}
	return strings.Join(L, "\n")
}

func lvDead(q *lvPerson) string {
	if q.Alive {
		return ""
	}
	return " {{K|(dead)}}"
}

func lvAboutTalk(g *Game, p, q *lvPerson) string {
	rel := "know"
	switch {
	case lvHas(p.Friends, q.Key):
		rel = "friend"
	case lvHas(p.Rivals, q.Key):
		rel = "rival"
	case lvHas(p.Kin, q.Key):
		rel = "kin"
	case p.Boss == q.Key:
		rel = "boss"
	}
	s := ""
	switch rel {
	case "friend":
		s = fmt.Sprintf("\"%s? A good friend. %s is %s.\"", q.Name, titleCase(lvSexWord(q, "he", "she")), lvAn(q.Role))
	case "rival":
		s = fmt.Sprintf("\"%s. Do not get me started. %s is %s, and we have never agreed.\"", q.Name, titleCase(lvSexWord(q, "he", "she")), lvAn(q.Role))
	case "kin":
		s = fmt.Sprintf("\"%s is my %s, %s.\"", q.Name, lvKinLabel(p, q), lvAn(q.Role))
	case "boss":
		s = fmt.Sprintf("\"%s? I answer to %s. %s is %s.\"", q.Name, lvSexWord(q, "him", "her"), titleCase(lvSexWord(q, "he", "she")), lvAn(q.Role))
	default:
		s = fmt.Sprintf("\"%s is %s here.\"", q.Name, lvAn(q.Role))
	}
	if !q.Alive {
		s += fmt.Sprintf(" They are dead now, since the year %d.", q.Died)
	} else {
		s += " " + titleCase(lvSexWord(q, "he", "she")) + " seems " + lvMoodWord(q) + " lately."
		if rel == "friend" || rel == "kin" {
			if q.Goal != "" && p.Opinion >= 0 {
				s += " I hear " + lvSexWord(q, "he", "she") + " wants to " + q.Goal + "."
			}
		}
	}
	return s
}

// ---------------------------------------------------------------- memory of you

func (g *Game) lvMemoryAdd(p *lvPerson, kind, text string, w int) {
	day := lvDay(g)
	if n := len(p.Mem); n > 0 && p.Mem[n-1].Kind == kind && p.Mem[n-1].Day == day && p.Mem[n-1].Text == text {
		return
	}
	p.remember(day, kind, text, w)
	p.Mood = clampi(p.Mood+w/2, -100, 100)
	lvCheckFriend(g, p)
}

// lvWitness: everyone bound to an actor in this zone who can see you remembers what you did.
func (g *Game) lvWitness(kind, text string, weight int) {
	if g.Cur == nil || g.P == nil || g.W == nil {
		return
	}
	b := lvb(g)
	st := lvSt(g)
	for _, a := range g.Cur.Actors {
		k, ok := b.Bind[a.ID]
		if !ok || a.HP <= 0 {
			continue
		}
		p := st.People[k]
		if p == nil || !p.Alive {
			continue
		}
		if maxi(abs(a.X-g.P.X), abs(a.Y-g.P.Y)) > 25 || !g.losClear(a.X, a.Y, g.P.X, g.P.Y) {
			continue
		}
		g.lvMemoryAdd(p, kind, text, weight)
	}
	if kind == "theft" || kind == "attack" {
		if !b.Firsts[kind] {
			b.Firsts[kind] = true
			g.deed("life", fmt.Sprintf("The wanderer %s was seen: %s", g.P.Name, text), 0, 0)
		}
	}
}

// lvTrade: call after a completed trade with a bound person (value in coin).
func (g *Game) lvTrade(a *Actor, value int) {
	p := lvPersonFor(g, a)
	if p == nil {
		return
	}
	day := lvDay(g)
	for _, m := range p.Mem {
		if m.Kind == "trade" && m.Day == day {
			return
		}
	}
	g.lvMemoryAdd(p, "trade", "You dealt fairly with them.", 1)
}

// lvGift: call when the player gives an item worth `value` to a person.
func (g *Game) lvGift(a *Actor, value int) {
	p := lvPersonFor(g, a)
	if p == nil {
		return
	}
	w := clampi(2+value/20, 2, 15)
	g.lvMemoryAdd(p, "gift", "You gave them a gift.", w)
	b := lvb(g)
	if !b.Firsts["gift"] {
		b.Firsts["gift"] = true
		g.deed("life", fmt.Sprintf("The wanderer %s gave a gift to %s of %s.", g.P.Name, p.Name, lvBurgName(g, p.Home)), 0, 0)
	}
}

func lvKill(g *Game, d, src *Actor) {
	if d == nil || g.W == nil {
		return
	}
	b := lvb(g)
	k, ok := b.Bind[d.ID]
	if !ok {
		return
	}
	delete(b.Bind, d.ID)
	st := lvSt(g)
	p := st.People[k]
	if p == nil || !p.Alive {
		return
	}
	year := lvYear(g)
	day := lvDay(g)
	byYou := src != nil && g.P != nil && src == g.P
	p.Alive = false
	p.Died = year
	switch {
	case byYou:
		p.Cause = "killed by " + g.P.Name
	case src != nil:
		p.Cause = "killed by " + src.Name
	default:
		p.Cause = "killed"
	}
	p.addLife(year, "death", "Died: "+p.Cause+".")
	if byYou {
		st.addNews(day, p.Home, "%s, %s of %s, was found dead; folk whisper of a stranger.", p.Name, p.Role, lvBurgName(g, p.Home))
	} else {
		st.addNews(day, p.Home, "%s, %s of %s, is dead.", p.Name, p.Role, lvBurgName(g, p.Home))
	}
	react := func(key string, w int, kind, text string) {
		q := st.People[key]
		if q == nil || !q.Alive {
			return
		}
		q.Mood = clampi(q.Mood-abs(w), -100, 100)
		if byYou {
			q.remember(day, kind, text, w)
		}
	}
	if p.Spouse != "" {
		if q := st.People[p.Spouse]; q != nil && q.Alive {
			q.Spouse = ""
			q.addLife(year, "death", "Widowed: "+p.Name+" died.")
			react(p.Spouse, -20, "kill", "You killed their "+lvSexWord(p, "husband", "wife")+", "+p.Name+".")
		}
	}
	for _, key := range p.Kin {
		if key != p.Spouse {
			if q := st.People[key]; q != nil && q.Alive {
				q.addLife(year, "death", "Lost "+lvName(g, p.Key)+", their "+lvKinLabel(q, p)+".")
			}
			react(key, -18, "kill", "You killed their "+lvKinLabel(st.People[key], p)+", "+p.Name+".")
		}
	}
	for _, key := range p.Friends {
		react(key, -12, "kill", "You killed their friend "+p.Name+".")
	}
	if p.Boss != "" {
		react(p.Boss, -5, "kill", "You killed "+p.Name+", who answered to them.")
	}
	for _, key := range p.Rivals {
		react(key, 4, "help", "You rid them of "+p.Name+".")
	}
	if byYou {
		g.lvWitness("kill", "You killed "+p.Name+" before their eyes.", -10)
		if !b.Firsts["kill"] {
			b.Firsts["kill"] = true
			g.deed("life", fmt.Sprintf("The wanderer %s killed %s, %s of %s.", g.P.Name, p.Name, lvAn(p.Role), lvBurgName(g, p.Home)), 0, 0)
		}
	}
}

// lvGreeting: a greeting that reflects what they think of you; "" for the default one.
func (g *Game) lvGreeting(a *Actor) string {
	p := lvPersonFor(g, a)
	if p == nil || !p.Met || !p.Alive {
		return ""
	}
	h := int(HashStr(p.Key+fmt.Sprint(lvDay(g))) % 1000)
	pick := func(l []string) string { return l[h%len(l)] }
	var worst *lvMemory
	for i := range p.Mem {
		if p.Mem[i].Weight <= -15 && (worst == nil || p.Mem[i].Weight < worst.Weight) {
			worst = &p.Mem[i]
		}
	}
	switch {
	case p.Opinion <= -25 && worst != nil:
		return "\"I remember what you did. " + worst.Text + "\" Their jaw tightens."
	case p.Opinion <= -60:
		return pick(lvGreetHostile)
	case p.Opinion <= -25:
		return pick(lvGreetCold)
	case p.Opinion >= 55:
		s := pick(lvGreetWarm)
		if strings.Contains(s, "%s") {
			s = fmt.Sprintf(s, g.P.Name)
		}
		return s
	case p.Opinion >= 25:
		return pick(lvGreetFriendly)
	}
	return ""
}

// lvPriceMod: 0.9 (adore you) .. 1.1 (hate you); 1 for strangers. Callers multiply what you pay; invert for what you get.
func (g *Game) lvPriceMod(a *Actor) float64 {
	p := lvPersonFor(g, a)
	if p == nil || !p.Met {
		return 1
	}
	return 1 - 0.1*float64(clampi(p.Opinion, -100, 100))/100
}

// ---------------------------------------------------------------- People screen

type lvRow struct {
	hdr string
	key string
}

func lvRows(g *Game) []lvRow {
	st := lvSt(g)
	var met []*lvPerson
	for _, p := range st.People {
		if p.Met {
			met = append(met, p)
		}
	}
	sort.Slice(met, func(i, j int) bool {
		a, b := lvBurgName(g, met[i].Home), lvBurgName(g, met[j].Home)
		if a != b {
			return a < b
		}
		if met[i].Name != met[j].Name {
			return met[i].Name < met[j].Name
		}
		return met[i].Key < met[j].Key
	})
	var rows []lvRow
	last := -1
	for _, p := range met {
		if p.Home != last {
			last = p.Home
			rows = append(rows, lvRow{hdr: lvBurgName(g, p.Home)})
		}
		rows = append(rows, lvRow{key: p.Key})
	}
	return rows
}

func (g *Game) lvOpenPeople(fromTalk bool) {
	b := lvb(g)
	b.fromTalk = fromTalk && g.Talk != nil
	b.bio, b.scroll, b.sel = "", 0, 0
	rows := lvRows(g)
	for i, r := range rows {
		if r.key != "" {
			b.sel = i
			break
		}
	}
	g.Mode = "lv"
}

func (g *Game) lvAbilities() []Ability {
	if g.Cur == nil || g.P == nil {
		return nil
	}
	return []Ability{{ID: "lv_people", Name: "People you know", Target: "self"}}
}

func (g *Game) lvAbUse(id string) {
	if id == "lv_people" {
		g.lvOpenPeople(false)
	}
}

func lvClose(g *Game) {
	b := lvb(g)
	if b.fromTalk && g.Talk != nil {
		b.fromTalk = false
		g.Mode = "talk"
		g.talkMenu(g.greeting(g.Talk))
		return
	}
	g.Mode = "main"
}

func lvKeys(g *Game, k string) {
	b := lvb(g)
	if b.bio != "" {
		switch k {
		case "Escape", "q", "Backspace":
			b.bio = ""
		case "ArrowDown", "N2", "j":
			b.scroll++
		case "ArrowUp", "N8", "k":
			b.scroll = maxi(0, b.scroll-1)
		case "PageDown":
			b.scroll += 15
		case "PageUp":
			b.scroll = maxi(0, b.scroll-15)
		case "Home":
			b.scroll = 0
		}
		return
	}
	rows := lvRows(g)
	move := func(d int) {
		i := b.sel
		for n := 0; n < len(rows); n++ {
			i += d
			if i < 0 || i >= len(rows) {
				return
			}
			if rows[i].key != "" {
				b.sel = i
				return
			}
		}
	}
	switch k {
	case "Escape", "q":
		lvClose(g)
	case "ArrowDown", "N2", "j":
		move(1)
	case "ArrowUp", "N8", "k":
		move(-1)
	case "PageDown":
		for i := 0; i < 10; i++ {
			move(1)
		}
	case "PageUp":
		for i := 0; i < 10; i++ {
			move(-1)
		}
	case "Enter", " ":
		if b.sel >= 0 && b.sel < len(rows) && rows[b.sel].key != "" {
			b.bio, b.scroll = rows[b.sel].key, 0
		}
	}
}

func (g *Game) lvDraw() {
	b := lvb(g)
	s := g.sc
	x, y, w, h := g.panel("People")
	if b.bio != "" {
		p := lvGet(g, b.bio)
		if p == nil {
			b.bio = ""
			return
		}
		var lines []string
		for _, l := range lvBio(g, p) {
			if l == "" {
				lines = append(lines, "")
				continue
			}
			lines = append(lines, Wrap(l, w)...)
		}
		view := h - 2
		b.scroll = clampi(b.scroll, 0, maxi(0, len(lines)-view))
		for i := 0; i < view && b.scroll+i < len(lines); i++ {
			s.TextB(x, y+i, lines[b.scroll+i], CGray, CBgDark, x+w)
		}
		s.Text(x, y+h-1, "{{K|Up/Down/PageUp/PageDown: scroll   Esc: back}}", CGray)
		return
	}
	rows := lvRows(g)
	if len(rows) == 0 {
		s.Text(x, y, "{{K|You have not talked with anyone yet. Speak with people to meet them.}}", CGray)
		s.Text(x, y+h-1, "{{K}}Esc: close", CGray)
		return
	}
	b.sel = clampi(b.sel, 0, len(rows)-1)
	view := h - 3
	if b.scroll > b.sel {
		b.scroll = b.sel
	}
	if b.sel >= b.scroll+view {
		b.scroll = b.sel - view + 1
	}
	b.scroll = maxi(0, b.scroll)
	s.Text(x, y, "{{K|name                role       mood      toward you          state}}", CGray)
	for i := 0; i < view; i++ {
		idx := b.scroll + i
		if idx >= len(rows) {
			break
		}
		r := rows[idx]
		yy := y + 1 + i
		if r.hdr != "" {
			s.TextB(x, yy, "{{Y|"+r.hdr+"}}", CGray, CBgDark, x+w)
			continue
		}
		p := lvGet(g, r.key)
		if p == nil {
			continue
		}
		bg := CBgDark
		if idx == b.sel {
			bg = CBgSel
		}
		s.Fill(x, yy, w, 1, bg)
		state := "alive"
		if !p.Alive {
			state = "{{R|dead}}"
		} else if p.Burg != p.Home {
			state = "{{K|away}}"
		}
		s.TextB(x+1, yy, lvTrunc(p.Name, 19), CWhite, bg, x+w)
		s.TextB(x+21, yy, lvTrunc(p.Role, 10), CGray, bg, x+w)
		s.TextB(x+32, yy, lvTrunc(lvMoodWord(p), 9), CGray, bg, x+w)
		s.TextB(x+42, yy, fmt.Sprintf("%s %+d", lvTrunc(lvOpinionWord(p), 13), p.Opinion), CCyan, bg, x+w)
		s.TextB(x+62, yy, state, CGray, bg, x+w)
	}
	s.Text(x, y+h-1, "{{K|Up/Down/PageUp/PageDown: select   Enter: their story   Esc: close}}", CGray)
}
