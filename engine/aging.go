package engine

import (
	"fmt"
	"sort"
	"strings"
)

// Aging and legacy (prefix ag). The player ages on the 112-day year clock (lvYear): species lifespans from
// the world's species data, five life stages with attribute shifts, wisdom, old-age illnesses, a dignified
// natural death, and a few genuinely rare ways to turn the clock back. Legacy page (mode "aglegacy",
// ability ag_legacy, key F12): bequest, will, monument, descendants. Descendants and monuments of past lives
// of the same world (hall of fame legends, halloffame.go) appear in later lives. State g.ext("ag") (global).
// Never draws g.R during generation: side rngs of seed+":ag:"+key. g.R only for play-time rolls.
//
// Optional hooks (see report): agXP, agCheckBonus, agRegenAdd, agUseItem, agOnDeath, agBeforeHeir/agAfterHeir,
// agSummary, agSagaLines, agDeathNote, agAbilities/agAbUse.

const agKey = "ag"

type agMonument struct {
	Planet int    `json:"pl"`
	Burg   int    `json:"b"`
	Name   string `json:"n"`
	Text   string `json:"t"`
	Year   int    `json:"y"`
}

type agWill struct {
	HeirPct  int    `json:"hp"` // percent of the gold that goes to heirs; the rest to the charity
	Charity  string `json:"ch,omitempty"`
	CharBurg int    `json:"cb,omitempty"`
	Items    []int  `json:"it,omitempty"` // item ids set aside for the heir
	Epitaph  int    `json:"ep,omitempty"`
	Done     bool   `json:"dn,omitempty"`
}

type agState struct {
	Name     string            `json:"n"`
	Inited   bool              `json:"i,omitempty"`
	AgeD     float64           `json:"a"`
	LastDay  int               `json:"ld"`
	LastAge  int               `json:"la"`
	Pace     int               `json:"pc,omitempty"` // 1 normal, 4 swift, 12 saga
	Stage    int               `json:"s"`
	Applied  [6]int            `json:"ap"`
	Warned   bool              `json:"w,omitempty"`
	RiteDone bool              `json:"rd,omitempty"`
	Rejuv    int               `json:"rj,omitempty"` // years turned back this life
	Springs  map[string][2]int `json:"sp,omitempty"`
	Used     map[string]bool   `json:"u,omitempty"`
	Prompted string            `json:"pr,omitempty"`
	Elixirs  []int             `json:"el,omitempty"`
	Will     agWill            `json:"wl"`
	Mons     []agMonument      `json:"mo,omitempty"`
	Held     []*Item           `json:"hd,omitempty"`
	HeldGold int               `json:"hg,omitempty"`
	Log      []string          `json:"lg,omitempty"`
	Lives    []string          `json:"lv,omitempty"` // legend lines of earlier lives in this save
}

func (g *Game) ag() *agState {
	return g.ext(agKey, func() any { return &agState{Pace: 1, Stage: -1, Will: agWill{HeirPct: 70, Charity: "none"}} }).(*agState)
}

type agUI struct{ tab, sel int }

var agUIs = map[*Game]*agUI{}

func agu(g *Game) *agUI {
	if u := agUIs[g]; u != nil {
		return u
	}
	u := &agUI{}
	agUIs[g] = u
	return u
}

func init() {
	turnHooks = append(turnHooks, agTurn)
	zoneHooks = append(zoneHooks, agZone)
	enterHooks = append(enterHooks, agEnter)
	talkHooks = append(talkHooks, agTalk)
	modeKeys["aglegacy"] = agKeys
	modeDraws["aglegacy"] = agDraw
	modeKeys["agmenu"] = func(g *Game, k string) { g.keyMenu(k, func() { agMenuClose(g) }) }
	modeDraws["agmenu"] = agDrawMenu
	if _, taken := mainKeys["F12"]; !taken {
		mainKeys["F12"] = func(g *Game) { agOpen(g) }
	}
	for _, il := range agIllnesses {
		d := &skStatusDef{ID: il.ID, Name: il.Name, Glyph: 'a'}
		switch il.ID {
		case "ag_joints":
			d.AGI, d.Speed = -2, -8
		case "ag_cough":
			d.MA, d.Speed = -1, -10
		case "ag_dim":
			d.Hit, d.DV = -3, -2
		case "ag_fog":
			d.INT, d.WIL = -2, -2
		case "ag_weak":
			d.Speed, d.DV, d.Hit = -15, -2, -2
		}
		skAddStatus(d)
	}
}

// ---------------------------------------------------------------- lifespan, stage

// agLifespan is the player's species lifespan in years: the life-sim table (lsLifespan) shaped by the world's
// species record (health, long/short-lived traits); species it has never heard of get a fallback.
func agLifespan(g *Game, species string) int {
	base := lsLifespan(species)
	if base == 70 {
		if v, ok := agSpeciesFallback[strings.ToLower(species)]; ok {
			base = v
		}
	}
	f := 1.0
	if g.W != nil {
		if sp := g.W.specByID[species]; sp != nil {
			f += float64(clampi(sp.Health, -30, 60)) / 200
			for _, t := range sp.Traits {
				lt := strings.ToLower(t)
				for _, w := range agLongTraits {
					if strings.Contains(lt, w) {
						f += 0.15
					}
				}
				for _, w := range agShortTraits {
					if strings.Contains(lt, w) {
						f -= 0.15
					}
				}
			}
		}
	}
	return clampi(int(float64(base)*f), 18, 3000)
}

