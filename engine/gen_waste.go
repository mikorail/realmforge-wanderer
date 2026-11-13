package engine

import (
	"fmt"
	"math"
)

// Genre signature: wasteland radiation and water (prefix gc). Every zone has a radiation level 0..100
// (pure hash of seed + parasang, fallout parasangs from socialworld/WorldData.Forces, dead towns, ruins,
// shielded underground and towns). Standing in it adds dose; dose drives the status "gcrad" tiers (nausea,
// hair loss, stat penalties, collapse, death). A geiger counter shows the level and clicks; rad-away
// tonics, rad-X pills, lead-lined armour, a few mutations and augments cut it; clinics and decon showers
// wash it off. Water is the survival currency: purifier machines turn dirty water into clean drams for a
// fee, caravans pay a scarcity price, and ruins hold salvage piles that sort into scrap and bits.
// Zone hook uses side rng seed:gc:wa:zoneKey only and only in genre wasteland.
// State: g.ext("gcw") planet-local (machines), g.ext("gcr") global (dose).

type gcMach struct {
	K     string `json:"k"` // purifier shower
	Queue int    `json:"q,omitempty"`
	Ready int    `json:"r,omitempty"`
	Last  int    `json:"l,omitempty"`
}

type gcWaState struct {
	Mach map[string]*gcMach `json:"m"`
	Tank map[string]bool    `json:"tk,omitempty"` // zones whose pools were fouled
}

type gcRadState struct {
	Dose   float64         `json:"d"`
	Shield int             `json:"sh,omitempty"` // rad-X protection until this game minute
	Tier   int             `json:"t"`
	Click  int             `json:"c"`
	Taste  int             `json:"ts"`
	Firsts map[string]bool `json:"f"`
}

func (g *Game) gcWa() *gcWaState {
	st := g.ext("gcw", func() any { return &gcWaState{} }).(*gcWaState)
	if st.Mach == nil {
		st.Mach = map[string]*gcMach{}
	}
	if st.Tank == nil {
		st.Tank = map[string]bool{}
	}
	return st
}

func (g *Game) gcRad() *gcRadState {
	st := g.ext("gcr", func() any { return &gcRadState{} }).(*gcRadState)
	if st.Firsts == nil {
		st.Firsts = map[string]bool{}
	}
	return st
}

func (g *Game) gcWaDeed(id, text string) {
	st := g.gcRad()
	if st.Firsts[id] {
		return
	}
	st.Firsts[id] = true
	g.deed("gc", fmt.Sprintf("The wanderer %s %s.", g.P.Name, text), 0, 0)
}

func init() {
	zoneHooks = append(zoneHooks, gcWaZone)
	enterHooks = append(enterHooks, gcWaEnter)
	turnHooks = append(turnHooks, gcWaTurn)
	sideHooks = append(sideHooks, gcWaSide)
	talkHooks = append(talkHooks, gcWaTalk)
	gcBumpers = append(gcBumpers, gcWaBump)
	gcUsers = append(gcUsers, gcWaUse)
	unLocalKeys = append(unLocalKeys, "gcw")
	for _, d := range []*skStatusDef{
		{ID: "gcrad", Name: "Irradiated", Glyph: 'r', Speed: -3},
		{ID: "gcrad2", Name: "Rad sickness", Glyph: 'R', AGI: -2, WIL: -2, Speed: -10},
		{ID: "gcrad3", Name: "Acute rad sickness", Glyph: 'R', AGI: -3, INT: -2, WIL: -3, DV: -2, Speed: -20},
	} {
		skAddStatus(d)
	}
}

// ---- radiation level (pure) ----

var gcRadCache = map[*WorldData]map[int]int{}

