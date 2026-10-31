package engine

import "testing"

func townZones(t *testing.T, g *Game) (*BurgData, []*Zone) {
	t.Helper()
	w := g.W
	for i := range w.Burgs {
		b := &w.Burgs[i]
		if !b.Alive || b.Layer != "surface" || w.cityTier[b.ID] != 2 {
			continue
		}
		var zs []*Zone
		for zy := 0; zy < 3; zy++ {
			for zx := 0; zx < 3; zx++ {
				zs = append(zs, g.GetZone(b.P%w.W, b.P/w.W, zx, zy, 0))
			}
		}
		return b, zs
	}
	t.Skip("no tier-2 town")
	return nil, nil
}

func TestSocietyAssign(t *testing.T) {
	g := newGame(t, nil)
	_ = g.cityPlanFor
	bd, zs := townZones(t, g)
	strata := map[string]bool{}
	for _, s := range g.strataFor(bd) {
		strata[s] = true
	}
	npcs, houses := 0, 0
	for _, zn := range zs {
		g.assignSociety(zn, NewRng("soc:"+zn.Key), bd)
		g.Cur = zn
		byID := map[int]*Actor{}
		for _, a := range zn.Actors {
			byID[a.ID] = a
		}
		for _, a := range zn.Actors {
			if a.Tpl != "npc" {
				continue
			}
			npcs++
			if !strata[a.Caste] {
				t.Errorf("%s (%s) caste %q not in %v", a.Name, a.Role, a.Caste, g.strataFor(bd))
			}
			if a.Rank < 0 || a.Rank > 3 {
				t.Errorf("rank %d", a.Rank)
			}
			if g.socialLine(a) == "" {
				t.Errorf("%s has no social line", a.Name)
			}
			if g.casteGreeting(a) == "" {
				t.Errorf("%s has no greeting", a.Name)
			}
			if m := g.castePriceMod(a); m < 0.8 || m > 1.5 {
				t.Errorf("price mod %f", m)
			}
			for _, k := range a.Kin {
				o := byID[k]
				if o == nil {
					t.Errorf("%s kin %d missing", a.Name, k)
					continue
				}
				if o.House != a.House {
					t.Errorf("%s and %s differ in house", a.Name, o.Name)
				}
				back := false
				for _, x := range o.Kin {
					back = back || x == a.ID
				}
				if !back {
					t.Errorf("kin link %s -> %s not symmetric", a.Name, o.Name)
				}
			}
			if len(a.Kin) > 0 {
				houses++
			}
		}
	}
	if npcs == 0 || houses == 0 {
		t.Fatalf("npcs %d, in households %d", npcs, houses)
	}
	t.Logf("%d npcs, %d in households, standing %d", npcs, houses, g.socialStanding())
	for _, a := range zs[4].Actors {
		if len(a.Kin) > 0 {
			t.Log(a.Name, "|", a.Title, "|", g.socialLine(a), "|", g.casteGreeting(a))
			break
		}
	}
}

func TestSocietyStrataDiffer(t *testing.T) {
	seen := map[string]bool{}
	for _, name := range []string{"ember-crown", "neon-tide", "spore-mother"} {
		g := NewGame(world(t, name), CharSpec{Name: "T", Species: world(t, name).Species[0].ID, Calling: "warrior", Attr: [6]int{16, 16, 16, 14, 14, 14}})
		for i := range g.W.Burgs {
			b := &g.W.Burgs[i]
			seen[name+":"+g.strataFor(b)[0]+"/"+g.strataFor(b)[len(g.strataFor(b))-1]] = true
		}
	}
	if len(seen) < 3 {
		t.Fatalf("strata too uniform: %v", seen)
	}
	t.Log(seen)
}
