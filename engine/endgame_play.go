package engine

import (
	"fmt"
	"strings"
)

// endgame_play.go (prefix eg): the finale at the Heart, the five endings, the war room and the
// reckoning screens.

const egArenaKey = "eg:heart"

// ---- the Heart ----

func (g *Game) egArena() *Zone {
	if zn := g.Zones[egArenaKey]; zn != nil {
		return zn
	}
	st := g.eg()
	d := st.def()
	w := g.W
	ox, oy := st.Origin%w.W, st.Origin/w.W
	zn := newZone(egArenaKey, ox, oy, 1, 1, 0)
	zn.Kind, zn.Name, zn.Biome = "heart", "The Heart of "+strings.TrimPrefix(st.Name, "The "), w.Biome[st.Origin]
	zn.Level = g.zoneLevel(ox, oy, 0) + 3
	r := NewRng(g.Seed + ":eg:arena")
	for i := range zn.T {
		zn.T[i] = d.Floor
	}
	for i := 0; i < 40; i++ {
		x, y := r.Int(8, MW-9), r.Int(2, MH-3)
		if abs(x-66) > 5 || abs(y-12) > 3 {
			zn.Set(x, y, TPillar)
		}
	}
	for i := 0; i < 60; i++ {
		x, y := r.Int(8, MW-9), r.Int(2, MH-3)
		if zn.At(x, y) == d.Floor && (abs(x-66) > 3 || abs(y-12) > 2) {
			zn.Set(x, y, d.Hazard)
		}
	}
	for dy := -1; dy <= 1; dy++ {
		for dx := -3; dx <= 3; dx++ {
			zn.Set(4+dx, 12+dy, d.Floor)
		}
	}
	zn.Set(2, 12, TInscription)
	zn.Notes[12*MW+2] = "The way back. Walk off the edge of the Heart to leave; the fight will wait."
	g.computeLight(zn)
	g.Zones[egArenaKey] = zn
	return zn
}

func (g *Game) egMakeBoss(zn *Zone) *Actor {
	st := g.eg()
	d := st.def()
	r := NewRng(g.Seed + ":eg:boss")
	lvl := clampi(maxi(g.P.Level+4, 16), 16, 40)
	var a *Actor
	if st.Kind == "titan" && st.MonID > 0 {
		a = g.Spawn("horror", lvl, r)
		for i := range g.W.Monsters {
			if m := &g.W.Monsters[i]; m.ID == st.MonID {
				a.SimMon, a.Desc = m.ID, m.Lore
			}
		}
	} else {
		a = g.Spawn(d.BossTpl, lvl, r)
		a.Desc = "The master of " + st.Name + "."
	}
	a.Name, a.Proper, a.Boss = st.BossName, true, true
	a.Set("boss")
	a.Set("egboss")
	ready, _ := g.egReadiness()
	hp := int(float64(350+lvl*22) * (1.3 - 0.5*float64(ready)/100))
	a.MaxHP, a.HP = hp, hp
	a.AVNat += 4
	a.XP, a.Gold = 2000+lvl*100, 400+lvl*30
	a.Faction, a.AI, a.Angry = d.Faction, "hostile", true
	a.Glyph, a.F = d.MobGlyph, CMagenta
	a.Inv = append(a.Inv, g.bossDrop(r, lvl))
	a.X, a.Y = 66, 12
	zn.Actors = append(zn.Actors, a)
	return a
}

func (g *Game) egSpawnMobs(zn *Zone, n int, tag string) {
	st := g.eg()
	d := st.def()
	r := NewRng(fmt.Sprintf("%s:eg:wave:%s:%d", g.Seed, tag, g.Turn))
	lvl := clampi(g.P.Level+1, 8, 40)
	for i := 0; i < n; i++ {
		x, y, ok := zn.FreeSpot(r, func(x, y int) bool { return x > 20 && (abs(x-g.P.X) > 6 || abs(y-g.P.Y) > 5) })
		if !ok {
			return
		}
		a := g.Spawn(Pick(r, d.Mobs), lvl, r)
		a.Name, a.Glyph, a.F = Pick(r, d.MobName), d.MobGlyph, d.MobF
		a.Faction, a.AI, a.Angry = d.Faction, "hostile", true
		a.Set("egmob")
		a.X, a.Y, a.HomeX, a.HomeY = x, y, x, y
		zn.Actors = append(zn.Actors, a)
	}
}

