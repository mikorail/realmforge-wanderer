package engine

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// Cross-genre incursions (prefix ci). Through rift gates (planets.go) and during rift storms
// (world_story.go) strangers from other genres arrive in this world with out-of-genre gear and
// speech. Per planet pair 1-3 arrivals come over time from a day-clock turn hook (side rng
// seed:ci:..., never g.R at generation). Roles: invader, refugee, lost explorer, merchant of
// strange goods, hunter pursuing someone, cult prophet. Their gear is made with the ordinary Item
// constructors under the origin genre's Skin and tagged Extra["alien"] (plus "alientech" or
// "alienmagic", which misfire in a world of the other class: see ciMisfire). Quests (journal Kind
// "ci"): escort a refugee/explorer to the rift gate, recover a stolen artefact for a hunter, stop an
// invader. State g.ext("ci"), planet-local ("ci" in unLocalKeys).

func init() {
	unLocalKeys = append(unLocalKeys, "ci")
	turnHooks = append(turnHooks, ciTurn)
	talkHooks = append(talkHooks, ciTalk)
	killHooks = append(killHooks, ciKill)
	sideHooks = append(sideHooks, ciSide)
}

type ciArr struct {
	ID       int    `json:"id"`
	Pair     string `json:"pr"`
	G        int    `json:"g"`
	Home     string `json:"home"`
	Role     string `json:"ro"`
	Name     string `json:"n"`
	Day      int    `json:"d"`
	Key      string `json:"k"`
	P        int    `json:"p"`
	State    string `json:"st,omitempty"` // "" here, dead, home, safe, left, lost, taken
	Quest    int    `json:"q,omitempty"`
	Favour   string `json:"fv,omitempty"`
	Link     int    `json:"ln,omitempty"`
	Word     int    `json:"w"`
	Seen     bool   `json:"sn,omitempty"`
	Taught   bool   `json:"tg,omitempty"`
	Explain  bool   `json:"ex,omitempty"`
	Esc      bool   `json:"esc,omitempty"`
	Raided   bool   `json:"rd,omitempty"`
	Hostile  bool   `json:"ho,omitempty"`
	Joined   bool   `json:"jn,omitempty"`
	Resolved bool   `json:"rs,omitempty"`
	Hidden   bool   `json:"hid,omitempty"`
	Gate     bool   `json:"gt,omitempty"`
}

type ciPair struct {
	Total int `json:"t"`
	Done  int `json:"d"`
	Next  int `json:"n"`
	Pl    int `json:"pl"` // origin planet index, -1 for storm-only pairs
	G     int `json:"g"`
}

type ciNews struct {
	Day  int    `json:"d"`
	P    int    `json:"p"`
	Text string `json:"t"`
}

type ciState struct {
	Arr    []*ciArr           `json:"a,omitempty"`
	Pairs  map[string]*ciPair `json:"pr,omitempty"`
	Last   int                `json:"l"`
	NextID int                `json:"nid"`
	Words  []string           `json:"w,omitempty"`
	News   []ciNews           `json:"nw,omitempty"`
	Unrest int                `json:"u,omitempty"`
}

func (g *Game) ci() *ciState {
	st := g.ext("ci", func() any { return &ciState{} }).(*ciState)
	if st.Pairs == nil {
		st.Pairs = map[string]*ciPair{}
	}
	return st
}

func ciNorm(genre string) string {
	if _, ok := ciGenreIdx[genre]; ok {
		return genre
	}
	return "fantasy"
}

func ciClass(genre string) string { return ciGenres[ciGenreIdx[ciNorm(genre)]].Class }

func ciColor(gi int) byte {
	return []byte{CWhite, CYellow, COrange, CBrown, CGreen, CMagenta, CDarkGreen, CCyan, CGray, CDarkMagenta}[gi%10]
}

// ---- gear ----

// ciWith builds items under another genre's skin.
func ciWith(g *Game, gi int, f func()) {
	old := g.Sk
	g.Sk = SkinFor(ciGenres[gi].ID)
	defer func() { g.Sk = old }()
	f()
}

func ciTag(it *Item, gi int) *Item {
	if it.Extra == nil {
		it.Extra = map[string]int{}
	}
	cg := &ciGenres[gi]
	it.Extra["alien"] = 1
	it.Extra["ciog"] = gi + 1
	switch {
	case cg.Class == "tech" && (it.Kind == "ranged" || it.Kind == "artifact" || it.Kind == "light"):
		it.Extra["alientech"] = 1
	case cg.Class == "magic" && (it.Kind == "weapon" || it.Kind == "artifact"):
		it.Extra["alienmagic"] = 1
	}
	if it.Lore == "" {
		it.Lore = "Made in a " + cg.ID + " world; it does not belong here."
	} else {
		it.Lore += " (From a " + cg.ID + " world.)"
	}
	return it
}

// ciAlien reports whether an item came from another genre (inventory hook: show ciMark).
func ciAlien(it *Item) bool { return it != nil && it.Extra != nil && it.Extra["alien"] > 0 }

func ciMark(it *Item) string {
	if !ciAlien(it) {
		return ""
	}
	if og := it.Extra["ciog"] - 1; og >= 0 && og < len(ciGenres) {
		return "{{M|alien: " + ciGenres[og].ID + "}}"
	}
	return "{{M|alien}}"
}

// ciMisfire: an alien-tech item fizzles half the time in a non-tech world, alien magic a third of the
// time in a tech world. Hook it where a ranged weapon or charged artefact fires. Draws g.R (play time).
func ciMisfire(g *Game, it *Item) bool {
	if !ciAlien(it) {
		return false
	}
	cls := ciClass(g.Genre)
	bad := it.Extra["alientech"] > 0 && cls != "tech" && g.R.Chance(0.5) ||
		it.Extra["alienmagic"] > 0 && cls == "tech" && g.R.Chance(0.35)
	if bad {
		g.Msg("{{R|The " + it.Name + " sputters and fizzles; it does not work right in this world's air.}}")
	}
	return bad
}

