package engine

import (
	"fmt"
	"math"
	"strings"
)

// Solarpunk signature: gardens and energy (prefix gd). Garden plots grow seeded crops over days with
// seasons, water, sun cover, frost and compost, simulated lazily by the day clock so they keep growing
// while you are away. Solar panels, wind turbines and battery banks form a per-zone grid integrated hour
// by hour from the weather schedule (WeatherAt); the grid lights lamps at night, irrigates plots, runs a
// purifier and a lift, charges sun-cells and refuels your sky craft. Gardeners trade seeds at seasonal
// prices and take parts to improve the town grid for reputation and perks.

type gdPlot struct {
	Crop    string  `json:"c,omitempty"`
	Planted int     `json:"pl,omitempty"`
	Prog    float64 `json:"pr,omitempty"`
	Last    int     `json:"ls,omitempty"`
	Wet     int     `json:"w,omitempty"`
	Cover   bool    `json:"cv,omitempty"`
	Comp    int     `json:"cp,omitempty"`
	CompT   int     `json:"ct,omitempty"`
	Dmg     int     `json:"d,omitempty"`
	Thirst  int     `json:"th,omitempty"`
	Ripe    int     `json:"rp,omitempty"` // turn it became ripe
	Dead    bool    `json:"dd,omitempty"`
	Care    int     `json:"ca,omitempty"`
	seedSel *Item
}

type gdGrid struct {
	Store float64 `json:"s"`
	T     int     `json:"t"`
	Gen   float64 `json:"g,omitempty"`
}

func init() {
	zoneHooks = append(zoneHooks, gdSolarZone)
	talkHooks = append(talkHooks, gdSolarTalk)
	turnHooks = append(turnHooks, gdSolarTurn)
	enterHooks = append(enterHooks, gdSolarEnter)
	sideHooks = append(sideHooks, gdSolarSide)
	for _, k := range []string{"plot", "solar", "wind", "battery", "lamp", "purifier", "lift"} {
		k := k
		gdDevOpen[k] = func(g *Game, d *Item) { gdSolarDev(g, d, k) }
	}
}

// ---- placement ----

func gdSolarZone(g *Game, zn *Zone, _ *Rng) {
	if g.Genre != "solarpunk" || !gdTown(g, zn) {
		return
	}
	r := NewRng(g.Seed + ":gd:sol:" + zn.Key)
	p := 0.5
	if zn.Kind == "outskirts" {
		p = 0.7
	}
	if !r.Chance(p) {
		return
	}
	ax, ay, ok := g.gdFreeSpot(zn, r, nil)
	if !ok {
		return
	}
	near := func(x, y int) bool { return abs(x-ax) <= 6 && abs(y-ay) <= 5 }
	put := func(k string, q int) {
		if x, y, ok := g.gdFreeSpot(zn, r, near); ok {
			g.gdPlaceDev(zn, k, x, y, q)
		}
	}
	for i := 0; i < r.Int(2, 3); i++ {
		put("plot", 1)
	}
	put("solar", 1)
	if r.Chance(0.5) {
		put("solar", 1)
	}
	if r.Chance(0.6) {
		put("wind", 1)
	}
	put("battery", 1)
	for i := 0; i < r.Int(1, 2); i++ {
		put("lamp", 1)
	}
	if zn.Kind == "town" && r.Chance(0.45) {
		put("purifier", 1)
	}
	if zn.Kind == "town" && r.Chance(0.25) {
		put("lift", 1)
	}
	if r.Chance(0.8) {
		bd := g.W.burgByID[zn.Burg]
		cu := g.W.cultByID[g.W.Culture[clampi(bd.P, 0, len(g.W.Culture)-1)]]
		if x, y, ok := g.gdFreeSpot(zn, r, near); ok {
			a := g.makeNPC("gardener", bd, cu, r)
			a.Title, a.F = "gardener", CGreen
			a.X, a.Y, a.HomeX, a.HomeY = x, y, x, y
			a.Gold = 100 + 20*a.Level
			a.Desc = fmt.Sprintf("A gardener of %s, hands dark with soil, a pouch of seeds at the belt.", bd.Name)
			zn.Actors = append(zn.Actors, a)
		}
	}
}

