package engine

import (
	"strings"
	"testing"
)

var augGenres = []string{"fantasy", "mythic", "steampunk", "dieselpunk", "atompunk", "cyberpunk", "biopunk", "solarpunk"}

func TestAugmentEveryGenreNamesEveryAugment(t *testing.T) {
	if n := len(AugDefs); n < 14 || n > 20 {
		t.Fatalf("want 14-20 augments, got %d", n)
	}
	for _, gen := range augGenres {
		sk, ok := augSkins[gen]
		if !ok {
			t.Fatalf("no augment skin for %s", gen)
		}
		if sk.Noun == "" || sk.Installer == "" || sk.Lore == "" || sk.Purist == "" || !strings.Contains(sk.Install, "%s") {
			t.Fatalf("%s skin incomplete: %+v", gen, sk)
		}
		seen := map[string]bool{}
		for _, d := range AugDefs {
			n := sk.Names[d.ID]
			if n == "" {
				t.Fatalf("%s has no name for %s", gen, d.ID)
			}
			if seen[n] {
				t.Fatalf("%s names %q twice", gen, n)
			}
			seen[n] = true
		}
		if len(sk.Names) != len(AugDefs) {
			t.Fatalf("%s has %d names for %d augments", gen, len(sk.Names), len(AugDefs))
		}
		if _, ok := augInstallerDistricts[gen]; !ok {
			t.Fatalf("%s has no installer districts", gen)
		}
	}
	for _, d := range AugDefs {
		if augSlotNames[d.Slot] == "" || d.Cost < 1 {
			t.Fatalf("bad def %+v", d)
		}
		if d.Power != "" && powerByID[d.Power] == nil {
			t.Fatalf("%s grants unknown power %s", d.ID, d.Power)
		}
	}
}

func augGame(t *testing.T) *Game {
	g := newGame(t, nil)
	g.P.Attr[TOU] = 16 // modifier 0
	g.P.Level = 1      // capacity 3
	for _, p := range g.P.Body {
		p.Item = nil
	}
	return g
}

func TestAugmentInstallEffectAndCapacity(t *testing.T) {
	g := augGame(t)
	if c := g.P.AugCapacity(); c != 3 {
		t.Fatalf("capacity %d, want 3", c)
	}
	str, av, dv := g.P.Score(STR), g.P.AV(), g.P.DV()
	servo := g.MakeAugment("servo")
	g.P.Inv = append(g.P.Inv, servo)
	if why := g.InstallAug(servo); why != "" {
		t.Fatal(why)
	}
	if g.P.Score(STR) != str+2 || g.P.AugLoad() != 2 {
		t.Fatalf("STR %d→%d load %d", str, g.P.Score(STR), g.P.AugLoad())
	}
	for _, it := range g.P.Inv {
		if it == servo {
			t.Fatal("installed augment still in the pack")
		}
	}
	eye := g.MakeAugment("nighteye")
	g.P.Inv = append(g.P.Inv, eye)
	if why := g.InstallAug(eye); why != "" || g.P.Pow("darkvision") != 1 {
		t.Fatalf("night eye: %q pow %d", why, g.P.Pow("darkvision"))
	}
	derm := g.MakeAugment("dermal")
	g.P.Inv = append(g.P.Inv, derm)
	if why := g.InstallAug(derm); !strings.Contains(why, "capacity") {
		t.Fatalf("capacity not enforced: %q", why)
	}
	if g.P.AV() != av {
		t.Fatal("failed install changed AV")
	}
	g.P.Level = 5 // capacity 4: still too little for 3+2
	if g.InstallAug(derm) == "" {
		t.Fatal("capacity 4 accepted load 5")
	}
	g.P.Level = 10
	if why := g.InstallAug(derm); why != "" || g.P.AV() != av+2 {
		t.Fatalf("dermal: %q AV %d→%d", why, av, g.P.AV())
	}
	st := g.MakeAugment("stride")
	g.P.Inv = append(g.P.Inv, st)
	g.P.Level = 20
	if why := g.InstallAug(st); why != "" || g.P.DV() != dv+2 {
		t.Fatalf("stride: %q DV %d→%d", why, dv, g.P.DV())
	}
	// one augment per part: a second face augment needs a free face
	hk := g.MakeAugment("hawkeye")
	g.P.Inv = append(g.P.Inv, hk)
	if why := g.InstallAug(hk); !strings.Contains(why, "no free") {
		t.Fatalf("second face augment: %q", why)
	}
	// a second arm augment fits the other hand
	s2 := g.MakeAugment("servo")
	g.P.Inv = append(g.P.Inv, s2)
	g.P.Level = 25
	if why := g.InstallAug(s2); why != "" || g.P.Score(STR) != str+4 {
		t.Fatalf("second servo: %q STR %d", why, g.P.Score(STR))
	}
	g.augSheet()
	if d := servo.Describe(g); len(d) < 3 {
		t.Fatalf("describe %v", d)
	}
}

