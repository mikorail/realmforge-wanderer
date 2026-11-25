package engine

import (
	"fmt"
	"math"
	"sort"
	"strings"
)

// Contract board (prefix jb). Every town posts 4 to 8 job-board contracts a week, generated from the
// world around it with side rngs only (NewRng(seed:jb:burg:week), never g.R): beasts to hunt, lairs to
// clear, travellers to escort, parcels to deliver, captives to rescue, lost things to recover, roads to
// patrol and night watches to stand. A board is generated once per town and week and kept in the saved
// state, so what you read is what you can take. Up to three contracts may be active. Rewards are paid in
// local money (cuEarn), scaled by level and by standing with the posting realm; finishing earns that
// realm's reputation, failing or abandoning costs some. The board is reached by talking to a guard, elder
// or innkeeper ("Read the contract board") or from the palette; the journal screen (mode "jb", palette
// "Contract journal") shows the active ones. Rumours (rumourmill.go) can seed hunts through
// rmContractSeeds. This file only reads other systems: it does not touch the criminal bounty ledgers of
// hunts.go or the shifts of cityjobs.go. State g.ext("jb"), planet-local ("jb" in unLocalKeys).

func init() {
	unLocalKeys = append(unLocalKeys, "jb")
	turnHooks = append(turnHooks, jbTurn)
	enterHooks = append(enterHooks, jbEnter)
	killHooks = append(killHooks, jbKill)
	talkHooks = append(talkHooks, jbTalk)
	sideHooks = append(sideHooks, jbSide)
	modeKeys["jb"] = jbKeys
	modeDraws["jb"] = func(g *Game) { g.jbDraw() }
	paEntries = append(paEntries,
		paEntry{Name: "Contract board", Keywords: "jobs work hire bounty board contracts hunt escort deliver", Cat: "Screen", Label: "town",
			Cond: func(g *Game) bool { return g.Cur != nil && g.Cur.Kind == "town" && g.Cur.Burg > 0 },
			Func: func(g *Game) { g.jbOpen(0, nil) }},
		paEntry{Name: "Contract journal", Keywords: "jobs active contracts work track abandon", Cat: "Screen", Label: "screen",
			Func: func(g *Game) { g.jbOpen(1, nil) }})
}

const jbMaxActive = 3

type jbC struct {
	Key    string `json:"k"`
	NID    int    `json:"n,omitempty"`
	Kind   string `json:"kd"`
	Burg   int    `json:"b"`
	Title  string `json:"ti"`
	Text   string `json:"tx"`
	Lvl    int    `json:"lv"`
	Reward int    `json:"rw"`
	Rep    int    `json:"rp"`
	Trust  bool   `json:"tr,omitempty"`
	WX     int    `json:"wx"`
	WY     int    `json:"wy"`
	ZX     int    `json:"zx"`
	ZY     int    `json:"zy"`
	TZ     string `json:"tz,omitempty"`
	Tpl    string `json:"tp,omitempty"`
	Foe    string `json:"fo,omitempty"`
	Need   int    `json:"nd,omitempty"`
	To     int    `json:"to,omitempty"`
	Person string `json:"pe,omitempty"`
	Item   string `json:"it,omitempty"`
	Days   int    `json:"dy,omitempty"`
	Src    string `json:"sr,omitempty"`
	State  string `json:"st,omitempty"` // "" posted, active, done, failed, abandoned
	Taken  int    `json:"ta,omitempty"`
	Due    int    `json:"du,omitempty"`
	Prog   int    `json:"pg,omitempty"`
	Placed bool   `json:"pl,omitempty"`
	Done   bool   `json:"dn,omitempty"`
	Actor  int    `json:"ac,omitempty"`
	QID    int    `json:"q,omitempty"`
}

type jbSite struct {
	WX, WY, ZX, ZY int
	S              *Site
}

type jbState struct {
	Next   int                `json:"n"`
	Active []*jbC             `json:"a,omitempty"`
	Boards map[string][]*jbC  `json:"b,omitempty"`
	Done   int                `json:"d,omitempty"`
	Failed int                `json:"f,omitempty"`
	Earned int                `json:"e,omitempty"`
	Hour   int                `json:"h,omitempty"`
	Log    []string           `json:"l,omitempty"`
	Kinds  map[string]int     `json:"k,omitempty"`

	tab, sel, burg int
	conf           string
	talker         *Actor
	sites          map[int][]jbSite
}

func (g *Game) jb() *jbState {
	s := g.ext("jb", func() any { return &jbState{Boards: map[string][]*jbC{}, Kinds: map[string]int{}} }).(*jbState)
	if s.Boards == nil {
		s.Boards = map[string][]*jbC{}
	}
	if s.Kinds == nil {
		s.Kinds = map[string]int{}
	}
	return s
}

// ---------------------------------------------------------------- small helpers

func jbFill(t string, kv ...string) string { return strings.NewReplacer(kv...).Replace(t) }

func jbDirName(dx, dy int) string {
	if abs(dx) <= abs(dy)/2 {
		dx = 0
	}
	if abs(dy) <= abs(dx)/2 {
		dy = 0
	}
	n := map[[2]int]string{{0, -1}: "north", {1, -1}: "north-east", {1, 0}: "east", {1, 1}: "south-east", {0, 1}: "south", {-1, 1}: "south-west", {-1, 0}: "west", {-1, -1}: "north-west", {0, 0}: "right here"}
	return n[[2]int{sign(dx), sign(dy)}]
}

func jbPar(dx, dy int) int { return maxi(1, int(math.Round(math.Hypot(float64(dx), float64(dy))))) }

func (g *Game) jbBurgXY(id int) (int, int, bool) {
	b := g.W.burgByID[id]
	if b == nil || b.P < 0 || b.P >= g.W.W*g.W.H {
		return 0, 0, false
	}
	return b.P % g.W.W, b.P / g.W.W, true
}

func (g *Game) jbBurgName(id int) string {
	if b := g.W.burgByID[id]; b != nil {
		return b.Name
	}
	return "a distant town"
}

func (g *Game) jbFac(burg int) string {
	if b := g.W.burgByID[burg]; b != nil && b.State > 0 {
		return fmt.Sprintf("state:%d", b.State)
	}
	return ""
}

func (g *Game) jbRep(burg int) int {
	if f := g.jbFac(burg); f != "" {
		return g.Rep[f]
	}
	return 0
}

func jbClock(m int) string {
	if m < 0 {
		m = 0
	}
	d, h := m/DayLen, (m%DayLen)/60
	if d > 0 {
		return fmt.Sprintf("%dd %dh", d, h)
	}
	return fmt.Sprintf("%dh %02dm", h, m%60)
}

func (g *Game) jbLog(s string) {
	st := g.jb()
	st.Log = append(st.Log, fmt.Sprintf("Day %d: %s", DayOf(g.Turn)+1, s))
	if len(st.Log) > 12 {
		st.Log = st.Log[len(st.Log)-12:]
	}
}

