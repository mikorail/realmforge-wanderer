package engine

import (
	"fmt"
	"sort"
	"strings"
)

// Skill trees (WP4). Economy: 1 SP = skQudPerSP Qud skill points (rounded, min 1 for any paid item), so a
// tree unlock costs 1-3 SP and nodes 0-7 SP. Level-ups grant 2 SP (+1 per 2 INT above 10), see leveling.go.
// Rank (g.Skills[tree], read all over combat/talk code) = nodes owned, capped at skRankCap, kept in sync
// by skSync; the old rank-up screen still works because skSync grants free nodes to match a raised rank.
const (
	skQudPerSP = 60
	skRankCap  = 3
)

type NodeDef struct {
	ID, Name string
	Q        int    // Qud point cost
	Req      string // node id required first
	Kind     string // passive | active | stance
	Cool     int    // player actions between uses
	Desc     string
	DV       int // passive: permanent DV (added to DVNat on purchase)
	HP       int // passive: permanent max HP
	Speed    int // passive: permanent speed
}

type TreeDef struct {
	ID, Name string
	Unlock   int // Qud point cost
	Attr     int // gating attribute (-1 none)
	AttrMin  int
	Nodes    []NodeDef
}

func skCost(q int) int {
	if q <= 0 {
		return 0
	}
	return maxi(1, (q+skQudPerSP/2)/skQudPerSP)
}

func (t *TreeDef) UnlockSP() int { return skCost(t.Unlock) }

