package engine

import (
	"fmt"
	"strings"
)

// Weapon and armour variety (wp). New weapon families with their own mechanics, genre firearms with
// magazines and reloading, throwing weapons, shields as an off-hand class, armour types and culture-sphere
// gear for smiths. Items stay Kind weapon/ranged/armor so rarity, crafting mods and inventory work as before;
// family data rides in Item.Extra: wpf (family index+1), reach, mag/load/rl/burst/jam/smart (guns),
// ret (return %) / ent (entangles) (thrown), block (shields), bal (ballistic armour).

// ---- skills ---------------------------------------------------------------

// wpSkillIDs are the new weapon skills (trees on K). Greatswords use the existing "heavy" tree,
// shields the existing "shield" tree.
var wpSkillIDs = []string{"spear", "flail", "whip", "staff", "fist", "hook", "thrown", "firearm"}

var wpSkillNames = map[string]string{"spear": "Spear and Polearm", "flail": "Flail", "whip": "Whip", "staff": "Staff", "fist": "Fist Weapon",
	"hook": "Hooked Blade", "thrown": "Throwing", "firearm": "Firearms", "heavy": "Heavy Weapon", "shield": "Shield"}

var wpTrees = []*TreeDef{
	{ID: "spear", Name: "Spear and Polearm", Unlock: 50, Attr: -1, Nodes: []NodeDef{
		{ID: "sp_prof", Name: "Spear Proficiency", Kind: "passive", Desc: "Better aim with spears and polearms."},
		{ID: "brace", Name: "Brace", Q: 100, Kind: "active", Cool: 12, Desc: "Set your spear: the next foe to close on you is struck first."},
		{ID: "impale", Name: "Impale", Q: 150, Kind: "active", Cool: 10, Desc: "Thrust at a foe up to two tiles away; it is pinned and bleeds."},
		{ID: "sp_phalanx", Name: "Phalanx", Q: 200, Req: "brace", Kind: "passive", DV: 1, Desc: "+1 DV."},
	}},
	{ID: "flail", Name: "Flail", Unlock: 50, Attr: -1, Nodes: []NodeDef{
		{ID: "fl_prof", Name: "Flail Proficiency", Kind: "passive", Desc: "Better aim with flails."},
		{ID: "smash", Name: "Overhead Smash", Q: 150, Kind: "active", Cool: 10, Desc: "A heavy blow over the shield: ignores blocks, often stuns."},
		{ID: "fl_whirl", Name: "Whirling Chain", Q: 100, Kind: "passive", HP: 4, Desc: "+4 max HP."},
	}},
	{ID: "whip", Name: "Whip", Unlock: 50, Attr: -1, Nodes: []NodeDef{
		{ID: "wh_prof", Name: "Whip Proficiency", Kind: "passive", Desc: "Better aim with whips."},
		{ID: "lash", Name: "Lash", Q: 100, Kind: "active", Cool: 8, Desc: "Lash a foe up to two tiles away: disarm it or pull it off its feet."},
		{ID: "wh_crack", Name: "Crack of the Whip", Q: 150, Kind: "passive", DV: 1, Desc: "+1 DV."},
	}},
	{ID: "staff", Name: "Staff", Unlock: 50, Attr: -1, Nodes: []NodeDef{
		{ID: "st_prof", Name: "Staff Proficiency", Kind: "passive", Desc: "Better aim with staves."},
		{ID: "stthrust", Name: "Thrust Back", Q: 100, Kind: "active", Cool: 8, Desc: "Drive a foe back three tiles and daze it."},
		{ID: "st_guard", Name: "Staff Guard", Q: 150, Kind: "passive", DV: 1, Desc: "+1 DV."},
	}},
	{ID: "fist", Name: "Fist Weapon", Unlock: 50, Attr: -1, Nodes: []NodeDef{
		{ID: "fi_prof", Name: "Fist Proficiency", Kind: "passive", Desc: "Better aim with fist weapons and claws."},
		{ID: "combo", Name: "Combination", Q: 100, Kind: "active", Cool: 8, Desc: "Three quick strikes with your fist weapon."},
		{ID: "fi_iron", Name: "Iron Knuckles", Q: 150, Kind: "passive", HP: 3, Desc: "+3 max HP."},
	}},
	{ID: "hook", Name: "Hooked Blade", Unlock: 50, Attr: -1, Nodes: []NodeDef{
		{ID: "hk_prof", Name: "Hooked Blade Proficiency", Kind: "passive", Desc: "Better aim with sickles and hooked swords."},
		{ID: "hooktrip", Name: "Hook and Trip", Q: 100, Kind: "active", Cool: 8, Desc: "Hook a leg: on a hit the foe falls."},
		{ID: "hk_reap", Name: "Reaper's Footing", Q: 150, Kind: "passive", DV: 1, Desc: "+1 DV."},
	}},
	{ID: "thrown", Name: "Throwing", Unlock: 50, Attr: -1, Nodes: []NodeDef{
		{ID: "tw_prof", Name: "Throwing Proficiency", Kind: "passive", Desc: "Better aim with thrown weapons."},
		{ID: "deadeye", Name: "Deadeye Throw", Q: 100, Kind: "active", Cool: 8, Desc: "A careful throw: +4 to hit and +3 damage."},
		{ID: "tw_catch", Name: "Sure Catch", Q: 150, Kind: "passive", Desc: "Returning weapons always come back to your hand."},
	}},
	{ID: "firearm", Name: "Firearms", Unlock: 100, Attr: -1, Nodes: []NodeDef{
		{ID: "fa_prof", Name: "Firearm Proficiency", Kind: "passive", Desc: "Better aim with guns; fewer jams."},
		{ID: "steadyaim", Name: "Steady Aim", Q: 100, Kind: "active", Cool: 15, Desc: "Take aim: your next shot gets +4 to hit and +1 penetration."},
		{ID: "fa_reload", Name: "Quick Reload", Q: 150, Kind: "passive", Desc: "Reloading takes half the time."},
		{ID: "fa_sling", Name: "Gunslinger", Q: 200, Req: "fa_reload", Kind: "passive", DV: 1, Desc: "+1 DV."},
	}},
}

