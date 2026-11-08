package engine

import (
	"fmt"
	"strings"
)

// Deep Taming and Beast systems (prefix td):
// 1. Breeding two tamed animals for stronger offspring with inherited morph/color (tdBreed).
// 2. Teaching tricks: fetch floor items, guard intercept damage, and hunt wild prey (tdTeachTrick, tdRunTricks, tdGuardIntercept).
// 3. Pack animals with saddlebags storing up to 12 items and +60 carry capacity (tdEquipPack, tdPackPut, tdPackTake, tdCarryBonus).
// 4. Animal reputation tracked across the wild: friendly beasts and herb gifts at >=40, aggressive fast predators at <=-40.

type tdState struct {
	BreedCool map[string]int     `json:"breedCool,omitempty"`
	PackItems map[string][]*Item `json:"packItems,omitempty"`
	AnimalRep int                `json:"animalRep,omitempty"`
	RepLog    []string           `json:"repLog,omitempty"`
	HuntTurn  map[string]int     `json:"huntTurn,omitempty"`
	HerbZones map[string]bool    `json:"herbZones,omitempty"`
}

func (g *Game) td() *tdState {
	st := g.ext("td", func() any { return &tdState{} }).(*tdState)
	if st.BreedCool == nil {
		st.BreedCool = map[string]int{}
	}
	if st.PackItems == nil {
		st.PackItems = map[string][]*Item{}
	}
	if st.HuntTurn == nil {
		st.HuntTurn = map[string]int{}
	}
	if st.HerbZones == nil {
		st.HerbZones = map[string]bool{}
	}
	return st
}

func init() {
	unLocalKeys = append(unLocalKeys, "td")
	turnHooks = append(turnHooks, func(g *Game) {
		if g.Cur == nil || g.Dead || g.P == nil {
			return
		}
		g.tdRunTricks()
		if g.Turn%30 == 0 {
			tdApplyWildlifeAttitude(g, g.Cur)
		}
	})
	enterHooks = append(enterHooks, func(g *Game, zn *Zone) {
		if zn != nil {
			tdApplyWildlifeAttitude(g, zn)
		}
	})
	killHooks = append(killHooks, tdOnKill)
	talkHooks = append(talkHooks, tdTalkHook)
	sideHooks = append(sideHooks, tdSideHook)
}

func tdIsTamedBeast(a *Actor) bool {
	if a == nil || a.HP <= 0 {
		return false
	}
	return a.AI == "follower" || a.Has("tm") || a.Has("td_bred")
}

// ---- 1. Breeding Two Tamed Animals -----------------------------------------

