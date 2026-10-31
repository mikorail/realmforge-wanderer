package engine

import (
	"fmt"
	"sort"
	"strings"
)

// Mutations, genotypes and morphotypes (QUD_REFERENCE 1.1-1.3). State lives in g.ext("mut").
// Actor.Powers[id] stays the effective level that the rest of the engine reads: mutSync rewrites it as
// base level + rapid advancement + Ego bonus (mental only). Prefix: mu.

type mutState struct {
	Lv        map[string]int `json:"lv"`    // base levels
	Rapid     map[string]int `json:"rapid"` // +3 per rapid advancement, outside the level cap
	Eff       map[string]int `json:"eff"`   // last value written to Actor.Powers (detects outside changes)
	Morph     string         `json:"morph,omitempty"`
	Pending   []string       `json:"pending,omitempty"` // three offered mutations
	LastLevel int            `json:"last"`
	RapidDue  int            `json:"rapidDue,omitempty"`
	Init      bool           `json:"init,omitempty"`
	MorphDone bool           `json:"morphDone,omitempty"`
	RepDone   bool           `json:"repDone,omitempty"`
	TwinDone  bool           `json:"twinDone,omitempty"`
}

func (g *Game) mu() *mutState {
	st := g.ext("mut", func() any { return &mutState{} }).(*mutState)
	if st.Lv == nil {
		st.Lv = map[string]int{}
	}
	if st.Rapid == nil {
		st.Rapid = map[string]int{}
	}
	if st.Eff == nil {
		st.Eff = map[string]int{}
	}
	return st
}

func init() {
	modeKeys["mutate"] = muKey
	modeDraws["mutate"] = muDraw
	mainKeys["V"] = func(g *Game) {
		muInit(g)
		muLevelPoll(g)
		g.Mode, g.Sel = "mutate", 0
	}
	enterHooks = append(enterHooks, func(g *Game, zn *Zone) { muInit(g); muPatchSkin(g) })
	turnHooks = append(turnHooks, muTurn)
	sideHooks = append(sideHooks, func(g *Game) []string { muPatchSkin(g); return nil })
}

// muMentalIDs are the mental mutations; every other non-defect power is physical.
var muMentalIDs = map[string]bool{
	"psiblast": true, "teleport": true, "beguile": true, "forcewall": true, "sense": true, "clairvoyance": true,
	"arcanebolt": true, "slumber": true, "glimmer": true, "forcepush": true, "timedilation": true, "forcebubble": true,
	"stunforce": true, "disintegrate": true, "fugue": true, "syphon": true,
}

func muMental(id string) bool { return muMentalIDs[id] }

// muPhys: a physical mutation (not mental, not a defect).
func muPhys(id string) bool {
	pd := powerByID[id]
	return pd != nil && !pd.Defect && !muMentalIDs[id]
}

// muCap is the level cap for mutations at a character level.
func muCap(level int) int { return mini(10, level/2+1) }

// muRapidLevel: rapid advancement at level 5 and every 10 after.
func muRapidLevel(l int) bool { return l >= 5 && (l-5)%10 == 0 }

func (g *Game) mutCap() int { return muCap(g.P.Level) }

// mutLevel is the effective level of a mutation (0 when not owned). Other packages call this.
func (g *Game) mutLevel(id string) int {
	if g.P == nil {
		return 0
	}
	return g.P.Pow(id)
}

// mutBase is the purchased level without rapid advancement or Ego bonus.
func (g *Game) mutBase(id string) int { return g.mu().Lv[id] }

func (g *Game) muEgoBonus(id string) int {
	if !muMental(id) {
		return 0
	}
	return maxi(0, g.P.M(EGO)) / 2
}

