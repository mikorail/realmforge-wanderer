package engine

// Traces of lives in towns and villages (prefix eh, state g.ext("esh"), planet-local).
//
// Built only from simulated data: the dead of the chronicle (PersonData), the dead, jailed, exiled and
// shopless of the life simulation (lvPerson, lsState), wars and peaces, plagues (Hazards outbreaks and
// chronicle sicknesses), fires, luminaries, falls of realms, coups and poisonings. Every scene is keyed by
// the event index or person key it comes from and is stored in g.ext("esh"), so its text is stable.
//
//   graves      epitaphs on the temple graveyard (enriched in place), village plots, a hero's cenotaph, an
//               empty grave for a ruler who fled, a child's grave with toys, deposed rulers, the hanged and a
//               lonely cairn outside the wall (outskirts)
//   memorials   war dead (real war name, years, count, names), plague plaque, a ruler's statue with a deed
//   houses      plague house, burnt house, abandoned workshop, shuttered shop, a house the watch emptied, a
//               cradle with a mourning cloth, a hoarder's room: props inside a real free house of the zone
//   public      notice board (letters.go news and bounties, refreshed on entry), wayside shrine (pack
//               festival offerings and saints), scratched graffiti, a lookout stone pointing at real places
//   unquiet     one per planet: the gallows tree of someone hanged for a ruler's poisoning, a soft night payoff
//
// Payoffs: case-notes facts (knLearn), journal secrets, XP, a little loot, keepsakes returned to a living
// relative through a talk option (lvMemoryAdd/remember). Zone hook paints with side rngs only
// (seed:es:<part>:<key>); scenes from the life simulation are added by the turn hook on entering a zone.

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

func init() {
	unLocalKeys = append(unLocalKeys, "esh")
	zoneHooks = append(zoneHooks, ehZone)
	turnHooks = append(turnHooks, ehTurn)
	talkHooks = append(talkHooks, ehTalk)
}

// ---------------------------------------------------------------- state

type ehScene struct {
	ID     string `json:"i"`
	Kind   string `json:"k"`
	Zone   string `json:"z"`
	Burg   int    `json:"b,omitempty"`
	X0     int    `json:"x0"`
	Y0     int    `json:"y0"`
	X1     int    `json:"x1"`
	Y1     int    `json:"y1"`
	Look   string `json:"l,omitempty"`  // said once when you step inside the rectangle
	Seen   bool   `json:"sn,omitempty"` //
	Read   string `json:"r,omitempty"`  // g.Read key that triggers the payoff
	Fact   string `json:"f,omitempty"`  // case-notes fact
	Subj   string `json:"sj,omitempty"` // person key the fact is about (joins an open case on them)
	Secret string `json:"s,omitempty"`
	SecP   int    `json:"sp,omitempty"` // parasang + 1 of the secret
	Marks  []int  `json:"m,omitempty"`  // parasangs marked on the map
	XP     int    `json:"xp,omitempty"`
	Night  bool   `json:"nt,omitempty"` // payoff only at night, standing near
	Title  string `json:"tt,omitempty"`
	Reveal string `json:"rv,omitempty"`
	Paid   bool   `json:"pd,omitempty"`
}

type ehKeep struct {
	Name    string   `json:"n"`
	Dead    string   `json:"d"`
	DeadKey string   `json:"dk,omitempty"`
	Kin     []string `json:"k"`
	Done    bool     `json:"o,omitempty"`
}

type ehState struct {
	Scenes   map[string]*ehScene `json:"s,omitempty"`
	Order    []string            `json:"o,omitempty"`
	Zones    map[string]int      `json:"z,omitempty"`  // static scenes per zone
	Dyn      map[string]int      `json:"y,omitempty"`  // life-simulation scenes per zone
	Keep     map[int]*ehKeep     `json:"k,omitempty"`  // keepsake item id -> errand
	Board    map[string]int      `json:"bd,omitempty"` // zone key -> tile of the notice board
	BoardDay map[string]int      `json:"bq,omitempty"`

	last string
}

func ehSt(g *Game) *ehState {
	s := g.ext("esh", func() any { return &ehState{} }).(*ehState)
	if s.Scenes == nil {
		s.Scenes = map[string]*ehScene{}
	}
	if s.Zones == nil {
		s.Zones = map[string]int{}
	}
	if s.Dyn == nil {
		s.Dyn = map[string]int{}
	}
	if s.Keep == nil {
		s.Keep = map[int]*ehKeep{}
	}
	if s.Board == nil {
		s.Board = map[string]int{}
	}
	if s.BoardDay == nil {
		s.BoardDay = map[string]int{}
	}
	return s
}

func (s *ehState) has(id string) bool { return s.Scenes[id] != nil }

func (s *ehState) add(sc *ehScene) {
	if s.Scenes[sc.ID] != nil {
		return
	}
	s.Scenes[sc.ID] = sc
	s.Order = append(s.Order, sc.ID)
}

// ---------------------------------------------------------------- small helpers

func ehRep(s string, kv ...string) string { return strings.NewReplacer(kv...).Replace(s) }

func ehFirst(name string) string {
	if i := strings.Index(name, " "); i > 0 {
		return name[:i]
	}
	return name
}

// ehSentence cuts a chronicle line at its first sentence end, at most n characters.
func ehSentence(s string, n int) string {
	if k := strings.Index(s, ": "); k > 0 && k < 40 && s[:k] == strings.ToUpper(s[:k]) {
		s = s[k+2:] // drop chronicle headers like "TAX REVOLT: "
	}
	for i := 0; i < len(s)-1; i++ {
		if (s[i] == '.' || s[i] == '!') && s[i+1] == ' ' && i > 20 {
			s = s[:i+1]
			break
		}
	}
	if len(s) > n {
		cut := strings.LastIndex(s[:n], " ")
		if cut < 20 {
			cut = n
		}
		s = s[:cut] + "..."
	}
	return s
}

func ehPickH(list []string, parts ...string) string {
	if len(list) == 0 {
		return ""
	}
	return list[int(HashStr(strings.Join(parts, ":"))%uint64(len(list)))]
}

var ehDirs4 = [][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}}

var ehGround = map[Terrain]bool{TCobble: true, TDirt: true, TGrass: true, TSand: true, TSnow: true, TMoss: true, TTallGrass: true}

var ehWalls = map[Terrain]bool{TWall: true, TStoneWall: true, TWoodWall: true, TBrickWall: true, TMarbleWall: true, TObsidianWall: true,
	TBoneWall: true, TAdobeWall: true, TBambooWall: true, TCoralWall: true, TMetalWall: true}

var ehFloors = map[Terrain]bool{TFloor: true, TTileFloor: true, TWoodFloor: true, TCarpet: true, TBed: true}

var ehBusy = map[Terrain]bool{TAltar: true, TStall: true, TThrone: true, TShelf: true, TAnvil: true, TBrazier: true, TDown: true, TUp: true,
	TInscription: true, TStatue: true, TSarcophagus: true, TMachine: true, TPortal: true, TWell: true, TCrate: true, TDummy: true, TLocked: true, TBarrel: true}

// ---------------------------------------------------------------- zone context

type ehCtx struct {
	g     *Game
	zn    *Zone
	bd    *BurgData
	kind  string
	st    *ehState
	n     int
	max   int
	dyn   bool
	rooms []*ehRoom
}

func (c *ehCtx) rng(part, key string) *Rng { return NewRng(c.g.Seed + ":es:" + part + ":" + key) }
func (c *ehCtx) full() bool                { return c.n >= c.max }
func (c *ehCtx) town() string              { return c.bd.Name }
func (c *ehCtx) realm() string             { return c.g.W.StateName(c.bd.State) }
func (c *ehCtx) tileKey(x, y int) string   { return c.zn.Key + strconv.Itoa(y*MW+x) }

func (c *ehCtx) scene(id, kind string, x0, y0, x1, y1 int) *ehScene {
	sc := &ehScene{ID: id, Kind: kind, Zone: c.zn.Key, Burg: c.bd.ID, X0: x0, Y0: y0, X1: x1, Y1: y1, SecP: c.bd.P + 1}
	c.st.add(sc)
	c.n++
	if c.dyn {
		c.st.Dyn[c.zn.Key]++
	} else {
		c.st.Zones[c.zn.Key]++
	}
	return sc
}

func (c *ehCtx) nearStreet(x, y int) bool { return abs(y-12) <= 1 || abs(x-40) <= 1 }

func (c *ehCtx) freeTile(x, y int) bool {
	zn := c.zn
	if x < 2 || y < 2 || x > MW-3 || y > MH-3 || !ehGround[zn.At(x, y)] {
		return false
	}
	return zn.ActorAt(x, y) == nil && len(zn.ItemsAt(x, y)) == 0 && zn.Notes[y*MW+x] == ""
}

// stripOK: a 1-row strip of w blocking props at (x,y) on free ground, with a walkable ring around it (the ring
// is a closed loop, so the zone stays connected) and away from the main streets.
func (c *ehCtx) stripOK(x, y, w int) bool {
	zn := c.zn
	for dy := -1; dy <= 1; dy++ {
		for dx := -1; dx <= w; dx++ {
			xx, yy := x+dx, y+dy
			if !zn.In(xx, yy) || c.nearStreet(xx, yy) {
				return false
			}
			if dy == 0 && dx >= 0 && dx < w {
				if !c.freeTile(xx, yy) {
					return false
				}
				continue
			}
			t := zn.At(xx, yy)
			if !zn.Walkable(xx, yy) || t == TDoor || t == TDoorOpen || t == TDown || t == TUp || t == TPortal || t.D().Water > 0 || zn.ActorAt(xx, yy) != nil {
				return false
			}
		}
	}
	return true
}

func (c *ehCtx) open(r *Rng, w int, pred func(x, y int) bool) (int, int, bool) {
	for i := 0; i < 600; i++ {
		x, y := r.Int(3, MW-4-w), r.Int(3, MH-4)
		if (pred == nil || pred(x, y)) && c.stripOK(x, y, w) {
			return x, y, true
		}
	}
	return 0, 0, false
}

// wallSpot: w wall tiles in a row of a building facing open ground (a plaque or a scratch on the wall). Walls
// never carry the path, so turning them into inscriptions keeps the zone connected.
func (c *ehCtx) wallSpot(r *Rng, w int) (int, int, bool) {
	zn := c.zn
	for i := 0; i < 900; i++ {
		x, y := r.Int(3, MW-4-w), r.Int(2, MH-3)
		ok := ehWalls[zn.At(x-1, y)] && ehWalls[zn.At(x+w, y)]
		for side := -1; side <= 1 && ok; side += 2 {
			good := true
			for dx := 0; dx < w; dx++ {
				xx := x + dx
				if !ehWalls[zn.At(xx, y)] || zn.Notes[y*MW+xx] != "" {
					good, ok = false, false
					break
				}
				o := zn.At(xx, y+side)
				if !ehGround[o] && o != TRoad || zn.ActorAt(xx, y+side) != nil {
					good = false
				}
			}
			if good {
				return x, y, true
			}
		}
	}
	return 0, 0, false
}

// ---------------------------------------------------------------- houses

