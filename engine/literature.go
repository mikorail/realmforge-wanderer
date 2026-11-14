package engine

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// Generated literature (gl): books written about the world's real events by real or invented
// authors with a bias that bends the facts. Everything is a pure function of (world seed, book seed,
// parasang), so a book's text and clue can be rebuilt from Item.Extra and the pack never has to
// store more than the rendered text. No g.R at generation time; GenBook takes one play-time roll.
//
// Item.Extra keys: gl (1), gl:seed, gl:p, gl:rare, gl:ban (state id + 1), gl:ev (event index + 1),
// gl:min (reading minutes), lxc (author's culture + 1, so languages.go garbles foreign books).

const glVis = MH - 2 // text rows on the reading screen

type glClaim struct {
	Title  string `json:"t"`
	Author string `json:"a"`
	Bias   string `json:"b"`
	Year   int    `json:"y"`
	Num    string `json:"n,omitempty"`
	Who    string `json:"w,omitempty"`
	Kind   string `json:"k,omitempty"`
	Val    string `json:"v,omitempty"`
}

type glShelfRow struct {
	Title  string `json:"t"`
	Author string `json:"a"`
	Form   string `json:"f"`
	Bias   string `json:"b"`
}

type glState struct {
	Done   map[string]bool      `json:"d,omitempty"`
	Claims map[string][]glClaim `json:"c,omitempty"`
	Shelf  []glShelfRow         `json:"s,omitempty"`
	Bought map[string]bool      `json:"o,omitempty"`
	Warned map[string]bool      `json:"w,omitempty"`

	View   *Item    `json:"-"`
	Lines  []string `json:"-"`
	Scroll int      `json:"-"`
	Minute int      `json:"-"`
	Shelfv bool     `json:"-"`
}

func (g *Game) glSt() *glState {
	st := g.ext("gl", func() any { return &glState{} }).(*glState)
	if st.Done == nil {
		st.Done = map[string]bool{}
	}
	if st.Claims == nil {
		st.Claims = map[string][]glClaim{}
	}
	if st.Bought == nil {
		st.Bought = map[string]bool{}
	}
	if st.Warned == nil {
		st.Warned = map[string]bool{}
	}
	return st
}

func init() {
	unLocalKeys = append(unLocalKeys, "gl") // event indices, claims and read state belong to one planet
	modeKeys["glr"] = glKeys
	modeDraws["glr"] = glDraw
	talkHooks = append(talkHooks, glTalk)
	enterHooks = append(enterHooks, glEnter)
}

// ---- the book itself ----

type glBk struct {
	Title, Author, Epithet, Form, Bias string
	AuthorState, Cult                  int
	Real                               bool
	Ev                                 int
	Year                               int
	Num, Who                           string
	Kind, Val                          string
	Lie                                bool
	Tech                               int
	Ban                                int // state id + 1, 0 = not proscribed
	Text                               string
	Minutes                            int
	SecretText, SecretKind             string
	SecretP                            int
	Realm                              string
}

var glLiar = map[string]float64{"propagandist": 0.5, "partisan": 0.4, "pious": 0.22, "cynical": 0.15, "romantic": 0.25}

var glFormClues = map[string][]string{
	"history":    {"lore", "crime", "site", "lore"},
	"epic":       {"site", "lore", "identity"},
	"sermon":     {"lore", "identity", "crime"},
	"satire":     {"crime", "identity", "crime"},
	"travelogue": {"site", "technique", "site", "lore"},
	"lovepoem":   {"identity", "lore", "site"},
	"dispatch":   {"site", "crime", "technique"},
	"natural":    {"technique", "site", "lore"},
	"memoir":     {"identity", "crime", "technique", "lore"},
	"pamphlet":   {"crime", "identity", "crime"},
}

func glDist(w *WorldData, a, b int) int {
	if a < 0 || b < 0 || w.W == 0 {
		return 1 << 20
	}
	return abs(a%w.W-b%w.W) + abs(a/w.W-b/w.W)
}

