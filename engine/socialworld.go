package engine

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// Social world (prefix sw): Realmforge's social orders, genre forces, reforms, persecutions, famine
// ethics, intermarriage and cultural prestige, read from the world file (StateData.SocialOrder/Forces/
// Reforms/Persecutions/Famine/Mixing, CultureData.Prestige/Origin, PersonData.Mixed, WorldData.Forces).
// Towns get flavour (unfree workers or constructs, notices, pickets, corp offices, vat clinics, commons
// gardens), reform movements get a quest-giver, havens get refugee camps, fallout and ley nodes touch the
// land. Generation only uses side rngs (seed:sw:zoneKey), so a world without these fields is unchanged.
// State: g.ext("sw").

func init() {
	zoneHooks = append(zoneHooks, swZone)
	enterHooks = append(enterHooks, swEnter)
	turnHooks = append(turnHooks, swTurn)
	talkHooks = append(talkHooks, swTalk)
	sideHooks = append(sideHooks, swSide)
	unLocalKeys = append(unLocalKeys, "sw") // quests, helped actors: planet-local
}

type swQuest struct {
	ID     int    `json:"id"` // Quest.ID in g.Quests
	Kind   string `json:"k"`  // pamphlets captive feed
	State  int    `json:"st"` // realm the movement / camp belongs to
	Giver  int    `json:"gv"` // actor id
	Burg   int    `json:"b,omitempty"`
	Zone   string `json:"z,omitempty"`
	Placed bool   `json:"pl,omitempty"`
	Stage  int    `json:"sg,omitempty"` // 1 = done, return to the giver
	Closed bool   `json:"c,omitempty"`
}

type swState struct {
	Quests map[int]*swQuest `json:"q,omitempty"`
	Helped map[int]bool     `json:"h,omitempty"` // unfree / refugee actors already helped
	Asked  map[int]int      `json:"a,omitempty"` // giver actor id → quest id
}

func swSt(g *Game) *swState {
	s := g.ext("sw", func() any { return &swState{} }).(*swState)
	if s.Quests == nil {
		s.Quests = map[int]*swQuest{}
	}
	if s.Helped == nil {
		s.Helped = map[int]bool{}
	}
	if s.Asked == nil {
		s.Asked = map[int]int{}
	}
	return s
}

// ---- reading the world ----

func (w *WorldData) swOrder(state int) *SocialOrderData {
	if st := w.stateByID[state]; st != nil && st.Alive {
		return st.SocialOrder
	}
	return nil
}

func swHasForce(st *StateData, kind string) *ForceData {
	if st == nil {
		return nil
	}
	for i := range st.Forces {
		if st.Forces[i].Kind == kind {
			return &st.Forces[i]
		}
	}
	return nil
}

func swEnacted(st *StateData, kind string) bool {
	if st == nil || st.Reforms == nil {
		return false
	}
	for _, k := range st.Reforms.Enacted {
		if k == kind {
			return true
		}
	}
	return false
}

// swActivePers: persecutions still under way in a realm.
func swActivePers(st *StateData) []*PersecutionData {
	if st == nil {
		return nil
	}
	var out []*PersecutionData
	for i := range st.Persecutions {
		if st.Persecutions[i].Ended == 0 {
			out = append(out, &st.Persecutions[i])
		}
	}
	return out
}

// swPeople: "the Quwaniha" or "the faithful of X".
func swPeople(p *PersecutionData) string {
	if p.Axis == "faith" {
		return "faithful of the " + p.TargetName
	}
	return p.TargetName
}

func (g *Game) swPlayerCulture() int {
	if b := g.W.burgByID[g.StartBurg]; b != nil {
		return b.Culture
	}
	if st := g.W.stateByID[g.HomeState]; st != nil {
		return st.Culture
	}
	return -1
}

// swPersecutesMe: a persecution in this realm aimed at the player's people or faith (nil if tolerated).
func (g *Game) swPersecutesMe(state int) *PersecutionData {
	st := g.W.stateByID[state]
	for _, p := range swActivePers(st) {
		switch p.Axis {
		case "faith":
			if p.Target == g.HomeFaith && !swEnacted(st, "tolerance") {
				return p
			}
		case "culture":
			if p.Target == g.swPlayerCulture() {
				return p
			}
			if cu := g.W.cultByID[p.Target]; cu != nil && g.P != nil && cu.Species != "" && cu.Species != "human" && cu.Species == g.P.Species {
				return p
			}
		}
	}
	return nil
}

func (w *WorldData) swCultName(id int) string {
	if c := w.cultByID[id]; c != nil {
		return c.Name
	}
	return "strangers"
}

func (w *WorldData) swSpeciesAdj(id string) string {
	if s := w.specByID[id]; s != nil {
		return strings.ToLower(s.Adj)
	}
	return id
}

// swSingular turns a workers term into one person: thralls → thrall, bondsfolk → bondsman.
func swSingular(s string) string {
	switch {
	case s == "":
		return "labourer"
	case strings.HasSuffix(s, "folk"):
		return strings.TrimSuffix(s, "folk") + "person"
	case strings.HasSuffix(s, "men"):
		return strings.TrimSuffix(s, "men") + "man"
	case s == "indentured" || s == "vatborn" || s == "uncleared" || s == "unmodded":
		return s + " worker"
	case strings.HasSuffix(s, "ies"):
		return strings.TrimSuffix(s, "ies") + "y"
	case strings.HasSuffix(s, "s"):
		return strings.TrimSuffix(s, "s")
	}
	return s
}

func swFreeSlots(g *Game, bd *BurgData) [][2]int {
	cp := g.cityPlanFor(bd)
	ol := [2]int{-9, -9}
	if g.W.outlawByBrg[bd.ID] != nil {
		ol = g.outlawZone(bd)
	}
	var out [][2]int
	for _, s := range [][2]int{{2, 2}, {0, 2}, {2, 0}, {0, 0}, {1, 2}, {1, 0}, {2, 1}, {0, 1}} {
		if _, ok := cp.districts[s]; ok || s == ol {
			continue
		}
		out = append(out, s)
	}
	return out
}

// ---- strata: the realm's order names its lowest stratum ----

var swStrataCache = map[string]*strataDef{}

// swStrata adapts a base ladder to the realm's social order: the unfree or bound sit in the low slot
// under their own name (thralls, serfs, company hands...); egalitarian realms use the commons ladder.
func (g *Game) swStrata(bd *BurgData, base *strataDef) *strataDef {
	if bd == nil || g.W == nil {
		return base
	}
	so := g.W.swOrder(bd.State)
	if so == nil || so.Workers == "" {
		return base
	}
	switch so.Kind {
	case "egalitarian":
		if base == defFeudal || base == defCaste || base == defSinic {
			return defCommunal
		}
		return base
	case "slave", "feudal", "caste":
	default:
		return base
	}
	key := fmt.Sprintf("%p:%s", base, so.Workers)
	if d, ok := swStrataCache[key]; ok {
		return d
	}
	d := &strataDef{list: append([]string(nil), base.list...), slot: map[string]int{}}
	for k, v := range base.slot {
		d.slot[k] = v
	}
	low, com := base.slot["low"], base.slot["common"]
	if low == com {
		// insert the bound stratum just below the commons
		at := com + 1
		d.list = append(d.list[:at], append([]string{so.Workers}, d.list[at:]...)...)
		for k, v := range d.slot {
			if v >= at {
				d.slot[k] = v + 1
			}
		}
		d.slot["low"] = at
	} else {
		d.list[low] = so.Workers
	}
	swStrataCache[key] = d
	return d
}

