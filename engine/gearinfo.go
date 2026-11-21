package engine

import (
	"strconv"
	"strings"
)

// Worn and wielded gear for the sprite layers (prefix gi). For every person drawn on the map it says
// what they wear on head, body, back, hands and feet, which shield and which weapon family they hold,
// with material colours and a rarity tier. The player's and companions' looks come from their real
// equipped items only; other people use their equipped items and fill the empty slots from their role,
// caste, culture sphere kit (wpSpheres), genre and armour tier, picked by hashes of seed and actor id.
// Read-only: it never draws g.R and changes nothing.

// GearLook is one person's layered gear. Short keys, empty fields omitted.
type GearLook struct {
	Hd  string `json:"h,omitempty"`  // helm greathelm hood hat cap circlet crown goggles mask turban bandana
	HdC string `json:"hc,omitempty"` // colour
	Bd  string `json:"b,omitempty"`  // rags robe tunic apron coat gambeson leather mail scale plate kevlar frame carapace
	BdC string `json:"bc,omitempty"`
	Tr  string `json:"t,omitempty"` // trim colour (belt, hems, crests)
	Bk  string `json:"k,omitempty"` // cloak cape quiver pack wings banner
	BkC string `json:"kc,omitempty"`
	Hn  string `json:"n,omitempty"` // gloves gauntlets
	HnC string `json:"nc,omitempty"`
	Ft  string `json:"f,omitempty"` // boots greaves sandals
	FtC string `json:"fc,omitempty"`
	Sh  string `json:"s,omitempty"` // round kite tower buckler riot energy
	ShC string `json:"sc,omitempty"`
	Wp  string `json:"w,omitempty"` // weapon family (see giWeaponFam)
	WpC string `json:"wc,omitempty"`
	Wr  int    `json:"wr,omitempty"` // weapon rarity 0 common .. 6 unique
	Of  string `json:"o,omitempty"`  // off-hand prop: purse book symbol lantern orb
	R   int    `json:"r,omitempty"`  // best rarity of anything worn or held
	Gn  string `json:"g,omitempty"`  // genre skin: cyber steam diesel atom bio solar waste myth dusk ("" fantasy)
}

// GearInfo maps actor id to gear for every person drawn on the map (same rules as SpriteActors).
func (g *Game) GearInfo() map[int]GearLook {
	if g.Cur == nil || g.Mode == "world" || g.OnWorld || !(g.Mode == "main" || g.Mode == "target" || g.Mode == "look" || g.Mode == "dir") {
		return nil
	}
	def := g.strataDefFor(g.localBurg())
	out := map[int]GearLook{}
	for _, a := range g.Cur.Actors {
		if a == g.P || a.HP <= 0 || a.Species == "" || !g.Cur.In(a.X, a.Y) || !g.visible(a.X, a.Y) {
			continue
		}
		out[a.ID] = g.giLook(a, def)
	}
	if g.P != nil {
		out[g.P.ID] = g.giLook(g.P, def)
	}
	return out
}

// GearJS converts gear looks for the wasm frame (syscall/js cannot take structs); keys are actor ids.
func GearJS(m map[int]GearLook) map[string]any {
	out := make(map[string]any, len(m))
	for id, l := range m {
		o := map[string]any{}
		put := func(k, v string) {
			if v != "" {
				o[k] = v
			}
		}
		put("h", l.Hd)
		put("hc", l.HdC)
		put("b", l.Bd)
		put("bc", l.BdC)
		put("t", l.Tr)
		put("k", l.Bk)
		put("kc", l.BkC)
		put("n", l.Hn)
		put("nc", l.HnC)
		put("f", l.Ft)
		put("fc", l.FtC)
		put("s", l.Sh)
		put("sc", l.ShC)
		put("w", l.Wp)
		put("wc", l.WpC)
		put("o", l.Of)
		put("g", l.Gn)
		if l.Wr > 0 {
			o["wr"] = l.Wr
		}
		if l.R > 0 {
			o["r"] = l.R
		}
		out[strconv.Itoa(id)] = o
	}
	return out
}

// ---- one person ---------------------------------------------------------------

