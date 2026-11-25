package engine

import (
	"fmt"
	"sort"
	"strings"
)

// chemcombo.go (prefix cc):
// 1. Combo & Reaction Synergies: Wet+Shock chain lightning, Oil+Fire spreading flames,
//    Frost+Heavy Bludgeon shatter shrapnel, Smoke+Backstab/Unaware guaranteed crit + bleed.
// 2. Captive-Taking, Ransom & Parley: subdue weakened foes, march captives, ransom at
//    faction strongholds/guards for gold & standing, or recruit under oath.
// 3. Fortified Wilderness Camp Defense: pitch barricades, tripwires, watchfire, and
//    companion sentry shifts; night ambushes trigger tripwire warnings & bonus defense.
//
// Saved state: g.ext("cc"), planet-local ("cc" in unLocalKeys).
// Screen: mode "cc", palette entry "Tactics, captives & camp".

const ccExtKey = "cc"

type ccCaptive struct {
	ID        int    `json:"id"`
	Name      string `json:"n"`
	Tpl       string `json:"tp"`
	Faction   string `json:"fa"`
	State     int    `json:"st,omitempty"`
	Level     int    `json:"lv"`
	HP        int    `json:"hp"`
	MaxHP     int    `json:"mhp"`
	Ransom    int    `json:"rn"`
	OathReady bool   `json:"or,omitempty"`
	BoundDay  int    `json:"bd"`
	ZoneKey   string `json:"zk,omitempty"`
	Role      string `json:"ro,omitempty"`
	Glyph     rune   `json:"g,omitempty"`
	F         byte   `json:"f,omitempty"`
	Desc      string `json:"ds,omitempty"`
}

type ccCamp struct {
	ZoneKey      string `json:"zk"`
	X            int    `json:"x"`
	Y            int    `json:"y"`
	DayPitched   int    `json:"dp"`
	Barricades   int    `json:"br"`
	Tripwires    int    `json:"tw"`
	Watchfire    bool   `json:"wf"`
	SentryName   string `json:"sn,omitempty"`
	SentryShift  string `json:"ss,omitempty"` // first | mid | dawn
	DefenseBonus int    `json:"db"`
	Active       bool   `json:"ac"`
	LastAmbush   int    `json:"la,omitempty"`
}

type ccState struct {
	Combos           int                `json:"co,omitempty"`
	ComboCounts      map[string]int     `json:"cc,omitempty"`
	SmokeTiles       map[string]int     `json:"sm,omitempty"`
	Captives         []*ccCaptive       `json:"cp,omitempty"`
	Ransomed         int                `json:"rn,omitempty"`
	Recruited        int                `json:"rc,omitempty"`
	Camps            map[string]*ccCamp `json:"cm,omitempty"`
	AmbushesRepelled int                `json:"ar,omitempty"`
	Log              []string           `json:"lg,omitempty"`

	tab int
	sel int
}

func (g *Game) cc() *ccState {
	st := g.ext(ccExtKey, func() any {
		return &ccState{
			ComboCounts: map[string]int{},
			SmokeTiles:  map[string]int{},
			Camps:       map[string]*ccCamp{},
		}
	}).(*ccState)
	if st.ComboCounts == nil {
		st.ComboCounts = map[string]int{}
	}
	if st.SmokeTiles == nil {
		st.SmokeTiles = map[string]int{}
	}
	if st.Camps == nil {
		st.Camps = map[string]*ccCamp{}
	}
	return st
}

func init() {
	unLocalKeys = append(unLocalKeys, ccExtKey)
	turnHooks = append(turnHooks, ccTurn)
	enterHooks = append(enterHooks, ccEnter)
	talkHooks = append(talkHooks, ccTalk)
	sideHooks = append(sideHooks, ccSide)
	modeKeys["cc"] = ccKeys
	modeDraws["cc"] = func(g *Game) { g.ccDraw() }
	paEntries = append(paEntries, paEntry{
		Name:     "Tactics, captives & camp",
		Keywords: "combo reaction elemental wet shock oil fire frost shatter smoke backstab captive subdue ransom oath fortified camp ambush tripwire",
		Cat:      "Screen",
		Label:    "screen",
		Func:     func(g *Game) { g.ccOpen() },
	})
}

func (g *Game) ccOpen() {
	st := g.cc()
	st.tab, st.sel = 0, 0
	g.Mode = "cc"
}

func (g *Game) ccLogf(s string) {
	st := g.cc()
	st.Log = append(st.Log, fmt.Sprintf("Day %d: %s", g.Day()+1, s))
	if len(st.Log) > 24 {
		st.Log = st.Log[len(st.Log)-24:]
	}
}

func ccSmokeKey(zoneKey string, x, y int) string {
	return fmt.Sprintf("%s:%d,%d", zoneKey, x, y)
}

func (g *Game) ccAddSmoke(zoneKey string, x, y, turns int) {
	if turns <= 0 {
		return
	}
	st := g.cc()
	k := ccSmokeKey(zoneKey, x, y)
	if st.SmokeTiles[k] < turns {
		st.SmokeTiles[k] = turns
	}
}

func (g *Game) ccSmokeAt(zoneKey string, x, y int) int {
	return g.cc().SmokeTiles[ccSmokeKey(zoneKey, x, y)]
}

// ---- 1. Combo & Reaction Synergies ----