func agStageIdx(frac float64) int {
	idx := 0
	for i, s := range agStages {
		if frac >= s.From {
			idx = i
		}
	}
	return idx
}

func (st *agState) years() int { return int(st.AgeD / agYear) }

func agInit(g *Game) *agState {
	st := g.ag()
	if g.P == nil {
		return st
	}
	if !st.Inited || st.Name != g.P.Name {
		r := NewRng(g.Seed + ":ag:start:" + g.P.Name)
		life := agLifespan(g, g.P.Species)
		st.Inited, st.Name = true, g.P.Name
		st.AgeD = float64(life) * (0.30 + 0.06*r.Float()) * agYear
		st.LastDay = lvDay(g)
		st.LastAge = st.years()
		st.Stage = -1
		st.Applied = [6]int{}
		st.Warned, st.RiteDone, st.Rejuv = false, false, 0
		if st.Pace == 0 {
			st.Pace = 1
		}
		st.Will = agWill{HeirPct: 70, Charity: "none"}
		agApplyStage(g, st, false)
	}
	return st
}

// agStageNow: index into agStages, from the age and lifespan.
func agStageNow(g *Game) int {
	st := agInit(g)
	life := float64(agLifespan(g, g.P.Species))
	return agStageIdx(st.AgeD / agYear / life)
}

func agApplyStage(g *Game, st *agState, announce bool) {
	life := float64(agLifespan(g, g.P.Species))
	idx := agStageIdx(st.AgeD / agYear / life)
	if idx == st.Stage {
		return
	}
	st.Stage = idx
	want := agStages[idx].Attr
	for i := 0; i < 6; i++ {
		diff := want[i] - st.Applied[i]
		if diff == 0 {
			continue
		}
		nv := maxi(3, g.P.Attr[i]+diff)
		st.Applied[i] += nv - g.P.Attr[i]
		g.P.Attr[i] = nv
	}
	if announce {
		if m := agStageChange[agStages[idx].ID]; m != "" {
			g.Msg(m)
		}
		g.deed("age", fmt.Sprintf("%s came to be %s at the age of %d.", g.P.Name, agStages[idx].Name, st.years()), 0, 0)
	}
}

// ---------------------------------------------------------------- hooks used by other systems

// agXP scales experience by the wisdom of your years.
func (g *Game) agXP(n int) int {
	if g.P == nil || g.Sandboxed() {
		return n
	}
	return int(float64(n)*agStages[agStageNow(g)].XPMul + 0.5)
}

// agCheckBonus is added to skill-check modifiers (checks.go checkMod).
func (g *Game) agCheckBonus(kind string) int {
	if g.P == nil || g.Sandboxed() {
		return 0
	}
	return agStages[agStageNow(g)].Check[kind]
}

// agRegenAdd is added to the turns between natural healing ticks (game.go regen).
func (g *Game) agRegenAdd() int {
	if g.P == nil || g.Sandboxed() {
		return 0
	}
	return agStages[agStageNow(g)].Regen
}

// agUseItem handles the rejuvenation draught (powers.go useItem).
func (g *Game) agUseItem(it *Item) bool {
	if it == nil || it.Effect != "ag_rejuv" {
		return false
	}
	st := agInit(g)
	n := 8 + g.R.Int(0, 8)
	agTurnBack(g, st, n, "The draught is cold, then warm, then a memory of summer.")
	g.consume(it)
	return true
}

func agTurnBack(g *Game, st *agState, years int, why string) {
	life := float64(agLifespan(g, g.P.Species))
	floor := life * 0.30 * agYear
	st.AgeD = maxf(floor, st.AgeD-float64(years*agYear))
	st.Rejuv += years
	st.LastAge = st.years()
	g.Msg(fmt.Sprintf("{{C|%s Your body forgets %d winters. (You are now %d.)}}", why, years, st.years()))
	g.deed("age", fmt.Sprintf("%s turned back the years by %d.", g.P.Name, years), 0, 0)
	agApplyStage(g, st, true)
	for _, il := range agIllnesses {
		delete(g.P.Status, il.ID)
	}
}

// ---------------------------------------------------------------- turn

func agTurn(g *Game) {
	if g.P == nil || g.W == nil || g.Dead || g.Sandboxed() || g.Cur == nil {
		return
	}
	st := agInit(g)
	agSpringCheck(g, st)
	if g.Turn%40 == 0 {
		agElixirWatch(g, st)
	}
	day := lvDay(g)
	if day <= st.LastDay {
		return
	}
	st.AgeD += float64((day - st.LastDay) * maxi(1, st.Pace))
	st.LastDay = day
	prev := st.LastAge
	age := st.years()
	st.LastAge = age
	agApplyStage(g, st, true)
	if age > prev {
		agBirthday(g, st, age)
		if g.Dead {
			return
		}
	}
	agIllRoll(g, st)
}

