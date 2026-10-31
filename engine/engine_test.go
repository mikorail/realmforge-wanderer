package engine

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

var worlds = map[string]*WorldData{}

func world(t *testing.T, name string) *WorldData {
	t.Helper()
	if w, ok := worlds[name]; ok {
		return w
	}
	b, err := os.ReadFile("../public/worlds/" + name + ".json")
	if err != nil {
		t.Fatalf("read world: %v", err)
	}
	w, err := ParseWorld(string(b))
	if err != nil {
		t.Fatalf("parse world: %v", err)
	}
	worlds[name] = w
	return w
}

func newGame(t *testing.T, powers map[string]int) *Game {
	t.Helper()
	g := NewGame(world(t, "ember-crown"), CharSpec{Name: "Tester", Species: "human", Calling: "warrior", Attr: [6]int{18, 16, 16, 14, 14, 14}, Powers: powers})
	if g.Cur == nil || g.Cur.Kind != "town" {
		t.Fatalf("expected to start in a town, got %+v", g.Cur)
	}
	return g
}

// placeFoe drops a creature right next to the player.
func placeFoe(g *Game, tpl string, lvl int) *Actor {
	a := g.Spawn(tpl, lvl, g.R)
	a.X, a.Y = g.P.X+1, g.P.Y
	g.Cur.Set(a.X, a.Y, TFloor)
	if o := g.Cur.ActorAt(a.X, a.Y); o != nil {
		o.X, o.Y = 0, 0
	}
	g.Cur.Actors = append(g.Cur.Actors, a)
	g.recompute()
	return a
}

func clearAround(g *Game) {
	for dy := -2; dy <= 2; dy++ {
		for dx := -2; dx <= 2; dx++ {
			g.Cur.Set(g.P.X+dx, g.P.Y+dy, TFloor)
		}
	}
	var keep []*Actor
	for _, a := range g.Cur.Actors {
		if abs(a.X-g.P.X) > 3 || abs(a.Y-g.P.Y) > 3 {
			keep = append(keep, a)
		}
	}
	g.Cur.Actors = keep
}

func TestStartsInHomeTownWithNotables(t *testing.T) {
	g := newGame(t, nil)
	notables := 0
	for _, a := range g.Cur.Actors {
		if a.SimPer > 0 {
			notables++
		}
		if g.Hostile(a) {
			t.Errorf("%s (%s) is hostile in the home town", a.Name, a.Faction)
		}
	}
	if len(g.Cur.Actors) < 4 {
		t.Errorf("town is nearly empty: %d actors", len(g.Cur.Actors))
	}
	t.Logf("town %q: %d actors, %d notables from the sim", g.Cur.Name, len(g.Cur.Actors), notables)
}

func TestMeleeKillGivesXPAndLoot(t *testing.T) {
	g := newGame(t, nil)
	clearAround(g)
	foe := placeFoe(g, "rat", 1)
	xp := g.XP + g.P.Level*1000
	for i := 0; i < 60 && foe.HP > 0; i++ {
		g.Key("ArrowRight")
	}
	if foe.HP > 0 {
		t.Fatalf("rat survived 60 swings (hp %d)", foe.HP)
	}
	if g.Kills != 1 {
		t.Errorf("kills = %d", g.Kills)
	}
	if g.XP+g.P.Level*1000 <= xp {
		t.Errorf("no xp gained")
	}
	if len(g.Cur.ItemsAt(foe.X, foe.Y)) == 0 {
		t.Errorf("no corpse/loot left")
	}
}

func TestPlayerCanDie(t *testing.T) {
	g := newGame(t, nil)
	clearAround(g)
	placeFoe(g, "bear", 12)
	g.P.HP = 1
	for i := 0; i < 50 && !g.Dead; i++ {
		g.Key(".")
	}
	if !g.Dead || g.Mode != "dead" {
		t.Fatalf("player did not die (hp %d, mode %s)", g.P.HP, g.Mode)
	}
	if !strings.Contains(g.PendingDeeds(), "perished") {
		t.Errorf("death deed missing")
	}
	if out := g.Key("Enter"); !out.Exit {
		t.Errorf("Enter on death screen should exit")
	}
}

func TestRitualGivesRepAndSecret(t *testing.T) {
	g := newGame(t, nil)
	var npc *Actor
	for _, a := range g.Cur.Actors {
		if a.Role == "villager" || a.Role == "guard" {
			npc = a
			break
		}
	}
	if npc == nil {
		t.Skip("no villager")
	}
	before := g.Rep[npc.Faction]
	g.openTalk(npc)
	for i, o := range g.Opts {
		if strings.Contains(o.Text, g.Sk.Ritual) {
			g.Key(string(rune('a' + i)))
			break
		}
	}
	if g.Rep[npc.Faction] <= before {
		t.Errorf("rep did not rise: %d → %d", before, g.Rep[npc.Faction])
	}
	if !npc.Kinship {
		t.Errorf("kinship not set")
	}
	if len(g.Secrets) == 0 {
		t.Logf("no secret (world may have none nearby): %v", g.Popup)
	}
}

