package engine

import (
	"fmt"
	"strings"
)

// Tactical combat additions (prefix tc): called shots to specific body parts,
// positioning modifiers (flanking, high ground, cover, weather/glare),
// special ammunition and tactical throwables (nets, bolas, caltrops, smoke
// bombs, flashbangs), and squad morale (rout on leader death or heavy losses,
// boss/captain rally cry).

var tcParts = []string{"head", "arm", "leg", "wing", "eye"}

var tcPartDesc = map[string]string{
	"head": "-3 to hit; stuns and dazes on hit",
	"arm":  "-3 to hit; disarms or weakens on hit",
	"leg":  "-3 to hit; cripples and slows on hit",
	"wing": "-3 to hit; grounds flying foes on hit",
	"eye":  "-3 to hit; blinds on hit",
}

type tcState struct {
	ZoneKills map[string]map[string]int `json:"zk,omitempty"`
	Caltrops  map[string]int            `json:"cal,omitempty"`
	Smoke     map[string]map[int]int    `json:"smk,omitempty"`
}

func (g *Game) tc() *tcState {
	st := g.ext("tc", func() any { return &tcState{} }).(*tcState)
	if st.ZoneKills == nil {
		st.ZoneKills = map[string]map[string]int{}
	}
	if st.Caltrops == nil {
		st.Caltrops = map[string]int{}
	}
	if st.Smoke == nil {
		st.Smoke = map[string]map[int]int{}
	}
	return st
}

func init() {
	unLocalKeys = append(unLocalKeys, "tc")
	for _, d := range []*skStatusDef{
		{ID: "weakened", Name: "Weakened", Glyph: 'w', Hit: -3},
		{ID: "crippled", Name: "Crippled", Glyph: 'c', DV: -3, Speed: -40},
		{ID: "grounded", Name: "Grounded", Glyph: 'g', DV: -3, Speed: -30},
		{ID: "blind", Name: "Blind", Glyph: 'B', DV: -4, Hit: -6},
		{ID: "feared", Name: "Feared", Glyph: '!', DV: -2, Hit: -2, Tick: skTerrifiedTick},
	} {
		skAddStatus(d)
	}
	mainKeys["V"] = func(g *Game) { g.tcOpenCalledShot() }
	modeKeys["calledshot"] = tcCalledShotKey
	modeDraws["calledshot"] = tcDrawCalledShot
	killHooks = append(killHooks, func(g *Game, d, src *Actor) { g.tcMoraleKill(d, src) })
	turnHooks = append(turnHooks, tcMoraleTurn)
}

// ---- called shots ---------------------------------------------------------

func (g *Game) tcOpenCalledShot() {
	g.Mode = "calledshot"
	g.Sel = 0
	if t := g.nearestTarget(-1); t != nil {
		g.CX, g.CY = t.X, t.Y
	} else if g.P != nil {
		g.CX, g.CY = g.P.X, g.P.Y
	}
}

func (g *Game) tcDefaultTarget() *Actor {
	if g.Cur == nil || g.P == nil {
		return nil
	}
	if a := g.Cur.ActorAt(g.CX, g.CY); a != nil && a != g.P && a.HP > 0 {
		return a
	}
	if t := g.nearestTarget(-1); t != nil {
		return t
	}
	for _, a := range g.Cur.Actors {
		if a != g.P && a.HP > 0 && dist(a, g.P) <= 1 {
			return a
		}
	}
	return nil
}

