//go:build js && wasm

// WASM bridge: the page hands over a Realmforge world file, then streams key presses; every call returns a frame.
package main

import (
	"encoding/json"
	"fmt"
	"syscall/js"

	"wanderer/engine"
)

var (
	world *engine.WorldData
	game  *engine.Game
	rp    *engine.RpPlayer // replay viewer's own game (engine/replay.go)
)

func frame(out engine.Out) any { return frameOf(game, out) }

func frameOf(g *engine.Game, out engine.Out) any {
	s := g.Render()
	fg := make([]byte, len(s.F))
	bg := make([]byte, len(s.B))
	for i := range s.F {
		fg[i] = 'A' + s.F[i]
		bg[i] = 'A' + s.B[i]
	}
	fr := map[string]any{
		"w": engine.SW, "h": engine.SH, "g": string(s.G), "f": string(fg), "b": string(bg),
		"save": out.Save, "exit": out.Exit, "dead": g.Dead, "mode": g.Mode,
		"music": musicJS(g.MusicState()), "actors": actorsJS(g.SpriteActors()), "rpHl": g.ReplayHL(),
	}
	if pi := g.Portrait(); pi != nil { // speaker or paper doll rectangle (engine/portraits.go)
		pj := map[string]any{"x": pi.X, "y": pi.Y, "w": pi.W, "h": pi.H, "kind": pi.Kind, "actor": actorsJS([]engine.SprActor{pi.Actor}).([]any)[0]}
		if pf := g.PortraitFaceOf(pi); pf != nil { // expression, age, scars (engine/portraitmood.go)
			pj["mood"] = engine.PortraitMoodJS(*pf)
		}
		fr["portrait"] = pj
	}
	if fx := g.FxInfo(); fx != nil {
		fr["fx"] = fxJS(fx)
	}
	if gi := g.GearInfo(); len(gi) > 0 { // worn and wielded gear per person (engine/gearinfo.go)
		fr["gear"] = engine.GearJS(gi)
	}
	if st := g.StyleInfo(); st != nil { // culture building style (engine/styleinfo.go)
		fr["style"] = engine.StyleJS(st)
	}
	if sf := g.StatusFx(); len(sf) > 0 { // status auras and item glows (engine/styleinfo.go)
		fr["statuses"] = engine.StatusFxJS(sf)
	}
	if s := g.IconRowsJSON(); s != "" { // item icons per list row (engine/iconinfo.go)
		fr["icons"] = s
	}
	if s := g.IconStatusJSON(); s != "" { // sidebar status icons
		fr["status"] = s
	}
	if sl := g.PiSleepers(); len(sl) > 0 { // sleeping actors for poses (engine/poseinfo.go)
		ids := make([]any, len(sl))
		for i, id := range sl {
			ids[i] = id
		}
		fr["sleepers"] = ids
	}
	return fr
}