// mutSync writes the effective levels into Actor.Powers. Changes made elsewhere (the character screen
// buying a level) are detected through Eff and folded into the base level.
func (g *Game) mutSync() {
	st := g.mu()
	p := g.P
	if p.Powers == nil {
		p.Powers = map[string]int{}
	}
	for id, lv := range p.Powers {
		pd := powerByID[id]
		if pd == nil || pd.Defect {
			continue
		}
		eff, seen := st.Eff[id]
		if !seen {
			st.Lv[id] = maxi(0, lv-st.Rapid[id]-g.muEgoBonus(id))
		} else if lv != eff {
			st.Lv[id] = maxi(0, st.Lv[id]+lv-eff)
		}
		nv := st.Lv[id] + st.Rapid[id] + g.muEgoBonus(id)
		if st.Lv[id] <= 0 && st.Rapid[id] <= 0 {
			delete(p.Powers, id)
			delete(st.Lv, id)
			delete(st.Eff, id)
			continue
		}
		p.Powers[id] = nv
		st.Eff[id] = nv
	}
	for id := range st.Lv {
		if p.Powers[id] <= 0 {
			delete(st.Lv, id)
			delete(st.Rapid, id)
			delete(st.Eff, id)
		}
	}
}

// mutCool scales a base cooldown by Willpower: -5% per point above 16, at most -80%.
func (g *Game) mutCool(base int) int {
	f := 1 - 0.05*float64(maxi(0, g.P.Score(WIL)-16))
	if f < 0.2 {
		f = 0.2
	}
	return maxi(1, int(float64(base)*f+0.5))
}

func (g *Game) muMorphOK(id string) bool {
	switch g.mu().Morph {
	case "chimera":
		return !muMental(id)
	case "esper":
		return muMental(id)
	}
	return true
}

// mutRaise spends 1 MP on an owned mutation, up to the level cap.
func (g *Game) mutRaise(id string) (bool, string) {
	st := g.mu()
	g.mutSync()
	if g.Genotype == "truekin" {
		return false, "True Kin carry no mutations."
	}
	pd := powerByID[id]
	if pd == nil || pd.Defect || st.Lv[id] <= 0 {
		return false, "You do not have that mutation."
	}
	if st.Lv[id] >= g.mutCap() {
		return false, fmt.Sprintf("Level %d is the most you can raise it at character level %d.", g.mutCap(), g.P.Level)
	}
	if g.MP < 1 {
		return false, "You need 1 mutation point."
	}
	g.MP--
	st.Lv[id]++
	g.mutSync()
	return true, fmt.Sprintf("{{M|%s rises to level %d.}}", g.Sk.PowerName(id), g.P.Powers[id])
}

// mutBuy learns an offered (or any unowned) mutation for 4 MP.
func (g *Game) mutBuy(id string) (bool, string) {
	st := g.mu()
	g.mutSync()
	if g.Genotype == "truekin" {
		return false, "True Kin carry no mutations."
	}
	pd := powerByID[id]
	if pd == nil || pd.Defect {
		return false, "No such mutation."
	}
	if st.Lv[id] > 0 {
		return false, "You already have it."
	}
	if !g.muMorphOK(id) {
		return false, "Your morphotype forbids it."
	}
	if g.MP < 4 {
		return false, "A new mutation costs 4 mutation points."
	}
	g.MP -= 4
	st.Lv[id] = 1
	g.P.Powers[id] = 1
	muGrantBody(g.P, id)
	g.mutSync()
	for i, o := range st.Pending {
		if o == id {
			st.Pending = append(st.Pending[:i:i], st.Pending[i+1:]...)
			break
		}
	}
	return true, fmt.Sprintf("{{M|You develop %s.}}", g.Sk.PowerName(id))
}

// mutRapid applies a rapid advancement (+3 levels outside the cap) to an owned physical mutation.
func (g *Game) mutRapid(id string) (bool, string) {
	st := g.mu()
	g.mutSync()
	if st.RapidDue <= 0 {
		return false, "No rapid advancement is waiting."
	}
	if st.Lv[id] <= 0 || !muPhys(id) {
		return false, "Rapid advancement needs a physical mutation you own."
	}
	st.RapidDue--
	st.Rapid[id] += 3
	g.mutSync()
	return true, fmt.Sprintf("{{Y|Rapid advancement: %s is now level %d.}}", g.Sk.PowerName(id), g.P.Powers[id])
}