// egSpawnAllies fields the contingents of every allied realm and order as summoned followers.
func (g *Game) egSpawnAllies(zn *Zone, wave int) {
	r := NewRng(fmt.Sprintf("%s:eg:allies:%d", g.Seed, wave))
	total := 0
	for _, al := range g.egAllies() {
		n := al.Size
		if wave > 0 {
			n = (n + 1) / 2
		}
		for i := 0; i < n && total < 20; i++ {
			a := g.soldier(al.Realm, r)
			if a == nil {
				break
			}
			x, y, ok := zn.FreeSpot(r, func(x, y int) bool { return abs(x-g.P.X) <= 5 && abs(y-g.P.Y) <= 4 })
			if !ok {
				break
			}
			a.X, a.Y, a.HomeX, a.HomeY = x, y, x, y
			a.AI, a.Faction, a.Angry, a.F = "follower", "player", false, CGreen
			a.Title = "soldier of " + al.Name
			a.Set("egally")
			qcRescale(a, maxi(g.P.Level-1, a.Level))
			g.comp(a)
			zn.Actors = append(zn.Actors, a)
			total++
		}
	}
	if total > 0 {
		g.Msg(fmt.Sprintf("{{G|%d allied soldiers take the field beside you.}}", total))
	}
}

// egBegin enters (or re-enters) the Heart from anywhere in the origin parasang.
func (g *Game) egBegin() {
	st := g.eg()
	if st.Phase != 1 || st.Finale >= 2 || g.Cur == nil {
		return
	}
	if g.egHere() != st.Origin || g.Cur.Z != 0 || g.Cur.Key == egArenaKey && st.Finale == 1 {
		g.Msg("{{y|You must stand on the threat's own ground (" + g.wsPlace(st.Origin) + ") to march on the Heart.}}")
		return
	}
	if g.nearHostile() {
		g.Msg("{{R|Not with enemies in sight.}}")
		return
	}
	zn := g.egArena()
	first := st.Finale == 0
	if first {
		st.Finale, st.BossPhase = 1, 0
		g.egMakeBoss(zn)
		g.egSpawnMobs(zn, 4, "start")
		g.deed("endgame", fmt.Sprintf("The wanderer %s marched on the Heart of %s.", g.P.Name, st.Name), 0, 0)
	}
	g.enterZone(zn, 4, 12, true)
	g.Mode, g.OnWorld = "main", false
	if first {
		g.egSpawnAllies(zn, 0)
		ready, _ := g.egReadiness()
		g.showPopup(zn.Name, fmt.Sprintf("%s waits at the far side of the Heart. Your readiness is %d%%.\n\n%s", st.BossName, ready, st.def().Weak))
	}
	g.recompute()
}

func egBoss(zn *Zone) *Actor {
	for _, a := range zn.Actors {
		if a.Has("egboss") && a.HP > 0 {
			return a
		}
	}
	return nil
}

