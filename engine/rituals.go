package engine

import (
	"fmt"
	"regexp"
	"strings"
)

// Rituals, rites of passage and faith lore in play (prefix rt).
//
//  1. Festival rites: on a feast day "Join the festival rite" opens a step-by-step scene built from the
//     culture's FeastRitual (or a generic one without a pack): perform, offer, skip or break each step.
//     Dress, taboo, an offering and a custom (contest) decide the result; see rtFestEnd.
//  2. Rites of passage: rtRite / rtRiteLines / rtDrawLines supply ceremony step texts; rtGuestOptions lets the
//     player take part as a guest with proper etiquette.
//  3. Faith in play: priests lead prayers, liturgy at holy hours, saints' legends, scripture excerpts,
//     heresies, confession; clergy greetings; holy days that show in the side panel and close or raise shops.
//
// State in g.ext("rt") (planet-local). Never draws g.R; text picks go through w.cpChoose.

type rtState struct {
	Fest   map[string]int    `json:"f,omitempty"`  // feast key -> 1 kept, 2 broke
	Pend   map[string]string `json:"pe,omitempty"` // feast key -> custom still to play
	Pray   map[int]int       `json:"p,omitempty"`  // faith -> day of the last prayer blessing
	Lit    map[string]bool   `json:"l,omitempty"`  // faith:day:hour -> attended
	Read   map[int][]int     `json:"r,omitempty"`  // faith -> excerpt indices read
	Visit  map[int]int       `json:"v,omitempty"`  // actor id -> day a scripture excerpt was read
	Title  map[int]bool      `json:"t,omitempty"`  // faith -> scripture title earned
	Conf   map[int]int       `json:"c,omitempty"`  // actor id -> day of the last confession
	Guest  map[string]bool   `json:"g,omitempty"`  // ceremony id + act
	Legend map[int]int       `json:"lg,omitempty"`
	Heresy map[int]int       `json:"h,omitempty"`
	Seen   map[string]bool   `json:"s,omitempty"` // holy-day announcements
}

func init() {
	unLocalKeys = append(unLocalKeys, "rt")
	skAddStatus(&skStatusDef{ID: "rtblessed", Name: "Festival blessing", Glyph: '+', WIL: 2, EGO: 2, Hit: 1})
	skAddStatus(&skStatusDef{ID: "rtprayer", Name: "Prayer's grace", Glyph: '+', WIL: 1, DV: 1})
	talkHooks = append(talkHooks, rtTalk)
	sideHooks = append(sideHooks, rtSide)
	enterHooks = append(enterHooks, rtEnter)
}

func rtSt(g *Game) *rtState {
	s := g.ext("rt", func() any { return &rtState{} }).(*rtState)
	if s.Fest == nil {
		s.Fest = map[string]int{}
	}
	if s.Pend == nil {
		s.Pend = map[string]string{}
	}
	if s.Pray == nil {
		s.Pray = map[int]int{}
	}
	if s.Lit == nil {
		s.Lit = map[string]bool{}
	}
	if s.Read == nil {
		s.Read = map[int][]int{}
	}
	if s.Visit == nil {
		s.Visit = map[int]int{}
	}
	if s.Title == nil {
		s.Title = map[int]bool{}
	}
	if s.Conf == nil {
		s.Conf = map[int]int{}
	}
	if s.Guest == nil {
		s.Guest = map[string]bool{}
	}
	if s.Legend == nil {
		s.Legend = map[int]int{}
	}
	if s.Heresy == nil {
		s.Heresy = map[int]int{}
	}
	if s.Seen == nil {
		s.Seen = map[string]bool{}
	}
	return s
}

// ---------------------------------------------------------------- helpers

var rtBraceRe = regexp.MustCompile(`\s*\{[a-z]+\}`)

// rtClean fills placeholders and strips any left over, so no raw {placeholder} reaches the screen.
func rtClean(s string, vars map[string]string) string {
	s = cpFill(s, vars)
	s = rtBraceRe.ReplaceAllString(s, "")
	return strings.TrimSpace(s)
}

var rtStop = map[string]bool{"with": true, "and": true, "the": true, "from": true, "that": true, "this": true, "bowl": true, "small": true, "some": true, "long": true, "dark": true, "worn": true, "wear": true, "offering": true, "your": true}

// rtWords: the lower-case content words (4+ letters) of a phrase.
func rtWords(s string) []string {
	var out []string
	for _, w := range strings.FieldsFunc(strings.ToLower(s), func(r rune) bool { return r < 'a' || r > 'z' }) {
		if len(w) >= 4 && !rtStop[w] {
			out = append(out, w)
		}
	}
	return out
}

func rtHas(s string, words []string) bool {
	s = strings.ToLower(s)
	for _, w := range words {
		if strings.Contains(s, w) {
			return true
		}
	}
	return false
}

// rtCultureOfBurg: the culture id of a town, else the land you stand in, else the home realm's.
func (g *Game) rtCultureHere() int {
	if g == nil || g.W == nil {
		return -1
	}
	if b := g.localBurg(); b != nil {
		return b.Culture
	}
	if cu := g.localCulture(); cu != nil {
		return cu.ID
	}
	if s := g.W.stateByID[g.HomeState]; s != nil {
		return s.Culture
	}
	return -1
}

func (w *WorldData) rtCult(id int) *CultureData {
	if w == nil {
		return nil
	}
	return w.cultByID[id]
}

// rtVars are the placeholder values for pack text in a town.
func (g *Game) rtVars(b *BurgData, rel *ReligionData) map[string]string {
	v := map[string]string{"name": g.P.Name, "title": "traveller", "time": rtHourWord(g.Hour()), "kin": "the household", "item": "an offering"}
	if b != nil {
		v["town"] = b.Name
		if st := g.W.stateByID[b.State]; st != nil && st.ID > 0 {
			v["realm"] = st.Name
		}
		if cu := g.W.cultByID[b.Culture]; cu != nil {
			v["culture"] = cu.Name
		}
		if rel == nil {
			if st := g.W.stateByID[b.State]; st != nil {
				rel = g.W.relByID[st.Religion]
			}
		}
	}
	if rel != nil {
		v["faith"], v["deity"] = rel.Name, rel.Deity
	}
	return v
}

