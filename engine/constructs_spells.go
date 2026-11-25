package engine

import (
	"fmt"
	"sort"
	"strings"
)

// Golemancy Constructs, Living Grimoire Spell Weaving & Cursed Relic Bargains (prefix gc).
// Saved state lives in g.ext("gc"), planet-local ("gc" in unLocalKeys).
// Side RNGs seeded from g.Seed+":gc:..." are used for generation; g.R only for play-time rolls.

func init() {
	unLocalKeys = append(unLocalKeys, "gc")
	turnHooks = append(turnHooks, gcTurn)
	killHooks = append(killHooks, gcKill)
	zoneHooks = append(zoneHooks, gcZone)
	enterHooks = append(enterHooks, gcEnter)
	talkHooks = append(talkHooks, gcTalk)
	sideHooks = append(sideHooks, gcSide)
	modeKeys["gc"] = gcKeys
	modeDraws["gc"] = gcDraw
	paEntries = append(paEntries, paEntry{
		Name:     "Constructs, grimoire & bargains",
		Keywords: "construct golem clockwork grimoire spell weave cursed relic bargain",
		Cat:      "Craft",
		Func:     func(g *Game) { g.gcOpen() },
	})
}

// ---- Part Tables for Constructs ----

type gcPartDef struct {
	ID     string
	Name   string
	Slot   string
	HP     int
	AV     int
	DV     int
	Speed  int
	Damage int
	Color  byte
	Note   string
}

var gcChassisParts = []gcPartDef{
	{ID: "brass", Name: "Brass Chassis", Slot: "chassis", HP: 32, AV: 5, Speed: 0, Color: CYellow, Note: "Balanced clockwork hull"},
	{ID: "iron", Name: "Iron Chassis", Slot: "chassis", HP: 46, AV: 8, Speed: -10, Color: CGray, Note: "Heavy armored plating"},
	{ID: "bone", Name: "Bone Chassis", Slot: "chassis", HP: 24, AV: 3, Speed: 10, Color: CWhite, Note: "Light ossuary frame"},
	{ID: "flesh", Name: "Flesh Chassis", Slot: "chassis", HP: 40, AV: 4, Speed: 5, Color: CRed, Note: "Self-knitting bio-frame"},
}

var gcLocoParts = []gcPartDef{
	{ID: "treads", Name: "Iron Treads", Slot: "locomotion", HP: 8, AV: 2, DV: -1, Speed: 90, Note: "Steady traction"},
	{ID: "spider_legs", Name: "Spider Legs", Slot: "locomotion", HP: 0, AV: 0, DV: 3, Speed: 110, Note: "Nimble multi-limb gait"},
	{ID: "wings", Name: "Pinion Wings", Slot: "locomotion", HP: -4, AV: 0, DV: 5, Speed: 125, Note: "Swift hovering stride"},
	{ID: "heavy_legs", Name: "Heavy Legs", Slot: "locomotion", HP: 14, AV: 3, DV: -2, Speed: 85, Note: "Braced shock pistons"},
}

var gcWeaponParts = []gcPartDef{
	{ID: "sawblade", Name: "Rotary Sawblade", Slot: "weapon", Damage: 12, Note: "Rending melee edge"},
	{ID: "bolt_thrower", Name: "Bolt Thrower", Slot: "weapon", Damage: 10, Note: "Ranged spring launcher"},
	{ID: "maul", Name: "Crushing Maul", Slot: "weapon", Damage: 14, Note: "Heavy armor-cracking blow"},
	{ID: "claws", Name: "Articulated Claws", Slot: "weapon", Damage: 11, Note: "Fast slashing talons"},
}

var gcCoreParts = []gcPartDef{
	{ID: "steam", Name: "Steam Boiler", Slot: "core", HP: 10, Damage: 2, Speed: 5, Note: "Reliable pressure drive"},
	{ID: "ley_crystal", Name: "Ley Crystal", Slot: "core", HP: 6, Damage: 5, Speed: 10, Note: "Arcane resonance core"},
	{ID: "bio_heart", Name: "Bio-Heart", Slot: "core", HP: 16, Damage: 1, Speed: 0, Note: "Regenerates hull over time"},
	{ID: "void_cell", Name: "Void Cell", Slot: "core", HP: 4, Damage: 7, Speed: 15, Note: "High-output dark battery"},
}

func gcFindPart(list []gcPartDef, id string) *gcPartDef {
	for i := range list {
		if list[i].ID == id {
			return &list[i]
		}
	}
	return nil
}

// ---- Spell Weaving Tables ----

type gcWordDef struct {
	ID       string
	Kind     string // shape, element, modifier
	Name     string
	Power    int
	Cooldown int
	Range    int
	Desc     string
}

