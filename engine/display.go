package engine

import (
	"fmt"
	"sort"
	"strings"
)

// Display cases (prefix hm): up to six case tiles in every home of yours (TStatue tiles noted "Display case",
// placed lazily by turn/enter hooks the first time you are in a home zone; side rng seed:hm:zoneKey). Put a
// complete collection, an heirloom, a mounted head, a relic, a grave good, a unique item, a fossil or a
// great find from the finds ledger in a case: it leaves your pack and gives small permanent bonuses while it
// stands there. Mode "hmd". Global state g.ext("hm"); museums live in museum.go (planet-local "hml").
// Nothing here draws g.R.

const (
	hmKey  = "hm"
	hmLKey = "hml"
)

func init() {
	unLocalKeys = append(unLocalKeys, hmLKey)
	zoneHooks = append(zoneHooks, hmZone)
	turnHooks = append(turnHooks, hmTurn)
	enterHooks = append(enterHooks, hmEnter)
	talkHooks = append(talkHooks, hmTalk)
	sideHooks = append(sideHooks, hmSide)
	modeKeys["hmd"] = hmdKeys
	modeDraws["hmd"] = hmdDraw
	modeKeys["hmm"] = hmmKeys
	modeDraws["hmm"] = hmmDraw
}

type hmCase struct {
	Kind  string  `json:"k"`
	Title string  `json:"t"`
	Items []*Item `json:"i,omitempty"`
	Find  int     `json:"f,omitempty"` // finds ledger index + 1
	Set   string  `json:"s,omitempty"`
	Val   int     `json:"v,omitempty"`
	Day   int     `json:"d,omitempty"`
}

type hmHome struct {
	Spots [][2]int  `json:"sp"`
	Cases []*hmCase `json:"c"`
}

type hmState struct {
	Homes  map[string]*hmHome `json:"h,omitempty"`
	Seen   map[string]bool    `json:"s,omitempty"` // first-display rewards given
	Firsts map[string]bool    `json:"f,omitempty"`
	Last   map[string]int     `json:"l,omitempty"` // home → last day there + 1
	Shown  int                `json:"n,omitempty"`
	Told   int                `json:"t,omitempty"`
	Stolen int                `json:"b,omitempty"`
	Gave   int                `json:"g,omitempty"` // items donated to museums
	Titles []string           `json:"ti,omitempty"`

	h        *hmHome
	msg      string
	mus      *hmMus
	fromTalk bool
}

func hmSt(g *Game) *hmState {
	s := g.ext(hmKey, func() any { return &hmState{} }).(*hmState)
	if s.Homes == nil {
		s.Homes = map[string]*hmHome{}
	}
	if s.Seen == nil {
		s.Seen = map[string]bool{}
	}
	if s.Firsts == nil {
		s.Firsts = map[string]bool{}
	}
	if s.Last == nil {
		s.Last = map[string]int{}
	}
	return s
}

func (g *Game) hmFirst(key, text string) {
	s := hmSt(g)
	if s.Firsts[key] {
		return
	}
	s.Firsts[key] = true
	g.deed("event", text, 0, 0)
}

func hmHomeKey(g *Game, zkey string) string { return fmt.Sprintf("%d:%s", g.W.uniIdx, zkey) }

func hmArt(g *Game, id int) *ArtifactData {
	for i := range g.W.Artifacts {
		if g.W.Artifacts[i].ID == id {
			return &g.W.Artifacts[i]
		}
	}
	return nil
}

// ---------------------------------------------------------------- placing the cases

func hmZone(g *Game, zn *Zone, _ *Rng) { hmMuseumZone(g, zn) }

