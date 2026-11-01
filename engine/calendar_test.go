package engine

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestMonthNamesFromCulture(t *testing.T) {
	w := world(t, "ember-crown")
	var cu *CultureData
	for i := range w.Cultures {
		if len(w.Cultures[i].Months) == 12 && len(w.Cultures[i].Feasts) > 0 {
			cu = &w.Cultures[i]
			break
		}
	}
	if cu == nil {
		t.Fatal("bundled world should carry culture months and feasts")
	}
	// day 0 is the first of spring = Realmforge month 2; the last day of the year is month 1
	if d := w.DateOf(cu, 0); d.Name != cu.Months[2] || d.Day != 1 || d.Year != w.Year {
		t.Fatalf("day 0 = %+v, want 1st of %s", d, cu.Months[2])
	}
	if d := w.DateOf(cu, YearDays-1); d.Name != cu.Months[1] {
		t.Fatalf("last day = %s, want %s", d.Name, cu.Months[1])
	}
	if d := w.DateOf(cu, YearDays); d.Year != w.Year+1 || d.Day != 1 {
		t.Fatalf("new year: %+v", d)
	}
	// every month 9 or 10 days; three months per season
	total := 0
	for k := 0; k < 12; k++ {
		ln := monthStart(k+1, 12) - monthStart(k, 12)
		if ln < 9 || ln > 10 {
			t.Fatalf("month %d lasts %d days", k, ln)
		}
		total += ln
		if k%3 == 0 && monthStart(k, 12) != SeasonDays*(k/3) {
			t.Fatalf("month %d starts on %d, not a season start", k, monthStart(k, 12))
		}
	}
	if total != YearDays {
		t.Fatalf("months add up to %d days", total)
	}
	for doy := 0; doy < YearDays; doy++ {
		d := w.DateOf(cu, doy)
		if d.Day < 1 || d.Day > d.Len {
			t.Fatalf("day %d: %+v", doy, d)
		}
	}
	if d := w.DateOf(nil, 0); d.Name != "Seedtide" {
		t.Fatalf("default calendar: %s", d.Name)
	}
	g := newGame(t, nil)
	if l := g.dateLine(); !strings.Contains(l, " of ") {
		t.Fatalf("date line %q", l)
	}
}

// toFeast moves the game clock to the morning of a town's first feast and returns it.
func toFeast(t *testing.T, g *Game) (*BurgData, *FeastData) {
	t.Helper()
	b := g.feastTown()
	if b == nil {
		t.Fatal("not in a town")
	}
	fs := g.W.FeastsOf(b)
	if len(fs) == 0 {
		t.Fatal("town keeps no feasts")
	}
	doy := g.W.FeastDoY(fs[0], g.W.cultByID[b.Culture])
	g.Turn = doy*DayLen + 120
	_, f := g.feastToday()
	if f == nil {
		t.Fatalf("no feast on its own day %d", doy)
	}
	return b, f
}

func TestFeastDayDetection(t *testing.T) {
	g := newGame(t, nil)
	b := g.feastTown()
	cu := g.W.cultByID[b.Culture]
	for _, f := range g.W.FeastsOf(b) {
		doy := g.W.FeastDoY(f, cu)
		if g.W.FeastDoY(f, cu) != doy {
			t.Fatal("feast day not stable")
		}
		if rfMonth(monthOrdOf(doy, 12), 12) != f.Month {
			t.Fatalf("%s (month %d) falls in month %d", f.Name, f.Month, rfMonth(monthOrdOf(doy, 12), 12))
		}
		if got := g.W.FeastOn(b, doy); got == nil {
			t.Fatalf("%s not detected on day %d", f.Name, doy)
		}
		if got := g.W.FeastOn(b, doy+YearDays); got == nil {
			t.Fatalf("%s not detected next year", f.Name)
		}
	}
	_, f := toFeast(t, g)
	feastDays := 0
	for d := 0; d < YearDays; d++ {
		if g.W.FeastOn(b, d) != nil {
			feastDays++
		}
	}
	if feastDays == 0 || feastDays > 3 {
		t.Fatalf("%d feast days a year", feastDays)
	}
	g.FeastNote = ""
	g.feastAnnounce()
	if !strings.Contains(g.Msgs[len(g.Msgs)-1].T, f.Name) {
		t.Fatalf("no feast message: %q", g.Msgs[len(g.Msgs)-1].T)
	}
	n := len(g.Msgs)
	g.feastAnnounce()
	if len(g.Msgs) != n {
		t.Fatal("feast announced twice the same day")
	}
	// decorations show up somewhere in the town
	g.Render()
	decked := false
	for y := 0; y < MH && !decked; y++ {
		for x := 0; x < MW; x++ {
			if gl, _ := feastTile(f, x, y, TDirt, '.', CBrown, 0); gl != '.' {
				decked = true
				break
			}
		}
	}
	if !decked {
		t.Fatal("no festival decorations")
	}
	// wander the decked town: talk, trade, journal tabs, walking (smoke)
	keys := []string{"ArrowLeft", "ArrowDown", "j", "ArrowLeft", "Escape", "ArrowRight", "ArrowUp", "c", "Escape", "ArrowDown", "Enter", "Escape"}
	for i := 0; i < 600 && !g.Dead; i++ {
		g.Key(keys[i%len(keys)])
		g.Render()
	}
}

