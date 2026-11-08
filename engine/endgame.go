package engine

import (
	"fmt"
	"sort"
	"strings"
)

// endgame.go (prefix eg): the world-scale threat. Once the player is strong and the errand is
// under way, a threat chosen from the world's own data (monsters, rifts, hazards, wars, the
// abyss, or a genre doom) rises and spreads over parasangs on the worldsim day clock. State in
// g.ext("eg") (planet-local). Finale, endings and screens live in endgame_play.go.

type egRelic struct {
	Art  int `json:"a"`
	Burg int `json:"b"`
}

type egState struct {
	Phase     int             `json:"ph"` // 0 dormant, 1 risen, 3 resolved
	Kind      string          `json:"k"`
	Name      string          `json:"n"`
	BossName  string          `json:"bn"`
	Origin    int             `json:"o"`
	MonID     int             `json:"mon,omitempty"`
	RiseDay   int             `json:"rd"`
	LastDay   int             `json:"ld"`
	Occ       []int           `json:"occ"`
	Lib       map[int]int     `json:"lib"`
	Kills     map[int]int     `json:"kl"`
	Spawned   map[string]int  `json:"sp"`
	Siege     map[int]int     `json:"sg"`
	Fallen    []int           `json:"fa"`
	Ref       map[int]int     `json:"rf"`
	RefDone   map[int]int     `json:"rfd"`
	Allies    []string        `json:"al"`
	Relics    []egRelic       `json:"rl"`
	Held      map[int]bool    `json:"hd"`
	Keys      map[string]bool `json:"ky"`
	Finale    int             `json:"fi"` // 0 none, 1 fighting, 2 boss yields, 3 over
	BossPhase int             `json:"bp"`
	Ending    string          `json:"end,omitempty"`
	EndDay    int             `json:"ed,omitempty"`
	Freed     int             `json:"fr"`
	Mile      int             `json:"mi"`
	LastTithe int             `json:"lt,omitempty"`
	MoveTurn  int             `json:"mt,omitempty"`
	Log       []string        `json:"log"`

	occSet map[int]bool
	land   int
}

func init() {
	turnHooks = append(turnHooks, egTurn)
	zoneHooks = append(zoneHooks, egZone)
	enterHooks = append(enterHooks, egEnter)
	killHooks = append(killHooks, egKill)
	talkHooks = append(talkHooks, egTalk)
	sideHooks = append(sideHooks, egSide)
	mainKeys["E"] = func(g *Game) { g.Mode, g.Sel, g.Tab = "egwar", 0, 0 }
	modeKeys["egwar"] = egWarKeys
	modeDraws["egwar"] = func(g *Game) { g.egDrawWar() }
	modeKeys["egend"] = egEndKeys
	modeDraws["egend"] = func(g *Game) { g.egDrawEnd() }
}

func (g *Game) eg() *egState {
	st := g.ext("eg", func() any { return &egState{} }).(*egState)
	if st.Lib == nil {
		st.Lib = map[int]int{}
	}
	if st.Kills == nil {
		st.Kills = map[int]int{}
	}
	if st.Spawned == nil {
		st.Spawned = map[string]int{}
	}
	if st.Siege == nil {
		st.Siege = map[int]int{}
	}
	if st.Ref == nil {
		st.Ref = map[int]int{}
	}
	if st.RefDone == nil {
		st.RefDone = map[int]int{}
	}
	if st.Held == nil {
		st.Held = map[int]bool{}
	}
	if st.Keys == nil {
		st.Keys = map[string]bool{}
	}
	return st
}

func (st *egState) def() *egDef { return egDefs[st.Kind] }

func (st *egState) set() map[int]bool {
	if st.occSet == nil || len(st.occSet) != len(st.Occ) {
		st.occSet = make(map[int]bool, len(st.Occ))
		for _, p := range st.Occ {
			st.occSet[p] = true
		}
	}
	return st.occSet
}

func (st *egState) occupied(p int) bool { return st.Phase != 0 && st.set()[p] }

func (st *egState) add(p int) {
	if !st.set()[p] {
		st.Occ = append(st.Occ, p)
		st.occSet[p] = true
	}
}

func (st *egState) remove(p int) {
	if !st.set()[p] {
		return
	}
	for i, q := range st.Occ {
		if q == p {
			st.Occ = append(st.Occ[:i], st.Occ[i+1:]...)
			break
		}
	}
	delete(st.occSet, p)
}

func (st *egState) fallen(b int) bool {
	for _, f := range st.Fallen {
		if f == b {
			return true
		}
	}
	return false
}

func (st *egState) logf(s string) {
	st.Log = append(st.Log, s)
	if len(st.Log) > 30 {
		st.Log = st.Log[len(st.Log)-30:]
	}
}

// ---- helpers ----

func egH(g *Game, parts ...any) uint64 { return HashStr(g.Seed + ":eg:" + fmt.Sprint(parts...)) }

func egPickS(g *Game, key string, a []string) string {
	if len(a) == 0 {
		return ""
	}
	return a[egH(g, key)%uint64(len(a))]
}

func egDay(g *Game) int { return wsDay(g.wsHour()) }

func (g *Game) egHere() int {
	w := g.W
	if g.OnWorld || g.Cur == nil {
		if w.In(g.WX, g.WY) {
			return w.Idx(g.WX, g.WY)
		}
		return 0
	}
	if w.In(g.Cur.WX, g.Cur.WY) {
		return w.Idx(g.Cur.WX, g.Cur.WY)
	}
	return 0
}

