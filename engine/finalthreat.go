package engine

import (
	"fmt"
	"sort"
	"strings"
)

// Final threat & New Game+ (prefix ft): a world-ending cataclysm generated from the world's own
// history, genre, wonders, religions, hazards, and realms, paired with a 5-stage final quest chain
// and a New Game+ cycle option (harder world scaling, kept legacy items).
//
// Designed to complement endgame.go (eg), deadlines.go (dl), and politics.go (po) without duplicating
// eg's spreading war-room system: ft tracks a doom clock (accelerated by active political crises and
// unresolved threats), a structured final quest chain (Omens -> 3 Wards -> Forge World-Ward Relic ->
// Confront the World-Ender in its Sanctum -> New Game+ Cycle), and cross-cycle progression.
//
// State:
//   - g.ext("ft")  (planet-local, registered in unLocalKeys): the current planet's cataclysm & quest chain.
//   - g.ext("ftg") (global): New Game+ cycle counter, past victories, and carried legacy items.
// Screen: mode "ft" (palette entry "Final threat & New Game+").
// Generation uses side rngs (NewRng(g.Seed+":ft:...")), never g.R.

const (
	ftKey       = "ft"
	ftGlobalKey = "ftg"
	ftArenaKey  = "ft:sanctum"
)

type ftArch struct {
	ID       string
	Genres   []string
	Title    string
	HarbName string
	BossTpl  string
	MobTpls  []string
	Omen     string
	Relic    string
	Floor    Terrain
	Hazard   Terrain
	Glyph    rune
	Col      byte
}

var ftArchs = []ftArch{
	{
		ID: "unravelling", Genres: []string{"fantasy", "mythic", "twilight"},
		Title: "The Unravelling of the Leys", HarbName: "The Weaver of Ruin",
		BossTpl: "horror", MobTpls: []string{"wisp", "ashwraith", "shardmoth"},
		Omen: "Stars blur at dusk and ancient stones hum with a hollow chord.", Relic: "Aegis of the Bound Ley",
		Floor: TCrystalFloor, Hazard: TRift, Glyph: 'Ω', Col: CMagenta,
	},
	{
		ID: "cataclysm", Genres: []string{"dieselpunk", "atompunk", "steampunk"},
		Title: "The Great Sundering", HarbName: "The Ash Sovereign",
		BossTpl: "magma", MobTpls: []string{"magma", "salamander", "golem"},
		Omen: "Deep tremors crack the roads and red dust drifts across the horizon.", Relic: "Keystone of Quiet Earth",
		Floor: TAsh, Hazard: TLava, Glyph: '▲', Col: COrange,
	},
	{
		ID: "singularity", Genres: []string{"cyberpunk", "solarpunk"},
		Title: "The Grey Convergence", HarbName: "The Prime Architect",
		BossTpl: "golem", MobTpls: []string{"golem", "crawler", "turret"},
		Omen: "Silent signals hijack every relay and cold light flickers on the sky.", Relic: "Null-Cipher Core",
		Floor: TMachine, Hazard: TCrystal, Glyph: '¤', Col: CCyan,
	},
	{
		ID: "bloom", Genres: []string{"biopunk", "wasteland"},
		Title: "The World-Eating Bloom", HarbName: "The Root Mother",
		BossTpl: "shambler", MobTpls: []string{"shambler", "sporefolk", "ooze"},
		Omen: "Pale spores blanket the dawn and roots split the old paving stones.", Relic: "Censer of Still Growth",
		Floor: TFungus, Hazard: TMushroom, Glyph: '&', Col: CGreen,
	},
	{
		ID: "eclipse", Genres: []string{"fantasy", "mythic", "twilight", "wasteland"},
		Title: "The Endless Eclipse", HarbName: "The Sun-Eater",
		BossTpl: "wight", MobTpls: []string{"wight", "skeleton", "ghoul"},
		Omen: "Noon grows dim as a black ring creeps across the sun.", Relic: "Lantern of the First Dawn",
		Floor: TSnow, Hazard: TIce, Glyph: '☼', Col: CYellow,
	},
	{
		ID: "dominion", Genres: []string{"fantasy", "steampunk", "dieselpunk"},
		Title: "The Crown of Ashes", HarbName: "The Undying Regent",
		BossTpl: "chief", MobTpls: []string{"bandit", "archer", "wight"},
		Omen: "Black banners appear on the high passes and old treaties turn to ash.", Relic: "Seal of the Free Realms",
		Floor: TDirt, Hazard: TRubble, Glyph: '¥', Col: CRed,
	},
}

func ftArchByID(id string) *ftArch {
	for i := range ftArchs {
		if ftArchs[i].ID == id {
			return &ftArchs[i]
		}
	}
	return &ftArchs[0]
}

type ftVictory struct {
	Cycle      int    `json:"cy"`
	ThreatName string `json:"tn"`
	BossName   string `json:"bn"`
	HeroName   string `json:"hn"`
	Level      int    `json:"lv"`
	Year       int    `json:"y"`
	Day        int    `json:"d"`
}

type ftGlobal struct {
	Cycle     int         `json:"cy"`
	Victories []ftVictory `json:"vc,omitempty"`
}

type ftState struct {
	Inited    bool            `json:"in,omitempty"`
	Arch      string          `json:"ar,omitempty"`
	Name      string          `json:"n,omitempty"`
	BossName  string          `json:"bn,omitempty"`
	LoreNote  string          `json:"ln,omitempty"`
	Stage     int             `json:"st"` // 0 dormant, 1 omens, 2 wards, 3 forge, 4 sanctum, 5 resolved
	Doom      int             `json:"dm"` // 0..100
	DoomMile  int             `json:"ml"`
	LastDay   int             `json:"ld"`
	AnchorP   int             `json:"ap"`
	LoreBurg  int             `json:"lb"`
	PowerP    int             `json:"pp"`
	PowerName string          `json:"pn,omitempty"`
	UnitySt   int             `json:"us"`
	WardLore  bool            `json:"wl,omitempty"`
	WardPower bool            `json:"wp,omitempty"`
	WardUnity bool            `json:"wu,omitempty"`
	Forged    bool            `json:"fg,omitempty"`
	SanctumOn bool            `json:"so,omitempty"`
	Defeated  bool            `json:"df,omitempty"`
	Placed    map[string]bool `json:"pl,omitempty"`
	Log       []string        `json:"lg,omitempty"`

	tab     int
	sel     int
	ngPick  map[int]bool
	confirm bool
}

