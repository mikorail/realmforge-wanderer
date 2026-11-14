package engine

import (
	"fmt"
	"strings"
)

// Night life (prefix cj, shares state with cityjobs.go). Between 21:00 and 05:00 town zones get
// a few extra night-only actors (flag "cj", roles cj_*) placed once per zone per night with a
// side rng; existing townsfolk are never touched. They are removed again at dawn. Also: a curfew
// challenge for drawn steel in tier 1+ cities, cutpurses, and shuttered shops in small towns.

const cjMaxNight = 8

func (g *Game) cjNight() bool {
	h := g.Hour()
	return h >= 21 || h < 5
}

// cjNightID names one night: 21:00 to the next 05:00 share an id.
func (g *Game) cjNightID() int { return (g.Turn + startMinutes + 180) / DayLen }

func cjEnter(g *Game, zn *Zone) { g.cjNightRefresh(zn) }

func cjTurn(g *Game) {
	if g == nil || g.Cur == nil || g.P == nil {
		return
	}
	if g.Turn%10 == 0 {
		g.cjNightRefresh(g.Cur)
	}
	if !g.cjNight() || g.Cur.Kind != "town" {
		return
	}
	for _, a := range g.Cur.Actors {
		if a.HP > 0 && a.Role == "cj_cutpurse" && !a.Has("cj_done") && !a.Angry && dist(a, g.P) <= 1 {
			g.cjCutpurse(a)
			break
		}
	}
	if g.Mode == "main" {
		g.cjCurfew()
	}
}

func (g *Game) cjRemoveNight(zn *Zone) {
	keep := zn.Actors[:0]
	for _, a := range zn.Actors {
		if a != nil && a.Has("cj") && !a.Angry && a.AI != "follower" && a != g.Talk {
			continue
		}
		keep = append(keep, a)
	}
	zn.Actors = keep
}

func (g *Game) cjCount(zn *Zone) int {
	n := 0
	for _, a := range zn.Actors {
		if a != nil && a.HP > 0 && a.Has("cj") {
			n++
		}
	}
	return n
}

func (g *Game) cjMk(zn *Zone, r *Rng, role, name string, glyph rune, col byte, ai string, near *Actor) *Actor {
	x, y, ok := zn.FreeSpot(r, func(x, y int) bool {
		if g.P != nil && maxi(abs(x-g.P.X), abs(y-g.P.Y)) < 6 {
			return false
		}
		if near != nil && maxi(abs(x-near.X), abs(y-near.Y)) > 5 {
			return false
		}
		return near != nil || zn.At(x, y) == TCobble
	})
	if !ok {
		if x, y, ok = zn.FreeSpot(r, nil); !ok {
			return nil
		}
	}
	a := g.Spawn("bandit", maxi(1, zn.Level), r)
	a.Name, a.Role, a.Glyph, a.F, a.AI = name, "cj_"+role, glyph, col, ai
	a.Flags = map[string]bool{"cj": true}
	a.X, a.Y, a.HomeX, a.HomeY = x, y, x, y
	a.Faction = zn.Faction
	if a.Faction == "" {
		a.Faction = uwFac(g.uwStateHere())
	}
	zn.Actors = append(zn.Actors, a)
	return a
}

