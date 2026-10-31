package engine

import (
	"fmt"
	"sort"
	"strings"
)

// Render draws the whole frame for the current mode.
func (g *Game) Render() *Screen {
	s := g.sc
	s.Clear()
	switch g.Mode {
	case "world":
		g.drawWorld()
	case "journal":
		g.drawMap()
		g.drawJournal()
		return s
	default:
		g.drawMap()
	}
	g.drawSide()
	g.drawLog()
	switch g.Mode {
	case "inv", "item":
		g.drawInv()
	case "char":
		g.drawChar()
	case "factions":
		g.drawFactions()
	case "log":
		g.drawFullLog()
	case "help":
		g.drawHelp()
	case "talk":
		g.drawTalk()
	case "comp":
		g.drawCompanions()
	case "trade":
		g.drawTrade()
	case "popup", "confirm":
		g.drawPopup()
	case "abilities":
		g.drawAbilities()
	case "craft":
		g.drawCraft()
	case "cook":
		g.drawCook()
	case "dead":
		g.drawDead()
	case "target", "look":
		g.drawCursor()
	case "dir":
		s.TextB(1, 0, " Choose a direction (arrows / numpad), Esc to cancel ", CYellow, CBgDark, MW)
	}
	return s
}

func (g *Game) drawMap() {
	s := g.sc
	z := g.Cur
	anim := g.Tick / 10
	wo := g.weatherOverlayKind()
	for y := 0; y < MH; y++ {
		for x := 0; x < MW; x++ {
			i := y*MW + x
			t := z.T[i]
			d := t.D()
			gl, f, b := d.G, d.F, d.B
			if t == TWall && z.Kind == "town" {
				f = z.WallF
			}
			if z.Underwater && (z.Dry == nil || z.Dry[i] == false) {
				b = CDarkBlue
			}
			if t == TShallow || t == TDeep {
				if (x+y+anim/3)%7 == 0 {
					gl = map[Terrain]rune{TShallow: '≈', TDeep: '~'}[t]
				}
			}
			if t == TVoid && (x*7+y*13)%23 == 0 {
				gl, f = '·', CDarkBlue
			}
			switch z.Liq[i] {
			case 1:
				b = CBgRed
			case 2:
				b = CBrown
			case 3:
				f = CGreen
			}
			if z.Fire[i] > 0 {
				gl = '^'
				f = []byte{CRed, COrange, CYellow}[(i+anim)%3]
			}
			vis := g.visible(x, y)
			if !vis {
				if z.Seen[i] == 0 {
					continue
				}
				f, b = CDarkGray, CBgDark
				if t == TVoid || t == TChasm {
					gl = ' '
				}
				s.Put(x, y, gl, f, b)
				continue
			}
			if wo != nil && z.Fire[i] == 0 {
				gl, f = g.overlayTile(wo, x, y, t, gl, f, anim)
			}
			if its := z.ItemsAt(x, y); len(its) > 0 {
				it := its[len(its)-1]
				gl, f = it.Glyph, it.F
				if it.Kind == "artifact" && !it.Ident {
					gl = '÷'
				}
			}
			s.Put(x, y, gl, f, b)
		}
	}
	for _, a := range z.Actors {
		if a.HP <= 0 || !z.In(a.X, a.Y) {
			continue
		}
		i := a.Y*MW + a.X
		if g.visible(a.X, a.Y) {
			bg := g.sc.B[a.Y*SW+a.X]
			if a.Boss {
				bg = CBgRed
			}
			if a.AI == "follower" {
				bg = CBgSel
			}
			s.Put(a.X, a.Y, a.Glyph, a.F, bg)
		} else if g.sensed[i] {
			s.Put(a.X, a.Y, a.Glyph, CDarkMagenta, CBgDark)
		}
	}
	pf := CWhite
	if g.P.St("phased") > 0 {
		pf = CDarkMagenta
	} else if g.P.HP < g.P.MaxHP/4 {
		pf = CRed
	}
	s.Put(g.P.X, g.P.Y, '@', pf, s.B[g.P.Y*SW+g.P.X])
}

func hpColor(hp, max int) string {
	switch {
	case hp*3 > max*2:
		return "G"
	case hp*3 > max:
		return "W"
	}
	return "R"
}

