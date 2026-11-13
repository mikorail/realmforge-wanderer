package engine

import (
	"fmt"
	"strings"
)

// Personal nemesis (prefix nx). One enemy, drawn from the world's history (a person your realm wronged, a
// slain beast's brood, a villain, a sect elder), is generated lazily on the first turn from world seed +
// player name. They appear every few days in a zone near you with a retinue, taunt you, flee at low HP and
// return scarred and stronger, can be reasoned with (truce, reconciliation), yield (spare -> ally, or kill ->
// unique item + relic) or die to someone else. State g.ext("nx") (planet-local) plus a tiny global
// g.ext("nxg") that carries a living nemesis through rifts. Main key I: the "Hunters" dossier.

// nxRivalDossier is filled by the coordinator (wiring.go) with rival.go's rvDossier; nil = no rival section.
var nxRivalDossier func(g *Game) []string

func init() {
	turnHooks = append(turnHooks, nxTurn)
	enterHooks = append(enterHooks, nxEnter)
	zoneHooks = append(zoneHooks, nxZone)
	killHooks = append(killHooks, nxKill)
	talkHooks = append(talkHooks, nxTalk)
	sideHooks = append(sideHooks, nxSide)
	mainKeys["I"] = func(g *Game) { g.Mode, g.Sel = "nxdossier", 0 }
	modeKeys["nxdossier"] = func(g *Game, k string) {
		switch k {
		case "Escape", "Enter", " ", "I":
			g.Mode = "main"
		case "ArrowDown", "j":
			g.Sel++
		case "ArrowUp", "k":
			g.Sel = maxi(0, g.Sel-1)
		}
	}
	modeDraws["nxdossier"] = func(g *Game) { g.nxDraw() }
}

type nxNem struct {
	Kind     string   `json:"k"` // person monster
	PID      int      `json:"pid,omitempty"`
	MID      int      `json:"mid,omitempty"`
	Self     bool     `json:"self,omitempty"` // the monster itself (not a slain beast's scion)
	Name     string   `json:"n"`
	Epithet  string   `json:"ep"`
	Title    string   `json:"ti,omitempty"`
	Species  string   `json:"sp,omitempty"`
	Glyph    int32    `json:"gl,omitempty"`
	From     string   `json:"fr,omitempty"` // where they come from (town, lair, sect)
	State    int      `json:"st,omitempty"` // realm that posts the bounty
	Burg     int      `json:"bu,omitempty"`
	Reason   string   `json:"re"`
	Year     int      `json:"yr,omitempty"`
	Planet   int      `json:"pl"`
	Origin   string   `json:"or,omitempty"` // planet where the wrong was done
	Followed bool     `json:"fo,omitempty"`
	Scars    []string `json:"sc,omitempty"`
	Enc      int      `json:"en"`
	Flees    int      `json:"fl"`
	Status   string   `json:"s"` // hunt truce spared slain died
	Yield    bool     `json:"y,omitempty"`
	NoFlee   bool     `json:"nf,omitempty"`
	Parleyed bool     `json:"pa,omitempty"`
	Next     int      `json:"nx"`
	TruceTo  int      `json:"tt,omitempty"`
	Truces   int      `json:"tr,omitempty"`
	ActorID  int      `json:"aid,omitempty"`
	LastKey  string   `json:"lk,omitempty"`
	LastName string   `json:"ln,omitempty"`
	LastWX   int      `json:"lx,omitempty"`
	LastWY   int      `json:"ly,omitempty"`
	LastDay  int      `json:"ld,omitempty"`
	Seen     bool     `json:"sn,omitempty"`
	End      string   `json:"end,omitempty"`
}

type nxState struct {
	Init bool   `json:"i"`
	Nem  *nxNem `json:"nem,omitempty"`
}

type nxGlobal struct {
	Carry *nxNem `json:"c,omitempty"`
	Done  bool   `json:"d,omitempty"`
}

func nxSt(g *Game) *nxState  { return g.ext("nx", func() any { return &nxState{} }).(*nxState) }
func nxGl(g *Game) *nxGlobal { return g.ext("nxg", func() any { return &nxGlobal{} }).(*nxGlobal) }

func nxHash(g *Game, parts ...any) uint64 {
	return HashStr(g.W.Seed + ":nx:" + fmt.Sprint(parts...))
}

func nxFill(s string, g *Game, n *nxNem) string {
	home := "your realm"
	if st := g.W.stateByID[g.HomeState]; st != nil && g.HomeState > 0 {
		home = st.Name
	}
	scar := "the mark you left"
	if len(n.Scars) > 0 {
		scar = n.Scars[len(n.Scars)-1]
	}
	return strings.NewReplacer("{N}", n.Name, "{P}", g.P.Name, "{R}", home, "{W}", "the whole world", "{S}", scar).Replace(s)
}

func nxActive(n *nxNem) bool { return n != nil && (n.Status == "hunt" || n.Status == "truce") }

// ---- generation ----

func (g *Game) nxHome() int {
	h := g.HomeState
	if h <= 0 {
		if bd := g.W.burgByID[g.StartBurg]; bd != nil {
			h = bd.State
		}
	}
	return h
}

