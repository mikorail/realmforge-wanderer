package engine

import (
	"fmt"
	"sort"
	"strings"
)

// purposerooms.go (prefix pr): in dungeons, towers, ruins, crypts and lairs some rooms get a job (kitchen,
// barracks, shrine, library, armory, cellar, prison, alchemy lab, throne room, crypt chapel), laid out with
// matching props from existing terrain, one small story detail and fitting loot. The kind follows the site
// theme and the creatures living there. A zone hook only: side rng NewRng(seed+":pr:"+zoneKey), never g.R.
// Every room is built as a batch of edits and rolled back when it would cut any walkable tile off from the
// rest of the level; stairs are never touched.

func init() {
	zoneHooks = append(zoneHooks, prZone)
}

type prRect struct{ x, y, w, h int }

type prOp struct {
	x, y int
	t    Terrain
}

type prBuild struct {
	g       *Game
	zn      *Zone
	r       *Rng
	rc      prRect
	closed  bool
	crude   bool
	ent     map[int]bool
	used    map[int]bool
	ops     []prOp
	items   []*Item
	actors  []*Actor
	floor   Terrain
	tags    string
	theme   string
	border  []int
	inner   []int
	storyAt int
}

func prPlain(t Terrain) bool {
	switch t {
	case TFloor, TDirt, TCarpet, TFungus, TMoss, TCrystalFloor, TTileFloor, TWoodFloor:
		return true
	}
	return false
}

// prSkipTheme: themes whose floors are water or cave growth, where a furnished room makes no sense.
var prSkipTheme = map[string]bool{"sunken": true, "abyss": true, "chasm": true}

func prZone(g *Game, zn *Zone, _ *Rng) {
	if g.W == nil || zn == nil || zn.Underwater {
		return
	}
	var maxRooms int
	closed := true
	switch zn.Kind {
	case "dungeon":
		maxRooms = 3
	case "tower":
		maxRooms = 2
	case "ruin":
		maxRooms, closed = 2, false
	case "lair":
		maxRooms, closed = 1, false
	case "crypt":
		maxRooms, closed = 1, false
	default:
		return
	}
	theme := zn.Kind
	if zn.Kind == "dungeon" || zn.Kind == "tower" {
		if s := g.siteAt(zn.WX, zn.WY, zn.ZX, zn.ZY); s != nil {
			theme = s.Theme
		}
	}
	if prSkipTheme[theme] {
		return
	}
	r := NewRng(g.W.Seed + ":pr:" + zn.Key)
	tags := prCreatureTags(zn)
	rooms := prFindRooms(zn, r, closed, 1+r.Int(0, maxRooms-1))
	if len(rooms) == 0 {
		return
	}
	taken := map[string]bool{}
	changed := false
	for _, rc := range rooms {
		kind := prPickKind(r, theme, zn, tags, taken)
		if kind == "" {
			continue
		}
		b := prNewBuild(g, zn, r, rc, closed, theme, tags)
		if b.build(kind) && b.commit() {
			taken[kind] = true
			changed = true
		}
	}
	if changed {
		g.connect(zn)
		g.computeLight(zn)
	}
}

// ---- creatures and kinds ----

func prCreatureTags(zn *Zone) string {
	var sb strings.Builder
	for _, a := range zn.Actors {
		sb.WriteString(strings.ToLower(a.Tpl + " " + a.Name + " " + a.Faction + " "))
	}
	return sb.String()
}

func prHas(tags string, words ...string) bool {
	for _, w := range words {
		if strings.Contains(tags, w) {
			return true
		}
	}
	return false
}

var prBase = map[string]map[string]int{
	"catacomb": {"chapel": 4, "shrine": 2, "cellar": 1, "prison": 1, "library": 1},
	"barrow":   {"chapel": 3, "shrine": 2, "cellar": 1},
	"mine":     {"barracks": 3, "kitchen": 3, "cellar": 2, "armory": 1},
	"hideout":  {"kitchen": 3, "barracks": 3, "cellar": 2, "prison": 2, "armory": 1, "throne": 1},
	"vault":    {"armory": 3, "alchemy": 2, "library": 2, "cellar": 1},
	"arcane":   {"alchemy": 4, "library": 4, "shrine": 1},
	"magic":    {"library": 4, "alchemy": 4, "shrine": 1, "throne": 1},
	"trials":   {"armory": 3, "barracks": 3, "throne": 1, "prison": 1, "shrine": 1},
	"ruin":     {"library": 2, "shrine": 2, "cellar": 2, "kitchen": 1, "armory": 1, "barracks": 1, "throne": 1},
	"lair":     {"kitchen": 4, "barracks": 3, "cellar": 1},
	"crypt":    {"chapel": 6, "shrine": 1},
}

