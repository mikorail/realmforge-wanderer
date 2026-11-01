package engine

import (
	"fmt"
	"sort"
	"strings"
)

// Backpack, weight and encumbrance, plus the tabbed/sortable inventory screen.

// UI state only (not saved).
var (
	invTab  int // index into invTabs
	invSort int // index into invSorts
)

var invTabs = []string{"All", "Weapons", "Armor", "Consumables", "Artifacts", "Misc"}
var invSorts = []string{"category", "value", "weight", "name"}

// invCat returns the tab index (1..5) an item belongs to.
func invCat(it *Item) int {
	switch it.Kind {
	case "weapon", "ranged", "ammo":
		return 1
	case "armor":
		return 2
	case "food", "water", "tonic", "grenade", "ingredient", "dish":
		return 3
	case "artifact":
		return 4
	}
	return 5
}

// itemWeight: the item's own Weight (per unit for stacks) or a default by Kind, times Count.
func itemWeight(it *Item) int {
	if it == nil {
		return 0
	}
	n := maxi(it.Count, 1)
	if it.Weight > 0 {
		if it.Stack() {
			return it.Weight * n
		}
		return it.Weight
	}
	switch it.Kind {
	case "weapon", "ranged":
		return 4
	case "armor":
		if it.Slot == "body" {
			return 8
		}
		return 3
	case "ammo":
		return (n + 9) / 10
	case "food", "tonic", "grenade", "ingredient":
		return n
	case "water":
		return 2
	case "scrap":
		return (n + 3) / 4
	case "light", "book", "manual":
		return 1
	case "artifact", "trophy":
		return 2
	case "letter":
		return 0
	}
	return 1
}

// MakeBackpack builds a back-slot pack; tier 0-5 gives +15..+60 capacity.
func (g *Game) MakeBackpack(tier int) *Item {
	tier = clampi(tier, 0, 5)
	names := map[string][3]string{
		"steampunk":  {"leather satchel", "brass-framed pack", "pneumatic haversack"},
		"dieselpunk": {"canvas musette bag", "army rucksack", "riveted field pack"},
		"atompunk":   {"vinyl shoulder bag", "fallout rucksack", "chrome-ribbed rocket pack"},
		"cyberpunk":  {"mesh sling bag", "cyberdeck harness pack", "smart-fibre cargo rig"},
		"biopunk":    {"hide pouch", "brood-sac", "symbiote carapace-pack"},
		"solarpunk":  {"woven reed satchel", "bamboo-frame pack", "living-vine rucksack"},
		"wasteland":  {"feed-sack satchel", "scav frame pack", "vault-cloth caravan pack"},
		"twilight":   {"worn satchel", "hunter's haversack", "reliquary pack"},
	}
	n, ok := names[g.Genre]
	if !ok {
		n = [3]string{"satchel", "rucksack", "traveller's great pack"}
	}
	return &Item{ID: g.newID(), Kind: "armor", Slot: "back", Name: n[tier/2], Glyph: '(', F: tierColor[tier], Tier: tier,
		Weight: 2 + tier/2, Value: 12 * (tier + 1) * (tier + 1), Extra: map[string]int{"cap": 15 + tier*9}}
}

// capacity: 20 + STR*3 + equipped backpack bonus.
func (g *Game) capacity() int {
	c := 20 + g.P.Score(STR)*3
	for _, bp := range g.P.Body {
		if bp.Item != nil && !bp.Severed && bp.Item.Extra != nil {
			c += bp.Item.Extra["cap"]
		}
	}
	return maxi(c, 1)
}

// burden: total weight carried (pack + worn) and the capacity.
func (g *Game) burden() (carried, cap int) {
	for _, it := range g.invAll() {
		carried += itemWeight(it)
	}
	return carried, g.capacity()
}

// encumbrance: 0 ok, 1 burdened (> cap), 2 overloaded (> 1.5 x cap).
func (g *Game) encumbrance() int {
	c, cp := g.burden()
	switch {
	case c*2 > cp*3:
		return 2
	case c > cp:
		return 1
	}
	return 0
}

// encumbranceQuick is the penalty to add to the player's Quick().
func (g *Game) encumbranceQuick() int {
	switch g.encumbrance() {
	case 1:
		return -30
	case 2:
		return -60
	}
	return 0
}

// canMoveLoad is false when overloaded: you may only drop things.
func (g *Game) canMoveLoad() bool { return g.encumbrance() < 2 }

func (g *Game) encumbranceLabel() string {
	switch g.encumbrance() {
	case 1:
		return "{{W|Burdened}}"
	case 2:
		return "{{R|Overloaded}}"
	}
	return "{{G|Unburdened}}"
}

