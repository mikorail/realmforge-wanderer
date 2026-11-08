package engine

import (
	"fmt"
	"strings"
)

// Architectural & Urban Evolution (prefix ue):
// 1. Multi-story rooftops and subterranean cellar networks beneath every building with cat burglar skylights and chimney descents.
// 2. Destructible city gates, barricades, and siege ballistas during wartime and rebellions.
// 3. Seasonal urban dressing: lantern festivals with glowing paper lanterns, autumn harvest bunting, and winter snow drifts along roof cornices.
// 4. Player-commissioned guildhalls, custom estate construction, and commemorative victory monuments in city plazas.

type ueState struct {
	Guildhalls map[string]string `json:"guildhalls,omitempty"` // townKey -> guildType
	Estates    map[string]bool   `json:"estates,omitempty"`    // townKey -> hasEstate
	Monuments  map[string]string `json:"monuments,omitempty"`  // townKey -> inscription
	Barricades map[string]int    `json:"barricades,omitempty"` // zoneKey:pos -> hp
	BallistaCD map[string]int    `json:"ballista_cd,omitempty"`
}

func (g *Game) ue() *ueState {
	st := g.ext("ue", func() any { return &ueState{} }).(*ueState)
	if st.Guildhalls == nil {
		st.Guildhalls = map[string]string{}
	}
	if st.Estates == nil {
		st.Estates = map[string]bool{}
	}
	if st.Monuments == nil {
		st.Monuments = map[string]string{}
	}
	if st.Barricades == nil {
		st.Barricades = map[string]int{}
	}
	if st.BallistaCD == nil {
		st.BallistaCD = map[string]int{}
	}
	return st
}

func init() {
	unLocalKeys = append(unLocalKeys, "ue")
	zoneHooks = append(zoneHooks, ueZoneHook)
	turnHooks = append(turnHooks, ueTurnHook)
	talkHooks = append(talkHooks, ueTalkHook)
}

func ueTurnHook(g *Game) {
	st := g.ue()
	for k, cd := range st.BallistaCD {
		if cd > 0 {
			st.BallistaCD[k] = cd - 1
		}
	}
}

func ueZoneHook(g *Game, zn *Zone, r *Rng) {
	if g == nil || zn == nil {
		return
	}
	sr := NewRng(fmt.Sprintf("%s:ue:%s", g.Seed, zn.Key))

	// Subterranean Cellar stratum beneath towns (Z == 1)
	if zn.Z == 1 {
		site := g.siteAt(zn.WX, zn.WY, zn.ZX, zn.ZY)
		if site != nil && site.Kind == "town" {
			ueGenerateCellarNetwork(g, zn, sr)
			return
		}
	}

	if zn.Kind != "town" || zn.Z != 0 {
		return
	}

	// 1. Seasonal Urban Dressing
	ueApplySeasonalDressing(g, zn, sr)

	// 2. Wartime barricades & siege ballistas
	isWar := g.war().Realm > 0
	if isWar || sr.Chance(0.15) {
		ueFortifyGatesAndBarricades(g, zn, sr)
	}

	// 3. Cellar entrances, ladders to scaffolds, chimneys & skylights
	ueAddCellarHatchesAndRooftops(g, zn, sr)

	// 4. Player commissioned buildings / monuments
	uePlaceCommissionedFeatures(g, zn)
}

func ueApplySeasonalDressing(g *Game, zn *Zone, sr *Rng) {
	season := g.Season() // 0 spring, 1 summer, 2 autumn, 3 winter
	// Autumn: Harvest bunting across the central avenue
	if season == 2 {
		for x := 10; x < MW-10; x += 3 {
			if zn.At(x, 11) == TCobble || zn.At(x, 11) == TRoad || zn.At(x, 11) == TDirt {
				zn.Set(x, 11, TBunting)
			}
		}
	}

	// Winter: Snow drifts banked along north building walls
	if season == 3 {
		for y := 2; y < MH-2; y++ {
			for x := 2; x < MW-2; x++ {
				t := zn.At(x, y)
				if isWallTerrain(t) && zn.At(x, y-1).D().Walk && sr.Chance(0.35) {
					zn.Set(x, y-1, TSnowDrift)
				}
			}
		}
	}

	// Paper Lantern festivals (hung outside inns and shops)
	for y := 2; y < MH-2; y++ {
		for x := 2; x < MW-2; x++ {
			if (zn.At(x, y) == TDoor || zn.At(x, y) == TDoorOpen) && sr.Chance(0.4) {
				if zn.In(x+1, y) && zn.At(x+1, y).D().Walk {
					zn.Set(x+1, y, TLantern)
				}
			}
		}
	}
}