// muGrantBody adds the body parts a newly bought mutation needs.
func muGrantBody(p *Actor, id string) {
	switch id {
	case "wings":
		if p.PartFor("wings") == nil {
			p.Body = append(p.Body, &BodyPart{Name: "Wings", Slot: "wings", Limb: true})
		}
	case "stinger":
		for _, b := range p.Body {
			if b.Slot == "tail" && b.Natural != nil {
				return
			}
		}
		p.Body = append(p.Body, &BodyPart{Name: "Stinger", Slot: "tail", Limb: true, Natural: natural("Stinger", 1, 6, 3, "poison")})
	case "multiarms":
		applyBodyPowers(p)
	}
}

// muDropBody removes the body parts a mutation gave.
func muDropBody(p *Actor, id string) {
	var keep []*BodyPart
	for _, b := range p.Body {
		switch {
		case id == "wings" && b.Slot == "wings" && b.Name == "Wings":
			continue
		case id == "stinger" && b.Slot == "tail" && b.Natural != nil && b.Name == "Stinger":
			continue
		case id == "multiarms" && (b.Name == "Lower Right Hand" || b.Name == "Lower Left Hand"):
			if b.Item != nil {
				p.Inv = append(p.Inv, b.Item)
			}
			continue
		}
		keep = append(keep, b)
	}
	p.Body = keep
}

// muStrip removes owned powers matching drop, refunding MP for mutations and undoing defect side effects.
func muStrip(g *Game, drop func(id string, pd *PowerDef) bool) {
	p := g.P
	ids := make([]string, 0, len(p.Powers))
	for id := range p.Powers {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		pd := powerByID[id]
		if pd == nil || !drop(id, pd) {
			continue
		}
		lv := p.Powers[id]
		if pd.Defect {
			if id == "brittle" {
				p.AVNat += 2
			}
			g.MP += pd.Cost // a defect refunded points when taken
		} else {
			g.MP += pd.Cost + maxi(0, lv-1)
		}
		delete(p.Powers, id)
		muDropBody(p, id)
	}
	st := g.mu()
	st.Lv, st.Rapid, st.Eff = map[string]int{}, map[string]int{}, map[string]int{}
	g.MP = maxi(0, g.MP)
}

// muInit applies the genotype once per run (called from hooks, safe to call again).
func muInit(g *Game) {
	st := g.mu()
	if st.Init || g.P == nil {
		return
	}
	st.Init = true
	st.LastLevel = g.P.Level
	if g.Genotype == "truekin" {
		muStrip(g, func(string, *PowerDef) bool { return true })
		g.MP = 0
		muCasteGear(g)
	}
	if g.P.Pow("repugnant") > 0 && !st.RepDone {
		st.RepDone = true
		for k := range g.Rep {
			g.Rep[k] -= 50
		}
	}
	g.mutSync()
	muRoll(g)
}

// muNewGame records the creation choices; the host calls it once with the CharSpec right after
// NewGame has built the player.
func (g *Game) muNewGame(spec CharSpec) {
	muInit(g)
	if g.Genotype == "truekin" {
		return
	}
	switch spec.Morph {
	case "chimera", "esper":
		g.muMorph(spec.Morph)
	}
}

// muMorph applies a morphotype, dropping mutations it forbids and refunding their points.
func (g *Game) muMorph(m string) {
	st := g.mu()
	if st.MorphDone {
		return
	}
	st.MorphDone = true
	st.Morph = m
	g.mutSync()
	muStrip2 := func(id string, pd *PowerDef) bool {
		if pd.Defect {
			return false
		}
		if m == "chimera" {
			return muMental(id)
		}
		return !muMental(id)
	}
	keepLv := map[string]int{}
	for id, lv := range g.mu().Lv {
		if !muStrip2(id, powerByID[id]) {
			keepLv[id] = lv
		}
	}
	keepRapid := map[string]int{}
	for id, v := range g.mu().Rapid {
		if _, ok := keepLv[id]; ok {
			keepRapid[id] = v
		}
	}
	muStrip(g, muStrip2)
	// muStrip clears the maps; restore the survivors
	for id, lv := range keepLv {
		g.mu().Lv[id] = lv
		g.mu().Rapid[id] = keepRapid[id]
		g.P.Powers[id] = lv + keepRapid[id]
		g.mu().Eff[id] = g.P.Powers[id]
	}
	g.mutSync()
	st.Pending = nil
	muRoll(g)
}

