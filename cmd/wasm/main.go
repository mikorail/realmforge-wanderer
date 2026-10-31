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
	return map[string]any{
		"w": engine.SW, "h": engine.SH, "g": string(s.G), "f": string(fg), "b": string(bg),
		"save": out.Save, "exit": out.Exit, "dead": game.Dead, "mode": game.Mode,
		"music": musicJS(game.MusicState()),
	}
}

func musicJS(m engine.Music) any {
	return map[string]any{"mood": m.Mood, "danger": m.Danger, "genre": m.Genre, "biome": m.Biome, "depth": m.Depth, "day": m.Day, "hp": m.HP, "seed": m.Seed,
		"wx": m.Wx, "season": m.Season, "bolt": m.Bolt, "sphere": m.Sphere, "feast": m.Feast, "combat": m.Combat}
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
	js.Global().Set("qud", js.ValueOf(api))
	js.Global().Call("dispatchEvent", js.Global().Get("Event").New("qud-ready"))
	select {}
}