// egNear is the distance to the nearest occupied parasang and which one (999, -1 if none).
func (g *Game) egNear(p int) (int, int) {
	st := g.eg()
	best, bq := 999, -1
	for _, q := range st.Occ {
		if d := wsCheb(g.W, p, q); d < best || d == best && q < bq {
			best, bq = d, q
		}
	}
	return best, bq
}

func (g *Game) egLandCount() int {
	st := g.eg()
	if st.land == 0 {
		for p := range g.W.Biome {
			if !g.W.IsWater(p) {
				st.land++
			}
		}
	}
	return st.land
}

// egLand picks a land parasang by hash, between minD and maxD (0 = any) from the start.
func (g *Game) egLand(minD, maxD int, key string) int {
	w := g.W
	start := w.Idx(clampi(g.StartX, 0, w.W-1), clampi(g.StartY, 0, w.H-1))
	n := uint64(len(w.Biome))
	for k := 0; k < 800; k++ {
		p := int(egH(g, "land:", key, k) % n)
		if w.IsWater(p) || w.Burg[p] > 0 {
			continue
		}
		d := wsCheb(w, p, start)
		if d >= minD && (maxD == 0 || d <= maxD) {
			return p
		}
		if k == 400 {
			minD /= 2
		}
	}
	for p := range w.Biome {
		if !w.IsWater(p) {
			return p
		}
	}
	return 0
}

// ---- choosing the threat ----

var egOrder = []string{"titan", "rift", "plague", "war", "abyss", "ai", "gene", "winter", "bomb", "dust", "sunder"}

func (g *Game) egMonster() *MonsterData {
	var best *MonsterData
	for i := range g.W.Monsters {
		m := &g.W.Monsters[i]
		if m.Slain || g.Slain[m.ID] || m.P < 0 || m.P >= len(g.W.Biome) || g.W.IsWater(m.P) || m.Layer != "" && m.Layer != "surface" {
			continue
		}
		if best == nil || m.Power > best.Power || m.Power == best.Power && m.ID < best.ID {
			best = m
		}
	}
	return best
}

func (g *Game) egRiftBurg() *BurgData {
	w := g.W
	var ids []int
	for id := range unUni(w).ends[w.uniIdx] {
		if b := w.burgByID[id]; b != nil && b.Alive && (b.Layer == "" || b.Layer == "surface") && !w.IsWater(b.P) {
			ids = append(ids, id)
		}
	}
	if len(ids) == 0 {
		return nil
	}
	sort.Ints(ids)
	return w.burgByID[ids[egH(g, "rift")%uint64(len(ids))]]
}

func (g *Game) egMega(kind string) *MegaData {
	for i := range g.W.Megas {
		if g.W.Megas[i].Kind == kind && g.W.Megas[i].P >= 0 && g.W.Megas[i].P < len(g.W.Biome) {
			return &g.W.Megas[i]
		}
	}
	return nil
}

func (g *Game) egWarStates() []*StateData {
	var out []*StateData
	for i := range g.W.States {
		s := &g.W.States[i]
		if s.Alive && len(g.warsOf(s.ID)) > 0 {
			out = append(out, s)
		}
	}
	return out
}

func (g *Game) egPick() string {
	w := g.W
	type cand struct {
		k string
		w float64
	}
	var cs []cand
	alive := 0
	for _, s := range w.States {
		if s.Alive {
			alive++
		}
	}
	for _, k := range egOrder {
		d := egDefs[k]
		wt := 0.0
		switch k {
		case "titan":
			if g.egMonster() != nil {
				wt = 2
			}
		case "rift":
			if g.egRiftBurg() != nil {
				wt = 2
			}
		case "plague":
			wt = 1.5
			if w.Hazards != nil && len(w.Hazards.Outbreaks) > 0 {
				wt = 3
			}
		case "war":
			if alive >= 2 {
				wt = 1.5
				if len(g.egWarStates()) > 0 {
					wt = 3
				}
			}
		case "abyss":
			if g.egMega("abyss") != nil {
				wt = 2
			}
		}
		for _, gn := range d.Genres {
			if gn == w.Genre {
				wt = 4
			}
		}
		if wt > 0 {
			cs = append(cs, cand{k, wt})
		}
	}
	if len(cs) == 0 {
		return "plague"
	}
	tot := 0.0
	for _, c := range cs {
		tot += c.w
	}
	roll := float64(egH(g, "pick")%10000) / 10000 * tot
	for _, c := range cs {
		if roll < c.w {
			return c.k
		}
		roll -= c.w
	}
	return cs[len(cs)-1].k
}

// ---- threshold ----

// egReady is the single mid-game threshold for a threat to rise.
func (g *Game) egReady() bool {
	if g.Dead || g.Cur == nil || g.P.Level < 12 || g.Turn/DayLen < 60 {
		return false
	}
	sp := g.qcSpine()
	return sp.Started && (sp.Finished || sp.Step >= 2) || g.P.Level >= 18
}

func (g *Game) egThresholdLines() []string {
	sp := g.qcSpine()
	ok := func(b bool) string {
		if b {
			return "{{G|ok}}"
		}
		return "{{R|not yet}}"
	}
	prog := sp.Started && (sp.Finished || sp.Step >= 2) || g.P.Level >= 18
	return []string{
		fmt.Sprintf("Level %d of 12: %s", g.P.Level, ok(g.P.Level >= 12)),
		fmt.Sprintf("Days walked %d of 60: %s", g.Turn/DayLen, ok(g.Turn/DayLen >= 60)),
		"The great errand under way (or level 18): " + ok(prog),
	}
}

