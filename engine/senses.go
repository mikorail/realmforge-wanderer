package engine

import (
	"fmt"
	"hash/fnv"
	"strings"
)

// Noise and smell as stealth inputs (prefix ns).
//
// Noise: every turn the player's actions (steps by terrain, armour, rattling gear, fighting,
// gunfire, shouting, broken doors) give a noise value that decays 40% per turn. Awake creatures
// within its radius may walk to the source (nsInvestigate, called from aiAct).
// Smell: carried goods, blood, wet fur, smoke and rot make a scent value; the player leaves a
// scent trail that predators and hounds follow; wind per zone/day/weather carries it. Mud, herbs,
// rain, incense and water crossings mask it. nsNoticeBonus feeds ssNotices.

type nsMark struct {
	X, Y int
	T    int
	S    int
}

type nsTarget struct {
	X, Y, Until int
}

type nsState struct {
	Noise    int                 `json:"nz,omitempty"`
	Scent    int                 `json:"sc,omitempty"`
	Blood    int                 `json:"bl,omitempty"`
	Wet      int                 `json:"wt,omitempty"`
	Mud      int                 `json:"md,omitempty"`
	Incense  int                 `json:"ic,omitempty"`
	Smoke    int                 `json:"sm,omitempty"`
	Soak     int                 `json:"sk,omitempty"`
	LastLoud int                 `json:"ll,omitempty"`
	GuardMsg int                 `json:"gm,omitempty"`
	Trails   map[string][]nsMark `json:"tr,omitempty"`
	Target   map[int]nsTarget    `json:"tg,omitempty"`
	Reach    map[int]int         `json:"rc,omitempty"`
	zone     string
	px, py   int
	have     bool
	Why      []string `json:"-"`
	SWhy     []string `json:"-"`
	Contra   bool     `json:"-"`
}

func (g *Game) ns() *nsState {
	st := g.ext("ns", func() any { return &nsState{} }).(*nsState)
	if st.Trails == nil {
		st.Trails = map[string][]nsMark{}
	}
	if st.Target == nil {
		st.Target = map[int]nsTarget{}
	}
	if st.Reach == nil {
		st.Reach = map[int]int{}
	}
	return st
}

func init() {
	unLocalKeys = append(unLocalKeys, "ns")
	turnHooks = append(turnHooks, nsTurn)
	sideHooks = append(sideHooks, nsSide)
	killHooks = append(killHooks, nsKill)
	enterHooks = append(enterHooks, func(g *Game, zn *Zone) {
		st := g.ns()
		st.have = false
		st.Target = map[int]nsTarget{}
		st.Reach = map[int]int{}
	})
}

func nsHas(s string, words []string) bool {
	for _, w := range words {
		if strings.Contains(s, w) {
			return true
		}
	}
	return false
}

func nsFirst(s string, tab []nsKw) (int, bool) {
	for _, e := range tab {
		if strings.Contains(s, e.K) {
			return e.V, true
		}
	}
	return 0, false
}

// ---- labels ----

func nsNoiseLabel(n int) string {
	switch {
	case n >= nsNoiseLoud:
		return "loud"
	case n >= nsNoiseMuted:
		return "muted"
	}
	return "quiet"
}

func nsScentLabel(n int) string {
	switch {
	case n >= nsScentReek:
		return "reeking"
	case n >= nsScentStrong:
		return "strong"
	}
	return "faint"
}

// ---- wind ----

func nsHash(parts ...any) uint64 {
	h := fnv.New64a()
	fmt.Fprint(h, parts...)
	return h.Sum64()
}

// nsWind: direction the wind blows toward and its speed (0 still .. 3 gale) for the current zone,
// four-hour bucket and weather. No g.R.
func (g *Game) nsWind() (dx, dy, sp int) {
	z := g.Cur
	if z == nil || !outdoors(z) {
		return 0, 0, 0
	}
	h := nsHash("wind|", g.Seed, "|", z.WX/2, "|", z.WY/2, "|", g.Turn/240)
	d := dirs8[h%8]
	sp = int((h >> 8) % 3)
	if w := g.activeWeather(); w != nil {
		switch w.Kind.ID {
		case "storm", "tempest", "blizzard", "sandstorm", "radstorm", "ashrain":
			sp = 3
		case "fog", "heat":
			sp = 0
		case "rain", "snow", "acidrain", "fallout":
			sp = mini(3, sp+1)
		}
	}
	return d[0], d[1], sp
}