func init() {
	for _, t := range wpTrees {
		skTrees = append(skTrees, t)
		skTreeByID[t.ID] = t
		for i := range t.Nodes {
			skNodeByID[t.Nodes[i].ID] = &t.Nodes[i]
			skNodeTree[t.Nodes[i].ID] = t
		}
	}
	for _, id := range []string{"brace", "steadyaim"} {
		skSelfTarget[id] = true
	}
	skillDesc["spear"] = "+hit with spears and polearms; they strike two tiles away (f)."
	skillDesc["flail"] = "+hit with flails; they swing past shields and can entangle."
	skillDesc["whip"] = "+hit with whips; reach two tiles (f), may disarm."
	skillDesc["staff"] = "+hit with staves; two-handed, +DV, knock foes back."
	skillDesc["fist"] = "+hit with fist weapons; off-hand strikes land often."
	skillDesc["hook"] = "+hit with hooked blades; they trip foes."
	skillDesc["thrown"] = "+hit and range with thrown weapons."
	skillDesc["firearm"] = "+hit with guns, fewer jams."
	skillDesc["heavy"] = "+hit with two-handers; they cleave a second foe."
	skillDesc["shield"] = "+5% block chance per rank with a shield."
	for _, d := range []*skStatusDef{
		{ID: "braced", Name: "Braced", Glyph: 'B'},
		{ID: "aiming", Name: "Aiming", Glyph: 'a'},
		{ID: "entangled", Name: "Entangled", Glyph: '&', DV: -2, Speed: -50},
	} {
		skAddStatus(d)
	}
	zoneHooks = append(zoneHooks, wpZone)
	modDefs = append(modDefs,
		modDef{"balanced", "balanced", "weapon", map[int]int{2: 3}, func(it *Item) {
			if it.Skill == "thrown" {
				it.Range += 2
			}
			addBonus(it, AGI, 1)
		}},
		modDef{"extended", "extended-magazine", "ranged", map[int]int{3: 3}, func(it *Item) {
			if it.Extra != nil && it.Extra["mag"] > 0 {
				it.Extra["mag"] += maxi(1, it.Extra["mag"]/2)
			}
		}},
	)
}

// ---- families -----------------------------------------------------------

func wpFamIdx(fam string) int {
	for i, f := range wpFamilies {
		if f == fam {
			return i + 1
		}
	}
	return 0
}

// wpFamOf is the family id of an item ("" for legacy gear).
func wpFamOf(it *Item) string {
	if it == nil || it.Extra == nil {
		return ""
	}
	if i := it.Extra["wpf"]; i > 0 && i <= len(wpFamilies) {
		return wpFamilies[i-1]
	}
	return ""
}

func wpGenre(g string) string {
	if _, ok := wpNouns[g]; ok {
		return g
	}
	return "fantasy"
}

// wpNoun is a family's genre noun at a tier.
func wpNoun(genre, fam string, tier int) string {
	n := wpNouns[wpGenre(genre)][fam]
	if tier >= 3 && n[1] != "" {
		return n[1]
	}
	return n[0]
}

func wpGunOf(genre string) wpGun { return wpGuns[wpGenre(genre)] }

// wpAmmoName names the ammo of a tag in the current genre.
func (g *Game) wpAmmoName(tag string) string {
	if n, ok := wpAmmoNames[tag]; ok {
		if tag == "stone" && (g.Genre == "dieselpunk" || g.Genre == "atompunk" || g.Genre == "cyberpunk" || g.Genre == "wasteland") {
			return "steel shot"
		}
		return n
	}
	gp := wpGunOf(g.Genre)
	if gp.Tag == tag {
		return gp.Ammo
	}
	for _, gen := range wpGenres {
		if wpGuns[gen].Tag == tag {
			return wpGuns[gen].Ammo
		}
	}
	return g.Sk.Ammo
}

// wpAmmoLabel is what a ranged weapon fires (legacy weapons use the skin's ammo).
func (g *Game) wpAmmoLabel(it *Item) string {
	if it.Effect == "" {
		return g.Sk.Ammo
	}
	return g.wpAmmoName(it.Effect)
}

// MakeWpAmmo builds a stack of tagged ammo.
func (g *Game) MakeWpAmmo(tag string, n int) *Item {
	return &Item{ID: g.newID(), Kind: "ammo", Name: g.wpAmmoName(tag), Effect: tag, Glyph: '¡', F: CGray, Count: n, Value: 1 + map[string]int{"cell": 2, "round": 1, "pod": 1}[tag]}
}

