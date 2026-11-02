package engine

import (
	"fmt"
	"math"
	"strings"
)

// Life at sea (prefix sh): the deck zone of your ship, encounters on the crossing (pirates,
// merchants, naval patrols, sea monsters, wrecks), the ship-combat screen (broadside, board, run,
// pay), boarding actions fought in a real 80×25 zone of two lashed decks, salvage dives to wrecks
// on the sea floor and fishing over the side.

type shFight struct {
	Kind    string `json:"k"` // pirate patrol merchant monster
	Name    string `json:"n"`
	Hull    int    `json:"h"`
	MaxHull int    `json:"mh"`
	Guns    int    `json:"g"`
	Kmd     int    `json:"v"`
	Crew    int    `json:"c"`
	Level   int    `json:"l"`
	State   int    `json:"s,omitempty"` // patrol: realm
	Fine    int    `json:"fi,omitempty"`
	Prize   int    `json:"pr,omitempty"`
	Tpl     string `json:"tp,omitempty"` // monster template
	SimMon  int    `json:"sm,omitempty"` // a Realmforge monster of the sea
	Foes    []int  `json:"fo,omitempty"`
}

// ---- the deck ----

// shLayDeck paints a hull of size by crew berths centred on (cx, 12), deck planks on deep water.
func shLayDeck(zn *Zone, cx, half, beam int) {
	for y := 12 - beam; y <= 12+beam; y++ {
		for x := cx - half; x <= cx+half; x++ {
			dx := float64(x-cx) / float64(half)
			dy := float64(y-12) / float64(beam)
			// a pointed bow to the east, a square stern to the west
			lim := 1.0
			if dx > 0.4 {
				lim = 1 - (dx-0.4)/0.6
			}
			if math.Abs(dy) <= lim+0.01 && dx >= -1 {
				zn.Set(x, y, TBoat)
			}
		}
	}
	zn.Set(cx, 12, TPillar) // mainmast
	if half > 14 {
		zn.Set(cx-half/2, 12, TPillar)
		zn.Set(cx+half/2, 12, TPillar)
	}
}

func shNewSeaZone(g *Game, key, name string, p int) *Zone {
	w := g.W
	zn := newZone(key, p%w.W, p/w.W, 1, 1, shDeckZ)
	zn.Kind, zn.Name, zn.Level, zn.Biome = "ship", name, g.zoneLevel(p%w.W, p/w.W, 0), w.Biome[p]
	for i := range zn.T {
		zn.T[i] = TDeep
	}
	return zn
}

func shDeckSize(k *shKind) (int, int) {
	return clampi(8+k.MaxCrew/2, 9, 28), clampi(2+k.Hull/14, 3, 7)
}

// shEnterDeck walks you onto your ship's deck at sea (or in port).
func (g *Game) shEnterDeck() {
	st := g.sh()
	sp := st.Ship
	if sp == nil {
		return
	}
	zn := g.Zones[shDeckKey]
	if zn == nil || zn.Name != "Deck of the "+sp.Name {
		zn = shNewSeaZone(g, shDeckKey, "Deck of the "+sp.Name, sp.At)
		half, beam := shDeckSize(sp.K())
		shLayDeck(zn, 40, half, beam)
		zn.Set(40-half+1, 12, TInscription)
		zn.Notes[12*MW+40-half+1] = "The helm. Z for the ship; walk off the deck to take the helm."
		zn.Set(40+2, 12, TCrate)
		g.computeLight(zn)
		g.Zones[shDeckKey] = zn
	}
	zn.WX, zn.WY = sp.At%g.W.W, sp.At/g.W.W
	zn.Biome = g.W.Biome[sp.At]
	// the hired hands, as figures on deck (not combatants)
	keep := zn.Actors[:0]
	for _, a := range zn.Actors {
		if !a.Has("shhand") {
			keep = append(keep, a)
		}
	}
	zn.Actors = keep
	r := NewRng(fmt.Sprintf("%s:sh:hands:%d", g.Seed, g.Turn/DayLen))
	for i := 0; i < mini(sp.Crew, 8); i++ {
		a := g.shSailor(r, "shhand")
		a.AI = "still"
		if x, y, ok := zn.FreeSpot(r, func(x, y int) bool { return zn.At(x, y) == TBoat }); ok {
			a.X, a.Y, a.HomeX, a.HomeY = x, y, x, y
			zn.Actors = append(zn.Actors, a)
		}
	}
	sp.Aboard = true
	g.enterZone(zn, 40-3, 12, true)
	g.Mode = "main"
	g.recompute()
}

