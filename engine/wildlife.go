package engine

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// The bestiary of Earth (prefix bs): living animals and extinct ones across deep time.
// The catalogue is a data table (bestiary_*.go, one species per line) parsed at init into
// bsSp records and CTpl templates registered in tplByID under "bs_<slug>" (never appended
// to Templates, so pickCreature and every existing spawn table stay exactly as they were).
// Placement uses side rngs only (seed:bs:...): wildlife in wild zones, rare lost worlds
// (plateaus, hidden valleys, primordial caverns, abyssal trenches), tide-pool curiosities
// on sea floors and fossils in strata and ruins. This file's name sorts after every other
// zone-hook file so the hooks before it see the same zone they always did.

// bsSp is one species of the catalogue.
type bsSp struct {
	ID     string  // slug; template id is "bs_"+ID
	Name   string  // common name
	Group  string  // code: mam bird rept dino ptero amph fish ins arach crust myria arth moll cnid echi worm misc
	Era    string  // living recent pleistocene cenozoic mesozoic paleozoic precambrian
	Period string  // "Late Cretaceous" (prehistoric) or "" (living)
	Ma     float64 // millions of years ago (prehistoric)
	Gone   string  // year of extinction for recently extinct species ("1662", "c. 1400")
	Biomes []int   // land biomes it lives in (1..16)
	Sea    int     // 0 none, 1 any sea, 2 warm seas, 3 cold seas (open sea surface)
	Reef   int     // same for the sea floor
	Fresh  bool    // rivers and lakes
	Coast  bool    // shores of the sea
	Cave   bool    // caves and the deep
	Sky    bool    // sky isles
	Mtn    bool    // only on high ground (height >= 55)
	Ice    bool    // only where it is freezing (temp <= 2)
	Size   int     // 1 tiny .. 7 colossal
	Diet   byte    // h herbivore c carnivore o omnivore i insectivore f filter p piscivore s scavenger
	Temper string  // passive skittish herd territorial predator pack
	Act    byte    // d diurnal n nocturnal c crepuscular a any
	Flags  string  // actor flags (aquatic fly burrow poison bleed shock acid regen armored ivory)
	Body   string  // body plan
	Glyph  rune
	F      byte
	Rarity int    // 1 common .. 5 legendary
	Attack string // natural weapon name
	Desc   string
}

// bsData holds the raw catalogue blocks; each bestiary_*.go file registers its own with bsReg.
var bsData []string

func bsReg(block string) bool { bsData = append(bsData, block); return true }

var (
	bsAll   []*bsSp
	bsByID  = map[string]*bsSp{}
	bsByTpl = map[string]*bsSp{}
	bsTpls  []CTpl
)

var bsGroupNames = map[string]string{
	"mam": "mammal", "bird": "bird", "rept": "reptile", "dino": "dinosaur", "ptero": "pterosaur", "amph": "amphibian",
	"fish": "fish", "ins": "insect", "arach": "arachnid", "crust": "crustacean", "myria": "myriapod", "arth": "arthropod",
	"moll": "mollusc", "cnid": "cnidarian", "echi": "echinoderm", "worm": "worm", "misc": "enigmatic organism",
}

var bsTempers = map[string]bool{"passive": true, "skittish": true, "herd": true, "territorial": true, "predator": true, "pack": true}

var bsFlagOK = map[string]bool{"aquatic": true, "fly": true, "burrow": true, "poison": true, "bleed": true, "shock": true, "acid": true,
	"regen": true, "armored": true, "ivory": true, "pack": true}

var bsBodies = map[string]bool{"quadruped": true, "insect": true, "serpent": true, "avian": true, "fish": true, "ooze": true, "golem": true, "tentacled": true, "humanoid": true}

var bsSizeNames = []string{"", "tiny", "small", "medium", "large", "huge", "giant", "colossal"}

var bsEraNames = map[string]string{"living": "Living", "recent": "Recently extinct", "pleistocene": "Pleistocene megafauna",
	"cenozoic": "Cenozoic", "mesozoic": "Mesozoic", "paleozoic": "Paleozoic", "precambrian": "Precambrian"}

// bsEraOf maps a geological period name onto the bestiary's eras.
func bsEraOf(period string) string {
	p := strings.ToLower(period)
	switch {
	case strings.Contains(p, "pleistocene"):
		return "pleistocene"
	case strings.Contains(p, "holocene"):
		return "recent"
	case strings.Contains(p, "pliocene"), strings.Contains(p, "miocene"), strings.Contains(p, "oligocene"), strings.Contains(p, "eocene"), strings.Contains(p, "paleocene"):
		return "cenozoic"
	case strings.Contains(p, "cretaceous"), strings.Contains(p, "jurassic"), strings.Contains(p, "triassic"):
		return "mesozoic"
	case strings.Contains(p, "cambrian") && !strings.Contains(p, "pre"), strings.Contains(p, "ordovician"), strings.Contains(p, "silurian"),
		strings.Contains(p, "devonian"), strings.Contains(p, "carboniferous"), strings.Contains(p, "permian"):
		return "paleozoic"
	case strings.Contains(p, "ediacaran"), strings.Contains(p, "cryogenian"), strings.Contains(p, "precambrian"):
		return "precambrian"
	}
	return ""
}

func bsSlug(s string) string {
	var b strings.Builder
	us := false
	for _, r := range strings.ToLower(s) {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			b.WriteRune(r)
			us = false
		} else if !us && b.Len() > 0 {
			b.WriteByte('_')
			us = true
		}
	}
	return strings.TrimSuffix(b.String(), "_")
}

var bsHabGroups = map[string][]int{
	"desert": {1, 2}, "open": {3, 4}, "forest": {5, 6, 7, 8, 9}, "trop": {5, 7}, "temp": {6, 8}, "cold": {9, 10, 11},
	"any": {1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 12},
}