// muRoll rolls three offered mutations for the current character level, deterministically.
func muRoll(g *Game) {
	st := g.mu()
	r := NewRng(fmt.Sprintf("%s:mut:%d", g.Seed, g.P.Level))
	var pool []string
	for _, pd := range PowerDefs {
		if pd.Defect || st.Lv[pd.ID] > 0 || !g.muMorphOK(pd.ID) {
			continue
		}
		pool = append(pool, pd.ID)
	}
	st.Pending = nil
	for i := 0; i < 3 && len(pool) > 0; i++ {
		j := r.Int(0, len(pool)-1)
		st.Pending = append(st.Pending, pool[j])
		pool = append(pool[:j:j], pool[j+1:]...)
	}
}

// muLevelPoll notices level-ups (gainXP is shared code) and applies the per-level mutation rules.
func muLevelPoll(g *Game) {
	st := g.mu()
	if g.P == nil {
		return
	}
	for st.LastLevel < g.P.Level {
		st.LastLevel++
		l := st.LastLevel
		if g.Genotype == "truekin" {
			g.MP = maxi(0, g.MP-1) // gainXP hands out a mutation point; True Kin have none
			continue
		}
		if muRapidLevel(l) {
			st.RapidDue++
			g.Msg("{{Y|Rapid advancement!}} Press {{W|V}} and choose a physical mutation to gain +3 levels.")
		}
		if st.Morph == "chimera" {
			r := NewRng(fmt.Sprintf("%s:chimera:%d", g.Seed, l))
			if r.Chance(0.2) {
				g.P.Body = append(g.P.Body, &BodyPart{Name: "Extra Arm", Slot: "hand", Limb: true})
				g.Msg("{{G|Your flesh ripples: an extra limb sprouts.}}")
			}
		}
		st.Pending = nil
	}
	if len(st.Pending) == 0 && g.Genotype != "truekin" {
		muRoll(g)
	}
}

func muTurn(g *Game) {
	muInit(g)
	muLevelPoll(g)
	g.mutSync()
	muDefects(g)
}

// ---- defects ----

func muDefects(g *Game) {
	p := g.P
	if p.Pow("narcolepsy") > 0 && p.St("stunned") == 0 {
		for _, a := range g.Cur.Actors {
			if a.HP > 0 && a != p && dist(a, p) <= 12 && g.Hostile(a) {
				if g.R.Chance(1.0 / 40) {
					p.AddSt("stunned", 6)
					g.Msg("{{C|Narcolepsy: you slump into sudden sleep!}}")
				}
				break
			}
		}
	}
	if st := g.mu(); p.Pow("doppelganger") > 0 && !st.TwinDone && p.Level >= 5 {
		st.TwinDone = true
		if a := muSpawnNear(g, "bandit", p.Level, NewRng(g.Seed+":twin")); a != nil {
			a.Name, a.Proper, a.Glyph, a.F = p.Name, true, '@', CWhite
			a.Set("hunter")
			a.Angry = true
			g.Msg("{{R|Someone wearing your face steps out of the crowd.}}")
		}
	}
}

// ---- castes (True Kin) ----

var muCasteOrder = []string{"horticulturist", "sunpriest", "moonpriest", "syzygyrior", "artifex", "consul", "praetorian", "attendant", "hearthchild", "wheelchild", "deepchild", "fumingchild"}