func agBirthday(g *Game, st *agState, age int) {
	r := NewRng(fmt.Sprintf("%s:ag:bd:%s:%d", g.Seed, g.P.Name, age))
	stg := agStages[st.Stage]
	line := Pick(r, agBirthdays[stg.ID])
	msg := fmt.Sprintf("{{Y|Year %d: you are %d.}} %s", lvYear(g), age, line)
	if st.Stage >= 2 {
		xp := 4 + g.P.Level*2
		g.gainXP(xp)
		msg += fmt.Sprintf(" {{c|(+%d XP of hindsight)}}", xp)
	}
	g.Msg(msg)
	if age%10 == 0 {
		st.Log = append(st.Log, fmt.Sprintf("Turned %d in the year %d.", age, lvYear(g)))
	}
	life := agLifespan(g, g.P.Species)
	f := float64(age) / float64(life)
	if f >= 0.95 && !st.Warned {
		st.Warned = true
		g.Msg("{{R|You feel the end of the road nearing. Settle your affairs while you can: Legacy (F12).}}")
	}
	if f >= 0.95 && g.GameMode != "wander" {
		p := 0.02 + (f-0.95)*3.0
		if p > 0.85 {
			p = 0.85
		}
		if g.R.Chance(p) {
			agNaturalDeath(g, age)
		}
	}
}

func agIllRoll(g *Game, st *agState) {
	if st.Stage < 2 {
		return
	}
	chance := []float64{0, 0, 0, 0.03, 0.08}[st.Stage] + 0.015*float64(maxi(0, 2-g.P.M(TOU)))
	if st.Stage == 2 {
		chance = 0.008
	}
	if !g.R.Chance(chance) {
		return
	}
	var pool []agIllness
	for _, il := range agIllnesses {
		if st.Stage >= il.MinStage && g.P.St(il.ID) == 0 {
			pool = append(pool, il)
		}
	}
	if len(pool) == 0 {
		return
	}
	il := pool[g.R.Int(0, len(pool)-1)]
	g.P.AddSt(il.ID, il.Dur)
	g.Msg(il.Msg)
}

// ---------------------------------------------------------------- natural death

func agNaturalDeath(g *Game, age int) {
	st := g.ag()
	place := "the road"
	if g.Cur != nil {
		place = g.Cur.Name
	}
	r := NewRng(fmt.Sprintf("%s:ag:death:%s:%d", g.Seed, g.P.Name, age))
	g.Dead = true
	g.DeathText = fmt.Sprintf(Pick(r, agDeathCause), place, g.timeOfDay(), age)
	g.Msg("{{Y|" + g.DeathText + "}}")
	g.deed("death", fmt.Sprintf("The wanderer %s, %s, died of old age in %s at the age of %d, and was mourned.", g.P.Name, g.Sk.Calling(g.Calling), place, age), 0, 0)
	g.Mode = "dead"
	g.agOnDeath(true)
	_ = st
	g.qcOnDeath()
}

// agDeathNote: a clause for the death screen of any death (hook in drawDead lines).
func (g *Game) agDeathNote() []string {
	if g.P == nil || g.W == nil || g.Sandboxed() {
		return nil
	}
	st := g.ag()
	if !st.Inited {
		return nil
	}
	out := []string{fmt.Sprintf("You were %d, %s.", st.years(), agStages[agStageNow(g)].Name)}
	if l := g.agLegendLine(); l != "" {
		out = append(out, l)
	}
	return out
}

// ---------------------------------------------------------------- will and legacy

func agHeirs(g *Game) []string {
	var out []string
	st := g.fm()
	for _, k := range st.Kids {
		if k.Fate == "dead" || k.Fate == "lost" || k.Fate == "kidnapped" {
			continue
		}
		out = append(out, k.Name)
	}
	if st.Spouse != nil && st.Spouse.Gone == "" {
		out = append(out, st.Spouse.Name+" (spouse)")
	}
	return out
}

func agCharityOf(id string) agCharity {
	for _, c := range agCharities {
		if c.ID == id {
			return c
		}
	}
	return agCharities[0]
}

// agOnDeath executes the will once: gold to the charity, items set aside for the heir, a news line. Idempotent.
func (g *Game) agOnDeath(natural bool) {
	if g.P == nil || g.W == nil || g.Sandboxed() {
		return
	}
	st := agInit(g)
	if st.Will.Done {
		return
	}
	st.Will.Done = true
	heirs := agHeirs(g)
	hasHeir := g.fmHeir() != nil
	gold := g.cuTotal()
	ch := agCharityOf(st.Will.Charity)
	burg := st.Will.CharBurg
	if burg == 0 && g.Cur != nil {
		burg = g.Cur.Burg
	}
	if burg == 0 {
		burg = g.StartBurg
	}
	town := lvBurgName(g, burg)
	day := lvDay(g)
	if ch.ID != "none" {
		pct := 100 - clampi(st.Will.HeirPct, 0, 100)
		if !hasHeir {
			pct = 100
		}
		give := gold * pct / 100
		if give > 0 && g.cuSpend(give) {
			if ch.News != "" {
				lvSt(g).addNews(day, burg, ch.News, g.P.Name, town)
			}
			st.Log = append(st.Log, fmt.Sprintf("%s received %s from the estate.", ch.Name, g.cuFormat(give)))
		}
	}
	var kept []*Item
	for _, id := range st.Will.Items {
		for _, it := range g.P.Inv {
			if it.ID == id {
				kept = append(kept, it)
				break
			}
		}
	}
	for _, it := range kept {
		cp := *it
		g.removeInv(it)
		if hasHeir {
			st.Held = append(st.Held, &cp)
		} else if v := cp.Value / 2; v > 0 {
			st.HeldGold += v
		}
	}
	if len(kept) > 0 && hasHeir {
		st.Log = append(st.Log, fmt.Sprintf("%d heirlooms were set aside for %s.", len(kept), heirs[0]))
	}
	if st.HeldGold > 0 && ch.ID != "none" {
		lvSt(g).addNews(day, burg, "The goods of %s were sold for the good of %s.", g.P.Name, town)
	}
	// descendants of a family heard of it: remember the death in the home town
	lvSt(g).addNews(day, burg, "%s, the wanderer, is dead. %s", g.P.Name, map[bool]string{true: "They died of old age, and were mourned.", false: "The news travelled slowly."}[natural])
	if !natural {
		if n := g.agDeathNote(); len(n) > 0 {
			g.DeathText = strings.TrimSpace(g.DeathText + " " + n[0])
		}
	}
	st.Lives = append(st.Lives, g.agLegendLine())
}

