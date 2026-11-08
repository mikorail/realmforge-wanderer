package engine

import (
	"fmt"
	"sort"
	"strings"
)

// Beast-crafting (bc): animal materials from hunting become typed items with a quality and a tag,
// turned into armour, weapons, tonics, lures, trophies and mount gear at a campfire, at home or at a
// tanner's rack. State in g.ext("bc"). Quality and tag are pure functions of the species.

type bcState struct {
	Known  map[string]bool `json:"known"`
	Made   map[string]int  `json:"made"`
	Saddle int             `json:"saddle,omitempty"`
	Bard   int             `json:"bard,omitempty"`
	Mount  string          `json:"mount,omitempty"`
	Sold   int             `json:"sold,omitempty"`

	stage, sel int
	rec        *bcRecipe
	picks      []*Item
	tanner     bool
	note       string
}

func (g *Game) bc() *bcState {
	st := g.ext("bc", func() any { return &bcState{} }).(*bcState)
	if st.Known == nil {
		st.Known = map[string]bool{}
	}
	if st.Made == nil {
		st.Made = map[string]int{}
	}
	return st
}

func init() {
	modeKeys["bc"] = bcKeys
	modeDraws["bc"] = bcDraw
	zoneHooks = append(zoneHooks, bcZone)
	talkHooks = append(talkHooks, bcTalk)
	killHooks = append(killHooks, bcKill)
	turnHooks = append(turnHooks, bcTurn)
	enterHooks = append(enterHooks, bcEnter)
}

// ---- materials -------------------------------------------------------------

// bcMaterial identifies a beast material item (hunting drops and bcKill drops).
func bcMaterial(it *Item) (*bcMatDef, *bsSp) {
	if it == nil || it.Kind != "trophy" || !strings.HasPrefix(it.Effect, "bsp:") {
		return nil, nil
	}
	sp := bsByID[strings.TrimPrefix(it.Effect, "bsp:")]
	if sp == nil {
		return nil, nil
	}
	n := strings.ToLower(it.Name)
	for i := range bcMats {
		for _, suf := range bcMats[i].Suffix {
			if strings.HasSuffix(n, " "+suf) {
				return &bcMats[i], sp
			}
		}
	}
	return nil, nil
}

func bcLostWorld(sp *bsSp) bool {
	switch sp.Era {
	case "mesozoic", "paleozoic", "precambrian":
		return true
	}
	return false
}

// bcQual is the quality 1..5 of a species' materials: size, rarity and age.
func bcQual(sp *bsSp) int {
	score := sp.Size / 2
	score += clampi(sp.Rarity-1, 0, 4)
	switch sp.Era {
	case "pleistocene", "cenozoic":
		score++
	case "mesozoic", "paleozoic", "precambrian":
		score += 2
	}
	return clampi(1+score/2, 1, 5)
}

// bcTagFor is the tag a species gives a material.
func bcTagFor(sp *bsSp, mat *bcMatDef) string {
	if sp.Rarity >= 4 || bcLostWorld(sp) {
		return bcUniqueTags[HashStr("bc:"+sp.ID)%uint64(len(bcUniqueTags))]
	}
	txt := strings.ToLower(sp.Name + " " + sp.Desc + " " + sp.Body)
	switch {
	case sp.Has("poison"):
		return "venomous"
	case sp.Has("shock"):
		return "storm"
	case sp.Has("acid"):
		return "caustic"
	case sp.Ice || len(sp.Biomes) > 0 && sp.Biomes[0] >= 9 && sp.Biomes[0] <= 11:
		return "rime"
	case strings.Contains(txt, "fire") || strings.Contains(txt, "flame") || strings.Contains(txt, "magma") || strings.Contains(txt, "ember") || strings.Contains(txt, "volcan"):
		return "ember"
	case sp.Has("regen"):
		return "mending"
	case sp.Has("fly") && (mat.ID == "feather" || mat.ID == "bone" || mat.ID == "hide"):
		return "light"
	case sp.Has("armored") || sp.Size >= 6:
		return "sturdy"
	}
	switch mat.ID {
	case "hide":
		if sp.Size >= 5 {
			return "sturdy"
		}
		return "warm"
	case "pelt", "fat":
		return "warm"
	case "horn", "ivory":
		return "sharp"
	case "feather":
		return "light"
	case "sinew":
		return "supple"
	case "bone", "shell", "scale", "carapace":
		return "sturdy"
	case "organ", "gland", "ichor":
		return "mending"
	}
	return "keen"
}