// gcRadParasang: the base level of a parasang, 0..100.
func gcRadParasang(g *Game, p int) int {
	w := g.W
	m := gcRadCache[w]
	if m == nil {
		m = map[int]int{}
		gcRadCache[w] = m
	}
	if v, ok := m[p]; ok {
		return v
	}
	h := int(HashStr(fmt.Sprintf("%s:gc:rad:%d", g.Seed, p)) % 1000)
	lv := h % 6
	if h%47 == 0 {
		lv += 25 // a hot spot
	}
	if w.Forces != nil {
		for i := range w.Forces.Fallout {
			fz := &w.Forces.Fallout[i]
			for _, q := range fz.P {
				if q == p {
					lv = maxi(lv, 35+h%12)
				}
				if w.W > 0 {
					d := maxi(abs(q%w.W-p%w.W), abs(q/w.W-p/w.W))
					if d >= 1 && d <= 2 {
						lv += 12 / d
					}
				}
			}
			for _, q := range fz.Core {
				if q == p {
					lv = maxi(lv, 60+h%20)
				}
			}
		}
	}
	if b := w.burgByID[w.Burg[p]]; b != nil && !b.Alive {
		lv += 20
	}
	lv = clampi(lv, 0, 100)
	m[p] = lv
	return lv
}

// gcRadAt: the level in a zone.
func (g *Game) gcRadAt(zn *Zone) int {
	if g.Genre != "wasteland" || zn == nil || g.W == nil || !g.W.In(zn.WX, zn.WY) || zn.Underwater {
		return 0
	}
	lv := gcRadParasang(g, g.W.Idx(zn.WX, zn.WY))
	switch {
	case zn.Z < 0:
		lv /= 2
	case zn.Z > 0:
		lv = lv/3 + 4
		if zn.Kind == "dungeon" {
			if s := g.siteAt(zn.WX, zn.WY, zn.ZX, zn.ZY); s != nil && s.Ruin {
				lv += 12
			}
		}
	case zn.Kind == "ruin":
		lv += 15
	case zn.Kind == "town":
		lv = lv * 6 / 10
	}
	return clampi(lv, 0, 100)
}

// gcProt: share of radiation kept out, 0..0.85.
func (g *Game) gcProt() float64 {
	p := g.P
	pr := 0.0
	for _, bp := range p.Body {
		if bp.Item != nil && bp.Item.Extra != nil && bp.Item.Extra["gcLead"] > 0 {
			pr += 0.16
		}
	}
	if g.gcHasAug("ward") {
		pr += 0.2
	}
	pr += 0.04*float64(g.mutLevel("carapace")) + 0.03*float64(g.mutLevel("regen")) - 0.05*float64(g.mutLevel("albino"))
	pr += 0.04 * float64(g.checkMod("survival")/2)
	if g.gcRad().Shield > g.Turn {
		pr += 0.5
	}
	return math.Max(0, math.Min(0.85, pr))
}

func (g *Game) gcHasGeiger() bool {
	for _, it := range g.P.Inv {
		if it.Kind == "gcitem" && it.Effect == "geiger" {
			return true
		}
	}
	return false
}

func gcDoseTier(d float64) int {
	switch {
	case d >= 800:
		return 4
	case d >= 500:
		return 3
	case d >= 250:
		return 2
	case d >= 60:
		return 1
	}
	return 0
}

var gcDoseName = []string{"clean", "nauseous", "rad sick", "acute rad sickness", "dying of rad sickness"}