// agBeforeHeir / agAfterHeir wrap fmContinueAsHeir (family.go).
func (g *Game) agBeforeHeir() {
	if g.P != nil && !g.Ag().Will.Done {
		g.agOnDeath(!g.Dead)
	}
}

// Ag exposes the state to tests and the hall.
func (g *Game) Ag() *agState { return g.ag() }

func (g *Game) agAfterHeir() {
	st := g.ag()
	held, hg := st.Held, st.HeldGold
	st.Held, st.HeldGold, st.Inited = nil, 0, false
	agInit(g)
	for _, it := range held {
		it.ID = g.newID()
		g.give(it)
	}
	if len(held) > 0 {
		g.Msg(fmt.Sprintf("{{Y|You carry %d heirlooms from your forebear.}}", len(held)))
	}
	_ = hg
}

// agLegendLine is the one line the saga and the hall tell of this life.
func (g *Game) agLegendLine() string {
	if g.P == nil {
		return ""
	}
	st := g.ag()
	if !st.Inited {
		return ""
	}
	l := fmt.Sprintf("%s, %s %s, lived %d winters", g.P.Name, agStages[agStageNow(g)].Name, strings.ToLower(g.P.Species), st.years())
	if n := len(agHeirs(g)); n > 0 {
		l += fmt.Sprintf(", leaving %d to carry the name", n)
	}
	if st.Rejuv > 0 {
		l += fmt.Sprintf(", having turned back %d years", st.Rejuv)
	}
	for _, m := range st.Mons {
		if m.Planet == g.PlanetIndex() {
			l += fmt.Sprintf(", remembered by a monument in %s", lvBurgName(g, m.Burg))
			break
		}
	}
	return l + "."
}

// agSummary: JSON-friendly map for HFSummary ("legacy").
func (g *Game) agSummary() map[string]any {
	if g.P == nil {
		return nil
	}
	st := g.ag()
	return map[string]any{"age": st.years(), "stage": agStages[agStageNow(g)].ID, "legend": g.agLegendLine(),
		"monuments": len(st.Mons), "rejuv": st.Rejuv, "heirs": len(agHeirs(g))}
}

// agSagaLines: lines for the saga page.
func (g *Game) agSagaLines() []string {
	st := g.ag()
	if !st.Inited {
		return nil
	}
	out := []string{g.agLegendLine()}
	out = append(out, st.Log...)
	return out
}

// ---------------------------------------------------------------- zones: springs, elixirs, monuments

func agSurfaceBurgs(g *Game) []int {
	var ids []int
	for i := range g.W.Burgs {
		if b := &g.W.Burgs[i]; b.Alive && b.Layer == "surface" {
			ids = append(ids, b.ID)
		}
	}
	return ids
}

func agLegendBurg(g *Game, l *HFLegend, n int) int {
	ids := agSurfaceBurgs(g)
	if len(ids) == 0 {
		return -1
	}
	return ids[NewRng(fmt.Sprintf("%s:ag:lb:%s:%d", g.Seed, l.Name, n)).Int(0, len(ids)-1)]
}

func agZone(g *Game, zn *Zone, _ *Rng) {
	if g.W == nil {
		return
	}
	st := g.ag()
	if zn.Kind == "town" && zn.Z == 0 && zn.Burg > 0 {
		for _, m := range st.Mons {
			if m.Planet == g.PlanetIndex() && m.Burg == zn.Burg {
				agPlaceMonument(g, zn, m)
			}
		}
		for i := range g.hf().Legends {
			l := &g.hf().Legends[i]
			if l.Score >= 600 || l.Level >= 6 {
				if zn.Burg == agLegendBurg(g, l, 0) {
					agPlaceMonument(g, zn, agMonument{Name: l.Name, Year: l.Year, Text: agLegendMonText(l)})
				}
			}
		}
		return
	}
	if zn.Kind == "town" || zn.Underwater || zn.Z < 0 {
		return
	}
	r := NewRng(g.Seed + ":ag:rare:" + zn.Key)
	if r.Chance(0.0012) { // a rare spring (surface or underground)
		if x, y, ok := zn.FreeSpot(r, nil); ok {
			zn.Set(x, y, TInscription)
			if zn.Notes == nil {
				zn.Notes = map[int]string{}
			}
			zn.Notes[y*MW+x] = "{{C|" + Pick(r, agSpringLines) + "}}"
			if st.Springs == nil {
				st.Springs = map[string][2]int{}
			}
			st.Springs[zn.Key] = [2]int{x, y}
		}
	}
	if r.Chance(0.0015) { // a rare draught
		if x, y, ok := zn.FreeSpot(r, nil); ok {
			zn.Items = append(zn.Items, agElixir(g, x, y))
		}
	}
}