func tcCalledShotKey(g *Game, k string) {
	if i := letterIndex(k); i >= 0 && i < len(tcParts) {
		g.Sel = i
		tcRunCalledShotUI(g)
		return
	}
	if len(k) == 1 && k[0] >= '1' && k[0] <= '5' {
		g.Sel = int(k[0] - '1')
		tcRunCalledShotUI(g)
		return
	}
	switch k {
	case "ArrowUp", "k":
		g.Sel = maxi(0, g.Sel-1)
	case "ArrowDown", "j":
		g.Sel = mini(len(tcParts)-1, g.Sel+1)
	case "Tab", "+":
		var idx = -1
		cands := 0
		for _, a := range g.Cur.Actors {
			if a.HP > 0 && g.visible(a.X, a.Y) && g.Hostile(a) {
				if a.X == g.CX && a.Y == g.CY {
					idx = cands
				}
				cands++
			}
		}
		if t := g.nearestTarget(idx); t != nil {
			g.CX, g.CY = t.X, t.Y
		}
	case "Enter", " ":
		tcRunCalledShotUI(g)
	case "Escape", "V", "v":
		g.Mode = "main"
	}
}

func tcRunCalledShotUI(g *Game) {
	part := tcParts[clampi(g.Sel, 0, len(tcParts)-1)]
	tgt := g.tcDefaultTarget()
	g.Mode = "main"
	if tgt == nil {
		g.Msg("No target in reach for a called shot.")
		return
	}
	g.tcCalledShot(tgt, part)
	g.spend(1000)
}

func tcDrawCalledShot(g *Game) {
	if g.sc == nil {
		return
	}
	s := g.sc
	h := len(tcParts) + 5
	y0 := maxi(1, (MH-h)/2)
	s.Box(8, y0, SW-16, h, CYellow, "Called Shot (-3 to hit)")
	tgtName := "none"
	if t := g.tcDefaultTarget(); t != nil {
		tgtName = g.the(t)
	}
	s.TextB(10, y0+1, fmt.Sprintf("Target: {{Y|%s}} (Tab to cycle, Enter/a-e to strike)", tgtName), CGray, CBgDark, SW-10)
	y := y0 + 3
	for i, p := range tcParts {
		bg := CBgDark
		if i == g.Sel {
			bg = CBgSel
		}
		s.Fill(10, y, SW-20, 1, bg)
		s.TextB(10, y, fmt.Sprintf("{{W|%c)}} %-6s {{K|%s}}", 'a'+i, titleCase(p), tcPartDesc[p]), CGray, bg, SW-10)
		y++
	}
}

// tcCalledShot executes an aimed strike at a specific body part ("head", "arm", "leg", "wing", "eye")
// with a -3 accuracy penalty and part-specific status/disarm effects on hit.
func (g *Game) tcCalledShot(target *Actor, part string) bool {
	if g == nil || g.P == nil || target == nil || target.HP <= 0 {
		return false
	}
	part = strings.ToLower(strings.TrimSpace(part))
	switch part {
	case "head", "arm", "leg", "wing", "eye":
	default:
		return false
	}
	a := g.P
	d := dist(a, target)
	var w *Item
	ranged := false
	if d <= 1 {
		w = a.Weapons()[0]
	} else if rw := wpReachOf(a); rw != nil && d <= rw.Extra["reach"] {
		w = rw
	} else if mw := a.Missile(); mw != nil {
		w = mw
		ranged = true
		_ = g.wpUseAmmo(mw)
	} else {
		w = a.Weapons()[0]
	}

	sk := g.skillLv(a, w.Skill)
	if ranged {
		sk = g.skillLv(a, wpShotSkill(w))
	}
	roll := g.R.Int(1, 20)
	hit := roll + a.M(AGI) + sk + skStatusToHit(a) + thToHitBonus(g) - 3
	if ranged {
		hit += wpShotHit(a, w) - d/4 + g.tcWeatherRangedMod() - g.tcCoverBonus(target, a)
	} else {
		hit += wpHitMod(a, target, w) + g.tcFlankBonus(a, target) + g.tcHighGroundBonus(a, target)
	}
	if a.St("confused") > 0 {
		hit -= 4
	}
	crit := roll == 20 || g.R.Int(1, 100) <= thCritBonus(g)
	if !crit && (roll == 1 || hit < target.DV()) {
		g.Msg(fmt.Sprintf("Your called shot at %s's %s misses.", g.the(target), part))
		return false
	}

	pv := w.PV + mini(a.M(STR), w.MaxStr) + skStatusPV(target)
	if ranged {
		pv = w.PV + wpShotPV(a, target, w)
	}
	if crit {
		pv++
	}
	pens := maxi(1, g.penetrate(pv, target.AV(), crit))
	dmg := 0
	for i := 0; i < pens; i++ {
		dmg += g.R.Dice(maxi(w.DN, 1), maxi(w.DS, 1)) + w.DB
	}
	if part == "head" || part == "eye" {
		dmg += 2
	}
	tcApplyPartEffect(g, target, part)
	g.hurt(target, maxi(1, dmg), a, w.Name)
	return true
}

