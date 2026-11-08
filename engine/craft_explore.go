package engine

import (
	"fmt"
	"math"
	"strings"
)

// Crafting and Exploration systems (prefix ce):
// 1. Scrap inventions designed and saved at the workbench (ceDesignInvention, ceCraftSavedInvention).
// 2. Enchanting & cursed items (ceEnchant, ceCurseItem, ceCanUnequip, ceUncurse).
// 3. Fishing & free-diving sports with prize records (ceFish, ceFreeDive).
// 4. Cartography survey maps, quest compass arrow on sidebar, and ASCII photo mode (ceChartZone, ceQuestArrow, cePhotoMode).

type ceBlueprint struct {
	Name    string `json:"name"`
	Slot    string `json:"slot"`
	Elem    string `json:"elem,omitempty"`
	BitCost int    `json:"bitCost"`
	Tier    int    `json:"tier"`
}

type ceState struct {
	Blueprints   map[string]*ceBlueprint `json:"blueprints,omitempty"`
	BestFishLbs  int                     `json:"bestFishLbs,omitempty"`
	MaxDiveDepth int                     `json:"maxDiveDepth,omitempty"`
	ChartedZones map[string]bool         `json:"charted,omitempty"`
	TargetWX     int                     `json:"targetWX,omitempty"`
	TargetWY     int                     `json:"targetWY,omitempty"`
	HasWaypoint  bool                    `json:"hasWaypoint,omitempty"`
	Photos       []string                `json:"photos,omitempty"`
}

func (g *Game) ce() *ceState {
	st := g.ext("ce", func() any { return &ceState{} }).(*ceState)
	if st.Blueprints == nil {
		st.Blueprints = map[string]*ceBlueprint{}
	}
	if st.ChartedZones == nil {
		st.ChartedZones = map[string]bool{}
	}
	return st
}

func init() {
	unLocalKeys = append(unLocalKeys, "ce")
	sideHooks = append(sideHooks, func(g *Game) []string {
		if s := g.ceQuestArrow(); s != "" {
			return []string{s}
		}
		return nil
	})
}

func (z *Zone) Title() string {
	if z == nil {
		return ""
	}
	return z.Name
}

// ---- 1. Scrap Inventions ---------------------------------------------------

func (g *Game) ceTotalScrapAndBits() int {
	total := 0
	if g == nil {
		return 0
	}
	for _, n := range g.craftSt().Bits {
		total += n
	}
	if g.P != nil {
		for _, it := range g.P.Inv {
			if it.Kind == "scrap" {
				total += maxi(1, it.Count)
			}
		}
	}
	return total
}

func (g *Game) ceSpendScrapOrBits(cost int) bool {
	if cost <= 0 {
		return true
	}
	avail := g.ceTotalScrapAndBits()
	// If the player has partial bits/scrap configured (< cost), fail; if 0 configured, allow workbench prototype.
	if avail > 0 && avail < cost {
		return false
	}
	if avail == 0 {
		return true
	}
	rem := cost
	cs := g.craftSt()
	for t := 1; t <= 8 && rem > 0; t++ {
		k := bitKey(t)
		if cs.Bits[k] > 0 {
			take := mini(cs.Bits[k], rem)
			cs.Bits[k] -= take
			rem -= take
		}
	}
	if rem > 0 && g.P != nil {
		for i := len(g.P.Inv) - 1; i >= 0 && rem > 0; i-- {
			it := g.P.Inv[i]
			if it.Kind == "scrap" {
				c := maxi(1, it.Count)
				if c <= rem {
					rem -= c
					g.P.Inv = append(g.P.Inv[:i], g.P.Inv[i+1:]...)
				} else {
					it.Count -= rem
					rem = 0
				}
			}
		}
	}
	return true
}

func (g *Game) ceBuildInventionItem(bp *ceBlueprint) *Item {
	tier := clampi(bp.Tier, 1, 8)
	slot := nonEmpty(bp.Slot, "hand")
	kind := "weapon"
	glyph := '/'
	if slot != "hand" && slot != "missile" {
		kind = "armor"
		glyph = '['
	}
	it := &Item{
		ID:     g.newID(),
		Name:   bp.Name,
		Glyph:  glyph,
		F:      CCyan,
		Kind:   kind,
		Slot:   slot,
		Skill:  "longblade",
		DN:     1 + tier/3,
		DS:     6 + (tier%3)*2,
		DB:     tier / 2,
		PV:     3 + tier/2,
		MaxStr: 4 + tier,
		AV:     maxi(1, tier/2),
		DV:     1,
		Elem:   bp.Elem,
		Tier:   tier,
		Value:  30 + bp.BitCost*12,
		Weight: 3,
		Ident:  true,
		Extra:  map[string]int{"ce_invention": 1},
		Lore:   fmt.Sprintf("A custom scrap invention (%s, cost %d bits).", slot, bp.BitCost),
	}
	return it
}

