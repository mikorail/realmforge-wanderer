package engine

// What people know (prefix kn): facts, rumours that distort as they spread, questioning with trust and
// checks, cross-checking conflicting accounts, case notes and accusations. Classic simulation, no AI calls.
// State g.ext("kn"), planet-local. Facts live here; people (lvPerson.Knows) hold fact ids.
// Never draws g.R for generation: side rngs seeded from g.Seed + ":kn:" + key. Only checks (play) use g.R.

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

type knFact struct {
	ID      int    `json:"i"`
	Kind    string `json:"k"`           // sighting event rumour secret location crime
	Subject string `json:"s,omitempty"` // person key, "burg:N", "player" or item text
	Text    string `json:"t"`
	True    bool   `json:"tr,omitempty"` // the text is actually true
	Source  string `json:"so,omitempty"` // person key who originated this version
	Day     int    `json:"d"`
	Burg    int    `json:"b,omitempty"`
	Spread  int    `json:"sp,omitempty"` // times it was passed on
	Orig    int    `json:"o,omitempty"`  // fact this rumour distorts
	Case    int    `json:"c,omitempty"`
	Culprit string `json:"cu,omitempty"` // person key the fact points at (case clues)
	Lvl     int    `json:"lv,omitempty"` // trust needed to hear it: 0 hint, 1 detail, 2 name or secret
	Ver     int    `json:"v,omitempty"`  // 1 verified true, -1 verified false
	Dist    int    `json:"di,omitempty"`
}

type knLearn struct {
	Fact  int    `json:"f"`
	From  string `json:"w"`
	Name  string `json:"n"`
	Day   int    `json:"d"`
	Trust int    `json:"t"`
	Via   string `json:"v,omitempty"` // ask paid pressed gossip
}

type knRel struct {
	Hit  int `json:"h,omitempty"`
	Miss int `json:"m,omitempty"`
}

type knCase struct {
	ID      int    `json:"i"`
	Title   string `json:"t"`
	Subject string `json:"s,omitempty"`
	Culprit string `json:"c"`
	FactIDs []int  `json:"f,omitempty"`
	Status  string `json:"st"` // open solved failed
	Burg    int    `json:"b"`
	Day     int    `json:"d"`
	Kind    string `json:"k,omitempty"`
	Outcome string `json:"o,omitempty"`
	Wrong   int    `json:"w,omitempty"`
}

type knState struct {
	Facts   map[int]*knFact   `json:"f,omitempty"`
	Next    int               `json:"x,omitempty"`
	Learned []knLearn         `json:"l,omitempty"`
	Rel     map[string]*knRel `json:"r,omitempty"`
	Cases   []*knCase         `json:"c,omitempty"`
	Seen    map[string]int    `json:"s,omitempty"`
	Last    map[string]int    `json:"la,omitempty"`
	Memo    map[string]int    `json:"m,omitempty"`
	Asked   map[string]int    `json:"a,omitempty"`
	Firsts  map[string]bool   `json:"fi,omitempty"`
	Day     int               `json:"d,omitempty"`
	Solved  int               `json:"so,omitempty"`

	tab, sel int
	fromTalk bool
}

func init() {
	unLocalKeys = append(unLocalKeys, "kn")
	modeKeys["kn"] = knKeys
	modeDraws["kn"] = knDraw
	talkHooks = append(talkHooks, knTalk)
	turnHooks = append(turnHooks, knTurn)
	enterHooks = append(enterHooks, knEnter)
	lvKnowledgeOf = knKnowledgeOf
}

func knSt(g *Game) *knState {
	s := g.ext("kn", func() any { return &knState{} }).(*knState)
	if s.Facts == nil {
		s.Facts = map[int]*knFact{}
	}
	if s.Rel == nil {
		s.Rel = map[string]*knRel{}
	}
	if s.Seen == nil {
		s.Seen = map[string]int{}
	}
	if s.Last == nil {
		s.Last = map[string]int{}
	}
	if s.Memo == nil {
		s.Memo = map[string]int{}
	}
	if s.Asked == nil {
		s.Asked = map[string]int{}
	}
	if s.Firsts == nil {
		s.Firsts = map[string]bool{}
	}
	return s
}

func (st *knState) add(f knFact) *knFact {
	st.Next++
	f.ID = st.Next
	p := &f
	st.Facts[f.ID] = p
	return p
}

func (st *knState) fact(id int) *knFact { return st.Facts[id] }

func knGive(p *lvPerson, id int) bool {
	for _, k := range p.Knows {
		if k == id {
			return false
		}
	}
	p.Knows = append(p.Knows, id)
	if len(p.Knows) > 24 {
		p.Knows = p.Knows[len(p.Knows)-24:]
	}
	return true
}

func knKnown(p *lvPerson, id int) bool {
	for _, k := range p.Knows {
		if k == id {
			return true
		}
	}
	return false
}

func (g *Game) knBurgName(b int) string {
	if bd := g.W.burgByID[b]; bd != nil {
		return bd.Name
	}
	return "the road"
}

func (g *Game) knRng(parts ...any) *Rng {
	return NewRng(g.Seed + ":kn:" + fmt.Sprint(parts...))
}

// knPeople picks up to n living people present in a burg, shuffled by r, skipping the given key.
func knPeople(g *Game, burg, n int, r *Rng, skip string) []*lvPerson {
	all := lvPresent(g, burg)
	var pool []*lvPerson
	for _, p := range all {
		if p.Key != skip {
			pool = append(pool, p)
		}
	}
	for i := len(pool) - 1; i > 0; i-- {
		j := r.Int(0, i)
		pool[i], pool[j] = pool[j], pool[i]
	}
	if len(pool) > n {
		pool = pool[:n]
	}
	return pool
}

func knShort(s string, n int) string {
	s = Plain(s)
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n-1]) + "…"
}

// ---- looks ----

func knLook(p *lvPerson) string {
	h := HashStr("look:" + p.Key)
	sp := p.Species
	if sp == "" {
		sp = "person"
	}
	return fmt.Sprintf("a %s %s %s with %s", knHeights[h%uint64(len(knHeights))], sp, p.Role, knMarks[(h>>8)%uint64(len(knMarks))])
}

// ---- sources ----

var knDigits = regexp.MustCompile(`\d+`)

// knDistort makes a rumour: a copy of f with a name swapped, a number changed, a direction flipped or a
// place replaced. Nil when nothing in the text can change.
func knDistort(g *Game, f *knFact, by *lvPerson, r *Rng) *knFact {
	st := knSt(g)
	text, subj, cul := f.Text, f.Subject, f.Culprit
	burg := f.Burg
	if burg == 0 && by != nil {
		burg = by.Burg
	}
	changed := false
	if burg > 0 {
		ppl := lvPresent(g, burg)
		var hit *lvPerson
		for _, p := range ppl {
			if p.Name != "" && strings.Contains(text, p.Name) && (hit == nil || len(p.Name) > len(hit.Name)) {
				hit = p
			}
		}
		if hit != nil && len(ppl) > 1 {
			for tries := 0; tries < 6; tries++ {
				o := ppl[r.Int(0, len(ppl)-1)]
				if o.Key != hit.Key && o.Name != hit.Name {
					text = strings.ReplaceAll(text, hit.Name, o.Name)
					if subj == hit.Key {
						subj = o.Key
					}
					if cul == hit.Key {
						cul = o.Key
					}
					changed = true
					break
				}
			}
		}
	}
	if !changed {
		if loc := knDigits.FindStringIndex(text); loc != nil {
			n := 0
			fmt.Sscan(text[loc[0]:loc[1]], &n)
			d := r.Int(1, maxi(2, n/3))
			if r.Chance(0.5) && n-d > 0 {
				d = -d
			}
			if n+d != n {
				text = text[:loc[0]] + fmt.Sprint(n+d) + text[loc[1]:]
				changed = true
			}
		}
	}
	if !changed {
		for _, w := range []string{"north", "south", "east", "west", "left", "right", "upstream", "downstream"} {
			if strings.Contains(text, w) {
				text = strings.Replace(text, w, knDirs[w], 1)
				changed = true
				break
			}
		}
	}
	if !changed && burg > 0 {
		if bd := g.W.burgByID[burg]; bd != nil && strings.Contains(text, bd.Name) {
			for _, o := range g.W.Burgs {
				if o.ID != burg && o.Alive && maxi(abs(o.P%g.W.W-bd.P%g.W.W), abs(o.P/g.W.W-bd.P/g.W.W)) <= 10 {
					text = strings.ReplaceAll(text, bd.Name, o.Name)
					burg = o.ID
					changed = true
					break
				}
			}
		}
	}
	if !changed {
		return nil
	}
	nf := *f
	nf.Text, nf.Subject, nf.Culprit, nf.Burg = text, subj, cul, burg
	nf.True, nf.Ver, nf.Spread = false, 0, 0
	nf.Dist = f.Dist + 1
	if f.Orig != 0 {
		nf.Orig = f.Orig
	} else {
		nf.Orig = f.ID
	}
	if by != nil {
		nf.Source = by.Key
	}
	return st.add(nf)
}

