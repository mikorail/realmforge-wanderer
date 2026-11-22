// Fuzz driver: plays random keys against an exported world to shake out panics.
// go run ./cmd/fuzz world.json [steps]
package main

import (
	"fmt"
	"os"
	"sort"
	"strconv"

	"wanderer/engine"
)

func main() {
	b, err := os.ReadFile(os.Args[1])
	if err != nil {
		panic(err)
	}
	w, err := engine.ParseWorld(string(b))
	if err != nil {
		panic(err)
	}
	steps := 20000
	if len(os.Args) > 2 {
		steps, _ = strconv.Atoi(os.Args[2])
	}
	_ = engine.CharOptions(w)
	r := engine.NewRng("fuzz")
	keys := []string{"ArrowUp", "ArrowDown", "ArrowLeft", "ArrowRight", "Home", "End", "PageUp", "PageDown", "ArrowUp", "ArrowRight", "ArrowRight",
		">", "<", ">", "s", "g", "i", "a", "b", "c", "Enter", "Escape", "f", "t", "x", "j", "r", "m", "?", "e", "d", "z", "w", "Tab", "u", "k", "click:40,12", "y", "n", "T", "+", "F2", "~", "H", "F3", "C", "W", "O", "Tab", "s", "1", "2", " ", "o", "K", "V", "P", "Q", "Y", "X", "U", "D", "B", "N", "J", "Z"}
	g := engine.NewGame(w, engine.CharSpec{Name: "Fuzz", Species: "human", Calling: "warrior", Attr: [6]int{16, 16, 16, 16, 16, 16},
		Powers: map[string]int{"flames": 1, "teleport": 1, "wings": 1}, Start: 0})
	g.P.MaxHP, g.P.HP = 100000, 100000
	for i := 0; i < steps; i++ {
		k := keys[r.Int(0, len(keys)-1)]
		// the per-key information the web shell asks for (cmd/wasm/main.go): run it all so its panics show up here
		g.SfxBegin()
		g.HitBegin()
		g.VoxBegin()
		g.CueBegin()
		g.FoleyBegin()
		g.Key(k)
		g.Render()
		g.SfxEnd()
		g.SfxScene()
		evs := g.HitEnd()
		engine.HitJS(evs)
		g.HitOrigins()
		engine.VoxJS(g.VoxEnd())
		engine.CueJS(g.CueEnd())
		g.CueScene()
		g.FoleyEnd()
		g.SpriteActors()
		g.GearInfo()
		g.StyleInfo()
		g.StatusFx()
		g.IconRowsJSON()
		g.IconStatusJSON()
		g.PiSleepers()
		g.TeleJSON()
		g.TipJSON()
		g.TipAtJSON(r.Int(0, 80), r.Int(0, 25))
		g.HereJSON()
		if sn := g.SoundInfo(); sn != nil {
			engine.SoundJS(sn)
		}
		if ms := g.MusicStyle(); ms != nil {
			engine.MusicStyleJS(ms)
		}
		if pi := g.Portrait(); pi != nil {
			g.PortraitFaceOf(pi)
		}
		if i%250 == 0 {
			z := r.Int(-1, 5)
			if r.Chance(0.25) {
				z = -100 - r.Int(1, 6) // tower floors
			}
			g.Warp(r.Int(0, w.W-1), r.Int(0, w.H-1), r.Int(0, 2), r.Int(0, 2), z)
		}
		if i%700 == 350 && g.UniversePlanets() > 1 {
			g.FuzzRift(r.Int(0, 99)) // rift travel between planets
		}
		if i%700 == 600 {
			g.FuzzShip(r.Int(0, 99)) // a ship in port, then out to sea (sailing.go)
		}
		if i%1000 == 0 {
			s := g.Save()
			if _, err := engine.LoadGame(w, s); err != nil {
				panic(err)
			}
			fmt.Printf("step %d mode=%s zone=%s turn=%d save=%dKB zones=%d planet=%d\n", i, g.Mode, g.CurKey, g.Turn, len(s)/1024, len(g.Zones), g.PlanetIndex())
			if os.Getenv("SAVESIZE") != "" { // biggest saved feature states (ext keys)
				type kv struct {
					k string
					n int
				}
				var top []kv
				for k, v := range g.Store {
					top = append(top, kv{k, len(v)})
				}
				sort.Slice(top, func(a, b int) bool { return top[a].n > top[b].n })
				for j := 0; j < len(top) && j < 8; j++ {
					fmt.Printf("    %-8s %6d KB\n", top[j].k, top[j].n/1024)
				}
			}
		}
		if g.Dead {
			fmt.Println("died:", g.DeathText)
			g = engine.NewGame(w, engine.CharSpec{Name: "Fuzz2", Species: "elf", Calling: "scout", Attr: [6]int{16, 16, 16, 16, 16, 16}})
			g.P.MaxHP, g.P.HP = 100000, 100000
		}
	}
	fmt.Println("ok")
}