type ehRoom struct {
	cells          []int
	set            map[int]bool
	door, inner    int
	x0, y0, x1, y1 int
}

func (c *ehCtx) findRooms() []*ehRoom {
	zn := c.zn
	seen := make([]bool, MW*MH)
	occ := map[int]bool{}
	for _, a := range zn.Actors {
		occ[a.Y*MW+a.X] = true
	}
	for _, it := range zn.Items {
		occ[it.Y*MW+it.X] = true
	}
	if c.g.P != nil && c.g.Cur == zn {
		occ[c.g.P.Y*MW+c.g.P.X] = true
	}
	var out []*ehRoom
	for i := 0; i < MW*MH; i++ {
		if seen[i] || !ehFloors[zn.T[i]] {
			continue
		}
		rm := &ehRoom{door: -1, inner: -1, set: map[int]bool{}, x0: MW, y0: MH, x1: -1, y1: -1}
		bad := false
		q := []int{i}
		seen[i] = true
		for len(q) > 0 {
			j := q[0]
			q = q[1:]
			rm.cells = append(rm.cells, j)
			rm.set[j] = true
			x, y := j%MW, j/MW
			rm.x0, rm.y0, rm.x1, rm.y1 = mini(rm.x0, x), mini(rm.y0, y), maxi(rm.x1, x), maxi(rm.y1, y)
			if x <= 0 || y <= 0 || x >= MW-1 || y >= MH-1 || occ[j] || zn.Notes[j] != "" {
				bad = true
			}
			if c.dyn && zn.Seen != nil && zn.Seen[j] > 0 {
				bad = true // never change what the player has already looked at
			}
			for dy := -1; dy <= 1; dy++ {
				for dx := -1; dx <= 1; dx++ {
					if ehBusy[zn.At(x+dx, y+dy)] {
						bad = true
					}
				}
			}
			for _, d := range ehDirs4 {
				nx, ny := x+d[0], y+d[1]
				if !zn.In(nx, ny) {
					continue
				}
				k := ny*MW + nx
				t := zn.T[k]
				switch {
				case t == TDoor || t == TDoorOpen:
					if rm.door < 0 {
						rm.door, rm.inner = k, j
					}
				case ehFloors[t]:
					if !seen[k] {
						seen[k] = true
						q = append(q, k)
					}
				case t.D().Walk:
					bad = true // open to the street: a porch, not a closed house
				}
			}
		}
		if bad || rm.door < 0 || len(rm.cells) < 6 || len(rm.cells) > 70 || c.usedRect(rm.x0, rm.y0, rm.x1, rm.y1) {
			continue
		}
		out = append(out, rm)
	}
	return out
}

func (c *ehCtx) usedRect(x0, y0, x1, y1 int) bool {
	for _, id := range c.st.Order {
		sc := c.st.Scenes[id]
		if sc != nil && sc.Zone == c.zn.Key && sc.X0 <= x1+1 && sc.X1 >= x0-1 && sc.Y0 <= y1+1 && sc.Y1 >= y0-1 {
			return true
		}
	}
	return false
}

// room takes one free house, chosen by r; nil when none is left.
func (c *ehCtx) room(r *Rng) *ehRoom {
	if c.rooms == nil {
		c.rooms = c.findRooms()
	}
	var live []*ehRoom
	for _, rm := range c.rooms {
		if !c.usedRect(rm.x0, rm.y0, rm.x1, rm.y1) {
			live = append(live, rm)
		}
	}
	if len(live) == 0 {
		return nil
	}
	return live[r.Int(0, len(live)-1)]
}

func (c *ehCtx) roomConnected(rm *ehRoom) bool {
	zn := c.zn
	if !zn.Walkable(rm.inner%MW, rm.inner/MW) {
		return false
	}
	want := 0
	for _, j := range rm.cells {
		if zn.T[j].D().Walk {
			want++
		}
	}
	seen := map[int]bool{rm.inner: true}
	q := []int{rm.inner}
	for len(q) > 0 {
		j := q[0]
		q = q[1:]
		for _, d := range ehDirs4 {
			k := j + d[0] + d[1]*MW
			if rm.set[k] && !seen[k] && zn.T[k].D().Walk {
				seen[k] = true
				q = append(q, k)
			}
		}
	}
	return len(seen) == want
}

// edge: a floor cell against the wall, not beside the door.
func (c *ehCtx) edge(rm *ehRoom, j int) bool {
	if j == rm.inner || c.zn.T[j] == TBed {
		return false
	}
	x, y := j%MW, j/MW
	wall := false
	for _, d := range ehDirs4 {
		t := c.zn.At(x+d[0], y+d[1])
		if t == TDoor || t == TDoorOpen {
			return false
		}
		if !t.D().Walk {
			wall = true
		}
	}
	return wall
}

// block places up to n blocking props on wall-side cells, keeping every floor cell reachable from the door.
func (c *ehCtx) block(rm *ehRoom, r *Rng, t Terrain, n int) int {
	var cand []int
	for _, j := range rm.cells {
		if c.edge(rm, j) && c.zn.T[j] != t && c.zn.T[j].D().Walk {
			cand = append(cand, j)
		}
	}
	for i := len(cand) - 1; i > 0; i-- {
		k := r.Int(0, i)
		cand[i], cand[k] = cand[k], cand[i]
	}
	put := 0
	for _, j := range cand {
		if put >= n {
			break
		}
		old := c.zn.T[j]
		c.zn.T[j] = t
		if !c.roomConnected(rm) {
			c.zn.T[j] = old
			continue
		}
		put++
	}
	return put
}

// scatter repaints a share of the room's plain floor with a walkable terrain (ash, rubble, bones).
func (c *ehCtx) scatter(rm *ehRoom, r *Rng, t Terrain, share float64) {
	for _, j := range rm.cells {
		if ehFloors[c.zn.T[j]] && c.zn.T[j] != TBed && r.Chance(share) {
			c.zn.T[j] = t
		}
	}
}

func (c *ehCtx) drop(rm *ehRoom, r *Rng, it *Item) {
	var cand []int
	for _, j := range rm.cells {
		if j != rm.inner && c.zn.T[j].D().Walk && len(c.zn.ItemsAt(j%MW, j/MW)) == 0 {
			cand = append(cand, j)
		}
	}
	if len(cand) == 0 {
		cand = []int{rm.inner}
	}
	j := cand[r.Int(0, len(cand)-1)]
	it.X, it.Y = j%MW, j/MW
	c.zn.Items = append(c.zn.Items, it)
}

func (c *ehCtx) dropAt(x, y int, it *Item) {
	it.X, it.Y = x, y
	c.zn.Items = append(c.zn.Items, it)
}

func (c *ehCtx) houseScene(id, kind string, rm *ehRoom) *ehScene {
	sc := c.scene(id, kind, rm.x0, rm.y0, rm.x1, rm.y1)
	sc.Look = ehHouseLook[kind]
	return sc
}

// ---------------------------------------------------------------- items

func (c *ehCtx) note(title, text string) *Item {
	return &Item{ID: c.g.newID(), Kind: "book", Name: title, Glyph: '■', F: CWhite, Text: text, Value: 2, Weight: 1}
}

func (c *ehCtx) thing(name, lore string, glyph rune, f byte, val int) *Item {
	return &Item{ID: c.g.newID(), Kind: "trophy", Name: name, Glyph: glyph, F: f, Lore: lore, Value: val, Weight: 1}
}

// keepsake: a dead person's small thing that a living relative would want back.
func (c *ehCtx) keepsake(dead *lvPerson, what string) *Item {
	var kin []string
	var names []string
	for _, k := range dead.Kin {
		if q := lvGet(c.g, k); q != nil && q.Alive {
			kin = append(kin, k)
			if len(names) < 2 {
				names = append(names, q.Name)
			}
		}
	}
	if len(kin) == 0 {
		return nil
	}
	it := c.thing(ehFirst(dead.Name)+"'s "+what, fmt.Sprintf("It belonged to %s of %s. %s would want it back.", dead.Name, c.town(), strings.Join(names, " or ")), '♦', CYellow, 3)
	it.Extra = map[string]int{"eh:ks": 1}
	c.st.Keep[it.ID] = &ehKeep{Name: it.Name, Dead: dead.Name, DeadKey: dead.Key, Kin: kin}
	return it
}

// ---------------------------------------------------------------- voice and epitaphs

func ehCat(cause string) (cat, c, w string) {
	l := strings.ToLower(cause)
	switch {
	case l == "":
		return "other", "", ""
	case strings.Contains(l, "old age"):
		return "old", cause, ""
	case l == "battle wounds":
		return "battle", cause, ""
	case strings.HasPrefix(l, "in the war against"):
		return "war", cause, "the" + strings.TrimPrefix(cause, "in the")
	case strings.HasPrefix(l, "of the "):
		return "plague", cause, strings.TrimPrefix(cause, "of ")
	case strings.Contains(l, "spies of"):
		return "spies", "the spies of" + cause[strings.Index(l, "spies of")+8:], ""
	case strings.Contains(l, "poison"):
		return "poison", cause, ""
	case strings.Contains(l, "fever") || strings.Contains(l, "sickness"):
		return "sick", cause, ""
	case strings.Contains(l, "accident") || strings.Contains(l, "fall from"):
		return "accident", strings.TrimPrefix(cause, "in "), ""
	case strings.HasPrefix(l, "deposed") || l == "the coup":
		return "deposed", cause, ""
	case strings.Contains(l, "devoured") || strings.Contains(l, "rampage"):
		return "monster", "Devoured in a titan's rampage", ""
	case strings.HasPrefix(l, "murdered"):
		return "murder", cause, ""
	case strings.Contains(l, "gallows"):
		return "gallows", cause, ""
	case strings.Contains(l, "duel"):
		return "duel", cause, ""
	}
	return "other", cause, ""
}

// ehVerse picks a closing line in the culture's voice: a funeral rite, a proverb, the ideal death.
func (g *Game) ehVerse(culture int, cat, key string) string {
	if cp := g.W.cpOf(culture); cp != nil {
		var opts []string
		if cp.Customs != nil {
			for _, rt := range cp.Customs.Rites {
				if rt.ID == "funeral" && rt.Name != "" {
					rn := strings.TrimPrefix(strings.TrimPrefix(rt.Name, "The "), "the ")
					opts = append(opts, "Sent on the "+rn+".")
					if len(rt.Steps) > 0 {
						opts = append(opts, "Sent on the "+rn+". "+g.W.cpChoose(rt.Steps, "eh", key))
					}
				}
			}
			if id := cp.Customs.Ideals; id != nil && id.Death != "" && (cat == "battle" || cat == "war" || cat == "monster") {
				opts = append(opts, id.Death, id.Death)
			}
		}
		if cp.Voice != nil && len(cp.Voice.Proverbs) > 0 {
			opts = append(opts, "\""+g.W.cpChoose(cp.Voice.Proverbs, "eh:prov", key)+"\"")
		}
		if len(opts) > 0 {
			return cpFill(g.W.cpChoose(opts, "eh:verse", key), nil)
		}
	}
	return ehPickH(ehVerse, g.Seed, "eh:verse", key)
}

