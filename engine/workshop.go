package engine

import (
	"fmt"
	"sort"
	"strings"
)

// Crafting mastery and workshops (prefix wk). Sits beside crafting.go (T workbench, bits, schematics, mods),
// cooking.go, alchemy.go and beastcraft.go and does not change them. It adds:
//   - quality tiers (poor, fair, fine, masterwork) on gear made here or by the T workbench ("sig" items are
//     picked up by polling), decided by a 2d6 craft check, the line's rank and the tools at hand. Quality is
//     written into fields the engine already reads (DB, PV, AV, DV, Value) and tagged in Item.Extra "wkq";
//   - durability and repair: worn gear loses Extra "wkdur" (turn hook polling, kill hook); at zero the item
//     is "broken": its DB, PV, AV, DV and attribute bonuses are lifted into Extra stashes and put back by a
//     repair, so existing code needs no change; smiths, tailors and tanners repair for a price, a workshop
//     for a skill check;
//   - workshops: your home bench (tool tiers bought up), town smithies, kitchens and shops for hire by the
//     hour, dungeon kitchens, forges and labs (aa_purposerooms) free but crude; tools add to craft checks;
//   - apprenticeship: named masters (lives people with a craft trade) teach five techniques per line after
//     an errand or for a fee, up to their own rank (2 to 5);
//   - regional recipes: a culture's signature piece per line, learned from its local master.
// Screen: mode "wk" (palette "Workshop"). State: g.ext("wk") global (ranks, recipes), g.ext("wkl") planet-local
// (hires, home tools, errands). Generation uses side rngs only; g.R only rolls play-time outcomes.

func init() {
	unLocalKeys = append(unLocalKeys, "wkl")
	turnHooks = append(turnHooks, wkTurn)
	killHooks = append(killHooks, wkKill)
	talkHooks = append(talkHooks, wkTalk)
	modeKeys["wk"] = wkKeys
	modeDraws["wk"] = func(g *Game) { g.wkDraw() }
	paEntries = append(paEntries, paEntry{Name: "Workshop", Keywords: "craft smith repair master apprentice recipes quality tools bench forge mastery durability", Cat: "Screen", Label: "screen",
		Func: func(g *Game) { g.wkOpen(0) }})
	achDefs = append(achDefs,
		achDef{"wk_fine", "Fine Hands", "Make something of fine quality.", "Craft", "", 1, func(g *Game, st *acState) int { return wkSt(g).Fine }},
		achDef{"wk_master_work", "Masterwork", "Make a masterwork piece.", "Craft", "", 1, func(g *Game, st *acState) int { return wkSt(g).Master }},
		achDef{"wk_journeyman", "Journeyman", "Reach the third rank in a craft line.", "Craft", "", 3, func(g *Game, st *acState) int { return wkBestRank(g) }},
		achDef{"wk_the_master", "Master Craftsperson", "Reach master rank in a craft line.", "Craft", "Master Craftsperson", 5, func(g *Game, st *acState) int { return wkBestRank(g) }},
		achDef{"wk_mender", "Mender", "Repair twenty pieces of gear.", "Craft", "", 20, func(g *Game, st *acState) int { return wkSt(g).Repairs }},
	)
}

// ---------------------------------------------------------------- state

type wkState struct {
	Rank    map[string]int  `json:"r"`
	XP      map[string]int  `json:"x"`
	Recipes map[string]bool `json:"rc"`
	Made    int             `json:"m,omitempty"`
	Fine    int             `json:"fi,omitempty"`
	Master  int             `json:"ma,omitempty"`
	Repairs int             `json:"rp,omitempty"`
	Lessons int             `json:"ls,omitempty"`

	tab, sel int
	msg      string
	masters  []wkMaster
}

type wkErrand struct {
	Line string `json:"l"`
	Need int    `json:"n"`
	Due  int    `json:"d"`
}

type wkLocal struct {
	Hired     map[string]int       `json:"h,omitempty"`
	HomeTools map[string]int       `json:"ht,omitempty"`
	Errands   map[string]*wkErrand `json:"e,omitempty"`
	Taught    map[string]int       `json:"t,omitempty"`
	HP        int                  `json:"hp,omitempty"`
}

func wkSt(g *Game) *wkState {
	s := g.ext("wk", func() any { return &wkState{Rank: map[string]int{}, XP: map[string]int{}, Recipes: map[string]bool{}} }).(*wkState)
	if s.Rank == nil {
		s.Rank = map[string]int{}
	}
	if s.XP == nil {
		s.XP = map[string]int{}
	}
	if s.Recipes == nil {
		s.Recipes = map[string]bool{}
	}
	return s
}

func (g *Game) wk() *wkState { return wkSt(g) }

func (g *Game) wkl() *wkLocal {
	s := g.ext("wkl", func() any {
		return &wkLocal{Hired: map[string]int{}, HomeTools: map[string]int{}, Errands: map[string]*wkErrand{}, Taught: map[string]int{}}
	}).(*wkLocal)
	if s.Hired == nil {
		s.Hired = map[string]int{}
	}
	if s.HomeTools == nil {
		s.HomeTools = map[string]int{}
	}
	if s.Errands == nil {
		s.Errands = map[string]*wkErrand{}
	}
	if s.Taught == nil {
		s.Taught = map[string]int{}
	}
	return s
}

func wkLineByID(id string) *wkLine {
	for i := range wkLines {
		if wkLines[i].ID == id {
			return &wkLines[i]
		}
	}
	return nil
}

func (g *Game) wkRank(line string) int { return clampi(g.wk().Rank[line], 0, 5) }

func wkBestRank(g *Game) int {
	b := 0
	for _, l := range wkLines {
		b = maxi(b, g.wk().Rank[l.ID])
	}
	return b
}

func wkHas(l []string, s string) bool {
	for _, x := range l {
		if x == s {
			return true
		}
	}
	return false
}

func (g *Game) wkFirst(key, text string) {
	st := g.wk()
	k := "first:" + key
	if st.Recipes[k] {
		return
	}
	st.Recipes[k] = true
	g.deed("craft", fmt.Sprintf("The wanderer %s %s.", g.P.Name, text), 0, 0)
}

// wkTake removes n units of a pack item (a stack loses count, anything else goes).
func (g *Game) wkTake(it *Item, n int) {
	if it.Stack() && it.Count > n {
		it.Count -= n
		return
	}
	g.removeInv(it)
}

func wkUnits(it *Item) int {
	if it.Stack() {
		return maxi(1, it.Count)
	}
	return 1
}

// ---------------------------------------------------------------- stations (workshops)

type wkStation struct {
	Key, Name, Src string // Src: home, town, wild
	Lines          []string
	Tier           int
	Hire           int // base units per hour (town)
}

var wkAllLines = []string{"smith", "tailor", "alchemist", "tanner", "cook", "carpenter"}

func (g *Game) wkHostileSeen() bool {
	if g.Cur == nil || g.P == nil {
		return false
	}
	for _, a := range g.Cur.Actors {
		if a.HP > 0 && a != g.P && g.Hostile(a) && dist(a, g.P) <= 12 && g.visible(a.X, a.Y) {
			return true
		}
	}
	return false
}