// bcInfo is (material, species, quality, tag) of a pack item; mat is nil for other items.
func bcInfo(it *Item) (mat *bcMatDef, sp *bsSp, q int, tag string) {
	mat, sp = bcMaterial(it)
	if mat == nil {
		return
	}
	return mat, sp, bcQual(sp), bcTagFor(sp, mat)
}

func bcStamp(it *Item) {
	mat, sp, q, tag := bcInfo(it)
	if mat == nil || it.Extra["bcq"] > 0 {
		return
	}
	if it.Extra == nil {
		it.Extra = map[string]int{}
	}
	it.Extra["bcq"] = q
	td := bcTags[tag]
	it.Lore = fmt.Sprintf("%s %s of %s: %s, %s.", bcQualWord[q], mat.Label, strings.ToLower(sp.Display()), tag, td.Desc)
	if td.Unique {
		it.F = CMagenta
		it.Value = it.Value*3/2 + 10*q
	}
}

func bcTurn(g *Game) {
	if g.P == nil {
		return
	}
	for _, it := range g.P.Inv {
		bcStamp(it)
	}
}

func bcEnter(g *Game, zn *Zone) {
	bcTurn(g)
	if g.homeHere() == nil {
		return
	}
	n := 0
	for _, it := range g.homeSt().Stash {
		if it.Extra["bctroph"] > 0 {
			n++
		}
	}
	if n == 0 {
		return
	}
	g.P.AddSt("regen", 15+10*mini(n, 4))
	g.Msg("{{G|Mounted heads watch from the walls. You feel steadied.}}")
}

func bcLabel(it *Item) string {
	mat, _, q, tag := bcInfo(it)
	if mat == nil {
		return it.Display()
	}
	col := "K"
	if bcTags[tag].Unique {
		col = "M"
	}
	return fmt.Sprintf("%s {{%s|[%s, %s]}}", it.Name, col, bcQualWord[q], tag)
}

// ---- drops: sinew, fat, glands, ichor, organs --------------------------------

func bcKill(g *Game, d, src *Actor) {
	if d == nil || g.Cur == nil || !strings.HasPrefix(d.Tpl, "bs_") {
		return
	}
	if src != g.P && (src == nil || src.AI != "follower") {
		return
	}
	sp := bsByTpl[d.Tpl]
	if sp == nil {
		return
	}
	r := NewRng(fmt.Sprintf("%s:bc:drop:%d", g.Seed, d.ID))
	low := strings.ToLower(sp.Display())
	sz := sp.Size
	add := func(what string, val int, col byte) {
		it := &Item{ID: g.newID(), Kind: "trophy", Name: low + " " + what, Glyph: '~', F: col, Value: val, Weight: 1, Effect: "bsp:" + sp.ID, X: d.X, Y: d.Y}
		g.Cur.Items = append(g.Cur.Items, it)
	}
	switch sp.Group {
	case "mam", "rept", "dino", "amph":
		if sz >= 3 {
			add("sinew", 2+sz, CGray)
		}
	}
	if (sp.Group == "mam" && !sp.Aquatic() || sp.Group == "fish") && sz >= 3 {
		add("fat", 2+sz, CWhite)
	}
	if sp.Has("poison") {
		add("venom gland", 12+5*sz+sp.Rarity*8, CGreen)
	}
	if (sp.Rarity >= 4 || bcLostWorld(sp) || sp.Has("acid") || sp.Has("shock") || sp.Has("regen")) && r.Chance(0.7) {
		add("ichor", 20+sp.Rarity*15, CMagenta)
	}
	if sz >= 4 && r.Chance(0.6) {
		add("organs", 8+3*sz, CDarkRed)
	}
}

// ---- recipes ---------------------------------------------------------------

func bcSlotHas(sl bcSlot, matID string) bool {
	return strings.Contains(" "+sl.Mats+" ", " "+matID+" ")
}