func ciWeapon(g *Game, gi int, r *Rng, tier int) *Item {
	cg := &ciGenres[gi]
	var it *Item
	ciWith(g, gi, func() {
		if cg.Ranged {
			it = g.MakeRanged(tier, r)
		} else {
			it = g.MakeWeapon(cg.WSkill, tier, r)
		}
	})
	it.Name = cg.Weapon
	it.Elem = ""
	it.Value = it.Value*3 + 40
	return ciTag(it, gi)
}

func ciArmor(g *Game, gi int, r *Rng, slot string, tier int) *Item {
	var it *Item
	ciWith(g, gi, func() { it = g.MakeArmor(slot, tier, r) })
	return ciTag(it, gi)
}

func ciOddity(g *Game, gi int, r *Rng, tier int) *Item {
	cg := &ciGenres[gi]
	it := g.MakeOddity(r, tier)
	it.Name = Pick(r, cg.Goods)
	it.Effect = Pick(r, cg.Goods2)
	it.Lore = "A thing out of " + cg.ID + " lands."
	return ciTag(it, gi)
}

func ciBook(g *Game, gi int, r *Rng) *Item {
	cg := &ciGenres[gi]
	it := g.MakeBook("Primer of the "+cg.Folk, Pick(r, cg.Creed)+" "+Pick(r, cg.Greets)+" Words: "+cg.Words[0][0]+" ("+cg.Words[0][1]+"); "+cg.Words[1][0]+" ("+cg.Words[1][1]+").")
	return ciTag(it, gi)
}

// ciGoods builds n assorted out-of-genre items.
func ciGoods(g *Game, gi int, r *Rng, n, tier int) []*Item {
	var out []*Item
	for i := 0; i < n; i++ {
		var it *Item
		switch r.Int(0, 8) {
		case 0:
			it = ciWeapon(g, gi, r, tier)
		case 1:
			it = ciArmor(g, gi, r, Pick(r, []string{"body", "head", "hands", "feet", "back"}), tier)
		case 2:
			ciWith(g, gi, func() {
				it = ciTag(g.MakeTonic(Pick(r, []string{"heal", "salve", "haste", "strength", "antidote"}), r.Int(1, 2)), gi)
			})
		case 3:
			ciWith(g, gi, func() { it = ciTag(g.MakeGrenade(Pick(r, []string{"fire", "frost", "flash"}), r.Int(1, 3)), gi) })
		case 4:
			ciWith(g, gi, func() { it = ciTag(g.MakeFood(r, r.Int(2, 4)), gi) })
		case 5:
			ciWith(g, gi, func() { it = ciTag(g.MakeLight(r.Int(0, 2)), gi) })
		case 6:
			it = ciBook(g, gi, r)
		default:
			it = ciOddity(g, gi, r, tier)
		}
		out = append(out, it)
	}
	return out
}

// ---- building strangers ----

func ciSub(s string, arr *ciArr, g *Game) string {
	cg := &ciGenres[arr.G]
	w := cg.Words[arr.Word%len(cg.Words)]
	return strings.NewReplacer("{n}", arr.Name, "{home}", arr.Home, "{w}", w[0], "{m}", w[1], "{cur}", g.Sk.Currency).Replace(s)
}

func ciArticle(s string) string {
	if s != "" && strings.ContainsRune("aeiouAEIOU", rune(s[0])) {
		return "an " + s
	}
	return "a " + s
}

func ciMake(g *Game, r *Rng, arr *ciArr, lvl int, minion bool) *Actor {
	cg := &ciGenres[arr.G]
	role := arr.Role
	a := g.Spawn("bandit", lvl, r)
	a.Flags = map[string]bool{"ciS": true, fmt.Sprintf("ci#%d", arr.ID): true}
	if minion {
		a.Set("ciMin")
	}
	a.Inv = nil
	for _, p := range a.Body {
		p.Item = nil
	}
	a.Gold = 0
	a.Glyph, a.F = '@', ciColor(arr.G)
	a.Proper = !minion
	if minion {
		a.Name = cg.Folk + " trooper"
		a.Title = ""
	} else {
		a.Name = arr.Name
		a.Title = cg.Folk
	}
	a.Faction = "strangers"
	a.Species = "human"
	tier := clampi(lvl/3, 0, 5)
	if p := a.PartFor("hand"); p != nil {
		p.Item = ciWeapon(g, arr.G, r, tier)
	}
	if p := a.PartFor("body"); p != nil {
		p.Item = ciArmor(g, arr.G, r, "body", tier)
	}
	a.Desc = fmt.Sprintf("%s from %s, with %s.", titleCase(ciArticle(cg.Folk)), arr.Home, Pick(r, cg.Looks))
	a.AI = "wander"
	switch role {
	case "invader":
		a.AI, a.Angry = "hostile", true
		a.MaxHP += 10 + lvl*3
		a.XP += 40
		a.Gold = r.Int(30, 120)
		a.Inv = ciGoods(g, arr.G, r, r.Int(1, 3), tier)
	case "thief":
		a.AI, a.Angry = "hostile", true
		a.MaxHP += lvl * 2
		a.XP += 25
	case "refugee":
		a.MaxHP = a.MaxHP * 2 / 3
		a.Gold = r.Int(0, 15)
		a.Inv = ciGoods(g, arr.G, r, 1, tier)
		a.Desc += " They look hunted and exhausted."
	case "explorer":
		a.Gold = r.Int(10, 60)
		a.Inv = ciGoods(g, arr.G, r, r.Int(1, 2), tier)
		a.Inv = append(a.Inv, ciBook(g, arr.G, r))
	case "merchant":
		a.AI, a.Role = "still", "merchant"
		a.Gold = r.Int(150, 420) + lvl*20
		a.Inv = ciGoods(g, arr.G, r, r.Int(5, 7), tier+1)
		odd := []float64{0.15, 0.5, 1, 2, 3.5, 7}
		for _, it := range a.Inv {
			it.Value = maxi(1, int(float64(maxi(it.Value, 4))*odd[r.Int(0, len(odd)-1)]))
		}
		a.Desc += " Their goods are laid out on a cloth with no price on any of them."
	case "hunter":
		a.MaxHP += 8 + lvl*2
		a.XP += 30
		a.Inv = ciGoods(g, arr.G, r, 1, tier)
		a.Set("ciHunter")
	case "prophet":
		a.AI = "still"
		a.Gold = r.Int(5, 40)
		a.Inv = []*Item{ciOddity(g, arr.G, r, tier+1)}
		a.Desc += " They speak softly to the empty air."
	}
	a.HP = a.MaxHP
	return a
}

