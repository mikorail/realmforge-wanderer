package engine

import (
	"fmt"
	"sort"
	"strings"
)

// Hunting (prefix hn). The bestiary (bs_ species), senses.go (wind, noise), stealth.go (unaware) and the
// butcher products (bsProducts) already exist; this file adds the loop around them:
//   - tracking: "Read the signs" (palette, or ask a hunter or forester) reads prints and droppings in the
//     wild zone for game and wounded animals, and may point the quest arrow at the zone with game;
//   - game, seasons and law: each species has an open and a closed season by biome, each realm a bag limit
//     and (often) a licence, and private preserves around lords' towns; poaching goes to uwAddBounty when a
//     guard or game warden sees it, and marks the carcass otherwise;
//   - the hunt: stalking (the sidebar says if you are upwind and quiet), a clean-shot bonus to dressing when
//     the animal never noticed you, blood trails of wounded animals;
//   - field dressing (screen "hnd": hide, meat, bone and horn, trophy mount) graded by a skill check, and the
//     hunter's journal (screen "hn"): species taken, firsts, heaviest and rarest, with achievements;
//   - lodges: a huntsman in towns near wild country sells licences and preserve leases, buys meat, hides and
//     trophies, and posts a weekly cull job (own tiny list; contractboard.go is untouched).
// State: g.ext("hn") global (tracking rank), g.ext("hnl") planet-local (journal, bag, carcasses, jobs).
// Generation uses side rngs only; g.R only rolls play-time outcomes.

func init() {
	unLocalKeys = append(unLocalKeys, "hnl")
	turnHooks = append(turnHooks, hnTurn)
	killHooks = append(killHooks, hnKill)
	enterHooks = append(enterHooks, hnEnter)
	zoneHooks = append(zoneHooks, hnZone)
	talkHooks = append(talkHooks, hnTalk)
	sideHooks = append(sideHooks, hnSide)
	modeKeys["hn"] = hnKeys
	modeDraws["hn"] = func(g *Game) { g.hnDraw() }
	modeKeys["hnd"] = hnDressKeys
	modeDraws["hnd"] = func(g *Game) { g.hnDressDraw() }
	paEntries = append(paEntries,
		paEntry{Name: "Hunter's journal", Keywords: "hunting game species trophies seasons licence law records tracking", Cat: "Screen", Label: "screen",
			Func: func(g *Game) { g.hnOpen(0) }},
		paEntry{Name: "Read the signs", Keywords: "tracking tracks prints droppings game hunt blood trail", Cat: "Action", Label: "hunt",
			Cond: func(g *Game) bool { return g.hnCanRead() },
			Func: func(g *Game) { g.hnRead(0); g.hnOpen(0) }},
		paEntry{Name: "Field dress a kill", Keywords: "skin butcher hide meat trophy carcass hunt", Cat: "Action", Label: "hunt",
			Cond: func(g *Game) bool { return g.hnNearCarcass() != nil },
			Func: func(g *Game) { g.hnDressOpen() }})
	achDefs = append(achDefs,
		achDef{"hn_first", "First Game", "Take your first game animal.", "Hunt", "", 1, func(g *Game, st *acState) int { return g.hnl().Taken }},
		achDef{"hn_species", "Many Tracks", "Take ten different species.", "Hunt", "", 10, func(g *Game, st *acState) int { return len(g.hnl().Journal) }},
		achDef{"hn_clean", "Clean Shot", "Take ten animals that never knew you were there.", "Hunt", "", 10, func(g *Game, st *acState) int { return g.hnl().Clean }},
		achDef{"hn_trophy", "Trophy Wall", "Mount three trophies.", "Hunt", "", 3, func(g *Game, st *acState) int { return g.hnl().Trophies }},
		achDef{"hn_cull", "Pest Warden", "Finish five cull jobs for lodges.", "Hunt", "", 5, func(g *Game, st *acState) int { return g.hnl().Culls }},
		achDef{"hn_tracker", "Woodsman", "Reach the fourth tracking rank.", "Hunt", "Woodsman", 4, func(g *Game, st *acState) int { return g.hn().Rank }},
	)
	fvRegisterAct(fvAct{
		ID:    "hunt",
		Name:  "Hunting and tracking contest",
		Short: "the hunting contest",
		Fee:   1,
		Intro: "Huntsmen lay a cold trail out past the gate and set hides and antlers by the scales. Tracking skill, keen eyes and real trophies count.",
		Opts:  [3]string{"Follow the cold trail through the brush", "Present your finest hunting record", "Stalk and loose at the moving wood-stag"},
		Kinds: [3]string{"survival", "perception", "survival"},
		Legs:  []string{"the spoor", "the stalk", "the quarry"},
	}, []string{"harvest", "midwinter", "war", "spring"}, func(g *Game, b *BurgData, f *FeastData, ch int) (int, bool, string, bool) {
		rank := g.hn().Rank
		hnl := g.hnl()
		switch ch {
		case 0:
			return g.checkMod("survival") + rank, false, "You drop to one knee and read the pressed grass.", true
		case 1:
			bonus := mini(5, len(hnl.Journal)/2+hnl.Trophies)
			pre := "You lay out your hunter's tally before the lodge judges."
			if hnl.Taken == 0 {
				pre = "You have no kills logged yet and must talk up your field craft."
			}
			return g.checkMod("perception") + bonus, false, pre, true
		default:
			return g.checkMod("survival") + rank/2 + g.Skills["missile"], true, "You slip behind the willow blinds as the cord-pulled stag leaps.", true
		}
	})
}

// ---------------------------------------------------------------- state

type hnGlobal struct {
	Rank int `json:"r"`
	XP   int `json:"x"`
}

func (g *Game) hn() *hnGlobal {
	return g.ext("hn", func() any { return &hnGlobal{} }).(*hnGlobal)
}

type hnEntry struct {
	Taken  int    `json:"t"`
	First  int    `json:"f"` // day + 1
	BestKg int    `json:"b"`
	BestQ  int    `json:"q"`
	Clean  int    `json:"c"`
	Place  string `json:"p,omitempty"`
}

type hnRecord struct {
	SP string `json:"s,omitempty"`
	Kg int    `json:"k,omitempty"`
	R  int    `json:"r,omitempty"`
	D  int    `json:"d,omitempty"`
}

type hnCarcass struct {
	Zone  string `json:"z"`
	X     int    `json:"x"`
	Y     int    `json:"y"`
	Sp    string `json:"s"`
	Kg    int    `json:"k"`
	Clean bool   `json:"c,omitempty"`
	Wound bool   `json:"w,omitempty"`
	Poach int    `json:"p,omitempty"`
	Day   int    `json:"d"`
	Turn  int    `json:"t"`
	Done  int    `json:"n,omitempty"` // bitmask of finished parts
	Realm int    `json:"r,omitempty"`
}

type hnWound struct {
	Zone  string   `json:"z"`
	Sp    string   `json:"s"`
	Marks [][2]int `json:"m,omitempty"`
	X     int      `json:"x"`
	Y     int      `json:"y"`
	Turn  int      `json:"t"`
}

type hnCull struct {
	Burg   int    `json:"b"`
	Week   int    `json:"w"`
	Sp     string `json:"s"`
	Need   int    `json:"n"`
	Got    int    `json:"g"`
	Reward int    `json:"r"`
	Taken  bool   `json:"t,omitempty"`
	Done   bool   `json:"d,omitempty"`
}