// bsParse reads one catalogue line: name|grp|when|hab|size|diet|temper|act|flags|gc|rar|attack|desc
func bsParse(line string) (*bsSp, error) {
	f := strings.Split(line, "|")
	if len(f) != 13 {
		return nil, fmt.Errorf("want 13 fields, got %d", len(f))
	}
	for i := range f {
		f[i] = strings.TrimSpace(f[i])
	}
	s := &bsSp{Name: f[0], Group: f[1], Temper: f[6], Attack: f[11], Desc: f[12]}
	s.ID = bsSlug(s.Name)
	if _, ok := bsGroupNames[s.Group]; !ok {
		return nil, fmt.Errorf("bad group %q", s.Group)
	}
	switch w := f[2]; {
	case w == "L":
		s.Era = "living"
	case strings.HasPrefix(w, "X"):
		s.Era, s.Gone = "recent", strings.TrimSpace(w[1:])
	default:
		i := strings.LastIndex(w, ":")
		if i < 0 {
			return nil, fmt.Errorf("bad period %q", w)
		}
		s.Period = w[:i]
		ma, err := strconv.ParseFloat(w[i+1:], 64)
		if err != nil {
			return nil, fmt.Errorf("bad age %q", w)
		}
		s.Ma = ma
		if s.Era = bsEraOf(s.Period); s.Era == "" {
			return nil, fmt.Errorf("unknown period %q", s.Period)
		}
	}
	seen := map[int]bool{}
	addB := func(b int) {
		if !seen[b] {
			seen[b] = true
			s.Biomes = append(s.Biomes, b)
		}
	}
	for _, h := range strings.Split(f[3], ",") {
		h = strings.TrimSpace(h)
		lvl := 1
		if strings.HasSuffix(h, "w") && (strings.HasPrefix(h, "sea") || strings.HasPrefix(h, "reef")) {
			lvl, h = 2, strings.TrimSuffix(h, "w")
		} else if strings.HasSuffix(h, "c") && (strings.HasPrefix(h, "sea") || strings.HasPrefix(h, "reef")) && h != "sea" {
			lvl, h = 3, strings.TrimSuffix(h, "c")
		}
		switch h {
		case "sea":
			s.Sea = lvl
		case "reef":
			s.Reef = lvl
		case "fresh":
			s.Fresh = true
		case "coast":
			s.Coast = true
		case "cave":
			s.Cave = true
		case "sky":
			s.Sky = true
		case "mtn":
			s.Mtn = true
		case "ice":
			s.Ice = true
		case "":
		default:
			if gs, ok := bsHabGroups[h]; ok {
				for _, b := range gs {
					addB(b)
				}
				continue
			}
			b, err := strconv.Atoi(h)
			if err != nil || b < 1 || b > 16 {
				return nil, fmt.Errorf("bad habitat %q", h)
			}
			addB(b)
		}
	}
	if s.Mtn && len(s.Biomes) == 0 {
		for _, b := range bsHabGroups["any"] {
			addB(b)
		}
		addB(11)
	}
	if len(s.Biomes) == 0 && s.Sea == 0 && s.Reef == 0 && !s.Fresh && !s.Coast && !s.Cave && !s.Sky {
		return nil, fmt.Errorf("no habitat")
	}
	sz, err := strconv.Atoi(f[4])
	if err != nil || sz < 1 || sz > 7 {
		return nil, fmt.Errorf("bad size %q", f[4])
	}
	s.Size = sz
	if len(f[5]) != 1 || !strings.Contains("hcoifps", f[5]) {
		return nil, fmt.Errorf("bad diet %q", f[5])
	}
	s.Diet = f[5][0]
	if !bsTempers[s.Temper] {
		return nil, fmt.Errorf("bad temperament %q", s.Temper)
	}
	if len(f[7]) != 1 || !strings.Contains("dnca", f[7]) {
		return nil, fmt.Errorf("bad activity %q", f[7])
	}
	s.Act = f[7][0]
	var fl []string
	for _, t := range strings.Fields(f[8]) {
		if strings.HasPrefix(t, "@") {
			if !bsBodies[t[1:]] {
				return nil, fmt.Errorf("bad body %q", t)
			}
			s.Body = t[1:]
			continue
		}
		if t == "venom" {
			t = "poison"
		}
		if !bsFlagOK[t] {
			return nil, fmt.Errorf("bad flag %q", t)
		}
		fl = append(fl, t)
	}
	s.Flags = strings.Join(fl, " ")
	if s.Body == "" {
		s.Body = bsDefaultBody(s)
	}
	gc := []rune(f[9])
	if len(gc) != 2 {
		return nil, fmt.Errorf("bad glyph/colour %q", f[9])
	}
	if !(gc[0] >= 'a' && gc[0] <= 'z' || gc[0] >= 'A' && gc[0] <= 'Z') {
		return nil, fmt.Errorf("glyph must be a letter: %q", f[9])
	}
	col, ok := colorCode[byte(gc[1])]
	if !ok || gc[1] > 127 {
		return nil, fmt.Errorf("bad colour %q", f[9])
	}
	s.Glyph, s.F = gc[0], col
	if s.Rarity, err = strconv.Atoi(f[10]); err != nil || s.Rarity < 1 || s.Rarity > 5 {
		return nil, fmt.Errorf("bad rarity %q", f[10])
	}
	if s.Desc == "" || !strings.HasSuffix(s.Desc, ".") {
		return nil, fmt.Errorf("description must be a sentence")
	}
	return s, nil
}

func bsDefaultBody(s *bsSp) string {
	switch s.Group {
	case "bird", "ptero":
		return "avian"
	case "fish":
		return "fish"
	case "ins", "arach", "crust", "myria", "arth":
		return "insect"
	case "moll", "cnid", "echi", "misc":
		return "ooze"
	case "worm":
		return "serpent"
	}
	if s.Group == "mam" && (s.Sea > 0 || s.Reef > 0) && strings.Contains(s.Flags, "aquatic") {
		return "fish"
	}
	return "quadruped"
}

func (s *bsSp) Has(f string) bool { return strings.Contains(" "+s.Flags+" ", " "+f+" ") }

// Hostile species attack on sight; all others are wildlife that the player must provoke.
func (s *bsSp) Hostile() bool { return s.Temper == "predator" || s.Temper == "pack" }

func (s *bsSp) Tpl() string { return "bs_" + s.ID }

func (s *bsSp) Aquatic() bool { return s.Has("aquatic") }

func (s *bsSp) Extinct() bool { return s.Era != "living" }

// When is the period line: "Late Cretaceous, ~68 million years ago", "Extinct since 1662", "Living".
func (s *bsSp) When() string {
	switch s.Era {
	case "living":
		return "Living"
	case "recent":
		return "Extinct since " + s.Gone
	}
	return s.Period + ", ~" + bsAge(s.Ma)
}

func bsAge(ma float64) string {
	switch {
	case ma < 0.1:
		return fmt.Sprintf("%s years ago", bsThousands(int(ma*1e6+0.5)))
	case ma < 1:
		return fmt.Sprintf("%s years ago", bsThousands(int(ma*1e3+0.5)*1000))
	case ma < 10:
		return strings.TrimSuffix(strings.TrimSuffix(fmt.Sprintf("%.1f", ma), "0"), ".") + " million years ago"
	}
	return fmt.Sprintf("%d million years ago", int(ma+0.5))
}

func bsThousands(n int) string {
	s := strconv.Itoa(n)
	var out []string
	for len(s) > 3 {
		out = append([]string{s[len(s)-3:]}, out...)
		s = s[:len(s)-3]
	}
	return strings.Join(append([]string{s}, out...), ",")
}

var bsDietNames = map[byte]string{'h': "herbivore", 'c': "carnivore", 'o': "omnivore", 'i': "insectivore", 'f': "filter feeder", 'p': "fish-eater", 's': "scavenger"}

var bsTemperNames = map[string]string{"passive": "placid", "skittish": "shy, flees", "herd": "lives in herds", "territorial": "territorial",
	"predator": "dangerous predator", "pack": "hunts in packs"}