func (g *Game) ehFamilyLine(cat, kin, key string) string {
	var opts []string
	for _, s := range ehFamily[cat] {
		if kin != "" || !strings.Contains(s, "[k]") {
			opts = append(opts, s)
		}
	}
	if len(opts) == 0 {
		return "The stone is kept clean."
	}
	return ehRep(ehPickH(opts, g.Seed, "eh:fam", key), "[k]", kin)
}

// ehEpitaph lays out a stone: name and title, dates and how they died, a verse, what the family says.
func (g *Game) ehEpitaph(name, title, realm, town string, born, died int, cause string, culture int, kin, key string) string {
	cat, cs, wn := ehCat(cause)
	line := ehRep(ehPickH(ehCauseLine[cat], g.Seed, "eh:cause", key), "[c]", cs, "[w]", wn, "[t]", town)
	head := "{{Y|" + name + "}}"
	if title != "" {
		head += ", " + title
	}
	if realm != "" {
		head += " of " + realm
	}
	dates := fmt.Sprintf("Died in the year %d.", died)
	if born > 0 && born < died {
		dates = fmt.Sprintf("Born in the year %d, died in the year %d.", born, died)
	}
	return fmt.Sprintf("%s\n%s %s\n\n%s\n\n{{K|%s}}", head, dates, line, g.ehVerse(culture, cat, key), g.ehFamilyLine(cat, kin, key))
}

func (g *Game) ehKinOf(p *lvPerson) string {
	if q := lvGet(g, p.Spouse); q != nil && q.Alive {
		return q.Name
	}
	for _, k := range p.Kin {
		if q := lvGet(g, k); q != nil && q.Alive {
			return q.Name
		}
	}
	return ""
}

// ehNotableKin: a living townsperson of the same house as a dead notable, when the town is simulated.
func (g *Game) ehNotableKin(pp *PersonData, burg int) string {
	last := lsLast(pp.Name)
	if last == "" || !lvSt(g).Seeded[burg] {
		return ""
	}
	for _, q := range lvPresent(g, burg) {
		if lsLast(q.Name) == last {
			return q.Name
		}
	}
	return ""
}

func (c *ehCtx) personEpitaph(pp *PersonData) string {
	title, realm := nonEmpty(pp.Title, titleCase(pp.Role)), c.g.W.StateName(pp.State)
	if strings.Contains(title, realm) {
		realm = ""
	}
	return c.g.ehEpitaph(pp.Name, title, realm, c.town(), pp.Born, pp.Died, pp.Cause, pp.Culture, c.g.ehNotableKin(pp, c.bd.ID), "pd:"+strconv.Itoa(pp.ID))
}

func (c *ehCtx) lvEpitaph(p *lvPerson) string {
	return c.g.ehEpitaph(p.Name, lsJobName(p), "", c.town(), p.Born, p.Died, p.Cause, p.Culture, c.g.ehKinOf(p), "lv:"+p.Key)
}

// ---------------------------------------------------------------- world readings (pure)

type ehWar struct {
	Name       string
	Start, End int
	Text       string
}

var ehWarCache = map[*WorldData][]ehWar{}

var ehWarRe = []*regexp.Regexp{
	regexp.MustCompile(`as the ((?:[\p{Lu}][\p{L}'’\-–]*\s)*(?:War|Conquest|Crusade|Invasion|Campaign)(?: of [\p{Lu}][\p{L}'’\-]*)?) erupts`),
	regexp.MustCompile(`Banners unfurl for the ((?:[\p{Lu}][\p{L}'’\-–]*\s)*War(?: of [\p{Lu}][\p{L}'’\-]*)?)`),
}

func ehWars(w *WorldData) []ehWar {
	if l, ok := ehWarCache[w]; ok {
		return l
	}
	var out []ehWar
	seen := map[string]bool{}
	for _, e := range w.Events {
		if e.Kind != "war" {
			continue
		}
		for _, re := range ehWarRe {
			if m := re.FindStringSubmatch(e.Text); m != nil && !seen[m[1]] {
				seen[m[1]] = true
				wr := ehWar{Name: m[1], Start: e.Y, Text: e.Text}
				for _, p := range w.Events {
					if p.Kind == "peace" && p.Y >= e.Y && strings.Contains(p.Text, m[1]) {
						wr.End = p.Y
						break
					}
				}
				out = append(out, wr)
				break
			}
		}
	}
	ehWarCache[w] = out
	return out
}

var ehSickRe = regexp.MustCompile(`sickness the people call the ([^.;,]+?) breaks out in ([^;.,]+)`)

// plague: the outbreak in town now, or the last sickness the chronicle says broke out here.
func (c *ehCtx) plague() (name string, year int, dis *HzDisease) {
	w := c.g.W
	if ob := w.hzOutbreakAt(c.bd.P); ob != nil {
		return lsOutbreakName(ob), ob.Since, w.hzDisease(ob.Disease)
	}
	for i := len(w.Events) - 1; i >= 0; i-- {
		if m := ehSickRe.FindStringSubmatch(w.Events[i].Text); m != nil && strings.TrimSpace(m[2]) == c.bd.Name {
			return "the " + m[1], w.Events[i].Y, nil
		}
	}
	return "", 0, nil
}

func ehAt(e *EventData, bd *BurgData) bool { return e.P == bd.P || lmContainsWord(e.Text, bd.Name) }

var ehFireRe = regexp.MustCompile(`(?i)\b(flames?|fires?|burn(?:s|ed|ing|t)?|ablaze|alight|sacked)\b`)

func (c *ehCtx) fireEvent() int {
	w := c.g.W
	for i := len(w.Events) - 1; i >= 0; i-- {
		e := &w.Events[i]
		switch e.Kind {
		case "capture", "disaster", "war", "uprising", "eco_disaster", "monster":
			if ehAt(e, c.bd) && ehFireRe.MatchString(e.Text) {
				return i
			}
		}
	}
	return -1
}

var ehLumRe = regexp.MustCompile(`^(?:(Philosopher|Archmage|Master Artificer|Master Artisan) (\S+) of |(\S+) \((Philosopher|Archmage|Master Artificer|Master Artisan)\))`)

func (c *ehCtx) luminary() (idx int, name, role string) {
	w := c.g.W
	for i := len(w.Events) - 1; i >= 0; i-- {
		e := &w.Events[i]
		if e.Kind != "luminary" || !lmContainsWord(e.Text, c.bd.Name) {
			continue
		}
		if m := ehLumRe.FindStringSubmatch(e.Text); m != nil {
			if m[1] != "" {
				return i, m[2], m[1]
			}
			return i, m[3], m[4]
		}
	}
	return -1, "", ""
}

func (g *Game) ehPersonNamed(first string) *PersonData {
	for i := range g.W.People {
		if ehFirst(g.W.People[i].Name) == first {
			return &g.W.People[i]
		}
	}
	return nil
}

// ---------------------------------------------------------------- the zone hook

func ehKind(g *Game, zn *Zone) (string, *BurgData) {
	w := g.W
	if w == nil || zn.Z != 0 || zn.Sky || zn.Underwater || !w.In(zn.WX, zn.WY) {
		return "", nil
	}
	switch zn.Kind {
	case "town":
		bd := w.burgByID[zn.Burg]
		if bd == nil || !bd.Alive {
			return "", nil
		}
		if p := w.Idx(zn.WX, zn.WY); w.Burg[p] == bd.ID && g.cityPlanFor(bd).tier == 0 {
			return "village", bd
		}
		if k, b := g.districtAt(zn.WX, zn.WY, zn.ZX, zn.ZY); b != nil && b.ID == bd.ID && k != "" {
			return k, bd
		}
	case "outskirts":
		bd := w.burgByID[w.Burg[w.Idx(zn.WX, zn.WY)]]
		if bd != nil && bd.Alive {
			return "outskirts", bd
		}
	}
	return "", nil
}

func ehZone(g *Game, zn *Zone, _ *Rng) {
	kind, bd := ehKind(g, zn)
	if bd == nil {
		return
	}
	st := ehSt(g)
	if st.Zones[zn.Key] > 0 {
		return
	}
	c := &ehCtx{g: g, zn: zn, bd: bd, kind: kind, st: st, max: 3}
	switch kind {
	case "temple":
		c.enrichGraves()
		c.oddGraves()
	case "village":
		c.max = 5
		c.villageGraves()
		c.oddGraves()
		c.warMemorial()
		c.plaguePlaque()
		c.statue()
		c.board()
		c.plagueHouse()
		c.burntHouse()
		c.workshop()
		c.shrine()
		c.graffiti()
		c.lookout()
	case "plaza", "palace":
		c.max = 4
		c.warMemorial()
		c.plaguePlaque()
		c.statue()
		c.board()
		c.lookout()
	case "barracks":
		c.warMemorial()
		c.graffiti()
	case "homes", "slums":
		c.plagueHouse()
		c.burntHouse()
		c.shrine()
		c.graffiti()
	case "artisans", "forge", "academy":
		c.workshop()
		c.burntHouse()
		c.graffiti()
	case "market", "harbor":
		c.burntHouse()
		c.graffiti()
	case "outskirts":
		c.max = 2
		c.deposedGrave()
	}
	c.unquiet()
}

// ---------------------------------------------------------------- graves (static)

// enrichGraves rewrites the temple graveyard's plain stones (city.go) into full epitaphs of the same dead.
func (c *ehCtx) enrichGraves() {
	w := c.g.W
	var idx []int
	for i, n := range c.zn.Notes {
		if strings.HasPrefix(n, "A gravestone: {{Y|") {
			idx = append(idx, i)
		}
	}
	sort.Ints(idx)
	var best *PersonData
	bestI := -1
	for _, i := range idx {
		n := c.zn.Notes[i]
		name := strings.TrimPrefix(n, "A gravestone: {{Y|")
		if k := strings.Index(name, "}}"); k > 0 {
			name = name[:k]
		}
		var pp *PersonData
		for j := range w.People {
			if q := &w.People[j]; !q.Alive && q.Name == name && q.State == c.bd.State {
				pp = q
			}
		}
		if pp == nil {
			continue
		}
		c.zn.Notes[i] = c.personEpitaph(pp)
		if best == nil || pp.Renown > best.Renown {
			best, bestI = pp, i
		}
	}
	if best != nil && !c.st.has("epi:"+strconv.Itoa(best.ID)) {
		if line := c.g.lmChronicleLine(best); line != "" {
			x, y := bestI%MW, bestI/MW
			sc := c.scene("epi:"+strconv.Itoa(best.ID), "grave", x, y, x, y)
			sc.Read, sc.Fact, sc.XP = c.tileKey(x, y), best.Name+": "+line, 10
			c.n-- // an old stone made readable, not a new scene
		}
	}
}

func (c *ehCtx) graveyardPred() func(x, y int) bool {
	if c.kind == "temple" {
		return func(x, y int) bool { return y >= 13 && x <= 36 }
	}
	return func(x, y int) bool { return abs(x-40) >= 5 && abs(y-12) >= 3 }
}

