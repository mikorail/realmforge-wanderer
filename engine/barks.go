package engine

// Ambient barks and overheard chatter (prefix ba). A turn hook (once a game minute, rate limited, never in
// combat, off when the `barks` pref is false) makes a nearby visible NPC call a line to nobody in particular,
// chosen by setting (market guard tavern temple road night rain crowd smith farm harbor school), state
// (weather, hour, feast, town news, wanted status, drawn weapons, reputation, attire) and the speaker's
// culture voice pack (Voice.Barks, proverbs, idioms) with a fallback bank in barks_data.go. Two bound,
// friendly townsfolk close together also hold a 2-3 line exchange (gossip about real people, realm news,
// complaints, proverb duels, rumours from the knowledge state) that the player hears within 8 tiles in sight.
// Overheard true rumours go into the case notes (knLearn) with a small XP bonus once. Ability `ba_ear` listens in.
// Never draws g.R: every pick is a hash. State g.ext("ba").

import (
	"fmt"
	"regexp"
	"strings"
)

const (
	baGlobalGap = 6   // minutes between two barks or chatters
	baNPCGap    = 120 // minutes between two barks of one NPC
	baChatGap   = 20  // minutes between two overheard exchanges
	baPairGap   = 300 // minutes before the same pair talks again
	baEarMin    = 5   // minutes a Listen in lasts
)

type baState struct {
	Last  map[int]int    `json:"l,omitempty"` // actor id -> turn of its last bark
	Glob  int            `json:"g,omitempty"` // turn+1 of the last line from anyone
	Chat  int            `json:"c,omitempty"` // turn+1 of the last exchange
	Pair  map[string]int `json:"p,omitempty"`
	Heard map[int]bool   `json:"h,omitempty"` // fact ids already rewarded
	Said  map[int]string `json:"s,omitempty"`
	Ear   int            `json:"e,omitempty"` // turn until which Listen in lasts

	news    []wsLine
	newsDay int
}

func init() {
	unLocalKeys = append(unLocalKeys, "ba")
	turnHooks = append(turnHooks, baTick)
}

func baSt(g *Game) *baState {
	s := g.ext("ba", func() any { return &baState{} }).(*baState)
	if s.Last == nil {
		s.Last = map[int]int{}
	}
	if s.Pair == nil {
		s.Pair = map[string]int{}
	}
	if s.Heard == nil {
		s.Heard = map[int]bool{}
	}
	if s.Said == nil {
		s.Said = map[int]string{}
	}
	return s
}

// ---- ability ----

func (g *Game) baAbilities() []Ability {
	if g.Cur == nil || g.P == nil || g.OnWorld || g.Cur.Burg <= 0 {
		return nil
	}
	return []Ability{{ID: "ba_ear", Name: "Listen in", Target: "self"}}
}

// baUse: hook for usePower (id "ba_ear"). Costs a turn.
func (g *Game) baUse() {
	if g.Cur == nil {
		return
	}
	st := baSt(g)
	st.Ear = g.Turn + baEarMin
	g.Msg("{{c|You hold still and tilt your head, listening to the street.}}")
	g.spend(1000)
}

// ---- helpers ----

var baMarkup = regexp.MustCompile(`\{\{[^|}]*\|([^}]*)\}\}`)
var baLeft = regexp.MustCompile(`\{[a-z]+\}`)

func baPlain(s string) string { return baMarkup.ReplaceAllString(s, "$1") }

func baCap(s string) string {
	if r := []rune(s); len(r) > 0 {
		return strings.ToUpper(string(r[0])) + string(r[1:])
	}
	return s
}

func baHash(g *Game, parts ...any) uint64 {
	return HashStr(g.W.Seed + ":ba:" + fmt.Sprint(parts...))
}

func baPickOf(g *Game, list []string, parts ...any) string {
	if len(list) == 0 {
		return ""
	}
	return list[int(baHash(g, parts...)%uint64(len(list)))]
}

func (g *Game) baBurg() *BurgData {
	if g.Cur == nil || g.W == nil {
		return nil
	}
	return g.W.burgByID[g.Cur.Burg]
}

func (g *Game) baCulture(a *Actor, p *lvPerson) *CultureData {
	if p != nil && p.Culture >= 0 {
		if cu := g.W.cultByID[p.Culture]; cu != nil {
			return cu
		}
	}
	if bd := g.baBurg(); bd != nil {
		return g.W.cultByID[bd.Culture]
	}
	return nil
}