func nsWindText(dx, dy, sp int) string {
	if sp == 0 {
		return "the air is still"
	}
	name := "north"
	for i, d := range dirs8 {
		if d[0] == dx && d[1] == dy {
			name = nsWindNames[i]
		}
	}
	return fmt.Sprintf("%s wind carrying toward the %s", []string{"", "light", "steady", "strong"}[sp], name)
}

// ---- noise ----

func (g *Game) nsArmourNoise() (int, string) {
	sum := 0
	why := ""
	for _, bp := range g.P.Body {
		it := bp.Item
		if it == nil || bp.Severed || it.Kind != "armor" {
			continue
		}
		n := strings.ToLower(it.Name)
		v, ok := nsFirst(n, nsArmourLoud)
		if !ok {
			v = clampi(it.Weight/8, 0, 2)
		}
		if it.Extra != nil {
			v += it.Extra["noise"]
		}
		if v > 0 && why == "" {
			why = it.Name + " clanks"
		}
		sum += v
	}
	return clampi(sum, -2, 6), why
}

func (g *Game) nsPackRattle() int {
	n := 0
	for _, it := range g.P.Inv {
		if v, ok := nsFirst(strings.ToLower(it.Name), nsRattle); ok {
			n += v
		}
		if it.Extra != nil && it.Extra["noise"] > 0 && it.Kind != "armor" {
			n += it.Extra["noise"] // loud items carry an explicit flag
		}
	}
	return clampi(n, 0, 4)
}

// nsWeaponNoise: noise of a swing with the wielded weapon.
func (g *Game) nsWeaponNoise() int {
	v := 8
	if ws := g.P.Weapons(); len(ws) > 0 {
		if d, ok := nsFirst(strings.ToLower(ws[0].Name), nsWeaponLoud); ok {
			v += d
		}
		if ws[0].Extra != nil {
			v += ws[0].Extra["noise"]
		}
	}
	return clampi(v, 3, 16)
}

// nsEmit: a noise of volume vol at (x, y). byPlayer sets the player's noise value; nearby awake
// creatures may come to investigate.
func (g *Game) nsEmit(x, y, vol int, why string, byPlayer bool) {
	if g.Cur == nil || vol <= 0 {
		return
	}
	st := g.ns()
	if byPlayer {
		if vol > st.Noise {
			st.Noise = vol
		}
		if why != "" {
			st.Why = append(st.Why, why)
		}
	}
	if vol < nsNoiseMuted {
		return
	}
	wetMask := 0
	if w := g.activeWeather(); w != nil && w.Kind.Wet >= 2 {
		wetMask = 2
	}
	for _, a := range g.Cur.Actors {
		if a == g.P || a.HP <= 0 || a.AI == "follower" || a.Angry {
			continue
		}
		switch {
		case a.AI == "guard" || a.Role == "guard":
			if vol < nsNoiseLoud {
				continue
			}
		case a.AI == "hostile" || a.AI == "wander" || a.Faction == "beasts" || a.Has("pack"):
		default:
			continue
		}
		d := maxi(abs(a.X-x), abs(a.Y-y))
		reach := vol - wetMask
		if !g.losClear(a.X, a.Y, x, y) {
			reach = reach * 2 / 3
		}
		if d > reach || d <= 0 {
			continue
		}
		if a.St("asleep") > 0 {
			if vol < nsNoiseLoud || d > vol/2 || !g.R.Chance(0.35) {
				continue
			}
			delete(a.Status, "asleep")
		}
		if a.St("stunned") > 0 || a.St("frozen") > 0 {
			continue
		}
		ch := clampf(0.25+0.7*(1-float64(d)/float64(reach+1)), 0.1, 0.95)
		if !g.R.Chance(ch) {
			continue
		}
		if t, ok := st.Target[a.ID]; ok && t.Until > g.Turn && t.X == x && t.Y == y {
			continue
		}
		if byPlayer && g.visible(a.X, a.Y) && st.Target[a.ID].Until <= g.Turn {
			g.Msg("{{Y|" + g.The(a) + " turns toward the noise.}}")
		}
		st.Target[a.ID] = nsTarget{x, y, g.Turn + 20 + vol}
	}
}

