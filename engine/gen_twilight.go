package engine

import (
	"fmt"
	"math"
	"strings"
)

// Genre signature: twilight, the decay clock (prefix gc). Every realm (state id, 0 = the free lands) has a
// decay value 0..100: a hashed starting point, rising each day and faster with neglect (days since anyone
// tended the realm), falling with ACTS: relight beacons (placed in towns, burn fuel), mend ruin wards (stone
// tiles in ruins: scrap, craft or lore), return relics to a priest, lay the dead to rest (undead slain).
// High decay: more ghosts and the dead in the wilds (zone hook, side rng), the dead rise at night,
// plague in towns, crops fail (gcDecayPriceMod in price()), lanterns go dark in towns, the sidebar and
// world map (gcOverlay) show it. Only in genre twilight; zone hook uses side rng seed:gc:tw:zoneKey.
// State: g.ext("gct"), planet-local.

type gcBeacon struct {
	State int `json:"s"`
	X     int `json:"x"`
	Y     int `json:"y"`
	Until int `json:"u"` // burning until this game minute
}

type gcWard struct {
	State int  `json:"s"`
	X     int  `json:"x"`
	Y     int  `json:"y"`
	Tier  int  `json:"t"`
	Done  bool `json:"d,omitempty"`
}

type gcTwState struct {
	Rest    map[string]float64   `json:"r"`
	Last    map[string]int       `json:"l"`
	Cover   map[string]float64   `json:"c"`
	Beacons map[string]*gcBeacon `json:"b"`
	Wards   map[string]*gcWard   `json:"w"`
	Dark    map[string]bool      `json:"dk,omitempty"`
	Tier    map[string]int       `json:"tr,omitempty"`
	Firsts  map[string]bool      `json:"f"`
	Day     int                  `json:"d"`
	Rise    int                  `json:"ri"`
}

func (g *Game) gcTw() *gcTwState {
	st := g.ext("gct", func() any { return &gcTwState{} }).(*gcTwState)
	if st.Rest == nil {
		st.Rest = map[string]float64{}
	}
	if st.Last == nil {
		st.Last = map[string]int{}
	}
	if st.Cover == nil {
		st.Cover = map[string]float64{}
	}
	if st.Beacons == nil {
		st.Beacons = map[string]*gcBeacon{}
	}
	if st.Wards == nil {
		st.Wards = map[string]*gcWard{}
	}
	if st.Dark == nil {
		st.Dark = map[string]bool{}
	}
	if st.Tier == nil {
		st.Tier = map[string]int{}
	}
	if st.Firsts == nil {
		st.Firsts = map[string]bool{}
	}
	return st
}

func init() {
	unLocalKeys = append(unLocalKeys, "gct")
	zoneHooks = append(zoneHooks, gcTwZone)
	enterHooks = append(enterHooks, gcTwEnter)
	turnHooks = append(turnHooks, gcTwTurn)
	sideHooks = append(sideHooks, gcTwSide)
	talkHooks = append(talkHooks, gcTwTalk)
	killHooks = append(killHooks, gcTwKill)
	gcBumpers = append(gcBumpers, gcTwBump)
}

func (g *Game) gcTwDeed(id, text string) {
	st := g.gcTw()
	if st.Firsts[id] {
		return
	}
	st.Firsts[id] = true
	g.deed("gc", fmt.Sprintf("The wanderer %s %s.", g.P.Name, text), 0, 0)
}

// ---- the clock ----

func gcStateOfZone(g *Game, zn *Zone) int {
	if zn == nil || g.W == nil {
		return 0
	}
	if bd := g.W.burgByID[zn.Burg]; bd != nil && zn.Kind == "town" {
		return bd.State
	}
	if g.W.In(zn.WX, zn.WY) {
		return g.W.State[g.W.Idx(zn.WX, zn.WY)]
	}
	return 0
}