// glPool: indices of the famous events for a region (a realm's top eight, else the world's).
func glPool(w *WorldData, st, p int) []int {
	pick := func(local bool) []int {
		var idx []int
		for i := range w.Events {
			e := &w.Events[i]
			if glKindScore[e.Kind] == 0 {
				continue
			}
			if local && !(e.St == st || (e.P >= 0 && p >= 0 && glDist(w, e.P, p) <= 25)) {
				continue
			}
			idx = append(idx, i)
		}
		sort.SliceStable(idx, func(a, b int) bool {
			ea, eb := &w.Events[idx[a]], &w.Events[idx[b]]
			if sa, sb := glKindScore[ea.Kind], glKindScore[eb.Kind]; sa != sb {
				return sa > sb
			}
			return ea.Y > eb.Y
		})
		return idx
	}
	var idx []int
	if st > 0 {
		idx = pick(true)
		if len(idx) > 8 {
			idx = idx[:8]
		}
	}
	if len(idx) < 3 {
		idx = pick(false)
		if len(idx) > 12 {
			idx = idx[:12]
		}
	}
	return idx
}

// glSkew bends the event text the way the author's bias does. It never uses g.R.
func glSkew(text, bias string, r *Rng, own, evRealm, deity string) (out string, num, who string, dy int) {
	out, who = text, evRealm
	if loc := knDigits.FindStringIndex(text); loc != nil {
		n, _ := strconv.Atoi(text[loc[0]:loc[1]])
		m := n
		if n >= 2 {
			switch bias {
			case "propagandist":
				m = n * r.Int(2, 3)
			case "cynical":
				m = maxi(1, n/2)
			case "partisan":
				m = n + n/2
			case "romantic":
				m = (n + 4) / 5 * 5
			case "pious":
				for _, s := range []int{3, 7, 12, 40} {
					if abs(s-n) <= 2 {
						m = s
					}
				}
			}
		}
		out = text[:loc[0]] + strconv.Itoa(m) + text[loc[1]:]
		num = strconv.Itoa(m)
	}
	switch bias {
	case "propagandist", "partisan":
		if own != "" && own != evRealm && strings.Contains(out, evRealm) {
			out = strings.Replace(out, evRealm, own, 1)
			who = own
		}
		if bias == "partisan" {
			dy = 1
		}
	case "pious":
		who = deity
	case "cynical":
		who = "the moneyed"
	case "romantic":
		dy = r.Int(0, 1)*2 - 1
	}
	return
}

func (g *Game) glStateCulture(st int) *CultureData {
	if s := g.W.stateByID[st]; s != nil {
		return g.W.cultByID[s.Culture]
	}
	return nil
}

func (g *Game) glSiteNear(p0 int, r *Rng) (p int, text, region string, ok bool) {
	w := g.W
	if p0 < 0 || p0 >= len(w.State) {
		return
	}
	cx, cy := p0%w.W, p0/w.W
	for rad := 0; rad <= 10; rad++ {
		for dy := -rad; dy <= rad; dy++ {
			for dx := -rad; dx <= rad; dx++ {
				if maxi(abs(dx), abs(dy)) != rad || !w.In(cx+dx, cy+dy) {
					continue
				}
				pp := w.Idx(cx+dx, cy+dy)
				if w.Burg[pp] > 0 {
					continue
				}
				if s := g.parasangSite(pp); s != nil {
					what := "a dungeon"
					if s.Kind == "tower" {
						what = fmt.Sprintf("a tower of %d floors", s.Depth)
					}
					region = w.RegionName(pp)
					return pp, fmt.Sprintf("%s is %s lying near %s.", s.Name, what, region), region, true
				}
			}
		}
	}
	return
}

