package engine

import (
	"fmt"
	"strings"
)

// Service establishments (prefix pl): the modification saloon (role "modder": fits, strips and
// tunes weapon and armor mods for coin, no recipe or bits needed) and the mutation house (role
// "geneticist": buys mutations, raises one past the cap, strips defects, purges glimmer, black
// splices). Placed by zoneHooks with side rngs (`seed:pl:zoneKey`), so zone generation is unchanged.
// State in g.ext("pl"). Everything is dialogue menus; no extra screen.

type plState struct {
	Over    map[string]int  `json:"over,omitempty"`    // mutation id -> 1 once raised past the cap
	Sold    map[string]bool `json:"sold,omitempty"`    // saloon stock already bought: "<actor>:<week>:<i>"
	Bought  map[string]bool `json:"bought,omitempty"`  // mutation house offers bought this week
	PurgeTo int             `json:"purgeTo,omitempty"` // game turn until which the glimmer is purged
}

func (g *Game) pl() *plState {
	st := g.ext("pl", func() any { return &plState{} }).(*plState)
	if st.Over == nil {
		st.Over = map[string]int{}
	}
	if st.Sold == nil {
		st.Sold = map[string]bool{}
	}
	if st.Bought == nil {
		st.Bought = map[string]bool{}
	}
	return st
}

func init() {
	zoneHooks = append(zoneHooks, plZone)
	talkHooks = append(talkHooks, plTalk)
	enterHooks = append(enterHooks, plPurgeEnter)
	sideHooks = append(sideHooks, func(g *Game) []string {
		if g.plGlimmerPurged() {
			return []string{"{{c|Glimmer purged}}"}
		}
		return nil
	})
}

// plGlimmerPurged: the glimmer purge is still working. (Hunters that glimmer.go spawns while it
// lasts are removed by plPurgeEnter; muGlimmer may also return 0 while this is true.)
func (g *Game) plGlimmerPurged() bool { return g.pl().PurgeTo > g.Turn }

// ---- genre skins ----

type plSkin struct{ Saloon, Modder, House, Gene string }

var plSkins = map[string]plSkin{
	"fantasy":    {"runesmith's hall", "runesmith", "flesh-garden", "flesh-gardener"},
	"mythic":     {"daedalian workshop", "daedalian", "chimera-shrine", "metamorph-priest"},
	"steampunk":  {"gunsmith saloon", "gunsmith", "alchemist's vats", "vat-alchemist"},
	"dieselpunk": {"workshop-saloon", "wrench-man", "ward clinic", "ward doctor"},
	"atompunk":   {"mod parlour", "gadgeteer", "rad-clinic", "rad-doctor"},
	"cyberpunk":  {"chop-shop", "ripper", "splice parlour", "splicer"},
	"biopunk":    {"grafting parlour", "graft-wright", "gene clinic", "gene-wright"},
	"solarpunk":  {"makers' bay", "maker", "seed-garden", "seed-keeper"},
	"wasteland":  {"scrap saloon", "gunsmith", "rad-clinic", "mutie-doc"},
	"twilight":   {"reliquary smithy", "rune-wright", "blood ministry", "blood-minister"},
}

func (g *Game) plSkin() plSkin {
	if s, ok := plSkins[g.Genre]; ok {
		return s
	}
	return plSkins["fantasy"]
}

// ---- placement ----

func plSpot(cp *cityPlan, kinds []string) ([2]int, bool) {
	order := append([][2]int{{1, 1}}, sideOrder...)
	for _, k := range kinds {
		for _, s := range order {
			if cp.districts[s] == k {
				return s, true
			}
		}
	}
	return [2]int{}, false
}

