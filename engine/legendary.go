package engine

import (
	"fmt"
	"sort"
	"strings"
)

// Legendary creatures (WP5): named, magenta, tougher versions of ordinary creatures that wander
// the wilds, lead ruin cults and own the deepest hall of a lair. Each has a guaranteed rare+
// drop, a rolled opinion of the player (love/like/dislike/hate) and can be water-ritualed
// (ritual2.go). Everything hangs off hooks registered in init(); saved state is g.ext("legend").

func init() {
	zoneHooks = append(zoneHooks, lgZone)
	killHooks = append(killHooks, lgKill, lgOathKill)
	talkHooks = append(talkHooks, lgTalk)
}

type lgState struct {
	Opin    map[int]string `json:"op,omitempty"`   // actor id → love like dislike hate
	Lair    map[int]string `json:"lair,omitempty"` // actor id → where it lairs
	Slain   []string       `json:"sl,omitempty"`   // names of legendaries you killed
	Rit     map[string]int `json:"rit,omitempty"`  // faction → rituals performed
	Counted map[int]bool   `json:"cn,omitempty"`   // actors whose ritual is already counted
	Buys    map[int]int    `json:"by,omitempty"`   // actor id → secrets bought or asked
	Oath    int            `json:"oath,omitempty"`
}

func lgSt(g *Game) *lgState {
	s := g.ext("legend", func() any { return &lgState{} }).(*lgState)
	if s.Opin == nil {
		s.Opin = map[int]string{}
	}
	if s.Lair == nil {
		s.Lair = map[int]string{}
	}
	if s.Rit == nil {
		s.Rit = map[string]int{}
	}
	if s.Counted == nil {
		s.Counted = map[int]bool{}
	}
	if s.Buys == nil {
		s.Buys = map[int]int{}
	}
	return s
}

// Mutation id lists, duplicated locally so this file stays independent of WP1.
var lgPhysical = []string{"carapace", "spikes", "stinger", "regen", "wings", "sprint", "burrow", "multiarms", "leech"}
var lgMental = []string{"psiblast", "freeze", "flames", "lightning", "slumber", "sense", "forcepush", "glimmer", "teleport"}

var lgGiven = []string{"Skarn", "Ulveth", "Morrow", "Ashka", "Dravek", "Orsolya", "Veyra", "Karrow", "Thessin", "Brogan", "Iskra", "Halvar", "Nyx", "Garrick", "Zephra", "Kuul", "Maruk", "Selvane", "Torvic", "Yrsa"}

var lgEpithets = []string{"the Pale", "the Ashen", "the Hollow-Eyed", "Bonegnasher", "the Sleepless", "Thrice-Scarred", "the Gilded", "the Unyielding", "the Last of Her Litter", "Oathbreaker's Bane", "the Pale Marshal", "the Grey Regent"}

var lgGenreEpithets = map[string][]string{
	"steampunk":  {"the Boilered", "Brassjaw", "the Cogged"},
	"dieselpunk": {"Ironhide", "the Smokestack", "Rivet-Maw"},
	"atompunk":   {"the Glowing", "Half-Life", "the Fallout-Born"},
	"cyberpunk":  {"the Overclocked", "Zero-Day", "the Chromed"},
	"biopunk":    {"the Grafted", "the Overgrown", "the Spliced"},
	"mythic":     {"Godsbane", "the Hundred-Eyed", "the Sun-Touched"},
	"solarpunk":  {"the Sunbitten", "Rootmother", "the Verdant"},
}

// lgOpinion rolls how a legendary feels about the player and returns the word and a starting reputation.
func lgOpinion(r *Rng) (string, int) {
	switch x := r.Int(0, 99); {
	case x < 12:
		return "love", 380
	case x < 38:
		return "like", 140
	case x < 62:
		return "dislike", -100
	}
	return "hate", -450
}

func lgName(g *Game, zn *Zone, r *Rng) string {
	ep := append([]string{}, lgEpithets...)
	ep = append(ep, lgGenreEpithets[g.Genre]...)
	n := Pick(r, lgGiven) + " " + Pick(r, ep)
	if zn != nil && r.Chance(0.5) {
		n += " of " + g.W.RegionName(g.W.Idx(zn.WX, zn.WY))
	}
	return n
}

