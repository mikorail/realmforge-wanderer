package engine

import (
	"fmt"
	"math"
	"sort"
	"strings"
)

// Festival activities (prefix fv). calendar.go already gives feast days (decorations, greeting, "Join the
// feast" meal, festive dishes); rituals.go the step-by-step rite; ceremonies.go processions; living_world.go
// and society_emergent.go spectacles and fairs staged around NPCs. What was missing is somewhere to GO and
// something to PLAY: on a feast day the centre zone of the town gets a festival grounds (a stage, stalls, a
// contest ring, targets, lanterns or braziers, a sign and a herald, all placed on free tiles with a side rng
// and removed again when the day is over; no new terrain) and up to seven small contests, picked by the
// feast kind (fvKindActs): a footrace, archery and throwing, a wrestling or duelling bout (a real foe actor
// flagged "fvbout", non-lethal by a yield rule, removed when it ends), a cooking contest (reads the
// cookbook and pack, never uses anything up), storytelling (culture pack proverbs and works when present),
// festival market stalls (own tiny screen) and dance and music (games.go instrument when carried).
// Every contest has an entry fee in local money (cuSpend), prizes (cuEarn), a ribbon item, realm and town
// reputation, XP and a once-per-festival-per-year limit. Points per year (3/2/1 for first/second/third)
// make the yearly FESTIVAL CHAMPION title (6 points). Screen: mode "fv"; palette "Festival grounds" and
// "Festival champion"; talk "Ask what is on at the festival" with a herald or innkeeper; fvLine for barks.
// Placement and contests never touch g.R in generation (side rng seed:fv:...); g.R only rolls outcomes.
// State g.ext("fv"), planet-local.

func init() {
	unLocalKeys = append(unLocalKeys, "fv")
	zoneHooks = append(zoneHooks, fvZone)
	enterHooks = append(enterHooks, fvEnter)
	turnHooks = append(turnHooks, fvTurn)
	talkHooks = append(talkHooks, fvTalk)
	modeKeys["fv"] = fvKeys
	modeDraws["fv"] = func(g *Game) { g.fvDraw() }
	paEntries = append(paEntries,
		paEntry{Name: "Festival grounds", Keywords: "festival feast contest race archery bout wrestling cooking story dance market fair", Cat: "Screen", Label: "town",
			Cond: func(g *Game) bool { _, f := g.feastToday(); return f != nil },
			Func: func(g *Game) { g.fvOpen("menu", nil) }},
		paEntry{Name: "Festival champion", Keywords: "festival champion title ribbons points year", Cat: "Screen", Label: "screen",
			Func: func(g *Game) { g.fvOpen("titles", nil) }})
}

// ---------------------------------------------------------------- state

type fvTile struct {
	I int     `json:"i"`
	T Terrain `json:"t"`
}

type fvGround struct {
	Day    int      `json:"d"`
	Tiles  []fvTile `json:"t,omitempty"`
	Notes  []int    `json:"n,omitempty"`
	Herald int      `json:"h,omitempty"`
}

type fvTitle struct {
	Year   int    `json:"y"`
	Town   string `json:"t"`
	Feast  string `json:"f"`
	Points int    `json:"p"`
}

type fvBout struct {
	ID    int    `json:"i"`
	Zone  string `json:"z"`
	Burg  int    `json:"b"`
	Feast string `json:"f"`
	Year  int    `json:"y"`
	Style string `json:"s"`
	Opp   string `json:"o"`
}

type fvState struct {
	Done     map[string]int       `json:"done,omitempty"`   // feast key:activity -> place 1..4
	Grounds  map[string]*fvGround `json:"grounds,omitempty"` // zone key -> what was placed
	Points   map[int]int          `json:"points,omitempty"`  // year -> festival points
	Titles   []fvTitle            `json:"titles,omitempty"`
	TownFame map[int]int          `json:"fame,omitempty"` // burg -> town reputation from contests
	Stall    map[string]int       `json:"stall,omitempty"` // feast key -> bitmask of stall goods bought
	Ribbons  int                  `json:"ribbons,omitempty"`
	Entered  int                  `json:"entered,omitempty"`
	Wins     int                  `json:"wins,omitempty"`
	Bout     *fvBout              `json:"bout,omitempty"`

	mode, res string
	act       *fvAct
	talker    *Actor
	lines     []string
	pendLines []string
	disc      bool
	haggled   bool
}

func (g *Game) fv() *fvState {
	s := g.ext("fv", func() any {
		return &fvState{Done: map[string]int{}, Grounds: map[string]*fvGround{}, Points: map[int]int{}, TownFame: map[int]int{}, Stall: map[string]int{}}
	}).(*fvState)
	if s.Done == nil {
		s.Done = map[string]int{}
	}
	if s.Grounds == nil {
		s.Grounds = map[string]*fvGround{}
	}
	if s.Points == nil {
		s.Points = map[int]int{}
	}
	if s.TownFame == nil {
		s.TownFame = map[int]int{}
	}
	if s.Stall == nil {
		s.Stall = map[string]int{}
	}
	return s
}

// ---------------------------------------------------------------- helpers

func (g *Game) fvYear() int { return g.W.DateOf(nil, g.Day()).Year }

func fvKind(f *FeastData) string {
	if f == nil {
		return "harvest"
	}
	if _, ok := fvKindActs[f.Kind]; ok {
		return f.Kind
	}
	return "harvest"
}