func (g *Game) giLook(a *Actor, def *strataDef) GearLook {
	gn := giGenre(g.Genre)
	l := GearLook{Gn: gn}
	sphere := g.actorSphere(a)
	if a == g.P {
		if w := g.wornAttire(); len(w) > 0 {
			sphere = w[0].Region
		}
	}
	high, outcast := false, false
	if def != nil && a.Caste != "" {
		high, outcast = def.high(a.Caste), def.outcast(a.Caste)
	}
	h := HashStr(g.Seed + ":gi:" + strconv.Itoa(a.ID))
	cloth, trim := giCloth(sphere, gn, high, outcast, h)
	l.Tr = trim
	rar := 0
	note := func(it *Item) {
		if r := rarityIndex(it.Rarity); r > rar {
			rar = r
		}
	}
	// real equipped items first
	var face *Item
	for _, p := range a.Body {
		if p.Severed || p.Item == nil {
			continue
		}
		it := p.Item
		n := strings.ToLower(it.ShownName())
		switch p.Slot {
		case "head":
			l.Hd, l.HdC = giHead(n), giMat(n, gn, "")
			note(it)
		case "face":
			face = it
		case "body":
			l.Bd = giBody(n)
			l.BdC = giMat(n, gn, giClothFor(l.Bd, cloth, gn))
			note(it)
		case "back":
			l.Bk = giBack(n)
			l.BkC = giMat(n, gn, giShade(cloth))
			note(it)
		case "hands":
			l.Hn = giHands(n)
			l.HnC = giMat(n, gn, "#6a4a2a")
			note(it)
		case "feet":
			l.Ft = giFeet(n)
			l.FtC = giMat(n, gn, "#5a3a22")
			note(it)
		case "hand":
			if it.Kind == "armor" {
				if l.Sh == "" {
					l.Sh, l.ShC = giShield(n), giMat(n, gn, "#7a5a3a")
					note(it)
				}
			} else if l.Wp == "" && (it.Kind == "weapon" || it.Kind == "ranged" || it.Kind == "artifact" && it.DN > 0) && it.Skill != "natural" {
				l.Wp, l.WpC, l.Wr = giWeaponFam(it), giMat(n, gn, ""), rarityIndex(it.Rarity)
				note(it)
			} else if l.Of == "" {
				l.Of = giProp(it)
			}
		}
	}
	if l.Hd == "" && face != nil {
		n := strings.ToLower(face.ShownName())
		if hd := giHead(n); hd == "goggles" || hd == "mask" {
			l.Hd, l.HdC = hd, giMat(n, gn, "#25252c")
			note(face)
		}
	}
	if (l.Wp == "bow" || l.Wp == "crossbow") && l.Bk == "" {
		l.Bk, l.BkC = "quiver", "#6a4a2a"
	}
	if wings := a.PartFor("wings"); wings != nil && !wings.Severed && l.Bk == "" && a.Species != "celestial" {
		l.Bk, l.BkC = "wings", "#f4ecd0"
	}
	real := a == g.P || a.Comp != nil
	if !real {
		g.giDefaults(a, &l, sphere, gn, high, outcast, cloth, h)
		if a.Boss && rar < 3 {
			rar = 3
		}
	}
	if l.Bd == "" {
		l.Bd = "tunic"
		if real {
			l.Bd = ""
		}
	}
	if l.Bd != "" && l.BdC == "" {
		l.BdC = giClothFor(l.Bd, cloth, gn)
	}
	l.R = rar
	return l
}

