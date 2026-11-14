package engine

import (
	"fmt"
	"sort"
	"strings"
)

// foes.go (bw): enemy combat depth. Damage types and per-creature resistances, boss/elite phases
// with telegraphed lines, special enemy tactics, named elite variants with affixes and a sidebar
// line for known weaknesses. Everything is derived deterministically; play-time rolls use g.R,
// generation and phase content use side hashes only.

type bwState struct {
	Learn map[string]map[string]int  `json:"l,omitempty"` // hits taken per type, per creature key
	Known map[string]map[string]bool `json:"k,omitempty"`
	Tele  map[int]string             `json:"t,omitempty"` // actor id -> phase kind winding up
	Flee  map[int]int                `json:"f,omitempty"`
	Guard map[int]int                `json:"g,omitempty"` // bodyguard id -> leader id
	Cool  map[int]int                `json:"c,omitempty"`
	Last  map[int]string             `json:"x,omitempty"` // last damage type taken (phase 'resist')
}

func (g *Game) bw() *bwState {
	s := g.ext("bw", func() any { return &bwState{} }).(*bwState)
	if s.Learn == nil {
		s.Learn = map[string]map[string]int{}
		s.Known = map[string]map[string]bool{}
		s.Tele = map[int]string{}
		s.Flee = map[int]int{}
		s.Guard = map[int]int{}
		s.Cool = map[int]int{}
		s.Last = map[int]string{}
	}
	return s
}

func init() {
	zoneHooks = append(zoneHooks, bwZone)
	turnHooks = append(turnHooks, bwTurn)
	killHooks = append(killHooks, bwKill)
	sideHooks = append(sideHooks, bwSide)
}

func bwHas(s string, words []string) bool {
	for _, w := range words {
		if strings.Contains(s, w) {
			return true
		}
	}
	return false
}

// ---------- damage types and resistances ----------

var bwProfCache = map[string]map[string]int{}

// bwBase is the creature's innate resistance table (percent; negative = weakness, 100 = immune).
func (g *Game) bwBase(a *Actor) map[string]int {
	key := g.Genre + "|" + a.Tpl + "|" + a.Name
	if p, ok := bwProfCache[key]; ok {
		return p
	}
	p := map[string]int{}
	add := func(m map[string]int) {
		for k, v := range m {
			p[k] += v
		}
	}
	for f := range a.Flags {
		add(bwFlagRes[f])
	}
	name := strings.ToLower(a.Tpl + " " + a.Name)
	if t := tplByID[a.Tpl]; t != nil {
		name += " " + strings.ToLower(t.Name(g.Genre))
		for _, f := range strings.Fields(t.Flags) {
			if a.Flags == nil || !a.Flags[f] {
				add(bwFlagRes[f])
			}
		}
		if t.Elem != "" {
			switch t.Elem {
			case "frost":
				p["cold"] += 40
			default:
				p[t.Elem] += 40
			}
		}
	}
	if a.Tpl == "golem" {
		name += " golem"
	}
	for _, n := range bwNameRes {
		if bwHas(name, n.Words) {
			add(n.Res)
		}
	}
	add(bwGenreRes[g.Genre])
	if a.AVNat >= 6 {
		p["slash"] += 10
		p["pierce"] += 5
	}
	for k, v := range p {
		p[k] = clampi(v, -75, 100)
		if p[k] == 0 {
			delete(p, k)
		}
	}
	bwProfCache[key] = p
	return p
}

// bwProfile is the live table: the base plus affix and phase effects.
func (g *Game) bwProfile(a *Actor) map[string]int {
	base := g.bwBase(a)
	if len(a.Flags) == 0 {
		return base
	}
	var out map[string]int
	set := func(k string, v int) {
		if out == nil {
			out = map[string]int{}
			for a, b := range base {
				out[a] = b
			}
		}
		out[k] = clampi(out[k]+v, -75, 100)
	}
	if a.Has("bwaff_armoured") {
		set("slash", 25)
		set("pierce", 25)
		set("blunt", 15)
	}
	if a.Has("bwaff_warded") {
		for _, t := range []string{"fire", "cold", "shock", "holy", "dark", "psychic"} {
			set(t, 30)
		}
	}
	for _, t := range bwTypes {
		if a.Has("bwshield:" + t) {
			set(t, 60)
		}
		if a.Has("bwopen:" + t) {
			set(t, -50)
		}
	}
	if out == nil {
		return base
	}
	return out
}

