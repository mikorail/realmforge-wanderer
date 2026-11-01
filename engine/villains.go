package engine

import (
	"fmt"
	"sort"
	"sync"
)

// Villains (prefix vl). Every living person with Role "villain" — topped up to 2–4 per planet
// from the sim's criminals, traitors and cruel schemers (bhCriminal), then the merely cruel —
// keeps a hideout: an outlaw-style dungeon in a free sub-zone of their town's parasang (vlSite,
// hooked into findSite). On its deepest level the hideout's boss becomes the villain, with a
// sworn retinue and a guaranteed rare+ drop (vlEnsure). They leave their town (no duplicate
// notable), are gossiped about ("Ask about ..."), can be posted on bounty boards (vlBounty in
// bhGenPerson; bhPlaceTarget reuses the villain) and their death is a deed. State: g.ext("vl"),
// planet-local (planets.go unLocalKeys).

type vlEntry struct {
	P      *PersonData
	Burg   *BurgData
	ZX, ZY int
}

type vlState struct {
	Dead map[int]int `json:"dead,omitempty"` // person id → year slain
	Act  map[int]int `json:"act,omitempty"`  // actor id → person id
}

var (
	vlCacheMu sync.Mutex
	vlCache   = map[*WorldData][]*vlEntry{}
)

func init() {
	zoneHooks = append(zoneHooks, vlZone)
	killHooks = append(killHooks, vlKill)
	talkHooks = append(talkHooks, vlTalk)
}

func (g *Game) vl() *vlState {
	st := g.ext("vl", func() any { return &vlState{} }).(*vlState)
	if st.Dead == nil {
		st.Dead = map[int]int{}
	}
	if st.Act == nil {
		st.Act = map[int]int{}
	}
	return st
}

func vlDead(g *Game, id int) bool {
	_, ok := g.vl().Dead[id]
	return ok
}

func vlCruel(p *PersonData) bool {
	for _, t := range p.Traits {
		if t == "Cruel" {
			return true
		}
	}
	return false
}

// vlList: this planet's villains with their hideout sub-zones (deterministic, cached per world).
func (g *Game) vlList() []*vlEntry {
	w := g.W
	vlCacheMu.Lock()
	if l, ok := vlCache[w]; ok {
		vlCacheMu.Unlock()
		return l
	}
	vlCacheMu.Unlock()
	usable := func(p *PersonData) *BurgData {
		if !p.Alive || p.Burg <= 0 || p.Role == "ruler" {
			return nil
		}
		bd := w.burgByID[p.Burg]
		if bd == nil || !bd.Alive || bd.Layer != "surface" {
			return nil
		}
		return bd
	}
	rank := func(p *PersonData) int {
		switch {
		case p.Role == "villain":
			return 0
		case bhCriminal(p):
			return 1
		case vlCruel(p):
			return 2
		}
		return 9
	}
	var cs []*PersonData
	for i := range w.People {
		p := &w.People[i]
		if usable(p) != nil && rank(p) < 9 {
			cs = append(cs, p)
		}
	}
	sort.SliceStable(cs, func(i, j int) bool {
		if ri, rj := rank(cs[i]), rank(cs[j]); ri != rj {
			return ri < rj
		}
		if cs[i].Renown != cs[j].Renown {
			return cs[i].Renown > cs[j].Renown
		}
		return cs[i].ID < cs[j].ID
	})
	want := 2 + int(HashStr(w.Seed+":vl:n")%3)
	used := map[[3]int]bool{}
	var out []*vlEntry
	for _, p := range cs {
		if len(out) >= want && p.Role != "villain" {
			break
		}
		bd := usable(p)
		cp := g.cityPlanFor(bd)
		oz := g.outlawZone(bd)
		hasOutlaw := w.outlawByBrg[bd.ID] != nil
		for _, s := range [][2]int{{2, 1}, {0, 1}, {1, 2}, {1, 0}, {2, 2}, {0, 0}, {2, 0}, {0, 2}} {
			if _, ok := cp.districts[s]; ok || hasOutlaw && s == oz || used[[3]int{bd.ID, s[0], s[1]}] {
				continue
			}
			used[[3]int{bd.ID, s[0], s[1]}] = true
			out = append(out, &vlEntry{P: p, Burg: bd, ZX: s[0], ZY: s[1]})
			break
		}
	}
	vlCacheMu.Lock()
	vlCache[w] = out
	vlCacheMu.Unlock()
	return out
}