func (g *Game) ftg() *ftGlobal {
	return g.ext(ftGlobalKey, func() any { return &ftGlobal{} }).(*ftGlobal)
}

func (g *Game) ft() *ftState {
	st := g.ext(ftKey, func() any {
		return &ftState{
			Placed: map[string]bool{},
		}
	}).(*ftState)
	if st.Placed == nil {
		st.Placed = map[string]bool{}
	}
	if st.ngPick == nil {
		st.ngPick = map[int]bool{}
	}
	return st
}

func init() {
	unLocalKeys = append(unLocalKeys, ftKey)
	turnHooks = append(turnHooks, ftTurn)
	zoneHooks = append(zoneHooks, ftZone)
	enterHooks = append(enterHooks, ftEnter)
	killHooks = append(killHooks, ftKill)
	talkHooks = append(talkHooks, ftTalk)
	sideHooks = append(sideHooks, ftSide)
	modeKeys["ft"] = ftKeys
	modeDraws["ft"] = func(g *Game) { g.ftDraw() }
	paEntries = append(paEntries, paEntry{
		Name:     "Final threat & New Game+",
		Keywords: "final threat endgame cataclysm doom quest chain new game plus ng+ cycle legacy items",
		Cat:      "Screen",
		Label:    "screen",
		Func:     func(g *Game) { g.ftOpen() },
	})
}

func (g *Game) ftOpen() {
	st := g.ft()
	g.ftEnsure()
	st.tab, st.sel, st.confirm = 0, 0, false
	g.Mode = "ft"
}

func (g *Game) ftLogf(s string) {
	st := g.ft()
	st.Log = append(st.Log, fmt.Sprintf("Day %d: %s", g.Day()+1, s))
	if len(st.Log) > 20 {
		st.Log = st.Log[len(st.Log)-20:]
	}
}

// ---- world-derived threat generation ----

func (g *Game) ftEnsure() *ftState {
	st := g.ft()
	if st.Inited || g.W == nil {
		return st
	}
	w := g.W
	cyc := g.ftg().Cycle
	r := NewRng(fmt.Sprintf("%s:ft:gen:%d", g.Seed, cyc))

	// Score archetypes from world data (genre, hazards, wars, wonders, religions).
	var pool []ftArch
	for _, a := range ftArchs {
		wt := 1
		for _, gn := range a.Genres {
			if gn == w.Genre {
				wt += 3
			}
		}
		if a.ID == "cataclysm" && w.Hazards != nil && (len(w.Hazards.Volcanoes) > 0 || len(w.Hazards.Faults) > 0) {
			wt += 3
		}
		if a.ID == "dominion" && len(g.egWarStates()) > 0 {
			wt += 2
		}
		if a.ID == "eclipse" && len(w.Religions) > 0 {
			wt++
		}
		for i := 0; i < wt; i++ {
			pool = append(pool, a)
		}
	}
	arch := pool[r.Int(0, len(pool)-1)]

	// Derive name & lore from real world features.
	threatName := arch.Title
	bossName := arch.HarbName
	lore := arch.Omen

	if len(w.Events) > 0 {
		ev := w.Events[r.Int(0, len(w.Events)-1)]
		if strings.TrimSpace(ev.Text) != "" {
			lore = fmt.Sprintf("%s Scholars tie this omen to an older memory from year %d: %s", arch.Omen, ev.Y, shorten(Plain(ev.Text), 120))
		}
	}
	if len(w.Religions) > 0 && r.Chance(0.5) {
		rel := w.Religions[r.Int(0, len(w.Religions)-1)]
		if rel.Deity != "" {
			lore += fmt.Sprintf(" Priests of %s call it the last trial of %s.", rel.Name, rel.Deity)
		}
	}
	if cyc > 0 {
		threatName = fmt.Sprintf("%s (Cycle %d)", threatName, cyc+1)
		bossName = fmt.Sprintf("%s of the %dth Turning", arch.HarbName, cyc+1)
	}

	// Pick Anchor parasang: prefer a wonder or mega, else a distant land parasang.
	anchorP := -1
	if len(w.Wonders) > 0 {
		wd := w.Wonders[r.Int(0, len(w.Wonders)-1)]
		if wd.P >= 0 && wd.P < len(w.Biome) && !w.IsWater(wd.P) {
			anchorP = wd.P
		}
	}
	if anchorP < 0 && len(w.Megas) > 0 {
		mg := w.Megas[r.Int(0, len(w.Megas)-1)]
		if mg.P >= 0 && mg.P < len(w.Biome) && !w.IsWater(mg.P) {
			anchorP = mg.P
		}
	}
	if anchorP < 0 {
		anchorP = g.ftPickLand(r, 6)
	}

	// Pick Ward of Lore burg (a living surface town or ruin).
	loreBurg := g.StartBurg
	var burgs []*BurgData
	for i := range w.Burgs {
		b := &w.Burgs[i]
		if b.Alive && (b.Layer == "" || b.Layer == "surface") && !w.IsWater(b.P) {
			burgs = append(burgs, b)
		}
	}
	if len(burgs) > 0 {
		loreBurg = burgs[r.Int(0, len(burgs)-1)].ID
	}

	// Pick Ward of Power parasang (near anchor or distant wilds) and lieutenant name.
	powerP := g.ftPickLand(r, 4)
	if powerP == anchorP {
		powerP = g.ftPickLand(r, 3)
	}
	powerTitles := []string{"Dread Herald", "Ash Warden", "Hollow Champion", "Rift-Bearer", "Doom-Keeper"}
	powerName := Pick(r, powerTitles)
	if len(w.Monsters) > 0 {
		m := w.Monsters[r.Int(0, len(w.Monsters)-1)]
		if m.Name != "" {
			powerName = "Echo of " + strings.TrimPrefix(m.Name, "The ")
		}
	}

	// Pick Ward of Unity realm (a living state with at least one living surface burg).
	unitySt := g.HomeState
	hasBurg := map[int]bool{}
	for _, b := range burgs {
		if b.State > 0 {
			hasBurg[b.State] = true
		}
	}
	var states []*StateData
	for i := range w.States {
		if w.States[i].ID > 0 && w.States[i].Alive && hasBurg[w.States[i].ID] {
			states = append(states, &w.States[i])
		}
	}
	if len(states) > 0 {
		unitySt = states[r.Int(0, len(states)-1)].ID
	}

	st.Inited = true
	st.Arch = arch.ID
	st.Name = threatName
	st.BossName = bossName
	st.LoreNote = lore
	st.AnchorP = anchorP
	st.LoreBurg = loreBurg
	st.PowerP = powerP
	st.PowerName = powerName
	st.UnitySt = unitySt
	st.LastDay = g.Day()

	// In New Game+, the threat wakes immediately at Stage 1.
	if cyc > 0 {
		st.Stage = 1
		st.Doom = 10
		g.ftLogf("Cycle " + fmt.Sprint(cyc+1) + " opens under the shadow of " + st.Name + ".")
	}
	return st
}

