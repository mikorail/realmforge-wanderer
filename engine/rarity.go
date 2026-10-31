package engine

import (
	"fmt"
	"strings"
)

// Item rarity: tiers, special categories, genre re-skinning. Hooks live in items.go / combat.go.

var rarityOrder = []string{"common", "uncommon", "rare", "ultrarare", "legendary", "mythic", "unique"}

var rarityLabels = map[string]string{"common": "Common", "uncommon": "Uncommon", "rare": "Rare", "ultrarare": "Ultra Rare",
	"legendary": "Legendary", "mythic": "Mythic", "unique": "Unique"}

var rarityColors = map[string]byte{"common": CGray, "uncommon": CGreen, "rare": CBlue, "ultrarare": CCyan,
	"legendary": COrange, "mythic": CMagenta, "unique": CYellow}

// palette letters for {{X|text}} markup
var rarityMarkup = map[string]string{"common": "y", "uncommon": "G", "rare": "B", "ultrarare": "C",
	"legendary": "O", "mythic": "M", "unique": "W"}

// value multiplier per tier (category adds at most x1.25 so tiers stay ordered)
var rarityValueMul = []float64{1, 1.5, 2.5, 4, 7, 12, 20}

func RarityColor(r string) byte {
	if c, ok := rarityColors[r]; ok {
		return c
	}
	return CGray
}

func RarityLabel(r string) string {
	if l, ok := rarityLabels[r]; ok {
		return l
	}
	return "Common"
}

// rarityIndex is the tier position (common 0 .. unique 6); unknown strings are common.
func rarityIndex(r string) int {
	for i, s := range rarityOrder {
		if s == r {
			return i
		}
	}
	return 0
}

// rollRarity picks a tier for loot at zone level 1..40. Never returns "unique".
func (g *Game) rollRarity(r *Rng, level int) string {
	lv := float64(clampi(level, 1, 40))
	t := (lv - 1) / 39
	ramp := func(from, span float64) float64 {
		v := (lv - from) / span
		if v < 0 {
			return 0
		}
		if v > 1 {
			return 1
		}
		return v
	}
	w := []float64{
		80 - 35*t,
		15 + 5*t,
		3 + 14*t,
		0.1 + 6*t,
		ramp(5, 10) * (0.7 + 2*t),
		ramp(12, 8) * (0.1 + 1.2*t),
	}
	sum := 0.0
	for _, x := range w {
		sum += x
	}
	x := r.Float() * sum
	for i, v := range w {
		if x < v {
			return rarityOrder[i]
		}
		x -= v
	}
	return "common"
}

// ---- categories ----

var rarityCats = []string{"biblical", "occult", "omni", "eldritch", "celestial", "cursed", "ancestral", "techno"}

var catLabel = map[string]string{"biblical": "Biblical", "occult": "Occult", "omni": "Omni", "eldritch": "Eldritch",
	"celestial": "Celestial", "cursed": "Cursed", "ancestral": "Ancestral", "techno": "Techno"}

// genre-specific re-skins of category names
var catSkin = map[string]map[string]string{
	"fantasy":    {"biblical": "Sacred", "occult": "Forbidden", "techno": "Clockwork"},
	"mythic":     {"biblical": "Divine", "occult": "Chthonic", "techno": "Daedalian"},
	"steampunk":  {"biblical": "Consecrated", "occult": "Alchemist's", "techno": "Brass-Engine"},
	"dieselpunk": {"biblical": "Chapel-Blessed", "occult": "Bootleg-Rite", "techno": "Dynamo"},
	"atompunk":   {"biblical": "Atomic Gospel", "occult": "Fallout-Cult", "techno": "Isotope"},
	"cyberpunk":  {"biblical": "Neo-Gospel", "occult": "Black-ICE", "techno": "Netrunner"},
	"biopunk":    {"biblical": "Pure-Strain", "occult": "Heretic-Splice", "techno": "Gene-Wright"},
	"solarpunk":  {"biblical": "Sunlit", "occult": "Night-Root", "techno": "Photonic"},
}

