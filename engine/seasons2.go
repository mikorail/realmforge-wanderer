package engine

import (
	"fmt"
	"sort"
	"strings"
)

// Seasons, part two: seasonal wild herbs, merchants' seasonal specials, named festival dishes sold in the
// days before a feast, and townsfolk who take shelter from bad weather.
//
// Zones are generated once and cached, so seasonal herbs are not part of zone generation: an enter hook
// (and an hourly check while you stay) seeds the current season's herbs into a wild zone from a side rng
// keyed by world seed + zone + season, and lets the last season's uncollected ones wither. The zone's own
// generation, including the base herbs of scatterHerbs, is unchanged.
//
// Item.Extra marks: "snH" = season+1 of a wild seasonal herb, "snS" = season+1 of a merchant's seasonal
// special, "snF" = game day+1 of the feast a festival dish belongs to.

type snHerb struct {
	Name, Tag string
	Seasons   []int // 0 spring … 3 winter
	Biomes    []int // nil = any land
	Under     bool
}

var snHerbs = []snHerb{
	// spring greens
	{"young nettles", "mending", []int{0}, []int{4, 5, 6, 8, 12}, false},
	{"fiddlehead ferns", "spry", []int{0}, []int{6, 7, 8, 9, 12}, false},
	{"wild ramsons", "hearty", []int{0}, []int{6, 8, 5}, false},
	{"elderflower", "keen", []int{0, 1}, []int{4, 6, 8}, false},
	{"watercress", "deep", []int{0, 1}, []int{12, 7, 8, 6}, false},
	{"desert bloom nectar", "spry", []int{0}, []int{1, 2, 3}, false},
	{"meltwater sorrel", "rime", []int{0}, []int{9, 10, 11}, false},
	// summer berries
	{"wild strawberries", "spry", []int{1}, []int{4, 6, 8, 5}, false},
	{"bilberries", "keen", []int{1}, []int{9, 6, 8}, false},
	{"cloudberries", "rime", []int{1}, []int{9, 10, 11}, false},
	{"prickly pears", "lasting", []int{1}, []int{1, 2, 3}, false},
	{"sun-split figs", "ember", []int{1}, []int{3, 5, 7}, false},
	{"wild blackberries", "mending", []int{1, 2}, []int{4, 6, 8, 12}, false},
	// autumn mushrooms and nuts
	{"chanterelles", "hearty", []int{2}, []int{6, 8, 9}, false},
	{"hazelnuts", "thick", []int{2}, []int{4, 6, 8}, false},
	{"rosehips", "mending", []int{2}, []int{4, 5, 6, 3}, false},
	{"sweet chestnuts", "lasting", []int{2}, []int{6, 7, 8, 5}, false},
	{"fly agaric", "venom", []int{2}, []int{8, 9, 6}, false},
	{"date clusters", "lasting", []int{2}, []int{1, 3}, false},
	// winter roots and lichen
	{"frost-sweet parsnip", "lasting", []int{3}, []int{4, 6, 8, 5}, false},
	{"reindeer lichen", "rime", []int{3}, []int{9, 10, 11, 2}, false},
	{"frozen sloes", "venom", []int{3}, []int{4, 6, 8}, false},
	{"winter salsify", "hearty", []int{3}, []int{6, 9, 12, 4}, false},
	{"desert truffle", "thick", []int{3}, []int{1, 2, 3}, false},
	// any land, one season each
	{"dandelion greens", "mending", []int{0}, nil, false},
	{"wild honeycomb", "spry", []int{1}, nil, false},
	{"fallen acorns", "thick", []int{2}, nil, false},
	{"bare-root tubers", "lasting", []int{3}, nil, false},
	// below ground: cave fungi fruit in the damp of autumn and the warmth of spring
	{"weeping oyster caps", "mending", []int{0}, nil, true},
	{"autumn honey fungus", "hearty", []int{2}, nil, true},
	{"rime-blind lichen", "rime", []int{3}, nil, true},
}