type hnLocal struct {
	Journal  map[string]*hnEntry `json:"j"`
	Heavy    hnRecord            `json:"hv"`
	Rare     hnRecord            `json:"rr"`
	Bag      map[string]int      `json:"bg"`
	Pending  []*hnCarcass        `json:"pc,omitempty"`
	Wounds   map[int]*hnWound    `json:"wd"`
	Cull     map[int]*hnCull     `json:"cl"`
	Job      *hnCull             `json:"jb,omitempty"`
	Warded   map[string]bool     `json:"wa"`
	Noticed  map[string]int      `json:"nt"`
	Hints    []string            `json:"h,omitempty"`
	Taken    int                 `json:"tk,omitempty"`
	Clean    int                 `json:"cn,omitempty"`
	Trophies int                 `json:"tr,omitempty"`
	Poached  int                 `json:"po,omitempty"`
	Fines    int                 `json:"fi,omitempty"`
	Culls    int                 `json:"cu,omitempty"`
	Earned   int                 `json:"ea,omitempty"`
	Trails   int                 `json:"tl,omitempty"`

	tab, sel int
	msg      string
	unaware  map[int]bool
	hp       map[int]int
	dress    *hnCarcass
	dsel     int
	dmsg     string
}

func (g *Game) hnl() *hnLocal {
	s := g.ext("hnl", func() any {
		return &hnLocal{Journal: map[string]*hnEntry{}, Bag: map[string]int{}, Wounds: map[int]*hnWound{}, Cull: map[int]*hnCull{},
			Warded: map[string]bool{}, Noticed: map[string]int{}}
	}).(*hnLocal)
	if s.Journal == nil {
		s.Journal = map[string]*hnEntry{}
	}
	if s.Bag == nil {
		s.Bag = map[string]int{}
	}
	if s.Wounds == nil {
		s.Wounds = map[int]*hnWound{}
	}
	if s.Cull == nil {
		s.Cull = map[int]*hnCull{}
	}
	if s.Warded == nil {
		s.Warded = map[string]bool{}
	}
	if s.Noticed == nil {
		s.Noticed = map[string]int{}
	}
	if s.unaware == nil {
		s.unaware = map[int]bool{}
	}
	if s.hp == nil {
		s.hp = map[int]int{}
	}
	return s
}

func (g *Game) hnXP(n int) {
	h := g.hn()
	h.XP += n
	for h.Rank < 5 && h.XP >= hnXPAt[h.Rank] {
		h.Rank++
		g.Msg(fmt.Sprintf("{{G|Your tracking improves:}} {{Y|%s}}.", hnRankTitle[h.Rank]))
		g.deed("event", fmt.Sprintf("The wanderer %s grew into a %s.", g.P.Name, strings.ToLower(hnRankTitle[h.Rank])), 0, 0)
	}
}

// ---------------------------------------------------------------- species rules

func hnSp(a *Actor) *bsSp {
	if a == nil || !strings.HasPrefix(a.Tpl, "bs_") {
		return nil
	}
	return bsByTpl[a.Tpl]
}

func hnIsGame(s *bsSp) bool {
	if s == nil || s.Hostile() || s.Size < 2 || s.Aquatic() {
		return false
	}
	switch s.Group {
	case "mam", "bird", "rept", "dino":
		return true
	}
	return false
}

func hnDanger(s *bsSp) bool {
	return s != nil && s.Size >= 2 && !s.Aquatic() && (s.Hostile() || s.Temper == "territorial" && s.Size >= 3)
}

func hnHuntable(s *bsSp) bool { return hnIsGame(s) || hnDanger(s) }

func (g *Game) hnClosed(s *bsSp, biome, season int) (bool, string) {
	if s.Rarity >= 4 {
		return true, "protected all year"
	}
	if hnDanger(s) {
		return false, ""
	}
	switch biome {
	case 5, 7, 8:
		return false, "" // breeds all year in the rains
	}
	cold := biome >= 9 && biome <= 11
	closed := -1
	switch {
	case s.Group == "bird":
		closed = map[bool]int{true: 1, false: 0}[cold]
	case s.Diet == 'h':
		closed = map[bool]int{true: 1, false: 0}[cold]
	default:
		closed = int(HashStr("hnseason:"+s.ID) % 4)
	}
	if season == closed {
		return true, "breeding season, closed in " + strings.ToLower(SeasonNames[season])
	}
	return false, ""
}

func (g *Game) hnNeedsLicence(state int) bool {
	if g.W == nil || state <= 0 {
		return false
	}
	st := g.W.stateByID[state]
	if st == nil {
		return false
	}
	switch st.Ideology {
	case "monarchy", "autocracy", "theocracy", "oligarchy", "technocracy":
		return true
	case "republic", "communal":
		return g.uwPct(state, "hlic", 25)
	}
	return g.uwPct(state, "hlic", 50)
}

func (g *Game) hnLimit(state int, s *bsSp) int {
	if state <= 0 {
		return 99
	}
	lim := clampi(7-s.Size, 1, 6)
	if st := g.W.stateByID[state]; st != nil {
		switch st.Ideology {
		case "monarchy", "autocracy", "theocracy":
			lim--
		case "republic", "communal":
			lim++
		}
	}
	if g.uwLawsFor(state).Arms {
		lim--
	}
	return maxi(1, lim)
}

func (g *Game) hnBagKey(state int, s *bsSp) string {
	day := g.Day()
	return fmt.Sprintf("%d|%d|%s", state, day/SeasonDays, s.ID)
}

// hnPreserve: whether a wild zone is part of a private preserve, and who holds it.
func (g *Game) hnPreserve(z *Zone) (bool, int, string) {
	if g.W == nil || z == nil || z.Kind != "surface" || z.Z != 0 || !g.W.In(z.WX, z.WY) {
		return false, 0, ""
	}
	state := g.W.State[g.W.Idx(z.WX, z.WY)]
	st := g.W.stateByID[state]
	if state <= 0 || st == nil || hnOwner[st.Ideology] == "" {
		return false, 0, ""
	}
	near := -1
	for i := range g.W.Burgs {
		b := &g.W.Burgs[i]
		if !b.Alive || b.State != state || b.P < 0 || b.P >= g.W.W*g.W.H {
			continue
		}
		dx, dy := b.P%g.W.W-z.WX, b.P/g.W.W-z.WY
		if dx*dx+dy*dy > 9 {
			continue
		}
		if b.Capital || HashStr(fmt.Sprintf("%s:hn:estate:%d", g.Seed, b.ID))%100 < 35 {
			near = b.ID
			break
		}
	}
	if near < 0 || HashStr(fmt.Sprintf("%s:hn:pre:%d:%d", g.Seed, z.WX, z.WY))%100 >= 60 {
		return false, 0, ""
	}
	return true, state, g.jbBurgName(near)
}

func (g *Game) hnHasLicence(state int, preserve bool) bool {
	for _, it := range g.P.Inv {
		if it.Effect == "hnlic" && it.Extra != nil && it.Extra["hnlic"] == state+1 && it.Extra["hnexp"] > g.Day() && (!preserve || it.Extra["hnpre"] > 0) {
			return true
		}
	}
	return false
}

func (g *Game) hnWitness(state int) *Actor {
	if g.Cur == nil {
		return nil
	}
	for _, a := range g.Cur.Actors {
		if a.HP <= 0 || a == g.P || a.Faction != uwFac(state) && a.Role != "warden" {
			continue
		}
		if !(a.AI == "guard" || a.Role == "guard" || a.Role == "warden") {
			continue
		}
		d := dist(a, g.P)
		if d <= 22 && (g.visible(a.X, a.Y) || d <= 9) {
			return a
		}
	}
	return nil
}

func hnWeight(g *Game, d *Actor, s *bsSp) int {
	h := HashStr(fmt.Sprintf("%s:hn:kg:%d", g.Seed, d.ID))
	return maxi(1, hnSizeKg[clampi(s.Size, 1, 7)]*(70+int(h%61))/100)
}

// ---------------------------------------------------------------- stalking