func fvActByID(id string) *fvAct {
	for i := range fvActs {
		if fvActs[i].ID == id {
			return &fvActs[i]
		}
	}
	return nil
}

type fvExtPlay func(g *Game, b *BurgData, f *FeastData, ch int) (mod int, wild bool, pre string, ok bool)

var fvExtPlays = map[string]fvExtPlay{}

// fvRegisterAct adds a custom festival contest from another file (e.g. hunting.go).
func fvRegisterAct(act fvAct, kinds []string, play fvExtPlay) {
	if fvActByID(act.ID) == nil {
		fvActs = append(fvActs, act)
	}
	for _, k := range kinds {
		found := false
		for _, id := range fvKindActs[k] {
			if id == act.ID {
				found = true
				break
			}
		}
		if !found {
			fvKindActs[k] = append(fvKindActs[k], act.ID)
		}
	}
	if play != nil {
		fvExtPlays[act.ID] = play
	}
}

// fvRuns lists the activities running on this feast.
func fvRuns(f *FeastData) []*fvAct {
	var out []*fvAct
	for _, id := range fvKindActs[fvKind(f)] {
		if a := fvActByID(id); a != nil {
			out = append(out, a)
		}
	}
	return out
}

func (g *Game) fvFee(a *fvAct) int { return (6 + maxi(1, g.P.Level)*2) * a.Fee }

func (g *Game) fvDoneKey(b *BurgData, f *FeastData, a *fvAct) string {
	return feastKey(b, f, g.fvYear()) + ":" + a.ID
}

func fvActive(g *Game) (*BurgData, *FeastData) {
	if g.W == nil || g.P == nil || g.Cur == nil {
		return nil, nil
	}
	return g.feastToday()
}

func (g *Game) fvRival(r *Rng, cu *CultureData) string {
	return g.cultureName(cu, r) + " " + Pick(r, fvEpithets)
}

// fvContest plays one contest: legs rolls of 2d6 (or 3d6-3 when wild) plus mod against three rivals whose
// scores come from a side rng (seed, town, year, activity). Place 1..4, result lines.
func (g *Game) fvContest(b *BurgData, f *FeastData, a *fvAct, mod int, wild bool) (int, []string) {
	legs := maxi(1, len(a.Legs))
	if fvKindBonus[fvKind(f)] == a.ID {
		mod++
	}
	score := 0
	var lines []string
	for i := 0; i < legs; i++ {
		var roll int
		if wild {
			roll = g.R.Dice(3, 6) - 3
		} else {
			roll = g.R.Dice(2, 6)
		}
		v := roll + mod
		score += v
		diff := roll - 7
		verdict := "a bad stumble"
		switch {
		case diff >= 3:
			verdict = "very well done"
		case diff >= 0:
			verdict = "steady"
		case diff >= -3:
			verdict = "a slip"
		}
		name := "the contest"
		if i < len(a.Legs) {
			name = a.Legs[i]
		}
		lines = append(lines, fmt.Sprintf("%s: %s (%d)", rsUpper(name), verdict, v))
	}
	r := NewRng(fmt.Sprint(g.Seed, ":fv:rivals:", b.ID, ":", g.fvYear(), ":", a.ID))
	cu := g.W.cultByID[b.Culture]
	zl := maxi(1, g.Cur.Level)
	type rv struct {
		n string
		s int
	}
	rivals := make([]rv, 3)
	place := 1
	for i := range rivals {
		rivals[i] = rv{g.fvRival(r, cu), legs*(7+zl/4) + r.Int(-legs-2, legs+2)}
		if rivals[i].s > score {
			place++
		}
	}
	sort.Slice(rivals, func(i, j int) bool { return rivals[i].s > rivals[j].s })
	lines = append(lines, "", fmt.Sprintf("You scored %d. Best of the field: %s (%d), %s (%d).", score, rivals[0].n, rivals[0].s, rivals[1].n, rivals[1].s))
	return place, lines
}

// fvAward pays out a finished contest: prize money, reputation, XP, a ribbon, points, deed and news.
func (g *Game) fvAward(b *BurgData, f *FeastData, a *fvAct, place int) string {
	st := g.fv()
	st.Done[g.fvDoneKey(b, f, a)] = place
	st.Entered++
	lvl := maxi(1, g.P.Level)
	fee := g.fvFee(a)
	text := fvPlaceText[place]
	prize := []int{0, fee*4 + 12 + lvl*3, fee*2 + 4, fee, 0}[place]
	if prize > 0 {
		text += " {{K|Prize: " + g.cuEarn(prize) + ".}}"
	}
	repN := []int{0, 6, 3, 1, 0}[place]
	if repN > 0 && b.State > 0 {
		g.AddRep(fmt.Sprintf("state:%d", b.State), repN, " for the "+f.Name+" "+a.Short)
	}
	st.TownFame[b.ID] += []int{0, 3, 2, 1, 0}[place]
	g.gainXP(6 + lvl + []int{0, 14, 8, 4, 0}[place])
	if place <= 3 {
		st.Ribbons++
		g.give(&Item{ID: g.newID(), Kind: "trophy", Name: fmt.Sprintf("%s ribbon: %s, %s place", f.Name, strings.TrimPrefix(a.Short, "the "), fvPlaceName[place]),
			Glyph: '~', F: []byte{0, CYellow, CWhite, COrange}[place], Weight: 0, Value: 4 + (4-place)*6, Ident: true, Effect: "fv:ribbon",
			Extra: map[string]int{"fv": g.fvYear()},
			Lore:  fmt.Sprintf("Won at the %s in %s, year %d.", f.Name, b.Name, g.fvYear())})
	}
	pts := []int{0, 3, 2, 1, 0}[place]
	if pts > 0 {
		st.Points[g.fvYear()] += pts
	}
	if place == 1 {
		st.Wins++
		g.deed("festival", fmt.Sprintf("%s won %s at the %s in %s.", g.P.Name, a.Short, f.Name, b.Name), 0, 0)
		lvSt(g).addNews(lvDay(g), b.ID, "%s", fmt.Sprintf("A wanderer named %s won %s at the %s in %s.", g.P.Name, a.Short, f.Name, b.Name))
	}
	if t := g.fvCrown(b, f); t != "" {
		text += "\n\n" + t
	}
	return text
}