// gcDecayOf: the decay of a realm today, 0..100.
func (g *Game) gcDecayOf(state int) float64 {
	if g.Genre != "twilight" {
		return 0
	}
	st := g.gcTw()
	k := fmt.Sprint(state)
	base := 15 + float64(HashStr(fmt.Sprintf("%s:gc:decay:%d", g.Seed, state))%30)
	day := float64(g.Day())
	neglect := math.Min(25, 0.2*math.Max(0, day-float64(st.Last[k])-10))
	v := base + 0.3*day + neglect - st.Rest[k] - 0.15*st.Cover[k]
	return math.Max(0, math.Min(100, v))
}

// gcDecayHere: decay of the realm you stand in.
func (g *Game) gcDecayHere() float64 { return g.gcDecayOf(gcStateOfZone(g, g.Cur)) }

func gcDecayName(d float64) string {
	switch {
	case d >= 80:
		return "Fallen"
	case d >= 60:
		return "Rotting"
	case d >= 40:
		return "Failing"
	case d >= 20:
		return "Fraying"
	}
	return "Hale"
}

func gcDecayTier(d float64) int { return int(d) / 20 }

// restore lowers a realm's decay by n points and marks it tended today.
func (g *Game) gcRestore(state int, n float64) {
	st := g.gcTw()
	k := fmt.Sprint(state)
	before := g.gcDecayOf(state)
	st.Rest[k] += n
	st.Last[k] = g.Day()
	if after := g.gcDecayOf(state); after < before {
		g.Msg(fmt.Sprintf("{{G|Decay in %s eases: %d to %d.}}", g.uwRealm(state), int(before), int(after)))
		if after < 20 {
			g.gcTwDeed("hale", "brought a failing realm back to health")
		}
	}
	if state > 0 {
		g.Rep[uwFac(state)] += 4
	}
}

// gcDecayPriceMod: price factor from decay (hook in price()): failing crops make food dear, dark nights light.
func gcDecayPriceMod(g *Game, a *Actor, it *Item) float64 {
	if g == nil || g.Genre != "twilight" || g.Cur == nil || it == nil {
		return 1
	}
	d := g.gcDecayHere()
	switch it.Kind {
	case "food", "ingredient", "dish", "corpse":
		if d > 35 {
			return 1 + math.Min(1.2, (d-35)/65*1.2)
		}
	case "light":
		return 1 + d/150
	case "gcitem":
		if it.Effect == "fuel" {
			return 1 + d/100
		}
	}
	if d > 60 {
		return 1 + (d-60)/200
	}
	return 1
}

// gcOverlay: world-map hint for render.go (decay in twilight, fallout in wasteland); known parasangs only.
func (g *Game) gcOverlay(p int) (rune, byte, byte, bool) {
	switch g.Genre {
	case "wasteland":
		return g.gcWaOverlay(p)
	case "twilight":
		if g.W == nil || !g.Known[p] || g.W.IsWater(p) {
			return 0, 0, 0, false
		}
		d := g.gcDecayOf(g.W.State[p])
		switch {
		case d >= 80:
			return '▒', CMagenta, CDarkMagenta, true
		case d >= 60:
			return '░', CDarkMagenta, CBgDark, true
		}
	}
	return 0, 0, 0, false
}

// ---- zones ----

