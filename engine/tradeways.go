package engine

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// Trade routes, caravan contracts, smuggling and market events (prefix tq).
//
//   - Bulk goods (tradeways_data.go) are Items of Kind "goods" (Count, Weight = unit weight x Count).
//     Prices per town come from Realmforge sector shares (producers cheap, demanders dear), a slow
//     per-town drift, market events, wars, quarantines and your own trading volume. A factor (Role
//     "factor", market and harbour districts) and every merchant open the bulk-goods screen.
//   - Market events run on the day clock from side rngs only (glut, shortage, festival, bumper
//     harvest, blockade, embargo) and feed tqNews() for the news screen.
//   - Caravan contracts (haul N goods to town B by day D, or hire on as a guard) hit you with road
//     events through tqOnMove; settled automatically at the destination gate.
//   - Contraband = goods banned by the realm's statutes (crime.go laws) or an embargo. Gate customs
//     roll to find it; bribe, forged writ, or lose it plus fine and bounty. Fences buy it dear.
//   - F6 opens the Trade ledger. State g.ext("tq"), planet-local ("tq" in unLocalKeys).

func init() {
	unLocalKeys = append(unLocalKeys, "tq")
	zoneHooks = append(zoneHooks, tqZone)
	enterHooks = append(enterHooks, tqEnter)
	turnHooks = append(turnHooks, tqTurn)
	talkHooks = append(talkHooks, tqTalk)
	mainKeys["F6"] = func(g *Game) { g.Mode = "tqledger"; g.Sel = 0 }
	modeKeys["tqledger"] = tqLedgerKey
	modeDraws["tqledger"] = tqLedgerDraw
	modeKeys["tq"] = tqMarketKey
	modeDraws["tq"] = tqMarketDraw
	modeKeys["tqmenu"] = func(g *Game, k string) { g.keyMenu(k, tqMenuEsc(g)) }
	modeDraws["tqmenu"] = tqMenuDraw
}

// ---------------------------------------------------------------- state

type tqEvent struct {
	Kind  string  `json:"k"` // glut shortage festival bumper blockade embargo
	Good  string  `json:"g,omitempty"`
	Burg  int     `json:"b,omitempty"`
	Realm int     `json:"r,omitempty"`
	P     int     `json:"p"`
	Start int     `json:"s"`
	End   int     `json:"e"`
	Mul   float64 `json:"m"`
	Text  string  `json:"t"`
}

type tqContract struct {
	ID    int    `json:"id"`
	Kind  string `json:"k"` // haul guard
	Good  string `json:"g,omitempty"`
	N     int    `json:"n"`
	From  int    `json:"f"`
	To    int    `json:"t"`
	Due   int    `json:"d"`
	Fee   int    `json:"fee"`
	Bonus int    `json:"bo"`
	Dmg   bool   `json:"dm,omitempty"`
	Wagon bool   `json:"wg,omitempty"`
}

type tqVol struct {
	N   int `json:"n"`
	Day int `json:"d"`
}

type tqState struct {
	Day      int                          `json:"day"`
	Ev       []*tqEvent                   `json:"ev"`
	Ct       []*tqContract                `json:"ct"`
	Taken    map[int]bool                 `json:"tk"`
	Mem      map[string]map[string][3]int `json:"mem"`
	Vol      map[string]*tqVol            `json:"vol"`
	Pts      int                          `json:"pts"`
	Profit   int                          `json:"pr"`
	Trades   int                          `json:"tr"`
	Done     int                          `json:"dn"`
	Failed   int                          `json:"fl"`
	Smuggled int                          `json:"sm"`
	Caught   int                          `json:"ca"`
	Last     int                          `json:"last"`
	Deeds    map[string]bool              `json:"dd"`
	Wagon    bool                         `json:"wag,omitempty"`
}

func (g *Game) tq() *tqState {
	s := g.ext("tq", func() any { return &tqState{} }).(*tqState)
	if s.Taken == nil {
		s.Taken = map[int]bool{}
	}
	if s.Mem == nil {
		s.Mem = map[string]map[string][3]int{}
	}
	if s.Vol == nil {
		s.Vol = map[string]*tqVol{}
	}
	if s.Deeds == nil {
		s.Deeds = map[string]bool{}
	}
	return s
}

func (g *Game) tqToday() int { return g.Turn / 1440 }

func (g *Game) tqDeed(key, text string) {
	s := g.tq()
	if s.Deeds[key] {
		return
	}
	s.Deeds[key] = true
	g.Msg("{{Y|" + text + "}}")
	if g.Cur != nil {
		g.deed("trade", text, 0, 0)
	}
}

func (s *tqState) rank() int {
	r := 0
	for i, k := range tqRanks {
		if s.Pts >= k.Pts {
			r = i
		}
	}
	return r
}

func (g *Game) tqAddPts(n int) {
	s := g.tq()
	before := s.rank()
	s.Pts += n
	if s.rank() > before {
		g.tqDeed("rank"+strconv.Itoa(s.rank()), "Your name in trade is now: "+tqRanks[s.rank()].Title+".")
	}
}

func (g *Game) tqBurg() *BurgData {
	if g.Cur == nil || g.Cur.Burg <= 0 || g.Cur.Kind != "town" {
		return nil
	}
	return g.W.burgByID[g.Cur.Burg]
}

// ---------------------------------------------------------------- laws and prices

func (g *Game) tqBanned(state int, d *tqGood) bool {
	if d == nil {
		return false
	}
	day := g.tqToday()
	for _, e := range g.tq().Ev {
		if e.Kind == "embargo" && e.Realm == state && e.Good == d.ID && day >= e.Start && day <= e.End {
			return true
		}
	}
	if d.Ban == "" || state <= 0 {
		return false
	}
	l := g.uwLawsFor(state)
	switch d.Ban {
	case "drugs":
		return l.Drugs
	case "occult":
		return l.Occult
	case "heresy":
		return l.Heresy
	case "arms":
		return l.Arms
	case "augment":
		return l.Augment
	}
	return false
}

func (g *Game) tqEventMul(bd *BurgData, d *tqGood) float64 {
	m := 1.0
	day := g.tqToday()
	for _, e := range g.tq().Ev {
		if e.Kind == "embargo" || day < e.Start || day > e.End {
			continue
		}
		if !(e.Burg == bd.ID || (e.Burg == 0 && e.Realm == bd.State && e.Realm > 0)) {
			continue
		}
		if e.Good == "" || e.Good == d.ID {
			m *= e.Mul
		}
	}
	if g.W.hzQuarantine(bd.ID) != nil {
		if d.ID == "medicine" {
			m *= 1.8
		} else {
			m *= 1.2
		}
	}
	if (d.ID == "arms" || d.ID == "iron" || d.ID == "grain") && len(g.warsOf(bd.State)) > 0 {
		m *= 1.25
	}
	return m
}

