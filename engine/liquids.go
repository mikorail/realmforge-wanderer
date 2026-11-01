package engine

import (
	"fmt"
	"strings"
)

// Liquids (WP3, prefix lq). Containers are the existing Kind "water" items: Charges = drams held,
// Power = capacity, Effect = liquid key ("" = fresh water). Pools on the map use Zone.Liq ids:
// 1 blood 2 oil 3 acid 4 slime (core) and 5.. below.

type lqLiquid struct {
	Key, Name string
	ID        uint8 // Zone.Liq id
	F, B      byte  // map tint (B 0 = keep the tile's background)
	Holdable  bool
	Drink     string // short note for the pour screen
}

const (
	lqFresh = iota + 5
	lqSalt
	lqHoney
	lqWine
	lqLava
	lqBrackish
	lqTainted
	lqGenre
)

var lqTable = []lqLiquid{
	{"fresh", "fresh water", lqFresh, CCyan, CDarkBlue, true, "quenches"},
	{"salt", "salt water", lqSalt, CBlue, CDarkBlue, true, "makes you thirstier"},
	{"honey", "honey", lqHoney, CYellow, CBrown, true, "sweet and filling, a little sticky"},
	{"wine", "wine", lqWine, CMagenta, CDarkMagenta, true, "dulls the head"},
	{"lava", "lava", lqLava, COrange, CBgRed, false, "burns"},
	{"brackish", "brackish water", lqBrackish, CDarkCyan, CDarkBlue, true, "halfway drinkable, may sicken"},
	{"tainted", "tainted water", lqTainted, CGreen, CDarkGreen, true, "mixed with something foul"},
	{"genre", "", lqGenre, CMagenta, CDarkMagenta, true, ""},
	// core pool ids, so a container can hold what the pools hold
	{"blood", "blood", 1, CRed, CBgRed, true, "foul and thin"},
	{"oil", "oil", 2, CBrown, CBrown, true, "poisonous, burns"},
	{"acid", "acid", 3, CGreen, 0, false, "eats through anything"},
	{"slime", "slime", 4, CGreen, CDarkGreen, true, "makes you reel"},
}

type lqState struct {
	Surplus int            `json:"sp,omitempty"` // overdrinking
	Foraged map[string]int `json:"fg,omitempty"` // zone+tile → day last foraged
	Sel     int            `json:"-"`
	Mix     bool           `json:"-"` // choosing the container to pour into
}

func (g *Game) lq() *lqState { return g.ext("lq", func() any { return &lqState{} }).(*lqState) }

func (g *Game) lqGenreName() string {
	switch g.Genre {
	case "mythic":
		return "nectar"
	case "steampunk":
		return "boiler-water"
	case "dieselpunk":
		return "diesel"
	case "atompunk":
		return "glowing sludge"
	case "cyberpunk":
		return "coolant"
	case "biopunk":
		return "ichor"
	case "solarpunk":
		return "sap-nectar"
	case "wasteland":
		return "murky runoff"
	case "twilight":
		return "black wine"
	}
	return "nectar"
}

func (g *Game) lqGenreGood() bool {
	switch g.Genre {
	case "steampunk", "dieselpunk", "atompunk", "cyberpunk", "wasteland":
		return false
	}
	return true
}

func lqByKey(k string) *lqLiquid {
	if k == "" {
		k = "fresh"
	}
	for i := range lqTable {
		if lqTable[i].Key == k {
			return &lqTable[i]
		}
	}
	return &lqTable[0]
}

func lqByID(id uint8) *lqLiquid {
	for i := range lqTable {
		if lqTable[i].ID == id {
			return &lqTable[i]
		}
	}
	return nil
}

func (g *Game) lqName(k string) string {
	l := lqByKey(k)
	if l.Key == "genre" {
		return g.lqGenreName()
	}
	return l.Name
}

// lqTint is the render hook: the colours of a pool id >= 5 (and acid etc. if you want them).
func lqTint(id uint8) (f, b byte, ok bool) {
	if id < 5 {
		return 0, 0, false
	}
	if l := lqByID(id); l != nil {
		return l.F, l.B, true
	}
	return 0, 0, false
}

// ---- containers ----

func lqKey(it *Item) string {
	if it.Effect == "" {
		return "fresh"
	}
	return it.Effect
}

