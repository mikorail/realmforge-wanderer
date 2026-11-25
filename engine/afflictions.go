package engine

import (
	"fmt"
	"sort"
	"strings"
)

// Afflictions (prefix dz): chronic, progressive and cursed conditions beyond the outbreak diseases of
// hazards.go. 24 conditions (afflictions_data.go) with stages that advance by days, symptoms through the
// status table (Speed, DV, to-hit, AV via skAddStatus ids "dz_<id>_<stage>", attributes through dzScoreMod)
// and cures that are found out one by one: a herb or brew (alchemy.go properties, tonics, raw herbs), a
// physician for coin, a priest's rite by faith and rank, a pilgrimage to a named wonder of the world, killing
// the source that bit or cursed you, returning an offering to the grave you disturbed, or a ward-amulet from a
// master (arrests it while carried). Lycanthropy, the vampiric taint and the god-mark are boons with a cost.
// Sources: hits from flagged creatures (dzOnHit when wired, else polled from HP loss next to a hostile),
// eaten or drunk things (polled from the pack: raw and rotten meat, bad water, a pool), cursed items in the
// pack (the markers of craft_explore.go: Extra "cursed", Effect "cursed"), cursed ground (the statuses of
// burials.go and es_wild.go: gravesick, edchill, edfey; crypts), tainted metal (Extra dztaint, placed in
// ruins and deeps), long exposure (dust, wet nights, cold, sea, darkness, poison, radiation), sequels of
// hz outbreaks and local epidemics (sick townsfolk). Companions and townsfolk can catch the contagious ones.
// Genre variants through g.Genre (substitutions: wasteland radiation sores, cyberpunk neural virus,
// biopunk spore bloom; skins for names). Screen mode "dz" (palette "Afflictions"). A diagnosis scene with a
// skill check is offered by physicians, priests, herbalists and masters (talk option "Ask about my
// condition"); what you learn goes to the case notes. Public afflictions go to dzRumourSink when set.
// State g.ext("dz") global (the body travels), g.ext("dzl") planet-local.

func init() {
	unLocalKeys = append(unLocalKeys, "dzl")
	for _, d := range dzDefs {
		dzByID[d.ID] = d
		for j, s := range d.Stages {
			id := dzStat(d.ID, j)
			skAddStatus(&skStatusDef{ID: id, Name: s.Name, Glyph: 'd', Speed: s.Fx.Speed, DV: s.Fx.DV, Hit: s.Fx.Hit, AV: s.Fx.AV})
			a := s.Fx.A
			dzAttr[id] = &a
			if !s.Night.zero() {
				na := s.Night.A
				skAddStatus(&skStatusDef{ID: id + "n", Name: s.Name + " (night)", Glyph: 'd', Speed: s.Night.Speed, DV: s.Night.DV, Hit: s.Night.Hit, AV: s.Night.AV})
				dzAttr[id+"n"] = &na
			}
		}
	}
	turnHooks = append(turnHooks, dzTurn)
	killHooks = append(killHooks, dzKill)
	enterHooks = append(enterHooks, dzEnter)
	zoneHooks = append(zoneHooks, dzZone)
	talkHooks = append(talkHooks, dzTalk)
	modeKeys["dz"] = dzKeys
	modeDraws["dz"] = func(g *Game) { g.dzDraw() }
	modeKeys["dzr"] = dzrKeys
	modeDraws["dzr"] = func(g *Game) { g.dzrDraw() }
	paEntries = append(paEntries,
		paEntry{Name: "Afflictions", Keywords: "disease curse illness sickness condition cure symptoms poison lycanthropy vampire fever", Cat: "Screen", Label: "screen",
			Func: func(g *Game) { g.dz().sel = 0; g.Mode = "dz" }},
		paEntry{Name: "Lay an offering at the grave", Keywords: "grave curse offering return tomb barrow ghost mummy", Cat: "Action", Label: "grave",
			Cond: func(g *Game) bool { return g.dzGraveCond() != nil },
			Func: func(g *Game) { g.dzLayOffering() }})
}

// dzRumourSink, when set, hears about public afflictions (kind "affliction").
var dzRumourSink func(g *Game, kind, subject, text string)

var (
	dzByID = map[string]*dzDef{}
	dzAttr = map[string]*[6]int{}
)

func dzStat(id string, stage int) string { return fmt.Sprintf("dz_%s_%d", id, stage) }

// ---------------------------------------------------------------- state

type dzCond struct {
	ID       string `json:"i"`
	Stage    int    `json:"s"`
	Got      int    `json:"g"`
	Next     int    `json:"n"`
	Arrest   int    `json:"a,omitempty"`
	Cycles   int    `json:"cy,omitempty"`
	Src      string `json:"sr,omitempty"`
	SrcKey   string `json:"sk,omitempty"`
	SrcName  string `json:"sn,omitempty"`
	SrcActor int    `json:"sa,omitempty"`
	Item     int    `json:"it,omitempty"`
	Diag     bool   `json:"dg,omitempty"`
	Planet   int    `json:"pl,omitempty"`
	Told     bool   `json:"to,omitempty"`
	Until    int    `json:"un,omitempty"` // companions: turn they recover
}

type dzState struct {
	Conds  []*dzCond       `json:"c,omitempty"`
	Comp   map[int]*dzCond `json:"cp"`
	Imm    map[string]int  `json:"im"`
	Hints  map[string]int  `json:"h"`
	Cured  int             `json:"cu,omitempty"`
	Ever   map[string]int  `json:"ev"`
	Log    []string        `json:"l,omitempty"`
	Meals  []int           `json:"m,omitempty"`
	Hooked bool            `json:"hk,omitempty"`
	Prayed int             `json:"pr,omitempty"`
	LastHP int             `json:"hp,omitempty"`
	// exposure counters in minutes
	Dark, Sun, Dust, Wet, Cold, Sea, Land, Poison, Iron, NoIron, Rad, Chill, Fey, Hz int

	sel          int
	prevIt       map[int]*Item
	prevN        map[int]int
	pH, pT       int
	remSel       int
	remList      []dzRem
	remCond      *dzCond
}

type dzlState struct {
	Towns   map[int]string `json:"t"`
	Shrines map[string]int `json:"s"`
	Said    map[string]int `json:"d"`
}

func (g *Game) dz() *dzState {
	st := g.ext("dz", func() any {
		return &dzState{Comp: map[int]*dzCond{}, Imm: map[string]int{}, Hints: map[string]int{}, Ever: map[string]int{}}
	}).(*dzState)
	if st.Comp == nil {
		st.Comp = map[int]*dzCond{}
	}
	if st.Imm == nil {
		st.Imm = map[string]int{}
	}
	if st.Hints == nil {
		st.Hints = map[string]int{}
	}
	if st.Ever == nil {
		st.Ever = map[string]int{}
	}
	return st
}

func (g *Game) dzl() *dzlState {
	st := g.ext("dzl", func() any {
		return &dzlState{Towns: map[int]string{}, Shrines: map[string]int{}, Said: map[string]int{}}
	}).(*dzlState)
	if st.Towns == nil {
		st.Towns = map[int]string{}
	}
	if st.Shrines == nil {
		st.Shrines = map[string]int{}
	}
	if st.Said == nil {
		st.Said = map[string]int{}
	}
	return st
}

func (g *Game) dzReady() bool { return g != nil && g.W != nil && g.P != nil && g.Cur != nil }

// ---------------------------------------------------------------- names and lookups

func (g *Game) dzDefName(d *dzDef) string {
	if s, ok := d.Skin[g.Genre]; ok {
		return s
	}
	return d.Name
}

func (g *Game) dzAllowed(d *dzDef) bool {
	return d.Genres == "" || strings.Contains(" "+d.Genres+" ", " "+g.Genre+" ")
}

func (g *Game) dzResolve(id string) string {
	if d := dzByID[id]; d != nil {
		if s, ok := d.Subst[g.Genre]; ok {
			return s
		}
	}
	return id
}

func (g *Game) dzFind(id string) *dzCond {
	for _, c := range g.dz().Conds {
		if c.ID == id {
			return c
		}
	}
	return nil
}