func rtHourWord(h int) string {
	switch {
	case h < 5:
		return "the dead of night"
	case h < 9:
		return "morning"
	case h < 13:
		return "midday"
	case h < 18:
		return "afternoon"
	case h < 21:
		return "evening"
	}
	return "night"
}

// rtScene opens a choice screen in talk mode with the actor.
func (g *Game) rtScene(a *Actor, text string, opts ...Option) {
	g.Talk, g.Mode = a, "talk"
	g.Popup = Wrap(text, 100)
	g.Opts = opts
	g.Sel = 0
}

func (g *Game) rtBack(a *Actor) Option {
	return Option{"Step back.", func() { g.talkMenu(g.greeting(a)) }}
}

func (g *Game) rtWielding() bool {
	for _, p := range g.P.Body {
		if p.Item != nil && !p.Severed && p.Slot == "hand" && (p.Item.Kind == "weapon" || p.Item.Kind == "ranged") {
			return true
		}
	}
	return false
}

// rtDressFits: any worn piece shares a content word with the dress, or is regional to the town's sphere.
func (g *Game) rtDressFits(dress string, b *BurgData) bool {
	ws := rtWords(dress)
	for _, p := range g.P.Body {
		if p.Item == nil || p.Severed {
			continue
		}
		if len(ws) > 0 && rtHas(p.Item.ShownName(), ws) {
			return true
		}
	}
	if b != nil {
		if cu := g.W.cultByID[b.Culture]; cu != nil && cu.Sphere != "" {
			for _, it := range g.wornAttire() {
				if it.Region == cu.Sphere {
					return true
				}
			}
		}
	}
	return false
}

// rtTabooHit: what you carry or wear that a feast taboo forbids ("" = nothing detectable).
func (g *Game) rtTabooHit(taboo string) string {
	t := strings.ToLower(taboo)
	if t == "" {
		return ""
	}
	for _, tw := range rtTabooWords {
		if !rtHas(t, tw.Words) {
			continue
		}
		switch tw.What {
		case "weapon":
			if g.rtWielding() {
				return "you carry a weapon"
			}
		case "colour":
			if !rtHas(t, []string{"wear", "cloth", "dress", "garment", "colour", "color", "robe", "worn"}) {
				continue
			}
			for _, p := range g.P.Body {
				if p.Item != nil && !p.Severed {
					for _, c := range tw.Words {
						if strings.Contains(t, c) && strings.Contains(strings.ToLower(p.Item.ShownName()), c) {
							return "you wear " + c
						}
					}
				}
			}
		}
	}
	return ""
}

// rtTabooItem: the item would break the feast taboo if offered.
func rtTabooItem(taboo string, it *Item) bool {
	t := strings.ToLower(taboo)
	n := strings.ToLower(it.Name)
	for _, tw := range rtTabooWords {
		if !rtHas(t, tw.Words) {
			continue
		}
		switch tw.What {
		case "meat":
			if it.Kind == "corpse" || strings.Contains(n, "meat") || strings.Contains(n, "jerky") {
				return true
			}
		case "drink":
			if rtHas(n, []string{"wine", "ale", "beer", "mead", "spirit", "liquor"}) {
				return true
			}
		}
	}
	return false
}

func rtTake(g *Game, it *Item) {
	if it.Count > 1 {
		it.Count--
		return
	}
	g.removeInv(it)
}

// rtOfferItems: up to n suitable items from the pack for the offerings (keyword match by name), else any food.
func (g *Game) rtOfferItems(offerings []string, n int) []*Item {
	var ws []string
	for _, o := range offerings {
		ws = append(ws, rtWords(o)...)
	}
	var out []*Item
	for _, it := range g.P.Inv {
		if len(out) >= n {
			break
		}
		if uwIsStolen(it) || it.Value < 0 {
			continue
		}
		switch it.Kind {
		case "food", "dish", "ingredient", "tonic", "water", "trophy", "scrap", "book":
		default:
			continue
		}
		if rtHas(it.Name, ws) {
			out = append(out, it)
		}
	}
	return out
}

// ---------------------------------------------------------------- 1. festival rites

func rtFestKey(b *BurgData, f *FeastData, year int) string {
	return fmt.Sprintf("%d:%s:%d", b.ID, f.Name, year)
}

// rtFestRitual finds the pack ritual of a feast by name (case-insensitive); nil without a pack.
func (g *Game) rtFestRitual(b *BurgData, f *FeastData) *FeastRitual {
	pk := g.W.cpOf(b.Culture)
	if pk == nil || pk.Customs == nil {
		return nil
	}
	for i := range pk.Customs.Festivals {
		if strings.EqualFold(pk.Customs.Festivals[i].Feast, f.Name) {
			return &pk.Customs.Festivals[i]
		}
	}
	return nil
}

// rtAvailable: today is a feast in this town, the actor is friendly, and the rite has not been played this year.
func (g *Game) rtAvailable(a *Actor) bool {
	b, f := g.feastToday()
	if f == nil || a == nil || a.AI == "follower" || g.Hostile(a) {
		return false
	}
	return rtSt(g).Fest[rtFestKey(b, f, g.W.DateOf(nil, g.Day()).Year)] == 0
}

// rtOwnsFeast reports whether the richer rite replaces the old "Join the feast" option for this actor.
func (g *Game) rtOwnsFeast(a *Actor) bool { return g.rtAvailable(a) }

type rtFS struct {
	b      *BurgData
	f      *FeastData
	fr     *FeastRitual
	steps  []string
	vars   map[string]string
	key    string
	i      int
	done   int // performed or offered
	offers int
	dress  bool
	broke  bool
}

