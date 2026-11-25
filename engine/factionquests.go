package engine

import (
	"fmt"
	"sort"
	"strings"
)

// Faction quest chains (prefix fq). Once your standing with a faction passes a threshold, its
// representative (a named person placed in a town of that faction when you first walk in) offers a
// 3 to 5 step chain from fqTemplates (factionquests_data.go), built for this planet from real burgs,
// notables, rival realms and dungeon sites with side rngs only (NewRng(seed:fq:...), never g.R).
// Steps are checked by the engine (deliver, meet, scout, kill, fetch, rescue, hold, parley, avenge);
// some carry a choice that moves standing with the faction and its rival and sets flags that later
// steps read (a betrayed contact comes back, a rival starts courting you). Factions covered: realms
// (crowns), the realm's army and watch, faiths, merchant houses and leagues, the thieves' underworld
// and rival-realm spies. Orders, cults, secret societies and sects stay with orders.go/campaigns.go.
// At most 2 chains are active, one per faction. Endings give a rank title, a perk (fqPriceMod and a
// kind-specific bonus), gold and a deed; rivals may turn hostile (agents ambush you on the road) or
// court you. The active step is mirrored into g.Quests as Kind "fq". Screen: mode "fq" (palette
// "Faction work"). State g.ext("fq"), planet-local ("fq" in unLocalKeys).
//
// Hook outside this file (listed in the report): fqPriceMod in talk.go price() (buy and sell branch).

const fqMaxActive = 2

// fqRumourSink, when set, is told whenever a chain step completes (kind "faction").
var fqRumourSink func(g *Game, kind, subject, text string)

func init() {
	unLocalKeys = append(unLocalKeys, "fq")
	turnHooks = append(turnHooks, fqTurn)
	enterHooks = append(enterHooks, fqEnter)
	killHooks = append(killHooks, fqKill)
	talkHooks = append(talkHooks, fqTalk)
	sideHooks = append(sideHooks, fqSide)
	modeKeys["fq"] = fqKeys
	modeDraws["fq"] = func(g *Game) { g.fqDraw() }
	paEntries = append(paEntries,
		paEntry{Name: "Faction work", Keywords: "faction chain quests crown guild house watch faith spies underworld petition standing", Cat: "Screen", Label: "screen",
			Func: func(g *Game) { g.fqOpen() }})
}

type fqStep struct {
	Kind   string  `json:"k"`
	Brief  string  `json:"b"`
	Alt    string  `json:"al,omitempty"`
	If     string  `json:"if,omitempty"`
	Done   string  `json:"d"`
	Need   string  `json:"nd,omitempty"`
	Ch     *fqChT  `json:"ch,omitempty"`
	Days   int     `json:"dy,omitempty"`
	Amb    bool    `json:"am,omitempty"`
	Burg   int     `json:"bg,omitempty"`
	WX     int     `json:"wx,omitempty"`
	WY     int     `json:"wy,omitempty"`
	ZX     int     `json:"zx,omitempty"`
	ZY     int     `json:"zy,omitempty"`
	TZ     string  `json:"tz,omitempty"`
	Person string  `json:"pe,omitempty"`
	CTitle string  `json:"ct,omitempty"`
	Item   string  `json:"it,omitempty"`
	Foe    string  `json:"fo,omitempty"`
	Lvl    int     `json:"lv,omitempty"`
	State  string  `json:"st,omitempty"` // "" pending, act, ready, done, skip
	Placed bool    `json:"pl,omitempty"`
	Freed  bool    `json:"fr,omitempty"`
	AmbDo  bool    `json:"ad,omitempty"`
	Actor  int     `json:"ac,omitempty"`
	Start  int     `json:"s0,omitempty"`
	Due    int     `json:"du,omitempty"`
	Until  int     `json:"un,omitempty"`
	Pick   int     `json:"pk,omitempty"`
}

type fqChain struct {
	ID      int             `json:"id"`
	Fac     string          `json:"f"`
	Tpl     string          `json:"tp"`
	Title   string          `json:"t"`
	Hook    string          `json:"h"`
	Rep     string          `json:"r"`
	RepT    string          `json:"rt"`
	Burg    int             `json:"bg"`
	Rival   string          `json:"rv,omitempty"`
	Lvl     int             `json:"lv"`
	Steps   []*fqStep       `json:"s"`
	Cur     int             `json:"c"`
	State   string          `json:"st,omitempty"` // "" offered, active, done, failed, abandoned
	Flags   map[string]bool `json:"fl,omitempty"`
	RivNet  int             `json:"rn,omitempty"`
	Courted bool            `json:"co,omitempty"`
	Taken   int             `json:"tk,omitempty"`
	QID     int             `json:"q,omitempty"`
	Log     []string        `json:"lg,omitempty"`
}

func (c *fqChain) flag(f string) bool { return c.Flags != nil && c.Flags[f] }
func (c *fqChain) setFlag(f string) {
	if c.Flags == nil {
		c.Flags = map[string]bool{}
	}
	c.Flags[f] = true
}

type fqRep struct {
	Fac    string `json:"f"`
	Burg   int    `json:"b"`
	Name   string `json:"n"`
	Title  string `json:"t"`
	Actor  int    `json:"a,omitempty"`
	Placed bool   `json:"p,omitempty"`
}

type fqState struct {
	Next    int                 `json:"n"`
	Active  []*fqChain          `json:"a,omitempty"`
	Offers  map[string]*fqChain `json:"o,omitempty"`
	Reps    map[string]*fqRep   `json:"r,omitempty"`
	Stand   map[string]int      `json:"s,omitempty"`
	Rank    map[string]int      `json:"k,omitempty"`
	Done    map[string][]string `json:"d,omitempty"`
	Courted map[string]bool     `json:"c,omitempty"`
	Hostile map[string]int      `json:"h,omitempty"`
	Cool    map[string]int      `json:"cl,omitempty"`
	Log     []string            `json:"l,omitempty"`
	DoneN   int                 `json:"dn,omitempty"`
	FailN   int                 `json:"fn,omitempty"`
	Hour    int                 `json:"hr,omitempty"`
	NextAmb int                 `json:"na,omitempty"`

	tab, sel int
	conf     string
	facs     []*fqFac
	facW     *WorldData
	byKey    map[string]*fqFac
}

func (g *Game) fq() *fqState {
	s := g.ext("fq", func() any {
		return &fqState{Offers: map[string]*fqChain{}, Reps: map[string]*fqRep{}, Stand: map[string]int{}, Rank: map[string]int{},
			Done: map[string][]string{}, Courted: map[string]bool{}, Hostile: map[string]int{}, Cool: map[string]int{}}
	}).(*fqState)
	if s.Offers == nil {
		s.Offers = map[string]*fqChain{}
	}
	if s.Reps == nil {
		s.Reps = map[string]*fqRep{}
	}
	if s.Stand == nil {
		s.Stand = map[string]int{}
	}
	if s.Rank == nil {
		s.Rank = map[string]int{}
	}
	if s.Done == nil {
		s.Done = map[string][]string{}
	}
	if s.Courted == nil {
		s.Courted = map[string]bool{}
	}
	if s.Hostile == nil {
		s.Hostile = map[string]int{}
	}
	if s.Cool == nil {
		s.Cool = map[string]int{}
	}
	return s
}

// ---------------------------------------------------------------- factions

type fqFac struct {
	Key, Kind, Name string
	State, Faith    int
	Burg, Opp       int
	Need            int
	Rival           string
}

type fqKindInfo struct {
	Role     string
	RepTitle []string
	CTitle   []string
	Ranks    [4]string
	Perk     string
	Label    string
}

var fqKinds = map[string]fqKindInfo{
	"realm": {"elder", []string{"royal envoy", "seneschal", "crown reeve", "chamberlain"}, []string{"magistrate", "castellan", "quartermaster", "tax-clerk", "steward"},
		[4]string{"Sworn Courier", "Warden's Hand", "Knight-Errant", "Champion"}, "2% off in the realm's shops and 2% more for what you sell there, per rank", "Crown"},
	"faith": {"priest", []string{"almoner", "canon", "abbess", "lay-warden"}, []string{"shrine-keeper", "sexton", "pilgrim-guide", "deacon", "cantor"},
		[4]string{"Lay Pilgrim", "Almoner's Hand", "Canon of Service", "Chosen Guardian"}, "2% off at shops of the faithful, and a full healing at each chain's end", "Faith"},
	"watch": {"guard", []string{"watch-captain", "sergeant-major", "marshal's aide", "warden of the gate"}, []string{"sergeant", "informant", "gate-sergeant", "turnkey", "scout"},
		[4]string{"Auxiliary", "Corporal-at-large", "Lieutenant's Eyes", "Captain's Peer"}, "army merit at each chain's end if you are enlisted, and the watch's goodwill", "Army and watch"},
	"house": {"scholar", []string{"factor", "house steward", "counting-master", "guild clerk"}, []string{"debtor", "clerk", "dockmaster", "warehouse-keeper", "bookkeeper"},
		[4]string{"Trusted Runner", "House Factor", "Counting-House Partner", "Voice of the House"}, "3% off at the house's realm's traders, and 3% more when you sell to them, per rank", "Merchant house"},
	"uw": {"villager", []string{"fixer", "quiet one", "old hand", "broker"}, []string{"fence", "lookout", "cutpurse", "snitch", "dealer"},
		[4]string{"Cutter's Friend", "Made Hand", "Lieutenant of the Dark", "Whisper-Lord's Equal"}, "fences pay 3% more and sell 2% cheaper, per rank, and outlaws warm to you", "Underworld"},
	"spy": {"villager", []string{"stranger", "courier", "quiet traveller", "cipher-clerk"}, []string{"courier", "clerk", "tavern-keeper", "dead-drop keeper", "ferryman"},
		[4]string{"Courier", "Asset", "Handler's Trust", "Spymaster's Ear"}, "a map of the rival realm's towns at each chain's end", "Spies"},
}

var fqEpithets = []string{"Red", "Black", "Gaunt", "One-Eyed", "Hollow", "Grey", "Laughing", "Lame", "Pale", "Iron", "Quiet", "Crooked"}

func (g *Game) fqEnemy(sid int) int {
	s := g.W.stateByID[sid]
	if s == nil {
		return 0
	}
	for _, o := range s.Wars {
		if os := g.W.stateByID[o]; os != nil && os.Alive && o > 0 && o != sid {
			return o
		}
	}
	var ids []int
	for id := range s.Relations {
		if os := g.W.stateByID[id]; os != nil && os.Alive && id > 0 && id != sid {
			ids = append(ids, id)
		}
	}
	sort.Ints(ids)
	best, bv := 0, 1<<30
	for _, id := range ids {
		if v := s.Relations[id]; v < bv {
			best, bv = id, v
		}
	}
	return best
}