func (g *Game) tqVolN(bd *BurgData, d *tqGood) int {
	v := g.tq().Vol[fmt.Sprintf("%d:%s", bd.ID, d.ID)]
	if v == nil {
		return 0
	}
	n := v.N
	dec := 3 * (g.tqToday() - v.Day)
	switch {
	case n > 0:
		n = maxi(0, n-dec)
	case n < 0:
		n = mini(0, n+dec)
	}
	return n
}

func (g *Game) tqVolAdd(bd *BurgData, d *tqGood, n int) {
	k := fmt.Sprintf("%d:%s", bd.ID, d.ID)
	cur := g.tqVolN(bd, d)
	g.tq().Vol[k] = &tqVol{N: cur + n, Day: g.tqToday()}
}

// tqPrice: price per unit of a good in a town. buy = what the clerk charges, else what he pays.
func (g *Game) tqPrice(bd *BurgData, d *tqGood, buy bool) int {
	w := g.W
	prod := fclamp(w.Sector(bd, d.Prod)*3, 0, 1)
	dem := fclamp(w.Sector(bd, d.Dem)*3, 0, 1)
	f := 1 - 0.45*prod + 0.4*dem
	if d.Port && bd.Port {
		f -= 0.15
	}
	f += wsHashF(g.Seed, "tqp", bd.ID, d.ID, g.tqToday()/4) * 0.12
	if d.Ws != "" {
		f *= g.wsPrice(bd.P, d.Ws)
	}
	f *= g.tqEventMul(bd, d)
	f *= fclamp(1+0.012*float64(g.tqVolN(bd, d)), 0.7, 1.5)
	f = fclamp(f, 0.4, 2.6)
	v := float64(d.Base) * f
	pr := 0.0
	if s := g.tq(); s.rank() >= 2 {
		pr = fclamp(0.02*float64(s.rank()), 0, 0.1)
	}
	if buy {
		v *= 1.1 * (1 - pr)
	} else {
		v *= 0.9 * (1 + pr)
	}
	return maxi(1, int(v))
}

func (g *Game) tqCanBuy(bd *BurgData, d *tqGood) bool {
	if g.tqBanned(bd.State, d) {
		return false
	}
	if g.W.Sector(bd, d.Prod) > 0.03 {
		return true
	}
	return HashStr(fmt.Sprintf("%s:tq:stock:%d:%s", g.Seed, bd.ID, d.ID))%100 < 40
}

func (g *Game) tqRemember(bd *BurgData) {
	s := g.tq()
	m := map[string][3]int{}
	for _, d := range g.tqGoodsHere() {
		m[d.ID] = [3]int{g.tqPrice(bd, d, true), g.tqPrice(bd, d, false), g.tqToday()}
	}
	s.Mem[strconv.Itoa(bd.ID)] = m
}

// ---------------------------------------------------------------- goods items

func tqIdx(d *tqGood) int {
	for i := range tqGoods {
		if &tqGoods[i] == d {
			return i
		}
	}
	return 0
}

func tqItemGood(it *Item) *tqGood {
	if it == nil || it.Kind != "goods" || it.Extra == nil || it.Extra["tq"] != 1 {
		return nil
	}
	i := it.Extra["tq:i"]
	if i < 0 || i >= len(tqGoods) {
		return nil
	}
	return &tqGoods[i]
}

func (g *Game) tqSync(it *Item) {
	d := tqItemGood(it)
	if d == nil {
		return
	}
	n := maxi(it.Count, 1)
	it.Weight, it.Value = d.Wt*n, d.Base*n
	it.Name = g.tqName(d)
	it.Glyph, it.F = '=', CYellow
}

func (g *Game) tqNewItem(d *tqGood, n, unit, contract int) *Item {
	it := &Item{ID: g.newID(), Kind: "goods", Count: n, Extra: map[string]int{"tq": 1, "tq:i": tqIdx(d), "tq:u": unit, "tq:d": g.tqToday(), "tq:c": contract}}
	it.Text = "Bulk trade goods. Spoils if perishable; may be contraband in some realms."
	g.tqSync(it)
	return it
}

func tqMerge(inv []*Item, d *tqGood, n, unit, contract, day int) *Item {
	for _, o := range inv {
		if od := tqItemGood(o); od == d && o.Extra["tq:c"] == contract {
			tot := o.Count + n
			if tot > 0 {
				o.Extra["tq:u"] = (o.Extra["tq:u"]*o.Count + unit*n) / tot
			}
			if day < o.Extra["tq:d"] {
				o.Extra["tq:d"] = day
			}
			o.Count = tot
			return o
		}
	}
	return nil
}

func (g *Game) tqAdd(d *tqGood, n, unit, contract int, hold bool) {
	if hold {
		if sp := g.sh().Ship; sp != nil {
			if o := tqMerge(sp.Hold, d, n, unit, contract, g.tqToday()); o != nil {
				g.tqSync(o)
				return
			}
			sp.Hold = append(sp.Hold, g.tqNewItem(d, n, unit, contract))
			return
		}
	}
	if o := tqMerge(g.P.Inv, d, n, unit, contract, g.tqToday()); o != nil {
		g.tqSync(o)
		return
	}
	g.P.Inv = append(g.P.Inv, g.tqNewItem(d, n, unit, contract))
}

func tqDrop(inv []*Item, it *Item) []*Item {
	out := inv[:0:0]
	for _, o := range inv {
		if o != it {
			out = append(out, o)
		}
	}
	return out
}

func (g *Game) tqHeld(d *tqGood) int {
	n := 0
	for _, it := range g.P.Inv {
		if tqItemGood(it) == d && it.Extra["tq:c"] == 0 {
			n += it.Count
		}
	}
	if sp := g.sh().Ship; sp != nil {
		for _, it := range sp.Hold {
			if tqItemGood(it) == d && it.Extra["tq:c"] == 0 {
				n += it.Count
			}
		}
	}
	return n
}