// hnStalk says whether the animal has not noticed you, and why in a few words.
func (g *Game) hnStalk(a *Actor) (bool, string) {
	dx, dy, sp := g.nsWind()
	ax, ay := a.X-g.P.X, a.Y-g.P.Y
	down := sp > 0 && ax*dx+ay*dy > 0
	quiet := nsNoiseLabel(g.ns().Noise) == "quiet"
	sneak := g.ss().Sneak
	d := dist(a, g.P)
	wind := "still air"
	if sp > 0 {
		wind = map[bool]string{true: "downwind of you", false: "upwind of you"}[down]
	}
	un := g.ssUnaware(a) || (!down && quiet && (sneak || d >= 6) && !a.Angry)
	way := map[bool]string{true: "creeping", false: "walking"}[sneak]
	return un, fmt.Sprintf("%s, %s, %s", wind, nsNoiseLabel(g.ns().Noise), way)
}

func hnSide(g *Game) []string {
	if g.P == nil || g.Cur == nil || g.W == nil || g.Cur.Kind != "surface" || !outdoors(g.Cur) {
		return nil
	}
	var out []string
	if pre, st, name := g.hnPreserve(g.Cur); pre {
		own := hnOwner[g.W.stateByID[st].Ideology]
		out = append(out, fmt.Sprintf("{{y|Preserve of %s (%s)}}", own, name))
	}
	var best *Actor
	for _, a := range g.Cur.Actors {
		s := hnSp(a)
		if s == nil || a.HP <= 0 || !hnHuntable(s) || a.AI == "follower" || !g.visible(a.X, a.Y) || dist(a, g.P) > 18 {
			continue
		}
		if best == nil || dist(a, g.P) < dist(best, g.P) {
			best = a
		}
	}
	if best != nil {
		un, why := g.hnStalk(best)
		col := map[bool]string{true: "G", false: "y"}[un]
		out = append(out, fmt.Sprintf("{{%s|Hunt: %s %d, %s}}", col, bsByTpl[best.Tpl].Display(), dist(best, g.P), why))
	}
	return out
}

// ---------------------------------------------------------------- turn, enter, zone

func hnTurn(g *Game) {
	if g.P == nil || g.Cur == nil || g.W == nil || g.Dead {
		return
	}
	l := g.hnl()
	if g.Turn%DayLen == 0 {
		var keep []*hnCarcass
		for _, c := range l.Pending {
			if g.Turn-c.Turn < 3*DayLen {
				keep = append(keep, c)
			}
		}
		l.Pending = keep
		l.hp = map[int]int{}
	}
	for k := range l.unaware {
		delete(l.unaware, k)
	}
	if g.Cur.Z != 0 || g.Cur.Kind == "town" {
		return
	}
	for _, a := range g.Cur.Actors {
		s := hnSp(a)
		if s == nil || a.HP <= 0 {
			continue
		}
		if prev, ok := l.hp[a.ID]; ok && a.HP < prev && hnHuntable(s) && dist(a, g.P) <= 25 {
			w := l.Wounds[a.ID]
			if w == nil && len(l.Wounds) < 8 {
				w = &hnWound{Zone: g.Cur.Key, Sp: s.ID}
				l.Wounds[a.ID] = w
			}
			if w != nil && len(w.Marks) == 0 {
				w.Marks = append(w.Marks, [2]int{a.X, a.Y})
			}
		}
		l.hp[a.ID] = a.HP
		if w := l.Wounds[a.ID]; w != nil {
			if n := len(w.Marks); n == 0 || w.Marks[n-1] != [2]int{a.X, a.Y} {
				w.Marks = append(w.Marks, [2]int{a.X, a.Y})
				if len(w.Marks) > 16 {
					w.Marks = w.Marks[1:]
				}
			}
			w.X, w.Y, w.Turn = a.X, a.Y, g.Turn
		}
		if hnHuntable(s) && a.AI != "follower" && dist(a, g.P) <= 30 {
			if un, _ := g.hnStalk(a); un {
				l.unaware[a.ID] = true
			}
		}
	}
	for id, w := range l.Wounds {
		if w.Zone != g.Cur.Key {
			continue
		}
		alive := false
		for _, a := range g.Cur.Actors {
			if a.ID == id && a.HP > 0 {
				alive = true
			}
		}
		if !alive || g.Turn-w.Turn > 3*DayLen {
			delete(l.Wounds, id)
		}
	}
}

func hnEnter(g *Game, zn *Zone) {
	if g.P == nil || g.W == nil || zn == nil {
		return
	}
	pre, st, name := g.hnPreserve(zn)
	if !pre {
		return
	}
	l := g.hnl()
	own := hnOwner[g.W.stateByID[st].Ideology]
	if l.Noticed[zn.Key] != g.Day()+1 {
		l.Noticed[zn.Key] = g.Day() + 1
		g.Msg(fmt.Sprintf("{{y|Boundary stones and a game warden's board: this is a preserve of %s, kept for %s. Hunting here needs a lease.}}", own, name))
	}
	if l.Warded[zn.Key] {
		return
	}
	l.Warded[zn.Key] = true
	r := NewRng(fmt.Sprintf("%s:hn:ward:%s", g.Seed, zn.Key))
	if !r.Chance(0.5) {
		return
	}
	var bd *BurgData
	for i := range g.W.Burgs {
		b := &g.W.Burgs[i]
		if b.State == st && b.Alive {
			bd = b
			break
		}
	}
	if bd == nil {
		return
	}
	x, y, ok := zn.FreeSpot(r, nil)
	if !ok {
		return
	}
	cu := g.W.cultByID[bd.Culture]
	a := g.makeNPC("warden", bd, cu, r)
	a.Title, a.F, a.AI = "game warden", CGreen, "guard"
	a.X, a.Y, a.HomeX, a.HomeY = x, y, x, y
	a.Desc = "A game warden in the livery of " + own + ", watching the stones and the trails."
	zn.Actors = append(zn.Actors, a)
}

// hnZone places a huntsman (hunting lodge) in some towns near wild country.
func hnZone(g *Game, zn *Zone, _ *Rng) {
	if g.W == nil || zn == nil || zn.Kind != "town" || zn.Z != 0 || zn.Burg <= 0 || zn.ZX != 1 || zn.ZY != 1 {
		return
	}
	bd := g.W.burgByID[zn.Burg]
	if bd == nil || bd.P < 0 || bd.P >= g.W.W*g.W.H {
		return
	}
	wild := 0
	for dy := -2; dy <= 2; dy++ {
		for dx := -2; dx <= 2; dx++ {
			x, y := bd.P%g.W.W+dx, bd.P/g.W.W+dy
			if !g.W.In(x, y) {
				continue
			}
			switch g.W.Biome[g.W.Idx(x, y)] {
			case 4, 5, 6, 7, 8, 9, 10, 12:
				wild++
			}
		}
	}
	r := NewRng(fmt.Sprintf("%s:hn:lodge:%s", g.Seed, zn.Key))
	if wild < 5 || !r.Chance(0.65) {
		return
	}
	for _, a := range zn.Actors {
		if a.Role == "huntsman" {
			return
		}
	}
	x, y, ok := zn.FreeSpot(r, nil)
	if !ok {
		return
	}
	cu := g.W.cultByID[bd.Culture]
	a := g.makeNPC("huntsman", bd, cu, r)
	a.Title, a.F = "hunter", CGreen
	a.X, a.Y, a.HomeX, a.HomeY = x, y, x, y
	a.Gold = 200 + 30*a.Level
	a.Desc = fmt.Sprintf("The keeper of the hunting lodge of %s, with antlers on the wall and a ledger of seasons.", bd.Name)
	zn.Actors = append(zn.Actors, a)
}

// ---------------------------------------------------------------- kills