var bsActNames = map[byte]string{'d': "diurnal", 'n': "nocturnal", 'c': "crepuscular", 'a': "active day and night"}

// Lore is the encyclopedia line under a codex entry.
func (s *bsSp) Lore() string {
	return fmt.Sprintf("%s. %s %s; %s, %s, %s.", s.When(), titleCase(bsSizeNames[s.Size]), bsGroupNames[s.Group], bsDietNames[s.Diet], bsTemperNames[s.Temper], bsActNames[s.Act])
}

// bsSpecies looks a species up by common name or slug (for other systems, e.g. mounts).
func bsSpecies(name string) *bsSp {
	if s := bsByID[bsSlug(name)]; s != nil {
		return s
	}
	return bsByTpl[name]
}

func init() {
	for _, block := range bsData {
		for _, line := range strings.Split(block, "\n") {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			s, err := bsParse(line)
			if err != nil {
				panic("bestiary: " + err.Error() + ": " + line)
			}
			if bsByID[s.ID] != nil {
				panic("bestiary: duplicate species " + s.Name)
			}
			bsByID[s.ID] = s
			bsAll = append(bsAll, s)
		}
	}
	bsTpls = make([]CTpl, len(bsAll))
	for i, s := range bsAll {
		bsTpls[i] = bsTemplate(s)
		bsByTpl[bsTpls[i].ID] = s
	}
	for i := range bsTpls {
		tplByID[bsTpls[i].ID] = &bsTpls[i]
	}
	zoneHooks = append(zoneHooks, bsZone)
	killHooks = append(killHooks, bsKill)
	turnHooks = append(turnHooks, bsTurn)
	enterHooks = append(enterHooks, bsEnter)
	talkHooks = append(talkHooks, bsTalk)
}

// ---- templates ----

var (
	bsLvlT  = [8]int{0, 1, 1, 2, 4, 6, 9, 13}
	bsHPT   = [8]int{0, 3, 7, 13, 22, 38, 62, 100}
	bsStrT  = [8]int{0, 5, 8, 12, 16, 21, 26, 30}
	bsAgiT  = [8]int{0, 18, 17, 15, 14, 12, 10, 8}
	bsDiceT = [8][2]int{{0, 0}, {1, 2}, {1, 3}, {1, 4}, {1, 6}, {2, 5}, {2, 7}, {3, 7}}
	bsPVT   = [8]int{0, 0, 0, 1, 1, 2, 3, 4}
)

func bsTemplate(s *bsSp) CTpl {
	sz := s.Size
	lvl := bsLvlT[sz]
	if s.Hostile() {
		lvl++
	}
	agi := bsAgiT[sz]
	if s.Has("fly") {
		agi += 2
	}
	if s.Hostile() {
		agi += 2
	}
	str := bsStrT[sz]
	t := CTpl{ID: s.Tpl(), Glyph: s.Glyph, F: s.F, Lvl: lvl, Attr: [6]int{str, agi, str, 4 + boolInt(s.Group == "mam" || s.Group == "bird")*3, 8 + sz, 4 + sz},
		HP: bsHPT[sz], Speed: 100, Faction: "wildlife", Body: s.Body, Names: nm(s.Name), Desc: s.Desc}
	if s.Hostile() {
		t.Faction = "beasts"
		t.Speed += 10
	}
	switch s.Temper {
	case "skittish":
		t.Speed += 20 - 3*sz
	case "herd":
		t.Speed += 12 - 2*sz
	}
	if sz >= 6 {
		t.Speed -= 10
	}
	if s.Has("fly") {
		t.Speed = maxi(t.Speed, 115)
	}
	if s.Aquatic() && s.Group == "fish" {
		t.Speed += 10
	}
	t.Speed = clampi(t.Speed, 60, 150)
	switch s.Group {
	case "crust", "arth":
		t.AV = 3
	case "rept", "dino", "arach", "ins", "myria", "echi":
		t.AV = 1
	}
	if s.Has("armored") {
		t.AV += 4
	}
	if sz >= 6 {
		t.AV++
	}
	switch sz {
	case 1:
		t.DV = 4
	case 2:
		t.DV = 2
	}
	d := bsDiceT[sz]
	if s.Hostile() && d[1] < 8 {
		d[1]++
	}
	t.Bite = [3]int{d[0], d[1], bsPVT[sz]}
	var fl []string
	for _, f := range strings.Fields(s.Flags) {
		if f != "armored" && f != "ivory" {
			fl = append(fl, f)
		}
	}
	if s.Temper == "pack" && !strings.Contains(s.Flags, "pack") {
		fl = append(fl, "pack")
	}
	t.Flags = strings.Join(fl, " ")
	switch {
	case s.Has("shock"):
		t.Elem = "shock"
	case s.Has("acid"):
		t.Elem = "acid"
	case s.Has("poison"):
		t.Elem = "poison"
	}
	where := ""
	if len(s.Biomes) > 0 || s.Fresh || s.Coast {
		where += "s"
	}
	if s.Cave {
		where += "d"
	}
	if s.Sky {
		where += "k"
	}
	if s.Reef > 0 {
		where += "e"
	}
	if s.Sea > 0 && !strings.Contains(where, "s") {
		where += "s"
	}
	t.Where = where
	if len(s.Biomes) > 0 {
		t.Biomes = append([]int{}, s.Biomes...)
	}
	return t
}

// bsSpawn makes one animal of a species: Spawn plus its own natural weapon and element.
func (g *Game) bsSpawn(s *bsSp, lvl int, r *Rng) *Actor {
	t := tplByID[s.Tpl()]
	a := g.Spawn(t.ID, lvl, r)
	for _, p := range a.Body {
		if p.Natural != nil {
			p.Natural.Elem = t.Elem
			if s.Attack != "" && p.Slot == "head" || s.Attack != "" && p == bsFirstNatural(a) {
				p.Natural.Name = s.Attack
			}
		}
	}
	if s.Hostile() {
		a.AI = "hostile"
	} else {
		a.AI = "wander"
		a.XP = maxi(4, a.XP/3)
	}
	return a
}

func bsFirstNatural(a *Actor) *BodyPart {
	for _, p := range a.Body {
		if p.Natural != nil {
			return p
		}
	}
	return nil
}

// ---- spawning ----

var bsRarW = [6]float64{0, 10, 5, 2, 0.7, 0.25}

var bsPunk = map[string]bool{"steampunk": true, "dieselpunk": true, "atompunk": true, "cyberpunk": true, "biopunk": true, "solarpunk": true, "wasteland": true}

// bsIceAge: genres where Pleistocene megafauna still roam the cold steppes.
var bsIceAge = map[string]bool{"": true, "fantasy": true, "mythic": true, "wasteland": true, "twilight": true}

type bsPlace struct {
	where   byte // s surface, e sea floor, d deep, k sky, o open sea
	biome   int
	height  int
	temp    int
	coast   bool
	water   int // deep-water tiles in the zone
	genre   string
	inLost  string
	surface bool
}