func ciGates(g *Game, pl int) [][3]int {
	u := unUni(g.W)
	var out [][3]int
	seen := map[int]bool{}
	for burg, ris := range u.ends[g.W.uniIdx] {
		for _, ri := range ris {
			o := unOther(u.Rifts[ri], g.W.uniIdx, burg)
			if pl >= 0 && o.Planet != pl || seen[burg] {
				continue
			}
			bd := g.W.burgByID[burg]
			if bd == nil || bd.Layer == "sea" || bd.Layer == "sky" {
				continue
			}
			seen[burg] = true
			wx, wy, z := unTownZone(g.W, bd)
			out = append(out, [3]int{wx, wy, z})
		}
	}
	sort.Slice(out, func(i, j int) bool {
		for k := 0; k < 3; k++ {
			if out[i][k] != out[j][k] {
				return out[i][k] < out[j][k]
			}
		}
		return false
	})
	return out
}

func ciStorm(g *Game) bool {
	n := maxi(1, len(unUni(g.W).Rifts))
	for ri := 0; ri < n; ri++ {
		if wyIsRiftStorm(g, ri) {
			return true
		}
	}
	return false
}

func ciInit(g *Game, st *ciState) {
	if len(st.Pairs) > 0 {
		return
	}
	u := unUni(g.W)
	cur := ciNorm(g.Genre)
	add := func(key string, pl int, genre string) {
		if genre == cur {
			cs := []string{}
			for _, cg := range ciGenres {
				if cg.ID != cur {
					cs = append(cs, cg.ID)
				}
			}
			genre = cs[int(HashStr(g.Seed+":ci:g:"+key)%uint64(len(cs)))]
		}
		h := HashStr(g.Seed + ":ci:n:" + key)
		st.Pairs[key] = &ciPair{Total: 1 + int(h%3), Next: 2 + int((h>>8)%6), Pl: pl, G: ciGenreIdx[genre]}
	}
	for i := range u.Planets {
		if i != g.W.uniIdx {
			add(fmt.Sprintf("p%d", i), i, ciNorm(u.Planets[i].Genre))
		}
	}
	if len(u.Planets) <= 1 {
		add("storm", -1, "")
	}
}

func ciHome(g *Game, pl, gi int) string {
	u := unUni(g.W)
	if pl >= 0 && pl < len(u.Planets) {
		return u.planetLabel(pl)
	}
	return "a " + ciGenres[gi].ID + " world beyond the sky"
}

func ciName(r *Rng, gi int) string {
	cg := &ciGenres[gi]
	return Pick(r, cg.Given) + " " + Pick(r, cg.Family)
}

// ciSite picks the zone: next to a rift gate, else a wild surface zone near the player.
func ciSite(g *Game, r *Rng, gates [][3]int) (*Zone, bool) {
	if len(gates) > 0 && r.Chance(0.7) {
		gt := Pick(r, gates)
		if zn := g.GetZone(gt[0], gt[1], 1, 1, gt[2]); zn != nil && !zn.Underwater && !zn.Sky {
			if _, _, ok := g.unGate(zn); ok {
				return zn, true
			}
		}
	}
	for i := 0; i < 16; i++ {
		wx, wy := g.WX+r.Int(-3, 3), g.WY+r.Int(-3, 3)
		if !g.W.In(wx, wy) {
			continue
		}
		if zn := g.GetZone(wx, wy, r.Int(0, 2), r.Int(0, 2), 0); zn != nil && zn.Kind == "surface" {
			return zn, false
		}
	}
	return nil, false
}

func ciPlace(g *Game, r *Rng, zn *Zone, a *Actor, atGate bool) bool {
	var near func(x, y int) bool
	if gx, gy, ok := g.unGate(zn); atGate && ok {
		near = func(x, y int) bool { return abs(x-gx) <= 5 && abs(y-gy) <= 4 && (abs(x-gx) > 1 || abs(y-gy) > 1) }
	}
	x, y, ok := zn.FreeSpot(r, near)
	if !ok {
		return false
	}
	a.X, a.Y = x, y
	a.HomeX, a.HomeY = x, y
	zn.Actors = append(zn.Actors, a)
	return true
}

func ciSign(g *Game, r *Rng, zn *Zone, a *Actor, arr *ciArr) {
	cg := &ciGenres[arr.G]
	x, y, ok := zn.FreeSpot(r, func(x, y int) bool { return abs(x-a.X) <= 3 && abs(y-a.Y) <= 3 && (x != a.X || y != a.Y) })
	if !ok {
		return
	}
	zn.Set(x, y, TRift)
	zn.Notes[y*MW+x] = fmt.Sprintf("A tear in the sky: the air is scarred here, like a healed cut, and smells of %s.", cg.Smell)
}