func (g *Game) vlOf(id int) *vlEntry {
	for _, v := range g.vlList() {
		if v.P.ID == id {
			return v
		}
	}
	return nil
}

func (g *Game) vlAt(bd *BurgData, zx, zy int) *vlEntry {
	for _, v := range g.vlList() {
		if v.Burg.ID == bd.ID && v.ZX == zx && v.ZY == zy {
			return v
		}
	}
	return nil
}

func vlHideoutName(v *vlEntry) string { return "Hideout of " + v.P.Name }

// vlSite is the findSite hook: a villain's hideout under a free sub-zone of their town's parasang.
func (g *Game) vlSite(bd *BurgData, zx, zy int) *Site {
	if v := g.vlAt(bd, zx, zy); v != nil {
		return &Site{Kind: "dungeon", Theme: "hideout", Name: vlHideoutName(v), Depth: 2, Gold: 100 + v.P.Renown*3}
	}
	return nil
}

func vlTitle(v *vlEntry) string {
	if v.P.Title != "" {
		return v.P.Title
	}
	if v.P.Role == "villain" || v.P.Role == "" {
		return "villain"
	}
	return "renegade " + v.P.Role
}

// vlEnsure puts the villain on the deepest level of their hideout (idempotent); with id != 0 only
// that person. Returns the villain actor or nil.
func (g *Game) vlEnsure(zn *Zone, id int) *Actor {
	if zn.Kind != "dungeon" || zn.Z < 1 {
		return nil
	}
	w := g.W
	p := w.Idx(zn.WX, zn.WY)
	bd := w.burgByID[w.Burg[p]]
	if bd == nil {
		return nil
	}
	v := g.vlAt(bd, zn.ZX, zn.ZY)
	if v == nil || id != 0 && v.P.ID != id || vlDead(g, v.P.ID) {
		return nil
	}
	s := g.siteAt(zn.WX, zn.WY, zn.ZX, zn.ZY)
	if s == nil || zn.Z != s.Depth {
		return nil
	}
	st := g.vl()
	for _, a := range zn.Actors {
		if st.Act[a.ID] == v.P.ID {
			if a.HP > 0 {
				return a
			}
			return nil
		}
	}
	r := NewRng(w.Seed + ":vl:" + zn.Key)
	var boss *Actor
	for _, a := range zn.Actors {
		if a.Boss && a.Has("boss") && a.HP > 0 {
			boss = a
			break
		}
	}
	if boss == nil {
		boss = g.Spawn("chief", zn.Level+3, r)
		boss.Boss = true
		boss.Set("boss")
		x, y, ok := zn.FreeSpot(r, nil)
		if !ok {
			return nil
		}
		boss.X, boss.Y = x, y
		zn.Actors = append(zn.Actors, boss)
	}
	boss.Name, boss.Proper, boss.Title = v.P.Name, true, "villain"
	boss.Faction, boss.AI = "outlaws", "hostile"
	boss.Desc = fmt.Sprintf("%s, %s of %s, wanted for %s. This is their hideout.", v.P.Name, vlTitle(v), v.Burg.Name, bhCrime(v.P))
	boss.Level = maxi(boss.Level, zn.Level+3+v.P.Martial/4)
	boss.MaxHP += 10 + v.P.Martial*4
	boss.HP = boss.MaxHP
	boss.XP += 300 + v.P.Renown*5
	boss.Gold += 80 + v.P.Renown*2
	boss.F = CMagenta
	boss.Inv = append(boss.Inv, g.bossDrop(r, zn.Level+4)) // guaranteed rare or better
	st.Act[boss.ID] = v.P.ID
	for i := 0; i < 2+r.Int(0, 2); i++ {
		a := g.Spawn(Pick(r, []string{"bandit", "bandit", "archer"}), zn.Level+1, r)
		a.X, a.Y = boss.X+r.Int(-3, 3), boss.Y+r.Int(-2, 2)
		g.nudge(zn, a)
		a.Desc = "Sworn to " + v.P.Name + "."
		zn.Actors = append(zn.Actors, a)
	}
	return boss
}