func prPickKind(r *Rng, theme string, zn *Zone, tags string, taken map[string]bool) string {
	base := prBase[theme]
	if base == nil {
		base = map[string]int{"cellar": 2, "barracks": 1, "kitchen": 1, "shrine": 1}
	}
	w := map[string]int{}
	for k, v := range base {
		w[k] = v * 2
	}
	bump := func(n int, kinds ...string) {
		for _, k := range kinds {
			if _, ok := w[k]; ok {
				w[k] += n
			}
		}
	}
	if prHas(tags, "skeleton", "ghoul", "wight", "wraith", "ghost", "undead", "bone", "lich") {
		bump(4, "chapel", "shrine")
	}
	if prHas(tags, "bandit", "chief", "archer", "outlaw", "goblin", "orc", "ogre", "raider", "brute", "gnoll", "kobold") {
		bump(4, "kitchen", "barracks", "prison")
	}
	if prHas(tags, "golem", "turret", "machine", "drone", "automaton") {
		bump(4, "armory", "alchemy")
	}
	if prHas(tags, "wisp", "mage", "wizard", "witch", "shardmoth", "arcane") {
		bump(4, "alchemy", "library")
	}
	if prHas(tags, "cult", "priest", "acolyte", "faith", "inquisit") {
		bump(5, "shrine", "chapel")
	}
	// throne rooms only deeper in
	if zn.Z < 2 && zn.Z > -100 && zn.Kind == "dungeon" {
		delete(w, "throne")
	}
	var keys []string
	total := 0
	for k, v := range w {
		if taken[k] || v <= 0 {
			continue
		}
		keys = append(keys, k)
		total += v
	}
	if total == 0 {
		return ""
	}
	sort.Strings(keys)
	n := r.Int(0, total-1)
	for _, k := range keys {
		n -= w[k]
		if n < 0 {
			return k
		}
	}
	return keys[0]
}

// ---- finding rooms ----

func prFindRooms(zn *Zone, r *Rng, closed bool, want int) []prRect {
	var out []prRect
	occupied := map[int]bool{}
	for _, a := range zn.Actors {
		occupied[a.Y*MW+a.X] = true
	}
	for _, it := range zn.Items {
		occupied[it.Y*MW+it.X] = true
	}
	for tries := 0; tries < 500 && len(out) < want; tries++ {
		w, h := r.Int(5, 10), r.Int(3, 6)
		x, y := r.Int(2, MW-w-2), r.Int(2, MH-h-2)
		rc := prRect{x, y, w, h}
		if !prRectOK(zn, rc, closed, out) {
			continue
		}
		out = append(out, rc)
	}
	_ = occupied
	return out
}

func prRectOK(zn *Zone, rc prRect, closed bool, others []prRect) bool {
	for _, o := range others {
		if rc.x < o.x+o.w+2 && rc.x+rc.w+2 > o.x && rc.y < o.y+o.h+2 && rc.y+rc.h+2 > o.y {
			return false
		}
	}
	for y := rc.y - 2; y < rc.y+rc.h+2; y++ {
		for x := rc.x - 2; x < rc.x+rc.w+2; x++ {
			switch zn.At(x, y) {
			case TDown, TUp, TUpdraft, TPortal, TLava, TChasm, TDeep, TShallow:
				return false
			}
		}
	}
	for y := rc.y; y < rc.y+rc.h; y++ {
		for x := rc.x; x < rc.x+rc.w; x++ {
			if !prPlain(zn.At(x, y)) {
				return false
			}
		}
	}
	if !closed {
		return true
	}
	// enclosed: the ring just outside is mostly wall or rock
	solid, total := 0, 0
	for x := rc.x; x < rc.x+rc.w; x++ {
		for _, y := range []int{rc.y - 1, rc.y + rc.h} {
			total++
			if !zn.Walkable(x, y) {
				solid++
			}
		}
	}
	for y := rc.y; y < rc.y+rc.h; y++ {
		for _, x := range []int{rc.x - 1, rc.x + rc.w} {
			total++
			if !zn.Walkable(x, y) {
				solid++
			}
		}
	}
	return solid*10 >= total*6
}