func hnKill(g *Game, d, src *Actor) {
	if g.P == nil || g.Cur == nil || g.W == nil || d == nil {
		return
	}
	s := hnSp(d)
	if s == nil {
		return
	}
	byP := src == g.P
	if !byP && !(src != nil && src.AI == "follower") {
		return
	}
	l := g.hnl()
	if j := l.Job; j != nil && j.Taken && !j.Done && d.Tpl == "bs_"+j.Sp && j.Got < j.Need {
		if bx, by, ok := g.jbBurgXY(j.Burg); ok && jbPar(bx-g.Cur.WX, by-g.Cur.WY) <= 5 {
			j.Got++
			g.Msg(fmt.Sprintf("{{G|Cull: %d of %d.}}", j.Got, j.Need))
		}
	}
	wound := false
	if w := l.Wounds[d.ID]; w != nil && len(w.Marks) >= 4 {
		wound = true
	}
	delete(l.Wounds, d.ID)
	if !hnHuntable(s) || !byP || g.Cur.Z != 0 {
		return
	}
	state := g.uwStateHere()
	clean := l.unaware[d.ID]
	kg := hnWeight(g, d, s)
	poach := 0
	if hnIsGame(s) {
		poach = g.hnCheckLaw(s, state)
	}
	car := &hnCarcass{Zone: g.Cur.Key, X: d.X, Y: d.Y, Sp: s.ID, Kg: kg, Clean: clean, Wound: wound, Poach: poach, Day: g.Day(), Turn: g.Turn, Realm: state}
	l.Pending = append(l.Pending, car)
	if len(l.Pending) > 6 {
		l.Pending = l.Pending[1:]
	}
	l.Taken++
	e := l.Journal[s.ID]
	if e == nil {
		e = &hnEntry{First: g.Day() + 1, Place: g.Cur.Name}
		l.Journal[s.ID] = e
		g.Msg(fmt.Sprintf("{{c|First %s for your journal.}}", s.Display()))
		if s.Rarity >= 4 || s.Extinct() {
			g.deed("event", fmt.Sprintf("The wanderer %s took a %s, a rare beast.", g.P.Name, s.Display()), 0, 0)
		}
	}
	e.Taken++
	if kg > e.BestKg {
		e.BestKg = kg
	}
	if clean {
		e.Clean++
		l.Clean++
	}
	if kg > l.Heavy.Kg {
		l.Heavy = hnRecord{SP: s.ID, Kg: kg, D: g.Day() + 1}
		if l.Taken > 1 {
			g.Msg(fmt.Sprintf("{{Y|A record: %d kg, your heaviest.}}", kg))
		}
	}
	if s.Rarity > l.Rare.R || s.Rarity == l.Rare.R && kg > l.Rare.Kg {
		l.Rare = hnRecord{SP: s.ID, Kg: kg, R: s.Rarity, D: g.Day() + 1}
	}
	note := fmt.Sprintf("{{c|%s, about %d kg.}}", strings.ToUpper(s.Display()[:1])+s.Display()[1:], kg)
	if clean {
		note += " {{G|A clean shot: it never knew you were there.}}"
	}
	g.Msg(note + " {{K|Field dress it before you take anything (palette: Field dress a kill).}}")
	g.hnXP(1 + s.Size + map[bool]int{true: 2, false: 0}[clean])
	if hnDanger(s) {
		g.hnXP(2)
	}
}

// hnCheckLaw applies the realm's game law to a kill and returns the poach mark (state + 1) or 0.
func (g *Game) hnCheckLaw(s *bsSp, state int) int {
	if state <= 0 {
		return 0
	}
	l := g.hnl()
	season := SeasonOfDay(g.Day())
	key := g.hnBagKey(state, s)
	l.Bag[key]++
	var why []string
	fine := 0
	if s.Rarity >= 4 {
		why = append(why, "a protected species")
		fine += 45
	} else if closed, _ := g.hnClosed(s, g.Cur.Biome, season); closed {
		why = append(why, "out of season")
		fine += 20
	}
	if l.Bag[key] > g.hnLimit(state, s) {
		why = append(why, "over the limit")
		fine += 12
	}
	if g.hnNeedsLicence(state) && !g.hnHasLicence(state, false) {
		why = append(why, "without a licence")
		fine += 15
	}
	if pre, ps, _ := g.hnPreserve(g.Cur); pre && !g.hnHasLicence(ps, true) {
		why = append(why, "inside a private preserve")
		fine += 30
	}
	if len(why) == 0 {
		return 0
	}
	l.Poached++
	text := strings.Join(why, ", ")
	if g.hnWitness(state) != nil {
		l.Fines += fine
		g.uwAddBounty(state, fine, " (poaching: "+text+")")
	} else {
		g.Msg("{{y|That was poaching (" + text + "). No one saw it, but the carcass will mark you if anyone looks.}}")
	}
	return state + 1
}

// ---------------------------------------------------------------- tracking

func (g *Game) hnCanRead() bool {
	return g.P != nil && g.Cur != nil && !g.OnWorld && g.Cur.Kind != "town" && outdoors(g.Cur)
}

func hnDistWord(d int) string {
	switch {
	case d < 6:
		return "right at hand"
	case d < 20:
		return "close"
	case d < 40:
		return "a fair way off"
	}
	return "far off"
}

func (g *Game) hnLikely(n int) []*bsSp {
	cell := 0
	if g.W.In(g.Cur.WX, g.Cur.WY) {
		cell = g.W.Biome[g.W.Idx(g.Cur.WX, g.Cur.WY)]
	}
	var c []*bsSp
	for _, s := range bsAll {
		if s.Era != "living" || !hnHuntable(s) || s.Sea != 0 || s.Mtn || s.Ice {
			continue
		}
		for _, b := range s.Biomes {
			if b == cell {
				c = append(c, s)
				break
			}
		}
	}
	sort.Slice(c, func(i, j int) bool {
		hi := HashStr(fmt.Sprintf("%s:hn:likely:%d:%d:%s", g.Seed, g.Cur.WX, g.Cur.WY, c[i].ID))
		hj := HashStr(fmt.Sprintf("%s:hn:likely:%d:%d:%s", g.Seed, g.Cur.WX, g.Cur.WY, c[j].ID))
		return hi < hj
	})
	if len(c) > n {
		c = c[:n]
	}
	return c
}

