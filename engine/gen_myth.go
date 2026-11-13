package engine

import (
	"fmt"
	"strings"
)

// Fantasy / mythic signature: gods that answer and omens (prefix gd). Each religion's deity has a
// Favour meter (-100..100) raised by prayer, offerings and deeds in its domain (derived from the deity's
// name and doctrine), lowered by sacrilege (killing priests, robbing barrows). At thresholds and on
// request the god answers with real effects: a day-long blessing (status), a miracle (heal, cure plague,
// calm beasts, rain that also waters gardens), a sign (reveals a secret or lead through the secrets
// system) or, at the top, a relic; sacrilege brings a curse. Omens: a seeded daily sign (comet, ravens,
// bees...) foretells a real event a few days off. When the sign is true the event is injected into the
// world sim at its hour (plague, famine, war army, bandits, boom); storm signs are true weather forecasts.
// Priests and elders read them for coin. Pacts stay in arcane.go; this is public, shrine-side faith.

type gdFaith struct {
	Fav     int             `json:"f"`
	Prayed  int             `json:"p,omitempty"` // day+1 of the last prayer
	GaveD   int             `json:"gd,omitempty"`
	GaveN   int             `json:"gn,omitempty"`
	Got     int             `json:"gt,omitempty"` // answered thresholds bitmask
	Sacr    int             `json:"sc,omitempty"`
	Temples map[string]bool `json:"tm,omitempty"`
}

func init() {
	talkHooks = append(talkHooks, gdMythTalk)
	killHooks = append(killHooks, gdMythKill)
	turnHooks = append(turnHooks, gdMythTurn)
	sideHooks = append(sideHooks, gdMythSide)
	for _, d := range gdDomains {
		skAddStatus(gdBlessDef(d.ID))
	}
	skAddStatus(&skStatusDef{ID: "gdcurse", Name: "Cursed", Glyph: 'x', Hit: -3, DV: -3, WIL: -3, Speed: -10})
}

func gdBlessDef(dom string) *skStatusDef {
	d := &skStatusDef{ID: "gdb_" + dom, Name: "Blessed", Glyph: '+'}
	switch dom {
	case "war":
		d.Hit, d.Speed, d.AV = 3, 10, 1
	case "harvest":
		d.AV, d.WIL = 2, 2
		d.Tick = func(g *Game, a *Actor) { gdMend(g, a, 4) }
	case "death":
		d.DV, d.Hit = 3, 1
	case "sea":
		d.Speed, d.DV = 15, 2
	case "trade":
		d.EGO, d.INT = 4, 2
	case "wisdom":
		d.INT, d.WIL = 4, 3
	case "hearth":
		d.AV, d.DV = 1, 1
		d.Tick = func(g *Game, a *Actor) { gdMend(g, a, 3) }
	default:
		d.AGI, d.DV, d.Speed = 4, 2, 8
	}
	return d
}

func gdMend(g *Game, a *Actor, every int) {
	if g.Turn%every == 0 && a.HP < a.MaxHP {
		a.HP++
	}
}

func (g *Game) gdFaithOf(id int) *gdFaith {
	st := g.gd()
	f := st.Faith[id]
	if f == nil {
		f = &gdFaith{Temples: map[string]bool{}}
		st.Faith[id] = f
	}
	if f.Temples == nil {
		f.Temples = map[string]bool{}
	}
	return f
}

func (g *Game) gdRel(id int) *ReligionData {
	if g.W == nil {
		return nil
	}
	return g.W.relByID[id]
}

func (g *Game) gdDeity(id int) string {
	if r := g.gdRel(id); r != nil {
		return nonEmpty(r.Deity, r.Name)
	}
	return "the unnamed gods"
}