func gcTwZone(g *Game, zn *Zone, _ *Rng) {
	if g.Genre != "twilight" || g.W == nil || zn == nil {
		return
	}
	r := NewRng(g.Seed + ":gc:tw:" + zn.Key)
	state := gcStateOfZone(g, zn)
	d := g.gcDecayOf(state)
	st := g.gcTw()
	switch {
	case zn.Kind == "town" && zn.Z == 0 && zn.Burg >= 0:
		bd := g.W.burgByID[zn.Burg]
		if bd == nil || !bd.Alive {
			return
		}
		if zn.ZX == 1 && zn.ZY == 1 {
			n := 1 + g.W.cityTier[bd.ID]/2
			for i := 0; i < n; i++ {
				x, y, ok := gcFree(zn, r, gcWallside(zn))
				if !ok {
					break
				}
				zn.Set(x, y, TPillar)
				b := &gcBeacon{State: bd.State, X: x, Y: y, Until: 0}
				st.Beacons[gcID(zn.Key, x, y)] = b
			}
		}
	case zn.Kind == "ruin" && zn.Z == 0:
		if r.Chance(0.75) {
			if x, y, ok := gcFree(zn, r, nil); ok {
				zn.Set(x, y, TMonument)
				st.Wards[gcID(zn.Key, x, y)] = &gcWard{State: state, X: x, Y: y, Tier: 1 + r.Int(0, 2)}
			}
		}
	}
	if d >= 40 && zn.Z == 0 && zn.Kind != "town" && !zn.Underwater && !zn.Sky {
		n := 0
		if r.Chance(d / 160) {
			n++
		}
		if d >= 70 && r.Chance(0.4) {
			n++
		}
		ids := []string{"skeleton", "ashwraith"}
		if d >= 65 {
			ids = append(ids, "ghoul")
		}
		for i := 0; i < n; i++ {
			x, y, ok := gcFree(zn, r, nil)
			if !ok {
				break
			}
			a := g.Spawn(Pick(r, ids), zn.Level+1+int(d)/30, r)
			a.X, a.Y, a.HomeX, a.HomeY = x, y, x, y
			zn.Actors = append(zn.Actors, a)
		}
	}
	// crops fail
	if d >= 35 && (zn.Kind == "town" || zn.Kind == "outskirts" || zn.Kind == "surface") {
		p := (d - 30) / 100
		for i, t := range zn.T {
			if t == TCrop && r.Chance(p) {
				zn.T[i] = TDirt
			}
		}
	}
	gcTwDarken(g, zn, d)
}

// gcTwDarken puts out town lanterns and braziers as decay rises (hash per tile, never g.R).
func gcTwDarken(g *Game, zn *Zone, d float64) {
	if zn.Kind != "town" || d < 45 {
		return
	}
	thr := (d - 40) / 70
	st := g.gcTw()
	changed := false
	for i, t := range zn.T {
		if t != TBrazier && t != TLantern && t != TChandelier {
			continue
		}
		x, y := i%MW, i/MW
		if st.Beacons[gcID(zn.Key, x, y)] != nil {
			continue
		}
		if float64(HashStr(fmt.Sprintf("%s:gc:dark:%s:%d", g.Seed, zn.Key, i))%1000)/1000 < thr {
			if t == TChandelier {
				zn.T[i] = TFloor
			} else {
				zn.T[i] = TFloor
			}
			changed = true
		}
	}
	if changed && zn == g.Cur {
		g.computeLight(zn)
	}
}

func gcTwEnter(g *Game, zn *Zone) {
	if g.Genre != "twilight" || zn == nil {
		return
	}
	d := g.gcDecayOf(gcStateOfZone(g, zn))
	if zn.Kind == "town" && d >= 45 {
		before := 0
		for _, t := range zn.T {
			if t == TBrazier || t == TLantern {
				before++
			}
		}
		gcTwDarken(g, zn, d)
		g.computeLight(zn)
		if d >= 60 {
			g.Msg("{{K|The lanterns here are mostly dark. People keep their doors shut.}}")
		}
		_ = before
	}
	gcTwSyncBeacons(g, zn)
}

func gcTwSyncBeacons(g *Game, zn *Zone) {
	changed := false
	for id, b := range g.gcTw().Beacons {
		if !strings.HasPrefix(id, zn.Key+":") {
			continue
		}
		want := TPillar
		if b.Until > g.Turn {
			want = TBrazier
		}
		if zn.At(b.X, b.Y) != want {
			zn.Set(b.X, b.Y, want)
			changed = true
		}
	}
	if changed {
		g.computeLight(zn)
	}
}

// ---- turn ----