func (g *Game) egRise() {
	st := g.eg()
	w := g.W
	st.Kind = g.egPick()
	d := st.def()
	day := egDay(g)
	var mon *MonsterData
	switch st.Kind {
	case "titan":
		mon = g.egMonster()
		st.Origin, st.MonID = mon.P, mon.ID
		st.Name = "The Awakening of " + strings.TrimPrefix(mon.Name, "The ")
		st.BossName = mon.Name
	case "rift":
		b := g.egRiftBurg()
		st.Origin = b.P
	case "war":
		ws := g.egWarStates()
		if len(ws) > 0 {
			s := ws[egH(g, "warstate")%uint64(len(ws))]
			if b := w.burgByID[s.Capital]; b != nil && !w.IsWater(b.P) {
				st.Origin = b.P
			} else {
				st.Origin = g.egLand(8, 0, "war")
			}
			if f := g.warsOf(s.ID); len(f) > 0 {
				st.Name = fmt.Sprintf("The War of %s and %s", s.Name, g.realmName(f[0]))
			}
		} else {
			st.Origin = g.egLand(8, 0, "war")
		}
	case "abyss":
		st.Origin = g.egMega("abyss").P
	case "plague":
		st.Origin = -1
		if w.Hazards != nil && len(w.Hazards.Outbreaks) > 0 {
			ob := w.Hazards.Outbreaks[egH(g, "ob")%uint64(len(w.Hazards.Outbreaks))]
			for _, p := range ob.Parasangs {
				if p >= 0 && p < len(w.Biome) && !w.IsWater(p) {
					st.Origin = p
					break
				}
			}
			if ob.Name != "" {
				st.Name = "The " + strings.TrimPrefix(ob.Name, "The ")
			}
		}
	}
	if st.Origin <= 0 && st.Kind != "titan" || st.Origin < 0 || st.Origin >= len(w.Biome) || w.IsWater(st.Origin) {
		st.Origin = g.egLand(maxi(8, mini(w.W, w.H)/4), 0, st.Kind)
	}
	if st.Name == "" {
		st.Name = egPickS(g, "name", d.Names)
	}
	if st.BossName == "" {
		st.BossName = egPickS(g, "boss", d.Boss)
	}
	st.Phase, st.RiseDay, st.LastDay = 1, day, day
	st.Occ, st.occSet = nil, nil
	st.add(st.Origin)
	st.Relics = g.egChooseRelics()
	place := g.wsPlace(st.Origin)
	rise := fmt.Sprintf(d.Rise, place)
	st.logf(fmt.Sprintf("Day %d: %s", day, rise))
	g.deed("endgame", fmt.Sprintf("%s rose near %s while the wanderer %s walked the land.", st.Name, place, g.P.Name), st.MonID, 0)
	g.Msg("{{R|" + st.Name + " has risen!}} {{K|(E: war room)}}")
	g.showPopup(st.Name, rise+"\n\nThe realms are not ready and will not unite by themselves. Gather the realms and orders, recover the relics of the old rulers, take the keys of the great structures, and march on the Heart of the threat. The war room (E) tracks everything.\n\n"+d.Weak)
}

// egChooseRelics picks real world artifacts lying in ruins (crypts and rubble), famous first.
func (g *Game) egChooseRelics() []egRelic {
	w := g.W
	used := map[int]bool{}
	for _, c := range g.qcSpine().Chaps {
		if c.ArtID > 0 {
			used[c.ArtID] = true
		}
	}
	type cand struct {
		a *ArtifactData
		b *BurgData
	}
	var cs []cand
	for i := range w.Burgs {
		b := &w.Burgs[i]
		if b.Alive || b.Layer != "surface" || w.Burg[b.P] != b.ID {
			continue
		}
		for _, a := range qcCryptRelics(g, b) {
			cs = append(cs, cand{a, b})
		}
		for j := range w.Artifacts {
			a := &w.Artifacts[j]
			if a.HolderType == "burg" && a.HolderID == b.ID && !g.Taken[a.ID] {
				cs = append(cs, cand{a, b})
			}
		}
	}
	sort.SliceStable(cs, func(i, j int) bool {
		if cs[i].a.Fame != cs[j].a.Fame {
			return cs[i].a.Fame > cs[j].a.Fame
		}
		return cs[i].a.ID < cs[j].a.ID
	})
	var out []egRelic
	seenB := map[int]bool{}
	for _, c := range cs {
		if len(out) >= 3 {
			break
		}
		if used[c.a.ID] || g.Taken[c.a.ID] || seenB[c.b.ID] {
			continue
		}
		seenB[c.b.ID] = true
		out = append(out, egRelic{c.a.ID, c.b.ID})
	}
	return out
}

func (g *Game) egArt(id int) *ArtifactData {
	for i := range g.W.Artifacts {
		if g.W.Artifacts[i].ID == id {
			return &g.W.Artifacts[i]
		}
	}
	return nil
}

func (g *Game) egHeldCount() int {
	st := g.eg()
	n := 0
	for _, r := range st.Relics {
		if qcInvArt(g, r.Art) != nil {
			st.Held[r.Art] = true
		}
		if st.Held[r.Art] {
			n++
		}
	}
	return n
}

// ---- allies ----

type egAlly struct {
	Key, Name string
	Size      int
	Realm     int
}

