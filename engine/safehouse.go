package engine

import (
	"fmt"
	"strings"
)

// Safehouses and lodgings (prefix sy): places to sleep and hide that are not settlements and not
// founding anything. Two kinds:
//  1. INN ROOMS: rent a room from an innkeeper for nights (talk option, cost in local money). Sleep
//     heals fully. The room has a lockable chest (screen). Items stay while the rent is paid; when the
//     paid period ends the innkeeper holds them for 14 days (you may redeem them by paying the back
//     rent) and then sells them. The lists are []*Item in g.ext("sy"), so they survive save and load.
//  2. SAFEHOUSES: take over a cleared hideout or an abandoned building through home.go's public
//     helpers (claimHome, homeHere, openStash, sleepHome), then buy upgrades with gold and days: better
//     locks, a hidden stash, a bolt-hole tunnel, a rain barrel and kitchen, a proper bed. Upkeep, a
//     landlord's weekly ground rent (not underground), burglary by thieves scaled with your notoriety
//     (bounties, stolen goods, heat) and a rival who tips off the guards when heat is high.
// LYING LOW decays heat from heists (htHeat/htCool in heists.go) and the local bounty (uwBounty,
// uwClearBounty). Screen: mode "sy", palette "Lodgings and safehouses". State g.ext("sy"), planet-local.

const syHoldDays = 14

type syRoom struct {
	Burg   int     `json:"b"`
	Zone   string  `json:"z"`
	Town   string  `json:"tn"`
	Rate   int     `json:"r"`
	Paid   int     `json:"p"` // game minute the rent runs to
	Stash  []*Item `json:"s,omitempty"`
	Held   bool    `json:"h,omitempty"`
	HoldTo int     `json:"ht,omitempty"`
	Gone   bool    `json:"g,omitempty"`
}

type syHouse struct {
	Key      string         `json:"k"`
	Name     string         `json:"n"`
	Since    int            `json:"s"`
	Upg      map[string]int `json:"u,omitempty"`
	Build    map[string]int `json:"b,omitempty"` // upgrade id -> minute the latest level is ready
	Hidden   []*Item        `json:"hi,omitempty"`
	Rate     int            `json:"r"`
	Landlord bool           `json:"ll,omitempty"`
	RentDue  int            `json:"rd,omitempty"`
	Watched  int            `json:"w,omitempty"` // minute until the watch is watching it
	Seized   bool           `json:"sz,omitempty"`
	Burgled  int            `json:"bg,omitempty"`
	Raids    int            `json:"ra,omitempty"`
}

type syState struct {
	Rooms  []*syRoom  `json:"r,omitempty"`
	Houses []*syHouse `json:"h,omitempty"`
	Day    int        `json:"d,omitempty"`
	Log    []string   `json:"l,omitempty"`
	Slept  int        `json:"sl,omitempty"`
	Low    int        `json:"lo,omitempty"`

	tab, sel int
	conf     string
	cur      string // open chest: "room:<i>" or "hid:<i>"
	stTab    int
	stSel    [2]int
}

func (g *Game) sy() *syState {
	s := g.ext("sy", func() any { return &syState{} }).(*syState)
	for _, h := range s.Houses {
		if h.Upg == nil {
			h.Upg = map[string]int{}
		}
		if h.Build == nil {
			h.Build = map[string]int{}
		}
	}
	return s
}

func init() {
	unLocalKeys = append(unLocalKeys, "sy")
	talkHooks = append(talkHooks, syTalk)
	turnHooks = append(turnHooks, syTurn)
	enterHooks = append(enterHooks, syEnter)
	sideHooks = append(sideHooks, sySide)
	modeKeys["sy"] = syKeys
	modeDraws["sy"] = func(g *Game) { g.syDraw() }
	modeKeys["syst"] = syStashKeys
	modeDraws["syst"] = func(g *Game) { g.syStashDraw() }
	paEntries = append(paEntries, paEntry{Name: "Lodgings and safehouses", Keywords: "inn room rent stash chest safehouse hideout lie low hide heat bounty sleep upgrade", Cat: "Screen", Label: "screen",
		Func: func(g *Game) { g.syOpen(0) }})
}

// ---------------------------------------------------------------- helpers

func (g *Game) syLog(s string) {
	st := g.sy()
	st.Log = append(st.Log, fmt.Sprintf("Day %d: %s", DayOf(g.Turn)+1, s))
	if len(st.Log) > 10 {
		st.Log = st.Log[len(st.Log)-10:]
	}
}