func hmSpots(g *Game, zn *Zone, h *homeRec) [][2]int {
	free := func(x, y int) bool {
		t := zn.At(x, y)
		return zn.In(x, y) && (t == TCarpet || t == TFloor) && zn.ActorAt(x, y) == nil && len(zn.ItemsAt(x, y)) == 0 &&
			zn.Notes[y*MW+x] == "" && !(x == h.X && y == h.Y)
	}
	var out [][2]int
	x0, y0 := h.SX-5, h.SY-1
	if h.SX > 5 && zn.At(x0, y0) == TWall && zn.At(x0+3, y0+4) == TDoor && zn.At(h.SX, h.SY) == TShelf {
		for _, c := range [][2]int{{x0 + 2, y0 + 1}, {x0 + 4, y0 + 1}, {x0 + 1, y0 + 3}, {x0 + 5, y0 + 3}, {x0 + 1, y0 + 2}, {x0 + 5, y0 + 2}} {
			if free(c[0], c[1]) {
				out = append(out, c)
			}
		}
		return out
	}
	r := NewRng(g.Seed + ":hm:" + zn.Key)
	type cand struct{ x, y, d int }
	var cs []cand
	for y := h.Y - 5; y <= h.Y+5; y++ {
		for x := h.X - 5; x <= h.X+5; x++ {
			if !zn.In(x, y) || !zn.Walkable(x, y) || abs(x-h.X)+abs(y-h.Y) < 2 || (x == h.SX && y == h.SY) {
				continue
			}
			t := zn.At(x, y)
			if t == TUp || t == TDown || t == TDoor || t == TDoorOpen || t == TBed || t == TShelf || t == TRoad || zn.WaterAt(x, y) > 0 {
				continue
			}
			if zn.ActorAt(x, y) != nil || len(zn.ItemsAt(x, y)) > 0 || zn.Notes[y*MW+x] != "" {
				continue
			}
			n := 0
			for _, d := range dirs8 {
				if zn.Walkable(x+d[0], y+d[1]) {
					n++
				}
			}
			if n < 3 || n > 6 {
				continue
			}
			cs = append(cs, cand{x, y, abs(x-h.X) + abs(y-h.Y) + r.Int(0, 2)})
		}
	}
	sort.SliceStable(cs, func(i, j int) bool { return cs[i].d < cs[j].d })
	want := 4 + r.Int(0, 2)
	for _, c := range cs {
		if len(out) >= want {
			break
		}
		ok := true
		for _, o := range out {
			if abs(o[0]-c.x) <= 1 && abs(o[1]-c.y) <= 1 {
				ok = false
			}
		}
		if ok {
			out = append(out, [2]int{c.x, c.y})
		}
	}
	return out
}

// hmEnsure makes sure the home you stand in has its case tiles.
func hmEnsure(g *Game) *hmHome {
	h := g.homeHere()
	if h == nil || g.Cur == nil || g.W == nil {
		return nil
	}
	st := hmSt(g)
	key := hmHomeKey(g, h.Key)
	if rec := st.Homes[key]; rec != nil {
		return rec
	}
	zn := g.Cur
	rec := &hmHome{}
	for _, c := range hmSpots(g, zn, h) {
		if len(rec.Spots) >= 6 {
			break
		}
		zn.Set(c[0], c[1], TStatue)
		zn.Notes[c[1]*MW+c[0]] = "Display case"
		rec.Spots = append(rec.Spots, c)
		rec.Cases = append(rec.Cases, nil)
	}
	st.Homes[key] = rec
	if len(rec.Spots) > 0 {
		g.Msg("{{c|Display cases stand ready in your home. Bump one, or press d in your stash (H).}}")
		g.recompute()
	}
	return rec
}

// hmBump: bumping a display case opens the cases screen. Hook: top of interact().
func (g *Game) hmBump(x, y int) bool {
	if g.Cur == nil || g.Mode != "main" {
		return false
	}
	h := g.homeHere()
	if h == nil {
		return false
	}
	rec := hmSt(g).Homes[hmHomeKey(g, h.Key)]
	if rec == nil {
		return false
	}
	for i, s := range rec.Spots {
		if s[0] == x && s[1] == y {
			g.hmOpenDisplay()
			hmSt(g).h = rec
			g.Sel2 = i
			return true
		}
	}
	return false
}