func (s *bsSp) fits(pl *bsPlace) bool {
	if s.Ice && pl.temp > 2 {
		return false
	}
	seaOK := func(v int) bool {
		switch v {
		case 1:
			return true
		case 2:
			return pl.temp >= 14
		case 3:
			return pl.temp <= 12
		}
		return false
	}
	switch pl.where {
	case 'o':
		return seaOK(s.Sea) && (s.Aquatic() || s.Has("fly")) || s.Coast && s.Has("fly")
	case 'e':
		return seaOK(s.Reef)
	case 'd':
		return s.Cave
	case 'k':
		return s.Sky
	}
	if s.Aquatic() {
		if pl.water < 25 {
			return false
		}
		if pl.coast && seaOK(s.Sea) {
			return true
		}
		return s.Fresh
	}
	if s.Coast && pl.coast {
		return true
	}
	if s.Mtn && pl.height < 55 {
		return false
	}
	for _, b := range s.Biomes {
		if b == pl.biome {
			return true
		}
	}
	return s.Fresh && pl.water >= 25 && !s.Aquatic() && len(s.Biomes) == 0
}

// bsWeight is a species' share of the draw at a place (0 = never).
func (s *bsSp) weight(pl *bsPlace) float64 {
	if !s.fits(pl) {
		return 0
	}
	w := bsRarW[s.Rarity]
	switch s.Era {
	case "living":
	case "recent":
		w *= 0.3
	case "pleistocene":
		cold := pl.biome == 2 || pl.biome == 4 || pl.biome == 9 || pl.biome == 10 || pl.biome == 11
		if !bsIceAge[pl.genre] || !cold || pl.where != 's' {
			return 0
		}
		w *= 0.12
	default:
		return 0
	}
	if s.Hostile() {
		w *= 0.35
	}
	if s.Mtn && pl.height >= 55 {
		w *= 2
	}
	return w
}

func bsDraw(r *Rng, pool []*bsSp, ws []float64) *bsSp {
	tot := 0.0
	for _, w := range ws {
		tot += w
	}
	if tot <= 0 {
		return nil
	}
	x := r.Float() * tot
	for i, w := range ws {
		if x -= w; x <= 0 {
			return pool[i]
		}
	}
	return pool[len(pool)-1]
}

func (g *Game) bsPlaceOf(zn *Zone) *bsPlace {
	w := g.W
	p := w.Idx(clampi(zn.WX, 0, w.W-1), clampi(zn.WY, 0, w.H-1))
	pl := &bsPlace{biome: zn.Biome, height: w.Height[p], temp: w.Temp[p], genre: g.Genre}
	switch {
	case zn.Sky:
		pl.where = 'k'
	case zn.Underwater:
		pl.where = 'e'
	case zn.Z > 0:
		pl.where = 'd'
	case zn.Kind == "sea":
		pl.where = 'o'
	default:
		pl.where = 's'
		pl.surface = true
	}
	for _, d := range dirs8 {
		nx, ny := zn.WX+d[0], zn.WY+d[1]
		if w.In(nx, ny) && w.IsWater(w.Idx(nx, ny)) {
			pl.coast = true
		}
	}
	for i := range zn.T {
		if zn.T[i].D().Water >= 2 {
			pl.water++
		}
	}
	if zn.Underwater {
		pl.water = len(zn.T)
	}
	return pl
}

// bsSpot finds a free tile for an animal (water for swimmers) away from the zone centre.
func bsSpot(zn *Zone, r *Rng, s *bsSp, nx, ny int) (int, int, bool) {
	for i := 0; i < 300; i++ {
		x, y := r.Int(1, MW-2), r.Int(1, MH-2)
		if nx >= 0 {
			x, y = clampi(nx+r.Int(-3, 3), 1, MW-2), clampi(ny+r.Int(-2, 2), 1, MH-2)
		}
		if !zn.Walkable(x, y) || zn.ActorAt(x, y) != nil {
			continue
		}
		t := zn.At(x, y)
		w := zn.WaterAt(x, y)
		if s.Aquatic() {
			if w < 2 {
				continue
			}
		} else if !s.Has("fly") && (w >= 2 || t == TLava || t == TVoid || t == TChasm) {
			continue
		} else if s.Has("fly") && t == TLava {
			continue
		}
		if nx < 0 && abs(x-40) <= 6 && abs(y-12) <= 3 {
			continue
		}
		return x, y, true
	}
	return 0, 0, false
}

func (s *bsSp) groupSize(r *Rng) int {
	switch s.Temper {
	case "herd":
		if s.Size >= 5 {
			return r.Int(2, 5)
		}
		return r.Int(3, 7)
	case "pack":
		return r.Int(2, 4)
	case "skittish", "passive":
		if s.Size <= 2 {
			return r.Int(1, 3)
		}
		if s.Has("pack") {
			return r.Int(2, 4)
		}
	}
	if s.Has("pack") {
		return r.Int(2, 4)
	}
	return 1
}

// bsPutGroup places a group of one species; returns how many were placed.
func (g *Game) bsPutGroup(zn *Zone, r *Rng, s *bsSp, n int, morph float64) int {
	x0, y0, ok := bsSpot(zn, r, s, -1, -1)
	if !ok {
		return 0
	}
	put := 0
	for k := 0; k < n; k++ {
		x, y := x0, y0
		if k > 0 {
			if x, y, ok = bsSpot(zn, r, s, x0, y0); !ok {
				continue
			}
		}
		lvl := tplByID[s.Tpl()].Lvl
		if s.Hostile() {
			lvl = maxi(lvl, zn.Level-2)
		}
		a := g.bsSpawn(s, lvl, r)
		a.X, a.Y, a.HomeX, a.HomeY = x, y, x, y
		if morph > 0 && r.Chance(morph) {
			paApplyMorph(a, paBiomeMorph(zn.Biome, zn.Z, g.Genre, r))
		}
		zn.Actors = append(zn.Actors, a)
		put++
	}
	return put
}

// bsZone: the zone hook. Side rng only; it appends actors and items and never touches the rest.
func bsZone(g *Game, zn *Zone, _ *Rng) {
	if g.W == nil || len(g.W.Biome) == 0 {
		return
	}
	r := NewRng(g.W.Seed + ":bs:zone:" + zn.Key)
	if kind := g.bsLostAt(zn.WX, zn.WY, zn.ZX, zn.ZY, zn.Z); kind != "" && bsLostZoneOK(zn, kind) {
		g.bsPopulateLost(zn, r, kind)
		return
	}
	g.bsFossils(zn, r)
	switch zn.Kind {
	case "surface", "outskirts", "sea", "seafloor", "underdark", "strata", "sky":
	default:
		return
	}
	pl := g.bsPlaceOf(zn)
	var pool []*bsSp
	var ws []float64
	for _, s := range bsAll {
		if w := s.weight(pl); w > 0 {
			pool = append(pool, s)
			ws = append(ws, w)
		}
	}
	if len(pool) == 0 {
		return
	}
	n := r.Int(2, 4)
	switch zn.Kind {
	case "outskirts":
		n = r.Int(0, 1)
	case "sea", "sky", "underdark", "strata":
		n = r.Int(1, 2)
	case "seafloor":
		n = r.Int(1, 3)
	}
	if pl.where == 's' && pl.water >= 25 && n < 4 {
		n++ // a river or shore draws more life
	}
	morph := 0.03
	if bsPunk[g.Genre] {
		morph = 0.1
	}
	for i := 0; i < n; i++ {
		s := bsDraw(r, pool, ws)
		if s == nil {
			break
		}
		g.bsPutGroup(zn, r, s, s.groupSize(r), morph)
	}
	if zn.Kind == "seafloor" && r.Chance(0.12) {
		g.bsTidePool(zn, r)
	}
}