func (g *Game) hnRead(bonus int) {
	if !g.hnCanRead() {
		g.Msg("There is nothing to read here.")
		return
	}
	l := g.hnl()
	rank := g.hn().Rank
	g.spendTurns(5)
	roll := g.R.Dice(2, 6) + g.checkMod("survival") + rank + bonus
	margin := roll - 8
	var hints []string
	if margin < 0 {
		hints = append(hints, "The ground tells you nothing. Too many overlapping marks, or you rushed.")
		g.hnXP(1)
	} else {
		type cand struct {
			a *Actor
			s *bsSp
			d int
		}
		var cs []cand
		for _, a := range g.Cur.Actors {
			s := hnSp(a)
			if s != nil && a.HP > 0 && hnHuntable(s) && a.AI != "follower" {
				cs = append(cs, cand{a, s, dist(a, g.P)})
			}
		}
		sort.Slice(cs, func(i, j int) bool { return cs[i].d < cs[j].d })
		n := mini(len(cs), 1+margin/3)
		for i := 0; i < n; i++ {
			c := cs[i]
			signs := hnSigns[c.s.Group]
			if len(signs) == 0 {
				signs = hnSigns[""]
			}
			sign := signs[int(HashStr(fmt.Sprintf("%s:hn:sign:%d", g.Seed, c.a.ID))%uint64(len(signs)))]
			who := "something " + hnSizeWord[clampi(c.s.Size, 1, 7)]
			if margin >= 2 {
				who = c.s.Display()
			}
			dir := jbDirName(c.a.X-g.P.X, c.a.Y-g.P.Y)
			line := fmt.Sprintf("%s of %s lead %s, %s.", strings.ToUpper(sign[:1])+sign[1:], who, dir, hnDistWord(c.d))
			if margin >= 5 && hnDanger(c.s) {
				line += " Dangerous game."
			}
			hints = append(hints, line)
		}
		for id, w := range l.Wounds {
			if w.Zone != g.Cur.Key {
				continue
			}
			s := bsByID[w.Sp]
			if s == nil {
				continue
			}
			var tgt *Actor
			for _, a := range g.Cur.Actors {
				if a.ID == id && a.HP > 0 {
					tgt = a
				}
			}
			if tgt != nil && margin >= 1 {
				hints = append(hints, fmt.Sprintf("Blood spots from a wounded %s lead %s, %s. %d marks on the trail.", s.Display(), jbDirName(tgt.X-g.P.X, tgt.Y-g.P.Y), hnDistWord(dist(tgt, g.P)), len(w.Marks)))
				l.Trails++
			}
		}
		// neighbouring loaded zones
		R := clampi(1+rank/2+margin/4, 1, 3)
		var bz *Zone
		bd, cnt := 1<<30, 0
		for _, zn := range g.Zones {
			if zn == g.Cur || zn.Z != 0 || zn.Kind != "surface" || abs(zn.WX-g.Cur.WX) > R || abs(zn.WY-g.Cur.WY) > R {
				continue
			}
			k := 0
			for _, a := range zn.Actors {
				if s := hnSp(a); s != nil && a.HP > 0 && hnHuntable(s) {
					k++
				}
			}
			if k == 0 {
				continue
			}
			d := (zn.WX-g.Cur.WX)*(zn.WX-g.Cur.WX) + (zn.WY-g.Cur.WY)*(zn.WY-g.Cur.WY)
			if d < bd || d == bd && zn.Key < bz.Key {
				bz, bd, cnt = zn, d, k
			}
		}
		if bz != nil && (bz.WX != g.Cur.WX || bz.WY != g.Cur.WY) {
			hints = append(hints, fmt.Sprintf("Game is moving in the country to the %s, about %d parasang(s) off (%d animals seen in the signs).", jbDirName(bz.WX-g.Cur.WX, bz.WY-g.Cur.WY), jbPar(bz.WX-g.Cur.WX, bz.WY-g.Cur.WY), cnt))
			if margin >= 3 {
				g.ceSetQuestTarget(bz.WX, bz.WY)
				hints = append(hints, "You mark it on your compass.")
			}
		}
		if len(hints) == 0 {
			if ls := g.hnLikely(3); len(ls) > 0 {
				var nm []string
				for _, s := range ls {
					nm = append(nm, s.Display())
				}
				hints = append(hints, "Old marks only. This country holds "+strings.Join(nm, ", ")+", but nothing is close.")
			} else {
				hints = append(hints, "Old marks only. Nothing is close.")
			}
		}
		g.hnXP(2 + margin/2)
	}
	l.Hints = hints
	for _, h := range hints {
		g.Msg("{{c|" + h + "}}")
	}
}

// ---------------------------------------------------------------- field dressing

var hnParts = [4]string{"Skin it and take the hide", "Butcher the meat", "Take the bones, horn and tusk", "Mount a trophy"}

func (g *Game) hnNearCarcass() *hnCarcass {
	if g.P == nil || g.Cur == nil {
		return nil
	}
	for _, c := range g.hnl().Pending {
		if c.Zone == g.Cur.Key && maxi(abs(c.X-g.P.X), abs(c.Y-g.P.Y)) <= 2 && c.Done < 15 {
			return c
		}
	}
	return nil
}

func hnTrophyOK(s *bsSp) bool {
	return s.Size >= 3 || s.Rarity >= 2 || s.Has("ivory") || s.Attack == "antlers" || s.Attack == "horns" || s.Attack == "horn"
}

func (g *Game) hnDressOpen() {
	c := g.hnNearCarcass()
	if c == nil {
		g.Msg("None of your kills lies close by.")
		return
	}
	l := g.hnl()
	l.dress, l.dsel, l.dmsg = c, 0, ""
	g.Mode = "hnd"
}

func hnPartOf(it *Item, s *bsSp) int {
	n := strings.ToLower(it.Name)
	sfx := func(ws ...string) bool {
		for _, w := range ws {
			if strings.HasSuffix(n, w) {
				return true
			}
		}
		return false
	}
	switch {
	case it.Kind == "ingredient" && strings.HasPrefix(n, strings.ToLower(s.Display())) && sfx(" meat", " fillet", " flesh", " grubs"):
		return 1
	case it.Kind == "trophy" && it.Effect == "bsp:"+s.ID:
		switch {
		case sfx(" hide", " pelt", " feathers", " scales", " shell", " carapace"):
			return 0
		case sfx(" bones", " horns", " antlers", " horn", " ivory", " sinew", " fat", " blubber"):
			return 2
		}
	}
	return -1
}

func (g *Game) hnDoPart(c *hnCarcass, part int) string {
	s := bsByID[c.Sp]
	if s == nil {
		return "It is too far gone."
	}
	l := g.hnl()
	if c.Done&(1<<uint(part)) != 0 {
		return "That is done."
	}
	if part == 3 && !hnTrophyOK(s) {
		return "Not worth mounting."
	}
	if g.wkHostileSeen() {
		return "Not with enemies in sight."
	}
	g.spendTurns(6 + 2*s.Size)
	dc := []int{9 + s.Size, 7 + s.Size/2, 8, 10 + s.Rarity}[part]
	mod := g.checkMod("survival") + g.hn().Rank + map[bool]int{true: 2, false: 0}[c.Clean]
	if c.Wound && part == 0 {
		mod--
	}
	roll := g.R.Dice(2, 6) + mod
	margin := roll - dc
	grade := 0
	switch {
	case margin >= 8:
		grade = 3
	case margin >= 4:
		grade = 2
	case margin >= 0:
		grade = 1
	}
	c.Done |= 1 << uint(part)
	touched := 0
	if part < 3 {
		for _, it := range g.Cur.Items {
			if it.X != c.X || it.Y != c.Y || hnPartOf(it, s) != part {
				continue
			}
			touched++
			if it.Extra == nil {
				it.Extra = map[string]int{}
			}
			it.Extra["hnq"] = grade + 1
			if c.Poach > 0 {
				it.Extra["hnp"] = c.Poach
			}
			it.Value = maxi(1, it.Value*hnGradeValue[grade]/100)
			if part == 1 {
				switch grade {
				case 0:
					it.Count = maxi(1, it.Count/2)
				case 2:
					it.Count++
				case 3:
					it.Count += it.Count/2 + 1
				}
			}
			if grade != 1 && !strings.HasPrefix(it.Name, hnGradeWord[grade]+" ") {
				it.Name = hnGradeWord[grade] + " " + it.Name
			}
		}
		if touched == 0 {
			return fmt.Sprintf("%s: there is nothing left of that to work. (%d vs %d)", hnParts[part], roll, dc)
		}
	} else {
		who := s.Display()
		it := &Item{ID: g.newID(), Kind: "trophy", Name: hnMountWord[grade] + " " + who + " trophy", Glyph: '&', F: CYellow, Weight: 4, Effect: "hnt:" + s.ID,
			Extra: map[string]int{"bctroph": 1, "hnq": grade + 1, "hnkg": c.Kg}}
		v := 30 + 18*s.Size*s.Size/2 + 35*s.Rarity
		if hnDanger(s) {
			v = v * 3 / 2
		}
		it.Value = maxi(5, v*hnGradeValue[grade]/100)
		if c.Poach > 0 {
			it.Extra["hnp"] = c.Poach
		}
		how := map[bool]string{true: "a clean shot", false: "taken in the field"}[c.Clean]
		it.Lore = fmt.Sprintf("A mounted %s, %s, about %d kg: %s by %s on day %d of year %d at %s.%s", who, hnMountWord[grade], c.Kg, how, g.P.Name, c.Day+1, g.W.Year+c.Day/112, g.Cur.Name,
			map[bool]string{true: " It carries a poacher's mark.", false: ""}[c.Poach > 0])
		g.give(it)
		l.Trophies++
		if e := l.Journal[s.ID]; e != nil && grade+1 > e.BestQ {
			e.BestQ = grade + 1
		}
	}
	g.hnXP(1 + s.Size/2 + grade)
	g.gainXP(1 + s.Size)
	if c.Done >= 15 || (!hnTrophyOK(s) && c.Done >= 7) {
		for i, p := range l.Pending {
			if p == c {
				l.Pending = append(l.Pending[:i], l.Pending[i+1:]...)
				break
			}
		}
	}
	return fmt.Sprintf("%s: %s work. {{K|(%d vs %d)}}", hnParts[part], hnGradeWord[grade], roll, dc)
}

