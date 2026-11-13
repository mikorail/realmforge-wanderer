package engine

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"unicode"
)

// The end-of-life saga (prefix sg): a journey track and key markers recorded while you play (state
// g.ext("sg"), global across planets), and sgSaga(), a multi-chapter life story written from the run's data
// in the voice of a chronicler of your culture and genre. Templates and grammar only, deterministic from the
// run (side rng NewRng(seed+":sg:"+key), never g.R). The shell shows it (src/Saga.svelte) via qud.saga().

type sgMark struct {
	K  string `json:"k"`
	D  int    `json:"d"`
	Pl int    `json:"p"`
	Q  int    `json:"q"`
	T  string `json:"t"`
}

type sgPerson struct {
	Key   string `json:"k"`
	Name  string `json:"n"`
	Role  string `json:"r"`
	Home  string `json:"h"`
	Opin  int    `json:"o"`
	Quote string `json:"q"`
	Day   int    `json:"d"`
	Alive bool   `json:"a"`
}

type sgComp struct {
	Name string `json:"n"`
	Day  int    `json:"d"`
	Fell int    `json:"f,omitempty"` // day + 1 when it fell
}

type sgFoe struct {
	Kind   string `json:"k"`
	Name   string `json:"n"`
	Epi    string `json:"e"`
	Why    string `json:"w"`
	Status string `json:"s"`
}

type sgState struct {
	Track  []int       `json:"t,omitempty"` // planet, parasang, day triples
	Marks  []sgMark    `json:"m,omitempty"`
	Seen   int         `json:"s,omitempty"`
	Init   bool        `json:"i,omitempty"`
	Y0     int         `json:"y,omitempty"`
	Home   string      `json:"h,omitempty"`
	Realm  string      `json:"r,omitempty"`
	Ruler  string      `json:"ru,omitempty"`
	Era    string      `json:"e,omitempty"`
	World  string      `json:"w,omitempty"`
	Planet string      `json:"pn,omitempty"`
	Event  string      `json:"ev,omitempty"`
	Sphere string      `json:"sp,omitempty"`
	Folk   []*sgPerson `json:"f,omitempty"`
	Comps  []*sgComp   `json:"c,omitempty"`
	Foes   []*sgFoe    `json:"fo,omitempty"`
	Trial  string      `json:"tr,omitempty"`
	End    string      `json:"en,omitempty"`
	Kills  int         `json:"k,omitempty"`
	LastPl int         `json:"lp,omitempty"`
	LastQ  int         `json:"lq,omitempty"`
}

func (g *Game) sg() *sgState {
	return g.ext("sg", func() any { return &sgState{LastQ: -1} }).(*sgState)
}

func init() {
	turnHooks = append(turnHooks, sgTurn)
	killHooks = append(killHooks, sgKill)
}

func sgPlanets(g *Game) []*WorldData { return unUni(g.W).Planets }

func sgInit(g *Game, st *sgState) {
	st.Init = true
	st.Y0, st.Era, st.World, st.Planet = g.W.Year, g.W.Era, g.W.Name, g.W.Planet
	if b := g.W.burgByID[g.StartBurg]; b != nil {
		st.Home = b.Name
		if s := g.W.stateByID[b.State]; s != nil {
			st.Realm, st.Ruler = nonEmpty(s.FullName, s.Name), s.Ruler
		}
		if cu := g.W.cultByID[b.Culture]; cu != nil {
			st.Sphere = strings.ToLower(cu.Sphere)
		}
	}
	best := -1
	for i := range g.W.Events {
		e := &g.W.Events[i]
		if e.Y <= g.W.Year && (best < 0 || e.Y >= g.W.Events[best].Y) {
			best = i
		}
	}
	if best >= 0 {
		st.Event = sgTrim(Plain(g.W.Events[best].Text), 150)
	}
}

func sgTrim(s string, n int) string {
	s = strings.TrimSpace(s)
	if r := []rune(s); len(r) > n {
		return string(r[:n]) + "..."
	}
	return s
}

func (st *sgState) mark(k string, d, pl, q int, t string) {
	if len(st.Marks) > 0 {
		if l := st.Marks[len(st.Marks)-1]; l.T == t && l.K == k {
			return
		}
	}
	st.Marks = append(st.Marks, sgMark{k, d, pl, q, sgTrim(t, 160)})
	if len(st.Marks) > 200 {
		for _, drop := range []string{"life", "event", "kill"} {
			for i := range st.Marks {
				if st.Marks[i].K == drop {
					st.Marks = append(st.Marks[:i], st.Marks[i+1:]...)
					return
				}
			}
		}
		st.Marks = st.Marks[1:]
	}
}

