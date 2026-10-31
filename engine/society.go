package engine

import (
	"fmt"
	"sort"
	"strings"
)

// Society: strata per culture/ideology/genre, households with shared houses, mixed species, and the
// hierarchy helpers (standing, price modifier, "Who are you?" line, greetings).

// strataDef is a ladder of social strata, top to bottom, plus where each kind of person sits in it.
type strataDef struct {
	list []string
	slot map[string]int // ruler noble clergy scholar warrior merchant artisan common low outcast
}

func (d *strataDef) at(slot string) string {
	i, ok := d.slot[slot]
	if !ok {
		i = d.slot["common"]
	}
	return d.list[clampi(i, 0, len(d.list)-1)]
}

func (d *strataDef) index(name string) int {
	for i, s := range d.list {
		if s == name {
			return i
		}
	}
	return -1
}

func (d *strataDef) high(name string) bool {
	i := d.index(name)
	return i >= 0 && i <= maxi(d.slot["ruler"], d.slot["noble"])
}

func (d *strataDef) outcast(name string) bool {
	return name != "" && d.index(name) == d.slot["outcast"]
}

var (
	defFeudal = &strataDef{
		list: []string{"royalty", "nobility", "clergy", "men-at-arms", "burghers", "commoners", "serfs", "outcasts"},
		slot: map[string]int{"ruler": 0, "noble": 1, "clergy": 2, "scholar": 2, "warrior": 3, "merchant": 4, "artisan": 4, "common": 5, "low": 6, "outcast": 7},
	}
	defCaste = &strataDef{
		list: []string{"crowned lineage", "twice-anointed", "sworn-blades", "tally-folk", "toilers", "unclean-born"},
		slot: map[string]int{"ruler": 0, "noble": 2, "clergy": 1, "scholar": 1, "warrior": 2, "merchant": 3, "artisan": 3, "common": 4, "low": 4, "outcast": 5},
	}
	defSinic = &strataDef{
		list: []string{"imperial house", "gentry-scholars", "warrior retainers", "farmers", "artisans", "merchants", "outcastes"},
		slot: map[string]int{"ruler": 0, "noble": 1, "clergy": 1, "scholar": 1, "warrior": 2, "common": 3, "low": 3, "artisan": 4, "merchant": 5, "outcast": 6},
	}
	defSteppe = &strataDef{
		list: []string{"khan's kin", "noyans", "free herders", "bondsmen", "outcasts"},
		slot: map[string]int{"ruler": 0, "noble": 1, "clergy": 1, "scholar": 1, "warrior": 2, "merchant": 2, "common": 2, "artisan": 3, "low": 3, "outcast": 4},
	}
	defRepublic = &strataDef{
		list: []string{"councillors", "guild members", "citizens", "bound laborers", "drifters"},
		slot: map[string]int{"ruler": 0, "noble": 0, "scholar": 1, "merchant": 1, "artisan": 1, "clergy": 2, "warrior": 2, "common": 2, "low": 3, "outcast": 4},
	}
	defTechno = &strataDef{
		list: []string{"directors", "engineers", "technicians", "citizens", "unlicensed"},
		slot: map[string]int{"ruler": 0, "noble": 0, "scholar": 1, "artisan": 1, "merchant": 2, "warrior": 2, "clergy": 3, "common": 3, "low": 3, "outcast": 4},
	}
	defCommunal = &strataDef{
		list: []string{"circle-elders", "members", "hands", "strays"},
		slot: map[string]int{"ruler": 0, "noble": 0, "clergy": 0, "scholar": 1, "merchant": 1, "artisan": 1, "warrior": 1, "common": 1, "low": 2, "outcast": 3},
	}
	defOligarchy = &strataDef{
		list: []string{"oligarchs", "guild masters", "freemen", "laborers", "outcasts"},
		slot: map[string]int{"ruler": 0, "noble": 0, "scholar": 1, "merchant": 1, "artisan": 1, "clergy": 2, "warrior": 2, "common": 2, "low": 3, "outcast": 4},
	}
	defTheocracy = &strataDef{
		list: []string{"hierarchs", "clergy", "the faithful", "bondservants", "the unclean"},
		slot: map[string]int{"ruler": 0, "noble": 0, "clergy": 1, "scholar": 1, "warrior": 2, "merchant": 2, "artisan": 2, "common": 2, "low": 3, "outcast": 4},
	}
	defCyber = &strataDef{
		list: []string{"corporate execs", "salarymen", "street folk", "edgerunners", "zeroes"},
		slot: map[string]int{"ruler": 0, "noble": 0, "scholar": 1, "merchant": 1, "warrior": 1, "common": 1, "clergy": 2, "low": 2, "artisan": 3, "outcast": 4},
	}
	defBio = &strataDef{
		list: []string{"gene-elite", "splices", "brood-kin", "vat-born", "culls"},
		slot: map[string]int{"ruler": 0, "noble": 0, "scholar": 1, "clergy": 1, "merchant": 1, "artisan": 1, "warrior": 1, "common": 2, "low": 3, "outcast": 4},
	}
	defSolar = &strataDef{
		list: []string{"stewards", "members", "guests", "drifters"},
		slot: map[string]int{"ruler": 0, "noble": 0, "scholar": 0, "clergy": 1, "merchant": 1, "artisan": 1, "warrior": 1, "common": 1, "low": 2, "outcast": 3},
	}
)