// ---- zone generation ----

func swZone(g *Game, zn *Zone, _ *Rng) {
	w := g.W
	if w == nil || !w.In(zn.WX, zn.WY) {
		return
	}
	p := w.Idx(zn.WX, zn.WY)
	r := NewRng(w.Seed + ":sw:" + zn.Key)
	if zn.Kind == "town" && zn.Burg > 0 {
		if bd := w.burgByID[zn.Burg]; bd != nil && bd.Alive {
			swTown(g, zn, r, bd)
		}
	}
	if zn.Z == 0 && zn.Kind == "outskirts" {
		if bd := w.burgByID[w.Burg[p]]; bd != nil && bd.Alive {
			swOutskirts(g, zn, r, bd)
		}
	}
	if zn.Z == 0 && w.Forces != nil {
		if fz := swFalloutAt(w, p); fz != nil && zn.Kind != "town" {
			swFalloutZone(zn, r, fz)
		}
		if l, end := swLeyAt(w, p); l != nil && zn.ZX == 1 && zn.ZY == 1 {
			swLeyStone(g, zn, r, l, end)
		}
	}
	// people from the sim who are children of two peoples
	for _, a := range zn.Actors {
		if a.SimPer > 0 && !a.Has("swmix") {
			if pp := w.swPerson(a.SimPer); pp != nil && pp.Mixed != nil {
				a.Set("swmix")
				a.Desc += " " + w.swMixedLine(pp.Culture, pp.Mixed.Culture, pp.Species, pp.Mixed.Species)
			}
		}
	}
}

func (w *WorldData) swPerson(id int) *PersonData {
	for i := range w.People {
		if w.People[i].ID == id {
			return &w.People[i]
		}
	}
	return nil
}

func (w *WorldData) swMixedLine(a, b int, sa, sb string) string {
	s := fmt.Sprintf("Child of two peoples, the %s and the %s", w.swCultName(a), w.swCultName(b))
	if sb != "" && sb != sa {
		s += fmt.Sprintf(", %s and %s blood both", w.swSpeciesAdj(nonEmpty(sa, "human")), w.swSpeciesAdj(sb))
	}
	return s + "."
}

func swNote(zn *Zone, r *Rng, text string, near func(x, y int) bool) bool {
	x, y, ok := zn.FreeSpot(r, near)
	if !ok {
		return false
	}
	zn.Set(x, y, TInscription)
	zn.Notes[y*MW+x] = text
	return true
}

func swNear(cx, cy, dx, dy int) func(x, y int) bool {
	return func(x, y int) bool { return abs(x-cx) <= dx && abs(y-cy) <= dy }
}

func swPut(zn *Zone, r *Rng, a *Actor, near func(x, y int) bool) bool {
	x, y, ok := zn.FreeSpot(r, near)
	if !ok {
		return false
	}
	a.X, a.Y, a.HomeX, a.HomeY = x, y, x, y
	zn.Actors = append(zn.Actors, a)
	return true
}

var swConstructLook = map[string]struct {
	g    rune
	f    byte
	name string
	desc string
}{
	"golems":   {'G', CGray, "labour golem", "A hulking golem of clay and rune-cut stone, hauling without rest. A word of binding glows on its brow."},
	"spirits":  {'&', CCyan, "bound spirit", "A pale spirit bound to service by a sigil-chain. It works in silence; sometimes it looks at you."},
	"automata": {'A', CYellow, "labour automaton", "A brass automaton ticking through its rounds, a work-number stamped on its chest plate."},
	"drones":   {'d', CCyan, "servitor drone", "A servitor drone humming through its task queue, a corporate barcode on its shell."},
}

// swTown: social order, forces and mixing in a town or city district.
func swTown(g *Game, zn *Zone, r *Rng, bd *BurgData) {
	w := g.W
	st := w.stateByID[bd.State]
	if st == nil || !st.Alive {
		return
	}
	cu := w.cultByID[bd.Culture]
	center := zn.ZX == 1 && zn.ZY == 1
	near := swNear(40, 12, 22, 8)
	if so := st.SocialOrder; so != nil {
		swOrderFlavour(g, zn, r, bd, st, so, center)
	}
	// genre forces
	if swHasForce(st, "strikes") != nil && bd.swIndustrial(w) {
		n := r.Int(2, 3)
		for i := 0; i < n; i++ {
			a := g.makeNPC("villager", bd, cu, r)
			a.Role, a.Title, a.AI, a.F = "striker", "striking worker", "still", CRed
			a.Desc = fmt.Sprintf("A worker of %s on the picket line, holding a placard: NO WORK WITHOUT BREAD.", bd.Name)
			a.Set("swnorec")
			swPut(zn, r, a, swNear(40, 12, 10, 4))
		}
		if center {
			swNote(zn, r, "STRIKE! The works of "+bd.Name+" stand idle. Scabs will be remembered.", near)
		}
	}
	if w.Forces != nil && center {
		for _, c := range w.Forces.Corps {
			if c.HQ != bd.ID {
				continue
			}
			x0, y0 := 4, 2
			if zn.At(x0+3, y0+2).D().Water == 0 {
				dx, dy := g.building(zn, x0, y0, 9, 5, TWall, TCarpet)
				zn.Set(x0+4, y0+2, TMachine)
				zn.Notes[(y0+2)*MW+x0+4] = c.Name + " regional office. A terminal glows: QUARTERLY TARGETS MET. DISSENT IS A PERFORMANCE ISSUE."
				for i := 0; i < 2; i++ {
					a := g.makeNPC("guard", bd, cu, r)
					a.Title = c.Name + " security"
					a.Desc = fmt.Sprintf("Corporate security of %s, visor down, a company sigil on the shoulder.", c.Name)
					a.Set("swcorp")
					swPut(zn, r, a, swNear(dx, dy+1, 4, 2))
				}
			}
		}
	}
	if f := swHasForce(st, "genevats"); f != nil && center {
		a := g.makeNPC("apothecary", bd, cu, r)
		a.Title = "vat-tender"
		a.Desc = fmt.Sprintf("A vat-tender of %s's gene clinic, gloved to the elbow, smelling of brine.", bd.Name)
		if f.N > 0.6 {
			a.Desc += " The vats are overworked; they say a blight is coming."
		}
		if swPut(zn, r, a, near) {
			swNote(zn, r, "GENE-VAT CLINIC. Grafts, cures and harvests grown to order. Strain on the vats: "+strconv.Itoa(int(f.N*100))+"%.", swNear(a.X, a.Y, 4, 2))
		}
	}
	if swHasForce(st, "commons") != nil || (st.SocialOrder != nil && st.SocialOrder.FormID == "greencommons") {
		if center {
			swGarden(zn, r)
		}
	}
	// intermarriage: one household of two peoples
	for _, m := range st.Mixing {
		if m.A != bd.Culture && m.B != bd.Culture {
			continue
		}
		swMixHousehold(g, zn, r, bd, m)
		break
	}
	// a people born of two others
	if cu != nil && cu.Origin != nil {
		for _, a := range zn.Actors {
			if a.Tpl == "npc" && a.Role == "villager" && !a.Has("swmix") {
				a.Set("swmix")
				a.Desc += fmt.Sprintf(" The %s are a young people, born in the year %d of the %s and the %s.", cu.Name, cu.Origin.Year, w.swCultName(cu.Origin.A), w.swCultName(cu.Origin.B))
			}
		}
	}
}