// vlZone: villains wait in their hideouts, not in their towns.
func vlZone(g *Game, zn *Zone, _ *Rng) {
	if zn.Kind == "dungeon" {
		g.vlEnsure(zn, 0)
		return
	}
	if zn.Z != 0 || zn.Burg == 0 {
		return
	}
	out := zn.Actors[:0]
	for _, a := range zn.Actors {
		if a.SimPer > 0 && a.AI != "follower" {
			if v := g.vlOf(a.SimPer); v != nil && !vlDead(g, v.P.ID) {
				continue
			}
		}
		out = append(out, a)
	}
	zn.Actors = out
}

func vlKill(g *Game, d, src *Actor) {
	st := g.vl()
	pid := st.Act[d.ID]
	if pid == 0 {
		return
	}
	v := g.vlOf(pid)
	st.Dead[pid] = g.W.Year
	if v == nil || src != g.P && (src == nil || src.AI != "follower") {
		return
	}
	g.deed("villain", fmt.Sprintf("%s the %s brought down %s, %s of %s, in their own hideout.", g.P.Name, g.Sk.Calling(g.Calling), v.P.Name, vlTitle(v), v.Burg.Name), 0, 0)
	g.Msg("{{W|" + v.P.Name + " is dead. " + v.Burg.Name + " will sleep easier.}}")
	if v.Burg.State > 0 {
		g.AddRep(fmt.Sprintf("state:%d", v.Burg.State), 80, " for ending "+v.P.Name)
	}
}

// vlTalk: townsfolk know of villains nearby and where they hide.
func vlTalk(g *Game, a *Actor) []Option {
	if g.Cur == nil || g.Hostile(a) || a.AI == "follower" || a.AI != "town" && a.Role == "" {
		return nil
	}
	w := g.W
	var best *vlEntry
	bd2 := 1 << 30
	for _, v := range g.vlList() {
		if vlDead(g, v.P.ID) {
			continue
		}
		d := abs(v.Burg.P%w.W-g.Cur.WX) + abs(v.Burg.P/w.W-g.Cur.WY)
		if d <= 30 && d < bd2 {
			best, bd2 = v, d
		}
	}
	if best == nil {
		return nil
	}
	v := best
	return []Option{{"Ask about " + v.P.Name + ".", func() {
		dir := [3][3]string{{"north-west", "north", "north-east"}, {"west", "the edge", "east"}, {"south-west", "south", "south-east"}}[v.ZY][v.ZX]
		txt := fmt.Sprintf("\"%s? Once %s of %s. Now %s. They keep a hideout %s of %s, with cutthroats sworn to them. Nobody who went in after them came back out.\"",
			v.P.Name, nonEmpty(v.P.Title, v.P.Role), v.Burg.Name, bhCrime(v.P), dir, v.Burg.Name)
		g.Known[v.Burg.P] = true
		g.addSecret(fmt.Sprintf("%s hides in %s, %s of %s.", v.P.Name, vlHideoutName(v), dir, v.Burg.Name), "villain", v.Burg.P, w.Year)
		g.talkMenu(txt)
	}}}
}

// vlBounty: a wanted poster for a villain points at their hideout.
func (g *Game) vlBounty(p *PersonData) *bhBounty {
	v := g.vlOf(p.ID)
	if v == nil || vlDead(g, p.ID) {
		return nil
	}
	bx, by := v.Burg.P%g.W.W, v.Burg.P/g.W.W
	b := &bhBounty{Kind: "person", Name: p.Name, Person: p.ID, Burg: v.Burg.ID, State: v.Burg.State, WX: bx, WY: by, ZX: v.ZX, ZY: v.ZY, Z: 2,
		Lvl: maxi(4, g.zoneLevel(bx, by, 0)+4), Reward: 300 + p.Renown*4 + p.Martial*15, Where: vlHideoutName(v), Live: true}
	b.Poster = fmt.Sprintf("WANTED: %s, %s of %s, for %s. Holed up in the deepest hall of %s outside %s with sworn blades. Dead or alive; alive pays half again.",
		p.Name, vlTitle(v), v.Burg.Name, bhCrime(p), vlHideoutName(v), v.Burg.Name)
	return b
}