func (g *Game) jbNews(burg int, format string, a ...any) {
	lvSt(g).addNews(lvDay(g), burg, format, a...)
}

func (g *Game) jbInv(nid int) *Item {
	if nid == 0 {
		return nil
	}
	for _, it := range g.P.Inv {
		if it.Extra != nil && it.Extra["jb"] == nid {
			return it
		}
	}
	return nil
}

func (g *Game) jbDropItems(nid int) {
	for {
		it := g.jbInv(nid)
		if it == nil {
			return
		}
		g.removeInv(it)
	}
}

// jbActorAnywhere finds an actor in any loaded zone.
func (g *Game) jbActorAnywhere(id int) (*Actor, *Zone) {
	if id == 0 {
		return nil, nil
	}
	if g.Cur != nil {
		for _, a := range g.Cur.Actors {
			if a.ID == id {
				return a, g.Cur
			}
		}
	}
	for _, z := range g.Zones {
		if z == g.Cur {
			continue
		}
		for _, a := range z.Actors {
			if a.ID == id {
				return a, z
			}
		}
	}
	return nil, nil
}

func (g *Game) jbLookup(key string) *jbC {
	for _, c := range g.jb().Active {
		if c.Key == key {
			return c
		}
	}
	return nil
}

// ---------------------------------------------------------------- generation

// jbSites lists dungeon sites within four parasangs of a town (cached, not saved).
func (g *Game) jbSites(bd *BurgData) []jbSite {
	st := g.jb()
	if st.sites == nil {
		st.sites = map[int][]jbSite{}
	}
	if v, ok := st.sites[bd.ID]; ok {
		return v
	}
	w := g.W
	bx, by := bd.P%w.W, bd.P/w.W
	out := []jbSite{}
	for dy := -4; dy <= 4; dy++ {
		for dx := -4; dx <= 4; dx++ {
			x, y := bx+dx, by+dy
			if !w.In(x, y) || w.IsWater(w.Idx(x, y)) {
				continue
			}
			for zy := 0; zy < 3; zy++ {
				for zx := 0; zx < 3; zx++ {
					s := g.siteAt(x, y, zx, zy)
					if s == nil || s.Depth < 1 || (s.Kind != "dungeon" && s.Kind != "tower") || s.Theme == "abyss" || s.Theme == "trials" {
						continue
					}
					out = append(out, jbSite{x, y, zx, zy, s})
				}
			}
		}
	}
	st.sites[bd.ID] = out
	return out
}

func (g *Game) jbBoard(burg int) []*jbC {
	st := g.jb()
	week := DayOf(g.Turn) / 7
	key := fmt.Sprintf("%d:%d", burg, week)
	if b, ok := st.Boards[key]; ok {
		return b
	}
	for k := range st.Boards {
		var bb, wk int
		if _, err := fmt.Sscanf(k, "%d:%d", &bb, &wk); err != nil || week-wk > 2 {
			delete(st.Boards, k)
		}
	}
	b := g.jbMake(burg, week)
	if b == nil {
		b = []*jbC{}
	}
	st.Boards[key] = b
	return b
}

func (g *Game) jbMake(burg, week int) []*jbC {
	w := g.W
	bd := w.burgByID[burg]
	if bd == nil || bd.P < 0 || bd.P >= w.W*w.H {
		return nil
	}
	r := NewRng(fmt.Sprint(g.Seed, ":jb:", burg, ":", week))
	n := clampi(4+w.cityTier[burg]+r.Int(0, 2), 4, 8)
	var out []*jbC
	used := map[string]bool{}
	add := func(c *jbC) bool {
		if c == nil {
			return false
		}
		sig := c.Kind + "|" + c.Foe + "|" + c.Person + "|" + fmt.Sprint(c.WX, c.WY, c.ZX, c.ZY, c.To)
		if used[sig] {
			return false
		}
		used[sig] = true
		c.Burg = burg
		c.Key = fmt.Sprintf("%d:%d:%d", burg, week, len(out))
		c.Rep = 2 + c.Lvl/4
		if r.Chance(0.15) {
			c.Trust = true
			c.Reward = c.Reward * 125 / 100
			c.Rep++
			c.Title += " (trusted hands)"
		}
		out = append(out, c)
		return true
	}
	// rumour-fed hunts first (rumourmill.go); nil-safe
	seeds := rmContractSeeds(g, burg)
	for i := range seeds {
		if i >= 2 || len(out) >= n {
			break
		}
		sd := seeds[i]
		add(g.jbHunt(r, bd, &sd))
	}
	tot := 0
	for _, k := range jbKinds {
		tot += k.Weight
	}
	for tries := 0; len(out) < n && tries < n*8; tries++ {
		roll := r.Int(0, tot-1)
		kind := jbKinds[0].ID
		for _, k := range jbKinds {
			if roll < k.Weight {
				kind = k.ID
				break
			}
			roll -= k.Weight
		}
		switch kind {
		case "hunt":
			add(g.jbHunt(r, bd, nil))
		case "clear":
			add(g.jbClear(r, bd))
		case "escort", "deliver":
			add(g.jbTravel(r, bd, kind))
		case "rescue":
			add(g.jbRescue(r, bd))
		case "recover":
			add(g.jbRecover(r, bd))
		case "patrol":
			add(g.jbPatrol(r, bd))
		case "guard":
			add(g.jbGuard(r, bd))
		}
	}
	return out
}

func (g *Game) jbReward(r *Rng, kind string, lvl int, extra int) int {
	kd := jbKindOf(kind)
	base := kd.Base + kd.PerLvl*lvl + extra
	return base * (85 + r.Int(0, 30)) / 100
}

func (g *Game) jbHunt(r *Rng, bd *BurgData, seed *rmSeed) *jbC {
	w := g.W
	bx, by := bd.P%w.W, bd.P/w.W
	px, py := 0, 0
	found := false
	ok := func(x, y int) bool {
		return w.In(x, y) && !w.IsWater(w.Idx(x, y)) && w.Burg[w.Idx(x, y)] == 0 && maxi(abs(x-bx), abs(y-by)) >= 1
	}
	if seed != nil && seed.P >= 0 && seed.P < w.W*w.H {
		sx, sy := seed.P%w.W, seed.P/w.W
		for d := 0; d <= 2 && !found; d++ {
			for dy := -d; dy <= d && !found; dy++ {
				for dx := -d; dx <= d && !found; dx++ {
					if maxi(abs(dx), abs(dy)) == d && ok(sx+dx, sy+dy) {
						px, py, found = sx+dx, sy+dy, true
					}
				}
			}
		}
	}
	for i := 0; i < 60 && !found; i++ {
		x, y := bx+r.Int(-6, 6), by+r.Int(-6, 6)
		if maxi(abs(x-bx), abs(y-by)) >= 2 && ok(x, y) {
			px, py, found = x, y, true
		}
	}
	if !found {
		return nil
	}
	p := w.Idx(px, py)
	lvl := maxi(2, g.zoneLevel(px, py, 0)+g.P.Level/5+r.Int(0, 1))
	id := pickCreature(r, w.Biome[p], 's', lvl)
	t := tplByID[id]
	if t == nil {
		return nil
	}
	kindName := strings.ToLower(t.Name(g.Genre))
	foe := "the " + Pick(r, jbBeastAdj) + " " + t.Name(g.Genre)
	dx, dy := px-bx, py-by
	kv := []string{"{foe}", foe, "{kind}", kindName, "{where}", w.RegionName(p), "{dir}", jbDirName(dx, dy), "{n}", fmt.Sprint(jbPar(dx, dy)), "{town}", bd.Name}
	c := &jbC{Kind: "hunt", Lvl: lvl, WX: px, WY: py, Tpl: id, Foe: foe, Days: 14}
	c.Title = jbFill(Pick(r, jbHuntTitles), kv...)
	if seed != nil {
		c.Text = jbFill(Pick(r, jbRumourHuntText), kv...)
		c.Src = fmt.Sprintf("rm:%d", seed.RID)
	} else {
		c.Text = jbFill(Pick(r, jbHuntText), kv...)
	}
	c.Reward = g.jbReward(r, "hunt", lvl, 0)
	return c
}

