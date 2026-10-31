package engine

import (
	"fmt"
	"strings"
)

// Out flags the host reads after each key.
type Out struct {
	Save bool
	Exit bool
}

func dirOf(k string) (int, int, bool) {
	switch k {
	case "ArrowUp", "N8", "k":
		return 0, -1, true
	case "ArrowDown", "N2", "j":
		return 0, 1, true
	case "ArrowLeft", "N4", "h":
		return -1, 0, true
	case "ArrowRight", "N6", "l":
		return 1, 0, true
	case "Home", "N7", "y":
		return -1, -1, true
	case "PageUp", "N9", "u":
		return 1, -1, true
	case "End", "N1", "b":
		return -1, 1, true
	case "PageDown", "N3", "n":
		return 1, 1, true
	}
	return 0, 0, false
}

// arrows only (letters are commands in most modes)
func arrowOf(k string) (int, int, bool) {
	if len(k) == 1 {
		return 0, 0, false
	}
	return dirOf(k)
}

func letterIndex(k string) int {
	if len(k) == 1 && k[0] >= 'a' && k[0] <= 'z' {
		return int(k[0] - 'a')
	}
	return -1
}

func (g *Game) Key(k string) Out {
	var out Out
	defer func() {
		if g.Sel < 0 {
			g.Sel = 0
		}
	}()
	if strings.HasPrefix(k, "click:") {
		var x, y int
		fmt.Sscanf(k[6:], "%d,%d", &x, &y)
		g.click(x, y)
		return out
	}
	switch g.Mode {
	case "dead":
		if k == "Escape" || k == "Enter" {
			out.Exit = true
		}
	case "world":
		g.keyWorld(k)
	case "target", "look":
		g.keyTarget(k)
	case "dir":
		if dx, dy, ok := arrowOf(k); ok {
			g.Mode = "main"
			pend := g.Pend
			g.Pend = ""
			if strings.HasPrefix(pend, "power:") {
				g.usePower(pend[6:], nil, dx, dy)
			} else if strings.HasPrefix(pend, "art:") {
				var id int
				fmt.Sscanf(pend[4:], "%d", &id)
				for _, ab := range g.Abilities() {
					if ab.Item != nil && ab.Item.ID == id {
						g.usePower(ab.ID, ab.Item, dx, dy)
					}
				}
			}
		} else if k == "Escape" {
			g.Mode = "main"
		}
	case "popup":
		if k == "Escape" || k == "Enter" || k == " " {
			if g.Talk != nil {
				g.Mode = "talk"
			} else if g.ItemSel != nil {
				g.Mode = "inv"
			} else {
				g.Mode = "main"
			}
			g.ItemSel = nil
		}
	case "confirm":
		if k == "y" || k == "Y" {
			var x, y int
			fmt.Sscanf(strings.TrimPrefix(g.Pend, "step:"), "%d,%d", &x, &y)
			g.Mode = "main"
			g.stepTo(x, y)
		} else {
			g.Mode = "main"
		}
		g.Pend = ""
	case "talk":
		g.keyMenu(k, func() { g.Mode = "main"; g.Talk = nil })
	case "comp":
		g.keyMenu(k, func() { g.Mode = "main" })
	case "trade":
		g.keyTrade(k)
	case "inv":
		g.keyInv(k)
	case "item":
		g.keyItem(k)
	case "abilities":
		abs := g.Abilities()
		if i := letterIndex(k); i >= 0 && i < len(abs) {
			g.startAbility(abs[i])
		} else if k == "Escape" {
			g.Mode = "main"
		} else if k == "ArrowUp" {
			g.Sel = maxi(0, g.Sel-1)
		} else if k == "ArrowDown" {
			g.Sel = maxi(0, mini(len(abs)-1, g.Sel+1))
		} else if k == "Enter" && g.Sel < len(abs) {
			g.startAbility(abs[g.Sel])
		}
	case "craft":
		rs := g.Recipes()
		if i := letterIndex(k); i >= 0 && i < len(rs) {
			g.craft(rs[i])
		} else if k == "Escape" {
			g.Mode = "main"
		}
	case "char":
		g.keyChar(k)
	case "journal", "factions", "log", "help":
		switch k {
		case "Escape", "Enter":
			g.Mode = "main"
			if g.OnWorld {
				g.Mode = "world"
			}
		case "Tab", "ArrowRight":
			g.Tab = (g.Tab + 1) % 3
			g.Sel = 0
		case "ArrowLeft":
			g.Tab = (g.Tab + 2) % 3
			g.Sel = 0
		case "ArrowUp", "PageUp":
			g.Sel = maxi(0, g.Sel-1-boolInt(k == "PageUp")*10)
		case "ArrowDown", "PageDown":
			g.Sel += 1 + boolInt(k == "PageDown")*10
		}
	default:
		out = g.keyMain(k)
	}
	return out
}

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func (g *Game) keyMain(k string) Out {
	var out Out
	if g.Dead {
		return out
	}
	if dx, dy, ok := dirOf(k); ok && (len(k) > 1) {
		g.Path = nil
		g.MoveP(dx, dy)
		return out
	}
	switch k {
	case ".", "N5", "5", "Clear":
		g.spend(1000)
	case "<":
		g.Ascend()
	case ">":
		g.Descend()
	case "g", ",":
		g.pickup()
	case "i":
		g.Mode, g.Sel = "inv", 0
	case "c":
		g.Mode, g.Sel, g.Tab = "char", 0, 0
	case "j":
		g.Mode, g.Sel, g.Tab = "journal", 0, 0
	case "r":
		g.Mode, g.Sel = "factions", 0
	case "m":
		g.Mode, g.Sel = "log", 0
	case "?":
		g.Mode = "help"
	case "w", "M":
		g.openWorldMap()
	case "f":
		if g.P.Missile() == nil {
			g.Msg("You have no missile weapon equipped.")
		} else {
			g.beginTarget("fire")
		}
	case "t":
		for _, it := range g.P.Inv {
			if it.Kind == "grenade" {
				g.beginTarget(fmt.Sprintf("throw:%d", it.ID))
				return out
			}
		}
		g.Msg("You have nothing to throw. (Throw any item from the inventory.)")
	case "a":
		g.Mode, g.Sel = "abilities", 0
	case "o":
		g.openCompanions()
	case "x", "L":
		g.Mode = "look"
		g.CX, g.CY = g.P.X, g.P.Y
	case "R":
		g.rest(true)
	case "z":
		g.rest(false)
	case "s":
		g.Msg("You search carefully.")
		g.noticeTraps(4)
		g.spend(1000)
	case "d":
		if g.Cur.WaterAt(g.P.X, g.P.Y) > 0 && !g.Cur.Underwater || g.adjacent(TWell) || g.adjacent(TShallow) {
			g.drinkHere()
			g.spend(1000)
		} else {
			for _, it := range g.P.Inv {
				if it.Kind == "water" {
					g.useItem(it)
					return out
				}
			}
		}
	case "e":
		for _, it := range g.P.Inv {
			if it.Kind == "food" {
				g.useItem(it)
				return out
			}
		}
		g.Msg("You have nothing to eat.")
	case "T":
		if g.Skills["tinkering"] < 2 {
			g.Msg("You need Tinkering 2 to craft.")
		} else {
			g.Mode = "craft"
		}
	case "S":
		out.Save = true
		g.Msg("{{G|Game saved.}}")
	case "Escape":
		out.Exit = true
		out.Save = true
	}
	return out
}

