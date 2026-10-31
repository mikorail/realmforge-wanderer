package engine

import (
	"fmt"
	"strconv"
	"strings"
)

// Factions: every Realmforge realm ("state:N") and faith ("faith:N"), plus creature factions.
var creatureFactions = map[string]string{
	"beasts": "Beasts of the Wild", "vermin": "Vermin", "outlaws": "", "undead": "", "oozes": "Oozes", "fungi": "Fungal Colonies",
	"crystals": "Crystal Choir", "spirits": "Wandering Spirits", "elementals": "Elementals", "eldritch": "Things Below", "constructs": "Old Guardians", "legends": "Legendary Beasts",
}

func (g *Game) FactionName(f string) string {
	if strings.HasPrefix(f, "state:") {
		id, _ := strconv.Atoi(f[6:])
		if s := g.W.stateByID[id]; s != nil && id > 0 {
			return s.FullName
		}
		return "Free Folk"
	}
	if strings.HasPrefix(f, "faith:") {
		id, _ := strconv.Atoi(f[6:])
		if r := g.W.relByID[id]; r != nil {
			return r.Name
		}
	}
	switch f {
	case "outlaws":
		return g.Sk.Bandits
	case "undead":
		return titleCase(strings.TrimPrefix(g.Sk.Undead, "the "))
	case "player":
		return "Your companions"
	}
	if n, ok := creatureFactions[f]; ok && n != "" {
		return n
	}
	return titleCase(f)
}

func (g *Game) initRep() {
	g.Rep = map[string]int{}
	for f := range creatureFactions {
		g.Rep[f] = -500
	}
	g.Rep["outlaws"] = -400
	g.Rep["undead"] = -1000
	g.Rep["eldritch"] = -1000
	g.Rep["legends"] = -1000
	for _, s := range g.W.States {
		if !s.Alive {
			continue
		}
		g.Rep[fmt.Sprintf("state:%d", s.ID)] = 0
	}
	g.Rep["state:0"] = 0
	home := g.W.stateByID[g.HomeState]
	if home != nil {
		g.Rep[fmt.Sprintf("state:%d", home.ID)] = 350
		for id, rel := range home.Relations {
			k := fmt.Sprintf("state:%d", id)
			if _, ok := g.Rep[k]; ok {
				g.Rep[k] += clampi(rel*3, -300, 200)
			}
		}
		for _, wid := range home.Wars {
			g.Rep[fmt.Sprintf("state:%d", wid)] = -350
		}
	}
	for _, r := range g.W.Religions {
		g.Rep[fmt.Sprintf("faith:%d", r.ID)] = 0
	}
	if g.HomeFaith > 0 {
		g.Rep[fmt.Sprintf("faith:%d", g.HomeFaith)] = 250
	}
	// species kin
	for f := range g.Rep {
		_ = f
	}
}

func RepWord(v int) string {
	switch {
	case v >= 600:
		return "{{G|Loved}}"
	case v >= 250:
		return "{{g|Liked}}"
	case v > -250:
		return "{{y|Indifferent}}"
	case v > -600:
		return "{{r|Disliked}}"
	}
	return "{{R|Hated}}"
}

// Hostile decides whether an actor attacks the player.
func (g *Game) Hostile(a *Actor) bool {
	if a == g.P {
		return false
	}
	if a.AI == "follower" {
		return false
	}
	if a.Angry {
		return true
	}
	rep := g.Rep[a.Faction]
	if a.Religion > 0 {
		if fr, ok := g.Rep[fmt.Sprintf("faith:%d", a.Religion)]; ok && fr < rep && a.Role == "priest" {
			rep = fr
		}
	}
	rep += g.attireAffinity(a)
	if a.Role != "" && a.Role != "bandit" {
		return rep <= -600
	}
	return rep <= -250
}

// Enemies: whether two non-player actors fight each other (followers vs the player's foes, beasts vs towns).
func (g *Game) Enemies(a, b *Actor) bool {
	if a == g.P {
		return g.Hostile(b)
	}
	if b == g.P {
		return g.Hostile(a)
	}
	af, bf := a.AI == "follower", b.AI == "follower"
	if af && !bf {
		return g.Hostile(b)
	}
	if bf && !af {
		return g.Hostile(a)
	}
	if af && bf {
		return false
	}
	if a.Faction == b.Faction {
		return false
	}
	civ := func(x *Actor) bool { return strings.HasPrefix(x.Faction, "state:") }
	if civ(a) && civ(b) {
		return g.atWar(a.Faction, b.Faction)
	}
	if civ(a) != civ(b) {
		wild := a
		if civ(a) {
			wild = b
		}
		return wild.Faction == "outlaws" || wild.Faction == "undead" || wild.Faction == "eldritch" || wild.Faction == "legends"
	}
	return false
}

func (g *Game) atWar(fa, fb string) bool {
	a, _ := strconv.Atoi(strings.TrimPrefix(fa, "state:"))
	b, _ := strconv.Atoi(strings.TrimPrefix(fb, "state:"))
	if s := g.W.stateByID[a]; s != nil {
		for _, w := range s.Wars {
			if w == b {
				return true
			}
		}
	}
	return false
}

