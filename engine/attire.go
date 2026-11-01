package engine

import (
	"fmt"
	"strings"
)

// Regional clothing: every culture sphere has fictional garments inspired by real-world dress.
// Each entry is "plain/fine": the first for low tiers, the second for tier 3+.
var attireTable = map[string]map[string]string{
	"sinic": {
		"body": "hemp changshan/silk changshan", "head": "felt scholar's cap/lacquered helm", "back": "padded winter cloak/cloud-pattern cloak",
		"feet": "cloth shoes/embroidered boots", "face": "opera mask/jade face-guard"},
	"japonic": {
		"body": "cotton kimono/silk kimono and haori", "head": "straw sedge hat/kabuto", "back": "hemp haori/crested haori",
		"feet": "straw waraji/lacquered geta", "face": "paper kitsune mask/menpo war-mask"},
	"indic": {
		"body": "cotton dhoti and kurta/brocade sherwani", "head": "wrapped turban/jewelled pagri", "back": "shawl/pashmina shawl",
		"feet": "leather juttis/embroidered mojaris", "face": "silk veil/filigree nose-chain veil"},
	"sea": {
		"body": "batik sarong/gold-thread songket", "head": "woven caping hat/ceremonial headcloth", "back": "ikat shoulder cloth/ikat sash-cloak",
		"feet": "rattan sandals/carved sandals", "face": "carved wooden mask/tiger-spirit mask"},
	"steppe": {
		"body": "wool deel/silk-lined deel", "head": "fur hat/fox-fur malakhai", "back": "felt mantle/wolf-pelt mantle",
		"feet": "felt boots/gutal riding boots", "face": "horsehair veil/wind-scarf mask"},
	"persian": {
		"body": "wool qaba/brocade qaba", "head": "felt kolah/jewelled kolah", "back": "woven cloak/embroidered cloak",
		"feet": "leather slippers/curled slippers", "face": "gauze veil/pearl-fringed veil"},
	"arabic": {
		"body": "cotton thobe/embroidered thobe", "head": "keffiyeh/agal-bound keffiyeh", "back": "woollen bisht/gold-edged bisht",
		"feet": "leather sandals/soft nalayn", "face": "desert veil/silver litham"},
	"mande": {
		"body": "bogolan tunic/indigo boubou", "head": "woven cap/gold-banded cap", "back": "mud-cloth wrap/leopard-hide mantle",
		"feet": "leather sandals/beaded sandals", "face": "kore mask/antelope mask"},
	"yoruba": {
		"body": "aso-oke buba/embroidered agbada", "head": "plain fila/coral-beaded fila", "back": "iro wrap/aso-oke cape",
		"feet": "leather sandals/beaded slippers", "face": "gelede mask/ivory egungun mask"},
	"bantu": {
		"body": "bark-cloth wrap/kente-woven robe", "head": "beaded headband/feathered headdress", "back": "hide cloak/zebra-hide cloak",
		"feet": "hide sandals/beaded sandals", "face": "painted clay mask/ancestor mask"},
	"ethiopic": {
		"body": "cotton shamma/embroidered shamma", "head": "woven cap/silver-crossed cap", "back": "gabi blanket/royal gabi",
		"feet": "leather sandals/tooled sandals", "face": "cotton scarf/silver-filigree veil"},
	"nahua": {
		"body": "maguey tilmatli/feathered tilmatli", "head": "cotton headband/quetzal-feather headdress", "back": "xicolli cloak/jaguar-pelt cloak",
		"feet": "cactus-fibre sandals/cactli sandals", "face": "jade mask/obsidian mask"},
	"maya": {
		"body": "cotton huipil/embroidered huipil", "head": "cloth headwrap/jade-bead headdress", "back": "woven cape/quetzal cape",
		"feet": "hide sandals/jade-anklet sandals", "face": "painted face-mask/jade death-mask"},
	"andean": {
		"body": "alpaca poncho/vicuna poncho", "head": "chullo cap/tasselled chullo", "back": "woven manta/tocapu manta",
		"feet": "llama-hide ojotas/silver-buckled ojotas", "face": "wool scarf/gold sun-mask"},
	"turtle": {
		"body": "buckskin shirt/quilled buckskin shirt", "head": "porcupine roach/eagle-feather war bonnet", "back": "bison robe/painted bison robe",
		"feet": "moccasins/beaded moccasins", "face": "bark mask/shell gorget mask"},
	"inuit": {
		"body": "sealskin parka/caribou parka", "head": "fur hood/polar-bear hood", "back": "fur wrap/bear-hide wrap",
		"feet": "sealskin kamiks/embroidered kamiks", "face": "bone snow-goggles/ivory snow-goggles"},
	"polynesian": {
		"body": "tapa-cloth wrap/feather-and-tapa mahiole cape", "head": "woven pandanus hat/feathered mahiole", "back": "tapa cloak/feathered 'ahu 'ula",
		"feet": "plaited sandals/shell-anklet sandals", "face": "tiki mask/tattooed shell mask"},
	"norse": {
		"body": "wool tunic/mail-lined tunic", "head": "wool cap/horned spangenhelm", "back": "fur cloak/wolf-fur cloak",
		"feet": "turnshoes/fur-lined boots", "face": "nasal guard/wolf-pelt guard"},
	"celtic": {
		"body": "checked wool tunic/torc-and-plaid tunic", "head": "bronze circlet/winged bronze helm", "back": "woad-dyed plaid/great plaid",
		"feet": "hide brogues/bronze-studded brogues", "face": "woad-painted guard/bronze boar mask"},
	"latin": {
		"body": "wool tunic/senator's toga", "head": "bronze galea/crested galea", "back": "paenula cloak/purple-edged cloak",
		"feet": "caligae/gilded caligae", "face": "theatre mask/gilded cavalry mask"},
	"slavic": {
		"body": "embroidered rubakha/fur-trimmed kaftan", "head": "sheepskin cap/fur kolpak", "back": "wool svita/bear-fur shuba",
		"feet": "bast lapti/felt valenki", "face": "wool scarf/carved folk mask"},
	"eldritch": {
		"body": "tattered shroud/star-stitched shroud", "head": "tarnished cowl/many-eyed cowl", "back": "mist-grey cloak/void-lined cloak",
		"feet": "wrapped rags/hollow-soled sandals", "face": "blank mask/mirror-black mask"},
}

