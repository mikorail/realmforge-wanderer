package engine

import (
	"fmt"
	"sort"
	"strings"
)

// Player home / base: buy a house in a city, or claim a cleared ruin/camp/dungeon level as a
// hideout. Each home has a bed, a stash chest, a safe place for companions and a fast-travel
// target. Everything is stored in g.ext("home").

const maxHomes = 3

type homeRec struct {
	Key  string `json:"key"`
	Name string `json:"name"`
	X    int    `json:"x"` // where you arrive
	Y    int    `json:"y"`
	SX   int    `json:"sx"` // stash tile
	SY   int    `json:"sy"`
	Kind string `json:"kind"` // house hideout
}

type homeState struct {
	Homes []homeRec `json:"homes"`
	Stash []*Item   `json:"stash"`
}

func (g *Game) homeSt() *homeState {
	return g.ext("home", func() any { return &homeState{} }).(*homeState)
}

func (g *Game) homeHere() *homeRec {
	if g.Cur == nil {
		return nil
	}
	hs := g.homeSt()
	for i := range hs.Homes {
		if hs.Homes[i].Key == g.Cur.Key {
			return &hs.Homes[i]
		}
	}
	return nil
}

func init() {
	talkHooks = append(talkHooks, homeTalk)
	mainKeys["H"] = func(g *Game) { g.homeKey() }
	mainKeys["F3"] = func(g *Game) { g.returnHomeMenu() }
	modeKeys["stash"] = func(g *Game, k string) { g.keyStash(k) }
	modeDraws["stash"] = func(g *Game) { g.drawStash() }
	modeKeys["homepick"] = func(g *Game, k string) { g.keyMenu(k, func() { g.Mode = "main" }) }
	modeDraws["homepick"] = func(g *Game) { g.drawHomePick() }
	enterHooks = append(enterHooks, homeEnter)
	sideHooks = append(sideHooks, func(g *Game) []string {
		if h := g.homeHere(); h != nil {
			return []string{"{{G|Home: " + h.Name + "}}"}
		}
		return nil
	})
}

// ---- claiming ----

func (g *Game) housePrice(a *Actor) int {
	tier := 0
	if g.Cur != nil && g.W != nil {
		tier = g.W.cityTier[g.Cur.Burg]
	}
	base := 300 + 300*tier
	pct := clampi(g.Rep[a.Faction]/20, -50, 30)
	return base * (100 - pct) / 100
}

func homeTalk(g *Game, a *Actor) []Option {
	if a.Role != "elder" && a.Role != "notable" {
		return nil
	}
	if g.Cur == nil || g.Cur.Kind != "town" {
		return nil
	}
	if g.homeHere() != nil || len(g.homeSt().Homes) >= maxHomes {
		return nil
	}
	price := g.housePrice(a)
	return []Option{{fmt.Sprintf("{{W|Buy a house here.}} {{K|(%d %s)}}", price, g.Sk.Currency), func() {
		if g.P.Gold < price {
			g.talkMenu("\"You can't afford a house here.\"")
			return
		}
		g.P.Gold -= price
		name := "House in " + g.Cur.Name
		g.claimHome(name, "house")
		g.talkMenu("\"The deed is yours. The door is on the south wall; the bed and the chest are inside.\"")
	}}}
}

func (g *Game) hostilesLeft() bool {
	for _, a := range g.Cur.Actors {
		if a.HP > 0 && a != g.P && g.Hostile(a) {
			return true
		}
	}
	return false
}

func (g *Game) homeKey() {
	if g.Cur == nil || g.Mode != "main" {
		return
	}
	if h := g.homeHere(); h != nil {
		g.openStash()
		return
	}
	k := g.Cur.Kind
	ok := k == "ruin" || k == "camp" || k == "crypt" || k == "dungeon" || k == "underdark" || k == "lair" || g.Cur.Z > 0
	if k == "town" || !ok {
		g.Msg("You can't make a hideout here. (Ruins, camps and cleared dungeon levels will do.)")
		return
	}
	if len(g.homeSt().Homes) >= maxHomes {
		g.Msg("You already keep the most homes you can look after.")
		return
	}
	if g.hostilesLeft() {
		g.Msg("{{R|Clear out the hostile creatures here first.}}")
		return
	}
	g.claimHome("Hideout at "+g.Cur.Name, "hideout")
	g.Msg("{{G|You claim this place as a hideout. Press H to open your stash.}}")
}

