package engine

import "fmt"

// Status table (WP4). Numbers follow QUD_REFERENCE 1.7. Statuses live in Actor.Status as name → turns left,
// so "level"-scaled values (shaken: -level DV) use a fixed level of 3 here.
//
// stunned, frozen and prone already cost 6 DV inside Actor.DV(), so the table lists prone with DV 0 to avoid
// stacking; its AGI penalty is not applied either for the same reason.

type skStatusDef struct {
	ID, Name string
	Glyph    rune
	DV       int // flat DV
	Hit      int // flat to-hit
	AGI      int // attribute deltas; DV and to-hit follow AGI as delta/2
	INT      int
	WIL      int
	EGO      int
	AV       int
	MA       int
	Speed    int // added to Quick()
	PV       int // extra penetration value for attackers hitting this creature
	Tick     func(g *Game, a *Actor)
}

var skStatuses = map[string]*skStatusDef{}

func skAddStatus(d *skStatusDef) { skStatuses[d.ID] = d }

func init() {
	for _, d := range []*skStatusDef{
		{ID: "dazed", Name: "Dazed", Glyph: '*', AGI: -4, INT: -4, Speed: -10},
		{ID: "prone", Name: "Prone", Glyph: '_', Speed: -80},
		{ID: "shaken", Name: "Shaken", Glyph: '!', DV: -3},
		{ID: "disoriented", Name: "Disoriented", Glyph: '?', DV: -3, MA: -3},
		{ID: "shamed", Name: "Shamed", Glyph: '~', DV: -4, Hit: -4, WIL: -4, EGO: -4, Speed: -10},
		{ID: "hobbled", Name: "Hobbled", Glyph: 'h', Speed: -50},
		{ID: "asleep", Name: "Asleep", Glyph: 'z', DV: -12, PV: 4},
		{ID: "sluggish", Name: "Sluggish", Glyph: 's', Speed: -25},
		{ID: "frenzied", Name: "Frenzied", Glyph: 'F', Speed: 10},
		{ID: "wartrance", Name: "War trance", Glyph: 'W', WIL: 6, Speed: 5},
		{ID: "cracked", Name: "Armor cracked", Glyph: 'x', AV: -2},
		{ID: "ill", Name: "Ill", Glyph: 'i', AGI: -2, Speed: -10},
		{ID: "terrified", Name: "Terrified", Glyph: 'T', DV: -2, Hit: -2, Tick: skTerrifiedTick},
		{ID: "exhausted", Name: "Exhausted", Glyph: 'e', AGI: -2, Speed: -15},
		{ID: "syphoned", Name: "Syphoned", Glyph: 'v', Tick: func(g *Game, a *Actor) { g.hurt(a, 1, nil, "life drain") }},
		{ID: "choking", Name: "Choking", Glyph: 'c', Tick: func(g *Game, a *Actor) { g.hurt(a, 2, nil, "choking") }},
		{ID: "tumbling", Name: "Tumbling", Glyph: 't', DV: 6},
		{ID: "trailblazing", Name: "Trailblazing", Glyph: '>', Speed: 25},
		{ID: "beguiled", Name: "Beguiled", Glyph: 'b'},
		{ID: "proselytized", Name: "Proselytized", Glyph: 'p'},
		{ID: "glotrot", Name: "Glotrot", Glyph: 'g'},
		{ID: "metabolizing", Name: "Metabolizing", Glyph: 'm'},
		{ID: "inspired", Name: "Inspired", Glyph: '+'},
	} {
		skAddStatus(d)
	}
}

func skStatusSum(a *Actor, f func(d *skStatusDef) int) int {
	if a == nil || len(a.Status) == 0 {
		return 0
	}
	n := 0
	for k, v := range a.Status {
		if v > 0 {
			if d := skStatuses[k]; d != nil {
				n += f(d)
			}
		}
	}
	return n
}

// skStatMod is the summed attribute delta of an attribute index (STR..EGO) from statuses.
func skStatMod(a *Actor, attr int) int {
	return skStatusSum(a, func(d *skStatusDef) int {
		switch attr {
		case AGI:
			return d.AGI
		case INT:
			return d.INT
		case WIL:
			return d.WIL
		case EGO:
			return d.EGO
		}
		return 0
	})
}

// skStatusDV is the DV change from the status table (hook into Actor.DV()).
func skStatusDV(a *Actor) int {
	return skStatusSum(a, func(d *skStatusDef) int { return d.DV + d.AGI/2 })
}

// skStatusToHit is the to-hit change from statuses (hook into swing/shoot).
func skStatusToHit(a *Actor) int {
	return skStatusSum(a, func(d *skStatusDef) int { return d.Hit + d.AGI/2 })
}

// skStatusAV is the AV change (cracked armor) (hook into Actor.AV()).
func skStatusAV(a *Actor) int { return skStatusSum(a, func(d *skStatusDef) int { return d.AV }) }

// skStatusSpeed is the Quick() change (hook into Actor.Quick()).
func skStatusSpeed(a *Actor) int { return skStatusSum(a, func(d *skStatusDef) int { return d.Speed }) }

// skStatusMA is the mental armor change.
func skStatusMA(a *Actor) int { return skStatusSum(a, func(d *skStatusDef) int { return d.MA }) }

// skStatusPV is the extra penetration attackers get against d (asleep: +4) (hook into swing).
func skStatusPV(d *Actor) int { return skStatusSum(d, func(s *skStatusDef) int { return s.PV }) }

// skStatusEffect runs a status' per-turn effect; call it from statusTick's loop for every status name.
func skStatusEffect(g *Game, a *Actor, name string) {
	if a == nil || a.HP <= 0 {
		return
	}
	if name == "poisoned" && a.St(name) == 1 && a.St("ill") == 0 && !a.Has("undead") && a.Tpl != "golem" {
		a.AddSt("ill", 20) // poison ends in sickness
		if a == g.P {
			g.Msg("{{g|The poison leaves you ill.}}")
		}
		return
	}
	if d := skStatuses[name]; d != nil && d.Tick != nil {
		d.Tick(g, a)
	}
}

func skTerrifiedTick(g *Game, a *Actor) {
	if a == g.P || g.Cur == nil || a.AI == "follower" {
		return
	}
	g.stepToward(a, g.distMap(g.P.X, g.P.Y, false), true)
}

// skWake wakes a sleeper after damage; it comes up dazed (hook into hurt() optional).
func skWake(g *Game, d *Actor) {
	if d != nil && d.St("asleep") > 0 && d.HP > 0 {
		delete(d.Status, "asleep")
		d.AddSt("dazed", 3)
		if g.visible(d.X, d.Y) {
			g.Msg(fmt.Sprintf("%s wakes, dazed.", g.The(d)))
		}
	}
}

// skStatusNames lists the active table statuses of a for display.
func skStatusNames(a *Actor) []string {
	var out []string
	for k, v := range a.Status {
		if d := skStatuses[k]; d != nil && v > 0 {
			out = append(out, d.Name)
		}
	}
	return out
}
