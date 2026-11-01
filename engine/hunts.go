package engine

import (
	"fmt"
	"sort"
	"strings"
)

// Treasure hunts and bounties (prefix bh). Maps (Kind "map") point at buried caches through clues of
// rising precision; chains of 2-3 maps end at a lost relic from the world's artifacts. Bounty ledgers are
// read from guards and elders; kills need proof, persons can be subdued and walked to a guard.
// State: g.ext("bh"). Keys: D dig, B hunt log.

// bhPlayerBounty is pointed at crime.go's uwBounty by the coordinator.
var bhPlayerBounty = func(g *Game, state int) int { return 0 }

func init() {
	zoneHooks = append(zoneHooks, bhZone)
	killHooks = append(killHooks, bhKill)
	talkHooks = append(talkHooks, bhTalk)
	turnHooks = append(turnHooks, bhTurn)
	enterHooks = append(enterHooks, bhEnter)
	sideHooks = append(sideHooks, bhSide)
	mainKeys["D"] = bhDig
	mainKeys["B"] = bhLog
}

type bhCache struct {
	ID     int    `json:"id"`
	Step   int    `json:"s"`
	Of     int    `json:"of"`
	WX     int    `json:"wx"`
	WY     int    `json:"wy"`
	ZX     int    `json:"zx"`
	ZY     int    `json:"zy"`
	X      int    `json:"x"`
	Y      int    `json:"y"`
	Placed bool   `json:"pl,omitempty"`
	Dug    bool   `json:"dug,omitempty"`
	Gold   int    `json:"g"`
	Guard  string `json:"gd,omitempty"`
	Art    int    `json:"art,omitempty"`
	Region string `json:"rg"`
}

type bhBounty struct {
	ID      int    `json:"id"`
	Kind    string `json:"k"` // monster legend beast outlaw person deserter
	Burg    int    `json:"b"`
	State   int    `json:"st"`
	Name    string `json:"n"`
	Poster  string `json:"tx"`
	Where   string `json:"wh"`
	Reward  int    `json:"rw"`
	WX      int    `json:"wx"`
	WY      int    `json:"wy"`
	ZX      int    `json:"zx"`
	ZY      int    `json:"zy"`
	Z       int    `json:"z"`
	Mon     int    `json:"mon,omitempty"`
	Tpl     string `json:"tpl,omitempty"`
	Lvl     int    `json:"lv,omitempty"`
	Person  int    `json:"pe,omitempty"`
	ActorID int    `json:"aid,omitempty"`
	Placed  bool   `json:"pl,omitempty"`
	Taken   bool   `json:"tk,omitempty"`
	Dead    bool   `json:"dead,omitempty"`
	Cap     bool   `json:"cap,omitempty"`
	Done    bool   `json:"done,omitempty"`
	Live    bool   `json:"live,omitempty"` // may be taken alive
}

type bhState struct {
	N        int               `json:"n"`
	Caches   map[int]*bhCache  `json:"c,omitempty"`
	Clue     map[int]int       `json:"cl,omitempty"`
	Tried    map[string]int    `json:"tr,omitempty"`
	Prog     map[int]int       `json:"pg,omitempty"`
	Reserved map[int]bool      `json:"rs,omitempty"`
	Bounties map[int]*bhBounty `json:"bo,omitempty"`
	Board    map[string][]int  `json:"bd,omitempty"`
	Sold     map[int]int       `json:"sd,omitempty"`
	Hunted   map[string]bool   `json:"hu,omitempty"`
}

func bhSt(g *Game) *bhState {
	s := g.ext("bh", func() any { return &bhState{} }).(*bhState)
	if s.Caches == nil {
		s.Caches = map[int]*bhCache{}
	}
	if s.Clue == nil {
		s.Clue = map[int]int{}
	}
	if s.Tried == nil {
		s.Tried = map[string]int{}
	}
	if s.Prog == nil {
		s.Prog = map[int]int{}
	}
	if s.Reserved == nil {
		s.Reserved = map[int]bool{}
	}
	if s.Bounties == nil {
		s.Bounties = map[int]*bhBounty{}
	}
	if s.Board == nil {
		s.Board = map[string][]int{}
	}
	if s.Sold == nil {
		s.Sold = map[int]int{}
	}
	if s.Hunted == nil {
		s.Hunted = map[string]bool{}
	}
	return s
}

func (s *bhState) next() int { s.N++; return s.N }

// ---- geography helpers ----

func bhPxy(g *Game, p int) (int, int) { return p % g.W.W, p / g.W.W }

func bhDir(dx, dy int) string {
	if abs(dx) <= abs(dy)/2 {
		dx = 0
	}
	if abs(dy) <= abs(dx)/2 {
		dy = 0
	}
	n := map[[2]int]string{{0, -1}: "north", {1, -1}: "north-east", {1, 0}: "east", {1, 1}: "south-east", {0, 1}: "south", {-1, 1}: "south-west", {-1, 0}: "west", {-1, -1}: "north-west", {0, 0}: "right here"}
	return n[[2]int{sign(dx), sign(dy)}]
}

func bhLand(g *Game, x, y int) bool {
	w := g.W
	if !w.In(x, y) {
		return false
	}
	p := w.Idx(x, y)
	return !w.IsWater(p) && w.Burg[p] <= 0
}

func bhNearBurg(g *Game, x, y int) (*BurgData, int) {
	var best *BurgData
	bd := 1 << 30
	for i := range g.W.Burgs {
		b := &g.W.Burgs[i]
		if !b.Alive || b.Layer != "surface" {
			continue
		}
		bx, by := bhPxy(g, b.P)
		if d := maxi(abs(bx-x), abs(by-y)); d < bd {
			best, bd = b, d
		}
	}
	return best, bd
}

func bhThird(zx, zy int) string {
	v := []string{"western", "central", "eastern"}
	h := []string{"northern", "middle", "southern"}
	if zx == 1 && zy == 1 {
		return "the very heart"
	}
	return "the " + h[zy] + "-" + v[zx] + " part"
}