func (g *Game) wkStations() []wkStation {
	z := g.Cur
	if z == nil || g.P == nil || g.OnWorld {
		return nil
	}
	l := g.wkl()
	var out []wkStation
	if h := g.homeHere(); h != nil {
		for _, id := range wkAllLines {
			t := maxi(1, l.HomeTools[z.Key+":"+id])
			out = append(out, wkStation{Key: z.Key + ":home:" + id, Name: "your " + strings.ToLower(wkLineByID(id).Name) + " bench at " + h.Name, Src: "home", Lines: []string{id}, Tier: t})
		}
		return out
	}
	// terrain scan around the player
	cnt := map[Terrain]int{}
	for y := g.P.Y - 7; y <= g.P.Y+7; y++ {
		for x := g.P.X - 7; x <= g.P.X+7; x++ {
			if z.In(x, y) {
				cnt[z.At(x, y)]++
			}
		}
	}
	town := z.Kind == "town"
	src, tier := "wild", 1
	if town {
		src, tier = "town", 2
	}
	seen := map[string]bool{}
	add := func(kind, name string, lines ...string) {
		if seen[kind] {
			return
		}
		seen[kind] = true
		s := wkStation{Key: z.Key + ":" + kind, Name: name, Src: src, Lines: lines, Tier: tier}
		if town {
			s.Hire = 6 + 4*tier + g.P.Level/2
		}
		out = append(out, s)
	}
	if cnt[TAnvil] > 0 {
		add("forge", map[bool]string{true: "the town forge", false: "an old forge"}[town], "smith", "carpenter")
	}
	if cnt[TBrazier] > 0 && cnt[TStall] > 0 {
		add("kitchen", map[bool]string{true: "a kitchen hearth", false: "an old kitchen"}[town], "cook")
	}
	if cnt[TBrazier] > 0 && cnt[TMachine] > 0 {
		add("lab", map[bool]string{true: "an alchemist's bench", false: "an old alchemy lab"}[town], "alchemist")
	}
	if town {
		for _, a := range z.Actors {
			if a.HP <= 0 || a.AI == "hostile" || a.Angry || dist(a, g.P) > 12 {
				continue
			}
			t := strings.ToLower(a.Title)
			switch {
			case a.Role == "smith":
				add("forge", "the smith's forge", "smith", "carpenter")
			case a.Role == "tailor":
				add("tailor", "a tailor's shop", "tailor")
			case a.Role == "apothecary":
				add("lab", "an apothecary's bench", "alchemist")
			case a.Role == "tanner":
				add("tannery", "a tannery rack", "tanner")
			case strings.Contains(t, "inn") || strings.Contains(t, "baker") || strings.Contains(t, "cook"):
				add("kitchen", "a kitchen hearth", "cook")
			case strings.Contains(t, "carpenter") || strings.Contains(t, "joiner"):
				add("joiner", "a joiner's shop", "carpenter")
			}
		}
	}
	return out
}

// wkTools: the best tool tier for a line right here, its label, and a station offered for hire (or nil).
func (g *Game) wkTools(line string) (tier int, label string, offer *wkStation) {
	wild := g.wkHostileSeen()
	for _, s := range g.wkStations() {
		if !wkHas(s.Lines, line) {
			continue
		}
		switch s.Src {
		case "town":
			if g.wkl().Hired[s.Key] > g.Turn {
				if s.Tier > tier {
					tier, label = s.Tier, s.Name+" (hired)"
				}
			} else {
				c := s
				offer = &c
			}
		case "wild":
			if !wild && s.Tier > tier {
				tier, label = s.Tier, s.Name
			}
		default:
			if s.Tier > tier {
				tier, label = s.Tier, s.Name
			}
		}
	}
	return
}

func (g *Game) wkHire(s *wkStation, hours int) string {
	cost := s.Hire * hours
	if !g.cuSpend(cost) {
		return "You cannot pay for it."
	}
	l := g.wkl()
	l.Hired[s.Key] = maxi(l.Hired[s.Key], g.Turn) + 60*hours
	return fmt.Sprintf("You hire %s for %d hour(s) (%s).", s.Name, hours, g.cuFormat(cost))
}

// ---------------------------------------------------------------- checks and quality

func (g *Game) wkMod(line string) int {
	l := wkLineByID(line)
	if l == nil || g.P == nil {
		return 0
	}
	tier, _, _ := g.wkTools(line)
	return g.P.M(l.Attr) + g.wkRank(line) + tier + g.Skills["tinkering"] + g.ijCheckMod("tinkering") + g.agCheckBonus("tinkering")
}

// wkRoll decides a quality q in -1..2 for a piece of the given item tier.
func (g *Game) wkRoll(line string, itemTier int) (q, roll int) {
	tier, _, _ := g.wkTools(line)
	rank := g.wkRank(line)
	roll = g.R.Dice(2, 6) + g.wkMod(line)
	sh := itemTier / 2
	switch {
	case roll >= 14+sh:
		q = 2
	case roll >= 11+sh:
		q = 1
	case roll >= 7+sh:
		q = 0
	default:
		q = -1
	}
	if q == 2 && !(rank >= 5 || rank >= 3 && tier >= 2) {
		q = 1
	}
	if q == 1 && !(tier >= 1 || rank >= 4) {
		q = 0
	}
	if q == -1 && rank >= 1 && roll >= 5+sh {
		q = 0
	}
	return
}

func wkQ(it *Item) (int, bool) {
	if it == nil || it.Extra == nil || it.Extra["wkq"] == 0 {
		return 0, false
	}
	return it.Extra["wkq"] - 2, true
}

func wkTracks(it *Item) bool {
	return it != nil && it.Quest == 0 && (it.Kind == "weapon" || it.Kind == "ranged" || it.Kind == "armor")
}

// wkApplyQuality writes quality into fields the engine reads. Safe to call once per item.
func (g *Game) wkApplyQuality(it *Item, q int, rank int) {
	if it == nil {
		return
	}
	if _, done := wkQ(it); done {
		return
	}
	q = clampi(q, -1, 2)
	if it.Extra == nil {
		it.Extra = map[string]int{}
	}
	it.Extra["wkq"] = q + 2
	switch it.Kind {
	case "weapon", "ranged":
		it.DB += q
		if q == 2 {
			it.PV++
		}
	case "armor":
		if q < 0 {
			it.AV = maxi(0, it.AV-1)
		} else {
			it.AV += q
		}
		if q == 2 {
			it.DV++
		}
	}
	if it.Value > 0 {
		it.Value = maxi(1, it.Value*wkQValue[q+1]/100)
	}
	if q != 0 && !strings.HasPrefix(it.Name, wkQWord[q+1]+" ") {
		it.Name = wkQWord[q+1] + " " + it.Name
	}
	if wkTracks(it) {
		it.Extra["wkmax"] = g.wkMaxDur(it, rank)
		it.Extra["wkdur"] = it.Extra["wkmax"]
	}
}

// ---------------------------------------------------------------- durability

func (g *Game) wkMaxDur(it *Item, rank int) int {
	q, _ := wkQ(it)
	m := 36 + 12*it.Tier + 14*q + 6*rarityIndex(it.Rarity)
	if rank >= 3 {
		m += m / 4
	}
	return maxi(16, m)
}