func plZone(g *Game, zn *Zone, _ *Rng) {
	if g.W == nil {
		return
	}
	switch {
	case zn.Kind == "town" && zn.Z == 0 && zn.Burg >= 0:
		bd := g.W.burgByID[zn.Burg]
		if bd == nil || g.W.cityTier[bd.ID] < 1 {
			return
		}
		cp := g.cityPlanFor(bd)
		r := NewRng(g.Seed + ":pl:" + zn.Key)
		if s, ok := plSpot(cp, []string{"forge", "artisans"}); ok && s == [2]int{zn.ZX, zn.ZY} {
			plPlaceSaloon(g, zn, bd, r)
		}
		if cp.tier == 2 {
			if s, ok := plSpot(cp, []string{"slums", "academy"}); ok && s == [2]int{zn.ZX, zn.ZY} {
				plPlaceHouse(g, zn, bd, r)
			}
		}
	case zn.Kind == "dungeon" && zn.Z == 1:
		s := g.siteAt(zn.WX, zn.WY, zn.ZX, zn.ZY)
		if s == nil || s.Theme != "hideout" {
			return
		}
		r := NewRng(g.Seed + ":pl:" + zn.Key)
		if !r.Chance(0.4) {
			return
		}
		x, y, ok := zn.FreeSpot(r, nil)
		if !ok {
			return
		}
		a := g.Spawn("bandit", maxi(3, g.zoneLevel(zn.WX, zn.WY, 0)+2), r)
		a.Name, a.Proper, a.Role = Pick(r, []string{"Vessk", "Orrin Vane", "Mother Quill", "Doc Hallow", "Ketter", "Slate"}), true, "geneticist"
		a.Title = g.plSkin().Gene
		a.Desc = "A back-alley " + a.Title + " who works out of the " + g.plSkin().House + " hidden here."
		a.AI, a.Faction, a.Angry, a.Glyph, a.F = "still", "pl", false, '@', CGreen
		a.X, a.Y, a.HomeX, a.HomeY = x, y, x, y
		a.Gold = 150 + r.Int(0, 200)
		a.Set("hideout")
		zn.Actors = append(zn.Actors, a)
	}
}

// plBuild raises a small building with a sign note and returns a floor spot inside it.
func plBuild(g *Game, zn *Zone, r *Rng, name string) (int, int, bool) {
	if rc, ok := g.lot(zn, r, 6, 8, 5, 6); ok {
		dx, dy := g.building(zn, rc[0], rc[1], rc[2], rc[3], TWall, TFloor)
		zn.Notes[dy*MW+dx] = name
		return rc[0] + rc[2]/2, rc[1] + rc[3]/2, true
	}
	return zn.FreeSpot(r, func(x, y int) bool { return zn.At(x, y) == TCobble || zn.At(x, y) == TFloor })
}

func plPlaceNPC(g *Game, zn *Zone, a *Actor, x, y int) {
	a.AI = "still"
	a.X, a.Y, a.HomeX, a.HomeY = x, y, x, y
	zn.Set(x, y, TFloor)
	zn.Actors = append(zn.Actors, a)
}

func plPlaceSaloon(g *Game, zn *Zone, bd *BurgData, r *Rng) {
	sk := g.plSkin()
	x, y, ok := plBuild(g, zn, r, "The "+titleCase(sk.Saloon)+" of "+bd.Name)
	if !ok {
		return
	}
	a := g.makeNPC("modder", bd, g.W.cultByID[bd.Culture], r)
	a.Title = sk.Modder
	a.Desc = fmt.Sprintf("A %s of %s, who keeps the %s.", sk.Modder, bd.Name, sk.Saloon)
	a.Gold = r.Int(300, 600) + a.Level*30
	plPlaceNPC(g, zn, a, x, y)
}

func plPlaceHouse(g *Game, zn *Zone, bd *BurgData, r *Rng) {
	sk := g.plSkin()
	x, y, ok := plBuild(g, zn, r, "The "+titleCase(sk.House)+" of "+bd.Name)
	if !ok {
		return
	}
	a := g.makeNPC("geneticist", bd, g.W.cultByID[bd.Culture], r)
	a.Title = sk.Gene
	a.Desc = fmt.Sprintf("A %s of %s, who runs the %s.", sk.Gene, bd.Name, sk.House)
	a.Gold = r.Int(300, 600) + a.Level*30
	plPlaceNPC(g, zn, a, x, y)
}

// ---- shared helpers ----