func sgTurn(g *Game) {
	if g.P == nil || g.W == nil || g.Cur == nil || g.Dead {
		return
	}
	st := g.sg()
	if !st.Init {
		sgInit(g, st)
	}
	day := g.Day()
	pl := g.W.uniIdx
	q := g.W.Idx(g.WX, g.WY)
	if q >= 0 && q < len(g.W.Biome) && (pl != st.LastPl || q != st.LastQ || len(st.Track) == 0) {
		if len(st.Track) > 0 && pl != st.LastPl {
			st.mark("rift", day, pl, q, fmt.Sprintf("%s crossed a rift to %s.", g.P.Name, nonEmpty(g.W.Name, g.W.Planet)))
		}
		st.Track = append(st.Track, pl, q, day)
		st.LastPl, st.LastQ = pl, q
		if len(st.Track) >= 6000 {
			var nt []int
			for i := 0; i+2 < len(st.Track); i += 3 {
				if (i/3)%2 == 0 || i+5 >= len(st.Track) {
					nt = append(nt, st.Track[i:i+3]...)
				}
			}
			st.Track = nt
		}
	}
	if len(g.Deeds) > st.Seen {
		for _, d := range g.Deeds[st.Seen:] {
			if d.Text != "" {
				st.mark(d.Kind, day, pl, maxi(q, 0), Plain(d.Text))
			}
		}
		st.Seen = len(g.Deeds)
	}
	if g.Turn%30 == 0 {
		for _, a := range g.localCompanions() {
			sgNoteComp(st, a.Name, day, false)
		}
	}
	if g.Turn%60 == 0 {
		sgSnap(g, st)
	}
}

func sgNoteComp(st *sgState, name string, day int, fell bool) {
	if name == "" {
		return
	}
	for _, c := range st.Comps {
		if c.Name == name {
			if fell && c.Fell == 0 {
				c.Fell = day + 1
			}
			return
		}
	}
	if len(st.Comps) >= 12 {
		return
	}
	c := &sgComp{Name: name, Day: day}
	if fell {
		c.Fell = day + 1
	}
	st.Comps = append(st.Comps, c)
}

func sgKill(g *Game, d, src *Actor) {
	if g.P == nil || d == nil || g.Cur == nil {
		return
	}
	st := g.sg()
	day := g.Day()
	if d.AI == "follower" || d.Comp != nil {
		sgNoteComp(st, d.Name, day, true)
		return
	}
	if (src == g.P || (src != nil && src.AI == "follower")) && (d.Boss || d.Title != "") && d.SimMon == 0 && st.Kills < 20 {
		st.Kills++
		st.mark("kill", day, g.W.uniIdx, maxi(g.W.Idx(g.WX, g.WY), 0), fmt.Sprintf("%s slew %s.", g.P.Name, d.Name))
	}
}

func sgQuote(g *Game, p *lvPerson) string {
	r := NewRng(g.Seed + ":sg:q:" + p.Key)
	var best *lvMemory
	for i := range p.Mem {
		if best == nil || abs(p.Mem[i].Weight) > abs(best.Weight) {
			best = &p.Mem[i]
		}
	}
	if best != nil && abs(best.Weight) >= 3 {
		if l := sgQuoteByKind[best.Kind]; len(l) > 0 {
			return Pick(r, l)
		}
	}
	for _, e := range sgQuoteByOpinion {
		if p.Opinion < e[0].(int) {
			return e[1].(string)
		}
	}
	return "They are remembered."
}