// fvCrown names the Festival Champion of the year once the points reach six.
func (g *Game) fvCrown(b *BurgData, f *FeastData) string {
	st := g.fv()
	y := g.fvYear()
	if st.Points[y] < 6 {
		return ""
	}
	for _, t := range st.Titles {
		if t.Year == y {
			return ""
		}
	}
	st.Titles = append(st.Titles, fvTitle{Year: y, Town: b.Name, Feast: f.Name, Points: st.Points[y]})
	if b.State > 0 {
		g.AddRep(fmt.Sprintf("state:%d", b.State), 10, " as Festival Champion")
	}
	g.gainXP(40)
	g.deed("festival", fmt.Sprintf("%s was named Festival Champion of year %d.", g.P.Name, y), 0, 0)
	lvSt(g).addNews(lvDay(g), b.ID, "%s", fmt.Sprintf("%s is Festival Champion of year %d.", g.P.Name, y))
	return fmt.Sprintf("{{Y|You are named Festival Champion of year %d.}}", y)
}

func (g *Game) fvChampion() bool {
	y := g.fvYear()
	for _, t := range g.fv().Titles {
		if t.Year == y || t.Year == y-1 {
			return true
		}
	}
	return false
}

// fvLine is a bark about the festival (barks.go can call it); "" when nothing applies.
func fvLine(g *Game, a *Actor) string {
	if g == nil || g.W == nil || a == nil || g.P == nil || g.Cur == nil {
		return ""
	}
	h := HashStr(fmt.Sprint(g.Seed, ":fv:bark:", a.ID, ":", g.Turn/90))
	if b, f := fvActive(g); f != nil {
		runs := fvRuns(f)
		act := runs[int(h%uint64(len(runs)))]
		lines := []string{
			"Have you seen " + act.Short + "? The prize is worth the entry fee.",
			"Go and enter " + act.Short + ", they say it is open to strangers.",
			"I am backing the local favourite in " + act.Short + ".",
		}
		if g.fvChampion() && h%3 == 0 {
			return "They say the Festival Champion is a wanderer. Is it you?"
		}
		_ = b
		return lines[int(h/3%3)]
	}
	if b := g.feastTown(); b != nil && g.fv().TownFame[b.ID] >= 3 && h%4 == 0 {
		return "I saw you in the ring at the last feast. Good show."
	}
	return ""
}

// ---------------------------------------------------------------- the grounds

func fvFloorLike(t Terrain) bool {
	switch t {
	case TFloor, TGrass, TDirt, TRoad, TCobble, TSand, TWoodFloor, TTileFloor, TSnow, TAsh:
		return true
	}
	return false
}

func fvFree(zn *Zone, x, y int) bool {
	if x < 2 || y < 2 || x > MW-3 || y > MH-3 {
		return false
	}
	if !fvFloorLike(zn.At(x, y)) || zn.ActorAt(x, y) != nil || len(zn.ItemsAt(x, y)) > 0 {
		return false
	}
	_, noted := zn.Notes[y*MW+x]
	return !noted
}

// fvSafe: a blocking piece only goes where the walker keeps at least six open neighbours.
func fvSafe(zn *Zone, x, y int) bool {
	n := 0
	for dy := -1; dy <= 1; dy++ {
		for dx := -1; dx <= 1; dx++ {
			if (dx != 0 || dy != 0) && zn.Walkable(x+dx, y+dy) {
				n++
			}
		}
	}
	return n >= 6
}

func (g *Game) fvLayout(zn *Zone) (x0, y0, w, h int, ok bool) {
	for _, sz := range [][2]int{{25, 11}, {19, 9}, {13, 7}} {
		bw, bh := sz[0], sz[1]
		best := -1.0
		for yy := 2; yy+bh < MH-2; yy += 2 {
			for xx := 2; xx+bw < MW-2; xx += 2 {
				free := 0
				for dy := 0; dy < bh; dy++ {
					for dx := 0; dx < bw; dx++ {
						if fvFree(zn, xx+dx, yy+dy) {
							free++
						}
					}
				}
				frac := float64(free) / float64(bw*bh)
				if frac < 0.72 {
					continue
				}
				sc := frac - math.Hypot(float64(xx+bw/2-MW/2), float64(yy+bh/2-MH/2))/300
				if sc > best {
					best, x0, y0, w, h, ok = sc, xx, yy, bw, bh, true
				}
			}
		}
		if ok {
			return
		}
	}
	return 0, 0, 0, 0, false
}