func (g *Game) gdDomain(id int) *gdDomainDef {
	r := g.gdRel(id)
	txt := strings.ToLower(fmt.Sprint(id))
	if r != nil {
		txt = strings.ToLower(r.Name + " " + r.Deity + " " + r.Theology + " " + r.Doctrine)
	}
	best, bs := -1, 0
	for i, d := range gdDomains {
		s := 0
		for _, w := range d.Words {
			s += strings.Count(txt, w)
		}
		if s > bs {
			best, bs = i, s
		}
	}
	if best < 0 {
		best = int(HashStr(g.Seed+":gd:dom:"+txt) % uint64(len(gdDomains)))
	}
	return &gdDomains[best]
}

func favWord(f int) string {
	switch {
	case f >= 80:
		return "{{W|exalted}}"
	case f >= 50:
		return "{{G|beloved}}"
	case f >= 20:
		return "{{g|favoured}}"
	case f > -10:
		return "{{y|unnoticed}}"
	case f > -40:
		return "{{r|disfavoured}}"
	}
	return "{{R|cursed}}"
}

// gdLocalRel is the faith of the land you stand in, else your home faith.
func (g *Game) gdLocalRel() int {
	if g.W != nil && g.Cur != nil && g.W.In(g.Cur.WX, g.Cur.WY) {
		if p := g.W.Idx(g.Cur.WX, g.Cur.WY); p < len(g.W.Religion) && g.W.Religion[p] > 0 {
			return g.W.Religion[p]
		}
	}
	return maxi(g.HomeFaith, 0)
}

func (g *Game) gdPatron() int {
	st := g.gd()
	if st.Patron > 0 {
		return st.Patron
	}
	return maxi(g.HomeFaith, 0)
}

func (g *Game) gdAddFav(id, n int, why string) {
	if id <= 0 || n == 0 {
		return
	}
	f := g.gdFaithOf(id)
	f.Fav = clampi(f.Fav+n, -100, 100)
	if why != "" {
		col := "G"
		if n < 0 {
			col = "R"
		}
		g.Msg(fmt.Sprintf("{{%s|%+d}} favour with {{Y|%s}}%s.", col, n, g.gdDeity(id), why))
	}
}

// ---- altars and priests ----

func (g *Game) gdAltar(zn *Zone, x, y int) {
	rel := g.gdLocalRel()
	note := nonEmpty(zn.Notes[y*MW+x], "A small shrine.")
	if rel == 0 {
		g.showPopup("Altar", note+"\n\nYou pray a while.")
		if g.P.HP < g.P.MaxHP {
			g.P.HP = mini(g.P.MaxHP, g.P.HP+5)
		}
		g.spend(1000)
		return
	}
	g.gdFaithMenu(rel, note)
}

func (g *Game) gdFaithMenu(rel int, note string) {
	g.gdOpen("gdx", "Altar of "+g.gdDeity(rel), func(u *gdUI) { gdFaithBuild(g, u, rel, note) })
}