func (g *Game) rtBeginFeast(a *Actor) {
	b, f := g.feastToday()
	if f == nil {
		return
	}
	year := g.W.DateOf(nil, g.Day()).Year
	s := &rtFS{b: b, f: f, key: rtFestKey(b, f, year), fr: g.rtFestRitual(b, f)}
	s.vars = g.rtVars(b, nil)
	s.vars["time"] = f.Name
	if s.fr != nil && len(s.fr.Steps) > 0 {
		s.steps = s.fr.Steps
	} else {
		s.steps = rtFallbackFestSteps
	}
	dress := ""
	if s.fr != nil {
		dress = s.fr.Dress
	}
	s.dress = dress != "" && g.rtDressFits(dress, b)
	intro := fmt.Sprintf("{{W|%s}}\n%s", f.Name, rtClean(g.W.feastText(f, b), s.vars))
	if dress != "" {
		if s.dress {
			intro += fmt.Sprintf("\n\n{{G|You are dressed as the day asks (%s).}}", rtClean(dress, s.vars))
		} else {
			intro += fmt.Sprintf("\n\n{{y|The day asks for %s. You are not dressed for it, but you may still take part.}}", rtClean(dress, s.vars))
		}
	}
	if s.fr != nil && s.fr.Taboo != "" {
		intro += "\n\n{{R|Forbidden today:}} " + rtClean(s.fr.Taboo, s.vars)
		if why := g.rtTabooHit(s.fr.Taboo); why != "" {
			intro += fmt.Sprintf("\n{{R|A hush runs through the crowd: %s.}}", why)
			g.rtScene(a, intro,
				Option{"{{R|Go on regardless.}}", func() { g.rtFestBreak(a, s, "you carried on against the day's taboo") }},
				Option{"Withdraw and return when you are ready.", func() { g.talkMenu(g.greeting(a)) }})
			return
		}
	}
	g.rtScene(a, intro, Option{"Step into the rite.", func() { g.rtFestStep(a, s) }}, g.rtBack(a))
}

func (g *Game) rtFestStep(a *Actor, s *rtFS) {
	if s.i >= len(s.steps) {
		g.rtFestEnd(a, s)
		return
	}
	text := fmt.Sprintf("{{W|%s}} {{K|(step %d of %d)}}\n%s", s.f.Name, s.i+1, len(s.steps), rtClean(s.steps[s.i], s.vars))
	if s.i == 0 && s.fr != nil && s.fr.Song != "" {
		text += "\n\n{{K|A song rises: " + rtClean(s.fr.Song, s.vars) + "}}"
	}
	next := func() { s.i++; g.rtFestStep(a, s) }
	var opts []Option
	opts = append(opts, Option{"Perform it.", func() {
		g.spendTurns(8)
		s.done++
		next()
	}})
	var off []string
	if s.fr != nil {
		off = s.fr.Offerings
	}
	for _, it := range g.rtOfferItems(off, 2) {
		it := it
		opts = append(opts, Option{"Offer your " + it.Name + ".", func() {
			if s.fr != nil && s.fr.Taboo != "" && rtTabooItem(s.fr.Taboo, it) {
				g.rtFestBreak(a, s, "you offered what the day forbids")
				return
			}
			rtTake(g, it)
			g.spendTurns(5)
			s.done++
			s.offers++
			g.Msg("You offer your " + it.Name + ".")
			next()
		}})
	}
	if g.P.Gold >= 10 {
		opts = append(opts, Option{fmt.Sprintf("Offer 10 %s.", g.Sk.Currency), func() {
			g.P.Gold -= 10
			s.done++
			s.offers++
			g.Msg(fmt.Sprintf("You set 10 %s in the offering bowl.", g.Sk.Currency))
			next()
		}})
	}
	opts = append(opts,
		Option{"Skip this step.", next},
		Option{"{{R|Break the custom.}}", func() { g.rtFestBreak(a, s, "you mocked the rite") }})
	g.rtScene(a, text, opts...)
}

func (g *Game) rtFestBreak(a *Actor, s *rtFS, why string) {
	rs := rtSt(g)
	rs.Fest[s.key] = 2
	if g.FeastSeen == nil {
		g.FeastSeen = map[string]bool{}
	}
	g.FeastSeen[feastKey(s.b, s.f, g.W.DateOf(nil, g.Day()).Year)] = true // barred from the feast this year
	rebuke := "The crowd falls silent, then turns away."
	if s.fr != nil && s.fr.Taboo != "" {
		rebuke = "{{R|The rite is broken.}} " + rtClean(s.fr.Taboo, s.vars)
	}
	g.P.AddSt("shamed", 60)
	if s.b.State > 0 {
		g.AddRep(fmt.Sprintf("state:%d", s.b.State), -10, " for breaking the "+s.f.Name+" custom")
	}
	g.crKarma(4, -2, "broke a festival rite")
	g.talkMenu(fmt.Sprintf("%s Because %s, you are not welcome at the table this year.", rebuke, why))
}

// rtFestEnd: score the rite and give the results.
func (g *Game) rtFestEnd(a *Actor, s *rtFS) {
	n := len(s.steps)
	if s.done == 0 {
		g.talkMenu("You watch the whole rite from the edge of the crowd. No one minds, but no one invites you to the table either.")
		return
	}
	rs := rtSt(g)
	rs.Fest[s.key] = 1
	full := s.done >= n
	half := s.done*2 >= n
	var lines []string
	lines = append(lines, fmt.Sprintf("{{G|The %s rite is done.}}", s.f.Name))
	rep := 3
	dur := 360
	if half {
		rep, dur = 6, 720
	}
	if full {
		rep, dur = 9, 1080
	}
	if s.dress {
		rep, dur = rep+3, dur+360
		lines = append(lines, "Your dress was right for the day, and people nod at it.")
	}
	if s.offers > 0 {
		rep += s.offers
		dur += 120 * s.offers
	}
	g.P.AddSt("rtblessed", dur)
	lines = append(lines, "{{G|You feel blessed: steadier and warmer.}}")
	if s.b.State > 0 {
		g.AddRep(fmt.Sprintf("state:%d", s.b.State), rep, " for keeping the "+s.f.Name+" rite")
	}
	g.crKarma(4, 1, "kept a festival rite")
	g.gainXP(10 + 4*s.offers)
	if s.fr != nil && s.fr.Song != "" {
		lines = append(lines, "The singers close with: "+rtClean(s.fr.Song, s.vars))
	}
	dishName, dishTag := g.rtFestDish(s)
	text := strings.Join(lines, " ")
	custom := ""
	if s.fr != nil {
		custom = s.fr.Custom
	}
	if custom != "" {
		rs.Pend[s.key] = custom
	}
	year := g.W.DateOf(nil, g.Day()).Year
	if !g.FeastSeen[feastKey(s.b, s.f, year)] {
		g.joinFeast(a) // the shared meal, the dish to carry, the journal entry (calendar.go)
		g.Popup = append(Wrap(text, 100), append([]string{""}, g.Popup...)...)
	} else {
		g.talkMenu(text)
	}
	if dishName != "" {
		g.applyMeal(dishName, tagsOf(dishTag), DayLen/2, true)
		g.Msg("You share the {{W|" + dishName + "}} with the townsfolk.")
		g.Popup = append(g.Popup, "", "{{K|They press the "+dishName+" on you: "+dishTag+".}}")
	}
	if custom != "" {
		g.Popup = append(g.Popup, "", "{{y|The "+rtClean(custom, s.vars)+" is about to begin.}}")
	}
}