// shSailor is one of your crew as an actor.
func (g *Game) shSailor(r *Rng, flag string) *Actor {
	a := g.Spawn("bandit", maxi(1, g.P.Level-1), r)
	a.Name, a.Role, a.Faction, a.Glyph, a.F = "sailor", "crew", "shcrew", '@', CCyan
	a.Desc = "One of your crew, tarred hands and a knife in the belt."
	a.Set(flag)
	return a
}

// ---- encounters ----

func (g *Game) shNearState(p int) int {
	w := g.W
	best, bd := 0, 7
	for _, b := range w.Burgs {
		if !b.Alive || !b.Port || b.State == 0 || b.Layer != "surface" {
			continue
		}
		if d := maxi(abs(b.P%w.W-p%w.W), abs(b.P/w.W-p/w.W)); d < bd {
			best, bd = b.State, d
		}
	}
	return best
}

var shPirateNames = []string{"Red Corsair", "Drowned Jack", "Sea Wolf", "Black Gull", "Widowmaker", "Salt Hag", "Grinning Shark", "Kraken's Due"}

// shEncounterRoll may start a meeting at sea after a leg of days.
func (g *Game) shEncounterRoll(sp *shShip, np int, days float64) {
	st := g.sh()
	if st.Fight != nil || st.Ship != sp || st.quiet {
		return
	}
	for _, mi := range g.W.monstersAt[np] {
		m := &g.W.Monsters[mi]
		if !m.Slain && !g.Slain[m.ID] && (m.Layer == "surface" || m.Layer == "sea" || m.Layer == "") && g.R.Chance(0.5) {
			st.Fight = &shFight{Kind: "monster", Name: m.Name, SimMon: m.ID, Level: clampi(m.Power/7, 8, 35), Kmd: 300}
			g.shFightMenu()
			return
		}
	}
	chance := math.Min(0.45, 0.16*days+0.04)
	if !g.R.Chance(chance) {
		return
	}
	lane := len(g.W.nmRoutes(np, "sea")) > 0
	state := g.shNearState(np)
	wPirate, wMerch, wPatrol, wMon, wWreck := 30.0, 25.0, 0.0, 12.0, 18.0
	if lane {
		wPirate, wMerch = 15, 35
	}
	if state > 0 {
		wPatrol = 15
		if lane {
			wPatrol = 25
		}
	} else {
		wPirate += 15
	}
	if g.Genre == "wasteland" {
		wPirate += 15
	}
	roll := g.R.Float() * (wPirate + wMerch + wPatrol + wMon + wWreck)
	lvl := maxi(1, g.zoneLevel(g.WX, g.WY, 0)+g.R.Int(0, 2))
	switch {
	case roll < wPirate:
		guns := g.R.Int(0, 3) + lvl/4
		st.Fight = &shFight{Kind: "pirate", Name: "the " + Pick(g.R, shPirateNames), Hull: 16 + lvl*2 + guns*4, Guns: guns, Kmd: 170 + g.R.Int(0, 80), Crew: g.R.Int(3, 6), Level: lvl, Prize: 40 + lvl*20}
	case roll < wPirate+wMerch:
		st.Fight = &shFight{Kind: "merchant", Name: "the merchantman " + g.shNewName(fmt.Sprint("m", g.Turn)), Hull: 30, Guns: 1, Kmd: 160, Crew: 6, Level: lvl}
	case roll < wPirate+wMerch+wPatrol:
		f := &shFight{Kind: "patrol", Name: "a patrol of " + g.W.StateName(state), State: state, Hull: 30 + lvl*2, Guns: 2 + lvl/3, Kmd: 220, Crew: g.R.Int(4, 7), Level: lvl + 1}
		f.Fine = g.uwBounty(state)
		if f.Fine == 0 && g.Rep[fmt.Sprintf("state:%d", state)] > -250 {
			g.AddRep(fmt.Sprintf("state:%d", state), 2, "")
			g.Msg(fmt.Sprintf("{{C|A %s warship of %s hails you, looks you over, and lets you pass.}}", sp.K().Name, g.W.StateName(state)))
			return
		}
		if f.Fine == 0 {
			f.Fine = 50 + lvl*10
		}
		st.Fight = f
	case roll < wPirate+wMerch+wPatrol+wMon:
		s := g.shSeaBeast(np, true)
		if s == nil {
			return
		}
		st.Fight = &shFight{Kind: "monster", Name: bsArticle(strings.ToLower(s.Name)), Tpl: s.Tpl(), Level: lvl + 2, Kmd: 250}
	default:
		wr := shWreck{P: np, Name: "the " + g.shNewName(fmt.Sprint("w", np))}
		for _, w := range st.Wrecks {
			if w.P == np {
				return
			}
		}
		st.Wrecks = append(st.Wrecks, wr)
		g.Msg(fmt.Sprintf("{{C|Your lookout spots wreckage: %s lies broken on the bottom here.}} (Z: dive to salvage)", trCap(wr.Name)))
		return
	}
	if st.Fight != nil {
		st.Fight.MaxHull = st.Fight.Hull
		g.shFightMenu()
	}
}