func (g *Game) wkEnsureDur(it *Item) {
	if !wkTracks(it) || it.Kind == "artifact" {
		return
	}
	if it.Extra == nil {
		it.Extra = map[string]int{}
	}
	if it.Extra["wkmax"] == 0 {
		it.Extra["wkmax"] = g.wkMaxDur(it, 0)
		it.Extra["wkdur"] = it.Extra["wkmax"]
	}
}

func wkBroken(it *Item) bool { return it != nil && it.Extra != nil && it.Extra["wkbrk"] > 0 }

// wkBreak lifts the item's bonuses into stashes so the engine reads a plain, useless piece.
func (g *Game) wkBreak(it *Item) {
	if it.Extra == nil {
		it.Extra = map[string]int{}
	}
	if it.Extra["wkbrk"] > 0 {
		return
	}
	it.Extra["wkbrk"] = 1
	switch it.Kind {
	case "weapon", "ranged":
		pos := maxi(it.DB, 0)
		it.Extra["wkb_DB"] = pos + 1
		it.DB -= pos + 1
		it.Extra["wkb_PV"] = it.PV
		it.PV = 0
	case "armor":
		it.Extra["wkb_AV"] = it.AV
		it.AV = 0
		pos := maxi(it.DV, 0)
		it.Extra["wkb_DV"] = pos
		it.DV -= pos
	}
	for a, v := range it.Bonus {
		if v != 0 {
			it.Extra[fmt.Sprintf("wkb_B%d", a)] = v + 100
		}
	}
	it.Bonus = nil
	if !strings.HasPrefix(it.Name, "broken ") {
		it.Name = "broken " + it.Name
	}
}

// wkMend puts the stashes back and sets the durability.
func (g *Game) wkMend(it *Item, dur int) {
	if it.Extra == nil {
		it.Extra = map[string]int{}
	}
	if it.Extra["wkbrk"] > 0 {
		it.DB += it.Extra["wkb_DB"]
		it.PV += it.Extra["wkb_PV"]
		it.AV += it.Extra["wkb_AV"]
		it.DV += it.Extra["wkb_DV"]
		for a := 0; a < 6; a++ {
			k := fmt.Sprintf("wkb_B%d", a)
			if v := it.Extra[k]; v != 0 {
				if it.Bonus == nil {
					it.Bonus = map[int]int{}
				}
				it.Bonus[a] += v - 100
				delete(it.Extra, k)
			}
		}
		for _, k := range []string{"wkb_DB", "wkb_PV", "wkb_AV", "wkb_DV", "wkbrk"} {
			delete(it.Extra, k)
		}
		it.Name = strings.TrimPrefix(it.Name, "broken ")
	}
	it.Extra["wkdur"] = clampi(dur, 1, maxi(it.Extra["wkmax"], 1))
}

func (g *Game) wkWear(it *Item, n int) {
	if !wkTracks(it) || wkBroken(it) {
		return
	}
	g.wkEnsureDur(it)
	if it.Extra["wkmax"] == 0 {
		return
	}
	prev := it.Extra["wkdur"]
	it.Extra["wkdur"] = maxi(0, prev-n)
	d, m := it.Extra["wkdur"], it.Extra["wkmax"]
	switch {
	case d <= 0:
		g.wkBreak(it)
		g.Msg("{{R|Your " + strings.TrimPrefix(it.Name, "broken ") + " breaks!}} {{K|(a smith or a workshop can mend it)}}")
	case d*4 <= m && prev*4 > m:
		g.Msg("{{y|Your " + it.Name + " is badly worn.}}")
	}
}

func (g *Game) wkEquipped() []*Item {
	var out []*Item
	if g.P == nil {
		return nil
	}
	for _, p := range g.P.Body {
		if p.Item != nil && !p.Severed && wkTracks(p.Item) && p.Item.Kind != "artifact" {
			out = append(out, p.Item)
		}
	}
	return out
}

func (g *Game) wkAllGear() []*Item {
	out := g.wkEquipped()
	for _, it := range g.P.Inv {
		if wkTracks(it) && it.Kind != "artifact" {
			out = append(out, it)
		}
	}
	return out
}

func wkMissing(it *Item) int {
	if it.Extra == nil || it.Extra["wkmax"] == 0 {
		return 0
	}
	return it.Extra["wkmax"] - it.Extra["wkdur"]
}

// wkRepairLine: which craft line mends an item.
func wkRepairLine(it *Item) string {
	n := strings.ToLower(it.Name)
	has := func(ws ...string) bool {
		for _, w := range ws {
			if strings.Contains(n, w) {
				return true
			}
		}
		return false
	}
	switch it.Kind {
	case "ranged":
		return "carpenter"
	case "armor":
		if has("leather", "hide", "cured", "cloak", "boots", "jerkin", "glove", "rawhide", "boiled", "oiled") {
			return "tanner"
		}
		if has("robe", "coat", "silk", "linen", "wool", "cloth", "quilted", "tunic", "vest", "dress", "sash", "brocade", "hood") {
			return "tailor"
		}
	}
	return "smith"
}

func (g *Game) wkRepairCost(it *Item, line string) int {
	m := wkMissing(it)
	if m <= 0 && !wkBroken(it) {
		return 0
	}
	mx := maxi(1, it.Extra["wkmax"])
	c := (it.Value/5 + 3) * m / mx
	if wkBroken(it) {
		c += 4
	}
	c = maxi(2, c)
	if g.wkRank(line) >= 5 {
		c = c * 3 / 4
	}
	return maxi(1, c)
}

// wkWorn lists gear that needs mending, optionally only what a line can mend ("" = all).
func (g *Game) wkWorn(line string) []*Item {
	var out []*Item
	for _, it := range g.wkAllGear() {
		if wkMissing(it) > 0 || wkBroken(it) {
			if line == "" || wkRepairLine(it) == line {
				out = append(out, it)
			}
		}
	}
	return out
}

// wkSelfRepair: a skill check at a workshop for the item's line. Success mends it (a little less sturdy each
// time); failure costs some of its life.
func (g *Game) wkSelfRepair(it *Item) string {
	line := wkRepairLine(it)
	tier, _, offer := g.wkTools(line)
	if tier == 0 {
		if offer != nil {
			return fmt.Sprintf("You need a workshop: %s is for hire (H on the Workshop tab).", offer.Name)
		}
		return "You need a bench or forge for that. Use your home bench, hire one in town, or find an old one."
	}
	g.spendTurns(10)
	dc := 8 + wkMissing(it)*4/maxi(1, it.Extra["wkmax"])*2 + it.Tier/2
	if wkBroken(it) {
		dc += 2
	}
	roll := g.R.Dice(2, 6) + g.wkMod(line)
	st := g.wk()
	if roll >= dc {
		newMax := maxi(16, it.Extra["wkmax"]-3)
		it.Extra["wkmax"] = newMax
		g.wkMend(it, newMax)
		st.Repairs++
		g.wkPractice(line, 4)
		return fmt.Sprintf("You mend the %s. {{K|(%d vs %d)}}", it.Name, roll, dc)
	}
	it.Extra["wkmax"] = maxi(16, it.Extra["wkmax"]-4)
	it.Extra["wkdur"] = mini(it.Extra["wkdur"], it.Extra["wkmax"])
	g.wkPractice(line, 1)
	return fmt.Sprintf("{{y|You botch the work.}} The %s is a little worse for it. {{K|(%d vs %d)}}", it.Name, roll, dc)
}