func (g *Game) syRoomHere() *syRoom {
	if g.Cur == nil {
		return nil
	}
	for _, r := range g.sy().Rooms {
		if !r.Gone && r.Zone == g.Cur.Key {
			return r
		}
	}
	return nil
}

func (g *Game) syHouseHere() *syHouse {
	if g.Cur == nil {
		return nil
	}
	for _, h := range g.sy().Houses {
		if h.Key == g.Cur.Key {
			return h
		}
	}
	return nil
}

// syLvl is the finished level of an upgrade: a level still being built does not count yet.
func (g *Game) syLvl(h *syHouse, id string) int {
	l := h.Upg[id]
	if l > 0 && g.Turn < h.Build[id] {
		l--
	}
	return l
}

func (g *Game) syNightly() int {
	if g.Cur == nil {
		return 8
	}
	return 6 + 4*g.W.cityTier[g.Cur.Burg]
}

func (g *Game) syIsInnkeeper(a *Actor) bool {
	return a.Role == "innkeeper" || a.Role == "tavern" || a.Role == "lw_innkeeper"
}

func syDays(m int) int { return maxi(0, (m+DayLen-1)/DayLen) }

// syNotoriety: what thieves and rivals can smell on you, 0..100.
func (g *Game) syNotoriety() int {
	n := htHeat(g) / 2
	htSafe(func() {
		u := g.uwSt()
		for _, b := range u.Bounty {
			n += b / 40
		}
		n += u.Stolen/2 + u.Jailed*2
	})
	return clampi(n, 0, 100)
}

func (g *Game) syBurglaryPct(h *syHouse) int {
	p := 4 + g.syNotoriety()/4 - 5*g.syLvl(h, "lock")
	if !h.Landlord {
		p += 3 // underground and out of sight, but out of the law's reach too
	}
	return clampi(p, 1, 60)
}

func (g *Game) syTipoffPct() int {
	n := 0
	if htHeat(g) >= 50 {
		n += 5
	}
	htSafe(func() {
		for _, b := range g.uwSt().Bounty {
			if b >= 50 {
				n += 3
				break
			}
		}
	})
	return n
}

func (g *Game) syWeekly(h *syHouse) int {
	n := h.Rate
	for _, d := range syUpgrades {
		n += d.Upkeep * h.Upg[d.ID]
	}
	return n
}

// ---------------------------------------------------------------- talk: innkeepers

func syTalk(g *Game, a *Actor) []Option {
	if g.Cur == nil || g.Cur.Kind != "town" || !g.syIsInnkeeper(a) {
		return nil
	}
	var out []Option
	r := g.syRoomHere()
	switch {
	case r != nil && r.Held:
		back := g.syArrears(r)
		out = append(out, Option{fmt.Sprintf("{{Y|Collect my things.}} {{K|(back rent %s)}}", g.cuFormat(back)), func() { g.syRedeem(r, a) }})
	case r != nil:
		out = append(out, Option{"{{G|Extend my stay.}}", func() { g.syRentMenu(a, r) }})
	default:
		out = append(out, Option{fmt.Sprintf("{{G|Rent a room.}} {{K|(%s a night)}}", g.cuFormat(g.syNightly())), func() { g.syRentMenu(a, nil) }})
	}
	return out
}

func (g *Game) syArrears(r *syRoom) int {
	return maxi(1, syDays(g.Turn-r.Paid)) * r.Rate
}

func (g *Game) syRentMenu(a *Actor, r *syRoom) {
	rate := g.syNightly()
	if r != nil {
		rate = r.Rate
	}
	var opts []Option
	for _, n := range syStayOptions {
		n := n
		cost := rate * n
		if n >= 7 {
			cost = cost * 9 / 10
		}
		opts = append(opts, Option{fmt.Sprintf("%d night%s: %s", n, map[bool]string{true: "s", false: ""}[n > 1], g.cuFormat(cost)), func() {
			if !g.cuSpend(cost) {
				g.Msg("{{y|You cannot pay for that many nights.}}")
				g.talkMenu("\"Come back when you can.\"")
				return
			}
			g.syRentDone(a, n, rate)
		}})
	}
	opts = append(opts, Option{"Never mind.", func() { g.talkMenu("\"Another time.\"") }})
	line := syInnLines[int(HashStr(fmt.Sprint(g.Seed, ":sy:inn:", a.ID)))%len(syInnLines)]
	g.uwOpenMenu("A room", line, opts, false)
	g.Talk = a
}