// per season: the flavour and the word of a merchant's seasonal special
var snSpecial = [4]struct{ Tag, Fmt string }{
	{"mending", "spring-fresh %s"},
	{"spry", "midsummer %s"},
	{"hearty", "harvest-new %s"},
	{"lasting", "winter-cured %s"},
}

const snFeastLead = 3 // festival dishes go on sale this many days before the feast

type snState struct {
	Seeded map[string]int    `json:"z,omitempty"` // zone key → absolute season index+1 last seeded
	Marks  map[string]string `json:"m,omitempty"` // dish key → "Spring"… or "festival:<feast>"
}

func (g *Game) snSt() *snState {
	s := g.ext("seasons2", func() any { return &snState{} }).(*snState)
	if s.Seeded == nil {
		s.Seeded = map[string]int{}
	}
	if s.Marks == nil {
		s.Marks = map[string]string{}
	}
	return s
}

func init() {
	enterHooks = append(enterHooks, func(g *Game, zn *Zone) { g.snSeasonHerbs(zn) })
	turnHooks = append(turnHooks, func(g *Game) {
		if g.Turn%60 == 0 && g.Cur != nil && !g.OnWorld {
			g.snSeasonHerbs(g.Cur)
		}
	})
}

func snIn(xs []int, v int) bool {
	for _, x := range xs {
		if x == v {
			return true
		}
	}
	return false
}

// snHerbPool: the seasonal herbs that grow in a biome (under = below ground) in a season.
func snHerbPool(b, season int, under bool) []snHerb {
	var pool []snHerb
	for _, h := range snHerbs {
		if h.Under == under && snIn(h.Seasons, season) && (h.Biomes == nil || snIn(h.Biomes, b)) {
			pool = append(pool, h)
		}
	}
	return pool
}

// snHerbZone: zones that carry wild herbs at all (the same ones scatterHerbs fills).
func snHerbZone(zn *Zone) bool {
	return zn != nil && ((zn.Z == 0 && zn.Kind == "surface") || (zn.Z > 0 && zn.Kind == "underdark"))
}

// snHerbPicks is the pure plan for a zone's seasonal herbs: names, tags and spots, from a side rng keyed
// by world seed, zone and season (the same every year).
func (g *Game) snHerbPicks(zn *Zone, season int) []*Item {
	if !snHerbZone(zn) {
		return nil
	}
	b := -1
	if p := g.W.Idx(zn.WX, zn.WY); p >= 0 && p < len(g.W.Biome) {
		b = g.W.Biome[p]
	}
	pool := snHerbPool(b, season, zn.Z > 0)
	if len(pool) == 0 {
		return nil
	}
	r := NewRng(fmt.Sprintf("%s:herbs2:%s:%d", g.W.Seed, zn.Key, season))
	n := r.Int(1, 3)
	if zn.Z > 0 {
		n = r.Int(0, 1)
	}
	var out []*Item
	for i := 0; i < n; i++ {
		h := pool[r.Int(0, len(pool)-1)]
		x, y, ok := zn.FreeSpot(r, nil)
		if !ok {
			break
		}
		it := &Item{Kind: "ingredient", Name: h.Name, Effect: h.Tag, Glyph: '%', F: CGreen, Count: 1, Value: 5, Power: 120, X: x, Y: y, Extra: map[string]int{"snH": season + 1}}
		out = append(out, it)
	}
	return out
}

// snSeasonHerbs: wither last season's wild herbs in this zone and seed this season's, once per season.
func (g *Game) snSeasonHerbs(zn *Zone) {
	if !snHerbZone(zn) || g.W == nil {
		return
	}
	season := g.Season()
	abs := g.Day()/SeasonDays + 1
	st := g.snSt()
	if st.Seeded[zn.Key] == abs {
		return
	}
	st.Seeded[zn.Key] = abs
	keep := zn.Items[:0]
	for _, it := range zn.Items {
		if it.Extra != nil && it.Extra["snH"] > 0 && it.Extra["snH"] != season+1 {
			continue // withered
		}
		keep = append(keep, it)
	}
	zn.Items = keep
	for _, it := range g.snHerbPicks(zn, season) {
		dup := false
		for _, o := range zn.Items {
			if o.X == it.X && o.Y == it.Y && o.Name == it.Name {
				dup = true // still lying there from this season last year
			}
		}
		if !dup {
			it.ID = g.newID()
			zn.Items = append(zn.Items, it)
		}
	}
}