func (bd *BurgData) swIndustrial(w *WorldData) bool {
	for i, n := range w.SectorNames {
		if (n == "industry" || n == "crafts" || n == "mining") && i < len(bd.Sectors) && bd.Sectors[i] >= 0.2 {
			return true
		}
	}
	return len(bd.Sectors) == 0 || bd.Pop >= 8
}

func swGarden(zn *Zone, r *Rng) {
	for k := 0; k < 8; k++ {
		x, y := r.Int(6, MW-14), r.Int(3, MH-7)
		ok := true
		for dy := 0; dy < 3 && ok; dy++ {
			for dx := 0; dx < 7; dx++ {
				t := zn.At(x+dx, y+dy)
				if t != TGrass && t != TDirt && t != TTallGrass {
					ok = false
					break
				}
			}
		}
		if !ok {
			continue
		}
		for dy := 0; dy < 3; dy++ {
			for dx := 0; dx < 7; dx++ {
				zn.Set(x+dx, y+dy, TCrop)
			}
		}
		zn.Set(x+3, y+3, TInscription)
		zn.Notes[(y+3)*MW+x+3] = "COMMONS GARDEN. Tended by all, open to all. Take what you need; plant what you can."
		return
	}
}

func swOrderFlavour(g *Game, zn *Zone, r *Rng, bd *BurgData, st *StateData, so *SocialOrderData, center bool) {
	w := g.W
	cu := w.cultByID[bd.Culture]
	near := swNear(40, 12, 22, 8)
	one := swSingular(so.Workers)
	def := g.strataDefFor(bd)
	if center {
		var txt string
		switch so.Kind {
		case "slave":
			switch {
			case so.Construct:
				txt = fmt.Sprintf("BINDING YARD of %s. %s labour under the seal of %s. Tampering with a binding is a crime against the crown.", bd.Name, titleCase(so.Workers), st.Name)
			case so.FormID == "indenture" || so.FormID == "debt" || so.FormID == "vatborn":
				txt = fmt.Sprintf("LABOUR EXCHANGE of %s. Contracts of the %s bought and sold here; terms of service posted daily.", bd.Name, so.Workers)
			default:
				txt = fmt.Sprintf("MARKET BLOCK of %s. %s sold on the first day of each month by licence of %s.", bd.Name, titleCase(so.Workers), st.Name)
			}
		case "feudal":
			switch so.FormID {
			case "company", "corpfief":
				txt = fmt.Sprintf("COMPANY STORE. Wages of %s paid in company scrip only. Outsiders pay the exchange.", bd.Name)
			case "stewardship":
				txt = fmt.Sprintf("STEWARDSHIP ROLL of %s: the %s owe their tending-days to the steward of %s.", bd.Name, so.Workers, st.Name)
			default:
				txt = fmt.Sprintf("TITHE BARN of the lord of %s. The %s owe three days of labour in seven, and a tenth of the harvest.", bd.Name, so.Workers)
			}
		case "caste":
			txt = fmt.Sprintf("ORDINANCE of %s: each to their station. The %s may not draw from the upper well nor walk the upper street after dusk.", st.Name, so.Workers)
		case "egalitarian":
			txt = fmt.Sprintf("COMMONS HALL of %s. Here the %s meet as equals; the work rota and the shares are read aloud each new moon.", bd.Name, so.Workers)
		}
		if txt != "" {
			swNote(zn, r, txt, near)
		}
	}
	switch {
	case so.Kind == "slave" && so.Construct:
		look, ok := swConstructLook[so.FormID]
		if !ok {
			look = swConstructLook["golems"]
		}
		for i, n := 0, r.Int(1, 3); i < n; i++ {
			a := g.makeNPC("villager", bd, cu, r)
			a.Role, a.Name, a.Proper, a.Title, a.Glyph, a.F, a.AI = "construct", look.name, false, so.Form, look.g, look.f, "town"
			a.Desc = look.desc
			a.Caste, a.Rank = def.at("low"), 0
			a.Set("swnorec")
			swPut(zn, r, a, near)
		}
	case so.Kind == "slave" || (so.Kind == "feudal" && so.FormID != "stewardship") || (so.Kind == "caste" && r.Chance(0.5)):
		n := r.Int(1, 2)
		if so.Kind == "slave" {
			n++
		}
		for i := 0; i < n; i++ {
			a := g.makeNPC("villager", bd, cu, r)
			a.Role, a.Title, a.AI, a.F = "unfree", one, "town", CBrown
			a.Caste, a.Rank = def.at("low"), 0
			switch so.Kind {
			case "slave":
				a.Desc = fmt.Sprintf("One of the %s of %s, owned and worked. A brand or a tag marks whose they are.", so.Workers, bd.Name)
			case "feudal":
				a.Desc = fmt.Sprintf("One of the %s of %s, bound to the land and the lord's labour-days.", so.Workers, bd.Name)
			default:
				a.Desc = fmt.Sprintf("One of the %s of %s, kept to the lowest station by law and custom.", so.Workers, bd.Name)
			}
			a.Set("swnorec")
			swPut(zn, r, a, near)
		}
	}
	// freedfolk where the old order was overthrown or the movement is strong
	if center && ((so.Prev != "" && so.Kind != "slave") || (so.Movement && r.Chance(0.5))) {
		a := g.makeNPC("villager", bd, cu, r)
		a.Title, a.Caste, a.Rank = "freed labourer", def.at("common"), 0
		if so.Prev != "" {
			a.Desc = fmt.Sprintf("Once bound under the %s; free since the %s order came in the year %d.", strings.ToLower(so.Prev), strings.ToLower(so.Name), so.Since)
		} else {
			a.Desc = fmt.Sprintf("Bought their own freedom from the %s. They wear the mark of manumission proudly.", so.Workers)
		}
		a.Set("swfreed")
		swPut(zn, r, a, near)
	}
	// the reform movement's speaker
	if center && swMovementOf(st) != "" {
		a := g.makeNPC("villager", bd, cu, r)
		a.Role, a.F, a.AI = "reformer", CGreen, "still"
		a.Title = swReformerTitle(st)
		a.Desc = fmt.Sprintf("A speaker of the %s, handing out tracts and keeping one eye on the watch.", swMovementOf(st))
		if mv := swMove(st); mv != nil && mv.Leader != "" {
			a.Desc += fmt.Sprintf(" They follow %s (support %d%%).", mv.Leader, mv.Support)
		}
		a.Set("swnorec")
		swPut(zn, r, a, swNear(40, 12, 14, 6))
	}
}

