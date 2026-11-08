package engine

import (
	"fmt"
	"strings"
)

// Architecture and unique cultural/racial building shapes and materials (prefix ac).

func isWallTerrain(t Terrain) bool {
	switch t {
	case TWall, TWoodWall, TBrickWall, TStoneWall, TMarbleWall, TObsidianWall,
		TBoneWall, TAdobeWall, TBambooWall, TCoralWall, TMetalWall, THedgeWall, TIceWall:
		return true
	}
	return false
}

func isFloorTerrain(t Terrain) bool {
	switch t {
	case TFloor, TWoodFloor, TTileFloor, TCarpet:
		return true
	}
	return false
}

// acMaterials returns culturally and racially authentic wall and floor materials.
func (g *Game) acMaterials(bd *BurgData, cu *CultureData, role string) (wall, floor Terrain) {
	wall, floor = TStoneWall, TFloor
	sp := ""
	if cu != nil && cu.Species != "" {
		sp = strings.ToLower(cu.Species)
	}
	sphere := ""
	if cu != nil && cu.Sphere != "" {
		sphere = strings.ToLower(cu.Sphere)
	}
	genre := strings.ToLower(g.Genre)

	switch sp {
	case "dwarf", "chaos_dwarf":
		wall, floor = TStoneWall, TTileFloor
		if role == "smith" || role == "armorer" || genre == "steampunk" {
			wall = TMetalWall
		}
	case "elf", "celestial":
		wall, floor = TMarbleWall, TTileFloor
	case "wood_elf", "halfling":
		wall, floor = TWoodWall, TWoodFloor
		if role == "apothecary" || role == "herbalist" {
			wall = THedgeWall
		}
	case "dark_elf", "void_elf", "warp_fiend":
		wall, floor = TObsidianWall, TTileFloor
	case "orc", "troll", "ogor", "beastman", "ratfolk", "tomb_undead":
		wall, floor = TBoneWall, TFloor
		if role == "chief" || role == "shaman" {
			floor = TWoodFloor
		}
	case "merfolk":
		wall, floor = TCoralWall, TTileFloor
	case "lizardfolk", "saurian":
		wall, floor = TAdobeWall, TTileFloor
	default:
		// Culture spheres for humans and generic folk
		switch {
		case strings.Contains(sphere, "sinic") || strings.Contains(sphere, "japonic") || strings.Contains(sphere, "asian"):
			wall, floor = TBambooWall, TWoodFloor
			if role == "palace" || role == "temple" {
				wall = TBrickWall
			}
		case strings.Contains(sphere, "semitic") || strings.Contains(sphere, "persianate") || strings.Contains(sphere, "sahel") || strings.Contains(sphere, "ethiop") || strings.Contains(sphere, "nahua") || strings.Contains(sphere, "maya") || strings.Contains(sphere, "andean"):
			wall, floor = TAdobeWall, TTileFloor
		case strings.Contains(sphere, "nordic") || strings.Contains(sphere, "celtic") || strings.Contains(sphere, "slavic") || strings.Contains(sphere, "boreal") || strings.Contains(sphere, "turtle") || strings.Contains(sphere, "ocean"):
			wall, floor = TWoodWall, TWoodFloor
		case strings.Contains(sphere, "hellen") || strings.Contains(sphere, "roman") || strings.Contains(sphere, "imper") || strings.Contains(sphere, "frank") || strings.Contains(sphere, "romance"):
			wall, floor = TStoneWall, TTileFloor
			if role == "temple" || role == "palace" {
				wall = TMarbleWall
			}
		default:
			wall, floor = TWoodWall, TWoodFloor
			if bd != nil && bd.Capital {
				wall, floor = TStoneWall, TTileFloor
			}
		}
	}

	// Genre overrides
	if genre == "cyberpunk" || genre == "dieselpunk" || genre == "atompunk" {
		if role == "market" || role == "forge" || role == "barracks" {
			wall = TMetalWall
		} else {
			wall = TBrickWall
		}
	}
	return wall, floor
}