func gdFaithBuild(g *Game, u *gdUI, rel int, note string) {
	f := g.gdFaithOf(rel)
	dom := g.gdDomain(rel)
	day := g.Day()
	if note != "" {
		u.line("{{K|" + note + "}}")
	}
	u.line(fmt.Sprintf("{{Y|%s}}, god of %s. Favour %s %d, %s.%s", g.gdDeity(rel), dom.Name, gdBar(f.Fav+100, 200, 16), f.Fav, favWord(f.Fav),
		map[bool]string{true: "  {{C|You follow this god.}}", false: ""}[g.gdPatron() == rel]))
	if f.Prayed == day+1 {
		u.off("Pray (you have prayed today)")
	} else {
		u.row("Pray for a while.", func() {
			f.Prayed = day + 1
			g.spendTurns(10)
			if g.P.HP < g.P.MaxHP {
				g.P.HP = mini(g.P.MaxHP, g.P.HP+5)
			}
			add := 2 + boolI(g.gdPatron() == rel)
			if g.Cur != nil && !f.Temples[g.Cur.Key] {
				f.Temples[g.Cur.Key] = true
				add += 3
			}
			g.gdAddFav(rel, add, " for your prayer")
			g.gdFirst("pray", "knelt at an altar and prayed to "+g.gdDeity(rel)+".")
			g.gdAnswer(rel)
		})
	}
	for _, amt := range []int{10, 50, 200} {
		amt := amt
		gain := map[int]int{10: 1, 50: 4, 200: 12}[amt]
		if dom.ID == "trade" {
			gain *= 2
		}
		if f.GaveD != day+1 {
			f.GaveD, f.GaveN = day+1, 0
		}
		if f.GaveN >= 12 {
			break
		}
		u.row(fmt.Sprintf("Offer %s {{K|(+%d favour)}}", g.gdPrice(amt), gain), func() {
			if g.gdPay(amt) {
				f.GaveN += gain
				g.gdAddFav(rel, gain, " for your offering")
				g.gdAnswer(rel)
			}
		})
	}
	n := 0
	for _, it := range g.P.Inv {
		it := it
		if n >= 5 || it.Quest > 0 || it.Kind == "gddev" {
			continue
		}
		good := false
		for _, k := range dom.Gives {
			good = good || k == it.Kind
		}
		if !good {
			continue
		}
		n++
		gain := 2 + mini(it.Value/10, 6)
		u.row(fmt.Sprintf("Offer %s {{K|(+%d favour, %s likes this)}}", it.Display(), gain, g.gdDeity(rel)), func() {
			g.gdTakeOne(it)
			g.gdAddFav(rel, gain, " for the offering")
			g.gdAnswer(rel)
		})
	}
	if g.gdPatron() != rel {
		u.row("Swear to follow this god. (An old patron resents it.)", func() {
			if old := g.gdPatron(); old > 0 && old != rel {
				g.gdAddFav(old, -5, " for turning away")
			}
			g.gd().Patron = rel
			g.Msg("{{Y|You swear yourself to " + g.gdDeity(rel) + ".}}")
		})
	}
	gdBoonRows(g, u, rel)
	if f.Fav < 0 {
		pen := 20 * -f.Fav
		u.row(fmt.Sprintf("Do penance: make amends {{K|(%s)}}", g.gdPrice(pen)), func() {
			if g.gdPay(pen) {
				f.Fav = 0
				f.Sacr = 0
				delete(g.P.Status, "gdcurse")
				g.Msg("{{G|The weight lifts. " + g.gdDeity(rel) + " forgives.}}")
			}
		})
	}
}

func gdBoonRows(g *Game, u *gdUI, rel int) {
	f := g.gdFaithOf(rel)
	dom := g.gdDomain(rel)
	if f.Fav < 10 {
		u.line("{{K|The god does not yet know you well enough to grant boons.}}")
		return
	}
	u.row(fmt.Sprintf("Ask a blessing {{K|(10 favour: %s, a day)}}", dom.Bless), func() { g.gdBless(rel, 10) })
	if f.Fav >= 15 {
		u.row("Ask for a sign {{K|(15 favour: a secret revealed)}}", func() { g.gdSign(rel, 15) })
	}
	if f.Fav >= 15 {
		cost := func(m string) int {
			if dom.Miracle == m {
				return 15
			}
			return 25
		}
		for _, m := range []string{"heal", "cure", "calm", "rain"} {
			m := m
			c := cost(m)
			if f.Fav < c {
				continue
			}
			lbl := map[string]string{"heal": "Heal you and your companions", "cure": "Cure sickness", "calm": "Calm the beasts nearby", "rain": "Bring rain"}[m]
			u.row(fmt.Sprintf("Ask a miracle: %s {{K|(%d favour)}}", lbl, c), func() { g.gdMiracle(rel, m, c) })
		}
	}
}

func (g *Game) gdBless(rel, cost int) {
	f := g.gdFaithOf(rel)
	dom := g.gdDomain(rel)
	mins := 1440
	if f.Fav >= 60 {
		mins = 2160
	}
	if cost > 0 {
		f.Fav -= cost
	}
	g.P.AddSt("gdb_"+dom.ID, mins)
	g.Msg(fmt.Sprintf("{{Y|%s's blessing settles on you: %s.}}", g.gdDeity(rel), dom.Bless))
	g.gdFirst("bless", "was blessed by "+g.gdDeity(rel)+".")
}

