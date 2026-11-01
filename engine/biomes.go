package engine

import (
	"fmt"
	"strings"
)

// Special biomes (slimy, tarry, rusty, fungal) per parasang and terrain hazards (prefix qc).
// weather.go owns storms, sandstorms, heat and cold in the open; this file adds terrain hazards:
// lava heat, spores, crystal resonance and deep cold underground without warm clothing.

var qcBiomeIDs = [4]string{"slimy", "tarry", "rusty", "fungal"}

var qcBiomeNames = map[string][4]string{
	"":           {"Slimy marsh", "Tarry flats", "Rusted waste", "Fungal bloom"},
	"cyberpunk":  {"Coolant sump", "Tar sprawl", "Rust yards", "Bloom sprawl"},
	"biopunk":    {"Ooze beds", "Resin flats", "Corroded hide", "Spore garden"},
	"atompunk":   {"Sludge fields", "Tar pits", "Rusted fallout", "Fungal fallout"},
	"steampunk":  {"Slime fens", "Pitch flats", "Rust moor", "Mycelial fen"},
	"dieselpunk": {"Sludge pans", "Tar flats", "Rust moor", "Mould fields"},
	"solarpunk":  {"Algal fen", "Resin flats", "Rust garden", "Mycelial grove"},
	"wasteland":  {"Toxic sump", "Tar pits", "Rust flats", "Glowing mould"},
	"twilight":   {"Bile marsh", "Black pitch", "Corroded gilt", "Pale blight"},
}

func init() {
	zoneHooks = append(zoneHooks, qcBiomeZone)
	turnHooks = append(turnHooks, qcHazardTick)
	sideHooks = append(sideHooks, func(g *Game) []string {
		if g.Cur != nil && g.Cur.Z == 0 {
			if n := g.qcBiomeName(g.Cur.WX, g.Cur.WY); n != "" {
				return []string{"{{y|" + n + "}}"}
			}
		}
		return nil
	})
}

// qcBiomeAt: index 0-3 into qcBiomeIDs for a parasang, or -1. Patches are 3x3 parasangs.
func (g *Game) qcBiomeAt(wx, wy int) int {
	w := g.W
	if !w.In(wx, wy) {
		return -1
	}
	p := w.Idx(wx, wy)
	if w.IsWater(p) || w.Burg[p] > 0 {
		return -1
	}
	h := HashStr(fmt.Sprintf("%s:qcbiome:%d,%d", w.Seed, wx/3, wy/3))
	if h%100 >= 28 {
		return -1
	}
	return int(h/100) % 4
}

func (g *Game) qcBiomeName(wx, wy int) string {
	b := g.qcBiomeAt(wx, wy)
	if b < 0 {
		return ""
	}
	names, ok := qcBiomeNames[g.Genre]
	if !ok {
		names = qcBiomeNames[""]
	}
	return names[b]
}

func qcBiomeZone(g *Game, zn *Zone, r *Rng) {
	if zn.Kind != "surface" || zn.Z != 0 {
		return
	}
	b := g.qcBiomeAt(zn.WX, zn.WY)
	if b < 0 {
		return
	}
	rr := NewRng(g.W.Seed + ":qcbiomez:" + zn.Key)
	repl := map[string]Terrain{"slimy": TMoss, "tarry": TAsh, "rusty": TRubble, "fungal": TFungus}[qcBiomeIDs[b]]
	for i, t := range zn.T {
		switch t {
		case TGrass, TTallGrass, TDirt, TSand, TMoss, TSnow, TAsh:
			if rr.Chance(0.14) {
				zn.T[i] = repl
				switch qcBiomeIDs[b] {
				case "slimy":
					if rr.Chance(0.3) {
						zn.Liq[i] = 4
					}
				case "tarry":
					if rr.Chance(0.35) {
						zn.Liq[i] = 2
					}
				case "fungal":
					if rr.Chance(0.1) {
						zn.T[i] = TMushroom
					}
				}
			}
		}
	}
	for i := 0; i < rr.Int(2, 4); i++ {
		if x, y, ok := zn.FreeSpot(rr, nil); ok {
			var it *Item
			if qcBiomeIDs[b] == "rusty" || rr.Chance(0.5) {
				it = g.MakeScrap(rr, rr.Int(2, 8))
			} else {
				it = g.Loot(rr, zn.Level/3+1)
			}
			it.X, it.Y = x, y
			zn.Items = append(zn.Items, it)
		}
	}
	if x, y, ok := zn.FreeSpot(rr, nil); ok {
		it := g.Loot(rr, zn.Level/3+2)
		it.X, it.Y = x, y
		zn.Items = append(zn.Items, it)
	}
	zn.Name += " (" + g.qcBiomeName(zn.WX, zn.WY) + ")"
}

