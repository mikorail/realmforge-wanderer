package engine

// Narrative director (prefix nd): a pacing layer over the whole game.
//
// TENSION (0..100) is fed from play state each minute: nearby hostiles, low health, kills, a companion's death,
// quests finished, karma shifts, gold windfalls, and the time since the last beat (monotony pushes it up a little,
// quiet in a town pulls it down). A DIRECTOR picks story BEATS from templates (director_data.go) fed with the
// world's real data: nearby towns, living people from lives, enemy realms at war, the nemesis (nx) and rivals (rv)
// read-only, the prophecy verses (wy) read-only, and monsters not yet slain. Up to three ARCS run at once, each
// walking inciting incident, rising action, twist, crisis, climax, aftermath, and binds recurring characters
// (nemesis, rival, a mentor, a love interest) that stay the same across arcs.
//
// Beats are light interventions through other systems' public helpers: a letter queued with the courier system
// (letters.go lnQueue), a rumour pushed into lives' news (lvSt.addNews) and the message log, an ambush spawned when
// you next arrive somewhere that can host one, a calm festival in the next town. Quests are never touched.
//
// ENDINGS: every finished arc has a flavour (triumph, tragedy, redemption, vengeance, peace, mystery, sacrifice, ruin,
// legacy). Flavours seen in this run and in this world's past lives (hall of fame legends) bias which arc starts next
// (the less seen the likelier). ndStorySoFar() is the one-line summary for the death/Saga screen. Screen: mode `nd`,
// ability `nd_story`. State g.ext("nd") (global). Side rng only for choices that must be stable; play-time rolls use g.R.

import (
	"fmt"
	"sort"
	"strings"
)

func init() {
	turnHooks = append(turnHooks, ndTurn)
	enterHooks = append(enterHooks, ndEnter)
	killHooks = append(killHooks, ndKill)
	modeKeys["nd"] = ndKeys
	modeDraws["nd"] = func(g *Game) { g.ndDraw() }
}

const ndGap = 240 // minutes between beats

type ndChar struct {
	Role string `json:"r"` // foe friend
	Kind string `json:"k"` // nemesis rival mentor love person realm monster verse
	Key  string `json:"key,omitempty"`
	Name string `json:"n"`
	ID   int    `json:"id,omitempty"`
}

type ndArc struct {
	ID      int      `json:"id"`
	Tpl     string   `json:"t"`
	Title   string   `json:"ti"`
	Flavour string   `json:"fl"`
	Stage   int      `json:"s"`
	Pl      int      `json:"pl"`
	Town    int      `json:"to"`
	Other   int      `json:"ot"`
	Enemy   string   `json:"en,omitempty"`
	Verse   string   `json:"vs,omitempty"`
	Reason  string   `json:"rs,omitempty"`
	Cast    []ndChar `json:"c"`
	Start   int      `json:"st"`
	Last    int      `json:"la"`
	Next    int      `json:"nx"`
	Await   bool     `json:"aw,omitempty"`
	Beats   int      `json:"b"`
}

type ndBeat struct {
	Day   int    `json:"d"`
	Arc   string `json:"a"`
	Stage int    `json:"s"`
	Kind  string `json:"k"`
	Text  string `json:"t"`
}

type ndPend struct {
	Arc   int      `json:"a"`
	Kind  string   `json:"k"` // ambush boss festival
	Text  string   `json:"t"`
	Foe   string   `json:"f,omitempty"`
	Day   int      `json:"d"`
	Spawn []string `json:"sp,omitempty"`
	Burg  int      `json:"b,omitempty"`
}

type ndDone struct {
	Title   string `json:"t"`
	Flavour string `json:"f"`
	Day     int    `json:"d"`
	Tpl     string `json:"p"`
}

type ndState struct {
	Init     bool              `json:"i,omitempty"`
	Tension  float64           `json:"te"`
	Peak     float64           `json:"pk,omitempty"`
	LastBeat int               `json:"lb"`
	Arcs     []*ndArc          `json:"arcs,omitempty"`
	Done     []*ndDone         `json:"done,omitempty"`
	Beats    []ndBeat          `json:"beats,omitempty"`
	Pend     []*ndPend         `json:"pend,omitempty"`
	Next     int               `json:"n"`
	Seen     map[string]int    `json:"seen,omitempty"`
	Recur    map[string]string `json:"rec,omitempty"`
	Gold     int               `json:"g,omitempty"`
	Quests   int               `json:"q,omitempty"`
	Kar      [5]int            `json:"kar"`
	Combat   int               `json:"cb,omitempty"`
}

func ndSt(g *Game) *ndState {
	s := g.ext("nd", func() any { return &ndState{} }).(*ndState)
	if s.Seen == nil {
		s.Seen = map[string]int{}
	}
	if s.Recur == nil {
		s.Recur = map[string]string{}
	}
	return s
}

// ---------------------------------------------------------------- public helpers

func (g *Game) ndTension() int {
	if g == nil || g.W == nil {
		return 0
	}
	return int(ndSt(g).Tension)
}

func (g *Game) ndAbilities() []Ability {
	if g.Cur == nil || g.P == nil || g.W == nil {
		return nil
	}
	return []Ability{{ID: "nd_story", Name: "Story so far", Target: "self"}}
}