func (g *Game) lqContainers() []*Item {
	var out []*Item
	for _, it := range g.P.Inv {
		if it.Kind == "water" {
			out = append(out, it)
		}
	}
	return out
}

// lqSet puts n drams of liquid k in the container (replacing what it held) and renames it.
func (g *Game) lqSet(it *Item, k string, n int) {
	it.Charges = clampi(n, 0, it.Power)
	if k == "fresh" {
		it.Effect = ""
	} else {
		it.Effect = k
	}
	g.lqRename(it)
}

func (g *Game) lqRename(it *Item) {
	base := it.Name
	if i := strings.Index(base, " of "); i >= 0 {
		base = base[:i]
	}
	it.Name = base
	if it.Charges > 0 && lqKey(it) != "fresh" {
		it.Name = base + " of " + g.lqName(lqKey(it))
	}
	if it.Charges == 0 {
		it.Effect = ""
	}
}

// lqMixKey: what two liquids make when they meet.
func lqMixKey(a, b string) string {
	if a == b {
		return a
	}
	pair := func(x, y string) bool { return (a == x && b == y) || (a == y && b == x) }
	switch {
	case pair("fresh", "salt"), pair("fresh", "brackish"), pair("salt", "brackish"):
		return "brackish"
	case pair("fresh", "wine"):
		return "wine"
	case pair("fresh", "honey"):
		return "honey"
	}
	return "tainted"
}

// ---- pools ----

// lqSaltAt: 0 fresh, 1 brackish (coast), 2 salt (open sea).
func (g *Game) lqSaltAt(z *Zone) int {
	if z.Underwater {
		return 2
	}
	w := g.W
	if !w.In(z.WX, z.WY) {
		return 0
	}
	if w.IsWater(w.Idx(z.WX, z.WY)) {
		return 2
	}
	for _, d := range dirs8 {
		if x, y := z.WX+d[0], z.WY+d[1]; w.In(x, y) && w.IsWater(w.Idx(x, y)) {
			return 1
		}
	}
	return 0
}

// lqPoolAt: the liquid at a tile, if any is there to drink or fill from.
func (g *Game) lqPoolAt(x, y int) (string, bool) {
	z := g.Cur
	if z == nil || !z.In(x, y) {
		return "", false
	}
	if id := z.Liq[y*MW+x]; id > 0 {
		if l := lqByID(id); l != nil {
			return l.Key, true
		}
		return "", false
	}
	switch z.At(x, y) {
	case TShallow:
		return []string{"fresh", "brackish", "salt"}[g.lqSaltAt(z)], true
	case TWell:
		return "fresh", true
	}
	return "", false
}

// lqPoolNear: the pool you stand in or beside (own tile first).
func (g *Game) lqPoolNear() (key string, x, y int, ok bool) {
	p := g.P
	if k, ok := g.lqPoolAt(p.X, p.Y); ok {
		return k, p.X, p.Y, true
	}
	for _, d := range dirs8 {
		if k, ok := g.lqPoolAt(p.X+d[0], p.Y+d[1]); ok {
			return k, p.X + d[0], p.Y + d[1], true
		}
	}
	return "", 0, 0, false
}

// lqLook describes a pool; Survival tells safe water from brackish.
func (g *Game) lqLook(k string) string {
	if g.Skills["survival"] > 0 {
		switch k {
		case "fresh":
			return "fresh water (safe to drink)"
		case "brackish", "tainted":
			return g.lqName(k) + " (not safe)"
		case "salt":
			return "salt water (do not drink)"
		}
		return g.lqName(k)
	}
	switch k {
	case "fresh", "brackish", "tainted", "salt":
		return "still water"
	}
	return g.lqName(k)
}

// ---- drinking ----

