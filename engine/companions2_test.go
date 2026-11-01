package engine

import (
	"strconv"
	"strings"
	"testing"
)

func cbCount(g *Game, a *Actor) int {
	n := 0
	tag := "{{c|" + givenOf(a.Name) + ":}}"
	for _, m := range g.Msgs {
		if strings.Contains(m.T, tag) {
			n++
		}
	}
	return n
}

func cbReset(g *Game) {
	g.extLive["cb"] = &cbState{}
}

func TestBanterHomeTownAndCooldown(t *testing.T) {
	g := newGame(t, nil)
	a := recruitOne(t, g)
	home := g.Cur
	far := (home.ZX + 1) % 3
	g.Warp(home.WX+3, home.WY, far, home.ZY, 0) // somewhere else
	g.cb()
	cbReset(g)
	n := cbCount(g, a)
	g.Warp(home.WX, home.WY, home.ZX, home.ZY, 0)
	if g.Cur != home || !hasActor(g.Cur, a) {
		t.Fatalf("did not return home with the companion")
	}
	if cbCount(g, a) != n+1 {
		t.Fatalf("no banter on entering the home town (msgs %d→%d)", n, cbCount(g, a))
	}
	// immediately again: cooldown holds
	g.Warp(home.WX+3, home.WY, far, home.ZY, 0)
	g.Warp(home.WX, home.WY, home.ZX, home.ZY, 0)
	if cbCount(g, a) != n+1 {
		t.Fatalf("banter ignored its cooldown")
	}
	// after the cooldown it may speak again
	g.Turn += cbGap + cbGlobal + 1
	g.Warp(home.WX+3, home.WY, far, home.ZY, 0)
	g.cb().Global = 0
	delete(g.cb().Last, a.ID)
	g.Warp(home.WX, home.WY, home.ZX, home.ZY, 0)
	if cbCount(g, a) <= n+1 {
		t.Errorf("banter never returned after the cooldown")
	}
	// lines use real names, never raw placeholders
	for _, m := range g.Msgs {
		if strings.Contains(m.T, "{{c|") && strings.Contains(m.T, "{") && strings.Count(m.T, "{") > strings.Count(m.T, "{{")*2 {
			t.Errorf("unfilled placeholder: %s", m.T)
		}
	}
}

func TestBanterLinesFill(t *testing.T) {
	g := newGame(t, nil)
	a := recruitOne(t, g)
	vars := map[string]string{"foe": "F", "feast": "F", "feasttown": "T", "weather": "w", "other": "O", "otherrealm": "R", "othertown": "T", "victim": "V"}
	for trig, ls := range cbLines {
		for _, l := range ls {
			s := g.cbFill(a, l.Text, vars)
			if strings.ContainsAny(s, "{}") {
				t.Errorf("%s: unfilled %q", trig, s)
			}
		}
	}
}

func cqWarpTo(g *Game, key string) {
	ps := strings.Split(key, ".")
	v := make([]int, 5)
	for i := range v {
		v[i], _ = strconv.Atoi(ps[i])
	}
	g.Warp(v[0], v[1], v[2], v[3], v[4])
}

func cqPickOpt(t *testing.T, g *Game, sub string) {
	t.Helper()
	for _, o := range g.Opts {
		if strings.Contains(o.Text, sub) {
			o.Do()
			return
		}
	}
	var ts []string
	for _, o := range g.Opts {
		ts = append(ts, o.Text)
	}
	t.Fatalf("no option %q in %v", sub, ts)
}

func TestPersonalQuest(t *testing.T) {
	for _, kind := range []string{"avenge", "heirloom", "peace", "debt"} {
		t.Run(kind, func(t *testing.T) { cqRunKind(t, kind) })
	}
}