// tqHoldFree: free cargo slots in your ship's hold (0 without a ship). Eight units fill a slot.
func (g *Game) tqHoldFree() int {
	sp := g.sh().Ship
	if sp == nil || sp.K() == nil {
		return 0
	}
	used := 0
	for _, it := range sp.Hold {
		used += (maxi(it.Count, 1) + 7) / 8
	}
	return maxi(0, sp.K().Hold-used)
}

func (g *Game) tqShip() *shShip {
	sp := g.sh().Ship
	if sp != nil && g.Cur != nil && g.Cur.Burg > 0 && sp.Port == g.Cur.Burg {
		return sp
	}
	return nil
}

// tqTake removes up to n units of a free (non-contract) good, hold first when holdFirst.
// It returns the units taken and their total cost basis.
func (g *Game) tqTake(d *tqGood, n int, holdFirst bool) (int, int) {
	took, basis := 0, 0
	pull := func(inv *[]*Item) {
		for _, it := range append([]*Item(nil), *inv...) {
			if took >= n {
				return
			}
			if tqItemGood(it) != d || it.Extra["tq:c"] != 0 {
				continue
			}
			k := mini(it.Count, n-took)
			took += k
			basis += k * it.Extra["tq:u"]
			it.Count -= k
			if it.Count <= 0 {
				*inv = tqDrop(*inv, it)
			} else {
				g.tqSync(it)
			}
		}
	}
	sp := g.sh().Ship
	if holdFirst && sp != nil {
		pull(&sp.Hold)
	}
	pull(&g.P.Inv)
	if sp != nil {
		pull(&sp.Hold)
	}
	return took, basis
}

func (g *Game) tqHasWrit() *Item {
	for _, it := range g.P.Inv {
		if it.Extra != nil && it.Extra["tq:writ"] > 0 {
			return it
		}
	}
	return nil
}

// ---------------------------------------------------------------- buy and sell

var tqHoldMode bool

func (g *Game) tqBuy(bd *BurgData, d *tqGood, n int) {
	if !g.tqCanBuy(bd, d) {
		if g.tqBanned(bd.State, d) {
			g.Msg("{{R|The clerk will not sell that here: it is banned.}}")
		} else {
			g.Msg("None in stock.")
		}
		return
	}
	bought, spent := 0, 0
	for i := 0; i < n; i++ {
		pr := g.tqPrice(bd, d, true)
		toHold := tqHoldMode && g.tqShip() != nil
		if toHold {
			free := g.tqHoldFree()
			part := 0
			if sp := g.sh().Ship; sp != nil {
				for _, it := range sp.Hold {
					if tqItemGood(it) == d && it.Extra["tq:c"] == 0 {
						part = it.Count % 8
					}
				}
			}
			if free <= 0 && part == 0 {
				g.Msg("{{y|The hold is full.}}")
				break
			}
		}
		if !g.cuSpend(pr) {
			if bought == 0 {
				g.Msg("You can't afford that.")
			}
			break
		}
		g.tqAdd(d, 1, pr, 0, toHold)
		g.tqVolAdd(bd, d, 1)
		bought++
		spent += pr
	}
	if bought > 0 {
		g.tq().Trades++
		g.Msg(fmt.Sprintf("You buy %d %s for %s.", bought, g.tqName(d), g.cuFormat(spent)))
		g.tqRemember(bd)
	}
}

func (g *Game) tqSell(bd *BurgData, d *tqGood, n int) {
	if g.tqBanned(bd.State, d) {
		g.Msg("{{R|\"I won't touch that. Take it away before the watch sees.\"}}")
		return
	}
	got, basis := 0, 0
	rev := 0
	for i := 0; i < n && g.tqHeld(d) > 0; i++ {
		pr := g.tqPrice(bd, d, false)
		k, b := g.tqTake(d, 1, tqHoldMode)
		if k == 0 {
			break
		}
		got += k
		basis += b
		rev += pr
		g.tqVolAdd(bd, d, -1)
	}
	if got == 0 {
		g.Msg("You have none to sell.")
		return
	}
	g.tqPaid(bd, rev, basis, got, d)
}

func (g *Game) tqPaid(bd *BurgData, rev, basis, units int, d *tqGood) {
	s := g.tq()
	pay := g.cuEarn(rev)
	profit := rev - basis
	s.Profit += profit
	s.Trades++
	col := "G"
	if profit < 0 {
		col = "R"
	}
	g.Msg(fmt.Sprintf("You sell %d %s for %s. {{%s|(%+d profit)}}", units, g.tqName(d), pay, col, profit))
	if profit > 0 {
		g.tqAddPts(profit / 8)
		g.tqDeed("profit", "You turned your first profit in bulk goods.")
		if s.Profit >= 1000 {
			g.tqDeed("p1000", "Your trading profits passed a thousand.")
		}
	}
	g.tqRemember(bd)
}

// ---------------------------------------------------------------- market screen

func tqMarketKey(g *Game, k string) {
	bd := g.tqBurg()
	gs := g.tqGoodsHere()
	if bd == nil || len(gs) == 0 {
		g.Mode = "main"
		return
	}
	g.Sel = clampi(g.Sel, 0, len(gs)-1)
	d := gs[g.Sel]
	switch k {
	case "Escape", "q":
		g.Mode = "main"
	case "ArrowUp", "k":
		g.Sel = maxi(0, g.Sel-1)
	case "ArrowDown", "j":
		g.Sel = mini(len(gs)-1, g.Sel+1)
	case "b", "Enter":
		g.tqBuy(bd, d, 1)
	case "B":
		g.tqBuy(bd, d, 10)
	case "s":
		g.tqSell(bd, d, 1)
	case "S":
		g.tqSell(bd, d, 999)
	case "h":
		if g.tqShip() != nil {
			tqHoldMode = !tqHoldMode
		} else {
			g.Msg("No ship of yours is moored here.")
		}
	case "F6":
		g.Mode = "tqledger"
		g.Sel = 0
	}
}

