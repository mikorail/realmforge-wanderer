package engine

import (
	"fmt"
	"sort"
	"strings"
)

// Companion banter (prefix cb): occasional context-aware lines from recruited companions. Triggers:
// entering their home town, a town of their own realm, a town of a realm at war with theirs, a
// temple of their faith, wonders and ruins (once each), notable weather, feast days, shared meals
// and night camps, level-ups, low-loyalty grumbles, pairs of companions talking (kin, rival realms)
// and one-liners on kills and near-death. Not spammy: one line per companion every cbGap minutes,
// one line from anyone every cbGlobal minutes, nothing but one-liners while foes are near. Lines
// live in cbLines (cbdata.go), keyed by trigger and filtered by tags (genre group, caste, species,
// kin); the pick is a hash of seed + trigger + companion + hour, never g.R. State in g.ext("cb").

const (
	cbGap     = 240 // minutes between two lines of one companion
	cbGlobal  = 50  // minutes between any two lines
	cbQuip    = 40  // minutes between one-liners (kills, near death, level-ups) of one companion
	cbPairGap = 720 // minutes between two companion exchanges
)

type cbState struct {
	Last   map[int]int     `json:"l,omitempty"`  // companion id → turn of their last line
	Quip   map[int]int     `json:"q,omitempty"`  // companion id → turn of their last one-liner
	Global int             `json:"g,omitempty"`  // turn+1 of the last line from anyone
	Seen   map[string]bool `json:"s,omitempty"`  // "id:place" seen once
	Lvl    map[int]int     `json:"lv,omitempty"` // last level seen
	Meal   map[int]int     `json:"m,omitempty"`  // last Companion.MealAt seen
	Feast  map[int]int     `json:"f,omitempty"`  // day of the last feast line
	Wx     string          `json:"wx,omitempty"` // last weather commented on
	Low    map[int]bool    `json:"lo,omitempty"` // near-death quip given (until healed)
	Said   map[int]string  `json:"sd,omitempty"` // last line of each companion (no immediate repeats)
	Pair   int             `json:"p,omitempty"`  // turn of the last companion exchange
	PairN  int             `json:"pn,omitempty"` // exchanges so far (cycles the pair lines)
}

func (g *Game) cb() *cbState {
	st := g.ext("cb", func() any { return &cbState{} }).(*cbState)
	if st.Last == nil {
		st.Last, st.Quip, st.Seen, st.Lvl, st.Meal, st.Feast, st.Low = map[int]int{}, map[int]int{}, map[string]bool{}, map[int]int{}, map[int]int{}, map[int]int{}, map[int]bool{}
	}
	if st.Said == nil {
		st.Said = map[int]string{}
	}
	return st
}

// cbLine is one banter line; Tag "" fits anyone, otherwise it must be among the speaker's tags.
type cbLine struct {
	Tag  string
	Text string
}

// cbTalkers: recruited (not summoned) companions in this zone.
func (g *Game) cbTalkers() []*Actor {
	var out []*Actor
	for _, a := range g.localCompanions() {
		if a.Comp != nil && !a.Comp.Summoned {
			out = append(out, a)
		}
	}
	return out
}

func (g *Game) cbCombat() bool { return g.Cur != nil && len(g.foeNear(10)) > 0 }

// cbCan: the companion may speak a regular line now.
func (g *Game) cbCan(a *Actor) bool {
	st := g.cb()
	if g.cbCombat() || st.Global > 0 && g.Turn+1-st.Global < cbGlobal {
		return false
	}
	l, ok := st.Last[a.ID]
	return !ok || g.Turn-l >= cbGap
}

func (g *Game) cbCanQuip(a *Actor) bool {
	st := g.cb()
	l, ok := st.Quip[a.ID]
	return !ok || g.Turn-l >= cbQuip
}