// ---- treasure maps ----

func bhRelic(g *Game, st *bhState, r *Rng) *ArtifactData {
	var c []*ArtifactData
	for i := range g.W.Artifacts {
		a := &g.W.Artifacts[i]
		if g.Taken[a.ID] || st.Reserved[a.ID] || a.Lore == "" {
			continue
		}
		c = append(c, a)
	}
	if len(c) == 0 {
		return nil
	}
	sort.Slice(c, func(i, j int) bool {
		if c[i].Fame != c[j].Fame {
			return c[i].Fame > c[j].Fame
		}
		return c[i].ID < c[j].ID
	})
	if len(c) > 12 {
		c = c[:12]
	}
	return c[r.Int(0, len(c)-1)]
}

// bhNewChain lays out n caches starting near (wx,wy) and returns the first.
func bhNewChain(g *Game, r *Rng, wx, wy, n int) *bhCache {
	st := bhSt(g)
	var first *bhCache
	cx, cy := wx, wy
	for k := 0; k < n; k++ {
		var px, py int
		ok := false
		for i := 0; i < 80 && !ok; i++ {
			d := 3 + i/20
			px, py = cx+r.Int(-d-6, d+6), cy+r.Int(-d-6, d+6)
			ok = maxi(abs(px-cx), abs(py-cy)) >= 3 && bhLand(g, px, py)
		}
		if !ok {
			return first
		}
		c := &bhCache{ID: st.next(), Step: k, Of: n, WX: px, WY: py, ZX: r.Int(0, 2), ZY: r.Int(0, 2), X: -1, Y: -1}
		c.Region = g.W.RegionName(g.W.Idx(px, py))
		c.Gold = r.Int(60, 140) + 40*k + g.P.Level*10
		switch x := r.Int(0, 99); {
		case x < 35:
			c.Guard = "guardian"
		case x < 60:
			c.Guard = "trap"
		}
		if k == n-1 && n >= 2 {
			if a := bhRelic(g, st, r); a != nil {
				c.Art = a.ID
				st.Reserved[a.ID] = true
			}
		}
		st.Caches[c.ID] = c
		if first == nil {
			first = c
		}
		if zn := g.Zones[ZoneKey(c.WX, c.WY, c.ZX, c.ZY, 0)]; zn != nil {
			bhPlace(g, zn)
		}
		cx, cy = px, py
	}
	return first
}

var bhMapNames = []string{"weathered map", "oilcloth chart", "stained map", "map scratched on hide", "torn chart"}

func bhMapItem(g *Game, c *bhCache, r *Rng) *Item {
	name := Pick(r, bhMapNames)
	if c.Of > 1 {
		name += fmt.Sprintf(" (%d of %d)", c.Step+1, c.Of)
	}
	it := &Item{ID: g.newID(), Kind: "map", Name: name, Glyph: '?', F: CYellow, Value: 80 + 40*c.Step, Extra: map[string]int{"bhc": c.ID}, Ident: true,
		Lore: "A map to something buried. Read it to study the clues."}
	it.Text = bhMapText(g, c, 0)
	return it
}

// bhNewMap makes a map (the first of a chain, 30% chance of 2-3) near the player.
func bhNewMap(g *Game, r *Rng, origin string, chain bool) *Item {
	n := 1
	if chain && r.Chance(0.4) {
		n = r.Int(2, 3)
	}
	c := bhNewChain(g, r, g.Cur.WX, g.Cur.WY, n)
	if c == nil {
		return nil
	}
	return bhMapItem(g, c, r)
}

func bhMapText(g *Game, c *bhCache, lvl int) string {
	s := fmt.Sprintf("Inked in a careful hand: a red cross in %s.", c.Region)
	if c.Of > 1 {
		s += fmt.Sprintf(" Margin note: \"%d of %d. The trail ends at a lost relic.\"", c.Step+1, c.Of)
	}
	if lvl >= 1 {
		b, d := bhNearBurg(g, c.WX, c.WY)
		land := BiomeName(g.W.Biome[g.W.Idx(c.WX, c.WY)])
		if b != nil {
			bx, by := bhPxy(g, b.P)
			s += fmt.Sprintf("\n\nLandmark: the %s about %d parasangs %s of %s.", strings.ToLower(land), d, bhDir(c.WX-bx, c.WY-by), b.Name)
		} else {
			s += "\n\nLandmark: the " + strings.ToLower(land) + "."
		}
	}
	if lvl >= 2 {
		s += fmt.Sprintf("\n\nExact: parasang (%d,%d), in %s of it.", c.WX, c.WY, bhThird(c.ZX, c.ZY))
		s += " Stand on the spot and press D to dig."
	}
	return s
}

func bhMapCache(g *Game, it *Item) *bhCache {
	if it == nil || it.Kind != "map" || it.Extra == nil {
		return nil
	}
	return bhSt(g).Caches[it.Extra["bhc"]]
}

func bhHasMap(g *Game, c *bhCache) bool {
	for _, it := range g.P.Inv {
		if bhMapCache(g, it) == c {
			return true
		}
	}
	return false
}