func TestAugmentRemovalRestores(t *testing.T) {
	g := augGame(t)
	q, light := g.P.Quick(), g.P.LightRadius()
	sp := g.MakeAugment("sprinter")
	lm := g.MakeAugment("lumen")
	g.P.Inv = append(g.P.Inv, sp, lm)
	if g.InstallAug(sp) != "" || g.InstallAug(lm) != "" {
		t.Fatal("install failed")
	}
	if g.P.Quick() != q+5 || g.P.LightRadius() < 4 {
		t.Fatalf("quick %d light %d", g.P.Quick(), g.P.LightRadius())
	}
	found := false
	for _, ab := range g.Abilities() {
		if ab.ID == "sprint" && ab.Name == sp.Name {
			found = true
		}
	}
	if !found {
		t.Fatal("sprint ability not granted")
	}
	for _, p := range g.P.Body {
		if p.Aug != nil {
			if g.RemoveAug(p) == nil {
				t.Fatal("remove returned nil")
			}
		}
	}
	if g.P.Quick() != q || g.P.LightRadius() != light || g.P.AugLoad() != 0 || g.P.Pow("sprint") != 0 {
		t.Fatalf("not restored: quick %d light %d load %d", g.P.Quick(), g.P.LightRadius(), g.P.AugLoad())
	}
	n := 0
	for _, it := range g.P.Inv {
		if it == sp || it == lm {
			n++
		}
	}
	if n != 2 {
		t.Fatal("removed augments not returned to the pack")
	}
}

func TestAugmentSeveredLimbDisables(t *testing.T) {
	g := augGame(t)
	bl := g.MakeAugment("blade")
	g.P.Inv = append(g.P.Inv, bl)
	if g.InstallAug(bl) != "" {
		t.Fatal("install failed")
	}
	var part *BodyPart
	for _, p := range g.P.Body {
		if p.Aug == bl {
			part = p
		}
	}
	if part == nil || part.Slot != "hand" {
		t.Fatal("blade not in a hand")
	}
	armed := false
	for _, w := range g.P.Weapons() {
		if w == bl {
			armed = true
		}
	}
	if !armed {
		t.Fatal("blade augment is not a natural weapon")
	}
	str := g.P.Score(STR)
	sv := g.MakeAugment("servo")
	g.P.Inv = append(g.P.Inv, sv)
	g.P.Level = 10
	if g.InstallAug(sv) != "" || g.P.Score(STR) != str+2 {
		t.Fatal("servo install failed")
	}
	// cut off every limb but the head
	for _, p := range g.P.Body {
		if p.Limb && p.Slot != "head" {
			p.Severed = true
		}
	}
	if g.P.Score(STR) != str {
		t.Fatalf("severed servo still works: STR %d", g.P.Score(STR))
	}
	for _, w := range g.P.Weapons() {
		if w == bl {
			t.Fatal("severed blade still swings")
		}
	}
	if g.P.AugLoad() != 4 {
		t.Fatal("severed augments should still take capacity")
	}
	for _, p := range g.P.Body {
		p.Severed = false
	}
	if g.P.Score(STR) != str+2 {
		t.Fatal("regrown limb did not restore the servo")
	}
	// dismemberment path reports the silenced augment
	for _, p := range g.P.Body {
		if p.Limb && p.Slot != "head" && p.Aug == nil {
			p.Severed = true
		}
	}
	g.dismember(g.P)
	if g.P.Score(STR) == str+2 && func() bool {
		for _, p := range g.P.Body {
			if p.Aug == sv && p.Severed {
				return true
			}
		}
		return false
	}() {
		t.Fatal("dismembered servo still works")
	}
}

func TestAugmentResistAndHeal(t *testing.T) {
	g := augGame(t)
	g.P.Level = 10
	wd, md := g.MakeAugment("ward"), g.MakeAugment("bulwark")
	g.P.Inv = append(g.P.Inv, wd, md)
	if g.InstallAug(wd) != "" || g.InstallAug(md) != "" {
		t.Fatal("install failed")
	}
	if augResist(g.P, 9, "fire") != 6 || augResist(g.P, 9, "spikes") != 9 {
		t.Fatal("ward resist wrong")
	}
	var ab *Ability
	for _, a := range g.Abilities() {
		if a.ID == "forcebubble" {
			a := a
			ab = &a
		}
	}
	if ab == nil {
		t.Fatal("no bulwark ability")
	}
	av := g.P.AV()
	g.startAbility(*ab)
	if g.P.AV() <= av || g.P.Cool["forcebubble"] == 0 {
		t.Fatalf("bubble: AV %d→%d cool %d", av, g.P.AV(), g.P.Cool["forcebubble"])
	}
}

