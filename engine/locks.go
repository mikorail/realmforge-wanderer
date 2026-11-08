package engine

import (
	"fmt"
	"sort"
	"strings"
)

// Locks and chests (prefix lk). Lockpicking (lock mode "lk": feel pins against a risk meter),
// picks (Kind "lockpick": Power = quality, Charges = wear), chests placed by a zone hook, traps,
// magic locks (riddle, blood, password, ally, sun, moon), keys with stories (dead ruler's signet,
// guildmaster's ring, a key split in three parts), bait vaults and a locksmith NPC.
// Saved state in g.ext("lk"), planet-local. Chests reuse TCrate (see lkChestTile).
// Zone hook uses side rng seed:lk:zoneKey only.

// lkChestTile is the terrain drawn for our chests. Swap to TChest if the coordinator adds one.
var lkChestTile = TCrate

type lkLock struct {
	ID      string   `json:"id"`
	Kind    string   `json:"k"` // chest door
	Name    string   `json:"n"`
	X       int      `json:"x"`
	Y       int      `json:"y"`
	Tier    int      `json:"t"`
	Lvl     int      `json:"lv,omitempty"`
	Magic   string   `json:"m,omitempty"`
	Arg     string   `json:"a,omitempty"`
	Trap    string   `json:"tr,omitempty"`
	Found   bool     `json:"fd,omitempty"`
	Off     bool     `json:"of,omitempty"`
	Passive bool     `json:"ps,omitempty"`
	Key     int      `json:"ky,omitempty"`
	KeyHint string   `json:"kh,omitempty"`
	Special string   `json:"sp,omitempty"` // royal guild split
	Jam     int      `json:"j,omitempty"`  // jammed until this game minute
	Open    bool     `json:"o,omitempty"`
	Broken  bool     `json:"b,omitempty"`
	Wrong   []string `json:"w,omitempty"`
	Strikes int      `json:"s,omitempty"`
	Ponder  bool     `json:"pd,omitempty"`
}

type lkBait struct {
	DX, DY         int
	X0, Y0, X1, Y1 int
	Armed          bool
	Warned         bool
	Mode           string // ambush gas
}

type lkSplit struct {
	Name   string   `json:"n"`
	ID     int      `json:"id"`
	Placed int      `json:"p"`
	Zones  []string `json:"z"`
	Vault  string   `json:"v"`
}

type lkStudy struct {
	Label string `json:"l"`
	Tier  int    `json:"t"`
}

type lkSess struct {
	L          *lkLock
	pick       *Item
	pins, set  int
	risk, feel int
}

type lkState struct {
	Locks     map[string]*lkLock `json:"l"`
	Rank      int                `json:"r"`
	Picked    int                `json:"pk"`
	Firsts    map[string]bool    `json:"f"`
	Learned   map[string]bool    `json:"pw"`
	Heard     []string           `json:"hd"`
	Studied   map[string]lkStudy `json:"st"`
	Wants     map[string][]int   `json:"wt"`
	Delivered map[string]bool    `json:"dv"`
	Split     *lkSplit           `json:"sx"`
	Baits     map[string]*lkBait `json:"bt"`

	s     *lkSess
	title string
}

func (g *Game) lkSt() *lkState {
	st := g.ext("lk", func() any { return &lkState{} }).(*lkState)
	if st.Locks == nil {
		st.Locks = map[string]*lkLock{}
	}
	if st.Firsts == nil {
		st.Firsts = map[string]bool{}
	}
	if st.Learned == nil {
		st.Learned = map[string]bool{}
	}
	if st.Studied == nil {
		st.Studied = map[string]lkStudy{}
	}
	if st.Wants == nil {
		st.Wants = map[string][]int{}
	}
	if st.Delivered == nil {
		st.Delivered = map[string]bool{}
	}
	if st.Baits == nil {
		st.Baits = map[string]*lkBait{}
	}
	return st
}

func init() {
	unLocalKeys = append(unLocalKeys, "lk")
	zoneHooks = append(zoneHooks, lkZone)
	talkHooks = append(talkHooks, lkTalk)
	turnHooks = append(turnHooks, lkTurn)
	modeKeys["lk"] = func(g *Game, k string) { g.keyMenu(k, func() { g.lkClose() }) }
	modeDraws["lk"] = func(g *Game) { g.lkDraw() }
}

func lkH(s string) int { return int(HashStr(s)%900000) + 1 }

func lkID(zkey string, x, y int) string { return fmt.Sprintf("%s:%d", zkey, y*MW+x) }

// ---- menu plumbing ----

func (g *Game) lkMenu(title, text string, opts []Option) {
	st := g.lkSt()
	st.title = title
	g.Popup = Wrap(text, 100)
	g.Opts = opts
	g.Sel = 0
	g.Mode = "lk"
}

func (g *Game) lkClose() {
	g.Mode, g.Talk, g.Popup, g.Opts = "main", nil, nil, nil
}

