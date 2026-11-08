package engine

import (
	"fmt"
	"sort"
	"strings"
)

// starts.go (prefix st): starting scenarios chosen at character creation (CharSpec.Scenario, e.g.
// "heir" or "fallen+undead": at most one power start, then at most one twist). "" changes nothing.
// stApply runs once at the end of NewGame. State in g.ext("st") (global: nothing in it is keyed by
// zone, burg, realm or person ids, so it is not planet-local).

type stState struct {
	Scen    string   `json:"sc"`
	Done    bool     `json:"dn,omitempty"`
	Locked  []string `json:"lk,omitempty"` // fallen god: sealed powers, in unlock order
	Unlocks int      `json:"ul,omitempty"`
	LastLvl int      `json:"ll,omitempty"`
	Hunt    string   `json:"hu,omitempty"` // transmigrator: id of the sect that hunts you
	Art     int      `json:"ar,omitempty"` // relic bearer: relic id
	XPSeen  int      `json:"xs,omitempty"` // naked: total XP already doubled
	Day     int      `json:"d,omitempty"`  // last day a hunter party was sent
}

func init() {
	turnHooks = append(turnHooks, stTurn)
	killHooks = append(killHooks, stKill)
	enterHooks = append(enterHooks, stEnter)
	sideHooks = append(sideHooks, stSide)
}

func (g *Game) stSt() *stState { return g.ext("st", func() any { return &stState{} }).(*stState) }

// stGet returns the scenario state, or nil in a standard game (nothing is created or saved then).
func (g *Game) stGet() *stState {
	if g.extLive["st"] == nil {
		if _, ok := g.Store["st"]; !ok {
			return nil
		}
	}
	s := g.stSt()
	if s.Scen == "" || !s.Done {
		return nil
	}
	return s
}

func (s *stState) has(id string) bool {
	for _, p := range strings.Split(s.Scen, "+") {
		if p == id {
			return true
		}
	}
	return false
}

// stParse validates a scenario string: at most one power start and one twist, canonical order.
func stParse(s string) []string {
	var power, twist string
	for _, p := range strings.Split(s, "+") {
		sc := stScenByID[strings.TrimSpace(p)]
		if sc == nil {
			continue
		}
		if sc.Kind == "power" && power == "" {
			power = sc.ID
		} else if sc.Kind == "twist" && twist == "" {
			twist = sc.ID
		}
	}
	var out []string
	if power != "" {
		out = append(out, power)
	}
	if twist != "" {
		out = append(out, twist)
	}
	return out
}

func (g *Game) stRng(key string) *Rng { return NewRng(fmt.Sprintf("%s:st:%s", g.Seed, key)) }

// stApply sets up the chosen scenario. One call at the end of NewGame (after the first zone is entered).
func (g *Game) stApply(spec CharSpec) {
	ids := stParse(spec.Scenario)
	if len(ids) == 0 || g.P == nil || g.Cur == nil {
		return
	}
	st := g.stSt()
	st.Scen = strings.Join(ids, "+")
	if st.has("planet") {
		g.stWrongPlanet()
	}
	if st.has("naked") {
		g.stWarTown()
	}
	switch {
	case st.has("transmig"):
		g.stTransmigrator(st)
	case st.has("fallen"):
		g.stFallen(st)
	case st.has("heir"):
		g.stHeir()
	case st.has("relic"):
		g.stRelic(st)
	case st.has("beast"):
		g.stBeast()
	case st.has("survivor"):
		g.stSurvivor()
	}
	if st.has("undead") {
		g.stUndead()
	}
	if st.has("naked") {
		g.stStrip()
	}
	if st.has("mirror") {
		g.Msg("{{M|Fate rolled your life; it cannot be changed.}}")
		g.deed("event", fmt.Sprintf("%s was dealt a life by fate alone.", g.P.Name), 0, 0)
	}
	if st.has("clock") {
		g.Msg("{{R|The doom clock runs: the great threat rises on day 15.}}")
		g.deed("event", fmt.Sprintf("%s set out knowing the world had fifteen days.", g.P.Name), 0, 0)
	}
	g.cuSync()
	st.XPSeen = g.stXPTotal()
	st.Done = true
	g.recompute()
}