func (g *Game) dzCondName(c *dzCond) string {
	d := dzByID[c.ID]
	if d == nil {
		return "an ailment"
	}
	if c.Diag {
		return g.dzDefName(d)
	}
	return dzVague[d.Kind]
}

func (g *Game) dzNight() bool { h := g.Hour(); return h >= 20 || h < 5 }

func (g *Game) dzWard(id string) *Item {
	key := "dza:" + id
	for _, it := range g.P.Inv {
		if it.Extra != nil && it.Extra[key] > 0 {
			return it
		}
	}
	return nil
}

// ---------------------------------------------------------------- attribute and check channels

// dzScoreMod is the attribute change from afflictions (hook in Actor.Score; works for companions too).
func dzScoreMod(a *Actor, attr int) int {
	if a == nil || len(a.Status) == 0 || attr < 0 || attr > 5 {
		return 0
	}
	n := 0
	for k, v := range a.Status {
		if v > 0 && strings.HasPrefix(k, "dz_") {
			if f := dzAttr[k]; f != nil {
				n += f[attr]
			}
		}
	}
	return n
}

// dzCheckMod is a skill-check change from afflictions (hook in checkMod, optional).
func (g *Game) dzCheckMod(kind string) int {
	if g == nil || g.P == nil || g.extLive == nil {
		return 0
	}
	if _, ok := g.extLive["dz"]; !ok {
		return 0
	}
	n := 0
	for _, c := range g.dz().Conds {
		switch c.ID {
		case "night_blind":
			if kind == "perception" {
				n -= 1 + c.Stage
			}
		case "the_shakes":
			if kind == "tinkering" || kind == "acrobatics" {
				n -= 1 + c.Stage
			}
		case "rot_blight":
			if kind == "persuasion" || kind == "bluff" {
				n -= 1 + 2*c.Stage
			}
		case "hollow_soul":
			if kind == "empathy" || kind == "persuasion" {
				n -= 1 + c.Stage
			}
		case "god_mark":
			if kind == "persuasion" || kind == "intimidate" {
				n += 1 + c.Stage
			}
		case "witch_hex":
			n -= c.Stage
		}
	}
	return n
}

// ---------------------------------------------------------------- infection

func (g *Game) dzHasAny(id string) bool {
	if g.dzFind(id) != nil {
		return true
	}
	return false
}

// dzInfect gives the player an affliction. src is the plain-words source; att the creature that caused it.
func (g *Game) dzInfect(id, src string, att *Actor) bool {
	if !g.dzReady() {
		return false
	}
	id = g.dzResolve(id)
	d := dzByID[id]
	if d == nil || !g.dzAllowed(d) {
		return false
	}
	st := g.dz()
	if st.Imm[id] > g.Turn || g.dzHasAny(id) {
		return false
	}
	if it := g.dzWard(id); it != nil {
		g.Msg("{{C|Your ward-amulet grows warm and turns something aside.}}")
		return false
	}
	c := &dzCond{ID: id, Got: g.Day(), Src: src, SrcKey: g.Cur.Key, Planet: g.W.uniIdx, SrcName: g.Cur.Name}
	if att != nil {
		c.SrcActor = att.ID
		att.Set("dzsrc")
	}
	c.Next = g.Turn + d.Stages[0].Days*DayLen
	st.Conds = append(st.Conds, c)
	st.Ever[id]++
	g.dzLog(fmt.Sprintf("Day %d: %s", g.Day()+1, strings.TrimSuffix(src, ".")))
	g.dzSymptoms(c, d, "{{R|Something is wrong with you.}} ")
	if st.Ever["__hint"] == 0 {
		st.Ever["__hint"] = 1
		g.Msg("{{c|Open the Afflictions screen from the palette (;). A physician, priest or herbalist can tell you more.}}")
	}
	return true
}

func (g *Game) dzLog(s string) {
	st := g.dz()
	st.Log = append(st.Log, s)
	if len(st.Log) > 40 {
		st.Log = st.Log[len(st.Log)-40:]
	}
}

func (g *Game) dzSymptoms(c *dzCond, d *dzDef, lead string) {
	s := d.Stages[clampi(c.Stage, 0, len(d.Stages)-1)]
	if c.Diag {
		g.Msg(fmt.Sprintf("%s{{O|%s, %s:}} %s", lead, g.dzDefName(d), strings.ToLower(s.Name), s.Text))
		return
	}
	g.Msg(lead + s.Text)
}

// dzRemove ends a condition (cured, burnt out, or its source gone).
func (g *Game) dzRemove(c *dzCond, why string, immDays int) {
	st := g.dz()
	out := st.Conds[:0]
	for _, o := range st.Conds {
		if o != c {
			out = append(out, o)
		}
	}
	st.Conds = out
	g.dzClearStatuses(g.P, c.ID)
	st.Imm[c.ID] = g.Turn + immDays*DayLen
	st.Cured++
	if d := dzByID[c.ID]; d != nil {
		g.dzLog(fmt.Sprintf("Day %d: %s ended (%s).", g.Day()+1, g.dzDefName(d), why))
	}
}

func (g *Game) dzClearStatuses(a *Actor, id string) {
	if a == nil || a.Status == nil {
		return
	}
	pre := "dz_" + id + "_"
	for k := range a.Status {
		if strings.HasPrefix(k, pre) {
			delete(a.Status, k)
		}
	}
}

func dzSetSt(a *Actor, id string, n int) {
	if a.Status == nil {
		a.Status = map[string]int{}
	}
	a.Status[id] = n
}

// dzApplyStatuses refreshes the symptom statuses of one condition on an actor.
func (g *Game) dzApplyStatuses(a *Actor, c *dzCond, night bool) {
	d := dzByID[c.ID]
	if d == nil || a == nil {
		return
	}
	g.dzClearStatuses(a, c.ID)
	id := dzStat(c.ID, clampi(c.Stage, 0, len(d.Stages)-1))
	dzSetSt(a, id, 10)
	if night && !d.Stages[clampi(c.Stage, 0, len(d.Stages)-1)].Night.zero() {
		dzSetSt(a, id+"n", 10)
	}
}

// ---------------------------------------------------------------- sources: hits, food and water

// dzOnHit is the public hook: call it when the player is hurt by att (hurt() after the damage).
func dzOnHit(g *Game, att, def *Actor, dmg int) {
	if g == nil || g.P == nil || def != g.P || att == nil || dmg <= 0 {
		return
	}
	g.dz().Hooked = true
	g.dzHitCheck(att, dmg)
}

func (g *Game) dzHitCheck(att *Actor, dmg int) {
	if att == nil || att.AI == "follower" || att == g.P {
		return
	}
	name := strings.ToLower(att.Tpl + " " + att.Name)
	tough := g.P.M(TOU)
	for _, cr := range dzCarriers {
		if !strings.Contains(name, cr.Sub) {
			continue
		}
		ch := float64(cr.P+dmg/2-3*tough) / 100
		if ch < 0.03 {
			ch = 0.03
		}
		if ch > 0.6 {
			ch = 0.6
		}
		if g.R.Chance(ch) {
			if g.dzInfect(cr.ID, g.dzHitText(att), att) {
				return
			}
		}
		return
	}
	if s := bsByTpl[att.Tpl]; s != nil {
		if (s.Group == "ins" || s.Group == "arach") && (g.Cur.Biome == 12 || g.Cur.Biome == 7 || g.Cur.Biome == 8 || g.Cur.Biome == 5) && g.R.Chance(0.15) {
			g.dzInfect("marsh_fever", "Bitten by "+g.the(att)+" in the damp.", att)
			return
		}
		if s.Has("poison") && g.R.Chance(0.12) {
			g.dzInfect("venom_sick", "Poisoned by "+g.the(att)+".", att)
		}
	} else if att.Has("poison") && g.R.Chance(0.1) {
		g.dzInfect("venom_sick", "Poisoned by "+g.the(att)+".", att)
	}
}

func (g *Game) dzHitText(att *Actor) string {
	return fmt.Sprintf("Hurt by %s near %s.", g.the(att), g.Cur.Name)
}

