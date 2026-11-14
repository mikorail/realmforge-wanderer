package engine

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
)

// Disaster scenes and migrations (prefix ms). Part 1: a zone struck by a Realmforge disaster or outbreak
// (hazards.go index, read only) may, on the daily clock (side rng seed:ms:roll:zone:day), host a short
// rescue scene: dig trapped people out of rubble, hold a post against wreckers, fight a spreading fire,
// lead plague patients / shore survivors / evacuees to the one waiting at the goal. Reward: reputation,
// gold, XP, a deed, a grateful rescued person who remembers (lvPerson.remember) and news; failure kills
// people of the town (lvSt) and posts news. Part 2: refugee columns and pilgrimages walking between towns
// on the day clock (a leader who talks, visible as actors in wild zones along the line between the two
// towns; a pilgrimage you join blesses you at the shrine town), and beast territory fights in the wild
// (watch, help one side, or exploit the weakened survivors). State g.ext("ms"), planet-local.
// Screen: mode "ms", ability ms_events.

type msScene struct {
	ID      int           `json:"id"`
	Class   string        `json:"c"`
	Hz      string        `json:"h,omitempty"`
	Zone    string        `json:"z"`
	Place   string        `json:"pl"`
	Burg    int           `json:"b"`
	Start   int           `json:"st"`
	Due     int           `json:"du"`
	Total   int           `json:"to"`
	Got     int           `json:"go"`
	Lost    int           `json:"lo"`
	State   string        `json:"s"` // active won lost
	Ended   int           `json:"en,omitempty"`
	Waves   int           `json:"wv,omitempty"`
	Spawned int           `json:"sp,omitempty"`
	Next    int           `json:"nx,omitempty"`
	Burned  int           `json:"bu,omitempty"`
	Warn    int           `json:"wn,omitempty"`
	Gave    bool          `json:"gv,omitempty"`
	Rub     map[int][]int `json:"rb,omitempty"` // victim actor id -> zone cells of rubble still on them
	Age     map[int]int   `json:"ag,omitempty"` // fire actor id -> turn lit
}

type msCol struct {
	ID       int    `json:"id"`
	Kind     string `json:"k"` // refugee pilgrim
	From     int    `json:"f"`
	To       int    `json:"t"`
	Start    int    `json:"st"` // day
	Days     int    `json:"d"`
	Size     int    `json:"n"`
	Cause    string `json:"ca,omitempty"`
	Leader   string `json:"ld,omitempty"`
	Known    bool   `json:"kn,omitempty"`
	Fed      bool   `json:"fe,omitempty"`
	Escorted bool   `json:"es,omitempty"`
	Joined   bool   `json:"jo,omitempty"`
	Blessed  bool   `json:"bl,omitempty"`
	Done     bool   `json:"dn,omitempty"`
	ShownDay int    `json:"sd,omitempty"`
	ShownZ   string `json:"sz,omitempty"`
}

type msFight struct {
	ID     int    `json:"id"`
	Zone   string `json:"z"`
	A      string `json:"a"`
	B      string `json:"b"`
	Day    int    `json:"d"`
	Over   bool   `json:"o,omitempty"`
	Helped string `json:"h,omitempty"`
	Watch  bool   `json:"w,omitempty"`
}

type msState struct {
	Init    bool           `json:"i,omitempty"`
	LastDay int            `json:"ld"`
	Next    int            `json:"n"`
	Scenes  []*msScene     `json:"sc,omitempty"`
	Cols    []*msCol       `json:"co,omitempty"`
	Fights  []*msFight     `json:"fi,omitempty"`
	Rolled  map[string]int `json:"ro,omitempty"`
	Log     []string       `json:"lg,omitempty"`
	Saved   int            `json:"sv,omitempty"`
	Lost    int            `json:"ls,omitempty"`
	Won     int            `json:"wn,omitempty"`
	Pilgrim int            `json:"pg,omitempty"`
	Escort  int            `json:"ec,omitempty"`
}

func init() {
	unLocalKeys = append(unLocalKeys, "ms")
	turnHooks = append(turnHooks, msTurn)
	enterHooks = append(enterHooks, msEnter)
	talkHooks = append(talkHooks, msTalk)
	killHooks = append(killHooks, msKill)
	sideHooks = append(sideHooks, msSide)
	modeKeys["ms"] = msKeys
	modeDraws["ms"] = func(g *Game) { g.msDraw() }
}

func msSt(g *Game) *msState {
	s := g.ext("ms", func() any { return &msState{} }).(*msState)
	if s.Rolled == nil {
		s.Rolled = map[string]int{}
	}
	return s
}

func (g *Game) msLog(s string) {
	st := msSt(g)
	st.Log = append(st.Log, fmt.Sprintf("Day %d: %s", DayOf(g.Turn)+1, s))
	if len(st.Log) > 14 {
		st.Log = st.Log[len(st.Log)-14:]
	}
}

func (g *Game) msNews(burg int, text string) {
	lvSt(g).addNews(lvDay(g), burg, "%s", text)
}

func msBurgName(g *Game, id int) string {
	if b := g.W.burgByID[id]; b != nil {
		return b.Name
	}
	return "a distant town"
}

// msTag reads a numbered flag ("ms12", "mc3", "mf5") off an actor.
func msTag(a *Actor, prefix string) (int, bool) {
	for k, v := range a.Flags {
		if !v || !strings.HasPrefix(k, prefix) || len(k) <= len(prefix) {
			continue
		}
		if n, err := strconv.Atoi(k[len(prefix):]); err == nil {
			return n, true
		}
	}
	return 0, false
}

func msSceneByID(g *Game, id int) *msScene {
	for _, s := range msSt(g).Scenes {
		if s.ID == id {
			return s
		}
	}
	return nil
}

func msSceneOf(g *Game, a *Actor) *msScene {
	if id, ok := msTag(a, "ms"); ok {
		return msSceneByID(g, id)
	}
	return nil
}

func msColByID(g *Game, id int) *msCol {
	for _, c := range msSt(g).Cols {
		if c.ID == id {
			return c
		}
	}
	return nil
}

func msIsVictim(role string) bool {
	return role == "ms_trapped" || role == "ms_patient" || role == "ms_survivor" || role == "ms_evac"
}

func (g *Game) msActiveHere() *msScene {
	if g.Cur == nil {
		return nil
	}
	for _, s := range msSt(g).Scenes {
		if s.State == "active" && s.Zone == g.Cur.Key {
			return s
		}
	}
	return nil
}

// ---- turn / day clock ----

func msTurn(g *Game) {
	if g.W == nil || g.Cur == nil || g.P == nil {
		return
	}
	st := msSt(g)
	d := DayOf(g.Turn)
	if !st.Init {
		st.Init = true
		st.LastDay = d
	}
	if d != st.LastDay {
		from := maxi(st.LastDay+1, d-11)
		for day := from; day <= d; day++ {
			msDayTick(g, day)
		}
		st.LastDay = d
		msLocal(g, g.Cur)
		msStaleHere(g)
	}
	zn := g.Cur
	if zn.Z != 0 {
		return
	}
	if s := g.msActiveHere(); s != nil {
		msSceneTick(g, zn, s)
	}
	msFightTick(g, zn)
	msColTick(g, zn)
}

