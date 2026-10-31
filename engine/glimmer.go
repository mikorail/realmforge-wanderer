package engine

import (
	"fmt"
	"sort"
)

// Glimmer (QUD_REFERENCE 1.15): the sum of mental mutation levels. From 20 up, psychic hunters
// arrive when the player enters a zone. Prefix: mu.

func init() {
	enterHooks = append(enterHooks, muGlimmerEnter)
	killHooks = append(killHooks, muHunterKill)
	sideHooks = append(sideHooks, muGlimmerSide)
}

// muGlimmer: mental mutation levels (Ego bonus is already inside Actor.Powers); Espers add a quarter.
func muGlimmer(g *Game) int {
	if g.P == nil {
		return 0
	}
	sum := 0
	for id, lv := range g.P.Powers {
		if muMental(id) {
			sum += lv
		}
	}
	if g.mu().Morph == "esper" {
		sum += (sum + 3) / 4
	}
	return sum
}

func (g *Game) glimmer() int { return muGlimmer(g) }

var muHunterTier = []int{20, 35, 50, 65, 80, 95, 110}

// muHunterNames: bird tiers Osprey..Rukh, re-skinned per genre.
var muHunterNames = map[string][]string{
	"":           {"Osprey", "Harrier", "Owl", "Condor", "Strix", "Eagle", "Rukh"},
	"mythic":     {"Harpy-Seer", "Stymphalian", "Owl of Athena", "Griffon", "Strix of Hecate", "Eagle of Zeus", "Rukh of Olympus"},
	"steampunk":  {"Clockwork Osprey", "Brass Harrier", "Aether Owl", "Copper Condor", "Strix Engine", "Iron Eagle", "Rukh Dirigible"},
	"dieselpunk": {"Spotter", "Interceptor", "Night Owl", "Bomber", "Strix Wing", "Ace", "Rukh Squadron"},
	"atompunk":   {"Scout Drone", "Harrier Probe", "Owl Sensor", "Condor Sat", "Strix Array", "Eagle Beam", "Rukh Core"},
	"cyberpunk":  {"Sniffer", "Tracer", "Owl ICE", "Black ICE", "Strix Daemon", "Eagle Eye", "Rukh Overmind"},
	"biopunk":    {"Osprey Cluster", "Harrier Swarm", "Owl Mycelium", "Condor Bloom", "Strix Vector", "Eagle Chimera", "Rukh Colony"},
	"solarpunk":  {"Osprey Warden", "Harrier Warden", "Owl Warden", "Condor Warden", "Strix Warden", "Eagle Warden", "Rukh Warden"},
}

func muHunterTierOf(glim int) int {
	t := -1
	for i, th := range muHunterTier {
		if glim >= th {
			t = i
		}
	}
	return t
}

// muSpawnNear places a hostile creature on a free walkable tile near the player.
func muSpawnNear(g *Game, tpl string, lvl int, r *Rng) *Actor {
	z := g.Cur
	if z == nil {
		return nil
	}
	for rad := 2; rad <= 6; rad++ {
		var cells [][2]int
		for dy := -rad; dy <= rad; dy++ {
			for dx := -rad; dx <= rad; dx++ {
				x, y := g.P.X+dx, g.P.Y+dy
				if (dx != 0 || dy != 0) && z.In(x, y) && z.Walkable(x, y) && z.ActorAt(x, y) == nil {
					cells = append(cells, [2]int{x, y})
				}
			}
		}
		if len(cells) > 0 {
			c := Pick(r, cells)
			a := g.Spawn(tpl, lvl, r)
			a.X, a.Y = c[0], c[1]
			a.AI = "hostile"
			z.Actors = append(z.Actors, a)
			return a
		}
	}
	return nil
}

// muGlimmerEnter: one roll per zone entry; extra hunters (up to 5) each at 0.35.
func muGlimmerEnter(g *Game, zn *Zone) {
	glim := muGlimmer(g)
	tier := muHunterTierOf(glim)
	if tier < 0 || zn.Burg > 0 {
		return
	}
	for _, a := range zn.Actors {
		if a.HP > 0 && a.Has("hunter") {
			return
		}
	}
	r := NewRng(fmt.Sprintf("%s:glimmer:%s:%d", g.Seed, zn.Key, g.Turn))
	p := float64(glim-15) / 60
	if p > 0.9 {
		p = 0.9
	}
	if !r.Chance(p) {
		return
	}
	n := 1
	for n < 5 && r.Chance(0.35) {
		n++
	}
	names, ok := muHunterNames[g.Genre]
	if !ok {
		names = muHunterNames[""]
	}
	var mental []string
	for id := range g.P.Powers {
		if muMental(id) {
			mental = append(mental, id)
		}
	}
	sortStrings(mental)
	made := 0
	for i := 0; i < n; i++ {
		base := "harpy"
		if tier >= 3 {
			base = "roc"
		}
		a := muSpawnNear(g, base, g.P.Level, r)
		if a == nil {
			break
		}
		a.Name = names[tier]
		a.Set("hunter")
		a.Set("fly")
		a.Set("flying")
		a.Angry = true
		a.F = CCyan
		if glim >= 40 {
			a.F = COrange
		}
		a.Powers = map[string]int{}
		for k := 0; k < 2 && len(mental) > 0; k++ {
			id := mental[r.Int(0, len(mental)-1)]
			a.Powers[id] = g.P.Powers[id]
		}
		a.XP += 20 * g.P.Level
		made++
	}
	if made > 0 {
		g.Msg("{{M|You are being watched. Psychic hunters descend.}}")
	}
}

func muHunterKill(g *Game, d, src *Actor) {
	if d == nil || !d.Has("hunter") || src != g.P || d.Name == g.P.Name {
		return
	}
	if NewRng(fmt.Sprintf("%s:hunter:%d", g.Seed, d.ID)).Chance(0.1) {
		g.P.Attr[EGO]++
		g.Msg("{{Y|Mastering the hunter sharpens your will: +1 Ego.}}")
	}
}

func muGlimmerSide(g *Game) []string {
	if n := muGlimmer(g); n >= 15 {
		c := "M"
		if n >= 20 {
			c = "R"
		}
		return []string{fmt.Sprintf("{{%s|Glimmer %d}}", c, n)}
	}
	return nil
}

func sortStrings(a []string) { sort.Strings(a) }