var skTrees = []*TreeDef{
	{ID: "longblade", Name: "Long Blade", Unlock: 100, Attr: -1, Nodes: []NodeDef{
		{ID: "lb_prof", Name: "Long Blade Proficiency", Kind: "passive", Desc: "Better aim with long blades."},
		{ID: "lb_poise", Name: "Duelist's Poise", Q: 100, Kind: "passive", DV: 1, Desc: "+1 DV."},
		{ID: "lunge", Name: "Lunge", Q: 200, Req: "lb_prof", Kind: "active", Cool: 8, Desc: "Dash up to 3 tiles and run a foe through."},
		{ID: "lb_enguard", Name: "En Garde!", Q: 300, Req: "lunge", Kind: "passive", DV: 1, Desc: "+1 DV."},
	}},
	{ID: "shortblade", Name: "Short Blade", Unlock: 50, Attr: -1, Nodes: []NodeDef{
		{ID: "sb_prof", Name: "Short Blade Expertise", Kind: "passive", Desc: "Better aim with short blades."},
		{ID: "jab", Name: "Jab", Q: 100, Kind: "active", Cool: 6, Desc: "Two quick stabs, the second at a cost to the target's footing."},
		{ID: "hobble", Name: "Hobble", Q: 150, Kind: "active", Cool: 10, Desc: "Cut a leg: the foe is hobbled and bleeds."},
		{ID: "sb_shank", Name: "Shank", Q: 250, Req: "hobble", Kind: "passive", DV: 1, Desc: "+1 DV."},
	}},
	{ID: "axe", Name: "Axe", Unlock: 50, Attr: -1, Nodes: []NodeDef{
		{ID: "ax_prof", Name: "Axe Proficiency", Kind: "passive", Desc: "Better aim with axes."},
		{ID: "cleave", Name: "Cleave", Q: 150, Kind: "active", Cool: 8, Desc: "Hit every adjacent foe and crack their armor."},
		{ID: "dismember", Name: "Dismember", Q: 150, Kind: "active", Cool: 12, Desc: "Heavy blow that may sever a limb."},
		{ID: "ax_berserk", Name: "Berserk!", Q: 300, Req: "cleave", Kind: "passive", HP: 6, Desc: "+6 max HP."},
	}},
	{ID: "cudgel", Name: "Cudgel", Unlock: 50, Attr: -1, Nodes: []NodeDef{
		{ID: "cd_prof", Name: "Cudgel Proficiency", Kind: "passive", Desc: "Better aim with cudgels."},
		{ID: "cd_back", Name: "Backswing", Q: 100, Kind: "passive", DV: 1, Desc: "+1 DV."},
		{ID: "bludgeon", Name: "Bludgeon", Q: 150, Kind: "active", Cool: 8, Desc: "Crushing blow that often stuns."},
		{ID: "cd_demolish", Name: "Demolish", Q: 300, Req: "bludgeon", Kind: "passive", HP: 5, Desc: "+5 max HP."},
	}},
	{ID: "missile", Name: "Bow and Rifle", Unlock: 100, Attr: -1, Nodes: []NodeDef{
		{ID: "ms_bead", Name: "Draw a Bead", Kind: "passive", Desc: "Better aim and range."},
		{ID: "ms_steady", Name: "Steady Hands", Kind: "passive", DV: 1, Desc: "+1 DV."},
		{ID: "sureshot", Name: "Sure Shot", Q: 150, Kind: "active", Cool: 10, Desc: "A carefully aimed shot with bonus damage."},
		{ID: "ms_ultra", Name: "Ultra Fire", Q: 300, Req: "sureshot", Kind: "passive", Speed: 5, Desc: "+5 speed."},
	}},
	{ID: "heavy", Name: "Heavy Weapon", Unlock: 100, Attr: STR, AttrMin: 14, Nodes: []NodeDef{
		{ID: "hv_shoulders", Name: "Strapping Shoulders", Kind: "passive", HP: 4, Desc: "+4 max HP."},
		{ID: "hv_tank", Name: "Tank", Q: 150, Kind: "passive", HP: 6, Desc: "+6 max HP."},
		{ID: "sweep", Name: "Sweep", Q: 200, Kind: "active", Cool: 10, Desc: "Sweep all adjacent foes; they may be knocked prone."},
	}},
	{ID: "shield", Name: "Shield", Unlock: 100, Attr: -1, Nodes: []NodeDef{
		{ID: "sh_block", Name: "Block", Kind: "passive", DV: 1, Desc: "+1 DV."},
		{ID: "shieldslam", Name: "Shield Slam", Q: 100, Kind: "active", Cool: 8, Desc: "Slam a foe: stun and shove back."},
		{ID: "sh_swift", Name: "Swift Block", Q: 200, Req: "shieldslam", Kind: "passive", DV: 1, Desc: "+1 DV."},
		{ID: "sh_wall", Name: "Shield Wall", Q: 200, Req: "sh_swift", Kind: "passive", DV: 1, Desc: "+1 DV."},
	}},
	{ID: "multiweapon", Name: "Multiweapon Fighting", Unlock: 150, Attr: AGI, AttrMin: 12, Nodes: []NodeDef{
		{ID: "mw_prof", Name: "Multiweapon Proficiency", Kind: "passive", Desc: "Better aim."},
		{ID: "flurry", Name: "Flurry", Q: 100, Kind: "active", Cool: 10, Desc: "Swing every weapon twice."},
		{ID: "mw_exp", Name: "Multiweapon Expertise", Q: 150, Kind: "passive", DV: 1, Desc: "+1 DV."},
		{ID: "mw_mastery", Name: "Multiweapon Mastery", Q: 200, Req: "mw_exp", Kind: "passive", Speed: 5, Desc: "+5 speed."},
	}},
	{ID: "tactics", Name: "Tactics", Unlock: 100, Attr: -1, Nodes: []NodeDef{
		{ID: "tc_hurdle", Name: "Hurdle", Kind: "passive", Desc: "You vault low obstacles in your stride."},
		{ID: "tc_kick", Name: "Kickback", Q: 50, Kind: "passive", DV: 1, Desc: "+1 DV."},
		{ID: "charge", Name: "Charge", Q: 150, Kind: "active", Cool: 10, Desc: "Rush up to 4 tiles and strike with a bonus."},
		{ID: "tc_juke", Name: "Juke", Q: 200, Req: "charge", Kind: "passive", DV: 1, Desc: "+1 DV."},
	}},
	{ID: "acrobatics", Name: "Acrobatics", Unlock: 75, Attr: AGI, AttrMin: 12, Nodes: []NodeDef{
		{ID: "ac_dodge", Name: "Dodge", Kind: "passive", Desc: "Light on your feet."},
		{ID: "jump", Name: "Jump", Q: 100, Kind: "active", Cool: 6, Desc: "Leap up to 4 tiles over anything."},
		{ID: "tumble", Name: "Tumble", Q: 150, Kind: "active", Cool: 20, Desc: "Roll and weave: +6 DV for a few turns."},
	}},
	{ID: "endurance", Name: "Endurance", Unlock: 100, Attr: -1, Nodes: []NodeDef{
		{ID: "en_hardy", Name: "Hardy", Kind: "passive", HP: 4, Desc: "+4 max HP."},
		{ID: "shakeitoff", Name: "Shake It Off", Q: 150, Kind: "active", Cool: 40, Desc: "Throw off stuns, dazes, sickness and fear."},
		{ID: "en_weathered", Name: "Weathered", Q: 200, Kind: "passive", HP: 5, Desc: "+5 max HP."},
		{ID: "en_stride", Name: "Longstrider", Q: 250, Kind: "passive", Speed: 10, Desc: "+10 speed."},
	}},
	{ID: "selfdiscipline", Name: "Self-discipline", Unlock: 150, Attr: WIL, AttrMin: 12, Nodes: []NodeDef{
		{ID: "meditate", Name: "Meditate", Kind: "active", Cool: 60, Desc: "Recover cooldowns and a little health."},
		{ID: "sd_fasting", Name: "Fasting Way", Q: 50, Kind: "passive", Desc: "Hunger bites less."},
		{ID: "sd_ironmind", Name: "Iron Mind", Q: 100, Kind: "passive", HP: 4, Desc: "+4 max HP."},
	}},
	{ID: "persuasion", Name: "Persuasion", Unlock: 100, Attr: -1, Nodes: []NodeDef{
		{ID: "stare", Name: "Menacing Stare", Kind: "active", Cool: 20, Desc: "Unnerve a foe: shaken."},
		{ID: "berate", Name: "Berate", Q: 150, Kind: "active", Cool: 15, Desc: "Shame a foe (Ego vs Willpower)."},
		{ID: "intimidate", Name: "Intimidate", Q: 200, Req: "stare", Kind: "active", Cool: 20, Desc: "Shake and terrify a foe so it flees."},
		{ID: "pe_inspire", Name: "Inspiring Presence", Q: 250, Kind: "passive", DV: 1, Desc: "+1 DV."},
		{ID: "proselytize", Name: "Proselytize", Q: 300, Req: "berate", Kind: "active", Cool: 60, Desc: "Win over a peaceful creature to follow you."},
	}},
	{ID: "physic", Name: "Physic", Unlock: 50, Attr: -1, Nodes: []NodeDef{
		{ID: "staunch", Name: "Staunch Wounds", Kind: "active", Cool: 15, Desc: "Stop bleeding and mend a little."},
		{ID: "ph_surgeon", Name: "Field Surgeon", Q: 50, Kind: "passive", HP: 3, Desc: "+3 max HP."},
		{ID: "ph_apoth", Name: "Apothecary", Q: 100, Kind: "passive", HP: 3, Desc: "+3 max HP."},
	}},
	{ID: "tinkering", Name: "Tinkering", Unlock: 100, Attr: INT, AttrMin: 12, Nodes: []NodeDef{
		{ID: "ti_gadget", Name: "Gadget Inspector", Kind: "passive", Desc: "Spot what a device does."},
		{ID: "ti_disassemble", Name: "Disassemble", Q: 100, Kind: "passive", Desc: "Break gear into scrap."},
		{ID: "repair", Name: "Repair", Q: 100, Kind: "active", Cool: 12, Desc: "Mend an adjacent construct or companion."},
		{ID: "ti_scavenger", Name: "Scavenger", Q: 100, Kind: "passive", Desc: "More scrap from salvage."},
	}},
	{ID: "customs", Name: "Customs and Folklore", Unlock: 150, Attr: -1, Nodes: []NodeDef{
		{ID: "cu_tactful", Name: "Tactful", Kind: "passive", Desc: "Know what not to say."},
		{ID: "trashdiv", Name: "Trash Divining", Q: 100, Kind: "active", Cool: 100, Desc: "Sift refuse nearby for something worth keeping."},
	}},
	{ID: "survival", Name: "Wayfaring", Unlock: 100, Attr: -1, Nodes: []NodeDef{
		{ID: "sv_compass", Name: "Mind's Compass", Kind: "passive", Desc: "You rarely get lost."},
		{ID: "sv_camp", Name: "Make Camp", Kind: "passive", Desc: "Rest safer in the wild."},
		{ID: "trailblazer", Name: "Trailblazer", Q: 100, Kind: "active", Cool: 60, Desc: "Move faster for a while."},
		{ID: "sv_lore", Name: "Wilderness Lore", Q: 50, Kind: "passive", Speed: 5, Desc: "+5 speed."},
	}},
	{ID: "cooking", Name: "Cooking and Gathering", Unlock: 100, Attr: -1, Nodes: []NodeDef{
		{ID: "ck_prep", Name: "Meal Preparation", Kind: "passive", Desc: "Know your way around a fire."},
		{ID: "butchery", Name: "Butchery", Q: 100, Kind: "active", Cool: 0, Desc: "Carve a corpse at your feet or beside you into ingredients."},
		{ID: "ck_spicer", Name: "Spicer", Q: 50, Kind: "passive", Desc: "Spices make meals last."},
	}},
	{ID: "lore", Name: "Lore", Unlock: 75, Attr: INT, AttrMin: 11, Nodes: []NodeDef{
		{ID: "lo_scholar", Name: "Scholar", Kind: "passive", Desc: "Read relics and ruins better."},
		{ID: "lo_recall", Name: "Recall", Q: 100, Kind: "passive", Desc: "You remember what you read."},
		{ID: "lo_decipher", Name: "Decipher", Q: 150, Kind: "passive", Desc: "Rank 3 reads the oldest scripts."},
	}},
}

