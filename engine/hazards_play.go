package engine

import (
	"fmt"
	"strings"
)

// Hazards in play (hazards.go): quarantine at the gates, catching and curing diseases, tremors and
// rumbles, talk, gossip, news, sidebar. Rolls use hashes of seed + turn, never g.R.

// one disease "day" in game minutes: incubation and duration are scaled so an illness plays out in hours
const hzDay = 360

func (g *Game) hzRoll(tag string) *Rng {
	return NewRng(fmt.Sprintf("%s:hz:%s:%d", g.Seed, tag, g.Turn))
}

func (g *Game) hzHere() int {
	if g.Cur == nil || !g.W.In(g.Cur.WX, g.Cur.WY) {
		return -1
	}
	return g.W.Idx(g.Cur.WX, g.Cur.WY)
}

// ---- quarantine ----

// hzPermitted: physicians' callings (mystics) get in, rogues slip in, and anyone a smuggler took through.
func (g *Game) hzPermitted(burg int) (bool, string) {
	if hzLoc(g).Pass[burg] {
		return true, "The smuggler's drain brings you up inside the walls."
	}
	switch g.Calling {
	case "mystic":
		return true, "The guards let you through: the sick need priests and healers."
	case "rogue":
		return true, "You slip over the wall where the watch is thin."
	}
	return false, ""
}

func hzEnter(g *Game, zn *Zone) {
	if g.W.Hazards == nil {
		return
	}
	l := hzLoc(g)
	ob := (*HzOutbreak)(nil)
	if zn.Kind == "town" && zn.Burg > 0 {
		ob = g.W.hzQuarantine(zn.Burg)
	}
	if ob == nil {
		l.Last, l.LX, l.LY = zn.Key, g.P.X, g.P.Y
		return
	}
	prev := g.Zones[l.Last]
	if prev == nil || prev.Burg == zn.Burg && prev.Kind == "town" || l.Last == zn.Key {
		return // already inside (or started here)
	}
	bd := g.W.burgByID[zn.Burg]
	if ok, why := g.hzPermitted(zn.Burg); ok {
		if st := hzSt(g); !st.Seen["q:"+zn.Key] {
			st.Seen["q:"+zn.Key] = true
			g.Msg("{{y|" + why + "}}")
		}
		return
	}
	g.Msg(fmt.Sprintf("{{R|The gates of %s are chained: quarantine against %s. Masked guards turn you back.}}", bd.Name, ob.Name))
	g.addSecret(fmt.Sprintf("%s is under quarantine against %s. A smuggler in the outskirts may know a way in.", bd.Name, ob.Name), "hazard", bd.P, g.W.Year)
	g.enterZone(prev, l.LX, l.LY, true)
}

// ---- catching ----

// hzExposure: the disease and per-check chance of catching it at the player's position.
func (g *Game) hzExposure() (*HzOutbreak, float64) {
	p := g.hzHere()
	if p < 0 {
		return nil, 0
	}
	ob := g.W.hzOutbreakAt(p)
	if ob == nil {
		return nil, 0
	}
	d := g.W.hzDisease(ob.Disease)
	if d.Vector == "crop" {
		return nil, 0
	}
	ch := d.Trans * 0.01
	if g.Cur.Kind == "town" {
		ch *= 2
	}
	if d.Vector != "radiation" {
		for _, a := range g.Cur.Actors {
			if a.HP > 0 && a.Has("hzsick") && abs(a.X-g.P.X) <= 1 && abs(a.Y-g.P.Y) <= 1 {
				ch += 0.02
			}
		}
	}
	return ob, ch
}

// hzImmune: how far the player is protected against disease d (1 = fully).
func (g *Game) hzImmune(d *HzDisease) float64 {
	if _, ok := hzSt(g).Immune[d.ID]; ok {
		return d.Immunity
	}
	return 0
}

// hzTryCatch rolls exposure with chance ch (masks and immunity reduce it); true when caught.
func (g *Game) hzTryCatch(ob *HzOutbreak, ch float64, roll float64) bool {
	st := hzSt(g)
	if st.Ill != nil || ob == nil {
		return false
	}
	d := g.W.hzDisease(ob.Disease)
	if g.hzMasked(g.P) && d.Vector != "radiation" && d.Vector != "mosquito" {
		ch *= 0.4
	}
	ch *= 1 - g.hzImmune(d)
	if roll >= ch {
		return false
	}
	g.hzInfect(ob.Disease, ob.Name)
	return true
}