// sgSnap copies what planet-local systems know (people, nemesis, rivals, the great threat) into global state.
func sgSnap(g *Game, st *sgState) {
	day := g.Day()
	for _, p := range lvSt(g).People {
		if !p.Met || (abs(p.Opinion) < 15 && len(p.Mem) < 2) {
			continue
		}
		var cur *sgPerson
		for _, f := range st.Folk {
			if f.Key == p.Key && f.Home == lvBurgName(g, p.Home) {
				cur = f
			}
		}
		if cur == nil {
			if len(st.Folk) >= 40 {
				continue
			}
			cur = &sgPerson{Key: p.Key, Day: day}
			st.Folk = append(st.Folk, cur)
		}
		cur.Name, cur.Role, cur.Home, cur.Opin, cur.Alive = p.Name, nonEmpty(p.Job, p.Role), lvBurgName(g, p.Home), p.Opinion, p.Alive
		cur.Quote = sgQuote(g, p)
	}
	if n := nxSt(g).Nem; n != nil && n.Name != "" {
		sgFoeSet(st, "nemesis", n.Name, n.Epithet, sgTrim(Plain(n.Reason), 200), n.Status)
	}
	for _, rv := range g.rv().R {
		if rv != nil && rv.Met {
			s := "alive"
			if !rv.Alive {
				s = "died"
				if rv.Killer == g.P.Name {
					s = "slain"
				}
			}
			sgFoeSet(st, "rival", rv.Name, strings.ToLower(rv.Title), "", s)
		}
	}
	if e := g.eg(); e.Phase >= 1 && e.Name != "" {
		st.Trial = e.Name
		if e.Phase == 3 && e.Ending != "" && st.End == "" {
			st.End = e.Ending
			st.mark("end", day, g.W.uniIdx, maxi(g.W.Idx(g.WX, g.WY), 0), fmt.Sprintf("%s was %s by %s.", e.Name, strings.ToLower(egEndNames[e.Ending]), g.P.Name))
		}
	}
}

func sgFoeSet(st *sgState, kind, name, epi, why, status string) {
	for _, f := range st.Foes {
		if f.Name == name {
			f.Status, f.Epi = status, epi
			if why != "" {
				f.Why = why
			}
			return
		}
	}
	if len(st.Foes) < 8 {
		st.Foes = append(st.Foes, &sgFoe{kind, name, epi, why, status})
	}
}

// ---- output ----

type SagaChapter struct {
	Kind  string `json:"kind"`
	Title string `json:"title"`
	Text  string `json:"text"`
	Day0  int    `json:"day0"`
	Day1  int    `json:"day1"`
	T0    int    `json:"t0"` // track point range [t0, t1)
	T1    int    `json:"t1"`
}

type SagaMarker struct {
	K  string `json:"k"`
	D  int    `json:"d"`
	Pl int    `json:"pl"`
	Q  int    `json:"q"`
	T  string `json:"t"`
}

type SagaMap struct {
	Planet int    `json:"planet"`
	Name   string `json:"name"`
	W      int    `json:"w"`
	H      int    `json:"h"`
	Biome  string `json:"biome"`  // one letter per parasang: 'a' + biome id (a = open sea)
	Height string `json:"height"` // 'a' + height/4
	Towns  []int  `json:"towns"`
}

type SagaStats struct {
	Days  int `json:"days"`
	Km    int `json:"km"`
	Kills int `json:"kills"`
	Deeds int `json:"deeds"`
	Level int `json:"level"`
	Score int `json:"score"`
	Years int `json:"years"`
}

type SagaData struct {
	Title      string        `json:"title"`
	Subtitle   string        `json:"subtitle"`
	Chronicler string        `json:"chronicler"`
	Name       string        `json:"name"`
	World      string        `json:"world"`
	Seed       string        `json:"seed"`
	Dead       bool          `json:"dead"`
	Chapters   []SagaChapter `json:"chapters"`
	Epitaph    string        `json:"epitaph"`
	Recap      string        `json:"recap"`
	Stats      SagaStats     `json:"stats"`
	Track      []int         `json:"track"` // planet, parasang, day triples
	Markers    []SagaMarker  `json:"markers"`
	Maps       []SagaMap     `json:"maps"`
}

// Saga returns the saga as JSON (qud.saga()).
func (g *Game) Saga() string {
	if g.P == nil || g.W == nil {
		return "{}"
	}
	b, _ := json.Marshal(g.sgSaga())
	return string(b)
}

func sgPlace(w *WorldData, p int) string {
	if w == nil || p < 0 || p >= len(w.Biome) {
		return "the wilds"
	}
	if b := w.burgByID[w.Burg[p]]; b != nil {
		return b.Name
	}
	x, y := p%w.W, p/w.W
	var near *BurgData
	bd := 5
	for i := range w.Burgs {
		b := &w.Burgs[i]
		if !b.Alive || b.Layer != "surface" {
			continue
		}
		if d := maxi(abs(b.P%w.W-x), abs(b.P/w.W-y)); d < bd {
			bd, near = d, b
		}
	}
	biome := "the " + BiomeName(w.Biome[p])
	if near != nil {
		return biome + " near " + near.Name
	}
	return biome
}

