package engine

import (
	"fmt"
	"strings"
)

// Culture pack buildings (prefix pb): Material.Buildings (materials, house, hall, temple, gate, roof, colour,
// ornament) in the wall and floor a culture builds with, in the look text of its walls, doors and gates, in
// building signs, and in a line on first entering a town. Everything is optional: no pack, no Buildings, or words
// the engine cannot place leave the old behaviour untouched. Pure hashes of seed and zone key, never g.R. State
// g.ext("pb") (planet-local): the towns whose building line was shown.
//
//   - pbMaterials(cu, role, wall, floor)  hook in architecture.go acMaterials (before the genre overrides)
//   - pbLook(z, x, y)                     hook in render.go look text, after the terrain description
//   - zone hook                           one building sign by a door per town zone (temple always), a gate sign
//   - enter hook                          "the houses here are ..." once per town

type pbState struct {
	Seen map[int]bool `json:"s,omitempty"`
}

func (g *Game) pb() *pbState {
	st := g.ext("pb", func() any { return &pbState{} }).(*pbState)
	if st.Seen == nil {
		st.Seen = map[int]bool{}
	}
	return st
}

func init() {
	unLocalKeys = append(unLocalKeys, "pb")
	zoneHooks = append(zoneHooks, pbZone)
	enterHooks = append(enterHooks, pbEnter)
}

// pbStyle returns the Buildings of a culture id, or nil.
func (g *Game) pbStyle(culture int) *BuildingStyle {
	if p := cgPack(g.W, culture); p != nil && p.Material != nil {
		return p.Material.Buildings
	}
	return nil
}

var pbWallOf = map[string]Terrain{
	"timber": TWoodWall, "brick": TBrickWall, "stone": TStoneWall, "marble": TMarbleWall, "obsidian": TObsidianWall,
	"bone": TBoneWall, "adobe": TAdobeWall, "bamboo": TBambooWall, "coral": TCoralWall, "ironwork": TMetalWall,
	"hedge": THedgeWall, "ice": TIceWall,
}

var pbFloorOf = map[string]Terrain{
	"timber": TWoodFloor, "bamboo": TWoodFloor, "hedge": TWoodFloor, "bone": TFloor, "ice": TFloor,
	"brick": TTileFloor, "stone": TTileFloor, "marble": TTileFloor, "obsidian": TTileFloor, "adobe": TTileFloor,
	"coral": TTileFloor, "ironwork": TTileFloor,
}

// pbText picks the pack words that describe a role's building.
func pbText(b *BuildingStyle, role string) string {
	switch role {
	case "temple", "shrine", "cathedral":
		return b.Temple
	case "palace", "hall", "academy", "barracks", "court", "plaza", "chief", "guildhall":
		return b.Hall
	}
	return b.House
}

// pbMaterials: the wall and floor a culture's pack builds with; the arguments come back unchanged without a
// usable pack. Called from acMaterials with the culture's data.
func (g *Game) pbMaterials(cu *CultureData, role string, wall, floor Terrain) (Terrain, Terrain) {
	if g == nil || g.W == nil || cu == nil {
		return wall, floor
	}
	b := g.pbStyle(cu.ID)
	if b == nil {
		return wall, floor
	}
	key := syFirstWord(strings.ToLower(pbText(b, role)), syMatWords)
	if _, ok := pbWallOf[key]; !ok {
		key = syFirstWord(strings.ToLower(strings.Join(b.Materials, " ")), syMatWords)
	}
	w, ok := pbWallOf[key]
	if !ok {
		return wall, floor
	}
	if f, ok := pbFloorOf[key]; ok {
		floor = f
	}
	return w, floor
}

// pbKind: the district kind of a town zone ("" if unknown).
func (g *Game) pbKind(z *Zone) string {
	k, _ := g.districtAt(z.WX, z.WY, z.ZX, z.ZY)
	return k
}

func pbDistrictText(b *BuildingStyle, kind string) (string, string) {
	switch kind {
	case "temple":
		return "temple", b.Temple
	case "palace", "plaza", "academy", "barracks", "arena":
		return "hall", b.Hall
	}
	return "house", b.House
}