func agElixir(g *Game, x, y int) *Item {
	return &Item{ID: g.newID(), Name: "draught of second spring", Glyph: '!', F: CCyan, Kind: "tonic", Effect: "ag_rejuv",
		Value: 900, Rarity: "ultrarare", Ident: true, X: x, Y: y,
		Lore: "A cold clear liquid that smells of rain on warm stone. Those who drink it say the years loosen their grip."}
}

func agLegendMonText(l *HFLegend) string {
	s := fmt.Sprintf("{{Y|%s}}, %s %s. Year %d.", l.Name, strings.ToLower(l.Species), strings.ToLower(l.Calling), l.Year)
	if len(l.Deeds) > 0 {
		s += " " + strings.TrimSpace(Plain(l.Deeds[0]))
	}
	return s + " \"" + hfEpitaph(l) + "\""
}

func agPlaceMonument(g *Game, zn *Zone, m agMonument) {
	if zn.Notes == nil {
		zn.Notes = map[int]string{}
	}
	for _, v := range zn.Notes {
		if strings.Contains(v, m.Name) && strings.Contains(v, "monument") {
			return
		}
	}
	r := NewRng(g.Seed + ":ag:mon:" + zn.Key + ":" + m.Name)
	x, y, ok := zn.FreeSpot(r, func(x, y int) bool { return zn.At(x, y) == TCobble || zn.At(x, y) == TFloor })
	if !ok {
		x, y, ok = zn.FreeSpot(r, nil)
		if !ok {
			return
		}
	}
	zn.Set(x, y, TStatue)
	zn.Notes[y*MW+x] = fmt.Sprintf("A monument to the wanderer %s. %s", m.Name, m.Text)
}

func agSpringCheck(g *Game, st *agState) {
	if len(st.Springs) == 0 || g.Mode != "main" {
		return
	}
	pos, ok := st.Springs[g.Cur.Key]
	if !ok || st.Used[g.Cur.Key] {
		return
	}
	if g.P.X != pos[0] || g.P.Y != pos[1] {
		if st.Prompted == g.Cur.Key {
			st.Prompted = ""
		}
		return
	}
	if st.Prompted == g.Cur.Key {
		return
	}
	st.Prompted = g.Cur.Key
	key := g.Cur.Key
	g.Popup = Wrap(Plain(Pick(NewRng(g.Seed+":ag:sp:"+key), agSpringLines))+" It is very quiet here.", 60)
	g.Opts = []Option{
		{"Drink from the spring.", func() {
			st.Used[key] = true
			agMenuClose(g)
			if st.Stage < 1 && st.years() < 30 {
				g.Msg("The water is cold and sweet, and does nothing you can feel.")
				return
			}
			agTurnBack(g, st, 10+g.R.Int(0, 10), "The spring runs through you like a bell.")
			g.Msg("The spring murmurs, and runs dry.")
		}},
		{"Leave it be.", func() { agMenuClose(g) }},
	}
	if st.Used == nil {
		st.Used = map[string]bool{}
	}
	g.Sel = 0
	g.Mode = "agmenu"
}

func agMenuClose(g *Game) { g.Mode, g.Popup, g.Opts = "main", nil, nil }

func agDrawMenu(g *Game) {
	s := g.sc
	h := len(g.Popup) + len(g.Opts) + 4
	y0 := maxi(1, (MH-h)/2)
	s.Box(2, y0, SW-4, mini(h, SH-2), CDarkCyan, "A spring")
	y := y0 + 1
	for _, l := range g.Popup {
		s.TextB(6, y, l, CGray, CBgDark, SW-3)
		y++
	}
	y++
	for i, o := range g.Opts {
		bg := CBgDark
		if i == g.Sel {
			bg = CBgSel
		}
		s.Fill(4, y, SW-8, 1, bg)
		s.TextB(4, y, fmt.Sprintf("{{W|%c)}} %s", 'a'+i, o.Text), CGray, bg, SW-3)
		y++
	}
}

// agElixirWatch notes draughts in the pack; if one vanishes without agUseItem hooked and is neither on the floor
// nor on a trader, it was drunk: apply the effect anyway.
func agElixirWatch(g *Game, st *agState) {
	have := map[int]bool{}
	for _, it := range g.P.Inv {
		if it.Effect == "ag_rejuv" {
			have[it.ID] = true
		}
	}
	var keep []int
	for _, id := range st.Elixirs {
		if have[id] {
			keep = append(keep, id)
			continue
		}
		found := false
		for _, it := range g.Cur.Items {
			if it.ID == id {
				found = true
			}
		}
		if g.Talk != nil {
			for _, it := range g.Talk.Inv {
				if it.ID == id {
					found = true
				}
			}
		}
		if !found && g.Mode == "main" {
			agTurnBack(g, st, 8+g.R.Int(0, 8), "The draught is cold, then warm, then a memory of summer.")
		}
	}
	for id := range have {
		dup := false
		for _, k := range keep {
			if k == id {
				dup = true
			}
		}
		if !dup {
			keep = append(keep, id)
		}
	}
	st.Elixirs = keep
}

// ---------------------------------------------------------------- enter: descendants of past lives