func (g *Game) jbClear(r *Rng, bd *BurgData) *jbC {
	sites := g.jbSites(bd)
	if len(sites) == 0 {
		return nil
	}
	s := sites[r.Int(0, len(sites)-1)]
	w := g.W
	lvl := maxi(2, g.zoneLevel(s.WX, s.WY, 1)+s.S.Bonus+g.P.Level/5)
	need := clampi(4+lvl/3, 4, 10)
	dx, dy := s.WX-bd.P%w.W, s.WY-bd.P/w.W
	kv := []string{"{site}", s.S.Name, "{need}", fmt.Sprint(need), "{dir}", jbDirName(dx, dy), "{n}", fmt.Sprint(jbPar(dx, dy)), "{town}", bd.Name}
	c := &jbC{Kind: "clear", Lvl: lvl, WX: s.WX, WY: s.WY, ZX: s.ZX, ZY: s.ZY, Need: need, Days: 14}
	c.Title = jbFill(Pick(r, jbClearTitles), kv...)
	c.Text = jbFill(Pick(r, jbClearText), kv...)
	c.Reward = g.jbReward(r, "clear", lvl, 0)
	return c
}

func (g *Game) jbTravel(r *Rng, bd *BurgData, kind string) *jbC {
	w := g.W
	bx, by := bd.P%w.W, bd.P/w.W
	var cands []*BurgData
	for _, b := range g.ws().burgList(g) {
		if b.ID == bd.ID || b.P < 0 {
			continue
		}
		d := maxi(abs(b.P%w.W-bx), abs(b.P/w.W-by))
		if d >= 2 && d <= 9 && !cvWar(w, bd.State, b.State) {
			cands = append(cands, b)
		}
	}
	if len(cands) == 0 {
		return nil
	}
	dest := cands[r.Int(0, len(cands)-1)]
	dx, dy := dest.P%w.W-bx, dest.P/w.W-by
	km := math.Hypot(float64(dx), float64(dy)) * g.tvKm()
	days := int(km/jbKmDay*1.5) + 2
	lvl := maxi(1, g.zoneLevel(bx, by, 0)+g.P.Level/5)
	cu := w.cultByID[bd.Culture]
	kv := []string{"{dest}", dest.Name, "{dir}", jbDirName(dx, dy), "{n}", fmt.Sprint(jbPar(dx, dy)), "{town}", bd.Name, "{days}", fmt.Sprint(days)}
	c := &jbC{Kind: kind, Lvl: lvl, To: dest.ID, Days: days, WX: dest.P % w.W, WY: dest.P / w.W}
	if kind == "escort" {
		c.Person = g.cultureName(cu, r)
		why := Pick(r, jbEscortWhy)
		c.Item = why
		kv = append(kv, "{who}", c.Person, "{why}", why)
		c.Title = "Escort " + givenOf(c.Person) + " to " + dest.Name
		c.Text = jbFill(Pick(r, jbEscortText), kv...)
		c.Reward = g.jbReward(r, "escort", lvl, int(km/4))
		return c
	}
	c.Item = Pick(r, jbParcels)
	c.Person = "the reeve"
	if ids := w.peopleByBrg[dest.ID]; len(ids) > 0 {
		for i := 0; i < len(ids); i++ {
			p := &w.People[ids[(r.Int(0, len(ids)-1)+i)%len(ids)]]
			if p.Alive && p.Name != "" {
				c.Person = p.Name
				break
			}
		}
	}
	kv = append(kv, "{who}", c.Person, "{item}", c.Item)
	c.Title = "Deliver a " + c.Item + " to " + dest.Name
	c.Text = jbFill(Pick(r, jbDeliverText), kv...)
	c.Days = days + 1
	c.Reward = g.jbReward(r, "deliver", lvl, int(km/5))
	return c
}

func (g *Game) jbRescue(r *Rng, bd *BurgData) *jbC {
	var ds []jbSite
	for _, s := range g.jbSites(bd) {
		if s.S.Kind == "dungeon" {
			ds = append(ds, s)
		}
	}
	if len(ds) == 0 {
		return nil
	}
	s := ds[r.Int(0, len(ds)-1)]
	w := g.W
	lvl := maxi(2, g.zoneLevel(s.WX, s.WY, 1)+s.S.Bonus+g.P.Level/5)
	dx, dy := s.WX-bd.P%w.W, s.WY-bd.P/w.W
	cu := w.cultByID[bd.Culture]
	name := g.cultureName(cu, r)
	owner := Pick(r, jbCaptiveWho) + " of " + bd.Name
	kv := []string{"{who}", name, "{owner}", owner, "{site}", s.S.Name, "{dir}", jbDirName(dx, dy), "{n}", fmt.Sprint(jbPar(dx, dy)), "{town}", bd.Name}
	c := &jbC{Kind: "rescue", Lvl: lvl, WX: s.WX, WY: s.WY, ZX: s.ZX, ZY: s.ZY, TZ: ZoneKey(s.WX, s.WY, s.ZX, s.ZY, 1), Person: name, Days: 14}
	c.Title = "Rescue " + givenOf(name) + " from " + s.S.Name
	c.Text = jbFill(Pick(r, jbRescueText), kv...)
	c.Reward = g.jbReward(r, "rescue", lvl, 0)
	return c
}