func (c *ehCtx) villageGraves() {
	w := c.g.W
	var dead []*PersonData
	for i := range w.People {
		pp := &w.People[i]
		if !pp.Alive && pp.Died > 0 && pp.Name != "" && (pp.Burg == c.bd.ID || pp.State == c.bd.State && c.bd.Capital) && !strings.HasPrefix(pp.Cause, "deposed") {
			dead = append(dead, pp)
		}
	}
	sort.SliceStable(dead, func(i, j int) bool {
		if dead[i].Renown != dead[j].Renown {
			return dead[i].Renown > dead[j].Renown
		}
		return dead[i].ID < dead[j].ID
	})
	r := c.rng("vgraves", c.zn.Key)
	placed := 0
	for _, pp := range dead {
		if placed >= 4 {
			break
		}
		x, y, ok := c.open(r, 1, c.graveyardPred())
		if !ok {
			break
		}
		c.zn.Set(x, y, TInscription)
		c.zn.Notes[y*MW+x] = c.personEpitaph(pp)
		placed++
		if placed == 1 {
			if line := c.g.lmChronicleLine(pp); line != "" {
				sc := c.scene("epi:"+strconv.Itoa(pp.ID), "grave", x, y, x, y)
				sc.Read, sc.Fact, sc.XP = c.tileKey(x, y), pp.Name+": "+line, 10
			}
		}
	}
}

// oddGraves: a hero's cenotaph (no body came home) and an empty grave dug for a ruler who fled.
func (c *ehCtx) oddGraves() {
	w := c.g.W
	id := "ceno:" + strconv.Itoa(c.bd.ID)
	if !c.full() && !c.st.has(id) {
		var hero *PersonData
		for i := range w.People {
			pp := &w.People[i]
			if pp.Alive || pp.State != c.bd.State || pp.Name == "" {
				continue
			}
			cat, _, _ := ehCat(pp.Cause)
			if cat != "monster" && cat != "battle" {
				continue
			}
			if hero == nil || (cat == "monster") != (ehCatOf(hero) == "monster") && cat == "monster" || ehCatOf(hero) == cat && pp.Renown > hero.Renown {
				hero = pp
			}
		}
		if hero != nil {
			r := c.rng("ceno", id)
			if x, y, ok := c.open(r, 1, c.graveyardPred()); ok {
				c.zn.Set(x, y, TStatue)
				war := ""
				for _, wr := range ehWars(w) {
					if hero.Died >= wr.Start && (wr.End == 0 || hero.Died <= wr.End) && strings.Contains(wr.Text, w.StateName(hero.State)) {
						war = " in the " + wr.Name
						break
					}
				}
				c.zn.Notes[y*MW+x] = fmt.Sprintf("A cenotaph: a tall stone over no grave.\n\n%s\n\n{{K|It stands for %s, who fell%s in the year %d and was never brought home. The ground in front of it is bare earth, never dug.}}",
					c.personEpitaph(hero), hero.Name, war, hero.Died)
				sc := c.scene(id, "cenotaph", x, y, x, y)
				sc.Read, sc.XP = c.tileKey(x, y), 10
				sc.Secret = fmt.Sprintf("%s of %s fell in the year %d (%s) and lies in no grave. %s keeps a cenotaph.", hero.Name, w.StateName(hero.State), hero.Died, hero.Cause, c.town())
			}
		}
	}
	id = "empty:" + strconv.Itoa(c.bd.ID)
	if c.full() || c.st.has(id) {
		return
	}
	for i := len(w.Events) - 1; i >= 0; i-- {
		e := &w.Events[i]
		if e.Kind != "fall" || e.P != c.bd.P {
			continue
		}
		name := ""
		for _, verb := range []string{" flees", " escapes"} {
			if k := strings.Index(e.Text, verb); k > 0 {
				name = e.Text[:k]
				for _, sep := range []string{". ", "; ", ", "} {
					if j := strings.LastIndex(name, sep); j >= 0 {
						name = name[j+len(sep):]
					}
				}
				break
			}
		}
		if k := strings.LastIndex(name, " and "); k >= 0 {
			name = name[k+5:]
		}
		name = strings.TrimPrefix(name, "and ")
		if name == "" || len(name) > 40 {
			continue
		}
		r := c.rng("empty", id)
		x, y, ok := c.open(r, 1, c.graveyardPred())
		if !ok {
			return
		}
		c.zn.Set(x, y, TInscription)
		if c.zn.Walkable(x, y+1) {
			c.zn.Set(x, y+1, TRubble)
		}
		c.zn.Notes[y*MW+x] = fmt.Sprintf("{{Y|%s}}\nThe date of death is left blank.\n\n{{K|The grave below the stone is open and empty, the spoil heaped beside it gone to weeds. It was dug in the year %d, when the realm fell and %s fled. Nobody has filled it in, and nobody will say why.}}", name, e.Y, name)
		sc := c.scene(id, "empty", x, y, x, y+1)
		sc.Read, sc.XP = c.tileKey(x, y), 10
		sc.Fact = fmt.Sprintf("In the year %d: %s", e.Y, ehSentence(e.Text, 200))
		return
	}
}

func ehCatOf(pp *PersonData) string {
	cat, _, _ := ehCat(pp.Cause)
	return cat
}

// deposedGrave: a deposed ruler buried outside the wall, outside the royal crypt.
func (c *ehCtx) deposedGrave() {
	w := c.g.W
	var best *PersonData
	for i := range w.People {
		pp := &w.People[i]
		if pp.Alive || !strings.HasPrefix(pp.Cause, "deposed") || pp.Name == "" || c.st.has("deposed:"+strconv.Itoa(pp.ID)) {
			continue
		}
		if pp.Burg != c.bd.ID && !(c.bd.Capital && pp.State == c.bd.State) {
			continue
		}
		if best == nil || pp.Renown > best.Renown {
			best = pp
		}
	}
	if best == nil {
		return
	}
	id := "deposed:" + strconv.Itoa(best.ID)
	x, y, ok := c.open(c.rng("deposed", id), 1, nil)
	if !ok {
		return
	}
	c.zn.Set(x, y, TInscription)
	c.zn.Notes[y*MW+x] = "A grave in rough ground outside the wall, far from the royal dead.\n\n" + c.personEpitaph(best)
	sc := c.scene(id, "outcast", x, y, x, y)
	sc.Read, sc.XP = c.tileKey(x, y), 10
	if line := c.g.lmChronicleLine(best); line != "" {
		sc.Fact = best.Name + ": " + line
	}
}

// ---------------------------------------------------------------- memorials

func (c *ehCtx) mountOrStand(r *Rng, w int, t Terrain) (int, int, bool) {
	if x, y, ok := c.wallSpot(r, w); ok {
		for dx := 0; dx < w; dx++ {
			c.zn.Set(x+dx, y, TInscription)
		}
		return x, y, true
	}
	if x, y, ok := c.open(r, w, nil); ok {
		for dx := 0; dx < w; dx++ {
			c.zn.Set(x+dx, y, t)
		}
		return x, y, true
	}
	return 0, 0, false
}

func (c *ehCtx) warMemorial() {
	if c.full() {
		return
	}
	g, w := c.g, c.g.W
	realm := w.StateName(c.bd.State)
	type cand struct {
		wr    ehWar
		names []string
	}
	var best *cand
	for _, wr := range ehWars(w) {
		if !strings.Contains(wr.Text, realm) {
			continue
		}
		end := wr.End
		if end == 0 {
			end = w.Year
		}
		cd := &cand{wr: wr}
		for i := range w.People {
			pp := &w.People[i]
			if !pp.Alive && pp.State == c.bd.State && pp.Cause == "battle wounds" && pp.Died >= wr.Start && pp.Died <= end && pp.Name != "" {
				cd.names = append(cd.names, pp.Name)
			}
		}
		if wr.End == 0 && lvSt(g).Seeded[c.bd.ID] {
			for _, p := range ehPeople(g, c.bd.ID) {
				if !p.Alive && strings.HasPrefix(p.Cause, "in the war against") {
					cd.names = append(cd.names, p.Name)
				}
			}
		}
		if len(cd.names) > 0 && (best == nil || len(cd.names) > len(best.names)) {
			best = cd
		}
	}
	if best == nil {
		return
	}
	id := "war:" + strconv.Itoa(c.bd.ID)
	if c.st.has(id) {
		return
	}
	x, y, ok := c.mountOrStand(c.rng("war", id), 3, TInscription)
	if !ok {
		return
	}
	wr := best.wr
	years := fmt.Sprintf("From the year %d to the year %d.", wr.Start, wr.End)
	if wr.End == 0 {
		years = fmt.Sprintf("From the year %d. Not yet ended.", wr.Start)
	}
	n := len(best.names)
	first := best.names
	if len(first) > 3 {
		first = first[:3]
	}
	word := "names are"
	if n == 1 {
		word = "name is"
	}
	c.zn.Notes[y*MW+x+1] = fmt.Sprintf("{{Y|FOR THE DEAD OF %s IN THE %s}}\n%s\n\n%d %s cut here. The first: %s.\n\n%s\n\n{{K|%s}}",
		strings.ToUpper(realm), strings.ToUpper(wr.Name), years, n, word, strings.Join(first, ", "), g.ehVerse(c.bd.Culture, "war", id), g.ehFamilyLine("war", "", id))
	rest := "The stone is left smooth below the names, for names still to come."
	if n > 3 {
		more := best.names[3:]
		if len(more) > 6 {
			more = more[:6]
		}
		rest = "More names: " + strings.Join(more, ", ") + "."
	}
	c.zn.Notes[y*MW+x] = rest
	c.zn.Notes[y*MW+x+2] = "Fresh flowers wedged into the cracks. A ribbon in the colours of " + realm + "."
	sc := c.scene(id, "memorial", x, y, x+2, y)
	sc.Read, sc.XP = c.tileKey(x+1, y), 10
	sc.Fact = fmt.Sprintf("%d of %s fell in the %s (from the year %d), among them %s.", n, realm, wr.Name, wr.Start, best.names[0])
}

func (c *ehCtx) plaguePlaque() {
	if c.full() {
		return
	}
	g, w := c.g, c.g.W
	name, year, _ := c.plague()
	if name == "" {
		return
	}
	id := "plague:" + strconv.Itoa(c.bd.ID)
	if c.st.has(id) {
		return
	}
	var names []string
	short := strings.TrimPrefix(name, "the ")
	if lvSt(g).Seeded[c.bd.ID] {
		for _, p := range ehPeople(g, c.bd.ID) {
			if !p.Alive && strings.Contains(p.Cause, short) {
				names = append(names, p.Name)
			}
		}
	}
	for i := range w.People {
		pp := &w.People[i]
		if !pp.Alive && pp.State == c.bd.State && pp.Died >= year && pp.Died <= year+3 && ehCatOf(pp) == "sick" && pp.Name != "" {
			names = append(names, pp.Name)
		}
	}
	if len(names) == 0 {
		return
	}
	x, y, ok := c.mountOrStand(c.rng("plague", id), 1, TInscription)
	if !ok {
		return
	}
	show := names
	if len(show) > 5 {
		show = show[:5]
	}
	c.zn.Notes[y*MW+x] = fmt.Sprintf("{{Y|%s}}, the year %d.\nIn %s and the realm around it, it took %d we can name: %s.\n\n%s\n\n{{K|Below, in chalk and more recent: boil the water, burn the bedding, do not kiss the dead.}}",
		titleCase(name), year, c.town(), len(names), strings.Join(show, ", "), g.ehVerse(c.bd.Culture, "plague", id))
	sc := c.scene(id, "plaque", x, y, x, y)
	sc.Read, sc.XP = c.tileKey(x, y), 10
	sc.Fact = fmt.Sprintf("%s reached %s in the year %d; %s died of sickness in those years.", titleCase(name), c.town(), year, names[0])
}