// glCompose builds the whole book deterministically.
func (g *Game) glCompose(seed, p int, rare bool) *glBk {
	w := g.W
	if w == nil || len(w.Events) == 0 {
		return nil
	}
	r := NewRng(fmt.Sprintf("%s:gl:%d:%d", g.Seed, seed, p))
	st := 0
	if p >= 0 && p < len(w.State) {
		st = w.State[p]
	}
	pool := glPool(w, st, p)
	if len(pool) == 0 {
		return nil
	}
	ev := pool[r.Int(0, len(pool)-1)]
	e := w.Events[ev]
	evSt := e.St
	if w.stateByID[evSt] == nil {
		evSt = st
	}
	evRealm := w.StateName(evSt)

	var forms []string
	for _, f := range glKindForms[e.Kind] {
		if glForms[f] != nil {
			forms = append(forms, f)
		}
	}
	if len(forms) == 0 {
		forms = []string{"history"}
	}
	form := Pick(r, forms)
	f := glForms[form]
	bias := Pick(r, f.Biases)
	b := &glBk{Form: form, Bias: bias, Ev: ev, Realm: evRealm}

	// author: a real notable or an invented one with a name from the culture's own names
	var cand []*PersonData
	for i := range w.People {
		if w.People[i].State == evSt && w.People[i].Name != "" && w.People[i].Renown > 0 {
			cand = append(cand, &w.People[i])
		}
	}
	sort.SliceStable(cand, func(i, j int) bool { return cand[i].Renown > cand[j].Renown })
	if len(cand) > 12 {
		cand = cand[:12]
	}
	if len(cand) > 0 && r.Chance(0.4) {
		pd := cand[r.Int(0, len(cand)-1)]
		b.Real, b.Author, b.AuthorState, b.Cult = true, pd.Name, pd.State, pd.Culture
		b.Epithet = nonEmpty(pd.Title, pd.Role)
	} else {
		as := evSt
		if len(w.States) > 0 && r.Chance(0.35) {
			as = w.States[r.Int(0, len(w.States)-1)].ID
		}
		cu := g.glStateCulture(as)
		b.AuthorState = as
		if cu != nil {
			b.Cult = cu.ID
		}
		b.Author = g.cultureName(cu, r)
		b.Epithet = glRoleHint[form]
		if form == "pamphlet" && r.Chance(0.5) {
			b.Author, b.Epithet = "A Friend to the Realm", ""
		}
	}
	own := w.StateName(b.AuthorState)

	place := evRealm
	if e.P >= 0 {
		place = w.RegionName(e.P)
	}
	foe := "the neighbours"
	for i := range w.States {
		s := &w.States[i]
		if s.ID != evSt && s.Name != "" && strings.Contains(e.Text, s.Name) {
			foe = s.Name
			break
		}
	}
	if foe == "the neighbours" {
		if s := w.stateByID[evSt]; s != nil && len(s.Wars) > 0 {
			foe = w.StateName(s.Wars[r.Int(0, len(s.Wars)-1)])
		}
	}
	deity := "Heaven"
	if s := w.stateByID[b.AuthorState]; s != nil {
		if rel := w.relByID[s.Religion]; rel != nil && rel.Deity != "" {
			deity = rel.Deity
		}
	}
	ruler := "the ruler"
	if s := w.stateByID[evSt]; s != nil && s.Ruler != "" {
		ruler = s.Ruler
	}
	lover := Pick(r, glLoverNames)
	n := r.Int(3, 9)

	rp := strings.NewReplacer("{realm}", evRealm, "{place}", place)
	subj := rp.Replace(Pick(r, glSubjects[e.Kind]))
	adj := Pick(r, glBiasAdj[bias])
	vars := []string{"{author}", b.Author, "{realm}", evRealm, "{place}", place, "{foe}", foe, "{deity}", deity,
		"{ruler}", ruler, "{lover}", lover, "{n}", strconv.Itoa(n), "{subj}", subj, "{adj}", adj}
	fill := strings.NewReplacer(vars...)
	b.Title = fill.Replace(Pick(r, f.Titles))

	fact, num, who, dy := glSkew(e.Text, bias, r, own, evRealm, deity)
	b.Year, b.Num, b.Who = e.Y+dy, num, who
	vars = append(vars, "{year}", strconv.Itoa(b.Year), "{fact}", fact)

	// the clue: true, or a lie when the author's bias bends it
	liar := glLiar[bias]
	if !b.Real {
		liar *= 1.25
	}
	if rare {
		liar /= 2
	}
	b.Lie = r.Chance(liar)
	rt := NewRng(fmt.Sprintf("%s:gl:ev:%d", g.Seed, ev)) // the event's own truth, shared by every book about it
	kinds := glFormClues[form]
	kind := Pick(r, kinds)
	clue := ""
	b.SecretP = -1
	switch kind {
	case "site":
		pp, txt, reg, ok := g.glSiteNear(nonNeg(e.P, w.burgP(w.stateCapital(evSt))), rt)
		if !ok {
			kind = "lore"
			break
		}
		clue, b.Val = txt, reg
		if b.Lie {
			p2 := rt.Int(0, len(w.State)-1)
			wrong := w.RegionName(p2)
			if wrong == reg {
				wrong = w.RegionName((p2 + 97) % len(w.State))
			}
			clue = strings.Replace(txt, reg, wrong, 1)
			b.Val = wrong
		} else {
			b.SecretText, b.SecretKind, b.SecretP = txt, "dungeon", pp
		}
	case "crime":
		ci := rt.Int(0, len(glCrimes)-1)
		rl, fo := ruler, foe
		if b.Lie {
			for i := range w.States {
				s := &w.States[(i+ev)%len(w.States)]
				if s.ID != evSt && s.Ruler != "" {
					rl = s.Ruler
					break
				}
			}
		}
		clue = strings.NewReplacer("{ruler}", rl, "{foe}", fo, "{realm}", evRealm).Replace(glCrimes[ci])
		b.Val = rl + "|" + strconv.Itoa(ci)
		if !b.Lie {
			b.SecretText, b.SecretKind = "In \""+b.Title+"\": "+clue, "crime"
		}
	case "identity":
		var dead, alive []*PersonData
		for i := range w.People {
			pd := &w.People[i]
			if pd.Name == "" {
				continue
			}
			if !pd.Alive && pd.Died > 0 && (pd.State == evSt || len(dead) < 3) {
				dead = append(dead, pd)
			} else if pd.Alive && pd.Renown > 0 && (pd.State == evSt || len(alive) < 3) {
				alive = append(alive, pd)
			}
		}
		if len(dead) == 0 || len(alive) == 0 {
			kind = "lore"
			break
		}
		d := dead[rt.Int(0, len(dead)-1)]
		a := alive[rt.Int(0, len(alive)-1)]
		if b.Lie && len(alive) > 1 {
			a = alive[(rt.Int(0, len(alive)-1)+1+r.Int(0, len(alive)-2))%len(alive)]
		}
		where := evRealm
		sp := -1
		if bd := w.burgByID[a.Burg]; bd != nil {
			where, sp = bd.Name, bd.P
		}
		clue = strings.NewReplacer("{a}", a.Name, "{d}", d.Name, "{where}", where, "{dy}", strconv.Itoa(d.Died)).Replace(Pick(rt, glIdentityFrames))
		b.Val = a.Name
		if !b.Lie {
			b.SecretText, b.SecretKind, b.SecretP = "Rumoured in \""+b.Title+"\": "+clue, "identity", sp
		}
	case "technique":
		b.Tech = rt.Int(0, len(glTechs)-1)
		clue = glTechs[b.Tech].Text
		b.Val = "ok"
		if b.Lie {
			b.Val = "flawed"
		} else {
			b.SecretText, b.SecretKind = "Technique from \""+b.Title+"\": "+clue, "technique"
		}
	}
	if kind == "lore" {
		yr := e.Y
		if b.Lie {
			yr += r.Int(1, 4)
		}
		clue = fmt.Sprintf("the matter stands in the chronicles of %s under the year %d.", evRealm, yr)
		b.Val = strconv.Itoa(yr)
		if !b.Lie {
			b.SecretText, b.SecretKind, b.SecretP = fmt.Sprintf("Year %d, %s (per \"%s\")", e.Y, lowerFirst(e.Text), b.Title), "history", e.P
		}
	}
	b.Kind = kind
	vars = append(vars, "{clue}", clue)
	fill = strings.NewReplacer(vars...)

	// proscription
	if bs := evSt; w.stateByID[bs] != nil && (form == "pamphlet" || (form == "satire" && (bias == "cynical" || bias == "partisan")) ||
		(rare && r.Chance(0.5)) || r.Chance(0.06)) {
		pct := 55
		if g.uwLawsFor(bs).Heresy {
			pct = 90
		}
		if g.uwPct(bs, fmt.Sprintf("glban%d", seed), pct) {
			b.Ban = bs + 1
		}
	}

	// text
	cu := w.cultByID[b.Cult]
	var L []string
	L = append(L, "{{Y|"+b.Title+"}}")
	by := "a " + f.Name + " by " + b.Author
	if b.Epithet != "" {
		by += ", " + b.Epithet
	}
	L = append(L, "{{K|"+by+" of "+own+"}}")
	if b.Ban > 0 {
		L = append(L, "{{R|Proscribed in "+w.StateName(b.Ban-1)+".}}")
	}
	L = append(L, "")
	if cu != nil {
		v := "Written in the manner of the " + cu.Name
		if len(cu.Values) > 0 {
			v += ", who prize " + strings.ToLower(Pick(r, cu.Values))
		}
		v += "."
		if l := g.lxLangFor(cu); l != nil {
			for _, k := range []string{"god", "war", "king", "gold"} {
				if wd := l.Words[k]; wd != "" {
					v += fmt.Sprintf(" (In their speech, %s is \"%s\".)", k, wd)
					break
				}
			}
		}
		L = append(L, "{{K|"+v+"}}", "")
	}
	L = append(L, fill.Replace(Pick(r, f.Open)), "", fill.Replace(Pick(r, f.Frame)), "",
		fill.Replace(Pick(r, glBiasVoice[bias])), "", fill.Replace(Pick(r, glClueFrames[kind])), "",
		fill.Replace(Pick(r, f.Close)), "{{K|Copied in the year "+strconv.Itoa(w.Year-r.Int(0, 40))+".}}")
	b.Text = strings.Join(L, "\n")
	b.Minutes = clampi(len(strings.Fields(b.Text))/18, 4, 40)
	return b
}