func ueFortifyGatesAndBarricades(g *Game, zn *Zone, sr *Rng) {
	// Gate locations in standard town layouts
	gates := [][2]int{{40, 0}, {40, MH - 1}, {0, 12}, {MW - 1, 12}}
	for _, gt := range gates {
		gx, gy := gt[0], gt[1]
		// Add iron portcullis at gate
		if zn.In(gx, gy) && zn.At(gx, gy).D().Walk {
			zn.Set(gx, gy, TGateClosed)
		}
		// Mount siege ballistas on flanking gate bastions
		for _, dy := range []int{-2, 2} {
			bx, by := gx, gy+dy
			if gx == 0 || gx == MW-1 {
				bx, by = gx, gy+dy
			} else {
				bx, by = gx+dy, gy
			}
			if zn.In(bx, by) {
				zn.Set(bx, by, TBallista)
			}
		}
	}

	// Barricades across side alleys
	for y := 5; y < MH-5; y += 4 {
		for x := 10; x < MW-10; x += 15 {
			if zn.At(x, y).D().Walk && !zn.At(x-1, y).D().Walk && !zn.At(x+1, y).D().Walk {
				zn.Set(x, y, TBarricade)
			}
		}
	}
}

func ueAddCellarHatchesAndRooftops(g *Game, zn *Zone, sr *Rng) {
	// Look for enclosed rooms to place cellar stairs (TDown) and chimneys (TChimney)
	roomCount := 0
	for y := 4; y < MH-6; y += 7 {
		for x := 4; x < MW-8; x += 10 {
			if zn.At(x, y).D().Walk && isWallTerrain(zn.At(x-1, y)) {
				// Cellar entrance
				if roomCount%3 == 0 && zn.At(x, y) != TDown && zn.At(x, y) != TUp {
					zn.Set(x, y, TDown)
					zn.Notes[y*MW+x] = "A trapdoor and brick stair leading down into the building's vaulted cellar. (>)"
				}
				// Hearth & Chimney on back wall
				if zn.In(x+1, y-1) && isWallTerrain(zn.At(x+1, y-1)) {
					zn.Set(x+1, y-1, TChimney)
				}
				// Rooftop skylight
				if zn.In(x+3, y+2) && zn.At(x+3, y+2).D().Walk && sr.Chance(0.4) {
					zn.Set(x+3, y+2, TSkylight)
				}
				// External ladder/scaffold against outer wall
				if zn.In(x-2, y) && zn.At(x-2, y).D().Walk && sr.Chance(0.5) {
					zn.Set(x-2, y, TScaffold)
				}
				roomCount++
			}
		}
	}
}

func ueGenerateCellarNetwork(g *Game, zn *Zone, sr *Rng) {
	// Fill with solid rock/brick first
	for i := range zn.T {
		zn.T[i] = TWall
	}
	zn.Kind = "cellar"

	// Carve interconnected brick-vaulted storage cellars
	rooms := [][4]int{
		{10, 4, 18, 9},
		{32, 4, 16, 9},
		{52, 4, 18, 9},
		{12, 14, 20, 8},
		{36, 14, 18, 8},
		{56, 14, 16, 8},
	}

	for _, rm := range rooms {
		rx, ry, rw, rh := rm[0], rm[1], rm[2], rm[3]
		for y := ry; y < ry+rh && y < MH-1; y++ {
			for x := rx; x < rx+rw && x < MW-1; x++ {
				if x == rx || x == rx+rw-1 || y == ry || y == ry+rh-1 {
					zn.Set(x, y, TBrickWall)
				} else {
					zn.Set(x, y, TWoodFloor)
					// Scatter barrels, crates, cobwebs
					if sr.Chance(0.12) {
						zn.Set(x, y, TBarrel)
					} else if sr.Chance(0.12) {
						zn.Set(x, y, TCrate)
					}
				}
			}
		}
		// Doorway into corridor
		zn.Set(rx+rw/2, ry+rh-1, TDoor)
	}

	// Connect with corridors
	for x := 10; x < 72; x++ {
		zn.Set(x, 13, TTileFloor)
	}

	// Stairway up leading back to surface
	zn.Set(20, 8, TUp)
	zn.Notes[8*MW+20] = "Stone stairs climbing up to the surface street above. (<)"

	// Spawn cellar rats, spiders, or contraband smugglers
	verminKinds := []string{"rat", "spider"}
	for k := 0; k < 4; k++ {
		kind := Pick(sr, verminKinds)
		a := g.Spawn(kind, 1, sr)
		if a != nil {
			a.X, a.Y = 15+k*12, 8
			zn.Actors = append(zn.Actors, a)
		}
	}
}