var gcShapes = []gcWordDef{
	{ID: "bolt", Kind: "shape", Name: "Bolt", Power: 12, Cooldown: 3, Range: 7, Desc: "Focused single-target lance"},
	{ID: "cone", Kind: "shape", Name: "Cone", Power: 10, Cooldown: 5, Range: 4, Desc: "Sweeping fan hitting up to 3 foes"},
	{ID: "nova", Kind: "shape", Name: "Nova", Power: 11, Cooldown: 6, Range: 3, Desc: "Burst striking all nearby foes"},
	{ID: "trap", Kind: "shape", Name: "Trap", Power: 14, Cooldown: 4, Range: 5, Desc: "Inscribes a triggered glyph"},
}

var gcElements = []gcWordDef{
	{ID: "fire", Kind: "element", Name: "Fire", Power: 4, Cooldown: 1, Desc: "Scorches and ignites targets"},
	{ID: "frost", Kind: "element", Name: "Frost", Power: 3, Cooldown: 1, Desc: "Chills and slows foes"},
	{ID: "shock", Kind: "element", Name: "Shock", Power: 4, Cooldown: 1, Desc: "Jolts with lightning"},
	{ID: "acid", Kind: "element", Name: "Acid", Power: 3, Cooldown: 1, Desc: "Corrodes armor"},
	{ID: "void", Kind: "element", Name: "Void", Power: 6, Cooldown: 2, Desc: "Tears essence with dark force"},
}

var gcModifiers = []gcWordDef{
	{ID: "homing", Kind: "modifier", Name: "Homing", Power: 3, Cooldown: 1, Desc: "Seeks the nearest foe unerringly"},
	{ID: "vampiric", Kind: "modifier", Name: "Vampiric", Power: 1, Cooldown: 2, Desc: "Drains life back to the caster"},
	{ID: "silent", Kind: "modifier", Name: "Silent", Power: 0, Cooldown: 0, Desc: "Casts without sound or alert"},
	{ID: "chain", Kind: "modifier", Name: "Chain", Power: 2, Cooldown: 2, Desc: "Arcs to a second nearby foe"},
}

func gcFindWord(list []gcWordDef, id string) *gcWordDef {
	for i := range list {
		if list[i].ID == id {
			return &list[i]
		}
	}
	return nil
}

// ---- Cursed Relic Bargains ----

type gcBargainDef struct {
	ID   string
	Name string
	Desc string
}

var gcBargains = []gcBargainDef{
	{
		ID:   "sleepless_edge",
		Name: "Sleepless Edge",
		Desc: "Grows sharper each day without sleep (+2 damage per sleepless day); sleeping resets its edge.",
	},
	{
		ID:   "usurers_band",
		Name: "Usurer's Band",
		Desc: "Turns a lethal hit into survival at 1 HP in exchange for 500 gold of underworld debt.",
	},
	{
		ID:   "blood_ink_seal",
		Name: "Blood-Ink Seal",
		Desc: "Allows casting woven spells while on cooldown by paying HP instead of waiting.",
	},
}

func gcFindBargain(id string) *gcBargainDef {
	for i := range gcBargains {
		if gcBargains[i].ID == id {
			return &gcBargains[i]
		}
	}
	return nil
}

// ---- Saved State ----

type gcConstruct struct {
	ID         string `json:"id"`
	Name       string `json:"n"`
	Chassis    string `json:"ch"`
	Locomotion string `json:"lo"`
	Weapon     string `json:"wp"`
	Core       string `json:"co"`
	MaxHP      int    `json:"mhp"`
	HP         int    `json:"hp"`
	AV         int    `json:"av"`
	DV         int    `json:"dv"`
	Speed      int    `json:"sp"`
	Damage     int    `json:"dmg"`
	Deployed   bool   `json:"dep,omitempty"`
	ActorID    int    `json:"aid,omitempty"`
	Kills      int    `json:"k,omitempty"`
}

type gcSpell struct {
	ID         string `json:"id"`
	Name       string `json:"n"`
	Shape      string `json:"sh"`
	Element    string `json:"el"`
	Modifier   string `json:"mo"`
	Power      int    `json:"pw"`
	Cooldown   int    `json:"cd"`
	ReadyTurn  int    `json:"rt,omitempty"`
	Range      int    `json:"rg"`
	Casts      int    `json:"ca,omitempty"`
	SoldCopies int    `json:"sc,omitempty"`
}

type gcRelic struct {
	ID        string `json:"id"`
	BargainID string `json:"bid"`
	Name      string `json:"n"`
	Desc      string `json:"d"`
	Bound     bool   `json:"b,omitempty"`
	BoundDay  int    `json:"bd,omitempty"`
	Stacks    int    `json:"st,omitempty"`
	Triggers  int    `json:"tr,omitempty"`
}

type gcState struct {
	Constructs     []*gcConstruct  `json:"con,omitempty"`
	Words          map[string]bool `json:"wd,omitempty"`
	Spells         []*gcSpell      `json:"sp,omitempty"`
	Relics         []*gcRelic      `json:"rl,omitempty"`
	SleeplessDays  int             `json:"sd,omitempty"`
	LastTickDay    int             `json:"ltd,omitempty"`
	UnderworldDebt int             `json:"uwd,omitempty"`
	NextSeq        int             `json:"sq,omitempty"`

	tab int
	sel int
	// UI builder cursors
	chIdx int
	loIdx int
	wpIdx int
	coIdx int
	shIdx int
	elIdx int
	moIdx int
}