func gcWaTurn(g *Game) {
	if g.Genre != "wasteland" || g.Dead || g.Cur == nil || g.Mode == "world" {
		return
	}
	st := g.gcRad()
	p := g.P
	lv := g.gcRadAt(g.Cur)
	gain := float64(lv) / 80 * (1 - g.gcProt())
	if lv >= 10 {
		st.Dose += gain
	}
	if lv < 15 {
		st.Dose -= 0.05 + 0.02*float64(g.mutLevel("regen"))
	}
	if st.Dose < 0 {
		st.Dose = 0
	}
	if st.Dose > 1100 {
		st.Dose = 1100
	}
	// geiger feedback
	if lv >= 6 && g.gcHasGeiger() && g.Turn-st.Click >= 3 && g.R.Chance(float64(lv)/70) {
		st.Click = g.Turn
		switch {
		case lv >= 60:
			g.Msg("{{R|The geiger counter howls: a solid roar of clicks.}}")
		case lv >= 30:
			g.Msg("{{O|Click-click-clickclick. The geiger counter chatters.}}")
		default:
			g.Msg("{{K|Click.  ...click.}}")
		}
	} else if lv >= 40 && !g.gcHasGeiger() && g.Turn-st.Taste >= 40 {
		st.Taste = g.Turn
		g.Msg("{{G|Your teeth ache and there is a taste of metal in your mouth.}}")
	}
	t := gcDoseTier(st.Dose)
	if t > st.Tier {
		switch t {
		case 1:
			g.Msg("{{G|Nausea rolls through you. Something in this air is hurting you.}}")
			g.gcWaDeed("rad1", "first felt the sickness of the rads")
		case 2:
			g.Msg("{{O|Your hair comes out in your hands and your gums bleed. You are rad sick.}}")
		case 3:
			g.Msg("{{R|Your skin blisters and you vomit blood. Acute rad sickness: get clean, now.}}")
			g.gcWaDeed("rad3", "was nearly cooked by radiation")
		case 4:
			g.Msg("{{R|The world goes white at the edges. The dose is lethal.}}")
		}
	} else if t < st.Tier && t >= 0 {
		g.Msg("{{G|The sickness eases a little.}}")
	}
	st.Tier = t
	delete(p.Status, "gcrad")
	delete(p.Status, "gcrad2")
	delete(p.Status, "gcrad3")
	switch {
	case t >= 3:
		p.AddSt("gcrad3", 3)
	case t == 2:
		p.AddSt("gcrad2", 3)
	case t == 1:
		p.AddSt("gcrad", 3)
	}
	if t >= 1 && g.Turn%37 == 0 {
		g.Hunger += 30
		g.Msg("{{G|You retch.}}")
	}
	if t >= 2 && g.Turn%23 == 0 {
		g.Thirst += 25
	}
	switch {
	case st.Dose >= 1000:
		g.hurt(p, 4, nil, "radiation sickness")
	case t == 4 && g.Turn%2 == 0:
		g.hurt(p, 2, nil, "radiation sickness")
	case t == 3 && g.Turn%5 == 0:
		g.hurt(p, 1, nil, "radiation sickness")
	}
}

func gcWaSide(g *Game) []string {
	if g.Genre != "wasteland" || g.Cur == nil {
		return nil
	}
	st := g.gcRad()
	var out []string
	if g.gcHasGeiger() {
		lv := g.gcRadAt(g.Cur)
		col := "G"
		if lv >= 30 {
			col = "O"
		}
		if lv >= 60 {
			col = "R"
		}
		out = append(out, fmt.Sprintf("{{%s|Rad %d/h}}", col, lv))
		if st.Dose >= 1 {
			out = append(out, fmt.Sprintf("{{K|Dose %d}}", int(st.Dose)))
		}
	} else if st.Tier >= 1 {
		out = append(out, "{{G|"+gcDoseName[st.Tier]+"}}")
	}
	return out
}

// ---- items ----

func (g *Game) gcMakeWaste(kind string) *Item {
	switch kind {
	case "geiger":
		return &Item{ID: g.newID(), Kind: "gcitem", Effect: "geiger", Name: "geiger counter", Glyph: '☼', F: CYellow, Value: 40, Weight: 1, Ident: true,
			Lore: "Carry it and the sidebar shows the radiation level and your dose, and it clicks."}
	case "radaway":
		return &Item{ID: g.newID(), Kind: "gcitem", Effect: "radaway", Name: "rad-away tonic", Glyph: '!', F: CGreen, Value: 35, Weight: 0, Ident: true,
			Lore: "Chelating tonic. Use it to flush about 220 of your dose."}
	case "radx":
		return &Item{ID: g.newID(), Kind: "gcitem", Effect: "radx", Name: "rad-X pills", Glyph: '!', F: CCyan, Value: 28, Weight: 0, Ident: true,
			Lore: "Cuts incoming radiation by half for three hours."}
	case "lead":
		return &Item{ID: g.newID(), Kind: "gcitem", Effect: "lead", Name: "lead lining kit", Glyph: '▬', F: CGray, Value: 60, Weight: 4, Ident: true,
			Lore: "Sheets of lead to sew into worn armour: a permanent radiation shield and +1 armour value."}
	case "purifier":
		return &Item{ID: g.newID(), Kind: "gcitem", Effect: "purifier", Name: "portable water purifier", Glyph: 'Φ', F: CBlue, Charges: 14, Value: 70, Weight: 3, Ident: true,
			Lore: "A hand-pumped still. Use it to turn the brackish, salt and tainted water you carry into clean drams (14 charges)."}
	}
	return nil
}