func msDayTick(g *Game, day int) {
	st := msSt(g)
	st.Rolled = map[string]int{}
	msColGen(g, day)
	for _, c := range st.Cols {
		if !c.Done && day-c.Start > c.Days {
			msColSettle(g, c)
		}
	}
	// prune
	var keep []*msCol
	for _, c := range st.Cols {
		if !c.Done || day-c.Start < c.Days+16 {
			keep = append(keep, c)
		}
	}
	st.Cols = keep
	for _, s := range st.Scenes {
		if s.State == "active" && g.Turn > s.Due && (g.Cur == nil || g.Cur.Key != s.Zone) {
			msFinish(g, s, false, "the time ran out while you were away")
		}
	}
	if len(st.Scenes) > 14 {
		st.Scenes = st.Scenes[len(st.Scenes)-14:]
	}
	var fk []*msFight
	for _, f := range st.Fights {
		if day-f.Day < 6 {
			fk = append(fk, f)
		}
	}
	st.Fights = fk
}

func msEnter(g *Game, zn *Zone) {
	if g.W == nil {
		return
	}
	st := msSt(g)
	if !st.Init {
		st.Init = true
		st.LastDay = DayOf(g.Turn)
	}
	for _, s := range st.Scenes {
		if s.State == "active" && g.Turn > s.Due {
			if s.Zone == zn.Key {
				continue // resolved by the scene tick with the zone in view
			}
			msFinish(g, s, false, "the time ran out while you were away")
		}
	}
	msClean(g, zn)
	msLocal(g, zn)
	msBlessCheck(g, zn)
}

func msLocal(g *Game, zn *Zone) {
	if zn == nil || zn.Z != 0 || zn.Underwater || zn.Sky {
		return
	}
	msRoll(g, zn)
	msColShow(g, zn)
	msFightRoll(g, zn)
}

// msClean removes leftovers of finished scenes and of columns that moved on.
func msClean(g *Game, zn *Zone) {
	day := DayOf(g.Turn)
	p := g.W.Idx(zn.WX, zn.WY)
	for _, a := range append([]*Actor(nil), zn.Actors...) {
		if id, ok := msTag(a, "ms"); ok {
			s := msSceneByID(g, id)
			if s == nil || s.State != "active" {
				switch a.Role {
				case "ms_warden", "ms_healer", "ms_camp", "ms_marshal", "ms_fire":
					g.removeActor(zn, a)
				}
			}
		}
		if id, ok := msTag(a, "mc"); ok {
			c := msColByID(g, id)
			if c == nil || c.Done || c.ShownDay != day || c.ShownZ != zn.Key || g.msColPos(c, day-c.Start) != p {
				g.removeActor(zn, a)
			}
		}
	}
}

func msStaleHere(g *Game) {
	if g.Cur != nil {
		n := len(g.Cur.Actors)
		msClean(g, g.Cur)
		if len(g.Cur.Actors) < n {
			g.Msg("{{K|The column has moved on down the road.}}")
		}
	}
}

// ---- scene roll ----

func msEvent(g *Game, p int) (class, hzk string, chance float64) {
	if g.W.Hazards == nil {
		return
	}
	for _, e := range g.W.hzRecentAt(p) {
		c := msClassOf[e.Kind]
		if c == "" {
			continue
		}
		age := g.W.Year - e.Year
		ch := 0.6 * (1 - float64(age)/float64(hzMarkYears[e.Kind]+1))
		if ch < 0.15 {
			ch = 0.15
		}
		return c, e.Kind, ch
	}
	if ob := g.W.hzOutbreakAt(p); ob != nil {
		return "carry", "plague", 0.45
	}
	return
}

func msNearestBurg(g *Game, p int) *BurgData {
	var best *BurgData
	bd := 1 << 30
	for _, b := range g.ws().burgList(g) {
		if d := wsCheb(g.W, p, b.P); d < bd {
			best, bd = b, d
		}
	}
	return best
}

func msSpot(g *Game, zn *Zone, r *Rng, lo, hi int, pred func(x, y int) bool) (int, int, bool) {
	for pass := 0; pass < 2; pass++ {
		for i := 0; i < 400; i++ {
			x, y := r.Int(1, MW-2), r.Int(1, MH-2)
			d := maxi(abs(x-g.P.X), abs(y-g.P.Y))
			if d < lo || d > hi || (x == g.P.X && y == g.P.Y) {
				continue
			}
			t := zn.At(x, y)
			if !zn.Walkable(x, y) || zn.ActorAt(x, y) != nil || t.D().Water >= 2 || t == TLava || t == TVoid || t == TChasm {
				continue
			}
			if pred != nil && !pred(x, y) {
				continue
			}
			return x, y, true
		}
		lo, hi = 1, 80
	}
	return 0, 0, false
}

func msPerson(g *Game, zn *Zone, bd *BurgData, r *Rng, role, title string, s *msScene, adopt bool) *Actor {
	a := g.makeNPC("villager", bd, g.W.cultByID[bd.Culture], r)
	if a == nil {
		return nil
	}
	a.Role = "villager"
	a.Title = ""
	if adopt {
		lvAdopt(g, a, bd.ID, zn)
	}
	a.Role, a.Title = role, title
	a.AI = "town"
	a.Set("still")
	if s != nil {
		a.Set(fmt.Sprintf("ms%d", s.ID))
	}
	a.Desc = fmt.Sprintf("A %s from %s.", title, bd.Name)
	return a
}

func msPlace(zn *Zone, a *Actor, x, y int) {
	a.X, a.Y, a.HomeX, a.HomeY = x, y, x, y
	zn.Actors = append(zn.Actors, a)
}

func msRoll(g *Game, zn *Zone) {
	if g.W.Hazards == nil || zn.Z != 0 {
		return
	}
	if zn.Kind != "surface" && zn.Kind != "outskirts" && zn.Kind != "town" {
		return
	}
	st := msSt(g)
	day := DayOf(g.Turn)
	if st.Rolled[zn.Key] == day+1 {
		return
	}
	st.Rolled[zn.Key] = day + 1
	act := 0
	for _, s := range st.Scenes {
		if s.State == "active" {
			act++
			if s.Zone == zn.Key {
				return
			}
		}
	}
	if act >= 2 {
		return
	}
	p := g.W.Idx(zn.WX, zn.WY)
	class, hzk, ch := msEvent(g, p)
	if class == "" {
		return
	}
	r := NewRng(fmt.Sprint(g.Seed, ":ms:roll:", zn.Key, ":", day))
	if !r.Chance(ch) {
		return
	}
	// fall back to a class the ground can host
	switch class {
	case "fire":
		n := 0
		for _, t := range zn.T {
			if t.D().Burn {
				n++
			}
		}
		if n < 12 {
			class = "hold"
		}
	case "shore":
		n := 0
		for _, t := range zn.T {
			if t == TShallow {
				n++
			}
		}
		if n < 3 {
			class = "evac"
		}
	}
	var bd *BurgData
	if zn.Kind == "town" && zn.Burg > 0 {
		bd = g.W.burgByID[zn.Burg]
	}
	if bd == nil {
		bd = msNearestBurg(g, p)
	}
	if bd == nil {
		return
	}
	st.Next++
	cd := msClasses[class]
	place := bd.Name
	s := &msScene{ID: st.Next, Class: class, Hz: hzk, Zone: zn.Key, Place: place, Burg: bd.ID, Start: g.Turn, Due: g.Turn + cd.Mins*2, State: "active"}
	if !msSetup(g, zn, s, bd, r) {
		return
	}
	st.Scenes = append(st.Scenes, s)
	g.Msg(cd.Brief)
	g.Msg(fmt.Sprintf("{{Y|%s}} ({{R|%s}} left).", fmt.Sprintf(cd.Title, place), dlClock(s.Due-g.Turn)))
	g.msLog("Scene: " + fmt.Sprintf(cd.Title, place) + ".")
}