func (g *Game) ndAbUse(id string) {
	if id == "nd_story" {
		g.Mode, g.Tab, g.Sel = "nd", 0, 0
	}
}

// ndSeenAll: ending flavours this run has finished plus those read off this world's past lives.
func ndSeenAll(g *Game) map[string]int {
	out := map[string]int{}
	for k, v := range ndSt(g).Seen {
		out[k] += v
	}
	for _, l := range g.hf().Legends {
		out[ndLegendFlavour(l)]++
	}
	return out
}

func ndLegendFlavour(l HFLegend) string {
	all := strings.ToLower(l.Cause + " " + strings.Join(l.Deeds, " "))
	switch {
	case strings.Contains(all, "retired"):
		return "peace"
	case strings.Contains(all, "nemesis") || strings.Contains(all, "bounty"):
		return "vengeance"
	case strings.Contains(all, "prophecy") || strings.Contains(all, "champion"):
		return "triumph"
	case strings.Contains(all, "companion") || strings.Contains(all, "sacrific"):
		return "sacrifice"
	}
	return "tragedy"
}

// ndEndingLean: the flavour the run is currently heading for (the active arc furthest along, else the karma lean).
func ndEndingLean(g *Game) string {
	st := ndSt(g)
	var best *ndArc
	for _, a := range st.Arcs {
		if best == nil || a.Stage > best.Stage {
			best = a
		}
	}
	if best != nil {
		return best.Flavour
	}
	if len(st.Done) > 0 {
		return st.Done[len(st.Done)-1].Flavour
	}
	return ""
}

// ndStorySoFar is the one-line story summary for the death and Saga screens ("" when nothing happened).
// Hook: drawDead, after the obituary: `if s := g.ndStorySoFar(); s != "" { lines = append(lines, "", s) }`.
func (g *Game) ndStorySoFar() string {
	if g == nil || g.W == nil || g.P == nil {
		return ""
	}
	st := ndSt(g)
	var parts []string
	for _, d := range st.Done {
		parts = append(parts, fmt.Sprintf("%s ended as %s", d.Title, ndFlavours[d.Flavour]))
	}
	for _, a := range st.Arcs {
		parts = append(parts, fmt.Sprintf("%s is at its %s", a.Title, ndStageNames[mini(a.Stage, 5)]))
	}
	if len(parts) == 0 {
		return ""
	}
	if len(parts) > 3 {
		parts = parts[len(parts)-3:]
	}
	s := "Story so far: " + strings.Join(parts, "; ") + "."
	if lean := ndEndingLean(g); lean != "" {
		s += " It was heading toward " + ndFlavours[lean] + "."
	}
	return "{{M|" + s + "}}"
}

// ---------------------------------------------------------------- world data

type ndNear struct {
	B *BurgData
	D int
}

func ndNearTowns(g *Game, n int) []*BurgData {
	if g.W == nil || g.Cur == nil {
		return nil
	}
	here := g.W.Idx(g.Cur.WX, g.Cur.WY)
	var all []ndNear
	for i := range g.W.Burgs {
		b := &g.W.Burgs[i]
		if !b.Alive || b.P < 0 || b.P >= g.W.W*g.W.H {
			continue
		}
		all = append(all, ndNear{b, wsCheb(g.W, b.P, here)})
	}
	sort.Slice(all, func(i, j int) bool {
		if all[i].D != all[j].D {
			return all[i].D < all[j].D
		}
		return all[i].B.ID < all[j].B.ID
	})
	var out []*BurgData
	for i := 0; i < len(all) && i < n && all[i].D <= 16; i++ {
		out = append(out, all[i].B)
	}
	return out
}

func ndBurgName(g *Game, id int) string {
	if b := g.W.burgByID[id]; b != nil {
		return b.Name
	}
	return "a distant town"
}

func ndHomeState(g *Game, town int) int {
	if g.HomeState > 0 {
		return g.HomeState
	}
	if b := g.W.burgByID[town]; b != nil {
		return b.State
	}
	return 0
}

func ndEnemyOf(g *Game, town int) *StateData {
	st := g.W.stateByID[ndHomeState(g, town)]
	if st == nil {
		return nil
	}
	ws := append([]int(nil), st.Wars...)
	sort.Ints(ws)
	for _, id := range ws {
		if e := g.W.stateByID[id]; e != nil && e.Alive && e.Name != "" {
			return e
		}
	}
	return nil
}

func ndNearestBeast(g *Game) *MonsterData {
	if g.Cur == nil {
		return nil
	}
	here := g.W.Idx(g.Cur.WX, g.Cur.WY)
	var best *MonsterData
	bd := 1 << 30
	for i := range g.W.Monsters {
		m := &g.W.Monsters[i]
		if m.Slain || g.Slain[m.ID] || m.P < 0 || m.P >= g.W.W*g.W.H || m.Name == "" {
			continue
		}
		if d := wsCheb(g.W, m.P, here); d < bd {
			best, bd = m, d
		}
	}
	if bd > 24 {
		return nil
	}
	return best
}

func ndPeople(g *Game) []*lvPerson {
	var out []*lvPerson
	for _, p := range lvSt(g).People {
		if p.Alive {
			out = append(out, p)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Key < out[j].Key })
	return out
}