func (g *Game) ftPickLand(r *Rng, minD int) int {
	w := g.W
	start := w.Idx(clampi(g.StartX, 0, w.W-1), clampi(g.StartY, 0, w.H-1))
	n := len(w.Biome)
	for tries := 0; tries < 500; tries++ {
		p := r.Int(0, n-1)
		if w.IsWater(p) || w.Burg[p] > 0 {
			continue
		}
		if wsCheb(w, p, start) >= minD || tries > 300 {
			return p
		}
	}
	for p := range w.Biome {
		if !w.IsWater(p) {
			return p
		}
	}
	return 0
}

// ---- quest chain progression ----

func (g *Game) ftAwaken() {
	st := g.ftEnsure()
	if st.Stage > 0 {
		return
	}
	st.Stage = 1
	st.Doom = maxi(st.Doom, 8)
	st.LastDay = g.Day()
	g.ftLogf(st.Name + " stirs. " + ftArchByID(st.Arch).Omen)
	g.Msg("{{R|A world-ending omen rises: " + st.Name + ".}} {{K|(Open Final threat & New Game+ in palette)}}")
	g.deed("endgame", fmt.Sprintf("The first omens of %s appeared while %s walked the world.", st.Name, g.P.Name), 0, 0)
}

func (g *Game) ftAdvanceIfReady() {
	st := g.ft()
	if st.Stage == 2 && st.WardLore && st.WardPower && st.WardUnity {
		st.Stage = 3
		st.Doom = maxi(0, st.Doom-12)
		g.ftLogf("All three wards secured. Ready to forge the counter-relic.")
		g.Msg("{{Y|All three wards against " + st.Name + " are in hand. Forge the counter-relic from the Final Threat screen.}}")
	}
}

func (g *Game) ftForgeRelic() bool {
	st := g.ftEnsure()
	if st.Stage < 3 || st.Forged || !st.WardLore || !st.WardPower || !st.WardUnity {
		return false
	}
	arch := ftArchByID(st.Arch)
	cyc := g.ftg().Cycle
	it := &Item{
		ID:     g.newID(),
		Name:   arch.Relic,
		Glyph:  '*',
		F:      CYellow,
		Kind:   "artifact",
		Slot:   "float",
		AV:     2 + cyc,
		DV:     2 + cyc,
		Value:  1200 + cyc*400,
		Weight: 1,
		Tier:   5,
		Rarity: "unique",
		Ident:  true,
		Bonus:  map[int]int{WIL: 2 + cyc, TOU: 1 + cyc},
		Extra:  map[string]int{"ft_relic": 1, "ft_cycle": cyc},
		Lore:   fmt.Sprintf("Forged from three wards to break %s. Its light steadies the world.", st.Name),
	}
	g.give(it)
	st.Forged = true
	st.Stage = 4
	st.Doom = maxi(0, st.Doom-15)
	g.gainXP(150 + g.P.Level*20)
	g.ftLogf("Forged " + arch.Relic + ". The Sanctum awaits at " + g.wsPlace(st.AnchorP) + ".")
	g.deed("relic", fmt.Sprintf("%s forged %s to confront %s.", g.P.Name, arch.Relic, st.Name), 0, 0)
	g.Msg("{{Y|You forge " + arch.Relic + "! March to " + g.wsPlace(st.AnchorP) + " to enter the Sanctum.}}")
	return true
}

func (g *Game) ftHasRelic() bool {
	if g.P == nil {
		return false
	}
	for _, it := range g.P.Inv {
		if it != nil && it.Extra != nil && it.Extra["ft_relic"] > 0 {
			return true
		}
	}
	for _, bp := range g.P.Body {
		if bp != nil && bp.Item != nil && bp.Item.Extra != nil && bp.Item.Extra["ft_relic"] > 0 {
			return true
		}
	}
	return false
}

// ---- Sanctum finale ----

func (g *Game) ftSanctumZone() *Zone {
	if zn := g.Zones[ftArenaKey]; zn != nil {
		return zn
	}
	st := g.ftEnsure()
	arch := ftArchByID(st.Arch)
	w := g.W
	ax, ay := st.AnchorP%w.W, st.AnchorP/w.W
	zn := newZone(ftArenaKey, ax, ay, 1, 1, 0)
	zn.Kind = "sanctum"
	zn.Name = "Sanctum of " + strings.TrimPrefix(st.Name, "The ")
	zn.Biome = w.Biome[st.AnchorP]
	zn.Level = g.zoneLevel(ax, ay, 0) + 4 + g.ftg().Cycle*2

	r := NewRng(fmt.Sprintf("%s:ft:sanctum:%d", g.Seed, g.ftg().Cycle))
	for i := range zn.T {
		zn.T[i] = arch.Floor
	}
	for i := 0; i < 36; i++ {
		x, y := r.Int(8, MW-9), r.Int(2, MH-3)
		if abs(x-64) > 5 || abs(y-12) > 3 {
			zn.Set(x, y, TPillar)
		}
	}
	for i := 0; i < 45; i++ {
		x, y := r.Int(10, MW-10), r.Int(2, MH-3)
		if zn.At(x, y) == arch.Floor && (abs(x-64) > 3 || abs(y-12) > 2) {
			zn.Set(x, y, arch.Hazard)
		}
	}
	for dy := -1; dy <= 1; dy++ {
		for dx := -3; dx <= 3; dx++ {
			zn.Set(4+dx, 12+dy, arch.Floor)
		}
	}
	zn.Set(2, 12, TInscription)
	zn.Notes[12*MW+2] = "The threshold back to the world. Walk off the edge to retreat and regroup."

	// Spawn the World-Ender boss and retinue.
	cyc := g.ftg().Cycle
	lvl := clampi(maxi(g.P.Level+4, 14)+cyc*3, 14, 50)
	boss := g.Spawn(arch.BossTpl, lvl, r)
	boss.Name, boss.Proper, boss.Boss = st.BossName, true, true
	boss.Glyph, boss.F = arch.Glyph, arch.Col
	boss.Faction, boss.AI, boss.Angry = "eldritch", "hostile", true
	boss.Desc = fmt.Sprintf("The harbinger of %s.", st.Name)
	boss.Set("boss")
	boss.Set("ftboss")
	hp := int(float64(320+lvl*20) * (1.0 + 0.35*float64(cyc)))
	if g.ftHasRelic() {
		hp = hp * 80 / 100
	}
	boss.MaxHP, boss.HP = hp, hp
	boss.AVNat += 3 + cyc
	boss.XP = 2200 + lvl*120 + cyc*500
	boss.Gold = 500 + lvl*40 + cyc*200
	if drop := g.bossDrop(r, lvl); drop != nil {
		boss.Inv = append(boss.Inv, drop)
	}
	boss.X, boss.Y, boss.HomeX, boss.HomeY = 64, 12, 64, 12
	zn.Actors = append(zn.Actors, boss)

	for i := 0; i < 4+mini(cyc, 4); i++ {
		x, y, ok := zn.FreeSpot(r, func(x, y int) bool { return x > 24 && (abs(x-64) > 2 || abs(y-12) > 2) })
		if !ok {
			break
		}
		mob := g.Spawn(Pick(r, arch.MobTpls), clampi(lvl-2, 8, 45), r)
		mob.Faction, mob.AI, mob.Angry = "eldritch", "hostile", true
		mob.Set("ftmob")
		mob.X, mob.Y, mob.HomeX, mob.HomeY = x, y, x, y
		zn.Actors = append(zn.Actors, mob)
	}

	g.computeLight(zn)
	g.Zones[ftArenaKey] = zn
	return zn
}