// knIngest turns what the simulation produced near a burg into facts, known by witnesses.
func knIngest(g *Game, b, d int) {
	if b <= 0 || g.W.burgByID[b] == nil {
		return
	}
	st := knSt(g)
	lv := lvSt(g)
	made := 0
	// news
	for _, n := range lv.News {
		if n.Burg != b || made >= 6 {
			continue
		}
		key := fmt.Sprintf("n:%d:%d:%x", n.Day, n.Burg, HashStr(n.Text)&0xffff)
		if _, ok := st.Seen[key]; ok {
			continue
		}
		st.Seen[key] = d
		r := g.knRng("n:", key)
		subj := fmt.Sprintf("burg:%d", b)
		for _, p := range lvPresent(g, b) {
			if p.Name != "" && strings.Contains(n.Text, p.Name) {
				subj = p.Key
				break
			}
		}
		f := st.add(knFact{Kind: "event", Subject: subj, Text: n.Text, True: true, Day: n.Day, Burg: b})
		for _, w := range knPeople(g, b, r.Int(2, 3), r, "") {
			knGive(w, f.ID)
			if f.Source == "" {
				f.Source = w.Key
			}
		}
		made++
	}
	// life events
	yr := lvYear(g)
	for _, p := range lvPresent(g, b) {
		if made >= 8 {
			break
		}
		for _, e := range p.Life {
			if e.Y < yr-1 {
				continue
			}
			switch e.Kind {
			case "crime", "feud", "marriage", "child", "illness", "job":
			default:
				continue
			}
			key := fmt.Sprintf("l:%s:%d:%x", p.Key, e.Y, HashStr(e.Text)&0xffff)
			if _, ok := st.Seen[key]; ok {
				continue
			}
			st.Seen[key] = d
			r := g.knRng("l:", key)
			if e.Kind == "crime" {
				g.knNPCCrime(b, p, e.Text, d, r)
				made++
				continue
			}
			txt := e.Text
			if !strings.Contains(txt, p.Name) {
				txt = p.Name + ": " + txt
			}
			f := st.add(knFact{Kind: "event", Subject: p.Key, Text: txt, True: true, Day: d, Burg: b, Source: p.Key})
			for _, w := range knPeople(g, b, r.Int(1, 2), r, p.Key) {
				knGive(w, f.ID)
			}
			made++
		}
	}
	// Realmforge events near the burg
	bd := g.W.burgByID[b]
	bx, by := bd.P%g.W.W, bd.P/g.W.W
	got := 0
	for dy := -6; dy <= 6 && got < 3; dy++ {
		for dx := -6; dx <= 6 && got < 3; dx++ {
			if !g.W.In(bx+dx, by+dy) {
				continue
			}
			for _, ei := range g.W.eventsAt[g.W.Idx(bx+dx, by+dy)] {
				if ei < 0 || ei >= len(g.W.Events) || got >= 3 {
					continue
				}
				e := g.W.Events[ei]
				if e.Y < yr-8 {
					continue
				}
				key := fmt.Sprintf("e:%d:%d", b, ei)
				if _, ok := st.Seen[key]; ok {
					continue
				}
				st.Seen[key] = d
				got++
				r := g.knRng("e:", key)
				day := d - (yr-e.Y)*112
				f := st.add(knFact{Kind: "event", Subject: fmt.Sprintf("burg:%d", b), Text: e.Text, True: true, Day: day, Burg: b})
				for _, w := range knPeople(g, b, r.Int(1, 2), r, "") {
					knGive(w, f.ID)
					if f.Source == "" {
						f.Source = w.Key
					}
				}
				if title, ok := knCaseKinds[e.Kind]; ok && r.Chance(0.4) {
					g.knEventCase(b, title, e, f, d, r)
				}
			}
		}
	}
	// secrets
	nsec := 0
	for _, p := range lvPresent(g, b) {
		if p.Secret == "" || nsec >= 3 {
			continue
		}
		key := "s:" + p.Key
		if _, ok := st.Seen[key]; ok {
			continue
		}
		st.Seen[key] = d
		r := g.knRng("s:", p.Key)
		n := 0
		switch x := r.Float(); {
		case x < 0.4:
			n = 0
		case x < 0.85:
			n = 1
		default:
			n = 2
		}
		f := st.add(knFact{Kind: "secret", Subject: p.Key, Text: p.Name + " hides this: " + strings.TrimSuffix(p.Secret, "."), True: true, Day: d, Burg: b, Lvl: 2, Source: p.Key})
		for _, w := range knPeople(g, b, n, r, p.Key) {
			knGive(w, f.ID)
		}
		nsec++
	}
	// two people seen about town
	r := g.knRng("ps:", b, ":", d)
	ppl := lvPresent(g, b)
	for i := 0; i < 2 && len(ppl) > 2; i++ {
		x := ppl[r.Int(0, len(ppl)-1)]
		key := fmt.Sprintf("ps:%s:%d", x.Key, d)
		if _, ok := st.Seen[key]; ok {
			continue
		}
		st.Seen[key] = d
		job := x.Job
		if job == "" {
			job = x.Role
		}
		f := st.add(knFact{Kind: "sighting", Subject: x.Key, Text: fmt.Sprintf("%s (%s) was seen in %s", x.Name, job, bd.Name), True: true, Day: d, Burg: b})
		for _, w := range knPeople(g, b, r.Int(1, 2), r, x.Key) {
			knGive(w, f.ID)
			if f.Source == "" {
				f.Source = w.Key
			}
		}
	}
	if g.Cur != nil && g.Cur.Burg == b && g.Cur.Kind == "town" {
		// people who saw the wanderer
		key := fmt.Sprintf("pl:%d:%d", d, b)
		if _, ok := st.Seen[key]; !ok {
			st.Seen[key] = d
			f := st.add(knFact{Kind: "sighting", Subject: "player", Text: fmt.Sprintf("A wanderer called %s was seen in %s", g.P.Name, bd.Name), True: true, Day: d, Burg: b})
			for _, w := range knPeople(g, b, r.Int(1, 3), r, "") {
				knGive(w, f.ID)
				if f.Source == "" {
					f.Source = w.Key
				}
			}
		}
		// hints of places, from what the wanderer already learned
		for i, s := range g.Secrets {
			if s.Kind != "dungeon" && s.Kind != "relic" && s.Kind != "hoard" {
				continue
			}
			key := fmt.Sprintf("loc:%d", i)
			if _, ok := st.Seen[key]; ok {
				continue
			}
			st.Seen[key] = d
			f := st.add(knFact{Kind: "location", Subject: knShort(s.Text, 40), Text: s.Text, True: true, Day: d, Burg: b})
			for _, w := range knPeople(g, b, 1, g.knRng("loc:", i), "") {
				knGive(w, f.ID)
				f.Source = w.Key
			}
			break
		}
	}
}

// ---- crimes and cases ----