func TestAugmentSaveLoad(t *testing.T) {
	g := augGame(t)
	cx := g.MakeAugment("cortex")
	g.P.Inv = append(g.P.Inv, cx, g.MakeAugment("heart"))
	if g.InstallAug(cx) != "" {
		t.Fatal("install failed")
	}
	in := g.P.Score(INT)
	h, err := LoadGame(g.W, g.Save())
	if err != nil {
		t.Fatal(err)
	}
	if h.P.Score(INT) != in || h.P.AugLoad() != 2 {
		t.Fatalf("lost in round trip: INT %d load %d", h.P.Score(INT), h.P.AugLoad())
	}
	found := false
	for _, it := range h.P.Inv {
		if it.Kind == "augment" && it.Effect == "heart" {
			found = true
		}
	}
	if !found {
		t.Fatal("uninstalled augment lost")
	}
}

func TestAugmentInstallerInCity(t *testing.T) {
	g := augGame(t)
	w := g.W
	var inst *Actor
	for i := range w.Burgs {
		b := &w.Burgs[i]
		if !b.Alive || b.Layer != "surface" || w.cityTier[b.ID] < 1 {
			continue
		}
		cp := g.cityPlanFor(b)
		s, ok := g.augInstallerSpot(cp)
		if !ok {
			continue
		}
		n := 0
		for zy := 0; zy < 3; zy++ {
			for zx := 0; zx < 3; zx++ {
				if _, in := cp.districts[[2]int{zx, zy}]; !in {
					continue
				}
				zn := g.GetZone(b.P%w.W, b.P/w.W, zx, zy, 0)
				for _, a := range zn.Actors {
					if a.Role == "augmenter" {
						n++
						if [2]int{zx, zy} != s {
							t.Fatalf("installer outside its district in %s", b.Name)
						}
						inst = a
					}
				}
			}
		}
		if n != 1 {
			t.Fatalf("%s has %d installers", b.Name, n)
		}
		break
	}
	if inst == nil {
		t.Fatal("no installer found in any city")
	}
	if inst.Title != augSkins[g.Genre].Installer && g.Genre != "" {
		t.Fatalf("installer title %q", inst.Title)
	}
	g.Talk = inst
	g.Mode = "talk"
	g.talkMenu(g.greeting(inst))
	var open *Option
	for i := range g.Opts {
		if strings.Contains(g.Opts[i].Text, "Install or remove") {
			open = &g.Opts[i]
		}
	}
	if open == nil {
		t.Fatal("no installer talk option")
	}
	open.Do()
	g.P.Gold = 100000
	before := len(inst.Inv)
	var buy *Option
	for i := range g.Opts {
		if strings.HasPrefix(g.Opts[i].Text, "Buy and install") {
			buy = &g.Opts[i]
			break
		}
	}
	if buy == nil {
		t.Fatalf("nothing to buy: %v", g.Opts)
	}
	buy.Do()
	if g.P.AugLoad() == 0 || len(inst.Inv) != before-1 || g.P.Gold >= 100000 {
		t.Fatalf("buy and install failed: load %d stock %d→%d", g.P.AugLoad(), before, len(inst.Inv))
	}
	var rem *Option
	for i := range g.Opts {
		if strings.HasPrefix(g.Opts[i].Text, "Remove the") {
			rem = &g.Opts[i]
		}
	}
	if rem == nil {
		t.Fatal("no remove option")
	}
	rem.Do()
	if g.P.AugLoad() != 0 {
		t.Fatal("remove failed")
	}
}

func TestAugmentPuristsAndLoot(t *testing.T) {
	g := augGame(t)
	g.W.relByID[99901] = &ReligionData{ID: 99901, Theology: "animism"}
	g.W.relByID[99902] = &ReligionData{ID: 99902, Doctrine: "esoteric"}
	defer delete(g.W.relByID, 99901)
	defer delete(g.W.relByID, 99902)
	pur, fan := &Actor{Religion: 99901}, &Actor{Religion: 99902}
	if g.augPriceMod(pur) != 1 || g.augRemark(pur) != "" {
		t.Fatal("purist reacts to an unaugmented wanderer")
	}
	g.P.Level = 10
	for _, id := range []string{"dermal", "servo"} {
		it := g.MakeAugment(id)
		g.P.Inv = append(g.P.Inv, it)
		if g.InstallAug(it) != "" {
			t.Fatal("install failed")
		}
	}
	if g.augPriceMod(pur) <= 1 || g.augRemark(pur) == "" || g.augPriceMod(fan) >= 1 {
		t.Fatalf("attitudes: purist %v fan %v", g.augPriceMod(pur), g.augPriceMod(fan))
	}
	got := 0
	for i := 0; i < 200; i++ {
		if it := g.augLoot("k"+string(rune('a'+i%26))+string(rune('a'+i/26)), 0.5); it != nil {
			got++
			if it.Kind != "augment" || augByID[it.Effect] == nil {
				t.Fatalf("bad loot %+v", it)
			}
		}
	}
	if got == 0 || got == 200 {
		t.Fatalf("loot chance off: %d/200", got)
	}
	a, b := g.augLoot("same", 1), g.augLoot("same", 1)
	if a.Effect != b.Effect {
		t.Fatal("loot not deterministic")
	}
}