func ciNewsAdd(g *Game, st *ciState, p int, text string) {
	st.News = append(st.News, ciNews{g.Day(), p, text})
	if len(st.News) > 24 {
		st.News = st.News[len(st.News)-24:]
	}
}

// ciNewsLines feeds the world news (hook in wsNewsLines).
func (g *Game) ciNewsLines() []wsLine {
	if g.W == nil {
		return nil
	}
	st := g.ci()
	var out []wsLine
	for i := len(st.News) - 1; i >= 0 && len(out) < 3; i-- {
		n := st.News[i]
		if g.Day()-n.Day <= 20 {
			out = append(out, wsLine{n.P, "{{M|Strangers:}} " + n.Text})
		}
	}
	return out
}

func ciByID(st *ciState, id int) *ciArr {
	for _, a := range st.Arr {
		if a.ID == id {
			return a
		}
	}
	return nil
}

func ciArrOf(g *Game, a *Actor) *ciArr {
	if a == nil || !a.Has("ciS") {
		return nil
	}
	for k := range a.Flags {
		if strings.HasPrefix(k, "ci#") {
			id, _ := strconv.Atoi(k[3:])
			return ciByID(g.ci(), id)
		}
	}
	return nil
}

func ciFind(g *Game, arr *ciArr) (*Zone, *Actor) {
	key := fmt.Sprintf("ci#%d", arr.ID)
	for _, zn := range g.Zones {
		for _, a := range zn.Actors {
			if a.Has(key) && !a.Has("ciMin") {
				return zn, a
			}
		}
	}
	return nil, nil
}

func ciRemove(g *Game, a *Actor) {
	for _, zn := range g.Zones {
		for i, o := range zn.Actors {
			if o == a {
				zn.Actors = append(zn.Actors[:i], zn.Actors[i+1:]...)
				break
			}
		}
	}
	if g.Talk == a {
		g.Mode, g.Talk = "main", nil
	}
}

// ciArrive creates one stranger on the planet.
func ciArrive(g *Game, st *ciState, key string, p *ciPair) {
	r := NewRng(fmt.Sprintf("%s:ci:arr:%s:%d", g.Seed, key, p.Done))
	gates := ciGates(g, p.Pl)
	zn, atGate := ciSite(g, r, gates)
	if zn == nil {
		return
	}
	st.NextID++
	role := ciRoles[r.Int(0, len(ciRoles)-1)]
	arr := &ciArr{ID: st.NextID, Pair: key, G: p.G, Home: ciHome(g, p.Pl, p.G), Role: role, Name: ciName(r, p.G), Day: g.Day(), Key: zn.Key,
		P: g.W.Idx(zn.WX, zn.WY), Word: r.Int(0, 1), Gate: atGate}
	lvl := maxi(1, g.zoneLevel(zn.WX, zn.WY, 0)+boolInt(role == "invader" || role == "hunter")*2)
	a := ciMake(g, r, arr, lvl, false)
	if !ciPlace(g, r, zn, a, atGate) {
		return
	}
	ciSign(g, r, zn, a, arr)
	if role == "invader" {
		for i := 0; i < r.Int(1, 2); i++ {
			m := ciMake(g, r, arr, maxi(1, lvl-1), true)
			if !ciPlace(g, r, zn, m, atGate) {
				break
			}
		}
	}
	st.Arr = append(st.Arr, arr)
	where := "out in " + zn.Name
	if atGate {
		where = "near the rift gate of " + zn.Name
	}
	ciNewsAdd(g, st, arr.P, fmt.Sprintf("The sky tore open %s and %s %s, %s, stepped out.", where, ciArticle(ciGenres[arr.G].Folk), arr.Name, "from "+arr.Home))
	if abs(zn.WX-g.WX) <= 2 && abs(zn.WY-g.WY) <= 2 {
		g.Msg("{{M|Far off, the sky tears with a sound like torn cloth, and then is still.}}")
	}
}

func ciDaily(g *Game, st *ciState, d int) {
	ciInit(g, st)
	keys := make([]string, 0, len(st.Pairs))
	for k := range st.Pairs {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		p := st.Pairs[k]
		if p.Done >= p.Total || d < p.Next {
			continue
		}
		if len(ciGates(g, p.Pl)) == 0 && !ciStorm(g) {
			p.Next = d + 1 // no gate to this world: only a rift storm lets them through
			continue
		}
		ciArrive(g, st, k, p)
		p.Done++
		p.Next = d + 6 + int(HashStr(fmt.Sprintf("%s:ci:gap:%s:%d", g.Seed, k, p.Done))%15)
	}
	for _, arr := range st.Arr {
		if arr.State != "" || arr.Hidden && arr.Role != "thief" {
			continue
		}
		age := d - arr.Day
		zn, a := ciFind(g, arr)
		if a == nil {
			continue
		}
		gone := func(state, news string) {
			if arr.Esc || zn == g.Cur && g.visible(a.X, a.Y) {
				return
			}
			arr.State = state
			ciRemove(g, a)
			ciNewsAdd(g, st, arr.P, news)
			ciCloseQuest(g, arr.Quest, false)
		}
		switch arr.Role {
		case "invader":
			if age >= 4 && !arr.Raided {
				arr.Raided = true
				st.Unrest++
				ciNewsAdd(g, st, arr.P, fmt.Sprintf("Raiders from %s, led by %s, are plundering near %s.", arr.Home, arr.Name, zn.Name))
			}
		case "merchant":
			if age >= 10 {
				gone("left", fmt.Sprintf("%s, the peddler of strange goods, packed up and went looking for the way home.", arr.Name))
			}
		case "refugee":
			if age >= 15 {
				gone("taken", fmt.Sprintf("%s the refugee was found by pursuers and led away.", arr.Name))
			}
		case "explorer":
			if age >= 18 {
				gone("lost", fmt.Sprintf("%s, the lost explorer, wandered on and was not seen again.", arr.Name))
			}
		case "hunter":
			if age >= 12 {
				gone("left", fmt.Sprintf("%s the hunter has moved on, still searching.", arr.Name))
			}
		case "prophet":
			if !arr.Joined {
				st.Unrest++
			}
			if age == 5 || age == 10 || age == 15 {
				ciNewsAdd(g, st, arr.P, fmt.Sprintf("A crowd gathers around %s, the prophet from %s, who foretells the end of the sky.", arr.Name, arr.Home))
			}
			if age >= 22 {
				gone("left", fmt.Sprintf("%s the prophet walked off into the hills with a small following.", arr.Name))
			}
		case "thief":
			if age >= 20 {
				gone("left", "The thief slipped away with the stolen thing.")
			}
		}
	}
}

