package engine

import (
	"fmt"
	"strings"
)

// Traits and secondary stats (prefix th). Traits are earned by behaviour counters kept in
// thState.C; each has small passive effects and a value tag NPCs react to (thTraitRep).
// Secondary stats (Luck, Composure, Fame/Infamy, Perception) are derived and stored in thState.
// Shared state and the cabinet live in thoughts.go.

// thFx is a bundle of passive effects, used by thoughts and traits alike.
type thFx struct {
	At   [6]int         // attribute deltas
	Luck int            // secondary stats
	Comp int            // composure ceiling
	Perc int            // perception
	Chk  map[string]int // check kind (checks.go checkKinds) → modifier
}

func thA(pairs ...int) [6]int {
	var a [6]int
	for i := 0; i+1 < len(pairs); i += 2 {
		a[pairs[i]] += pairs[i+1]
	}
	return a
}

func (f thFx) text() string {
	var parts []string
	for i, v := range f.At {
		if v != 0 {
			parts = append(parts, fmt.Sprintf("%s %+d", AttrShort[i], v))
		}
	}
	add := func(n string, v int) {
		if v != 0 {
			parts = append(parts, fmt.Sprintf("%s %+d", n, v))
		}
	}
	add("Luck", f.Luck)
	add("Composure", f.Comp)
	add("Perception", f.Perc)
	for _, k := range sortedKeys(f.Chk) {
		add(titleCase(k), f.Chk[k])
	}
	if len(parts) == 0 {
		return "no lasting effect"
	}
	return strings.Join(parts, ", ")
}

type thTrait struct {
	ID, Name, Desc string
	Tag            string // value tag NPCs react to
	Fx             thFx
	Earn           func(c map[string]int, g *Game) bool
}

func thCnt(k string, n int) func(map[string]int, *Game) bool {
	return func(c map[string]int, _ *Game) bool { return c[k] >= n }
}

var thTraits = []thTrait{
	{"brutal", "Brutal", "You have killed so many that it no longer registers.", "violent", thFx{At: thA(STR, 1, EGO, -1), Comp: 5, Chk: map[string]int{"intimidate": 1, "empathy": -1}}, thCnt("kills", 40)},
	{"merciful", "Merciful", "You talk people down. They remember it.", "mercy", thFx{At: thA(WIL, 1), Chk: map[string]int{"empathy": 1, "persuasion": 1}}, thCnt("spared", 2)},
	{"greedy", "Greedy", "Your purse has been fat so long you check it in your sleep.", "greed", thFx{Luck: 1, Chk: map[string]int{"streetwise": 1}, At: thA(WIL, -1)}, thCnt("hoardMin", 2880)},
	{"pious", "Pious", "Altars feel like home.", "faith", thFx{At: thA(WIL, 1), Comp: 8, Chk: map[string]int{"intimidate": -1}}, thCnt("prayers", 8)},
	{"wanderer", "Wanderer", "The road is the only address you keep.", "road", thFx{At: thA(TOU, 1), Perc: 1, Chk: map[string]int{"survival": 1}}, thCnt("steps", 150)},
	{"scholar", "Scholar", "You read the margins as closely as the text.", "lore", thFx{At: thA(INT, 1, STR, -1), Chk: map[string]int{"lore": 2}}, thCnt("books", 6)},
	{"glutton", "Glutton", "You eat when you can and plan the next meal while chewing.", "appetite", thFx{At: thA(TOU, 1, AGI, -1), Comp: 3}, thCnt("meals", 40)},
	{"teetotal", "Teetotal", "Ten days and not one tonic touched your lips.", "temperance", thFx{At: thA(WIL, 1), Perc: 1, Luck: -1}, func(c map[string]int, g *Game) bool { return g.Day() >= 10 && c["tonics"] == 0 && g.P.Level >= 2 }},
	{"nightowl", "Night Owl", "Darkness is just another room.", "night", thFx{At: thA(AGI, 1), Perc: 1, Comp: -3}, thCnt("nightMin", 600)},
	{"hydrophile", "Hydrophile", "You never pass clean water without kneeling to it.", "temperance", thFx{At: thA(TOU, 1), Comp: 2}, thCnt("drinks", 40)},
	{"iconoclast", "Iconoclast", "You have put your hand through a god's face and the sky stayed put.", "heresy", thFx{At: thA(EGO, 1, WIL, -1), Chk: map[string]int{"bluff": 1}}, func(c map[string]int, g *Game) bool {
		if c["priestKills"] >= 1 {
			return true
		}
		for k, v := range g.Rep {
			if strings.HasPrefix(k, "faith:") && v <= -100 {
				return true
			}
		}
		return false
	}},
	{"mourner", "Mourner", "You carry the names of people who walked beside you.", "grief", thFx{At: thA(WIL, 1), Comp: -4, Chk: map[string]int{"empathy": 1}}, thCnt("compLost", 2)},
	{"companionable", "Companionable", "You have rarely slept a night alone on the road.", "fellowship", thFx{At: thA(EGO, 1), Comp: 4}, thCnt("followerMin", 2880)},
	{"gregarious", "Gregarious", "You never met a stranger you could not talk to.", "fellowship", thFx{At: thA(EGO, 1), Chk: map[string]int{"persuasion": 1, "streetwise": 1}}, thCnt("talks", 30)},
	{"delver", "Delver", "You trust stone over sky.", "depths", thFx{At: thA(TOU, 1), Perc: 1, Comp: 3}, thCnt("deep", 6)},
	{"seafarer", "Seafarer", "Your legs expect the floor to move.", "road", thFx{At: thA(AGI, 1), Luck: 1}, thCnt("sea", 5)},
	{"giantkiller", "Giant-Killer", "Big things have stopped frightening you. Small things have not.", "valor", thFx{At: thA(STR, 1), Comp: 10, Luck: -1}, thCnt("bossKills", 3)},
	{"survivor", "Survivor", "You have been one breath from dying often enough to know its weight.", "valor", thFx{At: thA(TOU, 1), Luck: 1, Comp: -4}, thCnt("lowHP", 6)},
	{"murderer", "Cold-Blooded", "Someone who did not need to die, died.", "blood", thFx{At: thA(EGO, 1), Comp: -6, Chk: map[string]int{"intimidate": 1, "empathy": -2}}, thCnt("innocents", 1)},
	{"ascetic", "Ascetic", "A hollow purse hardly slows you down.", "temperance", thFx{At: thA(WIL, 1), Luck: -1, Comp: 4}, thCnt("poorMin", 2880)},
	{"stormtouched", "Storm-Touched", "You have walked bareheaded through weather that killed livestock.", "road", thFx{At: thA(TOU, 1, INT, -1), Luck: 1}, thCnt("stormMin", 300)},
	{"haunted", "Haunted", "Something followed you home and has not left.", "dread", thFx{At: thA(INT, 1), Comp: -8, Perc: 2}, thCnt("lowCompMin", 120)},
}