var (
	skTreeByID = map[string]*TreeDef{}
	skNodeByID = map[string]*NodeDef{}
	skNodeTree = map[string]*TreeDef{}
)

type skState struct {
	Open map[string]bool `json:"open,omitempty"`
	Own  map[string]bool `json:"own,omitempty"`
}

func init() {
	for _, t := range skTrees {
		skTreeByID[t.ID] = t
		for i := range t.Nodes {
			skNodeByID[t.Nodes[i].ID] = &t.Nodes[i]
			skNodeTree[t.Nodes[i].ID] = t
		}
	}
	mainKeys["K"] = func(g *Game) { g.Mode, g.Sel, g.Tab = "skills", 0, 0 }
	modeKeys["skills"] = skKeys
	modeDraws["skills"] = func(g *Game) { g.skDraw() }
	turnHooks = append(turnHooks, func(g *Game) { g.skSync() })
	enterHooks = append(enterHooks, func(g *Game, zn *Zone) { g.skSync(); g.skDiscover(zn) })
}

func (g *Game) skState() *skState {
	s := g.ext("skills", func() any { return &skState{} }).(*skState)
	if s.Open == nil {
		s.Open = map[string]bool{}
	}
	if s.Own == nil {
		s.Own = map[string]bool{}
	}
	return s
}