func tqMarketDraw(g *Game) {
	s := g.sc
	bd := g.tqBurg()
	if bd == nil {
		return
	}
	gs := g.tqGoodsHere()
	s.Box(2, 1, SW-4, SH-2, CYellow, "Bulk goods: "+bd.Name)
	g.Sel = clampi(g.Sel, 0, maxi(len(gs)-1, 0))
	y := 3
	s.TextB(4, y, fmt.Sprintf("Purse: %s   Trader rank: {{Y|%s}}   Hold: %s", g.cuWallet(), tqRanks[g.tq().rank()].Title, tqHoldLabel(g)), CGray, CBgDark, SW-3)
	y += 2
	s.TextB(4, y, fmt.Sprintf("{{W|%-24s %-18s %-18s %5s  %s}}", "Good", "Buy", "Sell", "Held", "Market"), CGray, CBgDark, SW-3)
	y++
	for i, d := range gs {
		bg := CBgDark
		if i == g.Sel {
			bg = CBgSel
		}
		s.Fill(3, y, SW-6, 1, bg)
		buy, sell := g.tqPrice(bd, d, true), g.tqPrice(bd, d, false)
		bs := g.cuFormat(buy)
		note := ""
		switch {
		case g.tqBanned(bd.State, d):
			bs, note = "-", "{{R|BANNED here}}"
		case !g.tqCanBuy(bd, d):
			bs, note = "-", "{{K|not stocked}}"
		default:
			r := float64(buy) / float64(maxi(d.Base, 1))
			switch {
			case r < 0.85:
				note = "{{G|cheap}}"
			case r > 1.3:
				note = "{{O|dear}}"
			}
		}
		s.TextB(4, y, fmt.Sprintf("%-24s %-18s %-18s %5d  %s", g.tqName(d), bs, g.cuFormat(sell), g.tqHeld(d), note), CGray, bg, SW-3)
		y++
	}
	y += 1
	for _, e := range g.tqLocalEvents(bd) {
		s.TextB(4, y, e.Text, CGray, CBgDark, SW-3)
		y++
	}
	s.TextB(4, SH-3, "{{W|Up/Down}} pick  {{W|b/B}} buy 1/10  {{W|s/S}} sell 1/all  {{W|h}} ship hold  {{W|F6}} ledger  {{W|Esc}} leave", CGray, CBgDark, SW-3)
}

func tqHoldLabel(g *Game) string {
	if g.sh().Ship == nil {
		return "none"
	}
	m := "pack"
	if tqHoldMode && g.tqShip() != nil {
		m = "ship"
	}
	return fmt.Sprintf("%d slots free (buying into %s)", g.tqHoldFree(), m)
}

func (g *Game) tqLocalEvents(bd *BurgData) []*tqEvent {
	var out []*tqEvent
	day := g.tqToday()
	for _, e := range g.tq().Ev {
		if day >= e.Start && day <= e.End && (e.Burg == bd.ID || (e.Burg == 0 && e.Realm == bd.State)) {
			out = append(out, e)
		}
	}
	return out
}

// ---------------------------------------------------------------- events (day clock)

func tqTurn(g *Game) {
	if g.Turn%60 != 0 || g.W == nil {
		return
	}
	s := g.tq()
	day := g.tqToday()
	if day <= s.Day {
		return
	}
	bl := g.ws().burgList(g)
	from := maxi(s.Day+1, day-12)
	for d := from; d <= day; d++ {
		g.tqGenEvent(d, bl)
	}
	s.Day = day
	var keep []*tqEvent
	for _, e := range s.Ev {
		if e.End >= day-1 {
			keep = append(keep, e)
		}
	}
	if len(keep) > 14 {
		keep = keep[len(keep)-14:]
	}
	s.Ev = keep
	g.tqSpoil(day)
	g.tqExpire(day)
}

func (g *Game) tqGenEvent(day int, bl []*BurgData) {
	if len(bl) == 0 {
		return
	}
	r := NewRng(fmt.Sprintf("%s:tq:ev:%d", g.Seed, day))
	if !r.Chance(0.45) {
		return
	}
	bd := bl[r.Int(0, len(bl)-1)]
	gs := g.tqGoodsHere()
	d := gs[r.Int(0, len(gs)-1)]
	e := &tqEvent{Burg: bd.ID, P: bd.P, Start: day, Realm: bd.State}
	dur := r.Int(6, 14)
	rn := g.realmName(bd.State)
	switch r.Int(0, 5) {
	case 0:
		e.Kind, e.Good, e.Mul = "glut", d.ID, 0.6
		e.Text = fmt.Sprintf("{{G|Glut}}: %s flood the market at %s; prices have collapsed.", g.tqName(d), bd.Name)
	case 1:
		e.Kind, e.Good, e.Mul = "shortage", d.ID, 1.7
		e.Text = fmt.Sprintf("{{O|Shortage}}: no %s to be had at %s; buyers pay dearly.", g.tqName(d), bd.Name)
	case 2:
		lux := []string{"wine", "spice", "silk", "dye", "books", "beasts"}
		id := lux[r.Int(0, len(lux)-1)]
		ld := tqGoodByID(id)
		if ld == nil || g.tqGoodIdxMissing(ld) {
			ld = d
		}
		e.Kind, e.Good, e.Mul, dur = "festival", ld.ID, 1.5, r.Int(4, 7)
		e.Text = fmt.Sprintf("{{Y|Festival}} at %s: demand for %s is high.", bd.Name, g.tqName(ld))
	case 3:
		gr := tqGoodByID("grain")
		e.Kind, e.Good, e.Burg, e.Mul, dur = "bumper", gr.ID, 0, 0.55, r.Int(10, 16)
		e.Text = fmt.Sprintf("{{G|Bumper harvest}} across %s: grain is cheap.", rn)
	case 4:
		e.Kind, e.Good, e.Mul, dur = "blockade", "", 1.4, r.Int(8, 12)
		if len(g.warsOf(bd.State)) > 0 {
			e.Text = fmt.Sprintf("{{R|Blockade}}: war closes the roads to %s; goods there are dear.", bd.Name)
		} else {
			e.Text = fmt.Sprintf("{{R|Blockade}}: floods and brigands cut the roads to %s.", bd.Name)
		}
	default:
		e.Kind, e.Good, e.Burg, e.Mul, dur = "embargo", d.ID, 0, 1, r.Int(15, 25)
		e.Text = fmt.Sprintf("{{R|Embargo}}: %s bans %s outright; customs will be hunting for it.", rn, g.tqName(d))
	}
	e.End = day + dur
	g.tq().Ev = append(g.tq().Ev, e)
}

func (g *Game) tqGoodIdxMissing(d *tqGood) bool {
	for _, x := range g.tqGoodsHere() {
		if x == d {
			return false
		}
	}
	return true
}

// tqNews feeds wsNewsLines (report hook).
func (g *Game) tqNews() []wsLine {
	var out []wsLine
	day := g.tqToday()
	for _, e := range g.tq().Ev {
		if day >= e.Start && day <= e.End {
			out = append(out, wsLine{e.P, e.Text})
		}
	}
	return out
}