func (g *Game) jbRecover(r *Rng, bd *BurgData) *jbC {
	w := g.W
	bx, by := bd.P%w.W, bd.P/w.W
	var ruins []*BurgData
	for i := range w.Burgs {
		b := &w.Burgs[i]
		if b.Alive || b.P < 0 || b.P >= w.W*w.H || (b.Layer != "" && b.Layer != "surface") || w.IsWater(b.P) {
			continue
		}
		if maxi(abs(b.P%w.W-bx), abs(b.P/w.W-by)) <= 8 {
			ruins = append(ruins, b)
		}
	}
	cu := w.cultByID[bd.Culture]
	owner := g.cultureName(cu, r)
	item := Pick(r, jbLost)
	c := &jbC{Kind: "recover", Person: owner, Item: item, Days: 14}
	var site string
	var sx, sy int
	if len(ruins) > 0 && r.Chance(0.6) {
		b := ruins[r.Int(0, len(ruins)-1)]
		sx, sy = b.P%w.W, b.P/w.W
		site = "the ruins of " + b.Name
		c.WX, c.WY, c.ZX, c.ZY = sx, sy, 1, 1
		c.TZ = ZoneKey(sx, sy, 1, 1, 0)
		c.Lvl = maxi(2, g.zoneLevel(sx, sy, 0)+g.P.Level/5)
	} else {
		var ds []jbSite
		for _, s := range g.jbSites(bd) {
			if s.S.Kind == "dungeon" {
				ds = append(ds, s)
			}
		}
		if len(ds) == 0 {
			return nil
		}
		s := ds[r.Int(0, len(ds)-1)]
		sx, sy, site = s.WX, s.WY, s.S.Name
		c.WX, c.WY, c.ZX, c.ZY = s.WX, s.WY, s.ZX, s.ZY
		c.TZ = ZoneKey(s.WX, s.WY, s.ZX, s.ZY, 1)
		c.Lvl = maxi(2, g.zoneLevel(sx, sy, 1)+s.S.Bonus+g.P.Level/5)
	}
	dx, dy := sx-bx, sy-by
	kv := []string{"{owner}", owner, "{item}", item, "{site}", site, "{dir}", jbDirName(dx, dy), "{n}", fmt.Sprint(jbPar(dx, dy)), "{town}", bd.Name}
	c.Title = "Recover " + owner + "'s " + item
	c.Text = jbFill(Pick(r, jbRecoverText), kv...)
	c.Reward = g.jbReward(r, "recover", c.Lvl, 0)
	return c
}

func (g *Game) jbPatrol(r *Rng, bd *BurgData) *jbC {
	w := g.W
	bx, by := bd.P%w.W, bd.P/w.W
	lvl := maxi(1, g.zoneLevel(bx, by, 0)+g.P.Level/5)
	need := clampi(4+lvl/3, 4, 9)
	kv := []string{"{town}", bd.Name, "{need}", fmt.Sprint(need), "{days}", "6"}
	c := &jbC{Kind: "patrol", Lvl: lvl, WX: bx, WY: by, Need: need, Days: 6}
	c.Title = "Patrol the lanes about " + bd.Name
	c.Text = jbFill(Pick(r, jbPatrolText), kv...)
	c.Reward = g.jbReward(r, "patrol", lvl, 0)
	return c
}

func (g *Game) jbGuard(r *Rng, bd *BurgData) *jbC {
	w := g.W
	bx, by := bd.P%w.W, bd.P/w.W
	lvl := maxi(1, g.zoneLevel(bx, by, 0)+g.P.Level/5)
	need := 180 + 60*r.Int(0, 2)
	kv := []string{"{town}", bd.Name, "{need}", fmt.Sprint(need), "{days}", "5"}
	c := &jbC{Kind: "guard", Lvl: lvl, WX: bx, WY: by, Need: need, Days: 5}
	c.Title = "Stand a night watch in " + bd.Name
	c.Text = jbFill(Pick(r, jbGuardText), kv...)
	c.Reward = g.jbReward(r, "guard", lvl, 0)
	return c
}

// ---------------------------------------------------------------- accepting and ending

func (g *Game) jbCanAccept(c *jbC) string {
	st := g.jb()
	if len(st.Active) >= jbMaxActive {
		return fmt.Sprintf("You already carry %d contracts. Finish or abandon one.", jbMaxActive)
	}
	rep := g.jbRep(c.Burg)
	if rep < -50 {
		return "The board of " + g.jbBurgName(c.Burg) + " is closed to you: they do not trust you."
	}
	if c.Trust && rep < 40 {
		return "The clerk will not give this one to a stranger. Earn more standing with " + g.jbBurgName(c.Burg) + "'s realm."
	}
	if g.Cur == nil || g.Cur.Kind != "town" || g.Cur.Burg != c.Burg {
		return "You must be in " + g.jbBurgName(c.Burg) + " to take this."
	}
	return ""
}

func (g *Game) jbTargetP(c *jbC) (int, int) {
	if c.Done || c.Kind == "patrol" || c.Kind == "guard" {
		if x, y, ok := g.jbBurgXY(c.Burg); ok {
			return x, y
		}
	}
	if (c.Kind == "escort" || c.Kind == "deliver") && c.To > 0 {
		if x, y, ok := g.jbBurgXY(c.To); ok {
			return x, y
		}
	}
	return c.WX, c.WY
}

func (g *Game) jbMirror(c *jbC) {
	tx, ty := g.jbTargetP(c)
	tp := 0
	if g.W.In(tx, ty) {
		tp = g.W.Idx(tx, ty)
	}
	mq := &Quest{ID: g.newID(), Kind: "jb", Title: c.Title, Text: c.Text, Giver: "Contract board of " + g.jbBurgName(c.Burg), Faction: "player", Target: ZoneKey(tx, ty, 1, 1, 0), TP: tp}
	c.QID = mq.ID
	g.Quests = append(g.Quests, mq)
	g.Known[tp] = true
}

func (g *Game) jbCloseMirror(c *jbC) {
	for _, q := range g.Quests {
		if q.ID == c.QID && c.QID != 0 {
			q.Done, q.Closed = true, true
		}
	}
}

func (g *Game) jbTrack(c *jbC) {
	tx, ty := g.jbTargetP(c)
	if g.W.In(tx, ty) {
		g.ceSetQuestTarget(tx, ty)
		g.Msg(fmt.Sprintf("{{c|Tracking: %s. The quest arrow points to (%d,%d).}}", c.Title, tx, ty))
	}
}

func (g *Game) jbUntrack(c *jbC) {
	ce := g.ce()
	if !ce.HasWaypoint {
		return
	}
	pts := [][2]int{{c.WX, c.WY}}
	if x, y, ok := g.jbBurgXY(c.Burg); ok {
		pts = append(pts, [2]int{x, y})
	}
	if x, y, ok := g.jbBurgXY(c.To); ok {
		pts = append(pts, [2]int{x, y})
	}
	for _, p := range pts {
		if ce.TargetWX == p[0] && ce.TargetWY == p[1] {
			ce.HasWaypoint = false
			return
		}
	}
}