func (g *Game) gdSign(rel, cost int) {
	f := g.gdFaithOf(rel)
	if cost > 0 {
		f.Fav -= cost
	}
	g.Msg("{{Y|A vision from " + g.gdDeity(rel) + ".}}")
	txt := sxTell(g, g.P)
	g.Msg(strings.Trim(txt, "\""))
	g.addSecret(Plain(txt), "sign", g.gdParasang(), g.W.Year)
	g.gdFirst("sign", "was shown a sign by "+g.gdDeity(rel)+".")
}

func (g *Game) gdMiracle(rel int, m string, cost int) {
	f := g.gdFaithOf(rel)
	if cost > 0 {
		f.Fav -= cost
	}
	switch m {
	case "heal":
		g.P.HP = g.P.MaxHP
		for _, a := range g.localCompanions() {
			a.HP = a.MaxHP
		}
		delete(g.P.Status, "bleeding")
		g.Msg("{{G|Warm light: you and your companions are made whole.}}")
	case "cure":
		g.hzTreat("a miracle", 0)
		n := g.hzTreatCompanions()
		delete(g.P.Status, "poisoned")
		delete(g.P.Status, "ill")
		g.Msg(fmt.Sprintf("{{G|The sickness leaves you%s.}}", map[bool]string{true: " and your companions", false: ""}[n > 0]))
	case "calm":
		k := 0
		for _, a := range g.Cur.Actors {
			if a.HP > 0 && a != g.P && a.AI == "hostile" && !a.Boss && a.Role == "" && abs(a.X-g.P.X) <= 14 && abs(a.Y-g.P.Y) <= 14 {
				a.AddSt("asleep", 120)
				a.Angry = false
				k++
			}
		}
		g.Msg(fmt.Sprintf("{{G|A hush falls. %d creatures lie down and sleep.}}", k))
	case "rain":
		g.gd().Rain = g.Turn + 360
		g.Msg("{{B|Clouds gather out of nothing, and the rain begins.}}")
	}
	g.gdFirst("miracle", "was granted a miracle by "+g.gdDeity(rel)+".")
	g.spendTurns(2)
}

// gdAnswer: the god answers unprompted when favour first crosses a threshold.
func (g *Game) gdAnswer(rel int) {
	f := g.gdFaithOf(rel)
	for i, t := range []int{20, 40, 60, 80} {
		bit := 1 << i
		if f.Fav < t || f.Got&bit != 0 {
			continue
		}
		f.Got |= bit
		g.Msg("{{W|" + g.gdDeity(rel) + " answers you.}}")
		switch i {
		case 0:
			g.gdSign(rel, 0)
		case 1:
			g.gdBless(rel, 0)
		case 2:
			g.gdMiracle(rel, g.gdDomain(rel).Miracle, 0)
		case 3:
			r := g.gdRng("gift", rel)
			it := g.Loot(r, 5)
			it.Name = "relic of " + g.gdDeity(rel)
			it.Lore = "A gift of " + g.gdDeity(rel) + ", god of " + g.gdDomain(rel).Name + "."
			it.Rarity = "rare"
			g.give(it)
			g.Msg("{{Y|A gift lies on the altar: " + it.Display() + ".}}")
			g.gdFirst("relic", "received a relic from "+g.gdDeity(rel)+".")
		}
		return
	}
}

func gdMythTalk(g *Game, a *Actor) []Option {
	if (g.Genre != "fantasy" && g.Genre != "mythic") || g.W == nil || g.Cur == nil {
		return nil
	}
	var out []Option
	if a.Role == "priest" {
		rel := a.Religion
		if rel <= 0 {
			rel = g.gdLocalRel()
		}
		if rel > 0 {
			out = append(out, Option{"{{C|Pray with them, and ask what the god asks of you.}}", func() { g.gdFaithMenu(rel, "") }})
		}
	}
	if a.Role == "priest" || a.Role == "elder" || a.Role == "hermit" {
		out = append(out, Option{"{{M|Ask for a reading of the omens.}}", func() { gdReadMenu(g, a) }})
	}
	return out
}