func hnDressKeys(g *Game, k string) {
	l := g.hnl()
	c := l.dress
	if c == nil || k == "Escape" || k == "q" {
		g.Mode = "main"
		return
	}
	switch k {
	case "ArrowUp", "N8", "k":
		l.dsel = maxi(0, l.dsel-1)
	case "ArrowDown", "N2", "j":
		l.dsel = mini(3, l.dsel+1)
	case "Enter":
		l.dmsg = g.hnDoPart(c, l.dsel)
	case "a", "b", "c", "d":
		l.dsel = int(k[0] - 'a')
		l.dmsg = g.hnDoPart(c, l.dsel)
	}
	still := false
	for _, p := range l.Pending {
		if p == c {
			still = true
		}
	}
	if !still {
		g.Msg("{{G|The kill is dressed.}}")
		g.Mode = "main"
	}
}

func (g *Game) hnDressDraw() {
	sc := g.sc
	l := g.hnl()
	c := l.dress
	if c == nil {
		return
	}
	s := bsByID[c.Sp]
	title := "Field dressing"
	if s != nil {
		title += ": " + s.Display()
	}
	x, y, w, _ := g.panel(title)
	line := func(t string) {
		for _, ln := range Wrap(t, w) {
			sc.TextB(x, y, ln, CGray, CBgDark, x+w)
			y++
		}
	}
	line(fmt.Sprintf("About %d kg. %s %s", c.Kg, map[bool]string{true: "{{G|A clean shot: easier work, a clean hide.}}", false: ""}[c.Clean],
		map[bool]string{true: "{{y|It was wounded first: the hide is marked.}}", false: ""}[c.Wound]))
	if c.Poach > 0 {
		line("{{R|Poached: whatever you take carries a poacher's mark. A lodge will not buy it.}}")
	}
	line("{{K|Letters or Up/Down and Enter do a part. Each takes time and a skill check. Esc leaves; the carcass stays for a while.}}")
	y++
	for i, p := range hnParts {
		mark := "  "
		if i == l.dsel {
			mark = "> "
		}
		state := ""
		switch {
		case c.Done&(1<<uint(i)) != 0:
			state = " {{K|(done)}}"
		case i == 3 && s != nil && !hnTrophyOK(s):
			state = " {{K|(not worth it)}}"
		}
		sc.TextB(x, y, fmt.Sprintf("%s{{W|%c}}) %s%s", mark, 'a'+i, p, state), CGray, CBgDark, x+w)
		y++
	}
	y++
	if l.dmsg != "" {
		line(l.dmsg)
	}
}

// ---------------------------------------------------------------- lodge, licences, selling, cull

func hnIsWoods(a *Actor) bool {
	if a.Role == "huntsman" || a.Role == "warden" {
		return true
	}
	t := strings.ToLower(a.Title)
	for _, w := range []string{"hunter", "forester", "ranger", "trapper", "woodcutter", "warden"} {
		if strings.Contains(t, w) {
			return true
		}
	}
	return false
}

func hnTalk(g *Game, a *Actor) []Option {
	if g.W == nil || g.Cur == nil || g.P == nil || a == nil || a.HP <= 0 || a.Angry || a.AI == "hostile" || a.Tpl != "npc" || a.AI == "follower" {
		return nil
	}
	var out []Option
	if a.Role == "huntsman" && g.Cur.Kind == "town" {
		out = append(out, Option{"{{O|Ask the huntsman about licences, game and the lodge.}}", func() { g.hnLodge(a) }})
	}
	if hnIsWoods(a) {
		out = append(out, Option{"{{O|Ask where the game is moving.}}", func() {
			if g.Cur.Kind == "town" || !outdoors(g.Cur) {
				g.talkMenu("\"I read the ground, not cobbles. Find me outside the walls and I will show you what I see.\"")
				return
			}
			g.Mode = "main"
			g.Talk = nil
			g.hnRead(2)
			g.hnOpen(0)
		}})
	}
	return out
}

func (g *Game) hnMenu(text string, opts []Option, a *Actor) {
	g.Popup = Wrap(text, 100)
	g.Opts = append(opts, Option{"Back.", func() { g.hnLodge(a) }}, Option{"Leave it.", func() { g.talkMenu("\"Anything else?\"") }})
	g.Sel = 0
}

func hnSellable(it *Item) bool {
	n := strings.ToLower(it.Name)
	if it.Kind == "ingredient" {
		for _, w := range []string{" meat", " fillet", " flesh", " grubs"} {
			if strings.HasSuffix(n, w) {
				return it.Extra != nil && it.Extra["hnq"] > 0
			}
		}
		return false
	}
	if it.Kind == "trophy" && strings.HasPrefix(it.Effect, "bsp:") {
		for _, w := range []string{" hide", " pelt", " feathers", " scales", " shell", " carapace", " bones", " horns", " antlers", " horn", " ivory"} {
			if strings.HasSuffix(n, w) {
				return true
			}
		}
	}
	return false
}

func hnIsMount(it *Item) bool { return it.Kind == "trophy" && strings.HasPrefix(it.Effect, "hnt:") }

func (g *Game) hnSellList(mounts bool) (items []*Item, total int, poached int) {
	for _, it := range g.P.Inv {
		if mounts && !hnIsMount(it) || !mounts && !hnSellable(it) {
			continue
		}
		if it.Extra != nil && it.Extra["hnp"] > 0 {
			poached++
			continue
		}
		items = append(items, it)
		total += it.Value * wkUnitsHn(it)
	}
	total = total * (100 + 5*g.hn().Rank + 5*clampi(g.Skills["persuasion"], 0, 3)) / 100
	return
}

func wkUnitsHn(it *Item) int { return wkUnits(it) }