func (g *Game) tdBreed(a1, a2 *Actor) (*Actor, string) {
	if g == nil || g.Cur == nil || a1 == nil || a2 == nil || a1 == a2 {
		return nil, "You need two distinct tamed beasts to breed."
	}
	if !tdIsTamedBeast(a1) || !tdIsTamedBeast(a2) {
		return nil, "Both animals must be tamed companions."
	}
	st := g.td()
	if exp, ok := st.BreedCool[a1.Name]; ok && g.Turn < exp {
		return nil, a1.Name + " must rest before breeding again."
	}
	if exp, ok := st.BreedCool[a2.Name]; ok && g.Turn < exp {
		return nil, a2.Name + " must rest before breeding again."
	}

	st.BreedCool[a1.Name] = g.Turn + DayLen
	st.BreedCool[a2.Name] = g.Turn + DayLen

	tpl := nonEmpty(a1.Tpl, nonEmpty(a2.Tpl, "wolf"))
	lvl := maxi(a1.Level, a2.Level) + 1
	sr := NewRng(fmt.Sprintf("%s:td:breed:%s:%s:%d", g.W.Seed, a1.Name, a2.Name, g.Turn))
	child := g.Spawn(tpl, lvl, sr)
	child.Level = lvl

	baseHP := maxi(a1.MaxHP, a2.MaxHP)
	child.MaxHP = baseHP + maxi(2, baseHP*20/100)
	child.HP = child.MaxHP
	child.AVNat = maxi(a1.AVNat, a2.AVNat) + 1
	child.DVNat = maxi(a1.DVNat, a2.DVNat) + 1

	for i := range child.Attr {
		child.Attr[i] = maxi(a1.Attr[i], a2.Attr[i]) + 1
	}

	child.Name = fmt.Sprintf("%s-kin", a1.Name)
	child.Proper = true
	child.Title = "bred beast"
	child.AI = "follower"
	child.Faction = "player"
	child.Order = OrdFollow
	child.Comp = &Companion{
		Faction: "beasts",
		AI:      "wander",
		F:       a1.F,
		HomeKey: g.Cur.Key,
		Loyalty: 95,
		Since:   g.Turn,
	}
	if a1.F != 0 {
		child.F = a1.F
	} else if a2.F != 0 {
		child.F = a2.F
	} else {
		child.F = CGreen
	}

	child.Set("tm")
	child.Set("td_bred")

	// Inherit color/morph flags from parents
	inheritedMorph := false
	for _, p := range []*Actor{a1, a2} {
		for k, v := range p.Flags {
			if v && (strings.HasPrefix(k, "tr_hex:") || strings.HasPrefix(k, "morph:") || strings.HasPrefix(k, "td_morph:")) {
				child.Set(k)
				inheritedMorph = true
			}
		}
	}
	if !inheritedMorph {
		child.Set("td_morph:prime")
	}

	x, y := clampi(g.P.X+1, 1, MW-2), clampi(g.P.Y, 1, MH-2)
	if !g.Cur.Walkable(x, y) || g.Cur.ActorAt(x, y) != nil {
		if fx, fy, ok := g.Cur.FreeSpot(sr, func(px, py int) bool {
			return abs(px-g.P.X)+abs(py-g.P.Y) <= 4
		}); ok {
			x, y = fx, fy
		}
	}
	child.X, child.Y = x, y
	g.Cur.Actors = append(g.Cur.Actors, child)
	g.tm().Fed[child.ID] = g.Turn
	g.tdAddAnimalRep(5, "bred tamed beasts")

	msg := fmt.Sprintf("%s and %s produce a hardy offspring: %s (Lv %d, %d HP)!", a1.Name, a2.Name, child.Name, child.Level, child.MaxHP)
	g.Msg("{{G|" + msg + "}}")
	return child, msg
}

// ---- 2. Teaching Tricks ----------------------------------------------------

func (g *Game) tdTeachTrick(pet *Actor, trick string) bool {
	if g == nil || pet == nil {
		return false
	}
	trick = strings.ToLower(strings.TrimSpace(trick))
	switch trick {
	case "fetch", "guard", "hunt":
	default:
		return false
	}
	pet.Set("tm")
	if pet.AI != "follower" {
		pet.AI = "follower"
	}
	pet.Set("td_trick:" + trick)
	g.Msg(fmt.Sprintf("{{G|%s learns the '%s' trick!}}", pet.Name, trick))
	return true
}

func (g *Game) tdEnemyAdjacent(a *Actor) bool {
	if g == nil || g.Cur == nil || a == nil {
		return false
	}
	for _, o := range g.Cur.Actors {
		if o.HP > 0 && o != a && o != g.P && g.Hostile(o) {
			if maxi(abs(o.X-a.X), abs(o.Y-a.Y)) <= 1 {
				return true
			}
		}
	}
	return false
}