func (g *Game) wkPractice(line string, xp int) {
	st := g.wk()
	st.XP[line] += xp
	r := st.Rank[line]
	if r < 5 && st.XP[line] >= wkXPAt[r] && st.XP[line]-xp < wkXPAt[r] {
		g.Msg(fmt.Sprintf("{{G|You have practised enough in %s to learn more.}} {{K|(find a master who teaches it)}}", strings.ToLower(wkLineByID(line).Name)))
	}
}

// ---------------------------------------------------------------- hooks

func wkTurn(g *Game) {
	if g.P == nil || g.Cur == nil || g.W == nil || g.Dead {
		return
	}
	l := g.wkl()
	for _, it := range g.wkEquipped() {
		g.wkEnsureDur(it)
	}
	if l.HP > 0 && g.P.HP < l.HP {
		var arm []*Item
		for _, it := range g.wkEquipped() {
			if it.Kind == "armor" && !wkBroken(it) {
				arm = append(arm, it)
			}
		}
		if len(arm) > 0 {
			h := HashStr(fmt.Sprintf("%s:wkwear:%d", g.Seed, g.Turn))
			lost := (l.HP - g.P.HP) * 100 / maxi(1, g.P.MaxHP)
			if lost >= 5 || h%3 == 0 {
				g.wkWear(arm[int(h%uint64(len(arm)))], 1+lost/20)
			}
		}
	}
	l.HP = g.P.HP
	if g.Turn%3 == 0 {
		for _, a := range g.Cur.Actors {
			if a.HP > 0 && a != g.P && g.Hostile(a) && dist(a, g.P) <= 1 {
				for _, it := range g.wkEquipped() {
					if (it.Kind == "weapon" || it.Kind == "ranged") && it.Slot == "hand" && !wkBroken(it) {
						g.wkWear(it, 1)
						break
					}
				}
				break
			}
		}
	}
	if g.Turn%5 == 0 {
		for _, it := range append([]*Item{}, g.P.Inv...) {
			if wkTracks(it) && it.Extra != nil && it.Extra["sig"] > 0 {
				if _, done := wkQ(it); !done {
					line := wkRepairLine(it)
					q, _ := g.wkRoll(line, it.Tier)
					g.wkApplyQuality(it, q, g.wkRank(line))
					g.wkPractice(line, 5+it.Tier*2)
					g.Msg(fmt.Sprintf("{{c|Your workbench piece turns out %s:}} %s.", wkQWord[q+1], it.Name))
					g.wkCount(q)
				}
			}
		}
	}
}

func wkKill(g *Game, d, src *Actor) {
	if g.P == nil || src != g.P {
		return
	}
	for _, it := range g.wkEquipped() {
		if (it.Kind == "weapon" || it.Kind == "ranged") && it.Slot == "hand" && !wkBroken(it) {
			g.wkWear(it, 1)
			return
		}
	}
}

func (g *Game) wkCount(q int) {
	st := g.wk()
	st.Made++
	if q >= 1 {
		st.Fine++
	}
	if q >= 2 {
		st.Master++
		g.wkFirst("masterwork", "made a piece of masterwork")
	}
}

// ---------------------------------------------------------------- work orders

type wkOrder struct {
	Line, Name string
	Build      func(g *Game, r *Rng, tier int) *Item
}

var wkOrders []wkOrder

func init() {
	arm := func(slot, noun string, fab bool) func(g *Game, r *Rng, tier int) *Item {
		return func(g *Game, r *Rng, tier int) *Item {
			it := g.MakeArmor(slot, tier, r)
			if fab {
				it.Name = wkFabric[clampi(tier, 0, 5)] + " " + noun
			} else {
				it.Name = wkHide[clampi(tier, 0, 5)] + " " + noun
			}
			return it
		}
	}
	wp := func(skill string) func(g *Game, r *Rng, tier int) *Item {
		return func(g *Game, r *Rng, tier int) *Item { return g.MakeWeapon(skill, tier, r) }
	}
	tonic := func(kind string) func(g *Game, r *Rng, tier int) *Item {
		return func(g *Game, r *Rng, tier int) *Item { return g.MakeTonic(kind, 2) }
	}
	wkOrders = []wkOrder{
		{"smith", "Forge a long blade", wp("longblade")},
		{"smith", "Forge a short blade", wp("shortblade")},
		{"smith", "Forge an axe", wp("axe")},
		{"smith", "Forge a mace", wp("cudgel")},
		{"smith", "Forge a breastplate", func(g *Game, r *Rng, tier int) *Item { return g.MakeArmor("body", tier, r) }},
		{"smith", "Forge a helm", func(g *Game, r *Rng, tier int) *Item { return g.MakeArmor("head", tier, r) }},
		{"carpenter", "Build a bow", func(g *Game, r *Rng, tier int) *Item { return g.MakeRanged(tier, r) }},
		{"carpenter", "Make a shield", func(g *Game, r *Rng, tier int) *Item { return g.MakeShield("round", tier, r) }},
		{"carpenter", "Cut a quiver of ammunition", func(g *Game, r *Rng, tier int) *Item { return g.MakeAmmo(8 + 2*tier) }},
		{"tailor", "Sew a coat", arm("body", "coat", true)},
		{"tailor", "Sew a hood", arm("head", "hood", true)},
		{"tailor", "Sew a cloak", arm("back", "cloak", true)},
		{"tailor", "Sew gloves", arm("hands", "gloves", true)},
		{"tanner", "Make boots", arm("feet", "boots", false)},
		{"tanner", "Make a jerkin", arm("body", "jerkin", false)},
		{"tanner", "Make a hide cloak", arm("back", "cloak", false)},
		{"alchemist", "Brew a healing draught", tonic("heal")},
		{"alchemist", "Brew a salve", tonic("salve")},
		{"alchemist", "Brew a quickening draught", tonic("haste")},
		{"alchemist", "Brew an antidote", tonic("antidote")},
		{"cook", "Cook a good meal", func(g *Game, r *Rng, tier int) *Item {
			it := g.MakeFood(r, 2)
			it.Power += 80 * tier
			it.Value += tier * 2
			return it
		}},
	}
}

func wkBitLine(line string) bool {
	switch line {
	case "smith", "carpenter", "tailor", "tanner":
		return true
	}
	return false
}

// wkCost: bits for the hard crafts, ingredients for cook and alchemist.
func wkCost(line string, tier int) (map[int]int, int) {
	if wkBitLine(line) {
		return map[int]int{clampi(tier+1, 1, 8): 2 + tier}, 0
	}
	return nil, 2 + tier/2
}

func (g *Game) wkIngredients() []*Item {
	var out []*Item
	for _, it := range g.P.Inv {
		if it.Kind == "ingredient" {
			out = append(out, it)
		}
	}
	return out
}

func (g *Game) wkCostText(line string, tier int) string {
	b, n := wkCost(line, tier)
	if b != nil {
		return costText(b, 0) + " bits"
	}
	return fmt.Sprintf("%d ingredients", n)
}