// plAdj applies the proprietor's price mood: reputation with their realm and Persuasion.
func (g *Game) plAdj(a *Actor, p int) int {
	f := 100
	rep := g.Rep[a.Faction]
	switch {
	case rep >= 400:
		f = 80
	case rep >= 150:
		f = 90
	case rep <= -200:
		f = 125
	}
	f -= g.Skills["persuasion"] * 5
	return maxi(1, p*maxi(f, 50)/100)
}

func (g *Game) plPay(a *Actor, price int) bool {
	if g.P.Gold < price {
		return false
	}
	g.P.Gold -= price
	a.Gold += price
	g.Turn += 90
	return true
}

func plMenu(g *Game, text string) {
	g.Popup = Wrap(text, 100)
	g.Opts = nil
	g.Sel = 0
}

func plAdd(g *Game, t string, f func()) { g.Opts = append(g.Opts, Option{t, f}) }

func (g *Game) plCoin() string { return g.Sk.Currency }

func plTalk(g *Game, a *Actor) []Option {
	switch a.Role {
	case "modder":
		return []Option{{fmt.Sprintf("{{O|Visit the %s.}}", g.plSkin().Saloon), func() { plSaloon(g, a, "\"Mods, tune-ups, a rare piece or two. Pay and I'll start.\"") }}}
	case "geneticist":
		return []Option{{fmt.Sprintf("{{O|Visit the %s.}}", g.plSkin().House), func() { plHouse(g, a, "") }}}
	}
	return nil
}

// ---- modification saloon ----

func plModKind(it *Item) bool {
	return it.Kind == "weapon" || it.Kind == "ranged" || it.Kind == "armor"
}

func (g *Game) plGear() []*Item {
	var out []*Item
	for _, it := range g.ownedGear() {
		if plModKind(it) {
			out = append(out, it)
		}
	}
	return out
}

// plSlots: slots by rarity (no Tinkering needed here), plus a purchased extra slot.
func plSlots(it *Item) int {
	return []int{1, 1, 2, 2, 3, 4, 4}[rarityIndex(it.Rarity)] + it.Extra["xslot"]
}

func plModPrice(g *Game, a *Actor, it *Item, md *modDef) int {
	b := 0
	for t, n := range md.Cost {
		b += t * n * 6
	}
	return g.plAdj(a, (50+b)*(100+30*rarityIndex(it.Rarity))/100)
}

// plFit applies a mod without slot checks or bits.
func (g *Game) plFit(it *Item, md *modDef) {
	md.Apply(it)
	if it.Extra == nil {
		it.Extra = map[string]int{}
	}
	it.Extra["mod:"+md.ID] = 1
	it.Value += 15 * (len(md.Cost) + 1)
	g.refreshModName(it)
}

// plStrip undoes a mod's effect.
func (g *Game) plStrip(it *Item, md *modDef) {
	switch md.ID {
	case "sharp":
		it.PV -= 2
	case "counterweighted":
		addBonus(it, AGI, -1)
		if it.Bonus[AGI] == 0 {
			delete(it.Bonus, AGI)
		}
	case "flaming":
		if it.Elem == "fire" {
			it.Elem = ""
		}
	case "freezing":
		if it.Elem == "frost" {
			it.Elem = ""
		}
	case "electrified":
		if it.Elem == "shock" {
			it.Elem = ""
		}
	case "reinforced":
		it.AV--
	case "lightweight":
		it.Weight += 2
	case "scoped":
		it.Range -= 3
	case "masterwork":
		it.PV--
		it.DB--
	}
	delete(it.Extra, "mod:"+md.ID)
	it.Value = maxi(1, it.Value-15*(len(md.Cost)+1))
	g.refreshModName(it)
}