func (g *Game) strataDefFor(bd *BurgData) *strataDef {
	switch g.Genre {
	case "cyberpunk":
		return defCyber
	case "biopunk":
		return defBio
	case "solarpunk":
		return defSolar
	}
	var cu *CultureData
	var st *StateData
	var rel *ReligionData
	if bd != nil && g.W != nil {
		cu = g.W.cultByID[bd.Culture]
		st = g.W.stateByID[bd.State]
		if n := len(g.W.Religion); n > 0 {
			rel = g.W.relByID[g.W.Religion[clampi(bd.P, 0, n-1)]]
		}
	}
	if rel != nil && rel.Doctrine == "caste" {
		return defCaste
	}
	if st != nil {
		switch st.Ideology {
		case "republic":
			return defRepublic
		case "technocracy":
			return defTechno
		case "communal":
			return defCommunal
		case "oligarchy":
			return defOligarchy
		case "theocracy":
			return defTheocracy
		}
	}
	if cu != nil {
		switch cu.Sphere {
		case "indic":
			return defCaste
		case "japonic", "sinic", "korean":
			return defSinic
		case "steppe":
			return defSteppe
		}
	}
	return defFeudal
}

// strataFor lists the social strata of a town's realm, top to bottom.
func (g *Game) strataFor(bd *BurgData) []string { return g.strataDefFor(bd).list }

func (g *Game) localBurg() *BurgData {
	if g.Cur == nil || g.W == nil {
		return nil
	}
	return g.W.burgByID[g.Cur.Burg]
}

// ---- households ----

var familyRoles = map[string]bool{"mother": true, "father": true, "son": true, "daughter": true, "grandmother": true, "grandfather": true, "aunt": true, "uncle": true}
var femaleRoles = map[string]bool{"mother": true, "daughter": true, "grandmother": true, "aunt": true}

func famRole(a *Actor) string {
	p := strings.SplitN(a.Title, " and ", 2)[0]
	if familyRoles[p] {
		return p
	}
	return ""
}

func occupation(a *Actor) string {
	if i := strings.Index(a.Title, " and "); i >= 0 && familyRoles[a.Title[:i]] {
		return a.Title[i+5:]
	}
	if familyRoles[a.Title] {
		return ""
	}
	return a.Title
}

func isFemale(a *Actor) bool {
	if fr := famRole(a); fr != "" {
		return femaleRoles[fr]
	}
	if a.Has("soc_f") {
		return true
	}
	return false
}

