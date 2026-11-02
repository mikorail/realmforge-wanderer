package engine

import (
	"encoding/json"
	"strings"
	"testing"
)

// hzWorld: a fresh ember-crown with hazards injected around the home town.
func hzWorld(t *testing.T, quake, outbreak bool) (*WorldData, *BurgData) {
	t.Helper()
	w := swFresh(t, "ember-crown", false)
	home := swHome(w)
	h := &HazardsData{
		Faults:    []HzFault{{Name: "the Kadu Fault", Parasangs: []int{home.P}}},
		Volcanoes: []HzVolcano{{P: home.P + 1, Name: "Mount Ruthuk", Active: true, Last: w.Year - 12}},
		Diseases: []HzDisease{
			{ID: "pox", Name: "pox", Vector: "contact", Trans: 0.26, Lethal: 0.028, Immunity: 0.95, Incubation: 4, Duration: 8, Hits: []string{"ego", "tou"}, Symptoms: []string{"weeping pustules"}, Cures: []string{"physician", "medicine", "rest"}, Desc: "passed hand to hand"},
			{ID: "pestilence", Name: "plague", Vector: "flea", Trans: 0.3, Lethal: 0.045, Immunity: 0.3, Incubation: 2, Duration: 6, Hits: []string{"tou", "str"}, Symptoms: []string{"swollen black buboes"}, Cures: []string{"physician", "priest", "medicine"}, Desc: "carried by fleas"},
		},
	}
	if quake {
		h.Recent = append(h.Recent, HzRecent{Kind: "quake", Name: "the Kadu Fault", Year: w.Year - 3, P: home.P, Severity: 7.6, Towns: []int{home.ID}})
	}
	if outbreak {
		h.Outbreaks = append(h.Outbreaks, HzOutbreak{Disease: "pox", Name: "the Speckled Pox", Since: w.Year - 1, Parasangs: []int{home.P}, Quarantined: []int{home.ID}})
	}
	w.Hazards = h
	delete(hzIdxCache, w)
	return w, home
}

func hzTownZones(g *Game, bd *BurgData) (town, out *Zone) {
	wx, wy := bd.P%g.W.W, bd.P/g.W.W
	for zy := 0; zy < 3; zy++ {
		for zx := 0; zx < 3; zx++ {
			zn := g.GetZone(wx, wy, zx, zy, 0)
			if zn.Kind == "town" && zn.Burg == bd.ID && (town == nil || zx == 1 && zy == 1) {
				town = zn
			}
			if zn.Kind != "town" && out == nil {
				out = zn
			}
		}
	}
	return
}

func TestHazardsOldWorldUnchanged(t *testing.T) {
	w := swFresh(t, "ember-crown", false)
	if w.Hazards != nil {
		t.Skip("bundled world already carries hazards")
	}
	g := NewGame(w, swSpec())
	for i := 0; i < 600; i++ {
		hzTurn(g)
		g.Turn++
	}
	if hzSt(g).Ill != nil || len(hzSide(g)) != 0 || g.hzGossip(g.P) != "" || len(g.hzNews()) != 0 {
		t.Fatal("a world without hazards must stay quiet")
	}
	// the hazards key is optional in the schema
	var raw map[string]any
	b, _ := json.Marshal(w)
	_ = json.Unmarshal(b, &raw)
	if _, ok := raw["hazards"]; ok {
		t.Fatal("hazards must be omitted when empty")
	}
}

