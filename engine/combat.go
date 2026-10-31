package engine

import (
	"fmt"
	"strings"
)

// melee swings every wielded weapon / natural attack, Qud-style: primary always, off-hands by chance.
func (g *Game) melee(a, d *Actor) {
	ws := a.Weapons()
	for i, w := range ws {
		if d.HP <= 0 || a.HP <= 0 {
			return
		}
		if i > 0 {
			chance := 0.35
			if a == g.P && g.Skills["shortblade"] >= 2 && w.Skill == "shortblade" {
				chance = 0.65
			}
			if w.Skill == "natural" && a != g.P {
				chance = 0.6
			}
			if !g.R.Chance(chance) {
				continue
			}
		}
		g.swing(a, d, w)
	}
	if a == g.P && a.Pow("stinger") > 0 {
		// already included as natural tail
	}
	if d.HP > 0 && d.Pow("spikes") > 0 || d.HP > 0 && d == g.P && g.P.Pow("spikes") > 0 {
		g.hurt(a, g.R.Int(1, 3)+d.Pow("spikes"), d, "spines")
	}
	if a != g.P && d != g.P && !g.visible(a.X, a.Y) {
		return
	}
}

func (g *Game) skillLv(a *Actor, s string) int {
	if a == g.P {
		return g.Skills[s]
	}
	return a.Level / 5
}

func (g *Game) swing(a, d *Actor, w *Item) {
	sk := g.skillLv(a, w.Skill)
	roll := g.R.Int(1, 20)
	hit := roll + a.M(AGI) + sk
	if a.St("confused") > 0 {
		hit -= 4
	}
	crit := roll == 20
	if !crit && (roll == 1 || hit < d.DV()) || d.St("phased") > 0 && a.St("phased") == 0 {
		if a == g.P || d == g.P {
			g.Msg(fmt.Sprintf("%s %s %s.", g.The(a), verb(a == g.P, "miss", "misses"), g.the(d)))
		}
		return
	}
	pv := w.PV + mini(a.M(STR), w.MaxStr)
	if crit {
		pv++
	}
	if a == g.P && sk >= 3 {
		pv++
	}
	pens := g.penetrate(pv, d.AV(), crit)
	if pens == 0 {
		if a == g.P || d == g.P {
			g.Msg(fmt.Sprintf("%s %s %s with %s, but %s.", g.The(a), verb(a == g.P, "hit", "hits"), g.the(d), w.Name, "{{K|fail to penetrate the armor}}"))
		}
		return
	}
	dmg := 0
	for i := 0; i < pens; i++ {
		dmg += g.R.Dice(maxi(w.DN, 1), maxi(w.DS, 1)) + w.DB
	}
	dmg += g.categoryBonusDamage(w, d)
	if a.St("mighty") > 0 {
		dmg += 2
	}
	col := "R"
	if d == g.P {
		col = "r"
	}
	critS := ""
	if crit {
		critS = " {{W|Critical!}}"
	}
	if a == g.P || d == g.P || g.visible(d.X, d.Y) {
		g.Msg(fmt.Sprintf("%s %s %s (x%d) for {{%s|%d}} damage with %s.%s", g.The(a), verb(a == g.P, "hit", "hits"), g.the(d), pens, col, dmg, w.Name, critS))
	}
	g.hurt(d, dmg, a, w.Name)
	if d.HP <= 0 {
		return
	}
	g.onHit(a, d, w, crit, sk)
}

