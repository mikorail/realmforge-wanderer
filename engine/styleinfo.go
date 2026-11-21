package engine

import (
	"fmt"
	"math"
	"sort"
	"strings"
)

// Look-and-feel data for the shell's culture tiles and spell effects (prefix sy): the architectural style of the
// zone's town or culture (materials, tints, roof, ornament), the origins of the last key's hit events, and the
// status auras and item glows in view. Read-only and cosmetic: no g.R, no state is changed.

// StyleLook is the building style of the current zone, derived from the culture pack when there is one and from the
// culture, sphere and genre otherwise.
type StyleLook struct {
	Culture   int      `json:"culture"`          // culture id, -1 when unknown
	Sphere    string   `json:"sphere,omitempty"` // culture sphere
	Genre     string   `json:"genre,omitempty"`  // world genre
	Town      bool     `json:"town"`             // the zone is a town (the shell may limit tiles to these)
	Materials []string `json:"materials"`        // primary, secondary: stone timber adobe brick marble bone ironwork bamboo thatch glass obsidian coral ice hide hedge
	Accent    string   `json:"accent"`           // hex
	WallTint  string   `json:"wallTint"`         // hex, base wall colour
	RoofTint  string   `json:"roofTint"`         // hex, base roof colour
	Roof      string   `json:"roof"`             // tile thatch slate dome pagoda flat sod sails
	Ornament  string   `json:"ornament"`         // lintel banner lantern totem gargoyle mural gear neon vine
	Pack      bool     `json:"pack,omitempty"`   // the words came from a culture pack
	Seed      int      `json:"seed"`             // stable per-culture number for the shell's hashes
}

// StatusFx is one aura or glow to draw: Kind is burning poisoned bleeding frozen blessed cursed shielded hasted
// slowed stunned asleep, or glow-uncommon glow-rare glow-legendary (ID 0 for an item on the floor, the actor id for
// a wielded one).
type StatusFx struct {
	ID   int    `json:"id"`
	X    int    `json:"x"`
	Y    int    `json:"y"`
	Kind string `json:"kind"`
}

var syWallMat = map[Terrain]string{
	TWoodWall: "timber", TBrickWall: "brick", TStoneWall: "stone", TMarbleWall: "marble", TObsidianWall: "obsidian",
	TBoneWall: "bone", TAdobeWall: "adobe", TBambooWall: "bamboo", TCoralWall: "coral", TMetalWall: "ironwork",
	THedgeWall: "hedge", TIceWall: "ice", TTent: "hide", TWall: "stone",
}

var syFloorMat = map[Terrain]string{TWoodFloor: "timber", TTileFloor: "marble", TFloor: "stone", TCarpet: "hide"}

type syWord struct {
	key  string
	vals []string
}

var syMatWords = []syWord{
	{"marble", []string{"marble", "alabaster", "white stone"}},
	{"obsidian", []string{"obsidian", "basalt", "black stone"}},
	{"bone", []string{"bone", "hide", "skull", "tusk", "antler"}},
	{"hide", []string{"felt", "canvas", "leather", "tent", "yurt", "skin"}},
	{"adobe", []string{"adobe", "mudbrick", "mud brick", "mud-brick", "rammed", "clay", "plaster", "whitewash", "cob"}},
	{"brick", []string{"brick", "kiln"}},
	{"bamboo", []string{"bamboo", "reed", "paper", "lattice", "rattan"}},
	{"thatch", []string{"thatch", "straw", "turf", "sod", "wattle"}},
	{"ironwork", []string{"iron", "steel", "brass", "copper", "bronze", "gear", "metal", "rivet"}},
	{"glass", []string{"glass", "crystal", "chrome"}},
	{"coral", []string{"coral", "shell", "pearl"}},
	{"ice", []string{"ice", "snow", "frost"}},
	{"hedge", []string{"living", "grown", "hedge", "root", "vine", "tree"}},
	{"timber", []string{"timber", "wood", "log", "plank", "oak", "pine", "cedar", "beam", "half-timber"}},
	{"stone", []string{"stone", "granite", "limestone", "slate", "ashlar", "masonry", "rock", "flint", "sandstone"}},
}

var syRoofWords = []syWord{
	{"pagoda", []string{"pagoda", "upswept", "upturned", "curved eave", "swept eave"}},
	{"dome", []string{"dome", "cupola", "onion", "vault"}},
	{"sails", []string{"sail", "canvas", "awning", "felt", "tent", "yurt", "banner-roof"}},
	{"thatch", []string{"thatch", "straw", "reed"}},
	{"sod", []string{"sod", "turf", "grass roof", "green roof", "earth roof"}},
	{"slate", []string{"slate", "shingle", "lead", "steep"}},
	{"flat", []string{"flat", "terrace", "rooftop", "parapet"}},
	{"tile", []string{"tile", "terracotta", "pantile", "clay"}},
}