// CatLabel is the genre-flavoured display name of a category ("" for none).
func (g *Game) CatLabel(cat string) string {
	if cat == "" {
		return ""
	}
	if m, ok := catSkin[g.Genre]; ok {
		if s, ok := m[cat]; ok {
			return s
		}
	}
	if s, ok := catLabel[cat]; ok {
		return s
	}
	return titleCase(cat)
}

// suffix phrases per category ("... of the Seventh Seal")
var catSuffix = map[string][]string{
	"biblical":  {"of the Seventh Seal", "of the Lamb", "of Judgement", "of the Covenant"},
	"occult":    {"of the Forbidden Name", "of the Black Rite", "of the Unlit Door", "of Whispers"},
	"omni":      {"of the Whole", "of All Things", "of the Circle Unbroken"},
	"eldritch":  {"of the Deep Dark", "of Unseen Stars", "of the Maw Beyond"},
	"celestial": {"of the Morning Star", "of the Spheres", "of Falling Light"},
	"cursed":    {"of Sorrow", "of the Hanged", "of Ruin"},
	"ancestral": {"of the Elders", "of the First Line", "of Bones Remembered"},
	"techno":    {"of the Core", "of Perfect Logic", "of the Old Machine"},
}

var tierPrefix = [][]string{
	nil,
	{"Sturdy", "Keen", "Fine", "Balanced"},
	{"Gleaming", "Honed", "Tempered", "Superior"},
	{"Radiant", "Peerless", "Storm-forged", "Exalted"},
	{"Fabled", "Dread", "Sovereign", "Heroic"},
	{"Primordial", "World-ending", "Godforged", "Eternal"},
}

var tierSuffix = [][]string{
	nil,
	{"of the Wolf", "of the Hawk", "of the Ox"},
	{"of the Bear", "of Swiftness", "of the Warden"},
	{"of the Dragon", "of the Tempest", "of Kings"},
	{"of Legends", "of the Last Dawn", "of the Titan"},
	{"of Endings", "of the First Fire", "of the Sundered Sky"},
}

// tier prefix flavour by genre, mixed in occasionally
var genrePrefix = map[string][]string{
	"fantasy":    {"Runed", "Elven-wrought"},
	"mythic":     {"Olympian", "Titan-cast"},
	"steampunk":  {"Pressure-forged", "Gear-wound"},
	"dieselpunk": {"Piston-driven", "Chrome-plated"},
	"atompunk":   {"Isotopic", "Rad-hardened"},
	"cyberpunk":  {"Overclocked", "Chromed"},
	"biopunk":    {"Grafted", "Living"},
	"solarpunk":  {"Sun-tempered", "Verdant"},
}

func isGear(it *Item) bool {
	switch it.Kind {
	case "weapon", "armor", "ranged", "artifact", "light":
		return true
	}
	return false
}

func (g *Game) rollCategory(r *Rng, idx int) string {
	chance := []float64{0, 0, 0.12, 0.22, 0.40, 0.60}
	if idx >= len(chance) {
		idx = len(chance) - 1
	}
	if !r.Chance(chance[idx]) {
		return ""
	}
	pool := []string{"biblical", "occult", "eldritch", "celestial", "cursed", "ancestral", "techno"}
	if idx >= 4 && r.Chance(0.2) {
		return "omni"
	}
	return Pick(r, pool)
}

