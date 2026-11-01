package engine

import (
	"fmt"
	"strings"
)

// Stat-gated world interactions: sealed doors, ledges, puzzle locks, whispering doors, proud
// guardians, insight veils, heart seals and skill locks placed in dungeons, towers and ancient
// sites. All placement uses the side rng seed:in:zoneKey so zone generation is unchanged.
// Bumping a gate (interact hook) or pressing U beside it shows the requirement and your value.

var inFloors = map[Terrain]bool{TFloor: true, TMoss: true, TFungus: true, TDirt: true, TCobble: true, TCarpet: true,
	TAsh: true, TSand: true, TSnow: true, TBones: true, TCrystalFloor: true, TGrass: true}

var inReqs = []string{"str", "agi", "int", "wil", "ego", "insight", "hearts", "skill"}

var inGateName = map[string]string{
	"str": "A heavy door of fitted stone, stuck fast.", "agi": "A ledge of crumbling stone, barely a hand wide, runs past here.",
	"int": "A lock of nested rings and glyphs. A puzzle.", "wil": "A door that whispers. The whispers want a steady mind.",
	"ego":     "A proud guardian of stone stands here, arms crossed. It wants to be talked down.",
	"insight": "A veil hangs in the doorway, showing only what you are ready to see.",
	"hearts":  "A seal of red stone with heart-shaped hollows. It wants living hearts.", "skill": "A mechanism only a practised hand could open.",
}

var inGlyphs = []string{
	"THE FLOOR REMEMBERS EVERY STEP", "COUNT THE STAIRS AGAIN", "IT IS KINDER TO BE UNSEEN",
	"WE WERE HERE BEFORE THE WALLS", "DO NOT NAME THE ONE WHO LISTENS", "BEHIND THE STONE, MORE STONE, THEN US",
	"YOU HAVE ALREADY ARRIVED", "THE DEEP IS A KIND OF EYE", "LOOK AWAY AND IT LOOKS CLOSER",
	"THE LAST WHO READ THIS STAYED", "HALF OF EVERY PRAYER IS ANSWERED", "SOMETHING IS LEARNING YOUR NAME",
}

var inLore = []string{
	"A scratched ledger: the builders sealed a room for the thing they could not kill, and then forgot why they had.",
	"A pilgrim's note: \"I climbed to see the sky and found the sky had climbed down to see me.\"",
	"A carved prayer to a god of doors. Every line ends in the same word, worn smooth by fingers.",
	"A map of this place with one room drawn in blood and labelled only \"LISTENING\".",
}

func inKeyOf(zn *Zone, idx int) string { return fmt.Sprintf("%s#%d", zn.Key, idx) }

func inIsWall(t Terrain) bool { return t == TWall || t == TRock }

// inPocket finds a solid block next to a floor tile and carves a 3x2 chamber, leaving the wall
// tile between as the gate. Returns the gate index and chamber cells.
func inPocket(zn *Zone, r *Rng, carve bool) (gate int, cells [][2]int, ok bool) {
	dirs := [][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}}
	for tries := 0; tries < 120; tries++ {
		sx, sy := r.Int(3, MW-4), r.Int(3, MH-4)
		if !inFloors[zn.At(sx, sy)] {
			continue
		}
		d := Pick(r, dirs)
		dx, dy := d[0], d[1]
		px, py := -dy, dx // perpendicular
		clear := true
	scan:
		for depth := 1; depth <= 4; depth++ {
			for p := -2; p <= 2; p++ {
				x, y := sx+dx*depth+px*p, sy+dy*depth+py*p
				if x < 1 || y < 1 || x > MW-2 || y > MH-2 || !inIsWall(zn.At(x, y)) {
					clear = false
					break scan
				}
			}
		}
		if !clear {
			continue
		}
		fl := zn.At(sx, sy)
		for depth := 2; depth <= 3; depth++ {
			for p := -1; p <= 1; p++ {
				x, y := sx+dx*depth+px*p, sy+dy*depth+py*p
				cells = append(cells, [2]int{x, y})
				if carve {
					zn.Set(x, y, fl)
				}
			}
		}
		return (sy+dy)*MW + sx + dx, cells, true
	}
	return 0, nil, false
}