func (g *Game) hmOpenDisplay() {
	if g.Mode != "main" {
		return
	}
	rec := hmEnsure(g)
	if rec == nil {
		g.Msg("You have no home here.")
		return
	}
	if len(rec.Spots) == 0 {
		g.Msg("There is no room for display cases here.")
		return
	}
	hmSt(g).h = rec
	hmSt(g).msg = ""
	g.Mode = "hmd"
	g.Tab, g.Sel, g.Sel2 = 0, 0, 0
}

// hmStashHint: add to the stash help line in drawStash.
func hmStashHint(g *Game) string { return "  {{W|d}} display cases" }

// ---------------------------------------------------------------- candidates

type hmCand struct {
	Kind, Title, Set string
	Items            []*Item
	Find             int
	Val              int
}

func hmDisplayedFind(g *Game, idx int) bool {
	for _, h := range hmSt(g).Homes {
		for _, c := range h.Cases {
			if c != nil && c.Find == idx+1 {
				return true
			}
		}
	}
	return false
}

func hmCands(g *Game) []hmCand {
	var out []hmCand
	held := clHeld(g)
	for _, s := range clSets {
		if clSetHeld(held, s) < len(s.Items) {
			continue
		}
		var its []*Item
		for _, di := range s.Items {
			for _, it := range g.P.Inv {
				if it.Extra["cl:i"] == di+1 && it.Extra["cl:q"] == 0 {
					its = append(its, it)
					break
				}
			}
		}
		if len(its) == len(s.Items) {
			v := 0
			for _, it := range its {
				v += it.Value
			}
			out = append(out, hmCand{Kind: hmKSet, Title: "Collection: " + s.Title, Set: s.ID, Items: its, Val: v})
		}
	}
	for _, it := range g.P.Inv {
		if it.Quest > 0 || g.P.Equipped(it) != nil || it.Extra["bu:app"] > 0 {
			continue
		}
		k := ""
		switch {
		case it.SimArt > 0:
			k = hmKRelic
		case it.Extra["bctroph"] > 0:
			k = hmKHead
		case it.Kind == "fossil":
			k = hmKFossil
		case ggEligible(it) && ggLv(it) > 0:
			k = hmKGear
		case it.Extra["bu:find"] > 0 || it.Extra["bu:nom"] > 0:
			k = hmKGrave
		case it.Rarity == "unique":
			k = hmKUnique
		}
		if k != "" {
			out = append(out, hmCand{Kind: k, Title: Plain(it.Display()), Items: []*Item{it}, Val: it.Value})
		}
	}
	for i, f := range buSt(g).Finds {
		if f.Value > 0 && !hmDisplayedFind(g, i) {
			out = append(out, hmCand{Kind: hmKFind, Title: f.Title, Find: i + 1, Val: f.Value})
		}
	}
	return out
}

// hmSplit takes exactly one item out of the pack (splitting stacks).
func hmSplit(g *Game, it *Item) *Item {
	if it.Stack() && it.Count > 1 {
		it.Count--
		cp := *it
		cp.Count = 1
		return &cp
	}
	g.removeInv(it)
	return it
}

// ---------------------------------------------------------------- bonuses

func hmPrestige(c *hmCase) int {
	if c == nil {
		return 0
	}
	switch c.Kind {
	case hmKSet:
		return 10
	case hmKGear:
		if len(c.Items) > 0 {
			return 5 + 3*ggLv(c.Items[0])
		}
	case hmKHead:
		if len(c.Items) > 0 {
			return 4 + 2*c.Items[0].Extra["bctroph"]
		}
	case hmKRelic:
		return 15 + c.Val/60
	case hmKFossil:
		return 3 + c.Val/40
	case hmKFind:
		return 4 + c.Val/50
	case hmKGrave:
		return 6 + c.Val/50
	case hmKUnique:
		return 12
	}
	return 1
}

type hmTotals struct {
	Prestige, Sets, Heads, Cases int
	Stat                         [6]int
}