func gcSt(g *Game) *gcState {
	st := g.ext("gc", func() any { return &gcState{} }).(*gcState)
	if st.Words == nil {
		st.Words = map[string]bool{
			"bolt": true, "fire": true, "homing": true,
		}
	}
	return st
}

// ---- Golemancy, Clockwork & Bio-Constructs ----

// gcAssembleConstruct builds a custom construct from 4 part slots.
func (g *Game) gcAssembleConstruct(name, chassis, loco, weapon, core string) (*gcConstruct, bool) {
	ch := gcFindPart(gcChassisParts, chassis)
	lo := gcFindPart(gcLocoParts, loco)
	wp := gcFindPart(gcWeaponParts, weapon)
	co := gcFindPart(gcCoreParts, core)
	if ch == nil || lo == nil || wp == nil || co == nil {
		g.Msg("Invalid construct parts specified.")
		return nil, false
	}
	st := gcSt(g)
	st.NextSeq++
	if strings.TrimSpace(name) == "" {
		r := NewRng(fmt.Sprintf("%s:gc:con:%d", g.Seed, st.NextSeq))
		prefixes := []string{"Automaton", "Sentinel", "Stitchwork", "Golem", "Warden", "Strider"}
		name = fmt.Sprintf("%s-%d (%s)", Pick(r, prefixes), st.NextSeq, ch.ID)
	}
	maxHP := maxi(16, ch.HP+lo.HP+co.HP)
	av := maxi(1, ch.AV+lo.AV)
	dv := maxi(1, 5+lo.DV)
	spd := maxi(60, lo.Speed+ch.Speed+co.Speed)
	dmg := maxi(4, wp.Damage+co.Damage)

	c := &gcConstruct{
		ID:         fmt.Sprintf("con_%d", st.NextSeq),
		Name:       name,
		Chassis:    ch.ID,
		Locomotion: lo.ID,
		Weapon:     wp.ID,
		Core:       co.ID,
		MaxHP:      maxHP,
		HP:         maxHP,
		AV:         av,
		DV:         dv,
		Speed:      spd,
		Damage:     dmg,
	}
	st.Constructs = append(st.Constructs, c)
	g.Msg(fmt.Sprintf("{{G|Assembled construct %s (HP %d, AV %d, DMG %d, SPD %d).}}", c.Name, c.MaxHP, c.AV, c.Damage, c.Speed))
	return c, true
}

func (g *Game) gcFindConstruct(id string) *gcConstruct {
	st := gcSt(g)
	for _, c := range st.Constructs {
		if c.ID == id {
			return c
		}
	}
	return nil
}

// gcDeployConstruct spawns the assembled construct as an active follower actor in combat.
func (g *Game) gcDeployConstruct(id string) (*Actor, bool) {
	if g.Cur == nil {
		return nil, false
	}
	c := g.gcFindConstruct(id)
	if c == nil {
		return nil, false
	}
	if c.Deployed {
		for _, a := range g.Cur.Actors {
			if a != nil && a.ID == c.ActorID && a.HP > 0 {
				return a, true
			}
		}
	}
	ch := gcFindPart(gcChassisParts, c.Chassis)
	col := CYellow
	if ch != nil {
		col = ch.Color
	}
	r := NewRng(fmt.Sprintf("%s:gc:dep:%s:%d", g.Seed, c.ID, g.Turn))
	x, y, ok := g.Cur.FreeSpot(r, func(x, y int) bool { return abs(x-g.P.X)+abs(y-g.P.Y) <= 3 })
	if !ok {
		x, y = clampi(g.P.X+1, 1, MW-2), g.P.Y
	}
	hp := c.HP
	if hp <= 0 {
		hp = c.MaxHP / 2
		c.HP = hp
	}
	act := &Actor{
		ID:      g.newID(),
		Name:    c.Name,
		Proper:  true,
		Glyph:   'G',
		F:       col,
		X:       x,
		Y:       y,
		Tpl:     "gc_construct",
		Faction: "player",
		Level:   maxi(1, g.P.Level),
		Attr:    [6]int{12 + c.Damage/2, 10 + c.DV/2, 12 + c.MaxHP/10, 8, 10, 6},
		HP:      hp,
		MaxHP:   c.MaxHP,
		AVNat:   c.AV,
		DVNat:   c.DV,
		Speed:   c.Speed,
		AI:      "follower",
		Role:    "construct",
		Title:   fmt.Sprintf("%s/%s", c.Chassis, c.Core),
		Desc:    fmt.Sprintf("A custom %s construct with %s, %s, and a %s core.", c.Chassis, c.Locomotion, c.Weapon, c.Core),
		Comp: &Companion{
			Loyalty:  100,
			Summoned: true,
			Since:    g.Turn,
		},
	}
	act.Set("gc_construct")
	g.Cur.Actors = append(g.Cur.Actors, act)
	c.Deployed = true
	c.ActorID = act.ID
	g.Msg(fmt.Sprintf("{{C|You deploy %s onto the field!}}", c.Name))
	return act, true
}