// rtFestDish: the pack dish served at the feast (Dishes with a matching Feast, else the named Food), with its tag.
func (g *Game) rtFestDish(s *rtFS) (string, string) {
	_, tag, _, _, _ := feastStyleOf(s.f.Kind)
	pk := g.W.cpOf(s.b.Culture)
	if pk != nil && pk.Material != nil && pk.Material.Cuisine != nil {
		for _, d := range pk.Material.Cuisine.Dishes {
			if d.Feast != "" && strings.EqualFold(d.Feast, s.f.Name) && d.Tag != "" {
				return d.Name, d.Tag
			}
		}
		if s.fr != nil && s.fr.Food != "" {
			for _, d := range pk.Material.Cuisine.Dishes {
				if strings.EqualFold(d.Name, s.fr.Food) && d.Tag != "" {
					return d.Name, d.Tag
				}
			}
		}
	}
	if s.fr != nil && s.fr.Food != "" {
		return s.fr.Food, tag
	}
	return "", ""
}

// rtCustomOption: the festival's custom or contest, a small skill check game with a prize.
func (g *Game) rtCustomOption(a *Actor) []Option {
	b, f := g.feastToday()
	if f == nil {
		return nil
	}
	key := rtFestKey(b, f, g.W.DateOf(nil, g.Day()).Year)
	rs := rtSt(g)
	custom := rs.Pend[key]
	if custom == "" {
		return nil
	}
	vars := g.rtVars(b, nil)
	label := rtClean(custom, vars)
	kind := "endurance"
	for _, ck := range rtCustomKind {
		if rtHas(custom, ck.Words) {
			kind = ck.Kind
			break
		}
	}
	return []Option{{g.checkLabel(kind, dcMedium) + " Take part in the " + label + ".", func() {
		delete(rs.Pend, key)
		g.spendTurns(15)
		if g.roll2d6Check(g.checkMod(kind), dcMedium) {
			prize := 12 + 4*g.Day()%5
			g.P.Gold += prize
			g.gainXP(12)
			if b.State > 0 {
				g.AddRep(fmt.Sprintf("state:%d", b.State), 4, " for winning the "+label)
			}
			g.talkMenu(fmt.Sprintf("{{G|[%s: Success]}} You take the %s. The prize is %d %s and a loud cheer.", checkKinds[kind].label, label, prize, g.Sk.Currency))
			return
		}
		g.gainXP(4)
		g.talkMenu(fmt.Sprintf("{{R|[%s: Failure]}} You lose the %s, but the crowd likes you for trying. A neighbour hands you a cup.", checkKinds[kind].label, label))
	}}}
}

// ---------------------------------------------------------------- 2. rites of passage

// rtRite: the pack's rite for a culture, or a generic fallback; nil only for an unknown id.
func (g *Game) rtRite(culture int, id string) *Rite {
	id = strings.ToLower(strings.TrimSpace(id))
	if al, ok := rtRiteAliases[id]; ok {
		id = al
	}
	if g != nil && g.W != nil {
		if pk := g.W.cpOf(culture); pk != nil && pk.Customs != nil {
			for i := range pk.Customs.Rites {
				r := &pk.Customs.Rites[i]
				if r.ID == id || (id == "naming" && r.ID == "birth" && !rtHasRite(pk, "naming")) {
					if len(r.Steps) > 0 {
						return r
					}
				}
			}
		}
	}
	fb := rtFallbackRites[id]
	if fb == nil {
		return nil
	}
	r := *fb
	if cu := g.W.rtCult(culture); cu != nil && cu.Dress != "" && (id == "wedding" || id == "comingofage" || id == "betrothal") {
		r.Dress = cu.Dress
	}
	return &r
}

func rtHasRite(pk *CulturePack, id string) bool {
	for _, r := range pk.Customs.Rites {
		if r.ID == id {
			return true
		}
	}
	return false
}

// rtRiteLines: the ordered, filled lines of a rite (header first); nil if the rite is unknown.
func rtRiteLines(g *Game, culture int, id string, vars map[string]string) []string {
	r := g.rtRite(culture, id)
	if r == nil {
		return nil
	}
	out := []string{}
	if r.Name != "" {
		out = append(out, rtClean(r.Name, vars)+".")
	}
	for _, s := range r.Steps {
		if l := rtClean(s, vars); l != "" {
			out = append(out, l)
		}
	}
	return out
}

// rtKindID maps a ceremony kind (fwCer.Kind) to a rite id.
func rtKindID(kind string) string {
	switch kind {
	case "birth":
		return "naming"
	case "age":
		return "comingofage"
	case "wedding", "funeral":
		return kind
	}
	return ""
}

// rtDrawLines is the hook for fwDrawLines: the pack's steps for the ceremony, or nil to keep the old text.
// Only cultures with a pack rite are used (so bundled worlds are unchanged).
func rtDrawLines(g *Game, c *fwCer) []string {
	if g == nil || g.W == nil || c == nil {
		return nil
	}
	id := rtKindID(c.Kind)
	b := g.W.burgByID[c.Burg]
	if id == "" || b == nil {
		return nil
	}
	pk := g.W.cpOf(b.Culture)
	if pk == nil || pk.Customs == nil || len(pk.Customs.Rites) == 0 {
		return nil
	}
	n := fwNames(g, c)
	vars := g.rtVars(b, g.W.relByID[fwFaith(g, b)])
	if len(n) > 0 {
		vars["name"] = n[0]
	}
	if len(n) > 1 {
		vars["kin"] = n[1]
	}
	r := g.rtRite(b.Culture, id)
	if r == nil {
		return nil
	}
	found := false
	for _, pr := range pk.Customs.Rites {
		if pr.ID == r.ID {
			found = true
		}
	}
	if !found {
		return nil
	}
	return rtRiteLines(g, b.Culture, id, vars)
}

