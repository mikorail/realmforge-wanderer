package engine

import (
	"fmt"
	"sort"
	"strings"
)

// Legacy play (prefix lp): generational continuation when the wanderer dies or retires with a
// living descendant. Builds on family.go (fmHeir, fmContinueAsHeir), aging.go (will and heirlooms),
// halloffame.go (HFLegend) and dynasties_rogue.go without duplicating them.
//
// When the player dies with a living descendant, lpOnDeath flags a pending succession and offers
// to continue as a chosen heir. Continuation carries over a curated share of gear (tagged with the
// forebear's provenance) and half of the forebear's faction reputation (both goodwill and grudges),
// turns the old character's real deeds into local legends told by NPCs, and places gravestones at
// the site of death and in their home town.
//
// State: g.ext("lp") (global lineage chronicle, heirlooms, legends) and g.ext("lpl") (planet-local
// gravestones and visited-grave tracking, registered in unLocalKeys).
// Screen: mode "lp" (palette entry "Legacy and lineage"). Never uses g.R for generation.

const (
	lpKey  = "lp"
	lpLKey = "lpl"
)

type lpLife struct {
	Gen     int            `json:"g"`
	Name    string         `json:"n"`
	Species string         `json:"sp"`
	Calling string         `json:"c"`
	Level   int            `json:"lv"`
	Year    int            `json:"y"`
	Day     int            `json:"d"`
	Cause   string         `json:"ca"`
	Place   string         `json:"pl"`
	ZoneKey string         `json:"zk,omitempty"`
	WX      int            `json:"wx"`
	WY      int            `json:"wy"`
	Burg    int            `json:"bg,omitempty"`
	Planet  int            `json:"pn,omitempty"`
	Score   int            `json:"sc,omitempty"`
	Deeds   []string       `json:"dd,omitempty"`
	Gear    []string       `json:"gr,omitempty"`
	TopRep  map[string]int `json:"rp,omitempty"`
}

type lpGrave struct {
	LifeIdx int    `json:"li"`
	Name    string `json:"n"`
	Species string `json:"sp"`
	Calling string `json:"c"`
	Year    int    `json:"y"`
	Cause   string `json:"ca"`
	Deed    string `json:"dd,omitempty"`
	Place   string `json:"pl"`
	ZoneKey string `json:"zk,omitempty"`
	Burg    int    `json:"bg,omitempty"`
	P       int    `json:"p"`
	Kind    string `json:"k"` // grave | memorial
	Visited bool   `json:"v,omitempty"`
}

type lpState struct {
	Gen       int                 `json:"g"`
	House     string              `json:"h,omitempty"`
	Lives     []lpLife            `json:"l,omitempty"`
	Heirlooms map[int]string      `json:"hl,omitempty"` // item ID -> forebear name
	Legends   map[string][]string `json:"lg,omitempty"` // burg key or "world" -> legend lines
	Pending   bool                `json:"pd,omitempty"` // death succession waiting
	Log       []string            `json:"log,omitempty"`

	tab     int
	sel     int
	confirm bool
}

type lpLocal struct {
	Graves []lpGrave       `json:"gr,omitempty"`
	Told   map[string]bool `json:"td,omitempty"` // "npcID:lifeIdx" -> already heard
}

func (g *Game) lp() *lpState {
	st := g.ext(lpKey, func() any {
		return &lpState{
			Gen:       1,
			Heirlooms: map[int]string{},
			Legends:   map[string][]string{},
		}
	}).(*lpState)
	if st.Gen < 1 {
		st.Gen = 1
	}
	if st.Heirlooms == nil {
		st.Heirlooms = map[int]string{}
	}
	if st.Legends == nil {
		st.Legends = map[string][]string{}
	}
	return st
}

func (g *Game) lpl() *lpLocal {
	st := g.ext(lpLKey, func() any {
		return &lpLocal{
			Told: map[string]bool{},
		}
	}).(*lpLocal)
	if st.Told == nil {
		st.Told = map[string]bool{}
	}
	return st
}