func (g *Game) syRentDone(a *Actor, n, rate int) {
	st := g.sy()
	r := g.syRoomHere()
	if r == nil {
		r = &syRoom{Burg: g.Cur.Burg, Zone: g.Cur.Key, Town: g.jbBurgName(g.Cur.Burg), Rate: rate}
		st.Rooms = append(st.Rooms, r)
		g.syLog("Rented a room in " + r.Town)
	}
	if r.Paid < g.Turn {
		r.Paid = g.Turn
	}
	r.Paid += n * DayLen
	g.Msg(fmt.Sprintf("{{G|Your room at the inn is paid for until day %d.}}", DayOf(r.Paid)+1))
	g.talkMenu("\"Sleep well. The chest is behind the bed.\"")
}

func (g *Game) syRedeem(r *syRoom, a *Actor) {
	cost := g.syArrears(r)
	if !g.cuSpend(cost) {
		g.talkMenu("\"Back rent first. I have bills too.\"")
		return
	}
	r.Held = false
	r.Paid = g.Turn + DayLen
	g.Msg("{{G|You pay the back rent and the innkeeper hands your chest key back.}}")
	g.talkMenu("\"One night is on the house.\"")
}

// ---------------------------------------------------------------- the day

func syTurn(g *Game) {
	if g == nil || g.Cur == nil || g.P == nil {
		return
	}
	st := g.sy()
	d := DayOf(g.Turn)
	if st.Day == 0 || d < st.Day {
		st.Day = d
	}
	for n := 0; st.Day < d && n < 60; n++ {
		st.Day++
		g.syDay(st.Day)
	}
	st.Day = d
}

func (g *Game) syDay(day int) {
	st := g.sy()
	// rooms: expiry, hold, sale
	for _, r := range st.Rooms {
		if r.Gone {
			continue
		}
		if !r.Held && g.Turn > r.Paid && len(r.Stash) > 0 {
			r.Held, r.HoldTo = true, r.Paid+syHoldDays*DayLen
			g.Msg("{{Y|Your rent at the inn in " + r.Town + " has run out. The innkeeper will hold your chest until day " + fmt.Sprint(DayOf(r.HoldTo)+1) + ".}}")
			g.syLog("Rent ran out in " + r.Town)
		}
		if r.Held && g.Turn > r.HoldTo {
			r.Stash, r.Held, r.Gone = nil, false, true
			g.syLog("The innkeeper in " + r.Town + " sold your held goods")
			lvSt(g).addNews(lvDay(g), r.Burg, "%s", syNewsSold)
			g.Msg("{{R|The innkeeper in " + r.Town + " has sold the goods you left behind.}}")
		}
		if !r.Held && len(r.Stash) == 0 && g.Turn > r.Paid+30*DayLen {
			r.Gone = true
		}
	}
	for _, h := range st.Houses {
		if h.Seized {
			continue
		}
		if h.Landlord && day%7 == 0 {
			h.RentDue += g.syWeekly(h)
			if h.RentDue > 3*g.syWeekly(h) {
				h.Seized = true
				g.Msg("{{R|The landowner has barred your safehouse " + h.Name + " over unpaid rent.}}")
				g.syLog("Safehouse barred over rent: " + h.Name)
				continue
			}
		}
		here := g.Cur != nil && g.Cur.Key == h.Key
		if here {
			continue
		}
		r := NewRng(fmt.Sprint(g.Seed, ":sy:day:", h.Key, ":", day))
		if day%7 == 3 && r.Int(1, 100) <= g.syBurglaryPct(h) {
			g.syBurgle(h, r)
		}
		if p := g.syTipoffPct(); p > 0 && r.Int(1, 100) <= p && g.Turn >= h.Watched {
			h.Watched = g.Turn + 3*DayLen
			h.Raids++
			g.Msg("{{R|" + fmt.Sprintf(Pick(r, syRaidLines), "safehouse "+h.Name) + "}}")
			g.syLog("Tipped off: the watch is watching " + h.Name)
			if b := g.W.burgByID[g.Cur.Burg]; b != nil {
				lvSt(g).addNews(lvDay(g), b.ID, "%s", syNewsRaid)
			}
		}
	}
}