// lqDrinkEffect applies n drams of liquid k to the player.
func (g *Game) lqDrinkEffect(k string, n int) {
	p := g.P
	switch k {
	case "fresh":
		g.lqQuench(450 * n)
	case "salt":
		g.Thirst += 250 * n
		g.Msg("{{O|The brine burns your throat and leaves you thirstier.}}")
	case "brackish":
		g.lqQuench(150 * n)
		if g.R.Chance(0.3 * float64(n)) {
			p.AddSt("poisoned", 8)
			g.Msg("{{G|The brackish water turns your stomach.}}")
		}
	case "tainted":
		g.lqQuench(150 * n)
		if g.R.Chance(0.7) {
			p.AddSt("poisoned", 8)
			g.Msg("{{G|It tastes of whatever was mixed into it. Your gut protests.}}")
		}
	case "honey":
		g.Hunger = maxi(0, g.Hunger-120*n)
		g.Thirst += 40 * n
		g.Msg("{{Y|Sweet and thick; it clings to your teeth.}}")
	case "wine":
		g.lqQuench(80 * n)
		p.AddSt("confused", 3+n)
		g.Msg("{{M|The wine goes to your head.}}")
	case "blood":
		g.Hunger = maxi(0, g.Hunger-60*n)
		if g.R.Chance(0.4) {
			p.AddSt("poisoned", 6)
		}
		g.Msg("{{R|You gag it down.}}")
	case "oil":
		p.AddSt("poisoned", 10)
		g.hurt(p, 2*n, nil, "drinking oil")
		g.Msg("{{O|Oil coats your throat. You retch.}}")
	case "slime":
		p.AddSt("confused", 5+n)
		g.Msg("{{G|The slime sends the world sliding sideways.}}")
	case "genre":
		if g.lqGenreGood() {
			g.lqQuench(200 * n)
			p.HP = mini(p.MaxHP, p.HP+3*n)
			g.Msg("{{G|The " + g.lqGenreName() + " warms you from the inside.}}")
		} else {
			g.lqQuench(50 * n)
			p.AddSt("poisoned", 10)
			g.hurt(p, n, nil, "drinking "+g.lqGenreName())
			g.Msg("{{O|The " + g.lqGenreName() + " is not meant to be drunk.}}")
		}
	}
}

// lqQuench lowers Thirst; going below zero builds a surplus (overdrinking makes you vomit).
func (g *Game) lqQuench(amount int) {
	if g.Thirst >= amount {
		g.Thirst -= amount
		return
	}
	over := amount - g.Thirst
	g.Thirst = 0
	st := g.lq()
	st.Surplus += over
	if st.Surplus >= 900 {
		st.Surplus = 0
		g.Hunger += 150
		g.Msg("{{G|You drank more than you could hold and vomit it back up.}}")
		p := g.P
		if g.Cur.Liq[p.Y*MW+p.X] == 0 && g.Cur.At(p.X, p.Y).D().Water == 0 {
			g.Cur.Liq[p.Y*MW+p.X] = 4
		}
	}
}

// lqDrinkItem handles a drink from a container holding anything but fresh water. Hook for
// useItem(): returns true when handled.
func (g *Game) lqDrinkItem(it *Item) bool {
	if it.Kind != "water" || lqKey(it) == "fresh" || it.Charges <= 0 {
		return false
	}
	k := lqKey(it)
	it.Charges--
	g.Msg(fmt.Sprintf("You drink a dram of %s. (%d left)", g.lqName(k), it.Charges))
	g.lqDrinkEffect(k, 1)
	g.lqRename(it)
	return true
}

// lqDrinkHere handles drinking at a pool or well (hook for drinkHere(); returns true when handled,
// false to let the old code run for plain fresh water).
func (g *Game) lqDrinkHere() bool {
	k, x, y, ok := g.lqPoolNear()
	if !ok {
		return false
	}
	if k == "fresh" {
		if g.Thirst == 0 {
			g.lqQuench(300) // topping up while quenched fills you, then sickens you
		}
		return false
	}
	if k == "lava" || k == "acid" {
		g.Msg("{{R|You are not that desperate.}}")
		return true
	}
	if (k == "brackish" || k == "salt" || k == "tainted") && g.Skills["survival"] > 0 {
		g.Msg("{{W|Your survival lore stops you: this is " + g.lqLook(k) + ".}}")
		return true
	}
	_ = x
	_ = y
	g.Msg("You drink from the " + g.lqName(k) + ".")
	g.lqDrinkEffect(k, 2)
	return true
}

// ---- fill, pour, mix ----

const lqPoolDrams = 16