func init() {
	unLocalKeys = append(unLocalKeys, lpLKey)
	turnHooks = append(turnHooks, lpTurn)
	zoneHooks = append(zoneHooks, lpZone)
	enterHooks = append(enterHooks, lpEnter)
	talkHooks = append(talkHooks, lpTalk)
	sideHooks = append(sideHooks, lpSide)
	modeKeys["lp"] = lpKeys
	modeDraws["lp"] = func(g *Game) { g.lpDraw() }
	paEntries = append(paEntries, paEntry{
		Name:     "Legacy and lineage",
		Keywords: "legacy heir lineage dynasty gravestone legends ancestor descendant retire succession",
		Cat:      "Screen",
		Label:    "screen",
		Func:     func(g *Game) { g.lpOpen() },
	})
}

func (g *Game) lpOpen() {
	st := g.lp()
	st.tab, st.sel, st.confirm = 0, 0, false
	g.Mode = "lp"
}

func (g *Game) lpHouseName() string {
	st := g.lp()
	if st.House != "" {
		return st.House
	}
	if g.P != nil && g.P.Name != "" {
		parts := strings.Fields(g.P.Name)
		if len(parts) > 1 {
			st.House = parts[len(parts)-1]
		} else {
			st.House = parts[0]
		}
		return st.House
	}
	return "Wanderer"
}

func (g *Game) lpLogf(s string) {
	st := g.lp()
	yr := 0
	if g.W != nil {
		yr = g.W.DateOf(nil, g.Day()).Year
	}
	st.Log = append(st.Log, fmt.Sprintf("Y%d: %s", yr, s))
	if len(st.Log) > 20 {
		st.Log = st.Log[len(st.Log)-20:]
	}
}

// ---- heir discovery ----

type lpHeirCand struct {
	Child   *fmChild
	Name    string
	Species string
	Calling string
	AgeY    float64
	Bond    int
	Grown   bool
}

// lpCandidates returns all living children eligible to inherit (grown children first, then youths >= 1.5y).
func (g *Game) lpCandidates() []lpHeirCand {
	if g.W == nil || g.P == nil {
		return nil
	}
	fm := g.fm()
	var out []lpHeirCand
	for _, k := range fm.Kids {
		if k == nil || k.Fate == "dead" || k.Fate == "lost" || k.Fate == "kidnapped" {
			continue
		}
		age := fmAgeYears(g, k)
		if age < 1.5 {
			continue
		}
		out = append(out, lpHeirCand{
			Child:   k,
			Name:    k.Name,
			Species: k.Species,
			Calling: k.Calling,
			AgeY:    age,
			Bond:    k.Bond,
			Grown:   age >= 3.0,
		})
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Grown != out[j].Grown {
			return out[i].Grown
		}
		return out[i].Child.Born < out[j].Child.Born
	})
	return out
}

// lpHasHeir reports whether at least one descendant can take up the mantle.
func (g *Game) lpHasHeir() bool {
	return len(g.lpCandidates()) > 0
}

// ---- death hook & succession ----

// lpOnDeath is the death hook (recommended 1-line calls in combat.go die() and aging.go agNaturalDeath()).
// It marks pending legacy succession if a living descendant exists and logs the offer.
func (g *Game) lpOnDeath() {
	if g.W == nil || g.P == nil {
		return
	}
	st := g.lp()
	if g.lpHasHeir() {
		st.Pending = true
		cands := g.lpCandidates()
		g.Msg(fmt.Sprintf("{{Y|Your line endures: %s can take up your mantle. Open Legacy and lineage to continue as your heir.}}", cands[0].Name))
	}
}

func (g *Game) lpSnapshotDeeds() []string {
	var deeds []string
	seen := map[string]bool{}
	for i := len(g.Deeds) - 1; i >= 0 && len(deeds) < 5; i-- {
		t := strings.TrimSpace(Plain(g.Deeds[i].Text))
		if t == "" || seen[t] {
			continue
		}
		seen[t] = true
		deeds = append(deeds, t)
	}
	return deeds
}

// lpInheritReputation passes half of the forebear's realm and faith reputation to the heir
// and returns a summary map of the strongest inherited standings.
func (g *Game) lpInheritReputation() map[string]int {
	top := map[string]int{}
	if g.Rep == nil {
		return top
	}
	keys := make([]string, 0, len(g.Rep))
	for k := range g.Rep {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		v := g.Rep[k]
		if !strings.HasPrefix(k, "state:") && !strings.HasPrefix(k, "faith:") {
			continue
		}
		inh := v / 2
		g.Rep[k] = inh
		if abs(inh) >= 25 {
			top[k] = inh
		}
	}
	return top
}