// ---- building ----

func prNewBuild(g *Game, zn *Zone, r *Rng, rc prRect, closed bool, theme, tags string) *prBuild {
	b := &prBuild{g: g, zn: zn, r: r, rc: rc, closed: closed, theme: theme, tags: tags,
		ent: map[int]bool{}, used: map[int]bool{}, floor: zn.At(rc.x, rc.y)}
	b.crude = theme == "lair" || theme == "mine" || theme == "hideout" || prHas(tags, "goblin", "orc", "ogre", "gnoll", "kobold", "troglodyte", "bandit", "troll")
	for y := rc.y; y < rc.y+rc.h; y++ {
		for x := rc.x; x < rc.x+rc.w; x++ {
			if zn.ActorAt(x, y) != nil || len(zn.ItemsAt(x, y)) > 0 {
				b.used[y*MW+x] = true
			}
			if x == rc.x || y == rc.y || x == rc.x+rc.w-1 || y == rc.y+rc.h-1 {
				b.border = append(b.border, y*MW+x)
			} else {
				b.inner = append(b.inner, y*MW+x)
			}
		}
	}
	// entrances: border tiles with open ground just outside stay clear, with the tile inward of them
	if closed {
		for _, i := range b.border {
			x, y := i%MW, i/MW
			for _, d := range [][2]int{{0, -1}, {0, 1}, {-1, 0}, {1, 0}} {
				ox, oy := x+d[0], y+d[1]
				if prIn(rc, ox, oy) || !zn.Walkable(ox, oy) {
					continue
				}
				b.ent[i] = true
				b.ent[(y-d[1])*MW+x-d[0]] = true
			}
		}
	}
	// stable shuffles of the tile lists
	shuffle := func(a []int) {
		for i := len(a) - 1; i > 0; i-- {
			j := r.Int(0, i)
			a[i], a[j] = a[j], a[i]
		}
	}
	shuffle(b.border)
	shuffle(b.inner)
	return b
}

func prIn(rc prRect, x, y int) bool {
	return x >= rc.x && y >= rc.y && x < rc.x+rc.w && y < rc.y+rc.h
}

func (b *prBuild) free(i int) bool {
	return !b.used[i] && !b.ent[i] && prPlain(b.zn.T[i])
}

// prop puts terrain t on a free tile from the list (wall side first); false when none is left.
func (b *prBuild) prop(t Terrain, from []int) bool {
	for _, i := range from {
		if b.free(i) {
			b.used[i] = true
			b.ops = append(b.ops, prOp{i % MW, i / MW, t})
			return true
		}
	}
	return false
}

func (b *prBuild) walls(t Terrain, n int) int {
	c := 0
	for k := 0; k < n; k++ {
		if b.prop(t, b.border) {
			c++
		}
	}
	return c
}

func (b *prBuild) middle(t Terrain, n int) int {
	c := 0
	for k := 0; k < n; k++ {
		if b.prop(t, b.inner) {
			c++
		}
	}
	return c
}

// at puts t at a fixed tile when free.
func (b *prBuild) at(t Terrain, x, y int) bool {
	if !prIn(b.rc, x, y) {
		return false
	}
	return b.prop(t, []int{y*MW + x})
}

// layFloor repaves every free tile of the room (walkable to walkable only).
func (b *prBuild) layFloor(t Terrain) {
	for y := b.rc.y; y < b.rc.y+b.rc.h; y++ {
		for x := b.rc.x; x < b.rc.x+b.rc.w; x++ {
			i := y*MW + x
			if prPlain(b.zn.T[i]) && !b.used[i] {
				b.ops = append(b.ops, prOp{x, y, t})
			}
		}
	}
}

// drop places an item on a free tile that stays walkable (not already a prop).
func (b *prBuild) drop(it *Item) bool {
	if it == nil {
		return false
	}
	for _, i := range b.inner {
		if b.free(i) {
			b.used[i] = true
			it.X, it.Y = i%MW, i/MW
			b.items = append(b.items, it)
			return true
		}
	}
	for _, i := range b.border {
		if b.free(i) {
			b.used[i] = true
			it.X, it.Y = i%MW, i/MW
			b.items = append(b.items, it)
			return true
		}
	}
	return false
}