func (g *Game) syBurgle(h *syHouse, r *Rng) {
	hs := g.homeSt()
	h.Burgled++
	if len(hs.Stash) == 0 {
		g.syLog("Thieves found nothing in " + h.Name)
		return
	}
	n := mini(2, len(hs.Stash))
	var lost []string
	for i := 0; i < n && len(hs.Stash) > 0; i++ {
		best := 0
		for k, it := range hs.Stash {
			if it.Value*maxi(it.Count, 1) > hs.Stash[best].Value*maxi(hs.Stash[best].Count, 1) && r.Chance(0.7) {
				best = k
			}
		}
		lost = append(lost, hs.Stash[best].Name)
		hs.Stash = append(hs.Stash[:best], hs.Stash[best+1:]...)
	}
	hid := ""
	if len(h.Hidden) > 0 {
		hid = " They never found the hidden stash."
	}
	g.Msg("{{R|" + fmt.Sprintf(Pick(r, syBurgLines), "safehouse "+h.Name) + " Missing: " + strings.Join(lost, ", ") + ".}}" + hid)
	g.syLog("Burgled: " + h.Name + " lost " + strings.Join(lost, ", "))
}

func syEnter(g *Game, zn *Zone) {
	if zn == nil {
		return
	}
	for _, r := range g.sy().Rooms {
		if !r.Gone && r.Zone == zn.Key && r.Held {
			g.Msg("{{Y|" + fmt.Sprintf(syHeldLines[0], DayOf(r.HoldTo)+1) + "}}")
		}
	}
	if h := g.syHouseHere(); h != nil && h.Seized {
		g.Msg("{{R|Your safehouse here is barred over unpaid rent. See the Lodgings screen.}}")
	}
}

func sySide(g *Game) []string {
	if r := g.syRoomHere(); r != nil && !r.Held && g.Turn <= r.Paid {
		return []string{fmt.Sprintf("{{G|Room paid %dd}}", syDays(r.Paid-g.Turn))}
	}
	return nil
}

// ---------------------------------------------------------------- actions

func (g *Game) sySleep(boost bool) {
	p := g.P
	g.ijOnHomeSleep()
	p.HP = p.MaxHP
	g.Hunger, g.Thirst = 0, 0
	delete(p.Status, "bleeding")
	delete(p.Status, "poisoned")
	for _, bp := range p.Body {
		bp.Severed = false
	}
	g.Turn += 480
	g.sy().Slept++
	g.Msg("{{G|You sleep eight hours behind a bolted door and wake rested, fed and whole.}}")
	if boost {
		htCool(g, 1)
	}
	g.recompute()
}

// syLieLow passes days hidden away: heat and the local bounty decay.
func (g *Game) syLieLow(days int, room *syRoom, h *syHouse) {
	if days < 1 {
		return
	}
	state := g.uwStateHere()
	start := htHeat(g)
	startB := g.uwBounty(state)
	spent := 0
	for d := 0; d < days; d++ {
		cool := 3
		hunger := 192
		watched := false
		if h != nil {
			cool = 5 + g.syLvl(h, "bed")*2
			if g.syLvl(h, "kitchen") > 0 {
				hunger = 60
			}
			watched = g.Turn < h.Watched
			if watched {
				cool /= 2
				if g.R.Chance(0.25) {
					if g.syLvl(h, "tunnel") > 0 {
						g.Msg("{{Y|Boots on the stairs. You are already down the bolt-hole and gone before the door breaks.}}")
					} else {
						g.Msg("{{R|The watch breaks in and tears the place apart. You are lucky to get out.}}")
						htSafe(func() { g.uwAddBounty(state, 15, " (found at a watched safehouse)") })
						hs := g.homeSt()
						if len(hs.Stash) > 0 {
							k := g.R.Int(0, len(hs.Stash)-1)
							g.Msg("{{R|Confiscated from your stash: " + hs.Stash[k].Name + ".}}")
							hs.Stash = append(hs.Stash[:k], hs.Stash[k+1:]...)
						}
						h.Raids++
						g.Turn += 360
						spent++
						break
					}
				}
			}
		} else {
			hunger = 0 // meals come with the room
		}
		htCool(g, cool)
		if b := g.uwBounty(state); b > 0 && !watched {
			dec := maxi(2, b/20)
			if b-dec <= 0 {
				g.uwClearBounty(state)
			} else {
				g.uwSt().Bounty[state] = b - dec
			}
		}
		g.Hunger = mini(g.Hunger+hunger, 2000)
		g.Thirst = mini(g.Thirst+hunger, 1500)
		g.Turn += DayLen
		g.P.HP = g.P.MaxHP
		spent++
	}
	g.ijOnHomeSleep()
	g.sy().Low += spent
	g.Msg(fmt.Sprintf("{{G|You lie low for %d day(s). Heat %d to %d.}}", spent, start, htHeat(g)))
	if b := g.uwBounty(state); startB > 0 {
		g.Msg(fmt.Sprintf("Bounty here: %d to %d.", startB, b))
	}
	g.syLog(fmt.Sprintf("Lay low %d day(s): heat %d to %d", spent, start, htHeat(g)))
	g.recompute()
}