func tcApplyPartEffect(g *Game, target *Actor, part string) {
	switch part {
	case "head":
		target.AddSt("stunned", 3)
		target.AddSt("dazed", 4)
		g.Msg(fmt.Sprintf("{{Y|You strike %s in the head! %s is stunned!}}", g.the(target), g.The(target)))
	case "arm":
		disarmed := g.wpDisarm(target)
		target.AddSt("weakened", 6)
		if !disarmed {
			g.Msg(fmt.Sprintf("{{y|You strike %s in the arm! %s is weakened!}}", g.the(target), g.The(target)))
		}
	case "leg":
		target.AddSt("crippled", 6)
		target.AddSt("slowed", 6)
		target.AddSt("hobbled", 6)
		g.Msg(fmt.Sprintf("{{y|You strike %s in the leg! %s is crippled!}}", g.the(target), g.The(target)))
	case "wing":
		target.AddSt("grounded", 8)
		target.AddSt("slowed", 4)
		if target.Flags != nil {
			delete(target.Flags, "fly")
			delete(target.Flags, "flying")
		}
		g.Msg(fmt.Sprintf("{{y|You clip %s's wing! %s is grounded!}}", g.the(target), g.The(target)))
	case "eye":
		target.AddSt("blind", 6)
		target.AddSt("arblind", 6)
		g.Msg(fmt.Sprintf("{{Y|You strike %s in the eye! %s is blinded!}}", g.the(target), g.The(target)))
	}
}

// ---- tactical positioning modifiers ---------------------------------------

func tcIsCoAttacker(g *Game, att, def, o *Actor) bool {
	if o == nil || o == att || o == def || o.HP <= 0 {
		return false
	}
	if dist(o, def) > 1 {
		return false
	}
	if att == g.P || att.AI == "follower" {
		return o == g.P || o.AI == "follower" || g.Enemies(o, def)
	}
	if def == g.P {
		return g.Hostile(o) || o.Faction == att.Faction
	}
	return o.Faction == att.Faction || g.Enemies(o, def)
}

// tcFlankBonus returns +2 when an ally is on the opposite side of def, plus +1 per additional adjacent attacker.
func (g *Game) tcFlankBonus(att, def *Actor) int {
	if g == nil || g.Cur == nil || att == nil || def == nil || dist(att, def) > 1 {
		return 0
	}
	dx, dy := def.X-att.X, def.Y-att.Y
	var allies []*Actor
	if g.P != nil && tcIsCoAttacker(g, att, def, g.P) {
		allies = append(allies, g.P)
	}
	for _, o := range g.Cur.Actors {
		if tcIsCoAttacker(g, att, def, o) {
			allies = append(allies, o)
		}
	}
	if len(allies) == 0 {
		return 0
	}
	opposite := false
	for _, o := range allies {
		ox, oy := o.X-def.X, o.Y-def.Y
		if (ox == dx && oy == dy) || (dx*ox+dy*oy > 0) {
			opposite = true
			break
		}
	}
	if opposite {
		return 2 + (len(allies) - 1)
	}
	return len(allies)
}

func tcTileElev(zn *Zone, x, y int) int {
	if zn == nil || !zn.In(x, y) {
		return 0
	}
	switch zn.At(x, y) {
	case TUp, TCliff:
		return 2
	case TRubble, TThrone, TAltar, TStatue:
		return 1
	case TDown, TShallow, TDeep, TTar, TQuicksand:
		return -1
	}
	return 0
}