func (b *prBuild) gold(n int) {
	b.drop(&Item{ID: b.g.newID(), Kind: "gold", Name: b.g.Sk.Currency, Glyph: '$', F: CYellow, Count: n})
}

func (b *prBuild) tier() int { return clampi(b.zn.Level/3+1, 1, 5) }

// guard adds one inhabitant that fits the zone (one of its own templates, or the given fallback).
func (b *prBuild) guard(fallback string) {
	id := fallback
	var pool []string
	for _, a := range b.zn.Actors {
		if a.HP > 0 && !a.Boss && tplByID[a.Tpl] != nil && a.AI == "hostile" || a.HP > 0 && !a.Boss && tplByID[a.Tpl] != nil && a.AI == "wander" {
			pool = append(pool, a.Tpl)
		}
	}
	if len(pool) > 0 && b.r.Chance(0.7) {
		id = Pick(b.r, pool)
	}
	if tplByID[id] == nil {
		return
	}
	a := b.g.Spawn(id, b.zn.Level, b.r)
	for _, i := range b.inner {
		if b.free(i) {
			b.used[i] = true
			a.X, a.Y = i%MW, i/MW
			b.actors = append(b.actors, a)
			return
		}
	}
}

// ---- story details ----

func (b *prBuild) person() string {
	w := b.g.W
	if len(w.People) == 0 {
		return "someone"
	}
	return w.People[b.r.Int(0, len(w.People)-1)].Name
}

// story adds one detail: a carved line (stone) or a scrap of writing lying in the room.
func (b *prBuild) story(stone bool, title string, lines []string) {
	text := Pick(b.r, lines)
	text = strings.NewReplacer("{who}", b.person(), "{coin}", b.g.Sk.Currency).Replace(text)
	if stone {
		for _, i := range b.border {
			if b.free(i) {
				b.used[i] = true
				b.ops = append(b.ops, prOp{i % MW, i / MW, TInscription})
				b.storyAt = i + 1
				b.zn.Notes[i] = text
				return
			}
		}
	}
	b.drop(b.g.MakeBook(title, text))
}

// ---- the rooms ----