func ndAge(g *Game, p *lvPerson) int { return p.lvAge(lvYear(g)) }

// ndPickPerson finds a recurring character for a role: the saved one if still alive, else the best fit.
func ndPickPerson(g *Game, st *ndState, role string, town int, not map[string]bool) *lvPerson {
	if k := st.Recur[role]; k != "" {
		if p := lvGet(g, k); p != nil && p.Alive && !not[k] {
			return p
		}
	}
	lvEnsureTown(g, town)
	var best *lvPerson
	bs := -1 << 30
	for _, p := range ndPeople(g) {
		if not[p.Key] {
			continue
		}
		age := ndAge(g, p)
		sc := 0
		switch role {
		case "mentor":
			if age < 45 {
				continue
			}
			sc = p.Opinion + boolInt(p.Met)*30 + boolInt(p.Burg == town)*15 + boolInt(p.Role == "elder" || p.Role == "scholar" || p.Role == "priest" || p.Role == "smith")*10
		case "love":
			if age < 18 || age > 60 || p.Spouse != "" {
				continue
			}
			sc = p.Opinion + boolInt(p.Met)*30 + boolInt(p.Burg == town)*15 - abs(age-30)/2
		default: // friend, creditor
			if age < 16 {
				continue
			}
			sc = p.Opinion/2 + boolInt(p.Burg == town)*30 + boolInt(p.Met)*10
		}
		if sc > bs {
			best, bs = p, sc
		}
	}
	if best != nil && (role == "mentor" || role == "love") {
		st.Recur[role] = best.Key
	}
	return best
}

func ndPerson(g *Game, p *lvPerson, role string) ndChar {
	return ndChar{Role: role, Kind: "person", Key: p.Key, Name: p.Name}
}

// ndBind builds an arc of a template from real data, or nil when the world has nothing to bind.
func ndBind(g *Game, st *ndState, t *ndTpl) *ndArc {
	towns := ndNearTowns(g, 5)
	if len(towns) == 0 {
		return nil
	}
	a := &ndArc{Tpl: t.ID, Pl: lnPlanet(g), Flavour: t.Flavour, Start: g.Day(), Next: g.Day()}
	a.Town = towns[g.R.Int(0, mini(2, len(towns)-1))].ID
	a.Other = towns[g.R.Int(0, len(towns)-1)].ID
	if a.Other == a.Town && len(towns) > 1 {
		a.Other = towns[(g.R.Int(1, len(towns)-1))].ID
		if a.Other == a.Town {
			a.Other = towns[1].ID
		}
	}
	switch t.Needs {
	case "nem":
		n := nxSt(g).Nem
		if !nxActive(n) {
			return nil
		}
		a.Cast = append(a.Cast, ndChar{Role: "foe", Kind: "nemesis", Name: n.Name + " " + n.Epithet})
		a.Reason = strings.TrimSpace(Plain(n.Reason))
		if a.Reason != "" && !strings.HasSuffix(a.Reason, ".") {
			a.Reason += "."
		}
		if a.Reason == "" {
			a.Reason = "they were wronged once, and nobody ever answered for it."
		}
	case "rival":
		var rr *rvRival
		for _, r := range g.rv().R {
			if r.Alive && r.Name != "" {
				rr = r
				break
			}
		}
		if rr == nil {
			return nil
		}
		a.Cast = append(a.Cast, ndChar{Role: "foe", Kind: "rival", Name: strings.TrimSpace(rr.Name), ID: rr.Slot})
	case "war":
		e := ndEnemyOf(g, a.Town)
		if e == nil {
			return nil
		}
		a.Enemy = e.Name
		a.Cast = append(a.Cast, ndChar{Role: "foe", Kind: "realm", Name: "the agents of " + e.Name})
	case "mentor", "love":
		p := ndPickPerson(g, st, t.Needs, a.Town, map[string]bool{})
		if p == nil {
			return nil
		}
		if t.Needs == "love" {
			if m := st.Recur["mentor"]; m == p.Key {
				return nil
			}
		}
		a.Cast = append(a.Cast, ndPerson(g, p, "friend"))
		if p.Burg > 0 && g.W.burgByID[p.Burg] != nil {
			a.Town = p.Burg
		}
		a.Cast = append(a.Cast, ndChar{Role: "foe", Kind: "realm", Name: ndFoeName(g, a)})
	case "prophecy":
		pr := g.wy().Prophecy
		if pr == nil || pr.Verses[0] == "" {
			return nil
		}
		a.Verse = strings.TrimSpace(Plain(pr.Verses[g.R.Int(0, 2)]))
		a.Cast = append(a.Cast, ndChar{Role: "foe", Kind: "verse", Name: "the shadow of the verse"})
	case "beast":
		m := ndNearestBeast(g)
		if m == nil {
			return nil
		}
		a.Cast = append(a.Cast, ndChar{Role: "foe", Kind: "monster", Name: m.Name, ID: m.ID})
		if m.P >= 0 && m.P < g.W.W*g.W.H {
			if b := nearestBurgTo(g, m.P); b != nil {
				a.Other = b.ID
			}
		}
	case "person":
		p := ndPickPerson(g, st, "friend", a.Town, map[string]bool{})
		if p == nil {
			return nil
		}
		a.Cast = append(a.Cast, ndPerson(g, p, "friend"))
		if p.Burg > 0 && g.W.burgByID[p.Burg] != nil {
			a.Town = p.Burg
		}
		var cred *lvPerson
		for _, q := range ndPeople(g) {
			if q.Key != p.Key && q.Burg == p.Burg && ndAge(g, q) >= 25 {
				if cred == nil || q.Opinion < cred.Opinion {
					cred = q
				}
			}
		}
		if cred != nil {
			a.Cast = append(a.Cast, ndPerson(g, cred, "foe"))
		} else {
			a.Cast = append(a.Cast, ndChar{Role: "foe", Kind: "realm", Name: "the creditor"})
		}
	}
	if len(a.Cast) == 0 {
		return nil
	}
	if a.Cast[0].Role != "foe" && a.foe() == nil {
		return nil
	}
	if t.Needs != "mentor" && t.Needs != "love" && t.Needs != "person" {
		// optional recurring friend (a mentor or a love interest, if you have one) rides along
		for _, role := range []string{"mentor", "love"} {
			if k := st.Recur[role]; k != "" {
				if p := lvGet(g, k); p != nil && p.Alive && g.R.Chance(0.5) {
					a.Cast = append(a.Cast, ndPerson(g, p, "friend"))
					break
				}
			}
		}
	}
	return a
}