func (g *Game) skHas(node string) bool { return g.skState().Own[node] }

func (g *Game) skOwned(t *TreeDef) int {
	s := g.skState()
	n := 0
	for _, nd := range t.Nodes {
		if s.Own[nd.ID] {
			n++
		}
	}
	return n
}

// grant marks a node owned. fx applies its permanent passive effect (real purchases only).
func (g *Game) skGrant(nd *NodeDef, fx bool) {
	s := g.skState()
	if s.Own[nd.ID] {
		return
	}
	s.Own[nd.ID] = true
	if fx {
		g.P.DVNat += nd.DV
		g.P.MaxHP += nd.HP
		g.P.HP += nd.HP
		g.P.Speed += nd.Speed
	}
}

// skSync keeps the tree state and the legacy Skills ranks consistent (migration of old saves, old rank-up screen).
func (g *Game) skSync() {
	s := g.skState()
	if g.Skills == nil {
		g.Skills = map[string]int{}
	}
	for _, t := range skTrees {
		r := g.Skills[t.ID]
		if r > 0 && !s.Open[t.ID] {
			s.Open[t.ID] = true
			for i := range t.Nodes {
				if skCost(t.Nodes[i].Q) == 0 {
					g.skGrant(&t.Nodes[i], false)
				}
			}
		}
		if !s.Open[t.ID] {
			continue
		}
		for i := range t.Nodes {
			if g.skOwned(t) >= r || g.skOwned(t) >= skRankCap {
				break
			}
			if !s.Own[t.Nodes[i].ID] && (t.Nodes[i].Req == "" || s.Own[t.Nodes[i].Req]) {
				g.skGrant(&t.Nodes[i], false)
			}
		}
		g.Skills[t.ID] = mini(g.skOwned(t), skRankCap)
	}
}