func swMove(st *StateData) *ReformMovement {
	if st != nil && st.Reforms != nil && len(st.Reforms.Movements) > 0 {
		return &st.Reforms.Movements[0]
	}
	return nil
}

// swMovementOf names the movement abroad in a realm ("" = none).
func swMovementOf(st *StateData) string {
	if st == nil {
		return ""
	}
	if so := st.SocialOrder; so != nil && so.Movement {
		switch so.Kind {
		case "slave":
			return "Abolitionists of " + st.Name
		case "caste":
			return "Society of the Open Door"
		case "feudal":
			return "Free Furrow League"
		default:
			return "Reformers of " + st.Name
		}
	}
	if mv := swMove(st); mv != nil {
		return mv.Name
	}
	return ""
}

func swReformerTitle(st *StateData) string {
	if so := st.SocialOrder; so != nil && so.Movement {
		switch so.Kind {
		case "slave":
			return "abolitionist"
		case "caste":
			return "caste reformer"
		case "feudal":
			return "land reformer"
		}
		return "reformer"
	}
	if mv := swMove(st); mv != nil {
		return "speaker for " + strings.ToLower(mv.Reform)
	}
	return "reformer"
}

func swMixHousehold(g *Game, zn *Zone, r *Rng, bd *BurgData, m MixData) {
	w := g.W
	var heads []*Actor
	for _, a := range zn.Actors {
		if a.Tpl == "npc" && a.House != "" && len(a.Kin) > 0 && a.SimPer == 0 && !a.Has("swmix") {
			heads = append(heads, a)
		}
	}
	if len(heads) == 0 {
		return
	}
	sort.Slice(heads, func(i, j int) bool { return heads[i].ID < heads[j].ID })
	h := heads[r.Int(0, len(heads)-1)]
	other := m.B
	if bd.Culture == m.B {
		other = m.A
	}
	oc := w.cultByID[other]
	members := []*Actor{h}
	for _, id := range h.Kin {
		for _, a := range zn.Actors {
			if a.ID == id {
				members = append(members, a)
			}
		}
	}
	house := h.House
	if m.House != "" {
		house = m.House
	}
	for i, a := range members {
		a.Set("swmix")
		if house != a.House {
			a.Name = personalOf(a.Name, a.House) + " " + house
			a.House = house
		}
		sp := a.Species
		osp := ""
		if oc != nil && oc.Species != "" && oc.Species != sp {
			osp = oc.Species
		}
		if i == 0 {
			a.Desc += fmt.Sprintf(" House %s joins the %s and the %s: one of the %d marriages between the two peoples here since the year %d.", house, w.swCultName(bd.Culture), w.swCultName(other), m.Unions, m.Since)
		} else {
			a.Desc += " " + w.swMixedLine(bd.Culture, other, sp, osp)
		}
	}
}

func swOutskirts(g *Game, zn *Zone, r *Rng, bd *BurgData) {
	w := g.W
	slots := swFreeSlots(g, bd)
	at := [2]int{zn.ZX, zn.ZY}
	// refugee camps: a haven's towns nearest the persecuting realm
	for _, ps := range swCampsFor(g, bd) {
		if len(slots) == 0 || slots[0] != at {
			break
		}
		swCamp(g, zn, r, bd, ps.st, ps.p)
		return
	}
	// the captive of a movement quest
	for _, q := range swSt(g).Quests {
		if q.Kind == "captive" && !q.Closed && q.Zone == zn.Key && !q.Placed {
			swPlaceCaptive(g, zn, q)
		}
	}
	_ = w
}

type swPersAt struct {
	st *StateData
	p  *PersecutionData
}

// swCampsFor: persecutions whose refugees camp outside this town (up to two haven towns each, nearest
// the persecuting realm's capital).
func swCampsFor(g *Game, bd *BurgData) []swPersAt {
	w := g.W
	var out []swPersAt
	for i := range w.States {
		st := &w.States[i]
		if !st.Alive {
			continue
		}
		for _, p := range swActivePers(st) {
			if p.Haven <= 0 || p.Haven != bd.State {
				continue
			}
			for _, t := range swHavenTowns(w, st, p.Haven) {
				if t == bd.ID {
					out = append(out, swPersAt{st, p})
				}
			}
		}
	}
	return out
}

func swHavenTowns(w *WorldData, from *StateData, haven int) []int {
	cx, cy := w.W/2, w.H/2
	if c := w.burgByID[from.Capital]; c != nil {
		cx, cy = c.P%w.W, c.P/w.W
	}
	var bs []*BurgData
	for i := range w.Burgs {
		b := &w.Burgs[i]
		if b.Alive && b.State == haven && b.Layer == "surface" {
			bs = append(bs, b)
		}
	}
	d := func(b *BurgData) int { return maxi(abs(b.P%w.W-cx), abs(b.P/w.W-cy)) }
	sort.Slice(bs, func(i, j int) bool {
		if d(bs[i]) != d(bs[j]) {
			return d(bs[i]) < d(bs[j])
		}
		return bs[i].ID < bs[j].ID
	})
	var out []int
	for i := 0; i < len(bs) && i < 2; i++ {
		out = append(out, bs[i].ID)
	}
	return out
}

func swCamp(g *Game, zn *Zone, r *Rng, bd *BurgData, from *StateData, p *PersecutionData) {
	w := g.W
	cu := w.cultByID[bd.Culture]
	if p.Axis == "culture" {
		if c := w.cultByID[p.Target]; c != nil {
			cu = c
		}
	}
	cx, cy := 40, 12
	for i := 0; i < 4; i++ {
		x, y := cx-16+i*8+r.Int(-1, 1), cy-5+r.Int(0, 1)*7
		g.building(zn, x, y, 5, 4, TWall, TFloor)
	}
	g.clearAround(zn, cx, cy, 2)
	zn.Set(cx, cy, TBrazier)
	zn.Name = "Refugee camp outside " + bd.Name
	swNote(zn, r, fmt.Sprintf("Scratched on a board: WE ARE THE %s, DRIVEN FROM %s BY %s IN THE YEAR %d. FOOD AND BLANKETS WANTED.",
		strings.ToUpper(swPeople(p)), strings.ToUpper(from.Name), strings.ToUpper(p.Name), p.Since), swNear(cx, cy, 6, 3))
	n := r.Int(4, 6)
	for i := 0; i < n; i++ {
		a := g.makeNPC("villager", bd, cu, r)
		a.Role, a.Title, a.AI, a.F = "refugee", "refugee", "town", CBrown
		if p.Axis == "faith" {
			a.Religion = p.Target
		}
		a.Faction = stateFaction(bd.State)
		a.State = bd.State
		a.Desc = fmt.Sprintf("Fled the %s in %s in the year %d with what they could carry.", p.Name, from.Name, p.Since)
		a.Flags = map[string]bool{"swnorec": true, "swref:" + strconv.Itoa(from.ID): true}
		if i == 0 {
			a.Title = "camp elder"
			a.Set("swelder")
			a.Desc += " The others look to them."
		}
		swPut(zn, r, a, swNear(cx, cy, 14, 6))
	}
}