func (g *Game) fvPlace(zn *Zone, bd *BurgData, f *FeastData) {
	st := g.fv()
	if st.Grounds[zn.Key] != nil || f == nil {
		return
	}
	x0, y0, w, h, ok := g.fvLayout(zn)
	if !ok {
		return
	}
	year := g.W.DateOf(nil, g.Day()).Year
	r := NewRng(fmt.Sprint(g.Seed, ":fv:grounds:", zn.Key, ":", year))
	dr, okd := fvDress[fvKind(f)]
	if !okd {
		dr = fvDress["harvest"]
	}
	gr := &fvGround{Day: g.Day()}
	set := func(x, y int, t Terrain) bool {
		if !fvFree(zn, x, y) {
			return false
		}
		if !t.D().Walk && !fvSafe(zn, x, y) {
			return false
		}
		i := y*MW + x
		gr.Tiles = append(gr.Tiles, fvTile{I: i, T: zn.T[i]})
		zn.Set(x, y, t)
		return true
	}
	// stage on the left
	sx, sy := x0+1, y0+2
	for dy := 0; dy < 3; dy++ {
		for dx := 0; dx < 5; dx++ {
			set(sx+dx, sy+dy, TScaffold)
		}
	}
	set(sx-1, sy-1, dr.Corner)
	set(sx+5, sy-1, dr.Corner)
	// stalls along the top and bottom edges
	for x := x0 + 8; x < x0+w-1; x += 4 {
		set(x, y0, TStall)
		set(x, y0+h-1, TStall)
	}
	// the contest ring
	cx, cy := x0+w*2/3, y0+h/2
	for k := 0; k < 24; k++ {
		ang := float64(k) * math.Pi / 12
		set(cx+int(math.Round(3.2*math.Cos(ang))), cy+int(math.Round(2.2*math.Sin(ang))), dr.Ring)
	}
	// targets at the right end
	for _, ty := range []int{y0 + 2, y0 + h/2, y0 + h - 3} {
		set(x0+w-2, ty, TDummy)
	}
	// corners and spare decoration
	set(x0, y0, dr.Corner)
	set(x0+w-1, y0, dr.Corner)
	set(x0, y0+h-1, dr.Corner)
	set(x0+w-1, y0+h-1, dr.Corner)
	for k := 0; k < 4; k++ {
		set(x0+r.Int(1, w-2), y0+r.Int(1, h-2), dr.Spare)
	}
	// the sign
	var runs []string
	for _, a := range fvRuns(f) {
		runs = append(runs, strings.TrimPrefix(a.Short, "the "))
	}
	sign := fmt.Sprintf("%s %s Today: %s. Entry fees are paid to the herald.", f.Name+".", dr.Sign, strings.Join(runs, ", "))
	if set(sx+2, sy+4, TInscription) {
		i := (sy+4)*MW + sx + 2
		zn.Notes[i] = sign
		gr.Notes = append(gr.Notes, i)
	}
	// the herald
	cu := g.W.cultByID[bd.Culture]
	hd := g.makeNPC("villager", bd, cu, r)
	hd.Title = dr.Herald
	hd.Desc = fmt.Sprintf("A %s of %s in the day's colours, with a staff and a tin horn.", dr.Herald, bd.Name)
	hd.Set("fvherald")
	hd.AI = "still"
	hd.F = CYellow
	px, py := sx+1, sy+4
	if !fvFree(zn, px, py) {
		px, py = -1, -1
	}
	if px >= 0 {
		hd.X, hd.Y, hd.HomeX, hd.HomeY = px, py, px, py
		zn.Actors = append(zn.Actors, hd)
		gr.Herald = hd.ID
	} else if swPut(zn, r, hd, swNear(cx, cy, 8, 5)) {
		gr.Herald = hd.ID
	}
	st.Grounds[zn.Key] = gr
	g.computeLight(zn)
}

// fvClear takes the festival grounds away again.
func (g *Game) fvClear(zn *Zone) {
	st := g.fv()
	gr := st.Grounds[zn.Key]
	if gr == nil {
		return
	}
	for i := len(gr.Tiles) - 1; i >= 0; i-- {
		t := gr.Tiles[i]
		if t.I >= 0 && t.I < len(zn.T) {
			zn.T[t.I] = t.T
		}
	}
	for _, i := range gr.Notes {
		delete(zn.Notes, i)
	}
	for k, a := range zn.Actors {
		if a.ID == gr.Herald && gr.Herald != 0 {
			zn.Actors = append(zn.Actors[:k], zn.Actors[k+1:]...)
			break
		}
	}
	delete(st.Grounds, zn.Key)
	g.computeLight(zn)
}

func fvCentre(zn *Zone) bool {
	return zn != nil && zn.Z == 0 && zn.Kind == "town" && zn.Burg > 0 && zn.ZX == 1 && zn.ZY == 1
}

// fvSync makes the zone match the day: grounds on a feast day, none otherwise.
func (g *Game) fvSync(zn *Zone) {
	if !fvCentre(zn) || g.W == nil {
		return
	}
	bd := g.W.burgByID[zn.Burg]
	if bd == nil || !bd.Alive {
		return
	}
	f := g.W.FeastOn(bd, g.Day())
	gr := g.fv().Grounds[zn.Key]
	if gr != nil && (f == nil || gr.Day != g.Day()) {
		g.fvClear(zn)
		gr = nil
	}
	if gr == nil && f != nil {
		g.fvPlace(zn, bd, f)
	}
}

func fvZone(g *Game, zn *Zone, _ *Rng) { g.fvSync(zn) }