var attireSlots = []string{"body", "head", "back", "feet", "face"}

// attireOverlay is the genre material word put in front of regional garments.
var attireOverlay = map[string]string{
	"mythic": "ambrosial", "steampunk": "brass-buttoned", "dieselpunk": "oilcloth", "atompunk": "chrome-flecked",
	"cyberpunk": "neon-threaded", "biopunk": "spore-silk", "solarpunk": "sun-woven",
	"wasteland": "scrap-patched", "twilight": "ash-stained",
}

func attireSpheres() []string {
	out := make([]string, 0, len(attireTable))
	for k := range attireTable {
		out = append(out, k)
	}
	return out
}

// MakeAttire builds a regional garment for a culture sphere and slot. Unknown spheres fall back to MakeArmor.
func (g *Game) MakeAttire(sphere string, slot string, tier int, r *Rng) *Item {
	row, ok := attireTable[sphere]
	if !ok || row[slot] == "" {
		return g.MakeArmor(slot, tier, r)
	}
	tier = clampi(tier, 0, 5)
	parts := strings.SplitN(row[slot], "/", 2)
	noun := parts[0]
	if tier >= 3 && len(parts) > 1 {
		noun = parts[1]
	}
	name := noun
	if ov := attireOverlay[g.Genre]; ov != "" {
		name = ov + " " + noun
	}
	it := g.MakeArmor(slot, 0, r)
	it.ID = g.newID()
	it.Name, it.Tier, it.Region, it.F = name, tier, sphere, tierColor[tier]
	it.AV = tier / 3
	if slot == "body" {
		it.AV = 1 + tier/3
	}
	it.DV = 0
	it.Weight = 2
	it.Bonus = map[int]int{EGO: 1 + tier/3}
	it.Value = 10 * (tier + 1) * (tier + 1)
	it.Lore = fmt.Sprintf("Dress of the %s peoples.", sphere)
	return it
}

// stockTailor fills a tailor with the town's culture dress plus a few foreign pieces.
func (g *Game) stockTailor(a *Actor, bd *BurgData, r *Rng) {
	sphere := "latin"
	if bd != nil {
		if cu := g.W.cultByID[bd.Culture]; cu != nil && attireTable[cu.Sphere] != nil {
			sphere = cu.Sphere
		}
	}
	tier := clampi(a.Level/3, 0, 5)
	for _, sl := range attireSlots {
		a.Inv = append(a.Inv, g.MakeAttire(sphere, sl, tier+r.Int(-1, 1), r))
	}
	for i := 0; i < r.Int(4, 7); i++ {
		a.Inv = append(a.Inv, g.MakeAttire(sphere, Pick(r, attireSlots), tier+r.Int(-1, 2), r))
	}
	sp := attireSpheres()
	for i := 0; i < 3; i++ {
		other := sp[r.Int(0, len(sp)-1)]
		if other == sphere {
			continue
		}
		a.Inv = append(a.Inv, g.MakeAttire(other, Pick(r, attireSlots), tier+r.Int(-1, 1), r))
	}
	if a.Gold == 0 {
		a.Gold = r.Int(120, 300) + a.Level*20
	}
}

// actorSphere: culture sphere of an NPC's realm ("" if unknown).
func (g *Game) actorSphere(a *Actor) string {
	if a == nil || g.W == nil {
		return ""
	}
	return g.stateSphere(a.State)
}

func (g *Game) stateSphere(id int) string {
	st := g.W.stateByID[id]
	if st == nil {
		return ""
	}
	if cu := g.W.cultByID[st.Culture]; cu != nil {
		return cu.Sphere
	}
	return ""
}

