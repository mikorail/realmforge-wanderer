package engine

import (
	"fmt"
	"sort"
	"strings"
)

// Tinkering workbench (T): typed bits from disassembly, learned schematics, sphere and genre
// signature items, item mods and category infusion. Independent of the older scrap crafting in
// powers.go, which keeps working.

type craftState struct {
	Bits  map[string]int  `json:"bits"`  // bit tier "1".."8" -> count
	Ess   map[string]int  `json:"ess"`   // "essence of <creature>" -> count
	Known map[string]bool `json:"known"` // learned recipe ids

	tab, sel int
	target   *Item
}

func (g *Game) craftSt() *craftState {
	st := g.ext("craft", func() any { return &craftState{} }).(*craftState)
	if st.Bits == nil {
		st.Bits = map[string]int{}
	}
	if st.Ess == nil {
		st.Ess = map[string]int{}
	}
	if st.Known == nil {
		st.Known = map[string]bool{}
	}
	return st
}

func (g *Game) tinkRank() int { return clampi(g.Skills["tinkering"], 0, 3) }

func bitKey(t int) string { return fmt.Sprint(clampi(t, 1, 8)) }

func bitLabel(t int) string { return string(rune('A' + clampi(t, 1, 8) - 1)) }

func (st *craftState) bitsLine() string {
	var p []string
	for t := 1; t <= 8; t++ {
		if n := st.Bits[bitKey(t)]; n > 0 {
			p = append(p, fmt.Sprintf("%s:%d", bitLabel(t), n))
		}
	}
	if len(p) == 0 {
		p = append(p, "none")
	}
	return "Bits " + strings.Join(p, " ")
}

func (st *craftState) essCount() int {
	n := 0
	for _, v := range st.Ess {
		n += v
	}
	return n
}

func (st *craftState) takeEssence() string {
	var names []string
	for k, v := range st.Ess {
		if v > 0 {
			names = append(names, k)
		}
	}
	if len(names) == 0 {
		return ""
	}
	sort.Strings(names)
	st.Ess[names[0]]--
	if st.Ess[names[0]] <= 0 {
		delete(st.Ess, names[0])
	}
	return names[0]
}

func (st *craftState) canPay(cost map[int]int) bool {
	for t, n := range cost {
		if st.Bits[bitKey(t)] < n {
			return false
		}
	}
	return true
}

func (st *craftState) pay(cost map[int]int) {
	for t, n := range cost {
		st.Bits[bitKey(t)] -= n
	}
}

func costText(cost map[int]int, ess int) string {
	var ts []int
	for t := range cost {
		ts = append(ts, t)
	}
	sort.Ints(ts)
	var p []string
	for _, t := range ts {
		p = append(p, fmt.Sprintf("%d%s", cost[t], bitLabel(t)))
	}
	if ess > 0 {
		p = append(p, fmt.Sprintf("%d essence", ess))
	}
	return strings.Join(p, " ")
}

// ---- disassembly ----------------------------------------------------------

// bitYield is the bits an item breaks down into: tier from item tier and rarity, amount from value.
func bitYield(it *Item) map[int]int {
	t := clampi(it.Tier+1+rarityIndex(it.Rarity)/2, 1, 8)
	n := clampi(1+it.Value/25, 1, 14)
	if it.Kind == "scrap" || it.Kind == "ammo" {
		t, n = 1, maxi(1, it.Count)
	}
	out := map[int]int{t: n}
	if t > 1 && n > 1 {
		out[t-1] = n / 2
	}
	return out
}

func (g *Game) canBreak(it *Item) bool {
	if it.Quest != 0 || it.Kind == "gold" || it.Kind == "corpse" || it.Kind == "book" {
		return false
	}
	return true
}

// craftDisassemble breaks a pack item into bits and may teach its schematic.
func (g *Game) craftDisassemble(it *Item) string {
	if g.tinkRank() < 1 {
		return "You need the Tinkering skill."
	}
	if !g.canBreak(it) {
		return "That cannot be taken apart."
	}
	st := g.craftSt()
	y := bitYield(it)
	for t, n := range y {
		st.Bits[bitKey(t)] += n
	}
	g.removeInv(it)
	msg := fmt.Sprintf("You take apart %s: %s.", it.Name, costText(y, 0))
	if rc := g.tryLearnFrom(it, 0.25+0.1*float64(g.tinkRank())); rc != nil {
		msg += " {{M|You work out the schematic for " + rc.Name + "!}}"
	}
	g.spend(1000)
	return msg
}