func (g *Game) egAllies() []egAlly {
	st := g.eg()
	var out []egAlly
	for _, k := range st.Allies {
		var id int
		if _, err := fmt.Sscanf(k, "state:%d", &id); err == nil && g.W.stateByID[id] != nil {
			n := 0
			for i := range g.W.Burgs {
				if g.W.Burgs[i].Alive && g.W.Burgs[i].State == id {
					n++
				}
			}
			out = append(out, egAlly{k, g.realmName(id), clampi(2+n/4, 2, 5), id})
		}
	}
	for _, o := range g.Orders() {
		if g.OrderMembership(o.ID) != nil {
			out = append(out, egAlly{"order:" + o.ID, o.Name, 3, o.State})
		}
	}
	return out
}

func (g *Game) egAllied(realm int) bool {
	for _, k := range g.eg().Allies {
		if k == stateFaction(realm) {
			return true
		}
	}
	return false
}

// egReadiness scores the finale preparations 0..100.
func (g *Game) egReadiness() (int, []string) {
	st := g.eg()
	held := g.egHeldCount()
	needR := len(st.Relics)
	needK := len(g.W.Megas)
	keys := len(st.Keys)
	rel, key := 35.0, 25.0
	if needR > 0 {
		rel = 35 * float64(mini(held, needR)) / float64(needR)
	}
	if needK > 0 {
		key = 25 * float64(mini(keys, needK)) / float64(needK)
	}
	al := 25 * float64(mini(len(g.egAllies()), 3)) / 3
	fr := 15 * float64(mini(st.Freed, 5)) / 5
	tot := int(rel + key + al + fr)
	return tot, []string{
		fmt.Sprintf("relics %d/%d", held, needR), fmt.Sprintf("keys %d/%d", mini(keys, needK), needK),
		fmt.Sprintf("allies %d/3", mini(len(g.egAllies()), 3)), fmt.Sprintf("land freed %d/5", mini(st.Freed, 5)),
	}
}

// ---- the clock ----

func egTurn(g *Game) {
	if g.Cur == nil || g.Dead || g.W == nil {
		return
	}
	st := g.eg()
	if st.Phase == 0 {
		if g.Turn%60 == 0 && g.Mode == "main" && g.egReady() {
			g.egRise()
		}
		return
	}
	if g.Turn%60 == 0 {
		day := egDay(g)
		if day-st.LastDay > 60 {
			st.LastDay = day - 60
		}
		for st.LastDay < day {
			st.LastDay++
			g.egDayStep(st.LastDay)
		}
		if st.Phase == 1 {
			g.egHeldCount()
		}
	}
	if g.Turn%30 == 0 {
		g.egAmbient()
	}
	g.egMoveCheck()
	g.egFinaleTick()
}

func (g *Game) egDayStep(day int) {
	st := g.eg()
	d := st.def()
	if d == nil {
		return
	}
	switch st.Phase {
	case 1:
		if st.Finale == 0 && (day-st.RiseDay)%d.Grow == 0 {
			g.egSpread(day)
		}
		g.egTowns(day)
		for _, m := range []int{10, 25, 50, 100, 200} {
			if len(st.Occ) >= m && st.Mile < m {
				st.Mile = m
				st.logf(fmt.Sprintf("Day %d: %s now holds %d parasangs.", day, st.Name, m))
				g.Msg(fmt.Sprintf("{{R|News: %s now holds %d parasangs.}}", st.Name, m))
			}
		}
	case 3:
		if st.Ending == "bargain" && day-st.LastTithe >= 28 {
			st.LastTithe = day
			fee := mini(g.P.Gold/10, 500)
			if fee > 0 {
				g.P.Gold -= fee
				g.Msg(fmt.Sprintf("{{R|The pact's tithe is collected: %d %s.}}", fee, g.Sk.Currency))
			}
		}
	}
}

func (g *Game) egSpread(day int) {
	st := g.eg()
	d := st.def()
	w := g.W
	if float64(len(st.Occ)) >= float64(g.egLandCount())*d.Cap {
		return
	}
	type cand struct {
		p int
		h uint64
	}
	for ring := 1; ring <= 2; ring++ {
		seen := map[int]bool{}
		var cs []cand
		for _, p := range st.Occ {
			px, py := p%w.W, p/w.W
			for dy := -ring; dy <= ring; dy++ {
				for dx := -ring; dx <= ring; dx++ {
					if maxi(abs(dx), abs(dy)) != ring || !w.In(px+dx, py+dy) {
						continue
					}
					q := w.Idx(px+dx, py+dy)
					if seen[q] || w.IsWater(q) || st.set()[q] || st.Lib[q] > day {
						continue
					}
					seen[q] = true
					cs = append(cs, cand{q, egH(g, "spread", day, q)})
				}
			}
		}
		if len(cs) == 0 {
			continue
		}
		sort.Slice(cs, func(i, j int) bool {
			if cs[i].h != cs[j].h {
				return cs[i].h < cs[j].h
			}
			return cs[i].p < cs[j].p
		})
		n := d.Per + len(st.Occ)/40
		for i := 0; i < n && i < len(cs); i++ {
			st.add(cs[i].p)
			g.rvCull(cs[i].p, 1, 0.35, "cut down by "+st.Name)
		}
		return
	}
}