func familyOf(name string) string {
	parts := strings.Fields(name)
	if len(parts) >= 2 {
		return strings.Join(parts[1:], " ")
	}
	return ""
}

// personalOf picks the personal part of a name that isn't the household's family name
// (family-first spheres put the family name first, so the first word can be the house).
func personalOf(name, house string) string {
	for _, w := range strings.Fields(name) {
		if w != house && !strings.Contains(house, w) {
			return w
		}
	}
	return givenOf(name)
}

func givenOf(name string) string {
	if f := strings.Fields(name); len(f) > 0 {
		return f[0]
	}
	return name
}

func shuffleActors(r *Rng, a []*Actor) {
	for i := len(a) - 1; i > 0; i-- {
		j := r.Int(0, i)
		a[i], a[j] = a[j], a[i]
	}
}

func (g *Game) pickHouse(cu *CultureData, head *Actor, used map[string]bool, r *Rng) string {
	for try := 0; try < 10; try++ {
		var h string
		if cu != nil && len(cu.Given) > 0 {
			h = familyOf(Pick(r, cu.Given))
		}
		if h == "" && cu != nil && len(cu.Houses) > 0 {
			h = Pick(r, cu.Houses)
		}
		if h == "" && try == 0 {
			h = familyOf(head.Name)
		}
		if h == "" {
			h = Pick(r, []string{"Ash", "Bren", "Cael", "Doru", "Esk", "Fen", "Garo", "Hale", "Ilm", "Joru", "Kesh", "Lorn"})
		}
		if !used[h] {
			used[h] = true
			return h
		}
	}
	h := familyOf(head.Name)
	if h == "" {
		h = "Nameless"
	}
	return h
}

func (g *Game) casteSlotFor(a *Actor, r *Rng) (slot string, rank int) {
	switch a.Role {
	case "guard":
		rank = r.Int(0, 2)
		if r.Chance(0.1) {
			rank = 3
		}
		return "warrior", rank
	case "merchant", "bookseller", "exotic":
		rank = 1
		if a.Gold > 300 {
			rank++
		}
		if a.Gold > 450 {
			rank++
		}
		return "merchant", rank
	case "smith", "apothecary", "trainer", "ferry":
		return "artisan", r.Int(1, 2)
	case "arena":
		return "low", r.Int(0, 1)
	case "priest":
		return "clergy", r.Int(1, 3)
	case "elder":
		return "noble", 3
	}
	return "common", 0
}

func (g *Game) personSlot(a *Actor, r *Rng) (slot string, rank int) {
	for i := range g.W.People {
		pp := &g.W.People[i]
		if pp.ID != a.SimPer {
			continue
		}
		switch pp.Role {
		case "ruler":
			return "ruler", 3
		case "heir":
			return "noble", 3
		case "general", "hero":
			return "warrior", 3
		case "prophet", "priest":
			return "clergy", 3
		case "magician", "thinker", "inventor":
			return "scholar", r.Int(2, 3)
		case "artist":
			return "artisan", 3
		}
	}
	return "noble", r.Int(2, 3)
}

var anonHouse = []string{"Ash", "Bren", "Cael", "Doru", "Esk", "Fen"}