func msSetup(g *Game, zn *Zone, s *msScene, bd *BurgData, r *Rng) bool {
	cd := msClasses[s.Class]
	switch s.Class {
	case "dig":
		n := 2 + r.Int(0, 1)
		s.Rub = map[int][]int{}
		for i := 0; i < n; i++ {
			x, y, ok := msSpot(g, zn, r, 4, 22, nil)
			if !ok {
				break
			}
			a := msPerson(g, zn, bd, r, "ms_trapped", "trapped survivor", s, true)
			if a == nil {
				break
			}
			a.HP = maxi(1, a.MaxHP*55/100)
			msPlace(zn, a, x, y)
			dirs := dirs8[:]
			perm := r.Int(0, 7)
			cnt := 0
			for k := 0; k < 8 && cnt < 3; k++ {
				d := dirs[(k+perm)%8]
				cx, cy := x+d[0], y+d[1]
				if zn.Walkable(cx, cy) && zn.ActorAt(cx, cy) == nil && !(cx == g.P.X && cy == g.P.Y) {
					zn.Set(cx, cy, TRubble)
					s.Rub[a.ID] = append(s.Rub[a.ID], cy*MW+cx)
					cnt++
				}
			}
			s.Total++
		}
		return s.Total > 0
	case "hold":
		x, y, ok := msSpot(g, zn, r, 2, 8, nil)
		if !ok {
			return false
		}
		w := msPerson(g, zn, bd, r, "ms_warden", cd.Goal, s, false)
		if w == nil {
			return false
		}
		msPlace(zn, w, x, y)
		s.Waves = 3
		s.Total = 3
		s.Next = g.Turn + 8
		return true
	case "fire":
		x, y, ok := msSpot(g, zn, r, 2, 8, nil)
		if !ok {
			return false
		}
		w := msPerson(g, zn, bd, r, "ms_warden", cd.Goal, s, false)
		if w == nil {
			return false
		}
		msPlace(zn, w, x, y)
		s.Age = map[int]int{}
		n := 0
		for i := 0; i < 300 && n < 3; i++ {
			fx, fy := r.Int(1, MW-2), r.Int(1, MH-2)
			d := maxi(abs(fx-g.P.X), abs(fy-g.P.Y))
			if d < 5 || d > 18 || !zn.At(fx, fy).D().Burn || zn.ActorAt(fx, fy) != nil {
				continue
			}
			if msLight(g, zn, s, fx, fy) {
				n++
			}
		}
		return n > 0
	default: // carry evac shore
		gx, gy, ok := msSpot(g, zn, r, 10, 28, func(x, y int) bool { return zn.At(x, y).D().Water == 0 })
		if !ok {
			return false
		}
		goal := msPerson(g, zn, bd, r, cd.GoalRol, cd.Goal, s, false)
		if goal == nil {
			return false
		}
		msPlace(zn, goal, gx, gy)
		n := 3
		if g.P.Level > 6 {
			n = 4
		}
		title := msVictimTitles[cd.Role]
		for i := 0; i < n; i++ {
			var pred func(x, y int) bool
			if s.Class == "shore" {
				pred = func(x, y int) bool { return zn.At(x, y) == TShallow }
			}
			x, y, ok := msSpot(g, zn, r, 3, 16, pred)
			if !ok {
				break
			}
			a := msPerson(g, zn, bd, r, cd.Role, title, s, true)
			if a == nil {
				break
			}
			if s.Class == "carry" {
				a.HP = maxi(1, a.MaxHP*45/100)
			}
			msPlace(zn, a, x, y)
			s.Total++
		}
		return s.Total > 0
	}
}

func msLight(g *Game, zn *Zone, s *msScene, x, y int) bool {
	a := g.Spawn("golem", 1, g.R)
	if a == nil {
		return false
	}
	a.Name, a.Glyph, a.F, a.Tpl = "blaze", '^', COrange, "golem"
	a.Faction, a.AI = "vermin", "hostile"
	a.HP, a.MaxHP, a.AVNat, a.DVNat = 5, 5, 0, 0
	a.Role = "ms_fire"
	a.Desc = "Flames licking through the growth."
	a.Set("still")
	a.Set(fmt.Sprintf("ms%d", s.ID))
	a.X, a.Y = x, y
	zn.Actors = append(zn.Actors, a)
	if s.Age == nil {
		s.Age = map[int]int{}
	}
	s.Age[a.ID] = g.Turn
	return true
}

// ---- scene ticks ----

func msFindRole(g *Game, zn *Zone, s *msScene, role string) *Actor {
	for _, a := range zn.Actors {
		if a.Role == role && a.HP > 0 {
			if id, ok := msTag(a, "ms"); ok && id == s.ID {
				return a
			}
		}
	}
	return nil
}