func gcTwTurn(g *Game) {
	if g.Genre != "twilight" || g.Dead || g.Cur == nil || g.Mode == "world" {
		return
	}
	st := g.gcTw()
	zn := g.Cur
	// beacons burning bank restoration
	for _, b := range st.Beacons {
		if b.Until > g.Turn {
			st.Cover[fmt.Sprint(b.State)] += 1.0 / 1440
		}
	}
	if g.Turn%5 == 0 {
		gcTwSyncBeacons(g, zn)
	}
	state := gcStateOfZone(g, zn)
	d := g.gcDecayOf(state)
	// day rollover: announce tiers
	if day := g.Day(); day != st.Day {
		st.Day = day
		k := fmt.Sprint(state)
		if t := gcDecayTier(d); t > st.Tier[k] && t >= 2 {
			g.Msg("{{K|The days grow shorter in " + g.uwRealm(state) + ". Decay: " + gcDecayName(d) + ".}}")
			if t >= 3 {
				g.Msg("{{K|Last season's grain rotted in the bins.}}")
			}
		}
		st.Tier[k] = gcDecayTier(d)
	}
	if d < 50 {
		return
	}
	p := g.P
	night := !g.daylight()
	// the dead rise at night
	if night && zn.Z == 0 && g.Turn%15 == 0 && g.Turn-st.Rise >= 30 && (zn.Kind != "town" || d >= 70) && !zn.Underwater && g.R.Chance((d-40)/260) {
		undead := 0
		for _, a := range zn.Actors {
			if a.HP > 0 && a.Faction == "undead" {
				undead++
			}
		}
		if undead < 6 {
			st.Rise = g.Turn
			n := 1 + int(d)/45
			made := 0
			for i := 0; i < n; i++ {
				x, y, ok := zn.FreeSpot(g.R, func(x, y int) bool {
					dd := maxi(abs(x-p.X), abs(y-p.Y))
					return dd >= 6 && dd <= 14
				})
				if !ok {
					break
				}
				id := "skeleton"
				if d >= 65 && g.R.Chance(0.4) {
					id = "ghoul"
				}
				a := g.Spawn(id, zn.Level+1+int(d)/30, g.R)
				a.X, a.Y, a.HomeX, a.HomeY = x, y, x, y
				zn.Actors = append(zn.Actors, a)
				made++
			}
			if made > 0 {
				g.Msg("{{M|The ground heaves. The dead are walking again.}}")
			}
		}
	}
	// plague in towns
	if zn.Kind == "town" && zn.Z == 0 && d >= 60 && g.Turn%120 == 0 && p.St("ill") == 0 && g.R.Chance((d-55)/160) {
		p.AddSt("ill", 60)
		g.Msg("{{G|A grey cough runs through the town, and you catch it. The decay is making the people sick.}}")
	}
}

func gcTwKill(g *Game, d, _ *Actor) {
	if g.Genre != "twilight" || d == nil || d.Faction != "undead" || g.Cur == nil {
		return
	}
	st := g.gcTw()
	state := gcStateOfZone(g, g.Cur)
	k := fmt.Sprint(state)
	st.Rest[k] += 0.12
	st.Last[k] = g.Day()
	if g.R.Chance(0.15) {
		g.Msg("{{c|The restless dead lie quiet. The land feels a little lighter.}}")
	}
	g.gcTwDeed("rest", "laid the restless dead to rest")
}

func gcTwSide(g *Game) []string {
	if g.Genre != "twilight" || g.Cur == nil || g.W == nil {
		return nil
	}
	d := g.gcDecayHere()
	col := "G"
	switch {
	case d >= 80:
		col = "R"
	case d >= 60:
		col = "O"
	case d >= 40:
		col = "Y"
	}
	return []string{fmt.Sprintf("{{%s|Decay %d %s}}", col, int(d), gcDecayName(d))}
}

// ---- beacons and wards ----

func gcTwBump(g *Game, x, y int) bool {
	if g.Genre != "twilight" {
		return false
	}
	id := gcID(g.Cur.Key, x, y)
	st := g.gcTw()
	if b := st.Beacons[id]; b != nil {
		g.gcBeaconMenu(b)
		return true
	}
	if w := st.Wards[id]; w != nil {
		g.gcWardMenu(w)
		return true
	}
	return false
}