// gcRecallConstruct recalls a deployed construct from the active zone.
func (g *Game) gcRecallConstruct(id string) bool {
	c := g.gcFindConstruct(id)
	if c == nil || !c.Deployed {
		return false
	}
	if g.Cur != nil {
		for i, a := range g.Cur.Actors {
			if a != nil && a.ID == c.ActorID {
				c.HP = maxi(1, a.HP)
				g.Cur.Actors = append(g.Cur.Actors[:i], g.Cur.Actors[i+1:]...)
				break
			}
		}
	}
	c.Deployed = false
	c.ActorID = 0
	g.Msg(fmt.Sprintf("You recall %s into your workshop pack.", c.Name))
	return true
}

// ---- Living Grimoire & Spell Weaving ----

// gcDiscoverWord adds a rune/word to the player's Living Grimoire.
func (g *Game) gcDiscoverWord(word string) bool {
	if gcFindWord(gcShapes, word) == nil && gcFindWord(gcElements, word) == nil && gcFindWord(gcModifiers, word) == nil {
		return false
	}
	st := gcSt(g)
	if st.Words[word] {
		return false
	}
	st.Words[word] = true
	g.Msg("{{M|You inscribe a new rune word into your Living Grimoire: " + word + ".}}")
	return true
}

// gcWeaveSpell combines Shape + Element + Modifier into a custom named spell.
func (g *Game) gcWeaveSpell(name, shape, element, modifier string) (*gcSpell, bool) {
	sh := gcFindWord(gcShapes, shape)
	el := gcFindWord(gcElements, element)
	mo := gcFindWord(gcModifiers, modifier)
	if sh == nil || el == nil || mo == nil {
		g.Msg("Unknown spell word in weave.")
		return nil, false
	}
	st := gcSt(g)
	st.Words[sh.ID] = true
	st.Words[el.ID] = true
	st.Words[mo.ID] = true
	st.NextSeq++
	if strings.TrimSpace(name) == "" {
		name = fmt.Sprintf("%s %s %s", mo.Name, el.Name, sh.Name)
	}
	power := sh.Power + el.Power + mo.Power + maxi(0, g.P.M(INT)*2)
	cd := maxi(1, sh.Cooldown+el.Cooldown+mo.Cooldown)
	sp := &gcSpell{
		ID:       fmt.Sprintf("sp_%d", st.NextSeq),
		Name:     name,
		Shape:    sh.ID,
		Element:  el.ID,
		Modifier: mo.ID,
		Power:    power,
		Cooldown: cd,
		Range:    sh.Range,
	}
	st.Spells = append(st.Spells, sp)
	g.Msg(fmt.Sprintf("{{M|Woven spell inscribed: %s (%s + %s + %s, power %d, cd %d).}}",
		sp.Name, sp.Shape, sp.Element, sp.Modifier, sp.Power, sp.Cooldown))
	return sp, true
}

func (g *Game) gcFindSpell(id string) *gcSpell {
	st := gcSt(g)
	for _, sp := range st.Spells {
		if sp.ID == id {
			return sp
		}
	}
	return nil
}

func (g *Game) gcHasBargain(bargainID string) *gcRelic {
	st := gcSt(g)
	for _, r := range st.Relics {
		if r.BargainID == bargainID && r.Bound {
			return r
		}
	}
	return nil
}

