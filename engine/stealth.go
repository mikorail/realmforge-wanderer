package engine

import (
	"fmt"
	"sort"
	"strings"
)

// Stealth, tracking, assassination, disguise (prefix ss). State in g.ext("ss"), planet-local.
// Sneak is an ability (ss_sneak); hostiles roll to notice you through ssNotices (hook in aiAct).

type ssMark struct {
	X, Y int
	T    int  // turn laid
	W    int  // actor id; 0 = the player; -1 = false trail
	B    bool // blood
	Seen bool
}

type ssContract struct {
	Target int    `json:"t"`
	Name   string `json:"n"`
	Zone   string `json:"z"`
	State  int    `json:"s"`
	Reward int    `json:"r"`
	HX     int    `json:"hx"`
	HY     int    `json:"hy"`
}

type ssPend struct {
	ID  int    `json:"i"`
	At  int    `json:"a"`
	How string `json:"h"`
}

type ssLure struct{ X, Y, Until int }

type ssState struct {
	Sneak     bool                `json:"sneak,omitempty"`
	Alert     map[int]int         `json:"alert,omitempty"`
	Loud      int                 `json:"loud,omitempty"`
	Detected  int                 `json:"det,omitempty"`
	Alarm     int                 `json:"alarm,omitempty"`
	Lure      map[int]ssLure      `json:"lure,omitempty"`
	Trails    map[string][]ssMark `json:"trails,omitempty"`
	Wash      int                 `json:"wash,omitempty"`
	Rep       int                 `json:"rep,omitempty"`
	Kills     int                 `json:"kills,omitempty"`
	Contract  *ssContract         `json:"con,omitempty"`
	NextJob   int                 `json:"nj,omitempty"`
	Pend      []ssPend            `json:"pend,omitempty"`
	Susp      int                 `json:"susp,omitempty"`
	Expo      map[string]int      `json:"expo,omitempty"`
	Picked    map[int]bool        `json:"picked,omitempty"`
	CleanFor  int                 `json:"cf,omitempty"`
	Hint      string              `json:"-"`
	roll      map[int]int
	last      map[int][2]int
	rating    int
	ratingTrn int
}

func (g *Game) ss() *ssState {
	st := g.ext("ss", func() any { return &ssState{} }).(*ssState)
	if st.Alert == nil {
		st.Alert = map[int]int{}
	}
	if st.Lure == nil {
		st.Lure = map[int]ssLure{}
	}
	if st.Trails == nil {
		st.Trails = map[string][]ssMark{}
	}
	if st.Expo == nil {
		st.Expo = map[string]int{}
	}
	if st.Picked == nil {
		st.Picked = map[int]bool{}
	}
	if st.roll == nil {
		st.roll = map[int]int{}
	}
	if st.last == nil {
		st.last = map[int][2]int{}
	}
	return st
}

func init() {
	unLocalKeys = append(unLocalKeys, "ss")
	turnHooks = append(turnHooks, ssTurn)
	sideHooks = append(sideHooks, ssSide)
	talkHooks = append(talkHooks, ssTalk)
	killHooks = append(killHooks, ssKillHook)
	enterHooks = append(enterHooks, func(g *Game, zn *Zone) { g.ss().Susp = 0 })
}

// ---- rating ----

func ssMetallic(it *Item) bool {
	n := strings.ToLower(it.Name)
	for _, m := range ssMetal {
		if strings.Contains(n, m) {
			return true
		}
	}
	return false
}

func (g *Game) ssArmour() (pen int, clank bool) {
	w := 0
	for _, bp := range g.P.Body {
		if bp.Item != nil && !bp.Severed && bp.Item.Kind == "armor" {
			w += bp.Item.Weight
			if ssMetallic(bp.Item) {
				clank = true
			}
		}
	}
	return clampi(w/6, 0, 6), clank
}

// ssLight: how brightly you are lit (0 dark .. 3 open daylight).
func (g *Game) ssLight() int {
	z, p := g.Cur, g.P
	i := p.Y*MW + p.X
	lit := i >= 0 && i < len(z.Lit) && z.Lit[i] || p.LightRadius() > 0
	if z.Z == 0 && !z.Underwater {
		if g.daylight() {
			if roofed(z.At(p.X, p.Y)) {
				return 1
			}
			return 3
		}
		if lit {
			return 1
		}
		return 0
	}
	if z.Dark {
		if lit {
			return 2
		}
		return 0
	}
	return 2
}

func (g *Game) ssCover() int {
	z, p := g.Cur, g.P
	c := ssCoverTiles[z.At(p.X, p.Y)]
	if w := g.activeWeather(); w != nil && w.Kind != nil && w.Kind.Sight > 0 && !roofed(z.At(p.X, p.Y)) {
		if w.Kind.Sight <= 10 {
			c += 3
		} else {
			c += 2
		}
	}
	walls := 0
	for _, d := range dirs8 {
		if !z.At(p.X+d[0], p.Y+d[1]).D().Walk {
			walls++
		}
	}
	if walls >= 3 && g.ssLight() <= 1 {
		c++
	}
	if z.Kind == "town" {
		n := 0
		for _, a := range z.Actors {
			if a != p && a.HP > 0 && !g.Hostile(a) && abs(a.X-p.X) <= 4 && abs(a.Y-p.Y) <= 4 {
				n++
			}
		}
		if n >= 6 {
			c += 3
		} else if n >= 3 {
			c += 2
		}
	}
	return c
}