func (g *Game) AddRep(f string, n int, why string) {
	if f == "" || f == "player" {
		return
	}
	g.Rep[f] += n
	g.compRep(f, n)
	col := "G"
	sign := "+"
	if n < 0 {
		col, sign = "R", ""
	}
	g.Msg(fmt.Sprintf("{{%s|%s%d}} reputation with {{Y|%s}}%s.", col, sign, n, g.FactionName(f), why))
}

// --- NPCs ---

func (g *Game) cultureName(cu *CultureData, r *Rng) string {
	if cu == nil || len(cu.Given) == 0 {
		return Pick(r, []string{"Ash", "Bren", "Cael", "Doru", "Esk", "Fen", "Garo", "Hale"})
	}
	n := Pick(r, cu.Given)
	if len(cu.Houses) > 0 && r.Chance(0.7) {
		n += " " + Pick(r, cu.Houses)
	}
	return n
}

var roleTitle = map[string]string{
	"guard": "warden", "merchant": "merchant", "smith": "smith", "apothecary": "apothecary", "priest": "priest", "elder": "elder", "villager": "villager", "scholar": "scholar", "tailor": "tailor",
}

func (g *Game) roleTitle(role string) string {
	switch g.Genre {
	case "cyberpunk":
		switch role {
		case "guard":
			return "corpsec"
		case "smith":
			return "ripperdoc"
		case "apothecary":
			return "pharm dealer"
		case "elder":
			return "fixer"
		case "priest":
			return "street preacher"
		}
	case "steampunk", "dieselpunk":
		switch role {
		case "guard":
			return "constable"
		case "smith":
			return "machinist"
		case "apothecary":
			return "chemist"
		case "elder":
			return "alderman"
		}
	case "atompunk":
		switch role {
		case "guard":
			return "deputy"
		case "smith":
			return "gunsmith"
		case "apothecary":
			return "doc"
		case "elder":
			return "mayor"
		}
	case "biopunk":
		switch role {
		case "smith":
			return "flesh-sculptor"
		case "apothecary":
			return "gene-brewer"
		case "elder":
			return "brood-elder"
		}
	case "solarpunk":
		switch role {
		case "guard":
			return "warden"
		case "smith":
			return "repair-tech"
		case "apothecary":
			return "herbalist"
		case "elder":
			return "facilitator"
		}
	}
	if t, ok := roleTitle[role]; ok {
		return t
	}
	return role
}

var roleColor = map[string]byte{"guard": CCyan, "merchant": CYellow, "smith": COrange, "apothecary": CGreen, "priest": CWhite, "elder": CMagenta, "villager": CGray, "ferry": CBlue, "notable": CMagenta, "bookseller": CCyan, "exotic": CMagenta, "trainer": COrange, "arena": CRed, "tailor": CMagenta}

func (g *Game) speciesOf(cu *CultureData) *SpeciesData {
	if cu != nil && cu.Species != "" {
		if s := g.W.specByID[cu.Species]; s != nil {
			return s
		}
	}
	return g.W.specByID["human"]
}

func (g *Game) makeNPC(role string, bd *BurgData, cu *CultureData, r *Rng) *Actor {
	sp := g.speciesOf(cu)
	plan := "humanoid"
	spID := "human"
	if sp != nil {
		plan = BodyPlanForSpecies(sp.ID)
		spID = sp.ID
	}
	lvl := g.zoneLevel(bd.P%g.W.W, bd.P/g.W.W, 0) + 2
	a := &Actor{ID: g.newID(), Glyph: '@', F: roleColor[role], Tpl: "npc", Faction: fmt.Sprintf("state:%d", bd.State), Level: lvl,
		Attr: [6]int{14, 14, 14, 14, 14, 14}, Speed: 100, AI: "town", Role: role, Species: spID, Body: MakeBody(plan), State: bd.State}
	if sp != nil {
		a.Attr[STR] += sp.Martial / 2
		a.Attr[AGI] += sp.Intrigue / 2
		a.Attr[TOU] += sp.Health / 8
		a.Attr[EGO] += sp.Diplo / 2
		a.Attr[INT] += sp.Steward / 2
	}
	a.Name = g.cultureName(cu, r)
	a.Proper = true
	a.Title = g.roleTitle(role)
	if role == "villager" && r.Chance(0.4) {
		a.Title = Pick(r, []string{"farmer", "weaver", "potter", "fisher", "herder", "baker", "child", "elder", "drunk", "pilgrim"})
	}
	if sp != nil && sp.ID != "human" {
		a.Desc = fmt.Sprintf("A %s %s of %s.", strings.ToLower(sp.Adj), a.Title, bd.Name)
	} else {
		a.Desc = fmt.Sprintf("A %s of %s.", a.Title, bd.Name)
	}
	if cu != nil && cu.Dress != "" {
		a.Desc += " They wear " + cu.Dress + "."
	}
	a.Religion = g.W.Religion[clampi(bd.P, 0, len(g.W.Religion)-1)]
	a.MaxHP = 14 + lvl*4
	tier := clampi(lvl/3, 0, 5)
	switch role {
	case "guard":
		a.MaxHP += 12
		a.Attr[STR] += 3
		a.Body[4].Item = g.MakeWeapon(Pick(r, []string{"longblade", "axe", "cudgel"}), tier+1, r)
		if p := a.PartFor("body"); p != nil {
			p.Item = g.MakeArmor("body", tier+1, r)
		}
		if p := a.PartFor("head"); p != nil {
			p.Item = g.MakeArmor("head", tier, r)
		}
		a.AI = "guard"
	case "tailor":
		a.Gold = r.Int(120, 300) + lvl*20
		g.stockTailor(a, bd, r)
		a.AI = "still"
	case "trainer", "arena":
		a.AI = "still"
		a.MaxHP += 10
	case "merchant", "smith", "apothecary", "bookseller", "exotic":
		a.Gold = r.Int(150, 400) + lvl*30
		g.stockShop(a, role, tier, r)
		a.AI = "still"
	case "priest":
		a.Gold = r.Int(20, 60)
		a.Inv = append(a.Inv, g.MakeTonic("heal", 2))
	case "elder":
		a.Gold = r.Int(60, 200)
	}
	a.HP = a.MaxHP
	a.XP = 20 + lvl*10
	return a
}

