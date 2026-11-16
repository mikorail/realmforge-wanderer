package engine

import (
	"fmt"
	"strconv"
	"strings"
)

// Right-click context menus (prefix cm). The host asks CtxMenu(x, y) for the actions available at a
// screen cell (map cell in main mode, parasang on the world map, a row on the inventory screen) and
// runs the chosen one with CtxDo(id, x, y). Every action goes through the code the keys use: g.Key
// bumps, g.click walking, finishTarget, openTalk, wplTravel and so on, so turn costs, crime, noise
// and interruptions are exactly those of the keyboard. CtxDo re-derives the menu first and refuses an
// id that is not on it. Pref "contextMenu" (default on) turns the whole thing off.
//
// Keyboard route: the palette entry "Actions at the cursor" opens the look cursor; the shell then
// opens the menu on the cursor with a key (see CtxCursor and the notes in the shell file).

// CtxItem is one menu row.
type CtxItem struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	Hint  string `json:"hint,omitempty"` // right-hand note: a key, a distance
	Tone  string `json:"tone,omitempty"` // "hostile" | "warn" | ""
}

func init() {
	paEntries = append(paEntries, paEntry{Name: "Actions at the cursor (context menu)", Keywords: "right click menu actions context interact cursor", Cat: "Action",
		Cond: func(g *Game) bool { return g.Mode == "main" && g.Pref("contextMenu") },
		Func: func(g *Game) {
			g.Mode = "look"
			g.CX, g.CY = g.P.X, g.P.Y
			g.Msg("{{K|Move the cursor, then press a (or the Menu key) for actions.}}")
		}})
}

// CtxCursor is where the keyboard route puts the menu: the look cursor, else the player.
func (g *Game) CtxCursor() (int, int) {
	if g.P == nil {
		return 0, 0
	}
	if g.Mode == "look" {
		return g.CX, g.CY
	}
	return g.P.X, g.P.Y
}

func cmCheb(ax, ay, bx, by int) int { return maxi(abs(ax-bx), abs(ay-by)) }

// cmKeyFor names the numpad key for a step.
func cmKeyFor(dx, dy int) string {
	return map[[2]int]string{{-1, -1}: "N7", {0, -1}: "N8", {1, -1}: "N9", {-1, 0}: "N4", {1, 0}: "N6", {-1, 1}: "N1", {0, 1}: "N2", {1, 1}: "N3"}[[2]int{dx, dy}]
}

func cmSign(v int) int {
	switch {
	case v > 0:
		return 1
	case v < 0:
		return -1
	}
	return 0
}

func (g *Game) cmItemRow(y int) *Item {
	list := g.invVisible()
	g.invClampSel(len(list))
	const rows, y0 = 22, 6
	start := 0
	if g.Sel >= rows {
		start = g.Sel - rows + 1
	}
	i := start + (y - y0)
	if y < y0 || i < start || i >= len(list) || i >= start+rows {
		return nil
	}
	return list[i]
}

// cmWorldCell maps a screen cell on the world map to parasang coordinates.
func (g *Game) cmWorldCell(x, y int) (int, int, bool) {
	if x < 0 || y < 0 || x >= MW || y >= MH {
		return 0, 0, false
	}
	vx, vy := g.worldView()
	wx, wy := vx+x, vy+y
	return wx, wy, g.W.In(wx, wy)
}

// CtxMenu lists the actions at screen cell (x, y) for the current mode; nil when there are none.
func (g *Game) CtxMenu(x, y int) []CtxItem {
	if g.P == nil || g.Dead || !g.Pref("contextMenu") {
		return nil
	}
	switch g.Mode {
	case "main", "look":
		return g.cmMain(x, y)
	case "world":
		return g.cmWorld(x, y)
	case "inv":
		return g.cmInv(y)
	}
	return nil
}

func (g *Game) cmInv(y int) []CtxItem {
	it := g.cmItemRow(y)
	if it == nil {
		return nil
	}
	var out []CtxItem
	eq := g.P.Equipped(it) != nil
	if it.Slot != "" && it.Kind != "light" || it.Kind == "light" {
		if eq {
			out = append(out, CtxItem{ID: "i_equip", Label: "Unequip", Hint: "e"})
		} else {
			out = append(out, CtxItem{ID: "i_equip", Label: "Equip", Hint: "e"})
		}
	}
	if it.Slot == "" || it.Kind == "light" || it.Kind == "artifact" {
		out = append(out, CtxItem{ID: "i_use", Label: "Use", Hint: "u"})
	}
	out = append(out, CtxItem{ID: "i_examine", Label: "Examine", Hint: "x"})
	if !eq {
		out = append(out, CtxItem{ID: "i_throw", Label: "Throw", Hint: "t"})
	}
	out = append(out, CtxItem{ID: "i_drop", Label: "Drop", Hint: "d", Tone: "warn"})
	return out
}