func (g *Game) hzWhere() string {
	if g.Cur != nil && g.Cur.Burg > 0 {
		if bd := g.W.burgByID[g.Cur.Burg]; bd != nil {
			return bd.Name
		}
	}
	if g.Cur != nil {
		return g.Cur.Name
	}
	return "the road"
}

// hzInfect gives the player a disease (incubating).
func (g *Game) hzInfect(id, name string) {
	st := hzSt(g)
	d := g.W.hzDisease(id)
	if name == "" {
		name = d.Name
	}
	on := g.Turn + maxi(30, d.Incubation*hzDay)
	st.Ill = &hzIll{ID: id, Name: name, Caught: g.Turn, Onset: on, Until: on + maxi(hzDay, d.Duration*hzDay), Where: g.hzWhere()}
}

func hzStatusFor(d *HzDisease) []string {
	var out []string
	add := func(s string) {
		if !hzHasS(out, s) {
			out = append(out, s)
		}
	}
	for _, h := range d.Hits {
		switch h {
		case "str", "agi", "tou":
			add("hzsick")
		case "int", "wil":
			add("hzfog")
		case "ego":
			add("hzpox")
		}
	}
	if len(out) == 0 {
		out = append(out, "hzsick")
	}
	return out
}

// hzSymptoms runs once a minute for a sick actor: statuses, and slow harm that never kills.
func (g *Game) hzSymptoms(a *Actor, il *hzIll, d *HzDisease) {
	for _, s := range hzStatusFor(d) {
		a.AddSt(s, 3)
	}
	if g.Turn%60 == 0 && d.Lethal >= 0.02 && a.HP > a.MaxHP/4+1 && il.Cured == "" {
		a.HP--
	}
}

func (g *Game) hzRecover(a *Actor, il *hzIll, d *HzDisease) {
	for _, s := range hzStatusFor(d) {
		delete(a.Status, s)
	}
	st := hzSt(g)
	if a == g.P {
		st.Ill = nil
		st.Had++
		if d.Immunity > 0 {
			st.Immune[d.ID] = g.Turn
		}
		msg := fmt.Sprintf("The sickness leaves you. You have come through %s.", il.Name)
		if d.Immunity >= 0.5 {
			msg += " You will not take it again easily."
		}
		g.Msg("{{G|" + msg + "}}")
		g.addSecret(fmt.Sprintf("You caught %s in %s and lived.", il.Name, il.Where), "disease", g.hzHere(), g.W.Year)
		g.deed("survived", fmt.Sprintf("Survived %s, caught in %s.", il.Name, il.Where), 0, 0)
		return
	}
	delete(st.Comp, a.ID)
	if d.Immunity > 0 {
		st.CImm[a.ID] = append(st.CImm[a.ID], d.ID)
	}
	if g.Cur != nil && g.visible(a.X, a.Y) {
		g.Msg(fmt.Sprintf("{{G|%s is over %s.}}", g.The(a), il.Name))
	}
}