// acBuilding constructs non-square, diverse architectural footprints per culture/role.
func (g *Game) acBuilding(zn *Zone, x0, y0, w, h int, wall, floor Terrain, role string) (int, int) {
	if w < 5 || h < 4 {
		return g.buildingSimple(zn, x0, y0, w, h, wall, floor)
	}

	// Deterministic shape hash from zone key, coordinates and dimensions
	hVal := HashStr(fmt.Sprintf("%s:ac:%d:%d:%d:%d:%s", zn.Key, x0, y0, w, h, role))
	shapeChoice := hVal % 7

	switch shapeChoice {
	case 0:
		// Circular / Yurt / Rotunda (cut 4 corners)
		if w >= 6 && h >= 5 {
			for y := y0; y < y0+h; y++ {
				for x := x0; x < x0+w; x++ {
					isCorner := (x == x0 && (y == y0 || y == y0+h-1)) ||
						(x == x0+w-1 && (y == y0 || y == y0+h-1))
					if isCorner {
						continue
					}
					isWall := x == x0 || x == x0+w-1 || y == y0 || y == y0+h-1 ||
						(x == x0+1 && (y == y0 || y == y0+h-1)) ||
						(x == x0+w-2 && (y == y0 || y == y0+h-1))
					if isWall {
						zn.Set(x, y, wall)
					} else {
						zn.Set(x, y, floor)
					}
				}
			}
			// Central hearth or brazier for circular halls
			zn.Set(x0+w/2, y0+h/2, TBrazier)
			dx, dy := x0+w/2, y0+h-1
			zn.Set(dx, dy, TDoor)
			if w >= 8 {
				zn.Set(x0+1, y0+h/2, TWindow)
				zn.Set(x0+w-2, y0+h/2, TWindow)
			}
			return dx, dy
		}

	case 1:
		// L-shaped building with open courtyard/porch
		cutW, cutH := w/3+1, h/2
		for y := y0; y < y0+h; y++ {
			for x := x0; x < x0+w; x++ {
				if x >= x0+w-cutW && y < y0+cutH {
					// Open porch
					if x == x0+w-1 && y == y0+cutH-1 {
						zn.Set(x, y, TSeat)
					}
					continue
				}
				isOuter := x == x0 || y == y0+h-1 ||
					(y == y0 && x < x0+w-cutW) ||
					(x == x0+w-1 && y >= y0+cutH) ||
					(x == x0+w-cutW-1 && y < y0+cutH) ||
					(y == y0+cutH && x >= x0+w-cutW)
				if isOuter {
					zn.Set(x, y, wall)
				} else {
					zn.Set(x, y, floor)
				}
			}
		}
		dx, dy := x0+w/4+1, y0+h-1
		zn.Set(dx, dy, TDoor)
		zn.Set(x0, y0+h/2, TWindow)
		return dx, dy

	case 2:
		// Courtyard / Atrium / Siheyuan
		if w >= 8 && h >= 6 {
			for y := y0; y < y0+h; y++ {
				for x := x0; x < x0+w; x++ {
					if y == y0 || y == y0+h-1 || x == x0 || x == x0+w-1 {
						zn.Set(x, y, wall)
					} else {
						zn.Set(x, y, floor)
					}
				}
			}
			// Open courtyard in center
			cx, cy := x0+w/2, y0+h/2
			zn.Set(cx, cy, TTileFloor)
			zn.Set(cx-1, cy, TTileFloor)
			zn.Set(cx+1, cy, TTileFloor)
			zn.Set(cx, cy, TWell)
			dx, dy := x0+w/2, y0+h-1
			zn.Set(dx, dy, TDoor)
			zn.Set(x0+2, y0, TWindow)
			zn.Set(x0+w-3, y0, TWindow)
			return dx, dy
		}

	case 3:
		// Octagonal / Chamfered keep with corner pillars
		for y := y0; y < y0+h; y++ {
			for x := x0; x < x0+w; x++ {
				if y == y0 || y == y0+h-1 || x == x0 || x == x0+w-1 {
					zn.Set(x, y, wall)
				} else {
					zn.Set(x, y, floor)
				}
			}
		}
		// Corner pillars
		zn.Set(x0+1, y0+1, TPillar)
		zn.Set(x0+w-2, y0+1, TPillar)
		zn.Set(x0+1, y0+h-2, TPillar)
		zn.Set(x0+w-2, y0+h-2, TPillar)
		dx, dy := x0+w/2, y0+h-1
		zn.Set(dx, dy, TDoor)
		zn.Set(x0, y0+h/2, TWindow)
		zn.Set(x0+w-1, y0+h/2, TWindow)
		return dx, dy

	case 4:
		// Multi-room partitioned house (two interconnected rooms)
		for y := y0; y < y0+h; y++ {
			for x := x0; x < x0+w; x++ {
				if y == y0 || y == y0+h-1 || x == x0 || x == x0+w-1 {
					zn.Set(x, y, wall)
				} else {
					zn.Set(x, y, floor)
				}
			}
		}
		midX := x0 + w/2
		for y := y0 + 1; y < y0+h-1; y++ {
			zn.Set(midX, y, wall)
		}
		zn.Set(midX, y0+h/2, TDoorOpen) // internal door
		dx, dy := x0+w/4, y0+h-1
		zn.Set(dx, dy, TDoor)
		zn.Set(x0+1, y0, TWindow)
		zn.Set(x0+w-2, y0, TWindow)
		return dx, dy

	case 5:
		// Longhouse with apsidal curved ends
		for y := y0; y < y0+h; y++ {
			for x := x0; x < x0+w; x++ {
				if y == y0 || y == y0+h-1 || x == x0 || x == x0+w-1 {
					zn.Set(x, y, wall)
				} else {
					zn.Set(x, y, floor)
				}
			}
		}
		// Ridge-post pillars
		if w >= 7 {
			zn.Set(x0+w/3, y0+h/2, TPillar)
			zn.Set(x0+(2*w)/3, y0+h/2, TPillar)
		}
		dx, dy := x0+w/2, y0+h-1
		zn.Set(dx, dy, TDoor)
		zn.Set(x0, y0+h/2, TWindow)
		zn.Set(x0+w-1, y0+h/2, TWindow)
		return dx, dy

	default:
		// Rectangular pavilion with front porch and windows
		for y := y0; y < y0+h; y++ {
			for x := x0; x < x0+w; x++ {
				if y == y0 || y == y0+h-1 || x == x0 || x == x0+w-1 {
					zn.Set(x, y, wall)
				} else {
					zn.Set(x, y, floor)
				}
			}
		}
		zn.Set(x0+1, y0+h-2, TPillar)
		zn.Set(x0+w-2, y0+h-2, TPillar)
		dx, dy := x0+w/2, y0+h-1
		zn.Set(dx, dy, TDoor)
		zn.Set(x0+2, y0, TWindow)
		zn.Set(x0+w-3, y0, TWindow)
		return dx, dy
	}

	return g.buildingSimple(zn, x0, y0, w, h, wall, floor)
}