func (g *Game) hnLodge(a *Actor) {
	l := g.hnl()
	burg := g.Cur.Burg
	state := g.uwStateHere()
	var opts []Option
	h := int(HashStr(fmt.Sprintf("%s:hn:greet:%d", g.Seed, g.Day()+burg)) % uint64(len(hnLodgeGreet)))
	head := hnLodgeGreet[h]
	if state > 0 {
		head += fmt.Sprintf("\n\nRealm: %s. %s", g.uwRealm(state), g.hnLawLine(state))
		price := 30 + 8*g.P.Level
		if g.hnHasLicence(state, false) {
			head += "\nYou hold a licence for this season."
		} else {
			opts = append(opts, Option{fmt.Sprintf("Buy a hunting licence for this season {{K|(%s)}}", g.cuFormat(price)), func() {
				if !g.cuSpend(price) {
					g.hnMenu("\"Come back with the coin.\"", nil, a)
					return
				}
				g.give(g.hnLicence(state, false))
				g.hnMenu("\"Stamped and signed. Keep it dry and keep to the limits.\"", nil, a)
			}})
		}
		if st := g.W.stateByID[state]; st != nil && hnOwner[st.Ideology] != "" && !g.hnHasLicence(state, true) {
			pp := price * 3
			opts = append(opts, Option{fmt.Sprintf("Lease the preserves of %s for this season {{K|(%s, includes the licence)}}", hnOwner[st.Ideology], g.cuFormat(pp)), func() {
				if !g.cuSpend(pp) {
					g.hnMenu("\"The lease is dear. Come back with the coin.\"", nil, a)
					return
				}
				g.give(g.hnLicence(state, true))
				g.hnMenu("\"The wardens will know the seal. Do not shoot what is not on the list.\"", nil, a)
			}})
		}
	}
	items, total, poached := g.hnSellList(false)
	if len(items) > 0 {
		opts = append(opts, Option{fmt.Sprintf("{{W|Sell meat and hides.}} {{K|(%d lots, %s)}}", len(items), g.cuFormat(total)), func() {
			got, _, _ := g.hnSellList(false)
			_, tot, _ := g.hnSellList(false)
			for _, it := range got {
				g.removeInv(it)
			}
			l.Earned += tot
			g.hnXP(1)
			g.hnMenu(hnLodgeThanks[int(HashStr(fmt.Sprintf("%d", g.Turn))%uint64(len(hnLodgeThanks)))]+" You are paid "+g.cuEarn(tot)+".", nil, a)
		}})
	}
	mi, mt, mp := g.hnSellList(true)
	poached += mp
	if len(mi) > 0 {
		opts = append(opts, Option{fmt.Sprintf("{{W|Sell trophies.}} {{K|(%d, %s)}}", len(mi), g.cuFormat(mt)), func() {
			got, tot, _ := g.hnSellList(true)
			for _, it := range got {
				g.removeInv(it)
			}
			l.Earned += tot
			g.hnMenu("\"These will hang well in the lodge.\" You are paid "+g.cuEarn(tot)+".", nil, a)
		}})
	}
	if poached > 0 {
		head += "\n\n" + hnRefusePoach
	}
	if c := g.hnCullOffer(burg); c != nil {
		sp := bsByID[c.Sp]
		name := "pests"
		if sp != nil {
			name = sp.Display()
		}
		switch {
		case c.Done:
			head += "\n\nThis week's cull is done."
		case l.Job != nil && l.Job.Taken && !l.Job.Done && l.Job.Burg == burg && l.Job.Got >= l.Job.Need:
			opts = append(opts, Option{fmt.Sprintf("{{G|Report the cull}} {{K|(%s)}}", g.cuFormat(l.Job.Reward)), func() {
				j := l.Job
				j.Done = true
				c.Done = true
				l.Culls++
				l.Earned += j.Reward
				if st := g.uwStateHere(); st > 0 {
					g.AddRep(uwFac(st), 4, " for keeping the herds safe")
				}
				g.hnXP(4)
				g.gainXP(10 + 4*j.Need)
				g.hnMenu("\"Counted and paid.\" You are paid "+g.cuEarn(j.Reward)+".", nil, a)
			}})
		case l.Job != nil && l.Job.Taken && !l.Job.Done && l.Job.Burg == burg:
			head += fmt.Sprintf("\n\nCull in progress: %d of %d %s near %s.", l.Job.Got, l.Job.Need, name, g.jbBurgName(burg))
		default:
			head += fmt.Sprintf("\n\nCull posted this week: %d %s within five parasangs, %s.", c.Need, name, g.cuFormat(c.Reward))
			opts = append(opts, Option{"Take the cull job", func() {
				if l.Job != nil && l.Job.Taken && !l.Job.Done {
					g.hnMenu("\"You have a cull running already. Finish it or let it lapse.\"", nil, a)
					return
				}
				cp := *c
				cp.Taken = true
				l.Job = &cp
				g.hnMenu(fmt.Sprintf("\"%d %s, near here. Bring no proof; the herders count.\"", c.Need, name), nil, a)
			}})
		}
	}
	opts = append(opts, Option{"Ask about the law and the seasons", func() { g.hnMenu(strings.Join(g.hnLawLines(), "\n"), nil, a) }})
	g.hnMenu(head, opts, a)
}

func (g *Game) hnLicence(state int, pre bool) *Item {
	day := g.Day()
	exp := (day/SeasonDays + 1) * SeasonDays
	name := fmt.Sprintf("hunting licence of %s (%s, year %d)", g.uwRealm(state), SeasonNames[SeasonOfDay(day)], g.W.Year+day/112)
	if pre {
		name = fmt.Sprintf("preserve lease of %s (%s, year %d)", g.uwRealm(state), SeasonNames[SeasonOfDay(day)], g.W.Year+day/112)
	}
	it := &Item{ID: g.newID(), Kind: "trophy", Name: name, Glyph: '=', F: CYellow, Weight: 0, Value: 1, Effect: "hnlic", Ident: true,
		Extra: map[string]int{"hnlic": state + 1, "hnexp": exp}}
	if pre {
		it.Extra["hnpre"] = 1
	}
	it.Lore = "Valid in " + g.uwRealm(state) + " until the end of the season (day " + fmt.Sprint(exp) + ")."
	return it
}

func (g *Game) hnLawLine(state int) string {
	if state <= 0 {
		return "No one keeps the game here."
	}
	lic := "no licence is asked"
	if g.hnNeedsLicence(state) {
		lic = "a licence is needed"
	}
	return fmt.Sprintf("Here %s, and bag limits run from %d to %d a season by size.", lic, g.hnLimit(state, &bsSp{Size: 2}), g.hnLimit(state, &bsSp{Size: 6}))
}

func (g *Game) hnLawLines() []string {
	state := g.uwStateHere()
	out := []string{fmt.Sprintf("{{Y|Game law of %s}}", g.uwRealm(state)), g.hnLawLine(state)}
	if state > 0 {
		if pre, _, name := g.hnPreserve(g.Cur); pre {
			out = append(out, "You stand in a private preserve held for "+name+". A lease is needed.")
		}
		if cu := g.lxCultureAt(); cu != nil {
			if pk := cgPack(g.W, cu.ID); pk != nil && pk.Customs != nil && pk.Customs.Law != nil {
				for _, e := range pk.Customs.Law.Entries {
					if e.Crime == "trespass" {
						out = append(out, "The law on trespass: "+e.Punish+". "+e.Note)
					}
				}
			}
		}
		if g.hnHasLicence(state, false) {
			out = append(out, "You hold a licence for this season.")
		}
	}
	out = append(out, "", "{{Y|Seasons here}}")
	season := SeasonOfDay(g.Day())
	for _, s := range g.hnLikely(8) {
		closed, why := g.hnClosed(s, g.Cur.Biome, season)
		bag := ""
		if state > 0 {
			bag = fmt.Sprintf("  bag %d/%d", g.hnl().Bag[g.hnBagKey(state, s)], g.hnLimit(state, s))
		}
		if closed {
			out = append(out, fmt.Sprintf("  %-22s {{R|closed}} (%s)", s.Display(), why))
		} else {
			out = append(out, fmt.Sprintf("  %-22s {{G|open}}%s", s.Display(), bag))
		}
	}
	return out
}

// hnCullOffer is the burg's cull for this week, made from the pests of the land around it.
func (g *Game) hnCullOffer(burg int) *hnCull {
	l := g.hnl()
	week := g.Day() / 7
	if c := l.Cull[burg]; c != nil && c.Week == week {
		return c
	}
	bx, by, ok := g.jbBurgXY(burg)
	if !ok {
		return nil
	}
	cells := map[int]bool{}
	for dy := -3; dy <= 3; dy++ {
		for dx := -3; dx <= 3; dx++ {
			if g.W.In(bx+dx, by+dy) {
				cells[g.W.Biome[g.W.Idx(bx+dx, by+dy)]] = true
			}
		}
	}
	var pool []*bsSp
	for _, s := range bsAll {
		if s.Era != "living" || !s.Hostile() || s.Size > 4 || s.Size < 1 || s.Aquatic() || s.Sea != 0 || s.Rarity >= 4 {
			continue
		}
		for _, b := range s.Biomes {
			if cells[b] {
				pool = append(pool, s)
				break
			}
		}
	}
	if len(pool) == 0 {
		return nil
	}
	h := HashStr(fmt.Sprintf("%s:hn:cull:%d:%d", g.Seed, burg, week))
	sp := pool[int(h%uint64(len(pool)))]
	need := 3 + int(h/7%4)
	c := &hnCull{Burg: burg, Week: week, Sp: sp.ID, Need: need, Reward: 14 + 7*need + 5*sp.Size*need}
	l.Cull[burg] = c
	return c
}