// ---- quests ----

func ciQuest(g *Game, arr *ciArr, title, text string, tp int) *Quest {
	q := &Quest{ID: g.newID(), Kind: "ci", Title: title, Text: text, Giver: arr.Name, GiverID: -1, Faction: "strangers", TP: tp}
	g.Quests = append(g.Quests, q)
	g.Known[tp] = true
	g.Msg("{{W|New quest: " + title + "}}")
	return q
}

func ciCloseQuest(g *Game, id int, ok bool) {
	if id == 0 {
		return
	}
	for _, q := range g.Quests {
		if q.ID == id && !q.Closed {
			q.Closed, q.Done = true, ok
		}
	}
}

func ciRealm(g *Game) string {
	if g.Cur != nil {
		return g.Cur.Faction
	}
	return ""
}

func ciHasGates(g *Game) bool { return len(unUni(g.W).gateZ[g.W.uniIdx]) > 0 }

func ciAtDest(g *Game, a *Actor) bool {
	if !ciHasGates(g) {
		return g.Cur.Kind == "town"
	}
	if _, ok := unUni(g.W).gateZ[g.W.uniIdx][g.Cur.Key]; !ok {
		return false
	}
	gx, gy, ok := g.unGate(g.Cur)
	return ok && abs(a.X-gx) <= 2 && abs(a.Y-gy) <= 2
}

func ciTurn(g *Game) {
	if g == nil || g.W == nil || g.Cur == nil || g.P == nil || g.Dead {
		return
	}
	st := g.ci()
	if d := g.Day(); st.Last != d+1 {
		st.Last = d + 1
		ciDaily(g, st, d)
	}
	var esc *Actor
	for _, a := range g.Cur.Actors {
		arr := ciArrOf(g, a)
		if arr == nil || a.HP <= 0 {
			continue
		}
		if !arr.Seen && g.visible(a.X, a.Y) {
			ciFirstSight(g, st, a, arr)
		}
		if a.Has("ciEsc") && esc == nil {
			esc = a
		}
	}
	if esc == nil {
		return
	}
	earr := ciArrOf(g, esc)
	for _, h := range g.Cur.Actors {
		if h.Has("ciHunter") && !h.Angry && h.HP > 0 && abs(h.X-esc.X) <= 8 && abs(h.Y-esc.Y) <= 8 {
			if ha := ciArrOf(g, h); ha != nil && ha.G == earr.G {
				h.Angry, h.AI, ha.Hostile = true, "hostile", true
				g.Msg("{{R|" + h.Name + " spots their quarry and comes for you both!}}")
			}
		}
	}
	if ciAtDest(g, esc) {
		ciDeliver(g, st, esc, earr)
	}
}

func ciFirstSight(g *Game, st *ciState, a *Actor, arr *ciArr) {
	arr.Seen = true
	cg := &ciGenres[arr.G]
	g.Msg(fmt.Sprintf("{{M|%s: %s %s.}}", a.Name, titleCase(ciArticle(cg.Folk)), Pick(NewRng(g.Seed+":ci:cust:"+a.Name), cg.Customs)))
	if arr.Role == "invader" && arr.Quest == 0 {
		q := ciQuest(g, arr, "Stop the invader "+arr.Name,
			fmt.Sprintf("%s, %s from %s, came through a tear in the sky and means to take this land. Put them down before they raid again.", arr.Name, ciArticle(cg.Folk), arr.Home), arr.P)
		arr.Quest = q.ID
	}
}

func ciDeliver(g *Game, st *ciState, a *Actor, arr *ciArr) {
	ciRemove(g, a)
	arr.State, arr.Esc = "safe", false
	if arr.Role == "explorer" {
		arr.State = "home"
	}
	for _, it := range a.Inv {
		if len(a.Inv) > 0 && it.Extra != nil && it.Extra["alien"] > 0 {
			g.give(it)
			g.Msg("{{G|" + a.Name + " presses " + it.Name + " into your hands.}}")
			break
		}
	}
	text := "{{G|" + a.Name + " reaches the "
	if ciHasGates(g) {
		text += "rift gate and steps through with a last look back.}}"
	} else {
		text += "town and finds a place to stay.}}"
	}
	g.Msg(text)
	if !arr.Taught {
		g.Msg(ciTeach(g, arr))
	}
	g.P.Gold += 40 + g.P.Level*8
	g.gainXP(60 + g.P.Level*10)
	g.crKarma(0, 8, "saw a stranger to safety")
	g.AddRep(ciRealm(g), 25, " for your kindness to a stranger")
	ciCloseQuest(g, arr.Quest, true)
	g.deed("quest", fmt.Sprintf("The wanderer %s escorted %s of %s through the rift.", g.P.Name, arr.Name, arr.Home), 0, 0)
	ciNewsAdd(g, st, arr.P, fmt.Sprintf("%s of %s was seen safely to the gate by %s.", arr.Name, arr.Home, g.P.Name))
}