func hmTotal(g *Game) hmTotals {
	var t hmTotals
	left := 3
	keys := make([]string, 0, len(hmSt(g).Homes))
	for k := range hmSt(g).Homes {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		for _, c := range hmSt(g).Homes[k].Cases {
			if c == nil {
				continue
			}
			t.Cases++
			t.Prestige += hmPrestige(c)
			switch c.Kind {
			case hmKSet:
				t.Sets++
			case hmKHead:
				t.Heads++
			case hmKGear:
				it := c.Items[0]
				n := mini(ggLv(it)/2, left)
				if n > 0 {
					st := TOU
					if ggIsWeapon(it) {
						st = STR
						if it.Kind == "ranged" {
							st = AGI
						}
					}
					t.Stat[st] += n
					left -= n
				}
			}
		}
	}
	return t
}

var hmCacheP *Actor
var hmCache [6]int

func hmRefresh(g *Game) {
	if g.P == nil || g.W == nil {
		return
	}
	hmCacheP = g.P
	hmCache = hmTotal(g).Stat
}

// hmScoreMod: attribute bonus from displayed heirlooms. Hook: end of Actor.Score, `v += hmScoreMod(a, i)`.
func hmScoreMod(a *Actor, i int) int {
	if a == nil || a != hmCacheP || i < 0 || i > 5 {
		return 0
	}
	return hmCache[i]
}

// hmPriceMod: +1% on sales per displayed set; a town museum's patrons get a discount from its merchants.
// Hooks: price() sell branch `f *= hmPriceMod(g, g.Talk, false)`, buy branch `f *= hmPriceMod(g, g.Talk, true)`.
func hmPriceMod(g *Game, a *Actor, buy bool) float64 {
	if buy {
		if g.Cur != nil && g.W != nil && g.Cur.Kind == "town" {
			if m := hmL(g).Mus[fmt.Sprint(g.Cur.Burg)]; m != nil && hmCount(m) >= hmMileN[2] {
				return 0.93
			}
		}
		return 1
	}
	return 1 + 0.01*float64(hmTotal(g).Sets)
}

// ---------------------------------------------------------------- put / take

func (g *Game) hmPut(rec *hmHome, slot int, c hmCand) {
	cs := &hmCase{Kind: c.Kind, Title: c.Title, Set: c.Set, Find: c.Find, Val: c.Val, Day: g.Day()}
	for _, it := range c.Items {
		cs.Items = append(cs.Items, hmSplit(g, it))
	}
	rec.Cases[slot] = cs
	st := hmSt(g)
	st.Shown++
	key := fmt.Sprintf("%s:%s:%d", c.Kind, c.Set, c.Find)
	if len(cs.Items) == 1 {
		key = fmt.Sprintf("i:%d", cs.Items[0].ID)
	}
	if !st.Seen[key] {
		st.Seen[key] = true
		xp := 10
		switch c.Kind {
		case hmKFossil, hmKFind:
			xp = 25
		case hmKRelic:
			xp = 20
			g.hmRelicHonour(cs.Items[0])
		}
		g.gainXP(xp)
		g.Msg(fmt.Sprintf("{{Y|%s}} goes on display. {{K|(+%d XP the first time)}}", c.Title, xp))
	} else {
		g.Msg(Plain(c.Title) + " goes back on display.")
	}
	g.hmFirst("first", fmt.Sprintf("The wanderer %s set the first piece in a home display case.", g.P.Name))
	if c.Kind == hmKRelic {
		g.hmFirst("relic", fmt.Sprintf("The wanderer %s put the relic %s on display at home.", g.P.Name, cs.Items[0].Name))
	}
	if c.Kind == hmKSet {
		g.hmFirst("set", fmt.Sprintf("The wanderer %s mounted a whole collection in a case.", g.P.Name))
	}
	hmRefresh(g)
}

