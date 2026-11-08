package engine

import "strings"

// Smarter Foes & Weather in Combat (prefix ca).
// Wounded retreat & regroup, archer kiting, pack/tribal flanking,
// rain dousing fire, fog ranged concealment, and downwind scent detection.

// AddFlag sets a boolean flag on Actor (alias for Set).
func (a *Actor) AddFlag(f string) {
	if a != nil {
		a.Set(f)
	}
}

// Hour returns the current hour of the day (0..23).
func (g *Game) Hour() int {
	if g == nil {
		return 8
	}
	return ((g.Turn + startMinutes) % DayLen) / 60
}

// underRoof reports whether tile (x, y) is sheltered indoors / under a roof.
func (g *Game) underRoof(x, y int) bool {
	if g == nil || g.Cur == nil {
		return false
	}
	if !outdoors(g.Cur) {
		return true
	}
	if !g.Cur.In(x, y) {
		return false
	}
	return roofed(g.Cur.At(x, y))
}

func caHasFlagPrefix(a *Actor, pref string) bool {
	if a == nil || a.Flags == nil {
		return false
	}
	for k, v := range a.Flags {
		if v && strings.HasPrefix(k, pref) {
			return true
		}
	}
	return false
}

// caSmartAct executes tactical combat AI for an actor against tgt:
// 1. Wounded retreat & regroup (bind wounds at distance >= 5).
// 2. Archers keep distance (kite back when target is within 2 tiles).
// 3. Packs & tribal warriors surround / flank opposite an adjacent ally.
func (g *Game) caSmartAct(a *Actor, tgt *Actor) bool {
	if g == nil || g.Cur == nil || a == nil || tgt == nil || a.HP <= 0 || tgt.HP <= 0 {
		return false
	}
	d := dist(a, tgt)

	// 1. Wounded Retreat & Regroup
	if a.HP > 0 && a.HP <= a.MaxHP/4 && !a.Boss && !a.Has("undead") && !a.Has("construct") && a.Tpl != "golem" && a.AI != "follower" && !a.Has("ca_regrouped") {
		a.AddFlag("ca_fleeing")
		if d <= 4 && a.St("ca_flee_steps") < 2 {
			a.AddSt("ca_flee_steps", 2)
			if g.stepToward(a, g.distMap(tgt.X, tgt.Y, a.Has("flying")), true) {
				return true
			}
			// Fallback direct step away if distMap step was blocked
			bestD := d
			bx, by := -1, -1
			for _, dir := range dirs8 {
				nx, ny := a.X+dir[0], a.Y+dir[1]
				nd := maxi(abs(nx-tgt.X), abs(ny-tgt.Y))
				if nd > bestD && g.canEnter(a, nx, ny) {
					bestD, bx, by = nd, nx, ny
				}
			}
			if bx >= 0 {
				a.X, a.Y = bx, by
				return true
			}
		} else {
			// dist(a, tgt) >= 5: bandage wounds and rally
			heal := maxi(2, a.MaxHP/4)
			a.HP = mini(a.MaxHP, a.HP+heal)
			if a.Flags != nil {
				delete(a.Flags, "ca_fleeing")
			}
			a.AddFlag("ca_regrouped")
			if g.visible(a.X, a.Y) {
				g.Msg(g.The(a) + " binds its wounds and rallies!")
			}
			return true
		}
	}

	// 2. Archers Keep Distance (Kiting)
	if a.Missile() != nil && d <= 2 && !a.Has("still") && a.St("ca_kite_cd") == 0 {
		bestD := d
		bx, by := -1, -1
		for _, dir := range dirs8 {
			nx, ny := a.X+dir[0], a.Y+dir[1]
			nd := maxi(abs(nx-tgt.X), abs(ny-tgt.Y))
			if nd > bestD && g.canEnter(a, nx, ny) {
				bestD, bx, by = nd, nx, ny
			}
		}
		if bx >= 0 {
			a.X, a.Y = bx, by
			a.AddSt("ca_kite_cd", 2)
			return true
		}
	}

	// 3. Packs Surround / Flank
	if (a.Has("pack") || caHasFlagPrefix(a, "tr_member:")) && d <= 2 {
		var ally *Actor
		for _, o := range g.Cur.Actors {
			if o == nil || o == a || o == tgt || o.HP <= 0 {
				continue
			}
			if !g.Enemies(a, o) && dist(o, tgt) <= 1 {
				ally = o
				break
			}
		}
		if ally != nil {
			ox := tgt.X - (ally.X - tgt.X)
			oy := tgt.Y - (ally.Y - tgt.Y)
			bestScore := -1
			bx, by := -1, -1
			if d == 1 {
				bestScore = abs(a.X-ally.X) + abs(a.Y-ally.Y)
				if a.X == ox && a.Y == oy {
					bestScore += 10
				}
			}
			for _, dir := range dirs8 {
				nx, ny := a.X+dir[0], a.Y+dir[1]
				if maxi(abs(nx-tgt.X), abs(ny-tgt.Y)) != 1 {
					continue
				}
				if !g.canEnter(a, nx, ny) {
					continue
				}
				sc := abs(nx-ally.X) + abs(ny-ally.Y)
				if nx == ox && ny == oy {
					sc += 10
				}
				if sc > bestScore {
					bestScore, bx, by = sc, nx, ny
				}
			}
			if bx >= 0 {
				a.X, a.Y = bx, by
				g.melee(a, tgt)
				return true
			}
			if d == 1 {
				g.melee(a, tgt)
				return true
			}
		}
	}

	return false
}