func (g *Game) cmWorld(x, y int) []CtxItem {
	wx, wy, ok := g.cmWorldCell(x, y)
	if !ok {
		return nil
	}
	out := []CtxItem{{ID: "w_info", Label: "Info"}}
	here := wx == g.WX && wy == g.WY
	if here {
		out = append(out, CtxItem{ID: "w_enter", Label: "Step in", Hint: "Enter"})
	} else if _, mins, okr := g.wplRoute(wx, wy); okr && !g.ijNoFastTravel() {
		out = append(out, CtxItem{ID: "w_travel", Label: "Travel here", Hint: fmt.Sprintf("~%dh", mins/60)})
	}
	if !g.cmMarkedAt(wx, wy) && len(g.wpl().List) < wplMax {
		out = append(out, CtxItem{ID: "w_mark", Label: "Set waypoint here"})
	}
	out = append(out, CtxItem{ID: "w_auto", Label: "Auto-travel menu", Hint: "q"})
	return out
}

func (g *Game) cmMarkedAt(wx, wy int) bool {
	pl := g.PlanetIndex()
	for _, p := range g.wpl().List {
		if p.Planet == pl && p.WX == wx && p.WY == wy {
			return true
		}
	}
	return false
}

func (g *Game) cmMain(x, y int) []CtxItem {
	z := g.Cur
	if z == nil || !z.In(x, y) || x >= MW || y >= MH {
		return nil
	}
	out := []CtxItem{{ID: "look", Label: "Look", Hint: "x"}}
	if z.Seen[y*MW+x] == 0 {
		return out
	}
	p := g.P
	t := z.At(x, y)
	self := x == p.X && y == p.Y
	near := cmCheb(x, y, p.X, p.Y) <= 1
	walkLbl := "Walk here"
	if a := z.ActorAt(x, y); a != nil && a != p && g.visible(x, y) {
		name := Plain(g.the(a))
		switch {
		case isFollower(a):
			out = append(out, CtxItem{ID: "talk", Label: "Speak with " + name})
		case g.Hostile(a):
			out = append(out, CtxItem{ID: "attack", Label: "Attack " + name, Tone: "hostile"})
			if m := p.Missile(); m != nil && g.aaClear(a, "fire") {
				out = append(out, CtxItem{ID: "fire", Label: "Shoot " + name, Hint: "f", Tone: "hostile"})
			} else if wpReachOf(p) != nil && g.aaClear(a, "reach") {
				out = append(out, CtxItem{ID: "reach", Label: "Strike at " + name, Hint: "f", Tone: "hostile"})
			}
			if pend, it := g.cmThrowable(); it != nil && g.aaClear(a, pend) {
				out = append(out, CtxItem{ID: "throw", Label: "Throw " + Plain(it.Display()) + " at " + name, Hint: "t", Tone: "hostile"})
			}
		default:
			out = append(out, CtxItem{ID: "talk", Label: "Talk to " + name, Hint: "trade"})
			out = append(out, CtxItem{ID: "steal", Label: "Pick the pockets of " + name, Tone: "warn"})
		}
		if !isFollower(a) && len(g.localCompanions()) > 0 {
			out = append(out, CtxItem{ID: "order", Label: "Order companions to attack"})
		}
		return out
	}
	if self {
		if len(z.ItemsAt(x, y)) > 0 {
			out = append(out, CtxItem{ID: "pickup", Label: "Pick up", Hint: "g"})
		}
	} else if len(z.ItemsAt(x, y)) > 0 && z.Walkable(x, y) {
		out = append(out, CtxItem{ID: "pickup", Label: "Go and pick up", Hint: "g"})
	}
	switch {
	case t == TDown || t == TUp:
		lbl, key := "Take the stairs down", ">"
		if t == TUp {
			lbl, key = "Take the stairs up", "<"
		}
		out = append(out, CtxItem{ID: "stairs", Label: lbl, Hint: key})
	case t == TDoor:
		out = append(out, CtxItem{ID: "bump", Label: "Open the door"})
	case t == TLocked:
		out = append(out, CtxItem{ID: "bump", Label: "Try the lock"})
	case t == TOre:
		out = append(out, CtxItem{ID: "bump", Label: "Mine the ore"})
	case (t == TRock || t == TWall) && p.Pow("burrow") > 0:
		out = append(out, CtxItem{ID: "bump", Label: "Dig through"})
	case t == TStall:
		out = append(out, CtxItem{ID: "steal", Label: "Lift something from the stall", Tone: "warn"})
	case !t.D().Walk && (t.D().Interact || t == TStatue):
		out = append(out, CtxItem{ID: "bump", Label: "Examine closely / use"})
	}
	if !self && z.Walkable(x, y) && z.ActorAt(x, y) == nil && !near || !self && near && z.Walkable(x, y) {
		out = append(out, CtxItem{ID: "walk", Label: walkLbl})
	}
	if self {
		out = append(out, CtxItem{ID: "search", Label: "Search", Hint: "s"})
		out = append(out, CtxItem{ID: "mark", Label: "Mark this region on your map"})
		if !g.OnWorld && z.Z == 0 {
			out = append(out, CtxItem{ID: "wmap", Label: "Open the world map", Hint: "w"})
		}
	}
	return out
}