func (g *Game) gcMakeSalvage(r *Rng, hot bool) *Item {
	it := &Item{ID: g.newID(), Kind: "gcitem", Effect: "salvage", Name: "pile of pre-war salvage", Glyph: '%', F: CDarkGray, Value: 6, Weight: 4, Ident: true,
		Lore: "Wire, casings and circuit boards. Use it to sort scrap and bits out of it.", Extra: map[string]int{"tier": 1 + r.Int(0, 2)}}
	if hot {
		it.Extra["hot"] = 1
		it.Name = "pile of hot salvage"
		it.Lore += " It ticks: sorting it will cost some dose."
	}
	return it
}

func gcWaUse(g *Game, it *Item) bool {
	p := g.P
	st := g.gcRad()
	done := func() {
		g.Mode = "main"
		g.spend(1000)
	}
	switch it.Effect {
	case "geiger":
		g.Msg(fmt.Sprintf("The counter reads %d/h here. Your dose is %d.", g.gcRadAt(g.Cur), int(st.Dose)))
		return true
	case "radaway":
		st.Dose = math.Max(0, st.Dose-220)
		g.Msg("{{G|The tonic burns going down. You feel the poison leave you.}}")
		g.gcWaDeed("radaway", "flushed the rads out with a tonic")
		g.removeInv(it)
		done()
		return true
	case "radx":
		st.Shield = g.Turn + 180
		g.Msg("{{C|You swallow the pills. The air feels thinner of poison.}}")
		g.removeInv(it)
		done()
		return true
	case "lead":
		for _, bp := range p.Body {
			if bp.Item != nil && bp.Item.Kind == "armor" && (bp.Item.Extra == nil || bp.Item.Extra["gcLead"] == 0) && !bp.Severed {
				if bp.Item.Extra == nil {
					bp.Item.Extra = map[string]int{}
				}
				bp.Item.Extra["gcLead"] = 1
				bp.Item.AV++
				bp.Item.Name = "lead-lined " + bp.Item.Name
				g.Msg("{{C|You stitch lead sheets into your " + bp.Item.Name + ".}}")
				g.gcWaDeed("lead", "lined a coat with lead against the rads")
				g.removeInv(it)
				done()
				return true
			}
		}
		g.Msg("You have no unlined armour on to sew it into.")
		return true
	case "purifier":
		used := 0
		for _, c := range g.lqContainers() {
			k := lqKey(c)
			if c.Charges <= 0 || k == "fresh" || (k != "brackish" && k != "tainted" && k != "salt") {
				continue
			}
			if it.Charges < c.Charges {
				continue
			}
			it.Charges -= c.Charges
			used += c.Charges
			g.lqSet(c, "fresh", c.Charges)
		}
		if used == 0 {
			g.Msg("Nothing you carry is dirty water the purifier can handle (or it is out of charges).")
			return true
		}
		g.Msg(fmt.Sprintf("{{B|You pump and filter %d drams clean.}}", used))
		g.gcWaDeed("purify", "distilled clean water out of foul")
		if it.Charges <= 0 {
			g.removeInv(it)
			g.Msg("The filters are spent.")
		}
		done()
		return true
	case "salvage":
		r := NewRng(fmt.Sprintf("%s:gc:salv:%d", g.Seed, it.ID))
		n := r.Int(2, 6) + g.Skills["tinkering"]
		g.give(&Item{ID: g.newID(), Kind: "scrap", Name: g.skin().Scrap, Glyph: '*', F: CDarkCyan, Count: n, Value: 2})
		t := 1 + it.Extra["tier"]
		bits := r.Int(1, 3) + g.Skills["tinkering"]/2
		g.craftSt().Bits[bitKey(t)] += bits
		msg := fmt.Sprintf("{{y|You sort the pile: %d scrap and %d bits (%s).}}", n, bits, bitLabel(t))
		if it.Extra["hot"] > 0 {
			st.Dose += float64(r.Int(5, 14)) * (1 - g.gcProt())
			msg += " {{G|It ticks against your skin.}}"
		}
		if r.Chance(0.12) {
			g.give(g.gcMakeWaste(Pick(r, []string{"radaway", "radx", "geiger"})))
			msg += " {{c|Something useful was in there.}}"
		}
		g.Msg(msg)
		g.gcWaDeed("salvage", "sorted a first pile of pre-war salvage")
		g.gainXP(4)
		g.removeInv(it)
		done()
		return true
	}
	return false
}