// ---- helpers ----

func (g *Game) stXPTotal() int {
	t := g.XP
	for l := 1; l < g.P.Level; l++ {
		t += XPFor(l)
	}
	return t
}

// stSetLevel raises the player to level n with the usual level-up points, silently.
func (g *Game) stSetLevel(n int) {
	if g.P.Level >= n {
		return
	}
	nm := len(g.Msgs)
	if sl := g.skLevels(); sl.Done == 0 {
		sl.Done = g.P.Level
	}
	for g.P.Level < n {
		g.P.Level++
		hp := maxi(1, 3+g.P.M(TOU)) + g.Skills["endurance"]*3
		g.P.MaxHP += hp
		g.P.HP += hp
	}
	g.skOnLevel()
	g.XP = 0
	if len(g.Msgs) > nm {
		g.Msgs = g.Msgs[:nm]
	}
}

func (g *Game) stCulture() *CultureData {
	if bd := g.W.burgByID[g.StartBurg]; bd != nil {
		return g.W.cultByID[bd.Culture]
	}
	return nil
}

// stRehome makes a realm the home realm (friendly, its enemies not).
func (g *Game) stRehome(state int) {
	g.HomeState = state
	if s := g.W.stateByID[state]; s != nil {
		g.HomeFaith = s.Religion
	}
	if k := stateFaction(state); g.Rep[k] < 350 {
		g.Rep[k] = 350
	}
}

// stMoveTo puts the player in the town of bd (same planet).
func (g *Game) stMoveTo(bd *BurgData, rehome bool) {
	w := g.W
	g.StartBurg = bd.ID
	g.WX, g.WY = bd.P%w.W, bd.P/w.W
	g.StartX, g.StartY = g.WX, g.WY
	if rehome && bd.State != g.HomeState {
		g.stRehome(bd.State)
	}
	g.Known[bd.P] = true
	wx, wy, z := unTownZone(w, bd)
	g.enterZone(g.GetZone(wx, wy, 1, 1, z), 40, 13, true)
	g.Mode = "main"
}

// stHunter spawns one hostile hunter of a faction near the player (far enough to see them coming).
func (g *Game) stHunter(zn *Zone, rng *Rng, title, desc, faction string, lvl int, col byte) *Actor {
	x, y, ok := zn.FreeSpot(rng, func(x, y int) bool { return abs(x-g.P.X) > 9 || abs(y-g.P.Y) > 6 })
	if !ok {
		return nil
	}
	a := g.Spawn("bandit", maxi(1, lvl), rng)
	a.Name, a.Proper = g.cultureName(g.stCulture(), rng), true
	a.Title, a.Desc = title, desc
	a.Faction, a.AI, a.Angry, a.F = faction, "hostile", true, col
	a.Set("sthunter")
	a.X, a.Y, a.HomeX, a.HomeY = x, y, x, y
	zn.Actors = append(zn.Actors, a)
	return a
}

// ---- 1. Transmigrator ----

