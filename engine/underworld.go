package engine

import (
	"fmt"
	"sort"
	"strings"
)

// The underworld: shared state, black markets and fences in the warrens and outlaw hideouts, forged
// papers, and the fight pit under great cities. Laws, stealing and bounties are in crime.go.
// Keys: X light fingers, U underworld ledger (laws, bounty, pit record).

type uwBout struct {
	Key   string `json:"key"`
	Foes  []int  `json:"foes"`
	Stake int    `json:"stake"`
	Lvl   int    `json:"lvl"`
	N     int    `json:"n"`
}

type uwState struct {
	Bounty   map[int]int       `json:"bounty,omitempty"`
	Searched map[string]bool   `json:"searched,omitempty"`
	Looted   map[string]bool   `json:"looted,omitempty"`
	Rings    map[string][4]int `json:"rings,omitempty"`
	FenceDay map[int]int       `json:"fenceDay,omitempty"`
	LastBurg int               `json:"lastBurg,omitempty"`
	Pit      *uwBout           `json:"pit,omitempty"`
	Wins     int               `json:"wins,omitempty"`
	Crowd    int               `json:"crowd,omitempty"`
	Won      int               `json:"won,omitempty"`
	Stolen   int               `json:"stolen,omitempty"`
	Jailed   int               `json:"jailed,omitempty"`
	Fenced   int               `json:"fenced,omitempty"`

	title  string
	forced bool
}

func (g *Game) uwSt() *uwState {
	st := g.ext("uw", func() any { return &uwState{} }).(*uwState)
	if st.Bounty == nil {
		st.Bounty = map[int]int{}
	}
	if st.Searched == nil {
		st.Searched = map[string]bool{}
	}
	if st.Looted == nil {
		st.Looted = map[string]bool{}
	}
	if st.Rings == nil {
		st.Rings = map[string][4]int{}
	}
	if st.FenceDay == nil {
		st.FenceDay = map[int]int{}
	}
	return st
}

func init() {
	zoneHooks = append(zoneHooks, uwZone)
	enterHooks = append(enterHooks, uwEnter)
	killHooks = append(killHooks, uwKill, uwPitKill)
	turnHooks = append(turnHooks, uwTurn)
	talkHooks = append(talkHooks, uwTalk)
	sideHooks = append(sideHooks, uwSide)
	mainKeys["X"] = func(g *Game) { g.uwStealKey() }
	mainKeys["U"] = func(g *Game) { g.uwLedger() }
	modeKeys["uwmenu"] = func(g *Game, k string) {
		g.keyMenu(k, func() {
			if g.uwSt().forced && len(g.Opts) > 0 {
				g.Opts[0].Do()
				return
			}
			g.uwClose()
		})
	}
	modeDraws["uwmenu"] = func(g *Game) { g.uwDrawMenu() }
}

// ---- menu plumbing ----

func (g *Game) uwOpenMenu(title, text string, opts []Option, forced bool) {
	st := g.uwSt()
	st.title, st.forced = title, forced
	g.Popup = Wrap(text, 100)
	g.Opts = opts
	g.Sel = 0
	g.Mode = "uwmenu"
}

func (g *Game) uwClose() {
	g.uwSt().forced = false
	g.Mode, g.Talk, g.Popup, g.Opts = "main", nil, nil, nil
}