// invAll: equipped items first, then the pack.
func (g *Game) invAll() []*Item {
	var out []*Item
	for _, bp := range g.P.Body {
		if bp.Item != nil {
			out = append(out, bp.Item)
		}
	}
	return append(out, g.P.Inv...)
}

// invVisible: the current tab's items in the current sort order (stable).
func (g *Game) invVisible() []*Item {
	var out []*Item
	for _, it := range g.invAll() {
		if invTab <= 0 || invTab >= len(invTabs) || invCat(it) == invTab {
			out = append(out, it)
		}
	}
	sortInv(out, invSort)
	return out
}

func sortInv(l []*Item, mode int) {
	total := func(it *Item) int { return it.Value * maxi(it.Count, 1) }
	switch mode {
	case 0:
		sort.SliceStable(l, func(i, j int) bool { return invCat(l[i]) < invCat(l[j]) })
	case 1:
		sort.SliceStable(l, func(i, j int) bool { return total(l[i]) > total(l[j]) })
	case 2:
		sort.SliceStable(l, func(i, j int) bool { return itemWeight(l[i]) > itemWeight(l[j]) })
	case 3:
		sort.SliceStable(l, func(i, j int) bool {
			return strings.ToLower(Plain(l[i].Display())) < strings.ToLower(Plain(l[j].Display()))
		})
	}
}

// row labels skip 's' (the sort key)
const invLetters = "abcdefghijklmnopqrtuvwxyz"

func invLetterIndex(k string) int {
	if len(k) == 1 {
		return strings.Index(invLetters, k)
	}
	return -1
}

func (g *Game) invClampSel(n int) {
	if g.Sel >= n {
		g.Sel = n - 1
	}
	if g.Sel < 0 {
		g.Sel = 0
	}
}

func (g *Game) invKeys(k string) {
	list := g.invVisible()
	g.invClampSel(len(list))
	pick := func(i int) {
		if i >= 0 && i < len(list) {
			g.ItemSel = list[i]
			g.Mode = "item"
		}
	}
	setTab := func(t int) {
		invTab = (t + len(invTabs)) % len(invTabs)
		g.Sel = 0
	}
	if i := invLetterIndex(k); i >= 0 {
		pick(i)
		return
	}
	switch k {
	case "Tab", "ArrowRight":
		setTab(invTab + 1)
	case "ArrowLeft":
		setTab(invTab - 1)
	case "s":
		invSort = (invSort + 1) % len(invSorts)
	case "ArrowUp":
		g.Sel = maxi(0, g.Sel-1)
	case "ArrowDown":
		g.Sel = maxi(0, mini(len(list)-1, g.Sel+1))
	case "Enter":
		pick(g.Sel)
	case "Escape", "i":
		g.Mode = "main"
	}
}

// slotCompare: the item currently worn in the same slot as it (nil if none or it is worn).
func (g *Game) slotCompare(it *Item) *Item {
	if it.Slot == "" || g.P.Equipped(it) != nil {
		return nil
	}
	slot := it.Slot
	if it.Kind == "light" {
		slot = "float"
	}
	for _, bp := range g.P.Body {
		if bp.Slot == slot && bp.Item != nil && !bp.Severed {
			return bp.Item
		}
	}
	return nil
}

func dmgAvg(it *Item) float64 {
	if it.DN <= 0 {
		return 0
	}
	return float64(it.DN)*float64(it.DS+1)/2 + float64(it.DB)
}

func dmgDice(it *Item) string {
	if it.DN <= 0 {
		return "-"
	}
	return fmt.Sprintf("%dd%d%+d", it.DN, it.DS, it.DB)
}

func deltaStr(label string, d float64, f string) string {
	c := "K"
	if d > 0.001 {
		c = "G"
	} else if d < -0.001 {
		c = "R"
	}
	return fmt.Sprintf("{{%s|%s %s}}", c, label, fmt.Sprintf(f, d))
}

// compareLines: stat deltas of it versus what is worn in its slot.
func (g *Game) compareLines(it *Item) []string {
	if it.Slot == "" {
		return nil
	}
	if g.P.Equipped(it) != nil {
		return []string{"{{K|Currently equipped}}"}
	}
	cur := g.slotCompare(it)
	var base Item
	head := "{{K|slot empty}}"
	if cur != nil {
		base = *cur
		head = "{{K|vs}} " + cur.Display()
	}
	parts := []string{
		deltaStr("AV", float64(it.AV-base.AV), "%+.0f"),
		deltaStr("DV", float64(it.DV-base.DV), "%+.0f"),
		deltaStr("PV", float64(it.PV-base.PV), "%+.0f"),
	}
	if it.DN > 0 || base.DN > 0 {
		parts = append(parts, deltaStr("dmg "+dmgDice(&base)+"→"+dmgDice(it), dmgAvg(it)-dmgAvg(&base), "%+.1f"))
	}
	if cp := it.Extra["cap"] - base.Extra["cap"]; it.Extra["cap"] > 0 || base.Extra["cap"] > 0 {
		parts = append(parts, deltaStr("cap", float64(cp), "%+.0f"))
	}
	return []string{"COMPARE " + head, strings.Join(parts, "  ")}
}

