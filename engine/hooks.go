package engine

import "encoding/json"

// Plug-in points: feature files register here from init() instead of editing the core loop,
// input or renderer. Each registry is called from exactly one place (noted beside it).
var (
	modeKeys   = map[string]func(g *Game, k string){} // Key(): keys while g.Mode == name
	modeDraws  = map[string]func(g *Game){}           // Render(): draw over the map while g.Mode == name
	mainKeys   = map[string]func(g *Game){}           // keyMain(): extra keys on the main screen
	talkHooks  []func(g *Game, a *Actor) []Option     // talkMenu(): extra dialogue options
	turnHooks  []func(g *Game)                        // turn(): once per game minute
	zoneHooks  []func(g *Game, zn *Zone, r *Rng)      // genZone(): after a zone is generated
	killHooks  []func(g *Game, d, src *Actor)         // kill(): after a creature dies
	enterHooks []func(g *Game, zn *Zone)              // enterZone(): after the player arrives
	sideHooks  []func(g *Game) []string               // drawSide(): extra sidebar lines
)

// ext returns a feature's saved state, creating it with mk on first use. State is held live in
// g.extLive and serialised into g.Store by Save, so features never touch the Game struct.
func (g *Game) ext(key string, mk func() any) any {
	if g.extLive == nil {
		g.extLive = map[string]any{}
	}
	if v, ok := g.extLive[key]; ok {
		return v
	}
	v := mk()
	if raw, ok := g.Store[key]; ok {
		_ = json.Unmarshal(raw, v)
	}
	g.extLive[key] = v
	return v
}

func (g *Game) flushExt() {
	if len(g.extLive) == 0 {
		return
	}
	if g.Store == nil {
		g.Store = map[string]json.RawMessage{}
	}
	for k, v := range g.extLive {
		if b, err := json.Marshal(v); err == nil {
			g.Store[k] = b
		}
	}
}

// Pref reads a host setting (animations, hints, ...); unset means on.
func (g *Game) Pref(k string) bool {
	if g.Prefs == nil {
		return true
	}
	v, ok := g.Prefs[k]
	return !ok || v
}

func (g *Game) SetPref(k string, v bool) {
	if g.Prefs == nil {
		g.Prefs = map[string]bool{}
	}
	g.Prefs[k] = v
}
