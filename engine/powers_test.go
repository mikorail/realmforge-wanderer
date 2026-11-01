package engine

import "testing"

var newActives = []string{"climbing", "clipping", "arcanebolt", "slumber", "glimmer", "forcepush", "clairvoyance", "timedilation",
	"forcebubble", "syphon", "stunforce", "disintegrate", "corrosive", "fugue"}

func TestPowerNewActivesRun(t *testing.T) {
	for _, id := range newActives {
		g := newGame(t, map[string]int{id: 2})
		clearAround(g)
		f := placeFoe(g, "rat", 1)
		g.P.HP = g.P.MaxHP / 2
		pd := powerByID[id]
		if pd == nil || !pd.Active {
			t.Fatalf("%s not an active power", id)
		}
		tx, ty := f.X, f.Y
		if pd.Target == "dir" {
			tx, ty = 1, 0
		}
		if pd.Target == "self" {
			tx, ty = 0, 0
		}
		g.usePower(id, nil, tx, ty)
		g.P.Cool[id] = 0
	}
}

func TestPowerClairvoyanceMarksSeen(t *testing.T) {
	g := newGame(t, map[string]int{"clairvoyance": 1})
	g.usePower("clairvoyance", nil, 0, 0)
	for i, s := range g.Cur.Seen {
		if s == 0 {
			t.Fatalf("tile %d not seen", i)
		}
	}
}

func TestPowerForcePushMovesTarget(t *testing.T) {
	g := newGame(t, map[string]int{"forcepush": 1})
	clearAround(g)
	for dx := 1; dx <= 6; dx++ {
		g.Cur.Set(g.P.X+dx, g.P.Y, TFloor)
	}
	f := placeFoe(g, "rat", 1)
	f.HP, f.MaxHP = 500, 500
	x := f.X
	g.usePower("forcepush", nil, f.X, f.Y)
	if f.X <= x {
		t.Errorf("target not pushed: %d -> %d", x, f.X)
	}
}

func TestPowerFugueAppearsThenExpires(t *testing.T) {
	g := newGame(t, map[string]int{"fugue": 1})
	clearAround(g)
	n := len(g.Cur.Actors)
	g.usePower("fugue", nil, 0, 0)
	if len(g.Cur.Actors) <= n {
		t.Fatal("no echo spawned")
	}
	var echo *Actor
	for _, a := range g.Cur.Actors {
		if a.Tpl == "fugue" {
			echo = a
		}
	}
	if echo == nil || echo.AI != "follower" || echo.Faction != "player" {
		t.Fatalf("bad echo %+v", echo)
	}
	for i := 0; i < 80 && echo.HP > 0; i++ {
		g.powerStatusTick(echo)
		if echo.Status != nil {
			echo.Status["fugue"]--
		}
	}
	for _, a := range g.Cur.Actors {
		if a == echo {
			t.Fatal("echo did not expire")
		}
	}
}

func TestPowerClimbingAllowsRock(t *testing.T) {
	g := newGame(t, map[string]int{"climbing": 1})
	if !g.canClimb(TRock) || !g.canClimb(TTree) || g.canClimb(TWall) {
		t.Error("canClimb wrong")
	}
	if g.climbCost(g.P.X, g.P.Y, 1000) != 1000 {
		t.Error("floor cost changed")
	}
	g2 := newGame(t, nil)
	if g2.canClimb(TRock) {
		t.Error("climb without power")
	}
}

func TestPowerPassivesAndBody(t *testing.T) {
	g := newGame(t, map[string]int{"swimming": 1, "diving": 1, "multiarms": 1})
	if g.swimCostMult() != 1 {
		t.Error("swim mult")
	}
	if g.breathCap() < 90 {
		t.Error("breath cap")
	}
	h := 0
	for _, b := range g.P.Body {
		if b.Slot == "hand" {
			h++
		}
	}
	applyBodyPowers(g.P)
	applyBodyPowers(g.P)
	h2 := 0
	for _, b := range g.P.Body {
		if b.Slot == "hand" {
			h2++
		}
	}
	if h != 4 || h2 != 4 { // NewGame applies multiarms; reapplying must not add more
		t.Errorf("hands %d -> %d", h, h2)
	}
	g.P.AddSt("shielded", 5)
	if shieldAV(g.P) <= 0 {
		t.Error("shield av")
	}
}

func TestPowerNamesInEveryGenre(t *testing.T) {
	for _, genre := range []string{"", "mythic", "steampunk", "dieselpunk", "atompunk", "cyberpunk", "biopunk", "solarpunk", "wasteland", "twilight"} {
		sk := SkinFor(genre)
		for _, pd := range PowerDefs {
			if sk.PowerName(pd.ID) == "" {
				t.Errorf("%s: no name for %s", genre, pd.ID)
			}
		}
	}
}