// assignSociety gives every NPC in the zone a stratum, a rank and (for commoners and shopkeepers' families) a household.
func (g *Game) assignSociety(zn *Zone, r *Rng, bd *BurgData) {
	if zn == nil || bd == nil {
		return
	}
	def := g.strataDefFor(bd)
	cu := g.W.cultByID[bd.Culture]
	var pool, heads, all []*Actor
	for _, a := range zn.Actors {
		if a.Tpl != "npc" || a.Caste != "" {
			continue
		}
		all = append(all, a)
		if r.Chance(0.5) {
			a.Set("soc_f")
		}
		if a.SimPer > 0 {
			s, rk := g.personSlot(a, r)
			a.Caste, a.Rank = def.at(s), rk
			if h := familyOf(a.Name); h != "" {
				a.House = h
			}
			continue
		}
		switch a.Role {
		case "villager":
			occ := a.Title
			switch occ {
			case "drunk":
				if r.Chance(0.6) {
					a.Caste, a.Rank = def.at("outcast"), 0
				} else {
					a.Caste, a.Rank = def.at("low"), 0
				}
				continue
			case "pilgrim":
				a.Caste, a.Rank = def.at("clergy"), 0
				continue
			}
			pool = append(pool, a)
		case "guard", "merchant", "smith", "apothecary", "bookseller", "priest":
			s, rk := g.casteSlotFor(a, r)
			a.Caste, a.Rank = def.at(s), rk
			if a.Role != "guard" || r.Chance(0.4) {
				heads = append(heads, a)
			}
		default:
			s, rk := g.casteSlotFor(a, r)
			a.Caste, a.Rank = def.at(s), rk
			if h := familyOf(a.Name); h != "" {
				a.House = h
			}
		}
	}

	var common []*SpeciesData
	for i := range g.W.Species {
		if g.W.Species[i].Common > 0 {
			common = append(common, &g.W.Species[i])
		}
	}
	used := map[string]bool{}
	shuffleActors(r, pool)

	build := func(members []*Actor, fixedCaste bool) {
		head := members[0]
		house := g.pickHouse(cu, head, used, r)
		var spare *SpeciesData
		if !fixedCaste && len(common) > 1 && r.Chance(0.15) {
			var alts []*SpeciesData
			for _, s := range common {
				if s.ID != head.Species {
					alts = append(alts, s)
				}
			}
			if len(alts) > 0 {
				spare = Pick(r, alts)
			}
		}
		caste, rank := head.Caste, head.Rank
		if !fixedCaste {
			slot := "common"
			if x := r.Float(); x < 0.05 {
				slot = "outcast"
			} else if x < 0.25 {
				slot = "low"
			}
			caste, rank = def.at(slot), 1
			if slot == "outcast" {
				rank = 0
			}
		}
		roles := g.householdRoles(len(members), head, r)
		for i, m := range members {
			occ := occupation(m)
			if m != head || !fixedCaste {
				if occ == "child" || occ == "villager" {
					occ = ""
				}
			} else if occ == "" {
				occ = m.Title
			}
			fr := roles[i]
			if fr == "" {
				fr = famRole(m)
			}
			if len(members) > 1 && fr != "" {
				if occ != "" && m.Role != "" {
					m.Title = fr + " and " + occ
				} else {
					m.Title = fr
				}
			}
			m.House = house
			m.Name = personalOf(m.Name, house) + " " + house
			if !fixedCaste || m != head {
				m.Caste = caste
				if m == head {
					m.Rank = rank
				} else if fr == "son" || fr == "daughter" {
					m.Rank = 0
				} else {
					m.Rank = clampi(rank, 0, 1)
					if fixedCaste {
						m.Rank = head.Rank
					}
				}
			}
			if fr != "" {
				if femaleRoles[fr] {
					m.Set("soc_f")
				} else if m.Flags != nil {
					delete(m.Flags, "soc_f")
				}
			}
			m.Kin = nil
			for _, o := range members {
				if o != m {
					m.Kin = append(m.Kin, o.ID)
				}
			}
			if spare != nil && m.Role == "villager" {
				m.Species = spare.ID
				m.Desc = fmt.Sprintf("A %s %s of %s.", strings.ToLower(spare.Adj), nonEmpty(occ, "villager"), bd.Name)
				if cu != nil && cu.Dress != "" {
					m.Desc += " They wear " + cu.Dress + "."
				}
			}
		}
	}

	// shopkeepers' and guards' families first: they keep their own stratum
	for _, h := range heads {
		if len(pool) == 0 {
			if h.House == "" {
				h.House = g.pickHouse(cu, h, used, r)
				h.Name = personalOf(h.Name, h.House) + " " + h.House
			}
			continue
		}
		n := mini(len(pool), r.Int(1, 4))
		members := append([]*Actor{h}, pool[:n]...)
		pool = pool[n:]
		build(members, true)
	}
	for len(pool) > 0 {
		n := mini(len(pool), r.Int(2, 5))
		if rest := len(pool) - n; rest == 1 && n > 2 {
			n--
		}
		members := pool[:n]
		pool = pool[n:]
		build(members, false)
	}
	// a lone leftover villager still gets a house of their own
	for _, a := range all {
		if a.Caste == "" {
			a.Caste, a.Rank = def.at("common"), 0
		}
	}
}

