package engine

import (
	"sort"
	"strings"
)

// Auto-aim (prefix aa). With the "autoaim" pref on (default), f and t shoot or throw at the best
// target at once instead of opening the cursor: a hostile you can see, in range, with a clear line
// of fire (no wall and no bystander in the way). Your last target stays the pick while it is still
// a good one; otherwise the closest, then the weakest. F opens the cursor for a manual aim, and
// the cursor itself now starts on the best target and Tab cycles through clear shots first.

type aaState struct {
	Last int `json:"last,omitempty"` // actor id of the last auto-aimed target
}

func (g *Game) aaSt() *aaState { return g.ext("aa", func() any { return &aaState{} }).(*aaState) }

// aaRange is how far the pending shot can fly.
func (g *Game) aaRange(pend string) int {
	switch {
	case pend == "fire":
		if m := g.P.Missile(); m != nil {
			return m.Range + g.Skills["missile"]*2
		}
	case pend == "reach":
		if w := wpReachOf(g.P); w != nil {
			return w.Extra["reach"]
		}
	case strings.HasPrefix(pend, "throw:"), strings.HasPrefix(pend, "wpthrow:"):
		return 6 + g.P.M(STR) + g.Skills["missile"]
	}
	return 0
}

// aaClear: the shot from the player to a reaches it without hitting a wall or another creature first.
func (g *Game) aaClear(a *Actor, pend string) bool {
	rng := g.aaRange(pend)
	if rng <= 0 {
		return false
	}
	path := line(g.P.X, g.P.Y, a.X, a.Y)
	if len(path) > rng+1 {
		return false
	}
	for _, pt := range path {
		if pt[0] == a.X && pt[1] == a.Y {
			return true
		}
		if g.Cur.Opaque(pt[0], pt[1]) {
			return false
		}
		if o := g.Cur.ActorAt(pt[0], pt[1]); o != nil && o != g.P && o.HP > 0 {
			return false
		}
	}
	return true
}

// aaTargets lists the visible hostiles with a clear shot, best first.
func (g *Game) aaTargets(pend string) []*Actor {
	var out []*Actor
	for _, a := range g.Cur.Actors {
		if a.HP > 0 && g.visible(a.X, a.Y) && g.Hostile(a) && g.aaClear(a, pend) {
			out = append(out, a)
		}
	}
	last := g.aaSt().Last
	sort.SliceStable(out, func(i, j int) bool {
		if (out[i].ID == last) != (out[j].ID == last) {
			return out[i].ID == last
		}
		di, dj := dist(out[i], g.P), dist(out[j], g.P)
		if di != dj {
			return di < dj
		}
		return out[i].HP < out[j].HP
	})
	return out
}

// aaBest is the target a shot would pick, or nil.
func (g *Game) aaBest(pend string) *Actor {
	if ts := g.aaTargets(pend); len(ts) > 0 {
		return ts[0]
	}
	return nil
}

// aaQuick is the f / t key: shoot at once when auto-aim is on and a clear target exists.
// It returns false to let the caller open the manual cursor.
func (g *Game) aaQuick(pend string) bool {
	if !g.Pref("autoaim") {
		return false
	}
	t := g.aaBest(pend)
	if t == nil {
		if g.nearestTarget(-1) != nil {
			g.Msg("{{K|No clear shot: something is in the way or out of range. Aim by hand.}}")
		}
		return false
	}
	g.aaSt().Last = t.ID
	g.Pend = pend
	g.CX, g.CY = t.X, t.Y
	g.finishTarget()
	return true
}

// aaCursor puts the aim cursor on the best clear shot when a screen opens it (beginTarget hook).
func (g *Game) aaCursor(pend string) bool {
	if t := g.aaBest(pend); t != nil {
		g.CX, g.CY = t.X, t.Y
		return true
	}
	return false
}

// aaCycle moves the cursor to the next clear shot after the current one (Tab in the cursor).
func (g *Game) aaCycle(pend string) bool {
	ts := g.aaTargets(pend)
	if len(ts) == 0 {
		return false
	}
	// the list is sorted with the last target first: cycle in distance order instead
	sort.SliceStable(ts, func(i, j int) bool { return dist(ts[i], g.P) < dist(ts[j], g.P) })
	next := ts[0]
	for i, a := range ts {
		if a.X == g.CX && a.Y == g.CY {
			next = ts[(i+1)%len(ts)]
			break
		}
	}
	g.CX, g.CY = next.X, next.Y
	return true
}