// ssMod: the flat part of the Stealth rating (rating = 2d6 + ssMod).
func (g *Game) ssMod() int {
	m := maxi(g.P.M(AGI), g.P.M(INT)-1) + 2*g.Skills["acrobatics"] + g.Skills["shortblade"]
	pen, clank := g.ssArmour()
	m -= pen
	if clank {
		m -= 2
	}
	st := g.ss()
	if st.Loud > g.Turn {
		m -= 4
	}
	m -= g.ssLight()
	m += g.ssCover()
	return m
}

// ---- noticing ----

func (g *Game) ssAwake(a *Actor) bool { return a.HP > 0 && a.St("asleep") == 0 && a.St("stunned") == 0 }

// ssNotices: false when the (hostile) actor fails to notice the sneaking player.
func (g *Game) ssNotices(a *Actor) bool {
	st := g.ss()
	if !st.Sneak || a == nil || g.Cur == nil {
		return true
	}
	if !g.ssAwake(a) {
		return false
	}
	d := dist(a, g.P)
	if st.Alert[a.ID] > g.Turn {
		if g.visible(a.X, a.Y) || d < 6 {
			st.Alert[a.ID] = g.Turn + 25
		}
		return true
	}
	if a.Angry || d <= 1 {
		g.ssSpot(a, d)
		return true
	}
	if st.roll[a.ID] > g.Turn {
		return false
	}
	st.roll[a.ID] = g.Turn + 3
	aware := a.M(INT) + a.Level/4
	if a.Role == "guard" || a.AI == "guard" {
		aware += 2
	}
	if st.Alarm > g.Turn {
		aware += 4
	}
	score := g.R.Dice(2, 6) + aware + 6 - d/2
	score += g.nsNoticeBonus(a, d) // noise and scent (senses.go)
	if score >= g.R.Dice(2, 6)+g.ssMod()+3 {
		g.ssSpot(a, d)
		return true
	}
	return false
}

func (g *Game) ssSpot(a *Actor, d int) {
	st := g.ss()
	if st.Alert[a.ID] <= g.Turn {
		g.Msg("{{R|" + g.The(a) + " notices you!}}")
	}
	st.Alert[a.ID] = g.Turn + 25
	if d <= 3 {
		st.Detected = g.Turn + 10
	}
}

func (g *Game) ssStatus() string {
	st := g.ss()
	if st.Detected > g.Turn {
		return "Detected"
	}
	for id, u := range st.Alert {
		if u > g.Turn {
			for _, a := range g.Cur.Actors {
				if a.ID == id && a.HP > 0 {
					return "Noticed"
				}
			}
		}
	}
	return "Hidden"
}

// ssStep: sneaking is slow. Hook in stepTo.
func (g *Game) ssStep(nx, ny, cost int) int {
	if g.ss().Sneak {
		return cost * 3 / 2
	}
	return cost
}

// ---- sneak attacks ----

func (g *Game) ssUnaware(d *Actor) bool {
	if d == nil || d == g.P {
		return false
	}
	if d.St("asleep") > 0 || d.St("stunned") > 0 {
		return true
	}
	st := g.ss()
	return st.Sneak && st.Alert[d.ID] <= g.Turn && !d.Angry
}

// ssSneakBonus: percent bonus damage for att hitting def. Hook in swing.
func (g *Game) ssSneakBonus(att, def *Actor) int {
	if att != g.P {
		return 0
	}
	st := g.ss()
	unaware := g.ssUnaware(def)
	st.Loud = g.Turn + 6
	if !unaware {
		return 0
	}
	pct := 100
	if ws := att.Weapons(); len(ws) > 0 && ws[0].Skill == "shortblade" {
		pct = 200 + 25*g.Skills["shortblade"]
	}
	if def.St("asleep") > 0 {
		pct += 50
	}
	g.Msg("{{W|Sneak attack!}}")
	st.Alert[def.ID] = g.Turn + 30
	return pct
}

// ---- abilities ----