func (g *Game) onHit(a, d *Actor, w *Item, crit bool, sk int) {
	switch w.Elem {
	case "fire":
		if g.R.Chance(0.35) {
			d.AddSt("burning", 4)
			g.Msg(g.The(d) + " catches fire!")
		}
	case "frost":
		if g.R.Chance(0.25) {
			d.AddSt("frozen", 2)
			g.Msg(g.The(d) + " is frozen!")
		}
	case "shock":
		g.hurt(d, g.R.Int(1, 4), a, "a shock")
	case "acid":
		g.hurt(d, g.R.Int(1, 3), a, "acid")
		if g.Cur.In(d.X, d.Y) {
			g.Cur.Liq[d.Y*MW+d.X] = 3
		}
	case "poison":
		if g.R.Chance(0.4) {
			d.AddSt("poisoned", 8)
			if d == g.P {
				g.Msg("{{G|You are poisoned!}}")
			}
		}
	}
	bleed := a.Has("bleed") || a == g.P && w.Skill == "longblade" && sk >= 2 && crit || w.Skill == "shortblade" && crit
	if bleed && !d.Has("undead") && d.Tpl != "golem" {
		d.AddSt("bleeding", 6)
		if d == g.P || g.visible(d.X, d.Y) {
			g.Msg(g.The(d) + verb(d == g.P, " are", " is") + " {{r|bleeding}}.")
		}
	}
	if w.Skill == "cudgel" && (sk >= 2 || crit) && g.R.Chance(0.25) {
		d.AddSt("stunned", 2)
		g.Msg(g.The(d) + verb(d == g.P, " are", " is") + " stunned!")
	}
	if w.Skill == "axe" && (sk >= 2 && g.R.Chance(0.12) || crit && g.R.Chance(0.4)) {
		g.dismember(d)
	}
	if a.Has("leech") && a.HP < a.MaxHP {
		a.HP = mini(a.MaxHP, a.HP+2)
	}
}

// penetrate: up to three PV rolls vs AV per round; three successes chain into another round at -2 PV.
func (g *Game) penetrate(pv, av int, crit bool) int {
	total := 0
	for round := 0; round < 4; round++ {
		n := 0
		for i := 0; i < 3; i++ {
			if g.R.Int(1, 10)-2+pv > av {
				n++
			}
		}
		total += n
		if n < 3 {
			break
		}
		pv -= 2
	}
	if total == 0 && crit {
		total = 1
	}
	return total
}

func (g *Game) dismember(d *Actor) {
	var limbs []*BodyPart
	for _, p := range d.Body {
		if p.Limb && !p.Severed && p.Slot != "head" {
			limbs = append(limbs, p)
		}
	}
	if len(limbs) == 0 {
		return
	}
	p := Pick(g.R, limbs)
	p.Severed = true
	g.Msg(fmt.Sprintf("{{R|%s %s is severed!}}", possessive(g, d), strings.ToLower(p.Name)))
	limb := &Item{ID: g.newID(), Kind: "trophy", Name: strings.ToLower(possessive(g, d)) + " " + strings.ToLower(p.Name), Glyph: '~', F: CDarkRed, X: d.X, Y: d.Y, Value: 1}
	g.Cur.Items = append(g.Cur.Items, limb)
	if p.Item != nil {
		it := p.Item
		p.Item = nil
		if d == g.P {
			it.X, it.Y = d.X, d.Y
		} else {
			it.X, it.Y = d.X, d.Y
		}
		g.Cur.Items = append(g.Cur.Items, it)
	}
	d.AddSt("bleeding", 10)
}

func possessive(g *Game, a *Actor) string {
	if a == g.P {
		return "Your"
	}
	return g.The(a) + "'s"
}

func verb(you bool, a, b string) string {
	if you {
		return a
	}
	return b
}

// hurt applies damage and handles death; src may be nil.
func (g *Game) hurt(d *Actor, n int, src *Actor, cause string) {
	if d.HP <= 0 || n <= 0 {
		return
	}
	d.HP -= n
	if src == g.P && d != g.P && !g.Hostile(d) {
		g.provoke(d)
	}
	if src != nil && src != g.P && d != g.P && d.AI != "follower" {
		d.Angry = d.Angry || src.AI == "follower"
	}
	if g.Cur.In(d.X, d.Y) && n >= 3 && !d.Has("undead") && d.Tpl != "golem" && g.R.Chance(0.5) {
		g.Cur.Liq[d.Y*MW+d.X] = 1
	}
	if d.HP > 0 {
		if d == g.P && d.HP < d.MaxHP/4 {
			g.Msg("{{R|!!! Your health is critically low !!!}}")
		}
		return
	}
	if d == g.P {
		g.die(cause, src)
		return
	}
	g.kill(d, src)
}