// gcCastWoven casts a woven grimoire spell toward (tx, ty) or nearest hostile in range.
func (g *Game) gcCastWoven(id string, tx, ty int) (int, bool) {
	sp := g.gcFindSpell(id)
	if sp == nil || g.Cur == nil {
		return 0, false
	}
	usedBloodInk := false
	if g.Turn < sp.ReadyTurn {
		seal := g.gcHasBargain("blood_ink_seal")
		if seal == nil {
			g.Msg(fmt.Sprintf("%s is still recharging (%d turns left).", sp.Name, sp.ReadyTurn-g.Turn))
			return 0, false
		}
		hpCost := maxi(3, sp.Power/3)
		if g.P.HP <= hpCost {
			g.Msg("Not enough blood to feed the Blood-Ink Seal.")
			return 0, false
		}
		g.P.HP -= hpCost
		seal.Triggers++
		usedBloodInk = true
	}

	// Collect hostile targets in current zone.
	var foes []*Actor
	for _, a := range g.Cur.Actors {
		if a != nil && a.HP > 0 && a.AI != "follower" && a.Faction != "player" && g.Hostile(a) {
			foes = append(foes, a)
		}
	}
	sort.Slice(foes, func(i, j int) bool {
		di := abs(foes[i].X-g.P.X) + abs(foes[i].Y-g.P.Y)
		dj := abs(foes[j].X-g.P.X) + abs(foes[j].Y-g.P.Y)
		return di < dj
	})

	var targets []*Actor
	switch sp.Shape {
	case "nova":
		for _, f := range foes {
			if abs(f.X-g.P.X)+abs(f.Y-g.P.Y) <= sp.Range+1 {
				targets = append(targets, f)
			}
		}
	case "cone":
		for _, f := range foes {
			if abs(f.X-g.P.X)+abs(f.Y-g.P.Y) <= sp.Range+2 && len(targets) < 3 {
				targets = append(targets, f)
			}
		}
	case "trap":
		if a := g.Cur.ActorAt(tx, ty); a != nil && a.HP > 0 && a != g.P {
			targets = append(targets, a)
		} else if len(foes) > 0 {
			targets = append(targets, foes[0])
		}
	default: // bolt
		if a := g.Cur.ActorAt(tx, ty); a != nil && a.HP > 0 && a != g.P {
			targets = append(targets, a)
		} else if len(foes) > 0 {
			targets = append(targets, foes[0])
		}
	}

	baseDmg := sp.Power + g.R.Int(0, 4)
	if sp.Modifier == "homing" {
		baseDmg += baseDmg / 4
	}
	// Sleepless Edge bargain adds sharpness bonus if bound.
	if edge := g.gcHasBargain("sleepless_edge"); edge != nil && edge.Stacks > 0 {
		baseDmg += edge.Stacks * 2
	}

	totalDmg := 0
	for _, t := range targets {
		dmg := maxi(1, baseDmg)
		g.hurt(t, dmg, g.P, sp.Element)
		totalDmg += dmg
		switch sp.Element {
		case "fire":
			t.AddSt("burning", 3)
		case "frost":
			t.AddSt("frozen", 2)
		case "shock":
			t.AddSt("stunned", 1)
		}
	}
	// Chain modifier arcs 50% damage to an extra foe not yet hit.
	if sp.Modifier == "chain" && len(targets) > 0 && len(foes) > len(targets) {
		hit := map[int]bool{}
		for _, t := range targets {
			hit[t.ID] = true
		}
		for _, f := range foes {
			if !hit[f.ID] {
				arc := maxi(1, baseDmg/2)
				g.hurt(f, arc, g.P, sp.Element)
				totalDmg += arc
				break
			}
		}
	}
	// Vampiric modifier heals the caster.
	if sp.Modifier == "vampiric" && totalDmg > 0 {
		heal := maxi(1, totalDmg*3/10)
		g.P.HP = mini(g.P.MaxHP, g.P.HP+heal)
	}

	sp.Casts++
	if !usedBloodInk {
		sp.ReadyTurn = g.Turn + sp.Cooldown
	}
	note := ""
	if usedBloodInk {
		note = " (paid in blood)"
	}
	if sp.Modifier == "silent" {
		g.Msg(fmt.Sprintf("{{C|You weave %s in dead silence%s for %d %s damage.}}", sp.Name, note, totalDmg, sp.Element))
	} else {
		g.Msg(fmt.Sprintf("{{M|You cast %s%s for %d %s damage!}}", sp.Name, note, totalDmg, sp.Element))
	}
	return totalDmg, true
}

// gcSellSpellCopy transcribes and sells/teaches a copy of a woven spell to a scholar.
func (g *Game) gcSellSpellCopy(id string, buyer *Actor) (int, bool) {
	sp := g.gcFindSpell(id)
	if sp == nil {
		return 0, false
	}
	price := maxi(25, sp.Power*3-sp.SoldCopies*5)
	g.P.Gold += price
	sp.SoldCopies++
	buyerName := "a traveling scholar"
	if buyer != nil && buyer.Name != "" {
		buyerName = buyer.Name
	}
	g.Msg(fmt.Sprintf("{{G|You transcribe a copy of %s for %s and earn %d %s.}}", sp.Name, buyerName, price, g.Sk.Currency))
	return price, true
}

// ---- Cursed Relics with Active Bargains ----

// gcGenRelic deterministically creates a bargain-cursed relic record.
func (g *Game) gcGenRelic(bargainID string) *gcRelic {
	def := gcFindBargain(bargainID)
	if def == nil {
		return nil
	}
	st := gcSt(g)
	for _, r := range st.Relics {
		if r.BargainID == bargainID {
			return r
		}
	}
	rng := NewRng(g.Seed + ":gc:relic:" + bargainID)
	st.NextSeq++
	rel := &gcRelic{
		ID:        fmt.Sprintf("rel_%d_%d", st.NextSeq, rng.Int(100, 999)),
		BargainID: def.ID,
		Name:      def.Name,
		Desc:      def.Desc,
	}
	st.Relics = append(st.Relics, rel)
	return rel
}

