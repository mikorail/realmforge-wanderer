package engine

import "fmt"

// Hazard marks on generated zones (hazards.go). Side rng seed:hz:zoneKey only.

func hzZone(g *Game, zn *Zone, _ *Rng) {
	w := g.W
	if w == nil || w.Hazards == nil || zn.Z != 0 || !w.In(zn.WX, zn.WY) || zn.Sky || zn.Underwater {
		return
	}
	p := w.Idx(zn.WX, zn.WY)
	r := NewRng(w.Seed + ":hz:" + zn.Key)
	town := zn.Kind == "town" && zn.Burg > 0
	var bd *BurgData
	if zn.Burg > 0 {
		bd = w.burgByID[zn.Burg]
	}
	for _, e := range w.hzRecentAt(p) {
		hit := town && hzHas(e.Towns, zn.Burg)
		hzMark(g, zn, r, e, bd, hit)
	}
	if v := w.hzVolcanoAt(p); v != nil && zn.ZX == 1 && zn.ZY == 1 && !town {
		hzVolcanoZone(zn, r, v, w.Year)
	}
	if ob := w.hzOutbreakAt(p); ob != nil && bd != nil && bd.Alive {
		hzOutbreakZone(g, zn, r, ob, bd, town)
	}
}

// hzScatter turns a share of tiles matching from into to.
func hzScatter(zn *Zone, r *Rng, share float64, to Terrain, from ...Terrain) int {
	n := 0
	for i, t := range zn.T {
		for _, f := range from {
			if t == f && r.Chance(share) {
				zn.T[i] = to
				n++
				break
			}
		}
	}
	return n
}

func hzNoteAt(zn *Zone, r *Rng, text string) {
	if x, y, ok := zn.FreeSpot(r, func(x, y int) bool { return x > 2 && y > 2 && x < MW-3 && y < MH-3 }); ok {
		zn.Set(x, y, TInscription)
		zn.Notes[y*MW+x] = text
	}
}