// knCrime records a crime the wanderer committed in a burg ("player") or one a person committed.
// Witnesses present in the burg know it; call it from the law code (uwAddBounty) or from the simulation.
func (g *Game) knCrime(burg int, culprit string, text string) {
	if g == nil || g.W == nil || burg <= 0 || g.P == nil {
		return
	}
	st := knSt(g)
	d := lvDay(g)
	r := g.knRng("crime:", burg, ":", culprit, ":", d, ":", HashStr(text)&0xffff)
	if culprit != "" && culprit != "player" {
		if p := lvGet(g, culprit); p != nil {
			key := fmt.Sprintf("c:%s:%d:%x", p.Key, d, HashStr(text)&0xffff)
			if _, ok := st.Seen[key]; ok {
				return
			}
			st.Seen[key] = d
			g.knNPCCrime(burg, p, text, d, r)
		}
		return
	}
	key := fmt.Sprintf("cp:%d:%d", burg, d)
	if st.Seen[key] > 0 && st.Seen[key] == d+1 {
		return
	}
	st.Seen[key] = d + 1
	where := g.knBurgName(burg)
	txt := strings.TrimSpace(Plain(text))
	if txt == "" {
		txt = "a crime"
	}
	desc := st.add(knFact{Kind: "crime", Subject: "player", Text: fmt.Sprintf("In %s a stranger, a %s, was seen committing %s", where, strings.ToLower(g.Sk.Calling(g.Calling)), strings.TrimPrefix(txt, "!")), True: true, Day: d, Burg: burg, Culprit: "player", Lvl: 0})
	name := st.add(knFact{Kind: "crime", Subject: "player", Text: fmt.Sprintf("It was the wanderer %s who did it, in %s", g.P.Name, where), True: true, Day: d, Burg: burg, Culprit: "player", Lvl: 2})
	for _, w := range knPeople(g, burg, 3, r, "") {
		knGive(w, desc.ID)
		desc.Source = w.Key
		if r.Chance(0.6) {
			knGive(w, name.ID)
			name.Source = w.Key
		}
		w.remember(d, "witnessed", "saw you commit a crime", -8)
	}
}

func (g *Game) knNPCCrime(b int, culprit *lvPerson, text string, d int, r *Rng) {
	st := knSt(g)
	where := g.knBurgName(b)
	t := strings.TrimSuffix(strings.ReplaceAll(Plain(text), culprit.Name, "someone"), ".")
	f := st.add(knFact{Kind: "crime", Subject: fmt.Sprintf("burg:%d", b), Text: fmt.Sprintf("A crime in %s: %s", where, t), True: true, Day: d, Burg: b, Lvl: 0})
	wit := knPeople(g, b, 3, r, culprit.Key)
	for _, w := range wit {
		knGive(w, f.ID)
		f.Source = w.Key
	}
	g.knMakeCase(b, "A crime in "+where, culprit, f, d, "crime", wit, r)
}

func (g *Game) knEventCase(b int, title string, e EventData, src *knFact, d int, r *Rng) {
	ppl := lvPresent(g, b)
	var cand []*lvPerson
	for _, p := range ppl {
		if p.lvAge(lvYear(g)) >= 18 && p.Role != "ruler" {
			cand = append(cand, p)
		}
	}
	if len(cand) < 3 {
		return
	}
	c := cand[r.Int(0, len(cand)-1)]
	g.knMakeCase(b, title+" in "+g.knBurgName(b), c, src, d, e.Kind, nil, r)
}

// knMakeCase builds a case with clues of rising detail spread over different people, and one false lead.
func (g *Game) knMakeCase(b int, title string, culprit *lvPerson, src *knFact, d int, kind string, witnesses []*lvPerson, r *Rng) {
	st := knSt(g)
	open := 0
	for _, c := range st.Cases {
		if c.Status == "open" {
			open++
		}
		if c.Culprit == culprit.Key && c.Status == "open" {
			return
		}
	}
	if open >= 4 {
		return
	}
	cs := &knCase{ID: len(st.Cases) + 1, Title: title, Subject: fmt.Sprintf("burg:%d", b), Culprit: culprit.Key, Status: "open", Burg: b, Day: d, Kind: kind}
	cs.FactIDs = append(cs.FactIDs, src.ID)
	src.Case = cs.ID
	pool := knPeople(g, b, 8, r, culprit.Key)
	pick := func(i int) *lvPerson {
		if len(pool) == 0 {
			return nil
		}
		return pool[i%len(pool)]
	}
	mk := func(lvl int, text string, who ...*lvPerson) *knFact {
		f := st.add(knFact{Kind: "crime", Subject: culprit.Key, Text: text, True: true, Day: d, Burg: b, Case: cs.ID, Culprit: culprit.Key, Lvl: lvl})
		for _, w := range who {
			if w != nil {
				knGive(w, f.ID)
				f.Source = w.Key
			}
		}
		cs.FactIDs = append(cs.FactIDs, f.ID)
		return f
	}
	sp := culprit.Species
	if sp == "" {
		sp = "person"
	}
	mk(0, fmt.Sprintf("The one responsible was %s", knLook(culprit)), pick(0), pick(1))
	detail := fmt.Sprintf("The culprit works as a %s", culprit.Role)
	if culprit.Job != "" {
		detail = fmt.Sprintf("The culprit works as a %s", culprit.Job)
	}
	switch {
	case culprit.Spouse != "" && lvGet(g, culprit.Spouse) != nil && r.Chance(0.4):
		detail = fmt.Sprintf("The culprit is wed to %s", lvGet(g, culprit.Spouse).Name)
	case len(culprit.Rivals) > 0 && lvGet(g, culprit.Rivals[0]) != nil && r.Chance(0.5):
		detail = fmt.Sprintf("The culprit %s, and is no friend of %s", knMotives[r.Int(0, len(knMotives)-1)], lvGet(g, culprit.Rivals[0]).Name)
	}
	mk(1, detail, pick(2))
	mk(1, fmt.Sprintf("A %s %s was seen near the place that night, going toward the %s", sp, culprit.Role, []string{"market", "river", "gate", "temple"}[r.Int(0, 3)]), pick(3))
	mk(2, fmt.Sprintf("It was %s who did it", culprit.Name), pick(4))
	// false lead
	if len(pool) > 5 {
		inn := pool[5]
		df := st.add(knFact{Kind: "crime", Subject: inn.Key, Text: fmt.Sprintf("Folk say %s was seen near the scene", inn.Name), Day: d, Burg: b, Case: cs.ID, Culprit: inn.Key, Lvl: 1})
		for _, w := range pool[6:] {
			knGive(w, df.ID)
			df.Source = w.Key
			break
		}
		if len(pool) > 7 {
			knGive(pool[7], df.ID)
		}
		cs.FactIDs = append(cs.FactIDs, df.ID)
	}
	st.Cases = append(st.Cases, cs)
}

// ---- the daily turn ----

func knTurn(g *Game) {
	if g.P == nil || g.W == nil || g.Cur == nil {
		return
	}
	st := knSt(g)
	day := lvDay(g)
	if st.Day == 0 && len(st.Facts) == 0 {
		st.Day = day
		return
	}
	if day <= st.Day {
		return
	}
	from := st.Day
	if day-from > 3 {
		from = day - 3
		knDrift(g, day, day-st.Day)
	}
	st.Day = day
	active := knActive(g)
	for d := from + 1; d <= day; d++ {
		for _, b := range active {
			knIngest(g, b, d)
			knSpread(g, b, d)
			knTravel(g, b, d)
			knForget(g, b, d)
		}
		if d%7 == 0 {
			knDrift(g, d, 1)
		}
	}
	knVerifyHere(g)
	knPrune(g, day)
}

func knEnter(g *Game, zn *Zone) {
	if g.P == nil || zn == nil || zn.Kind != "town" || zn.Burg <= 0 || zn.Z != 0 {
		return
	}
	if lvEnsureTown != nil {
		lvEnsureTown(g, zn.Burg)
	}
	knIngest(g, zn.Burg, lvDay(g))
	knVerifyHere(g)
}

// knActive: the player's burg and up to three seeded neighbours.
func knActive(g *Game) []int {
	var out []int
	px, py := g.Cur.WX, g.Cur.WY
	if g.Cur.Burg > 0 && g.Cur.Kind == "town" {
		out = append(out, g.Cur.Burg)
		lvEnsureTown(g, g.Cur.Burg)
	}
	type bd struct{ id, d int }
	var near []bd
	for i := range g.W.Burgs {
		b := &g.W.Burgs[i]
		if !b.Alive || b.ID == g.Cur.Burg || !lvSt(g).Seeded[b.ID] {
			continue
		}
		if d := maxi(abs(b.P%g.W.W-px), abs(b.P/g.W.W-py)); d <= 14 {
			near = append(near, bd{b.ID, d})
		}
	}
	sort.Slice(near, func(i, j int) bool {
		if near[i].d != near[j].d {
			return near[i].d < near[j].d
		}
		return near[i].id < near[j].id
	})
	for i := 0; i < len(near) && i < 3; i++ {
		out = append(out, near[i].id)
	}
	return out
}