func (g *Game) syEstablish() {
	if g.Cur == nil {
		return
	}
	st := g.sy()
	if g.syHouseHere() != nil {
		g.Msg("This is already your safehouse.")
		return
	}
	zn := g.Cur
	if g.homeHere() == nil {
		k := zn.Kind
		ok := k == "ruin" || k == "camp" || k == "crypt" || k == "dungeon" || k == "underdark" || k == "lair" || zn.Z > 0
		if k == "town" || !ok {
			g.Msg("{{y|A safehouse needs an abandoned building or a cleared hideout: ruins, camps, crypts and dungeon levels.}}")
			return
		}
		if len(g.homeSt().Homes) >= maxHomes {
			g.Msg("{{y|You already keep as many homes as you can look after.}}")
			return
		}
		if g.hostilesLeft() {
			g.Msg("{{R|Clear out the hostile creatures here first.}}")
			return
		}
		if !g.cuSpend(60) {
			g.Msg("{{y|Boarding the place up takes about " + g.cuFormat(60) + ".}}")
			return
		}
		g.claimHome("Safehouse at "+zn.Name, "hideout")
	}
	name := zn.Name
	if h := g.homeHere(); h != nil {
		name = h.Name
	}
	landlord := zn.Z <= 0 && zn.Kind != "dungeon" && zn.Kind != "underdark" && zn.Kind != "crypt"
	h := &syHouse{Key: zn.Key, Name: name, Since: g.Turn, Upg: map[string]int{}, Build: map[string]int{}, Landlord: landlord, Rate: 0}
	if landlord {
		h.Rate = 12 + 4*g.W.cityTier[maxi(zn.Burg, 0)]
	}
	st.Houses = append(st.Houses, h)
	g.syLog("Took over " + name)
	g.Msg("{{G|" + name + " is now your safehouse.}}")
}

func (g *Game) syBuy(h *syHouse, d syUpgDef) {
	lvl := h.Upg[d.ID]
	if lvl >= d.Max {
		g.Msg("That upgrade is finished.")
		return
	}
	if lvl > 0 && g.Turn < h.Build[d.ID] {
		g.Msg("{{y|The last stage is still being built.}}")
		return
	}
	cost := d.Cost[lvl]
	if !g.cuSpend(cost) {
		g.Msg("{{y|You cannot pay for it: " + g.cuFormat(cost) + ".}}")
		return
	}
	h.Upg[d.ID] = lvl + 1
	h.Build[d.ID] = g.Turn + d.Days[lvl]*DayLen
	g.Msg(fmt.Sprintf("{{G|%s begun. It will be ready in %d day(s).}}", d.Name, d.Days[lvl]))
	g.syLog("Ordered " + d.Name + " at " + h.Name)
}

func (g *Game) syPayRent(h *syHouse) {
	if h.RentDue <= 0 {
		g.Msg("Nothing owed.")
		return
	}
	pay := h.RentDue
	if h.Seized {
		pay += g.syWeekly(h)
	}
	if !g.cuSpend(pay) {
		g.Msg("{{y|You owe " + g.cuFormat(pay) + ".}}")
		return
	}
	h.RentDue, h.Seized = 0, false
	g.Msg("{{G|The landowner takes the rent and unbars the door.}}")
}

func (g *Game) syBoltHole(h *syHouse) {
	zn := g.Cur
	r := NewRng(fmt.Sprint(g.Seed, ":sy:bolt:", zn.Key, ":", g.Turn/60))
	x, y, ok := zn.FreeSpot(r, func(x, y int) bool { return abs(x-g.P.X)+abs(y-g.P.Y) > 24 })
	if !ok {
		g.Msg("The tunnel is caved in.")
		return
	}
	g.P.X, g.P.Y = x, y
	htCool(g, 4)
	for _, a := range zn.Actors {
		if a.Angry && abs(a.X-x) > 6 && abs(a.Y-y) > 6 && uwIsGuardOf(a, g.uwStateHere()) {
			a.Angry = false
		}
	}
	g.Msg("{{Y|You slip down the bolt-hole and come out far from the door. Nobody saw where you went.}}")
	g.recompute()
	g.Mode = "main"
}