// ---------------------------------------------------------------- journal screen

var hnTabs = []string{"Signs", "Journal", "Records", "Law and seasons"}

func (g *Game) hnOpen(tab int) {
	l := g.hnl()
	l.tab, l.sel, l.msg = tab, 0, ""
	g.Mode = "hn"
}

func (g *Game) hnLines(tab int) []string {
	l := g.hnl()
	var out []string
	switch tab {
	case 0:
		h := g.hn()
		next := "top rank"
		if h.Rank < 5 {
			next = fmt.Sprintf("practice %d of %d", h.XP, hnXPAt[h.Rank])
		}
		out = append(out, fmt.Sprintf("Tracking: {{Y|%s}} (%s).", hnRankTitle[h.Rank], next), "")
		if g.Cur != nil && g.Cur.Kind == "surface" && outdoors(g.Cur) {
			seen := false
			for _, a := range g.Cur.Actors {
				s := hnSp(a)
				if s == nil || a.HP <= 0 || !hnHuntable(s) || a.AI == "follower" || !g.visible(a.X, a.Y) || dist(a, g.P) > 24 {
					continue
				}
				un, why := g.hnStalk(a)
				seen = true
				out = append(out, fmt.Sprintf("In sight: %s, %d away, %s. %s", s.Display(), dist(a, g.P), jbDirName(a.X-g.P.X, a.Y-g.P.Y), map[bool]string{true: "{{G|It has not noticed you.}}", false: "{{y|It may have noticed you.}}"}[un]), "  "+why)
			}
			if !seen {
				out = append(out, "No game in sight.")
			}
			out = append(out, "", "{{K|Enter reads the signs here (five minutes, a Survival check).}}")
		} else {
			out = append(out, "{{K|Signs can only be read in the wild, out of doors.}}")
		}
		out = append(out, "")
		if len(l.Hints) == 0 {
			out = append(out, "No signs read yet.")
		} else {
			out = append(out, "{{Y|What the ground said}}")
			for _, h := range l.Hints {
				out = append(out, "  "+h)
			}
		}
	case 1:
		var ids []string
		for id := range l.Journal {
			ids = append(ids, id)
		}
		sort.Strings(ids)
		sort.SliceStable(ids, func(i, j int) bool { return l.Journal[ids[i]].Taken > l.Journal[ids[j]].Taken })
		if len(ids) == 0 {
			out = append(out, "Nothing taken yet. Hunt game in the wild; every first goes in here.")
		}
		for _, id := range ids {
			e, s := l.Journal[id], bsByID[id]
			if s == nil {
				continue
			}
			q := ""
			if e.BestQ > 0 {
				q = ", best trophy " + hnMountWord[clampi(e.BestQ-1, 0, 3)]
			}
			out = append(out, fmt.Sprintf("%-24s x%-3d best %d kg, %d clean%s", s.Display(), e.Taken, e.BestKg, e.Clean, q),
				fmt.Sprintf("    {{K|first on day %d at %s}}", e.First, nonEmpty(e.Place, "somewhere")))
		}
	case 2:
		out = append(out, fmt.Sprintf("Animals taken: %d, species: %d, clean shots: %d, trophies mounted: %d.", l.Taken, len(l.Journal), l.Clean, l.Trophies),
			fmt.Sprintf("Poaching marks: %d, fines paid in bounty: %d, cull jobs done: %d, blood trails read: %d.", l.Poached, l.Fines, l.Culls, l.Trails),
			fmt.Sprintf("Earned from game goods: %s.", g.cuFormat(l.Earned)), "")
		if s := bsByID[l.Heavy.SP]; s != nil {
			out = append(out, fmt.Sprintf("{{Y|Heaviest}}: %s, %d kg (day %d).", s.Display(), l.Heavy.Kg, l.Heavy.D))
		} else {
			out = append(out, "{{Y|Heaviest}}: none yet.")
		}
		if s := bsByID[l.Rare.SP]; s != nil {
			out = append(out, fmt.Sprintf("{{Y|Rarest}}: %s (rarity %d, day %d).", s.Display(), l.Rare.R, l.Rare.D))
		} else {
			out = append(out, "{{Y|Rarest}}: none yet.")
		}
		out = append(out, "")
		n := 0
		for _, c := range l.Pending {
			if s := bsByID[c.Sp]; s != nil {
				n++
				out = append(out, fmt.Sprintf("Undressed kill: %s, %d kg%s.", s.Display(), c.Kg, map[bool]string{true: " (poached)", false: ""}[c.Poach > 0]))
			}
		}
		if n == 0 {
			out = append(out, "No undressed kills.")
		}
	case 3:
		out = g.hnLawLines()
		out = append(out, "")
		if j := l.Job; j != nil && j.Taken && !j.Done {
			if s := bsByID[j.Sp]; s != nil {
				out = append(out, fmt.Sprintf("{{Y|Cull job}}: %d of %d %s near %s.", j.Got, j.Need, s.Display(), g.jbBurgName(j.Burg)))
			}
		} else {
			out = append(out, "{{K|No cull job. Hunting lodges (a huntsman in towns near wild country) post one each week.}}")
		}
	}
	return out
}

func hnKeys(g *Game, k string) {
	l := g.hnl()
	switch k {
	case "Escape", "q":
		g.Mode = "main"
	case "Tab", "ArrowRight":
		l.tab, l.sel = (l.tab+1)%len(hnTabs), 0
	case "ArrowLeft":
		l.tab, l.sel = (l.tab+len(hnTabs)-1)%len(hnTabs), 0
	case "1", "2", "3", "4":
		l.tab, l.sel = int(k[0]-'1'), 0
	case "ArrowUp", "N8", "k":
		l.sel = maxi(0, l.sel-1)
	case "ArrowDown", "N2", "j":
		l.sel++
	case "PageUp":
		l.sel = maxi(0, l.sel-10)
	case "PageDown":
		l.sel += 10
	case "Enter":
		if l.tab == 0 && g.hnCanReadFromScreen() {
			g.hnRead(0)
		}
	}
	if n := len(g.hnLines(l.tab)); l.sel > n {
		l.sel = maxi(0, n-1)
	}
}

func (g *Game) hnCanReadFromScreen() bool {
	return g.P != nil && g.Cur != nil && g.Cur.Kind == "surface" && outdoors(g.Cur)
}

func (g *Game) hnDraw() {
	sc := g.sc
	l := g.hnl()
	x, y, w, h := g.panel("Hunter's journal")
	bottom := 2 + 1 + h - 1
	var tb strings.Builder
	for i, t := range hnTabs {
		c := "K"
		if i == l.tab {
			c = "W"
		}
		fmt.Fprintf(&tb, "{{%s|[ %d %s ]}}  ", c, i+1, t)
	}
	sc.TextB(x, y, tb.String(), CGray, CBgDark, x+w)
	y++
	sc.TextB(x, y, "{{K|Up/Down scroll  Tab or 1-4 switch  Enter read the signs (Signs)  Esc close}}", CGray, CBgDark, x+w)
	y += 2
	var rows []string
	for _, ln := range g.hnLines(l.tab) {
		rows = append(rows, Wrap(ln, w)...)
	}
	if l.sel > len(rows) {
		l.sel = maxi(0, len(rows)-1)
	}
	for i := l.sel; i < len(rows) && y < bottom; i++ {
		sc.TextB(x, y, rows[i], CGray, CBgDark, x+w)
		y++
	}
}