// MakeFamily builds a weapon of a new family (wpFamilies, not "shield") at a tier 0-5.
func (g *Game) MakeFamily(fam string, tier int, r *Rng) *Item {
	if fam == "shield" {
		return g.MakeShield(wpShieldOrder[r.Int(0, 4)], tier, r)
	}
	t := clampi(tier, 0, 5)
	it := &Item{ID: g.newID(), Kind: "weapon", Slot: "hand", Tier: t, F: tierColor[t], Weight: 3, Extra: map[string]int{"wpf": wpFamIdx(fam)}}
	it.Name = g.Sk.Mats[t] + " " + wpNoun(g.Genre, fam, t)
	val := 8 * (t + 1) * (t + 1)
	switch fam {
	case "spear":
		it.Skill, it.Glyph, it.DN, it.DS, it.PV, it.MaxStr, it.Two, it.Weight = "spear", '↑', 1, 6+t, 2+t/2, 3+t, true, 5
		it.Extra["reach"] = 2
	case "flail":
		it.Skill, it.Glyph, it.DN, it.DS, it.PV, it.MaxStr, it.Weight = "flail", 'ƒ', 1, 6+t, 2+t/2, 4+t, 6
	case "whip":
		it.Skill, it.Glyph, it.DN, it.DS, it.PV, it.MaxStr, it.Weight = "whip", '§', 1, 3+t/2, 1+t/3, 1+t/2, 2
		it.Extra["reach"] = 2
	case "staff":
		it.Skill, it.Glyph, it.DN, it.DS, it.PV, it.MaxStr, it.Two, it.Weight = "staff", '│', 1, 5+t/2, 1+t/2, 3+t, true, 4
		it.DV = 1 + t/3
	case "fist":
		it.Skill, it.Glyph, it.DN, it.DS, it.PV, it.MaxStr, it.Weight = "fist", '¬', 1, 3+t/2, 2+t/2, 2+t/2, 1
	case "hook":
		it.Skill, it.Glyph, it.DN, it.DS, it.PV, it.MaxStr, it.Weight = "hook", 'Γ', 1, 5+t, 2+t/2, 3+t/2, 3
	case "great":
		it.Skill, it.Glyph, it.DN, it.DS, it.PV, it.MaxStr, it.Two, it.Weight = "heavy", '†', 2, 4+t/2, 2+t/2, 5+t, true, 9
		val += val / 3
	case "javelin", "taxe", "tknife", "chakram", "boomerang", "bolas":
		it.Skill, it.Glyph = "thrown", '¤'
		switch fam {
		case "javelin":
			it.DN, it.DS, it.PV, it.MaxStr, it.Range, it.Weight = 1, 6+t, 3+t/2, 3+t, 8, 3
			it.Count = 2
		case "taxe":
			it.DN, it.DS, it.PV, it.MaxStr, it.Range, it.Weight = 1, 6+t, 2+t/2, 3+t, 6, 2
			it.Count = 2
		case "tknife":
			it.DN, it.DS, it.PV, it.MaxStr, it.Range, it.Weight = 1, 3+t/2, 2+t/2, 2, 7, 1
			it.Count = 4
		case "chakram":
			it.DN, it.DS, it.PV, it.MaxStr, it.Range, it.Weight = 1, 5+t/2, 2+t/2, 2+t/2, 8, 1
			it.Extra["ret"] = 60
		case "boomerang":
			it.DN, it.DS, it.PV, it.MaxStr, it.Range, it.Weight = 1, 3+t/2, 1+t/3, 2, 8, 1
			it.Extra["ret"] = 100
		case "bolas":
			it.DN, it.DS, it.PV, it.MaxStr, it.Range, it.Weight = 1, 2+t/2, 1, 1, 6, 1
			it.Extra["ent"] = 1
			it.Count = 2
		}
	case "sling", "blowgun", "crossbow", "longbow", "atlatl":
		it.Kind, it.Skill, it.Glyph = "ranged", "missile", '}'
		switch fam {
		case "sling":
			it.Effect, it.DN, it.DS, it.PV, it.Range, it.Weight = "stone", 1, 4+t, 2+t/2, 8+t, 1
		case "blowgun":
			it.Effect, it.DN, it.DS, it.PV, it.Range, it.Weight, it.Elem = "dart", 1, 2+t/2, 1+t/3, 7+t/2, 1, "poison"
		case "crossbow":
			it.Effect, it.DN, it.DS, it.PV, it.Range, it.Weight, it.Two = "bolt", 1, 8+t, 4+t/2, 12+t, 6, true
			it.Extra["mag"], it.Extra["load"], it.Extra["rl"], it.Extra["burst"] = 1, 1, 1500, 1
			if wpNoun(g.Genre, fam, t) == "repeating crossbow" {
				it.Extra["mag"], it.Extra["load"] = 5, 5
			}
		case "longbow":
			it.Effect, it.DN, it.DS, it.PV, it.Range, it.Weight, it.Two = "arrow", 1, 7+t, 3+t/2, 14+t, 3, true
		case "atlatl":
			it.Effect, it.DN, it.DS, it.PV, it.Range, it.Weight = "dart", 1, 7+t, 3+t/2, 10+t, 2
		}
	case "pistol", "longgun", "repeater":
		gp := wpGunOf(g.Genre)
		k := map[string]int{"pistol": 0, "longgun": 1, "repeater": 2}[fam]
		it.Kind, it.Skill, it.Glyph, it.Effect, it.Elem = "ranged", "firearm", '⌐', gp.Tag, gp.Elem
		switch k {
		case 0:
			it.DN, it.DS, it.PV, it.Range, it.Weight = 1, 6+t, 4+t/2+gp.PV, 8+t, 2
		case 1:
			it.DN, it.DS, it.PV, it.Range, it.Weight, it.Two = 1, 9+t, 5+t/2+gp.PV, 14+t, 6, true
		case 2:
			it.DN, it.DS, it.PV, it.Range, it.Weight, it.Two = 1, 5+t, 3+t/2+gp.PV, 10+t, 7, true
		}
		it.Extra["mag"], it.Extra["load"], it.Extra["rl"], it.Extra["burst"] = gp.Mag[k], gp.Mag[k], gp.RL[k], gp.Burst[k]
		if gp.Jam > 0 {
			it.Extra["jam"] = gp.Jam
		}
		if gp.Smart {
			it.Extra["smart"] = 1
		}
		val += val / 2
	}
	if it.Kind == "weapon" && it.Skill != "thrown" && t >= 3 && r.Chance(0.15) {
		e := Pick(r, []string{"fire", "frost", "shock"})
		it.Elem = e
		it.Name = map[string]string{"fire": "flaming ", "frost": "frostbitten ", "shock": "humming "}[e] + it.Name
	}
	it.Value = val
	return it
}

// MakeShield builds an off-hand shield of a type (wpShieldTypes).
func (g *Game) MakeShield(typ string, tier int, r *Rng) *Item {
	st := wpShieldTypes[typ]
	if st == nil {
		typ, st = "round", wpShieldTypes["round"]
	}
	t := clampi(tier, 0, 5)
	name := st.Name
	if t >= 2 {
		name = g.Sk.Mats[t] + " " + name
	}
	it := &Item{ID: g.newID(), Kind: "armor", Slot: "hand", Glyph: ']', F: tierColor[t], Tier: t, Name: name, AV: st.AV + t/3, DV: st.DV, Weight: st.W,
		Value: 7 * (t + 1) * (t + 1), Extra: map[string]int{"wpf": wpFamIdx("shield"), "block": st.Block + 2*t}}
	if st.Full {
		it.Extra["full"] = 1
	}
	return it
}

// wpGenreShields are the shield types a genre's smiths make.
func wpGenreShields(genre string) []string {
	var out []string
	for _, k := range wpShieldOrder {
		if wpAllowed(wpShieldTypes[k].Genres, genre) {
			out = append(out, k)
		}
	}
	return out
}

func wpAllowed(gs []string, genre string) bool {
	if len(gs) == 0 {
		return true
	}
	for _, x := range gs {
		if x == genre {
			return true
		}
	}
	return false
}

// MakeArmorType builds body armour of a type (wpArmorTypes).
func (g *Game) MakeArmorType(typ string, tier int, r *Rng) *Item {
	at := wpArmorTypes[typ]
	if at == nil {
		return g.MakeArmor("body", tier, r)
	}
	t := clampi(tier, 0, 5)
	name := at.Name
	if n := at.Names[g.Genre]; n != "" {
		name = n
	}
	if t >= 2 {
		name = g.Sk.Mats[t] + " " + name
	}
	it := &Item{ID: g.newID(), Kind: "armor", Slot: at.Slot, Glyph: '[', F: tierColor[t], Tier: t, Name: name, AV: at.AV + t/2, DV: at.DV, Weight: at.W,
		Value: 7 * (t + 1) * (t + 1), Extra: map[string]int{}}
	if b := wpBallistic[typ]; b > 0 {
		it.Extra["bal"] = b
	}
	if at.STR > 0 {
		it.Bonus = map[int]int{STR: at.STR}
		it.Value += 80
	}
	return it
}

// wpGenreArmors are the armour types found in a genre.
func wpGenreArmors(genre string) []string {
	var out []string
	for _, k := range []string{"gambeson", "leather", "hide", "slat", "chitin", "mail", "scale", "lamellar", "brigandine", "cuirass", "plate", "kevlar", "powerframe"} {
		if wpAllowed(wpArmorTypes[k].Genres, genre) {
			out = append(out, k)
		}
	}
	return out
}

// wpMakeAny builds a legacy skill weapon or a new family weapon.
func (g *Game) wpMakeAny(fam string, tier int, r *Rng) *Item {
	switch fam {
	case "longblade", "shortblade", "axe", "cudgel":
		return g.MakeWeapon(fam, tier, r)
	}
	return g.MakeFamily(fam, tier, r)
}

