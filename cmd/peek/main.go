// Peek renders the screen as plain text after scripted actions: go run ./cmd/peek world.json <script>
// Script words: talk (walk to nearest friendly NPC and bump), down (find > and descend), key:<k>, show.
package main

import (
	"fmt"
	"os"
	"strings"

	"wanderer/engine"
)

func dump(g *engine.Game) {
	s := g.Render()
	for y := 0; y < engine.SH; y++ {
		fmt.Println(strings.TrimRight(string(s.G[y*engine.SW:(y+1)*engine.SW]), " "))
	}
	fmt.Println(strings.Repeat("=", engine.SW))
}

func main() {
	b, _ := os.ReadFile(os.Args[1])
	w, err := engine.ParseWorld(string(b))
	if err != nil {
		panic(err)
	}
	g := engine.NewGame(w, engine.CharSpec{Name: "Peek", Species: "dwarf", Calling: "scholar", Attr: [6]int{14, 14, 14, 16, 14, 14}, Powers: map[string]int{"flames": 1}})
	for _, cmd := range os.Args[2:] {
		switch {
		case cmd == "show":
			dump(g)
		case cmd == "talk":
			g.DebugTalkNearest()
		case cmd == "down":
			g.DebugGotoStairs()
			g.Key(">")
		case strings.HasPrefix(cmd, "key:"):
			g.Key(cmd[4:])
		}
	}
}