// cbTags: what a companion is, for line filters.
func (g *Game) cbTags(a *Actor) map[string]bool {
	t := map[string]bool{}
	switch g.Sk.ID {
	case "steampunk", "dieselpunk", "atompunk", "cyberpunk", "biopunk", "solarpunk", "wasteland":
		t["tech"] = true
	default:
		t["old"] = true
	}
	t["g:"+g.Sk.ID] = true
	if a.Species != "" {
		t["sp:"+a.Species] = true
	}
	c := a.Comp
	if c != nil && len(c.Kin) > 0 {
		t["kin"] = true
	}
	if a.Caste != "" && c != nil {
		if bd := g.cbHomeBurg(c); bd != nil {
			d := g.strataDefFor(bd)
			switch {
			case d.high(a.Caste):
				t["high"] = true
			case d.outcast(a.Caste) || d.index(a.Caste) >= len(d.list)-2:
				t["low"] = true
			}
		}
	}
	if a.Religion > 0 {
		t["faith"] = true
	}
	return t
}

func (g *Game) cbHomeBurg(c *Companion) *BurgData {
	x, y, ok := cqHome(c)
	if !ok || !g.W.In(x, y) {
		return nil
	}
	return g.W.burgByID[g.W.Burg[g.W.Idx(x, y)]]
}

// cbFill replaces placeholders with real names from the world.
func (g *Game) cbFill(a *Actor, s string, vars map[string]string) string {
	c := a.Comp
	kin := "family"
	if c != nil && len(c.Kin) > 0 {
		kin = c.Kin[int(HashStr(fmt.Sprintf("%d:%d", a.ID, g.Day()))%uint64(len(c.Kin)))]
	}
	realm, faith := "my people", "the old ways"
	if c != nil && strings.HasPrefix(c.Faction, "state:") {
		realm = g.FactionName(c.Faction)
	}
	if fk := g.compFaith(a); fk != "" {
		faith = g.FactionName(fk)
	}
	place := ""
	if g.Cur != nil {
		place = g.Cur.Name
	}
	town := "home"
	if c != nil && c.Town != "" {
		town = c.Town
	}
	pairs := []string{"{me}", givenOf(a.Name), "{you}", givenOf(g.P.Name), "{town}", town, "{kin}", kin, "{realm}", realm,
		"{faith}", faith, "{place}", place, "{house}", nonEmpty(a.House, givenOf(a.Name)), "{coin}", g.Sk.Currency}
	keys := make([]string, 0, len(vars))
	for k := range vars {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		pairs = append(pairs, "{"+k+"}", vars[k])
	}
	return strings.NewReplacer(pairs...).Replace(s)
}

// cbPick chooses a line for the trigger fitting the companion (hash, not g.R). "" if none.
func (g *Game) cbPick(trigger string, a *Actor, vars map[string]string) string {
	tags := g.cbTags(a)
	var cands, tagged []string
	for _, l := range cbLines[trigger] {
		if l.Tag == "" {
			cands = append(cands, l.Text)
		} else if tags[l.Tag] {
			tagged = append(tagged, l.Text)
		}
	}
	h := HashStr(fmt.Sprintf("%s:cb:%s:%d:%d", g.W.Seed, trigger, a.ID, g.Turn/60))
	if len(tagged) > 0 && (len(cands) == 0 || h%2 == 0) {
		cands = tagged
	}
	if len(cands) == 0 {
		return ""
	}
	k := int((h / 2) % uint64(len(cands)))
	if len(cands) > 1 && cands[k] == g.cb().Said[a.ID] {
		k = (k + 1) % len(cands)
	}
	g.cb().Said[a.ID] = cands[k]
	return g.cbFill(a, cands[k], vars)
}

func (g *Game) cbSay(a *Actor, text string) {
	if r := []rune(text); len(r) > 0 {
		text = strings.ToUpper(string(r[0])) + string(r[1:])
	}
	g.Msg(fmt.Sprintf("{{c|%s:}} \"%s\"", givenOf(a.Name), text))
}

// cbSpeak: a regular line (respects all cooldowns). Returns whether something was said.
func (g *Game) cbSpeak(trigger string, a *Actor, vars map[string]string) bool {
	if !g.cbCan(a) {
		return false
	}
	t := g.cbPick(trigger, a, vars)
	if t == "" {
		return false
	}
	st := g.cb()
	st.Last[a.ID], st.Global = g.Turn, g.Turn+1
	g.cbSay(a, t)
	return true
}