func (g *Game) cjNightRefresh(zn *Zone) {
	if g == nil || zn == nil || g.W == nil || g.P == nil || zn.Z != 0 {
		return
	}
	if !g.cjNight() {
		if g.cjCount(zn) > 0 {
			g.cjRemoveNight(zn)
		}
		return
	}
	st := g.cj()
	nid := g.cjNightID()
	if st.Spawned[zn.Key] == nid+1 {
		return
	}
	st.Spawned[zn.Key] = nid + 1
	r := NewRng(fmt.Sprintf("%s:cj:night:%s:%d", g.Seed, zn.Key, nid))
	switch zn.Kind {
	case "outskirts":
		if r.Chance(0.3) {
			if a := g.cjMk(zn, r, "prowler", "prowling wolf", 'C', CGray, "hostile", nil); a != nil {
				*a = *g.cjBeast(a, "wolf", zn, r)
			}
		}
		return
	case "ruin":
		if r.Chance(0.45 + 0.35*float64(boolInt(g.Genre == "twilight"))) {
			g.cjGhost(zn, r, "hostile")
		}
		return
	case "town":
	default:
		return
	}
	d, bd := g.districtAt(zn.WX, zn.WY, zn.ZX, zn.ZY)
	tier := g.W.cityTier[zn.Burg]
	in := func(ks ...string) bool {
		for _, k := range ks {
			if d == k {
				return true
			}
		}
		return false
	}
	// watchmen with lanterns
	nw := 1
	if tier >= 1 && in("market", "harbor", "palace", "plaza", "barracks") {
		nw = 2
	}
	for i := 0; i < nw && g.cjCount(zn) < cjMaxNight; i++ {
		title := cjWatchTitles[g.Genre]
		if title == "" {
			title = "night watchman"
		}
		if a := g.cjMk(zn, r, "watch", title, '@', CYellow, "guard", nil); a != nil {
			a.Set("armed")
			a.MaxHP += 14
			a.HP = a.MaxHP
			a.Desc = "A watchman making the rounds with a shuttered lantern."
			hands := 0
			for _, p := range a.Body {
				if p.Slot != "hand" || p.Item != nil {
					continue
				}
				if hands == 0 {
					p.Item = g.MakeWeapon("cudgel", 1, r)
				} else if hands == 1 {
					p.Item = &Item{ID: g.newID(), Kind: "light", Name: "watch lantern", Glyph: '¥', F: CYellow, Light: 6, Weight: 1}
				}
				hands++
			}
		}
	}
	// revellers outside the tavern
	var inn *Actor
	for _, a := range zn.Actors {
		if a.HP > 0 && !a.Has("cj") && (a.Role == "innkeeper" || a.Role == "tavern" || a.Role == "lw_innkeeper") {
			inn = a
			break
		}
	}
	if inn != nil {
		for i := 0; i < 2 && g.cjCount(zn) < cjMaxNight; i++ {
			if a := g.cjMk(zn, r, "drunk", Pick(r, []string{"merry reveller", "swaying drunk", "singing sailor"}), '@', COrange, "town", inn); a != nil {
				a.Desc = "Somebody enjoying the night a little too much."
			}
		}
	}
	if in("slums", "homes", "temple", "market", "") && r.Chance(0.5) && g.cjCount(zn) < cjMaxNight {
		if a := g.cjMk(zn, r, "beggar", "night beggar", '@', CGray, "town", nil); a != nil {
			a.Desc = "A huddled figure in a doorway with an open palm."
		}
	}
	if in("market", "harbor", "slums", "plaza", "") && r.Chance(0.3+0.3*float64(boolInt(tier >= 1))) && g.cjCount(zn) < cjMaxNight {
		if a := g.cjMk(zn, r, "cutpurse", "hooded stranger", '@', CDarkMagenta, "town", nil); a != nil {
			a.Desc = "A hooded figure who keeps looking at the purses of passers-by."
			a.Gold = 0
		}
	}
	if in("harbor", "slums") && (tier >= 1 || (bd != nil && bd.Bandit)) && g.cjCount(zn) < cjMaxNight {
		if a := g.cjMk(zn, r, "nightmkt", "night-market dealer", '@', CDarkMagenta, "still", nil); a != nil {
			a.Proper = false
			a.Gold = 200
			state := 0
			if bd != nil {
				state = bd.State
			}
			g.uwStockDealer(a, state, r)
			a.Desc = "A dealer behind a covered cart, lit by one hooded lamp. The night market sells what the day shops will not."
		}
		hasFence := false
		for _, a := range zn.Actors {
			if uwIsFence(a) {
				hasFence = true
			}
		}
		if !hasFence && r.Chance(0.6) && g.cjCount(zn) < cjMaxNight {
			if a := g.cjMk(zn, r, "smuggler", "dock fence", '@', CDarkMagenta, "still", nil); a != nil {
				a.Role, a.Title = "fence", "fence"
				a.Gold = 160
				a.Set("uw:fence")
				a.Desc = "A fence who only meets people at the docks after dark."
			}
		}
	}
	gp := 0.08
	if in("temple") {
		gp = 0.55
	} else if g.Genre == "twilight" {
		gp = 0.3
	}
	if r.Chance(gp) && g.cjCount(zn) < cjMaxNight {
		g.cjGhost(zn, r, "town")
	}
	if in("homes", "market", "artisans", "") && r.Chance(0.5) && g.cjCount(zn) < cjMaxNight {
		if a := g.cjMk(zn, r, "baker", "night baker", '@', CWhite, "town", nil); a != nil {
			a.Desc = "A flour-dusted baker with a tray of warm loaves."
		}
	}
	if in("slums", "temple") && r.Chance(0.5) && g.cjCount(zn) < cjMaxNight {
		if a := g.cjMk(zn, r, "healer", "street healer", '@', CGreen, "town", nil); a != nil {
			a.Desc = "A healer with a satchel of salves who walks the poor streets at night."
		}
	}
}