// ---- lost worlds ----

var bsLostNames = map[string]string{"plateau": "The Lost Plateau", "valley": "The Hidden Valley", "cavern": "The Primordial Cavern", "trench": "The Abyssal Trench"}

// bsLostAt says whether a zone is one of the rare lost worlds: deterministic from the world seed alone.
func (g *Game) bsLostAt(wx, wy, zx, zy, z int) string {
	w := g.W
	if !w.In(wx, wy) || zx == 1 && zy == 1 {
		return ""
	}
	p := w.Idx(wx, wy)
	sub := func(h uint64) (int, int) {
		i := int(h % 8)
		if i >= 4 {
			i++
		}
		return i % 3, i / 3
	}
	switch {
	case z == 0:
		if w.IsWater(p) || w.Burg[p] > 0 {
			return ""
		}
		h := HashStr(fmt.Sprintf("%s:bs:lost:%d:%d", w.Seed, wx, wy))
		kind := ""
		switch w.Biome[p] {
		case 5, 7, 8, 12:
			if h%100 < 3 {
				kind = "plateau"
			}
		case 2, 4, 9, 10, 11:
			if h%100 < 4 {
				kind = "valley"
			}
		}
		if kind == "" {
			return ""
		}
		if sx, sy := sub(h / 100); sx != zx || sy != zy {
			return ""
		}
		if g.siteAt(wx, wy, zx, zy) != nil {
			return ""
		}
		if k, _ := g.ringAt(wx, wy, zx, zy); k != "" {
			return ""
		}
		return kind
	case z == 1 && w.IsWater(p):
		h := HashStr(fmt.Sprintf("%s:bs:trench:%d:%d", w.Seed, wx, wy))
		if h%100 < 3 {
			if sx, sy := sub(h / 100); sx == zx && sy == zy {
				return "trench"
			}
		}
	case z >= 2 && z <= 8:
		h := HashStr(fmt.Sprintf("%s:bs:cavern:%d:%d:%d", w.Seed, wx, wy, z))
		if h%1000 < 15 {
			if sx, sy := sub(h / 1000); sx == zx && sy == zy {
				return "cavern"
			}
		}
	}
	return ""
}

func bsLostZoneOK(zn *Zone, kind string) bool {
	switch kind {
	case "plateau", "valley":
		return zn.Kind == "surface"
	case "trench":
		return zn.Kind == "seafloor"
	case "cavern":
		return zn.Kind == "strata" || zn.Kind == "underdark"
	}
	return false
}

// bsLostPool is who lives in a kind of lost world, with weights.
func bsLostPool(kind string) ([]*bsSp, []float64) {
	var pool []*bsSp
	var ws []float64
	for _, s := range bsAll {
		w := 0.0
		land := len(s.Biomes) > 0 || s.Fresh && !s.Aquatic()
		marine := s.Sea > 0 || s.Reef > 0
		switch kind {
		case "plateau":
			if land && !marine || s.Fresh {
				switch s.Era {
				case "mesozoic":
					w = 1
				case "cenozoic":
					w = 0.35
				}
			}
		case "valley":
			if land && !marine {
				switch s.Era {
				case "pleistocene":
					w = 1
				case "cenozoic":
					if s.Ma < 25 {
						w = 0.25
					}
				}
			}
		case "cavern":
			if land || s.Fresh || s.Cave {
				switch {
				case s.Era == "paleozoic":
					w = 1
				case s.Era == "mesozoic" && strings.Contains(s.Period, "Triassic"):
					w = 0.5
				}
			}
		case "trench":
			if marine {
				switch s.Era {
				case "mesozoic", "paleozoic":
					w = 1
				case "cenozoic":
					w = 0.6
				case "precambrian":
					w = 0.3
				}
			}
		}
		if w > 0 {
			w *= bsRarW[mini(s.Rarity+1, 5)]
			if s.Hostile() {
				w *= 0.6
			}
			pool = append(pool, s)
			ws = append(ws, w)
		}
	}
	return pool, ws
}

func (g *Game) bsPopulateLost(zn *Zone, r *Rng, kind string) {
	zn.Name = bsLostNames[kind]
	pool, ws := bsLostPool(kind)
	if len(pool) == 0 {
		return
	}
	// the first group is always the lost world's signature fauna
	first := func(s *bsSp) bool {
		switch kind {
		case "plateau":
			return s.Group == "dino"
		case "valley":
			return s.Group == "mam" && s.Era == "pleistocene"
		case "cavern":
			return s.Era == "paleozoic"
		}
		return s.Group == "rept" || s.Group == "fish"
	}
	var fp []*bsSp
	var fw []float64
	for i, s := range pool {
		if first(s) {
			fp = append(fp, s)
			fw = append(fw, ws[i])
		}
	}
	n := r.Int(5, 8)
	for i := 0; i < n; i++ {
		s := bsDraw(r, pool, ws)
		if i == 0 && len(fp) > 0 {
			s = bsDraw(r, fp, fw)
		}
		if s == nil {
			continue
		}
		if g.bsPutGroup(zn, r, s, s.groupSize(r), 0) == 0 && i == 0 {
			n++ // keep trying for the signature beast
		}
		if n > 14 {
			break
		}
	}
	// fossils in the walls of caverns and plateaus
	if kind != "trench" {
		for k := 0; k < r.Int(1, 2); k++ {
			if x, y, ok := zn.FreeSpot(r, nil); ok {
				it := g.bsMakeFossil(Pick(r, pool), r)
				it.X, it.Y = x, y
				zn.Items = append(zn.Items, it)
			}
		}
	}
}

// bsTidePool drops one living Precambrian or Paleozoic curiosity on a sea floor.
func (g *Game) bsTidePool(zn *Zone, r *Rng) {
	var pool []*bsSp
	for _, s := range bsAll {
		if (s.Era == "precambrian" || s.Era == "paleozoic") && s.Reef > 0 && !s.Hostile() && s.Size <= 3 {
			pool = append(pool, s)
		}
	}
	if len(pool) == 0 {
		return
	}
	s := Pick(r, pool)
	if g.bsPutGroup(zn, r, s, 1, 0) > 0 {
		a := zn.Actors[len(zn.Actors)-1]
		a.Desc += " By every right it should have been gone for hundreds of millions of years."
	}
}