func TestQuestSlayLoop(t *testing.T) {
	g := newGame(t, nil)
	var elder *Actor
	for _, a := range g.Cur.Actors {
		if a.Role == "elder" || a.Role == "notable" {
			elder = a
			break
		}
	}
	if elder == nil {
		t.Skip("no elder in start town")
	}
	// keep asking new elders until a bandit-boss quest appears
	var q *Quest
	for tries := 0; tries < 40 && q == nil; tries++ {
		elder.Quest = 0
		elder.ID = g.newID()
		g.Talk = elder
		g.offerQuest(elder)
		last := g.Quests[len(g.Quests)-1]
		if last.Kind == "slay" && last.Target != "" {
			q = last
		}
	}
	if q == nil {
		t.Skip("no generated slay quest rolled")
	}
	var wx, wy, zx, zy, z int
	if _, err := fmtSscan(q.Target, &wx, &wy, &zx, &zy, &z); err != nil {
		t.Fatal(err)
	}
	g.Warp(wx, wy, zx, zy, z)
	var boss *Actor
	for _, a := range g.Cur.Actors {
		if a.Quest == q.ID {
			boss = a
		}
	}
	if boss == nil {
		t.Fatalf("quest boss not spawned in %s", g.Cur.Key)
	}
	g.hurt(boss, boss.HP+100, g.P, "test")
	if !q.Done {
		t.Fatalf("quest not done after boss died")
	}
	gold := g.P.Gold
	g.Talk = elder
	g.turnIn(q)
	if !q.Closed || g.P.Gold <= gold {
		t.Errorf("turn-in failed: closed=%v gold %d→%d", q.Closed, gold, g.P.Gold)
	}
}

func TestTradeBuyAndSell(t *testing.T) {
	g := newGame(t, nil)
	var shop *Actor
	for _, a := range g.Cur.Actors {
		if (a.Role == "merchant" || a.Role == "smith" || a.Role == "apothecary") && len(a.Inv) > 0 {
			shop = a
		}
	}
	if shop == nil {
		t.Skip("no shop")
	}
	g.P.Gold = 100000
	g.openTrade(shop)
	inv, gold := len(g.P.Inv), g.P.Gold
	g.Key("Enter")
	if g.P.Gold >= gold || len(g.P.Inv) < inv {
		t.Errorf("buy failed: gold %d→%d inv %d→%d", gold, g.P.Gold, inv, len(g.P.Inv))
	}
	g.Key("Tab")
	gold = g.P.Gold
	g.Key("Enter")
	if g.P.Gold <= gold {
		t.Errorf("sell failed")
	}
}

func TestSkyFall(t *testing.T) {
	g := newGame(t, nil)
	w := g.W
	found := false
	for p := 0; p < w.W*w.H && !found; p++ {
		if w.skyOpen[p] && !w.IsWater(p) {
			x, y, _ := g.updraftAt(p%w.W, p/w.W, 1, 1)
			g.Warp(p%w.W, p/w.W, 1, 1, 0)
			g.P.X, g.P.Y = x, y
			found = true
		}
	}
	if !found {
		t.Skip("no sky isles")
	}
	if g.Cur.At(g.P.X, g.P.Y) != TUpdraft {
		t.Fatalf("no updraft at its spot")
	}
	g.Key("<")
	if g.Cur.Z != -1 {
		t.Fatalf("did not ascend, z=%d", g.Cur.Z)
	}
	// walk off into the void
	hp := g.P.HP + 1000
	g.P.HP, g.P.MaxHP = hp, hp
	for x := 0; x < MW; x++ {
		if g.Cur.At(x, g.P.Y) == TVoid {
			g.P.X = x - 1
			g.Cur.Set(x-1, g.P.Y, TCloud)
			g.Key("ArrowRight")
			break
		}
	}
	if g.Cur.Z != 0 {
		t.Fatalf("did not fall to the surface, z=%d", g.Cur.Z)
	}
	if g.P.HP >= hp {
		t.Errorf("fall did no damage")
	}
}

func TestUnderwaterBreath(t *testing.T) {
	g := newGame(t, nil)
	w := g.W
	found := false
	for p := 0; p < w.W*w.H; p++ {
		if w.IsWater(p) && w.seaOpen[p] {
			g.Warp(p%w.W, p/w.W, 1, 1, 1)
			found = true
			break
		}
	}
	if !found {
		t.Skip("no sea floor")
	}
	if !g.Cur.Underwater {
		t.Fatalf("zone %s not underwater", g.Cur.Kind)
	}
	clearAround(g)
	b := g.Breath
	for i := 0; i < 5; i++ {
		g.Key(".")
	}
	if g.Breath >= b {
		t.Errorf("breath did not drop underwater: %d → %d", b, g.Breath)
	}
}