func (g *Game) fqFacs() []*fqFac {
	st := g.fq()
	if st.facs != nil && st.facW == g.W {
		return st.facs
	}
	w := g.W
	var out []*fqFac
	for i := range w.States {
		s := &w.States[i]
		if !s.Alive || s.ID <= 0 {
			continue
		}
		rival := ""
		if e := g.fqEnemy(s.ID); e > 0 {
			rival = stateFaction(e)
		}
		nm := s.FullName
		if nm == "" {
			nm = s.Name
		}
		out = append(out, &fqFac{Key: stateFaction(s.ID), Kind: "realm", Name: nm, State: s.ID, Need: 20, Rival: rival})
		out = append(out, &fqFac{Key: fmt.Sprintf("watch:%d", s.ID), Kind: "watch", Name: "the army and watch of " + s.Name, State: s.ID, Need: 25, Rival: "uw"})
		for _, o := range s.Wars {
			if os := w.stateByID[o]; os != nil && os.Alive && o > 0 && o != s.ID {
				out = append(out, &fqFac{Key: fmt.Sprintf("spy:%d:%d", o, s.ID), Kind: "spy", Name: "the Quiet Hand of " + os.Name, State: o, Opp: s.ID, Need: 15, Rival: stateFaction(s.ID)})
			}
		}
	}
	for i := range w.Religions {
		rl := &w.Religions[i]
		rival := ""
		for j := range w.Religions {
			if o := &w.Religions[j]; o.ID != rl.ID && o.Type != rl.Type {
				rival = fmt.Sprintf("faith:%d", o.ID)
				break
			}
		}
		if rival == "" {
			for j := range w.Religions {
				if o := &w.Religions[j]; o.ID != rl.ID {
					rival = fmt.Sprintf("faith:%d", o.ID)
					break
				}
			}
		}
		out = append(out, &fqFac{Key: fmt.Sprintf("faith:%d", rl.ID), Kind: "faith", Name: rl.Name, Faith: rl.ID, Need: 20, Rival: rival})
	}
	var cities []*BurgData
	leagues := map[string]bool{}
	for i := range w.Burgs {
		b := &w.Burgs[i]
		if !b.Alive || b.State <= 0 || b.P < 0 || b.P >= w.W*w.H || (b.Layer != "" && b.Layer != "surface") {
			continue
		}
		if w.cityTier[b.ID] >= 1 {
			cities = append(cities, b)
		}
		if b.League != "" && !leagues[b.League] {
			leagues[b.League] = true
			out = append(out, &fqFac{Key: "league:" + b.League, Kind: "house", Name: b.League, State: b.State, Burg: b.ID, Need: 12, Rival: "uw"})
		}
	}
	for i, b := range cities {
		hn := ""
		if cu := w.cultByID[b.Culture]; cu != nil && len(cu.Houses) > 0 {
			hn = cu.Houses[int(HashStr(fmt.Sprint(g.Seed, ":fq:house:", b.ID))%uint64(len(cu.Houses)))]
		}
		name := "the merchants' house of " + b.Name
		if hn != "" {
			name = "House " + hn + " of " + b.Name
		}
		rival := "uw"
		if len(cities) > 1 {
			o := cities[(i+1+int(HashStr(fmt.Sprint(g.Seed, ":fq:hrival:", b.ID))%uint64(len(cities)-1)))%len(cities)]
			if o.ID != b.ID {
				rival = fmt.Sprintf("house:%d", o.ID)
			}
		}
		out = append(out, &fqFac{Key: fmt.Sprintf("house:%d", b.ID), Kind: "house", Name: name, State: b.State, Burg: b.ID, Need: 12, Rival: rival})
	}
	out = append(out, &fqFac{Key: "uw", Kind: "uw", Name: "the thieves' underworld", Need: 25})
	st.facs, st.facW = out, w
	st.byKey = map[string]*fqFac{}
	for _, f := range out {
		st.byKey[f.Key] = f
	}
	return out
}

func (g *Game) fqFac(key string) *fqFac {
	g.fqFacs()
	return g.fq().byKey[key]
}

func (g *Game) fqName(key string) string {
	if key == "" {
		return "its rivals"
	}
	if f := g.fqFac(key); f != nil {
		return f.Name
	}
	return g.FactionName(key)
}

func (g *Game) fqStanding(f *fqFac) int {
	st := g.fq()
	own := st.Stand[f.Key]
	switch f.Kind {
	case "realm", "faith":
		return g.Rep[f.Key]
	case "watch":
		s := own + g.Rep[stateFaction(f.State)]/2
		if es := g.war(); es.Realm == f.State {
			s += 10 + g.warRank(es)*5 + es.Merit/5
		}
		return s
	case "house":
		return own + g.Rep[stateFaction(f.State)]/3
	case "uw":
		u := g.uwSt()
		return own + maxi(0, g.Rep["outlaws"]+400) + 5*(u.Stolen+u.Fenced+u.Wins+u.Won)
	case "spy":
		return own + g.Rep[stateFaction(f.Opp)]/2
	}
	return own
}

func (g *Game) fqEligible(f *fqFac) bool {
	if g.fq().Courted[f.Key] {
		return true
	}
	return g.fqStanding(f) >= f.Need
}

func (g *Game) fqMove(key string, d int, why string) {
	if d == 0 || key == "" {
		return
	}
	st := g.fq()
	f := g.fqFac(key)
	if f == nil {
		if strings.HasPrefix(key, "state:") || strings.HasPrefix(key, "faith:") {
			g.AddRep(key, d, why)
		}
		return
	}
	st.Stand[key] = clampi(st.Stand[key]+d, -60, 300)
	if f.Kind == "realm" || f.Kind == "faith" {
		g.AddRep(key, d, why)
		return
	}
	col, sg := "G", "+"
	if d < 0 {
		col, sg = "R", ""
	}
	g.Msg(fmt.Sprintf("{{%s|%s%d}} standing with {{Y|%s}}%s.", col, sg, d, f.Name, why))
}

func (g *Game) fqRankTitle(f *fqFac, rank int) string {
	if rank <= 0 {
		return ""
	}
	ki := fqKinds[f.Kind]
	return ki.Ranks[clampi(rank, 1, 4)-1] + " of " + f.Name
}

func (g *Game) fqTitleOf(key string) string {
	f := g.fqFac(key)
	if f == nil {
		return ""
	}
	return g.fqRankTitle(f, g.fq().Rank[key])
}

// fqPriceMod: perks from finished chains. buy = what you pay (<= 1), else what you are paid (>= 1).
// Hook: talk.go price(): buy branch `f *= fqPriceMod(g, g.Talk, true)` before the 0.75 floor, sell branch
// `f *= fqPriceMod(g, g.Talk, false)` after the hmPriceMod line.
func fqPriceMod(g *Game, a *Actor, buy bool) float64 {
	if g == nil || g.W == nil || a == nil {
		return 1
	}
	st := g.fq()
	if len(st.Rank) == 0 {
		return 1
	}
	day := g.Day()
	f := 1.0
	for key, rk := range st.Rank {
		if rk <= 0 || st.Hostile[key] > day {
			continue
		}
		fc := g.fqFac(key)
		if fc == nil {
			continue
		}
		var pct float64
		switch fc.Kind {
		case "realm":
			if a.State == fc.State || a.Faction == key {
				pct = 0.02 * float64(rk)
			}
		case "faith":
			if fc.Faith > 0 && a.Religion == fc.Faith {
				pct = 0.02 * float64(rk)
			}
		case "house":
			if fc.State > 0 && (a.State == fc.State || a.Faction == stateFaction(fc.State)) && isTrader(a) {
				pct = 0.03 * float64(rk)
			}
		case "uw":
			if uwIsFence(a) {
				if buy {
					pct = 0.02 * float64(rk)
				} else {
					pct = 0.03 * float64(rk)
				}
			}
		}
		if buy {
			f *= 1 - pct
		} else {
			f *= 1 + pct
		}
	}
	if buy && f < 0.85 {
		f = 0.85
	}
	if !buy && f > 1.15 {
		f = 1.15
	}
	return f
}

// ---------------------------------------------------------------- small helpers

func fqFill(t string, kv []string) string { return strings.NewReplacer(kv...).Replace(t) }