func (g *Game) drawSide() {
	s := g.sc
	x0 := MW + 1
	s.Fill(MW, 0, SW-MW, SH, CBgDark)
	for y := 0; y < SH; y++ {
		s.Put(MW, y, '│', CDarkCyan, CBgDark)
	}
	p := g.P
	y := 0
	line := func(t string) {
		s.TextB(x0, y, t, CGray, CBgDark, SW)
		y++
	}
	line("{{Y|" + p.Name + "}}")
	sp := g.W.specByID[p.Species]
	spn := p.Species
	if sp != nil {
		spn = sp.Name
	}
	line(fmt.Sprintf("{{c|%s %s}}", spn, g.Sk.Calling(g.Calling)))
	line(fmt.Sprintf("Level %d  {{K|XP %d/%d}}", p.Level, g.XP, XPFor(p.Level)))
	a, b := Bar(p.HP, p.MaxHP, 20)
	s.TextB(x0, y, fmt.Sprintf("HP {{%s|%d}}/%d", hpColor(p.HP, p.MaxHP), p.HP, p.MaxHP), CGray, CBgDark, SW)
	y++
	for i := 0; i < a; i++ {
		s.Put(x0+i, y, '█', colorCode[hpColor(p.HP, p.MaxHP)[0]], CBgDark)
	}
	for i := 0; i < b; i++ {
		s.Put(x0+a+i, y, '░', CDarkGray, CBgDark)
	}
	y++
	line(fmt.Sprintf("{{C|◆}}AV %d  {{K|○}}DV %d  {{m|ψ}}MA %d", p.AV(), p.DV(), p.MA()))
	line(fmt.Sprintf("Quickness %d  {{W|$}}%d", maxi(p.Quick()+g.encumbranceQuick(), 20), p.Gold))
	at := ""
	for i := 0; i < 6; i++ {
		at += fmt.Sprintf("{{K|%s}}%d ", AttrShort[i], p.Score(i))
		if i == 2 {
			line(at)
			at = ""
		}
	}
	line(at)
	hs := "{{G|Sated}}"
	if g.Hunger > 2400 {
		hs = "{{R|Famished}}"
	} else if g.Hunger > 1200 {
		hs = "{{W|Hungry}}"
	}
	ts := "{{B|Quenched}}"
	if g.Thirst > 1800 {
		ts = "{{R|Parched}}"
	} else if g.Thirst > 900 {
		ts = "{{W|Thirsty}}"
	}
	line(hs + " " + ts)
	if g.Meal() != "" {
		line("{{G|Well fed}} {{K|(j: cookbook)}}")
	}
	if g.Cur != nil && g.Cur.Underwater && p.Pow("gills") == 0 && !p.Has("swimmer") {
		line(fmt.Sprintf("{{B|Breath %d}}", g.Breath))
	}
	var sts []string
	for k := range p.Status {
		sts = append(sts, k)
	}
	sort.Strings(sts)
	if len(sts) > 0 {
		line("{{O|" + strings.Join(sts, ", ") + "}}")
	}
	if g.SP > 0 || g.MP > 0 || g.AP > 0 {
		line(fmt.Sprintf("{{W|Points: %dSP %dMP %dAP (c)}}", g.SP, g.MP, g.AP))
	}
	y++
	if g.Cur != nil {
		for _, l := range Wrap("{{C|"+g.Cur.Name+"}}", SW-x0-1) {
			line(l)
		}
		depth := "surface"
		if g.Cur.Z > 0 {
			depth = fmt.Sprintf("%d strata deep", g.Cur.Z)
		} else if g.Cur.Z < 0 {
			depth = "aloft"
		}
		line(fmt.Sprintf("{{K|%s · %s}}", g.W.StateName(g.W.State[g.W.Idx(g.WX, g.WY)]), depth))
		line("{{K|" + g.timeOfDay() + "}}")
		line(g.weatherLine())
	}
	y++
	if g.Mode == "world" {
		line("{{K|Here:}}")
		for _, l := range g.wmInfo {
			for _, ww := range Wrap(l, SW-x0-1) {
				line(ww)
			}
		}
	}
	if g.Mode != "world" && g.Cur != nil {
		y = g.drawCompanionsSide(x0, y)
		line("{{K|Nearby:}}")
		n := 0
		for _, a := range g.Cur.Actors {
			if a.HP <= 0 || !g.visible(a.X, a.Y) || n >= 8 || a.AI == "follower" {
				continue
			}
			n++
			col := "y"
			if g.Hostile(a) {
				col = "R"
			} else if a.AI == "follower" {
				col = "G"
			}
			nm := a.Name
			if a.Title != "" && a.Proper {
				nm += ", " + a.Title
			}
			if len([]rune(nm)) > 22 {
				nm = string([]rune(nm)[:22])
			}
			hb, _ := Bar(a.HP, a.MaxHP, 4)
			s.Put(x0, y, a.Glyph, a.F, CBgDark)
			s.TextB(x0+2, y, fmt.Sprintf("{{%s|%s}}", col, nm), CGray, CBgDark, SW-5)
			for i := 0; i < 4; i++ {
				c := CDarkGray
				if i < hb {
					c = colorCode[hpColor(a.HP, a.MaxHP)[0]]
				}
				s.Put(SW-5+i, y, '▪', c, CBgDark)
			}
			y++
		}
	}
	s.TextB(x0, SH-2, "{{K|? help  i inv  c char  j journal}}", CGray, CBgDark, SW)
	s.TextB(x0, SH-1, "{{K|a powers  f fire  w world  Esc exit}}", CGray, CBgDark, SW)
}