func gdMythKill(g *Game, d, src *Actor) {
	if (g.Genre != "fantasy" && g.Genre != "mythic") || d == nil || (src != g.P) {
		return
	}
	if d.Role == "priest" && d.Religion > 0 {
		f := g.gdFaithOf(d.Religion)
		f.Sacr++
		g.gdAddFav(d.Religion, -30, " for the blood of a priest")
		g.gdCurse(d.Religion)
		return
	}
	p := g.gdPatron()
	if p <= 0 {
		return
	}
	switch g.gdDomain(p).ID {
	case "war":
		if d.Boss {
			g.gdAddFav(p, 3, " for a mighty foe")
		} else if d.Role == "bandit" || d.Tpl == "bandit" || d.Tpl == "archer" || d.Tpl == "chief" {
			g.gdAddFav(p, 1, "")
		}
	case "death":
		if d.Has("undead") {
			g.gdAddFav(p, 1, "")
		}
	case "wild":
		if d.Role == "bandit" {
			g.gdAddFav(p, 1, "")
		}
	}
}

func (g *Game) gdCurse(rel int) {
	f := g.gdFaithOf(rel)
	if f.Fav > -20 {
		return
	}
	g.P.AddSt("gdcurse", 720)
	g.Msg("{{R|" + g.gdDeity(rel) + " turns from you. A curse settles on your shoulders.}}")
	g.gdFirst("curse", "was cursed by "+g.gdDeity(rel)+".")
}

// ---- turn: rain, sacrilege, day deeds, omen fulfilment ----

func gdMythTurn(g *Game) {
	if (g.Genre != "fantasy" && g.Genre != "mythic") || g.W == nil || g.Cur == nil {
		return
	}
	st := g.gd()
	if st.Rain > g.Turn && g.Turn%10 == 0 {
		if p := g.gdParasang(); p >= 0 {
			w := g.W.WeatherAt(p, g.Turn)
			w.Kind = WeatherByID("rain")
			g.wxForce = &w
			st.rainOn = true
		}
	} else if st.Rain <= g.Turn && st.rainOn {
		g.wxForce, st.rainOn = nil, false
	}
	if g.Turn%60 != 0 {
		return
	}
	if n := len(buSt(g).Finds); n > st.Ctx["bu"] {
		if st.Ctx["bu"] > 0 || n > 0 {
			if p := g.gdPatron(); p > 0 {
				g.gdFaithOf(p).Sacr++
				g.gdAddFav(p, -15, " for robbing the dead")
				g.gdCurse(p)
			}
		}
		st.Ctx["bu"] = n
	}
	gdDayDeed(g)
	gdOmenFulfil(g)
}

func gdDayDeed(g *Game) {
	p := g.gdPatron()
	if p <= 0 {
		return
	}
	st := g.gd()
	dom := g.gdDomain(p)
	day := g.Day()
	if st.Ctx["dd:"+dom.ID] == day+1 {
		return
	}
	zn := g.Cur
	ok := false
	switch dom.ID {
	case "harvest":
		ok = zn.Z == 0 && (zn.Kind == "outskirts" || strings.Contains(strings.ToLower(zn.Name), "field"))
	case "sea":
		for dy := -5; dy <= 5 && !ok; dy++ {
			for dx := -5; dx <= 5 && !ok; dx++ {
				ok = zn.In(g.P.X+dx, g.P.Y+dy) && zn.WaterAt(g.P.X+dx, g.P.Y+dy) > 0
			}
		}
	case "wild":
		ok = zn.Z == 0 && zn.Kind != "town" && zn.Kind != "outskirts"
	case "trade":
		ok = g.P.Gold >= st.Ctx["gold"]+50 && st.Ctx["gold"] > 0
		st.Ctx["gold"] = g.P.Gold
	case "wisdom":
		ok = len(g.Read) > st.Ctx["read"]
		st.Ctx["read"] = len(g.Read)
	case "hearth":
		ok = g.MealLeft > 0 || g.homeHere() != nil
	case "death":
		ok = zn.Kind == "crypt"
	}
	if dom.ID == "trade" && st.Ctx["gold"] == 0 {
		st.Ctx["gold"] = g.P.Gold
	}
	if ok {
		st.Ctx["dd:"+dom.ID] = day + 1
		g.gdAddFav(p, 1, " for a deed in "+dom.Name+"'s name")
	}
}