func hzTurn(g *Game) {
	if g.W == nil || g.Cur == nil {
		return
	}
	st := hzSt(g)
	if g.W.Hazards == nil && st.Ill == nil && len(st.Comp) == 0 {
		return
	}
	// exposure every 10 minutes
	if g.Turn%10 == 0 && st.Ill == nil {
		if ob, ch := g.hzExposure(); ob != nil {
			g.hzTryCatch(ob, ch, g.hzRoll("catch").Float())
		}
	}
	if il := st.Ill; il != nil {
		d := g.W.hzDisease(il.ID)
		switch {
		case g.Turn >= il.Until:
			g.hzRecover(g.P, il, d)
		case g.Turn >= il.Onset:
			if il.Said == 0 {
				g.Msg(fmt.Sprintf("{{R|You are sick: %s. You caught %s in %s.}}", Pick(g.hzRoll("sym"), d.Symptoms), il.Name, il.Where))
				g.addSecret(fmt.Sprintf("You fell sick with %s (%s). Cures: %s.", il.Name, d.Name, strings.Join(hzCureWords(g, d), ", ")), "disease", g.hzHere(), g.W.Year)
				il.Said = g.Turn
			} else if g.Turn-il.Said >= 240 {
				g.Msg(fmt.Sprintf("{{r|%s: %s.}}", hzCap(il.Name), Pick(g.hzRoll("sym"), d.Symptoms)))
				il.Said = g.Turn
			}
			g.hzSymptoms(g.P, il, d)
			// herbal help: a mending meal eases it
			if il.Cured == "" && hzHasS(d.Cures, "herbal") && g.P.Meal["mending"] > 0 {
				g.hzTreat("herbs", 0.5)
				g.Msg("{{G|The mending herbs in your meal ease the fever.}}")
			}
			// contagion to companions nearby, hourly
			if g.Turn%60 == 0 && d.Vector != "radiation" && d.Vector != "mosquito" {
				for _, a := range g.Cur.Actors {
					if a.AI != "follower" || a.HP <= 0 || st.Comp[a.ID] != nil || hzHasS(st.CImm[a.ID], d.ID) || abs(a.X-g.P.X)+abs(a.Y-g.P.Y) > 3 {
						continue
					}
					if NewRng(fmt.Sprintf("%s:hz:comp:%d:%d", g.Seed, a.ID, g.Turn)).Chance(d.Trans * 0.4) {
						on := g.Turn + maxi(30, d.Incubation*hzDay)
						st.Comp[a.ID] = &hzIll{ID: d.ID, Name: il.Name, Caught: g.Turn, Onset: on, Until: on + maxi(hzDay, d.Duration*hzDay), Where: g.hzWhere()}
					}
				}
			}
		}
	}
	// companions' illnesses
	for id, il := range st.Comp {
		var a *Actor
		for _, x := range g.Cur.Actors {
			if x.ID == id {
				a = x
				break
			}
		}
		if a == nil {
			if g.Turn >= il.Until { // recovered out of sight
				delete(st.Comp, id)
			}
			continue
		}
		d := g.W.hzDisease(il.ID)
		if g.Turn >= il.Until {
			g.hzRecover(a, il, d)
		} else if g.Turn >= il.Onset {
			if il.Said == 0 {
				il.Said = g.Turn
				g.Msg(fmt.Sprintf("{{r|%s has caught %s from you: %s.}}", g.The(a), il.Name, Pick(g.hzRoll("csym"), d.Symptoms)))
			}
			g.hzSymptoms(a, il, d)
		}
	}
	if g.W.Hazards != nil && g.Turn%60 == 0 {
		g.hzTremor()
	}
}

// hzTreat shortens the illness to frac of what is left (0 = cured now); companions in the zone too when all.
func (g *Game) hzTreat(how string, frac float64) {
	st := hzSt(g)
	if il := st.Ill; il != nil {
		if g.Turn < il.Onset {
			il.Onset = g.Turn
		}
		left := il.Until - g.Turn
		il.Until = g.Turn + maxi(1, int(float64(left)*frac))
		il.Cured = how
	}
}

func (g *Game) hzTreatCompanions() int {
	st := hzSt(g)
	n := 0
	for _, a := range g.Cur.Actors {
		if il := st.Comp[a.ID]; il != nil && a.AI == "follower" {
			il.Until = g.Turn + 1
			n++
		}
	}
	return n
}

func hzCureWords(g *Game, d *HzDisease) []string {
	var out []string
	for _, c := range d.Cures {
		switch c {
		case "physician":
			out = append(out, "a "+g.hzPhysicianTitle())
		case "priest":
			out = append(out, "a priest's blessing")
		case "herbal":
			out = append(out, "a mending meal")
		case "medicine":
			if g.hzLate() {
				out = append(out, g.hzCureName())
			}
		case "rest":
			out = append(out, "time")
		}
	}
	if len(out) == 0 {
		out = append(out, "time")
	}
	return out
}