func (g *Game) nxGenerate() *nxNem {
	w := g.W
	H := g.nxHome()
	home := "your people"
	if s := w.stateByID[H]; s != nil && H > 0 {
		home = s.Name
	}
	type cand struct {
		w int
		n *nxNem
	}
	var cs []cand
	foes := map[int]bool{}
	for _, f := range g.warsOf(H) {
		foes[f] = true
	}
	var hev []EventData
	if H > 0 {
		for _, e := range w.Events {
			if (e.Kind == "capture" || e.Kind == "war" || e.Kind == "fall") && strings.Contains(e.Text, home) {
				hev = append(hev, e)
			}
		}
	}
	mk := func(n *nxNem) *nxNem { n.Planet, n.Origin, n.Status = w.uniIdx, w.Planet, "hunt"; return n }
	for i := range w.People {
		p := &w.People[i]
		if H <= 0 || !p.Alive || p.Burg <= 0 || p.State <= 0 || p.State == H || p.Role == "ruler" || g.vlOf(p.ID) != nil {
			continue
		}
		bd := w.burgByID[p.Burg]
		if bd == nil || !bd.Alive || bd.Layer != "surface" {
			continue
		}
		foe := w.StateName(p.State)
		reason, yr := "", 0
		for _, e := range hev {
			if !strings.Contains(e.Text, bd.Name) && !strings.Contains(e.Text, foe) {
				continue
			}
			yr = e.Y
			switch e.Kind {
			case "capture":
				reason = fmt.Sprintf("Your realm, %s, took their home in the year %d and ended its freedom.", home, e.Y)
			case "war":
				reason = fmt.Sprintf("Your realm, %s, marched against their realm, %s, in the year %d. Their kin died in that war.", home, foe, e.Y)
			default:
				reason = fmt.Sprintf("Their realm, %s, fell in the year %d, and yours was among the victors.", foe, e.Y)
			}
			break
		}
		if reason == "" && foes[p.State] {
			yr = w.Year
			reason = fmt.Sprintf("Their realm, %s, has been at war with yours, %s, for as long as they can remember. Kin of theirs died in it.", foe, home)
		}
		if reason == "" {
			continue
		}
		cs = append(cs, cand{1 + p.Martial/4 + p.Renown/40, mk(&nxNem{Kind: "person", PID: p.ID, Name: p.Name, Title: nonEmpty(p.Title, p.Role), Species: p.Species,
			State: p.State, Burg: p.Burg, From: bd.Name, Reason: reason, Year: yr})})
	}
	for _, v := range g.vlList() {
		if vlDead(g, v.P.ID) {
			continue
		}
		p := v.P
		if H > 0 && v.Burg.State == H {
			yr := w.Year - 6 - int(HashStr(p.Name)%40)
			cs = append(cs, cand{3 + p.Martial/5, mk(&nxNem{Kind: "person", PID: p.ID, Name: p.Name, Title: vlTitle(v), Species: p.Species, State: v.Burg.State, Burg: v.Burg.ID,
				From: v.Burg.Name, Year: yr, Reason: fmt.Sprintf("Your realm's courts branded them for %s in the year %d and drove them into the hills. They blame %s for every day since.", bhCrime(p), yr, home)})})
		} else if foes[v.Burg.State] {
			cs = append(cs, cand{2 + p.Martial/5, mk(&nxNem{Kind: "person", PID: p.ID, Name: p.Name, Title: vlTitle(v), Species: p.Species, State: v.Burg.State, Burg: v.Burg.ID,
				From: v.Burg.Name, Year: w.Year, Reason: fmt.Sprintf("Their realm is at war with yours, %s, and they have made a trade of your people's grief.", home)})})
		}
	}
	sx, sy := g.StartX, g.StartY
	for i := range w.Monsters {
		m := &w.Monsters[i]
		if H <= 0 || m.Layer != "surface" || m.P < 0 || m.P >= len(w.State) {
			continue
		}
		glyph := int32('H')
		if m.Species != "" {
			glyph = []rune(strings.ToUpper(m.Species))[0]
		}
		if m.Slain {
			by := m.SlainBy
			var sp *PersonData
			for j := range w.People {
				if by != "" && w.People[j].Name == by && w.People[j].State == H {
					sp = &w.People[j]
					break
				}
			}
			if sp != nil || by != "" && strings.Contains(by, home) {
				cs = append(cs, cand{5, mk(&nxNem{Kind: "monster", MID: m.ID, Name: "Scion of " + m.Name, Species: m.Species, Glyph: glyph, From: nonEmpty(m.Lair, w.RegionName(m.P)),
					Year: m.SlainYear, Reason: fmt.Sprintf("%s of %s slew %s in the year %d. The brood remembers, and it has grown.", nonEmpty(by, "A champion"), home, m.Name, m.SlainYear)})})
			}
			continue
		}
		mx, my := m.P%w.W, m.P/w.W
		if m.Immortal || m.Power > 70 || maxi(abs(mx-sx), abs(my-sy)) > 40 {
			continue
		}
		yr := w.Year - 8 - int(HashStr(m.Name)%60)
		cs = append(cs, cand{1, mk(&nxNem{Kind: "monster", MID: m.ID, Self: true, Name: m.Name, Species: m.Species, Glyph: glyph, From: nonEmpty(m.Lair, w.RegionName(m.P)), Year: yr,
			Reason: fmt.Sprintf("Settlers of %s burned out its lair in the year %d, and its young died in the fire.", home, yr)})})
	}
	if H > 0 {
		for _, s := range g.Sects() {
			if s.State != H || len(s.Elders) == 0 {
				continue
			}
			var bd *BurgData
			for _, h := range s.Halls {
				if b := w.burgByID[h.Burg]; b != nil && b.Alive {
					bd = b
					break
				}
			}
			if bd == nil {
				continue
			}
			yr := w.Year - 5 - int(HashStr(s.ID)%50)
			el := s.Elders[int(HashStr(s.ID+"nx")%uint64(len(s.Elders)))]
			cs = append(cs, cand{2, mk(&nxNem{Kind: "person", Name: el, Title: "elder of " + s.Short(), Burg: bd.ID, State: 0, From: s.HQName, Year: yr,
				Reason: fmt.Sprintf("In the year %d the levies of %s burned the hall of %s at %s. The elder swore revenge over the ashes.", yr, home, s.Name, s.HQName)})})
		}
	}
	r := NewRng(fmt.Sprintf("%s:nx:pick:%d", w.Seed, HashStr(g.P.Name)))
	var n *nxNem
	if len(cs) > 0 {
		tot := 0
		for _, c := range cs {
			tot += c.w
		}
		x := r.Int(0, tot-1)
		for _, c := range cs {
			if x < c.w {
				n = c.n
				break
			}
			x -= c.w
		}
	}
	if n == nil { // no recorded grievance: the best-known fighter of another realm
		var best *PersonData
		for i := range w.People {
			p := &w.People[i]
			bd := w.burgByID[p.Burg]
			if p.Alive && p.State != H && p.Role != "ruler" && bd != nil && bd.Alive && bd.Layer == "surface" && g.vlOf(p.ID) == nil && (best == nil || p.Martial+p.Renown/10 > best.Martial+best.Renown/10) {
				best = p
			}
		}
		if best == nil {
			return nil
		}
		n = mk(&nxNem{Kind: "person", PID: best.ID, Name: best.Name, Title: nonEmpty(best.Title, best.Role), Species: best.Species, State: best.State, Burg: best.Burg,
			From: w.burgByID[best.Burg].Name, Year: w.Year, Reason: "Their kin fell in the border wars of your age, and they never forgave your people."})
	}
	n.Epithet = nxKillTitles[r.Int(0, len(nxKillTitles)-1)]
	n.Next = DayOf(g.Turn) + 2 + r.Int(0, 2)
	if n.PID > 0 && g.vlOf(n.PID) != nil {
		g.vl().Dead[n.PID] = w.Year // the villain left their hideout to hunt you (villains.go)
	}
	return n
}