// ---- zones ----

func gcWaZone(g *Game, zn *Zone, _ *Rng) {
	if g.Genre != "wasteland" || g.W == nil || zn == nil {
		return
	}
	r := NewRng(g.Seed + ":gc:wa:" + zn.Key)
	lv := g.gcRadAt(zn)
	switch {
	case zn.Kind == "ruin" && zn.Z == 0, zn.Kind == "dungeon" && zn.Z >= 1:
		n := 0
		if zn.Kind == "ruin" {
			n = r.Int(2, 4)
		} else if s := g.siteAt(zn.WX, zn.WY, zn.ZX, zn.ZY); s != nil && s.Ruin {
			n = r.Int(1, 2)
		} else if r.Chance(0.25) {
			n = 1
		}
		for i := 0; i < n; i++ {
			x, y, ok := gcFree(zn, r, nil)
			if !ok {
				break
			}
			it := g.gcMakeSalvage(r, lv >= 20 && r.Chance(0.6))
			it.X, it.Y = x, y
			zn.Items = append(zn.Items, it)
		}
	case zn.Kind == "town" && zn.Z == 0 && zn.Burg >= 0:
		bd := g.W.burgByID[zn.Burg]
		if bd == nil || !bd.Alive {
			return
		}
		near := gcWallside(zn)
		if zn.ZX == 1 && zn.ZY == 1 {
			if r.Chance(0.75) {
				g.gcWaPlace(zn, r, "purifier", near)
			}
			if r.Chance(0.6) {
				g.gcWaPlace(zn, r, "shower", near)
			}
		} else if r.Chance(0.12) {
			g.gcWaPlace(zn, r, "purifier", near)
		}
	}
}

func (g *Game) gcWaPlace(zn *Zone, r *Rng, kind string, near func(int, int) bool) {
	x, y, ok := gcFree(zn, r, near)
	if !ok {
		return
	}
	zn.Set(x, y, TMachine)
	g.gcWa().Mach[gcID(zn.Key, x, y)] = &gcMach{K: kind}
}

// gcWaEnter fouls open water in irradiated zones once: it becomes tainted, dirty water for purifiers.
func gcWaEnter(g *Game, zn *Zone) {
	if g.Genre != "wasteland" || zn == nil {
		return
	}
	st := g.gcWa()
	if st.Tank[zn.Key] {
		return
	}
	st.Tank[zn.Key] = true
	if g.gcRadAt(zn) < 20 {
		return
	}
	for i, v := range zn.Liq {
		if v == lqFresh || v == lqBrackish {
			zn.Liq[i] = lqTainted
		}
	}
}

// ---- machines ----