func (g *Game) drawLog() {
	s := g.sc
	s.Fill(0, MH, MW, SH-MH, CBgDark)
	for x := 0; x < MW; x++ {
		s.Put(x, MH, '─', CDarkCyan, CBgDark)
	}
	rows := SH - MH - 1
	var lines []string
	for i := len(g.Msgs) - 1; i >= 0 && len(lines) < rows; i-- {
		m := g.Msgs[i]
		t := m.T
		if m.Rep > 0 {
			t += fmt.Sprintf(" {{K|(x%d)}}", m.Rep+1)
		}
		w := Wrap(t, MW-2)
		for j := len(w) - 1; j >= 0 && len(lines) < rows; j-- {
			if m.Turn < g.Turn-1 {
				lines = append(lines, "{{K|"+Plain(w[j])+"}}")
			} else {
				lines = append(lines, w[j])
			}
		}
	}
	for i, l := range lines {
		s.TextB(1, SH-1-i, l, CGray, CBgDark, MW)
	}
}

// ---- world map ----

func (g *Game) worldView() (int, int) {
	vx := clampi(g.CX-MW/2, 0, maxi(0, g.W.W-MW))
	vy := clampi(g.CY-MH/2, 0, maxi(0, g.W.H-MH))
	return vx, vy
}

var statePal = []byte{CRed, CBlue, CGreen, CYellow, CMagenta, CCyan, COrange, CDarkRed, CDarkBlue, CDarkGreen, CDarkMagenta, CDarkCyan, CBrown, CDarkOrange}

func (g *Game) worldGlyph(p int) (rune, byte, byte) {
	w := g.W
	b := w.Biome[p]
	h := w.Height[p]
	if b == 0 {
		gl := '≈'
		if (p*7)%5 == 0 {
			gl = '~'
		}
		return gl, CBlue, CDarkBlue
	}
	if h >= 70 {
		return '▲', CGray, CBg
	}
	if h >= 55 {
		return '∩', CBrown, CBg
	}
	if w.River[p] > 1 {
		return '~', CBlue, CBg
	}
	switch b {
	case 1:
		return '∙', CYellow, CBg
	case 2:
		return ':', CBrown, CBg
	case 3:
		return '"', CYellow, CBg
	case 4:
		return '"', CGreen, CBg
	case 5:
		return '♣', CGreen, CBg
	case 6:
		return '♣', CDarkGreen, CBg
	case 7:
		return '♠', CGreen, CBg
	case 8:
		return '♠', CDarkGreen, CBg
	case 9:
		return '↑', CDarkGreen, CBg
	case 10:
		return '·', CGray, CBg
	case 11:
		return '░', CWhite, CBg
	case 12:
		return '~', CDarkCyan, CBg
	case 13:
		return '·', CDarkGray, CBg
	case 14:
		return '*', CMagenta, CBg
	case 15:
		return '♠', CMagenta, CBg
	case 16:
		return '≈', COrange, CDarkRed
	}
	return '.', CGray, CBg
}