var thTraitByID = func() map[string]*thTrait {
	m := map[string]*thTrait{}
	for i := range thTraits {
		m[thTraits[i].ID] = &thTraits[i]
	}
	return m
}()

// thHasTrait reports whether the wanderer has earned a trait.
func thHasTrait(g *Game, id string) bool {
	for _, t := range thSt(g).Traits {
		if t == id {
			return true
		}
	}
	return false
}

// thEarn awards every trait whose condition now holds.
func thEarn(g *Game) {
	s := thSt(g)
	for i := range thTraits {
		t := &thTraits[i]
		if thHasTrait(g, t.ID) || !t.Earn(s.C, g) {
			continue
		}
		s.Traits = append(s.Traits, t.ID)
		g.Msg(fmt.Sprintf("{{G|Trait earned: %s.}} %s {{K|(%s)}}", t.Name, t.Desc, t.Fx.text()))
	}
}

// ---- NPC values ----

type thValues struct{ like, dislike []string }

var thRoleValues = map[string]thValues{
	"priest":     {[]string{"faith", "mercy", "temperance", "lore"}, []string{"heresy", "violent", "greed", "blood"}},
	"merchant":   {[]string{"greed", "road", "fellowship"}, []string{"violent", "blood"}},
	"smith":      {[]string{"valor", "road"}, []string{"blood", "heresy"}},
	"apothecary": {[]string{"lore", "mercy", "temperance"}, []string{"blood", "violent"}},
	"bookseller": {[]string{"lore", "road"}, []string{"heresy", "violent"}},
	"guard":      {[]string{"valor", "fellowship"}, []string{"blood", "heresy"}},
	"elder":      {[]string{"lore", "road", "faith", "grief"}, []string{"blood", "heresy", "violent"}},
	"notable":    {[]string{"lore", "greed", "road"}, []string{"blood", "heresy"}},
	"villager":   {[]string{"mercy", "fellowship", "temperance"}, []string{"violent", "blood", "dread"}},
}