// MakeSphereWeapon builds one of a culture sphere's weapons (idx < 0 picks one).
func (g *Game) MakeSphereWeapon(sphere string, idx, tier int, r *Rng) *Item {
	k := wpSpheres[sphere]
	if k == nil || len(k.Weapons) == 0 {
		return g.MakeWeapon("longblade", tier, r)
	}
	if idx < 0 || idx >= len(k.Weapons) {
		idx = r.Int(0, len(k.Weapons)-1)
	}
	e := k.Weapons[idx]
	it := g.wpMakeAny(e.Fam, tier, r)
	g.wpRegional(it, e, sphere)
	return it
}

func (g *Game) wpRegional(it *Item, e wpGear, sphere string) {
	pre := ""
	if strings.HasPrefix(it.Name, "flaming ") || strings.HasPrefix(it.Name, "frostbitten ") || strings.HasPrefix(it.Name, "humming ") {
		pre = it.Name[:strings.Index(it.Name, " ")+1]
	}
	name := e.Name
	if it.Tier >= 2 {
		name = g.Sk.Mats[it.Tier] + " " + name
	}
	it.Name, it.Region, it.Lore = pre+name, sphere, e.Desc
	it.Value += it.Value / 4
	if it.Extra == nil {
		it.Extra = map[string]int{}
	}
	it.Extra["wps"] = 1
}

// MakeSphereArmor builds one of a sphere's armours, or its shield when shield is true.
func (g *Game) MakeSphereArmor(sphere string, shield bool, tier int, r *Rng) *Item {
	k := wpSpheres[sphere]
	if k == nil {
		return g.MakeArmorType("mail", tier, r)
	}
	var e wpGear
	var it *Item
	if shield {
		if k.Shield == nil {
			return nil
		}
		e = *k.Shield
		it = g.MakeShield(e.Fam, tier, r)
	} else {
		if len(k.Armor) == 0 {
			return nil
		}
		e = Pick(r, k.Armor)
		it = g.MakeArmorType(e.Fam, tier, r)
	}
	g.wpRegional(it, e, sphere)
	return it
}

// ---- loot ----------------------------------------------------------------

var wpMeleeFams = []string{"spear", "flail", "whip", "staff", "fist", "hook", "great", "javelin", "taxe", "tknife", "chakram", "boomerang", "bolas"}

var wpGunGenres = map[string]bool{"steampunk": true, "dieselpunk": true, "atompunk": true, "cyberpunk": true, "biopunk": true, "wasteland": true, "twilight": true}

// wpLoot: a new-family melee or thrown weapon, sometimes a far land's sphere weapon.
func (g *Game) wpLoot(r *Rng, tier int) *Item {
	if r.Chance(0.2) {
		sp := attireSpheresSorted() // sorted: map order would break determinism
		s := sp[r.Int(0, len(sp)-1)]
		if wpSpheres[s] != nil {
			return g.MakeSphereWeapon(s, -1, tier, r)
		}
	}
	return g.MakeFamily(Pick(r, wpMeleeFams), tier, r)
}

// wpLootRanged: a sling, blowgun, crossbow, bow, spear-thrower or genre firearm.
func (g *Game) wpLootRanged(r *Rng, tier int) *Item {
	p := 0.15
	if wpGunGenres[g.Genre] {
		p = 0.6
	}
	if r.Chance(p) {
		return g.MakeFamily(Pick(r, []string{"pistol", "pistol", "longgun", "repeater"}), tier, r)
	}
	return g.MakeFamily(Pick(r, []string{"sling", "blowgun", "crossbow", "longbow", "atlatl"}), tier, r)
}

// wpLootArmor: a shield or a typed armour.
func (g *Game) wpLootArmor(r *Rng, tier int) *Item {
	if r.Chance(0.45) {
		return g.MakeShield(Pick(r, wpGenreShields(g.Genre)), tier, r)
	}
	return g.MakeArmorType(Pick(r, wpGenreArmors(g.Genre)), tier, r)
}

// wpLootAmmo: ammo for one of the new families.
func (g *Game) wpLootAmmo(r *Rng) *Item {
	tags := []string{"arrow", "bolt", "stone", "dart"}
	if wpGunGenres[g.Genre] {
		tags = append(tags, wpGunOf(g.Genre).Tag, wpGunOf(g.Genre).Tag)
	}
	return g.MakeWpAmmo(Pick(r, tags), r.Int(6, 16))
}

// ---- smiths ----------------------------------------------------------------

// wpZone adds the town culture's weapons and armour to each smith (side rng, zone generation unchanged).
func wpZone(g *Game, zn *Zone, _ *Rng) {
	if zn.Burg == 0 || g.W == nil {
		return
	}
	bd := g.W.burgByID[zn.Burg]
	if bd == nil {
		return
	}
	sphere := ""
	if cu := g.W.cultByID[bd.Culture]; cu != nil {
		sphere = cu.Sphere
	}
	for _, a := range zn.Actors {
		if a.Role == "smith" {
			g.wpStockSmith(a, sphere, NewRng(fmt.Sprintf("%s:wp:%s:%d:%d", g.W.Seed, zn.Key, a.X, a.Y)))
		}
	}
}

func (g *Game) wpStockSmith(a *Actor, sphere string, r *Rng) {
	tier := clampi(a.Level/3, 0, 5)
	if wpSpheres[sphere] != nil {
		for i := 0; i < r.Int(3, 5); i++ {
			a.Inv = append(a.Inv, g.MakeSphereWeapon(sphere, -1, tier+r.Int(-1, 1), r))
		}
		if it := g.MakeSphereArmor(sphere, false, tier+r.Int(-1, 1), r); it != nil {
			a.Inv = append(a.Inv, it)
		}
		if it := g.MakeSphereArmor(sphere, true, tier, r); it != nil && r.Chance(0.7) {
			a.Inv = append(a.Inv, it)
		}
	}
	for i := 0; i < 2; i++ {
		a.Inv = append(a.Inv, g.MakeFamily(Pick(r, wpMeleeFams), tier+r.Int(-1, 1), r))
	}
	if r.Chance(0.6) {
		a.Inv = append(a.Inv, g.MakeShield(Pick(r, wpGenreShields(g.Genre)), tier, r))
	}
	if r.Chance(0.5) {
		a.Inv = append(a.Inv, g.MakeArmorType(Pick(r, wpGenreArmors(g.Genre)), tier, r))
	}
	if r.Chance(0.5) {
		a.Inv = append(a.Inv, g.wpLootRanged(r, tier))
	}
	// ammo for whatever needs it
	seen := map[string]bool{}
	for _, it := range a.Inv {
		if it.Kind == "ranged" && it.Effect != "" && !seen[it.Effect] {
			seen[it.Effect] = true
			a.Inv = append(a.Inv, g.MakeWpAmmo(it.Effect, r.Int(12, 30)))
		}
	}
}

// ---- equipping -----------------------------------------------------------