// dzScanConsume watches the pack for things eaten or drunk (hunger and thirst fall, a stack shrinks).
func (g *Game) dzScanConsume(st *dzState) {
	cur := map[int]int{}
	items := map[int]*Item{}
	for _, it := range g.P.Inv {
		switch it.Kind {
		case "food", "corpse", "ingredient", "dish":
			cur[it.ID] = maxi(it.Count, 1)
			items[it.ID] = it
		case "water":
			cur[it.ID] = it.Charges
			items[it.ID] = it
		}
	}
	ate := st.pH-g.Hunger >= 50
	drank := st.pT-g.Thirst >= 100
	usedWater := false
	if ate || drank {
		for id, n := range st.prevN {
			if cur[id] >= n {
				continue
			}
			it := st.prevIt[id]
			if it == nil {
				continue
			}
			if it.Kind == "water" {
				if drank {
					usedWater = true
					g.dzBadWater(lqKey(it))
				}
			} else if ate {
				g.dzBadFood(it)
			}
		}
		if drank && !usedWater {
			if k, _, _, ok := g.lqPoolNear(); ok {
				g.dzBadWater(k)
			}
		}
	}
	st.prevIt, st.prevN, st.pH, st.pT = items, cur, g.Hunger, g.Thirst
}

func (g *Game) dzBadWater(k string) {
	switch k {
	case "brackish":
		if g.R.Chance(0.25) {
			g.dzInfect("gut_parasites", "Drank brackish water.", nil)
		}
	case "tainted":
		if g.R.Chance(0.2) {
			g.dzInfect("rot_blight", "Drank tainted water.", nil)
		} else if g.R.Chance(0.2) {
			g.dzInfect("gut_parasites", "Drank tainted water.", nil)
		}
	case "blood":
		if g.R.Chance(0.2) {
			g.dzInfect("gut_parasites", "Drank blood.", nil)
		}
	}
}

func (g *Game) dzBadFood(it *Item) {
	st := g.dz()
	n := strings.ToLower(it.Name)
	rotten := strings.Contains(n, "rotten") || strings.Contains(n, "spoiled") || it.Extra != nil && it.Extra["cl:rot"] > 0
	raw := it.Kind == "corpse" || it.Kind == "ingredient" && (strings.Contains(n, "meat") || strings.Contains(n, "fish") || strings.Contains(n, "liver") || strings.Contains(n, "organ") || strings.Contains(n, "brain"))
	switch {
	case rotten:
		if g.R.Chance(0.25) {
			g.dzInfect("rot_blight", "Ate "+it.Name+".", nil)
		} else if g.R.Chance(0.15) {
			g.dzInfect("gut_parasites", "Ate "+it.Name+".", nil)
		}
	case raw:
		day := g.Day()
		keep := st.Meals[:0]
		for _, d := range st.Meals {
			if day-d <= 20 {
				keep = append(keep, d)
			}
		}
		st.Meals = append(keep, day)
		if g.R.Chance(0.3) {
			g.dzInfect("gut_parasites", "Ate raw meat.", nil)
		} else if it.Kind == "corpse" && g.R.Chance(0.04+0.04*float64(len(st.Meals)-1)) {
			g.dzInfect("brain_wasting", "Ate the flesh of a beast that was already sick.", nil)
		}
	}
}

// ---------------------------------------------------------------- the minute and the hour

func dzTurn(g *Game) {
	if !g.dzReady() {
		return
	}
	st := g.dz()
	if g.Turn%2 == 0 {
		g.dzScanConsume(st)
	}
	if !st.Hooked && st.LastHP > g.P.HP && g.P.HP > 0 {
		for _, a := range g.Cur.Actors {
			if a.HP > 0 && a != g.P && a.AI != "follower" && g.Hostile(a) && abs(a.X-g.P.X) <= 1 && abs(a.Y-g.P.Y) <= 1 {
				g.dzHitCheck(a, st.LastHP-g.P.HP)
				break
			}
		}
	}
	st.LastHP = g.P.HP
	g.dzCounters(st)
	night := g.dzNight()
	for _, c := range st.Conds {
		g.dzApplyStatuses(g.P, c, night)
	}
	if len(st.Comp) > 0 {
		g.dzCompStatuses(st, night)
	}
	if g.Turn%60 == 0 {
		g.dzHour(st)
	}
}

func (g *Game) dzCounters(st *dzState) {
	zn, p := g.Cur, g.P
	out := outdoors(zn)
	night := g.dzNight()
	dec := func(v *int, by int) {
		if *v > 0 {
			*v = maxi(0, *v-by)
		}
	}
	if zn.Z > 0 || zn.Dark || !out {
		st.Dark++
		if zn.Z > 0 {
			st.Sun = 0
		}
	} else if g.daylight() {
		dec(&st.Dark, 3)
		st.Sun++
	}
	low := strings.ToLower(zn.Name)
	if out && zn.Biome == 13 || strings.Contains(low, "mine") && zn.Z > 0 {
		st.Dust++
	} else if g.Turn%4 == 0 {
		dec(&st.Dust, 1)
	}
	h := g.Hour()
	if out && (zn.Biome == 12 || zn.Biome == 7 || zn.Biome == 8) && (h >= 18 || h < 6) {
		st.Wet++
	} else if g.Turn%4 == 0 {
		dec(&st.Wet, 1)
	}
	if out && (zn.Biome == 10 || zn.Biome == 11 || zn.Biome == 9 && g.Season() == 3) && g.Season() == 3 {
		st.Cold++
	} else if g.Turn%4 == 0 {
		dec(&st.Cold, 1)
	}
	if zn.Kind == "sea" || strings.HasPrefix(zn.Key, "sh:") {
		st.Sea++
		st.Land = 0
	} else {
		st.Land++
		if g.Turn%4 == 0 {
			dec(&st.Sea, 1)
		}
	}
	if p.St("poisoned") > 0 {
		st.Poison++
	} else if g.Turn%8 == 0 {
		dec(&st.Poison, 1)
	}
	if g.dzTaintCarried() {
		st.Iron++
		st.NoIron = 0
	} else {
		st.NoIron++
		if g.Turn%8 == 0 {
			dec(&st.Iron, 1)
		}
	}
	if p.St("gcrad2") > 0 || p.St("gcrad3") > 0 {
		st.Rad++
	} else if g.Turn%4 == 0 {
		dec(&st.Rad, 1)
	}
	if p.St("edchill") > 0 {
		st.Chill++
	} else if g.Turn%4 == 0 {
		dec(&st.Chill, 1)
	}
	if p.St("edfey") > 0 {
		st.Fey++
	} else if g.Turn%4 == 0 {
		dec(&st.Fey, 1)
	}
	if p.St("hzsick") > 0 || p.St("hzpox") > 0 || p.St("hzfog") > 0 {
		st.Hz++
	}
	_ = night
}

func (g *Game) dzTaintCarried() bool {
	for _, it := range g.P.Inv {
		if it.Extra != nil && it.Extra["dztaint"] > 0 {
			return true
		}
		n := strings.ToLower(it.Name)
		if (it.Kind == "weapon" || it.Kind == "armor") && (strings.Contains(n, "tainted") || strings.Contains(n, "blighted")) {
			return true
		}
	}
	return false
}

func (g *Game) dzCursedItem() *Item {
	for _, it := range g.P.Inv {
		if !g.ceCanUnequip(it) {
			return it
		}
	}
	for _, bp := range g.P.Body {
		if bp.Item != nil && !g.ceCanUnequip(bp.Item) {
			return bp.Item
		}
	}
	return nil
}

func (g *Game) dzFindItem(id int) *Item {
	for _, it := range g.P.Inv {
		if it.ID == id {
			return it
		}
	}
	for _, bp := range g.P.Body {
		if bp.Item != nil && bp.Item.ID == id {
			return bp.Item
		}
	}
	return nil
}