func ccWetAt(g *Game, a *Actor) bool {
	if g == nil || a == nil {
		return false
	}
	if a.St("wet") > 0 || a.St("soaked") > 0 {
		return true
	}
	if g.Cur != nil && g.Cur.In(a.X, a.Y) && g.Cur.WaterAt(a.X, a.Y) > 0 {
		return true
	}
	if wx := g.activeWeather(); wx != nil && wx.Kind != nil && !g.underRoof(a.X, a.Y) {
		switch wx.Kind.ID {
		case "rain", "storm", "monsoon", "downpour":
			return true
		}
	}
	return false
}

func ccOiledAt(g *Game, a *Actor) bool {
	if g == nil || a == nil {
		return false
	}
	if a.St("oiled") > 0 || a.St("oil") > 0 {
		return true
	}
	if g.Cur != nil && g.Cur.In(a.X, a.Y) && len(g.Cur.Liq) == MW*MH && g.Cur.Liq[a.Y*MW+a.X] == 2 {
		return true
	}
	return false
}

func ccFrozenAt(a *Actor) bool {
	if a == nil {
		return false
	}
	return a.St("frozen") > 0 || a.St("chilled") > 0
}

func ccSmokedAt(g *Game, a *Actor) bool {
	if g == nil || a == nil {
		return false
	}
	if a.St("smoke") > 0 || a.St("blinded") > 0 {
		return true
	}
	if g.Cur != nil && g.ccSmokeAt(g.Cur.Key, a.X, a.Y) > 0 {
		return true
	}
	return false
}

func ccUnawareOfPlayer(g *Game, a *Actor) bool {
	if g == nil || a == nil {
		return false
	}
	if a.St("unaware") > 0 || a.St("asleep") > 0 || a.St("blinded") > 0 {
		return true
	}
	return !a.Angry && a.AI != "hostile"
}

// ccApplyElementalCombo checks and executes elemental/tactical reaction combos on target.
// Returns (comboID, bonusDamageDealt).
func ccApplyElementalCombo(g *Game, target *Actor, elem, weaponSkill string) (string, int) {
	if g == nil || g.Cur == nil || g.P == nil || target == nil || target.HP <= 0 {
		return "", 0
	}
	z := g.Cur
	st := g.cc()
	el := strings.ToLower(strings.TrimSpace(elem))
	sk := strings.ToLower(strings.TrimSpace(weaponSkill))

	// 1. Wet + Shock: chains lightning damage to adjacent wet/standing-in-water targets.
	if el == "shock" && ccWetAt(g, target) {
		bonus := 6 + g.R.Int(2, 6)
		g.hurt(target, bonus, g.P, "chained lightning")
		target.AddSt("dazed", 2)
		chained := 0
		for _, other := range z.Actors {
			if other == nil || other == g.P || other == target || other.HP <= 0 {
				continue
			}
			if dist(target, other) <= 2 && ccWetAt(g, other) {
				cdmg := 5 + g.R.Int(2, 5)
				g.hurt(other, cdmg, g.P, "chained lightning")
				other.AddSt("dazed", 1)
				chained++
			}
		}
		st.Combos++
		st.ComboCounts["wet_shock"]++
		if chained > 0 {
			g.Msg(fmt.Sprintf("{{C|LIGHTNING CHAIN!}} Shock surges through soaked %s (+%d dmg) and arcs to %d wet foe(s)!", target.Name, bonus, chained))
		} else {
			g.Msg(fmt.Sprintf("{{C|CONDUCTION!}} Shock surges through wet %s for +%d bonus damage!", target.Name, bonus))
		}
		g.ccLogf(fmt.Sprintf("Wet+Shock combo on %s (+%d dmg, %d chained).", target.Name, bonus, chained))
		return "wet_shock", bonus
	}

	// 2. Oil + Fire: ignites spreading flame tiles (z.Fire) and applies burning.
	if el == "fire" && ccOiledAt(g, target) {
		bonus := 8 + g.R.Int(3, 7)
		g.hurt(target, bonus, g.P, "ignited oil")
		target.AddSt("burning", 6)
		if target.Status != nil {
			delete(target.Status, "oiled")
			delete(target.Status, "oil")
		}
		ignited := 0
		for dy := -1; dy <= 1; dy++ {
			for dx := -1; dx <= 1; dx++ {
				nx, ny := target.X+dx, target.Y+dy
				if !z.In(nx, ny) || !z.Walkable(nx, ny) {
					continue
				}
				idx := ny*MW + nx
				if len(z.Fire) == MW*MH {
					if z.Fire[idx] < 6 {
						z.Fire[idx] = 6
					}
					ignited++
				}
				if len(z.Liq) == MW*MH && z.Liq[idx] == 2 {
					z.Liq[idx] = 0
				}
				if other := z.ActorAt(nx, ny); other != nil && other != g.P && other != target && other.HP > 0 {
					other.AddSt("burning", 4)
				}
			}
		}
		// Spread fire to any adjacent oil slick tiles one ring further.
		for dy := -2; dy <= 2; dy++ {
			for dx := -2; dx <= 2; dx++ {
				nx, ny := target.X+dx, target.Y+dy
				if !z.In(nx, ny) {
					continue
				}
				idx := ny*MW + nx
				if len(z.Liq) == MW*MH && z.Liq[idx] == 2 && len(z.Fire) == MW*MH {
					z.Liq[idx] = 0
					if z.Fire[idx] < 5 {
						z.Fire[idx] = 5
					}
					ignited++
				}
			}
		}
		st.Combos++
		st.ComboCounts["oil_fire"]++
		g.Msg(fmt.Sprintf("{{R|FLASH IGNITION!}} Oil erupts around %s (+%d fire dmg), spreading flames across %d tiles!", target.Name, bonus, ignited))
		g.ccLogf(fmt.Sprintf("Oil+Fire combo on %s (+%d dmg, %d tiles ignited).", target.Name, bonus, ignited))
		return "oil_fire", bonus
	}

	// 3. Frost + Heavy Bludgeon (cudgel/heavy): shatters frozen target for bonus shard damage + AoE shrapnel.
	if (sk == "cudgel" || sk == "heavy" || el == "shatter") && ccFrozenAt(target) {
		bonus := 10 + g.R.Int(4, 9)
		g.hurt(target, bonus, g.P, "ice shatter")
		if target.Status != nil {
			delete(target.Status, "frozen")
			delete(target.Status, "chilled")
		}
		hits := 0
		for _, other := range z.Actors {
			if other == nil || other == g.P || other == target || other.HP <= 0 {
				continue
			}
			if dist(target, other) <= 1 {
				sdmg := 5 + g.R.Int(2, 5)
				g.hurt(other, sdmg, g.P, "ice shrapnel")
				other.AddSt("bleeding", 4)
				hits++
			}
		}
		st.Combos++
		st.ComboCounts["frost_shatter"]++
		if hits > 0 {
			g.Msg(fmt.Sprintf("{{C|SHATTER!}} Heavy impact splinters frozen %s (+%d dmg), spraying ice shrapnel into %d nearby foe(s)!", target.Name, bonus, hits))
		} else {
			g.Msg(fmt.Sprintf("{{C|SHATTER!}} Heavy impact splinters frozen %s for +%d shard damage!", target.Name, bonus))
		}
		g.ccLogf(fmt.Sprintf("Frost+Heavy shatter on %s (+%d dmg, %d shrapnel hits).", target.Name, bonus, hits))
		return "frost_shatter", bonus
	}

	// 4. Smoke + Backstab/Unaware (shortblade or stealth): guaranteed critical strike + bonus bleeding.
	if (sk == "shortblade" || sk == "stealth" || el == "backstab" || el == "stealth") && (ccSmokedAt(g, target) || ccUnawareOfPlayer(g, target)) {
		bonus := 9 + g.R.Int(4, 8)
		g.hurt(target, bonus, g.P, "smoke backstab critical")
		target.AddSt("bleeding", 6)
		target.AddSt("dazed", 2)
		st.Combos++
		st.ComboCounts["smoke_backstab"]++
		g.Msg(fmt.Sprintf("{{M|CRITICAL BACKSTAB!}} From cover of smoke you strike %s for +%d critical damage and deep bleeding!", target.Name, bonus))
		g.ccLogf(fmt.Sprintf("Smoke+Backstab crit on %s (+%d dmg).", target.Name, bonus))
		return "smoke_backstab", bonus
	}

	return "", 0
}