func fvEnter(g *Game, zn *Zone) {
	if g.W == nil || zn == nil {
		return
	}
	g.fvSync(zn)
	st := g.fv()
	for k := 0; k < len(zn.Actors); k++ {
		a := zn.Actors[k]
		if a.Has("fvbout") && (st.Bout == nil || st.Bout.ID != a.ID) {
			zn.Actors = append(zn.Actors[:k], zn.Actors[k+1:]...)
			k--
		}
	}
}

// ---------------------------------------------------------------- turn: dawn changes, bouts

func fvTurn(g *Game) {
	if g.W == nil || g.P == nil || g.Cur == nil {
		return
	}
	st := g.fv()
	if g.Turn%60 == 0 {
		g.fvSync(g.Cur)
	}
	if st.pendLines != nil && g.Mode == "main" {
		l := st.pendLines
		st.pendLines = nil
		st.lines = l
		g.fvOpen("result", nil)
		return
	}
	bt := st.Bout
	if bt == nil {
		return
	}
	var foe *Actor
	if g.Cur.Key == bt.Zone {
		for _, a := range g.Cur.Actors {
			if a.ID == bt.ID {
				foe = a
			}
		}
	}
	place := 0
	switch {
	case g.Cur.Key != bt.Zone:
		if z := g.Zones[bt.Zone]; z != nil {
			for k, a := range z.Actors {
				if a.ID == bt.ID {
					z.Actors = append(z.Actors[:k], z.Actors[k+1:]...)
					break
				}
			}
		}
		g.Msg("{{y|You left the ring in the middle of the bout. It is called against you.}}")
		place = 4
	case foe == nil || foe.HP <= 0:
		place = 1
	case foe.HP*100 <= foe.MaxHP*40:
		g.Msg("{{G|" + foe.Name + " raises a hand and yields.}}")
		place = 1
	case g.P.HP*100 <= g.P.MaxHP*30:
		g.Msg("{{y|The umpire stops the bout. " + foe.Name + " steps back.}}")
		place = 4
		if foe.HP*100 <= foe.MaxHP*70 {
			place = 3
		}
	}
	if place == 0 {
		return
	}
	if foe != nil {
		for k, a := range g.Cur.Actors {
			if a == foe {
				g.Cur.Actors = append(g.Cur.Actors[:k], g.Cur.Actors[k+1:]...)
				break
			}
		}
	}
	st.Bout = nil
	b := g.W.burgByID[bt.Burg]
	a := fvActByID("bout")
	if b == nil || a == nil {
		return
	}
	f := &FeastData{Name: bt.Feast}
	if ff := g.W.FeastOn(b, g.Day()); ff != nil {
		f = ff
	}
	f2 := *f
	f2.Name = bt.Feast
	res := g.fvAward(b, &f2, a, place)
	st.pendLines = []string{fmt.Sprintf("%s against %s.", rsUpper(bt.Style), bt.Opp), "", res}
	st.act = a
}

func (g *Game) fvStartBout(b *BurgData, f *FeastData, a *fvAct, style string) {
	st := g.fv()
	r := NewRng(fmt.Sprint(g.Seed, ":fv:bout:", b.ID, ":", g.Day()))
	foe := g.Spawn("bandit", maxi(1, g.P.Level-1), r)
	cu := g.W.cultByID[b.Culture]
	foe.Name = g.cultureName(cu, r)
	foe.Proper = true
	foe.Title = "ring champion"
	foe.Desc = "A broad ring champion in a rope belt, who has won this bout before."
	foe.Faction = g.Cur.Faction
	foe.Role = ""
	foe.Angry, foe.AI = true, "hostile"
	foe.MaxHP = clampi(g.P.MaxHP*8/10, 12, 400)
	foe.HP = foe.MaxHP
	foe.Set("fvbout")
	x, y, ok := g.Cur.FreeSpot(r, func(px, py int) bool { return abs(px-g.P.X) <= 3 && abs(py-g.P.Y) <= 3 && (px != g.P.X || py != g.P.Y) })
	if !ok {
		g.Msg("{{y|There is no room to fight here. Try the open ground of the plaza.}}")
		return
	}
	foe.X, foe.Y, foe.HomeX, foe.HomeY = x, y, x, y
	g.Cur.Actors = append(g.Cur.Actors, foe)
	st.Bout = &fvBout{ID: foe.ID, Zone: g.Cur.Key, Burg: b.ID, Feast: f.Name, Year: g.fvYear(), Style: style, Opp: foe.Name}
	g.Msg("{{W|" + foe.Name + " steps into the ring.}} Win by making them yield (a heavy blow or two), and keep your own footing: if you are badly hurt the umpire ends it.")
}

// ---------------------------------------------------------------- talk

func fvTalk(g *Game, a *Actor) []Option {
	if g.W == nil || g.Cur == nil || g.P == nil || a == nil || a.HP <= 0 || a.Angry || a.AI == "hostile" || a.AI == "follower" {
		return nil
	}
	if !(a.Has("fvherald") || a.Role == "innkeeper" || a.Role == "tavern" || a.Role == "lw_herald" || a.Role == "herald") {
		return nil
	}
	b, f := g.feastToday()
	if f == nil {
		return nil
	}
	return []Option{
		{"Ask what is on at the festival", func() { g.talkMenu(g.fvOnLines(b, f)) }},
		{"Enter a festival contest", func() { g.fvOpen("menu", a) }},
	}
}