func nearestBurgTo(g *Game, p int) *BurgData {
	var best *BurgData
	bd := 1 << 30
	for i := range g.W.Burgs {
		b := &g.W.Burgs[i]
		if !b.Alive || b.P < 0 {
			continue
		}
		if d := wsCheb(g.W, b.P, p); d < bd {
			best, bd = b, d
		}
	}
	return best
}

func ndFoeName(g *Game, a *ndArc) string {
	if e := ndEnemyOf(g, a.Town); e != nil {
		return "the agents of " + e.Name
	}
	return "the hired blades"
}

func (a *ndArc) char(role string) *ndChar {
	for i := range a.Cast {
		if a.Cast[i].Role == role {
			return &a.Cast[i]
		}
	}
	return nil
}
func (a *ndArc) foe() *ndChar    { return a.char("foe") }
func (a *ndArc) friend() *ndChar { return a.char("friend") }

func ndCharName(g *Game, c *ndChar) string {
	if c == nil {
		return "someone"
	}
	if c.Key != "" {
		if p := lvGet(g, c.Key); p != nil {
			return p.Name
		}
	}
	return c.Name
}

// ---------------------------------------------------------------- text

func ndVoice(g *Game) ndVoc {
	if v, ok := ndVocs[g.Genre]; ok {
		return v
	}
	return ndVocs["fantasy"]
}