func (g *Game) wkCanPay(line string, tier int) bool {
	b, n := wkCost(line, tier)
	if b != nil {
		return g.craftSt().canPay(b)
	}
	have := 0
	for _, it := range g.wkIngredients() {
		have += wkUnits(it)
	}
	return have >= n
}

func (g *Game) wkPay(line string, tier int) {
	b, n := wkCost(line, tier)
	if b != nil {
		g.craftSt().pay(b)
		return
	}
	for _, it := range g.wkIngredients() {
		if n <= 0 {
			break
		}
		u := mini(n, wkUnits(it))
		g.wkTake(it, u)
		n -= u
	}
}

func (g *Game) wkOrderTier(line string) int { return clampi(g.wkRank(line)+1, 1, 5) }

// wkMake runs one work order or regional recipe.
func (g *Game) wkMake(line, name string, tier int, build func(g *Game, r *Rng, tier int) *Item, reg int) string {
	if !g.wkCanPay(line, tier) {
		return fmt.Sprintf("You lack the materials: %s.", g.wkCostText(line, tier))
	}
	if g.wkHostileSeen() {
		return "Not with enemies in sight."
	}
	g.wkPay(line, tier)
	g.spendTurns(10 + tier*5)
	st := g.wk()
	q, roll := g.wkRoll(line, tier)
	r := NewRng(fmt.Sprintf("%s:wk:make:%d", g.Seed, st.Made))
	it := build(g, r, tier)
	if it == nil {
		return "The work comes to nothing."
	}
	if it.Extra == nil {
		it.Extra = map[string]int{}
	}
	it.Extra["wkm"] = 1
	if reg > 0 {
		it.Extra["wkreg"] = reg
	}
	g.wkApplyQuality(it, q, g.wkRank(line))
	it.Lore = strings.TrimSpace(it.Lore + fmt.Sprintf(" %s work by %s (%s).", wkCap(wkQWord[q+1]), g.P.Name, wkRankTitle[g.wkRank(line)]))
	g.give(it)
	g.wkCount(q)
	g.wkPractice(line, 6+tier*3)
	g.gainXP(2 + tier*2)
	g.wkFirst("make:"+line, "finished a first piece of "+strings.ToLower(wkLineByID(line).Name))
	return fmt.Sprintf("You make %s: %s. {{K|(check %d, %s)}}", name, it.Name, roll, wkQWord[q+1])
}

// ---------------------------------------------------------------- regional recipes

type wkReg struct {
	Key, Name, Line, CName string
	Culture                int
	Build                  func(g *Game, r *Rng, tier int) *Item
}

func wkCraftLine(name string) string {
	n := strings.ToLower(name)
	has := func(ws ...string) bool {
		for _, w := range ws {
			if strings.Contains(n, w) {
				return true
			}
		}
		return false
	}
	switch {
	case has("smith", "metal", "forge", "iron", "blade", "steel", "bronze", "weapon"):
		return "smith"
	case has("weav", "cloth", "tailor", "dye", "textile", "embroid", "silk", "wool"):
		return "tailor"
	case has("leather", "hide", "tan", "fur"):
		return "tanner"
	case has("potion", "herb", "brew", "apothec", "medic", "incense", "perfum"):
		return "alchemist"
	case has("cook", "bak", "food", "spice", "ferment"):
		return "cook"
	case has("wood", "carp", "boat", "lacquer", "bow", "basket"):
		return "carpenter"
	}
	return ""
}

func (g *Game) wkRegional(cid int, line string) *wkReg {
	if g.W == nil {
		return nil
	}
	cu := g.W.cultByID[cid]
	nouns := wkRegionalNouns[line]
	if cu == nil || len(nouns) == 0 {
		return nil
	}
	h := HashStr(fmt.Sprintf("%s:wkreg:%d:%s", g.Seed, cid, line))
	ni := int(h % uint64(len(nouns)))
	noun := nouns[ni]
	mark := ""
	if pk := cgPack(g.W, cid); pk != nil && pk.Material != nil {
		for _, cr := range pk.Material.Crafts {
			if wkCraftLine(cr.Name) == line {
				mark = "Its trade: " + cr.Name + "."
				if cr.Mark != "" {
					mark += " Its mark: " + cr.Mark + "."
				}
				break
			}
		}
	}
	name := cu.Name + " " + noun
	if line == "smith" || line == "carpenter" {
		name = cu.Name + "-pattern " + noun
	}
	attr := []int{STR, AGI, WIL}[int(h/5%3)]
	lore := "A signature piece of the " + cu.Name + " " + strings.ToLower(wkLineByID(line).Name) + " tradition. " + mark
	build := func(g *Game, r *Rng, tier int) *Item {
		var it *Item
		switch line {
		case "smith":
			sk := map[string]string{"blade": "longblade", "axe": "axe", "knife": "shortblade", "mace": "cudgel"}[noun]
			it = g.MakeWeapon(sk, tier, r)
			addBonus(it, attr, 1)
		case "tailor":
			it = g.MakeArmor("body", tier, r)
			it.Region = cu.Sphere
			addBonus(it, EGO, 1)
		case "tanner":
			it = g.MakeArmor("feet", tier, r)
			it.AV++
			it.Region = cu.Sphere
		case "alchemist":
			it = g.MakeTonic("heal", 2)
			it.Value += 10
		case "cook":
			if ds := g.cgDishes(cid); len(ds) > 0 {
				it = g.cgDishItem(cid, ds[int(h%uint64(len(ds)))], 2)
				name = it.Name
			} else if nm, key := localDish(cu); nm != "" {
				it = &Item{ID: g.newID(), Kind: "dish", Name: nm, Effect: key, Glyph: '%', F: CYellow, Count: 2, Value: 20, Power: 500, Weight: 1}
				name = nm
			} else {
				it = g.MakeFood(r, 2)
				it.Power += 100
			}
		default:
			it = g.MakeRanged(tier, r)
			it.Range += 2
		}
		if line != "cook" || it.Kind != "dish" {
			it.Name = name
		}
		it.Lore = lore
		it.Value += 6 * tier
		return it
	}
	return &wkReg{Key: fmt.Sprintf("%d:%s", cid, line), Name: name, Line: line, CName: cu.Name, Culture: cid, Build: build}
}

func (g *Game) wkRegByKey(key string) *wkReg {
	var cid int
	var line string
	i := strings.Index(key, ":")
	if i < 0 {
		return nil
	}
	if _, err := fmt.Sscanf(key[:i], "%d", &cid); err != nil {
		return nil
	}
	line = key[i+1:]
	return g.wkRegional(cid, line)
}

// ---------------------------------------------------------------- masters

type wkMaster struct {
	Key, Name, Line string
	Burg, Rank      int
	Where           int // burg they are in now (0 = on the road)
}

func (g *Game) wkMasterOf(burg int, line string) *wkMaster {
	if g.W == nil || g.W.burgByID[burg] == nil {
		return nil
	}
	lvEnsure(g, burg)
	var keys []string
	for k, p := range lvSt(g).People {
		if p.Alive && p.Home == burg && wkTradeLine[p.Role] == line && p.Role != "child" {
			keys = append(keys, k)
		}
	}
	if len(keys) == 0 {
		return nil
	}
	sort.Strings(keys)
	p := lvSt(g).People[keys[0]]
	return &wkMaster{Key: p.Key, Name: p.Name, Line: line, Burg: burg, Where: p.Burg,
		Rank: 2 + int(HashStr(g.Seed+":wkm:"+p.Key)%4)}
}