// giDefaults fills a non-player's empty slots from role, sphere kit, genre and armour tier.
func (g *Game) giDefaults(a *Actor, l *GearLook, sphere, gn string, high, outcast bool, cloth string, h uint64) {
	key := strings.ToLower(a.Role + " " + a.Title)
	kit := giKitFor(key)
	tier := clampi(a.Level/5, 0, 5)
	mat := ""
	if g.Sk != nil {
		mat = g.Sk.Mats[tier]
	}
	metal := giMat(mat, gn, "")
	armour := sprArmor(a)
	sk := wpSpheres[sphere]
	guns := wpGunGenres[g.Genre]
	modern := gn == "cyber" || gn == "atom" || gn == "diesel" || gn == "solar" || gn == "waste" || gn == "bio"

	// weapon: wielded, else carried (not shop stock), else the role's
	if l.Wp == "" && !isTrader(a) {
		for _, it := range a.Inv {
			if (it.Kind == "weapon" || it.Kind == "ranged") && it.Skill != "natural" {
				n := strings.ToLower(it.ShownName())
				l.Wp, l.WpC, l.Wr = giWeaponFam(it), giMat(n, gn, metal), rarityIndex(it.Rarity)
				break
			}
		}
	}
	if l.Wp == "" && kit.wp != "" {
		wp := kit.wp
		switch {
		case wp == "spear" && guns && kit.martial:
			wp = "longgun"
		case wp == "kit" && sk != nil && len(sk.Weapons) > 0: // soldier: a sphere weapon
			wp = giFamName(sk.Weapons[h%uint64(len(sk.Weapons))].Fam)
		case wp == "kit":
			wp = "sword"
		}
		l.Wp, l.WpC = wp, metal
		if wp == "staff" || wp == "book" || wp == "bow" {
			l.WpC = ""
		}
	}
	if l.Of == "" {
		l.Of = kit.of
	}
	// shield
	if l.Sh == "" && kit.sh && !giTwoHanded(l.Wp) {
		typ := "round"
		if sk != nil && sk.Shield != nil {
			typ = giShield(sk.Shield.Fam)
		}
		if modern {
			typ = "riot"
		}
		l.Sh, l.ShC = typ, giShade(cloth)
		if typ == "riot" {
			l.ShC = "#3a4048"
		}
	}
	// head
	if l.Hd == "" {
		hd := kit.hd
		if hd == "" && high && h%3 == 0 {
			hd = "circlet"
		}
		if hd == "" && !outcast && h%5 == 0 && sphere != "" {
			hd = giSphereHat(sphere)
		}
		if hd == "helm" && armour >= 3 {
			hd = "greathelm"
		}
		if hd == "goggles" && gn != "steam" && gn != "diesel" && gn != "atom" && gn != "cyber" && gn != "waste" {
			hd = "cap"
		}
		l.Hd = hd
		switch hd {
		case "helm", "greathelm":
			l.HdC = metal
			if modern {
				l.HdC = giGenreArmour(gn)
			}
		case "crown", "circlet":
			l.HdC = "#f0d060"
		case "goggles", "mask":
			l.HdC = "#25252c"
		case "":
		default:
			l.HdC = giShade(cloth)
		}
	}
	// body: armour tier wins over role clothes
	if l.Bd == "" {
		bd := kit.bd
		if outcast && (bd == "tunic" || bd == "") {
			bd = "rags"
		}
		switch {
		case armour >= 3:
			bd = "plate"
			if gn == "cyber" || gn == "atom" {
				bd = "frame"
			}
		case armour == 2 && bd != "robe":
			bd = "mail"
			if sk != nil && len(sk.Armor) > 0 {
				bd = giBody(sk.Armor[h%uint64(len(sk.Armor))].Fam)
			}
			if modern {
				bd = "kevlar"
			}
			if gn == "bio" {
				bd = "carapace"
			}
		case armour == 1 && (bd == "tunic" || bd == "rags" || bd == ""):
			bd = "leather"
			if h%2 == 0 && !modern {
				bd = "gambeson"
			}
			if gn == "cyber" {
				bd = "coat"
			}
		}
		if gn == "waste" && bd == "tunic" {
			bd = "leather"
		}
		l.Bd = bd
		switch bd {
		case "mail", "scale", "lamellar", "plate", "frame":
			l.BdC = metal
			if modern {
				l.BdC = giGenreArmour(gn)
			}
		case "kevlar":
			l.BdC = giGenreArmour(gn)
		case "carapace":
			l.BdC = "#4a3a28"
		}
	}
	// back
	if l.Bk == "" {
		bk := kit.bk
		if bk == "" && high && (h>>3)%2 == 0 {
			bk = "cape"
		}
		l.Bk = bk
		switch bk {
		case "quiver", "pack":
			l.BkC = "#6a4a2a"
			if bk == "pack" && gn == "cyber" {
				l.BkC = "#26282c"
			}
		case "banner":
			l.BkC = cloth
		case "":
		default:
			l.BkC = giShade(cloth)
			if high {
				l.BkC = giRich(h)
			}
		}
	}
	if l.Wp == "bow" && l.Bk == "" {
		l.Bk, l.BkC = "quiver", "#6a4a2a"
	}
	// hands and feet
	if l.Hn == "" && (armour >= 2 || kit.gloves) {
		l.Hn, l.HnC = "gloves", "#6a4a2a"
		if armour >= 3 {
			l.Hn, l.HnC = "gauntlets", metal
		}
	}
	if l.Ft == "" && !outcast {
		l.Ft, l.FtC = "boots", "#5a3a22"
		if armour >= 3 {
			l.Ft, l.FtC = "greaves", metal
		} else if sphere == "polynesian" || sphere == "sea" || sphere == "mythic" || gn == "myth" || sphere == "maya" || sphere == "nahua" || sphere == "indic" {
			l.Ft, l.FtC = "sandals", "#8a6a3a"
		}
	}
	// bandits wear what they found: a mismatched piece
	if kit.mismatch {
		l.BkC = ""
		if l.Bk == "" && h%2 == 0 {
			l.Bk, l.BkC = "cloak", "#4a3a2a"
		} else if l.Bk != "" {
			l.BkC = "#3a3a32"
		}
		if l.Hn == "" {
			l.Hn, l.HnC = "gloves", "#7a5232"
		}
		l.FtC = "#3a2a1a"
	}
}