// cbQuipLine: a one-liner allowed in combat (kills, near death, level-ups).
func (g *Game) cbQuipLine(trigger string, a *Actor, vars map[string]string) bool {
	if !g.cbCanQuip(a) {
		return false
	}
	t := g.cbPick(trigger, a, vars)
	if t == "" {
		return false
	}
	st := g.cb()
	st.Quip[a.ID] = g.Turn
	g.cbSay(a, t)
	return true
}

// ---- triggers ----

// cbEnter: arriving somewhere (enter hook).
func (g *Game) cbEnter(zn *Zone) {
	st := g.cb()
	for _, a := range g.cbTalkers() {
		c := a.Comp
		home := g.cbHomeBurg(c)
		switch {
		case zn.Key == c.HomeKey || home != nil && zn.Burg == home.ID && zn.Burg > 0:
			g.cbSpeak("home", a, nil)
		case zn.Kind == "wonder" && !st.Seen[fmt.Sprintf("%d:%s", a.ID, zn.Name)]:
			if g.cbSpeak("wonder", a, nil) {
				st.Seen[fmt.Sprintf("%d:%s", a.ID, zn.Name)] = true
			}
		case (zn.Kind == "ruin" || zn.Kind == "crypt") && !st.Seen[fmt.Sprintf("%d:%s", a.ID, zn.Name)]:
			if g.cbSpeak("ruin", a, nil) {
				st.Seen[fmt.Sprintf("%d:%s", a.ID, zn.Name)] = true
			}
		case zn.Burg > 0 && strings.HasPrefix(c.Faction, "state:") && strings.HasPrefix(zn.Faction, "state:") && g.atWar(c.Faction, zn.Faction):
			g.cbSpeak("enemy", a, map[string]string{"foe": g.FactionName(zn.Faction)})
		case zn.Burg > 0 && g.cbTemple(zn, a):
			g.cbSpeak("temple", a, nil)
		case zn.Burg > 0 && zn.Faction == c.Faction && c.Faction != "":
			g.cbSpeak("realm", a, nil)
		}
	}
}

// cbTemple: a priest of the companion's faith serves here.
func (g *Game) cbTemple(zn *Zone, a *Actor) bool {
	if a.Religion == 0 {
		return false
	}
	for _, o := range zn.Actors {
		if o.Role == "priest" && o.Religion == a.Religion && o.HP > 0 {
			return true
		}
	}
	return false
}

var cbWeather = map[string]string{"storm": "storm", "tempest": "storm", "radstorm": "storm", "blizzard": "snow", "snow": "snow",
	"rain": "rain", "acidrain": "rain", "sporerain": "rain", "fallout": "rain", "ashrain": "rain", "heat": "heat", "fog": "fog",
	"smog": "fog", "steamfog": "fog", "miasma": "fog", "sandstorm": "storm", "bloodmoon": "omen"}