// ---- fossils ----

var bsFossilParts = map[string][]string{
	"mam": {"skull", "jawbone", "molar", "femur"}, "bird": {"wing bone", "skull", "egg"}, "rept": {"skull", "vertebra", "tooth"},
	"dino": {"tooth", "claw", "vertebra", "skull", "egg"}, "ptero": {"wing bone", "skull"}, "amph": {"skull", "skeleton"},
	"fish": {"tooth", "skull plate", "scale"}, "ins": {"wing in amber", "imprint"}, "arach": {"imprint", "body in amber"},
	"crust": {"carapace", "claw"}, "myria": {"trackway", "segment"}, "arth": {"carapace", "imprint"}, "moll": {"shell"},
	"cnid": {"imprint", "colony"}, "echi": {"cast"}, "worm": {"burrow cast"}, "misc": {"imprint"},
}

// bsMakeFossil: Kind "fossil", Effect "bs:<id>", the encyclopedia entry as Lore.
func (g *Game) bsMakeFossil(s *bsSp, r *Rng) *Item {
	part := Pick(r, bsFossilParts[s.Group])
	val := 20 + s.Rarity*12 + s.Size*6
	switch s.Era {
	case "mesozoic":
		val += 40
	case "paleozoic", "precambrian":
		val += 30
	case "cenozoic", "pleistocene":
		val += 15
	}
	return &Item{ID: g.newID(), Kind: "fossil", Name: "fossil " + part + " of " + bsArticle(s.Name), Glyph: '§', F: CGray, Value: val, Weight: 2,
		Effect: "bs:" + s.ID, Ident: true, Lore: s.Name + ": " + s.Lore() + " " + s.Desc}
}

func bsArticle(n string) string {
	if n == "" {
		return n
	}
	if n[0] >= 'A' && n[0] <= 'Z' && !strings.Contains(n, " ") {
		return n // a genus name: "of Tyrannosaurus"
	}
	if strings.ContainsRune("aeiouAEIOU", rune(n[0])) {
		return "an " + n
	}
	return "a " + n
}

// bsFossilSpecies is the species a fossil item belongs to.
func bsFossilSpecies(it *Item) *bsSp {
	if it == nil || !strings.HasPrefix(it.Effect, "bs:") {
		return nil
	}
	return bsByID[it.Effect[3:]]
}

func (g *Game) bsFossils(zn *Zone, r *Rng) {
	chance := 0.0
	switch zn.Kind {
	case "strata":
		chance = 0.22
	case "underdark":
		chance = 0.08
	case "ruin", "crypt":
		chance = 0.1
	case "surface":
		if b := zn.Biome; b == 1 || b == 2 {
			chance = 0.03 // badlands
		}
	}
	if chance == 0 || !r.Chance(chance) {
		return
	}
	var pool []*bsSp
	for _, s := range bsAll {
		if s.Extinct() && s.Era != "recent" {
			pool = append(pool, s)
		}
	}
	if len(pool) == 0 {
		return
	}
	if x, y, ok := zn.FreeSpot(r, nil); ok {
		it := g.bsMakeFossil(Pick(r, pool), r)
		it.X, it.Y = x, y
		zn.Items = append(zn.Items, it)
	}
}

// ---- behaviour ----

type bsState struct {
	Lost  map[string]bool `json:"lost,omitempty"`  // lost worlds announced
	Fossi map[string]bool `json:"fos,omitempty"`   // species known from fossils
	Rumor map[string]bool `json:"rumor,omitempty"` // rumours heard (npc id:site)
	alarm map[int]int     // actor id → turn until which it is fleeing
}

func (g *Game) bs() *bsState {
	st := g.ext("bs", func() any { return &bsState{} }).(*bsState)
	if st.Lost == nil {
		st.Lost = map[string]bool{}
	}
	if st.Fossi == nil {
		st.Fossi = map[string]bool{}
	}
	if st.Rumor == nil {
		st.Rumor = map[string]bool{}
	}
	if st.alarm == nil {
		st.alarm = map[int]int{}
	}
	return st
}

// bsAwake: nocturnal animals sleep by day and diurnal ones by night unless something is close.
func (g *Game) bsAwake(s *bsSp, a *Actor) bool {
	if a.Angry || s.Act == 'a' || s.Act == 'c' {
		return true
	}
	day := g.daylight()
	if s.Act == 'n' && !day || s.Act == 'd' && day {
		return true
	}
	return dist(a, g.P) <= 2
}

func (g *Game) bsFleeStep(a *Actor, from *Actor) bool {
	best := dist(a, from)
	bx, by := -1, -1
	for _, d := range dirs8 {
		nx, ny := a.X+d[0], a.Y+d[1]
		if !g.Cur.In(nx, ny) || g.Cur.ActorAt(nx, ny) != nil || !g.canEnter(a, nx, ny) {
			continue
		}
		if v := maxi(abs(nx-from.X), abs(ny-from.Y)); v > best {
			best, bx, by = v, nx, ny
		}
	}
	if bx < 0 {
		return false
	}
	a.X, a.Y = bx, by
	return true
}

func (g *Game) bsStepTo(a *Actor, to *Actor) bool {
	best := dist(a, to)
	bx, by := -1, -1
	for _, d := range dirs8 {
		nx, ny := a.X+d[0], a.Y+d[1]
		if !g.Cur.In(nx, ny) || g.Cur.ActorAt(nx, ny) != nil || !g.canEnter(a, nx, ny) {
			continue
		}
		if v := maxi(abs(nx-to.X), abs(ny-to.Y)); v < best {
			best, bx, by = v, nx, ny
		}
	}
	if bx < 0 {
		return false
	}
	a.X, a.Y = bx, by
	return true
}

