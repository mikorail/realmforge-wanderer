package engine

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"sort"
	"strings"
)

// Item and status icon data for the shell's pixel icons (src/icons.ts). Prefix ic.
// Pure functions of game state: no g.R, no writes. Hook: cmd/wasm/main.go fr["icons"] / fr["status"].

// IconSpec describes one icon; the shell draws it procedurally.
type IconSpec struct {
	Kind string `json:"k"`            // silhouette: sword axe bow gun helm plate shield loaf flask scroll key gem coin bones ...
	Mat  string `json:"m,omitempty"`  // material palette
	Rar  int    `json:"r"`            // 0 common .. 6 unique
	Tag  string `json:"tg,omitempty"` // effect: cursed poisoned brewed fire frost shock acid enchanted, cooking tag, status class
	Lv   int    `json:"lv,omitempty"` // heirloom level 0..4, or status severity
	Hash uint32 `json:"h"`            // stable per-name hash for variation
	Fam  string `json:"f,omitempty"`  // weapon family (wpweapons id or legacy skill)
	Name string `json:"n,omitempty"`  // statuses: label
}

// IconRow is one list row of the current screen: the item under text row y between columns x0 and x1.
type IconRow struct {
	Y    int      `json:"y"`
	X0   int      `json:"x0"`
	X1   int      `json:"x1"`
	Icon IconSpec `json:"i"`
	Name string   `json:"n"`
	Desc string   `json:"d,omitempty"`
}

// IconStatus is the sidebar strip: icons for statuses, drawn on grid row Y from column X0.
type IconStatus struct {
	Y     int        `json:"y"`
	X0    int        `json:"x0"`
	Icons []IconSpec `json:"icons"`
}

func icHash(s string) uint32 {
	h := fnv.New32a()
	h.Write([]byte(s))
	return h.Sum32()
}

var icMatWords = []struct{ w, m string }{
	{"starmetal", "plasma"}, {"plasma", "plasma"}, {"energy", "plasma"}, {"photon", "plasma"}, {"ichor", "plasma"},
	{"mithril", "silver"}, {"silver", "silver"}, {"moonsteel", "silver"}, {"gold", "gold"}, {"orichalcum", "gold"}, {"golden", "gold"},
	{"adamant", "steel"}, {"steel", "steel"}, {"titanium", "steel"}, {"alloy", "steel"}, {"chrome", "steel"}, {"carbon", "steel"},
	{"iron", "iron"}, {"bronze", "bronze"}, {"brass", "bronze"}, {"copper", "bronze"}, {"clockwork", "bronze"},
	{"obsidian", "stone"}, {"stone", "stone"}, {"flint", "stone"}, {"granite", "stone"}, {"basalt", "stone"}, {"slate", "stone"},
	{"crystal", "crystal"}, {"prism", "crystal"}, {"diamond", "crystal"}, {"quartz", "crystal"}, {"glass", "glass"}, {"vial", "glass"},
	{"bone", "bone"}, {"ivory", "bone"}, {"tusk", "bone"}, {"fang", "bone"}, {"horn", "bone"}, {"antler", "bone"},
	{"chitin", "bio"}, {"carapace", "bio"}, {"shell", "bio"}, {"scale", "bio"}, {"flesh", "bio"}, {"spore", "bio"}, {"living", "bio"},
	{"leather", "leather"}, {"hide", "leather"}, {"pelt", "leather"}, {"fur", "leather"}, {"studded", "leather"},
	{"linen", "cloth"}, {"silk", "cloth"}, {"wool", "cloth"}, {"cotton", "cloth"}, {"cloth", "cloth"}, {"robe", "cloth"}, {"cloak", "cloth"},
	{"wooden", "wood"}, {"wood", "wood"}, {"oak", "wood"}, {"yew", "wood"}, {"ash", "wood"}, {"bamboo", "wood"}, {"willow", "wood"}, {"reed", "wood"},
}

var icTierMat = [6]string{"wood", "bronze", "iron", "steel", "crystal", "plasma"}