func (g *Game) bcRank() int {
	return clampi(maxi(g.Skills["tinkering"], g.Skills["survival"]), 0, 3)
}

func (g *Game) bcKnows(rc *bcRecipe) bool { return rc.Basic || g.bc().Known[rc.ID] }

// bcCandidates are the pack items that fit a slot and are not already picked.
func (g *Game) bcCandidates(sl bcSlot, picked []*Item) []*Item {
	var out []*Item
	for _, it := range g.P.Inv {
		mat, _ := bcMaterial(it)
		if mat == nil || !bcSlotHas(sl, mat.ID) {
			continue
		}
		used := false
		for _, p := range picked {
			used = used || p == it
		}
		if !used {
			out = append(out, it)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		_, _, a, _ := bcInfo(out[i])
		_, _, b, _ := bcInfo(out[j])
		return a > b
	})
	return out
}

func (g *Game) bcCanMake(rc *bcRecipe) bool {
	var picked []*Item
	for _, sl := range rc.Slots {
		c := g.bcCandidates(sl, picked)
		if len(c) == 0 {
			return false
		}
		picked = append(picked, c[0])
	}
	return true
}

func bcMergeBonus(it *Item, b map[int]int) {
	for k, v := range b {
		addBonus(it, k, v)
	}
}

// bcBuild makes the crafted item for quality q from the picked materials.
func (g *Game) bcBuild(rc *bcRecipe, picks []*Item, q int) *Item {
	mat, sp, _, tag := bcInfo(picks[0])
	for _, p := range picks {
		if _, _, _, t := bcInfo(p); bcTags[t].Unique {
			tag = t
			break
		}
	}
	td := bcTags[tag]
	q = clampi(q, 1, 5)
	low := strings.ToLower(sp.Display())
	name := fmt.Sprintf("%s %s %s %s", bcQualWord[q], tag, low, rc.Noun)
	r := NewRng(fmt.Sprintf("%s:bc:build:%s:%d", g.Seed, rc.ID, q))
	var it *Item
	switch rc.Cat {
	case "armor":
		it = g.MakeArmor(rc.Slot, clampi(q-1, 0, 4), r)
		it.AV += mat.AV + td.AV
		it.DV += mat.DV + td.DV
		it.Weight = maxi(1, it.Weight+mat.Weight)
		bcMergeBonus(it, td.ABonus)
	case "weapon":
		tier := clampi(q-1, 0, 4)
		if mat.ID == "ivory" {
			tier = clampi(tier+1, 0, 5)
		}
		if rc.Slot == "ranged" {
			it = g.MakeRanged(tier, r)
		} else {
			it = g.MakeWeapon(rc.Slot, tier, r)
		}
		it.PV += td.PV
		it.DB += td.DB
		if td.Elem != "" {
			it.Elem = td.Elem
		}
		bcMergeBonus(it, td.WBonus)
	case "tonic":
		it = g.MakeTonic(rc.Slot, 1+q/2+td.Extra)
		name = fmt.Sprintf("%s %s", bcQualWord[q], it.Name)
		if td.Unique {
			name = fmt.Sprintf("%s %s %s", bcQualWord[q], tag, it.Name)
		}
		it.Value = 15 + 6*q
	case "bait":
		bonus := clampi((q+1)/2+boolInt(td.Extra > 0 && q >= 4), 1, 3)
		it = &Item{ID: g.newID(), Kind: "food", Glyph: '%', F: CYellow, Count: 1, Value: 4 + 3*q, Power: 200,
			Extra: map[string]int{"bcbait": bonus}}
		name = fmt.Sprintf("%s %s %s", bcQualWord[q], low, rc.Noun)
	case "trophy":
		it = &Item{ID: g.newID(), Kind: "trophy", Glyph: '&', F: CYellow, Weight: 4, Value: 40 + 30*q + sp.Rarity*30,
			Extra: map[string]int{"bctroph": q}}
		name = fmt.Sprintf("mounted %s head (%s)", low, bcQualWord[q])
	case "saddle":
		it = &Item{ID: g.newID(), Kind: "trophy", Glyph: '&', F: CBrown, Weight: 5, Value: 50 + 25*q, Extra: map[string]int{}}
		if rc.Slot == "barding" {
			it.Extra["bcbard"] = q
		} else {
			it.Extra["bcsaddle"] = q
		}
	}
	it.Name = name
	if rc.Cat == "armor" || rc.Cat == "weapon" {
		it.Rarity = bcQualRarity[q]
		it.F = RarityColor(it.Rarity)
		it.Value = it.Value*(2+q)/2 + 6
	}
	if it.Extra == nil {
		it.Extra = map[string]int{}
	}
	it.Extra["bc"] = q
	it.Lore = fmt.Sprintf("Made of %s %s, %s.", low, mat.Label, td.Desc)
	return it
}