func hzMark(g *Game, zn *Zone, r *Rng, e *HzRecent, bd *BurgData, hit bool) {
	age := g.W.Year - e.Year
	fade := 1 - float64(age)/float64(maxi(1, hzMarkYears[e.Kind]+1))
	label := hzLabel(e)
	switch e.Kind {
	case "quake", "tornado", "landslide", "avalanche":
		if hit {
			// a ruined district: walls broken to rubble in one quarter of the zone, homeless townsfolk
			x0, y0 := r.Int(4, MW-30), r.Int(3, MH-12)
			for y := y0; y < y0+9; y++ {
				for x := x0; x < x0+26; x++ {
					t := zn.At(x, y)
					if (t == TWall || t == TDoor || t == TFloor || t == TStall) && r.Chance(0.45*fade+0.15) {
						zn.Set(x, y, TRubble)
					} else if (t == TRoad || t == TCobble || t == TGrass || t == TDirt) && r.Chance(0.12*fade+0.04) {
						zn.Set(x, y, TRubble)
					}
				}
			}
			if e.Kind == "avalanche" {
				hzScatter(zn, r, 0.25*fade, TSnow, TRubble, TRoad, TCobble)
			}
			hzNoteAt(zn, r, fmt.Sprintf("Cracked walls and heaps of rubble: what is left of these streets after %s.", label))
			if bd != nil && bd.Alive {
				n := r.Int(1, 3)
				for k := 0; k < n; k++ {
					a := g.makeNPC("villager", bd, g.W.cultByID[bd.Culture], r)
					a.Role, a.Title = "homeless", "homeless "+Pick(r, []string{"weaver", "potter", "mason", "widow", "child", "carter"})
					a.Desc = fmt.Sprintf("Lost their home in %s. They sleep in the rubble with what they could carry.", label)
					a.Set("hzhomeless")
					swPut(zn, r, a, func(x, y int) bool { return x >= x0 && x < x0+26 && y >= y0 && y < y0+9 })
				}
			}
		} else if e.Kind == "quake" && e.Severity >= 7 && zn.Kind != "town" {
			hzScatter(zn, r, 0.05*fade, TRubble, TGrass, TDirt, TSand)
			hzScatter(zn, r, 0.25*fade, TDeadTree, TTree)
		} else if e.Kind == "landslide" || e.Kind == "avalanche" {
			if zn.Kind != "town" {
				to := TRubble
				if e.Kind == "avalanche" {
					to = TSnow
				}
				hzScatter(zn, r, 0.12*fade, to, TGrass, TDirt, TRoad)
			}
		}
	case "tsunami", "flood", "storm":
		// mud, driftwood and wrack: dirt over grass and roads, rubble in the low streets, stranded boats
		share := 0.25 * fade
		if !hit && zn.Kind == "town" {
			share *= 0.3
		}
		hzScatter(zn, r, share, TDirt, TGrass, TTallGrass, TRoad, TCobble, TCrop)
		hzScatter(zn, r, share*0.25, TRubble, TFloor, TStall, TBush)
		if e.Kind == "storm" {
			hzScatter(zn, r, 0.3*fade, TDeadTree, TTree)
		}
		if hit {
			for k := 0; k < 3; k++ {
				if x, y, ok := zn.FreeSpot(r, nil); ok {
					zn.Items = append(zn.Items, &Item{ID: g.newID(), Kind: "scrap", Name: Pick(r, []string{"driftwood", "a smashed oar", "a waterlogged chest lid", "a broken net"}), Glyph: '~', F: CBrown, X: x, Y: y, Value: 1, Weight: 2})
				}
			}
			hzNoteAt(zn, r, fmt.Sprintf("A line of dried mud on the walls, chest-high: the water mark of %s.", label))
		}
	case "eruption":
		share := 0.5 * fade
		if zn.Kind == "town" {
			share *= 0.4
		}
		hzScatter(zn, r, share, TAsh, TGrass, TTallGrass, TDirt, TSand, TCrop, TMoss)
		hzScatter(zn, r, share, TDeadTree, TTree, TBush)
		if hit {
			hzNoteAt(zn, r, fmt.Sprintf("Roofs bowed under grey ash, doors dug out by hand: %s buried this town.", label))
		}
	case "wildfire":
		if zn.Kind != "town" {
			hzScatter(zn, r, 0.7*fade, TDeadTree, TTree)
			hzScatter(zn, r, 0.4*fade, TAsh, TBush, TTallGrass, TGrass)
		}
	case "locust":
		hzScatter(zn, r, 0.8, TDirt, TCrop)
		hzScatter(zn, r, 0.6, TGrass, TTallGrass)
		if hit {
			hzNoteAt(zn, r, fmt.Sprintf("Stalks stripped to the stem and drifts of dead insects: %s.", label))
		}
	}
}

func hzVolcanoZone(zn *Zone, r *Rng, v *HzVolcano, year int) {
	cx, cy := MW/2, MH/2
	for y := cy - 7; y <= cy+7; y++ {
		for x := cx - 16; x <= cx+16; x++ {
			d := float64((x-cx)*(x-cx))/4 + float64((y-cy)*(y-cy))
			switch {
			case d < 6 && v.Active:
				zn.Set(x, y, TLava)
			case d < 12:
				zn.Set(x, y, TRock)
			case d < 40 && r.Chance(0.8):
				zn.Set(x, y, TAsh)
			}
		}
	}
	zn.Set(cx, cy+4, TAsh)
	when := "no one remembers it waking"
	if v.Last > 0 {
		when = fmt.Sprintf("it last woke in Y%d, %d years ago", v.Last, year-v.Last)
	}
	state := "sleeping"
	if v.Active {
		state = "smoking"
	}
	zn.Set(cx, cy+5, TInscription)
	zn.Notes[(cy+5)*MW+cx] = fmt.Sprintf("%s, %s; %s.", v.Name, state, when)
	zn.Name = v.Name
}