// claimHome builds the house in the current zone and records it.
func (g *Game) claimHome(name, kind string) {
	zn := g.Cur
	h := homeRec{Key: zn.Key, Name: name, Kind: kind}
	if x0, y0, ok := g.findHouseSite(zn, kind == "hideout"); ok {
		for y := y0; y < y0+5; y++ {
			for x := x0; x < x0+7; x++ {
				t := TWall
				if x > x0 && x < x0+6 && y > y0 && y < y0+4 {
					t = TCarpet
				}
				zn.Set(x, y, t)
				delete(zn.Notes, y*MW+x)
			}
		}
		zn.Set(x0+3, y0+4, TDoor)
		zn.Set(x0+1, y0+1, TBed)
		zn.Set(x0+5, y0+1, TShelf)
		zn.Notes[(y0+1)*MW+x0+1] = "Your bed. Open your stash (H) to sleep."
		h.SX, h.SY = x0+5, y0+1
		h.X, h.Y = x0+3, y0+2
		zn.Notes[h.SY*MW+h.SX] = "Your stash"
	} else {
		// no room for a house: furnish the nearest free cells around the player
		var free [][2]int
		for _, d := range dirs8 {
			x, y := g.P.X+d[0], g.P.Y+d[1]
			t := zn.At(x, y)
			if zn.In(x, y) && t.D().Walk && t != TUp && t != TDown && zn.ActorAt(x, y) == nil {
				free = append(free, [2]int{x, y})
			}
		}
		if len(free) < 2 {
			free = append(free, [2]int{g.P.X, g.P.Y}, [2]int{g.P.X, g.P.Y})
		}
		b, s := free[0], free[1]
		zn.Set(b[0], b[1], TBed)
		zn.Set(s[0], s[1], TShelf)
		zn.Notes[b[1]*MW+b[0]] = "Your bed. Open your stash (H) to sleep."
		zn.Notes[s[1]*MW+s[0]] = "Your stash"
		h.SX, h.SY = s[0], s[1]
		h.X, h.Y = g.P.X, g.P.Y
	}
	hs := g.homeSt()
	hs.Homes = append(hs.Homes, h)
	g.Msg("{{G|" + name + " is yours.}}")
	g.recompute()
}

// findHouseSite finds a 7×5 plot of open ground (or, for hideouts, solid rock too) nearest the player.
func (g *Game) findHouseSite(zn *Zone, carve bool) (int, int, bool) {
	type cand struct{ x, y, d int }
	var cs []cand
	for y := 1; y+5 < MH; y++ {
		for x := 1; x+7 < MW; x++ {
			cs = append(cs, cand{x, y, abs(x+3-g.P.X) + abs(y+2-g.P.Y)})
		}
	}
	sort.SliceStable(cs, func(i, j int) bool { return cs[i].d < cs[j].d })
	okCell := func(x, y int) bool {
		if !zn.In(x, y) {
			return false
		}
		t := zn.At(x, y)
		if t == TUp || t == TDown || t == TUpdraft || t == TDoor || t == TDoorOpen || t == TLocked || zn.WaterAt(x, y) > 0 {
			return false
		}
		if _, n := zn.Notes[y*MW+x]; n {
			return false
		}
		if zn.ActorAt(x, y) != nil || len(zn.ItemsAt(x, y)) > 0 {
			return false
		}
		if t.D().Walk {
			return true
		}
		return carve && (t == TRock || t == TWall)
	}
	for _, c := range cs {
		good := true
		for y := c.y; y < c.y+5 && good; y++ {
			for x := c.x; x < c.x+7; x++ {
				if !okCell(x, y) {
					good = false
					break
				}
			}
		}
		if !good {
			continue
		}
		ox, oy := c.x+3, c.y+5
		if !zn.Walkable(ox, oy) || zn.WaterAt(ox, oy) > 0 {
			continue
		}
		if abs(g.P.X-ox) > 25 { // keep the house close
			continue
		}
		return c.x, c.y, true
	}
	return 0, 0, false
}