// householdRoles gives the family role of each member; the head (index 0) is a parent.
func (g *Game) householdRoles(n int, head *Actor, r *Rng) []string {
	roles := make([]string, n)
	if n == 1 {
		return roles
	}
	female := isFemale(head)
	parent := "father"
	spouse := "mother"
	if female {
		parent, spouse = "mother", "father"
	}
	roles[0] = parent
	i := 1
	if n >= 3 || (n == 2 && r.Chance(0.4)) {
		roles[i] = spouse
		i++
	}
	if n >= 4 && r.Chance(0.4) {
		roles[i] = pickStr(r, "grandmother", "grandfather")
		i++
	}
	if n == 5 && i < 4 && r.Chance(0.5) {
		roles[i] = pickStr(r, "aunt", "uncle")
		i++
	}
	for ; i < n; i++ {
		roles[i] = pickStr(r, "son", "daughter")
	}
	return roles
}

func pickStr(r *Rng, a ...string) string { return a[r.Int(0, len(a)-1)] }

// ---- hierarchy helpers ----

func (g *Game) socialStanding() int {
	s := 0
	if g.Cur != nil && g.W != nil && g.W.In(g.Cur.WX, g.Cur.WY) {
		sid := g.W.State[g.W.Idx(g.Cur.WX, g.Cur.WY)]
		rep := g.Rep[fmt.Sprintf("state:%d", sid)]
		s += clampi(rep/60, -6, 10)
		if st := g.W.stateByID[sid]; st != nil && g.P != nil {
			maj := st.Species
			if maj == "" {
				if cu := g.W.cultByID[st.Culture]; cu != nil {
					maj = cu.Species
				}
			}
			if maj != "" && maj == g.P.Species {
				s += 2
			}
		}
	}
	switch {
	case g.ArenaWins >= 5:
		s += 3
	case g.ArenaWins >= 1:
		s++
	}
	return clampi(s, -8, 16)
}

func (g *Game) standingWord() string {
	s := g.socialStanding()
	switch {
	case s <= -3:
		return "despised"
	case s < 3:
		return "unknown"
	case s < 8:
		return "respected"
	}
	return "honored"
}

func (g *Game) casteClass(a *Actor) (high, outcast bool) {
	def := g.strataDefFor(g.localBurg())
	return def.high(a.Caste), def.outcast(a.Caste)
}

func isTrader(a *Actor) bool {
	switch a.Role {
	case "merchant", "smith", "apothecary", "bookseller", "exotic":
		return true
	}
	return false
}

// castePriceMod multiplies what a person charges (divide it when they pay you).
func (g *Game) castePriceMod(a *Actor) float64 {
	if a == nil {
		return 1
	}
	s := g.socialStanding()
	high, out := g.casteClass(a)
	f := 1.0
	if isTrader(a) {
		switch {
		case s < 3:
			f += math64min(0.3, float64(3-s)*0.03)
		case s >= 12:
			f -= 0.08
		case s >= 8:
			f -= 0.05
		}
		if a.Rank >= 3 {
			f += 0.03
		}
	}
	if high {
		f += 0.12
		if s < 3 {
			f += 0.05
		}
	}
	if out {
		f -= 0.07
	}
	if f < 0.88 {
		f = 0.88
	}
	if f > 1.4 {
		f = 1.4
	}
	return f
}