func icMaterial(it *Item, low string) string {
	for _, e := range icMatWords {
		if strings.Contains(low, e.w) {
			return e.m
		}
	}
	switch it.Kind {
	case "weapon", "ranged", "ammo", "armor", "light", "scrap":
		if it.Kind == "armor" && (it.Slot == "body" || it.Slot == "back" || it.Slot == "feet" || it.Slot == "hands" || it.Slot == "hand") && it.Tier <= 1 {
			return "leather"
		}
		return icTierMat[clampi(it.Tier, 0, 5)]
	case "food", "dish", "ingredient", "corpse", "egg", "water", "tonic":
		return "bio"
	case "book", "letter":
		return "cloth"
	case "key":
		return "iron"
	case "gold":
		return "gold"
	}
	return ""
}

func icHas(low string, words ...string) bool {
	for _, w := range words {
		if strings.Contains(low, w) {
			return true
		}
	}
	return false
}

// icKind picks the silhouette and the weapon family.
func icKind(it *Item, low string) (kind, fam string) {
	if f := wpFamOf(it); f != "" {
		fam = f
		switch f {
		case "spear", "javelin", "atlatl":
			return "spear", f
		case "flail", "whip":
			return f, f
		case "staff":
			return "staff", f
		case "fist":
			return "fist", f
		case "hook":
			return "hook", f
		case "great":
			if icHas(low, "axe", "halberd", "glaive") {
				return "axe", f
			}
			if icHas(low, "hammer", "maul", "mace") {
				return "mace", f
			}
			return "sword", f
		case "taxe":
			return "axe", f
		case "tknife":
			return "dagger", f
		case "chakram", "boomerang", "bolas":
			return f, f
		case "sling", "blowgun":
			return f, f
		case "crossbow":
			return "crossbow", f
		case "longbow":
			return "bow", f
		case "pistol", "longgun", "repeater":
			return "gun", f
		case "shield":
			return "shield", f
		}
	}
	switch it.Kind {
	case "weapon":
		switch it.Skill {
		case "longblade":
			return "sword", it.Skill
		case "shortblade":
			return "dagger", it.Skill
		case "axe":
			return "axe", it.Skill
		case "cudgel":
			return "mace", it.Skill
		}
		return "sword", it.Skill
	case "ranged":
		if icHas(low, "gun", "rifle", "pistol", "musket", "blaster", "carbine", "arquebus", "cannon", "laser") {
			return "gun", it.Skill
		}
		if icHas(low, "crossbow") {
			return "crossbow", it.Skill
		}
		return "bow", it.Skill
	case "armor":
		switch it.Slot {
		case "head":
			return "helm", ""
		case "face":
			return "mask", ""
		case "back":
			return "cloak", ""
		case "hand", "hands":
			if icHas(low, "shield", "buckler", "targe") {
				return "shield", ""
			}
			return "gloves", ""
		case "feet":
			return "boots", ""
		case "body":
			if icHas(low, "robe", "cloak", "tunic", "dress", "coat", "vestment", "gown", "jacket", "shirt", "garb", "clothes") {
				return "robe", ""
			}
			return "plate", ""
		}
		if icHas(low, "shield", "buckler") {
			return "shield", ""
		}
		return "plate", ""
	case "ammo":
		return "ammo", ""
	case "food":
		if icHas(low, "meat", "jerky", "steak", "ham", "haunch", "sausage", "fish") {
			return "meat", ""
		}
		return "loaf", ""
	case "dish":
		return "bowl", ""
	case "water":
		return "flask", ""
	case "tonic":
		return "potion", ""
	case "grenade":
		return "grenade", ""
	case "book":
		if icHas(low, "scroll", "map", "chart") {
			return "scroll", ""
		}
		return "book", ""
	case "letter":
		return "letter", ""
	case "key", "lockpick":
		return "key", ""
	case "light":
		return "lantern", ""
	case "scrap":
		return "gear", ""
	case "ingredient":
		return "herb", ""
	case "corpse":
		return "meat", ""
	case "egg":
		return "egg", ""
	case "fossil":
		return "fossil", ""
	case "gold":
		return "coin", ""
	case "augment":
		return "aug", ""
	case "artifact":
		return "relic", ""
	case "relic":
		return "relic", ""
	}
	// names for the open-ended kinds (trophy, clutter, oddity, gcitem, ...)
	switch {
	case icHas(low, "fossil", "ammonite", "trilobite", "amber"):
		return "fossil", ""
	case icHas(low, "scroll", "parchment", "map "):
		return "scroll", ""
	case icHas(low, "key"):
		return "key", ""
	case icHas(low, "gem", "jewel", "ruby", "emerald", "sapphire", "opal", "pearl", "diamond", "crystal", "shard"):
		return "gem", ""
	case icHas(low, "coin", "ducat", "medal", "token"):
		return "coin", ""
	case icHas(low, "flute", "lute", "drum", "harp", "fiddle", "horn ", "pipe", "lyre", "zither", "whistle"):
		return "instrument", ""
	case icHas(low, "collar", "leash", "saddle", "bridle", "kibble", "bird cage", "pet "):
		return "pet", ""
	case icHas(low, "letter", "note", "missive"):
		return "letter", ""
	case icHas(low, "bone", "skull", "tooth", "tusk", "fang", "rib", "skeleton"):
		return "bones", ""
	case icHas(low, "pick", "hammer", "tongs", "saw", "chisel", "needle", "shears", "trowel", "spade", "tool", "kit"):
		return "tool", ""
	case icHas(low, "ring", "amulet", "pendant", "locket", "brooch", "bracelet", "circlet", "crown"):
		return "ring", ""
	case it.Kind == "trophy" || it.Kind == "oddity":
		return "relic", ""
	case it.Kind == "clutter":
		return "curio", ""
	}
	return "junk", ""
}