// ---- enter: companions waiting at home ----

func homeEnter(g *Game, zn *Zone) {
	if g.homeHere() == nil {
		return
	}
	for _, a := range zn.Actors {
		if a.Flags["stayhome"] && a.HP > 0 {
			g.Msg(fmt.Sprintf("{{G|%s is waiting for you at home.}}", a.Name))
		}
	}
}

// ---- stash screen ----

func (g *Game) openStash() {
	g.Mode = "stash"
	g.Tab, g.Sel, g.Sel2 = 0, 0, 0
}

func (g *Game) stashLists() (inv, st []*Item) {
	return g.P.Inv, g.homeSt().Stash
}

func (g *Game) keyStash(k string) {
	hs := g.homeSt()
	inv := &g.P.Inv
	if g.Tab == 1 {
		inv = &hs.Stash
	}
	switch k {
	case "Escape":
		g.Mode = "main"
	case "ArrowUp":
		g.Sel = maxi(0, g.Sel-1)
	case "ArrowDown":
		g.Sel = mini(maxi(0, len(*inv)-1), g.Sel+1)
	case "PageUp":
		g.Sel = maxi(0, g.Sel-10)
	case "PageDown":
		g.Sel = mini(maxi(0, len(*inv)-1), g.Sel+10)
	case "Tab", "ArrowLeft", "ArrowRight":
		g.Sel, g.Sel2 = g.Sel2, g.Sel
		g.Tab ^= 1
	case "Enter", " ":
		if g.Sel < len(*inv) {
			it := (*inv)[g.Sel]
			if g.Tab == 0 {
				g.removeInv(it)
				stashAdd(hs, it)
			} else {
				*inv = append((*inv)[:g.Sel], (*inv)[g.Sel+1:]...)
				g.give(it)
			}
			g.Sel = mini(g.Sel, maxi(0, len(*inv)-1))
		}
	case "s", "S":
		g.sleepHome()
	case "c", "C":
		g.companionsStay()
	case "v", "V":
		g.companionsRecall()
	}
}

func stashAdd(hs *homeState, it *Item) {
	if it.Stack() {
		for _, o := range hs.Stash {
			if o.Kind == it.Kind && o.Name == it.Name && o.Effect == it.Effect {
				o.Count += maxi(it.Count, 1)
				return
			}
		}
	}
	hs.Stash = append(hs.Stash, it)
}

func (g *Game) sleepHome() {
	p := g.P
	p.HP = p.MaxHP
	g.Hunger, g.Thirst = 0, 0
	delete(p.Status, "bleeding")
	delete(p.Status, "poisoned")
	for _, bp := range p.Body {
		bp.Severed = false
	}
	g.Turn += 480
	g.Msg("{{G|You sleep eight hours in your own bed and wake rested, fed and whole.}}")
}

func (g *Game) companionsStay() {
	h := g.homeHere()
	if h == nil {
		return
	}
	n := 0
	for _, a := range g.Cur.Actors {
		if a.AI == "follower" && a.HP > 0 {
			a.AI = "town"
			a.HomeX, a.HomeY = h.X, h.Y
			if a.Flags == nil {
				a.Flags = map[string]bool{}
			}
			a.Flags["stayhome"] = true
			n++
			g.Msg(fmt.Sprintf("%s settles in to wait at home.", a.Name))
		}
	}
	if n == 0 {
		g.Msg("No companion is with you.")
	}
}

func (g *Game) companionsRecall() {
	n := 0
	for _, a := range g.Cur.Actors {
		if a.Flags["stayhome"] && a.HP > 0 {
			a.AI = "follower"
			delete(a.Flags, "stayhome")
			n++
			g.Msg(fmt.Sprintf("{{G|%s joins you again.}}", a.Name))
		}
	}
	if n == 0 {
		g.Msg("No companion is waiting at home.")
	}
}