// bhRead studies a map: free upgrades near the place, Lore / Survival checks once a day otherwise.
// Hook it into useItem: case "map": bhRead(g, it).
func bhRead(g *Game, it *Item) {
	c := bhMapCache(g, it)
	if c == nil {
		g.Msg("The map is blank.")
		return
	}
	st := bhSt(g)
	var notes []string
	day := DayOf(g.Turn)
	lvl := st.Clue[c.ID]
	for lvl < 2 && !c.Dug {
		key := fmt.Sprintf("%d:%d", c.ID, lvl)
		if st.Tried[key] == day+1 {
			notes = append(notes, "{{K|You have puzzled over it enough today.}}")
			break
		}
		st.Tried[key] = day + 1
		kind, dc := "lore", dcMedium-1
		if lvl == 1 {
			kind, dc = "survival", dcMedium
		}
		if g.roll2d6Check(g.checkMod(kind), dc) {
			lvl++
			st.Clue[c.ID] = lvl
			notes = append(notes, fmt.Sprintf("{{G|%s: the clues fall into place.}}", checkKinds[kind].label))
			g.gainXP(10)
		} else {
			notes = append(notes, fmt.Sprintf("{{R|%s fails: the marks stay vague.}}", checkKinds[kind].label))
			break
		}
	}
	text := bhMapText(g, c, st.Clue[c.ID])
	if c.Dug {
		text += "\n\nThis cache is already emptied."
	}
	if len(notes) > 0 {
		text += "\n\n" + strings.Join(notes, "\n")
	}
	g.showPopup(it.Name, text)
}

func bhTool(g *Game) bool {
	for _, it := range g.P.Inv {
		if it.Kind == "tool" && it.Effect == "dig" {
			return true
		}
	}
	return false
}

func bhSpade(g *Game) *Item {
	return &Item{ID: g.newID(), Kind: "tool", Effect: "dig", Name: "digging spade", Glyph: '\\', F: CGray, Value: 30, Weight: 4, Ident: true}
}

// bhPlace picks the dig spot when a cache's zone exists.
func bhPlace(g *Game, zn *Zone) {
	st := bhSt(g)
	if zn.Z != 0 || zn.Kind == "town" || zn.Kind == "sea" || zn.Underwater {
		return
	}
	for _, c := range st.Caches {
		if c.Placed || c.WX != zn.WX || c.WY != zn.WY || c.ZX != zn.ZX || c.ZY != zn.ZY {
			continue
		}
		r := NewRng(fmt.Sprintf("%s:bhspot:%d", g.Seed, c.ID))
		if x, y, ok := zn.FreeSpot(r, nil); ok {
			c.X, c.Y, c.Placed = x, y, true
		}
	}
}

func bhZoneCache(g *Game) *bhCache {
	zn := g.Cur
	for _, c := range bhSt(g).Caches {
		if c.Placed && !c.Dug && zn.Z == 0 && c.WX == zn.WX && c.WY == zn.WY && c.ZX == zn.ZX && c.ZY == zn.ZY {
			return c
		}
	}
	return nil
}

func bhDig(g *Game) {
	zn := g.Cur
	if g.Dead || zn == nil {
		return
	}
	if zn.Z != 0 || zn.Kind == "town" || zn.WaterAt(g.P.X, g.P.Y) > 0 || zn.Underwater {
		g.Msg("You cannot dig here.")
		return
	}
	st := bhSt(g)
	c := bhZoneCache(g)
	here := c != nil && c.X == g.P.X && c.Y == g.P.Y
	need := 45
	if bhTool(g) {
		need = 12
	}
	if !here {
		need /= 2
	}
	id := 0
	if here {
		id = c.ID
	}
	for st.Prog[id] < need {
		for _, a := range zn.Actors {
			if a.HP > 0 && g.Hostile(a) && g.visible(a.X, a.Y) {
				g.Msg("{{R|You stop digging: " + g.The(a) + " is near!}}")
				return
			}
		}
		st.Prog[id]++
		g.spend(1000)
		if g.Dead {
			return
		}
	}
	delete(st.Prog, id)
	if !here {
		g.Msg("You dig a hole and find only roots and stones.")
		if c != nil && st.Clue[c.ID] >= 2 && bhHasMap(g, c) {
			dx, dy := c.X-g.P.X, c.Y-g.P.Y
			g.Msg(fmt.Sprintf("{{c|The map's marks seem to point about %d paces %s.}}", maxi(abs(dx), abs(dy)), bhDir(dx, dy)))
		}
		return
	}
	bhOpen(g, c)
}

func bhOpen(g *Game, c *bhCache) {
	st := bhSt(g)
	zn := g.Cur
	c.Dug = true
	r := NewRng(fmt.Sprintf("%s:bhcache:%d", g.Seed, c.ID))
	lvl := zn.Level + 2 + c.Step*2
	g.Msg("{{Y|Your spade strikes something hard: a buried chest!}}")
	g.P.Gold += c.Gold
	g.Msg(fmt.Sprintf("{{Y|%d %s.}}", c.Gold, g.Sk.Currency))
	g.gainXP(30 + 20*c.Step)
	if c.Step < c.Of-1 {
		if nx := st.Caches[c.ID+1]; nx != nil {
			it := bhMapItem(g, nx, r)
			g.give(it)
			g.Msg("{{G|Inside, wrapped in oilcloth: " + it.Name + ".}}")
		}
		it := g.Loot(r, lvl/4+1)
		g.give(it)
	} else if a := bhArt(g, c.Art); a != nil {
		it := g.RelicItem(a)
		it.Ident = true
		g.give(it)
		g.Taken[a.ID] = true
		g.Msg("{{M|You lift out the " + a.Name + "!}}")
		g.showPopup(a.Name, a.Lore+"\n\n"+it.Text)
		g.deed("relic", fmt.Sprintf("%s the %s unearthed the lost %s.", g.P.Name, g.Sk.Calling(g.Calling), a.Name), 0, a.ID)
	} else {
		var it *Item
		if r.Chance(0.4) {
			base := g.MakeWeapon("longblade", clampi(lvl/6+1, 2, 5), r)
			if r.Chance(0.5) {
				base = g.MakeArmor("body", clampi(lvl/6+1, 2, 5), r)
			}
			nm := Pick(r, []string{"the Drowned King's Gift", "Widow's Tithe", "the Last Oath", "Smuggler's Regret", "the Hollow Crown"})
			it = g.MakeUnique(base, nm, "Buried near "+c.Region+" by someone who never came back.")
		} else {
			it = g.bossDrop(r, lvl)
		}
		g.give(it)
		g.Msg("{{M|Treasure: " + it.Display() + ".}}")
	}
	switch c.Guard {
	case "guardian":
		for i := 0; i < 1+r.Int(0, 1); i++ {
			a := g.Spawn(pickCreature(r, zn.Biome, 's', lvl), lvl, r)
			a.X, a.Y = g.P.X, g.P.Y
			g.nudge(zn, a)
			a.Desc = "It has lurked over this hoard for years."
			zn.Actors = append(zn.Actors, a)
		}
		g.Msg("{{R|Something that guarded the hoard rises from cover!}}")
	case "trap":
		if g.roll2d6Check(g.checkMod("perception"), dcMedium) {
			g.Msg("{{G|You spot a tripwire in the lid and cut it.}}")
		} else {
			g.Msg("{{R|A hidden spring snaps: needles fly from the chest!}}")
			g.hurt(g.P, 6+lvl, nil, "a trapped chest")
		}
	}
}