// shSeaBeast picks a sea creature of the bestiary for parasang p (big predators or fish).
func (g *Game) shSeaBeast(p int, big bool) *bsSp {
	temp := g.W.TempAt(p, g.Turn)
	var pool []*bsSp
	for _, s := range bsAll {
		if s.Sea == 0 || s.Era != "living" && !(big && s.Era == "mesozoic" && g.Genre == "mythic") {
			continue
		}
		if s.Sea == 2 && temp < 16 || s.Sea == 3 && temp > 12 {
			continue
		}
		if big && (s.Size < 5 || !s.Hostile()) || !big && (s.Group != "fish" || s.Size > 5) {
			continue
		}
		pool = append(pool, s)
	}
	if len(pool) == 0 {
		return nil
	}
	return Pick(g.R, pool)
}

// ---- the fight screen ----

func (g *Game) shFightMenu() {
	st := g.sh()
	f, sp := st.Fight, st.Ship
	if f == nil || sp == nil {
		st.Fight = nil
		return
	}
	g.Talk = nil
	st.Title, st.Back = "At sea", "world"
	if !g.OnWorld {
		st.Back = "main"
	}
	g.Opts = nil
	add := func(t string, fn func()) { g.Opts = append(g.Opts, Option{t, fn}) }
	gw := shGuns(g.Genre)
	status := fmt.Sprintf("{{K|Your %s: hull %d/%d, %d %s, %d crew. Theirs: hull %d/%d, %d %s.}}", sp.Name, sp.Hull, sp.MaxHull, sp.K().Guns, gw, g.shHands(sp), f.Hull, f.MaxHull, f.Guns, gw)
	runDC := clampi(7+(f.Kmd-sp.K().Kmd)/60, 4, 12)
	switch f.Kind {
	case "pirate":
		g.Popup = Wrap(fmt.Sprintf("{{R|Sails on the horizon: %s, pirates, bearing down on you!}} %s", trCap(f.Name), status), 100)
		g.shCombatOpts(add, runDC)
		toll := clampi(g.P.Gold/4, 20, 60+f.Level*40)
		add(fmt.Sprintf("Pay them off. — %s", g.cuFormat(toll)), func() {
			if !g.cuSpend(toll) {
				g.shFightLine("\"That's not enough to buy your life.\"")
				return
			}
			st.Fight = nil
			g.shEnd("The pirates take their tribute and sheer off, laughing.")
		})
	case "patrol":
		g.Popup = Wrap(fmt.Sprintf("{{y|%s runs out its guns and orders you to heave to.}} \"You are wanted in %s. Pay %s or be taken.\" %s", trCap(f.Name), g.W.StateName(f.State), g.cuFormat(f.Fine), status), 100)
		add(fmt.Sprintf("Pay the fine. — %s", g.cuFormat(f.Fine)), func() {
			if !g.cuSpend(f.Fine) {
				g.shFightLine("\"Then we take it from your hold.\"")
				return
			}
			g.uwSt().Bounty[f.State] = 0
			st.Fight = nil
			g.shEnd("The officer counts the coin and waves you on.")
		})
		g.shCombatOpts(add, runDC)
	case "merchant":
		g.Popup = Wrap(fmt.Sprintf("{{C|%s comes alongside and dips her flag in greeting.}}", trCap(f.Name)), 100)
		cost := 12
		add(fmt.Sprintf("Buy 15 days of stores. — %s", g.cuFormat(cost)), func() {
			if !g.cuSpend(cost) {
				g.shFightLine("\"Coin, friend.\"")
				return
			}
			sp.Prov += 15
			g.shFightLine("Barrels swing across on a line.")
		})
		add("Swap news of the ports.", func() {
			n := 0
			for _, b := range g.W.Burgs {
				if b.Alive && b.Port && b.Layer == "surface" && !g.Known[b.P] && n < 3 {
					g.Known[b.P] = true
					n++
				}
			}
			g.shFightLine(fmt.Sprintf("Their captain marks %d harbours on your chart.", n))
		})
		add("Sell your catch and ingredients.", func() {
			total := 0
			keep := g.P.Inv[:0]
			for _, it := range g.P.Inv {
				if it.Kind == "ingredient" && it.Quest == 0 {
					total += maxi(1, it.Value) * maxi(1, it.Count)
					continue
				}
				keep = append(keep, it)
			}
			g.P.Inv = keep
			g.cuEarn(total)
			g.shFightLine(fmt.Sprintf("They pay %s for the lot.", g.cuFormat(total)))
		})
		add("Sail on.", func() { st.Fight = nil; g.shEnd("You part with a wave.") })
	case "monster":
		g.Popup = Wrap(fmt.Sprintf("{{R|The sea heaves: %s rises beside the %s!}} %s", f.Name, sp.Name, status), 100)
		add("{{R|Fight it from the deck.}}", func() { g.shBoardAction() })
		if sp.K().Guns > 0 {
			add("Loose the "+gw+" at it.", func() {
				if g.roll2d6Check(g.checkMod("perception")+sp.K().Guns/2, 9) {
					st.Fight = nil
					g.gainXP(20 + f.Level*5)
					g.shEnd(fmt.Sprintf("{{G|A volley strikes home; %s sinks away, trailing blood.}}", f.Name))
					return
				}
				d := g.R.Int(3, 8)
				sp.Hull -= d
				if sp.Hull <= 0 {
					g.Mode = st.Back
					g.shSink(sp, "was dragged under by "+f.Name)
					return
				}
				g.shFightLine(fmt.Sprintf("{{R|You miss; it rams the hull for %d.}}", d))
			})
		}
		g.shRunOpt(add, runDC)
	}
	g.Sel = 0
	g.Mode = "ship"
}