func (g *Game) stTransmigrator(st *stState) {
	g.stSetLevel(14)
	g.P.Gold = 0
	home := stateFaction(g.HomeState)
	g.Rep[home] = -180
	for k := range g.Rep {
		if strings.HasPrefix(k, "state:") && k != home && g.Rep[k] > -60 {
			g.Rep[k] = -60
		}
	}
	sects := g.Sects()
	if len(sects) > 0 {
		a := sects[HashStr(g.Seed+":st:tm:a")%uint64(len(sects))]
		mr := g.mr()
		mr.Realm, mr.Fav, mr.FavSet = 4, a.Art.Stat, true
		g.mrApply()
		g.mrQi()
		for i := 0; i < 3; i++ {
			g.mrLearn(a.Art.Tech[i].ID)
		}
		var b *MrSect
		for _, id := range a.Rivals {
			if s := g.SectByID(id); s != nil && s != a {
				b = s
				break
			}
		}
		for _, s := range sects {
			if b == nil && s != a && s.Align != a.Align {
				b = s
			}
		}
		for _, s := range sects {
			if b == nil && s != a {
				b = s
			}
		}
		if b != nil {
			st.Hunt = b.ID
			g.Rep[b.RepKey()] = -700
			g.Msg(fmt.Sprintf("{{R|Disciples of %s hunt you.}}", b.Name))
		}
	}
	g.Msg("{{M|You wake in a stranger's body, a cultivator of a high realm, with no friends and no coin.}}")
	g.deed("murim", fmt.Sprintf("%s woke in a foreign body, already a master of the way.", g.P.Name), 0, 0)
}

// ---- 2. Fallen god ----

func (g *Game) stFallen(st *stState) {
	p := g.P
	for i := range p.Attr {
		p.Attr[i] += 7
	}
	p.MaxHP += 40
	p.HP = p.MaxHP
	for _, id := range stFallenOpen {
		p.Powers[id] = 2
	}
	st.Locked = append([]string(nil), stFallenPowers...)
	st.LastLvl = p.Level
	g.MP = 0
	g.Msg(fmt.Sprintf("{{M|A curse seals %d of your gifts. One breaks every 2 levels, or when you fell a mighty foe. Each breaking draws the doom nearer.}}", len(st.Locked)))
	g.deed("event", fmt.Sprintf("%s fell from godhood, gifts sealed by a curse.", g.P.Name), 0, 0)
}

func (g *Game) stUnlock(st *stState, why string) {
	if len(st.Locked) == 0 {
		return
	}
	id := st.Locked[0]
	st.Locked = st.Locked[1:]
	st.Unlocks++
	g.P.Powers[id] = 2
	if g.P.Cool == nil {
		g.P.Cool = map[string]int{}
	}
	g.Msg(fmt.Sprintf("{{W|A seal breaks (%s): you regain %s.}} {{R|The doom stirs.}}", why, g.Sk.PowerName(id)))
	g.deed("event", fmt.Sprintf("%s regained %s, and the world's doom drew nearer.", g.P.Name, g.Sk.PowerName(id)), 0, 0)
}

// ---- 3. Dynasty heir ----

func (g *Game) stHeir() {
	st := g.W.stateByID[g.HomeState]
	if g.W.burgByID[g.StartBurg] != nil {
		st = g.W.stateByID[g.W.burgByID[g.StartBurg].State]
	}
	if st == nil || !st.Alive {
		return
	}
	if cap := g.W.burgByID[st.Capital]; cap != nil && cap.Alive && cap.Layer == "surface" && g.W.Idx(cap.P%g.W.W, cap.P/g.W.W) >= 0 {
		g.stMoveTo(cap, true)
	} else {
		g.stRehome(st.ID)
	}
	realm := st.ID
	po := g.po()
	po.Office[realm] = 0
	po.Paid[realm] = g.poDay()
	po.Aud[realm] = g.poDay() + 1
	po.Stand[realm] = 30
	g.Rep[stateFaction(realm)] = 450
	g.P.Gold = 1500
	for _, f := range g.poFacs(realm) {
		switch f.Kind {
		case "crown":
			po.Fav[f.ID] = 70
		case "sword", "heir":
			if po.Fav[f.ID] > -80 && f.Kind == "sword" {
				po.Fav[f.ID] = -90
			}
		}
	}
	// a rival realm has put a price on you
	best, bestRel := 0, 1<<30
	ids := make([]int, 0, len(st.Relations))
	for id := range st.Relations {
		ids = append(ids, id)
	}
	sort.Ints(ids)
	for _, id := range ids {
		if o := g.W.stateByID[id]; o != nil && o.Alive && st.Relations[id] < bestRel {
			best, bestRel = id, st.Relations[id]
		}
	}
	if best > 0 {
		g.uwSt().Bounty[best] = 250
	}
	g.Msg(fmt.Sprintf("{{M|You are heir to %s: the court receives you, and a faction at court wants you dead.}}", st.FullName))
	g.deed("event", fmt.Sprintf("%s, heir of %s, took a seat at court.", g.P.Name, st.FullName), 0, 0)
}