// nxEnsure creates (or inherits through a rift) the nemesis once per planet.
func nxEnsure(g *Game) *nxState {
	st := nxSt(g)
	if st.Init || g.P == nil || g.Cur == nil || g.W == nil {
		return st
	}
	st.Init = true
	gl := nxGl(g)
	if c := gl.Carry; c != nil && nxActive(c) && c.Planet != g.W.uniIdx {
		n := *c
		n.Scars = append([]string(nil), c.Scars...)
		n.Origin = c.Origin
		n.Followed, n.Planet = true, g.W.uniIdx
		n.PID, n.MID, n.State, n.ActorID, n.LastKey = 0, 0, 0, 0, ""
		n.Burg = g.StartBurg
		n.Parleyed = false
		n.Next = DayOf(g.Turn) + 2
		if n.Status == "truce" && n.TruceTo < n.Next {
			n.Status = "hunt"
		}
		st.Nem = &n
		g.Msg("{{R|The rift is no barrier to a long grudge. " + n.Name + " has followed you.}}")
		g.deed("nemesis", fmt.Sprintf("%s followed %s through the rift to %s.", n.Name, g.P.Name, g.W.Planet), 0, 0)
		return st
	}
	if gl.Done || gl.Carry != nil {
		return st
	}
	st.Nem = g.nxGenerate()
	if st.Nem != nil {
		g.Msg("{{R|Somewhere a name is spoken against yours. (I: Hunters)}}")
	}
	return st
}

// ---- strength and bodies ----

func nxLevel(g *Game, n *nxNem) int {
	return clampi(g.P.Level+1+n.Enc/2+len(n.Scars)+g.Kills/60, 2, g.P.Level+10)
}