func knSpread(g *Game, b, d int) {
	st := knSt(g)
	ppl := lvPresent(g, b)
	if len(ppl) < 2 {
		return
	}
	r := g.knRng("sp:", b, ":", d)
	off := r.Int(0, len(ppl)-1)
	for k := 0; k < len(ppl) && k < 12; k++ {
		p := ppl[(k+off)%len(ppl)]
		if len(p.Knows) == 0 || !r.Chance(0.04+float64(p.Trait[2])/100*0.22) {
			continue
		}
		var cand []*knFact
		for _, id := range p.Knows {
			if f := st.fact(id); f != nil && f.Subject != p.Key && (f.Kind != "secret" || f.Lvl > 2) {
				if f.Kind == "secret" && !knFriend(p, f.Subject) {
					continue
				}
				cand = append(cand, f)
			}
		}
		if len(cand) == 0 {
			continue
		}
		f := cand[r.Int(0, len(cand)-1)]
		var rec []*lvPerson
		for _, o := range ppl {
			if o.Key != p.Key && o.Key != f.Subject && !knKnown(o, f.ID) {
				rec = append(rec, o)
			}
		}
		if len(rec) == 0 {
			continue
		}
		o := rec[r.Int(0, len(rec)-1)]
		give := f
		dc := 0.06 + float64(100-p.Trait[3])/100*0.05 + float64(p.Trait[4])/100*0.05
		if r.Chance(dc) {
			if nf := knDistort(g, f, p, r); nf != nil {
				give = nf
			}
		}
		knGive(o, give.ID)
		f.Spread++
		give.Spread++
	}
}

func knFriend(p *lvPerson, key string) bool {
	if p.Spouse == key || p.Boss == key {
		return true
	}
	for _, l := range [][]string{p.Kin, p.Friends} {
		for _, k := range l {
			if k == key {
				return true
			}
		}
	}
	return false
}

// knTravel: people who arrived from elsewhere tell locals what they carry; a sighting marks the arrival.
func knTravel(g *Game, b, d int) {
	st := knSt(g)
	r := g.knRng("tr:", b, ":", d)
	n := 0
	for _, p := range lvPresent(g, b) {
		last, had := st.Last[p.Key]
		st.Last[p.Key] = b
		if !had || last == b || last == 0 || n >= 6 {
			continue
		}
		n++
		f := st.add(knFact{Kind: "sighting", Subject: p.Key, Text: fmt.Sprintf("%s has come to %s from %s", p.Name, g.knBurgName(b), g.knBurgName(last)), True: true, Day: d, Burg: b, Source: p.Key})
		locals := knPeople(g, b, 3, r, p.Key)
		for _, o := range locals {
			knGive(o, f.ID)
		}
		var carry []*knFact
		for _, id := range p.Knows {
			if cf := st.fact(id); cf != nil && cf.Kind != "secret" && cf.Subject != "player" && d-cf.Day < 50 {
				carry = append(carry, cf)
			}
		}
		for i := 0; i < 2 && len(carry) > 0 && len(locals) > 0; i++ {
			cf := carry[r.Int(0, len(carry)-1)]
			o := locals[r.Int(0, len(locals)-1)]
			give := cf
			if r.Chance(0.15) {
				if nf := knDistort(g, cf, p, r); nf != nil {
					give = nf
				}
			}
			if knGive(o, give.ID) {
				cf.Spread++
			}
		}
	}
}

func knForget(g *Game, b, d int) {
	st := knSt(g)
	for _, p := range lvPresent(g, b) {
		keep := p.Knows[:0]
		for _, id := range p.Knows {
			f := st.fact(id)
			if f == nil {
				continue
			}
			limit := 60 + int(HashStr(fmt.Sprint(p.Key, ":", id))%40)
			if f.Case > 0 {
				limit = 130
			} else if f.Kind == "crime" {
				limit = 100
			}
			if d-f.Day > limit {
				continue
			}
			keep = append(keep, id)
		}
		p.Knows = keep
	}
}

// knDrift: coarse spread across the world, a few facts reaching people far away.
func knDrift(g *Game, d, span int) {
	st := knSt(g)
	var ids []int
	for id, f := range st.Facts {
		if (f.Kind == "event" || f.Kind == "rumour" || f.Kind == "crime") && d-f.Day < 80 && f.Subject != "player" && f.Lvl == 0 {
			ids = append(ids, id)
		}
	}
	if len(ids) == 0 {
		return
	}
	sort.Ints(ids)
	var keys []string
	for k, p := range lvSt(g).People {
		if p.Alive {
			keys = append(keys, k)
		}
	}
	if len(keys) == 0 {
		return
	}
	sort.Strings(keys)
	r := g.knRng("drift:", d)
	for i := 0; i < 3*mini(span, 4); i++ {
		f := st.fact(ids[r.Int(0, len(ids)-1)])
		p := lvSt(g).People[keys[r.Int(0, len(keys)-1)]]
		if f == nil || p == nil || p.Burg == f.Burg || p.Burg == 0 {
			continue
		}
		give := f
		if r.Chance(0.25) {
			if nf := knDistort(g, f, p, r); nf != nil {
				give = nf
			}
		}
		if knGive(p, give.ID) {
			f.Spread++
		}
	}
}

func knPrune(g *Game, day int) {
	st := knSt(g)
	for k, v := range st.Seen {
		limit := 700
		if strings.HasPrefix(k, "ps:") || strings.HasPrefix(k, "pl:") || strings.HasPrefix(k, "cp:") {
			limit = 6
		}
		if day-v > limit && v > 0 {
			delete(st.Seen, k)
		}
	}
	for k, id := range st.Memo {
		if st.Facts[id] == nil {
			delete(st.Memo, k)
		}
	}
	if len(st.Facts) <= 400 {
		return
	}
	ref := map[int]bool{}
	for _, l := range st.Learned {
		ref[l.Fact] = true
		if f := st.fact(l.Fact); f != nil && f.Orig > 0 {
			ref[f.Orig] = true
		}
	}
	for _, c := range st.Cases {
		for _, id := range c.FactIDs {
			ref[id] = true
		}
	}
	var ids []int
	for id := range st.Facts {
		if !ref[id] {
			ids = append(ids, id)
		}
	}
	sort.Slice(ids, func(i, j int) bool {
		a, b := st.Facts[ids[i]], st.Facts[ids[j]]
		if a.Day != b.Day {
			return a.Day < b.Day
		}
		return a.ID < b.ID
	})
	for _, id := range ids {
		if len(st.Facts) <= 340 {
			break
		}
		delete(st.Facts, id)
	}
}

// knKnowledgeOf: the facts a person knows at creation (lvKnowledgeOf).
func knKnowledgeOf(g *Game, p *lvPerson) []int {
	st := knSt(g)
	d := lvDay(g)
	knIngest(g, p.Home, d)
	var ids []int
	for id, f := range st.Facts {
		if (f.Burg == p.Home || f.Burg == p.Burg) && f.Kind != "secret" && f.Lvl == 0 && f.Subject != p.Key && f.Subject != "player" && d-f.Day < 60 {
			ids = append(ids, id)
		}
	}
	sort.Ints(ids)
	r := g.knRng("start:", p.Key)
	var out []int
	for _, id := range ids {
		if len(out) >= r.Int(1, 4) {
			break
		}
		if r.Chance(0.35) {
			out = append(out, id)
		}
	}
	return out
}

// ---- verification and reliability ----

func (st *knState) relOf(key string) *knRel {
	r := st.Rel[key]
	if r == nil {
		r = &knRel{}
		st.Rel[key] = r
	}
	return r
}

func knRelLabel(r *knRel) string {
	if r == nil || r.Hit+r.Miss == 0 {
		return "{{K|untested}}"
	}
	switch {
	case r.Miss == 0 || r.Hit >= 2*r.Miss:
		return "{{G|reliable}}"
	case r.Hit == 0 || r.Miss >= 2*r.Hit:
		return "{{R|liar}}"
	}
	return "{{y|uncertain}}"
}

// knSetVer marks a fact verified true or false and updates every informant who gave it.
func (g *Game) knSetVer(id, v int) {
	st := knSt(g)
	f := st.fact(id)
	if f == nil || f.Ver == v {
		return
	}
	f.Ver = v
	seen := map[string]bool{}
	for _, l := range st.Learned {
		if l.Fact != id || seen[l.From] {
			continue
		}
		seen[l.From] = true
		r := st.relOf(l.From)
		if v > 0 {
			r.Hit++
		} else {
			r.Miss++
			if r.Miss > r.Hit && !st.Firsts["liar"] {
				st.Firsts["liar"] = true
				g.deed("kn", fmt.Sprintf("The wanderer %s caught an informant in a lie and learned to weigh what people say.", g.P.Name), 0, 0)
			}
		}
	}
}