// makeLegendary raises base into a legendary: +1 to every attribute, double HP, level x1.5,
// XP = level x 250, 0-2 physical and 0-2 mental powers at level 1-4, a name, a rolled opinion
// and a guaranteed rare-or-better drop. where is "wild", "lair" or "cult".
func makeLegendary(g *Game, base *Actor, r *Rng, zn *Zone, where string) *Actor {
	if base == nil || base.Has("legend") {
		return base
	}
	old := base.Name
	for i := range base.Attr {
		base.Attr[i]++
	}
	base.MaxHP *= 2
	base.HP = base.MaxHP
	base.Level = maxi(2, base.Level*3/2)
	base.AVNat++
	base.DVNat++
	base.XP = base.Level * 250
	base.Set("legend")
	base.F = CMagenta
	base.Proper = true
	base.Title = old
	base.Name = lgName(g, zn, r)
	machine := base.Faction == "constructs" || base.Has("undead")
	if !machine {
		if base.Powers == nil {
			base.Powers = map[string]int{}
		}
		for i, n := 0, r.Int(0, 2); i < n; i++ {
			base.Powers[Pick(r, lgPhysical)] = r.Int(1, 4)
		}
		for i, n := 0, r.Int(0, 2); i < n; i++ {
			base.Powers[Pick(r, lgMental)] = r.Int(1, 4)
		}
	}
	word, rep := lgOpinion(r)
	if where == "lair" || where == "cult" {
		word, rep = "hate", -450 // those who guard a hall do not greet strangers
		if r.Chance(0.25) {
			word, rep = "dislike", -100
		}
	}
	st := lgSt(g)
	st.Opin[base.ID] = word
	base.Faction = fmt.Sprintf("legend:%d", base.ID)
	g.Rep[base.Faction] = rep
	base.Desc = fmt.Sprintf("A legendary %s, named in a hundred campfire tales.", old)
	if !base.Boss {
		base.Inv = append(base.Inv, g.bossDrop(r, base.Level))
	}
	if base.Gold < 40*base.Level {
		base.Gold += r.Int(20, 40) * base.Level
	}
	return base
}

func lgZone(g *Game, zn *Zone, r *Rng) {
	if zn.Kind == "town" || zn.Kind == "sea" || zn.Kind == "outskirts" {
		return
	}
	side := NewRng(g.Seed + ":legend:" + zn.Key)
	st := lgSt(g)
	for _, a := range zn.Actors {
		if a.Boss && a.Has("boss") && !a.Has("legend") && side.Chance(0.5) {
			makeLegendary(g, a, side, zn, "lair")
			st.Lair[a.ID] = zn.Name
			lgNote(zn, a, "Lair of "+a.Name+".")
		}
	}
	if zn.Z != 0 || zn.Kind == "dungeon" || zn.Kind == "tower" {
		return
	}
	chance, where := 1.0/12, "wild"
	if zn.Kind == "ruin" {
		chance, where = 1.0/6, "cult"
	}
	if !side.Chance(chance) {
		return
	}
	w := byte('s')
	switch {
	case zn.Underwater:
		w = 'e'
	case zn.Sky:
		w = 'k'
	case zn.Kind == "ruin":
		w = 'r'
	}
	lvl := zn.Level + 2
	a := g.Spawn(pickCreature(side, zn.Biome, w, lvl), lvl, side)
	x, y, ok := zn.FreeSpot(side, nil)
	if !ok {
		return
	}
	a.X, a.Y = x, y
	a.HomeX, a.HomeY = x, y
	makeLegendary(g, a, side, zn, where)
	zn.Actors = append(zn.Actors, a)
	if where == "cult" {
		a.Desc = "The leader of a cult that gathers among the ruins, " + a.Desc
	}
	st.Lair[a.ID] = zn.Name
	lgNote(zn, a, "Den of "+a.Name+".")
}

func lgNote(zn *Zone, a *Actor, text string) {
	if zn.Notes == nil {
		zn.Notes = map[int]string{}
	}
	if zn.In(a.X, a.Y) {
		zn.Notes[a.Y*MW+a.X] = text
	}
}

// lgKill: a slain legendary becomes a deed in the chronicle.
func lgKill(g *Game, d, src *Actor) {
	if !d.Has("legend") {
		return
	}
	st := lgSt(g)
	st.Slain = append(st.Slain, d.Name)
	if src == g.P || src != nil && src.AI == "follower" {
		g.deed("slay", fmt.Sprintf("%s the %s slew the legendary %s, %s.", g.P.Name, g.Sk.Calling(g.Calling), d.Name, d.Title), 0, 0)
		g.Msg("{{M|The tale of " + d.Name + " ends here, and yours grows.}}")
	}
}

// lgOathKill: killing someone you shared the ritual with makes you an oathbreaker (ritual2.go).
func lgOathKill(g *Game, d, src *Actor) {
	if src != g.P || !d.Kinship || d.AI == "follower" || d.Has("lg:oath") {
		return
	}
	d.Set("lg:oath")
	r := NewRng(fmt.Sprintf("%s:oath:%d", g.Seed, d.ID))
	loss := r.Int(100, 200)
	var keys []string
	for k := range g.Rep {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		g.Rep[k] -= loss
	}
	lgSt(g).Oath++
	g.deed("murder", fmt.Sprintf("The wanderer %s broke the %s and killed %s.", g.P.Name, g.Sk.Ritual, d.Name), 0, 0)
	g.Msg(fmt.Sprintf("{{R|Oathbreaker! You slew one who shared the %s with you: -%d reputation with everyone.}}", strings.ToLower(g.Sk.Ritual), loss))
}