// ---- 4. Relic bearer ----

func (g *Game) stRelic(st *stState) {
	var c []*ArtifactData
	for i := range g.W.Artifacts {
		a := &g.W.Artifacts[i]
		if !g.Taken[a.ID] && a.Lore != "" {
			c = append(c, a)
		}
	}
	if len(c) == 0 {
		for i := range g.W.Artifacts {
			if !g.Taken[g.W.Artifacts[i].ID] {
				c = append(c, &g.W.Artifacts[i])
			}
		}
	}
	if len(c) > 0 {
		sort.Slice(c, func(i, j int) bool {
			if c[i].Fame != c[j].Fame {
				return c[i].Fame > c[j].Fame
			}
			return c[i].ID < c[j].ID
		})
		a := c[0]
		it := g.RelicItem(a)
		it.Ident = true
		g.Taken[a.ID] = true
		st.Art = a.ID
		g.give(it)
		g.equip(it)
		g.Msg(fmt.Sprintf("{{M|You carry %s. Everyone with a banner wants it.}}", a.Name))
		g.deed("relic", fmt.Sprintf("The wanderer %s set out bearing the relic %s.", g.P.Name, a.Name), 0, a.ID)
	}
	var states []int
	for _, s := range g.W.States {
		if s.Alive && s.ID != g.HomeState {
			states = append(states, s.ID)
		}
	}
	sort.Slice(states, func(i, j int) bool {
		return HashStr(fmt.Sprint(g.Seed, ":st:rb:", states[i])) < HashStr(fmt.Sprint(g.Seed, ":st:rb:", states[j]))
	})
	for i := 0; i < len(states) && i < 3; i++ {
		g.uwSt().Bounty[states[i]] = 350
	}
}

// ---- 5. Beast-bound ----

func (g *Game) stBeast() {
	var c []*bsSp
	for _, pass := range []int{4, 3, 0} {
		for _, s := range bsAll {
			if tmTameable(s) && !tmMountOnly(s) && s.Size >= 5 && s.Rarity >= pass {
				if k := tmRideClass(s); k == "ground" || k == "fly" {
					c = append(c, s)
				}
			}
		}
		if len(c) > 0 {
			break
		}
	}
	p := g.P
	for i := range p.Attr {
		p.Attr[i] = maxi(8, p.Attr[i]-2)
	}
	p.MaxHP = maxi(6, p.MaxHP-4)
	p.HP = p.MaxHP
	if len(c) == 0 {
		return
	}
	s := c[HashStr(g.Seed+":st:beast")%uint64(len(c))]
	rng := g.stRng("beast")
	a := g.bsSpawn(s, 8, rng)
	a.X, a.Y = p.X, p.Y
	g.Cur.Actors = append(g.Cur.Actors, a)
	g.nudge(g.Cur, a)
	g.tmJoin(a, s, 90, g.Cur.Key)
	g.deed("companion", fmt.Sprintf("The wanderer %s was bound to %s %s from the start.", p.Name, bsArticle(s.Display()), s.Display()), 0, 0)
	g.tmSaddle(a, s)
	g.Msg("{{M|Your beast is saddled as a mount. Unsaddle it from the mount menu to let it fight beside you.}}")
}

// ---- 6. Undead ----

func (g *Game) stUndead() {
	for id, lv := range stUndeadPowers {
		if g.P.Powers[id] < lv {
			g.P.Powers[id] = lv
		}
	}
	g.Rep["undead"] = 300
	g.Hunger, g.Thirst = 0, 0
	g.Msg("{{M|You are dead and walking. You need no food or water; the living will not suffer you.}}")
	g.deed("event", fmt.Sprintf("%s rose from the grave.", g.P.Name), 0, 0)
}

