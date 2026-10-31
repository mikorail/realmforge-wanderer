package engine

import "fmt"

// Maneuvers (WP4): active skill-tree nodes listed in Abilities() as "sk_<node>". Melee ones take a tile
// target; skUse is called from usePower with (id, tx, ty) and handles cooldown and the turn itself.

var skSelfTarget = map[string]bool{"tumble": true, "shakeitoff": true, "meditate": true, "staunch": true,
	"trailblazer": true, "butchery": true, "trashdiv": true, "flurry": true, "cleave": true, "sweep": true, "repair": true}

// skAbilities lists the player's unlocked maneuvers for the abilities screen.
func (g *Game) skAbilities() []Ability {
	g.skSync()
	var out []Ability
	for _, nd := range g.skActiveNodes() {
		tg := "tile"
		if skSelfTarget[nd.ID] {
			tg = "self"
		}
		id := "sk_" + nd.ID
		out = append(out, Ability{ID: id, Name: nd.Name, Target: tg, Cool: g.P.Cool[id]})
	}
	return out
}

func (g *Game) skWeapon() *Item {
	ws := g.P.Weapons()
	return ws[0]
}

// skFoeAt returns a living creature at the tile within range r of the player, or nil with a message.
func (g *Game) skFoeAt(tx, ty, r int) *Actor {
	d := g.Cur.ActorAt(tx, ty)
	if d == nil || d == g.P || d.HP <= 0 {
		g.Msg("There is nothing there to target.")
		return nil
	}
	if maxi(abs(tx-g.P.X), abs(ty-g.P.Y)) > r {
		g.Msg("{{K|Too far away.}}")
		return nil
	}
	return d
}

// skSwing makes one main-weapon swing; returns true if it drew blood. extra damage lands only on a hit.
func (g *Game) skSwing(d *Actor, extra int) bool {
	w := g.skWeapon()
	before := d.HP
	g.swing(g.P, d, w)
	hit := d.HP < before
	if hit && d.HP > 0 && extra > 0 {
		g.hurt(d, extra, g.P, w.Name)
	}
	return hit
}

func (g *Game) skAdjacentFoes() []*Actor {
	var out []*Actor
	for _, a := range g.Cur.Actors {
		if a.HP > 0 && a != g.P && dist(a, g.P) == 1 && g.Hostile(a) {
			out = append(out, a)
		}
	}
	return out
}

// skRush moves the player along the line toward (tx,ty) up to n tiles, stopping next to the target.
func (g *Game) skRush(tx, ty, n int) bool {
	moved := 0
	for _, pt := range line(g.P.X, g.P.Y, tx, ty) {
		if pt[0] == tx && pt[1] == ty {
			break
		}
		if moved >= n || !g.Cur.Walkable(pt[0], pt[1]) || g.Cur.ActorAt(pt[0], pt[1]) != nil {
			return false
		}
		g.P.X, g.P.Y = pt[0], pt[1]
		moved++
	}
	return maxi(abs(tx-g.P.X), abs(ty-g.P.Y)) <= 1
}

func (g *Game) skKnock(d *Actor, n int) {
	dx, dy := skSgn(d.X-g.P.X), skSgn(d.Y-g.P.Y)
	for i := 0; i < n; i++ {
		nx, ny := d.X+dx, d.Y+dy
		if !g.Cur.Walkable(nx, ny) || g.Cur.ActorAt(nx, ny) != nil || nx == g.P.X && ny == g.P.Y {
			return
		}
		d.X, d.Y = nx, ny
	}
}

func skSgn(v int) int {
	switch {
	case v > 0:
		return 1
	case v < 0:
		return -1
	}
	return 0
}

// skUse runs the maneuver with the given ability id ("sk_<node>").
func (g *Game) skUse(id string, tx, ty int) {
	if len(id) < 4 || id[:3] != "sk_" {
		return
	}
	nd := skNodeByID[id[3:]]
	if nd == nil || !g.skHas(nd.ID) {
		return
	}
	if g.P.Cool[id] > 0 {
		g.Msg(fmt.Sprintf("%s is recharging (%d turns).", nd.Name, g.P.Cool[id]))
		return
	}
	if !g.skRun(nd.ID, tx, ty) {
		return
	}
	if g.P.Cool == nil {
		g.P.Cool = map[string]int{}
	}
	if nd.Cool > 0 {
		g.P.Cool[id] = nd.Cool
	}
	g.spend(1000)
}