func (g *Game) ftEnterSanctum() {
	st := g.ftEnsure()
	if st.Stage < 4 || st.Defeated || g.Cur == nil {
		return
	}
	here := g.egHere()
	if here != st.AnchorP || g.Cur.Z != 0 {
		g.Msg("{{y|Travel to " + g.wsPlace(st.AnchorP) + " to enter the Sanctum.}}")
		return
	}
	if g.nearHostile() {
		g.Msg("{{R|Clear nearby enemies before crossing the threshold.}}")
		return
	}
	zn := g.ftSanctumZone()
	first := !st.SanctumOn
	st.SanctumOn = true
	g.enterZone(zn, 4, 12, true)
	g.Mode, g.OnWorld = "main", false
	if first {
		note := fmt.Sprintf("%s waits at the far end of the Sanctum.", st.BossName)
		if g.ftHasRelic() {
			note += " Your counter-relic flares, cracking its outer ward."
		}
		g.showPopup(zn.Name, note)
	}
	g.recompute()
}

// ---- New Game+ (Cycle ascension) ----

// ftNGUnlocked reports whether New Game+ can be started (defeated Final Threat or won endgame.go).
func (g *Game) ftNGUnlocked() bool {
	if g.ft().Defeated {
		return true
	}
	eg := g.eg()
	return eg.Phase == 3 && (eg.Ending == "destroy" || eg.Ending == "seal" || eg.Ending == "ally")
}

func (g *Game) ftCollectCarryCandidates() []*Item {
	var out []*Item
	seen := map[int]bool{}
	for _, bp := range g.P.Body {
		if bp != nil && bp.Item != nil && !seen[bp.Item.ID] {
			seen[bp.Item.ID] = true
			out = append(out, bp.Item)
		}
	}
	for _, it := range g.P.Inv {
		if it != nil && !seen[it.ID] {
			seen[it.ID] = true
			out = append(out, it)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Value != out[j].Value {
			return out[i].Value > out[j].Value
		}
		return out[i].ID < out[j].ID
	})
	return out
}

func (g *Game) ftStartNewGamePlus() bool {
	if !g.ftNGUnlocked() || g.P == nil || g.W == nil {
		return false
	}
	st := g.ft()
	fg := g.ftg()

	// Gather selected legacy items (up to 3; if none picked, auto-pick top 3 by value).
	cands := g.ftCollectCarryCandidates()
	var kept []*Item
	for _, it := range cands {
		if st.ngPick[it.ID] && len(kept) < 3 {
			cp := *it
			kept = append(kept, &cp)
		}
	}
	if len(kept) == 0 {
		for i := 0; i < len(cands) && i < 3; i++ {
			cp := *cands[i]
			kept = append(kept, &cp)
		}
	}

	// Record victory & Hall of Fame legend before turning the cycle.
	yr := g.W.DateOf(nil, g.Day()).Year
	fg.Victories = append(fg.Victories, ftVictory{
		Cycle:      fg.Cycle + 1,
		ThreatName: st.Name,
		BossName:   st.BossName,
		HeroName:   g.P.Name,
		Level:      g.P.Level,
		Year:       yr,
		Day:        g.Day(),
	})
	if len(fg.Victories) > 32 {
		fg.Victories = fg.Victories[len(fg.Victories)-32:]
	}
	g.hf().Legends = append(g.hf().Legends, HFLegend{
		Name:    g.P.Name,
		Species: g.P.Species,
		Calling: g.Calling,
		Year:    yr,
		Cause:   fmt.Sprintf("ascended into Cycle %d", fg.Cycle+2),
		Place:   g.W.Name,
		Level:   g.P.Level,
		Score:   g.hfScore(),
		Deeds:   g.lpSnapshotDeeds(),
	})

	fg.Cycle++
	cyc := fg.Cycle

	// Reset world state for a fresh, harder run on this world.
	g.Zones = map[string]*Zone{}
	g.Visited = map[string]bool{}
	g.Read = map[string]bool{}
	g.Slain = map[int]bool{}
	g.Taken = map[int]bool{}
	g.Known = map[int]bool{}
	g.Quests = nil
	g.ArenaWins = 0
	g.ArenaFoes = nil
	g.Turn = 0
	g.Hunger, g.Thirst = 0, 0
	g.Dead, g.DeathText = false, ""

	// Reset planet-local ft state so a new, harder threat generates for the new cycle.
	delete(g.extLive, ftKey)
	delete(g.Store, ftKey)

	// Carry over legacy items and a modest cycle boon.
	g.P.Inv = nil
	for i := range g.P.Body {
		if g.P.Body[i] != nil {
			g.P.Body[i].Item = nil
			g.P.Body[i].Severed = false
		}
	}
	// Grant +1 to the player's highest attribute (up to +4 from cycles).
	if cyc <= 4 {
		bestA := 0
		for i := 1; i < 6; i++ {
			if g.P.Attr[i] > g.P.Attr[bestA] {
				bestA = i
			}
		}
		g.P.Attr[bestA]++
	}
	g.P.Level = maxi(1, g.P.Level/2)
	g.XP = 0
	g.P.MaxHP = 14 + g.P.Level*4 + g.P.Attr[TOU]
	g.P.HP = g.P.MaxHP
	g.P.Status = map[string]int{}
	g.P.Gold = maxi(100, g.P.Gold/4)

	for _, it := range kept {
		cp := *it
		cp.ID = g.newID()
		if cp.Extra == nil {
			cp.Extra = map[string]int{}
		}
		cp.Extra["ft_ng"] = cyc
		note := fmt.Sprintf("Legacy relic carried into Cycle %d.", cyc+1)
		if cp.Lore == "" {
			cp.Lore = note
		} else if !strings.Contains(cp.Lore, "Cycle ") {
			cp.Lore = cp.Lore + " " + note
		}
		g.give(&cp)
		g.equip(&cp)
	}

	// Re-enter the starting town zone fresh.
	g.WX, g.WY = g.StartX, g.StartY
	zn := g.GetZone(g.StartX, g.StartY, 1, 1, 0)
	g.enterZone(zn, MW/2, MH/2, true)
	g.Mode, g.OnWorld = "main", false
	g.ftEnsure()
	g.recompute()
	g.deed("endgame", fmt.Sprintf("%s began Cycle %d (New Game+), carrying %d legacy items into a harder world.", g.P.Name, cyc+1, len(kept)), 0, 0)
	g.Msg(fmt.Sprintf("{{Y|Cycle %d (New Game+) begins! Foes are stronger (+%d%% HP, +%d levels). You carry %d legacy items.}}", cyc+1, cyc*35, cyc*2, len(kept)))
	return true
}

