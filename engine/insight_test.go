package engine

import "testing"

func TestInInsightTiersAndHearts(t *testing.T) {
	g := newGame(t, nil)
	inGain(g, 4, "")
	if inTier(inSt(g).Ins) != 0 {
		t.Fatal("tier too early")
	}
	inGain(g, 1, "")
	if inTier(inSt(g).Ins) != 1 {
		t.Fatal("tier 1 expected")
	}
	hp := g.P.MaxHP
	for i := 0; i < 4; i++ {
		inHeartPiece(g, "")
	}
	if inHearts(g) != 1 || g.P.MaxHP != hp+inHeartHP {
		t.Errorf("hearts %d maxhp %d→%d", inHearts(g), hp, g.P.MaxHP)
	}
}

func TestInSeesHidden(t *testing.T) {
	g := newGame(t, nil)
	a := &Actor{}
	a.Set("in_veil")
	inSt(g)
	if inSeesHidden(a) {
		t.Error("veiled should be hidden at 0")
	}
	inGain(g, 15, "")
	if !inSeesHidden(a) {
		t.Error("visible at 15")
	}
}

func TestInGateBump(t *testing.T) {
	g := newGame(t, nil)
	clearAround(g)
	x, y := g.P.X+1, g.P.Y
	g.Cur.Set(x, y, TStatue)
	st := inSt(g)
	sp := &inSpot{Zone: g.Cur.Key, Idx: y*MW + x, K: "gate", Req: "str", N: g.P.Score(STR) + 5, Reward: "gold", Direct: true}
	st.Spots[inKeyOf(g.Cur, sp.Idx)] = sp
	if !inBump(g, x, y) || sp.Done {
		t.Fatal("should refuse")
	}
	g.P.Attr[STR] += 10
	gold := g.P.Gold
	if !inBump(g, x, y) || !sp.Done || g.Cur.At(x, y) == TStatue || g.P.Gold <= gold {
		t.Fatal("should open")
	}
	// hearts seal
	sp2 := &inSpot{Zone: g.Cur.Key, Idx: y*MW + x - 1, K: "gate", Req: "hearts", N: 1, Reward: "gold", Direct: true}
	g.Cur.Set(x-1, y, TStatue)
	st.Spots[inKeyOf(g.Cur, sp2.Idx)] = sp2
	inBump(g, x-1, y)
	if sp2.Done {
		t.Fatal("heart seal opened without hearts")
	}
	for i := 0; i < 4; i++ {
		inHeartPiece(g, "")
	}
	inBump(g, x-1, y)
	if !sp2.Done {
		t.Fatal("heart seal should open")
	}
}

func TestInGlyphAndSecretReveal(t *testing.T) {
	g := newGame(t, nil)
	clearAround(g)
	st := inSt(g)
	gx, gy := g.P.X+4, g.P.Y
	g.Cur.Set(gx, gy, TWall)
	st.Spots[inKeyOf(g.Cur, gy*MW+gx)] = &inSpot{Zone: g.Cur.Key, Idx: gy*MW + gx, K: "glyph", Text: "X"}
	sx, sy := g.P.X, g.P.Y+4
	g.Cur.Set(sx, sy, TWall)
	st.Spots[inKeyOf(g.Cur, sy*MW+sx)] = &inSpot{Zone: g.Cur.Key, Idx: sy*MW + sx, K: "secret"}
	inGain(g, 5, "")
	if g.Cur.At(gx, gy) != TInscription || g.Cur.At(sx, sy) != TWall {
		t.Fatal("glyph only at tier 1")
	}
	inGain(g, 25, "")
	if g.Cur.At(sx, sy) != TDoor {
		t.Fatal("secret door at 30")
	}
}

func TestInShrineSpend(t *testing.T) {
	g := newGame(t, nil)
	inGain(g, 10, "")
	inShrineMenu(g, &Actor{Name: "Shrine"}, "x", nil)
	n := len(g.P.Inv)
	g.Opts[0].Do()
	if inSt(g).Ins != 5 || len(g.P.Inv) != n+1 {
		t.Errorf("ins %d inv %d→%d", inSt(g).Ins, n, len(g.P.Inv))
	}
}

func TestInGenerationAndSave(t *testing.T) {
	g := newGame(t, nil)
	found := 0
	for p := 0; p < g.W.W*g.W.H && found < 8; p++ {
		for zy := 0; zy < 3; zy++ {
			for zx := 0; zx < 3; zx++ {
				s := g.siteAt(p%g.W.W, p/g.W.W, zx, zy)
				if s == nil || s.Kind != "dungeon" {
					continue
				}
				found++
				for z := 1; z <= mini(s.Depth, 4); z++ {
					g.GetZone(p%g.W.W, p/g.W.W, zx, zy, z)
				}
			}
		}
	}
	if len(inSt(g).Spots) == 0 {
		t.Skip("no spots generated")
	}
	n := len(inSt(g).Spots)
	g2, err := LoadGame(g.W, g.Save())
	if err != nil || len(inSt(g2).Spots) != n {
		t.Fatalf("save/load lost spots: %v", err)
	}
}