// knVerifyHere checks learned sightings against where people are now.
func knVerifyHere(g *Game) {
	if g.Cur == nil || g.Cur.Burg <= 0 || g.Cur.Kind != "town" {
		return
	}
	st := knSt(g)
	d := lvDay(g)
	for _, l := range st.Learned {
		f := st.fact(l.Fact)
		if f == nil || f.Ver != 0 || f.Kind != "sighting" || f.Burg != g.Cur.Burg {
			continue
		}
		p := lvGet(g, f.Subject)
		if p == nil {
			continue
		}
		if p.Alive && p.Burg == g.Cur.Burg {
			g.knSetVer(f.ID, 1)
		} else if d-f.Day <= 1 && f.True == false {
			g.knSetVer(f.ID, -1)
		}
	}
}

// knConflicts flags learned facts that contradict each other.
func knConflicts(g *Game) map[int][]int {
	st := knSt(g)
	var ids []int
	seen := map[int]bool{}
	for _, l := range st.Learned {
		if !seen[l.Fact] && st.fact(l.Fact) != nil {
			seen[l.Fact] = true
			ids = append(ids, l.Fact)
		}
	}
	out := map[int][]int{}
	for i := 0; i < len(ids); i++ {
		for j := i + 1; j < len(ids); j++ {
			a, b := st.fact(ids[i]), st.fact(ids[j])
			c := false
			switch {
			case a.Orig == b.ID || b.Orig == a.ID || (a.Orig != 0 && a.Orig == b.Orig):
				c = true
			case a.Case > 0 && a.Case == b.Case && a.Culprit != "" && b.Culprit != "" && a.Culprit != b.Culprit:
				c = true
			case a.Kind == "sighting" && b.Kind == "sighting" && a.Subject == b.Subject && a.Burg != b.Burg && abs(a.Day-b.Day) <= 1:
				c = true
			}
			if c {
				out[a.ID] = append(out[a.ID], b.ID)
				out[b.ID] = append(out[b.ID], a.ID)
			}
		}
	}
	return out
}

// ---- learning ----

func (g *Game) knLearn(f *knFact, from *lvPerson, trust int, via string) {
	st := knSt(g)
	for i := range st.Learned {
		if st.Learned[i].Fact == f.ID && st.Learned[i].From == from.Key {
			if trust > st.Learned[i].Trust {
				st.Learned[i].Trust = trust
			}
			return
		}
	}
	st.Learned = append(st.Learned, knLearn{Fact: f.ID, From: from.Key, Name: from.Name, Day: lvDay(g), Trust: trust, Via: via})
	if len(st.Learned) > 300 {
		st.Learned = st.Learned[len(st.Learned)-300:]
	}
	if !st.Firsts["first"] {
		st.Firsts["first"] = true
		g.deed("kn", fmt.Sprintf("The wanderer %s began keeping notes on what folk whisper.", g.P.Name), 0, 0)
		g.Msg("{{c|You start a page of case notes. Open it from your abilities, or by talking to someone and comparing notes.}}")
	}
	if len(knConflicts(g)) > 0 && !st.Firsts["conflict"] {
		st.Firsts["conflict"] = true
		g.Msg("{{y|Two accounts in your notes disagree. Someone is wrong, or lying.}}")
		g.deed("kn", fmt.Sprintf("The wanderer %s noticed that two witnesses could not both be telling the truth.", g.P.Name), 0, 0)
	}
}

// ---- gossip and news hooks ----

func knTone(p *lvPerson, r *Rng) string {
	switch {
	case p.Trait[4] > 65:
		return Pick(r, knToneNerv)
	case p.Trait[3] < 35:
		return Pick(r, knToneSneer)
	case p.Trait[2] > 65:
		return Pick(r, knToneExtra)
	case p.Trait[1] > 65:
		return Pick(r, knToneCons)
	case p.Trait[0] > 70:
		return Pick(r, knToneOpen)
	}
	return Pick(r, knToneMild)
}

// knSpeak returns what a person actually says of a fact: usually the truth as they know it, sometimes an embellishment.
func (g *Game) knSpeak(p *lvPerson, f *knFact, paid bool) *knFact {
	st := knSt(g)
	prob := float64(100-p.Trait[3]) * float64(100-p.Trait[1]) / 10000 * 0.3
	if p.Opinion < -30 {
		prob += 0.15
	}
	if paid {
		prob *= 1.5
	}
	if f.Source == p.Key {
		prob *= 0.5
	}
	if f.Kind == "secret" {
		prob = 0
	}
	h := float64(HashStr(g.Seed+":kn:lie:"+p.Key+fmt.Sprint(f.ID))%1000) / 1000
	if h >= prob {
		return f
	}
	key := fmt.Sprint(p.Key, ":lie:", f.ID)
	if id, ok := st.Memo[key]; ok && st.fact(id) != nil {
		return st.fact(id)
	}
	nf := knDistort(g, f, p, g.knRng("lie:", p.Key, ":", f.ID))
	if nf == nil {
		return f
	}
	st.Memo[key] = nf.ID
	knGive(p, nf.ID)
	return nf
}

func (g *Game) knSpeakable(p *lvPerson, lvl int, crime bool) []*knFact {
	st := knSt(g)
	var out []*knFact
	for _, id := range p.Knows {
		f := st.fact(id)
		if f == nil || f.Lvl > lvl || f.Subject == p.Key {
			continue
		}
		if f.Kind == "secret" && lvl < 2 {
			continue
		}
		isCrime := f.Kind == "crime"
		if crime != isCrime {
			continue
		}
		out = append(out, f)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Day != out[j].Day {
			return out[i].Day > out[j].Day
		}
		return out[i].ID > out[j].ID
	})
	return out
}

// knGossip: one rumour from what the speaker knows, as a quote, or "" (use the usual gossip).
func (g *Game) knGossip(a *Actor) string {
	if a == nil || g.W == nil {
		return ""
	}
	p := lvPersonOf(g, a)
	if p == nil {
		return ""
	}
	r := NewRng(fmt.Sprint(g.Seed, ":kn:gossip:", a.ID, ":", g.Turn/600))
	if !r.Chance(0.6) {
		return ""
	}
	facts := g.knSpeakable(p, 0, false)
	var fs []*knFact
	for _, f := range facts {
		if f.Subject != "player" && f.Kind != "location" {
			fs = append(fs, f)
		}
	}
	if len(fs) == 0 {
		return ""
	}
	f := fs[r.Int(0, mini(len(fs)-1, 5))]
	said := g.knSpeak(p, f, false)
	g.knLearn(said, p, knLevel(g, a, p), "gossip")
	return fmt.Sprintf("\"%s %s.\" {{K|(%s)}}", Pick(r, knOpeners), lowerFirst(said.Text), knTone(p, r))
}

// knNews: lines about what is being talked of widely, for wsNewsLines.
func (g *Game) knNews() []wsLine {
	if g.W == nil || g.Cur == nil {
		return nil
	}
	st := knSt(g)
	d := lvDay(g)
	var ids []int
	for id, f := range st.Facts {
		if f.Spread >= 3 && d-f.Day < 30 && f.Kind != "secret" && f.Lvl == 0 && f.Subject != "player" {
			ids = append(ids, id)
		}
	}
	sort.Slice(ids, func(i, j int) bool {
		a, b := st.Facts[ids[i]], st.Facts[ids[j]]
		if a.Spread != b.Spread {
			return a.Spread > b.Spread
		}
		return a.ID < b.ID
	})
	var out []wsLine
	for _, id := range ids {
		if len(out) >= 3 {
			break
		}
		f := st.Facts[id]
		bd := g.W.burgByID[f.Burg]
		if bd == nil {
			continue
		}
		out = append(out, wsLine{bd.P, fmt.Sprintf("{{y|Talk in %s:}} %s.", bd.Name, strings.TrimSuffix(f.Text, "."))})
	}
	return out
}

// ---- trust ----

// knLevel is how much a person trusts you: -1 refuses, 0 hints, 1 details, 2 names and secrets.
func knLevel(g *Game, a *Actor, p *lvPerson) int {
	if a.Angry {
		return -1
	}
	t := p.Opinion + clampi(g.Rep[a.Faction]/8, -30, 30)
	if p.Met {
		t += 8
	}
	switch {
	case t < -25:
		return -1
	case t < 15:
		return 0
	case t < 45:
		return 1
	}
	return 2
}