// bwTypeOf names the damage type of a weapon (nil allowed) or a hurt() cause string.
func bwTypeOf(w *Item, cause string) string {
	if w != nil {
		switch w.Elem {
		case "fire", "shock", "acid", "poison", "holy", "dark", "psychic":
			return w.Elem
		case "frost", "cold":
			return "cold"
		}
		if t, ok := bwPhysSkill[w.Skill]; ok {
			return t
		}
		if w.Skill != "natural" && w.Skill != "" {
			if strings.Contains(w.Skill, "blade") {
				return "slash"
			}
		}
		if cause == "" {
			cause = w.Name
		}
	}
	c := strings.ToLower(cause)
	for _, e := range bwCauseType {
		if strings.Contains(c, e.Word) {
			return e.Type
		}
	}
	switch {
	case bwHas(c, bwNatPierce):
		return "pierce"
	case bwHas(c, bwNatSlash):
		return "slash"
	}
	if w != nil && (w.Skill == "natural" || w.Skill == "") && w.DS > 0 {
		return "blunt"
	}
	if c == "" {
		return ""
	}
	return "blunt"
}

func (g *Game) bwKey(d *Actor) string {
	for _, t := range bwTypes {
		if d.Has("bwshield:"+t) || d.Has("bwopen:"+t) {
			return fmt.Sprintf("a%d", d.ID)
		}
	}
	return d.Tpl
}

// bwKnows reports whether the player knows how d reacts to the type.
func (g *Game) bwKnows(d *Actor, t string) bool {
	if g.Skills["lore"] >= 2 {
		return true
	}
	prof := g.bwProfile(d)
	if g.Skills["lore"] >= 1 {
		best, bv := "", 0
		for k, v := range prof {
			if abs(v) > bv {
				best, bv = k, abs(v)
			}
		}
		if best == t {
			return true
		}
	}
	if g.bwKey(d) == d.Tpl {
		if b := g.codex().Beasts[d.Tpl]; b != nil && b.Kills >= 3 {
			return true
		}
	}
	return g.bw().Known[g.bwKey(d)][t]
}

func bwPhrase(pct int) string {
	switch {
	case pct >= 100:
		return "immune to"
	case pct > 0:
		return "resistant to"
	}
	return "weak to"
}

// bwDamageMod scales damage n of type dmgType against d. src is the attacker (may be nil).
func (g *Game) bwDamageMod(d, src *Actor, dmgType string, n int) int {
	if d == nil || n <= 0 || dmgType == "" || d == g.P {
		return n
	}
	g.bw().Last[d.ID] = dmgType
	pct := g.bwProfile(d)[dmgType]
	if pct == 0 {
		return n
	}
	out := n * (100 - pct) / 100
	if pct < 100 && out < 1 {
		out = 1
	}
	if src == g.P {
		st := g.bw()
		known := g.bwKnows(d, dmgType)
		if !known {
			k := g.bwKey(d)
			if st.Learn[k] == nil {
				st.Learn[k] = map[string]int{}
			}
			st.Learn[k][dmgType]++
			if abs(pct) >= 25 && (st.Learn[k][dmgType] >= 3 || pct >= 100) {
				if st.Known[k] == nil {
					st.Known[k] = map[string]bool{}
				}
				st.Known[k][dmgType] = true
				g.Msg(fmt.Sprintf("{{G|You learn that %s is %s %s.}}", g.the(d), bwPhrase(pct), dmgType))
			}
		} else if abs(pct) >= 40 && g.visible(d.X, d.Y) {
			if pct > 0 {
				g.Msg(fmt.Sprintf("{{K|%s shrugs off the %s.}}", g.The(d), dmgType))
			} else {
				g.Msg(fmt.Sprintf("{{G|%s is vulnerable to %s!}}", g.The(d), dmgType))
			}
		}
	}
	return out
}

