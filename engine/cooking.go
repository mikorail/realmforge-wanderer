package engine

import (
	"fmt"
	"sort"
	"strings"
)

// Cooking: campfires, ingredients and meals. Every ingredient carries one flavour tag; a meal cooked from
// two or three of them grants the tags' effects for about a day. Ingredients come from beasts you kill
// (their nature decides the tag), wild herbs and fungi (by biome), and the staple foods of each culture,
// read from the Realmforge export (CultureData.Food), sold by town merchants. New tag combinations are
// written into the cookbook under a name drawn from the local culture.

// tagInfo: display name, colour code and effect text per level (1 or 2).
type cookTag struct {
	Name, Col string
	Desc      func(lv int) string
}

var cookTags = map[string]cookTag{
	"hearty":  {"hearty", "O", func(lv int) string { return fmt.Sprintf("+%d Strength", 2*lv) }},
	"spry":    {"spry", "C", func(lv int) string { return fmt.Sprintf("+%d DV", 2*lv) }},
	"thick":   {"thick", "y", func(lv int) string { return fmt.Sprintf("+%d AV", lv) }},
	"mending": {"mending", "G", func(lv int) string { return map[int]string{1: "faster healing", 2: "much faster healing"}[lv] }},
	"keen":    {"keen", "M", func(lv int) string { return fmt.Sprintf("+%d Intelligence and Willpower", 2*lv) }},
	"ember":   {"ember", "R", func(lv int) string { return fmt.Sprintf("fire harms you %d%% less", 50+25*(lv-1)) }},
	"rime": {"rime", "B", func(lv int) string {
		return fmt.Sprintf("frost harms you %d%% less; you shake off freezing", 50+25*(lv-1))
	}},
	"deep": {"deep", "b", func(lv int) string { return fmt.Sprintf("+%d breath underwater", 30*lv) }},
	"lasting": {"lasting", "W", func(lv int) string {
		return map[int]string{1: "hunger and thirst come half as fast", 2: "hunger and thirst come a third as fast"}[lv]
	}},
	"venom": {"venom", "g", func(lv int) string { return fmt.Sprintf("%d%% of your blows poison", 20*lv) }},
}

var cookTagIDs = []string{"hearty", "spry", "thick", "mending", "keen", "ember", "rime", "deep", "lasting", "venom"}

// Dish is a recipe you invented: a sorted tag multiset and the name it was given where you first cooked it.
type Dish struct {
	Key     string   `json:"k"`
	Name    string   `json:"n"`
	Tags    []string `json:"t"`
	Where   string   `json:"w,omitempty"`
	Cooked  int      `json:"c"`
	FirstAt int      `json:"y,omitempty"` // game day
}

type herb struct {
	Name, Tag string
	Biomes    []int // nil = any land
	Under     bool  // grows below ground
}

var herbs = []herb{
	{"bitterroot", "mending", []int{4, 6, 8, 9}, false},
	{"sunthistle", "ember", []int{1, 3, 4}, false},
	{"frostcap lichen", "rime", []int{9, 10, 11, 2}, false},
	{"marsh reed", "deep", []int{12, 7, 8}, false},
	{"wild garlic", "hearty", []int{6, 8, 5}, false},
	{"hare's sorrel", "spry", []int{4, 3, 6}, false},
	{"ironbark tuber", "thick", []int{9, 6, 7}, false},
	{"dreaming poppy", "keen", []int{3, 5, 4, 12}, false},
	{"camel thorn gum", "lasting", []int{1, 2, 3}, false},
	{"nightshade berry", "venom", []int{5, 7, 12}, false},
	{"ash pepper", "ember", []int{13, 16}, false},
	{"glasswort", "keen", []int{14}, false},
	{"glowing morel", "keen", nil, true},
	{"cave truffle", "hearty", nil, true},
	{"blind eel roe", "deep", nil, true},
	{"stone puffball", "thick", nil, true},
	{"pale ink-cap", "venom", nil, true},
}

// staples splits a culture's food line ("rice, coconut, fish sauce and chilli") into ingredient names.
func staples(food string) []string {
	var out []string
	for _, part := range strings.Split(food, ",") {
		for _, s := range strings.Split(part, " and ") {
			s = strings.TrimSpace(s)
			if s != "" && len(s) <= 28 {
				out = append(out, s)
			}
		}
	}
	return out
}