func (g *Game) skCanUnlock(t *TreeDef) (bool, string) {
	s := g.skState()
	if s.Open[t.ID] {
		return false, "already learned"
	}
	if t.Attr >= 0 && g.P.Score(t.Attr) < t.AttrMin {
		return false, fmt.Sprintf("needs %s %d", AttrNames[t.Attr], t.AttrMin)
	}
	if g.SP < t.UnlockSP() {
		return false, fmt.Sprintf("needs %d SP", t.UnlockSP())
	}
	return true, ""
}

func (g *Game) skUnlock(t *TreeDef) bool {
	if ok, why := g.skCanUnlock(t); !ok {
		g.Msg("{{K|" + t.Name + ": " + why + ".}}")
		return false
	}
	g.SP -= t.UnlockSP()
	g.skState().Open[t.ID] = true
	for i := range t.Nodes {
		if skCost(t.Nodes[i].Q) == 0 {
			g.skGrant(&t.Nodes[i], true)
		}
	}
	g.skSync()
	g.Msg(fmt.Sprintf("{{G|You learn %s.}}", t.Name))
	return true
}

func (g *Game) skCanBuy(t *TreeDef, nd *NodeDef) (bool, string) {
	s := g.skState()
	switch {
	case !s.Open[t.ID]:
		return false, "learn the skill first"
	case s.Own[nd.ID]:
		return false, "owned"
	case nd.Req != "" && !s.Own[nd.Req]:
		return false, "needs " + skNodeByID[nd.Req].Name
	case g.SP < skCost(nd.Q):
		return false, fmt.Sprintf("needs %d SP", skCost(nd.Q))
	}
	return true, ""
}

func (g *Game) skBuy(t *TreeDef, nd *NodeDef) bool {
	if ok, why := g.skCanBuy(t, nd); !ok {
		g.Msg("{{K|" + nd.Name + ": " + why + ".}}")
		return false
	}
	g.SP -= skCost(nd.Q)
	g.skGrant(nd, true)
	g.skSync()
	g.Msg(fmt.Sprintf("{{G|You learn %s.}}", nd.Name))
	return true
}

// ---- screen (K) ----

func skRows(g *Game) (rows []string) {
	// one row per tree, then its nodes when expanded (g.Tab = expanded tree index + 1, 0 none)
	for i, t := range skTrees {
		rows = append(rows, "t:"+t.ID)
		if g.Tab == i+1 {
			for _, nd := range t.Nodes {
				rows = append(rows, "n:"+nd.ID)
			}
		}
	}
	return
}