func (g *Game) dzExposure(st *dzState) {
	roll := func(p float64) bool { return g.R.Chance(p) }
	if st.Dark >= 4320 && roll(0.25) {
		g.dzInfect("night_blind", "Too long without the sun.", nil)
	}
	if st.Dust >= 480 && roll(0.08) {
		g.dzInfect("wasting_lung", "Breathing dust and ash for too long.", nil)
	}
	if st.Wet >= 240 && roll(0.1) {
		g.dzInfect("marsh_fever", "Bitten and chilled in the damp of the night.", nil)
	}
	if st.Cold >= 300 && roll(0.08) {
		g.dzInfect("frost_rot", "Too long in the winter cold.", nil)
	}
	if st.Sea >= 480 && roll(0.1) {
		g.dzInfect("sea_madness", "Too long on the open water.", nil)
	}
	if st.Poison >= 200 && roll(0.1) {
		g.dzInfect("the_shakes", "Poisoned too often.", nil)
	}
	if st.Iron >= 1440 && roll(0.1) {
		g.dzInfect("iron_sickness", "Carrying tainted metal.", nil)
	}
	if st.Rad >= 90 && g.Genre != "" && roll(0.1) {
		g.dzInfect("rad_sores", "Too much radiation.", nil)
	}
	if st.Chill >= 20 && roll(0.1) {
		g.dzInfect("ghost_sickness", "The dead's chill of cursed ground got into you.", nil)
	}
	if st.Fey >= 20 && roll(0.1) {
		g.dzInfect("witch_hex", "The fey ground took hold of you.", nil)
	}
	if g.Cur.Kind == "crypt" && g.dzWard("mummy_curse") == nil && roll(0.04) {
		g.dzSetSrcZone(g.dzInfectRet("mummy_curse", "Breathing a crypt's dust."))
	}
	if g.P.St("gravesick") > 0 && g.dzFind("mummy_curse") == nil && g.dzFind("ghost_sickness") == nil && roll(0.5) {
		id := "ghost_sickness"
		if HashStr(fmt.Sprint(g.Seed, ":dz:gs:", g.Day()))%2 == 0 {
			id = "mummy_curse"
		}
		g.dzSetSrcZone(g.dzInfectRet(id, "Robbed a grave."))
	}
	if it := g.dzCursedItem(); it != nil && g.dzFind("item_binding") == nil && roll(0.08) {
		if c := g.dzInfectRet("item_binding", "Carrying "+it.Name+", which is cursed."); c != nil {
			c.Item = it.ID
		}
	}
	// hz sequel
	if st.Hz > 0 && g.P.St("hzsick") == 0 && g.P.St("hzpox") == 0 && g.P.St("hzfog") == 0 {
		if st.Hz >= 1440 && roll(0.2) {
			g.dzInfect("wasting_lung", "A lung that never recovered from a plague.", nil)
		} else if st.Hz >= 720 && roll(0.1) {
			g.dzInfect("the_shakes", "Hands that never quite recovered from a plague.", nil)
		}
		st.Hz = 0
	}
	// local epidemic
	if g.Cur.Kind == "town" && g.Cur.Burg > 0 {
		if id := g.dzl().Towns[g.Cur.Burg]; id != "" {
			for _, a := range g.Cur.Actors {
				if a.Has("dzsick") && a.HP > 0 && abs(a.X-g.P.X) <= 2 && abs(a.Y-g.P.Y) <= 2 && roll(0.06) {
					g.dzInfect(id, "Caught it from the sick of "+g.Cur.Name+".", nil)
					break
				}
			}
		}
	}
}

func (g *Game) dzInfectRet(id, src string) *dzCond {
	if g.dzInfect(id, src, nil) {
		return g.dzFind(g.dzResolve(id))
	}
	return nil
}

func (g *Game) dzSetSrcZone(c *dzCond) {
	if c != nil {
		c.SrcKey, c.SrcName = g.Cur.Key, g.Cur.Name
	}
}

func (g *Game) dzHour(st *dzState) {
	g.dzExposure(st)
	night := g.dzNight()
	for _, c := range append([]*dzCond{}, st.Conds...) {
		d := dzByID[c.ID]
		if d == nil {
			g.dzRemove(c, "unknown", 0)
			continue
		}
		g.dzTick(c, d, true, night)
	}
	for id, c := range st.Comp {
		d := dzByID[c.ID]
		if d == nil || g.Turn >= c.Until {
			delete(st.Comp, id)
			continue
		}
		g.dzTick(c, d, false, night)
	}
	g.dzSpread(st)
	if g.Turn%DayLen == 0 {
		g.dzDaily(st)
	}
}

// dzTimeOK: the plain "time and rest" cures.
func (g *Game) dzTimeOK(st *dzState, c *dzCond) bool {
	switch c.ID {
	case "night_blind":
		return st.Sun >= 2000
	case "iron_sickness":
		return c.Stage <= 1 && st.NoIron >= 3000
	case "sea_madness":
		return c.Stage <= 1 && st.Land >= 3000
	}
	return false
}

func (g *Game) dzTick(c *dzCond, d *dzDef, player bool, night bool) {
	st := g.dz()
	last := len(d.Stages) - 1
	// wards keep it where it is
	if player {
		if g.dzWard(c.ID) != nil && g.Turn+120 > c.Arrest {
			c.Arrest = g.Turn + 120
		}
		if c.ID == "item_binding" {
			it := g.dzFindItem(c.Item)
			if it == nil || g.ceCanUnequip(it) {
				g.Msg("{{G|The binding loosens: the cursed thing is gone from you.}}")
				g.dzRemove(c, "the item left you", 20)
				return
			}
		}
		for _, cu := range d.Cures {
			if cu.Kind == "time" && g.dzTimeOK(st, c) {
				g.Msg(fmt.Sprintf("{{G|%s fades with time.}}", g.dzDefName(d)))
				g.dzRemove(c, "time", 20)
				return
			}
		}
	}
	arrested := g.Turn < c.Arrest
	if arrested {
		c.Next += 60
	}
	for !arrested && g.Turn >= c.Next && (c.Stage < last || d.Loop) {
		if c.Stage < last {
			c.Stage++
		} else {
			c.Stage = 0
			c.Cycles++
			if d.Natural > 0 && g.R.Chance(float64(d.Natural)/100*float64(c.Cycles)) {
				if player {
					g.Msg(fmt.Sprintf("{{G|The last bout of %s was the last.}}", strings.ToLower(g.dzDefName(d))))
					g.dzRemove(c, "burnt out", 20)
				}
				return
			}
		}
		s := d.Stages[c.Stage]
		if s.Days > 0 {
			c.Next = g.Turn + s.Days*DayLen
		} else {
			c.Next = g.Turn + 1<<30
		}
		if player {
			g.dzSymptoms(c, d, "{{R|It is getting worse.}} ")
			g.dzPublic(c, d)
		}
	}
	if !player || arrested {
		return
	}
	s := d.Stages[clampi(c.Stage, 0, last)]
	if s.Drain > 0 {
		g.hurt(g.P, s.Drain, nil, strings.ToLower(g.dzDefName(d)))
	}
	if s.Hunger > 0 {
		g.Hunger += s.Hunger
	}
	if s.Thirst > 0 {
		g.Thirst += s.Thirst
	}
	g.dzSpecial(c, d, night)
}

func (g *Game) dzSpecial(c *dzCond, d *dzDef, night bool) {
	p := g.P
	switch d.ID {
	case "lycanthropy":
		if night && c.Stage >= 1 && g.R.Chance(0.08*float64(c.Stage)) {
			p.AddSt("frenzied", 40)
			g.Hunger += 150
			g.Msg("{{R|The beast surges. Your vision reddens and you want blood and meat.}}")
		}
	case "vampiric_taint":
		if c.Stage >= 2 && outdoors(g.Cur) && g.daylight() && !g.underRoof(p.X, p.Y) {
			g.hurt(p, 1, nil, "sunlight")
			g.Msg("{{O|The sun scalds your skin.}}")
		}
	case "god_mark":
		if c.Stage >= 1 && g.R.Chance(0.03) {
			p.AddSt("inspired", 60)
			g.Msg("{{Y|The mark flares and you feel certain.}}")
		}
	case "sea_madness":
		if c.Stage >= 1 && g.Cur.Kind == "sea" {
			p.AddSt("disoriented", 8)
		}
	case "witch_hex":
		if c.Stage >= 1 && g.R.Chance(0.04) {
			p.AddSt("disoriented", 5)
			g.Msg("{{g|Something small goes wrong: a strap snaps, a foot slips.}}")
		}
	case "ghost_sickness":
		if c.Stage >= 1 && g.R.Chance(0.04) {
			p.AddSt("shaken", 10)
			g.Msg("{{b|Someone breathes beside your ear.}}")
		}
	}
}