// stapleTag is a stable tag for a staple name, with a few readable overrides.
func stapleTag(name string) string {
	n := strings.ToLower(name)
	switch {
	case strings.Contains(n, "chilli") || strings.Contains(n, "pepper") || strings.Contains(n, "spice"):
		return "ember"
	case strings.Contains(n, "fish") || strings.Contains(n, "seaweed") || strings.Contains(n, "eel") || strings.Contains(n, "shell"):
		return "deep"
	case strings.Contains(n, "meat") || strings.Contains(n, "mutton") || strings.Contains(n, "beef") || strings.Contains(n, "pork"):
		return "hearty"
	case strings.Contains(n, "tea") || strings.Contains(n, "coffee") || strings.Contains(n, "cacao") || strings.Contains(n, "wine"):
		return "keen"
	case strings.Contains(n, "bread") || strings.Contains(n, "rice") || strings.Contains(n, "millet") || strings.Contains(n, "maize") || strings.Contains(n, "dried"):
		return "lasting"
	case strings.Contains(n, "milk") || strings.Contains(n, "curd") || strings.Contains(n, "cheese") || strings.Contains(n, "butter"):
		return "thick"
	case strings.Contains(n, "fruit") || strings.Contains(n, "berr") || strings.Contains(n, "honey"):
		return "spry"
	}
	return cookTagIDs[HashStr(n)%uint64(len(cookTagIDs))]
}

func (g *Game) makeIngredient(name, tag string, n int, col byte) *Item {
	return &Item{ID: g.newID(), Kind: "ingredient", Name: name, Effect: tag, Glyph: '%', F: col, Count: n, Value: 4, Power: 120}
}

// corpseTag picks the flavour a beast's flesh carries; sapient folk and the dead give none.
func corpseTag(d *Actor) string {
	if d.Role != "" || d.Proper || d.Species != "" || d.SimPer > 0 || d.Has("undead") {
		return ""
	}
	switch {
	case d.Has("fire"):
		return "ember"
	case d.Has("frost"):
		return "rime"
	case d.Has("aquatic"):
		return "deep"
	case d.Has("fly"):
		return "spry"
	case d.Has("regen"):
		return "mending"
	case d.Has("poison") || d.Has("acid"):
		return "venom"
	case d.Has("burrow") || d.AVNat >= 4:
		return "thick"
	case d.Boss || d.MaxHP >= 40:
		return "hearty"
	}
	return []string{"hearty", "lasting", "thick", "spry"}[HashStr(d.Tpl)%4]
}

// scatterHerbs drops a few wild herbs in a zone; a side rng keeps the zone's own generation unchanged.
func (g *Game) scatterHerbs(zn *Zone, p int) {
	r := NewRng(fmt.Sprintf("%s:herbs:%s", g.W.Seed, zn.Key))
	under := zn.Z > 0
	b := -1
	if p >= 0 && p < len(g.W.Biome) {
		b = g.W.Biome[p]
	}
	var pool []herb
	for _, h := range herbs {
		if h.Under != under {
			continue
		}
		if h.Biomes == nil {
			pool = append(pool, h)
			continue
		}
		for _, hb := range h.Biomes {
			if hb == b {
				pool = append(pool, h)
				break
			}
		}
	}
	if len(pool) == 0 {
		return
	}
	n := r.Int(0, 2)
	for i := 0; i < n; i++ {
		h := pool[r.Int(0, len(pool)-1)]
		x, y, ok := zn.FreeSpot(r, nil)
		if !ok {
			return
		}
		it := g.makeIngredient(h.Name, h.Tag, 1, CGreen)
		it.X, it.Y = x, y
		zn.Items = append(zn.Items, it)
	}
}

// stockIngredients: a merchant sells the staples of their own culture.
func (g *Game) stockIngredients(a *Actor, cu *CultureData, r *Rng) {
	if cu == nil {
		return
	}
	st := staples(cu.Food)
	for i, s := range st {
		if i >= 4 {
			break
		}
		a.Inv = append(a.Inv, g.makeIngredient(s, stapleTag(s), r.Int(2, 4), CYellow))
	}
}

// ---- meals ----

func (a *Actor) mealLv(tag string) int {
	if a.Meal == nil {
		return 0
	}
	return a.Meal[tag]
}