func init() {
	add := func(id, desc, arc string, attr map[int]int, sk map[string]int) {
		callingDefs = append(callingDefs, CallingDef{ID: id, Desc: desc, Attr: attr, Skills: sk, Caste: true, Arc: arc})
	}
	add("horticulturist", "Tends the arcology gardens; patient and hardy.", "garden", map[int]int{TOU: 2, WIL: 1}, map[string]int{"survival": 2, "lore": 1})
	add("sunpriest", "Keeper of the day-rites; a calm, commanding voice.", "garden", map[int]int{WIL: 2, EGO: 1}, map[string]int{"lore": 1, "persuasion": 1})
	add("moonpriest", "Keeper of the night-rites; reads omens and stars.", "garden", map[int]int{WIL: 2, INT: 1}, map[string]int{"lore": 2})
	add("syzygyrior", "Battle-mystic of the eclipse orders.", "garden", map[int]int{WIL: 2, AGI: 1}, map[string]int{"lore": 1, "acrobatics": 1, "shortblade": 1})
	add("artifex", "Engineer of the old machines.", "forge", map[int]int{INT: 3}, map[string]int{"tinkering": 2, "shortblade": 1})
	add("consul", "Diplomat and administrator.", "forge", map[int]int{EGO: 3, INT: 1}, map[string]int{"persuasion": 2, "lore": 1})
	add("praetorian", "Armoured guard of the high houses.", "forge", map[int]int{STR: 2, TOU: 2}, map[string]int{"longblade": 1, "endurance": 1})
	add("attendant", "Court servant: quick, discreet, well informed.", "forge", map[int]int{AGI: 2, EGO: 2}, map[string]int{"shortblade": 1, "persuasion": 1})
	add("hearthchild", "Raised at the communal hearths.", "wheel", map[int]int{TOU: 2, STR: 1}, map[string]int{"cudgel": 1, "survival": 1})
	add("wheelchild", "Raised among the great turning engines.", "wheel", map[int]int{AGI: 2, STR: 1}, map[string]int{"missile": 1, "acrobatics": 1})
	add("deepchild", "Raised in the flooded lower decks.", "wheel", map[int]int{TOU: 2, WIL: 1}, map[string]int{"survival": 2, "endurance": 1})
	add("fumingchild", "Raised in the furnace districts; a brawler.", "wheel", map[int]int{STR: 3, TOU: 1}, map[string]int{"axe": 1, "endurance": 1})
}

// muCasteNames re-skins the twelve castes per genre (same order as muCasteOrder).
var muCasteNames = map[string][]string{
	"":           {"Gardener", "Priest of the Sun", "Priest of the Moon", "Star-Knight", "Artificer", "Envoy", "Praetorian", "Chamberlain", "Hearth-Born", "Wheel-Born", "Deep-Born", "Forge-Born"},
	"mythic":     {"Orchardist", "Priest of Helios", "Priest of Selene", "Eclipse-Oracle", "Daedalian", "Herald", "Spartiate", "Pedagogue", "Hearth-Child of Hestia", "Child of the Wheel", "Child of Poseidon", "Child of Hephaestus"},
	"steampunk":  {"Hothouse Keeper", "Heliograph Priest", "Lunar Cantor", "Orrery Knight", "Chief Engineer", "Consul-General", "Household Guard", "Valet", "Boiler-Room Child", "Flywheel Child", "Bilge Child", "Foundry Child"},
	"dieselpunk": {"Hydroponist", "Radiant Chaplain", "Nightwatch Chaplain", "Eclipse Pilot", "Master Mechanic", "Attache", "Blackshirt Guard", "Aide-de-Camp", "Canteen Child", "Gearbox Child", "Bilge Child", "Smelter Child"},
	"atompunk":   {"Greenhouse Tech", "Sunlamp Pastor", "Moonbase Pastor", "Eclipse Trooper", "Reactor Engineer", "Ambassador", "Vault Marshal", "Concierge", "Commons Child", "Turbine Child", "Coolant Child", "Reactor-Born"},
	"cyberpunk":  {"Grow-Op Tech", "Sunrise Preacher", "Nightcycle Preacher", "Eclipse Runner", "Fixer-Engineer", "Corp Liaison", "Enforcer", "Handler", "Block Child", "Datacore Child", "Sublevel Child", "Furnace Child"},
	"biopunk":    {"Cultivator", "Light-Tender", "Dark-Tender", "Symbiont Knight", "Gene-Wright", "Broker", "Hive Guard", "Drone-Attendant", "Nest-Born", "Root-Born", "Pool-Born", "Spore-Born"},
	"solarpunk":  {"Permaculturist", "Sun-Singer", "Moon-Singer", "Eclipse Weaver", "Maker", "Delegate", "Commons Guard", "Steward", "Hearth-Kin", "Windmill-Kin", "Tide-Kin", "Kiln-Kin"},
}

