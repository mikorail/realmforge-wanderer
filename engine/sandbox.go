package engine

import "fmt"

// Sandbox / creative mode: F2 or ~ opens a cheat menu. All state lives in g.ext("sandbox") so
// saves remember that a run was sandboxed; UI-only fields are excluded from the save.

type sbState struct {
	Used    bool `json:"used"`
	God     bool `json:"god"`
	Frozen  bool `json:"frozen"`
	OrigMax int  `json:"origMax,omitempty"`

	sub        string // menu tele telez spawn spawnlvl itemkind itemrar attr
	cx, cy     int
	page       int
	tplID      string
	lvl        int
	itemKind   string
	attrSel    int
	frozenMark map[*Actor]bool
}

const sbFrozenEnergy = -1000000000

var sbItemKinds = []string{"longblade", "shortblade", "axe", "cudgel", "ranged", "armor", "backpack", "oddity", "tonic", "light", "random loot"}

var sbMenu = []string{"God mode", "Reveal map", "Teleport (world map)", "Spawn creature", "Spawn item", "Level up", "Attributes & skills",
	"Give gold (+1000)", "Freeze time", "Kill all hostiles", "Full heal & cure", "Jump to nearest dungeon", "Jump to nearest tower", "Jump to nearest city"}

func init() {
	mainKeys["F2"] = sbOpen
	mainKeys["~"] = sbOpen
	modeKeys["sandbox"] = sbKey
	modeDraws["sandbox"] = sbDraw
	turnHooks = append(turnHooks, sbTurn)
	enterHooks = append(enterHooks, func(g *Game, zn *Zone) {
		if st := g.sb(); st.Frozen {
			sbFreezeZone(zn, st)
		}
	})
	sideHooks = append(sideHooks, func(g *Game) []string {
		st := g.sb()
		if !st.Used {
			return nil
		}
		tag := "{{M|SANDBOX}}"
		if st.God {
			tag += " {{W|god}}"
		}
		if st.Frozen {
			tag += " {{C|frozen}}"
		}
		return []string{tag}
	})
}

func (g *Game) sb() *sbState {
	return g.ext("sandbox", func() any { return &sbState{} }).(*sbState)
}

// Sandboxed reports whether any sandbox cheat was used in this run (tag deeds with it).
func (g *Game) Sandboxed() bool { return g.sb().Used }

func sbOpen(g *Game) {
	st := g.sb()
	st.sub = "menu"
	g.Mode, g.Sel = "sandbox", 0
}

func sbFreezeZone(zn *Zone, st *sbState) {
	for _, a := range zn.Actors {
		if a.HP > 0 {
			a.Energy = sbFrozenEnergy
		}
	}
}

func sbTurn(g *Game) {
	st := g.sb()
	if st.God && g.P != nil && !g.Dead {
		g.P.HP = g.P.MaxHP
		g.Hunger, g.Thirst = 0, 0
	}
	if st.Frozen && g.Cur != nil {
		sbFreezeZone(g.Cur, st)
	}
}

func sbSetGod(g *Game, on bool) {
	st := g.sb()
	if on == st.God {
		return
	}
	st.God = on
	if on {
		st.OrigMax = g.P.MaxHP
		g.P.MaxHP = 1000000
		g.P.HP = g.P.MaxHP
		g.Msg("{{M|[Sandbox]}} God mode on.")
	} else {
		if st.OrigMax > 0 {
			g.P.MaxHP = st.OrigMax
		}
		g.P.HP = minInt(g.P.HP, g.P.MaxHP)
		g.Msg("{{M|[Sandbox]}} God mode off.")
	}
}