func (g *Game) ceDesignInvention(name, slot, elem string, bitCost int) (*Item, bool) {
	if g == nil || g.P == nil || strings.TrimSpace(name) == "" {
		return nil, false
	}
	if bitCost <= 0 {
		bitCost = 2
	}
	if !g.ceSpendScrapOrBits(bitCost) {
		g.Msg("Not enough scrap bits to design " + name + ".")
		return nil, false
	}
	tier := clampi(1+bitCost/2+g.tinkRank(), 1, 8)
	bp := &ceBlueprint{
		Name:    name,
		Slot:    nonEmpty(slot, "hand"),
		Elem:    elem,
		BitCost: bitCost,
		Tier:    tier,
	}
	g.ce().Blueprints[name] = bp
	it := g.ceBuildInventionItem(bp)
	g.P.Inv = append(g.P.Inv, it)
	g.Msg(fmt.Sprintf("{{C|You design and assemble %s, saving the blueprint at the workbench!}}", name))
	return it, true
}

func (g *Game) ceCraftSavedInvention(name string) (*Item, bool) {
	if g == nil || g.P == nil {
		return nil, false
	}
	bp := g.ce().Blueprints[name]
	if bp == nil {
		return nil, false
	}
	if !g.ceSpendScrapOrBits(bp.BitCost) {
		g.Msg("Not enough scrap bits to craft " + name + ".")
		return nil, false
	}
	it := g.ceBuildInventionItem(bp)
	g.P.Inv = append(g.P.Inv, it)
	g.Msg(fmt.Sprintf("{{C|You craft %s from your saved blueprint.}}", name))
	return it, true
}

// ---- 2. Enchanting & Cursed Items ------------------------------------------

func (g *Game) ceEnchant(it *Item, runeName string) string {
	if it == nil || strings.TrimSpace(runeName) == "" {
		return ""
	}
	runeName = strings.ToLower(strings.TrimSpace(runeName))
	if it.Extra == nil {
		it.Extra = map[string]int{}
	}
	flagKey := "ce_ench:" + runeName
	it.Extra[flagKey] = 1
	switch runeName {
	case "fire", "frost", "shock", "acid", "poison":
		it.Elem = runeName
		it.PV++
		it.DB++
	case "ward", "bastion", "armor", "av":
		it.AV += 2
		it.DV++
	default:
		it.PV++
		it.DB++
		it.AV++
	}
	it.Value += 35
	if !strings.Contains(it.Effect, flagKey) {
		if it.Effect == "" {
			it.Effect = flagKey
		} else {
			it.Effect += " " + flagKey
		}
	}
	msg := fmt.Sprintf("Runed %s with %s (+1 PV/DB/AV).", it.Name, flagKey)
	if g != nil {
		g.Msg("{{M|" + msg + "}}")
	}
	return msg
}

func (g *Game) ceCurseItem(it *Item) {
	if it == nil {
		return
	}
	if it.Extra == nil {
		it.Extra = map[string]int{}
	}
	if it.Extra["cursed"] == 0 {
		it.PV += 3
		it.DB += 3
		it.AV += 3
	}
	it.Extra["cursed"] = 1
	if !strings.Contains(it.Effect, "cursed") {
		if it.Effect == "" {
			it.Effect = "cursed"
		} else {
			it.Effect += " cursed"
		}
	}
	if g != nil {
		g.Msg(fmt.Sprintf("{{R|A dark curse binds %s!}}", it.Name))
	}
}

func (g *Game) ceCanUnequip(it *Item) bool {
	if it == nil {
		return true
	}
	if it.Extra != nil && it.Extra["cursed"] > 0 {
		return false
	}
	if strings.Contains(it.Effect, "cursed") {
		return false
	}
	return true
}

func (g *Game) ceUncurse(it *Item) bool {
	if it == nil || g.ceCanUnequip(it) {
		return false
	}
	if it.Extra != nil {
		delete(it.Extra, "cursed")
	}
	it.Effect = strings.TrimSpace(strings.ReplaceAll(it.Effect, "cursed", ""))
	if g != nil {
		g.Msg(fmt.Sprintf("{{G|The binding curse on %s dissolves into mist.}}", it.Name))
	}
	return true
}

// ---- 3. Fishing & Diving as Sports -----------------------------------------