func skKeys(g *Game, k string) {
	rows := skRows(g)
	g.Sel = clampi(g.Sel, 0, len(rows)-1)
	switch k {
	case "Escape", "q", "K":
		g.Mode = "main"
	case "ArrowUp", "N8", "k":
		g.Sel = maxi(0, g.Sel-1)
	case "ArrowDown", "N2", "j":
		g.Sel = mini(len(rows)-1, g.Sel+1)
	case "PageUp":
		g.Sel = maxi(0, g.Sel-10)
	case "PageDown":
		g.Sel = mini(len(rows)-1, g.Sel+10)
	case "Enter", " ":
		r := rows[g.Sel]
		id := r[2:]
		if strings.HasPrefix(r, "t:") {
			t := skTreeByID[id]
			if !g.skState().Open[id] {
				g.skUnlock(t)
				return
			}
			for i, o := range skTrees {
				if o == t {
					if g.Tab == i+1 {
						g.Tab = 0
					} else {
						g.Tab = i + 1
					}
				}
			}
		} else {
			g.skBuy(skNodeTree[id], skNodeByID[id])
		}
	}
}

func (g *Game) skDraw() {
	s := g.sc
	x, y, w, h := g.panel("Skills")
	g.skSync()
	s.TextB(x, y, fmt.Sprintf("{{W|Skill points: %d}}  {{K|Enter learn / expand / buy · Esc}}", g.SP), CGray, CBgDark, x+w)
	rows := skRows(g)
	g.Sel = clampi(g.Sel, 0, len(rows)-1)
	lh := h - 8
	top := maxi(0, g.Sel-lh+1)
	lw := 40
	st := g.skState()
	for i := top; i < len(rows) && i-top < lh; i++ {
		bg := CBgDark
		if i == g.Sel {
			bg = CBgSel
		}
		ry := y + 2 + i - top
		s.Fill(x, ry, lw, 1, bg)
		id := rows[i][2:]
		if strings.HasPrefix(rows[i], "t:") {
			t := skTreeByID[id]
			if st.Open[id] {
				s.TextB(x, ry, fmt.Sprintf("{{G|%s}} {{K|%d/%d}}", t.Name, g.skOwned(t), len(t.Nodes)), CGray, bg, x+lw)
			} else {
				s.TextB(x, ry, fmt.Sprintf("{{K|%s}} {{y|%d SP}}", t.Name, t.UnlockSP()), CGray, bg, x+lw)
			}
		} else {
			nd := skNodeByID[id]
			mark := "{{K|·}}"
			if st.Own[id] {
				mark = "{{G|*}}"
			}
			s.TextB(x, ry, fmt.Sprintf("  %s %s {{K|%d}}", mark, nd.Name, skCost(nd.Q)), CGray, bg, x+lw)
		}
	}
	dx, dw := x+lw+2, w-lw-2
	dy := y + 2
	r := rows[g.Sel]
	var lines []string
	if strings.HasPrefix(r, "t:") {
		t := skTreeByID[r[2:]]
		lines = append(lines, "{{W|"+t.Name+"}}", fmt.Sprintf("Unlock: %d SP", t.UnlockSP()))
		if t.Attr >= 0 {
			lines = append(lines, fmt.Sprintf("Requires %s %d", AttrNames[t.Attr], t.AttrMin))
		}
		if ok, why := g.skCanUnlock(t); !ok && !st.Open[t.ID] {
			lines = append(lines, "{{r|"+why+"}}")
		}
		lines = append(lines, "", "Rank (used by older checks): "+fmt.Sprint(g.Skills[t.ID]))
	} else {
		nd := skNodeByID[r[2:]]
		lines = append(lines, "{{W|"+nd.Name+"}} {{K|("+nd.Kind+")}}", nd.Desc)
		if nd.Kind == "active" {
			lines = append(lines, fmt.Sprintf("Cooldown %d turns. Use from the abilities screen (a).", nd.Cool))
		}
		if ok, why := g.skCanBuy(skNodeTree[nd.ID], nd); !ok {
			lines = append(lines, "{{K|"+why+"}}")
		}
	}
	for _, l := range lines {
		for _, wl := range Wrap(l, dw) {
			if dy < y+h-1 {
				s.TextB(dx, dy, wl, CGray, CBgDark, x+w)
				dy++
			}
		}
	}
}

// skActiveNodes lists the owned active nodes, sorted.
func (g *Game) skActiveNodes() []*NodeDef {
	var out []*NodeDef
	for id := range g.skState().Own {
		if nd := skNodeByID[id]; nd != nil && nd.Kind == "active" {
			out = append(out, nd)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}