// ---- merchants: seasonal special, festival dishes ----

func (g *Game) snTownCulture() *CultureData {
	if g.Cur != nil && g.Cur.Kind == "town" {
		if bd := g.W.burgByID[g.Cur.Burg]; bd != nil {
			if cu := g.W.cultByID[bd.Culture]; cu != nil {
				return cu
			}
		}
	}
	return g.localCulture()
}

// snSpecialFor: a culture's seasonal special (one of its staples, prepared for the season).
func snSpecialFor(cu *CultureData, season int) (name, tag string) {
	if cu == nil {
		return "", ""
	}
	st := staples(cu.Food)
	if len(st) == 0 {
		return "", ""
	}
	s := st[HashStr(fmt.Sprintf("%s:special:%d", cu.Name, season))%uint64(len(st))]
	sp := snSpecial[season&3]
	return fmt.Sprintf(sp.Fmt, s), sp.Tag
}

// snStock runs when a merchant's wares are opened: last season's special and past festival dishes leave
// the stall, this season's special arrives once per season.
func (g *Game) snStock(a *Actor) {
	if a == nil || a.Role != "merchant" {
		return
	}
	season, day := g.Season(), g.Day()
	keep := a.Inv[:0]
	for _, it := range a.Inv {
		if it.Extra != nil {
			if s := it.Extra["snS"]; s > 0 && s != season+1 {
				continue
			}
			if f := it.Extra["snF"]; f > 0 && day > f-1 {
				continue
			}
		}
		keep = append(keep, it)
	}
	a.Inv = keep
	flag := fmt.Sprintf("snS:%d", day/SeasonDays)
	if a.Has(flag) {
		return
	}
	a.Set(flag)
	if name, tag := snSpecialFor(g.snTownCulture(), season); name != "" {
		it := g.makeIngredient(name, tag, 2, CYellow)
		it.Value = 7
		it.Extra = map[string]int{"snS": season + 1}
		a.Inv = append(a.Inv, it)
	}
}

// snFeastWindow: the town you stand in and its feast that is today or at most snFeastLead days away,
// with the game day it falls on.
func (g *Game) snFeastWindow() (*BurgData, *FeastData, int) {
	b := g.feastTown()
	if b == nil {
		return nil, nil, 0
	}
	day := g.Day()
	if f := g.W.FeastOn(b, day); f != nil {
		return b, f, day
	}
	if f, d := g.W.nextFeast(b, day); f != nil && d <= snFeastLead {
		return b, f, day + d
	}
	return nil, nil, 0
}

// snFestivalDish: a feast's own dish, named after the feast, its kind's dish word and one of the
// culture's staples; it always carries the feast's tag twice (level 2) plus the staple's flavour.
func (w *WorldData) snFestivalDish(b *BurgData, f *FeastData) (name, key string) {
	tag, word, _, _, _ := feastStyleOf(f.Kind)
	tags := []string{tag, tag}
	name = f.Name + " " + word
	if cu := w.cultByID[b.Culture]; cu != nil {
		if st := staples(cu.Food); len(st) > 0 {
			s := st[HashStr(cu.Name+":fest:"+f.Name)%uint64(len(st))]
			name += " with " + s
			if t := stapleTag(s); t != tag {
				tags = append(tags, t)
			} else {
				// the staple tastes of the feast already: a third, unusual note from the feast's name
				alt := cookTagIDs[HashStr(f.Name)%uint64(len(cookTagIDs))]
				if alt == tag {
					alt = cookTagIDs[(HashStr(f.Name)+1)%uint64(len(cookTagIDs))]
				}
				tags = append(tags, alt)
			}
		}
	}
	sort.Strings(tags)
	return name, strings.Join(tags, "+")
}