// gcBindBargain binds a cursed relic and activates its dynamic bargain trade-off.
func (g *Game) gcBindBargain(bargainID string) (*gcRelic, bool) {
	rel := g.gcGenRelic(bargainID)
	if rel == nil {
		return nil, false
	}
	if rel.Bound {
		return rel, true
	}
	rel.Bound = true
	rel.BoundDay = g.Day()
	st := gcSt(g)
	st.LastTickDay = g.Day()
	item := &Item{
		ID:     g.newID(),
		Name:   rel.Name,
		Glyph:  '*',
		F:      CMagenta,
		Kind:   "artifact",
		Effect: "gc_bargain_" + rel.BargainID,
		Value:  300,
		Weight: 1,
		Ident:  true,
		Lore:   rel.Desc,
	}
	g.give(item)
	g.Msg(fmt.Sprintf("{{M|You bind the cursed relic %s: %s}}", rel.Name, rel.Desc))
	g.deed("relic", fmt.Sprintf("%s bound the cursed bargain of %s.", g.P.Name, rel.Name), 0, 0)
	return rel, true
}

// gcUnbindBargain breaks a bound relic bargain.
func (g *Game) gcUnbindBargain(bargainID string) bool {
	rel := g.gcHasBargain(bargainID)
	if rel == nil {
		return false
	}
	rel.Bound = false
	rel.Stacks = 0
	g.Msg(fmt.Sprintf("{{O|You unbind %s; its bargain falls silent.}}", rel.Name))
	return true
}

// gcPreventLethal triggers the Usurer's Band bargain when the player faces a lethal blow.
func (g *Game) gcPreventLethal() bool {
	band := g.gcHasBargain("usurers_band")
	if band == nil || g.P == nil {
		return false
	}
	st := gcSt(g)
	g.P.HP = maxi(1, g.P.MaxHP/4)
	g.Dead = false
	st.UnderworldDebt += 500
	band.Triggers++
	if bd := g.localBurg(); bd != nil {
		g.uwAddBounty(bd.State, 150, " (Usurer's Band debt)")
	}
	g.Msg(fmt.Sprintf("{{M|The Usurer's Band burns white-hot! Death is deferred for 500 gold of underworld debt (total debt: %d).}}", st.UnderworldDebt))
	return true
}

// gcTickSleepless advances Sleepless Edge sharpness stacks across sleepless days.
func (g *Game) gcTickSleepless() {
	st := gcSt(g)
	edge := g.gcHasBargain("sleepless_edge")
	if edge == nil {
		return
	}
	day := g.Day()
	if day > st.LastTickDay {
		diff := day - st.LastTickDay
		st.SleeplessDays += diff
		edge.Stacks = clampi(st.SleeplessDays, 0, 10)
		st.LastTickDay = day
		g.Msg(fmt.Sprintf("{{R|Sleepless Edge drinks your waking hours (+%d bonus damage, %d sleepless days).}}", edge.Stacks*2, st.SleeplessDays))
	}
}

// ---- Hooks ----

func gcTurn(g *Game) {
	st := gcSt(g)
	// Sync bio_heart construct regen.
	if g.Cur != nil && g.Turn%10 == 0 {
		for _, c := range st.Constructs {
			if !c.Deployed || c.Core != "bio_heart" {
				continue
			}
			for _, a := range g.Cur.Actors {
				if a != nil && a.ID == c.ActorID && a.HP > 0 && a.HP < a.MaxHP {
					a.HP = mini(a.MaxHP, a.HP+2)
					c.HP = a.HP
				}
			}
		}
	}
	// Tick Sleepless Edge daily sharpness.
	g.gcTickSleepless()
	// Check Usurer's Band lethal protection.
	if g.P != nil && (g.P.HP <= 0 || g.Dead) && g.gcHasBargain("usurers_band") != nil {
		g.gcPreventLethal()
	}
}

func gcKill(g *Game, d, src *Actor) {
	if src == nil || !src.Has("gc_construct") {
		return
	}
	st := gcSt(g)
	for _, c := range st.Constructs {
		if c.ActorID == src.ID {
			c.Kills++
			break
		}
	}
}

func gcZone(g *Game, zn *Zone, _ *Rng) {
	if zn == nil || zn.Kind == "town" {
		return
	}
	// Deterministically place an occasional rune scrap in ruins/dungeons without touching g.R.
	r := NewRng(g.Seed + ":gc:zone:" + zn.Key)
	if !r.Chance(0.10) {
		return
	}
	allWords := []string{"cone", "nova", "trap", "frost", "shock", "acid", "void", "vampiric", "silent", "chain"}
	w := Pick(r, allWords)
	x, y, ok := zn.FreeSpot(r, nil)
	if !ok {
		return
	}
	zn.Items = append(zn.Items, &Item{
		ID:     g.newID(),
		Name:   "Rune Shard (" + w + ")",
		Glyph:  '?',
		F:      CMagenta,
		Kind:   "book",
		Effect: "gc_word:" + w,
		Value:  30,
		Weight: 1,
		X:      x,
		Y:      y,
		Lore:   "An etched shard bearing the spell-word '" + w + "'.",
	})
}