// thTraitRep is how an NPC feels about your earned traits: -4 (cold) to +4 (warm). Use it in
// greetings and prices.
func thTraitRep(g *Game, a *Actor) int {
	if a == nil || g == nil || g.P == nil {
		return 0
	}
	s := thSt(g)
	vals, ok := thRoleValues[a.Role]
	if !ok {
		vals = thValues{[]string{"mercy", "fellowship"}, []string{"blood"}}
	}
	n := 0
	for _, id := range s.Traits {
		t := thTraitByID[id]
		if t == nil {
			continue
		}
		for _, l := range vals.like {
			if l == t.Tag {
				n++
			}
		}
		for _, d := range vals.dislike {
			if d == t.Tag {
				n--
			}
		}
		if a.Religion > 0 && a.Religion == g.HomeFaith {
			if t.ID == "pious" {
				n++
			}
			if t.ID == "iconoclast" {
				n -= 2
			}
		}
	}
	return clampi(n, -4, 4)
}

// thTraitLine is an NPC's one-line reaction to your strongest matching trait.
func thTraitLine(g *Game, a *Actor) string {
	s := thSt(g)
	vals, ok := thRoleValues[a.Role]
	if !ok {
		vals = thValues{[]string{"mercy", "fellowship"}, []string{"blood"}}
	}
	for _, id := range s.Traits {
		t := thTraitByID[id]
		for _, d := range vals.dislike {
			if t != nil && d == t.Tag {
				return fmt.Sprintf("%s's eyes settle on you and cool. \"%s. It shows.\"", a.Name, t.Name)
			}
		}
	}
	for _, id := range s.Traits {
		t := thTraitByID[id]
		for _, l := range vals.like {
			if t != nil && l == t.Tag {
				return fmt.Sprintf("%s studies you and softens. \"%s. I can tell. That counts for something.\"", a.Name, t.Name)
			}
		}
	}
	return ""
}

// ---- secondary stats ----

func thLuck(g *Game) int         { return thSt(g).Luck }
func thComposure(g *Game) int    { return thSt(g).Comp }
func thComposureMax(g *Game) int { return thSt(g).CompMax }
func thFame(g *Game) int         { return thSt(g).Fame }
func thInfamy(g *Game) int       { return thSt(g).Infamy }
func thPerception(g *Game) int   { return thSt(g).Perc }

// thCheckBonus is the extra modifier for a check of the given kind (checks.go checkKinds):
// thought and trait bonuses, perception on perception checks, a penalty when shaken.
func thCheckBonus(g *Game, kind string) int {
	s := thSt(g)
	b := s.Chk[kind]
	if kind == "perception" {
		b += (s.Perc - 5) / 2
	}
	if s.Comp < 25 {
		b--
	}
	return b
}

// thCritBonus is extra crit/lucky-hit percent from Luck (combat.go could add it to a crit roll).
func thCritBonus(g *Game) int { return clampi(thSt(g).Luck, -5, 10) }

// thToHitBonus is a small to-hit modifier from Perception and Composure (combat.go).
func thToHitBonus(g *Game) int {
	b := (thSt(g).Perc - 5) / 4
	if thSt(g).Comp < 25 {
		b--
	}
	return b
}

func thFameFromDeeds(g *Game) {
	s := thSt(g)
	for ; s.DeedsSeen < len(g.Deeds); s.DeedsSeen++ {
		d := g.Deeds[s.DeedsSeen]
		switch d.Kind {
		case "slay":
			if strings.Contains(d.Text, "legendary") {
				s.FameB += 12
			} else {
				s.FameB += 6
			}
		case "quest":
			s.FameB += 8
		case "relic":
			s.FameB += 6
		case "order":
			s.FameB += 5
		case "kinship":
			s.FameB += 3
		case "murder":
			s.InfB += 10
		}
	}
	s.Fame = s.FameB + g.Kills/20
	s.Infamy = s.InfB + s.C["innocents"]*4
}

// thFameTitle names how the realms speak of you.
func thFameTitle(g *Game) string {
	s := thSt(g)
	switch {
	case s.Fame >= 60 && s.Fame >= s.Infamy:
		return "Hero of the Songs"
	case s.Infamy >= 60 && s.Infamy > s.Fame:
		return "A Name for Frightening Children"
	case s.Fame >= 20 && s.Infamy >= 20:
		return "Famous and Feared"
	case s.Fame >= 20:
		return "Known on the Roads"
	case s.Infamy >= 20:
		return "Whispered About"
	}
	return "Nobody Yet"
}