func (g *Game) shCombatOpts(add func(string, func()), runDC int) {
	st := g.sh()
	f, sp := st.Fight, st.Ship
	if sp.K().Guns > 0 {
		add("{{Y|Broadside!}} "+g.checkLabel("perception", 8-sp.K().Guns/2), func() {
			hit := g.roll2d6Check(g.checkMod("perception")+sp.K().Guns/2, 8)
			line := "{{R|Your volley falls short.}}"
			if hit {
				d := sp.K().Guns*g.R.Int(2, 4) + g.R.Int(0, 3)
				f.Hull -= d
				line = fmt.Sprintf("{{G|Your %s smash into %s: %d damage.}}", shGuns(g.Genre), f.Name, d)
			}
			if f.Hull <= 0 {
				st.Fight = nil
				g.shVictory(f, "strikes her colours")
				return
			}
			if f.Guns > 0 && g.R.Chance(0.6) {
				d := f.Guns*g.R.Int(1, 3) + g.R.Int(0, 2)
				sp.Hull -= d
				line += fmt.Sprintf(" {{R|They answer: %d damage to your hull.}}", d)
				if sp.Hull <= 0 {
					g.Mode = st.Back
					g.shSink(sp, "was sunk by "+f.Name)
					return
				}
			}
			g.shFightLine(line)
		})
	}
	add("{{R|Grapple and board them!}}", func() { g.shBoardAction() })
	g.shRunOpt(add, runDC)
}

func (g *Game) shRunOpt(add func(string, func()), dc int) {
	add("Run for it. "+g.checkLabel("acrobatics", dc), func() {
		if g.roll2d6Check(g.checkMod("acrobatics"), dc) {
			g.sh().Fight = nil
			g.shEnd("You crowd on sail and leave them in your wake.")
			return
		}
		g.shFlee(false)
	})
}

