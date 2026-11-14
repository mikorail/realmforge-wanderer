package engine

import (
	"fmt"
	"strings"
)

// pets.go (prefix pq): pets and play modes.
//
// Pets are Actors with Tpl "pqpet", flag `pqpet` and AI "follower" (so the stock follower AI, zone
// crossing and bump-swap just work) but with NO Companion record and no Order: they never touch
// taming.go / companions.go state. A stray carries flag `pqstray` + `pqsp_<species>`, faction "pqstray".
// Everything else (needs, tricks, memorials) lives in g.ext("pq") (global) and g.ext("pql") (planet-local).
//
// Play modes (pqModes in pets_data.go): pqApply(g, ids) at start, toggles on the "pq" screen.
// Nothing here draws g.R during generation; side hashes of g.Seed only.

type pqPet struct {
	ID      int            `json:"id"`
	Name    string         `json:"n"`
	Sp      string         `json:"sp"`
	Food    int            `json:"fd"`
	Aff     int            `json:"af"`
	XP      int            `json:"xp,omitempty"`
	Prac    map[string]int `json:"pr,omitempty"`
	Born    int            `json:"b"`
	Kills   int            `json:"k,omitempty"`
	Zone    string         `json:"z,omitempty"`
	Young   bool           `json:"y,omitempty"`
	Hurt    bool           `json:"h,omitempty"`
	PracAt  int            `json:"pa,omitempty"`
	PlayAt  int            `json:"pl,omitempty"`
	FetchAt int            `json:"fa,omitempty"`
	FetchID int            `json:"fi,omitempty"`
	FetchTo int            `json:"ft,omitempty"`
	StealAt int            `json:"sa,omitempty"`
	HerbAt  int            `json:"ha,omitempty"`
	AlertAt int            `json:"aa,omitempty"`
	WarmAt  int            `json:"wa,omitempty"`
	Carry   *Item          `json:"c,omitempty"`
}

type pqMemorial struct {
	Name  string `json:"n"`
	Sp    string `json:"sp"`
	Days  int    `json:"d"`
	Place string `json:"p"`
	Cause string `json:"c"`
	Day   int    `json:"dy"`
}

type pqState struct {
	Pets  []*pqPet        `json:"pets,omitempty"`
	Mem   []pqMemorial    `json:"mem,omitempty"`
	Modes map[string]bool `json:"modes,omitempty"`
	Calm  map[int]string  `json:"calm,omitempty"` // actor id -> faction before peaceful mode
	Revs  int             `json:"revs,omitempty"`

	// screen state (not saved)
	Tab    int            `json:"-"`
	Sel    int            `json:"-"`
	Sub    string         `json:"-"` // "", "trick", "name", "rename", "release"
	Buf    string         `json:"-"`
	Edit   int            `json:"-"` // pet id being named
	Note   string         `json:"-"`
	Warned map[int]int    `json:"-"`
	Seen   map[string]int `json:"-"`
}

// pqLocalState is planet-local (zone keys): last inn for story mode, claimed litters, bought offers.
type pqLocalState struct {
	InnKey  string         `json:"ik,omitempty"`
	InnX    int            `json:"ix,omitempty"`
	InnY    int            `json:"iy,omitempty"`
	InnName string         `json:"in,omitempty"`
	Litter  map[string]int `json:"lit,omitempty"`
	Sold    map[string]int `json:"sold,omitempty"`
}

func (g *Game) pq() *pqState { return g.ext("pq", func() any { return &pqState{} }).(*pqState) }
func (g *Game) pql() *pqLocalState {
	return g.ext("pql", func() any { return &pqLocalState{} }).(*pqLocalState)
}

func (st *pqState) rec(id int) *pqPet {
	for _, p := range st.Pets {
		if p.ID == id {
			return p
		}
	}
	return nil
}

func (st *pqState) drop(id int) {
	for i, p := range st.Pets {
		if p.ID == id {
			st.Pets = append(st.Pets[:i], st.Pets[i+1:]...)
			return
		}
	}
}

func init() {
	unLocalKeys = append(unLocalKeys, "pql")
	zoneHooks = append(zoneHooks, pqZone)
	talkHooks = append(talkHooks, pqTalk)
	turnHooks = append(turnHooks, pqTurn)
	killHooks = append(killHooks, pqKill)
	enterHooks = append(enterHooks, pqEnter)
	modeKeys["pq"] = pqKeys
	modeDraws["pq"] = func(g *Game) { g.pqDraw() }
	paEntries = append(paEntries,
		paEntry{Name: "Pets", Keywords: "pets pet animal cat dog hawk companion feed trick", Cat: "Life", Label: "ability", Func: func(g *Game) { g.pqOpen(0) }},
		paEntry{Name: "Play modes", Keywords: "play modes peaceful story ironman cozy hunger difficulty", Cat: "Life", Label: "ability", Func: func(g *Game) { g.pqOpen(1) }})
}

// ---- small helpers ----

func (g *Game) pqHash(parts ...any) uint64 {
	return HashStr(g.Seed + ":pq:" + fmt.Sprint(parts...))
}

func pqDir(dx, dy int) string {
	ns, ew := "", ""
	if dy < -1 {
		ns = "north"
	} else if dy > 1 {
		ns = "south"
	}
	if dx > 1 {
		ew = "east"
	} else if dx < -1 {
		ew = "west"
	}
	if ns == "" && ew == "" {
		return "close by"
	}
	return "to the " + strings.Trim(ns+"-"+ew, "-")
}

func pqBar(v int) string {
	n := clampi(v/10, 0, 10)
	col := "G"
	if v < 25 {
		col = "R"
	} else if v < 50 {
		col = "Y"
	}
	return "{{" + col + "|" + strings.Repeat("#", n) + "}}{{K|" + strings.Repeat("-", 10-n) + "}}"
}

func (g *Game) pqMaxPets() int {
	n := 2
	if g.Skills["persuasion"] >= 2 {
		n = 3
	}
	return n
}

func (g *Game) pqNoun(sp *pqSp) string { return sp.noun(g.Genre) }

func pqSpOf(a *Actor) *pqSp {
	for i := range pqSpecies {
		if a.Has("pqsp_" + pqSpecies[i].ID) {
			return &pqSpecies[i]
		}
	}
	return nil
}

// pqFind locates a pet's actor anywhere on this planet.
func (g *Game) pqFind(id int) (*Actor, *Zone) {
	if g.Cur != nil {
		for _, a := range g.Cur.Actors {
			if a.ID == id && a.HP > 0 {
				return a, g.Cur
			}
		}
	}
	for _, z := range g.Zones {
		if z == g.Cur {
			continue
		}
		for _, a := range z.Actors {
			if a.ID == id && a.HP > 0 {
				return a, z
			}
		}
	}
	return nil, nil
}

type pqLP struct {
	A  *Actor
	Pr *pqPet
	Sp *pqSp
}

// pqLocal lists the player's pets in the current zone.
func (g *Game) pqLocal() []pqLP {
	var out []pqLP
	if g.Cur == nil {
		return out
	}
	st := g.pq()
	for _, a := range g.Cur.Actors {
		if a.HP <= 0 || !a.Has("pqpet") || a.AI != "follower" {
			continue
		}
		if pr := st.rec(a.ID); pr != nil {
			if sp := pqSpByID[pr.Sp]; sp != nil {
				out = append(out, pqLP{a, pr, sp})
			}
		}
	}
	return out
}