func (g *Game) hmRelicHonour(it *Item) {
	realm := 0
	art := hmArt(g, it.SimArt)
	if art != nil && art.P >= 0 && art.P < len(g.W.State) {
		realm = g.W.State[art.P]
	}
	if realm <= 0 {
		realm = g.fmRealm()
	}
	if realm > 0 {
		g.AddRep(fmt.Sprintf("state:%d", realm), 5, " for keeping their relic with honour")
		g.fmFame(realm, 1)
	}
	if art != nil && art.P >= 0 && art.P < len(g.W.Religion) {
		if rel := g.W.Religion[art.P]; rel > 0 {
			g.AddRep(fmt.Sprintf("faith:%d", rel), 3, "")
		}
	}
}

func (g *Game) hmTake(rec *hmHome, slot int) {
	c := rec.Cases[slot]
	if c == nil {
		return
	}
	for _, it := range c.Items {
		g.give(it)
	}
	rec.Cases[slot] = nil
	g.Msg("You take " + Plain(c.Title) + " out of the case.")
	hmRefresh(g)
}

// ---------------------------------------------------------------- hooks: turn, enter, side, talk

func hmTurn(g *Game) {
	if g.P == nil || g.Cur == nil || g.W == nil {
		return
	}
	if hmCacheP != g.P {
		hmRefresh(g)
	}
	h := g.homeHere()
	if h == nil {
		return
	}
	hmEnsure(g)
	st := hmSt(g)
	if g.Turn%60 == 0 {
		st.Last[hmHomeKey(g, h.Key)] = g.Day() + 1
	}
	if heads := hmTotal(g).Heads; heads > 0 && g.P.HP < g.P.MaxHP && g.Turn%(20-3*mini(heads, 4)) == 0 {
		g.P.HP++
	}
}

func hmGuarded(g *Game, h *homeRec) bool {
	for _, a := range g.Cur.Actors {
		if a.Flags["stayhome"] && a.HP > 0 {
			return true
		}
	}
	for id, L := range g.lkSt().Locks { // optional read: a good lock on the home door
		if strings.HasPrefix(id, h.Key+":") && L.Kind == "door" && L.Tier >= 3 && !L.Broken {
			return true
		}
	}
	return false
}

func hmEnter(g *Game, zn *Zone) {
	h := g.homeHere()
	if h == nil || g.W == nil {
		return
	}
	rec := hmEnsure(g)
	st := hmSt(g)
	key := hmHomeKey(g, h.Key)
	day := g.Day()
	last := st.Last[key] - 1
	st.Last[key] = day + 1
	if rec == nil {
		return
	}
	filled := 0
	for _, c := range rec.Cases {
		if c != nil {
			filled++
		}
	}
	if filled > 0 && last >= 0 && day-last > 10 && !hmGuarded(g, h) &&
		HashStr(fmt.Sprintf("%s:hm:burgle:%s:%d", g.Seed, key, day))%1000 < 80 {
		n := 1 + int(HashStr(fmt.Sprintf("%s:hm:n:%s:%d", g.Seed, key, day))%3)
		var gone []string
		for i := 0; i < len(rec.Cases) && len(gone) < n; i++ {
			j := (i + int(HashStr(fmt.Sprintf("%s:hm:p:%s:%d", g.Seed, key, day))%uint64(len(rec.Cases)))) % len(rec.Cases)
			if c := rec.Cases[j]; c != nil {
				gone = append(gone, Plain(c.Title))
				rec.Cases[j] = nil
			}
		}
		st.Stolen += len(gone)
		g.Msg(hmBurgleLines[int(HashStr(key)%uint64(len(hmBurgleLines)))])
		g.Msg("{{R|Gone from the cases: " + strings.Join(gone, ", ") + ".}}")
		g.deed("event", fmt.Sprintf("Thieves emptied the display cases in the home of the wanderer %s while they were away.", g.P.Name), 0, 0)
		hmRefresh(g)
	}
	if g.fmMarried() && st.Told < st.Shown && hmTotal(g).Cases > 0 {
		sp := g.fm().Spouse
		g.Msg("{{c|" + fmt.Sprintf(hmSpouseLines[int(HashStr(fmt.Sprint(st.Shown))%uint64(len(hmSpouseLines)))], sp.Name) + "}}")
	}
	st.Told = st.Shown
}