func (g *Game) stockShop(a *Actor, role string, tier int, r *Rng) {
	n := r.Int(8, 14)
	for i := 0; i < n; i++ {
		var it *Item
		switch role {
		case "smith":
			if r.Chance(0.55) {
				it = g.MakeWeapon(Pick(r, []string{"longblade", "shortblade", "axe", "cudgel"}), tier+r.Int(-1, 1), r)
			} else if r.Chance(0.75) {
				it = g.MakeArmor(Pick(r, []string{"body", "head", "feet", "hands", "back", "face"}), tier+r.Int(-1, 1), r)
			} else {
				it = g.MakeRanged(tier, r)
			}
		case "apothecary":
			it = g.MakeTonic(Pick(r, []string{"heal", "heal", "salve", "haste", "strength", "antidote"}), r.Int(1, 3))
		case "bookseller":
			if r.Chance(0.5) {
				it = g.MakeManual(Pick(r, SkillIDs))
			} else {
				it = g.annalsBook(g.W.States[r.Int(0, len(g.W.States)-1)].ID)
			}
		case "exotic":
			switch r.Int(0, 3) {
			case 0, 1:
				it = g.MakeOddity(r, tier+1)
			case 2:
				it = g.MakeGrenade(Pick(r, []string{"fire", "frost", "flash"}), r.Int(1, 3))
			default:
				it = g.MakeLight(r.Int(1, 2))
			}
		default:
			it = g.Loot(r, tier)
		}
		a.Inv = append(a.Inv, it)
	}
	if role == "merchant" {
		a.Inv = append(a.Inv, g.MakeFood(r, 6), g.MakeWater(), g.MakeLight(1), g.MakeAmmo(30))
	}
}

// npcFromPerson brings a Realmforge person into the zone: a ruler, general, prophet, magician...
func (g *Game) npcFromPerson(pp *PersonData, bd *BurgData, r *Rng) *Actor {
	cu := g.W.cultByID[pp.Culture]
	a := g.makeNPC("notable", bd, cu, r)
	a.Name = pp.Name
	a.SimPer = pp.ID
	a.Role = "notable"
	a.F = CMagenta
	if pp.Species != "" {
		a.Species = pp.Species
		a.Body = MakeBody(BodyPlanForSpecies(pp.Species))
	}
	a.Title = nonEmpty(pp.Title, pp.Role)
	a.Attr[STR] = 12 + pp.Martial
	a.Attr[EGO] = 12 + pp.Diplo
	a.Attr[INT] = 12 + pp.Steward
	a.Attr[AGI] = 12 + pp.Intrigue
	a.Level += pp.Renown / 20
	a.MaxHP = 20 + a.Level*5 + pp.Martial*2
	a.HP = a.MaxHP
	a.Religion = pp.Religion
	a.Desc = fmt.Sprintf("%s, %s. Born in the year %d.", pp.Name, a.Title, pp.Born)
	if len(pp.Traits) > 0 {
		a.Desc += " Known as " + strings.ToLower(strings.Join(pp.Traits, ", ")) + "."
	}
	if pp.Role == "ruler" || pp.Role == "general" || pp.Role == "hero" {
		a.Body[4].Item = g.MakeWeapon("longblade", 3+r.Int(0, 1), r)
		if p := a.PartFor("body"); p != nil {
			p.Item = g.MakeArmor("body", 3, r)
		}
	}
	for i := range g.W.Artifacts {
		ar := &g.W.Artifacts[i]
		if ar.HolderType == "person" && ar.HolderID == pp.ID && !g.Taken[ar.ID] {
			it := g.RelicItem(ar)
			it.Ident = true
			if p := a.PartFor(it.Slot); p != nil && p.Item == nil {
				p.Item = it
			} else {
				a.Inv = append(a.Inv, it)
			}
		}
	}
	a.Gold = r.Int(50, 300)
	a.AI = "town"
	return a
}