func (g *Game) jbAccept(c *jbC) {
	if why := g.jbCanAccept(c); why != "" {
		g.Msg("{{y|" + why + "}}")
		return
	}
	st := g.jb()
	st.Next++
	c.NID = st.Next
	c.State, c.Taken = "active", g.Turn
	if c.Days > 0 {
		c.Due = g.Turn + c.Days*DayLen
	}
	bd := g.W.burgByID[c.Burg]
	switch c.Kind {
	case "escort":
		r := NewRng(fmt.Sprint(g.Seed, ":jb:esc:", c.Key))
		a := g.makeNPC("villager", bd, g.W.cultByID[bd.Culture], r)
		a.Name, a.Proper, a.Title = c.Person, true, "traveller"
		a.Desc = fmt.Sprintf("%s, %s, bound for %s.", c.Person, c.Item, g.jbBurgName(c.To))
		a.X, a.Y = g.P.X, g.P.Y
		a.AI, a.Faction, a.Order, a.F = "follower", "player", OrdFollow, CGreen
		g.comp(a)
		a.Set("jbesc")
		g.Cur.Actors = append(g.Cur.Actors, a)
		g.nudge(g.Cur, a)
		c.Actor = a.ID
		g.Msg(fmt.Sprintf("{{G|%s falls in beside you.}} Keep them alive as far as %s.", c.Person, g.jbBurgName(c.To)))
	case "deliver":
		it := &Item{ID: g.newID(), Name: c.Item + " for " + c.Person, Kind: "trophy", Glyph: '=', F: CYellow, Weight: 1, Value: 0, Ident: true,
			Extra: map[string]int{"jb": c.NID}, Lore: "Sealed and addressed to " + c.Person + " in " + g.jbBurgName(c.To) + ". Not yours to open."}
		g.give(it)
		g.Msg("You take the " + it.Name + ".")
	case "rescue":
		c.To = c.Burg
	}
	g.jbMirror(c)
	st.Active = append(st.Active, c)
	if len(st.Active) == 1 {
		g.jbTrack(c)
	}
	st.Kinds[c.Kind]++
	g.jbLog("Took: " + c.Title + ".")
	g.Msg("{{W|Contract taken:}} " + c.Title + ".")
	if c.Due > 0 {
		g.Msg("{{K|Time allowed: " + jbClock(c.Due-g.Turn) + ".}}")
	}
}

func (g *Game) jbRemoveActive(c *jbC) {
	st := g.jb()
	for i, o := range st.Active {
		if o == c {
			st.Active = append(st.Active[:i], st.Active[i+1:]...)
			return
		}
	}
}

// jbCleanup drops what the contract put in your hands or at your heels.
func (g *Game) jbCleanup(c *jbC) {
	g.jbDropItems(c.NID)
	if c.Actor != 0 && (c.Kind == "escort" || c.Kind == "rescue") {
		if a, _ := g.jbActorAnywhere(c.Actor); a != nil && a.HP > 0 {
			if isFollower(a) {
				a.AI, a.Order, a.Comp = "town", "", nil
				if f := g.jbFac(c.Burg); f != "" {
					a.Faction = f
				}
				a.F = CGray
			}
		}
	}
	g.jbUntrack(c)
	g.jbCloseMirror(c)
}

func (g *Game) jbFail(c *jbC, why string) {
	if c.State != "active" {
		return
	}
	c.State = "failed"
	st := g.jb()
	st.Failed++
	g.jbCleanup(c)
	g.jbRemoveActive(c)
	g.Msg("{{R|Contract failed:}} " + c.Title + ". " + why)
	g.jbLog("Failed: " + c.Title + ".")
	if f := g.jbFac(c.Burg); f != "" {
		g.AddRep(f, -(2 + c.Lvl/4), " for a job left undone")
	}
	g.jbNews(c.Burg, "%s took a contract at the board of %s and left it undone.", g.P.Name, g.jbBurgName(c.Burg))
}

func (g *Game) jbAbandon(c *jbC) {
	if c.State != "active" {
		return
	}
	c.State = "abandoned"
	g.jbCleanup(c)
	g.jbRemoveActive(c)
	g.Msg("You hand back: " + c.Title + ".")
	g.jbLog("Abandoned: " + c.Title + ".")
	if f := g.jbFac(c.Burg); f != "" && !c.Done {
		g.AddRep(f, -2, " for dropping a contract")
	}
}

// jbFinish pays a finished contract. fromTalk is the person who handed it in, if any.
func (g *Game) jbFinish(c *jbC, mul int) {
	if c.State != "active" {
		return
	}
	st := g.jb()
	c.State = "done"
	g.jbCleanup(c)
	g.jbRemoveActive(c)
	bd := g.W.burgByID[c.Burg]
	pct := clampi(g.jbRep(c.Burg)/8, -20, 40)
	if c.Due > 0 && c.Days > 0 && c.Due-g.Turn > c.Days*DayLen/2 {
		pct += 10
	}
	amt := c.Reward * (100 + pct) / 100 * mul / 100
	got := g.cuEarn(amt)
	xp := 10 + c.Lvl*6
	g.gainXP(xp)
	st.Done++
	st.Earned += amt
	g.Msg(fmt.Sprintf("{{G|Contract done:}} %s. You are paid {{Y|%s}} (+%d xp).", c.Title, got, xp))
	if f := g.jbFac(c.Burg); f != "" && bd != nil {
		g.AddRep(f, c.Rep, " for work done for "+bd.Name)
	}
	g.jbLog("Done: " + c.Title + ".")
	if f, ok := jbOutcomeNews[c.Kind]; ok {
		g.jbNews(c.Burg, f, g.P.Name, g.jbBurgName(c.Burg))
	}
	if (c.Kind == "rescue" || c.Kind == "clear") && c.Lvl >= 3 {
		g.deed("quest", fmt.Sprintf("The wanderer %s took work from the board of %s: %s.", g.P.Name, g.jbBurgName(c.Burg), strings.ToLower(c.Title[:1])+c.Title[1:]), 0, 0)
	}
	if st.talker != nil {
		if p := lvPersonOf(g, st.talker); p != nil {
			p.remember(lvDay(g), "quest", "did contract work for the town: "+c.Title, 4)
		}
	}
	if strings.HasPrefix(c.Src, "rm:") {
		var rid int
		if _, err := fmt.Sscanf(c.Src, "rm:%d", &rid); err == nil {
			rmConfirm(g, rid)
		}
	}
}

func (g *Game) jbReady(c *jbC) {
	if c.Done {
		return
	}
	c.Done = true
	g.Msg(fmt.Sprintf("{{G|Work finished:}} %s. Report to the board in %s.", c.Title, g.jbBurgName(c.Burg)))
	g.jbUntrack(c)
	if g.jb().Active != nil && len(g.jb().Active) > 0 && g.jb().Active[0] == c {
		g.jbTrack(c)
	}
}