func (g *Game) provoke(d *Actor) {
	if d.Angry {
		return
	}
	d.Angry = true
	if strings.HasPrefix(d.Faction, "state:") {
		g.AddRep(d.Faction, -50, " for attacking "+g.the(d))
		for _, o := range g.Cur.Actors {
			if o.Faction == d.Faction && o.AI == "guard" {
				o.Angry = true
			}
		}
	}
}

func (g *Game) kill(d, src *Actor) {
	if g.visible(d.X, d.Y) || src == g.P {
		g.Msg(fmt.Sprintf("{{R|%s %s.}}", g.The(d), deathVerb(d)))
	}
	z := g.Cur
	for _, p := range d.Body {
		if p.Item != nil && g.R.Chance(0.7) || p.Item != nil && p.Item.Kind == "artifact" {
			it := p.Item
			it.X, it.Y = d.X, d.Y
			z.Items = append(z.Items, it)
		}
	}
	for _, it := range d.Inv {
		if d.Role == "merchant" || d.Role == "smith" || d.Role == "apothecary" {
			if !g.R.Chance(0.3) {
				continue
			}
		}
		it.X, it.Y = d.X, d.Y
		z.Items = append(z.Items, it)
	}
	if d.Gold > 0 {
		z.Items = append(z.Items, &Item{ID: g.newID(), Kind: "gold", Name: g.Sk.Currency, Glyph: '$', F: CYellow, Count: d.Gold, X: d.X, Y: d.Y})
	}
	if !d.Has("undead") && d.Tpl != "golem" && d.Tpl != "turret" {
		z.Items = append(z.Items, &Item{ID: g.newID(), Kind: "corpse", Name: d.Name + " corpse", Glyph: '%', F: CDarkRed, X: d.X, Y: d.Y, Power: 200})
	}
	if src == g.P || src != nil && src.AI == "follower" {
		g.Kills++
		g.gainXP(d.XP)
		if strings.HasPrefix(d.Faction, "state:") {
			g.AddRep(d.Faction, -100, " for murder")
		} else if d.Faction != "" && d.Faction != "player" {
			for _, s := range g.W.States {
				_ = s
			}
		}
		if d.SimMon > 0 {
			g.Slain[d.SimMon] = true
			txt := fmt.Sprintf("%s the %s slew %s.", g.P.Name, g.Sk.Calling(g.Calling), d.Name)
			g.deed("slay", txt, d.SimMon, 0)
			g.Msg("{{W|The chronicles will remember this: you have slain " + d.Name + "!}}")
			g.gainRepAll(100, " for slaying "+d.Name)
		}
		if d.SimPer > 0 {
			g.deed("murder", fmt.Sprintf("%s was slain by the wanderer %s.", d.Name, g.P.Name), 0, 0)
		}
		g.questKill(d)
	}
	for _, h := range killHooks {
		h(g, d, src)
	}
}

func (g *Game) gainRepAll(n int, why string) {
	p := g.W.Idx(g.Cur.WX, g.Cur.WY)
	f := fmt.Sprintf("state:%d", g.W.State[p])
	if g.W.State[p] > 0 {
		g.AddRep(f, n, why)
	}
}

func deathVerb(a *Actor) string {
	if a.Has("undead") {
		return "crumbles to dust"
	}
	if a.Tpl == "golem" || a.Tpl == "turret" || a.Tpl == "crystalgolem" {
		return "is destroyed"
	}
	return "dies"
}

func (g *Game) die(cause string, src *Actor) {
	g.Dead = true
	who := cause
	if src != nil {
		who = g.the(src)
	}
	g.DeathText = fmt.Sprintf("You were killed by %s in %s, %s.", who, g.Cur.Name, g.timeOfDay())
	g.Msg("{{R|" + g.DeathText + "}}")
	g.deed("death", fmt.Sprintf("The wanderer %s, %s, perished in %s, killed by %s.", g.P.Name, g.Sk.Calling(g.Calling), g.Cur.Name, who), 0, 0)
	g.Mode = "dead"
}

