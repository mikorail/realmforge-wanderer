package engine

import (
	"reflect"
	"strings"
	"testing"
)

func lmTestZone(key string, bd *BurgData) *Zone {
	zn := newZone(key, bd.P%80, bd.P/80, 1, 1, 0)
	for i := range zn.T {
		zn.T[i] = TCobble
	}
	for x := 0; x < MW; x++ {
		zn.Set(x, 12, TRoad)
	}
	for y := 0; y < MH; y++ {
		zn.Set(40, y, TRoad)
	}
	zn.Set(5, 5, TDown)
	zn.Set(75, 20, TUp)
	return zn
}

func TestLandmarks(t *testing.T) {
	g := newGame(t, nil)
	w := g.W
	kinds := []string{"plaza", "palace", "market", "temple", "academy", "barracks", "forge", "artisans", "mines", "farms", "slums", "homes", "harbor", "arena"}
	inscr, uniques, placed := 0, 0, 0
	cities := 0
	for i := range w.Burgs {
		bd := &w.Burgs[i]
		if !bd.Alive || w.cityTier[bd.ID] < 1 {
			continue
		}
		cities++
		if cities > 12 {
			break
		}
		a, b := g.landmarkNames(bd), g.landmarkNames(bd)
		if !reflect.DeepEqual(a, b) {
			t.Fatalf("names unstable: %v vs %v", a, b)
		}
		for _, k := range kinds {
			key := ZoneKey(bd.P%80, bd.P/80, 1, 1, 0) + k
			mk := func() *Zone {
				zn := lmTestZone(key, bd)
				g.placeLandmarks(zn, NewRng("x"), bd, k)
				return zn
			}
			zn, zn2 := mk(), mk()
			if !reflect.DeepEqual(zn.T, zn2.T) || len(zn.Items) != len(zn2.Items) || zn.Name != zn2.Name {
				t.Fatalf("not deterministic in %s/%s", bd.Name, k)
			}
			if zn.At(5, 5) != TDown || zn.At(75, 20) != TUp {
				t.Fatalf("stairs overwritten")
			}
			for x := 0; x < MW; x++ {
				if zn.At(x, 12) != TRoad {
					t.Fatalf("road y=12 blocked at %d (%s/%s)", x, bd.Name, k)
				}
			}
			for y := 0; y < MH; y++ {
				if zn.At(40, y) != TRoad {
					t.Fatalf("road x=40 blocked")
				}
			}
			for y := 0; y < MH; y++ {
				for x := 0; x < MW; x++ {
					if zn.At(x, y) == TInscription {
						inscr++
						if !strings.Contains(zn.Notes[y*MW+x], "{{Y|") {
							t.Fatalf("inscription without text at %d,%d", x, y)
						}
					}
				}
			}
			for _, it := range zn.Items {
				if it.Rarity != "unique" || !it.Ident || !zn.Walkable(it.X, it.Y) {
					t.Fatalf("bad unique %+v", it)
				}
				uniques++
			}
			if strings.Contains(zn.Name, ", ") && len(zn.Notes) > 0 {
				placed++
			}
		}
	}
	if cities == 0 || inscr == 0 || uniques == 0 || placed == 0 {
		t.Fatalf("cities=%d inscr=%d uniques=%d placed=%d", cities, inscr, uniques, placed)
	}
	t.Logf("cities=%d inscriptions=%d uniques=%d", cities, inscr, uniques)
	for i := range w.Burgs {
		if w.cityTier[w.Burgs[i].ID] == 2 && w.Burgs[i].Alive {
			t.Logf("%s: %v", w.Burgs[i].Name, g.landmarkNames(&w.Burgs[i]))
			break
		}
	}
}