func baVoice(cu *CultureData) *VoicePack {
	if cu == nil || cu.Pack == nil {
		return nil
	}
	return cu.Pack.Voice
}

// baVars fills every placeholder a bark may carry.
func (g *Game) baVars(a *Actor, cu *CultureData) map[string]string {
	v := map[string]string{"name": givenOf(a.Name), "title": a.Title, "coin": g.Sk.Currency, "ruler": "the crown", "realm": "the realm",
		"town": "town", "time": strings.ToLower(g.timeOfDay())}
	if bd := g.baBurg(); bd != nil {
		v["town"] = bd.Name
		if st := g.W.stateByID[bd.State]; st != nil && st.ID > 0 {
			v["realm"] = st.Name
			if r := g.poRulerName(bd.State); r != "" {
				v["ruler"] = r
			}
		}
	}
	if cu != nil {
		v["culture"] = cu.Name
	}
	if a.Religion > 0 {
		if rel := g.W.relByID[a.Religion]; rel != nil {
			v["faith"], v["deity"] = rel.Name, rel.Deity
		}
	}
	if _, f := g.feastToday(); f != nil {
		v["feast"] = f.Name
	}
	return v
}

func baFill(s string, v map[string]string) string {
	s = cpFill(s, v)
	pairs := []string{}
	for _, k := range []string{"coin", "ruler", "realm", "town", "feast", "name"} {
		pairs = append(pairs, "{"+k+"}", v[k])
	}
	s = strings.NewReplacer(pairs...).Replace(s)
	s = baLeft.ReplaceAllString(s, "")
	return strings.Join(strings.Fields(s), " ")
}

func baName(a *Actor) string {
	if a.Proper {
		return a.Name
	}
	return "The " + a.Name
}

func (g *Game) baArmed() bool {
	for _, bp := range g.P.Body {
		if bp.Slot == "hand" && bp.Item != nil && (bp.Item.Kind == "weapon" || bp.Item.Kind == "ranged") {
			return true
		}
	}
	return false
}

func (g *Game) baRaining() bool {
	w := g.activeWeather()
	return w != nil && w.Kind != nil && (w.Kind.Overlay == "rain" || w.Kind.Overlay == "storm")
}

// baCandidates: NPCs the player can see and that may talk.
func (g *Game) baCandidates(rad int) []*Actor {
	var out []*Actor
	for _, a := range g.Cur.Actors {
		if a.HP <= 0 || a.Tpl != "npc" || a.AI == "hostile" || a.AI == "follower" || a.Angry || a.Comp != nil {
			continue
		}
		if a.Status["sleep"] > 0 || dist(a, g.P) > rad || !g.visible(a.X, a.Y) {
			continue
		}
		out = append(out, a)
	}
	return out
}

// ---- turn hook ----

func baTick(g *Game) {
	if g.Cur == nil || g.W == nil || g.OnWorld || g.P == nil || g.Dead || g.Mode != "main" || !g.Pref("barks") {
		return
	}
	if g.Cur.Burg <= 0 && g.Cur.Kind != "surface" {
		return
	}
	if len(g.foeNear(10)) > 0 {
		return
	}
	st := baSt(g)
	ear := st.Ear > g.Turn
	gap, chance, chat := baGlobalGap, 0.14, 0.4
	if ear {
		gap, chance, chat = 2, 0.5, 0.7
	}
	if st.Glob > 0 && g.Turn+1-st.Glob < gap {
		return
	}
	r := NewRng(fmt.Sprint(g.W.Seed, ":ba:t:", g.Turn))
	if !r.Chance(chance) {
		return
	}
	if r.Chance(chat) && g.baChatter(st, ear) {
		st.Glob = g.Turn + 1
		return
	}
	if g.baBark(st, ear) {
		st.Glob = g.Turn + 1
	}
}

// ---- barks ----