func TestHazardsQuarantineGate(t *testing.T) {
	w, home := hzWorld(t, false, true)
	g := NewGame(w, swSpec())
	town, out := hzTownZones(g, home)
	if town == nil || out == nil {
		t.Skip("no town/outskirts pair")
	}
	if !swNoteWith(town, "QUARANTINE") {
		t.Fatal("quarantined town has no notice")
	}
	if len(swRole(town, "physician")) == 0 {
		t.Fatal("no physician in the sick town")
	}
	sick := 0
	for _, a := range town.Actors {
		if a.Has("hzsick") {
			sick++
		}
	}
	t.Logf("guards at gates: %d, sick townsfolk: %d", func() int {
		n := 0
		for _, a := range town.Actors {
			if a.Has("hzgate") {
				n++
			}
		}
		return n
	}(), sick)
	// the biggest city has walls: its gates get quarantine guards
	var big *BurgData
	for i := range w.Burgs {
		if b := &w.Burgs[i]; b.Alive && b.Layer == "surface" && (big == nil || b.Pop > big.Pop) {
			big = b
		}
	}
	w.Hazards.Outbreaks = append(w.Hazards.Outbreaks, HzOutbreak{Disease: "pestilence", Name: "the Black Death", Since: w.Year, Parasangs: []int{big.P}, Quarantined: []int{big.ID}})
	delete(hzIdxCache, w)
	guards := 0
	for zy := 0; zy < 3; zy++ {
		for zx := 0; zx < 3; zx++ {
			for _, a := range g.GetZone(big.P%w.W, big.P/w.W, zx, zy, 0).Actors {
				if a.Has("hzgate") {
					guards++
				}
			}
		}
	}
	if guards == 0 {
		t.Fatalf("%s: no gate guards", big.Name)
	}
	g.enterZone(out, 40, 12, true)
	g.enterZone(town, 40, 12, true)
	if g.Cur != out {
		t.Fatalf("entered a quarantined town without a pass (now in %s)", g.Cur.Name)
	}
	hzLoc(g).Pass[home.ID] = true
	g.enterZone(town, 40, 12, true)
	if g.Cur != town {
		t.Fatal("pass did not let the player in")
	}
}

func TestHazardsCatchCureImmunity(t *testing.T) {
	w, home := hzWorld(t, false, true)
	g := NewGame(w, swSpec())
	town, _ := hzTownZones(g, home)
	if town != nil {
		g.enterZone(town, g.P.X, g.P.Y, true)
	}
	ob, ch := g.hzExposure()
	if ob == nil || ch <= 0 {
		t.Fatal("no exposure in the outbreak parasang")
	}
	if !g.hzTryCatch(ob, 1, 0) {
		t.Fatal("did not catch with chance 1")
	}
	st := hzSt(g)
	il := st.Ill
	if il == nil || il.ID != "pox" || il.Onset <= g.Turn {
		t.Fatalf("bad illness record %+v", il)
	}
	// incubation, then symptoms
	g.Turn = il.Onset
	hzTurn(g)
	if g.P.St("hzpox") == 0 || g.P.St("hzsick") == 0 {
		t.Fatalf("no symptoms after onset: %v", g.P.Status)
	}
	if side := strings.Join(hzSide(g), " "); !strings.Contains(side, "Sick") {
		t.Fatalf("sidebar silent: %s", side)
	}
	// physician treatment cures at once
	var doc *Actor
	for _, a := range g.Cur.Actors {
		if a.Role == "physician" {
			doc = a
		}
	}
	if doc == nil {
		doc = &Actor{Role: "physician", Tpl: "npc"}
	}
	g.Talk = doc
	g.P.Gold += 1000
	g.cuEarn(1000)
	var treat func()
	for _, o := range hzTalk(g, doc) {
		if strings.HasPrefix(o.Text, "Ask for treatment") {
			treat = o.Do
		}
	}
	if treat == nil {
		t.Fatal("physician offers no treatment")
	}
	treat()
	g.Turn++
	hzTurn(g)
	if st.Ill != nil {
		t.Fatal("still sick after treatment")
	}
	if _, ok := st.Immune["pox"]; !ok || g.P.St("hzpox") != 0 {
		t.Fatal("no immunity / statuses left after recovery")
	}
	if g.hzTryCatch(ob, 0.5, 0.1) {
		t.Fatal("caught the pox again despite immunity")
	}
	// immunity survives save/load
	g2, err := LoadGame(w, g.Save())
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := hzSt(g2).Immune["pox"]; !ok {
		t.Fatal("immunity lost on save/load")
	}
	// medicine tonic
	g.hzInfect("pestilence", "the Black Death")
	g.Turn = hzSt(g).Ill.Onset + 1
	g.hzUseTonic(g.hzMakeCure(false))
	g.Turn += 2
	hzTurn(g)
	if hzSt(g).Ill != nil {
		t.Fatal("cure tonic did not work")
	}
}