func nonNeg(a, def int) int {
	if a >= 0 {
		return a
	}
	return def
}

func (w *WorldData) stateCapital(st int) int {
	if s := w.stateByID[st]; s != nil {
		return s.Capital
	}
	return 0
}

func (w *WorldData) burgP(id int) int {
	if b := w.burgByID[id]; b != nil {
		return b.P
	}
	return -1
}

// glItem makes the book item for a seed (nil when the world has nothing to write about).
func (g *Game) glItem(seed, p int, rare bool) *Item {
	b := g.glCompose(seed, p, rare)
	if b == nil {
		return nil
	}
	it := g.MakeBook(b.Title, b.Text)
	it.Lore = b.Form + " by " + b.Author
	it.Value = 20 + len(b.Text)/60
	it.Extra = map[string]int{"gl": 1, "gl:seed": seed, "gl:p": p, "gl:ev": b.Ev + 1, "gl:min": b.Minutes, "lxc": b.Cult + 1}
	if rare {
		it.Extra["gl:rare"] = 1
		it.Value *= 4
		it.Rarity = "rare"
	}
	if b.Ban > 0 {
		it.Extra["gl:ban"] = b.Ban
		it.Extra["uw:forbidden"] = 1
		it.Value += it.Value / 2
	}
	return it
}