// tryLearnFrom teaches a recipe related to the item with the given chance.
func (g *Game) tryLearnFrom(it *Item, chance float64) *cRecipe {
	if !g.R.Chance(chance) {
		return nil
	}
	st := g.craftSt()
	var pool []*cRecipe
	for _, rc := range g.craftRecipes() {
		if rc.Basic || st.Known[rc.ID] || rc.Kind != it.Kind {
			continue
		}
		if it.Region != "" && rc.Sphere != it.Region {
			continue
		}
		pool = append(pool, rc)
	}
	if len(pool) == 0 {
		return nil
	}
	rc := Pick(g.R, pool)
	st.Known[rc.ID] = true
	return rc
}

// ---- recipes -------------------------------------------------------------

type cRecipe struct {
	ID     string
	Name   string
	Kind   string // weapon ranged armor light tool
	Rank   int
	Sphere string
	Genre  string
	Basic  bool
	Cost   map[int]int
	Ess    int
	Make   func() *Item
}

type sigSpec struct {
	Name, Kind, Sub, Elem string
	Bonus                 map[int]int
	AV, PV, DV            int
}

func sw(name, kind, sub string) sigSpec { return sigSpec{Name: name, Kind: kind, Sub: sub} }

var sphereSigs = map[string][2]sigSpec{
	"sinic":      {sw("nine-ring jian", "weapon", "longblade"), sw("jade-scale lamellar", "armor", "body")},
	"japonic":    {sw("folded-steel tachi", "weapon", "longblade"), sw("lacquered do-maru", "armor", "body")},
	"indic":      {sw("whirling chakram", "weapon", "shortblade"), sw("mirrored char-aina", "armor", "body")},
	"sea":        {sw("wavy-bladed kris", "weapon", "shortblade"), sw("carved prau-cloak", "armor", "back")},
	"steppe":     {sw("horn recurve bow", "ranged", ""), sw("felt-lined lamellar coat", "armor", "body")},
	"persian":    {sw("curved shamshir", "weapon", "longblade"), sw("mail-fringed kolah", "armor", "head")},
	"arabic":     {sw("desert scimitar", "weapon", "longblade"), sw("chain-lined bisht", "armor", "back")},
	"mande":      {sw("griot's spear", "weapon", "longblade"), sw("mud-cloth hunter's tunic", "armor", "body")},
	"yoruba":     {sw("ceremonial adze", "weapon", "axe"), sw("beaded oba crown", "armor", "head")},
	"bantu":      {sw("throwing-blade", "weapon", "shortblade"), sw("hide war-shield cloak", "armor", "back")},
	"ethiopic":   {sw("sickle shotel", "weapon", "shortblade"), sw("lion-mane cloak", "armor", "back")},
	"nahua":      {sw("obsidian macuahuitl", "weapon", "cudgel"), sw("jaguar-warrior suit", "armor", "body")},
	"maya":       {sw("obsidian atlatl", "ranged", ""), sw("quetzal headdress", "armor", "head")},
	"andean":     {sw("star-headed macana", "weapon", "cudgel"), sw("quilted cotton cuirass", "armor", "body")},
	"turtle":     {sw("shell-backed war club", "weapon", "cudgel"), sw("turtleshell breastplate", "armor", "body")},
	"inuit":      {sw("bone harpoon-gun", "ranged", ""), sw("sealskin parka", "armor", "body")},
	"polynesian": {sw("shark-tooth club", "weapon", "cudgel"), sw("feathered cloak", "armor", "back")},
	"norse":      {sw("bearded axe", "weapon", "axe"), sw("ringmail byrnie", "armor", "body")},
	"celtic":     {sw("leaf-bladed sword", "weapon", "longblade"), sw("chieftain's torc-mask", "armor", "face")},
	"latin":      {sw("legion gladius", "weapon", "shortblade"), sw("segmented lorica", "armor", "body")},
	"slavic":     {sw("hooked bardiche", "weapon", "axe"), sw("scale brynia", "armor", "body")},
	"eldritch":   {sw("void-touched dagger", "weapon", "shortblade"), sw("veil of whispers", "armor", "face")},
}