// jbTurnIn is the player's report at the board of the posting town.
func (g *Game) jbTurnIn(c *jbC) {
	if c.State != "active" {
		return
	}
	atBoard := g.Cur != nil && g.Cur.Kind == "town" && g.Cur.Burg == c.Burg
	switch c.Kind {
	case "deliver":
		if g.Cur == nil || g.Cur.Kind != "town" || g.Cur.Burg != c.To {
			g.Msg("The parcel is meant for " + g.jbBurgName(c.To) + ". Find " + c.Person + " there, or leave it with the clerk of the board.")
			return
		}
		if g.jbInv(c.NID) == nil {
			g.Msg("You do not have the parcel any more.")
			return
		}
		g.Msg("The clerk takes the parcel and promises it will reach " + c.Person + ". Less than the full fee, but honest pay.")
		g.jbFinish(c, 70)
		return
	case "escort", "rescue":
		g.Msg("Bring them to " + g.jbBurgName(c.To) + " and they will settle with you themselves.")
		return
	case "recover":
		if g.jbInv(c.NID) == nil {
			g.Msg("You have not found the " + c.Item + " yet.")
			return
		}
		if !atBoard {
			g.Msg("Take it to the board of " + g.jbBurgName(c.Burg) + ".")
			return
		}
		g.jbFinish(c, 100)
		return
	}
	if !c.Done {
		g.Msg("Not done yet: " + g.jbProgress(c))
		return
	}
	if !atBoard {
		g.Msg("Report to the board of " + g.jbBurgName(c.Burg) + ".")
		return
	}
	g.jbFinish(c, 100)
}

func (g *Game) jbProgress(c *jbC) string {
	switch c.Kind {
	case "clear", "patrol":
		return fmt.Sprintf("%d of %d killed.", c.Prog, c.Need)
	case "guard":
		return fmt.Sprintf("%d of %d minutes watched.", c.Prog, c.Need)
	case "hunt":
		if c.Placed {
			return "the beast is out there, near (" + fmt.Sprint(c.WX, ",", c.WY) + ")."
		}
		return "go to (" + fmt.Sprint(c.WX, ",", c.WY) + ") and find it."
	case "escort":
		return "bring " + c.Person + " to " + g.jbBurgName(c.To) + "."
	case "rescue":
		return "free " + c.Person + " and walk them home."
	case "deliver":
		return "carry the " + c.Item + " to " + c.Person + " in " + g.jbBurgName(c.To) + "."
	case "recover":
		if g.jbInv(c.NID) != nil {
			return "you have it. Take it to the board."
		}
		return "find the " + c.Item + " near (" + fmt.Sprint(c.WX, ",", c.WY) + ")."
	}
	return ""
}

// ---------------------------------------------------------------- hooks

func jbFoe(d *Actor) bool {
	return d != nil && d.AI == "hostile" && d.Faction != "player" && !isFollower(d)
}

func jbKill(g *Game, d, src *Actor) {
	if g.W == nil || g.P == nil || g.Cur == nil || d == nil || d == g.P {
		return
	}
	st := g.jb()
	if len(st.Active) == 0 {
		return
	}
	byP := src != nil && (src == g.P || isFollower(src))
	for _, c := range append([]*jbC(nil), st.Active...) {
		if c.State != "active" {
			continue
		}
		if c.Actor != 0 && d.ID == c.Actor {
			switch c.Kind {
			case "hunt":
				g.jbReady(c)
			case "escort", "rescue":
				if !c.Done {
					g.jbFail(c, d.Name+" is dead.")
				}
			}
			continue
		}
		if c.Done || !byP || !jbFoe(d) {
			continue
		}
		switch c.Kind {
		case "clear":
			if g.Cur.Z != 0 && g.Cur.WX == c.WX && g.Cur.WY == c.WY && g.Cur.ZX == c.ZX && g.Cur.ZY == c.ZY {
				c.Prog++
			} else {
				continue
			}
		case "patrol":
			if g.Cur.Z == 0 && g.Cur.Kind != "town" && maxi(abs(g.Cur.WX-c.WX), abs(g.Cur.WY-c.WY)) <= 2 {
				c.Prog++
			} else {
				continue
			}
		default:
			continue
		}
		if c.Prog >= c.Need {
			g.jbReady(c)
		} else {
			g.Msg(fmt.Sprintf("{{c|Contract: %s (%d/%d)}}", c.Title, c.Prog, c.Need))
		}
	}
}

func jbTurn(g *Game) {
	if g.W == nil || g.P == nil || g.Cur == nil {
		return
	}
	st := g.jb()
	if len(st.Active) == 0 {
		return
	}
	hour := ((g.Turn + startMinutes) % DayLen) / 60
	night := hour >= 20 || hour < 5
	for _, c := range append([]*jbC(nil), st.Active...) {
		if c.State != "active" {
			continue
		}
		if c.Kind == "guard" && !c.Done && night && g.Cur.Kind == "town" && g.Cur.Burg == c.Burg {
			c.Prog++
			if c.Prog >= c.Need {
				g.jbReady(c)
			}
		}
		if c.Due > 0 && g.Turn > c.Due && !(c.Done && c.Kind != "recover") {
			g.jbFail(c, "The time ran out.")
		}
	}
	h := g.Turn / 60
	if h == st.Hour {
		return
	}
	st.Hour = h
	for _, c := range append([]*jbC(nil), st.Active...) {
		if c.State != "active" || c.Done {
			continue
		}
		switch c.Kind {
		case "recover":
			if g.jbInv(c.NID) != nil {
				g.jbReady(c)
			}
		case "escort", "rescue":
			if c.Actor != 0 {
				if a, _ := g.jbActorAnywhere(c.Actor); a == nil {
					g.jbFail(c, "You lost "+c.Person+" along the way.")
				}
			}
		}
	}
}

func jbEnter(g *Game, zn *Zone) {
	if g.W == nil || g.P == nil || zn == nil {
		return
	}
	st := g.jb()
	if len(st.Active) == 0 {
		return
	}
	for _, c := range append([]*jbC(nil), st.Active...) {
		if c.State != "active" {
			continue
		}
		switch c.Kind {
		case "hunt":
			if !c.Placed && zn.Z == 0 && zn.Kind != "town" && zn.WX == c.WX && zn.WY == c.WY {
				g.jbPlaceBeast(c, zn)
			}
		case "rescue":
			if !c.Placed && zn.Key == c.TZ {
				g.jbPlaceCaptive(c, zn)
			}
		case "recover":
			if !c.Placed && zn.Key == c.TZ {
				g.jbPlaceItem(c, zn)
			}
		}
		if (c.Kind == "escort" || c.Kind == "rescue") && !c.Done && c.Actor != 0 && zn.Kind == "town" && zn.Burg == c.To {
			for _, a := range zn.Actors {
				if a.ID == c.Actor && isFollower(a) {
					g.Msg(fmt.Sprintf("{{G|%s has reached %s.}} \"Thank you. Take this for your trouble.\"", a.Name, g.jbBurgName(c.To)))
					g.removeActor(zn, a)
					c.Actor = 0
					c.Done = true
					g.jbFinish(c, 100)
					break
				}
			}
		}
	}
}