// ---- role kits -----------------------------------------------------------------

type giKit struct {
	keys           []string
	hd, bd, bk, wp string
	of             string
	sh, gloves     bool
	martial        bool // guards: guns replace spears in gun genres
	mismatch       bool
}

// giKits: first match on role + title wins; wp "kit" picks a sphere weapon.
var giKits = []giKit{
	{keys: []string{"guard", "sentry", "patrol", "watch", "hzgate", "gaoler", "jailer", "warden"}, hd: "helm", bd: "gambeson", wp: "spear", sh: true, martial: true},
	{keys: []string{"soldier", "marshal", "knight", "duellist", "warrior", "mercenary", "champion", "gladiator", "recruit"}, hd: "helm", bd: "gambeson", wp: "kit", sh: true},
	{keys: []string{"executioner"}, hd: "mask", bd: "leather", wp: "great"},
	{keys: []string{"bandit", "outlaw", "raider", "thief", "brigand", "cutthroat", "smuggler", "pirate", "nightguild", "villain"}, hd: "bandana", bd: "leather", wp: "dagger", mismatch: true},
	{keys: []string{"hunter", "huntress", "ranger", "scout", "trapper"}, hd: "hood", bd: "leather", wp: "bow"},
	{keys: []string{"woodcutter", "lumberjack", "forester"}, bd: "tunic", wp: "axe"},
	{keys: []string{"smith", "armourer", "armorer", "forge", "tinker", "mechanic", "shipwright", "tanner"}, hd: "goggles", bd: "apron", wp: "hammer", gloves: true},
	{keys: []string{"priest", "cleric", "monk", "nun", "abbot", "shaman", "imam", "rabbi", "druid", "oracle", "seer", "clergy"}, hd: "", bd: "robe", wp: "staff", of: "symbol"},
	{keys: []string{"pilgrim", "hermit"}, hd: "hood", bd: "robe", wp: "staff"},
	{keys: []string{"scholar", "chronicler", "naturalist", "sage", "scribe", "librarian", "geneticist", "astronomer", "teacher"}, hd: "cap", bd: "robe", of: "book"},
	{keys: []string{"physician", "apothecary", "herbalist", "healer", "alchemist", "doctor"}, hd: "cap", bd: "robe", of: "lantern"},
	{keys: []string{"bookseller"}, hd: "cap", bd: "coat", of: "book"},
	{keys: []string{"merchant", "trader", "hawker", "shopkeeper", "exotic", "fence", "banker", "moneylender", "dockmaster", "stationmaster"}, hd: "hat", bd: "coat", of: "purse"},
	{keys: []string{"khan", "king", "queen", "noble", "lord", "lady", "prince", "princess", "duke", "count", "emperor", "empress", "ruler", "sultan", "chief"}, hd: "crown", bd: "robe", bk: "cape"},
	{keys: []string{"notable", "magistrate", "governor", "mayor", "elder", "judge", "reformer"}, hd: "", bd: "robe", bk: "cape"},
	{keys: []string{"herald", "crier"}, hd: "hat", bd: "tunic", bk: "banner"},
	{keys: []string{"farmer", "shepherd", "drover", "forager", "prospector", "ostler", "beastkeeper", "gardener"}, hd: "hat", bd: "tunic", wp: "staff"},
	{keys: []string{"sailor", "ferry", "fisher", "diver", "boatman"}, hd: "bandana", bd: "tunic"},
	{keys: []string{"weaver", "tailor", "cook", "innkeeper", "bard", "minstrel"}, bd: "tunic"},
	{keys: []string{"child", "captive", "refugee", "homeless", "unfree", "beggar", "slave", "condemned", "patient", "trapped"}, bd: "rags"},
	{keys: []string{"rival adventurer", "adventurer", "former companion"}, hd: "hood", bd: "leather", bk: "pack", wp: "kit"},
	{keys: []string{"foe"}, bd: "leather", wp: "kit"},
}