var genreSigs = map[string][]sigSpec{
	"steampunk": {
		{Name: "pressure gauntlet", Kind: "armor", Sub: "hands", Bonus: map[int]int{STR: 2}, AV: 1},
		{Name: "aether-pistol", Kind: "ranged", Elem: "shock", PV: 1},
		{Name: "clockwork cuirass", Kind: "armor", Sub: "body", Bonus: map[int]int{TOU: 1}, AV: 2},
		{Name: "gyro-lamp", Kind: "light"},
	},
	"cyberpunk": {
		{Name: "smart-link", Kind: "armor", Sub: "face", Bonus: map[int]int{AGI: 2, INT: 1}},
		{Name: "monowire", Kind: "weapon", Sub: "shortblade", PV: 2},
		{Name: "subdermal weave", Kind: "armor", Sub: "body", AV: 2, DV: 1},
		{Name: "ghost-cloak", Kind: "armor", Sub: "back", DV: 2},
	},
	"biopunk": {
		{Name: "grown carapace", Kind: "armor", Sub: "body", AV: 3},
		{Name: "barbed lash", Kind: "weapon", Sub: "shortblade", Elem: "poison"},
		{Name: "spore-lung mask", Kind: "armor", Sub: "face", Bonus: map[int]int{TOU: 2}},
		{Name: "chitin spear", Kind: "weapon", Sub: "longblade", PV: 1},
	},
	"solarpunk": {
		{Name: "kite-glider", Kind: "armor", Sub: "back", Bonus: map[int]int{AGI: 2}, DV: 2},
		{Name: "sunblade", Kind: "weapon", Sub: "longblade", Elem: "fire"},
		{Name: "thorn-bow", Kind: "ranged", PV: 1},
		{Name: "heliotrope lantern", Kind: "light"},
	},
	"mythic": {
		{Name: "aegis of the gods", Kind: "armor", Sub: "body", Bonus: map[int]int{WIL: 1}, AV: 3},
		{Name: "thunder-spear", Kind: "weapon", Sub: "longblade", Elem: "shock"},
		{Name: "winged sandals", Kind: "armor", Sub: "feet", Bonus: map[int]int{AGI: 2}, DV: 2},
		{Name: "laurel of insight", Kind: "armor", Sub: "head", Bonus: map[int]int{INT: 2}},
	},
	"dieselpunk": {
		{Name: "trench-cleaver", Kind: "weapon", Sub: "axe", PV: 1},
		{Name: "riveted plate", Kind: "armor", Sub: "body", AV: 3},
		{Name: "flame-pistol", Kind: "ranged", Elem: "fire"},
		{Name: "aviator's goggles", Kind: "armor", Sub: "face", Bonus: map[int]int{AGI: 1}},
	},
	"atompunk": {
		{Name: "ray-gun", Kind: "ranged", Elem: "shock", PV: 1},
		{Name: "lead-lined vest", Kind: "armor", Sub: "body", AV: 3},
		{Name: "power-fist", Kind: "armor", Sub: "hands", Bonus: map[int]int{STR: 3}},
		{Name: "geiger-lantern", Kind: "light"},
	},
}

func (g *Game) sigTier() int { return clampi(2+g.P.Level/4, 2, 4) }

func (g *Game) buildSig(s sigSpec, sphere string, tier int, rarity string) *Item {
	var it *Item
	switch s.Kind {
	case "weapon":
		it = g.MakeWeapon(s.Sub, tier, g.R)
		it.Elem = s.Elem
	case "ranged":
		it = g.MakeRanged(tier, g.R)
		if s.Elem != "" {
			it.Elem = s.Elem
		}
	case "light":
		it = g.MakeLight(2)
	default:
		it = g.MakeArmor(s.Sub, tier, g.R)
	}
	it.Name, it.Region = s.Name, sphere
	it.AV += s.AV
	it.DV += s.DV
	it.PV += s.PV
	if len(s.Bonus) > 0 {
		if it.Bonus == nil {
			it.Bonus = map[int]int{}
		}
		for k, v := range s.Bonus {
			it.Bonus[k] += v
		}
	}
	if rarity != "" {
		it.Rarity, it.F = rarity, RarityColor(rarity)
	}
	it.Value = it.Value*2 + 20
	if it.Extra == nil {
		it.Extra = map[string]int{}
	}
	it.Extra["sig"] = 1
	return it
}