// hzUseTonic: the hzcure / hzvax tonics (one line in useItem's tonic switch).
func (g *Game) hzUseTonic(it *Item) {
	st := hzSt(g)
	switch it.Effect {
	case "hzcure":
		if st.Ill == nil {
			g.Msg("You feel no different; you were not sick.")
			return
		}
		d := g.W.hzDisease(st.Ill.ID)
		if hzHasS(d.Cures, "medicine") || hzHasS(d.Cures, "herbal") {
			g.hzTreat("medicine", 0)
			g.Msg("{{G|The fever breaks within the hour.}}")
		} else {
			g.hzTreat("medicine", 0.6)
			g.Msg("{{y|It dulls the symptoms, but this sickness is not one it cures.}}")
		}
	case "hzvax":
		n := 0
		if g.W.Hazards != nil {
			for _, d := range g.W.Hazards.Diseases {
				if d.Immunity >= 0.5 {
					st.Immune[d.ID] = g.Turn
					n++
				}
			}
		}
		g.Msg(fmt.Sprintf("{{G|A sore arm, a day's fever; you are protected against %d contagions that spare survivors.}}", n))
	}
}

// ---- tremors and rumbles ----

func (g *Game) hzTremor() {
	p := g.hzHere()
	if p < 0 {
		return
	}
	w := g.W
	r := g.hzRoll("tremor")
	if f := w.hzFaultAt(p); f != nil && r.Chance(0.05) {
		dark := g.Cur.Z > 0 || g.Cur.Dark
		g.Msg(fmt.Sprintf("{{O|The ground shudders beneath you: %s stirs.}}", f.Name))
		if dark {
			n := 0
			for k := 0; k < 8 && n < 3; k++ {
				x, y := g.P.X+r.Int(-4, 4), g.P.Y+r.Int(-3, 3)
				if g.Cur.Walkable(x, y) && g.Cur.ActorAt(x, y) == nil && !(x == g.P.X && y == g.P.Y) && g.Cur.At(x, y) != TDown && g.Cur.At(x, y) != TUp {
					g.Cur.Set(x, y, TRubble)
					n++
				}
			}
			if r.Chance(0.3) {
				g.Msg("{{R|Stones fall from the roof and strike you!}}")
				g.hurt(g.P, r.Int(1, 4), nil, "falling rock")
			} else {
				g.Msg("Dust and stones rain from the ceiling.")
			}
		}
		return
	}
	for dy := -2; dy <= 2; dy++ {
		for dx := -2; dx <= 2; dx++ {
			x, y := g.Cur.WX+dx, g.Cur.WY+dy
			if !w.In(x, y) {
				continue
			}
			if v := w.hzVolcanoAt(w.Idx(x, y)); v != nil && v.Active && r.Chance(0.04) {
				g.Msg(fmt.Sprintf("{{O|A deep rumble rolls from %s; a little ash drifts down.}}", v.Name))
				return
			}
		}
	}
}

// ---- talk ----