func (g *Game) egTowns(day int) {
	st := g.eg()
	for _, b := range g.ws().burgList(g) {
		if st.occupied(b.P) {
			st.Siege[b.ID]++
			if st.Siege[b.ID] == 6 && !st.fallen(b.ID) {
				st.Fallen = append(st.Fallen, b.ID)
				st.logf(fmt.Sprintf("Day %d: %s has fallen.", day, b.Name))
				g.Msg("{{R|News: " + b.Name + " has fallen to " + st.Name + ".}}")
				g.rvCull(b.P, 2, 0.5, "lost when "+b.Name+" fell to "+st.Name)
			}
			continue
		}
		if st.fallen(b.ID) {
			continue
		}
		if dd, _ := g.egNear(b.P); dd <= 5 && day%2 == 0 {
			st.Ref[b.ID] = mini(12, st.Ref[b.ID]+1)
		}
	}
}

// ---- ambient dangers ----

func (g *Game) egAmbient() {
	st := g.eg()
	if st.Phase == 0 || g.Cur == nil || g.Cur.Z != 0 || g.Cur.Key == egArenaKey {
		return
	}
	d := st.def()
	p := g.egHere()
	if !st.occupied(p) || st.Phase == 3 && st.Ending != "abandon" {
		return
	}
	g.egHazard(d, 0.3)
}

// egHazard applies the threat's flavour hardship with the given strength.
func (g *Game) egHazard(d *egDef, f float64) {
	r := NewRng(fmt.Sprintf("%s:eg:hz:%d", g.Seed, g.Turn))
	surv := 1 - 0.15*float64(g.Skills["survival"])
	switch d.ID {
	case "plague", "gene":
		if g.P.St("plague") == 0 && r.Chance(0.1*f*surv) {
			g.P.AddSt("plague", 1500)
			g.Msg("{{R|You feel feverish: the sickness has found you.}}")
		}
	case "winter":
		g.Hunger += int(30 * f)
		if r.Chance(0.1 * f) {
			g.Msg("{{C|The bitter cold gnaws at you.}}")
		}
	case "dust", "bomb":
		g.Thirst += int(40 * f)
		if r.Chance(0.1 * f) {
			g.Msg("{{O|The air is thick and bitter.}}")
		}
	default:
		if r.Chance(0.06 * f * surv) {
			g.hurt(g.P, 1+g.P.Level/6, nil, d.Cause)
		}
	}
}

// egOnMove: one-line hook for WorldMove (after wsOnMove): travel through occupied land is dangerous.
func egOnMove(g *Game) {
	st := g.eg()
	if st.Phase == 0 || !g.W.In(g.WX, g.WY) {
		return
	}
	p := g.W.Idx(g.WX, g.WY)
	if !st.occupied(p) || st.Phase == 3 && st.Ending != "abandon" && st.Ending != "bargain" {
		return
	}
	g.egMoveAt(p)
}

func (g *Game) egMoveAt(p int) {
	st := g.eg()
	key := g.Turn*7 + p
	if st.MoveTurn == key {
		return
	}
	st.MoveTurn = key
	d := st.def()
	r := NewRng(fmt.Sprintf("%s:eg:mv:%d:%d", g.Seed, p, g.Turn))
	g.Msg("{{R|You travel through land held by " + st.Name + ".}}")
	surv := 1 - 0.12*float64(g.Skills["survival"])
	if r.Chance(0.4 * surv) {
		dmg := 2 + g.P.Level/2 + r.Int(0, 3)
		g.Msg("{{R|Ambush: " + d.Cause + " costs you " + fmt.Sprint(dmg) + " health.}}")
		g.hurt(g.P, dmg, nil, d.Cause)
	}
	g.egHazard(d, 2)
}

func (g *Game) egMoveCheck() {
	st := g.eg()
	if g.OnWorld && g.W.In(g.WX, g.WY) {
		if p := g.W.Idx(g.WX, g.WY); st.occupied(p) && (st.Phase == 1 || st.Ending == "abandon") {
			g.egMoveAt(p)
		}
	}
}

// ---- zones: hostile occupiers, refugees ----

func egZone(g *Game, zn *Zone, _ *Rng) { g.egOccupy(zn) }

func egEnter(g *Game, zn *Zone) {
	st := g.eg()
	if st.Phase == 0 {
		return
	}
	g.egOccupy(zn)
	g.egRefugees(zn)
}

func (g *Game) egOccupy(zn *Zone) {
	st := g.eg()
	if st.Phase == 0 || zn.Z != 0 || zn.Key == egArenaKey || !g.W.In(zn.WX, zn.WY) {
		return
	}
	d := st.def()
	p := g.W.Idx(zn.WX, zn.WY)
	if !st.occupied(p) {
		return
	}
	if zn.Kind == "town" && !st.fallen(zn.Burg) {
		return
	}
	if st.Phase == 3 && st.Ending == "ally" {
		return
	}
	if st.Kind == "titan" && st.MonID > 0 { // the sleeper has risen from its lair
		keep := zn.Actors[:0]
		for _, a := range zn.Actors {
			if a.SimMon != st.MonID {
				keep = append(keep, a)
			}
		}
		zn.Actors = keep
	}
	wave := egDay(g)/12 + 1
	if st.Spawned[zn.Key] >= wave {
		return
	}
	st.Spawned[zn.Key] = wave
	live := 0
	for _, a := range zn.Actors {
		if a.HP > 0 && a.Has("egmob") {
			live++
		}
	}
	r := NewRng(fmt.Sprintf("%s:eg:occ:%s:%d", g.Seed, zn.Key, wave))
	dd := wsCheb(g.W, p, st.Origin)
	n := 2 + maxi(0, 4-dd/3) + r.Int(0, 2)
	if st.Phase == 3 {
		n = n / 2
		if st.Ending == "seal" && p != st.Origin {
			return
		}
	}
	lvl := clampi(maxi(zn.Level, g.P.Level-2)+1, 4, 40)
	for i := 0; i < n && live < 9; i++ {
		x, y, ok := zn.FreeSpot(r, func(x, y int) bool { return abs(x-g.P.X) > 8 || abs(y-g.P.Y) > 6 || g.Cur != zn })
		if !ok {
			break
		}
		a := g.Spawn(Pick(r, d.Mobs), lvl, r)
		a.Name, a.Glyph, a.F = Pick(r, d.MobName), d.MobGlyph, d.MobF
		a.Faction, a.AI, a.Angry = d.Faction, "hostile", true
		a.Set("egmob")
		a.X, a.Y, a.HomeX, a.HomeY = x, y, x, y
		zn.Actors = append(zn.Actors, a)
		live++
	}
}