// egFinaleTick: phases of the boss fight (turn hook).
func (g *Game) egFinaleTick() {
	st := g.eg()
	if st.Phase != 1 || st.Finale == 0 || g.Cur == nil || g.Cur.Key != egArenaKey {
		return
	}
	zn := g.Cur
	b := egBoss(zn)
	if b == nil {
		return
	}
	d := st.def()
	switch st.Finale {
	case 2:
		b.AddSt("stunned", 5)
		return
	case 1:
		frac := float64(b.HP) / float64(maxi(b.MaxHP, 1))
		switch {
		case frac <= 0.12:
			g.egYield(zn, b)
		case frac <= 0.33 && st.BossPhase < 2:
			st.BossPhase = 2
			b.Speed += 20
			b.AVNat -= 2 * len(st.Keys)
			g.Msg("{{R|" + b.Name + " howls and fights without restraint!}}")
			if len(st.Keys) > 0 {
				g.Msg("{{M|The seals you carry crack its guard.}}")
			}
			g.egSpawnMobs(zn, 6, "p3")
			g.egSurge(b, 0.3)
		case frac <= 0.66 && st.BossPhase < 1:
			st.BossPhase = 1
			g.Msg("{{R|The Heart convulses. " + d.Cause + " fills the air!}}")
			g.egScatterHazard(zn, d, 0.1)
			g.egSpawnMobs(zn, 5, "p2")
			g.egSpawnAllies(zn, 1)
			g.egSurge(b, 0.4)
		}
		if g.Turn%40 == 0 {
			n := 0
			for _, a := range zn.Actors {
				if a.HP > 0 && a.Has("egmob") {
					n++
				}
			}
			if n < 3 {
				g.egSpawnMobs(zn, 1, "trickle")
			}
		}
	}
}

// egSurge: each relic carried burns the boss (never below a floor).
func (g *Game) egSurge(b *Actor, floor float64) {
	held := 0
	for _, r := range g.eg().Relics {
		if qcInvArt(g, r.Art) != nil {
			held++
			if a := g.egArt(r.Art); a != nil {
				g.Msg("{{Y|" + a.Name + " flares against " + b.Name + ".}}")
			}
		}
	}
	dmg := held * b.MaxHP * 6 / 100
	lim := int(float64(b.MaxHP) * floor)
	if b.HP-dmg < lim {
		dmg = maxi(0, b.HP-lim)
	}
	b.HP -= dmg
}

func (g *Game) egScatterHazard(zn *Zone, d *egDef, pct float64) {
	r := NewRng(fmt.Sprintf("%s:eg:haz:%d", g.Seed, g.Turn))
	for i := range zn.T {
		x, y := i%MW, i/MW
		if zn.T[i] == d.Floor && r.Chance(pct) && zn.ActorAt(x, y) == nil && !(x == g.P.X && y == g.P.Y) {
			zn.T[i] = d.Hazard
		}
	}
	g.computeLight(zn)
}

func (g *Game) egYield(zn *Zone, b *Actor) {
	st := g.eg()
	st.Finale = 2
	for _, a := range zn.Actors {
		if a.Has("egmob") && a.HP > 0 {
			a.HP = 0
		}
	}
	b.AddSt("stunned", 5)
	g.Msg("{{M|" + b.Name + " staggers and lowers its guard. It is yours to decide.}}")
	g.Mode, g.Sel = "egend", 0
}

// egBossDown: the boss fell to a killing blow.
func (g *Game) egBossDown(d *Actor) {
	st := g.eg()
	if st.Phase != 1 {
		return
	}
	g.egResolve("destroy", d)
}

// ---- endings ----

func (g *Game) egSealOK() bool {
	st := g.eg()
	if len(g.W.Megas) > 0 {
		return len(st.Keys) >= len(g.W.Megas)
	}
	return true
}

func (g *Game) egAllyOK() bool { return len(g.egAllies()) >= 3 }