func (g *Game) craftRecipes() []*cRecipe {
	r := g.R
	basic := func(id, name, kind string, cost map[int]int, mk func() *Item) *cRecipe {
		return &cRecipe{ID: id, Name: name, Kind: kind, Rank: 1, Basic: true, Cost: cost, Make: mk}
	}
	rs := []*cRecipe{
		basic("b:heal", g.Sk.Heal, "tool", map[int]int{1: 3}, func() *Item { return g.MakeTonic("heal", 1) }),
		basic("b:grenade", g.Sk.GrenFire, "tool", map[int]int{1: 4}, func() *Item { return g.MakeGrenade("fire", 1) }),
		basic("b:ammo", g.Sk.Ammo+" x10", "tool", map[int]int{1: 2}, func() *Item { return g.MakeAmmo(10) }),
		basic("b:blade", "a plain blade", "weapon", map[int]int{1: 5}, func() *Item {
			return g.MakeWeapon(Pick(r, []string{"longblade", "shortblade", "axe", "cudgel"}), 1, r)
		}),
		basic("b:armor", "a plain cuirass", "armor", map[int]int{1: 5}, func() *Item { return g.MakeArmor("body", 1, r) }),
	}
	var spheres []string
	for sp := range sphereSigs {
		spheres = append(spheres, sp)
	}
	sort.Strings(spheres)
	for _, sp := range spheres {
		for i, s := range sphereSigs[sp] {
			sp, s := sp, s
			rs = append(rs, &cRecipe{ID: fmt.Sprintf("s:%s:%d", sp, i), Name: s.Name, Kind: s.Kind, Rank: 2, Sphere: sp,
				Cost: map[int]int{2: 4, 3: 2}, Make: func() *Item { return g.buildSig(s, sp, g.sigTier(), "uncommon") }})
		}
	}
	for i, s := range genreSigs[g.Genre] {
		s := s
		kind := s.Kind
		if kind == "light" {
			kind = "tool"
		}
		rs = append(rs, &cRecipe{ID: fmt.Sprintf("g:%s:%d", g.Genre, i), Name: s.Name, Kind: kind, Rank: 3, Genre: g.Genre,
			Cost: map[int]int{3: 4, 4: 2}, Ess: 1, Make: func() *Item { return g.buildSig(s, "", g.sigTier()+1, "rare") }})
	}
	return rs
}

func (g *Game) recipeByID(id string) *cRecipe {
	for _, rc := range g.craftRecipes() {
		if rc.ID == id {
			return rc
		}
	}
	return nil
}

func (g *Game) knows(rc *cRecipe) bool {
	return rc.Basic || g.craftSt().Known[rc.ID]
}

// craftRecipe builds a known recipe from bits (and an essence for genre masterworks).
func (g *Game) craftRecipe(rc *cRecipe) string {
	st := g.craftSt()
	switch {
	case g.tinkRank() < rc.Rank:
		return fmt.Sprintf("You need Tinkering rank %d.", rc.Rank)
	case !g.knows(rc):
		return "You do not know that schematic."
	case !st.canPay(rc.Cost):
		return "Not enough bits (" + costText(rc.Cost, 0) + ")."
	case st.essCount() < rc.Ess:
		return "You need an essence component."
	}
	st.pay(rc.Cost)
	for i := 0; i < rc.Ess; i++ {
		st.takeEssence()
	}
	it := rc.Make()
	g.give(it)
	g.spend(2000)
	return "You craft " + it.Display() + "."
}

// MakeSchematic is a readable data disk teaching one recipe (study it at the workbench).
func (g *Game) MakeSchematic(rc *cRecipe) *Item {
	return &Item{ID: g.newID(), Kind: "book", Name: "schematic: " + rc.Name, Glyph: '■', F: CMagenta, Effect: "schem:" + rc.ID,
		Text: "A data disk. Study it at the workbench (C) to learn the recipe.", Value: 60 + 40*rc.Rank}
}

func (g *Game) studySchematic(it *Item) string {
	if g.tinkRank() < 1 {
		return "You need the Tinkering skill."
	}
	id := strings.TrimPrefix(it.Effect, "schem:")
	rc := g.recipeByID(id)
	if rc == nil {
		return "The disk is unreadable."
	}
	g.craftSt().Known[id] = true
	g.removeInv(it)
	g.spend(1000)
	return "You learn the schematic for " + rc.Name + "."
}

func isSchematic(it *Item) bool { return strings.HasPrefix(it.Effect, "schem:") }