func bcStats(it *Item) string {
	var p []string
	if it.AV != 0 {
		p = append(p, fmt.Sprintf("AV %d", it.AV))
	}
	if it.DV != 0 {
		p = append(p, fmt.Sprintf("DV %+d", it.DV))
	}
	if it.Kind == "weapon" || it.Kind == "ranged" {
		p = append(p, fmt.Sprintf("%dd%d", it.DN, it.DS), fmt.Sprintf("PV %d", it.PV))
	}
	if it.Elem != "" {
		p = append(p, it.Elem)
	}
	var ks []int
	for k := range it.Bonus {
		ks = append(ks, k)
	}
	sort.Ints(ks)
	for _, k := range ks {
		if k >= 0 && k < len(AttrNames) {
			p = append(p, fmt.Sprintf("%s %+d", AttrNames[k], it.Bonus[k]))
		}
	}
	switch {
	case it.Extra["bcbait"] > 0:
		p = append(p, fmt.Sprintf("taming +%d", it.Extra["bcbait"]))
	case it.Kind == "tonic":
		p = append(p, fmt.Sprintf("x%d", maxi(1, it.Count)))
	case it.Extra["bctroph"] > 0:
		p = append(p, "home: steadying")
	case it.Extra["bcsaddle"] > 0:
		p = append(p, fmt.Sprintf("mount pace +%d%%", 8*it.Extra["bcsaddle"]))
	case it.Extra["bcbard"] > 0:
		p = append(p, fmt.Sprintf("mount stamina +%d%%", 6*it.Extra["bcbard"]))
	}
	return strings.Join(p, "  ")
}

func (g *Game) bcPreview(rc *bcRecipe, picks []*Item, q int) *Item {
	id := g.NextID
	it := g.bcBuild(rc, picks, q)
	g.NextID = id
	return it
}

func bcAvgQ(picks []*Item) int {
	t := 0
	for _, p := range picks {
		_, _, q, _ := bcInfo(p)
		t += q
	}
	return clampi((t+len(picks)/2)/maxi(1, len(picks)), 1, 5)
}

func (g *Game) bcMod(tanner bool) int {
	m := maxi(g.checkMod("tinkering"), g.checkMod("survival"))
	if tanner {
		m++
	}
	return m
}

func (g *Game) bcDC(rc *bcRecipe, picks []*Item) int { return 7 + 2*rc.Rank + bcAvgQ(picks) }

func (g *Game) bcFirst(key, text string) {
	st := g.bc()
	if st.Made["first:"+key] > 0 {
		return
	}
	st.Made["first:"+key] = 1
	g.deed("craft", fmt.Sprintf("The wanderer %s %s.", g.P.Name, text), 0, 0)
}