func inOpenFloor(g *Game, zn *Zone, r *Rng) (int, int, bool) {
	for tries := 0; tries < 200; tries++ {
		x, y := r.Int(3, MW-4), r.Int(3, MH-4)
		if !inFloors[zn.At(x, y)] || zn.ActorAt(x, y) != nil || len(zn.ItemsAt(x, y)) > 0 {
			continue
		}
		if _, n := zn.Notes[y*MW+x]; n {
			continue
		}
		open := true
		for dy := -1; dy <= 1 && open; dy++ {
			for dx := -1; dx <= 1; dx++ {
				if !zn.At(x+dx, y+dy).D().Walk {
					open = false
					break
				}
			}
		}
		if open {
			return x, y, true
		}
	}
	return 0, 0, false
}

func inRollGate(zn *Zone, r *Rng) *inSpot {
	lvl := zn.Level
	sp := &inSpot{Zone: zn.Key, K: "gate", Req: Pick(r, inReqs)}
	base := 15 + clampi(lvl/3, 0, 9)
	switch sp.Req {
	case "insight":
		sp.N = inTiers[clampi(lvl/6, 0, 2)]
	case "hearts":
		sp.N = clampi(1+lvl/12, 1, 4)
	case "skill":
		sp.Skill = Pick(r, []string{"tinkering", "lore", "persuasion", "survival"})
		sp.N = 1 + lvl/10
		if sp.N > 2 {
			sp.N = 2
		}
	default:
		sp.N = base + r.Int(0, 2)
	}
	sp.Reward = Pick(r, []string{"loot", "loot", "lore", "heart", "shortcut"})
	return sp
}

func inZone(g *Game, zn *Zone, _ *Rng) {
	switch zn.Kind {
	case "dungeon", "tower", "ruin", "crypt", "wonder", "lair", "underdark":
	default:
		return
	}
	st := inSt(g)
	r := NewRng(g.Seed + ":in:" + zn.Key)
	solid := zn.Kind == "dungeon" || zn.Kind == "tower" || zn.Kind == "underdark"
	add := func(idx int, sp *inSpot) { sp.Idx = idx; st.Spots[inKeyOf(zn, idx)] = sp }

	if r.Chance(0.4) {
		sp := inRollGate(zn, r)
		if solid {
			if gi, cells, ok := inPocket(zn, r, true); ok {
				gx, gy := gi%MW, gi/MW
				zn.Set(gx, gy, TStatue)
				zn.Notes[gi] = inGateName[sp.Req]
				fl := zn.At(cells[0][0], cells[0][1])
				c := cells[len(cells)-2]
				switch sp.Reward {
				case "lore":
					zn.Set(c[0], c[1], TInscription)
					zn.Notes[c[1]*MW+c[0]] = Pick(r, inLore)
				case "loot", "shortcut":
					for i := 0; i < 2+r.Int(0, 1); i++ {
						it := g.Loot(r, zn.Level/3+2)
						it.X, it.Y = cells[i][0], cells[i][1]
						zn.Items = append(zn.Items, it)
					}
					if sp.Reward == "shortcut" {
						for tries := 0; tries < 60; tries++ {
							x, y := r.Int(2, MW-3), r.Int(2, MH-3)
							if inFloors[zn.At(x, y)] && abs(x-c[0])+abs(y-c[1]) > 25 {
								g.corridor(zn, c[0], c[1], x, y, fl, r)
								zn.Set(gx, gy, TStatue)
								break
							}
						}
					}
				}
				add(gi, sp)
			}
		} else if x, y, ok := inOpenFloor(g, zn, r); ok {
			sp.Direct = true
			sp.Reward = Pick(r, []string{"loot", "lore", "heart", "gold"})
			zn.Set(x, y, TStatue)
			zn.Notes[y*MW+x] = inGateName[sp.Req]
			add(y*MW+x, sp)
		}
	}
	if solid && r.Chance(0.25) {
		if gi, cells, ok := inPocket(zn, r, true); ok {
			for i := 0; i < 2; i++ {
				it := g.Loot(r, zn.Level/3+3)
				it.X, it.Y = cells[i][0], cells[i][1]
				zn.Items = append(zn.Items, it)
			}
			add(gi, &inSpot{Zone: zn.Key, K: "secret"})
		}
	}
	if r.Chance(0.5) {
		for tries := 0; tries < 80; tries++ {
			x, y := r.Int(2, MW-3), r.Int(2, MH-3)
			i := y*MW + x
			if !inIsWall(zn.At(x, y)) || st.Spots[inKeyOf(zn, i)] != nil {
				continue
			}
			if zn.At(x+1, y).D().Walk && inFloors[zn.At(x+1, y)] || zn.At(x-1, y).D().Walk && inFloors[zn.At(x-1, y)] ||
				zn.At(x, y+1).D().Walk && inFloors[zn.At(x, y+1)] || zn.At(x, y-1).D().Walk && inFloors[zn.At(x, y-1)] {
				add(i, &inSpot{Zone: zn.Key, K: "glyph", Text: Pick(r, inGlyphs)})
				break
			}
		}
	}
	if r.Chance(0.12) {
		if x, y, ok := inOpenFloor(g, zn, r); ok {
			zn.Set(x, y, TAltar)
			zn.Notes[y*MW+x] = "A small red shrine, its offering bowl shaped like a heart. Bump to pray."
			add(y*MW+x, &inSpot{Zone: zn.Key, K: "heart"})
		}
	}
	deep := zn.Kind == "tower" || zn.Z >= 3
	if deep && r.Chance(0.12) {
		if x, y, ok := inOpenFloor(g, zn, r); ok {
			zn.Set(x, y, TAltar)
			zn.Notes[y*MW+x] = "An eldritch shrine. The altar is too many-angled. Bump to approach."
			add(y*MW+x, &inSpot{Zone: zn.Key, K: "eldritch"})
		}
	}
	// some horrors are veiled from the start
	for _, a := range zn.Actors {
		if inEldritch(a) && r.Chance(0.4) {
			a.Set("in_veil")
		}
	}
}