// ---- hooks: turn, zone, enter, kill, talk, side ----

var ftDoomMilestones = [...]int{25, 50, 75, 90}

func ftTurn(g *Game) {
	if g.W == nil || g.P == nil || g.Cur == nil || g.Dead || g.Turn%60 != 0 {
		return
	}
	st := g.ftEnsure()
	if st.Defeated || st.Stage == 5 {
		return
	}
	// Awaken naturally once the wanderer is seasoned or when endgame.go rises.
	if st.Stage == 0 {
		if g.P.Level >= 8 || g.Day() >= 30 || g.eg().Phase >= 1 {
			g.ftAwaken()
		}
		return
	}
	day := g.Day()
	if day <= st.LastDay {
		return
	}
	elapsed := mini(day-st.LastDay, 30)
	st.LastDay = day

	// Advance doom clock slowly; active political crises and unresolved eg threat speed it up.
	rate := 1
	if g.eg().Phase == 1 {
		rate++
	}
	if st.UnitySt > 0 && g.poCrisisAt(st.UnitySt) != nil {
		rate++
	}
	if g.ftg().Cycle > 0 {
		rate++
	}
	// Every 3 days adds `rate` doom (capped at 100).
	if day%3 == 0 || elapsed >= 3 {
		st.Doom = clampi(st.Doom+rate, 0, 100)
	}
	for _, m := range ftDoomMilestones {
		if st.Doom >= m && st.DoomMile < m {
			st.DoomMile = m
			msg := fmt.Sprintf("%s deepens (Doom %d%%): %s", st.Name, st.Doom, ftArchByID(st.Arch).Omen)
			g.ftLogf(msg)
			g.Msg("{{R|" + msg + "}}")
			if st.LoreBurg > 0 {
				lvSt(g).addNews(lvDay(g), st.LoreBurg, "%s", msg)
			}
		}
	}
}

func ftZone(g *Game, zn *Zone, _ *Rng) {
	if g.W == nil || zn == nil {
		return
	}
	// 1. New Game+ harder world scaling (only when Cycle > 0; Cycle 0 is byte-identical).
	if cyc := g.ftg().Cycle; cyc > 0 && zn.Key != ftArenaKey {
		r := NewRng(fmt.Sprintf("%s:ft:ng:%s:%d", g.Seed, zn.Key, cyc))
		for _, a := range zn.Actors {
			if a == nil || a.HP <= 0 || a.AI != "hostile" || a.Has("ft_ng_scaled") {
				continue
			}
			a.Set("ft_ng_scaled")
			a.Level += cyc * 2
			a.MaxHP = a.MaxHP * (100 + cyc*35) / 100
			a.HP = a.MaxHP
			a.AVNat += cyc
			a.XP = a.XP * (100 + cyc*25) / 100
			a.Gold = a.Gold * (100 + cyc*30) / 100
			if a.Boss && r.Chance(0.5) {
				if extra := g.Loot(r, clampi(3+cyc, 3, 6)); extra != nil {
					a.Inv = append(a.Inv, extra)
				}
			}
		}
	}

	// 2. Place Omen Stone in the Lore burg or Anchor parasang (append-only, side rng).
	st := g.ftEnsure()
	if st.Defeated || zn.Z != 0 {
		return
	}
	if zn.Kind == "town" && zn.Burg > 0 && zn.Burg == st.LoreBurg && !st.Placed["omen:"+zn.Key] {
		r := NewRng(g.Seed + ":ft:omen:" + zn.Key)
		if x, y, ok := zn.FreeSpot(r, func(x, y int) bool { return zn.At(x, y) == TCobble || zn.At(x, y) == TFloor }); ok {
			zn.Set(x, y, TInscription)
			if zn.Notes == nil {
				zn.Notes = map[int]string{}
			}
			zn.Notes[y*MW+x] = fmt.Sprintf("{{M|An omen stone carved with warnings of %s.}} %s", st.Name, st.LoreNote)
			st.Placed["omen:"+zn.Key] = true
		}
	}
	if g.W.In(zn.WX, zn.WY) && g.W.Idx(zn.WX, zn.WY) == st.PowerP && zn.Kind != "town" && st.Stage >= 2 && !st.WardPower {
		g.ftPlaceHarbinger(zn)
	}
}