// bcCraft rolls the check, spends the materials and gives the item.
func (g *Game) bcCraft(rc *bcRecipe, picks []*Item) string {
	st := g.bc()
	if g.bcRank() < rc.Rank {
		return fmt.Sprintf("You need rank %d in Tinkering or Survival.", rc.Rank)
	}
	for _, p := range picks {
		found := false
		for _, o := range g.P.Inv {
			found = found || o == p
		}
		if !found {
			return "A material is no longer in your pack."
		}
	}
	q := bcAvgQ(picks)
	dc := g.bcDC(rc, picks)
	margin := g.R.Dice(2, 6) + g.bcMod(st.tanner) - dc
	g.spend(2000)
	if margin <= -6 {
		g.removeInv(picks[0])
		return "{{R|The work goes wrong and the " + picks[0].Name + " is ruined.}}"
	}
	switch {
	case margin >= 8:
		q++
	case margin >= 5:
		q += boolInt(g.R.Chance(0.5))
	case margin < 0:
		q--
	}
	q = clampi(q, 1, 5)
	for _, p := range picks {
		g.removeInv(p)
	}
	it := g.bcBuild(rc, picks, q)
	g.give(it)
	g.gainXP(6 + rc.Rank*6 + q*3)
	st.Made[rc.Cat]++
	g.bcFirst("any", "worked beast-hide and bone into gear for the first time, a "+rc.Name)
	g.bcFirst("cat:"+rc.Cat, "crafted their first "+bcCatWord(rc.Cat)+" from the spoils of the hunt")
	if q == 5 {
		g.bcFirst("peerless", "crafted a peerless "+rc.Name+" from beast materials")
	}
	if _, _, _, t := bcInfo(picks[0]); bcTags[t].Unique {
		g.bcFirst("unique", "worked the rare "+t+" materials of a legendary beast into a "+rc.Name)
	}
	return "You make " + it.Display() + "."
}

func bcCatWord(c string) string {
	switch c {
	case "armor":
		return "armour"
	case "weapon":
		return "weapon"
	case "tonic":
		return "beast tonic"
	case "bait":
		return "lure"
	case "trophy":
		return "mounted trophy"
	}
	return "mount gear"
}

// bcExperiment tries a material to discover a recipe.
func (g *Game) bcExperiment(it *Item) string {
	mat, _ := bcMaterial(it)
	if mat == nil {
		return "That is not a beast material."
	}
	var pool []*bcRecipe
	for _, rc := range bcRecipes {
		if g.bcKnows(rc) || rc.Rank > g.bcRank()+1 {
			continue
		}
		for _, sl := range rc.Slots {
			if bcSlotHas(sl, mat.ID) {
				pool = append(pool, rc)
				break
			}
		}
	}
	g.spend(1000)
	if len(pool) == 0 {
		return "You turn the " + mat.Label + " over, but learn nothing new from it."
	}
	rc := Pick(g.R, pool)
	if g.R.Dice(2, 6)+g.bcMod(false) < 8+2*rc.Rank {
		return "You cut and bind, but nothing useful comes of it. Perhaps a better hand could teach you."
	}
	g.bc().Known[rc.ID] = true
	g.bcFirst("discover", "worked out a beast-craft recipe by trial alone: "+rc.Name)
	return "{{M|By trial you work out how to make a " + rc.Name + "!}}"
}

// bcFit mounts a saddle or barding on the current mount.
func (g *Game) bcFit(it *Item) string {
	m := g.tr().Mount
	if m == nil {
		return "You have no mount to fit it to."
	}
	st := g.bc()
	if st.Mount != m.Def {
		st.Saddle, st.Bard, st.Mount = 0, 0, m.Def
	}
	if q := it.Extra["bcsaddle"]; q > 0 {
		st.Saddle = q
	}
	if q := it.Extra["bcbard"]; q > 0 {
		st.Bard = q
	}
	g.removeInv(it)
	g.spend(1000)
	g.bcFirst("fit", "fitted hand-made tack to a mount")
	return "You fit " + it.Name + " to " + m.Name + "."
}

// ---- hooks for other systems -------------------------------------------------

// bcBaitOf is the taming bonus 0..3 of one item.
func bcBaitOf(it *Item) int { return clampi(it.Extra["bcbait"], 0, 3) }

// bcBait is the best lure bonus 0..3 in your pack that this beast would take (taming.go).
func (g *Game) bcBait(a *Actor) int {
	s := bsByTpl[a.Tpl]
	if s == nil || g.P == nil {
		return 0
	}
	best := 0
	for _, it := range g.P.Inv {
		if b := bcBaitOf(it); b > best && tmFoodQ(s, it) > 0 {
			best = b
		}
	}
	return best
}

// bcBaitSpend uses up one lure that bcBait counted, unless the offered item is itself the lure
// (taming removes that one on success).
func (g *Game) bcBaitSpend(a *Actor, offered *Item) {
	if offered != nil && bcBaitOf(offered) > 0 {
		return
	}
	s := bsByTpl[a.Tpl]
	if s == nil {
		return
	}
	var pick *Item
	for _, it := range g.P.Inv {
		if b := bcBaitOf(it); b > 0 && tmFoodQ(s, it) > 0 && (pick == nil || b > bcBaitOf(pick)) {
			pick = it
		}
	}
	if pick != nil {
		g.takeOne(pick)
		g.Msg("You rub " + pick.Name + " on your hands for the scent.")
	}
}