func (g *Game) baBark(st *baState, ear bool) bool {
	rad := 9
	if ear {
		rad = 14
	}
	var pool []*Actor
	for _, a := range g.baCandidates(rad) {
		if l, ok := st.Last[a.ID]; !ok || g.Turn-l >= baNPCGap {
			pool = append(pool, a)
		}
	}
	if len(pool) == 0 {
		return false
	}
	a := pool[int(baHash(g, "who", g.Turn)%uint64(len(pool)))]
	text := g.baLine(st, a)
	if baHash(g, "rm", g.Turn)%6 == 0 {
		if s := rmLine(g, a); s != "" {
			text = s
		}
	}
	if text == "" {
		return false
	}
	if len(st.Last) > 80 {
		for k, t := range st.Last {
			if g.Turn-t > baNPCGap {
				delete(st.Last, k)
				delete(st.Said, k)
			}
		}
	}
	st.Last[a.ID], st.Said[a.ID] = g.Turn, text
	g.Msg(fmt.Sprintf("{{c|%s}} calls: \"%s\"", baName(a), baCap(text)))
	return true
}

// baSetting: what kind of place and speaker this is.
func (g *Game) baSetting(a *Actor, p *lvPerson) string {
	job := a.Role
	if p != nil && p.Job != "" && p.Job != "retired" {
		job = p.Job
	}
	switch job {
	case "guard", "sergeant":
		return "guard"
	case "innkeeper":
		return "tavern"
	case "priest", "acolyte":
		return "temple"
	case "smith", "smelter":
		return "smith"
	case "farmer", "herder", "miller":
		return "farm"
	case "sailor", "fisher", "ferryman", "ferry":
		return "harbor"
	case "tutor", "scribe", "bookseller", "chronicler":
		return "school"
	case "merchant", "baker", "tailor", "apothecary":
		return "market"
	}
	n := g.Cur.Name
	switch {
	case g.Cur.Burg <= 0:
		return "road"
	case strings.Contains(n, "Temple"):
		return "temple"
	case strings.Contains(n, "Harbor"):
		return "harbor"
	case strings.Contains(n, "Fields"):
		return "farm"
	case strings.Contains(n, "Scholars"):
		return "school"
	case strings.Contains(n, "Market"):
		return "market"
	case strings.Contains(n, "Forge"):
		return "smith"
	}
	return "crowd"
}

// baPool: pack barks for the setting, else the fallback bank plus genre lines.
func (g *Game) baPool(key string, vp *VoicePack) []string {
	if vp != nil && len(vp.Barks[key]) > 0 {
		return vp.Barks[key]
	}
	out := append([]string{}, baBank[key]...)
	out = append(out, baGenre[g.Genre][key]...)
	return out
}

func (g *Game) baLine(st *baState, a *Actor) string {
	p := lvPersonOf(g, a)
	cu := g.baCulture(a, p)
	vp := baVoice(cu)
	vars := g.baVars(a, cu)
	set := g.baSetting(a, p)
	r := NewRng(fmt.Sprint(g.W.Seed, ":ba:l:", a.ID, ":", g.Turn))
	fin := func(list []string, parts ...any) string {
		if len(list) == 0 {
			return ""
		}
		parts = append(parts, a.ID, g.Turn/30)
		t := baPickOf(g, list, parts...)
		if t == st.Said[a.ID] && len(list) > 1 {
			t = baPickOf(g, list, append(parts, "alt")...)
		}
		return baFill(t, vars)
	}
	bd := g.baBurg()
	// guards: wanted status, drawn weapons
	if set == "guard" {
		if g.uwWantedHere() >= 10 && r.Chance(0.6) {
			return fin(baBank["wanted"], "wanted")
		}
		if g.baArmed() && bd != nil && r.Chance(0.4) {
			return fin(baBank["armed"], "armed")
		}
	}
	// a feast day
	if _, f := g.feastToday(); f != nil && g.Cur.Burg > 0 && r.Chance(0.4) {
		return fin(baBank["feast"], "feast")
	}
	// something that happened in town lately
	if g.Cur.Burg > 0 && r.Chance(0.3) {
		if t := g.baNewsBark(a, vars); t != "" {
			return t
		}
	}
	if g.baRaining() && r.Chance(0.35) {
		return fin(g.baPool("rain", vp), "rain")
	}
	if !g.daylight() && r.Chance(0.35) {
		return fin(g.baPool("night", vp), "night")
	}
	// the player's name in the street
	if rep := g.Rep[a.Faction]; a.Faction != "" && r.Chance(0.2) {
		if rep >= 50 {
			return fin(baBank["famous"], "famous")
		} else if rep <= -40 {
			return fin(baBank["infamous"], "infamous")
		}
	}
	if r.Chance(0.1) {
		if t := g.attireRemark(a); t != "" {
			return strings.Trim(baPlain(t), "\"")
		}
	}
	// the speaker's voice: proverbs and idioms, caste
	if vp != nil && r.Chance(0.15) {
		if len(vp.Idioms) > 0 && r.Chance(0.4) {
			if id := vp.Idioms[int(baHash(g, "idiom", a.ID, g.Turn/60)%uint64(len(vp.Idioms)))]; id.Say != "" {
				return baFill(id.Say, vars)
			}
		}
		if t := fin(vp.Proverbs, "prov"); t != "" {
			return t
		}
	} else if vp == nil && r.Chance(0.08) {
		return fin(baProverbs, "prov")
	}
	if a.Caste != "" && bd != nil && r.Chance(0.1) {
		d := g.strataDefFor(bd)
		switch {
		case d.high(a.Caste):
			return fin(baBank["high"], "high")
		case d.outcast(a.Caste) || d.index(a.Caste) >= len(d.list)-2:
			return fin(baBank["low"], "low")
		}
	}
	return fin(g.baPool(set, vp), set)
}