func uePlaceCommissionedFeatures(g *Game, zn *Zone) {
	st := g.ue()
	townKey := fmt.Sprintf("%d,%d", zn.WX, zn.WY)

	// Monument in town plaza
	if insc, ok := st.Monuments[townKey]; ok {
		zn.Set(40, 10, TMonument)
		zn.Notes[10*MW+40] = insc
	}

	// Commissioned Guildhall
	if gh, ok := st.Guildhalls[townKey]; ok {
		// Place guildhall banner and master in east sector
		zn.Set(58, 8, TMonument)
		zn.Notes[8*MW+58] = fmt.Sprintf("Grand %s Guildhall of the Realm", strings.Title(gh))
	}

	// Commissioned Private Estate
	if st.Estates[townKey] {
		// Luxury estate in northwest quarter
		for y := 4; y <= 8; y++ {
			for x := 10; x <= 16; x++ {
				if x == 10 || x == 16 || y == 4 || y == 8 {
					zn.Set(x, y, TMarbleWall)
				} else {
					zn.Set(x, y, TTileFloor)
				}
			}
		}
		zn.Set(13, 8, TDoorOpen)
		zn.Set(13, 5, TBed)
		zn.Notes[5*MW+13] = "Your Private Estate Bedchamber. Rest here to recover fully."
	}
}

// ueBump handles interactive bump on urban evolution terrain
func ueBump(g *Game, x, y int) bool {
	zn := g.Cur
	if zn == nil || !zn.In(x, y) {
		return false
	}
	t := zn.At(x, y)
	st := g.ue()
	posKey := fmt.Sprintf("%s:%d,%d", zn.Key, x, y)

	switch t {
	case TBarricade:
		hp := st.Barricades[posKey]
		if hp == 0 {
			hp = 40
		}
		dmg := 12 + g.P.Pow("melee")*3
		hp -= dmg
		g.spend(1000)
		if hp <= 0 {
			delete(st.Barricades, posKey)
			zn.Set(x, y, TDirt)
			g.Msg(fmt.Sprintf("{{G|CRASH!}} You smash through the barricade! (%d dmg)", dmg))
			zn.Items = append(zn.Items, &Item{
				ID: g.newID(), Kind: "scrap", Name: "splintered barricade timber", Glyph: '=', F: CBrown, Weight: 2, X: x, Y: y,
			})
		} else {
			st.Barricades[posKey] = hp
			g.Msg(fmt.Sprintf("You hack at the fortified barricade. (%d dmg, %d HP remains)", dmg, hp))
		}
		return true

	case TBallista:
		if cd := st.BallistaCD[posKey]; cd > 0 {
			g.Msg(fmt.Sprintf("The siege ballista is being cranked and reloaded (%d turns).", cd))
			return true
		}
		st.BallistaCD[posKey] = 8
		g.spend(1200)
		g.Msg("{{Y|THWAAANG!}} You trip the lever of the mounted siege ballista!")

		// Determine bolt path (default: east or toward center)
		dx, dy := 1, 0
		if x > 40 {
			dx = -1
		} else if y == 0 {
			dx, dy = 0, 1
		} else if y == MH-1 {
			dx, dy = 0, -1
		}

		// Fire bolt up to 18 tiles
		bx, by := x+dx, y+dy
		pierced := 0
		for step := 0; step < 18; step++ {
			if !zn.In(bx, by) {
				break
			}
			bt := zn.At(bx, by)
			if isWallTerrain(bt) {
				g.Msg(fmt.Sprintf("The ballista bolt shatters against the %s!", bt.D().Name))
				break
			}
			if bt == TBarricade {
				zn.Set(bx, by, TDirt)
				g.Msg("The ballista bolt {{R|splinters}} the barricade to kindling!")
			}
			if a := zn.ActorAt(bx, by); a != nil && a != g.P {
				boltDmg := 45 + g.R.Int(10, 30)
				g.hurt(a, boltDmg, g.P, "a siege ballista bolt")
				g.Msg(fmt.Sprintf("The heavy iron-headed bolt {{R|skewers}} %s for %d damage!", a.Name, boltDmg))
				pierced++
			}
			bx += dx
			by += dy
		}
		if pierced == 0 {
			g.Msg("The bolt embeds deeply into the stone pavement.")
		}
		return true

	case TChimney:
		g.spend(1000)
		g.Msg("{{C|You squeeze into the soot-stained chimney and slide down the flue!}}")
		// Drop into the hearth inside
		for dy := 1; dy <= 3; dy++ {
			if zn.In(x, y+dy) && zn.At(x, y+dy).D().Walk {
				g.P.X, g.P.Y = x, y+dy
				g.P.AddSt("stealth", 5)
				g.Msg("You land softly in the hearth in a puff of warm ash, unseen.")
				return true
			}
		}
		return true

	case TSkylight:
		g.spend(800)
		g.Msg("{{C|You lever open the leaded skylight sash and lower yourself through!}}")
		// Drop into room below
		for dy := 1; dy <= 2; dy++ {
			if zn.In(x, y+dy) && zn.At(x, y+dy).D().Walk {
				g.P.X, g.P.Y = x, y+dy
				g.P.AddSt("stealth", 4)
				g.Msg("You drop silently onto the floorboards past the locked entrance.")
				return true
			}
		}
		return true

	case TMonument:
		note := zn.Notes[y*MW+x]
		if note == "" {
			note = "A monumental granite obelisk recording legendary triumphs."
		}
		g.showPopup("Commemorative Monument", note)
		g.gainXP(20)
		return true

	case TGateClosed:
		zn.Set(x, y, TGateArch)
		g.spend(600)
		g.Msg("{{G|Rattle-clank!}} You hoist open the heavy iron portcullis.")
		return true

	case TLantern:
		g.Msg("The oiled paper lantern glows with gentle, fragrant oil light.")
		return true
	}

	return false
}