// bsAct is the wildlife brain (one line in aiAct): sleeping, fleeing, herds, warnings, predation.
// It returns false to hand the actor to the normal AI.
func (g *Game) bsAct(a *Actor) bool {
	if !strings.HasPrefix(a.Tpl, "bs_") || a.AI == "follower" || g.Cur == nil || g.P == nil {
		return false
	}
	s := bsByTpl[a.Tpl]
	if s == nil {
		return false
	}
	if !g.bsAwake(s, a) {
		return true
	}
	st := g.bs()
	p := g.P
	dp := dist(a, p)
	if s.Hostile() {
		if dp <= 10 || a.Angry {
			return false
		}
		// hunt: the nearest smaller wildlife within 8
		var prey *Actor
		pb := 9
		for _, o := range g.Cur.Actors {
			if o == a || o.HP <= 0 || o.AI == "follower" || !strings.HasPrefix(o.Tpl, "bs_") {
				continue
			}
			os := bsByTpl[o.Tpl]
			if os == nil || os.Hostile() || os.Size >= s.Size+1 || os.Size < s.Size-3 || os.Aquatic() != s.Aquatic() {
				continue
			}
			if d := dist(a, o); d < pb {
				prey, pb = o, d
			}
		}
		if prey == nil || g.R.Chance(0.5) {
			return false
		}
		st.alarm[prey.ID] = g.Turn + 20
		if pb <= 1 {
			g.melee(a, prey)
			return true
		}
		g.bsStepTo(a, prey)
		return true
	}
	if a.Angry {
		if (s.Temper == "skittish" || s.Temper == "herd" && s.Size <= 3) && a.HP < a.MaxHP*2/3 {
			if g.bsFleeStep(a, p) {
				return true
			}
		}
		return false
	}
	// threats: the player and nearby hunters
	threat, td := (*Actor)(nil), 99
	seen := g.visible(a.X, a.Y)
	if seen {
		threat, td = p, dp
	}
	for _, o := range g.Cur.Actors {
		if o == a || o.HP <= 0 || !strings.HasPrefix(o.Tpl, "bs_") {
			continue
		}
		if os := bsByTpl[o.Tpl]; os != nil && os.Hostile() && os.Size >= s.Size-1 {
			if d := dist(a, o); d < td && d <= 6 {
				threat, td = o, d
			}
		}
	}
	switch s.Temper {
	case "passive":
		if g.R.Chance(0.1) {
			g.wander(a)
		}
		return true
	case "territorial":
		if threat == p && td <= 3 {
			if a.Flags["bs_warned"] && td <= 1 {
				g.provoke(a)
				return false
			}
			if !a.Flags["bs_warned"] {
				a.Set("bs_warned")
				g.Msg(fmt.Sprintf("{{O|%s %s a warning.}}", g.The(a), bsWarnVerb(s)))
			}
			return true
		}
		if threat != nil && threat != p && td <= 2 {
			return false
		}
	case "skittish", "herd":
		radius := 6
		if s.Temper == "herd" {
			radius = 5
			if s.Size >= 5 {
				radius = 3
			}
		}
		if p.Has("sneak") || p.St("hidden") > 0 {
			radius /= 2
		}
		if threat != nil && td <= radius {
			st.alarm[a.ID] = g.Turn + 15
		}
		if s.Temper == "herd" && st.alarm[a.ID] <= g.Turn {
			for _, o := range g.Cur.Actors {
				if o != a && o.Tpl == a.Tpl && dist(a, o) <= 5 && st.alarm[o.ID] > g.Turn {
					st.alarm[a.ID] = st.alarm[o.ID]
					break
				}
			}
		}
		if st.alarm[a.ID] > g.Turn {
			from := threat
			if from == nil {
				from = p
			}
			if g.bsFleeStep(a, from) {
				return true
			}
		}
	}
	if g.R.Chance(0.25) {
		g.wander(a)
	}
	return true
}

func bsWarnVerb(s *bsSp) string {
	switch s.Group {
	case "bird":
		return "hisses and spreads its wings in"
	case "rept", "dino":
		return "hisses"
	case "ins", "arach", "myria":
		return "rears up in"
	}
	if s.Size >= 5 {
		return "bellows"
	}
	return "growls"
}

// ---- hunting ----

func bsMeatTag(s *bsSp) string {
	switch {
	case s.Has("poison"):
		return "venom"
	case s.Ice || len(s.Biomes) > 0 && s.Biomes[0] >= 9 && s.Biomes[0] <= 11:
		return "rime"
	case s.Aquatic():
		return "deep"
	case s.Has("fly"):
		return "spry"
	case s.Has("armored"):
		return "thick"
	case s.Size >= 5:
		return "hearty"
	}
	return []string{"hearty", "lasting", "thick", "spry", "keen"}[HashStr(s.ID)%5]
}

// bsProducts drops what a hunted animal yields: meat, hide or pelt, bones, ivory, horns, feathers, shell.
func (g *Game) bsProducts(s *bsSp, d *Actor) []*Item {
	r := NewRng(fmt.Sprintf("%s:bs:kill:%d", g.W.Seed, d.ID))
	var out []*Item
	low := strings.ToLower(s.Name)
	prod := func(what string, val, wt int, col byte) {
		out = append(out, &Item{ID: g.newID(), Kind: "trophy", Name: low + " " + what, Glyph: '~', F: col, Value: val, Weight: wt, Effect: "bsp:" + s.ID})
	}
	if s.Size >= 2 && s.Group != "cnid" && s.Group != "echi" && s.Group != "misc" && s.Group != "worm" {
		meat := "meat"
		switch s.Group {
		case "fish":
			meat = "fillet"
		case "crust", "moll":
			meat = "flesh"
		case "ins", "arach", "myria", "arth":
			meat = "grubs"
		}
		n := maxi(1, s.Size-1)
		out = append(out, &Item{ID: g.newID(), Kind: "ingredient", Name: low + " " + meat, Effect: bsMeatTag(s), Glyph: '%', F: CRed, Count: n, Value: 3 + s.Size, Power: 120, Weight: 1})
	}
	sz := s.Size
	switch s.Group {
	case "mam":
		if sz >= 2 && !s.Aquatic() {
			what := "hide"
			if s.Ice || len(s.Biomes) > 0 && s.Biomes[0] >= 9 || sz <= 3 {
				what = "pelt"
			}
			prod(what, 4*sz*sz+r.Int(0, 6), sz, CBrown)
		}
		if s.Aquatic() && sz >= 4 {
			prod("blubber", 6*sz, sz, CWhite)
		}
	case "bird", "ptero":
		if s.Group == "bird" {
			prod("feathers", 3+sz*sz*2+s.Rarity*4, 1, CWhite)
		}
	case "rept", "dino", "amph", "fish":
		if s.Has("armored") {
			prod("shell", 5*sz*sz, sz, CDarkGreen)
		} else if sz >= 3 && s.Group != "amph" {
			prod("scales", 3*sz*sz, 1, CDarkGreen)
		}
	case "crust", "arth", "ins", "arach", "myria":
		if sz >= 3 {
			prod("carapace", 4*sz*sz, sz, COrange)
		}
	case "moll":
		if sz >= 2 && !strings.Contains(s.Body, "tentacled") {
			prod("shell", 3*sz*sz+s.Rarity*5, 1, CWhite)
		}
	}
	if sz >= 4 && s.Group != "ins" && s.Group != "crust" && s.Group != "moll" && s.Group != "cnid" {
		prod("bones", 2*sz*sz, sz/2, CWhite)
	}
	switch s.Attack {
	case "antlers", "horns", "horn":
		prod(s.Attack, 6*sz*sz+s.Rarity*10, 2, CYellow)
	}
	if s.Has("ivory") {
		prod("ivory", 20*sz*sz+s.Rarity*20, 3, CWhite)
	}
	if s.Extinct() {
		for _, it := range out {
			it.Value = it.Value * 3 / 2
		}
	}
	return out
}