// bwWeakText lists what the player knows about d ("weak: fire, cold; resists: slash").
func (g *Game) bwWeakText(d *Actor) (weak, res []string, unknown bool) {
	prof := g.bwProfile(d)
	keys := make([]string, 0, len(prof))
	for k := range prof {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		if !g.bwKnows(d, k) {
			unknown = true
			continue
		}
		if prof[k] < 0 {
			weak = append(weak, k)
		} else if prof[k] >= 100 {
			res = append(res, k+" (immune)")
		} else {
			res = append(res, k)
		}
	}
	return
}

// ---------- sidebar ----------

func bwTarget(g *Game) *Actor {
	if g.Cur == nil || g.P == nil {
		return nil
	}
	var best *Actor
	bd := 11
	for _, a := range g.Cur.Actors {
		if a == g.P || a.HP <= 0 || a.AI == "follower" || a.Tpl == "" || a.Tpl == "npc" {
			continue
		}
		if d := dist(a, g.P); d < bd && g.visible(a.X, a.Y) && g.Hostile(a) {
			best, bd = a, d
		}
	}
	return best
}

func bwSide(g *Game) []string {
	d := bwTarget(g)
	if d == nil {
		return nil
	}
	var out []string
	if w := bwAffixWords(d); w != "" {
		out = append(out, "{{O|"+d.Name+":}} {{R|"+w+"}}")
	}
	weak, res, unknown := g.bwWeakText(d)
	var parts []string
	if len(weak) > 0 {
		parts = append(parts, "{{G|weak: "+strings.Join(weak, ", ")+"}}")
	}
	if len(res) > 0 {
		parts = append(parts, "{{c|resists: "+strings.Join(res, ", ")+"}}")
	}
	if len(parts) > 0 {
		out = append(out, strings.Join(parts, " "))
	} else if unknown {
		out = append(out, "{{K|weaknesses unknown}}")
	}
	return out
}

func bwAffixWords(a *Actor) string {
	var w []string
	for _, af := range bwAffixes {
		if a.Has("bwaff_" + af.ID) {
			w = append(w, af.Word)
		}
	}
	return strings.Join(w, ", ")
}

// ---------- elites ----------

func bwZone(g *Game, zn *Zone, _ *Rng) {
	if zn == nil || len(zn.Actors) == 0 {
		return
	}
	er := NewRng(g.Seed + ":bw:elite:" + zn.Key)
	chance := minf(0.10+0.02*float64(zn.Level)+0.03*float64(maxi(zn.Z, 0)), 0.4)
	if !er.Chance(chance) {
		return
	}
	var cand []*Actor
	for _, a := range zn.Actors {
		if a.HP <= 0 || a.Boss || a.Has("boss") || a.SimMon > 0 || a.SimPer > 0 || a.Role != "" || a.Level < 2 || a.Has("still") || a.Has("bwelite") {
			continue
		}
		if a.AI != "hostile" || a.Tpl == "" || a.Tpl == "npc" || strings.HasPrefix(a.Faction, "state:") || a.Faction == "wildlife" || a.Comp != nil {
			continue
		}
		cand = append(cand, a)
	}
	if len(cand) == 0 {
		return
	}
	g.bwMakeElite(zn, Pick(er, cand), er)
}