// ---- cookbook marks ----

func (g *Game) snMarkSeasonal(key string, ings []*Item) {
	for _, it := range ings {
		if it.Extra != nil {
			s := it.Extra["snH"]
			if s == 0 {
				s = it.Extra["snS"]
			}
			if s > 0 {
				if _, ok := g.snSt().Marks[key]; !ok {
					g.snSt().Marks[key] = SeasonNames[(s-1)&3]
				}
				return
			}
		}
	}
}

// snLearnFestival: a festival dish you eat (or share at the feast) goes into the cookbook, marked.
func (g *Game) snLearnFestival(name, key, where, feast string) {
	if key == "" {
		return
	}
	g.snSt().Marks[key] = "festival:" + feast
	if !g.knowsRecipe(key) {
		g.Cookbook = append(g.Cookbook, Dish{Key: key, Name: name, Tags: strings.Split(key, "+"), Where: where, FirstAt: g.Turn / 1440})
	}
}

// snNoteEaten: called when a ready dish is eaten; festival dishes are remembered.
func (g *Game) snNoteEaten(it *Item) {
	if it.Extra == nil || it.Extra["snF"] == 0 {
		return
	}
	g.snLearnFestival(it.Name, it.Effect, "", it.Name)
}

// snMark is the cookbook badge of a dish key ("" for ordinary dishes).
func (g *Game) snMark(key string) string {
	m := g.snSt().Marks[key]
	switch {
	case m == "":
		return ""
	case strings.HasPrefix(m, "festival:"):
		return " {{M|[festival]}}"
	}
	return " {{G|[" + strings.ToLower(m) + "]}}"
}

// ---- weather shelter ----

// snShelterKinds: weather that drives townsfolk under a roof.
var snShelterKinds = map[string]bool{"storm": true, "tempest": true, "blizzard": true, "sandstorm": true, "radstorm": true, "acidrain": true, "fallout": true, "ashrain": true}

// snFoul: the weather over the current zone is bad enough to shelter from ("" when not).
func (g *Game) snFoul() string {
	if g.Cur == nil || g.OnWorld {
		return ""
	}
	w := g.activeWeather()
	if w == nil || !snShelterKinds[w.Kind.ID] {
		return ""
	}
	return w.Kind.Name
}

func (g *Game) snThreat(a *Actor) bool {
	if g.Hostile(a) {
		return true
	}
	for _, o := range g.Cur.Actors {
		if o != a && o.HP > 0 && dist(a, o) < 14 && g.Enemies(a, o) {
			return true
		}
	}
	return false
}

// snShelterMap: steps from every free roofed tile (multi-source BFS), or from trees/bushes for beasts.
func (g *Game) snShelterMap(beast bool) []int {
	z := g.Cur
	d := make([]int, MW*MH)
	var q []int
	for i := range d {
		d[i] = 1 << 20
		x, y := i%MW, i/MW
		t := z.T[i]
		src := roofed(t)
		if beast {
			src = t == TTree || t == TBush || t == TDeadTree || t == TMushroom
		}
		if src && z.ActorAt(x, y) == nil && !(x == g.P.X && y == g.P.Y) {
			d[i] = 0
			q = append(q, i)
		}
	}
	for len(q) > 0 {
		i := q[0]
		q = q[1:]
		cx, cy := i%MW, i/MW
		for _, dd := range dirs8 {
			nx, ny := cx+dd[0], cy+dd[1]
			if !z.In(nx, ny) {
				continue
			}
			ni := ny*MW + nx
			if d[ni] <= d[i]+1 {
				continue
			}
			t := z.At(nx, ny)
			if !t.D().Walk && t != TDoor || t == TLava || t == TVoid || t == TChasm {
				continue
			}
			d[ni] = d[i] + 1
			q = append(q, ni)
		}
	}
	return d
}