func gdSolarForage(g *Game, zn *Zone, x, y int) {
	t := zn.At(x, y)
	if t != TTree && t != TTrunk {
		return
	}
	st := g.gd()
	key := fmt.Sprintf("%s:%d", zn.Key, y*MW+x)
	if d, ok := st.Cut[key]; ok && g.Day()-d < 4 {
		return
	}
	st.Cut[key] = g.Day()
	r := g.gdRng("seedtree", key, g.Day())
	if !r.Chance(0.35 + 0.1*float64(g.Skills["survival"])) {
		return
	}
	var pool []*gdCrop
	for i := range gdCrops {
		if !gdCrops[i].Rare && gdCrops[i].Season[g.Season()] {
			pool = append(pool, &gdCrops[i])
		}
	}
	if len(pool) == 0 {
		return
	}
	c := Pick(r, pool)
	g.gdGive(g.gdSeedItem(c, r.Int(1, 3)))
	g.Msg("{{G|You shake a few " + c.Name + " seeds from the branches.}}")
}

func (g *Game) gdSeedItem(c *gdCrop, n int) *Item {
	return &Item{ID: g.newID(), Kind: "gdseed", Effect: c.ID, Name: c.Name + " seeds", Glyph: ',', F: c.Col, Count: n, Value: c.Price * n / 2, Weight: 1,
		Lore: fmt.Sprintf("Seeds of %s. Takes %d days; thrives in %s.", c.Name, c.Days, gdSeasonsOf(c))}
}

func gdSeasonsOf(c *gdCrop) string {
	var s []string
	for i, ok := range c.Season {
		if ok {
			s = append(s, strings.ToLower(SeasonNames[i]))
		}
	}
	return strings.Join(s, ", ")
}

// ---- the grid ----

type gdParts struct{ solar, wind, bat, lamp, plot int }

func (g *Game) gdParts(zn *Zone) gdParts {
	var p gdParts
	for _, d := range g.gdDevs(zn, "") {
		switch d.Effect {
		case "solar":
			p.solar++
		case "wind":
			p.wind++
		case "battery":
			p.bat++
		case "lamp":
			p.lamp++
		case "plot":
			p.plot++
		}
	}
	return p
}

func (g *Game) gdLevel(zn *Zone) int { return g.gd().Towns[zn.Burg] }

func (g *Game) gdGridCap(zn *Zone, p gdParts) float64 {
	return float64(p.bat) * 100 * (1 + 0.2*float64(g.gdLevel(zn)))
}

func gdLevelMul(lv int) float64 {
	m := 1 + 0.25*float64(lv)
	if lv >= 4 {
		m = 2
	}
	return m
}

func (g *Game) gdHourGen(zn *Zone, p gdParts, turn int) (gen float64) {
	pi := g.W.Idx(zn.WX, zn.WY)
	if pi < 0 || pi >= len(g.W.Biome) {
		return 0
	}
	wx := g.W.WeatherAt(pi, turn)
	id := "clear"
	if wx.Kind != nil {
		id = wx.Kind.ID
	}
	hod := ((turn + startMinutes) / 60) % 24
	sun := 0.0
	if hod >= 6 && hod < 20 {
		sun = math.Sin(math.Pi * float64(hod-6) / 14)
	}
	sf, ok := gdSolarW[id]
	if !ok {
		sf = 0.5
	}
	wf, ok := gdWindW[id]
	if !ok {
		wf = 0.8
	}
	wf *= 0.7 + 0.6*float64(HashStr(fmt.Sprint(g.Seed, "gdw", turn/60, zn.Key))%100)/100
	gen = float64(p.solar)*4*sun*sf*gdSeasonSun[SeasonOfDay(DayOf(turn))] + float64(p.wind)*3.2*wf
	return gen * gdLevelMul(g.gdLevel(zn))
}