func (g *Game) jbSpot(zn *Zone, r *Rng, far bool) (int, int, bool) {
	if far {
		if x, y, ok := zn.FreeSpot(r, func(x, y int) bool { return abs(x-g.P.X) > 8 || abs(y-g.P.Y) > 5 }); ok {
			return x, y, true
		}
	}
	return zn.FreeSpot(r, nil)
}

func (g *Game) jbPlaceBeast(c *jbC, zn *Zone) {
	r := NewRng(fmt.Sprint(g.Seed, ":jb:p:", c.Key))
	f := g.Spawn(c.Tpl, c.Lvl, r)
	f.Name, f.Proper = c.Foe, true
	f.MaxHP = f.MaxHP * 3 / 2
	f.HP = f.MaxHP
	f.Set("jb")
	x, y, ok := g.jbSpot(zn, r, true)
	if !ok {
		return
	}
	f.X, f.Y = x, y
	f.HomeX, f.HomeY = x, y
	zn.Actors = append(zn.Actors, f)
	c.Placed, c.Actor = true, f.ID
	g.Msg("{{Y|Fresh tracks and torn brush: " + c.Foe + " is somewhere in this land.}}")
}

func (g *Game) jbPlaceCaptive(c *jbC, zn *Zone) {
	bd := g.W.burgByID[c.Burg]
	if bd == nil {
		return
	}
	r := NewRng(fmt.Sprint(g.Seed, ":jb:p:", c.Key))
	a := g.makeNPC("villager", bd, g.W.cultByID[bd.Culture], r)
	a.Name, a.Proper, a.Title = c.Person, true, "captive"
	a.Desc = c.Person + ", bound hand and foot."
	a.AI = "still"
	a.HP = maxi(1, a.MaxHP*6/10)
	a.Set("jbcap")
	x, y, ok := g.jbSpot(zn, r, true)
	if !ok {
		return
	}
	a.X, a.Y = x, y
	a.HomeX, a.HomeY = x, y
	zn.Actors = append(zn.Actors, a)
	n := 2 + c.Lvl/6
	for i := 0; i < n; i++ {
		m := g.Spawn(pickCreature(r, zn.Biome, 's', c.Lvl), c.Lvl, r)
		mx, my, ok := zn.FreeSpot(r, func(px, py int) bool { return abs(px-x) <= 4 && abs(py-y) <= 3 })
		if !ok {
			mx, my, ok = zn.FreeSpot(r, nil)
		}
		if !ok {
			break
		}
		m.X, m.Y = mx, my
		zn.Actors = append(zn.Actors, m)
	}
	c.Placed, c.Actor = true, a.ID
	g.Msg("{{Y|Someone is bound in the dark of this place. You hear them breathing.}}")
}

func (g *Game) jbPlaceItem(c *jbC, zn *Zone) {
	r := NewRng(fmt.Sprint(g.Seed, ":jb:p:", c.Key))
	x, y, ok := g.jbSpot(zn, r, true)
	if !ok {
		return
	}
	it := &Item{ID: g.newID(), Name: c.Person + "'s " + c.Item, Kind: "trophy", Glyph: '♦', F: CYellow, Weight: 1, Value: 0, Ident: true, X: x, Y: y,
		Extra: map[string]int{"jb": c.NID}, Lore: "Lost by " + c.Person + ". The board of " + g.jbBurgName(c.Burg) + " wants it back."}
	zn.Items = append(zn.Items, it)
	c.Placed = true
	g.Msg("{{Y|Somewhere in here lies " + it.Name + ".}}")
}

func jbBoardRole(a *Actor) bool {
	switch a.Role {
	case "guard", "elder", "innkeeper", "tavern", "lw_innkeeper":
		return true
	}
	return false
}

func jbTalk(g *Game, a *Actor) []Option {
	if g.W == nil || g.Cur == nil || g.P == nil || a == nil || a.HP <= 0 || a.Angry || a.AI == "hostile" {
		return nil
	}
	st := g.jb()
	var out []Option
	if a.Has("jbcap") {
		for _, c := range st.Active {
			c := c
			if c.State == "active" && c.Kind == "rescue" && c.Actor == a.ID {
				out = append(out, Option{"Cut " + a.Name + " loose", func() {
					a.AI, a.Faction, a.Order, a.F = "follower", "player", OrdFollow, CGreen
					g.comp(a)
					a.Flags["jbcap"] = false
					a.Set("jbesc")
					a.Title = "freed captive"
					g.Msg(fmt.Sprintf("{{G|%s is free.}} \"Take me home to %s, please.\"", a.Name, g.jbBurgName(c.Burg)))
					g.Mode = "main"
				}})
			}
		}
	}
	if g.Cur.Kind == "town" && g.Cur.Burg > 0 {
		for _, c := range st.Active {
			c := c
			if c.State == "active" && c.Kind == "deliver" && c.Person == a.Name && g.Cur.Burg == c.To && g.jbInv(c.NID) != nil {
				out = append(out, Option{"Hand over the " + c.Item + " to " + a.Name + " (contract)", func() {
					st.talker = a
					g.Msg("\"Ah. At last.\" " + a.Name + " breaks the seal, reads, and nods.")
					g.jbFinish(c, 100)
					g.Mode = "main"
				}})
			}
		}
		if jbBoardRole(a) {
			out = append(out, Option{"Read the contract board", func() { g.jbOpen(0, a) }})
		}
	}
	return out
}

func jbSide(g *Game) []string {
	if g.W == nil || g.P == nil {
		return nil
	}
	st := g.jb()
	if len(st.Active) == 0 {
		return nil
	}
	soon := -1
	for _, c := range st.Active {
		if c.Due > 0 && !c.Done {
			if left := c.Due - g.Turn; soon < 0 || left < soon {
				soon = left
			}
		}
	}
	line := fmt.Sprintf("Jobs: {{W|%d}}", len(st.Active))
	if soon >= 0 {
		col := "G"
		if soon < DayLen {
			col = "R"
		} else if soon < 3*DayLen {
			col = "Y"
		}
		line += fmt.Sprintf("  next {{%s|%s}}", col, jbClock(soon))
	}
	return []string{line}
}

// ---------------------------------------------------------------- screen

func (g *Game) jbOpen(tab int, talker *Actor) {
	st := g.jb()
	st.talker = talker
	st.tab, st.sel, st.conf = tab, 0, ""
	if g.Cur != nil && g.Cur.Kind == "town" && g.Cur.Burg > 0 {
		st.burg = g.Cur.Burg
	} else if tab == 0 {
		st.tab = 1
	}
	g.Mode = "jb"
}

func (g *Game) jbAtBoard() bool {
	st := g.jb()
	return g.Cur != nil && g.Cur.Kind == "town" && g.Cur.Burg > 0 && g.Cur.Burg == st.burg
}

func (g *Game) jbRows() []*jbC {
	st := g.jb()
	if st.tab == 0 {
		if !g.jbAtBoard() {
			return nil
		}
		var out []*jbC
		for _, c := range g.jbBoard(st.burg) {
			if c.State == "" {
				out = append(out, c)
			}
		}
		sort.SliceStable(out, func(i, j int) bool { return out[i].Lvl < out[j].Lvl })
		return out
	}
	return st.Active
}

