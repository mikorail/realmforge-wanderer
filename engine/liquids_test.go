package engine

import "testing"

func lqSkin(g *Game) *Item {
	for _, it := range g.P.Inv {
		if it.Kind == "water" {
			return it
		}
	}
	it := g.MakeWater()
	g.P.Inv = append(g.P.Inv, it)
	return it
}

func TestLqMixAndPay(t *testing.T) {
	g := newGame(t, nil)
	a := lqSkin(g)
	g.lqSet(a, "fresh", 10)
	b := g.MakeWater()
	g.P.Inv = append(g.P.Inv, b)
	g.lqSet(b, "wine", 4)
	g.lqPourInto(b, a)
	if lqKey(a) != "wine" || a.Charges != 14 {
		t.Errorf("fresh+wine = %s %d", lqKey(a), a.Charges)
	}
	g.lqSet(b, "salt", 3)
	g.lqSet(a, "fresh", 5)
	g.lqPourInto(b, a)
	if lqKey(a) != "brackish" {
		t.Errorf("fresh+salt = %s", lqKey(a))
	}
	g.lqSet(a, "fresh", 8)
	g.lqSet(b, "blood", 2)
	g.lqPourInto(b, a)
	if lqKey(a) != "tainted" {
		t.Errorf("fresh+blood = %s", lqKey(a))
	}
	g.lqSet(a, "fresh", 8)
	g.lqSet(b, "fresh", 5)
	if g.lqDrams() != 13 {
		t.Errorf("drams = %d", g.lqDrams())
	}
	if g.lqPay(20) || g.lqDrams() != 13 {
		t.Error("overpay should fail and take nothing")
	}
	if !g.lqPay(10) || g.lqDrams() != 3 {
		t.Errorf("pay 10: %d left", g.lqDrams())
	}
}

func TestLqPriceFactor(t *testing.T) {
	g := newGame(t, nil)
	g.P.Attr[EGO] = 8
	lo := g.lqPriceFactor(true)
	g.P.Attr[EGO] = 24
	hi := g.lqPriceFactor(true)
	if hi >= lo {
		t.Errorf("higher ego should buy cheaper: %f vs %f", hi, lo)
	}
	if g.lqPriceFactor(false) <= 0 {
		t.Error("sell factor")
	}
}

func TestLqDrinkEffects(t *testing.T) {
	g := newGame(t, nil)
	g.Thirst = 800
	g.lqDrinkEffect("salt", 1)
	if g.Thirst <= 800 {
		t.Error("salt water should worsen thirst")
	}
	g.Thirst = 800
	g.lqDrinkEffect("fresh", 1)
	if g.Thirst != 350 {
		t.Errorf("fresh dram: %d", g.Thirst)
	}
	if lqThirstState(1000) != "Thirsty" || lqHungerState(2500) != "Famished" {
		t.Error("states")
	}
}

func TestLqAutoDrinkAndScreen(t *testing.T) {
	g := newGame(t, nil)
	it := lqSkin(g)
	g.lqSet(it, "fresh", 6)
	g.Thirst = 1000
	g.Turn = 300
	lqSurvivalTick(g)
	if it.Charges != 5 || g.Thirst >= 1000 {
		t.Errorf("auto-drink: %d dr, thirst %d", it.Charges, g.Thirst)
	}
	g.Key("P")
	if g.Mode != "liquid" {
		t.Fatalf("mode %s", g.Mode)
	}
	g.Key("p")
	g.Render()
	g.Key("Escape")
	if g.Mode != "main" {
		t.Error("escape")
	}
}

func TestLqPoolsAndSave(t *testing.T) {
	g := newGame(t, map[string]int{})
	g.Skills["survival"] = 1
	p := g.P
	idx := p.Y*MW + p.X
	g.Cur.Set(p.X, p.Y, TFloor)
	g.Cur.Liq[idx] = lqSalt
	if k, ok := g.lqPoolAt(p.X, p.Y); !ok || k != "salt" {
		t.Errorf("pool %s %v", k, ok)
	}
	if !g.lqDrinkHere() {
		t.Error("survival lore should stop a salt drink")
	}
	if f, _, ok := lqTint(lqLava); !ok || f == 0 {
		t.Error("tint")
	}
	it := lqSkin(g)
	g.lqSet(it, "honey", 3)
	g.lq().Surplus = 7
	data := g.Save()
	g2, err := LoadGame(g.W, data)
	if err != nil {
		t.Fatal(err)
	}
	if g2.lq().Surplus != 7 {
		t.Error("lq state lost in save")
	}
}