func giKitFor(key string) giKit {
	for _, k := range giKits {
		for _, w := range k.keys {
			if strings.Contains(key, w) {
				return k
			}
		}
	}
	return giKit{bd: "tunic"}
}

// ---- item classification ----------------------------------------------------------

func giHas(n string, words ...string) bool {
	for _, w := range words {
		if strings.Contains(n, w) {
			return true
		}
	}
	return false
}

func giHead(n string) string {
	switch {
	case giHas(n, "crown", "diadem", "tiara"):
		return "crown"
	case giHas(n, "circlet", "laurel", "halo", "coronet"):
		return "circlet"
	case giHas(n, "goggle", "lens", "monocle", "shades", "sunglasses", "eye-cluster", "sun visor"):
		return "goggles"
	case giHas(n, "great helm", "tanker", "power helmet", "riot helm", "diving helm", "welder", "exo-helm", "crowned helm", "corinthian", "crested", "smart helmet", "kabuto"):
		return "greathelm"
	case giHas(n, "helm", "casque", "sallet", "morion", "bascinet", "visor", "shako"):
		return "helm"
	case giHas(n, "hood", "cowl"):
		return "hood"
	case giHas(n, "turban", "pagri", "keffiyeh", "headcloth", "headwrap", "kolah"):
		return "turban"
	case giHas(n, "bandana", "kerchief", "headband", "bandanna"):
		return "bandana"
	case giHas(n, "mask", "veil", "rebreather", "face-guard"):
		return "mask"
	case giHas(n, "hat", "bowler", "tricorn", "sombrero", "straw", "brim", "fez", "pilos", "sedge", "caping"):
		return "hat"
	}
	return "cap"
}

func giBody(n string) string {
	switch {
	case giHas(n, "powerframe", "power armo", "exo", "frame", "walker harness"):
		return "frame"
	case giHas(n, "kevlar", "ballistic", "flak", "armorweave", "tire-tread", "combat armor"):
		return "kevlar"
	case giHas(n, "chitin", "carapace", "shell", "keratin"):
		return "carapace"
	case giHas(n, "plate", "cuirass", "harness", "oyoroi", "o-yoroi", "boiler"):
		return "plate"
	case giHas(n, "mail", "hauberk", "chain", "zirah"):
		return "mail"
	case giHas(n, "lamellar", "do-maru", "slat"):
		return "lamellar"
	case giHas(n, "scale", "brigandine", "nails", "riveted"):
		return "scale"
	case giHas(n, "gambeson", "padded", "quilted", "linothorax", "cork"):
		return "gambeson"
	case giHas(n, "coat", "duster", "trench", "jacket", "garb"):
		return "coat"
	case giHas(n, "leather", "hide", "jerkin", "pelt", "fur"):
		return "leather"
	case giHas(n, "robe", "vestment", "habit", "cassock", "kimono", "changshan", "toga", "chiton", "himation", "kaftan", "caftan", "gown", "dress", "sari"):
		return "robe"
	case giHas(n, "apron"):
		return "apron"
	case giHas(n, "rag", "tattered", "threadbare", "sack"):
		return "rags"
	}
	return "tunic"
}

func giBack(n string) string {
	switch {
	case giHas(n, "wing"):
		return "wings"
	case giHas(n, "quiver"):
		return "quiver"
	case giHas(n, "banner", "standard", "flag", "sashimono"):
		return "banner"
	case giHas(n, "pack", "rucksack", "harness", "jetpack", "brood-sac", "tank", "satchel", "bag"):
		return "pack"
	case giHas(n, "cape", "mantle", "chlamys", "haori", "shawl", "sash", "himation", "apron"):
		return "cape"
	}
	return "cloak"
}

func giHands(n string) string {
	if giHas(n, "gauntlet", "bracer", "power", "claw", "knuckle", "scrap", "plate", "steel", "iron") {
		return "gauntlets"
	}
	return "gloves"
}

func giFeet(n string) string {
	switch {
	case giHas(n, "greave", "sabaton", "plate", "steel", "iron", "mag-boot", "hoof"):
		return "greaves"
	case giHas(n, "sandal", "geta", "zori", "slipper", "wraps"):
		return "sandals"
	}
	return "boots"
}