func (g *Game) skRun(n string, tx, ty int) bool {
	p := g.P
	lv := p.Level
	switch n {
	case "lunge", "charge":
		d := g.skFoeAt(tx, ty, 4)
		if d == nil {
			return false
		}
		rng := 3
		bonus := 3 + lv/4
		if n == "charge" {
			rng, bonus = 4, 2+lv/4
		}
		if !g.skRush(d.X, d.Y, rng) {
			g.Msg("{{K|The way is blocked or too far.}}")
			return false
		}
		g.Msg("{{W|You " + map[string]string{"lunge": "lunge", "charge": "charge"}[n] + "!}}")
		g.skSwing(d, bonus)
	case "cleave", "sweep":
		foes := g.skAdjacentFoes()
		if len(foes) == 0 {
			g.Msg("No foes beside you.")
			return false
		}
		g.Msg("{{W|You " + n + " through the foes around you!}}")
		for _, d := range foes {
			if g.skSwing(d, 0) && d.HP > 0 {
				d.AddSt("cracked", 5)
				if n == "sweep" && g.R.Chance(0.4) {
					d.AddSt("prone", 2)
				}
			}
		}
	case "dismember":
		d := g.skFoeAt(tx, ty, 1)
		if d == nil {
			return false
		}
		if g.skSwing(d, 2+lv/4) && d.HP > 0 && g.R.Chance(0.5) {
			g.dismember(d)
		}
	case "bludgeon":
		d := g.skFoeAt(tx, ty, 1)
		if d == nil {
			return false
		}
		if g.skSwing(d, 2+lv/3) && d.HP > 0 && g.R.Chance(0.5) {
			d.AddSt("stunned", 2)
			g.Msg(g.The(d) + " is stunned!")
		}
	case "jab":
		d := g.skFoeAt(tx, ty, 1)
		if d == nil {
			return false
		}
		g.skSwing(d, 0)
		if d.HP > 0 && g.skSwing(d, 1) {
			d.AddSt("dazed", 2)
		}
	case "hobble":
		d := g.skFoeAt(tx, ty, 1)
		if d == nil {
			return false
		}
		if g.skSwing(d, 0) && d.HP > 0 {
			d.AddSt("hobbled", 10)
			d.AddSt("bleeding", 4)
			g.Msg(g.The(d) + " is {{r|hobbled}}.")
		}
	case "flurry":
		foes := g.skAdjacentFoes()
		if len(foes) == 0 {
			g.Msg("No foes beside you.")
			return false
		}
		d := foes[0]
		for i := 0; i < 2 && d.HP > 0; i++ {
			g.melee(p, d)
		}
	case "shieldslam":
		d := g.skFoeAt(tx, ty, 1)
		if d == nil {
			return false
		}
		g.Msg("{{W|You slam your shield into " + g.the(d) + "!}}")
		d.AddSt("stunned", 1)
		g.hurt(d, g.R.Int(1, 3)+lv/4, p, "a shield slam")
		if d.HP > 0 {
			g.skKnock(d, 1)
		}
	case "sureshot":
		m := p.Missile()
		d := g.Cur.ActorAt(tx, ty)
		if m == nil || d == nil || d == p {
			g.Msg("You need a missile weapon and a target.")
			return false
		}
		if !g.useAmmo() {
			g.Msg("You're out of " + g.Sk.Ammo + ".")
			return false
		}
		before := d.HP
		g.shoot(p, tx, ty, m)
		if d.HP > 0 && d.HP < before {
			g.hurt(d, 2+lv/4, p, m.Name)
		}
	case "tumble":
		p.AddSt("tumbling", 8)
		g.Msg("{{c|You tumble and weave.}}")
	case "jump":
		if maxi(abs(tx-p.X), abs(ty-p.Y)) > 4 || !g.Cur.Walkable(tx, ty) || g.Cur.ActorAt(tx, ty) != nil {
			g.Msg("{{K|You cannot land there.}}")
			return false
		}
		p.X, p.Y = tx, ty
		g.Msg("You leap.")
	case "shakeitoff":
		for _, s := range []string{"dazed", "stunned", "shaken", "confused", "sluggish", "hobbled", "ill", "terrified", "disoriented", "shamed", "exhausted", "poisoned"} {
			delete(p.Status, s)
		}
		p.HP = mini(p.MaxHP, p.HP+maxi(2, p.MaxHP/10))
		g.Msg("{{G|You shake it off.}}")
	case "meditate":
		for k := range p.Cool {
			if k != "sk_meditate" {
				p.Cool[k] = 0
			}
		}
		p.HP = mini(p.MaxHP, p.HP+maxi(1, p.MaxHP/10))
		g.Msg("{{c|You breathe, and your focus returns.}}")
	case "stare", "berate", "intimidate":
		d := g.skFoeAt(tx, ty, 8)
		if d == nil {
			return false
		}
		if !g.visible(d.X, d.Y) {
			g.Msg("You cannot see it.")
			return false
		}
		if g.R.Int(1, 20)+p.M(EGO)+g.Skills["persuasion"] < 10+d.Level/2+d.M(WIL) {
			g.Msg(g.The(d) + " is unmoved.")
			return true
		}
		switch n {
		case "stare":
			d.AddSt("shaken", 6)
		case "berate":
			d.AddSt("shamed", 8)
			d.AddSt("shaken", 4)
		case "intimidate":
			d.AddSt("shaken", 8)
			d.AddSt("terrified", 6)
		}
		g.Msg("{{W|" + g.The(d) + " quails.}}")
	case "proselytize":
		d := g.skFoeAt(tx, ty, 6)
		if d == nil {
			return false
		}
		if g.Hostile(d) || d.AI == "follower" || d.Boss || d.Role == "guard" || d.AI == "guard" || d.Level > lv+4 {
			g.Msg(g.The(d) + " will not hear you.")
			return false
		}
		if g.companionCount() >= 2+p.M(EGO) {
			g.Msg("You have as many followers as you can lead.")
			return false
		}
		if g.R.Int(1, 20)+p.M(EGO)+g.Skills["persuasion"]*2 < 12+d.M(WIL) {
			g.Msg(g.The(d) + " shakes their head.")
			return true
		}
		d.AddSt("proselytized", 5)
		g.joinCompanion(d)
	case "butchery":
		return g.skButcher(tx, ty)
	case "trailblazer":
		p.AddSt("trailblazing", 80)
		g.Msg("{{c|You pick up the pace.}}")
	case "repair":
		for _, a := range g.Cur.Actors {
			if a.HP > 0 && a.HP < a.MaxHP && dist(a, p) <= 1 && (a.AI == "follower" || a.Tpl == "golem" || a.Tpl == "turret" || a.Tpl == "crystalgolem") && !g.Hostile(a) {
				a.HP = mini(a.MaxHP, a.HP+8+lv+p.M(INT)*2)
				g.Msg("You tend to " + g.the(a) + ".")
				return true
			}
		}
		g.Msg("Nothing nearby needs repair.")
		return false
	case "staunch":
		if p.St("bleeding") == 0 && p.HP >= p.MaxHP {
			g.Msg("You are in good shape.")
			return false
		}
		delete(p.Status, "bleeding")
		p.HP = mini(p.MaxHP, p.HP+maxi(2, p.MaxHP/8))
		g.Msg("{{G|You staunch your wounds.}}")
	case "trashdiv":
		if g.R.Chance(0.35 + 0.05*float64(g.Skills["customs"])) {
			amt := g.R.Int(2, 8) + lv
			g.give(&Item{ID: g.newID(), Kind: "gold", Name: g.Sk.Currency, Glyph: '$', F: CYellow, Count: amt})
			g.Msg(fmt.Sprintf("{{y|Among the refuse you find %d %s.}}", amt, g.Sk.Currency))
		} else {
			g.Msg("You sift the rubbish and find only rubbish.")
		}
	default:
		g.Msg("Nothing happens.")
		return false
	}
	return true
}

func (g *Game) skButcher(tx, ty int) bool {
	var corpse *Item
	for _, d := range append(dirs8[:len(dirs8):len(dirs8)], [2]int{0, 0}) {
		for _, it := range g.Cur.ItemsAt(g.P.X+d[0], g.P.Y+d[1]) {
			if it.Kind == "corpse" {
				corpse = it
			}
		}
	}
	if corpse == nil {
		g.Msg("There is no corpse within reach.")
		return false
	}
	tag := corpse.Effect
	if tag == "" {
		g.Msg("This one yields nothing worth cooking.")
		return false
	}
	n := 2 + g.Skills["cooking"]/2
	g.give(g.makeIngredient("butchered meat", tag, n, CDarkRed))
	var keep []*Item
	for _, it := range g.Cur.Items {
		if it != corpse {
			keep = append(keep, it)
		}
	}
	g.Cur.Items = keep
	g.Msg(fmt.Sprintf("{{G|You butcher the corpse: %d portions.}}", n))
	return true
}