// ---- 7. Last survivor ----

func (g *Game) stSurvivor() {
	w := g.W
	here := w.Idx(clampi(g.Cur.WX, 0, w.W-1), clampi(g.Cur.WY, 0, w.H-1))
	st := g.eg()
	if st.Phase == 0 {
		g.egRise()
	}
	st.Occ, st.occSet = nil, nil
	st.Origin = g.egLand(5, 0, "ls")
	st.add(st.Origin)
	for i := 0; i < 60; i++ {
		g.egSpread(1000 + i)
	}
	// the siege ring, but the town's own parasang stays free
	for q := range w.Biome {
		if w.IsWater(q) {
			continue
		}
		d := wsCheb(w, q, here)
		if d == 2 {
			st.add(q)
		} else if d <= 1 {
			st.remove(q)
		}
	}
	for i := range w.Burgs {
		b := &w.Burgs[i]
		if b.Alive && st.occupied(b.P) && b.ID != g.StartBurg {
			st.Siege[b.ID] = 6
			if !st.fallen(b.ID) {
				st.Fallen = append(st.Fallen, b.ID)
			}
		}
	}
	for _, m := range []int{10, 25, 50, 100, 200} {
		if len(st.Occ) >= m {
			st.Mile = m
		}
	}
	st.LastDay = egDay(g)
	st.logf(fmt.Sprintf("The wanderer %s began in a besieged town.", g.P.Name))
	bd := w.burgByID[g.Cur.Burg]
	if bd == nil {
		bd = w.burgByID[g.StartBurg]
	}
	if bd != nil && g.Cur.Kind == "town" {
		rng := g.stRng("allies")
		cu := w.cultByID[bd.Culture]
		for i := 0; i < 3 && g.companionCount() < maxCompanions; i++ {
			a := g.makeNPC(Pick(rng, []string{"guard", "guard", "villager"}), bd, cu, rng)
			x, y, ok := g.Cur.FreeSpot(rng, func(x, y int) bool { return abs(x-g.P.X) <= 6 && abs(y-g.P.Y) <= 4 })
			if !ok {
				continue
			}
			a.X, a.Y = x, y
			g.Cur.Actors = append(g.Cur.Actors, a)
			g.joinCompanion(a)
		}
	}
	g.Msg(fmt.Sprintf("{{R|%s holds %d parasangs and the walls are ringed.}}", st.Name, len(st.Occ)))
	g.deed("endgame", fmt.Sprintf("%s began as the last survivor of a town ringed by %s.", g.P.Name, st.Name), 0, 0)
}

// ---- 8. Wrong planet ----

func (g *Game) stWrongPlanet() {
	u := unUni(g.W)
	cur := g.W.uniIdx
	type opt struct{ pl, burg int }
	var opts []opt
	for pl := range u.Planets {
		if pl == cur {
			continue
		}
		var ids []int
		for id := range u.ends[pl] {
			if b := u.Planets[pl].burgByID[id]; b != nil && b.Alive {
				ids = append(ids, id)
			}
		}
		sort.Ints(ids)
		for _, id := range ids {
			opts = append(opts, opt{pl, id})
		}
	}
	if len(opts) > 0 {
		o := opts[HashStr(g.Seed+":st:planet")%uint64(len(opts))]
		if g.UnGo(o.pl, o.burg) {
			g.P.Gold = mini(g.P.Gold, 30)
			g.Msg("{{M|You step out of a rift onto a world that has never heard of you.}}")
			g.deed("event", fmt.Sprintf("%s fell out of a rift into %s.", g.P.Name, g.unPlanetName()), 0, 0)
			return
		}
	}
	// a one-world file: the farthest town from home
	w := g.W
	var far *BurgData
	fd := -1
	for i := range w.Burgs {
		b := &w.Burgs[i]
		if b.Alive && b.Layer == "surface" && b.State != g.HomeState {
			if d := wsCheb(w, b.P, w.Idx(g.StartX, g.StartY)); d > fd {
				far, fd = b, d
			}
		}
	}
	if far != nil {
		g.stMoveTo(far, false)
		for k := range g.Rep {
			if strings.HasPrefix(k, "state:") {
				g.Rep[k] = 0
			}
		}
		g.HomeState = -1
		g.Msg("{{M|No other world answers; you wake in a far country where nobody knows you.}}")
	}
}