func hzOutbreakZone(g *Game, zn *Zone, r *Rng, ob *HzOutbreak, bd *BurgData, town bool) {
	w := g.W
	d := w.hzDisease(ob.Disease)
	cu := w.cultByID[bd.Culture]
	if d.Vector == "crop" {
		hzScatter(zn, r, 0.6, TDeadTree, TCrop)
		return
	}
	if town {
		// sick townsfolk
		for _, a := range zn.Actors {
			if a.Tpl == "npc" && a.SimPer == 0 && (a.Role == "villager" || a.Role == "") && r.Chance(0.35) {
				a.Set("hzsick")
				a.Desc += fmt.Sprintf(" Pale and sweating, they show %s: %s.", ob.Name, Pick(r, d.Symptoms))
			}
		}
		if zn.ZX == 1 && zn.ZY == 1 || r.Chance(0.3) {
			doc := g.makeNPC("apothecary", bd, cu, r)
			doc.Role, doc.Title = "physician", g.hzPhysicianTitle()
			doc.Desc = fmt.Sprintf("A %s come to fight %s. They smell of vinegar and smoke.", doc.Title, ob.Name)
			doc.AI = "town"
			swPut(zn, r, doc, swNear(MW/2, MH/2, 20, 8))
		}
		// marked doors
		hzNoteAt(zn, r, fmt.Sprintf("A cross daubed on the door and a line beneath it: %s is in this house.", ob.Name))
		if w.hzQuarantine(bd.ID) != nil {
			hzQuarantineGates(g, zn, r, bd, ob)
		}
		return
	}
	if zn.Kind == "outskirts" {
		// burning of the dead
		if x, y, ok := zn.FreeSpot(r, func(x, y int) bool { return x > 4 && y > 4 && x < MW-5 && y < MH-5 }); ok {
			zn.Set(x, y, TBrazier)
			for _, dd := range [][2]int{{-1, 0}, {1, 0}, {0, -1}, {0, 1}} {
				if zn.Walkable(x+dd[0], y+dd[1]) {
					zn.Set(x+dd[0], y+dd[1], TBones)
				}
			}
			zn.Set(x, y+2, TInscription)
			zn.Notes[(y+2)*MW+x] = fmt.Sprintf("The pyre of the dead of %s. They burn them here at dusk, so %s does not walk back into town.", bd.Name, ob.Name)
		}
		if w.hzQuarantine(bd.ID) != nil {
			sm := g.makeNPC("villager", bd, cu, r)
			sm.Role, sm.Title = "hzsmuggler", "smuggler"
			sm.Desc = fmt.Sprintf("Knows a drain under the walls of %s. For a price, the quarantine has a hole in it.", bd.Name)
			sm.AI = "still"
			swPut(zn, r, sm, nil)
		}
	}
}

// hzQuarantineGates: guards in front of every gate of a quarantined town zone, and a notice.
func hzQuarantineGates(g *Game, zn *Zone, r *Rng, bd *BurgData, ob *HzOutbreak) {
	cu := g.W.cultByID[bd.Culture]
	gates := [][4]int{{40, 0, 40, 1}, {40, MH - 1, 40, MH - 2}, {0, 12, 1, 12}, {MW - 1, 12, MW - 2, 12}}
	n := 0
	for _, gt := range gates {
		t := zn.At(gt[0], gt[1])
		if t != TDoorOpen && t != TDoor {
			continue
		}
		zn.Set(gt[0], gt[1], TDoor)
		gd := g.makeNPC("guard", bd, cu, r)
		gd.Title = "quarantine guard"
		gd.Desc = fmt.Sprintf("Masked and armed, guarding the gate of %s against %s. No one goes in or out.", bd.Name, ob.Name)
		gd.AI = "still"
		gd.Set("hzgate")
		gd.X, gd.Y, gd.HomeX, gd.HomeY = gt[2], gt[3], gt[2], gt[3]
		if zn.ActorAt(gd.X, gd.Y) == nil && zn.Walkable(gd.X, gd.Y) {
			zn.Actors = append(zn.Actors, gd)
			n++
		}
	}
	if n > 0 || zn.ZX == 1 && zn.ZY == 1 {
		hzNoteAt(zn, r, fmt.Sprintf("QUARANTINE. By order of %s the gates of %s are closed while %s lasts. Physicians only.", g.W.StateName(bd.State), bd.Name, ob.Name))
	}
}