func sgIsTown(w *WorldData, p int) bool {
	return w != nil && p >= 0 && p < len(w.Burg) && w.burgByID[w.Burg[p]] != nil
}

func sgCap(s string) string {
	r := []rune(s)
	for i, c := range r {
		if unicode.IsLetter(c) {
			r[i] = unicode.ToUpper(c)
			break
		}
		if c == '"' || c == ' ' {
			continue
		}
		break
	}
	return string(r)
}

func sgFill(tpl string, v map[string]string) (string, bool) {
	var b strings.Builder
	for i := 0; i < len(tpl); i++ {
		if tpl[i] != '{' {
			b.WriteByte(tpl[i])
			continue
		}
		j := strings.IndexByte(tpl[i:], '}')
		if j < 0 {
			b.WriteString(tpl[i:])
			break
		}
		val := v[tpl[i+1:i+j]]
		if val == "" {
			return "", false
		}
		b.WriteString(val)
		i += j
	}
	return sgCap(b.String()), true
}

func sgShuffle(r *Rng, a []string) []string {
	out := append([]string(nil), a...)
	for i := len(out) - 1; i > 0; i-- {
		j := r.Int(0, i)
		out[i], out[j] = out[j], out[i]
	}
	return out
}

// sgCompose builds a chapter: one opener, up to maxMid fillable middle sentences, one closer.
func sgCompose(r *Rng, p sgPool, v map[string]string, maxMid int) string {
	var out []string
	pick := func(list []string) string {
		for _, t := range sgShuffle(r, list) {
			if s, ok := sgFill(t, v); ok {
				return s
			}
		}
		return ""
	}
	if s := pick(p.Open); s != "" {
		out = append(out, s)
	}
	n := 0
	for _, t := range sgShuffle(r, p.Mid) {
		if n >= maxMid {
			break
		}
		if s, ok := sgFill(t, v); ok {
			out = append(out, s)
			n++
		}
	}
	if s := pick(p.Close); s != "" {
		out = append(out, s)
	}
	return strings.Join(out, " ")
}

var sgFoeKinds = map[string]bool{"nemesis": true, "rival": true, "villain": true, "bounty": true}
var sgTrialKinds = map[string]bool{"endgame": true}

func sgIsLove(m *sgMark) bool {
	switch m.K {
	case "kinship":
		return true
	case "death":
		return !strings.Contains(m.T, "perished")
	case "companion":
		return strings.Contains(m.T, "fell in") || strings.Contains(m.T, "starved")
	}
	return false
}

func sgIsGreat(m *sgMark) bool {
	if sgFoeKinds[m.K] || sgTrialKinds[m.K] || sgIsLove(m) {
		return false
	}
	switch m.K {
	case "life", "event", "rift", "end", "death":
		return false
	}
	return true
}

func sgClause(t, name string) string {
	t = strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(t), "."))
	t = strings.ReplaceAll(t, "The wanderer ", "")
	t = strings.ReplaceAll(t, "the wanderer ", "")
	if strings.HasPrefix(t, "The ") {
		t = "the " + t[4:]
	}
	return t
}

type sgSpec struct {
	kind  string
	start int
}