func hmSide(g *Game) []string {
	if g.homeHere() == nil {
		return nil
	}
	if t := hmTotal(g); t.Cases > 0 {
		return []string{fmt.Sprintf("{{c|Prestige: %d}}", t.Prestige)}
	}
	return nil
}

func hmTalk(g *Game, a *Actor) []Option {
	if a.Role == "curator" {
		return hmCuratorTalk(g, a)
	}
	if a.Has("fm_spouse") && g.homeHere() != nil {
		t := hmTotal(g)
		if t.Cases == 0 {
			return nil
		}
		return []Option{{"Ask what they think of the display cases.", func() {
			name := "newest piece"
			for _, h := range hmSt(g).Homes {
				for _, c := range h.Cases {
					if c != nil {
						name = Plain(c.Title)
					}
				}
			}
			g.talkMenu(fmt.Sprintf(hmSpouseAsk[int(HashStr(a.Name)%uint64(len(hmSpouseAsk)))], name))
		}}}
	}
	return nil
}

// hmAbilities / hmAbUse: Abilities entry "hm_display". Hooks: powers.go Abilities() and usePower().
func (g *Game) hmAbilities() []Ability {
	if g.Cur == nil || g.P == nil || g.W == nil {
		return nil
	}
	if g.homeHere() != nil {
		return []Ability{{ID: "hm_display", Name: "Display cases", Target: "self"}}
	}
	if hmMuseumHere(g) != nil {
		return []Ability{{ID: "hm_display", Name: "Museum rolls", Target: "self"}}
	}
	return nil
}

func (g *Game) hmAbUse(id string) {
	if id != "hm_display" {
		return
	}
	if g.homeHere() != nil {
		g.hmOpenDisplay()
	} else if m := hmMuseumHere(g); m != nil {
		g.hmOpenMuseum(m, false)
	}
}

// ---------------------------------------------------------------- screen

func hmdKeys(g *Game, k string) {
	st := hmSt(g)
	rec := st.h
	if rec == nil {
		g.Mode = "main"
		return
	}
	cands := hmCands(g)
	n := len(cands)
	if g.Tab == 1 {
		n = len(rec.Cases)
	}
	switch k {
	case "Escape":
		g.Mode = "main"
	case "ArrowUp":
		g.Sel = maxi(0, g.Sel-1)
	case "ArrowDown":
		g.Sel = mini(maxi(0, n-1), g.Sel+1)
	case "PageUp":
		g.Sel = maxi(0, g.Sel-10)
	case "PageDown":
		g.Sel = mini(maxi(0, n-1), g.Sel+10)
	case "Tab", "ArrowLeft", "ArrowRight":
		g.Sel, g.Sel2 = g.Sel2, g.Sel
		g.Tab ^= 1
	case "Enter", " ":
		if g.Tab == 1 {
			if g.Sel < len(rec.Cases) && rec.Cases[g.Sel] != nil {
				g.hmTake(rec, g.Sel)
			}
			return
		}
		if g.Sel >= len(cands) {
			return
		}
		slot := -1
		if g.Sel2 < len(rec.Cases) && rec.Cases[g.Sel2] == nil {
			slot = g.Sel2
		}
		for i := 0; slot < 0 && i < len(rec.Cases); i++ {
			if rec.Cases[i] == nil {
				slot = i
			}
		}
		if slot < 0 {
			st.msg = "Every case is full. Take something out first."
			return
		}
		st.msg = ""
		g.hmPut(rec, slot, cands[g.Sel])
		g.Sel = mini(g.Sel, maxi(0, len(hmCands(g))-1))
	}
}