var syOrnWords = []syWord{
	{"neon", []string{"neon", "holo", "led", "glow"}},
	{"gear", []string{"gear", "cog", "clockwork", "piston", "brass"}},
	{"gargoyle", []string{"gargoyle", "grotesque", "spire", "buttress"}},
	{"totem", []string{"totem", "pole", "idol", "skull", "tusk"}},
	{"lantern", []string{"lantern", "lamp", "lamplit", "torch"}},
	{"banner", []string{"banner", "flag", "pennant", "standard", "heraldic", "heraldry", "crest"}},
	{"mural", []string{"mural", "fresco", "painted", "mosaic", "tile-work", "tilework", "calligraph", "glaz"}},
	{"vine", []string{"vine", "ivy", "moss", "living", "garden", "flower", "bloom", "creeper"}},
	{"lintel", []string{"lintel", "carv", "relief", "rune", "column", "frieze", "arch", "sculpt"}},
}

var syColourWords = []struct{ w, hex string }{
	{"whitewash", "#e8e2d0"}, {"white", "#e6e2d6"}, {"lime", "#e4dfc8"}, {"cream", "#e8dcb8"}, {"ivory", "#e6dcc0"},
	{"crimson", "#a8283a"}, {"scarlet", "#c0332a"}, {"vermilion", "#c8452f"}, {"red", "#b4452f"}, {"rust", "#a5522d"},
	{"terracotta", "#c8693a"}, {"orange", "#d27a32"}, {"ochre", "#c9973f"}, {"saffron", "#e0a82e"}, {"golden", "#d6a640"}, {"gold", "#d6a640"}, {"yellow", "#d8c050"},
	{"jade", "#4d9a73"}, {"green", "#4d8a53"}, {"teal", "#3b8a8a"}, {"turquoise", "#38a8a0"},
	{"indigo", "#3b4f94"}, {"blue", "#3e6aa0"}, {"azure", "#3e8ac0"},
	{"violet", "#6e4a9e"}, {"purple", "#7a4a8e"}, {"lilac", "#9a7ab8"}, {"pink", "#d98fa3"}, {"rose", "#c9708a"},
	{"black", "#34303a"}, {"obsidian", "#2e2a34"}, {"charcoal", "#43424a"}, {"grey", "#7b7f86"}, {"gray", "#7b7f86"}, {"slate", "#5d6670"},
	{"silver", "#b4bac4"}, {"bronze", "#a8783a"}, {"copper", "#b4683a"}, {"brass", "#b89a3a"},
	{"umber", "#6c4a30"}, {"brown", "#7a5434"}, {"timber", "#7a5434"}, {"sand", "#cdb98a"}, {"tan", "#c8aa78"}, {"bone", "#d8d0b8"},
}

func syFirstWord(text string, set []syWord) string {
	best, bi := "", len(text)+1
	for _, e := range set {
		for _, v := range e.vals {
			if i := strings.Index(text, v); i >= 0 && i < bi {
				best, bi = e.key, i
			}
		}
	}
	return best
}

// syAllWords returns every key of set found in text, in order of first appearance.
func syAllWords(text string, set []syWord) []string {
	type hit struct {
		k string
		i int
	}
	var hs []hit
	for _, e := range set {
		bi := -1
		for _, v := range e.vals {
			if i := strings.Index(text, v); i >= 0 && (bi < 0 || i < bi) {
				bi = i
			}
		}
		if bi >= 0 {
			hs = append(hs, hit{e.key, bi})
		}
	}
	sort.SliceStable(hs, func(i, j int) bool { return hs[i].i < hs[j].i })
	out := make([]string, len(hs))
	for i, h := range hs {
		out[i] = h.k
	}
	return out
}

func syColours(text string) []string {
	type hit struct {
		hex string
		i   int
	}
	var hs []hit
	taken := map[string]bool{}
	for _, c := range syColourWords {
		i := strings.Index(text, c.w)
		if i < 0 || taken[c.hex] {
			continue
		}
		// a longer word covering the same place wins (whitewash over white): skip when an earlier hit starts here
		dup := false
		for _, h := range hs {
			if h.i == i {
				dup = true
			}
		}
		if dup {
			continue
		}
		taken[c.hex] = true
		hs = append(hs, hit{c.hex, i})
	}
	sort.SliceStable(hs, func(i, j int) bool { return hs[i].i < hs[j].i })
	out := make([]string, len(hs))
	for i, h := range hs {
		out[i] = h.hex
	}
	return out
}