// pbLook: look text for a building tile of a pack culture's town; "" otherwise.
func (g *Game) pbLook(z *Zone, x, y int) string {
	if g == nil || g.W == nil || z == nil || z.Kind != "town" || z.Z != 0 || z.Burg <= 0 {
		return ""
	}
	t := z.At(x, y)
	bd := g.W.burgByID[z.Burg]
	if bd == nil {
		return ""
	}
	b := g.pbStyle(bd.Culture)
	if b == nil {
		return ""
	}
	switch {
	case t == TGateArch || t == TGateClosed:
		if b.Gate != "" {
			return "{{K|The gate here: " + pdClean(b.Gate) + ".}}"
		}
	case t == TDoor || t == TWindow:
		if b.Ornament != "" {
			return "{{K|Local ornament: " + pdClean(b.Ornament) + ".}}"
		}
	case isWallTerrain(t):
		name, txt := pbDistrictText(b, g.pbKind(z))
		if txt == "" {
			return ""
		}
		s := "{{K|A " + name + " built in the local manner: " + pdClean(txt) + "."
		if b.Roof != "" {
			s += " Roofs: " + pdClean(b.Roof) + "."
		}
		if b.Color != "" {
			s += " Colour: " + pdClean(b.Color) + "."
		}
		return s + "}}"
	}
	return ""
}

// pbZone: building signs. Side hashes only.
func pbZone(g *Game, zn *Zone, _ *Rng) {
	if g == nil || g.W == nil || zn == nil || zn.Kind != "town" || zn.Z != 0 || zn.Burg <= 0 {
		return
	}
	bd := g.W.burgByID[zn.Burg]
	if bd == nil {
		return
	}
	b := g.pbStyle(bd.Culture)
	if b == nil {
		return
	}
	if zn.Notes == nil {
		zn.Notes = map[int]string{}
	}
	h := HashStr(g.W.Seed + ":pb:" + zn.Key)
	kind := g.pbKind(zn)
	name, txt := pbDistrictText(b, kind)

	// a sign beside one door: temples always, other districts about a third of the time
	if txt != "" && (name == "temple" || h%3 == 0) {
		var doors []int
		for i, t := range zn.T {
			if t == TDoor {
				doors = append(doors, i)
			}
		}
		if len(doors) > 0 {
			head := map[string]string{"temple": "House of worship", "hall": "Hall", "house": "House"}[name]
			body := fmt.Sprintf("{{Y|%s of %s}}\nBuilt in the manner of the %s: %s.", head, bd.Name, pbPeople(g, bd), pdClean(txt))
			if b.Ornament != "" && (name != "house" || h%2 == 0) {
				body += " Over the door: " + pdClean(b.Ornament) + "."
			}
			start := int((h >> 8) % uint64(len(doors)))
		doorLoop:
			for k := 0; k < len(doors); k++ {
				d := doors[(start+k)%len(doors)]
				x, y := d%MW, d/MW
				for _, o := range [][2]int{{-1, 1}, {1, 1}, {-1, -1}, {1, -1}} {
					if nmSafe(zn, x+o[0], y+o[1]) {
						zn.Set(x+o[0], y+o[1], TInscription)
						zn.Notes[(y+o[1])*MW+x+o[0]] = body
						break doorLoop
					}
				}
			}
		}
	}

	// a sign by the gate
	if b.Gate != "" {
		for i, t := range zn.T {
			if t == TGateArch || t == TGateClosed {
				nmSign(zn, i%MW, i/MW, 3, fmt.Sprintf("{{Y|The gate of %s}}\n%s.", bd.Name, pbUpFirst(pdClean(b.Gate))))
				break
			}
		}
	}
}

func pbPeople(g *Game, bd *BurgData) string {
	if cu := g.W.cultByID[bd.Culture]; cu != nil && cu.Name != "" {
		return cu.Name
	}
	return "local"
}

// pbEnter: once per town, a line about how the place is built.
func pbEnter(g *Game, zn *Zone) {
	if g == nil || g.W == nil || zn == nil || zn.Kind != "town" || zn.Z != 0 || zn.Burg <= 0 {
		return
	}
	st := g.pb()
	if st.Seen[zn.Burg] {
		return
	}
	bd := g.W.burgByID[zn.Burg]
	if bd == nil {
		return
	}
	b := g.pbStyle(bd.Culture)
	if b == nil || b.House == "" {
		return
	}
	st.Seen[zn.Burg] = true
	s := "{{K|The houses here are " + pdClean(b.House) + "."
	if b.Roof != "" {
		s += " Roofs: " + pdClean(b.Roof) + "."
	}
	if b.Color != "" {
		s += " Colour: " + pdClean(b.Color) + "."
	}
	g.Msg(s + "}}")
}

func pbUpFirst(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}