// nsShot: hook for the player's missile or gun shot (call from shoot with the attacker and weapon).
func (g *Game) nsShot(a *Actor, w *Item) {
	if a != g.P || w == nil {
		return
	}
	v := 6
	if d, ok := nsFirst(strings.ToLower(w.Name), nsShotLoud); ok {
		v = d
	} else if w.Skill == "firearm" {
		v = 15
	}
	if w.Extra != nil {
		v += w.Extra["noise"]
	}
	g.nsEmit(a.X, a.Y, clampi(v, 1, 24), "a shot rings out", true)
}

// nsDoorBreak: hook for smashing a door or barricade at (x, y).
func (g *Game) nsDoorBreak(x, y int) {
	g.nsEmit(x, y, 12, "a door splinters", true)
}

// ---- scent ----

func (g *Game) nsScentCalc() (int, []string) {
	st := g.ns()
	total := 1
	why := []string{"body odour"}
	add := func(n int, s string) {
		if n > 0 {
			total += n
			why = append(why, fmt.Sprintf("%s +%d", s, n))
		}
	}
	meat, rot, perf, fur, smoke, contra, herbs := 0, 0, 0, 0, 0, 0, 0
	wet := st.Wet > g.Turn
	check := func(it *Item) {
		n := strings.ToLower(it.Name)
		cooked := nsHas(n, nsCookedWords)
		switch {
		case it.Kind == "corpse" || nsHas(n, nsRotWords):
			rot++
		case nsHas(n, nsMeatWords) && !cooked && (it.Kind == "food" || it.Kind == "ingredient" || strings.Contains(n, "raw")):
			meat++
		}
		if nsHas(n, nsPerfumeWords) {
			perf++
		}
		if nsHas(n, nsFurWords) && wet {
			fur++
		}
		if nsHas(n, nsSmokeWords) && !strings.Contains(n, "torch") {
			smoke++
		}
		if nsHas(n, nsContraWords) {
			contra++
		}
		if nsHas(n, nsHerbWords) || (it.Kind == "ingredient" && strings.Contains(n, "herb")) {
			herbs++
		}
	}
	for _, it := range g.P.Inv {
		check(it)
	}
	for _, bp := range g.P.Body {
		if bp.Item != nil && !bp.Severed {
			check(bp.Item)
		}
	}
	st.Contra = contra > 0
	add(mini(meat*3, 8), "raw meat")
	add(mini(rot*5, 10), "rotting flesh")
	add(mini(perf*4, 6), "perfume")
	add(mini(fur*2, 4), "wet fur")
	add(mini(smoke+contra, 2), "pungent goods")
	if st.Blood > g.Turn {
		b := 4
		if g.P.St("bleeding") > 0 {
			b = 6
		}
		add(b, "blood on you")
	}
	if st.Smoke > g.Turn || g.P.St("burning") > 0 {
		add(2, "smoke in your clothes")
	}
	mask := 0
	madd := func(n int, s string) {
		mask += n
		why = append(why, fmt.Sprintf("%s -%d", s, n))
	}
	if st.Mud > g.Turn {
		madd(4, "caked in mud")
	}
	if st.Incense > g.Turn {
		madd(5, "incense smoke")
	}
	if herbs >= 3 {
		madd(2, "pungent herbs")
	} else if herbs > 0 {
		madd(1, "a sprig of herbs")
	}
	if st.Soak > g.Turn {
		madd(3, "just out of the water")
	}
	if w := g.activeWeather(); w != nil && w.Kind.Wet >= 1 && !roofed(g.Cur.At(g.P.X, g.P.Y)) {
		madd(3, "rain washes the air")
	}
	if g.P.St("almask") > 0 {
		madd(6, "scent mask")
	}
	total -= mask
	return clampi(total, 0, 30), why
}

func nsTrackerKind(a *Actor) bool {
	if a.Faction == "beasts" || a.Has("pack") || a.Role == "hunter" || a.Role == "tracker" {
		return true
	}
	return nsHas(strings.ToLower(a.Name+" "+a.Tpl), nsHoundWords)
}