func (g *Game) adjacent(t Terrain) bool {
	for _, d := range dirs8 {
		if g.Cur.At(g.P.X+d[0], g.P.Y+d[1]) == t {
			return true
		}
	}
	return false
}

func (g *Game) click(x, y int) {
	if g.Mode == "world" {
		vx, vy := g.worldView()
		wx, wy := vx+x, vy+y
		if g.W.In(wx, wy) && x < MW && y < MH {
			g.CX, g.CY = wx, wy
		}
		return
	}
	if g.Mode == "target" || g.Mode == "look" {
		if x < MW && y < MH {
			if g.CX == x && g.CY == y && g.Mode == "target" {
				g.finishTarget()
			}
			g.CX, g.CY = x, y
		}
		return
	}
	if g.Mode != "main" || x >= MW || y >= MH {
		return
	}
	// auto-travel along a path to the clicked tile
	if !g.Cur.Walkable(x, y) || g.Cur.Seen[y*MW+x] == 0 {
		return
	}
	dm := g.distMap(x, y, false)
	for steps := 0; steps < 80 && !(g.P.X == x && g.P.Y == y) && !g.Dead; steps++ {
		best := dm[g.P.Y*MW+g.P.X]
		bx, by := 0, 0
		found := false
		for _, d := range dirs8 {
			nx, ny := g.P.X+d[0], g.P.Y+d[1]
			if g.Cur.In(nx, ny) && dm[ny*MW+nx] < best && g.Cur.ActorAt(nx, ny) == nil {
				best, bx, by, found = dm[ny*MW+nx], d[0], d[1], true
			}
		}
		if !found {
			break
		}
		hp := g.P.HP
		key := g.Cur.Key
		g.MoveP(bx, by)
		if g.nearHostile() || g.P.HP < hp || g.Cur.Key != key || g.Mode != "main" {
			break
		}
	}
}