func syHex(r, g, b float64) string {
	cl := func(v float64) int { return int(math.Max(0, math.Min(255, math.Round(v*255)))) }
	return fmt.Sprintf("#%02x%02x%02x", cl(r), cl(g), cl(b))
}

// syHSL converts h (0..360), s and l (0..1) to a hex colour.
func syHSL(h, s, l float64) string {
	c := (1 - math.Abs(2*l-1)) * s
	hp := math.Mod(h, 360) / 60
	x := c * (1 - math.Abs(math.Mod(hp, 2)-1))
	var r, g, b float64
	switch {
	case hp < 1:
		r, g, b = c, x, 0
	case hp < 2:
		r, g, b = x, c, 0
	case hp < 3:
		r, g, b = 0, c, x
	case hp < 4:
		r, g, b = 0, x, c
	case hp < 5:
		r, g, b = x, 0, c
	default:
		r, g, b = c, 0, x
	}
	m := l - c/2
	return syHex(r+m, g+m, b+m)
}

func syMatDefault(sphere, genre string) (string, string) {
	switch {
	case strings.Contains(genre, "cyber"):
		return "glass", "ironwork"
	case strings.Contains(genre, "steam") || strings.Contains(genre, "diesel"):
		return "brick", "ironwork"
	}
	return "", ""
}

func syRoofFor(mat, sphere, genre string) string {
	switch {
	case strings.Contains(genre, "cyber") || strings.Contains(genre, "atom") || strings.Contains(genre, "diesel"):
		return "flat"
	case strings.Contains(sphere, "sinic") || strings.Contains(sphere, "japonic") || strings.Contains(sphere, "asian"):
		return "pagoda"
	case strings.Contains(sphere, "persianate") || strings.Contains(sphere, "semitic"):
		return "dome"
	}
	switch mat {
	case "adobe", "marble":
		if strings.Contains(sphere, "hellen") || strings.Contains(sphere, "roman") || strings.Contains(sphere, "romance") {
			return "tile"
		}
		return "flat"
	case "hide":
		return "sails"
	case "bone", "hedge":
		return "sod"
	case "timber", "bamboo":
		return "thatch"
	case "stone", "obsidian", "ironwork":
		return "slate"
	case "brick":
		return "tile"
	case "coral":
		return "sails"
	}
	return "tile"
}

func syOrnFor(mat, sphere, genre string, seed int) string {
	switch {
	case strings.Contains(genre, "cyber"):
		return "neon"
	case strings.Contains(genre, "steam") || strings.Contains(genre, "clock"):
		return "gear"
	case strings.Contains(genre, "dark") || strings.Contains(genre, "gothic") || strings.Contains(genre, "horror"):
		return "gargoyle"
	}
	switch mat {
	case "bone", "hide":
		return "totem"
	case "bamboo":
		return "lantern"
	case "adobe":
		return "mural"
	case "hedge", "coral":
		return "vine"
	case "marble":
		return "lintel"
	case "stone":
		return []string{"banner", "gargoyle", "lintel"}[seed%3]
	case "timber":
		return []string{"banner", "lintel", "totem"}[seed%3]
	}
	return "banner"
}