func hzTalk(g *Game, a *Actor) []Option {
	if g.W == nil {
		return nil
	}
	st := hzSt(g)
	var out []Option
	sick := st.Ill != nil && g.Turn >= st.Ill.Onset
	compSick := false
	for _, x := range g.Cur.Actors {
		if st.Comp[x.ID] != nil && x.AI == "follower" {
			compSick = true
		}
	}
	switch a.Role {
	case "physician":
		cost := 15 + g.P.Level*3
		if sick || compSick || st.Ill != nil {
			out = append(out, Option{fmt.Sprintf("Ask for treatment. (%s)", g.cuFormat(cost)), func() {
				if !g.cuSpend(cost) {
					g.talkMenu("\"Medicine costs money, friend.\"")
					return
				}
				msg := "\"Drink this, keep warm, burn your bedding.\""
				if st.Ill != nil {
					d := g.W.hzDisease(st.Ill.ID)
					if hzHasS(d.Cures, "physician") {
						g.hzTreat("physician", 0)
						msg += " The fever breaks."
					} else {
						g.hzTreat("physician", 0.5)
						msg += " It helps a little; this one only time cures."
					}
				}
				if n := g.hzTreatCompanions(); n > 0 {
					msg += fmt.Sprintf(" They see to your companions too (%d).", n)
				}
				g.talkMenu(msg)
			}})
		}
		out = append(out, Option{fmt.Sprintf("Buy a %s. (%s)", g.hzMaskName(), g.cuFormat(25)), func() {
			if !g.cuSpend(25) {
				g.talkMenu("\"No coin, no mask.\"")
				return
			}
			g.P.Inv = append(g.P.Inv, g.hzMakeMask())
			g.talkMenu("\"Wear it over nose and mouth. It will not save you, but it helps.\"")
		}})
		cureName := g.hzCureName()
		out = append(out, Option{fmt.Sprintf("Buy %s. (%s)", cureName, g.cuFormat(35)), func() {
			if !g.cuSpend(35) {
				g.talkMenu("\"It is dear, I know.\"")
				return
			}
			g.P.Inv = append(g.P.Inv, g.hzMakeCure(false))
			g.talkMenu("\"Take it at the first fever.\"")
		}})
		if g.hzLate() {
			out = append(out, Option{fmt.Sprintf("Buy a %s. (%s)", g.hzVaxName(), g.cuFormat(40)), func() {
				if !g.cuSpend(40) {
					g.talkMenu("\"Prevention is cheaper than the grave, but not free.\"")
					return
				}
				g.P.Inv = append(g.P.Inv, g.hzMakeCure(true))
				g.talkMenu("\"One dose. It will make you feel poorly for a day.\"")
			}})
		}
		out = append(out, Option{"What is this sickness?", func() { g.talkMenu(g.hzDiseaseTalk()) }})
	case "priest":
		if sick {
			d := g.W.hzDisease(st.Ill.ID)
			out = append(out, Option{fmt.Sprintf("Pray with me against %s.", st.Ill.Name), func() {
				r := 0
				if v, ok := g.Rep[fmt.Sprintf("faith:%d", a.Religion)]; ok {
					r = v
				}
				switch {
				case hzHasS(d.Cures, "priest") && r >= 0:
					g.hzTreat("priest", 0)
					g.talkMenu("They lay hands on your brow and pray. The fever goes out of you like a tide.")
				case r >= 0:
					g.hzTreat("priest", 0.75)
					g.talkMenu("\"I will pray for you. Find a physician as well.\" You feel steadier.")
				default:
					g.talkMenu("\"Our prayers are for our own.\"")
				}
			}})
		}
	case "hzsmuggler":
		if bd := g.W.burgByID[g.W.Burg[clampi(g.hzHere(), 0, len(g.W.Burg)-1)]]; bd != nil && g.W.hzQuarantine(bd.ID) != nil && !hzLoc(g).Pass[bd.ID] {
			cost := 20 + g.P.Level*2
			out = append(out, Option{fmt.Sprintf("Get me inside %s. (%s)", bd.Name, g.cuFormat(cost)), func() {
				if !g.cuSpend(cost) {
					g.talkMenu("\"The drain is not free, friend.\"")
					return
				}
				hzLoc(g).Pass[bd.ID] = true
				g.talkMenu("\"Follow the old drain under the east wall. Hold your breath and don't touch the walls.\"")
			}})
		}
	case "homeless":
		out = append(out, Option{"What happened here?", func() { g.talkMenu(g.hzDisasterTalk(a)) }})
	}
	if a.Has("hzsick") && a.Role != "physician" {
		out = append(out, Option{"You look unwell.", func() {
			g.talkMenu("They cough into their sleeve. \"Keep your distance, if you have any sense.\"")
		}})
	}
	return out
}

func (g *Game) hzDiseaseTalk() string {
	p := g.hzHere()
	ob := g.W.hzOutbreakAt(maxi(0, p))
	if ob == nil {
		return "\"Nothing here yet, thank the heavens. Wash your hands and keep from crowds.\""
	}
	d := g.W.hzDisease(ob.Disease)
	return fmt.Sprintf("\"%s, since Y%d. It is %s: %s. The signs are %s. What helps: %s.\"", hzCap(ob.Name), ob.Since, d.Name, d.Desc, strings.Join(d.Symptoms, ", "), strings.Join(hzCureWords(g, d), ", "))
}

func (g *Game) hzDisasterTalk(a *Actor) string {
	for _, e := range g.W.hzRecentAt(maxi(0, g.hzHere())) {
		return fmt.Sprintf("\"%s. We lost the house, the tools, my %s. The %s promised help. We are still waiting.\"", hzCap(hzLabel(e)), Pick(NewRng(fmt.Sprintf("%d", a.ID)), []string{"brother", "mother", "daughter", "husband", "neighbours"}), "crown")
	}
	return "\"The earth, the water, the fire. Does it matter which? It took everything.\""
}