func (b *prBuild) build(kind string) bool {
	g, r := b.g, b.r
	t := b.tier()
	w := b.rc.w
	name := ""
	switch kind {
	case "kitchen":
		name = "kitchen"
		if !b.crude {
			b.layFloor(TTileFloor)
		}
		b.walls(TBrazier, 1)
		b.walls(TBarrel, 2+r.Int(0, 1))
		b.walls(TShelf, 1+r.Int(0, 1))
		b.middle(TStall, 1) // the chopping table
		if b.crude {
			b.middle(TBones, 2+r.Int(0, 2))
			b.story(false, "greasy tally-board", []string{
				"Scratched on a board: one mark for every {who} we did not eat. The marks are many.",
				"Whoever cooked here wrote 'more salt, less screaming' on the lid of the pot.",
				"A list in a clumsy hand: bread, bones, boots, and whatever the last boot belonged to."})
			b.drop(g.makeIngredient("scorched marrow-bone", "hearty", 2, CWhite))
			b.drop(g.makeIngredient("smoked offal", "lasting", 2, CBrown))
		} else {
			b.story(false, "stained recipe card", []string{
				"A recipe in a neat hand, with a note in the margin: {who} liked it with less pepper.",
				"The cook's ledger: flour short again, and someone has been at the good wine.",
				"A grease-spotted menu for a feast that was never served."})
			b.drop(g.makeIngredient("dried peppers", "ember", 2, CRed))
			b.drop(g.makeIngredient("pickled roots", "mending", 2, CGreen))
			b.drop(g.makeIngredient("salted haunch", "lasting", 2, CBrown))
		}
		b.drop(g.MakeFood(r, 2))
	case "barracks":
		name = "barracks"
		if !b.crude {
			b.layFloor(TWoodFloor)
		}
		b.walls(TBed, 2+w/3)
		b.walls(TShelf, 2) // weapon racks
		b.walls(TCrate, 1)
		if b.crude {
			b.middle(TBones, 1+r.Int(0, 1))
		}
		b.story(false, "bunk-list", []string{
			"A roster nailed to a post. Half the names are crossed out. {who} is written at the bottom, twice.",
			"Under a pillow, a letter never sent: 'tell my mother the pay is good and the food is not.'",
			"Dice scratched into a bedpost, and a debt of fifty {coin} chalked beside them."})
		b.drop(g.MakeWeapon(Pick(r, []string{"longblade", "shortblade", "axe", "cudgel"}), clampi(t-1, 0, 5), r))
		b.gold(8 * t * (1 + r.Int(0, 2)))
		b.guard("bandit")
	case "shrine":
		name = "shrine"
		b.layFloor(TCarpet)
		cx := b.rc.x + w/2
		b.at(TAltar, cx, b.rc.y)
		b.at(TBrazier, cx-2, b.rc.y)
		b.at(TBrazier, cx+2, b.rc.y)
		b.walls(TLantern, 2) // candles
		b.story(true, "", []string{
			"Carved above the altar: 'Light a candle for {who}, who did not come back.'",
			"The faithful have scratched their prayers here, layer over layer, until the stone is grey with them.",
			"A single line, freshly cut: the candles must never go out."})
		b.gold(6 * t * (1 + r.Int(0, 2)))
		b.drop(g.Loot(r, t))
	case "library":
		name = "library"
		b.layFloor(TWoodFloor)
		b.walls(TShelf, 3+w/3)
		b.middle(TStall, 1) // the reading desk
		b.story(false, "marginal note", []string{
			"A page with the margin full of angry notes in two hands, arguing over the death of {who}.",
			"A reader's slip: 'borrowed, not stolen, will return after the war.' It is old.",
			"A half-finished copy of a history, ending mid-sentence. The ink is still faintly wet."})
		b.drop(g.MakeBook("a bound chronicle", fmt.Sprintf("A chronicle of the wars of the realm, with %s mentioned often and rarely kindly.", b.person())))
		b.drop(g.Loot(r, t))
	case "armory":
		name = "armory"
		b.layFloor(TTileFloor)
		b.walls(TShelf, 3) // racks
		b.walls(TCrate, 2)
		b.walls(TAnvil, 1)
		b.story(false, "quartermaster's chit", []string{
			"A requisition: forty blades signed out, thirty-one returned. The rest were 'lost in the usual way'.",
			"A smith's note: 'the left greave is not mine, whatever {who} says.'",
			"An inventory pinned to the rack, every line ticked but the last."})
		b.drop(g.MakeWeapon(Pick(r, []string{"longblade", "axe", "cudgel"}), t, r))
		b.drop(g.MakeArmor(Pick(r, []string{"body", "head", "hands"}), clampi(t-1, 0, 5), r))
		b.guard("golem")
	case "cellar":
		name = "cellar"
		b.walls(TBarrel, 3+r.Int(0, 2))
		b.walls(TCrate, 2)
		b.walls(TShelf, 1)
		b.story(false, "cellar book", []string{
			"A cellar ledger: wine, oil, salt, and one entry that says only 'do not open the blue cask'.",
			"Someone has chalked 'it was {who}' on every barrel. Whatever it was, it was thorough.",
			"A tally of casks in a shaking hand, the last four lines crossed out."})
		b.drop(g.MakeFood(r, 2+r.Int(0, 2)))
		b.drop(g.MakeWater())
		if r.Chance(0.6) {
			b.drop(g.Loot(r, t))
		}
	case "prison":
		name = "prison"
		// cells: short walls of bars down from the north side, open to the south
		for x := b.rc.x + 2; x < b.rc.x+w-1; x += 3 {
			b.at(TMetalWall, x, b.rc.y)
			b.at(TMetalWall, x, b.rc.y+1)
			b.at(TBones, x-1, b.rc.y)
		}
		b.walls(TBed, 1)
		b.story(true, "", []string{
			"Scratched into the wall, a tally of days. It stops at four hundred and eleven.",
			"A name carved low on the wall: {who}. Beside it, 'I did not do it.'",
			"A child's drawing of a sun, scratched into the stone where the light reaches for an hour a day."})
		b.drop(g.Loot(r, t))
		b.gold(5 * t)
	case "alchemy":
		name = "alchemy lab"
		b.layFloor(TTileFloor)
		b.walls(TMachine, 1) // the still
		b.walls(TBrazier, 1)
		b.walls(TShelf, 2)
		b.middle(TStall, 1)
		b.story(false, "splattered notebook", []string{
			"Lab notes: 'Batch nine did not explode. Batch ten did.' The page is scorched.",
			"A formula struck through with a single remark from {who}: 'this will kill you.'",
			"A shopping list for reagents, ending: 'and a larger door, and a new apprentice.'"})
		for _, tag := range []string{"keen", "mending", "venom"} {
			if r.Chance(0.6) {
				b.drop(g.makeIngredient("reagent "+tag+" root", tag, 1+r.Int(0, 1), CMagenta))
			}
		}
		b.drop(g.Loot(r, t))
	case "throne":
		name = "throne room"
		b.layFloor(TCarpet)
		cx := b.rc.x + w/2
		b.at(TThrone, cx, b.rc.y)
		b.at(TBrazier, cx-2, b.rc.y)
		b.at(TBrazier, cx+2, b.rc.y)
		b.walls(TPillar, 2)
		b.story(true, "", []string{
			"Cut into the dais: 'Here sat {who}, and here the whole realm learned what that cost.'",
			"The throne's arm is worn smooth on one side and gouged on the other.",
			"An oath of fealty, chiselled deep, and below it a second oath, chiselled over the first."})
		b.gold(14 * t * (1 + r.Int(0, 2)))
		b.drop(g.Loot(r, t+1))
		b.guard("skeleton")
	case "chapel":
		name = "crypt chapel"
		b.layFloor(TCarpet)
		cx := b.rc.x + w/2
		b.at(TAltar, cx, b.rc.y)
		b.at(TBrazier, cx-2, b.rc.y)
		b.at(TBrazier, cx+2, b.rc.y)
		b.walls(TSarcophagus, 2+r.Int(0, 1))
		b.walls(TLantern, 1)
		b.middle(TBones, 1)
		b.story(true, "", []string{
			"An epitaph: 'Beloved of {who}, who set this stone. Let none lift the lid.'",
			"The chapel names the dead in a long column, and the last four names have been scratched out.",
			"A prayer for rest, and under it, in a different hand: 'it did not take.'"})
		b.gold(7 * t * (1 + r.Int(0, 2)))
		b.drop(g.Loot(r, t))
		b.guard("skeleton")
	default:
		return false
	}
	if len(b.ops) < 3 {
		return false
	}
	_ = name
	return true
}