// resistMeal scales elemental damage for ember/rime meals.
func resistMeal(d *Actor, n int, cause string) int {
	tag := map[string]string{"fire": "ember", "frost": "rime"}[cause]
	if tag == "" {
		return n
	}
	switch d.mealLv(tag) {
	case 1:
		return (n + 1) / 2
	case 2:
		return n / 4
	}
	return n
}

func (g *Game) mealTick() {
	p := g.P
	if p.Meal == nil {
		return
	}
	g.MealLeft--
	if g.MealLeft == 120 {
		g.Msg("{{K|The warmth of your last meal is fading.}}")
	}
	if g.MealLeft <= 0 {
		p.Meal = nil
		g.MealName = ""
		g.Msg("{{K|The benefits of your meal wear off.}}")
	}
}

// slowHunger: lasting meals skip hunger/thirst on some turns.
func (g *Game) slowHunger() bool {
	lv := g.P.mealLv("lasting")
	return lv > 0 && g.Turn%(lv+1) != 0
}

func (g *Game) mealText() string {
	p := g.P
	var parts []string
	for _, t := range cookTagIDs {
		if lv := p.mealLv(t); lv > 0 {
			parts = append(parts, "{{"+cookTags[t].Col+"|"+cookTags[t].Desc(lv)+"}}")
		}
	}
	return strings.Join(parts, ", ")
}

// ---- the cooking screen ----

// cookCandidates: tagged ingredients and beast corpses carried or lying where you stand.
func (g *Game) cookCandidates() []*Item {
	var out []*Item
	ok := func(it *Item) bool { return (it.Kind == "ingredient" || it.Kind == "corpse") && it.Effect != "" }
	for _, it := range g.P.Inv {
		if ok(it) {
			out = append(out, it)
		}
	}
	for _, it := range g.Cur.ItemsAt(g.P.X, g.P.Y) {
		if ok(it) {
			out = append(out, it)
		}
	}
	return out
}

func (g *Game) maxIngredients() int {
	if g.Skills["survival"] >= 2 {
		return 3
	}
	return 2
}

// canCook explains why you can't light a fire here ("" when you can).
func (g *Game) canCook() string {
	z := g.Cur
	switch {
	case z == nil:
		return "There is nowhere to cook."
	case z.Underwater:
		return "You can't light a fire underwater."
	case z.WaterAt(g.P.X, g.P.Y) > 0:
		return "You can't make a fire standing in water."
	case g.nearHostile():
		return "You can't cook with enemies in sight."
	}
	return ""
}

func (g *Game) fireWord() string {
	if g.Cur.Kind == "town" {
		return "a public hearth"
	}
	for dy := -1; dy <= 1; dy++ {
		for dx := -1; dx <= 1; dx++ {
			if x, y := g.P.X+dx, g.P.Y+dy; g.Cur.In(x, y) && g.Cur.Fire[y*MW+x] > 0 {
				return "the flames nearby"
			}
		}
	}
	return "a small campfire"
}

func (g *Game) openCook() {
	if why := g.canCook(); why != "" {
		g.Msg(why)
		return
	}
	g.CookSel = map[int]bool{}
	g.Sel = 0
	g.Mode = "cook"
}

func (g *Game) keyCook(k string) {
	cands := g.cookCandidates()
	switch {
	case k == "Escape":
		g.Mode = "main"
	case k == "Enter":
		g.cook(cands)
	case k == "ArrowUp":
		g.Sel = maxi(0, g.Sel-1)
	case k == "ArrowDown":
		g.Sel = mini(maxi(0, len(cands)-1), g.Sel+1)
	case k == " " && g.Sel < len(cands):
		g.toggleCook(cands[g.Sel])
	case len(k) == 1 && k[0] >= 'a' && k[0] <= 'z':
		if i := int(k[0] - 'a'); i < len(cands) {
			g.Sel = i
			g.toggleCook(cands[i])
		}
	}
}

func (g *Game) toggleCook(it *Item) {
	if g.CookSel[it.ID] {
		delete(g.CookSel, it.ID)
		return
	}
	if len(g.CookSel) >= g.maxIngredients() {
		g.Msg(fmt.Sprintf("You can only combine %d ingredients%s.", g.maxIngredients(), map[bool]string{true: " (Survival 2 allows 3)", false: ""}[g.maxIngredients() == 2]))
		return
	}
	g.CookSel[it.ID] = true
}