// ---- mods ----------------------------------------------------------------

type modDef struct {
	ID, Name string
	Kinds    string // weapon ranged armor
	Cost     map[int]int
	Apply    func(it *Item)
}

func isKind(kinds, k string) bool { return strings.Contains(" "+kinds+" ", " "+k+" ") }

var modDefs = []modDef{
	{"sharp", "sharp", "weapon ranged", map[int]int{2: 3}, func(it *Item) { it.PV += 2 }},
	{"counterweighted", "counterweighted", "weapon ranged", map[int]int{2: 3}, func(it *Item) { addBonus(it, AGI, 1) }},
	{"flaming", "flaming", "weapon ranged", map[int]int{3: 3}, func(it *Item) { it.Elem = "fire" }},
	{"freezing", "freezing", "weapon ranged", map[int]int{3: 3}, func(it *Item) { it.Elem = "frost" }},
	{"electrified", "electrified", "weapon ranged", map[int]int{3: 3}, func(it *Item) { it.Elem = "shock" }},
	{"reinforced", "reinforced", "armor", map[int]int{2: 3}, func(it *Item) { it.AV++ }},
	{"lightweight", "lightweight", "weapon ranged armor", map[int]int{2: 2}, func(it *Item) { it.Weight = maxi(0, it.Weight-2) }},
	{"scoped", "scoped", "ranged", map[int]int{3: 2}, func(it *Item) { it.Range += 3 }},
	{"masterwork", "masterwork", "weapon ranged", map[int]int{4: 3}, func(it *Item) { it.PV++; it.DB++ }},
}

func addBonus(it *Item, a, v int) {
	if it.Bonus == nil {
		it.Bonus = map[int]int{}
	}
	it.Bonus[a] += v
}

func modByID(id string) *modDef {
	for i := range modDefs {
		if modDefs[i].ID == id {
			return &modDefs[i]
		}
	}
	return nil
}

func modsOf(it *Item) []string {
	var out []string
	for k, v := range it.Extra {
		if v > 0 && strings.HasPrefix(k, "mod:") {
			out = append(out, strings.TrimPrefix(k, "mod:"))
		}
	}
	sort.Strings(out)
	return out
}

// modSlots: slots by rarity (common 1 .. mythic 4), capped by Tinkering rank (1, 2, 4).
func (g *Game) modSlots(it *Item) int {
	bySlot := []int{1, 1, 2, 2, 3, 4, 4}[rarityIndex(it.Rarity)]
	byRank := []int{0, 1, 2, 4}[g.tinkRank()]
	return mini(bySlot, byRank)
}

func (g *Game) refreshModName(it *Item) {
	base := it.Name
	if i := strings.LastIndex(base, " <"); i > 0 && strings.HasSuffix(base, ">") {
		base = base[:i]
	}
	parts := modsOf(it)
	if it.Extra["infused"] > 0 {
		parts = append(parts, "infused")
	}
	if len(parts) > 0 {
		base += " <" + strings.Join(parts, ",") + ">"
	}
	it.Name = base
}

func (g *Game) applyMod(it *Item, id string) string {
	md := modByID(id)
	st := g.craftSt()
	switch {
	case md == nil:
		return "Unknown mod."
	case g.tinkRank() < 1:
		return "You need the Tinkering skill."
	case !isKind(md.Kinds, it.Kind):
		return "That mod doesn't fit this item."
	case it.Extra["mod:"+id] > 0:
		return "Already modded that way."
	case len(modsOf(it)) >= g.modSlots(it):
		return fmt.Sprintf("No free mod slot (%d/%d).", len(modsOf(it)), g.modSlots(it))
	case !st.canPay(md.Cost):
		return "Not enough bits (" + costText(md.Cost, 0) + ")."
	}
	st.pay(md.Cost)
	md.Apply(it)
	if it.Extra == nil {
		it.Extra = map[string]int{}
	}
	it.Extra["mod:"+id] = 1
	it.Value += 15 * (len(md.Cost) + 1)
	g.refreshModName(it)
	g.spend(1500)
	return "You fit the " + md.Name + " mod to " + it.Name + "."
}

var infuseCost = map[int]int{4: 2}