func ftEnter(g *Game, zn *Zone) {
	if g.W == nil || zn == nil || g.P == nil {
		return
	}
	st := g.ftEnsure()
	if st.Defeated {
		return
	}
	if zn.Z == 0 && g.W.In(zn.WX, zn.WY) {
		p := g.W.Idx(zn.WX, zn.WY)
		if p == st.PowerP && zn.Kind != "town" && st.Stage >= 2 && !st.WardPower {
			g.ftPlaceHarbinger(zn)
		}
		if p == st.AnchorP && st.Stage >= 4 && zn.Key != ftArenaKey {
			g.Msg("{{M|The air trembles: the Sanctum of " + st.Name + " lies open here. Open Final threat & New Game+ to enter.}}")
		}
	}
}

func (g *Game) ftPlaceHarbinger(zn *Zone) {
	st := g.ft()
	for _, a := range zn.Actors {
		if a != nil && a.HP > 0 && a.Has("ft_lt") {
			return
		}
	}
	if st.Placed["lt:"+zn.Key] {
		return
	}
	arch := ftArchByID(st.Arch)
	r := NewRng(fmt.Sprintf("%s:ft:lt:%s:%d", g.Seed, zn.Key, g.ftg().Cycle))
	x, y, ok := zn.FreeSpot(r, func(x, y int) bool { return abs(x-g.P.X) > 6 || abs(y-g.P.Y) > 5 || g.Cur != zn })
	if !ok {
		return
	}
	lvl := clampi(maxi(g.P.Level+1, 10)+g.ftg().Cycle*2, 10, 45)
	a := g.Spawn(Pick(r, arch.MobTpls), lvl, r)
	a.Name, a.Proper, a.Boss = st.PowerName, true, true
	a.Glyph, a.F = arch.Glyph, CRed
	a.Faction, a.AI, a.Angry = "eldritch", "hostile", true
	a.MaxHP = a.MaxHP * (140 + g.ftg().Cycle*30) / 100
	a.HP = a.MaxHP
	a.Set("ft_lt")
	a.Desc = fmt.Sprintf("A lieutenant of %s, guarding the Ward of Power.", st.Name)
	a.X, a.Y, a.HomeX, a.HomeY = x, y, x, y
	zn.Actors = append(zn.Actors, a)
	st.Placed["lt:"+zn.Key] = true
}

func ftKill(g *Game, d, src *Actor) {
	if g.W == nil || d == nil {
		return
	}
	st := g.ft()
	if d.Has("ft_lt") && !st.WardPower {
		st.WardPower = true
		st.Doom = maxi(0, st.Doom-10)
		g.gainXP(120 + g.P.Level*15)
		g.ftLogf("Slew " + d.Name + " and claimed the Ward of Power.")
		g.Msg("{{G|You have slain " + d.Name + " and claimed the Ward of Power!}}")
		g.deed("slay", fmt.Sprintf("%s slew %s, herald of %s, taking the Ward of Power.", g.P.Name, d.Name, st.Name), 0, 0)
		g.ftAdvanceIfReady()
	}
	if d.Has("ftboss") && !st.Defeated {
		st.Defeated = true
		st.Stage = 5
		st.Doom = 0
		xp := 1800 + g.P.Level*100 + g.ftg().Cycle*400
		g.gainXP(xp)
		for i := range g.W.States {
			if g.W.States[i].Alive {
				g.AddRep(stateFaction(g.W.States[i].ID), 60, " for ending "+st.Name)
			}
		}
		arch := ftArchByID(st.Arch)
		trophy := &Item{
			ID:     g.newID(),
			Name:   "Crown-Shard of " + strings.TrimPrefix(st.BossName, "The "),
			Glyph:  '*',
			F:      CMagenta,
			Kind:   "artifact",
			Slot:   "head",
			AV:     3 + g.ftg().Cycle,
			DV:     2 + g.ftg().Cycle,
			Value:  1800 + g.ftg().Cycle*500,
			Weight: 1,
			Tier:   6,
			Rarity: "unique",
			Ident:  true,
			Extra:  map[string]int{"ft_trophy": 1, "ft_cycle": g.ftg().Cycle},
			Lore:   fmt.Sprintf("Taken from %s when %s ended %s.", st.BossName, g.P.Name, st.Name),
		}
		g.give(trophy)
		_ = arch
		g.ftLogf("Defeated " + st.BossName + "! " + st.Name + " is broken. New Game+ is unlocked.")
		g.deed("endgame", fmt.Sprintf("%s defeated %s and ended %s.", g.P.Name, st.BossName, st.Name), 0, 0)
		g.showPopup("Victory: "+st.Name+" Ended", fmt.Sprintf(
			"You have broken %s in its Sanctum and claimed %s.\n\nThe world breathes again. You may keep walking these roads, or open Final threat & New Game+ to begin Cycle %d with harder foes and up to 3 kept legacy items.",
			st.BossName, trophy.Name, g.ftg().Cycle+2,
		))
	}
}