// baNewsBark: a recent happening in this town as street talk.
func (g *Game) baNewsBark(a *Actor, vars map[string]string) string {
	today := lvDay(g)
	var recent []lvNews
	for _, n := range lvSt(g).News {
		if n.Burg == g.Cur.Burg && today-n.Day <= 6 && n.Text != "" {
			recent = append(recent, n)
		}
	}
	if len(recent) == 0 {
		return ""
	}
	n := recent[int(baHash(g, "news", a.ID, today)%uint64(len(recent)))]
	tx := baPlain(n.Text)
	low := strings.ToLower(tx)
	pre := baPickOf(g, baOpeners, "np", a.ID, today)
	switch {
	case strings.Contains(low, "plague") || strings.Contains(low, "fever"):
		pre = baPickOf(g, baBank["plague"], "pl", a.ID)
		return baFill(pre, vars)
	case strings.Contains(low, "died") || strings.Contains(low, "killed") || strings.Contains(low, "dead"):
		return baFill(baPickOf(g, baBank["death"], "de", a.ID), vars) + " " + tx
	case strings.Contains(low, "wed") || strings.Contains(low, "married"):
		return baFill(baPickOf(g, baBank["wed"], "we", a.ID), vars) + " " + tx
	}
	return baFill(pre, vars) + " " + lowerFirst(strings.TrimSuffix(tx, "."))
}

// ---- overheard chatter ----

func (g *Game) baChatter(st *baState, ear bool) bool {
	gap, rad := baChatGap, 8
	if ear {
		gap, rad = 4, 13
	}
	if st.Chat > 0 && g.Turn+1-st.Chat < gap {
		return false
	}
	type pr struct {
		a, b   *Actor
		pa, pb *lvPerson
	}
	var pairs []pr
	for _, a := range g.baCandidates(rad) {
		for _, b := range g.Cur.Actors {
			if b == a || b.ID < a.ID && g.baIn(b, rad) || b.HP <= 0 || b.Tpl != "npc" || b.AI == "hostile" || b.AI == "follower" || b.Angry || b.Comp != nil {
				continue
			}
			if dist(a, b) > 3 || st.Pair[baPairKey(a, b)] > 0 && g.Turn-st.Pair[baPairKey(a, b)] < baPairGap {
				continue
			}
			pa, pb := lvPersonFor(g, a), lvPersonFor(g, b)
			if pa == nil || pb == nil || !pa.Alive || !pb.Alive || lvHas(pa.Rivals, pb.Key) {
				continue
			}
			pairs = append(pairs, pr{a, b, pa, pb})
		}
	}
	if len(pairs) == 0 {
		return false
	}
	pick := pairs[int(baHash(g, "pair", g.Turn)%uint64(len(pairs)))]
	lines := g.baExchange(st, pick.a, pick.b, pick.pa, pick.pb)
	if len(lines) == 0 {
		return false
	}
	st.Chat = g.Turn + 1
	k := baPairKey(pick.a, pick.b)
	st.Pair[k] = g.Turn
	if len(st.Pair) > 60 {
		for kk, t := range st.Pair {
			if g.Turn-t > baPairGap {
				delete(st.Pair, kk)
			}
		}
	}
	g.Msg("{{K|" + fmt.Sprintf(baPickOf(g, baIntro, "intro", g.Turn), baName(pick.a), baName(pick.b)) + "}}")
	for _, l := range lines {
		g.Msg(l)
	}
	return true
}