// infuse sets an uncategorised gear item's category using an essence (rank 3).
func (g *Game) infuse(it *Item, cat string) string {
	st := g.craftSt()
	switch {
	case g.tinkRank() < 3:
		return "Infusion needs Tinkering rank 3."
	case !isGear(it) || it.Kind == "light":
		return "That item can't hold an infusion."
	case it.Cat != "":
		return "It is already of a category."
	case st.essCount() < 1:
		return "You need an essence."
	case !st.canPay(infuseCost):
		return "Not enough bits (" + costText(infuseCost, 1) + ")."
	}
	ok := false
	for _, c := range rarityCats {
		ok = ok || c == cat
	}
	if !ok {
		return "Unknown category."
	}
	ess := st.takeEssence()
	st.pay(infuseCost)
	it.Cat = cat
	g.applyCategory(it, g.R, cat, rarityIndex(it.Rarity))
	it.Value = it.Value * 5 / 4
	if it.Extra == nil {
		it.Extra = map[string]int{}
	}
	it.Extra["infused"] = 1
	g.refreshModName(it)
	g.spend(2500)
	return fmt.Sprintf("The %s sinks into %s: it is now %s.", ess, it.Name, g.CatLabel(cat))
}

// ---- hooks ---------------------------------------------------------------

func init() {
	// T (Tinker) — C is cooking
	mainKeys["T"] = func(g *Game) {
		if g.tinkRank() < 1 {
			g.Msg("You need the Tinkering skill to use a workbench.")
			return
		}
		st := g.craftSt()
		st.tab, st.sel, st.target = 0, 0, nil
		g.Mode = "bench"
	}
	modeKeys["bench"] = benchKeys
	modeDraws["bench"] = benchDraw
	talkHooks = append(talkHooks, func(g *Game, a *Actor) []Option {
		if a.Role != "smith" && a.Role != "tinker" && !strings.Contains(a.Teach, "tinkering") {
			return nil
		}
		var pool []*cRecipe
		sp := g.actorSphere(a)
		for _, rc := range g.craftRecipes() {
			if rc.Basic || g.craftSt().Known[rc.ID] || rc.Rank > g.tinkRank()+1 {
				continue
			}
			if rc.Genre != "" || rc.Sphere == sp {
				pool = append(pool, rc)
			}
		}
		if len(pool) == 0 {
			return nil
		}
		cost := 50 * (1 + pool[0].Rank) * (100 - g.Skills["persuasion"]*10) / 100
		return []Option{{fmt.Sprintf("{{O|Teach me a schematic.}} {{K|(%d %s)}}", cost, g.Sk.Currency), func() {
			if g.P.Gold < cost {
				g.talkMenu("\"Schematics are not free.\"")
				return
			}
			rc := Pick(g.R, pool)
			g.P.Gold -= cost
			g.craftSt().Known[rc.ID] = true
			g.talkMenu("\"Watch closely. This is how a " + rc.Name + " is made.\" You learn its schematic.")
		}}}
	})
	killHooks = append(killHooks, func(g *Game, d, src *Actor) {
		if src != g.P && (src == nil || src.AI != "follower") {
			return
		}
		if !(d.Boss && g.R.Chance(0.6)) && !(d.Level >= 8 && g.R.Chance(0.04)) {
			return
		}
		name := "essence of " + strings.ToLower(d.Name)
		g.craftSt().Ess[name]++
		g.Msg("{{M|You draw " + name + " from the fallen.}}")
	})
}

// ---- workbench UI ---------------------------------------------------------

type benchEntry struct {
	label string
	do    func() string
}

var benchTabs = []string{"Craft", "Mods", "Take apart", "Study"}

func (g *Game) ownedGear() []*Item {
	var out []*Item
	for _, it := range g.P.Inv {
		if isGear(it) && it.Kind != "artifact" && it.Kind != "light" {
			out = append(out, it)
		}
	}
	for _, p := range g.P.Body {
		if p.Item != nil && !p.Severed && isGear(p.Item) && p.Item.Kind != "artifact" && p.Item.Kind != "light" {
			out = append(out, p.Item)
		}
	}
	return out
}