func hmDetail(c *hmCase) string {
	if c == nil {
		return "An empty case."
	}
	switch c.Kind {
	case hmKSet:
		var ns []string
		for _, it := range c.Items {
			ns = append(ns, Plain(it.Name))
		}
		return strings.Join(ns, ", ") + ". {{G|+1% on every sale.}}"
	case hmKGear:
		return fmt.Sprintf("{{G|Heirloom level %d: +%d to a related attribute (3 at most in all).}} %s", ggLv(c.Items[0]), ggLv(c.Items[0])/2, c.Items[0].Lore)
	case hmKHead:
		return "{{G|Slow healing while you are home.}} " + c.Items[0].Lore
	case hmKRelic:
		return "{{G|Honour with its realm and faith, fame, once.}} " + c.Items[0].Lore
	case hmKFind:
		return "A great find from your ledger. Only the record stands here."
	}
	if len(c.Items) > 0 {
		return c.Items[0].Lore
	}
	return ""
}

func hmCandDetail(c hmCand) string {
	cs := &hmCase{Kind: c.Kind, Title: c.Title, Items: c.Items, Val: c.Val}
	return hmDetail(cs)
}

func hmdDraw(g *Game) {
	st := hmSt(g)
	rec := st.h
	if rec == nil {
		return
	}
	s := g.sc
	x, y, w, ph := g.panel("Display cases")
	half := w / 2
	cands := hmCands(g)
	rows := ph - 10
	s.TextB(x, y, fmt.Sprintf("{{Y|You may display}} {{K|(%d)}}", len(cands)), CGray, CBgDark, x+half-1)
	sel := g.Sel
	if g.Tab == 1 {
		sel = g.Sel2
	}
	top := maxi(0, sel-rows+1)
	for i := top; i < len(cands) && i-top < rows; i++ {
		bg := CBgDark
		if i == sel && g.Tab == 0 {
			bg = CBgSel
		}
		s.Fill(x, y+2+i-top, half-2, 1, bg)
		s.TextB(x, y+2+i-top, fmt.Sprintf("{{%s|%s}} %s", hmKindCol[cands[i].Kind], hmKindName[cands[i].Kind], cands[i].Title), CGray, bg, x+half-2)
	}
	cx := x + half
	s.TextB(cx, y, fmt.Sprintf("{{Y|Cases}} {{K|(%d)}}", len(rec.Cases)), CGray, CBgDark, x+w)
	csel := g.Sel
	if g.Tab == 0 {
		csel = g.Sel2
	}
	for i, c := range rec.Cases {
		bg := CBgDark
		if i == csel && g.Tab == 1 {
			bg = CBgSel
		}
		s.Fill(cx, y+2+i, half-2, 1, bg)
		txt := "{{K|empty}}"
		if c != nil {
			txt = fmt.Sprintf("{{%s|%s}} %s", hmKindCol[c.Kind], hmKindName[c.Kind], Plain(c.Title))
		}
		s.TextB(cx, y+2+i, fmt.Sprintf("%d. %s", i+1, txt), CGray, bg, x+w)
	}
	// detail of the selected line
	detail := ""
	if g.Tab == 0 && g.Sel < len(cands) {
		detail = hmCandDetail(cands[g.Sel])
	} else if g.Tab == 1 && g.Sel < len(rec.Cases) {
		detail = hmDetail(rec.Cases[g.Sel])
	}
	dy := y + ph - 8
	for i, l := range Wrap(detail, w) {
		if i >= 3 {
			break
		}
		s.TextB(x, dy+i, l, CGray, CBgDark, x+w)
	}
	t := hmTotal(g)
	var bs []string
	for i, v := range t.Stat {
		if v > 0 {
			bs = append(bs, fmt.Sprintf("+%d %s", v, AttrShort[i]))
		}
	}
	line := fmt.Sprintf("{{c|Prestige %d}}   sales +%d%%", t.Prestige, t.Sets)
	if len(bs) > 0 {
		line += "   " + strings.Join(bs, " ")
	}
	if t.Heads > 0 {
		line += "   slow healing at home"
	}
	s.TextB(x, y+ph-4, line, CGray, CBgDark, x+w)
	if st.msg != "" {
		s.TextB(x, y+ph-3, "{{R|"+st.msg+"}}", CGray, CBgDark, x+w)
	}
	s.TextB(x, y+ph-2, "{{K|Tab switch  Enter put on display / take back  Esc leave}}", CGray, CBgDark, x+w)
}
