package engine

import (
	"fmt"
	"sort"
	"strings"
)

// smugglers_exped.go (prefix sm):
// 1. Underground & Coastal Smuggler Networks: deterministic tide-locked sea-caves / coastal
//    grottoes (low-tide access) and forgotten cellar tunnels linking town districts past locked gates.
// 2. Expedition Logistics & Porter Trains: hire pack-mules, porters, and a cartographer; track
//    rations, wheel/gear wear, morale, and camp fever across multi-day journeys.
// 3. Living Landmarks & Moving Points of Interest: deterministic Wandering Night-Market
//    (new-moon days), Migrating Floating Island (3-day drift), and Drifting Ghost-Ship (fog/night).
//
// Saved state: g.ext("sm"), planet-local ("sm" in unLocalKeys).
// Screen: mode "sm", palette entry "Smugglers, expeditions & sights".

const smExtKey = "sm"

type smPassage struct {
	ID           string `json:"id"`
	Kind         string `json:"k"` // sea_cave | cellar_tunnel
	Name         string `json:"n"`
	FromKey      string `json:"fk"`
	ToKey        string `json:"tk,omitempty"`
	BurgID       int    `json:"bg,omitempty"`
	DistrictFrom string `json:"df,omitempty"`
	DistrictTo   string `json:"dt,omitempty"`
	WX           int    `json:"wx"`
	WY           int    `json:"wy"`
	X            int    `json:"x"`
	Y            int    `json:"y"`
	TideLocked   bool   `json:"tl,omitempty"`
	Discovered   bool   `json:"dc,omitempty"`
	Looted       bool   `json:"lt,omitempty"`
}

type smExpedition struct {
	Active       bool   `json:"ac"`
	Name         string `json:"n"`
	DestP        int    `json:"dp"`
	DestName     string `json:"dn"`
	StartDay     int    `json:"sd"`
	LastTickDay  int    `json:"ld"`
	Mules        int    `json:"mu"`
	Porters      int    `json:"po"`
	Cartographer bool   `json:"cg"`
	Rations      int    `json:"ra"`
	MaxRations   int    `json:"mr"`
	GearWear     int    `json:"gw"` // 0 pristine .. 100 broken
	Morale       int    `json:"mo"` // 0..100
	Fever        int    `json:"fv"` // 0 healthy .. 10 severe camp fever
	DaysOut      int    `json:"do"`
	UpkeepPaid   int    `json:"up"`
}

type smPOI struct {
	ID         string `json:"id"`
	Kind       string `json:"k"` // night_market | floating_island | ghost_ship
	Name       string `json:"n"`
	WX         int    `json:"wx"`
	WY         int    `json:"wy"`
	P          int    `json:"p"`
	Active     bool   `json:"ac"`
	Window     string `json:"wi"`
	Desc       string `json:"ds"`
	ExpiresDay int    `json:"ed"`
}

type smState struct {
	Passages    map[string]*smPassage `json:"ps,omitempty"`
	TunnelsUsed int                   `json:"tu,omitempty"`
	Exped       *smExpedition         `json:"ex,omitempty"`
	VisitedPOIs map[string]int        `json:"vp,omitempty"` // poiKind:bucket -> day visited
	Log         []string              `json:"lg,omitempty"`

	tab int
	sel int
}

func (g *Game) sm() *smState {
	st := g.ext(smExtKey, func() any {
		return &smState{
			Passages:    map[string]*smPassage{},
			VisitedPOIs: map[string]int{},
		}
	}).(*smState)
	if st.Passages == nil {
		st.Passages = map[string]*smPassage{}
	}
	if st.VisitedPOIs == nil {
		st.VisitedPOIs = map[string]int{}
	}
	return st
}

func init() {
	unLocalKeys = append(unLocalKeys, smExtKey)
	turnHooks = append(turnHooks, smTurn)
	zoneHooks = append(zoneHooks, smZone)
	enterHooks = append(enterHooks, smEnter)
	talkHooks = append(talkHooks, smTalk)
	sideHooks = append(sideHooks, smSide)
	modeKeys["sm"] = smKeys
	modeDraws["sm"] = func(g *Game) { g.smDraw() }
	paEntries = append(paEntries, paEntry{
		Name:     "Smugglers, expeditions & sights",
		Keywords: "smuggler tunnel cellar sea cave tide expedition porter mule cartographer rations night market floating island ghost ship poi",
		Cat:      "Screen",
		Label:    "screen",
		Func:     func(g *Game) { g.smOpen() },
	})
}