func (g *Game) keyWorld(k string) {
	if dx, dy, ok := arrowOf(k); ok {
		g.WorldMove(dx, dy)
		return
	}
	switch k {
	case ">", "Enter":
		g.EnterFromWorld()
	case "Escape", "<":
		g.Mode = "main"
		g.OnWorld = false
	case "Tab", "p":
		g.Tab = 1 - g.Tab
	case "j":
		g.Mode, g.Sel, g.Tab = "journal", 0, 0
	}
}

func (g *Game) keyTarget(k string) {
	if dx, dy, ok := arrowOf(k); ok {
		g.CX = clampi(g.CX+dx, 0, MW-1)
		g.CY = clampi(g.CY+dy, 0, MH-1)
		return
	}
	switch k {
	case "Escape":
		g.Mode = "main"
		g.Pend = ""
	case "Tab", "+":
		var cur *Actor
		idx := -1
		cands := 0
		for _, a := range g.Cur.Actors {
			if a.HP > 0 && g.visible(a.X, a.Y) && g.Hostile(a) {
				if a.X == g.CX && a.Y == g.CY {
					cur = a
					idx = cands
				}
				cands++
			}
		}
		_ = cur
		if t := g.nearestTarget(idx); t != nil {
			g.CX, g.CY = t.X, t.Y
		}
	case "Enter", "f", " ", "t":
		if g.Mode == "target" {
			g.finishTarget()
		} else {
			g.Mode = "main"
		}
	}
}

func (g *Game) keyMenu(k string, esc func()) {
	if i := letterIndex(k); i >= 0 && i < len(g.Opts) {
		g.Opts[i].Do()
		return
	}
	switch k {
	case "ArrowUp":
		g.Sel = maxi(0, g.Sel-1)
	case "ArrowDown":
		g.Sel = maxi(0, mini(len(g.Opts)-1, g.Sel+1))
	case "Enter", " ":
		if g.Sel < len(g.Opts) {
			g.Opts[g.Sel].Do()
		}
	case "Escape":
		esc()
	}
}

func (g *Game) keyTrade(k string) {
	list := g.Talk.Inv
	if g.Tab == 1 {
		list = g.P.Inv
	}
	switch k {
	case "Tab", "ArrowLeft", "ArrowRight":
		g.Tab = 1 - g.Tab
		g.Sel = 0
	case "ArrowUp":
		g.Sel = maxi(0, g.Sel-1)
	case "ArrowDown":
		g.Sel = maxi(0, mini(len(list)-1, g.Sel+1))
	case "Enter", " ":
		g.tradeAct()
		if g.Tab == 0 {
			g.Sel = mini(g.Sel, len(g.Talk.Inv)-1)
		} else {
			g.Sel = mini(g.Sel, len(g.P.Inv)-1)
		}
		g.Sel = maxi(g.Sel, 0)
	case "x":
		if g.Sel < len(list) {
			g.showPopup(Plain(list[g.Sel].Display()), strings.Join(list[g.Sel].Describe(g), "\n"))
		}
	case "Escape":
		g.talkMenu(g.greeting(g.Talk))
		g.Mode = "talk"
	}
}

