package engine

import "fmt"

// ---- passive helpers (hooks live in game.go) ----

// swimCostMult is the time multiplier for crossing deep water.
func (g *Game) swimCostMult() int {
	if g.P.Pow("swimming") > 0 || g.canSwim() {
		return 1
	}
	return 2
}

// breathCap is the player's maximum breath; diving triples it.
func (g *Game) breathCap() int {
	c := 30 + g.P.M(TOU)*5
	if g.P.Pow("diving") > 0 {
		c *= 3
	}
	return c
}

// tryDive plunges from deep open sea straight to the sea floor below.
func (g *Game) tryDive() bool {
	p, z := g.P, g.Cur
	if p.Pow("diving") == 0 || z == nil || z.Z != 0 || z.Underwater || z.WaterAt(p.X, p.Y) < 2 {
		return false
	}
	idx := g.W.Idx(z.WX, z.WY)
	if !g.W.IsWater(idx) || !g.W.seaOpen[idx] {
		return false
	}
	g.Msg("{{C|You take a deep breath and dive.}}")
	g.enterZone(g.GetZone(z.WX, z.WY, z.ZX, z.ZY, 1), p.X, p.Y, true)
	g.spend(1000)
	return true
}

func (g *Game) canClimb(t Terrain) bool {
	return g.P.Pow("climbing") > 0 && (t == TRock || t == TTree)
}

// climbCost is the time to enter a tile; climbing rock or trees takes 3x.
func (g *Game) climbCost(x, y, base int) int {
	if g.canClimb(g.Cur.At(x, y)) {
		return base * 3
	}
	return base
}

// shieldAV is the AV bonus of a force bubble.
func shieldAV(a *Actor) int {
	if a.St("shielded") > 0 {
		return 3 + a.Level/4
	}
	return 0
}

// applyBodyPowers adds limbs granted by powers (call after powers are set).
func applyBodyPowers(a *Actor) {
	if a.Pow("multiarms") <= 0 {
		return
	}
	for _, bp := range a.Body {
		if bp.Name == "Lower Right Hand" {
			return
		}
	}
	a.Body = append(a.Body, &BodyPart{Name: "Lower Right Hand", Slot: "hand", Limb: true}, &BodyPart{Name: "Lower Left Hand", Slot: "hand", Limb: true})
}

// powerStatusTick handles corroding and fugue statuses; call from statusTick before the generic decrement.
func (g *Game) powerStatusTick(a *Actor) {
	if a == nil || a.Status == nil || a.HP <= 0 {
		return
	}
	if a.St("corroding") > 0 {
		g.hurt(a, g.R.Int(1, 3), nil, "acid")
		if a.HP <= 0 {
			return
		}
	}
	if v, ok := a.Status["fugue"]; ok && v <= 1 && a != g.P {
		if g.visible(a.X, a.Y) {
			g.Msg("{{M|Your echo fades away.}}")
		}
		a.HP = 0
		delete(a.Status, "fugue")
		if g.Cur != nil {
			keep := make([]*Actor, 0, len(g.Cur.Actors))
			for _, o := range g.Cur.Actors {
				if o != a {
					keep = append(keep, o)
				}
			}
			g.Cur.Actors = keep
		}
	}
}

// ---- active powers ----

func sgn(v int) int { return sign(v) }

func (g *Game) foeNear(r int) []*Actor {
	var out []*Actor
	for _, a := range g.Cur.Actors {
		if a.HP > 0 && a != g.P && a.AI != "follower" && dist(a, g.P) <= r && (g.Hostile(a) || a.Angry) {
			out = append(out, a)
		}
	}
	return out
}