// tcHighGroundBonus returns +2 when att attacks def from higher elevation, stairs, or hill terrain.
func (g *Game) tcHighGroundBonus(att, def *Actor) int {
	if g == nil || g.Cur == nil || att == nil || def == nil {
		return 0
	}
	if tcTileElev(g.Cur, att.X, att.Y) > tcTileElev(g.Cur, def.X, def.Y) {
		return 2
	}
	if g.W != nil {
		altA, _ := g.tgElev(g.Cur, att.X, att.Y)
		altD, _ := g.tgElev(g.Cur, def.X, def.Y)
		if altA-altD >= 1.5 {
			return 2
		}
	}
	return 0
}

func tcCoverTile(t Terrain) bool {
	switch t {
	case TWall, TRock, TPillar, TTree, TDeadTree, TMushroom, TTrunk, TStatue, TCrate, TSarcophagus, TIceWall, TCrystal:
		return true
	}
	return false
}

// tcCoverBonus returns +2 DV when def is adjacent to a wall, pillar, or tree against ranged attacks (or in smoke).
func (g *Game) tcCoverBonus(def, att *Actor) int {
	if g == nil || g.Cur == nil || def == nil {
		return 0
	}
	if att != nil && dist(att, def) <= 1 {
		return 0
	}
	if g.tcInSmoke(def.X, def.Y) {
		return 2
	}
	for _, d := range dirs8 {
		nx, ny := def.X+d[0], def.Y+d[1]
		if g.Cur.In(nx, ny) && tcCoverTile(g.Cur.At(nx, ny)) {
			return 2
		}
	}
	return 0
}

// tcWeatherRangedMod computes weather and time-of-day ranged to-hit modifiers (negative on penalties).
func (g *Game) tcWeatherRangedMod() int {
	if g == nil || g.Cur == nil {
		return 0
	}
	mod := 0
	var wk *WeatherKind
	var temp int
	if g.wxForce != nil && g.wxForce.Kind != nil {
		wk = g.wxForce.Kind
		temp = g.wxForce.Temp
	} else if w := g.activeWeather(); w != nil && w.Kind != nil {
		wk = w.Kind
		temp = w.Temp
	}
	if wk != nil {
		if wk.Wet > 0 || wk.Overlay == "rain" || wk.Overlay == "storm" {
			mod -= maxi(1, wk.Wet)
			if g.P != nil {
				if m := g.P.Missile(); m != nil && (m.Elem == "fire" || m.Skill == "firearm") {
					mod -= 2
				}
			}
		}
		if wk.Overlay == "fog" || wk.ID == "fog" || wk.ID == "smog" || wk.ID == "miasma" || wk.ID == "steamfog" {
			mod -= 2
		}
		if wk.ID == "blizzard" || wk.ID == "sandstorm" || wk.ID == "radstorm" || wk.ID == "snow" {
			mod -= maxi(1, wk.Missile)
		}
		if wk.ID == "heat" || temp >= 35 {
			mod -= 1
		}
	}
	if outdoors(g.Cur) {
		if g.W != nil && g.W.In(g.WX, g.WY) {
			_, str := g.shWind(g.W.Idx(g.WX, g.WY))
			if str >= 0.65 {
				mod -= 2
			} else if str >= 0.45 {
				mod -= 1
			}
		}
		if !g.daylight() {
			if g.P == nil || (g.P.Pow("darkvision") == 0 && g.P.St("alnight") == 0) {
				mod -= 2
			}
		} else {
			h := (g.Turn/60 + 8) % 24
			if h >= 11 && h <= 14 && g.P != nil && g.Cur.In(g.P.X, g.P.Y) {
				t := g.Cur.At(g.P.X, g.P.Y)
				glareTile := t == TSand || t == TSnow || t == TIce || t == TSalt
				glareBiome := false
				if g.W != nil && g.W.In(g.WX, g.WY) {
					b := g.W.biomeAt(g.W.Idx(g.WX, g.WY))
					glareBiome = b == 1 || b == 11
				}
				if glareTile || glareBiome {
					mod -= 1
				}
			}
		}
	}
	return mod
}