func (g *Game) drawWorld() {
	s := g.sc
	w := g.W
	vx, vy := g.worldView()
	questAt := map[int]bool{}
	for _, q := range g.Quests {
		if !q.Closed {
			questAt[q.TP] = true
		}
	}
	for y := 0; y < MH; y++ {
		for x := 0; x < MW; x++ {
			wx, wy := vx+x, vy+y
			if !w.In(wx, wy) {
				continue
			}
			p := w.Idx(wx, wy)
			gl, f, b := g.worldGlyph(p)
			if g.Tab == 1 && w.State[p] > 0 && !w.IsWater(p) {
				b = statePal[w.State[p]%len(statePal)]
				f = CBlack
			}
			if g.Known[p] {
				if bd := w.burgByID[w.Burg[p]]; bd != nil {
					switch {
					case !bd.Alive:
						gl, f = 'Ω', CDarkGray
					case bd.Capital:
						gl, f = '■', CYellow
					default:
						gl, f = 'o', CWhite
					}
				}
				for _, mi := range w.monstersAt[p] {
					if !g.Slain[w.Monsters[mi].ID] {
						gl, f = '¤', CRed
					}
				}
				if len(w.wondersAt[p]) > 0 {
					gl, f = '☼', CMagenta
				}
				if w.Burg[p] == 0 {
					if s := g.parasangSite(p); s != nil {
						switch {
						case s.Theme == "abyss":
							gl, f = '◘', CMagenta
						case s.Kind == "tower":
							gl, f = 'Δ', CMagenta
						default:
							gl, f = '▼', COrange
						}
					}
				}
			}
			if questAt[p] && (g.Tick/5)%2 == 0 {
				gl, f = '!', CYellow
			}
			s.Put(x, y, gl, f, b)
		}
	}
	px, py := g.WX-vx, g.WY-vy
	s.Put(px, py, '@', CWhite, CBgSel)
	if g.CX != g.WX || g.CY != g.WY {
		s.Put(g.CX-vx, g.CY-vy, 'X', CYellow, CBgSel)
	}
	s.TextB(1, 0, " World map — arrows travel, Enter/> enter, Tab realms, Esc back ", CYellow, CBgDark, MW)
	// parasang info in the sidebar slot of the log
	p := w.Idx(g.CX, g.CY)
	info := []string{fmt.Sprintf("{{Y|%s}} — %s", titleCase(BiomeName(w.Biome[p])), w.StateName(w.State[p])), g.weatherInfo(p)}
	if cu := w.cultByID[w.Culture[p]]; cu != nil && !w.IsWater(p) {
		info = append(info, "People: "+cu.Name)
	}
	if rl := w.relByID[w.Religion[p]]; rl != nil && !w.IsWater(p) {
		info = append(info, "Faith: "+rl.Name)
	}
	var below []string
	for d := 1; d <= w.maxDepth; d++ {
		if w.DeepOpen(p, d) {
			below = append(below, fmt.Sprint(d))
		}
	}
	if len(below) > 0 {
		info = append(info, fmt.Sprintf("{{m|%s below at depth %s}}", w.LayerName("deep", "Caverns"), strings.Join(below, ",")))
	}
	if w.skyOpen[p] {
		info = append(info, "{{C|"+w.LayerName("sky", "Sky isles")+" drift above}}")
	}
	if w.seaOpen[p] {
		info = append(info, "{{B|"+w.LayerName("sea", "Sea floor")+" beneath the waves}}")
	}
	if g.Known[p] {
		if bd := w.burgByID[w.Burg[p]]; bd != nil {
			st := "{{W|" + bd.Name + "}}, " + strings.ToLower(bd.Kind)
			if !bd.Alive {
				st = "{{K|Ruins of " + bd.Name + "}}"
			}
			info = append(info, st)
		}
	}
	g.Popup = nil
	g.wmInfo = info
}

// ---- overlays ----

func (g *Game) drawCursor() {
	s := g.sc
	if g.Mode == "target" {
		for _, pt := range line(g.P.X, g.P.Y, g.CX, g.CY) {
			if pt[0] == g.CX && pt[1] == g.CY {
				break
			}
			i := pt[1]*SW + pt[0]
			s.B[i] = CBgSel
		}
	}
	i := g.CY*SW + g.CX
	s.B[i] = CYellow
	s.F[i] = CBlack
	desc := g.describeTile(g.CX, g.CY)
	head := " Look — arrows move, Esc done "
	if g.Mode == "target" {
		head = " Target — arrows/Tab, Enter fires, Esc cancels "
	}
	s.TextB(1, 0, head, CYellow, CBgDark, MW)
	lines := Wrap(desc, 46)
	bx := 2
	if g.CX < MW/2 {
		bx = MW - 50
	}
	h := len(lines) + 2
	by := MH - h - 1
	if g.CY > MH/2 {
		by = 1
	}
	s.Box(bx, by, 50, h, CDarkCyan, "")
	for k, l := range lines {
		s.TextB(bx+2, by+1+k, l, CGray, CBgDark, bx+49)
	}
}

