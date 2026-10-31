package engine

import (
	"fmt"
	"sort"
	"strings"
)

type Ability struct {
	ID     string
	Name   string
	Target string
	Cool   int
	Item   *Item
}

// Abilities lists activatable powers and charged artifacts.
func (g *Game) Abilities() []Ability {
	var out []Ability
	var ids []string
	for id, lv := range g.P.Powers {
		if lv > 0 {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	for _, id := range ids {
		pd := powerByID[id]
		if pd == nil || !pd.Active {
			continue
		}
		out = append(out, Ability{ID: id, Name: g.Sk.PowerName(id), Target: pd.Target, Cool: g.P.Cool[id]})
	}
	all := append([]*Item{}, g.P.Inv...)
	for _, bp := range g.P.Body {
		if bp.Item != nil {
			all = append(all, bp.Item)
		}
	}
	for _, it := range all {
		if it.Kind == "artifact" && it.Effect != "" && it.Ident && it.Charges > 0 {
			pd := powerByID[it.Effect]
			tg := "self"
			if pd != nil {
				tg = pd.Target
			}
			out = append(out, Ability{ID: it.Effect, Name: fmt.Sprintf("%s (%d)", it.Name, it.Charges), Target: tg, Item: it})
		}
	}
	return out
}

func (g *Game) startAbility(ab Ability) {
	if ab.Item == nil && g.P.Cool[ab.ID] > 0 {
		g.Msg(fmt.Sprintf("%s is recharging (%d turns).", ab.Name, g.P.Cool[ab.ID]))
		g.Mode = "main"
		return
	}
	src := "power:" + ab.ID
	if ab.Item != nil {
		src = fmt.Sprintf("art:%d", ab.Item.ID)
	}
	switch ab.Target {
	case "self":
		g.Mode = "main"
		g.usePower(ab.ID, ab.Item, 0, 0)
	case "dir":
		g.Mode = "dir"
		g.Pend = src
		g.Msg("Which direction?")
	default:
		g.beginTarget(src)
	}
}

func (g *Game) powerLevel(id string, it *Item) int {
	if it != nil {
		return maxi(it.Power, 2)
	}
	return maxi(g.P.Pow(id), 1) + g.P.Level/4
}

// usePower resolves an active power at a target tile (or a direction for dir powers).
func (g *Game) usePower(id string, it *Item, tx, ty int) {
	p := g.P
	lv := g.powerLevel(id, it)
	name := g.Sk.PowerName(id)
	if it != nil {
		it.Charges--
		name = it.Name
	}
	pd := powerByID[id]
	used := true
	switch id {
	case "flames", "freeze":
		kind := map[string]string{"flames": "fire", "freeze": "frost"}[id]
		g.Msg(fmt.Sprintf("You %s {{R|%s}}!", g.Sk.PowerVerb, name))
		for i, pt := range line(p.X, p.Y, tx, ty) {
			if i > 8+lv || g.Cur.Opaque(pt[0], pt[1]) {
				break
			}
			if a := g.Cur.ActorAt(pt[0], pt[1]); a != nil {
				g.explode(pt[0], pt[1], kind, lv, p)
				break
			}
			if kind == "fire" && g.Cur.At(pt[0], pt[1]).D().Burn {
				g.Cur.Fire[pt[1]*MW+pt[0]] = uint8(g.R.Int(3, 6))
			}
			if i == 8+lv || pt[0] == tx && pt[1] == ty {
				g.explode(pt[0], pt[1], kind, lv, p)
				break
			}
		}
	case "lightning":
		g.Msg("{{C|Lightning leaps from you!}}")
		cur := g.Cur.ActorAt(tx, ty)
		hit := map[*Actor]bool{}
		for k := 0; k < 2+lv/2 && cur != nil; k++ {
			hit[cur] = true
			dmg := g.R.Dice(2+lv/2, 6)
			g.Msg(fmt.Sprintf("The bolt strikes %s for {{C|%d}}.", g.the(cur), dmg))
			g.hurt(cur, dmg, p, "lightning")
			var next *Actor
			for _, o := range g.Cur.Actors {
				if o.HP > 0 && !hit[o] && dist(o, cur) <= 4 && o.AI != "follower" {
					next = o
					break
				}
			}
			cur = next
		}
	case "psiblast":
		if a := g.Cur.ActorAt(tx, ty); a != nil {
			if g.R.Int(1, 20)+lv+p.M(EGO) >= a.MA()+10 {
				dmg := g.R.Dice(lv, 8)
				g.Msg(fmt.Sprintf("{{M|Your mind lances into %s for %d!}}", g.the(a), dmg))
				g.hurt(a, dmg, p, "a psionic blast")
				a.AddSt("stunned", 1)
			} else {
				g.Msg(g.The(a) + " resists your mind.")
				g.hurt(a, 0, p, "")
				g.provoke(a)
			}
		}
	case "leech":
		if a := g.Cur.ActorAt(tx, ty); a != nil && dist(a, p) <= 3+lv/2 {
			dmg := g.R.Dice(1+lv/2, 6)
			g.Msg(fmt.Sprintf("You drain {{R|%d}} life from %s.", dmg, g.the(a)))
			g.hurt(a, dmg, p, "life drain")
			p.HP = mini(p.MaxHP, p.HP+dmg)
		} else {
			g.Msg("Too far.")
			used = false
		}
	case "teleport":
		if g.Cur.Walkable(tx, ty) && g.Cur.ActorAt(tx, ty) == nil && g.visible(tx, ty) {
			p.X, p.Y = tx, ty
			g.Msg("{{M|You blink across space.}}")
		} else {
			g.Msg("You can't teleport there.")
			used = false
		}
	case "phase":
		p.AddSt("phased", 6+lv*2)
		g.Msg("{{M|You phase out of sync with the world.}}")
	case "sprint":
		p.AddSt("sprint", 5+lv*2)
		g.Msg("{{C|You surge forward!}}")
	case "heal":
		h := g.R.Dice(lv, 8) + 4
		p.HP = mini(p.MaxHP, p.HP+h)
		delete(p.Status, "bleeding")
		delete(p.Status, "poisoned")
		g.Msg(fmt.Sprintf("{{G|You heal %d.}}", h))
	case "beguile":
		a := g.Cur.ActorAt(tx, ty)
		if a == nil || a.Boss {
			g.Msg("Nothing there you can beguile.")
			used = false
			break
		}
		if g.R.Int(1, 20)+lv+p.M(EGO) >= a.MA()+a.Level/2+8 {
			a.AI, a.Faction, a.F, a.Angry = "follower", "player", CGreen, false
			g.Msg(fmt.Sprintf("{{G|%s is now your loyal companion.}}", g.The(a)))
		} else {
			g.Msg(g.The(a) + " shakes off your influence.")
			g.provoke(a)
		}
	case "forcewall":
		cx, cy := p.X+tx, p.Y+ty
		for k := -2; k <= 2; k++ {
			x, y := cx+k*abs(ty), cy+k*abs(tx)
			if tx != 0 && ty != 0 {
				x, y = cx+k, cy-k*sign(tx)*sign(ty)
			}
			if g.Cur.Walkable(x, y) && g.Cur.ActorAt(x, y) == nil {
				g.Cur.Set(x, y, TWall)
			}
		}
		g.Msg("A shimmering wall of force rises.")
	case "burrow":
		x, y := p.X+tx, p.Y+ty
		if t := g.Cur.At(x, y); t == TRock || t == TWall || t == TCrystal || t == TTree {
			g.dig(x, y)
			return
		}
		g.Msg("Nothing to dig there.")
		used = false
	}
	if used && it == nil && pd != nil {
		p.Cool[id] = maxi(5, pd.Cool-lv*2)
	}
	if it != nil && it.Charges <= 0 {
		g.Msg("The " + it.Name + " goes dark, spent.")
		if bp := p.Equipped(it); bp != nil {
			bp.Item = nil
		} else {
			g.removeInv(it)
		}
	}
	if used {
		g.spend(1000)
	}
}

// ---- targeting ----

func (g *Game) beginTarget(pend string) {
	g.Pend = pend
	g.Mode = "target"
	g.CX, g.CY = g.P.X, g.P.Y
	if t := g.nearestTarget(-1); t != nil {
		g.CX, g.CY = t.X, t.Y
	}
}

func (g *Game) nearestTarget(skip int) *Actor {
	var cands []*Actor
	for _, a := range g.Cur.Actors {
		if a.HP > 0 && g.visible(a.X, a.Y) && g.Hostile(a) {
			cands = append(cands, a)
		}
	}
	sort.Slice(cands, func(i, j int) bool { return dist(cands[i], g.P) < dist(cands[j], g.P) })
	if len(cands) == 0 {
		return nil
	}
	if skip >= 0 {
		return cands[(skip+1)%len(cands)]
	}
	return cands[0]
}

func (g *Game) finishTarget() {
	pend := g.Pend
	g.Pend = ""
	g.Mode = "main"
	tx, ty := g.CX, g.CY
	if tx == g.P.X && ty == g.P.Y && !strings.HasPrefix(pend, "look") {
		g.Msg("You can't target yourself.")
		return
	}
	switch {
	case pend == "fire":
		m := g.P.Missile()
		if m == nil {
			return
		}
		if !g.useAmmo() {
			g.Msg("You're out of " + g.Sk.Ammo + ".")
			return
		}
		g.shoot(g.P, tx, ty, m)
		g.spend(1000)
	case strings.HasPrefix(pend, "throw:"):
		var id int
		fmt.Sscanf(pend[6:], "%d", &id)
		for _, it := range g.P.Inv {
			if it.ID == id {
				g.throw(it, tx, ty)
				return
			}
		}
	case strings.HasPrefix(pend, "power:"):
		g.usePower(pend[6:], nil, tx, ty)
	case strings.HasPrefix(pend, "art:"):
		var id int
		fmt.Sscanf(pend[4:], "%d", &id)
		for _, ab := range g.Abilities() {
			if ab.Item != nil && ab.Item.ID == id {
				g.usePower(ab.ID, ab.Item, tx, ty)
				return
			}
		}
	}
}

func (g *Game) useAmmo() bool {
	for _, it := range g.P.Inv {
		if it.Kind == "ammo" && it.Count > 0 {
			it.Count--
			if it.Count == 0 {
				g.removeInv(it)
			}
			return true
		}
	}
	return false
}

func (g *Game) throw(it *Item, tx, ty int) {
	path := line(g.P.X, g.P.Y, tx, ty)
	rng := 6 + g.P.M(STR) + g.Skills["missile"]
	lx, ly := g.P.X, g.P.Y
	for i, pt := range path {
		if i >= rng || g.Cur.Opaque(pt[0], pt[1]) {
			break
		}
		lx, ly = pt[0], pt[1]
		if g.Cur.ActorAt(pt[0], pt[1]) != nil {
			break
		}
	}
	one := it
	if it.Count > 1 {
		it.Count--
		cp := *it
		cp.Count = 1
		one = &cp
	} else {
		g.removeInv(it)
	}
	if one.Kind == "grenade" {
		g.Msg("You throw the " + one.Name + ". {{O|BOOM!}}")
		g.explode(lx, ly, one.Effect, 2+g.P.Level/3, g.P)
	} else {
		one.X, one.Y = lx, ly
		g.Cur.Items = append(g.Cur.Items, one)
		if a := g.Cur.ActorAt(lx, ly); a != nil {
			dmg := g.R.Dice(1, 4) + maxi(0, g.P.M(STR))
			g.Msg(fmt.Sprintf("You hit %s with the %s for %d.", g.the(a), one.Name, dmg))
			g.hurt(a, dmg, g.P, one.Name)
		}
	}
	g.spend(1000)
}

// ---- items ----

func (g *Game) useItem(it *Item) {
	p := g.P
	switch it.Kind {
	case "food", "corpse":
		g.Hunger = maxi(0, g.Hunger-maxi(it.Power, 300))
		g.Msg("You eat the " + it.Name + ".")
		if it.Kind == "corpse" && g.R.Chance(0.3) {
			p.AddSt("poisoned", 6)
			g.Msg("{{G|It disagrees with you.}}")
		}
		g.consume(it)
	case "water":
		if it.Charges <= 0 {
			g.Msg("Your " + it.Name + " is empty.")
			return
		}
		it.Charges--
		g.Thirst = maxi(0, g.Thirst-450)
		g.Msg(fmt.Sprintf("You drink a dram of water. (%d left)", it.Charges))
	case "tonic":
		switch it.Effect {
		case "heal":
			h := 12 + p.Level*3
			p.HP = mini(p.MaxHP, p.HP+h)
			g.Msg(fmt.Sprintf("{{G|You feel better. (+%d HP)}}", h))
		case "salve":
			delete(p.Status, "bleeding")
			delete(p.Status, "burning")
			p.HP = mini(p.MaxHP, p.HP+6)
			g.Msg("Your wounds stop bleeding.")
		case "haste":
			p.AddSt("hasted", 40)
			g.Msg("{{C|Time slows around you.}}")
		case "strength":
			p.AddSt("mighty", 40)
			g.Msg("{{O|Your muscles swell.}}")
		case "antidote":
			delete(p.Status, "poisoned")
			g.Msg("The poison leaves you.")
		}
		g.consume(it)
	case "book":
		k := fmt.Sprintf("book:%s", it.Name)
		if !g.Read[k] {
			g.Read[k] = true
			g.gainXP(15 + g.Skills["lore"]*15)
		}
		g.showPopup(it.Name, it.Text)
		return
	case "light":
		if it.Extra == nil {
			it.Extra = map[string]int{}
		}
		it.Extra["lit"] = 1 - it.Extra["lit"]
		g.Msg(fmt.Sprintf("You %s the %s.", map[int]string{0: "douse", 1: "light"}[it.Extra["lit"]], it.Name))
	case "letter":
		g.showPopup(it.Name, "It is sealed. You shouldn't open it.")
		return
	default:
		g.Msg("You can't use that.")
		return
	}
	g.Mode = "main"
	g.spend(1000)
}

func (g *Game) consume(it *Item) {
	if it.Count > 1 {
		it.Count--
		return
	}
	g.removeInv(it)
}

// examine identifies oddities and relics (Tinkering for devices, Lore for relics, INT for both).
func (g *Game) examine(it *Item) {
	if it.Kind == "artifact" && !it.Ident {
		skill := g.Skills["tinkering"]
		if it.SimArt > 0 {
			skill = g.Skills["lore"]
		}
		if g.R.Int(1, 20)+skill*4+g.P.M(INT) >= 12+it.Tier {
			it.Ident = true
			g.Msg("{{M|You figure it out: it is " + it.Name + "!}}")
			g.gainXP(20 + it.Tier*10)
			if it.SimArt > 0 {
				g.Msg("{{W|A relic from the chronicles of this world.}}")
			}
		} else {
			g.Msg("You turn it over and over but can't make sense of it.")
			g.spend(1000)
			return
		}
	}
	g.showPopup(Plain(it.Display()), strings.Join(it.Describe(g), "\n"))
}

func (g *Game) disassemble(it *Item) {
	if g.Skills["tinkering"] == 0 {
		g.Msg("You need the Tinkering skill.")
		return
	}
	n := maxi(1, it.Value/8)
	g.removeInv(it)
	g.give(g.MakeScrap(g.R, n))
	g.Msg(fmt.Sprintf("You take apart the %s for %d %s.", it.Name, n, g.Sk.Scrap))
	g.spend(1000)
}

type Recipe struct {
	Name string
	Cost int
	Make func() *Item
}

func (g *Game) Recipes() []Recipe {
	r := g.R
	lv := g.Skills["tinkering"]
	rs := []Recipe{
		{g.Sk.Heal, 6, func() *Item { return g.MakeTonic("heal", 1) }},
		{g.Sk.Salve, 4, func() *Item { return g.MakeTonic("salve", 1) }},
		{g.Sk.GrenFire, 5, func() *Item { return g.MakeGrenade("fire", 1) }},
		{g.Sk.GrenFlash, 4, func() *Item { return g.MakeGrenade("flash", 1) }},
		{g.Sk.Ammo + " x10", 3, func() *Item { return g.MakeAmmo(10) }},
	}
	if lv >= 3 {
		rs = append(rs, Recipe{g.Sk.GrenFrost, 6, func() *Item { return g.MakeGrenade("frost", 1) }},
			Recipe{"a fine weapon", 30, func() *Item {
				return g.MakeWeapon(Pick(r, []string{"longblade", "axe", "cudgel", "shortblade"}), 2+g.P.Level/4, r)
			}})
	}
	return rs
}

func (g *Game) scrapCount() int {
	n := 0
	for _, it := range g.P.Inv {
		if it.Kind == "scrap" {
			n += it.Count
		}
	}
	return n
}

func (g *Game) craft(rc Recipe) {
	if g.scrapCount() < rc.Cost {
		g.Msg("Not enough " + g.Sk.Scrap + ".")
		return
	}
	left := rc.Cost
	for _, it := range append([]*Item{}, g.P.Inv...) {
		if it.Kind == "scrap" && left > 0 {
			take := mini(left, it.Count)
			it.Count -= take
			left -= take
			if it.Count <= 0 {
				g.removeInv(it)
			}
		}
	}
	o := rc.Make()
	g.give(o)
	g.Msg("You craft " + o.Display() + ".")
	g.spend(2000)
}

func (g *Game) pickup() {
	its := g.Cur.ItemsAt(g.P.X, g.P.Y)
	if len(its) == 0 {
		g.Msg("There is nothing here.")
		return
	}
	for _, it := range its {
		g.take(it)
	}
	g.spend(1000)
}

func (g *Game) take(it *Item) {
	for i, o := range g.Cur.Items {
		if o == it {
			g.Cur.Items = append(g.Cur.Items[:i], g.Cur.Items[i+1:]...)
			break
		}
	}
	if it.Kind == "gold" {
		g.P.Gold += it.Count
		g.Msg(fmt.Sprintf("You pick up {{W|%d %s}}.", it.Count, g.Sk.Currency))
		return
	}
	g.give(it)
	g.Msg("You pick up " + it.Display() + ".")
	if it.SimArt > 0 && !g.Taken[it.SimArt] {
		g.Taken[it.SimArt] = true
		g.deed("relic", fmt.Sprintf("The wanderer %s recovered the relic %s from %s.", g.P.Name, it.Name, g.Cur.Name), 0, it.SimArt)
		g.gainXP(80)
	}
	if it.Quest > 0 {
		g.questPickup(it)
	}
}

func (g *Game) drop(it *Item) {
	if bp := g.P.Equipped(it); bp != nil {
		bp.Item = nil
	} else {
		g.removeInv(it)
	}
	it.X, it.Y = g.P.X, g.P.Y
	g.Cur.Items = append(g.Cur.Items, it)
	g.Msg("You drop " + it.Display() + ".")
	g.spend(500)
}

func (g *Game) toggleEquip(it *Item) {
	if bp := g.P.Equipped(it); bp != nil {
		bp.Item = nil
		g.P.Inv = append(g.P.Inv, it)
		g.Msg("You unequip " + it.Display() + ".")
		g.spend(500)
		return
	}
	if it.Slot == "" && it.Kind != "light" {
		g.Msg("You can't equip that.")
		return
	}
	if !g.equip(it) {
		g.Msg("You have nowhere to wear that.")
		return
	}
	g.Msg("You equip " + it.Display() + ".")
	g.spend(500)
}

func (g *Game) rest(until bool) {
	if g.nearHostile() {
		g.Msg("You can't rest with enemies in sight.")
		return
	}
	n := 0
	for n < 400 && !g.Dead {
		g.spend(1000)
		n++
		if g.nearHostile() {
			g.Msg("{{R|You are interrupted!}}")
			break
		}
		if until && g.P.HP >= g.P.MaxHP {
			break
		}
		if !until && n >= 10 {
			break
		}
	}
	if until && g.Skills["survival"] > 0 && (g.Cur.Kind == "surface" || g.Cur.Kind == "outskirts") && g.R.Chance(0.2*float64(g.Skills["survival"])) {
		g.give(g.MakeFood(g.R, 1))
		g.Msg("You forage something to eat.")
	}
	g.Msg(fmt.Sprintf("You rest for %d turns.", n))
}