func (g *Game) dzDaily(st *dzState) {
	day := g.Day()
	for _, c := range append([]*dzCond{}, st.Conds...) {
		d := dzByID[c.ID]
		if d == nil || d.Natural <= 0 || d.Loop || c.Stage != 0 || g.Turn < c.Arrest {
			continue
		}
		if g.R.Chance(float64(d.Natural) / 100) {
			g.Msg(fmt.Sprintf("{{G|%s fades by itself.}}", g.dzDefName(d)))
			g.dzRemove(c, "faded", 15)
		}
	}
	_ = day
}

// dzPublic: afflictions that people can see go to the rumour mill.
func (g *Game) dzPublic(c *dzCond, d *dzDef) {
	if c.Told || dzRumourSink == nil || c.Stage < 1 {
		return
	}
	pub := d.Spread >= 1 || d.ID == "god_mark" || d.ID == "lycanthropy" && c.Stage >= 2 || d.ID == "vampiric_taint" && c.Stage >= 2 || d.ID == "rot_blight"
	if !pub {
		return
	}
	c.Told = true
	dzRumourSink(g, "affliction", g.P.Name, fmt.Sprintf("%s is said to be marked by %s.", g.P.Name, strings.ToLower(g.dzDefName(d))))
}

// dzSpread: contagion to companions and townsfolk, and companions' own course.
func (g *Game) dzSpread(st *dzState) {
	for _, c := range st.Conds {
		d := dzByID[c.ID]
		if d == nil || d.Spread < 1 || c.Stage < 1 || g.Turn < c.Arrest {
			continue
		}
		for _, a := range g.localCompanions() {
			if st.Comp[a.ID] != nil || a.HP <= 0 || abs(a.X-g.P.X) > 3 || abs(a.Y-g.P.Y) > 3 {
				continue
			}
			if g.R.Chance(0.08) {
				st.Comp[a.ID] = &dzCond{ID: c.ID, Stage: clampi(c.Stage-1, 0, len(d.Stages)-1), Got: g.Day(), Until: g.Turn + 10*DayLen, Next: g.Turn + 1<<30}
				g.Msg(fmt.Sprintf("{{O|%s has caught it from you.}}", g.The(a)))
			}
		}
		if d.Spread >= 2 && g.Cur.Kind == "town" && g.Cur.Burg > 0 {
			for _, a := range g.Cur.Actors {
				if a.Tpl == "npc" && a.AI == "town" && !a.Has("dzsick") && abs(a.X-g.P.X) <= 2 && abs(a.Y-g.P.Y) <= 2 && g.R.Chance(0.05) {
					a.Set("dzsick")
					a.Desc += " They look unwell."
					g.dzl().Towns[g.Cur.Burg] = c.ID
					n := 0
					for _, o := range g.Cur.Actors {
						if o.Has("dzsick") {
							n++
						}
					}
					if n == 3 && dzRumourSink != nil {
						dzRumourSink(g, "affliction", g.Cur.Name, fmt.Sprintf("Sickness is spreading in %s: %s.", g.Cur.Name, strings.ToLower(g.dzDefName(d))))
					}
					break
				}
			}
		}
	}
}

func (g *Game) dzCompStatuses(st *dzState, night bool) {
	for _, a := range g.localCompanions() {
		if c := st.Comp[a.ID]; c != nil {
			g.dzApplyStatuses(a, c, night)
		}
	}
}

// ---------------------------------------------------------------- hooks: kill, enter, zone

func dzKill(g *Game, d, src *Actor) {
	if !g.dzReady() || d == nil {
		return
	}
	st := g.dz()
	if len(st.Conds) == 0 {
		return
	}
	if src != g.P && !(src != nil && src.AI == "follower") {
		return
	}
	for _, c := range append([]*dzCond{}, st.Conds...) {
		if c.SrcActor != d.ID || c.SrcActor == 0 {
			continue
		}
		def := dzByID[c.ID]
		for i, cu := range def.Cures {
			if cu.Kind == "kill" {
				st.Hints[c.ID] |= 1 << i
				c.Diag = true
				g.Msg(g.dzApply(c, cu, "You killed the one who cursed you. "))
				break
			}
		}
	}
}

func dzEnter(g *Game, zn *Zone) {
	if !g.dzReady() || zn == nil {
		return
	}
	st := g.dz()
	// shrines: a named wonder of the world
	if zn.Kind == "wonder" {
		for _, c := range append([]*dzCond{}, st.Conds...) {
			d := dzByID[c.ID]
			for i, cu := range d.Cures {
				if cu.Kind != "shrine" {
					continue
				}
				if w := g.dzSite(c.ID); w != nil && w.Name == zn.Name && c.Planet == g.W.uniIdx {
					key := fmt.Sprintf("%s:%d", c.ID, c.Got)
					if g.dzl().Shrines[key] > 0 {
						continue
					}
					g.dzl().Shrines[key] = g.Day() + 1
					st.Hints[c.ID] |= 1 << i
					c.Diag = true
					g.Msg(g.dzApply(c, cu, fmt.Sprintf("You rest and pray at %s. ", zn.Name)))
				}
			}
		}
		// the god lays a hand on a few who come (once per wonder)
		k := "mark:" + zn.Key
		if g.dzl().Said[k] == 0 {
			g.dzl().Said[k] = 1
			if HashStr(fmt.Sprint(g.Seed, ":dz:mark:", zn.Key))%100 < 12 && g.dzFind("god_mark") == nil {
				if g.dzInfect("god_mark", "Marked at "+zn.Name+".", nil) {
					g.Msg("{{Y|A warmth settles on your palm. Something has noticed you.}}")
				}
			}
		}
	}
	g.dzTownSick(zn)
}

// dzSite is the named wonder where a shrine cure works: a pure function of the seed and the affliction.
func (g *Game) dzSite(id string) *WonderData {
	var ok []*WonderData
	for i := range g.W.Wonders {
		w := &g.W.Wonders[i]
		if !w.Destroyed && (w.Layer == "" || w.Layer == "surface") && w.P >= 0 && w.P < g.W.W*g.W.H {
			ok = append(ok, w)
		}
	}
	if len(ok) == 0 {
		return nil
	}
	sort.Slice(ok, func(i, j int) bool { return ok[i].ID < ok[j].ID })
	return ok[int(HashStr(g.Seed+":dz:site:"+id)%uint64(len(ok)))]
}

func dzZone(g *Game, zn *Zone, _ *Rng) {
	if g.W == nil || zn == nil {
		return
	}
	r := NewRng(g.Seed + ":dz:" + zn.Key)
	// tainted iron in ruins, crypts and the deeps
	if (zn.Z > 0 || zn.Kind == "ruin" || zn.Kind == "crypt") && r.Chance(0.05) {
		if x, y, ok := zn.FreeSpot(r, nil); ok {
			zn.Items = append(zn.Items, &Item{ID: g.newID(), Kind: "scrap", Name: "blackened iron ingot", Glyph: '=', F: CDarkGray, Value: 6, Weight: 4, Count: 1, X: x, Y: y,
				Lore: "Iron with a wrong, sweet smell. Hands that carry it for long days feel it.", Extra: map[string]int{"dztaint": 1}})
		}
	}
	// the odd town with a local sickness
	if zn.Kind == "town" && zn.Burg > 0 && zn.Z == 0 {
		year := g.Day() / (4 * SeasonDays)
		if HashStr(fmt.Sprint(g.Seed, ":dz:epi:", zn.Burg, ":", year))%100 < 4 {
			ids := []string{"rot_blight", "marsh_fever", "gut_parasites", "fungal_spores", "wasting_lung"}
			id := ids[int(HashStr(fmt.Sprint(g.Seed, ":dz:epid:", zn.Burg, ":", year))%uint64(len(ids)))]
			id = g.dzResolve(id)
			if d := dzByID[id]; d != nil && g.dzAllowed(d) {
				n := 0
				for _, a := range zn.Actors {
					if a.Tpl == "npc" && a.AI == "town" && n < 2 && a.Role == "villager" {
						a.Set("dzsick")
						a.Desc += " They look unwell."
						n++
					}
				}
				if n > 0 {
					g.dzl().Towns[zn.Burg] = id
				}
			}
		}
	}
}

