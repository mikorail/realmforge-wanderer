package engine

// ---- field of view: recursive shadowcasting ----

func (g *Game) recompute() {
	if g.Cur == nil {
		return
	}
	z := g.Cur
	n := MW * MH
	if len(g.vis) != n {
		g.vis = make([]bool, n)
		g.sensed = make([]bool, n)
	}
	for i := range g.vis {
		g.vis[i] = false
		g.sensed[i] = false
	}
	p := g.P
	los := make([]bool, n)
	radius := 60
	if p.Pow("nearsight") > 0 {
		radius = 8
	}
	los[p.Y*MW+p.X] = true
	for oct := 0; oct < 8; oct++ {
		g.castLight(los, p.X, p.Y, 1, 1.0, 0.0, radius, oct)
	}
	light := p.LightRadius()
	dark := z.Dark || (z.Z == 0 && !g.daylight())
	moon := 0
	if z.Z == 0 && !g.daylight() {
		moon = 4
	}
	dv := 0
	if l := p.Pow("darkvision"); l > 0 {
		dv = 7 + l*2
	}
	if p.Species == "dwarf" || p.Species == "goblin" || p.Species == "orc" || p.Species == "void_elf" || p.Species == "ratfolk" {
		dv = maxi(dv, 5)
	}
	if z.Underwater {
		dark = true
		light = mini(light, 4)
	}
	for i := 0; i < n; i++ {
		if !los[i] {
			continue
		}
		x, y := i%MW, i/MW
		d2 := (x-p.X)*(x-p.X) + (y-p.Y)*(y-p.Y)*4
		r := maxi(maxi(light, dv), moon)
		if !dark || z.Lit[i] || d2 <= 4*r*r || abs(x-p.X) <= 1 && abs(y-p.Y) <= 1 {
			g.vis[i] = true
			z.Seen[i] = 1
		}
	}
	if l := p.Pow("sense"); l > 0 {
		r := 6 + l*3
		for _, a := range z.Actors {
			if abs(a.X-p.X) <= r*2 && abs(a.Y-p.Y) <= r {
				g.sensed[a.Y*MW+a.X] = true
			}
		}
	}
}

var octMul = [4][8]int{
	{1, 0, 0, -1, -1, 0, 0, 1},
	{0, 1, -1, 0, 0, -1, 1, 0},
	{0, 1, 1, 0, 0, -1, -1, 0},
	{1, 0, 0, 1, -1, 0, 0, -1},
}

func (g *Game) castLight(los []bool, cx, cy, row int, start, end float64, radius, oct int) {
	if start < end {
		return
	}
	xx, xy, yx, yy := octMul[0][oct], octMul[1][oct], octMul[2][oct], octMul[3][oct]
	z := g.Cur
	newStart := 0.0
	for j := row; j <= radius; j++ {
		dx, dy := -j-1, -j
		blocked := false
		for dx <= 0 {
			dx++
			X, Y := cx+dx*xx+dy*xy, cy+dx*yx+dy*yy
			ls, rs := (float64(dx)-0.5)/(float64(dy)+0.5), (float64(dx)+0.5)/(float64(dy)-0.5)
			if start < rs {
				continue
			} else if end > ls {
				break
			}
			if X >= 0 && Y >= 0 && X < MW && Y < MH && dx*dx+dy*dy <= radius*radius {
				los[Y*MW+X] = true
			}
			op := z.Opaque(X, Y)
			if blocked {
				if op {
					newStart = rs
					continue
				}
				blocked = false
				start = newStart
			} else if op && j < radius {
				blocked = true
				g.castLight(los, cx, cy, j+1, start, ls, radius, oct)
				newStart = rs
			}
		}
		if blocked {
			break
		}
	}
}

func (g *Game) visible(x, y int) bool {
	if g.Cur == nil || !g.Cur.In(x, y) || len(g.vis) == 0 {
		return false
	}
	return g.vis[y*MW+x]
}

// ---- AI ----

// distMap is a BFS distance field from (x,y) over walkable tiles: monsters roll downhill toward the player.
func (g *Game) distMap(x, y int, swim bool) []int {
	z := g.Cur
	d := make([]int, MW*MH)
	for i := range d {
		d[i] = 1 << 20
	}
	d[y*MW+x] = 0
	q := []int{y*MW + x}
	for len(q) > 0 {
		i := q[0]
		q = q[1:]
		cx, cy := i%MW, i/MW
		for _, dd := range dirs8 {
			nx, ny := cx+dd[0], cy+dd[1]
			if !z.In(nx, ny) {
				continue
			}
			ni := ny*MW + nx
			if d[ni] <= d[i]+1 {
				continue
			}
			t := z.At(nx, ny)
			if !t.D().Walk && t != TDoor || t == TLava || (t == TVoid || t == TChasm) && !swim {
				continue
			}
			d[ni] = d[i] + 1
			q = append(q, ni)
		}
	}
	return d
}

func (g *Game) canEnter(a *Actor, x, y int) bool {
	z := g.Cur
	if !z.In(x, y) {
		return false
	}
	if x == g.P.X && y == g.P.Y {
		return false
	}
	if z.ActorAt(x, y) != nil {
		return false
	}
	t := z.At(x, y)
	if t == TDoor {
		return a.Tpl == "npc" || a.Has("armed")
	}
	if !t.D().Walk {
		return a.St("phased") > 0
	}
	if t == TLava && !a.Has("fire") {
		return false
	}
	if (t == TVoid || t == TChasm) && !a.Has("flying") {
		return false
	}
	w := z.WaterAt(x, y)
	if a.Has("aquatic") && w == 0 && !z.Underwater && a.Tpl != "croc" && a.Tpl != "leech" {
		return false
	}
	if w >= 2 && !z.Underwater && !a.Has("aquatic") && !a.Has("flying") && a.AI != "follower" {
		return false
	}
	return true
}