func (g *Game) egRefugees(zn *Zone) {
	st := g.eg()
	if zn.Kind != "town" || zn.Burg <= 0 || st.Phase == 3 && st.Ending == "ally" {
		return
	}
	bd := g.W.burgByID[zn.Burg]
	if bd == nil || st.occupied(bd.P) {
		return
	}
	want := mini(st.Ref[bd.ID], 10)
	if st.Phase == 3 {
		want = mini(want, 4)
	}
	for st.RefDone[bd.ID] < want {
		k := st.RefDone[bd.ID]
		st.RefDone[bd.ID]++
		r := NewRng(fmt.Sprintf("%s:eg:ref:%d:%d", g.Seed, bd.ID, k))
		a := g.makeNPC("villager", bd, g.W.cultByID[bd.Culture], r)
		x, y, ok := zn.FreeSpot(r, nil)
		if !ok {
			return
		}
		a.X, a.Y, a.HomeX, a.HomeY = x, y, x, y
		a.Title = "refugee"
		a.Set("egref")
		a.Desc = "A refugee, thin and watchful, who fled " + st.Name + "."
		zn.Actors = append(zn.Actors, a)
	}
}

// ---- kills ----

func egKill(g *Game, d, src *Actor) {
	st := g.eg()
	if g.Cur == nil {
		return
	}
	if d.Has("egmob") && st.Phase != 0 && g.W.In(g.Cur.WX, g.Cur.WY) && (src == g.P || src != nil && src.AI == "follower") {
		p := g.W.Idx(g.Cur.WX, g.Cur.WY)
		st.Kills[p]++
		if st.Kills[p] >= 8 && p != st.Origin && st.occupied(p) {
			st.remove(p)
			st.Lib[p] = egDay(g) + 20
			st.Kills[p] = 0
			st.Freed++
			g.Msg("{{G|You have broken the hold on this land: it is free of " + st.Name + ".}}")
			g.gainXP(40 + g.P.Level*5)
			if s := g.W.State[p]; s > 0 {
				g.AddRep(stateFaction(s), 25, " for freeing their land")
			}
			st.logf(fmt.Sprintf("Day %d: the wanderer freed land near %s.", egDay(g), g.wsPlace(p)))
		}
	}
	if d.Has("egboss") {
		g.egBossDown(d)
	}
	if d.Boss && g.Cur.Z != 0 || d.Boss && g.Cur.Kind == "tower" {
		g.egMegaKey(g.Cur)
	}
}

// egMegaKey: the master of a megastructure's deepest hall drops its key.
func (g *Game) egMegaKey(zn *Zone) {
	st := g.eg()
	s := g.siteAt(zn.WX, zn.WY, zn.ZX, zn.ZY)
	if s == nil || (s.Theme != "abyss" && s.Theme != "trials") {
		return
	}
	kind := "abyss"
	if s.Theme == "trials" {
		kind = "tower"
	}
	m := g.egMega(kind)
	if m == nil || m.P != g.W.Idx(zn.WX, zn.WY) || st.Keys[kind] {
		return
	}
	st.Keys[kind] = true
	it := &Item{ID: g.newID(), Kind: "key", Name: "Key of " + strings.TrimPrefix(m.Name, "The "), Glyph: '¬', F: CMagenta, Value: 500, Rarity: "unique", Ident: true,
		Extra: map[string]int{"eg": 1}, Lore: "A key to the great structure of " + m.Name + ", the kind that ends things."}
	g.give(it)
	g.Msg("{{M|From the master's hoard you lift the " + it.Name + ".}}")
	g.deed("endgame", fmt.Sprintf("The wanderer %s took the key of %s.", g.P.Name, m.Name), 0, 0)
}

// ---- dialogue ----

func (g *Game) egSay(a *Actor) string {
	st := g.eg()
	d := st.def()
	here := g.egHere()
	var parts []string
	if st.Phase == 3 {
		switch st.Ending {
		case "destroy":
			parts = append(parts, "\"They say you broke "+st.Name+". The fields are being replanted.\"")
		case "seal":
			parts = append(parts, "\"The seal holds. I try not to listen at night for it to stop.\"")
		case "bargain":
			parts = append(parts, "\"The pact keeps the worst away. The tithe keeps us poor.\"")
		case "ally":
			parts = append(parts, "\"It guards the borders now. We left out bread for it. I swear it ate the bread.\"")
		default:
			parts = append(parts, "\"You could have stopped it. Everyone knows it. Everyone remembers.\"")
		}
		return strings.Join(parts, "\n\n")
	}
	if a.Has("egref") {
		parts = append(parts, fmt.Sprintf("\"I fled %s. We walked nine days. Please, do not make me talk about the rest.\"", st.Name))
	} else {
		parts = append(parts, d.Greet[int(HashStr(fmt.Sprint(a.ID, st.Kind))%uint64(len(d.Greet)))])
	}
	parts = append(parts, d.Gossip[int(HashStr(fmt.Sprint(a.ID, st.Kind, "g"))%uint64(len(d.Gossip)))])
	if dd, q := g.egNear(here); q >= 0 {
		if dd == 0 {
			parts = append(parts, "\"It is here. Look around you.\"")
		} else {
			parts = append(parts, fmt.Sprintf("\"The nearest of it is %s.\"", g.wsDirDist(here, q)))
		}
	}
	if bd := g.W.burgByID[g.Cur.Burg]; bd != nil && g.egAllied(bd.State) {
		parts = append(parts, "\"Our levies march with you, wanderer. Whatever you need.\"")
	}
	if n := len(st.Allies); n == 0 {
		parts = append(parts, "{{K|(The realms still squabble. An elder or noble could be asked to call their people to arms.)}}")
	}
	return strings.Join(parts, "\n\n")
}