// ---- hazards ----

func qcCount(zn *Zone, x, y, rad int, f func(t Terrain) bool) int {
	n := 0
	for dy := -rad; dy <= rad; dy++ {
		for dx := -rad; dx <= rad; dx++ {
			if zn.In(x+dx, y+dy) && f(zn.At(x+dx, y+dy)) {
				n++
			}
		}
	}
	return n
}

// qcWarm: dressed for the cold (furs, wool, cloaks), or fed against it.
func qcWarm(a *Actor) bool {
	if a.mealLv("rime") > 0 {
		return true
	}
	for _, p := range a.Body {
		if p.Item == nil || p.Item.Kind != "armor" && p.Item.Kind != "clothing" && p.Item.Slot != "back" {
			continue
		}
		n := lower(p.Item.Name + " " + p.Item.Look)
		for _, k := range []string{"fur", "wool", "pelt", "hide", "cloak", "mantle", "parka", "coat", "quilt", "thermal", "insulated", "cowl"} {
			if strings.Contains(n, k) {
				return true
			}
		}
	}
	return false
}

func qcHazardTick(g *Game) {
	zn, p := g.Cur, g.P
	if zn == nil || g.Dead || zn.Underwater || zn.Sky || g.Mode == "world" {
		return
	}
	// lava heat
	if n := qcCount(zn, p.X, p.Y, 2, func(t Terrain) bool { return t == TLava }); n > 0 && g.Turn%3 == 0 && p.Pow("carapace") == 0 {
		g.Msg("{{O|The heat of the molten rock scorches you.}}")
		g.hurt(p, 1+n/6, nil, "fire")
		g.Thirst++
	}
	// spores in fungal ground
	fungal := zn.Z == 0 && g.qcBiomeAt(zn.WX, zn.WY) == 3 || zn.Biome == 15 && zn.Z == 0
	if (fungal || qcCount(zn, p.X, p.Y, 2, func(t Terrain) bool { return t == TMushroom }) >= 3) && g.Turn%12 == 0 {
		if g.R.Int(1, 20)+p.M(TOU) < 11 {
			g.Msg("{{G|You breathe in a lungful of spores and cough.}}")
			p.AddSt("poisoned", 6)
		}
	}
	// crystal resonance
	if qcCount(zn, p.X, p.Y, 3, func(t Terrain) bool { return t == TCrystal }) >= 4 && g.Turn%15 == 0 {
		if g.R.Int(1, 20)+p.M(WIL) < 12 {
			g.Msg("{{C|The crystals ring in your skull. You stagger.}}")
			p.AddSt("stunned", 2)
		}
	}
	// deep cold underground without warm clothing (open-air cold is weather.go's)
	if zn.Z != 0 && qcCount(zn, p.X, p.Y, 4, func(t Terrain) bool { return t == TIce || t == TSnow }) >= 8 && g.Turn%4 == 0 && !qcWarm(p) {
		g.Msg("{{B|The deep cold bites through your clothes.}}")
		g.Hunger++
		g.hurt(p, 1, nil, "frost")
	}
}