func knTrustScore(g *Game, a *Actor, p *lvPerson) int {
	return p.Opinion + clampi(g.Rep[a.Faction]/8, -30, 30)
}

// ---- talk ----

func knTalk(g *Game, a *Actor) []Option {
	if g == nil || a == nil || g.W == nil || a.AI == "hostile" || a.AI == "follower" {
		return nil
	}
	p := lvPersonOf(g, a)
	if p == nil {
		return nil
	}
	var opts []Option
	opts = append(opts, Option{"{{c|Ask around...}}", func() { g.knAskMenu(a, p) }})
	opts = append(opts, Option{"{{c|Compare notes.}}", func() { g.knOpen(true) }})
	st := knSt(g)
	for _, c := range st.Cases {
		if c.Status != "open" {
			continue
		}
		c := c
		if knSupport(g, c, p.Key) > 0 {
			opts = append(opts, Option{fmt.Sprintf("{{R|Accuse %s: %s.}}", p.Name, c.Title), func() { g.knAccuse(a, p, c) }})
		}
	}
	return opts
}

func (g *Game) knShow(text string, opts []Option) {
	g.Popup = Wrap(text, 100)
	g.Opts = append(opts, Option{"Back.", func() { g.talkMenu("\"Anything else?\"") }})
	g.Sel = 0
}

func (g *Game) knAskMenu(a *Actor, p *lvPerson) {
	st := knSt(g)
	lvl := knLevel(g, a, p)
	if lvl < 0 {
		g.knShow(Pick(NewRng(p.Key), knRefusals), nil)
		return
	}
	dc := clampi(9-knTrustScore(g, a, p)/12, 5, 13)
	price := knPrice(g)
	pdc := 8 + p.Trait[1]/25
	var opts []Option
	opts = append(opts, Option{"Ask what they have heard.", func() { g.knAsk(a, p, "heard", "plain") }})
	opts = append(opts, Option{g.checkLabel("persuasion", dc) + " Coax out more gossip.", func() { g.knAsk(a, p, "heard", "persuade") }})
	opts = append(opts, Option{fmt.Sprintf("{{Y|Offer coin for information.}} {{K|(%s)}}", g.cuFormat(price)), func() { g.knAsk(a, p, "heard", "coin") }})
	opts = append(opts, Option{g.checkLabel("intimidate", pdc) + " {{R|Press them.}}", func() { g.knAsk(a, p, "heard", "press") }})
	if n := len(g.knSpeakable(p, 2, true)); n > 0 {
		opts = append(opts, Option{"Ask about the crime.", func() { g.knCrimeMenu(a, p) }})
	}
	if len(g.knSubjects(p)) > 0 {
		opts = append(opts, Option{"Ask about someone.", func() { g.knPeopleMenu(a, p) }})
	}
	_ = st
	g.knShow("\"What is it you want to know?\"", opts)
}

func knPrice(g *Game) int { return mini(60, 6+3*g.P.Level) }

func (g *Game) knCrimeMenu(a *Actor, p *lvPerson) {
	dc := clampi(9-knTrustScore(g, a, p)/12, 5, 13)
	pdc := 8 + p.Trait[1]/25
	price := knPrice(g)
	opts := []Option{
		{"Ask what they saw.", func() { g.knAsk(a, p, "crime", "plain") }},
		{g.checkLabel("persuasion", dc) + " Appeal to their conscience.", func() { g.knAsk(a, p, "crime", "persuade") }},
		{fmt.Sprintf("{{Y|Offer coin.}} {{K|(%s)}}", g.cuFormat(price)), func() { g.knAsk(a, p, "crime", "coin") }},
		{g.checkLabel("intimidate", pdc) + " {{R|Lean on them.}}", func() { g.knAsk(a, p, "crime", "press") }},
	}
	g.knShow("\"The crime? Aye. What of it?\"", opts)
}

// knAsk runs one questioning: topic heard|crime, mode plain|persuade|coin|press.
func (g *Game) knAsk(a *Actor, p *lvPerson, topic, mode string) {
	st := knSt(g)
	d := lvDay(g)
	key := fmt.Sprint(p.Key, ":", topic, ":", mode)
	if st.Asked[key] == d+1 {
		g.knShow("\"I have told you what I will today.\"", nil)
		return
	}
	st.Asked[key] = d + 1
	lvl := knLevel(g, a, p)
	maxN := 3
	via := "ask"
	prefix := ""
	dc := clampi(9-knTrustScore(g, a, p)/12, 5, 13)
	switch mode {
	case "persuade":
		if g.roll2d6Check(g.checkMod("persuasion"), dc) {
			lvl = mini(2, lvl+1)
			maxN = 4
			prefix = "{{G|[Persuasion: Success]}}\n\n"
		} else {
			g.knShow("{{R|[Persuasion: Failure]}}\n\n\"I have said enough.\" They turn away.", nil)
			return
		}
	case "coin":
		if p.Opinion < -60 {
			g.knShow(Pick(NewRng(p.Key), knRefusals), nil)
			return
		}
		if !g.cuSpend(knPrice(g)) {
			g.knShow("\"Come back with coin.\"", nil)
			st.Asked[key] = 0
			return
		}
		lvl = mini(2, maxi(lvl, 0)+1)
		maxN = 4
		via = "paid"
		prefix = "{{y|The coin vanishes into their sleeve.}}\n\n"
	case "press":
		pdc := 8 + p.Trait[1]/25
		if g.roll2d6Check(g.checkMod("intimidate"), pdc) {
			lvl = 2
			maxN = 6
			via = "pressed"
			prefix = "{{G|[Intimidate: Success]}}\n\n"
			p.remember(d, "insult", "pressed me for what I know", -6)
		} else {
			p.remember(d, "insult", "threatened me", -10)
			p.Opinion = clampi(p.Opinion-5, -100, 100)
			msg := "{{R|[Intimidate: Failure]}}\n\n\"Get out of my face!\""
			if g.Cur != nil && g.Cur.Burg > 0 && g.R.Chance(0.4) {
				a.Angry = true
				g.uwAlertGuards(g.uwStateHere(), g.Cur)
				msg += "\n\n{{R|They shout for the watch.}}"
			}
			g.knShow(msg, nil)
			return
		}
	default:
		if lvl < 0 {
			g.knShow(Pick(NewRng(p.Key), knRefusals), nil)
			return
		}
		if lvl >= 1 {
			maxN = 4
		}
	}
	if lvl < 0 {
		lvl = 0
	}
	facts := g.knSpeakable(p, lvl, topic == "crime")
	if topic == "heard" {
		var tmp []*knFact
		for _, f := range facts {
			if f.Kind != "location" || lvl >= 1 {
				tmp = append(tmp, f)
			}
		}
		facts = tmp
	}
	if len(facts) == 0 {
		g.knShow(prefix+Pick(NewRng(p.Key+fmt.Sprint(d)), knNothing), nil)
		return
	}
	// new facts first
	learned := map[int]bool{}
	for _, l := range st.Learned {
		if l.From == p.Key {
			learned[l.Fact] = true
		}
	}
	sort.SliceStable(facts, func(i, j int) bool { return !learned[facts[i].ID] && learned[facts[j].ID] })
	if len(facts) > maxN {
		facts = facts[:maxN]
	}
	r := NewRng(fmt.Sprint(g.Seed, ":kn:tone:", p.Key, ":", d))
	var b strings.Builder
	b.WriteString(prefix)
	for i, f := range facts {
		said := g.knSpeak(p, f, via == "paid")
		g.knLearn(said, p, lvl, via)
		first := ""
		if said.Source == p.Key {
			first = " {{K|(They saw it themselves.)}}"
		} else if said.Kind != "secret" {
			first = " {{K|(Heard it from someone.)}}"
		}
		if i > 0 {
			b.WriteString("\n\n")
		}
		fmt.Fprintf(&b, "{{K|%s:}} \"%s.\"%s", knTone(p, r), strings.TrimSuffix(said.Text, "."), first)
	}
	g.knShow(b.String(), nil)
	g.gainXP(2 + len(facts))
}