func ueTalkHook(g *Game, a *Actor) []Option {
	var opts []Option
	if g == nil || a == nil || g.Cur == nil || g.Cur.Kind != "town" {
		return opts
	}

	role := strings.ToLower(a.Role)
	if !strings.Contains(role, "elder") && !strings.Contains(role, "mayor") &&
		!strings.Contains(role, "magistrate") && !strings.Contains(role, "architect") {
		return opts
	}

	townKey := fmt.Sprintf("%d,%d", g.Cur.WX, g.Cur.WY)
	st := g.ue()

	// 1. Victory Monument
	if _, has := st.Monuments[townKey]; !has {
		opts = append(opts, Option{
			Text: "{{Y|Commission Commemorative Victory Monument (300 gold).}}",
			Do: func() {
				if !g.cuSpend(300) {
					g.talkMenu("Carving an obelisk in the plaza requires 300 gold!")
					return
				}
				insc := fmt.Sprintf("Erected in honor of %s, triumphant Wanderer of the realms.", g.P.Name)
				st.Monuments[townKey] = insc
				g.Cur.Set(40, 10, TMonument)
				g.Cur.Notes[10*MW+40] = insc
				g.gainXP(50)
				g.talkMenu("Stonecarvers set to work! A grand victory monument now stands in the plaza.")
			},
		})
	}

	// 2. Private Estate
	if !st.Estates[townKey] {
		opts = append(opts, Option{
			Text: "{{G|Construct Private Walled Estate (1000 gold).}}",
			Do: func() {
				if !g.cuSpend(1000) {
					g.talkMenu("Land rights and masonry for a private estate cost 1000 gold.")
					return
				}
				st.Estates[townKey] = true
				uePlaceCommissionedFeatures(g, g.Cur)
				g.gainXP(100)
				g.talkMenu("Masons and carpenters complete your luxurious private estate!")
			},
		})
	}

	// 3. Guildhall
	if _, has := st.Guildhalls[townKey]; !has {
		opts = append(opts, Option{
			Text: "{{C|Endow Adventurer Guildhall (500 gold).}}",
			Do: func() {
				if !g.cuSpend(500) {
					g.talkMenu("Endowing a chartered guildhall requires 500 gold in charter fees.")
					return
				}
				st.Guildhalls[townKey] = "adventurers"
				uePlaceCommissionedFeatures(g, g.Cur)
				g.gainXP(75)
				g.talkMenu("The Adventurers Guildhall opens its gilded doors to heroes across the realm!")
			},
		})
	}

	return opts
}
