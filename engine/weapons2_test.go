package engine

import (
	"encoding/json"
	"strings"
	"testing"
)

func wpGame(t *testing.T) *Game {
	t.Helper()
	g := newGame(t, nil)
	clearAround(g)
	for _, p := range g.P.Body {
		if p.Slot == "hand" {
			p.Item = nil
		}
	}
	g.P.Attr[AGI] = 30
	g.P.Attr[STR] = 20
	return g
}

// wpFoeAt drops a creature at an offset from the player with a clear line between.
func wpFoeAt(g *Game, dx, dy int) *Actor {
	for i := 1; i <= maxi(abs(dx), abs(dy)); i++ {
		g.Cur.Set(g.P.X+i*skSgn(dx), g.P.Y+i*skSgn(dy), TFloor)
	}
	a := placeFoe(g, "bandit", 2)
	a.X, a.Y = g.P.X+dx, g.P.Y+dy
	g.Cur.Set(a.X, a.Y, TFloor)
	a.MaxHP, a.HP = 500, 500
	a.DVNat, a.AVNat = -20, 0
	g.recompute()
	return a
}

func wpWield(g *Game, it *Item) {
	g.give(it)
	if !g.equip(it) {
		panic("equip failed")
	}
}

func TestWpEveryGenreNamesEveryFamily(t *testing.T) {
	g := wpGame(t)
	for _, gen := range wpGenres {
		g.Genre, g.Sk = gen, SkinFor(gen)
		tbl, ok := wpNouns[gen]
		if !ok {
			t.Fatalf("no nouns for %s", gen)
		}
		if _, ok := wpGuns[gen]; !ok {
			t.Fatalf("no gun profile for %s", gen)
		}
		for _, f := range wpFamilies {
			n := tbl[f]
			if n[0] == "" || n[1] == "" {
				t.Fatalf("%s does not name %s", gen, f)
			}
			if f == "shield" {
				continue
			}
			for tier := 0; tier <= 5; tier += 5 {
				it := g.MakeFamily(f, tier, NewRng("wp"))
				if !strings.Contains(it.Name, wpNoun(gen, f, tier)) || it.DN == 0 || it.Value <= 0 || wpFamOf(it) != f {
					t.Fatalf("%s %s tier %d: bad item %+v", gen, f, tier, it)
				}
				if it.Kind == "ranged" && (it.Effect == "" || g.wpAmmoName(it.Effect) == "") {
					t.Fatalf("%s %s fires nothing", gen, f)
				}
			}
		}
		if len(wpGenreShields(gen)) < 5 || len(wpGenreArmors(gen)) < 10 {
			t.Fatalf("%s: too few shields/armours", gen)
		}
	}
}

func TestWpSphereTables(t *testing.T) {
	g := wpGame(t)
	for _, sp := range attireSpheresSorted() {
		k := wpSpheres[sp]
		if k == nil || len(k.Weapons) < 4 || len(k.Armor) == 0 {
			t.Fatalf("sphere %s has no kit", sp)
		}
		for i, e := range k.Weapons {
			legacy := e.Fam == "longblade" || e.Fam == "shortblade" || e.Fam == "axe" || e.Fam == "cudgel"
			if !legacy && wpFamIdx(e.Fam) == 0 || e.Desc == "" {
				t.Fatalf("%s %s: bad family %q", sp, e.Name, e.Fam)
			}
			it := g.MakeSphereWeapon(sp, i, 3, NewRng("s"))
			if it.Region != sp || !strings.Contains(it.Name, e.Name) || it.Lore != e.Desc {
				t.Fatalf("%s weapon %+v", sp, it)
			}
		}
		for _, e := range k.Armor {
			if wpArmorTypes[e.Fam] == nil {
				t.Fatalf("%s armour %s: unknown type %s", sp, e.Name, e.Fam)
			}
		}
		if k.Shield != nil && wpShieldTypes[k.Shield.Fam] == nil {
			t.Fatalf("%s shield type %s", sp, k.Shield.Fam)
		}
	}
}