// wpEquipped keeps two-handed melee weapons consistent: they need every other hand free (hook in equip).
func (g *Game) wpEquipped(a *Actor, part *BodyPart, it *Item) {
	if part == nil || part.Slot != "hand" {
		return
	}
	for _, p := range a.Body {
		if p == part || p.Slot != "hand" || p.Severed || p.Item == nil {
			continue
		}
		if it.Kind == "weapon" && it.Two || p.Item.Kind == "weapon" && p.Item.Two {
			o := p.Item
			p.Item = nil
			a.Inv = append(a.Inv, o)
			if a == g.P {
				g.Msg(fmt.Sprintf("{{K|You put away %s to free your hands.}}", o.Name))
			}
		}
	}
}

// ---- combat hooks --------------------------------------------------------

var (
	wpCleaving     bool // guards the greatsword cleave from recursing
	wpIgnoreShield bool // Overhead Smash
)

// wpShieldOf is the shield an actor holds in a working hand.
func wpShieldOf(a *Actor) *Item {
	for _, p := range a.Body {
		if p.Slot == "hand" && !p.Severed && p.Item != nil && p.Item.Kind == "armor" && p.Item.Extra != nil && p.Item.Extra["block"] > 0 {
			return p.Item
		}
	}
	return nil
}

// wpHitMod: flails swing past a shield's DV (hook in swing's to-hit).
func wpHitMod(a, d *Actor, w *Item) int {
	if w == nil || w.Skill != "flail" {
		return 0
	}
	if s := wpShieldOf(d); s != nil && s.DV > 0 {
		return s.DV
	}
	return 0
}

// wpBlock: a held shield may stop a blow (or a missile, at half chance unless it is a full shield).
func (g *Game) wpBlock(a, d *Actor, w *Item, missile bool) bool {
	s := wpShieldOf(d)
	if s == nil || wpIgnoreShield || d.St("stunned") > 0 || d.St("prone") > 0 || d.St("frozen") > 0 || d.St("asleep") > 0 {
		return false
	}
	ch := s.Extra["block"] + 5*g.skillLv(d, "shield")
	if missile && s.Extra["full"] == 0 {
		ch /= 2
	}
	if w != nil && w.Skill == "flail" {
		ch /= 2
	}
	if g.R.Int(1, 100) > ch {
		return false
	}
	if a == g.P || d == g.P || g.visible(d.X, d.Y) {
		g.Msg(fmt.Sprintf("%s %s the blow with %s.", g.The(d), verb(d == g.P, "block", "blocks"), s.Name))
	}
	return true
}

// wpOffChance: fist weapons strike from the off hand far more often (hook in melee).
func wpOffChance(w *Item, ch float64) float64 {
	if w != nil && w.Skill == "fist" && ch < 0.7 {
		return 0.7
	}
	return ch
}

// wpBeforeMelee: a braced spear meets the attacker first (hook at the top of melee).
func (g *Game) wpBeforeMelee(a, d *Actor) {
	if d.St("braced") <= 0 || a == d || a.HP <= 0 {
		return
	}
	w := wpReachOf(d)
	if w == nil {
		w = wpWielded(d, "spear")
	}
	delete(d.Status, "braced")
	if w == nil {
		return
	}
	if a == g.P || d == g.P || g.visible(d.X, d.Y) {
		g.Msg(fmt.Sprintf("{{W|%s %s the charge with %s!}}", g.The(d), verb(d == g.P, "meet", "meets"), w.Name))
	}
	before := a.HP
	g.swing(d, a, w)
	if a.HP > 0 && a.HP < before {
		g.hurt(a, g.R.Int(1, 4)+d.Level/4, d, w.Name)
	}
}

// wpOnHit: family effects after a melee hit lands (hook at the end of onHit).
func (g *Game) wpOnHit(a, d *Actor, w *Item, crit bool, sk int) {
	if d.HP <= 0 || w == nil {
		return
	}
	seen := a == g.P || d == g.P || g.visible(d.X, d.Y)
	say := func(s string) {
		if seen {
			g.Msg(s)
		}
	}
	switch w.Skill {
	case "spear":
		if crit && !d.Has("undead") {
			d.AddSt("bleeding", 4)
		}
	case "flail":
		if g.R.Chance(0.12 + 0.04*float64(sk)) {
			d.AddSt("entangled", 3)
			say(g.The(d) + verb(d == g.P, " are", " is") + " {{y|entangled}} by the chain.")
		}
	case "whip":
		if g.R.Chance(0.10 + 0.05*float64(sk)) {
			g.wpDisarm(d)
		}
	case "staff":
		if g.R.Chance(0.2 + 0.05*float64(sk)) {
			g.wpKnock(a, d, 1)
		}
	case "fist":
		if crit {
			d.AddSt("dazed", 2)
		}
	case "hook":
		if wpHasLegs(d) && g.R.Chance(0.15+0.05*float64(sk)) {
			d.AddSt("prone", 2)
			say(g.The(d) + verb(d == g.P, " are", " is") + " hooked off " + verb(d == g.P, "your", "its") + " feet!")
		}
	case "heavy":
		if !wpCleaving && w.Two && g.R.Chance(0.3+0.1*float64(sk)) {
			for _, o := range g.Cur.Actors {
				if o != d && o != a && o.HP > 0 && dist(o, a) == 1 && (a == g.P && g.Hostile(o) || a != g.P && o == g.P && g.Hostile(a)) {
					wpCleaving = true
					say(fmt.Sprintf("%s cleaves on into %s!", titleCase(w.Name), g.the(o)))
					g.swing(a, o, w)
					wpCleaving = false
					break
				}
			}
			if a != g.P && d != g.P && dist(g.P, a) == 1 && g.Hostile(a) && !wpCleaving {
				wpCleaving = true
				g.swing(a, g.P, w)
				wpCleaving = false
			}
		}
	}
}

func wpHasLegs(a *Actor) bool {
	for _, p := range a.Body {
		if p.Slot == "feet" && !p.Severed {
			return true
		}
	}
	return false
}

// wpDisarm knocks the weapon out of a creature's hand onto the floor.
func (g *Game) wpDisarm(d *Actor) bool {
	for _, p := range d.Body {
		if p.Slot == "hand" && !p.Severed && p.Item != nil && (p.Item.Kind == "weapon" || p.Item.Kind == "ranged") {
			it := p.Item
			p.Item = nil
			it.X, it.Y = d.X, d.Y
			g.Cur.Items = append(g.Cur.Items, it)
			g.Msg(fmt.Sprintf("{{y|%s %s disarmed: %s falls to the ground.}}", g.The(d), verb(d == g.P, "are", "is"), it.Name))
			return true
		}
	}
	return false
}

// wpKnock pushes d n tiles away from a.
func (g *Game) wpKnock(a, d *Actor, n int) {
	dx, dy := skSgn(d.X-a.X), skSgn(d.Y-a.Y)
	moved := false
	for i := 0; i < n; i++ {
		nx, ny := d.X+dx, d.Y+dy
		if !g.Cur.Walkable(nx, ny) || g.Cur.ActorAt(nx, ny) != nil || nx == g.P.X && ny == g.P.Y {
			break
		}
		d.X, d.Y = nx, ny
		moved = true
	}
	if moved && (a == g.P || d == g.P || g.visible(d.X, d.Y)) {
		g.Msg(g.The(d) + verb(d == g.P, " are", " is") + " driven back.")
	}
}