func nxBuild(g *Game, n *nxNem, r *Rng) *Actor {
	lvl := nxLevel(g, n)
	var a *Actor
	if n.Kind == "monster" {
		a = g.Spawn("horror", lvl, r)
		if n.Glyph != 0 {
			a.Glyph = rune(n.Glyph)
		}
		a.F = CRed
		a.MaxHP += lvl * 4
	} else {
		bd := g.W.burgByID[n.Burg]
		if bd == nil {
			for i := range g.W.Burgs {
				if g.W.Burgs[i].Alive && g.W.Burgs[i].Layer == "surface" {
					bd = &g.W.Burgs[i]
					break
				}
			}
		}
		if bd == nil {
			a = g.Spawn("chief", lvl, r)
		} else {
			a = g.makeNPC("guard", bd, g.W.cultByID[bd.Culture], r)
			if n.Species != "" {
				a.Species = n.Species
			}
			tier := clampi(lvl/3, 1, 5)
			if p := a.PartFor("hand"); p != nil {
				p.Item = g.MakeWeapon(Pick(r, []string{"longblade", "axe", "shortblade"}), tier, r)
			}
			if p := a.PartFor("body"); p != nil {
				p.Item = g.MakeArmor("body", tier, r)
			}
			a.MaxHP = 30 + lvl*7
			a.Attr[STR] += lvl / 3
			a.Attr[AGI] += lvl / 4
			a.Attr[TOU] += lvl / 3
		}
		a.F = CMagenta
	}
	a.Level = lvl
	a.Name, a.Proper, a.Role, a.Title, a.Faction, a.State = n.Name, true, "", n.Epithet, "nemesis", 0
	a.AI, a.Angry, a.Boss = "hostile", true, false
	a.Desc = fmt.Sprintf("%s %s, who has sworn to end you. %s", n.Name, n.Epithet, n.Reason)
	for _, sname := range n.Scars {
		for _, sc := range nxScars {
			if sc.Name == sname {
				a.MaxHP += a.MaxHP * sc.HP / 100
				a.AVNat += sc.AV
				a.DVNat += sc.DV
				if sc.Flag != "" {
					a.Set(sc.Flag)
				}
			}
		}
	}
	a.HP = a.MaxHP
	a.XP = 200 + lvl*40
	a.Gold = 60 + lvl*10
	a.Set("nx")
	return a
}

func nxFindActor(g *Game, n *nxNem) *Actor {
	if g.Cur == nil {
		return nil
	}
	for _, a := range g.Cur.Actors {
		if a.Has("nx") && a.HP > 0 && (n.ActorID == 0 || a.ID == n.ActorID) {
			return a
		}
	}
	return nil
}

func nxDropEscorts(g *Game, zn *Zone) {
	out := zn.Actors[:0]
	for _, a := range zn.Actors {
		if !a.Has("nxe") {
			out = append(out, a)
		}
	}
	zn.Actors = out
}

func nxCanHost(zn *Zone) bool {
	return zn != nil && zn.Kind != "town" && zn.Kind != "sea" && !zn.Underwater && !zn.Sky
}

func nxNote(g *Game, n *nxNem, zn *Zone) {
	n.LastKey, n.LastName, n.LastWX, n.LastWY, n.LastDay = zn.Key, zn.Name, zn.WX, zn.WY, DayOf(g.Turn)
}

// nxAppear puts the nemesis and their retinue into zn near the player.
func nxAppear(g *Game, zn *Zone) bool {
	st := nxSt(g)
	n := st.Nem
	if !nxActive(n) || n.Status != "hunt" || DayOf(g.Turn) < n.Next || !nxCanHost(zn) || nxFindActor(g, n) != nil {
		return false
	}
	for _, z := range g.Zones { // they are one person: take them out of wherever they were last
		if z == zn {
			continue
		}
		for _, a := range z.Actors {
			if a.Has("nx") {
				g.removeActor(z, a)
				nxDropEscorts(g, z)
				break
			}
		}
	}
	r := NewRng(fmt.Sprintf("%s:nx:app:%d:%d", g.W.Seed, n.Enc, n.Flees))
	far := func(x, y int) bool { return abs(x-g.P.X) >= 10 || abs(y-g.P.Y) >= 7 }
	x, y, ok := zn.FreeSpot(r, far)
	if !ok {
		if x, y, ok = zn.FreeSpot(r, nil); !ok {
			return false
		}
	}
	a := nxBuild(g, n, r)
	a.X, a.Y, a.HomeX, a.HomeY = x, y, x, y
	zn.Actors = append(zn.Actors, a)
	k := clampi(1+n.Enc/3+a.Level/12, 1, 4)
	for i := 0; i < k; i++ {
		e := g.Spawn(Pick(r, nxEscortTpl), maxi(2, a.Level-3), r)
		e.Faction, e.Angry, e.AI = "nemesis", true, "hostile"
		e.Desc = "Sworn to " + n.Name + "."
		e.Set("nxe")
		e.X, e.Y = a.X+r.Int(-2, 2), a.Y+r.Int(-2, 2)
		g.nudge(zn, e)
		zn.Actors = append(zn.Actors, e)
	}
	n.Enc++
	n.ActorID, n.Parleyed, n.Yield = a.ID, false, false
	n.Next = DayOf(g.Turn) + 3
	nxNote(g, n, zn)
	first := !n.Seen
	n.Seen = true
	g.Msg(fmt.Sprintf("{{R|%s %s steps out of the land and into your road.}}", n.Name, n.Epithet))
	if first {
		g.deed("nemesis", fmt.Sprintf("%s, %s, found %s on the road at last.", n.Name, n.Epithet, g.P.Name), 0, 0)
	}
	return true
}

func nxEnter(g *Game, zn *Zone) {
	st := nxEnsure(g)
	if st.Nem != nil && g.Mode != "dead" {
		nxAppear(g, zn)
	}
}