// shFlee: a failed escape (or cutting the grapples) costs hull and ends the fight.
func (g *Game) shFlee(fromBoard bool) {
	st := g.sh()
	sp := st.Ship
	st.Fight = nil
	if sp == nil {
		return
	}
	d := g.R.Int(2, 7)
	sp.Hull -= d
	if sp.Hull <= 0 {
		g.Mode = "world"
		g.shSink(sp, "was sunk while fleeing")
		return
	}
	if fromBoard {
		g.shClearBoard()
		g.shHelm()
		g.Msg(fmt.Sprintf("{{R|You cut the grapples and run. Shot follows you: %d hull damage.}}", d))
		return
	}
	g.shEnd(fmt.Sprintf("{{R|They rake you as you run: %d hull damage, but you get away.}}", d))
}

func (g *Game) shFightLine(s string) {
	g.Msg(s)
	if g.sh().Fight != nil {
		g.shFightMenu()
		g.Popup = append(Wrap(s, 100), append([]string{""}, g.Popup...)...)
	}
}

func (g *Game) shEnd(s string) {
	g.Msg(s)
	b := g.sh().Back
	if b == "" {
		b = "world"
	}
	g.Mode = b
}

func (g *Game) shVictory(f *shFight, how string) {
	g.cuEarn(f.Prize)
	g.gainXP(25 + f.Level*8)
	if sp := g.sh().Ship; sp != nil {
		sp.Morale = mini(100, sp.Morale+15)
	}
	if f.Kind == "patrol" {
		g.AddRep(fmt.Sprintf("state:%d", f.State), -150, " for fighting their navy")
	}
	g.deed("ship", fmt.Sprintf("%s beat %s at sea.", g.P.Name, f.Name), 0, 0)
	g.shEnd(fmt.Sprintf("{{G|%s %s!}} You take %s in plunder.", trCap(f.Name), how, g.cuFormat(f.Prize)))
}

// ---- boarding ----

// shBoardAction lays out your deck lashed to theirs and puts you on it: a real fight.
func (g *Game) shBoardAction() {
	st := g.sh()
	f, sp := st.Fight, st.Ship
	if f == nil || sp == nil {
		return
	}
	r := NewRng(fmt.Sprintf("%s:sh:board:%d", g.Seed, g.Turn))
	zn := shNewSeaZone(g, shBoardKey, "Boarding "+f.Name, sp.At)
	half, beam := shDeckSize(sp.K())
	half = mini(half, 16)
	ox := 4 + half
	shLayDeck(zn, ox, half, beam)
	f.Foes = nil
	if f.Kind == "monster" {
		a := g.shMonster(f, r)
		a.X, a.Y = ox+half+2, 12
		zn.Actors = append(zn.Actors, a)
		f.Foes = append(f.Foes, a.ID)
	} else {
		ex := MW - 4 - 14
		shLayDeck(zn, ex, 14, 5)
		for _, y := range []int{10, 12, 14} {
			for x := ox; x <= ex; x++ {
				if zn.At(x, y) == TDeep {
					zn.Set(x, y, TBridge) // gangplanks and grapples
				}
			}
		}
		for i := 0; i < f.Crew; i++ {
			tp := "bandit"
			a := g.Spawn(tp, f.Level, r)
			switch f.Kind {
			case "patrol":
				a.Name, a.Faction = "marine", fmt.Sprintf("state:%d", f.State)
			default:
				a.Name = "pirate"
				if i == 0 {
					a.Name, a.Proper = "Captain of "+f.Name, true
					a.MaxHP += 10
					a.HP = a.MaxHP
				}
			}
			a.Angry, a.AI = true, "hostile"
			if x, y, ok := zn.FreeSpot(r, func(x, y int) bool { return zn.At(x, y) == TBoat && x > ex-10 }); ok {
				a.X, a.Y = x, y
				zn.Actors = append(zn.Actors, a)
				f.Foes = append(f.Foes, a.ID)
			}
		}
		// their hold: plunder lies on the enemy deck
		for i := 0; i < 2+f.Level/4; i++ {
			it := g.Loot(r, f.Level/3)
			if x, y, ok := zn.FreeSpot(r, func(x, y int) bool { return zn.At(x, y) == TBoat && x > ex }); ok {
				it.X, it.Y = x, y
				zn.Items = append(zn.Items, it)
			}
		}
		zn.Set(ex+3, 12, TCrate)
	}
	for i := 0; i < mini(sp.Crew, 6); i++ {
		a := g.shSailor(r, "shcrew")
		a.AI = "follower"
		if x, y, ok := zn.FreeSpot(r, func(x, y int) bool { return zn.At(x, y) == TBoat && x < ox }); ok {
			a.X, a.Y = x, y
			zn.Actors = append(zn.Actors, a)
		}
	}
	g.computeLight(zn)
	g.Zones[shBoardKey] = zn
	g.enterZone(zn, ox, 12, true)
	g.Mode, g.OnWorld = "main", false
	g.recompute()
	g.Msg("{{R|Steel out! Walk off the deck's edge to cut the grapples and flee.}}")
}