func msSceneTick(g *Game, zn *Zone, s *msScene) {
	cd := msClasses[s.Class]
	p := g.P
	left := s.Due - g.Turn
	total := s.Due - s.Start
	if left <= 0 {
		msFinish(g, s, s.Class == "hold" && false, "time ran out")
		return
	}
	if s.Warn < 1 && left < total/2 {
		s.Warn = 1
		g.Msg(fmt.Sprintf("{{Y|%s: half the time is gone (%s left).}}", fmt.Sprintf(cd.Title, s.Place), dlClock(left)))
	} else if s.Warn < 2 && left < total/5 {
		s.Warn = 2
		g.Msg(fmt.Sprintf("{{R|%s: almost out of time (%s)!}}", fmt.Sprintf(cd.Title, s.Place), dlClock(left)))
	}
	switch s.Class {
	case "dig":
		pos := p.Y*MW + p.X
		for aid, l := range s.Rub {
			for i, c := range l {
				if c == pos {
					s.Rub[aid] = append(l[:i:i], l[i+1:]...)
					zn.Set(p.X, p.Y, TDirt)
					g.Turn += 2
					g.Msg("You heave the rubble aside.")
					break
				}
			}
		}
	case "hold":
		if s.Spawned < s.Waves && g.Turn >= s.Next {
			s.Spawned++
			s.Next = g.Turn + 22
			n := 2 + g.P.Level/5 + s.Spawned/2
			if n > 6 {
				n = 6
			}
			made := 0
			for i := 0; i < n; i++ {
				x, y, ok := msSpot(g, zn, g.R, 12, 30, nil)
				if !ok {
					break
				}
				a := g.Spawn("bandit", maxi(zn.Level, 1), g.R)
				if a == nil {
					break
				}
				a.Faction, a.AI, a.Role = "outlaws", "hostile", "ms_raider"
				a.Name = "wrecker"
				a.Set(fmt.Sprintf("ms%d", s.ID))
				a.X, a.Y = x, y
				zn.Actors = append(zn.Actors, a)
				made++
			}
			g.Msg(fmt.Sprintf("{{R|Wave %d of %d: %d wreckers come out of the wreckage!}}", s.Spawned, s.Waves, made))
		}
		if s.Spawned >= s.Waves && msCount(g, zn, s, "ms_raider") == 0 {
			msFinish(g, s, true, "the last wave broke")
		}
	case "fire":
		msFireTick(g, zn, s)
	default:
		msCarryTick(g, zn, s)
	}
	if s.State == "active" && msIsVictimClass(s.Class) && s.Got+s.Lost >= s.Total && s.Total > 0 {
		msFinish(g, s, s.Got > 0 && s.Got*2 >= s.Total, "all accounted for")
	}
}

func msIsVictimClass(c string) bool {
	return c == "dig" || c == "carry" || c == "evac" || c == "shore"
}

func msCount(g *Game, zn *Zone, s *msScene, role string) int {
	n := 0
	for _, a := range zn.Actors {
		if a.Role == role && a.HP > 0 {
			if id, ok := msTag(a, "ms"); ok && id == s.ID {
				n++
			}
		}
	}
	return n
}

func msCarryTick(g *Game, zn *Zone, s *msScene) {
	p := g.P
	var goal *Actor
	for _, a := range zn.Actors {
		if a.HP > 0 && (a.Role == "ms_healer" || a.Role == "ms_camp" || a.Role == "ms_marshal") {
			if id, ok := msTag(a, "ms"); ok && id == s.ID {
				goal = a
			}
		}
	}
	for _, a := range append([]*Actor(nil), zn.Actors...) {
		if a.HP <= 0 || !msIsVictim(a.Role) || !a.Has("ms_led") {
			continue
		}
		if id, ok := msTag(a, "ms"); !ok || id != s.ID {
			continue
		}
		if goal != nil && dist(a, goal) <= 2 {
			msFree(g, zn, s, a, "delivered")
			continue
		}
		if dist(a, p) > 2 {
			g.stepToward(a, g.distMap(p.X, p.Y, false), false)
		}
	}
}

func msFireTick(g *Game, zn *Zone, s *msScene) {
	p := g.P
	var fires []*Actor
	for _, a := range zn.Actors {
		if a.HP > 0 && a.Role == "ms_fire" {
			if id, ok := msTag(a, "ms"); ok && id == s.ID {
				fires = append(fires, a)
			}
		}
	}
	// buckets douse an adjacent blaze at once
	for _, f := range fires {
		if dist(f, p) <= 1 {
			for _, it := range p.Inv {
				if it.Name == "bucket of water" {
					g.consume(it)
					f.HP = 0
					g.kill(f, p)
					g.Msg("{{C|You douse the flames with a bucket of water.}}")
					break
				}
			}
			if f.HP > 0 && g.Turn%3 == 0 {
				g.hurt(p, 1, nil, "scorching flames")
			}
		}
	}
	fires = fires[:0]
	for _, a := range zn.Actors {
		if a.HP > 0 && a.Role == "ms_fire" {
			if id, ok := msTag(a, "ms"); ok && id == s.ID {
				fires = append(fires, a)
			}
		}
	}
	for _, f := range fires {
		if g.Turn-s.Age[f.ID] > 40 {
			f.HP = 0
			zn.Set(f.X, f.Y, TAsh)
			s.Burned++
			g.removeActor(zn, f)
		}
	}
	if g.Turn%3 == 0 && len(fires) < 24 {
		for _, f := range fires {
			if f.HP <= 0 || !g.R.Chance(0.3) {
				continue
			}
			d := dirs8[g.R.Int(0, 7)]
			nx, ny := f.X+d[0], f.Y+d[1]
			if zn.In(nx, ny) && zn.At(nx, ny).D().Burn && zn.ActorAt(nx, ny) == nil && !(nx == p.X && ny == p.Y) {
				msLight(g, zn, s, nx, ny)
			}
		}
	}
	alive := 0
	for _, a := range zn.Actors {
		if a.HP > 0 && a.Role == "ms_fire" {
			if id, ok := msTag(a, "ms"); ok && id == s.ID {
				alive++
			}
		}
	}
	if s.Burned >= 28 {
		msFinish(g, s, false, "the fire got away")
	} else if alive == 0 {
		msFinish(g, s, true, "the last flame is out")
	}
}

// msFree turns a rescued person into a grateful one who remembers.
func msFree(g *Game, zn *Zone, s *msScene, a *Actor, how string) {
	a.Role = "ms_saved"
	a.Title = "rescued " + a.Title
	delete(a.Flags, "still")
	delete(a.Flags, "ms_led")
	a.AI = "town"
	a.HP = maxi(a.HP, a.MaxHP*7/10)
	s.Got++
	msSt(g).Saved++
	g.gainXP(15 + 5*g.P.Level/2)
	if p := lvPersonFor(g, a); p != nil {
		p.remember(lvDay(g), "saved", "You saved their life in "+s.Place+".", 18)
		p.Mood = clampi(p.Mood+30, -100, 100)
	}
	if how == "delivered" {
		g.Msg("{{G|" + a.Name + " is safe.}}")
	} else {
		g.Msg("{{G|You pull " + a.Name + " free, alive.}}")
	}
}

func msLose(g *Game, zn *Zone, s *msScene, a *Actor) {
	s.Lost++
	msSt(g).Lost++
	if p := lvPersonFor(g, a); p != nil && p.Alive {
		p.Alive = false
		p.Died = lvYear(g)
		p.Cause = "died in the " + msHzWord(s.Hz) + " at " + s.Place
		p.addLife(lvYear(g), "death", "Died: "+p.Cause+".")
		g.msNews(s.Burg, fmt.Sprintf("%s died in the %s at %s.", p.Name, msHzWord(s.Hz), s.Place))
	}
	if zn == g.Cur {
		g.Msg("{{R|" + a.Name + " does not make it.}}")
	}
	a.HP = 0
	g.removeActor(zn, a)
}

func msHzWord(h string) string {
	if h == "plague" {
		return "plague"
	}
	if w, ok := hzKindName[h]; ok {
		return w
	}
	return "disaster"
}