// ---- omens ----

type gdOmenT struct {
	Has, Truth bool
	Kind, Sign string
	Sky        bool
	Lead, At   int
	Burg       *BurgData
}

func (g *Game) gdOmen(d int) gdOmenT {
	var o gdOmenT
	h := HashStr(fmt.Sprintf("%s:gd:omen:%d", g.Seed, d))
	if h%100 >= 55 || d < 0 {
		return o
	}
	o.Has = true
	o.Kind = gdOmenKinds[(h>>8)%uint64(len(gdOmenKinds))]
	o.Truth = (h>>16)%100 < 72
	o.Lead = 1 + int((h>>24)%4)
	sg := gdSigns[o.Kind]
	s := sg[(h>>28)%uint64(len(sg))]
	o.Sign, o.Sky = s.Text, s.Sky
	o.At = (d+o.Lead)*DayLen + int((h>>32)%uint64(DayLen)) - startMinutes
	if bl := g.ws().burgList(g); len(bl) > 0 {
		o.Burg = bl[(h>>40)%uint64(len(bl))]
	}
	return o
}

func gdOmenFulfil(g *Game) {
	st := g.gd()
	today := g.Day()
	for d := maxi(0, today-10); d <= today; d++ {
		if st.Done[d] {
			continue
		}
		o := g.gdOmen(d)
		if !o.Has || g.Turn < o.At {
			continue
		}
		st.Done[d] = true
		delete(st.Done, d-30)
		if !o.Truth || o.Kind == "storm" || o.Burg == nil {
			continue
		}
		g.gdMakeReal(d, o)
		if st.Read[d] != "" {
			g.gdFirst("omen_true", "saw a read omen come true: "+o.Kind+".")
		}
	}
}

func (g *Game) gdMakeReal(d int, o gdOmenT) {
	g.wsAdvance()
	hour := o.At / 60
	b := o.Burg
	var e *wsEvent
	switch o.Kind {
	case "plague":
		e = g.wsAddEvent(&wsEvent{Kind: "plague", P: b.P, R: 1, Str: 3, Start: hour, End: hour + 24*14, Name: "Plague in " + b.Name})
	case "famine":
		if b.State > 0 {
			e = g.wsAddEvent(&wsEvent{Kind: "famine", P: b.P, Realm: b.State, Str: 2, Start: hour, End: hour + 24*20, Name: "Famine in " + g.realmName(b.State)})
		}
	case "bandits":
		e = g.wsAddEvent(&wsEvent{Kind: "bandits", P: b.P, R: 1, Str: 3, Start: hour, Realm: b.State, Name: "Bandits gather near " + b.Name})
	case "boom":
		e = g.wsAddEvent(&wsEvent{Kind: "boom", P: b.P, R: 3, Str: 2, Start: hour, End: hour + 24*12, Realm: b.State, Name: "Trade booms in " + b.Name})
	case "war":
		if foes := g.warsOf(b.State); len(foes) > 0 {
			r := g.gdRng("omenwar", d)
			g.wsSpawnArmy(d, b.State, foes[r.Int(0, len(foes)-1)], r, hour)
			return
		}
		e = g.wsAddEvent(&wsEvent{Kind: "bandits", P: b.P, R: 1, Str: 3, Start: hour, Realm: b.State, Name: "Deserters turn raider near " + b.Name})
	}
	if e != nil {
		g.wsLog(hour, e.Name)
	}
}