func (g *Game) pqMood(pr *pqPet, a *Actor) string {
	hpr := 100
	if a != nil && a.MaxHP > 0 {
		hpr = a.HP * 100 / a.MaxHP
	}
	switch {
	case pr.Food <= 5:
		return "starving"
	case hpr < 35:
		return "frightened"
	case pr.Food < 25:
		return "hungry"
	case pr.Aff < 20:
		return "lonely"
	case pr.Aff >= 80 && pr.Food >= 50:
		return "devoted"
	case pr.Aff >= 55 && pr.Food >= 50:
		return "playful"
	}
	return "content"
}

// pqEff scales a pet's perks and tricks by its mood.
func pqEff(mood string) float64 {
	switch mood {
	case "starving", "frightened":
		return 0
	case "hungry", "lonely":
		return 0.5
	}
	return 1
}

func (pr *pqPet) lv() int { return clampi(1+pr.XP/25, 1, 10) }

func (pr *pqPet) trick(id string) int {
	if pr.Prac == nil {
		return 0
	}
	return pqTrickLv(pr.Prac[id])
}

func (g *Game) pqLive(pr *pqPet) int { return maxi(0, (g.Turn-pr.Born)/DayLen) }

// pqSync recomputes a pet actor's stats from its record.
func (g *Game) pqSync(a *Actor, pr *pqPet) {
	sp := pqSpByID[pr.Sp]
	if sp == nil {
		return
	}
	lv := pr.lv()
	hp := sp.HP + (lv-1)*3 + pr.trick("guard")*3
	if pr.Young {
		hp = hp * 2 / 3
	}
	hp = maxi(hp, 3)
	bonus := a.MaxHP == 0
	a.MaxHP = hp
	if bonus {
		a.HP = hp
	}
	a.HP = mini(a.HP, a.MaxHP)
	a.Level = lv
	a.DVNat = 4 + lv/2
	a.AVNat = lv / 4
	for _, p := range a.Body {
		if p.Natural != nil {
			p.Natural.DN, p.Natural.DS, p.Natural.PV = sp.Bite[0], sp.Bite[1]+lv/3, sp.Bite[2]
			p.Natural.DB = pr.trick("guard")
			if pr.Young {
				p.Natural.DS = maxi(2, p.Natural.DS-1)
			}
		}
	}
}

// ---- creating actors ----

func (g *Game) pqMakeActor(sp *pqSp, young bool, x, y int) *Actor {
	a := &Actor{ID: g.newID(), Name: g.pqNoun(sp), Glyph: sp.Glyph, F: sp.Col, X: x, Y: y, Tpl: "pqpet", Faction: "pqstray", Level: 1,
		Attr: [6]int{8, 14, 10, 6 + sp.Smart*2, 10, 8}, Speed: sp.Spd, AI: "wander", Desc: sp.Desc, Body: MakeBody(sp.Body), HomeX: x, HomeY: y}
	a.Set("pqpet")
	a.Set("pqstray")
	a.Set("pqsp_" + sp.ID)
	if sp.Fly {
		a.Set("fly")
		a.Set("flying")
	}
	if young {
		a.Set("pqyoung")
	}
	pr := &pqPet{Young: young}
	pr.Sp = sp.ID
	g.pqSync(a, pr)
	a.HP = a.MaxHP
	return a
}

func pqStrayPool(g *Game) []*pqSp {
	var out []*pqSp
	for i := range pqSpecies {
		s := &pqSpecies[i]
		if s.Wt > 0 && s.genreOK(g.Genre) {
			out = append(out, s)
		}
	}
	return out
}

func (g *Game) pqPickSp(h uint64, pool []*pqSp) *pqSp {
	tot := 0
	for _, s := range pool {
		tot += s.Wt
	}
	if tot == 0 {
		return nil
	}
	n := int(h % uint64(tot))
	for _, s := range pool {
		if n < s.Wt {
			return s
		}
		n -= s.Wt
	}
	return pool[0]
}

// pqZone: strays in the open and at the edges of towns, an animal seller in some towns.
func pqZone(g *Game, zn *Zone, _ *Rng) {
	if g.W == nil || zn.Z != 0 || zn.Sky || zn.Underwater || zn.Dark {
		return
	}
	p := 0.0
	switch zn.Kind {
	case "town", "outskirts":
		p = 0.14
	case "surface", "camp":
		p = 0.05
	default:
		return
	}
	r := NewRng(g.Seed + ":pq:" + zn.Key)
	if r.Chance(p) {
		if sp := g.pqPickSp(uint64(r.Int(0, 1<<30)), pqStrayPool(g)); sp != nil {
			if x, y, ok := zn.FreeSpot(r, nil); ok {
				a := g.pqMakeActor(sp, r.Chance(0.25), x, y)
				zn.Actors = append(zn.Actors, a)
			}
		}
	}
	if zn.Kind == "town" && zn.Burg > 0 && r.Chance(0.2) {
		bd := g.W.burgByID[zn.Burg]
		if bd == nil || g.W.cultByID[bd.Culture] == nil {
			return
		}
		x, y, ok := zn.FreeSpot(r, func(x, y int) bool {
			t := zn.At(x, y)
			return (t == TCobble || t == TRoad || t == TDirt || t == TGrass) && x > 2 && x < MW-3 && y > 1 && y < MH-2
		})
		if !ok {
			return
		}
		a := g.makeNPC("pqseller", bd, g.W.cultByID[bd.Culture], r)
		a.Title = "animal seller"
		a.Desc = fmt.Sprintf("A seller of pets and beasts in %s, with a few cages and a basket.", bd.Name)
		a.F = CGreen
		a.X, a.Y, a.HomeX, a.HomeY = x, y, x, y
		zn.Actors = append(zn.Actors, a)
	}
}

// ---- adopting ----

func (g *Game) pqCount() int { return len(g.pq().Pets) }

func (g *Game) pqAdopt(a *Actor, sp *pqSp, young bool) *pqPet {
	st := g.pq()
	pr := &pqPet{ID: a.ID, Sp: sp.ID, Food: 70, Aff: 30, Born: g.Turn, Young: young, Prac: map[string]int{}, Zone: g.Cur.Key}
	pr.Name = g.pqSuggest(sp, a.ID)
	st.Pets = append(st.Pets, pr)
	if a.Flags != nil {
		delete(a.Flags, "pqstray")
	}
	a.AI, a.Faction, a.Proper, a.F = "follower", "player", true, sp.Col
	a.Name = pr.Name
	a.Title = g.pqNoun(sp)
	g.pqSync(a, pr)
	g.Msg(fmt.Sprintf("{{G|A %s joins you.}} {{K|(pets screen: the Pets ability or palette)}}", g.pqNoun(sp)))
	g.deed("event", fmt.Sprintf("%s took in a %s.", g.P.Name, g.pqNoun(sp)), 0, 0)
	st.Sub, st.Tab, st.Edit, st.Buf = "name", 0, a.ID, ""
	g.Talk = nil
	g.Mode = "pq"
	return pr
}

func (g *Game) pqSuggest(sp *pqSp, id int) string {
	used := map[string]bool{}
	for _, p := range g.pq().Pets {
		used[p.Name] = true
	}
	n := len(sp.Names)
	h := int(g.pqHash("name", id) % uint64(n))
	for i := 0; i < n; i++ {
		if c := sp.Names[(h+i)%n]; !used[c] {
			return c
		}
	}
	return sp.Names[h]
}

func (g *Game) pqFoodItem(sp *pqSp) *Item {
	var best *Item
	bn := 0
	for _, it := range g.P.Inv {
		if n := pqNourish(sp, it); n > 0 && (best == nil || n < bn) {
			best, bn = it, n
		}
	}
	return best
}