func swFalloutAt(w *WorldData, p int) *FalloutData {
	if w.Forces == nil {
		return nil
	}
	for i := range w.Forces.Fallout {
		for _, q := range w.Forces.Fallout[i].P {
			if q == p {
				return &w.Forces.Fallout[i]
			}
		}
	}
	return nil
}

func swFalloutCore(fz *FalloutData, p int) bool {
	for _, q := range fz.Core {
		if q == p {
			return true
		}
	}
	return false
}

func swFalloutZone(zn *Zone, r *Rng, fz *FalloutData) {
	for i, t := range zn.T {
		switch t {
		case TGrass, TTallGrass, TMoss, TBush, TTree:
			if r.Chance(0.6) {
				zn.T[i] = TAsh
			} else if r.Chance(0.3) {
				zn.T[i] = TDeadTree
			}
		}
	}
	swNote(zn, r, fmt.Sprintf("A buckled warning sign: FALLOUT ZONE. Ash of the strike on %s. Do not drink the water. Do not linger.", nonEmpty(fz.Town, "this land")), nil)
}

// swLeyAt: a ley node at this parasang (end 0 = a, 1 = b).
func swLeyAt(w *WorldData, p int) (*LeyData, int) {
	if w.Forces == nil {
		return nil, 0
	}
	for i := range w.Forces.Leys {
		l := &w.Forces.Leys[i]
		if l.A == p && (l.LA == "surface" || l.LA == "") {
			return l, 0
		}
		if l.B == p && (l.LB == "surface" || l.LB == "") {
			return l, 1
		}
	}
	return nil, 0
}

func swLeyStone(g *Game, zn *Zone, r *Rng, l *LeyData, end int) {
	here, there := l.AN, l.BN
	if end == 1 {
		here, there = l.BN, l.AN
	}
	x, y, ok := zn.FreeSpot(r, swNear(40, 12, 20, 6))
	if !ok {
		return
	}
	for _, d := range [][2]int{{-2, 0}, {2, 0}, {0, -1}, {0, 1}} {
		if zn.Walkable(x+d[0], y+d[1]) && zn.ActorAt(x+d[0], y+d[1]) == nil {
			zn.Set(x+d[0], y+d[1], TPillar)
		}
	}
	zn.Set(x, y, TInscription)
	zn.Notes[y*MW+x] = fmt.Sprintf("A ley stone, humming. The line runs from %s to %s. Power gathers here: your gifts recover swiftly while you stand in this place.", nonEmpty(here, "here"), nonEmpty(there, "afar"))
}

func swPlaceCaptive(g *Game, zn *Zone, q *swQuest) {
	w := g.W
	bd := w.burgByID[q.Burg]
	if bd == nil {
		return
	}
	q.Placed = true
	r := NewRng(fmt.Sprintf("%s:swcap:%d", w.Seed, q.ID))
	so := w.swOrder(bd.State)
	workers := "bondsfolk"
	if so != nil {
		workers = so.Workers
	}
	a := g.makeNPC("villager", bd, w.cultByID[bd.Culture], r)
	a.Role, a.Title, a.AI, a.F = "captive", "chained "+swSingular(workers), "still", CBrown
	a.Desc = "Chained to a post, waiting to be marched to the market. They watch you without hope."
	a.Quest = q.ID
	a.Set("swnorec")
	if !swPut(zn, r, a, swNear(40, 12, 16, 6)) {
		return
	}
	for i := 0; i < 2; i++ {
		m := g.Spawn("bandit", zn.Level, r)
		m.Name, m.Proper, m.Desc = "overseer", false, "A coffle overseer with a whip and a ring of keys."
		swPut(zn, r, m, swNear(a.X, a.Y, 6, 3))
	}
}

// ---- play ----

func swEnter(g *Game, zn *Zone) {
	st := swSt(g)
	for _, q := range st.Quests {
		if q.Closed || q.Stage > 0 {
			continue
		}
		switch q.Kind {
		case "captive":
			if q.Zone == zn.Key && !q.Placed {
				swPlaceCaptive(g, zn, q)
			}
		case "pamphlets":
			if zn.Burg == q.Burg && zn.Kind == "town" {
				for _, it := range g.P.Inv {
					if it.Quest == q.ID {
						g.removeInv(it)
						q.Stage = 1
						g.Msg("{{G|You paste the pamphlets on doors and pass them hand to hand. Return to the movement.}}")
						break
					}
				}
			}
		}
	}
	if w := g.W; zn.Z == 0 && w.Forces != nil && w.In(zn.WX, zn.WY) {
		if fz := swFalloutAt(w, w.Idx(zn.WX, zn.WY)); fz != nil {
			g.Msg("{{G|The air tastes of metal. This is a fallout zone.}}")
		}
	}
	if g.swSmogHere() {
		g.Msg("{{K|Smog hangs over the works; your eyes sting.}}")
	}
}

func (g *Game) swSmogHere() bool {
	zn := g.Cur
	if zn == nil || zn.Z != 0 || zn.Kind != "town" || g.W.Forces == nil {
		return false
	}
	for _, b := range g.W.Forces.Smog {
		if b == zn.Burg {
			return true
		}
	}
	return false
}

// swSight caps sight in smog (ai.go recompute).
func (g *Game) swSight(radius int) int {
	if g.swSmogHere() && !roofed(g.Cur.At(g.P.X, g.P.Y)) {
		return mini(radius, 12)
	}
	return radius
}

func swTurn(g *Game) {
	zn, p := g.Cur, g.P
	if zn == nil || g.Dead || g.Mode == "world" || zn.Z != 0 {
		return
	}
	w := g.W
	if g.swSmogHere() && g.Turn%30 == 0 {
		g.Thirst++
		if g.R.Chance(0.3) {
			g.Msg("{{K|You cough on the smog.}}")
		}
	}
	if w.Forces == nil || !w.In(zn.WX, zn.WY) {
		return
	}
	pp := w.Idx(zn.WX, zn.WY)
	if fz := swFalloutAt(w, pp); fz != nil && g.Turn%20 == 0 {
		core := swFalloutCore(fz, pp)
		if g.R.Int(1, 20)+p.M(TOU) < map[bool]int{true: 15, false: 10}[core] {
			g.Msg("{{G|Rad sickness: your gums bleed and your stomach turns.}}")
			p.AddSt("poisoned", map[bool]int{true: 8, false: 4}[core])
		}
	}
	if l, _ := swLeyAt(w, pp); l != nil && zn.ZX == 1 && zn.ZY == 1 {
		for k, v := range p.Cool {
			if v > 0 {
				p.Cool[k] = v - 1 // a second tick: powers recharge twice as fast on a ley node
			}
		}
	}
}

func swSide(g *Game) []string {
	zn := g.Cur
	if zn == nil || zn.Z != 0 || g.W == nil {
		return nil
	}
	w := g.W
	var out []string
	if g.swSmogHere() {
		out = append(out, "{{K|Smog}}")
	}
	if w.Forces != nil && w.In(zn.WX, zn.WY) {
		pp := w.Idx(zn.WX, zn.WY)
		if swFalloutAt(w, pp) != nil {
			out = append(out, "{{G|Fallout zone}}")
		}
		if l, _ := swLeyAt(w, pp); l != nil && zn.ZX == 1 && zn.ZY == 1 {
			out = append(out, "{{M|Ley node}}")
		}
	}
	return out
}