// commit applies the batch if no walkable tile loses its connection, else leaves the zone untouched.
func (b *prBuild) commit() bool {
	zn := b.zn
	changed := map[int]bool{}
	blocks := 0
	for _, o := range b.ops {
		i := o.y*MW + o.x
		if !o.t.D().Walk {
			changed[i] = true
			blocks++
		}
	}
	// anchor: a walkable tile of the room that stays open, or just outside it
	anchor := -1
	for y := b.rc.y - 1; y <= b.rc.y+b.rc.h && anchor < 0; y++ {
		for x := b.rc.x - 1; x <= b.rc.x+b.rc.w; x++ {
			i := y*MW + x
			if zn.Walkable(x, y) && !changed[i] && zn.At(x, y) != TLava && zn.At(x, y) != TChasm {
				anchor = i
				break
			}
		}
	}
	if anchor < 0 {
		return false
	}
	before := b.g.reachable(zn, anchor%MW, anchor/MW)
	expect := 0
	for i, ok := range before {
		if ok && !changed[i] {
			expect++
		}
	}
	old := make([]Terrain, len(b.ops))
	for k, o := range b.ops {
		old[k] = zn.At(o.x, o.y)
		zn.Set(o.x, o.y, o.t)
	}
	after := b.g.reachable(zn, anchor%MW, anchor/MW)
	got := 0
	for _, ok := range after {
		if ok {
			got++
		}
	}
	// stairs and the level edges must still be in the same region as before
	stairsOK := true
	for i, t := range zn.T {
		if (t == TDown || t == TUp || t == TUpdraft) && before[i] && !after[i] {
			stairsOK = false
		}
	}
	if got < expect || !stairsOK {
		for k := len(b.ops) - 1; k >= 0; k-- {
			zn.Set(b.ops[k].x, b.ops[k].y, old[k])
		}
		for i := range zn.Notes {
			if b.storyAt > 0 && i == b.storyAt-1 {
				delete(zn.Notes, i)
			}
		}
		return false
	}
	zn.Items = append(zn.Items, b.items...)
	zn.Actors = append(zn.Actors, b.actors...)
	return true
}