// wpReachOf is a wielded weapon that strikes two tiles away.
func wpReachOf(a *Actor) *Item {
	for _, w := range a.Weapons() {
		if w.Extra != nil && w.Extra["reach"] >= 2 {
			return w
		}
	}
	return nil
}

// wpWielded is a wielded weapon of a skill (nil if none).
func wpWielded(a *Actor, skill string) *Item {
	for _, w := range a.Weapons() {
		if w.Skill == skill {
			return w
		}
	}
	return nil
}

// wpReachTarget checks a reach strike at (tx,ty): a creature within reach and a clear line.
func (g *Game) wpReachTarget(w *Item, tx, ty int) *Actor {
	d := g.Cur.ActorAt(tx, ty)
	if d == nil || d == g.P || d.HP <= 0 {
		g.Msg("There is nothing there to strike.")
		return nil
	}
	if maxi(abs(tx-g.P.X), abs(ty-g.P.Y)) > w.Extra["reach"] {
		g.Msg("{{K|Out of reach.}}")
		return nil
	}
	for _, pt := range line(g.P.X, g.P.Y, tx, ty) {
		if pt[0] == tx && pt[1] == ty {
			break
		}
		if g.Cur.Opaque(pt[0], pt[1]) {
			g.Msg("{{K|Something is in the way.}}")
			return nil
		}
	}
	return d
}

// wpReachAttack: f with a spear or whip strikes a foe two tiles away.
func (g *Game) wpReachAttack(tx, ty int) {
	w := wpReachOf(g.P)
	if w == nil {
		return
	}
	d := g.wpReachTarget(w, tx, ty)
	if d == nil {
		return
	}
	g.swing(g.P, d, w)
	g.spend(1000)
}

// ---- targeting (hook in finishTarget) -------------------------------------

// wpTarget handles the targeting modes of this file; false lets finishTarget carry on.
func (g *Game) wpTarget(pend string, tx, ty int) bool {
	switch {
	case pend == "reach":
		g.wpReachAttack(tx, ty)
		return true
	case pend == "fire":
		m := g.P.Missile()
		if m == nil || m.Effect == "" {
			return false
		}
		g.wpFire(m, tx, ty)
		return true
	case strings.HasPrefix(pend, "wpthrow:"):
		var id int
		fmt.Sscanf(pend[8:], "%d", &id)
		if it := g.wpFindThrown(id); it != nil {
			g.wpThrow(it, tx, ty)
		}
		return true
	}
	return false
}

// wpFireKey: f without a missile weapon strikes at reach.
func (g *Game) wpFireKey() bool {
	if g.P.Missile() != nil || wpReachOf(g.P) == nil {
		return false
	}
	g.beginTarget("reach")
	return true
}

// ---- firearms and tagged ammo --------------------------------------------

// wpUseAmmo spends one round for a ranged weapon (tagged weapons need their own ammo).
func (g *Game) wpUseAmmo(m *Item) bool {
	if m == nil || m.Effect == "" {
		return g.useAmmo()
	}
	if m.Extra != nil && m.Extra["mag"] > 0 {
		if m.Extra["load"] <= 0 {
			return false
		}
		m.Extra["load"]--
		return true
	}
	for _, it := range g.P.Inv {
		if it.Kind == "ammo" && it.Effect == m.Effect && it.Count > 0 {
			it.Count--
			if it.Count == 0 {
				g.removeInv(it)
			}
			return true
		}
	}
	return false
}

// wpReload fills a magazine from the pack; returns false if there was nothing to load.
func (g *Game) wpReload(m *Item) bool {
	need := m.Extra["mag"] - m.Extra["load"]
	if need <= 0 {
		g.Msg(fmt.Sprintf("%s is fully loaded.", titleCase(m.Name)))
		return false
	}
	n := 0
	for _, it := range append([]*Item{}, g.P.Inv...) {
		if n >= need {
			break
		}
		if it.Kind == "ammo" && it.Effect == m.Effect && it.Count > 0 {
			k := mini(need-n, it.Count)
			it.Count -= k
			n += k
			if it.Count == 0 {
				g.removeInv(it)
			}
		}
	}
	if n == 0 {
		g.Msg("You have no " + g.wpAmmoName(m.Effect) + " to load.")
		return false
	}
	m.Extra["load"] += n
	cost := maxi(500, m.Extra["rl"])
	if g.skHas("fa_reload") {
		cost /= 2
	}
	g.Msg(fmt.Sprintf("You reload %s (%d/%d).", m.Name, m.Extra["load"], m.Extra["mag"]))
	g.spend(cost)
	return true
}

// wpFire shoots a tagged ranged weapon: magazines reload when empty, guns may jam, bursts fire several rounds.
func (g *Game) wpFire(m *Item, tx, ty int) {
	if m.Extra == nil {
		m.Extra = map[string]int{}
	}
	shots := 1
	if m.Extra["mag"] > 0 {
		if m.Extra["load"] <= 0 {
			g.wpReload(m)
			return
		}
		if j := m.Extra["jam"] - 3*g.Skills["firearm"]; j > 0 && g.R.Int(1, 100) <= j {
			m.Extra["load"]--
			g.Msg("{{r|Click.}} " + titleCase(m.Name) + " misfires.")
			g.spend(1000)
			return
		}
		shots = clampi(m.Extra["burst"], 1, m.Extra["load"])
	} else if !g.wpUseAmmo(m) {
		g.Msg("You're out of " + g.wpAmmoName(m.Effect) + ".")
		return
	}
	for i := 0; i < shots; i++ {
		if m.Extra["mag"] > 0 {
			m.Extra["load"]--
		}
		g.wpShot(g.P, tx, ty, m)
	}
	delete(g.P.Status, "aiming")
	g.spend(1000)
}

// wpShot fires one round through shoot and applies the weapon's element to whoever it hurt.
func (g *Game) wpShot(a *Actor, tx, ty int, m *Item) {
	hp := map[*Actor]int{}
	for _, o := range g.Cur.Actors {
		hp[o] = o.HP
	}
	g.shoot(a, tx, ty, m)
	if m.Elem == "" {
		return
	}
	for _, o := range g.Cur.Actors {
		if o.HP > 0 && o.HP < hp[o] {
			g.onHit(a, o, m, false, 0)
		}
	}
}

// wpShotSkill is the skill a missile weapon trains (hook in shoot).
func wpShotSkill(w *Item) string {
	if w != nil && (w.Skill == "firearm" || w.Skill == "thrown") {
		return w.Skill
	}
	return "missile"
}

// wpShotHit: smart guns and Steady Aim (hook in shoot's to-hit).
func wpShotHit(a *Actor, w *Item) int {
	n := 0
	if w != nil && w.Extra != nil && w.Extra["smart"] > 0 {
		n += 3
	}
	if a.St("aiming") > 0 {
		n += 4
	}
	return n
}