func (g *Game) stepToward(a *Actor, dm []int, flee bool) bool {
	best := dm[a.Y*MW+a.X]
	bx, by := -1, -1
	for _, d := range dirs8 {
		nx, ny := a.X+d[0], a.Y+d[1]
		if !g.Cur.In(nx, ny) {
			continue
		}
		v := dm[ny*MW+nx]
		if flee {
			v = -v
			if best > 0 {
				best = -best
			}
		}
		if v < best && g.canEnter(a, nx, ny) {
			best, bx, by = v, nx, ny
		}
	}
	if bx < 0 {
		return false
	}
	if g.Cur.At(bx, by) == TDoor {
		g.Cur.Set(bx, by, TDoorOpen)
		return true
	}
	a.X, a.Y = bx, by
	return true
}

func (g *Game) wander(a *Actor) {
	d := dirs8[g.R.Int(0, 7)]
	nx, ny := a.X+d[0], a.Y+d[1]
	if a.AI == "town" || a.AI == "guard" {
		if abs(nx-a.HomeX) > 6 || abs(ny-a.HomeY) > 4 {
			return
		}
	}
	if g.canEnter(a, nx, ny) {
		a.X, a.Y = nx, ny
	}
}

func dist(a, b *Actor) int { return maxi(abs(a.X-b.X), abs(a.Y-b.Y)) }

// aiAct: one action for a non-player actor.
func (g *Game) aiAct(a *Actor) {
	if a.St("frozen") > 0 || a.St("stunned") > 0 {
		return
	}
	if a.St("confused") > 0 && g.R.Chance(0.5) {
		g.wander(a)
		return
	}
	p := g.P
	// pick a target: the player if hostile, else the nearest enemy actor
	var tgt *Actor
	best := 1 << 20
	if g.Hostile(a) && a.AI != "follower" {
		dd := dist(a, p)
		if dd < 30 && (g.visible(a.X, a.Y) || dd < 6) {
			tgt, best = p, dd
		}
	}
	for _, o := range g.Cur.Actors {
		if o == a || o.HP <= 0 {
			continue
		}
		if g.Enemies(a, o) {
			if dd := dist(a, o); dd < best && dd < 14 {
				tgt, best = o, dd
			}
		}
	}
	if a.Has("still") {
		if tgt != nil && best <= 12 {
			if m := a.Missile(); m != nil {
				g.shoot(a, tgt.X, tgt.Y, m)
			}
		}
		return
	}
	if tgt == nil {
		switch a.AI {
		case "follower":
			if dist(a, p) > 2 {
				g.stepToward(a, g.distMap(p.X, p.Y, a.Has("flying")), false)
			} else if g.R.Chance(0.3) {
				g.wander(a)
			}
		case "town", "guard", "wander":
			if g.R.Chance(0.3) {
				g.wander(a)
			}
		case "hostile":
			if g.R.Chance(0.2) {
				g.wander(a)
			}
		}
		return
	}
	// flee when badly hurt (except bosses, undead and constructs)
	if a.HP < a.MaxHP/5 && !a.Boss && !a.Has("undead") && a.Tpl != "golem" && a.AI != "follower" && a.Role != "guard" {
		if a.Has("pack") || g.R.Chance(0.5) {
			g.stepToward(a, g.distMap(tgt.X, tgt.Y, a.Has("flying")), true)
			return
		}
	}
	if best <= 1 {
		g.melee(a, tgt)
		return
	}
	if m := a.Missile(); m != nil && best <= m.Range && g.losClear(a.X, a.Y, tgt.X, tgt.Y) {
		if best >= 3 || g.R.Chance(0.5) {
			g.shoot(a, tgt.X, tgt.Y, m)
			return
		}
	}
	// breath weapons for elemental creatures
	if (a.Has("fire") || a.Has("frost")) && best <= 6 && g.R.Chance(0.15) && g.losClear(a.X, a.Y, tgt.X, tgt.Y) {
		k := "fire"
		if a.Has("frost") {
			k = "frost"
		}
		if g.visible(a.X, a.Y) {
			g.Msg(g.The(a) + " breathes " + k + "!")
		}
		g.explode(tgt.X, tgt.Y, k, a.Level/3, a)
		return
	}
	if a.Has("burrow") && g.R.Chance(0.3) && best > 2 {
		// burrowers pop up near their prey
		for i := 0; i < 8; i++ {
			nx, ny := tgt.X+g.R.Int(-2, 2), tgt.Y+g.R.Int(-2, 2)
			if g.canEnter(a, nx, ny) {
				a.X, a.Y = nx, ny
				if g.visible(nx, ny) {
					g.Msg(g.The(a) + " bursts from the ground!")
				}
				return
			}
		}
	}
	if !g.stepToward(a, g.distMap(tgt.X, tgt.Y, a.Has("flying")), false) {
		g.wander(a)
	}
}

func (g *Game) losClear(x0, y0, x1, y1 int) bool {
	for _, pt := range line(x0, y0, x1, y1) {
		if pt[0] == x1 && pt[1] == y1 {
			return true
		}
		if g.Cur.Opaque(pt[0], pt[1]) {
			return false
		}
	}
	return true
}