func gdNight(turn int) bool {
	h := ((turn + startMinutes) / 60) % 24
	return h >= 19 || h < 6
}

// gdGridSim brings a zone's grid up to date hour by hour; it returns the grid (nil without a bank).
func (g *Game) gdGridSim(zn *Zone) *gdGrid {
	p := g.gdParts(zn)
	if p.bat == 0 && p.solar+p.wind == 0 {
		return nil
	}
	st := g.gd()
	gr := st.Grids[zn.Key]
	if gr == nil {
		gr = &gdGrid{Store: 15 * float64(p.bat), T: g.Turn}
		st.Grids[zn.Key] = gr
	}
	c := g.gdGridCap(zn, p)
	from := gr.T/60 + 1
	if to := g.Turn / 60; to-from > 120 {
		from = to - 120
	}
	for h := from; h <= g.Turn/60; h++ {
		gen := g.gdHourGen(zn, p, h*60)
		use := 0.0
		if gdNight(h*60) && gr.Store > 0 {
			use = float64(p.lamp)
		}
		gr.Gen = gen
		gr.Store = math.Max(0, math.Min(c, gr.Store+gen-use))
	}
	gr.T = g.Turn
	return gr
}

// gdUse spends grid energy; free when free is true. False when the bank is short.
func (g *Game) gdUse(zn *Zone, n float64, free bool) bool {
	gr := g.gdGridSim(zn)
	if free {
		return gr != nil && gr.Store >= 1
	}
	if gr == nil || gr.Store < n {
		g.Msg(fmt.Sprintf("{{y|The bank is too low (needs %.0f).}}", n))
		return false
	}
	gr.Store -= n
	return true
}

func (g *Game) gdLamps(zn *Zone) bool {
	lit := false
	gr := g.gdGridSim(zn)
	on := gr != nil && gr.Store >= 1 && gdNight(g.Turn)
	changed := false
	for _, d := range g.gdDevs(zn, "lamp") {
		want := TMachine
		f := byte(CDarkGray)
		if on {
			want, f, lit = TBrazier, CYellow, true
		}
		if zn.At(d.X, d.Y) != want {
			zn.Set(d.X, d.Y, want)
			changed = true
		}
		d.F = f
	}
	if changed {
		g.computeLight(zn)
		g.recompute()
	}
	return lit
}

func gdSolarTurn(g *Game) {
	if g.Genre != "solarpunk" || g.Cur == nil {
		return
	}
	if g.Turn%10 == 0 {
		g.gdLamps(g.Cur)
	}
}

func gdSolarEnter(g *Game, zn *Zone) {
	if g.Genre != "solarpunk" {
		return
	}
	g.gdGridSim(zn)
	if len(g.gdDevs(zn, "lamp")) > 0 {
		g.gdLamps(zn)
	}
	st := g.gd()
	ripe, wilt := 0, 0
	for _, d := range g.gdDevs(zn, "plot") {
		if p := st.Plots[gdDevKey(zn, d)]; p != nil && p.Crop != "" {
			g.gdPlotSim(zn, p)
			d.Name = gdPlotName(p)
			if p.Dead {
				wilt++
			} else if p.Prog >= 1 {
				ripe++
			}
		}
	}
	if ripe > 0 {
		g.Msg(fmt.Sprintf("{{G|%d of your garden plots here are ripe.}}", ripe))
	}
	if wilt > 0 {
		g.Msg(fmt.Sprintf("{{r|%d plots have withered.}}", wilt))
	}
}

func gdSolarSide(g *Game) []string {
	if g.Genre != "solarpunk" || g.Cur == nil {
		return nil
	}
	p := g.gdParts(g.Cur)
	if p.bat == 0 {
		return nil
	}
	if gr := g.gd().Grids[g.Cur.Key]; gr != nil {
		return []string{fmt.Sprintf("{{Y|Grid}} %s", gdBar(int(gr.Store), int(g.gdGridCap(g.Cur, p)), 8))}
	}
	return nil
}

// ---- garden plots ----