func ciKill(g *Game, d, src *Actor) {
	arr := ciArrOf(g, d)
	if arr == nil {
		return
	}
	st := g.ci()
	byMe := src == g.P || src != nil && src.AI == "follower"
	if d.Has("ciMin") {
		return
	}
	arr.State = "dead"
	arr.Esc = false
	switch {
	case arr.Role == "invader":
		if byMe {
			g.AddRep(ciRealm(g), 60, " for stopping the invader")
			g.P.Gold += 60 + g.P.Level*10
			g.crKarma(2, 6, "stopped an invader")
		}
		ciCloseQuest(g, arr.Quest, byMe)
		ciNewsAdd(g, st, arr.P, fmt.Sprintf("%s of %s, who raided the countryside, is dead.", arr.Name, arr.Home))
	case arr.Role == "thief":
		ciNewsAdd(g, st, arr.P, "The thief from beyond the sky is dead; whatever they stole lies where they fell.")
	case arr.Role == "hunter":
		if byMe {
			g.AddRep(ciRealm(g), 10, "")
		}
		ciCloseQuest(g, arr.Quest, false)
		ciNewsAdd(g, st, arr.P, fmt.Sprintf("%s the hunter from %s was killed.", arr.Name, arr.Home))
	default:
		if byMe && !arr.Hostile {
			g.crKarma(0, -12, "killed a helpless stranger")
			g.AddRep(ciRealm(g), -30, " for a murdered stranger")
		}
		ciCloseQuest(g, arr.Quest, false)
		ciNewsAdd(g, st, arr.P, fmt.Sprintf("%s of %s was found dead.", arr.Name, arr.Home))
	}
}

func ciSide(g *Game) []string {
	if g.W == nil {
		return nil
	}
	for _, arr := range g.ci().Arr {
		if arr.Esc && arr.State == "" {
			return []string{"{{M|Escorting " + arr.Name + "}}"}
		}
	}
	return nil
}

// ---- teaching ----

func ciTeach(g *Game, arr *ciArr) string {
	cg := &ciGenres[arr.G]
	arr.Taught = true
	st := g.ci()
	switch arr.ID % 3 {
	case 0:
		if g.Skills == nil {
			g.Skills = map[string]int{}
		}
		g.Skills[cg.Skill] = mini(g.Skills[cg.Skill]+1, 5)
		return fmt.Sprintf("{{G|%s shows you %s. %s +1.}}", arr.Name, cg.Tech, skillName(cg.Skill))
	case 1:
		r := NewRng(fmt.Sprintf("%s:ci:recipe:%d", g.Seed, arr.ID))
		var food *Item
		ciWith(g, arr.G, func() { food = ciTag(g.MakeFood(r, 3), arr.G) })
		food.Name = cg.Recipe
		g.give(ciTag(g.MakeBook("Recipe: "+cg.Recipe, cg.Recipe+": "+Pick(r, cg.Creed)), arr.G))
		g.give(food)
		return fmt.Sprintf("{{G|%s writes down their recipe for %s and cooks a portion for the road.}}", arr.Name, strings.ToLower(cg.Recipe))
	}
	w := cg.Words[arr.Word%len(cg.Words)]
	line := fmt.Sprintf("%s: %s (%s)", w[0], w[1], arr.Home)
	st.Words = append(st.Words, line)
	g.crKarma(1, 2, "learned a stranger's word")
	g.gainXP(30)
	return fmt.Sprintf("{{G|%s teaches you a word: \"%s\", %s.}}", arr.Name, w[0], w[1])
}

func ciHasFood(g *Game) *Item {
	for _, it := range g.P.Inv {
		if (it.Kind == "food" || it.Kind == "water" || it.Kind == "dish") && !ciAlien(it) {
			return it
		}
	}
	return nil
}

// ---- talk ----