func (g *Game) describeTile(x, y int) string {
	z := g.Cur
	if z.Seen[y*MW+x] == 0 {
		return "You haven't seen this place."
	}
	if !g.visible(x, y) {
		return "{{K|You remember " + z.At(x, y).D().Name + " here.}}"
	}
	var parts []string
	if x == g.P.X && y == g.P.Y {
		parts = append(parts, "{{Y|You.}}")
	} else if a := z.ActorAt(x, y); a != nil {
		nm := a.Name
		if a.Title != "" {
			nm += ", " + a.Title
		}
		st := "{{g|friendly}}"
		if g.Hostile(a) {
			st = "{{R|hostile}}"
		}
		wnd := "{{G|uninjured}}"
		switch {
		case a.HP*4 < a.MaxHP:
			wnd = "{{R|badly wounded}}"
		case a.HP*2 < a.MaxHP:
			wnd = "{{W|wounded}}"
		case a.HP < a.MaxHP:
			wnd = "{{g|lightly wounded}}"
		}
		parts = append(parts, fmt.Sprintf("{{W|%s}} (lv %d, %s, %s)", nm, a.Level, st, wnd))
		if a.Desc != "" {
			parts = append(parts, a.Desc)
		}
		parts = append(parts, fmt.Sprintf("{{K|%s · AV %d DV %d}}", g.FactionName(a.Faction), a.AV(), a.DV()))
		for _, bp := range a.Body {
			if bp.Item != nil {
				parts = append(parts, fmt.Sprintf("{{K|%s:}} %s", bp.Name, bp.Item.Display()))
			}
			if bp.Severed {
				parts = append(parts, "{{r|Missing "+strings.ToLower(bp.Name)+".}}")
			}
		}
	}
	for _, it := range z.ItemsAt(x, y) {
		parts = append(parts, "Item: "+it.Display())
	}
	t := z.At(x, y)
	td := "{{y|" + t.D().Name + "}}"
	if t.D().Desc != "" {
		td += ". " + t.D().Desc
	}
	parts = append(parts, td)
	if z.Fire[y*MW+x] > 0 {
		parts = append(parts, "{{R|It is on fire.}}")
	}
	if n, ok := z.Notes[y*MW+x]; ok && (t == TUp || t == TDown || t == TUpdraft) {
		parts = append(parts, n)
	}
	return strings.Join(parts, "\n")
}

func (g *Game) panel(title string) (int, int, int, int) {
	x0, y0, w, h := 4, 2, MW-8, MH+4
	g.sc.Box(x0, y0, w, h, CDarkCyan, title)
	return x0 + 2, y0 + 1, w - 4, h - 2
}

func (g *Game) drawInv() { g.drawInventory() }

func (g *Game) drawChar() {
	s := g.sc
	x, y, w, _ := g.panel("Character")
	p := g.P
	tabs := []string{"Attributes & Skills", g.Sk.PowersName, titleCase(g.augSkin().Noun) + "s"}
	tx := x
	for i, t := range tabs {
		c := "K"
		if g.Tab == i {
			c = "Y"
		}
		tx = s.TextB(tx, y, fmt.Sprintf("{{%s|[ %s ]}} ", c, t), CGray, CBgDark, x+w)
	}
	y += 2
	if g.Tab == 0 {
		s.TextB(x, y, fmt.Sprintf("{{W|Attribute points: %d   Skill points: %d}}", g.AP, g.SP), CGray, CBgDark, x+w)
		y += 2
		for i := 0; i < 6; i++ {
			bg := CBgDark
			if g.Sel == i {
				bg = CBgSel
			}
			s.Fill(x, y, 40, 1, bg)
			md := p.M(i)
			ms := fmt.Sprintf("%+d", md)
			s.TextB(x, y, fmt.Sprintf("%-13s {{W|%2d}} {{K|(%s)}}", AttrNames[i], p.Score(i), ms), CGray, bg, x+40)
			y++
		}
		y++
		for i, id := range SkillIDs {
			bg := CBgDark
			if g.Sel == 6+i {
				bg = CBgSel
			}
			s.Fill(x, y, w, 1, bg)
			rk := g.Skills[id]
			stars := strings.Repeat("■", rk) + strings.Repeat("·", 3-rk)
			s.TextB(x, y, fmt.Sprintf("%-12s {{G|%s}} {{K|%s}}", skillName(id), stars, skillDesc[id]), CGray, bg, x+w)
			y++
		}
		s.TextB(x, y+1, "{{K|↑↓ select · Enter spend (attribute 1 AP, skill rank+1 SP) · Tab powers · Esc}}", CGray, CBgDark, x+w)
		return
	}
	if g.Tab == 2 {
		for _, ln := range g.augSheet() {
			s.TextB(x, y, ln, CGray, CBgDark, x+w)
			y++
		}
		return
	}
	s.TextB(x, y, fmt.Sprintf("{{W|%s points: %d}}  {{K|new: cost · upgrade: 1}}", g.Sk.PowersName, g.MP), CGray, CBgDark, x+w)
	y += 2
	row := 0
	for _, pd := range PowerDefs {
		if pd.Defect {
			continue
		}
		bg := CBgDark
		if g.Sel == row {
			bg = CBgSel
		}
		s.Fill(x, y, w, 1, bg)
		lv := p.Powers[pd.ID]
		lvs := "{{K|—}}"
		if lv > 0 {
			lvs = fmt.Sprintf("{{M|lv %d}}", lv)
		}
		act := ""
		if pd.Active {
			act = "{{c|active}} "
		}
		s.TextB(x, y, fmt.Sprintf("%-22s %s {{K|(%d)}} %s{{K|%s}}", g.Sk.PowerName(pd.ID), lvs, pd.Cost, act, pd.Desc), CGray, bg, x+w)
		y++
		row++
	}
	for id := range p.Powers {
		if pd := powerByID[id]; pd != nil && pd.Defect {
			s.TextB(x, y, "{{r|Defect: "+titleCase(id)+"}} {{K|"+pd.Desc+"}}", CGray, CBgDark, x+w)
			y++
		}
	}
}