func agEnter(g *Game, zn *Zone) {
	if g.W == nil || zn == nil || zn.Kind != "town" || zn.Burg <= 0 || zn.Z != 0 {
		return
	}
	ls := g.hf().Legends
	if len(ls) == 0 {
		return
	}
	lvEnsure(g, zn.Burg)
	st := lvSt(g)
	bd := g.W.burgByID[zn.Burg]
	if bd == nil {
		return
	}
	cu := g.W.cultByID[bd.Culture]
	year := lvYear(g)
	for i := range ls {
		l := &ls[i]
		r0 := NewRng(fmt.Sprintf("%s:ag:desc:%s", g.Seed, l.Name))
		count := 1 + r0.Int(0, 1)
		life := lsLifespan(l.Species)
		for n := 0; n < count; n++ {
			if agLegendBurg(g, l, 10+n) != zn.Burg {
				continue
			}
			key := fmt.Sprintf("ag:%s:%d", l.Name, n)
			if st.People[key] != nil {
				continue
			}
			r := NewRng(fmt.Sprintf("%s:ag:descp:%s:%d", g.Seed, l.Name, n))
			born := l.Year + 12 + n*20 + r.Int(0, 10)
			age := year - born
			if age < 18 || age > life*7/10 {
				continue
			}
			surname := l.Name
			if f := strings.Fields(l.Name); len(f) > 1 {
				surname = f[len(f)-1]
			}
			given := lvGiven(g, cu, r)
			p := lvBase(g, bd, cu, key, given+" "+surname, Pick(r, agLegendTrades), born, r.Chance(0.5), r)
			if l.Species != "" {
				p.Species = l.Species
			}
			p.Opinion = 10
			p.addLife(born, "birth", fmt.Sprintf("Born in %s, of the line of the wanderer %s.", bd.Name, l.Name))
			p.Goal, p.GoalID = lvPickGoal(r, p, nil, nil, g)
			lvPickFear(r, p)
			st.People[key] = p
		}
	}
}

// ---------------------------------------------------------------- talk

func agPriestly(a *Actor) bool {
	s := strings.ToLower(a.Role + " " + a.Title)
	for _, w := range []string{"priest", "cleric", "shaman", "druid", "monk", "oracle", "abbot", "imam", "rabbi", "acolyte"} {
		if strings.Contains(s, w) {
			return true
		}
	}
	return false
}

func agTalk(g *Game, a *Actor) []Option {
	if g.P == nil || g.W == nil || g.Sandboxed() || a == nil || a.Tpl != "npc" {
		return nil
	}
	st := agInit(g)
	var out []Option
	if p := lvPersonFor(g, a); p != nil && strings.HasPrefix(p.Key, "ag:") {
		out = append(out, Option{"Ask about their family.", func() { agDescTalk(g, a, p) }})
	}
	tag := "ag_" + agStages[st.Stage].ID
	if !a.Has(tag) && a.AI != "hostile" && a.AI != "follower" {
		out = append(out, Option{"Remark on your years.", func() { agRemark(g, a, tag) }})
	}
	if st.Stage >= 3 && !st.RiteDone && agPriestly(a) {
		cost := 500 + g.P.Level*40
		out = append(out, Option{fmt.Sprintf("Ask about the Rite of Renewal (%s).", g.cuFormat(cost)), func() { agRite(g, a, st, cost) }})
	}
	return out
}

func agRemark(g *Game, a *Actor, tag string) {
	st := g.ag()
	a.Set(tag)
	rel := 2
	if p := lvPersonFor(g, a); p != nil {
		myBorn := lvYear(g) - st.years()
		if p.Born < myBorn-3 {
			rel = 0
		} else if p.Born > myBorn+3 {
			rel = 1
		}
		if st.Stage >= 3 && rel == 1 {
			p.remember(lvDay(g), "help", "Gave you their respect for your years.", 2)
		}
	} else if a.Role == "elder" {
		rel = 0
	}
	ls := agTalkLines[agStages[st.Stage].ID][rel]
	r := NewRng(fmt.Sprintf("%s:ag:talk:%d:%s", g.Seed, a.ID, agStages[st.Stage].ID))
	g.showPopup(a.Name, Pick(r, ls))
}

func agDescTalk(g *Game, a *Actor, p *lvPerson) {
	parts := strings.Split(p.Key, ":")
	name := ""
	if len(parts) >= 2 {
		name = parts[1]
	}
	var l *HFLegend
	for i := range g.hf().Legends {
		if g.hf().Legends[i].Name == name {
			l = &g.hf().Legends[i]
		}
	}
	r := NewRng(g.Seed + ":ag:dt:" + p.Key)
	txt := Pick(r, agDescTalkOpen)
	if l != nil {
		txt += fmt.Sprintf(" \"%s was a %s, they say, who lived in the year %d.", l.Name, strings.ToLower(l.Calling), l.Year)
		if len(l.Deeds) > 0 {
			txt += " The family tale is: " + lowerFirst(strings.TrimSuffix(strings.TrimSpace(Plain(l.Deeds[0])), ".")) + "."
		}
		txt += "\""
		if !a.Has("ag_gave") {
			a.Set("ag_gave")
			it := &Item{ID: g.newID(), Name: l.Name + "'s letter", Glyph: '?', F: CYellow, Kind: "letter", Value: 40, Ident: true,
				Text: fmt.Sprintf("A folded letter in a worn hand. It says: \"To whoever follows: I walked far, and the road was worth it. Level %d at the end. Remember me kindly. - %s\"", l.Level, l.Name)}
			g.give(it)
			txt += fmt.Sprintf(" They press %s into your hand.", it.Name)
		}
	}
	g.showPopup(a.Name, txt)
}