func pqNourish(sp *pqSp, it *Item) int {
	if it == nil {
		return 0
	}
	if sp.Mech {
		if it.Kind == "scrap" {
			return 35
		}
		return 0
	}
	carn := sp.ID == "cat" || sp.ID == "hawk" || sp.ID == "ferret"
	switch it.Kind {
	case "food":
		return 40
	case "corpse":
		return 50
	case "egg":
		return 25
	case "ingredient":
		if carn {
			return 15
		}
		return 25
	}
	return 0
}

func (g *Game) pqUseItem(it *Item) {
	if it.Count > 1 {
		it.Count--
		return
	}
	removeItem(&g.P.Inv, it)
}

// pqStrayOpts replaces the talk menu of a stray animal.
func (g *Game) pqStrayOpts(a *Actor) []Option {
	sp := pqSpOf(a)
	if sp == nil {
		return nil
	}
	g.Opts = nil
	young := a.Has("pqyoung")
	desc := sp.Desc
	if young {
		desc = "A young one, thin and wary. " + desc
	}
	g.Popup = Wrap("A stray "+g.pqNoun(sp)+". "+desc, 100)
	var opts []Option
	if g.pqCount() >= g.pqMaxPets() {
		g.Popup = append(g.Popup, "", "{{y|You cannot look after more animals than you have.}}")
		return opts
	}
	if it := g.pqFoodItem(sp); it != nil {
		opts = append(opts, Option{"{{G|Offer it " + strings.ToLower(it.Name) + ".}}", func() {
			g.pqUseItem(it)
			g.Msg("It eats, watching you, and does not run.")
			g.pqAdopt(a, sp, young)
		}})
	}
	opts = append(opts, Option{g.checkLabel("survival", 9) + " Win it over with patience.", func() {
		g.spendTurns(5)
		if g.roll2d6Check(g.checkMod("survival"), 9) {
			g.Msg("After a while it comes close and sniffs your hand.")
			g.pqAdopt(a, sp, young)
			return
		}
		g.Msg("It bolts a few paces and watches you from there.")
		g.wander(a)
		g.wander(a)
		g.Mode, g.Talk = "main", nil
	}})
	return opts
}

func pqSellerNames(r *Rng) string {
	return Pick(r, []string{"Mind the biting one.", "All fed this morning.", "Healthy and clean, every one.", "They choose you, not the other way."})
}

// pqOffers: what a seller (or this town this season) has for sale.
func (g *Game) pqOffers(a *Actor) []*pqSp {
	season := g.Day() / 28
	var pool []*pqSp
	for i := range pqSpecies {
		if pqSpecies[i].Price > 0 && pqSpecies[i].genreOK(g.Genre) {
			pool = append(pool, &pqSpecies[i])
		}
	}
	var out []*pqSp
	seen := map[string]bool{}
	for i := 0; len(out) < 3 && i < 12; i++ {
		s := pool[int(g.pqHash("offer", g.Cur.Key, a.ID, season, i)%uint64(len(pool)))]
		if !seen[s.ID] {
			seen[s.ID] = true
			out = append(out, s)
		}
	}
	return out
}

func (g *Game) pqSellerMenu(a *Actor, text string) {
	if g.Mode != "talk" {
		return
	}
	g.Popup = Wrap(text, 100)
	g.Opts = nil
	season := g.Day() / 28
	l := g.pql()
	for _, sp := range g.pqOffers(a) {
		sp := sp
		key := fmt.Sprintf("%s:%s:%d", g.Cur.Key, sp.ID, season)
		if l.Sold[key] > 0 {
			continue
		}
		g.Opts = append(g.Opts, Option{fmt.Sprintf("{{W|Buy the %s.}} {{K|(%s)}}", g.pqNoun(sp), g.cuFormat(sp.Price)), func() {
			if g.pqCount() >= g.pqMaxPets() {
				g.pqSellerMenu(a, "\"You have your hands full already.\"")
				return
			}
			if !g.cuSpend(sp.Price) {
				g.pqSellerMenu(a, "\"That is not enough for this one.\"")
				return
			}
			if l.Sold == nil {
				l.Sold = map[string]int{}
			}
			l.Sold[key] = 1
			g.pqSpawnNear(sp, false)
		}})
	}
	g.Opts = append(g.Opts, Option{"Never mind.", func() { g.talkMenu(g.greeting(a)) }})
	g.Sel = 0
}

func (g *Game) pqSpawnNear(sp *pqSp, young bool) {
	x, y, _ := g.nearestFree(g.Cur, g.P.X+1, g.P.Y)
	a := g.pqMakeActor(sp, young, x, y)
	g.Cur.Actors = append(g.Cur.Actors, a)
	g.nudge(g.Cur, a)
	g.pqAdopt(a, sp, young)
}

func (g *Game) pqLitterFor(a *Actor) *pqSp {
	if g.Cur == nil || g.Cur.Kind == "" {
		return nil
	}
	season := g.Day() / 28
	if g.pql().Litter[g.Cur.Key] > season {
		return nil
	}
	if g.pqHash("litter", g.Cur.Key, season)%100 >= 30 {
		return nil
	}
	var pool []*pqSp
	for _, id := range []string{"cat", "dog", "biopet"} {
		if s := pqSpByID[id]; s.genreOK(g.Genre) {
			pool = append(pool, s)
		}
	}
	return pool[int(g.pqHash("litsp", g.Cur.Key, season)%uint64(len(pool)))]
}

func pqLitterHost(a *Actor) bool {
	switch a.Role {
	case "innkeeper", "tavern", "lw_innkeeper", "lw_farmer":
		return true
	}
	return strings.Contains(strings.ToLower(a.Title), "farmer")
}

func pqTalk(g *Game, a *Actor) []Option {
	if a == nil || g.W == nil {
		return nil
	}
	if a.Has("pqstray") {
		return g.pqStrayOpts(a)
	}
	var out []Option
	if a.Role == "pqseller" {
		out = append(out, Option{"{{W|Ask what animals you have.}}", func() {
			g.pqSellerMenu(a, "\""+pqSellerNames(NewRng(fmt.Sprint(a.ID, g.Day())))+"\" The animals on offer:")
		}})
		return out
	}
	if pqLitterHost(a) && g.Cur != nil && g.Mode == "talk" && !g.Hostile(a) {
		if sp := g.pqLitterFor(a); sp != nil && g.pqCount() < g.pqMaxPets() {
			out = append(out, Option{"{{W|Ask whether any young animals need a home.}}", func() {
				l := g.pql()
				if l.Litter == nil {
					l.Litter = map[string]int{}
				}
				l.Litter[g.Cur.Key] = g.Day()/28 + 1
				g.Msg("\"A litter came, and the last of it needs someone. Take one.\"")
				g.pqSpawnNear(sp, true)
			}})
		}
	}
	return out
}

// ---- release, death, memorials ----

func (g *Game) pqRelease(pr *pqPet, a *Actor, cause string) {
	st := g.pq()
	sp := pqSpByID[pr.Sp]
	if a != nil && sp != nil {
		a.AI, a.Faction, a.Proper, a.Name = "wander", "pqstray", false, g.pqNoun(sp)
		a.Set("pqstray")
		a.Title = ""
	}
	st.drop(pr.ID)
	g.Msg(fmt.Sprintf("{{y|%s %s.}}", pr.Name, cause))
}