// cjBeast turns a freshly placed actor into a real beast of template id in place.
func (g *Game) cjBeast(old *Actor, id string, zn *Zone, r *Rng) *Actor {
	b := g.Spawn(id, maxi(1, zn.Level+1), r)
	b.X, b.Y, b.HomeX, b.HomeY = old.X, old.Y, old.X, old.Y
	b.ID = old.ID
	b.Flags = map[string]bool{"cj": true}
	b.Role = ""
	return b
}

func (g *Game) cjGhost(zn *Zone, r *Rng, ai string) {
	a := g.cjMk(zn, r, "ghost", "restless shade", 'G', CCyan, ai, nil)
	if a == nil {
		return
	}
	g.cjAsGhost(a, zn, r)
}

func (g *Game) cjAsGhost(a *Actor, zn *Zone, r *Rng) {
	s := g.Spawn("skeleton", maxi(1, zn.Level), r)
	a.Body, a.Attr, a.MaxHP, a.HP, a.AVNat, a.DVNat = s.Body, s.Attr, s.MaxHP, s.HP, s.AVNat, s.DVNat
	a.Tpl = "skeleton"
	a.Set("undead")
	if a.AI == "hostile" {
		a.Faction = "undead"
	}
	a.Desc = "A pale shape that drifts a hand's breadth above the ground."
}

// ---- talk ----