// applyRarity upgrades gear in place for the given tier. Consumables and ammo stay common.
func (g *Game) applyRarity(it *Item, r *Rng, rarity string) {
	idx := rarityIndex(rarity)
	if rarity == "unique" {
		idx = 6
	}
	if !isGear(it) || idx == 0 || rarity == "common" {
		it.Rarity = "common"
		return
	}
	it.Rarity = rarityOrder[idx]
	st := idx
	if st > 5 {
		st = 5
	}
	it.F = RarityColor(it.Rarity)

	cat := ""
	if idx < 6 {
		cat = g.rollCategory(r, idx)
	}
	it.Cat = cat

	switch it.Kind {
	case "weapon", "ranged":
		it.DS += st
		it.PV += (st + 1) / 2
		if idx >= 4 {
			it.DN++
		}
		if it.Kind == "weapon" {
			it.MaxStr += st / 2
		}
	case "armor":
		it.AV += (st + 1) / 2
		it.DV += st / 3
		if idx >= 5 {
			it.AV++
		}
	case "light":
		it.Light += st
	case "artifact":
		it.Power += st
		if it.Charges > 0 {
			it.Charges += st
		}
	}

	// attribute bonuses
	nb := []int{0, 0, 1, 1, 2, 3, 3}[idx]
	if idx == 3 && r.Chance(0.5) {
		nb = 2
	}
	mag := 1 + idx/4
	if nb > 0 && it.Bonus == nil {
		it.Bonus = map[int]int{}
	}
	for i := 0; i < nb; i++ {
		it.Bonus[r.Int(0, 5)] += mag
	}

	// elements on weapons
	if (it.Kind == "weapon" || it.Kind == "ranged") && it.Elem == "" {
		p := []float64{0, 0, 0.05, 0.3, 0.5, 0.8, 1}[idx]
		if r.Chance(p) {
			it.Elem = Pick(r, []string{"fire", "frost", "shock"})
		}
	}

	g.applyCategory(it, r, cat, idx)

	mul := rarityValueMul[idx]
	if cat != "" {
		mul *= 1.25
	}
	base := it.Value
	if base < 4 {
		base = 4
	}
	it.Value = int(float64(base) * mul)

	g.rarityName(it, r, idx, cat)
}

func (g *Game) applyCategory(it *Item, r *Rng, cat string, idx int) {
	if cat == "" {
		return
	}
	if it.Bonus == nil {
		it.Bonus = map[int]int{}
	}
	big := 1 + idx/3
	switch cat {
	case "biblical":
		it.Bonus[WIL] += big
		it.Bonus[EGO] += big
		if (it.Kind == "weapon" || it.Kind == "ranged") && it.Elem == "" {
			it.Elem = "fire"
		}
	case "occult":
		it.Bonus[Pick(r, []int{STR, AGI, INT, WIL})] += big + 2
		it.Bonus[INT] += big
		it.Bonus[TOU] -= 1 + idx/3
		if it.Kind == "weapon" || it.Kind == "ranged" {
			it.Elem = Pick(r, []string{"poison", "shock"})
			it.DS++
		}
	case "omni":
		n := 1 + idx/5
		for a := 0; a < 6; a++ {
			it.Bonus[a] += n
		}
		if r.Chance(0.5) {
			it.Bonus[r.Int(0, 5)]++
		}
	case "eldritch":
		it.Bonus[INT] += big
		it.Bonus[WIL] += 1
		if it.Kind == "armor" {
			it.DV++
		}
	case "celestial":
		it.Bonus[EGO] += big
		it.Bonus[AGI] += 1
		if it.Kind == "light" {
			it.Light += 2
		}
	case "cursed":
		// bloody bargain: power at the price of vigour
		it.Bonus[STR] += big + 1
		it.Bonus[TOU] -= 1
		if it.Kind == "weapon" || it.Kind == "ranged" {
			it.DS += 2
		} else if it.Kind == "armor" {
			it.AV++
		}
	case "ancestral":
		it.Bonus[TOU] += big
		it.Bonus[STR] += 1
		if it.Kind == "armor" {
			it.AV++
		}
	case "techno":
		it.Bonus[INT] += big
		it.Bonus[AGI] += 1
		if (it.Kind == "weapon" || it.Kind == "ranged") && it.Elem == "" {
			it.Elem = "shock"
		}
	}
	for k, v := range it.Bonus {
		if v == 0 {
			delete(it.Bonus, k)
		}
	}
}