func plSaloon(g *Game, a *Actor, text string) {
	plMenu(g, text)
	coin := g.plCoin()
	plAdd(g, "{{O|Fit a mod to my gear.}}", func() { plPickGear(g, a, "fit") })
	plAdd(g, "{{O|Strip a mod from my gear.}}", func() { plPickGear(g, a, "strip") })
	plAdd(g, "{{O|Cut an extra mod slot.}} {{K|(once per item)}}", func() { plPickGear(g, a, "slot") })
	plAdd(g, "{{O|Masterwork tune-up.}} {{K|(once per item)}}", func() { plPickGear(g, a, "tune") })
	week := g.Day() / 7
	for i, it := range plStock(g, a) {
		i, it := i, it
		key := fmt.Sprintf("%d:%d:%d", a.ID, week, i)
		if g.pl().Sold[key] {
			continue
		}
		price := g.plAdj(a, it.Value*3/2+60)
		plAdd(g, fmt.Sprintf("Buy %s {{K|(%s)}} — %d %s", it.Display(), strings.Join(plModsList(it), ", "), price, coin), func() {
			if !g.plPay(a, price) {
				plSaloon(g, a, "\"Come back with coin.\"")
				return
			}
			g.pl().Sold[key] = true
			g.give(it)
			g.Msg("{{G|You buy " + it.Display() + ".}}")
			plSaloon(g, a, "\"A fine piece. Treat it well.\"")
		})
	}
	plAdd(g, "Enough for now.", func() { g.talkMenu(g.greeting(a)) })
}

func plModsList(it *Item) []string {
	var out []string
	for _, id := range modsOf(it) {
		out = append(out, id)
	}
	if len(out) == 0 {
		out = append(out, "unmodded")
	}
	return out
}

func plPickGear(g *Game, a *Actor, op string) {
	plMenu(g, "\"Which piece?\"")
	for _, it := range g.plGear() {
		it := it
		plAdd(g, fmt.Sprintf("%s {{K|(%d/%d slots)}}", it.Display(), len(modsOf(it)), plSlots(it)), func() {
			switch op {
			case "fit":
				plPickMod(g, a, it)
			case "strip":
				plPickStrip(g, a, it)
			case "slot":
				plExtraSlot(g, a, it)
			case "tune":
				plTune(g, a, it)
			}
		})
	}
	plAdd(g, "Never mind.", func() { plSaloon(g, a, "\"Anything else?\"") })
}

func plPickMod(g *Game, a *Actor, it *Item) {
	plMenu(g, fmt.Sprintf("\"What shall %s be fitted with?\" {{K|Slots %d/%d.}}", it.Name, len(modsOf(it)), plSlots(it)))
	for i := range modDefs {
		md := &modDefs[i]
		if !isKind(md.Kinds, it.Kind) || it.Extra["mod:"+md.ID] > 0 {
			continue
		}
		price := plModPrice(g, a, it, md)
		plAdd(g, fmt.Sprintf("%s — %d %s", md.Name, price, g.plCoin()), func() {
			if len(modsOf(it)) >= plSlots(it) {
				plSaloon(g, a, "\"No free slot on that. I can cut another, for a price.\"")
				return
			}
			if !g.plPay(a, price) {
				plSaloon(g, a, "\"Come back with coin.\"")
				return
			}
			g.plFit(it, md)
			g.Msg(fmt.Sprintf("{{G|The %s is fitted to %s.}}", md.Name, it.Name))
			plSaloon(g, a, "\"Done. Try the weight of it.\"")
		})
	}
	plAdd(g, "Never mind.", func() { plSaloon(g, a, "\"Anything else?\"") })
}

func plPickStrip(g *Game, a *Actor, it *Item) {
	plMenu(g, "\"Which mod comes off?\"")
	for _, id := range modsOf(it) {
		md := modByID(id)
		if md == nil {
			continue
		}
		price := plModPrice(g, a, it, md) / 3
		plAdd(g, fmt.Sprintf("%s — %d %s", md.Name, maxi(1, price), g.plCoin()), func() {
			if !g.plPay(a, maxi(1, price)) {
				plSaloon(g, a, "\"Come back with coin.\"")
				return
			}
			g.plStrip(it, md)
			g.Msg(fmt.Sprintf("{{Y|The %s is stripped from %s.}}", md.Name, it.Name))
			plSaloon(g, a, "\"Clean as new.\"")
		})
	}
	plAdd(g, "Never mind.", func() { plSaloon(g, a, "\"Anything else?\"") })
}