func (c *ehCtx) statue() {
	if c.full() {
		return
	}
	g, w := c.g, c.g.W
	id := "statue:" + strconv.Itoa(c.bd.ID)
	if c.st.has(id) {
		return
	}
	for _, ru := range g.lmDeadRulers(c.bd.State) {
		taken := false
		for _, n := range c.zn.Notes {
			if strings.Contains(n, ru.Name) {
				taken = true
				break
			}
		}
		if taken {
			continue
		}
		var deed *EventData
		for i := range w.Events {
			e := &w.Events[i]
			switch e.Kind {
			case "war", "peace", "found", "capture", "culture", "faith", "tech", "monster", "quest":
				if lmContainsWord(e.Text, ru.Name) {
					deed = e
				}
			}
			if deed != nil {
				break
			}
		}
		if deed == nil {
			continue
		}
		x, y, ok := c.open(c.rng("statue", id), 1, nil)
		if !ok {
			return
		}
		c.zn.Set(x, y, TStatue)
		scratch := ""
		switch ehCatOf(ru) {
		case "deposed":
			scratch = "\n\n{{K|Someone has scratched a word into the base: TYRANT. Someone else has tried to scrub it off.}}"
		case "poison", "spies":
			scratch = "\n\n{{K|Scratched into the base, small: WHO POURED IT?}}"
		case "old":
			scratch = "\n\n{{K|The bronze toes are rubbed bright. Folk touch them for long life.}}"
		}
		c.zn.Notes[y*MW+x] = fmt.Sprintf("A statue of {{Y|%s}}, %s of %s, born in the year %d, died in the year %d.\n\nThe plaque reads: \"%s\" (the year %d)%s",
			ru.Name, nonEmpty(ru.Title, "ruler"), w.StateName(ru.State), ru.Born, ru.Died, ehSentence(deed.Text, 200), deed.Y, scratch)
		sc := c.scene(id, "statue", x, y, x, y)
		sc.Read, sc.XP = c.tileKey(x, y), 10
		sc.Fact = fmt.Sprintf("In the year %d: %s", deed.Y, ehSentence(deed.Text, 200))
		return
	}
}

// ---------------------------------------------------------------- public traces

func (c *ehCtx) board() {
	if c.full() {
		return
	}
	if _, ok := c.st.Board[c.zn.Key]; ok {
		return
	}
	x, y, ok := c.open(c.rng("board", c.zn.Key), 1, nil)
	if !ok {
		return
	}
	c.zn.Set(x, y, TInscription)
	c.zn.Notes[y*MW+x] = "A notice board, its papers pinned three deep."
	c.st.Board[c.zn.Key] = y*MW + x
	c.st.BoardDay[c.zn.Key] = -1
	c.scene("board:"+c.zn.Key, "board", x, y, x, y)
}

func (g *Game) ehBoardText(bd *BurgData, day int) string {
	var b strings.Builder
	b.WriteString("{{c|A notice board, papers pinned three deep.}}\n")
	n := 0
	for _, it := range g.lnNewsFor(bd, day) {
		if n >= 3 {
			break
		}
		n++
		b.WriteString("\n- " + it.Text)
	}
	m := 0
	for _, l := range g.lnBoardLines(bd, day) {
		if m >= 2 {
			break
		}
		m++
		b.WriteString("\n- " + l)
	}
	if n+m == 0 {
		b.WriteString("\nOld papers, too rained on to read.")
	}
	return b.String()
}

func ehBoardRefresh(g *Game, zn *Zone, force bool) {
	st := ehSt(g)
	idx, ok := st.Board[zn.Key]
	if !ok || zn.Burg <= 0 {
		return
	}
	day := lvDay(g)
	if !force && st.BoardDay[zn.Key] == day {
		return
	}
	bd := g.W.burgByID[zn.Burg]
	if bd == nil || zn.T[idx] != TInscription {
		return
	}
	st.BoardDay[zn.Key] = day
	zn.Notes[idx] = g.ehBoardText(bd, day)
}

func (c *ehCtx) shrine() {
	if c.full() {
		return
	}
	g, w := c.g, c.g.W
	id := "shrine:" + strconv.Itoa(c.bd.ID)
	if c.st.has(id) {
		return
	}
	cu := w.cultByID[c.bd.Culture]
	rel := w.relByID[w.Religion[clampi(c.bd.P, 0, len(w.Religion)-1)]]
	to := "the household gods"
	legend := ""
	if rel != nil {
		to = rel.Deity
		if rel.Lore != nil && len(rel.Lore.Saints) > 0 {
			s := rel.Lore.Saints[w.cpIdx(len(rel.Lore.Saints), "eh:saint", strconv.Itoa(c.bd.ID))]
			to = s.Name
			legend = " " + cpFill(s.Legend, nil)
		}
	}
	feast, offs := "", []string(nil)
	if cp := w.cpOf(c.bd.Culture); cp != nil && cp.Customs != nil && len(cp.Customs.Festivals) > 0 {
		fr := cp.Customs.Festivals[w.cpIdx(len(cp.Customs.Festivals), "eh:fest", strconv.Itoa(c.bd.ID))]
		feast, offs = fr.Feast, fr.Offerings
	}
	if len(offs) == 0 && cu != nil && len(cu.Feasts) > 0 {
		f := cu.Feasts[int(HashStr(g.Seed+":eh:feast:"+strconv.Itoa(c.bd.ID))%uint64(len(cu.Feasts)))]
		feast, offs = f.Name, ehOfferings[f.Kind]
	}
	if len(offs) == 0 {
		offs = ehOfferings[""]
	}
	r := c.rng("shrine", id)
	x, y, ok := c.open(r, 1, nil)
	if !ok {
		return
	}
	c.zn.Set(x, y, TAltar)
	left := offs
	if len(left) > 2 {
		left = left[:2]
	}
	for i, o := range left {
		if dx := i*2 - 1; c.zn.Walkable(x+dx, y+1) {
			c.dropAt(x+dx, y+1, c.thing(o, "An offering left at a wayside shrine. It is not yours.", '*', CYellow, 0))
		}
	}
	why := ""
	if feast != "" {
		why = ", left for the " + feast
	}
	c.zn.Notes[y*MW+x] = fmt.Sprintf("A wayside shrine to %s.%s\n\nOn the step: %s%s.", to, legend, strings.Join(left, " and "), why)
	c.scene(id, "shrine", x, y, x, y+1)
}

func (c *ehCtx) graffiti() {
	if c.full() {
		return
	}
	w := c.g.W
	id := "graf:" + strconv.Itoa(c.bd.ID) + ":" + c.kind
	if c.st.has(id) || HashStr(c.g.Seed+":es:grafgate:"+id)%100 >= 40 {
		return
	}
	for i := len(w.Events) - 1; i >= 0; i-- {
		e := &w.Events[i]
		lines := ehGraffiti[e.Kind]
		if len(lines) == 0 || !ehAt(e, c.bd) {
			continue
		}
		x, y, ok := c.wallSpot(c.rng("graf", id), 1)
		if !ok {
			return
		}
		text := ehRep(ehPickH(lines, c.g.Seed, id), "[y]", strconv.Itoa(e.Y), "[r]", c.realm(), "[t]", c.town())
		c.zn.Set(x, y, TInscription)
		c.zn.Notes[y*MW+x] = fmt.Sprintf("Scratched deep into the wall: {{W|%s}}\n\n{{K|Under it, smaller, the year and what it is about: %d, %s}}", text, e.Y, ehSentence(e.Text, 160))
		sc := c.scene(id, "graffiti", x, y, x, y)
		sc.Read, sc.XP = c.tileKey(x, y), 5
		sc.Fact = fmt.Sprintf("In the year %d: %s", e.Y, ehSentence(e.Text, 200))
		return
	}
}

var ehCompass = []string{"east", "south-east", "south", "south-west", "west", "north-west", "north", "north-east"}

func ehDir(dx, dy int) string {
	if dx == 0 && dy == 0 {
		return "here"
	}
	// octant by slope, no trig: compare |dx| and |dy|
	ax, ay := abs(dx), abs(dy)
	var o int
	switch {
	case ay*5 < ax*2:
		o = 0
	case ax*5 < ay*2:
		o = 2
	default:
		o = 1
	}
	switch {
	case dx >= 0 && dy >= 0:
		return ehCompass[o]
	case dx < 0 && dy >= 0:
		return ehCompass[(4-o)%8]
	case dx < 0 && dy < 0:
		return ehCompass[4+o]
	}
	return ehCompass[(8-o)%8]
}

func (c *ehCtx) lookout() {
	if c.full() {
		return
	}
	w := c.g.W
	id := "look:" + strconv.Itoa(c.bd.ID)
	if c.st.has(id) || w.cityTier[c.bd.ID] == 0 && !c.bd.Capital && HashStr(c.g.Seed+":es:lookgate:"+id)%2 == 0 {
		return
	}
	px, py := c.bd.P%w.W, c.bd.P/w.W
	type tgt struct {
		text string
		p, d int
	}
	var ts []tgt
	add := func(p int, text string) {
		if p < 0 || p == c.bd.P {
			return
		}
		d := maxi(abs(p%w.W-px), abs(p/w.W-py))
		if d <= 30 {
			ts = append(ts, tgt{text, p, d})
		}
	}
	for i := range w.Wonders {
		wd := &w.Wonders[i]
		if wd.Layer == "" || wd.Layer == "surface" {
			s := wd.Name
			if wd.Destroyed {
				s = "the ruin of " + wd.Name
			}
			add(wd.P, s)
		}
	}
	for i := range w.Megas {
		add(w.Megas[i].P, w.Megas[i].Name)
	}
	for i := range w.Monsters {
		m := &w.Monsters[i]
		if !m.Slain && (m.Layer == "" || m.Layer == "surface") && m.Lair != "" {
			add(m.P, m.Lair+", where "+m.Name+" lairs")
		}
	}
	for i := range w.Burgs {
		b := &w.Burgs[i]
		if b.Alive && b.Layer == "surface" && (b.Capital || w.cityTier[b.ID] >= 1) {
			add(b.P, b.Name)
		}
	}
	if len(ts) == 0 {
		return
	}
	sort.SliceStable(ts, func(i, j int) bool {
		if ts[i].d != ts[j].d {
			return ts[i].d < ts[j].d
		}
		return ts[i].p < ts[j].p
	})
	if len(ts) > 4 {
		ts = ts[:4]
	}
	x, y, ok := c.open(c.rng("look", id), 1, nil)
	if !ok {
		return
	}
	var lines []string
	var marks []int
	for _, t := range ts {
		lines = append(lines, fmt.Sprintf("- %s, %d parasangs: %s", titleCase(ehDir(t.p%w.W-px, t.p/w.W-py)), t.d, t.text))
		marks = append(marks, t.p)
	}
	c.zn.Set(x, y, TInscription)
	c.zn.Notes[y*MW+x] = "A lookout stone, waist high, its top cut with a compass rose. Notches on the rim point out over the roofs:\n\n" + strings.Join(lines, "\n") + "\n\n{{K|Children have worn the top smooth sitting on it.}}"
	sc := c.scene(id, "lookout", x, y, x, y)
	sc.Read, sc.Marks, sc.XP = c.tileKey(x, y), marks, 5
}