func (g *Game) gdOmenReading(d int, o gdOmenT) string {
	what := map[string]string{"plague": "sickness", "war": "war", "famine": "hunger", "bandits": "raiders on the roads", "boom": "good trade", "storm": "a great storm"}[o.Kind]
	place := "in the land"
	if o.Burg != nil {
		place = "near " + o.Burg.Name
	}
	txt := fmt.Sprintf("\"The sign: %s. It foretells %s %s, within %d days.\"", o.Sign, what, place, o.Lead)
	if o.Kind == "storm" {
		hit := "the sky will stay quiet"
		if p := g.gdParasang(); p >= 0 {
			for h := 0; h <= 72; h += 6 {
				if wx := g.W.WeatherAt(p, g.Turn+h*60); wx.Kind != nil && (wx.Kind.Bolt || wx.Kind.ID == "blizzard") {
					hit = fmt.Sprintf("%s comes in about %d hours", strings.ToLower(wx.Kind.Name), h)
					break
				}
			}
		}
		txt = fmt.Sprintf("\"The sign: %s. Here, %s.\"", o.Sign, hit)
	} else if !o.Truth && g.gdRng("read", d).Chance(0.3+0.15*float64(g.Skills["lore"])) {
		txt += " \"...though it rings hollow to me. I would not stake much on it.\""
	}
	return txt
}

func gdReadMenu(g *Game, a *Actor) {
	st := g.gd()
	g.Popup = Wrap("\"The sky and the beasts speak, if one has the patience.\"", 100)
	g.Opts = nil
	today := g.Day()
	n := 0
	for d := today; d >= maxi(0, today-2); d-- {
		o := g.gdOmen(d)
		if !o.Has {
			continue
		}
		n++
		d := d
		if t := st.Read[d]; t != "" {
			g.Opts = append(g.Opts, Option{"{{K|Remind me what the sign of day " + fmt.Sprint(d+1) + " meant.}}", func() { g.talkMenu(t) }})
			continue
		}
		cost := 25 + 10*o.Lead
		g.Opts = append(g.Opts, Option{fmt.Sprintf("Read the sign of day %d {{K|(%s)}}", d+1, g.gdPrice(cost)), func() {
			if !g.gdPay(cost) {
				g.talkMenu("\"Coin first. Omens are hard work.\"")
				return
			}
			t := g.gdOmenReading(d, g.gdOmen(d))
			st.Read[d] = t
			g.gdFirst("omen_read", "had an omen read by "+a.Name+".")
			g.talkMenu(t)
		}})
	}
	if n == 0 {
		g.Popup = Wrap("\"Nothing in the sky or the beasts these three days. Come back when the signs do.\"", 100)
	}
	g.Opts = append(g.Opts, Option{"Enough.", func() { g.talkMenu(g.greeting(a)) }})
	g.Sel = 0
}

// gdOmenLines: the signs you may have seen in the last two days (for the sidebar and news).
func (g *Game) gdOmenLines() []string {
	if (g.Genre != "fantasy" && g.Genre != "mythic") || g.W == nil || g.Cur == nil || g.Cur.Z != 0 {
		return nil
	}
	var out []string
	day := g.Day()
	st := g.gd()
	for d := day; d >= maxi(0, day-1); d-- {
		o := g.gdOmen(d)
		if !o.Has || o.Sky && !gdNight(g.Turn) && d == day {
			continue
		}
		l := "{{Y|Omen:}} " + o.Sign
		if st.Read[d] != "" {
			l += " {{K|(read)}}"
		}
		out = append(out, l)
	}
	return out
}

func gdMythSide(g *Game) []string {
	if g.Genre != "fantasy" && g.Genre != "mythic" {
		return nil
	}
	var out []string
	if p := g.gdPatron(); p > 0 {
		if f := g.gdFaithOf(p); f.Fav != 0 {
			out = append(out, fmt.Sprintf("{{Y|%s}} %+d", g.gdDeity(p), f.Fav))
		}
	}
	if l := g.gdOmenLines(); len(l) > 0 {
		out = append(out, l[0])
	}
	return out
}
