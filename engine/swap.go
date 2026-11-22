package engine

// swap.go: peaceful people never wall the player in. Bumping a non-hostile person normally opens a talk, but
//   - when they stand in a one-tile passage (the cell beside them on both sides is blocked) the player squeezes past, and
//   - bumping the same person twice within a few turns swaps places instead of talking again.
// Hostile actors, bosses, companions (handled by MoveP) and anyone you cannot swap with (sleeping, stuck) are untouched.

// swapState is not saved: it only remembers the last bump.
type swapState struct {
	id   int
	turn int
}

var swapMem = map[*Game]*swapState{}

func (g *Game) swapMemory() *swapState {
	s := swapMem[g]
	if s == nil {
		s = &swapState{id: -1}
		swapMem[g] = s
	}
	return s
}

// npcSwap tries to trade places with a at (nx, ny); true when it did (the move is spent).
func (g *Game) npcSwap(a *Actor, dx, dy, nx, ny int) bool {
	p := g.P
	if a == nil || p == nil || a == p || a.HP <= 0 || a.Boss || g.Hostile(a) || a.St("frozen") > 0 || a.St("stunned") > 0 {
		return false
	}
	z := g.Cur
	m := g.swapMemory()
	again := m.id == a.ID && g.Turn-m.turn <= 6
	m.id, m.turn = a.ID, g.Turn
	// a one-tile passage: both cells beside them are closed and the way on is open
	blocked := false
	if dx != 0 || dy != 0 {
		sx, sy := -dy, dx
		side1 := !z.In(nx+sx, ny+sy) || !g.canEnter(p, nx+sx, ny+sy)
		side2 := !z.In(nx-sx, ny-sy) || !g.canEnter(p, nx-sx, ny-sy)
		diag := dx != 0 && dy != 0
		if diag {
			side1 = !g.canEnter(p, nx-dx, ny) && !g.canEnter(p, nx, ny-dy)
			side2 = true
		}
		beyond := z.In(nx+dx, ny+dy) && g.canEnter(p, nx+dx, ny+dy)
		blocked = side1 && side2 && beyond
	}
	if !again && !blocked {
		return false
	}
	a.X, a.Y, p.X, p.Y = p.X, p.Y, nx, ny
	g.Msg("{{K|You squeeze past " + g.The(a) + ".}}")
	g.spend(1000)
	m.id = -1
	return true
}