// bcMount scales the mount pace bonus with a fitted saddle (transport.go, after tmAdjust).
func (g *Game) bcMount(mul float64) float64 {
	st := g.bc()
	m := g.tr().Mount
	if m == nil || st.Mount != m.Def || st.Saddle == 0 || mul <= 1 {
		return mul
	}
	return 1 + (mul-1)*(1+0.08*float64(st.Saddle))
}

// bcStamina multiplies the mount's maximum stamina (taming.go tmRegen).
func (g *Game) bcStamina() float64 {
	st := g.bc()
	m := g.tr().Mount
	if m == nil || st.Mount != m.Def {
		return 1
	}
	return 1 + 0.06*float64(st.Bard) + 0.03*float64(st.Saddle)
}

// ---- abilities ---------------------------------------------------------------

func (g *Game) bcWhyNot() string {
	if g.homeHere() != nil {
		return ""
	}
	return g.canCook()
}

func (g *Game) bcAbilities() []Ability {
	if g.Cur == nil || g.P == nil || g.bcWhyNot() != "" {
		return nil
	}
	return []Ability{{ID: "bc_craft", Name: "Beast-craft", Target: "self"}}
}

func (g *Game) bcUse(id string, tx, ty int) {
	if id != "bc_craft" {
		return
	}
	if why := g.bcWhyNot(); why != "" {
		g.Msg(why)
		return
	}
	g.bcOpen(false)
}

func (g *Game) bcOpen(tanner bool) {
	st := g.bc()
	bcTurn(g)
	st.stage, st.sel, st.rec, st.picks, st.tanner, st.note = 0, 0, nil, nil, tanner, ""
	g.Mode = "bc"
}

// ---- tanner NPC and talk -------------------------------------------------------

func bcZone(g *Game, zn *Zone, _ *Rng) {
	if g.W == nil || zn.Kind != "town" || zn.Z != 0 || zn.Burg < 0 {
		return
	}
	bd := g.W.burgByID[zn.Burg]
	if bd == nil {
		return
	}
	r := NewRng(g.Seed + ":bc:tanner:" + zn.Key)
	smith := false
	for _, a := range zn.Actors {
		smith = smith || a.Role == "smith"
	}
	if !smith && !r.Chance(0.12) {
		return
	}
	x, y, ok := zn.FreeSpot(r, nil)
	if !ok {
		return
	}
	cu := g.W.cultByID[g.W.Culture[clampi(bd.P, 0, len(g.W.Culture)-1)]]
	a := g.makeNPC("tanner", bd, cu, r)
	a.Title, a.F = "tanner", COrange
	a.X, a.Y, a.HomeX, a.HomeY = x, y, x, y
	a.Gold = 150 + 40*a.Level
	a.Desc = fmt.Sprintf("A tanner of %s, apron stiff with salt and old blood, with stretched hides on every rack.", bd.Name)
	zn.Actors = append(zn.Actors, a)
}

func (g *Game) bcDiscount(cost int) int {
	return maxi(1, cost*(100-g.Skills["persuasion"]*8)/100)
}

func (g *Game) bcSellPrice(it *Item) int {
	_, _, q, _ := bcInfo(it)
	p := it.Value*3/2 + 2*q
	return p * (100 + g.Skills["persuasion"]*5) / 100
}

func (g *Game) bcMaterials() []*Item {
	var out []*Item
	for _, it := range g.P.Inv {
		if m, _ := bcMaterial(it); m != nil {
			out = append(out, it)
		}
	}
	return out
}