// lpContinueAsHeir hands the run to the chosen descendant index from lpCandidates().
func (g *Game) lpContinueAsHeir(idx int) bool {
	cands := g.lpCandidates()
	if len(cands) == 0 {
		return false
	}
	if idx < 0 || idx >= len(cands) {
		idx = 0
	}
	cand := cands[idx]
	st := g.lp()
	lpl := g.lpl()

	// Snapshot the old character before handover.
	oldName := g.P.Name
	oldSpecies := g.P.Species
	oldCalling := g.Calling
	oldLevel := g.P.Level
	oldScore := g.hfScore()
	year := g.W.DateOf(nil, g.Day()).Year
	cause := "retired from the road"
	if g.Dead && strings.TrimSpace(g.DeathText) != "" {
		cause = strings.TrimSpace(Plain(g.DeathText))
	}
	place := "the wilds"
	zkey := ""
	wx, wy, burg := g.WX, g.WY, g.StartBurg
	if g.Cur != nil {
		place = g.Cur.Name
		zkey = g.Cur.Key
		wx, wy = g.Cur.WX, g.Cur.WY
		if g.Cur.Burg > 0 {
			burg = g.Cur.Burg
		}
	}
	pIdx := 0
	if g.W.In(wx, wy) {
		pIdx = g.W.Idx(wx, wy)
	}
	deeds := g.lpSnapshotDeeds()
	if len(deeds) == 0 {
		deeds = []string{fmt.Sprintf("%s walked the roads of %s as a %s.", oldName, g.W.Name, g.Sk.Calling(oldCalling))}
	}

	// Curate gear to inherit: equipped items + top 4 pack items by value.
	var keepItems []*Item
	var gearNames []string
	for _, bp := range g.P.Body {
		if bp != nil && bp.Item != nil {
			cp := *bp.Item
			keepItems = append(keepItems, &cp)
			gearNames = append(gearNames, Plain(cp.Display()))
		}
	}
	pack := append([]*Item(nil), g.P.Inv...)
	sort.SliceStable(pack, func(i, j int) bool {
		if pack[i].Value != pack[j].Value {
			return pack[i].Value > pack[j].Value
		}
		return pack[i].ID < pack[j].ID
	})
	for i := 0; i < len(pack) && i < 4; i++ {
		cp := *pack[i]
		keepItems = append(keepItems, &cp)
		gearNames = append(gearNames, Plain(cp.Display()))
	}

	// Run aging will and martial lineage hooks before changing character.
	g.agBeforeHeir()
	g.mdRecordLineage()

	// Inherit half of realm/faith reputation.
	topRep := g.lpInheritReputation()

	// Record this life in lpState and Hall of Fame legends.
	life := lpLife{
		Gen:     st.Gen,
		Name:    oldName,
		Species: oldSpecies,
		Calling: oldCalling,
		Level:   oldLevel,
		Year:    year,
		Day:     g.Day(),
		Cause:   cause,
		Place:   place,
		ZoneKey: zkey,
		WX:      wx,
		WY:      wy,
		Burg:    burg,
		Planet:  g.W.uniIdx,
		Score:   oldScore,
		Deeds:   deeds,
		Gear:    gearNames,
		TopRep:  topRep,
	}
	lifeIdx := len(st.Lives)
	st.Lives = append(st.Lives, life)
	if len(st.Lives) > 40 {
		st.Lives = st.Lives[len(st.Lives)-40:]
		lifeIdx = len(st.Lives) - 1
	}
	g.lpBuildLegends(&life)

	// Register gravestones: one where they fell (if not retired), and a memorial in home burg.
	topDeed := deeds[0]
	if g.Dead && zkey != "" {
		lpl.Graves = append(lpl.Graves, lpGrave{
			LifeIdx: lifeIdx,
			Name:    oldName,
			Species: oldSpecies,
			Calling: oldCalling,
			Year:    year,
			Cause:   cause,
			Deed:    topDeed,
			Place:   place,
			ZoneKey: zkey,
			Burg:    burg,
			P:       pIdx,
			Kind:    "grave",
		})
	}
	homeBurg := g.StartBurg
	if homeBurg <= 0 {
		homeBurg = burg
	}
	if bd := g.W.burgByID[homeBurg]; bd != nil && bd.Alive {
		lpl.Graves = append(lpl.Graves, lpGrave{
			LifeIdx: lifeIdx,
			Name:    oldName,
			Species: oldSpecies,
			Calling: oldCalling,
			Year:    year,
			Cause:   cause,
			Deed:    topDeed,
			Place:   bd.Name,
			Burg:    bd.ID,
			P:       bd.P,
			Kind:    "memorial",
		})
	}
	if len(lpl.Graves) > 40 {
		lpl.Graves = lpl.Graves[len(lpl.Graves)-40:]
	}

	// Also push to Hall of Fame legends so cross-system callers see it.
	g.hf().Legends = append(g.hf().Legends, HFLegend{
		Name:    oldName,
		Species: oldSpecies,
		Calling: oldCalling,
		Year:    year,
		Cause:   cause,
		Place:   place,
		Level:   oldLevel,
		Score:   oldScore,
		Deeds:   deeds,
	})

	// Remove the chosen child from fm.Kids and any zone actor.
	fm := g.fm()
	var rest []*fmChild
	for _, k := range fm.Kids {
		if k == cand.Child {
			continue
		}
		rest = append(rest, k)
	}
	fm.Kids = rest
	if a, z := g.fmKidActor(cand.Child); a != nil && z != nil {
		g.removeActor(z, a)
	}
	fm.Spouse = nil
	fm.Worn = ""
	for id := range fm.Fame {
		fm.Fame[id] = fm.Fame[id] / 2
	}

	// Apply the heir's identity, attributes, level, and inherited gear.
	g.lpHouseName()
	st.Gen++
	st.Pending = false
	st.confirm = false

	g.P.Name = cand.Name
	if cand.Species != "" {
		g.P.Species = cand.Species
	}
	if cand.Calling != "" {
		g.Calling = strings.ToLower(cand.Calling)
	}
	if cand.Child != nil {
		g.P.Attr = cand.Child.Attr
	}
	frac := fmClampf(0.45+0.02*float64(cand.Bond), 0.45, 0.80)
	if !cand.Grown {
		frac *= 0.85
	}
	g.P.Level = maxi(1, int(float64(oldLevel)*frac))
	g.mdApplyLineageToHeir()
	g.XP = 0
	g.P.MaxHP = 14 + g.P.Level*4 + g.P.Attr[TOU]
	g.P.HP = g.P.MaxHP
	g.P.Status = map[string]int{}
	g.Hunger, g.Thirst = 0, 0
	g.Dead = false
	g.DeathText = ""

	// Inherit half of gold if not already spent by will.
	g.P.Gold = maxi(25, g.P.Gold/2)

	// Clear inventory/body and grant inherited gear with provenance.
	g.P.Inv = nil
	for i := range g.P.Body {
		if g.P.Body[i] != nil {
			g.P.Body[i].Item = nil
			g.P.Body[i].Severed = false
		}
	}
	if len(st.Heirlooms) > 64 {
		st.Heirlooms = map[int]string{}
	}
	for _, it := range keepItems {
		cp := *it
		cp.ID = g.newID()
		if cp.Extra == nil {
			cp.Extra = map[string]int{}
		}
		cp.Extra["lp_gen"] = st.Gen - 1
		note := fmt.Sprintf("Heirloom carried by %s (Gen %d, year %d).", oldName, st.Gen-1, year)
		if cp.Lore == "" {
			cp.Lore = note
		} else if !strings.Contains(cp.Lore, oldName) {
			cp.Lore = cp.Lore + " " + note
		}
		st.Heirlooms[cp.ID] = oldName
		g.give(&cp)
		g.equip(&cp)
	}

	// Place gravestone immediately if currently standing in that zone.
	if g.Cur != nil {
		g.lpPlaceGraves(g.Cur)
	}

	g.Mode = "main"
	g.recompute()
	g.agAfterHeir()
	g.deed("kinship", fmt.Sprintf("%s (Gen %d) took up the mantle of %s and walked on.", cand.Name, st.Gen, oldName), 0, 0)
	g.lpLogf(fmt.Sprintf("Gen %d: %s succeeded %s (level %d, %d heirlooms, %d inherited standings).", st.Gen, cand.Name, oldName, g.P.Level, len(keepItems), len(topRep)))
	g.Msg(fmt.Sprintf("{{Y|Generation %d begins: you walk as %s, heir of %s (level %d).}}", st.Gen, cand.Name, oldName, g.P.Level))
	if len(topRep) > 0 {
		g.Msg(fmt.Sprintf("{{c|Folk remember %s: half of their standing across %d realms and faiths clings to your name.}}", oldName, len(topRep)))
	}
	return true
}

