package engine

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

type PowerDef struct {
	ID     string `json:"id"`
	Cost   int    `json:"cost"`
	Active bool   `json:"active"`
	Cool   int    `json:"cool"`
	Target string `json:"target"` // self dir ranged
	Desc   string `json:"desc"`
	Defect bool   `json:"defect"`
	Name   string `json:"name"`
}

var PowerDefs = []PowerDef{
	{ID: "flames", Cost: 4, Active: true, Cool: 10, Target: "ranged", Desc: "Hurl a gout of fire along a line. Sets grass, oil and enemies alight."},
	{ID: "freeze", Cost: 4, Active: true, Cool: 10, Target: "ranged", Desc: "A freezing ray: damages and freezes a target solid."},
	{ID: "lightning", Cost: 5, Active: true, Cool: 14, Target: "ranged", Desc: "Lightning that arcs from foe to foe."},
	{ID: "psiblast", Cost: 4, Active: true, Cool: 12, Target: "ranged", Desc: "Strike a mind directly; ignores armor, opposed by Willpower."},
	{ID: "teleport", Cost: 3, Active: true, Cool: 40, Target: "ranged", Desc: "Vanish and reappear at a spot you can see."},
	{ID: "phase", Cost: 4, Active: true, Cool: 60, Target: "self", Desc: "Become insubstantial: walk through walls and weapons for a while."},
	{ID: "sprint", Cost: 2, Active: true, Cool: 50, Target: "self", Desc: "Double speed for a short burst."},
	{ID: "heal", Cost: 3, Active: true, Cool: 60, Target: "self", Desc: "Mend your wounds and stop bleeding."},
	{ID: "beguile", Cost: 5, Active: true, Cool: 80, Target: "ranged", Desc: "Ego vs Willpower: a creature becomes your loyal companion."},
	{ID: "forcewall", Cost: 3, Active: true, Cool: 60, Target: "dir", Desc: "Raise a wall of force in a direction."},
	{ID: "burrow", Cost: 3, Active: true, Cool: 0, Target: "dir", Desc: "Dig through rock and walls."},
	{ID: "leech", Cost: 3, Active: true, Cool: 20, Target: "ranged", Desc: "Drain the life of an adjacent-or-near foe into yourself."},
	{ID: "regen", Cost: 3, Desc: "You regenerate quickly and can regrow lost limbs."},
	{ID: "carapace", Cost: 3, Desc: "+AV from a hardened hide."},
	{ID: "spikes", Cost: 2, Desc: "Spines: melee attackers are hurt."},
	{ID: "stinger", Cost: 3, Desc: "A poison natural weapon that strikes alongside your others."},
	{ID: "sense", Cost: 2, Desc: "Sense creatures through walls nearby."},
	{ID: "darkvision", Cost: 1, Desc: "See in the dark."},
	{ID: "gills", Cost: 1, Desc: "Breathe underwater and swim freely; cross seas on the world map."},
	{ID: "wings", Cost: 4, Desc: "Fly: cross water and chasms, never fall from the sky, travel faster."},
	// defects grant points
	{ID: "brittle", Cost: -3, Defect: true, Desc: "Brittle bones: -2 AV."},
	{ID: "hemophilia", Cost: -3, Defect: true, Desc: "Your bleeding never stops on its own."},
	{ID: "nearsight", Cost: -2, Defect: true, Desc: "You can't see far: sight radius 8."},
	{ID: "glutton", Cost: -2, Defect: true, Desc: "You hunger twice as fast."},
	{ID: "albino", Cost: -2, Defect: true, Desc: "Sunlight wears you down: -1 regen by day on the surface."},
}

var powerByID = func() map[string]*PowerDef {
	m := map[string]*PowerDef{}
	for i := range PowerDefs {
		m[PowerDefs[i].ID] = &PowerDefs[i]
	}
	return m
}()

type CallingDef struct {
	ID     string         `json:"id"`
	Name   string         `json:"name"`
	Desc   string         `json:"desc"`
	Attr   map[int]int    `json:"attr"`
	Skills map[string]int `json:"skills"`
}

var callingDefs = []CallingDef{
	{ID: "warrior", Desc: "Trained in arms. Starts armored with a good blade.", Attr: map[int]int{STR: 2, TOU: 1}, Skills: map[string]int{"longblade": 1, "axe": 1, "endurance": 1}},
	{ID: "scout", Desc: "Ranges far, shoots straight, lives off the land.", Attr: map[int]int{AGI: 2, WIL: 1}, Skills: map[string]int{"missile": 2, "survival": 1}},
	{ID: "scholar", Desc: "Reads ruins and relics; knows the chronicles of the realms.", Attr: map[int]int{INT: 2, WIL: 1}, Skills: map[string]int{"lore": 2, "tinkering": 1}},
	{ID: "tinker", Desc: "Takes things apart, builds bombs and tonics from scrap.", Attr: map[int]int{INT: 2, TOU: 1}, Skills: map[string]int{"tinkering": 2, "cudgel": 1}},
	{ID: "mystic", Desc: "Faith or strange power: one extra power point, good with words.", Attr: map[int]int{WIL: 2, EGO: 1}, Skills: map[string]int{"lore": 1, "persuasion": 1}},
	{ID: "merchant", Desc: "Starts rich; better prices, trusted by towns.", Attr: map[int]int{EGO: 2, INT: 1}, Skills: map[string]int{"persuasion": 2, "shortblade": 1}},
	{ID: "rogue", Desc: "Quick hands, two knives, hard to pin down.", Attr: map[int]int{AGI: 2, EGO: 1}, Skills: map[string]int{"shortblade": 2, "acrobatics": 1}},
}

