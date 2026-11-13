//go:build js && wasm

// WASM bridge: the page hands over a Realmforge world file, then streams key presses; every call returns a frame.
package main

import (
	"encoding/json"
	"syscall/js"

	"wanderer/engine"
)

var (
	world *engine.WorldData
	game  *engine.Game
)

func frame(out engine.Out) any {
	s := game.Render()
	fg := make([]byte, len(s.F))
	bg := make([]byte, len(s.B))
	for i := range s.F {
		fg[i] = 'A' + s.F[i]
		bg[i] = 'A' + s.B[i]
	}
	fr := map[string]any{
		"w": engine.SW, "h": engine.SH, "g": string(s.G), "f": string(fg), "b": string(bg),
		"save": out.Save, "exit": out.Exit, "dead": game.Dead, "mode": game.Mode,
		"music": musicJS(game.MusicState()), "actors": actorsJS(game.SpriteActors()),
	}
	if pi := game.Portrait(); pi != nil { // speaker or paper doll rectangle (engine/portraits.go)
		fr["portrait"] = map[string]any{"x": pi.X, "y": pi.Y, "w": pi.W, "h": pi.H, "kind": pi.Kind, "actor": actorsJS([]engine.SprActor{pi.Actor}).([]any)[0]}
	}
	return fr
}

func actorsJS(list []engine.SprActor) any {
	out := make([]any, 0, len(list))
	for _, a := range list {
		m := map[string]any{"x": a.X, "y": a.Y, "i": a.ID}
		for k, v := range map[string]string{"s": a.Species, "c": a.Calling, "r": a.Role, "t": a.Tpl, "b": a.Body, "gr": a.Group, "w": a.Weapon, "k": a.Caste, "hd": a.Head, "bk": a.Back, "ft": a.Feet, "hn": a.Hands, "bi": a.Torso} {
			if v != "" {
				m[k] = v
			}
		}
		if a.Size > 0 {
			m["z"] = a.Size
		}
		if a.Armor > 0 {
			m["a"] = a.Armor
		}
		if a.Player {
			m["p"] = true
		}
		if a.Gear {
			m["ge"] = true
		}
		if a.Shield {
			m["sh"] = true
		}
		out = append(out, m)
	}
	return out
}

func musicJS(m engine.Music) any {
	return map[string]any{"mood": m.Mood, "danger": m.Danger, "genre": m.Genre, "biome": m.Biome, "depth": m.Depth, "day": m.Day, "hp": m.HP, "seed": m.Seed,
		"wx": m.Wx, "season": m.Season, "bolt": m.Bolt, "sphere": m.Sphere, "feast": m.Feast, "combat": m.Combat, "planet": m.Planet, "doom": m.Doom}
}

func errObj(err error) any { return map[string]any{"error": err.Error()} }

func main() {
	api := map[string]any{}
	api["palette"] = js.FuncOf(func(this js.Value, a []js.Value) any {
		b, _ := json.Marshal(engine.Palette)
		return string(b)
	})
	api["init"] = js.FuncOf(func(this js.Value, a []js.Value) any {
		w, err := engine.ParseWorld(a[0].String())
		if err != nil {
			return errObj(err)
		}
		world = w
		return engine.CharOptions(w)
	})
	api["newGame"] = js.FuncOf(func(this js.Value, a []js.Value) any {
		var spec engine.CharSpec
		if err := json.Unmarshal([]byte(a[0].String()), &spec); err != nil {
			return errObj(err)
		}
		game = engine.NewGame(world, spec)
		return frame(engine.Out{})
	})
	api["load"] = js.FuncOf(func(this js.Value, a []js.Value) any {
		g, err := engine.LoadGame(world, a[0].String())
		if err != nil {
			return errObj(err)
		}
		game = g
		return frame(engine.Out{})
	})
	api["key"] = js.FuncOf(func(this js.Value, a []js.Value) any {
		if game == nil {
			return nil
		}
		return frame(game.Key(a[0].String()))
	})
	api["frame"] = js.FuncOf(func(this js.Value, a []js.Value) any {
		if game == nil {
			return nil
		}
		return frame(engine.Out{})
	})
	api["save"] = js.FuncOf(func(this js.Value, a []js.Value) any {
		if game == nil {
			return ""
		}
		return game.Save()
	})
	api["meta"] = js.FuncOf(func(this js.Value, a []js.Value) any {
		if game == nil {
			return "{}"
		}
		return game.Meta()
	})
	api["pref"] = js.FuncOf(func(this js.Value, a []js.Value) any {
		if game != nil {
			game.SetPref(a[0].String(), a[1].Bool())
		}
		return nil
	})
	api["deeds"] = js.FuncOf(func(this js.Value, a []js.Value) any {
		if game == nil {
			return "[]"
		}
		return game.PendingDeeds()
	})
	api["hall"] = js.FuncOf(func(this js.Value, a []js.Value) any {
		if game == nil {
			return "{}"
		}
		return game.HFSummary()
	})
	api["legends"] = js.FuncOf(func(this js.Value, a []js.Value) any {
		var l []engine.HFLegend
		if game != nil && json.Unmarshal([]byte(a[0].String()), &l) == nil {
			game.HFSetLegends(l)
		}
		return nil
	})
	js.Global().Set("qud", js.ValueOf(api))
	js.Global().Call("dispatchEvent", js.Global().Get("Event").New("qud-ready"))
	select {}
}