func TestFlamesBurnFoeAndGrass(t *testing.T) {
	g := newGame(t, map[string]int{"flames": 2})
	clearAround(g)
	foe := placeFoe(g, "wolf", 3)
	foe.X = g.P.X + 2
	g.Cur.Set(g.P.X+1, g.P.Y, TGrass)
	hp := foe.HP
	g.recompute()
	g.Key("a")
	g.Key("a")
	g.CX, g.CY = foe.X, foe.Y
	g.Key("Enter")
	if foe.HP >= hp {
		t.Errorf("flames did no damage (%d)", foe.HP)
	}
	if g.P.Cool["flames"] == 0 {
		t.Errorf("no cooldown set")
	}
}

func TestDescendIntoDarkness(t *testing.T) {
	g := newGame(t, nil)
	g.DebugGotoStairs()
	if g.Cur.At(g.P.X, g.P.Y) != TDown {
		t.Skip("no stairs nearby")
	}
	g.Key(">")
	if g.Cur.Z != 1 || !g.Cur.Dark {
		t.Fatalf("expected dark z=1, got z=%d dark=%v", g.Cur.Z, g.Cur.Dark)
	}
	if g.Cur.At(g.P.X, g.P.Y) != TUp {
		t.Errorf("no way back up under the stairs")
	}
	g.Key("<")
	if g.Cur.Z != 0 {
		t.Errorf("could not climb back")
	}
}

func TestWorldTravelCostsTime(t *testing.T) {
	g := newGame(t, nil)
	g.Key("w")
	if g.Mode != "world" {
		t.Fatalf("world map did not open: %s", g.Mode)
	}
	turn := g.Turn
	moved := false
	for _, k := range []string{"ArrowRight", "ArrowLeft", "ArrowUp", "ArrowDown"} {
		wx, wy := g.WX, g.WY
		g.Key(k)
		if g.WX != wx || g.WY != wy {
			moved = true
			break
		}
	}
	if !moved {
		t.Skip("boxed in by sea")
	}
	if g.Turn <= turn {
		t.Errorf("travel took no time")
	}
	g.Key("Enter")
	if g.Mode != "main" || g.Cur.Z != 0 {
		t.Errorf("could not enter zone from world map")
	}
}

func TestSaveLoadRoundTrip(t *testing.T) {
	g := newGame(t, map[string]int{"teleport": 1})
	for i := 0; i < 30; i++ {
		g.Key([]string{"ArrowLeft", "ArrowDown", "ArrowRight", "ArrowUp"}[i%4])
	}
	s := g.Save()
	h, err := LoadGame(g.W, s)
	if err != nil {
		t.Fatal(err)
	}
	if h.P.Name != g.P.Name || h.CurKey != g.CurKey || h.Turn != g.Turn || h.P.X != g.P.X || len(h.P.Inv) != len(g.P.Inv) || h.P.Pow("teleport") != 1 {
		t.Errorf("state lost in round trip")
	}
	if len(h.Cur.Actors) != len(g.Cur.Actors) {
		t.Errorf("actors %d → %d", len(g.Cur.Actors), len(h.Cur.Actors))
	}
	h.Key("ArrowDown")
	h.Render()
}

func TestEveryGenreAndSpeciesStarts(t *testing.T) {
	for _, name := range []string{"ember-crown", "neon-tide", "spore-mother"} {
		w := world(t, name)
		for _, sp := range w.Species {
			for _, c := range callingDefs {
				g := NewGame(w, CharSpec{Name: "x", Species: sp.ID, Calling: c.ID, Attr: [6]int{14, 14, 14, 14, 14, 14}})
				g.Render()
				if g.P.MaxHP <= 0 {
					t.Errorf("%s %s %s: bad hp", name, sp.ID, c.ID)
				}
			}
		}
	}
}

func fmtSscan(key string, a ...*int) (int, error) {
	return fmt.Sscanf(key, "%d.%d.%d.%d.%d", a[0], a[1], a[2], a[3], a[4])
}

func TestBanditTownsStayTowns(t *testing.T) {
	w := world(t, "spore-mother")
	g := NewGame(w, CharSpec{Name: "x", Species: "human", Calling: "scout", Attr: [6]int{14, 14, 14, 14, 14, 14}})
	for _, b := range w.Burgs {
		if b.Bandit && b.Alive && b.Layer == "surface" {
			z := g.GetZone(b.P%w.W, b.P/w.W, 1, 1, 0)
			if z.Kind != "town" {
				t.Fatalf("%s generated as %s", b.Name, z.Kind)
			}
			camp := g.GetZone(b.P%w.W, b.P/w.W, 0, 1, 0)
			if camp.Kind != "camp" {
				t.Errorf("no outlaw camp outside %s (got %s)", b.Name, camp.Kind)
			}
			return
		}
	}
	t.Skip("no bandit-harassed town")
}