// ---------------------------------------------------------------- houses (static)

func (c *ehCtx) plagueHouse() {
	if c.full() {
		return
	}
	id := "plaguehouse:" + strconv.Itoa(c.bd.ID)
	if c.st.has(id) {
		return
	}
	name, year, dis := c.plague()
	if name == "" {
		return
	}
	r := c.rng("plaguehouse", id)
	rm := c.room(r)
	if rm == nil {
		return
	}
	house := "a"
	if cu := c.g.W.cultByID[c.bd.Culture]; cu != nil && len(cu.Houses) > 0 {
		house = "the " + Pick(r, cu.Houses)
	}
	sym, cure := "a fever and a dry cough", ""
	if dis != nil {
		if len(dis.Symptoms) > 0 {
			sym = Pick(r, dis.Symptoms)
		}
		if len(dis.Cures) > 0 {
			cure = Pick(r, dis.Cures)
		}
	}
	c.scatter(rm, r, TAsh, 0.3)
	c.block(rm, r, TCrate, 1)
	c.drop(rm, r, c.thing("vinegar-soaked rag", "Tied over the mouth against the sickness. It still stinks.", '~', CGray, 0))
	cureLine := "Day 9. The physician talks through the shutter now. He will not come in."
	if cure != "" {
		cureLine = fmt.Sprintf("Day 9. They say %s helps. We have none, and the apothecary has none.", cure)
	}
	text := fmt.Sprintf("Kept in %s house, %s, the year %d.\n\nDay 1. Came home from the market with %s.\nDay 4. Now the little one too. We keep the shutters closed.\nDay 6. Someone chalked the cross on our door in the night.\n%s\n\n{{K|The rest of the pages are blank.}}", house, c.town(), year, sym, cureLine)
	book := c.note("Diary from "+house+" house in "+c.town(), text)
	c.drop(rm, r, book)
	sc := c.houseScene(id, "plague", rm)
	sc.Read, sc.XP = "book:"+book.Name, 15
	sc.Fact = fmt.Sprintf("%s came to %s in the year %d: %s.", titleCase(name), c.town(), year, sym)
	if cure != "" {
		sc.Secret = fmt.Sprintf("In %s, the sick of %s swore by %s.", c.town(), name, cure)
	}
}

func (c *ehCtx) burntHouse() {
	if c.full() {
		return
	}
	ei := c.fireEvent()
	if ei < 0 {
		return
	}
	id := "burnt:" + strconv.Itoa(c.bd.ID)
	if c.st.has(id) {
		return
	}
	e := &c.g.W.Events[ei]
	r := c.rng("burnt", id)
	rm := c.room(r)
	if rm == nil {
		return
	}
	c.scatter(rm, r, TAsh, 0.65)
	c.scatter(rm, r, TRubble, 0.12)
	c.drop(rm, r, c.thing(Pick(r, []string{"charred cradle-rocker", "blackened cooking pot", "half-burnt chair leg", "melted lantern"}), "What the fire left.", '%', CDarkGray, 0))
	page := c.note("Scorched page from "+c.town(), fmt.Sprintf("Burnt at the edges. What can be read:\n\n\"...the year %d. %s ... we got the children out by the back. The rest went up...\"", e.Y, ehSentence(e.Text, 150)))
	c.drop(rm, r, page)
	c.drop(rm, r, &Item{ID: c.g.newID(), Kind: "gold", Name: c.g.Sk.Currency, Glyph: '$', F: CYellow, Count: 8 + r.Int(0, 20)})
	sc := c.houseScene(id, "burnt", rm)
	sc.Look += " Something glints under the ash by the hearth."
	sc.Read, sc.XP = "book:"+page.Name, 10
	sc.Fact = fmt.Sprintf("In the year %d, %s burned: %s", e.Y, c.town(), ehSentence(e.Text, 180))
}

func (c *ehCtx) workshop() {
	if c.full() {
		return
	}
	ei, name, role := c.luminary()
	if ei < 0 {
		return
	}
	id := "work:" + strconv.Itoa(c.bd.ID)
	if c.st.has(id) {
		return
	}
	g, w := c.g, c.g.W
	e := &w.Events[ei]
	fate := "has not been seen at the bench since."
	if pp := g.ehPersonNamed(name); pp != nil {
		switch {
		case !pp.Alive:
			fate = fmt.Sprintf("died in the year %d%s.", pp.Died, causeText(pp.Cause))
		case pp.Burg != c.bd.ID && w.burgByID[pp.Burg] != nil:
			fate = "left for " + w.burgByID[pp.Burg].Name + " and never sent for the tools."
		default:
			return // still here and working: not abandoned
		}
	}
	r := c.rng("work", id)
	rm := c.room(r)
	if rm == nil {
		return
	}
	mech := false
	switch g.Genre {
	case "steampunk", "dieselpunk", "atompunk", "cyberpunk", "biopunk", "solarpunk", "wasteland":
		mech = true
	}
	prop := TAnvil
	thing := "unfinished statuette, the face only roughed out"
	switch role {
	case "Master Artificer":
		thing = "half-built astrolabe, three gears short"
		if mech {
			prop, thing = TMachine, "half-built engine, its boiler unplumbed"
		}
	case "Archmage":
		prop, thing = TShelf, "unfinished ward-lattice of silver wire"
	case "Philosopher":
		prop, thing = TShelf, "stack of loose pages, a treatise with no last chapter"
	}
	c.block(rm, r, prop, 2)
	c.drop(rm, r, c.thing(thing, fmt.Sprintf("Left on %s's bench in %s.", name, c.town()), '&', CYellow, 40))
	opus := ""
	if m := regexp.MustCompile(`"([^"]+)"`).FindStringSubmatch(e.Text); m != nil {
		opus = fmt.Sprintf("Already finished: \"%s\". Everyone wants the next one.\n", m[1])
	}
	book := c.note(name+"'s workbook", fmt.Sprintf("In a quick, cramped hand:\n\n%s%s\nNext: the one that will matter. Need a better lens, a steadier hand, a quiet year.\n\n{{K|The last entry stops mid-line. A note pinned to the cover, in another hand, says %s %s}}",
		opus, ehSentence(e.Text, 160), name, fate))
	c.drop(rm, r, book)
	sc := c.houseScene(id, "workshop", rm)
	sc.Read, sc.XP = "book:"+book.Name, 20
	sc.Fact = fmt.Sprintf("In the year %d: %s %s %s", e.Y, ehSentence(e.Text, 160), name, fate)
}

// ---------------------------------------------------------------- the unquiet place (one per planet)

type ehUnq struct {
	dead   *PersonData
	burg   *BurgData
	goat   string
	truth  string
	zx, zy int
}

var ehUnqCache = map[*WorldData]*ehUnq{}

func (g *Game) ehUnquiet() *ehUnq {
	w := g.W
	if u, ok := ehUnqCache[w]; ok {
		return u
	}
	ehUnqCache[w] = nil
	var cands []*PersonData
	for i := range w.People {
		pp := &w.People[i]
		if !pp.Alive && pp.Name != "" && (strings.Contains(pp.Cause, "poison") || strings.HasPrefix(pp.Cause, "deposed in a military coup")) {
			cands = append(cands, pp)
		}
	}
	rank := func(p *PersonData) int {
		switch {
		case strings.Contains(p.Cause, "spies of"):
			return 2
		case strings.Contains(p.Cause, "poison"):
			return 1
		}
		return 0
	}
	sort.SliceStable(cands, func(i, j int) bool {
		if rank(cands[i]) != rank(cands[j]) {
			return rank(cands[i]) > rank(cands[j])
		}
		if cands[i].Renown != cands[j].Renown {
			return cands[i].Renown > cands[j].Renown
		}
		return cands[i].ID < cands[j].ID
	})
	for _, pp := range cands {
		st := w.stateByID[pp.State]
		if st == nil {
			continue
		}
		bd := w.burgByID[st.Capital]
		if bd == nil || !bd.Alive || bd.Layer != "surface" {
			continue
		}
		truth := ""
		poison := strings.Contains(pp.Cause, "poison")
		if k := strings.Index(pp.Cause, "spies of "); k >= 0 {
			truth = "It was the spies of " + pp.Cause[k+9:] + " who poisoned " + pp.Name + "."
		} else {
			for i := range w.Events {
				e := &w.Events[i]
				if e.Y < pp.Died-1 || e.Y > pp.Died {
					continue
				}
				if poison && e.Kind == "intrigue" && strings.Contains(e.Text, st.Name) || !poison && e.Kind == "coup" && strings.Contains(e.Text, st.Name) {
					truth = fmt.Sprintf("In the year %d: %s", e.Y, ehSentence(e.Text, 220))
					break
				}
			}
		}
		if truth == "" {
			continue
		}
		r := NewRng(g.Seed + ":es:unq:" + strconv.Itoa(pp.ID))
		goat := "a servant of the palace"
		if cu := w.cultByID[bd.Culture]; cu != nil && len(cu.Given) > 0 {
			goat = Pick(r, cu.Given)
			if len(cu.Houses) > 0 {
				goat += " " + Pick(r, cu.Houses)
			}
		}
		if poison {
			goat += ", the cupbearer"
		} else {
			goat += ", the gate-warden"
		}
		u := &ehUnq{dead: pp, burg: bd, goat: goat, truth: truth, zx: 1, zy: 1}
		order := [][2]int{{0, 0}, {2, 0}, {0, 2}, {2, 2}, {1, 0}, {0, 1}, {2, 1}, {1, 2}}
		found := false
		ol := w.outlawByBrg[bd.ID] != nil
		for _, z := range order {
			if k, _ := g.districtAt(bd.P%w.W, bd.P/w.W, z[0], z[1]); k == "" && !(ol && g.outlawZone(bd) == z) {
				u.zx, u.zy, found = z[0], z[1], true
				break
			}
		}
		if !found {
			for _, z := range order {
				if k, _ := g.districtAt(bd.P%w.W, bd.P/w.W, z[0], z[1]); k == "temple" || k == "homes" || k == "slums" {
					u.zx, u.zy = z[0], z[1]
					break
				}
			}
		}
		ehUnqCache[w] = u
		return u
	}
	return nil
}