func (g *Game) dzTownSick(zn *Zone) {}

// ---------------------------------------------------------------- cures

func (g *Game) dzReveal(c *dzCond, i int) { g.dz().Hints[c.ID] |= 1 << i }

func (g *Game) dzRevealed(c *dzCond, i int) bool { return g.dz().Hints[c.ID]&(1<<i) != 0 }

// dzApply runs a cure on the player's condition and returns the text.
func (g *Game) dzApply(c *dzCond, cu dzCure, lead string) string {
	d := dzByID[c.ID]
	pow := cu.Pow
	if cu.Max > 0 && c.Stage+1 > cu.Max {
		pow--
	}
	name := g.dzDefName(d)
	switch {
	case pow >= 2:
		if c.ID == "item_binding" {
			if it := g.dzFindItem(c.Item); it != nil {
				g.ceUncurse(it)
			}
		}
		g.dzRemove(c, "cured", 30)
		return fmt.Sprintf("{{G|%s%s is gone from you.}}", lead, name)
	case pow == 1:
		if c.Stage == 0 {
			g.dzRemove(c, "cured", 20)
			return fmt.Sprintf("{{G|%s%s lets go of you.}}", lead, name)
		}
		c.Stage--
		s := d.Stages[c.Stage]
		c.Arrest = g.Turn + 3*DayLen
		if s.Days > 0 {
			c.Next = g.Turn + s.Days*DayLen
		}
		return fmt.Sprintf("{{G|%s%s eases back to %s.}}", lead, name, strings.ToLower(s.Name))
	}
	c.Arrest = g.Turn + 7*DayLen
	return fmt.Sprintf("{{C|%s%s is held where it is for about a week.}}", lead, name)
}

type dzRem struct {
	It   *Item
	Cure dzCure
	Idx  int
	Raw  bool
	Why  string
}

var dzTonicProp = map[string]string{"heal": "healing", "salve": "healing", "antidote": "antidote"}

func (g *Game) dzRemedies(c *dzCond) []dzRem {
	d := dzByID[c.ID]
	var out []dzRem
	for _, it := range g.P.Inv {
		props := map[string]bool{}
		raw := false
		switch {
		case it.Kind == "tonic" && it.Effect == "brewed":
			for p := range alItemEffects(it) {
				props[p] = true
			}
		case it.Kind == "tonic":
			if p, ok := dzTonicProp[it.Effect]; ok {
				props[p] = true
			}
		case it.Kind == "ingredient":
			if p, ok := alTagProp[it.Effect]; ok {
				props[p] = true
				raw = true
			}
		}
		if len(props) == 0 {
			continue
		}
		for i, cu := range d.Cures {
			if cu.Kind == "herb" && props[cu.Prop] {
				out = append(out, dzRem{It: it, Cure: cu, Idx: i, Raw: raw, Why: alPropByID[cu.Prop].Name})
				break
			}
		}
	}
	return out
}

func (g *Game) dzUseRemedy(c *dzCond, r dzRem) {
	cu := r.Cure
	if r.Raw && cu.Pow > 0 {
		cu.Pow--
	}
	g.dzReveal(c, r.Idx)
	c.Diag = true
	g.consume(r.It)
	g.Msg(g.dzApply(c, cu, fmt.Sprintf("You take the %s. ", strings.ToLower(Plain(r.It.Name)))))
}

// grave offering
func (g *Game) dzGraveCond() *dzCond {
	if !g.dzReady() {
		return nil
	}
	for _, c := range g.dz().Conds {
		d := dzByID[c.ID]
		if c.SrcKey == "" || c.SrcKey != g.Cur.Key {
			continue
		}
		for _, cu := range d.Cures {
			if cu.Kind == "grave" && g.dzOffering() != nil {
				return c
			}
		}
	}
	return nil
}

func (g *Game) dzOffering() *Item {
	var best *Item
	for _, it := range g.P.Inv {
		if (it.Kind == "trophy" || it.Kind == "artifact" || it.Kind == "book") && it.Value >= 20 && it.Quest == 0 {
			if best == nil || it.Value < best.Value {
				best = it
			}
		}
	}
	return best
}

func (g *Game) dzLayOffering() {
	c := g.dzGraveCond()
	if c == nil {
		return
	}
	it := g.dzOffering()
	d := dzByID[c.ID]
	for i, cu := range d.Cures {
		if cu.Kind == "grave" {
			g.dzReveal(c, i)
			c.Diag = true
			g.consume(it)
			g.Msg(g.dzApply(c, cu, fmt.Sprintf("You lay %s at the grave, and ask pardon. ", strings.ToLower(Plain(it.Name)))))
			g.spend(600)
			return
		}
	}
}

// ---------------------------------------------------------------- talk: diagnosis and treatment

func (g *Game) dzMenu(text string, opts []Option, a *Actor) {
	g.Popup = Wrap(text, 100)
	g.Opts = append(opts, Option{"Back.", func() { g.talkMenu("\"Anything else?\"") }})
	g.Sel = 0
}

func dzRoleOf(a *Actor) (phys, pri, herb, mas bool) {
	phys = ijIsPhysician(a) || a.Role == "physician"
	pri = a.Role == "priest"
	herb = dzHerbalistRoles[a.Role]
	mas = a.Teach != "" || a.Role == "seer" || a.Role == "scholar"
	return
}

func dzTalk(g *Game, a *Actor) []Option {
	if !g.dzReady() || a == nil || a.HP <= 0 || a.Angry || a.AI == "hostile" || a.AI == "follower" || a.Tpl != "npc" {
		return nil
	}
	phys, pri, herb, mas := dzRoleOf(a)
	if !(phys || pri || herb || mas) {
		return nil
	}
	if len(g.dz().Conds) == 0 {
		return nil
	}
	return []Option{{"{{G|Ask about my condition.}}", func() { g.dzAsk(a) }}}
}

func (g *Game) dzAsk(a *Actor) {
	st := g.dz()
	var opts []Option
	for _, c := range st.Conds {
		c := c
		d := dzByID[c.ID]
		if d == nil {
			continue
		}
		stage := d.Stages[clampi(c.Stage, 0, len(d.Stages)-1)]
		opts = append(opts, Option{fmt.Sprintf("{{W|%s}} {{K|(%s)}}", g.dzCondName(c), strings.ToLower(stage.Name)), func() { g.dzCondMenu(a, c) }})
	}
	g.dzMenu("\"Tell me where it hurts, and since when.\"", opts, a)
}

// dzFacets: which kinds of cure each helper knows about.
func dzKnows(a *Actor) map[string]bool {
	phys, pri, herb, mas := dzRoleOf(a)
	k := map[string]bool{}
	if phys {
		k["herb"], k["doctor"], k["time"], k["ward"] = true, true, true, true
	}
	if pri {
		k["priest"], k["shrine"], k["grave"], k["kill"] = true, true, true, true
	}
	if herb {
		k["herb"], k["time"] = true, true
	}
	if mas {
		k["ward"], k["shrine"], k["kill"], k["grave"] = true, true, true, true
	}
	return k
}