// baIn: whether an actor is itself a listener-range candidate (dedups mirrored pairs).
func (g *Game) baIn(b *Actor, rad int) bool {
	return dist(b, g.P) <= rad && g.visible(b.X, b.Y)
}

func baPairKey(a, b *Actor) string {
	if a.ID > b.ID {
		a, b = b, a
	}
	return fmt.Sprintf("%d:%d", a.ID, b.ID)
}

func (g *Game) baSay(a *Actor, text string) string {
	return fmt.Sprintf("{{c|%s}}: \"%s\"", baName(a), baCap(text))
}

// baExchange builds 2 or 3 message lines for a pair, or nil.
func (g *Game) baExchange(st *baState, a, b *Actor, pa, pb *lvPerson) []string {
	cuA, cuB := g.baCulture(a, pa), g.baCulture(b, pb)
	vA, vB := baVoice(cuA), baVoice(cuB)
	varsA, varsB := g.baVars(a, cuA), g.baVars(b, cuB)
	opener := baPickOf(g, baOpeners, "op", a.ID, g.Turn/60)
	if vA != nil && len(vA.Gossip) > 0 {
		opener = baPickOf(g, vA.Gossip, "gop", a.ID, g.Turn/60)
	}
	opener = baFill(opener, varsA)
	day := lvDay(g)
	order := int(baHash(g, "topic", a.ID, b.ID, g.Turn/60) % 5)
	for i := 0; i < 5; i++ {
		var body, reply string
		switch (order + i) % 5 {
		case 0:
			body = g.baGossipPerson(pa, pb, a.ID, day)
		case 1:
			body = g.baGossipNews(st, a.ID, day)
		case 2:
			body = baFill(baPickOf(g, baComplaints, "cmp", a.ID, g.Turn/60), varsA)
			reply = baFill(baPickOf(g, baComplaintReplies, "cr", b.ID, g.Turn/60), varsB)
			opener = ""
		case 3:
			p1, p2 := g.baProverb(vA, "a", a.ID), g.baProverb(vB, "b", b.ID)
			if p1 == "" || p2 == "" || p1 == p2 {
				continue
			}
			return []string{g.baSay(a, "As they say, "+lowerFirst(p1)),
				g.baSay(b, baPickOf(g, baDuelCaps, "dc", b.ID, g.Turn/60)+": "+lowerFirst(p2)),
				g.baSay(a, baPickOf(g, baDuelEnds, "de", a.ID, g.Turn/60))}
		case 4:
			if lines := g.baRumour(st, a, b, pa, opener, varsB, vB); lines != nil {
				return lines
			}
			continue
		}
		if body == "" {
			continue
		}
		first := body
		if opener != "" {
			first = opener + " " + lowerFirst(body)
		}
		if reply == "" {
			reply = g.baReply(vB, varsB, b.ID)
		}
		out := []string{g.baSay(a, first), g.baSay(b, reply)}
		if baHash(g, "third", a.ID, g.Turn/60)%2 == 0 {
			follow := baPickOf(g, baFollows, "fo", a.ID, g.Turn/60)
			if vA != nil && len(vA.Fillers) > 0 && baHash(g, "ff", a.ID)%2 == 0 {
				follow = baPickOf(g, vA.Fillers, "fil", a.ID, g.Turn/60)
			}
			out = append(out, g.baSay(a, baFill(follow, varsA)))
		}
		return out
	}
	return nil
}

func (g *Game) baReply(vp *VoicePack, vars map[string]string, id int) string {
	if vp != nil && len(vp.Agree) > 0 && baHash(g, "ag", id, g.Turn/60)%2 == 0 {
		return baFill(baPickOf(g, vp.Agree, "agree", id, g.Turn/60), vars)
	}
	return baFill(baPickOf(g, baReplies, "rep", id, g.Turn/60), vars)
}

