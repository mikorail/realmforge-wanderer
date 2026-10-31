package engine

// Game modes (prefix qc): classic (permadeath), roleplay (checkpoint in towns, restore on death),
// wander (no kill XP, indifferent wildlife, discovery and quest XP doubled).

type qcModeState struct {
	Check     string `json:"check,omitempty"` // Save() taken on entering a town
	CheckKey  string `json:"ck,omitempty"`
	CheckAt   int    `json:"ca,omitempty"`
	CheckName string `json:"cn,omitempty"`
	Wander    bool   `json:"wander,omitempty"` // wander setup done
}

func (g *Game) qcMode() *qcModeState {
	return g.ext("mode", func() any { return &qcModeState{} }).(*qcModeState)
}

func init() {
	enterHooks = append(enterHooks, qcModeEnter)
	modeKeys["qcrestore"] = func(g *Game, k string) {
		switch k {
		case "Enter", "y", "Y":
			g.qcRestore()
		case "Escape", "n", "N":
			g.Mode = "dead"
		}
	}
	modeDraws["qcrestore"] = func(g *Game) {
		s := g.sc
		lines := append(Wrap(g.DeathText, 56), "", "Roleplay mode: you may wake at the last hearth you passed.", "",
			"{{W|[Enter] restore "+g.qcMode().CheckName+"}}   {{K|[Esc] accept death}}")
		h := len(lines) + 4
		y0 := (MH - h) / 2
		s.Box(8, y0, 64, h, CRed, "You died")
		for i, l := range lines {
			s.TextB(10, y0+2+i, l, CGray, CBgDark, 71)
		}
	}
}

func qcModeEnter(g *Game, zn *Zone) {
	m := g.qcMode()
	switch g.GameMode {
	case "wander":
		if !m.Wander {
			m.Wander = true
			for f := range creatureFactions {
				g.Rep[f] = 0
			}
			g.Rep["outlaws"] = 0
		}
	case "roleplay":
		if zn.Kind == "town" && zn.Burg > 0 && (zn.Key != m.CheckKey || g.Turn-m.CheckAt > 600) {
			g.qcCheckpoint()
		}
	}
}

// qcCheckpoint stores a full save as the roleplay restore point.
func (g *Game) qcCheckpoint() {
	m := g.qcMode()
	m.Check = ""
	s := g.Save()
	if s == "" {
		return
	}
	m.Check, m.CheckKey, m.CheckAt, m.CheckName = s, g.Cur.Key, g.Turn, g.Cur.Name
	g.Msg("{{K|(Checkpoint: " + g.Cur.Name + ")}}")
}

// qcOnDeath is called at the end of die(): in roleplay mode with a checkpoint it offers the
// restore screen. It returns true when it took over the death screen.
func (g *Game) qcOnDeath() bool {
	if g.GameMode != "roleplay" || g.qcMode().Check == "" {
		return false
	}
	g.Mode = "qcrestore"
	return true
}

func (g *Game) qcRestore() {
	m := g.qcMode()
	cp, name := m.Check, m.CheckName
	ng, err := LoadGame(g.W, cp)
	if err != nil {
		g.Mode = "dead"
		return
	}
	prefs := g.Prefs
	*g = *ng
	g.Prefs = prefs
	nm := g.qcMode()
	nm.Check, nm.CheckKey, nm.CheckName, nm.CheckAt = cp, g.Cur.Key, name, g.Turn
	g.Mode = "main"
	g.recompute()
	g.Msg("{{C|You wake at the last hearth you passed: " + name + ".}}")
}

// qcKillXP: kill XP is nothing in wander mode.
func (g *Game) qcKillXP(n int) int {
	if g.GameMode == "wander" {
		return 0
	}
	return n
}

// qcXP: discovery and quest XP doubles in wander mode (call at the top of gainXP).
func (g *Game) qcXP(n int) int {
	if g.GameMode == "wander" {
		return n * 2
	}
	return n
}