func (g *Game) ssAbilities() []Ability {
	if g.Cur == nil {
		return nil
	}
	st := g.ss()
	name := "Sneak (off)"
	if st.Sneak {
		name = "Sneak (on)"
	}
	out := []Ability{{ID: "ss_sneak", Name: name, Target: "self"},
		{ID: "ss_ko", Name: "Knock out unaware foe", Target: "dir"},
		{ID: "ss_stab", Name: "Silent kill", Target: "dir"},
		{ID: "ss_track", Name: "Track", Target: "self"},
		{ID: "ss_false", Name: "Lay false trail", Target: "self"}}
	if g.P.Gold > 0 {
		out = append(out, Ability{ID: "ss_distract", Name: "Throw a coin to distract", Target: "tile"})
	}
	if g.ssBodyNear() != nil {
		out = append(out, Ability{ID: "ss_hide", Name: "Hide the body", Target: "self"})
	}
	if st.Contract != nil || g.Cur.Kind == "town" {
		out = append(out, Ability{ID: "ss_accident", Name: "Rig an accident", Target: "tile"})
	}
	if g.Cur.WaterAt(g.P.X, g.P.Y) > 0 {
		out = append(out, Ability{ID: "ss_wash", Name: "Wash away your scent", Target: "self"})
	}
	return out
}

func (g *Game) ssUse(id string, tx, ty int) {
	st := g.ss()
	p := g.P
	switch id {
	case "ss_sneak":
		st.Sneak = !st.Sneak
		if st.Sneak {
			pen, clank := g.ssArmour()
			g.Msg(fmt.Sprintf("{{K|You creep low. Stealth about %d%s%s.}}", 7+g.ssMod(), map[bool]string{true: ", armour clanks", false: ""}[clank], map[bool]string{true: ", heavy kit", false: ""}[pen > 2]))
		} else {
			g.Msg("You stand straight.")
		}
	case "ss_ko", "ss_stab":
		t := g.Cur.ActorAt(p.X+tx, p.Y+ty)
		if t == nil || t == p {
			g.Msg("Nobody there.")
			return
		}
		if !g.ssUnaware(t) {
			g.Msg("{{K|They are watching. You cannot take them unseen.}}")
			return
		}
		if id == "ss_ko" {
			t.AddSt("stunned", 80)
			t.AddSt("asleep", 80)
			g.Msg("You club " + g.the(t) + " senseless.")
			g.ssNoise(p.X, p.Y, 2)
		} else {
			g.ssKill(t, "knife")
		}
		g.spend(1000)
	case "ss_distract":
		if p.Gold < 1 {
			return
		}
		p.Gold--
		n := 0
		for _, a := range g.Cur.Actors {
			if a != p && a.HP > 0 && !a.Angry && a.AI != "follower" && g.ssAwake(a) && abs(a.X-tx) <= 10 && abs(a.Y-ty) <= 10 {
				st.Lure[a.ID] = ssLure{tx, ty, g.Turn + 25}
				n++
			}
		}
		g.Msg(fmt.Sprintf("A coin clinks at a distance. %d turn to look.", n))
		g.spend(1000)
	case "ss_hide":
		g.ssHideBody()
	case "ss_accident":
		g.ssRig(tx, ty)
	case "ss_track":
		g.ssTrack()
	case "ss_false":
		dir := dirs8[g.R.Int(0, 7)]
		x, y, n := p.X, p.Y, 0
		for i := 0; i < 8; i++ {
			x, y = x+dir[0], y+dir[1]
			if !g.Cur.In(x, y) || !g.Cur.At(x, y).D().Walk {
				break
			}
			st.addMark(g.Cur.Key, ssMark{X: x, Y: y, T: g.Turn, W: -1})
			n++
		}
		g.Msg(fmt.Sprintf("You scuff %d false tracks leading away.", n))
		g.spend(2000)
	case "ss_wash":
		st.Wash = g.Turn + 150
		kept := st.Trails[g.Cur.Key][:0]
		for _, m := range st.Trails[g.Cur.Key] {
			if m.W != 0 {
				kept = append(kept, m)
			}
		}
		st.Trails[g.Cur.Key] = kept
		g.Msg("You rinse the trail and your scent from yourself.")
		g.spend(1000)
	}
}

// ssNoise alerts awake actors within r tiles of a sound.
func (g *Game) ssNoise(x, y, r int) {
	st := g.ss()
	for _, a := range g.Cur.Actors {
		if a != g.P && g.ssAwake(a) && a.AI != "follower" && abs(a.X-x) <= r && abs(a.Y-y) <= r && g.Hostile(a) {
			st.Alert[a.ID] = g.Turn + 20
		}
	}
}

// ---- bodies ----

func (g *Game) ssBodyNear() *Item {
	if g.Cur == nil {
		return nil
	}
	for _, it := range g.Cur.Items {
		if it.Kind == "corpse" && abs(it.X-g.P.X) <= 1 && abs(it.Y-g.P.Y) <= 1 {
			return it
		}
	}
	return nil
}

func (g *Game) ssHideBody() {
	b := g.ssBodyNear()
	if b == nil {
		g.Msg("No body here.")
		return
	}
	z := g.Cur
	bx, by, bd := -1, -1, 99
	for dy := -6; dy <= 6; dy++ {
		for dx := -6; dx <= 6; dx++ {
			x, y := g.P.X+dx, g.P.Y+dy
			if !z.In(x, y) || z.ActorAt(x, y) != nil {
				continue
			}
			if ssCoverTiles[z.At(x, y)] >= 2 && z.At(x, y).D().Walk || !z.Lit[y*MW+x] && !g.daylight() && z.At(x, y).D().Walk {
				if d := maxi(abs(dx), abs(dy)); d < bd {
					bx, by, bd = x, y, d
				}
			}
		}
	}
	if bx < 0 {
		g.Msg("{{K|Nowhere to stash it nearby.}}")
		return
	}
	b.X, b.Y = bx, by
	if b.Extra == nil {
		b.Extra = map[string]int{}
	}
	b.Extra["sshid"] = 1
	g.Msg(ssHideLines[g.R.Int(0, len(ssHideLines)-1)])
	g.spend(1000 * (1 + bd))
}