func (g *Game) tqSpoil(day int) {
	lost := 0
	rot := func(inv []*Item, rate int) []*Item {
		for _, it := range append([]*Item(nil), inv...) {
			d := tqItemGood(it)
			if d == nil || d.Perish == 0 || day-it.Extra["tq:d"] <= d.Perish {
				continue
			}
			k := maxi(1, it.Count*rate/100)
			k = mini(k, it.Count)
			it.Count -= k
			lost += k
			if c := it.Extra["tq:c"]; c != 0 {
				for _, ct := range g.tq().Ct {
					if ct.ID == c {
						ct.Dmg = true
					}
				}
			}
			if it.Count <= 0 {
				inv = tqDrop(inv, it)
			} else {
				g.tqSync(it)
			}
		}
		return inv
	}
	g.P.Inv = rot(g.P.Inv, 8)
	if sp := g.sh().Ship; sp != nil {
		sp.Hold = rot(sp.Hold, 4)
	}
	if lost > 0 {
		g.Msg(fmt.Sprintf("{{y|%d units of your goods have spoiled.}}", lost))
	}
}

// ---------------------------------------------------------------- caravan contracts

func (g *Game) tqOffers(bd *BurgData) []*tqContract {
	day := g.tqToday()
	bl := g.ws().burgList(g)
	type bdist struct {
		b *BurgData
		d int
	}
	var near []bdist
	for _, o := range bl {
		d := wsCheb(g.W, bd.P, o.P)
		if o.ID != bd.ID && d >= 3 && d <= 20 {
			near = append(near, bdist{o, d})
		}
	}
	if len(near) == 0 {
		return nil
	}
	sort.SliceStable(near, func(i, j int) bool {
		if near[i].d != near[j].d {
			return near[i].d < near[j].d
		}
		return near[i].b.ID < near[j].b.ID
	})
	if len(near) > 6 {
		near = near[:6]
	}
	var best *tqGood
	bs := -1.0
	for _, d := range g.tqGoodsHere() {
		if g.tqBanned(bd.State, d) {
			continue
		}
		if v := g.W.Sector(bd, d.Prod); v > bs {
			best, bs = d, v
		}
	}
	if best == nil {
		return nil
	}
	var out []*tqContract
	for i := 0; i < 3; i++ {
		r := NewRng(fmt.Sprintf("%s:tq:ct:%d:%d:%d", g.Seed, bd.ID, day/5, i))
		to := near[r.Int(0, len(near)-1)]
		id := int(HashStr(fmt.Sprintf("%s:tq:cid:%d:%d:%d", g.Seed, bd.ID, day/5, i))%900000) + 100
		if g.tq().Taken[id] {
			continue
		}
		c := &tqContract{ID: id, Kind: "haul", Good: best.ID, From: bd.ID, To: to.b.ID, Due: (day/5)*5 + 5 + 6 + to.d*4}
		if g.tqBanned(to.b.State, best) {
			continue
		}
		if i == 2 {
			c.Kind, c.Good = "guard", ""
			c.Fee = 40 + to.d*35
			c.Bonus = c.Fee / 5
		} else {
			c.N = r.Int(6, 16)
			c.Fee = int(float64(c.N*best.Base) * (0.35 + 0.04*float64(to.d)))
			c.Bonus = c.Fee / 4
		}
		out = append(out, c)
	}
	return out
}

func (g *Game) tqContractText(c *tqContract) string {
	from, to := g.W.burgByID[c.From], g.W.burgByID[c.To]
	if from == nil || to == nil {
		return "a lost contract"
	}
	late := c.Due - g.tqToday()
	if c.Kind == "guard" {
		return fmt.Sprintf("Guard the caravan %s -> %s, %d days. Fee %s (+%s if no losses).", from.Name, to.Name, late, g.cuFormat(c.Fee), g.cuFormat(c.Bonus))
	}
	d := tqGoodByID(c.Good)
	return fmt.Sprintf("Haul %d %s %s -> %s within %d days. Fee %s (+%s if undamaged).", c.N, g.tqName(d), from.Name, to.Name, late, g.cuFormat(c.Fee), g.cuFormat(c.Bonus))
}

func (g *Game) tqAccept(c *tqContract) {
	s := g.tq()
	if len(s.Ct) >= 3 {
		g.Msg("{{y|You carry enough contracts already.}}")
		return
	}
	s.Taken[c.ID] = true
	cp := *c
	s.Ct = append(s.Ct, &cp)
	if cp.Kind == "haul" {
		d := tqGoodByID(cp.Good)
		g.tqAdd(d, cp.N, 0, cp.ID, false)
		if g.tr().Cart == "" {
			g.tr().Cart = "handcart"
			s.Wagon = true
			g.Msg("The caravan master lends you a handcart for the load.")
		}
	}
	g.Msg("{{G|Contract taken:}} " + g.tqContractText(&cp))
	g.tqDeed("firstct", "You signed on to your first caravan contract.")
}

func (g *Game) tqContractMenu(a *Actor) {
	bd := g.tqBurg()
	if bd == nil {
		g.talkMenu("\"No caravans leave from here.\"")
		return
	}
	offers := g.tqOffers(bd)
	var opts []Option
	for _, c := range offers {
		c := c
		opts = append(opts, Option{g.tqContractText(c), func() {
			g.tqAccept(c)
			g.Mode = "main"
			g.Talk = nil
		}})
	}
	opts = append(opts, Option{"Never mind.", func() { g.Mode = "main"; g.Talk = nil }})
	txt := "\"These loads need hands. Late, damaged or lost cargo costs you.\""
	if len(offers) == 0 {
		txt = "\"Nothing going out this week.\""
	}
	g.tqOpenMenu("Caravan contracts", txt, opts, false)
}

func (g *Game) tqExpire(day int) {
	s := g.tq()
	var keep []*tqContract
	for _, c := range s.Ct {
		if day > c.Due+6 {
			s.Failed++
			g.tqDropCargo(c)
			g.Msg("{{R|The contract to " + g.tqTownName(c.To) + " has lapsed; the caravan house writes you off.}}")
			s.Pts = maxi(0, s.Pts-10)
			continue
		}
		keep = append(keep, c)
	}
	s.Ct = keep
}

func (g *Game) tqTownName(id int) string {
	if b := g.W.burgByID[id]; b != nil {
		return b.Name
	}
	return "somewhere"
}