func (g *Game) lqFill(it *Item) {
	k, x, y, ok := g.lqPoolNear()
	if !ok {
		g.Msg("There is no pool or well within reach.")
		return
	}
	if !lqByKey(k).Holdable {
		g.Msg("It would burn through your " + it.Name + ".")
		return
	}
	if it.Charges > 0 && lqKey(it) != k {
		k = lqMixKey(lqKey(it), k)
		g.Msg("{{O|The liquids swirl together.}}")
	}
	room := it.Power - it.Charges
	if room <= 0 {
		g.Msg("It is full.")
		return
	}
	n := room
	idx := y*MW + x
	if g.Cur.Liq[idx] > 0 { // a pool tile holds a limited amount
		n = mini(room, lqPoolDrams)
		g.Cur.Liq[idx] = 0
	}
	g.lqSet(it, k, it.Charges+n)
	g.Msg(fmt.Sprintf("You fill your %s with %d drams of %s.", it.Name, n, g.lqName(k)))
	g.spend(1000)
}

func (g *Game) lqPourOut(it *Item) {
	if it.Charges <= 0 {
		g.Msg("It is empty.")
		return
	}
	k := lqKey(it)
	p, z := g.P, g.Cur
	idx := p.Y*MW + p.X
	g.Msg(fmt.Sprintf("You pour out %d drams of %s.", it.Charges, g.lqName(k)))
	switch {
	case z.Fire[idx] > 0 && (k == "fresh" || k == "salt" || k == "wine" || k == "brackish"):
		z.Fire[idx] = 0
		g.Msg("{{B|The fire hisses out.}}")
	case z.At(p.X, p.Y).D().Water > 0 || z.Underwater:
		g.Msg("It vanishes into the water.")
	case !z.At(p.X, p.Y).D().Walk:
		g.Msg("It soaks away.")
	default:
		id := lqByKey(k).ID
		if old := z.Liq[idx]; old != 0 && old != id {
			if ol := lqByID(old); ol != nil {
				id = lqByKey(lqMixKey(k, ol.Key)).ID
			}
		}
		z.Liq[idx] = id
		if k == "honey" {
			g.Msg("{{Y|It pools thick and sticky.}}")
		}
	}
	g.lqSet(it, "fresh", 0)
	g.spend(1000)
}

func (g *Game) lqPourInto(src, dst *Item) {
	if src == dst || src.Charges <= 0 {
		return
	}
	room := dst.Power - dst.Charges
	if room <= 0 {
		g.Msg("The " + dst.Name + " is full.")
		return
	}
	n := mini(room, src.Charges)
	k := lqKey(src)
	if dst.Charges > 0 {
		k2 := lqMixKey(lqKey(dst), k)
		if k2 != k || lqKey(dst) != k {
			g.Msg("{{O|The liquids mix into " + g.lqName(k2) + ".}}")
		}
		k = k2
	}
	g.lqSet(dst, k, dst.Charges+n)
	g.lqSet(src, lqKey(src), src.Charges-n)
	g.Msg(fmt.Sprintf("You pour %d drams into the %s.", n, dst.Name))
	g.spend(1000)
}

// ---- foraging (Survival) ----

func (g *Game) lqForage() {
	sv := g.Skills["survival"]
	z, p := g.Cur, g.P
	if sv < 1 {
		g.Msg("You don't know where to look for water. (Survival)")
		return
	}
	if z.Underwater || (z.Kind != "surface" && z.Kind != "outskirts" && z.Z == 0) || z.Z != 0 {
		g.Msg("There is nothing to forage for water here.")
		return
	}
	if g.nearHostile() {
		g.Msg("Not with enemies in sight.")
		return
	}
	st := g.lq()
	key := fmt.Sprintf("%s:%d:%d", z.Key, p.X/6, p.Y/4)
	if st.Foraged == nil {
		st.Foraged = map[string]int{}
	}
	if d, ok := st.Foraged[key]; ok && d == g.Day() {
		g.Msg("You've already picked this patch clean today.")
		return
	}
	st.Foraged[key] = g.Day()
	chance := 0.30 + 0.15*float64(sv)
	switch z.At(p.X, p.Y) {
	case TSand, TAsh:
		chance -= 0.2
	case TSnow, TIce:
		chance += 0.2
	case TGrass, TTallGrass, TMoss, TReeds, TBush, TTree:
		chance += 0.15
	}
	g.spend(4000)
	h := float64(HashStr(fmt.Sprintf("%s:forage:%s:%d", g.Seed, key, g.Day()))%1000) / 1000
	if h >= chance {
		g.Msg("You search for dew, roots and seeps but find nothing.")
		return
	}
	n := 2 + sv*2
	for _, it := range g.lqContainers() {
		if it.Charges > 0 && lqKey(it) != "fresh" {
			continue
		}
		add := mini(n, it.Power-it.Charges)
		if add > 0 {
			g.lqSet(it, "fresh", it.Charges+add)
			g.Msg(fmt.Sprintf("{{G|You gather %d drams of clean water into your %s.}}", add, it.Name))
			return
		}
	}
	g.lqQuench(450 * mini(n, 3))
	g.Msg("{{G|You find a seep and drink your fill.}}")
}