func nxCheckKinds(g *Game, n *nxNem, kind string, extra int) (string, int, func() bool) {
	dc := clampi(11+n.Flees+n.Enc/3+extra, 10, 17)
	return g.checkLabel(kind, dc), dc, func() bool { return g.roll2d6Check(g.checkMod(kind), dc) }
}

// ---- per-minute behaviour ----

func nxTurn(g *Game) {
	if g.Cur == nil || g.P == nil || g.Dead {
		return
	}
	st := nxEnsure(g)
	n := st.Nem
	if n == nil {
		return
	}
	day := DayOf(g.Turn)
	gl := nxGl(g)
	if nxActive(n) {
		c := *n
		c.Scars = append([]string(nil), n.Scars...)
		gl.Carry = &c
	} else {
		gl.Carry, gl.Done = nil, true
	}
	if n.Status == "truce" && day >= n.TruceTo {
		n.Status, n.Next, n.Parleyed = "hunt", day+1, true
		g.Msg("{{R|The truce with " + n.Name + " has run out.}}")
		if a := nxFindActor(g, n); a != nil {
			a.Angry, a.AI = true, "hostile"
		}
	}
	if n.Status != "hunt" {
		return
	}
	a := nxFindActor(g, n)
	if a == nil {
		if g.Mode == "main" && day >= n.Next && nxHash(g, "t", g.Turn)%90 == 0 {
			nxAppear(g, g.Cur)
		}
		return
	}
	nxNote(g, n, g.Cur)
	if n.Yield {
		return
	}
	if a.HP*100 <= a.MaxHP*30 {
		if !n.NoFlee && (n.Flees < 2 || n.Flees < 4 && nxHash(g, "fl", n.Enc, n.Flees)%3 != 0) {
			nxFlee(g, n, a)
		} else {
			nxYield(g, n, a)
		}
		return
	}
	if !n.Parleyed && g.Mode == "main" && a.Angry && dist(a, g.P) <= 9 && g.visible(a.X, a.Y) {
		nxParley(g, n, a)
	}
}

func nxFlee(g *Game, n *nxNem, a *Actor) {
	g.removeActor(g.Cur, a)
	day := DayOf(g.Turn)
	var sc nxScar
	for i := 0; i < len(nxScars); i++ {
		c := nxScars[int((nxHash(g, "sc", n.Flees)+uint64(i))%uint64(len(nxScars)))]
		dup := false
		for _, s := range n.Scars {
			dup = dup || s == c.Name
		}
		if !dup {
			sc = c
			break
		}
	}
	if sc.Name == "" {
		sc = nxScars[0]
		sc.Name = fmt.Sprintf("old wound number %d", len(n.Scars)+1)
	}
	n.Scars = append(n.Scars, sc.Name)
	n.Flees++
	n.ActorID, n.Next = 0, day+3+int(nxHash(g, "nxt", n.Flees)%4)
	line := Pick(NewRng(fmt.Sprintf("%s:nx:fleeline:%d", g.W.Seed, n.Flees)), nxFleeLines)
	g.Msg("{{R|" + nxFill(line, g, n) + "}}")
	g.Msg("{{K|They will return with " + sc.Name + ".}}")
	g.deed("nemesis", fmt.Sprintf("%s drove off %s, who escaped bearing %s.", g.P.Name, n.Name, sc.Name), 0, 0)
	n.Parleyed = false
}

func nxYield(g *Game, n *nxNem, a *Actor) {
	n.Yield = true
	a.Angry, a.AI = false, "still"
	a.Set("nxy")
	nxDropEscorts(g, g.Cur)
	line := Pick(NewRng(fmt.Sprintf("%s:nx:yield:%d", g.W.Seed, n.Enc)), nxYieldLines)
	g.Msg("{{Y|" + n.Name + " drops to a knee: " + nxFill(line, g, n) + "}} {{K|(bump to speak)}}")
}

// ---- parley ----