// StyleInfo is the building style of the zone the player is in; nil on the world map.
func (g *Game) StyleInfo() *StyleLook {
	if g == nil || g.Cur == nil || g.P == nil || g.W == nil || g.OnWorld || g.Mode == "world" {
		return nil
	}
	z := g.Cur
	cu := g.lxCultureAt()
	var bd *BurgData
	if z.Kind == "town" {
		bd = g.W.burgByID[z.Burg]
	}
	genre := strings.ToLower(g.Genre)
	s := &StyleLook{Culture: -1, Genre: genre, Town: z.Kind == "town"}
	var bs *BuildingStyle
	sphere := ""
	if cu != nil {
		s.Culture, s.Sphere = cu.ID, cu.Sphere
		sphere = strings.ToLower(cu.Sphere)
		if cu.Pack != nil && cu.Pack.Material != nil {
			bs = cu.Pack.Material.Buildings
		}
	}
	s.Seed = (s.Culture + 1) * 7919

	wallT, floorT := g.acMaterials(bd, cu, "")
	primary, secondary := syWallMat[wallT], syFloorMat[floorT]
	if primary == "" {
		primary = "stone"
	}
	if d1, d2 := syMatDefault(sphere, genre); d1 != "" {
		primary, secondary = d1, d2
	}

	text := ""
	if bs != nil {
		text = strings.ToLower(strings.Join(bs.Materials, " ") + " " + bs.House + " " + bs.Hall)
		s.Pack = true
		if ms := syAllWords(strings.ToLower(strings.Join(bs.Materials, " ")), syMatWords); len(ms) > 0 {
			primary = ms[0]
			secondary = ""
			if len(ms) > 1 {
				secondary = ms[1]
			}
		} else if ms := syAllWords(text, syMatWords); len(ms) > 0 {
			primary = ms[0]
			if len(ms) > 1 {
				secondary = ms[1]
			}
		}
	} else if cu != nil && cu.Architecture != "" {
		text = strings.ToLower(cu.Architecture)
		if ms := syAllWords(text, syMatWords); len(ms) > 0 {
			primary = ms[0]
			if len(ms) > 1 {
				secondary = ms[1]
			}
		}
	}
	if secondary == "" || secondary == primary {
		secondary = syFloorMat[floorT]
		if secondary == primary || secondary == "" {
			secondary = "timber"
			if primary == "timber" {
				secondary = "stone"
			}
		}
	}
	s.Materials = []string{primary, secondary}

	// roof and ornament
	roofText, ornText := text, text
	if bs != nil {
		roofText = strings.ToLower(bs.Roof + " " + bs.House + " " + bs.Hall)
		ornText = strings.ToLower(bs.Ornament + " " + bs.Gate + " " + bs.Temple)
	} else if cu != nil {
		roofText, ornText = strings.ToLower(cu.Architecture), strings.ToLower(cu.Architecture)
	}
	s.Roof = syFirstWord(roofText, syRoofWords)
	if s.Roof == "" {
		s.Roof = syRoofFor(primary, sphere, genre)
	}
	s.Ornament = syFirstWord(ornText, syOrnWords)
	if s.Ornament == "" {
		s.Ornament = syOrnFor(primary, sphere, genre, int(uint(s.Seed)/7919))
	}

	// colours
	var cols []string
	if bs != nil {
		cols = syColours(strings.ToLower(bs.Color))
	}
	if len(cols) == 0 && cu != nil {
		cols = syColours(strings.ToLower(cu.Dress + " " + cu.Architecture))
	}
	hue := float64(HashStr(fmt.Sprintf("sy:%d:%s", s.Culture, sphere)) % 360)
	base := map[string]string{
		"stone": "#8a8a90", "timber": "#7a5434", "adobe": "#c8985a", "brick": "#a4522f", "marble": "#e0dcd0", "bone": "#cfc6a8",
		"ironwork": "#6a6e76", "bamboo": "#9aa650", "thatch": "#b09a52", "glass": "#7ab4c8", "obsidian": "#3a3640", "coral": "#d89a8a",
		"ice": "#b8dcec", "hide": "#b49a72", "hedge": "#4f7a3c",
	}
	s.WallTint = base[primary]
	if s.WallTint == "" {
		s.WallTint = syHSL(hue, 0.25, 0.5)
	}
	roofBase := map[string]string{
		"tile": "#b4552f", "thatch": "#a8894a", "slate": "#4d525c", "dome": "#3e78a8", "pagoda": "#3a6a5a", "flat": "#9a8a70",
		"sod": "#4f7a3c", "sails": "#d8cfb4",
	}
	s.RoofTint = roofBase[s.Roof]
	s.Accent = syHSL(hue, 0.55, 0.52)
	if len(cols) > 0 {
		s.WallTint = cols[0]
		if len(cols) > 1 {
			s.RoofTint = cols[1]
		}
		if len(cols) > 2 {
			s.Accent = cols[2]
		} else if len(cols) > 1 {
			s.Accent = cols[1]
		}
	}
	if strings.Contains(genre, "cyber") {
		s.Accent = []string{"#00e5ff", "#ff2bd6", "#9dff3a"}[int(HashStr(fmt.Sprintf("syn:%d", s.Culture))%3)]
	}
	return s
}

// HitOrigins maps actor ids to their cell at the start of the last key (from the combat-feel snapshot), falling
// back to where they stand now; the shell draws ranged streaks from the attacker's origin to the event cell.
func (g *Game) HitOrigins() map[int][2]int {
	if g == nil || g.Cur == nil || g.P == nil || g.OnWorld {
		return nil
	}
	out := map[int][2]int{}
	for _, a := range g.Cur.Actors {
		if a != nil && a.HP > 0 {
			out[a.ID] = [2]int{a.X, a.Y}
		}
	}
	out[g.P.ID] = [2]int{g.P.X, g.P.Y}
	if st, ok := g.ext("hitfx", func() any { return &hxState{} }).(*hxState); ok && st.s.valid && g.Cur.Key == st.s.zone {
		for _, a := range st.s.list {
			out[a.id] = [2]int{a.x, a.y}
		}
	}
	return out
}