func plExtraSlot(g *Game, a *Actor, it *Item) {
	price := g.plAdj(a, 400+150*rarityIndex(it.Rarity))
	switch {
	case it.Extra["xslot"] > 0:
		plSaloon(g, a, "\"I've cut that one as far as it will go.\"")
	case !g.plPay(a, price):
		plSaloon(g, a, fmt.Sprintf("\"A new slot is %d %s. Come back with it.\"", price, g.plCoin()))
	default:
		if it.Extra == nil {
			it.Extra = map[string]int{}
		}
		it.Extra["xslot"] = 1
		it.Value += 60
		g.Msg(fmt.Sprintf("{{G|%s gains an extra mod slot (%d/%d).}}", it.Name, len(modsOf(it)), plSlots(it)))
		plSaloon(g, a, "\"There. Room for one more.\"")
	}
}

func plTune(g *Game, a *Actor, it *Item) {
	price := g.plAdj(a, 120+60*rarityIndex(it.Rarity))
	switch {
	case it.Extra["tuned"] > 0:
		plSaloon(g, a, "\"Already tuned. More would only break it.\"")
	case !g.plPay(a, price):
		plSaloon(g, a, fmt.Sprintf("\"A tune-up is %d %s.\"", price, g.plCoin()))
	default:
		if it.Extra == nil {
			it.Extra = map[string]int{}
		}
		it.Extra["tuned"] = 1
		if it.Kind == "armor" {
			it.AV++
		} else {
			it.PV++
		}
		it.Value += 40
		g.Msg(fmt.Sprintf("{{G|%s is tuned to masterwork standard.}}", it.Name))
		plSaloon(g, a, "\"Balanced, honed, polished.\"")
	}
}

// plStock is the rotating weekly stock: three pre-modded rare pieces, deterministic per proprietor and week.
func plStock(g *Game, a *Actor) []*Item {
	r := NewRng(fmt.Sprintf("%s:plstock:%d:%d", g.Seed, a.ID, g.Day()/7))
	tier := clampi(g.P.Level/4+1, 1, 5)
	rar := "rare"
	if g.P.Level >= 12 {
		rar = "ultrarare"
	}
	var out []*Item
	for i := 0; i < 3; i++ {
		var it *Item
		switch i {
		case 0:
			it = g.MakeWeapon(Pick(r, []string{"longblade", "shortblade", "axe", "cudgel"}), tier, r)
		case 1:
			it = g.MakeArmor(Pick(r, []string{"body", "head", "back", "hands"}), tier, r)
		default:
			it = g.MakeRanged(tier, r)
		}
		g.applyRarity(it, r, rar)
		if it.Extra == nil {
			it.Extra = map[string]int{}
		}
		var pool []*modDef
		for k := range modDefs {
			if isKind(modDefs[k].Kinds, it.Kind) {
				pool = append(pool, &modDefs[k])
			}
		}
		for n := 0; n < 2 && len(pool) > 0; n++ {
			j := r.Int(0, len(pool)-1)
			g.plFit(it, pool[j])
			pool = append(pool[:j:j], pool[j+1:]...)
		}
		out = append(out, it)
	}
	return out
}

// ---- mutation house ----

func (g *Game) plTrueKin() bool { return g.Genotype == "truekin" }