func math64min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

var ordinals = []string{"First", "Second", "Third", "Fourth", "Fifth", "Sixth"}

var trades = map[string]string{
	"merchant": "keeps a stall in the market", "smith": "works the forge", "apothecary": "mixes remedies", "weaver": "weaves cloth for the market",
	"potter": "throws pots", "farmer": "works the fields", "fisher": "hauls nets", "herder": "tends the herds", "baker": "bakes the morning bread",
	"priest": "tends the altar", "bookseller": "sells books", "elder": "sits among the elders", "drunk": "drinks away the days", "pilgrim": "walks the pilgrim roads",
	"warden": "walks the watch", "ferry": "poles the ferry",
}

func tradePhrase(a *Actor) string {
	o := occupation(a)
	if o == "" || o == "villager" || o == "child" {
		return "is still at home"
	}
	if p, ok := trades[o]; ok {
		return p
	}
	return "works as a " + o
}

var guildTrades = map[string]string{"weaver": "Weaver's", "potter": "Potter's", "baker": "Baker's", "smith": "Smith's", "apothecary": "Apothecary's", "merchant": "Merchant's", "bookseller": "Bookseller's", "machinist": "Machinist's", "ripperdoc": "Ripperdoc's", "flesh-sculptor": "Sculptor's", "gunsmith": "Gunsmith's", "chemist": "Chemist's", "fisher": "Fisher's"}

func (g *Game) findActor(id int) *Actor {
	if g.Cur == nil {
		return nil
	}
	for _, o := range g.Cur.Actors {
		if o.ID == id {
			return o
		}
	}
	return nil
}

// kinRelation names who kin is to a, from their household roles.
func kinRelation(a, k *Actor) string {
	ar, kr := famRole(a), famRole(k)
	f := femaleRoles[kr]
	pick := func(m, w string) string {
		if f {
			return w
		}
		return m
	}
	parent := func(x string) bool { return x == "mother" || x == "father" }
	grand := func(x string) bool { return x == "grandmother" || x == "grandfather" }
	child := func(x string) bool { return x == "son" || x == "daughter" }
	aunt := func(x string) bool { return x == "aunt" || x == "uncle" }
	switch {
	case ar == "" || kr == "":
		return pick("kinsman", "kinswoman")
	case child(ar) && parent(kr):
		return pick("father", "mother")
	case child(ar) && child(kr):
		return pick("brother", "sister")
	case child(ar) && grand(kr):
		return pick("grandfather", "grandmother")
	case child(ar) && aunt(kr):
		return pick("uncle", "aunt")
	case parent(ar) && parent(kr):
		return pick("husband", "wife")
	case parent(ar) && child(kr):
		return pick("son", "daughter")
	case parent(ar) && grand(kr):
		return pick("father", "mother")
	case parent(ar) && aunt(kr):
		return pick("brother", "sister")
	case grand(ar) && parent(kr):
		return pick("son", "daughter-in-law")
	case grand(ar) && child(kr):
		return pick("grandson", "granddaughter")
	case aunt(ar) && parent(kr):
		return pick("brother", "sister")
	case aunt(ar) && child(kr):
		return pick("nephew", "niece")
	}
	return pick("kinsman", "kinswoman")
}