func TestWpReachHitsAtTwo(t *testing.T) {
	g := wpGame(t)
	wpWield(g, g.MakeFamily("spear", 2, g.R))
	foe := wpFoeAt(g, 2, 0)
	if !g.wpFireKey() || g.Pend != "reach" {
		t.Fatal("f with a spear should target a reach strike")
	}
	g.Mode, g.Pend = "main", ""
	for i := 0; i < 20 && foe.HP == 500; i++ {
		g.wpTarget("reach", foe.X, foe.Y)
	}
	if foe.HP == 500 {
		t.Fatal("spear never hit at two tiles")
	}
	far := wpFoeAt(g, 3, 0)
	far.Y = g.P.Y + 1
	g.Cur.Set(far.X, far.Y, TFloor)
	hp := far.HP
	g.wpTarget("reach", far.X, far.Y)
	if far.HP != hp {
		t.Fatal("spear reached three tiles")
	}
	// whips reach too; swords do not
	g2 := wpGame(t)
	wpWield(g2, g2.MakeWeapon("longblade", 1, g2.R))
	if g2.wpFireKey() {
		t.Fatal("a sword has no reach")
	}
	if wpReachOf(&Actor{Body: []*BodyPart{{Slot: "hand", Item: g2.MakeFamily("whip", 1, g2.R)}}}) == nil {
		t.Fatal("whip should have reach")
	}
}

func TestWpFirearmNeedsAmmoAndReload(t *testing.T) {
	g := wpGame(t)
	g.Genre, g.Sk = "steampunk", SkinFor("steampunk")
	gun := g.MakeFamily("pistol", 2, g.R)
	gun.Extra["jam"] = 0
	gun.Extra["load"] = 0
	wpWield(g, gun)
	for _, it := range append([]*Item{}, g.P.Inv...) {
		if it.Kind == "ammo" {
			g.removeInv(it)
		}
	}
	foe := wpFoeAt(g, 3, 0)
	g.wpTarget("fire", foe.X, foe.Y)
	if gun.Extra["load"] != 0 || foe.HP != 500 {
		t.Fatal("fired without ammo")
	}
	g.give(g.g_legacyAmmo())
	g.wpTarget("fire", foe.X, foe.Y)
	if gun.Extra["load"] != 0 {
		t.Fatal("a gun must not load generic ammo")
	}
	g.give(g.MakeWpAmmo(gun.Effect, 6))
	g.wpTarget("fire", foe.X, foe.Y)
	if gun.Extra["load"] != gun.Extra["mag"] || foe.HP != 500 {
		t.Fatalf("empty fire should reload, not shoot: load %d", gun.Extra["load"])
	}
	hits := 0
	for i := 0; i < 12; i++ {
		before := foe.HP
		g.wpTarget("fire", foe.X, foe.Y) // shot (load 1 → 0), then reload, ...
		if foe.HP < before {
			hits++
		}
	}
	if hits == 0 {
		t.Fatal("the pistol never hit")
	}
	// a repeater fires bursts
	g.Genre, g.Sk = "dieselpunk", SkinFor("dieselpunk")
	smg := g.MakeFamily("repeater", 2, g.R)
	g.P.Equipped(gun).Item = nil
	if smg.Extra["burst"] != 3 || smg.Extra["mag"] != 20 {
		t.Fatalf("diesel SMG profile %+v", smg.Extra)
	}
	wpWield(g, smg)
	g.wpTarget("fire", foe.X, foe.Y)
	if smg.Extra["load"] != 17 {
		t.Fatalf("burst should spend 3 rounds, load %d", smg.Extra["load"])
	}
}

func (g *Game) g_legacyAmmo() *Item { return g.MakeAmmo(10) }