func nxParley(g *Game, n *nxNem, a *Actor) {
	n.Parleyed = true
	r := NewRng(fmt.Sprintf("%s:nx:taunt:%d:%d", g.W.Seed, n.Enc, n.Flees))
	pool := nxTaunts
	if len(n.Scars) > 0 && r.Chance(0.5) {
		pool = nxScarTaunts
	}
	text := nxFill(Pick(r, pool), g, n)
	g.Talk, g.Mode, g.Sel = a, "talk", 0
	g.Popup = Wrap(fmt.Sprintf("{{R|%s %s}} (level %d) stands before you. %s\n\n%s", n.Name, n.Epithet, a.Level, n.Reason, text), 100)
	g.Opts = nil
	add := func(t string, f func()) { g.Opts = append(g.Opts, Option{t, f}) }
	close := func() { g.Mode, g.Talk = "main", nil }
	add("{{R|Draw steel.}}", close)
	lbl, _, ok := nxCheckKinds(g, n, "persuasion", 0)
	add(lbl+" \"Hear me out. What was done to you was real, and I will not call it a lie.\"", func() {
		if ok() {
			g.gainXP(15 + a.Level)
			nxTruce(g, n, a, 4+int(nxHash(g, "tr", n.Truces)%6))
			g.Popup = Wrap("{{G|[Persuasion: Success]}}\n\n"+nxFill(Pick(r, nxTruceLines), g, n), 100)
			g.Opts = []Option{{"Go on your way.", close}}
			g.Sel = 0
			return
		}
		g.Popup = Wrap("{{R|[Persuasion: Failure]}}\n\n"+nxFill(Pick(r, nxFailLines), g, n), 100)
		g.Opts = []Option{{"{{R|Fight.}}", close}}
		g.Sel = 0
	})
	lbl2, _, ok2 := nxCheckKinds(g, n, "intimidate", 1)
	add(lbl2+" \"Walk away, or your retinue buries you.\"", func() {
		if ok2() {
			g.gainXP(10 + a.Level)
			g.removeActor(g.Cur, a)
			nxDropEscorts(g, g.Cur)
			n.ActorID, n.Next = 0, DayOf(g.Turn)+5
			g.Msg("{{Y|" + n.Name + " backs away, eyes never leaving you. They will be back.}}")
			close()
			return
		}
		g.Popup = Wrap("{{R|[Intimidate: Failure]}}\n\n"+n.Name+" laughs, and their retinue spreads out.", 100)
		g.Opts = []Option{{"{{R|Fight.}}", close}}
		g.Sel = 0
	})
	cost := 100 + a.Level*25 + n.Enc*40
	if g.P.Gold >= cost {
		add(fmt.Sprintf("{{y|Offer blood-money to settle it.}} {{K|(%d %s)}}", cost, g.Sk.Currency), func() {
			g.P.Gold -= cost
			nxTruce(g, n, a, 3)
			g.Msg("{{Y|" + n.Name + " takes the coin without thanks.}}")
			close()
		})
	}
	add("{{K|Say nothing.}}", close)
}

func nxTruce(g *Game, n *nxNem, a *Actor, days int) {
	n.Status, n.TruceTo = "truce", DayOf(g.Turn)+days
	n.Truces++
	a.Angry, a.AI = false, "wander"
	nxDropEscorts(g, g.Cur)
	g.Msg(fmt.Sprintf("{{Y|Truce with %s for %d days.}}", n.Name, days))
	g.deed("nemesis", fmt.Sprintf("%s won a truce of %d days from %s.", g.P.Name, days, n.Name), 0, 0)
}

// nxSpare ends the feud in peace: the nemesis becomes an ally (companion if there is room).
func nxSpare(g *Game, n *nxNem, a *Actor, how string) {
	n.Status, n.End, n.Yield = "spared", how, false
	a.Angry = false
	delete(a.Flags, "nxy")
	nxDropEscorts(g, g.Cur)
	g.Msg("{{G|" + nxFill(Pick(NewRng(g.W.Seed+":nx:spare"), nxSpareLines), g, n) + "}}")
	if g.companionCount() < g.ijMaxCompanions(maxCompanions) {
		g.joinCompanion(a)
		if a.Comp != nil {
			a.Comp.Loyalty = 85
		}
	} else {
		a.AI = "still"
		g.Msg("{{Y|You have company enough. " + n.Name + " will wait here until there is room.}}")
	}
	r := NewRng(g.W.Seed + ":nx:gift")
	it := g.MakeUnique(g.MakeOddity(r, 5), "Peace-token of "+n.Name, "Pressed into your hand by "+n.Name+", once your sworn enemy.")
	g.give(it)
	g.gainXP(150 + a.Level*20)
	g.deed("nemesis", fmt.Sprintf("%s and %s, once sworn enemies, made an end of the feud (%s).", g.P.Name, n.Name, how), 0, 0)
}

func nxTalk(g *Game, a *Actor) []Option {
	st := nxSt(g)
	n := st.Nem
	if n == nil || a == nil {
		return nil
	}
	var out []Option
	if a.Has("nx") {
		day := DayOf(g.Turn)
		switch {
		case n.Yield && n.Status == "hunt":
			out = append(out, Option{"{{G|Spare " + n.Name + ".}} {{K|(an ally, if they will have you)}}", func() {
				nxSpare(g, n, a, "spared at your mercy")
				g.Mode, g.Talk = "main", nil
			}}, Option{"{{R|Finish " + n.Name + ".}}", func() {
				n.Yield, n.NoFlee = false, true
				a.Angry, a.AI = true, "hostile"
				delete(a.Flags, "nxy")
				g.Msg("{{R|" + n.Name + " grips their blade once more.}}")
				g.Mode, g.Talk = "main", nil
			}})
		case n.Status == "truce":
			lbl, _, ok := nxCheckKinds(g, n, "persuasion", 3)
			out = append(out, Option{lbl + " Ask them to put the grudge down for good.", func() {
				if ok() {
					nxSpare(g, n, a, "reconciled in a truce")
					g.Mode, g.Talk = "main", nil
					return
				}
				n.TruceTo = day
				g.talkMenu("{{R|[Persuasion: Failure]}}\n\n\"Some things are not mended with talk.\" The truce ends at dusk.")
			}})
		case n.Status == "spared" && a.AI != "follower":
			if g.companionCount() < g.ijMaxCompanions(maxCompanions) {
				out = append(out, Option{"{{G|Travel with me.}}", func() { g.joinCompanion(a); g.Mode = "main"; g.Talk = nil }})
			}
		}
		return out
	}
	if nxActive(n) && g.Cur != nil && g.Cur.Kind == "town" && a.Role != "" && !g.Hostile(a) {
		out = append(out, Option{"Ask about the one who hunts you.", func() { g.talkMenu(g.nxRumour(true)) }})
	}
	return out
}