func ftTalk(g *Game, a *Actor) []Option {
	if g.W == nil || g.P == nil || a == nil || a.HP <= 0 || a.AI == "follower" || g.Hostile(a) || g.Cur == nil {
		return nil
	}
	st := g.ftEnsure()
	if st.Defeated {
		return nil
	}
	var out []Option
	isScholar := a.Role == "scholar" || a.Role == "priest" || a.Role == "elder" || a.Role == "notable" || a.Role == "chronicler"

	// Stage 0 or 1: investigate the omens.
	if st.Stage <= 1 && isScholar {
		out = append(out, Option{
			Text: "{{M|Ask about the omens of " + shorten(st.Name, 26) + ".}}",
			Do: func() {
				if st.Stage == 0 {
					g.ftAwaken()
				}
				if st.Stage == 1 {
					st.Stage = 2
					g.ftLogf("Learned the three wards needed to break " + st.Name + ".")
					g.gainXP(60)
				}
				g.talkMenu(fmt.Sprintf(
					"\"%s\"\n\n\"To break %s you need three wards: the Ward of Lore in %s, the Ward of Power held by %s (%s), and the Ward of Unity from %s.\"",
					st.LoreNote, st.Name, g.dlBurgName(st.LoreBurg), st.PowerName, g.wsDirDist(g.egHere(), st.PowerP), g.realmName(st.UnitySt),
				))
			},
		})
	}

	// Stage 2: Ward of Lore in LoreBurg.
	if st.Stage == 2 && !st.WardLore && g.Cur.Burg > 0 && g.Cur.Burg == st.LoreBurg && isScholar {
		out = append(out, Option{
			Text: "{{Y|Request the Ward of Lore against " + shorten(st.Name, 24) + ".}}",
			Do: func() {
				fee := 120 + g.ftg().Cycle*60
				if g.P.Score(INT) >= 16 || g.Skills["lore"] >= 1 {
					st.WardLore = true
					st.Doom = maxi(0, st.Doom-8)
					g.gainXP(80)
					g.ftLogf("Secured the Ward of Lore in " + g.dlBurgName(st.LoreBurg) + ".")
					g.talkMenu("\"You read the old ciphers as well as any of us. Take this Ward of Lore; may it hold.\"")
					g.ftAdvanceIfReady()
					return
				}
				if g.cuSpend(fee) {
					st.WardLore = true
					st.Doom = maxi(0, st.Doom-8)
					g.gainXP(80)
					g.ftLogf("Purchased the Ward of Lore in " + g.dlBurgName(st.LoreBurg) + ".")
					g.talkMenu(fmt.Sprintf("\"This covers the ink and the vault seals (%s). Take the Ward of Lore.\"", g.cuFormat(fee)))
					g.ftAdvanceIfReady()
					return
				}
				g.talkMenu(fmt.Sprintf("\"Decoding the cipher takes rare inks (%s), unless you know the lore yourself (INT 16+ or Lore skill).\"", g.cuFormat(fee)))
			},
		})
	}

	// Stage 2: Ward of Unity from UnitySt realm.
	if st.Stage == 2 && !st.WardUnity && g.Cur.Burg > 0 && (a.Role == "elder" || a.Role == "notable" || a.Title == "ruler") {
		if bd := g.W.burgByID[g.Cur.Burg]; bd != nil && bd.State == st.UnitySt {
			out = append(out, Option{
				Text: "{{G|Pledge " + g.realmName(st.UnitySt) + " to grant the Ward of Unity.}}",
				Do: func() {
					rep := g.Rep[stateFaction(st.UnitySt)]
					allied := g.egAllied(st.UnitySt) || g.po().Stand[st.UnitySt] >= 20
					fee := 180 + g.ftg().Cycle*80
					if rep >= 80 || allied {
						st.WardUnity = true
						st.Doom = maxi(0, st.Doom-8)
						g.gainXP(80)
						g.ftLogf("Secured the Ward of Unity from " + g.realmName(st.UnitySt) + ".")
						g.talkMenu(fmt.Sprintf("\"%s stands with you, wanderer. Take the Ward of Unity.\"", g.realmName(st.UnitySt)))
						g.ftAdvanceIfReady()
						return
					}
					if g.cuSpend(fee) {
						st.WardUnity = true
						st.Doom = maxi(0, st.Doom-8)
						g.gainXP(80)
						g.ftLogf("Secured the Ward of Unity from " + g.realmName(st.UnitySt) + " with tribute.")
						g.talkMenu(fmt.Sprintf("\"Your coin equips our watch (%s). %s grants you the Ward of Unity.\"", g.cuFormat(fee), g.realmName(st.UnitySt)))
						g.ftAdvanceIfReady()
						return
					}
					g.talkMenu(fmt.Sprintf("\"Earn our trust (rep 80+, court standing, or alliance) or pay %s to muster our seal.\"", g.cuFormat(fee)))
				},
			})
		}
	}
	return out
}

func ftSide(g *Game) []string {
	var out []string
	if cyc := g.ftg().Cycle; cyc > 0 {
		out = append(out, fmt.Sprintf("{{M|NG+ Cycle %d}}", cyc+1))
	}
	st := g.ft()
	if st.Inited && st.Stage > 0 && !st.Defeated {
		col := "y"
		if st.Doom >= 75 {
			col = "R"
		}
		out = append(out, fmt.Sprintf("{{%s|Doom: %d%% (Stg %d/4)}}", col, st.Doom, st.Stage))
	}
	return out
}

// ---- screen: mode "ft" ----

func ftKeys(g *Game, k string) {
	st := g.ftEnsure()
	switch k {
	case "Escape", "q":
		st.confirm = false
		g.Mode = "main"
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
	case " ":
		if st.tab == 2 && g.ftNGUnlocked() {
			cands := g.ftCollectCarryCandidates()
			if st.sel >= 0 && st.sel < len(cands) {
				id := cands[st.sel].ID
				if st.ngPick[id] {
					delete(st.ngPick, id)
				} else if len(st.ngPick) < 3 {
					st.ngPick[id] = true
				} else {
					g.Msg("{{y|At most 3 legacy items can be carried into New Game+.}}")
				}
			}
		}
	case "Enter":
		switch st.tab {
		case 0:
			switch st.Stage {
			case 0:
				g.ftAwaken()
			case 3:
				g.ftForgeRelic()
			case 4:
				g.ftEnterSanctum()
			}
		case 2:
			if !g.ftNGUnlocked() {
				g.Msg("{{K|Defeat the Final Threat (or win the Endgame War) to unlock New Game+.}}")
				return
			}
			if !st.confirm {
				st.confirm = true
				return
			}
			g.ftStartNewGamePlus()
		}
	}
}