func gdPlotName(p *gdPlot) string {
	switch {
	case p.Crop == "":
		return "garden plot"
	case p.Dead:
		return "garden plot (withered)"
	case p.Prog >= 1:
		return "garden plot (ripe)"
	}
	return "garden plot (growing)"
}

// gdPlotSim advances a plot day by day since it was last looked at.
func (g *Game) gdPlotSim(zn *Zone, p *gdPlot) {
	if p.Crop == "" || p.Dead {
		p.Last = g.Turn
		return
	}
	c := gdCropByID[p.Crop]
	if c == nil {
		return
	}
	if p.Last == 0 {
		p.Last = p.Planted
	}
	pi := g.W.Idx(zn.WX, zn.WY)
	st := g.gd()
	first, last := DayOf(p.Last)+1, DayOf(g.Turn)
	if last-first > 40 {
		first = last - 40
	}
	gr := g.gdGridSim(zn)
	for d := first; d <= last && !p.Dead; d++ {
		noon := maxi(0, d*DayLen+720-startMinutes)
		se := SeasonOfDay(d)
		wxk, temp := "clear", 15
		if pi >= 0 && pi < len(g.W.Biome) {
			wx := g.W.WeatherAt(pi, noon)
			if wx.Kind != nil {
				wxk = wx.Kind.ID
			}
			temp = wx.Temp
		}
		wetK := wxk == "rain" || wxk == "storm" || wxk == "tempest" || wxk == "sporerain" || (st.Rain > noon && noon > st.Rain-480)
		if wetK {
			p.Wet = 2
		} else if p.Wet <= 0 && gr != nil && gr.Store >= 2 && g.gdParts(zn).bat > 0 {
			gr.Store -= 1 // drip irrigation off the grid
			p.Wet = 1
		}
		if p.Wet > 0 {
			p.Wet--
			p.Thirst = 0
		} else {
			p.Thirst++
			if p.Thirst >= 3 {
				p.Dead = true
				break
			}
			continue
		}
		f := 1.0
		if !c.Season[se] {
			f = 0.4
		}
		if (se == 3 || temp <= 0 || wxk == "frost" || wxk == "blizzard") && !c.Hardy && !p.Cover {
			p.Dmg++
			f = 0
			if p.Dmg >= 3 {
				p.Dead = true
				break
			}
		} else if (se == 3 || temp <= 0) && c.Hardy {
			f *= 0.5
		}
		cloudy := wxk == "overcast" || wxk == "rain" || wxk == "fog" || wxk == "snow"
		if c.Sun == 1 && p.Cover || c.Sun == 0 && !p.Cover && !cloudy && se != 3 {
			f *= 0.6
		}
		if p.CompT >= noon {
			f *= 1.25
		}
		if temp >= 36 && !p.Cover && c.Sun == 0 {
			p.Dmg++
			if p.Dmg >= 3 {
				p.Dead = true
				break
			}
		}
		if p.Prog < 1 {
			p.Prog += f / float64(c.Days)
			if p.Prog >= 1 {
				p.Prog, p.Ripe = 1, noon
			}
		} else if noon-p.Ripe > 8*DayLen {
			p.Dead = true
		}
	}
	p.Last = g.Turn
}

func (g *Game) gdSeeds() []*Item {
	var out []*Item
	for _, it := range g.P.Inv {
		if it.Kind == "gdseed" {
			out = append(out, it)
		}
	}
	return out
}

func (g *Game) gdPlotOpen(d *Item) {
	zn := g.Cur
	key := gdDevKey(zn, d)
	st := g.gd()
	p := st.Plots[key]
	if p == nil {
		p = &gdPlot{}
		st.Plots[key] = p
	}
	g.gdPlotSim(zn, p)
	g.gdOpen("gdx", "Garden plot", func(u *gdUI) { gdPlotBuild(g, u, d, p) })
}