func cqRunKind(t *testing.T, kind string) {
	g := newGame(t, nil)
	a := recruitOne(t, g)
	found := false
	for i := 0; i < 60 && !found; i++ {
		a.ID = 200000 + i
		if q := g.cqMake(a); q != nil && q.Kind == kind {
			found = true
		}
	}
	if !found {
		t.Skip("kind not available near this home")
	}
	a.Comp.Loyalty = 70
	a.Comp.Since = g.Turn - 4*DayLen
	g.Turn = (g.Turn/60 + 1) * 60
	g.cqTick()
	q := g.cq().Q[a.ID]
	if q == nil || q.Stage != 0 {
		t.Fatalf("no personal quest offered")
	}
	g.openTalk(a)
	cqPickOpt(t, g, "on your mind")
	cqPickOpt(t, g, "I'll help")
	if q.Stage != 1 || g.cqMirror(q) == nil {
		t.Fatalf("quest not accepted: stage %d", q.Stage)
	}
	t.Logf("%s quest: %s — %s", q.Kind, q.Title, q.Text)
	g.Mode, g.Talk = "main", nil
	cqWarpTo(g, q.Target)
	if !hasActor(g.Cur, a) {
		t.Fatalf("companion did not come along")
	}
	if !q.Placed {
		t.Fatalf("quest target not placed at %s", q.Target)
	}
	attr, hp, av, dv := a.Attr, a.MaxHP, a.AVNat, a.DVNat
	deeds := len(g.Deeds)
	switch q.Kind {
	case "avenge":
		f := g.findActor(q.NPC)
		if f == nil || f.Name != q.Foe {
			t.Fatalf("foe not found")
		}
		g.kill(f, g.P)
	case "heirloom":
		var it *Item
		for _, o := range g.Cur.Items {
			if o.Quest == q.QID {
				it = o
			}
		}
		if it == nil {
			t.Fatalf("heirloom not placed")
		}
		g.Cur.Items = nil
		g.give(it)
		g.cqTick()
		if q.Stage != 2 {
			t.Fatalf("heirloom not noticed")
		}
		g.openTalk(a)
		cqPickOpt(t, g, "It is done")
	case "peace", "debt":
		n := g.findActor(q.NPC)
		if n == nil {
			t.Fatalf("relative/creditor not placed")
		}
		g.P.Gold = 100000
		g.openTalk(n)
		if q.Kind == "peace" {
			cqPickOpt(t, g, "make peace")
		} else {
			cqPickOpt(t, g, "Settle")
		}
	}
	if q.Stage != 3 {
		t.Fatalf("quest not completed: stage %d", q.Stage)
	}
	if a.Comp.Loyalty != 100 {
		t.Errorf("loyalty not rewarded: %d", a.Comp.Loyalty)
	}
	if a.Attr == attr && a.MaxHP == hp && a.AVNat == av && a.DVNat == dv {
		t.Errorf("no perk granted")
	}
	if len(g.Deeds) != deeds+1 || g.Deeds[deeds].Kind != "companion" {
		t.Errorf("no deed recorded")
	}
	if mq := g.cqMirror(q); mq == nil || !mq.Closed {
		t.Errorf("journal quest not closed")
	}
	// one per companion
	g.Turn += 5 * DayLen
	g.Turn = (g.Turn/60 + 1) * 60
	g.cqTick()
	if g.cq().Q[a.ID] != q {
		t.Errorf("second personal quest offered")
	}
}

func TestPersonalQuestEveryKind(t *testing.T) {
	g := newGame(t, nil)
	a := recruitOne(t, g)
	seen := map[string]bool{}
	for i := 0; i < 40; i++ {
		a.ID = 100000 + i
		if q := g.cqMake(a); q != nil {
			seen[q.Kind] = true
			if q.Title == "" || q.Text == "" || q.Target == "" || strings.Contains(q.Text, "%!") {
				t.Errorf("bad quest %+v", q)
			}
		}
	}
	if len(seen) < 3 {
		t.Errorf("few quest kinds: %v", seen)
	}
}

func cpSetup(t *testing.T, g *Game) *Actor {
	a := recruitOne(t, g)
	a.Order = OrdFollow
	return a
}

func cpFit(g *Game, a *Actor, id string) {
	d := augByID[id]
	for _, p := range a.Body {
		if augFits(d.Slot, p) && p.Aug == nil && !p.Severed {
			p.Aug = g.MakeAugment(id)
			return
		}
	}
}