func pqKill(g *Game, d, src *Actor) {
	if d == nil || !d.Has("pqpet") || d.Has("pqstray") {
		return
	}
	st := g.pq()
	pr := st.rec(d.ID)
	if pr == nil {
		return
	}
	// no corpse, a memorial instead
	for i := len(g.Cur.Items) - 1; i >= 0; i-- {
		it := g.Cur.Items[i]
		if it.Kind == "corpse" && it.X == d.X && it.Y == d.Y && it.Name == d.Name+" corpse" {
			g.Cur.Items = append(g.Cur.Items[:i], g.Cur.Items[i+1:]...)
		}
	}
	cause := "wounds"
	if src != nil {
		cause = "a " + strings.ToLower(src.Name)
		if src.Proper {
			cause = src.Name
		}
	}
	days := g.pqLive(pr)
	st.Mem = append(st.Mem, pqMemorial{Name: pr.Name, Sp: pr.Sp, Days: days, Place: g.Cur.Name, Cause: cause, Day: g.Day()})
	st.drop(pr.ID)
	sp := pqSpByID[pr.Sp]
	noun := "companion"
	if sp != nil {
		noun = g.pqNoun(sp)
	}
	epitaph := fmt.Sprintf("Here lies %s, a faithful %s of %s, lost to %s after %d days.", pr.Name, noun, g.P.Name, cause, days)
	z := g.Cur
	t := z.At(d.X, d.Y)
	if z.Kind != "town" && (t == TGrass || t == TDirt || t == TTallGrass || t == TSand || t == TMoss || t == TSnow || t == TFloor) {
		z.Set(d.X, d.Y, TInscription)
		if z.Notes == nil {
			z.Notes = map[int]string{}
		}
		z.Notes[d.Y*MW+d.X] = epitaph
	} else {
		z.Items = append(z.Items, &Item{ID: g.newID(), Kind: "trophy", Name: "keepsake of " + pr.Name, Glyph: '†', F: CGray, X: d.X, Y: d.Y, Lore: epitaph})
	}
	g.Msg(fmt.Sprintf("{{R|%s is gone.}} {{K|You lay a marker where it fell.}}", pr.Name))
	g.deed("event", epitaph, 0, 0)
	if g.Mode == "pq" {
		g.Mode = "main"
	}
}

// ---- perks (hook functions for other systems) ----

// pqCheckBonus: skill-check bonus from pets nearby (perception from keen eyes and ears, survival from a nose,
// empathy and persuasion from a fond pet). Hook: checkMod.
func (g *Game) pqCheckBonus(kind string) int {
	if g.P == nil || g.Cur == nil {
		return 0
	}
	n := 0
	for _, lp := range g.pqLocal() {
		if dist(lp.A, g.P) > 8 {
			continue
		}
		eff := pqEff(g.pqMood(lp.Pr, lp.A))
		if eff < 1 {
			continue
		}
		switch kind {
		case "perception":
			if lp.Sp.has(lp.Sp.Perks, "eyes") || lp.Sp.has(lp.Sp.Perks, "alarm") {
				n++
			}
		case "survival":
			if lp.Sp.has(lp.Sp.Perks, "nose") {
				n++
			}
		case "empathy", "persuasion":
			if lp.Sp.has(lp.Sp.Perks, "charm") && lp.Pr.Aff >= 60 {
				n++
			}
		}
	}
	return mini(n, 3)
}

// pqWarmth: true when a pet that knows "warm" is curled against you (for weather/cold hooks).
func (g *Game) pqWarmth() bool {
	for _, lp := range g.pqLocal() {
		if lp.Pr.trick("warm") > 0 && dist(lp.A, g.P) <= 1 && pqEff(g.pqMood(lp.Pr, lp.A)) > 0 {
			return true
		}
	}
	return false
}

// pqNoiseRange: how far a pet will sense and growl at unseen hostiles (0 = no alarm pet). Hook: senses.go.
func (g *Game) pqNoiseRange() int {
	best := 0
	for _, lp := range g.pqLocal() {
		r := 0
		if lp.Sp.has(lp.Sp.Perks, "alarm") {
			r = 9
		} else if lp.Sp.has(lp.Sp.Perks, "eyes") {
			r = 7
		}
		if t := lp.Pr.trick("warn"); t > 0 {
			r += 3 * t
		}
		if r > best && pqEff(g.pqMood(lp.Pr, lp.A)) > 0 {
			best = r
		}
	}
	return best
}

var pqVerbs = map[string]string{"cat": "hisses", "dog": "growls", "hawk": "screes", "ferret": "chitters", "monkey": "screeches", "lizard": "hisses", "mouse": "squeaks",
	"robodog": "whirs and growls", "drone": "beeps urgently", "imp": "snickers uneasily", "biopet": "hums sharply"}

// ---- turn ----

func pqOutdoors(z *Zone) bool {
	return z.Z == 0 && !z.Dark && !z.Underwater && (z.Kind == "surface" || z.Kind == "camp" || z.Kind == "outskirts")
}