func (g *Game) sgSaga() *SagaData {
	st := g.sg()
	if !st.Init {
		sgInit(g, st)
	}
	sgSnap(g, st)
	day := g.Day()
	days := g.Turn / DayLen
	name := g.P.Name
	call := strings.ToLower(g.Sk.Calling(g.Calling))
	sp := strings.ToLower(g.P.Species)
	genre := g.W.Genre
	r := NewRng(g.Seed + ":sg:saga")

	// voice
	v, ok := sgGenreVoice[genre]
	if !ok {
		v = sgDefaultVoice
	}
	if sgSphereGenres[genre] {
		if sv, ok := sgSphereVoice[st.Sphere]; ok {
			v.Role, v.Noun = sv.Role, sv.Noun
			v.Open = sv.Open
		}
	}

	// epithet and motif
	epi, motif := "", ""
	if t := fmTitleByID[g.fm().Worn]; t != nil {
		epi = strings.TrimSpace(fmTitleName(t, ""))
	} else if t := g.acTitle(); t != "" {
		epi = t
	}
	var traitIDs []string
	traitIDs = append(traitIDs, thSt(g).Traits...)
	for _, id := range traitIDs {
		if epi == "" && sgTraitEpithet[id] != "" {
			epi = sgTraitEpithet[id]
		}
		if motif == "" && sgTraitMotif[id] != "" {
			motif = sgTraitMotif[id]
		}
	}
	if epi == "" {
		cnt := map[string]int{}
		for _, m := range st.Marks {
			cnt[m.K]++
		}
		kinds := make([]string, 0, len(sgKindEpithet))
		for k := range sgKindEpithet {
			kinds = append(kinds, k)
		}
		sort.Strings(kinds)
		bk, bn := "", 0
		for _, k := range kinds {
			if cnt[k] > bn {
				bk, bn = k, cnt[k]
			}
		}
		switch {
		case bk != "":
			epi = sgKindEpithet[bk]
		case g.fm().Km >= 1000:
			epi = "the Far-Walker"
		default:
			epi = "the Unnamed"
		}
	}
	if !strings.HasPrefix(strings.ToLower(epi), "the ") {
		epi = "the " + epi
	} else {
		epi = "the " + epi[4:]
	}
	if motif == "" {
		motif = "the road itself"
	}

	home := nonEmpty(st.Home, "a nameless village")
	y0 := st.Y0
	y1 := y0 + day/112
	pls := sgPlanets(g)
	planetW := func(i int) *WorldData {
		if i >= 0 && i < len(pls) {
			return pls[i]
		}
		return g.W
	}
	km := int(g.fm().Km)

	base := map[string]string{
		"n": name, "call": call, "sp": sp, "epi": epi, "motif": motif, "home": home, "realm": nonEmpty(st.Realm, "a small realm"),
		"ruler": st.Ruler, "era": st.Era, "world": nonEmpty(st.World, g.W.Name), "planet": st.Planet, "y0": fmt.Sprint(y0), "y1": fmt.Sprint(y1),
		"event": st.Event, "role": v.Role, "kills": fmt.Sprint(g.Kills),
	}
	if km > 0 {
		base["km"] = fmt.Sprint(km)
	}
	if days > 0 {
		base["days"] = fmt.Sprint(days)
	}

	// family, companions, foes, trial
	fm := g.fm()
	if sp := fm.Spouse; sp != nil {
		base["spouse"] = sp.Name
		switch sp.Gone {
		case "dead":
			base["spousefate"] = sp.Name + " died, and the house went quiet."
			base["spousewho"] = sp.Name + " went into the ground before " + name + " did, and was missed to the end."
		case "left":
			base["spousefate"] = sp.Name + " left, and did not say where the road went."
			base["spousewho"] = sp.Name + " had gone long before, and is not spoken of kindly or unkindly."
		case "lost":
			base["spousefate"] = sp.Name + " was lost to the world, and " + name + " went looking."
		default:
			base["spousewho"] = sp.Name + " outlived the road, and keeps the house and the name."
		}
	}
	if len(fm.Kids) > 0 {
		k := fm.Kids[0]
		base["kid"] = k.Name
		var ks []string
		for _, c := range fm.Kids {
			ks = append(ks, c.Name)
		}
		base["kidwho"] = "The children, " + strings.Join(ks, " and ") + ", carry the name into the next age."
		if len(ks) == 1 {
			base["kidwho"] = ks[0] + ", the child, carries the name into the next age."
		}
	}
	var fell, lived []string
	for _, c := range st.Comps {
		if c.Fell > 0 {
			fell = append(fell, c.Name)
		} else {
			lived = append(lived, c.Name)
		}
	}
	if len(st.Comps) > 0 {
		base["comp"] = st.Comps[0].Name
	}
	if len(fell) > 0 {
		base["compfell"] = strings.Join(fell, " and ") + " fell on the road, and " + name + " carried the memory on."
	}
	switch {
	case len(lived) > 0 && len(fell) > 0:
		base["compwho"] = strings.Join(fell, " and ") + " fell on the road; " + strings.Join(lived, " and ") + " lived to tell of it."
	case len(lived) > 0:
		base["compwho"] = strings.Join(lived, " and ") + " walked beside " + name + " and lived to tell of it."
	case len(fell) > 0:
		base["compwho"] = strings.Join(fell, " and ") + " fell on the road beside " + name + ", and is counted in the song."
	}
	var foe *sgFoe
	for _, f := range st.Foes {
		if foe == nil || f.Kind == "nemesis" {
			foe = f
		}
	}
	if foe != nil {
		base["foe"], base["foeepi"] = foe.Name, nonEmpty(foe.Epi, "an old enemy")
		if foe.Why != "" {
			base["foewhy"] = strings.TrimSuffix(foe.Why, ".") + "."
		}
		var fate, who string
		switch foe.Status {
		case "slain":
			fate, who = foe.Name+" fell to the hand of "+name+", at the last.", foe.Name+" fell to "+name+" and was not mourned loudly."
		case "spared", "truce":
			fate, who = name+" and "+foe.Name+" made a peace of sorts, and neither forgot.", foe.Name+" lives still, spared by "+name+", and tells it both ways."
		case "died":
			fate, who = foe.Name+" died to another hand, and "+name+" never had the last word.", foe.Name+" died to another hand before the account was settled."
		default:
			fate, who = foe.Name+" was hunting "+name+" still when the end came.", foe.Name+" was still on the road, with the whole feud unfinished."
		}
		base["foefate"], base["foewho"] = fate, who
	}
	if st.Trial != "" {
		base["trial"] = st.Trial
		if w := sgEndWord[st.End]; w != "" {
			base["endline"] = "At the last, " + name + " " + w + "."
		}
	}
	base["open"], _ = sgFill(Pick(r, v.Open), base)
	base["close"], _ = sgFill(Pick(r, v.Close), base)

	// song and epitaph
	var best *fmSong
	for _, s := range fm.Songs {
		if best == nil || s.Sung > best.Sung {
			best = s
		}
	}
	top := achTopDeeds(g, 1)
	switch {
	case best != nil:
		base["song"] = "The song that is sung of " + name + " goes, in the taverns: \"" + sgClause(Plain(best.Text), name) + ".\""
	case len(top) > 0:
		base["song"] = "What is told of " + name + " in the taverns is this: " + sgClause(Plain(top[0]), name) + "."
	}
	cause := strings.TrimSpace(Plain(g.DeathText))
	cause = strings.TrimPrefix(cause, "You were killed by ")
	cause = strings.TrimSuffix(cause, ".")
	dplace := "the wilds"
	if g.Cur != nil {
		dplace = g.Cur.Name
	}
	base["dplace"] = dplace
	if g.Dead {
		if cause != "" && cause != strings.TrimSpace(Plain(g.DeathText)) {
			base["enddeath"], _ = sgFill("And so we come to the end of it: {n} fell at {dplace}, in the year {y1}, killed by "+cause+".", base)
		} else {
			base["enddeath"], _ = sgFill("And so we come to the end of it: {n} fell at {dplace}, in the year {y1}.", base)
		}
	} else {
		base["enddeath"], _ = sgFill("At last {n} put down the road, in the year {y1}, at {dplace}, and no one made them.", base)
	}
	eline := Pick(r, sgEpitaphDeath)
	if !g.Dead {
		eline = Pick(r, sgEpitaphRetire)
	}
	for _, id := range traitIDs {
		if s := sgEpitaphByTrait[id]; s != "" {
			eline = s
			break
		}
	}
	base["epitaph1"] = eline
	epitaph := fmt.Sprintf("HERE LIES %s, %s\n%s %s of %s\n%d - %d\n\"%s\"", strings.ToUpper(name), epi, titleCase(sp), call, home, y0, y1, eline)

	// marks by index, specs
	marks := st.Marks
	firstDay := func(f func(*sgMark) bool) int {
		for i := range marks {
			if f(&marks[i]) {
				return marks[i].D
			}
		}
		return -1
	}
	endStart := maxi(0, day-maxi(1, day/15))
	specs := []sgSpec{{"wander", 0}}
	if d := firstDay(sgIsGreat); d >= 0 {
		specs = append(specs, sgSpec{"first", d})
	}
	if d := firstDay(func(m *sgMark) bool { return sgFoeKinds[m.K] }); d >= 0 || foe != nil {
		specs = append(specs, sgSpec{"foe", maxi(d, 0)})
	}
	if d := firstDay(sgIsLove); d >= 0 {
		specs = append(specs, sgSpec{"love", d})
	} else if fm.Spouse != nil || len(fm.Kids) > 0 {
		specs = append(specs, sgSpec{"love", day / 2})
	}
	if d := firstDay(func(m *sgMark) bool { return sgTrialKinds[m.K] }); d >= 0 || st.Trial != "" {
		specs = append(specs, sgSpec{"trial", maxi(d, 0)})
	}
	rank := map[string]int{"wander": 0, "first": 1, "foe": 2, "love": 3, "trial": 4, "road": 5}
	sort.SliceStable(specs, func(i, j int) bool {
		if specs[i].start != specs[j].start {
			return specs[i].start < specs[j].start
		}
		return rank[specs[i].kind] < rank[specs[j].kind]
	})
	for i := range specs {
		if i > 0 {
			specs[i].start = maxi(specs[i].start, specs[i-1].start)
		}
		specs[i].start = mini(specs[i].start, endStart)
	}
	specs = append(specs, sgSpec{"end", endStart})
	for len(specs) < 5 || (len(specs) < 6 && day > 60) {
		bi, bg := 0, -1
		for i := 0; i < len(specs); i++ {
			next := day
			if i+1 < len(specs) {
				next = specs[i+1].start
			}
			if g := next - specs[i].start; g > bg && specs[i].kind != "end" {
				bi, bg = i, g
			}
		}
		ns := sgSpec{"road", specs[bi].start + maxi(bg/2, 0)}
		specs = append(specs[:bi+1], append([]sgSpec{ns}, specs[bi+1:]...)...)
	}

	// helpers over the track
	track := st.Track
	npts := len(track) / 3
	ptIdx := func(d int) int {
		for i := 0; i < npts; i++ {
			if track[i*3+2] >= d {
				return i
			}
		}
		return npts
	}
	placesIn := func(d0, d1 int) []string {
		seen := map[string]bool{}
		var towns, wild []string
		for i := 0; i < npts; i++ {
			dd := track[i*3+2]
			if dd < d0 || dd >= d1 {
				continue
			}
			w := planetW(track[i*3])
			n := sgPlace(w, track[i*3+1])
			if seen[n] {
				continue
			}
			seen[n] = true
			if sgIsTown(w, track[i*3+1]) {
				towns = append(towns, n)
			} else {
				wild = append(wild, n)
			}
		}
		all := append(towns, wild...)
		if len(all) == 0 && npts > 0 {
			i := mini(ptIdx(d0), npts-1)
			all = []string{sgPlace(planetW(track[i*3]), track[i*3+1])}
		}
		return all
	}

	used := map[int]bool{}
	take := func(f func(*sgMark) bool, d0, d1, n int) []string {
		var out []string
		for i := range marks {
			if len(out) >= n {
				break
			}
			m := &marks[i]
			if used[i] || m.D < d0 || m.D >= d1 || !f(m) {
				continue
			}
			used[i] = true
			out = append(out, sgClause(m.T, name))
		}
		return out
	}

	var chapters []SagaChapter
	chapters = append(chapters, SagaChapter{Kind: "prologue", Title: "Prologue", Text: sgCompose(NewRng(g.Seed+":sg:prologue"), sgPools["prologue"], base, 3)})
	titleN := map[string]int{}
	folkIdx := 0
	for i, s := range specs {
		d0, d1 := s.start, 1<<30
		if i+1 < len(specs) {
			d1 = specs[i+1].start
		}
		if i == 0 {
			d0 = 0
		}
		vars := map[string]string{}
		for k, val := range base {
			vars[k] = val
		}
		vars["cy"] = fmt.Sprint(y0 + d0/112)
		if cd := mini(d1, day+1) - d0; cd > 0 {
			vars["days"] = fmt.Sprint(cd)
		} else {
			delete(vars, "days")
		}
		pl := placesIn(d0, d1)
		for k, nm := range []string{"pl1", "pl2", "pl3"} {
			if k < len(pl) {
				vars[nm] = pl[k]
			}
		}
		// the chapter's person
		if len(st.Folk) > 0 {
			var f *sgPerson
			for _, c := range st.Folk {
				if c.Day >= d0 && c.Day < d1 {
					f = c
					break
				}
			}
			if f == nil {
				f = st.Folk[folkIdx%len(st.Folk)]
			}
			folkIdx++
			vars["folk"], vars["frole"], vars["fhome"], vars["quote"] = f.Name, strings.ToLower(f.Role), f.Home, f.Quote
		}
		var ds []string
		switch s.kind {
		case "wander":
			ds = take(func(m *sgMark) bool { return sgIsGreat(m) || m.K == "event" }, d0, d1, 1)
		case "first":
			ds = take(sgIsGreat, 0, 1<<30, 2)
		case "foe":
			ds = take(func(m *sgMark) bool { return sgFoeKinds[m.K] }, 0, 1<<30, 2)
		case "love":
			ds = take(sgIsLove, 0, 1<<30, 2)
		case "trial":
			ds = take(func(m *sgMark) bool { return sgTrialKinds[m.K] || m.K == "end" }, 0, 1<<30, 2)
		case "road":
			ds = take(func(m *sgMark) bool { return sgIsGreat(m) || m.K == "event" || m.K == "rift" }, d0, d1, 2)
		case "end":
			ds = take(func(m *sgMark) bool { return m.K != "life" && m.K != "death" }, d0, 1<<30, 1)
		}
		for k, nm := range []string{"deed1", "deed2"} {
			if k < len(ds) {
				vars[nm] = ds[k]
			}
		}
		if s.kind == "foe" && foe == nil {
			vars["foe"] = ""
		}
		text := sgCompose(NewRng(fmt.Sprintf("%s:sg:ch:%d", g.Seed, i)), sgPools[s.kind], vars, 3)
		if text == "" {
			text = sgCompose(NewRng(fmt.Sprintf("%s:sg:chx:%d", g.Seed, i)), sgPools["road"], vars, 3)
		}
		ts := sgTitles[s.kind]
		title := ts[titleN[s.kind]%len(ts)]
		titleN[s.kind]++
		t1 := npts
		if i+1 < len(specs) {
			t1 = ptIdx(specs[i+1].start)
		}
		chapters = append(chapters, SagaChapter{Kind: s.kind, Title: title, Text: text, Day0: d0, Day1: mini(d1, day), T0: ptIdx(d0), T1: t1})
	}
	chapters = append(chapters, SagaChapter{Kind: "epilogue", Title: "Epilogue", Text: sgCompose(NewRng(g.Seed+":sg:epilogue"), sgPools["epilogue"], base, 4), Day0: day, Day1: day, T0: npts, T1: npts})

	// markers and maps
	var mk []SagaMarker
	for _, m := range marks {
		switch m.K {
		case "kill", "discovery", "kinship", "death", "companion", "rift", "end", "endgame", "nemesis", "rival", "slay", "relic":
			mk = append(mk, SagaMarker{m.K, m.D, m.Pl, m.Q, m.T})
		}
	}
	if len(mk) > 120 {
		mk = mk[len(mk)-120:]
	}
	count := map[int]int{}
	for i := 0; i < npts; i++ {
		count[track[i*3]]++
	}
	var order []int
	for p := range count {
		order = append(order, p)
	}
	sort.Slice(order, func(a, b int) bool {
		if count[order[a]] != count[order[b]] {
			return count[order[a]] > count[order[b]]
		}
		return order[a] < order[b]
	})
	if len(order) > 3 {
		order = order[:3]
	}
	var maps []SagaMap
	for _, p := range order {
		w := planetW(p)
		if w == nil || len(w.Biome) != w.W*w.H || len(w.Height) != len(w.Biome) {
			continue
		}
		bb := make([]byte, len(w.Biome))
		hb := make([]byte, len(w.Biome))
		for i := range bb {
			bb[i] = byte('a' + clampi(w.Biome[i], 0, 25))
			hb[i] = byte('a' + clampi(w.Height[i]/4, 0, 25))
		}
		var towns []int
		for i := range w.Burgs {
			if b := &w.Burgs[i]; b.Alive && b.Layer == "surface" && len(towns) < 250 {
				towns = append(towns, b.P)
			}
		}
		maps = append(maps, SagaMap{Planet: p, Name: nonEmpty(w.Name, w.Planet), W: w.W, H: w.H, Biome: string(bb), Height: string(hb), Towns: towns})
	}

	title := fmt.Sprintf("The %s of %s %s", v.Noun, name, epi)
	stats := SagaStats{Days: days, Km: km, Kills: g.Kills, Deeds: len(g.Deeds), Level: g.P.Level, Score: g.hfScore(), Years: day / 112}
	recap := fmt.Sprintf("%s: level %d %s %s, %d days, %d km, %d slain. \"%s\" Your story is told in %s.", title, g.P.Level, sp, call, days, km, g.Kills, eline, base["world"])
	return &SagaData{
		Title: title, Subtitle: fmt.Sprintf("%s, %s %s of %s, years %d to %d", name, sp, call, home, y0, y1),
		Chronicler: fmt.Sprintf("Told by a %s, %s.", v.Role, v.Medium), Name: name, World: base["world"], Seed: g.unHome(), Dead: g.Dead,
		Chapters: chapters, Epitaph: epitaph, Recap: recap, Stats: stats, Track: append([]int{}, track...), Markers: mk, Maps: maps,
	}
}