func (g *Game) benchEntries() []benchEntry {
	st := g.craftSt()
	var out []benchEntry
	switch st.tab {
	case 0:
		for _, rc := range g.craftRecipes() {
			rc := rc
			if !g.knows(rc) {
				continue
			}
			lock := ""
			if rc.Rank > g.tinkRank() {
				lock = fmt.Sprintf(" {{R|rank %d}}", rc.Rank)
			}
			out = append(out, benchEntry{fmt.Sprintf("%s {{K|(%s)}}%s", rc.Name, costText(rc.Cost, rc.Ess), lock), func() string { return g.craftRecipe(rc) }})
		}
	case 1:
		if st.target == nil || !g.owns(st.target) {
			st.target = nil
			for _, it := range g.ownedGear() {
				it := it
				out = append(out, benchEntry{fmt.Sprintf("%s {{K|(%d/%d slots)}}", it.Name, len(modsOf(it)), g.modSlots(it)), func() string {
					st.target, st.sel = it, 0
					return ""
				}})
			}
			break
		}
		it := st.target
		out = append(out, benchEntry{"{{K|< choose another item}}", func() string { st.target, st.sel = nil, 0; return "" }})
		for _, md := range modDefs {
			md := md
			if !isKind(md.Kinds, it.Kind) || it.Extra["mod:"+md.ID] > 0 {
				continue
			}
			out = append(out, benchEntry{fmt.Sprintf("mod: %s {{K|(%s)}}", md.Name, costText(md.Cost, 0)), func() string { return g.applyMod(it, md.ID) }})
		}
		if it.Cat == "" && g.tinkRank() >= 3 {
			for _, c := range rarityCats {
				c := c
				out = append(out, benchEntry{fmt.Sprintf("infuse: %s {{K|(%s)}}", g.CatLabel(c), costText(infuseCost, 1)), func() string { return g.infuse(it, c) }})
			}
		}
	case 2:
		for _, it := range append([]*Item{}, g.P.Inv...) {
			it := it
			if g.canBreak(it) && !isSchematic(it) {
				out = append(out, benchEntry{it.Display(), func() string { return g.craftDisassemble(it) }})
			}
		}
	case 3:
		for _, it := range append([]*Item{}, g.P.Inv...) {
			it := it
			if isSchematic(it) {
				out = append(out, benchEntry{it.Display(), func() string { return g.studySchematic(it) }})
			}
		}
	}
	return out
}

func (g *Game) owns(t *Item) bool {
	for _, it := range g.ownedGear() {
		if it == t {
			return true
		}
	}
	return false
}

func benchKeys(g *Game, k string) {
	st := g.craftSt()
	es := g.benchEntries()
	switch k {
	case "Escape":
		g.Mode = "main"
	case "Tab", "ArrowRight":
		st.tab, st.sel, st.target = (st.tab+1)%len(benchTabs), 0, nil
	case "ArrowLeft":
		st.tab, st.sel, st.target = (st.tab+len(benchTabs)-1)%len(benchTabs), 0, nil
	case "ArrowUp":
		st.sel = maxi(0, st.sel-1)
	case "ArrowDown":
		st.sel = mini(maxi(0, len(es)-1), st.sel+1)
	case "Enter", " ":
		if st.sel < len(es) {
			if m := es[st.sel].do(); m != "" {
				g.Msg(m)
			}
			st.sel = mini(st.sel, maxi(0, len(g.benchEntries())-1))
		}
	}
}

func benchDraw(g *Game) {
	st := g.craftSt()
	x, y, _, h := g.panel("Workbench")
	s := g.sc
	tabs := ""
	for i, t := range benchTabs {
		if i == st.tab {
			tabs += "{{W|[" + t + "]}} "
		} else {
			tabs += "{{K|" + t + "}}  "
		}
	}
	s.Text(x, y, tabs, CGray)
	s.Text(x, y+1, fmt.Sprintf("{{C|%s}}   Essences %d   Tinkering rank %d", st.bitsLine(), st.essCount(), g.tinkRank()), CGray)
	es := g.benchEntries()
	rows := h - 6
	top := maxi(0, st.sel-rows+1)
	for i := top; i < len(es) && i < top+rows; i++ {
		mark := "  "
		if i == st.sel {
			mark = "{{Y|>}} "
		}
		s.Text(x, y+3+i-top, mark+es[i].label, CGray)
	}
	if len(es) == 0 {
		s.Text(x, y+3, "{{K|(nothing here)}}", CGray)
	}
	if n := len(g.Msgs); n > 0 {
		s.Text(x, y+h-2, g.Msgs[n-1].T, CGray)
	}
	s.Text(x, y+h-1, "{{K|Tab: tab   Up/Down: select   Enter: do   Esc: close}}", CGray)
}