func (g *Game) egResolve(end string, boss *Actor) {
	st := g.eg()
	d := st.def()
	day := egDay(g)
	held := g.egHeldCount()
	nal := len(g.egAllies())
	allyNames := []string{}
	for _, a := range g.egAllies() {
		allyNames = append(allyNames, a.Name)
	}
	st.Phase, st.Ending, st.EndDay, st.Finale, st.LastTithe = 3, end, day, 3, day
	zn := g.Zones[egArenaKey]
	if boss == nil && zn != nil {
		boss = egBoss(zn)
	}
	rewardLvl := 1500 + g.P.Level*80
	for _, al := range g.egAllies() {
		f := stateFaction(al.Realm)
		if al.Realm <= 0 {
			continue
		}
		switch end {
		case "destroy":
			g.AddRep(f, 100, " for the victory")
		case "seal":
			g.AddRep(f, 60, " for the sealing")
		case "bargain":
			g.AddRep(f, -250, " for bargaining with the enemy")
		case "ally":
			g.AddRep(f, 150, " for the peace")
		case "abandon":
			g.AddRep(f, -100, " for walking away")
		}
	}
	switch end {
	case "destroy":
		st.Occ, st.occSet = nil, nil
		if boss != nil && boss.HP > 0 {
			boss.HP = 0
			g.kill(boss, g.P)
		}
		it := g.Loot(NewRng(g.Seed+":eg:reward"), 5)
		if it != nil {
			it.Rarity, it.Ident = "unique", true
			it.Name = "Spoils of " + strings.TrimPrefix(st.BossName, "the ")
			g.give(it)
		}
		g.gainXP(rewardLvl)
	case "seal":
		st.Occ, st.occSet = nil, nil
		st.add(st.Origin)
		if boss != nil {
			boss.HP = 0
		}
		g.gainXP(rewardLvl * 4 / 5)
	case "bargain":
		a := d.Attr
		if a >= 0 && a < len(g.P.Attr) {
			g.P.Attr[a] += 3
		}
		g.P.MaxHP += 30
		g.P.HP = g.P.MaxHP
		var keep []int
		for i, p := range st.Occ {
			if i%3 == 0 || p == st.Origin {
				keep = append(keep, p)
			}
		}
		st.Occ, st.occSet = keep, nil
		for _, r := range st.Relics {
			if it := qcInvArt(g, r.Art); it != nil {
				g.removeInv(it)
				break
			}
		}
		if boss != nil {
			boss.HP = 0
		}
		g.gainXP(rewardLvl / 2)
	case "ally":
		st.Occ, st.occSet = nil, nil
		if boss != nil {
			boss.AI, boss.Faction, boss.Angry, boss.Boss, boss.F = "follower", "player", false, false, CGreen
			boss.Flags["egboss"] = false
			boss.Status = nil
			boss.HP = boss.MaxHP
			g.comp(boss)
		}
		g.gainXP(rewardLvl)
	case "abandon":
		if zn != nil && boss != nil && g.Cur != zn {
			boss.HP = 0
		}
	}
	for _, a := range func() []*Actor {
		if zn == nil {
			return nil
		}
		return zn.Actors
	}() {
		if a.Has("egmob") {
			a.HP = 0
		}
	}
	st.logf(fmt.Sprintf("Day %d: %s.", day, egEndNames[end]))
	deedText := map[string]string{
		"destroy": "destroyed", "seal": "sealed away", "bargain": "bargained with", "ally": "made peace with and befriended", "abandon": "abandoned the world to",
	}[end]
	g.deed("endgame", fmt.Sprintf("The wanderer %s %s %s.", g.P.Name, deedText, st.Name), st.MonID, 0)
	g.Mode = "main"
	g.showPopup("Epilogue: "+egEndNames[end], g.egEpilogue(end, day, held, nal, allyNames))
}

func (g *Game) egEpilogue(end string, day, held, nal int, allies []string) string {
	st := g.eg()
	d := st.def()
	var b strings.Builder
	b.WriteString(egEndText[end])
	b.WriteString("\n\n" + d.Dread)
	fmt.Fprintf(&b, "\n\nThe crisis lasted %d days. %d parasangs were won back; %d towns fell.", day-st.RiseDay, st.Freed, len(st.Fallen))
	if len(allies) > 0 {
		b.WriteString(" Beside you stood " + strings.Join(allies, ", ") + ".")
	}
	if held > 0 {
		fmt.Fprintf(&b, " You carried %d of the old relics.", held)
	}
	b.WriteString("\n\nThe world goes on. Its markets, roads and people still feel what happened, for a while.")
	return b.String()
}

// ---- screens ----

type egOpt struct {
	Text string
	Do   func()
	On   bool
}