func plHouse(g *Game, a *Actor, text string) {
	sk := g.plSkin()
	if g.plTrueKin() {
		plMenu(g, fmt.Sprintf("The %s looks you over and shakes their head. \"Pure strain. There is nothing here for you to change. Seek an augmenter; they work with metal and graft, not flesh.\"", sk.Gene))
		plAdd(g, "Leave.", func() { g.talkMenu(g.greeting(a)) })
		return
	}
	if text == "" {
		text = "\"The flesh is clay. What shall we shape?\""
	}
	plMenu(g, text)
	g.mutSync()
	coin := g.plCoin()
	plAdd(g, "{{O|Buy a mutation.}}", func() { plBuyList(g, a, false) })
	plAdd(g, "{{O|Raise a mutation past its limit.}} {{K|(risky)}}", func() { plOverList(g, a) })
	plAdd(g, "{{O|Remove a defect.}}", func() { plDefectList(g, a) })
	pp := g.plAdj(a, 120+25*muGlimmer(g)/2)
	plAdd(g, fmt.Sprintf("{{O|Purge my glimmer for three days.}} {{K|(%d %s)}}", pp, coin), func() {
		switch {
		case g.plGlimmerPurged():
			plHouse(g, a, "\"You are still clouded over. It will wear off on its own.\"")
		case !g.plPay(a, pp):
			plHouse(g, a, "\"Not without coin.\"")
		default:
			g.pl().PurgeTo = g.Turn + 3*DayLen
			g.Msg("{{c|The glimmer dims. Hunters will not smell you for three days.}}")
			plHouse(g, a, "\"Quiet now. It will come back.\"")
		}
	})
	plAdd(g, "{{R|Ask about the back room: a black splice.}} {{K|(cheap, risky)}}", func() { plBuyList(g, a, true) })
	plAdd(g, "Enough for now.", func() { g.talkMenu(g.greeting(a)) })
}

func plOffers(g *Game, a *Actor) []string {
	r := NewRng(fmt.Sprintf("%s:plgene:%d:%d", g.Seed, a.ID, g.Day()/7))
	st := g.mu()
	var pool []string
	for _, pd := range PowerDefs {
		if pd.Defect || st.Lv[pd.ID] > 0 || !g.muMorphOK(pd.ID) {
			continue
		}
		pool = append(pool, pd.ID)
	}
	var out []string
	for i := 0; i < 5 && len(pool) > 0; i++ {
		j := r.Int(0, len(pool)-1)
		out = append(out, pool[j])
		pool = append(pool[:j:j], pool[j+1:]...)
	}
	return out
}

func plMutPrice(g *Game, a *Actor, id string, black bool) int {
	p := 220 + 70*maxi(1, powerByID[id].Cost) + 30*g.P.Level
	if black {
		p = p * 55 / 100
	}
	return g.plAdj(a, p)
}

// plLearn buys a mutation with coin through the normal rules (genotype, morphotype, ownership).
func (g *Game) plLearn(id string) (bool, string) {
	g.MP += 4
	ok, msg := g.mutBuy(id)
	if !ok {
		g.MP -= 4
	}
	return ok, msg
}

// plGiveDefect afflicts the player with a random defect and returns its id ("" when none are left).
func (g *Game) plGiveDefect() string {
	var pool []string
	for _, pd := range PowerDefs {
		if pd.Defect && g.P.Pow(pd.ID) == 0 && pd.ID != "repugnant" && pd.ID != "doppelganger" {
			pool = append(pool, pd.ID)
		}
	}
	if len(pool) == 0 {
		return ""
	}
	id := Pick(g.R, pool)
	g.P.Powers[id] = 1
	if id == "brittle" {
		g.P.AVNat -= 2
	}
	return id
}

func plBuyList(g *Game, a *Actor, black bool) {
	text := "\"These are in the vats this week.\""
	if black {
		text = "\"Keep your voice down. The back room is cheaper and it does not always take. Nothing I do there is guaranteed.\""
	}
	plMenu(g, text)
	week := g.Day() / 7
	for _, id := range plOffers(g, a) {
		id := id
		key := fmt.Sprintf("%d:%d:%s", a.ID, week, id)
		if g.pl().Bought[key] {
			continue
		}
		pd := powerByID[id]
		price := plMutPrice(g, a, id, black)
		plAdd(g, fmt.Sprintf("%s {{K|(%s)}} — %d %s", g.Sk.PowerName(id), pd.Desc, price, g.plCoin()), func() {
			if g.P.Gold < price {
				plHouse(g, a, "\"Not without coin.\"")
				return
			}
			if !black {
				ok, msg := g.plLearn(id)
				if !ok {
					plHouse(g, a, "\""+msg+"\"")
					return
				}
				g.plPay(a, price)
				g.pl().Bought[key] = true
				g.Msg(msg)
				plHouse(g, a, "\"It will settle in a day or two.\"")
				return
			}
			g.plPay(a, price)
			g.pl().Bought[key] = true
			roll := g.R.Float()
			var out string
			switch {
			case roll < 0.55:
				ok, msg := g.plLearn(id)
				g.Msg(msg)
				out = map[bool]string{true: "\"Clean. Lucky you.\"", false: "\"" + msg + "\""}[ok]
			case roll < 0.80:
				ok, msg := g.plLearn(id)
				d := g.plGiveDefect()
				g.Msg(msg)
				if d != "" {
					g.Msg("{{R|The splice goes wrong: " + titleCase(d) + ".}}")
				}
				out = "\"It took, mostly.\""
				if !ok {
					out = "\"" + msg + "\""
				}
			default:
				d := g.plGiveDefect()
				if d != "" {
					g.Msg("{{R|The back-room splice fails. You wake with a new affliction: " + titleCase(d) + ".}}")
				}
				out = "\"Ah. That happens. No refunds.\""
			}
			_ = out
			plHouse(g, a, out)
		})
	}
	plAdd(g, "Never mind.", func() { plHouse(g, a, "") })
}