func (g *Game) dzCondMenu(a *Actor, c *dzCond) {
	d := dzByID[c.ID]
	phys, pri, herb, mas := dzRoleOf(a)
	var opts []Option
	opts = append(opts, Option{"{{c|Be examined, and say what you can.}}", func() { g.dzDiagMenu(a, c) }})
	if phys {
		for _, cu := range d.Cures {
			if cu.Kind == "doctor" {
				fee := d.Fee + c.Stage*d.Fee/2 + 5
				opts = append(opts, Option{fmt.Sprintf("{{Y|Ask for treatment.}} {{K|(%s)}}", g.cuFormat(fee)), func() {
					if !g.cuSpend(fee) {
						g.dzMenu("\"That takes coin, and you have not got enough.\"", nil, a)
						return
					}
					c.Diag = true
					for i, o := range d.Cures {
						if o.Kind == "doctor" {
							g.dzReveal(c, i)
						}
					}
					g.dzMenu(g.dzApply(c, cu, "\"Hold still.\" "), nil, a)
				}})
				break
			}
		}
		if len(g.dz().Comp) > 0 {
			opts = append(opts, Option{"{{Y|Ask them to see to my companions too.}}", func() {
				n := len(g.dz().Comp)
				for id := range g.dz().Comp {
					delete(g.dz().Comp, id)
				}
				g.dzMenu(fmt.Sprintf("They examine your companions and treat %d of them.", n), nil, a)
			}})
		}
	}
	if pri {
		for i, cu := range d.Cures {
			if cu.Kind != "priest" {
				continue
			}
			i, cu := i, cu
			rank := g.dzFaithRank(a.Religion)
			label := fmt.Sprintf("{{Y|Ask for the rite.}} {{K|(needs standing %d with this faith; you have %d)}}", cu.Rank, rank)
			opts = append(opts, Option{label, func() {
				if rank < cu.Rank || rank == 0 {
					g.dz().Hints[c.ID] |= 1 << i
					g.dzMenu("\"I cannot do that rite for you. It is for someone more trusted by our faith, or by someone of higher rank than I am. Earn the faith's trust and come back.\"", nil, a)
					return
				}
				tithe := 10 + 8*cu.Rank
				if !g.cuSpend(tithe) {
					g.dzMenu("\"A tithe is customary. You have not got it.\"", nil, a)
					return
				}
				g.dzReveal(c, i)
				c.Diag = true
				g.dzMenu(g.dzApply(c, cu, "They lay their hands on you and say the rite. "), nil, a)
			}})
		}
		if c.ID == "god_mark" {
			opts = append(opts, Option{"{{Y|Ask them to pray over the mark.}}", func() {
				c.Arrest = g.Turn + 7*DayLen
				g.dz().Prayed = g.Day()
				g.dzMenu("They pray over your palm. The burning eases for about a week.", nil, a)
			}})
		}
	}
	if herb {
		for i, cu := range d.Cures {
			if cu.Kind != "herb" || !(g.dzRevealed(c, i) || c.Diag) {
				continue
			}
			i, cu := i, cu
			price := 18 + 6*c.Stage + 4*cu.Pow
			opts = append(opts, Option{fmt.Sprintf("{{Y|Buy a %s remedy.}} {{K|(%s)}}", alPropByID[cu.Prop].Name, g.cuFormat(price)), func() {
				if !g.cuSpend(price) {
					g.dzMenu("\"You cannot afford it.\"", nil, a)
					return
				}
				g.dzReveal(c, i)
				it := &Item{ID: g.newID(), Kind: "tonic", Effect: "brewed", Name: alPropByID[cu.Prop].Name + " remedy", Glyph: '!', F: CGreen, Count: 1, Value: price,
					Extra: map[string]int{"alp:" + cu.Prop: 2}}
				g.give(it)
				g.dzMenu("\"Take it as soon as you can, and eat well after. You can also take it on the Afflictions screen.\"", nil, a)
			}})
		}
	}
	if mas {
		for i, cu := range d.Cures {
			if cu.Kind != "ward" || !(g.dzRevealed(c, i) || c.Diag) {
				continue
			}
			i := i
			price := 90 + 40*c.Stage
			opts = append(opts, Option{fmt.Sprintf("{{Y|Ask for a ward-amulet.}} {{K|(%s)}}", g.cuFormat(price)), func() {
				if g.dzWard(c.ID) != nil {
					g.dzMenu("\"You already carry one.\"", nil, a)
					return
				}
				if !g.cuSpend(price) {
					g.dzMenu("\"A ward takes days of work and good materials. I need the coin.\"", nil, a)
					return
				}
				g.dzReveal(c, i)
				it := &Item{ID: g.newID(), Kind: "trophy", Effect: "dzamulet", Name: "ward-amulet against " + strings.ToLower(g.dzDefName(d)), Glyph: '♦', F: CCyan, Value: 60, Weight: 1, Count: 1,
					Extra: map[string]int{"dza:" + c.ID: 1}, Lore: "A master's amulet. While carried it holds the affliction where it is and turns new ones aside. It does not cure."}
				g.give(it)
				g.dzMenu("\"Keep it on you. It holds it back, no more. For a cure you must look further.\"", nil, a)
			}})
		}
	}
	_ = mas
	g.dzMenu("\"What is it you want of me?\"", opts, a)
}

func (g *Game) dzFaithRank(faith int) int {
	r := 1
	if v, ok := g.Rep[fmt.Sprintf("faith:%d", faith)]; ok {
		switch {
		case v < 0:
			r = 0
		case v < 40:
			r = 1
		case v < 120:
			r = 2
		default:
			r = 3
		}
	}
	for _, o := range g.Orders() {
		if o.Faith == faith && faith > 0 && (o.Kind == "holy" || o.Kind == "monastic") {
			if m := g.OrderMembership(o.ID); m != nil && m.Rank > 0 {
				r = mini(3, r+1)
				break
			}
		}
	}
	return r
}

func (g *Game) dzDiagMenu(a *Actor, c *dzCond) {
	d := dzByID[c.ID]
	phys, pri, _, _ := dzRoleOf(a)
	dc := 9 + d.Stages[0].Days/6 - c.Stage
	if d.Boon != "" {
		dc++
	}
	if phys && (d.Kind == "disease" || d.Kind == "parasite" || d.Kind == "poison") || pri && (d.Kind == "curse" || d.Kind == "bite") {
		dc--
	}
	dc = clampi(dc, 6, 14)
	fee := 0
	if !pri {
		fee = 4 + g.P.Level
	}
	approaches := []struct{ Text, Kind string }{
		{"Describe the symptoms plainly.", "perception"},
		{"Show them the signs and bear the poking.", "endurance"},
		{"Offer a theory from what you have read or heard.", "lore"},
	}
	var opts []Option
	for _, ap := range approaches {
		ap := ap
		opts = append(opts, Option{fmt.Sprintf("%s %s", g.checkLabel(ap.Kind, dc), ap.Text), func() {
			if fee > 0 && !g.cuSpend(fee) {
				g.dzMenu("\"A consultation has a fee.\"", nil, a)
				return
			}
			margin := g.R.Dice(2, 6) + g.checkMod(ap.Kind) - dc
			g.dzDiagnose(a, c, margin)
		}})
	}
	g.dzMenu("\"Describe it to me. What did you do before it started, and what do you feel?\"", opts, a)
}

func (g *Game) dzDiagnose(a *Actor, c *dzCond, margin int) {
	d := dzByID[c.ID]
	knows := dzKnows(a)
	var idx []int
	for i, cu := range d.Cures {
		if knows[cu.Kind] && !g.dzRevealed(c, i) {
			idx = append(idx, i)
		}
	}
	n := 0
	text := ""
	switch {
	case margin >= 0:
		c.Diag = true
		n = 1
		if margin >= 3 {
			n = len(idx)
		} else if margin >= 1 {
			n = mini(2, len(idx))
		}
		text = fmt.Sprintf("They study you for a long moment. \"It is %s. %s\"", strings.ToLower(g.dzDefName(d)), d.Lore)
	case margin >= -2:
		n = mini(1, len(idx))
		text = "\"I cannot name it for certain. But I have seen something like it.\""
	default:
		text = "\"I do not know what it is. I am sorry. Come back when it has shown itself more, or ask someone else.\""
	}
	var said []string
	for k := 0; k < n && k < len(idx); k++ {
		g.dzReveal(c, idx[k])
		said = append(said, d.Cures[idx[k]].Hint)
	}
	if len(said) > 0 {
		text += "\n\n" + strings.Join(said, " ")
	}
	if c.Diag {
		g.dzNote(a, d, c, said)
	}
	g.dzMenu(text, nil, a)
}

// dzNote puts the diagnosis into the case notes.
func (g *Game) dzNote(a *Actor, d *dzDef, c *dzCond, said []string) {
	body := fmt.Sprintf("%s has %s (%s). %s", g.P.Name, strings.ToLower(g.dzDefName(d)), strings.ToLower(d.Stages[clampi(c.Stage, 0, len(d.Stages)-1)].Name), strings.Join(said, " "))
	f := knSt(g).add(knFact{Kind: "event", Subject: "player", Text: strings.TrimSpace(body), True: true, Day: lvDay(g), Ver: 1})
	g.knLearn(f, &lvPerson{Key: "dz:" + a.Name, Name: a.Name}, 2, "ask")
}