var SkillIDs = []string{"longblade", "shortblade", "axe", "cudgel", "missile", "tinkering", "persuasion", "survival", "acrobatics", "endurance", "lore"}

func skillName(id string) string {
	switch id {
	case "longblade":
		return "Long Blade"
	case "shortblade":
		return "Short Blade"
	case "natural":
		return "natural weapon"
	}
	return titleCase(id)
}

var skillDesc = map[string]string{
	"longblade":  "+hit with long blades; rank 2: critical hits make foes bleed.",
	"shortblade": "+hit with short blades; rank 2: off-hand strikes more often.",
	"axe":        "+hit with axes; rank 2: chance to sever a limb.",
	"cudgel":     "+hit with cudgels; rank 2: chance to stun.",
	"missile":    "+hit and range with ranged weapons and throwing.",
	"tinkering":  "Identify oddities; disassemble gear into scrap; rank 2: craft from scrap.",
	"persuasion": "Better prices, warmer rituals; rank 2: recruit companions.",
	"survival":   "Faster, safer overland travel; forage while resting in the wild.",
	"acrobatics": "+1 DV per rank.",
	"endurance":  "+3 max HP per rank per level; hunger and thirst slower.",
	"lore":       "Identify relics; learn more from inscriptions, books and secrets.",
}

// CharOptions is everything the chargen screen needs, as JSON.
func CharOptions(w *WorldData) string {
	sk := SkinFor(w.Genre)
	type sp struct {
		ID, Name, Desc string
		Common         int
		Attr           map[int]int
		Traits         []string
		Plan           string
	}
	var sps []sp
	for _, s := range w.Species {
		sps = append(sps, sp{s.ID, s.Name, s.Desc, s.Common, speciesAttr(&s), s.Traits, BodyPlanForSpecies(s.ID)})
	}
	sort.SliceStable(sps, func(i, j int) bool { return sps[i].Common > sps[j].Common })
	var cs []CallingDef
	for _, c := range callingDefs {
		c.Name = sk.Calling(c.ID)
		cs = append(cs, c)
	}
	var ps []PowerDef
	for _, p := range PowerDefs {
		p.Name = sk.PowerName(p.ID)
		if p.Defect {
			p.Name = titleCase(p.ID)
		}
		ps = append(ps, p)
	}
	type st struct {
		ID      int
		Name    string
		Realm   string
		Culture string
		Species string
		Pop     float64
		Layer   string
		Capital bool
	}
	var starts []st
	for _, b := range w.Burgs {
		if !b.Alive || b.Bandit || b.Layer != "surface" {
			continue
		}
		cu := w.cultByID[b.Culture]
		cn, spn := "", ""
		if cu != nil {
			cn = cu.Name
			spn = cu.Species
		}
		starts = append(starts, st{b.ID, b.Name, w.StateName(b.State), cn, spn, b.Pop, b.Layer, b.Capital})
	}
	sort.Slice(starts, func(i, j int) bool { return starts[i].Pop > starts[j].Pop })
	if len(starts) > 60 {
		starts = starts[:60]
	}
	out := map[string]any{
		"genre": w.Genre, "world": w.Name, "year": w.Year, "powersName": sk.PowersName, "currency": sk.Currency,
		"species": sps, "callings": cs, "powers": ps, "starts": starts, "attrNames": AttrNames, "skillDesc": skillDesc,
		"attrPoints": 30, "powerPoints": 12,
	}
	b, _ := json.Marshal(out)
	return string(b)
}

func speciesAttr(s *SpeciesData) map[int]int {
	m := map[int]int{}
	add := func(k, v int) {
		if v != 0 {
			m[k] += v
		}
	}
	add(STR, clampi(s.Martial/2, -3, 4))
	add(AGI, clampi(s.Intrigue/2, -3, 4))
	add(TOU, clampi(s.Health/8, -3, 4))
	add(INT, clampi(s.Steward/2, -3, 4))
	add(EGO, clampi(s.Diplo/2, -3, 4))
	return m
}

type CharSpec struct {
	Name    string         `json:"name"`
	Species string         `json:"species"`
	Calling string         `json:"calling"`
	Attr    [6]int         `json:"attr"`
	Powers  map[string]int `json:"powers"`
	Start   int            `json:"start"`
}

func validName(s string) string {
	s = strings.TrimSpace(s)
	if len(s) > 24 {
		s = s[:24]
	}
	if s == "" {
		return "Wanderer"
	}
	return s
}

func (g *Game) describeStart() string {
	sp := g.W.specByID[g.P.Species]
	spn := "wanderer"
	if sp != nil {
		spn = strings.ToLower(sp.Name)
	}
	b := g.W.burgByID[g.StartBurg]
	bn := "the wilds"
	if b != nil {
		bn = b.Name
	}
	return fmt.Sprintf("You are {{Y|%s}}, a %s %s, setting out from {{W|%s}} in the year %d of %s.", g.P.Name, spn, strings.ToLower(g.Sk.Calling(g.Calling)), bn, g.W.Year, g.W.Name)
}