// inReveal shows hidden glyphs (insight 5) and secret doors (insight 30) in a zone.
func inReveal(g *Game, zn *Zone) {
	st := inSt(g)
	tier := inTier(st.Ins)
	found := false
	for _, sp := range st.Spots {
		if sp.Zone != zn.Key || sp.Done {
			continue
		}
		t := zn.At(sp.Idx%MW, sp.Idx/MW)
		switch sp.K {
		case "glyph":
			if tier >= 1 && inIsWall(t) {
				zn.Set(sp.Idx%MW, sp.Idx/MW, TInscription)
				zn.Notes[sp.Idx] = "Glyphs only you can see:\n\n{{M|" + sp.Text + "}}"
			}
		case "secret":
			if tier >= 3 && inIsWall(t) {
				zn.Set(sp.Idx%MW, sp.Idx/MW, TDoor)
				found = true
			}
		}
	}
	if found && g.Cur == zn {
		g.Msg("{{M|Your insight finds a hidden door.}}")
	}
}

func inGateVal(g *Game, sp *inSpot) (int, string) {
	p := g.P
	switch sp.Req {
	case "str":
		return p.Score(STR), "STR"
	case "agi":
		return p.Score(AGI), "AGI"
	case "int":
		return p.Score(INT) + g.Skills["tinkering"], "INT"
	case "wil":
		return p.Score(WIL), "WIL"
	case "ego":
		return p.Score(EGO) + 2*g.Skills["persuasion"], "EGO"
	case "insight":
		return inSt(g).Ins, "Insight"
	case "hearts":
		return inSt(g).Pieces / 4, "Hearts"
	case "skill":
		return g.Skills[sp.Skill], skillName(sp.Skill)
	}
	return 0, ""
}

var inFailText = map[string]string{
	"str": "You heave. It does not budge.", "agi": "You edge out, look down, and edge back.",
	"int": "The rings turn and turn and settle wrong.", "wil": "The whispers crowd in. You step back.",
	"ego": "The guardian looks through you. \"Not worth the breath.\"", "insight": "The veil shows you nothing but your own face.",
	"hearts": "The hollows stay dark.", "skill": "You do not know where to begin.",
}