func (g *Game) gcFuelOptions() []struct {
	name  string
	hours int
	use   func()
} {
	var out []struct {
		name  string
		hours int
		use   func()
	}
	add := func(n string, h int, f func()) {
		out = append(out, struct {
			name  string
			hours int
			use   func()
		}{n, h, f})
	}
	for _, it := range g.P.Inv {
		it := it
		switch {
		case it.Kind == "gcitem" && it.Effect == "fuel":
			add("a flask of lamp oil", 24, func() { g.removeInv(it) })
		case it.Kind == "light":
			add("your "+it.Name, 6, func() { g.removeInv(it) })
		case it.Kind == "scrap" && maxi(it.Count, 1) >= 3:
			add("3 "+it.Name, 3, func() {
				if it.Count > 3 {
					it.Count -= 3
				} else {
					g.removeInv(it)
				}
			})
		}
	}
	return out
}

func (g *Game) gcBeaconMenu(b *gcBeacon) {
	left := (b.Until - g.Turn) / 60
	text := "A beacon-post of iron, set to be lit against the dark."
	if left > 0 {
		text += fmt.Sprintf(" It burns, with about %d hours of fuel left. Decay in %s: %d.", left, g.uwRealm(b.State), int(g.gcDecayOf(b.State)))
	} else {
		text += " It is cold. The people say it has been dark for a long time."
	}
	var opts []Option
	seen := map[string]bool{}
	for _, f := range g.gcFuelOptions() {
		f := f
		if seen[f.name] {
			continue
		}
		seen[f.name] = true
		opts = append(opts, Option{fmt.Sprintf("{{O|Feed it %s (%d hours).}}", f.name, f.hours), func() {
			f.use()
			cold := b.Until <= g.Turn
			if cold {
				b.Until = g.Turn
			}
			b.Until += f.hours * 60
			if b.Until-g.Turn > 72*60 {
				b.Until = g.Turn + 72*60
			}
			g.Cur.Set(b.X, b.Y, TBrazier)
			g.computeLight(g.Cur)
			if cold {
				g.Msg("{{Y|The beacon catches. Light climbs the square, and faces come to the windows.}}")
				g.gcRestore(b.State, 1.5)
				g.gcTwDeed("beacon", "relit a dead beacon")
				g.gainXP(12)
			} else {
				g.Msg("You feed the beacon.")
				g.gcRestore(b.State, 0.4)
			}
			g.gcClose()
			g.spend(2000)
		}})
	}
	if len(opts) == 0 {
		text += " You carry nothing to burn: torches, lanterns, lamp oil, or three pieces of scrap would do."
	}
	opts = append(opts, Option{"Walk away.", func() { g.gcClose() }})
	g.gcMenu("gcm", "BEACON", text, opts)
}