// cbTick: once a game minute (turn hook).
func (g *Game) cbTick() {
	if g.Cur == nil || g.OnWorld {
		return
	}
	talkers := g.cbTalkers()
	if len(talkers) == 0 {
		return
	}
	st := g.cb()
	for _, a := range talkers {
		c := a.Comp
		// level-ups
		if lv, ok := st.Lvl[a.ID]; !ok || a.Level < lv {
			st.Lvl[a.ID] = a.Level
		} else if a.Level > lv {
			st.Lvl[a.ID] = a.Level
			g.cbQuipLine("levelup", a, nil)
		}
		// near death
		if a.HP*4 < a.MaxHP {
			if !st.Low[a.ID] && g.cbQuipLine("neardeath", a, nil) {
				st.Low[a.ID] = true
			}
		} else if a.HP*2 > a.MaxHP {
			delete(st.Low, a.ID)
		}
		// shared meals and night camps
		if c.MealAt != st.Meal[a.ID] {
			st.Meal[a.ID] = c.MealAt
			if c.MealAt != 0 && g.Turn-c.MealAt < 5 {
				trig := "meal"
				if !g.daylight() {
					trig = "camp"
				}
				st.Global = 0 // mealtime talk is welcome
				g.cbSpeak(trig, a, nil)
			}
		}
	}
	if g.Turn%30 != 0 {
		return
	}
	who := talkers[int(HashStr(fmt.Sprintf("%s:cbw:%d", g.W.Seed, g.Turn/30))%uint64(len(talkers)))]
	// weather
	if w := g.activeWeather(); w != nil && w.Kind != nil {
		if trig := cbWeather[w.Kind.ID]; trig != "" && st.Wx != w.Kind.ID {
			if g.cbSpeak("wx_"+trig, who, map[string]string{"weather": strings.ToLower(w.Kind.Name)}) {
				st.Wx = w.Kind.ID
			}
		}
	} else if w == nil {
		st.Wx = ""
	}
	// feast days in town
	if b, f := g.feastToday(); f != nil && st.Feast[who.ID] != g.Day() && g.Cur.Burg > 0 {
		if g.cbSpeak("feast", who, map[string]string{"feast": f.Name, "feasttown": b.Name}) {
			st.Feast[who.ID] = g.Day()
		}
	}
	if g.Turn%60 != 0 {
		return
	}
	// grumbles
	for _, a := range talkers {
		if a.Comp.Loyalty < 30 && HashStr(fmt.Sprintf("%s:cbg:%d:%d", g.W.Seed, a.ID, g.Turn/60))%3 == 0 {
			g.cbSpeak("grumble", a, nil)
		}
	}
	// two companions talking
	if len(talkers) >= 2 && HashStr(fmt.Sprintf("%s:cbp:%d", g.W.Seed, g.Turn/60))%8 == 0 {
		n := uint64(len(talkers))
		h := HashStr(fmt.Sprintf("%s:cbq:%d", g.W.Seed, g.Turn/60))
		i := h % n
		a, b := talkers[i], talkers[(i+1+(h/n)%(n-1))%n]
		if a != b && dist(a, b) <= 6 {
			g.cbPair(a, b)
		}
	}
}

// cbPair: a and b exchange a line and a reply (both cooldowns apply).
func (g *Game) cbPair(a, b *Actor) {
	st := g.cb()
	if !g.cbCan(a) || !g.cbCan(b) || st.Pair > 0 && g.Turn-st.Pair < cbPairGap {
		return
	}
	ca, cb2 := a.Comp, b.Comp
	trig := "pair"
	switch {
	case ca.HomeKey == cb2.HomeKey || a.House != "" && a.House == b.House:
		trig = "pair_kin"
	case strings.HasPrefix(ca.Faction, "state:") && strings.HasPrefix(cb2.Faction, "state:") && g.atWar(ca.Faction, cb2.Faction):
		trig = "pair_rival"
	}
	open, re := cbLines[trig], cbLines[trig+"_re"]
	if len(open) == 0 || len(re) == 0 {
		return
	}
	// openers and replies go together by index; cycle through them so pairs rarely repeat
	i := int((HashStr(g.W.Seed+":cbpair:"+trig) + uint64(st.PairN)) % uint64(len(open)))
	st.PairN++
	st.Pair = g.Turn
	g.cbSay(a, g.cbFill(a, open[i].Text, map[string]string{"other": givenOf(b.Name), "otherrealm": g.FactionName(cb2.Faction), "othertown": nonEmpty(cb2.Town, "elsewhere")}))
	g.cbSay(b, g.cbFill(b, re[i%len(re)].Text, map[string]string{"other": givenOf(a.Name), "otherrealm": g.FactionName(ca.Faction), "othertown": nonEmpty(ca.Town, "elsewhere")}))
	st.Last[a.ID], st.Last[b.ID], st.Global = g.Turn, g.Turn, g.Turn+1
}

// cbKill: a companion's kill gets a one-liner now and then (kill hook).
func (g *Game) cbKill(d, src *Actor) {
	if src == nil || src.AI != "follower" || src.Comp == nil || src.Comp.Summoned || d.AI == "follower" {
		return
	}
	if HashStr(fmt.Sprintf("%s:cbk:%d", g.W.Seed, d.ID))%3 != 0 && !d.Boss {
		return
	}
	g.cbQuipLine("kill", src, map[string]string{"victim": g.the(d)})
}

func init() {
	enterHooks = append(enterHooks, func(g *Game, zn *Zone) { g.cbEnter(zn) })
	turnHooks = append(turnHooks, func(g *Game) { g.cbTick() })
	killHooks = append(killHooks, func(g *Game, d, src *Actor) { g.cbKill(d, src) })
}