// nsSmells: whether a (awake) actor at its distance d can smell the player right now, by scent
// level, its nose, and the wind.
func (g *Game) nsSmells(a *Actor, d int) bool {
	st := g.ns()
	sc := st.Scent
	nose := nsTrackerKind(a)
	if !nose {
		// people: only reeking carries, and only close by
		if sc < nsScentReek {
			return false
		}
	}
	var rng float64
	switch {
	case sc >= nsScentReek:
		rng = 16
	case sc >= nsScentStrong:
		rng = 9
	default:
		rng = 4
	}
	if nose {
		rng *= 1.5
	} else {
		rng = 5
	}
	dx, dy, sp := g.nsWind()
	if sp > 0 {
		dot := (a.X-g.P.X)*dx + (a.Y-g.P.Y)*dy
		if dot > 0 {
			rng *= 1 + 0.25*float64(sp)
		} else if dot < 0 {
			rng *= 0.4
		}
	}
	if g.underRoof(a.X, a.Y) != g.underRoof(g.P.X, g.P.Y) {
		rng *= 0.5
	}
	return float64(d) <= rng
}

// nsScentNotice: hook for aiAct's target test; true when the actor catches the player's scent.
func (g *Game) nsScentNotice(a *Actor) bool {
	if g.P == nil || a == nil || a.HP <= 0 || g.Cur == nil || !g.ssAwake(a) || a.AI == "follower" {
		return false
	}
	return g.nsSmells(a, dist(a, g.P))
}

// nsNoticeBonus: extra to ssNotices's noticing score for actor a at distance d (hook).
func (g *Game) nsNoticeBonus(a *Actor, d int) int {
	st := g.ns()
	b := 0
	if st.Noise >= nsNoiseMuted {
		b += clampi(st.Noise*2/(d+2), 0, 6)
	}
	if g.nsSmells(a, d) {
		lv := 1
		if st.Scent >= nsScentReek {
			lv = 3
		} else if st.Scent >= nsScentStrong {
			lv = 2
		}
		if !nsTrackerKind(a) {
			lv = 1
		}
		b += lv
	}
	return b
}

// ---- AI hook ----

func nsTrailLife(s int) int { return maxi(20, s*25) }

// nsInvestigate: hook for aiAct (when the actor has no target). Walks to a heard noise or follows
// the player's scent trail. Returns true if the actor acted.
func (g *Game) nsInvestigate(a *Actor) bool {
	if g.Cur == nil || a == nil || a.HP <= 0 || a.AI == "follower" || !g.ssAwake(a) || a.Angry {
		return false
	}
	st := g.ns()
	if t, ok := st.Target[a.ID]; ok {
		if t.Until <= g.Turn || maxi(abs(a.X-t.X), abs(a.Y-t.Y)) <= 1 {
			delete(st.Target, a.ID)
		} else if g.stepToward(a, g.distMap(t.X, t.Y, a.Has("flying")), false) {
			return true
		}
	}
	if nsTrackerKind(a) && g.Hostile(a) && dist(a, g.P) <= 28 {
		return g.nsTrack(a)
	}
	return false
}

// nsTrack: predators and hounds follow the oldest unreached fresh scent mark nearby.
func (g *Game) nsTrack(a *Actor) bool {
	st := g.ns()
	tr := st.Trails[g.Cur.Key]
	if len(tr) == 0 {
		return false
	}
	need := 4
	if nsHas(strings.ToLower(a.Name+" "+a.Tpl), nsHoundWords) {
		need = 1
	}
	for loop := 0; loop < 2; loop++ {
		var best *nsMark
		for i := range tr {
			m := &tr[i]
			if m.T <= st.Reach[a.ID] || m.S < need || g.Turn-m.T > nsTrailLife(m.S) {
				continue
			}
			if maxi(abs(a.X-m.X), abs(a.Y-m.Y)) > 30 {
				continue
			}
			if best == nil || m.T < best.T {
				best = m
			}
		}
		if best == nil {
			return false
		}
		if maxi(abs(a.X-best.X), abs(a.Y-best.Y)) <= 1 {
			st.Reach[a.ID] = best.T
			continue
		}
		return g.stepToward(a, g.distMap(best.X, best.Y, a.Has("flying")), false)
	}
	return false
}