// ccThrowPrimer applies a tactical primer (water, oil, frost, smoke) onto target or around the player.
func (g *Game) ccThrowPrimer(kind string, target *Actor) (bool, string) {
	if g == nil || g.Cur == nil || g.P == nil {
		return false, "No active zone."
	}
	z := g.Cur
	cx, cy := g.P.X, g.P.Y
	if target != nil {
		cx, cy = target.X, target.Y
	}
	switch kind {
	case "water":
		if target != nil {
			target.AddSt("wet", 6)
		}
		for dy := -1; dy <= 1; dy++ {
			for dx := -1; dx <= 1; dx++ {
				if a := z.ActorAt(cx+dx, cy+dy); a != nil && a != g.P {
					a.AddSt("wet", 6)
				}
			}
		}
		g.spend(500)
		msg := "You toss a water bladder, soaking the area for shock conduction."
		g.Msg("{{C|" + msg + "}}")
		return true, msg
	case "oil":
		for dy := -1; dy <= 1; dy++ {
			for dx := -1; dx <= 1; dx++ {
				nx, ny := cx+dx, cy+dy
				if z.In(nx, ny) && z.Walkable(nx, ny) && len(z.Liq) == MW*MH {
					z.Liq[ny*MW+nx] = 2
				}
				if a := z.ActorAt(nx, ny); a != nil && a != g.P {
					a.AddSt("oiled", 8)
				}
			}
		}
		g.spend(500)
		msg := "You smash an oil flask, slicking the ground for ignition."
		g.Msg("{{Y|" + msg + "}}")
		return true, msg
	case "frost":
		if target != nil {
			target.AddSt("frozen", 4)
		}
		g.spend(500)
		msg := "You crack a rime vial, encasing the target in brittle frost."
		g.Msg("{{C|" + msg + "}}")
		return true, msg
	case "smoke":
		for dy := -1; dy <= 1; dy++ {
			for dx := -1; dx <= 1; dx++ {
				nx, ny := cx+dx, cy+dy
				if z.In(nx, ny) {
					g.ccAddSmoke(z.Key, nx, ny, 8)
					if a := z.ActorAt(nx, ny); a != nil && a != g.P {
						a.AddSt("smoke", 6)
					}
				}
			}
		}
		g.spend(500)
		msg := "You drop a smoke pellet; thick gray haze blankets the ground."
		g.Msg("{{K|" + msg + "}}")
		return true, msg
	}
	return false, "Unknown primer."
}

func (g *Game) ccNearestFoe(maxDist int) *Actor {
	if g == nil || g.Cur == nil || g.P == nil {
		return nil
	}
	var best *Actor
	bestD := maxDist + 1
	for _, a := range g.Cur.Actors {
		if a == nil || a == g.P || a.HP <= 0 || a.AI == "follower" {
			continue
		}
		if a.AI != "hostile" && !a.Angry {
			continue
		}
		d := dist(g.P, a)
		if d < bestD {
			bestD = d
			best = a
		}
	}
	return best
}