func gdPlotBuild(g *Game, u *gdUI, d *Item, p *gdPlot) {
	d.Name = gdPlotName(p)
	cov := "open to the sun"
	if p.Cover {
		cov = "under a cloth cover"
	}
	u.line(fmt.Sprintf("{{C|%s}}, %s. Soil moisture: %s. Compost: %d/3.", g.gdSeasonName(), cov, []string{"dry", "damp", "wet"}[clampi(p.Wet, 0, 2)], p.Comp))
	if p.Crop != "" {
		c := gdCropByID[p.Crop]
		switch {
		case p.Dead:
			u.line("{{r|The " + c.Name + " has withered.}}")
		case p.Prog >= 1:
			u.line("{{G|The " + c.Name + " is ripe.}}")
		default:
			u.line(fmt.Sprintf("{{Y|%s}} %s %d%%, thrives in %s, wants %s.", gdBar(int(p.Prog*100), 100, 16), c.Name, int(p.Prog*100), gdSeasonsOf(c), map[int]string{1: "open sun", 0: "shade"}[c.Sun]))
		}
	} else {
		u.line("Bare soil.")
	}
	zn := g.Cur
	if p.Crop == "" {
		seeds := g.gdSeeds()
		if len(seeds) == 0 {
			u.off("Plant (you carry no seeds)")
		}
		for _, s := range seeds {
			s := s
			u.row(fmt.Sprintf("Plant %s", s.Display()), func() {
				p.Crop, p.Planted, p.Last, p.Prog, p.Dmg, p.Thirst, p.Dead, p.Ripe, p.Wet = s.Effect, g.Turn, g.Turn, 0, 0, 0, false, 0, 1
				g.gdTake("gdseed", s.Effect, 1)
				g.spendTurns(3)
				g.Msg("You sow the bed.")
			})
		}
	} else if p.Dead || p.Prog >= 1 {
		if !p.Dead {
			u.row("{{G|Harvest.}}", func() { g.gdPlotHarvest(d, p) })
		}
		u.row("Pull everything up (it becomes compost).", func() {
			*p = gdPlot{Comp: mini(3, p.Comp+1), CompT: g.Turn + 3*DayLen, Last: g.Turn}
			g.spendTurns(2)
		})
	} else {
		u.row("Pull the crop up.", func() { *p = gdPlot{Comp: p.Comp, CompT: p.CompT, Last: g.Turn} })
	}
	if p.Crop == "" || (!p.Dead && p.Prog < 1) {
		u.row("Water the bed.", func() { p.Wet, p.Thirst = 2, 0; g.spendTurns(2); g.Msg("You water the bed.") })
	}
	cv := "Spread a cloth cover (frost, heat, shade)."
	if p.Cover {
		cv = "Take the cover off (open to sun)."
	}
	u.row(cv, func() { p.Cover = !p.Cover; g.spendTurns(1) })
	u.row("Add compost.", func() {
		if p.Comp >= 3 {
			g.Msg("The bed is already rich.")
			return
		}
		if g.gdTake("gdcomp", "", 1) {
		} else {
			var scrap *Item
			for _, it := range g.P.Inv {
				if it.Kind == "food" || it.Kind == "ingredient" {
					scrap = it
					break
				}
			}
			if scrap == nil {
				g.Msg("{{y|You have no compost husks or scraps to add.}}")
				return
			}
			g.gdTakeOne(scrap)
		}
		p.Comp++
		p.CompT = g.Turn + 3*DayLen
		g.spendTurns(2)
		g.Msg("You work compost into the soil.")
	})
	if gr := g.gdGridSim(zn); gr != nil {
		u.line("{{K|The grid will drip-water this bed if the bank holds charge.}}")
	}
}