func (g *Game) wkNearMasters() []wkMaster {
	if g.W == nil || g.Cur == nil {
		return nil
	}
	type bd struct {
		id, d int
	}
	var bs []bd
	for i := range g.W.Burgs {
		b := &g.W.Burgs[i]
		if !b.Alive || b.P < 0 || b.P >= g.W.W*g.W.H {
			continue
		}
		dx, dy := b.P%g.W.W-g.Cur.WX, b.P/g.W.W-g.Cur.WY
		d := dx*dx + dy*dy
		if d <= 14*14 {
			bs = append(bs, bd{b.ID, d})
		}
	}
	sort.Slice(bs, func(i, j int) bool {
		if bs[i].d != bs[j].d {
			return bs[i].d < bs[j].d
		}
		return bs[i].id < bs[j].id
	})
	var out []wkMaster
	for i := 0; i < len(bs) && i < 7; i++ {
		for _, l := range wkLines {
			if m := g.wkMasterOf(bs[i].id, l.ID); m != nil {
				out = append(out, *m)
			}
		}
	}
	return out
}

func wkIsMat(line string, it *Item) bool {
	n := strings.ToLower(it.Name)
	has := func(ws ...string) bool {
		for _, w := range ws {
			if strings.Contains(n, w) {
				return true
			}
		}
		return false
	}
	switch line {
	case "smith":
		return it.Kind == "scrap" || (it.Kind == "trade" || it.Kind == "trophy") && has("ore", "ingot", "iron", "steel", "bronze")
	case "tailor":
		return (it.Kind == "trade" || it.Kind == "trophy" || it.Kind == "scrap") && has("cloth", "silk", "wool", "thread", "fabric", "linen", "pelt", "fur")
	case "alchemist", "cook":
		return it.Kind == "ingredient"
	case "tanner":
		return it.Kind == "trophy" && strings.HasPrefix(it.Effect, "bsp:") && has(" hide", " pelt", "scales", "leather")
	case "carpenter":
		return it.Kind == "scrap" && (has("wood", "log", "plank", "timber", "oak", "pine") || len(n) > 0)
	}
	return false
}

func (g *Game) wkMats(line string) ([]*Item, int) {
	var out []*Item
	n := 0
	for _, it := range g.P.Inv {
		if wkIsMat(line, it) {
			out = append(out, it)
			n += wkUnits(it)
		}
	}
	return out, n
}

func (g *Game) wkLearn(line, from string) {
	st := g.wk()
	st.Rank[line] = clampi(st.Rank[line]+1, 0, 5)
	st.Lessons++
	r := st.Rank[line]
	l := wkLineByID(line)
	g.Msg(fmt.Sprintf("{{G|You learn %s}} ({{Y|%s}}, %s rank %d). %s", l.Techs[r-1].Name, wkRankTitle[r], strings.ToLower(l.Name), r, l.Techs[r-1].Text))
	g.gainXP(15 * r)
	g.wkFirst("learn:"+line, "began to learn "+strings.ToLower(l.Name)+" from "+from)
	if r == 5 {
		g.wkFirst("master:"+line, "became a master of "+strings.ToLower(l.Name))
	}
}

func (g *Game) wkFee(rank int) int {
	f := 60 * (rank + 1) * (rank + 1)
	return maxi(10, f*(100-clampi(g.Skills["persuasion"], 0, 3)*5)/100)
}

func (g *Game) wkMenu(text string, opts []Option) {
	g.Popup = Wrap(text, 100)
	g.Opts = append(opts, Option{"Never mind.", func() { g.talkMenu("\"Anything else?\"") }})
	g.Sel = 0
}

func (g *Game) wkTeach(a *Actor, m *wkMaster) {
	st, l := g.wk(), g.wkl()
	ln := wkLineByID(m.Line)
	rank := st.Rank[m.Line]
	h := HashStr(fmt.Sprintf("%s:wkt:%s:%d", g.Seed, m.Key, l.Taught[m.Key]))
	head := fmt.Sprintf("%s is a master of %s, rank %d. You are %s (rank %d).", m.Name, strings.ToLower(ln.Name), m.Rank, strings.ToLower(wkRankTitle[rank]), rank)
	var opts []Option
	switch {
	case rank >= 5:
		head += "\n\n\"There is nothing left I can teach you.\""
	case rank >= m.Rank:
		head += "\n\n\"I have shown you all I know. Find a greater master than me.\""
	case st.XP[m.Line] < wkXPAt[rank]:
		head += fmt.Sprintf("\n\n\"Not yet. Practise first, then come back.\" {{K|(practice %d of %d)}}", st.XP[m.Line], wkXPAt[rank])
	default:
		next := ln.Techs[rank]
		head += fmt.Sprintf("\n\nNext: {{W|%s}}. %s", next.Name, next.Text)
		fee := g.wkFee(rank)
		opts = append(opts, Option{fmt.Sprintf("Pay the fee {{K|(%s)}}", g.cuFormat(fee)), func() {
			if !g.cuSpend(fee) {
				g.wkMenu("\"Come back with the coin.\"", nil)
				return
			}
			l.Taught[m.Key]++
			g.wkLearn(m.Line, m.Name)
			g.spendTurns(60)
			g.talkMenu(wkTeachLines[int(h%uint64(len(wkTeachLines)))])
		}})
		er := l.Errands[m.Key]
		if er != nil && (er.Line != m.Line || g.Turn > er.Due) {
			delete(l.Errands, m.Key)
			er = nil
		}
		need := 3 + rank
		if er != nil {
			need = er.Need
		}
		mats, have := g.wkMats(m.Line)
		switch {
		case have >= need:
			opts = append(opts, Option{fmt.Sprintf("Hand over the %s {{K|(%d)}}", ln.Mat, need), func() {
				left := need
				for _, it := range mats {
					if left <= 0 {
						break
					}
					u := mini(left, wkUnits(it))
					g.wkTake(it, u)
					left -= u
				}
				delete(l.Errands, m.Key)
				l.Taught[m.Key]++
				g.wkLearn(m.Line, m.Name)
				g.spendTurns(60)
				g.talkMenu(wkTeachLines[int(h%uint64(len(wkTeachLines)))])
			}})
		case er == nil:
			opts = append(opts, Option{fmt.Sprintf("Take the errand: bring %d of %s", need, ln.Mat), func() {
				l.Errands[m.Key] = &wkErrand{Line: m.Line, Need: need, Due: g.Turn + 12*DayLen}
				g.wkMenu(fmt.Sprintf(wkErrandLines[int(h%uint64(len(wkErrandLines)))], need, ln.Mat)+" {{K|(twelve days)}}", nil)
			}})
		default:
			head += fmt.Sprintf("\n\nErrand open: %d more of %s.", need-have, ln.Mat)
		}
	}
	if b := g.W.burgByID[m.Burg]; b != nil {
		if rg := g.wkRegional(b.Culture, m.Line); rg != nil {
			key := rg.Key
			if st.Recipes[key] {
				head += fmt.Sprintf("\n\nYou already know the %s.", rg.Name)
			} else {
				fee := 30 + 25*rank
				opts = append(opts, Option{fmt.Sprintf("Learn the %s {{K|(%s)}}", rg.Name, g.cuFormat(fee)), func() {
					if !g.cuSpend(fee) {
						g.wkMenu("\"Come back with the coin.\"", nil)
						return
					}
					st.Recipes[key] = true
					g.wkFirst("regional", "learned a regional recipe")
					g.Msg(fmt.Sprintf("{{G|You learn the %s.}} {{K|(Workshop, Recipes)}}", rg.Name))
					g.spendTurns(30)
					g.talkMenu("\"Remember whose it is.\"")
				}})
			}
		}
	}
	g.wkMenu(head, opts)
}