func (g *Game) drawJournal() {
	s := g.sc
	s.Box(0, 0, SW, SH, CDarkCyan, "Journal")
	tabs := []string{"Quests", "Secrets & Places", "Chronicle of " + g.W.Name, "Cookbook"}
	tx := 2
	for i, t := range tabs {
		c := "K"
		if g.Tab == i {
			c = "Y"
		}
		tx = s.TextB(tx, 1, fmt.Sprintf("{{%s|[ %s ]}} ", c, t), CGray, CBgDark, SW-1)
	}
	var lines []string
	switch g.Tab {
	case 0:
		for i := len(g.Quests) - 1; i >= 0; i-- {
			q := g.Quests[i]
			st := "{{W|active}}"
			if q.Closed {
				st = "{{K|complete}}"
			} else if q.Done {
				st = "{{G|return to " + q.Giver + "}}"
			}
			lines = append(lines, fmt.Sprintf("{{Y|%s}} — %s", q.Title, st))
			lines = append(lines, Wrap("  "+Plain(q.Text)+" {{K|(from "+q.Giver+")}}", SW-6)...)
			lines = append(lines, "")
		}
		if len(lines) == 0 {
			lines = []string{"{{K|No quests yet. Elders and notable people may have work.}}"}
		}
	case 1:
		for i := len(g.Secrets) - 1; i >= 0; i-- {
			sc := g.Secrets[i]
			lines = append(lines, Wrap(fmt.Sprintf("{{c|[%s]}} %s", sc.Kind, sc.Text), SW-6)...)
		}
		if len(lines) == 0 {
			lines = []string{"{{K|Perform the " + g.Sk.Ritual + " with people to learn secrets.}}"}
		}
	case 2:
		for i := len(g.W.Events) - 1; i >= 0 && len(lines) < 400; i-- {
			e := g.W.Events[i]
			lines = append(lines, Wrap(fmt.Sprintf("{{K|Y%d}} %s", e.Y, e.Text), SW-6)...)
		}
	case 3:
		lines = g.cookbookLines()
	}
	g.Sel = clampi(g.Sel, 0, maxi(0, len(lines)-(SH-5)))
	for i := 0; i < SH-5 && g.Sel+i < len(lines); i++ {
		s.TextB(2, 3+i, lines[g.Sel+i], CGray, CBgDark, SW-1)
	}
	s.TextB(2, SH-1, " ←→ tabs · ↑↓ PgUp PgDn scroll · Esc close ", CYellow, CBgDark, SW-1)
}

func (g *Game) drawFactions() {
	s := g.sc
	x, y, w, h := g.panel("Reputation")
	keys := sortedKeys(g.Rep)
	sort.SliceStable(keys, func(i, j int) bool { return g.Rep[keys[i]] > g.Rep[keys[j]] })
	g.Sel = clampi(g.Sel, 0, maxi(0, len(keys)-h+2))
	for i := g.Sel; i < len(keys) && y < 2+h; i++ {
		k := keys[i]
		s.TextB(x, y, fmt.Sprintf("%-46s %5d  %s", g.FactionName(k), g.Rep[k], RepWord(g.Rep[k])), CGray, CBgDark, x+w)
		y++
	}
}