func ssKillHook(g *Game, d, src *Actor) {
	st := g.ss()
	if g.Cur == nil {
		return
	}
	if (src == g.P || st.CleanFor == d.ID) && d.AI != "hostile" && d.AI != "follower" && strings.HasPrefix(d.Faction, "state:") {
		for _, it := range g.Cur.Items {
			if it.Kind == "corpse" && it.X == d.X && it.Y == d.Y && strings.HasPrefix(it.Name, d.Name) {
				if it.Extra == nil {
					it.Extra = map[string]int{}
				}
				it.Extra["ssbody"] = g.Turn + 1
			}
		}
	}
	if c := st.Contract; c != nil && c.Target == d.ID {
		pay := c.Reward
		if st.CleanFor == d.ID {
			g.Msg(fmt.Sprintf("{{G|Clean work. A purse of %d %s waits at the dead drop.}}", pay, g.Sk.Currency))
		} else {
			pay = pay * 6 / 10
			g.Msg(fmt.Sprintf("{{y|Messy, but done. The fixer sends %d %s.}}", pay, g.Sk.Currency))
			st.Rep--
		}
		g.P.Gold += pay
		st.Contract = nil
		st.Kills++
		g.deed("crime", fmt.Sprintf("%s was slain on contract by the assassin %s.", c.Name, g.P.Name), 0, 0)
	}
}

// ssKill: assassinate target by "knife", "poison", "draught" (knock-out), "accident". Returns whether the
// target was dealt with. Works across zones for a target not in view. Witnesses bring a bounty; a clean job
// builds assassin standing and blames someone else.
func (g *Game) ssKill(target *Actor, how string) bool {
	if target == nil || target.HP <= 0 || g.Cur == nil {
		return false
	}
	st := g.ss()
	local := false
	for _, a := range g.Cur.Actors {
		if a == target {
			local = true
		}
	}
	seen := false
	if local {
		p := 0.08
		switch how {
		case "knife":
			p = 0.55
		case "accident":
			p = 0.04
		}
		if st.Sneak || !g.daylight() {
			p /= 2
		}
		for _, w := range g.uwWitnesses(target) {
			if !g.ssAwake(w) || g.Hostile(w) {
				continue
			}
			q := p
			if w.AI == "guard" {
				q *= 1.6
			}
			if g.R.Chance(q) {
				seen = true
			}
		}
	}
	if how == "draught" {
		if local {
			target.AddSt("stunned", 150)
			target.AddSt("asleep", 150)
			g.Msg(g.The(target) + " slumps, deeply asleep.")
		}
		return true
	}
	var state int
	fmt.Sscanf(target.Faction, "state:%d", &state)
	if local {
		st.CleanFor = 0
		if !seen {
			st.CleanFor = target.ID
		}
		if how == "knife" {
			g.Msg("Your blade finds " + g.the(target) + " without a sound.")
		}
		target.HP = 0
		g.kill(target, nil)
		g.gainXP(g.killXP(target, g.P))
	} else {
		for _, zn := range g.Zones {
			keep := zn.Actors[:0]
			for _, a := range zn.Actors {
				if a != target {
					keep = append(keep, a)
				}
			}
			zn.Actors = keep
		}
		target.HP = 0
		st.CleanFor = target.ID
		ssKillHook(g, target, nil)
	}
	if seen {
		g.uwAddBounty(state, 250, " (assassination)")
		g.AddRep(target.Faction, -100, " for murder")
		g.Msg("{{R|Someone saw it.}}")
	} else {
		st.Rep++
		if how != "knife" && local {
			var pool []*Actor
			for _, a := range g.Cur.Actors {
				if a != g.P && a.HP > 0 && a.AI == "town" && a != target {
					pool = append(pool, a)
				}
			}
			if len(pool) > 0 {
				b := pool[g.R.Int(0, len(pool)-1)]
				g.Msg("{{K|" + fmt.Sprintf(ssBlameLines[g.R.Int(0, len(ssBlameLines)-1)], g.the(b)) + "}}")
			}
		}
		g.Msg("{{K|No one saw. The underworld takes note.}}")
	}
	return true
}

