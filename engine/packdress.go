package engine

import (
	"fmt"
	"strings"
)

// Culture pack dress (prefix pd): the pack's Material.Dress (low common high ritual mourning wedding war) worn by
// the people of a town, by rank. Everything is optional: a culture without a pack, or a pack without a Dress map,
// adds nothing, and every function tolerates nil. No state, no g.R: the rank of a person comes from their stratum
// (society.go) and role, the item from a stable hash.
//
//   - pdLook(a)           one line in the look text of a townsperson (hook in render.go, after Desc)
//   - zone hook           an everyday garment named after the pack's dress for the person's rank, in an empty body
//                         slot (AV 0, so combat is untouched); look/inspect shows it, a corpse may drop it
//   - pdCeremony(c)       mourning and wedding dress lines on the ceremony screen (hook in ceremonies.go fwDraw)
//   - pdFeast(b, f, has)  what the town wears to a feast day, unless the feast's own rite names the dress
//   - tailor talk option  "Ask what each rank wears here."

func init() {
	zoneHooks = append(zoneHooks, pdZone)
	talkHooks = append(talkHooks, pdTalk)
}

var pdRankOrder = []string{"low", "common", "high", "ritual", "war", "mourning", "wedding"}

var pdRankPhrase = map[string]string{
	"low":      "Plain, hard-wearing clothes of the poor here",
	"common":   "Everyday dress of the common people here",
	"high":     "Fine dress of the high-born here",
	"ritual":   "Dress for the rites",
	"war":      "Dress for arms",
	"mourning": "Mourning dress",
	"wedding":  "Wedding dress",
}

var pdRoleRank = map[string]string{
	"priest": "ritual", "shaman": "ritual", "monk": "ritual", "oracle": "ritual", "cleric": "ritual", "acolyte": "ritual",
	"guard": "war", "soldier": "war", "captain": "war", "warrior": "war", "knight": "war", "sergeant": "war",
	"beggar": "low", "homeless": "low", "refugee": "low", "unfree": "low", "thief": "low", "pickpocket": "low",
	"notable": "high", "ruler": "high", "noble": "high", "chief": "high", "mayor": "high", "lord": "high", "lady": "high",
}

// pdDress returns the pack's dress text for key, falling back along sensible chains; "" when the pack has none.
func pdDress(p *CulturePack, key string) string {
	if p == nil || p.Material == nil || len(p.Material.Dress) == 0 {
		return ""
	}
	get := func(k string) string {
		for kk, v := range p.Material.Dress {
			if strings.EqualFold(strings.TrimSpace(kk), k) && strings.TrimSpace(v) != "" {
				return strings.TrimSpace(v)
			}
		}
		return ""
	}
	chain := map[string][]string{
		"low": {"low", "common"}, "common": {"common", "low"}, "high": {"high", "ritual", "common"},
		"ritual": {"ritual", "high", "common"}, "war": {"war", "common"}, "mourning": {"mourning", "common"},
		"wedding": {"wedding", "high", "ritual"},
	}[key]
	if chain == nil {
		chain = []string{key, "common"}
	}
	for _, k := range chain {
		if s := get(k); s != "" {
			return s
		}
	}
	return ""
}

// pdClean lowercases the first letter and drops the final full stop, so text drops into a sentence.
func pdClean(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimRight(s, ". ")
	if s == "" {
		return ""
	}
	if len(s) > 1 && s[1] >= 'a' && s[1] <= 'z' || len(s) == 1 {
		s = strings.ToLower(s[:1]) + s[1:]
	}
	return s
}

// pdRank is the dress key for a townsperson: role first (priests, guards, the destitute), then stratum.
func (g *Game) pdRank(a *Actor, bd *BurgData) string {
	if a == nil {
		return "common"
	}
	if k, ok := pdRoleRank[a.Role]; ok {
		return k
	}
	if a.Caste != "" && bd != nil {
		def := g.strataDefFor(bd)
		i := def.index(a.Caste)
		switch {
		case i < 0:
		case i <= maxi(def.slot["ruler"], def.slot["noble"]):
			return "high"
		case i >= def.slot["low"] || def.outcast(a.Caste):
			return "low"
		case i == def.slot["clergy"]:
			return "ritual"
		case i == def.slot["warrior"] && def.slot["warrior"] != def.slot["common"]:
			return "war"
		}
	}
	return "common"
}

// pdTownPack gives the pack and burg of a town zone, or nils.
func (g *Game) pdTownPack(zn *Zone) (*CulturePack, *BurgData) {
	if g == nil || g.W == nil || zn == nil || zn.Kind != "town" || zn.Burg <= 0 {
		return nil, nil
	}
	bd := g.W.burgByID[zn.Burg]
	if bd == nil {
		return nil, nil
	}
	return cgPack(g.W, bd.Culture), bd
}

// pdLook: the dress line for the look text of a townsperson; "" when there is nothing to say.
func (g *Game) pdLook(a *Actor) string {
	if a == nil || g == nil || g.Cur == nil || a.Tpl != "npc" || a.AI == "follower" || g.Hostile(a) {
		return ""
	}
	p, bd := g.pdTownPack(g.Cur)
	if p == nil {
		return ""
	}
	rk := g.pdRank(a, bd)
	txt := pdDress(p, rk)
	if txt == "" {
		return ""
	}
	return "{{K|" + pdRankPhrase[rk] + ": " + pdClean(txt) + ".}}"
}