func (g *Game) egWarOpts() []egOpt {
	st := g.eg()
	var out []egOpt
	if st.Phase == 1 {
		switch st.Finale {
		case 0, 1:
			ok := g.egHere() == st.Origin && g.Cur != nil && g.Cur.Z == 0
			txt := "March on the Heart"
			if st.Finale == 1 {
				txt = "Return to the Heart"
			}
			if !ok {
				txt += " {{K|(stand in " + g.wsPlace(st.Origin) + ")}}"
			}
			out = append(out, egOpt{txt, func() { g.Mode = "main"; g.egBegin() }, ok})
		case 2:
			out = append(out, egOpt{"Decide the fate of " + st.BossName, func() { g.Mode, g.Sel = "egend", 0 }, true})
		}
		lbl := "Abandon the world to its fate"
		if g.Tab == 1 {
			lbl = "{{R|Press Enter again: abandon the world}}"
		}
		out = append(out, egOpt{lbl, func() {
			if g.Tab != 1 {
				g.Tab = 1
				return
			}
			g.egResolve("abandon", nil)
		}, true})
	}
	return out
}

func egWarKeys(g *Game, k string) {
	opts := g.egWarOpts()
	switch k {
	case "Escape", "E", "q":
		g.Mode = "main"
	case "ArrowDown", "j", "N2":
		g.Sel = mini(g.Sel+1, len(opts)-1)
		g.Tab = 0
	case "ArrowUp", "k", "N8":
		g.Sel = maxi(g.Sel-1, 0)
		g.Tab = 0
	case "Enter", " ":
		if g.Sel >= 0 && g.Sel < len(opts) {
			if o := opts[g.Sel]; o.On {
				o.Do()
			} else {
				g.Msg("{{y|Not yet.}}")
			}
		}
	}
}

func (g *Game) egDrawWar() {
	st := g.eg()
	x, y, w, h := g.panel("War room")
	var rows []string
	add := func(str string) { rows = append(rows, Wrap(str, w)...) }
	if st.Phase == 0 {
		add("{{W|The world is quiet.}} {{K|For now. When the wanderer is seasoned and the great errand under way, something stirs.}}")
		rows = append(rows, "")
		for _, l := range g.egThresholdLines() {
			add("  " + l)
		}
		g.egDrawRows(rows, nil, x, y, w, h)
		return
	}
	d := st.def()
	here := g.egHere()
	if st.Phase == 1 {
		add("{{R|" + st.Name + "}}  {{K|day " + fmt.Sprint(egDay(g)-st.RiseDay) + "}}")
		add(fmt.Sprintf("Heart: {{Y|%s}} {{K|(%s)}}", g.wsPlace(st.Origin), g.wsDirDist(here, st.Origin)))
		add(fmt.Sprintf("Held: {{R|%d}} parasangs ({{y|%d%%}} of the land)   Fallen towns: %d   Won back: {{G|%d}}", len(st.Occ), 100*len(st.Occ)/maxi(1, g.egLandCount()), len(st.Fallen), st.Freed))
		ref := 0
		for _, n := range st.Ref {
			ref += n
		}
		add(fmt.Sprintf("Refugees on the roads: %d   Marking its dread: %s", ref, d.Cause))
	} else {
		add("{{c|" + st.Name + ": " + egEndNames[st.Ending] + "}}  {{K|" + fmt.Sprint(egDay(g)-st.EndDay) + " days ago}}")
		add(fmt.Sprintf("Lingering: %d%%   Occupied remnant: %d parasangs", int(g.egLinger()*100), len(st.Occ)))
	}
	rows = append(rows, "", "{{W|Allies}}")
	if al := g.egAllies(); len(al) == 0 {
		add("  {{K|None. Ask an elder or noble to call their realm; orders you belong to join by themselves.}}")
	} else {
		for _, a := range al {
			add(fmt.Sprintf("  {{G|%s}} {{K|(contingent of %d)}}", a.Name, a.Size))
		}
	}
	rows = append(rows, "", "{{W|Relics}}")
	held := g.egHeldCount()
	for _, r := range st.Relics {
		a := g.egArt(r.Art)
		if a == nil {
			continue
		}
		if st.Held[r.Art] {
			add("  {{G|[x] " + a.Name + "}}")
		} else if b := g.W.burgByID[r.Burg]; b != nil {
			add(fmt.Sprintf("  {{y|[ ] %s}} {{K|lies with the dead of %s (%s)}}", a.Name, b.Name, g.wsDirDist(here, b.P)))
		}
	}
	if len(st.Relics) == 0 {
		add("  {{K|No relics are known to lie in the ruins.}}")
	}
	rows = append(rows, "", "{{W|Keys of the great structures}}")
	for _, m := range g.W.Megas {
		kind := m.Kind
		if st.Keys[kind] {
			add("  {{G|[x] Key of " + m.Name + "}}")
		} else {
			add(fmt.Sprintf("  {{y|[ ] %s}} {{K|slay the master of its deepest hall (%s)}}", m.Name, g.wsDirDist(here, m.P)))
		}
	}
	if st.Phase == 1 {
		rd, parts := g.egReadiness()
		rows = append(rows, "", fmt.Sprintf("{{W|Finale readiness: %d%%}} {{K|(%s)}}", rd, strings.Join(parts, ", ")))
		add("{{K|" + d.Weak + "}}")
		_ = held
	}
	g.egDrawRows(rows, g.egWarOpts(), x, y, w, h)
}