func TestWpThrownLandsAndReturns(t *testing.T) {
	g := wpGame(t)
	foe := wpFoeAt(g, 3, 0)
	jav := g.MakeFamily("javelin", 2, g.R)
	jav.Count = 1
	g.give(jav)
	tx, ty := foe.X, foe.Y
	g.throw(jav, tx, ty)
	landed := false
	for _, it := range g.Cur.Items {
		if it == jav && abs(it.X-tx) <= 1 && it.Y == ty {
			landed = true
		}
	}
	if !landed {
		for _, m := range g.Msgs {
			t.Log(m.T)
		}
		t.Fatalf("javelin should land by the target: %d,%d foe %d,%d", jav.X, jav.Y, foe.X, foe.Y)
	}
	boom := g.MakeFamily("boomerang", 2, g.R)
	g.give(boom)
	for i := 0; i < 5; i++ {
		it := g.wpThrowPick()
		if it == nil {
			t.Fatal("boomerang did not come back")
		}
		g.wpTarget("wpthrow:"+itoa(it.ID), foe.X, foe.Y)
	}
	if foe.HP == 500 {
		t.Fatal("thrown weapons never hit")
	}
	// knives come in stacks and split
	kn := g.MakeFamily("tknife", 1, g.R)
	g.give(kn)
	n := kn.Count
	g.throw(kn, foe.X, foe.Y)
	if kn.Count != n-1 {
		t.Fatalf("stack should drop by one: %d → %d", n, kn.Count)
	}
}

func TestWpShieldBlocks(t *testing.T) {
	g := wpGame(t)
	wpWield(g, g.MakeWeapon("longblade", 2, g.R))
	foe := wpFoeAt(g, 1, 0)
	sh := g.MakeShield("tower", 2, g.R)
	sh.Extra["block"] = 100
	for _, p := range foe.Body {
		if p.Slot == "hand" {
			p.Item = sh
			break
		}
	}
	for i := 0; i < 15; i++ {
		g.swing(g.P, foe, g.P.Weapons()[0])
	}
	if foe.HP != 500 {
		t.Fatal("a 100% shield let a blow through")
	}
	// a stunned shield-bearer cannot block
	foe.AddSt("stunned", 50)
	for i := 0; i < 15; i++ {
		g.swing(g.P, foe, g.P.Weapons()[0])
	}
	if foe.HP == 500 {
		t.Fatal("stunned foe still blocked everything")
	}
	if !strings.Contains(strings.Join(g.wpDescribe(sh), " "), "block") {
		t.Fatal("shield description lacks block")
	}
}

func TestWpTwoHandedAndShieldHands(t *testing.T) {
	g := wpGame(t)
	sword := g.MakeWeapon("longblade", 1, g.R)
	sh := g.MakeShield("round", 1, g.R)
	wpWield(g, sword)
	wpWield(g, sh)
	if g.P.Equipped(sword) == nil || g.P.Equipped(sh) == nil {
		t.Fatal("sword and shield should share the hands")
	}
	gs := g.MakeFamily("great", 2, g.R)
	wpWield(g, gs)
	if g.P.Equipped(gs) == nil || g.P.Equipped(sword) != nil || g.P.Equipped(sh) != nil {
		t.Fatal("a greatsword needs both hands")
	}
	wpWield(g, sh)
	if g.P.Equipped(gs) != nil || g.P.Equipped(sh) == nil {
		t.Fatal("taking up a shield should put the greatsword away")
	}
}

func TestWpFamilyMechanics(t *testing.T) {
	g := wpGame(t)
	// brace: the attacker is struck first
	wpWield(g, g.MakeFamily("spear", 2, g.R))
	foe := wpFoeAt(g, 1, 0)
	g.P.AddSt("braced", 5)
	g.melee(foe, g.P)
	if g.P.St("braced") != 0 {
		t.Fatal("brace not spent")
	}
	all := ""
	for _, m := range g.Msgs {
		all += m.T + "|"
	}
	if !strings.Contains(all, "charge") {
		t.Fatal("no brace strike message")
	}
	// hooks trip, whips disarm, flails entangle (over many hits)
	for _, c := range []struct{ fam, st string }{{"hook", "prone"}, {"flail", "entangled"}} {
		g2 := wpGame(t)
		wpWield(g2, g2.MakeFamily(c.fam, 3, g2.R))
		f := wpFoeAt(g2, 1, 0)
		ok := false
		for i := 0; i < 80 && !ok; i++ {
			delete(f.Status, c.st)
			g2.swing(g2.P, f, g2.P.Weapons()[0])
			ok = f.St(c.st) > 0
		}
		if !ok {
			t.Fatalf("%s never applied %s", c.fam, c.st)
		}
	}
	g3 := wpGame(t)
	wpWield(g3, g3.MakeFamily("whip", 3, g3.R))
	f := wpFoeAt(g3, 1, 0)
	for _, p := range f.Body {
		if p.Slot == "hand" {
			p.Item = g3.MakeWeapon("axe", 1, g3.R)
			break
		}
	}
	disarmed := false
	for i := 0; i < 80 && !disarmed; i++ {
		g3.swing(g3.P, f, g3.P.Weapons()[0])
		disarmed = true
		for _, p := range f.Body {
			if p.Item != nil && p.Item.Skill == "axe" {
				disarmed = false
			}
		}
	}
	if !disarmed {
		t.Fatal("whip never disarmed")
	}
}