// ---- 2. Captive-Taking, Ransom & Parley ----

func ccCanSubdue(a *Actor) bool {
	if a == nil || a.HP <= 0 || a.AI == "follower" {
		return false
	}
	if a.MaxHP > 0 && a.HP*100 <= a.MaxHP*40 {
		return true
	}
	return a.St("dazed") > 0 || a.St("frozen") > 0 || a.St("pinned") > 0 || a.St("asleep") > 0
}

// ccSubdue binds a weakened hostile or named foe as a marching captive.
func (g *Game) ccSubdue(a *Actor) (bool, string) {
	if g == nil || g.Cur == nil || g.P == nil || a == nil {
		return false, "No target to subdue."
	}
	if a == g.P || a.AI == "follower" {
		return false, "You cannot subdue an ally."
	}
	if a.HP <= 0 {
		return false, "They are already dead."
	}
	if !ccCanSubdue(a) {
		return false, fmt.Sprintf("%s is still too strong to subdue (weaken below 40%% HP or daze/pin them).", a.Name)
	}
	st := g.cc()
	if len(st.Captives) >= 6 {
		return false, "You already have too many captives in tow."
	}

	ransom := 20 + maxi(a.Level, 1)*12
	if a.Proper {
		ransom += 25
	}
	if a.Boss {
		ransom += 60
	}
	if a.Role != "" {
		ransom += 10
	}

	capRec := &ccCaptive{
		ID:        a.ID,
		Name:      a.Name,
		Tpl:       a.Tpl,
		Faction:   a.Faction,
		State:     a.State,
		Level:     maxi(a.Level, 1),
		HP:        a.HP,
		MaxHP:     maxi(a.MaxHP, a.HP),
		Ransom:    ransom,
		OathReady: true,
		BoundDay:  g.Day(),
		ZoneKey:   g.Cur.Key,
		Role:      a.Role,
		Glyph:     a.Glyph,
		F:         a.F,
		Desc:      a.Desc,
	}
	st.Captives = append(st.Captives, capRec)

	// Remove from live zone actors while marched in captivity.
	rest := g.Cur.Actors[:0]
	for _, act := range g.Cur.Actors {
		if act != a {
			rest = append(rest, act)
		}
	}
	g.Cur.Actors = rest
	g.spend(800)

	msg := fmt.Sprintf("You bind %s and take them captive (ransom value %d coin).", a.Name, ransom)
	g.Msg("{{Y|" + msg + "}}")
	g.ccLogf(fmt.Sprintf("Subdued %s (ransom %d).", a.Name, ransom))
	return true, msg
}

func (g *Game) ccFindCaptive(capID int) (int, *ccCaptive) {
	st := g.cc()
	for i, c := range st.Captives {
		if c.ID == capID {
			return i, c
		}
	}
	return -1, nil
}

// ccRansom hands over a captive to a faction stronghold/guard for gold and standing.
func (g *Game) ccRansom(capID int, buyer *Actor) (bool, int, string) {
	if g == nil || g.P == nil {
		return false, 0, "No active game."
	}
	idx, capRec := g.ccFindCaptive(capID)
	if capRec == nil {
		return false, 0, "Captive not found."
	}
	inStronghold := g.Cur != nil && (g.Cur.Kind == "town" || g.Cur.Burg > 0)
	if buyer == nil && !inStronghold {
		return false, 0, "Bring captives to a town, stronghold, or guard to collect ransom."
	}

	payout := capRec.Ransom
	repKey := capRec.Faction
	repGain := 12
	if buyer != nil {
		if buyer.Faction != "" && buyer.Faction == capRec.Faction {
			payout = payout * 5 / 4
			repGain = 18
		} else if buyer.State > 0 && buyer.State == capRec.State {
			payout = payout * 5 / 4
			repGain = 18
		}
		if repKey == "" && buyer.Faction != "" {
			repKey = buyer.Faction
		}
	}
	if repKey == "" && g.Cur != nil && g.Cur.Faction != "" {
		repKey = g.Cur.Faction
	}
	if repKey == "" && g.HomeState > 0 {
		repKey = fmt.Sprintf("state:%d", g.HomeState)
	}

	g.P.Gold += payout
	if repKey != "" {
		if g.Rep == nil {
			g.Rep = map[string]int{}
		}
		g.Rep[repKey] += repGain
	}

	st := g.cc()
	st.Captives = append(st.Captives[:idx], st.Captives[idx+1:]...)
	st.Ransomed++
	g.spend(400)

	who := "the local garrison"
	if buyer != nil && buyer.Name != "" {
		who = buyer.Name
	}
	msg := fmt.Sprintf("You ransom %s to %s for %d coin and +%d standing.", capRec.Name, who, payout, repGain)
	g.Msg("{{G|" + msg + "}}")
	g.ccLogf(fmt.Sprintf("Ransomed %s for %d coin.", capRec.Name, payout))
	return true, payout, msg
}