func muCasteName(genre, id string) string {
	names, ok := muCasteNames[genre]
	if !ok {
		names = muCasteNames[""]
	}
	for i, c := range muCasteOrder {
		if c == id && i < len(names) {
			return names[i]
		}
	}
	return titleCase(id)
}

func muIsCaste(id string) bool {
	for _, c := range callingDefs {
		if c.ID == id {
			return c.Caste
		}
	}
	return false
}

// muPatchSkin shows the genre's caste name for a True Kin calling (the skin is rebuilt on load).
func muPatchSkin(g *Game) {
	if g.Sk == nil || !muIsCaste(g.Calling) {
		return
	}
	n := muCasteName(g.Genre, g.Calling)
	if g.Sk.Callings[g.Calling] == n {
		return
	}
	m := map[string]string{}
	for k, v := range g.Sk.Callings {
		m[k] = v
	}
	m[g.Calling] = n
	g.Sk.Callings = m
}

// muCasteGear gives the caste's starting weapon (NewGame's kit has none for unknown callings).
func muCasteGear(g *Game) {
	if !muIsCaste(g.Calling) {
		return
	}
	r := NewRng(g.Seed + ":caste:" + g.Calling)
	var it *Item
	switch g.Calling {
	case "praetorian":
		it = g.MakeWeapon("longblade", 1, r)
	case "artifex", "attendant", "syzygyrior":
		it = g.MakeWeapon("shortblade", 0, r)
	case "wheelchild":
		it = g.MakeRanged(0, r)
		g.give(g.MakeAmmo(30))
	case "fumingchild":
		it = g.MakeWeapon("axe", 1, r)
	case "hearthchild", "deepchild":
		it = g.MakeWeapon("cudgel", 0, r)
	case "horticulturist", "sunpriest", "moonpriest":
		it = g.MakeWeapon("cudgel", 0, r)
		g.give(g.MakeTonic("heal", 1))
	case "consul":
		it = g.MakeWeapon("shortblade", 0, r)
		g.P.Gold += 100
	}
	if it == nil {
		return
	}
	g.give(it)
	if p := g.P.PartFor(it.Slot); p != nil && p.Item == nil {
		g.equip(it)
	}
}

// ---- mutation screen (V) ----

type muRow struct {
	kind string // own | offer
	id   string
}

func muRows(g *Game) []muRow {
	st := g.mu()
	var rows []muRow
	var own []string
	for id := range st.Lv {
		own = append(own, id)
	}
	sort.Strings(own)
	for _, id := range own {
		rows = append(rows, muRow{"own", id})
	}
	for _, id := range st.Pending {
		rows = append(rows, muRow{"offer", id})
	}
	return rows
}

func muKey(g *Game, k string) {
	rows := muRows(g)
	switch k {
	case "Escape", "V":
		g.Mode = "main"
		return
	case "ArrowUp":
		g.Sel = maxi(0, g.Sel-1)
		return
	case "ArrowDown":
		g.Sel = mini(maxi(0, len(rows)-1), g.Sel+1)
		return
	}
	if g.Sel >= len(rows) {
		return
	}
	row := rows[g.Sel]
	var ok bool
	var msg string
	switch {
	case k == "r" || k == "R":
		ok, msg = g.mutRapid(row.id)
	case k == "Enter" || k == "+":
		if row.kind == "own" {
			ok, msg = g.mutRaise(row.id)
		} else {
			ok, msg = g.mutBuy(row.id)
		}
	default:
		return
	}
	if msg != "" {
		g.Msg(msg)
	}
	_ = ok
	g.Sel = mini(g.Sel, maxi(0, len(muRows(g))-1))
}