func pqTurn(g *Game) {
	if g.P == nil || g.Cur == nil || g.W == nil {
		return
	}
	st := g.pq()
	pqModesTurn(g, st)
	if len(st.Pets) == 0 {
		return
	}
	if st.Warned == nil {
		st.Warned = map[int]int{}
	}
	z := g.Cur
	hour := g.Turn%60 == 0
	for _, lp := range g.pqLocal() {
		a, pr, sp := lp.A, lp.Pr, lp.Sp
		pr.Zone = z.Key
		mood := g.pqMood(pr, a)
		eff := pqEff(mood)
		if g.Turn%10 == 0 && a.HP < a.MaxHP && pr.Food > 20 {
			a.HP++
		}
		if hurt := a.HP*100 < a.MaxHP*35; hurt != pr.Hurt {
			pr.Hurt = hurt
			if hurt {
				g.Msg(fmt.Sprintf("{{R|%s is badly hurt and keeps close.}}", pr.Name))
			} else {
				g.Msg(fmt.Sprintf("{{G|%s is on its feet again.}}", pr.Name))
			}
		}
		// vermin hunting
		if eff > 0 && sp.has(sp.Perks, "vermin") {
			for _, v := range z.Actors {
				if v.HP > 0 && v.Faction == "vermin" && dist(a, v) <= 2 && g.R.Chance(0.35*eff) {
					g.Msg(fmt.Sprintf("%s pounces on %s.", pr.Name, g.the(v)))
					g.hurt(v, v.HP+5, a, pr.Name)
					pr.Kills++
					pr.XP++
					break
				}
			}
		}
		// warnings
		if g.Turn >= pr.AlertAt && eff > 0 {
			rng := 0
			if sp.has(sp.Perks, "alarm") {
				rng = 9
			} else if sp.has(sp.Perks, "eyes") {
				rng = 7
			}
			wl := pr.trick("warn")
			if wl > 0 {
				rng += 3 * wl
			}
			if rng > 0 {
				for _, h := range z.Actors {
					if h == a || h.HP <= 0 || h.Has("pqpet") || !g.Hostile(h) || dist(a, h) > rng || g.visible(h.X, h.Y) {
						continue
					}
					if t, ok := st.Warned[h.ID]; ok && g.Turn-t < 120 {
						continue
					}
					st.Warned[h.ID] = g.Turn
					pr.AlertAt = g.Turn + 8
					v := pqVerbs[sp.ID]
					if v == "" {
						v = "growls"
					}
					where := ""
					if wl >= 2 {
						where = " " + pqDir(h.X-g.P.X, h.Y-g.P.Y)
					}
					g.Msg(fmt.Sprintf("{{W|%s %s%s.}}", pr.Name, v, where))
					if wl > 0 {
						pr.Prac["warn"]++
					}
					break
				}
			}
		}
		// scouting
		if lv := pr.trick("scout"); lv > 0 && eff > 0 && pqOutdoors(z) && g.Turn%6 == 0 {
			rad := 3 + 2*lv
			for dy := -rad; dy <= rad; dy++ {
				for dx := -rad; dx <= rad; dx++ {
					if x, y := a.X+dx, a.Y+dy; z.In(x, y) && dx*dx+dy*dy <= rad*rad {
						z.Seen[y*MW+x] = 1
					}
				}
			}
		}
		// herbs
		if lv := pr.trick("herbs"); lv > 0 && eff > 0 && pqOutdoors(z) && g.Turn >= pr.HerbAt && g.R.Chance(0.012*float64(lv)*eff) {
			pr.HerbAt = g.Turn + 60
			tag := cookTagIDs[g.R.Int(0, len(cookTagIDs)-1)]
			it := g.makeIngredient("herb sprig", tag, 1, CGreen)
			it.X, it.Y = g.P.X, g.P.Y
			z.Items = append(z.Items, it)
			pr.Prac["herbs"]++
			g.Msg(fmt.Sprintf("%s noses out a herb and drops it at your feet.", pr.Name))
		}
		// stealing small things
		if lv := pr.trick("steal"); lv > 0 && eff > 0 && (z.Kind == "town" || z.Kind == "outskirts") && g.Turn >= pr.StealAt && g.R.Chance(0.01*float64(lv)*eff) {
			var vic *Actor
			for _, o := range z.Actors {
				if o != a && o.HP > 0 && o.Gold > 0 && o.AI != "follower" && !g.Hostile(o) && o.Role != "guard" && dist(a, o) <= 4 {
					vic = o
					break
				}
			}
			if vic != nil {
				pr.StealAt = g.Turn + 180
				n := mini(vic.Gold, g.R.Int(1, 3+3*lv))
				vic.Gold -= n
				z.Items = append(z.Items, &Item{ID: g.newID(), Kind: "gold", Name: g.Sk.Currency, Glyph: '$', F: CYellow, Count: n, X: g.P.X, Y: g.P.Y})
				pr.Prac["steal"]++
				if g.R.Chance(maxf(0.1, 0.35-0.08*float64(lv))) {
					g.Msg(fmt.Sprintf("{{R|%s is caught with a purse-string in its teeth! Someone saw.}}", pr.Name))
					if strings.HasPrefix(vic.Faction, "state:") {
						g.AddRep(vic.Faction, -15, " for a thieving pet")
					}
				} else {
					g.Msg(fmt.Sprintf("%s slips back with a few coins.", pr.Name))
				}
			}
		}
		// warmth
		if lv := pr.trick("warm"); lv > 0 && eff > 0 && dist(a, g.P) <= 1 && g.P.HP < g.P.MaxHP && g.Turn%(40-8*lv) == 0 {
			g.P.HP++
			if g.Turn >= pr.WarmAt {
				pr.WarmAt = g.Turn + 240
				g.Msg(fmt.Sprintf("%s curls against you, warm and heavy.", pr.Name))
			}
			pr.Prac["warm"]++
		}
		if !hour {
			continue
		}
		// hourly needs
		pr.Food = clampi(pr.Food-2, 0, 100)
		if dist(a, g.P) <= 8 {
			if pr.Food >= 30 && pr.Aff < 70 {
				pr.Aff++
			}
		} else if pr.Aff > 10 {
			pr.Aff--
		}
		if pr.Food == 0 {
			if a.HP > 1 {
				a.HP--
			}
			if g.Turn%240 == 0 {
				g.Msg(fmt.Sprintf("{{R|%s is starving.}}", pr.Name))
			}
			if pr.Aff <= 0 {
				g.pqRelease(pr, a, "slipped away in the night, hungry and unloved")
				continue
			}
			pr.Aff = maxi(0, pr.Aff-1)
		} else if pr.Food < 25 && g.Turn%240 == 0 {
			g.Msg(fmt.Sprintf("{{W|%s whines and looks at your pack.}}", pr.Name))
		}
		if g.Turn%DayLen == 0 && pr.Aff >= 40 {
			pr.XP++
		}
		if pr.Young && pr.XP >= 15 {
			pr.Young = false
			g.Msg(fmt.Sprintf("{{G|%s has grown up.}}", pr.Name))
		}
		g.pqSync(a, pr)
	}
	// pets left elsewhere
	if hour {
		for _, zz := range g.Zones {
			if zz == z {
				continue
			}
			for _, a := range zz.Actors {
				if !a.Has("pqpet") || a.AI != "follower" {
					continue
				}
				if pr := st.rec(a.ID); pr != nil {
					pr.Zone = zz.Key
					pr.Food = clampi(pr.Food-1, 0, 100)
					if pr.Aff > 0 && g.Turn%240 == 0 {
						pr.Aff--
					}
				}
			}
		}
	}
}

// ---- follower AI hook ----

// pqAct: wounded pets keep clear of fights; fetchers bring small things. Hook: top of aiAct.
func (g *Game) pqAct(a *Actor) bool {
	if a == nil || !a.Has("pqpet") || a.AI != "follower" || g.Cur == nil {
		return false
	}
	pr := g.pq().rec(a.ID)
	if pr == nil {
		return false
	}
	p := g.P
	fly := a.Has("flying")
	var foe *Actor
	best := 1 << 20
	for _, o := range g.Cur.Actors {
		if o.HP > 0 && o != a && !o.Has("pqpet") && g.Hostile(o) {
			if d := dist(a, o); d < best {
				foe, best = o, d
			}
		}
	}
	if a.HP*100 < a.MaxHP*35 {
		if foe != nil && best <= 3 {
			g.stepToward(a, g.distMap(foe.X, foe.Y, fly), true)
		} else if dist(a, p) > 2 {
			g.stepToward(a, g.distMap(p.X, p.Y, fly), false)
		}
		return true
	}
	if foe != nil && best <= 9 {
		return false
	}
	z := g.Cur
	if pr.Carry != nil {
		if dist(a, p) <= 1 {
			it := pr.Carry
			it.X, it.Y = p.X, p.Y
			z.Items = append(z.Items, it)
			pr.Carry = nil
			pr.FetchAt = g.Turn + 45 - 10*pr.trick("fetch")
			pr.Prac["fetch"]++
			g.Msg(fmt.Sprintf("%s drops %s at your feet.", pr.Name, strings.ToLower(it.Name)))
		} else {
			g.stepToward(a, g.distMap(p.X, p.Y, fly), false)
		}
		return true
	}
	lv := pr.trick("fetch")
	if lv == 0 || z.Kind == "town" || pqEff(g.pqMood(pr, a)) == 0 {
		return false
	}
	if pr.FetchID != 0 {
		var it *Item
		for _, x := range z.Items {
			if x.ID == pr.FetchID {
				it = x
				break
			}
		}
		if it == nil || g.Turn > pr.FetchTo {
			pr.FetchID, pr.FetchAt = 0, g.Turn+15
			return false
		}
		if a.X == it.X && a.Y == it.Y {
			removeItem(&z.Items, it)
			pr.Carry, pr.FetchID = it, 0
			g.Msg(fmt.Sprintf("%s picks up %s.", pr.Name, strings.ToLower(it.Name)))
			return true
		}
		g.stepToward(a, g.distMap(it.X, it.Y, fly), false)
		return true
	}
	if g.Turn < pr.FetchAt || !g.R.Chance(0.25) {
		return false
	}
	rad := 4 + 2*lv
	dm := g.distMap(a.X, a.Y, fly)
	var pick *Item
	pd := 1 << 20
	for _, it := range z.Items {
		if !pqSmallItem(it) || maxi(abs(it.X-a.X), abs(it.Y-a.Y)) > rad || !g.visible(it.X, it.Y) {
			continue
		}
		if d := dm[it.Y*MW+it.X]; d < pd && d < 2*rad+2 {
			pick, pd = it, d
		}
	}
	if pick == nil {
		pr.FetchAt = g.Turn + 20
		return false
	}
	pr.FetchID, pr.FetchTo = pick.ID, g.Turn+40
	return false
}