func (g *Game) fvOnLines(b *BurgData, f *FeastData) string {
	st := g.fv()
	dr, ok := fvDress[fvKind(f)]
	if !ok {
		dr = fvDress["harvest"]
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("\"It is the %s. %s\"\n\n", f.Name, dr.Sign))
	for _, a := range fvRuns(f) {
		fee := "free"
		if a.Fee > 0 {
			fee = g.cuFormat(g.fvFee(a))
		}
		state := ""
		if p := st.Done[g.fvDoneKey(b, f, a)]; p > 0 {
			state = " (you have taken part)"
		}
		sb.WriteString(fmt.Sprintf("{{W|%s}}: %s, fee %s%s\n", a.Name, a.Intro, fee, state))
	}
	sb.WriteString(fmt.Sprintf("\n\"Six points in a year and you are named Festival Champion. You have %d.\"", st.Points[g.fvYear()]))
	return sb.String()
}

// ---------------------------------------------------------------- screen

func (g *Game) fvOpen(mode string, talker *Actor) {
	s := g.fv()
	s.mode, s.talker = mode, talker
	if mode == "menu" {
		if _, f := g.feastToday(); f == nil {
			g.Msg("There is no festival here today.")
			return
		}
		s.act = nil
	}
	g.Mode = "fv"
}

func fvKeys(g *Game, k string) {
	s := g.fv()
	leave := func() {
		tk := s.talker
		s.talker, s.mode = nil, ""
		if tk != nil && g.Talk != nil && tk.HP > 0 {
			g.Mode = "talk"
			g.talkMenu("\"Anything else?\"")
			return
		}
		g.Mode = "main"
	}
	b, f := g.feastToday()
	switch s.mode {
	case "result":
		if f != nil {
			s.mode, s.act = "menu", nil
		} else {
			leave()
		}
	case "titles":
		if k == "Escape" || k == "q" {
			leave()
		}
	case "stall":
		if k == "Escape" || k == "q" {
			s.mode = "menu"
			return
		}
		if f == nil {
			leave()
			return
		}
		if k == "h" {
			if s.haggled {
				g.Msg("The stallkeeper has already given you their best price.")
				return
			}
			s.haggled = true
			if g.roll2d6Check(g.checkMod("persuasion"), 9) {
				s.disc = true
				g.Msg("{{G|The stallkeeper grins and takes a quarter off.}}")
			} else {
				g.Msg("{{y|The stallkeeper laughs. 'On a feast day? Try again tomorrow.'}}")
			}
			return
		}
		if len(k) == 1 && k[0] >= '1' && k[0] <= '5' {
			g.fvBuy(b, f, int(k[0]-'1'))
		}
	case "menu":
		if k == "Escape" || k == "q" {
			leave()
			return
		}
		if k == "Tab" || k == "t" {
			s.mode = "titles"
			return
		}
		if f == nil {
			leave()
			return
		}
		runs := fvRuns(f)
		if len(k) == 1 && k[0] >= 'a' && k[0] < 'a'+byte(len(runs)) {
			act := runs[int(k[0]-'a')]
			s.act = act
			if act.ID == "market" {
				s.mode, s.disc, s.haggled = "stall", false, false
				return
			}
			if p := g.fv().Done[g.fvDoneKey(b, f, act)]; p > 0 {
				g.Msg("{{y|You have already taken part in " + act.Short + " this year.}}")
				s.act = nil
				return
			}
			if g.fv().Bout != nil {
				g.Msg("{{y|A bout is already under way.}}")
				s.act = nil
				return
			}
			s.mode = "act"
		}
	case "act":
		if k == "Escape" || k == "q" {
			s.mode, s.act = "menu", nil
			return
		}
		if f == nil || s.act == nil {
			leave()
			return
		}
		if len(k) == 1 && k[0] >= 'a' && k[0] <= 'c' && s.act.Opts[int(k[0]-'a')] != "" {
			g.fvPlay(b, f, s.act, int(k[0]-'a'))
		}
	}
}

func (g *Game) fvBuy(b *BurgData, f *FeastData, i int) {
	st := g.fv()
	key := feastKey(b, f, g.fvYear())
	if st.Stall[key]&(1<<uint(i)) != 0 {
		g.Msg("{{y|You have bought that already today.}}")
		return
	}
	price := fvStalls[i].Price
	if st.disc {
		price = price * 3 / 4
	}
	if !g.cuSpend(price) {
		g.Msg("{{y|You cannot pay for that.}}")
		return
	}
	var it *Item
	r := NewRng(fmt.Sprint(g.Seed, ":fv:stall:", b.ID, ":", g.Day(), ":", i))
	switch i {
	case 0:
		it = g.feastItem(b, f, 1)
	case 1:
		it = g.MakeFood(r, 2)
	case 2:
		it = g.MakeLight(1)
	case 3:
		it = g.MakeTonic("heal", 1)
	default:
		it = &Item{ID: g.newID(), Kind: "trophy", Name: f.Name + " favour", Glyph: '~', F: CMagenta, Weight: 0, Value: 5, Ident: true, Effect: "fv:favour",
			Lore: fmt.Sprintf("A knotted ribbon from the %s in %s, year %d.", f.Name, b.Name, g.fvYear())}
	}
	g.give(it)
	st.Stall[key] |= 1 << uint(i)
	st.TownFame[b.ID]++
	g.Msg("You buy " + it.Name + ".")
}

// fvHasInstrument and the other mods read other systems without changing them.
func (g *Game) fvBestDish() (*Dish, int) {
	var best *Dish
	bs := -1
	for i := range g.Cookbook {
		d := &g.Cookbook[i]
		sc := len(d.Tags) + mini(3, d.Cooked/2)
		if sc > bs {
			best, bs = d, sc
		}
	}
	return best, bs
}