func (g *Game) tqDropCargo(c *tqContract) {
	g.P.Inv = func() []*Item {
		out := g.P.Inv[:0:0]
		for _, it := range g.P.Inv {
			if it.Extra != nil && it.Extra["tq:c"] == c.ID {
				continue
			}
			out = append(out, it)
		}
		return out
	}()
	g.tqReturnCart()
}

func (g *Game) tqReturnCart() {
	s := g.tq()
	if !s.Wagon {
		return
	}
	for _, c := range s.Ct {
		if c.Kind == "haul" {
			return
		}
	}
	if g.tr().Cart == "handcart" {
		g.tr().Cart = ""
	}
	s.Wagon = false
}

// tqSettle pays the contracts that end in this town.
func (g *Game) tqSettle(bd *BurgData) {
	s := g.tq()
	day := g.tqToday()
	var keep []*tqContract
	for _, c := range s.Ct {
		if c.To != bd.ID {
			keep = append(keep, c)
			continue
		}
		got := c.N
		if c.Kind == "haul" {
			got = 0
			for _, it := range g.P.Inv {
				if it.Extra != nil && it.Extra["tq:c"] == c.ID {
					got += it.Count
				}
			}
			if got < c.N {
				c.Dmg = true
			}
		}
		pay := c.Fee
		if c.Kind == "haul" {
			pay = c.Fee * got / maxi(c.N, 1)
		}
		late := day > c.Due
		if late {
			pay /= 2
		}
		if !c.Dmg && !late {
			pay += c.Bonus
		}
		g.tqDropCargoNoCart(c)
		out := g.cuEarn(pay)
		s.Profit += pay
		s.Done++
		g.tqAddPts(20 + pay/20)
		tag := ""
		if late {
			tag = " (late: half fee)"
		} else if c.Dmg {
			tag = " (no bonus: losses)"
		}
		g.Msg(fmt.Sprintf("{{G|The factor at %s settles your contract:}} %s%s.", bd.Name, out, tag))
		g.tqDeed("ct1", "You completed a caravan contract.")
		if s.Done >= 10 {
			g.tqDeed("ct10", "Ten caravan contracts delivered.")
		}
	}
	s.Ct = keep
	g.tqReturnCart()
}

func (g *Game) tqDropCargoNoCart(c *tqContract) {
	out := g.P.Inv[:0:0]
	for _, it := range g.P.Inv {
		if it.Extra != nil && it.Extra["tq:c"] == c.ID {
			continue
		}
		out = append(out, it)
	}
	g.P.Inv = out
}

// tqOnMove: one-line hook for WorldMove (after wsOnMove). Road events for contracts.
func tqOnMove(g *Game) {
	s := g.tq()
	if len(s.Ct) == 0 || g.W == nil {
		return
	}
	r := NewRng(fmt.Sprintf("%s:tq:mv:%d:%d:%d", g.Seed, g.Turn, g.WX, g.WY))
	p := 0.035
	if !r.Chance(p) {
		return
	}
	c := s.Ct[r.Int(0, len(s.Ct)-1)]
	switch r.Int(0, 3) {
	case 0:
		mod := (g.P.Score(STR)-10)/2 + g.P.Level/3
		if c.Kind == "guard" {
			mod += 2
		}
		if g.roll2d6Check(mod, dcMedium) {
			g.Msg("{{G|Bandits spring from the brush, but you and the wagons beat them back.}}")
			g.gainXP(30)
			return
		}
		c.Dmg = true
		g.Msg("{{R|Bandits ambush the caravan! You are hurt and part of the load is lost.}}")
		g.hurt(g.P, r.Int(2, 6), nil, "road bandits")
		g.tqLose(c, 0.15+0.15*r.Float())
	case 1:
		toll := 5 + g.tqCargoValue(c)/50
		if g.cuSpend(toll) {
			g.Msg(fmt.Sprintf("{{y|A toll-keeper takes %s for the road.}}", g.cuFormat(toll)))
		} else {
			c.Dmg = true
			g.Msg("{{y|You cannot pay the toll; the keeper seizes some of the load.}}")
			g.tqLose(c, 0.1)
		}
	case 2:
		g.Msg("{{y|Foul weather rots the load.}}")
		c.Dmg = true
		g.tqLose(c, 0.1)
	default:
		if g.W.In(g.WX, g.WY) {
			g.tqCustoms(g.W.State[g.W.Idx(g.WX, g.WY)], "a roadside inspection", 0)
		}
	}
}

func (g *Game) tqCargoValue(c *tqContract) int {
	v := 0
	for _, it := range g.P.Inv {
		if it.Extra != nil && it.Extra["tq:c"] == c.ID {
			v += it.Value
		}
	}
	return v
}

func (g *Game) tqLose(c *tqContract, frac float64) {
	for _, it := range append([]*Item(nil), g.P.Inv...) {
		if it.Extra != nil && it.Extra["tq:c"] == c.ID {
			k := maxi(1, int(float64(it.Count)*frac))
			it.Count -= k
			if it.Count <= 0 {
				g.P.Inv = tqDrop(g.P.Inv, it)
			} else {
				g.tqSync(it)
			}
		}
	}
}

// ---------------------------------------------------------------- customs and smuggling

type tqFound struct {
	It   *Item
	Hold bool
}

