package engine

import (
	"fmt"
	"sort"
)

// Companions using their own active powers in combat (prefix cp). Powers come from installed
// augments (AugDef.Power) and from natural powers / mutations in Actor.Powers. Each power has a
// simple usage rule (heal when hurt, flames on a cluster that spares allies, sprint/phase to flee,
// a force bubble when surrounded, …) and a cooldown kept in the companion's own Actor.Cool,
// ticked down once a game minute by cpTick. Hook: one line at the top of companionAct.

// cpPower is one usable power of a companion: its id, display name and level.
type cpPower struct {
	ID   string
	Name string
	Lv   int
}

// cpPowers lists a companion's active powers that have a usage rule, sorted by id.
func (g *Game) cpPowers(a *Actor) []cpPower {
	seen := map[string]bool{}
	var out []cpPower
	add := func(id, name string) {
		pd := powerByID[id]
		if seen[id] || pd == nil || !pd.Active || !cpUsable[id] {
			return
		}
		seen[id] = true
		out = append(out, cpPower{ID: id, Name: name, Lv: maxi(a.Pow(id), 1) + a.Level/4})
	}
	for _, p := range a.Body {
		if d := augOn(p); d != nil && d.Power != "" {
			add(d.Power, p.Aug.Name)
		}
	}
	ids := make([]string, 0, len(a.Powers))
	for id, lv := range a.Powers {
		if lv > 0 {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	for _, id := range ids {
		add(id, g.Sk.PowerName(id))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

var cpUsable = map[string]bool{"heal": true, "flames": true, "freeze": true, "lightning": true, "arcanebolt": true, "leech": true,
	"syphon": true, "phase": true, "sprint": true, "forcebubble": true, "stunforce": true}

func (g *Game) cpReady(a *Actor, id string) bool { return a.Cool == nil || a.Cool[id] <= 0 }

func (g *Game) cpSetCool(a *Actor, pw cpPower) {
	if a.Cool == nil {
		a.Cool = map[string]int{}
	}
	cool := 40
	if pd := powerByID[pw.ID]; pd != nil && pd.Cool > 0 {
		cool = pd.Cool
	}
	a.Cool[pw.ID] = maxi(8, cool-pw.Lv*2)
}

// cpTick: once a game minute, companions' power cooldowns recharge.
func (g *Game) cpTick() {
	if g.Cur == nil {
		return
	}
	for _, a := range g.Cur.Actors {
		if a.AI != "follower" || a.HP <= 0 {
			continue
		}
		for k, v := range a.Cool {
			if v > 0 {
				a.Cool[k] = v - 1
			}
		}
	}
}

// cpFoes: enemies of a companion within r of it.
func (g *Game) cpFoes(a *Actor, r int) []*Actor {
	var out []*Actor
	for _, o := range g.Cur.Actors {
		if g.compEnemy(a, o) && dist(a, o) <= r {
			out = append(out, o)
		}
	}
	return out
}

// cpSafeBlast: no one but enemies (and no burnable town) within rad of the impact.
func (g *Game) cpSafeBlast(a *Actor, x, y, rad int) bool {
	if maxi(abs(g.P.X-x), abs(g.P.Y-y)) <= rad {
		return false
	}
	for _, o := range g.Cur.Actors {
		if o.HP > 0 && maxi(abs(o.X-x), abs(o.Y-y)) <= rad && !g.compEnemy(a, o) {
			return false
		}
	}
	return true
}

// cpClearLine: the line from a to t reaches t before anyone else.
func (g *Game) cpClearLine(a, t *Actor) bool {
	for i, pt := range line(a.X, a.Y, t.X, t.Y) {
		if i == 0 {
			continue
		}
		if pt[0] == t.X && pt[1] == t.Y {
			return true
		}
		if g.Cur.Opaque(pt[0], pt[1]) || pt[0] == g.P.X && pt[1] == g.P.Y || g.Cur.ActorAt(pt[0], pt[1]) != nil {
			return false
		}
	}
	return true
}

func (g *Game) cpSay(a *Actor, text string) {
	if g.visible(a.X, a.Y) || dist(a, g.P) <= 8 {
		g.Msg(text)
	}
}

// cpCast: a companion's turn spent on a power, if one fits the moment. Hook at the top of companionAct.
func (g *Game) cpCast(a *Actor) bool {
	if a.Order == OrdStay && len(g.cpFoes(a, 2)) == 0 {
		return false
	}
	pws := g.cpPowers(a)
	if len(pws) == 0 {
		return false
	}
	foes := g.cpFoes(a, 12)
	adj := 0
	for _, o := range foes {
		if dist(a, o) <= 1 {
			adj++
		}
	}
	hurt := a.HP*100 < a.MaxHP*40
	fleeing := a.Order == OrdPassive || a.HP*100 < a.MaxHP*25
	for _, pw := range pws {
		if !g.cpReady(a, pw.ID) {
			continue
		}
		if g.cpUse(a, pw, foes, adj, hurt, fleeing) {
			g.cpSetCool(a, pw)
			return true
		}
	}
	return false
}

func (g *Game) cpUse(a *Actor, pw cpPower, foes []*Actor, adj int, hurt, fleeing bool) bool {
	lv := pw.Lv
	switch pw.ID {
	case "heal":
		p := g.P
		switch {
		case hurt || a.HP < a.MaxHP && len(foes) == 0 && a.HP*100 < a.MaxHP*60:
			h := g.R.Dice(lv, 8) + 4
			a.HP = mini(a.MaxHP, a.HP+h)
			delete(a.Status, "bleeding")
			delete(a.Status, "poisoned")
			g.cpSay(a, fmt.Sprintf("{{G|%s calls on the %s and mends %d.}}", a.Name, pw.Name, h))
			return true
		case p.HP*100 < p.MaxHP*35 && dist(a, p) <= 1:
			h := g.R.Dice(lv, 8) + 4
			p.HP = mini(p.MaxHP, p.HP+h)
			delete(p.Status, "bleeding")
			g.Msg(fmt.Sprintf("{{G|%s lays the %s on you: you heal %d.}}", a.Name, pw.Name, h))
			return true
		}
	case "flames", "freeze", "lightning", "arcanebolt":
		if fleeing && a.Order == OrdPassive {
			return false
		}
		t := g.cpBlastTarget(a, foes, 8+lv, pw.ID == "flames" || pw.ID == "freeze")
		if t == nil {
			return false
		}
		switch pw.ID {
		case "flames", "freeze":
			kind := map[string]string{"flames": "fire", "freeze": "frost"}[pw.ID]
			g.cpSay(a, fmt.Sprintf("{{R|%s looses the %s at %s!}}", a.Name, pw.Name, g.the(t)))
			g.explode(t.X, t.Y, kind, lv, a)
		case "lightning":
			g.cpSay(a, fmt.Sprintf("{{C|Lightning leaps from %s's %s!}}", a.Name, pw.Name))
			hit := map[*Actor]bool{}
			cur := t
			for k := 0; k < 2+lv/2 && cur != nil; k++ {
				hit[cur] = true
				g.hurt(cur, g.R.Dice(2+lv/2, 6), a, "lightning")
				var next *Actor
				for _, o := range g.Cur.Actors {
					if o.HP > 0 && !hit[o] && dist(o, cur) <= 4 && g.compEnemy(a, o) {
						next = o
						break
					}
				}
				cur = next
			}
		default:
			dmg := g.R.Dice(1+lv/2, 6) + lv
			g.cpSay(a, fmt.Sprintf("{{C|%s's %s strikes %s for %d.}}", a.Name, pw.Name, g.the(t), dmg))
			g.hurt(t, dmg, a, "a force bolt")
		}
		return true
	case "leech", "syphon":
		if a.HP*100 >= a.MaxHP*70 {
			return false
		}
		for _, t := range foes {
			if dist(a, t) <= 3 {
				dmg := g.R.Dice(1+lv/2, 6)
				if pw.ID == "syphon" {
					dmg = g.R.Dice(2+lv, 6)
				}
				g.cpSay(a, fmt.Sprintf("{{R|%s drains %d from %s.}}", a.Name, dmg, g.the(t)))
				g.hurt(t, dmg, a, "life drain")
				a.HP = mini(a.MaxHP, a.HP+dmg)
				return true
			}
		}
	case "phase", "sprint":
		if !fleeing || adj == 0 || a.St(cpStatus[pw.ID]) > 0 {
			return false
		}
		if pw.ID == "phase" {
			a.AddSt("phased", 6+lv*2)
			g.cpSay(a, fmt.Sprintf("{{M|%s flickers out of sync with the world (%s).}}", a.Name, pw.Name))
		} else {
			a.AddSt("sprint", 5+lv*2)
			g.cpSay(a, fmt.Sprintf("{{C|%s bolts clear on the %s.}}", a.Name, pw.Name))
		}
		g.cpFleeStep(a, foes)
		return true
	case "forcebubble":
		if a.St("shielded") > 0 || !(adj >= 2 || adj >= 1 && hurt) {
			return false
		}
		a.AddSt("shielded", 15+lv*5)
		g.cpSay(a, fmt.Sprintf("{{C|A bubble of force snaps up around %s.}}", a.Name))
		return true
	case "stunforce":
		if adj < 2 || dist(a, g.P) <= 2+lv/3 {
			return false
		}
		n := 0
		for _, o := range foes {
			if dist(a, o) <= 2+lv/3 {
				o.AddSt("stunned", 3+lv/2)
				n++
			}
		}
		g.cpSay(a, fmt.Sprintf("{{C|%s's %s stuns %d foes.}}", a.Name, pw.Name, n))
		return n > 0
	}
	return false
}

var cpStatus = map[string]string{"phase": "phased", "sprint": "sprint"}

// cpBlastTarget picks the foe whose surroundings hold the most foes, in range and line of fire,
// whose blast would spare the player, companions and bystanders.
func (g *Game) cpBlastTarget(a *Actor, foes []*Actor, rng int, blast bool) *Actor {
	var best *Actor
	bs := -1
	for _, t := range foes {
		d := dist(a, t)
		if d > rng || d < 2 && blast || !g.cpClearLine(a, t) {
			continue
		}
		if blast && !g.cpSafeBlast(a, t.X, t.Y, 1) {
			continue
		}
		s := 0
		for _, o := range foes {
			if dist(o, t) <= 1 {
				s++
			}
		}
		if s > bs || s == bs && d < dist(a, best) {
			best, bs = t, s
		}
	}
	return best
}

func (g *Game) cpFleeStep(a *Actor, foes []*Actor) {
	if len(foes) == 0 {
		return
	}
	t := foes[0]
	for _, o := range foes {
		if dist(a, o) < dist(a, t) {
			t = o
		}
	}
	g.stepToward(a, g.distMap(t.X, t.Y, a.Has("flying")), true)
}

func init() {
	turnHooks = append(turnHooks, func(g *Game) { g.cpTick() })
}