// ccRecruitOath swears a captive into service under oath as a loyal companion.
func (g *Game) ccRecruitOath(capID int) (bool, *Actor, string) {
	if g == nil || g.Cur == nil || g.P == nil {
		return false, nil, "No active zone."
	}
	idx, capRec := g.ccFindCaptive(capID)
	if capRec == nil {
		return false, nil, "Captive not found."
	}
	if g.companionCount() >= maxCompanions {
		return false, nil, fmt.Sprintf("Your party is full (%d/%d companions).", g.companionCount(), maxCompanions)
	}

	tpl := capRec.Tpl
	if tpl == "" {
		tpl = "bandit"
	}
	a := g.Spawn(tpl, capRec.Level, g.R)
	if a == nil {
		a = g.Spawn("bandit", capRec.Level, g.R)
	}
	if a == nil {
		return false, nil, "Could not muster captive into party."
	}
	a.ID = capRec.ID
	a.Name = capRec.Name
	a.Proper = true
	if capRec.Glyph != 0 {
		a.Glyph = capRec.Glyph
	}
	if capRec.F != 0 {
		a.F = capRec.F
	}
	if capRec.Desc != "" {
		a.Desc = capRec.Desc
	}
	a.MaxHP = maxi(capRec.MaxHP, a.MaxHP)
	a.HP = clampi(maxi(capRec.HP, a.MaxHP/2), 1, a.MaxHP)
	a.Angry = false
	a.Role = capRec.Role
	a.State = capRec.State
	a.X, a.Y = g.P.X, g.P.Y

	loyalty := clampi(50+g.P.M(EGO)*4+g.Skills["persuasion"]*6, 35, 95)
	a.Comp = &Companion{
		Faction: capRec.Faction,
		AI:      "hostile",
		F:       a.F,
		HomeKey: g.Cur.Key,
		HomeX:   g.P.X,
		HomeY:   g.P.Y,
		Town:    g.Cur.Name,
		Loyalty: loyalty,
		Since:   g.Turn,
	}
	a.AI = "follower"
	a.Order = OrdFollow
	a.Faction = "player"
	a.Set("oathbound")

	g.Cur.Actors = append(g.Cur.Actors, a)
	g.nudge(g.Cur, a)

	st := g.cc()
	st.Captives = append(st.Captives[:idx], st.Captives[idx+1:]...)
	st.Recruited++
	g.spend(600)

	msg := fmt.Sprintf("%s swears an oath of service and joins your band (loyalty %d).", a.Name, loyalty)
	g.Msg("{{C|" + msg + "}}")
	g.ccLogf(fmt.Sprintf("Recruited %s under oath.", a.Name))
	return true, a, msg
}

func (g *Game) ccReleaseCaptive(capID int) (bool, string) {
	idx, capRec := g.ccFindCaptive(capID)
	if capRec == nil {
		return false, "Captive not found."
	}
	st := g.cc()
	st.Captives = append(st.Captives[:idx], st.Captives[idx+1:]...)
	if capRec.Faction != "" {
		if g.Rep == nil {
			g.Rep = map[string]int{}
		}
		g.Rep[capRec.Faction] += 5
	}
	msg := fmt.Sprintf("You cut %s's bonds and let them go free.", capRec.Name)
	g.Msg("{{G|" + msg + "}}")
	g.ccLogf(fmt.Sprintf("Released %s.", capRec.Name))
	return true, msg
}

// ---- 3. Fortified Wilderness Camp Defense ----

// ccPitchCamp fortifies a wilderness camp with barricades, tripwires, watchfire, and sentry shift.
func (g *Game) ccPitchCamp() (bool, string) {
	if g == nil || g.Cur == nil || g.P == nil {
		return false, "No active zone."
	}
	z := g.Cur
	if z.Kind == "town" {
		return false, "No need to fortify a camp inside town walls."
	}
	if g.nearHostile() {
		return false, "You cannot fortify camp with hostiles nearby."
	}

	barricades := 0
	barPts := [][2]int{{0, -2}, {0, 2}, {-2, 0}, {2, 0}}
	for _, d := range barPts {
		bx, by := g.P.X+d[0], g.P.Y+d[1]
		if z.In(bx, by) && z.Walkable(bx, by) && z.ActorAt(bx, by) == nil {
			z.Set(bx, by, TCrate)
			if z.Notes == nil {
				z.Notes = map[int]string{}
			}
			z.Notes[by*MW+bx] = "Sharpened camp stakes and wooden barricades."
			barricades++
		}
	}

	tripwires := 0
	tripPts := [][2]int{{-2, -2}, {2, 2}, {-2, 2}, {2, -2}}
	for _, d := range tripPts {
		if tripwires >= 2 {
			break
		}
		tx, ty := g.P.X+d[0], g.P.Y+d[1]
		if z.In(tx, ty) && z.Walkable(tx, ty) && z.ActorAt(tx, ty) == nil {
			z.Set(tx, ty, TTrap)
			if z.Traps == nil {
				z.Traps = map[int]uint8{}
			}
			z.Traps[ty*MW+tx] = 1
			if z.Notes == nil {
				z.Notes = map[int]string{}
			}
			z.Notes[ty*MW+tx] = "A taut camp tripwire strung with warning bells."
			tripwires++
		}
	}

	watchfire := false
	firePts := [][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}}
	for _, d := range firePts {
		fx, fy := g.P.X+d[0], g.P.Y+d[1]
		if z.In(fx, fy) && z.Walkable(fx, fy) && z.ActorAt(fx, fy) == nil {
			z.Set(fx, fy, TBrazier)
			if len(z.Fire) == MW*MH {
				z.Fire[fy*MW+fx] = 12
			}
			if z.Notes == nil {
				z.Notes = map[int]string{}
			}
			z.Notes[fy*MW+fx] = "A crackling fortified camp watchfire."
			watchfire = true
			break
		}
	}

	sentryName := ""
	sentryShift := "first"
	if comps := g.localCompanions(); len(comps) > 0 {
		sentryName = comps[0].Name
	}

	defBonus := barricades + tripwires*2
	if watchfire {
		defBonus += 2
	}
	if sentryName != "" {
		defBonus += 3
	}

	camp := &ccCamp{
		ZoneKey:      z.Key,
		X:            g.P.X,
		Y:            g.P.Y,
		DayPitched:   g.Day(),
		Barricades:   barricades,
		Tripwires:    tripwires,
		Watchfire:    watchfire,
		SentryName:   sentryName,
		SentryShift:  sentryShift,
		DefenseBonus: defBonus,
		Active:       true,
	}
	g.cc().Camps[z.Key] = camp
	g.P.AddSt("fortified", 8)
	g.spend(1000)

	msg := fmt.Sprintf("Fortified camp pitched (%d barricades, %d tripwires, def +%d).", barricades, tripwires, defBonus)
	if sentryName != "" {
		msg = fmt.Sprintf("Fortified camp pitched (%d barricades, %d tripwires, sentry %s, def +%d).", barricades, tripwires, sentryName, defBonus)
	}
	g.Msg("{{G|" + msg + "}}")
	g.ccLogf(msg)
	return true, msg
}