func (g *Game) tqCustoms(state int, where string, burg int) {
	var found []tqFound
	r := NewRng(fmt.Sprintf("%s:tq:cu:%d:%d:%s", g.Seed, g.Turn, state, where))
	chance := 0.6
	if g.tr().Cart != "" {
		chance -= 0.12
	}
	if g.roll2d6Check(g.checkMod("streetwise"), dcMedium) {
		chance -= 0.2
	}
	any := false
	scan := func(inv []*Item, hold bool) {
		for _, it := range inv {
			d := tqItemGood(it)
			if d == nil || it.Extra["tq:c"] != 0 || !g.tqBanned(state, d) {
				continue
			}
			any = true
			c := chance
			if hold {
				c *= 0.6
			}
			if r.Chance(c) {
				found = append(found, tqFound{it, hold})
			}
		}
	}
	scan(g.P.Inv, false)
	if sp := g.sh().Ship; sp != nil && burg > 0 && sp.Port == burg {
		scan(sp.Hold, true)
	}
	if !any {
		return
	}
	g.tq().Smuggled++
	if len(found) == 0 {
		g.Msg("{{G|The watch waves you through " + where + ". Your contraband stays hidden.}}")
		g.tqDeed("smug", "You slipped contraband past customs.")
		return
	}
	val := 0
	for _, f := range found {
		val += f.It.Value
	}
	caught := func(double bool) {
		fine := maxi(10, val/4)
		if double {
			fine *= 2
		}
		for _, f := range found {
			if f.Hold {
				if sp := g.sh().Ship; sp != nil {
					sp.Hold = tqDrop(sp.Hold, f.It)
				}
			} else {
				g.P.Inv = tqDrop(g.P.Inv, f.It)
			}
		}
		if !g.cuSpend(fine) {
			g.Msg("{{R|You cannot pay the fine.}}")
		} else {
			g.Msg(fmt.Sprintf("{{R|Contraband seized. Fine: %s.}}", g.cuFormat(fine)))
		}
		if state > 0 {
			g.uwAddBounty(state, 6+val/60, " (smuggling)")
		}
		g.tq().Caught++
		g.tqDeed("caught", "Customs caught you smuggling.")
		g.Mode, g.Talk = "main", nil
	}
	pass := func() {
		g.Msg("{{G|You are waved on.}}")
		g.Mode, g.Talk = "main", nil
	}
	var opts []Option
	if w := g.tqHasWrit(); w != nil {
		opts = append(opts, Option{"Show your forged writ.", func() {
			g.P.Inv = tqDrop(g.P.Inv, w)
			if r.Chance(0.7) {
				pass()
			} else {
				g.Msg("The clerk squints at the seal and calls the guards.")
				caught(true)
			}
		}})
	}
	bribe := maxi(20, val*35/100)
	opts = append(opts, Option{fmt.Sprintf("Slip them a bribe (%s).", g.cuFormat(bribe)), func() {
		if !g.cuSpend(bribe) {
			g.Msg("You cannot cover it.")
			caught(false)
			return
		}
		if r.Chance(0.7) {
			pass()
		} else {
			g.Msg("The officer pockets the coin and arrests you anyway.")
			caught(true)
		}
	}}, Option{"Hand it over.", func() { caught(false) }})
	g.tqOpenMenu("Customs", fmt.Sprintf("The watch searches your baggage at %s and finds banned goods worth about %s.", where, g.cuFormat(val)), opts, true)
}

// ---------------------------------------------------------------- hooks

func tqZone(g *Game, zn *Zone, _ *Rng) {
	if zn.Kind != "town" || zn.Z != 0 || zn.Burg <= 0 {
		return
	}
	bd := g.W.burgByID[zn.Burg]
	if bd == nil || !bd.Alive {
		return
	}
	if !(strings.Contains(zn.Name, "Market") || strings.Contains(zn.Name, "Harbor") || !strings.Contains(zn.Name, ",")) {
		return
	}
	g.trPlace(zn, bd, "factor", "factor", fmt.Sprintf("A factor of %s who brokers bulk goods and caravan contracts.", bd.Name), "")
	for _, a := range zn.Actors {
		if a.Role == "factor" {
			a.F = CYellow
		}
	}
}

func tqEnter(g *Game, zn *Zone) {
	s := g.tq()
	if zn.Kind != "town" || zn.Burg <= 0 || zn.Z != 0 {
		if zn.Kind != "town" {
			s.Last = 0
		}
		return
	}
	bd := g.W.burgByID[zn.Burg]
	if bd == nil || s.Last == bd.ID {
		return
	}
	s.Last = bd.ID
	g.tqRemember(bd)
	g.tqCustoms(bd.State, "the gate of "+bd.Name, bd.ID)
	g.tqSettle(bd)
}

func tqTalk(g *Game, a *Actor) []Option {
	var out []Option
	bd := g.tqBurg()
	if bd == nil {
		return nil
	}
	open := func() {
		g.Mode, g.Talk, g.Sel = "tq", nil, 0
		g.tqRemember(bd)
	}
	switch {
	case a.Role == "factor":
		out = append(out,
			Option{"{{Y|Trade in bulk goods.}}", open},
			Option{"{{Y|Ask about caravan contracts.}}", func() { g.tqContractMenu(a) }},
			Option{"{{Y|Open my trade ledger.}}", func() { g.Mode, g.Talk, g.Sel = "tqledger", nil, 0 }})
	case a.Role == "merchant":
		out = append(out, Option{"{{Y|Trade in bulk goods.}}", open},
			Option{"{{Y|Any caravan work going?}}", func() { g.tqContractMenu(a) }})
	case uwIsFence(a):
		out = append(out, Option{"{{M|Sell contraband by the crate.}}", func() { g.tqFenceMenu(a, bd) }},
			Option{"{{M|I need papers: a forged writ.}}", func() {
				if !g.cuSpend(140) {
					g.talkMenu("\"Come back with the coin.\"")
					return
				}
				g.give(&Item{ID: g.newID(), Kind: "letter", Name: "forged writ", Glyph: '?', F: CYellow, Value: 120,
					Text: "A customs writ with a convincing seal. It may get a load past a gate.", Extra: map[string]int{"tq:writ": 1}})
				g.talkMenu("\"Don't show it twice to the same clerk.\"")
			}})
	}
	return out
}

func (g *Game) tqFenceMenu(f *Actor, bd *BurgData) {
	var opts []Option
	for _, d := range g.tqGoodsHere() {
		d := d
		n := g.tqHeld(d)
		if n == 0 {
			continue
		}
		mul := 1.0
		tag := ""
		if g.tqBanned(bd.State, d) {
			mul, tag = 1.45, " {{R|[banned here]}}"
		} else if d.Ban != "" {
			mul, tag = 1.2, " {{O|[contraband elsewhere]}}"
		}
		pr := int(float64(g.tqPrice(bd, d, false)) * mul)
		opts = append(opts, Option{fmt.Sprintf("%d %s%s at %s each", n, g.tqName(d), tag, g.cuFormat(pr)), func() {
			k, basis := g.tqTake(d, n, false)
			rev := pr * k
			g.tqPaid(bd, rev, basis, k, d)
			if d.Ban != "" {
				g.tqAddPts(rev / 10)
				g.tqDeed("fenced", "You fenced a load of contraband.")
			}
			g.Mode, g.Talk = "main", nil
		}})
	}
	opts = append(opts, Option{"Done.", func() { g.Mode, g.Talk = "main", nil }})
	g.tqOpenMenu("Fence", "No questions, no receipts. Banned goods fetch the best cut.", opts, false)
}

// ---------------------------------------------------------------- generic menu

var tqMenuTitle string
var tqMenuForced bool