// ---------------------------------------------------------------- screen

func dzKeys(g *Game, k string) {
	st := g.dz()
	n := len(st.Conds)
	switch k {
	case "Escape", "q":
		g.Mode = "main"
	case "ArrowUp", "N8", "k":
		st.sel = maxi(0, st.sel-1)
	case "ArrowDown", "N2", "j":
		st.sel = mini(maxi(0, n-1), st.sel+1)
	case "r":
		if st.sel < n {
			c := st.Conds[st.sel]
			st.remCond, st.remList, st.remSel = c, g.dzRemedies(c), 0
			if len(st.remList) == 0 {
				g.Msg("{{K|You carry nothing that would help. Herbs, tonics and brews can: look for an herbalist.}}")
				return
			}
			g.Mode = "dzr"
		}
	case "t":
		if st.sel < n {
			c := st.Conds[st.sel]
			d := dzByID[c.ID]
			for i, cu := range d.Cures {
				if cu.Kind == "shrine" && g.dzRevealed(c, i) {
					if w := g.dzSite(c.ID); w != nil {
						x, y := w.P%g.W.W, w.P/g.W.W
						g.ceSetQuestTarget(x, y)
						g.Msg(fmt.Sprintf("{{c|The quest arrow points to %s (%d,%d).}}", w.Name, x, y))
						return
					}
				}
			}
			g.Msg("{{K|You do not know a place that would help yet.}}")
		}
	}
}

func dzrKeys(g *Game, k string) {
	st := g.dz()
	switch k {
	case "Escape", "q":
		g.Mode = "dz"
	case "ArrowUp", "N8", "k":
		st.remSel = maxi(0, st.remSel-1)
	case "ArrowDown", "N2", "j":
		st.remSel = mini(maxi(0, len(st.remList)-1), st.remSel+1)
	case "Enter", " ":
		if st.remCond != nil && st.remSel < len(st.remList) {
			g.dzUseRemedy(st.remCond, st.remList[st.remSel])
			g.Mode = "dz"
		}
	}
}

func (g *Game) dzrDraw() {
	sc := g.sc
	st := g.dz()
	x, y, w, _ := g.panel("Take a remedy")
	sc.TextB(x, y, "{{K|Up/Down choose  Enter take it  Esc back}}", CGray, CBgDark, x+w)
	y += 2
	for i, r := range st.remList {
		bg := CBgDark
		if i == st.remSel {
			bg = CBgSel
		}
		sc.Fill(x, y, w, 1, bg)
		raw := ""
		if r.Raw {
			raw = " (raw, weaker)"
		}
		sc.TextB(x, y, fmt.Sprintf("{{W|%s}} {{K|%s%s}}", Plain(r.It.Name), r.Why, raw), CGray, bg, x+w)
		y++
	}
}

func (g *Game) dzEffectLine(f dzFx, night bool) string {
	var parts []string
	add := func(n int, s string) {
		if n != 0 {
			parts = append(parts, fmt.Sprintf("%s %+d", s, n))
		}
	}
	add(f.Speed, "speed")
	add(f.DV, "defence")
	add(f.Hit, "to-hit")
	add(f.AV, "armour")
	for i, v := range f.A {
		add(v, strings.ToLower(AttrNames[i][:3]))
	}
	if len(parts) == 0 {
		return ""
	}
	s := strings.Join(parts, ", ")
	if night {
		return "at night: " + s
	}
	return s
}

func (g *Game) dzCureLine(c *dzCond, d *dzDef, i int) string {
	cu := d.Cures[i]
	if !g.dzRevealed(c, i) {
		return "{{K|A cure you have not yet heard of.}}"
	}
	var extra string
	switch cu.Kind {
	case "shrine":
		if w := g.dzSite(c.ID); w != nil && c.Planet == g.W.uniIdx {
			extra = fmt.Sprintf(" {{c|The place is %s (%d,%d).}}", w.Name, w.P%g.W.W, w.P/g.W.W)
		}
	case "grave":
		if c.SrcName != "" {
			extra = " {{c|The grave is at " + c.SrcName + ".}}"
		}
	case "kill":
		if c.SrcActor != 0 {
			extra = " {{c|Look for the one that did it.}}"
		}
	case "priest":
		extra = fmt.Sprintf(" {{K|(faith standing %d)}}", cu.Rank)
	}
	lim := ""
	if cu.Max > 0 {
		lim = fmt.Sprintf(" {{K|(in the first %d stages)}}", cu.Max)
	}
	pow := [3]string{"holds it", "eases it", "cures it"}[clampi(cu.Pow, 0, 2)]
	return fmt.Sprintf("{{W|%s}} %s{{K| [%s]}}%s%s", strings.Title(cu.Kind), cu.Hint, pow, lim, extra)
}

func (g *Game) dzDraw() {
	sc := g.sc
	st := g.dz()
	x, y, w, h := g.panel("Afflictions")
	bottom := y + h - 1
	sc.TextB(x, y, "{{K|Up/Down choose  r take a remedy from the pack  t point the quest arrow at a cure  Esc close}}", CGray, CBgDark, x+w)
	y += 2
	line := func(s string) {
		for _, ln := range Wrap(s, w) {
			if y < bottom {
				sc.TextB(x, y, ln, CGray, CBgDark, x+w)
				y++
			}
		}
	}
	if len(st.Conds) == 0 {
		line("{{G|You are free of afflictions.}}")
		if st.Cured > 0 {
			line(fmt.Sprintf("{{K|You have shaken off %d.}}", st.Cured))
		}
		for _, l := range st.Log {
			line("{{K|" + l + "}}")
		}
		return
	}
	st.sel = clampi(st.sel, 0, len(st.Conds)-1)
	for i, c := range st.Conds {
		d := dzByID[c.ID]
		bg := CBgDark
		if i == st.sel {
			bg = CBgSel
		}
		sc.Fill(x, y, w, 1, bg)
		s := d.Stages[clampi(c.Stage, 0, len(d.Stages)-1)]
		sc.TextB(x, y, fmt.Sprintf("{{W|%s}}  {{O|stage %d of %d: %s}}", g.dzCondName(c), c.Stage+1, len(d.Stages), strings.ToLower(s.Name)), CGray, bg, x+w)
		y++
	}
	y++
	c := st.Conds[st.sel]
	d := dzByID[c.ID]
	s := d.Stages[clampi(c.Stage, 0, len(d.Stages)-1)]
	line("{{Y|" + s.Name + "}}: " + s.Text)
	if e := g.dzEffectLine(s.Fx, false); e != "" {
		line("{{R|Effects: " + e + "}}")
	}
	if e := g.dzEffectLine(s.Night, true); e != "" {
		line("{{R|" + e + "}}")
	}
	if s.Drain > 0 {
		line(fmt.Sprintf("{{R|It costs you %d HP an hour.}}", s.Drain))
	}
	if s.Hunger > 0 || s.Thirst > 0 {
		line("{{R|You hunger and thirst faster.}}")
	}
	if g.Turn < c.Arrest {
		line(fmt.Sprintf("{{C|It is held at this stage for about %d more days.}}", (c.Arrest-g.Turn)/DayLen+1))
	} else if c.Stage < len(d.Stages)-1 || d.Loop {
		if c.Diag {
			line(fmt.Sprintf("{{O|It gets worse in about %d days.}}", maxi(1, (c.Next-g.Turn)/DayLen+1)))
		} else {
			line("{{K|It seems to be getting worse, but you cannot say how fast.}}")
		}
	} else {
		line("{{R|This is the last stage.}}")
	}
	if c.Src != "" {
		line("{{K|How it began: " + c.Src + "}}")
	}
	if !c.Diag {
		line("{{c|Not yet diagnosed. A physician, priest, herbalist or master can examine you: talk to them and ask about your condition.}}")
		return
	}
	y++
	line("{{Y|" + d.Lore + "}}")
	if d.Boon != "" {
		line("{{G|Boon and cost: " + d.Boon + "}}")
	}
	y++
	line("{{Y|Cures and relief}}")
	for i := range d.Cures {
		line("  " + g.dzCureLine(c, d, i))
	}
	if g.dzWard(c.ID) != nil {
		line("{{C|You carry a ward-amulet against this.}}")
	}
}
