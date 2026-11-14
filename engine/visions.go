package engine

import (
	"fmt"
	"sort"
	"strings"
)

// Dreams, prophecy and omens (prefix vs). Deepens world_story.go (wy) without editing it.
//
//  1. Prophecy page (mode "vs", ability vs_prophecy): the three start verses of wy.Prophecy are read
//     (never written) and given a status from play state: unfulfilled, foreshadowed, fulfilled, defied;
//     hints given by seers and dreams are listed with them.
//  2. Seers (Role "seer", placed by a zone hook in a few towns, side rng seed:vs:seer:zone): a paid
//     reading foretells something real and upcoming (a deadline event, a rival's route, a dungeon
//     guardian near you, a coming war, a sky omen) in cryptic verse. The seer's skill sets how true the
//     reading is (true / half-true / false), the verse's ambiguity sets the Lore check to interpret it.
//     Readings are judged later against the world (came to pass / defied / lapsed).
//  3. Dreams: after a long sleep, nightmares and visions drawn from kills, injuries, lost companions,
//     the nemesis and mercy; a recurring motif that evolves through five stages and at the later stages
//     gives insight (lucid, XP, a hint) while nightmares may leave "Disturbed rest" (vs_dread).
//  4. Omens: comet, blood moon, two-headed calf, ravens, rift sign. A pure schedule of the day clock
//     (hash of seed + 12-day window, never g.R): sidebar line once seen or heard, news in nearby burgs,
//     small effects (vsPriceMul, vsSpawnMul, vsCultMul, extra night wanderers on a blood moon) and talk
//     option "Ask what folk make of ..." for townsfolk and clergy.
//  5. The Visions journal tab lists dreams, omens and readings (vsLines for Journal+).
//
// State g.ext("vs") (global).

const vsWin = 12

type vsOmen struct {
	ID     string `json:"id"`
	Start  int    `json:"s"`
	End    int    `json:"e"`
	Burg   int    `json:"b,omitempty"`
	Planet int    `json:"pl,omitempty"`
}

type vsRead struct {
	ID     int      `json:"id"`
	Day    int      `json:"d"`
	Seer   string   `json:"sr"`
	Town   string   `json:"tn"`
	Kind   string   `json:"k"` // dl rival boss war omen prophecy
	Ref    int      `json:"r"`
	P      int      `json:"p"`
	Verse  []string `json:"v"`
	Plain  string   `json:"pl"`
	Truth  string   `json:"t"` // true partial false
	Real   bool     `json:"re"`
	Interp bool     `json:"i,omitempty"`
	Tried  bool     `json:"tr,omitempty"`
	DC     int      `json:"dc"`
	Sense  string   `json:"se,omitempty"`
	Status string   `json:"s"` // pending came defied lapsed
	DueDay int      `json:"dd"`
	Planet int      `json:"pn"`
}

type vsJ struct {
	Day  int    `json:"d"`
	Kind string `json:"k"`
	Text string `json:"t"`
}

type vsState struct {
	Reads   []*vsRead       `json:"rd,omitempty"`
	Journal []vsJ           `json:"j,omitempty"`
	Hint    [3]bool         `json:"h"`
	HintTxt [3]string       `json:"ht"`
	Last    [3]string       `json:"ls"`
	Motif   string          `json:"m,omitempty"`
	Stage   int             `json:"sg"`
	Dreams  int             `json:"dn"`
	Slept   int             `json:"sl,omitempty"`
	Lost    []string        `json:"lost,omitempty"`
	Seen    map[string]bool `json:"seen,omitempty"`
	Heard   map[string]bool `json:"heard,omitempty"`
	Told    map[string]bool `json:"told,omitempty"`
	Spawned map[string]bool `json:"spw,omitempty"`
	Fore    []vsOmen        `json:"fore,omitempty"`
	Seq     int             `json:"sq,omitempty"`
	Tab     int             `json:"-"`
	Scroll  int             `json:"-"`
}

func init() {
	turnHooks = append(turnHooks, vsTurn)
	zoneHooks = append(zoneHooks, vsZone)
	enterHooks = append(enterHooks, vsEnter)
	killHooks = append(killHooks, vsKill)
	talkHooks = append(talkHooks, vsTalk)
	sideHooks = append(sideHooks, vsSide)
	modeKeys["vs"] = vsKeys
	modeDraws["vs"] = func(g *Game) { g.vsDraw() }
	skAddStatus(&skStatusDef{ID: "vs_dread", Name: "Disturbed rest", Glyph: 'd', Hit: -1, WIL: -1, Speed: -5})
	skAddStatus(&skStatusDef{ID: "vs_lucid", Name: "Lucid", Glyph: 'l', Hit: 1, WIL: 1})
}

func (g *Game) vs() *vsState {
	s := g.ext("vs", func() any { return &vsState{} }).(*vsState)
	if s.Seen == nil {
		s.Seen = map[string]bool{}
	}
	if s.Heard == nil {
		s.Heard = map[string]bool{}
	}
	if s.Told == nil {
		s.Told = map[string]bool{}
	}
	if s.Spawned == nil {
		s.Spawned = map[string]bool{}
	}
	return s
}

func vsAbs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

func (g *Game) vsGenre() string {
	if _, ok := vsSeerTitle[g.Genre]; ok {
		return g.Genre
	}
	return "fantasy"
}

func vsFill(s string, kv ...string) string {
	for i := 0; i+1 < len(kv); i += 2 {
		s = strings.ReplaceAll(s, "{"+kv[i]+"}", kv[i+1])
	}
	return s
}

func (g *Game) vsPlayerP() int { return g.W.Idx(g.WX, g.WY) }

func (g *Game) vsJournal(kind, text string) {
	st := g.vs()
	st.Journal = append(st.Journal, vsJ{g.Day() + 1, kind, text})
	if len(st.Journal) > 90 {
		st.Journal = st.Journal[len(st.Journal)-90:]
	}
}

// ---- omens ----

func (g *Game) vsSkin(id string) vsSkin {
	if m := vsSkins[g.vsGenre()]; m != nil {
		if s, ok := m[id]; ok {
			return s
		}
	}
	return vsSkins["fantasy"][id]
}

func (g *Game) vsOmenWin(b int) *vsOmen {
	if b < 0 {
		return nil
	}
	h := HashStr(fmt.Sprintf("%s:vs:om:%d", g.Seed, b))
	if h%100 >= 52 {
		return nil
	}
	id := vsOmenIDs[(h>>8)%uint64(len(vsOmenIDs))]
	s := b*vsWin + int((h>>16)%10)
	o := &vsOmen{ID: id, Start: s, End: s + vsOmenDays[id] - 1, Planet: g.W.uniIdx}
	if id == "calf" || id == "ravens" {
		if bl := g.ws().burgList(g); len(bl) > 0 {
			o.Burg = bl[int((h>>24)%uint64(len(bl)))].ID
		}
	}
	return o
}