func (g *Game) ceNearWater() bool {
	if g == nil || g.Cur == nil || g.P == nil {
		return false
	}
	if g.Cur.Underwater {
		return true
	}
	for dy := -1; dy <= 1; dy++ {
		for dx := -1; dx <= 1; dx++ {
			x, y := g.P.X+dx, g.P.Y+dy
			if g.Cur.In(x, y) {
				t := g.Cur.At(x, y)
				if g.Cur.WaterAt(x, y) > 0 || t == TShallow || t == TDeep {
					return true
				}
			}
		}
	}
	return false
}

func (g *Game) ceFish() (*Item, string) {
	if !g.ceNearWater() {
		return nil, "You must stand beside water to cast a fishing line."
	}
	g.Turn += 10
	sr := NewRng(fmt.Sprintf("%s:ce:fish:%d", g.W.Seed, g.Turn))
	catches := []struct {
		name  string
		kind  string
		val   int
		minLb int
		maxLb int
	}{
		{"river trout", "food", 10, 3, 12},
		{"golden carp", "food", 24, 6, 22},
		{"abyssal eel", "food", 32, 10, 35},
		{"leviathan fry", "food", 55, 20, 60},
		{"sunken chest", "misc", 75, 15, 40},
	}
	bonus := g.Skills["survival"]*2 + maxi(0, g.P.M(AGI))
	idx := clampi(sr.Int(0, len(catches)-1)+bonus/6, 0, len(catches)-1)
	c := catches[idx]
	lbs := sr.Int(c.minLb, c.maxLb) + maxi(0, bonus)
	st := g.ce()
	if lbs > st.BestFishLbs {
		st.BestFishLbs = lbs
	}
	it := &Item{
		ID:     g.newID(),
		Name:   c.name,
		Glyph:  '%',
		F:      CCyan,
		Kind:   c.kind,
		Effect: "meal",
		Power:  20 + lbs,
		Count:  1,
		Value:  c.val,
		Weight: maxi(1, lbs/5),
		Ident:  true,
		Extra:  map[string]int{"ce_lbs": lbs},
	}
	g.P.Inv = append(g.P.Inv, it)
	msg := fmt.Sprintf("You land a %d-lb %s! (Best catch: %d lbs)", lbs, c.name, st.BestFishLbs)
	g.Msg("{{C|" + msg + "}}")
	return it, msg
}

func (g *Game) ceFreeDive() (*Item, string) {
	if !g.ceNearWater() {
		return nil, "You need deep water to free-dive."
	}
	g.Turn += 10
	sr := NewRng(fmt.Sprintf("%s:ce:dive:%d", g.W.Seed, g.Turn))
	endBonus := g.Skills["endurance"]*8 + maxi(0, g.P.M(TOU))*4
	depth := 12 + sr.Int(4, 28) + endBonus
	st := g.ce()
	if depth > st.MaxDiveDepth {
		st.MaxDiveDepth = depth
	}
	prizes := []struct {
		name string
		val  int
	}{
		{"pearl oyster", 35},
		{"sunken relic", 65},
		{"luminous abyssal pearl", 90},
	}
	pr := prizes[sr.Int(0, len(prizes)-1)]
	it := &Item{
		ID:     g.newID(),
		Name:   pr.name,
		Glyph:  '*',
		F:      CCyan,
		Kind:   "misc",
		Count:  1,
		Value:  pr.val,
		Weight: 1,
		Ident:  true,
		Extra:  map[string]int{"ce_depth": depth},
	}
	g.P.Inv = append(g.P.Inv, it)
	msg := fmt.Sprintf("You free-dive to %d fathoms and surface with a %s! (Record: %d fathoms)", depth, pr.name, st.MaxDiveDepth)
	g.Msg("{{C|" + msg + "}}")
	return it, msg
}

// ---- 4. Cartography, Quest Arrow & Photo Mode ------------------------------