func giShield(n string) string {
	switch {
	case giHas(n, "buckler"):
		return "buckler"
	case giHas(n, "kite", "heraldic"):
		return "kite"
	case giHas(n, "tower", "pavise", "kalasag", "tate", "scutum", "boiler"):
		return "tower"
	case giHas(n, "riot", "trench shield"):
		return "riot"
	case giHas(n, "energy", "hardlight", "force"):
		return "energy"
	}
	return "round"
}

// giFamName maps a legacy skill or wpFamilies id to a drawn family.
func giFamName(f string) string {
	switch f {
	case "longblade":
		return "sword"
	case "shortblade":
		return "dagger"
	case "cudgel":
		return "mace"
	case "heavy":
		return "great"
	case "longbow", "missile":
		return "bow"
	case "thrown":
		return "tknife"
	case "firearm":
		return "pistol"
	}
	return f
}

// giWeaponFam: the drawn family of a wielded item. Families: sword dagger axe mace hammer spear flail whip
// staff fist hook great javelin taxe tknife chakram boomerang bolas sling blowgun crossbow bow atlatl
// pistol longgun repeater wand orb.
func giWeaponFam(it *Item) string {
	n := strings.ToLower(it.ShownName())
	if f := wpFamOf(it); f != "" && f != "shield" {
		return giFamName(f)
	}
	switch it.Skill {
	case "missile", "firearm":
		switch {
		case giHas(n, "pistol", "revolver", "derringer", "blaster", "pepperbox"):
			return "pistol"
		case giHas(n, "gatling", "machine", "repeater", "smg", "tommy", "minigun"):
			return "repeater"
		case giHas(n, "gun", "rifle", "musket", "arquebus", "carbine", "laser", "cannon", "railgun", "shotgun"):
			return "longgun"
		case giHas(n, "crossbow", "arbalest"):
			return "crossbow"
		case giHas(n, "sling"):
			return "sling"
		}
		return "bow"
	case "cudgel":
		if giHas(n, "hammer", "maul", "mallet") {
			return "hammer"
		}
		return "mace"
	case "longblade", "shortblade", "axe", "heavy", "spear", "flail", "whip", "staff", "fist", "hook", "thrown":
		return giFamName(it.Skill)
	}
	switch {
	case giHas(n, "wand", "rod", "sceptre", "scepter"):
		return "wand"
	case giHas(n, "staff", "stave"):
		return "staff"
	case giHas(n, "sword", "blade", "sabre", "saber", "katana"):
		return "sword"
	case giHas(n, "axe"):
		return "axe"
	case giHas(n, "spear", "lance", "pike", "trident"):
		return "spear"
	case giHas(n, "hammer", "mace", "club"):
		return "mace"
	case giHas(n, "knife", "dagger"):
		return "dagger"
	}
	if it.Kind == "ranged" {
		return "bow"
	}
	return "orb"
}

// giProp: a non-weapon held in a hand.
func giProp(it *Item) string {
	n := strings.ToLower(it.ShownName())
	switch {
	case it.Kind == "book" || giHas(n, "book", "tome", "scroll"):
		return "book"
	case it.Kind == "light" || giHas(n, "lantern", "lamp", "torch"):
		return "lantern"
	case giHas(n, "purse", "coin", "pouch"):
		return "purse"
	case giHas(n, "symbol", "icon", "ankh", "rosary", "holy"):
		return "symbol"
	}
	return ""
}

func giTwoHanded(f string) bool {
	switch f {
	case "great", "staff", "longgun", "repeater", "bow", "crossbow":
		return true
	}
	return false
}

// ---- colours ---------------------------------------------------------------------

func giGenre(g string) string {
	switch g {
	case "cyberpunk":
		return "cyber"
	case "steampunk":
		return "steam"
	case "dieselpunk":
		return "diesel"
	case "atompunk":
		return "atom"
	case "biopunk":
		return "bio"
	case "solarpunk":
		return "solar"
	case "wasteland":
		return "waste"
	case "mythic":
		return "myth"
	case "twilight":
		return "dusk"
	}
	return ""
}