func (g *Game) fvIngredientTags() (int, int) {
	n, set := 0, map[string]bool{}
	for _, it := range g.P.Inv {
		if it.Kind == "ingredient" {
			n += maxi(1, it.Count)
			if it.Effect != "" {
				set[it.Effect] = true
			}
		}
	}
	return n, len(set)
}

func (g *Game) fvProverb(b *BurgData, r *Rng) string {
	if pk := g.W.cpOf(b.Culture); pk != nil && pk.Voice != nil && len(pk.Voice.Proverbs) > 0 {
		return Pick(r, pk.Voice.Proverbs)
	}
	return Pick(r, fvProverbs)
}

func (g *Game) fvTale(b *BurgData, r *Rng) (string, bool) {
	if pk := g.W.cpOf(b.Culture); pk != nil && pk.Literature != nil && len(pk.Literature.Works) > 0 {
		w := pk.Literature.Works[r.Int(0, len(pk.Literature.Works)-1)]
		line := ""
		if len(w.Lines) > 0 {
			line = w.Lines[0]
		}
		return fmt.Sprintf("You tell \"%s\" by %s. %s", w.Title, nonEmpty(w.Author, "an old hand"), line), true
	}
	var pool []string
	for _, e := range g.W.Events {
		if e.St == b.State && e.Text != "" {
			pool = append(pool, e.Text)
		}
	}
	if len(pool) > 0 {
		return "You tell how it was told to you: " + Pick(r, pool), false
	}
	return "You tell an old tale of the road, as best you remember it.", false
}

// fvPlay runs the chosen option of a contest.
func (g *Game) fvPlay(b *BurgData, f *FeastData, a *fvAct, ch int) {
	s := g.fv()
	r := NewRng(fmt.Sprint(g.Seed, ":fv:play:", b.ID, ":", g.Day(), ":", a.ID, ":", ch))
	mod, wild := 0, false
	var pre string
	switch a.ID {
	case "race":
		switch ch {
		case 0:
			mod = maxi(g.P.M(AGI), g.P.M(TOU)) + g.Skills["endurance"]
		case 1:
			mod = g.checkMod("endurance")
		default:
			mod, wild = g.checkMod("acrobatics"), true
		}
	case "archery":
		switch ch {
		case 0:
			mod = g.P.M(AGI) + 2*g.Skills["missile"]
			for _, it := range g.P.Inv {
				if it.Kind == "ranged" {
					mod++
					break
				}
			}
		case 1:
			mod = maxi(g.P.M(AGI), g.P.M(STR)) + 2*maxi(g.Skills["thrown"], g.Skills["shortblade"])
		default:
			mod, wild = g.checkMod("perception"), true
		}
	case "bout":
		if !g.cuSpend(g.fvFee(a)) {
			g.Msg("{{y|You cannot pay the entry fee.}}")
			return
		}
		style := "wrestling bout"
		if ch == 1 {
			style = "blunt-blade duel"
		}
		g.fvStartBout(b, f, a, style)
		if g.fv().Bout == nil {
			g.cuEarn(g.fvFee(a))
			return
		}
		s.mode, s.talker = "", nil
		g.Mode = "main"
		return
	case "cook":
		switch ch {
		case 0:
			d, sc := g.fvBestDish()
			if d == nil {
				g.Msg("{{y|Your cookbook is empty. Cook something at a fire first.}}")
				return
			}
			mod = g.checkMod("survival") + sc
			_, ftag := fvStyleTag(f)
			for _, t := range d.Tags {
				if t == ftag {
					mod += 2
				}
			}
			pre = "You present " + d.Name + "."
		case 1:
			n, tags := g.fvIngredientTags()
			if n < 2 {
				g.Msg("{{y|You need at least two ingredients in your pack to improvise.}}")
				return
			}
			mod, wild = g.checkMod("survival")+mini(3, tags)-1, true
			pre = "You improvise a dish from what is in your pack."
		default:
			name, _ := g.feastDish(b, f)
			found := false
			for _, it := range g.P.Inv {
				if it.Kind == "dish" && it.Name == name {
					found = true
				}
			}
			if !found {
				g.Msg("{{y|You carry no " + name + ". Buy one at the stalls or join the feast.}}")
				return
			}
			mod = g.checkMod("survival") + 3
			pre = "You enter the " + name + " of the day."
		}
	case "story":
		switch ch {
		case 0:
			mod = g.checkMod("persuasion")
			pre = "\"" + g.fvProverb(b, r) + "\" you say, and explain it."
		case 1:
			t, packed := g.fvTale(b, r)
			mod = g.checkMod("persuasion")
			if packed {
				mod++
			}
			pre = t
		default:
			mod = g.checkMod("persuasion") + mini(3, len(g.Deeds)/4)
			if len(g.Deeds) == 0 {
				mod--
				pre = "You tell of the road, though there is little to tell yet."
			} else {
				pre = "You tell your own story: " + g.Deeds[len(g.Deeds)-1].Text
			}
		}
	case "dance":
		switch ch {
		case 0:
			it := g.gmInstrument()
			if it == nil {
				g.Msg("{{y|You carry no instrument.}}")
				return
			}
			mod = g.gmPerfMod(it)
			pre = "You play " + it.Name + " for the dancers."
			if line, bonus := g.pmFestivalLine(b, f, it); line != "" {
				pre = line
				mod += bonus
			}
		case 1:
			mod = g.checkMod("acrobatics")
			if line, _ := g.pmFestivalLine(b, f, nil); line != "" {
				pre = line
			}
		default:
			mod = (g.checkMod("persuasion") + g.checkMod("acrobatics")) / 2
			if line, _ := g.pmFestivalLine(b, f, nil); line != "" {
				pre = line
			}
		}
	default:
		if fn, ok := fvExtPlays[a.ID]; ok {
			m, w, p, goOn := fn(g, b, f, ch)
			if !goOn {
				return
			}
			mod, wild, pre = m, w, p
		}
	}
	if !g.cuSpend(g.fvFee(a)) {
		g.Msg("{{y|You cannot pay the entry fee.}}")
		return
	}
	g.spendTurns(30)
	place, lines := g.fvContest(b, f, a, mod, wild)
	if pre != "" {
		lines = append([]string{pre, ""}, lines...)
	}
	lines = append(lines, "", g.fvAward(b, f, a, place))
	s.lines = lines
	s.mode = "result"
}