// ssRig stages an accident for a victim standing by a fixture.
func (g *Game) ssRig(tx, ty int) {
	t := g.Cur.ActorAt(tx, ty)
	if t == nil || t == g.P {
		g.Msg("Pick a target in view.")
		return
	}
	for dy := -1; dy <= 1; dy++ {
		for dx := -1; dx <= 1; dx++ {
			if f, ok := ssAccidentFeatures[g.Cur.At(t.X+dx, t.Y+dy)]; ok {
				if dist(t, g.P) > 8 || !g.losClear(g.P.X, g.P.Y, t.X, t.Y) {
					g.Msg("You cannot reach the spot from here.")
					return
				}
				if !g.roll2d6Check(g.checkMod("tinkering"), dcMedium) {
					g.Msg("{{R|Your tampering is clumsy; nothing happens.}}")
					g.spend(1000)
					return
				}
				g.Msg("You work quietly: " + f + ".")
				g.ss().Pend = append(g.ss().Pend, ssPend{t.ID, g.Turn + g.R.Int(3, 8), "accident"})
				g.spend(2000)
				return
			}
		}
	}
	g.Msg("{{K|Nothing there to tamper with.}}")
}

// ---- turn hook ----

func ssTurn(g *Game) {
	if g.Cur == nil || g.P == nil {
		return
	}
	st := g.ss()
	z := g.Cur
	for id, u := range st.Alert {
		if u+60 < g.Turn {
			delete(st.Alert, id)
			delete(st.roll, id)
		}
	}
	// pending deaths
	rest := st.Pend[:0]
	for _, pd := range st.Pend {
		if pd.At > g.Turn {
			rest = append(rest, pd)
			continue
		}
		for _, a := range z.Actors {
			if a.ID == pd.ID && a.HP > 0 {
				if pd.How == "poison" {
					g.Msg("{{K|Someone clutches their throat and falls.}}")
				} else {
					g.Msg("{{K|A terrible accident.}}")
				}
				g.ssKill(a, pd.How)
				break
			}
		}
	}
	st.Pend = rest
	// lured actors walk to the sound
	for id, l := range st.Lure {
		if l.Until < g.Turn {
			delete(st.Lure, id)
			continue
		}
		for _, a := range z.Actors {
			if a.ID == id && a.HP > 0 && !a.Angry && g.ssAwake(a) {
				if abs(a.X-l.X) <= 1 && abs(a.Y-l.Y) <= 1 {
					delete(st.Lure, id)
				} else {
					g.stepToward(a, g.distMap(l.X, l.Y, false), false)
				}
			}
		}
	}
	// body discovery
	for _, it := range z.Items {
		if it.Kind != "corpse" || it.Extra == nil || it.Extra["ssbody"] <= 0 {
			continue
		}
		rng := 7
		if it.Extra["sshid"] > 0 {
			rng = 2
		}
		for _, o := range z.Actors {
			if o == g.P || !g.ssAwake(o) || o.AI == "follower" || o.AI == "hostile" || abs(o.X-it.X) > rng || abs(o.Y-it.Y) > rng || !g.losClear(o.X, o.Y, it.X, it.Y) {
				continue
			}
			it.Extra["ssbody"] = -1
			st.Alarm = g.Turn + 80
			if g.visible(o.X, o.Y) {
				g.Msg("{{R|" + g.The(o) + " finds a body and raises the alarm!}}")
			}
			for _, gd := range z.Actors {
				if (gd.AI == "guard" || gd.Role == "guard") && gd.HP > 0 && !gd.Angry {
					st.Lure[gd.ID] = ssLure{it.X, it.Y, g.Turn + 60}
				}
			}
			break
		}
	}
	// trails
	if st.Wash <= g.Turn {
		g.ssLeavesTrail(g.P)
	}
	n := 0
	for _, a := range z.Actors {
		if a.HP > 0 && a.AI != "follower" && dist(a, g.P) < 40 && n < 10 {
			g.ssLeavesTrail(a)
			n++
		}
	}
	if g.Turn%30 == 0 {
		tr := st.Trails[z.Key]
		age := 400
		if g.activeWeather() != nil {
			age = 150
		}
		kept := tr[:0]
		for _, m := range tr {
			if g.Turn-m.T < age {
				kept = append(kept, m)
			}
		}
		if len(kept) == 0 {
			delete(st.Trails, z.Key)
		} else {
			st.Trails[z.Key] = kept
		}
		// tracks in zones left behind fade too (save size)
		for k, tr := range st.Trails {
			if k != z.Key && (len(tr) == 0 || g.Turn-tr[len(tr)-1].T > 1500) {
				delete(st.Trails, k)
			}
		}
	}
	g.ssDisguiseTick()
	st.Hint = g.ssTrailHint()
}

// ---- tracking ----

func (st *ssState) addMark(key string, m ssMark) {
	tr := append(st.Trails[key], m)
	if len(tr) > 90 {
		tr = tr[len(tr)-90:]
	}
	st.Trails[key] = tr
}