// vsOmensOn lists the omens whose days include the given day.
func (g *Game) vsOmensOn(day int) []*vsOmen {
	if g.W == nil {
		return nil
	}
	var out []*vsOmen
	for b := day/vsWin - 1; b <= day/vsWin; b++ {
		if o := g.vsOmenWin(b); o != nil && day >= o.Start && day <= o.End {
			out = append(out, o)
		}
	}
	return out
}

// vsNextOmen is the first omen that starts after the day within about 40 days.
func (g *Game) vsNextOmen(day int) *vsOmen {
	for b := day / vsWin; b <= day/vsWin+3; b++ {
		if o := g.vsOmenWin(b); o != nil && o.Start > day {
			return o
		}
	}
	return nil
}

func (g *Game) vsKey(o *vsOmen) string { return fmt.Sprintf("%s:%s:%d", g.Seed, o.ID, o.Start) }

func (g *Game) vsLocal(o *vsOmen) bool { return o.ID == "calf" || o.ID == "ravens" }

func (g *Game) vsBurgName(id int) string {
	if b := g.W.burgByID[id]; b != nil {
		return b.Name
	}
	return "a distant town"
}

func (g *Game) vsNearBurg(o *vsOmen) bool {
	if o.Burg == 0 || g.Cur == nil {
		return false
	}
	if g.Cur.Burg == o.Burg {
		return true
	}
	if b := g.W.burgByID[o.Burg]; b != nil {
		return wsCheb(g.W, b.P, g.vsPlayerP()) <= 2
	}
	return false
}

// vsRelevant: a local omen only matters near its town; the others are sky-wide.
func (g *Game) vsRelevant(o *vsOmen) bool { return !g.vsLocal(o) || g.vsNearBurg(o) }

func (g *Game) vsVisible(o *vsOmen) bool {
	h := g.Hour()
	switch o.ID {
	case "comet", "bloodmoon":
		return h >= 19 || h < 5
	case "rift":
		return true
	}
	return g.vsNearBurg(o)
}

func (g *Game) vsActive() []*vsOmen {
	var out []*vsOmen
	for _, o := range g.vsOmensOn(g.Day()) {
		if g.vsRelevant(o) {
			out = append(out, o)
		}
	}
	return out
}

func (g *Game) vsOmenName(o *vsOmen) string { return g.vsSkin(o.ID).Name }

// vsPriceMul is the small price swing the omens cause (read by price() if wired).
func (g *Game) vsPriceMul(it *Item) float64 {
	if g == nil || g.W == nil || it == nil {
		return 1
	}
	m := 1.0
	for _, o := range g.vsActive() {
		switch o.ID {
		case "comet":
			if it.Kind == "tonic" || it.Kind == "book" || it.Kind == "manual" {
				m *= 1.1
			}
		case "bloodmoon":
			switch it.Kind {
			case "weapon", "ranged", "armor", "grenade", "ammo":
				m *= 1.1
			}
		case "calf":
			if it.Kind == "food" || it.Kind == "water" {
				m *= 1.12
			}
		case "rift":
			if it.Kind == "artifact" || it.Kind == "scrap" {
				m *= 1.1
			}
		}
	}
	return m
}

// vsSpawnMul is the wandering-monster multiplier while omens hang over the land (1 normally).
func (g *Game) vsSpawnMul() float64 {
	if g == nil || g.W == nil {
		return 1
	}
	m := 1.0
	h := g.Hour()
	for _, o := range g.vsActive() {
		switch o.ID {
		case "bloodmoon":
			if h >= 19 || h < 5 {
				m *= 1.5
			}
		case "rift":
			m *= 1.2
		case "ravens":
			m *= 1.1
		}
	}
	return m
}

// vsCultMul is the multiplier for heretic cult activity (orders.go may read it).
func (g *Game) vsCultMul() float64 {
	if g == nil || g.W == nil {
		return 1
	}
	m := 1.0
	for _, o := range g.vsActive() {
		switch o.ID {
		case "comet", "rift":
			m *= 1.25
		case "bloodmoon":
			m *= 1.5
		}
	}
	return m
}

func (g *Game) vsNearestBurgs(n int) []*BurgData {
	pp := g.vsPlayerP()
	bl := append([]*BurgData(nil), g.ws().burgList(g)...)
	sort.SliceStable(bl, func(i, j int) bool { return wsCheb(g.W, bl[i].P, pp) < wsCheb(g.W, bl[j].P, pp) })
	if len(bl) > n {
		bl = bl[:n]
	}
	return bl
}

func (g *Game) vsOmenTick(st *vsState) {
	day := g.Day()
	for _, o := range g.vsOmensOn(day) {
		key := g.vsKey(o)
		sk := g.vsSkin(o.ID)
		if !st.Told[key] {
			st.Told[key] = true
			if g.vsLocal(o) {
				lvSt(g).addNews(lvDay(g), o.Burg, "%s", sk.News)
			} else {
				for _, b := range g.vsNearestBurgs(3) {
					lvSt(g).addNews(lvDay(g), b.ID, "%s", sk.News)
				}
			}
			if o.ID == "comet" || o.ID == "bloodmoon" || o.ID == "rift" {
				if bl := g.vsNearestBurgs(1); len(bl) > 0 {
					lvSt(g).addNews(lvDay(g), bl[0].ID, "Hooded gatherings were reported after %s; the watch is asking questions.", sk.Name)
				}
			}
		}
		if !st.Seen[key] && g.Cur.Z == 0 && g.vsVisible(o) && g.vsRelevant(o) {
			st.Seen[key] = true
			g.Msg("{{M|" + sk.Sight + "}}")
			g.vsJournal("omen", capFirst(sk.Name)+": "+sk.Sight)
		}
	}
}