func bsKill(g *Game, d, src *Actor) {
	if d == nil || g.Cur == nil || !strings.HasPrefix(d.Tpl, "bs_") {
		return
	}
	s := bsByTpl[d.Tpl]
	if s == nil {
		return
	}
	byP := src == g.P || src != nil && src.AI == "follower"
	if byP && !g.Hostile(d) { // the codex only counts hostile kills; wildlife counts here
		if b := g.cxBeastFor(d); b != nil {
			b.Kills++
			b.Lore = s.Lore()
		}
	}
	if !byP {
		return
	}
	for _, it := range g.bsProducts(s, d) {
		it.X, it.Y = d.X, d.Y
		g.Cur.Items = append(g.Cur.Items, it)
	}
}

// ---- codex, enter, talk ----

func bsTurn(g *Game) {
	if g.Cur == nil || g.P == nil || g.Dead {
		return
	}
	c := g.codex()
	for _, a := range g.Cur.Actors {
		if a.HP <= 0 || !strings.HasPrefix(a.Tpl, "bs_") || a.AI == "follower" || !g.visible(a.X, a.Y) {
			continue
		}
		s := bsByTpl[a.Tpl]
		if s == nil {
			continue
		}
		b := g.cxBeastFor(a)
		if b == nil {
			continue
		}
		if !c.seen[a.ID] {
			c.seen[a.ID] = true
			b.Seen++
		}
		if b.Lore == "" {
			b.Lore = s.Lore()
		}
	}
	if g.Turn%30 == 0 {
		st := g.bs()
		for _, it := range g.P.Inv {
			if s := bsFossilSpecies(it); s != nil && !st.Fossi[s.ID] {
				st.Fossi[s.ID] = true
				key := s.Tpl()
				if c.Beasts[key] == nil {
					c.Beasts[key] = &cxBeast{Name: s.Name, Tpl: key, Place: "known from a fossil", Desc: s.Desc, Lore: s.Lore(), MinLv: tplByID[key].Lvl, MaxLv: tplByID[key].Lvl, Order: c.next()}
				}
				g.Msg("{{c|You add " + s.Name + " to your bestiary from the fossil.}}")
			}
		}
	}
}

func bsEnter(g *Game, zn *Zone) {
	if zn == nil {
		return
	}
	kind := g.bsLostAt(zn.WX, zn.WY, zn.ZX, zn.ZY, zn.Z)
	if kind == "" || !bsLostZoneOK(zn, kind) {
		return
	}
	st := g.bs()
	lk := g.W.Seed + "|" + zn.Key
	if st.Lost[lk] {
		return
	}
	st.Lost[lk] = true
	g.Msg("{{G|" + bsLostLines[kind] + "}}")
	g.gainXP(25)
}

var bsLostLines = map[string]string{
	"plateau": "Tree ferns close overhead and the air is thick and warm. Something enormous calls across the valley: a world that should have ended long ago.",
	"valley":  "A hidden valley of steaming springs and wind-flattened grass. Shaggy giants graze here as if the ice had never left.",
	"cavern":  "The cavern opens into a vast humid hollow. Giant ferns grow by glowworm light, and something with too many legs rustles among them.",
	"trench":  "The sea floor drops into an abyssal trench. Shapes far older than the fishes glide along the cold walls.",
}

// bsNearestLost finds the nearest surface lost world within r parasangs of (wx, wy).
func (g *Game) bsNearestLost(wx, wy, rad int) (int, int, int, int, string) {
	best, bx, by, bzx, bzy, bk := 1<<30, 0, 0, 0, 0, ""
	for y := wy - rad; y <= wy+rad; y++ {
		for x := wx - rad; x <= wx+rad; x++ {
			if !g.W.In(x, y) {
				continue
			}
			d := maxi(abs(x-wx), abs(y-wy))
			if d >= best {
				continue
			}
			for i := 0; i < 9; i++ {
				if k := g.bsLostAt(x, y, i%3, i/3, 0); k != "" {
					best, bx, by, bzx, bzy, bk = d, x, y, i%3, i/3, k
				}
			}
		}
	}
	return bx, by, bzx, bzy, bk
}

var bsLearned = map[string]bool{"bookseller": true, "naturalist": true, "chronicler": true, "beastkeeper": true, "forager": true}

func (g *Game) bsCollector(a *Actor) bool {
	if bsLearned[a.Role] {
		return true
	}
	return a.Role != "" && g.Cur != nil && (strings.Contains(g.Cur.Name, "Academy") || strings.Contains(g.Cur.Name, "University"))
}

func bsTalk(g *Game, a *Actor) []Option {
	if a == nil || a.Faction == "beasts" || !g.bsCollector(a) {
		return nil
	}
	var opts []Option
	var fos []*Item
	for _, it := range g.P.Inv {
		if it.Kind == "fossil" {
			fos = append(fos, it)
		}
	}
	if len(fos) > 0 {
		opts = append(opts, Option{Text: fmt.Sprintf("{{c|Offer %d fossil(s) for the collection.}}", len(fos)), Do: func() {
			total := 0
			for _, it := range fos {
				total += it.Value
				g.removeInv(it)
			}
			paid := g.cuEarn(total)
			g.gainXP(5 * len(fos))
			g.talkMenu(fmt.Sprintf("%s turns the stones over with care. \"Remarkable. These belong in a collection.\" You receive %s.", g.The(a), paid))
		}})
	}
	if a.Role != "beastkeeper" {
		st := g.bs()
		key := fmt.Sprintf("%s|%d", g.W.Seed, a.ID)
		if !st.Rumor[key] {
			opts = append(opts, Option{Text: "Heard of any strange beasts?", Do: func() {
				st.Rumor[key] = true
				x, y, zx, zy, k := g.bsNearestLost(g.WX, g.WY, 24)
				if k == "" {
					g.talkMenu("\"Bones in the cliffs, nothing more. Whatever made them is long gone.\"")
					return
				}
				dir := g.cxDir(g.W.Idx(x, y))
				_ = zx
				_ = zy
				g.talkMenu(fmt.Sprintf("\"Hunters speak of %s, %s from here, where beasts from the stones still walk. I'd not go without a stout spear.\"", strings.ToLower(bsLostNames[k][:1])+bsLostNames[k][1:], dir))
			}})
		}
	}
	return opts
}

// bsCodexNote is the bestiary tab's summary line of the Earth catalogue.
func (g *Game) bsCodexNote() []string {
	c := g.codex()
	n, ext := 0, 0
	for k := range c.Beasts {
		if s := bsByTpl[k]; s != nil {
			n++
			if s.Extinct() {
				ext++
			}
		}
	}
	if n == 0 {
		return nil
	}
	return []string{fmt.Sprintf("{{K|Natural history: %d of %d known species recorded, %d of them extinct elsewhere.}}", n, len(bsAll), ext), ""}
}

// bsCounts tallies the catalogue by group and by era (tests and reports).
func bsCounts() (map[string]int, map[string]int) {
	gr, er := map[string]int{}, map[string]int{}
	for _, s := range bsAll {
		gr[bsGroupNames[s.Group]]++
		er[s.Era]++
	}
	return gr, er
}

func bsSortedKeys(m map[string]int) []string {
	var k []string
	for x := range m {
		k = append(k, x)
	}
	sort.Strings(k)
	return k
}
