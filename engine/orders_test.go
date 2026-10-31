package engine

import "testing"

func orderGame(t *testing.T, name string) *Game {
	t.Helper()
	w := world(t, name)
	return NewGame(w, CharSpec{Name: "Tester", Species: "human", Calling: "warrior", Attr: [6]int{18, 18, 16, 18, 14, 18}})
}

func TestOrdersGeneratedAllWorlds(t *testing.T) {
	for _, n := range []string{"ember-crown", "neon-tide", "spore-mother"} {
		g := orderGame(t, n)
		os := g.Orders()
		kinds := map[string]int{}
		for _, o := range os {
			kinds[o.Kind]++
			if o.Name == "" || o.Leader == "" || o.Ranks[4] == "" || o.Creed == "" || powerByID[o.Power] == nil {
				t.Errorf("%s: incomplete order %+v", n, o)
			}
		}
		t.Logf("%s: %d orders %v", n, len(os), kinds)
		if kinds["secret"] != 1 || kinds["thieves"] == 0 || kinds["merchant"] == 0 || kinds["holy"]+kinds["monastic"] == 0 {
			t.Errorf("%s: missing kinds %v", n, kinds)
		}
	}
}

func orderOf(g *Game, kind string) *Order {
	for _, o := range g.Orders() {
		if o.Kind == kind {
			return o
		}
	}
	return nil
}

func TestOrderLeaderSpawnsAndJoinFlow(t *testing.T) {
	g := orderGame(t, "ember-crown")
	o := orderOf(g, "thieves")
	if o == nil {
		t.Fatal("no thieves guild")
	}
	zn := g.GetZone(o.WX, o.WY, o.ZX, o.ZY, 0)
	var leader *Actor
	for _, a := range zn.Actors {
		if a.Has("ord:" + o.ID) {
			leader = a
		}
	}
	if leader == nil {
		t.Fatalf("leader of %s not in zone %s", o.Name, zn.Name)
	}
	g.P.Gold = 5000
	g.P.Attr[AGI] = 20
	if ok, why := g.orderMeets(o); !ok {
		t.Fatal(why)
	}
	g.orderJoin(o)
	m := g.OrderMembership(o.ID)
	if m == nil || m.Rank != 1 {
		t.Fatalf("join failed: %+v", m)
	}
	// run tasks until rank 5
	for i := 0; i < 40 && m.Rank < 5; i++ {
		m.Task = g.newOrderTask(o, m, false)
		m.Task.Have, m.Task.Ready = m.Task.Need, true
		g.orderReport(o, m)
	}
	if m.Rank != 5 || !m.GotItem || g.P.Pow(o.Power) == 0 {
		t.Fatalf("rank %d item %v power %d", m.Rank, m.GotItem, g.P.Pow(o.Power))
	}
	// save/load
	g2, err := LoadGame(g.W, g.Save())
	if err != nil {
		t.Fatal(err)
	}
	m2 := g2.OrderMembership(o.ID)
	if m2 == nil || m2.Rank != 5 || m2.Done != m.Done {
		t.Fatalf("membership lost: %+v", m2)
	}
	// talk menu offers the order option
	g2.Talk = leader
	found := false
	for _, op := range orderTalkHook(g2, leader) {
		found = found || op.Text != ""
	}
	if !found {
		t.Error("no talk option")
	}
}

func TestOrderInitiationAndKill(t *testing.T) {
	g := orderGame(t, "ember-crown")
	o := orderOf(g, "secret")
	g.orderJoin(o)
	m := g.OrderMembership(o.ID)
	if m.Rank != 0 || m.Task == nil || !m.Task.Init {
		t.Fatalf("expected initiation: %+v", m)
	}
	m.Task.Kind, m.Task.Target, m.Task.Need, m.Task.Have = "kill", "outlaws", 1, 0
	orderKillHook(g, &Actor{Faction: "outlaws"}, g.P)
	if !m.Task.Ready {
		t.Fatal("kill did not complete task")
	}
	g.orderReport(o, m)
	if m.Rank != 1 || g.Rep[o.RepKey()] < 50 {
		t.Fatalf("not admitted: %+v", m)
	}
	g.Mode = "orders"
	g.Render()
}