// invList: the visible (tab-filtered, sorted) items; see inventory.go
func (g *Game) invList() []*Item { return g.invVisible() }

func (g *Game) keyInv(k string) { g.invKeys(k) }

func (g *Game) keyItem(k string) {
	it := g.ItemSel
	if it == nil {
		g.Mode = "inv"
		return
	}
	back := func() {
		if g.Mode == "item" {
			g.Mode = "inv"
		}
		g.ItemSel = nil
	}
	switch k {
	case "e":
		g.toggleEquip(it)
		back()
	case "u", "Enter":
		if it.Kind == "artifact" && it.Effect != "" && it.Ident {
			g.Mode = "main"
			g.startAbility(Ability{ID: it.Effect, Target: powerByID[it.Effect].Target, Item: it})
			return
		}
		if it.Slot != "" && it.Kind != "light" {
			g.toggleEquip(it)
			back()
			return
		}
		g.useItem(it)
		if g.Mode == "main" || g.Mode == "item" {
			g.Mode = "inv"
		}
		if g.Mode != "popup" {
			g.ItemSel = nil
		}
	case "d":
		g.drop(it)
		back()
	case "t":
		if g.P.Equipped(it) != nil {
			g.Msg("Unequip it first.")
			return
		}
		g.ItemSel = nil
		g.beginTarget(fmt.Sprintf("throw:%d", it.ID))
	case "x":
		g.examine(it)
	case "k":
		if g.P.Equipped(it) == nil {
			g.disassemble(it)
			back()
		}
	case "Escape":
		back()
	}
}

func (g *Game) keyChar(k string) {
	switch k {
	case "Tab", "ArrowRight", "ArrowLeft":
		g.Tab = 1 - g.Tab
		g.Sel = 0
		return
	case "Escape", "c":
		g.Mode = "main"
		return
	case "ArrowUp":
		g.Sel = maxi(0, g.Sel-1)
		return
	case "ArrowDown":
		g.Sel++
		return
	}
	if k != "Enter" && k != "+" {
		return
	}
	if g.Tab == 0 {
		// rows: 6 attributes then skills
		if g.Sel < 6 {
			if g.AP > 0 {
				g.AP--
				g.P.Attr[g.Sel]++
				if g.Sel == TOU {
					g.P.MaxHP += g.P.Level
					g.P.HP += g.P.Level
				}
			}
			return
		}
		i := g.Sel - 6
		if i < len(SkillIDs) {
			id := SkillIDs[i]
			cost := g.Skills[id] + 1
			if g.Skills[id] < 3 && g.SP >= cost {
				g.SP -= cost
				g.Skills[id]++
				if id == "endurance" {
					g.P.MaxHP += 3 * g.P.Level
				}
				g.Msg(fmt.Sprintf("You train %s to rank %d.", skillName(id), g.Skills[id]))
			}
		}
		return
	}
	var ps []PowerDef
	for _, p := range PowerDefs {
		if !p.Defect {
			ps = append(ps, p)
		}
	}
	if g.Sel < len(ps) {
		pd := ps[g.Sel]
		lv := g.P.Pow(pd.ID)
		cost := pd.Cost
		if lv > 0 {
			cost = 1
		}
		if lv < 10 && g.MP >= cost {
			g.MP -= cost
			g.P.Powers[pd.ID] = lv + 1
			if pd.ID == "wings" && lv == 0 && g.P.PartFor("wings") == nil {
				g.P.Body = append(g.P.Body, &BodyPart{Name: "Wings", Slot: "wings", Limb: true})
			}
			g.Msg(fmt.Sprintf("{{M|%s rises to level %d.}}", g.Sk.PowerName(pd.ID), lv+1))
		}
	}
}
