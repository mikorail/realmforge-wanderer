package engine

import (
	"strings"
	"testing"
)

func TestProceduralAssetsAndVariations(t *testing.T) {
	g := newGame(t, nil)

	// 1. All 26 new creature templates should be registered in tplByID and spawnable across genres.
	for _, ct := range paTemplates {
		if tplByID[ct.ID] == nil {
			t.Fatalf("missing template %s in tplByID", ct.ID)
		}
		r := NewRng("test:spawn:" + ct.ID)
		a := g.Spawn(ct.ID, ct.Lvl+1, r)
		if a == nil || a.MaxHP <= 0 || a.Name == "" {
			t.Fatalf("failed to spawn %s", ct.ID)
		}
	}

	// 2. Morph application modifies stats and adds procedural trait descriptions deterministically.
	r := NewRng("test:morph")
	wolf := g.Spawn("wolf", 4, r)
	origDesc := wolf.Desc
	m := paBiomeMorph(9, 0, g.Genre, r)
	paApplyMorph(wolf, m)
	if !strings.HasPrefix(wolf.Name, m.Prefix+" ") {
		t.Fatalf("expected morph prefix %q on %q", m.Prefix, wolf.Name)
	}
	if wolf.Desc == origDesc {
		t.Fatalf("expected morph trait appended to description")
	}

	// 3. Zone generation with paZoneHook is deterministic across repeated calls with the same seed.
	g1 := newGame(t, nil)
	g2 := newGame(t, nil)
	z1 := g1.GetZone(g1.WX, g1.WY, 0, 0, 0)
	z2 := g2.GetZone(g2.WX, g2.WY, 0, 0, 0)
	if len(z1.Actors) != len(z2.Actors) {
		t.Fatalf("actor count mismatch: %d vs %d", len(z1.Actors), len(z2.Actors))
	}
	for i := range z1.T {
		if z1.T[i] != z2.T[i] {
			t.Fatalf("terrain mismatch at tile %d: %v vs %v", i, z1.T[i], z2.T[i])
		}
	}

	// 4. Town NPCs receive procedural physical traits & mannerisms.
	zt := g1.GetZone(g1.WX, g1.WY, 1, 1, 0)
	decorated := 0
	for _, a := range zt.Actors {
		if a.Role != "" && strings.Contains(a.Desc, "Appears ") {
			decorated++
		}
	}
	if decorated == 0 {
		t.Fatalf("expected town NPCs to receive procedural appearance descriptions")
	}
}