func (g *Game) egDrawRows(rows []string, opts []egOpt, x, y, w, h int) {
	s := g.sc
	maxRows := h - 2 - len(opts)
	if len(opts) > 0 {
		maxRows--
	}
	for i := 0; i < len(rows) && i < maxRows; i++ {
		s.TextB(x, y+i, rows[i], CGray, CBgDark, x+w)
	}
	oy := y + h - 2 - len(opts)
	for i, o := range opts {
		bg := CBgDark
		if i == g.Sel {
			bg = CBgSel
		}
		s.Fill(x, oy+i, w, 1, bg)
		col := "W"
		if !o.On {
			col = "K"
		}
		s.TextB(x, oy+i, "{{"+col+"|> }}"+o.Text, CGray, bg, x+w)
	}
	s.TextB(x, y+h-1, "{{K|Up/Down select, Enter act, Esc/E close}}", CGray, CBgDark, x+w)
}

func (g *Game) egEndOpts() []egOpt {
	st := g.eg()
	d := st.def()
	return []egOpt{
		{"{{R|Destroy it.}} {{K|Strike the final blow.}}", func() { g.egResolve("destroy", nil) }, true},
		{"{{C|Seal it.}} {{K|Needs the keys of every great structure.}}", func() { g.egResolve("seal", nil) }, g.egSealOK()},
		{"{{M|Bargain.}} {{K|A gift of power now; the realms will not forgive; a tithe each season.}}", func() { g.egResolve("bargain", nil) }, true},
		{"{{G|Unite and turn it.}} {{K|Needs three allied realms or orders.}}", func() { g.egResolve("ally", nil) }, g.egAllyOK()},
		{"{{y|Walk away.}} {{K|" + shorten(d.Dread, 40) + "}}", func() { g.egResolve("abandon", nil) }, true},
	}
}

func egEndKeys(g *Game, k string) {
	opts := g.egEndOpts()
	switch k {
	case "Escape", "q":
		g.Mode = "main"
	case "ArrowDown", "j", "N2":
		g.Sel = mini(g.Sel+1, len(opts)-1)
	case "ArrowUp", "k", "N8":
		g.Sel = maxi(g.Sel-1, 0)
	case "Enter", " ":
		if g.Sel >= 0 && g.Sel < len(opts) {
			if o := opts[g.Sel]; o.On {
				o.Do()
			} else {
				g.Msg("{{y|You lack what that ending needs.}}")
			}
		}
	}
}

func (g *Game) egDrawEnd() {
	st := g.eg()
	x, y, w, h := g.panel("The reckoning")
	rows := Wrap(fmt.Sprintf("{{M|%s}} lies at your mercy, beaten and listening. Whatever you choose, the world will remember it. Esc decides later.", st.BossName), w)
	g.egDrawRows(rows, g.egEndOpts(), x, y, w, h)
}