func (g *Game) ccCycleSentryShift() (bool, string) {
	if g.Cur == nil {
		return false, "No active camp."
	}
	camp := g.cc().Camps[g.Cur.Key]
	if camp == nil || !camp.Active {
		return false, "Pitch a fortified camp here first."
	}
	shifts := []string{"first", "mid", "dawn"}
	next := "first"
	for i, s := range shifts {
		if s == camp.SentryShift {
			next = shifts[(i+1)%len(shifts)]
			break
		}
	}
	camp.SentryShift = next
	if comps := g.localCompanions(); len(comps) > 0 {
		camp.SentryName = comps[0].Name
	}
	msg := fmt.Sprintf("Sentry watch rotated to %s shift.", next)
	g.Msg("{{C|" + msg + "}}")
	return true, msg
}

// ccCampAmbush resolves a night ambush against the fortified camp in g.Cur.
// Returns (tripwireTriggered, raidersSpawned, summary).
func (g *Game) ccCampAmbush() (bool, int, string) {
	if g == nil || g.Cur == nil || g.P == nil {
		return false, 0, "No active zone."
	}
	z := g.Cur
	st := g.cc()
	camp := st.Camps[z.Key]
	if camp == nil || !camp.Active {
		return false, 0, "No fortified camp in this zone."
	}

	tripTriggered := false
	if camp.Tripwires > 0 {
		camp.Tripwires--
		tripTriggered = true
	}

	// Bonus defense for the player during the ambush.
	defTurns := 5 + camp.DefenseBonus/2
	g.P.AddSt("fortified", defTurns)

	count := 1 + g.R.Int(0, 1)
	spawned := 0
	edgePts := [][2]int{{-3, -3}, {3, -3}, {-3, 3}, {3, 3}, {0, -4}, {0, 4}}
	for i := 0; i < count; i++ {
		tpl := "bandit"
		if g.R.Chance(0.45) {
			tpl = "wolf"
		}
		raider := g.Spawn(tpl, maxi(z.Level, 1), g.R)
		if raider == nil {
			continue
		}
		raider.AI = "hostile"
		raider.Angry = true
		px, py := camp.X, camp.Y
		for _, off := range edgePts {
			tx, ty := camp.X+off[0], camp.Y+off[1]
			if z.In(tx, ty) && z.Walkable(tx, ty) && z.ActorAt(tx, ty) == nil {
				px, py = tx, ty
				break
			}
		}
		raider.X, raider.Y = px, py
		if tripTriggered {
			raider.AddSt("dazed", 3)
			g.hurt(raider, 4+camp.DefenseBonus/2, g.P, "camp tripwire and stakes")
		}
		if camp.SentryName != "" && raider.HP > 0 {
			g.hurt(raider, 3+g.R.Int(1, 3), g.P, "sentry warning volley")
		}
		if raider.HP > 0 {
			z.Actors = append(z.Actors, raider)
		}
		spawned++
	}

	camp.LastAmbush = g.Day() + 1
	st.AmbushesRepelled++

	var msg string
	if tripTriggered {
		msg = fmt.Sprintf("Night ambush! Tripwire bells snap, dazing %d raider(s) as your camp defenses hold (+%d def).", spawned, camp.DefenseBonus)
		g.Msg("{{Y|" + msg + "}}")
	} else {
		msg = fmt.Sprintf("Night ambush! %d raider(s) probe your camp perimeter (+%d def).", spawned, camp.DefenseBonus)
		g.Msg("{{R|" + msg + "}}")
	}
	g.ccLogf(msg)
	return tripTriggered, spawned, msg
}

// ---- hooks ----

func ccTurn(g *Game) {
	if g == nil || g.Dead {
		return
	}
	st := g.cc()
	if len(st.SmokeTiles) > 0 {
		for k, n := range st.SmokeTiles {
			if n <= 1 {
				delete(st.SmokeTiles, k)
			} else {
				st.SmokeTiles[k] = n - 1
			}
		}
	}
	// Night ambush check while sleeping/waiting at an active wilderness camp.
	if g.Cur != nil && !g.OnWorld {
		if camp := st.Camps[g.Cur.Key]; camp != nil && camp.Active {
			h := g.Hour()
			if (h >= 22 || h <= 4) && camp.LastAmbush != g.Day()+1 && g.Turn%60 == 0 {
				// Deterministic low-frequency check using side RNG so normal turns stay calm.
				sr := NewRng(fmt.Sprintf("%s:cc:amb:%s:%d:%d", g.Seed, g.Cur.Key, g.Day(), h))
				if sr.Chance(0.08) {
					g.ccCampAmbush()
				}
			}
		}
	}
}