func msFinish(g *Game, s *msScene, won bool, why string) {
	if s.State != "active" {
		return
	}
	cd := msClasses[s.Class]
	zn := g.Zones[s.Zone]
	if zn != nil {
		for _, a := range append([]*Actor(nil), zn.Actors...) {
			if a.HP <= 0 {
				continue
			}
			id, ok := msTag(a, "ms")
			if !ok || id != s.ID {
				continue
			}
			switch {
			case msIsVictim(a.Role):
				msLose(g, zn, s, a)
			case a.Role == "ms_fire":
				zn.Set(a.X, a.Y, TAsh)
				g.removeActor(zn, a)
			}
		}
	}
	if msIsVictimClass(s.Class) {
		won = s.Got > 0 && s.Got*2 >= s.Total
	}
	st := msSt(g)
	s.Ended = g.Turn
	title := fmt.Sprintf(cd.Title, s.Place)
	r := NewRng(fmt.Sprint(g.Seed, ":ms:fin:", s.ID))
	fac := ""
	if bd := g.W.burgByID[s.Burg]; bd != nil && bd.State > 0 {
		fac = fmt.Sprintf("state:%d", bd.State)
	}
	if won {
		s.State = "won"
		st.Won++
		n := maxi(1, s.Got)
		if s.Class == "hold" || s.Class == "fire" {
			n = 2
		}
		if s.Class == "fire" {
			n = maxi(1, 3-s.Burned/10)
		}
		gold := cd.Reward + cd.Reward*g.P.Level/6 + 10*n
		g.Msg("{{G|" + title + ": done (" + why + ").}} You earn " + g.cuEarn(gold) + ".")
		if fac != "" {
			g.AddRep(fac, 15+10*n, " for the rescue at "+s.Place)
		}
		g.gainXP(40 + 25*n)
		g.deed("rescue", fmt.Sprintf("%s helped at %s: %s.", g.P.Name, s.Place, strings.ToLower(title)), 0, 0)
		g.msNews(s.Burg, fmt.Sprintf(Pick(r, cd.Won), s.Place, g.P.Name))
		g.msLog("Won: " + title + ".")
	} else {
		s.State = "lost"
		g.Msg("{{R|" + title + ": failed (" + why + ").}}")
		n := s.Lost
		if s.Class == "hold" || s.Class == "fire" {
			n = 1 + g.P.Level/5 + r.Int(0, 1)
		}
		if n < 1 {
			n = 1
		}
		msKillFolk(g, s.Burg, n, "the "+msHzWord(s.Hz)+" at "+s.Place)
		g.msNews(s.Burg, fmt.Sprintf(Pick(r, cd.Lost), s.Place))
		if fac != "" && s.Got == 0 {
			g.AddRep(fac, -5, " for failing to help")
		}
		g.msLog("Lost: " + title + ".")
	}
}