// swGreeting: unfree, constructs, refugees, pickets, and guards of a realm that persecutes the player.
func (g *Game) swGreeting(a *Actor) string {
	r := NewRng(fmt.Sprintf("swgreet:%d:%d", a.ID, g.Turn/600))
	switch a.Role {
	case "construct":
		return "It does not answer. It works, and the binding hums."
	case "unfree":
		return "They keep their eyes down. \"" + Pick(r, []string{"I may not stop, stranger. The overseer counts.", "Speak quickly. If I am seen idle, I am beaten.", "Whose are you? ...No one's? Then you are lucky."}) + "\""
	case "captive":
		return "\"Please. The keys are on the overseers. Please.\""
	case "refugee":
		return "\"" + Pick(r, []string{"We left everything. Everything.", "Have you come from the border? Is it still burning?", "They let us camp here, for now. For now."}) + "\""
	case "striker":
		return "\"" + Pick(r, []string{"Don't cross the line, friend.", "We're out till the bosses pay bread wages.", "Solidarity! Or move along."}) + "\""
	case "reformer":
		st := g.W.stateByID[a.State]
		return fmt.Sprintf("\"The %s welcomes every ear. Things here can change: they must.\"", nonEmpty(swMovementOf(st), "movement"))
	}
	if a.Has("swcorp") {
		return "\"Badge or business, citizen. This is corporate property.\""
	}
	if a.Has("swfreed") {
		return "\"Free, and nobody's. Say it with me: nobody's.\""
	}
	if ps := g.swPersecutesMe(a.State); ps != nil && (a.Role == "guard" || isTrader(a)) {
		if a.Role == "guard" {
			return fmt.Sprintf("The guard's hand drops to a weapon. \"One of the %s? Your kind are not welcome in %s.\"", swPeople(ps), g.W.StateName(a.State))
		}
		return fmt.Sprintf("\"I'll sell to %s, but you'll pay for the risk.\"", swPeople(ps))
	}
	// admired dress of a prestigious people
	if a.Tpl == "npc" && r.Chance(0.35) {
		for _, it := range g.wornAttire() {
			for i := range g.W.Cultures {
				cu := &g.W.Cultures[i]
				if cu.Prestige != nil && cu.Prestige.Rank > 0 && cu.Prestige.Rank <= 3 && cu.Sphere == it.Region {
					return fmt.Sprintf("\"Is that %s? The %s set the fashion everywhere these days.\"", it.ShownName(), cu.Name)
				}
			}
		}
	}
	return ""
}

// swPriceMod multiplies a buy price (buy) or a sell price offered to the player (!buy).
func (g *Game) swPriceMod(a *Actor, it *Item, buy bool) float64 {
	if a == nil || g.W == nil {
		return 1
	}
	st := g.W.stateByID[a.State]
	if st == nil {
		return 1
	}
	m := 1.0
	if g.swPersecutesMe(a.State) != nil {
		if buy {
			m *= 1.3
		} else {
			m *= 0.75
		}
	}
	food := it != nil && (it.Kind == "food" || it.Kind == "ingredient" || it.Kind == "dish")
	if buy && food {
		if swEnacted(st, "land") {
			m *= 0.85
		}
		if f := st.Famine; f != nil && g.W.Year-f.Year <= 15 && (f.Choice == "hoard" || f.Choice == "requisition") {
			m *= 1.2
		}
	}
	if buy && st.SocialOrder != nil && (st.SocialOrder.FormID == "company" || st.SocialOrder.FormID == "corpfief") {
		m *= 1.1 // the scrip exchange
	}
	if buy && !food && g.Cur != nil {
		if bd := g.W.burgByID[g.Cur.Burg]; bd != nil {
			if cu := g.W.cultByID[bd.Culture]; cu != nil && cu.Prestige != nil && cu.Prestige.Mult > 0 {
				m *= 1 + (cu.Prestige.Mult-1)/2 // admired peoples' goods fetch more
			}
		}
	}
	return m
}

// swGossip: a line about reforms, famine, persecution, order or forces of this realm (about 40% of the time).
func (g *Game) swGossip(a *Actor) string {
	w := g.W
	if g.Cur == nil || !w.In(g.Cur.WX, g.Cur.WY) {
		return ""
	}
	st := w.stateByID[w.State[w.Idx(g.Cur.WX, g.Cur.WY)]]
	if st == nil || !st.Alive {
		return ""
	}
	r := NewRng(fmt.Sprintf("swgos:%d:%d", a.ID, g.Turn/600))
	if !r.Chance(0.4) {
		return ""
	}
	lines := g.swRealmLines(st)
	if len(lines) == 0 {
		return ""
	}
	return "\"" + Pick(r, lines) + "\""
}

func (g *Game) swRealmLines(st *StateData) []string {
	w := g.W
	var out []string
	if so := st.SocialOrder; so != nil {
		switch so.Kind {
		case "slave":
			out = append(out, fmt.Sprintf("Here the %s do the work. %s since the year %d, and the masters like it so.", so.Workers, so.Form, so.Since))
		case "feudal":
			out = append(out, fmt.Sprintf("The %s owe the lords their days. %s, as it has been since the year %d.", so.Workers, so.Form, so.Since))
		case "caste":
			out = append(out, fmt.Sprintf("Every soul in %s has a station. The %s keep to theirs.", st.Name, so.Workers))
		case "egalitarian":
			out = append(out, fmt.Sprintf("No masters in %s: the %s share the work and the harvest.", st.Name, so.Workers))
		}
		if so.Prev != "" {
			out = append(out, fmt.Sprintf("The old %s order is gone; the elders still grumble about it.", strings.ToLower(so.Prev)))
		}
		if so.Movement {
			out = append(out, fmt.Sprintf("The %s grow bolder; the crown's patience is thin.", swMovementOf(st)))
		}
	}
	if rf := st.Reforms; rf != nil {
		for _, k := range rf.Enacted {
			if l, ok := swReformLine[k]; ok {
				out = append(out, fmt.Sprintf(l, st.Name))
			}
		}
		for _, mv := range rf.Movements {
			out = append(out, fmt.Sprintf("The %s, led by %s, want %s. About %d in a hundred are with them.", mv.Name, mv.Leader, strings.ToLower(mv.Reform), mv.Support))
		}
	}
	if f := st.Famine; f != nil {
		out = append(out, fmt.Sprintf("In the famine of the year %d, %s %s. %s", f.Year, nonEmpty(f.Ruler, "the council"), f.Text, swFamineMood[f.Choice]))
	}
	for _, p := range swActivePers(st) {
		s := fmt.Sprintf("The %s against the %s goes on, since the year %d.", p.Name, swPeople(p), p.Since)
		if p.Haven > 0 {
			s += fmt.Sprintf(" Many fled to %s.", w.StateName(p.Haven))
		}
		out = append(out, s)
	}
	for _, f := range st.Forces {
		if l, ok := swForceLine[f.Kind]; ok {
			out = append(out, fmt.Sprintf(l, st.Name, f.Note))
		}
	}
	for _, m := range st.Mixing {
		out = append(out, fmt.Sprintf("The %s and the %s marry each other now; %d weddings at least.", w.swCultName(m.A), w.swCultName(m.B), m.Unions))
	}
	// refugees taken in here
	for i := range w.States {
		o := &w.States[i]
		for _, p := range swActivePers(o) {
			if p.Haven == st.ID {
				out = append(out, fmt.Sprintf("Refugees of the %s camp outside our towns, fleeing %s.", swPeople(p), o.Name))
			}
		}
	}
	return out
}