// rarityName prefixes/suffixes the item's name with tier and category flavour.
func (g *Game) rarityName(it *Item, r *Rng, idx int, cat string) {
	st := idx
	if st > 5 {
		st = 5
	}
	base := it.Name
	if base == "" {
		base = "relic"
	}
	if cat != "" {
		if r.Chance(0.5) {
			base = g.CatLabel(cat) + " " + base
		} else {
			base = base + " " + Pick(r, catSuffix[cat])
		}
		if idx >= 2 && st >= 1 && r.Chance(0.4) {
			base = Pick(r, tierPrefix[st]) + " " + base
		}
	} else if st >= 1 {
		switch {
		case r.Chance(0.2) && len(genrePrefix[g.Genre]) > 0 && st >= 2:
			base = Pick(r, genrePrefix[g.Genre]) + " " + base
		case r.Chance(0.5):
			base = Pick(r, tierPrefix[st]) + " " + base
		default:
			base = base + " " + Pick(r, tierSuffix[st])
		}
	}
	it.Name = strings.TrimSpace(base)
}

// MakeUnique turns base into a one-of-a-kind item with strong boosts.
func (g *Game) MakeUnique(base *Item, name, lore string) *Item {
	r := NewRng("unique:" + name)
	kind := base.Kind
	if !isGear(base) {
		base.Kind = "artifact"
		base.Slot = "float"
	}
	g.applyRarity(base, r, "mythic")
	if kind != base.Kind {
		base.Power += 3
	}
	if base.Bonus == nil {
		base.Bonus = map[int]int{}
	}
	base.Bonus[r.Int(0, 5)] += 2
	if base.DS > 0 {
		base.DS += 2
		base.PV += 2
	}
	if base.AV > 0 {
		base.AV += 2
	}
	base.Value = base.Value*2 + 200
	base.Rarity = "unique"
	base.Ident = true
	base.Name = name
	base.Lore = lore
	base.F = RarityColor("unique")
	return base
}

// bossDrop is a guaranteed rare-or-better weapon, armor or artifact.
func (g *Game) bossDrop(r *Rng, level int) *Item {
	tier := clampi(level/8, 2, 5)
	var it *Item
	switch x := r.Int(0, 99); {
	case x < 40:
		it = g.MakeWeapon(Pick(r, []string{"longblade", "shortblade", "axe", "cudgel"}), tier, r)
	case x < 75:
		it = g.MakeArmor(Pick(r, []string{"body", "head", "feet", "hands", "back", "face"}), tier, r)
	default:
		it = g.MakeOddity(r, tier)
		it.Ident = true
	}
	rar := "common"
	for i := 0; i < 8 && rarityIndex(rar) < 2; i++ {
		rar = g.rollRarity(r, level+12)
	}
	if rarityIndex(rar) < 2 {
		rar = "rare"
	}
	g.applyRarity(it, r, rar)
	return it
}

// RarityTag is markup like "{{O|[Legendary]}}" (with the generic category name appended).
func (it *Item) RarityTag() string {
	return it.rarityTag("")
}

// RarityTagG is RarityTag with the genre-flavoured category name.
func (it *Item) RarityTagG(g *Game) string {
	return it.rarityTag(g.CatLabel(it.Cat))
}

func (it *Item) rarityTag(cat string) string {
	r := it.Rarity
	if r == "" {
		r = "common"
	}
	s := fmt.Sprintf("{{%s|[%s]}}", rarityMarkup[r], RarityLabel(r))
	if it.Cat != "" {
		if cat == "" {
			cat = catLabel[it.Cat]
		}
		s += " {{M|" + cat + "}}"
	}
	return s
}

// categoryBonusDamage is extra damage from a weapon's category against defender d.
func (g *Game) categoryBonusDamage(w *Item, d *Actor) int {
	if w == nil || w.Cat == "" || d == nil {
		return 0
	}
	idx := rarityIndex(w.Rarity)
	f := d.Faction
	holyFoe := f == "undead" || f == "eldritch"
	switch w.Cat {
	case "biblical":
		if holyFoe {
			return g.R.Dice(1, 3+idx)
		}
	case "celestial":
		if holyFoe {
			return g.R.Dice(1, 2+idx/2)
		}
	case "eldritch":
		if f == "beasts" || f == "vermin" {
			return g.R.Dice(1, 2+idx/2)
		}
	case "techno":
		if f == "constructs" || f == "machine" {
			return g.R.Dice(1, 3+idx/2)
		}
	case "occult":
		return g.R.Dice(1, 2)
	}
	return 0
}