// ---- special ammo & tactical throwables -----------------------------------

// tcMakeSpecialAmmo builds a stack of specialized ammunition:
// "bodkin", "broadhead", "fire", "frost", "poison", "silver", or "emp".
func (g *Game) tcMakeSpecialAmmo(kind string, n int) *Item {
	if n <= 0 {
		n = 1
	}
	kind = strings.ToLower(strings.TrimSpace(kind))
	it := &Item{
		ID:     g.newID(),
		Kind:   "ammo",
		Glyph:  '¡',
		F:      CGray,
		Count:  n,
		Effect: kind,
		Value:  3,
		Extra:  map[string]int{"tc_ammo": 1},
	}
	switch kind {
	case "bodkin":
		it.Name = "bodkin arrows"
		it.PV = 3
		it.Extra["ap"] = 3
		it.Lore = "Needle-tipped shafts forged to punch through heavy plate."
	case "broadhead":
		it.Name = "broadhead arrows"
		it.F = CRed
		it.DN, it.DS = 1, 4
		it.Extra["bleed"] = 1
		it.Lore = "Wide barbed heads that open ragged, bleeding wounds."
	case "fire":
		it.Name = "incendiary arrows"
		it.F = COrange
		it.Elem = "fire"
		it.Value = 4
		it.Lore = "Alchemically tipped shafts that ignite on impact."
	case "frost":
		it.Name = "frost-forged arrows"
		it.F = CCyan
		it.Elem = "frost"
		it.Value = 4
		it.Lore = "Rime-chilled arrowheads that numb and freeze flesh."
	case "poison":
		it.Name = "venom-tipped arrows"
		it.F = CGreen
		it.Elem = "poison"
		it.Value = 4
		it.Lore = "Coated with potent neurotoxic venom."
	case "silver":
		it.Name = "silver-tipped arrows"
		it.F = CWhite
		it.Value = 6
		it.Extra["silver"] = 1
		it.Lore = "Pure silver heads that sear the undead and otherworldly horrors."
	case "emp":
		it.Name = "EMP pulse rounds"
		it.F = CCyan
		it.Elem = "shock"
		it.Value = 6
		it.Extra["emp"] = 1
		it.Lore = "Electromagnetic pulse rounds that overload constructs and turrets."
	default:
		it.Name = kind + " arrows"
	}
	return it
}

// tcApplyAmmoHit applies the on-hit effect of special ammunition to d.
func (g *Game) tcApplyAmmoHit(ammo *Item, d *Actor) {
	if g == nil || ammo == nil || d == nil || d.HP <= 0 {
		return
	}
	kind := ammo.Effect
	if kind == "" {
		kind = ammo.Elem
	}
	switch kind {
	case "bodkin":
		d.AddSt("cracked", 6)
		g.hurt(d, maxi(2, d.AV()), g.P, ammo.Name)
	case "broadhead":
		if !d.Has("undead") && d.Tpl != "golem" {
			d.AddSt("bleeding", 8)
		}
		g.hurt(d, 2, g.P, ammo.Name)
	case "fire":
		d.AddSt("burning", 5)
		g.hurt(d, g.R.Int(2, 5), g.P, "fire")
	case "frost":
		d.AddSt("frozen", 3)
		d.AddSt("slowed", 5)
		g.hurt(d, g.R.Int(1, 4), g.P, "frost")
	case "poison":
		if !d.Has("undead") && d.Tpl != "golem" {
			d.AddSt("poisoned", 8)
		}
		g.hurt(d, 2, g.P, "poison")
	case "silver":
		bonus := 2
		if d.Has("undead") || d.Faction == "undead" || d.Faction == "spirits" || d.Faction == "eldritch" {
			bonus = g.R.Dice(2, 6) + 4
			d.AddSt("shaken", 5)
		}
		g.hurt(d, bonus, g.P, "silver")
	case "emp":
		if d.Faction == "constructs" || d.Tpl == "golem" || d.Tpl == "turret" || d.Tpl == "crystalgolem" {
			d.AddSt("stunned", 5)
			g.hurt(d, g.R.Dice(2, 8)+4, g.P, "EMP")
		} else {
			d.AddSt("dazed", 2)
			g.hurt(d, g.R.Int(1, 4), g.P, "EMP")
		}
	}
}