func (g *Game) localCulture() *CultureData {
	if g.W == nil || !g.W.In(g.WX, g.WY) {
		return nil
	}
	p := g.W.Idx(g.WX, g.WY)
	if p < len(g.W.Culture) {
		return g.W.cultByID[g.W.Culture[p]]
	}
	return nil
}

var dishWords = []string{"stew", "broth", "skewers", "pie", "porridge", "roast", "dumplings", "curry", "hash", "fritters", "flatbread", "pot"}

func shortName(it *Item) string {
	n := strings.TrimSuffix(it.Name, " corpse")
	if it.Kind == "corpse" {
		n += " meat"
	}
	return n
}

// dishName names a new recipe after the local culture and its two main ingredients.
func (g *Game) dishName(key string, ings []*Item) string {
	word := dishWords[HashStr(key)%uint64(len(dishWords))]
	style := ""
	if cu := g.localCulture(); cu != nil && cu.Name != "" && cu.Name != "Wildlands" {
		style = cu.Name + "-style "
	}
	names := make([]string, 0, len(ings))
	for _, it := range ings {
		names = append(names, shortName(it))
	}
	switch len(names) {
	case 0:
		return "plain " + word
	case 1:
		return style + names[0] + " " + word
	case 2:
		return style + word + " of " + names[0] + " and " + names[1]
	}
	return style + word + " of " + names[0] + ", " + names[1] + " and " + names[2]
}

func (g *Game) cook(cands []*Item) {
	var ings []*Item
	for _, it := range cands {
		if g.CookSel[it.ID] {
			ings = append(ings, it)
		}
	}
	p := g.P
	place := g.fireWord()
	if len(ings) == 0 {
		var food *Item
		for _, it := range p.Inv {
			if it.Kind == "food" {
				food = it
				break
			}
		}
		if food == nil {
			g.Msg("Choose ingredients first (letters or Space), or carry some food for a simple meal.")
			return
		}
		g.Mode = "main"
		g.Hunger = maxi(0, g.Hunger-maxi(food.Power, 300)-200)
		h := maxi(2, p.MaxHP/5)
		p.HP = mini(p.MaxHP, p.HP+h)
		g.Msg(fmt.Sprintf("You warm the %s over %s and eat. {{G|(+%d HP)}}", food.Name, place, h))
		g.consume(food)
		g.spendTurns(10)
		return
	}
	tags := map[string]int{}
	var list []string
	for _, it := range ings {
		tags[it.Effect] = mini(2, tags[it.Effect]+1)
		list = append(list, it.Effect)
	}
	sort.Strings(list)
	key := strings.Join(list, "+")
	var dish *Dish
	for i := range g.Cookbook {
		if g.Cookbook[i].Key == key {
			dish = &g.Cookbook[i]
		}
	}
	isNew := dish == nil
	if isNew {
		where := ""
		if cu := g.localCulture(); cu != nil {
			where = cu.Name
		}
		g.Cookbook = append(g.Cookbook, Dish{Key: key, Name: g.dishName(key, ings), Tags: list, Where: where, FirstAt: g.Turn / 1440})
		dish = &g.Cookbook[len(g.Cookbook)-1]
	}
	dish.Cooked++
	for _, it := range ings {
		g.useUp(it)
	}
	p.Meal = tags
	dur := 1440 + 360*g.Skills["survival"]
	if !isNew {
		dur += dur / 4 // a dish you know comes out better
	}
	g.MealLeft, g.MealName = dur, dish.Name
	g.Hunger = maxi(0, g.Hunger-500)
	h := maxi(2, p.MaxHP/8)
	p.HP = mini(p.MaxHP, p.HP+h)
	g.Mode = "main"
	g.Msg(fmt.Sprintf("You cook over %s and eat {{Y|%s}}.", place, dish.Name))
	g.Msg("You feel: " + g.mealText() + ".")
	if isNew {
		g.Msg("{{G|You write a new recipe into your cookbook.}}")
		g.gainXP(15 + 5*len(ings))
	}
	g.spendTurns(15)
}

// useUp consumes one of an ingredient, from the pack or from the floor.
func (g *Game) useUp(it *Item) {
	for _, own := range g.P.Inv {
		if own == it {
			g.consume(it)
			return
		}
	}
	if it.Count > 1 {
		it.Count--
		return
	}
	for i, fl := range g.Cur.Items {
		if fl == it {
			g.Cur.Items = append(g.Cur.Items[:i], g.Cur.Items[i+1:]...)
			return
		}
	}
}