// bcTaught is what a teacher can show you: the tanner's three, others one in their trade.
func (g *Game) bcTaught(a *Actor) []*bcRecipe {
	cats := bcRoleCats[a.Role]
	var pool []*bcRecipe
	for _, rc := range bcRecipes {
		if rc.Basic || rc.Rank > 2 {
			continue
		}
		if a.Role != "tanner" && !strings.Contains(" "+cats+" ", " "+rc.Cat+" ") {
			continue
		}
		pool = append(pool, rc)
	}
	if len(pool) == 0 {
		return nil
	}
	r := NewRng(g.Seed + ":bc:teach:" + a.Name + ":" + a.Role)
	for i := len(pool) - 1; i > 0; i-- {
		j := r.Int(0, i)
		pool[i], pool[j] = pool[j], pool[i]
	}
	n := 1
	if a.Role == "tanner" {
		n = 3
	}
	return pool[:mini(n, len(pool))]
}

func bcTalk(g *Game, a *Actor) []Option {
	var out []Option
	if a.Role == "tanner" {
		out = append(out, Option{"{{O|Work at the tanner's rack.}}", func() { g.bcOpen(true) }})
		if mats := g.bcMaterials(); len(mats) > 0 {
			total := 0
			for _, it := range mats {
				total += g.bcSellPrice(it)
			}
			out = append(out, Option{fmt.Sprintf("{{W|Sell my beast materials.}} {{K|(%d items, %d %s)}}", len(mats), total, g.Sk.Currency), func() {
				got := 0
				for _, it := range g.bcMaterials() {
					got += g.bcSellPrice(it)
					g.removeInv(it)
				}
				g.P.Gold += got
				g.bc().Sold++
				g.bcFirst("sell", "sold a pack of beast materials to a tanner")
				g.talkMenu(fmt.Sprintf("\"Good stock. I pay well for clean work.\" You are paid %d %s.", got, g.Sk.Currency))
			}})
		}
	}
	for _, rc := range g.bcTaught(a) {
		rc := rc
		if g.bcKnows(rc) {
			continue
		}
		cost := g.bcDiscount(30 + 25*rc.Rank)
		out = append(out, Option{fmt.Sprintf("{{O|Teach me to make a %s.}} {{K|(%d %s)}}", rc.Name, cost, g.Sk.Currency), func() {
			if g.P.Gold < cost {
				g.talkMenu("\"Craft is not free.\"")
				return
			}
			g.P.Gold -= cost
			g.bc().Known[rc.ID] = true
			g.talkMenu("\"Watch my hands, and mind the sinew.\" You learn to make a " + rc.Name + ".")
		}})
	}
	return out
}

// ---- screen ----------------------------------------------------------------------

type bcEntry struct {
	label string
	do    func()
}

func (g *Game) bcEntries() []bcEntry {
	st := g.bc()
	var out []bcEntry
	switch st.stage {
	case 0:
		for _, rc := range bcRecipes {
			rc := rc
			if !g.bcKnows(rc) {
				continue
			}
			mark := "{{R|missing materials}}"
			if g.bcCanMake(rc) {
				mark = "{{G|ready}}"
			}
			lock := ""
			if rc.Rank > g.bcRank() {
				lock = fmt.Sprintf(" {{R|rank %d}}", rc.Rank)
			}
			out = append(out, bcEntry{fmt.Sprintf("%-18s {{K|%s}}  %s%s", rc.Name, rc.Cat, mark, lock), func() {
				st.rec, st.picks, st.stage, st.sel = rc, nil, 1, 0
			}})
		}
		out = append(out, bcEntry{"{{C|Experiment with a material...}}", func() { st.stage, st.sel = 3, 0 }})
		if g.tr().Mount != nil {
			for _, it := range g.P.Inv {
				it := it
				if it.Extra["bcsaddle"] > 0 || it.Extra["bcbard"] > 0 {
					out = append(out, bcEntry{"{{Y|Fit}} " + it.Name, func() { st.note = g.bcFit(it); st.sel = 0 }})
				}
			}
		}
	case 1:
		for _, it := range g.bcCandidates(st.rec.Slots[len(st.picks)], st.picks) {
			it := it
			out = append(out, bcEntry{bcLabel(it), func() {
				st.picks = append(st.picks, it)
				st.sel = 0
				if len(st.picks) >= len(st.rec.Slots) {
					st.stage = 2
				}
			}})
		}
	case 3:
		for _, it := range g.bcMaterials() {
			it := it
			out = append(out, bcEntry{bcLabel(it), func() { st.note = g.bcExperiment(it); st.stage, st.sel = 0, 0 }})
		}
	}
	return out
}