func (g *Game) bwMakeElite(zn *Zone, a *Actor, er *Rng) {
	base := a.Name
	a.Set("bwelite")
	var pool []bwAffixDef
	for _, af := range bwAffixes {
		if af.Min <= a.Level {
			pool = append(pool, af)
		}
	}
	n := 1
	if a.Level >= 6 {
		n = 2
	}
	var picked []bwAffixDef
	for i := 0; i < n && len(pool) > 0; i++ {
		j := er.Int(0, len(pool)-1)
		picked = append(picked, pool[j])
		pool = append(pool[:j], pool[j+1:]...)
	}
	for _, af := range picked {
		a.Set("bwaff_" + af.ID)
		switch af.ID {
		case "armoured":
			a.AVNat += 3
		case "swift":
			a.Speed += 20
		}
		a.Desc += " " + af.Hint
	}
	a.MaxHP = a.MaxHP * 3 / 2
	a.HP = a.MaxHP
	a.XP = a.XP * 2
	a.Gold += 8 * a.Level
	a.Name = Pick(er, bwScarPrefix) + " " + titleCase(base)
	a.Proper = true
	if er.Chance(0.7) {
		it := g.Loot(er, clampi(a.Level/3+1, 1, 5))
		if it != nil {
			a.Inv = append(a.Inv, it)
		}
	}
	// bodyguards: a few of its own kind nearby
	st := g.bw()
	cnt := 0
	for _, o := range zn.Actors {
		if o == a || o.HP <= 0 || o.Tpl != a.Tpl || o.Boss || o.Has("bwelite") || cnt >= 3 {
			continue
		}
		if dist(o, a) <= 8 {
			st.Guard[o.ID] = a.ID
			cnt++
		}
	}
}

func bwTurn(g *Game) {
	if g.Cur == nil || g.P == nil {
		return
	}
	for _, a := range g.Cur.Actors {
		if a.HP > 0 && a.HP < a.MaxHP && a.Has("bwaff_regen") {
			a.HP = mini(a.MaxHP, a.HP+1+a.Level/5)
		}
	}
}

func bwKill(g *Game, d, src *Actor) {
	if d == nil || d == g.P {
		return
	}
	st := g.bw()
	delete(st.Tele, d.ID)
	delete(st.Flee, d.ID)
	delete(st.Last, d.ID)
	if d.Has("bwaff_explosive") && g.Cur != nil {
		if g.visible(d.X, d.Y) {
			g.Msg(fmt.Sprintf("{{R|%s bursts in a volatile blast!}}", g.The(d)))
		}
		g.explode(d.X, d.Y, "fire", 2+d.Level/3, d)
	}
	if d.Has("bwelite") && g.visible(d.X, d.Y) && src == g.P {
		g.Msg("{{Y|The scarred one is finished; you take something of its hoard.}}")
	}
	// bodyguards lose their leader and rage
	for id, ld := range st.Guard {
		if ld == d.ID {
			delete(st.Guard, id)
		}
	}
	delete(st.Guard, d.ID)
}

// bwOnHit: affix effects when an elite strikes (a = attacker, dmg dealt). Hook in swing after hurt.
func (g *Game) bwOnHit(a, d *Actor, dmg int) {
	if a == nil || d == nil || len(a.Flags) == 0 || d.HP <= 0 && !a.Has("bwaff_vampiric") {
		return
	}
	if a.Has("bwaff_venom") && d.HP > 0 && !d.Has("undead") && g.R.Chance(0.5) {
		d.AddSt("poisoned", 5)
		if d == g.P {
			g.Msg("{{G|The venom burns in your veins.}}")
		}
	}
	if a.Has("bwaff_vampiric") && dmg > 0 && a.HP < a.MaxHP {
		a.HP = mini(a.MaxHP, a.HP+maxi(1, dmg/2))
		if g.visible(a.X, a.Y) {
			g.Msg(fmt.Sprintf("{{r|%s drinks your blood.}}", g.The(a)))
		}
	}
	if a.Has("bwaff_thief") && d == g.P && !a.Has("bwstole") && g.R.Chance(0.4) {
		for i, it := range g.P.Inv {
			if it != nil && it.Kind == "food" {
				take := *it
				take.Count = 1
				if it.Count > 1 {
					it.Count--
				} else {
					g.P.Inv = append(g.P.Inv[:i], g.P.Inv[i+1:]...)
				}
				a.Inv = append(a.Inv, &take)
				a.Set("bwstole")
				g.bw().Flee[a.ID] = 14
				g.Msg(fmt.Sprintf("{{R|%s snatches %s from your pack and bolts!}}", g.The(a), take.Name))
				return
			}
		}
	}
}

// ---------- AI: phases and tactics ----------