func snBeast(a *Actor) bool {
	return a.AI == "wander" && a.Role == "" && !a.Proper && a.Species == "" && a.SimPer == 0 && (a.Has("fly") || a.Has("pack"))
}

// snShelterAct (ai.go hook, before target selection): in foul weather townsfolk walk to the nearest free
// roofed tile and stay; guards keep their posts; flying and pack beasts make for cover. Afterwards
// sheltered townsfolk walk back home. Returns true when it used the actor's action.
func (g *Game) snShelterAct(a *Actor) bool {
	if a.AI != "town" && a.AI != "guard" && a.AI != "wander" {
		return false
	}
	foul := g.snFoul() != ""
	switch {
	case a.AI == "guard":
		return foul && !g.snThreat(a) // stand fast at the post
	case a.AI == "town":
		if g.snThreat(a) {
			return false
		}
		if foul {
			a.Set("snShelter")
			if roofed(g.Cur.At(a.X, a.Y)) {
				return true
			}
			dm := g.snShelterMap(false)
			if dm[a.Y*MW+a.X] >= 1<<20 {
				return false // nowhere to go: carry on as usual
			}
			g.stepToward(a, dm, false)
			return true
		}
		if a.Has("snShelter") {
			if abs(a.X-a.HomeX) <= 1 && abs(a.Y-a.HomeY) <= 1 {
				delete(a.Flags, "snShelter")
				return false
			}
			if !g.stepToward(a, g.distMap(a.HomeX, a.HomeY, false), false) {
				delete(a.Flags, "snShelter") // the way home is blocked: resume the usual routine
				return false
			}
			return true
		}
	case snBeast(a) && foul && !g.snThreat(a):
		t := g.Cur.At(a.X, a.Y)
		if t == TTree || t == TBush || t == TDeadTree || t == TMushroom {
			return true
		}
		dm := g.snShelterMap(true)
		if v := dm[a.Y*MW+a.X]; v < 10 {
			g.stepToward(a, dm, false)
			return true
		}
	}
	return false
}

// snGreeting (talk.go greeting): what townsfolk and guards say while it storms.
func (g *Game) snGreeting(a *Actor) string {
	wx := g.snFoul()
	if wx == "" || a.AI == "follower" || g.Hostile(a) {
		return ""
	}
	wx = strings.ToLower(wx)
	switch {
	case a.AI == "guard":
		return fmt.Sprintf("\"%s or no %s, I keep my post.\" Water runs off their helm.", strings.ToUpper(wx[:1])+wx[1:], wx)
	case snShop(a) && !roofed(g.Cur.At(a.X, a.Y)):
		return fmt.Sprintf("\"Stalls are shut till this %s blows over. Come back when it clears.\"", wx)
	case a.AI == "town":
		return fmt.Sprintf("\"Get under a roof, stranger! Nobody walks the streets in a %s.\"", wx)
	}
	return ""
}

func snShop(a *Actor) bool {
	switch a.Role {
	case "merchant", "smith", "apothecary", "bookseller", "exotic", "tailor":
		return true
	}
	return false
}

// snMarketClosed (talk.go openTrade): outdoor stalls do not trade in foul weather.
func (g *Game) snMarketClosed(a *Actor) bool {
	if a == nil || !snShop(a) || g.snFoul() == "" || roofed(g.Cur.At(a.X, a.Y)) {
		return false
	}
	g.Talk, g.Mode = a, "talk"
	g.talkMenu(fmt.Sprintf("%s has lashed a cloth over the wares. \"Market's closed in a %s. Come back when it clears.\"", g.The(a), strings.ToLower(g.snFoul())))
	return true
}

// snShore (Music.Shore): open water fills a good part of this outdoor zone, for the surf sound.
func (g *Game) snShore() bool {
	z := g.Cur
	if g.OnWorld || z == nil || z.Z != 0 || z.Underwater || z.Sky {
		return false
	}
	n := 0
	for _, t := range z.T {
		if t == TDeep || t == TShallow {
			n++
		}
	}
	return n >= 120
}
