package engine

// Who stands where, for the web shell's sprite mode (prefix spi). The glyph grid still decides what
// is visible; this only says what each drawn actor is, so the shell can paint a species, a calling or
// role outfit, or a creature's body plan, varied per individual by `ID`. Read-only: it never draws
// g.R and changes nothing.

// SprActor is one drawn actor on the map region of the screen.
type SprActor struct {
	X       int    `json:"x"`
	Y       int    `json:"y"`
	ID      int    `json:"i"`
	Species string `json:"s,omitempty"`
	Calling string `json:"c,omitempty"` // the player's calling
	Role    string `json:"r,omitempty"`
	Tpl     string `json:"t,omitempty"`
	Body    string `json:"b,omitempty"`  // creature body plan
	Group   string `json:"gr,omitempty"` // bestiary group
	Size    int    `json:"z,omitempty"`  // bestiary size 1..7
	Player  bool   `json:"p,omitempty"`
	Weapon  string `json:"w,omitempty"` // skill of the wielded weapon
	Armor   int    `json:"a,omitempty"` // 0 none .. 3 heavy
	Caste   string `json:"k,omitempty"`
}

func sprArmor(a *Actor) int {
	switch av := a.AV(); {
	case av >= 9:
		return 3
	case av >= 6:
		return 2
	case av >= 3:
		return 1
	}
	return 0
}

func (g *Game) sprOf(a *Actor) SprActor {
	s := SprActor{X: a.X, Y: a.Y, ID: a.ID, Species: a.Species, Role: a.Role, Tpl: a.Tpl, Caste: a.Caste, Armor: sprArmor(a)}
	for _, w := range a.Weapons() {
		if w.Skill != "" && w.Skill != "natural" {
			s.Weapon = w.Skill
			break
		}
	}
	if a.Species == "" {
		if t := tplByID[a.Tpl]; t != nil {
			s.Body = t.Body
		}
		if b := bsSpecies(a.Tpl); b != nil {
			s.Group, s.Size = b.Group, b.Size
			if s.Body == "" {
				s.Body = b.Body
			}
		}
	}
	return s
}

// SpriteActors lists the actors currently drawn in the map area, and the player last.
func (g *Game) SpriteActors() []SprActor {
	if g.Cur == nil || g.Mode == "world" || g.OnWorld {
		return nil
	}
	var out []SprActor
	for _, a := range g.Cur.Actors {
		if a.HP <= 0 || !g.Cur.In(a.X, a.Y) || !g.visible(a.X, a.Y) {
			continue
		}
		out = append(out, g.sprOf(a))
	}
	p := g.sprOf(g.P)
	p.Player, p.Calling = true, g.Calling
	return append(out, p)
}