// GenBook is the drop-in replacement for RandomBook: about two books in three are generated
// literature about the world's real events, the rest fall back to the codex books.
func (g *Game) GenBook(r *Rng, p int) *Item {
	if g.W == nil || len(g.W.Events) == 0 || r.Chance(0.35) {
		return g.RandomBook(r, p)
	}
	seed := r.Int(1, 1<<30)
	if it := g.glItem(seed, g.cxP(p), false); it != nil {
		return it
	}
	return g.RandomBook(r, p)
}

// ---- reading ----

// glRead opens the reading screen for a generated book (call from useItem case "book").
func (g *Game) glRead(it *Item) bool {
	if it == nil || it.Kind != "book" || it.Extra["gl"] == 0 {
		return false
	}
	st := g.glSt()
	if ban := it.Extra["gl:ban"] - 1; ban >= 0 && g.Cur != nil && g.uwStateHere() == ban {
		for _, a := range g.Cur.Actors {
			if uwIsGuardOf(a, ban) && abs(a.X-g.P.X)+abs(a.Y-g.P.Y) <= 10 {
				if HashStr(fmt.Sprintf("%s:glread:%d:%d", g.Seed, g.Day(), it.Extra["gl:seed"]))%100 < 45 {
					g.removeInv(it)
					g.Msg("{{R|A warden snatches the proscribed book from your hands.}}")
					g.uwAddBounty(ban, 20, " for reading a proscribed book")
					return true
				}
				g.Msg("{{y|You read with one eye on the watch.}}")
				break
			}
		}
	}
	key := strconv.Itoa(it.Extra["gl:seed"])
	mins := 2
	if !st.Done[key] {
		mins = maxi(3, it.Extra["gl:min"]-g.Skills["lore"]*2)
	}
	g.Turn += mins
	st.Minute = mins
	st.View, st.Scroll, st.Shelfv = it, 0, false
	st.Lines = Wrap(g.lxBookText(it), MW-14)
	if g.Skills["lore"] >= 2 {
		if b := g.glCompose(it.Extra["gl:seed"], it.Extra["gl:p"], it.Extra["gl:rare"] > 0); b != nil {
			st.Lines = append(st.Lines, "", "{{K|(Your learning tells you this author leans "+b.Bias+".)}}")
		}
	}
	g.Mode = "glr"
	if len(st.Lines) <= glVis {
		g.glFinish(it)
	}
	return true
}

