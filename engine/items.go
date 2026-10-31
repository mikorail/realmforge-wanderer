package engine

import (
	"fmt"
	"strings"
)

var tierColor = [6]byte{CBrown, CGray, CWhite, CCyan, CMagenta, CYellow}

func (g *Game) skin() *Skin { return g.Sk }

// MakeWeapon builds a melee weapon of a skill family and tier (0-5).
func (g *Game) MakeWeapon(skill string, tier int, r *Rng) *Item {
	s := g.skin()
	tier = clampi(tier, 0, 5)
	noun := Pick(r, s.Nouns[skill])
	it := &Item{ID: g.newID(), Kind: "weapon", Slot: "hand", Skill: skill, Tier: tier, F: tierColor[tier], Weight: 4}
	it.Name = s.Mats[tier] + " " + noun
	switch skill {
	case "longblade":
		it.Glyph, it.DN, it.DS, it.PV, it.MaxStr = '/', 1, 6+tier, 2+tier/2, 3+tier
	case "shortblade":
		it.Glyph, it.DN, it.DS, it.PV, it.MaxStr = '-', 1, 4+tier/2, 3+tier/2, 2+tier/2
		it.Weight = 1
	case "axe":
		it.Glyph, it.DN, it.DS, it.PV, it.MaxStr = '\\', 1, 7+tier, 1+tier/2, 3+tier
		it.Weight = 6
	case "cudgel":
		it.Glyph, it.DN, it.DS, it.PV, it.MaxStr = '|', 2, 3+tier/2, 1+tier/3, 4+tier
		it.Weight = 7
	}
	if tier >= 3 && r.Chance(0.15) {
		e := Pick(r, []string{"fire", "frost", "shock"})
		it.Elem = e
		it.Name = map[string]string{"fire": "flaming ", "frost": "frostbitten ", "shock": "humming "}[e] + it.Name
	}
	it.Value = 8 * (tier + 1) * (tier + 1)
	return it
}

func (g *Game) MakeRanged(tier int, r *Rng) *Item {
	s := g.skin()
	tier = clampi(tier, 0, 5)
	noun := s.Ranged[clampi(tier/2, 0, len(s.Ranged)-1)]
	return &Item{ID: g.newID(), Kind: "ranged", Slot: "hand", Skill: "missile", Glyph: '}', F: tierColor[tier], Tier: tier, Two: true,
		Name: s.Mats[tier] + " " + noun, DN: 1, DS: 6 + tier, PV: 3 + tier/2, Range: 10 + tier, Value: 10 * (tier + 1) * (tier + 1), Weight: 4}
}

func (g *Game) MakeAmmo(n int) *Item {
	return &Item{ID: g.newID(), Kind: "ammo", Name: g.skin().Ammo, Glyph: '¡', F: CGray, Count: n, Value: 1}
}

func (g *Game) MakeArmor(slot string, tier int, r *Rng) *Item {
	s := g.skin()
	tier = clampi(tier, 0, 5)
	nouns := s.Armor[slot]
	noun := nouns[clampi(tier*len(nouns)/6, 0, len(nouns)-1)]
	it := &Item{ID: g.newID(), Kind: "armor", Slot: slot, Tier: tier, F: tierColor[tier], Weight: 3}
	it.Name = s.Mats[tier] + "-trimmed " + noun
	if tier <= 1 {
		it.Name = noun
	}
	switch slot {
	case "body":
		it.Glyph, it.AV, it.DV = '[', 2+tier, -tier/3
		it.Weight = 8
	case "head":
		it.Glyph, it.AV = '^', 1+tier/2
	case "feet":
		it.Glyph, it.AV, it.DV = '%', tier/2, 1
	case "hands":
		it.Glyph, it.AV = '%', tier/3
		if tier >= 2 {
			it.Bonus = map[int]int{STR: 1}
		}
	case "back":
		it.Glyph, it.AV, it.DV = '(', tier/3, 1+tier/3
	case "face":
		it.Glyph, it.AV = '&', tier/3
		it.Bonus = map[int]int{WIL: 1 + tier/3}
	}
	it.Value = 6 * (tier + 1) * (tier + 1)
	return it
}

func (g *Game) MakeLight(tier int) *Item {
	s := g.skin()
	tier = clampi(tier, 0, 2)
	return &Item{ID: g.newID(), Kind: "light", Slot: "float", Name: s.Light[tier], Glyph: '¥', F: CYellow, Light: 4 + tier*2, Value: 10 * (tier + 1), Weight: 1,
		Extra: map[string]int{"lit": 1}}
}