func (g *Game) gdPlotHarvest(d *Item, p *gdPlot) {
	c := gdCropByID[p.Crop]
	r := g.gdRng("harvest", gdDevKey(g.Cur, d), g.Turn)
	n := r.Int(c.Yield[0], c.Yield[1]) + p.Comp/2 + g.Skills["survival"]/2
	switch c.Kind {
	case "food":
		g.give(&Item{ID: g.newID(), Kind: "food", Name: c.Name, Glyph: '%', F: c.Col, Count: n, Value: c.Price / 2, Power: 350})
	case "herb":
		g.give(&Item{ID: g.newID(), Kind: "ingredient", Name: c.Name, Effect: c.Tag, Glyph: '♣', F: c.Col, Count: n, Value: c.Price / 2, Power: 150})
	default:
		g.gdGive(&Item{ID: g.newID(), Kind: "gdflower", Name: c.Name + " blooms", Glyph: '*', F: c.Col, Count: n, Value: c.Price * n / 2})
	}
	sd := r.Int(1, 2)
	g.gdGive(g.gdSeedItem(c, sd))
	g.gdGive(&Item{ID: g.newID(), Kind: "gdcomp", Name: "compost husks", Glyph: ',', F: CBrown, Count: 2, Value: 2, Weight: 1})
	g.Msg(fmt.Sprintf("{{G|You harvest %d %s, %d seed%s and some husks.}}", n, c.Name, sd, map[bool]string{true: "s", false: ""}[sd > 1]))
	g.gainXP(5 + c.Days)
	g.gdFirst("harvest", "brought in a first harvest from a garden plot.")
	*p = gdPlot{Comp: p.Comp, CompT: p.CompT, Last: g.Turn}
	d.Name = gdPlotName(p)
	g.spendTurns(4)
}

// ---- grid machines ----

func gdSolarDev(g *Game, d *Item, kind string) {
	zn := g.Cur
	if kind == "plot" {
		g.gdPlotOpen(d)
		return
	}
	g.gdGridSim(zn)
	g.gdLamps(zn)
	g.gdOpen("gdx", gdDevDefs[kind].Name, func(u *gdUI) { gdGridBuild(g, u, d, kind) })
}

func gdGridBuild(g *Game, u *gdUI, d *Item, kind string) {
	zn := g.Cur
	p := g.gdParts(zn)
	gr := g.gdGridSim(zn)
	lv := g.gdLevel(zn)
	capv := g.gdGridCap(zn, p)
	if gr == nil {
		u.line("{{K|Nothing is wired here.}}")
		return
	}
	pi := g.W.Idx(zn.WX, zn.WY)
	wn := "unknown"
	if wx := g.W.WeatherAt(pi, g.Turn); wx.Kind != nil {
		wn = wx.Kind.Name
	}
	u.line(fmt.Sprintf("{{Y|Bank}} %s %.0f/%.0f. Output now %.1f/h. Weather: %s. %d panels, %d turbines, %d banks, %d lamps. Town grid level %d.", gdBar(int(gr.Store), int(capv), 14), gr.Store, capv, g.gdHourGen(zn, p, g.Turn), wn, p.solar, p.wind, p.bat, p.lamp, lv))
	free := func(minLv int) bool { return lv >= minLv }
	switch kind {
	case "battery":
		u.row("Charge a sun-cell (25 units).", func() {
			if g.gdUse(zn, 25, false) {
				g.gdGive(&Item{ID: g.newID(), Kind: "gdcell", Effect: "sun", Name: "sun-cell", Glyph: '=', F: CYellow, Count: 1, Value: 20, Lore: "A charged sun-cell. Feed it to any battery bank for 25 units, or to a sky craft."})
				g.Msg("A sun-cell clicks full.")
			}
		})
		if n := g.gdCount("gdcell", "sun"); n > 0 {
			u.row(fmt.Sprintf("Empty a sun-cell into the bank (%d carried).", n), func() {
				g.gdTake("gdcell", "sun", 1)
				gr.Store = math.Min(capv, gr.Store+25)
			})
			if jy := g.jy(); jy.Craft != "" {
				if jd := jy.def(g); jd != nil && jd.FuelMax > 0 && jy.Fuel < jd.FuelMax {
					u.row(fmt.Sprintf("Charge your craft's %s with a cell (+0.5 day).", jd.fuel(g)), func() {
						g.gdTake("gdcell", "sun", 1)
						jy.Fuel = math.Min(jd.FuelMax, jy.Fuel+0.5)
						g.Msg(fmt.Sprintf("{{G|Your %s now holds %.1f days.}}", jd.name(g), jy.Fuel))
					})
				}
			}
		}
	case "purifier":
		u.row("Purify water: refill your flasks, drink, clear poison.", func() {
			if !g.gdUse(zn, 6, free(2)) {
				return
			}
			g.Thirst = 0
			for _, it := range g.P.Inv {
				if it.Kind == "water" {
					it.Charges = it.Power
				}
			}
			delete(g.P.Status, "poisoned")
			g.spendTurns(3)
			g.Msg("{{C|Clean water, sweet and cold.}}")
		})
	case "lift":
		u.row("Ride the lift to the top and survey the land.", func() {
			if !g.gdUse(zn, 8, free(3)) {
				return
			}
			for i := range zn.Seen {
				zn.Seen[i] = 1
			}
			for dy := -2; dy <= 2; dy++ {
				for dx := -2; dx <= 2; dx++ {
					if g.W.In(zn.WX+dx, zn.WY+dy) {
						g.Known[g.W.Idx(zn.WX+dx, zn.WY+dy)] = true
					}
				}
			}
			var fc []string
			for h := 1; h <= 3; h++ {
				if wx := g.W.WeatherAt(pi, g.Turn+h*DayLen/3); wx.Kind != nil {
					fc = append(fc, wx.Kind.Name)
				}
			}
			g.spendTurns(2)
			g.Msg("{{C|From the top the land lies open. Forecast: " + strings.Join(fc, ", ") + ".}}")
		})
	case "lamp":
		u.line("It burns by itself on dark nights while the bank has charge.")
	}
}