// knSubjects lists people the speaker knows something about, or is close to.
func (g *Game) knSubjects(p *lvPerson) []*lvPerson {
	st := knSt(g)
	seen := map[string]bool{p.Key: true, "player": true}
	var out []*lvPerson
	add := func(k string) {
		if k == "" || seen[k] || len(out) >= 6 {
			return
		}
		if q := lvGet(g, k); q != nil {
			seen[k] = true
			out = append(out, q)
		}
	}
	for i := len(p.Knows) - 1; i >= 0; i-- {
		if f := st.fact(p.Knows[i]); f != nil && f.Kind != "secret" {
			add(f.Subject)
		}
	}
	add(p.Spouse)
	add(p.Boss)
	for _, l := range [][]string{p.Friends, p.Kin, p.Rivals} {
		if len(l) > 0 {
			add(l[0])
		}
	}
	return out
}

func (g *Game) knPeopleMenu(a *Actor, p *lvPerson) {
	var opts []Option
	for _, q := range g.knSubjects(p) {
		q := q
		opts = append(opts, Option{fmt.Sprintf("%s {{K|(%s)}}", q.Name, q.Role), func() { g.knPersonMenu(a, p, q) }})
	}
	g.knShow("\"Who?\"", opts)
}

func (g *Game) knPersonMenu(a *Actor, p, q *lvPerson) {
	opts := []Option{
		{fmt.Sprintf("What do you know of %s?", q.Name), func() { g.knAbout(a, p, q) }},
		{fmt.Sprintf("Where is %s?", q.Name), func() { g.knWhere(a, p, q) }},
	}
	g.knShow(fmt.Sprintf("\"%s? What of them?\"", q.Name), opts)
}

func (g *Game) knAbout(a *Actor, p, q *lvPerson) {
	st := knSt(g)
	lvl := knLevel(g, a, p)
	if lvl < 0 {
		g.knShow(Pick(NewRng(p.Key), knRefusals), nil)
		return
	}
	var lines []string
	for _, id := range p.Knows {
		f := st.fact(id)
		if f == nil || f.Subject != q.Key || f.Kind == "sighting" || f.Lvl > lvl {
			continue
		}
		said := g.knSpeak(p, f, false)
		g.knLearn(said, p, lvl, "ask")
		lines = append(lines, fmt.Sprintf("\"%s.\"", strings.TrimSuffix(said.Text, ".")))
		if len(lines) >= 3 {
			break
		}
	}
	rel := ""
	switch {
	case p.Spouse == q.Key:
		rel = "That is my spouse."
	case p.Boss == q.Key:
		rel = "They are my master."
	case knFriend(p, q.Key):
		rel = "We are close."
	}
	for _, k := range p.Rivals {
		if k == q.Key {
			rel = "We do not get on."
		}
	}
	if len(lines) == 0 {
		g.knShow(rel+" \"I know little of them.\"", nil)
		return
	}
	g.knShow(strings.TrimSpace(rel+"\n\n"+strings.Join(lines, "\n\n")), nil)
}

func (g *Game) knAgo(d int) string {
	n := lvDay(g) - d
	switch {
	case n <= 0:
		return "today"
	case n == 1:
		return "yesterday"
	}
	return fmt.Sprintf("%d days ago", n)
}

func (g *Game) knWhere(a *Actor, p, q *lvPerson) {
	st := knSt(g)
	lvl := knLevel(g, a, p)
	if lvl < 0 {
		g.knShow(Pick(NewRng(p.Key), knRefusals), nil)
		return
	}
	d := lvDay(g)
	if !q.Alive {
		g.knShow(fmt.Sprintf("\"%s? They died in the year %d.\"", q.Name, q.Died), nil)
		return
	}
	var best *knFact
	for _, id := range p.Knows {
		if f := st.fact(id); f != nil && f.Kind == "sighting" && f.Subject == q.Key && (best == nil || f.Day > best.Day) {
			best = f
		}
	}
	if best == nil && knFriend(p, q.Key) {
		key := fmt.Sprint(p.Key, ":where:", q.Key, ":", d)
		if id, ok := st.Memo[key]; ok && st.fact(id) != nil {
			best = st.fact(id)
		} else {
			bb := q.Burg
			txt := fmt.Sprintf("%s is in %s", q.Name, g.knBurgName(bb))
			if bb == 0 && q.Dest > 0 {
				bb = q.Dest
				txt = fmt.Sprintf("%s set out for %s", q.Name, g.knBurgName(q.Dest))
			}
			best = st.add(knFact{Kind: "sighting", Subject: q.Key, Text: txt, True: true, Day: d - 1, Burg: bb, Source: p.Key})
			st.Memo[key] = best.ID
			knGive(p, best.ID)
		}
	}
	if best == nil {
		g.knShow("\"I couldn't say where "+q.Name+" is.\"", nil)
		return
	}
	said := g.knSpeak(p, best, false)
	g.knLearn(said, p, lvl, "ask")
	g.knShow(fmt.Sprintf("\"%s. That was %s.\"", strings.TrimSuffix(said.Text, "."), g.knAgo(said.Day)), nil)
}

// ---- accusation ----

// knSupport counts the learned, distinct facts of a case that point at a person.
func knSupport(g *Game, c *knCase, key string) int {
	st := knSt(g)
	seen := map[int]bool{}
	for _, l := range st.Learned {
		if f := st.fact(l.Fact); f != nil && f.Case == c.ID && f.Culprit == key && f.Ver >= 0 {
			seen[f.ID] = true
		}
	}
	return len(seen)
}

func (g *Game) knAccuse(a *Actor, p *lvPerson, c *knCase) {
	st := knSt(g)
	d := lvDay(g)
	n := knSupport(g, c, p.Key)
	realm := g.fmRealm()
	if n < 3 {
		p.Opinion = clampi(p.Opinion-8, -100, 100)
		g.knShow(fmt.Sprintf("\"%s\" They laugh in your face.\n\n{{y|You have %d consistent fact(s) against them. You need 3.}}", "Hearsay! Bring me proof.", n), nil)
		return
	}
	pts := map[int]bool{}
	for _, l := range st.Learned {
		if f := st.fact(l.Fact); f != nil && f.Case == c.ID {
			pts[f.ID] = true
		}
	}
	if p.Key != c.Culprit {
		c.Wrong++
		p.remember(d, "insult", "accused me falsely of a crime", -25)
		g.Rep[a.Faction] -= 15
		g.fmFame(realm, -6)
		for id := range pts {
			if f := st.fact(id); f != nil && f.Culprit == p.Key {
				g.knSetVer(id, -1)
			}
		}
		if c.Wrong >= 2 {
			c.Status = "failed"
		}
		g.knShow("\"You accuse ME? On whose word?\" The crowd mutters against you.\n\n{{R|A false accusation. Your name suffers, and so does their opinion of you.}}", nil)
		return
	}
	for id := range pts {
		f := st.fact(id)
		if f == nil {
			continue
		}
		if f.Culprit == c.Culprit {
			g.knSetVer(id, 1)
		} else {
			g.knSetVer(id, -1)
		}
	}
	c.Status = "solved"
	st.Solved++
	var out string
	switch {
	case p.Trait[3] >= 55 || p.Trait[4] >= 60:
		c.Outcome = "confession"
		out = fmt.Sprintf("\"...Enough. Yes. It was me.\" %s breaks down and confesses.", p.Name)
		p.addLife(lvYear(g), "crime", "confessed when the wanderer laid out the proof")
		p.Mood = clampi(p.Mood-30, -100, 100)
	case p.Trait[3] < 35:
		c.Outcome = "duel"
		out = fmt.Sprintf("\"You'll prove it with steel!\" %s draws a blade.", p.Name)
		a.Angry = true
	default:
		c.Outcome = "arrest"
		out = fmt.Sprintf("The watch is called. %s is led away in irons.", p.Name)
		p.addLife(lvYear(g), "crime", "was arrested after being named")
		a.AI = "still"
		a.Desc = strings.TrimSpace(a.Desc + " In custody.")
		p.Mood = clampi(p.Mood-40, -100, 100)
	}
	coin := 30 + 10*n
	g.gainXP(60 + 10*n)
	g.fmFame(realm, 5)
	reward := g.cuEarn(coin)
	if !st.Firsts["solved"] {
		st.Firsts["solved"] = true
	}
	g.deed("kn", fmt.Sprintf("The wanderer %s pieced together the truth of \"%s\" from gossip and named %s.", g.P.Name, c.Title, p.Name), 0, 0)
	g.knShow(fmt.Sprintf("{{G|Case solved: %s.}}\n\n%s\n\nYou are rewarded %s, and the realm thinks well of you.", c.Title, out, reward), nil)
}