// bwAct runs before the stock AI; true = the actor's turn is used.
func (g *Game) bwAct(a *Actor) bool {
	if g.Cur == nil || a == g.P || a.HP <= 0 || a.AI == "follower" || a.Comp != nil || a.Tpl == "" {
		return false
	}
	st := g.bw()
	boss := a.Boss || a.Has("boss")
	elite := a.Has("bwelite")
	_, guard := st.Guard[a.ID]
	role := g.bwRole(a)
	if !boss && !elite && !guard && role == 0 && a.Missile() == nil && !a.Has("pack") {
		return false
	}
	if a.Has("bwaff_frenzy") && a.HP < a.MaxHP/2 && a.St("mighty") == 0 {
		a.AddSt("mighty", 30)
		if g.visible(a.X, a.Y) {
			g.Msg(fmt.Sprintf("{{R|%s goes wild with pain!}}", g.The(a)))
		}
	}
	if kind := st.Tele[a.ID]; kind != "" {
		delete(st.Tele, a.ID)
		g.bwDo(a, kind)
		return true
	}
	if n := st.Flee[a.ID]; n > 0 {
		st.Flee[a.ID] = n - 1
		if g.stepToward(a, g.distMap(g.P.X, g.P.Y, a.Has("flying")), true) {
			return true
		}
		st.Flee[a.ID] = 0
	}
	if (boss || elite) && g.Hostile(a) {
		if k := g.bwPhaseCheck(a, boss); k != "" {
			st.Tele[a.ID] = k
			if g.visible(a.X, a.Y) {
				g.Msg("{{O|" + strings.ReplaceAll(Pick(NewRng(fmt.Sprint(g.Seed, a.ID, k, g.Turn)), bwTele[k]), "{n}", g.The(a)) + "}}")
			}
			return true
		}
	}
	tgt := g.bwTargetFor(a)
	if tgt == nil {
		return false
	}
	d := dist(a, tgt)
	flying := a.Has("flying")
	// cowards break and run when hurt
	if role&2 != 0 && !boss && a.HP < a.MaxHP*35/100 && !a.Has("undead") {
		if g.stepToward(a, g.distMap(tgt.X, tgt.Y, flying), true) {
			if !a.Has("bwcry") {
				a.Set("bwcry")
				if g.visible(a.X, a.Y) {
					g.Msg(fmt.Sprintf("%s breaks and runs!", g.The(a)))
				}
			}
			return true
		}
		return false
	}
	// bodyguards cover their leader
	if lid, ok := st.Guard[a.ID]; ok {
		var ld *Actor
		for _, o := range g.Cur.Actors {
			if o.ID == lid && o.HP > 0 {
				ld = o
				break
			}
		}
		if ld == nil {
			delete(st.Guard, a.ID)
		} else if dist(ld, tgt) <= 8 && d > 1 {
			cx, cy := ld.X+sgn(tgt.X-ld.X), ld.Y+sgn(tgt.Y-ld.Y)
			if a.X == cx && a.Y == cy {
				if m := a.Missile(); m != nil && d <= m.Range && g.losClear(a.X, a.Y, tgt.X, tgt.Y) {
					g.shoot(a, tgt.X, tgt.Y, m)
				}
				return true
			}
			if g.canEnter(a, cx, cy) && g.stepToward(a, g.distMap(cx, cy, flying), false) {
				return true
			}
		}
	}
	// casters keep range and bolster allies
	if role&1 != 0 && g.bwCast(a, tgt, d) {
		return true
	}
	// archers keep their distance
	if m := a.Missile(); m != nil && a.AI == "hostile" {
		if d <= 2 && g.R.Chance(0.6) {
			if g.stepToward(a, g.distMap(tgt.X, tgt.Y, flying), true) {
				return true
			}
		}
		if d >= 2 && d <= m.Range && g.losClear(a.X, a.Y, tgt.X, tgt.Y) {
			g.shoot(a, tgt.X, tgt.Y, m)
			return true
		}
		return false
	}
	// pack hunters surround
	if a.Has("pack") && d >= 2 && d <= 9 {
		if g.bwSurround(a, tgt) {
			return true
		}
	}
	return false
}