// rtGuestOptions adds guest etiquette choices to a ceremony screen: the proper gift, the right dress, the
// custom's greeting. The ceremony's fwOpt type is reused.
func rtGuestOptions(g *Game, c *fwCer) []fwOpt {
	if g == nil || g.W == nil || c == nil || c.Kind == "feast" {
		return nil
	}
	b := g.W.burgByID[c.Burg]
	id := rtKindID(c.Kind)
	if b == nil || id == "" {
		return nil
	}
	r := g.rtRite(b.Culture, id)
	if r == nil {
		return nil
	}
	rs := rtSt(g)
	hon := fwHonorees(g, c)
	vars := g.rtVars(b, nil)
	var out []fwOpt
	// Dress: take your place properly dressed.
	dkey := c.ID + ":dress"
	if r.Dress != "" {
		o := fwOpt{label: "Take your place among the guests, dressed as the custom asks.", ok: !rs.Guest[dkey]}
		if !o.ok {
			o.why = "done"
		}
		o.do = func() {
			rs.Guest[dkey] = true
			if g.rtDressFits(r.Dress, b) {
				fwBoost(g, hon, "gift", "You came dressed as the custom asks.", 4)
				g.gainXP(5)
				fwRes(g, "{{G|You are dressed rightly ("+rtClean(r.Dress, vars)+").}} Kin nod to you as one who knows how it is done.")
				return
			}
			fwBoost(g, hon, "insult", "You came to their rite in the wrong dress.", -2)
			fwRes(g, "The custom asks for "+rtClean(r.Dress, vars)+". Your clothes draw glances, though no one says a word.")
		}
		out = append(out, o)
	}
	// Gift from Rite.Gifts.
	gkey := c.ID + ":gift"
	if len(r.Gifts) > 0 && !rs.Guest[gkey] {
		gift := r.Gifts[g.W.cpIdx(len(r.Gifts), "riteg", c.ID)]
		o := fwOpt{label: "Bring the proper gift (" + rtClean(gift, vars) + ").", ok: true}
		o.do = func() {
			rs.Guest[gkey] = true
			ws := rtWords(gift)
			var found *Item
			for _, it := range g.P.Inv {
				if !uwIsStolen(it) && it.Kind != "weapon" && it.Kind != "armor" && rtHas(it.Name, ws) {
					found = it
					break
				}
			}
			w := 5
			switch {
			case found != nil:
				rtTake(g, found)
				w = 9
				g.gainXP(6)
				fwBoost(g, hon, "gift", "You brought the proper gift, "+found.Name+".", w)
				fwRes(g, "You hand over your "+found.Name+", exactly what the custom asks. It is received with both hands.")
			case g.P.Gold >= 12:
				g.P.Gold -= 12
				fwBoost(g, hon, "gift", "You brought a gift in coin.", w)
				fwRes(g, fmt.Sprintf("You have nothing of the right sort, so you give 12 %s toward it. They accept it politely.", g.Sk.Currency))
			default:
				fwBoost(g, hon, "insult", "You came to their rite with empty hands.", -2)
				fwRes(g, "You have nothing fit to give. The kin are kind about it.")
			}
		}
		out = append(out, o)
	}
	// Etiquette: one gesture of the culture.
	ekey := c.ID + ":manner"
	if pk := g.W.cpOf(b.Culture); pk != nil && pk.Customs != nil && len(pk.Customs.Etiquette) > 0 && !rs.Guest[ekey] {
		et := pk.Customs.Etiquette[g.W.cpIdx(len(pk.Customs.Etiquette), "riteet", c.ID)]
		o := fwOpt{label: "Greet them as the custom asks.", ok: true}
		o.do = func() {
			rs.Guest[ekey] = true
			fwBoost(g, hon, "gift", "You observed their manners at the rite.", 3+et.Reward)
			g.gainXP(4)
			fwRes(g, "You follow the old manner: "+rtClean(et.Text, vars))
		}
		out = append(out, o)
	}
	return out
}

// ---------------------------------------------------------------- 3. faith in play

func rtIsClergy(a *Actor) bool {
	if a == nil {
		return false
	}
	switch a.Role {
	case "priest", "acolyte", "monk", "shaman", "cleric":
		return true
	}
	return false
}

// rtFaithOf: the religion a clergy actor serves (their own, else the town's).
func (g *Game) rtFaithOf(a *Actor) *ReligionData {
	if g == nil || g.W == nil || a == nil {
		return nil
	}
	if a.Religion > 0 {
		if r := g.W.relByID[a.Religion]; r != nil {
			return r
		}
	}
	if b := g.localBurg(); b != nil {
		if st := g.W.stateByID[b.State]; st != nil {
			return g.W.relByID[st.Religion]
		}
	}
	return nil
}

func (g *Game) rtFaithful(rel *ReligionData) bool {
	if rel == nil {
		return false
	}
	return rel.ID == g.HomeFaith || g.Rep[fmt.Sprintf("faith:%d", rel.ID)] >= 150
}

func rtLore(rel *ReligionData) *FaithPack {
	if rel == nil {
		return nil
	}
	return rel.Lore
}

