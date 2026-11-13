package engine

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Hall of fame (prefix hf): a score for the run, a compact life summary for the shell's hall, and the
// past lives of the same world coming back as old wanderers, rumours and graves. State in g.ext("hf").
// The shell stores the hall (IndexedDB) and feeds back the lives of this world through HFSetLegends.

// HFLegend is one past life, as the shell hands it back after newGame/load.
type HFLegend struct {
	Name    string   `json:"name"`
	Species string   `json:"species"`
	Calling string   `json:"calling"`
	Year    int      `json:"year"`  // year the life ended
	Cause   string   `json:"cause"` // death text, or "retired"
	Place   string   `json:"place"` // last place (town or zone name)
	Level   int      `json:"level"`
	Score   int      `json:"score"`
	Deeds   []string `json:"deeds"`
}

type hfState struct {
	Legends  []HFLegend `json:"lg,omitempty"`
	GoldPeak int        `json:"gp,omitempty"`
	Y0       int        `json:"y0,omitempty"`
}

func (g *Game) hf() *hfState {
	return g.ext("hf", func() any { return &hfState{} }).(*hfState)
}

func init() {
	turnHooks = append(turnHooks, hfTurn)
	zoneHooks = append(zoneHooks, hfZone)
	talkHooks = append(talkHooks, hfTalk)
}

func hfTurn(g *Game) {
	st := g.hf()
	if st.Y0 == 0 {
		st.Y0 = g.W.Year
	}
	if g.P != nil && g.P.Gold > st.GoldPeak {
		st.GoldPeak = g.P.Gold
	}
}

// HFSetLegends replaces the past lives of this world (at most 8 are used).
func (g *Game) HFSetLegends(l []HFLegend) {
	if len(l) > 8 {
		l = l[:8]
	}
	for i := range l {
		if len(l[i].Deeds) > 4 {
			l[i].Deeds = l[i].Deeds[:4]
		}
	}
	g.hf().Legends = l
}

// hfScore: level, deeds, days, kilometres and kills, scaled by scenario, mode and sandbox.
func (g *Game) hfScore() int {
	if g.Sandboxed() || g.P == nil {
		return 0
	}
	days := g.Turn / DayLen
	km := g.fm().Km
	s := float64(g.P.Level)*100 + float64(len(g.Deeds))*30 + float64(days)*2 + km*0.5 + float64(g.Kills)*2 + float64(g.hf().GoldPeak)/20
	if st := g.stGet(); st != nil {
		for range strings.Split(st.Scen, "+") {
			s *= 1.2
		}
	}
	switch g.GameMode {
	case "roleplay":
		s *= 0.9
	case "wander":
		s *= 0.7
	}
	s *= g.ijScoreMul() // challenge modifiers (challenges.go)
	return int(s)
}

// HFSummary returns the life summary the shell stores in the hall (JSON).
func (g *Game) HFSummary() string {
	st := g.hf()
	days := g.Turn / DayLen
	var deeds []string
	for i := len(g.Deeds) - 1; i >= 0 && len(deeds) < 8; i-- {
		deeds = append(deeds, Plain(g.Deeds[i].Text))
	}
	scen := ""
	if s := g.stGet(); s != nil {
		scen = s.Scen
	}
	y0 := st.Y0
	if y0 == 0 {
		y0 = g.W.Year
	}
	cause := ""
	if g.Dead {
		cause = Plain(g.DeathText)
	}
	b, _ := json.Marshal(map[string]any{
		"score": g.hfScore(), "days": days, "km": int(g.fm().Km), "kills": g.Kills, "deedCount": len(g.Deeds),
		"goldPeak": st.GoldPeak, "deeds": deeds, "family": g.fmSummary(), "cause": cause, "dead": g.Dead,
		"mode": g.GameMode, "scenario": scen, "genotype": g.Genotype, "sandbox": g.Sandboxed(),
		"yearStart": y0, "yearEnd": y0 + days/365, "planet": g.unPlanetName(), "turns": g.Turn,
	})
	return string(b)
}

// ---- old characters in the world ----

func (l *HFLegend) tag() string {
	return fmt.Sprintf("Old %s of %d", strings.ToLower(l.Calling), l.Year)
}

func (g *Game) hfLegendOf(a *Actor) *HFLegend {
	if !a.Has("hf_legend") {
		return nil
	}
	for i := range g.hf().Legends {
		if g.hf().Legends[i].Name == a.Name {
			return &g.hf().Legends[i]
		}
	}
	return nil
}

// hfHome is the surface town an old wanderer settled in.
func (g *Game) hfHome(l *HFLegend) int {
	var ids []int
	for i := range g.W.Burgs {
		if b := &g.W.Burgs[i]; b.Alive && b.Layer == "surface" {
			ids = append(ids, b.ID)
		}
	}
	if len(ids) == 0 {
		return -1
	}
	return ids[NewRng(g.Seed+":hf:home:"+l.Name).Int(0, len(ids)-1)]
}

