package engine

// First-time tips: each fires once per save (state kept in g.ext "hints") and only while the
// host's "hints" preference is on. At most one tip appears per call.

type hintState struct {
	Shown map[string]bool `json:"shown"`
	Level int             `json:"level"`
}

func (g *Game) hintState() *hintState {
	h := g.ext("hints", func() any { return &hintState{} }).(*hintState)
	if h.Shown == nil {
		h.Shown = map[string]bool{}
	}
	return h
}

// tip shows msg once per id. It reports whether it fired.
func (g *Game) tip(id, msg string) bool {
	if !g.Pref("hints") {
		return false
	}
	h := g.hintState()
	if h.Shown[id] {
		return false
	}
	h.Shown[id] = true
	g.Msg("{{c|Tip:}} " + msg)
	return true
}

func init() {
	enterHooks = append(enterHooks, hintEnter)
	turnHooks = append(turnHooks, hintTurn)
	killHooks = append(killHooks, hintKill)
	sideHooks = append(sideHooks, hintSide)
}

func hintEnter(g *Game, zn *Zone) {
	switch {
	case zn.Sky:
		g.tip("sky", "Sky isles hang over the void. Stay off the edge, and look for stairs or a way back down.")
	case zn.Underwater || zn.Kind == "sea":
		g.tip("sea", "You can hold your breath only so long underwater. Watch the breath warning and surface in time.")
	case zn.Kind == "tower":
		g.tip("tower", "Towers climb floor by floor. Use > to go up and < to go back down; the top holds the prize.")
	case zn.Kind == "dungeon" || zn.Kind == "crypt":
		g.tip("dungeon", "Press s to search for hidden traps and doors. Locked vaults need a key found elsewhere in the dungeon.")
	case zn.Kind == "town":
		ritual := "the local ritual"
		if g.Sk != nil && g.Sk.Ritual != "" {
			ritual = "the " + g.Sk.Ritual
		}
		g.tip("town", "Walk into a townsperson to talk. Performing "+ritual+" with someone earns trust, secrets and better trade.")
	}
}

func hintKill(g *Game, d, src *Actor) {
	g.tip("kill", "Slain foes may drop loot. Stand on it and press g to pick it up.")
}

func hintSide(g *Game) []string {
	if g.OnWorld {
		g.tip("world", "This is the world map. Move the cursor to pick a destination and press Enter to travel; Esc returns.")
	}
	return nil
}

func hintTurn(g *Game) {
	if !g.Pref("hints") || g.P == nil || g.Cur == nil {
		return
	}
	h := g.hintState()
	p := g.P
	if h.Level == 0 {
		h.Level = p.Level
	}
	if p.Level > h.Level {
		h.Level = p.Level
		if g.tip("level", "You gained a level. Press c to spend skill points and attribute points on the character screen.") {
			return
		}
	}
	for _, a := range g.Cur.Actors {
		if a != p && a.HP > 0 && g.Hostile(a) && g.visible(a.X, a.Y) {
			if g.tip("hostile", "A hostile is in sight. Bump to melee, f to fire a ranged weapon, t to throw, a to use a power.") {
				return
			}
			break
		}
	}
	if p.MaxHP > 0 && p.HP*3 < p.MaxHP {
		if g.tip("lowhp", "You are badly hurt. Press R to rest, drink a tonic from your inventory (i), or retreat.") {
			return
		}
	}
	if g.Hunger >= 1200 || g.Thirst >= 900 {
		if g.tip("hungry", "Hungry or thirsty? Eat or drink from your inventory (i). Water tiles and town wells help too.") {
			return
		}
	}
	if g.encumbrance() >= 1 {
		g.tip("load", "You are overloaded. Drop heavy things from the inventory (i); a backpack raises your capacity.")
	}
}