func (g *Game) lpBuildLegends(life *lpLife) {
	st := g.lp()
	keys := []string{"world"}
	if life.Burg > 0 {
		keys = append(keys, fmt.Sprintf("b:%d", life.Burg))
	}
	if g.StartBurg > 0 && g.StartBurg != life.Burg {
		keys = append(keys, fmt.Sprintf("b:%d", g.StartBurg))
	}
	var lines []string
	for _, d := range life.Deeds {
		clean := strings.TrimSuffix(strings.TrimSpace(d), ".")
		if clean == "" {
			continue
		}
		lines = append(lines, fmt.Sprintf("Folk in %s still tell how %s.", life.Place, lowerFirst(clean)))
	}
	if life.Cause != "" && life.Cause != "retired from the road" {
		lines = append(lines, fmt.Sprintf("At %s in year %d, %s met their end: %s.", life.Place, life.Year, life.Name, lowerFirst(strings.TrimSuffix(life.Cause, "."))))
	} else {
		lines = append(lines, fmt.Sprintf("After %d levels on the road, %s laid down their pack in year %d.", life.Level, life.Name, life.Year))
	}
	for _, k := range keys {
		st.Legends[k] = append(st.Legends[k], lines...)
		if len(st.Legends[k]) > 18 {
			st.Legends[k] = st.Legends[k][len(st.Legends[k])-18:]
		}
	}
}