func TestHazardsCompanionContagion(t *testing.T) {
	w, _ := hzWorld(t, false, false)
	g := NewGame(w, swSpec())
	c := &Actor{ID: g.newID(), Name: "Friend", Tpl: "npc", AI: "follower", HP: 20, MaxHP: 20, X: g.P.X + 1, Y: g.P.Y}
	g.Cur.Actors = append(g.Cur.Actors, c)
	g.hzInfect("pestilence", "the Black Death")
	st := hzSt(g)
	start := st.Ill.Onset
	for g.Turn = start; g.Turn < start+3000 && st.Comp[c.ID] == nil && st.Ill != nil; g.Turn++ {
		hzTurn(g)
	}
	if st.Comp[c.ID] == nil {
		t.Skip("companion stayed healthy this run")
	}
	il := st.Comp[c.ID]
	g.Turn = il.Onset
	hzTurn(g)
	if c.St("hzsick") == 0 {
		t.Fatal("companion shows no symptoms")
	}
	g.Turn = il.Until
	hzTurn(g)
	if st.Comp[c.ID] != nil {
		t.Fatal("companion never recovered")
	}
}

func TestHazardsQuakeRuins(t *testing.T) {
	w, home := hzWorld(t, true, false)
	g := NewGame(w, swSpec())
	town, _ := hzTownZones(g, home)
	if town == nil {
		t.Skip("no town zone")
	}
	rubble := 0
	for _, tt := range town.T {
		if tt == TRubble {
			rubble++
		}
	}
	if rubble < 10 || !swNoteWith(town, "rubble") {
		t.Fatalf("no ruined district (rubble %d)", rubble)
	}
	hl := swRole(town, "homeless")
	if len(hl) == 0 || !strings.Contains(hl[0].Desc, "Kadu") {
		t.Fatal("no homeless townsfolk from the quake")
	}
	var talk string
	g.Talk = hl[0]
	for _, o := range hzTalk(g, hl[0]) {
		if strings.HasPrefix(o.Text, "What happened") {
			o.Do()
			talk = "ok"
		}
	}
	if talk == "" {
		t.Fatal("homeless have nothing to say")
	}
	news := g.hzNews()
	if len(news) == 0 || !strings.Contains(news[0].Text, "earthquake") {
		t.Fatalf("no quake in the news: %v", news)
	}
	t.Logf("rubble %d, homeless %d: %s | %s", rubble, len(hl), hl[0].Desc, news[0].Text)
}

func TestHazardsWorldParses(t *testing.T) {
	w := swFresh(t, "ember-crown", false)
	raw := `{"faults":[{"name":"the X Fault","parasangs":[1,2]}],"volcanoes":[{"p":3,"name":"Mount Y","active":true,"last":10}],"recent":[{"kind":"eruption","name":"Mount Y","year":5,"p":3,"severity":5.2,"towns":[],"area":[3,4]}],"outbreaks":[],"diseases":[]}`
	var h HazardsData
	if err := json.Unmarshal([]byte(raw), &h); err != nil {
		t.Fatal(err)
	}
	w.Hazards = &h
	delete(hzIdxCache, w)
	if v := w.hzVolcanoAt(3); v == nil || v.Name != "Mount Y" || w.hzFaultAt(2) == nil {
		t.Fatal("index lookups failed")
	}
	if d := w.hzDisease("nothing"); d == nil || d.ID != "nothing" {
		t.Fatal("missing disease profiles need a fallback")
	}
}
