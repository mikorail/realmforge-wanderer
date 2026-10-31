package engine

import (
	"strings"
	"testing"
)

// recruitOne recruits a villager or guard of the start town and puts them next to the player.
func recruitOne(t *testing.T, g *Game) *Actor {
	t.Helper()
	if g.companionCount() == 0 {
		clearAround(g)
	}
	var a *Actor
	for _, o := range g.Cur.Actors {
		if (o.Role == "villager" || o.Role == "guard") && o.AI != "follower" && o.HP > 0 {
			a = o
			break
		}
	}
	if a == nil {
		t.Fatal("no villager or guard to recruit")
	}
	g.Rep[a.Faction] = 1000
	g.Talk = a
	g.recruit(a)
	if a.AI != "follower" || a.Comp == nil {
		t.Fatalf("recruit failed: ai=%s", a.AI)
	}
	a.X, a.Y = g.P.X-1, g.P.Y-g.companionCount()+1
	g.Cur.Set(a.X, a.Y, TFloor)
	g.Mode, g.Talk = "main", nil
	return a
}

func hasActor(z *Zone, a *Actor) bool {
	for _, o := range z.Actors {
		if o == a {
			return true
		}
	}
	return false
}

func TestCompanionStayAndFollowAcrossZones(t *testing.T) {
	g := newGame(t, nil)
	a := recruitOne(t, g)
	b := recruitOne(t, g)
	home := g.Cur
	// order a to stay via the companion talk menu
	g.openTalk(a)
	if g.Mode != "talk" || !strings.Contains(g.Opts[0].Text, "orders") {
		t.Fatalf("companion talk menu not shown: %v", g.Opts)
	}
	g.Opts[0].Do()
	for _, o := range g.Opts {
		if strings.HasPrefix(o.Text, "Stay here") {
			o.Do()
		}
	}
	if a.Order != OrdStay {
		t.Fatalf("order not set: %q", a.Order)
	}
	g.Warp(home.WX, home.WY, (home.ZX+1)%3, home.ZY, 0)
	if g.Cur == home {
		t.Fatal("did not change zone")
	}
	if !hasActor(home, a) || hasActor(g.Cur, a) {
		t.Errorf("staying companion moved with the player")
	}
	if !hasActor(g.Cur, b) {
		t.Errorf("following companion was left behind")
	}
	if w := g.waitingCompanions(); len(w) != 1 || w[0].A != a {
		t.Errorf("waiting list wrong: %v", w)
	}
	g.openCompanions()
	g.Render()
	if g.companionCount() != 2 {
		t.Errorf("count %d", g.companionCount())
	}
	// world map travel takes the follower along
	g.Warp(home.WX, home.WY, home.ZX, home.ZY, 0)
	if !hasActor(g.Cur, a) || !hasActor(g.Cur, b) {
		t.Errorf("returning should find both companions")
	}
	g.setOrder(a, OrdFollow)
	g.Key("w")
	if g.Mode == "world" {
		g.Key("Enter")
		if !hasActor(g.Cur, a) || !hasActor(g.Cur, b) {
			t.Errorf("companions lost on world-map travel")
		}
	}
}

func TestCompanionAttackOrder(t *testing.T) {
	g := newGame(t, nil)
	a := recruitOne(t, g)
	a.Attr[STR], a.Attr[AGI] = 30, 30
	foe := g.Spawn("rat", 1, g.R)
	foe.X, foe.Y = a.X-2, a.Y
	g.Cur.Set(foe.X, foe.Y, TFloor)
	g.Cur.Set(a.X-1, a.Y, TFloor)
	g.Cur.Actors = append(g.Cur.Actors, foe)
	g.recompute()
	g.Pend = "order:0"
	g.orderAttack("order:0", foe.X, foe.Y)
	if a.Order != OrdAttack || a.Comp.Target != foe.ID {
		t.Fatalf("attack order not given: %q %d", a.Order, a.Comp.Target)
	}
	for i := 0; i < 60 && foe.HP > 0; i++ {
		g.spend(1000)
	}
	if foe.HP > 0 {
		t.Errorf("companion did not kill the target (foe hp %d)", foe.HP)
	}
	if a.Comp.Kills == 0 {
		t.Errorf("kill not credited")
	}
	// refuses to attack their own realm's people
	var own *Actor
	for _, o := range g.Cur.Actors {
		if o.Faction == a.Comp.Faction && o.AI != "follower" && o.HP > 0 {
			own = o
			break
		}
	}
	if own != nil {
		g.orderAttack("order:0", own.X, own.Y)
		if a.Order == OrdAttack && a.Comp.Target == own.ID {
			t.Errorf("companion agreed to attack their own people")
		}
	}
}