// ---- hooks: turn, zone, enter, talk, side ----

func lpTurn(g *Game) {
	if g.P == nil || g.W == nil {
		return
	}
	// Ensure house name is anchored once the player exists.
	if g.lp().House == "" && g.P.Name != "" {
		g.lpHouseName()
	}
}

func lpZone(g *Game, zn *Zone, _ *Rng) {
	if g.W == nil || zn == nil {
		return
	}
	g.lpPlaceGraves(zn)
}

func lpEnter(g *Game, zn *Zone) {
	if g.W == nil || zn == nil || g.P == nil {
		return
	}
	g.lpPlaceGraves(zn)
	lpl := g.lpl()
	for i := range lpl.Graves {
		gr := &lpl.Graves[i]
		if gr.Visited {
			continue
		}
		match := (gr.ZoneKey != "" && zn.Key == gr.ZoneKey) || (gr.Kind == "memorial" && zn.Kind == "town" && zn.Z == 0 && zn.Burg > 0 && zn.Burg == gr.Burg)
		if match {
			gr.Visited = true
			g.Msg(fmt.Sprintf("{{Y|You stand where a stone bears the name of your forebear, %s.}}", gr.Name))
			g.gainXP(15)
		}
	}
}

func (g *Game) lpPlaceGraves(zn *Zone) {
	lpl := g.lpl()
	if len(lpl.Graves) == 0 {
		return
	}
	if zn.Notes == nil {
		zn.Notes = map[int]string{}
	}
	for i := range lpl.Graves {
		gr := &lpl.Graves[i]
		match := false
		if gr.Kind == "grave" && gr.ZoneKey != "" && zn.Key == gr.ZoneKey {
			match = true
		} else if gr.Kind == "memorial" && zn.Kind == "town" && zn.Z == 0 && zn.Burg > 0 && zn.Burg == gr.Burg {
			match = true
		}
		if !match {
			continue
		}
		marker := "gravestone of " + gr.Name
		already := false
		for _, txt := range zn.Notes {
			if strings.Contains(txt, marker) {
				already = true
				break
			}
		}
		if already {
			continue
		}
		r := NewRng(fmt.Sprintf("%s:lp:grave:%s:%s:%s", g.Seed, zn.Key, gr.Name, gr.Kind))
		x, y, ok := zn.FreeSpot(r, func(x, y int) bool {
			t := zn.At(x, y)
			return t == TCobble || t == TFloor || t == TGrass || t == TDirt
		})
		if !ok {
			x, y, ok = zn.FreeSpot(r, nil)
			if !ok {
				continue
			}
		}
		zn.Set(x, y, TInscription)
		epitaph := fmt.Sprintf("Here stands the %s, %s %s (Year %d).", marker, strings.ToLower(gr.Species), strings.ToLower(gr.Calling), gr.Year)
		if gr.Deed != "" {
			epitaph += " Remembered for: " + gr.Deed
		}
		if gr.Cause != "" && gr.Kind == "grave" {
			epitaph += " Fell here: " + shorten(gr.Cause, 100)
		}
		zn.Notes[y*MW+x] = "{{Y|" + epitaph + "}}"
	}
}