func TestWpSmithStocksSphereWeapons(t *testing.T) {
	g := newGame(t, nil)
	found := 0
	for _, bd := range g.W.Burgs {
		if found >= 2 || !bd.Alive || bd.Layer != "surface" {
			continue
		}
		cu := g.W.cultByID[bd.Culture]
		if cu == nil || wpSpheres[cu.Sphere] == nil {
			continue
		}
		wx, wy := bd.P%g.W.W, bd.P/g.W.W
		for zy := 0; zy < 3; zy++ {
			for zx := 0; zx < 3; zx++ {
				zn := g.GetZone(wx, wy, zx, zy, 0)
				for _, a := range zn.Actors {
					if a.Role != "smith" {
						continue
					}
					n := 0
					for _, it := range a.Inv {
						if it.Region == cu.Sphere && it.Extra["wps"] > 0 {
							n++
						}
					}
					if n < 3 {
						t.Fatalf("smith of %s (%s) has %d sphere items", bd.Name, cu.Sphere, n)
					}
					found++
				}
			}
		}
	}
	if found == 0 {
		t.Fatal("found no smith to check")
	}
}

func TestWpOldSaveSkillsLoad(t *testing.T) {
	g := newGame(t, nil)
	g.Skills["bogus_skill"] = 2
	s := g.Save()
	// an old save: no new trees in the skill store
	var m map[string]json.RawMessage
	if err := json.Unmarshal([]byte(s), &m); err != nil {
		t.Fatal(err)
	}
	g2, err := LoadGame(g.W, s)
	if err != nil {
		t.Fatal(err)
	}
	g2.skSync()
	if g2.Skills["longblade"] != g.Skills["longblade"] || g2.Skills["spear"] != 0 {
		t.Fatalf("skills changed on load: %v", g2.Skills)
	}
	if skillName("spear") != "Spear and Polearm" || skillName("longblade") != "Long Blade" {
		t.Fatal("skill names")
	}
	// the new trees can be learned and give rank
	g2.SP = 20
	if !g2.skUnlock(skTreeByID["firearm"]) || g2.Skills["firearm"] < 1 {
		t.Fatal("cannot learn Firearms")
	}
	// legacy gear still fights
	foe := placeFoe(g2, "bandit", 1)
	foe.MaxHP, foe.HP, foe.DVNat = 500, 500, -20
	for i := 0; i < 10; i++ {
		g2.melee(g2.P, foe)
	}
	if foe.HP == 500 {
		t.Fatal("old weapon never hit")
	}
}

func TestWpLootAndCrafting(t *testing.T) {
	g := newGame(t, nil)
	r := NewRng("loot")
	fams := map[string]bool{}
	for i := 0; i < 600; i++ {
		if it := g.Loot(r, 2); it != nil {
			if f := wpFamOf(it); f != "" {
				fams[f] = true
			}
		}
	}
	if len(fams) < 8 {
		t.Fatalf("loot rarely yields the new families: %v", fams)
	}
	n := 0
	for _, rc := range g.craftRecipes() {
		if strings.HasPrefix(rc.ID, "w:") || strings.HasPrefix(rc.ID, "b:wp:") {
			n++
			if rc.Make() == nil {
				t.Fatalf("recipe %s makes nothing", rc.ID)
			}
		}
	}
	if n < 50 {
		t.Fatalf("only %d weapon recipes", n)
	}
	it := g.MakeFamily("pistol", 1, r)
	modByID("extended").Apply(it)
	if it.Extra["mag"] < 2 {
		t.Fatal("extended magazine did nothing")
	}
}