func hfZone(g *Game, zn *Zone, _ *Rng) {
	if zn.Z != 0 {
		return
	}
	for i := range g.hf().Legends {
		l := &g.hf().Legends[i]
		if zn.Kind == "town" && zn.Burg > 0 && zn.Burg == g.hfHome(l) {
			g.hfPlaceOld(zn, l)
		}
		if l.Place != "" && zn.Name == l.Place && l.Cause != "retired" {
			g.hfPlaceGrave(zn, l)
		}
	}
}

func (g *Game) hfPlaceOld(zn *Zone, l *HFLegend) {
	for _, a := range zn.Actors {
		if a.Has("hf_legend") && a.Name == l.Name {
			return
		}
	}
	bd := g.W.burgByID[zn.Burg]
	if bd == nil {
		return
	}
	r := NewRng(g.Seed + ":hf:npc:" + l.Name + zn.Key)
	x, y, ok := zn.FreeSpot(r, func(x, y int) bool { return zn.At(x, y) == TCobble || zn.At(x, y) == TFloor })
	if !ok {
		return
	}
	a := g.makeNPC("notable", bd, g.W.cultByID[bd.Culture], r)
	a.Name, a.Proper, a.Title, a.F, a.AI = l.Name, true, l.tag(), CYellow, "town"
	a.Desc = fmt.Sprintf("%s, grey and slow, a %s who walked this world in your footsteps long ago.", l.Name, strings.ToLower(l.Species))
	a.X, a.Y, a.HomeX, a.HomeY = x, y, x, y
	a.Level = maxi(a.Level, l.Level)
	a.Set("hf_legend")
	zn.Actors = append(zn.Actors, a)
}

func (g *Game) hfPlaceGrave(zn *Zone, l *HFLegend) {
	r := NewRng(g.Seed + ":hf:grave:" + l.Name)
	x, y, ok := zn.FreeSpot(r, nil)
	if !ok {
		return
	}
	zn.Set(x, y, TInscription)
	zn.Notes[y*MW+x] = fmt.Sprintf("A grave marker: {{Y|%s}}, %s %s. Year %d. \"%s\"", l.Name, l.Species, strings.ToLower(l.Calling), l.Year, hfEpitaph(l))
}

func hfEpitaph(l *HFLegend) string {
	c := strings.TrimSpace(Plain(l.Cause))
	if c == "" || c == "retired" {
		return "Walked far, rested well."
	}
	if len(c) > 140 {
		c = c[:140] + "..."
	}
	return c
}

func hfTalk(g *Game, a *Actor) []Option {
	l := g.hfLegendOf(a)
	if l == nil {
		return nil
	}
	opts := []Option{{"Ask about their life.", func() { g.showPopup(l.tag(), g.hfMonologue(l)) }}}
	if !a.Has("hf_gave") {
		opts = append(opts, Option{"Ask whether they kept anything.", func() {
			a.Set("hf_gave")
			r := NewRng(g.Seed + ":hf:heir:" + l.Name)
			sk := Pick(r, []string{"longblade", "shortblade", "axe", "cudgel"})
			it := g.MakeWeapon(sk, clampi(1+l.Level/6, 1, 4), r)
			noun := it.Name
			if i := strings.LastIndex(noun, " "); i >= 0 {
				noun = noun[i+1:]
			}
			it.Name = l.Name + "'s " + noun
			it.Value *= 2
			g.P.Inv = append(g.P.Inv, it)
			g.Msg("{{Y|" + l.Name + " presses " + it.Name + " into your hands. \"It served me. Let it serve you.\"}}")
		}})
	}
	return opts
}

func (g *Game) hfMonologue(l *HFLegend) string {
	var b strings.Builder
	fmt.Fprintf(&b, "\"I was a level %d %s once, in the year %d.", l.Level, strings.ToLower(l.Calling), l.Year)
	for i, d := range l.Deeds {
		d = strings.TrimSuffix(strings.TrimSpace(Plain(d)), ".")
		if i == 0 {
			b.WriteString(" Here is what I remember: " + lowerFirst(d) + ".")
		} else {
			b.WriteString(" And " + lowerFirst(d) + ".")
		}
	}
	if c := strings.TrimSpace(Plain(l.Cause)); c != "" && c != "retired" {
		b.WriteString(" They say I died: " + lowerFirst(c))
	} else {
		b.WriteString(" Then I put the road down, and no one made me.")
	}
	b.WriteString("\"")
	return b.String()
}

// hfGossip: a rumour about a past wanderer of this world (hook: talk.go gossip()).
func (g *Game) hfGossip(a *Actor) string {
	ls := g.hf().Legends
	if len(ls) == 0 {
		return ""
	}
	r := NewRng(fmt.Sprintf("%d:hf:%d", a.ID, g.Turn/600))
	if !r.Chance(0.3) {
		return ""
	}
	l := ls[r.Int(0, len(ls)-1)]
	s := fmt.Sprintf("\"They still speak of %s, an old %s from %d.", l.Name, strings.ToLower(l.Calling), l.Year)
	if len(l.Deeds) > 0 {
		s += " " + hfCap(lowerFirst(strings.TrimSuffix(Plain(l.Deeds[0]), "."))) + "."
	}
	if l.Place != "" && l.Cause != "retired" {
		s += " Their grave is near " + l.Place + "."
	}
	return s + "\""
}

func hfCap(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}