func fqCap(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func (g *Game) fqLog(c *fqChain, s string) {
	st := g.fq()
	line := fmt.Sprintf("Day %d: %s", g.Day()+1, s)
	st.Log = append(st.Log, line)
	if len(st.Log) > 14 {
		st.Log = st.Log[len(st.Log)-14:]
	}
	if c != nil {
		c.Log = append(c.Log, line)
		if len(c.Log) > 12 {
			c.Log = c.Log[len(c.Log)-12:]
		}
	}
}

func (g *Game) fqNews(burg int, format string, a ...any) {
	lvSt(g).addNews(lvDay(g), burg, format, a...)
}

func (g *Game) fqSink(f *fqFac, text string) {
	if fqRumourSink != nil && f != nil {
		fqRumourSink(g, "faction", f.Name, text)
	}
}

func fqNeedOK(c *fqChain, need string) bool {
	if need == "" {
		return true
	}
	if strings.HasPrefix(need, "!") {
		return !c.flag(need[1:])
	}
	return c.flag(need)
}

func (g *Game) fqBurgXY(id int) (int, int, bool) { return g.jbBurgXY(id) }

func (g *Game) fqBurgName(id int) string { return g.jbBurgName(id) }

func (g *Game) fqInv(id, idx int) *Item {
	for _, it := range g.P.Inv {
		if it.Extra != nil && it.Extra["fq"] == id && it.Extra["fqs"] == idx+1 {
			return it
		}
	}
	return nil
}

func (g *Game) fqDropItems(id int) {
	for _, it := range append([]*Item(nil), g.P.Inv...) {
		if it.Extra != nil && it.Extra["fq"] == id {
			g.removeInv(it)
		}
	}
}

func (g *Game) fqActorAnywhere(id int) *Actor {
	if id == 0 {
		return nil
	}
	a, _ := g.jbActorAnywhere(id)
	return a
}

func (g *Game) fqStepOf(c *fqChain) *fqStep {
	if c == nil || c.Cur < 0 || c.Cur >= len(c.Steps) {
		return nil
	}
	return c.Steps[c.Cur]
}

func (s *fqStep) brief(c *fqChain) string {
	if s.If != "" && s.Alt != "" && c.flag(s.If) {
		return s.Alt
	}
	return s.Brief
}

func (g *Game) fqActiveFor(key string) *fqChain {
	for _, c := range g.fq().Active {
		if c.Fac == key {
			return c
		}
	}
	return nil
}

func fqIsTown(zn *Zone) bool { return zn != nil && zn.Kind == "town" && zn.Burg > 0 }

func fqWildZone(zn *Zone) bool { return zn != nil && zn.Z == 0 && zn.Kind != "town" }

func (g *Game) fqChance(zn *Zone, tag string, pct uint64) bool {
	return HashStr(fmt.Sprint(g.Seed, ":fq:rnd:", zn.Key, ":", g.Day(), ":", tag))%100 < pct
}

// ---------------------------------------------------------------- generation

type fqSiteT struct {
	WX, WY, ZX, ZY int
	Name, TZ       string
	Bonus          int
}

func (g *Game) fqSite(r *Rng, home *BurgData) (fqSiteT, bool) {
	w := g.W
	var ds []jbSite
	for _, s := range g.jbSites(home) {
		if s.S.Kind == "dungeon" {
			ds = append(ds, s)
		}
	}
	bx, by := home.P%w.W, home.P/w.W
	var ruins []*BurgData
	for i := range w.Burgs {
		b := &w.Burgs[i]
		if b.Alive || b.P < 0 || b.P >= w.W*w.H || (b.Layer != "" && b.Layer != "surface") || w.IsWater(b.P) {
			continue
		}
		if maxi(abs(b.P%w.W-bx), abs(b.P/w.W-by)) <= 8 {
			ruins = append(ruins, b)
		}
	}
	if len(ruins) > 0 && (len(ds) == 0 || r.Chance(0.35)) {
		b := ruins[r.Int(0, len(ruins)-1)]
		x, y := b.P%w.W, b.P/w.W
		return fqSiteT{WX: x, WY: y, ZX: 1, ZY: 1, Name: "the ruins of " + b.Name, TZ: ZoneKey(x, y, 1, 1, 0)}, true
	}
	if len(ds) == 0 {
		return fqSiteT{}, false
	}
	s := ds[r.Int(0, len(ds)-1)]
	return fqSiteT{WX: s.WX, WY: s.WY, ZX: s.ZX, ZY: s.ZY, Name: s.S.Name, TZ: ZoneKey(s.WX, s.WY, s.ZX, s.ZY, 1), Bonus: s.S.Bonus}, true
}

func (g *Game) fqWild(r *Rng, home *BurgData, lo, hi int) (int, int, bool) {
	w := g.W
	bx, by := home.P%w.W, home.P/w.W
	for i := 0; i < 80; i++ {
		x, y := bx+r.Int(-hi, hi), by+r.Int(-hi, hi)
		if maxi(abs(x-bx), abs(y-by)) >= lo && w.In(x, y) && !w.IsWater(w.Idx(x, y)) && w.Burg[w.Idx(x, y)] == 0 {
			return x, y, true
		}
	}
	return 0, 0, false
}

func (g *Game) fqPickTown(r *Rng, home *BurgData, f *fqFac, used map[int]bool) *BurgData {
	w := g.W
	bx, by := home.P%w.W, home.P/w.W
	pool := func(lo, hi int, sameState bool) []*BurgData {
		var out []*BurgData
		for i := range w.Burgs {
			b := &w.Burgs[i]
			if !b.Alive || b.ID == home.ID || used[b.ID] || b.P < 0 || b.P >= w.W*w.H || (b.Layer != "" && b.Layer != "surface") || w.IsWater(b.P) {
				continue
			}
			d := maxi(abs(b.P%w.W-bx), abs(b.P/w.W-by))
			if d < lo || d > hi || (sameState && b.State != home.State) {
				continue
			}
			out = append(out, b)
		}
		return out
	}
	same := f.Kind == "realm" || f.Kind == "watch" || f.Kind == "house"
	for _, try := range []struct {
		lo, hi int
		ss     bool
	}{{2, 9, same}, {2, 9, false}, {1, 18, false}} {
		if p := pool(try.lo, try.hi, try.ss); len(p) > 0 {
			return p[r.Int(0, len(p)-1)]
		}
	}
	return home
}

func (g *Game) fqDead(r *Rng, home *BurgData) string {
	var names []string
	for i := range g.W.People {
		p := &g.W.People[i]
		if !p.Alive && p.Name != "" && p.State == home.State {
			names = append(names, p.Name)
		}
	}
	if len(names) == 0 {
		return "a forgotten ruler"
	}
	return names[r.Int(0, len(names)-1)]
}

var fqItemWords = map[string][]string{
	"crown_writ":    {"a sealed writ", "a wax-sealed writ", "an oilcloth packet of writs"},
	"crown_seal":    {"the seal of %s", "the signet ring of %s"},
	"faith_pilgrim": {"a purse of offering-silver", "a chest of votive candles and silver", "a bundle of incense and tithe"},
	"faith_hymnal":  {"the forbidden hymnal of %s", "a heretic's hymnal said to be the work of %s"},
	"watch_cache":   {"the smugglers' tally-book", "a sealed crate of tally-sticks and ledgers"},
	"house_caravan": {"the caravan's strongbox", "the sealed cargo of spices and cloth", "a chest of the house's trade goods"},
	"house_debt":    {"a stamped receipt", "a notarised receipt of debt"},
	"house_ledger":  {"the rival's second ledger", "a hidden counting-book"},
	"uw_job":        {"the vault's jewelled reliquary", "a strongbox of old gems", "the collector's sealed coffer"},
	"spy_drop":      {"a coded letter", "a letter in a folded cipher"},
	"spy_double":    {"the sealed plans of the {enemy} marshals", "the {enemy} border plans"},
}

func (g *Game) fqFoeName(r *Rng, cu *CultureData, home *BurgData, withOutlaw bool) string {
	given := g.cultureName(cu, r)
	if i := strings.Index(given, " "); i > 0 {
		given = given[:i]
	}
	foe := Pick(r, fqEpithets) + " " + given
	if withOutlaw {
		w := g.W
		bx, by := home.P%w.W, home.P/w.W
		var near []*OutlawData
		for i := range w.Outlaws {
			o := &w.Outlaws[i]
			if o.P >= 0 && o.P < w.W*w.H && maxi(abs(o.P%w.W-bx), abs(o.P/w.W-by)) <= 8 {
				near = append(near, o)
			}
		}
		if len(near) > 0 {
			foe += ", chief of " + near[r.Int(0, len(near)-1)].Name
		}
	}
	return foe
}

// fqBuild makes one chain from a template for a representative's town; nil when the world cannot host it.
func (g *Game) fqBuild(f *fqFac, t *fqTplT, rep *fqRep, nth int) *fqChain {
	w := g.W
	home := w.burgByID[rep.Burg]
	if home == nil || home.P < 0 || home.P >= w.W*w.H {
		return nil
	}
	r := NewRng(fmt.Sprint(g.Seed, ":fq:chain:", f.Key, ":", t.ID, ":", rep.Burg, ":", nth))
	hx, hy := home.P%w.W, home.P/w.W
	rival := f.Rival
	if f.Kind == "uw" {
		rival = fmt.Sprintf("watch:%d", home.State)
	}
	enemy := "a rival realm"
	if e := g.fqEnemy(home.State); e > 0 {
		enemy = g.realmName(e)
	}
	c := &fqChain{Fac: f.Key, Tpl: t.ID, Rep: rep.Name, RepT: rep.Title, Burg: rep.Burg, Rival: rival, Flags: map[string]bool{}}
	c.Lvl = maxi(2, g.P.Level/3+2)
	base := []string{"{rep}", rep.Name, "{fac}", f.Name, "{rival}", g.fqName(rival), "{realm}", g.realmName(home.State), "{town}", home.Name, "{enemy}", enemy}
	cur := map[string]string{"{person}": "", "{town2}": "", "{item}": "", "{foe}": "", "{site}": "", "{dir}": "", "{n}": "", "{contact}": ""}
	first := map[string]string{}
	set := func(k, v string) {
		cur[k] = v
		if _, ok := first[k]; !ok {
			first[k] = v
		}
	}
	setDir := func(x, y int) {
		set("{dir}", jbDirName(x-hx, y-hy))
		set("{n}", fmt.Sprint(jbPar(x-hx, y-hy)))
	}
	ki := fqKinds[f.Kind]
	used := map[int]bool{}
	contact := ""
	for si, ts := range t.Steps {
		s := &fqStep{Kind: ts.Kind, Need: ts.Need, Ch: nil, Days: ts.Days, Amb: ts.Amb, If: ts.If}
		lvlAt := func(x, y, z int) int { return maxi(2, g.zoneLevel(x, y, z)+g.P.Level/5) }
		switch ts.Kind {
		case "meet", "deliver":
			d := g.fqPickTown(r, home, f, used)
			used[d.ID] = true
			s.Burg = d.ID
			s.Person = g.cultureName(w.cultByID[d.Culture], r)
			s.CTitle = Pick(r, ki.CTitle)
			contact = s.Person
			set("{person}", s.Person)
			set("{town2}", d.Name)
			if x, y, ok := g.fqBurgXY(d.ID); ok {
				setDir(x, y)
			}
			if ts.Kind == "deliver" {
				words := fqItemWords[t.ID]
				if len(words) == 0 {
					words = []string{"a sealed packet"}
				}
				it := Pick(r, words)
				if strings.Contains(it, "%s") {
					it = fmt.Sprintf(it, g.fqDead(r, home))
				}
				s.Item = fqFill(it, base)
				set("{item}", s.Item)
			}
		case "scout":
			x, y, ok := g.fqWild(r, home, 2, 7)
			if !ok {
				return nil
			}
			s.WX, s.WY = x, y
			setDir(x, y)
		case "kill":
			x, y, ok := g.fqWild(r, home, 2, 7)
			if !ok {
				return nil
			}
			s.WX, s.WY = x, y
			s.Lvl = lvlAt(x, y, 0) + 1
			s.Foe = g.fqFoeName(r, w.cultByID[home.Culture], home, r.Chance(0.6))
			set("{foe}", s.Foe)
			setDir(x, y)
		case "fetch", "rescue":
			site, ok := g.fqSite(r, home)
			if !ok {
				return nil
			}
			s.WX, s.WY, s.ZX, s.ZY, s.TZ = site.WX, site.WY, site.ZX, site.ZY, site.TZ
			z := 1
			if strings.HasPrefix(site.Name, "the ruins of") {
				z = 0
			}
			s.Lvl = lvlAt(site.WX, site.WY, z) + site.Bonus
			set("{site}", site.Name)
			setDir(site.WX, site.WY)
			if ts.Kind == "fetch" {
				words := fqItemWords[t.ID]
				if len(words) == 0 {
					words = []string{"a lost strongbox"}
				}
				it := Pick(r, words)
				if strings.Contains(it, "%s") {
					it = fmt.Sprintf(it, g.fqDead(r, home))
				}
				s.Item = fqFill(it, base)
				set("{item}", s.Item)
			} else {
				s.Person = g.cultureName(w.cultByID[home.Culture], r)
				set("{person}", s.Person)
			}
		case "hold":
			if s.Days <= 0 {
				s.Days = 2
			}
		case "avenge":
			s.Foe = g.fqFoeName(r, w.cultByID[home.Culture], home, false)
			if !strings.Contains(ts.Brief, "{foe}") && contact != "" {
				s.Foe = contact
			}
			if contact == "" {
				contact = s.Foe
			}
			s.Lvl = maxi(2, g.P.Level-1)
			set("{foe}", s.Foe)
			s.Person = contact
		}
		cur["{contact}"] = contact
		kv := append([]string{}, base...)
		kv = append(kv, "{days}", fmt.Sprint(maxi(1, s.Days)))
		for k, v := range cur {
			kv = append(kv, k, v)
		}
		s.Brief = fqFill(ts.Brief, kv)
		s.Done = fqFill(ts.Done, kv)
		if ts.Alt != "" {
			s.Alt = fqFill(ts.Alt, kv)
		}
		if ts.Ch != nil {
			ch := &fqChT{Prompt: fqFill(ts.Ch.Prompt, kv)}
			for _, o := range ts.Ch.Opts {
				o.Text, o.Win, o.Lose = fqFill(o.Text, kv), fqFill(o.Win, kv), fqFill(o.Lose, kv)
				ch.Opts = append(ch.Opts, o)
			}
			s.Ch = ch
		}
		if ts.Kind == "hold" {
			s.Days = maxi(1, s.Days)
		} else if s.Days == 0 && (ts.Kind == "kill" || ts.Kind == "fetch" || ts.Kind == "rescue" || ts.Kind == "deliver") {
			s.Days = 12
		}
		_ = si
		c.Steps = append(c.Steps, s)
	}
	fk := []string{}
	fk = append(fk, base...)
	for k, v := range first {
		fk = append(fk, k, v)
	}
	fk = append(fk, "{contact}", contact)
	c.Title = fqFill(t.Title, base)
	c.Hook = fqFill(t.Hook, fk)
	if g.fq().Courted[f.Key] {
		c.Courted = true
		c.Hook = "Word of your dealings with " + g.fqName(rival) + " has reached " + f.Name + ". " + c.Hook
	}
	return c
}

func (g *Game) fqOfferFor(f *fqFac, rep *fqRep) *fqChain {
	st := g.fq()
	if c := st.Offers[f.Key]; c != nil {
		return c
	}
	var cands []*fqTplT
	for i := range fqTemplates {
		t := &fqTemplates[i]
		if t.Kinds == f.Kind {
			cands = append(cands, t)
		}
	}
	done := st.Done[f.Key]
	var fresh []*fqTplT
	for _, t := range cands {
		seen := false
		for _, d := range done {
			if d == t.ID {
				seen = true
			}
		}
		if !seen {
			fresh = append(fresh, t)
		}
	}
	if len(fresh) > 0 {
		cands = fresh
	}
	if len(cands) == 0 {
		return nil
	}
	start := int(HashStr(fmt.Sprint(g.Seed, ":fq:pick:", f.Key, ":", len(done))) % uint64(len(cands)))
	for k := 0; k < len(cands); k++ {
		t := cands[(start+k)%len(cands)]
		if c := g.fqBuild(f, t, rep, len(done)); c != nil {
			st.Offers[f.Key] = c
			return c
		}
	}
	return nil
}

// ---------------------------------------------------------------- representatives

func fqRepFits(g *Game, f *fqFac, bd *BurgData) bool {
	switch f.Kind {
	case "realm", "watch":
		return bd.State == f.State
	case "house":
		if strings.HasPrefix(f.Key, "league:") {
			return bd.League == f.Name
		}
		return bd.ID == f.Burg
	case "faith":
		i := clampi(bd.P, 0, len(g.W.Religion)-1)
		return g.W.Religion[i] == f.Faith
	case "uw":
		return true
	case "spy":
		return bd.State == f.Opp
	}
	return false
}

func (g *Game) fqEnsureReps(zn *Zone) {
	if !fqIsTown(zn) {
		return
	}
	bd := g.W.burgByID[zn.Burg]
	if bd == nil || !bd.Alive {
		return
	}
	st := g.fq()
	day := g.Day()
	fresh := 0
	for _, f := range g.fqFacs() {
		if !fqRepFits(g, f, bd) || st.Hostile[f.Key] > day {
			continue
		}
		rp := st.Reps[f.Key]
		if rp == nil {
			if !g.fqEligible(f) || fresh >= 3 {
				continue
			}
			fresh++
			ki := fqKinds[f.Kind]
			r := NewRng(fmt.Sprint(g.Seed, ":fq:rep:", f.Key, ":", bd.ID))
			rp = &fqRep{Fac: f.Key, Burg: bd.ID, Name: g.cultureName(g.W.cultByID[bd.Culture], r), Title: Pick(r, ki.RepTitle)}
			st.Reps[f.Key] = rp
		}
		if rp.Burg != bd.ID {
			continue
		}
		if rp.Placed {
			found := false
			for _, a := range zn.Actors {
				if a.ID == rp.Actor {
					found = true
					break
				}
			}
			if found || g.fqActorAnywhere(rp.Actor) != nil {
				continue
			}
		}
		g.fqPlaceRep(f, rp, zn, bd)
	}
}

func (g *Game) fqPlaceRep(f *fqFac, rp *fqRep, zn *Zone, bd *BurgData) {
	ki := fqKinds[f.Kind]
	r := NewRng(fmt.Sprint(g.Seed, ":fq:repa:", f.Key, ":", bd.ID))
	a := g.makeNPC(ki.Role, bd, g.W.cultByID[bd.Culture], r)
	a.Name, a.Proper, a.Title = rp.Name, true, rp.Title
	a.Desc = fmt.Sprintf("%s, %s for %s. They look at you as if weighing a debt.", rp.Name, rp.Title, f.Name)
	a.AI = "still"
	a.Set("fqrep")
	x, y, ok := zn.FreeSpot(r, nil)
	if !ok {
		return
	}
	a.X, a.Y, a.HomeX, a.HomeY = x, y, x, y
	zn.Actors = append(zn.Actors, a)
	first := !rp.Placed
	rp.Placed, rp.Actor = true, a.ID
	if first {
		g.Msg(fmt.Sprintf("{{c|%s, %s for %s, has been asking after you. Talk to them for work.}}", rp.Name, rp.Title, f.Name))
	}
}

func (g *Game) fqRepOfActor(a *Actor) (*fqFac, *fqRep) {
	for k, rp := range g.fq().Reps {
		if rp.Actor == a.ID {
			return g.fqFac(k), rp
		}
	}
	return nil, nil
}

// ---------------------------------------------------------------- chain lifecycle

func (g *Game) fqTarget(c *fqChain) (int, int) {
	s := g.fqStepOf(c)
	hx, hy, _ := g.fqBurgXY(c.Burg)
	if s == nil {
		return hx, hy
	}
	switch {
	case s.State == "ready":
		return hx, hy
	case s.Kind == "meet" || s.Kind == "deliver":
		if x, y, ok := g.fqBurgXY(s.Burg); ok {
			return x, y
		}
	case s.Kind == "scout" || s.Kind == "kill" || s.Kind == "fetch":
		return s.WX, s.WY
	case s.Kind == "rescue":
		if s.Freed {
			return hx, hy
		}
		return s.WX, s.WY
	case s.Kind == "avenge":
		return g.Cur.WX, g.Cur.WY
	}
	return hx, hy
}

func (g *Game) fqStepCount(c *fqChain) (int, int) {
	n, at := 0, 0
	for i, s := range c.Steps {
		if s.State == "skip" || (s.State == "" && !fqNeedOK(c, s.Need) && i > c.Cur) {
			continue
		}
		n++
		if i <= c.Cur {
			at = n
		}
	}
	return at, n
}

func (g *Game) fqMirror(c *fqChain) {
	s := g.fqStepOf(c)
	if s == nil {
		return
	}
	tx, ty := g.fqTarget(c)
	tp := 0
	if g.W.In(tx, ty) {
		tp = g.W.Idx(tx, ty)
	}
	at, n := g.fqStepCount(c)
	title := fmt.Sprintf("%s (%d/%d)", c.Title, at, n)
	text := s.brief(c)
	if s.State == "ready" {
		text = "Done: " + s.Done + " Report to " + c.Rep + " in " + g.fqBurgName(c.Burg) + "."
	}
	giver := c.Rep + " of " + g.fqName(c.Fac)
	for _, q := range g.Quests {
		if q.ID == c.QID && c.QID != 0 {
			q.Title, q.Text, q.Target, q.TP = title, text, ZoneKey(tx, ty, 1, 1, 0), tp
			g.Known[tp] = true
			return
		}
	}
	mq := &Quest{ID: g.newID(), Kind: "fq", Title: title, Text: text, Giver: giver, Faction: "player", Target: ZoneKey(tx, ty, 1, 1, 0), TP: tp}
	c.QID = mq.ID
	g.Quests = append(g.Quests, mq)
	g.Known[tp] = true
}

func (g *Game) fqCloseMirror(c *fqChain) {
	for _, q := range g.Quests {
		if q.ID == c.QID && c.QID != 0 {
			q.Done, q.Closed = true, true
		}
	}
}

func (g *Game) fqTrack(c *fqChain) {
	tx, ty := g.fqTarget(c)
	if g.W.In(tx, ty) {
		g.ceSetQuestTarget(tx, ty)
		g.Msg(fmt.Sprintf("{{c|Tracking: %s. The quest arrow points to (%d,%d).}}", c.Title, tx, ty))
	}
}

func (g *Game) fqUntrack(c *fqChain) {
	ce := g.ce()
	if !ce.HasWaypoint {
		return
	}
	tx, ty := g.fqTarget(c)
	if ce.TargetWX == tx && ce.TargetWY == ty {
		ce.HasWaypoint = false
	}
}

func (g *Game) fqAccept(f *fqFac, c *fqChain) bool {
	st := g.fq()
	if len(st.Active) >= fqMaxActive {
		g.Msg(fmt.Sprintf("{{y|You already carry %d faction chains. Finish or abandon one.}}", fqMaxActive))
		return false
	}
	if g.fqActiveFor(f.Key) != nil {
		g.Msg("{{y|You already have work from them.}}")
		return false
	}
	st.Next++
	c.ID = st.Next
	c.State, c.Taken = "active", g.Turn
	delete(st.Offers, f.Key)
	delete(st.Courted, f.Key)
	st.Active = append(st.Active, c)
	c.Cur = -1
	g.fqLog(c, "Took: "+c.Title+".")
	g.Msg("{{W|Faction work taken:}} " + c.Title + ".")
	g.fqAdvance(c)
	if len(st.Active) == 1 {
		g.fqTrack(c)
	}
	return true
}

// fqAdvance moves to the next step whose condition holds, or finishes the chain.
func (g *Game) fqAdvance(c *fqChain) {
	for i := c.Cur + 1; i < len(c.Steps); i++ {
		s := c.Steps[i]
		if fqNeedOK(c, s.Need) {
			c.Cur = i
			g.fqStart(c, s)
			return
		}
		s.State = "skip"
	}
	g.fqFinish(c)
}

func (g *Game) fqStart(c *fqChain, s *fqStep) {
	s.State, s.Start = "act", g.Turn
	switch s.Kind {
	case "deliver":
		it := &Item{ID: g.newID(), Name: fqCap(s.Item), Kind: "trophy", Glyph: '=', F: CYellow, Weight: 1, Value: 0, Ident: true,
			Extra: map[string]int{"fq": c.ID, "fqs": c.Cur + 1}, Lore: fmt.Sprintf("For %s in %s. Entrusted to you by %s. Not yours to open.", s.Person, g.fqBurgName(s.Burg), c.Rep)}
		g.give(it)
		g.Msg("You take " + s.Item + ".")
		s.Due = g.Turn + s.Days*DayLen
	case "hold":
		s.Until = g.Turn + s.Days*DayLen
	case "avenge":
		s.Due = g.Turn + 6*DayLen
	case "meet", "kill", "fetch", "rescue":
		if s.Days > 0 {
			s.Due = g.Turn + s.Days*DayLen
		}
	}
	g.fqMirror(c)
	at, n := g.fqStepCount(c)
	g.Msg(fmt.Sprintf("{{W|%s (%d/%d):}} %s", c.Title, at, n, s.brief(c)))
	if s.Due > 0 && s.Kind != "avenge" {
		g.Msg("{{K|Time allowed: " + jbClock(s.Due-g.Turn) + ".}}")
	}
	if g.Cur != nil {
		g.fqUntrackIfNotFirst(c)
	}
	if s.Kind == "parley" {
		g.fqReady(c, s)
	}
}

func (g *Game) fqUntrackIfNotFirst(c *fqChain) {
	st := g.fq()
	if len(st.Active) > 0 && st.Active[0] == c {
		g.fqTrack(c)
	}
}

func (g *Game) fqReady(c *fqChain, s *fqStep) {
	if s.State == "ready" {
		return
	}
	s.State = "ready"
	if s.Kind != "parley" {
		g.Msg(fmt.Sprintf("{{G|Step done:}} %s Report to %s in %s.", s.Done, c.Rep, g.fqBurgName(c.Burg)))
	} else {
		g.Msg(fmt.Sprintf("{{c|%s waits in %s with your answer to give.}}", c.Rep, g.fqBurgName(c.Burg)))
	}
	g.fqMirror(c)
	g.fqUntrackIfNotFirst(c)
}

func (g *Game) fqStepGold(c *fqChain, s *fqStep) int {
	lvl := maxi(c.Lvl, s.Lvl/2)
	base := 25 + 9*lvl
	if s.Kind == "parley" || s.Kind == "hold" {
		base /= 2
	}
	if c.Courted {
		base = base * 12 / 10
	}
	return base
}

// fqStepFinish marks the step done, pays a little and moves on. choice < 0: no choice was involved.
func (g *Game) fqStepFinish(c *fqChain, s *fqStep, choice int) {
	if s.State == "done" || c.State != "active" {
		return
	}
	s.State = "done"
	f := g.fqFac(c.Fac)
	if s.Kind == "deliver" || s.Kind == "fetch" {
		if it := g.fqInv(c.ID, c.Cur); it != nil {
			g.removeInv(it)
		}
	}
	if s.Kind == "rescue" {
		if a := g.fqActorAnywhere(s.Actor); a != nil && a.HP > 0 && isFollower(a) {
			g.fqReleaseCaptive(c, a)
		}
	}
	{
		base := g.fqStepGold(c, s)
		got := g.cuEarn(base)
		xp := 8 + c.Lvl*4
		g.gainXP(xp)
		g.Msg(fmt.Sprintf("{{G|%s}} You are paid {{Y|%s}} (+%d xp).", s.Done, got, xp))
	}
	name := c.Fac
	if f != nil {
		name = f.Name
	}
	g.fqLog(c, "Done: "+s.Done)
	g.fqNews(c.Burg, "%s did a task for %s in %s.", g.P.Name, name, g.fqBurgName(c.Burg))
	g.fqSink(f, fmt.Sprintf("%s has been doing errands for %s.", g.P.Name, name))
	g.fqUntrack(c)
	g.fqAdvance(c)
}

func (g *Game) fqReleaseCaptive(c *fqChain, a *Actor) {
	a.AI, a.Order, a.Comp = "town", "", nil
	if bd := g.W.burgByID[c.Burg]; bd != nil && bd.State > 0 {
		a.Faction = stateFaction(bd.State)
	}
	a.F = CGray
}

func (g *Game) fqCleanup(c *fqChain) {
	g.fqDropItems(c.ID)
	for _, s := range c.Steps {
		if s.Kind == "rescue" && s.Actor != 0 {
			if a := g.fqActorAnywhere(s.Actor); a != nil && a.HP > 0 && isFollower(a) {
				g.fqReleaseCaptive(c, a)
			}
		}
	}
	g.fqUntrack(c)
	g.fqCloseMirror(c)
	st := g.fq()
	for i, o := range st.Active {
		if o == c {
			st.Active = append(st.Active[:i], st.Active[i+1:]...)
			break
		}
	}
}

func (g *Game) fqFail(c *fqChain, why string) {
	if c.State != "active" {
		return
	}
	c.State = "failed"
	st := g.fq()
	st.FailN++
	f := g.fqFac(c.Fac)
	g.fqCleanup(c)
	st.Cool[c.Fac] = g.Day() + 3
	g.Msg("{{R|Faction work failed:}} " + c.Title + ". " + why)
	g.fqLog(c, "Failed: "+c.Title+".")
	g.fqMove(c.Fac, -4, " for work left undone")
	if f != nil {
		g.fqNews(c.Burg, "%s took work from %s and left it undone.", g.P.Name, f.Name)
	}
}

func (g *Game) fqAbandon(c *fqChain) {
	if c.State != "active" {
		return
	}
	c.State = "abandoned"
	st := g.fq()
	g.fqCleanup(c)
	st.Cool[c.Fac] = g.Day() + 2
	g.Msg("You give up: " + c.Title + ".")
	g.fqLog(c, "Abandoned: "+c.Title+".")
	g.fqMove(c.Fac, -2, " for dropping the work")
}

func (g *Game) fqFinish(c *fqChain) {
	if c.State != "active" {
		return
	}
	c.State = "done"
	st := g.fq()
	f := g.fqFac(c.Fac)
	g.fqCleanup(c)
	st.Done[c.Fac] = append(st.Done[c.Fac], c.Tpl)
	st.DoneN++
	st.Cool[c.Fac] = g.Day() + 1
	if f == nil {
		return
	}
	rank := mini(4, st.Rank[c.Fac]+1)
	st.Rank[c.Fac] = rank
	st.Stand[c.Fac] = clampi(st.Stand[c.Fac]+8, -60, 300)
	base := 120 + c.Lvl*30
	if c.Courted {
		base = base * 12 / 10
	}
	got := g.cuEarn(base)
	xp := 40 + c.Lvl*12
	g.gainXP(xp)
	title := g.fqRankTitle(f, rank)
	g.Msg(fmt.Sprintf("{{G|Chain complete:}} %s. %s pays {{Y|%s}} (+%d xp). You are now {{W|%s}}.", c.Title, c.Rep, got, xp, title))
	switch f.Kind {
	case "realm", "faith":
		g.AddRep(f.Key, 6+2*rank, " for loyal service")
		if f.Kind == "faith" {
			g.P.HP = g.P.MaxHP
			g.Msg("{{G|A blessing mends you.}}")
		}
	case "watch":
		g.AddRep(stateFaction(f.State), 4, " for service to the watch")
		if es := g.war(); es.Realm == f.State {
			g.warAddMerit(10 + 5*rank)
		}
	case "house":
		g.AddRep(stateFaction(f.State), 2, " from the house's friends")
	case "uw":
		g.AddRep("outlaws", 15, " in the underworld")
	case "spy":
		n := 0
		for i := range g.W.Burgs {
			b := &g.W.Burgs[i]
			if b.Alive && b.State == f.State && b.P >= 0 && b.P < g.W.W*g.W.H && n < 5 {
				g.Known[b.P] = true
				n++
			}
		}
		if n > 0 {
			g.Msg(fmt.Sprintf("{{G|Your handler marks %d towns on your map.}}", n))
		}
	}
	day := g.Day()
	if c.Rival != "" {
		rn := g.fqName(c.Rival)
		if c.RivNet <= -6 {
			st.Hostile[c.Rival] = day + 10
			g.Msg("{{R|" + fqCap(rn) + " will not forget which side you took.}}")
		} else if c.RivNet >= 4 {
			if rf := g.fqFac(c.Rival); rf != nil && st.Rank[c.Rival] < 4 {
				st.Courted[c.Rival] = true
				g.Msg("{{Y|" + fqCap(rn) + " has taken notice of you. A petition may be waiting.}}")
			}
		}
	}
	g.deed("quest", fmt.Sprintf("%s finished %s for %s and was named %s.", g.P.Name, c.Title, f.Name, title), 0, 0)
	g.fqLog(c, "Finished: "+c.Title+".")
	g.fqNews(c.Burg, "%s was named %s.", g.P.Name, title)
	g.fqSink(f, fmt.Sprintf("%s was named %s.", g.P.Name, title))
	if st.Hostile[c.Fac] > day {
		delete(st.Hostile, c.Fac)
	}
}

// ---------------------------------------------------------------- choices

func (g *Game) fqSay(text string) {
	g.Popup = Wrap(text, 100)
	g.Opts = []Option{{"Go on.", func() { g.tpBackToTalk() }}}
	g.Sel = 0
	delete(th2Back, g)
}

func (g *Game) fqChoose(c *fqChain, s *fqStep) {
	var br []tpBranch
	for i, op := range s.Ch.Opts {
		i, op := i, op
		b := tpBranch{Text: op.Text, Win: op.Win, Lose: op.Lose, Cost: op.Cost,
			OnWin:  func() { g.fqApplyChoice(c, s, i, true) },
			OnLose: func() { g.fqApplyChoice(c, s, i, false) }}
		if b.Lose == "" {
			b.Lose = "It does not go as you hoped."
		}
		if op.Chk != "" {
			b.Check, b.DC = op.Chk, op.DC
		}
		if b.Win == "" {
			b.Win = "It is done."
		}
		br = append(br, b)
	}
	g.tpChoice(s.Ch.Prompt, br)
}

func (g *Game) fqApplyChoice(c *fqChain, s *fqStep, i int, win bool) {
	if i < 0 || i >= len(s.Ch.Opts) || s.State == "done" || c.State != "active" {
		return
	}
	op := s.Ch.Opts[i]
	st := g.fq()
	if win {
		if op.Flag != "" {
			c.setFlag(op.Flag)
		}
		g.fqMove(c.Fac, op.Fac, " for how you handled it")
		if c.Rival != "" && op.Riv != 0 {
			g.fqMove(c.Rival, op.Riv, " for how you handled it")
		}
		c.RivNet += op.Riv
		if op.Gold > 0 {
			g.Msg(fmt.Sprintf("You gain {{Y|%s}}.", g.cuEarn(op.Gold)))
		}
		if op.Hostile {
			st.Hostile[c.Fac] = g.Day() + 12
		}
	} else {
		g.fqMove(c.Fac, -2, " for the failed attempt")
	}
	g.fqLog(c, "Chose: "+op.Text)
	g.fqStepFinish(c, s, i)
}

// fqTurnIn is the player's report to the representative.
func (g *Game) fqTurnIn(c *fqChain, s *fqStep) {
	if c.State != "active" || s.State != "ready" {
		return
	}
	switch s.Kind {
	case "fetch":
		if g.fqInv(c.ID, c.Cur) == nil {
			g.fqSay("\"You do not have it.\"")
			return
		}
	}
	if s.Ch != nil {
		g.fqChoose(c, s)
		return
	}
	g.fqStepFinish(c, s, -1)
	g.fqSay(s.Done)
}

// ---------------------------------------------------------------- talk

func fqTalk(g *Game, a *Actor) []Option {
	if g.W == nil || g.Cur == nil || g.P == nil || a == nil || a.HP <= 0 || a.Angry || a.AI == "hostile" {
		return nil
	}
	st := g.fq()
	var out []Option
	if a.Has("fqcap") {
		for _, c := range st.Active {
			c := c
			s := g.fqStepOf(c)
			if s != nil && s.Kind == "rescue" && s.State == "act" && s.Actor == a.ID {
				out = append(out, Option{"Cut " + a.Name + " loose (petition)", func() {
					a.AI, a.Faction, a.Order, a.F = "follower", "player", OrdFollow, CGreen
					g.comp(a)
					a.Flags["fqcap"] = false
					a.Set("fqfree")
					a.Title = "freed captive"
					s.Freed = true
					g.fqMirror(c)
					g.Msg(fmt.Sprintf("{{G|%s is free.}} \"Take me to %s in %s, please.\"", a.Name, c.Rep, g.fqBurgName(c.Burg)))
					g.Mode = "main"
				}})
			}
		}
	}
	if a.Has("fqct") {
		for _, c := range st.Active {
			c := c
			s := g.fqStepOf(c)
			if s == nil || s.State != "act" || s.Actor != a.ID || g.Cur.Burg != s.Burg {
				continue
			}
			switch s.Kind {
			case "deliver":
				if g.fqInv(c.ID, c.Cur) != nil {
					out = append(out, Option{"Hand over " + s.Item + " to " + a.Name + " (petition)", func() { g.fqContact(c, s, a) }})
				}
			case "meet":
				out = append(out, Option{"I come on behalf of " + c.Rep + " (petition)", func() { g.fqContact(c, s, a) }})
			}
		}
	}
	if a.Has("fqrep") {
		f, rp := g.fqRepOfActor(a)
		if f == nil || rp == nil {
			return out
		}
		if c := g.fqActiveFor(f.Key); c != nil {
			s := g.fqStepOf(c)
			if s != nil && s.State == "ready" {
				out = append(out, Option{"Report on the petition: " + c.Title, func() { g.fqTurnIn(c, s) }})
			} else if s != nil {
				out = append(out, Option{"Remind me of the petition: " + c.Title, func() {
					text := "{{W|" + c.Title + "}}\n\n" + s.brief(c)
					if s.Due > 0 && s.Kind != "avenge" && s.Kind != "hold" {
						text += "\n\nTime left: " + jbClock(s.Due-g.Turn) + "."
					}
					g.fqSay(text)
				}})
			}
			return out
		}
		out = append(out, Option{"Ask for a petition from " + f.Name, func() { g.fqShowOffer(f, rp) }})
	}
	return out
}

func (g *Game) fqShowOffer(f *fqFac, rp *fqRep) {
	st := g.fq()
	day := g.Day()
	if st.Hostile[f.Key] > day {
		g.fqSay("\"We have nothing for you. Not after what you did.\"")
		return
	}
	if st.Cool[f.Key] > day {
		g.fqSay("\"Not today. Come back in a few days.\"")
		return
	}
	if !g.fqEligible(f) {
		g.fqSay("\"We do not know you well enough yet. Do us some good first.\"")
		return
	}
	c := g.fqOfferFor(f, rp)
	if c == nil {
		g.fqSay("\"Nothing I can trust to you at the moment.\"")
		return
	}
	prompt := "{{W|" + c.Title + "}}\n\n" + c.Hook + "\n\n"
	prompt += fmt.Sprintf("%d steps. Pay about %s at the end, a new rank, and a perk (%s).", g.fqPlannedSteps(c), g.cuFormat(120+c.Lvl*30), fqKinds[f.Kind].Perk)
	if len(st.Active) >= fqMaxActive {
		g.tpChoice(prompt+"\n\n{{y|You carry too many chains to take another.}}", nil)
		return
	}
	g.tpChoice(prompt, []tpBranch{{Text: "I will see it done.", Win: fmt.Sprintf("\"Good. Come back to me when it is finished, or if the ground changes under you.\" (%s)", c.Rep),
		OnWin: func() { g.fqAccept(f, c) }}})
}

func (g *Game) fqPlannedSteps(c *fqChain) int {
	n := 0
	for _, s := range c.Steps {
		if s.Need == "" {
			n++
		}
	}
	return n
}

func (g *Game) fqContact(c *fqChain, s *fqStep, a *Actor) {
	if s.Ch != nil {
		g.fqChoose(c, s)
		a.Flags["fqct"] = false
		return
	}
	a.Flags["fqct"] = false
	g.fqStepFinish(c, s, -1)
	g.fqSay(s.Done)
}

// ---------------------------------------------------------------- world hooks

func fqKill(g *Game, d, src *Actor) {
	if g.W == nil || g.P == nil || d == nil || d == g.P {
		return
	}
	st := g.fq()
	for _, c := range append([]*fqChain(nil), st.Active...) {
		s := g.fqStepOf(c)
		if s == nil || c.State != "active" || s.Actor != d.ID || d.ID == 0 {
			continue
		}
		switch s.Kind {
		case "kill":
			if s.State == "act" {
				g.fqReady(c, s)
			}
		case "rescue":
			if s.State == "act" {
				g.fqFail(c, d.Name+" is dead.")
			}
		case "avenge":
			if s.State == "act" {
				g.fqStepFinish(c, s, -1)
			}
		}
	}
}

func fqEnter(g *Game, zn *Zone) {
	if g.W == nil || g.P == nil || zn == nil {
		return
	}
	g.fqEnsureReps(zn)
	st := g.fq()
	day := g.Day()
	for _, c := range append([]*fqChain(nil), st.Active...) {
		s := g.fqStepOf(c)
		if s == nil || c.State != "active" {
			continue
		}
		idx := c.Cur
		switch s.Kind {
		case "scout":
			if s.State == "act" && fqWildZone(zn) && zn.WX == s.WX && zn.WY == s.WY {
				g.fqStepFinish(c, s, -1)
			}
		case "kill":
			if s.State == "act" && !s.Placed && fqWildZone(zn) && zn.WX == s.WX && zn.WY == s.WY {
				g.fqPlaceFoe(c, s, zn, idx, false)
			}
		case "avenge":
			if s.State == "act" && !s.Placed && fqWildZone(zn) {
				g.fqPlaceFoe(c, s, zn, idx, true)
			}
		case "fetch":
			if s.State == "act" && !s.Placed && zn.Key == s.TZ {
				g.fqPlaceItem(c, s, zn, idx)
			}
		case "rescue":
			if s.State == "act" && !s.Placed && zn.Key == s.TZ {
				g.fqPlaceCaptive(c, s, zn, idx)
			}
			if s.State == "act" && s.Freed && fqIsTown(zn) && zn.Burg == c.Burg {
				for _, a := range zn.Actors {
					if a.ID == s.Actor && isFollower(a) {
						g.Msg(fmt.Sprintf("{{G|%s has reached %s.}}", a.Name, g.fqBurgName(c.Burg)))
						g.fqReleaseCaptive(c, a)
						g.fqReady(c, s)
						break
					}
				}
			}
		case "meet", "deliver":
			if s.State == "act" && !s.Placed && fqIsTown(zn) && zn.Burg == s.Burg {
				g.fqPlaceContact(c, s, zn, idx)
			}
		case "hold":
			if s.State == "act" && s.Amb && !s.AmbDo && g.Turn-s.Start > 180 && fqWildZone(zn) && g.fqChance(zn, "hold", 35) {
				s.AmbDo = true
				g.fqAmbush(zn, g.fqName(c.Rival), maxi(2, g.P.Level-1), 3, "The agents of")
			}
		}
	}
	// a faction you wronged sends agents
	if fqWildZone(zn) && day >= st.NextAmb && g.P.Level >= 3 {
		var keys []string
		for k, until := range st.Hostile {
			if until > day {
				keys = append(keys, k)
			}
		}
		sort.Strings(keys)
		if len(keys) > 0 && g.fqChance(zn, "hostile", 12) {
			k := keys[int(HashStr(fmt.Sprint(g.Seed, ":fq:hk:", day))%uint64(len(keys)))]
			if g.fqAmbush(zn, g.fqName(k), clampi(g.P.Level-1, 2, 25), 3, "The agents of") {
				st.NextAmb = day + 6
			}
		}
	}
}

// fqAmbush sends a small hostile squad into zn; true when it did.
func (g *Game) fqAmbush(zn *Zone, from string, lvl, n int, lead string) bool {
	if from == "its rivals" {
		from = "hidden enemies"
	}
	r := NewRng(fmt.Sprint(g.Seed, ":fq:amb:", zn.Key, ":", g.Day()))
	lx, ly, ok := zn.FreeSpot(r, func(x, y int) bool { return abs(x-g.P.X) >= 8 || abs(y-g.P.Y) >= 6 })
	if !ok {
		return false
	}
	for i := 0; i < n; i++ {
		tpl := "bandit"
		if i == n-1 {
			tpl = "archer"
		}
		a := g.Spawn(tpl, lvl+boolInt(i == 0), r)
		a.Name = "agent of " + from
		a.Faction, a.AI, a.Angry = "outlaws", "hostile", true
		a.Desc = fqCap(lead) + " " + from + " have a score to settle with you."
		a.Set("fqx")
		a.X, a.Y = lx+r.Int(-2, 2), ly+r.Int(-2, 2)
		g.nudge(zn, a)
		zn.Actors = append(zn.Actors, a)
	}
	g.Msg("{{R|" + fqCap(lead) + " " + from + " step out of the land and into your road.}}")
	return true
}

func (g *Game) fqPlaceContact(c *fqChain, s *fqStep, zn *Zone, idx int) {
	bd := g.W.burgByID[s.Burg]
	if bd == nil {
		return
	}
	r := NewRng(fmt.Sprint(g.Seed, ":fq:ct:", c.Fac, ":", c.Tpl, ":", c.Burg, ":", idx))
	a := g.makeNPC("villager", bd, g.W.cultByID[bd.Culture], r)
	a.Name, a.Proper, a.Title = s.Person, true, s.CTitle
	a.Desc = fmt.Sprintf("%s, %s of %s. They are waiting for someone.", s.Person, s.CTitle, bd.Name)
	a.AI = "still"
	a.Set("fqct")
	x, y, ok := zn.FreeSpot(r, nil)
	if !ok {
		return
	}
	a.X, a.Y, a.HomeX, a.HomeY = x, y, x, y
	zn.Actors = append(zn.Actors, a)
	s.Placed, s.Actor = true, a.ID
	g.Msg(fmt.Sprintf("{{Y|%s, %s, is somewhere in %s.}}", s.Person, s.CTitle, bd.Name))
}

func (g *Game) fqFarSpot(zn *Zone, r *Rng) (int, int, bool) {
	if x, y, ok := zn.FreeSpot(r, func(x, y int) bool { return abs(x-g.P.X) > 8 || abs(y-g.P.Y) > 5 }); ok {
		return x, y, true
	}
	return zn.FreeSpot(r, nil)
}

func (g *Game) fqPlaceFoe(c *fqChain, s *fqStep, zn *Zone, idx int, avenger bool) {
	r := NewRng(fmt.Sprint(g.Seed, ":fq:foe:", c.Fac, ":", c.Tpl, ":", c.Burg, ":", idx))
	lvl := maxi(2, s.Lvl)
	f := g.Spawn("bandit", lvl, r)
	f.Name, f.Proper = s.Foe, true
	f.MaxHP = f.MaxHP * 3 / 2
	f.HP = f.MaxHP
	f.AI = "hostile"
	f.Set("fqfoe")
	x, y, ok := g.fqFarSpot(zn, r)
	if !ok {
		return
	}
	f.X, f.Y, f.HomeX, f.HomeY = x, y, x, y
	zn.Actors = append(zn.Actors, f)
	for i := 0; i < 2; i++ {
		h := g.Spawn(pickCreature(r, zn.Biome, 's', lvl), maxi(1, lvl-1), r)
		h.AI = "hostile"
		h.X, h.Y = x+r.Int(-3, 3), y+r.Int(-3, 3)
		g.nudge(zn, h)
		zn.Actors = append(zn.Actors, h)
	}
	s.Placed, s.Actor = true, f.ID
	if avenger {
		g.Msg("{{R|" + s.Foe + " steps out of the land and into your road.}}")
	} else {
		g.Msg("{{Y|Fresh tracks and cold ashes: " + s.Foe + " is somewhere in this land.}}")
	}
}

func (g *Game) fqPlaceItem(c *fqChain, s *fqStep, zn *Zone, idx int) {
	r := NewRng(fmt.Sprint(g.Seed, ":fq:item:", c.Fac, ":", c.Tpl, ":", c.Burg, ":", idx))
	x, y, ok := g.fqFarSpot(zn, r)
	if !ok {
		return
	}
	it := &Item{ID: g.newID(), Name: fqCap(s.Item), Kind: "trophy", Glyph: '♦', F: CYellow, Weight: 1, Value: 0, Ident: true, X: x, Y: y,
		Extra: map[string]int{"fq": c.ID, "fqs": idx + 1}, Lore: "Wanted by " + c.Rep + ". Not yours to keep."}
	zn.Items = append(zn.Items, it)
	s.Placed = true
	g.Msg("{{Y|Somewhere in here lies " + s.Item + ".}}")
}

func (g *Game) fqPlaceCaptive(c *fqChain, s *fqStep, zn *Zone, idx int) {
	bd := g.W.burgByID[c.Burg]
	if bd == nil {
		return
	}
	r := NewRng(fmt.Sprint(g.Seed, ":fq:cap:", c.Fac, ":", c.Tpl, ":", c.Burg, ":", idx))
	a := g.makeNPC("villager", bd, g.W.cultByID[bd.Culture], r)
	a.Name, a.Proper, a.Title = s.Person, true, "captive"
	a.Desc = s.Person + ", bound hand and foot."
	a.AI = "still"
	a.HP = maxi(1, a.MaxHP*6/10)
	a.Set("fqcap")
	x, y, ok := g.fqFarSpot(zn, r)
	if !ok {
		return
	}
	a.X, a.Y, a.HomeX, a.HomeY = x, y, x, y
	zn.Actors = append(zn.Actors, a)
	lvl := maxi(2, s.Lvl)
	for i := 0; i < 2+lvl/6; i++ {
		m := g.Spawn(pickCreature(r, zn.Biome, 's', lvl), lvl, r)
		mx, my, ok := zn.FreeSpot(r, func(px, py int) bool { return abs(px-x) <= 4 && abs(py-y) <= 3 })
		if !ok {
			mx, my, ok = zn.FreeSpot(r, nil)
		}
		if !ok {
			break
		}
		m.X, m.Y = mx, my
		zn.Actors = append(zn.Actors, m)
	}
	s.Placed, s.Actor = true, a.ID
	g.Msg("{{Y|Someone is bound in the dark of this place. You hear them breathing.}}")
}

func fqTurn(g *Game) {
	if g.W == nil || g.P == nil || g.Cur == nil {
		return
	}
	st := g.fq()
	if len(st.Active) == 0 {
		return
	}
	for _, c := range append([]*fqChain(nil), st.Active...) {
		s := g.fqStepOf(c)
		if s == nil || c.State != "active" {
			continue
		}
		switch s.Kind {
		case "hold":
			if s.State == "act" && g.Turn >= s.Until {
				g.fqReady(c, s)
			}
		case "avenge":
			if s.State == "act" && s.Due > 0 && g.Turn > s.Due {
				s.State = "skip"
				g.Msg("{{K|The one who wanted you has lost your trail.}}")
				g.fqAdvance(c)
			}
		default:
			if s.Due > 0 && g.Turn > s.Due && (s.State == "act" || s.State == "ready") {
				g.fqFail(c, "The time ran out.")
			}
		}
	}
	h := g.Turn / 60
	if h == st.Hour {
		return
	}
	st.Hour = h
	for _, c := range append([]*fqChain(nil), st.Active...) {
		s := g.fqStepOf(c)
		if s == nil || c.State != "active" {
			continue
		}
		switch s.Kind {
		case "fetch":
			has := g.fqInv(c.ID, c.Cur) != nil
			if s.State == "act" && has {
				g.fqReady(c, s)
			} else if s.State == "ready" && !has {
				g.fqFail(c, "You lost what you were sent for.")
			}
		case "deliver":
			if s.State == "act" && g.fqInv(c.ID, c.Cur) == nil {
				g.fqFail(c, "You no longer carry what you were given.")
			}
		case "kill":
			if s.State == "act" && s.Placed && g.fqActorAnywhere(s.Actor) == nil {
				g.Msg("{{K|" + s.Foe + " is gone from the land; someone else has settled it.}}")
				g.fqReady(c, s)
			}
		case "rescue":
			if s.State == "act" && s.Placed && g.fqActorAnywhere(s.Actor) == nil {
				g.fqFail(c, "You lost "+s.Person+" along the way.")
			}
		}
	}
}

func fqSide(g *Game) []string {
	if g.W == nil || g.P == nil {
		return nil
	}
	st := g.fq()
	if len(st.Active) == 0 {
		return nil
	}
	soon := -1
	for _, c := range st.Active {
		if s := g.fqStepOf(c); s != nil && s.Due > 0 && s.Kind != "avenge" && s.State != "done" {
			if left := s.Due - g.Turn; soon < 0 || left < soon {
				soon = left
			}
		}
	}
	line := fmt.Sprintf("Chains: {{W|%d}}", len(st.Active))
	if soon >= 0 {
		col := "G"
		if soon < DayLen {
			col = "R"
		} else if soon < 3*DayLen {
			col = "Y"
		}
		line += fmt.Sprintf("  next {{%s|%s}}", col, jbClock(soon))
	}
	return []string{line}
}

// ---------------------------------------------------------------- screen

func (g *Game) fqOpen() {
	st := g.fq()
	st.tab, st.sel, st.conf = 0, 0, ""
	if len(st.Active) == 0 {
		st.tab = 1
	}
	g.Mode = "fq"
}

// fqRows lists the factions worth showing: met, active, or close to a representative's notice.
func (g *Game) fqRows() []*fqFac {
	st := g.fq()
	type row struct {
		f *fqFac
		s int
	}
	var rs []row
	for _, f := range g.fqFacs() {
		sc := g.fqStanding(f)
		_, hasRep := st.Reps[f.Key]
		if hasRep || g.fqActiveFor(f.Key) != nil || st.Rank[f.Key] > 0 || st.Courted[f.Key] || sc >= f.Need/2 || st.Hostile[f.Key] > g.Day() {
			rs = append(rs, row{f, sc})
		}
	}
	sort.SliceStable(rs, func(i, j int) bool { return rs[i].s > rs[j].s })
	var out []*fqFac
	for _, r := range rs {
		out = append(out, r.f)
	}
	return out
}

func fqKeys(g *Game, k string) {
	st := g.fq()
	nrows := len(st.Active)
	if st.tab == 1 {
		nrows = len(g.fqRows())
	}
	switch k {
	case "Escape", "q":
		g.Mode = "main"
		return
	case "Tab", "ArrowRight", "ArrowLeft":
		st.tab = 1 - st.tab
		st.sel, st.conf = 0, ""
	case "ArrowUp", "N8", "k":
		st.sel = maxi(0, st.sel-1)
		st.conf = ""
	case "ArrowDown", "N2", "j":
		st.sel = mini(maxi(0, nrows-1), st.sel+1)
		st.conf = ""
	case "t":
		if st.tab == 0 && st.sel < len(st.Active) {
			g.fqTrack(st.Active[st.sel])
		}
	case "x":
		if st.tab == 0 && st.sel < len(st.Active) {
			c := st.Active[st.sel]
			if st.conf != c.Title {
				st.conf = c.Title
				g.Msg("{{y|Press x again to give up: " + c.Title + ".}}")
				return
			}
			st.conf = ""
			g.fqAbandon(c)
			st.sel = maxi(0, st.sel-1)
		}
	}
	if st.tab == 0 && st.sel >= len(st.Active) {
		st.sel = maxi(0, len(st.Active)-1)
	}
}

func (g *Game) fqWhere(burg int) string {
	x, y, ok := g.fqBurgXY(burg)
	if !ok || g.Cur == nil {
		return g.fqBurgName(burg)
	}
	dx, dy := x-g.Cur.WX, y-g.Cur.WY
	if dx == 0 && dy == 0 {
		return g.fqBurgName(burg) + " (here)"
	}
	return fmt.Sprintf("%s ({{K|%d parasangs %s}})", g.fqBurgName(burg), jbPar(dx, dy), jbDirName(dx, dy))
}

func (g *Game) fqChainDetail(c *fqChain) []string {
	f := g.fqFac(c.Fac)
	fname := g.fqName(c.Fac)
	out := []string{"{{W|" + c.Title + "}}", "For " + fname + ", through " + c.Rep + " (" + c.RepT + ") in " + g.fqWhere(c.Burg) + "."}
	for i, s := range c.Steps {
		mark := "  "
		switch {
		case s.State == "done":
			mark = "{{G|x}} "
		case s.State == "skip" || (s.State == "" && i > c.Cur && !fqNeedOK(c, s.Need)):
			if s.Need != "" {
				continue
			}
			mark = "- "
		case i == c.Cur:
			mark = "{{Y|>}} "
		}
		line := mark + s.brief(c)
		if s.State == "done" {
			line = mark + "{{K|" + s.Done + "}}"
		} else if s.State == "" && i > c.Cur && s.Need != "" {
			line = mark + "{{K|(a step that depends on your choices)}}"
		}
		out = append(out, line)
	}
	if s := g.fqStepOf(c); s != nil {
		switch s.State {
		case "ready":
			out = append(out, "{{G|Ready: report to "+c.Rep+" in "+g.fqBurgName(c.Burg)+".}}")
		case "act":
			if s.Due > 0 && s.Kind != "avenge" {
				out = append(out, "{{R|Time left: "+jbClock(s.Due-g.Turn)+"}}")
			}
			if s.Kind == "hold" {
				out = append(out, "{{K|Until the wait ends: "+jbClock(s.Until-g.Turn)+"}}")
			}
		}
	}
	if f != nil {
		rv := ""
		if c.Rival != "" {
			rv = fmt.Sprintf(" Rival: %s (%d).", g.fqName(c.Rival), g.Rep[c.Rival])
		}
		out = append(out, fmt.Sprintf("Standing with them: %d (need %d).%s", g.fqStanding(f), f.Need, rv))
		next := mini(4, g.fq().Rank[c.Fac]+1)
		out = append(out, fmt.Sprintf("On finishing: about %s, the rank {{W|%s}}. Perk: %s.", g.cuFormat(120+c.Lvl*30), g.fqRankTitle(f, next), fqKinds[f.Kind].Perk))
	}
	if len(c.Log) > 0 {
		out = append(out, "{{K|"+c.Log[len(c.Log)-1]+"}}")
	}
	return out
}

func (g *Game) fqFacDetail(f *fqFac) []string {
	st := g.fq()
	ki := fqKinds[f.Kind]
	sc := g.fqStanding(f)
	out := []string{"{{W|" + fqCap(f.Name) + "}} {{K|(" + ki.Label + ")}}"}
	col := "R"
	if sc >= f.Need {
		col = "G"
	}
	out = append(out, fmt.Sprintf("Standing {{%s|%d}} of %d needed for a petition.", col, sc, f.Need))
	rk := st.Rank[f.Key]
	if rk > 0 {
		out = append(out, "Your rank: {{W|"+g.fqRankTitle(f, rk)+"}}.")
	} else {
		out = append(out, "No rank yet.")
	}
	if rk < 4 {
		out = append(out, fmt.Sprintf("Next reward: {{W|%s}} and %s.", g.fqRankTitle(f, rk+1), ki.Perk))
	} else {
		out = append(out, "You hold their highest rank.")
	}
	if rp := st.Reps[f.Key]; rp != nil {
		out = append(out, rp.Name+", "+rp.Title+", in "+g.fqWhere(rp.Burg)+".")
	} else if sc >= f.Need || st.Courted[f.Key] {
		out = append(out, "Their representative will find you in a town they favour.")
	} else {
		out = append(out, "Earn more standing and a representative will seek you out.")
	}
	if c := st.Offers[f.Key]; c != nil {
		out = append(out, "Offer waiting: {{W|"+c.Title+"}}.")
	}
	if st.Courted[f.Key] {
		out = append(out, "{{Y|They court you now.}}")
	}
	if until := st.Hostile[f.Key]; until > g.Day() {
		out = append(out, fmt.Sprintf("{{R|They are hostile for %d more days.}}", until-g.Day()))
	}
	if n := len(st.Done[f.Key]); n > 0 {
		out = append(out, fmt.Sprintf("Chains finished: %d.", n))
	}
	if f.Rival != "" {
		out = append(out, "Rival: "+g.fqName(f.Rival)+".")
	}
	return out
}

func (g *Game) fqDraw() {
	sc := g.sc
	st := g.fq()
	x, y, w, h := g.panel("Faction work")
	bottom := 2 + 1 + h - 1
	tabs := fmt.Sprintf("{{%s|[ Chains %d/%d ]}}  {{%s|[ Factions ]}}", map[bool]string{true: "W", false: "K"}[st.tab == 0], len(st.Active), fqMaxActive, map[bool]string{true: "W", false: "K"}[st.tab == 1])
	sc.TextB(x, y, tabs, CGray, CBgDark, x+w)
	y++
	if st.tab == 0 {
		sc.TextB(x, y, "{{K|Up/Down choose  t track  x give up (twice)  Tab factions  Esc close}}", CGray, CBgDark, x+w)
	} else {
		sc.TextB(x, y, "{{K|Up/Down choose  Tab chains  Esc close}}", CGray, CBgDark, x+w)
	}
	y += 2
	var detail []string
	if st.tab == 0 {
		if len(st.Active) == 0 {
			sc.TextB(x, y, "No faction chains taken. Earn standing with a crown, faith, house, the watch, the underworld or a spy network;", CGray, CBgDark, x+w)
			sc.TextB(x, y+1, "their representative will find you. Standing is shown on the Factions tab.", CGray, CBgDark, x+w)
			y += 3
		}
		for i, c := range st.Active {
			mark := "  "
			if i == st.sel {
				mark = "> "
			}
			at, n := g.fqStepCount(c)
			tail := "{{K|step " + fmt.Sprint(at, "/", n) + "}}"
			if s := g.fqStepOf(c); s != nil && s.State == "ready" {
				tail = "{{G|ready to report}}"
			}
			sc.TextB(x, y, fmt.Sprintf("%s%s  %s", mark, c.Title, tail), CGray, CBgDark, x+w)
			y++
		}
		if len(st.Active) > 0 {
			detail = g.fqChainDetail(st.Active[clampi(st.sel, 0, len(st.Active)-1)])
		}
	} else {
		rows := g.fqRows()
		if len(rows) == 0 {
			sc.TextB(x, y, "No faction has taken notice of you yet. Do good work for a realm, a faith or a house.", CGray, CBgDark, x+w)
			y += 2
		}
		if st.sel >= len(rows) {
			st.sel = maxi(0, len(rows)-1)
		}
		lim := 10
		top := 0
		if st.sel >= lim {
			top = st.sel - lim + 1
		}
		for i := top; i < len(rows) && i < top+lim; i++ {
			f := rows[i]
			mark := "  "
			if i == st.sel {
				mark = "> "
			}
			s := g.fqStanding(f)
			c := "R"
			if s >= f.Need {
				c = "G"
			}
			tail := ""
			if rk := st.Rank[f.Key]; rk > 0 {
				tail = " {{W|" + fqKinds[f.Kind].Ranks[rk-1] + "}}"
			}
			if g.fqActiveFor(f.Key) != nil {
				tail += " {{Y|active}}"
			}
			sc.TextB(x, y, fmt.Sprintf("%s%s {{%s|%d/%d}}%s", mark, fqCap(f.Name), c, s, f.Need, tail), CGray, CBgDark, x+w)
			y++
		}
		if len(rows) > 0 {
			detail = g.fqFacDetail(rows[clampi(st.sel, 0, len(rows)-1)])
		}
	}
	y++
	for _, d := range detail {
		for _, ln := range Wrap(d, w) {
			if y < bottom {
				sc.TextB(x, y, ln, CGray, CBgDark, x+w)
				y++
			}
		}
	}
	if len(st.Log) > 0 && y < bottom-2 {
		y++
		sc.TextB(x, y, fmt.Sprintf("{{K|Recent  (finished %d, failed %d)}}", st.DoneN, st.FailN), CGray, CBgDark, x+w)
		y++
		for i := len(st.Log) - 1; i >= 0 && y < bottom; i-- {
			sc.TextB(x, y, "{{K|"+st.Log[i]+"}}", CGray, CBgDark, x+w)
			y++
		}
	}
}