// ---- gardeners: seeds, produce, the town grid ----

func gdSolarTalk(g *Game, a *Actor) []Option {
	if g.Genre != "solarpunk" || a.Role != "gardener" {
		return nil
	}
	return []Option{
		{"{{O|Buy seeds.}}", func() { gdSeedShop(g, a) }},
		{"{{O|Sell seeds or produce.}}", func() { gdSellMenu(g, a) }},
		{"{{C|What grows now?}}", func() { g.talkMenu(gdSeasonTalk(g)) }},
		{"{{W|Improve the town grid (deliver parts).}}", func() { gdGridTalk(g, a) }},
	}
}

func gdSeasonTalk(g *Game) string {
	var in []string
	for _, c := range gdCrops {
		if c.Season[g.Season()] && !c.Rare {
			in = append(in, c.Name)
		}
	}
	return fmt.Sprintf("\"It is %s. Sow %s. Frost-tender things want a cloth in winter; shade-lovers a cloth in summer.\"", strings.ToLower(g.gdSeasonName()), strings.Join(in, ", "))
}

func (g *Game) gdSeedPrice(c *gdCrop, buy bool) int {
	m := 1.4
	if c.Season[g.Season()] {
		m = 0.8
	}
	lv := 0
	if g.Cur != nil {
		lv = g.gdLevel(g.Cur)
	}
	if lv >= 1 {
		m -= 0.1
	}
	if lv >= 3 {
		m -= 0.1
	}
	v := float64(c.Price) * m
	if !buy {
		v *= 0.5
	}
	return maxi(1, int(v))
}

func gdSeedShop(g *Game, a *Actor) {
	g.Popup = Wrap("\"Seeds, by the season. Cheaper when they want to grow.\"", 100)
	g.Opts = nil
	r := g.gdRng("seeds", a.ID, g.Day())
	var pool []*gdCrop
	for i := range gdCrops {
		pool = append(pool, &gdCrops[i])
	}
	r.Chance(0.5)
	for i := 0; i < 6 && len(pool) > 0; i++ {
		j := r.Int(0, len(pool)-1)
		c := pool[j]
		pool = append(pool[:j], pool[j+1:]...)
		if c.Rare && !r.Chance(0.35) {
			continue
		}
		price := g.gdSeedPrice(c, true) * 3
		if c.Rare {
			price *= 2
		}
		g.Opts = append(g.Opts, Option{fmt.Sprintf("3 %s seeds {{K|(%s; %s)}}", c.Name, g.gdPrice(price), strings.Join(strings.Fields(gdSeasonsOf(c)), " ")), func() {
			if g.gdPay(price) {
				g.gdGive(g.gdSeedItem(c, 3))
				gdSeedShop(g, a)
			}
		}})
	}
	g.Opts = append(g.Opts, Option{"Enough.", func() { g.talkMenu(g.greeting(a)) }})
	g.Sel = 0
}