func gcEnter(g *Game, zn *Zone) {
	if zn == nil {
		return
	}
	// Keep construct deployment state synced when changing zones.
	st := gcSt(g)
	for _, c := range st.Constructs {
		if !c.Deployed {
			continue
		}
		found := false
		for _, a := range zn.Actors {
			if a != nil && a.ID == c.ActorID && a.HP > 0 {
				found = true
				break
			}
		}
		if !found {
			c.Deployed = false
			c.ActorID = 0
		}
	}
}

func gcSide(g *Game) []string {
	st := gcSt(g)
	var out []string
	dep := 0
	for _, c := range st.Constructs {
		if c.Deployed {
			dep++
		}
	}
	if dep > 0 {
		out = append(out, fmt.Sprintf("{{Y|Constructs active: %d}}", dep))
	}
	if edge := g.gcHasBargain("sleepless_edge"); edge != nil && edge.Stacks > 0 {
		out = append(out, fmt.Sprintf("{{R|Sleepless Edge +%d}}", edge.Stacks*2))
	}
	if st.UnderworldDebt > 0 {
		out = append(out, fmt.Sprintf("{{O|Relic Debt: %d}}", st.UnderworldDebt))
	}
	return out
}

func gcTalk(g *Game, a *Actor) []Option {
	if a == nil {
		return nil
	}
	st := gcSt(g)
	var out []Option
	if a.Role == "scholar" || a.Role == "scribe" || a.Role == "mage" || a.Role == "priest" {
		for _, sp := range st.Spells {
			sp := sp
			out = append(out, Option{fmt.Sprintf("{{M|Sell a copy of woven spell '%s'.}}", sp.Name), func() {
				if gold, ok := g.gcSellSpellCopy(sp.ID, a); ok {
					g.talkMenu(fmt.Sprintf("\"A clever weave! Here is %d %s for the transcription.\"", gold, g.Sk.Currency))
				}
			}})
		}
		out = append(out, Option{"{{C|Study a new grimoire rune word.}} {{K|(40 " + g.Sk.Currency + ")}}", func() {
			if g.P.Gold < 40 {
				g.talkMenu("\"Bring coin if you want my notes.\"")
				return
			}
			allWords := []string{"cone", "nova", "trap", "frost", "shock", "acid", "void", "vampiric", "silent", "chain"}
			for _, w := range allWords {
				if !st.Words[w] {
					g.P.Gold -= 40
					g.gcDiscoverWord(w)
					g.talkMenu("\"Trace the glyph carefully: '" + w + "'.\"")
					return
				}
			}
			g.talkMenu("\"You already know every rune in my folio.\"")
		}})
	}
	if a.Role == "smith" || a.Role == "merchant" || a.Role == "tinker" {
		out = append(out, Option{"{{Y|Open construct & grimoire workshop.}}", func() {
			g.gcOpen()
		}})
	}
	return out
}

// ---- Interactive Mode "gc" Screen ----

func (g *Game) gcOpen() {
	st := gcSt(g)
	st.tab = 0
	st.sel = 0
	g.Mode = "gc"
}

func gcKeys(g *Game, k string) {
	st := gcSt(g)
	switch k {
	case "Escape", "q":
		g.Mode = "main"
		return
	case "Tab", "ArrowRight":
		st.tab = (st.tab + 1) % 3
		st.sel = 0
		return
	case "ArrowLeft":
		st.tab = (st.tab + 2) % 3
		st.sel = 0
		return
	case "ArrowUp", "N8", "k":
		st.sel = maxi(0, st.sel-1)
		return
	case "ArrowDown", "N2", "j":
		st.sel++
		return
	}
	switch st.tab {
	case 0: // Constructs
		switch k {
		case "1":
			st.chIdx = (st.chIdx + 1) % len(gcChassisParts)
		case "2":
			st.loIdx = (st.loIdx + 1) % len(gcLocoParts)
		case "3":
			st.wpIdx = (st.wpIdx + 1) % len(gcWeaponParts)
		case "4":
			st.coIdx = (st.coIdx + 1) % len(gcCoreParts)
		case "a":
			g.gcAssembleConstruct("",
				gcChassisParts[st.chIdx].ID,
				gcLocoParts[st.loIdx].ID,
				gcWeaponParts[st.wpIdx].ID,
				gcCoreParts[st.coIdx].ID)
		case "Enter", "d":
			if len(st.Constructs) > 0 {
				st.sel = clampi(st.sel, 0, len(st.Constructs)-1)
				c := st.Constructs[st.sel]
				if c.Deployed {
					g.gcRecallConstruct(c.ID)
				} else {
					g.gcDeployConstruct(c.ID)
				}
			}
		}
	case 1: // Living Grimoire
		switch k {
		case "1":
			st.shIdx = (st.shIdx + 1) % len(gcShapes)
		case "2":
			st.elIdx = (st.elIdx + 1) % len(gcElements)
		case "3":
			st.moIdx = (st.moIdx + 1) % len(gcModifiers)
		case "w":
			g.gcWeaveSpell("",
				gcShapes[st.shIdx].ID,
				gcElements[st.elIdx].ID,
				gcModifiers[st.moIdx].ID)
		case "Enter", "c":
			if len(st.Spells) > 0 {
				st.sel = clampi(st.sel, 0, len(st.Spells)-1)
				g.gcCastWoven(st.Spells[st.sel].ID, 0, 0)
			}
		}
	case 2: // Cursed Bargains
		st.sel = clampi(st.sel, 0, len(gcBargains)-1)
		def := gcBargains[st.sel]
		switch k {
		case "Enter", "b":
			if g.gcHasBargain(def.ID) != nil {
				g.gcUnbindBargain(def.ID)
			} else {
				g.gcBindBargain(def.ID)
			}
		}
	}
}