func (g *Game) uwDrawMenu() {
	s := g.sc
	h := len(g.Popup) + len(g.Opts) + 4
	y0 := maxi(1, (MH-h)/2)
	s.Box(2, y0, SW-4, mini(h, SH-2), CDarkMagenta, g.uwSt().title)
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

// ---- economy ----

// uwWealth: the black market's purse in a realm, richer where outlaws thrive (1.0 .. 1.8).
func (g *Game) uwWealth(state int) float64 {
	s := 0.0
	for _, ol := range g.W.Outlaws {
		if bd := g.W.burgByID[ol.Burg]; bd != nil && bd.State == state {
			s += float64(ol.Strength)/10 + float64(ol.Gold)/500
		}
	}
	return 1 + fclamp(s/20, 0, 0.8)
}

func (g *Game) uwCrowdTitle() string {
	c := g.uwSt().Crowd
	switch {
	case c >= 80:
		return "Champion of the pit"
	case c >= 50:
		return "Pit favourite"
	case c >= 25:
		return "Crowd pleaser"
	case c >= 10:
		return "Fresh meat"
	}
	return "Unknown"
}

func (g *Game) uwFencePays(it *Item, state int, fence *Actor) int {
	f := 0.25
	switch {
	case uwIsStolen(it):
		f = 0.42
	case g.uwIllegal(it, state):
		f = 0.5
	}
	f += float64(g.P.M(EGO))*0.02 + float64(g.Skills["persuasion"])*0.04 + (g.uwWealth(state)-1)*0.2 + float64(g.uwSt().Crowd)/1000
	f = fclamp(f, 0.1, 0.75)
	return maxi(1, int(float64(maxi(it.Value, 1))*f))
}

func (g *Game) uwDealerPrice(it *Item, state int) int {
	f := 1.5 - float64(g.P.M(EGO))*0.03 - float64(g.Skills["persuasion"])*0.05 - (g.uwWealth(state)-1)*0.3 - float64(g.uwSt().Crowd)/800
	f = fclamp(f, 1.0, 2.0)
	return maxi(1, int(float64(maxi(it.Value, 1))*f))
}

// uwStolenFactor: optional factor in price() for a sale to an honest merchant.
func (g *Game) uwStolenFactor(it *Item, buy bool) float64 {
	if !buy && uwIsStolen(it) {
		return 0.5
	}
	return 1
}

// uwNoSale: optional hook in tradeAct's sell branch; honest merchants refuse goods that are illegal in their realm.
func (g *Game) uwNoSale(it *Item, a *Actor) bool {
	if a == nil || !g.uwIllegal(it, a.State) {
		return false
	}
	g.Msg("{{R|\"I won't touch that. Take it away before the watch sees.\"}}")
	return true
}

// ---- contraband and stock ----

func (g *Game) uwVenom(r *Rng) *Item {
	n := Pick(r, []string{"nightshade", "viper", "black-lotus", "grave-moss", "gall", "wyrm-spit"})
	return &Item{ID: g.newID(), Kind: "ingredient", Effect: "venom", Name: "vial of " + n + " venom", Glyph: '!', F: CGreen, Count: 1, Value: 70 + r.Int(0, 40),
		Extra: map[string]int{"uw:poison": 1, "uw:black": 1}}
}

func (g *Game) uwPapers() *Item {
	return &Item{ID: g.newID(), Kind: "letter", Name: "forged papers", Glyph: '?', F: CYellow, Value: 140,
		Text: "A pardon-seal, convincingly aged. Use it from the underworld ledger (U) to clear part of a bounty.", Extra: map[string]int{"uw:forged": 1, "uw:black": 1}}
}

func (g *Game) uwMakeContraband(state int, wealth float64, r *Rng) *Item {
	l := g.uwLawsFor(state)
	var kinds []string
	if l.Occult {
		kinds = append(kinds, "occult")
	}
	if l.Heresy {
		kinds = append(kinds, "heresy")
	}
	if l.Augment {
		kinds = append(kinds, "augment")
	}
	if l.Drugs {
		kinds = append(kinds, "drugs")
	}
	if l.Arms {
		kinds = append(kinds, "arms")
	}
	if len(kinds) == 0 {
		kinds = []string{"augment", "occult", "mod"}
	}
	kinds = append(kinds, "mod")
	var it *Item
	switch Pick(r, kinds) {
	case "occult":
		it = g.MakeWeapon(Pick(r, []string{"shortblade", "longblade"}), 2+r.Int(0, 1), r)
		g.applyRarity(it, r, "rare")
		it.Cat = "occult"
		it.Name = "forbidden " + it.Name
		it.Value = it.Value*3 + 60
	case "heresy":
		it = g.MakeBook(Pick(r, []string{"The Unlit Heresies", "Whispers Beneath the Altar", "On the Falseness of the Throne", "The Black Catechism"}), "Pages the censors burned. The margins are full of a reader's furious notes.")
		it.Extra = map[string]int{"uw:forbidden": 1}
		it.Value = 90
	case "augment":
		var pool []string
		for _, d := range AugDefs {
			if !d.Rare || wealth >= 1.4 {
				pool = append(pool, d.ID)
			}
		}
		it = g.MakeAugment(Pick(r, pool))
		if it != nil {
			it.Value = it.Value * 3 / 2
		}
	case "drugs":
		it = g.MakeTonic(Pick(r, []string{"haste", "strength"}), 2)
		it.Value = 45
	case "arms":
		it = g.MakeRanged(3, r)
		g.applyRarity(it, r, "uncommon")
	default: // an illicit mod fitted without the bench
		it = g.MakeWeapon(Pick(r, []string{"longblade", "axe", "shortblade", "cudgel"}), 2, r)
		var cand []int
		for i := range modDefs {
			if isKind(modDefs[i].Kinds, "weapon") {
				cand = append(cand, i)
			}
		}
		md := modDefs[cand[r.Int(0, len(cand)-1)]]
		md.Apply(it)
		if it.Extra == nil {
			it.Extra = map[string]int{}
		}
		it.Extra["mod:"+md.ID] = 1
		it.Value += 90
		g.refreshModName(it)
	}
	if it == nil {
		return g.uwVenom(r)
	}
	if it.Extra == nil {
		it.Extra = map[string]int{}
	}
	it.Extra["uw:black"] = 1
	return it
}

func (g *Game) uwStockDealer(a *Actor, state int, r *Rng) {
	wealth := g.uwWealth(state)
	n := 3 + int(wealth*2) + r.Int(0, 2)
	for i := 0; i < n; i++ {
		a.Inv = append(a.Inv, g.uwMakeContraband(state, wealth, r))
	}
	for i := 0; i < r.Int(1, 2+int(wealth)); i++ {
		a.Inv = append(a.Inv, g.uwVenom(r))
	}
	a.Inv = append(a.Inv, g.uwPapers(), g.uwPapers())
}

// ---- placement ----

func (g *Game) uwNPC(r *Rng, role, title string, cu *CultureData, x, y int, lvl int) *Actor {
	a := g.Spawn("villager", lvl, r)
	a.Role, a.Title, a.Proper, a.AI = role, title, true, "still"
	a.Glyph, a.F = '@', CDarkMagenta
	if cu != nil {
		a.Name = g.cultureName(cu, r)
	} else {
		a.Name = Pick(r, []string{"Silk", "Nine-Fingers", "Magpie", "Old Tam", "Vesk", "Sallow", "Brindle", "Quill"})
	}
	a.Desc = fmt.Sprintf("A %s who keeps their back to the wall.", title)
	a.X, a.Y, a.HomeX, a.HomeY = x, y, x, y
	a.Gold = 150
	return a
}

func (g *Game) uwFreeNear(zn *Zone, r *Rng) (int, int, bool) {
	return zn.FreeSpot(r, nil)
}

func uwIsFence(a *Actor) bool { return a.Role == "fence" || a.Title == "fence" }

func uwZone(g *Game, zn *Zone, r0 *Rng) {
	uwZoneOwned(g, zn, nil)
	r := NewRng(g.Seed + ":uw:" + zn.Key)
	switch {
	case zn.Kind == "town" && strings.HasSuffix(zn.Name, districtNames["slums"]):
		bd := g.W.burgByID[zn.Burg]
		if bd == nil {
			return
		}
		cu := g.W.cultByID[bd.Culture]
		tier := g.W.cityTier[bd.ID]
		hasFence := false
		for _, a := range zn.Actors {
			if uwIsFence(a) {
				hasFence = true
				a.Set("uw:fence")
			}
		}
		if !hasFence {
			if x, y, ok := zn.FreeSpot(r, nil); ok {
				zn.Actors = append(zn.Actors, g.uwNPC(r, "fence", "fence", cu, x, y, 3))
			}
		}
		if tier >= 1 || bd.Bandit {
			if x, y, ok := zn.FreeSpot(r, nil); ok {
				d := g.uwNPC(r, "bmdealer", "black-market dealer", cu, x, y, 4+tier)
				g.uwStockDealer(d, bd.State, r)
				zn.Actors = append(zn.Actors, d)
			}
		}
		if tier == 2 || tier == 1 && r.Chance(0.35) {
			g.uwBuildPit(zn, r, cu)
		}
	case zn.Kind == "dungeon" && zn.Z == 1:
		s := g.siteAt(zn.WX, zn.WY, zn.ZX, zn.ZY)
		if s == nil || s.Theme != "hideout" {
			return
		}
		state := 0
		if g.W.In(zn.WX, zn.WY) {
			state = g.W.State[g.W.Idx(zn.WX, zn.WY)]
		}
		for i, role := range []string{"fence", "bmdealer"} {
			x, y, ok := zn.FreeSpot(r, nil)
			if !ok {
				return
			}
			title := map[string]string{"fence": "fence", "bmdealer": "black-market dealer"}[role]
			a := g.uwNPC(r, role, title, nil, x, y, 4)
			a.Faction = "outlaws"
			if i == 1 {
				g.uwStockDealer(a, state, r)
			}
			zn.Actors = append(zn.Actors, a)
		}
	}
}

func (g *Game) uwBuildPit(zn *Zone, r *Rng, cu *CultureData) {
	rc, ok := g.lot(zn, r, 13, 15, 8, 9)
	if !ok {
		return
	}
	x, y, w, h := rc[0], rc[1], rc[2], rc[3]
	g.building(zn, x, y, w, h, TWall, TSand)
	zn.Set(x+w/2, y+h-1, TDoorOpen)
	g.uwSt().Rings[zn.Key] = rc
	zn.Notes[(y+h-1)*MW+x+w/2] = "A sunken doorway. The noise inside is not music."
	px, py := x+w/2+1, y+h
	if !zn.Walkable(px, py) || zn.ActorAt(px, py) != nil {
		var ok2 bool
		px, py, ok2 = zn.FreeSpot(r, nil)
		if !ok2 {
			return
		}
	}
	zn.Actors = append(zn.Actors, g.uwNPC(r, "pitmaster", "pit-master", cu, px, py, 6))
}

// ---- turn / side ----

func uwTurn(g *Game) {
	st := g.uwSt()
	if g.Turn%1440 == 0 {
		here := g.uwStateHere()
		for s, b := range st.Bounty {
			if s != here {
				b -= maxi(1, b/15)
			}
			if b <= 0 {
				g.uwClearBounty(s)
			} else {
				st.Bounty[s] = b
			}
		}
	}
	if g.Turn%5 == 0 {
		if s := g.uwStateHere(); st.Bounty[s] >= 10 {
			g.uwAlertGuards(s, g.Cur)
		}
	}
	if st.Pit != nil && g.Cur != nil && g.Cur.Key != st.Pit.Key {
		g.uwPitForfeit()
	}
}

func uwSide(g *Game) []string {
	var out []string
	if b := g.uwBounty(g.uwStateHere()); b > 0 {
		out = append(out, fmt.Sprintf("{{R|Wanted: %d}}", b))
	}
	if g.Cur != nil && g.Cur.Kind == "town" && len(g.uwContraband(g.uwStateHere())) > 0 {
		out = append(out, "{{y|Carrying contraband}}")
	}
	return out
}

// ---- ledger (U) ----

func (g *Game) uwLedger() {
	if g.Mode != "main" {
		return
	}
	st := g.uwSt()
	state := g.uwStateHere()
	var sb strings.Builder
	fmt.Fprintf(&sb, "{{Y|%s}} forbids: %s.\n", g.uwRealm(state), strings.Join(g.uwLawsFor(state).list(g), ", "))
	b := st.Bounty[state]
	fmt.Fprintf(&sb, "\nBounty here: {{R|%d}} %s.", b, g.Sk.Currency)
	var other []string
	for s, v := range st.Bounty {
		if s != state && v > 0 {
			other = append(other, fmt.Sprintf("%s %d", g.uwRealm(s), v))
		}
	}
	sort.Strings(other)
	if len(other) > 0 {
		sb.WriteString(" Elsewhere: " + strings.Join(other, "; ") + ".")
	}
	fmt.Fprintf(&sb, "\nLifted: %d  Jailed: %d  Fenced: %d  Pit: %d wins, %s (crowd %d).", st.Stolen, st.Jailed, st.Fenced, st.Wins, g.uwCrowdTitle(), st.Crowd)
	var opts []Option
	if b > 0 {
		opts = append(opts,
			Option{fmt.Sprintf("{{W|Pay the bounty.}} {{K|(%d %s)}}", b, g.Sk.Currency), func() { g.uwPayBounty(state); g.uwLedger() }},
			Option{"{{y|Surrender and serve time.}}", func() { g.uwClose(); g.uwJail(state) }})
		for _, it := range g.P.Inv {
			if uwIsForged(it) {
				it := it
				opts = append(opts, Option{"{{c|Present forged papers.}}", func() { g.uwUsePapers(it, state); g.uwLedger() }})
				break
			}
		}
	}
	opts = append(opts, Option{"Close.", func() { g.uwClose() }})
	g.uwOpenMenu("Underworld ledger", sb.String(), opts, false)
}

// uwUsePapers: a believable forgery wipes 60% of the bounty; a bad one adds to it.
func (g *Game) uwUsePapers(it *Item, state int) {
	g.removeInv(it)
	if g.roll2d6Check(g.checkMod("bluff")+g.Skills["tinkering"], 8) {
		st := g.uwSt()
		st.Bounty[state] = st.Bounty[state] * 4 / 10
		g.Msg("{{G|The clerk stamps the papers. Most of your record vanishes.}}")
		if st.Bounty[state] < 10 {
			g.uwClearBounty(state)
		}
		return
	}
	g.Msg("{{R|The clerk spots the forgery.}}")
	g.uwSt().Bounty[state] += 40
	g.uwAlertGuards(state, g.Cur)
}

// ---- talk ----

func uwTalk(g *Game, a *Actor) []Option {
	var out []Option
	state := g.uwStateHere()
	if (a.Role == "elder" || a.Role == "notable") && g.uwBounty(state) > 0 {
		b := g.uwBounty(state)
		out = append(out,
			Option{fmt.Sprintf("{{W|Settle my bounty.}} {{K|(%d %s)}}", b, g.Sk.Currency), func() {
				if g.uwPayBounty(state) {
					g.talkMenu("\"The magistrate's seal is on it. You are clean.\"")
				} else {
					g.talkMenu("\"Come back with the full sum.\"")
				}
			}},
			Option{"{{y|Surrender and serve time.}}", func() { g.Mode, g.Talk = "main", nil; g.uwJail(state) }})
	}
	switch {
	case uwIsFence(a):
		out = append(out,
			Option{"{{M|Sell goods quietly.}}", func() { g.uwFenceMenu(a, 0) }},
			Option{"{{M|What's against the law around here?}}", func() {
				g.talkMenu("\"Here? " + strings.Join(g.uwLawsFor(state).list(g), ", ") + ". Watch yourself at the gates.\"")
			}})
	case a.Role == "bmdealer":
		out = append(out, Option{"{{M|See what's in the back room.}}", func() { g.uwDealerMenu(a, 0) }})
	case a.Role == "pitmaster":
		out = append(out, g.uwPitOptions(a)...)
	}
	return out
}

func (g *Game) uwFenceMenu(f *Actor, page int) {
	st := g.uwSt()
	state := g.uwStateHere()
	if d := g.Turn / 1440; st.FenceDay[f.ID] != d+1 {
		st.FenceDay[f.ID] = d + 1
		f.Gold = maxi(f.Gold, int(150*g.uwWealth(state))*(1+g.W.cityTier[g.Cur.Burg]))
	}
	var items []*Item
	for _, it := range g.P.Inv {
		if it.Quest == 0 {
			items = append(items, it)
		}
	}
	sort.SliceStable(items, func(i, j int) bool {
		si, sj := uwIsStolen(items[i]) || g.uwIllegal(items[i], state), uwIsStolen(items[j]) || g.uwIllegal(items[j], state)
		if si != sj {
			return si
		}
		return items[i].Value > items[j].Value
	})
	var opts []Option
	for i, it := range items {
		if i < page*16 || i >= page*16+16 {
			continue
		}
		it := it
		pr := g.uwFencePays(it, state, f)
		tag := ""
		if uwIsStolen(it) {
			tag = " {{R|[hot]}}"
		} else if g.uwIllegal(it, state) {
			tag = " {{R|[banned]}}"
		}
		opts = append(opts, Option{fmt.Sprintf("%s%s — %d %s", it.Display(), tag, pr, g.Sk.Currency), func() {
			if f.Gold < pr {
				g.uwFenceMenu(f, page)
				g.Msg("\"I'm light on coin today.\"")
				return
			}
			f.Gold -= pr
			g.P.Gold += pr
			one := g.uwTake(&g.P.Inv, it)
			if g.P.Equipped(one) == nil {
				f.Inv = append(f.Inv, one)
			}
			st.Fenced++
			g.Msg(fmt.Sprintf("The fence counts out %d %s. No questions.", pr, g.Sk.Currency))
			g.uwFenceMenu(f, page)
		}})
	}
	if (page+1)*16 < len(items) {
		opts = append(opts, Option{"More...", func() { g.uwFenceMenu(f, page+1) }})
	}
	opts = append(opts, Option{"Done.", func() { g.talkMenu("\"Pleasure doing nothing with you.\"") }})
	g.uwOpenMenu("Fence", fmt.Sprintf("Their purse: %d %s. Hot goods and banned goods fetch the best cut.", f.Gold, g.Sk.Currency), opts, false)
	g.Talk = f
}

func (g *Game) uwDealerMenu(d *Actor, page int) {
	state := g.uwStateHere()
	var opts []Option
	for i, it := range d.Inv {
		if i < page*16 || i >= page*16+16 {
			continue
		}
		it := it
		pr := g.uwDealerPrice(it, state)
		opts = append(opts, Option{fmt.Sprintf("%s — %d %s", it.Display(), pr, g.Sk.Currency), func() {
			if g.P.Gold < pr {
				g.uwDealerMenu(d, page)
				g.Msg("You can't afford that.")
				return
			}
			g.P.Gold -= pr
			d.Gold += pr
			one := g.uwTake(&d.Inv, it)
			g.give(one)
			g.Msg("You buy " + one.Display() + ".")
			g.uwDealerMenu(d, page)
		}})
	}
	if (page+1)*16 < len(d.Inv) {
		opts = append(opts, Option{"More...", func() { g.uwDealerMenu(d, page+1) }})
	}
	opts = append(opts, Option{"Done.", func() { g.talkMenu("\"Come back when the watch is looking elsewhere.\"") }})
	g.uwOpenMenu("Back room", "Everything here is illegal somewhere. In "+g.uwRealm(state)+", ask the dealer which parts.", opts, false)
	g.Talk = d
}

// ---- the pit ----

func (g *Game) uwPitOptions(pm *Actor) []Option {
	st := g.uwSt()
	var out []Option
	if st.Pit != nil {
		return []Option{{"{{K|(A bout is under way.)}}", func() { g.talkMenu("\"Finish what you started.\"") }}}
	}
	out = append(out,
		Option{fmt.Sprintf("{{R|Fight in the pit.}} {{K|(deadly rules; %s; wins %d)}}", g.uwCrowdTitle(), st.Wins), func() { g.uwPitStakeMenu(pm) }},
		Option{"{{W|Wager on a bout between others.}}", func() { g.uwWagerSides(pm) }})
	return out
}

func (g *Game) uwPitStakeMenu(pm *Actor) {
	var opts []Option
	for _, s := range []int{0, 25, 50, 100, 250} {
		s := s
		if s > g.P.Gold {
			continue
		}
		label := "No bet."
		if s > 0 {
			label = fmt.Sprintf("Bet %d %s on yourself (pays double).", s, g.Sk.Currency)
		}
		opts = append(opts, Option{label, func() { g.uwStartBout(pm, s) }})
	}
	opts = append(opts, Option{"Back.", func() { g.talkMenu(g.greeting(pm)) }})
	g.uwOpenMenu("The pit", "\"No judges, no yielding. The door is barred until one side stops breathing. The crowd pays for blood.\"", opts, false)
	g.Talk = pm
}

func (g *Game) uwStartBout(pm *Actor, stake int) {
	st := g.uwSt()
	rc, ok := st.Rings[g.Cur.Key]
	if !ok {
		g.uwClose()
		g.Msg("The pit is shut tonight.")
		return
	}
	n := mini(1+st.Wins/3, 4)
	lvl := g.P.Level + 1 + st.Wins/3
	tpl := arenaFoes[mini(st.Wins/2+g.R.Int(0, 2), len(arenaFoes)-1)]
	g.P.Gold -= stake
	bout := &uwBout{Key: g.Cur.Key, Stake: stake, Lvl: lvl, N: n}
	for i := 0; i < n; i++ {
		a := g.Spawn(tpl, lvl, g.R)
		a.Faction, a.Angry = "pit", true
		a.X, a.Y = rc[0]+3+i*3, rc[1]+2
		g.nudge(g.Cur, a)
		g.Cur.Actors = append(g.Cur.Actors, a)
		bout.Foes = append(bout.Foes, a.ID)
	}
	g.P.X, g.P.Y = rc[0]+rc[2]/2, rc[1]+rc[3]-2
	g.Cur.Set(rc[0]+rc[2]/2, rc[1]+rc[3]-1, TWall) // the door is barred
	st.Pit = bout
	g.uwClose()
	g.Msg(fmt.Sprintf("{{R|The door slams. The crowd bays: %d %s!}}", n, tplByID[tpl].Name(g.Genre)))
	g.recompute()
}

func uwPitKill(g *Game, d, src *Actor) {
	st := g.uwSt()
	if d.Faction != "pit" || st.Pit == nil {
		return
	}
	for i, id := range st.Pit.Foes {
		if id == d.ID {
			st.Pit.Foes = append(st.Pit.Foes[:i], st.Pit.Foes[i+1:]...)
			break
		}
	}
	if len(st.Pit.Foes) > 0 {
		return
	}
	b := st.Pit
	purse := (30 + b.Lvl*15) * b.N * (100 + st.Crowd) / 100
	total := purse + b.Stake*2
	g.P.Gold += total
	st.Won += total
	st.Wins++
	st.Crowd = mini(st.Crowd+2+b.N, 100)
	g.gainXP(20*b.N + b.Lvl*5)
	g.uwOpenRing(b.Key)
	st.Pit = nil
	g.Msg(fmt.Sprintf("{{G|The pit roars your name. Purse: %d %s. You are now: %s.}}", total, g.Sk.Currency, g.uwCrowdTitle()))
}

func (g *Game) uwOpenRing(key string) {
	st := g.uwSt()
	if rc, ok := st.Rings[key]; ok {
		if zn := g.Zones[key]; zn != nil {
			zn.Set(rc[0]+rc[2]/2, rc[1]+rc[3]-1, TDoorOpen)
		}
	}
}

func (g *Game) uwPitForfeit() {
	st := g.uwSt()
	b := st.Pit
	if b == nil {
		return
	}
	if zn := g.Zones[b.Key]; zn != nil {
		keep := zn.Actors[:0]
		for _, a := range zn.Actors {
			if a.Faction != "pit" {
				keep = append(keep, a)
			}
		}
		zn.Actors = keep
	}
	g.uwOpenRing(b.Key)
	st.Crowd = maxi(0, st.Crowd-5)
	st.Pit = nil
	g.Msg("{{R|You left the pit. The crowd jeers; your stake is lost.}}")
}

// ---- wagers on other fighters ----

type uwFighter struct {
	tpl, name string
	lvl       int
	rating    float64
}

func (g *Game) uwFighters() [2]uwFighter {
	r := NewRng(fmt.Sprintf("%s:uw:wager:%d", g.Seed, g.Turn/30))
	var fs [2]uwFighter
	for i := range fs {
		tpl := arenaFoes[r.Int(0, len(arenaFoes)-1)]
		lvl := g.P.Level + r.Int(0, 4)
		a := g.Spawn(tpl, lvl, r)
		fs[i] = uwFighter{tpl, tplByID[tpl].Name(g.Genre), lvl, float64(a.MaxHP + 8*lvl + a.AVNat*4)}
	}
	if fs[0].tpl == fs[1].tpl {
		fs[1].name += " (the elder)"
		fs[1].rating *= 1.1
	}
	return fs
}

// uwOdds: the payout multiplier for each fighter (house keeps 8%).
func uwOdds(fs [2]uwFighter) [2]float64 {
	p := fs[0].rating / (fs[0].rating + fs[1].rating)
	return [2]float64{0.92 / p, 0.92 / (1 - p)}
}

func (g *Game) uwWagerSides(pm *Actor) {
	fs := g.uwFighters()
	od := uwOdds(fs)
	var opts []Option
	for i := range fs {
		i := i
		opts = append(opts, Option{fmt.Sprintf("{{W|%s}} (level %d), pays x%.2f", fs[i].name, fs[i].lvl, od[i]), func() { g.uwWagerStake(pm, fs, i) }})
	}
	opts = append(opts, Option{"Back.", func() { g.talkMenu(g.greeting(pm)) }})
	g.uwOpenMenu("Wager", "Two fighters are dragged out of the cages. Deadly rules: the loser rarely walks away.", opts, false)
	g.Talk = pm
}

func (g *Game) uwWagerStake(pm *Actor, fs [2]uwFighter, side int) {
	var opts []Option
	for _, s := range []int{10, 25, 50, 100, 250} {
		s := s
		if s > g.P.Gold {
			continue
		}
		opts = append(opts, Option{fmt.Sprintf("Stake %d %s.", s, g.Sk.Currency), func() { g.uwResolveWager(pm, fs, side, s) }})
	}
	opts = append(opts, Option{"Back.", func() { g.uwWagerSides(pm) }})
	g.uwOpenMenu("Wager", "How much on "+fs[side].name+"?", opts, false)
	g.Talk = pm
}

func (g *Game) uwResolveWager(pm *Actor, fs [2]uwFighter, side, stake int) {
	st := g.uwSt()
	od := uwOdds(fs)
	p0 := fs[0].rating / (fs[0].rating + fs[1].rating)
	win := 0
	if !g.R.Chance(p0) {
		win = 1
	}
	g.P.Gold -= stake
	text := fmt.Sprintf("%s and %s circle, then tear into each other. ", fs[0].name, fs[1].name)
	if win == side {
		pay := int(float64(stake) * od[side])
		g.P.Gold += pay
		st.Won += pay - stake
		text += fmt.Sprintf("%s wins. {{G|You collect %d %s.}}", fs[win].name, pay, g.Sk.Currency)
	} else {
		text += fmt.Sprintf("%s wins. {{R|Your stake of %d is gone.}}", fs[win].name, stake)
	}
	st.Crowd = mini(st.Crowd+1, 100)
	g.Turn += 20
	g.uwOpenMenu("Wager", text, []Option{
		{"Another bout.", func() { g.uwWagerSides(pm) }},
		{"Enough.", func() { g.talkMenu(g.greeting(pm)) }},
	}, false)
	g.Talk = pm
}