// shMonster is the beast of a sea fight: a Realmforge legend or a bestiary predator.
func (g *Game) shMonster(f *shFight, r *Rng) *Actor {
	var m *MonsterData
	for i := range g.W.Monsters {
		if f.SimMon > 0 && g.W.Monsters[i].ID == f.SimMon {
			m = &g.W.Monsters[i]
		}
	}
	if m == nil {
		a := g.Spawn(f.Tpl, f.Level, r)
		a.Angry, a.AI = true, "hostile"
		return a
	}
	a := g.Spawn("horror", f.Level, r)
	a.Name, a.Proper, a.Boss, a.SimMon, a.Desc = m.Name, true, true, m.ID, m.Lore
	if sp := []rune(strings.ToUpper(m.Species)); len(sp) > 0 {
		a.Glyph = sp[0]
	}
	a.F, a.Faction, a.Angry, a.AI = CRed, "legends", true, "hostile"
	a.MaxHP = maxi(a.MaxHP, m.HP)
	a.HP = a.MaxHP
	a.AVNat += int(m.Armor * 14)
	a.XP, a.Gold = 400+m.Power*20, 100+m.Power*3
	if m.Trophy != "" {
		a.Inv = append(a.Inv, &Item{ID: g.newID(), Kind: "trophy", Name: m.Trophy, Glyph: '♦', F: CYellow, Value: 200 + m.Power*4, Lore: "Proof you slew " + m.Name + "."})
	}
	return a
}

// ---- islands ----

var shIsleCache = map[*WorldData]map[int]int{}

// shIsleSize is the land area (parasangs, capped at 9) of the island holding p; 0 for sea.
func shIsleSize(w *WorldData, p int) int {
	if w.IsWater(p) {
		return 0
	}
	c := shIsleCache[w]
	if c == nil {
		c = map[int]int{}
		shIsleCache[w] = c
	}
	if n, ok := c[p]; ok {
		return n
	}
	seen := map[int]bool{p: true}
	q := []int{p}
	for len(q) > 0 && len(seen) < 10 {
		x := q[0]
		q = q[1:]
		for _, d := range dirs8 {
			if !w.In(x%w.W+d[0], x/w.W+d[1]) {
				continue
			}
			n := w.Idx(x%w.W+d[0], x/w.W+d[1])
			if !seen[n] && !w.IsWater(n) {
				seen[n] = true
				q = append(q, n)
			}
		}
	}
	n := mini(len(seen), 9)
	c[p] = n
	return n
}