// rtClergyGreet is the clergy greeting for the dvSay/greeting hook ("" for non-clergy).
func (g *Game) rtClergyGreet(a *Actor) string {
	if !rtIsClergy(a) {
		return ""
	}
	rel := g.rtFaithOf(a)
	if rel == nil {
		return ""
	}
	vars := g.rtVars(g.localBurg(), rel)
	vars["title"] = "friend"
	faithful := g.rtFaithful(rel)
	day := fmt.Sprint(g.Day())
	if lore := rtLore(rel); lore != nil && lore.Clergy != nil {
		cv := lore.Clergy
		if cv.Title != "" {
			vars["title"] = cv.Title
		}
		s := cv.GreetStranger
		if faithful {
			s = cv.GreetFaithful
		}
		if s != "" {
			return "\"" + rtClean(s, vars) + "\""
		}
	}
	list := rtGreetStranger
	if faithful {
		list = rtGreetFaithful
	}
	return "\"" + rtClean(g.W.cpChoose(list, "cler", fmt.Sprint(a.ID), day), vars) + "\""
}

// rtHolyHour: the name of the daily liturgy hour now, or "".
func (g *Game) rtHolyHour() (string, int) {
	h := g.Hour()
	for _, hh := range rtHolyHours {
		if h >= hh.From && h < hh.To {
			return hh.Name, hh.From
		}
	}
	return "", 0
}

// rtHolyDay: today's holy day of a faith, with the culture whose calendar counts it.
func (g *Game) rtHolyDay(rel *ReligionData, cu *CultureData) *HolyDay {
	lore := rtLore(rel)
	if lore == nil {
		return nil
	}
	for i := range lore.HolyDays {
		hd := &lore.HolyDays[i]
		if g.W.FeastDoY(FeastData{Name: hd.Name, Month: hd.Month}, cu) == g.Day()%YearDays {
			return hd
		}
	}
	return nil
}

// rtTownHoly: the holy day (and faith) kept today in the town you stand in (its realm's faith).
func (g *Game) rtTownHoly() (*ReligionData, *HolyDay) {
	b := g.feastTown()
	if b == nil {
		return nil, nil
	}
	cu := g.W.cultByID[b.Culture]
	st := g.W.stateByID[b.State]
	if st == nil {
		return nil, nil
	}
	rel := g.W.relByID[st.Religion]
	if hd := g.rtHolyDay(rel, cu); hd != nil {
		return rel, hd
	}
	return nil, nil
}

func rtSide(g *Game) []string {
	if g == nil || g.W == nil || g.OnWorld || g.P == nil {
		return nil
	}
	var out []string
	if _, hd := g.rtTownHoly(); hd != nil {
		out = append(out, "{{Y|Holy day: "+hd.Name+"}}")
	}
	rs := rtSt(g)
	for fid := range rs.Title {
		if r := g.W.relByID[fid]; r != nil && rtScriptTitle(r) != "" {
			out = append(out, "{{c|"+rtScriptTitle(r)+"}}")
			break
		}
	}
	return out
}

func rtScriptTitle(r *ReligionData) string {
	if r == nil || r.Lore == nil || r.Lore.Scripture == nil {
		return ""
	}
	return "Keeper of the " + r.Lore.Scripture.Title
}

func rtEnter(g *Game, zn *Zone) {
	if g == nil || g.W == nil {
		return
	}
	rel, hd := g.rtTownHoly()
	if hd == nil {
		return
	}
	b := g.feastTown()
	key := fmt.Sprintf("%d:%d", b.ID, g.Day())
	rs := rtSt(g)
	if rs.Seen[key] {
		return
	}
	rs.Seen[key] = true
	if len(rs.Seen) > 40 {
		rs.Seen = map[string]bool{key: true}
	}
	g.Msg(fmt.Sprintf("{{Y|It is %s, a holy day of %s. Shops of the faith keep it.}}", hd.Name, rel.Name))
}

// rtShopClosed (openTrade hook, like snMarketClosed): the shop's faith keeps a holy day today; half the shops close.
func (g *Game) rtShopClosed(a *Actor) bool {
	if g == nil || a == nil || !snShop(a) || a.Religion <= 0 || g.W == nil {
		return false
	}
	rel := g.W.relByID[a.Religion]
	if rel == nil || rtLore(rel) == nil {
		return false
	}
	var cu *CultureData
	if b := g.localBurg(); b != nil {
		cu = g.W.cultByID[b.Culture]
	}
	hd := g.rtHolyDay(rel, cu)
	if hd == nil || g.W.cpIdx(2, "holyshop", fmt.Sprint(a.ID), fmt.Sprint(g.Day())) != 0 {
		return false
	}
	g.Talk, g.Mode = a, "talk"
	g.talkMenu(fmt.Sprintf("%s has shuttered the stall for %s. \"No trade on a holy day. Come back tomorrow.\"", g.The(a), hd.Name))
	return true
}

// rtPriceMod (price() hook): shops that stay open on their faith's holy day charge more.
func (g *Game) rtPriceMod(a *Actor) float64 {
	if g == nil || a == nil || !snShop(a) || a.Religion <= 0 || g.W == nil {
		return 1
	}
	rel := g.W.relByID[a.Religion]
	if rtLore(rel) == nil {
		return 1
	}
	var cu *CultureData
	if b := g.localBurg(); b != nil {
		cu = g.W.cultByID[b.Culture]
	}
	if g.rtHolyDay(rel, cu) != nil {
		return 1.12
	}
	return 1
}

// rtCalendarLines: holy days of the local faith for the journal's Calendar tab.
func (g *Game) rtCalendarLines() []string {
	b := g.feastTown()
	if b == nil || g.W == nil {
		return nil
	}
	st := g.W.stateByID[b.State]
	if st == nil {
		return nil
	}
	rel := g.W.relByID[st.Religion]
	lore := rtLore(rel)
	if lore == nil || len(lore.HolyDays) == 0 {
		return nil
	}
	cu := g.W.cultByID[b.Culture]
	lines := []string{"", "{{C|Holy days of " + rel.Name + "}}"}
	for _, hd := range lore.HolyDays {
		doy := g.W.FeastDoY(FeastData{Name: hd.Name, Month: hd.Month}, cu)
		d := g.W.DateOf(cu, doy)
		l := fmt.Sprintf("  {{W|%s}} {{K|%s of %s}}", hd.Name, ordinal(d.Day), d.Name)
		if hd.Rite != "" {
			l += " " + hd.Rite
		}
		lines = append(lines, Wrap(l, SW-6)...)
	}
	return lines
}