var inOpenText = map[string]string{
	"str": "The stone grinds aside.", "agi": "You shuffle along the ledge and drop into the space beyond.",
	"int": "The rings click home. The lock gives.", "wil": "The whispers quiet and the door swings wide.",
	"ego": "You talk it down. It bows, slowly, and steps aside.", "insight": "The veil parts for those who have seen.",
	"hearts": "The hollows glow red. The seal opens.", "skill": "Your practised hands find the trick.",
}

// inBump handles interaction with any tile this feature owns. Returns true if handled.
func inBump(g *Game, x, y int) bool {
	zn := g.Cur
	if zn == nil || !zn.In(x, y) {
		return false
	}
	st := inSt(g)
	sp := st.Spots[inKeyOf(zn, y*MW+x)]
	if sp == nil {
		return false
	}
	switch sp.K {
	case "gate":
		if sp.Done {
			return false
		}
		v, label := inGateVal(g, sp)
		if v < sp.N {
			g.Msg(fmt.Sprintf("{{R|%s}} Requires %s %d (you have %d).", inFailText[sp.Req], label, sp.N, v))
			g.spend(500)
			return true
		}
		sp.Done = true
		if sp.Direct || sp.Req == "ego" {
			zn.Set(x, y, TFloor)
		} else {
			zn.Set(x, y, TDoorOpen)
		}
		delete(zn.Notes, y*MW+x)
		g.Msg("{{Y|" + inOpenText[sp.Req] + "}}")
		inReward(g, sp, zn, x, y)
		g.spend(1000)
		return true
	case "glyph":
		k := zn.Key + "glyph" + fmt.Sprint(sp.Idx)
		if zn.At(x, y) != TInscription {
			return false
		}
		g.showPopup("Glyphs", "{{M|"+sp.Text+"}}")
		if !g.Read[k] {
			g.Read[k] = true
			inGain(g, 1, ": the glyphs rearrange behind your eyes.")
			inComposureHit(g, 1)
		}
		g.spend(500)
		return true
	case "heart":
		if sp.Done {
			g.Msg("The bowl is empty and cool.")
			return true
		}
		sp.Done = true
		g.Msg("{{R|You pray at the red shrine. Something warm settles in the bowl.}}")
		inHeartPiece(g, "")
		g.spend(1000)
		return true
	case "eldritch":
		inShrineMenu(g, &Actor{Name: "Eldritch shrine", Glyph: 'Ω', F: CMagenta}, "The altar turns toward you without moving. \"Pay in what you have seen.\"", nil)
		if !sp.Done {
			// first approach is itself a sight
			sp.Done = true
			inGain(g, 2, ": the shrine shows itself.")
			inComposureHit(g, 1)
		}
		return true
	}
	return false
}

func inReward(g *Game, sp *inSpot, zn *Zone, x, y int) {
	r := NewRng(g.Seed + ":in:rw:" + inKeyOf(zn, sp.Idx))
	switch sp.Reward {
	case "heart":
		inHeartPiece(g, ": a cache beyond.")
	case "lore":
		if sp.Direct {
			text := Pick(r, inLore)
			g.showPopup("Hidden record", text)
			g.addSecret(Plain(text), "inscription", g.W.Idx(zn.WX, zn.WY), g.W.Year)
			g.gainXP(10 + g.Skills["lore"]*10)
		} else {
			g.Msg("A record is carved in the alcove beyond.")
		}
	case "gold":
		n := 20 * maxi(1, zn.Level)
		g.P.Gold += n
		g.Msg(fmt.Sprintf("Coins spill out: %d %s.", n, g.Sk.Currency))
	case "loot":
		if sp.Direct {
			for i := 0; i < 2; i++ {
				it := g.Loot(r, zn.Level/3+2)
				g.give(it)
				g.Msg("You take " + it.Display() + ".")
			}
		} else {
			g.Msg("Treasure glints in the chamber beyond.")
		}
	case "shortcut":
		g.Msg("A passage opens beyond, running a long way through the rock.")
	}
	if strings.Contains(sp.Req, "ego") {
		g.gainXP(10)
	}
}