func pqSmallItem(it *Item) bool {
	if it.Quest != 0 || it.SimArt != 0 || it.Extra["uw:own"] != 0 {
		return false
	}
	switch it.Kind {
	case "gold", "ingredient", "scrap":
		return true
	}
	return false
}

// ---- abilities ----

func (g *Game) pqAbilities() []Ability {
	if g.P == nil || g.W == nil {
		return nil
	}
	return []Ability{{ID: "pq_pets", Name: "Pets", Target: "self"}, {ID: "pq_modes", Name: "Play modes", Target: "self"}}
}

func (g *Game) pqAbUse(id string) {
	switch id {
	case "pq_pets":
		g.pqOpen(0)
	case "pq_modes":
		g.pqOpen(1)
	}
}

func (g *Game) pqOpen(tab int) {
	if g.P == nil || g.Mode != "main" && g.Mode != "pq" {
		return
	}
	st := g.pq()
	st.Tab, st.Sel, st.Sub, st.Note = tab, 0, "", ""
	g.Mode = "pq"
}

// ---- play modes ----

func pqNorm(id string) string { return strings.TrimPrefix(id, "pq:") }

func (g *Game) pqOn(id string) bool { return g.pq().Modes[id] }

// pqChallengeList: play modes in the challenge format, ids "pq:<id>", for the character creation list.
func pqChallengeList() []ijChallenge {
	out := make([]ijChallenge, 0, len(pqModes))
	for _, m := range pqModes {
		d := m.Desc
		if m.StartOnly {
			d += " (chosen at the start only)"
		}
		out = append(out, ijChallenge{ID: "pq:" + m.ID, Name: m.Name, Desc: d, Mul: m.Mul})
	}
	return out
}

// pqScoreMul multiplies the hall-of-fame score (halloffame.go hfScore).
func (g *Game) pqScoreMul() float64 {
	m := 1.0
	for id, on := range g.pq().Modes {
		if md := pqModeByID[id]; on && md != nil {
			m *= md.Mul
		}
	}
	return m
}

// pqApply switches on the named play modes (ids with or without the "pq:" prefix; unknown ids are ignored).
// Called at the end of NewGame with CharSpec.Mods.
func pqApply(g *Game, modes []string) {
	if g == nil || g.P == nil {
		return
	}
	var names []string
	for _, raw := range modes {
		id := pqNorm(raw)
		if m := pqModeByID[id]; m != nil && g.pqEnable(m, true) {
			names = append(names, m.Name)
		}
	}
	if len(names) > 0 {
		g.Msg("{{Y|Play mode:}} " + strings.Join(names, ", "))
	}
}

// pqEnable switches a mode on; false means refused (excluded by another mode or already on).
func (g *Game) pqEnable(m *pqMode, on bool) bool {
	st := g.pq()
	if st.Modes == nil {
		st.Modes = map[string]bool{}
	}
	if on {
		if st.Modes[m.ID] {
			return false
		}
		for _, x := range m.Excludes {
			if st.Modes[x] {
				return false
			}
		}
		if m.ID == "story" && g.ijHas("ironman") {
			return false
		}
		st.Modes[m.ID] = true
		switch m.ID {
		case "ironman":
			if !g.ijHas("ironman") {
				ij := g.ijSt()
				ij.Mods = append(ij.Mods, "ironman")
			}
			g.GameMode = "classic"
		case "onehit":
			pqCap(g)
		case "peaceful":
			for _, z := range g.Zones {
				pqCalm(g, z)
			}
		}
		return true
	}
	if !st.Modes[m.ID] {
		return false
	}
	delete(st.Modes, m.ID)
	if m.ID == "peaceful" {
		pqUncalm(g)
	}
	return true
}

func pqCap(g *Game) {
	p := g.P
	if c := 12 + p.Level; p.MaxHP > c {
		p.MaxHP = c
		p.HP = mini(p.HP, c)
	}
}

// pqCalm quietens most hostile creatures of a zone (peaceful explorer). Pure hash, never g.R.
func pqCalm(g *Game, z *Zone) {
	if !g.pq().Modes["peaceful"] || z == nil {
		return
	}
	st := g.pq()
	for _, a := range z.Actors {
		if a.HP <= 0 || a.AI != "hostile" || a.Boss || a.Angry || a.SimMon != 0 || a.Comp != nil || a.Has("pqpet") || a.Has("pqcalm") || a.Faction == "player" {
			continue
		}
		if !g.Hostile(a) {
			continue
		}
		if g.pqHash("calm", a.ID)%100 >= 92 { // a few stay aggressive
			continue
		}
		if st.Calm == nil {
			st.Calm = map[int]string{}
		}
		st.Calm[a.ID] = a.Faction
		a.Faction = "pqcalm"
		a.Set("pqcalm")
	}
}

func pqUncalm(g *Game) {
	st := g.pq()
	for _, z := range g.Zones {
		for _, a := range z.Actors {
			if a.Has("pqcalm") {
				if f, ok := st.Calm[a.ID]; ok {
					a.Faction = f
				}
				delete(a.Flags, "pqcalm")
			}
		}
	}
	st.Calm = nil
}

func pqModesTurn(g *Game, st *pqState) {
	if len(st.Modes) == 0 {
		return
	}
	if st.Modes["peaceful"] {
		for _, a := range g.Cur.Actors {
			if a.Has("pqcalm") && a.Angry {
				if f, ok := st.Calm[a.ID]; ok {
					a.Faction = f
				}
				delete(a.Flags, "pqcalm")
			}
		}
		if g.Turn%5 == 0 {
			pqCalm(g, g.Cur)
		}
	}
	if st.Modes["cozy"] {
		if g.Hunger > 900 {
			g.Hunger = 900
		}
		if g.Thirst > 700 {
			g.Thirst = 700
		}
	}
	if st.Modes["onehit"] {
		pqCap(g)
	}
}

func pqEnter(g *Game, zn *Zone) {
	if g.P == nil {
		return
	}
	pqCalm(g, zn)
	if zn.Kind != "town" {
		return
	}
	l := g.pql()
	for _, a := range zn.Actors {
		if a.HP > 0 && (a.Role == "innkeeper" || a.Role == "tavern" || a.Role == "lw_innkeeper") {
			x, y, _ := g.nearestFree(zn, a.X, a.Y)
			l.InnKey, l.InnX, l.InnY, l.InnName = zn.Key, x, y, zn.Name
			return
		}
	}
	if l.InnKey == "" {
		l.InnKey, l.InnX, l.InnY, l.InnName = zn.Key, g.P.X, g.P.Y, zn.Name
	}
}

// pqRevive (story mode): instead of dying, wake at the last inn for some coin. Returns true when it took
// over. Hook: first line of die().
func (g *Game) pqRevive(cause string, src *Actor) bool {
	st := g.pq()
	if !st.Modes["story"] || g.P == nil || g.Cur == nil || g.ijHas("ironman") {
		return false
	}
	p := g.P
	cost := 20 + 12*p.Level
	if have := g.cuTotal(); cost > have {
		cost = have
	}
	paid := ""
	if cost > 0 && g.cuSpend(cost) {
		paid = fmt.Sprintf(" The healer's bill: %s.", g.cuFormat(cost))
	}
	l := g.pql()
	where := g.Cur.Name
	if zn := g.Zones[l.InnKey]; zn != nil && l.InnKey != "" {
		if zn != g.Cur {
			g.enterZone(zn, l.InnX, l.InnY, true)
		} else {
			p.X, p.Y = l.InnX, l.InnY
		}
		where = l.InnName
	} else if x, y, ok := g.nearestFree(g.Cur, p.X, p.Y); ok {
		p.X, p.Y = x, y
	}
	p.HP = p.MaxHP
	p.Status = nil
	g.Dead, g.DeathText = false, ""
	g.Hunger, g.Thirst = mini(g.Hunger, 700), mini(g.Thirst, 500)
	g.Breath = g.breathCap()
	g.Turn += 240
	g.Mode = "main"
	st.Revs++
	g.Msg("{{Y|Story mode:}} the dark closes in... you wake in a quiet bed at " + where + "." + paid)
	g.deed("event", fmt.Sprintf("%s was carried half-dead to %s and woke there.", p.Name, where), 0, 0)
	g.recompute()
	return true
}