func ciTalk(g *Game, a *Actor) []Option {
	arr := ciArrOf(g, a)
	if arr == nil || arr.State != "" || a.Angry && a.AI == "hostile" {
		return nil
	}
	st := g.ci()
	cg := &ciGenres[arr.G]
	say := func(s string) { g.talkMenu(ciSub(s, arr, g)) }
	var out []Option
	out = append(out, Option{"Ask who they are and why they came.", func() {
		intro := ciRoleIntro[arr.Role]
		line := ciSub(Pick(NewRng(g.Seed+":ci:intro:"+a.Name), intro), arr, g)
		say(line + "\n\n" + Pick(NewRng(g.Seed+":ci:greet:"+a.Name), cg.Greets) + "\n\n{{K|They " + Pick(NewRng(g.Seed+":ci:c2:"+a.Name), cg.Customs) + ".}}")
	}})
	if !arr.Explain {
		out = append(out, Option{"Explain how things are done here.", func() {
			arr.Explain = true
			g.crKarma(1, 2, "taught a stranger local ways")
			g.AddRep(ciRealm(g), 10, " for minding a stranger")
			say("They listen carefully, nodding too much. \"So the {w} does not apply. Thank you. I will try to remember.\"")
		}})
	}
	if !arr.Taught && arr.Role != "hunter" {
		out = append(out, Option{"Ask what they could teach me.", func() {
			if arr.Favour == "" {
				switch (arr.ID + arr.Word) % 3 {
				case 0:
					arr.Favour = "food"
				case 1:
					arr.Favour = "gold"
				default:
					arr.Favour = "escort"
				}
				if arr.Favour == "escort" && arr.Role != "refugee" && arr.Role != "explorer" {
					arr.Favour = "gold"
				}
			}
			ask := map[string]string{"food": "a meal and clean water, the way it is done here", "gold": fmt.Sprintf("%d %s, a fair price", ciPrice(g), g.Sk.Currency), "escort": "a safe walk to the rift gate"}[arr.Favour]
			say("\"I know " + cg.Tech + ", and some small things besides. For a favour I would teach it: " + ask + ".\"")
		}})
		switch arr.Favour {
		case "food":
			if f := ciHasFood(g); f != nil {
				out = append(out, Option{"Offer food for the lesson.", func() {
					g.removeInv(f)
					g.crKarma(3, 3, "fed a stranger")
					g.talkMenu(ciTeach(g, arr))
				}})
			}
		case "gold":
			price := ciPrice(g)
			if g.P.Gold >= price {
				out = append(out, Option{fmt.Sprintf("Pay %d %s for the lesson.", price, g.Sk.Currency), func() {
					g.P.Gold -= price
					g.talkMenu(ciTeach(g, arr))
				}})
			}
		}
	}
	switch arr.Role {
	case "refugee", "explorer":
		if !arr.Esc {
			label := "Offer to escort them to the rift gate."
			if !ciHasGates(g) {
				label = "Offer to take them to the nearest town to settle."
			}
			out = append(out, Option{label, func() { ciAcceptEscort(g, a, arr) }})
		} else {
			out = append(out, Option{"Tell them to wait here.", func() {
				arr.Esc = false
				a.AI = "still"
				delete(a.Flags, "ciEsc")
				ciCloseQuest(g, arr.Quest, false)
				arr.Quest = 0
				say("\"I will wait, but not for long.\"")
			}})
		}
		if arr.Role == "explorer" {
			out = append(out, Option{"Ask whether they can be sent home through the rift.", func() {
				if ciHasGates(g) {
					say("\"There is a gate near, you say? Take me there and I will not forget it. {home} is waiting.\"")
				} else {
					say("\"No gate on this world, then. A storm will have to open the way. I am in no hurry to be hurled about again.\"")
				}
			}})
		}
	case "merchant":
		out = append(out, Option{"Ask why their prices are so strange.", func() {
			say("\"Back in {home} this is common as bread and I would ask a copper. Here it is a wonder and I ask the moon. I have given up fixing one price; I read faces.\"")
		}})
		if f := ciHasFood(g); f != nil && len(a.Inv) > 0 {
			out = append(out, Option{"Barter a local good for something strange.", func() {
				i := int(HashStr(fmt.Sprintf("%s:ci:barter:%d:%d", g.Seed, arr.ID, len(a.Inv)))) % len(a.Inv)
				if i < 0 {
					i = -i
				}
				it := a.Inv[i]
				a.Inv = append(a.Inv[:i], a.Inv[i+1:]...)
				g.removeInv(f)
				g.give(it)
				say("They hold your " + f.Name + " to the light, sniff it, and laugh with delight. \"Remarkable! A fair swap: take this " + it.Name + ".\"")
			}})
		}
	case "hunter":
		out = append(out, ciHunterOpts(g, st, a, arr)...)
	case "prophet":
		out = append(out, ciProphetOpts(g, st, a, arr)...)
	}
	return out
}

func ciPrice(g *Game) int { return 40 + g.P.Level*10 }

func ciAcceptEscort(g *Game, a *Actor, arr *ciArr) {
	a.AI = "follower"
	a.Set("ciEsc")
	arr.Esc = true
	tp := arr.P
	if g.Cur != nil {
		tp = g.W.Idx(g.Cur.WX, g.Cur.WY)
	}
	title := "Escort " + arr.Name + " to the rift gate"
	text := fmt.Sprintf("%s of %s needs to reach a rift gate alive. Walk them there; they follow close.", arr.Name, arr.Home)
	if !ciHasGates(g) {
		title = "Take " + arr.Name + " to a town"
		text = fmt.Sprintf("There is no gate on this world. %s of %s wants a roof in the nearest town.", arr.Name, arr.Home)
	} else {
		best := -1
		u := unUni(g.W)
		for burg := range u.ends[g.W.uniIdx] {
			bd := g.W.burgByID[burg]
			if bd == nil {
				continue
			}
			dd := abs(bd.P%g.W.W-g.WX) + abs(bd.P/g.W.W-g.WY)
			if best < 0 || dd < best {
				best, tp = dd, bd.P
			}
		}
	}
	if arr.Quest == 0 {
		arr.Quest = ciQuest(g, arr, title, text, tp).ID
	}
	g.Mode, g.Talk = "main", nil
	g.Msg("{{G|" + arr.Name + " falls in behind you.}}")
}

func ciOtherQuarry(st *ciState, arr *ciArr) *ciArr {
	for _, o := range st.Arr {
		if o.State == "" && o.G == arr.G && (o.Role == "refugee" || o.Role == "explorer") && !o.Hidden {
			return o
		}
	}
	return nil
}