func (g *Game) socialLine(a *Actor) string {
	if a == nil || a.Caste == "" {
		return ""
	}
	var s string
	if a.House != "" {
		fr := famRole(a)
		switch {
		case fr == "son" || fr == "daughter":
			ids := []int{a.ID}
			for _, k := range a.Kin {
				if o := g.findActor(k); o != nil && famRole(o) == fr {
					ids = append(ids, k)
				}
			}
			sort.Ints(ids)
			pos := 0
			for i, id := range ids {
				if id == a.ID {
					pos = i
				}
			}
			if len(ids) > 1 && pos < len(ordinals) {
				s = fmt.Sprintf("%s %s of House %s", ordinals[pos], fr, a.House)
			} else {
				s = fmt.Sprintf("%s of House %s", titleCase(fr), a.House)
			}
		case fr != "":
			s = fmt.Sprintf("%s of House %s", titleCase(fr), a.House)
		default:
			s = fmt.Sprintf("%s of House %s", titleCase(nonEmpty(a.Title, "Kin")), a.House)
		}
	} else {
		adj := []string{"A humble", "A modest", "A respected", "A leading"}[clampi(a.Rank, 0, 3)]
		s = fmt.Sprintf("%s %s", adj, nonEmpty(occupation(a), a.Role))
	}
	if gt, ok := guildTrades[occupation(a)]; ok && (a.Role == "villager" || isTrader(a)) {
		s += fmt.Sprintf(", %s of the %s Guild", a.Caste, gt)
	} else {
		s += ", among the " + a.Caste
	}
	s += "."
	for _, id := range a.Kin {
		k := g.findActor(id)
		if k == nil {
			continue
		}
		kr := famRole(k)
		_ = kr
		poss := "His"
		if isFemale(a) {
			poss = "Her"
		}
		s += fmt.Sprintf(" %s %s %s %s.", poss, kinRelation(a, k), givenOf(k.Name), tradePhrase(k))
		break
	}
	return s
}

// casteGreeting is how someone opens a conversation, given the hierarchy between them and the player.
func (g *Game) casteGreeting(a *Actor) string {
	if a == nil || a.Caste == "" {
		return ""
	}
	r := NewRng(fmt.Sprintf("greet:%d:%d", a.ID, g.Turn/600))
	s := g.socialStanding()
	high, out := g.casteClass(a)
	switch {
	case high && s < 3:
		return "\"" + Pick(r, []string{"State your business, wanderer, and be brief.", "You are standing in my light. Speak.", "Do I know you? No. Then why are you addressing me?"}) + "\""
	case high && s < 8:
		return "\"" + Pick(r, []string{"Hm. You may speak.", "A traveler. Very well, what is it?"}) + "\""
	case high:
		return "\"" + Pick(r, []string{"Honored guest. Be welcome in our house.", "Your name is spoken here with respect. Please, ask what you will."}) + "\""
	case out && s < 3:
		return "They glance at the exits. \"" + Pick(r, []string{"I have nothing you want. Please.", "I didn't do it. Whatever it is.", "Keep your voice down, stranger."}) + "\""
	case out:
		return "They lower their eyes. \"" + Pick(r, []string{"Someone like you, speaking to me? Ask, and I will answer.", "I am not worthy of your time. But I listen."}) + "\""
	case a.Role == "guard" && s < 0:
		return "\"" + Pick(r, []string{"Keep your hands where I can see them.", "We know your sort. Move along, or answer to me."}) + "\""
	case a.Role == "guard" && s >= 8:
		return "\"" + Pick(r, []string{"Good day to you, honored one.", "The watch is at your service."}) + "\""
	case isTrader(a) && s < 3:
		return "\"" + Pick(r, []string{"Coin first, friend. Then we talk.", "Looking, or buying? I don't give credit to strangers."}) + "\""
	case isTrader(a) && s >= 8:
		return "\"" + Pick(r, []string{"Ah! A pleasure. My best goods are yours.", "Welcome, welcome! For you, a fair price."}) + "\""
	case s < 0:
		return "\"" + Pick(r, []string{"Mind your own business, outlander.", "We don't want trouble here."}) + "\""
	case s >= 8:
		return "\"" + Pick(r, []string{"It is an honor, truly.", "Well met! We have heard of you."}) + "\""
	}
	return "\"" + Pick(r, []string{"Well met.", "Good day, traveler.", "What brings you our way?"}) + "\""
}