// cmThrowable is the thing the t key would throw: a grenade, else a throwing weapon.
func (g *Game) cmThrowable() (string, *Item) {
	for _, it := range g.P.Inv {
		if it.Kind == "grenade" {
			return fmt.Sprintf("throw:%d", it.ID), it
		}
	}
	if it := g.wpThrowPick(); it != nil {
		return fmt.Sprintf("wpthrow:%d", it.ID), it
	}
	return "", nil
}

// cmApproach walks toward a cell until the player is next to it (the click walk: it stops for
// hostiles, damage and zone changes) and reports whether the player now stands adjacent.
func (g *Game) cmApproach(x, y int) bool {
	if cmCheb(x, y, g.P.X, g.P.Y) <= 1 {
		return true
	}
	z := g.Cur
	dm := g.distMap(g.P.X, g.P.Y, false)
	best, bx, by := 1<<20, 0, 0
	for _, d := range dirs8 {
		nx, ny := x+d[0], y+d[1]
		if !z.In(nx, ny) || !z.Walkable(nx, ny) || z.Seen[ny*MW+nx] == 0 || z.ActorAt(nx, ny) != nil {
			continue
		}
		if v := dm[ny*MW+nx]; v < best {
			best, bx, by = v, nx, ny
		}
	}
	if best >= 1<<20 {
		g.Msg("{{K|You can't find a way there.}}")
		return false
	}
	g.click(bx, by)
	return g.Mode == "main" && !g.Dead && g.Cur == z && cmCheb(x, y, g.P.X, g.P.Y) <= 1
}

// cmBump steps into (x, y) from where the player stands, as the arrow keys would.
func (g *Game) cmBump(x, y int) {
	if k := cmKeyFor(cmSign(x-g.P.X), cmSign(y-g.P.Y)); k != "" && cmCheb(x, y, g.P.X, g.P.Y) == 1 {
		g.Key(k)
	}
}