func (g *Game) drawFullLog() {
	s := g.sc
	x, y, w, h := g.panel("Message log")
	var lines []string
	for _, m := range g.Msgs {
		lines = append(lines, Wrap(m.T, w)...)
	}
	start := maxi(0, len(lines)-h-g.Sel)
	for i := start; i < len(lines) && y < 2+h+1; i++ {
		s.TextB(x, y, lines[i], CGray, CBgDark, x+w)
		y++
	}
}

func (g *Game) drawHelp() {
	s := g.sc
	x, y, w, _ := g.panel("Help")
	help := []string{
		"{{Y|Moving}}  arrows / numpad / Home PgUp End PgDn (diagonals). Click a tile to walk there.",
		"Walk off a screen's edge to reach the next zone. Each parasang is 3×3 zones; {{W|<}} on the surface opens the world map.",
		"{{W|>}} descend stairs into caverns and the " + g.W.LayerName("deep", "deep") + "; {{W|<}} climb up, or ride a " + g.Sk.Updraft + " to the " + g.W.LayerName("sky", "sky") + ".",
		"",
		"{{Y|Fighting}}  bump to attack. To-hit: d20 + AGI vs DV. Each hit rolls penetration ({{c|→}}PV vs {{C|◆}}AV) up to 3 times.",
		"Axes sever limbs, cudgels stun, blades make foes bleed. {{W|f}} fire a missile weapon, {{W|t}} throw, {{W|a}} " + strings.ToLower(g.Sk.PowersName) + " and artifacts.",
		"",
		"{{Y|People}}  bump a friendly person to talk. The {{C|" + g.Sk.Ritual + "}} earns reputation and a secret from the world's real history.",
		"Elders give quests; merchants, smiths and apothecaries trade; harbourmasters sell sea passage; priests heal.",
		"{{W|o}} companions: orders (follow, wait, guard, attack, keep distance, stay out of fights), talk, gifts of gear. Loyalty follows your name with their realm and faith.",
		"",
		"{{Y|Survival}}  {{W|e}} eat, {{W|d}} drink (from water, wells or your " + g.Sk.Water + "), {{W|R}} rest until healed, {{W|z}} rest a while, {{W|C}} cook (combine ingredients for a day of buffs).",
		"Underground is dark: keep a " + g.Sk.Light[0] + " lit. Underwater you must hold your breath unless you have " + g.Sk.Gills() + ".",
		"",
		"{{Y|Delving}}  ▼ dungeons and Δ towers on the world map. {{W|s}} searches for traps (^). Keys open locked vaults; Tinkering 2 picks locks. Bump ore to mine it.",
		"Cities are several zones wide: market, harbor, temple, scholars, barracks, arena, forges. Trainers teach skills for coin; the arena master pays for bouts.",
		"",
		"{{Y|Screens}}  {{W|i}} inventory  {{W|c}} character  {{W|j}} journal  {{W|r}} reputation  {{W|m}} log  {{W|x}} look  {{W|T}} craft  {{W|g}} pick up",
		"{{W|S}} save · {{W|Esc}} save and return to the chronicle. Slain legends and recovered relics are written into Realmforge's history.",
	}
	for _, l := range help {
		for _, ww := range Wrap(l, w) {
			s.TextB(x, y, ww, CGray, CBgDark, x+w)
			y++
		}
	}
}

func (g *Game) drawTalk() {
	s := g.sc
	a := g.Talk
	if a == nil {
		return
	}
	title := a.Name
	if a.Title != "" {
		title += ", " + a.Title
	}
	h := len(g.Popup) + len(g.Opts) + 5
	y0 := maxi(1, (MH-h)/2)
	// dialogue covers the whole screen width so long lines and skill checks fit
	s.Box(2, y0, SW-4, h, CDarkCyan, title)
	s.Put(4, y0+1, a.Glyph, a.F, CBgDark)
	y := y0 + 1
	for _, l := range g.Popup {
		s.TextB(6, y, l, CGray, CBgDark, SW-3)
		y++
	}
	y++
	for i, o := range g.Opts {
		bg := CBgDark
		if i == g.Sel {
			bg = CBgSel
		}
		s.Fill(4, y, SW-8, 1, bg)
		s.TextB(4, y, fmt.Sprintf("{{W|%c)}} %s", 'a'+i, o.Text), CGray, bg, SW-3)
		y++
	}
}