// usePower2 resolves the newer powers; it returns whether the power was used.
func (g *Game) usePower2(id string, lv, tx, ty int) bool {
	p := g.P
	switch id {
	case "climbing":
		z := g.Cur
		if z.Z <= 0 {
			g.Msg("There is nothing to climb up out of here.")
			return false
		}
		g.Msg("You scale the walls up to the level above.")
		g.enterZone(g.GetZone(z.WX, z.WY, z.ZX, z.ZY, z.Z-1), p.X, p.Y, true)
	case "clipping":
		dx, dy := sgn(tx), sgn(ty)
		if dx == 0 && dy == 0 {
			return false
		}
		z := g.Cur
		blocked := false
		lx, ly, found := 0, 0, false
		for i := 1; i <= 3+lv; i++ {
			x, y := p.X+dx*i, p.Y+dy*i
			if !z.In(x, y) {
				break
			}
			t := z.At(x, y)
			free := z.Walkable(x, y) && z.ActorAt(x, y) == nil && t != TLava && t != TChasm && t != TVoid && (z.WaterAt(x, y) < 2 || z.Underwater)
			if !z.Walkable(x, y) {
				blocked = true
			}
			if free {
				lx, ly, found = x, y, true
				if blocked {
					break
				}
			}
		}
		if !found {
			g.Msg("There is nowhere to step through to.")
			return false
		}
		p.X, p.Y = lx, ly
		g.Msg("{{M|You slip through the world.}}")
	case "arcanebolt":
		a := g.Cur.ActorAt(tx, ty)
		if a == nil || dist(a, p) > 8+lv {
			g.Msg("Nothing there to strike.")
			return false
		}
		dmg := g.R.Dice(1+lv/2, 6) + lv
		g.Msg(fmt.Sprintf("{{C|A bolt of force strikes %s for %d.}}", g.the(a), dmg))
		g.hurt(a, dmg, p, "a force bolt")
		g.provoke(a)
	case "slumber":
		a := g.Cur.ActorAt(tx, ty)
		if a == nil || a.Boss || dist(a, p) > 8+lv {
			g.Msg("Nothing there you can put to sleep.")
			return false
		}
		if g.R.Int(1, 20)+lv+p.M(WIL) >= a.MA()+a.Level/2+8 {
			a.AddSt("stunned", 12+lv*3)
			g.Msg(fmt.Sprintf("{{C|%s falls fast asleep.}}", g.The(a)))
		} else {
			g.Msg(g.The(a) + " resists your lullaby.")
			g.provoke(a)
		}
	case "glimmer":
		g.Msg("{{Y|A blinding light bursts from you!}}")
		for _, a := range g.Cur.Actors {
			if a.HP > 0 && a != p && dist(a, p) <= 1+lv/3 {
				a.AddSt("confused", 5+lv)
				if g.Hostile(a) {
					g.provoke(a)
				}
			}
		}
		g.computeLight(g.Cur)
	case "forcepush":
		a := g.Cur.ActorAt(tx, ty)
		if a == nil || dist(a, p) > 8+lv {
			g.Msg("Nothing there to push.")
			return false
		}
		dx, dy := sgn(a.X-p.X), sgn(a.Y-p.Y)
		moved := 0
		for i := 0; i < 4; i++ {
			nx, ny := a.X+dx, a.Y+dy
			if o := g.Cur.ActorAt(nx, ny); o != nil || !g.Cur.Walkable(nx, ny) {
				dmg := g.R.Dice(1, 6) + lv
				g.Msg(fmt.Sprintf("{{R|%s slams into an obstacle for %d!}}", g.The(a), dmg))
				g.hurt(a, dmg, p, "a force push")
				if o != nil && o != a {
					g.hurt(o, dmg/2, p, "a force push")
				}
				break
			}
			a.X, a.Y = nx, ny
			moved++
		}
		if moved > 0 {
			g.Msg(fmt.Sprintf("%s is hurled back %d.", g.The(a), moved))
		}
		g.provoke(a)
	case "clairvoyance":
		for i := range g.Cur.Seen {
			g.Cur.Seen[i] = 1
		}
		g.Msg("{{M|The whole place unfolds in your mind's eye.}}")
	case "timedilation":
		n := 0
		for _, a := range g.foeNear(10) {
			a.AddSt("slowed", 10+lv*3)
			n++
		}
		p.AddSt("hasted", 4+lv)
		g.Msg(fmt.Sprintf("{{C|Time thickens around your enemies. (%d slowed)}}", n))
	case "forcebubble":
		p.AddSt("shielded", 15+lv*5)
		g.Msg("{{C|A shimmering bubble of force surrounds you.}}")
	case "syphon":
		a := g.Cur.ActorAt(tx, ty)
		if a == nil || dist(a, p) > 4+lv/2 {
			g.Msg("Too far.")
			return false
		}
		dmg := g.R.Dice(2+lv, 6)
		g.Msg(fmt.Sprintf("You syphon {{R|%d}} vim from %s.", dmg, g.the(a)))
		g.hurt(a, dmg, p, "vim syphon")
		p.HP = mini(p.MaxHP, p.HP+dmg)
	case "stunforce":
		n := 0
		for _, a := range g.Cur.Actors {
			if a.HP > 0 && a != p && a.AI != "follower" && dist(a, p) <= 2+lv/3 {
				a.AddSt("stunned", 3+lv/2)
				g.provoke(a)
				n++
			}
		}
		g.Msg(fmt.Sprintf("{{C|A shockwave rolls outward. (%d stunned)}}", n))
	case "disintegrate":
		g.Msg("{{R|Reality unravels around you!}}")
		for dy := -2; dy <= 2; dy++ {
			for dx := -2; dx <= 2; dx++ {
				x, y := p.X+dx, p.Y+dy
				if !g.Cur.In(x, y) || x == 0 || y == 0 || x == MW-1 || y == MH-1 {
					continue
				}
				if t := g.Cur.At(x, y); t == TWall || t == TRock || t == TCrystal || t == TTree || t == TDoor || t == TLocked {
					g.Cur.Set(x, y, TRubble)
				}
			}
		}
		for _, a := range append([]*Actor{}, g.Cur.Actors...) {
			if a.HP > 0 && a != p && a.AI != "follower" && dist(a, p) <= 2 {
				g.hurt(a, g.R.Dice(2+lv/2, 8), p, "disintegration")
			}
		}
		g.computeLight(g.Cur)
	case "corrosive":
		if !g.Cur.In(tx, ty) {
			return false
		}
		for dy := -2; dy <= 2; dy++ {
			for dx := -2; dx <= 2; dx++ {
				x, y := tx+dx, ty+dy
				if !g.Cur.In(x, y) || dx*dx+dy*dy > 5 {
					continue
				}
				if g.Cur.Walkable(x, y) {
					g.Cur.Liq[y*MW+x] = 3
				}
				if a := g.Cur.ActorAt(x, y); a != nil && a != p && a.AI != "follower" {
					a.AddSt("corroding", 5+lv)
					g.provoke(a)
				}
			}
		}
		g.Msg("{{G|A hissing cloud of acid billows out.}}")
	case "fugue":
		n := 1
		if lv >= 3 {
			n = 2
		}
		made := 0
		for i := 0; i < n; i++ {
			x, y, ok := g.nearestFree(g.Cur, p.X+1, p.Y)
			if !ok || (x == p.X && y == p.Y) {
				continue
			}
			c := &Actor{ID: g.newID(), Name: p.Name + "'s echo", Proper: true, Glyph: '@', F: CDarkGray, Tpl: "fugue", Faction: "player",
				Level: p.Level, Attr: p.Attr, MaxHP: maxi(8, p.MaxHP/2), AVNat: p.AV(), DVNat: p.DV() - 6 - p.M(AGI), Speed: 100,
				AI: "follower", Body: MakeBody("humanoid"), X: x, Y: y}
			c.HP = c.MaxHP
			c.Set("armed")
			if hp := c.PartFor("hand"); hp != nil {
				if w := p.Missile(); w == nil {
					hp.Natural = natural("strike", 1, 6, 2+p.Level/4, "")
				}
				for _, bp := range p.Body {
					if bp.Slot == "hand" && bp.Item != nil && bp.Item.Kind == "weapon" {
						cp := *bp.Item
						hp.Item = &cp
						break
					}
				}
			}
			c.Status = map[string]int{"fugue": 20 + lv*8}
			g.Cur.Actors = append(g.Cur.Actors, c)
			made++
		}
		if made == 0 {
			g.Msg("There is no room for your echoes.")
			return false
		}
		g.Msg("{{M|Time folds, and your echoes step out of it.}}")
	default:
		return false
	}
	g.recompute()
	return true
}