// shIsleZone hides a castaway's cache on small, empty islands (zone hook, side rng only).
func shIsleZone(g *Game, zn *Zone, _ *Rng) {
	if zn.Z != 0 || zn.ZX != 1 || zn.ZY != 1 || zn.Kind != "surface" || zn.Burg > 0 {
		return
	}
	w := g.W
	p := w.Idx(zn.WX, zn.WY)
	if w.Burg[p] > 0 || shIsleSize(w, p) > 6 || HashStr(g.Seed+":sh:isle:"+zn.Key)%2 != 0 {
		return
	}
	r := NewRng(g.Seed + ":sh:isle:" + zn.Key)
	x, y, ok := zn.FreeSpot(r, func(x, y int) bool {
		return abs(x-40) < 25 && abs(y-12) < 8 && zn.At(x, y).D().Water == 0 && zn.ActorAt(x, y) == nil
	})
	if !ok {
		return
	}
	notes := []string{
		"A cairn of sea-worn stones. Scratched on the top one: \"Left here by the crew of %s. Dig if you are hungry.\"",
		"A driftwood cross over a mound. \"%s — we buried what we could not carry.\"",
		"A cave mouth stacked with shells in a spiral, and a tally of days scratched beside it: the last mark of %s.",
	}
	zn.Set(x, y, TInscription)
	zn.Notes[y*MW+x] = fmt.Sprintf(Pick(r, notes), "the "+g.shNewName(zn.Key))
	put := func(it *Item) {
		if ix, iy, ok := zn.FreeSpot(r, func(ix, iy int) bool { return abs(ix-x) <= 2 && abs(iy-y) <= 2 && zn.At(ix, iy).D().Water == 0 }); ok {
			it.X, it.Y = ix, iy
			zn.Items = append(zn.Items, it)
		}
	}
	put(g.Loot(r, 1+zn.Level/3))
	put(g.Loot(r, 1+zn.Level/3))
	put(&Item{ID: g.newID(), Kind: "gold", Name: g.Sk.Currency, Glyph: '$', F: CYellow, Count: 20 + r.Int(0, 50)})
}

// shBoardTick ends a boarding action when every foe is down (turn hook).
func shBoardTick(g *Game) {
	st := g.sh()
	f := st.Fight
	if f == nil || g.Cur == nil || g.Cur.Key != shBoardKey {
		return
	}
	for _, a := range g.Cur.Actors {
		for _, id := range f.Foes {
			if a.ID == id && a.HP > 0 {
				return
			}
		}
	}
	st.Fight = nil
	if f.Kind == "monster" {
		g.gainXP(30 + f.Level*8)
		g.deed("ship", fmt.Sprintf("%s slew %s from the deck of the %s.", g.P.Name, f.Name, st.Ship.Name), 0, 0)
		g.Msg(fmt.Sprintf("{{G|%s is dead! Its carcass rolls alongside.}} Walk off the deck to take the helm.", trCap(f.Name)))
		return
	}
	g.gainXP(25 + f.Level*8)
	g.cuEarn(f.Prize)
	if f.Kind == "patrol" {
		g.AddRep(fmt.Sprintf("state:%d", f.State), -150, " for fighting their navy")
	}
	if sp := st.Ship; sp != nil {
		sp.Morale = mini(100, sp.Morale+15)
	}
	g.deed("ship", fmt.Sprintf("%s boarded and took %s.", g.P.Name, f.Name), 0, 0)
	g.Msg(fmt.Sprintf("{{G|The deck is yours! %s is taken.}} Their strongbox holds %s; the rest lies on their deck. Walk off the edge to take the helm.", trCap(f.Name), g.cuFormat(f.Prize)))
}

// shClearBoard removes the hired hands from the boarding zone.
func (g *Game) shClearBoard() {
	if zn := g.Zones[shBoardKey]; zn != nil {
		keep := zn.Actors[:0]
		for _, a := range zn.Actors {
			if !a.Has("shcrew") {
				keep = append(keep, a)
			}
		}
		zn.Actors = keep
	}
}

// shEnter: tidies ship zones behind you, and a diver surfacing at the ship climbs aboard.
func shEnter(g *Game, zn *Zone) {
	st := g.sh()
	if zn.Key != shBoardKey && g.Zones[shBoardKey] != nil {
		delete(g.Zones, shBoardKey)
	}
	keep := zn.Actors[:0]
	for _, a := range zn.Actors {
		if !a.Has("shcrew") || zn.Key == shBoardKey {
			keep = append(keep, a)
		}
	}
	zn.Actors = keep
	sp := st.Ship
	if sp == nil {
		return
	}
	p := g.W.Idx(zn.WX, zn.WY)
	if zn.Z == 1 && zn.Underwater {
		for i := range st.Wrecks {
			if wr := &st.Wrecks[i]; wr.P == p && !wr.Placed && zn.ZX == 1 && zn.ZY == 1 {
				g.shPlaceWreck(zn, wr)
			}
		}
	}
	if zn.Z == 0 && sp.Aboard && p == sp.At && g.W.IsWater(p) && st.Deck == 1 {
		st.Deck = 0
		g.Msg("Your crew hauls you back aboard.")
		g.shEnterDeck()
	}
}

// ---- salvage ----