func jbKeys(g *Game, k string) {
	st := g.jb()
	rows := g.jbRows()
	switch k {
	case "Escape", "q":
		tk := st.talker
		st.talker = nil
		if tk != nil && g.Talk != nil {
			g.Mode = "talk"
			g.talkMenu("\"Anything else?\"")
		} else {
			g.Mode = "main"
		}
	case "Tab", "ArrowRight", "ArrowLeft":
		if st.tab == 0 {
			st.tab = 1
		} else if g.jbAtBoard() {
			st.tab = 0
		}
		st.sel, st.conf = 0, ""
	case "ArrowUp", "N8", "k":
		st.sel = maxi(0, st.sel-1)
		st.conf = ""
	case "ArrowDown", "N2", "j":
		st.sel = mini(maxi(0, len(rows)-1), st.sel+1)
		st.conf = ""
	case "Enter", "a":
		if st.sel >= len(rows) {
			return
		}
		if st.tab == 0 {
			g.jbAccept(rows[st.sel])
			st.sel = 0
		} else if k == "Enter" {
			g.jbTurnIn(rows[st.sel])
		}
	case "r":
		if st.tab == 1 && st.sel < len(rows) {
			g.jbTurnIn(rows[st.sel])
		}
	case "t":
		if st.tab == 1 && st.sel < len(rows) {
			g.jbTrack(rows[st.sel])
		}
	case "x":
		if st.tab == 1 && st.sel < len(rows) {
			c := rows[st.sel]
			if st.conf != c.Key {
				st.conf = c.Key
				g.Msg("{{y|Press x again to abandon: " + c.Title + ".}}")
				return
			}
			st.conf = ""
			g.jbAbandon(c)
			st.sel = maxi(0, st.sel-1)
		}
	}
	if rows = g.jbRows(); st.sel >= len(rows) {
		st.sel = maxi(0, len(rows)-1)
	}
}

func (g *Game) jbDetail(c *jbC) []string {
	var out []string
	out = append(out, "{{W|"+c.Title+"}}")
	out = append(out, c.Text)
	rewardTxt := g.cuFormat(c.Reward)
	pct := clampi(g.jbRep(c.Burg)/8, -20, 40)
	line := fmt.Sprintf("Level %d. Pay about {{Y|%s}}", c.Lvl, rewardTxt)
	if pct != 0 {
		line += fmt.Sprintf(" ({{%s|%+d%%}} for your standing)", map[bool]string{true: "G", false: "R"}[pct > 0], pct)
	}
	out = append(out, line+fmt.Sprintf(". Standing with the realm: +%d for success, -%d for failure.", c.Rep, 2+c.Lvl/4))
	if c.Days > 0 && c.State == "" {
		out = append(out, fmt.Sprintf("Time allowed once taken: %d days.", c.Days))
	}
	if c.Trust {
		col := "Y"
		if g.jbRep(c.Burg) < 40 {
			col = "R"
		}
		out = append(out, "{{"+col+"|Trusted hands only (standing 40 or more).}}")
	}
	if strings.HasPrefix(c.Src, "rm:") {
		out = append(out, "{{K|Posted on the strength of rumours in the road.}}")
	}
	if c.State == "active" {
		if c.Due > 0 && !(c.Done && c.Kind != "recover") {
			out = append(out, "{{R|Time left: "+jbClock(c.Due-g.Turn)+"}}")
		}
		if c.Done {
			out = append(out, "{{G|Finished. Report to the board of "+g.jbBurgName(c.Burg)+" (r).}}")
		} else {
			out = append(out, "Now: "+g.jbProgress(c))
		}
	}
	return out
}

func (g *Game) jbDraw() {
	sc := g.sc
	st := g.jb()
	title := "Contracts"
	if st.tab == 0 {
		title = "Contract board of " + g.jbBurgName(st.burg)
	}
	x, y, w, h := g.panel(title)
	bottom := 2 + 1 + h - 1
	tabs := fmt.Sprintf("{{%s|[ Board ]}}  {{%s|[ Active %d/%d ]}}", map[bool]string{true: "W", false: "K"}[st.tab == 0], map[bool]string{true: "W", false: "K"}[st.tab == 1], len(st.Active), jbMaxActive)
	sc.TextB(x, y, tabs, CGray, CBgDark, x+w)
	y++
	if st.tab == 0 {
		sc.TextB(x, y, "{{K|Up/Down choose  Enter take  Tab journal  Esc close}}", CGray, CBgDark, x+w)
	} else {
		sc.TextB(x, y, "{{K|Up/Down choose  t track  r turn in  x abandon (twice)  Tab board  Esc close}}", CGray, CBgDark, x+w)
	}
	y += 2
	rows := g.jbRows()
	if len(rows) == 0 {
		msg := "Nothing is posted that you can take. Look again next week."
		if st.tab == 0 && !g.jbAtBoard() {
			msg = "The board can only be read in town."
		}
		if st.tab == 1 {
			msg = "No contracts taken. Read a town's board (guard, elder or innkeeper, or the palette)."
		}
		sc.TextB(x, y, msg, CGray, CBgDark, x+w)
		y += 2
	} else {
		if st.sel >= len(rows) {
			st.sel = len(rows) - 1
		}
		for i, c := range rows {
			if y >= bottom-9 {
				break
			}
			mark := "  "
			if i == st.sel {
				mark = "> "
			}
			tail := g.cuFormat(c.Reward)
			if st.tab == 1 {
				switch {
				case c.Done:
					tail = "{{G|ready}}"
				case c.Due > 0:
					tail = "{{Y|" + jbClock(c.Due-g.Turn) + "}}"
				}
			} else if c.Trust {
				tail = "{{Y|" + tail + "}}"
			}
			line := fmt.Sprintf("%s{{W|%-7s}} %s {{K|L%d}}  %s", mark, "["+jbKindOf(c.Kind).Name+"]", c.Title, c.Lvl, tail)
			sc.TextB(x, y, line, CGray, CBgDark, x+w)
			y++
		}
		y++
		for _, d := range g.jbDetail(rows[st.sel]) {
			for _, ln := range Wrap(d, w) {
				if y < bottom {
					sc.TextB(x, y, ln, CGray, CBgDark, x+w)
					y++
				}
			}
		}
	}
	if st.tab == 1 && len(st.Log) > 0 && y < bottom-2 {
		y++
		sc.TextB(x, y, fmt.Sprintf("{{K|Recent  (done %d, failed %d)}}", st.Done, st.Failed), CGray, CBgDark, x+w)
		y++
		for i := len(st.Log) - 1; i >= 0 && y < bottom; i-- {
			sc.TextB(x, y, "{{K|"+st.Log[i]+"}}", CGray, CBgDark, x+w)
			y++
		}
	}
}