func (g *Game) smOpen() {
	st := g.sm()
	st.tab, st.sel = 0, 0
	g.Mode = "sm"
}

func (g *Game) smLogf(s string) {
	st := g.sm()
	st.Log = append(st.Log, fmt.Sprintf("Day %d: %s", g.Day()+1, s))
	if len(st.Log) > 24 {
		st.Log = st.Log[len(st.Log)-24:]
	}
}

// ---- 1. Underground & Coastal Smuggler Networks ----

// smLowTide reports whether the tide is currently low enough to enter coastal sea-caves.
// Low-tide windows occur twice per day: 02:00-06:59 and 14:00-18:59.
func smLowTide(g *Game) bool {
	if g == nil {
		return false
	}
	h := g.Hour()
	return (h >= 2 && h <= 6) || (h >= 14 && h <= 18)
}

func (w *WorldData) smIsCoastal(wx, wy int) bool {
	if w == nil || !w.In(wx, wy) {
		return false
	}
	p := w.Idx(wx, wy)
	if w.IsWater(p) {
		return false
	}
	for dy := -1; dy <= 1; dy++ {
		for dx := -1; dx <= 1; dx++ {
			if dx == 0 && dy == 0 {
				continue
			}
			nx, ny := wx+dx, wy+dy
			if w.In(nx, ny) && w.IsWater(w.Idx(nx, ny)) {
				return true
			}
		}
	}
	return false
}

func smZone(g *Game, zn *Zone, r *Rng) {
	if g == nil || zn == nil || zn.Z != 0 {
		return
	}
	sr := NewRng(fmt.Sprintf("%s:sm:zone:%s", g.Seed, zn.Key))
	st := g.sm()

	// Forgotten cellar tunnels in towns linking districts so fugitives can bypass locked gates.
	if zn.Kind == "town" {
		pid := "cellar:" + zn.Key
		if _, exists := st.Passages[pid]; !exists {
			tx, ty := 14+sr.Int(0, 12), 6+sr.Int(0, 10)
			if !zn.In(tx, ty) || !zn.Walkable(tx, ty) {
				if fx, fy, ok := zn.FreeSpot(sr, nil); ok {
					tx, ty = fx, fy
				}
			}
			if zn.In(tx, ty) && zn.Walkable(tx, ty) {
				zn.Set(tx, ty, TDown)
				if zn.Notes == nil {
					zn.Notes = map[int]string{}
				}
				zn.Notes[ty*MW+tx] = "A forgotten smuggler cellar hatch linking town districts beneath the gates."
				districts := []string{"Dockside Cellar", "Old Market Vault", "Tannery Drain", "Bell-Tower Undercroft"}
				dFrom := districts[sr.Int(0, len(districts)-1)]
				dTo := districts[(sr.Int(1, len(districts)-1)+sr.Int(0, 2))%len(districts)]
				if dTo == dFrom {
					dTo = "Outskirts Gate-Drain"
				}
				townName := zn.Name
				if g.W != nil && zn.Burg > 0 {
					if bd := g.W.burgByID[zn.Burg]; bd != nil {
						townName = bd.Name
					}
				}
				st.Passages[pid] = &smPassage{
					ID:           pid,
					Kind:         "cellar_tunnel",
					Name:         fmt.Sprintf("%s Cellar Tunnel", townName),
					FromKey:      zn.Key,
					BurgID:       zn.Burg,
					DistrictFrom: dFrom,
					DistrictTo:   dTo,
					WX:           zn.WX,
					WY:           zn.WY,
					X:            tx,
					Y:            ty,
					TideLocked:   false,
					Discovered:   true,
				}
			}
		}
	}

	// Deterministic tide-locked sea-caves / coastal grottoes along coastal parasangs.
	coastal := g.W != nil && g.W.smIsCoastal(zn.WX, zn.WY)
	if !coastal && g.W != nil && zn.Burg > 0 {
		if bd := g.W.burgByID[zn.Burg]; bd != nil && bd.Port {
			coastal = true
		}
	}
	if coastal && sr.Chance(0.65) {
		pid := "seacave:" + zn.Key
		if _, exists := st.Passages[pid]; !exists {
			gx, gy := 5+sr.Int(0, 10), 4+sr.Int(0, MH-9)
			if !zn.In(gx, gy) || !zn.Walkable(gx, gy) {
				if fx, fy, ok := zn.FreeSpot(sr, nil); ok {
					gx, gy = fx, fy
				}
			}
			if zn.In(gx, gy) && zn.Walkable(gx, gy) {
				zn.Set(gx, gy, TDown)
				if zn.Notes == nil {
					zn.Notes = map[int]string{}
				}
				zn.Notes[gy*MW+gx] = "A tide-locked coastal sea-cave grotto, dry only during low-tide hours."
				grottoNames := []string{"Salt-Arch Grotto", "Barnacle Smuggler Cove", "Drowned Lantern Cave", "Kelp-Mouth Hollow"}
				st.Passages[pid] = &smPassage{
					ID:         pid,
					Kind:       "sea_cave",
					Name:       grottoNames[sr.Int(0, len(grottoNames)-1)],
					FromKey:    zn.Key,
					BurgID:     zn.Burg,
					WX:         zn.WX,
					WY:         zn.WY,
					X:          gx,
					Y:          gy,
					TideLocked: true,
					Discovered: true,
				}
			}
		}
	}
}