// ---- ranged ----

// line returns the Bresenham path from a to b (excluding a).
func line(x0, y0, x1, y1 int) [][2]int {
	var out [][2]int
	dx, dy := abs(x1-x0), -abs(y1-y0)
	sx, sy := sign(x1-x0), sign(y1-y0)
	err := dx + dy
	x, y := x0, y0
	for i := 0; i < 200; i++ {
		if x == x1 && y == y1 {
			break
		}
		e2 := 2 * err
		if e2 >= dy {
			err += dy
			x += sx
		}
		if e2 <= dx {
			err += dx
			y += sy
		}
		out = append(out, [2]int{x, y})
	}
	return out
}

// shoot fires a missile weapon along a line, hitting the first creature.
func (g *Game) shoot(a *Actor, tx, ty int, w *Item) {
	path := line(a.X, a.Y, tx, ty)
	rng := w.Range
	if a == g.P {
		rng += g.Skills["missile"] * 2
	}
	for i, pt := range path {
		if i >= rng {
			break
		}
		x, y := pt[0], pt[1]
		if g.Cur.Opaque(x, y) {
			if a == g.P {
				g.Msg("The shot strikes " + g.Cur.At(x, y).D().Name + ".")
			}
			return
		}
		var d *Actor
		if x == g.P.X && y == g.P.Y {
			d = g.P
		} else {
			d = g.Cur.ActorAt(x, y)
		}
		if d == nil || d == a {
			continue
		}
		roll := g.R.Int(1, 20)
		hit := roll + a.M(AGI) + g.skillLv(a, "missile") - i/4
		if roll != 20 && hit < d.DV() {
			if a == g.P || d == g.P {
				g.Msg(fmt.Sprintf("%s %s %s.", g.The(a), verb(a == g.P, "miss", "misses"), g.the(d)))
			}
			continue
		}
		pens := g.penetrate(w.PV, d.AV(), roll == 20)
		if pens == 0 {
			if a == g.P || d == g.P {
				g.Msg(fmt.Sprintf("The shot glances off %s.", g.the(d)))
			}
			return
		}
		dmg := 0
		for k := 0; k < pens; k++ {
			dmg += g.R.Dice(w.DN, w.DS)
		}
		g.Msg(fmt.Sprintf("%s %s %s (x%d) for {{R|%d}} damage.", g.The(a), verb(a == g.P, "shoot", "shoots"), g.the(d), pens, dmg))
		g.hurt(d, dmg, a, w.Name)
		return
	}
}

// explode: thrown grenades and fire powers
func (g *Game) explode(x, y int, kind string, power int, src *Actor) {
	rad := 1
	if kind == "flash" {
		rad = 2
	}
	for dy := -rad; dy <= rad; dy++ {
		for dx := -rad; dx <= rad; dx++ {
			nx, ny := x+dx, y+dy
			if !g.Cur.In(nx, ny) {
				continue
			}
			var d *Actor
			if nx == g.P.X && ny == g.P.Y {
				d = g.P
			} else {
				d = g.Cur.ActorAt(nx, ny)
			}
			switch kind {
			case "fire":
				if g.Cur.At(nx, ny).D().Burn || g.Cur.Liq[ny*MW+nx] == 2 || g.R.Chance(0.4) {
					g.Cur.Fire[ny*MW+nx] = uint8(g.R.Int(4, 8))
				}
				if d != nil {
					d.AddSt("burning", 4)
					g.hurt(d, g.R.Dice(2, 4)+power, src, "fire")
				}
			case "frost":
				if d != nil {
					d.AddSt("frozen", 3)
					g.hurt(d, g.R.Dice(1, 6)+power, src, "frost")
				}
				if g.Cur.At(nx, ny) == TShallow || g.Cur.At(nx, ny) == TDeep {
					g.Cur.Set(nx, ny, TIce)
				}
			case "flash":
				if d != nil {
					d.AddSt("stunned", 3)
					d.AddSt("confused", 4)
				}
			}
		}
	}
	g.computeLight(g.Cur)
}