func (g *Game) MakeTonic(kind string, n int) *Item {
	s := g.skin()
	name := map[string]string{"heal": s.Heal, "salve": s.Salve, "haste": s.Haste, "strength": s.Strength, "antidote": s.Antidote}[kind]
	col := map[string]byte{"heal": CRed, "salve": CWhite, "haste": CCyan, "strength": COrange, "antidote": CGreen}[kind]
	return &Item{ID: g.newID(), Kind: "tonic", Effect: kind, Name: name, Glyph: '!', F: col, Count: n, Value: 15}
}

func (g *Game) MakeGrenade(kind string, n int) *Item {
	s := g.skin()
	name := map[string]string{"fire": s.GrenFire, "frost": s.GrenFrost, "flash": s.GrenFlash}[kind]
	col := map[string]byte{"fire": CRed, "frost": CCyan, "flash": CWhite}[kind]
	return &Item{ID: g.newID(), Kind: "grenade", Effect: kind, Name: name, Glyph: 'ö', F: col, Count: n, Value: 20}
}

func (g *Game) MakeFood(r *Rng, n int) *Item {
	return &Item{ID: g.newID(), Kind: "food", Name: Pick(r, g.skin().Food), Glyph: '%', F: CBrown, Count: n, Value: 3, Power: 400}
}

func (g *Game) MakeWater() *Item {
	return &Item{ID: g.newID(), Kind: "water", Name: g.skin().Water, Glyph: 'δ', F: CBlue, Charges: 8, Power: 16, Value: 8}
}

func (g *Game) MakeScrap(r *Rng, n int) *Item {
	return &Item{ID: g.newID(), Kind: "scrap", Name: g.skin().Scrap, Glyph: '*', F: CDarkCyan, Count: n, Value: 2}
}

func (g *Game) MakeBook(title, text string) *Item {
	return &Item{ID: g.newID(), Kind: "book", Name: title, Glyph: '■', F: CCyan, Text: text, Value: 12}
}

// Loot rolls one random item of a rough tier.
func (g *Game) Loot(r *Rng, tier int) *Item {
	tier = clampi(tier, 0, 5)
	gear := func(it *Item) *Item {
		g.applyRarity(it, r, g.rollRarity(r, tier*3+r.Int(0, 3)))
		return it
	}
	switch x := r.Int(0, 99); {
	case x < 20:
		return gear(g.MakeWeapon(Pick(r, []string{"longblade", "shortblade", "axe", "cudgel"}), tier, r))
	case x < 33:
		return gear(g.MakeArmor(Pick(r, []string{"body", "head", "feet", "hands", "back", "face"}), tier, r))
	case x < 35:
		return gear(g.MakeBackpack(clampi(tier, 0, 5)))
	case x < 42:
		return gear(g.MakeRanged(tier, r))
	case x < 50:
		return g.MakeAmmo(r.Int(8, 20))
	case x < 66:
		return g.MakeTonic(Pick(r, []string{"heal", "heal", "salve", "haste", "strength", "antidote"}), 1)
	case x < 74:
		return g.MakeGrenade(Pick(r, []string{"fire", "frost", "flash"}), r.Int(1, 2))
	case x < 84:
		return g.MakeFood(r, r.Int(1, 3))
	case x < 88:
		return gear(g.MakeLight(r.Int(0, 2)))
	case x < 93:
		return gear(g.MakeOddity(r, tier))
	}
	return g.MakeScrap(r, r.Int(2, 8))
}

var oddUnid = []string{"humming disc", "warm sphere", "glass rod", "folded metal", "pulsing seed", "carved idol", "rune-cube", "black prism"}

// MakeOddity is a Qud-style unidentified artifact: you learn what it is by examining it.
func (g *Game) MakeOddity(r *Rng, tier int) *Item {
	kinds := []struct{ name, effect string }{
		{"rod of displacement", "teleport"}, {"heart-stone", "heal"}, {"storm prism", "lightning"}, {"ember orb", "flames"},
		{"rime lens", "freeze"}, {"phase shard", "phase"}, {"seer's eye", "sense"}, {"bulwark charm", "forcewall"},
	}
	k := Pick(r, kinds)
	it := &Item{ID: g.newID(), Kind: "artifact", Slot: "float", Glyph: '÷', F: CMagenta, Name: k.name, Effect: k.effect, Charges: r.Int(3, 8),
		Unid: Pick(r, oddUnid), Tier: tier, Value: 60 + tier*30, Power: 3 + tier, Lore: "Its makers are long forgotten."}
	return it
}