// wpShotPV: aiming adds penetration; ballistic armour takes it away from bullets and bolts (hook in shoot).
func wpShotPV(a, d *Actor, w *Item) int {
	n := 0
	if a.St("aiming") > 0 {
		n++
	}
	if w != nil && (w.Skill == "firearm" || w.Effect == "bolt") {
		for _, p := range d.Body {
			if p.Item != nil && !p.Severed && p.Item.Extra != nil {
				n -= p.Item.Extra["bal"]
			}
		}
	}
	return n
}

// ---- throwing --------------------------------------------------------------

func wpIsThrown(it *Item) bool { return it != nil && it.Kind == "weapon" && it.Skill == "thrown" }

// wpThrowPick is the first thrown weapon in the pack, else one in hand (for the t key).
func (g *Game) wpThrowPick() *Item {
	for _, it := range g.P.Inv {
		if wpIsThrown(it) {
			return it
		}
	}
	for _, p := range g.P.Body {
		if !p.Severed && wpIsThrown(p.Item) {
			return p.Item
		}
	}
	return nil
}

func (g *Game) wpFindThrown(id int) *Item {
	for _, it := range g.P.Inv {
		if it.ID == id {
			return it
		}
	}
	for _, p := range g.P.Body {
		if p.Item != nil && p.Item.ID == id {
			return p.Item
		}
	}
	return nil
}

// wpThrow throws a thrown weapon (hook at the top of throw); false for anything else.
func (g *Game) wpThrow(it *Item, tx, ty int) bool {
	if !wpIsThrown(it) {
		return false
	}
	g.wpThrowAt(it, tx, ty, 0)
	g.spend(1000)
	return true
}

// wpThrowAt resolves the throw without spending time: to-hit, shield, penetration, damage, return or landing.
func (g *Game) wpThrowAt(it *Item, tx, ty, bonus int) {
	p := g.P
	rng := maxi(it.Range, 4) + p.M(STR)/2 + g.Skills["thrown"]
	lx, ly := p.X, p.Y
	var d *Actor
	for i, pt := range line(p.X, p.Y, tx, ty) {
		if i >= rng || g.Cur.Opaque(pt[0], pt[1]) {
			break
		}
		lx, ly = pt[0], pt[1]
		if a := g.Cur.ActorAt(pt[0], pt[1]); a != nil && a.HP > 0 {
			d = a
			break
		}
	}
	// take one off the stack (or out of the hand)
	one := it
	if it.Count > 1 {
		it.Count--
		cp := *it
		cp.Count = 1
		cp.ID = g.newID()
		if it.Extra != nil {
			cp.Extra = map[string]int{}
			for k, v := range it.Extra {
				cp.Extra[k] = v
			}
		}
		one = &cp
	} else if bp := p.Equipped(it); bp != nil {
		bp.Item = nil
	} else {
		g.removeInv(it)
	}
	if one.Count > 1 {
		one.Count = 1
	}
	if d != nil {
		roll := g.R.Int(1, 20)
		hit := roll + p.M(AGI) + g.skillLv(p, "thrown") + bonus + skStatusToHit(p) - g.weatherMissile(p)
		if roll == 20 || roll != 1 && hit >= d.DV() {
			if !g.wpBlock(p, d, one, true) {
				pens := g.penetrate(one.PV+mini(p.M(STR), one.MaxStr), d.AV(), roll == 20)
				if pens == 0 {
					g.Msg(fmt.Sprintf("The %s glances off %s.", one.Name, g.the(d)))
				} else {
					dmg := 0
					for k := 0; k < pens; k++ {
						dmg += g.R.Dice(maxi(one.DN, 1), maxi(one.DS, 1)) + one.DB
					}
					if bonus > 0 {
						dmg += 3
					}
					g.Msg(fmt.Sprintf("You hit %s with the %s (x%d) for {{R|%d}} damage.", g.the(d), one.Name, pens, dmg))
					g.hurt(d, dmg, p, one.Name)
					if d.HP > 0 {
						g.onHit(p, d, one, roll == 20, g.Skills["thrown"])
						if one.Extra["ent"] > 0 && wpHasLegs(d) {
							d.AddSt("entangled", 4)
							g.Msg(g.The(d) + " is {{y|entangled}}!")
						}
						if one.Extra["ret"] >= 100 && g.R.Chance(0.25) {
							d.AddSt("dazed", 2)
						}
					}
				}
			}
		} else {
			g.Msg(fmt.Sprintf("The %s misses %s.", one.Name, g.the(d)))
		}
	} else {
		g.Msg("You throw the " + one.Name + ".")
	}
	ret := one.Extra["ret"]
	if ret > 0 && g.skHas("tw_catch") {
		ret = 100
	}
	if ret > 0 && g.R.Int(1, 100) <= ret {
		g.Msg(fmt.Sprintf("{{c|The %s returns to your hand.}}", one.Name))
		if it != one && it.Count > 0 && g.wpFindThrown(it.ID) == it {
			it.Count++
			return
		}
		if bp := p.PartFor("hand"); bp != nil && bp.Item == nil {
			bp.Item = one
			return
		}
		g.give(one)
		return
	}
	one.X, one.Y = lx, ly
	g.Cur.Items = append(g.Cur.Items, one)
}

// ---- maneuvers (hook in skRun) ---------------------------------------------