// ---- 9. Naked and nameless ----

func (g *Game) stWarTown() {
	w := g.W
	var warring []int
	for _, s := range w.States {
		if s.Alive && len(g.warsOf(s.ID)) > 0 {
			warring = append(warring, s.ID)
		}
	}
	if len(warring) == 0 {
		return
	}
	sort.Ints(warring)
	realm := warring[HashStr(g.Seed+":st:war")%uint64(len(warring))]
	enemy := map[int]bool{}
	for _, e := range g.warsOf(realm) {
		enemy[e] = true
	}
	var best *BurgData
	bd := 1 << 30
	for i := range w.Burgs {
		b := &w.Burgs[i]
		if !b.Alive || b.Layer != "surface" || b.State != realm || w.IsWater(b.P) {
			continue
		}
		d := 1 << 20
		for j := range w.Burgs {
			o := &w.Burgs[j]
			if o.Alive && enemy[o.State] {
				d = mini(d, wsCheb(w, b.P, o.P))
			}
		}
		if d < bd {
			best, bd = b, d
		}
	}
	if best != nil {
		g.stMoveTo(best, true)
		g.Msg(fmt.Sprintf("{{R|%s is at war, and you are in the thick of it.}}", g.realmName(realm)))
	}
}

func (g *Game) stStrip() {
	p := g.P
	for _, bp := range p.Body {
		if bp.Item != nil && bp.Item.SimArt == 0 {
			bp.Item = nil
		}
	}
	var keep []*Item
	for _, it := range p.Inv {
		if it.SimArt > 0 {
			keep = append(keep, it)
		}
	}
	p.Inv = keep
	p.Powers = map[string]int{}
	p.MaxHP, p.HP = 5, 5
	p.Gold = 0
	g.MP = 0
	g.Msg("{{M|You have nothing: 5 HP, no gear, no gifts. What you learn, you learn twice as fast.}}")
	g.deed("event", fmt.Sprintf("%s began with nothing in a town at war.", p.Name), 0, 0)
}

// ---- 11. endgame clock ----

// stRiseDay is the day the world threat is forced to rise, or 0 for the normal threshold.
func (st *stState) riseDay() int {
	d := 0
	if st.has("clock") {
		d = 15
	}
	if st.has("fallen") {
		f := maxi(12, 54-6*st.Unlocks)
		if d == 0 || f < d {
			d = f
		}
	}
	return d
}

// ---- hooks ----

func stTurn(g *Game) {
	st := g.stGet()
	if st == nil || g.Dead || g.P == nil {
		return
	}
	if st.has("undead") {
		if g.Hunger > 100 {
			g.Hunger = 100
		}
		if g.Thirst > 100 {
			g.Thirst = 100
		}
	}
	if st.has("naked") {
		if tot := g.stXPTotal(); tot > st.XPSeen {
			d := tot - st.XPSeen
			g.gainXP(d)
			st.XPSeen = g.stXPTotal()
		}
	}
	if st.has("fallen") {
		for st.LastLvl < g.P.Level {
			st.LastLvl++
			if st.LastLvl%2 == 0 {
				g.stUnlock(st, fmt.Sprintf("level %d", st.LastLvl))
			}
		}
	}
	if g.Turn%60 == 0 && g.Mode == "main" && g.Cur != nil {
		if d := st.riseDay(); d > 0 && g.Turn/DayLen >= d && g.eg().Phase == 0 {
			g.egRise()
		}
	}
}