// HitOriginsJS is HitOrigins as a JS-friendly map keyed by id string: {"12":[x,y]}. Only ids in ids are kept
// (pass the attackerIds of the hit events) to keep the frame small; nil ids keeps all.
func HitOriginsJS(m map[int][2]int, ids []int) map[string]any {
	out := map[string]any{}
	if ids == nil {
		for k, v := range m {
			out[fmt.Sprint(k)] = []any{v[0], v[1]}
		}
		return out
	}
	for _, k := range ids {
		if v, ok := m[k]; ok {
			out[fmt.Sprint(k)] = []any{v[0], v[1]}
		}
	}
	return out
}

var syStatusKinds = []struct{ st, kind string }{
	{"burning", "burning"}, {"poisoned", "poisoned"}, {"bleeding", "bleeding"}, {"frozen", "frozen"},
	{"rtblessed", "blessed"}, {"blessed", "blessed"}, {"cursed", "cursed"}, {"curse", "cursed"},
	{"shielded", "shielded"}, {"hasted", "hasted"}, {"slowed", "slowed"}, {"stunned", "stunned"},
	{"asleep", "asleep"}, {"sleeping", "asleep"},
}

func syGlow(it *Item) string {
	if it == nil {
		return ""
	}
	switch it.Rarity {
	case "uncommon":
		return "glow-uncommon"
	case "rare":
		return "glow-rare"
	case "ultrarare", "legendary", "mythic", "unique":
		return "glow-legendary"
	}
	if it.Cat != "" || (it.Kind == "artifact" && it.Ident) {
		return "glow-rare"
	}
	return ""
}

// StatusFx lists the status auras of visible actors and the glows of magic items on the floor and wielded, at most
// 60 entries, nearest to the player first.
func (g *Game) StatusFx() []StatusFx {
	if g == nil || g.Cur == nil || g.P == nil || g.OnWorld || g.Mode == "world" {
		return nil
	}
	var out []StatusFx
	add := func(a *Actor) {
		seen := map[string]bool{}
		for _, sk := range syStatusKinds {
			if a.St(sk.st) > 0 && !seen[sk.kind] {
				seen[sk.kind] = true
				out = append(out, StatusFx{ID: a.ID, X: a.X, Y: a.Y, Kind: sk.kind})
			}
		}
		best := ""
		for _, bp := range a.Body {
			if bp == nil || bp.Severed || bp.Item == nil || (bp.Slot != "hand" && bp.Slot != "hands") {
				continue
			}
			if k := syGlow(bp.Item); k > best {
				best = k
			}
		}
		if best != "" {
			out = append(out, StatusFx{ID: a.ID, X: a.X, Y: a.Y, Kind: best})
		}
	}
	add(g.P)
	for _, a := range g.Cur.Actors {
		if a != nil && a != g.P && a.HP > 0 && g.Cur.In(a.X, a.Y) && g.visible(a.X, a.Y) {
			add(a)
		}
	}
	for _, it := range g.Cur.Items {
		if k := syGlow(it); k != "" && g.Cur.In(it.X, it.Y) && g.visible(it.X, it.Y) {
			out = append(out, StatusFx{X: it.X, Y: it.Y, Kind: k})
		}
	}
	p := g.P
	d := func(f StatusFx) int { return (f.X-p.X)*(f.X-p.X) + (f.Y-p.Y)*(f.Y-p.Y) }
	sort.SliceStable(out, func(i, j int) bool { return d(out[i]) < d(out[j]) })
	if len(out) > 60 {
		out = out[:60]
	}
	return out
}

// StatusFxJS is the frame form of StatusFx.
func StatusFxJS(l []StatusFx) []any {
	out := make([]any, len(l))
	for i, f := range l {
		out[i] = map[string]any{"id": f.ID, "x": f.X, "y": f.Y, "kind": f.Kind}
	}
	return out
}

// StyleJS is the frame form of StyleLook.
func StyleJS(s *StyleLook) any {
	if s == nil {
		return nil
	}
	ms := make([]any, len(s.Materials))
	for i, m := range s.Materials {
		ms[i] = m
	}
	return map[string]any{"culture": s.Culture, "sphere": s.Sphere, "genre": s.Genre, "town": s.Town, "materials": ms,
		"accent": s.Accent, "wallTint": s.WallTint, "roofTint": s.RoofTint, "roof": s.Roof, "ornament": s.Ornament, "seed": s.Seed}
}