func bcKeys(g *Game, k string) {
	st := g.bc()
	es := g.bcEntries()
	switch k {
	case "Escape":
		switch st.stage {
		case 0:
			if st.tanner && g.Talk != nil {
				g.Mode = "talk"
				g.talkMenu("\"Anything else?\"")
			} else {
				g.Mode = "main"
			}
		case 1:
			if len(st.picks) > 0 {
				st.picks = st.picks[:len(st.picks)-1]
			} else {
				st.stage = 0
			}
			st.sel = 0
		case 2:
			st.picks, st.stage, st.sel = st.picks[:len(st.picks)-1], 1, 0
		default:
			st.stage, st.sel = 0, 0
		}
	case "ArrowUp":
		st.sel = maxi(0, st.sel-1)
	case "ArrowDown":
		st.sel = mini(maxi(0, len(es)-1), st.sel+1)
	case "Enter", " ":
		if st.stage == 2 {
			msg := g.bcCraft(st.rec, st.picks)
			st.note = msg
			g.Msg(msg)
			st.stage, st.rec, st.picks, st.sel = 0, nil, nil, 0
			return
		}
		if st.sel < len(es) {
			st.note = ""
			es[st.sel].do()
		}
	}
}

func bcDraw(g *Game) {
	st := g.bc()
	x, y, _, h := g.panel("Beast-craft")
	s := g.sc
	where := "at the fire"
	if st.tanner {
		where = "at the tanner's rack (+1)"
	} else if g.homeHere() != nil {
		where = "at home"
	}
	s.Text(x, y, fmt.Sprintf("{{C|Skill rank %d}}  %s  {{K|%d materials in your pack}}", g.bcRank(), where, len(g.bcMaterials())), CGray)
	switch st.stage {
	case 2:
		it := g.bcPreview(st.rec, st.picks, bcAvgQ(st.picks))
		dc := g.bcDC(st.rec, st.picks)
		mod := g.bcMod(st.tanner)
		s.Text(x, y+2, "{{W|"+st.rec.Name+"}}", CGray)
		s.Text(x, y+4, "Expected: "+it.Name, CGray)
		s.Text(x, y+5, "{{K|"+bcStats(it)+"}}", CGray)
		s.Text(x, y+7, fmt.Sprintf("Materials: %s", strings.Join(bcNames(st.picks), ", ")), CGray)
		s.Text(x, y+9, fmt.Sprintf("%s  {{K|chance of a clean job %d%%; a good roll raises the quality, a poor one lowers it}}", g.checkLabel("tinkering", dc), int(chanceFor(mod, dc)*100+0.5)), CGray)
		s.Text(x, y+h-1, "{{K|Enter: craft   Esc: back}}", CGray)
		return
	case 1:
		sl := st.rec.Slots[len(st.picks)]
		s.Text(x, y+2, fmt.Sprintf("{{W|%s}}  choose material %d of %d: {{Y|%s}}", st.rec.Name, len(st.picks)+1, len(st.rec.Slots), strings.Join(strings.Fields(sl.Mats), " / ")), CGray)
	case 3:
		s.Text(x, y+2, "{{W|Experiment}}  choose a material to try", CGray)
	default:
		s.Text(x, y+2, "{{W|Recipes}}  known recipes; talk to tanners and trades to learn more", CGray)
	}
	es := g.bcEntries()
	rows := h - 8
	top := maxi(0, st.sel-rows+1)
	for i := top; i < len(es) && i < top+rows; i++ {
		mark := "  "
		if i == st.sel {
			mark = "{{Y|>}} "
		}
		s.Text(x, y+4+i-top, mark+es[i].label, CGray)
	}
	if len(es) == 0 {
		s.Text(x, y+4, "{{K|(nothing here; hunt beasts for hide, bone, horn and more)}}", CGray)
	}
	if st.note != "" {
		s.Text(x, y+h-2, st.note, CGray)
	}
	s.Text(x, y+h-1, "{{K|Up/Down: select   Enter: choose   Esc: back}}", CGray)
}

func bcNames(its []*Item) []string {
	var out []string
	for _, it := range its {
		out = append(out, it.Name)
	}
	return out
}