func gcDraw(g *Game) {
	s := g.sc
	st := gcSt(g)
	x, y, w, h := g.panel("Constructs, Grimoire & Bargains")
	bottom := y + h - 1
	tabs := []string{"Constructs", "Living Grimoire", "Cursed Bargains"}
	tx := x
	for i, t := range tabs {
		c := "K"
		if st.tab == i {
			c = "Y"
		}
		tx = s.TextB(tx, y, fmt.Sprintf("{{%s|[ %s ]}} ", c, t), CGray, CBgDark, x+w)
	}
	y += 2
	put := func(str string) {
		for _, l := range Wrap(str, w) {
			if y < bottom {
				s.TextB(x, y, l, CGray, CBgDark, x+w)
				y++
			}
		}
	}
	row := func(sel bool, str string) {
		if y >= bottom {
			return
		}
		bg := CBgDark
		if sel {
			bg = CBgSel
		}
		s.Fill(x, y, w, 1, bg)
		s.TextB(x, y, str, CGray, bg, x+w)
		y++
	}
	switch st.tab {
	case 0:
		put(fmt.Sprintf("{{Y|Blueprint:}} [1] %s  [2] %s  [3] %s  [4] %s",
			gcChassisParts[st.chIdx].ID, gcLocoParts[st.loIdx].ID,
			gcWeaponParts[st.wpIdx].ID, gcCoreParts[st.coIdx].ID))
		y++
		if len(st.Constructs) == 0 {
			put("{{K|No constructs assembled yet. Press a to build the blueprint above.}}")
		} else {
			st.sel = clampi(st.sel, 0, len(st.Constructs)-1)
			for i, c := range st.Constructs {
				state := "stowed"
				if c.Deployed {
					state = "deployed"
				}
				row(st.sel == i, fmt.Sprintf("{{W|%-22s}} {{K|[%s]}} HP %d/%d AV %d DMG %d (%s/%s)",
					c.Name, state, c.HP, c.MaxHP, c.AV, c.Damage, c.Chassis, c.Core))
			}
		}
		y++
		put("{{K|1-4: cycle parts  a: assemble  Enter/d: deploy or recall  Esc: close}}")
	case 1:
		put(fmt.Sprintf("{{M|Weave:}} [1] Shape: %s  [2] Element: %s  [3] Modifier: %s",
			gcShapes[st.shIdx].ID, gcElements[st.elIdx].ID, gcModifiers[st.moIdx].ID))
		y++
		if len(st.Spells) == 0 {
			put("{{K|Your grimoire has no woven spells yet. Press w to weave one.}}")
		} else {
			st.sel = clampi(st.sel, 0, len(st.Spells)-1)
			for i, sp := range st.Spells {
				cd := "ready"
				if g.Turn < sp.ReadyTurn {
					cd = fmt.Sprintf("cd %d", sp.ReadyTurn-g.Turn)
				}
				row(st.sel == i, fmt.Sprintf("{{C|%-24s}} {{K|[%s]}} %s+%s+%s  Pwr %d",
					sp.Name, cd, sp.Shape, sp.Element, sp.Modifier, sp.Power))
			}
		}
		y++
		put("{{K|1-3: cycle words  w: weave spell  Enter/c: cast woven  Esc: close}}")
	case 2:
		put(fmt.Sprintf("{{O|Underworld Relic Debt: %d %s}}  {{K|Sleepless Days: %d}}", st.UnderworldDebt, g.Sk.Currency, st.SleeplessDays))
		y++
		st.sel = clampi(st.sel, 0, len(gcBargains)-1)
		for i, b := range gcBargains {
			status := "unbound"
			if r := g.gcHasBargain(b.ID); r != nil {
				status = fmt.Sprintf("BOUND (stacks %d, triggers %d)", r.Stacks, r.Triggers)
			}
			row(st.sel == i, fmt.Sprintf("{{M|%-18s}} {{K|[%s]}} %s", b.Name, status, b.Desc))
		}
		y++
		put("{{K|Enter/b: bind or unbind bargain  Tab: switch tab  Esc: close}}")
	}
}