func (g *Game) lkDraw() {
	s := g.sc
	h := len(g.Popup) + len(g.Opts) + 4
	y0 := maxi(1, (MH-h)/2)
	s.Box(2, y0, SW-4, mini(h, SH-2), CBrown, g.lkSt().title)
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

func (g *Game) lkFirst(id, text string) {
	st := g.lkSt()
	if st.Firsts[id] {
		return
	}
	st.Firsts[id] = true
	g.deed("lock", fmt.Sprintf("The wanderer %s %s.", g.P.Name, text), 0, 0)
}

// ---- items ----

func (g *Game) lkMakePick(q int, improvised bool) *Item {
	q = clampi(q, 1, 5)
	it := &Item{ID: g.newID(), Kind: "lockpick", Name: lkPickName[q], Glyph: '~', F: CGray, Power: q, Charges: 6 + 2*q, Value: lkPickPrice[q] / 2, Weight: 1, Ident: true}
	if improvised {
		it.Name, it.Charges, it.Value = "improvised pick", 3, 3
		it.Lore = "Bits of scrap bent into a pick. It will not last."
	} else {
		it.Lore = fmt.Sprintf("A quality %d lockpick.", q)
	}
	return it
}

func (g *Game) lkPicks() []*Item {
	var out []*Item
	for _, it := range g.P.Inv {
		if it.Kind == "lockpick" && it.Charges > 0 {
			out = append(out, it)
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Power > out[j].Power })
	return out
}

func (g *Game) lkMakeKey(name, lore string, id int, col byte) *Item {
	return &Item{ID: g.newID(), Kind: "key", Name: name, Glyph: '¬', F: col, Extra: map[string]int{"lkkey": id}, Value: 60, Lore: lore, Ident: true}
}

func (g *Game) lkOrigKey() int { return int(HashStr(g.Cur.Key) % 1000000) }

func (g *Game) lkKeysFor(L *lkLock) []*Item {
	var out []*Item
	for _, it := range g.P.Inv {
		if it.Kind != "key" || it.Extra == nil {
			continue
		}
		if (L.Key != 0 && it.Extra["lkkey"] == L.Key) || it.Extra["lkcut"] == lkH(L.ID) || (L.Kind == "door" && L.Special == "" && it.Extra["key"] == g.lkOrigKey() && it.Extra["key"] != 0) {
			out = append(out, it)
		}
	}
	return out
}

// ---- lock lookup / creation ----

// lkRollTier: lock strength from depth with a little variance.
func lkRollTier(r *Rng, lvl, bonus int) int {
	return clampi(1+r.Int(0, 1+lvl/4)+bonus, 1, 5)
}

// lkRollMagicTrap fills in magic and trap on a fresh lock using the side rng.
func lkRollMagicTrap(r *Rng, L *lkLock, magicP, trapP float64) {
	if r.Chance(magicP) {
		L.Magic = Pick(r, lkMagicList)
		if L.Magic == "password" {
			L.Arg = Pick(r, lkWords)
		}
	}
	if r.Chance(trapP) {
		L.Trap = Pick(r, lkTrapDefs).ID
	}
}

// lkDoorLock returns (creating lazily) the lock of a TLocked tile in the current zone.
func (g *Game) lkDoorLock(x, y int) *lkLock {
	st := g.lkSt()
	id := lkID(g.Cur.Key, x, y)
	if L := st.Locks[id]; L != nil {
		return L
	}
	r := NewRng(g.Seed + ":lk:door:" + id)
	L := &lkLock{ID: id, Kind: "door", Name: "locked door", X: x, Y: y, Lvl: g.Cur.Level}
	L.Tier = clampi(2+g.Cur.Level/8+r.Int(0, 1), 2, 5)
	if r.Chance(0.10) {
		L.Trap = Pick(r, lkTrapDefs[:4]).ID
	}
	st.Locks[id] = L
	return L
}

// lkBumpLocked handles bumping a TLocked tile. True when it handled the bump; false leaves
// dungeon.go's own unlock() (iron key / Tinkering) to run.
func (g *Game) lkBumpLocked(x, y int) bool {
	if g.Cur == nil || g.Cur.At(x, y) != TLocked {
		return false
	}
	L := g.lkDoorLock(x, y)
	special := L.Magic != "" || (L.Trap != "" && !L.Off) || L.Jam > g.Turn || L.Special != ""
	if !special && len(g.lkPicks()) == 0 {
		for _, it := range g.P.Inv {
			if it.Kind == "key" && it.Extra != nil && it.Extra["lkcut"] == lkH(L.ID) {
				special = true
			}
		}
	}
	if !special && len(g.lkPicks()) == 0 {
		return false
	}
	g.lkBegin(L)
	return true
}

// lkBumpChest handles bumping one of our chests.
func (g *Game) lkBumpChest(x, y int) bool {
	if g.Cur == nil {
		return false
	}
	L := g.lkSt().Locks[lkID(g.Cur.Key, x, y)]
	if L == nil || L.Kind != "chest" {
		return false
	}
	if L.Open {
		g.Msg("The " + L.Name + " is empty.")
		return true
	}
	g.lkBegin(L)
	return true
}

func (g *Game) lkBegin(L *lkLock) {
	st := g.lkSt()
	st.s = &lkSess{L: L}
	if L.Magic == "password" && !st.Learned[L.ID] {
		for _, it := range g.P.Inv {
			if it.Kind == "book" && it.Extra != nil && it.Extra["lkpw"] == lkH(L.ID) {
				st.Learned[L.ID] = true
				g.Msg("{{c|Your " + it.Name + " names the word: \"" + L.Arg + "\".}}")
			}
		}
		if !st.Learned[L.ID] {
			known := false
			for _, h := range st.Heard {
				known = known || h == L.ID
			}
			if !known {
				st.Heard = append(st.Heard, L.ID)
			}
		}
	}
	if !L.Passive {
		L.Passive = true
		if L.Trap != "" && !L.Found && !L.Off {
			if g.R.Dice(2, 6)+g.checkMod("perception")-2 >= 7+L.Tier {
				L.Found = true
				g.Msg("{{y|You notice " + lkTrapByID(L.Trap).Sign + ".}}")
			}
		}
	}
	g.lkOverview("")
}

// ---- overview ----

func (g *Game) lkOverview(note string) {
	st := g.lkSt()
	s := st.s
	if s == nil || g.Dead {
		return
	}
	L := s.L
	lines := []string{fmt.Sprintf("{{Y|%s}}  {{K|(%s, tier %d)}}", L.Name, lkTierName[L.Tier], L.Tier), lkTierBlurb[L.Tier]}
	if L.Magic != "" {
		lines = append(lines, "{{M|"+lkMagicText[L.Magic]+"}}")
	}
	switch {
	case L.Trap != "" && L.Off:
		lines = append(lines, "{{G|The trap has been disarmed.}}")
	case L.Trap != "" && L.Found:
		lines = append(lines, "{{R|Trapped: "+lkTrapByID(L.Trap).Name+" ("+lkTrapByID(L.Trap).Sign+").}}")
	}
	if L.KeyHint != "" {
		lines = append(lines, "Sealed to "+L.KeyHint+".")
	}
	if L.Jam > g.Turn {
		lines = append(lines, "{{R|The lock is jammed.}}")
	}
	if note != "" {
		lines = append(lines, "", note)
	}
	var opts []Option
	add := func(t string, f func()) { opts = append(opts, Option{t, f}) }
	for _, k := range g.lkKeysFor(L) {
		k := k
		add("{{G|Use the "+k.Name+".}}", func() { g.lkUseKey(k) })
	}
	if L.Magic != "" {
		g.lkMagicOptions(L, add)
	} else if L.Jam <= g.Turn {
		if ps := g.lkPicks(); len(ps) > 0 {
			add("Pick the lock.", func() { g.lkChoosePick() })
		}
	}
	if L.Magic == "" && g.craftSt().Bits["1"] >= 2 && L.Jam <= g.Turn {
		add("{{y|Bend an improvised pick from two bits.}}", func() {
			g.craftSt().Bits["1"] -= 2
			g.give(g.lkMakePick(1, true))
			g.Msg("You bend a rough pick from scrap.")
			g.spend(3000)
			g.lkOverview("")
		})
	}
	if L.Trap != "" && !L.Found && !L.Off {
		add("{{c|Search for traps.}}", func() { g.lkSearch() })
	}
	if L.Trap != "" && L.Found && !L.Off {
		add("{{c|Disarm the trap.}} "+g.checkLabel("tinkering", 7+L.Tier), func() { g.lkDisarm() })
	}
	if L.Jam > g.Turn {
		add("Clear the jam. "+g.checkLabel("tinkering", 10), func() { g.lkUnjam() })
	}
	add(fmt.Sprintf("Force it open (noisy, risks the contents). {{c|[Strength %d%%]}}", int(chanceFor(g.P.M(STR), g.lkBashDC(L))*100+0.5)), func() { g.lkBash() })
	if L.Magic == "" {
		if _, ok := st.Studied[L.ID]; !ok {
			add("Take a wax impression for a locksmith.", func() {
				st.Studied[L.ID] = lkStudy{Label: L.Name + " in " + g.Cur.Name, Tier: L.Tier}
				g.Msg("You press the lock into wax. A locksmith could cut a key from it.")
				g.spend(2000)
				g.lkOverview("")
			})
		}
	}
	if sp := st.Split; sp != nil && L.Special == "split" && g.lkHasParts() {
		add("{{G|Fit the three key parts together.}}", func() { g.lkJoinParts(); g.lkOverview("") })
	}
	add("Leave it.", func() { g.lkClose() })
	g.lkMenu(L.Name, strings.Join(lines, "\n"), opts)
}

func (g *Game) lkUseKey(k *Item) {
	L := g.lkSt().s.L
	if k.Extra["lkcut"] != 0 || (L.Kind == "door" && k.Extra["key"] != 0) {
		g.removeInv(k)
	}
	g.Msg("{{Y|The " + k.Name + " turns.}}")
	if L.Magic != "" {
		g.Msg("{{M|The ward still holds.}}")
		g.lkOverview("The key turns, but the ward must be answered too.")
		return
	}
	g.spend(1000)
	g.lkSucceed("key")
}

// ---- picking ----

func (g *Game) lkChoosePick() {
	ps := g.lkPicks()
	if len(ps) == 1 {
		g.lkPickStart(ps[0])
		return
	}
	var opts []Option
	for _, p := range ps {
		p := p
		opts = append(opts, Option{fmt.Sprintf("%s {{K|(quality %d, wear %d)}}", p.Name, p.Power, p.Charges), func() { g.lkPickStart(p) }})
	}
	opts = append(opts, Option{"Back.", func() { g.lkOverview("") }})
	g.lkMenu("Choose a pick", "Better picks help on harder locks; a rough one wears out fast.", opts)
}

func (g *Game) lkMod(p *Item, L *lkLock) int {
	st := g.lkSt()
	m := maxi(g.checkMod("streetwise"), g.checkMod("tinkering"))
	if g.Calling == "rogue" {
		m += 2
	}
	m += 2*st.Rank + g.uwGuildBonus()
	if p != nil {
		m += clampi(p.Power-L.Tier, -3, 2)
	}
	return m
}

func (g *Game) lkPickStart(p *Item) {
	s := g.lkSt().s
	s.pick, s.pins, s.set, s.risk, s.feel = p, s.L.Tier+1, 0, 0, 0
	g.lkPickScreen("You slide the pick in and feel for the pins.")
}

func (g *Game) lkPickScreen(note string) {
	s := g.lkSt().s
	L := s.L
	dc := 5 + 2*L.Tier - s.feel
	mod := g.lkMod(s.pick, L)
	pins := ""
	for i := 0; i < s.pins; i++ {
		if i < s.set {
			pins += "{{G|[#]}} "
		} else {
			pins += "{{K|[ ]}} "
		}
	}
	bar := strings.Repeat("#", s.risk) + strings.Repeat(".", maxi(0, 10-s.risk))
	col := "G"
	if s.risk >= 7 {
		col = "R"
	} else if s.risk >= 4 {
		col = "O"
	}
	text := fmt.Sprintf("Pins  %s\nRisk  {{%s|[%s]}}  {{K|The lock fights back at 10.}}\n%s  {{K|wear %d}}", pins, col, bar, s.pick.Name, s.pick.Charges)
	if note != "" {
		text += "\n\n" + note
	}
	pct := func(d int) int { return int(chanceFor(mod, d)*100 + 0.5) }
	opts := []Option{
		{fmt.Sprintf("Set the next pin carefully. {{c|[%d%%]}}", pct(dc)), func() { g.lkStep("careful") }},
		{fmt.Sprintf("Rake the pins, fast and sloppy. {{c|[%d%%]}}", pct(dc+3)), func() { g.lkStep("rake") }},
		{"Feel the pins (next try is easier).", func() { g.lkStep("feel") }},
		{"Ease off and breathe (risk down).", func() { g.lkStep("ease") }},
		{"Give up.", func() { g.lkOverview("") }},
	}
	g.lkMenu("Picking: "+L.Name, text, opts)
}

func (g *Game) lkWear(p float64) bool {
	s := g.lkSt().s
	if !g.R.Chance(p) {
		return false
	}
	s.pick.Charges--
	if s.pick.Charges <= 0 {
		g.removeInv(s.pick)
		g.Msg("{{R|Your " + s.pick.Name + " snaps!}}")
		return true
	}
	return false
}

func (g *Game) lkStep(act string) {
	s := g.lkSt().s
	L := s.L
	hp := g.P.HP
	cost := 1000
	if act == "ease" {
		cost = 2000
	}
	g.spend(cost)
	if g.Dead || g.P.HP < hp {
		g.lkClose()
		g.Msg("You are interrupted!")
		return
	}
	mod := g.lkMod(s.pick, L)
	note := ""
	snapped := false
	switch act {
	case "feel":
		s.feel = 2
		note = "{{c|You feel the pin bind. The next try will be easier.}}"
	case "ease":
		s.risk = maxi(0, s.risk-2)
		note = "You ease the tension off."
	case "careful", "rake":
		dc := 5 + 2*L.Tier - s.feel
		wearP, fail := 0.15, 2
		if act == "rake" {
			dc += 3
			wearP, fail = 0.35, 3
		}
		s.feel = 0
		roll := g.R.Dice(2, 6) + mod
		if roll >= dc {
			n := 1
			if act == "rake" && roll-dc >= 2 {
				n = 2
			}
			s.set = mini(s.pins, s.set+n)
			note = "{{G|Click.}}"
			if n == 2 {
				note = "{{G|Click-click.}}"
			}
		} else {
			s.risk += fail
			note = "{{R|The pin slips back.}}"
			wearP += 0.25
		}
		snapped = g.lkWear(wearP)
	}
	if g.Dead {
		return
	}
	if s.pick.Charges <= 0 && snapped {
		g.lkOverview("The pick broke in the lock.")
		return
	}
	if s.set >= s.pins {
		g.Msg("{{G|The last pin sets. The lock gives.}}")
		g.lkSt().Picked++
		g.lkFirst("pick", "picked a lock for the first time")
		g.lkSucceed("pick")
		return
	}
	if s.risk >= 10 {
		g.lkPickFail()
		return
	}
	g.lkPickScreen(note)
}

func (g *Game) lkPickFail() {
	s := g.lkSt().s
	L := s.L
	x := g.R.Int(1, 100) + L.Tier*3
	g.lkWitness(L, false)
	switch {
	case x <= 40:
		g.removeInv(s.pick)
		g.Msg("{{R|The pick snaps in the lock!}}")
		g.lkOverview("The pick snapped.")
	case x <= 70:
		g.Msg("{{R|Something rattles loudly inside the lock!}}")
		g.lkAlarm(false)
		if g.Dead {
			return
		}
		s.risk = 6
		g.lkPickScreen("The noise carries. Hurry, or give up.")
	default:
		L.Jam = g.Turn + 120
		g.Msg("{{R|The lock jams.}}")
		g.lkOverview("The lock jams solid.")
	}
}

// ---- traps ----

func (g *Game) lkSearch() {
	L := g.lkSt().s.L
	g.spend(2000)
	if g.R.Dice(2, 6)+g.checkMod("perception")+3 >= 6+L.Tier {
		if L.Trap != "" {
			L.Found = true
			g.Msg("{{y|You spot " + lkTrapByID(L.Trap).Sign + ".}}")
			g.lkOverview("You find a trap.")
			return
		}
	}
	g.lkOverview("You find nothing that looks like a trap.")
}

func (g *Game) lkDisarm() {
	L := g.lkSt().s.L
	g.spend(3000)
	roll := g.R.Dice(2, 6) + g.checkMod("tinkering")
	dc := 7 + L.Tier
	switch {
	case roll >= dc:
		L.Off = true
		g.gainXP(6 + dc)
		g.lkFirst("disarm", "disarmed a trap on a lock")
		g.lkOverview("You disarm the trap.")
	case roll < dc-4:
		g.Msg("{{R|Your tool slips!}}")
		L.Off = true
		g.lkFireTrap(L)
		if g.Dead {
			return
		}
		g.lkOverview("You set it off.")
	default:
		g.lkOverview("You fumble it, but nothing happens. Try again.")
	}
}

func (g *Game) lkUnjam() {
	L := g.lkSt().s.L
	g.spend(3000)
	if g.R.Dice(2, 6)+g.checkMod("tinkering") >= 10 {
		L.Jam = 0
		g.lkOverview("You work the jam loose.")
		return
	}
	g.lkOverview("It will not budge yet.")
}

// lkFireTrap springs the trap on the player. Returns the trap id.
func (g *Game) lkFireTrap(L *lkLock) string {
	id := L.Trap
	if id == "" {
		return ""
	}
	L.Trap = ""
	p := g.P
	z := g.Cur
	tier := L.Tier
	g.lkFirst("trap", "set off a trap")
	switch id {
	case "needle":
		g.Msg("{{G|A needle jabs out of the lock!}}")
		g.hurt(p, g.R.Dice(1, 4), nil, "a poison needle")
		p.AddSt("poisoned", 8+tier*2)
	case "gas":
		g.Msg("{{G|A cloud of gas hisses from the lock!}}")
		p.AddSt("poisoned", 6+tier*2)
		p.AddSt("dazed", 3)
	case "alarm":
		g.Msg("{{O|A bell clangs somewhere inside!}}")
		g.lkAlarm(true)
	case "blades":
		g.Msg("{{R|Blades scythe out of the casing!}}")
		g.hurt(p, g.R.Dice(2, 6)+tier*2, nil, "scything blades")
		p.AddSt("bleeding", 4)
	case "mimic":
		if L.Kind == "chest" {
			z.Set(L.X, L.Y, TFloor)
			L.Open = true
		}
		a := g.Spawn(pickCreature(g.R, z.Biome, 'd', z.Level), z.Level+1, g.R)
		a.Name, a.Glyph, a.F = "mimic", '■', CBrown
		a.X, a.Y = L.X, L.Y
		g.nudge(z, a)
		z.Actors = append(z.Actors, a)
		g.Msg("{{R|The chest sprouts teeth. It was a mimic!}}")
	case "explosive":
		g.Msg("{{R|The hasp was wired to a powder charge!}}")
		L.Broken = true
		g.explode(L.X, L.Y, "fire", z.Level/3+tier, nil)
	}
	return id
}

// lkAlarm wakes the zone; loud brings reinforcements.
func (g *Game) lkAlarm(loud bool) {
	z := g.Cur
	for _, a := range z.Actors {
		if a == g.P || a.HP <= 0 || isFollower(a) {
			continue
		}
		if a.Status != nil {
			delete(a.Status, "asleep")
		}
	}
	if z.Kind == "town" {
		g.uwAlertGuards(g.uwStateHere(), z)
		return
	}
	if loud {
		for i := 0; i < 2; i++ {
			a := g.Spawn(pickCreature(g.R, z.Biome, 'd', z.Level), z.Level, g.R)
			a.X, a.Y = g.P.X+g.R.Int(-6, 6), g.P.Y+g.R.Int(-3, 3)
			g.nudge(z, a)
			z.Actors = append(z.Actors, a)
		}
	}
}

// lkWitness: stealing or breaking in inside a town is a crime when seen.
func (g *Game) lkWitness(L *lkLock, loud bool) {
	if g.Cur == nil || g.Cur.Kind != "town" || g.homeHere() != nil {
		return
	}
	state := g.uwStateHere()
	if loud {
		g.uwAddBounty(state, 40+L.Tier*20, " (breaking and entering)")
		g.uwAlertGuards(state, g.Cur)
		return
	}
	if g.uwSeen(nil) {
		g.uwAddBounty(state, 25+L.Tier*15, " (burglary)")
	}
}

// ---- force ----

func (g *Game) lkBashDC(L *lkLock) int {
	dc := 7 + 2*L.Tier
	if L.Kind == "door" {
		dc += 3
	}
	return dc
}

func (g *Game) lkBash() {
	L := g.lkSt().s.L
	g.spend(2000)
	if L.Magic != "" {
		g.Msg("{{M|The ward answers with a lash of force!}}")
		g.hurt(g.P, g.R.Dice(1, 6)+L.Tier, nil, "a warded lock")
		if !g.Dead {
			g.lkOverview("Brute force does nothing against a ward.")
		}
		return
	}
	g.lkNoise()
	if g.R.Dice(2, 6)+g.P.M(STR) >= g.lkBashDC(L) {
		g.Msg("{{O|The lock gives with a crash.}}")
		L.Broken = true
		g.lkWitness(L, true)
		g.lkSucceed("bash")
		return
	}
	g.Msg("{{R|It holds. Your shoulder rings.}}")
	g.hurt(g.P, g.R.Dice(1, 4), nil, "a stubborn lock")
	g.lkWitness(L, true)
	if !g.Dead {
		g.lkOverview("It will not break yet.")
	}
}

func (g *Game) lkNoise() {
	for _, a := range g.Cur.Actors {
		if a == g.P || isFollower(a) || a.HP <= 0 {
			continue
		}
		if abs(a.X-g.P.X) <= 14 && abs(a.Y-g.P.Y) <= 8 && a.Status != nil {
			delete(a.Status, "asleep")
		}
	}
}

// ---- magic locks ----

func (g *Game) lkMagicOptions(L *lkLock, add func(string, func())) {
	st := g.lkSt()
	switch L.Magic {
	case "riddle":
		rd := g.lkRiddle(L.ID)
		add("{{M|Answer the ward.}}", func() { g.lkRiddleScreen() })
		_ = rd
	case "blood":
		cost := 3 + 2*L.Tier
		add(fmt.Sprintf("{{R|Press your bleeding palm to the sigil.}} {{K|(-%d HP)}}", cost), func() {
			if g.P.HP <= cost+1 {
				g.lkOverview("You are too weak to pay in blood.")
				return
			}
			g.hurt(g.P, cost, nil, "a blood sigil")
			if g.Dead {
				return
			}
			g.Msg("{{R|The sigil drinks and glows.}}")
			g.lkFirst("blood", "paid a blood sigil with their own blood")
			g.spend(1000)
			g.lkSucceed("magic")
		})
	case "password":
		if st.Learned[L.ID] {
			add("{{M|Speak the word: \""+L.Arg+"\".}}", func() { g.spend(1000); g.lkSucceed("magic") })
		} else if L.Jam <= g.Turn {
			add("{{M|Guess a word...}}", func() { g.lkWordScreen() })
		}
	case "ally":
		add("{{M|Set the plates with a companion at your side.}}", func() {
			for _, c := range g.localCompanions() {
				if c.HP > 0 && maxi(abs(c.X-g.P.X), abs(c.Y-g.P.Y)) <= 3 {
					g.Msg(fmt.Sprintf("You and %s press the plates together.", c.Name))
					g.lkFirst("ally", "opened a lock that needed two pairs of hands")
					g.spend(1000)
					g.lkSucceed("magic")
					return
				}
			}
			g.lkOverview("No one stands close enough to reach the other plate.")
		})
	case "sun", "moon":
		add("{{M|Touch the inlay.}}", func() {
			if (L.Magic == "sun") != g.daylight() {
				g.lkOverview("The inlay stays cold. It is the wrong hour.")
				return
			}
			g.spend(1000)
			g.lkSucceed("magic")
		})
	}
}

func (g *Game) lkWrong(L *lkLock, w string) bool {
	for _, x := range L.Wrong {
		if x == w {
			return true
		}
	}
	return false
}

func (g *Game) lkRiddleScreen() {
	L := g.lkSt().s.L
	rd := g.lkRiddle(L.ID)
	var opts []Option
	for _, o := range rd.Opts {
		o := o
		if g.lkWrong(L, o) {
			continue
		}
		opts = append(opts, Option{"\"" + o + "\"", func() {
			if o == rd.A {
				g.Msg("{{G|The ward unravels.}}")
				g.lkFirst("riddle", "answered a lock's riddle")
				g.spend(1000)
				g.lkSucceed("magic")
				return
			}
			L.Wrong = append(L.Wrong, o)
			g.hurt(g.P, 2+L.Tier, nil, "a warded lock")
			if !g.Dead {
				g.lkRiddleScreen()
			}
		}})
	}
	if !L.Ponder && g.Skills["lore"] > 0 {
		opts = append(opts, Option{"{{c|Ponder what you know.}} " + g.checkLabel("lore", 9), func() {
			L.Ponder = true
			g.spend(2000)
			if g.roll2d6Check(g.checkMod("lore"), 9) {
				n := 0
				for _, o := range rd.Opts {
					if o != rd.A && !g.lkWrong(L, o) && n < 2 {
						L.Wrong = append(L.Wrong, o)
						n++
					}
				}
			}
			g.lkRiddleScreen()
		}})
	}
	opts = append(opts, Option{"Step back.", func() { g.lkOverview("") }})
	g.lkMenu("The ward speaks", "{{M|"+rd.Q+"}}\n\nA wrong answer burns. Somewhere nearby, carved or written, the answer may be recorded.", opts)
}

func (g *Game) lkWordScreen() {
	L := g.lkSt().s.L
	r := NewRng(g.Seed + ":lk:pwopts:" + L.ID)
	words := []string{L.Arg}
	for len(words) < 4 {
		w := Pick(r, lkWords)
		dup := false
		for _, x := range words {
			dup = dup || x == w
		}
		if !dup {
			words = append(words, w)
		}
	}
	for i := len(words) - 1; i > 0; i-- {
		j := r.Int(0, i)
		words[i], words[j] = words[j], words[i]
	}
	var opts []Option
	for _, w := range words {
		w := w
		if g.lkWrong(L, w) {
			continue
		}
		opts = append(opts, Option{"Say \"" + w + "\".", func() {
			if w == L.Arg {
				g.lkSt().Learned[L.ID] = true
				g.Msg("{{G|The letters settle. The lock opens.}}")
				g.lkFirst("password", "guessed a lock's password")
				g.spend(1000)
				g.lkSucceed("magic")
				return
			}
			L.Wrong = append(L.Wrong, w)
			L.Strikes++
			g.hurt(g.P, 1+L.Tier, nil, "a warded lock")
			if g.Dead {
				return
			}
			if L.Strikes >= 2 {
				L.Jam = g.Turn + 120
				L.Strikes = 0
				L.Wrong = nil
				g.lkOverview("The mouth snaps shut and seals itself for a while.")
				return
			}
			g.lkWordScreen()
		}})
	}
	opts = append(opts, Option{"Step back.", func() { g.lkOverview("") }})
	g.lkMenu("Say the word", "The ward-letters wait. Two wrong words and it seals. Someone in a town, or a library book, may know the word.", opts)
}

// ---- success and loot ----

func (g *Game) lkSucceed(how string) {
	L := g.lkSt().s.L
	g.lkClose()
	if L.Trap != "" && !L.Off {
		p := 1.0
		if L.Found {
			p = 0.4
		}
		if how == "key" {
			p *= 0.5
		}
		if g.R.Chance(p) {
			id := g.lkFireTrap(L)
			if g.Dead || id == "mimic" {
				return
			}
		}
	}
	if how == "pick" || how == "key" || how == "magic" {
		g.lkWitness(L, false)
	}
	g.gainXP(6 + L.Tier*5)
	if L.Kind == "door" {
		g.Cur.Set(L.X, L.Y, TDoorOpen)
		L.Open = true
		g.Msg("{{Y|The door swings open.}}")
		g.spend(1000)
		return
	}
	L.Open = true
	g.lkFirst("chest", "opened a locked chest")
	if L.Special != "" {
		g.lkFirst("vault"+L.Special, "opened a "+L.Special+"-sealed vault")
	}
	g.Msg("{{Y|The " + L.Name + " opens.}}")
	g.lkLoot(L)
	g.spend(1000)
}

func (g *Game) lkLoot(L *lkLock) {
	z := g.Cur
	r := NewRng(g.Seed + ":lk:loot:" + L.ID)
	lt := clampi(z.Level/3+L.Tier-1, 0, 5)
	n := 1 + r.Int(0, 1) + L.Tier/2
	var items []*Item
	for i := 0; i < n; i++ {
		items = append(items, g.Loot(r, lt))
	}
	if L.Tier >= 4 || L.Special == "split" {
		items = append(items, g.bossDrop(r, z.Level+L.Tier*4))
	}
	gold := r.Int(10, 30) * (L.Tier + 1) * maxi(1, z.Level/2)
	state := g.uwStateHere()
	theft := z.Kind == "town" && g.homeHere() == nil
	lost := 0
	for _, it := range items {
		if L.Broken && it.Kind != "artifact" && r.Chance(0.3) {
			lost++
			continue
		}
		if theft {
			g.uwMarkStolen(it, state)
		}
		g.give(it)
		g.Msg("{{G|You find " + it.Display() + ".}}")
	}
	if L.Broken {
		gold /= 2
		if lost > 0 {
			g.Msg(fmt.Sprintf("{{R|%d things inside were smashed.}}", lost))
		}
	}
	g.P.Gold += gold
	g.Msg(fmt.Sprintf("{{G|You pocket %d %s.}}", gold, g.Sk.Currency))
}

// ---- riddles ----

type lkRiddleT struct {
	Q, A, Hint string
	Opts       []string
}

func (g *Game) lkRiddle(id string) lkRiddleT {
	r := NewRng(g.Seed + ":lk:rd:" + id)
	w := g.W
	finish := func(q, a, hint string, pool []string) (lkRiddleT, bool) {
		opts := []string{a}
		for _, p := range pool {
			dup := p == "" || len(opts) >= 4
			for _, o := range opts {
				dup = dup || o == p
			}
			if !dup {
				opts = append(opts, p)
			}
		}
		if len(opts) < 4 {
			return lkRiddleT{}, false
		}
		for i := len(opts) - 1; i > 0; i-- {
			j := r.Int(0, i)
			opts[i], opts[j] = opts[j], opts[i]
		}
		return lkRiddleT{Q: q, A: a, Hint: hint, Opts: opts}, true
	}
	shuf := func(a []string) []string {
		for i := len(a) - 1; i > 0; i-- {
			j := r.Int(0, i)
			a[i], a[j] = a[j], a[i]
		}
		return a
	}
	kind := r.Int(0, 2)
	for tries := 0; tries < 3; tries++ {
		switch (kind + tries) % 3 {
		case 0:
			var rulers []string
			var st *StateData
			for i := range w.States {
				if w.States[i].Ruler != "" {
					rulers = append(rulers, w.States[i].Ruler)
					if st == nil && r.Chance(0.5) {
						st = &w.States[i]
					}
				}
			}
			if st == nil && len(w.States) > 0 && len(rulers) > 0 {
				for i := range w.States {
					if w.States[i].Ruler != "" {
						st = &w.States[i]
						break
					}
				}
			}
			if st != nil {
				if rd, ok := finish(fmt.Sprintf("Who sits the seat of %s in the year %d?", st.Name, w.Year), st.Ruler,
					fmt.Sprintf("Chiselled above the lintel: \"%s rules %s.\"", st.Ruler, st.Name), shuf(rulers)); ok {
					return rd
				}
			}
		case 1:
			var names []string
			var capital, owner string
			for i := range w.Burgs {
				b := &w.Burgs[i]
				names = append(names, b.Name)
				if b.Capital && b.Alive && capital == "" && r.Chance(0.5) {
					capital, owner = b.Name, w.StateName(b.State)
				}
			}
			if capital != "" {
				if rd, ok := finish(fmt.Sprintf("I am the seat of the crown of %s, and my streets are old. Name me.", owner), capital,
					fmt.Sprintf("Scratched in the plaster: \"%s keeps the crown of %s.\"", capital, owner), shuf(names)); ok {
					return rd
				}
			}
		case 2:
			var evs []EventData
			for _, e := range w.Events {
				if e.Y > 0 && len(e.Text) > 12 {
					evs = append(evs, e)
				}
			}
			if len(evs) > 0 {
				e := evs[r.Int(0, len(evs)-1)]
				txt := e.Text
				if len(txt) > 90 {
					txt = txt[:90] + "..."
				}
				pool := []string{}
				for i := 0; i < 6; i++ {
					pool = append(pool, fmt.Sprint(maxi(1, e.Y+r.Int(-40, 40))))
				}
				if rd, ok := finish(fmt.Sprintf("In what year did this come to pass: \"%s\"?", txt), fmt.Sprint(e.Y),
					fmt.Sprintf("Weathered letters, year %d: \"%s\"", e.Y, txt), pool); ok {
					return rd
				}
			}
		}
	}
	sr := lkStaticRiddles[r.Int(0, len(lkStaticRiddles)-1)]
	opts := []string{sr.A, sr.W[0], sr.W[1], sr.W[2]}
	for i := len(opts) - 1; i > 0; i-- {
		j := r.Int(0, i)
		opts[i], opts[j] = opts[j], opts[i]
	}
	return lkRiddleT{Q: sr.Q, A: sr.A, Hint: sr.Hint, Opts: opts}
}

// ---- zone hook ----

func lkNearFn(x, y, rx, ry int) func(int, int) bool {
	return func(px, py int) bool { return abs(px-x) <= rx && abs(py-y) <= ry }
}

func (g *Game) lkFree(zn *Zone, r *Rng, near func(int, int) bool) (int, int, bool) {
	return zn.FreeSpot(r, func(x, y int) bool {
		if zn.At(x, y) == TStall || (near != nil && !near(x, y)) {
			return false
		}
		for _, it := range zn.Items {
			if it.X == x && it.Y == y {
				return false
			}
		}
		for _, d := range dirs8 {
			if t := zn.At(x+d[0], y+d[1]); t == TUp || t == TDown || t == TLocked || t == TDoor {
				return false
			}
		}
		return true
	})
}

func (g *Game) lkPlaceChest(zn *Zone, r *Rng, x, y int, L *lkLock) {
	zn.Set(x, y, lkChestTile)
	L.ID, L.Kind, L.X, L.Y, L.Lvl = lkID(zn.Key, x, y), "chest", x, y, zn.Level
	if L.Name == "" {
		L.Name = lkChestByKind[zn.Kind]
		if L.Name == "" {
			L.Name = "chest"
		}
	}
	g.lkSt().Locks[L.ID] = L
	g.lkHints(zn, r, L)
}

// lkHints leaves the answer to a magic lock somewhere nearby.
func (g *Game) lkHints(zn *Zone, r *Rng, L *lkLock) {
	switch L.Magic {
	case "riddle":
		if x, y, ok := g.lkFree(zn, r, lkNearFn(L.X, L.Y, 12, 7)); ok {
			zn.Set(x, y, TInscription)
			if zn.Notes == nil {
				zn.Notes = map[int]string{}
			}
			zn.Notes[y*MW+x] = g.lkRiddle(L.ID).Hint
		}
	case "password":
		if x, y, ok := g.lkFree(zn, r, lkNearFn(L.X, L.Y, 16, 9)); ok {
			b := g.MakeBook("Torn notebook", fmt.Sprintf("{{Y|A torn notebook}}\n\nThe %s here answers to one word, spoken plainly: \"%s\". Do not write it down twice.", L.Name, L.Arg))
			b.Extra = map[string]int{"lkpw": lkH(L.ID)}
			b.X, b.Y = x, y
			zn.Items = append(zn.Items, b)
		}
	}
}

func (g *Game) lkStateOf(zn *Zone) int {
	if g.W.In(zn.WX, zn.WY) {
		return g.W.State[g.W.Idx(zn.WX, zn.WY)]
	}
	return 0
}

func (g *Game) lkRulers(state int) []PersonData {
	var out []PersonData
	for _, p := range g.W.People {
		if !p.Alive && p.Died > 0 && p.State == state && state != 0 {
			out = append(out, p)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Renown != out[j].Renown {
			return out[i].Renown > out[j].Renown
		}
		return out[i].ID < out[j].ID
	})
	if len(out) > 6 {
		out = out[:6]
	}
	return out
}

func lkZone(g *Game, zn *Zone, _ *Rng) {
	if g.W == nil || zn == nil {
		return
	}
	r := NewRng(g.Seed + ":lk:" + zn.Key)
	st := g.lkSt()
	state := g.lkStateOf(zn)

	// chests
	n := 0
	switch zn.Kind {
	case "dungeon", "tower", "crypt", "ruin", "underdark":
		if r.Chance(0.7) {
			n++
		}
		if r.Chance(0.25) {
			n++
		}
	case "camp", "lair":
		if r.Chance(0.4) {
			n++
		}
	case "town":
		if zn.Z == 0 && zn.Burg >= 0 && r.Chance(0.5) {
			n++
		}
	case "caravel", "steamer":
		n = 1
	}
	for i := 0; i < n; i++ {
		var near func(int, int) bool
		if zn.Kind == "town" {
			near = func(x, y int) bool {
				if zn.At(x, y) != TFloor {
					return false
				}
				for _, d := range dirs8 {
					if zn.At(x+d[0], y+d[1]) == TWall {
						return true
					}
				}
				return false
			}
		}
		x, y, ok := g.lkFree(zn, r, near)
		if !ok {
			continue
		}
		lvl := zn.Level
		bonus := 0
		if zn.Kind == "town" {
			lvl = 0
			if bd := g.W.burgByID[zn.Burg]; bd != nil {
				lvl = g.W.cityTier[bd.ID] * 4
			}
		}
		L := &lkLock{Tier: lkRollTier(r, lvl, bonus)}
		lkRollMagicTrap(r, L, 0.14, 0.22+0.05*float64(L.Tier))
		if zn.Kind == "town" && L.Magic == "password" {
			L.Magic = ""
		}
		if (zn.Kind == "ruin" || zn.Kind == "dungeon") && i == 0 && r.Chance(0.14) && L.Magic == "" {
			if rs := g.lkRulers(state); len(rs) > 0 {
				p := rs[r.Int(0, len(rs)-1)]
				if !st.Delivered[fmt.Sprint(p.ID)] {
					L.Special, L.Key, L.Tier = "royal", lkH("ruler:"+fmt.Sprint(p.ID)), maxi(L.Tier, 3)
					L.KeyHint = "the signet of " + p.Name
					L.Name = "royal strongbox"
					key := fmt.Sprint(state)
					have := false
					for _, w := range st.Wants[key] {
						have = have || w == p.ID
					}
					if !have {
						st.Wants[key] = append(st.Wants[key], p.ID)
					}
				}
			}
		}
		g.lkPlaceChest(zn, r, x, y, L)
	}

	switch zn.Kind {
	case "crypt":
		g.lkCryptKey(zn, r, state)
	case "dungeon":
		g.lkHideout(zn, r)
		g.lkVaults(zn)
	case "tower":
		g.lkVaults(zn)
	case "town":
		g.lkLocksmith(zn, r)
	}
	if (zn.Kind == "dungeon" || zn.Kind == "tower") && zn.Z != 0 {
		g.lkSplitPart(zn, r)
	}
}

func (g *Game) lkCryptKey(zn *Zone, r *Rng, state int) {
	st := g.lkSt()
	rs := g.lkRulers(state)
	if len(rs) == 0 {
		return
	}
	pid := 0
	key := fmt.Sprint(state)
	if w := st.Wants[key]; len(w) > 0 && r.Chance(0.85) {
		pid = w[0]
		st.Wants[key] = w[1:]
	} else if r.Chance(0.15) {
		pid = rs[r.Int(0, len(rs)-1)].ID
	}
	if pid == 0 || st.Delivered[fmt.Sprint(pid)] {
		return
	}
	var who PersonData
	for _, p := range rs {
		if p.ID == pid {
			who = p
		}
	}
	if who.ID == 0 && who.Name == "" {
		for _, p := range g.W.People {
			if p.ID == pid {
				who = p
			}
		}
	}
	x, y, ok := g.lkFree(zn, r, nil)
	for yy := 1; yy < MH-1 && !ok; yy++ {
		for xx := 1; xx < MW-1; xx++ {
			if zn.At(xx, yy) == TSarcophagus {
				x, y, ok = xx, yy+1, zn.Walkable(xx, yy+1)
				break
			}
		}
	}
	if !ok {
		return
	}
	st.Delivered[fmt.Sprint(pid)] = true
	it := g.lkMakeKey("signet key of "+who.Name, fmt.Sprintf("The signet key of %s, who died in year %d. It opens whatever the crown sealed.", who.Name, who.Died), lkH("ruler:"+fmt.Sprint(pid)), CYellow)
	it.Rarity, it.Value = "rare", 150
	it.X, it.Y = x, y
	zn.Items = append(zn.Items, it)
}

// lkHideout: outlaw bosses carry a guildmaster's ring that fits a strongbox in the same hall.
func (g *Game) lkHideout(zn *Zone, r *Rng) {
	s := g.siteAt(zn.WX, zn.WY, zn.ZX, zn.ZY)
	if s == nil || s.Theme != "hideout" || !r.Chance(0.6) {
		return
	}
	var boss *Actor
	for _, a := range zn.Actors {
		if a.HP > 0 && a.AI != "town" && a.AI != "follower" && (boss == nil || a.Level > boss.Level) {
			boss = a
		}
	}
	x, y, ok := g.lkFree(zn, r, nil)
	if boss == nil || !ok {
		return
	}
	id := lkH("guild:" + zn.Key)
	boss.Inv = append(boss.Inv, g.lkMakeKey("guildmaster's ring", "A heavy signet ring of the thieves' guild. Its seal fits the guild's strongboxes.", id, CYellow))
	L := &lkLock{Tier: lkRollTier(r, zn.Level, 1), Special: "guild", Key: id, KeyHint: "the guildmaster's ring", Name: "guild strongbox"}
	g.lkPlaceChest(zn, r, x, y, L)
}

// lkSplitPart: the three parts of one key lie in three dungeon levels; the vault is where part 1 lies.
func (g *Game) lkSplitPart(zn *Zone, r *Rng) {
	st := g.lkSt()
	if st.Split == nil {
		sr := NewRng(g.Seed + ":lk:split")
		st.Split = &lkSplit{Name: "the " + Pick(sr, lkSplitA) + " " + Pick(sr, lkSplitB), ID: lkH("split:" + g.Seed)}
	}
	sp := st.Split
	if sp.Placed >= 3 || !r.Chance(0.12) {
		return
	}
	for _, z := range sp.Zones {
		if z == zn.Key {
			return
		}
	}
	x, y, ok := g.lkFree(zn, r, nil)
	if !ok {
		return
	}
	sp.Placed++
	sp.Zones = append(sp.Zones, zn.Key)
	it := &Item{ID: g.newID(), Kind: "keypart", Name: fmt.Sprintf("key part %d of 3", sp.Placed), Glyph: '¬', F: CMagenta, Value: 80, Ident: true,
		Extra: map[string]int{"lkpart": sp.Placed, "lkset": sp.ID}, X: x, Y: y,
		Lore: fmt.Sprintf("One of three parts of the key to %s. The others lie in deep halls of this land. A locksmith can join them.", sp.Name)}
	zn.Items = append(zn.Items, it)
	if sp.Placed == 1 {
		if cx, cy, ok := g.lkFree(zn, r, lkNearFn(x, y, 25, 10)); ok {
			L := &lkLock{Tier: 5, Special: "split", Key: sp.ID, KeyHint: sp.Name, Name: "sealed reliquary"}
			L.Trap = Pick(r, []lkTrapDef{lkTrapByID("blades"), lkTrapByID("explosive"), lkTrapByID("gas")}).ID
			g.lkPlaceChest(zn, r, cx, cy, L)
			sp.Vault = zn.Name
		}
	}
}

func (g *Game) lkHasParts() bool {
	sp := g.lkSt().Split
	if sp == nil {
		return false
	}
	have := map[int]bool{}
	for _, it := range g.P.Inv {
		if it.Kind == "keypart" && it.Extra["lkset"] == sp.ID {
			have[it.Extra["lkpart"]] = true
		}
	}
	return have[1] && have[2] && have[3]
}

func (g *Game) lkJoinParts() {
	sp := g.lkSt().Split
	if sp == nil || !g.lkHasParts() {
		return
	}
	used := map[int]bool{}
	for _, it := range append([]*Item(nil), g.P.Inv...) {
		n := it.Extra["lkpart"]
		if it.Kind == "keypart" && it.Extra["lkset"] == sp.ID && !used[n] {
			used[n] = true
			g.removeInv(it)
		}
	}
	k := g.lkMakeKey("key of "+sp.Name, "Three parts joined into one key.", sp.ID, CMagenta)
	k.Rarity, k.Value = "unique", 400
	g.give(k)
	g.Msg("{{M|The three parts lock together into the key of " + sp.Name + ".}}")
	g.lkFirst("joinkey", "joined the three parts of a split key")
}

// ---- bait vaults ----

func (g *Game) lkVaults(zn *Zone) {
	st := g.lkSt()
	for i, t := range zn.T {
		if t != TLocked {
			continue
		}
		x, y := i%MW, i/MW
		id := lkID(zn.Key, x, y)
		r := NewRng(g.Seed + ":lk:vault:" + id)
		if st.Locks[id] != nil || st.Baits[zn.Key] != nil {
			continue
		}
		if r.Chance(0.22) {
			g.lkMakeBait(zn, r, x, y)
			continue
		}
		// a real vault door: maybe a magic lock with a hint nearby
		L := &lkLock{ID: id, Kind: "door", Name: "vault door", X: x, Y: y, Lvl: zn.Level}
		L.Tier = clampi(2+zn.Level/8+r.Int(0, 1), 2, 5)
		lkRollMagicTrap(r, L, 0.10, 0.12)
		if L.Magic == "password" || L.Magic == "riddle" {
			g.lkHints(zn, r, L)
		}
		st.Locks[id] = L
	}
}

func (g *Game) lkMakeBait(zn *Zone, r *Rng, dx, dy int) {
	fill := func(sx, sy int) (x0, y0, x1, y1 int, ok bool) {
		seen := map[int]bool{dy*MW + dx: true}
		q := [][2]int{{sx, sy}}
		x0, y0, x1, y1 = sx, sy, sx, sy
		for len(q) > 0 {
			c := q[0]
			q = q[1:]
			i := c[1]*MW + c[0]
			if seen[i] || !zn.In(c[0], c[1]) || !zn.At(c[0], c[1]).D().Walk {
				continue
			}
			seen[i] = true
			if len(seen) > 90 {
				return 0, 0, 0, 0, false
			}
			x0, y0, x1, y1 = mini(x0, c[0]), mini(y0, c[1]), maxi(x1, c[0]), maxi(y1, c[1])
			for _, d := range [][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
				q = append(q, [2]int{c[0] + d[0], c[1] + d[1]})
			}
		}
		return x0, y0, x1, y1, true
	}
	for _, d := range [][2]int{{0, -1}, {0, 1}, {1, 0}, {-1, 0}} {
		x0, y0, x1, y1, ok := fill(dx+d[0], dy+d[1])
		if !ok {
			continue
		}
		b := &lkBait{DX: dx, DY: dy, X0: x0, Y0: y0, X1: x1, Y1: y1, Armed: true, Mode: Pick(r, []string{"ambush", "gas"})}
		g.lkSt().Baits[zn.Key] = b
		zn.Set(dx, dy, TDoorOpen)
		keyID := int(HashStr(zn.Key) % 1000000)
		kept := zn.Items[:0]
		for _, it := range zn.Items {
			if !(it.Kind == "key" && it.Extra != nil && it.Extra["key"] == keyID) {
				kept = append(kept, it)
			}
		}
		zn.Items = kept
		return
	}
}

func lkTurn(g *Game) {
	if g.Cur == nil || g.W == nil || g.Dead {
		return
	}
	b := g.lkSt().Baits[g.Cur.Key]
	if b == nil || !b.Armed {
		return
	}
	px, py := g.P.X, g.P.Y
	if !b.Warned && abs(px-b.DX) <= 4 && abs(py-b.DY) <= 4 {
		b.Warned = true
		if g.R.Dice(2, 6)+g.checkMod("perception") >= 9 {
			g.Msg("{{y|The vault stands open and untouched. No dust on the sill. That is wrong.}}")
		}
	}
	if px < b.X0 || px > b.X1 || py < b.Y0 || py > b.Y1 {
		return
	}
	b.Armed = false
	z := g.Cur
	z.Set(b.DX, b.DY, TLocked)
	id := lkID(z.Key, b.DX, b.DY)
	g.lkSt().Locks[id] = &lkLock{ID: id, Kind: "door", Name: "sealed vault door", X: b.DX, Y: b.DY, Tier: 3, Lvl: z.Level, Special: "bait"}
	g.lkFirst("bait", "was caught in a bait vault")
	g.Msg("{{R|The door slams shut behind you! It was a snare.}}")
	if b.Mode == "gas" {
		g.Msg("{{G|Gas pours from the vents.}}")
		g.P.AddSt("poisoned", 14)
		g.P.AddSt("dazed", 3)
		return
	}
	for i := 0; i < 3; i++ {
		a := g.Spawn(pickCreature(g.R, z.Biome, 'd', z.Level), z.Level+1, g.R)
		a.X, a.Y = px+g.R.Int(-2, 2), py+g.R.Int(-2, 2)
		g.nudge(z, a)
		z.Actors = append(z.Actors, a)
	}
	g.Msg("{{R|Figures step out of the walls.}}")
}

// ---- locksmith ----

func (g *Game) lkLocksmith(zn *Zone, r *Rng) {
	if zn.Z != 0 || zn.ZX != 1 || zn.ZY != 1 || zn.Burg < 0 {
		return
	}
	bd := g.W.burgByID[zn.Burg]
	if bd == nil || !r.Chance(0.55+0.15*float64(g.W.cityTier[bd.ID])) {
		return
	}
	x, y, ok := zn.FreeSpot(r, nil)
	if !ok {
		return
	}
	a := g.makeNPC("locksmith", bd, g.W.cultByID[bd.Culture], r)
	a.Role, a.Title, a.F = "locksmith", "locksmith", COrange
	a.Desc = "A locksmith of " + bd.Name + ", keys jangling on a ring."
	a.AI, a.Gold = "still", 80+r.Int(0, 150)
	a.X, a.Y, a.HomeX, a.HomeY = x, y, x, y
	zn.Actors = append(zn.Actors, a)
}

func lkTalk(g *Game, a *Actor) []Option {
	st := g.lkSt()
	var out []Option
	if a.Role == "locksmith" {
		out = append(out, Option{"{{O|Lockpicks, lessons, keys.}}", func() { g.lkSmithMenu(a) }})
	}
	if len(st.Heard) > 0 && !a.Has("lkpw") && a.AI != "follower" && (a.Role == "elder" || a.Role == "notable" || a.Role == "scholar" || a.Role == "bookseller" || a.Role == "locksmith") {
		out = append(out, Option{"{{y|Ask if anyone knows the word for a sealed lock.}}", func() {
			a.Set("lkpw")
			id := st.Heard[0]
			L := st.Locks[id]
			if L != nil && g.roll2d6Check(g.checkMod("persuasion"), dcMedium) {
				st.Learned[id] = true
				st.Heard = st.Heard[1:]
				g.talkMenu(fmt.Sprintf("\"Hush. They say the %s opens to the word '%s'. You did not hear it from me.\"", L.Name, L.Arg))
				return
			}
			g.talkMenu("\"I know nothing about that.\"")
		}})
	}
	return out
}

func (g *Game) lkSmithMenu(a *Actor) {
	st := g.lkSt()
	cur := g.Sk.Currency
	back := func() { g.talkMenu(g.greeting(a)) }
	var opts []Option
	for q := 1; q <= 3; q++ {
		q := q
		price := lkPickPrice[q]
		opts = append(opts, Option{fmt.Sprintf("Buy %s. {{K|(%d %s, quality %d)}}", lkPickName[q], price, cur, q), func() {
			if g.P.Gold < price {
				g.lkMenu("Locksmith", "\"Come back with coin.\"", []Option{{"Back.", func() { g.lkSmithMenu(a) }}})
				return
			}
			g.P.Gold -= price
			g.give(g.lkMakePick(q, false))
			g.lkSmithMenu(a)
		}})
	}
	if st.Rank < 3 {
		fee := 120 * (st.Rank + 1)
		opts = append(opts, Option{fmt.Sprintf("Teach me lockpicking. {{K|(%d %s, rank %d to %d)}}", fee, cur, st.Rank, st.Rank+1), func() {
			if g.P.Gold < fee {
				g.lkMenu("Locksmith", "\"Lessons are not free.\"", []Option{{"Back.", func() { g.lkSmithMenu(a) }}})
				return
			}
			g.P.Gold -= fee
			st.Rank++
			g.spend(60000)
			g.gainXP(15 * st.Rank)
			g.lkFirst("learn", "learned the art of lockpicking from a locksmith")
			g.lkMenu("Locksmith", fmt.Sprintf("\"Again. Feel the pin, not the metal.\" Your lockpicking is now rank %d.", st.Rank), []Option{{"Thank you.", func() { g.lkSmithMenu(a) }}})
		}})
	}
	var ids []string
	for id := range st.Studied {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		id := id
		sd := st.Studied[id]
		fee := 60 + sd.Tier*60
		opts = append(opts, Option{fmt.Sprintf("Cut a key from my wax: %s. {{K|(%d %s)}}", sd.Label, fee, cur), func() {
			if g.P.Gold < fee {
				g.lkMenu("Locksmith", "\"The file does not turn for free.\"", []Option{{"Back.", func() { g.lkSmithMenu(a) }}})
				return
			}
			g.P.Gold -= fee
			delete(st.Studied, id)
			k := g.lkMakeKey("cut key ("+sd.Label+")", "A key filed from a wax impression. It will fit once.", 0, CGray)
			k.Extra = map[string]int{"lkcut": lkH(id)}
			k.Value = 5
			g.give(k)
			g.spend(120000)
			g.lkFirst("cutkey", "had a locksmith cut a key from a wax impression")
			g.lkSmithMenu(a)
		}})
	}
	if g.lkHasParts() {
		opts = append(opts, Option{"{{G|Join these three key parts.}}", func() { g.lkJoinParts(); g.lkSmithMenu(a) }})
	}
	opts = append(opts, Option{"Goodbye.", back})
	g.lkMenu("Locksmith: "+a.Name, Pick(NewRng(g.Seed+":lk:smith:"+a.Name), lkSmithLines)+fmt.Sprintf("\n\nYour lockpicking rank: %d. You carry %d %s.", st.Rank, g.P.Gold, cur), opts)
}
