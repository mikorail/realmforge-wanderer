package engine

import (
	"strings"
	"testing"
)

func TestProceduralFloraAndAtmosphere(t *testing.T) {
	g := newGame(t, nil)

	// 1. Verify pfMakeItem generates biome-, depth-, and genre-skinned items deterministically.
	genres := []string{"", "steampunk", "dieselpunk", "atompunk", "cyberpunk", "biopunk", "solarpunk", "mythic", "wasteland", "twilight"}
	for _, genre := range genres {
		g.Genre = genre
		for biome := 1; biome <= 16; biome++ {
			zn := &Zone{Key: "test", Biome: biome, Level: 2}
			r1 := NewRng("pf:test:" + genre)
			r2 := NewRng("pf:test:" + genre)
			it1 := pfMakeItem(g, zn, r1)
			it2 := pfMakeItem(g, zn, r2)
			if it1 == nil || it1.Name == "" || it1.Value <= 0 || it1.Lore == "" {
				t.Fatalf("invalid flora item for genre=%q biome=%d", genre, biome)
			}
			if it1.Name != it2.Name || it1.Value != it2.Value {
				t.Fatalf("non-deterministic flora item for genre=%q biome=%d: %q vs %q", genre, biome, it1.Name, it2.Name)
			}
		}
	}

	// 2. Verify non-town zones receive procedural flora/curiosity items and notes.
	g1 := newGame(t, nil)
	// find a genuinely wild zone near the start (the start town may sprawl into its neighbours)
	var zWild *Zone
	for d := 0; d < 6 && zWild == nil; d++ {
		for _, o := range [][2]int{{d, 0}, {-d, 0}, {0, d}, {0, -d}} {
			wx, wy := g1.WX+o[0], g1.WY+o[1]
			if !g1.W.In(wx, wy) {
				continue
			}
			if z := g1.GetZone(wx, wy, 0, 0, 0); z != nil && z.Kind == "surface" {
				zWild = z
				break
			}
		}
	}
	if zWild == nil {
		t.Skip("no wild surface zone near the start")
	}
	if len(zWild.Items) == 0 {
		t.Fatalf("expected procedural flora/curiosity items in wilderness zone")
	}
	if len(zWild.Notes) == 0 {
		t.Fatalf("expected procedural ambient notes in wilderness zone")
	}

	// 3. Verify pfSideHook returns a formatted atmospheric sidebar line without em dashes.
	g1.Cur = zWild
	lines := pfSideHook(g1)
	if len(lines) != 1 || !strings.HasPrefix(lines[0], "{{K|") {
		t.Fatalf("expected 1 atmospheric sidebar line, got %v", lines)
	}
	if strings.Contains(lines[0], "—") {
		t.Fatalf("sidebar line must not contain em dashes: %q", lines[0])
	}
}