// smUseTunnel slips through a cellar tunnel (bypassing locked city gates) or a tide-locked sea-cave.
func (g *Game) smUseTunnel(passageID string) (bool, string) {
	if g == nil || g.P == nil {
		return false, "No active game."
	}
	st := g.sm()
	ps := st.Passages[passageID]
	if ps == nil {
		return false, "Unknown smuggler passage."
	}
	if ps.TideLocked && !smLowTide(g) {
		msg := fmt.Sprintf("%s is flooded at high tide (hour %02d:00). Return at low tide (02:00-06:00 or 14:00-18:00).", ps.Name, g.Hour())
		g.Msg("{{B|" + msg + "}}")
		return false, msg
	}

	ps.Discovered = true
	st.TunnelsUsed++
	g.spend(400)

	if ps.Kind == "cellar_tunnel" {
		// Relocate player to the far side of the zone (slipping past locked city gates).
		if g.Cur != nil {
			destX := MW - 4
			if g.P.X > MW/2 {
				destX = 3
			}
			destY := clampi(MH-1-g.P.Y, 2, MH-3)
			if fx, fy, ok := g.nearestFree(g.Cur, destX, destY); ok {
				g.P.X, g.P.Y = fx, fy
			}
		}
		if g.P.Status != nil {
			delete(g.P.Status, "pursued")
			delete(g.P.Status, "spotted")
		}
		msg := fmt.Sprintf("You slip through %s (%s -> %s), bypassing the locked city gates.", ps.Name, ps.DistrictFrom, ps.DistrictTo)
		g.Msg("{{C|" + msg + "}}")
		g.smLogf(msg)
		return true, msg
	}

	// Tide-locked sea-cave grotto: dry at low tide; first visit yields a smuggler cache.
	if !ps.Looted {
		ps.Looted = true
		sr := NewRng(fmt.Sprintf("%s:sm:cache:%s", g.Seed, ps.ID))
		coin := 25 + sr.Int(10, 45)
		g.P.Gold += coin
		msg := fmt.Sprintf("At low tide you wade into %s and recover a hidden smuggler cache (%d coin).", ps.Name, coin)
		g.Msg("{{Y|" + msg + "}}")
		g.smLogf(msg)
		return true, msg
	}

	msg := fmt.Sprintf("You slip through %s while the tide stays out.", ps.Name)
	g.Msg("{{C|" + msg + "}}")
	g.smLogf(msg)
	return true, msg
}