func bhArt(g *Game, id int) *ArtifactData {
	for i := range g.W.Artifacts {
		if g.W.Artifacts[i].ID == id {
			return &g.W.Artifacts[i]
		}
	}
	return nil
}

// ---- bounties ----

func bhBountyFaction(b *bhBounty) string { return stateFaction(b.State) }

func bhLedger(g *Game, bd *BurgData) []*bhBounty {
	st := bhSt(g)
	key := fmt.Sprintf("%d:%d", bd.ID, DayOf(g.Turn)/14)
	if ids, ok := st.Board[key]; ok {
		var out []*bhBounty
		for _, id := range ids {
			if b := st.Bounties[id]; b != nil && !bhStale(g, b) {
				out = append(out, b)
			}
		}
		return out
	}
	r := NewRng(g.Seed + ":bhboard:" + key)
	var ids []int
	add := func(b *bhBounty) {
		if b != nil {
			b.ID, b.Burg, b.State = st.next(), bd.ID, bd.State
			st.Bounties[b.ID] = b
			ids = append(ids, b.ID)
		}
	}
	bx, by := bhPxy(g, bd.P)
	kinds := []string{"monster", "legend", "beast", "beast", "outlaw", "person", "deserter"}
	for i := 0; i < 8 && len(ids) < 4; i++ {
		switch Pick(r, kinds) {
		case "monster":
			add(bhGenMonster(g, r, bd, bx, by))
		case "legend":
			add(bhGenLegend(g, r, bd, bx, by))
		case "beast":
			add(bhGenBeast(g, r, bd, bx, by))
		case "outlaw":
			add(bhGenOutlaw(g, r, bd, bx, by))
		case "person":
			add(bhGenPerson(g, r, bd, bx, by))
		case "deserter":
			add(bhGenDeserter(g, r, bd, bx, by))
		}
	}
	if len(ids) == 0 {
		add(bhGenBeast(g, r, bd, bx, by))
	}
	st.Board[key] = ids
	var out []*bhBounty
	for _, id := range ids {
		out = append(out, st.Bounties[id])
	}
	return out
}

func bhStale(g *Game, b *bhBounty) bool {
	if b.Done {
		return true
	}
	if b.Kind == "monster" && g.Slain[b.Mon] && !b.Taken {
		return true
	}
	return false
}

func bhHeld(st *bhState, f func(b *bhBounty) bool) bool {
	for _, b := range st.Bounties {
		if !b.Done && f(b) {
			return true
		}
	}
	return false
}

func bhGenMonster(g *Game, r *Rng, bd *BurgData, bx, by int) *bhBounty {
	st := bhSt(g)
	var c []*MonsterData
	for i := range g.W.Monsters {
		m := &g.W.Monsters[i]
		mx, my := bhPxy(g, m.P)
		if m.P < 0 || m.P >= len(g.W.State) || (m.Layer != "surface" && m.Layer != "sea") || g.Slain[m.ID] || maxi(abs(mx-bx), abs(my-by)) > 25 {
			continue
		}
		if bhHeld(st, func(b *bhBounty) bool { return b.Kind == "monster" && b.Mon == m.ID }) {
			continue
		}
		c = append(c, m)
	}
	if len(c) == 0 {
		return nil
	}
	m := c[r.Int(0, len(c)-1)]
	mx, my := bhPxy(g, m.P)
	b := &bhBounty{Kind: "monster", Name: m.Name, Mon: m.ID, WX: mx, WY: my, ZX: 1, ZY: 1, Reward: 150 + m.Power*8, Placed: true,
		Where: nonEmpty(m.Lair, g.W.RegionName(m.P))}
	b.Poster = fmt.Sprintf("WANTED, DEAD: %s, a %s of the %s tier. %s Last seen at %s. Proof: %s.", m.Name, strings.ToLower(m.Species), m.Tier, m.Lore, b.Where, nonEmpty(m.Trophy, "its head"))
	return b
}