func TestCompanionCastsHealAndFlames(t *testing.T) {
	g := newGame(t, nil)
	a := cpSetup(t, g)
	cpFit(g, a, "medic")
	a.HP = maxi(1, a.MaxHP/5)
	hp := a.HP
	if !g.companionAct(a) || a.HP <= hp || a.Cool["heal"] <= 0 {
		t.Fatalf("companion did not heal: hp %d→%d cool %v", hp, a.HP, a.Cool)
	}
	// on cooldown: no second heal
	a.HP = hp
	if g.cpCast(a) {
		t.Errorf("heal ignored its cooldown")
	}
	cd := a.Cool["heal"]
	for i := 0; i < 3; i++ {
		g.cpTick()
	}
	if a.Cool["heal"] != cd-3 {
		t.Errorf("cooldown not ticking: %d→%d", cd, a.Cool["heal"])
	}
	// flames at a foe away from allies
	a.HP = a.MaxHP
	cpFit(g, a, "flamer")
	foe := g.Spawn("rat", 1, g.R)
	foe.MaxHP, foe.HP = 500, 500
	foe.X, foe.Y = a.X-4, a.Y
	var keep []*Actor
	for _, o := range g.Cur.Actors {
		if o == a || abs(o.Y-a.Y) > 2 || o.X > a.X || o.X < foe.X-2 {
			keep = append(keep, o)
		}
	}
	g.Cur.Actors = append(keep, foe)
	for x := foe.X - 1; x <= a.X; x++ {
		for y := a.Y - 1; y <= a.Y+1; y++ {
			g.Cur.Set(x, y, TFloor)
		}
	}
	if !g.compEnemy(a, foe) {
		t.Skip("rat not an enemy here")
	}
	if !g.cpCast(a) || a.Cool["flames"] <= 0 || foe.HP >= 500 && foe.St("burning") == 0 {
		t.Fatalf("companion did not cast flames: foe hp %d cool %v", foe.HP, a.Cool)
	}
	if g.P.HP < g.P.MaxHP {
		t.Errorf("flames hurt the player")
	}
}

func TestCompanionFleesAndShields(t *testing.T) {
	g := newGame(t, nil)
	a := cpSetup(t, g)
	cpFit(g, a, "bulwark")
	cpFit(g, a, "sprinter")
	for i := 0; i < 2; i++ {
		f := g.Spawn("rat", 1, g.R)
		f.X, f.Y = a.X-1, a.Y-1+2*i
		g.Cur.Set(f.X, f.Y, TFloor)
		g.Cur.Actors = append(g.Cur.Actors, f)
	}
	if !g.cpCast(a) || a.St("shielded") == 0 {
		t.Fatalf("no force bubble when surrounded")
	}
	a.Order = OrdPassive
	if !g.cpCast(a) || a.St("sprint") == 0 {
		t.Fatalf("passive companion did not sprint away")
	}
}

func TestCompanions2SaveLoad(t *testing.T) {
	g := newGame(t, nil)
	a := recruitOne(t, g)
	a.Comp.Loyalty = 80
	a.Comp.Since = g.Turn - 4*DayLen
	g.Turn = (g.Turn/60 + 1) * 60
	g.cqTick()
	q := g.cq().Q[a.ID]
	if q == nil {
		t.Fatal("no quest")
	}
	g.cqAccept(a, q)
	g.cb().Last[a.ID] = 1234
	g.cb().Seen["x"] = true
	a.Cool = map[string]int{"heal": 17}
	h, err := LoadGame(g.W, g.Save())
	if err != nil {
		t.Fatal(err)
	}
	hq := h.cq().Q[a.ID]
	if hq == nil || hq.Stage != 1 || hq.Title != q.Title || hq.Target != q.Target || hq.QID != q.QID {
		t.Fatalf("quest lost: %+v", hq)
	}
	if h.cqMirror(hq) == nil {
		t.Errorf("mirror quest lost")
	}
	if h.cb().Last[a.ID] != 1234 || !h.cb().Seen["x"] {
		t.Errorf("banter state lost")
	}
	if b := h.findActor(a.ID); b == nil || b.Cool["heal"] != 17 {
		t.Errorf("companion cooldown lost")
	}
}