// ---- screen ----

func pqKeys(g *Game, k string) {
	st := g.pq()
	switch st.Sub {
	case "name", "rename":
		pqNameKey(g, st, k)
		return
	}
	if k == "Escape" || k == "q" {
		if st.Sub != "" {
			st.Sub = ""
			return
		}
		g.Mode = "main"
		return
	}
	if k == "Tab" || k == "ArrowLeft" || k == "ArrowRight" {
		st.Tab = 1 - st.Tab
		st.Sel, st.Sub, st.Note = 0, "", ""
		return
	}
	if st.Tab == 1 {
		pqModeKeys(g, st, k)
		return
	}
	n := len(st.Pets)
	switch k {
	case "ArrowUp", "N8", "k":
		st.Sel = maxi(0, st.Sel-1)
		st.Sub = ""
		return
	case "ArrowDown", "N2", "j":
		st.Sel = mini(maxi(0, n-1), st.Sel+1)
		st.Sub = ""
		return
	}
	if n == 0 {
		return
	}
	st.Sel = clampi(st.Sel, 0, n-1)
	pr := st.Pets[st.Sel]
	a, _ := g.pqFind(pr.ID)
	sp := pqSpByID[pr.Sp]
	if sp == nil {
		return
	}
	here := a != nil && pqInZone(g, a)
	if st.Sub == "release" {
		if k == "y" || k == "Y" {
			st.Sub = ""
			g.pqRelease(pr, a, "is left to fend for itself")
			st.Note = pr.Name + " goes its own way."
		} else {
			st.Sub = ""
		}
		return
	}
	if st.Sub == "trick" {
		if len(k) == 1 && k[0] >= '1' && k[0] <= '7' {
			pqPractise(g, st, pr, a, sp, pqTricks[int(k[0]-'1')].ID)
		}
		return
	}
	switch k {
	case "f":
		if !here {
			st.Note = "It is not here to feed."
			return
		}
		it := g.pqFoodItem(sp)
		if it == nil {
			if sp.Mech {
				st.Note = "It wants scrap to run on; you have none."
			} else {
				st.Note = "You have nothing it will eat."
			}
			return
		}
		n := pqNourish(sp, it)
		name := it.Name
		g.pqUseItem(it)
		pr.Food = clampi(pr.Food+n, 0, 100)
		pr.Aff = clampi(pr.Aff+2, 0, 100)
		g.spendTurns(1)
		st.Note = fmt.Sprintf("%s eats the %s.", pr.Name, strings.ToLower(name))
	case "p":
		if !here {
			st.Note = "It is not here."
			return
		}
		if g.Turn < pr.PlayAt {
			st.Note = pr.Name + " has had enough attention for now."
			return
		}
		pr.PlayAt = g.Turn + 20
		pr.Aff = clampi(pr.Aff+4, 0, 100)
		g.spendTurns(3)
		st.Note = fmt.Sprintf("You spend a few minutes with %s. It leans into you.", pr.Name)
	case "t":
		st.Sub = "trick"
		st.Note = ""
	case "r":
		st.Sub, st.Edit, st.Buf = "rename", pr.ID, ""
	case "x":
		st.Sub = "release"
	case "d":
		if a == nil {
			st.drop(pr.ID)
			st.Note = "You stop looking for " + pr.Name + "."
		}
	}
}

func pqInZone(g *Game, a *Actor) bool {
	for _, o := range g.Cur.Actors {
		if o == a {
			return true
		}
	}
	return false
}

func pqPractise(g *Game, st *pqState, pr *pqPet, a *Actor, sp *pqSp, trick string) {
	if a == nil || !pqInZone(g, a) {
		st.Note = "It has to be here to practise."
		return
	}
	for _, o := range g.Cur.Actors {
		if o.HP > 0 && !o.Has("pqpet") && g.Hostile(o) && g.visible(o.X, o.Y) {
			st.Note = "Not with enemies in sight."
			return
		}
	}
	if g.Turn < pr.PracAt {
		st.Note = pr.Name + " is tired of lessons for the moment."
		return
	}
	mood := g.pqMood(pr, a)
	if pr.Food < 15 || pqEff(mood) < 0.5 {
		st.Note = fmt.Sprintf("%s is too %s to learn today.", pr.Name, mood)
		return
	}
	if pr.Prac == nil {
		pr.Prac = map[string]int{}
	}
	limit := 9
	switch sp.Smart {
	case 2:
		limit = 19
	case 3:
		limit = 99
	}
	if pr.Prac[trick] >= limit {
		st.Note = fmt.Sprintf("%s has learned all it can of this.", pr.Name)
		return
	}
	pr.PracAt = g.Turn + 30
	pr.Food = clampi(pr.Food-4, 0, 100)
	g.spendTurns(10)
	chance := 0.5 + float64(pr.Aff)/250
	gain := 1
	if sp.has(sp.Apt, trick) {
		gain++
		chance += 0.1
	}
	if g.R.Chance(minf(chance, 0.92)) {
		before := pqTrickLv(pr.Prac[trick])
		pr.Prac[trick] = mini(pr.Prac[trick]+gain, limit)
		pr.XP++
		after := pqTrickLv(pr.Prac[trick])
		t := pqTrickByID[trick]
		if after > before {
			st.Note = fmt.Sprintf("{{G|%s has learned %s (level %d)!}}", pr.Name, t.Name, after)
			g.pqSync(a, pr)
		} else {
			st.Note = fmt.Sprintf("%s makes progress with %s.", pr.Name, strings.ToLower(t.Name))
		}
	} else {
		st.Note = fmt.Sprintf("%s loses interest and chases its tail.", pr.Name)
	}
}

func pqNameKey(g *Game, st *pqState, k string) {
	pr := st.rec(st.Edit)
	switch k {
	case "Enter", "Escape":
		if pr != nil {
			if s := strings.TrimSpace(st.Buf); s != "" && k == "Enter" {
				pr.Name = s
			}
			if a, _ := g.pqFind(pr.ID); a != nil {
				a.Name = pr.Name
			}
			g.Msg(fmt.Sprintf("{{G|%s it is.}}", pr.Name))
		}
		st.Sub, st.Buf = "", ""
		if st.Tab == 0 && len(st.Pets) == 0 {
			g.Mode = "main"
		}
	case "Backspace":
		if rs := []rune(st.Buf); len(rs) > 0 {
			st.Buf = string(rs[:len(rs)-1])
		}
	default:
		if rs := []rune(k); len(rs) == 1 && rs[0] >= 32 && len([]rune(st.Buf)) < 16 {
			st.Buf += k
		}
	}
}