func bhGenLegend(g *Game, r *Rng, bd *BurgData, bx, by int) *bhBounty {
	st := bhSt(g)
	keys := make([]string, 0, len(g.Zones))
	for k := range g.Zones {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var zs []*Zone
	var as []*Actor
	for _, k := range keys {
		zn := g.Zones[k]
		for _, a := range zn.Actors {
			if a.Has("legend") && a.HP > 0 && maxi(abs(zn.WX-bx), abs(zn.WY-by)) <= 30 && !bhHeld(st, func(b *bhBounty) bool { return b.ActorID == a.ID }) {
				zs, as = append(zs, zn), append(as, a)
			}
		}
	}
	if len(as) == 0 {
		return nil
	}
	i := r.Int(0, len(as)-1)
	zn, a := zs[i], as[i]
	b := &bhBounty{Kind: "legend", Name: a.Name, ActorID: a.ID, WX: zn.WX, WY: zn.WY, ZX: zn.ZX, ZY: zn.ZY, Z: zn.Z, Reward: 120 + a.Level*25, Placed: true, Where: zn.Name}
	b.Poster = fmt.Sprintf("WANTED, DEAD: the legendary %s, %s. Seen near %s. Proof: its pelt or head.", a.Title, a.Name, zn.Name)
	return b
}

var bhBeastAdj = []string{"Red-Maw", "Grey", "Night-Prowler", "Black-Tusk", "Blood-Eyed", "Hollow", "Scarred"}

func bhGenBeast(g *Game, r *Rng, bd *BurgData, bx, by int) *bhBounty {
	for i := 0; i < 60; i++ {
		px, py := bx+r.Int(-6, 6), by+r.Int(-6, 6)
		if maxi(abs(px-bx), abs(py-by)) < 2 || !bhLand(g, px, py) {
			continue
		}
		p := g.W.Idx(px, py)
		lvl := maxi(3, g.zoneLevel(px, py, 0)+3)
		id := pickCreature(r, g.W.Biome[p], 's', lvl)
		t := tplByID[id]
		if t == nil {
			continue
		}
		place := g.W.RegionName(p)
		name := "the " + Pick(r, bhBeastAdj) + " " + t.Name(g.Genre)
		b := &bhBounty{Kind: "beast", Name: name, Tpl: id, Lvl: lvl, WX: px, WY: py, ZX: r.Int(0, 2), ZY: r.Int(0, 2), Reward: 60 + lvl*20, Where: place}
		b.Poster = fmt.Sprintf("WANTED, DEAD: %s, a %s that has been terrorising %s (%d parasangs from %s). Flocks and travellers lost. Proof: its head.", name, strings.ToLower(t.Name(g.Genre)), place, maxi(abs(px-bx), abs(py-by)), bd.Name)
		return b
	}
	return nil
}

func bhGenOutlaw(g *Game, r *Rng, bd *BurgData, bx, by int) *bhBounty {
	st := bhSt(g)
	for i := range g.W.Outlaws {
		ol := &g.W.Outlaws[i]
		ob := g.W.burgByID[ol.Burg]
		if ob == nil {
			continue
		}
		ox, oy := bhPxy(g, ob.P)
		if maxi(abs(ox-bx), abs(oy-by)) > 30 || bhHeld(st, func(b *bhBounty) bool { return b.Kind == "outlaw" && b.Where == ol.Name }) {
			continue
		}
		z := g.outlawZone(ob)
		if z[0] < 0 {
			continue
		}
		s := g.siteAt(ox, oy, z[0], z[1])
		if s == nil || s.Outlaw == nil {
			continue
		}
		name := g.cultureName(g.W.cultByID[ob.Culture], r) + ", chief of " + ol.Name
		b := &bhBounty{Kind: "outlaw", Name: name, WX: ox, WY: oy, ZX: z[0], ZY: z[1], Z: s.Depth, Reward: 200 + ol.Strength*12 + ol.Gold/150, Where: ol.Name, Live: true}
		b.Poster = fmt.Sprintf("WANTED: %s. Hides in the deepest hall of %s near %s. Dead or alive; alive pays half again. Proof: the chief's seal, or the chief in chains.", name, ol.Name, ob.Name)
		return b
	}
	return nil
}

func bhCriminal(p *PersonData) bool {
	if p.Role == "villain" || p.Role == "criminal" || p.Role == "traitor" {
		return true
	}
	cruel, bad := false, false
	for _, t := range p.Traits {
		cruel = cruel || t == "Cruel"
		bad = bad || t == "Greedy" || t == "Ambitious" || t == "Paranoid"
	}
	return p.Alive && cruel && bad && p.Role != "ruler"
}

func bhCrime(p *PersonData) string {
	for _, t := range p.Traits {
		switch t {
		case "Greedy":
			return "extortion and robbery on the roads"
		case "Ambitious":
			return "plotting against the realm"
		case "Paranoid":
			return "murdering those they suspected"
		}
	}
	return "banditry"
}

func bhWild(g *Game, r *Rng, bx, by int) (int, int, bool) {
	for i := 0; i < 80; i++ {
		px, py := bx+r.Int(-8, 8), by+r.Int(-8, 8)
		if maxi(abs(px-bx), abs(py-by)) >= 2 && bhLand(g, px, py) {
			return px, py, true
		}
	}
	return 0, 0, false
}

func bhGenPerson(g *Game, r *Rng, bd *BurgData, bx, by int) *bhBounty {
	st := bhSt(g)
	var c []*PersonData
	for i := range g.W.People {
		p := &g.W.People[i]
		if bhCriminal(p) && !bhHeld(st, func(b *bhBounty) bool { return b.Person == p.ID }) {
			c = append(c, p)
		}
	}
	px, py, ok := bhWild(g, r, bx, by)
	if len(c) == 0 || !ok {
		return nil
	}
	p := c[r.Int(0, len(c)-1)]
	where := g.W.RegionName(g.W.Idx(px, py))
	b := &bhBounty{Kind: "person", Name: p.Name, Person: p.ID, WX: px, WY: py, ZX: r.Int(0, 2), ZY: r.Int(0, 2), Lvl: maxi(4, g.zoneLevel(px, py, 0)+3),
		Reward: 180 + p.Renown*3 + p.Martial*10, Where: where, Live: true}
	b.Poster = fmt.Sprintf("WANTED: %s, once %s, for %s. Believed hiding in %s. Dead or alive; alive pays half again.", p.Name, nonEmpty(p.Title, p.Role), bhCrime(p), where)
	return b
}

func bhGenDeserter(g *Game, r *Rng, bd *BurgData, bx, by int) *bhBounty {
	px, py, ok := bhWild(g, r, bx, by)
	if !ok {
		return nil
	}
	name := g.cultureName(g.W.cultByID[bd.Culture], r)
	where := g.W.RegionName(g.W.Idx(px, py))
	lvl := maxi(3, g.zoneLevel(px, py, 0)+2)
	b := &bhBounty{Kind: "deserter", Name: name, WX: px, WY: py, ZX: r.Int(0, 2), ZY: r.Int(0, 2), Lvl: lvl, Reward: 70 + lvl*12, Where: where, Live: true}
	b.Poster = fmt.Sprintf("DESERTER: %s of the garrison of %s fled the colours. Tracked toward %s. The realm prefers them alive for trial.", name, bd.Name, where)
	return b
}

// bhPlaceTarget puts an accepted bounty's quarry into a freshly generated (or already loaded) zone.
func bhPlaceTarget(g *Game, zn *Zone, b *bhBounty) {
	if b.Placed || b.Done || b.Dead || !b.Taken || zn.WX != b.WX || zn.WY != b.WY || zn.ZX != b.ZX || zn.ZY != b.ZY || zn.Z != b.Z {
		return
	}
	r := NewRng(fmt.Sprintf("%s:bhtarget:%d", g.Seed, b.ID))
	var a *Actor
	switch b.Kind {
	case "outlaw":
		for _, o := range zn.Actors {
			if o.Boss && strings.HasPrefix(o.Name, "the chief of") {
				a = o
				a.Name, a.Proper = b.Name, true
			}
		}
		if a == nil {
			return
		}
	case "beast":
		if zn.Kind == "town" || zn.Kind == "sea" {
			return
		}
		a = g.Spawn(b.Tpl, b.Lvl, r)
		a.Name, a.Proper = b.Name, true
		a.MaxHP *= 2
		a.HP = a.MaxHP
		a.F = CRed
		a.Desc = "A beast the whole district fears."
	case "person", "deserter":
		bd := g.W.burgByID[b.Burg]
		if bd == nil || zn.Kind == "town" || zn.Kind == "sea" {
			return
		}
		a = g.makeNPC("guard", bd, g.W.cultByID[bd.Culture], r)
		a.Name, a.Proper, a.Role, a.AI, a.Faction, a.State = b.Name, true, "", "hostile", "outlaws", 0
		a.Title, a.Level = "wanted", b.Lvl
		a.MaxHP = 20 + b.Lvl*6
		a.HP = a.MaxHP
		a.Desc = "A fugitive with a price on their head."
	default:
		return
	}
	if b.Kind != "outlaw" {
		x, y, ok := zn.FreeSpot(r, nil)
		if !ok {
			return
		}
		a.X, a.Y, a.HomeX, a.HomeY = x, y, x, y
		zn.Actors = append(zn.Actors, a)
	}
	a.Set("bhT")
	b.ActorID, b.Placed = a.ID, true
}

func bhActive(g *Game) []*bhBounty {
	var out []*bhBounty
	for _, b := range bhSt(g).Bounties {
		if b.Taken && !b.Done {
			out = append(out, b)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func bhAccept(g *Game, b *bhBounty) {
	b.Taken = true
	if !b.Placed {
		if zn := g.Zones[ZoneKey(b.WX, b.WY, b.ZX, b.ZY, b.Z)]; zn != nil {
			bhPlaceTarget(g, zn, b)
		}
	}
	g.Msg("{{Y|Bounty accepted: " + b.Name + ".}}")
}

// bhMatch: does this dead actor belong to the bounty?
func bhMatch(b *bhBounty, d *Actor) bool {
	if b.Kind == "monster" {
		return d.SimMon == b.Mon
	}
	return b.ActorID != 0 && d.ID == b.ActorID
}

func bhProof(g *Game, b *bhBounty) *Item {
	m := (*MonsterData)(nil)
	if b.Kind == "monster" {
		for i := range g.W.Monsters {
			if g.W.Monsters[i].ID == b.Mon {
				m = &g.W.Monsters[i]
			}
		}
	}
	for _, it := range g.P.Inv {
		if it.Extra != nil && it.Extra["bhp"] == b.ID {
			return it
		}
		if m != nil && m.Trophy != "" && it.Kind == "trophy" && it.Name == m.Trophy {
			return it
		}
	}
	return nil
}

func bhKill(g *Game, d, src *Actor) {
	if src != g.P && (src == nil || src.AI != "follower") {
		return
	}
	st := bhSt(g)
	// map drops from legendaries and dungeon bosses
	if (d.Has("legend") || d.Has("boss")) && g.Cur.Z >= 0 {
		r := NewRng(fmt.Sprintf("%s:bhdrop:%d", g.Seed, d.ID))
		if r.Chance(map[bool]float64{true: 0.6, false: 0.2}[d.Has("legend")]) {
			if it := bhNewMap(g, r, "drop", true); it != nil {
				it.X, it.Y = d.X, d.Y
				g.Cur.Items = append(g.Cur.Items, it)
				g.Msg("{{Y|A rolled map lies among its things.}}")
			}
		}
	}
	for _, b := range st.Bounties {
		if !b.Taken || b.Done || b.Dead || !bhMatch(b, d) {
			continue
		}
		b.Dead = true
		if b.Kind != "monster" || bhProof(g, b) == nil {
			if d.Has("bhcap") {
				return
			}
			g.Cur.Items = append(g.Cur.Items, &Item{ID: g.newID(), Kind: "trophy", Name: d.Name + "'s token", Glyph: '♦', F: CYellow, Value: 20,
				Extra: map[string]int{"bhp": b.ID}, Lore: "Proof of the bounty on " + d.Name + ".", X: d.X, Y: d.Y})
		}
		g.Msg("{{Y|Bounty target down. Take the proof to a warden or elder of " + g.realmName(b.State) + ".}}")
	}
}

func bhPay(g *Game, b *bhBounty, alive bool) {
	b.Done = true
	pay := b.Reward
	if alive {
		pay = pay * 3 / 2
	}
	g.P.Gold += pay
	g.gainXP(pay / 2)
	g.AddRep(bhBountyFaction(b), 50+pay/20, " for the bounty on "+b.Name)
	how := "slain"
	if alive {
		how = "brought in alive"
	}
	g.deed("bounty", fmt.Sprintf("%s the %s collected the bounty on %s (%s).", g.P.Name, g.Sk.Calling(g.Calling), b.Name, how), 0, 0)
}

// bhTurn: wounded persons yield; one pass per game minute.
func bhTurn(g *Game) {
	if g.Cur == nil {
		return
	}
	st := bhSt(g)
	for _, a := range g.Cur.Actors {
		if a.HP <= 0 || !a.Has("bhT") || a.Has("yield") || a.HP*10 > a.MaxHP*3 {
			continue
		}
		for _, b := range st.Bounties {
			if b.Taken && !b.Done && b.Live && b.ActorID == a.ID {
				a.Set("yield")
				a.AI, a.Angry, a.Faction = "still", false, "captive"
				g.Msg("{{Y|" + a.Name + " throws down their arms: \"Enough! I yield!\"}}")
			}
		}
	}
}

// ---- dialogue: board, hand-over, dealers ----

func bhTalk(g *Game, a *Actor) []Option {
	if a == nil {
		return nil
	}
	st := bhSt(g)
	var out []Option
	if a.Has("yield") && a.Has("bhT") && !a.Has("bhcap") {
		out = append(out, Option{"{{Y|Bind " + a.Name + " and take them in.}}", func() {
			a.Set("bhcap")
			a.AI, a.Faction = "follower", "captive"
			for _, b := range st.Bounties {
				if b.ActorID == a.ID {
					b.Cap = true
				}
			}
			g.Mode, g.Talk = "main", nil
			g.Msg("{{G|" + a.Name + " is bound and follows you. Walk them to a warden or elder of the realm.}}")
		}})
	}
	if bd := g.W.burgByID[g.Cur.Burg]; bd != nil && g.Cur.Kind == "town" && (a.Role == "guard" || a.Role == "elder" || a.Role == "notable") {
		out = append(out, Option{"{{Y|Read the bounty board.}}", func() { bhBoard(g, a, bd) }})
	}
	if bhDealer(g, a) {
		day := DayOf(g.Turn)
		if st.Sold[a.ID] == 0 || day-st.Sold[a.ID] >= 30 {
			price := 120 + g.P.Level*15
			out = append(out, Option{fmt.Sprintf("{{y|Ask quietly about a treasure map. (%d %s)}}", price, g.Sk.Currency), func() {
				if g.P.Gold < price {
					g.talkMenu("\"Come back with coin.\"")
					return
				}
				r := NewRng(fmt.Sprintf("%s:bhdeal:%d:%d", g.Seed, a.ID, day))
				it := bhNewMap(g, r, "dealer", true)
				if it == nil {
					g.talkMenu("\"Nothing today.\"")
					return
				}
				g.P.Gold -= price
				st.Sold[a.ID] = day + 1
				g.give(it)
				g.talkMenu("\"Don't ask where it came from.\" They slide you " + it.Name + ".")
			}})
		}
		if !bhTool(g) {
			out = append(out, Option{fmt.Sprintf("{{y|Buy a digging spade. (30 %s)}}", g.Sk.Currency), func() {
				if g.P.Gold < 30 {
					g.talkMenu("\"Thirty, no less.\"")
					return
				}
				g.P.Gold -= 30
				g.give(bhSpade(g))
				g.talkMenu("\"Dig deep.\"")
			}})
		}
	}
	return out
}

func bhDealer(g *Game, a *Actor) bool {
	return a.Role == "exotic" || a.Role == "bookseller" || a.Role == "merchant" && strings.HasSuffix(g.Cur.Name, districtNames["slums"])
}

func bhBoard(g *Game, a *Actor, bd *BurgData) {
	list := bhLedger(g, bd)
	lines := []string{"{{Y|Bounty board of " + bd.Name + "}}", ""}
	g.Mode, g.Talk = "talk", a
	g.Opts = nil
	for _, b := range list {
		b := b
		tag := ""
		if b.Taken {
			tag = " (accepted)"
		}
		lines = append(lines, fmt.Sprintf("%s: %d %s%s", b.Name, b.Reward, g.Sk.Currency, tag))
		if !b.Taken {
			g.Opts = append(g.Opts, Option{"Read the poster: " + b.Name, func() { bhPoster(g, a, bd, b) }})
		}
	}
	if len(list) == 0 {
		lines = append(lines, "Nothing is posted.")
	}
	for _, b := range bhActive(g) {
		b := b
		if b.State != bd.State {
			continue
		}
		if p := bhProof(g, b); b.Dead && p != nil {
			g.Opts = append(g.Opts, Option{"{{G|Hand in the proof: " + b.Name + ".}}", func() {
				g.removeInv(p)
				bhPay(g, b, false)
				g.talkMenu(fmt.Sprintf("\"Proof enough.\" You are paid %d %s.", b.Reward, g.Sk.Currency))
			}})
		}
		if b.Cap {
			for _, c := range g.Cur.Actors {
				if c.ID == b.ActorID && c.Has("bhcap") && c.HP > 0 && dist(c, g.P) <= 10 {
					c := c
					g.Opts = append(g.Opts, Option{"{{G|Hand over the prisoner: " + b.Name + ".}}", func() {
						c.HP = 0
						g.cleanup()
						bhPay(g, b, true)
						g.talkMenu(fmt.Sprintf("\"Alive, and in chains. Well done.\" You are paid %d %s.", b.Reward*3/2, g.Sk.Currency))
					}})
				}
			}
		}
	}
	g.Popup = Wrap(strings.Join(lines, "\n"), 100)
	g.Opts = append(g.Opts, Option{"Back.", func() { g.talkMenu(g.greeting(a)) }})
	g.Sel = 0
}

func bhPoster(g *Game, a *Actor, bd *BurgData, b *bhBounty) {
	g.Popup = Wrap(fmt.Sprintf("%s\n\nReward: %d %s (reputation with %s)", b.Poster, b.Reward, g.Sk.Currency, g.realmName(b.State)), 100)
	g.Opts = []Option{
		{"{{G|Accept this bounty.}}", func() { bhAccept(g, b); bhBoard(g, a, bd) }},
		{"Back.", func() { bhBoard(g, a, bd) }},
	}
	g.Sel = 0
}

// ---- zone / enter / side / log ----

func bhZone(g *Game, zn *Zone, r *Rng) {
	st := bhSt(g)
	bhPlace(g, zn)
	for _, b := range st.Bounties {
		bhPlaceTarget(g, zn, b)
	}
	side := NewRng(g.Seed + ":bhmap:" + zn.Key)
	if zn.Kind == "town" {
		if bd := g.W.burgByID[zn.Burg]; bd != nil && strings.HasSuffix(zn.Name, districtNames["plaza"]) {
			if x, y, ok := zn.FreeSpot(side, func(x, y int) bool { return abs(x-40) < 12 && abs(y-12) < 6 }); ok {
				zn.Set(x, y, TInscription)
				zn.Notes[y*MW+x] = "BOUNTY BOARD of " + bd.Name + ". Notices flutter under rusty nails. The wardens and elders keep the full ledger: ask them."
			}
		}
		return
	}
	if zn.Kind == "sea" || zn.Underwater || zn.Sky {
		return
	}
	if (zn.Kind == "ruin" || zn.Kind == "dungeon" || zn.Kind == "crypt") && side.Chance(0.15) {
		if x, y, ok := zn.FreeSpot(side, nil); ok {
			c := bhNewChain(g, side, zn.WX, zn.WY, map[bool]int{true: 2, false: 1}[side.Chance(0.3)])
			if c != nil {
				it := bhMapItem(g, c, side)
				it.X, it.Y = x, y
				zn.Items = append(zn.Items, it)
			}
		}
	}
}

func bhEnter(g *Game, zn *Zone) {
	st := bhSt(g)
	// clues sharpen near the place
	for _, it := range g.P.Inv {
		c := bhMapCache(g, it)
		if c == nil || c.Dug {
			continue
		}
		d := maxi(abs(c.WX-zn.WX), abs(c.WY-zn.WY))
		want := 0
		if d <= 3 {
			want = 1
		}
		if d == 0 {
			want = 2
		}
		if want > st.Clue[c.ID] {
			st.Clue[c.ID] = want
			g.Msg("{{c|The land matches your map. The clues sharpen.}}")
		}
	}
	if zn.Kind == "town" || zn.Kind == "sea" || zn.Z != 0 || !g.W.In(zn.WX, zn.WY) {
		return
	}
	bhMaybeHunters(g, zn, false)
}

// bhMaybeHunters: when the realm here has a bounty on the player, hunters follow their trail.
func bhMaybeHunters(g *Game, zn *Zone, force bool) int {
	p := g.W.Idx(zn.WX, zn.WY)
	state := g.W.State[p]
	if state <= 0 {
		return 0
	}
	amt := bhPlayerBounty(g, state)
	if amt <= 0 {
		return 0
	}
	st := bhSt(g)
	key := fmt.Sprintf("%d:%d", state, DayOf(g.Turn))
	if st.Hunted[key] {
		return 0
	}
	if !force && !g.R.Chance(minf(0.15+float64(amt)/2000, 0.6)) {
		return 0
	}
	bd := g.warBurg(state)
	if bd == nil {
		return 0
	}
	st.Hunted[key] = true
	n := 1 + amt/300
	if n > 3 {
		n = 3
	}
	r := NewRng(fmt.Sprintf("%s:bhhunt:%s:%d", g.Seed, key, g.Turn))
	made := 0
	for i := 0; i < n; i++ {
		x, y, ok := zn.FreeSpot(r, func(x, y int) bool { return abs(x-g.P.X) > 9 || abs(y-g.P.Y) > 6 })
		if !ok {
			continue
		}
		a := g.makeNPC("guard", bd, g.W.cultByID[bd.Culture], r)
		a.Name, a.Proper, a.Role, a.AI, a.Faction, a.State = g.cultureName(g.W.cultByID[bd.Culture], r), true, "", "hostile", "hunters", 0
		a.Title, a.Angry = "bounty hunter", true
		a.Level += amt / 400
		a.MaxHP += amt / 40
		a.HP = a.MaxHP
		a.Desc = fmt.Sprintf("A bounty hunter carrying a warrant from %s.", g.realmName(state))
		a.Set("bhhunter")
		a.X, a.Y, a.HomeX, a.HomeY = x, y, x, y
		zn.Actors = append(zn.Actors, a)
		made++
	}
	if made > 0 {
		g.Msg("{{R|Hard-eyed riders have tracked you into these lands.}}")
	}
	return made
}

func minf(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func bhSide(g *Game) []string {
	n, m := len(bhActive(g)), 0
	for _, it := range g.P.Inv {
		if it.Kind == "map" {
			m++
		}
	}
	if n == 0 && m == 0 {
		return nil
	}
	return []string{fmt.Sprintf("{{y|Hunt: %d bounty, %d map (B)}}", n, m)}
}

func bhLog(g *Game) {
	st := bhSt(g)
	var lines []string
	for _, it := range g.P.Inv {
		if c := bhMapCache(g, it); c != nil {
			lines = append(lines, "{{Y|"+it.Name+"}}", bhMapText(g, c, st.Clue[c.ID]), "")
		}
	}
	for _, b := range bhActive(g) {
		s := "hunting"
		switch {
		case b.Cap:
			s = "prisoner in tow"
		case b.Dead:
			s = "slain: hand in the proof"
		}
		lines = append(lines, fmt.Sprintf("{{R|%s}} (%s, %d %s, for %s)", b.Name, s, b.Reward, g.Sk.Currency, g.realmName(b.State)), b.Poster, "")
	}
	if len(lines) == 0 {
		lines = []string{"No maps and no bounties. Read a bounty board (ask a warden) or ask a dealer in the Warrens."}
	}
	g.showPopup("Hunts and bounties", strings.Join(lines, "\n"))
}