func lpTalk(g *Game, a *Actor) []Option {
	if g.W == nil || g.P == nil || a == nil || a.HP <= 0 || a.AI == "follower" || g.Hostile(a) {
		return nil
	}
	st := g.lp()
	if len(st.Lives) == 0 {
		return nil
	}
	// Pick the most relevant forebear (local burg match first, otherwise most recent).
	burg := 0
	if g.Cur != nil {
		burg = g.Cur.Burg
	}
	lifeIdx := len(st.Lives) - 1
	for i := len(st.Lives) - 1; i >= 0; i-- {
		if burg > 0 && st.Lives[i].Burg == burg {
			lifeIdx = i
			break
		}
	}
	life := st.Lives[lifeIdx]
	npc := a
	return []Option{
		{
			Text: fmt.Sprintf("Ask what folk remember of %s.", life.Name),
			Do: func() {
				g.lpTellLegend(npc, lifeIdx)
			},
		},
	}
}

func (g *Game) lpTellLegend(a *Actor, lifeIdx int) {
	st := g.lp()
	if lifeIdx < 0 || lifeIdx >= len(st.Lives) {
		return
	}
	life := st.Lives[lifeIdx]
	h := HashStr(fmt.Sprintf("%s:lp:talk:%d:%d", g.Seed, a.ID, lifeIdx))
	var b strings.Builder
	openers := []string{
		fmt.Sprintf("\"You ask after %s? Aye, the road has not forgotten them.\"", life.Name),
		fmt.Sprintf("\"%s, the %s? Folk still tell stories by the hearth.\"", life.Name, strings.ToLower(life.Calling)),
		fmt.Sprintf("\"You have a look of %s about you, wanderer. People here remember them well.\"", life.Name),
	}
	b.WriteString(openers[h%uint64(len(openers))])
	if len(life.Deeds) > 0 {
		deed := life.Deeds[int((h/7)%uint64(len(life.Deeds)))]
		b.WriteString("\n\n\"They say: " + deed + "\"")
	}
	if life.Cause != "" && life.Cause != "retired from the road" {
		b.WriteString(fmt.Sprintf("\n\n\"Word came from %s in year %d that they fell. Someone raised a stone in their name.\"", life.Place, life.Year))
	} else {
		b.WriteString(fmt.Sprintf("\n\n\"They walked away from the road in year %d, and left their name behind.\"", life.Year))
	}
	lpl := g.lpl()
	key := fmt.Sprintf("%d:%d", a.ID, lifeIdx)
	if !lpl.Told[key] {
		if len(lpl.Told) > 128 {
			lpl.Told = map[string]bool{}
		}
		lpl.Told[key] = true
		g.gainXP(5)
	}
	g.talkMenu(b.String())
}

// lpGossip returns a local legend rumour about a forebear (optional hook for talk.go gossip).
func (g *Game) lpGossip(a *Actor) string {
	st := g.lp()
	if len(st.Lives) == 0 || a == nil {
		return ""
	}
	h := HashStr(fmt.Sprintf("%s:lp:gos:%d:%d", g.Seed, a.ID, g.Day()))
	if h%100 >= 35 {
		return ""
	}
	life := st.Lives[h%uint64(len(st.Lives))]
	if len(life.Deeds) > 0 {
		return fmt.Sprintf("\"Folk still speak of %s. %s\"", life.Name, life.Deeds[0])
	}
	return fmt.Sprintf("\"Old %s walked these roads in year %d. Their name still carries weight.\"", life.Name, life.Year)
}