func icSpecOf(g *Game, it *Item) IconSpec {
	name := Plain(it.ShownName())
	low := strings.ToLower(name)
	kind, fam := icKind(it, low)
	sp := IconSpec{Kind: kind, Fam: fam, Hash: icHash(name)}
	if it.Kind == "artifact" && !it.Ident {
		sp.Kind, sp.Mat = "relic", "crystal"
	} else {
		sp.Mat = icMaterial(it, low)
	}
	if it.Rarity != "" {
		sp.Rar = rarityIndex(it.Rarity)
	} else {
		sp.Rar = clampi(it.Tier/2, 0, 6)
		if it.Kind == "artifact" || it.SimArt > 0 {
			sp.Rar = maxi(sp.Rar, 4)
		}
	}
	switch {
	case it.Extra["cursed"] > 0 || strings.Contains(it.Effect, "cursed") || it.Cat == "cursed":
		sp.Tag = "cursed"
	case it.Elem == "poison" || icHas(strings.ToLower(it.Effect), "poison", "venom") && it.Kind != "ingredient":
		sp.Tag = "poisoned"
	case it.Kind == "tonic" && it.Effect == "brewed":
		sp.Tag = "brewed"
	case it.Elem != "":
		sp.Tag = it.Elem
	case it.Kind == "ingredient" || it.Kind == "dish" || it.Kind == "corpse":
		if it.Effect != "" {
			sp.Tag = strings.SplitN(it.Effect, "+", 2)[0]
		}
	case it.Cat != "" || len(it.Bonus) > 0 || it.Extra["ench"] > 0:
		sp.Tag = "enchanted"
	}
	if it.Kind == "weapon" || it.Kind == "armor" || it.Kind == "ranged" {
		sp.Lv = ggLv(it)
	}
	return sp
}

func icRowOf(g *Game, it *Item, y, x0, x1 int) IconRow {
	r := IconRow{Y: y, X0: x0, X1: x1, Icon: icSpecOf(g, it), Name: Plain(it.Display())}
	r.Desc = icDesc(g, it)
	return r
}

// icDesc is the item's Describe text without markup, one line.
func icDesc(g *Game, it *Item) string {
	lines := it.Describe(g)
	if len(lines) > 2 && it.Kind != "artifact" {
		lines = lines[2:]
	} else if len(lines) > 1 && it.Kind != "artifact" {
		lines = lines[1:]
	}
	var parts []string
	for _, l := range lines {
		if p := strings.TrimSpace(Plain(l)); p != "" {
			parts = append(parts, p)
		}
		if len(parts) >= 3 {
			break
		}
	}
	d := strings.Join(parts, " · ")
	if r := []rune(d); len(r) > 200 {
		d = string(r[:199]) + "…"
	}
	return d
}