func gcWaBump(g *Game, x, y int) bool {
	if g.Genre != "wasteland" || g.Cur.At(x, y) != TMachine {
		return false
	}
	m := g.gcWa().Mach[gcID(g.Cur.Key, x, y)]
	if m == nil {
		return false
	}
	if m.K == "shower" {
		g.gcShowerMenu()
	} else {
		g.gcPurifierMenu(m)
	}
	return true
}

func (g *Game) gcDirtyDrams() int {
	n := 0
	for _, c := range g.lqContainers() {
		if k := lqKey(c); c.Charges > 0 && k != "fresh" && k != "wine" && k != "honey" && k != "blood" && k != "oil" && k != "acid" && k != "lava" {
			n += c.Charges
		}
	}
	return n
}

func (g *Game) gcPurifierMenu(m *gcMach) {
	// time passes: 1 dram every 3 minutes
	if m.Queue > 0 {
		el := (g.Turn - m.Last) / 3
		mv := mini(el, m.Queue)
		if mv > 0 {
			m.Queue -= mv
			m.Ready += mv
			m.Last += mv * 3
		}
	}
	if m.Queue == 0 {
		m.Last = g.Turn
	}
	text := fmt.Sprintf("A pre-war purifier, pumps ticking. Filtering: %d drams. Clean and waiting: %d drams. Fee 1 per dram. It takes about 3 minutes a dram.", m.Queue, m.Ready)
	var opts []Option
	if d := g.gcDirtyDrams(); d > 0 {
		opts = append(opts, Option{fmt.Sprintf("{{B|Feed in your dirty water: %d drams for %s.}}", d, g.cuFormat(d)), func() {
			if !g.cuSpend(d) {
				g.Msg("You can't cover the fee.")
				g.gcClose()
				return
			}
			n := 0
			for _, c := range g.lqContainers() {
				if k := lqKey(c); c.Charges > 0 && k != "fresh" && k != "wine" && k != "honey" && k != "blood" && k != "oil" && k != "acid" && k != "lava" {
					n += c.Charges
					g.lqSet(c, "fresh", 0)
				}
			}
			if m.Queue == 0 {
				m.Last = g.Turn
			}
			m.Queue += n
			g.Msg(fmt.Sprintf("You pour %d drams into the intake.", n))
			g.gcWaDeed("machine", "put dirty water through a purifier")
			g.gcClose()
			g.spend(1000)
		}})
	}
	if m.Ready > 0 {
		opts = append(opts, Option{fmt.Sprintf("{{G|Collect clean water (%d drams).}}", m.Ready), func() {
			got := g.lqGain(m.Ready)
			m.Ready -= got
			if got == 0 {
				g.Msg("You have no room in your containers.")
			} else {
				g.Msg(fmt.Sprintf("{{B|You draw %d clean drams.}}", got))
			}
			g.gcClose()
			g.spend(1000)
		}})
	}
	opts = append(opts, Option{"Walk away.", func() { g.gcClose() }})
	g.gcMenu("gcm", "WATER PURIFIER", text, opts)
}

func (g *Game) gcShowerMenu() {
	st := g.gcRad()
	text := fmt.Sprintf("A decontamination shower: lead-grey tiles and a rusted drain. Your dose is about %d. Two drams of clean water, or 6 in coin.", int(st.Dose))
	opts := []Option{{"{{C|Scrub down.}}", func() {
		if g.lqDrams() >= 2 && g.lqPay(2) {
		} else if !g.cuSpend(6) {
			g.Msg("You can't pay for the water.")
			g.gcClose()
			return
		}
		st.Dose = math.Max(0, math.Min(st.Dose*0.6, st.Dose-40))
		g.Msg("{{C|The water runs grey off you. You feel lighter.}}")
		g.gcWaDeed("shower", "scrubbed the fallout off in a decon shower")
		g.gcClose()
		g.spend(5000)
	}}, {"Walk away.", func() { g.gcClose() }}}
	g.gcMenu("gcm", "DECON SHOWER", text, opts)
}