func (g *Game) ftDraw() {
	st := g.ftEnsure()
	fg := g.ftg()
	s := g.sc
	x, y0, w, h := g.panel("Final threat & New Game+")
	y := y0

	tabs := []string{"1:Final Threat", "2:Quest Chain", "3:New Game+"}
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
	here := g.egHere()

	switch st.tab {
	case 0:
		cycLine := fmt.Sprintf("{{W|Cycle %d}}", fg.Cycle+1)
		if fg.Cycle > 0 {
			cycLine += fmt.Sprintf(" {{M|(Harder world: +%d%% foe HP, +%d foe levels)}}", fg.Cycle*35, fg.Cycle*2)
		}
		s.TextB(x, y, cycLine, CGray, CBgDark, x+w)
		y += 2

		if st.Stage == 0 {
			for _, line := range Wrap("The world is quiet for now, though faint signs gather. Reach level 8 or day 30, talk to a scholar/elder about the omens, or press Enter here to wake the omen early.", w) {
				s.TextB(x, y, "{{K|"+line+"}}", CGray, CBgDark, x+w)
				y++
			}
			y++
			s.TextB(x, y, "{{Y|[Enter] Uncover the looming threat now}}", CGray, CBgDark, x+w)
		} else {
			col := "R"
			if st.Defeated {
				col = "G"
			}
			s.TextB(x, y, fmt.Sprintf("{{%s|%s}}  Harbinger: {{M|%s}}", col, st.Name, st.BossName), CGray, CBgDark, x+w)
			y++
			s.TextB(x, y, fmt.Sprintf("Doom clock: {{R|%d%%}}   Anchor: {{Y|%s}} {{K|(%s)}}", st.Doom, g.wsPlace(st.AnchorP), g.wsDirDist(here, st.AnchorP)), CGray, CBgDark, x+w)
			y += 2
			for _, line := range Wrap(st.LoreNote, w) {
				if y >= maxY-4 {
					break
				}
				s.TextB(x, y, line, CGray, CBgDark, x+w)
				y++
			}
			y++
			if st.Defeated {
				s.TextB(x, y, "{{G|The Final Threat is broken. Open tab 3 (New Game+) to begin a harder cycle.}}", CGray, CBgDark, x+w)
			} else if st.Stage == 3 {
				s.TextB(x, y, "{{Y|[Enter] Forge "+ftArchByID(st.Arch).Relic+" from the three wards}}", CGray, CBgDark, x+w)
			} else if st.Stage == 4 {
				if here == st.AnchorP && g.Cur != nil && g.Cur.Z == 0 {
					s.TextB(x, y, "{{R|[Enter] Cross into the Sanctum and face "+st.BossName+"}}", CGray, CBgDark, x+w)
				} else {
					s.TextB(x, y, "{{y|Travel to "+g.wsPlace(st.AnchorP)+" ("+g.wsDirDist(here, st.AnchorP)+") to enter the Sanctum.}}", CGray, CBgDark, x+w)
				}
			} else {
				s.TextB(x, y, "{{c|See tab 2 (Quest Chain) for your active objectives.}}", CGray, CBgDark, x+w)
			}
		}

	case 1:
		mark := func(done, cur bool, txt string) string {
			switch {
			case done:
				return "{{G|[x] " + txt + "}}"
			case cur:
				return "{{Y|[>] " + txt + "}}"
			}
			return "{{K|[ ] " + txt + "}}"
		}
		steps := []string{
			mark(st.Stage >= 2, st.Stage == 1, "Read the omens: speak with a scholar, priest, or elder in any town"),
			mark(st.WardLore, st.Stage == 2 && !st.WardLore, fmt.Sprintf("Ward of Lore: visit %s and ask a scholar/elder", g.dlBurgName(st.LoreBurg))),
			mark(st.WardPower, st.Stage == 2 && !st.WardPower, fmt.Sprintf("Ward of Power: slay %s near %s (%s)", st.PowerName, g.wsPlace(st.PowerP), g.wsDirDist(here, st.PowerP))),
			mark(st.WardUnity, st.Stage == 2 && !st.WardUnity, fmt.Sprintf("Ward of Unity: secure the seal of %s from an elder or ruler", g.realmName(st.UnitySt))),
			mark(st.Forged, st.Stage == 3, "Forge "+ftArchByID(st.Arch).Relic+" (on tab 1 once all 3 wards are held)"),
			mark(st.Defeated, st.Stage == 4, fmt.Sprintf("Enter the Sanctum at %s (%s) and defeat %s", g.wsPlace(st.AnchorP), g.wsDirDist(here, st.AnchorP), st.BossName)),
		}
		for _, line := range steps {
			for _, wl := range Wrap(line, w) {
				if y >= maxY-6 {
					break
				}
				s.TextB(x, y, wl, CGray, CBgDark, x+w)
				y++
			}
		}
		y++
		if y < maxY-1 {
			s.TextB(x, y, "{{W|Chronicle:}}", CGray, CBgDark, x+w)
			y++
			for i := len(st.Log) - 1; i >= 0 && y < maxY; i-- {
				s.TextB(x+1, y, "{{K|- "+shorten(st.Log[i], w-4)+"}}", CGray, CBgDark, x+w)
				y++
			}
		}

	case 2:
		s.TextB(x, y, fmt.Sprintf("{{W|Current Cycle: %d}}   Past victories: %d", fg.Cycle+1, len(fg.Victories)), CGray, CBgDark, x+w)
		y++
		if !g.ftNGUnlocked() {
			y++
			s.TextB(x, y, "{{K|New Game+ unlocks once you defeat the Final Threat (or win the Endgame War).}}", CGray, CBgDark, x+w)
			y += 2
			s.TextB(x, y, "{{c|In New Game+ you start a fresh age on this world with:}}", CGray, CBgDark, x+w)
			y++
			s.TextB(x+2, y, "- Stronger enemies (+35% HP and +2 levels per cycle)", CGray, CBgDark, x+w)
			y++
			s.TextB(x+2, y, "- Up to 3 kept legacy items carried into the new age", CGray, CBgDark, x+w)
			y++
			s.TextB(x+2, y, "- A new world-ending threat and +1 to your highest attribute", CGray, CBgDark, x+w)
		} else {
			s.TextB(x, y, fmt.Sprintf("{{G|New Game+ unlocked!}} Select up to 3 legacy items with Space (%d/3 chosen):", len(st.ngPick)), CGray, CBgDark, x+w)
			y += 2
			cands := g.ftCollectCarryCandidates()
			if st.sel >= len(cands) && len(cands) > 0 {
				st.sel = len(cands) - 1
			}
			for i, it := range cands {
				if y >= maxY-3 {
					break
				}
				bg := CBgDark
				if i == st.sel {
					bg = CBgSel
				}
				box := "[ ]"
				if st.ngPick[it.ID] {
					box = "{{Y|[x]}}"
				}
				s.Fill(x, y, w, 1, bg)
				s.TextB(x, y, fmt.Sprintf("%s %s {{K|(val %d)}}", box, shorten(it.Display(), 42), it.Value), CGray, bg, x+w)
				y++
			}
			y++
			if st.confirm {
				s.TextB(x, y, fmt.Sprintf("{{R|Press Enter again to begin Cycle %d (New Game+)!}}", fg.Cycle+2), CGray, CBgDark, x+w)
			} else {
				s.TextB(x, y, fmt.Sprintf("{{Y|[Enter] Begin Cycle %d (New Game+) with chosen legacy items}}", fg.Cycle+2), CGray, CBgDark, x+w)
			}
		}
	}

	s.TextB(x, y0+h-1, "{{K|Tab/1-3 switch tab  Up/Down select  Space toggle item  Enter act  Esc close}}", CGray, CBgDark, x+w)
}