func (c *ehCtx) unquiet() {
	u := c.g.ehUnquiet()
	w := c.g.W
	if u == nil || u.burg.ID != c.bd.ID || c.zn.WX != u.burg.P%w.W || c.zn.WY != u.burg.P/w.W || c.zn.ZX != u.zx || c.zn.ZY != u.zy {
		return
	}
	id := "unquiet:" + strconv.Itoa(u.dead.ID)
	if c.st.has(id) {
		return
	}
	x, y, ok := c.open(c.rng("unq", id), 2, nil)
	if !ok {
		return
	}
	c.zn.Set(x, y, TInscription)
	c.zn.Set(x+1, y, TDeadTree)
	if ehGround[c.zn.At(x+1, y+1)] {
		c.zn.Set(x+1, y+1, TBones)
	}
	act := "FOR THE POISONING OF"
	if !strings.Contains(u.dead.Cause, "poison") {
		act = "FOR OPENING THE GATE ON"
	}
	c.zn.Notes[y*MW+x] = fmt.Sprintf("Scratched into a stone under a dead tree, in a hand that did not write often:\n\n{{Y|%s}}\nHANGED IN THE YEAR %d %s %s.\nI DID NOT DO IT.\n\n{{K|The grass will not grow in a ring around the tree. Folk walk wide of it after dark, and say it is worst at night.}}",
		strings.ToUpper(u.goat), u.dead.Died, act, strings.ToUpper(u.dead.Name))
	sc := c.scene(id, "unquiet", x, y, x+1, y+1)
	c.n--
	sc.Read, sc.Night, sc.XP = c.tileKey(x, y), true, 60
	sc.Title = "Under the dead tree"
	sc.Reveal = fmt.Sprintf("The wind drops all at once. Under the tree something hangs that was not there a moment ago, turning slowly; then it is a figure standing on the grass with its head at the wrong angle. It does not speak. It looks toward the palace of %s, and you know what it knows:\n\n%s\n\n%s was hanged for it, and did not do it.\n\n{{K|When you look again there is only the tree, and the grass at your feet is wet.}}",
		u.burg.Name, u.truth, u.goat)
	sc.Fact = fmt.Sprintf("%s was hanged in the year %d for the death of %s, and was innocent. %s", u.goat, u.dead.Died, u.dead.Name, u.truth)
	sc.Secret = sc.Fact
}

// ---------------------------------------------------------------- life-simulation scenes (turn hook)

func ehPeople(g *Game, burg int) []*lvPerson {
	var out []*lvPerson
	for _, p := range lvSt(g).People {
		if p.Home == burg {
			out = append(out, p)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Key < out[j].Key })
	return out
}

func ehDynamic(g *Game, zn *Zone) {
	kind, bd := ehKind(g, zn)
	if bd == nil || !lvSt(g).Seeded[bd.ID] {
		return
	}
	st := ehSt(g)
	c := &ehCtx{g: g, zn: zn, bd: bd, kind: kind, st: st, dyn: true, max: 3}
	c.n = st.Dyn[zn.Key]
	if c.full() {
		return
	}
	people := ehPeople(g, bd.ID)
	switch kind {
	case "temple", "village":
		c.lvGraves(people)
	case "outskirts":
		c.lvOutcasts(people)
	}
	switch kind {
	case "homes", "slums", "market", "artisans", "forge", "harbor", "village":
		c.shopHouse(people)
		c.jailHouse(people)
		c.cradleHouse(people)
		c.hoardHouse(people)
		c.crimeGraffiti(people)
	}
	if c.n > 0 {
		g.computeLight(zn)
	}
}

func (c *ehCtx) lvGraves(people []*lvPerson) {
	ls := lsSt(c.g)
	put := 0
	for _, p := range people {
		if c.full() || put >= 2 {
			return
		}
		id := "grave:" + p.Key
		if p.Alive || p.Died == 0 || c.st.has(id) || strings.Contains(p.Cause, "gallows") || ehLonely(p) {
			continue
		}
		r := c.rng("grave", id)
		x, y, ok := c.open(r, 1, c.graveyardPred())
		if !ok {
			return
		}
		c.zn.Set(x, y, TInscription)
		child := p.Role == "child" || p.lvAge(p.Died) < 12
		text := c.lvEpitaph(p)
		if child {
			text = c.g.ehEpitaph(p.Name, "", "", c.town(), p.Born, p.Died, "", p.Culture, c.g.ehKinOf(p), "lv:"+p.Key)
			text = strings.Replace(text, "{{K|", "{{K|"+ehRep(ehPickH(ehFamily["child"], c.g.Seed, id), "[k]", c.g.ehKinOf(p))+" ", 1)
			c.dropAt(x, y+1, c.thing(ehPickH(ehToys, c.g.Seed, id+":toy"), "Left on "+p.Name+"'s grave. Taking it would be a poor thing to do.", '*', CBrown, 0))
		}
		c.zn.Notes[y*MW+x] = text
		sc := c.scene(id, "grave", x, y, x, y+1)
		sc.Read, sc.XP = c.tileKey(x, y), 5
		if strings.HasPrefix(p.Cause, "murdered by ") {
			sc.Fact = fmt.Sprintf("The gravestone of %s in %s says %s was %s.", p.Name, c.town(), ehFirst(p.Name), p.Cause)
			sc.Subj = ls.Avenge[p.Key]
		}
		put++
	}
}

// ehLonely: died "in an accident" within a year of losing a spouse, in deep grief. The town has its own view.
func ehLonely(p *lvPerson) bool {
	if p.Alive || p.Cause != "in an accident" || p.Mood > -40 || p.Role == "child" {
		return false
	}
	for _, e := range p.Life {
		if e.Kind == "death" && strings.HasPrefix(e.Text, "Lost ") && e.Y >= p.Died-1 {
			return true
		}
	}
	return false
}

func (c *ehCtx) lvOutcasts(people []*lvPerson) {
	ls := lsSt(c.g)
	for _, p := range people {
		if c.full() {
			return
		}
		if p.Alive || p.Died == 0 {
			continue
		}
		hanged := strings.Contains(p.Cause, "gallows")
		if !hanged && !ehLonely(p) {
			continue
		}
		id := "outcast:" + p.Key
		if c.st.has(id) {
			continue
		}
		x, y, ok := c.open(c.rng("outcast", id), 1, nil)
		if !ok {
			return
		}
		c.zn.Set(x, y, TInscription)
		sc := c.scene(id, "outcast", x, y, x, y)
		sc.Read, sc.XP = c.tileKey(x, y), 5
		if hanged {
			victim := ""
			for v, k := range ls.Avenge {
				if k == p.Key {
					victim = lvName(c.g, v)
				}
			}
			why := strings.TrimPrefix(p.Cause, "on the gallows ")
			c.zn.Notes[y*MW+x] = fmt.Sprintf("A stake driven into unconsecrated ground outside the wall. Cut into it: {{Y|%s}}, the year %d. %s\n\n{{K|%s}}",
				p.Name, p.Died, ehPickH(ehCauseLine["gallows"], c.g.Seed, id), c.g.ehFamilyLine("gallows", "", id))
			if victim != "" {
				sc.Fact = fmt.Sprintf("%s was hanged outside %s in the year %d %s: the death of %s.", p.Name, c.town(), p.Died, why, victim)
			} else {
				sc.Fact = fmt.Sprintf("%s was hanged outside %s in the year %d %s.", p.Name, c.town(), p.Died, why)
			}
		} else {
			c.zn.Notes[y*MW+x] = fmt.Sprintf("A low cairn outside the wall. No stone, no name cut, only a faded ribbon tied round the top stone.\n\n{{K|In town they say %s died in an accident in the year %d, the year after a death in the family. The priests buried the body out here all the same, and the family has never said a word against it.}}", p.Name, p.Died)
			sc.Secret = fmt.Sprintf("%s of %s lies outside the wall. The town does not believe it was an accident.", p.Name, c.town())
		}
	}
}

func (c *ehCtx) shopHouse(people []*lvPerson) {
	if c.full() {
		return
	}
	ls := lsSt(c.g)
	for _, j := range ls.Vacant[c.bd.ID] {
		var owner *lvPerson
		for _, p := range people {
			if !p.Alive && p.Job == j && (owner == nil || p.Died > owner.Died) {
				owner = p
			}
		}
		if owner == nil {
			continue
		}
		id := "shop:" + owner.Key
		if c.st.has(id) {
			continue
		}
		r := c.rng("shop", id)
		rm := c.room(r)
		if rm == nil {
			return
		}
		c.block(rm, r, TCrate, 2)
		var lines []string
		for i := len(owner.Life) - 1; i >= 0 && len(lines) < 3; i-- {
			if e := owner.Life[i]; e.Kind != "death" {
				lines = append([]string{fmt.Sprintf("The year %d: %s", e.Y, e.Text)}, lines...)
			}
		}
		book := c.note(fmt.Sprintf("Ledger of the %s, %s", j, owner.Name), fmt.Sprintf("Accounts in %s's hand, and in the margins a few lines of a life:\n\n%s\n\n{{K|After that, a different hand: \"Shut, the year %d. %s.\"}}",
			owner.Name, strings.Join(lines, "\n"), owner.Died, titleCase(strings.TrimSpace(owner.Cause))))
		c.drop(rm, r, book)
		if ks := c.keepsake(owner, ehPickH(ehKeeps, c.g.Seed, id)); ks != nil {
			c.drop(rm, r, ks)
		}
		sc := c.houseScene(id, "shop", rm)
		sc.Read, sc.XP = "book:"+book.Name, 10
		sc.Fact = fmt.Sprintf("The %s's shop in %s has stood shuttered since %s died in the year %d, %s.", j, c.town(), owner.Name, owner.Died, owner.Cause)
		return
	}
}

func (c *ehCtx) jailHouse(people []*lvPerson) {
	if c.full() {
		return
	}
	ls := lsSt(c.g)
	for _, p := range people {
		_, jailed := ls.Jail[p.Key]
		exiled := ls.Mig[p.Key] && ls.From[p.Key] == c.bd.ID
		if !p.Alive || !(jailed && ls.From[p.Key] == c.bd.ID) && !exiled {
			continue
		}
		id := "jail:" + p.Key
		if c.st.has(id) {
			continue
		}
		crime := ""
		for i := len(p.Life) - 1; i >= 0; i-- {
			if p.Life[i].Kind == "crime" {
				crime = p.Life[i].Text
				break
			}
		}
		if crime == "" {
			continue
		}
		r := c.rng("jail", id)
		rm := c.room(r)
		if rm == nil {
			return
		}
		c.scatter(rm, r, TRubble, 0.18)
		what := "held against the fine"
		if exiled {
			what = "forfeit: the owner is driven out of town"
		}
		book := c.note("Notice of the watch of "+c.town()+", "+p.Name, fmt.Sprintf("Nailed inside the door:\n\nBY ORDER OF THE WATCH OF %s. The goods of %s are %s.\n\n%s\n\n{{K|Someone has written under it in pencil: \"and who is feeding the cat?\"}}", strings.ToUpper(c.town()), p.Name, what, crime))
		c.drop(rm, r, book)
		sc := c.houseScene(id, "jail", rm)
		sc.Read, sc.XP, sc.Subj = "book:"+book.Name, 10, p.Key
		sc.Fact = fmt.Sprintf("%s of %s: %s", p.Name, c.town(), crime)
		if p.Secret != "" {
			sc.Secret = fmt.Sprintf("%s (searched house in %s): %s", p.Name, c.town(), p.Secret)
		}
		return
	}
}