// wornAttire lists the player's equipped regional pieces.
func (g *Game) wornAttire() []*Item {
	var out []*Item
	for _, p := range g.P.Body {
		if p.Item != nil && !p.Severed && p.Item.Region != "" {
			out = append(out, p.Item)
		}
	}
	return out
}

// enemySphere reports whether an NPC's realm is at war with a realm of the given sphere.
func (g *Game) enemySphere(a *Actor, sphere string) bool {
	st := g.W.stateByID[a.State]
	if st == nil {
		return false
	}
	for _, w := range st.Wars {
		if g.stateSphere(w) == sphere {
			return true
		}
	}
	return false
}

// attireAffinity: reputation-like modifier from the regional clothes you wear (+50 per matching piece up to +150,
// -40 per piece of a sphere the NPC's realm is at war with, down to -120).
func (g *Game) attireAffinity(a *Actor) int {
	mine := g.actorSphere(a)
	if mine == "" {
		return 0
	}
	plus, minus := 0, 0
	for _, it := range g.wornAttire() {
		if it.Region == mine {
			plus += 50
		} else if g.enemySphere(a, it.Region) {
			minus += 40
		}
	}
	return clampi(plus, 0, 150) - clampi(minus, 0, 120)
}

// attireRemark is what an NPC says about your clothes, or "".
func (g *Game) attireRemark(a *Actor) string {
	mine := g.actorSphere(a)
	if mine == "" {
		return ""
	}
	var home, foe *Item
	for _, it := range g.wornAttire() {
		if it.Region == mine && home == nil {
			home = it
		} else if g.enemySphere(a, it.Region) && foe == nil {
			foe = it
		}
	}
	switch {
	case foe != nil:
		return fmt.Sprintf("\"That %s... I know whose people wear that. I do not like it.\"", foe.ShownName())
	case home != nil && len(g.wornAttire()) >= 3:
		return fmt.Sprintf("\"You wear the %s of the %s peoples like one of us.\"", home.ShownName(), mine)
	case home != nil:
		return fmt.Sprintf("\"Good %s. That is honest %s work.\"", home.ShownName(), mine)
	}
	return ""
}

// ShownName is the appearance name (transmog) or the real name.
func (it *Item) ShownName() string {
	if it.Look != "" {
		return it.Look
	}
	return it.Name
}

func (g *Game) clearLook(it *Item) { it.Look = "" }

func tailorCost(it *Item) int { return 25 * (it.Tier + 1) }

// tailorMenu: pick an equipped item, then an appearance source from the pack.
func (g *Game) tailorMenu(a *Actor) {
	g.Popup = Wrap("\"Tell me which garment to remake, and which fashion to remake it after.\"", 100)
	g.Opts = nil
	back := Option{"Enough for now.", func() { g.talkMenu(g.greeting(a)) }}
	for _, p := range g.P.Body {
		eq := p.Item
		if eq == nil || p.Severed || eq.Slot == "" || eq.Kind == "light" {
			continue
		}
		cost := tailorCost(eq) * (100 - g.Skills["persuasion"]*10) / 100
		label := eq.ShownName()
		if eq.Look != "" {
			label += " {{K|(" + eq.Name + ")}}"
		}
		g.Opts = append(g.Opts, Option{fmt.Sprintf("Remake %s — %d %s", label, cost, g.Sk.Currency), func() { g.tailorPick(a, eq, cost) }})
	}
	g.Opts = append(g.Opts, back)
	g.Sel = 0
}

func (g *Game) tailorPick(a *Actor, eq *Item, cost int) {
	g.Popup = Wrap(fmt.Sprintf("\"What shall your %s look like?\"", eq.Name), 100)
	g.Opts = nil
	if eq.Look != "" {
		g.Opts = append(g.Opts, Option{"Restore its original look (free)", func() {
			g.clearLook(eq)
			g.Msg(fmt.Sprintf("{{G|Your %s looks as it did before.}}", eq.Name))
			g.tailorMenu(a)
		}})
	}
	for _, src := range g.P.Inv {
		src := src
		if src == eq || src.Slot != eq.Slot || src.Kind != eq.Kind || src.Name == "" {
			continue
		}
		g.Opts = append(g.Opts, Option{fmt.Sprintf("Look like %s — %d %s", src.ShownName(), cost, g.Sk.Currency), func() {
			if g.P.Gold < cost {
				g.talkMenu("\"Come back with coin.\"")
				return
			}
			g.P.Gold -= cost
			a.Gold += cost
			eq.Look = src.ShownName()
			g.Turn += 60
			g.Msg(fmt.Sprintf("{{G|The tailor remakes your %s to look like %s.}}", eq.Name, eq.Look))
			g.tailorMenu(a)
		}})
	}
	if len(g.Opts) == 0 || (eq.Look != "" && len(g.Opts) == 1) {
		g.Opts = append(g.Opts, Option{"{{K|(You carry nothing of that kind to copy.)}}", func() { g.tailorMenu(a) }})
	}
	g.Opts = append(g.Opts, Option{"Never mind.", func() { g.tailorMenu(a) }})
	g.Sel = 0
}