func capFirst(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func vsEnter(g *Game, zn *Zone) {
	if g.W == nil || g.P == nil || zn == nil || zn.Z != 0 || zn.Kind == "town" {
		return
	}
	h := g.Hour()
	if h < 19 && h >= 5 {
		return
	}
	st := g.vs()
	for _, o := range g.vsOmensOn(g.Day()) {
		if o.ID != "bloodmoon" {
			continue
		}
		k := fmt.Sprintf("%s:%d", zn.Key, g.Day())
		if st.Spawned[k] || !g.R.Chance(0.6) {
			continue
		}
		st.Spawned[k] = true
		n := g.R.Int(1, 2)
		for i := 0; i < n; i++ {
			a := g.Spawn(pickCreature(g.R, zn.Biome, 's', zn.Level), zn.Level, g.R)
			x, y, ok := zn.FreeSpot(g.R, nil)
			if !ok {
				continue
			}
			a.X, a.Y = x, y
			a.AI = "hostile"
			zn.Actors = append(zn.Actors, a)
		}
		g.Msg("{{R|Under the red moon, something stirs in the dark.}}")
	}
}

func vsSide(g *Game) []string {
	if g.W == nil || g.P == nil || g.Cur == nil {
		return nil
	}
	st := g.vs()
	day := g.Day()
	var out []string
	for _, o := range g.vsActive() {
		k := g.vsKey(o)
		if st.Seen[k] || st.Heard[k] {
			out = append(out, fmt.Sprintf("{{M|Sky: %s}} {{K|%dd}}", vsOmenShort[o.ID], o.End-day+1))
		}
	}
	return out
}

// ---- killed companions ----

func vsKill(g *Game, d, src *Actor) {
	if g.W == nil || d == nil || d.AI != "follower" || d.Comp == nil || d.Comp.Summoned {
		return
	}
	st := g.vs()
	st.Lost = append(st.Lost, d.Name)
	if len(st.Lost) > 8 {
		st.Lost = st.Lost[len(st.Lost)-8:]
	}
}

// ---- seers ----

func (g *Game) vsQuality(burg int) int {
	n := HashStr(fmt.Sprintf("%s:vs:q:%d", g.Seed, burg)) % 10
	switch {
	case n <= 1:
		return 0
	case n >= 8:
		return 2
	}
	return 1
}

func vsZone(g *Game, zn *Zone, _ *Rng) {
	if zn.Kind != "town" || zn.ZX != 1 || zn.ZY != 1 || zn.Z != 0 || zn.Burg <= 0 {
		return
	}
	bd := g.W.burgByID[zn.Burg]
	if bd == nil || !bd.Alive {
		return
	}
	if _, _, z := unTownZone(g.W, bd); z != zn.Z {
		return
	}
	chance := uint64(18)
	if bd.Capital {
		chance = 50
	}
	if HashStr(fmt.Sprintf("%s:vs:seer:%d", g.Seed, bd.ID))%100 >= chance {
		return
	}
	for _, a := range zn.Actors {
		if a.Role == "seer" {
			return
		}
	}
	r := NewRng(g.Seed + ":vs:seer:" + zn.Key)
	x, y, ok := zn.FreeSpot(r, func(x, y int) bool {
		t := zn.At(x, y)
		return (t == TCobble || t == TRoad || t == TDirt || t == TGrass || t == TFloor) && x > 2 && x < MW-3 && y > 1 && y < MH-2
	})
	if !ok {
		return
	}
	a := g.makeNPC("seer", bd, g.W.cultByID[bd.Culture], r)
	gn := g.vsGenre()
	a.Title = vsSeerTitle[gn]
	a.Desc = fmt.Sprintf("A %s of %s, who %s for those who pay.", a.Title, bd.Name, vsSeerWay[gn])
	a.AI = "still"
	a.F = CMagenta
	a.X, a.Y, a.HomeX, a.HomeY = x, y, x, y
	zn.Actors = append(zn.Actors, a)
}

func (g *Game) vsFee(fate bool) int {
	f := 30 + 8*g.P.Level
	if fate {
		f *= 2
	}
	return f
}

func vsTalk(g *Game, a *Actor) []Option {
	if g.W == nil || g.Cur == nil || g.P == nil || a == nil || a.HP <= 0 || a.Angry || a.AI == "follower" || a.AI == "hostile" {
		return nil
	}
	var out []Option
	if a.Role == "seer" {
		out = append(out, Option{"{{M|Ask the " + a.Title + " for a reading.}}", func() { g.vsSeerMenu(a) }})
		return out
	}
	if g.Cur.Kind != "town" {
		return nil
	}
	n := 0
	for _, o := range g.vsActive() {
		o := o
		if n >= 2 {
			break
		}
		n++
		out = append(out, Option{"{{m|Ask what folk make of " + g.vsOmenName(o) + ".}}", func() {
			g.vs().Heard[g.vsKey(o)] = true
			pool := vsFolk[o.ID]
			switch a.Role {
			case "priest", "monk", "nun", "shaman", "druid", "cleric", "abbot":
				pool = vsHoly[o.ID]
			}
			line := Pick(NewRng(fmt.Sprintf("%d:%s", a.ID, g.vsKey(o))), pool)
			g.talkMenu(vsFill(line, "omen", g.vsOmenName(o), "town", g.Cur.Name))
		}})
	}
	return out
}

func (g *Game) vsSeerMenu(a *Actor) {
	st := g.vs()
	q := g.vsQuality(g.Cur.Burg)
	g.Popup = Wrap(fmt.Sprintf("The %s %s. \"Coin first, then the future. Which will it be?\"", a.Title, vsSeerWay[g.vsGenre()]), 100)
	g.Opts = nil
	add := func(t string, f func()) { g.Opts = append(g.Opts, Option{t, f}) }
	fee, fee2 := g.vsFee(false), g.vsFee(true)
	add(fmt.Sprintf("{{M|A reading of what is coming.}} {{K|(%s)}}", g.cuFormat(fee)), func() { g.vsBuy(a, q, false) })
	add(fmt.Sprintf("{{M|A reading of your own fate.}} {{K|(%s)}}", g.cuFormat(fee2)), func() { g.vsBuy(a, q, true) })
	if st.Motif != "" && st.Dreams > 0 {
		add(fmt.Sprintf("{{m|Have your dreams read.}} {{K|(%s)}}", g.cuFormat(fee/2)), func() { g.vsDreamRead(a) })
	}
	add("{{m|Ask what the sky holds.}}", func() { g.vsSkyTalk(a) })
	add("Leave it.", func() { g.talkMenu(g.greeting(a)) })
}

func (g *Game) vsSkyTalk(a *Actor) {
	st := g.vs()
	day := g.Day()
	var parts []string
	for _, o := range g.vsOmensOn(day) {
		st.Heard[g.vsKey(o)] = true
		parts = append(parts, fmt.Sprintf("%s hangs over us now (%d days left).", capFirst(g.vsOmenName(o)), o.End-day+1))
	}
	if n := g.vsNextOmen(day); n != nil {
		known := false
		for _, f := range st.Fore {
			if f.ID == n.ID && f.Start == n.Start && f.Planet == n.Planet {
				known = true
			}
		}
		if !known {
			st.Fore = append(st.Fore, *n)
			if len(st.Fore) > 6 {
				st.Fore = st.Fore[1:]
			}
			g.vsJournal("omen", fmt.Sprintf("%s foretold in %d days.", capFirst(g.vsOmenName(n)), n.Start-day))
		}
		parts = append(parts, fmt.Sprintf("In %d days I see %s.", n.Start-day, g.vsOmenName(n)))
	}
	if len(parts) == 0 {
		parts = append(parts, "The sky is quiet, and quiet skies are the most dangerous.")
	}
	g.vsSayBack(a, "\""+strings.Join(parts, " ")+"\"")
}

func (g *Game) vsSayBack(a *Actor, text string) {
	g.Popup = Wrap(text, 100)
	g.Opts = []Option{{"Go back.", func() { g.vsSeerMenu(a) }}, {"Leave it.", func() { g.talkMenu(g.greeting(a)) }}}
}

func (g *Game) vsBuy(a *Actor, q int, fate bool) {
	fee := g.vsFee(fate)
	if !g.cuSpend(fee) {
		g.vsSayBack(a, "\"Come back when your purse is heavier.\"")
		return
	}
	rd := g.vsMake(q, fate, a.Title, g.Cur.Name)
	g.vsShowRead(a, rd)
}

func (g *Game) vsShowRead(a *Actor, rd *vsRead) {
	g.Popup = Wrap(strings.Join(rd.Verse, "\n"), 100)
	if rd.Interp {
		g.Popup = append(g.Popup, "")
		g.Popup = append(g.Popup, Wrap("{{G|"+rd.Plain+"}}", 100)...)
		if rd.Sense != "" {
			g.Popup = append(g.Popup, Wrap("{{K|"+rd.Sense+"}}", 100)...)
		}
	} else {
		g.Popup = append(g.Popup, "", "{{K|The meaning is hidden in the verse.}}")
	}
	g.Opts = nil
	add := func(t string, f func()) { g.Opts = append(g.Opts, Option{t, f}) }
	if !rd.Interp {
		if !rd.Tried {
			add(fmt.Sprintf("Try to work out what it means. %s", g.checkLabel("lore", rd.DC)), func() {
				rd.Tried = true
				roll := g.R.Dice(2, 6) + g.checkMod("lore")
				if roll >= rd.DC {
					g.vsInterpret(rd, roll-rd.DC >= 4)
				} else {
					g.Msg("{{K|The meaning slips away like smoke.}}")
				}
				g.vsShowRead(a, rd)
			})
		}
		fee := g.vsFee(rd.Kind == "prophecy")
		add(fmt.Sprintf("Ask the %s to explain it. {{K|(%s)}}", a.Title, g.cuFormat(fee)), func() {
			if !g.cuSpend(fee) {
				g.vsSayBack(a, "\"Explanations cost the same as riddles.\"")
				return
			}
			g.vsInterpret(rd, false)
			g.vsShowRead(a, rd)
		})
	}
	add("Back to the seer.", func() { g.vsSeerMenu(a) })
	add("Leave it.", func() { g.talkMenu(g.greeting(a)) })
}

func (g *Game) vsInterpret(rd *vsRead, sense bool) {
	rd.Interp = true
	if sense {
		switch rd.Truth {
		case "true":
			rd.Sense = "Your gut says the reading rings true."
		case "partial":
			rd.Sense = "Your gut says the reading is only half right."
		default:
			rd.Sense = "Your gut says the reading rings hollow."
		}
	}
	st := g.vs()
	if rd.Kind == "prophecy" && rd.Ref >= 0 && rd.Ref < 3 {
		st.Hint[rd.Ref] = true
		st.HintTxt[rd.Ref] = rd.Plain
	}
	if rd.P >= 0 && rd.Truth != "false" && g.W.In(rd.P%g.W.W, rd.P/g.W.W) {
		g.Known[rd.P] = true
	}
	g.vsJournal("reading", "Understood: "+rd.Plain)
}

func (g *Game) vsDreamRead(a *Actor) {
	st := g.vs()
	fee := g.vsFee(false) / 2
	if !g.cuSpend(fee) {
		g.vsSayBack(a, "\"The dream-trade is cheap, but not free.\"")
		return
	}
	m := vsMotifOf(st.Motif)
	txt := "\"A recurring dream is a letter you keep refusing to open. Yours is " + m.Name + ". "
	if g.roll2d6Check(g.checkMod("empathy"), 9) {
		g.P.AddSt("vs_lucid", 720)
		delete(g.P.Status, "vs_dread")
		txt += "Say its name before you sleep and it will lose its teeth.\" {{G|You feel steadier. (Lucid, and the dread lifts.)}}"
	} else {
		txt += "I can tell you only that it is not finished with you.\""
	}
	g.vsSayBack(a, txt)
}

// ---- foretelling ----

type vsCand struct {
	Kind  string
	Ref   int
	P     int
	Days  int
	Name  string
	Img   []string
	Fmt   string
	W     float64
	Realm string
}

func vsWhen(days int) string {
	switch {
	case days < 0:
		return "when the time is ripe"
	case days == 0:
		return "within the day"
	case days == 1:
		return "in about a day"
	}
	return fmt.Sprintf("in about %d days", days)
}

func (g *Game) vsDir(p int) (string, int) {
	w := g.W
	tx, ty := p%w.W, p/w.W
	return wyCompassDir(g.WX, g.WY, tx, ty), maxi(vsAbs(tx-g.WX), vsAbs(ty-g.WY))
}

func (g *Game) vsWhere(p int) string {
	if p < 0 {
		return "over all the land"
	}
	d, n := g.vsDir(p)
	if d == "here" {
		return "right where you stand"
	}
	return fmt.Sprintf("to the %s, about %d parasangs off, near %s", d, n, g.wsPlace(p))
}

func (g *Game) vsCands() []vsCand {
	var out []vsCand
	today := g.Day()
	pp := g.vsPlayerP()
	for _, e := range g.dlActive() {
		_, left := g.dlCount(e)
		p := e.Site
		if b := g.W.burgByID[e.Burg]; b != nil {
			p = b.P
		}
		img := vsImgDL[e.Kind]
		if len(img) == 0 {
			img = vsImgDL["relic"]
		}
		out = append(out, vsCand{Kind: "dl", Ref: e.ID, P: p, Days: maxi(0, left/DayLen), Name: g.dlTitle(e), Img: img,
			Fmt: "%[1]s. It comes to a head %[3]s; look %[2]s.", W: 3})
	}
	for _, rv := range g.rv().R {
		if rv == nil || !rv.Alive || (rv.Act != "travel" && rv.Act != "delve" && rv.Act != "hunt") || !g.W.In(rv.Dest%g.W.W, rv.Dest/g.W.W) {
			continue
		}
		out = append(out, vsCand{Kind: "rival", Ref: rv.Slot, P: rv.Dest, Days: maxi(0, rv.Arrive-g.rvToday()), Name: g.rvName(rv), Img: vsImgRival,
			Fmt: "%[1]s, a rival wanderer, is bound %[2]s and arrives %[3]s.", W: 2})
	}
	sites := g.rvSites(pp, 6)
	sort.SliceStable(sites, func(i, j int) bool { return wsCheb(g.W, sites[i].P, pp) < wsCheb(g.W, sites[j].P, pp) })
	for i, s := range sites {
		if i >= 2 {
			break
		}
		out = append(out, vsCand{Kind: "boss", Ref: s.P, P: s.P, Days: -1, Name: s.S.Name, Img: vsImgBoss,
			Fmt: "A guardian waits in %[1]s, %[2]s. Face it %[3]s.", W: 1})
	}
	var pairs [][2]int
	for i := range g.W.States {
		s := &g.W.States[i]
		if !s.Alive {
			continue
		}
		for _, w := range g.warsOf(s.ID) {
			if s.ID < w {
				pairs = append(pairs, [2]int{s.ID, w})
			}
		}
	}
	if len(pairs) > 0 {
		pr := pairs[int(HashStr(fmt.Sprintf("%s:vs:war:%d", g.Seed, today))%uint64(len(pairs)))]
		if b := g.warBurg(pr[0]); b != nil {
			out = append(out, vsCand{Kind: "war", Ref: pr[0]*1000 + pr[1], P: b.P, Days: -1,
				Name: g.realmName(pr[0]) + " and " + g.realmName(pr[1]), Img: vsImgWar,
				Fmt: "%[1]s are at war, and it will not be settled %[3]s; the fighting is hottest %[2]s.", W: 1})
		}
	}
	if o := g.vsNextOmen(today); o != nil {
		p := -1
		if g.vsLocal(o) {
			if b := g.W.burgByID[o.Burg]; b != nil {
				p = b.P
			}
		}
		out = append(out, vsCand{Kind: "omen", Ref: o.Start, P: p, Days: o.Start - today, Name: g.vsOmenName(o), Img: vsImgOmen,
			Fmt: "The sky will show %[1]s %[3]s, seen %[2]s.", W: 1.5, Realm: o.ID})
	}
	return out
}

func vsTruth(q int, r *Rng) string {
	f := r.Float()
	var tr, pa float64
	switch q {
	case 2:
		tr, pa = 0.9, 0.98
	case 0:
		tr, pa = 0.4, 0.8
	default:
		tr, pa = 0.7, 0.95
	}
	switch {
	case f < tr:
		return "true"
	case f < pa:
		return "partial"
	}
	return "false"
}

// vsMake builds a reading. q: seer quality 0..2. fate: a reading of the start prophecy.
func (g *Game) vsMake(q int, fate bool, seer, town string) *vsRead {
	st := g.vs()
	st.Seq++
	r := NewRng(fmt.Sprintf("%s:vs:read:%d:%d", g.Seed, g.Day(), st.Seq))
	gn := g.vsGenre()
	rd := &vsRead{ID: st.Seq, Day: g.Day() + 1, Seer: seer, Town: town, Status: "pending", Planet: g.W.uniIdx, P: -1}
	rd.Verse = append(rd.Verse, Pick(r, vsOpeners[gn]))
	amb := r.Int(0, 2)
	rd.DC = 7 + 2*amb
	rd.Truth = vsTruth(q, r)
	rd.Real = rd.Truth != "false"
	if fate {
		g.vsFateRead(rd, r, amb)
		return g.vsFinish(rd)
	}
	cands := g.vsCands()
	if len(cands) == 0 {
		g.vsFateRead(rd, r, amb)
		return g.vsFinish(rd)
	}
	tot := 0.0
	for _, c := range cands {
		tot += c.W
	}
	f := r.Float() * tot
	c := cands[0]
	for _, x := range cands {
		if f < x.W {
			c = x
			break
		}
		f -= x.W
	}
	rd.Kind, rd.Ref, rd.P = c.Kind, c.Ref, c.P
	days, p, name := c.Days, c.P, c.Name
	switch rd.Truth {
	case "partial":
		if days > 0 {
			days = days*3/2 + 1
		}
		if p >= 0 {
			x, y := p%g.W.W+r.Int(-3, 3), p/g.W.W+r.Int(-3, 3)
			if g.W.In(x, y) {
				p = g.W.Idx(x, y)
			}
		}
	case "false":
		if days >= 0 {
			days = days*2 + 3
		}
		if bl := g.ws().burgList(g); len(bl) > 0 && p >= 0 {
			fb := bl[r.Int(0, len(bl)-1)]
			if b := g.W.burgByID[g.vsBurgAt(c.P)]; b != nil {
				name = strings.ReplaceAll(name, b.Name, fb.Name)
			}
			p = fb.P
		}
	}
	rd.P = p
	rd.DueDay = g.Day() + maxi(days, 4)
	if c.Kind == "omen" {
		rd.DueDay = g.Day() + days
	}
	rd.Plain = fmt.Sprintf(c.Fmt, name, g.vsWhere(p), vsWhen(days))
	img := Pick(r, c.Img)
	rd.Verse = append(rd.Verse, img)
	t := vsTimeMeta[4]
	switch {
	case days < 0:
	case days <= 1:
		t = vsTimeMeta[0]
	case days <= 3:
		t = vsTimeMeta[1]
	case days <= 7:
		t = vsTimeMeta[2]
	case days <= 14:
		t = vsTimeMeta[3]
	}
	rd.Verse = append(rd.Verse, vsDirLine(g, p, amb, t))
	return g.vsFinish(rd)
}

func (g *Game) vsBurgAt(p int) int {
	if p >= 0 && p < len(g.W.Burg) {
		return g.W.Burg[p]
	}
	return 0
}

func vsDirLine(g *Game, p, amb int, t string) string {
	if p < 0 {
		return capFirst(t) + ", under every open sky."
	}
	d, _ := g.vsDir(p)
	tbl := vsDirPrecise
	if amb > 0 {
		tbl = vsDirAmbig
	}
	m := tbl[d]
	if m == "" {
		m = "right beneath your own feet"
	}
	return capFirst(t) + ", " + m + "."
}

func (g *Game) vsFinish(rd *vsRead) *vsRead {
	st := g.vs()
	st.Reads = append(st.Reads, rd)
	if len(st.Reads) > 30 {
		st.Reads = st.Reads[len(st.Reads)-30:]
	}
	g.vsJournal("reading", fmt.Sprintf("A %s of %s gave a reading: %s", rd.Seer, rd.Town, strings.Join(rd.Verse[1:], " ")))
	return rd
}

// vsFateRead foretells one of the three start verses and counts as a seer's hint to it.
func (g *Game) vsFateRead(rd *vsRead, r *Rng, amb int) {
	st := g.vs()
	i := r.Int(0, 2)
	for k := 0; k < 3; k++ {
		if !st.Hint[(i+k)%3] {
			i = (i + k) % 3
			break
		}
	}
	rd.Kind, rd.Ref = "prophecy", i
	rd.Truth, rd.Real = "true", true
	rd.DC = 8 + 2*amb
	rd.Verse = append(rd.Verse, Pick(r, vsVerseHint[i]))
	rd.Verse = append(rd.Verse, capFirst(vsTimeMeta[4])+", when you have forgotten that you were told.")
	rd.Plain = g.vsPropPlain(i)
	rd.DueDay = g.Day() + 60
	if !st.Hint[i] {
		st.Hint[i] = true
		st.HintTxt[i] = "A seer spoke of it in riddles: " + rd.Verse[1]
	}
}

// vsPropPlain is the plain meaning of verse i for the player's present place.
func (g *Game) vsPropPlain(i int) string {
	pp := g.vsPlayerP()
	switch i {
	case 0:
		sites := g.rvSites(pp, 10)
		sort.SliceStable(sites, func(a, b int) bool { return wsCheb(g.W, sites[a].P, pp) < wsCheb(g.W, sites[b].P, pp) })
		if len(sites) > 0 {
			return fmt.Sprintf("The Beast of the verse is the guardian of %s, %s.", sites[0].S.Name, g.vsWhere(sites[0].P))
		}
		return "The Beast is not yet found; walk the wilds and it will find you."
	case 1:
		best, bn := 99, (*BurgData)(nil)
		for _, b := range g.ws().burgList(g) {
			if unRiftBurgAt(g.W, b.P) {
				if d := wsCheb(g.W, b.P, pp); d < best {
					best, bn = d, b
				}
			}
		}
		if bn != nil {
			return fmt.Sprintf("The gate of the verse stands at %s, %s.", bn.Name, g.vsWhere(bn.P))
		}
		return "No gate is near; look for it where the sky is thin."
	}
	best, bn := 99, (*BurgData)(nil)
	for _, b := range g.ws().burgList(g) {
		if b.Capital {
			if d := wsCheb(g.W, b.P, pp); d < best {
				best, bn = d, b
			}
		}
	}
	if bn != nil {
		return fmt.Sprintf("The thrones will first weigh you at %s, %s.", bn.Name, g.vsWhere(bn.P))
	}
	return "The thrones will weigh you wherever you are known."
}

func (g *Game) vsJudge(rd *vsRead) string {
	if rd.Status != "pending" || rd.Planet != g.W.uniIdx {
		return rd.Status
	}
	today := g.Day()
	switch rd.Kind {
	case "dl":
		var e *dlEv
		for _, x := range g.dl().Evs {
			if x.ID == rd.Ref {
				e = x
			}
		}
		switch {
		case !rd.Real:
			if today > rd.DueDay+2 {
				return "lapsed"
			}
		case e == nil:
			return "lapsed"
		case e.State == "won" || e.State == "shared":
			return "defied"
		case !e.live():
			return "came"
		}
	case "omen":
		if !rd.Real {
			if today > rd.DueDay {
				return "lapsed"
			}
		} else if today >= rd.DueDay {
			return "came"
		}
	case "war":
		a, b := rd.Ref/1000, rd.Ref%1000
		if today >= rd.DueDay {
			if !rd.Real {
				return "lapsed"
			}
			for _, w := range g.warsOf(a) {
				if w == b {
					return "came"
				}
			}
			return "defied"
		}
	case "rival", "boss":
		if today >= rd.DueDay+3 {
			if rd.Real {
				return "came"
			}
			return "lapsed"
		}
	}
	return "pending"
}

func (g *Game) vsJudgeAll(st *vsState) {
	for _, rd := range st.Reads {
		s := g.vsJudge(rd)
		if s != rd.Status {
			rd.Status = s
			word := map[string]string{"came": "came to pass", "defied": "was defied", "lapsed": "came to nothing"}[s]
			g.vsJournal("reading", fmt.Sprintf("The reading of day %d %s: %s", rd.Day, word, rd.Plain))
			g.Msg("{{m|A foretelling " + word + ".}}")
		}
	}
}

// ---- prophecy ----

func (g *Game) vsRepBest() (int, int) {
	best, worst := 0, 0
	for k, v := range g.Rep {
		if strings.HasPrefix(k, "state:") {
			best, worst = maxi(best, v), mini(worst, v)
		}
	}
	return best, worst
}

func (g *Game) vsNearestRift() *BurgData {
	for _, b := range g.ws().burgList(g) {
		if unRiftBurgAt(g.W, b.P) {
			return b
		}
	}
	return nil
}

// vsVerse gives the status and the reason of start verse i.
func (g *Game) vsVerse(i int) (string, string) {
	pr := g.wy().Prophecy
	if pr == nil {
		return "unfulfilled", "The prophecy has not yet been spoken."
	}
	st := g.vs()
	best, worst := g.vsRepBest()
	switch i {
	case 0:
		switch {
		case pr.OmenBeast == "slain":
			return "fulfilled", "A great quarry has fallen to you."
		case pr.Resolved && pr.OmenBeast == "spared":
			return "defied", "You walked the wilds and left the quarry alone."
		case g.crTrait(0) >= 70 && g.Day() > 40:
			return "defied", "Your mercy has turned the quarry's price aside, for now."
		case st.Hint[0] || g.P.Level >= 4:
			return "foreshadowed", "Something old in the wilds has noticed you."
		}
	case 1:
		switch {
		case pr.OmenRift == "crossed":
			return "fulfilled", "You stepped through the astral gates."
		case pr.Resolved && pr.OmenRift == "earthbound":
			return "defied", "You kept to the home-soil."
		case st.Hint[1] || (g.vsNearestRift() != nil && g.Day() >= 10):
			return "foreshadowed", "A gate on this world waits for you."
		}
	default:
		switch {
		case pr.OmenCrown == "champion":
			return "fulfilled", "The courts hail you as champion."
		case pr.OmenCrown == "outcast":
			return "fulfilled", "The courts name you outcast."
		case pr.Resolved && pr.OmenCrown == "wanderer":
			return "defied", "You bent the knee to no throne."
		case st.Hint[2] || best >= 40 || worst <= -40:
			return "foreshadowed", "Thrones are beginning to take your measure."
		}
	}
	return "unfulfilled", "Nothing yet speaks of it."
}

func vsStatusColor(s string) string {
	switch s {
	case "fulfilled", "came":
		return "G"
	case "defied":
		return "C"
	case "foreshadowed":
		return "Y"
	case "lapsed":
		return "R"
	}
	return "K"
}

func (g *Game) vsProphecyTick(st *vsState) {
	if g.wy().Prophecy == nil {
		return
	}
	for i := 0; i < 3; i++ {
		s, why := g.vsVerse(i)
		if st.Last[i] != "" && st.Last[i] != s {
			g.vsJournal("prophecy", fmt.Sprintf("Verse %d is %s. %s", i+1, s, why))
			g.Msg(fmt.Sprintf("{{M|Verse %s of the prophecy is %s.}}", []string{"I", "II", "III"}[i], s))
		}
		st.Last[i] = s
	}
}

// ---- turn ----

func vsTurn(g *Game) {
	if g.W == nil || g.P == nil || g.Cur == nil {
		return
	}
	st := g.vs()
	if g.P.St("asleep") > 0 {
		st.Slept++
	} else if st.Slept > 0 {
		n := st.Slept
		st.Slept = 0
		if n >= 90 {
			g.vsDream(n)
		}
	}
	if g.Turn%10 != 0 {
		return
	}
	g.vsOmenTick(st)
	if g.Turn%60 == 0 {
		g.vsProphecyTick(st)
		g.vsJudgeAll(st)
	}
}

// ---- dreams ----

func vsMotifOf(id string) *vsMotif {
	for i := range vsMotifs {
		if vsMotifs[i].ID == id {
			return &vsMotifs[i]
		}
	}
	return &vsMotifs[0]
}

func vsMotifBySrc(src string) *vsMotif {
	for i := range vsMotifs {
		if vsMotifs[i].Src == src {
			return &vsMotifs[i]
		}
	}
	return &vsMotifs[0]
}

func (g *Game) vsDream(mins int) {
	st := g.vs()
	p := g.P
	type srcW struct {
		id string
		w  float64
	}
	var srcs []srcW
	add := func(id string, w float64) {
		if w > 0 {
			srcs = append(srcs, srcW{id, w})
		}
	}
	nem := ""
	if n := nxSt(g).Nem; n != nil && n.Status == "hunt" {
		nem = n.Name
		add("nemesis", 3)
	}
	if len(st.Lost) > 0 {
		add("lost", 3)
	}
	if g.Kills >= 10 || g.crTrait(0) <= -30 {
		add("blood", 1.5+minf(2.5, float64(g.Kills)/40))
	}
	scar := "old"
	if inj := g.ijSt().Inj; len(inj) > 0 {
		scar = strings.ToLower(inj[len(inj)-1].Name)
		add("wound", 2)
	}
	if g.crTrait(0) >= 30 {
		add("mercy", 2)
	}
	if len(srcs) == 0 || g.R.Chance(0.12) {
		if g.R.Chance(0.3) {
			g.wyDream() // the older historical dream of world_story.go
		}
		if len(srcs) == 0 {
			return
		}
		return
	}
	tot := 0.0
	for _, s := range srcs {
		tot += s.w
	}
	f := g.R.Float() * tot
	src := srcs[0].id
	for _, s := range srcs {
		if f < s.w {
			src = s.id
			break
		}
		f -= s.w
	}
	// recurring motif: continue the current one most of the time, else begin the source's own
	if st.Motif != "" && g.R.Chance(0.7) {
		st.Stage = mini(4, st.Stage+1)
	} else {
		nm := vsMotifBySrc(src).ID
		if st.Motif == nm {
			st.Stage = mini(4, st.Stage+1)
		} else {
			st.Motif, st.Stage = nm, 0
		}
	}
	m := vsMotifOf(st.Motif)
	lost := "a friend"
	if len(st.Lost) > 0 {
		lost = st.Lost[len(st.Lost)-1]
	}
	kv := []string{"lost", lost, "nem", nem, "scar", scar, "kills", fmt.Sprint(g.Kills)}
	if nem == "" {
		kv[3] = "a pursuer"
	}
	intro := vsFill(Pick(g.R, vsIntro[src]), kv...)
	line := vsFill(m.Lines[st.Stage], kv...)
	text := intro + " " + line + " " + vsGenreDream[g.vsGenre()]
	st.Dreams++
	stage := st.Stage
	if src == "mercy" {
		p.AddSt("vs_lucid", 360)
		delete(p.Status, "vs_dread")
		g.Msg("{{m|" + text + "}}")
		g.Msg("{{G|You wake at peace. (Lucid)}}")
	} else {
		dc := 8 + stage
		for _, o := range g.vsOmensOn(g.Day()) {
			if o.ID == "bloodmoon" {
				dc += 2
			}
		}
		g.Msg("{{m|" + text + "}}")
		if g.roll2d6Check(p.M(WIL), dc) {
			g.Msg("You wake shaken but steady.")
		} else {
			p.AddSt("vs_dread", 480)
			delete(p.Status, "rested")
			g.Msg("{{R|You wake in a cold sweat, and the rest did not take. (Disturbed rest)}}")
		}
	}
	g.vsJournal("dream", fmt.Sprintf("[%s, stage %d] %s", m.Name, stage+1, text))
	if stage >= 2 && (stage == 4 || g.R.Chance(0.35)) {
		g.vsInsight(stage)
		if stage == 4 {
			g.vsJournal("dream", "The dream of "+m.Name+" has said what it came to say, and is laid to rest.")
			st.Motif, st.Stage = "", 0
		}
	}
}

func (g *Game) vsInsight(stage int) {
	st := g.vs()
	g.Msg("{{G|" + Pick(g.R, vsInsightLines) + "}}")
	g.P.AddSt("vs_lucid", 720)
	g.gainXP(10 + 10*stage)
	for i := 0; i < 3; i++ {
		if !st.Hint[i] && g.wy().Prophecy != nil {
			st.Hint[i] = true
			st.HintTxt[i] = "A dream showed it: " + g.vsPropPlain(i)
			g.Msg("{{M|In the dream you glimpse the prophecy's verse " + []string{"I", "II", "III"}[i] + ": " + g.vsPropPlain(i) + "}}")
			g.vsJournal("dream", "Insight into verse "+fmt.Sprint(i+1)+": "+g.vsPropPlain(i))
			return
		}
	}
	rd := g.vsMake(1, false, "dream", "your sleep")
	g.vsInterpret(rd, false)
	g.Msg("{{M|The dream shows you plainly: " + rd.Plain + "}}")
}

// ---- screen ----

func (g *Game) vsAbilities() []Ability {
	if g.P == nil || g.W == nil {
		return nil
	}
	return []Ability{{ID: "vs_prophecy", Name: "Prophecy, omens and visions", Target: "self"}}
}

func (g *Game) vsAbUse(id string) {
	if id == "vs_prophecy" {
		st := g.vs()
		st.Tab, st.Scroll = 0, 0
		g.Mode = "vs"
	}
}

var vsTabs = []string{"Prophecy", "Readings", "Visions", "Sky"}

func vsKeys(g *Game, k string) {
	st := g.vs()
	switch k {
	case "Escape", "q":
		g.Mode = "main"
	case "Tab", "ArrowRight", "l", "N6":
		st.Tab, st.Scroll = (st.Tab+1)%len(vsTabs), 0
	case "ArrowLeft", "h", "N4":
		st.Tab, st.Scroll = (st.Tab+len(vsTabs)-1)%len(vsTabs), 0
	case "ArrowUp", "k", "N8":
		st.Scroll = maxi(0, st.Scroll-1)
	case "ArrowDown", "j", "N2":
		st.Scroll++
	case "PageUp":
		st.Scroll = maxi(0, st.Scroll-8)
	case "PageDown":
		st.Scroll += 8
	}
}

func (g *Game) vsTabLines(tab, w int) []string {
	st := g.vs()
	var out []string
	wrap := func(s string) { out = append(out, Wrap(s, w)...) }
	switch tab {
	case 0:
		pr := g.wy().Prophecy
		if pr == nil {
			wrap("{{K|No prophecy has yet been spoken over you.}}")
			return out
		}
		for i := 0; i < 3; i++ {
			s, why := g.vsVerse(i)
			wrap(pr.Verses[i])
			wrap(fmt.Sprintf("  {{%s|%s}} {{K|%s}}", vsStatusColor(s), strings.ToUpper(s), why))
			if st.HintTxt[i] != "" {
				wrap("  {{M|Hint:}} " + st.HintTxt[i])
			}
			out = append(out, "")
		}
		if pr.Resolved {
			wrap("{{K|The verses have been weighed at your ending.}}")
		}
	case 1:
		if len(st.Reads) == 0 {
			wrap("{{K|No seer has read for you yet. Seers sit in a few towns; they want coin.}}")
		}
		for i := len(st.Reads) - 1; i >= 0; i-- {
			rd := st.Reads[i]
			wrap(fmt.Sprintf("Day %d, a %s of %s  {{%s|%s}}", rd.Day, rd.Seer, rd.Town, vsStatusColor(rd.Status), rd.Status))
			for _, v := range rd.Verse[1:] {
				wrap("  {{K|" + v + "}}")
			}
			if rd.Interp {
				wrap("  {{G|" + rd.Plain + "}}")
				if rd.Sense != "" {
					wrap("  {{K|" + rd.Sense + "}}")
				}
			} else {
				wrap("  {{K|(its meaning is not yet understood)}}")
			}
			out = append(out, "")
		}
	case 2:
		if st.Motif != "" {
			wrap(fmt.Sprintf("{{M|Recurring dream:}} %s (stage %d of 5)", vsMotifOf(st.Motif).Name, st.Stage+1))
			out = append(out, "")
		}
		n := 0
		for i := len(st.Journal) - 1; i >= 0; i-- {
			j := st.Journal[i]
			wrap(fmt.Sprintf("Day %d {{K|[%s]}} %s", j.Day, j.Kind, j.Text))
			n++
		}
		if n == 0 {
			wrap("{{K|No visions yet. Sleep long, or watch the sky.}}")
		}
	default:
		day := g.Day()
		any := false
		for _, o := range g.vsOmensOn(day) {
			k := g.vsKey(o)
			if st.Seen[k] || st.Heard[k] {
				any = true
				wrap(fmt.Sprintf("{{M|%s}} {{K|%d days left}}", capFirst(g.vsOmenName(o)), o.End-day+1))
				wrap("  " + g.vsSkin(o.ID).Sight)
			}
		}
		for _, f := range st.Fore {
			if f.Planet == g.W.uniIdx && f.Start > day {
				any = true
				wrap(fmt.Sprintf("{{y|Foretold:}} %s in %d days", capFirst(g.vsOmenName(&f)), f.Start-day))
			}
		}
		if !any {
			wrap("{{K|Nothing remarkable is known of the sky. Look up at night, or ask a seer.}}")
		}
		out = append(out, "")
		wrap(fmt.Sprintf("{{K|Effects now: prices %+d%% on some goods, wanderers x%.1f, cults x%.2f.}}", int((g.vsPriceMulAny()-1)*100+0.5), g.vsSpawnMul(), g.vsCultMul()))
	}
	return out
}

func (g *Game) vsPriceMulAny() float64 {
	m := 1.0
	for _, k := range []string{"tonic", "weapon", "food", "artifact"} {
		m = maxf(m, g.vsPriceMul(&Item{Kind: k}))
	}
	return m
}

func (g *Game) vsDraw() {
	sc := g.sc
	st := g.vs()
	x, y, w, h := g.panel("Prophecy, omens and visions")
	var tabs []string
	for i, t := range vsTabs {
		if i == st.Tab {
			tabs = append(tabs, "{{W|["+t+"]}}")
		} else {
			tabs = append(tabs, "{{K|"+t+"}}")
		}
	}
	sc.TextB(x, y, strings.Join(tabs, "  ")+"   {{K|Tab/Left/Right switch  Up/Down scroll  Esc close}}", CGray, CBgDark, x+w)
	y += 2
	lines := g.vsTabLines(st.Tab, w)
	room := h - 4
	if room < 1 {
		room = 1
	}
	if st.Scroll > maxi(0, len(lines)-room) {
		st.Scroll = maxi(0, len(lines)-room)
	}
	for i := st.Scroll; i < len(lines) && i < st.Scroll+room; i++ {
		sc.TextB(x, y, lines[i], CGray, CBgDark, x+w)
		y++
	}
}

// vsLines: plain lines for the journal (Journal+ tab, if wired).
func (g *Game) vsLines() []string {
	var out []string
	for _, l := range g.vsTabLines(0, 100) {
		out = append(out, l)
	}
	for _, l := range g.vsTabLines(2, 100) {
		out = append(out, l)
	}
	return out
}