func fvStyleTag(f *FeastData) (string, string) {
	tag, dish, _, _, _ := feastStyleOf(f.Kind)
	return dish, tag
}

func (g *Game) fvDraw() {
	sc := g.sc
	s := g.fv()
	b, f := g.feastToday()
	title := "Festival"
	if f != nil {
		title = f.Name
	}
	if s.mode == "titles" {
		title = "Festival champion"
	}
	x, y, w, h := g.panel(title)
	bottom := 2 + 1 + h - 1
	put := func(txt string) {
		for _, ln := range Wrap(txt, w) {
			if y < bottom {
				sc.TextB(x, y, ln, CGray, CBgDark, x+w)
				y++
			}
		}
	}
	foot := func(t string) { sc.TextB(x, bottom, "{{K|"+t+"}}", CGray, CBgDark, x+w) }
	st := g.fv()
	year := g.fvYear()
	switch s.mode {
	case "titles":
		put(fmt.Sprintf("{{W|Year %d}}  festival points: %d of 6 for the title.", year, st.Points[year]))
		put(fmt.Sprintf("Contests entered %d, won %d, ribbons %d.", st.Entered, st.Wins, st.Ribbons))
		y++
		put("{{W|Titles}}")
		if len(st.Titles) == 0 {
			put("{{K|None yet. Win or place in contests at feast days: first place 3 points, second 2, third 1.}}")
		}
		for _, t := range st.Titles {
			put(fmt.Sprintf("{{Y|Festival Champion of year %d}}  %s, at the %s (%d points)", t.Year, t.Town, t.Feast, t.Points))
		}
		y++
		put("{{W|Town fame}}")
		var ids []int
		for id, v := range st.TownFame {
			if v > 0 {
				ids = append(ids, id)
			}
		}
		sort.Ints(ids)
		for _, id := range ids {
			if bd := g.W.burgByID[id]; bd != nil {
				put(fmt.Sprintf("%s: %d", bd.Name, st.TownFame[id]))
			}
		}
		foot("Esc close")
		return
	case "result":
		for _, ln := range s.lines {
			put(ln)
		}
		foot("Press any key")
		return
	}
	if f == nil || b == nil {
		put("{{K|There is no festival here today.}}")
		foot("Esc close")
		return
	}
	dr, ok := fvDress[fvKind(f)]
	if !ok {
		dr = fvDress["harvest"]
	}
	if s.mode == "stall" {
		put("{{W|The " + dr.Stall + "}}  " + f.Name)
		y++
		key := feastKey(b, f, year)
		for i, it := range fvStalls {
			price := it.Price
			if s.disc {
				price = price * 3 / 4
			}
			tail := g.cuFormat(price)
			if st.Stall[key]&(1<<uint(i)) != 0 {
				tail = "bought"
			}
			put(fmt.Sprintf("{{W|%d}}) %s  {{K|%s}}", i+1, it.Label, tail))
		}
		y++
		put("{{K|h haggle (Persuasion, once)  Esc back}}")
		return
	}
	if s.mode == "act" && s.act != nil {
		a := s.act
		put("{{W|" + a.Name + "}}  entry " + g.cuFormat(g.fvFee(a)))
		y++
		put(a.Intro)
		y++
		for i, o := range a.Opts {
			if o == "" {
				continue
			}
			tail := ""
			if k := a.Kinds[i]; k != "" {
				tail = fmt.Sprintf(" {{c|[%s %d%%]}}", checkKinds[k].label, int(chanceFor(g.checkMod(k), 9)*100+0.5))
			}
			put(fmt.Sprintf("{{W|%c}}) %s%s", 'a'+i, o, tail))
		}
		foot("Letter choose  Esc back")
		return
	}
	// the day's list
	put(fmt.Sprintf("{{W|%s}} in %s  {{K|fame %d, points this year %d of 6}}", f.Name, b.Name, st.TownFame[b.ID], st.Points[year]))
	y++
	for i, a := range fvRuns(f) {
		fee := "free"
		if a.Fee > 0 {
			fee = g.cuFormat(g.fvFee(a))
		}
		state := ""
		if p := st.Done[g.fvDoneKey(b, f, a)]; p > 0 {
			state = " {{G|taken part}}"
		}
		put(fmt.Sprintf("{{W|%c}}) %s  {{K|%s}}%s", 'a'+i, a.Name, fee, state))
	}
	foot("Letter choose  Tab titles  Esc close")
}