// ---------------------------------------------------------------- screen

var syTabs = []string{"Rooms", "Safehouses", "Risks"}

func (g *Game) syOpen(tab int) {
	st := g.sy()
	st.tab, st.sel, st.conf = tab, 0, ""
	g.Mode = "sy"
}

func (g *Game) syBuildRows() []htRow {
	st := g.sy()
	var rows []htRow
	switch st.tab {
	case 0:
		for i, r := range st.Rooms {
			if r.Gone {
				continue
			}
			i, r := i, r
			status := fmt.Sprintf("paid until day %d", DayOf(r.Paid)+1)
			if g.Turn > r.Paid {
				status = "{{R|rent ran out}}"
			}
			if r.Held {
				status = fmt.Sprintf("{{R|held by the innkeeper, sold on day %d}}", DayOf(r.HoldTo)+1)
			}
			rows = append(rows, htRow{Text: fmt.Sprintf("{{W|Room in %s}}: %s. Chest: %d item(s).", r.Town, status, len(r.Stash)), Dim: true})
			here := g.Cur != nil && g.Cur.Key == r.Zone
			valid := !r.Held && g.Turn <= r.Paid
			if here && valid {
				rows = append(rows,
					htRow{Text: "   Open the chest", Do: func() { st.cur, st.stTab, st.stSel = fmt.Sprint("room:", i), 0, [2]int{}; g.Mode = "syst" }},
					htRow{Text: "   Sleep (heals fully)", Do: func() { g.sySleep(false) }},
					htRow{Text: "   Lie low for one day", Do: func() { g.syLieLow(1, r, nil) }},
					htRow{Text: "   Lie low for three days", Do: func() { g.syLieLow(3, r, nil) }},
					htRow{Text: "   Lie low for a week", Do: func() { g.syLieLow(7, r, nil) }})
			} else if here && r.Held {
				rows = append(rows, htRow{Text: "   Talk to the innkeeper to collect your things.", Dim: true})
			} else if !here {
				rows = append(rows, htRow{Text: "   {{K|(be at the inn in " + r.Town + ")}}", Dim: true})
			}
		}
		if len(rows) == 0 {
			rows = append(rows, htRow{Text: "No rooms rented. Talk to an innkeeper: \"Rent a room\".", Dim: true})
		}
	case 1:
		zn := g.Cur
		hh := g.syHouseHere()
		for i, h := range st.Houses {
			i, h := i, h
			rent := ""
			if h.Landlord {
				rent = fmt.Sprintf(" Rent %s a week, owed %s.", g.cuFormat(g.syWeekly(h)), g.cuFormat(h.RentDue))
			}
			state := "ok"
			if h.Seized {
				state = "{{R|barred}}"
			}
			if g.Turn < h.Watched {
				state += " {{R|watched}}"
			}
			rows = append(rows, htRow{Text: fmt.Sprintf("{{W|%s}}: %s.%s Hidden stash: %d item(s).", h.Name, state, rent, len(h.Hidden)), Dim: true})
			if hh != h {
				continue
			}
			if h.RentDue > 0 || h.Seized {
				rows = append(rows, htRow{Text: "   Pay the rent owed", Do: func() { g.syPayRent(h) }})
			}
			if h.Seized {
				continue
			}
			rows = append(rows,
				htRow{Text: "   Open my stash (home.go)", Do: func() { g.openStash() }},
				htRow{Text: "   Sleep (heals fully)", Do: func() { g.sySleep(g.syLvl(h, "bed") > 0) }},
				htRow{Text: "   Lie low for one day", Do: func() { g.syLieLow(1, nil, h) }},
				htRow{Text: "   Lie low for three days", Do: func() { g.syLieLow(3, nil, h) }},
				htRow{Text: "   Lie low for a week", Do: func() { g.syLieLow(7, nil, h) }})
			if g.syLvl(h, "hidden") > 0 {
				rows = append(rows, htRow{Text: "   Open the hidden stash", Do: func() { st.cur, st.stTab, st.stSel = fmt.Sprint("hid:", i), 0, [2]int{}; g.Mode = "syst" }})
			}
			if g.syLvl(h, "tunnel") > 0 {
				rows = append(rows, htRow{Text: "   Slip out through the bolt-hole", Do: func() { g.syBoltHole(h) }})
			}
			for _, d := range syUpgrades {
				d := d
				lvl := h.Upg[d.ID]
				switch {
				case lvl >= d.Max && g.Turn >= h.Build[d.ID]:
					rows = append(rows, htRow{Text: fmt.Sprintf("   %s: done. %s", d.Name, d.Desc), Dim: true})
				case lvl > 0 && g.Turn < h.Build[d.ID]:
					rows = append(rows, htRow{Text: fmt.Sprintf("   %s: building, %d day(s) left.", d.Name, syDays(h.Build[d.ID]-g.Turn)), Dim: true})
				default:
					rows = append(rows, htRow{Text: fmt.Sprintf("   Buy %s (level %d, %s, %d days). %s", d.Name, lvl+1, g.cuFormat(d.Cost[lvl]), d.Days[lvl], d.Desc), Do: func() { g.syBuy(h, d) }})
				}
			}
		}
		if hh == nil && zn != nil {
			rows = append(rows, htRow{Text: "{{G|Take over this place as a safehouse}} {{K|(an abandoned building or cleared hideout; about " + g.cuFormat(60) + ")}}", Do: func() { g.syEstablish() }})
		}
	default:
		rows = append(rows, htRow{Text: "{{K|Nothing to do here: this is the risk sheet.}}", Dim: true})
	}
	return rows
}