func (g *Game) spendTurns(n int) {
	for i := 0; i < n && !g.Dead; i++ {
		g.spend(1000)
	}
}

func (g *Game) drawCook() {
	s := g.sc
	x, y, w, h := g.panel("Cooking over " + g.fireWord())
	cands := g.cookCandidates()
	para := func(txt string) {
		for _, l := range Wrap(txt, w) {
			s.TextB(x, y, l, CGray, CBgDark, x+w)
			y++
		}
	}
	para(fmt.Sprintf("{{K|Pick up to %d ingredients (letters / Space), Enter to cook, Esc to leave.}}", g.maxIngredients()))
	y++
	if len(cands) == 0 {
		para("{{K|You have no ingredients. Wild herbs and fungi grow in the wilds and below; beasts you kill leave meat; markets sell the local staples.}}")
		y++
		para("{{W|Enter}} cook a simple meal from your food (more filling than eating it cold).")
		return
	}
	sel := map[string]int{}
	for i, it := range cands {
		if y > h-6 {
			break
		}
		mark, bg := "[ ]", byte(CBgDark)
		if g.CookSel[it.ID] {
			mark = "{{G|[x]}}"
			sel[it.Effect] = mini(2, sel[it.Effect]+1)
		}
		if i == g.Sel {
			bg = CBgSel
		}
		where := ""
		if g.P.Equipped(it) == nil && !g.carrying(it) {
			where = " {{K|(on the ground)}}"
		}
		t := cookTags[it.Effect]
		s.TextB(x, y, fmt.Sprintf("{{W|%c)}} %s %s {{%s|<%s>}}%s", 'a'+i, mark, it.Display(), t.Col, t.Name, where), CGray, bg, x+w)
		y++
	}
	y++
	if len(sel) == 0 {
		para("{{K|Nothing chosen. Enter alone cooks a simple meal from your food.}}")
		return
	}
	s.TextB(x, y, "{{Y|This meal:}}", CGray, CBgDark, x+w)
	y++
	for _, t := range cookTagIDs {
		if lv := sel[t]; lv > 0 {
			s.TextB(x+2, y, "{{"+cookTags[t].Col+"|"+cookTags[t].Desc(lv)+"}}", CGray, CBgDark, x+w)
			y++
		}
	}
	if g.Meal() != "" {
		y++
		para("{{K|Replaces your current meal: " + g.MealName + ".}}")
	}
}

// Meal is the name of the dish whose effects you currently carry ("" when none).
func (g *Game) Meal() string {
	if g.P == nil || g.P.Meal == nil {
		return ""
	}
	return g.MealName
}

func (g *Game) carrying(it *Item) bool {
	for _, own := range g.P.Inv {
		if own == it {
			return true
		}
	}
	return false
}

// cookbookLines feeds the journal's Cookbook tab.
func (g *Game) cookbookLines() []string {
	var lines []string
	if g.Meal() != "" {
		lines = append(lines, fmt.Sprintf("{{Y|Now:}} %s {{K|(%d h left)}}", g.MealName, (g.MealLeft+59)/60), "  "+g.mealText(), "")
	}
	for i := len(g.Cookbook) - 1; i >= 0; i-- {
		d := g.Cookbook[i]
		lv := map[string]int{}
		for _, t := range d.Tags {
			lv[t] = mini(2, lv[t]+1)
		}
		var eff []string
		for _, t := range cookTagIDs {
			if lv[t] > 0 {
				eff = append(eff, "{{"+cookTags[t].Col+"|"+cookTags[t].Desc(lv[t])+"}}")
			}
		}
		where := ""
		if d.Where != "" {
			where = " first cooked among the " + d.Where + ","
		}
		lines = append(lines, fmt.Sprintf("{{W|%s}} {{K|(%s;%s cooked %d×)}}", d.Name, strings.Join(d.Tags, " + "), where, d.Cooked))
		lines = append(lines, Wrap("  "+strings.Join(eff, ", "), SW-6)...)
	}
	if len(lines) == 0 {
		lines = []string{"{{K|No recipes yet. Press C to cook: combine two tagged ingredients to invent a dish.}}"}
	}
	return lines
}