func agRite(g *Game, a *Actor, st *agState, cost int) {
	if g.cuTotal() < cost {
		g.showPopup(a.Name, "\"The Rite asks a gift to the temple. You do not have it yet.\"")
		return
	}
	if !g.cuSpend(cost) {
		return
	}
	st.RiteDone = true
	g.Msg(Plain(Pick(NewRng(g.Seed+":ag:rite:"+a.Name), agRiteLines)))
	if g.R.Chance(0.55) {
		agTurnBack(g, st, 6+g.R.Int(0, 8), "The rite takes.")
		g.showPopup(a.Name, "\"It has worked. It does not always.\"")
	} else {
		g.P.AddSt("ag_weak", 1200)
		g.showPopup(a.Name, "\"The rite did not take. This too is a teaching. Rest, and be gentle with yourself.\"")
	}
}

// ---------------------------------------------------------------- legacy page

func agAbilities(g *Game) []Ability {
	if g.P == nil || g.Cur == nil {
		return nil
	}
	return []Ability{{ID: "ag_legacy", Name: "Legacy and years", Target: "self"}}
}

func (g *Game) agAbilities() []Ability { return agAbilities(g) }

func (g *Game) agAbUse(id string) {
	if id == "ag_legacy" {
		agOpen(g)
	}
}

func agOpen(g *Game) {
	if g.P == nil || g.W == nil {
		return
	}
	agInit(g)
	u := agu(g)
	u.sel = 0
	g.Mode = "aglegacy"
}

var agTabs = []string{"Years", "Bequest", "Will", "Monument", "Line"}

func agKeys(g *Game, k string) {
	u := agu(g)
	st := g.ag()
	switch k {
	case "Escape", "q":
		g.Mode = "main"
		return
	case "Tab", "ArrowRight", "l":
		u.tab, u.sel = (u.tab+1)%len(agTabs), 0
		return
	case "ArrowLeft", "h":
		u.tab, u.sel = (u.tab+len(agTabs)-1)%len(agTabs), 0
		return
	case "ArrowUp", "k":
		u.sel = maxi(0, u.sel-1)
		return
	case "ArrowDown", "j":
		u.sel++
		return
	}
	if k >= "1" && k <= "5" {
		u.tab, u.sel = int(k[0]-'1'), 0
		return
	}
	if k != "Enter" && k != " " {
		return
	}
	switch u.tab {
	case 0: // pace of years
		st.Pace = map[int]int{1: 4, 4: 12, 12: 1}[maxi(1, st.Pace)]
		if st.Pace == 0 {
			st.Pace = 1
		}
	case 1:
		inv := agBequestable(g)
		if u.sel >= len(inv) {
			return
		}
		it := inv[u.sel]
		for i, id := range st.Will.Items {
			if id == it.ID {
				st.Will.Items = append(st.Will.Items[:i], st.Will.Items[i+1:]...)
				return
			}
		}
		if len(st.Will.Items) >= 6 {
			g.Msg("A will can name six heirlooms at most.")
			return
		}
		st.Will.Items = append(st.Will.Items, it.ID)
	case 2:
		switch u.sel {
		case 0:
			st.Will.HeirPct = (st.Will.HeirPct + 25) % 125
			if st.Will.HeirPct > 100 {
				st.Will.HeirPct = 0
			}
		case 1:
			idx := 0
			for i, c := range agCharities {
				if c.ID == st.Will.Charity {
					idx = i
				}
			}
			st.Will.Charity = agCharities[(idx+1)%len(agCharities)].ID
		case 2:
			if g.Cur != nil && g.Cur.Kind == "town" && g.Cur.Burg > 0 {
				st.Will.CharBurg = g.Cur.Burg
				lvSt(g).addNews(lvDay(g), g.Cur.Burg, "%s has made a will at %s.", g.P.Name, lvBurgName(g, g.Cur.Burg))
				g.Msg("You have the will witnessed here.")
			} else {
				g.Msg("A will must be witnessed in a town.")
			}
		}
	case 3:
		agCommission(g, st)
	}
}

func agBequestable(g *Game) []*Item {
	var out []*Item
	for _, it := range g.P.Inv {
		if it.Value >= 20 || it.Kind == "weapon" || it.Kind == "armor" || it.Kind == "book" || it.Kind == "artifact" {
			out = append(out, it)
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Value > out[j].Value })
	return out
}

func agCommission(g *Game, st *agState) {
	if g.Cur == nil || g.Cur.Kind != "town" || g.Cur.Burg <= 0 || g.Cur.Z != 0 {
		g.Msg("A monument can only be raised in a town.")
		return
	}
	for _, m := range st.Mons {
		if m.Planet == g.PlanetIndex() && m.Burg == g.Cur.Burg {
			g.Msg("There is already a monument to you in this town.")
			return
		}
	}
	cost := 150 + g.P.Level*30
	if !g.cuSpend(cost) {
		g.Msg("The masons want " + g.cuFormat(cost) + " for a monument.")
		return
	}
	ep := agEpitaphs[st.Will.Epitaph%len(agEpitaphs)]
	m := agMonument{Planet: g.PlanetIndex(), Burg: g.Cur.Burg, Name: g.P.Name, Year: lvYear(g),
		Text: fmt.Sprintf("Raised in the year %d. \"%s\"", lvYear(g), ep)}
	st.Mons = append(st.Mons, m)
	agPlaceMonument(g, g.Cur, m)
	g.deed("kinship", fmt.Sprintf("%s had a monument raised in %s.", g.P.Name, g.Cur.Name), 0, 0)
	lvSt(g).addNews(lvDay(g), g.Cur.Burg, "Masons in %s raised a monument to the wanderer %s.", g.Cur.Name, g.P.Name)
	g.Msg("{{Y|The masons set the stone. It will outlast you.}}")
}