// IconRows lists the item under each list row of the current screen (inv, trade, stash, cook, hmd).
func (g *Game) IconRows() []IconRow {
	if g.P == nil || g.Cur == nil {
		return nil
	}
	var out []IconRow
	switch g.Mode {
	case "inv":
		const bx, by, bw = 2, 1, SW - 4
		x, y := bx+2, by+1
		listW := 68
		list := g.invVisible()
		sel := mini(g.Sel, maxi(len(list)-1, 0))
		const rows = 22
		start := 0
		if sel >= rows {
			start = sel - rows + 1
		}
		y += 2 + 1 + 1
		for i := start; i < len(list) && i < start+rows; i++ {
			out = append(out, icRowOf(g, list[i], y, x+3, x+listW))
			y++
		}
		ry := by + 3 + 1
		for _, bp := range g.P.Body {
			if bp.Item != nil && !bp.Severed {
				out = append(out, icRowOf(g, bp.Item, ry, x+listW+2, x+bw-4))
			}
			ry++
		}
	case "trade":
		if g.Talk == nil {
			return nil
		}
		const px, py, pw = 6, 3, MW - 8 - 4
		half := pw / 2
		y := py + 2
		for col, list := range [][]*Item{g.Talk.Inv, g.P.Inv} {
			cx := px + col*(half+1)
			for i, it := range list {
				if i > 22 {
					break
				}
				out = append(out, icRowOf(g, it, y+i, cx, cx+half-1))
			}
		}
	case "stash":
		hs := g.homeSt()
		const px, py, pw, ph = 6, 3, MW - 8 - 4, MH + 4 - 2
		half := pw / 2
		rows := ph - 5
		for c, list := range [][]*Item{g.P.Inv, hs.Stash} {
			sel := g.Sel2
			if c == g.Tab {
				sel = g.Sel
			}
			top := maxi(0, sel-rows+1)
			for i := top; i < len(list) && i-top < rows; i++ {
				cx := px + c*half
				out = append(out, icRowOf(g, list[i], py+2+i-top, cx, cx+half-2))
			}
		}
	case "cook":
		const px, py, pw, ph = 6, 3, MW - 8 - 4, MH + 4 - 2
		y := py + len(Wrap(fmt.Sprintf("{{K|Pick up to %d ingredients (letters / Space), Enter to cook, Esc to leave.}}", g.maxIngredients()), pw)) + 1
		for _, it := range g.cookCandidates() {
			if y > ph-6 {
				break
			}
			out = append(out, icRowOf(g, it, y, px+4, px+pw))
			y++
		}
	case "hmd":
		st := hmSt(g)
		if st.h == nil {
			return nil
		}
		const px, py, pw, ph = 6, 3, MW - 8 - 4, MH + 4 - 2
		half := pw / 2
		cands := hmCands(g)
		rows := ph - 10
		sel := g.Sel
		if g.Tab == 1 {
			sel = g.Sel2
		}
		top := maxi(0, sel-rows+1)
		for i := top; i < len(cands) && i-top < rows; i++ {
			out = append(out, icExhibit(g, cands[i].Kind, cands[i].Title, cands[i].Items, py+2+i-top, px, px+half-2))
		}
		for i, c := range st.h.Cases {
			if c != nil {
				out = append(out, icExhibit(g, c.Kind, Plain(c.Title), c.Items, py+2+i, px+half, px+pw))
			}
		}
	}
	return out
}

func icExhibit(g *Game, kind, title string, items []*Item, y, x0, x1 int) IconRow {
	if len(items) > 0 && items[0] != nil && kind != hmKSet {
		r := icRowOf(g, items[0], y, x0, x1)
		r.Name = Plain(title)
		return r
	}
	k := map[string]string{hmKSet: "curio", hmKGear: "sword", hmKHead: "bones", hmKRelic: "relic", hmKGrave: "ring",
		hmKUnique: "relic", hmKFossil: "fossil", hmKFind: "gem"}[kind]
	if k == "" {
		k = "curio"
	}
	rar := map[string]int{hmKRelic: 4, hmKUnique: 5, hmKFind: 3, hmKGear: 3}[kind]
	return IconRow{Y: y, X0: x0, X1: x1, Name: Plain(title), Icon: IconSpec{Kind: k, Rar: rar, Hash: icHash(title), Mat: map[string]string{"fossil": "stone", "bones": "bone", "ring": "gold"}[k]}}
}