func (g *Game) smPassageList() []*smPassage {
	st := g.sm()
	out := make([]*smPassage, 0, len(st.Passages))
	for _, p := range st.Passages {
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// ---- 2. Expedition Logistics & Porter Trains ----

// smStartExpedition hires pack-mules, porters, and an optional cartographer for a multi-day journey.
func (g *Game) smStartExpedition(mules, porters int, cartographer bool, destP int) (bool, string) {
	if g == nil || g.P == nil {
		return false, "No active game."
	}
	st := g.sm()
	if st.Exped != nil && st.Exped.Active {
		return false, "An expedition train is already active. Disband it before hiring another."
	}
	mules = clampi(mules, 0, 6)
	porters = clampi(porters, 0, 8)
	if mules == 0 && porters == 0 {
		return false, "Hire at least one pack-mule or porter."
	}

	cost := mules*18 + porters*12
	if cartographer {
		cost += 35
	}
	if g.P.Gold < cost {
		return false, fmt.Sprintf("Hiring this expedition train costs %d coin (you have %d).", cost, g.P.Gold)
	}
	g.P.Gold -= cost

	maxRat := 12 + mules*8 + porters*4
	destName := "the deep frontier"
	if g.W != nil && destP >= 0 && destP < g.W.W*g.W.H {
		destName = g.wsPlace(destP)
	}

	st.Exped = &smExpedition{
		Active:       true,
		Name:         fmt.Sprintf("Train to %s", destName),
		DestP:        destP,
		DestName:     destName,
		StartDay:     g.Day(),
		LastTickDay:  g.Day(),
		Mules:        mules,
		Porters:      porters,
		Cartographer: cartographer,
		Rations:      maxRat,
		MaxRations:   maxRat,
		GearWear:     0,
		Morale:       80,
		Fever:        0,
		DaysOut:      0,
		UpkeepPaid:   cost,
	}

	if cartographer && g.W != nil {
		g.smChartNearby(2)
	}
	g.spend(600)

	msg := fmt.Sprintf("Expedition assembled (%d mules, %d porters, cartographer: %v) for %d coin. Rations: %d.", mules, porters, cartographer, cost, maxRat)
	g.Msg("{{G|" + msg + "}}")
	g.smLogf(msg)
	return true, msg
}

func (g *Game) smChartNearby(radius int) {
	if g == nil || g.W == nil {
		return
	}
	if g.Known == nil {
		g.Known = map[int]bool{}
	}
	for dy := -radius; dy <= radius; dy++ {
		for dx := -radius; dx <= radius; dx++ {
			nx, ny := g.WX+dx, g.WY+dy
			if g.W.In(nx, ny) {
				g.Known[g.W.Idx(nx, ny)] = true
			}
		}
	}
}

// smExpeditionTick advances expedition logistics by one day: rations, gear wear, morale, and camp fever.
func (g *Game) smExpeditionTick() (string, bool) {
	if g == nil {
		return "No active game.", false
	}
	ex := g.sm().Exped
	if ex == nil || !ex.Active {
		return "No active expedition.", false
	}

	ex.DaysOut++
	ex.LastTickDay = g.Day()

	// Rations consumption.
	need := 1 + (ex.Porters+1)/2 + ex.Mules/2
	starving := false
	if ex.Rations >= need {
		ex.Rations -= need
	} else {
		ex.Rations = 0
		starving = true
	}

	// Terrain harshness (polar/tundra/glacier/desert/swamp increase wear & fever risk).
	harsh := false
	if g.W != nil && g.W.In(g.WX, g.WY) {
		b := g.W.biomeAt(g.W.Idx(g.WX, g.WY))
		if b == 1 || b == 2 || b == 9 || b == 10 || b == 11 || b == 12 {
			harsh = true
		}
	}

	wear := 4 + g.R.Int(1, 4)
	if harsh {
		wear += 3
	}
	if ex.Cartographer {
		wear = maxi(2, wear-2)
		g.smChartNearby(1)
	}
	ex.GearWear = clampi(ex.GearWear+wear, 0, 100)

	moraleDelta := 0
	if starving {
		moraleDelta -= 14
	}
	if ex.GearWear >= 70 {
		moraleDelta -= 6
	}
	if harsh && g.R.Chance(0.35) || starving && g.R.Chance(0.5) {
		ex.Fever = clampi(ex.Fever+1, 0, 10)
	}
	if ex.Fever > 0 {
		moraleDelta -= 5 + ex.Fever
	} else if !starving && ex.GearWear < 50 {
		moraleDelta += 2
	}
	ex.Morale = clampi(ex.Morale+moraleDelta, 0, 100)

	summary := fmt.Sprintf("Expedition day %d: rations %d/%d, wear %d%%, morale %d%%, fever %d.",
		ex.DaysOut, ex.Rations, ex.MaxRations, ex.GearWear, ex.Morale, ex.Fever)

	if ex.Morale <= 0 {
		ex.Active = false
		summary += " Morale collapsed and the porter train deserted."
		g.Msg("{{R|" + summary + "}}")
		g.smLogf(summary)
		return summary, false
	}

	g.Msg("{{c|" + summary + "}}")
	g.smLogf(summary)
	return summary, true
}

// smRepairWheels repairs axle/wheel/harness wear on the active expedition train.
func (g *Game) smRepairWheels() (bool, string) {
	if g == nil || g.P == nil {
		return false, "No active game."
	}
	ex := g.sm().Exped
	if ex == nil || !ex.Active {
		return false, "No active expedition train."
	}
	if ex.GearWear == 0 {
		return false, "Wheels and harnesses are already in good order."
	}
	cost := 8
	if g.P.Gold < cost {
		return false, fmt.Sprintf("Repairing wheels and harnesses requires %d coin.", cost)
	}
	g.P.Gold -= cost
	fixed := 35 + g.R.Int(0, 10)
	ex.GearWear = clampi(ex.GearWear-fixed, 0, 100)
	ex.Morale = clampi(ex.Morale+5, 0, 100)
	g.spend(600)

	msg := fmt.Sprintf("Wheels and pack harnesses repaired (wear now %d%%, morale %d%%).", ex.GearWear, ex.Morale)
	g.Msg("{{G|" + msg + "}}")
	g.smLogf(msg)
	return true, msg
}

// smForageRations gathers field rations for the porter train.
func (g *Game) smForageRations() (bool, int, string) {
	if g == nil || g.P == nil {
		return false, 0, "No active game."
	}
	ex := g.sm().Exped
	if ex == nil || !ex.Active {
		return false, 0, "No active expedition train."
	}
	gain := 4 + g.R.Int(2, 6) + ex.Porters
	before := ex.Rations
	ex.Rations = clampi(ex.Rations+gain, 0, ex.MaxRations)
	added := ex.Rations - before
	ex.Morale = clampi(ex.Morale+4, 0, 100)
	g.spend(700)

	msg := fmt.Sprintf("Porters forage +%d rations (%d/%d).", added, ex.Rations, ex.MaxRations)
	g.Msg("{{G|" + msg + "}}")
	g.smLogf(msg)
	return true, added, msg
}

func (g *Game) smTreatFever() (bool, string) {
	if g == nil || g.P == nil {
		return false, "No active game."
	}
	ex := g.sm().Exped
	if ex == nil || !ex.Active {
		return false, "No active expedition train."
	}
	if ex.Fever <= 0 {
		return false, "No camp fever in the train."
	}
	ex.Fever = clampi(ex.Fever-3, 0, 10)
	ex.Morale = clampi(ex.Morale+8, 0, 100)
	g.spend(600)
	msg := fmt.Sprintf("Camp fever treated with rest and bitter herbs (fever %d, morale %d%%).", ex.Fever, ex.Morale)
	g.Msg("{{G|" + msg + "}}")
	g.smLogf(msg)
	return true, msg
}

func (g *Game) smDisbandExpedition() (bool, string) {
	ex := g.sm().Exped
	if ex == nil || !ex.Active {
		return false, "No active expedition train."
	}
	ex.Active = false
	msg := "You pay off the porters and disband the expedition train."
	g.Msg("{{Y|" + msg + "}}")
	g.smLogf(msg)
	return true, msg
}

// ---- 3. Living Landmarks & Moving Points of Interest ----

// smMoonPhase returns the deterministic 0..27 lunar day (0..2 = new moon, 13..15 = full moon).
func smMoonPhase(day int) int {
	return ((day % 28) + 28) % 28
}

func smMoonName(day int) string {
	ph := smMoonPhase(day)
	switch {
	case ph <= 2:
		return "New Moon"
	case ph <= 6:
		return "Waxing Crescent"
	case ph <= 12:
		return "First Quarter"
	case ph <= 15:
		return "Full Moon"
	case ph <= 21:
		return "Last Quarter"
	default:
		return "Waning Crescent"
	}
}

func (g *Game) smPickLandP(sr *Rng, coastalOnly bool) (int, int, int) {
	if g == nil || g.W == nil || g.W.W == 0 || g.W.H == 0 {
		return 0, 0, 0
	}
	w := g.W
	var cands []int
	for y := 0; y < w.H; y++ {
		for x := 0; x < w.W; x++ {
			p := w.Idx(x, y)
			if w.IsWater(p) {
				continue
			}
			if coastalOnly && !w.smIsCoastal(x, y) {
				continue
			}
			cands = append(cands, p)
		}
	}
	if len(cands) == 0 {
		for y := 0; y < w.H; y++ {
			for x := 0; x < w.W; x++ {
				p := w.Idx(x, y)
				if !w.IsWater(p) {
					cands = append(cands, p)
				}
			}
		}
	}
	if len(cands) == 0 {
		return g.WX, g.WY, w.Idx(g.WX, g.WY)
	}
	p := cands[sr.Int(0, len(cands)-1)]
	return p % w.W, p / w.W, p
}

// smMovingPOIs computes all 3 living/moving world landmarks deterministically from day/moon/weather.
func (g *Game) smMovingPOIs() []smPOI {
	if g == nil || g.W == nil {
		return nil
	}
	day := g.Day()
	lunarCycle := day / 28
	ph := smMoonPhase(day)

	// 1. Wandering Night-Market: appears on new-moon days (phase 0..2), rare goods.
	nmRng := NewRng(fmt.Sprintf("%s:sm:poi:nightmarket:%d", g.Seed, lunarCycle))
	nmx, nmy, nmp := g.smPickLandP(nmRng, false)
	nmActive := ph <= 2
	nm := smPOI{
		ID:         fmt.Sprintf("night_market:%d", lunarCycle),
		Kind:       "night_market",
		Name:       "Wandering Night-Market",
		WX:         nmx,
		WY:         nmy,
		P:          nmp,
		Active:     nmActive,
		Window:     "New-moon days (lunar day 0-2)",
		Desc:       "Lantern-hung stalls selling rare curios, spices, and forbidden charts.",
		ExpiresDay: lunarCycle*28 + 3,
	}

	// 2. Migrating Floating Island: drifts across regions for 3 days at a time.
	fiBucket := day / 3
	fiRng := NewRng(fmt.Sprintf("%s:sm:poi:floatisland:%d", g.Seed, fiBucket))
	fix, fiy, fip := g.smPickLandP(fiRng, false)
	fi := smPOI{
		ID:         fmt.Sprintf("floating_island:%d", fiBucket),
		Kind:       "floating_island",
		Name:       "Migrating Floating Island",
		WX:         fix,
		WY:         fiy,
		P:          fip,
		Active:     true,
		Window:     fmt.Sprintf("Drifts every 3 days (until day %d)", (fiBucket+1)*3+1),
		Desc:       "A shadowed spire of airborne stone trailing hanging ropes and sky-crystals.",
		ExpiresDay: (fiBucket + 1) * 3,
	}

	// 3. Drifting Ghost-Ship: boardable in coastal fog or at night.
	gsRng := NewRng(fmt.Sprintf("%s:sm:poi:ghostship:%d", g.Seed, day))
	gsx, gsy, gsp := g.smPickLandP(gsRng, true)
	isNight := g.Hour() >= 20 || g.Hour() < 5
	isFog := false
	wx := g.WeatherHere()
	if wx.Kind != nil && (wx.Kind.ID == "fog" || wx.Kind.ID == "mist" || wx.Kind.ID == "miasma") {
		isFog = true
	}
	gs := smPOI{
		ID:         fmt.Sprintf("ghost_ship:%d", day),
		Kind:       "ghost_ship",
		Name:       "Drifting Ghost-Ship",
		WX:         gsx,
		WY:         gsy,
		P:          gsp,
		Active:     isNight || isFog,
		Window:     "Coastal fog or night (20:00-04:59)",
		Desc:       "A salt-rimed hulk drifting silent along the shoals with tattered sails.",
		ExpiresDay: day + 1,
	}

	return []smPOI{nm, fi, gs}
}

func (g *Game) smVisitPOI(kind string) (bool, string) {
	if g == nil || g.P == nil {
		return false, "No active game."
	}
	pois := g.smMovingPOIs()
	var found *smPOI
	for i := range pois {
		if pois[i].Kind == kind || pois[i].ID == kind {
			found = &pois[i]
			break
		}
	}
	if found == nil {
		return false, "No such landmark."
	}
	if !found.Active {
		return false, fmt.Sprintf("%s is not currently accessible (%s).", found.Name, found.Window)
	}
	st := g.sm()
	if st.VisitedPOIs[found.ID] > 0 {
		return false, fmt.Sprintf("You have already explored %s during this window.", found.Name)
	}
	st.VisitedPOIs[found.ID] = g.Day() + 1
	g.spend(600)

	switch found.Kind {
	case "night_market":
		curio := &Item{
			ID: g.newID(), Kind: "artifact", Name: "Night-Market Star Compass", Glyph: '*', F: CYellow,
			Value: 120, Weight: 1, Ident: true, Lore: "Bought beneath a dark moon from travelling lantern-merchants.",
		}
		g.give(curio)
		msg := "You browse the Wandering Night-Market and acquire a Night-Market Star Compass!"
		g.Msg("{{Y|" + msg + "}}")
		g.smLogf(msg)
		return true, msg
	case "floating_island":
		shard := &Item{
			ID: g.newID(), Kind: "scrap", Name: "drifting sky-crystal", Glyph: '*', F: CCyan,
			Value: 85, Weight: 1, Count: 1,
		}
		g.give(shard)
		g.XP += 45
		msg := "You climb the hanging ropes of the Migrating Floating Island and claim a drifting sky-crystal (+45 XP)!"
		g.Msg("{{C|" + msg + "}}")
		g.smLogf(msg)
		return true, msg
	case "ghost_ship":
		coin := 60 + g.R.Int(15, 45)
		g.P.Gold += coin
		g.XP += 35
		msg := fmt.Sprintf("You board the Drifting Ghost-Ship through the gloom and salvage %d coin from the captain's chest (+35 XP)!", coin)
		g.Msg("{{M|" + msg + "}}")
		g.smLogf(msg)
		return true, msg
	}
	return false, "Nothing found."
}

// ---- hooks ----

func smTurn(g *Game) {
	if g == nil || g.Dead {
		return
	}
	st := g.sm()
	if st.Exped != nil && st.Exped.Active && g.Day() > st.Exped.LastTickDay {
		g.smExpeditionTick()
	}
}

func smEnter(g *Game, zn *Zone) {
	if g == nil || zn == nil || g.W == nil {
		return
	}
	here := g.W.Idx(zn.WX, zn.WY)
	for _, p := range g.smMovingPOIs() {
		if p.P == here && p.Active {
			g.Msg(fmt.Sprintf("{{Y|Living Landmark nearby: %s!}} (%s)", p.Name, p.Desc))
		}
	}
}

func smTalk(g *Game, a *Actor) []Option {
	if g == nil || a == nil {
		return nil
	}
	var out []Option
	role := strings.ToLower(a.Role)
	if role == "merchant" || role == "trader" || role == "innkeeper" || role == "sailor" || role == "dockworker" || a.AI == "town" {
		out = append(out, Option{
			Text: "Ask about smuggler tunnels, tide hours, and moving landmarks",
			Do: func() {
				tide := "high"
				if smLowTide(g) {
					tide = "low (sea-caves open)"
				}
				pois := g.smMovingPOIs()
				g.Msg(fmt.Sprintf("{{C|%s lowers their voice:}} Tide is %s. Moon: %s. %d moving landmarks tracked.",
					a.Name, tide, smMoonName(g.Day()), len(pois)))
				g.smOpen()
			},
		})
	}
	return out
}

func smSide(g *Game) []string {
	if g == nil {
		return nil
	}
	var lines []string
	if ex := g.sm().Exped; ex != nil && ex.Active {
		lines = append(lines, fmt.Sprintf("{{C|Exped: rat %d wear %d%%}}", ex.Rations, ex.GearWear))
	}
	return lines
}

// ---- interactive mode "sm" screen ----

func smKeys(g *Game, k string) {
	st := g.sm()
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
	case 0: // Smuggler Passages
		list := g.smPassageList()
		if len(list) == 0 {
			return
		}
		if st.sel >= len(list) {
			st.sel = len(list) - 1
		}
		if k == "Enter" || k == " " {
			g.smUseTunnel(list[st.sel].ID)
		}

	case 1: // Expedition Train
		switch k {
		case "h", "Enter":
			if st.Exped == nil || !st.Exped.Active {
				dest := 0
				if g.W != nil {
					dest = g.W.Idx(g.WX, g.WY)
				}
				g.smStartExpedition(2, 3, true, dest)
			}
		case "f":
			g.smForageRations()
		case "r":
			g.smRepairWheels()
		case "t":
			g.smTreatFever()
		case "d":
			g.smDisbandExpedition()
		}

	case 2: // Moving Landmarks
		pois := g.smMovingPOIs()
		if len(pois) == 0 {
			return
		}
		if st.sel >= len(pois) {
			st.sel = len(pois) - 1
		}
		if k == "Enter" || k == " " {
			g.smVisitPOI(pois[st.sel].Kind)
		}
	}
}

func (g *Game) smDraw() {
	s := g.sc
	st := g.sm()
	x0, y0, w, h := 4, 2, SW-8, SH-4
	s.Box(x0, y0, w, h, CYellow, "Smugglers, Expeditions & Sights")

	tabs := []string{"1: Smuggler Passages", "2: Expedition Train", "3: Moving Landmarks"}
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
		tideStr := "{{R|HIGH TIDE (sea-caves flooded)}}"
		if smLowTide(g) {
			tideStr = "{{G|LOW TIDE (sea-caves open)}}"
		}
		s.TextB(x, y, fmt.Sprintf("Hour %02d:00  Tide: %s   Passages used: %d", g.Hour(), tideStr, st.TunnelsUsed), CGray, CBgDark, x0+w-2)
		y += 2
		list := g.smPassageList()
		if len(list) == 0 {
			s.TextB(x, y, "{{K|No smuggler passages discovered yet. Visit towns or coastal zones.}}", CGray, CBgDark, x0+w-2)
		} else {
			if st.sel >= len(list) {
				st.sel = len(list) - 1
			}
			for i, p := range list {
				if y >= maxY-2 {
					break
				}
				bg := CBgDark
				if i == st.sel {
					bg = CBgSel
				}
				s.Fill(x, y, w-4, 1, bg)
				tag := "{{C|cellar}}"
				if p.TideLocked {
					tag = "{{B|sea-cave}}"
				}
				s.TextB(x, y, fmt.Sprintf("%s {{Y|%s}} at (%d,%d) %s", tag, p.Name, p.WX, p.WY, p.DistrictFrom), CGray, bg, x0+w-2)
				y++
			}
		}
		s.TextB(x, maxY-1, "{{C|Actions:}} [Enter] Slip through selected passage", CGray, CBgDark, x0+w-2)

	case 1:
		ex := st.Exped
		if ex == nil || !ex.Active {
			s.TextB(x, y, "{{K|No active expedition train.}}", CGray, CBgDark, x0+w-2)
			y += 2
			s.TextB(x, y, "{{Y|[h/Enter] Hire standard porter train (2 mules, 3 porters, cartographer: 107 coin)}}", CGray, CBgDark, x0+w-2)
		} else {
			s.TextB(x, y, fmt.Sprintf("{{W|%s}}  Days out: %d", ex.Name, ex.DaysOut), CGray, CBgDark, x0+w-2)
			y++
			s.TextB(x, y, fmt.Sprintf("Crew: %d mules, %d porters, Cartographer: %v", ex.Mules, ex.Porters, ex.Cartographer), CGray, CBgDark, x0+w-2)
			y += 2
			s.TextB(x, y, fmt.Sprintf("Rations: {{G|%d/%d}}   Gear Wear: {{Y|%d%%}}   Morale: {{C|%d%%}}   Camp Fever: {{R|%d}}",
				ex.Rations, ex.MaxRations, ex.GearWear, ex.Morale, ex.Fever), CGray, CBgDark, x0+w-2)
			y += 2
			s.TextB(x, y, "{{C|Actions:}} [f] Forage rations  [r] Repair wheels (8 coin)  [t] Treat fever  [d] Disband", CGray, CBgDark, x0+w-2)
		}

	case 2:
		here := 0
		if g.W != nil && g.W.In(g.WX, g.WY) {
			here = g.W.Idx(g.WX, g.WY)
		}
		s.TextB(x, y, fmt.Sprintf("Day %d   Moon: {{Y|%s}} (phase %d/28)", g.Day()+1, smMoonName(g.Day()), smMoonPhase(g.Day())), CGray, CBgDark, x0+w-2)
		y += 2
		pois := g.smMovingPOIs()
		if st.sel >= len(pois) {
			st.sel = len(pois) - 1
		}
		for i, p := range pois {
			if y >= maxY-2 {
				break
			}
			bg := CBgDark
			if i == st.sel {
				bg = CBgSel
			}
			s.Fill(x, y, w-4, 1, bg)
			status := "{{K|DORMANT}}"
			if p.Active {
				status = "{{G|ACTIVE}}"
			}
			dir := ""
			if g.W != nil {
				dir = g.wsDirDist(here, p.P)
			}
			s.TextB(x, y, fmt.Sprintf("%s {{Y|%s}} at (%d,%d) %s - %s", status, p.Name, p.WX, p.WY, dir, p.Window), CGray, bg, x0+w-2)
			y++
		}
		s.TextB(x, maxY-1, "{{C|Actions:}} [Enter] Visit/board active landmark", CGray, CBgDark, x0+w-2)
	}

	s.TextB(x0+2, y0+h-1, "{{K|Tab/1-3 switch tab  Up/Down select  Enter act  Esc close}}", CGray, CBgDark, x0+w-2)
}