// giMats: material keyword → colour, first match wins.
var giMats = []struct{ k, c string }{
	{"starmetal", "#a8b0ff"}, {"ichor", "#e8c860"}, {"uranium", "#7ad04a"}, {"aether", "#8ad0e0"}, {"mercurial", "#c8d0d8"},
	{"adamant", "#6a8aa0"}, {"mithril", "#dce4f0"}, {"orichalcum", "#d89a4a"}, {"celestial", "#f0e0a0"}, {"tungsten", "#5a5e66"},
	{"chrome", "#d4dce4"}, {"crucible", "#9aaabb"}, {"tool-steel", "#a0acb8"}, {"steel", "#a8b8c8"}, {"wrought", "#7a7e86"},
	{"cast-iron", "#6a6e76"}, {"iron", "#8a8e96"}, {"bronze", "#b0773a"}, {"copper", "#b8693a"}, {"brass", "#c8a040"},
	{"gilded", "#e0c050"}, {"gold", "#e0c050"}, {"silver", "#d0d4dc"}, {"jade", "#5aa878"}, {"obsidian", "#2a2030"},
	{"crystal", "#a8e0f0"}, {"bone", "#e0d6b8"}, {"ivory", "#ece4cc"}, {"horn", "#c8b088"}, {"chitin", "#4a3a28"},
	{"carapace", "#4a3a28"}, {"kevlar", "#2c3034"}, {"carbon", "#222428"}, {"synth", "#2a2c32"}, {"armorweave", "#26282c"},
	{"scrap", "#8a5a3a"}, {"rust", "#8a4a2a"}, {"tire", "#2a2a2a"}, {"leather", "#7a5232"}, {"hide", "#8a6040"},
	{"fur", "#8a6a4a"}, {"pelt", "#8a6a4a"}, {"rattan", "#b08a50"}, {"bamboo", "#a89a50"}, {"wood", "#8a6a3a"},
	{"hemp", "#a89870"}, {"silk", "#c84a5a"}, {"lacquer", "#8a1a1a"},
}

// giMat is the colour of a material named in n; fallback when none (or the genre's metal when fallback is "").
func giMat(n, gn, fallback string) string {
	n = strings.ToLower(n)
	for _, m := range giMats {
		if strings.Contains(n, m.k) {
			return m.c
		}
	}
	if fallback != "" {
		return fallback
	}
	switch gn {
	case "cyber":
		return "#2e3238"
	case "steam":
		return "#c8a040"
	case "waste":
		return "#8a5a3a"
	case "bio":
		return "#5a6a3a"
	case "myth":
		return "#b0773a"
	case "atom", "diesel":
		return "#6a7468"
	case "dusk":
		return "#6a6878"
	}
	return "#9aa0aa"
}

// giGenreArmour: plate and vest colours of the modern genres.
func giGenreArmour(gn string) string {
	switch gn {
	case "cyber":
		return "#2a2c30"
	case "atom":
		return "#7a8a78"
	case "diesel":
		return "#5a6048"
	case "waste":
		return "#6a5a48"
	case "solar":
		return "#7a9a5a"
	case "bio":
		return "#4a5a30"
	case "steam":
		return "#b08a40"
	}
	return "#8a909a"
}

// giSpherePal: dress colours per culture sphere {cloth..., trim}.
var giSpherePal = map[string][]string{
	"sinic": {"#8a2a2a", "#2a4a6a", "#3a5a3a", "#d8b040"}, "japonic": {"#2a3a5a", "#6a2a3a", "#4a4a4a", "#c8c0a8"},
	"indic": {"#c86a1a", "#8a2a5a", "#d8b02a", "#f0d060"}, "sea": {"#6a3a1a", "#2a6a5a", "#a84a2a", "#e0b040"},
	"steppe": {"#6a4a2a", "#3a5a7a", "#8a3a2a", "#d0a050"}, "arabic": {"#e8e0c8", "#2a4a6a", "#5a3a2a", "#c8a040"},
	"persian": {"#2a5a7a", "#7a2a4a", "#3a6a5a", "#e0c050"}, "latin": {"#8a2a2a", "#e0d8c0", "#4a4a6a", "#d8b040"},
	"celtic": {"#3a6a3a", "#6a4a2a", "#2a4a6a", "#c89040"}, "norse": {"#5a4a3a", "#3a4a5a", "#7a2a2a", "#b0a080"},
	"slavic": {"#a82a2a", "#e8e0d0", "#3a4a2a", "#d8a040"}, "bantu": {"#c86a1a", "#2a5a3a", "#8a2a2a", "#e0c050"},
	"yoruba": {"#3a3a8a", "#c8a02a", "#6a2a5a", "#e8d8a0"}, "mande": {"#e8e0c8", "#4a3a2a", "#2a4a7a", "#c8a040"},
	"ethiopic": {"#e8e4d8", "#2a6a3a", "#c8a030", "#d84a2a"}, "maya": {"#2a7a6a", "#c84a2a", "#e8d8a0", "#40a890"},
	"nahua": {"#c84a2a", "#2a6a8a", "#e8d8a0", "#40a890"}, "andean": {"#a82a3a", "#d8a02a", "#2a5a6a", "#e8d0a0"},
	"inuit": {"#8a6a4a", "#c8b898", "#5a4a3a", "#e0e0e0"}, "polynesian": {"#a86a3a", "#e8d8b0", "#2a6a5a", "#d84a2a"},
	"turtle": {"#8a5a3a", "#c8b088", "#3a5a3a", "#d84a2a"}, "eldritch": {"#3a2a4a", "#2a3a3a", "#5a2a3a", "#8ad0a0"},
}