func (g *Game) baProverb(vp *VoicePack, tag string, id int) string {
	if vp != nil && len(vp.Proverbs) > 0 {
		return strings.TrimSpace(baPickOf(g, vp.Proverbs, "pv", tag, id, g.Turn/60))
	}
	return baPickOf(g, baProverbs, "pv", tag, id, g.Turn/60)
}

// baGossipPerson: a real fact about someone in town (courtship, feud, life events, shop and birth changes).
func (g *Game) baGossipPerson(pa, pb *lvPerson, id, day int) string {
	burg := g.Cur.Burg
	if burg <= 0 {
		return ""
	}
	ls, year := lsSt(g), lvYear(g)
	people := lvPresent(g, burg)
	var cands []string
	for _, q := range people {
		if q.Key == pa.Key || q.Key == pb.Key {
			continue
		}
		if w := ls.Wooing[q.Key]; w != "" && q.Key < w {
			if o := lvSt(g).People[w]; o != nil && o.Alive {
				cands = append(cands, fmt.Sprintf("%s has been walking out with %s", q.Name, o.Name))
			}
		}
		if len(q.Rivals) > 0 && q.Key < q.Rivals[0] {
			if o := lvSt(g).People[q.Rivals[0]]; o != nil && o.Alive {
				cands = append(cands, fmt.Sprintf("%s and %s are not on speaking terms", q.Name, o.Name))
			}
		}
		if _, bad := ls.Crim[q.Key]; bad {
			cands = append(cands, fmt.Sprintf("%s is in trouble with the law", q.Name))
		}
		for _, ev := range q.Life {
			if ev.Y < year-1 {
				continue
			}
			switch ev.Kind {
			case "marriage", "child", "job", "crime", "feud":
				t := strings.TrimSuffix(ev.Text, ".")
				if strings.HasPrefix(t, "Courtship") {
					cands = append(cands, q.Name+"'s "+lowerFirst(t))
				} else {
					cands = append(cands, q.Name+" "+lowerFirst(t))
				}
			}
		}
	}
	for _, ch := range ls.Chg[burg] {
		if ch.Kind == "shop" || ch.Kind == "birth" || ch.Kind == "come" {
			cands = append(cands, lowerFirst(strings.TrimSuffix(ch.Text, ".")))
		}
	}
	if len(cands) == 0 {
		return ""
	}
	return cands[int(baHash(g, "gp", id, day)%uint64(len(cands)))]
}

// baGossipNews: the realm's news (worldsim, read-only), cached per hour.
func (g *Game) baGossipNews(st *baState, id, day int) string {
	h := g.Turn / 60
	if st.news == nil || st.newsDay != h {
		st.news, st.newsDay = g.wsNewsLines(), h
		if st.news == nil {
			st.news = []wsLine{}
		}
	}
	if len(st.news) == 0 {
		return ""
	}
	t := baPlain(st.news[int(baHash(g, "wn", id, day)%uint64(len(st.news)))].Text)
	return "word from beyond the walls: " + lowerFirst(strings.TrimSuffix(t, "."))
}

// baRumour: speaker a passes on something it knows. True ones reach the case notes (and give XP once).
func (g *Game) baRumour(st *baState, a, b *Actor, pa *lvPerson, opener string, varsB map[string]string, vB *VoicePack) []string {
	var fs []*knFact
	for _, f := range g.knSpeakable(pa, 0, false) {
		if f.Subject != "player" && f.Kind != "location" && f.Text != "" {
			fs = append(fs, f)
		}
	}
	if len(fs) == 0 {
		return nil
	}
	f := fs[int(baHash(g, "rum", a.ID, g.Turn/60)%uint64(mini(len(fs), 6)))]
	said := g.knSpeak(pa, f, false)
	if said == nil {
		return nil
	}
	if opener == "" {
		opener = baOpeners[0]
	}
	lines := []string{g.baSay(a, opener+" "+lowerFirst(strings.TrimSuffix(said.Text, "."))+"."), g.baSay(b, g.baReply(vB, varsB, b.ID))}
	if said.True {
		g.knLearn(said, pa, 1, "overheard")
		if !st.Heard[said.ID] {
			st.Heard[said.ID] = true
			xp := 4 + g.P.Level
			g.gainXP(xp)
			lines = append(lines, fmt.Sprintf("{{G|You commit what you overheard to your case notes. (+%d xp)}}", xp))
		}
	}
	return lines
}