// rtDateSuffix: " Holy day: X" for the date line, or "".
func (g *Game) rtDateSuffix() string {
	if _, hd := g.rtTownHoly(); hd != nil {
		return " {{Y|" + hd.Name + "}}"
	}
	return ""
}

func (g *Game) rtFallbackLiturgy(rel *ReligionData) *Liturgy {
	l := &Liturgy{Name: "The common office"}
	if rel != nil {
		l.Name = "The office of " + rel.Name
	}
	l.Steps = []string{"The congregation stands as the keeper lights the lamps and calls {deity} by name."}
	if rel != nil && rel.Theology != "" {
		l.Steps = append(l.Steps, "The keeper speaks the teaching of "+rel.Theology+", and the people answer in one voice.")
	}
	if rel != nil && rel.Doctrine != "" {
		l.Steps = append(l.Steps, "The words of "+rel.Doctrine+" are recited, line and response.")
	}
	l.Steps = append(l.Steps, "A blessing is spoken, and the lamps are left burning.")
	return l
}

// rtTalk: all dialogue options of this file.
func rtTalk(g *Game, a *Actor) []Option {
	if g == nil || g.W == nil || a == nil || a.AI == "follower" || g.P == nil {
		return nil
	}
	var o []Option
	if g.rtAvailable(a) {
		o = append(o, Option{"{{W|Join the festival rite.}}", func() { g.rtBeginFeast(a) }})
	}
	o = append(o, g.rtCustomOption(a)...)
	if !rtIsClergy(a) || g.Hostile(a) {
		return o
	}
	rel := g.rtFaithOf(a)
	if rel == nil {
		return o
	}
	lore := rtLore(rel)
	rs := rtSt(g)
	vars := g.rtVars(g.localBurg(), rel)
	day := g.Day()
	o = append(o, Option{"Ask the priest to lead a prayer.", func() { g.rtPrayer(a, rel, lore, vars) }})
	if hn, hfrom := g.rtHolyHour(); hn != "" {
		key := fmt.Sprintf("%d:%d:%d", rel.ID, day, hfrom)
		if !rs.Lit[key] {
			o = append(o, Option{"{{Y|Join the liturgy.}} {{K|(" + hn + ")}}", func() { g.rtLiturgy(a, rel, lore, vars, key, 0) }})
		}
	}
	if lore != nil && len(lore.Saints) > 0 {
		o = append(o, Option{"Ask to hear a saint's legend.", func() { g.rtSaint(a, rel, lore, vars) }})
	}
	o = append(o, Option{"Ask about the scripture.", func() { g.rtScripture(a, rel, lore, vars) }})
	if lore != nil && len(lore.Heresies) > 0 {
		o = append(o, Option{"Ask about heresies.", func() { g.rtHeresy(a, rel, lore, vars) }})
	}
	o = append(o, Option{"Confess your sins.", func() { g.rtConfess(a, rel, lore, vars) }})
	return o
}

func (g *Game) rtPrayer(a *Actor, rel *ReligionData, lore *FaithPack, vars map[string]string) {
	rs := rtSt(g)
	list := rtFallbackPrayers
	if lore != nil && len(lore.Prayers) > 0 {
		list = lore.Prayers
	}
	p := rtClean(g.W.cpChoose(list, "pray", fmt.Sprint(a.ID), fmt.Sprint(g.Day()), fmt.Sprint(rs.Pray[rel.ID])), vars)
	g.spendTurns(5)
	text := fmt.Sprintf("%s lays a hand on your shoulder and leads the prayer.\n\n{{W|\"%s\"}}", g.The(a), p)
	if rs.Pray[rel.ID] == g.Day()+1 {
		text += "\n\nThe words comfort you, but you already carry today's grace."
	} else {
		rs.Pray[rel.ID] = g.Day() + 1
		dur := 600
		if g.rtFaithful(rel) {
			dur = 900
		}
		if rs.Title[rel.ID] {
			dur *= 2
		}
		g.P.AddSt("rtprayer", dur)
		g.AddRep(fmt.Sprintf("faith:%d", rel.ID), 2, " for praying with the priest")
		g.crKarma(4, 1, "prayed with a priest")
		text += "\n\n{{G|A small blessing settles on you.}}"
	}
	g.talkMenu(text)
}

func (g *Game) rtLiturgy(a *Actor, rel *ReligionData, lore *FaithPack, vars map[string]string, key string, i int) {
	lit := g.rtFallbackLiturgy(rel)
	if lore != nil && lore.Liturgy != nil && len(lore.Liturgy.Steps) > 0 {
		lit = lore.Liturgy
	}
	if i >= len(lit.Steps) {
		rs := rtSt(g)
		rs.Lit[key] = true
		if len(rs.Lit) > 60 {
			rs.Lit = map[string]bool{key: true}
		}
		g.spendTurns(20)
		n := 6
		if g.rtFaithful(rel) {
			n = 10
		}
		g.AddRep(fmt.Sprintf("faith:%d", rel.ID), n, " for attending the liturgy")
		g.crKarma(4, 2, "attended the liturgy")
		g.gainXP(10)
		g.talkMenu("{{G|The liturgy ends.}} You leave the temple lighter than you came in.")
		return
	}
	step := rtClean(lit.Steps[i], vars)
	head := fmt.Sprintf("{{Y|%s}} {{K|(%d of %d)}}", rtClean(lit.Name, vars), i+1, len(lit.Steps))
	g.rtScene(a, head+"\n"+step,
		Option{"Take part and go on.", func() { g.rtLiturgy(a, rel, lore, vars, key, i+1) }},
		Option{"Slip out quietly.", func() { g.talkMenu("You leave before the liturgy ends. Heads turn, and turn back.") }})
}