func (g *Game) caWeatherKindID() string {
	if g == nil {
		return ""
	}
	w := g.WeatherHere()
	if w.Kind == nil {
		return ""
	}
	return w.Kind.ID
}

func caIsWetWeather(id string) bool {
	switch id {
	case "rain", "storm", "blizzard", "acidrain", "sporerain", "ashrain", "tempest", "fallout":
		return true
	}
	return false
}

func caIsObscuringWeather(id string) bool {
	switch id {
	case "fog", "sandstorm", "ash", "ashfall", "smog", "steamfog", "miasma", "spores":
		return true
	}
	return false
}

// caWeatherTurn runs each turn: rain douses fire on exposed actors, and wind carries scent to downwind predators.
func caWeatherTurn(g *Game) {
	if g == nil || g.Cur == nil || g.Cur.Z != 0 {
		return
	}
	wid := g.caWeatherKindID()
	if caIsWetWeather(wid) {
		if g.P != nil && g.P.St("burning") > 0 && !g.underRoof(g.P.X, g.P.Y) {
			delete(g.P.Status, "burning")
			g.Msg("The rain douses the flames on you.")
		}
		for _, a := range g.Cur.Actors {
			if a != nil && a.HP > 0 && a.St("burning") > 0 && !g.underRoof(a.X, a.Y) {
				delete(a.Status, "burning")
			}
		}
	}
	for _, a := range g.Cur.Actors {
		if a != nil && a.HP > 0 && (a.Faction == "beasts" || a.Has("pack")) {
			_ = g.caScentNotice(a)
		}
	}
}

// caFogConcealShot returns true if fog/sandstorm/ash conceals a ranged shot from dist >= 4,
// preserving the shooter's "hidden" status.
func (g *Game) caFogConcealShot(shooter *Actor, dist int) bool {
	if g == nil || dist < 4 {
		return false
	}
	if !caIsObscuringWeather(g.caWeatherKindID()) {
		return false
	}
	if shooter != nil && shooter.St("hidden") > 0 {
		shooter.AddSt("hidden", maxi(shooter.St("hidden"), 2))
	}
	return true
}

// caWeatherRangedDV returns the +2 ranged concealment DV bonus in fog, sandstorm, or ash at dist >= 4.
func (g *Game) caWeatherRangedDV(target *Actor, dist int) int {
	if g == nil || dist < 4 {
		return 0
	}
	if target != nil && g.underRoof(target.X, target.Y) {
		return 0
	}
	if caIsObscuringWeather(g.caWeatherKindID()) {
		return 2
	}
	return 0
}

// caWindVec returns the deterministic wind direction vector (wx, wy) in {-1, 0, 1} from g.Day() and g.Hour().
func (g *Game) caWindVec() (int, int) {
	if g == nil {
		return 1, 0
	}
	idx := ((g.Day()*7+g.Hour())%8 + 8) % 8
	d := dirs8[idx]
	return d[0], d[1]
}

// caScentNotice checks whether wind carries the player's scent to a downwind predator within 14 tiles.
func (g *Game) caScentNotice(predator *Actor) bool {
	if g == nil || g.P == nil || predator == nil || predator.HP <= 0 {
		return false
	}
	if predator.Faction != "beasts" && !predator.Has("pack") {
		return false
	}
	if g.P.St("almask") > 0 {
		return false
	}
	wid := g.caWeatherKindID()
	if wid == "rain" || wid == "storm" || wid == "tempest" || wid == "acidrain" || wid == "sporerain" {
		return false
	}
	if dist(predator, g.P) > 14 {
		return false
	}
	wx, wy := g.caWindVec()
	dot := (predator.X-g.P.X)*wx + (predator.Y-g.P.Y)*wy
	if dot <= 0 {
		return false
	}
	if predator.Status != nil {
		delete(predator.Status, "asleep")
	}
	predator.AddFlag("ca_scented")
	return true
}

func init() {
	turnHooks = append(turnHooks, caWeatherTurn)
}