var glTechSkill = func(i int) string { return glTechs[i%len(glTechs)].Skill }

func (g *Game) glFinish(it *Item) {
	st := g.glSt()
	key := strconv.Itoa(it.Extra["gl:seed"])
	if st.Done[key] {
		return
	}
	st.Done[key] = true
	g.Read["book:"+it.Name] = true
	b := g.glCompose(it.Extra["gl:seed"], it.Extra["gl:p"], it.Extra["gl:rare"] > 0)
	if b == nil {
		return
	}
	g.gainXP(10 + g.Skills["lore"]*5)
	if len(st.Shelf) < 80 {
		st.Shelf = append(st.Shelf, glShelfRow{b.Title, b.Author, b.Form, b.Bias})
	}
	if b.SecretText != "" && g.addSecret(b.SecretText, b.SecretKind, b.SecretP, b.Year) {
		g.Msg("{{W|(Recorded in your journal.)}}")
	}
	if b.Kind == "technique" && !b.Lie {
		sk := glTechSkill(b.Tech)
		if g.Skills[sk] < 1 {
			g.Skills[sk] = 1
			g.Msg("{{G|You try the method until it sticks. " + skillName(sk) + " rises to rank 1.}}")
		} else {
			g.gainXP(25)
			g.Msg("{{G|The method sharpens what you already know. +25 XP}}")
		}
	}
	ek := "e" + strconv.Itoa(b.Ev)
	cl := glClaim{b.Title, b.Author, b.Bias, b.Year, b.Num, b.Who, b.Kind, b.Val}
	for _, o := range st.Claims[ek] {
		g.glCompare(o, cl)
	}
	if len(st.Claims[ek]) < 6 {
		st.Claims[ek] = append(st.Claims[ek], cl)
	}
	g.Msg("{{c|You finish \"" + it.Name + "\".}}")
}

func (g *Game) glCompare(a, b glClaim) {
	if a.Title == b.Title {
		return
	}
	var diff []string
	if a.Year != b.Year {
		diff = append(diff, fmt.Sprintf("\"%s\" dates it to the year %d, \"%s\" to %d.", a.Title, a.Year, b.Title, b.Year))
	}
	if a.Num != "" && b.Num != "" && a.Num != b.Num {
		diff = append(diff, fmt.Sprintf("They give different numbers: %s (%s, %s) against %s (%s, %s).", a.Num, a.Author, a.Bias, b.Num, b.Author, b.Bias))
	}
	if a.Who != b.Who && a.Who != "" && b.Who != "" {
		diff = append(diff, fmt.Sprintf("They credit it to different hands: %s (%s) against %s (%s).", a.Who, a.Bias, b.Who, b.Bias))
	}
	if a.Kind == b.Kind && a.Val != b.Val && a.Kind != "" {
		diff = append(diff, fmt.Sprintf("Their asides on %s disagree: \"%s\" against \"%s\". One of them lied.", a.Kind, strings.Replace(a.Val, "|", " / ", 1), strings.Replace(b.Val, "|", " / ", 1)))
	}
	if len(diff) == 0 {
		g.Msg("{{G|Both books agree on the facts, though one is " + a.Bias + " and the other " + b.Bias + ".}}")
		return
	}
	for _, d := range diff {
		line := "Contradiction between two books on the same event: " + d
		if g.addSecret(line, "contradiction", -1, a.Year) {
			g.Msg("{{Y|Contradiction noted: " + d + "}}")
		}
	}
}

func glClose(g *Game) {
	st := g.glSt()
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

func glKeys(g *Game, k string) {
	st := g.glSt()
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
	case "l":
		st.Shelfv = !st.Shelfv
	case "Escape", "q", "Enter":
		glClose(g)
		return
	}
	st.Scroll = clampi(st.Scroll, 0, max)
	if st.Scroll >= max && st.View != nil {
		g.glFinish(st.View)
	}
}