func ccEnter(g *Game, zn *Zone) {
	if g == nil || zn == nil {
		return
	}
	if camp := g.cc().Camps[zn.Key]; camp != nil && camp.Active {
		g.Msg(fmt.Sprintf("{{G|Your fortified camp still stands here (%d barricades, %d tripwires).}}", camp.Barricades, camp.Tripwires))
	}
}

func ccTalk(g *Game, a *Actor) []Option {
	if g == nil || a == nil {
		return nil
	}
	var out []Option
	st := g.cc()

	// Ransom captives to guards, captains, elders, governors, or town officials.
	canTakeRansom := a.AI == "guard" || a.Role == "guard" || a.Role == "captain" || a.Role == "elder" || a.Role == "governor" || a.Role == "warden" || (g.Cur != nil && g.Cur.Kind == "town" && a.AI == "town")
	if canTakeRansom && len(st.Captives) > 0 {
		for _, capRec := range st.Captives {
			c := capRec
			out = append(out, Option{
				Text: fmt.Sprintf("Hand over captive %s for ransom (%d coin)", c.Name, c.Ransom),
				Do: func() {
					g.ccRansom(c.ID, a)
					g.Mode = ""
				},
			})
		}
	}

	// Parley: demand surrender from a weakened foe.
	if (a.AI == "hostile" || a.Angry) && ccCanSubdue(a) {
		out = append(out, Option{
			Text: fmt.Sprintf("Demand %s lay down their arms and surrender", a.Name),
			Do: func() {
				g.ccSubdue(a)
				g.Mode = ""
			},
		})
	}
	return out
}

func ccSide(g *Game) []string {
	if g == nil {
		return nil
	}
	st := g.cc()
	var lines []string
	if len(st.Captives) > 0 {
		lines = append(lines, fmt.Sprintf("{{Y|Captives: %d}}", len(st.Captives)))
	}
	if g.Cur != nil {
		if camp := st.Camps[g.Cur.Key]; camp != nil && camp.Active {
			lines = append(lines, fmt.Sprintf("{{G|Camp: def +%d (tw %d)}}", camp.DefenseBonus, camp.Tripwires))
		}
	}
	return lines
}

// ---- interactive mode "cc" screen ----

func ccKeys(g *Game, k string) {
	st := g.cc()
	switch k {
	case "Escape", "q":
		g.Mode = ""
		return
	case "Tab":
		st.tab = (st.tab + 1) % 3
		st.sel = 0
		return
	case "1":
		st.tab, st.sel = 0, 0
		return
	case "2":
		st.tab, st.sel = 1, 0
		return
	case "3":
		st.tab, st.sel = 2, 0
		return
	case "Up", "k":
		if st.sel > 0 {
			st.sel--
		}
		return
	case "Down", "j":
		st.sel++
		return
	}

	switch st.tab {
	case 0: // Combos & Field Craft
		foe := g.ccNearestFoe(4)
		switch k {
		case "w":
			g.ccThrowPrimer("water", foe)
		case "o":
			g.ccThrowPrimer("oil", foe)
		case "f":
			g.ccThrowPrimer("frost", foe)
		case "s":
			g.ccThrowPrimer("smoke", foe)
		case "Enter", " ":
			if foe == nil {
				g.Msg("No hostile foe within 4 tiles to strike with a combo.")
				return
			}
			switch st.sel % 4 {
			case 0:
				ccApplyElementalCombo(g, foe, "shock", "")
			case 1:
				ccApplyElementalCombo(g, foe, "fire", "")
			case 2:
				ccApplyElementalCombo(g, foe, "", "cudgel")
			case 3:
				ccApplyElementalCombo(g, foe, "", "shortblade")
			}
		}

	case 1: // Captives & Ransom
		if k == "s" {
			if foe := g.ccNearestFoe(2); foe != nil {
				g.ccSubdue(foe)
			} else {
				g.Msg("No weakened hostile adjacent to subdue.")
			}
			return
		}
		if len(st.Captives) == 0 {
			return
		}
		if st.sel >= len(st.Captives) {
			st.sel = len(st.Captives) - 1
		}
		capRec := st.Captives[st.sel]
		switch k {
		case "r", "Enter":
			g.ccRansom(capRec.ID, nil)
		case "o":
			g.ccRecruitOath(capRec.ID)
		case "x":
			g.ccReleaseCaptive(capRec.ID)
		}

	case 2: // Fortified Camp
		switch k {
		case "p", "Enter":
			g.ccPitchCamp()
		case "s":
			g.ccCycleSentryShift()
		case "a":
			g.ccCampAmbush()
		}
	}
}