func lpSide(g *Game) []string {
	st := g.lp()
	var out []string
	if st.Gen > 1 {
		out = append(out, fmt.Sprintf("{{Y|Gen %d: House %s}}", st.Gen, g.lpHouseName()))
	}
	if g.Dead && g.lpHasHeir() {
		out = append(out, "{{G|Heir ready: Legacy}}")
	}
	return out
}

// ---- screen: mode "lp" ----

func lpKeys(g *Game, k string) {
	st := g.lp()
	switch k {
	case "Escape", "q":
		st.confirm = false
		if g.Dead {
			g.Mode = "dead"
		} else {
			g.Mode = "main"
		}
	case "Tab", "ArrowRight", "l", "N6":
		st.tab = (st.tab + 1) % 3
		st.sel = 0
		st.confirm = false
	case "ArrowLeft", "h", "N4":
		st.tab = (st.tab + 2) % 3
		st.sel = 0
		st.confirm = false
	case "1":
		st.tab, st.sel, st.confirm = 0, 0, false
	case "2":
		st.tab, st.sel, st.confirm = 1, 0, false
	case "3":
		st.tab, st.sel, st.confirm = 2, 0, false
	case "ArrowDown", "j", "N2":
		st.sel++
		st.confirm = false
	case "ArrowUp", "k", "N8":
		st.sel = maxi(0, st.sel-1)
		st.confirm = false
	case "Enter", " ":
		if st.tab == 0 {
			cands := g.lpCandidates()
			if len(cands) == 0 {
				g.Msg("{{K|You have no descendant old enough to take up the mantle.}}")
				return
			}
			if st.sel >= len(cands) {
				st.sel = len(cands) - 1
			}
			if !st.confirm {
				st.confirm = true
				return
			}
			g.lpContinueAsHeir(st.sel)
		}
	}
}

