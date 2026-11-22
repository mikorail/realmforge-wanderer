package engine

import "encoding/json"

// teleinfo.go (prefix tl): the cells the shell tints as "something is about to happen". Bosses and elites that
// are winding up a phase move (foes.go `bwState.Tele`: enrage, heal, flee, resist, summon) mark their own cell;
// a summoner also marks the ring around it, where the new creatures will appear.

// TeleCell is one marked cell.
type TeleCell struct {
	X int    `json:"x"`
	Y int    `json:"y"`
	K string `json:"k,omitempty"`
}

// TeleCells lists the marked cells of the current zone that the player can see.
func (g *Game) TeleCells() []TeleCell {
	if g.Cur == nil || g.P == nil {
		return nil
	}
	st := g.bw()
	if len(st.Tele) == 0 {
		return nil
	}
	var out []TeleCell
	for _, a := range g.Cur.Actors {
		kind, ok := st.Tele[a.ID]
		if !ok || a.HP <= 0 || !g.visible(a.X, a.Y) {
			continue
		}
		out = append(out, TeleCell{X: a.X, Y: a.Y, K: kind})
		if kind == "summon" {
			for dy := -2; dy <= 2; dy++ {
				for dx := -2; dx <= 2; dx++ {
					if (dx != 0 || dy != 0) && g.canEnter(a, a.X+dx, a.Y+dy) && len(out) < 40 {
						out = append(out, TeleCell{X: a.X + dx, Y: a.Y + dy, K: kind})
					}
				}
			}
		}
	}
	return out
}

// TeleJSON is the frame field "tele" as a JSON string ("" when nothing is winding up).
func (g *Game) TeleJSON() string {
	c := g.TeleCells()
	if len(c) == 0 {
		return ""
	}
	b, err := json.Marshal(c)
	if err != nil {
		return ""
	}
	return string(b)
}