func cjNightTalk(g *Game, a *Actor) []Option {
	if g == nil || a == nil || !a.Has("cj") {
		return nil
	}
	back := func(s string) {
		g.Talk = a
		g.talkMenu(s)
	}
	pick := func(lines []string, salt string) string {
		return Pick(NewRng(fmt.Sprintf("%s:cj:line:%d:%s:%d", g.Seed, a.ID, salt, g.Turn/60)), lines)
	}
	var o []Option
	switch a.Role {
	case "cj_watch":
		o = append(o, Option{"Ask the watchman what is afoot tonight.", func() { back(pick(cjWatchLines, "w")) }})
	case "cj_drunk":
		o = append(o, Option{"Ask the reveller what they have heard.", func() {
			t := pick(cjDrunkLines, "d")
			if NewRng(fmt.Sprintf("%s:cj:dr:%d", g.Seed, a.ID)).Chance(0.35) {
				g.addSecret(Pick(NewRng(fmt.Sprintf("%s:cj:dr2:%d", g.Seed, a.ID)), cjRumours), "rumour", -1, g.W.Year)
				t += "\n\n{{c|You file the rumour away.}}"
			}
			back(t)
		}})
	case "cj_beggar":
		o = append(o, Option{"Give the beggar a coin.", func() {
			if !g.cuSpend(1) {
				back("You have nothing to give.")
				return
			}
			g.crKarma(3, 2, "gave to a beggar at night")
			g.cjFirst("beggar", "Gave a coin to a beggar in the dark.")
			back(pick(cjBeggarLines, "b"))
		}})
	case "cj_cutpurse":
		o = append(o, Option{"Ask what they are doing out here so late.", func() {
			back("\"Walking. Only walking. The night is for everybody.\" The stranger's eyes stray to your purse.")
		}})
		if a.Gold > 0 {
			o = append(o, Option{"{{c|[Intimidate]}} Demand your coin back.", func() {
				if g.roll2d6Check(g.checkMod("intimidate"), 9) {
					g.cuEarn(a.Gold)
					g.Msg("{{G|They fish out your coin and hand it over, shaking.}}")
					a.Gold = 0
					g.cjFirst("recover", "Made a cutpurse give back stolen coin.")
					back("\"Take it, take it. Just don't call the watch.\"")
					return
				}
				a.AI = "wander"
				a.Set("cj_done")
				back("\"Don't know what you mean.\" The stranger slips into the dark.")
			}})
		}
	case "cj_nightmkt":
		o = append(o, Option{"Ask about the night market.", func() {
			g.cjFirst("nightmkt", "Found the night market where the day shops' forbidden goods are sold.")
			g.Talk = a
			g.talkMenu("\"Keep your voice down. Everything has a price. Nothing has a receipt.\"")
			g.Opts = append([]Option{{"Look over the wares.", func() { g.openTrade(a) }}}, g.Opts...)
		}})
	case "cj_ghost":
		o = append(o, Option{"Listen to the shade.", func() {
			t := pick(cjGhostLines, "g")
			if site, p, ok := g.nearestSite(); ok && NewRng(fmt.Sprintf("%s:cj:gh:%d", g.Seed, a.ID)).Chance(0.35) {
				g.addSecret(site, "dungeon", p, g.W.Year)
				t += "\n\n{{c|" + site + "}}"
			}
			back(t)
		}})
	case "cj_baker":
		o = append(o, Option{"Buy a warm loaf. {{K|(2)}}", func() {
			if !g.cuSpend(2) {
				back("\"Two coppers for a loaf, friend.\"")
				return
			}
			g.give(&Item{ID: g.newID(), Kind: "food", Name: "warm night loaf", Glyph: '%', F: CBrown, Value: 3, Power: 500})
			back("\"Still hot. Mind your fingers.\"")
		}})
	case "cj_healer":
		miss := g.P.MaxHP - g.P.HP
		cost := 3 + miss/3
		o = append(o, Option{fmt.Sprintf("Ask the street healer to tend your wounds. {{K|(%d)}}", cost), func() {
			if miss <= 0 {
				back("\"You look sound to me. Keep your coin.\"")
				return
			}
			if !g.cuSpend(cost) {
				back("\"I'd do it for less, but not for nothing.\"")
				return
			}
			g.P.HP = g.P.MaxHP
			g.cjFirst("healer", "Was patched up by a healer who works the streets at night.")
			back("Salve, a clean bandage and a murmured word. You feel whole again.")
		}})
	}
	return o
}

// ---- cutpurse ----

func (g *Game) cjCutpurse(a *Actor) {
	if g.P.Gold < 10 {
		return
	}
	a.Set("cj_done")
	dc := 8 + a.M(AGI)
	if g.roll2d6Check(g.checkMod("perception"), dc) {
		g.Msg("{{G|You catch a hand an inch from your purse. The stranger wrenches free and flees.}}")
		g.gainXP(8)
		g.cjFirst("catchpurse", "Caught a cutpurse in the act.")
		a.AI = "wander"
		return
	}
	amt := clampi(g.P.Gold/6, 3, 60)
	if !g.cuSpend(amt) {
		return
	}
	a.Gold = amt
	a.AI = "wander"
	g.Msg("{{R|A hooded stranger jostles you in the dark. Your purse feels lighter.}}")
	g.cjFirst("cutpurse", "Had a purse cut in the dark.")
}

// ---- curfew ----