func stKill(g *Game, d, src *Actor) {
	st := g.stGet()
	if st == nil || !st.has("fallen") || src != g.P || d == nil {
		return
	}
	if d.Level >= g.P.Level+3 || d.Faction == "legends" {
		g.stUnlock(st, "a mighty foe fell")
	}
}

func stEnter(g *Game, zn *Zone) {
	st := g.stGet()
	if st == nil || g.Dead || zn == nil {
		return
	}
	if st.has("undead") && zn.Kind == "town" {
		n := 0
		for _, a := range zn.Actors {
			if a.HP > 0 && (a.AI == "guard" || a.Role == "guard" || a.Role == "priest") && !a.Angry {
				a.Angry = true
				n++
			}
		}
		if n > 0 {
			g.Msg("{{R|The guards and priests see what you are.}}")
		}
	}
	day := g.Turn / DayLen
	if st.Day == day+1 {
		return
	}
	rng := NewRng(fmt.Sprintf("%s:st:ent:%s:%d", g.Seed, zn.Key, g.Turn))
	surface := zn.Z == 0 && zn.Kind != "town" && g.W.In(zn.WX, zn.WY)
	lvl := g.P.Level
	switch {
	case st.has("transmig") && surface && st.Hunt != "" && rng.Chance(0.3):
		s := g.SectByID(st.Hunt)
		if s == nil {
			return
		}
		made := 0
		for i := 0; i < 1+rng.Int(0, 1); i++ {
			a := g.mrDisciple(s, maxi(1, lvl-2), rng, 1+rng.Int(0, 1))
			x, y, ok := zn.FreeSpot(rng, func(x, y int) bool { return abs(x-g.P.X) > 9 || abs(y-g.P.Y) > 6 })
			if !ok {
				continue
			}
			a.X, a.Y, a.HomeX, a.HomeY = x, y, x, y
			a.AI, a.Angry = "hostile", true
			zn.Actors = append(zn.Actors, a)
			made++
		}
		if made > 0 {
			st.Day = day + 1
			g.Msg(fmt.Sprintf("{{R|Disciples of %s have found you.}}", s.Name))
		}
	case st.has("heir") && g.Turn < 40*DayLen && rng.Chance(0.25):
		realm := g.HomeState
		name := "a court faction"
		for _, f := range g.poFacs(realm) {
			if f.Kind == "sword" {
				name = f.Name
			}
		}
		if g.stHunter(zn, rng, "assassin", "A hired knife sent by "+name+" to end the heir.", "hunters", lvl+1, CRed) != nil {
			st.Day = day + 1
			g.Msg("{{R|Someone is watching you with a knife in hand.}}")
		}
	case st.has("relic") && surface && rng.Chance(0.3):
		title, desc, fac := "relic bandit", "A bandit who has heard what you carry.", "outlaws"
		if s := g.W.stateByID[g.W.State[g.W.Idx(zn.WX, zn.WY)]]; s != nil && rng.Chance(0.6) {
			title, desc, fac = "relic hunter", "A hunter sent by "+s.FullName+" to take the relic.", "hunters"
		}
		n := 1 + rng.Int(0, 1)
		made := 0
		for i := 0; i < n; i++ {
			if g.stHunter(zn, rng, title, desc, fac, lvl, CRed) != nil {
				made++
			}
		}
		if made > 0 {
			st.Day = day + 1
			g.Msg("{{R|Armed strangers have come for the relic.}}")
		}
	}
}

func stSide(g *Game) []string {
	st := g.stGet()
	if st == nil {
		return nil
	}
	var out []string
	if st.has("fallen") && len(st.Locked) > 0 {
		out = append(out, fmt.Sprintf("{{M|Curse: %d seals}}", len(st.Locked)))
	}
	if d := st.riseDay(); d > 0 && g.eg().Phase == 0 {
		if left := d - g.Turn/DayLen; left > 0 {
			out = append(out, fmt.Sprintf("{{R|Doom in %d days}}", left))
		}
	}
	return out
}