var swReformLine = map[string]string{
	"legal":      "%s has written law and courts now. Even a lord can be summoned.",
	"temperance": "Drink is taxed to the bone in %s since the temperance laws. The taverns sulk.",
	"franchise":  "Commoners of %s have a voice in council since the franchise. Imagine that.",
	"land":       "The land reform gave the fields of %s to those who work them. Bread is cheaper for it.",
	"tolerance":  "Every altar may stand in %s by the edict of tolerance.",
}

var swFamineMood = map[string]string{
	"relief":      "People still bless them for it.",
	"hoard":       "Nobody has forgotten who ate and who starved.",
	"requisition": "The army ate; the villages did not.",
	"scapegoat":   "Then they found someone to blame, as they always do.",
}

var swForceLine = map[string]string{
	"smog":       "The works of %s choke the sky (%s).",
	"strikes":    "The workers of %s walked out (%s). The bosses have long memories.",
	"wareconomy": "In %s every factory makes shells now: %s.",
	"armsrace":   "%s builds weapons it prays never to use: %s.",
	"fallout":    "Parts of %s are ash and poison: %s. Stay out of the grey country.",
	"megacorps":  "In %s the corporations hold the leash: %s.",
	"commons":    "%s keeps green commons: %s.",
	"leylines":   "The ley lines run through %s: %s. Mages come to drink from them.",
	"genevats":   "The vats of %s feed us all (%s). If they sicken, so do we.",
}

// swNews: lines for the world news (worldsim.go wsNewsLines) about this realm and its neighbours.
func (g *Game) swNews() []wsLine {
	w := g.W
	if g.Cur == nil || !w.In(g.Cur.WX, g.Cur.WY) {
		return nil
	}
	here := w.stateByID[w.State[w.Idx(g.Cur.WX, g.Cur.WY)]]
	if here == nil {
		return nil
	}
	var out []wsLine
	cap := func(st *StateData) int {
		if b := w.burgByID[st.Capital]; b != nil {
			return b.P
		}
		return -1
	}
	add := func(st *StateData) {
		if st == nil || !st.Alive {
			return
		}
		if mv := swMove(st); mv != nil {
			out = append(out, wsLine{cap(st), fmt.Sprintf("{{G|Reform}}: the %s press %s for %s (%d%% support).", mv.Name, st.Name, strings.ToLower(mv.Reform), mv.Support)})
		}
		if so := st.SocialOrder; so != nil && so.Movement {
			out = append(out, wsLine{cap(st), fmt.Sprintf("{{G|Unrest}}: the %s of %s stir against the %s order.", so.Workers, st.Name, strings.ToLower(so.Name))})
		}
		for _, p := range swActivePers(st) {
			out = append(out, wsLine{cap(st), fmt.Sprintf("{{R|Persecution}}: %s of the %s in %s.", p.Name, swPeople(p), st.Name)})
		}
		if f := st.Famine; f != nil && w.Year-f.Year <= 10 {
			out = append(out, wsLine{cap(st), fmt.Sprintf("{{O|Hunger}}: in %s's famine of the year %d, %s %s.", st.Name, f.Year, nonEmpty(f.Ruler, "the council"), f.Text)})
		}
	}
	add(here)
	for id := range here.Relations {
		if st := w.stateByID[id]; st != nil && id != here.ID {
			if b := w.burgByID[st.Capital]; b != nil && abs(b.P%w.W-g.Cur.WX) <= 12 && abs(b.P/w.W-g.Cur.WY) <= 9 {
				add(st)
			}
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Text < out[j].Text })
	return out
}

func (g *Game) swFactionName(f string) string {
	if id, err := strconv.Atoi(strings.TrimPrefix(f, "sw:mv:")); err == nil {
		if n := swMovementOf(g.W.stateByID[id]); n != "" {
			return n
		}
		return "Reformers of " + g.W.StateName(id)
	}
	return titleCase(f)
}

func swMvFaction(state int) string { return "sw:mv:" + strconv.Itoa(state) }

// ---- talk ----

func swFood(g *Game) int {
	n := 0
	for _, it := range g.P.Inv {
		if it.Kind == "food" {
			n += maxi(1, it.Count)
		}
	}
	return n
}

func swTakeFood(g *Game, n int) {
	for n > 0 {
		var f *Item
		for _, it := range g.P.Inv {
			if it.Kind == "food" {
				f = it
				break
			}
		}
		if f == nil {
			return
		}
		if f.Count > 1 {
			f.Count--
		} else {
			g.removeInv(f)
		}
		n--
	}
}

func swTalk(g *Game, a *Actor) []Option {
	st := swSt(g)
	var out []Option
	add := func(t string, f func()) { out = append(out, Option{t, f}) }
	switch a.Role {
	case "unfree":
		if !st.Helped[a.ID] {
			add(fmt.Sprintf("{{G|Slip them food and coin.}} {{K|(5 %s)}}", g.Sk.Currency), func() { g.swHelpUnfree(a) })
		}
	case "captive":
		for _, q := range st.Quests {
			if q.ID == a.Quest && q.Stage == 0 && !q.Closed {
				q := q
				add("{{G|Break their chains.}}", func() { g.swFreeCaptive(a, q) })
			}
		}
	case "reformer":
		add("Tell me about your cause.", func() { g.talkMenu(g.swCause(a)) })
		if qid, ok := st.Asked[a.ID]; ok {
			if q := st.Quests[qid]; q != nil && q.Stage == 1 && !q.Closed {
				add("{{G|It is done.}}", func() { g.swTurnIn(a, q) })
			}
		} else {
			add("How can I help the cause?", func() { g.swOffer(a) })
		}
	case "refugee":
		if a.Has("swelder") {
			if qid, ok := st.Asked[a.ID]; !ok {
				add("How can I help?", func() { g.swOfferFeed(a) })
			} else if q := st.Quests[qid]; q != nil && !q.Closed && swFood(g) >= 3 {
				add("{{G|Share food with the camp. (3 rations)}}", func() { g.swFeed(a, q) })
			}
		}
	}
	return out
}

func (g *Game) swCause(a *Actor) string {
	w := g.W
	st := w.stateByID[a.State]
	if st == nil {
		return "\"Justice. That is all.\""
	}
	s := ""
	if so := st.SocialOrder; so != nil && so.Movement {
		s = fmt.Sprintf("\"%s holds the %s under %s. We say no one is born to be owned or bound. Pressure grows: %d in a hundred would see it ended.\"", st.Name, so.Workers, strings.ToLower(so.Form), so.Pressure)
	}
	if mv := swMove(st); mv != nil {
		if s != "" {
			s += "\n\n"
		}
		s += fmt.Sprintf("\"The %s want %s. %s leads us; %d in a hundred are with us.\"", mv.Name, strings.ToLower(mv.Reform), mv.Leader, mv.Support)
	}
	return nonEmpty(s, "\"Justice. That is all.\"")
}