var bwRoleCache = map[string]int{}

// bwRole: bit 1 caster, bit 2 coward.
func (g *Game) bwRole(a *Actor) int {
	key := a.Tpl + "|" + a.Name
	if r, ok := bwRoleCache[key]; ok {
		return r
	}
	n := strings.ToLower(a.Tpl + " " + a.Name)
	r := 0
	if a.Has("caster") || bwHas(n, bwCasterWords) {
		r |= 1
	}
	if (a.Has("flee") || bwHas(n, bwCowardWords) || a.Faction == "vermin") && !a.Boss && !a.Has("undead") && a.Tpl != "golem" && !a.Has("bwelite") {
		r |= 2
	}
	bwRoleCache[key] = r
	return r
}

func (g *Game) bwTargetFor(a *Actor) *Actor {
	p := g.P
	if !g.Hostile(a) {
		return nil
	}
	d := dist(a, p)
	if d > 14 || !(g.visible(a.X, a.Y) || d < 6) {
		return nil
	}
	return p
}

func (g *Game) bwSurround(a, tgt *Actor) bool {
	var mates []*Actor
	for _, o := range g.Cur.Actors {
		if o != a && o.HP > 0 && o.Has("pack") && o.Faction == a.Faction && dist(o, tgt) <= 10 {
			mates = append(mates, o)
		}
	}
	if len(mates) == 0 {
		return false
	}
	bx, by, bs := -1, -1, -1<<30
	for dy := -1; dy <= 1; dy++ {
		for dx := -1; dx <= 1; dx++ {
			if dx == 0 && dy == 0 {
				continue
			}
			x, y := tgt.X+dx, tgt.Y+dy
			if !g.canEnter(a, x, y) && !(a.X == x && a.Y == y) {
				continue
			}
			sp := 6
			for _, o := range mates {
				if c := maxi(abs(o.X-x), abs(o.Y-y)); c < sp {
					sp = c
				}
			}
			s := sp*2 - maxi(abs(a.X-x), abs(a.Y-y))
			if s > bs {
				bx, by, bs = x, y, s
			}
		}
	}
	if bx < 0 || (a.X == bx && a.Y == by) {
		return false
	}
	st := g.bw()
	if g.Turn-st.Cool[-1] > 40 && g.visible(a.X, a.Y) {
		st.Cool[-1] = g.Turn
		g.Msg(fmt.Sprintf("{{O|%s fan out to surround you!}}", titleCase(g.the(a))+"s"))
	}
	return g.stepToward(a, g.distMap(bx, by, a.Has("flying")), false)
}

func (g *Game) bwCast(a, tgt *Actor, d int) bool {
	st := g.bw()
	if d <= 2 && g.R.Chance(0.5) {
		if g.stepToward(a, g.distMap(tgt.X, tgt.Y, a.Has("flying")), true) {
			return true
		}
	}
	if g.Turn-st.Cool[a.ID] < 6 || !g.R.Chance(0.6) {
		return false
	}
	var hurt, plain *Actor
	for _, o := range g.Cur.Actors {
		if o == a || o.HP <= 0 || o == g.P || o.AI == "follower" || o.Faction != a.Faction || dist(a, o) > 7 || !g.losClear(a.X, a.Y, o.X, o.Y) {
			continue
		}
		if o.HP*2 < o.MaxHP {
			if hurt == nil || o.HP < hurt.HP {
				hurt = o
			}
		} else if o.St("mighty") == 0 && dist(o, tgt) <= 10 && plain == nil {
			plain = o
		}
	}
	vis := g.visible(a.X, a.Y)
	switch {
	case hurt != nil:
		hurt.HP = mini(hurt.MaxHP, hurt.HP+3+hurt.Level/2+g.R.Int(1, 6))
		if vis {
			g.Msg(fmt.Sprintf("{{O|%s mends %s with a murmured word.}}", g.The(a), g.the(hurt)))
		}
	case plain != nil:
		plain.AddSt("mighty", 10)
		if vis {
			g.Msg(fmt.Sprintf("{{O|%s empowers %s: its eyes flare!}}", g.The(a), g.the(plain)))
		}
	default:
		return false
	}
	st.Cool[a.ID] = g.Turn
	return true
}