func (g *Game) cjDrawn() bool {
	for _, p := range g.P.Body {
		if p.Slot == "hand" && !p.Severed && p.Item != nil && p.Item.Kind == "weapon" {
			return true
		}
	}
	return false
}

func (g *Game) cjCurfew() {
	zn := g.Cur
	tier := g.W.cityTier[zn.Burg]
	if tier < 1 || g.Talk != nil {
		return
	}
	st := g.cj()
	key := fmt.Sprintf("%d:%d", zn.Burg, g.cjNightID())
	if st.Fined[key] != 0 {
		return
	}
	state := g.uwStateHere()
	b := g.uwBounty(state)
	armed := g.cjDrawn()
	if !armed && !(b > 0 && b < 10) {
		return
	}
	var gd *Actor
	for _, a := range zn.Actors {
		if a.HP > 0 && !a.Angry && a.AI != "follower" && (a.Role == "guard" || a.Role == "cj_watch" || a.AI == "guard") && dist(a, g.P) <= 6 && g.visible(a.X, a.Y) {
			gd = a
			break
		}
	}
	if gd == nil {
		return
	}
	if g.R.Dice(2, 6)+g.ssMod() >= 12 { // slipped past in the dark
		return
	}
	st.Fined[key] = 1
	g.cjFirst("curfew", "Was stopped by the night watch for breaking curfew.")
	fine := 8 * (tier + 1)
	g.Talk, g.Mode = gd, "talk"
	why := "Drawn steel on the street after dark"
	if !armed {
		why = "You're known to the watch"
	}
	g.Popup = Wrap(fmt.Sprintf("\"Halt! %s. It's curfew in this town. That is a fine of %s, traveller, or we do this at the guardhouse.\"", why, g.cuFormat(fine)), 100)
	end := func(s string) { g.Msg(s); g.Mode, g.Talk = "main", nil }
	g.Opts = []Option{
		{fmt.Sprintf("Pay the night-fine. {{K|(%s)}}", g.cuFormat(fine)), func() {
			if g.cuSpend(fine) {
				end("{{y|You pay the fine. The watch waves you on.}}")
				return
			}
			g.uwAddBounty(state, fine, " for curfew")
			end("{{R|You cannot pay. They write your name in a book.}}")
		}},
		{"{{c|[Persuasion]}} Slip them a bribe and move along. " + fmt.Sprintf("{{K|(%s)}}", g.cuFormat(fine/2+1)), func() {
			if g.roll2d6Check(g.checkMod("persuasion"), 10) && g.cuSpend(fine/2+1) {
				g.crKarma(1, -1, "bribed the night watch")
				end("{{y|The coin vanishes. \"I didn't see you.\"}}")
				return
			}
			g.uwAddBounty(state, 6, " for attempted bribery")
			end("{{R|The watchman recoils. \"Bribery!\" You are fined and noted.}}")
		}},
		{"{{c|[Bluff]}} Say you were just walking home.", func() {
			if g.roll2d6Check(g.checkMod("bluff"), 11) {
				end("{{y|They look you over and let you go with a warning.}}")
				return
			}
			g.cuSpend(fine)
			end("{{R|They do not believe you and fine you anyway.}}")
		}},
		{"Refuse and walk on.", func() {
			gd.Angry = true
			g.uwAddBounty(state, 15, " for defying the curfew")
			g.Mode, g.Talk = "main", nil
		}},
	}
	g.Sel = 0
}

// cjShopClosed (one line in openTrade after snMarketClosed): the shutters of ordinary shops in
// villages and towns (tier 0-1) are down at night. Fences and night dealers still trade.
func (g *Game) cjShopClosed(a *Actor) bool {
	if g == nil || a == nil || g.Cur == nil || g.Cur.Kind != "town" || !g.cjNight() || a.Has("cj") || uwIsFence(a) || a.Role == "bmdealer" {
		return false
	}
	if g.cjTier() > 1 || !snShop(a) {
		return false
	}
	g.Talk, g.Mode = a, "talk"
	g.talkMenu(strings.ToLower("The shutters are down.") + " \"Come back at dawn.\"")
	return true
}