func (c *ehCtx) cradleHouse(people []*lvPerson) {
	if c.full() {
		return
	}
	g := c.g
	for _, p := range people {
		if p.Alive || p.Died == 0 || !(p.Role == "child" || p.lvAge(p.Died) < 12) {
			continue
		}
		id := "cradle:" + p.Key
		if c.st.has(id) {
			continue
		}
		alive := false
		for _, q := range lsParents(g, p) {
			alive = alive || q.Alive
		}
		if !alive {
			continue
		}
		r := c.rng("cradle", id)
		rm := c.room(r)
		if rm == nil {
			return
		}
		for _, j := range rm.cells {
			if c.edge(rm, j) {
				c.zn.T[j] = TBed
				break
			}
		}
		mourn := "a plain black cloth"
		if cp := g.W.cpOf(c.bd.Culture); cp != nil && cp.Material != nil && cp.Material.Dress["mourning"] != "" {
			mourn = "a mourning cloth: " + strings.ToLower(strings.TrimSuffix(cp.Material.Dress["mourning"], "."))
		}
		c.drop(rm, r, c.thing("black mourning cloth", "Pinned above a small bed. "+titleCase(mourn)+".", '~', CDarkGray, 0))
		song := ""
		if cp := g.W.cpOf(c.bd.Culture); cp != nil && cp.Customs != nil {
			for _, rt := range cp.Customs.Rites {
				if (rt.ID == "birth" || rt.ID == "naming") && rt.Song != "" {
					song = "\n\n" + rt.Song
					break
				}
			}
		}
		book := c.note("Scrap pinned over a cradle in "+c.town(), fmt.Sprintf("%s. Born in the year %d. Gone in the year %d, %s.%s\n\n{{K|The ink has run in two places.}}", p.Name, p.Born, p.Died, strings.TrimSpace(p.Cause), song))
		c.drop(rm, r, book)
		if ks := c.keepsake(p, ehPickH(ehToys, g.Seed, id)); ks != nil {
			c.drop(rm, r, ks)
		}
		sc := c.houseScene(id, "cradle", rm)
		sc.Read, sc.XP = "book:"+book.Name, 5
		sc.Secret = fmt.Sprintf("%s, a child of %s, died in the year %d, %s.", p.Name, c.town(), p.Died, p.Cause)
		return
	}
}

func (c *ehCtx) hoardHouse(people []*lvPerson) {
	if c.full() {
		return
	}
	id := "hoard:" + strconv.Itoa(c.bd.ID)
	if c.st.has(id) {
		return
	}
	g := c.g
	yr := lvYear(g)
	for _, p := range people {
		if !p.Alive || p.Trait[1] > 25 || p.Trait[4] < 65 || p.lvAge(yr) < 30 {
			continue
		}
		r := c.rng("hoard", id)
		rm := c.room(r)
		if rm == nil {
			return
		}
		c.block(rm, r, TCrate, 6)
		c.block(rm, r, TBarrel, 1)
		for _, n := range []string{"bundle of bent nails", "cracked jug", "string of old keys", "stack of rotten almanacs"} {
			if r.Chance(0.6) {
				c.drop(rm, r, &Item{ID: g.newID(), Kind: "scrap", Name: n, Glyph: '%', F: CBrown, Value: 1, Weight: 1})
			}
		}
		c.drop(rm, r, g.Loot(r, 1))
		fear := ""
		if p.Fear != "" {
			fear = fmt.Sprintf("\n\nIn the margin, pressed hard: \"%s\"", p.Fear)
		}
		book := c.note(p.Name+"'s tally", fmt.Sprintf("A list, kept for years, in %s's hand. NOT TO BE THROWN AWAY: the keys (all), the jugs (even cracked), the almanacs, the nails, the string, the good sack and the other good sack.%s", p.Name, fear))
		c.drop(rm, r, book)
		sc := c.houseScene(id, "hoard", rm)
		sc.Read, sc.XP, sc.Subj = "book:"+book.Name, 10, p.Key
		sc.Fact = fmt.Sprintf("%s of %s keeps a house full of things nobody may touch.", p.Name, c.town())
		if p.Secret != "" {
			sc.Secret = fmt.Sprintf("%s of %s: %s", p.Name, c.town(), p.Secret)
		}
		return
	}
}

func (c *ehCtx) crimeGraffiti(people []*lvPerson) {
	if c.full() {
		return
	}
	ls := lsSt(c.g)
	for _, p := range people {
		if _, at := ls.Crim[p.Key]; !at || !p.Alive {
			continue
		}
		id := "grafc:" + p.Key
		if c.st.has(id) {
			continue
		}
		x, y, ok := c.wallSpot(c.rng("grafc", id), 1)
		if !ok {
			return
		}
		c.zn.Set(x, y, TInscription)
		c.zn.Notes[y*MW+x] = "Scratched into the wall, fresh: {{W|" + ehRep(ehPickH(ehGraffiti["crime"], c.g.Seed, id), "[n]", strings.ToUpper(p.Name)) + "}}"
		sc := c.scene(id, "graffiti", x, y, x, y)
		sc.Read, sc.XP, sc.Subj = c.tileKey(x, y), 5, p.Key
		sc.Fact = fmt.Sprintf("Someone in %s has scratched on a wall that %s is a thief who walks free.", c.town(), p.Name)
		return
	}
}

// ---------------------------------------------------------------- turn hook: looks and payoffs

func ehTurn(g *Game) {
	if g.W == nil || g.Cur == nil || g.P == nil {
		return
	}
	st := ehSt(g)
	zn := g.Cur
	if st.last != zn.Key {
		st.last = zn.Key
		ehDynamic(g, zn)
		ehBoardRefresh(g, zn, true)
	} else if g.Turn%60 == 0 {
		ehBoardRefresh(g, zn, false)
	}
	if len(st.Order) == 0 {
		return
	}
	h := g.Hour()
	night := h >= 20 || h < 5
	px, py := g.P.X, g.P.Y
	for _, id := range st.Order {
		sc := st.Scenes[id]
		if sc == nil {
			continue
		}
		here := sc.Zone == zn.Key
		if here && !sc.Seen && sc.Look != "" && px >= sc.X0 && px <= sc.X1 && py >= sc.Y0 && py <= sc.Y1 {
			sc.Seen = true
			g.Msg("{{K|" + sc.Look + "}}")
		}
		if sc.Paid || sc.Read == "" || !g.Read[sc.Read] {
			continue
		}
		if sc.Night && (!here || !night || g.Mode != "main" || maxi(abs(px-sc.X0), abs(py-sc.Y0)) > 3) {
			continue
		}
		ehPay(g, sc)
	}
}

func ehPay(g *Game, sc *ehScene) {
	sc.Paid = true
	if sc.Fact != "" {
		ehFact(g, sc)
	}
	if sc.Secret != "" {
		g.addSecret(sc.Secret, "trace", sc.SecP-1, lvYear(g))
	}
	for _, p := range sc.Marks {
		if g.Known != nil {
			g.Known[p] = true
		}
	}
	if len(sc.Marks) > 0 {
		g.Msg("{{c|You mark the places the lookout stone points to on your map.}}")
	}
	if sc.XP > 0 {
		g.gainXP(sc.XP)
	}
	if sc.Reveal != "" {
		g.showPopup(nonEmpty(sc.Title, "A truth"), sc.Reveal)
		g.deed("lore", fmt.Sprintf("The wanderer %s learned the truth of an old hanging: %s", g.P.Name, sc.Fact), 0, 0)
	} else if sc.Fact != "" {
		g.Msg("{{c|You copy what it says into your case notes.}}")
	}
}

func ehFact(g *Game, sc *ehScene) {
	st := knSt(g)
	f := st.add(knFact{Kind: "event", Subject: fmt.Sprintf("burg:%d", sc.Burg), Text: strings.TrimSuffix(Plain(sc.Fact), "."), True: true, Day: lvDay(g), Burg: sc.Burg, Ver: 1})
	if sc.Subj != "" && lvGet(g, sc.Subj) != nil {
		f.Subject = sc.Subj
		for _, cs := range st.Cases {
			if cs.Status == "open" && cs.Culprit == sc.Subj {
				f.Kind, f.Case, f.Culprit, f.Lvl, f.Ver = "crime", cs.ID, sc.Subj, 1, 0
				cs.FactIDs = append(cs.FactIDs, f.ID)
				break
			}
		}
	}
	g.knLearn(f, &lvPerson{Key: "es:trace", Name: "what the town left written"}, 2, "read")
}

// ---------------------------------------------------------------- keepsakes: the errand

func ehTalk(g *Game, a *Actor) []Option {
	st := ehSt(g)
	if len(st.Keep) == 0 || g.P == nil {
		return nil
	}
	p := lvPersonOf(g, a)
	if p == nil || !p.Alive {
		return nil
	}
	var opts []Option
	for _, it := range g.P.Inv {
		if it.Extra == nil || it.Extra["eh:ks"] == 0 {
			continue
		}
		k := st.Keep[it.ID]
		if k == nil || k.Done || !lsHas(k.Kin, p.Key) {
			continue
		}
		it, k := it, k
		opts = append(opts, Option{"Hand over " + it.Name + ".", func() { ehReturn(g, a, p, it, k) }})
	}
	return opts
}

func ehReturn(g *Game, a *Actor, p *lvPerson, it *Item, k *ehKeep) {
	g.removeInv(it)
	k.Done = true
	g.lvMemoryAdd(p, "gift", "brought back "+k.Name, 14)
	g.gainXP(25)
	thanks := "Thank you."
	if cp := g.W.cpOf(p.Culture); cp != nil && cp.Voice != nil && len(cp.Voice.Thanks) > 0 {
		thanks = cpFill(g.W.cpChoose(cp.Voice.Thanks, "eh:thanks", p.Key), map[string]string{"name": g.P.Name})
	}
	what := strings.TrimPrefix(it.Name, ehFirst(k.Dead)+"'s ")
	text := fmt.Sprintf("%s turns the %s over and over. \"%s I thought it was lost. %s kept it close, always.\"", p.Name, what, thanks, ehFirst(k.Dead))
	if d := lvGet(g, k.DeadKey); d != nil {
		for i := len(d.Life) - 1; i >= 0; i-- {
			if e := d.Life[i]; e.Kind != "death" && e.Kind != "birth" {
				text += fmt.Sprintf("\n\nThey tell you about %s, and about the year %d: %s", ehFirst(k.Dead), e.Y, strings.TrimSuffix(lvLower(e.Text), "."))
				text += "."
				break
			}
		}
		if d.Secret != "" {
			text += "\n\nThen, quieter: \"" + d.Secret + "\""
			g.addSecret(fmt.Sprintf("%s (told by %s): %s", d.Name, p.Name, d.Secret), "trace", -1, lvYear(g))
		}
	}
	g.talkMenu(text + "\n\n{{G|They will remember this.}}")
}