var giGenrePal = map[string][]string{
	"cyber": {"#26282c", "#2a2a3a", "#3a2a3a", "#e040c0"}, "steam": {"#5a3a2a", "#3a3a4a", "#6a4a2a", "#c8a040"},
	"diesel": {"#4a4a3a", "#3a3a3a", "#5a4a3a", "#b0a070"}, "atom": {"#3a6a8a", "#c8c0a8", "#8a3a3a", "#f0c040"},
	"bio": {"#3a5a3a", "#5a4a3a", "#4a3a4a", "#8ad060"}, "solar": {"#6a8a4a", "#e8d8a0", "#3a6a7a", "#f0c040"},
	"waste": {"#6a5a48", "#5a4a3a", "#7a6a50", "#c8b898"}, "dusk": {"#3a3440", "#2a2a2a", "#4a3a3a", "#a89878"},
}

// giCloth picks a person's dress colour and trim from sphere (else genre) palette and caste.
func giCloth(sphere, gn string, high, outcast bool, h uint64) (string, string) {
	pal := giSpherePal[sphere]
	if gp := giGenrePal[gn]; gp != nil && (pal == nil || gn == "cyber" || gn == "waste") {
		pal = gp
	}
	if pal == nil {
		pal = []string{"#8a6a4a", "#6a7a4a", "#5a6a8a", "#7a5a6a", "#b8a078"}
	}
	c := pal[h%uint64(len(pal)-1)]
	t := pal[len(pal)-1]
	switch {
	case high:
		c = giRich(h >> 5)
		t = "#f0d060"
	case outcast:
		c, t = "#6a5e4e", "#8a7a60"
	}
	return c, t
}

func giRich(h uint64) string {
	rich := []string{"#7a2a5a", "#2a4a7a", "#8a6a1a", "#6a1a2a", "#2a5a4a"}
	return rich[h%uint64(len(rich))]
}

// giClothFor: the drawn colour of a body garment of this kind.
func giClothFor(bd, cloth, gn string) string {
	switch bd {
	case "leather":
		return "#7a5232"
	case "apron":
		return "#5a3a22"
	case "rags":
		return "#7a6e58"
	case "gambeson":
		return "#b8a888"
	case "kevlar", "frame":
		return giGenreArmour(gn)
	case "carapace":
		return "#4a3a28"
	case "mail", "scale", "lamellar", "plate":
		return giMat("", gn, "")
	}
	return cloth
}

// giShade darkens a #rrggbb colour by a quarter.
func giShade(c string) string {
	if len(c) != 7 || c[0] != '#' {
		return c
	}
	v, err := strconv.ParseUint(c[1:], 16, 32)
	if err != nil {
		return c
	}
	r, g, b := (v>>16)&255*3/4, (v>>8)&255*3/4, v&255*3/4
	const hx = "0123456789abcdef"
	return string([]byte{'#', hx[r>>4], hx[r&15], hx[g>>4], hx[g&15], hx[b>>4], hx[b&15]})
}

// giSphereHat: a sphere's everyday headwear.
func giSphereHat(s string) string {
	switch s {
	case "arabic", "persian", "indic", "mande", "ethiopic":
		return "turban"
	case "sinic", "japonic", "sea", "andean":
		return "hat"
	case "steppe", "slavic", "norse", "inuit", "celtic":
		return "cap"
	case "polynesian", "nahua", "maya", "yoruba", "bantu", "turtle":
		return "bandana"
	}
	return "hood"
}