func agDraw(g *Game) {
	s := g.sc
	st := agInit(g)
	u := agu(g)
	x, y, w, h := g.panel("Legacy")
	tx := x
	for i, t := range agTabs {
		c := "K"
		if u.tab == i {
			c = "Y"
		}
		tx = s.TextB(tx, y, fmt.Sprintf("{{%s|[%d %s]}} ", c, i+1, t), CGray, CBgDark, x+w)
	}
	y += 2
	line := func(txt string) {
		for _, l := range Wrap(txt, w) {
			if y < y+h {
				s.TextB(x, y, l, CGray, CBgDark, x+w)
				y++
			}
		}
	}
	row := func(i int, txt string) {
		bg := CBgDark
		if i == u.sel {
			bg = CBgSel
		}
		s.Fill(x, y, w, 1, bg)
		s.TextB(x+1, y, txt, CGray, bg, x+w)
		y++
	}
	life := agLifespan(g, g.P.Species)
	switch u.tab {
	case 0:
		stg := agStages[st.Stage]
		line(fmt.Sprintf("{{W|%s}}, %s %s: %d winters of about %d.", g.P.Name, stg.Name, strings.ToLower(g.P.Species), st.years(), life))
		line(fmt.Sprintf("Stage: {{Y|%s}}: %s.", stg.Name, stg.Note))
		var at []string
		for i := 0; i < 6; i++ {
			if st.Applied[i] != 0 {
				at = append(at, fmt.Sprintf("%s %+d", AttrShort[i], st.Applied[i]))
			}
		}
		if len(at) == 0 {
			at = []string{"none"}
		}
		line("Attribute shifts from age: " + strings.Join(at, ", ") + ".")
		line(fmt.Sprintf("Wisdom: experience x%.2f; healing %+d turns between ticks.", stg.XPMul, stg.Regen))
		if st.Rejuv > 0 {
			line(fmt.Sprintf("Years turned back so far: %d.", st.Rejuv))
		}
		y++
		pace := map[int]string{1: "normal (a year per 112 days)", 4: "swift (x4)", 12: "saga (x12)"}[maxi(1, st.Pace)]
		row(0, "Years pass: "+pace+"  (Enter to change)")
		y++
		line("{{K|Rejuvenation is rare: a draught of second spring in some deep place, an ageless spring, or the Rite of Renewal from an old priest.}}")
	case 1:
		heirs := agHeirs(g)
		if len(heirs) == 0 {
			line("{{K|You have no heirs yet. Unclaimed gifts go to your charity (see Will).}}")
		} else {
			line("Heirs: " + strings.Join(heirs, ", ") + ".")
		}
		y++
		inv := agBequestable(g)
		for i, it := range inv {
			if i >= h-8 {
				break
			}
			mark := " "
			for _, id := range st.Will.Items {
				if id == it.ID {
					mark = "x"
				}
			}
			row(i, fmt.Sprintf("[%s] %s  (%s)", mark, it.Name, g.cuFormat(it.Value)))
		}
		if len(inv) == 0 {
			line("{{K|Nothing worth naming in your pack.}}")
		}
	case 2:
		ch := agCharityOf(st.Will.Charity)
		row(0, fmt.Sprintf("Gold to heirs: %d%%  (Enter cycles)", st.Will.HeirPct))
		row(1, "Remainder to: "+ch.Name+"  (Enter cycles)")
		where := "wherever you die"
		if st.Will.CharBurg > 0 {
			where = lvBurgName(g, st.Will.CharBurg)
		}
		row(2, "Witnessed at: "+where+"  (Enter in a town to witness)")
		y++
		line(fmt.Sprintf("Heirloom items named: %d. The will is read and carried out when you die, and goes into the town's news.", len(st.Will.Items)))
	case 3:
		cost := 150 + g.P.Level*30
		row(0, fmt.Sprintf("Commission a monument in this town: %s  (Enter)", g.cuFormat(cost)))
		y++
		line("Epitaph: \"" + agEpitaphs[st.Will.Epitaph%len(agEpitaphs)] + "\"")
		for _, m := range st.Mons {
			line(fmt.Sprintf("Monument in %s, year %d.", lvBurgName(g, m.Burg), m.Year))
		}
		y++
		line("{{K|Past wanderers of high renown get a monument in some town of this world, and kin who may greet you.}}")
	case 4:
		line("{{W|Legend:}} " + g.agLegendLine())
		y++
		var ds []string
		for k, p := range lvSt(g).People {
			if strings.HasPrefix(k, "ag:") {
				ds = append(ds, fmt.Sprintf("%s, %s of %s", p.Name, p.Role, lvBurgName(g, p.Home)))
			}
		}
		sort.Strings(ds)
		if len(ds) == 0 {
			line("{{K|No descendants of earlier wanderers are known in this land.}}")
		}
		for _, d := range ds {
			line("Descendant: " + d)
		}
		for _, l := range st.Lives {
			if l != "" {
				line("Earlier: " + l)
			}
		}
	}
	s.Text(x, y+0, "", CGray)
	s.Text(x, 2+h-1, "{{K|Tab/Left/Right: page   Up/Down: select   Enter: act   Esc: close}}", CGray)
}