// egGreeting is a one-line greeting for talk.go's greeting() when the threat is close; "" otherwise.
func (g *Game) egGreeting(a *Actor) string {
	st := g.eg()
	if st.Phase != 1 || g.Cur == nil {
		return ""
	}
	if dd, _ := g.egNear(g.egHere()); dd > 6 && !a.Has("egref") {
		return ""
	}
	d := st.def()
	return d.Greet[int(HashStr(fmt.Sprint(a.ID, st.Kind))%uint64(len(d.Greet)))]
}

// egGossip is the threat's gossip line for talk.go's gossip(); "" when nothing is afoot.
func (g *Game) egGossip(a *Actor) string {
	st := g.eg()
	if st.Phase == 0 || g.Cur == nil || HashStr(fmt.Sprint(a.ID, "egg"))%100 >= 45 {
		return ""
	}
	return g.egSay(a)
}

func egTalk(g *Game, a *Actor) []Option {
	st := g.eg()
	if st.Phase == 0 || a.AI == "follower" || a.HP <= 0 || g.Cur == nil || g.Hostile(a) {
		return nil
	}
	var out []Option
	label := "{{R|Ask about " + shorten(st.Name, 28) + ".}}"
	if st.Phase == 3 {
		label = "{{c|Ask how the land fares since " + shorten(st.Name, 24) + ".}}"
	}
	out = append(out, Option{label, func() { g.talkMenu(g.egSay(a)) }})
	if st.Phase == 1 && st.Finale < 3 && (a.Role == "elder" || a.Role == "notable" || a.Title == "ruler") && g.Cur.Burg > 0 {
		if bd := g.W.burgByID[g.Cur.Burg]; bd != nil && bd.State > 0 && !g.egAllied(bd.State) {
			realm := bd.State
			out = append(out, Option{"{{W|Call " + g.realmName(realm) + " to arms against " + shorten(st.Name, 22) + ".}}", func() { g.egRally(realm) }})
		}
	}
	return out
}

func (g *Game) egRally(realm int) {
	st := g.eg()
	rep := g.Rep[stateFaction(realm)]
	name := g.realmName(realm)
	fee := 150 + g.P.Level*30
	switch {
	case rep < -100:
		g.talkMenu("\"" + name + " does not take orders from someone it has reason to distrust. Earn our good name first.\"")
	case rep >= 150:
		st.Allies = append(st.Allies, stateFaction(realm))
	case !g.cuCanSpend(fee):
		g.talkMenu(fmt.Sprintf("\"Levies cost coin: %s to equip and march them. Come back when you can pay it.\"", g.cuFormat(fee)))
		return
	default:
		g.cuSpend(fee)
		st.Allies = append(st.Allies, stateFaction(realm))
	}
	if !g.egAllied(realm) {
		return
	}
	g.AddRep(stateFaction(realm), 40, " for uniting behind you")
	st.logf(fmt.Sprintf("Day %d: %s pledged its host.", egDay(g), name))
	g.deed("endgame", fmt.Sprintf("%s pledged its host to the wanderer %s against %s.", name, g.P.Name, st.Name), 0, 0)
	g.talkMenu(fmt.Sprintf("\"%s will send its host to the Heart when you march. Our banners are yours.\"", name))
}

// ---- sidebar, overlay, doom, prices, news, spine ----

func egSide(g *Game) []string {
	st := g.eg()
	if st.Phase == 0 || g.Cur == nil {
		return nil
	}
	if st.Phase == 3 {
		if st.Ending == "abandon" && st.occupied(g.egHere()) {
			return []string{"{{r|Occupied land}}"}
		}
		return nil
	}
	out := []string{"{{R|Threat:}} " + shorten(strings.TrimPrefix(st.Name, "The "), 18)}
	here := g.egHere()
	if dd, q := g.egNear(here); dd == 0 {
		out = append(out, "{{r|You stand in occupied land}}")
	} else if q >= 0 {
		out = append(out, "{{y|Front: "+g.wsDirDist(here, q)+"}}")
	}
	if st.Finale == 2 {
		out = append(out, "{{M|The Heart waits: E}}")
	}
	return out
}

// egOverlay colours threatened parasangs on the world map (hook in drawWorld after worldGlyph).
func (g *Game) egOverlay(p int) (rune, byte, byte, bool) {
	st := g.eg()
	if st.Phase == 0 || !st.occupied(p) {
		return 0, 0, 0, false
	}
	d := st.def()
	f := CRed
	if (p/3+g.Turn/240)%2 == 0 {
		f = COrange
	}
	if p == st.Origin {
		f = CMagenta
	}
	if st.Phase == 3 {
		f = CDarkRed
		if st.Ending == "bargain" {
			f = CDarkMagenta
		}
	}
	return d.Glyph, f, CBgRed, true
}