// ---- gossip, news, sidebar ----

// hzGossip (one line in gossip): disasters and sickness near here, about a third of the time.
func (g *Game) hzGossip(a *Actor) string {
	w := g.W
	if w == nil || w.Hazards == nil || g.Cur == nil {
		return ""
	}
	r := NewRng(fmt.Sprintf("%s:hz:gos:%d:%d", g.Seed, a.ID, g.Turn/600))
	if !r.Chance(0.35) {
		return ""
	}
	p := g.hzHere()
	if ob := w.hzOutbreakAt(p); ob != nil {
		d := w.hzDisease(ob.Disease)
		return fmt.Sprintf("\"%s is here. They say it is %s. Cover your mouth, and if you see %s, run.\"", hzCap(ob.Name), d.Desc, Pick(r, d.Symptoms))
	}
	cx, cy := g.Cur.WX, g.Cur.WY
	var best *HzRecent
	for _, e := range w.Hazards.Recent {
		e := e
		ex, ey := e.P%w.W, e.P/w.W
		if abs(ex-cx) <= 8 && abs(ey-cy) <= 8 && w.Year-e.Year <= 30 && (best == nil || e.Year > best.Year) {
			best = &e
		}
	}
	if best != nil {
		town := ""
		if len(best.Towns) > 0 {
			if bd := w.burgByID[best.Towns[0]]; bd != nil {
				town = " " + bd.Name + " still has not recovered."
			}
		}
		return fmt.Sprintf("\"Do you remember %s?%s My grandmother talks of nothing else.\"", hzLabel(best), town)
	}
	for _, v := range w.Hazards.Volcanoes {
		vx, vy := v.P%w.W, v.P/w.W
		if v.Active && abs(vx-cx) <= 6 && abs(vy-cy) <= 6 {
			return fmt.Sprintf("\"%s has been smoking again. The goats will not graze its slopes.\"", v.Name)
		}
	}
	if f := w.hzFaultAt(p); f != nil {
		return fmt.Sprintf("\"We live on %s. Every few years the cups rattle on the shelf and we all go quiet.\"", f.Name)
	}
	return ""
}

// hzNews (one line in wsNewsLines): live outbreaks and disasters of the last ten years.
func (g *Game) hzNews() []wsLine {
	w := g.W
	if w == nil || w.Hazards == nil {
		return nil
	}
	var out []wsLine
	for _, ob := range w.Hazards.Outbreaks {
		if len(ob.Parasangs) == 0 {
			continue
		}
		p := ob.Parasangs[len(ob.Parasangs)/2]
		txt := fmt.Sprintf("{{R|%s}} spreads around %s (%d leagues of country).", hzCap(ob.Name), g.wsPlace(p), len(ob.Parasangs))
		if len(ob.Quarantined) > 0 {
			var names []string
			for _, b := range ob.Quarantined {
				if bd := w.burgByID[b]; bd != nil {
					names = append(names, bd.Name)
				}
			}
			if len(names) > 0 {
				txt += " Under quarantine: " + strings.Join(names, ", ") + "."
			}
		}
		out = append(out, wsLine{p, txt})
	}
	for i := len(w.Hazards.Recent) - 1; i >= 0 && len(out) < 8; i-- {
		e := &w.Hazards.Recent[i]
		if w.Year-e.Year > 10 {
			continue
		}
		out = append(out, wsLine{e.P, fmt.Sprintf("{{O|%s}} near %s: still clearing up.", hzCap(hzLabel(e)), g.wsPlace(e.P))})
	}
	return out
}

func hzSide(g *Game) []string {
	st := hzSt(g)
	var out []string
	if il := st.Ill; il != nil && g.Turn >= il.Onset {
		out = append(out, "{{R|Sick: "+il.Name+"}}")
	}
	if g.W != nil && g.W.Hazards != nil && g.Cur != nil {
		if p := g.hzHere(); p >= 0 {
			if ob := g.W.hzOutbreakAt(p); ob != nil {
				out = append(out, "{{r|"+hzCap(ob.Name)+" here}}")
			}
			if g.Cur.Burg > 0 && g.W.hzQuarantine(g.Cur.Burg) != nil {
				out = append(out, "{{y|Quarantine}}")
			}
		}
	}
	return out
}