// tcMakeTacticalItem creates a stack of tactical throwables:
// "net", "bolas", "caltrops", "smokebomb", or "flashbang".
func (g *Game) tcMakeTacticalItem(kind string, n int) *Item {
	if n <= 0 {
		n = 1
	}
	kind = strings.ToLower(strings.TrimSpace(kind))
	it := &Item{
		ID:     g.newID(),
		Kind:   "grenade",
		Effect: kind,
		Count:  n,
		Range:  6,
		Weight: 2,
		Value:  15,
		Extra:  map[string]int{"tc_tac": 1},
	}
	switch kind {
	case "net":
		it.Name = "weighted net"
		it.Glyph = '#'
		it.F = CBrown
		it.Weight = 3
		it.Value = 18
		it.Lore = "A corded throwing net weighted with lead sinkers."
	case "bolas":
		it.Name = "hunting bolas"
		it.Glyph = '¤'
		it.F = CBrown
		it.Value = 15
		it.Lore = "Weighted cords that wrap around a target's legs and bring it down."
	case "caltrops":
		it.Name = "iron caltrops"
		it.Glyph = '^'
		it.F = CGray
		it.Value = 12
		it.Lore = "Four-pointed iron spikes that hobble and bleed anything stepping on them."
	case "smokebomb":
		it.Name = "smoke bomb"
		it.Glyph = 'o'
		it.F = CDarkGray
		it.Weight = 1
		it.Value = 20
		it.Lore = "Releases a dense cloud of grey smoke that blocks sight."
	case "flashbang":
		it.Name = "flashbang charge"
		it.Glyph = '*'
		it.F = CYellow
		it.Weight = 1
		it.Value = 25
		it.Lore = "Detonates with a blinding flash and concussive bang."
	default:
		it.Name = kind
		it.Glyph = '*'
		it.F = CGray
	}
	return it
}

func (g *Game) tcActorAt(x, y int) *Actor {
	if g.Cur == nil {
		return nil
	}
	if a := g.Cur.ActorAt(x, y); a != nil {
		return a
	}
	if g.P != nil && g.P.X == x && g.P.Y == y {
		return g.P
	}
	return nil
}

func (g *Game) tcInSmoke(x, y int) bool {
	if g == nil || g.Cur == nil || !g.Cur.In(x, y) {
		return false
	}
	m := g.tc().Smoke[g.Cur.Key]
	return m != nil && m[y*MW+x] > 0
}