func (g *Game) tdRunTricks() {
	if g == nil || g.Cur == nil || g.P == nil {
		return
	}
	st := g.td()
	for _, pet := range g.Cur.Actors {
		if pet.HP <= 0 || !tdIsTamedBeast(pet) {
			continue
		}
		// Fetch trick: pick up floor item within 4 tiles if no enemies are adjacent.
		if pet.Has("td_trick:fetch") && !g.tdEnemyAdjacent(pet) && !g.tdEnemyAdjacent(g.P) {
			for idx, it := range g.Cur.Items {
				if it.Kind == "nest" {
					continue
				}
				if maxi(abs(it.X-pet.X), abs(it.Y-pet.Y)) <= 4 {
					g.Cur.Items = append(g.Cur.Items[:idx], g.Cur.Items[idx+1:]...)
					g.P.Inv = append(g.P.Inv, it)
					g.Msg(fmt.Sprintf("{{G|%s fetches %s and drops it into your pack!}}", pet.Name, it.Name))
					break
				}
			}
		}
		// Hunt trick: periodically brings back fresh meat or beast pelt.
		if pet.Has("td_trick:hunt") {
			last, seen := st.HuntTurn[pet.Name]
			if !seen || g.Turn-last >= 120 {
				st.HuntTurn[pet.Name] = g.Turn
				sr := NewRng(fmt.Sprintf("%s:td:hunt:%s:%d", g.W.Seed, pet.Name, g.Turn))
				var prize *Item
				if sr.Chance(0.6) {
					prize = &Item{
						ID:     g.newID(),
						Name:   "fresh meat",
						Glyph:  '%',
						F:      CRed,
						Kind:   "food",
						Effect: "meal",
						Power:  25,
						Count:  1,
						Value:  8,
						Weight: 2,
						Ident:  true,
					}
				} else {
					prize = &Item{
						ID:     g.newID(),
						Name:   "beast pelt",
						Glyph:  ',',
						F:      CBrown,
						Kind:   "misc",
						Count:  1,
						Value:  16,
						Weight: 2,
						Ident:  true,
					}
				}
				g.P.Inv = append(g.P.Inv, prize)
				g.Msg(fmt.Sprintf("{{G|%s returns from a quick hunt carrying %s!}}", pet.Name, prize.Name))
			}
		}
	}
}

func (g *Game) tdGuardIntercept(dmg int) int {
	if g == nil || g.Cur == nil || g.P == nil || dmg <= 1 {
		return dmg
	}
	if g.P.HP > g.P.MaxHP/2 {
		return dmg
	}
	for _, pet := range g.Cur.Actors {
		if pet.HP <= 1 || !tdIsTamedBeast(pet) || !pet.Has("td_trick:guard") {
			continue
		}
		if maxi(abs(pet.X-g.P.X), abs(pet.Y-g.P.Y)) <= 2 {
			intercept := dmg / 2
			rem := dmg - intercept
			pet.HP = maxi(1, pet.HP-intercept)
			g.Msg(fmt.Sprintf("{{C|%s leaps in front of you, intercepting %d damage!}}", pet.Name, intercept))
			return rem
		}
	}
	return dmg
}

// ---- 3. Pack Animals -------------------------------------------------------

func (g *Game) tdEquipPack(pet *Actor) bool {
	if g == nil || pet == nil || !tdIsTamedBeast(pet) {
		return false
	}
	pet.Set("td_pack")
	st := g.td()
	if _, ok := st.PackItems[pet.Name]; !ok {
		st.PackItems[pet.Name] = []*Item{}
	}
	g.Msg(fmt.Sprintf("{{G|You outfit %s with sturdy pack saddlebags (+60 carry capacity).}}", pet.Name))
	return true
}

func (g *Game) tdPackPut(pet *Actor, it *Item) bool {
	if g == nil || pet == nil || it == nil || !pet.Has("td_pack") {
		return false
	}
	st := g.td()
	items := st.PackItems[pet.Name]
	if len(items) >= 12 {
		g.Msg(pet.Name + "'s saddlebags are full (12/12 items).")
		return false
	}
	if g.P != nil {
		for idx, invIt := range g.P.Inv {
			if invIt == it || invIt.ID == it.ID {
				g.P.Inv = append(g.P.Inv[:idx], g.P.Inv[idx+1:]...)
				break
			}
		}
	}
	st.PackItems[pet.Name] = append(items, it)
	return true
}

func (g *Game) tdPackTake(pet *Actor, idx int) *Item {
	if g == nil || pet == nil || !pet.Has("td_pack") {
		return nil
	}
	st := g.td()
	items := st.PackItems[pet.Name]
	if idx < 0 || idx >= len(items) {
		return nil
	}
	it := items[idx]
	st.PackItems[pet.Name] = append(items[:idx], items[idx+1:]...)
	if g.P != nil {
		g.P.Inv = append(g.P.Inv, it)
	}
	return it
}

func (g *Game) tdCarryBonus() int {
	if g == nil || g.Cur == nil {
		return 0
	}
	bonus := 0
	for _, a := range g.Cur.Actors {
		if a.HP > 0 && a.Has("td_pack") && tdIsTamedBeast(a) {
			bonus += 60
		}
	}
	return bonus
}

// ---- 4. Animal Reputation --------------------------------------------------