func syKeys(g *Game, k string) {
	st := g.sy()
	switch k {
	case "Escape", "q":
		g.Mode = "main"
		return
	case "Tab", "ArrowRight":
		st.tab = (st.tab + 1) % len(syTabs)
		st.sel, st.conf = 0, ""
		return
	case "ArrowLeft":
		st.tab = (st.tab + len(syTabs) - 1) % len(syTabs)
		st.sel, st.conf = 0, ""
		return
	}
	rows := g.syBuildRows()
	switch k {
	case "ArrowUp", "N8", "k":
		st.sel = maxi(0, st.sel-1)
	case "ArrowDown", "N2", "j":
		st.sel = mini(maxi(0, len(rows)-1), st.sel+1)
	case "Enter", " ":
		if st.sel < len(rows) && rows[st.sel].Do != nil {
			rows[st.sel].Do()
		}
	}
}

func (g *Game) syRisks() []string {
	st := g.sy()
	out := []string{
		fmt.Sprintf("Heat: %d. Notoriety (what thieves and rivals can smell on you): %d.", htHeat(g), g.syNotoriety()),
		fmt.Sprintf("Chance a rival tips off the guards each day: %d%%.", g.syTipoffPct()),
	}
	state := g.uwStateHere()
	htSafe(func() {
		if b := g.uwBounty(state); b > 0 {
			out = append(out, fmt.Sprintf("Bounty with %s: %d. Lying low cuts it by a twentieth a day, but not at a watched house.", g.uwRealm(state), b))
		}
	})
	for _, h := range st.Houses {
		line := fmt.Sprintf("%s: burglary chance %d%% a week.", h.Name, g.syBurglaryPct(h))
		if h.Landlord {
			line += fmt.Sprintf(" Weekly cost %s; owed %s.", g.cuFormat(g.syWeekly(h)), g.cuFormat(h.RentDue))
		}
		if g.Turn < h.Watched {
			line += fmt.Sprintf(" Watched for %d more day(s).", syDays(h.Watched-g.Turn))
		}
		if g.syLvl(h, "hidden") > 0 {
			line += " The hidden stash is safe from thieves."
		}
		out = append(out, line)
	}
	for _, r := range st.Rooms {
		if r.Gone {
			continue
		}
		switch {
		case r.Held:
			out = append(out, fmt.Sprintf("Room in %s: held; back rent %s buys it back until day %d.", r.Town, g.cuFormat(g.syArrears(r)), DayOf(r.HoldTo)+1))
		case g.Turn > r.Paid && len(r.Stash) > 0:
			out = append(out, "Room in "+r.Town+": rent has run out and the chest is about to be held.")
		default:
			out = append(out, fmt.Sprintf("Room in %s: paid for %d more day(s).", r.Town, syDays(r.Paid-g.Turn)))
		}
	}
	out = append(out, "The visible stash is shared by all your homes; the hidden stash and room chests are not.")
	return append(out, st.Log...)
}