// egDoom is 0..1 for the soundtrack.
func (g *Game) egDoom() float64 {
	st := g.eg()
	if st.Phase == 0 || g.W == nil {
		return 0
	}
	prox := 0.0
	if dd, _ := g.egNear(g.egHere()); dd < 999 {
		prox = egClampF(1-float64(dd)/12, 0, 1)
	}
	if st.Phase == 3 {
		switch st.Ending {
		case "abandon":
			return egClampF(0.25+prox*0.35, 0, 1)
		case "bargain":
			return egClampF(0.15+prox*0.2, 0, 1)
		case "seal":
			return egClampF(prox*0.2, 0, 1)
		}
		return 0
	}
	if g.Cur != nil && g.Cur.Key == egArenaKey {
		if st.Finale == 2 {
			return 0.6
		}
		return 1
	}
	frac := float64(len(st.Occ)) / (float64(g.egLandCount()) * st.def().Cap)
	return egClampF(0.15+0.3*egClampF(frac, 0, 1)+0.55*prox, 0, 1)
}

func egClampF(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// egLinger is how strongly the aftermath still bites (0..1).
func (g *Game) egLinger() float64 {
	st := g.eg()
	if st.Phase != 3 {
		return 0
	}
	switch st.Ending {
	case "destroy":
		return egClampF(1-float64(egDay(g)-st.EndDay)/84, 0, 1)
	case "seal":
		return 0.25
	case "bargain":
		return 0.6
	case "abandon":
		return 1
	}
	return 0
}

// egPriceMod multiplies a market price (hook in price() beside wsItemMul).
func (g *Game) egPriceMod(it *Item) float64 {
	st := g.eg()
	if st.Phase == 0 || it == nil || g.Cur == nil {
		return 1
	}
	k := wsItemKind(it)
	if k == "" {
		return 1
	}
	d := st.def()
	f := d.Price[k]
	if f == 0 {
		f = 1
	}
	m := 1.0
	switch st.Phase {
	case 1:
		dd, _ := g.egNear(g.egHere())
		inten := egClampF(1-float64(dd)/12, 0, 1)
		ramp := egClampF(float64(len(st.Occ))/6, 0, 1)
		m = 1 + (f-1)*(0.35+0.65*inten)*ramp
	case 3:
		m = 1 + (f-1)*0.5*g.egLinger()
		if st.Ending == "ally" {
			m = 0.95
		}
	}
	if k == "food" && g.Cur.Burg > 0 && st.Phase == 1 {
		m *= 1 + 0.015*float64(st.Ref[g.Cur.Burg])
	}
	return m
}

// egNews adds the threat to the world news (hook in wsNewsLines).
func (g *Game) egNews() []wsLine {
	st := g.eg()
	if st.Phase == 0 {
		return nil
	}
	var out []wsLine
	switch st.Phase {
	case 1:
		out = append(out, wsLine{st.Origin, fmt.Sprintf("{{R|%s}} holds %d parasangs and grows. %s", st.Name, len(st.Occ), "Refugees fill the roads.")})
		for _, f := range st.Fallen {
			if b := g.W.burgByID[f]; b != nil && st.occupied(b.P) {
				out = append(out, wsLine{b.P, "{{R|" + b.Name + "}} has fallen."})
			}
		}
		if st.Freed > 0 {
			out = append(out, wsLine{st.Origin, fmt.Sprintf("{{G|%d regions}} have been won back from %s.", st.Freed, st.Name)})
		}
	case 3:
		out = append(out, wsLine{st.Origin, "{{c|" + egEndNames[st.Ending] + ":}} " + st.Name + ". The land is changed."})
	}
	return out
}

// egSpineLines is the endgame chapter for the Q screen (hook in qcDraw after the village tasks).
func (g *Game) egSpineLines() []string {
	st := g.eg()
	if st.Phase == 0 {
		return nil
	}
	mark := func(done bool, cur bool, s string) string {
		switch {
		case done:
			return "{{G|[x] " + s + "}}"
		case cur:
			return "{{Y|[>] " + s + "}}"
		}
		return "{{K|[ ] " + s + "}}"
	}
	held := g.egHeldCount()
	needK := len(g.W.Megas)
	allies := len(g.egAllies())
	fin := st.Phase == 3
	lines := []string{"", "{{R|The last errand: " + st.Name + "}}"}
	lines = append(lines,
		mark(true, false, "The warning: it has risen"),
		mark(allies >= 3 || fin, allies < 3 && !fin, fmt.Sprintf("Unite the realms and orders (%d of 3)", mini(allies, 3))),
		mark(held >= len(st.Relics) || fin, held < len(st.Relics) && !fin, fmt.Sprintf("Gather the relics (%d of %d)", held, len(st.Relics))),
		mark(len(st.Keys) >= needK || fin, len(st.Keys) < needK && !fin, fmt.Sprintf("Take the keys of the great structures (%d of %d)", mini(len(st.Keys), needK), needK)),
		mark(st.Finale >= 2 || fin, st.Finale < 2 && !fin, "March on the Heart and break its master"),
		mark(fin, st.Finale >= 2 && !fin, "The reckoning: destroy, seal, bargain, ally or abandon"))
	if fin {
		lines = append(lines, "{{G|"+egEndNames[st.Ending]+".}}")
	}
	return lines
}