func (g *Game) ceChartZone() (*Item, string) {
	if g == nil || g.Cur == nil || g.P == nil {
		return nil, "No zone to chart."
	}
	z := g.Cur
	walkable := 0
	seenWalkable := 0
	for y := 0; y < MH; y++ {
		for x := 0; x < MW; x++ {
			if z.Walkable(x, y) {
				walkable++
				if z.Seen[y*MW+x] != 0 {
					seenWalkable++
				}
			}
		}
	}
	if walkable > 0 && seenWalkable*100/walkable < 50 {
		msg := fmt.Sprintf("You have only surveyed %d%% of %s (need at least 50%%).", seenWalkable*100/walkable, z.Title())
		g.Msg(msg)
		return nil, msg
	}
	st := g.ce()
	st.ChartedZones[z.Key] = true
	title := nonEmpty(z.Title(), z.Key)
	it := &Item{
		ID:     g.newID(),
		Name:   "Survey Map of " + title,
		Glyph:  '?',
		F:      CYellow,
		Kind:   "misc",
		Count:  1,
		Value:  65,
		Weight: 1,
		Ident:  true,
		Extra:  map[string]int{"ce_map": 1},
		Lore:   fmt.Sprintf("A hand-drawn cartographic survey of %s (%d,%d).", title, z.WX, z.WY),
	}
	g.P.Inv = append(g.P.Inv, it)
	msg := fmt.Sprintf("You complete a Survey Map of %s (worth 65 gold).", title)
	g.Msg("{{Y|" + msg + "}}")
	return it, msg
}

func (g *Game) ceSetQuestTarget(wx, wy int) {
	st := g.ce()
	st.TargetWX = wx
	st.TargetWY = wy
	st.HasWaypoint = true
}

func (g *Game) ceActiveQuestCoords() (int, int, bool) {
	if g == nil {
		return 0, 0, false
	}
	st := g.ce()
	if st.HasWaypoint {
		return st.TargetWX, st.TargetWY, true
	}
	sp := g.qcSpine()
	if sp.Started && !sp.Finished && sp.Step >= 0 && sp.Step < len(sp.Chaps) {
		p := sp.Chaps[sp.Step].P
		if p > 0 && g.W != nil && g.W.W > 0 {
			return p % g.W.W, p / g.W.W, true
		}
	}
	if m := g.mr().Mission; m != nil && (m.WX != 0 || m.WY != 0) {
		return m.WX, m.WY, true
	}
	return 0, 0, false
}

func (g *Game) ceQuestArrow() string {
	tx, ty, ok := g.ceActiveQuestCoords()
	if !ok {
		return ""
	}
	dx := tx - g.WX
	dy := ty - g.WY
	dist := int(math.Round(math.Hypot(float64(dx), float64(dy))))
	if dx == 0 && dy == 0 {
		return "{{Y|Quest: ◎ 0p}}"
	}
	ang := math.Atan2(float64(dy), float64(dx)) * 180 / math.Pi
	// 8-way compass arrows: E=0, SE=45, S=90, SW=135, W=180/-180, NW=-135, N=-90, NE=-45
	var arrow string
	switch {
	case ang >= -22.5 && ang < 22.5:
		arrow = "→"
	case ang >= 22.5 && ang < 67.5:
		arrow = "↘"
	case ang >= 67.5 && ang < 112.5:
		arrow = "↓"
	case ang >= 112.5 && ang < 157.5:
		arrow = "↙"
	case ang >= 157.5 || ang < -157.5:
		arrow = "←"
	case ang >= -157.5 && ang < -112.5:
		arrow = "↖"
	case ang >= -112.5 && ang < -67.5:
		arrow = "↑"
	default:
		arrow = "↗"
	}
	return fmt.Sprintf("{{Y|Quest: %s %dp}}", arrow, dist)
}

func (g *Game) cePhotoMode(caption string) string {
	if g == nil || g.Cur == nil || g.P == nil {
		return ""
	}
	z := g.Cur
	w := 24
	h := 9
	x0 := clampi(g.P.X-w/2, 0, maxi(0, MW-w))
	y0 := clampi(g.P.Y-h/2, 0, maxi(0, MH-h))

	var sb strings.Builder
	border := "+" + strings.Repeat("-", w) + "+\n"
	sb.WriteString(border)
	for y := y0; y < y0+h && y < MH; y++ {
		sb.WriteByte('|')
		for x := x0; x < x0+w && x < MW; x++ {
			ch := z.At(x, y).D().G
			if a := z.ActorAt(x, y); a != nil {
				ch = a.Glyph
			} else if its := z.ItemsAt(x, y); len(its) > 0 {
				ch = its[0].Glyph
			}
			if ch < 32 || ch > 126 {
				ch = '.'
			}
			sb.WriteRune(ch)
		}
		sb.WriteString("|\n")
	}
	sb.WriteString(border)
	header := fmt.Sprintf("%s @ %s (%d,%d) - Day %d", g.P.Name, nonEmpty(z.Title(), z.Key), g.WX, g.WY, g.Day()+1)
	sb.WriteString(header + "\n")
	if strings.TrimSpace(caption) != "" {
		sb.WriteString("\"" + strings.TrimSpace(caption) + "\"\n")
	}
	out := sb.String()
	g.ce().Photos = append(g.ce().Photos, out)
	return out
}