// plOverList raises a capped mutation one level past the cap; 25% chance of gaining a defect.
func plOverList(g *Game, a *Actor) {
	plMenu(g, "\"Past the limit the flesh remembers it was never meant to hold. One step only, and it may bite back.\"")
	st := g.mu()
	capLv := g.mutCap()
	for _, id := range sortedKeys(st.Lv) {
		id := id
		if st.Lv[id] < capLv || g.pl().Over[id] > 0 {
			continue
		}
		price := g.plAdj(a, 350+120*st.Lv[id])
		plAdd(g, fmt.Sprintf("%s to level %d — %d %s", g.Sk.PowerName(id), g.P.Powers[id]+1, price, g.plCoin()), func() {
			if !g.plPay(a, price) {
				plHouse(g, a, "\"Not without coin.\"")
				return
			}
			st.Lv[id]++
			g.pl().Over[id] = 1
			g.mutSync()
			g.Msg(fmt.Sprintf("{{M|%s surges to level %d.}}", g.Sk.PowerName(id), g.P.Powers[id]))
			if g.R.Chance(0.25) {
				if d := g.plGiveDefect(); d != "" {
					g.Msg("{{R|The flesh pays for it: " + titleCase(d) + ".}}")
				}
			}
			plHouse(g, a, "\"Done. Do not ask for more.\"")
		})
	}
	plAdd(g, "Never mind.", func() { plHouse(g, a, "") })
}

func (g *Game) plRemoveDefect(id string) {
	if id == "brittle" {
		g.P.AVNat += 2
	}
	delete(g.P.Powers, id)
}

func plDefectList(g *Game, a *Actor) {
	plMenu(g, "\"Every flaw can be cut out, for a fee.\"")
	for _, pd := range PowerDefs {
		pd := pd
		if !pd.Defect || g.P.Pow(pd.ID) == 0 {
			continue
		}
		price := g.plAdj(a, 250+60*(-pd.Cost)+20*g.P.Level)
		plAdd(g, fmt.Sprintf("%s {{K|(%s)}} — %d %s", titleCase(pd.ID), pd.Desc, price, g.plCoin()), func() {
			if !g.plPay(a, price) {
				plHouse(g, a, "\"Not without coin.\"")
				return
			}
			g.plRemoveDefect(pd.ID)
			g.Msg("{{G|The " + titleCase(pd.ID) + " is cut out of you.}}")
			plHouse(g, a, "\"Clean.\"")
		})
	}
	plAdd(g, "Never mind.", func() { plHouse(g, a, "") })
}

// plPurgeEnter drops glimmer hunters that glimmer.go spawned while the purge lasts.
func plPurgeEnter(g *Game, zn *Zone) {
	if !g.plGlimmerPurged() {
		return
	}
	var keep []*Actor
	for _, o := range zn.Actors {
		if o.Has("hunter") && o.Name != g.P.Name && o.Angry {
			continue
		}
		keep = append(keep, o)
	}
	zn.Actors = keep
}