func (g *Game) buildingSimple(zn *Zone, x0, y0, w, h int, wall, floor Terrain) (int, int) {
	for y := y0; y < y0+h; y++ {
		for x := x0; x < x0+w; x++ {
			if y == y0 || y == y0+h-1 || x == x0 || x == x0+w-1 {
				zn.Set(x, y, wall)
			} else {
				zn.Set(x, y, floor)
			}
		}
	}
	dx, dy := x0+w/2, y0+h-1
	if y0+h-1 >= MH-2 {
		dy = y0
	}
	zn.Set(dx, dy, TDoor)
	return dx, dy
}

// acCityWalls builds monumental gatehouses, corner bastions and cultural city walls.
func (g *Game) acCityWalls(zn *Zone, cp *cityPlan, bd *BurgData) {
	cu := g.W.cultByID[bd.Culture]
	wallMat, _ := g.acMaterials(bd, cu, "wall")

	in := func(x, y int) bool {
		_, ok := cp.districts[[2]int{x, y}]
		return ok && x >= 0 && y >= 0 && x < 3 && y < 3
	}

	// Helper to build a gatehouse at (gx, gy) along horizontal or vertical wall
	buildGateH := func(gx, gy int) {
		// Twin flanking gate towers (3x2)
		for y := gy; y < gy+2 && y < MH; y++ {
			for x := gx - 4; x <= gx-2; x++ {
				zn.Set(x, y, wallMat)
			}
			for x := gx + 2; x <= gx+4; x++ {
				zn.Set(x, y, wallMat)
			}
		}
		// Murder-hole windows and brazier torches
		zn.Set(gx-3, gy, TWindow)
		zn.Set(gx+3, gy, TWindow)
		if gy == 0 && gy+2 < MH {
			zn.Set(gx-2, gy+2, TBrazier)
			zn.Set(gx+2, gy+2, TBrazier)
		} else if gy > 2 {
			zn.Set(gx-2, gy-1, TBrazier)
			zn.Set(gx+2, gy-1, TBrazier)
		}
		// Grand gateway arch and inscription plaque
		zn.Set(gx-1, gy, TGateArch)
		zn.Set(gx, gy, TGateArch)
		zn.Set(gx+1, gy, TGateArch)
		zn.Set(gx, gy, TInscription)
		zn.Notes[gy*MW+gx] = fmt.Sprintf("Monumental Gate of {{Y|%s}}. Guards watch from the bastions.", bd.Name)
	}

	buildGateV := func(gx, gy int) {
		// Flanking gate towers (2x3)
		for x := gx; x < gx+2 && x < MW; x++ {
			for y := gy - 3; y <= gy-1; y++ {
				zn.Set(x, y, wallMat)
			}
			for y := gy + 1; y <= gy+3; y++ {
				zn.Set(x, y, wallMat)
			}
		}
		zn.Set(gx, gy-2, TWindow)
		zn.Set(gx, gy+2, TWindow)
		if gx == 0 && gx+2 < MW {
			zn.Set(gx+2, gy-1, TBrazier)
			zn.Set(gx+2, gy+1, TBrazier)
		} else if gx > 2 {
			zn.Set(gx-1, gy-1, TBrazier)
			zn.Set(gx-1, gy+1, TBrazier)
		}
		zn.Set(gx, gy-1, TGateArch)
		zn.Set(gx, gy, TGateArch)
		zn.Set(gx, gy+1, TGateArch)
		zn.Set(gx, gy, TInscription)
		zn.Notes[gy*MW+gx] = fmt.Sprintf("Gatehouse of {{Y|%s}}.", bd.Name)
	}

	// North edge
	if !in(zn.ZX, zn.ZY-1) {
		for x := 0; x < MW; x++ {
			zn.Set(x, 0, wallMat)
			// Bastion crenellations every 12 tiles
			if x%12 == 0 && x > 2 && x < MW-3 && abs(x-40) > 6 {
				zn.Set(x, 1, wallMat)
				zn.Set(x, 0, TWindow)
			}
		}
		buildGateH(40, 0)
	}

	// South edge
	if !in(zn.ZX, zn.ZY+1) {
		for x := 0; x < MW; x++ {
			zn.Set(x, MH-1, wallMat)
			if x%12 == 0 && x > 2 && x < MW-3 && abs(x-40) > 6 {
				zn.Set(x, MH-2, wallMat)
				zn.Set(x, MH-1, TWindow)
			}
		}
		buildGateH(40, MH-1)
	}

	// West edge
	if !in(zn.ZX-1, zn.ZY) {
		for y := 0; y < MH; y++ {
			zn.Set(0, y, wallMat)
			if y%8 == 0 && y > 1 && y < MH-2 && abs(y-12) > 4 {
				zn.Set(1, y, wallMat)
				zn.Set(0, y, TWindow)
			}
		}
		buildGateV(0, 12)
	}

	// East edge
	if !in(zn.ZX+1, zn.ZY) {
		for y := 0; y < MH; y++ {
			zn.Set(MW-1, y, wallMat)
			if y%8 == 0 && y > 1 && y < MH-2 && abs(y-12) > 4 {
				zn.Set(MW-2, y, wallMat)
				zn.Set(MW-1, y, TWindow)
			}
		}
		buildGateV(MW-1, 12)
	}
}