// pdName shortens a dress text to the name of a garment: the first clause, at most five words, no article.
func pdName(txt string) string {
	s := strings.ToLower(strings.TrimSpace(txt))
	for _, sep := range []string{",", ";", " and ", " with ", " over ", " trimmed ", " edged ", " worn ", " in "} {
		if i := strings.Index(s, sep); i > 0 {
			s = s[:i]
		}
	}
	for _, art := range []string{"a ", "an ", "the ", "their ", "some "} {
		s = strings.TrimPrefix(s, art)
	}
	w := strings.Fields(s)
	if len(w) > 5 {
		w = w[:5]
	}
	return strings.Join(w, " ")
}

func pdZone(g *Game, zn *Zone, _ *Rng) {
	p, bd := g.pdTownPack(zn)
	if p == nil || p.Material == nil || len(p.Material.Dress) == 0 || zn.Z != 0 {
		return
	}
	for _, a := range zn.Actors {
		if a == nil || a.Tpl != "npc" || a.AI == "follower" || a.Role == "guard" {
			continue
		}
		bp := a.PartFor("body")
		if bp == nil || bp.Item != nil || bp.Severed {
			continue
		}
		rk := g.pdRank(a, bd)
		txt := pdDress(p, rk)
		nm := pdName(txt)
		if nm == "" {
			continue
		}
		tier := map[string]int{"low": 0, "common": 1, "high": 4, "ritual": 3, "war": 2}[rk]
		it := &Item{ID: g.newID(), Kind: "armor", Slot: "body", Name: nm, Glyph: '[', F: tierColor[tier], Weight: 2,
			Value: 4 + 8*tier*tier, Ident: true, Tier: tier,
			Lore: fmt.Sprintf("Dress of the %s here: %s.", map[string]string{"low": "poor", "common": "common folk", "high": "high-born", "ritual": "rites", "war": "armed"}[rk], pdClean(txt))}
		bp.Item = it
	}
}

// pdCeremony: dress lines for the ceremony screen (funeral mourning, wedding finery, feast processions).
func (g *Game) pdCeremony(c *fwCer) []string {
	if g == nil || g.W == nil || c == nil {
		return nil
	}
	bd := g.W.burgByID[c.Burg]
	if bd == nil {
		return nil
	}
	p := cgPack(g.W, bd.Culture)
	switch c.Kind {
	case "funeral":
		if t := pdDress(p, "mourning"); t != "" && p.Material.Dress["mourning"] != "" {
			return []string{"The mourners wear " + pdClean(t) + "."}
		}
	case "wedding":
		if p != nil && p.Material != nil && p.Material.Dress["wedding"] != "" {
			t := pdClean(p.Material.Dress["wedding"])
			if h := pdDress(p, "high"); h != "" {
				return []string{"The couple wear " + t + ". The guests come in their best: " + pdClean(h) + "."}
			}
			return []string{"The couple wear " + t + "."}
		}
	case "feast":
		if t := pdDress(p, "ritual"); t != "" {
			return []string{"The pilgrims are in " + pdClean(t) + "."}
		}
	}
	return nil
}

// pdFeast: what the people wear on a feast day; "" when the feast's own rite names the dress or the pack is silent.
func (g *Game) pdFeast(b *BurgData, f *FeastData, riteHasDress bool) string {
	if g == nil || g.W == nil || b == nil || f == nil || riteHasDress {
		return ""
	}
	key := "ritual"
	switch strings.ToLower(f.Kind) {
	case "war", "victory", "founding":
		key = "war"
	case "harvest", "spring", "summer", "market", "fair":
		key = "common"
	case "king", "royal", "crown", "coronation":
		key = "high"
	}
	t := pdDress(cgPack(g.W, b.Culture), key)
	if t == "" {
		return ""
	}
	return "\n\n{{K|The people of " + b.Name + " keep the day in " + pdClean(t) + ".}}"
}

var pdTailorRoles = map[string]bool{"tailor": true, "weaver": true, "dyer": true, "seamstress": true}

// pdTailorNote returns a short local dress note for attire.go's tailorMenu, or "" without a pack.
func (g *Game) pdTailorNote() string {
	_, p := g.cgHere()
	if t := pdDress(p, "common"); t != "" {
		return " Around here folk favour " + pdClean(t) + "."
	}
	return ""
}

func pdTalk(g *Game, a *Actor) []Option {
	if !cgOK(g, a) || !pdTailorRoles[a.Role] {
		return nil
	}
	_, p := g.cgHere()
	if p == nil || p.Material == nil || len(p.Material.Dress) == 0 {
		return nil
	}
	return []Option{{"Ask what each rank wears here.", func() {
		lines := []string{"\"Here is how it is done,\" " + g.the(a) + " says."}
		for _, k := range pdRankOrder {
			if t := strings.TrimSpace(p.Material.Dress[k]); t != "" {
				lines = append(lines, "{{Y|"+titleCase(k)+":}} "+t)
			}
		}
		g.talkMenu(strings.Join(lines, "\n"))
	}}}
}