// RelicItem turns a Realmforge relic (forged in simulated history) into a unique artifact.
func (g *Game) RelicItem(a *ArtifactData) *Item {
	stat := map[string]int{"martial": STR, "diplomacy": EGO, "stewardship": TOU, "intrigue": AGI, "stability": WIL}[a.Stat]
	bonus := clampi(a.Val/2+1, 1, 6)
	it := &Item{ID: g.newID(), Name: a.Name, Kind: "artifact", Glyph: '☼', F: CYellow, Tier: 5, Lore: a.Lore, SimArt: a.ID,
		Bonus: map[int]int{stat: bonus}, Value: 300 + a.Fame*10, Unid: "ancient " + strings.ToLower(a.Kind), Weight: 2}
	switch a.Kind {
	case "weapon":
		it.Slot, it.Skill, it.Glyph, it.DN, it.DS, it.PV, it.MaxStr = "hand", "longblade", '/', 2, 6, 5, 8
	case "crown":
		it.Slot, it.Glyph, it.AV = "head", '^', 2
	case "armor":
		it.Slot, it.Glyph, it.AV, it.DV = "body", '[', 7, 1
	case "tome":
		it.Slot, it.Glyph = "float", '■'
		it.Bonus[INT] += 2
	default:
		it.Slot = "float"
	}
	it.Text = fmt.Sprintf("Forged in the year %d. %s", a.Year, a.Origin)
	return it
}

func (it *Item) Describe(g *Game) []string {
	var l []string
	if it.Kind == "artifact" && !it.Ident {
		return []string{"An object of unknown make: {{M|" + it.Unid + "}}.", "Examine it closely to learn more (Tinkering or Lore help)."}
	}
	l = append(l, "{{Y|"+it.ShownName()+"}}", it.RarityTagG(g))
	switch it.Kind {
	case "weapon":
		l = append(l, fmt.Sprintf("%s  {{c|→}}%d  {{R|♥}}%dd%d  max STR bonus %d", skillName(it.Skill), it.PV, it.DN, it.DS, it.MaxStr))
	case "ranged":
		l = append(l, fmt.Sprintf("missile  {{c|→}}%d  {{R|♥}}%dd%d  range %d  uses %s", it.PV, it.DN, it.DS, it.Range, g.Sk.Ammo))
	case "armor":
		l = append(l, fmt.Sprintf("worn on %s  {{C|◆}}%d AV  {{K|○}}%d DV", it.Slot, it.AV, it.DV))
	case "light":
		l = append(l, fmt.Sprintf("light radius %d", it.Light))
	case "tonic":
		l = append(l, "A tonic: "+it.Effect)
	case "food":
		l = append(l, "Food. Eat it when hungry, or cook it (C) for a warmer meal.")
	case "dish":
		l = append(l, "A cooked dish, ready to eat: "+describeTags(tagsOf(it.Effect))+".")
	case "ingredient", "corpse":
		if t, ok := cookTags[it.Effect]; ok {
			l = append(l, fmt.Sprintf("A cooking ingredient {{%s|<%s>}}: in a meal, %s.", t.Col, t.Name, t.Desc(1)))
		}
	case "water":
		l = append(l, fmt.Sprintf("Holds %d/%d drams of fresh water.", it.Charges, it.Power))
	case "grenade":
		l = append(l, "Throw it (t) at a target.")
	case "book":
		l = append(l, "Read it (r).")
	case "artifact":
		if it.DN > 0 {
			l = append(l, fmt.Sprintf("{{c|→}}%d  {{R|♥}}%dd%d", it.PV, it.DN, it.DS))
		}
		if it.AV > 0 {
			l = append(l, fmt.Sprintf("{{C|◆}}%d AV", it.AV))
		}
		if it.Effect != "" {
			l = append(l, fmt.Sprintf("Activate (a) to invoke %s. %d charges.", g.Sk.PowerName(it.Effect), it.Charges))
		}
	}
	if it.Elem != "" {
		l = append(l, "Deals "+it.Elem+" damage.")
	}
	for k, v := range it.Bonus {
		l = append(l, fmt.Sprintf("{{G|+%d %s}}", v, AttrNames[k]))
	}
	if it.Lore != "" {
		l = append(l, "", it.Lore)
	}
	if it.Text != "" && it.Kind != "book" {
		l = append(l, it.Text)
	}
	l = append(l, "", fmt.Sprintf("Value: %d %s", it.Value, g.Sk.Currency))
	return l
}

// MakeManual teaches a skill rank when read (up to rank 2; higher ranks need skill points).
func (g *Game) MakeManual(skill string) *Item {
	return &Item{ID: g.newID(), Kind: "manual", Effect: skill, Name: fmt.Sprintf("%s on %s", titleCase(g.Sk.Book), skillName(skill)), Glyph: '■', F: COrange, Value: 140,
		Text: skillDesc[skill]}
}