func (g *Game) drawInventory() {
	s := g.sc
	const bx, by, bw, bh = 2, 1, SW - 4, 32
	s.Box(bx, by, bw, bh, CDarkCyan, "Inventory")
	x, y, w := bx+2, by+1, bw-4
	listW := 68
	right := x + listW + 2
	list := g.invVisible()
	g.invClampSel(len(list))
	eq := map[*Item]string{}
	for _, bp := range g.P.Body {
		if bp.Item != nil {
			eq[bp.Item] = bp.Name
		}
	}
	carried, cp := g.burden()
	s.TextB(x, y, fmt.Sprintf("{{W|Weight %d/%d}} · %s {{K|· %d items · %d %s}}", carried, cp, g.encumbranceLabel(), len(g.invAll()), g.P.Gold, g.Sk.Currency), CGray, CBgDark, x+w)
	y += 2
	tx := x
	for i, t := range invTabs {
		c := "K"
		if i == invTab {
			c = "Y"
		}
		tx = s.TextB(tx, y, fmt.Sprintf("{{%s|[ %s ]}} ", c, t), CGray, CBgDark, x+listW)
	}
	y++
	s.TextB(x, y, fmt.Sprintf("{{K|sort: %s}}", invSorts[invSort]), CGray, CBgDark, x+listW)
	s.TextB(x+listW-14, y, "{{K|  wt     value}}", CGray, CBgDark, x+listW)
	y++
	const rows = 22
	start := 0
	if g.Sel >= rows {
		start = g.Sel - rows + 1
	}
	nameW := listW - 5 - 14
	for i := start; i < len(list) && i < start+rows; i++ {
		it := list[i]
		bg := CBgDark
		if i == g.Sel {
			bg = CBgSel
		}
		s.Fill(x, y, listW, 1, bg)
		key := ' '
		if i < len(invLetters) {
			key = rune(invLetters[i])
		}
		s.TextB(x, y, fmt.Sprintf("{{W|%c)}} ", key), CGray, bg, x+listW)
		s.Put(x+3, y, it.Glyph, it.F, bg)
		lbl := it.Display()
		if where, ok := eq[it]; ok {
			lbl += " {{K|(" + strings.ToLower(where) + ")}}"
		}
		s.TextB(x+5, y, lbl, CGray, bg, x+5+nameW)
		s.TextB(x+listW-13, y, fmt.Sprintf("{{y|%4d}} {{W|%7d}}", itemWeight(it), it.Value*maxi(it.Count, 1)), CGray, bg, x+listW)
		y++
	}
	if len(list) == 0 {
		s.TextB(x+2, y, "{{K|nothing here}}", CGray, CBgDark, x+listW)
	}
	if len(list) > rows {
		s.TextB(x, by+bh-3, fmt.Sprintf("{{K|%d-%d of %d}}", start+1, mini(start+rows, len(list)), len(list)), CGray, CBgDark, x+listW)
	}
	// body plan
	ry := by + 3
	s.TextB(right, ry, "{{Y|Body}}", CGray, CBgDark, x+w)
	ry++
	for _, bp := range g.P.Body {
		lbl := "{{K|—}}"
		if bp.Item != nil {
			lbl = bp.Item.Display()
		}
		if bp.Severed {
			lbl = "{{r|severed}}"
		}
		s.TextB(right, ry, fmt.Sprintf("{{c|%s:}} %s", bp.Name, lbl), CGray, CBgDark, x+w)
		ry++
	}
	ry++
	if g.Sel < len(list) {
		for _, l := range g.compareLines(list[g.Sel]) {
			s.TextB(right, ry, l, CGray, CBgDark, x+w)
			ry++
		}
	}
	if g.Mode == "item" && g.ItemSel != nil {
		it := g.ItemSel
		lines := it.Describe(g)
		lines = append(lines, "", "{{W|e}} equip/unequip  {{W|u}} use  {{W|d}} drop  {{W|t}} throw  {{W|x}} examine  {{W|k}} disassemble  {{W|Esc}}")
		var ww []string
		for _, l := range lines {
			ww = append(ww, Wrap(l, 56)...)
		}
		h := len(ww) + 2
		s.Box(12, 6, 60, h, CYellow, Plain(it.Display()))
		for k, l := range ww {
			s.TextB(14, 7+k, l, CGray, CBgDark, 71)
		}
	} else {
		s.TextB(x, by+bh-2, "{{K|letter/Enter: item actions · Tab/←→: category · s: sort · Esc: close}}", CGray, CBgDark, x+w)
	}
}