func (g *Game) tqOpenMenu(title, text string, opts []Option, forced bool) {
	tqMenuTitle, tqMenuForced = title, forced
	g.Popup = Wrap(text, 100)
	g.Opts = opts
	g.Sel = 0
	g.Mode = "tqmenu"
}

func tqMenuEsc(g *Game) func() {
	return func() {
		if tqMenuForced {
			return
		}
		g.Mode, g.Talk, g.Popup, g.Opts = "main", nil, nil, nil
	}
}

func tqMenuDraw(g *Game) {
	s := g.sc
	h := len(g.Popup) + len(g.Opts) + 4
	y0 := maxi(1, (MH-h)/2)
	s.Box(2, y0, SW-4, mini(h, SH-2), CYellow, tqMenuTitle)
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

// ---------------------------------------------------------------- ledger screen (F6)

func tqLedgerKey(g *Game, k string) {
	switch k {
	case "Escape", "q", "F6":
		g.Mode = "main"
	case "ArrowDown", "j":
		g.Sel++
	case "ArrowUp", "k":
		g.Sel = maxi(0, g.Sel-1)
	case "t":
		if bd := g.tqBurg(); bd != nil {
			g.Mode, g.Sel = "tq", 0
		}
	}
}

func (g *Game) tqRoutes() []string {
	s := g.tq()
	type rt struct {
		txt string
		pr  int
	}
	var rs []rt
	for _, d := range g.tqGoodsHere() {
		bestBuy, bestSell := -1, -1
		var bb, sb string
		var bday, sday int
		for k, m := range s.Mem {
			v, ok := m[d.ID]
			if !ok {
				continue
			}
			id, _ := strconv.Atoi(k)
			b := g.W.burgByID[id]
			if b == nil || g.tqBanned(b.State, d) {
				continue
			}
			if bestBuy < 0 || v[0] < bestBuy {
				bestBuy, bb, bday = v[0], b.Name, v[2]
			}
			if v[1] > bestSell {
				bestSell, sb, sday = v[1], b.Name, v[2]
			}
		}
		if bestBuy > 0 && bestSell > bestBuy && bb != sb {
			p := bestSell - bestBuy
			rs = append(rs, rt{fmt.Sprintf("%-22s buy at %s (%d, %dd ago)  ->  sell at %s (%d, %dd ago): {{G|+%d each}}",
				g.tqName(d), bb, bestBuy, g.tqToday()-bday, sb, bestSell, g.tqToday()-sday, p), p * 100 / maxi(bestBuy, 1)})
		}
	}
	sort.SliceStable(rs, func(i, j int) bool { return rs[i].pr > rs[j].pr })
	var out []string
	for i, r := range rs {
		if i >= 8 {
			break
		}
		out = append(out, r.txt)
	}
	return out
}

func (g *Game) tqLedgerLines() []string {
	s := g.tq()
	rk := s.rank()
	next := ""
	if rk+1 < len(tqRanks) {
		next = fmt.Sprintf(" (next: %s at %d)", tqRanks[rk+1].Title, tqRanks[rk+1].Pts)
	}
	L := []string{
		fmt.Sprintf("Rank: {{Y|%s}}, standing %d%s", tqRanks[rk].Title, s.Pts, next),
		fmt.Sprintf("Profit to date: %d base   Trades: %d   Contracts done: %d   Failed: %d", s.Profit, s.Trades, s.Done, s.Failed),
		fmt.Sprintf("Smuggling runs: %d   Caught: %d", s.Smuggled, s.Caught),
	}
	if rk >= 2 {
		L = append(L, fmt.Sprintf("Merchants' guild perk: %d%% better prices on bulk goods.", mini(rk*2, 10)))
	}
	L = append(L, "", "{{W|Goods in hand}}")
	n := 0
	show := func(inv []*Item, where string) {
		for _, it := range inv {
			if d := tqItemGood(it); d != nil {
				n++
				age := g.tqToday() - it.Extra["tq:d"]
				c := ""
				if it.Extra["tq:c"] != 0 {
					c = " {{Y|[contract]}}"
				}
				sp := ""
				if d.Perish > 0 && age > d.Perish {
					sp = " {{R|[spoiling]}}"
				}
				L = append(L, fmt.Sprintf("  %s x%d (%s, %dd old, cost %d each)%s%s", g.tqName(d), it.Count, where, age, it.Extra["tq:u"], c, sp))
			}
		}
	}
	show(g.P.Inv, "pack")
	if sp := g.sh().Ship; sp != nil {
		show(sp.Hold, "ship hold")
		L = append(L, fmt.Sprintf("  Ship hold: %d slots free", g.tqHoldFree()))
	}
	if n == 0 {
		L = append(L, "  none")
	}
	L = append(L, "", "{{W|Contracts}}")
	for _, c := range s.Ct {
		L = append(L, "  "+g.tqContractText(c))
	}
	if len(s.Ct) == 0 {
		L = append(L, "  none: ask a factor in a market or harbour district")
	}
	L = append(L, "", "{{W|Best known routes}} (from prices you have seen)")
	rs := g.tqRoutes()
	for _, r := range rs {
		L = append(L, "  "+r)
	}
	if len(rs) == 0 {
		L = append(L, "  visit more towns to learn their prices")
	}
	L = append(L, "", "{{W|Market news}}")
	nl := g.tqNews()
	for _, l := range nl {
		L = append(L, "  "+l.Text)
	}
	if len(nl) == 0 {
		L = append(L, "  The markets are quiet.")
	}
	if l := g.uwLawsFor(g.uwStateHere()).list(g); len(l) > 0 {
		L = append(L, "", "Banned where you stand: "+strings.Join(l, ", "))
	}
	return L
}

func tqLedgerDraw(g *Game) {
	s := g.sc
	s.Box(2, 1, SW-4, SH-2, CYellow, "Trade ledger")
	L := g.tqLedgerLines()
	var wrapped []string
	for _, l := range L {
		if l == "" {
			wrapped = append(wrapped, "")
			continue
		}
		wrapped = append(wrapped, Wrap(l, SW-10)...)
	}
	view := SH - 6
	g.Sel = clampi(g.Sel, 0, maxi(len(wrapped)-view, 0))
	for i := 0; i < view && g.Sel+i < len(wrapped); i++ {
		s.TextB(4, 3+i, wrapped[g.Sel+i], CGray, CBgDark, SW-3)
	}
	s.TextB(4, SH-3, "{{W|Up/Down}} scroll  {{W|t}} bulk goods (in a town)  {{W|Esc}} close", CGray, CBgDark, SW-3)
}