// ssLeavesTrail records a footprint (or blood) for a creature that moved; other systems may call it.
func (g *Game) ssLeavesTrail(a *Actor) {
	if a == nil || g.Cur == nil {
		return
	}
	st := g.ss()
	prev, ok := st.last[a.ID+ssB(a == g.P)*100000]
	cur := [2]int{a.X, a.Y}
	st.last[a.ID+ssB(a == g.P)*100000] = cur
	if !ok || prev == cur {
		return
	}
	bleed := a.St("bleeding") > 0 || a.MaxHP > 0 && a.HP*2 < a.MaxHP
	if !bleed && !ssSoft[g.Cur.At(prev[0], prev[1])] {
		return
	}
	w := a.ID
	if a == g.P {
		w = 0
	}
	st.addMark(g.Cur.Key, ssMark{X: prev[0], Y: prev[1], T: g.Turn, W: w, B: bleed})
}

func ssB(b bool) int {
	if b {
		return 1
	}
	return 0
}

// ssTrailAt: glyph for a revealed track on a tile (render hook).
func (g *Game) ssTrailAt(x, y int) (rune, byte, bool) {
	if g.Cur == nil {
		return 0, 0, false
	}
	for _, m := range g.ss().Trails[g.Cur.Key] {
		if m.Seen && m.X == x && m.Y == y {
			if m.B {
				return ',', CDarkRed, true
			}
			return '.', CGray, true
		}
	}
	return 0, 0, false
}

// ssFollowStep: the next trail position after the one nearest a, for trackers hunting the player.
func (g *Game) ssFollowStep(a *Actor) (int, int, bool) {
	tr := g.ss().Trails[g.Cur.Key]
	bi, bd := -1, 4
	for i, m := range tr {
		if m.W == 0 && dist(a, &Actor{X: m.X, Y: m.Y}) <= bd {
			bi, bd = i, dist(a, &Actor{X: m.X, Y: m.Y})
		}
	}
	for i := bi + 1; bi >= 0 && i < len(tr); i++ {
		if tr[i].W == 0 {
			return tr[i].X, tr[i].Y, true
		}
	}
	return 0, 0, false
}

func (g *Game) ssTrack() {
	st := g.ss()
	mod := maxi(g.checkMod("survival"), g.checkMod("perception"))
	found := 0
	roll := g.R.Dice(2, 6)
	var newest *ssMark
	tr := st.Trails[g.Cur.Key]
	for i := range tr {
		m := &tr[i]
		if m.W == 0 || abs(m.X-g.P.X) > 14 || abs(m.Y-g.P.Y) > 14 {
			continue
		}
		dc := 7 + (g.Turn-m.T)/60
		if m.W == -1 {
			dc += 3
		}
		if roll+mod >= dc {
			if !m.Seen {
				found++
			}
			m.Seen = true
			if newest == nil || m.T > newest.T {
				newest = m
			}
		}
	}
	g.spend(1000)
	if newest == nil {
		g.Msg("{{K|You search the ground and find no sign.}}")
		return
	}
	if newest.W == -1 && roll+mod >= 7+(g.Turn-newest.T)/60+6 {
		g.Msg("{{y|The tracks are too neat. A false trail.}}")
		return
	}
	g.Msg(fmt.Sprintf("You read the ground: %d marks. Following.", found))
	dm := g.distMap(newest.X, newest.Y, false)
	for i := 0; i < 20 && !g.Dead; i++ {
		if dm[g.P.Y*MW+g.P.X] <= 1 {
			break
		}
		bx, by, bv := -1, -1, dm[g.P.Y*MW+g.P.X]
		for _, d := range dirs8 {
			x, y := g.P.X+d[0], g.P.Y+d[1]
			if g.Cur.In(x, y) && dm[y*MW+x] < bv && g.canEnter(g.P, x, y) {
				bx, by, bv = x, y, dm[y*MW+x]
			}
		}
		if bx < 0 {
			break
		}
		z := g.Cur
		g.stepTo(bx, by)
		if g.Cur != z {
			return
		}
		stop := false
		for _, a := range g.Cur.Actors {
			if a != g.P && a.HP > 0 && g.Hostile(a) && g.visible(a.X, a.Y) && dist(a, g.P) < 9 {
				stop = true
			}
		}
		if stop {
			g.Msg("{{R|The trail ends in a face.}}")
			break
		}
	}
}

func (g *Game) ssTrailHint() string {
	var best *ssMark
	for _, m := range g.ss().Trails[g.Cur.Key] {
		m := m
		if m.Seen && m.W != 0 && (best == nil || m.T > best.T) {
			best = &m
		}
	}
	if best == nil {
		return ""
	}
	dx, dy := best.X-g.P.X, best.Y-g.P.Y
	s := ""
	if dy < -2 {
		s += "N"
	} else if dy > 2 {
		s += "S"
	}
	if dx < -2 {
		s += "W"
	} else if dx > 2 {
		s += "E"
	}
	if s == "" {
		s = "here"
	}
	return s
}

// ---- disguise ----

func ssStateOf(fac string) int {
	var s int
	if _, err := fmt.Sscanf(fac, "state:%d", &s); err != nil {
		return 0
	}
	return s
}

func (g *Game) ssPieces(fac string) int {
	s := ssStateOf(fac)
	if s <= 0 || g.W == nil {
		return 0
	}
	sp := g.stateSphere(s)
	n := 0
	for _, it := range g.wornAttire() {
		if sp != "" && it.Region == sp {
			n++
		}
	}
	return n
}