// msKillFolk kills n living townsfolk of a burg (common folk first), with news.
func msKillFolk(g *Game, burg, n int, cause string) {
	lst := lvSt(g)
	var keys []string
	for k, p := range lst.People {
		if p.Alive && p.Home == burg && strings.HasPrefix(k, "g") {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	r := NewRng(fmt.Sprint(g.Seed, ":ms:folk:", burg, ":", g.Turn))
	for i := 0; i < n && len(keys) > 0; i++ {
		j := r.Int(0, len(keys)-1)
		p := lst.People[keys[j]]
		keys = append(keys[:j], keys[j+1:]...)
		p.Alive = false
		p.Died = lvYear(g)
		p.Cause = "lost in " + cause
		p.addLife(lvYear(g), "death", "Died: lost in "+cause+".")
		lst.addNews(lvDay(g), burg, "%s, %s of %s, was lost in %s.", p.Name, p.Role, msBurgName(g, burg), cause)
	}
}

// ---- kills ----

func msKill(g *Game, d, src *Actor) {
	if d == nil || g.W == nil {
		return
	}
	if id, ok := msTag(d, "ms"); ok {
		if s := msSceneByID(g, id); s != nil && s.State == "active" && msIsVictim(d.Role) {
			s.Lost++
			msSt(g).Lost++
		}
	}
	if fid, ok := msTag(d, "mf"); ok && src == g.P {
		for _, f := range msSt(g).Fights {
			if f.ID == fid && !f.Over {
				if d.Has("mfa") {
					f.Helped = "b"
				} else {
					f.Helped = "a"
				}
			}
		}
	}
}

// ---- talk ----

func msTalk(g *Game, a *Actor) []Option {
	if g.W == nil || a == nil {
		return nil
	}
	var o []Option
	if id, ok := msTag(a, "mc"); ok {
		if c := msColByID(g, id); c != nil && a.Role == "ms_leader" {
			return msLeaderTalk(g, a, c)
		}
		return nil
	}
	s := msSceneOf(g, a)
	switch a.Role {
	case "ms_trapped":
		if s == nil || s.State != "active" {
			return nil
		}
		zn := g.Cur
		left := s.Rub[a.ID]
		if len(left) > 0 {
			o = append(o, Option{"{{Y|Dig at the rubble pinning them.}}", func() {
				l := s.Rub[a.ID]
				if len(l) > 0 {
					c := l[0]
					s.Rub[a.ID] = l[1:]
					zn.Set(c%MW, c/MW, TDirt)
					g.Turn += 3
				}
				g.talkMenu("You haul stone and beams aside. \"Hurry... it is still on my legs.\"")
			}})
		} else {
			o = append(o, Option{"{{G|Pull them free.}}", func() {
				msFree(g, zn, s, a, "dug")
				g.talkMenu(Pick(NewRng(fmt.Sprint(a.ID)), msGratefulLines))
			}})
		}
	case "ms_patient", "ms_survivor", "ms_evac":
		if s == nil || s.State != "active" {
			return nil
		}
		if a.Has("ms_led") {
			o = append(o, Option{"Tell them to wait here.", func() {
				delete(a.Flags, "ms_led")
				g.talkMenu("\"I will wait.\"")
			}})
		} else {
			o = append(o, Option{"{{G|Help them up and lead them to safety.}}", func() {
				a.Set("ms_led")
				g.talkMenu("\"Lead on. I will keep close behind you.\"")
			}})
		}
	case "ms_warden", "ms_healer", "ms_camp", "ms_marshal":
		if s == nil || s.State != "active" {
			return nil
		}
		cd := msClasses[s.Class]
		o = append(o, Option{"Ask what is needed.", func() {
			g.talkMenu(cd.GoalMsg + fmt.Sprintf("\n\n(%s left.)", dlClock(s.Due-g.Turn)))
		}})
		if s.Class == "fire" && !s.Gave {
			o = append(o, Option{"{{G|Take the water buckets.}}", func() {
				s.Gave = true
				for i := 0; i < 4; i++ {
					g.give(&Item{ID: g.newID(), Kind: "trade", Name: "bucket of water", Glyph: 'U', F: CBlue, Value: 1, Weight: 2})
				}
				g.talkMenu("\"Four buckets. Stand next to a blaze and the water does the rest.\"")
			}})
		}
	case "ms_saved":
		if a.Has("ms_paid") {
			return nil
		}
		o = append(o, Option{"{{G|Accept their thanks.}}", func() {
			a.Set("ms_paid")
			g.talkMenu(Pick(NewRng(fmt.Sprint(a.ID)), msGratefulLines) + "\n\nThey press a purse into your hand: " + g.cuEarn(12+g.P.Level*3) + ".")
		}})
	}
	return o
}

func msLeaderTalk(g *Game, a *Actor, c *msCol) []Option {
	from, to := msBurgName(g, c.From), msBurgName(g, c.To)
	c.Known = true
	var o []Option
	if c.Kind == "refugee" {
		o = append(o, Option{"Ask who they are.", func() {
			g.talkMenu(fmt.Sprintf(Pick(NewRng(fmt.Sprint(c.ID)), msRefugeeAsk), from, to) + fmt.Sprintf("\n\n%d souls fled %s. They go to %s.", c.Size, c.Cause, to))
		}})
		if !c.Escorted {
			o = append(o, Option{"{{G|Offer to escort the column.}}", func() {
				c.Escorted = true
				msSt(g).Escort++
				g.msLog("You vowed to see the column from " + from + " safely to " + to + ".")
				g.talkMenu("\"May the road be kind to you. We will tell them in " + to + " who walked with us.\"")
			}})
		}
		if !c.Fed {
			o = append(o, Option{"Share your food with them.", func() {
				for _, it := range g.P.Inv {
					if it.Kind == "food" {
						g.consume(it)
						c.Fed = true
						g.gainXP(10)
						g.talkMenu("The children eat first. \"We will not forget it.\"")
						return
					}
				}
				g.talkMenu("You have nothing to spare.")
			}})
		}
		o = append(o, Option{"Ask for news of the road.", func() {
			t := fmt.Sprintf("\"%s is in ruins since %s. The road to %s is long and bandits watch it.\"", from, c.Cause, to)
			g.msNews(c.To, fmt.Sprintf("Refugees on the road tell of %s.", c.Cause))
			g.talkMenu(t)
		}})
	} else {
		o = append(o, Option{"Ask about the pilgrimage.", func() {
			g.talkMenu(fmt.Sprintf(Pick(NewRng(fmt.Sprint(c.ID)), msPilgrimLines), to) + fmt.Sprintf("\n\n%d pilgrims walk from %s.", c.Size, from))
		}})
		if !c.Joined {
			o = append(o, Option{"{{G|Walk with the pilgrims.}}", func() {
				c.Joined = true
				left := c.Start + c.Days - DayOf(g.Turn)
				g.msLog("You joined the pilgrimage from " + from + " to " + to + ".")
				g.talkMenu(fmt.Sprintf("\"Walk with us. We reach %s in about %d days; come to the shrine gate and be blessed.\"", to, maxi(left, 0)))
			}})
		}
		o = append(o, Option{"Offer alms to the pilgrims.", func() {
			if !g.cuSpend(5) {
				g.talkMenu("You cannot spare the coin.")
				return
			}
			if f := g.dlStateFac(c.To); f != "" {
				g.AddRep(f, 3, " for alms")
			}
			g.talkMenu("\"A blessing on your road.\"")
		}})
	}
	return o
}

// ---- columns ----

func (g *Game) msColPos(c *msCol, idx int) int {
	w := g.W
	a, b := w.burgByID[c.From], w.burgByID[c.To]
	if a == nil || b == nil {
		return -1
	}
	n := maxi(1, wsCheb(w, a.P, b.P))
	i := clampi(idx, 0, n)
	x0, y0, x1, y1 := a.P%w.W, a.P/w.W, b.P%w.W, b.P/w.W
	x := x0 + int(math.Round(float64(x1-x0)*float64(i)/float64(n)))
	y := y0 + int(math.Round(float64(y1-y0)*float64(i)/float64(n)))
	return w.Idx(x, y)
}

func msColGen(g *Game, day int) {
	st := msSt(g)
	act := 0
	for _, c := range st.Cols {
		if !c.Done {
			act++
		}
	}
	if act >= 3 || g.Cur == nil || !g.W.In(g.Cur.WX, g.Cur.WY) {
		return
	}
	r := NewRng(fmt.Sprint(g.Seed, ":ms:col:", day))
	if !r.Chance(0.3) {
		return
	}
	bl := append([]*BurgData(nil), g.ws().burgList(g)...)
	if len(bl) < 3 {
		return
	}
	w := g.W
	pp := w.Idx(g.Cur.WX, g.Cur.WY)
	sort.SliceStable(bl, func(i, j int) bool {
		a, b := wsCheb(w, pp, bl[i].P), wsCheb(w, pp, bl[j].P)
		if a != b {
			return a < b
		}
		return bl[i].ID < bl[j].ID
	})
	pool := bl[:mini(14, len(bl))]
	used := map[int]bool{}
	for _, c := range st.Cols {
		if !c.Done {
			used[c.From] = true
		}
	}
	c := &msCol{Start: day}
	if r.Chance(0.5) && w.Hazards != nil {
		var cands []*BurgData
		var cause string
		for i := range w.Hazards.Recent {
			e := &w.Hazards.Recent[i]
			if w.Year-e.Year > hzMarkYears[e.Kind] {
				continue
			}
			for _, tid := range e.Towns {
				b := w.burgByID[tid]
				if b != nil && !used[b.ID] && wsCheb(w, pp, b.P) <= 14 && b.Alive {
					cands = append(cands, b)
					cause = msCauseWord[e.Kind]
				}
			}
		}
		if len(cands) > 0 {
			from := cands[r.Int(0, len(cands)-1)]
			var to *BurgData
			bd := 1 << 30
			for _, b := range pool {
				d := wsCheb(w, from.P, b.P)
				if b.ID != from.ID && d >= 3 && d <= 14 && d < bd {
					to, bd = b, d
				}
			}
			if to != nil {
				c.Kind, c.From, c.To, c.Cause = "refugee", from.ID, to.ID, cause
				c.Size = 8 + r.Int(0, 22)
			}
		}
	}
	if c.Kind == "" {
		var best *BurgData
		for _, b := range pool {
			if best == nil || b.Pop*(0.5+r.Float()) > best.Pop*(0.5+r.Float()) {
				best = b
			}
		}
		var from *BurgData
		for _, b := range pool {
			d := wsCheb(w, best.P, b.P)
			if b.ID != best.ID && !used[b.ID] && d >= 3 && d <= 12 && (from == nil || r.Chance(0.3)) {
				from = b
			}
		}
		if from == nil {
			return
		}
		c.Kind, c.From, c.To = "pilgrim", from.ID, best.ID
		c.Size = 10 + r.Int(0, 30)
	}
	c.Days = maxi(2, wsCheb(w, w.burgByID[c.From].P, w.burgByID[c.To].P))
	st.Next++
	c.ID = st.Next
	st.Cols = append(st.Cols, c)
	if c.Kind == "refugee" {
		g.msNews(c.From, fmt.Sprintf("%d people leave %s for %s after %s.", c.Size, msBurgName(g, c.From), msBurgName(g, c.To), c.Cause))
	} else {
		g.msNews(c.From, fmt.Sprintf("A pilgrim column of %d sets out from %s for the shrine at %s.", c.Size, msBurgName(g, c.From), msBurgName(g, c.To)))
	}
}

func msColSettle(g *Game, c *msCol) {
	c.Done = true
	to, from := msBurgName(g, c.To), msBurgName(g, c.From)
	if c.Kind == "refugee" {
		n := c.Size
		if !c.Escorted && !c.Fed {
			n = n * 2 / 3
		}
		g.msNews(c.To, fmt.Sprintf("%d refugees from %s reach %s; the lanes are crowded and rents climb.", n, from, to))
		g.msNews(c.From, fmt.Sprintf("%s is quieter now: %d of its people left for %s.", from, c.Size, to))
		if c.Escorted {
			f := g.dlStateFac(c.To)
			if f != "" {
				g.AddRep(f, 40, " for escorting refugees")
			}
			g.Msg("{{G|The refugees you escorted have reached " + to + ". You earn " + g.cuEarn(30+c.Size*3) + ".}}")
			g.deed("escort", fmt.Sprintf("%s escorted refugees from %s to %s.", g.P.Name, from, to), 0, 0)
		}
		g.msLog("The column from " + from + " settled in " + to + ".")
	} else {
		g.msNews(c.To, fmt.Sprintf("%d pilgrims from %s arrive at %s for the feast; the inns fill and candles burn.", c.Size, from, to))
		g.msLog("Pilgrims reached " + to + ".")
	}
}

func msColShow(g *Game, zn *Zone) {
	if zn.Z != 0 || (zn.Kind != "surface" && zn.Kind != "outskirts") {
		return
	}
	day := DayOf(g.Turn)
	p := g.W.Idx(zn.WX, zn.WY)
	st := msSt(g)
	for _, c := range st.Cols {
		idx := day - c.Start
		if c.Done || idx < 0 || idx > c.Days || g.msColPos(c, idx) != p {
			continue
		}
		if c.ShownDay == day && c.ShownZ == zn.Key {
			continue
		}
		bd := g.W.burgByID[c.From]
		if bd == nil {
			continue
		}
		r := NewRng(fmt.Sprint(g.Seed, ":ms:show:", c.ID, ":", day))
		cx, cy, ok := msSpot(g, zn, r, 5, 18, nil)
		if !ok {
			continue
		}
		role, title := "refugee", "refugee"
		if c.Kind == "pilgrim" {
			role, title = "pilgrim", "pilgrim"
		}
		flag := fmt.Sprintf("mc%d", c.ID)
		ld := msPerson(g, zn, bd, r, "ms_leader", map[string]string{"refugee": "column leader", "pilgrim": "pilgrim guide"}[c.Kind], nil, false)
		if ld == nil {
			continue
		}
		ld.Set(flag)
		msPlace(zn, ld, cx, cy)
		c.Leader = ld.Name
		n := clampi(c.Size/6, 3, 6)
		for i := 0; i < n; i++ {
			mx, my, ok := msSpot(g, zn, r, 1, 80, func(x, y int) bool { return maxi(abs(x-cx), abs(y-cy)) <= 4 })
			if !ok {
				break
			}
			m := msPerson(g, zn, bd, r, role, title, nil, false)
			if m == nil {
				break
			}
			m.Set(flag)
			msPlace(zn, m, mx, my)
		}
		c.ShownDay, c.ShownZ, c.Known = day, zn.Key, true
		if c.Kind == "refugee" {
			g.Msg("{{Y|A column of refugees trudges along here, bundles on their backs.}}")
		} else {
			g.Msg("{{Y|A column of pilgrims files past, singing.}}")
		}
	}
}

// msColTick lets column members mill around their leader.
func msColTick(g *Game, zn *Zone) {
	var leaders map[int]*Actor
	for _, a := range zn.Actors {
		if a.Role == "ms_leader" {
			if id, ok := msTag(a, "mc"); ok {
				if leaders == nil {
					leaders = map[int]*Actor{}
				}
				leaders[id] = a
			}
		}
	}
	if leaders == nil {
		return
	}
	for _, a := range zn.Actors {
		if a.HP <= 0 || (a.Role != "refugee" && a.Role != "pilgrim") {
			continue
		}
		id, ok := msTag(a, "mc")
		if !ok || leaders[id] == nil {
			continue
		}
		l := leaders[id]
		if dist(a, l) > 4 {
			g.stepToward(a, g.distMap(l.X, l.Y, false), false)
		} else if g.R.Chance(0.15) {
			g.wander(a)
		}
	}
}

func msBlessCheck(g *Game, zn *Zone) {
	if zn.Kind != "town" || zn.Burg <= 0 {
		return
	}
	day := DayOf(g.Turn)
	for _, c := range msSt(g).Cols {
		if c.Kind != "pilgrim" || !c.Joined || c.Blessed || c.To != zn.Burg {
			continue
		}
		arrive := c.Start + c.Days
		if day < arrive-1 {
			continue
		}
		if day > arrive+4 {
			c.Blessed = true
			continue
		}
		c.Blessed = true
		msSt(g).Pilgrim++
		g.P.AddSt("inspired", 4000)
		g.P.HP = mini(g.P.MaxHP, g.P.HP+10)
		g.Msg("{{G|At the shrine gate of " + zn.Name + " the pilgrims gather around you and a priest speaks a blessing over you.}}")
		if f := g.dlStateFac(c.To); f != "" {
			g.AddRep(f, 30, " for the pilgrimage")
		}
		g.gainXP(60)
		g.deed("pilgrim", fmt.Sprintf("%s walked the pilgrim road from %s to %s and was blessed.", g.P.Name, msBurgName(g, c.From), msBurgName(g, c.To)), 0, 0)
		g.msNews(c.To, "The wanderer "+g.P.Name+" was blessed at the shrine with the pilgrims.")
		g.msLog("Blessed at " + msBurgName(g, c.To) + ".")
	}
}

// ---- beast territory fights ----

func msFightRoll(g *Game, zn *Zone) {
	if zn.Kind != "surface" || zn.Z != 0 {
		return
	}
	st := msSt(g)
	day := DayOf(g.Turn)
	key := "f:" + zn.Key
	if st.Rolled[key] == day+1 {
		return
	}
	st.Rolled[key] = day + 1
	for _, f := range st.Fights {
		if f.Zone == zn.Key && !f.Over {
			return
		}
	}
	if g.msActiveHere() != nil {
		return
	}
	r := NewRng(fmt.Sprint(g.Seed, ":ms:fight:", zn.Key, ":", day))
	if !r.Chance(0.06) {
		return
	}
	pair := Pick(r, msFightPairs)
	ax, ay, ok := msSpot(g, zn, r, 8, 22, nil)
	if !ok {
		return
	}
	bx, by, ok := msSpot(g, zn, r, 1, 80, func(x, y int) bool { d := maxi(abs(x-ax), abs(y-ay)); return d >= 4 && d <= 7 })
	if !ok {
		return
	}
	st.Next++
	f := &msFight{ID: st.Next, Zone: zn.Key, A: pair[0], B: pair[1], Day: day}
	for side, tpl := range []string{pair[0], pair[1]} {
		cx, cy := ax, ay
		flag := "mfa"
		if side == 1 {
			cx, cy, flag = bx, by, "mfb"
		}
		for i := 0; i < 2+r.Int(0, 1); i++ {
			x, y, ok := msSpot(g, zn, r, 1, 80, func(x, y int) bool { return maxi(abs(x-cx), abs(y-cy)) <= 2 })
			if !ok {
				break
			}
			a := g.Spawn(tpl, maxi(zn.Level, 1), r)
			if a == nil {
				break
			}
			a.Set("still")
			a.Set(flag)
			a.Set(fmt.Sprintf("mf%d", f.ID))
			a.X, a.Y = x, y
			zn.Actors = append(zn.Actors, a)
		}
	}
	st.Fights = append(st.Fights, f)
	g.Msg(fmt.Sprintf("{{Y|Snarls and roars carry across the ground: a %s pack and a %s pack are fighting over it.}}", pair[0], pair[1]))
}

func msFightTick(g *Game, zn *Zone) {
	for _, f := range msSt(g).Fights {
		if f.Over || f.Zone != zn.Key {
			continue
		}
		tag := fmt.Sprintf("mf%d", f.ID)
		var as, bs []*Actor
		for _, a := range zn.Actors {
			if a.HP > 0 && a.Has(tag) {
				if a.Has("mfa") {
					as = append(as, a)
				} else {
					bs = append(bs, a)
				}
			}
		}
		if len(as) == 0 || len(bs) == 0 {
			msFightEnd(g, zn, f, as, bs)
			continue
		}
		if g.Turn%2 == 0 {
			continue
		}
		for _, side := range [][2][]*Actor{{as, bs}, {bs, as}} {
			for _, a := range side[0] {
				var tgt *Actor
				bd := 1 << 20
				for _, o := range side[1] {
					if o.HP > 0 && dist(a, o) < bd {
						tgt, bd = o, dist(a, o)
					}
				}
				if tgt == nil {
					continue
				}
				if bd <= 1 {
					g.melee(a, tgt)
				} else {
					g.stepToward(a, g.distMap(tgt.X, tgt.Y, false), false)
				}
			}
		}
	}
}

func msFightEnd(g *Game, zn *Zone, f *msFight, as, bs []*Actor) {
	f.Over = true
	surv := as
	name := f.A
	if len(bs) > 0 {
		surv, name = bs, f.B
	}
	side := "a"
	if len(bs) > 0 {
		side = "b"
	}
	near := dist(g.P, g.P) == 0
	_ = near
	if f.Helped == side && len(surv) > 0 {
		for _, a := range surv {
			delete(a.Flags, "still")
			a.AI, a.Faction, a.Name = "wander", "citizens", "grateful "+a.Name
		}
		g.gainXP(40 + 10*g.P.Level)
		g.Msg("{{G|You turned the fight. The surviving " + name + "s keep their ground and watch you go without a growl.}}")
		g.deed("beasts", g.P.Name+" took the side of the "+name+"s in a fight over territory.", 0, 0)
		g.msLog("You helped the " + name + "s win their ground.")
		return
	}
	for _, a := range surv {
		delete(a.Flags, "still")
		a.AI = "hostile"
	}
	if len(surv) > 0 {
		g.Msg("{{Y|The fight ends: the " + name + "s hold the ground, bloodied and weak. They turn toward you.}}")
		g.gainXP(10 + 3*g.P.Level)
	} else {
		g.Msg("{{Y|The fight ends with no victor.}}")
	}
	g.msLog("A fight between " + f.A + " and " + f.B + " ended.")
}

// ---- sidebar and screen ----

func msSide(g *Game) []string {
	if g.W == nil {
		return nil
	}
	s := g.msActiveHere()
	if s == nil {
		return nil
	}
	cd := msClasses[s.Class]
	left := dlClock(s.Due - g.Turn)
	switch s.Class {
	case "hold":
		return []string{fmt.Sprintf("{{Y|Hold:}} wave %d/%d, %s", s.Spawned, s.Waves, left)}
	case "fire":
		return []string{fmt.Sprintf("{{Y|Fire:}} burned %d/28, %s", s.Burned, left)}
	}
	return []string{fmt.Sprintf("{{Y|%s:}} %d/%d saved, %s", titleCase(cd.Victim), s.Got, s.Total, left)}
}

func (g *Game) msAbilities() []Ability {
	if g.P == nil || g.W == nil {
		return nil
	}
	return []Ability{{ID: "ms_events", Name: "Disasters and migrations", Target: "self"}}
}

func (g *Game) msAbUse(id string) {
	if id == "ms_events" {
		g.Mode = "ms"
	}
}

func msKeys(g *Game, k string) {
	switch k {
	case "Escape", "q", "Enter":
		g.Mode = "main"
	}
}

func (g *Game) msDraw() {
	sc := g.sc
	st := msSt(g)
	x, y, w, h := g.panel("Scenes and Events")
	bottom := y + h
	put := func(s string) {
		for _, ln := range Wrap(s, w) {
			if y < bottom {
				sc.TextB(x, y, ln, CGray, CBgDark, x+w)
				y++
			}
		}
	}
	put("{{K|Esc close}}")
	y++
	put(fmt.Sprintf("Lives pulled out: %d   lost: %d   scenes won: %d   pilgrimages: %d   columns escorted: %d", st.Saved, st.Lost, st.Won, st.Pilgrim, st.Escort))
	y++
	put("{{C|Scenes}}")
	n := 0
	for i := len(st.Scenes) - 1; i >= 0 && n < 6; i-- {
		s := st.Scenes[i]
		cd := msClasses[s.Class]
		tail := "{{K|" + s.State + "}}"
		if s.State == "active" {
			tail = "{{Y|" + dlClock(s.Due-g.Turn) + " left}}"
		}
		put(fmt.Sprintf("%s  %s", fmt.Sprintf(cd.Title, s.Place), tail))
		n++
	}
	if n == 0 {
		put("{{K|None yet. Disasters leave people in need in the zones they struck.}}")
	}
	y++
	put("{{C|Roads}}")
	day := DayOf(g.Turn)
	m := 0
	for _, c := range st.Cols {
		if !c.Known {
			continue
		}
		state := fmt.Sprintf("day %d/%d", clampi(day-c.Start, 0, c.Days), c.Days)
		if c.Done {
			state = "arrived"
		}
		extra := ""
		if c.Escorted {
			extra = " (escorted)"
		}
		if c.Joined {
			extra += " (you walk with them)"
		}
		put(fmt.Sprintf("%s: %s to %s, %d people, %s%s", titleCase(c.Kind), msBurgName(g, c.From), msBurgName(g, c.To), c.Size, state, extra))
		m++
	}
	if m == 0 {
		put("{{K|No column met yet. Refugees and pilgrims walk between towns.}}")
	}
	if len(st.Log) > 0 {
		y++
		put("{{C|Recent}}")
		for i := len(st.Log) - 1; i >= 0 && y < bottom; i-- {
			put("{{K|" + st.Log[i] + "}}")
		}
	}
}