func muDraw(g *Game) {
	s := g.sc
	st := g.mu()
	g.mutSync()
	x, y, w, _ := g.panel(g.Sk.PowersName)
	hdr := fmt.Sprintf("{{W|%d MP}}  level %d (cap %d)  {{M|glimmer %d}}", g.MP, g.P.Level, g.mutCap(), muGlimmer(g))
	if st.Morph != "" {
		hdr += "  {{c|" + titleCase(st.Morph) + "}}"
	}
	if st.RapidDue > 0 {
		hdr += fmt.Sprintf("  {{Y|rapid advancement x%d (r)}}", st.RapidDue)
	}
	s.TextB(x, y, hdr, CGray, CBgDark, x+w)
	y += 2
	if g.Genotype == "truekin" {
		s.TextB(x, y, "{{K|True Kin carry no mutations. Their gifts are body augments (c, third tab).}}", CGray, CBgDark, x+w)
		return
	}
	rows := muRows(g)
	offerHdr := false
	for i, row := range rows {
		if row.kind == "offer" && !offerHdr {
			offerHdr = true
			y++
			s.TextB(x, y, "{{K|Offered this level: Enter learns one for 4 MP}}", CGray, CBgDark, x+w)
			y++
		}
		bg := CBgDark
		if g.Sel == i {
			bg = CBgSel
		}
		s.Fill(x, y, w, 1, bg)
		pd := powerByID[row.id]
		kind := "{{o|physical}}"
		if muMental(row.id) {
			kind = "{{M|mental}}"
		}
		if row.kind == "own" {
			rap := ""
			if st.Rapid[row.id] > 0 {
				rap = fmt.Sprintf(" +%d", st.Rapid[row.id])
			}
			act := "{{K|1 MP}}"
			if st.Lv[row.id] >= g.mutCap() {
				act = "{{K|at cap}}"
			}
			s.TextB(x, y, fmt.Sprintf("%-22s {{M|lv %d%s}} %s %s {{K|%s}}", g.Sk.PowerName(row.id), g.P.Powers[row.id], rap, kind, act, pd.Desc), CGray, bg, x+w)
		} else {
			s.TextB(x, y, fmt.Sprintf("%-22s %s {{K|%s}}", g.Sk.PowerName(row.id), kind, pd.Desc), CGray, bg, x+w)
		}
		y++
	}
	if len(rows) == 0 {
		s.TextB(x, y, "{{K|No mutations.}}", CGray, CBgDark, x+w)
	}
	s.TextB(x, y+2, "{{K|↑↓ select · Enter raise or learn · r rapid advancement (physical) · Esc}}", CGray, CBgDark, x+w)
}

// ---- creation options ----

type muGenotype struct {
	ID, Name, Desc string
	AttrBonus      int
	Mutations      bool
}

func muGenotypes(sk *Skin) []muGenotype {
	return []muGenotype{
		{"mutant", "Mutant", "Warped by the world. Start with " + strings.ToLower(sk.PowersName) + ", raise them with mutation points, and choose a calling.", 0, true},
		{"truekin", "True Kin", "Pure-strain lineage: no mutations, +2 attribute points, a caste instead of a calling; body augments are your edge.", 2, false},
	}
}

type muMorphDef struct{ ID, Name, Desc string }

var muMorphs = []muMorphDef{
	{"", "None", "Any mutation, physical or mental."},
	{"chimera", "Chimera", "Physical mutations only. Each level-up may grow an extra limb."},
	{"esper", "Esper", "Mental mutations only. Draws more glimmer, and psychic hunters."},
}

type muModeDef struct{ ID, Name, Desc string }

var muModes = []muModeDef{
	{"classic", "Classic", "Permadeath. One life."},
	{"roleplay", "Roleplay", "A checkpoint is kept on entering towns; death offers to restore it."},
	{"wander", "Wander", "No XP from kills, creatures start indifferent, discovery and quests give double XP."},
}