func townsfolk(g *Game) *Actor {
	for _, a := range g.Cur.Actors {
		if a.HP > 0 && a.Role != "" && a.AI != "follower" && !g.Hostile(a) {
			return a
		}
	}
	return nil
}

func TestJoinFeastOnce(t *testing.T) {
	g := newGame(t, nil)
	b, f := toFeast(t, g)
	a := townsfolk(g)
	if a == nil {
		t.Fatal("no townsfolk")
	}
	if !g.canJoinFeast(a) {
		t.Fatal("cannot join the feast")
	}
	g.openTalk(a)
	found := false
	for _, o := range g.Opts {
		if strings.Contains(o.Text, "Join the") {
			found = true
		}
	}
	if !found {
		t.Fatal("no Join the feast option")
	}
	fac := "state:" + itoa(b.State)
	rep, inv := g.Rep[fac], len(g.P.Inv)
	g.joinFeast(a)
	if b.State > 0 && g.Rep[fac] <= rep {
		t.Fatalf("rep %d → %d", rep, g.Rep[fac])
	}
	if len(g.Feasts) != 1 || g.Feasts[0].Name != f.Name || g.Meal() == "" || len(g.P.Inv) != inv+1 {
		t.Fatalf("feast not joined: %+v meal %q inv %d→%d", g.Feasts, g.Meal(), inv, len(g.P.Inv))
	}
	if g.canJoinFeast(a) {
		t.Fatal("can join the same feast twice")
	}
	r2 := g.Rep[fac]
	g.joinFeast(a)
	if len(g.Feasts) != 1 || g.Rep[fac] != r2 {
		t.Fatal("second join had an effect")
	}
	// the journal shows it; it survives a save
	if l := strings.Join(g.calendarLines(), "\n"); !strings.Contains(l, f.Name) {
		t.Fatal("calendar tab lacks the feast")
	}
	h, err := LoadGame(g.W, g.Save())
	if err != nil {
		t.Fatal(err)
	}
	if !h.FeastSeen[feastKey(b, f, g.W.DateOf(nil, g.Day()).Year)] || len(h.Feasts) != 1 {
		t.Fatal("feast state lost in save")
	}
	// next year it can be joined again
	g.Turn += YearDays * DayLen
	if !g.canJoinFeast(a) {
		t.Fatal("feast not open again next year")
	}
	// merchants stock the festive dish once
	for _, m := range g.Cur.Actors {
		if m.Role == "merchant" {
			n := len(m.Inv)
			g.feastStock(m)
			g.feastStock(m)
			if len(m.Inv) != n+1 {
				t.Fatalf("merchant stock %d → %d", n, len(m.Inv))
			}
			break
		}
	}
	// ordinary day: no feast talk
	g.Turn += DayLen
	if g.W.FeastOn(b, g.Day()) == nil && (g.canJoinFeast(a) || g.feastGreeting(a) != "") {
		t.Fatal("feast talk on an ordinary day")
	}
}

func itoa(n int) string { b, _ := json.Marshal(n); return string(b) }

func TestOldWorldWithoutCalendarLoads(t *testing.T) {
	raw, err := os.ReadFile("../public/worlds/ember-crown.json")
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	if m["kind"] == "wanderer-universe" { // test the start planet as a lone world file
		m = m["planets"].([]any)[0].(map[string]any)
	}
	for _, c := range m["cultures"].([]any) {
		for _, k := range []string{"months", "feasts", "festivals", "music", "architecture", "values"} {
			delete(c.(map[string]any), k)
		}
	}
	for _, s := range m["states"].([]any) {
		delete(s.(map[string]any), "feasts")
	}
	b, _ := json.Marshal(m)
	w, err := ParseWorld(string(b))
	if err != nil {
		t.Fatal(err)
	}
	g := NewGame(w, CharSpec{Name: "Old", Species: "human", Calling: "warrior", Attr: [6]int{18, 16, 16, 14, 14, 14}})
	if g.Cur == nil {
		t.Fatal("no start zone")
	}
	if d := w.DateOf(g.calendarCulture(), g.Day()); d.Name != "Seedtide" {
		t.Fatalf("old world calendar: %s", d.Name)
	}
	if fs := w.FeastsOf(g.feastTown()); len(fs) != len(fallbackFeasts) {
		t.Fatalf("fallback feasts: %v", fs)
	}
	toFeast(t, g)
	g.Render()
	g.calendarLines()
}