// ---- talk: clinics, water, stock ----

func gcWaPrice(g *Game) int {
	lv := g.gcRadAt(g.Cur)
	spike := 2.2 + float64(lv)/80
	if g.W != nil && g.W.In(g.Cur.WX, g.Cur.WY) && g.W.Temp[g.W.Idx(g.Cur.WX, g.Cur.WY)] >= 28 {
		spike += 0.6
	}
	for id, m := range g.gcWa().Mach {
		if m.K == "purifier" && len(id) > len(g.Cur.Key) && id[:len(g.Cur.Key)+1] == g.Cur.Key+":" {
			spike *= 0.75
			break
		}
	}
	return maxi(3, int(math.Round(2*spike*g.lqPriceFactor(false))))
}

func gcWaTalk(g *Game, a *Actor) []Option {
	if g.Genre != "wasteland" || a == nil || g.Cur == nil {
		return nil
	}
	st := g.gcRad()
	if (a.Role == "merchant" || a.Role == "apothecary" || a.Role == "smith" || a.Role == "exotic") && !a.Has("gcstock") {
		a.Set("gcstock")
		r := NewRng(fmt.Sprintf("%s:gc:wstock:%d", g.Seed, a.ID))
		if r.Chance(0.6) {
			a.Inv = append(a.Inv, g.gcMakeWaste("geiger"))
		}
		if a.Role == "apothecary" || r.Chance(0.35) {
			for i := r.Int(1, 2); i > 0; i-- {
				a.Inv = append(a.Inv, g.gcMakeWaste("radaway"))
			}
			a.Inv = append(a.Inv, g.gcMakeWaste("radx"))
		}
		if a.Role == "smith" || r.Chance(0.3) {
			a.Inv = append(a.Inv, g.gcMakeWaste("lead"))
		}
		if a.Role == "merchant" && r.Chance(0.4) {
			a.Inv = append(a.Inv, g.gcMakeWaste("purifier"))
		}
	}
	var out []Option
	if a.Role == "apothecary" || a.Role == "physician" {
		if st.Dose >= 20 {
			cost := 15 + int(st.Dose)/5
			out = append(out, Option{fmt.Sprintf("{{G|Decontaminate me (chelation and a scrub).}} {{K|%s}}", g.cuFormat(cost)), func() {
				if !g.cuSpend(cost) {
					g.Msg("You can't afford the treatment.")
					g.talkMenu(g.greeting(a))
					return
				}
				st.Dose *= 0.2
				g.Msg("{{G|They scrub you, dose you, and bleed the poison out. You are nearly clean.}}")
				g.gcWaDeed("clinic", "was decontaminated at a clinic")
				g.talkMenu(g.greeting(a))
				g.spend(3000)
			}})
		}
	}
	if a.Role == "merchant" || a.Role == "exotic" || a.Role == "elder" || a.Title == "caravan master" {
		pr := gcWaPrice(g)
		if d := g.lqDrams(); d > 4 {
			n := d - 4
			out = append(out, Option{fmt.Sprintf("{{B|Sell water at the scarcity price: %d drams at %d each (water is short).}}", n, pr), func() {
				if g.lqPay(n) {
					got := g.cuEarn(n * pr)
					g.Msg(fmt.Sprintf("You sell %d drams for %s.", n, got))
					g.gcWaDeed("water", "sold water to a thirsty trader at a scarcity price")
				}
				g.talkMenu(g.greeting(a))
			}})
		}
	}
	return out
}

// gcWaOverlay: known parasangs with heavy fallout glow on the world map.
func (g *Game) gcWaOverlay(p int) (rune, byte, byte, bool) {
	if g.Genre != "wasteland" || g.W == nil || !g.Known[p] {
		return 0, 0, 0, false
	}
	if lv := gcRadParasang(g, p); lv >= 30 {
		f := CGreen
		if lv >= 60 {
			f = CYellow
		}
		return '░', f, CDarkGreen, true
	}
	return 0, 0, 0, false
}