func (g *Game) wkPaidRepair(a *Actor, line string) {
	items := g.wkWorn(line)
	if len(items) == 0 {
		g.talkMenu("\"Nothing here needs me.\"")
		return
	}
	var opts []Option
	total := 0
	for _, it := range items {
		it := it
		c := g.wkRepairCost(it, line)
		total += c
		opts = append(opts, Option{fmt.Sprintf("%s {{K|(%d/%d, %s)}}", it.Name, it.Extra["wkdur"], it.Extra["wkmax"], g.cuFormat(c)), func() {
			if !g.cuSpend(c) {
				g.wkMenu("\"Not on credit.\"", nil)
				return
			}
			g.wkMend(it, it.Extra["wkmax"])
			g.wk().Repairs++
			g.spendTurns(15)
			g.wkPaidRepair(a, line)
		}})
		if len(opts) >= 8 {
			break
		}
	}
	if len(items) > 1 {
		opts = append(opts, Option{fmt.Sprintf("{{W|Mend everything.}} {{K|(%s)}}", g.cuFormat(total)), func() {
			if !g.cuSpend(total) {
				g.wkMenu("\"Not on credit.\"", nil)
				return
			}
			for _, it := range items {
				g.wkMend(it, it.Extra["wkmax"])
				g.wk().Repairs++
			}
			g.spendTurns(15 * len(items))
			g.talkMenu("\"Good as it gets. Mind the edge.\"")
		}})
	}
	g.wkMenu("\"Show me what is worn.\"", opts)
}

func wkTalk(g *Game, a *Actor) []Option {
	if g.W == nil || g.Cur == nil || g.P == nil || a == nil || a.HP <= 0 || a.Angry || a.AI == "hostile" || a.Tpl != "npc" || g.Cur.Kind != "town" {
		return nil
	}
	var out []Option
	if line := wkRoleLine[a.Role]; line != "" {
		if n := len(g.wkWorn(line)); n > 0 {
			out = append(out, Option{fmt.Sprintf("{{O|Have gear mended.}} {{K|(%d worn)}}", n), func() { g.wkPaidRepair(a, line) }})
		}
	}
	if g.Cur.Burg > 0 && a.AI != "follower" {
		if p := lvPersonFor(g, a); p != nil && p.Alive {
			if line := wkTradeLine[p.Role]; line != "" {
				if m := g.wkMasterOf(p.Home, line); m != nil && m.Key == p.Key {
					out = append(out, Option{fmt.Sprintf("{{O|Ask about learning %s.}}", strings.ToLower(wkLineByID(line).Name)), func() { g.wkTeach(a, m) }})
				}
			}
		}
	}
	return out
}

// ---------------------------------------------------------------- screen

type wkRow struct {
	Text, Detail string
	Do           func()
}

var wkTabs = []string{"Craft lines", "Masters", "Workshop", "Recipes"}

func (g *Game) wkOpen(tab int) {
	st := g.wk()
	st.tab, st.sel, st.msg = tab, 0, ""
	st.masters = g.wkNearMasters()
	for _, it := range g.wkAllGear() {
		g.wkEnsureDur(it)
	}
	g.Mode = "wk"
}