func (g *Game) syDraw() {
	st := g.sy()
	sc := g.sc
	x, y, w, h := g.panel("Lodgings and safehouses")
	bottom := 2 + 1 + h - 1
	tabs := ""
	for i, t := range syTabs {
		c := "K"
		if i == st.tab {
			c = "W"
		}
		tabs += fmt.Sprintf("{{%s|[ %s ]}}  ", c, t)
	}
	sc.TextB(x, y, tabs, CGray, CBgDark, x+w)
	y++
	sc.TextB(x, y, "{{K|Up/Down choose  Enter act  Tab next  Esc close.}}", CGray, CBgDark, x+w)
	y += 2
	if st.tab == 2 {
		for _, l := range g.syRisks() {
			for _, w2 := range Wrap(l, w) {
				if y < bottom-1 {
					sc.TextB(x, y, w2, CGray, CBgDark, x+w)
					y++
				}
			}
		}
		return
	}
	rows := g.syBuildRows()
	if st.sel >= len(rows) {
		st.sel = maxi(0, len(rows)-1)
	}
	for i, r := range rows {
		mark := "  "
		if i == st.sel {
			mark = "> "
		}
		txt := r.Text
		if r.Dim {
			txt = "{{K|" + stripMarkup(r.Text) + "}}"
		}
		for k, l := range Wrap(mark+txt, w) {
			if k > 0 {
				l = "  " + l
			}
			if y < bottom-1 {
				sc.TextB(x, y, l, CGray, CBgDark, x+w)
				y++
			}
		}
	}
}

// ---------------------------------------------------------------- chests

func (g *Game) syChest() *[]*Item {
	st := g.sy()
	var kind string
	var i int
	if _, err := fmt.Sscanf(strings.Replace(st.cur, ":", " ", 1), "%s %d", &kind, &i); err != nil {
		return nil
	}
	switch kind {
	case "room":
		if i >= 0 && i < len(st.Rooms) {
			return &st.Rooms[i].Stash
		}
	case "hid":
		if i >= 0 && i < len(st.Houses) {
			return &st.Houses[i].Hidden
		}
	}
	return nil
}

func syStashKeys(g *Game, k string) {
	st := g.sy()
	ch := g.syChest()
	if ch == nil {
		g.Mode = "sy"
		return
	}
	list := &g.P.Inv
	if st.stTab == 1 {
		list = ch
	}
	sel := &st.stSel[st.stTab]
	switch k {
	case "Escape", "q":
		g.Mode = "sy"
	case "Tab", "ArrowLeft", "ArrowRight":
		st.stTab ^= 1
	case "ArrowUp":
		*sel = maxi(0, *sel-1)
	case "ArrowDown":
		*sel = mini(maxi(0, len(*list)-1), *sel+1)
	case "Enter", " ":
		if *sel >= len(*list) {
			return
		}
		it := (*list)[*sel]
		if st.stTab == 0 {
			if it.Quest != 0 {
				g.Msg("{{y|You cannot leave that behind.}}")
				return
			}
			g.removeInv(it)
			*ch = append(*ch, it)
		} else {
			*ch = append((*ch)[:*sel], (*ch)[*sel+1:]...)
			g.give(it)
		}
		*sel = mini(*sel, maxi(0, len(*list)-1))
	}
}

func (g *Game) syStashDraw() {
	st := g.sy()
	ch := g.syChest()
	sc := g.sc
	x, y, w, h := g.panel("Chest")
	if ch == nil {
		return
	}
	sc.TextB(x, y, "{{K|Tab switch side  Enter move  Esc back}}", CGray, CBgDark, x+w)
	y += 2
	half := w / 2
	for side := 0; side < 2; side++ {
		list := g.P.Inv
		title := "Your pack"
		if side == 1 {
			list = *ch
			title = "The chest"
		}
		c := "K"
		if st.stTab == side {
			c = "W"
		}
		sc.TextB(x+side*half, y, fmt.Sprintf("{{%s|%s (%d)}}", c, title, len(list)), CGray, CBgDark, x+side*half+half-2)
		for i, it := range list {
			yy := y + 1 + i - maxi(0, st.stSel[side]-(h-8))
			if yy < y+1 || yy >= y+h-6 {
				continue
			}
			mark := "  "
			if st.stTab == side && i == st.stSel[side] {
				mark = "> "
			}
			sc.TextB(x+side*half, yy, mark+it.Display(), CGray, CBgDark, x+side*half+half-2)
		}
	}
}