func ndCap(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func ndFill(g *Game, a *ndArc, s string) string {
	v := ndVoice(g)
	realm := "the realm"
	if st := g.W.stateByID[ndHomeState(g, a.Town)]; st != nil {
		realm = st.Name
	}
	enemy := a.Enemy
	if enemy == "" {
		enemy = "the enemy"
	}
	friend := "an old friend"
	if f := a.friend(); f != nil {
		friend = ndCharName(g, f)
	}
	foe := "your enemy"
	if f := a.foe(); f != nil {
		foe = ndCharName(g, f)
	}
	rep := strings.NewReplacer("{Msgr}", ndCap(v.Msgr), "{msgr}", v.Msgr, "{letter}", v.Letter, "{fest}", v.Fest,
		"{Agents}", ndCap(v.Agents), "{agents}", v.Agents, "{Omen}", ndCap(v.Omen), "{omen}", v.Omen, "{mood}", v.Mood,
		"{Rumour}", ndCap(v.Rumour), "{foe}", foe, "{friend}", friend, "{town}", ndBurgName(g, a.Town), "{other}", ndBurgName(g, a.Other),
		"{realm}", realm, "{enemy}", enemy, "{verse}", a.Verse, "{reason}", a.Reason, "{me}", g.P.Name)
	return rep.Replace(s)
}

func (a *ndArc) title(g *Game) string {
	for _, t := range ndTpls {
		if t.ID == a.Tpl {
			return ndFill(g, a, t.Title)
		}
	}
	return a.Tpl
}

func ndTplOf(id string) *ndTpl {
	for _, t := range ndTpls {
		if t.ID == id {
			return t
		}
	}
	return nil
}

// ---------------------------------------------------------------- the director

func ndTurn(g *Game) {
	if g.W == nil || g.P == nil || g.Cur == nil || g.Dead || g.Mode == "dead" {
		return
	}
	st := ndSt(g)
	if !st.Init {
		st.Init, st.LastBeat, st.Tension = true, g.Turn, 12
		st.Gold = g.P.Gold
		for _, q := range g.Quests {
			if q.Done {
				st.Quests++
			}
		}
		st.Kar = crKs(g).Ax
	}
	t := st.Tension
	danger := g.nearHostile()
	town := g.Cur.Kind == "town"
	if danger {
		t += 0.07
		st.Combat = 8
	} else {
		if st.Combat > 0 {
			st.Combat--
		}
		if town {
			t -= 0.05
		} else {
			t -= 0.015
		}
		if t > 35 {
			t -= 0.01
		}
	}
	if g.P.MaxHP > 0 && g.P.HP*100 < g.P.MaxHP*35 {
		t += 0.12
	}
	if since := g.Turn - st.LastBeat; since > 720 && t < 35 {
		t += 0.02
	}
	if g.Turn%5 == 0 { // cheap polls
		if d := g.P.Gold - st.Gold; d >= 50 {
			t += minf(6, float64(d)/50)
		}
		st.Gold = g.P.Gold
		n := 0
		for _, q := range g.Quests {
			if q.Done {
				n++
			}
		}
		if n > st.Quests {
			t += 10 * float64(n-st.Quests)
		}
		st.Quests = n
		ax := crKs(g).Ax
		sum := 0
		for i := range ax {
			sum += abs(ax[i] - st.Kar[i])
		}
		if sum > 0 {
			t += minf(8, float64(sum)/2)
		}
		st.Kar = ax
	}
	st.Tension = clampf(t, 0, 100)
	if st.Tension > st.Peak {
		st.Peak = st.Tension
	}
	if g.Turn%10 == 0 {
		ndDirect(g, st, danger)
	}
}

func ndCanHost(zn *Zone) bool {
	return zn != nil && zn.Kind != "town" && zn.Kind != "sea" && !zn.Underwater && !zn.Sky
}

func ndReady(a *ndArc, t float64, today int) bool {
	if a.Await || a.Next > today {
		return false
	}
	switch a.Stage {
	case 0, 1:
		return t <= 70
	case 2:
		return t >= 20 && t <= 85
	case 3:
		return t >= 35
	case 4:
		return t >= 50 || today-a.Last >= 6
	}
	return true
}

func ndDirect(g *Game, st *ndState, danger bool) {
	today := g.Day()
	pl := lnPlanet(g)
	// expire stale pending beats
	kept := st.Pend[:0]
	for _, p := range st.Pend {
		if today-p.Day > 5 {
			if p.Kind == "boss" {
				for _, a := range st.Arcs {
					if a.ID == p.Arc {
						ndResolve(g, st, a, "mystery")
					}
				}
			}
			continue
		}
		kept = append(kept, p)
	}
	st.Pend = kept
	ndWatch(g, st)
	if g.Mode == "main" && !danger {
		ndSpawnPending(g, g.Cur)
	}
	if g.Mode != "main" || danger || len(st.Pend) > 0 {
		return
	}
	since := g.Turn - st.LastBeat
	if since < ndGap {
		return
	}
	if !(since >= 720 || g.R.Chance(0.08)) {
		return
	}
	t := st.Tension
	if t >= 78 {
		ndBreather(g, st)
		return
	}
	var elig []*ndArc
	for _, a := range st.Arcs {
		if a.Pl == pl && ndReady(a, t, today) {
			elig = append(elig, a)
		}
	}
	sort.Slice(elig, func(i, j int) bool { return elig[i].Last < elig[j].Last })
	if len(elig) > 0 && (len(st.Arcs) >= 3 || g.R.Chance(0.7)) {
		ndFire(g, st, elig[0])
		return
	}
	if len(st.Arcs) < 3 && (len(st.Arcs) == 0 || t < 50) {
		if a := ndStart(g, st); a != nil {
			ndFire(g, st, a)
		}
	}
}

// ndStart picks a template, biased toward ending flavours not seen yet.
func ndStart(g *Game, st *ndState) *ndArc {
	seen := ndSeenAll(g)
	active := map[string]bool{}
	for _, a := range st.Arcs {
		active[a.Tpl] = true
	}
	recent := map[string]bool{}
	for _, d := range st.Done {
		if g.Day()-d.Day < 10 {
			recent[d.Tpl] = true
		}
	}
	type cand struct {
		t *ndTpl
		a *ndArc
		w int
	}
	var cs []cand
	tot := 0
	for _, t := range ndTpls {
		if active[t.ID] {
			continue
		}
		a := ndBind(g, st, t)
		if a == nil {
			continue
		}
		w := t.W * 100 / (1 + 2*seen[t.Flavour])
		if recent[t.ID] {
			w /= 4
		}
		if w < 1 {
			w = 1
		}
		cs = append(cs, cand{t, a, w})
		tot += w
	}
	if tot == 0 {
		return nil
	}
	x := g.R.Int(0, tot-1)
	for _, c := range cs {
		if x < c.w {
			st.Next++
			c.a.ID = st.Next
			c.a.Title = c.a.title(g)
			st.Arcs = append(st.Arcs, c.a)
			return c.a
		}
		x -= c.w
	}
	return nil
}

// ndWatch: bound characters who died or were resolved elsewhere change the arc's course.
func ndWatch(g *Game, st *ndState) {
	pl := lnPlanet(g)
	for _, a := range st.Arcs {
		if a.Pl != pl || a.Stage >= 5 {
			continue
		}
		if f := a.friend(); f != nil && f.Key != "" {
			if p := lvGet(g, f.Key); p != nil && !p.Alive {
				a.Flavour = "tragedy"
				a.Stage, a.Await, a.Next = 5, false, g.Day()
				g.Msg("{{R|" + ndCap(ndCharName(g, f)) + " is dead. The story you were in has turned.}}")
				continue
			}
		}
		foe := a.foe()
		if foe == nil {
			continue
		}
		over := false
		switch foe.Kind {
		case "nemesis":
			n := nxSt(g).Nem
			over = n == nil || (n.Status != "hunt" && n.Status != "truce")
		case "rival":
			over = true
			for _, r := range g.rv().R {
				if r.Slot == foe.ID && r.Alive {
					over = false
				}
			}
		case "monster":
			over = g.Slain[foe.ID]
		}
		if over && a.Stage >= 2 && a.Stage < 5 {
			a.Stage, a.Await, a.Next = 5, false, g.Day()
		}
	}
}

func ndPush(st *ndState, d float64) { st.Tension = clampf(st.Tension+d, 0, 100) }

var ndStagePush = [6]float64{8, 10, 14, 18, 22, -28}

// ndFire delivers the arc's next beat.
func ndFire(g *Game, st *ndState, a *ndArc) {
	t := ndTplOf(a.Tpl)
	if t == nil {
		return
	}
	stage := mini(a.Stage, 5)
	def := t.Stages[stage]
	lines := def.L
	if stage == 5 && a.Flavour == "tragedy" && len(t.Tragic) > 0 && a.friend() != nil {
		def = ndStageDef{K: "letter", Sub: "note", From: "town", L: t.Tragic}
		lines = t.Tragic
	}
	text := ndFill(g, a, lines[g.R.Int(0, len(lines)-1)])
	day := g.Day()
	rec := func(kind, tx string) {
		st.Beats = append(st.Beats, ndBeat{Day: day, Arc: a.Title, Stage: stage, Kind: kind, Text: tx})
		if len(st.Beats) > 40 {
			st.Beats = st.Beats[len(st.Beats)-40:]
		}
	}
	switch def.K {
	case "letter":
		ndSendLetter(g, a, def, text)
		g.Msg("{{c|" + ndCap(ndFill(g, a, "{msgr}")) + " is looking for you with a " + ndVoice(g).Letter + ".}}")
		rec("letter", text)
	case "rumour":
		lvSt(g).addNews(day, a.Town, "%s", text)
		g.Msg("{{c|Rumour:}} " + text)
		rec("rumour", text)
	case "reveal":
		g.Msg("{{M|" + text + "}}")
		if g.Cur != nil {
			g.deed("event", Plain(text), 0, 0)
		}
		g.gainXP(10 + g.P.Level*2)
		rec("twist", text)
	case "ambush", "boss":
		p := &ndPend{Arc: a.ID, Kind: def.K, Text: text, Day: day, Spawn: t.Spawn}
		if f := a.foe(); f != nil {
			p.Foe = ndCharName(g, f)
		}
		st.Pend = append(st.Pend, p)
		rec(def.K, text)
		if def.K == "boss" {
			a.Await = true
		}
		g.Msg("{{R|Something is closing in on you. Stay alert.}}")
		ndSpawnPending(g, g.Cur)
	}
	if stage == 5 {
		g.gainXP(30 + g.P.Level*8)
		ndResolve(g, st, a, a.Flavour)
		ndPush(st, ndStagePush[5])
	} else {
		a.Stage++
		ndPush(st, ndStagePush[stage])
	}
	a.Beats++
	a.Last = day
	a.Next = day + 1
	st.LastBeat = g.Turn
}

func ndSendLetter(g *Game, a *ndArc, def ndStageDef, text string) {
	l := &lnLetter{Kind: def.Sub, Text: text, Planet: lnPlanet(g), Place: ndBurgName(g, a.Town)}
	switch def.From {
	case "friend":
		if f := a.friend(); f != nil {
			l.Name, l.From = ndCharName(g, f), f.Key
			if p := lvGet(g, f.Key); p != nil {
				l.Role = p.Role
			}
		}
	case "foe":
		l.Name = ndCharName(g, a.foe())
	}
	if l.Name == "" {
		l.Name, l.Role = "The reeve of "+ndBurgName(g, a.Town), "reeve"
	}
	if def.Sub == "" {
		l.Kind = "note"
	}
	if a.Stage == 5 && a.Flavour != "tragedy" {
		l.Coin = 20 + g.P.Level*10
	}
	lnQueue(g, l, a.Town, g.R, g.Day())
}

func ndResolve(g *Game, st *ndState, a *ndArc, flavour string) {
	if flavour == "" {
		flavour = a.Flavour
	}
	a.Flavour = flavour
	st.Seen[flavour]++
	st.Done = append(st.Done, &ndDone{Title: a.Title, Flavour: flavour, Day: g.Day(), Tpl: a.Tpl})
	if len(st.Done) > 12 {
		st.Done = st.Done[len(st.Done)-12:]
	}
	for i, x := range st.Arcs {
		if x == a {
			st.Arcs = append(st.Arcs[:i], st.Arcs[i+1:]...)
			break
		}
	}
	if g.Cur != nil {
		g.deed("event", fmt.Sprintf("%s ended in %s.", a.Title, ndFlavours[flavour]), 0, 0)
	}
}

func ndBreather(g *Game, st *ndState) {
	var a *ndArc
	for _, x := range st.Arcs {
		a = x
		break
	}
	town := ndNearTowns(g, 3)
	if len(town) == 0 {
		return
	}
	b := town[g.R.Int(0, len(town)-1)]
	v := ndVoice(g)
	var text string
	switch g.R.Int(0, 2) {
	case 0:
		text = fmt.Sprintf("%s holds its %s. For one night nobody mentions the trouble on the roads.", b.Name, v.Fest)
	case 1:
		text = fmt.Sprintf("The people of %s have called a %s. Somebody has saved you a seat; the rest can wait a night.", b.Name, v.Fest)
	default:
		text = fmt.Sprintf("A %s in %s: lanterns, noise, bread. Whatever is coming can wait until morning.", v.Fest, b.Name)
	}
	aid := 0
	if a != nil {
		aid = a.ID
	}
	st.Pend = append(st.Pend, &ndPend{Arc: aid, Kind: "festival", Text: text, Day: g.Day(), Burg: b.ID})
	st.Beats = append(st.Beats, ndBeat{Day: g.Day(), Arc: "A calm", Kind: "breather", Text: text})
	if len(st.Beats) > 40 {
		st.Beats = st.Beats[len(st.Beats)-40:]
	}
	g.Msg("{{c|Word reaches you:}} " + text)
	ndPush(st, -15)
	st.LastBeat = g.Turn
	ndSpawnPending(g, g.Cur)
}

// ---------------------------------------------------------------- pending beats on arrival

func ndEnter(g *Game, zn *Zone) {
	if g.W == nil || g.P == nil || g.Mode == "dead" {
		return
	}
	ndSpawnPending(g, zn)
}

func ndSpawnPending(g *Game, zn *Zone) {
	if zn == nil || g.P == nil {
		return
	}
	st := ndSt(g)
	if len(st.Pend) == 0 {
		return
	}
	kept := st.Pend[:0]
	for _, p := range st.Pend {
		done := false
		switch p.Kind {
		case "festival":
			if zn.Kind == "town" && (p.Burg == 0 || zn.Burg == p.Burg || g.R.Chance(0.3)) {
				done = true
				g.Msg("{{G|" + p.Text + "}}")
				if g.P.HP < g.P.MaxHP {
					g.P.HP = mini(g.P.MaxHP, g.P.HP+g.P.MaxHP/4)
				}
				g.gainXP(8 + g.P.Level)
				ndPush(st, -15)
				lvSt(g).addNews(g.Day(), zn.Burg, "%s", p.Text)
			}
		case "ambush", "boss":
			if ndCanHost(zn) && p.Day <= g.Day() {
				done = ndAmbush(g, zn, p)
			}
		}
		if !done {
			kept = append(kept, p)
		}
	}
	st.Pend = kept
}

func ndAmbush(g *Game, zn *Zone, p *ndPend) bool {
	r := NewRng(fmt.Sprintf("%s:nd:amb:%d:%d:%d", g.Seed, p.Arc, p.Day, g.Turn/60))
	boss := p.Kind == "boss"
	n := clampi(2+g.P.Level/8, 2, 4)
	if boss {
		n = clampi(2+g.P.Level/10, 2, 3)
	}
	far := func(x, y int) bool { return abs(x-g.P.X) >= 6 || abs(y-g.P.Y) >= 4 }
	var spawned int
	for i := 0; i <= n; i++ {
		x, y, ok := zn.FreeSpot(r, far)
		if !ok {
			if x, y, ok = zn.FreeSpot(r, nil); !ok {
				break
			}
		}
		tpl := Pick(r, p.Spawn)
		lvl := maxi(1, g.P.Level-1+boolInt(boss && i == 0)*3)
		a := g.Spawn(tpl, lvl, r)
		a.X, a.Y, a.HomeX, a.HomeY = x, y, x, y
		a.Angry, a.AI = true, "hostile"
		a.Set("nd")
		a.Set(fmt.Sprintf("nd#%d", p.Arc))
		if boss && i == 0 && p.Foe != "" {
			a.Name, a.Proper = p.Foe, true
			a.MaxHP *= 2
			a.HP = a.MaxHP
			a.Set("ndboss")
		}
		zn.Actors = append(zn.Actors, a)
		spawned++
	}
	if spawned == 0 {
		return false
	}
	g.Msg("{{R|" + p.Text + "}}")
	ndPush(ndSt(g), 6)
	return true
}

// ndKill: kills feed tension; the arc's boss ends the climax, a companion's fall hits hard.
func ndKill(g *Game, d, src *Actor) {
	if g.W == nil || d == nil || g.P == nil {
		return
	}
	st := ndSt(g)
	if d.Comp != nil {
		ndPush(st, 18)
		return
	}
	if src == nil || src == g.P {
		ndPush(st, 2)
		if d.Level >= g.P.Level {
			ndPush(st, 3)
		}
	}
	if !d.Has("ndboss") {
		return
	}
	for k := range d.Flags {
		var id int
		if _, err := fmt.Sscanf(k, "nd#%d", &id); err != nil {
			continue
		}
		for _, a := range st.Arcs {
			if a.ID == id && a.Await {
				a.Await, a.Stage, a.Next = false, 5, g.Day()
				st.LastBeat = g.Turn - ndGap + 30
				ndPush(st, 8)
				g.Msg("{{G|The worst of it is behind you. The road seems to breathe again.}}")
			}
		}
	}
}

// ---------------------------------------------------------------- screen

func ndKeys(g *Game, k string) {
	switch k {
	case "Escape", "q", "Enter":
		g.Mode = "main"
	case "Tab", "ArrowRight", "l":
		g.Tab, g.Sel = (g.Tab+1)%3, 0
	case "ArrowLeft", "h":
		g.Tab, g.Sel = (g.Tab+2)%3, 0
	case "ArrowDown", "j":
		g.Sel++
	case "ArrowUp", "k":
		g.Sel = maxi(0, g.Sel-1)
	}
}

func ndMeter(t float64) string {
	n := int(t * 30 / 100)
	var b strings.Builder
	for i := 0; i < 30; i++ {
		c := "="
		col := "G"
		if i >= 12 {
			col = "Y"
		}
		if i >= 21 {
			col = "R"
		}
		if i >= n {
			c, col = "-", "K"
		}
		b.WriteString("{{" + col + "|" + c + "}}")
	}
	return b.String()
}

func ndBand(t float64) string {
	switch {
	case t < 25:
		return "{{G|quiet}}"
	case t < 50:
		return "{{Y|stirring}}"
	case t < 75:
		return "{{y|tense}}"
	}
	return "{{R|breaking}}"
}

func (g *Game) ndDraw() {
	sc := g.sc
	st := ndSt(g)
	x, y, w, h := g.panel("Story")
	tabs := []string{"Arcs", "Beats", "Endings"}
	tx := x
	for i, t := range tabs {
		c := "K"
		if i == g.Tab {
			c = "W"
		}
		tx = sc.TextB(tx, y, fmt.Sprintf("{{%s|[%s]}} ", c, t), CGray, CBgDark, x+w)
	}
	sc.TextB(tx+2, y, "{{K|Tab switch · Up/Down scroll · Esc}}", CGray, CBgDark, x+w)
	y += 2
	sc.TextB(x, y, fmt.Sprintf("Tension %s {{W|%d}}  %s", ndMeter(st.Tension), int(st.Tension), ndBand(st.Tension)), CGray, CBgDark, x+w)
	y += 2
	var lines []string
	add := func(t string) { lines = append(lines, Wrap(t, w)...) }
	switch g.Tab {
	case 0:
		if len(st.Arcs) == 0 {
			add("{{K|No story is running yet. The road is quiet. Keep walking.}}")
		}
		for _, a := range st.Arcs {
			add("{{W|" + a.Title + "}} {{K|(" + ndFlavours[a.Flavour] + ")}}")
			var tr strings.Builder
			for i := 0; i < 6; i++ {
				switch {
				case i < a.Stage:
					tr.WriteString("{{G|o}}")
				case i == a.Stage:
					tr.WriteString("{{Y|O}}")
				default:
					tr.WriteString("{{K|.}}")
				}
				tr.WriteString(" ")
			}
			add("  " + tr.String() + "  " + ndStageNames[mini(a.Stage, 5)] + ndAwait(a))
			for _, c := range a.Cast {
				add(fmt.Sprintf("  {{K|%s:}} %s", c.Role, ndCharName(g, &c)))
			}
			add("  {{K|centred on}} " + ndBurgName(g, a.Town))
			add("")
		}
		for _, p := range st.Pend {
			add("{{R|Waiting for you:}} " + map[string]string{"ambush": "an ambush", "boss": "a reckoning", "festival": "a festival"}[p.Kind])
		}
	case 1:
		if len(st.Beats) == 0 {
			add("{{K|Nothing has happened yet that the story would remember.}}")
		}
		for i := len(st.Beats) - 1; i >= 0; i-- {
			b := st.Beats[i]
			add(fmt.Sprintf("{{K|Day %d, %s (%s):}} %s", b.Day+1, b.Arc, b.Kind, b.Text))
		}
	default:
		seen := ndSeenAll(g)
		add("Endings are shaped by what you have not yet seen. Each flavour below counts this run and your past lives in this world.")
		add("")
		for _, f := range ndFlavourOrder {
			mark := "{{K|-}}"
			if seen[f] > 0 {
				mark = fmt.Sprintf("{{G|%d}}", seen[f])
			}
			add(fmt.Sprintf("%s  {{W|%-10s}} {{K|%s}}", mark, f, ndFlavours[f]))
		}
		if lean := ndEndingLean(g); lean != "" {
			add("")
			add("Heading toward: {{M|" + ndFlavours[lean] + "}}")
		}
	}
	if g.Sel > len(lines)-1 {
		g.Sel = maxi(0, len(lines)-1)
	}
	for i := g.Sel; i < len(lines) && y < sc2Bottom(x, h, g); i++ {
		sc.TextB(x, y, lines[i], CGray, CBgDark, x+w)
		y++
	}
}

func ndAwait(a *ndArc) string {
	if a.Await {
		return "  {{R|(it waits for you)}}"
	}
	return ""
}