func gdSellMenu(g *Game, a *Actor) {
	g.Popup = Wrap("\"I buy seed and good produce.\"", 100)
	g.Opts = nil
	mul := 1.0
	if g.Cur != nil && g.gdLevel(g.Cur) >= 4 {
		mul = 1.3
	}
	for _, it := range g.P.Inv {
		it := it
		var price int
		switch it.Kind {
		case "gdseed":
			if c := gdCropByID[it.Effect]; c != nil {
				price = g.gdSeedPrice(c, false) * maxi(it.Count, 1)
			}
		case "gdflower":
			price = int(float64(it.Value) * 1.2 * mul)
		case "food", "ingredient":
			for _, c := range gdCrops {
				if strings.HasPrefix(it.Name, c.Name) {
					scarce := 1.0
					if !c.Season[g.Season()] {
						scarce = 1.5
					}
					price = int(float64(c.Price) * 0.6 * scarce * mul * float64(maxi(it.Count, 1)))
				}
			}
		}
		if price <= 0 {
			continue
		}
		if len(g.Opts) >= 9 {
			break
		}
		g.Opts = append(g.Opts, Option{fmt.Sprintf("Sell %s {{K|(%s)}}", it.Display(), g.gdPrice(price)), func() {
			g.removeInv(it)
			g.cuEarn(price)
			g.Msg("You sell " + it.Display() + ".")
			gdSellMenu(g, a)
		}})
	}
	g.Opts = append(g.Opts, Option{"Enough.", func() { g.talkMenu(g.greeting(a)) }})
	g.Sel = 0
}

func gdGridTalk(g *Game, a *Actor) {
	st := g.gd()
	bid := g.Cur.Burg
	lv := st.Towns[bid]
	g.Popup = Wrap(fmt.Sprintf("\"The grid here is level %d. %s\"", lv, map[bool]string{true: "It is as good as a grid gets.", false: "Bring parts and it grows: " + gdGridPerk[mini(lv+1, 5)] + "."}[lv >= 5]), 100)
	g.Opts = nil
	if lv < 5 {
		cost := gdGridCost[lv+1]
		g.Opts = append(g.Opts, Option{fmt.Sprintf("Deliver %s to reach level %d.", gdCostLine(cost), lv+1), func() {
			if !g.gdPayBits(cost) {
				g.talkMenu("\"You have not got the parts, friend.\"")
				return
			}
			st.Towns[bid] = lv + 1
			g.AddRep(a.Faction, 30+30*lv, " for improving the town grid")
			g.gainXP(30 * (lv + 1))
			g.gdFirst("grid1", "improved a town's grid by delivering parts.")
			if lv+1 == 5 {
				g.give(&Item{ID: g.newID(), Kind: "artifact", Name: "grid-wrights' heliograph", Glyph: '*', F: CYellow, Ident: true, Value: 400, Bonus: map[int]int{WIL: 2, INT: 2}, Rarity: "rare",
					Lore: "A mirror that talks to every grid in the world.", Slot: "face"})
				g.gdFirst("grid5", "completed a town's grid and was gifted a heliograph.")
			}
			g.talkMenu(fmt.Sprintf("\"Level %d! Now: %s.\"", lv+1, gdGridPerk[lv+1]))
		}})
	}
	g.Opts = append(g.Opts, Option{"Enough.", func() { g.talkMenu(g.greeting(a)) }})
	g.Sel = 0
}