var bwOpposite = map[string]string{"fire": "cold", "cold": "fire", "shock": "acid", "acid": "shock", "holy": "dark", "dark": "holy",
	"poison": "psychic", "psychic": "poison", "slash": "blunt", "blunt": "pierce", "pierce": "slash"}

// bwPhaseCheck: which phase kind should start now (empty = none).
func (g *Game) bwPhaseCheck(a *Actor, boss bool) string {
	thr := []int{50}
	if boss {
		thr = []int{75, 50, 25}
	}
	for i, t := range thr {
		fl := fmt.Sprintf("bwph%d", i)
		if a.Has(fl) {
			continue
		}
		if a.HP*100 > a.MaxHP*t {
			return ""
		}
		a.Set(fl)
		start := int(HashStr(fmt.Sprint(g.Seed, ":bw:phase:", a.Tpl, ":", a.ID, ":", i)) % uint64(len(bwPhaseKinds)))
		for j := 0; j < len(bwPhaseKinds); j++ {
			k := bwPhaseKinds[(int(start)+j)%len(bwPhaseKinds)]
			if a.Has("bwk_"+k) || k == "heal" && a.Has("bwhealed") {
				continue
			}
			if k == "flee" && (boss && i < 2 || a.Has("bwaff_thief")) {
				continue
			}
			a.Set("bwk_" + k)
			return k
		}
		return ""
	}
	return ""
}

func (g *Game) bwDo(a *Actor, kind string) {
	st := g.bw()
	vis := g.visible(a.X, a.Y)
	say := func(f string, args ...any) {
		if vis {
			g.Msg("{{R|" + strings.ReplaceAll(fmt.Sprintf(f, args...), "{n}", g.The(a)) + "}}")
		}
	}
	rr := NewRng(fmt.Sprint(g.Seed, ":bw:do:", a.ID, kind, g.Turn))
	switch kind {
	case "enrage":
		a.Speed += 15
		a.AddSt("mighty", 60)
		say(Pick(rr, bwDone["enrage"]))
	case "heal":
		a.Set("bwhealed")
		a.HP = mini(a.MaxHP, a.HP+a.MaxHP*35/100)
		say(Pick(rr, bwDone["heal"]))
	case "flee":
		st.Flee[a.ID] = 6
		say(Pick(rr, bwDone["flee"]))
	case "resist":
		last := st.Last[a.ID]
		if last == "" {
			last = "slash"
		}
		for _, t := range bwTypes {
			delete(a.Flags, "bwshield:"+t)
			delete(a.Flags, "bwopen:"+t)
		}
		op := bwOpposite[last]
		a.Set("bwshield:" + last)
		a.Set("bwopen:" + op)
		say(fmt.Sprintf(bwDone["resist"][0], last, op))
	case "summon":
		n := 2
		if a.Boss {
			n = 3
		}
		g.bwSummon(a, n, rr)
		say(Pick(rr, bwDone["summon"]))
	}
}

func (g *Game) bwSummon(a *Actor, n int, rr *Rng) {
	zn := g.Cur
	where := byte('s')
	if zn.Z > 0 {
		where = 'd'
	}
	lvl := maxi(2, a.Level-3)
	made := 0
	for i := 0; i < 24 && made < n; i++ {
		x, y := a.X+rr.Int(-3, 3), a.Y+rr.Int(-3, 3)
		if !g.canEnter(a, x, y) {
			continue
		}
		id := pickCreature(rr, zn.Biome, where, lvl)
		if t := tplByID[id]; t != nil && (t.Faction == "wildlife") {
			id = a.Tpl
		}
		m := g.Spawn(id, lvl, rr)
		m.Faction = a.Faction
		m.AI = "hostile"
		m.X, m.Y = x, y
		m.Energy = 0
		m.XP /= 2
		zn.Actors = append(zn.Actors, m)
		g.bw().Guard[m.ID] = a.ID
		made++
	}
}