// ---- per-turn ----

func nsTurn(g *Game) {
	if g.Cur == nil || g.P == nil || g.OnWorld || g.P.HP <= 0 {
		return
	}
	st := g.ns()
	z := g.Cur
	p := g.P
	if st.zone != z.Key || !st.have {
		st.zone, st.px, st.py, st.have = z.Key, p.X, p.Y, true
	}
	moved := p.X != st.px || p.Y != st.py
	st.px, st.py = p.X, p.Y
	st.Why = st.Why[:0]
	sneak := g.ss().Sneak
	t := z.At(p.X, p.Y)
	inWater := z.WaterAt(p.X, p.Y) > 0 || t == TShallow || t == TDeep || t == TFalls
	// decay, then this turn's sources
	st.Noise = st.Noise * 3 / 5
	if moved {
		v := nsTileNoise[t]
		why := nsTileWhy[t]
		if sneak {
			v = (v + 1) / 2
		}
		if an, aw := g.nsArmourNoise(); an != 0 {
			if sneak {
				an = an / 2
			}
			v += an
			if aw != "" && an > 0 {
				why = aw
			}
		}
		v += g.nsPackRattle()
		if sneak && v > 0 {
			v = maxi(0, v-1)
		}
		if v > 0 {
			g.nsEmit(p.X, p.Y, clampi(v, 0, 12), why, true)
		}
	}
	if ls := g.ss().Loud; ls > st.LastLoud && ls > g.Turn {
		st.LastLoud = ls
		st.Blood = maxi(st.Blood, g.Turn+40)
		g.nsEmit(p.X, p.Y, g.nsWeaponNoise(), "fighting", true)
	} else if ls < st.LastLoud && ls <= g.Turn {
		st.LastLoud = ls
	}
	if st.Noise > 0 && len(st.Why) == 0 {
		st.Why = append(st.Why, "fading echoes")
	}
	// wet, soaked, smoke
	if w := g.activeWeather(); w != nil && w.Kind.Wet >= 1 && !roofed(t) {
		st.Wet = g.Turn + 60
	}
	if inWater {
		st.Wet = g.Turn + 60
		st.Soak = g.Turn + 25
		st.Blood = 0
		st.Mud = 0
		if len(st.Trails[z.Key]) > 0 {
			st.Trails[z.Key] = nil // wading breaks the trail
		}
	}
	if p.St("burning") > 0 {
		st.Smoke = g.Turn + 30
	}
	// scent value and trail
	st.Scent, st.SWhy = g.nsScentCalc()
	if moved && !inWater && st.Scent >= 1 {
		tr := append(st.Trails[z.Key], nsMark{p.X, p.Y, g.Turn, st.Scent})
		if len(tr) > 80 {
			tr = tr[len(tr)-80:]
		}
		st.Trails[z.Key] = tr
	}
	if g.Turn%30 == 0 {
		for k, tr := range st.Trails {
			kept := tr[:0]
			for _, m := range tr {
				if g.Turn-m.T <= nsTrailLife(m.S) {
					kept = append(kept, m)
				}
			}
			if len(kept) == 0 {
				delete(st.Trails, k)
			} else {
				st.Trails[k] = kept
			}
		}
	}
	nsGuards(g, st)
}

// nsGuards: guards in towns only react lightly: a sniff at contraband, a remark at a reek.
func nsGuards(g *Game, st *nsState) {
	z := g.Cur
	if z.Kind != "town" || st.GuardMsg > g.Turn || (st.Scent < nsScentReek && !st.Contra) {
		return
	}
	for _, a := range z.Actors {
		if a == g.P || a.HP <= 0 || !(a.AI == "guard" || a.Role == "guard") || g.Hostile(a) || !g.ssAwake(a) || dist(a, g.P) > 3 {
			continue
		}
		if !g.R.Chance(0.2) {
			continue
		}
		st.GuardMsg = g.Turn + 40
		if st.Scent >= nsScentReek {
			g.Msg("{{y|" + g.The(a) + " wrinkles a nose. \"You reek. Wash before you trouble the market.\"}}")
			if g.ss().Susp > 0 {
				g.ss().Susp = mini(100, g.ss().Susp+3)
			}
		} else {
			g.Msg("{{y|" + g.The(a) + " sniffs, frowns, and lets it pass.}}")
		}
		return
	}
}