func ciHunterOpts(g *Game, st *ciState, a *Actor, arr *ciArr) []Option {
	var out []Option
	say := func(s string) { g.talkMenu(ciSub(s, arr, g)) }
	leave := func(state string) {
		arr.State = state
		ciRemove(g, a)
		g.Mode, g.Talk = "main", nil
	}
	if q := ciOtherQuarry(st, arr); q != nil && !arr.Resolved {
		out = append(out, Option{"Tell them where their quarry is.", func() {
			arr.Resolved = true
			q.State = "taken"
			if _, qa := ciFind(g, q); qa != nil {
				ciRemove(g, qa)
			}
			ciCloseQuest(g, q.Quest, false)
			g.crKarma(0, -12, "sold a refugee to a hunter")
			g.crKarma(3, -4, "took a hunter's coin")
			g.P.Gold += 90 + g.P.Level*12
			if it := a.Inv; len(it) > 0 {
				g.give(it[0])
			}
			ciNewsAdd(g, st, q.P, fmt.Sprintf("%s of %s was dragged back through the sky by a hunter.", q.Name, q.Home))
			g.Msg("{{R|The hunter nods, pays, and goes.}}")
			leave("left")
		}})
		out = append(out, Option{"Send them the wrong way.", func() {
			arr.Resolved = true
			g.crKarma(1, -4, "lied to a hunter")
			g.crKarma(0, 6, "shielded a stranger")
			if g.R.Chance(0.4) {
				a.Angry, a.AI, arr.Hostile = true, "hostile", true
				g.Mode, g.Talk = "main", nil
				g.Msg("{{R|" + a.Name + " squints, laughs once, and draws their weapon: \"You lie badly.\"}}")
				return
			}
			g.Msg("{{G|The hunter believes you and strides off the wrong way.}}")
			leave("left")
		}})
		out = append(out, Option{"Tell them you will not help, and they should leave.", func() {
			arr.Resolved = true
			a.Angry, a.AI, arr.Hostile = true, "hostile", true
			g.crKarma(2, 4, "stood against a hunter")
			g.Mode, g.Talk = "main", nil
			g.Msg("{{R|" + a.Name + " sets their jaw and reaches for a weapon.}}")
		}})
		return out
	}
	if arr.Quest == 0 {
		out = append(out, Option{"Hear their request about a theft.", func() {
			r := NewRng(fmt.Sprintf("%s:ci:thief:%d", g.Seed, arr.ID))
			st.NextID++
			t := &ciArr{ID: st.NextID, Pair: arr.Pair, G: arr.G, Home: arr.Home, Role: "thief", Name: ciName(r, arr.G), Day: g.Day(), Link: arr.ID, Hidden: true}
			zn, _ := ciSite(g, r, nil)
			if zn == nil {
				say("\"Nothing to be done; the thief is beyond reach.\"")
				return
			}
			ta := ciMake(g, r, t, maxi(1, g.zoneLevel(zn.WX, zn.WY, 0)+2), false)
			if !ciPlace(g, r, zn, ta, false) {
				say("\"Nothing to be done; the thief is beyond reach.\"")
				return
			}
			t.Key, t.P = zn.Key, g.W.Idx(zn.WX, zn.WY)
			cg := &ciGenres[arr.G]
			q := ciQuest(g, arr, "Recover "+cg.Thing, fmt.Sprintf("%s says a thief named %s fled with %s and hides somewhere near %s. Bring it back.", arr.Name, t.Name, cg.Thing, zn.Name), t.P)
			arr.Quest = q.ID
			t.Quest = q.ID
			it := ciOddity(g, arr.G, r, 3)
			it.Name = cg.Thing
			it.Quest = q.ID
			it.Extra["ciq"] = q.ID
			it.Value += 200
			ta.Inv = append(ta.Inv, it)
			st.Arr = append(st.Arr, t)
			say("\"A thief of my own people fled through the sky with " + cg.Thing + ". Find them near " + zn.Name + ". It is not theirs to keep, and I will pay.\"")
		}})
		return out
	}
	for _, it := range g.P.Inv {
		if it.Extra != nil && it.Extra["ciq"] == arr.Quest {
			it := it
			out = append(out, Option{"Hand over " + it.Name + ".", func() {
				g.removeInv(it)
				ciCloseQuest(g, arr.Quest, true)
				g.P.Gold += 120 + g.P.Level*15
				g.gainXP(120)
				g.crKarma(1, 4, "returned a stolen thing")
				g.AddRep(ciRealm(g), 30, " for the stolen thing returned")
				if len(a.Inv) > 0 {
					g.give(a.Inv[0])
				}
				g.deed("quest", fmt.Sprintf("The wanderer %s recovered %s for the hunter %s.", g.P.Name, it.Name, arr.Name), 0, 0)
				g.Msg("{{G|" + arr.Name + " takes it with shaking hands and pays you from a pouch of strange coin.}}")
				leave("left")
			}})
			break
		}
	}
	return out
}

func ciProphetOpts(g *Game, st *ciState, a *Actor, arr *ciArr) []Option {
	cg := &ciGenres[arr.G]
	say := func(s string) { g.talkMenu(ciSub(s, arr, g)) }
	out := []Option{{"Listen to the prophecy.", func() {
		line := Pick(NewRng(fmt.Sprintf("%s:ci:creed:%d:%d", g.Seed, arr.ID, g.Day())), cg.Creed)
		if !arr.Taught {
			g.Msg(ciTeach(g, arr))
		}
		say("\"" + line + "\" They look at you as if checking whether the words took.\n\nThe word they keep repeating is \"{w}\": {m}.")
	}}}
	if !arr.Joined {
		out = append(out, Option{"Join the vigil.", func() {
			arr.Joined = true
			g.crKarma(4, -8, "knelt to a foreign prophet")
			g.AddRep(ciRealm(g), -20, " for joining a foreign cult")
			g.give(ciOddity(g, arr.G, NewRng(fmt.Sprintf("%s:ci:vigil:%d", g.Seed, arr.ID)), 2))
			say("You keep the vigil till the stars turn. Afterwards they press a small thing into your hand. \"Carry it, and do not lose the {w}.\"")
		}})
	}
	out = append(out, Option{"Denounce them as a false prophet.", func() {
		arr.Hostile = true
		a.Angry, a.AI = true, "hostile"
		g.crKarma(4, 5, "denounced a foreign prophet")
		g.AddRep(ciRealm(g), 20, " for denouncing a foreign cult")
		ciNewsAdd(g, st, arr.P, fmt.Sprintf("%s was denounced in the open as a false prophet.", arr.Name))
		g.Mode, g.Talk = "main", nil
		g.Msg("{{R|" + a.Name + " goes white, then raises a hand and shouts a curse in a tongue you do not know.}}")
	}})
	return out
}