func (g *Game) drawTrade() {
	s := g.sc
	a := g.Talk
	x, y, w, _ := g.panel("Trade with " + a.Name)
	half := w / 2
	s.TextB(x, y, fmt.Sprintf("{{Y|%s}} {{W|%d %s}}", a.Name, a.Gold, g.Sk.Currency), CGray, CBgDark, x+half)
	s.TextB(x+half+1, y, fmt.Sprintf("{{Y|You}} {{W|%d %s}}", g.P.Gold, g.Sk.Currency), CGray, CBgDark, x+w)
	y += 2
	for col, list := range [][]*Item{a.Inv, g.P.Inv} {
		cx := x + col*(half+1)
		for i, it := range list {
			if i > 22 {
				break
			}
			bg := CBgDark
			if g.Tab == col && g.Sel == i {
				bg = CBgSel
			}
			s.Fill(cx, y+i, half-1, 1, bg)
			s.Put(cx, y+i, it.Glyph, it.F, bg)
			pr := g.price(it, col == 0)
			s.TextB(cx+2, y+i, it.Display(), CGray, bg, cx+half-8)
			s.TextB(cx+half-7, y+i, fmt.Sprintf("{{W|%5d}}", pr), CGray, bg, cx+half)
		}
	}
	s.TextB(x, MH+4, "{{K|Tab switch side · ↑↓ select · Enter buy/sell · x examine · Esc back}}", CGray, CBgDark, x+w)
}

func (g *Game) drawPopup() {
	s := g.sc
	h := len(g.Popup) + 3
	w := 74
	y0 := maxi(1, (MH-h)/2)
	s.Box(3, y0, w, h, CYellow, g.PopT)
	for i, l := range g.Popup {
		s.TextB(5, y0+1+i, l, CGray, CBgDark, 3+w-1)
	}
	hint := "[Enter]"
	if g.Mode == "confirm" {
		hint = "[y/n]"
	}
	s.TextB(3+w-10, y0+h-1, hint, CYellow, CBgDark, 3+w)
}

func (g *Game) drawAbilities() {
	s := g.sc
	x, y, w, _ := g.panel(g.Sk.PowersName + " & artifacts")
	abs := g.Abilities()
	if len(abs) == 0 {
		s.TextB(x, y, "{{K|You have no active abilities. Gain them on the character screen (c) or find artifacts.}}", CGray, CBgDark, x+w)
	}
	for i, ab := range abs {
		bg := CBgDark
		if i == g.Sel {
			bg = CBgSel
		}
		s.Fill(x, y, w, 1, bg)
		cd := ""
		if ab.Cool > 0 && ab.Item == nil {
			cd = fmt.Sprintf(" {{K|(%d)}}", ab.Cool)
		}
		s.TextB(x, y, fmt.Sprintf("{{W|%c)}} {{M|%s}}%s {{K|%s}}", 'a'+i, ab.Name, cd, ab.Target), CGray, bg, x+w)
		y++
	}
}

func (g *Game) drawCraft() {
	s := g.sc
	x, y, w, _ := g.panel("Tinkering")
	s.TextB(x, y, fmt.Sprintf("{{K|You have %d %s.}}", g.scrapCount(), g.Sk.Scrap), CGray, CBgDark, x+w)
	y += 2
	for i, rc := range g.Recipes() {
		c := "y"
		if g.scrapCount() < rc.Cost {
			c = "K"
		}
		s.TextB(x, y, fmt.Sprintf("{{W|%c)}} {{%s|%s}} — %d %s", 'a'+i, c, rc.Name, rc.Cost, g.Sk.Scrap), CGray, CBgDark, x+w)
		y++
	}
}

func (g *Game) drawDead() {
	s := g.sc
	lines := Wrap(g.DeathText, 60)
	lines = append(lines, "", fmt.Sprintf("Level %d · %d kills · %d secrets learned · %d quests", g.P.Level, g.Kills, len(g.Secrets), len(g.Quests)), "",
		"Your death has been written into the chronicle.", "", "{{W|[Enter] return to Realmforge}}")
	h := len(lines) + 4
	y0 := (MH - h) / 2
	s.Box(8, y0, 64, h, CRed, "You died")
	for i, l := range lines {
		s.TextB(10, y0+2+i, l, CGray, CBgDark, 71)
	}
}