func nsKill(g *Game, d, src *Actor) {
	if src != g.P || g.P == nil {
		return
	}
	st := g.ns()
	st.Blood = maxi(st.Blood, g.Turn+180)
}

// ---- abilities (hook lines in powers.go) ----

func (g *Game) nsAbilities() []Ability {
	if g.Cur == nil || g.P == nil {
		return nil
	}
	out := []Ability{{ID: "ns_shout", Name: "Shout", Target: "self"}}
	z, p := g.Cur, g.P
	t := z.At(p.X, p.Y)
	wet := false
	for _, d := range dirs8 {
		if z.WaterAt(p.X+d[0], p.Y+d[1]) > 0 {
			wet = true
		}
	}
	if t == TDirt || t == TMoss || t == TFungus || t == TShallow || wet || z.WaterAt(p.X, p.Y) > 0 {
		out = append(out, Ability{ID: "ns_mud", Name: "Smear yourself with mud", Target: "self"})
	}
	if g.nsIncense() != nil {
		out = append(out, Ability{ID: "ns_incense", Name: "Burn incense", Target: "self"})
	}
	return out
}

func (g *Game) nsIncense() *Item {
	for _, it := range g.P.Inv {
		if strings.Contains(strings.ToLower(it.Name), "incense") {
			return it
		}
	}
	return nil
}

// nsUse runs an ns_ ability; returns true when id was one of ours.
func (g *Game) nsUse(id string) bool {
	st := g.ns()
	switch id {
	case "ns_shout":
		g.Msg("You shout.")
		g.nsEmit(g.P.X, g.P.Y, 18, "you shouted", true)
		g.spend(1000)
	case "ns_mud":
		st.Mud = g.Turn + 240
		g.Msg("You smear cool mud over your clothes and skin. Your scent dulls.")
		g.spend(1000)
	case "ns_incense":
		it := g.nsIncense()
		if it == nil {
			return true
		}
		g.consume(it)
		st.Incense = g.Turn + 90
		g.Msg("You burn incense; its sweet smoke hides your scent for a while.")
		g.spend(1000)
	default:
		return false
	}
	return true
}

// ---- display ----

func nsSide(g *Game) []string {
	if g.Cur == nil || g.OnWorld {
		return nil
	}
	st := g.ns()
	nc := map[string]string{"quiet": "G", "muted": "Y", "loud": "R"}[nsNoiseLabel(st.Noise)]
	sc := map[string]string{"faint": "G", "strong": "Y", "reeking": "R"}[nsScentLabel(st.Scent)]
	return []string{fmt.Sprintf("Noise: {{%s|%s}}  Scent: {{%s|%s}}", nc, nsNoiseLabel(st.Noise), sc, nsScentLabel(st.Scent))}
}

// NsLines: tooltip-friendly breakdown of noise, scent, wind and who is following you.
func (g *Game) NsLines() []string {
	if g.Cur == nil || g.P == nil {
		return nil
	}
	st := g.ns()
	out := []string{fmt.Sprintf("Noise: %s (%d)", nsNoiseLabel(st.Noise), st.Noise)}
	for _, w := range st.Why {
		out = append(out, "  "+w)
	}
	out = append(out, fmt.Sprintf("Scent: %s (%d)", nsScentLabel(st.Scent), st.Scent))
	for _, w := range st.SWhy {
		out = append(out, "  "+w)
	}
	dx, dy, sp := g.nsWind()
	out = append(out, "Wind: "+nsWindText(dx, dy, sp))
	n := 0
	for id, tg := range st.Target {
		if tg.Until > g.Turn {
			for _, a := range g.Cur.Actors {
				if a.ID == id && a.HP > 0 {
					n++
				}
			}
		}
	}
	if n > 0 {
		out = append(out, fmt.Sprintf("%d creature(s) are coming to investigate", n))
	}
	return out
}
