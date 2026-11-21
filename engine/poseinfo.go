package engine

// PiSleepers lists the ids of drawn actors that are asleep, so the shell can lay them down
// (src/poses.ts setSleepers). Same visibility rules as SpriteActors.
func (g *Game) PiSleepers() []int {
	var out []int
	for _, s := range g.SpriteActors() {
		if s.Player {
			continue
		}
		for _, a := range g.Cur.Actors {
			if a.ID == s.ID && a.St("asleep") > 0 {
				out = append(out, a.ID)
				break
			}
		}
	}
	return out
}