// ---- turn hooks ----

func lqPoolTick(g *Game) {
	z, p := g.Cur, g.P
	if z == nil || p == nil {
		return
	}
	// stand effects
	switch z.Liq[p.Y*MW+p.X] {
	case lqLava:
		g.Msg("{{R|The lava sears you!}}")
		p.AddSt("burning", 4)
		g.hurt(p, 3+g.R.Dice(1, 4), nil, "lava")
	case lqHoney:
		if g.Turn%5 == 0 {
			p.Energy -= 300
			g.Msg("{{Y|Honey clings to your feet.}}")
		}
	}
	// puddles of water dry up
	if g.Turn%60 == 0 {
		for i, id := range z.Liq {
			if id != lqFresh && id != lqSalt && id != lqBrackish && id != lqTainted {
				continue
			}
			if HashStr(fmt.Sprintf("%s:evap:%s:%d:%d", g.Seed, z.Key, g.Turn/60, i))%100 < 8 {
				z.Liq[i] = 0
			}
		}
	}
	if g.Turn%60 == 0 {
		if st := g.lq(); st.Surplus > 0 {
			st.Surplus = maxi(0, st.Surplus-150)
		}
	}
}

// lqZone scatters puddles on the surface and lava deep underground (side rng, never the zone's).
func lqZone(g *Game, zn *Zone, _ *Rng) {
	r := NewRng(fmt.Sprintf("%s:lq:%s", g.Seed, zn.Key))
	var kind uint8
	n := 0
	switch {
	case zn.Z == 0 && (zn.Kind == "surface" || zn.Kind == "outskirts") && !zn.Underwater:
		if !r.Chance(0.3) {
			return
		}
		kind, n = lqFresh, r.Int(2, 6)
		switch g.lqSaltAt(zn) {
		case 1:
			kind = lqBrackish
		case 2:
			return
		}
		if g.W.In(zn.WX, zn.WY) && g.W.Temp[g.W.Idx(zn.WX, zn.WY)] >= 30 && kind == lqFresh && r.Chance(0.6) {
			kind = lqBrackish
		}
	case zn.Z >= 3:
		if !r.Chance(0.15) {
			return
		}
		kind, n = lqLava, r.Int(2, 5)
	case zn.Z >= 1:
		if !r.Chance(0.12) {
			return
		}
		kind, n = lqFresh, r.Int(2, 4)
	default:
		return
	}
	cx, cy := r.Int(5, MW-6), r.Int(4, MH-5)
	for i := 0; i < n*3 && n > 0; i++ {
		x, y := cx+r.Int(-2, 2), cy+r.Int(-1, 1)
		if !zn.In(x, y) {
			continue
		}
		switch zn.At(x, y) {
		case TFloor, TGrass, TTallGrass, TSand, TDirt, TMoss, TAsh, TSnow, TFungus:
			if zn.Liq[y*MW+x] == 0 && zn.Fire[y*MW+x] == 0 {
				zn.Liq[y*MW+x] = kind
				n--
			}
		}
	}
}

func lqSide(g *Game) []string {
	var out []string
	if d := g.lqDrams(); d > 0 {
		out = append(out, fmt.Sprintf("{{B|Water: %d drams}}", d))
	}
	if k, _, _, ok := g.lqPoolNear(); ok && g.Cur.Liq[g.P.Y*MW+g.P.X] > 0 {
		out = append(out, "{{K|In "+g.lqName(k)+"}}")
	}
	return out
}