// ---- kills ----

func nxKill(g *Game, d, src *Actor) {
	if !d.Has("nx") {
		return
	}
	n := nxSt(g).Nem
	if n == nil {
		return
	}
	byUs := src == g.P || src != nil && src.AI == "follower"
	switch {
	case n.Status == "spared":
		n.Status, n.End = "died", "fell as your ally"
		g.deed("nemesis", fmt.Sprintf("%s, once a nemesis and later a friend, fell at %s.", n.Name, d.Name), 0, 0)
	case byUs:
		n.Status, n.End = "slain", "slain by you"
		lvl := d.Level
		r := NewRng(g.W.Seed + ":nx:reward")
		tier := clampi(lvl/6+1, 2, 5)
		var base *Item
		var name string
		if r.Chance(0.5) {
			base = g.MakeWeapon(Pick(r, []string{"longblade", "axe", "shortblade", "cudgel"}), tier, r)
			name = n.Name + "'s Grudge"
		} else {
			base = g.MakeArmor(Pick(r, []string{"body", "head", "hands", "back"}), tier, r)
			name = n.Name + "'s Mantle"
		}
		u := g.MakeUnique(base, name, "Taken from "+n.Name+" "+n.Epithet+" after a feud that began: "+n.Reason)
		rel := g.MakeUnique(g.MakeOddity(r, 5), "Relic of "+n.Name, "What the long grudge left behind.")
		for _, it := range []*Item{u, rel} {
			it.X, it.Y = d.X, d.Y
			g.Cur.Items = append(g.Cur.Items, it)
		}
		g.gainXP(300 + lvl*40)
		if n.Self && n.MID > 0 {
			g.Slain[n.MID] = true
		}
		nxDropEscorts(g, g.Cur)
		g.Msg("{{W|" + n.Name + " " + n.Epithet + " is dead. The long grudge ends with you.}}")
		g.deed("nemesis", fmt.Sprintf("%s ended the feud with %s %s, slain after %d meetings.", g.P.Name, n.Name, n.Epithet, n.Enc), 0, 0)
	default:
		n.Status, n.End = "died", "fell to another hand"
		g.Msg("{{Y|" + n.Name + " is dead, and not by your hand.}}")
		g.deed("nemesis", fmt.Sprintf("%s %s, who hunted %s, died to another hand.", n.Name, n.Epithet, g.P.Name), 0, 0)
	}
	n.ActorID = 0
	nxGl(g).Carry, nxGl(g).Done = nil, true
}

// ---- world traces ----

// nxZone: taunt carvings in zones near where the nemesis was last seen.
func nxZone(g *Game, zn *Zone, _ *Rng) {
	n := nxSt(g).Nem
	if n == nil || n.Status != "hunt" || n.LastKey == "" || zn.Key == n.LastKey || zn.Z != 0 || !nxCanHost(zn) {
		return
	}
	if maxi(abs(zn.WX-n.LastWX), abs(zn.WY-n.LastWY)) > 1 {
		return
	}
	side := NewRng(g.W.Seed + ":nx:mark:" + zn.Key)
	if !side.Chance(0.3) {
		return
	}
	if x, y, ok := zn.FreeSpot(side, nil); ok {
		zn.Set(x, y, TInscription)
		zn.Notes[y*MW+x] = nxFill(fmt.Sprintf("Cut into the stone: %s %s WAS HERE. \"{P}, I am a day behind you, and closing.\"", strings.ToUpper(n.Name), n.Epithet), g, n)
	}
}

func nxSide(g *Game) []string {
	n := nxSt(g).Nem
	if n == nil || g.Cur == nil {
		return nil
	}
	if a := nxFindActor(g, n); a != nil && a.Angry {
		return []string{"{{R|Hunted by " + n.Name + "}}"}
	}
	if n.Status == "truce" {
		return []string{fmt.Sprintf("{{y|Truce: %s (%dd) (I)}}", n.Name, maxi(0, n.TruceTo-DayOf(g.Turn)))}
	}
	return nil
}

func (g *Game) nxRumour(inTown bool) string {
	n := nxSt(g).Nem
	if !nxActive(n) {
		return "\"Nothing I'd tell you.\""
	}
	where := "somewhere on the roads"
	if n.LastName != "" {
		ago := DayOf(g.Turn) - n.LastDay
		when := "today"
		if ago > 0 {
			when = fmt.Sprintf("%d days ago", ago)
		}
		where = fmt.Sprintf("at %s, %s", n.LastName, when)
	}
	s := fmt.Sprintf("{{R|%s %s}} asks after a wanderer named %s, last seen %s.", n.Name, n.Epithet, g.P.Name, where)
	if inTown {
		s = "\"" + s + "\" " + n.Reason
		if b := g.nxBounty(g.nxTownState()); b > 0 {
			s += fmt.Sprintf(" The realm has posted %d %s on your head.", b, g.Sk.Currency)
		}
	}
	return s
}