func (g *Game) lpDraw() {
	st := g.lp()
	lpl := g.lpl()
	s := g.sc
	x, y0, w, h := g.panel("Legacy and lineage")
	y := y0

	tabs := []string{"1:Succession", "2:Forebears", "3:Legends & Graves"}
	tx := x
	for i, t := range tabs {
		col := "K"
		if i == st.tab {
			col = "Y"
		}
		s.TextB(tx, y, "{{"+col+"|["+t+"]}}", CGray, CBgDark, x+w)
		tx += len(t) + 4
	}
	y += 2
	maxY := y0 + h - 2

	switch st.tab {
	case 0:
		status := fmt.Sprintf("{{W|House %s}}  Generation %d  Current wanderer: {{Y|%s}} (Lv %d)", g.lpHouseName(), st.Gen, g.P.Name, g.P.Level)
		if g.Dead {
			status = fmt.Sprintf("{{R|%s has fallen.}} Choose a descendant to continue the line.", g.P.Name)
		}
		s.TextB(x, y, status, CGray, CBgDark, x+w)
		y += 2

		cands := g.lpCandidates()
		if len(cands) == 0 {
			for _, line := range Wrap("No descendant is old enough to inherit yet. Raise a family (F7) and wait until a child reaches youth (1.5+ years) or adulthood (3+ years).", w) {
				if y >= maxY {
					break
				}
				s.TextB(x, y, "{{K|"+line+"}}", CGray, CBgDark, x+w)
				y++
			}
		} else {
			if st.sel >= len(cands) {
				st.sel = len(cands) - 1
			}
			s.TextB(x, y, "{{W|Eligible heirs:}}", CGray, CBgDark, x+w)
			y++
			for i, c := range cands {
				if y >= maxY-6 {
					break
				}
				bg := CBgDark
				prefix := "  "
				if i == st.sel {
					bg = CBgSel
					prefix = "> "
				}
				stage := "youth"
				if c.Grown {
					stage = "grown"
				}
				s.Fill(x, y, w, 1, bg)
				line := fmt.Sprintf("%s{{Y|%s}} (%s, %s, age %.1fy, bond %d)", prefix, c.Name, strings.ToLower(c.Species), stage, c.AgeY, c.Bond)
				s.TextB(x, y, line, CGray, bg, x+w)
				y++
			}
			y++
			if y < maxY-3 {
				s.TextB(x, y, "{{c|Inheritance preview:}} equipped gear + top 4 pack items, 50% of realm/faith standing, half gold.", CGray, CBgDark, x+w)
				y++
				s.TextB(x, y, "{{c|Aftermath:}} deeds become local legends told by NPCs; gravestones raised at death site and home.", CGray, CBgDark, x+w)
				y += 2
			}
			if y < maxY {
				if st.confirm {
					s.TextB(x, y, "{{R|Press Enter again to pass the mantle to "+cands[st.sel].Name+".}}", CGray, CBgDark, x+w)
				} else if g.Dead {
					s.TextB(x, y, "{{G|Press Enter to continue as "+cands[st.sel].Name+".}}", CGray, CBgDark, x+w)
				} else {
					s.TextB(x, y, "{{Y|Press Enter to retire and pass the mantle to "+cands[st.sel].Name+".}}", CGray, CBgDark, x+w)
				}
			}
		}

	case 1:
		if len(st.Lives) == 0 {
			s.TextB(x, y, "{{K|No forebears recorded yet. You are the first generation of House "+g.lpHouseName()+".}}", CGray, CBgDark, x+w)
		} else {
			if st.sel >= len(st.Lives) {
				st.sel = len(st.Lives) - 1
			}
			for i := len(st.Lives) - 1; i >= 0 && y < maxY-6; i-- {
				lf := st.Lives[i]
				bg := CBgDark
				if i == st.sel {
					bg = CBgSel
				}
				s.Fill(x, y, w, 1, bg)
				s.TextB(x, y, fmt.Sprintf("Gen %d: {{Y|%s}} (%s %s, Lv %d, Year %d) - %s", lf.Gen, lf.Name, strings.ToLower(lf.Species), strings.ToLower(lf.Calling), lf.Level, lf.Year, shorten(lf.Cause, 32)), CGray, bg, x+w)
				y++
			}
			y++
			lf := st.Lives[st.sel]
			if y < maxY {
				s.TextB(x, y, "{{W|Deeds of "+lf.Name+":}}", CGray, CBgDark, x+w)
				y++
			}
			for _, d := range lf.Deeds {
				for _, line := range Wrap("- "+d, w-2) {
					if y >= maxY {
						break
					}
					s.TextB(x+1, y, line, CGray, CBgDark, x+w)
					y++
				}
			}
		}

	case 2:
		here := 0
		if g.Cur != nil && g.W.In(g.Cur.WX, g.Cur.WY) {
			here = g.W.Idx(g.Cur.WX, g.Cur.WY)
		}
		s.TextB(x, y, "{{W|Gravestones and memorials on this world:}}", CGray, CBgDark, x+w)
		y++
		if len(lpl.Graves) == 0 {
			s.TextB(x, y, "{{K|No gravestones raised on this world yet.}}", CGray, CBgDark, x+w)
			y += 2
		} else {
			for _, gr := range lpl.Graves {
				if y >= maxY-5 {
					break
				}
				vis := "{{K|unvisited}}"
				if gr.Visited {
					vis = "{{G|visited}}"
				}
				dir := g.wsDirDist(here, gr.P)
				s.TextB(x, y, fmt.Sprintf("- {{Y|%s}} (%s, %s, %s) %s", gr.Name, gr.Kind, gr.Place, dir, vis), CGray, CBgDark, x+w)
				y++
			}
			y++
		}
		if y < maxY-1 {
			s.TextB(x, y, "{{W|Local legends circulating:}}", CGray, CBgDark, x+w)
			y++
			legends := st.Legends["world"]
			if len(legends) == 0 {
				s.TextB(x, y, "{{K|None yet. Your deeds will become legends for your heirs.}}", CGray, CBgDark, x+w)
			} else {
				for i := len(legends) - 1; i >= 0 && y < maxY; i-- {
					for _, line := range Wrap("- "+legends[i], w-2) {
						if y >= maxY {
							break
						}
						s.TextB(x+1, y, line, CGray, CBgDark, x+w)
						y++
					}
				}
			}
		}
	}

	s.TextB(x, y0+h-1, "{{K|Tab/1-3 switch tab  Up/Down select  Enter act  Esc close}}", CGray, CBgDark, x+w)
}
