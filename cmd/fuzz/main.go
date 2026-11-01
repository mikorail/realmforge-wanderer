// Fuzz driver: plays random keys against an exported world to shake out panics.
// go run ./cmd/fuzz world.json [steps]
package main

import (
	"fmt"
	"os"
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
		">", "<", ">", "s", "g", "i", "a", "b", "c", "Enter", "Escape", "f", "t", "x", "j", "r", "m", "?", "e", "d", "z", "w", "Tab", "u", "k", "click:40,12", "y", "n", "T", "+", "F2", "~", "H", "F3", "C", "W", "O", "Tab", "s", "1", "2", " ", "o", "K", "V", "P", "Q", "Y", "X", "U", "D", "B", "N", "J"}
	g := engine.NewGame(w, engine.CharSpec{Name: "Fuzz", Species: "human", Calling: "warrior", Attr: [6]int{16, 16, 16, 16, 16, 16},
		Powers: map[string]int{"flames": 1, "teleport": 1, "wings": 1}, Start: 0})
	g.P.MaxHP, g.P.HP = 100000, 100000
	for i := 0; i < steps; i++ {
		k := keys[r.Int(0, len(keys)-1)]
		g.Key(k)
		g.Render()
		if i%250 == 0 {
			z := r.Int(-1, 5)
			if r.Chance(0.25) {
				z = -100 - r.Int(1, 6) // tower floors
			}
			g.Warp(r.Int(0, w.W-1), r.Int(0, w.H-1), r.Int(0, 2), r.Int(0, 2), z)
		}
		if i%1000 == 0 {
			s := g.Save()
			if _, err := engine.LoadGame(w, s); err != nil {
				panic(err)
			}
			fmt.Printf("step %d mode=%s zone=%s turn=%d save=%dKB zones=%d\n", i, g.Mode, g.CurKey, g.Turn, len(s)/1024, len(g.Zones))
		}
		if g.Dead {
			fmt.Println("died:", g.DeathText)
			g = engine.NewGame(w, engine.CharSpec{Name: "Fuzz2", Species: "elf", Calling: "scout", Attr: [6]int{16, 16, 16, 16, 16, 16}})
			g.P.MaxHP, g.P.HP = 100000, 100000
		}
	}
	fmt.Println("ok")
}