// CtxDo runs a menu action at (x, y). It returns the Out the keys it pressed produced.
func (g *Game) CtxDo(id string, x, y int) Out {
	var out Out
	if g.P == nil || g.Dead {
		return out
	}
	if g.Mode == "look" || g.Mode == "target" {
		g.Mode, g.Pend = "main", ""
	}
	ok := false
	for _, it := range g.CtxMenu(x, y) {
		if it.ID == id {
			ok = true
			break
		}
	}
	if !ok {
		return out
	}
	if g.Mode == "inv" {
		return g.cmDoInv(id, y)
	}
	if g.Mode == "world" {
		g.cmDoWorld(id, x, y)
		return out
	}
	z := g.Cur
	a := z.ActorAt(x, y)
	switch id {
	case "look":
		g.Mode = "look"
		g.CX, g.CY = x, y
	case "walk":
		g.click(x, y)
	case "pickup":
		if x != g.P.X || y != g.P.Y {
			g.click(x, y)
		}
		if g.Mode == "main" && g.P.X == x && g.P.Y == y {
			out = g.Key("g")
		}
	case "stairs":
		if x != g.P.X || y != g.P.Y {
			g.click(x, y)
		}
		if g.Mode == "main" && g.P.X == x && g.P.Y == y {
			if z.At(x, y) == TUp {
				out = g.Key("<")
			} else {
				out = g.Key(">")
			}
		}
	case "search":
		out = g.Key("s")
	case "mark":
		g.wpUse("wp_mark", 0, 0)
	case "wmap":
		out = g.Key("w")
	case "bump":
		if g.cmApproach(x, y) {
			g.cmBump(x, y)
		}
	case "talk":
		if a == nil {
			return out
		}
		id := a.ID
		if g.cmApproach(a.X, a.Y) {
			if b := g.cmActorByID(id); b != nil && cmCheb(b.X, b.Y, g.P.X, g.P.Y) <= 1 {
				if isFollower(b) {
					g.openTalk(b)
				} else {
					g.cmBump(b.X, b.Y)
				}
			}
		}
	case "attack":
		if a == nil {
			return out
		}
		id := a.ID
		if g.cmApproach(a.X, a.Y) {
			if b := g.cmActorByID(id); b != nil && g.Hostile(b) && cmCheb(b.X, b.Y, g.P.X, g.P.Y) <= 1 {
				g.cmBump(b.X, b.Y)
			}
		}
	case "fire", "reach", "throw":
		if a == nil {
			return out
		}
		pend := id
		if id == "fire" || id == "reach" {
			pend = id
		} else {
			pend, _ = g.cmThrowable()
		}
		g.aaSt().Last = a.ID
		g.Pend = pend
		g.CX, g.CY = a.X, a.Y
		g.finishTarget()
	case "order":
		if a == nil {
			return out
		}
		g.Pend = "order:0"
		g.CX, g.CY = a.X, a.Y
		g.finishTarget()
	case "steal":
		if a != nil {
			id := a.ID
			if g.cmApproach(a.X, a.Y) {
				if b := g.cmActorByID(id); b != nil && cmCheb(b.X, b.Y, g.P.X, g.P.Y) <= 1 {
					g.uwActorMenu(b, false)
				}
			}
			return out
		}
		if g.cmApproach(x, y) {
			if k := g.uwKeeperFor(x, y); k != nil {
				g.uwActorMenu(k, true)
			} else {
				g.uwUnattended(x, y)
			}
		}
	}
	return out
}

func (g *Game) cmActorByID(id int) *Actor {
	for _, a := range g.Cur.Actors {
		if a.ID == id && a.HP > 0 {
			return a
		}
	}
	return nil
}

func (g *Game) cmDoInv(id string, y int) Out {
	var out Out
	it := g.cmItemRow(y)
	if it == nil {
		return out
	}
	g.ItemSel = it
	g.Mode = "item"
	keys := map[string]string{"i_use": "u", "i_equip": "e", "i_drop": "d", "i_throw": "t", "i_examine": "x"}
	if k := keys[id]; k != "" {
		out = g.Key(k)
	}
	return out
}

func (g *Game) cmDoWorld(id string, x, y int) {
	wx, wy, ok := g.cmWorldCell(x, y)
	if !ok {
		return
	}
	switch id {
	case "w_info":
		g.CX, g.CY = wx, wy
	case "w_enter":
		g.Key("Enter")
	case "w_auto":
		g.Key("q")
	case "w_travel":
		g.CX, g.CY = wx, wy
		g.wplTravel(&wpPlace{Name: g.cmWorldName(wx, wy), WX: wx, WY: wy, Planet: g.PlanetIndex()})
	case "w_mark":
		st := g.wpl()
		pl := g.PlanetIndex()
		st.List = append(st.List, wpPlace{Name: g.cmWorldName(wx, wy), WX: wx, WY: wy, Icon: wplIcons[len(st.List)%len(wplIcons)], Planet: pl, Turn: g.Turn})
		st.touch()
		g.CX, g.CY = wx, wy
		g.Msg(fmt.Sprintf("You mark {{C|%s}} on your map.", st.List[len(st.List)-1].Name))
	}
}

// cmWorldName names a parasang: the town on it, else "Waypoint N".
func (g *Game) cmWorldName(wx, wy int) string {
	if bd := g.W.burgByID[g.W.Burg[g.W.Idx(wx, wy)]]; bd != nil && strings.TrimSpace(bd.Name) != "" {
		return bd.Name
	}
	return fmt.Sprintf("Waypoint %d", len(g.wpl().List)+1)
}

// cmKey runs a context-menu action sent as the pseudo key "ctx:<id>:<x>:<y>". The shell sends
// menu picks this way so they pass through Key and land in the replay log like any other input.
func (g *Game) cmKey(k string) Out {
	// the id may hold colons itself: the two coordinates are the last two fields
	p := strings.Split(strings.TrimPrefix(k, "ctx:"), ":")
	if len(p) < 3 {
		return Out{}
	}
	x, ex := strconv.Atoi(p[len(p)-2])
	y, ey := strconv.Atoi(p[len(p)-1])
	if ex != nil || ey != nil {
		return Out{}
	}
	return g.CtxDo(strings.Join(p[:len(p)-2], ":"), x, y)
}