// shPlaceWreck lays a broken hull with cargo on the sea floor zone.
func (g *Game) shPlaceWreck(zn *Zone, wr *shWreck) {
	wr.Placed = true
	r := NewRng(fmt.Sprintf("%s:sh:wreck:%d", g.Seed, wr.P))
	for y := 9; y <= 15; y++ {
		for x := 28; x <= 52; x++ {
			if abs(y-12)*4 <= 12-abs(x-40)/3 {
				if r.Chance(0.8) {
					zn.Set(x, y, TBoat)
				} else {
					zn.Set(x, y, TRubble)
				}
			}
		}
	}
	zn.Set(40, 12, TCrate)
	zn.Set(36, 12, TInscription)
	zn.Notes[12*MW+36] = "A name board, barnacled: " + trCap(wr.Name) + "."
	zn.Set(30, 12, TUp)
	for i := 0; i < 3; i++ {
		it := g.Loot(r, maxi(1, zn.Level/3)+1)
		it.X, it.Y = 38+i*2, 11+i%2*2
		zn.Items = append(zn.Items, it)
	}
	zn.Items = append(zn.Items, &Item{ID: g.newID(), Kind: "gold", Name: g.Sk.Currency, Glyph: '$', F: CYellow, Count: 30 + r.Int(0, 60) + zn.Level*10, X: 44, Y: 12})
	g.computeLight(zn)
}

// shDive goes down to a wreck: on the sea floor where you can, else your crew salvages it.
func (g *Game) shDive(wr *shWreck) {
	st := g.sh()
	sp := st.Ship
	p := wr.P
	w := g.W
	if w.seaOpen[p] {
		zn := g.GetZone(p%w.W, p/w.W, 1, 1, 1)
		if !wr.Placed {
			g.shPlaceWreck(zn, wr)
		}
		wr.Salvaged = true
		st.Deck = 1
		sp.Aboard = true
		g.Msg("{{C|You go over the side and down the anchor line to the wreck.}} Climb the line (<) to return.")
		g.enterZone(zn, 31, 12, true)
		g.Mode, g.OnWorld = "main", false
		g.recompute()
		return
	}
	wr.Salvaged = true
	g.tvPass(240)
	r := NewRng(fmt.Sprintf("%s:sh:wreck:%d", g.Seed, p))
	if !g.roll2d6Check(g.checkMod("survival")+g.shHands(sp)/3, 8) {
		g.shMenuMsg("The wreck lies too deep; your divers come up empty-handed.")
		return
	}
	n := 1 + r.Int(0, 2)
	for i := 0; i < n; i++ {
		sp.Hold = append(sp.Hold, g.Loot(r, 2))
	}
	g.shMenuMsg(fmt.Sprintf("Your divers bring up %d lots from %s into the hold.", n, wr.Name))
}

// ---- fishing ----

func (g *Game) shFish() {
	st := g.sh()
	sp := st.Ship
	g.tvPass(120)
	st.Fished = g.Turn
	if g.R.Chance(0.04) && st.Fight == nil {
		if s := g.shSeaBeast(sp.At, true); s != nil {
			st.Fight = &shFight{Kind: "monster", Name: bsArticle(strings.ToLower(s.Name)), Tpl: s.Tpl(), Level: g.P.Level + 2, Kmd: 250}
			g.Msg("{{R|Something enormous takes the bait!}}")
			g.shFightMenu()
			return
		}
	}
	s := g.shSeaBeast(sp.At, false)
	if s == nil || !g.roll2d6Check(g.checkMod("survival"), 7) {
		g.shMenuMsg("Two hours with a line over the side, and nothing bites.")
		return
	}
	n := 1 + g.R.Int(0, 1+g.Skills["survival"])
	it := &Item{ID: g.newID(), Kind: "ingredient", Name: strings.ToLower(s.Name) + " fillet", Effect: bsMeatTag(s), Glyph: '%', F: CRed, Count: n, Value: 3 + s.Size, Power: 120, Weight: 1}
	g.give(it)
	g.shMenuMsg(fmt.Sprintf("You land %s: %d fillets of %s.", bsArticle(strings.ToLower(s.Name)), n, strings.ToLower(s.Name)))
}

func init() {
	turnHooks = append(turnHooks, shBoardTick)
	zoneHooks = append(zoneHooks, shIsleZone)
	enterHooks = append(enterHooks, shEnter)
}