// tcUseTacticalItem deploys or throws a tactical item ("net", "bolas", "caltrops", "smokebomb", "flashbang") at (tx, ty).
func (g *Game) tcUseTacticalItem(it *Item, tx, ty int) bool {
	if g == nil || g.Cur == nil || it == nil || !g.Cur.In(tx, ty) {
		return false
	}
	kind := strings.ToLower(strings.TrimSpace(it.Effect))
	switch kind {
	case "net":
		if d := g.tcActorAt(tx, ty); d != nil {
			d.AddSt("entangled", 6)
			d.AddSt("slowed", 6)
			d.AddSt("grounded", 6)
			if d.Flags != nil {
				delete(d.Flags, "fly")
				delete(d.Flags, "flying")
			}
			g.Msg(fmt.Sprintf("{{y|%s is entangled in the net!}}", g.The(d)))
		} else {
			g.Msg("The net lands on empty ground.")
		}
	case "bolas":
		if d := g.tcActorAt(tx, ty); d != nil {
			d.AddSt("entangled", 4)
			d.AddSt("prone", 3)
			d.AddSt("hobbled", 5)
			d.AddSt("crippled", 5)
			d.AddSt("slowed", 5)
			g.hurt(d, g.R.Int(1, 3), g.P, it.Name)
			g.Msg(fmt.Sprintf("{{y|The bolas wrap around %s's legs!}}", g.the(d)))
		} else {
			g.Msg("The bolas clatter across the ground.")
		}
	case "caltrops":
		if g.Cur.Traps == nil {
			g.Cur.Traps = map[int]uint8{}
		}
		idx := ty*MW + tx
		g.Cur.Traps[idx] = 1
		g.tc().Caltrops[fmt.Sprintf("%s:%d", g.Cur.Key, idx)] = 8
		g.Msg("You scatter iron caltrops across the ground.")
		if d := g.Cur.ActorAt(tx, ty); d != nil {
			g.tcTriggerCaltrops(d)
		}
	case "smokebomb":
		sm := g.tc().Smoke[g.Cur.Key]
		if sm == nil {
			sm = map[int]int{}
			g.tc().Smoke[g.Cur.Key] = sm
		}
		for dy := -1; dy <= 1; dy++ {
			for dx := -1; dx <= 1; dx++ {
				nx, ny := tx+dx, ty+dy
				if !g.Cur.In(nx, ny) {
					continue
				}
				sm[ny*MW+nx] = 8
				if len(g.vis) == MW*MH {
					g.vis[ny*MW+nx] = false
				}
				if d := g.Cur.ActorAt(nx, ny); d != nil {
					d.AddSt("blind", 3)
					d.AddSt("disoriented", 3)
				}
			}
		}
		g.Msg("{{K|Thick smoke billows out, obscuring sight!}}")
	case "flashbang":
		for dy := -2; dy <= 2; dy++ {
			for dx := -2; dx <= 2; dx++ {
				nx, ny := tx+dx, ty+dy
				if !g.Cur.In(nx, ny) {
					continue
				}
				if d := g.Cur.ActorAt(nx, ny); d != nil {
					d.AddSt("stunned", 3)
					d.AddSt("blind", 4)
					d.AddSt("arblind", 4)
					d.AddSt("confused", 4)
					d.AddSt("dazed", 4)
				}
			}
		}
		g.Msg("{{Y|The flashbang detonates with a blinding flash!}}")
	default:
		return false
	}
	if it.Count > 1 {
		it.Count--
	} else {
		it.Count = 0
		g.removeInv(it)
	}
	return true
}

func (g *Game) tcTriggerCaltrops(d *Actor) {
	if g == nil || g.Cur == nil || d == nil || d.HP <= 0 || d.Has("flying") || d.Has("fly") {
		return
	}
	idx := d.Y*MW + d.X
	key := fmt.Sprintf("%s:%d", g.Cur.Key, idx)
	if g.tc().Caltrops[key] <= 0 && (g.Cur.Traps == nil || g.Cur.Traps[idx] == 0) {
		return
	}
	delete(g.tc().Caltrops, key)
	if g.Cur.Traps != nil {
		delete(g.Cur.Traps, idx)
	}
	d.AddSt("bleeding", 5)
	d.AddSt("hobbled", 5)
	d.AddSt("slowed", 5)
	d.AddSt("crippled", 4)
	g.Msg(fmt.Sprintf("{{r|%s steps on caltrops and is hobbled!}}", g.The(d)))
	g.hurt(d, 2, g.P, "caltrops")
}

// ---- morale system --------------------------------------------------------

func tcIsLeader(a *Actor) bool {
	if a == nil {
		return false
	}
	return a.Boss || a.Has("boss") || a.Role == "captain" || a.Role == "chief" || a.Title == "captain" || a.Title == "chief"
}

func tcCanRout(a *Actor) bool {
	if a == nil || a.HP <= 0 || tcIsLeader(a) || a.AI == "follower" {
		return false
	}
	if a.Has("undead") || a.Faction == "undead" || a.Faction == "constructs" || a.Tpl == "golem" || a.Tpl == "turret" || a.Tpl == "crystalgolem" {
		return false
	}
	return true
}