func glDraw(g *Game) {
	s := g.sc
	st := g.glSt()
	x, y, w, h := g.panel("Reading")
	if st.Shelfv || st.View == nil {
		s.Text(x, y, "{{Y|Your shelf}} {{K|("+strconv.Itoa(len(st.Shelf))+" books read)}}", CGray)
		for i := len(st.Shelf) - 1; i >= 0 && y+2+len(st.Shelf)-1-i < y+h-2; i-- {
			r := st.Shelf[i]
			s.TextB(x, y+2+len(st.Shelf)-1-i, fmt.Sprintf("%s {{K|- %s, %s, %s}}", r.Title, r.Author, r.Form, r.Bias), CGray, CBgDark, x+w)
		}
		s.Text(x, y+h-1, "{{K|l: back to the book   Esc: close}}", CGray)
		return
	}
	s.TextB(x, y, fmt.Sprintf("{{K}}{{K|Reading took about %d minutes. Line %d of %d.}}", st.Minute, mini(st.Scroll+glVis, len(st.Lines)), len(st.Lines)), CGray, CBgDark, x+w)
	for i := 0; i < glVis && st.Scroll+i < len(st.Lines); i++ {
		s.TextB(x, y+2+i, st.Lines[st.Scroll+i], CGray, CBgDark, x+w)
	}
	s.Text(x, y+h-1, "{{K|Up/Down/PgUp/PgDn: scroll   l: shelf   Esc: close}}", CGray)
}

// ---- proscribed books ----

func glEnter(g *Game, zn *Zone) {
	if zn == nil || zn.Kind != "town" {
		return
	}
	bd := g.W.burgByID[zn.Burg]
	if bd == nil {
		return
	}
	state := bd.State
	var held []*Item
	for _, it := range g.P.Inv {
		if it.Extra["gl:ban"] == state+1 {
			held = append(held, it)
		}
	}
	if len(held) == 0 {
		return
	}
	guard := false
	for _, a := range zn.Actors {
		if uwIsGuardOf(a, state) {
			guard = true
			break
		}
	}
	st := g.glSt()
	dk := fmt.Sprintf("%d:%d", state, g.Day())
	if !guard || st.Warned[dk] {
		return
	}
	st.Warned[dk] = true
	if HashStr(fmt.Sprintf("%s:glban:%s", g.Seed, dk))%100 < 30 {
		it := held[0]
		g.removeInv(it)
		g.Msg("{{R|The gate wardens search your pack and burn \"" + it.Name + "\" on the spot.}}")
		g.uwAddBounty(state, 25, " for carrying a proscribed book")
		return
	}
	g.Msg("{{y|\"" + held[0].Name + "\" is proscribed in " + g.W.StateName(state) + ". Keep it out of sight.}}")
}

// ---- booksellers ----

func glTalk(g *Game, a *Actor) []Option {
	if a == nil || a.Role != "bookseller" || g.Cur == nil {
		return nil
	}
	return []Option{{"Ask about rare books.", func() { g.glRare(a) }}}
}

func (g *Game) glRare(a *Actor) {
	w := g.W
	st := g.glSt()
	p := w.Idx(g.Cur.WX, g.Cur.WY)
	slot := g.Day() / 20
	var opts []Option
	lines := []string{"The bookseller glances at the door and lowers their voice. \"Not for the shelf, you understand.\""}
	for i := 0; i < 2; i++ {
		seed := int(HashStr(fmt.Sprintf("%s:glrare:%d:%d:%d", g.Seed, a.ID, slot, i)) & 0x3fffffff)
		key := strconv.Itoa(seed)
		if st.Bought[key] {
			continue
		}
		it := g.glItem(seed, p, true)
		if it == nil {
			continue
		}
		b := g.glCompose(seed, p, true)
		line := fmt.Sprintf("\"%s\": a %s by %s of %s.", it.Name, b.Form, b.Author, w.StateName(b.AuthorState))
		if g.Skills["lore"] >= 2 {
			line += " (" + b.Bias + ")"
		}
		if b.Ban > 0 {
			line += " {{R|Proscribed in " + w.StateName(b.Ban-1) + ".}}"
		}
		price := it.Value
		line += " Asking " + g.cuFormat(price) + "."
		lines = append(lines, line)
		itc, k, pr := it, key, price
		opts = append(opts, Option{"Buy \"" + it.Name + "\".", func() {
			if !g.cuSpend(pr) {
				g.talkMenu("\"Come back with the coin.\"")
				return
			}
			st.Bought[k] = true
			g.give(itc)
			g.talkMenu("They wrap it in plain cloth. \"You never saw me.\"")
		}})
	}
	if len(opts) == 0 {
		g.talkMenu("\"Nothing rare this season. Ask again when the carts have come.\"")
		return
	}
	g.talkMenu(strings.Join(lines, "\n\n"))
	opts = append(opts, Option{"Never mind.", func() { g.talkMenu(g.greeting(a)) }})
	g.Opts = opts
}