func TestCompanionLevelsUp(t *testing.T) {
	g := newGame(t, nil)
	a := recruitOne(t, g)
	lv, hp := a.Level, a.MaxHP
	for i := 0; i < 6; i++ {
		foe := g.Spawn("rat", 3, g.R)
		foe.X, foe.Y = a.X, a.Y+1
		g.Cur.Actors = append(g.Cur.Actors, foe)
		g.kill(foe, a)
	}
	if a.Level <= lv || a.MaxHP <= hp {
		t.Errorf("no level up: lv %d→%d hp %d→%d (xp %d)", lv, a.Level, hp, a.MaxHP, a.Comp.XP)
	}
	found := false
	for _, m := range g.Msgs {
		if strings.Contains(m.T, "grows more seasoned") {
			found = true
		}
	}
	if !found {
		t.Errorf("no level-up message")
	}
}

func TestCompanionLoyaltyDeparture(t *testing.T) {
	g := newGame(t, nil)
	a := recruitOne(t, g)
	fac := a.Comp.Faction
	l0 := a.Comp.Loyalty
	g.AddRep(fac, -100, " for murder")
	if a.Comp.Loyalty >= l0 {
		t.Fatalf("loyalty did not drop: %d", a.Comp.Loyalty)
	}
	g.give(g.MakeFood(g.R, 1))
	var food *Item
	for _, it := range g.P.Inv {
		if it.Kind == "food" {
			food = it
		}
	}
	l1 := a.Comp.Loyalty
	g.compGive(a, food)
	if a.Comp.Loyalty <= l1 {
		t.Errorf("gift did not raise loyalty")
	}
	g.Rep[fac] = -1000
	g.Hunger = 3000
	for i := 0; i < 60*80 && a.AI == "follower"; i++ {
		g.Turn++
		g.companionsTick()
	}
	if a.AI == "follower" || a.Faction != fac || a.Comp != nil {
		t.Errorf("disloyal companion did not leave: ai=%s faction=%s", a.AI, a.Faction)
	}
}

func TestCompanionGearDeathAndSave(t *testing.T) {
	g := newGame(t, nil)
	a := recruitOne(t, g)
	w := g.MakeWeapon("longblade", 3, g.R)
	g.give(w)
	msg := g.compGive(a, w)
	if a.Equipped(w) == nil {
		t.Errorf("companion did not wield the better weapon: %s", msg)
	}
	g.setOrder(a, OrdGuard)
	a.Comp.Loyalty = 77
	s := g.Save()
	h, err := LoadGame(g.W, s)
	if err != nil {
		t.Fatal(err)
	}
	var b *Actor
	for _, o := range h.Cur.Actors {
		if o.ID == a.ID {
			b = o
		}
	}
	if b == nil || b.AI != "follower" || b.Order != OrdGuard || b.Comp == nil || b.Comp.Loyalty != 77 || b.Comp.Faction != a.Comp.Faction {
		t.Fatalf("companion state lost in round trip: %+v", b)
	}
	h.Key("o")
	h.Render()
	h.Key("Escape")
	// death reaches the chronicle
	n := len(h.Deeds)
	h.hurt(b, 9999, nil, "test")
	if len(h.Deeds) != n+1 || h.Deeds[n].Kind != "companion" {
		t.Errorf("companion death not recorded as a deed")
	}
}