// StatusIcons lists the player's active conditions, then weather and time of day.
func (g *Game) StatusIcons() []IconSpec {
	if g.P == nil {
		return nil
	}
	var out []IconSpec
	add := func(kind, tag, name string, lv int) {
		out = append(out, IconSpec{Kind: kind, Tag: tag, Name: name, Lv: lv, Hash: icHash(kind + tag + name)})
	}
	switch {
	case g.Hunger > 2400:
		add("hunger", "", "Famished", 2)
	case g.Hunger > 1200:
		add("hunger", "", "Hungry", 1)
	}
	switch {
	case g.Thirst > 1800:
		add("thirst", "", "Parched", 2)
	case g.Thirst > 900:
		add("thirst", "", "Thirsty", 1)
	}
	if m := g.Meal(); m != "" {
		add("meal", "", "Well fed: "+m, 1)
	}
	var keys []string
	for k, v := range g.P.Status {
		if v > 0 {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	for _, k := range keys {
		add("status", icStatusClass(k), k, g.P.Status[k])
	}
	if g.Cur != nil {
		for _, in := range g.ijSt().Inj {
			add("injury", in.Part, in.Name, 1)
		}
	}
	if n := len(g.mu().Lv); n > 0 {
		add("mutation", "", "Mutations", n)
	}
	if s := g.cs().Stance; s != "" {
		add("stance", s, map[string]string{"guard": "Guard", "aggr": "Aggressive", "evade": "Evasive", "aim": "Steady aim"}[s], 1)
	}
	if st := g.ns(); st.Noise >= nsNoiseLoud {
		add("noise", "", "Noisy", 1)
	}
	if n, _ := g.nsScentCalc(); n >= 8 {
		add("scent", "", "Strong scent", 1)
	}
	if fx := g.FxInfo(); fx != nil {
		if fx.Wx != "" {
			add("weather", fx.Wx, fx.Wx, 1)
		}
		ph := "day"
		switch {
		case fx.Daylight < 0.15:
			ph = "night"
		case fx.Daylight < 0.6 && fx.Phase < 0.5:
			ph = "dawn"
		case fx.Daylight < 0.6:
			ph = "dusk"
		}
		if !fx.Indoors && !fx.Dark {
			add("time", ph, ph, 1)
		}
	}
	return out
}

func icStatusClass(id string) string {
	l := strings.ToLower(id)
	switch {
	case icHas(l, "poison", "venom", "toxin"):
		return "poison"
	case icHas(l, "bleed"):
		return "bleed"
	case icHas(l, "burn", "fire", "scald", "heatstroke", "ember"):
		return "burn"
	case icHas(l, "bless", "blessed", "prayer", "holy"):
		return "blessed"
	case icHas(l, "sleep", "asleep", "tired", "ij_", "drows", "fatigue"):
		return "sleepy"
	case icHas(l, "ill", "sick", "hz", "fever", "plague", "rad"):
		return "sick"
	case icHas(l, "frost", "frozen", "cold", "chill"):
		return "frost"
	case icHas(l, "stun", "daze", "stagger", "confus", "off-balance"):
		return "dazed"
	case icHas(l, "haste", "sprint", "swift", "quick"):
		return "haste"
	case icHas(l, "wind", "altsick", "breath", "drown"):
		return "air"
	case icHas(l, "fed", "meal", "well"):
		return "meal"
	}
	return "other"
}

// IconStatus is the strip placed on the sidebar's hunger row (row 9, right of "Sated Quenched").
func (g *Game) IconStatusInfo() *IconStatus {
	switch g.Mode {
	case "main", "world", "target", "look", "dir":
	default:
		return nil // full-screen boxes cover the sidebar
	}
	ic := g.StatusIcons()
	if len(ic) == 0 {
		return nil
	}
	return &IconStatus{Y: 9, X0: MW + 1 + 17, Icons: ic}
}

// IconRowsJSON / IconStatusJSON: JSON strings for the wasm bridge ("" when there is nothing to show).
func (g *Game) IconRowsJSON() string {
	rows := g.IconRows()
	if len(rows) == 0 {
		return ""
	}
	b, err := json.Marshal(rows)
	if err != nil {
		return ""
	}
	return string(b)
}

func (g *Game) IconStatusJSON() string {
	st := g.IconStatusInfo()
	if st == nil {
		return ""
	}
	b, err := json.Marshal(st)
	if err != nil {
		return ""
	}
	return string(b)
}