func (g *Game) nxTownState() int {
	if g.Cur != nil && g.W.In(g.Cur.WX, g.Cur.WY) {
		return g.W.State[g.W.Idx(g.Cur.WX, g.Cur.WY)]
	}
	return 0
}

// nxBounty: the price the nemesis's realm has put on the player (hook in bhPlayerBounty, wiring.go).
func (g *Game) nxBounty(state int) int {
	n := nxSt(g).Nem
	if !nxActive(n) || n.State <= 0 || n.State != state {
		return 0
	}
	return 100 + n.Enc*60 + nxLevel(g, n)*8
}

// nxNews: world-news lines (hook in wsNewsLines).
func (g *Game) nxNews() []wsLine {
	n := nxSt(g).Nem
	if !nxActive(n) || !n.Seen {
		return nil
	}
	p := g.W.Idx(clampi(n.LastWX, 0, g.W.W-1), clampi(n.LastWY, 0, g.W.H-1))
	out := []wsLine{{p, g.nxRumour(false)}}
	if b := g.nxBounty(n.State); b > 0 {
		out = append(out, wsLine{p, fmt.Sprintf("{{O|Bounty}} of %d %s posted by %s on the wanderer %s.", b, g.Sk.Currency, g.realmName(n.State), g.P.Name)})
	}
	return out
}

// nxGossip: townsfolk sometimes repeat the rumour (hook in gossip).
func (g *Game) nxGossip(a *Actor) string {
	n := nxSt(g).Nem
	if !nxActive(n) || !n.Seen || !NewRng(fmt.Sprintf("%d:nx:%d", a.ID, g.Turn/600)).Chance(0.3) {
		return ""
	}
	return "\"" + g.nxRumour(false) + "\""
}

// ---- dossier ----

func (g *Game) nxDossier() []string {
	n := nxSt(g).Nem
	var L []string
	if n == nil {
		L = append(L, "{{K|No one has sworn your name as a curse. Yet.}}")
	} else {
		day := DayOf(g.Turn)
		who := n.Title
		if sp := g.W.specByID[n.Species]; sp != nil {
			who = strings.ToLower(sp.Name) + " " + who
		}
		L = append(L, fmt.Sprintf("{{R|%s %s}}", n.Name, n.Epithet), strings.TrimSpace(who+" of "+nonEmpty(n.From, "the wilds")), "")
		L = append(L, "{{W|Why}}", n.Reason)
		if n.Followed {
			L = append(L, "They followed you through the rift from "+n.Origin+".")
		}
		L = append(L, "")
		last := "never seen"
		if n.LastName != "" {
			last = fmt.Sprintf("%s (%s)", n.LastName, map[bool]string{true: "today", false: fmt.Sprintf("%d days ago", day-n.LastDay)}[day == n.LastDay])
		}
		L = append(L, "{{W|Last seen}} "+last)
		lvl := nxLevel(g, n)
		L = append(L, fmt.Sprintf("{{W|Strength}} level %d   {{W|Meetings}} %d   {{W|Escapes}} %d", lvl, n.Enc, n.Flees))
		if len(n.Scars) > 0 {
			L = append(L, "{{W|Marks}} "+strings.Join(n.Scars, "; "))
		}
		st := map[string]string{"hunt": "{{R|Hunting you}}", "truce": fmt.Sprintf("{{y|Truce, %d days left}}", maxi(0, n.TruceTo-day)),
			"spared": "{{G|Your ally}}", "slain": "{{K|Slain by you}}", "died": "{{K|Dead}} (" + n.End + ")"}[n.Status]
		if n.Status == "hunt" && n.Yield {
			st = "{{Y|Beaten and waiting on your mercy}}"
		}
		L = append(L, "{{W|Status}} "+st)
		if b := g.nxBounty(n.State); b > 0 {
			L = append(L, fmt.Sprintf("{{W|Bounty on you}} %d %s, posted by %s", b, g.Sk.Currency, g.realmName(n.State)))
		}
		if n.Status == "hunt" && !n.Yield {
			L = append(L, "", "{{K|Words may buy a truce (Persuasion). Wound them badly and they flee, scarred, or yield.}}")
		}
	}
	if nxRivalDossier != nil {
		if rv := nxRivalDossier(g); len(rv) > 0 {
			L = append(L, "", "{{C|Rival adventurer}}")
			L = append(L, rv...)
		}
	}
	return L
}

func (g *Game) nxDraw() {
	s := g.sc
	x, y, w, h := g.panel("Hunters")
	var rows []string
	for _, l := range g.nxDossier() {
		rows = append(rows, Wrap(l, w)...)
	}
	maxOff := maxi(0, len(rows)-(h-3))
	g.Sel = clampi(g.Sel, 0, maxOff)
	for i := g.Sel; i < len(rows) && i-g.Sel < h-3; i++ {
		s.TextB(x, y+i-g.Sel, rows[i], CGray, CBgDark, x+w)
	}
	s.TextB(x, y+h-2, "{{K|Esc closes   up/down scrolls}}", CGray, CBgDark, x+w)
}