func (g *Game) wkRows() []wkRow {
	st := g.wk()
	var rows []wkRow
	switch st.tab {
	case 0:
		for _, l := range wkLines {
			l := l
			r := st.Rank[l.ID]
			xp := fmt.Sprintf("practice %d", st.XP[l.ID])
			if r < 5 {
				xp = fmt.Sprintf("practice %d/%d", st.XP[l.ID], wkXPAt[r])
			}
			var d []string
			for i, t := range l.Techs {
				mark := "{{K|-}}"
				if i < r {
					mark = "{{G|+}}"
				}
				d = append(d, fmt.Sprintf("%s %s: %s", mark, t.Name, t.Text))
			}
			tier, lab, _ := g.wkTools(l.ID)
			d = append(d, fmt.Sprintf("Check here: %+d (%s, tools tier %d).", g.wkMod(l.ID), nonEmpty(lab, "no workshop"), tier))
			rows = append(rows, wkRow{Text: fmt.Sprintf("%-10s %-11s %s", l.Name, wkRankTitle[r], xp), Detail: strings.Join(d, "\n")})
		}
	case 1:
		for _, m := range st.masters {
			m := m
			ln := wkLineByID(m.Line)
			b := g.W.burgByID[m.Burg]
			bx, by, _ := g.jbBurgXY(m.Burg)
			dir := jbDirName(bx-g.Cur.WX, by-g.Cur.WY)
			here := "at home"
			if m.Where != m.Burg {
				here = "away from home"
			}
			rk := st.Rank[m.Line]
			det := fmt.Sprintf("%s of %s, %s (%s). Teaches %s up to rank %d.", m.Name, g.jbBurgName(m.Burg), dir, here, strings.ToLower(ln.Name), m.Rank)
			switch {
			case rk >= m.Rank || rk >= 5:
				det += " Nothing more for you here."
			case st.XP[m.Line] < wkXPAt[rk]:
				det += fmt.Sprintf(" You need more practice first (%d of %d).", st.XP[m.Line], wkXPAt[rk])
			default:
				det += fmt.Sprintf(" Next: %s, for %s or %d of %s.", ln.Techs[rk].Name, g.cuFormat(g.wkFee(rk)), 3+rk, ln.Mat)
			}
			if b != nil {
				if rg := g.wkRegional(b.Culture, m.Line); rg != nil {
					det += " Local piece: " + rg.Name + "."
				}
			}
			rows = append(rows, wkRow{Text: fmt.Sprintf("%-18s %-10s rank %d  %s", m.Name, ln.Name, m.Rank, g.jbBurgName(m.Burg)), Detail: det + "\nEnter marks their town for the map arrow.",
				Do: func() {
					g.ceSetQuestTarget(bx, by)
					st.msg = "Marked " + g.jbBurgName(m.Burg) + "."
				}})
		}
		if len(rows) == 0 {
			rows = append(rows, wkRow{Text: "No craft masters are known within a few parasangs.", Detail: "Towns have smiths, tailors, healers, tanners, bakers and carpenters. Walk into one and talk to the tradespeople."})
		}
	case 2:
		stations := g.wkStations()
		for i := range stations {
			s := stations[i]
			desc := fmt.Sprintf("Serves: %s. Tools: %s.", strings.Join(s.Lines, ", "), wkToolWord[clampi(s.Tier, 0, 3)])
			do := func() {}
			switch s.Src {
			case "town":
				hired := g.wkl().Hired[s.Key] > g.Turn
				if hired {
					desc += fmt.Sprintf(" Hired for %s more.", jbClock(g.wkl().Hired[s.Key]-g.Turn))
				} else {
					desc += fmt.Sprintf(" For hire, %s an hour. Enter hires one hour.", g.cuFormat(s.Hire))
				}
				do = func() { st.msg = g.wkHire(&s, 1) }
			case "home":
				line := s.Lines[0]
				cur := maxi(1, g.wkl().HomeTools[g.Cur.Key+":"+line])
				if cur < 3 {
					desc += fmt.Sprintf(" Enter upgrades to %s for %s.", wkToolWord[cur+1], g.cuFormat(wkHomeUpgrade[cur+1]))
				}
				do = func() {
					c := maxi(1, g.wkl().HomeTools[g.Cur.Key+":"+line])
					if c >= 3 {
						st.msg = "The bench is as good as it gets."
						return
					}
					if !g.cuSpend(wkHomeUpgrade[c+1]) {
						st.msg = "You cannot afford the tools."
						return
					}
					g.wkl().HomeTools[g.Cur.Key+":"+line] = c + 1
					st.msg = "Your bench now has " + wkToolWord[c+1] + "."
				}
			default:
				desc += " Free to use, but crude."
			}
			rows = append(rows, wkRow{Text: fmt.Sprintf("%s ({{Y|tier %d}})", s.Name, s.Tier), Detail: desc, Do: do})
		}
		if len(stations) == 0 {
			rows = append(rows, wkRow{Text: "No workshop here.", Detail: "Claim a home (H) for a bench of your own, stand near a town forge, kitchen or shop to hire it, or find an old forge or kitchen in a dungeon."})
		}
		sound := 0
		for _, it := range g.wkAllGear() {
			it := it
			if wkMissing(it) > 0 || wkBroken(it) {
				ln := wkRepairLine(it)
				rows = append(rows, wkRow{Text: fmt.Sprintf("Repair: %s  %d/%d%s", it.Name, it.Extra["wkdur"], it.Extra["wkmax"], map[bool]string{true: " BROKEN", false: ""}[wkBroken(it)]),
					Detail: fmt.Sprintf("Mended by %s work. Enter tries it here (a skill check); each mend costs a little lasting strength. A smith, tailor or tanner in town will do it for about %s.", strings.ToLower(wkLineByID(ln).Name), g.cuFormat(g.wkRepairCost(it, ln))),
					Do:     func() { st.msg = g.wkSelfRepair(it) }})
			} else {
				sound++
			}
		}
		rows = append(rows, wkRow{Text: fmt.Sprintf("%d piece(s) of gear in good shape.", sound), Detail: "Gear wears with fighting and with blows taken. At zero it breaks and gives no bonus until mended."})
	case 3:
		for _, o := range wkOrders {
			o := o
			t := g.wkOrderTier(o.Line)
			tier, lab, _ := g.wkTools(o.Line)
			rows = append(rows, wkRow{Text: fmt.Sprintf("%-28s %-9s tier %d  needs %s", o.Name, wkLineByID(o.Line).Name, t, g.wkCostText(o.Line, t)),
				Detail: fmt.Sprintf("Your check here: %+d with %s. Quality depends on the roll, your rank and tools (tier %d).", g.wkMod(o.Line), nonEmpty(lab, "no workshop"), tier),
				Do:     func() { st.msg = g.wkMake(o.Line, strings.ToLower(o.Name), t, o.Build, 0) }})
		}
		var keys []string
		for k, v := range st.Recipes {
			if v && !strings.HasPrefix(k, "first:") {
				keys = append(keys, k)
			}
		}
		sort.Strings(keys)
		for _, k := range keys {
			rg := g.wkRegByKey(k)
			if rg == nil {
				continue
			}
			t := clampi(g.wkRank(rg.Line)+1, 1, 4)
			rows = append(rows, wkRow{Text: fmt.Sprintf("{{Y|%s}} (%s)  needs %s", rg.Name, rg.Line, g.wkCostText(rg.Line, t)),
				Detail: "A regional recipe from " + rg.CName + " masters. It carries a small bonus and the culture's name.",
				Do:     func() { st.msg = g.wkMake(rg.Line, rg.Name, t, rg.Build, rg.Culture+1) }})
		}
	}
	return rows
}

func wkKeys(g *Game, k string) {
	st := g.wk()
	rows := g.wkRows()
	switch k {
	case "Escape", "q":
		g.Mode = "main"
		return
	case "Tab", "ArrowRight":
		st.tab, st.sel, st.msg = (st.tab+1)%len(wkTabs), 0, ""
	case "ArrowLeft":
		st.tab, st.sel, st.msg = (st.tab+len(wkTabs)-1)%len(wkTabs), 0, ""
	case "ArrowUp", "N8", "k":
		st.sel = maxi(0, st.sel-1)
	case "ArrowDown", "N2", "j":
		st.sel = mini(maxi(0, len(rows)-1), st.sel+1)
	case "Enter", "a":
		if st.sel < len(rows) && rows[st.sel].Do != nil {
			rows[st.sel].Do()
			if st.tab == 1 {
				break
			}
			if len(g.wkRows()) <= st.sel {
				st.sel = maxi(0, len(g.wkRows())-1)
			}
		}
	case "H", "h":
		if st.tab == 2 && st.sel < len(rows) && rows[st.sel].Do != nil {
			rows[st.sel].Do()
		}
	case "1", "2", "3", "4":
		st.tab, st.sel, st.msg = int(k[0]-'1'), 0, ""
	}
	if st.tab == 1 && len(st.masters) == 0 {
		st.masters = g.wkNearMasters()
	}
}

func (g *Game) wkDraw() {
	sc := g.sc
	st := g.wk()
	x, y, w, h := g.panel("Workshop")
	bottom := 2 + 1 + h - 1
	var tb strings.Builder
	for i, t := range wkTabs {
		c := "K"
		if i == st.tab {
			c = "W"
		}
		fmt.Fprintf(&tb, "{{%s|[ %d %s ]}}  ", c, i+1, t)
	}
	sc.TextB(x, y, tb.String(), CGray, CBgDark, x+w)
	y++
	sc.TextB(x, y, "{{K|Up/Down choose  Enter act  Tab or 1-4 switch  Esc close}}", CGray, CBgDark, x+w)
	y += 2
	rows := g.wkRows()
	if st.sel >= len(rows) {
		st.sel = maxi(0, len(rows)-1)
	}
	for i, r := range rows {
		if y >= bottom-10 {
			break
		}
		mark := "  "
		if i == st.sel {
			mark = "> "
		}
		sc.TextB(x, y, mark+r.Text, CGray, CBgDark, x+w)
		y++
	}
	y++
	if st.sel < len(rows) {
		for _, d := range strings.Split(rows[st.sel].Detail, "\n") {
			for _, ln := range Wrap(d, w) {
				if y < bottom-2 {
					sc.TextB(x, y, ln, CGray, CBgDark, x+w)
					y++
				}
			}
		}
	}
	if st.msg != "" {
		for i, ln := range Wrap("{{Y|"+st.msg+"}}", w) {
			if i < 2 {
				sc.TextB(x, bottom-2+i, ln, CGray, CBgDark, x+w)
			}
		}
	}
}

func wkCap(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}