func (g *Game) tdAddAnimalRep(delta int, reason string) {
	if g == nil || delta == 0 {
		return
	}
	st := g.td()
	st.AnimalRep = clampi(st.AnimalRep+delta, -100, 100)
	if reason != "" {
		st.RepLog = append(st.RepLog, fmt.Sprintf("%+d (%s)", delta, reason))
		if len(st.RepLog) > 25 {
			st.RepLog = st.RepLog[len(st.RepLog)-25:]
		}
	}
	if g.Cur != nil {
		tdApplyWildlifeAttitude(g, g.Cur)
	}
}

func (g *Game) tdAnimalRep() int {
	if g == nil {
		return 0
	}
	return g.td().AnimalRep
}

func tdOnKill(g *Game, d, src *Actor) {
	if g == nil || d == nil || src != g.P {
		return
	}
	low := strings.ToLower(d.Name)
	if d.Has("ntyg") || d.Has("ntpar") || strings.Contains(low, "young") || strings.Contains(low, "cub") || strings.Contains(low, "hatchling") {
		g.tdAddAnimalRep(-15, "slaying "+d.Name)
	}
}

func tdApplyWildlifeAttitude(g *Game, zn *Zone) {
	if g == nil || zn == nil {
		return
	}
	rep := g.tdAnimalRep()
	st := g.td()
	switch {
	case rep >= 40:
		if g.Rep != nil && g.Rep["beasts"] < 0 {
			g.Rep["beasts"] = 0
		}
		foundBeast := false
		for _, a := range zn.Actors {
			if a.HP <= 0 || a.AI == "follower" {
				continue
			}
			if a.Faction == "beasts" {
				a.AI = "wander"
				a.Angry = false
				delete(a.Flags, "Hostile")
				a.Set("td_friendly")
				foundBeast = true
			}
		}
		if foundBeast && g.P != nil && !st.HerbZones[zn.Key] {
			st.HerbZones[zn.Key] = true
			herb := &Item{
				ID:     g.newID(),
				Name:   "foraged wild herb",
				Glyph:  ',',
				F:      CGreen,
				Kind:   "ingredient",
				Effect: "heal",
				Power:  15,
				Count:  1,
				Value:  12,
				Weight: 1,
				X:      g.P.X,
				Y:      g.P.Y,
				Ident:  true,
			}
			zn.Items = append(zn.Items, herb)
			g.Msg("{{G|A friendly woodland creature drops a foraged wild herb at your feet.}}")
		}
	case rep <= -40:
		if g.Rep != nil && g.Rep["beasts"] > -600 {
			g.Rep["beasts"] = -600
		}
		for _, a := range zn.Actors {
			if a.HP <= 0 || a.AI == "follower" {
				continue
			}
			if a.Faction == "beasts" {
				a.AI = "hostile"
				a.Angry = true
				a.Set("Hostile")
				if !a.Has("td_enraged_speed") {
					a.Speed += 1
					a.Set("td_enraged_speed")
				}
			}
		}
	}
}

func tdTalkHook(g *Game, a *Actor) []Option {
	if a == nil || !tdIsTamedBeast(a) {
		return nil
	}
	var opts []Option
	if !a.Has("td_pack") {
		opts = append(opts, Option{
			Text: "{{G|Fit pack saddlebags (+60 carry capacity).}}",
			Do: func() {
				g.tdEquipPack(a)
				g.Mode, g.Talk = "main", nil
			},
		})
	}
	for _, tr := range []string{"fetch", "guard", "hunt"} {
		tr := tr
		if !a.Has("td_trick:" + tr) {
			opts = append(opts, Option{
				Text: fmt.Sprintf("{{C|Teach trick: %s.}}", tr),
				Do: func() {
					g.tdTeachTrick(a, tr)
					g.Mode, g.Talk = "main", nil
				},
			})
		}
	}
	return opts
}

func tdSideHook(g *Game) []string {
	rep := g.tdAnimalRep()
	if rep == 0 {
		return nil
	}
	col := "y"
	if rep >= 40 {
		col = "G"
	} else if rep <= -40 {
		col = "R"
	}
	return []string{fmt.Sprintf("{{%s|Wild Rep: %+d}}", col, rep)}
}