// ---- case notes screen ----

func (g *Game) knOpen(fromTalk bool) {
	st := knSt(g)
	st.tab, st.sel, st.fromTalk = 0, 0, fromTalk
	g.Mode = "kn"
}

func (g *Game) knAbilities() []Ability {
	if g.Cur == nil || g.P == nil {
		return nil
	}
	st := knSt(g)
	if len(st.Learned) == 0 && len(st.Cases) == 0 {
		return nil
	}
	return []Ability{{ID: "kn_notes", Name: "Case notes", Target: "self"}}
}

func (g *Game) knAbUse(id string) {
	if id == "kn_notes" {
		g.knOpen(false)
	}
}

type knRow struct {
	text string
	head bool
	fact int
}

func (g *Game) knRows() []knRow {
	st := knSt(g)
	conf := knConflicts(g)
	var rows []knRow
	switch st.tab {
	case 0:
		for _, c := range st.Cases {
			status := map[string]string{"open": "{{y|open}}", "solved": "{{G|solved}}", "failed": "{{R|failed}}"}[c.Status]
			rows = append(rows, knRow{text: fmt.Sprintf("%s {{K|[%s, day %d]}} %s", c.Title, g.knBurgName(c.Burg), c.Day, status), fact: c.ID})
		}
	case 1:
		groups := map[string][]int{}
		var order []string
		seen := map[int]bool{}
		for _, l := range st.Learned {
			f := st.fact(l.Fact)
			if f == nil || seen[f.ID] {
				continue
			}
			seen[f.ID] = true
			sub := "Other"
			switch {
			case f.Subject == "player":
				sub = "About you"
			case strings.HasPrefix(f.Subject, "burg:"):
				sub = "Places and happenings"
			default:
				if p := lvGet(g, f.Subject); p != nil {
					sub = p.Name
				}
			}
			if _, ok := groups[sub]; !ok {
				order = append(order, sub)
			}
			groups[sub] = append(groups[sub], f.ID)
		}
		sort.Strings(order)
		for _, sub := range order {
			rows = append(rows, knRow{text: "{{C|" + sub + "}}", head: true})
			for _, id := range groups[sub] {
				f := st.fact(id)
				mark := ""
				switch {
				case f.Ver > 0:
					mark = " {{G|[true]}}"
				case f.Ver < 0:
					mark = " {{R|[false]}}"
				case len(conf[id]) > 0:
					mark = " {{y|[contradicted]}}"
				}
				rows = append(rows, knRow{text: knShort(f.Text, 62) + mark, fact: id})
			}
		}
	case 2:
		var keys []string
		cnt := map[string]int{}
		name := map[string]string{}
		for _, l := range st.Learned {
			if cnt[l.From] == 0 {
				keys = append(keys, l.From)
			}
			cnt[l.From]++
			name[l.From] = l.Name
		}
		sort.Strings(keys)
		for _, k := range keys {
			rows = append(rows, knRow{text: fmt.Sprintf("%-22s %d told  %s", knShort(name[k], 22), cnt[k], knRelLabel(st.Rel[k])), fact: 0})
		}
	}
	return rows
}

func knKeys(g *Game, k string) {
	st := knSt(g)
	rows := g.knRows()
	switch k {
	case "Escape":
		if st.fromTalk && g.Talk != nil {
			st.fromTalk = false
			g.Mode = "talk"
			g.talkMenu("\"Anything else?\"")
		} else {
			g.Mode = "main"
		}
	case "Tab", "ArrowRight":
		st.tab, st.sel = (st.tab+1)%3, 0
	case "ArrowLeft":
		st.tab, st.sel = (st.tab+2)%3, 0
	case "ArrowUp":
		st.sel = maxi(0, st.sel-1)
		for st.sel > 0 && st.sel < len(rows) && rows[st.sel].head {
			st.sel--
		}
	case "ArrowDown":
		st.sel = mini(maxi(0, len(rows)-1), st.sel+1)
		for st.sel < len(rows)-1 && rows[st.sel].head {
			st.sel++
		}
	}
}

func knDraw(g *Game) {
	st := knSt(g)
	x, y, w, h := g.panel("Case notes")
	s := g.sc
	tabs := []string{"Cases", "Facts learned", "Informants"}
	tx := x
	for i, t := range tabs {
		c := "K"
		if i == st.tab {
			c = "W"
		}
		tx = s.Text(tx, y, "{{"+c+"|"+t+"}}  ", CGray)
	}
	rows := g.knRows()
	if st.sel >= len(rows) {
		st.sel = maxi(0, len(rows)-1)
	}
	listH := h - 12
	if listH < 4 {
		listH = 4
	}
	if len(rows) == 0 {
		msg := []string{"No cases yet. Crimes and mysteries turn up in towns as time passes.", "Nothing learned yet. Talk to people and ask around.", "No informants yet."}[st.tab]
		s.Text(x, y+2, "{{K|"+msg+"}}", CGray)
	}
	top := maxi(0, st.sel-listH+1)
	for i := top; i < len(rows) && i < top+listH; i++ {
		mark := "  "
		if i == st.sel {
			mark = "{{Y|>}} "
		}
		if rows[i].head {
			mark = "  "
		}
		s.Text(x, y+2+i-top, mark+rows[i].text, CGray)
	}
	// detail
	dy := y + 3 + listH
	if st.sel < len(rows) {
		r := rows[st.sel]
		var lines []string
		switch st.tab {
		case 0:
			lines = g.knCaseDetail(r.fact)
		case 1:
			if !r.head {
				lines = g.knFactDetail(r.fact)
			}
		}
		for i, l := range lines {
			if dy+i >= y+h-1 {
				break
			}
			s.Text(x, dy+i, l, CGray)
		}
	}
	_ = w
	s.Text(x, y+h-1, "{{K|Tab/arrows: tabs and list · Esc: close}}", CGray)
}

func (g *Game) knFactDetail(id int) []string {
	st := knSt(g)
	f := st.fact(id)
	if f == nil {
		return nil
	}
	var out []string
	for _, l := range Wrap(f.Text, 90) {
		out = append(out, l)
	}
	var from []string
	for _, l := range st.Learned {
		if l.Fact == id {
			from = append(from, fmt.Sprintf("%s (day %d, %s)", l.Name, l.Day, []string{"low trust", "some trust", "high trust"}[clampi(l.Trust, 0, 2)]))
		}
	}
	out = append(out, "{{K|From: "+strings.Join(from, "; ")+"}}")
	for _, o := range knConflicts(g)[id] {
		if of := st.fact(o); of != nil {
			out = append(out, "{{y|Conflicts with: "+knShort(of.Text, 70)+"}}")
			break
		}
	}
	return out
}

func (g *Game) knCaseDetail(cid int) []string {
	st := knSt(g)
	var c *knCase
	for _, x := range st.Cases {
		if x.ID == cid {
			c = x
		}
	}
	if c == nil {
		return nil
	}
	var out []string
	if c.Status == "solved" {
		cu := lvGet(g, c.Culprit)
		nm := "the culprit"
		if cu != nil {
			nm = cu.Name
		}
		out = append(out, fmt.Sprintf("{{G|Solved: %s (%s).}}", nm, c.Outcome))
	}
	sup := map[string]int{}
	var keys []string
	for _, l := range st.Learned {
		if f := st.fact(l.Fact); f != nil && f.Case == c.ID && f.Culprit != "" {
			seen := false
			for _, k := range keys {
				if k == f.Culprit {
					seen = true
				}
			}
			if !seen {
				keys = append(keys, f.Culprit)
			}
		}
	}
	sort.Strings(keys)
	for _, k := range keys {
		sup[k] = knSupport(g, c, k)
		nm := k
		if p := lvGet(g, k); p != nil {
			nm = p.Name
		}
		tag := ""
		if sup[k] >= 3 {
			tag = " {{G|enough to accuse}}"
		}
		out = append(out, fmt.Sprintf("Points at %s: %d fact(s)%s", nm, sup[k], tag))
	}
	if len(keys) == 0 {
		out = append(out, "{{K|Nothing learned yet. Ask about the crime in "+g.knBurgName(c.Burg)+".}}")
	}
	if len(keys) > 1 {
		out = append(out, "{{y|Your notes point at more than one person. Someone is lying or mistaken.}}")
	}
	return out
}