// wpRun runs this file's maneuvers; done is false for ids it does not know.
func (g *Game) wpRun(n string, tx, ty int) (ok, done bool) {
	p := g.P
	lv := p.Level
	need := func(skill, what string) *Item {
		w := wpWielded(p, skill)
		if w == nil {
			g.Msg("{{K|You need " + what + " in hand.}}")
		}
		return w
	}
	swingWith := func(d *Actor, w *Item, extra int) bool {
		before := d.HP
		g.swing(p, d, w)
		hit := d.HP < before
		if hit && d.HP > 0 && extra > 0 {
			g.hurt(d, extra, p, w.Name)
		}
		return hit
	}
	switch n {
	case "brace":
		if wpReachOf(p) == nil && wpWielded(p, "spear") == nil {
			g.Msg("{{K|You need a spear or polearm in hand.}}")
			return false, true
		}
		p.AddSt("braced", 8)
		g.Msg("{{c|You set your spear and wait for the charge.}}")
	case "impale":
		w := need("spear", "a spear")
		if w == nil {
			return false, true
		}
		d := g.wpReachTarget(w, tx, ty)
		if d == nil {
			return false, true
		}
		if swingWith(d, w, 2+lv/4) && d.HP > 0 {
			d.AddSt("hobbled", 6)
			d.AddSt("bleeding", 4)
			g.Msg(g.The(d) + " is {{r|impaled}}.")
		}
	case "smash":
		w := need("flail", "a flail")
		if w == nil {
			return false, true
		}
		d := g.skFoeAt(tx, ty, 1)
		if d == nil {
			return false, true
		}
		g.Msg("{{W|You bring the flail down overhead!}}")
		wpIgnoreShield = true
		hit := swingWith(d, w, 3+lv/3)
		wpIgnoreShield = false
		if hit && d.HP > 0 && g.R.Chance(0.5) {
			d.AddSt("stunned", 2)
			g.Msg(g.The(d) + " is stunned!")
		}
	case "lash":
		w := need("whip", "a whip")
		if w == nil {
			return false, true
		}
		d := g.wpReachTarget(w, tx, ty)
		if d == nil {
			return false, true
		}
		if swingWith(d, w, 0) && d.HP > 0 {
			if !(g.R.Chance(0.5) && g.wpDisarm(d)) && wpHasLegs(d) {
				d.AddSt("prone", 2)
				g.Msg(g.The(d) + " is pulled off its feet!")
			}
		}
	case "stthrust":
		w := need("staff", "a staff")
		if w == nil {
			return false, true
		}
		d := g.skFoeAt(tx, ty, 1)
		if d == nil {
			return false, true
		}
		if swingWith(d, w, 0) && d.HP > 0 {
			g.wpKnock(p, d, 3)
			d.AddSt("dazed", 2)
		}
	case "combo":
		w := need("fist", "a fist weapon")
		if w == nil {
			return false, true
		}
		d := g.skFoeAt(tx, ty, 1)
		if d == nil {
			return false, true
		}
		for i := 0; i < 3 && d.HP > 0; i++ {
			swingWith(d, w, 0)
		}
	case "hooktrip":
		w := need("hook", "a hooked blade")
		if w == nil {
			return false, true
		}
		d := g.skFoeAt(tx, ty, 1)
		if d == nil {
			return false, true
		}
		if swingWith(d, w, 0) && d.HP > 0 && wpHasLegs(d) {
			d.AddSt("prone", 3)
			g.Msg(g.The(d) + " goes down!")
		}
	case "deadeye":
		it := g.wpThrowPick()
		if it == nil {
			g.Msg("{{K|You have nothing to throw.}}")
			return false, true
		}
		if tx == p.X && ty == p.Y {
			return false, true
		}
		g.wpThrowAt(it, tx, ty, 4)
	case "steadyaim":
		if m := p.Missile(); m == nil {
			g.Msg("{{K|You need a missile weapon.}}")
			return false, true
		}
		p.AddSt("aiming", 6)
		g.Msg("{{c|You steady your breath and take aim.}}")
	default:
		return false, false
	}
	return true, true
}

// ---- crafting --------------------------------------------------------------

// wpRecipes: ammo for the new families (basic) and every sphere weapon (learned by taking one apart).
func (g *Game) wpRecipes() []*cRecipe {
	r := g.R
	var rs []*cRecipe
	tags := []string{"arrow", "bolt", "stone", "dart"}
	if wpGunGenres[g.Genre] {
		tags = append(tags, wpGunOf(g.Genre).Tag)
	}
	for _, tag := range tags {
		tag := tag
		rs = append(rs, &cRecipe{ID: "b:wp:" + tag, Name: g.wpAmmoName(tag) + " x10", Kind: "tool", Rank: 1, Basic: true, Cost: map[int]int{1: 2},
			Make: func() *Item { return g.MakeWpAmmo(tag, 10) }})
	}
	for _, sp := range attireSpheresSorted() {
		k := wpSpheres[sp]
		if k == nil {
			continue
		}
		for i, e := range k.Weapons {
			sp, i, e := sp, i, e
			kind := "weapon"
			if f := e.Fam; f == "sling" || f == "blowgun" || f == "crossbow" || f == "longbow" || f == "atlatl" {
				kind = "ranged"
			}
			rs = append(rs, &cRecipe{ID: fmt.Sprintf("w:%s:%d", sp, i), Name: e.Name, Kind: kind, Rank: 2, Sphere: sp, Cost: map[int]int{2: 4, 3: 2},
				Make: func() *Item {
					it := g.MakeSphereWeapon(sp, i, g.sigTier(), r)
					it.Rarity, it.F = "uncommon", RarityColor("uncommon")
					return it
				}})
		}
	}
	return rs
}

func attireSpheresSorted() []string {
	sp := attireSpheres()
	for i := 1; i < len(sp); i++ {
		for j := i; j > 0 && sp[j] < sp[j-1]; j-- {
			sp[j], sp[j-1] = sp[j-1], sp[j]
		}
	}
	return sp
}

// ---- describe --------------------------------------------------------------

// wpDescribe adds family lines to Item.Describe.
func (g *Game) wpDescribe(it *Item) []string {
	var l []string
	if it.Extra == nil && !it.Two {
		return nil
	}
	if it.Kind == "weapon" && it.Two {
		l = append(l, "Two-handed.")
	}
	if it.Extra["reach"] >= 2 {
		l = append(l, "Reach: strikes two tiles away (f).")
	}
	switch it.Skill {
	case "flail":
		l = append(l, "Swings past shields; may entangle.")
	case "whip":
		l = append(l, "May disarm.")
	case "staff":
		l = append(l, "Knocks foes back.")
	case "fist":
		l = append(l, "Fast: off-hand strikes land often.")
	case "hook":
		l = append(l, "Hooks legs: may trip.")
	case "heavy":
		if it.Two {
			l = append(l, "Cleaves into a second foe.")
		}
	case "thrown":
		s := fmt.Sprintf("Throw it (t), range %d.", it.Range)
		if r := it.Extra["ret"]; r >= 100 {
			s += " It comes back."
		} else if r > 0 {
			s += fmt.Sprintf(" Comes back %d%% of the time.", r)
		}
		if it.Extra["ent"] > 0 {
			s += " Entangles."
		}
		l = append(l, s)
	}
	if it.Kind == "ranged" && it.Extra["mag"] > 0 {
		l = append(l, fmt.Sprintf("Loaded %d/%d. Fire (f) when empty to reload (%d turns).", it.Extra["load"], it.Extra["mag"], maxi(1, it.Extra["rl"]/1000)))
		if it.Extra["burst"] > 1 {
			l = append(l, fmt.Sprintf("Fires bursts of %d.", it.Extra["burst"]))
		}
		if it.Extra["jam"] > 0 {
			l = append(l, fmt.Sprintf("Misfires %d%% of the time.", it.Extra["jam"]))
		}
		if it.Extra["smart"] > 0 {
			l = append(l, "Smart: +3 to hit.")
		}
	}
	if b := it.Extra["block"]; b > 0 {
		s := fmt.Sprintf("Shield: %d%% to block blows", b)
		if it.Extra["full"] > 0 {
			s += " and missiles"
		} else {
			s += ", half against missiles"
		}
		l = append(l, s+".")
	}
	if b := it.Extra["bal"]; b > 0 {
		l = append(l, fmt.Sprintf("Ballistic: -%d penetration for bullets and bolts.", b))
	}
	return l
}