func pqModeKeys(g *Game, st *pqState, k string) {
	switch k {
	case "ArrowUp", "N8", "k":
		st.Sel = maxi(0, st.Sel-1)
	case "ArrowDown", "N2", "j":
		st.Sel = mini(len(pqModes)-1, st.Sel+1)
	case "Enter", " ", "Space":
		m := &pqModes[clampi(st.Sel, 0, len(pqModes)-1)]
		if m.StartOnly {
			st.Note = m.Name + " can only be chosen when a run begins."
			return
		}
		on := g.pq().Modes[m.ID]
		if g.pqEnable(m, !on) {
			if on {
				st.Note = m.Name + " is off."
			} else {
				st.Note = m.Name + " is on. Your score counts less."
			}
		} else {
			st.Note = m.Name + " cannot change right now."
		}
	}
}

func (g *Game) pqDraw() {
	sc := g.sc
	st := g.pq()
	x, y, w, h := g.panel("Pets and play modes")
	tabs := []string{"Pets", "Play modes"}
	line := ""
	for i, t := range tabs {
		if i == st.Tab {
			line += "{{W|[" + t + "]}} "
		} else {
			line += "{{K| " + t + " }} "
		}
	}
	sc.TextB(x, y, line+"  {{K|Tab switch  Esc close}}", CGray, CBgDark, x+w)
	y += 2
	bottom := y + h - 6
	if st.Sub == "name" || st.Sub == "rename" {
		pr := st.rec(st.Edit)
		who := "your new companion"
		if pr != nil {
			who = pr.Name
		}
		sc.TextB(x, y, "What will you call "+who+"?", CGray, CBgDark, x+w)
		sc.TextB(x, y+2, "> "+st.Buf+"_", CYellow, CBgDark, x+w)
		sc.TextB(x, y+4, "{{K|Enter accepts (empty keeps the suggested name). Esc keeps the current name.}}", CGray, CBgDark, x+w)
		return
	}
	if st.Tab == 1 {
		for i, m := range pqModes {
			mark := "  "
			if i == st.Sel {
				mark = "> "
			}
			box := "[ ]"
			if st.Modes[m.ID] {
				box = "[x]"
			}
			tail := ""
			if m.StartOnly {
				tail = " {{K|(start only)}}"
			}
			sc.TextB(x, y, fmt.Sprintf("%s%s %s%s", mark, box, m.Name, tail), CGray, CBgDark, x+w)
			y++
		}
		y++
		if st.Sel >= 0 && st.Sel < len(pqModes) {
			for _, l := range Wrap("{{K|"+pqModes[st.Sel].Desc+"}}", w) {
				sc.TextB(x, y, l, CGray, CBgDark, x+w)
				y++
			}
		}
		y++
		sc.TextB(x, y, fmt.Sprintf("{{K|Score multiplier from modes: x%.2f}}", g.pqScoreMul()), CGray, CBgDark, x+w)
		sc.TextB(x, bottom+2, "{{W|Up/Down}} choose   {{W|Enter}} toggle (not for start-only modes)   "+st.Note, CGray, CBgDark, x+w)
		return
	}
	if len(st.Pets) == 0 {
		sc.TextB(x, y, "You have no pets. Strays turn up on roads and at the edges of towns; animal sellers work some markets;", CGray, CBgDark, x+w)
		sc.TextB(x, y+1, "innkeepers and farmers now and then have a litter that needs a home.", CGray, CBgDark, x+w)
		y += 3
	}
	st.Sel = clampi(st.Sel, 0, maxi(0, len(st.Pets)-1))
	for i, pr := range st.Pets {
		sp := pqSpByID[pr.Sp]
		if sp == nil {
			continue
		}
		a, zn := g.pqFind(pr.ID)
		where := "whereabouts unknown"
		if zn != nil {
			where = zn.Name
			if zn == g.Cur {
				where = "here"
			}
		}
		mark := "  "
		if i == st.Sel {
			mark = "> "
		}
		mood := g.pqMood(pr, a)
		sc.TextB(x, y, fmt.Sprintf("%s{{W|%s}} the %s  lv %d  %s  {{K|(%s)}}", mark, pr.Name, g.pqNoun(sp), pr.lv(), pqMoodText[mood], where), CGray, CBgDark, x+w)
		y++
	}
	if len(st.Pets) > 0 {
		pr := st.Pets[st.Sel]
		sp := pqSpByID[pr.Sp]
		a, _ := g.pqFind(pr.ID)
		y++
		hp := ""
		if a != nil {
			hp = fmt.Sprintf("  HP %d/%d", a.HP, a.MaxHP)
		}
		sc.TextB(x, y, fmt.Sprintf("Food %s %3d   Affection %s %3d%s   Days together %d", pqBar(pr.Food), pr.Food, pqBar(pr.Aff), pr.Aff, hp, g.pqLive(pr)), CGray, CBgDark, x+w)
		y++
		for _, l := range Wrap("{{K|"+sp.Desc+"}}", w) {
			sc.TextB(x, y, l, CGray, CBgDark, x+w)
			y++
		}
		y++
		var perks []string
		for _, p := range sp.Perks {
			perks = append(perks, pqPerkText[p])
		}
		sc.TextB(x, y, "Natural gifts: "+strings.Join(perks, "; "), CGray, CBgDark, x+w)
		y++
		sc.TextB(x, y, fmt.Sprintf("Kills %d   Eats %s", pr.Kills, pqDiet(sp)), CGray, CBgDark, x+w)
		y += 2
		sc.TextB(x, y, "{{W|Tricks}}", CGray, CBgDark, x+w)
		y++
		for i, t := range pqTricks {
			lv := pr.trick(t.ID)
			pts := 0
			if pr.Prac != nil {
				pts = pr.Prac[t.ID]
			}
			txt := "{{K|not yet learned}}"
			if lv > 0 {
				txt = t.Desc[lv-1]
			}
			apt := ""
			if sp.has(sp.Apt, t.ID) {
				apt = " {{c|(gifted)}}"
			}
			pre := fmt.Sprintf("%d ", i+1)
			if st.Sub != "trick" {
				pre = "  "
			}
			if y < bottom {
				sc.TextB(x, y, fmt.Sprintf("%s%-18s lv %d {{K|(%2d practice)}}%s  %s", pre, t.Name, lv, pts, apt, txt), CGray, CBgDark, x+w)
				y++
			}
		}
	}
	if len(st.Mem) > 0 && y < bottom {
		y++
		sc.TextB(x, y, "{{W|In memory}}", CGray, CBgDark, x+w)
		y++
		for i := len(st.Mem) - 1; i >= 0 && y < bottom+1; i-- {
			m := st.Mem[i]
			sc.TextB(x, y, fmt.Sprintf("{{K|%s the %s, %d days, lost to %s near %s}}", m.Name, m.Sp, m.Days, m.Cause, m.Place), CGray, CBgDark, x+w)
			y++
		}
	}
	keys := "{{W|f}} feed  {{W|p}} play  {{W|t}} practise a trick  {{W|r}} rename  {{W|x}} give away  {{W|d}} forget (if lost)"
	if st.Sub == "trick" {
		keys = "{{W|1-7}} practise that trick   Esc back"
	} else if st.Sub == "release" {
		keys = "{{R|Give it up? y/n}}"
	}
	sc.TextB(x, bottom+1, keys, CGray, CBgDark, x+w)
	sc.TextB(x, bottom+2, st.Note, CGray, CBgDark, x+w)
}

var pqPerkText = map[string]string{
	"vermin": "hunts vermin", "alarm": "raises the alarm at unseen foes", "eyes": "sharp eyes (+perception)", "nose": "keen nose (+survival)", "charm": "winning ways when fond of you",
}

func pqDiet(sp *pqSp) string {
	if sp.Mech {
		return "scrap"
	}
	if sp.ID == "cat" || sp.ID == "hawk" || sp.ID == "ferret" {
		return "meat and scraps"
	}
	return "food and scraps"
}