func sbSetFrozen(g *Game, on bool) {
	st := g.sb()
	st.Frozen = on
	if on {
		sbFreezeZone(g.Cur, st)
		g.Msg("{{M|[Sandbox]}} Time freezes around you.")
		return
	}
	for _, a := range g.Cur.Actors {
		if a.Energy <= sbFrozenEnergy/2 {
			a.Energy = 0
		}
	}
	g.Msg("{{M|[Sandbox]}} Time flows again.")
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// sbSiteZone finds the surface zone of a parasang that roots a site of the wanted kind.
func (g *Game) sbSiteZone(p int, kind string) (int, int, bool) {
	wx, wy := p%g.W.W, p/g.W.W
	for zy := 0; zy < 3; zy++ {
		for zx := 0; zx < 3; zx++ {
			if s := g.siteAt(wx, wy, zx, zy); s != nil && (kind == "" || s.Kind == kind) {
				return zx, zy, true
			}
		}
	}
	return 1, 1, false
}

// sbNearest returns the parasang nearest the player satisfying ok (excluding the current one).
func (g *Game) sbNearest(ok func(p int) bool) int {
	best, bd := -1, 1<<30
	for p := range g.W.Biome {
		if g.W.IsWater(p) || !ok(p) {
			continue
		}
		x, y := p%g.W.W, p/g.W.W
		d := (x-g.WX)*(x-g.WX) + (y-g.WY)*(y-g.WY)
		if d < bd {
			best, bd = p, d
		}
	}
	return best
}

func (g *Game) sbJump(p int, zx, zy, z int) {
	g.OnWorld = false
	g.Known[p] = true
	g.Warp(p%g.W.W, p/g.W.W, zx, zy, z)
}

func sbRun(g *Game, i int) {
	st := g.sb()
	cheat := true
	switch i {
	case 0:
		sbSetGod(g, !st.God)
	case 1:
		for k := range g.Cur.Seen {
			g.Cur.Seen[k] = 1
		}
		for p := range g.W.Biome {
			g.Known[p] = true
		}
		g.Msg("{{M|[Sandbox]}} The map is revealed.")
	case 2:
		st.sub, st.cx, st.cy = "tele", g.WX, g.WY
		cheat = false
	case 3:
		st.sub, st.page = "spawn", 0
		cheat = false
	case 4:
		st.sub = "itemkind"
		cheat = false
	case 5:
		g.gainXP(XPFor(g.P.Level) - g.XP)
	case 6:
		st.sub, st.attrSel = "attr", 0
		cheat = false
	case 7:
		g.P.Gold += 1000
		g.Msg(fmt.Sprintf("{{M|[Sandbox]}} +1000 %s.", g.Sk.Currency))
	case 8:
		sbSetFrozen(g, !st.Frozen)
	case 9:
		n := 0
		for _, a := range append([]*Actor{}, g.Cur.Actors...) {
			if a.HP > 0 && g.Hostile(a) {
				g.hurt(a, a.HP+1000, g.P, "sandbox")
				n++
			}
		}
		g.cleanup()
		g.Msg(fmt.Sprintf("{{M|[Sandbox]}} %d hostiles slain.", n))
	case 10:
		g.P.HP = g.P.MaxHP
		g.P.Status = map[string]int{}
		g.Hunger, g.Thirst = 0, 0
		g.Breath = 30 + g.P.M(TOU)*5
		g.Msg("{{M|[Sandbox]}} You are whole.")
	case 11, 12:
		kind := "dungeon"
		if i == 12 {
			kind = "tower"
		}
		p := g.sbNearest(func(p int) bool {
			_, _, ok := g.sbSiteZone(p, kind)
			return ok
		})
		if p < 0 {
			g.Msg("No such place in this world.")
			return
		}
		zx, zy, _ := g.sbSiteZone(p, kind)
		g.sbJump(p, zx, zy, 0)
	case 13:
		p := g.sbNearest(func(p int) bool { b := g.W.burgByID[g.W.Burg[p]]; return b != nil && b.Alive })
		if p < 0 {
			g.Msg("No city in this world.")
			return
		}
		g.sbJump(p, 1, 1, 0)
	}
	if cheat {
		st.Used = true
	}
	g.recompute()
}

func sbKey(g *Game, k string) {
	st := g.sb()
	if st.sub == "" {
		st.sub = "menu"
	}
	back := func() { st.sub = "menu"; g.Sel = 0 }
	switch st.sub {
	case "menu":
		switch k {
		case "Escape", "F2", "~":
			g.Mode = "main"
			return
		case "ArrowUp":
			g.Sel = maxi(0, g.Sel-1)
			return
		case "ArrowDown":
			g.Sel = minInt(len(sbMenu)-1, g.Sel+1)
			return
		case "Enter":
			sbRun(g, g.Sel)
			return
		}
		if i := letterIndex(k); i >= 0 && i < len(sbMenu) {
			g.Sel = i
			sbRun(g, i)
		}
	case "tele":
		if dx, dy, ok := arrowOf(k); ok {
			st.cx = clampi(st.cx+dx, 0, g.W.W-1)
			st.cy = clampi(st.cy+dy, 0, g.W.H-1)
			return
		}
		switch k {
		case "Escape":
			back()
		case "Enter":
			if g.W.IsWater(g.W.Idx(st.cx, st.cy)) {
				g.Msg("That is open water.")
				return
			}
			st.sub = "telez"
		}
	case "telez":
		if k == "Escape" {
			st.sub = "tele"
			return
		}
		p := g.W.Idx(st.cx, st.cy)
		zx, zy, has := g.sbSiteZone(p, "")
		var s *Site
		if has {
			s = g.siteAt(p%g.W.W, p/g.W.W, zx, zy)
		}
		switch k {
		case "a":
			st.Used = true
			g.sbJump(p, 1, 1, 0)
		case "b":
			if s == nil || s.Kind != "dungeon" {
				g.Msg("No dungeon beneath that parasang.")
				return
			}
			st.Used = true
			g.sbJump(p, zx, zy, 1)
		case "c":
			if s == nil || s.Kind != "tower" {
				g.Msg("No tower on that parasang.")
				return
			}
			st.Used = true
			g.sbJump(p, zx, zy, towerBase-1)
		case "d":
			st.Used = true
			g.sbJump(p, 1, 1, -1)
		default:
			return
		}
		sbAfterWarp(g)
	case "spawn":
		pages := (len(Templates) + 19) / 20
		switch k {
		case "Escape":
			back()
		case "ArrowRight", "ArrowDown", "PageDown":
			st.page = (st.page + 1) % pages
		case "ArrowLeft", "ArrowUp", "PageUp":
			st.page = (st.page + pages - 1) % pages
		default:
			if i := letterIndex(k); i >= 0 && st.page*20+i < len(Templates) && i < 20 {
				t := Templates[st.page*20+i]
				st.tplID, st.lvl, st.sub = t.ID, maxi(t.Lvl, g.P.Level), "spawnlvl"
			}
		}
	case "spawnlvl":
		switch k {
		case "Escape":
			st.sub = "spawn"
		case "ArrowRight", "+", "=":
			st.lvl = minInt(st.lvl+1, 60)
		case "ArrowLeft", "-":
			st.lvl = maxi(1, st.lvl-1)
		case "ArrowUp":
			st.lvl = minInt(st.lvl+5, 60)
		case "ArrowDown":
			st.lvl = maxi(1, st.lvl-5)
		case "Enter":
			if a := sbSpawn(g, st.tplID, st.lvl); a != nil {
				st.Used = true
				g.Msg(fmt.Sprintf("{{M|[Sandbox]}} %s (level %d) appears.", a.Name, a.Level))
				if st.Frozen {
					a.Energy = sbFrozenEnergy
				}
			} else {
				g.Msg("No room to spawn.")
			}
			g.Mode = "main"
			g.recompute()
		}
	case "itemkind":
		if k == "Escape" {
			back()
		} else if i := letterIndex(k); i >= 0 && i < len(sbItemKinds) {
			st.itemKind, st.sub = sbItemKinds[i], "itemrar"
		}
	case "itemrar":
		if k == "Escape" {
			st.sub = "itemkind"
		} else if i := letterIndex(k); i >= 0 && i < len(rarityOrder) {
			if it := g.sbItem(st.itemKind, rarityOrder[i]); it != nil {
				g.give(it)
				st.Used = true
				g.Msg(fmt.Sprintf("{{M|[Sandbox]}} Gained %s.", it.Display()))
			}
			g.Mode = "main"
			g.recompute()
		}
	case "attr":
		n := 6 + len(SkillIDs)
		switch k {
		case "Escape":
			back()
		case "ArrowUp":
			st.attrSel = (st.attrSel + n - 1) % n
		case "ArrowDown":
			st.attrSel = (st.attrSel + 1) % n
		case "ArrowRight", "+", "=", "ArrowLeft", "-":
			d := 1
			if k == "ArrowLeft" || k == "-" {
				d = -1
			}
			if st.attrSel < 6 {
				g.P.Attr[st.attrSel] = clampi(g.P.Attr[st.attrSel]+d, 1, 60)
			} else {
				id := SkillIDs[st.attrSel-6]
				g.Skills[id] = clampi(g.Skills[id]+d, 0, 12)
			}
			st.Used = true
			g.recompute()
		}
	}
}

// sbAfterWarp: Warp resets the mode, so reopen nothing; the menu closes on teleport.
func sbAfterWarp(g *Game) {
	g.Mode = "main"
	if st := g.sb(); st.Frozen {
		sbFreezeZone(g.Cur, st)
	}
}

func sbSpawn(g *Game, id string, lvl int) *Actor {
	z := g.Cur
	for r := 1; r <= 4; r++ {
		var cells [][2]int
		for dy := -r; dy <= r; dy++ {
			for dx := -r; dx <= r; dx++ {
				x, y := g.P.X+dx, g.P.Y+dy
				if (dx != 0 || dy != 0) && z.In(x, y) && z.Walkable(x, y) && z.ActorAt(x, y) == nil {
					cells = append(cells, [2]int{x, y})
				}
			}
		}
		if len(cells) > 0 {
			c := Pick(g.R, cells)
			a := g.Spawn(id, lvl, g.R)
			a.X, a.Y = c[0], c[1]
			a.AI = "hostile"
			z.Actors = append(z.Actors, a)
			return a
		}
	}
	return nil
}

func (g *Game) sbItem(kind, rarity string) *Item {
	r := NewRng(fmt.Sprintf("sandbox:%d:%d", g.Turn, g.NextID))
	tier := clampi(g.P.Level/4, 0, 5)
	var it *Item
	switch kind {
	case "longblade", "shortblade", "axe", "cudgel":
		it = g.MakeWeapon(kind, tier, r)
	case "ranged":
		it = g.MakeRanged(tier, r)
	case "armor":
		it = g.MakeArmor(Pick(r, []string{"body", "head", "feet", "hands", "back", "face"}), tier, r)
	case "backpack":
		it = g.MakeBackpack(tier)
	case "oddity":
		it = g.MakeOddity(r, tier)
	case "tonic":
		it = g.MakeTonic("heal", 3)
	case "light":
		it = g.MakeLight(tier % 3)
	default:
		it = g.Loot(r, tier)
	}
	if it == nil {
		return nil
	}
	if rarity == "unique" {
		return g.MakeUnique(it, "Sandbox "+it.Name, "Conjured from nothing.")
	}
	if rarity != "common" || isGear(it) {
		g.applyRarity(it, r, rarity)
	}
	return it
}

// ---- drawing ----

func sbDraw(g *Game) {
	st := g.sb()
	s := g.sc
	switch st.sub {
	case "tele":
		sbDrawTele(g, st)
		return
	}
	x, y, w, _ := g.panel("Sandbox")
	row := func(sel bool, txt string) {
		bg := CBgDark
		if sel {
			bg = CBgSel
		}
		s.Fill(x, y, w, 1, bg)
		s.TextB(x, y, txt, CGray, bg, x+w)
		y++
	}
	switch st.sub {
	case "spawn":
		s.TextB(x, y, "{{K|Spawn creature — letter picks, arrows change page}}", CGray, CBgDark, x+w)
		y += 2
		lo := st.page * 20
		for i := 0; i < 20 && lo+i < len(Templates); i++ {
			t := &Templates[lo+i]
			row(false, fmt.Sprintf("{{W|%c)}} {{Y|%c}} %s {{K|lv%d}}", 'a'+i, t.Glyph, t.Name(g.Genre), t.Lvl))
		}
		s.TextB(x, y+1, fmt.Sprintf("{{K|page %d/%d}}", st.page+1, (len(Templates)+19)/20), CGray, CBgDark, x+w)
	case "spawnlvl":
		s.TextB(x, y, fmt.Sprintf("Spawn {{Y|%s}} at level {{W|%d}}", st.tplID, st.lvl), CGray, CBgDark, x+w)
		s.TextB(x, y+2, "{{K|left/right +-1, up/down +-5, Enter spawn, Esc back}}", CGray, CBgDark, x+w)
	case "itemkind":
		s.TextB(x, y, "{{K|Spawn item — kind}}", CGray, CBgDark, x+w)
		y += 2
		for i, k := range sbItemKinds {
			row(false, fmt.Sprintf("{{W|%c)}} %s", 'a'+i, k))
		}
	case "itemrar":
		s.TextB(x, y, fmt.Sprintf("{{K|Spawn %s — rarity}}", st.itemKind), CGray, CBgDark, x+w)
		y += 2
		for i, r := range rarityOrder {
			row(false, fmt.Sprintf("{{W|%c)}} {{%s|%s}}", 'a'+i, rarityMarkup[r], RarityLabel(r)))
		}
	case "telez":
		s.TextB(x, y, fmt.Sprintf("{{K|Teleport to parasang %d,%d — layer}}", st.cx, st.cy), CGray, CBgDark, x+w)
		y += 2
		for i, l := range []string{"Surface", "Dungeon", "Tower", "Sky isles"} {
			row(false, fmt.Sprintf("{{W|%c)}} %s", 'a'+i, l))
		}
	case "attr":
		s.TextB(x, y, "{{K|Up/down select, left/right (or +/-) change, Esc back}}", CGray, CBgDark, x+w)
		y += 2
		for i := 0; i < 6+len(SkillIDs); i++ {
			if i < 6 {
				row(i == st.attrSel, fmt.Sprintf("%-14s {{W|%d}}", AttrNames[i], g.P.Attr[i]))
			} else {
				row(i == st.attrSel, fmt.Sprintf("%-14s {{W|%d}}", SkillIDs[i-6], g.Skills[SkillIDs[i-6]]))
			}
		}
	default:
		s.TextB(x, y, "{{K|Letter or Enter to act, Esc closes}}", CGray, CBgDark, x+w)
		y += 2
		for i, m := range sbMenu {
			extra := ""
			switch i {
			case 0:
				extra = sbOnOff(st.God)
			case 8:
				extra = sbOnOff(st.Frozen)
			}
			row(i == g.Sel, fmt.Sprintf("{{W|%c)}} %s %s", 'a'+i, m, extra))
		}
		if st.Used {
			s.TextB(x, y+1, "{{M|This run is marked SANDBOX.}}", CGray, CBgDark, x+w)
		}
	}
}

func sbOnOff(b bool) string {
	if b {
		return "{{G|[on]}}"
	}
	return "{{K|[off]}}"
}

func sbDrawTele(g *Game, st *sbState) {
	s := g.sc
	x, y, w, h := g.panel("Sandbox — teleport")
	vw, vh := w, h-4
	vx := clampi(st.cx-vw/2, 0, maxi(0, g.W.W-vw))
	vy := clampi(st.cy-vh/2, 0, maxi(0, g.W.H-vh))
	for j := 0; j < vh; j++ {
		for i := 0; i < vw; i++ {
			wx, wy := vx+i, vy+j
			if !g.W.In(wx, wy) {
				continue
			}
			p := g.W.Idx(wx, wy)
			gl, f, b := g.worldGlyph(p)
			if bd := g.W.burgByID[g.W.Burg[p]]; bd != nil {
				gl, f = 'o', CWhite
			}
			s.Put(x+i, y+j, gl, f, b)
		}
	}
	s.Put(x+st.cx-vx, y+st.cy-vy, 'X', CYellow, CBgSel)
	p := g.W.Idx(st.cx, st.cy)
	info := fmt.Sprintf("%s — %s", titleCase(BiomeName(g.W.Biome[p])), g.W.StateName(g.W.State[p]))
	if bd := g.W.burgByID[g.W.Burg[p]]; bd != nil {
		info += " — " + bd.Name
	}
	if _, _, ok := g.sbSiteZone(p, ""); ok {
		info += " {{W|(site)}}"
	}
	s.TextB(x, y+vh+1, info, CGray, CBgDark, x+w)
	s.TextB(x, y+vh+2, "{{K|arrows move, Enter choose layer, Esc back}}", CGray, CBgDark, x+w)
}