// ssDisguiseRating: attire match plus Persuasion/Ego.
func (g *Game) ssDisguiseRating(fac string) int {
	return 25*mini(g.ssPieces(fac), 4) + 5*(g.P.M(EGO)+2*g.Skills["persuasion"])
}

// ssDisguised: true when you pass as one of faction fac's own (gates, guards, restricted areas).
func (g *Game) ssDisguised(fac string) bool {
	if g.P == nil || ssStateOf(fac) <= 0 {
		return false
	}
	st := g.ss()
	if st.Expo[fac] > g.Turn || g.uwBounty(ssStateOf(fac)) >= 10 {
		return false
	}
	return g.ssPieces(fac) >= 2 && g.ssDisguiseRating(fac) >= 40
}

func (g *Game) ssDisguiseTick() {
	st := g.ss()
	fac := uwFac(g.uwStateHere())
	if g.uwStateHere() <= 0 || g.ssPieces(fac) < 2 {
		st.Susp = 0
		return
	}
	rise := 0
	armed := false
	if ws := g.P.Weapons(); len(ws) > 0 && ws[0].Kind == "weapon" {
		armed = true
	}
	for _, a := range g.Cur.Actors {
		if a == g.P || !g.ssAwake(a) || a.AI == "follower" || a.Faction != fac || dist(a, g.P) > 5 || !g.visible(a.X, a.Y) {
			continue
		}
		if r := g.Rep[fac]; r >= 100 || r <= -100 {
			rise += 2
		}
		if a.Talked {
			rise++
		}
		if armed {
			rise++
		}
	}
	rise -= g.ssDisguiseRating(fac) / 50
	if rise > 0 {
		st.Susp += rise
	} else if st.Susp > 0 {
		st.Susp--
	}
	if st.Susp >= 100 && st.Expo[fac] <= g.Turn {
		st.Expo[fac] = g.Turn + 720
		st.Susp = 0
		g.Msg("{{R|They see through your disguise!}}")
		for _, a := range g.Cur.Actors {
			if a.Faction == fac && a.HP > 0 {
				a.Angry = true
			}
		}
	}
}

// ---- sidebar ----

func ssSide(g *Game) []string {
	var out []string
	if g.Cur == nil {
		return nil
	}
	st := g.ss()
	if st.Sneak {
		c := map[string]string{"Hidden": "G", "Noticed": "Y", "Detected": "R"}[g.ssStatus()]
		out = append(out, fmt.Sprintf("{{%s|%s}} {{K|(stealth %d)}}", c, g.ssStatus(), 7+g.ssMod()))
	}
	if fac := uwFac(g.uwStateHere()); g.ssPieces(fac) >= 2 && g.uwStateHere() > 0 {
		if g.ssDisguised(fac) {
			out = append(out, fmt.Sprintf("{{C|Disguised}} {{K|(suspicion %d%%)}}", st.Susp))
		} else if st.Expo[fac] > g.Turn {
			out = append(out, "{{R|Disguise blown}}")
		}
	}
	if c := st.Contract; c != nil {
		out = append(out, "{{M|Contract: "+c.Name+"}}")
	}
	if st.Hint != "" {
		out = append(out, "{{K|Trail: "+st.Hint+"}}")
	}
	return out
}

// ---- talk: contracts, poison, pickpocket ----

func ssTalk(g *Game, a *Actor) []Option {
	var out []Option
	st := g.ss()
	if a.AI == "follower" || g.Hostile(a) {
		return nil
	}
	if uwIsFence(a) {
		out = append(out, Option{"{{M|Ask about quiet work.}}", func() { g.ssJobMenu(a) }},
			Option{fmt.Sprintf("{{M|Buy a sleeping draught.}} {{K|(30 %s)}}", g.Sk.Currency), func() {
				if g.P.Gold < 30 {
					g.talkMenu("\"Come back with coin.\"")
					return
				}
				g.P.Gold -= 30
				g.P.Inv = append(g.P.Inv, &Item{ID: g.newID(), Name: "sleeping draught", Kind: "scrap", Glyph: '!', F: CCyan, Weight: 1, Value: 30, Extra: map[string]int{"ssdraught": 1}})
				g.talkMenu("\"Tasteless. Strong. Do not drink it yourself.\"")
			}})
	}
	if c := st.Contract; c != nil && c.Target == a.ID {
		for _, it := range g.P.Inv {
			it := it
			if uwIsPoison(it) {
				out = append(out, Option{"{{R|Slip poison into their food.}} {{K|(" + it.Name + ")}}", func() { g.ssSpike(a, it, "poison") }})
				break
			}
		}
		for _, it := range g.P.Inv {
			it := it
			if it.Extra != nil && it.Extra["ssdraught"] > 0 {
				out = append(out, Option{"{{R|Spike their cup with the sleeping draught.}}", func() { g.ssSpike(a, it, "draught") }})
				break
			}
		}
	}
	if a.AI == "town" || a.AI == "guard" || a.AI == "wander" {
		out = append(out, Option{"{{M|Pick their pocket.}} " + g.checkLabel("streetwise", ssPocketDC(a)), func() { g.ssPocket(a) }})
	}
	return out
}