func (g *Game) drawStash() {
	h := g.homeHere()
	title := "Stash"
	if h != nil {
		title = h.Name
	}
	s := g.sc
	x, y, w, ph := g.panel(title)
	half := w / 2
	hs := g.homeSt()
	cols := [][]*Item{g.P.Inv, hs.Stash}
	heads := []string{"You carry", "Stash"}
	rows := ph - 5
	for c := 0; c < 2; c++ {
		cx := x + c*half
		s.TextB(cx, y, fmt.Sprintf("{{Y|%s}} {{K|(%d)}}", heads[c], len(cols[c])), CGray, CBgDark, cx+half-1)
		sel := 0
		if c == g.Tab {
			sel = g.Sel
		} else {
			sel = g.Sel2
		}
		top := maxi(0, sel-rows+1)
		for i := top; i < len(cols[c]) && i-top < rows; i++ {
			bg := CBgDark
			if i == sel && c == g.Tab {
				bg = CBgSel
			}
			s.Fill(cx, y+2+i-top, half-2, 1, bg)
			s.TextB(cx, y+2+i-top, cols[c][i].Display(), CGray, bg, cx+half-2)
		}
	}
	nf := 0
	nw := 0
	for _, a := range g.Cur.Actors {
		if a.AI == "follower" {
			nf++
		}
		if a.Flags["stayhome"] {
			nw++
		}
	}
	s.TextB(x, y+ph-3, "{{K|Tab switch  Enter move item  Esc leave}}", CGray, CBgDark, x+w)
	s.TextB(x, y+ph-2, fmt.Sprintf("{{W|s}} sleep in your bed (8 hours)   {{W|c}} companions stay home (%d with you)   {{W|v}} call them (%d waiting)", nf, nw), CGray, CBgDark, x+w)
}

// ---- fast travel ----

func (g *Game) homeCost(h homeRec) (parasangs, turns int, ok bool) {
	zn := g.Zones[h.Key]
	if zn == nil {
		return 0, 0, false
	}
	parasangs = maxi(abs(zn.WX-g.WX), abs(zn.WY-g.WY))
	turns = parasangs * 300
	return parasangs, turns, true
}

func (g *Game) returnHomeMenu() {
	if g.Mode != "main" || g.Cur == nil {
		return
	}
	hs := g.homeSt()
	if len(hs.Homes) == 0 {
		g.Msg("You have no home to return to.")
		return
	}
	for _, a := range g.Cur.Actors {
		if a.HP > 0 && a != g.P && g.Hostile(a) && g.visible(a.X, a.Y) {
			g.Msg("{{R|You can't slip away with enemies in sight.}}")
			return
		}
	}
	g.Opts = nil
	for _, h := range hs.Homes {
		h := h
		if h.Key == g.CurKey {
			continue
		}
		ps, turns, ok := g.homeCost(h)
		if !ok {
			continue
		}
		g.Opts = append(g.Opts, Option{fmt.Sprintf("%s {{K|(%d parasangs, %d turns, food and water)}}", h.Name, ps, turns), func() { g.travelHome(h) }})
	}
	if len(g.Opts) == 0 {
		g.Msg("You are already home.")
		return
	}
	g.Popup = []string{"Where to?"}
	g.Sel = 0
	g.Mode = "homepick"
}

func (g *Game) travelHome(h homeRec) {
	_, turns, ok := g.homeCost(h)
	zn := g.Zones[h.Key]
	if !ok || zn == nil {
		g.Mode = "main"
		return
	}
	g.Turn += turns
	g.Hunger = mini(g.Hunger+turns/2, 2300)
	g.Thirst = mini(g.Thirst+turns/2, 1700)
	g.Mode = "main"
	g.enterZone(zn, h.X, h.Y, true)
	g.Msg(fmt.Sprintf("You make your way home. (%d turns)", turns))
	g.recompute()
}

func (g *Game) drawHomePick() {
	s := g.sc
	h := len(g.Popup) + len(g.Opts) + 4
	y0 := maxi(1, (MH-h)/2)
	s.Box(2, y0, SW-4, h, CDarkCyan, "Return home")
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
	_ = strings.TrimSpace
}