func (g *Game) rtSaint(a *Actor, rel *ReligionData, lore *FaithPack, vars map[string]string) {
	rs := rtSt(g)
	n := rs.Legend[rel.ID]
	rs.Legend[rel.ID] = n + 1
	s := lore.Saints[(n+g.W.cpIdx(len(lore.Saints), "saint", fmt.Sprint(rel.ID)))%len(lore.Saints)]
	text := fmt.Sprintf("{{W|%s}}\n%s", s.Name, rtClean(s.Legend, vars))
	if s.Feast != "" {
		when := ""
		for _, hd := range lore.HolyDays {
			if strings.EqualFold(hd.Name, s.Feast) {
				var cu *CultureData
				if b := g.localBurg(); b != nil {
					cu = g.W.cultByID[b.Culture]
				}
				d := g.W.DateOf(cu, g.W.FeastDoY(FeastData{Name: hd.Name, Month: hd.Month}, cu))
				when = fmt.Sprintf(", on the %s of %s", ordinal(d.Day), d.Name)
			}
		}
		text += fmt.Sprintf("\n\n{{K|Their feast: %s%s.}}", rtClean(s.Feast, vars), when)
	}
	g.gainXP(3)
	g.talkMenu(text)
}

func (g *Game) rtScripture(a *Actor, rel *ReligionData, lore *FaithPack, vars map[string]string) {
	rs := rtSt(g)
	if lore == nil || lore.Scripture == nil || len(lore.Scripture.Excerpts) == 0 {
		t := "The priest speaks of the teaching of the faith"
		if rel.Theology != "" {
			t += ": " + rel.Theology
		}
		if rel.Doctrine != "" {
			t += ", and its doctrine, " + rel.Doctrine
		}
		g.talkMenu(t + ".")
		return
	}
	sc := lore.Scripture
	if rs.Visit[a.ID] == g.Day()+1 {
		g.talkMenu("\"One passage is enough to carry for a day. Come back after you have lived with it.\"")
		return
	}
	rs.Visit[a.ID] = g.Day() + 1
	read := map[int]bool{}
	for _, i := range rs.Read[rel.ID] {
		read[i] = true
	}
	idx := -1
	for i := range sc.Excerpts {
		if !read[i] {
			idx = i
			break
		}
	}
	text := ""
	if idx < 0 {
		idx = g.W.cpIdx(len(sc.Excerpts), "scrip", fmt.Sprint(g.Day()))
		text = fmt.Sprintf("{{W|%s}}\nThe priest recites a passage you know already:\n\"%s\"", sc.Title, rtClean(sc.Excerpts[idx], vars))
	} else {
		rs.Read[rel.ID] = append(rs.Read[rel.ID], idx)
		g.gainXP(5)
		text = fmt.Sprintf("{{W|%s}}\n\"%s\"\n{{K|(%d of %d passages heard.)}}", sc.Title, rtClean(sc.Excerpts[idx], vars), len(rs.Read[rel.ID]), len(sc.Excerpts))
		if len(rs.Read[rel.ID]) >= len(sc.Excerpts) && !rs.Title[rel.ID] {
			rs.Title[rel.ID] = true
			g.deed("event", fmt.Sprintf("%s learned every passage of the %s.", g.P.Name, sc.Title), 0, 0)
			g.AddRep(fmt.Sprintf("faith:%d", rel.ID), 10, " for knowing the scripture")
			text += fmt.Sprintf("\n\n{{G|You have heard every passage. The priest names you Keeper of the %s; your prayers will carry twice as far.}}", sc.Title)
		}
	}
	g.talkMenu(text)
}

func (g *Game) rtHeresy(a *Actor, rel *ReligionData, lore *FaithPack, vars map[string]string) {
	rs := rtSt(g)
	n := rs.Heresy[rel.ID]
	rs.Heresy[rel.ID] = n + 1
	h := lore.Heresies[(n+g.W.cpIdx(len(lore.Heresies), "heresy", fmt.Sprint(rel.ID)))%len(lore.Heresies)]
	text := fmt.Sprintf("{{R|%s}}\n%s\n\n\"The faith rejects it, and says why.\"", h.Name, rtClean(h.Claim, vars))
	// a heretic standing nearby argues back
	if g.Cur != nil {
		for _, o := range g.Cur.Actors {
			if o == nil || o == a || o.HP <= 0 || rtIsClergy(o) || o.AI == "follower" || o.Religion <= 0 || o.Religion == rel.ID {
				continue
			}
			if dist(o, a) > 10 || g.W.cpIdx(100, "argue", fmt.Sprint(o.ID), fmt.Sprint(g.Day())) >= 45 {
				continue
			}
			text += fmt.Sprintf("\n\n%s, listening nearby, calls out: \"You say that? %s is not the whole of it. %s\"", g.The(o), rtClean(rtFirstSentence(h.Claim), vars), "We keep our own faith, and it asks no apology.")
			break
		}
	}
	g.talkMenu(text)
}

func rtFirstSentence(s string) string {
	if i := strings.IndexAny(s, ".!?"); i > 0 {
		return s[:i+1]
	}
	return s
}

// rtConfess: confess a sin of the faith; a karma penalty is eased (karma.go crTrait read, crKarma to move).
func (g *Game) rtConfess(a *Actor, rel *ReligionData, lore *FaithPack, vars map[string]string) {
	rs := rtSt(g)
	if d := rs.Conf[a.ID]; d > 0 && g.Day() < d+7 {
		g.talkMenu("\"You have been shriven this week. Live a little before you come to confess again.\"")
		return
	}
	sins := rtFallbackSins
	if lore != nil && len(lore.Sins) > 0 {
		sins = lore.Sins
	}
	sin := rtClean(g.W.cpChoose(sins, "sin", fmt.Sprint(a.ID), fmt.Sprint(g.Day())), vars)
	worst, wv := -1, 0
	for ax := 0; ax < 5; ax++ {
		if v := g.crTrait(ax); v < wv {
			worst, wv = ax, v
		}
	}
	g.spendTurns(10)
	if worst < 0 {
		g.talkMenu(fmt.Sprintf("You speak of %s. The priest listens, then says there is little on your account to forgive. \"Go lightly.\"", sin))
		return
	}
	rs.Conf[a.ID] = g.Day() + 1
	ease := clampi(-wv/3, 4, 15)
	g.crKarma(worst, ease, "confession")
	g.AddRep(fmt.Sprintf("faith:%d", rel.ID), 2, " for confessing")
	g.talkMenu(fmt.Sprintf("You confess, and the priest names the weight: {{W|%s}}. \"It is spoken, so it can be set down.\"\n\n{{G|Your ledger eases (+%d).}}", sin, ease))
}