func (g *Game) ssSpike(a *Actor, it *Item, how string) {
	if it.Count > 1 {
		it.Count--
	} else {
		g.removeInv(it)
	}
	g.Mode, g.Talk = "main", nil
	if how == "draught" {
		g.ssKill(a, "draught")
		g.spend(1000)
		return
	}
	g.ss().Pend = append(g.ss().Pend, ssPend{a.ID, g.Turn + g.R.Int(3, 8), "poison"})
	g.Msg("You tip it into their meal and move away.")
	g.spend(1000)
}

func ssPocketDC(a *Actor) int { return clampi(8+a.Level/3, 8, 15) }

func (g *Game) ssPocket(a *Actor) {
	st := g.ss()
	g.Mode, g.Talk = "main", nil
	if st.Picked[a.ID] {
		g.Msg("{{K|You already lifted what was worth taking.}}")
		return
	}
	st.Picked[a.ID] = true
	g.spend(1000)
	state := ssStateOf(a.Faction)
	if !g.roll2d6Check(g.checkMod("streetwise")+g.uwStealMod()/2, ssPocketDC(a)) {
		g.Msg("{{R|" + g.The(a) + " catches your hand!}}")
		g.provoke(a)
		if g.Cur.Kind == "town" {
			g.uwAddBounty(state, 30, " (theft)")
		}
		return
	}
	if len(a.Inv) > 0 && g.R.Chance(0.5) {
		it := a.Inv[g.R.Int(0, len(a.Inv)-1)]
		got := g.uwTake(&a.Inv, it)
		g.uwMarkStolen(got, state)
		g.P.Inv = append(g.P.Inv, got)
		g.Msg("You lift " + got.Display() + ".")
	} else if a.Gold > 0 {
		n := maxi(1, a.Gold/3)
		a.Gold -= n
		g.P.Gold += n
		g.Msg(fmt.Sprintf("You palm %d %s.", n, g.Sk.Currency))
	} else {
		g.Msg("{{K|Empty pockets.}}")
		return
	}
	g.gainXP(5)
	if g.Cur.Kind == "town" && g.uwSeen(a) {
		g.Msg("{{R|Someone saw!}}")
		a.Angry = true
		g.uwAddBounty(state, 40, " (theft)")
	}
}

// ---- contracts ----

func (g *Game) ssJobMenu(f *Actor) {
	st := g.ss()
	if c := st.Contract; c != nil {
		g.talkMenu(fmt.Sprintf("\"You still owe us %s. He keeps near %s. A clean job pays in full.\"", c.Name, ssWhere(c.HX, c.HY)))
		return
	}
	if st.NextJob > g.Turn {
		g.talkMenu("\"Nothing today. Come back tomorrow.\"")
		return
	}
	r := NewRng(fmt.Sprintf("%s:ss:%s:%d", g.Seed, g.Cur.Key, g.Turn/1440))
	var pool []*Actor
	for _, a := range g.Cur.Actors {
		if a.HP > 0 && a != f && (a.Role == "notable" || strings.Contains(strings.ToLower(a.Title), "captain")) {
			pool = append(pool, a)
		}
	}
	if len(pool) == 0 {
		g.talkMenu("\"No one here worth the knife.\"")
		return
	}
	t := pool[r.Int(0, len(pool)-1)]
	c := &ssContract{Target: t.ID, Name: t.Name, Zone: g.Cur.Key, State: ssStateOf(t.Faction), Reward: 150 + t.Level*35 + 40*st.Rep, HX: t.HomeX, HY: t.HomeY}
	g.Mode, g.Talk = "main", nil
	g.uwOpenMenu("A quiet job", fmt.Sprintf("\"%s. Keeps near %s, walks the same rounds. %d %s, more if it is clean.\nPoison their food, stage an accident, or the knife. Be unseen.\"", t.Name, ssWhere(c.HX, c.HY), c.Reward, g.Sk.Currency), []Option{
		{"{{G|Take the contract.}}", func() {
			st.Contract = c
			st.NextJob = g.Turn + 1440
			g.uwClose()
			g.Msg("{{M|Contract taken: " + c.Name + ".}}")
		}},
		{"{{y|Refuse.}}", func() { g.uwClose() }}}, false)
}

func ssWhere(x, y int) string {
	v, h := "middle", "middle"
	if y < MH/3 {
		v = "north"
	} else if y > 2*MH/3 {
		v = "south"
	}
	if x < MW/3 {
		h = "west"
	} else if x > 2*MW/3 {
		h = "east"
	}
	if v == h {
		return "the middle of town"
	}
	if v == "middle" {
		return "the " + h + " side"
	}
	if h == "middle" {
		return "the " + v + " side"
	}
	return "the " + v + h
}

var _ = sort.Ints