func (g *Game) ccDraw() {
	s := g.sc
	st := g.cc()
	x0, y0, w, h := 4, 2, SW-8, SH-4
	s.Box(x0, y0, w, h, CCyan, "Tactics, Captives & Camp")

	tabs := []string{"1: Combos & Field Craft", "2: Captives & Ransom", "3: Fortified Camp"}
	tx := x0 + 2
	for i, label := range tabs {
		col := CGray
		bg := CBgDark
		if i == st.tab {
			col = CYellow
			bg = CBgSel
		}
		s.TextB(tx, y0+1, " "+label+" ", col, bg, x0+w-2)
		tx += len(label) + 4
	}

	x, y := x0+2, y0+3
	maxY := y0 + h - 2

	switch st.tab {
	case 0:
		s.TextB(x, y, fmt.Sprintf("{{W|Elemental Combos Triggered: %d}}", st.Combos), CGray, CBgDark, x0+w-2)
		y += 2
		combos := []struct {
			id, title, desc string
		}{
			{"wet_shock", "Wet + Shock (Chain Lightning)", "Arcs lightning to adjacent wet or standing-in-water foes."},
			{"oil_fire", "Oil + Fire (Flash Ignition)", "Ignites spreading fire tiles and inflicts heavy burning."},
			{"frost_shatter", "Frost + Heavy Bludgeon (Shatter)", "Shatters frozen foes with bonus shard damage + AoE shrapnel."},
			{"smoke_backstab", "Smoke + Shortblade/Stealth (Backstab)", "Guaranteed critical strike with deep bleeding."},
		}
		if st.sel >= len(combos) {
			st.sel = len(combos) - 1
		}
		for i, c := range combos {
			bg := CBgDark
			if i == st.sel {
				bg = CBgSel
			}
			s.Fill(x, y, w-4, 1, bg)
			s.TextB(x, y, fmt.Sprintf("{{Y|%s}} {{K|(x%d)}} - %s", c.title, st.ComboCounts[c.id], c.desc), CGray, bg, x0+w-2)
			y++
		}
		y++
		if foe := g.ccNearestFoe(4); foe != nil {
			s.TextB(x, y, fmt.Sprintf("Nearest hostile: {{R|%s}} (%d/%d HP, dist %d)", foe.Name, foe.HP, foe.MaxHP, dist(g.P, foe)), CGray, CBgDark, x0+w-2)
		} else {
			s.TextB(x, y, "{{K|No hostile within 4 tiles.}}", CGray, CBgDark, x0+w-2)
		}
		y += 2
		s.TextB(x, y, "{{C|Field Craft Primers:}} [w] Water  [o] Oil  [f] Frost  [s] Smoke  [Enter] Trigger selected combo", CGray, CBgDark, x0+w-2)

	case 1:
		s.TextB(x, y, fmt.Sprintf("{{W|Captives in Tow: %d/6}}   Ransomed: %d   Oathbound: %d", len(st.Captives), st.Ransomed, st.Recruited), CGray, CBgDark, x0+w-2)
		y += 2
		if len(st.Captives) == 0 {
			s.TextB(x, y, "{{K|No captives currently bound. Weaken a foe below 40% HP and press [s] to subdue.}}", CGray, CBgDark, x0+w-2)
			y += 2
		} else {
			if st.sel >= len(st.Captives) {
				st.sel = len(st.Captives) - 1
			}
			for i, c := range st.Captives {
				if y >= maxY-3 {
					break
				}
				bg := CBgDark
				if i == st.sel {
					bg = CBgSel
				}
				s.Fill(x, y, w-4, 1, bg)
				s.TextB(x, y, fmt.Sprintf("{{Y|%s}} (Lv %d, %s)  Ransom: {{G|%d coin}}", c.Name, c.Level, c.Faction, c.Ransom), CGray, bg, x0+w-2)
				y++
			}
			y++
		}
		s.TextB(x, y, "{{C|Actions:}} [s] Subdue nearby weakened foe  [r] Ransom in town  [o] Recruit under oath  [x] Release", CGray, CBgDark, x0+w-2)

	case 2:
		s.TextB(x, y, fmt.Sprintf("{{W|Fortified Wilderness Camp}}   Ambushes Repelled: %d", st.AmbushesRepelled), CGray, CBgDark, x0+w-2)
		y += 2
		var camp *ccCamp
		if g.Cur != nil {
			camp = st.Camps[g.Cur.Key]
		}
		if camp == nil || !camp.Active {
			s.TextB(x, y, "{{K|No fortified camp in this zone. Press [p] outdoors to build barricades, tripwires, and watchfire.}}", CGray, CBgDark, x0+w-2)
			y += 2
		} else {
			sentry := camp.SentryName
			if sentry == "" {
				sentry = "None (solo watch)"
			}
			s.TextB(x, y, fmt.Sprintf("Barricades: {{G|%d}}   Tripwires: {{Y|%d}}   Watchfire: %v   Defense Bonus: {{C|+%d}}", camp.Barricades, camp.Tripwires, camp.Watchfire, camp.DefenseBonus), CGray, CBgDark, x0+w-2)
			y++
			s.TextB(x, y, fmt.Sprintf("Sentry: {{Y|%s}}   Watch Shift: {{C|%s}}", sentry, camp.SentryShift), CGray, CBgDark, x0+w-2)
			y += 2
		}
		if len(st.Camps) > 0 && y < maxY-4 {
			keys := make([]string, 0, len(st.Camps))
			for k := range st.Camps {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			s.TextB(x, y, fmt.Sprintf("{{K|Total fortified camps on planet: %d}}", len(keys)), CGray, CBgDark, x0+w-2)
			y += 2
		}
		s.TextB(x, y, "{{C|Actions:}} [p] Pitch fortified camp  [s] Rotate sentry shift  [a] Resolve night ambush", CGray, CBgDark, x0+w-2)
	}

	s.TextB(x0+2, y0+h-1, "{{K|Tab/1-3 switch tab  Up/Down select  Enter act  Esc close}}", CGray, CBgDark, x0+w-2)
}