func (g *Game) gcWardMenu(w *gcWard) {
	if w.Done {
		g.gcMenu("gcm", "WARD-STONE", "The ward-stone hums, its old words bright again. The ground around it feels quiet.", []Option{{"Leave it.", func() { g.gcClose() }}})
		return
	}
	text := fmt.Sprintf("A cracked ward-stone, grey with age. Its words have gone dull: whatever it kept out has been walking in. Strength %d.", w.Tier)
	scrap := func() *Item {
		for _, it := range g.P.Inv {
			if it.Kind == "scrap" && maxi(it.Count, 1) >= 4 {
				return it
			}
		}
		return nil
	}
	var opts []Option
	finish := func(ok bool, bonus float64) {
		if ok {
			w.Done = true
			g.Msg("{{Y|The ward-stone flares and settles. Its words burn clean.}}")
			g.gcRestore(w.State, 2.5+bonus)
			g.gcTwDeed("ward", "mended a ward-stone in an old ruin")
			g.gainXP(20 + 5*w.Tier)
		} else {
			g.Msg("{{R|The stone spits grey light. Something has noticed you.}}")
			if g.R.Chance(0.6) {
				if x, y, ok := g.nearestFree(g.Cur, g.P.X+2, g.P.Y); ok {
					a := g.Spawn("ashwraith", g.Cur.Level+w.Tier+1, g.R)
					a.X, a.Y, a.HomeX, a.HomeY = x, y, x, y
					g.Cur.Actors = append(g.Cur.Actors, a)
				}
			}
		}
		g.gcClose()
		g.spend(3000)
	}
	if s := scrap(); s != nil {
		opts = append(opts, Option{fmt.Sprintf("{{O|Patch it with scrap and craft.}} {{K|(4 scrap, Tinkering, DC %d)}}", 8+w.Tier), func() {
			ok := g.R.Dice(2, 6)+g.checkMod("tinkering") >= 8+w.Tier
			if s.Count > 4 {
				s.Count -= 4
			} else if ok {
				g.removeInv(s)
			}
			finish(ok, 0)
		}})
	}
	opts = append(opts, Option{fmt.Sprintf("{{C|Recite the old ward-words.}} {{K|(Lore, DC %d)}}", 9+w.Tier), func() {
		finish(g.R.Dice(2, 6)+g.checkMod("lore") >= 9+w.Tier, 0)
	}})
	for _, it := range g.P.Inv {
		it := it
		if it.Kind == "artifact" && it.SimArt > 0 {
			opts = append(opts, Option{"{{M|Bind the relic " + it.Name + " into the stone.}} {{K|(it is lost; surest)}}", func() {
				g.removeInv(it)
				finish(true, 2.0)
			}})
			break
		}
	}
	opts = append(opts, Option{"Walk away.", func() { g.gcClose() }})
	g.gcMenu("gcm", "WARD-STONE", text, opts)
}

// ---- relics and priests ----

func gcTwTalk(g *Game, a *Actor) []Option {
	if g.Genre != "twilight" || a == nil || g.Cur == nil {
		return nil
	}
	var out []Option
	if a.Role == "merchant" && !a.Has("gcfuel") {
		a.Set("gcfuel")
		r := NewRng(fmt.Sprintf("%s:gc:fuel:%d", g.Seed, a.ID))
		if r.Chance(0.6) {
			a.Inv = append(a.Inv, &Item{ID: g.newID(), Kind: "gcitem", Effect: "fuel", Name: "flask of lamp oil", Glyph: '!', F: COrange, Value: 14, Weight: 1, Ident: true,
				Lore: "Enough oil to keep a beacon burning for a day. Bump a beacon with it in your pack."})
		}
	}
	if a.Role == "priest" || a.Role == "elder" {
		state := gcStateOfZone(g, g.Cur)
		for _, it := range g.P.Inv {
			it := it
			if it.Kind == "artifact" && it.SimArt > 0 {
				out = append(out, Option{"{{M|Return the relic " + it.Name + " to the shrine.}}", func() {
					g.removeInv(it)
					pts := 4.0 + float64(it.Tier)
					g.Msg("{{Y|" + a.Name + " lays it upon the altar. The air in the room loosens.}}")
					g.gcRestore(state, pts)
					g.gcTwDeed("relic", "returned a relic to a shrine and eased the decay")
					g.cuEarn(maxi(20, it.Value/3))
					g.gainXP(40)
					g.talkMenu(g.greeting(a))
				}})
			}
		}
		if d := g.gcDecayOf(state); d >= 30 {
			out = append(out, Option{"{{K|Ask how the land fares.}}", func() {
				g.talkMenu(fmt.Sprintf("\"%s is %s,\" they say. \"Decay stands at %d of a hundred. Relight the beacons, mend the ward-stones in the ruins, give the shrines back their relics, and put the dead to rest.\"", g.uwRealm(state), strings.ToLower(gcDecayName(d)), int(d)))
			}})
		}
	}
	return out
}