func init() {
	mainKeys["P"] = func(g *Game) { g.lq().Sel = 0; g.lq().Mix = false; g.Mode = "liquid" }
	modeKeys["liquid"] = lqKeys
	modeDraws["liquid"] = lqDraw
	turnHooks = append(turnHooks, lqPoolTick, lqSurvivalTick)
	zoneHooks = append(zoneHooks, lqZone)
	sideHooks = append(sideHooks, lqSide)
	talkHooks = append(talkHooks, lqTalk)
}

// ---- screen ----

func lqKeys(g *Game, k string) {
	st := g.lq()
	list := g.lqContainers()
	if st.Sel >= len(list) {
		st.Sel = maxi(0, len(list)-1)
	}
	pick := func(i int) {
		if i < 0 || i >= len(list) {
			return
		}
		if st.Mix {
			st.Mix = false
			g.lqPourInto(list[st.Sel], list[i])
			return
		}
		st.Sel = i
	}
	switch k {
	case "Escape":
		if st.Mix {
			st.Mix = false
			return
		}
		g.Mode = "main"
		return
	case "ArrowUp":
		st.Sel = maxi(0, st.Sel-1)
		return
	case "ArrowDown":
		st.Sel = mini(maxi(0, len(list)-1), st.Sel+1)
		return
	}
	if st.Mix {
		if k == "Enter" {
			pick(st.Sel)
		} else if i := letterIndex(k); i >= 0 {
			pick(i)
		}
		return
	}
	if len(list) == 0 {
		if k == "e" {
			g.lqForage()
		}
		return
	}
	cur := list[st.Sel]
	switch k {
	case "1", "2", "3", "4", "5", "6", "7", "8", "9":
		pick(int(k[0] - '1'))
	case "d":
		if cur.Charges <= 0 {
			g.Msg("It is empty.")
		} else if lqKey(cur) == "fresh" {
			cur.Charges--
			g.lqDrinkEffect("fresh", 1)
			g.Msg(fmt.Sprintf("You drink a dram of water. (%d left)", cur.Charges))
			g.lqRename(cur)
			g.spend(1000)
		} else {
			g.lqDrinkItem(cur)
			g.spend(1000)
		}
	case "p":
		g.lqPourOut(cur)
	case "f":
		g.lqFill(cur)
	case "m":
		if cur.Charges <= 0 {
			g.Msg("It is empty.")
		} else if len(list) < 2 {
			g.Msg("You need a second container to pour into.")
		} else {
			st.Mix = true
			g.Msg("Pour into which container? (1-9)")
		}
	case "e":
		g.lqForage()
	}
}

func lqDraw(g *Game) {
	s := g.sc
	x, y, w, _ := g.panel("Liquids")
	st := g.lq()
	line := func(txt string) {
		for _, l := range Wrap(txt, w) {
			s.TextB(x, y, l, CGray, CBgDark, x+w)
			y++
		}
	}
	line(fmt.Sprintf("{{B|Fresh water on you: %d drams}}  {{K|(4 drams weigh 1 lb; a dram is also coin.)}}", g.lqDrams()))
	if k, _, _, ok := g.lqPoolNear(); ok {
		line("Nearby: {{C|" + g.lqLook(k) + "}}")
	}
	y++
	list := g.lqContainers()
	if len(list) == 0 {
		line("{{K|You carry no waterskin or flask.}}")
	}
	for i, it := range list {
		bg := byte(CBgDark)
		if i == st.Sel {
			bg = CBgSel
		}
		what := "empty"
		if it.Charges > 0 {
			what = g.lqName(lqKey(it))
		}
		txt := fmt.Sprintf("%d) %-28s %2d/%-2d  %s", i+1, it.Name, it.Charges, it.Power, what)
		s.TextB(x, y, txt, CGray, bg, x+w)
		y++
	}
	y++
	if st.Mix {
		line("{{W|Pour into which container?}} (number, Esc cancels)")
		return
	}
	line("{{W|d}} drink  {{W|f}} fill from the pool or well beside you  {{W|p}} pour out on the ground  {{W|m}} pour into another container (mixing spoils water)  {{W|e}} forage for water (Survival)  {{W|Esc}} close")
	y++
	if len(list) > 0 && st.Sel < len(list) && list[st.Sel].Charges > 0 {
		l := lqByKey(lqKey(list[st.Sel]))
		d := l.Drink
		if l.Key == "genre" {
			d = "unusual"
		}
		line("{{K|" + g.lqName(l.Key) + ": " + d + "}}")
	}
}