func (g *Game) swHelpUnfree(a *Actor) {
	if g.P.Gold < 5 {
		g.talkMenu("\"You have nothing to give. But thank you for asking.\"")
		return
	}
	g.P.Gold -= 5
	swSt(g).Helped[a.ID] = true
	g.AddRep(swMvFaction(a.State), 20, " for kindness to the "+nonEmpty(a.Title, "unfree"))
	g.AddRep(stateFaction(a.State), -5, " for meddling with the crown's labour")
	g.talkMenu("They hide the coin fast. \"May you never wear a collar.\"")
}

func (g *Game) swOffer(a *Actor) {
	w := g.W
	st := w.stateByID[a.State]
	bd := w.burgByID[g.Cur.Burg]
	if st == nil || bd == nil {
		g.talkMenu("\"Not now. The watch is close.\"")
		return
	}
	s := swSt(g)
	q := &Quest{ID: g.newID(), Kind: "sw", Giver: a.Name, GiverID: a.ID, Faction: swMvFaction(st.ID), HomeKey: g.Cur.Key, Reward: 50 + g.P.Level*20}
	sq := &swQuest{ID: q.ID, State: st.ID, Giver: a.ID}
	so := st.SocialOrder
	slots := swFreeSlots(g, bd)
	captive := so != nil && so.Movement && !so.Construct && (so.Kind == "slave" || so.Kind == "feudal") && len(slots) > 1 && HashStr(fmt.Sprint(w.Seed, a.ID))%2 == 0
	if captive {
		sl := slots[1]
		sq.Kind, sq.Burg, sq.Zone = "captive", bd.ID, ZoneKey(g.Cur.WX, g.Cur.WY, sl[0], sl[1], 0)
		q.Title = "Free the captive outside " + bd.Name
		q.Text = fmt.Sprintf("\"A coffle waits in the fields %s of town, to be sold come morning. Break one of the %s loose. The overseers will not ask nicely.\"", bhThird(sl[0], sl[1]), so.Workers)
		q.TP, q.Target = bd.P, sq.Zone
	} else {
		var tgt *BurgData
		bestD := 1 << 30
		for i := range w.Burgs {
			b := &w.Burgs[i]
			if !b.Alive || b.ID == bd.ID || b.Layer != "surface" || b.State != st.ID {
				continue
			}
			if d := maxi(abs(b.P%w.W-bd.P%w.W), abs(b.P/w.W-bd.P/w.W)); d < bestD {
				tgt, bestD = b, d
			}
		}
		if tgt == nil {
			tgt = bd
		}
		sq.Kind, sq.Burg = "pamphlets", tgt.ID
		mv := nonEmpty(swMovementOf(st), "movement")
		q.Title = "Spread the word in " + tgt.Name
		q.Text = fmt.Sprintf("\"Carry these tracts of the %s to %s and see them read. Mind the watch: they burn what they find.\"", mv, tgt.Name)
		q.TP = tgt.P
		g.give(&Item{ID: g.newID(), Kind: "letter", Name: "bundle of " + mv + " pamphlets", Glyph: '■', F: CGreen, Quest: q.ID})
		if tgt == bd {
			q.Text = fmt.Sprintf("\"Take these tracts of the %s to the districts of this town. Walk out and come back in through another gate; they must be seen everywhere.\"", mv)
		}
	}
	s.Quests[q.ID] = sq
	s.Asked[a.ID] = q.ID
	g.Quests = append(g.Quests, q)
	g.Known[q.TP] = true
	g.Msg("{{W|New quest: " + q.Title + "}}")
	g.talkMenu(q.Text + "\n\n{{K|(Quest added to your journal.)}}")
}

func (g *Game) swFreeCaptive(a *Actor, q *swQuest) {
	for _, m := range g.Cur.Actors {
		if m.HP > 0 && m.Name == "overseer" && g.Hostile(m) && maxi(abs(m.X-a.X), abs(m.Y-a.Y)) <= 6 {
			g.talkMenu("\"Not with the overseers watching! Deal with them first.\"")
			return
		}
	}
	q.Stage = 1
	swSt(g).Helped[a.ID] = true
	g.AddRep(swMvFaction(q.State), 60, " for freeing a captive")
	g.AddRep(stateFaction(q.State), -40, " for stealing the crown's property")
	for i, m := range g.Cur.Actors {
		if m == a {
			g.Cur.Actors = append(g.Cur.Actors[:i], g.Cur.Actors[i+1:]...)
			break
		}
	}
	g.Mode, g.Talk = "main", nil
	g.Msg("{{G|The chains fall. The captive is gone into the tall grass before you can say a word. Return to the movement.}}")
}

func (g *Game) swTurnIn(a *Actor, q *swQuest) {
	q.Closed = true
	for _, gq := range g.Quests {
		if gq.ID == q.ID {
			gq.Done, gq.Closed = true, true
			g.P.Gold += gq.Reward
			g.gainXP(gq.Reward)
			g.deed("quest", fmt.Sprintf("The wanderer %s worked for the %s: %s.", g.P.Name, g.FactionName(swMvFaction(q.State)), gq.Title), 0, 0)
		}
	}
	g.AddRep(swMvFaction(q.State), 100, " for serving the cause")
	g.AddRep(stateFaction(q.State), -25, " for consorting with agitators")
	g.talkMenu("\"Every hand like yours brings the day nearer.\" They press a purse on you.")
}

func (g *Game) swOfferFeed(a *Actor) {
	s := swSt(g)
	q := &Quest{ID: g.newID(), Kind: "sw", Giver: a.Name, GiverID: a.ID, Faction: a.Faction, HomeKey: g.Cur.Key, Reward: 30 + g.P.Level*10, TP: g.W.Idx(g.Cur.WX, g.Cur.WY)}
	q.Title = "Feed the refugees at " + g.Cur.Name
	q.Text = "\"The children are hungry and the town gives little. Three rations would keep us a week.\""
	s.Quests[q.ID] = &swQuest{ID: q.ID, Kind: "feed", State: a.State, Giver: a.ID}
	s.Asked[a.ID] = q.ID
	g.Quests = append(g.Quests, q)
	g.Msg("{{W|New quest: " + q.Title + "}}")
	g.talkMenu(q.Text + "\n\n{{K|(Quest added to your journal.)}}")
}

func (g *Game) swFeed(a *Actor, q *swQuest) {
	swTakeFood(g, 3)
	q.Stage, q.Closed = 1, true
	for _, gq := range g.Quests {
		if gq.ID == q.ID {
			gq.Done, gq.Closed = true, true
			g.gainXP(gq.Reward)
			g.deed("quest", fmt.Sprintf("The wanderer %s fed the refugees at %s.", g.P.Name, g.Cur.Name), 0, 0)
		}
	}
	g.AddRep(a.Faction, 60, " for feeding the refugees")
	if a.Religion > 0 {
		g.AddRep(fmt.Sprintf("faith:%d", a.Religion), 40, " for charity to the faithful")
	}
	g.talkMenu("The camp gathers round the fire. \"We will remember the one who fed us.\"")
}