func fxJS(f *engine.FxInfo) any {
	lights := make([]any, 0, len(f.Lights))
	for _, l := range f.Lights {
		lights = append(lights, map[string]any{"x": l.X, "y": l.Y, "r": l.R, "k": l.Kind, "fl": l.Flicker})
	}
	m := map[string]any{"wind": f.Wind, "ph": f.Phase, "dl": f.Daylight, "lights": lights}
	if f.Wx != "" {
		m["wx"], m["ov"], m["in"] = f.Wx, f.Overlay, f.Intensity
	}
	if f.Indoors {
		m["ind"] = true
	}
	if f.Dark {
		m["dk"] = true
	}
	if f.Depth > 0 {
		m["d"] = f.Depth
	}
	return m
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
		if a.Nm != "" {
			m["nm"] = a.Nm
		}
		if a.Ti != "" {
			m["ti"] = a.Ti
		}
		if a.Hp > 0 {
			m["hp"] = a.Hp
		}
		if a.Ho {
			m["ho"] = true
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

func rpFrame() any {
	f := frameOf(rp.Game(), engine.Out{}).(map[string]any)
	f["info"] = rp.Info()
	return f
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
		game.SfxBegin()
		game.HitBegin()
		fr := frame(game.Key(a[0].String()))
		if m, ok := fr.(map[string]any); ok {
			m["sfx"] = game.SfxEnd()
			m["sfxs"] = game.SfxScene()
			evs := game.HitEnd()
			m["hits"] = engine.HitJS(evs)
			ids := []int{}
			for _, e := range evs {
				if e.Attacker != 0 {
					ids = append(ids, e.Attacker)
				}
			}
			m["origins"] = engine.HitOriginsJS(game.HitOrigins(), ids) // where attackers stood (engine/styleinfo.go)
		}
		return fr
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
	api["saga"] = js.FuncOf(func(this js.Value, a []js.Value) any {
		if game == nil {
			return "{}"
		}
		return game.Saga()
	})
	api["legends"] = js.FuncOf(func(this js.Value, a []js.Value) any {
		var l []engine.HFLegend
		if game != nil && json.Unmarshal([]byte(a[0].String()), &l) == nil {
			game.HFSetLegends(l)
		}
		return nil
	})
	// replays (engine/replay.go)
	api["replay"] = js.FuncOf(func(this js.Value, a []js.Value) any {
		if game == nil {
			return ""
		}
		return game.ReplayData()
	})
	api["replayRestore"] = js.FuncOf(func(this js.Value, a []js.Value) any {
		if game == nil {
			return "no game"
		}
		if err := game.ReplayRestore(a[0].String()); err != nil {
			return err.Error()
		}
		return ""
	})
	api["replayHl"] = js.FuncOf(func(this js.Value, a []js.Value) any {
		if game == nil {
			return ""
		}
		return game.ReplayLastHl()
	})
	api["replayOpen"] = js.FuncOf(func(this js.Value, a []js.Value) any {
		if rp != nil {
			rp.Close()
			rp = nil
		}
		p, err := engine.NewReplayPlayer(a[0].String(), a[1].String())
		if err != nil {
			return errObj(err)
		}
		rp = p
		return rpFrame()
	})
	api["replayStep"] = js.FuncOf(func(this js.Value, a []js.Value) any {
		if rp == nil {
			return nil
		}
		for i := 0; i < a[0].Int() && rp.Step(); i++ {
		}
		return rpFrame()
	})
	api["replayTo"] = js.FuncOf(func(this js.Value, a []js.Value) any {
		if rp == nil {
			return nil
		}
		if err := rp.StepTo(a[0].Int()); err != nil {
			return errObj(err)
		}
		return rpFrame()
	})
	api["replayHighlight"] = js.FuncOf(func(this js.Value, a []js.Value) any {
		if rp == nil {
			return nil
		}
		rp.JumpHighlight(a[0].Int())
		return rpFrame()
	})
	api["replayRestart"] = js.FuncOf(func(this js.Value, a []js.Value) any {
		if rp == nil {
			return nil
		}
		if err := rp.Restart(); err != nil {
			return errObj(err)
		}
		return rpFrame()
	})
	api["replayClose"] = js.FuncOf(func(this js.Value, a []js.Value) any {
		if rp != nil {
			rp.Close()
			rp = nil
		}
		return nil
	})
	// right-click context menu (engine/ctxmenu.go)
	api["ctx"] = js.FuncOf(func(this js.Value, a []js.Value) any {
		if game == nil {
			return "[]"
		}
		it := game.CtxMenu(a[0].Int(), a[1].Int())
		if it == nil {
			it = []engine.CtxItem{}
		}
		b, _ := json.Marshal(it)
		return string(b)
	})
	api["ctxDo"] = js.FuncOf(func(this js.Value, a []js.Value) any {
		if game == nil {
			return nil
		}
		// sent as a key so the replay log records it
		game.SfxBegin()
		game.HitBegin()
		fr := frame(game.Key(fmt.Sprintf("ctx:%s:%d:%d", a[0].String(), a[1].Int(), a[2].Int())))
		if m, ok := fr.(map[string]any); ok {
			m["sfx"] = game.SfxEnd()
			m["sfxs"] = game.SfxScene()
			evs := game.HitEnd()
			m["hits"] = engine.HitJS(evs)
			ids := []int{}
			for _, e := range evs {
				if e.Attacker != 0 {
					ids = append(ids, e.Attacker)
				}
			}
			m["origins"] = engine.HitOriginsJS(game.HitOrigins(), ids) // where attackers stood (engine/styleinfo.go)
		}
		return fr
	})
	api["ctxAt"] = js.FuncOf(func(this js.Value, a []js.Value) any {
		if game == nil {
			return ""
		}
		x, y := game.CtxCursor()
		return fmt.Sprintf("%d,%d", x, y)
	})
	js.Global().Set("qud", js.ValueOf(api))
	js.Global().Call("dispatchEvent", js.Global().Get("Event").New("qud-ready"))
	select {}
}