// tcMoraleKill runs when d is killed by src: if d was a leader or >=50% of same-faction hostiles
// in sight have fallen, remaining non-boss, non-undead, non-construct allies roll morale and rout.
func (g *Game) tcMoraleKill(d, src *Actor) {
	if g == nil || g.Cur == nil || d == nil {
		return
	}
	st := g.tc()
	zk := st.ZoneKills[g.Cur.Key]
	if zk == nil {
		zk = map[string]int{}
		st.ZoneKills[g.Cur.Key] = zk
	}
	fac := d.Faction
	if fac == "" {
		fac = d.Tpl
	}
	zk[fac]++
	deadN := zk[fac]

	leaderDied := tcIsLeader(d)
	aliveSight := 0
	for _, a := range g.Cur.Actors {
		if a == d || a.HP <= 0 {
			continue
		}
		af := a.Faction
		if af == "" {
			af = a.Tpl
		}
		if af != fac {
			continue
		}
		if a.Leader == d.ID && d.ID != 0 {
			leaderDied = true
		}
		if g.visible(a.X, a.Y) || dist(a, d) <= 12 {
			aliveSight++
		}
	}
	halfFell := deadN > 0 && deadN >= aliveSight
	if !leaderDied && !halfFell {
		return
	}
	for _, a := range g.Cur.Actors {
		if a == d || a.HP <= 0 {
			continue
		}
		af := a.Faction
		if af == "" {
			af = a.Tpl
		}
		if af != fac || !tcCanRout(a) {
			continue
		}
		r := NewRng(fmt.Sprintf("%s:tc:morale:%d:%d", g.Seed, a.ID, g.Turn))
		roll := r.Int(1, 20)
		dc := 16
		if leaderDied {
			dc = 19
		}
		if roll+a.M(WIL) < dc || a.M(WIL) <= 2 {
			a.AddSt("feared", 8)
			g.Msg(fmt.Sprintf("{{y|%s loses heart and routs!}}", g.The(a)))
		}
	}
}

// tcMoraleTurn runs once per game minute in turnHooks: ticks tactical smoke/caltrops
// and lets surviving bosses/captains issue a rally cry to clear fear on nearby allies.
func tcMoraleTurn(g *Game) {
	if g == nil || g.Cur == nil {
		return
	}
	st := g.tc()
	if sm := st.Smoke[g.Cur.Key]; len(sm) > 0 {
		for idx, v := range sm {
			if v <= 1 {
				delete(sm, idx)
			} else {
				sm[idx] = v - 1
				x, y := idx%MW, idx/MW
				if len(g.vis) == MW*MH && g.Cur.In(x, y) {
					g.vis[idx] = false
				}
			}
		}
	}
	for _, a := range g.Cur.Actors {
		if a.HP <= 0 {
			continue
		}
		g.tcTriggerCaltrops(a)
	}
	for _, b := range g.Cur.Actors {
		if b.HP <= 0 || !tcIsLeader(b) || b.St("feared") > 0 || b.St("stunned") > 0 {
			continue
		}
		if b.Cool != nil && b.Cool["tc_rally"] > 0 {
			b.Cool["tc_rally"]--
			continue
		}
		rallied := 0
		for _, o := range g.Cur.Actors {
			if o == b || o.HP <= 0 || o.Faction != b.Faction || dist(b, o) > 8 {
				continue
			}
			if o.St("feared") > 0 || o.St("terrified") > 0 || o.St("shaken") > 0 {
				delete(o.Status, "feared")
				delete(o.Status, "terrified")
				delete(o.Status, "shaken")
				o.AddSt("inspired", 5)
				rallied++
			}
		}
		if rallied > 0 {
			if b.Cool == nil {
				b.Cool = map[string]int{}
			}
			b.Cool["tc_rally"] = 8
			g.Msg(fmt.Sprintf("{{W|%s rallies the line!}}", g.The(b)))
		}
	}
}
