package engine

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// Family & Fame (prefix fm): courtship and marriage, children, reputation titles, songs and bards, duels
// and city tournaments. Screen: F7. Global state g.ext("fm") (spouse and children follow you across
// planets); planet-local state g.ext("fml") (what each town has heard, the tournament in progress), "fml"
// is in unLocalKeys. Nothing here draws g.R: side rngs/hashes of seed+":fm:"+key only.

const (
	fmKey  = "fm"
	fmLKey = "fml"
)

type fmSpouse struct {
	ID         int             `json:"id"`
	Name       string          `json:"n"`
	Species    string          `json:"sp,omitempty"`
	Role       string          `json:"ro,omitempty"`
	Caste      string          `json:"ca,omitempty"`
	Raw        json.RawMessage `json:"raw,omitempty"`
	Attr       [6]int          `json:"at"`
	Level      int             `json:"lv"`
	Love       int             `json:"lo"`
	Wed        int             `json:"wed"`
	HomeKey    string          `json:"hk,omitempty"`
	HomePlanet int             `json:"hp,omitempty"`
	Pay        int             `json:"pay,omitempty"`
	News       int             `json:"nw,omitempty"`
	Quarrel    int             `json:"qu,omitempty"`
	Gold       int             `json:"g,omitempty"`
	Away       bool            `json:"away,omitempty"`
	Gone       string          `json:"gone,omitempty"` // dead left lost
	Female     bool            `json:"f,omitempty"`
}

type fmChild struct {
	ID      int     `json:"id"`
	Name    string  `json:"n"`
	Species string  `json:"sp,omitempty"`
	Born    int     `json:"b"`
	Attr    [6]int  `json:"at"`
	Calling string  `json:"c"`
	Gift    string  `json:"gi"`
	Female  bool    `json:"f,omitempty"`
	Fate    string  `json:"fa,omitempty"` // "" home, comp, away, kidnapped, lost, dead
	Sick    bool    `json:"sk,omitempty"`
	SickDay int     `json:"sd,omitempty"`
	Bond    int     `json:"bo,omitempty"`
	Play    int     `json:"pl,omitempty"`
	Away    int     `json:"aw,omitempty"`
	Frac    float64 `json:"fr,omitempty"`
}

type fmCourt struct {
	ID          int             `json:"id"`
	Name        string          `json:"n"`
	Aff         int             `json:"a"`
	Stage       int             `json:"s"` // 0 acquaintance 1 courting 2 betrothed
	Meets       int             `json:"m"`
	Last        int             `json:"l"` // day+1 of the last time together
	Interest    string          `json:"i"`
	Rival       string          `json:"r,omitempty"`
	RivalStr    int             `json:"rs,omitempty"`
	RivalBeaten bool            `json:"rb,omitempty"`
	Raw         json.RawMessage `json:"raw,omitempty"`
	ZoneKey     string          `json:"z"`
	Burg        int             `json:"bg"`
	Planet      int             `json:"pl,omitempty"`
	Betrothed   int             `json:"be,omitempty"`
	Gifts       int             `json:"gf,omitempty"`
}

type fmDuel struct {
	OppID    int    `json:"o"`
	OppName  string `json:"on"`
	ZoneKey  string `json:"z"`
	Rules    string `json:"r"`
	Stake    string `json:"s"`
	ItemName string `json:"in,omitempty"`
	ItemID   int    `json:"ii,omitempty"`
	Amt      int    `json:"am,omitempty"`
	Hand     int    `json:"h,omitempty"`
	CX       int    `json:"cx"`
	CY       int    `json:"cy"`
	R        int    `json:"rr"`
	OppHP0   int    `json:"oh"`
	PHP0     int    `json:"ph"`
	YieldHP  int    `json:"yh"`
	Start    int    `json:"st"`
	End      int    `json:"en"`
	Wit      int    `json:"w"`
	Sec      int    `json:"sc"`
	Phase    string `json:"p"` // fight watch
	OppFac   string `json:"of"`
	OppAI    string `json:"oa"`
	Tourney  bool   `json:"t,omitempty"`
}

type fmChal struct {
	Name    string `json:"n"`
	Text    string `json:"t"`
	Rules   string `json:"r"`
	Stake   string `json:"s"`
	Amt     int    `json:"a"`
	Expires int    `json:"e"`
	Hand    int    `json:"h,omitempty"`
}

type fmRescue struct {
	Kid       int    `json:"kid"` // child id, 0 = the spouse
	Name      string `json:"n"`
	Kidnapper string `json:"k"`
	Ransom    int    `json:"r"`
	Deadline  int    `json:"d"`
	Planet    int    `json:"p"`
	Placed    bool   `json:"pl,omitempty"`
	ZoneKey   string `json:"z,omitempty"`
}

type fmSong struct {
	Deed  int    `json:"d"`
	Text  string `json:"t"`
	Day   int    `json:"dy"`
	OX    int    `json:"ox"`
	OY    int    `json:"oy"`
	Boost int    `json:"b,omitempty"`
	Sung  int    `json:"s,omitempty"`
}

type fmState struct {
	Spouse   *fmSpouse        `json:"sp,omitempty"`
	Kids     []*fmChild       `json:"kids,omitempty"`
	Courts   map[int]*fmCourt `json:"courts,omitempty"`
	NextBorn int              `json:"nb,omitempty"`
	Titles   map[string]int   `json:"ti,omitempty"`
	Worn     string           `json:"worn,omitempty"`
	Songs    []*fmSong        `json:"songs,omitempty"`
	Scan     int              `json:"scan,omitempty"`
	Fame     map[int]int      `json:"fame,omitempty"`
	DeedCt   map[string]int   `json:"dct,omitempty"`
	Peak     int              `json:"peak,omitempty"`
	Km       float64          `json:"km,omitempty"`
	KmInit   bool             `json:"ki,omitempty"`
	LastWX   int              `json:"lx,omitempty"`
	LastWY   int              `json:"ly,omitempty"`
	LastPl   int              `json:"lpl,omitempty"`
	Big      int              `json:"big,omitempty"`
	Spared   int              `json:"spared,omitempty"`
	Oath     int              `json:"oath,omitempty"`
	Died     int              `json:"died,omitempty"`
	DuelW    int              `json:"dw,omitempty"`
	DuelL    int              `json:"dl,omitempty"`
	Champs   int              `json:"ch,omitempty"`
	Tourneys int              `json:"tn,omitempty"`
	Tips     int              `json:"tips,omitempty"`
	Duel     *fmDuel          `json:"duel,omitempty"`
	Chal     *fmChal          `json:"chal,omitempty"`
	LastChal int              `json:"lc,omitempty"`
	Rescue   *fmRescue        `json:"res,omitempty"`
	LastKid  int              `json:"lk,omitempty"`
	LastDay  int              `json:"day,omitempty"`
	Log      []string         `json:"log,omitempty"`
}

type fmFighter struct {
	Name    string `json:"n"`
	Species string `json:"s,omitempty"`
	Lvl     int    `json:"l"`
}

type fmTourn struct {
	Burg    int         `json:"b"`
	ZoneKey string      `json:"z"`
	Day     int         `json:"d"`
	Round   int         `json:"r"`
	Foes    []fmFighter `json:"f"`
	Wait    int         `json:"w"`
	Tier    int         `json:"t"`
	Name    string      `json:"n"`
	Purse   int         `json:"p"`
}

type fmLocal struct {
	Heard map[string]int  `json:"h,omitempty"`
	Done  map[string]bool `json:"d,omitempty"`
	Tourn *fmTourn        `json:"t,omitempty"`
}

func (g *Game) fm() *fmState {
	s := g.ext(fmKey, func() any { return &fmState{} }).(*fmState)
	if s.Courts == nil {
		s.Courts = map[int]*fmCourt{}
	}
	if s.Titles == nil {
		s.Titles = map[string]int{}
	}
	if s.Fame == nil {
		s.Fame = map[int]int{}
	}
	if s.DeedCt == nil {
		s.DeedCt = map[string]int{}
	}
	return s
}

func (g *Game) fml() *fmLocal {
	s := g.ext(fmLKey, func() any { return &fmLocal{} }).(*fmLocal)
	if s.Heard == nil {
		s.Heard = map[string]int{}
	}
	if s.Done == nil {
		s.Done = map[string]bool{}
	}
	return s
}

func (s *fmState) fameMax() int {
	m := 0
	for _, v := range s.Fame {
		if v > m {
			m = v
		}
	}
	return m
}

func (s *fmState) sungCount() int {
	n := 0
	for _, x := range s.Songs {
		if x.Sung > 0 {
			n++
		}
	}
	return n
}

func init() {
	unLocalKeys = append(unLocalKeys, fmLKey)
	talkHooks = append(talkHooks, fmTalk)
	turnHooks = append(turnHooks, fmTurn)
	zoneHooks = append(zoneHooks, fmZone)
	killHooks = append(killHooks, fmKill)
	enterHooks = append(enterHooks, fmEnter)
	sideHooks = append(sideHooks, fmSide)
	mainKeys["F7"] = func(g *Game) { g.Mode, g.Sel, g.Tab = "fam", 0, 0 }
	modeKeys["fam"] = fmKeys
	modeDraws["fam"] = func(g *Game) { g.fmDraw() }
}

// ---- small helpers ----

func (g *Game) fmH(parts ...any) uint64 { return HashStr(g.Seed + ":fm:" + fmt.Sprint(parts...)) }
func (g *Game) fmRoll(parts ...any) float64 {
	return float64(g.fmH(parts...)%10000) / 10000
}

func (g *Game) fmCoin(n int) string { return g.cuFormat(n) }

func (g *Game) fmRealm() int {
	if bd := g.localBurg(); bd != nil {
		return bd.State
	}
	if g.Cur != nil && g.W.In(g.Cur.WX, g.Cur.WY) {
		return g.W.State[g.W.Idx(g.Cur.WX, g.Cur.WY)]
	}
	return 0
}

func (g *Game) fmFame(realm, n int) {
	if realm <= 0 {
		return
	}
	st := g.fm()
	st.Fame[realm] = clampi(st.Fame[realm]+n, 0, 100)
}

func (g *Game) fmLog(s string) {
	st := g.fm()
	st.Log = append(st.Log, fmt.Sprintf("Y%d d%d: %s", g.W.DateOf(nil, g.Day()).Year, g.Day()%YearDays+1, s))
	if len(st.Log) > 14 {
		st.Log = st.Log[len(st.Log)-14:]
	}
}

func (g *Game) fmFindActor(pred func(a *Actor) bool) (*Actor, *Zone) {
	if g.Cur != nil {
		for _, a := range g.Cur.Actors {
			if pred(a) {
				return a, g.Cur
			}
		}
	}
	for _, z := range g.Zones {
		if z == g.Cur {
			continue
		}
		for _, a := range z.Actors {
			if pred(a) {
				return a, z
			}
		}
	}
	return nil, nil
}

func (g *Game) fmSpouseActor() (*Actor, *Zone) {
	sp := g.fm().Spouse
	if sp == nil {
		return nil, nil
	}
	return g.fmFindActor(func(a *Actor) bool { return a.ID == sp.ID && a.Has("fm_spouse") })
}

func (g *Game) fmKidActor(k *fmChild) (*Actor, *Zone) {
	return g.fmFindActor(func(a *Actor) bool { return a.Has("fm_child") && a.Name == k.Name })
}

func (g *Game) fmHasTag(tag string) bool {
	for _, id := range thSt(g).Traits {
		if t := thTraitByID[id]; t != nil && t.Tag == tag {
			return true
		}
	}
	return false
}

func (g *Game) fmMarried() bool {
	sp := g.fm().Spouse
	return sp != nil && sp.Gone == ""
}

func fmAgeYears(g *Game, k *fmChild) float64 { return float64(g.Day()-k.Born) / float64(YearDays) }

func fmStage(age float64) string {
	switch {
	case age < 0.5:
		return "infant"
	case age < 1.5:
		return "child"
	case age < 3:
		return "youth"
	}
	return "grown"
}

func (g *Game) fmFamilyKey() string {
	st := g.fm()
	if sp := st.Spouse; sp != nil && sp.Gone == "" && sp.HomeKey != "" && sp.HomePlanet == g.W.uniIdx {
		return sp.HomeKey
	}
	if hs := g.homeSt(); len(hs.Homes) > 0 {
		return hs.Homes[0].Key
	}
	return ""
}

// ---- hooks: zone (bards), enter, side ----

func fmZone(g *Game, zn *Zone, _ *Rng) {
	if zn.Z != 0 || zn.Kind != "town" || zn.Burg <= 0 {
		return
	}
	bd := g.W.burgByID[zn.Burg]
	if bd == nil || g.fmRoll("bardzone", zn.Key) >= 0.45 {
		return
	}
	for _, a := range zn.Actors {
		if a.Role == "bard" {
			return
		}
	}
	r := NewRng(g.Seed + ":fm:bard:" + zn.Key)
	x, y, ok := zn.FreeSpot(r, func(x, y int) bool { return zn.At(x, y) == TCobble || zn.At(x, y) == TFloor })
	if !ok {
		return
	}
	a := g.makeNPC("bard", bd, g.W.cultByID[bd.Culture], r)
	a.Title, a.F, a.AI = "bard", CMagenta, "town"
	a.Desc = fmt.Sprintf("A travelling bard in %s, a lute on their back and half a song on their lips.", bd.Name)
	a.X, a.Y, a.HomeX, a.HomeY = x, y, x, y
	a.Gold = r.Int(20, 90)
	a.Inv = nil
	zn.Actors = append(zn.Actors, a)
}

func fmEnter(g *Game, zn *Zone) {
	st := g.fm()
	// kilometres walked
	pl := g.W.uniIdx
	if !st.KmInit || st.LastPl != pl {
		st.KmInit, st.LastPl, st.LastWX, st.LastWY = true, pl, zn.WX, zn.WY
	} else if zn.WX != st.LastWX || zn.WY != st.LastWY {
		d := maxi(abs(zn.WX-st.LastWX), abs(zn.WY-st.LastWY))
		st.Km += g.tvKm() * float64(mini(d, 2))
		st.LastWX, st.LastWY = zn.WX, zn.WY
	}
	// the household
	key := g.fmFamilyKey()
	if sp := st.Spouse; sp != nil && sp.Gone == "" && !sp.Away && zn.Key == sp.HomeKey && sp.HomePlanet == pl {
		if a, _ := g.fmSpouseActor(); a == nil {
			g.fmPlaceSpouse(zn)
		}
	}
	if key != "" && zn.Key == key {
		g.fmPlaceKids(zn)
	}
	g.fmRescuePlace(zn)
}

func fmSide(g *Game) []string {
	st := g.fm()
	var out []string
	if t := fmTitleByID[st.Worn]; t != nil {
		out = append(out, "{{Y|"+t.Name+"}}")
	}
	if d := st.Duel; d != nil {
		out = append(out, fmt.Sprintf("{{R|Duel: %s, %s}}", d.OppName, fmRuleName[d.Rules]))
	}
	if tn := g.fml().Tourn; tn != nil {
		out = append(out, fmt.Sprintf("{{O|%s: round %d/%d}}", tn.Name, tn.Round+1, len(tn.Foes)))
	}
	if r := st.Rescue; r != nil && r.Planet == g.W.uniIdx {
		out = append(out, fmt.Sprintf("{{R|%s taken! %d days}}", givenOf(r.Name), maxi(0, r.Deadline-g.Day())))
	}
	return out
}

// ---- exports for other files ----

// fmGreeting: a prefix greeting for NPCs when you wear a title (hook in talk.go greeting).
func (g *Game) fmGreeting(a *Actor) string {
	st := g.fm()
	t := fmTitleByID[st.Worn]
	if t == nil || a == nil || a.Has("fm_spouse") || a.Has("fm_child") || a.AI == "follower" {
		return ""
	}
	vals, ok := thRoleValues[a.Role]
	if !ok {
		vals = thValues{[]string{"mercy", "fellowship"}, []string{"blood"}}
	}
	nm := fmTitleName(t, givenOf(g.P.Name))
	for _, d := range vals.dislike {
		if d == t.Tag {
			return fmt.Sprintf("\"%s. I know that name, and not kindly.\" %s keeps a hand near the door.", nm, givenOf(a.Name))
		}
	}
	for _, l := range vals.like {
		if l == t.Tag {
			return fmt.Sprintf("\"Ah, %s! Your name is spoken with respect here.\"", nm)
		}
	}
	return fmt.Sprintf("\"Ah, %s. Well met.\"", nm)
}

// fmRepMod: percent price shift from your worn title for this NPC (negative = cheaper).
func (g *Game) fmRepMod(a *Actor) int {
	t := fmTitleByID[g.fm().Worn]
	if t == nil || a == nil {
		return 0
	}
	if v, ok := t.Mod[a.Role]; ok {
		return v
	}
	return t.Mod["*"]
}

// fmPriceMod is fmRepMod as a multiplier.
func (g *Game) fmPriceMod(a *Actor) float64 { return 1 + float64(g.fmRepMod(a))/100 }

// fmGossip: a rumour of the songs about you that this town has heard (hook in talk.go gossip).
func (g *Game) fmGossip(a *Actor) string {
	bd := g.localBurg()
	if bd == nil || a == nil || g.fmRoll("gos", a.ID, g.Day()) >= 0.5 {
		return ""
	}
	known := g.fmKnown(bd)
	if len(known) == 0 {
		return ""
	}
	s := g.fm().Songs[known[len(known)-1]]
	txt := s.Text
	if len(txt) > 160 {
		txt = txt[:160] + "..."
	}
	return fmt.Sprintf("\"A bard passed through singing of a wanderer called %s. They sing that %s\"", g.P.Name, txt)
}

// fmSummary: one paragraph about your family and fame for a hall of fame.
func (g *Game) fmSummary() string {
	st := g.fm()
	var parts []string
	if sp := st.Spouse; sp != nil {
		w := "wed to " + sp.Name
		switch sp.Gone {
		case "dead":
			w = "widowed of " + sp.Name
		case "left":
			w = "left by " + sp.Name
		case "lost":
			w = "bereaved of " + sp.Name
		}
		parts = append(parts, w)
	}
	if len(st.Kids) > 0 {
		var ks []string
		for _, k := range st.Kids {
			ks = append(ks, k.Name)
		}
		parts = append(parts, "children "+strings.Join(ks, ", "))
	}
	if t := fmTitleByID[st.Worn]; t != nil {
		parts = append(parts, "known as "+fmTitleName(t, g.P.Name))
	}
	if n := st.sungCount(); n > 0 {
		parts = append(parts, fmt.Sprintf("%d songs sung", n))
	}
	if st.DuelW+st.DuelL > 0 {
		parts = append(parts, fmt.Sprintf("duels %d-%d", st.DuelW, st.DuelL))
	}
	if st.Champs > 0 {
		parts = append(parts, fmt.Sprintf("%d tournaments won", st.Champs))
	}
	if len(parts) == 0 {
		return "No family, no songs."
	}
	return strings.Join(parts, "; ") + "."
}

// fmHeirData is what a legacy-run system needs to continue as your child.
type fmHeirData struct {
	Name      string
	Species   string
	Calling   string
	Female    bool
	LevelFrac float64 // fraction of your level the heir starts at
	Items     []*Item // your equipped and most valuable gear
	Orphaned  bool    // no living parent left besides you
	Fame      int     // best realm fame to inherit a share of
	Titles    []string
	Parent    string
}

// fmHeir picks the eldest grown, living child; nil if none.
func (g *Game) fmHeir() *fmHeirData {
	st := g.fm()
	var best *fmChild
	for _, k := range st.Kids {
		if k.Fate == "dead" || k.Fate == "lost" || k.Fate == "kidnapped" || fmAgeYears(g, k) < 3 {
			continue
		}
		if best == nil || k.Born < best.Born {
			best = k
		}
	}
	if best == nil {
		return nil
	}
	h := &fmHeirData{Name: best.Name, Species: best.Species, Calling: best.Calling, Female: best.Female, Parent: g.P.Name}
	h.LevelFrac = fmClampf(0.45+0.02*float64(best.Bond), 0.45, 0.8)
	for _, bp := range g.P.Body {
		if bp.Item != nil {
			cp := *bp.Item
			h.Items = append(h.Items, &cp)
		}
	}
	inv := append([]*Item(nil), g.P.Inv...)
	sort.SliceStable(inv, func(i, j int) bool { return inv[i].Value > inv[j].Value })
	for i := 0; i < len(inv) && i < 6; i++ {
		cp := *inv[i]
		h.Items = append(h.Items, &cp)
	}
	h.Orphaned = st.Spouse == nil || st.Spouse.Gone != ""
	h.Fame = st.fameMax() / 2
	for _, t := range fmTitles {
		if st.Titles[t.ID] > 0 {
			h.Titles = append(h.Titles, t.Name)
		}
	}
	return h
}

func fmClampf(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// ---- turn ----

func fmTurn(g *Game) {
	if g.Cur == nil || g.P == nil {
		return
	}
	st := g.fm()
	if st.Duel != nil {
		g.fmDuelTick()
	}
	if g.fml().Tourn != nil {
		g.fmTournTick()
	}
	if g.Turn%30 == 0 {
		if g.P.Gold > st.Peak {
			st.Peak = g.P.Gold
		}
		g.fmRescueTick()
	}
	if g.Turn%60 == 7 {
		g.fmScanDeeds()
		g.fmEvalTitles()
	}
	if d := g.Day(); d != st.LastDay {
		st.LastDay = d
		g.fmDaily()
	}
}

func (g *Game) fmDaily() {
	st := g.fm()
	day := g.Day()
	g.fmScanDeeds()
	g.fmEvalTitles()
	if g.P.Gold > st.Peak {
		st.Peak = g.P.Gold
	}
	// the spouse
	if sp := st.Spouse; sp != nil && sp.Gone == "" {
		if a, z := g.fmSpouseActor(); a != nil {
			if b, err := json.Marshal(a); err == nil {
				sp.Raw = b
			}
			sp.Gold = a.Gold
			if a.AI != "follower" && z != nil {
				sp.HomeKey, sp.HomePlanet = z.Key, g.W.uniIdx
			}
			if a.AI == "follower" {
				if day%3 == 0 {
					sp.Love = mini(100, sp.Love+1)
				}
			} else if day%7 == 0 {
				sp.Love--
			}
		} else if !sp.Away && day%7 == 0 {
			sp.Love--
		}
		if sp.Love <= 0 {
			g.fmSpouseLeaves()
		}
	}
	// kids: sync companion state, apprentices, sickness
	for _, k := range st.Kids {
		switch k.Fate {
		case "comp":
			if a, _ := g.fmKidActor(k); a == nil {
				k.Fate = ""
			} else if a.AI != "follower" {
				k.Fate = ""
			}
		case "away":
			if day-k.Away >= 28 {
				k.Fate = ""
				gold := 60 + k.Attr[0]*3
				g.cuEarn(gold)
				g.fmFame(g.fmRealm(), 3)
				g.Msg(fmt.Sprintf("{{G|%s returns from the road with a name of their own, and %s.}}", k.Name, g.fmCoin(gold)))
				g.deed("kinship", fmt.Sprintf("%s's %s %s made a name on the roads.", g.P.Name, fmKid(k), k.Name), 0, 0)
			}
		case "":
			g.fmSickness(k)
		}
	}
	g.fmBirthCheck()
	g.fmKidnapCheck()
	if r := st.Rescue; r != nil && day > r.Deadline {
		g.fmRescueFail()
	}
	g.fmChallengeCheck()
	if c := st.Chal; c != nil && day > c.Expires {
		st.Chal = nil
		g.fmFame(g.fmRealm(), -3)
		g.Msg("{{K|The challenge lapses unanswered. People talk.}}")
	}
	g.fmSongsDaily()
}

func fmKid(k *fmChild) string {
	if k.Female {
		return "daughter"
	}
	return "son"
}

// ---- titles ----

var fmDigits = regexp.MustCompile(`\b\d+\b`)

func (g *Game) fmScanDeeds() {
	st := g.fm()
	for i := st.Scan; i < len(g.Deeds); i++ {
		d := g.Deeds[i]
		low := strings.ToLower(d.Text)
		for k, v := range fmDeedKeys {
			if strings.Contains(low, k) {
				st.DeedCt[v]++
			}
		}
		if fmMajorKind[d.Kind] && d.Text != "" && len(st.Songs) < 40 {
			h := g.fmH("song", i)
			if h%5 != 0 || d.Kind == "slay" || d.Kind == "relic" || d.Kind == "quest" || d.Kind == "endgame" {
				ox, oy := 0, 0
				if g.Cur != nil {
					ox, oy = g.Cur.WX, g.Cur.WY
				}
				st.Songs = append(st.Songs, &fmSong{Deed: i, Text: d.Text, Day: g.Day(), OX: ox, OY: oy})
			}
		}
	}
	st.Scan = len(g.Deeds)
}

func (g *Game) fmEvalTitles() {
	st := g.fm()
	for i := range fmTitles {
		t := &fmTitles[i]
		if st.Titles[t.ID] > 0 || !t.Cond(g, st) {
			continue
		}
		st.Titles[t.ID] = g.Day() + 1
		g.Msg(fmt.Sprintf("{{Y|Title earned: %s.}} {{K|%s (F7)}}", t.Name, t.Perk))
		g.deed("event", fmt.Sprintf("%s was hailed as %s.", g.P.Name, fmTitleName(t, "")), 0, 0)
		g.fmLog("Earned the title " + t.Name)
		if st.Worn == "" {
			st.Worn = t.ID
		}
	}
}

// ---- songs and bards ----

func (g *Game) fmSongDelay(s *fmSong, bd *BurgData) int {
	bx, by := bd.P%g.W.W, bd.P/g.W.W
	dist := maxi(abs(bx-s.OX), abs(by-s.OY))
	return maxi(1, 2+dist*2-s.Boost*3)
}

func (g *Game) fmKnown(bd *BurgData) []int {
	var out []int
	for i, s := range g.fm().Songs {
		if g.Day() >= s.Day+g.fmSongDelay(s, bd) {
			out = append(out, i)
		}
	}
	return out
}

func (g *Game) fmSongsDaily() {
	// towns you pass through gradually learn: nothing to simulate, knowledge is computed on demand
}

func (g *Game) fmSongText(s *fmSong) string {
	r := NewRng(fmt.Sprintf("%s:fm:lyric:%d", g.Seed, s.Deed))
	flair := clampi(s.Boost+r.Int(0, 1), 0, 3)
	txt := s.Text
	if flair >= 2 {
		txt = fmDigits.ReplaceAllStringFunc(txt, func(m string) string {
			n := 0
			fmt.Sscanf(m, "%d", &n)
			return fmt.Sprint(n * 2)
		})
	}
	out := fmt.Sprintf(Pick(r, fmSongOpen), g.P.Name) + "\n\n\"" + txt + "\""
	if f := fmFlairs[flair]; f[0] != "" || len(f) > 1 {
		if l := Pick(r, f); l != "" {
			out += " " + l
		}
	}
	return out
}

func (g *Game) fmBardOpts(a *Actor) []Option {
	bd := g.localBurg()
	if bd == nil {
		return nil
	}
	return []Option{
		{"{{M|Ask for a song of your deeds.}}", func() { g.talkMenu(g.fmBardSing(a, bd)) }},
		{"{{M|Ask for an old ballad.}}", func() { g.talkMenu(g.fmOldBallad(bd)) }},
		{"{{y|Tip the bard.}}", func() { g.fmTipMenu(a) }},
	}
}

func (g *Game) fmBardSing(a *Actor, bd *BurgData) string {
	st := g.fm()
	known := g.fmKnown(bd)
	if len(known) == 0 {
		if len(st.Songs) == 0 {
			return "The bard strums. \"Nothing of yours yet, friend. Do something worth singing and we'll see.\""
		}
		return "\"I have heard of you, but the tale has not reached our town in song yet. Give it a few days. A tip would hurry it along.\""
	}
	idx := known[int(g.fmH("pick", a.ID, g.Day())%uint64(len(known)))]
	s := st.Songs[idx]
	key := fmt.Sprintf("%d:%d", idx, bd.ID)
	if _, ok := g.fml().Heard[key]; !ok {
		g.fml().Heard[key] = g.Day()
		s.Sung++
		g.fmFame(bd.State, 2+s.Boost)
		g.Msg(fmt.Sprintf("{{M|%s sings of you. Fame in %s grows.}}", a.Name, g.realmName(bd.State)))
	}
	return g.fmSongText(s)
}

func (g *Game) fmOldBallad(bd *BurgData) string {
	var pool []EventData
	for _, e := range g.W.Events {
		if e.P == bd.P || e.St == bd.State {
			pool = append(pool, e)
		}
	}
	if len(pool) == 0 {
		pool = g.W.Events
	}
	if len(pool) == 0 {
		return "The bard hums an old tune without words."
	}
	e := pool[int(g.fmH("old", bd.ID, g.Day())%uint64(len(pool)))]
	return fmt.Sprintf("The bard clears their throat. \"This one is older than I am. In the year %d: %s\"", e.Y, e.Text)
}

func (g *Game) fmTipMenu(a *Actor) {
	g.Popup = Wrap("How much will you tip? A bard sings your songs sooner, farther and grander.", 100)
	g.Opts = nil
	g.Sel = 0
	for _, amt := range []int{10, 50, 200} {
		amt := amt
		g.Opts = append(g.Opts, Option{fmt.Sprintf("Tip %s.", g.fmCoin(amt)), func() { g.fmTip(a, amt) }})
	}
	g.Opts = append(g.Opts, Option{"Never mind.", func() { g.talkMenu(g.greeting(a)) }})
}

func (g *Game) fmTip(a *Actor, amt int) {
	st := g.fm()
	if !g.cuSpend(amt) {
		g.talkMenu("\"Keep your coin, friend. You haven't got it.\"")
		return
	}
	a.Gold += amt
	st.Tips++
	lv := 1
	if amt >= 200 {
		lv = 4
	} else if amt >= 50 {
		lv = 2
	}
	if n := len(st.Songs); n > 0 {
		st.Songs[n-1].Boost = mini(st.Songs[n-1].Boost+lv, 6)
	}
	g.fmFame(g.fmRealm(), lv)
	g.talkMenu("The bard pockets the coin with a bow. \"Your name will travel faster than you do.\"")
}

// ---- courtship ----

func fmStageName(s int) string {
	return [...]string{"acquaintance", "courting", "betrothed"}[clampi(s, 0, 2)]
}

var fmCourtRoles = map[string]bool{"villager": true, "merchant": true, "smith": true, "apothecary": true, "bookseller": true, "exotic": true,
	"tailor": true, "augmenter": true, "notable": true, "priest": true, "guard": true, "trainer": true, "bard": true, "ferry": true}

func (g *Game) fmEligible(a *Actor) bool {
	if a == nil || a == g.P || a.AI == "follower" || a.Kinship || a.HP <= 0 || g.Hostile(a) || !fmCourtRoles[a.Role] {
		return false
	}
	for _, f := range []string{"fm_spouse", "fm_child", "fm_hostage"} {
		if a.Has(f) {
			return false
		}
	}
	if famRole(a) == "grandmother" || famRole(a) == "grandfather" || famRole(a) == "aunt" || famRole(a) == "uncle" {
		return false
	}
	t := strings.ToLower(a.Title)
	return !strings.Contains(t, "child") && !strings.Contains(t, "elder")
}

func (g *Game) fmInterestOf(a *Actor) string {
	likes := fmInterests
	if v, ok := thRoleValues[a.Role]; ok {
		likes = v.like
	}
	return likes[int(g.fmH("int", a.ID)%uint64(len(likes)))]
}

func (g *Game) fmCompat(a *Actor, c *fmCourt) int {
	v := 30 + 8*thTraitRep(g, a) + clampi(thFame(g)/5, 0, 10) + clampi(g.Rep[a.Faction]/40, -10, 10)
	if c.Interest != "" && g.fmHasTag(c.Interest) {
		v += 12
	}
	if a.Species == g.P.Species {
		v += 6
	}
	v += int(g.fmH("cp", a.ID)%15) - 5 + g.Skills["persuasion"]*2
	return clampi(v, 0, 100)
}

func (g *Game) fmGetCourt(a *Actor) *fmCourt {
	st := g.fm()
	if c := st.Courts[a.ID]; c != nil {
		return c
	}
	c := &fmCourt{ID: a.ID, Name: a.Name, Interest: g.fmInterestOf(a), Burg: g.Cur.Burg, ZoneKey: g.Cur.Key, Planet: g.W.uniIdx}
	if g.fmRoll("rival", a.ID) < 0.6 {
		c.RivalStr = 8 + int(g.fmH("rs", a.ID)%25)
		cu := g.W.cultByID[0]
		if bd := g.localBurg(); bd != nil {
			cu = g.W.cultByID[bd.Culture]
		}
		c.Rival = g.cultureName(cu, NewRng(g.Seed+":fm:rivalname:"+fmt.Sprint(a.ID)))
	}
	if b, err := json.Marshal(a); err == nil {
		c.Raw = b
	}
	st.Courts[a.ID] = c
	return c
}

func (g *Game) fmCourtOpts(a *Actor) []Option {
	st := g.fm()
	if g.fmMarried() {
		return nil
	}
	for id, c := range st.Courts {
		if c.Stage == 2 && id != a.ID {
			return nil
		}
	}
	if !g.fmEligible(a) && !a.Has("fm_betrothed") {
		return nil
	}
	label := "{{M|Court them.}}"
	if c := st.Courts[a.ID]; c != nil {
		label = fmt.Sprintf("{{M|Court %s.}} {{K|(%s, affection %d)}}", givenOf(a.Name), fmStageName(c.Stage), c.Aff)
	}
	return []Option{{label, func() { g.fmCourtMenu(a, "") }}}
}

func (g *Game) fmAffWord(n int) string {
	switch {
	case n < 15:
		return "barely know you"
	case n < 35:
		return "are warming to you"
	case n < 60:
		return "like you well"
	case n < 80:
		return "are fond of you"
	}
	return "are devoted to you"
}

func (g *Game) fmCourtMenu(a *Actor, note string) {
	c := g.fmGetCourt(a)
	text := note
	if text != "" {
		text += "\n\n"
	}
	text += fmt.Sprintf("%s: {{Y|%s}}. They %s. Affection %d/100.", a.Name, fmStageName(c.Stage), g.fmAffWord(c.Aff), c.Aff)
	if c.Meets >= 2 {
		text += fmt.Sprintf(" They seem to care for %s.", fmInterestWord[c.Interest])
	}
	if c.Rival != "" && !c.RivalBeaten && c.Meets >= 1 {
		text += fmt.Sprintf(" You hear that {{R|%s}} is also courting them.", c.Rival)
	}
	if c.Stage == 2 {
		text += " You are betrothed: hold the wedding at any temple (talk to a priest)."
	}
	g.Talk, g.Mode = a, "talk"
	g.Popup = Wrap(text, 100)
	g.Opts = nil
	g.Sel = 0
	add := func(t string, f func()) { g.Opts = append(g.Opts, Option{t, f}) }
	if c.Stage < 2 {
		add("{{M|Spend time together.}}", func() { g.fmSpend(a, c) })
		add("{{W|Give a gift.}}", func() { g.fmGiftMenu(a, c) })
	}
	if c.Stage >= 1 && c.Stage < 2 {
		add("{{R|Ask for their hand.}}", func() { g.fmAskHand(a, c) })
		if c.Rival != "" && !c.RivalBeaten {
			add(fmt.Sprintf("{{R|Challenge %s for their hand.}}", c.Rival), func() { g.fmRivalDuel(a, c) })
		}
	}
	add("Never mind.", func() { g.talkMenu(g.greeting(a)) })
}

func (g *Game) fmSpend(a *Actor, c *fmCourt) {
	day := g.Day()
	if c.Last == day+1 {
		g.fmCourtMenu(a, "You have already spent today with them. Let them breathe.")
		return
	}
	c.Last = day + 1
	g.spendTurns(60)
	comp := g.fmCompat(a, c)
	roll := g.fmRoll("sp", a.ID, day)
	gain := 3 + comp/20 + g.Skills["persuasion"] + int(roll*4)
	line := Pick(NewRng(fmt.Sprint(g.fmH("sl", a.ID, day))), fmSpendLines[c.Interest])
	if g.fmHasTag(c.Interest) {
		gain += 2
	}
	if roll < 0.15 {
		gain = -2
		line = Pick(NewRng(fmt.Sprint(g.fmH("cl", a.ID, day))), fmCoolLines)
	}
	c.Aff = clampi(c.Aff+gain, 0, 100)
	c.Meets++
	if c.Stage == 0 && c.Meets >= 2 {
		c.Stage = 1
		line += " {{M|You are courting now.}}"
	}
	g.fmCourtMenu(a, line)
}

func (g *Game) fmPickGift(prompt string, back func(), cb func(it *Item)) {
	g.Popup = Wrap(prompt, 100)
	g.Opts = nil
	g.Sel = 0
	for _, it := range g.P.Inv {
		if !giftable(it) || len(g.Opts) >= 16 {
			continue
		}
		it := it
		g.Opts = append(g.Opts, Option{it.Display(), func() { cb(it) }})
	}
	g.Opts = append(g.Opts, Option{"Never mind.", back})
}

func fmGiftFit(interest string, it *Item) bool {
	switch interest {
	case "lore":
		return it.Kind == "book"
	case "valor":
		return it.Kind == "weapon" || it.Kind == "armor" || it.Kind == "ranged"
	case "greed":
		return it.Value >= 60
	case "temperance", "mercy":
		return it.Kind == "food" || it.Kind == "dish" || it.Kind == "tonic"
	case "road":
		return it.Kind == "light" || it.Kind == "scrap"
	case "faith":
		return it.Kind == "artifact" || it.Kind == "trophy"
	}
	return it.Kind == "food" || it.Kind == "dish"
}

func (g *Game) fmGiftMenu(a *Actor, c *fmCourt) {
	g.fmPickGift("What will you give "+a.Name+"?", func() { g.fmCourtMenu(a, "") }, func(it *Item) {
		name := it.Display()
		val := it.Value
		g.takeOne(it)
		gain := clampi(val/8, 1, 16)
		line := "They turn it over, pleased."
		if fmGiftFit(c.Interest, it) {
			gain += 5
			line = "It is exactly the sort of thing they love."
		}
		c.Aff = clampi(c.Aff+gain, 0, 100)
		c.Gifts++
		g.fmCourtMenu(a, fmt.Sprintf("You give %s. %s (+%d)", Plain(name), line, gain))
	})
}

func (g *Game) fmSettlement(a *Actor) (int, bool, bool) {
	high, out := g.casteClass(a)
	base := 60 + a.Level*35
	price := base
	switch {
	case high:
		price = base * 3
	case out:
		price = base / 2
	}
	return price, high, out
}

func (g *Game) fmAskHand(a *Actor, c *fmCourt) {
	price, high, _ := g.fmSettlement(a)
	need := 55 + c.RivalStr/2 - thFame(g)/10
	stand := g.socialStanding()
	switch {
	case g.Rep[a.Faction] <= -250 || (high && stand < 0 && thFame(g) < 25):
		g.fmCourtMenu(a, "\"My family would never consent to such a match.\" Their caste and yours are too far apart for now.")
		return
	case c.Aff < need:
		line := Pick(NewRng(fmt.Sprint(g.fmH("refuse", a.ID, g.Day()))), fmRefuseLines)
		if c.Rival != "" && !c.RivalBeaten {
			line = fmt.Sprintf("\"%s has the family's ear. Settle it between you.\"", c.Rival)
		}
		g.fmCourtMenu(a, line)
		return
	}
	g.Popup = Wrap(fmt.Sprintf("\"The family consents,\" says %s, \"if the settlement is met: %s as bride-price and dowry, as custom here asks.\"", a.Name, g.fmCoin(price)), 100)
	g.Opts = []Option{
		{fmt.Sprintf("{{G|Pay the settlement.}} {{K|(%s)}}", g.fmCoin(price)), func() {
			if !g.cuSpend(price) {
				g.fmCourtMenu(a, "You cannot meet the settlement yet.")
				return
			}
			c.Stage = 2
			c.Betrothed = g.Day()
			if b, err := json.Marshal(a); err == nil {
				c.Raw = b
			}
			a.Set("fm_betrothed")
			g.deed("kinship", fmt.Sprintf("%s was betrothed to %s.", g.P.Name, a.Name), 0, 0)
			g.fmLog("Betrothed to " + a.Name)
			if c.Rival != "" && !c.RivalBeaten {
				g.fmQueueChallenge(c.Rival, fmt.Sprintf(fmJilted, c.Rival), "blood", 0)
			}
			g.fmCourtMenu(a, "{{G|You are betrothed.}} Hold the wedding at a temple when you are ready (at least two days from now).")
		}},
		{"Not yet.", func() { g.fmCourtMenu(a, "") }},
	}
	g.Sel = 0
}

func (g *Game) fmRivalDuel(a *Actor, c *fmCourt) {
	if g.Cur.Kind != "town" {
		g.fmCourtMenu(a, "This is no place for a duel of honour.")
		return
	}
	rules := g.fmRuleMax()
	if rules == "death" {
		rules = "yield"
	}
	opp := g.fmSpawnFighter(c.Rival, "human", maxi(1, g.P.Level+c.RivalStr/12))
	if opp == nil {
		g.fmCourtMenu(a, "No room to fight here.")
		return
	}
	g.fmDuelStart(opp, rules, "hand", 0, nil, false, c.ID)
}

// ---- weddings ----

func (g *Game) fmWeddingOpts(a *Actor) []Option {
	st := g.fm()
	if a.Role != "priest" || g.Cur.Kind != "town" || g.fmMarried() {
		return nil
	}
	for _, c := range st.Courts {
		if c.Stage == 2 && g.Day() >= c.Betrothed+2 {
			c := c
			fee := 30 + 30*g.W.cityTier[g.Cur.Burg]
			return []Option{{fmt.Sprintf("{{M|Hold the wedding with %s here.}} {{K|(%s)}}", c.Name, g.fmCoin(fee)), func() { g.fmWedding(a, c, fee) }}}
		}
	}
	return nil
}

func (g *Game) fmWedding(priest *Actor, c *fmCourt, fee int) {
	st := g.fm()
	if !g.cuSpend(fee) {
		g.talkMenu("\"The temple asks " + g.fmCoin(fee) + " for the rites. Come back when you have it.\"")
		return
	}
	var raw Actor
	if err := json.Unmarshal(c.Raw, &raw); err != nil {
		g.talkMenu("Something is wrong with the records of the betrothal.")
		return
	}
	if c.Planet != g.W.uniIdx {
		// the betrothed is of another world; they arrive from their own people regardless
		c.Planet = g.W.uniIdx
	}
	if z := g.Zones[c.ZoneKey]; z != nil {
		for _, o := range z.Actors {
			if o.ID == c.ID {
				g.removeActor(z, o)
				break
			}
		}
	}
	sp := &fmSpouse{ID: c.ID, Name: c.Name, Species: raw.Species, Role: raw.Role, Caste: raw.Caste, Raw: c.Raw, Attr: raw.Attr, Level: raw.Level,
		Love: clampi(40+c.Aff/2, 40, 90), Wed: g.Day(), HomeKey: g.Cur.Key, HomePlanet: g.W.uniIdx, Gold: raw.Gold, Female: isFemale(&raw) || raw.Has("soc_f")}
	st.Spouse = sp
	st.NextBorn = g.Day() + 84
	delete(st.Courts, c.ID)
	g.spendTurns(60)
	bd := g.localBurg()
	town := g.Cur.Name
	if bd != nil {
		town = bd.Name
		g.AddRep(fmt.Sprintf("state:%d", bd.State), 25, " for the wedding feast")
		g.fmFame(bd.State, 3)
	}
	_, f := g.feastToday()
	if f != nil && bd != nil {
		g.AddRep(fmt.Sprintf("state:%d", bd.State), 10, " for wedding on a feast day")
	}
	g.applyMeal("wedding feast", tagsOf("hearty"), DayLen, true)
	g.gainXP(20)
	g.Feasts = append(g.Feasts, FeastLog{Name: "Wedding of " + sp.Name, Town: town, Year: g.W.DateOf(nil, g.Day()).Year, Day: g.Day(),
		Text: fmt.Sprintf("You wed %s at the temple of %s. The town eats and sings until dusk.", sp.Name, town)})
	g.deed("kinship", fmt.Sprintf("%s wed %s in %s.", g.P.Name, sp.Name, town), 0, 0)
	g.fmLog("Wed " + sp.Name)
	g.fmPlaceSpouse(g.Cur)
	g.fmWedChoice(priest, sp)
}

func (g *Game) fmWedChoice(priest *Actor, sp *fmSpouse) {
	g.Talk, g.Mode = priest, "talk"
	g.Popup = Wrap(fmt.Sprintf("{{G|%s and you are wed.}} The town shares a feast in the temple court; the bells ring for an hour, and %s is sent off with a portion wrapped for the road.\n\n{{W|Where will %s live?}}", sp.Name, sp.Name, sp.Name), 100)
	g.Opts = []Option{
		{"Come along with me, as a companion.", func() {
			if a, _ := g.fmSpouseActor(); a != nil {
				g.fmSpouseFollow(a)
			}
			g.talkMenu("\"May your road be kind to both of you.\"")
		}},
		{"Keep house for me.", func() {
			if hs := g.homeSt(); len(hs.Homes) > 0 && g.homeHere() == nil {
				sp.HomeKey, sp.HomePlanet = hs.Homes[0].Key, g.W.uniIdx
				if a, z := g.fmSpouseActor(); a != nil && z != nil {
					g.removeActor(z, a)
				}
				g.Msg(fmt.Sprintf("%s goes ahead to your house.", sp.Name))
			} else if g.homeHere() == nil {
				g.Msg(fmt.Sprintf("%s will wait here until you have a house to keep. (F7)", sp.Name))
			}
			g.talkMenu("\"A home is a good thing to come back to.\"")
		}},
	}
	g.Sel = 0
}

func (g *Game) fmMakeSpouseActor(sp *fmSpouse) *Actor {
	var a Actor
	if json.Unmarshal(sp.Raw, &a) != nil {
		return nil
	}
	a.ID = sp.ID
	a.Set("fm_spouse")
	a.Flags["fm_spouse"] = true
	delete(a.Flags, "fm_betrothed")
	a.Comp, a.Order = nil, ""
	a.AI, a.Faction, a.Angry = "still", "player", false
	a.F = CMagenta
	a.Title = "your spouse"
	a.Talked = false
	a.HP = a.MaxHP
	a.Kin = nil
	return &a
}

func (g *Game) fmPlaceSpouse(zn *Zone) {
	sp := g.fm().Spouse
	if sp == nil {
		return
	}
	a := g.fmMakeSpouseActor(sp)
	if a == nil {
		return
	}
	px, py := g.P.X+1, g.P.Y
	if zn != g.Cur {
		px, py = 40, 12
	}
	if h := g.fmHomeRec(zn.Key); h != nil {
		px, py = h.X, h.Y
	}
	a.X, a.Y = px, py
	g.nudge(zn, a)
	a.HomeX, a.HomeY = a.X, a.Y
	zn.Actors = append(zn.Actors, a)
}

func (g *Game) fmHomeRec(key string) *homeRec {
	hs := g.homeSt()
	for i := range hs.Homes {
		if hs.Homes[i].Key == key {
			return &hs.Homes[i]
		}
	}
	return nil
}

func (g *Game) fmSpouseFollow(a *Actor) {
	if g.companionCount() >= maxCompanions {
		g.Msg("You already travel with all the companions you can look after.")
		return
	}
	a.AI = "still"
	g.joinCompanion(a)
	if a.Comp != nil {
		a.Comp.Loyalty = 100
	}
}

// fmSpouseKeepHouse: the spouse keeps the house you are standing in.
func (g *Game) fmSpouseKeepHouse() {
	sp := g.fm().Spouse
	if sp == nil || sp.Gone != "" || g.homeHere() == nil {
		g.Msg("You need to be in your own house for that.")
		return
	}
	a, z := g.fmSpouseActor()
	if a == nil {
		sp.HomeKey, sp.HomePlanet = g.Cur.Key, g.W.uniIdx
		g.fmPlaceSpouse(g.Cur)
		g.Msg(fmt.Sprintf("%s settles into the house.", sp.Name))
		return
	}
	if z != g.Cur {
		g.removeActor(z, a)
		sp.HomeKey, sp.HomePlanet = g.Cur.Key, g.W.uniIdx
		g.fmPlaceSpouse(g.Cur)
		g.Msg(fmt.Sprintf("%s moves into the house.", sp.Name))
		return
	}
	a.Comp, a.Order, a.AI, a.Faction = nil, "", "still", "player"
	a.F = CMagenta
	sp.HomeKey, sp.HomePlanet = g.Cur.Key, g.W.uniIdx
	a.HomeX, a.HomeY = a.X, a.Y
	g.Msg(fmt.Sprintf("%s will keep the house.", sp.Name))
}

func (g *Game) fmSpouseLeaves() {
	st := g.fm()
	sp := st.Spouse
	sp.Gone = "left"
	if a, z := g.fmSpouseActor(); a != nil && z != nil {
		g.removeActor(z, a)
	}
	g.Msg(fmt.Sprintf("{{R|%s has left you. The house is quiet.}}", sp.Name))
	g.deed("kinship", fmt.Sprintf("%s was left by %s.", g.P.Name, sp.Name), 0, 0)
	g.fmLog(sp.Name + " left")
}

func (g *Game) fmSpouseOpts(a *Actor) []Option {
	st := g.fm()
	sp := st.Spouse
	if sp == nil || a.AI == "follower" {
		return nil
	}
	day := g.Day()
	var out []Option
	out = append(out, Option{"{{M|Trade news.}}", func() {
		if sp.News == day+1 {
			g.talkMenu("\"You told me already today. Sit with me.\"")
			return
		}
		sp.News = day + 1
		sp.Love = mini(100, sp.Love+3)
		txt := "They listen with their chin on their hand."
		if n := len(g.Deeds); n > 0 {
			txt = fmt.Sprintf("You tell them of the road. They listen closely: \"%s\"", g.Deeds[n-1].Text)
		}
		if gs := g.gossip(a); gs != "" {
			txt += "\n\nIn turn, they tell what the town has been saying: " + gs
		}
		g.talkMenu(txt)
	}})
	if sp.Gold >= 150 || a.Gold >= 150 {
		out = append(out, Option{"{{y|Ask how the house fares.}}", func() {
			gold := maxi(sp.Gold, a.Gold)
			days := mini(14, maxi(0, day-maxi(sp.Pay, sp.Wed)))
			pay := mini(60, gold/15) * days
			if pay <= 0 {
				g.talkMenu("\"Well enough. Nothing to set aside yet. Ask again tomorrow.\"")
				return
			}
			sp.Pay = day
			g.cuEarn(pay)
			g.talkMenu(fmt.Sprintf("\"The household stays in the black.\" %s presses %s into your hand.", givenOf(sp.Name), g.fmCoin(pay)))
		}})
	}
	out = append(out, Option{"{{W|Give a gift.}}", func() {
		g.fmPickGift("What will you give "+sp.Name+"?", func() { g.talkMenu(g.greeting(a)) }, func(it *Item) {
			val := it.Value
			g.takeOne(it)
			sp.Love = mini(100, sp.Love+clampi(val/10, 2, 15))
			g.talkMenu("They smile in a way they save for you.")
		})
	}})
	if sp.Love < 55 || sp.Quarrel > 0 {
		out = append(out, Option{"{{G|Make up.}}", func() {
			sp.Love = mini(100, sp.Love+10)
			sp.Quarrel = 0
			g.spendTurns(30)
			g.talkMenu("You talk for a long time, and the knot loosens.")
		}})
	}
	out = append(out, Option{"{{r|Quarrel over your long absences.}}", func() {
		sp.Love = maxi(0, sp.Love-8)
		sp.Quarrel = day + 1
		g.talkMenu("Words are said that can't be unsaid. They turn away.")
	}})
	out = append(out, Option{"{{C|Come along with me.}}", func() { g.fmSpouseFollow(a); g.Mode, g.Talk = "main", nil }})
	return out
}

// ---- children ----

func (g *Game) fmBirthCheck() {
	st := g.fm()
	sp := st.Spouse
	if sp == nil || sp.Gone != "" || sp.Away || len(st.Kids) >= 4 || g.Day() < st.NextBorn || sp.Love < 30 {
		return
	}
	idx := len(st.Kids)
	r := NewRng(fmt.Sprintf("%s:fm:child:%d", g.Seed, idx))
	k := &fmChild{ID: g.newID(), Born: g.Day(), Female: r.Chance(0.5)}
	if sp.Species == "" || sp.Species == g.P.Species {
		k.Species = g.P.Species
	} else if r.Chance(0.5) {
		k.Species = g.P.Species
	} else {
		k.Species = sp.Species
	}
	for i := range k.Attr {
		k.Attr[i] = clampi((g.P.Attr[i]+sp.Attr[i])/2+r.Int(-1, 1), 6, 22)
	}
	best := 0
	for i := range k.Attr {
		if k.Attr[i] > k.Attr[best] {
			best = i
		}
	}
	k.Calling = fmCallings[best]
	k.Gift = Pick(r, fmChildGifts)
	var cu *CultureData
	if bd := g.localBurg(); bd != nil {
		cu = g.W.cultByID[bd.Culture]
	}
	given := givenOf(g.cultureName(cu, r))
	fam := familyOf(g.P.Name)
	k.Name = strings.TrimSpace(given + " " + fam)
	st.Kids = append(st.Kids, k)
	st.NextBorn = g.Day() + 140 + int(g.fmH("nb", idx)%120)
	g.Msg(fmt.Sprintf("{{G|A %s is born to you and %s: %s.}}", fmKid(k), sp.Name, k.Name))
	g.deed("kinship", fmt.Sprintf("%s and %s welcomed a %s, %s.", g.P.Name, sp.Name, fmKid(k), k.Name), 0, 0)
	g.fmLog("Birth of " + k.Name)
	if g.Cur != nil && g.Cur.Key == g.fmFamilyKey() {
		g.fmPlaceKids(g.Cur)
	}
}

func (g *Game) fmChildActor(k *fmChild) *Actor {
	age := fmAgeYears(g, k)
	scale := 0.4 + 0.2*age
	if scale > 1 {
		scale = 1
	}
	lvl := 1
	glyph := 'c'
	switch fmStage(age) {
	case "infant":
		glyph = 'o'
	case "grown":
		glyph = '@'
		lvl = maxi(2, g.P.Level*2/3)
	case "youth":
		glyph = '@'
		lvl = maxi(1, g.P.Level/3)
	}
	a := &Actor{ID: g.newID(), Name: k.Name, Proper: true, Glyph: rune(glyph), F: CCyan, Tpl: "npc", Faction: "player", Level: lvl, Speed: 100, AI: "town",
		Species: k.Species, Body: MakeBody(BodyPlanForSpecies(k.Species))}
	for i := range a.Attr {
		a.Attr[i] = maxi(6, int(float64(k.Attr[i])*scale))
	}
	a.MaxHP = 8 + lvl*4 + a.Attr[TOU]
	a.HP = a.MaxHP
	a.Title = "your " + fmKid(k)
	a.Desc = fmt.Sprintf("Your %s, %s (%s). They have %s.", fmKid(k), strings.ToLower(fmStage(age)), k.Calling, k.Gift)
	a.Set("fm_child")
	return a
}

func (g *Game) fmPlaceKids(zn *Zone) {
	st := g.fm()
	keep := zn.Actors[:0]
	for _, a := range zn.Actors {
		if a.Has("fm_child") && a.AI != "follower" {
			continue
		}
		keep = append(keep, a)
	}
	zn.Actors = keep
	for _, k := range st.Kids {
		if k.Fate != "" {
			continue
		}
		a := g.fmChildActor(k)
		px, py := 40, 12
		if h := g.fmHomeRec(zn.Key); h != nil {
			px, py = h.X, h.Y
		}
		a.X, a.Y = px+1, py+1
		g.nudge(zn, a)
		a.HomeX, a.HomeY = a.X, a.Y
		zn.Actors = append(zn.Actors, a)
	}
}

func (g *Game) fmKidByName(name string) *fmChild {
	for _, k := range g.fm().Kids {
		if k.Name == name {
			return k
		}
	}
	return nil
}

func (g *Game) fmChildOpts(a *Actor) []Option {
	k := g.fmKidByName(a.Name)
	if k == nil || a.AI == "follower" {
		return nil
	}
	age := fmAgeYears(g, k)
	day := g.Day()
	var out []Option
	out = append(out, Option{"{{M|Play with them.}}", func() {
		if k.Play == day+1 {
			g.talkMenu("They're worn out from earlier. A story, maybe.")
			return
		}
		k.Play = day + 1
		k.Bond = mini(30, k.Bond+1)
		g.spendTurns(20)
		g.talkMenu(Pick(NewRng(fmt.Sprint(g.fmH("play", k.ID, day))), fmChildPlay))
	}})
	if age >= 0.5 {
		out = append(out, Option{"{{c|Tell them about your travels.}}", func() {
			txt := "They hang on every word."
			if n := len(g.Deeds); n > 0 {
				txt = fmt.Sprintf("You tell them: \"%s\" Their eyes are huge.", g.Deeds[n-1].Text)
			}
			k.Bond = mini(30, k.Bond+1)
			g.talkMenu(txt)
		}})
	}
	if fmStage(age) == "youth" {
		out = append(out, Option{"{{W|Teach them what you know.}}", func() {
			if k.Play == day+1 {
				g.talkMenu("\"Tomorrow, please. My head is full.\"")
				return
			}
			k.Play = day + 1
			i := int(g.fmH("tch", k.ID, day) % 6)
			if k.Attr[i] < 20 {
				k.Attr[i]++
			}
			k.Bond = mini(30, k.Bond+2)
			g.spendTurns(60)
			g.talkMenu(fmt.Sprintf("You spend the afternoon on %s. They get a little better at it.", strings.ToLower(AttrNames[i])))
		}})
	}
	if k.Sick {
		out = append(out, Option{"{{R|They are feverish. Care for them.}}", func() {
			if k.SickDay != 0 && g.fmRoll("care", k.ID, day) < 0.5 {
				k.Sick = false
				g.talkMenu("The fever breaks by dawn.")
				return
			}
			g.talkMenu("You sit with them. They need a healer or a remedy: ask a priest or apothecary.")
		}})
	}
	if fmStage(age) == "grown" {
		out = append(out, Option{"{{C|Take them along as a companion.}}", func() { g.fmKidJoin(a, k) }})
		out = append(out, Option{"{{y|Send them out to make their name.}}", func() {
			k.Fate, k.Away = "away", day
			g.removeActor(g.Cur, a)
			g.Msg(fmt.Sprintf("%s sets out with your blessing.", k.Name))
			g.Mode, g.Talk = "main", nil
		}})
	}
	return out
}

func (g *Game) fmKidJoin(a *Actor, k *fmChild) {
	if g.companionCount() >= maxCompanions {
		g.talkMenu("You travel with as many companions as you can look after.")
		return
	}
	lvl := maxi(2, int(float64(g.P.Level)*fmClampf(0.45+0.02*float64(k.Bond), 0.45, 0.8)))
	a.Attr = k.Attr
	a.Level = lvl
	a.MaxHP = 14 + lvl*4 + a.Attr[TOU]
	a.HP = a.MaxHP
	a.AI = "still"
	a.Glyph = '@'
	k.Fate = "comp"
	g.Mode, g.Talk = "main", nil
	g.joinCompanion(a)
}

func (g *Game) fmSickness(k *fmChild) {
	st := g.fm()
	day := g.Day()
	if k.Sick {
		sp := st.Spouse
		carer := sp != nil && sp.Gone == "" && !sp.Away
		switch {
		case carer && g.fmRoll("heal", k.ID, day) < 0.12:
			k.Sick = false
			g.Msg(fmt.Sprintf("{{G|%s recovers, nursed by %s.}}", k.Name, sp.Name))
		case day-k.SickDay >= 10:
			if g.fmRoll("fever", k.ID) < 0.5 {
				k.Fate, k.Sick = "dead", false
				g.Msg(fmt.Sprintf("{{R|%s has died of the fever.}}", k.Name))
				g.deed("kinship", fmt.Sprintf("%s's %s %s died of a fever.", g.P.Name, fmKid(k), k.Name), 0, 0)
				g.fmLog(k.Name + " died of fever")
			} else {
				k.Sick = false
			}
		}
		return
	}
	if g.fmRoll("sick", k.ID, day) < 0.012 {
		k.Sick, k.SickDay = true, day
		g.Msg(fmt.Sprintf("{{R|A runner brings word: %s has fallen sick. A priest or apothecary can help.}}", k.Name))
	}
}

func (g *Game) fmHealerOpts(a *Actor) []Option {
	if a.Role != "priest" && a.Role != "apothecary" {
		return nil
	}
	var out []Option
	for _, k := range g.fm().Kids {
		if !k.Sick {
			continue
		}
		k := k
		cost := 40 + 10*g.P.Level
		out = append(out, Option{fmt.Sprintf("{{G|A remedy for %s.}} {{K|(%s)}}", k.Name, g.fmCoin(cost)), func() {
			if !g.cuSpend(cost) {
				g.talkMenu("\"You can't afford it.\"")
				return
			}
			k.Sick = false
			g.talkMenu(fmt.Sprintf("You send the remedy home with a runner. %s will mend.", k.Name))
		}})
	}
	return out
}

// ---- kidnapping ----

func (g *Game) fmKidnapCheck() {
	st := g.fm()
	day := g.Day()
	if st.Rescue != nil || day < st.LastKid+50 || g.fmRoll("kid", day) >= 0.02 {
		return
	}
	key := g.fmFamilyKey()
	if key == "" || g.Cur.Key == key {
		return
	}
	r := &fmRescue{Planet: g.W.uniIdx, Deadline: day + 12}
	var cu *CultureData
	if bd := g.localBurg(); bd != nil {
		cu = g.W.cultByID[bd.Culture]
	}
	r.Kidnapper = g.cultureName(cu, NewRng(fmt.Sprintf("%s:fm:kidnapper:%d", g.Seed, day)))
	r.Ransom = 150 + g.P.Level*40
	z := g.Zones[key]
	for _, k := range st.Kids {
		if k.Fate == "" && fmStage(fmAgeYears(g, k)) != "infant" {
			k.Fate = "kidnapped"
			r.Kid, r.Name = k.ID, k.Name
			if z != nil {
				if a, _ := g.fmKidActor(k); a != nil {
					g.removeActor(z, a)
				}
			}
			break
		}
	}
	if r.Name == "" {
		sp := st.Spouse
		if sp == nil || sp.Gone != "" || sp.Away {
			return
		}
		if a, _ := g.fmSpouseActor(); a == nil || a.AI == "follower" {
			return
		}
		sp.Away = true
		r.Name = sp.Name
		if a, az := g.fmSpouseActor(); a != nil {
			g.removeActor(az, a)
		}
	}
	st.Rescue, st.LastKid = r, day
	g.Msg(fmt.Sprintf("{{R|A ransom note: %s has been taken by %s's band. %s in coin, or find them in the wilds, within %d days. (F7)}}", r.Name, r.Kidnapper, g.fmCoin(r.Ransom), 12))
	g.fmLog(r.Name + " taken by " + r.Kidnapper)
}

func (g *Game) fmReturnVictim() {
	st := g.fm()
	r := st.Rescue
	if r == nil {
		return
	}
	if r.Kid == 0 {
		if sp := st.Spouse; sp != nil {
			sp.Away = false
			sp.Love = mini(100, sp.Love+10)
		}
	} else {
		for _, k := range st.Kids {
			if k.ID == r.Kid {
				k.Fate = ""
				k.Bond = mini(30, k.Bond+5)
			}
		}
	}
	st.Rescue = nil
	if g.Cur != nil && g.Cur.Key == g.fmFamilyKey() {
		fmEnter(g, g.Cur)
	}
}

func (g *Game) fmPayRansom() {
	r := g.fm().Rescue
	if r == nil {
		return
	}
	if !g.cuSpend(r.Ransom) {
		g.Msg("You can't raise the ransom.")
		return
	}
	g.Msg(fmt.Sprintf("{{G|You pay the ransom. %s is sent home.}}", r.Name))
	g.fmLog("Paid the ransom for " + r.Name)
	g.fmReturnVictim()
}

func (g *Game) fmRescueFail() {
	st := g.fm()
	r := st.Rescue
	if r.Kid == 0 {
		if sp := st.Spouse; sp != nil {
			sp.Gone, sp.Away = "lost", false
		}
	} else {
		for _, k := range st.Kids {
			if k.ID == r.Kid {
				k.Fate = "lost"
			}
		}
	}
	g.Msg(fmt.Sprintf("{{R|No ransom came. %s was never seen again.}}", r.Name))
	g.deed("kinship", fmt.Sprintf("%s lost %s to %s's band.", g.P.Name, r.Name, r.Kidnapper), 0, 0)
	g.fmLog(r.Name + " lost")
	st.Rescue = nil
}

var fmCampKinds = map[string]bool{"camp": true, "lair": true, "ruin": true, "crypt": true, "dungeon": true, "tower": true}

func (g *Game) fmRescuePlace(zn *Zone) {
	st := g.fm()
	r := st.Rescue
	if r == nil || r.Placed || r.Planet != g.W.uniIdx || !fmCampKinds[zn.Kind] || g.fmRoll("res", zn.Key) >= 0.35 {
		return
	}
	rg := NewRng(g.Seed + ":fm:rescue:" + zn.Key)
	n := mini(5, 2+g.P.Level/5)
	for i := 0; i < n; i++ {
		c := g.Spawn("bandit", maxi(2, g.P.Level), rg)
		if i == 0 {
			c.Name, c.Proper, c.Boss = r.Kidnapper, true, false
			c.Gold = r.Ransom
		}
		c.Set("fm_captor")
		x, y, ok := zn.FreeSpot(rg, nil)
		if !ok {
			continue
		}
		c.X, c.Y = x, y
		zn.Actors = append(zn.Actors, c)
	}
	var h *Actor
	if r.Kid != 0 {
		for _, k := range st.Kids {
			if k.ID == r.Kid {
				h = g.fmChildActor(k)
			}
		}
	} else if sp := st.Spouse; sp != nil {
		h = g.fmMakeSpouseActor(sp)
	}
	if h == nil {
		return
	}
	x, y, ok := zn.FreeSpot(rg, nil)
	if !ok {
		return
	}
	h.X, h.Y, h.AI = x, y, "still"
	h.Set("fm_hostage")
	zn.Actors = append(zn.Actors, h)
	r.Placed, r.ZoneKey = true, zn.Key
	g.Msg(fmt.Sprintf("{{R|You hear a familiar voice calling for help: %s is held here by %s's band!}}", r.Name, r.Kidnapper))
}

func (g *Game) fmRescueTick() {
	st := g.fm()
	r := st.Rescue
	if r == nil || !r.Placed || g.Cur.Key != r.ZoneKey {
		return
	}
	for _, a := range g.Cur.Actors {
		if a.HP > 0 && a.Has("fm_captor") {
			return
		}
	}
	for _, a := range g.Cur.Actors {
		if a.Has("fm_hostage") {
			g.removeActor(g.Cur, a)
			break
		}
	}
	g.Msg(fmt.Sprintf("{{G|You cut %s free and send them home with an escort of grateful strangers.}}", r.Name))
	g.deed("kinship", fmt.Sprintf("%s rescued %s from %s's band.", g.P.Name, r.Name, r.Kidnapper), 0, 0)
	g.fmFame(g.fmRealm(), 4)
	g.gainXP(40)
	g.fmLog("Rescued " + r.Name)
	g.fmReturnVictim()
}

// ---- duels ----

func (g *Game) fmRuleMax() string {
	bd := g.localBurg()
	if bd == nil {
		return "death"
	}
	switch h := g.fmH("rule", bd.Culture) % 10; {
	case h < 3:
		return "blood"
	case h < 8:
		return "yield"
	}
	return "death"
}

var fmDuelRoles = map[string]bool{"guard": true, "notable": true, "smith": true, "merchant": true, "villager": true, "trainer": true, "arena": true,
	"bard": true, "apothecary": true, "bookseller": true, "exotic": true}

func (g *Game) fmDuelOpts(a *Actor) []Option {
	if !fmDuelRoles[a.Role] || a.AI == "follower" || a.HP <= 0 || g.fm().Duel != nil || a.Has("fm_spouse") || a.Has("fm_child") || a.Has("fm_hostage") || g.Cur.Kind != "town" {
		return nil
	}
	if strings.Contains(strings.ToLower(a.Title), "child") {
		return nil
	}
	return []Option{{"{{R|Challenge to a duel.}}", func() { g.fmChallengeMenu(a) }}}
}

func (g *Game) fmChallengeMenu(a *Actor) {
	maxR := g.fmRuleMax()
	g.Popup = Wrap(fmt.Sprintf("Custom here allows duels up to {{Y|%s}}. Choose the terms.", fmRuleName[maxR]), 100)
	g.Opts = nil
	g.Sel = 0
	for _, r := range []string{"blood", "yield", "death"} {
		if fmRuleSev[r] > fmRuleSev[maxR] {
			continue
		}
		r := r
		g.Opts = append(g.Opts, Option{"Fight " + fmRuleName[r] + ".", func() { g.fmStakeMenu(a, r) }})
	}
	g.Opts = append(g.Opts, Option{"Never mind.", func() { g.talkMenu(g.greeting(a)) }})
}

func (g *Game) fmStakeMenu(a *Actor, rules string) {
	g.Popup = Wrap("What is at stake?", 100)
	g.Opts = nil
	g.Sel = 0
	add := func(t string, f func()) { g.Opts = append(g.Opts, Option{t, f}) }
	add("Honour only.", func() { g.fmConfirmDuel(a, rules, "honor", 0, nil) })
	for _, amt := range []int{50, 200} {
		amt := amt
		if a.Gold >= amt && g.cuCanSpend(amt) {
			add(fmt.Sprintf("A purse of %s each.", g.fmCoin(amt)), func() { g.fmConfirmDuel(a, rules, "gold", amt, nil) })
		}
	}
	add("Standing in this realm: the loser is shamed.", func() { g.fmConfirmDuel(a, rules, "rep", 0, nil) })
	add("Wager an item of yours against their best.", func() {
		g.fmPickGift("Which item do you wager?", func() { g.fmStakeMenu(a, rules) }, func(it *Item) { g.fmConfirmDuel(a, rules, "item", 0, it) })
	})
	add("Never mind.", func() { g.talkMenu(g.greeting(a)) })
}

func (g *Game) fmConfirmDuel(a *Actor, rules, stake string, amt int, it *Item) {
	wit, sec := 0, 0
	for _, o := range g.Cur.Actors {
		if o == a || o.HP <= 0 || dist(o, g.P) > 12 {
			continue
		}
		if o.AI == "follower" {
			sec++
		} else if !g.Hostile(o) {
			wit++
		}
	}
	txt := fmt.Sprintf("Terms: {{Y|%s}}, stake: {{Y|%s}}. %d witness(es) in sight, %d second(s) at your side. {{K|Strike a foe who has yielded and you are an Oathbreaker.}}", fmRuleName[rules], stake, wit, sec)
	if rules == "death" {
		txt += " {{R|This is to the death.}}"
	}
	g.Popup = Wrap(txt, 100)
	g.Opts = []Option{
		{"{{R|Draw. Let it begin.}}", func() { g.fmDuelStart(a, rules, stake, amt, it, false, 0) }},
		{"Never mind.", func() { g.talkMenu(g.greeting(a)) }},
	}
	g.Sel = 0
}

func (g *Game) fmDuelStart(opp *Actor, rules, stake string, amt int, item *Item, tourney bool, hand int) {
	st := g.fm()
	if st.Duel != nil {
		g.Msg("You are already in a duel.")
		return
	}
	d := &fmDuel{OppID: opp.ID, OppName: opp.Name, ZoneKey: g.Cur.Key, Rules: rules, Stake: stake, Amt: amt, Phase: "fight", OppFac: opp.Faction, OppAI: opp.AI,
		Start: g.Turn, Tourney: tourney, Hand: hand, R: 8, OppHP0: opp.HP, PHP0: g.P.HP}
	d.CX, d.CY = (g.P.X+opp.X)/2, (g.P.Y+opp.Y)/2
	if item != nil {
		d.ItemName, d.ItemID = item.Name, item.ID
	}
	for _, o := range g.Cur.Actors {
		if o == opp || o.HP <= 0 || dist(o, g.P) > 12 {
			continue
		}
		if o.AI == "follower" {
			d.Sec++
		} else if !g.Hostile(o) {
			d.Wit++
		}
	}
	opp.Angry = true
	opp.Set("fm_duel")
	st.Duel = d
	g.Mode, g.Talk = "main", nil
	g.Msg(fmt.Sprintf("{{R|Duel with %s %s!}} Keep within %d paces of the ring; yielding foes must be spared.", opp.Name, fmRuleName[rules], d.R))
}

func (g *Game) fmDuelOpp(d *fmDuel) *Actor {
	z := g.Zones[d.ZoneKey]
	if z == nil {
		return nil
	}
	for _, a := range z.Actors {
		if a.ID == d.OppID {
			return a
		}
	}
	return nil
}

func (g *Game) fmDuelTick() {
	st := g.fm()
	d := st.Duel
	if d == nil {
		return
	}
	if g.Cur.Key != d.ZoneKey {
		if d.Phase == "fight" {
			g.fmDuelEnd(false, "you left the ring")
		} else {
			st.Duel = nil
		}
		return
	}
	opp := g.fmDuelOpp(d)
	if opp == nil {
		st.Duel = nil
		return
	}
	if d.Phase == "watch" {
		if opp.HP < d.YieldHP {
			g.fmOath(d, opp)
			st.Duel = nil
			return
		}
		if g.Turn > d.End {
			st.Spared++
			st.Duel = nil
		}
		return
	}
	if opp.HP <= 0 {
		return
	}
	if d.Rules != "death" && maxi(abs(g.P.X-d.CX), abs(g.P.Y-d.CY)) > d.R+4 {
		g.fmDuelEnd(false, "you stepped out of the ring")
		return
	}
	switch d.Rules {
	case "blood":
		if opp.HP < d.OppHP0 {
			g.fmDuelEnd(true, "first blood")
		} else if g.P.HP < d.PHP0 {
			g.fmDuelEnd(false, "first blood")
		}
	case "yield":
		if opp.HP*100 <= opp.MaxHP*35 {
			g.fmDuelEnd(true, "yield")
		} else if g.P.HP*100 <= g.P.MaxHP*25 {
			g.fmDuelEnd(false, "yield")
		}
	}
}

func (g *Game) fmOath(d *fmDuel, opp *Actor) {
	st := g.fm()
	st.Oath++
	g.Msg(fmt.Sprintf("{{R|You struck %s after they yielded. Oathbreaker!}}", d.OppName))
	g.deed("murder", fmt.Sprintf("%s struck down %s, who had yielded. An oath was broken.", g.P.Name, d.OppName), 0, 0)
	g.fmLog("Oathbreaker: struck " + d.OppName)
	if bd := g.localBurg(); bd != nil && d.Wit > 0 {
		g.AddRep(fmt.Sprintf("state:%d", bd.State), -60, " for striking a yielded foe")
		g.fmFame(bd.State, -8)
	}
}

func (g *Game) fmDuelEnd(win bool, how string) {
	st := g.fm()
	d := st.Duel
	if d == nil {
		return
	}
	opp := g.fmDuelOpp(d)
	bd := g.localBurg()
	fac := ""
	if bd != nil {
		fac = fmt.Sprintf("state:%d", bd.State)
	}
	if opp != nil {
		opp.Angry = false
		delete(opp.Flags, "fm_duel")
		opp.Faction, opp.AI = d.OppFac, d.OppAI
	}
	// stakes
	switch d.Stake {
	case "gold":
		if win && opp != nil {
			gain := mini(d.Amt, maxi(opp.Gold, 0))
			opp.Gold -= gain
			g.cuEarn(gain)
			g.Msg(fmt.Sprintf("{{G|%s pays you %s.}}", d.OppName, g.fmCoin(gain)))
		} else if !win {
			pay := mini(d.Amt, g.P.Gold)
			g.cuSpend(pay)
			if opp != nil {
				opp.Gold += pay
			}
			g.Msg(fmt.Sprintf("{{R|You pay %s %s.}}", d.OppName, g.fmCoin(pay)))
		}
	case "rep":
		if fac != "" {
			if win {
				g.AddRep(fac, 35, " for winning the duel")
			} else {
				g.AddRep(fac, -35, " for losing the duel")
			}
		}
	case "item":
		var mine *Item
		for _, it := range g.P.Inv {
			if it.ID == d.ItemID {
				mine = it
			}
		}
		if win && opp != nil {
			var prize *Item
			if len(opp.Body) > 4 && opp.Body[4].Item != nil {
				prize = opp.Body[4].Item
				opp.Body[4].Item = nil
			}
			if prize != nil {
				g.P.Inv = append(g.P.Inv, prize)
				g.Msg(fmt.Sprintf("{{G|You take %s.}}", Plain(prize.Display())))
			} else if mine != nil {
				g.cuEarn(maxi(10, mine.Value))
				g.Msg("{{G|They can only pay in coin.}}")
			}
		} else if !win && mine != nil {
			g.removeInv(mine)
			if opp != nil {
				opp.Inv = append(opp.Inv, mine)
			}
			g.Msg(fmt.Sprintf("{{R|You lose %s.}}", Plain(mine.Display())))
		}
	case "hand":
		if c := st.Courts[d.Hand]; c != nil {
			if win {
				c.Aff = mini(100, c.Aff+25)
				c.RivalBeaten, c.RivalStr = true, 0
				g.Msg("{{G|The rival withdraws from the courtship.}}")
			} else {
				c.Aff = maxi(0, c.Aff-25)
				c.RivalStr += 10
				g.Msg("{{R|The rival presses their suit.}}")
			}
		}
	}
	realm := g.fmRealm()
	if win {
		st.DuelW++
		g.fmFame(realm, 1+d.Wit)
		g.gainXP(10 + 5*fmRuleSev[d.Rules])
		g.Msg(fmt.Sprintf("{{G|You win the duel against %s (%s).}}", d.OppName, how))
		if d.Stake != "honor" || d.Rules == "death" || d.Tourney || (opp != nil && opp.Role == "notable") {
			g.deed("event", fmt.Sprintf("%s defeated %s in a duel %s.", g.P.Name, d.OppName, fmRuleName[d.Rules]), 0, 0)
		}
	} else {
		st.DuelL++
		g.fmFame(realm, -1)
		g.Msg(fmt.Sprintf("{{R|You lose the duel against %s (%s).}}", d.OppName, how))
		g.P.HP = maxi(1, g.P.HP)
	}
	g.fmLog(fmt.Sprintf("Duel vs %s %s: %s", d.OppName, fmRuleName[d.Rules], map[bool]string{true: "won", false: "lost"}[win]))
	if d.Tourney && opp != nil {
		g.removeActor(g.Cur, opp)
	}
	wasTourney := d.Tourney
	if win && d.Rules == "yield" && opp != nil && !wasTourney {
		d.Phase, d.YieldHP, d.End = "watch", opp.HP, g.Turn+40
		g.Msg(Pick(NewRng(fmt.Sprint(g.fmH("yl", d.OppID))), fmYieldLines))
	} else {
		st.Duel = nil
	}
	if wasTourney {
		g.fmTournAfter(win)
	}
}

func fmKill(g *Game, d, src *Actor) {
	st := g.fm()
	if d == nil {
		return
	}
	if d.Has("legend") || (d.Boss && d.SimMon > 0) {
		st.Big++
	}
	if du := st.Duel; du != nil && d.ID == du.OppID {
		switch {
		case du.Phase == "watch":
			g.fmOath(du, d)
			st.Duel = nil
		case du.Rules == "death":
			g.fmDuelEnd(true, "slain")
		default:
			if bd := g.localBurg(); bd != nil {
				g.AddRep(fmt.Sprintf("state:%d", bd.State), -25, " for killing in a lesser duel")
			}
			g.fmDuelEnd(true, "killed")
		}
	}
	if d.Has("fm_spouse") && st.Spouse != nil && st.Spouse.ID == d.ID {
		st.Spouse.Gone = "dead"
		g.Msg(fmt.Sprintf("{{R|%s, your spouse, is dead.}}", d.Name))
		g.deed("death", fmt.Sprintf("%s lost their spouse %s.", g.P.Name, d.Name), 0, 0)
		g.fmLog(d.Name + " died")
	}
	if d.Has("fm_child") {
		if k := g.fmKidByName(d.Name); k != nil {
			k.Fate = "dead"
			g.Msg(fmt.Sprintf("{{R|%s, your %s, is dead.}}", d.Name, fmKid(k)))
			g.deed("death", fmt.Sprintf("%s lost their %s %s.", g.P.Name, fmKid(k), d.Name), 0, 0)
			g.fmLog(d.Name + " died")
		}
	}
}

// fmSpawnFighter places a hostile-once-challenged fighter near the player.
func (g *Game) fmSpawnFighter(name, species string, lvl int) *Actor {
	r := NewRng(g.Seed + ":fm:fighter:" + name)
	a := g.Spawn("bandit", maxi(1, lvl), r)
	a.Name, a.Proper, a.Title = name, true, "duellist"
	a.Faction, a.AI, a.F = "fmduel", "hostile", CYellow
	if species != "" {
		a.Species = species
	}
	a.Desc = fmt.Sprintf("%s, a fighter who has come to the lists.", name)
	a.Gold = r.Int(20, 80)
	x, y, ok := g.nearestFree(g.Cur, g.P.X+4, g.P.Y)
	if !ok {
		return nil
	}
	a.X, a.Y, a.HomeX, a.HomeY = x, y, x, y
	g.Cur.Actors = append(g.Cur.Actors, a)
	return a
}

// ---- challenges ----

func (g *Game) fmQueueChallenge(name, text, rules string, hand int) {
	st := g.fm()
	if st.Chal != nil {
		return
	}
	st.Chal = &fmChal{Name: name, Text: text, Rules: rules, Stake: "honor", Expires: g.Day() + 14, Hand: hand}
	g.Msg("{{R|A challenge has been sent to you. (F7, Duels)}}")
}

func (g *Game) fmChallengeCheck() {
	st := g.fm()
	day := g.Day()
	if st.Chal != nil || day < st.LastChal+20 || (thFame(g) < 15 && st.fameMax() < 20) || g.fmRoll("chal", day) >= 0.06 {
		return
	}
	st.LastChal = day
	var cu *CultureData
	if bd := g.localBurg(); bd != nil {
		cu = g.W.cultByID[bd.Culture]
	}
	name := g.cultureName(cu, NewRng(fmt.Sprintf("%s:fm:challenger:%d", g.Seed, day)))
	rules := []string{"blood", "yield", "yield", "death"}[int(g.fmH("cr", day)%4)]
	text := fmt.Sprintf(Pick(NewRng(fmt.Sprint(g.fmH("ci", day))), fmInsults), name)
	g.fmQueueChallenge(name, text, rules, 0)
	if st.Chal != nil {
		st.Chal.Stake = []string{"honor", "gold", "rep"}[int(g.fmH("cs", day)%3)]
		st.Chal.Amt = 100
	}
}

func (g *Game) fmAcceptChallenge() {
	st := g.fm()
	c := st.Chal
	if c == nil {
		return
	}
	if g.Cur.Kind != "town" || st.Duel != nil {
		g.Msg("Meet them in a town square, with no duel under way.")
		return
	}
	rules := c.Rules
	if mx := g.fmRuleMax(); fmRuleSev[rules] > fmRuleSev[mx] {
		rules = mx
	}
	stake, amt := c.Stake, c.Amt
	if stake == "gold" && !g.cuCanSpend(amt) {
		stake, amt = "honor", 0
	}
	opp := g.fmSpawnFighter(c.Name, "human", g.P.Level+int(g.fmH("cl", c.Name)%3))
	if opp == nil {
		g.Msg("There is no room to fight here.")
		return
	}
	opp.Gold = maxi(opp.Gold, amt)
	st.Chal = nil
	g.fmDuelStart(opp, rules, stake, amt, nil, false, c.Hand)
	g.Mode = "main"
}

func (g *Game) fmRefuseChallenge() {
	st := g.fm()
	if st.Chal == nil {
		return
	}
	g.fmFame(g.fmRealm(), -3)
	if bd := g.localBurg(); bd != nil {
		g.AddRep(fmt.Sprintf("state:%d", bd.State), -10, " for refusing a challenge")
	}
	st.Chal = nil
	g.Msg("You decline. The tale of it will not be kind.")
}

// ---- tournaments ----

func (g *Game) fmTournToday() (bd *BurgData, name string, tier int, ok bool) {
	bd = g.localBurg()
	if bd == nil || g.Cur.Kind != "town" {
		return
	}
	tier = g.W.cityTier[bd.ID]
	if tier < 1 {
		return
	}
	d := g.Day()
	want := 6 + int(g.fmH("tday", bd.ID, d/SeasonDays)%16)
	if d%SeasonDays != want {
		return
	}
	return bd, fmTournName[(d/SeasonDays)%4], tier, true
}

func (g *Game) fmTournOpts(a *Actor) []Option {
	if a.Role != "guard" && a.Role != "arena" && a.Role != "notable" {
		return nil
	}
	bd, name, tier, ok := g.fmTournToday()
	if !ok || g.fml().Tourn != nil || g.fm().Duel != nil {
		return nil
	}
	key := fmt.Sprintf("%d:%d", bd.ID, g.Day()/SeasonDays)
	if g.fml().Done[key] {
		return nil
	}
	fee := 20 * (tier + 1)
	return []Option{{fmt.Sprintf("{{O|Enter %s.}} {{K|(%d bouts, entry %s)}}", name, 1+tier, g.fmCoin(fee)), func() {
		if !g.cuSpend(fee) {
			g.talkMenu("\"The entry fee is " + g.fmCoin(fee) + ". Come back with it.\"")
			return
		}
		g.fml().Done[key] = true
		g.fmTournStart(bd, name, tier)
	}}}
}

func (g *Game) fmTournStart(bd *BurgData, name string, tier int) {
	rounds := 1 + tier // tier 1: 4 fighters, 2 rounds; tier 2: 8 fighters, 3 rounds
	if rounds > 3 {
		rounds = 3
	}
	var pool []PersonData
	for _, p := range g.W.People {
		if p.Alive && p.Martial > 0 && (p.Burg == bd.ID || p.State == bd.State) {
			pool = append(pool, p)
		}
	}
	sort.SliceStable(pool, func(i, j int) bool {
		if pool[i].Martial != pool[j].Martial {
			return pool[i].Martial > pool[j].Martial
		}
		return pool[i].ID < pool[j].ID
	})
	n := 1<<uint(rounds) - 1
	if len(pool) > n {
		pool = pool[:n]
	}
	var fs []fmFighter
	for _, p := range pool {
		fs = append(fs, fmFighter{Name: p.Name, Species: p.Species, Lvl: maxi(1, g.P.Level-1+p.Martial/8)})
	}
	cu := g.W.cultByID[bd.Culture]
	r := NewRng(fmt.Sprintf("%s:fm:tourn:%d:%d", g.Seed, bd.ID, g.Day()/SeasonDays))
	for len(fs) < n {
		fs = append(fs, fmFighter{Name: g.cultureName(cu, r), Species: "human", Lvl: maxi(1, g.P.Level-1+r.Int(0, 2))})
	}
	sort.SliceStable(fs, func(i, j int) bool { return fs[i].Lvl < fs[j].Lvl })
	foes := make([]fmFighter, 0, rounds)
	for i := 0; i < rounds; i++ {
		foes = append(foes, fs[(i+1)*len(fs)/rounds-1])
	}
	tn := &fmTourn{Burg: bd.ID, ZoneKey: g.Cur.Key, Day: g.Day(), Foes: foes, Tier: tier, Name: name, Purse: 250*tier + 120*rounds, Wait: g.Turn + 5}
	g.fml().Tourn = tn
	g.Mode, g.Talk = "main", nil
	g.Msg(fmt.Sprintf("{{O|You enter %s of %s: %d fighters, %d bouts. First opponent: %s.}}", name, bd.Name, n+1, rounds, foes[0].Name))
}

func (g *Game) fmTournTick() {
	tn := g.fml().Tourn
	if tn == nil {
		return
	}
	if g.Cur.Key != tn.ZoneKey || g.Day() != tn.Day {
		g.fml().Tourn = nil
		if st := g.fm(); st.Duel != nil && st.Duel.Tourney {
			g.fmDuelEnd(false, "withdrawn")
		}
		g.Msg("{{K|You leave the lists; the tournament goes on without you.}}")
		return
	}
	if tn.Wait > 0 && g.Turn >= tn.Wait && g.fm().Duel == nil {
		tn.Wait = 0
		f := tn.Foes[tn.Round]
		opp := g.fmSpawnFighter(f.Name, f.Species, f.Lvl)
		if opp == nil {
			tn.Wait = g.Turn + 5
			return
		}
		rules := "yield"
		if tn.Tier == 1 && tn.Round < len(tn.Foes)-1 {
			rules = "blood"
		}
		g.Msg(fmt.Sprintf("{{O|Bout %d/%d: %s steps into the ring.}}", tn.Round+1, len(tn.Foes), f.Name))
		g.fmDuelStart(opp, rules, "honor", 0, nil, true, 0)
	}
}

func (g *Game) fmTournAfter(win bool) {
	st := g.fm()
	tn := g.fml().Tourn
	if tn == nil {
		return
	}
	if !win {
		g.fml().Tourn = nil
		g.Msg(fmt.Sprintf("{{R|You are knocked out of %s in bout %d.}}", tn.Name, tn.Round+1))
		return
	}
	tn.Round++
	if tn.Round >= len(tn.Foes) {
		g.fml().Tourn = nil
		st.Champs++
		st.Tourneys++
		g.cuEarn(tn.Purse)
		g.fmFame(g.fmRealm(), 8)
		bd := g.localBurg()
		if bd != nil {
			g.AddRep(fmt.Sprintf("state:%d", bd.State), 40, " for winning the tournament")
		}
		town := g.Cur.Name
		if bd != nil {
			town = bd.Name
		}
		g.gainXP(60)
		g.Msg(fmt.Sprintf("{{Y|You win %s! The purse is %s.}}", tn.Name, g.fmCoin(tn.Purse)))
		g.deed("event", fmt.Sprintf("%s won %s in %s.", g.P.Name, tn.Name, town), 0, 0)
		g.fmLog("Won " + tn.Name)
		return
	}
	g.P.HP = mini(g.P.MaxHP, g.P.HP+g.P.MaxHP/3)
	tn.Wait = g.Turn + 15
	g.Msg(fmt.Sprintf("{{G|Through to the next round. Next: %s. You catch your breath.}}", tn.Foes[tn.Round].Name))
}

// ---- talk hook ----

func fmTalk(g *Game, a *Actor) []Option {
	if g.Cur == nil {
		return nil
	}
	switch {
	case a.Has("fm_spouse"):
		return g.fmSpouseOpts(a)
	case a.Has("fm_child"):
		return g.fmChildOpts(a)
	}
	var out []Option
	if a.Role == "bard" {
		out = append(out, g.fmBardOpts(a)...)
	}
	out = append(out, g.fmWeddingOpts(a)...)
	out = append(out, g.fmHealerOpts(a)...)
	out = append(out, g.fmTournOpts(a)...)
	out = append(out, g.fmCourtOpts(a)...)
	out = append(out, g.fmDuelOpts(a)...)
	return out
}

// ---- the F7 screen ----

type fmAct struct {
	label string
	do    func()
}

var fmTabNames = []string{"Family", "Titles", "Songs", "Duels"}

func fmKeys(g *Game, k string) {
	lines, acts := g.fmTabContent(70)
	_ = lines
	switch k {
	case "Escape":
		g.Mode = "main"
	case "Tab", "ArrowRight":
		g.Tab, g.Sel = (g.Tab+1)%len(fmTabNames), 0
	case "ArrowLeft":
		g.Tab, g.Sel = (g.Tab+len(fmTabNames)-1)%len(fmTabNames), 0
	case "ArrowUp":
		g.Sel = maxi(0, g.Sel-1)
	case "ArrowDown":
		g.Sel = maxi(0, mini(len(acts)-1, g.Sel+1))
	case "Enter", " ":
		if g.Sel < len(acts) {
			acts[g.Sel].do()
		}
	}
}

func (g *Game) fmDraw() {
	s := g.sc
	x, y, w, h := g.panel("Family & Fame")
	tx := x
	for i, t := range fmTabNames {
		c := "K"
		if g.Tab == i {
			c = "Y"
		}
		tx = s.TextB(tx, y, fmt.Sprintf("{{%s|[ %s ]}} ", c, t), CGray, CBgDark, x+w)
	}
	lines, acts := g.fmTabContent(w)
	if len(acts) > 8 {
		acts = acts[:8]
	}
	maxInfo := h - 4 - len(acts)
	row := y + 2
	for i, l := range lines {
		if i >= maxInfo {
			break
		}
		s.TextB(x, row, l, CGray, CBgDark, x+w)
		row++
	}
	ar := y + h - 1 - len(acts)
	for i, a := range acts {
		mark := "  "
		if i == g.Sel {
			mark = "{{Y|>}} "
		}
		s.TextB(x, ar+i, mark+a.label, CGray, CBgDark, x+w)
	}
	s.TextB(x, y+h-1, "{{K|Left/Right: tab   Up/Down: select   Enter: do   Esc: close}}", CGray, CBgDark, x+w)
}

func (g *Game) fmTabContent(w int) ([]string, []fmAct) {
	switch g.Tab {
	case 1:
		return g.fmTabTitles(w)
	case 2:
		return g.fmTabSongs(w)
	case 3:
		return g.fmTabDuels(w)
	}
	return g.fmTabFamily(w)
}

func fmWrapAll(w int, ss ...string) []string {
	var out []string
	for _, s := range ss {
		out = append(out, Wrap(s, w)...)
	}
	return out
}

func (g *Game) fmLoveWord(n int) string {
	switch {
	case n < 20:
		return "estranged"
	case n < 45:
		return "cool"
	case n < 70:
		return "warm"
	}
	return "devoted"
}

func (g *Game) fmTabFamily(w int) ([]string, []fmAct) {
	st := g.fm()
	var ls []string
	var acts []fmAct
	if sp := st.Spouse; sp != nil {
		where := "keeps house"
		a, z := g.fmSpouseActor()
		switch {
		case sp.Gone != "":
			where = map[string]string{"dead": "is dead", "left": "has left you", "lost": "was lost"}[sp.Gone]
		case sp.Away:
			where = "is held for ransom"
		case a != nil && a.AI == "follower":
			where = "travels with you"
		case z != nil:
			where = "waits at " + z.Name
		default:
			where = "is at home"
		}
		ls = append(ls, fmWrapAll(w, fmt.Sprintf("{{M|Spouse:}} %s (%s) %s. Bond: %s (%d). Married %d days.", sp.Name, sp.Species, where, g.fmLoveWord(sp.Love), sp.Love, g.Day()-sp.Wed))...)
		if sp.Gone == "" && !sp.Away {
			if a != nil && a.AI != "follower" {
				acts = append(acts, fmAct{"Ask " + givenOf(sp.Name) + " to travel with you (be beside them)", func() {
					if a != nil && g.Cur == z {
						g.fmSpouseFollow(a)
					} else {
						g.Msg("Go to them first.")
					}
				}})
			}
			if g.homeHere() != nil {
				acts = append(acts, fmAct{"Have " + givenOf(sp.Name) + " keep this house", g.fmSpouseKeepHouse})
			}
		}
	} else {
		ls = append(ls, "{{K|Unwed. Talk to someone in a town and choose Court them.}}")
	}
	for _, c := range st.Courts {
		if c.Stage < 3 && (c.Planet == g.W.uniIdx) {
			ls = append(ls, fmWrapAll(w, fmt.Sprintf("{{c|Courting:}} %s, %s, affection %d%s.", c.Name, fmStageName(c.Stage), c.Aff, map[bool]string{true: ", rival " + c.Rival, false: ""}[c.Rival != "" && !c.RivalBeaten]))...)
		}
	}
	if len(st.Kids) > 0 {
		ls = append(ls, "", "{{M|Children:}}")
		for _, k := range st.Kids {
			age := fmAgeYears(g, k)
			fate := ""
			switch k.Fate {
			case "comp":
				fate = ", travels with you"
			case "away":
				fate = ", making a name abroad"
			case "kidnapped":
				fate = ", {{R|taken}}"
			case "lost":
				fate = ", lost"
			case "dead":
				fate = ", dead"
			}
			if k.Sick {
				fate += ", {{R|sick}}"
			}
			ls = append(ls, fmWrapAll(w, fmt.Sprintf("  %s, %s, %.1f years, %s (%s)%s.", k.Name, fmKid(k), age, fmStage(age), k.Calling, fate))...)
		}
	}
	if r := st.Rescue; r != nil {
		ls = append(ls, "", fmt.Sprintf("{{R|%s is held by %s. Ransom %s, or rescue them in the wilds, within %d days.}}", r.Name, r.Kidnapper, g.fmCoin(r.Ransom), maxi(0, r.Deadline-g.Day())))
		acts = append(acts, fmAct{"Pay the ransom (" + g.fmCoin(r.Ransom) + ")", g.fmPayRansom})
	}
	if len(st.Log) > 0 {
		ls = append(ls, "", "{{K|Recent:}}")
		from := maxi(0, len(st.Log)-4)
		for _, l := range st.Log[from:] {
			ls = append(ls, "{{K|"+l+"}}")
		}
	}
	return ls, acts
}

func (g *Game) fmTabTitles(w int) ([]string, []fmAct) {
	st := g.fm()
	var ls []string
	var acts []fmAct
	worn := "none"
	if t := fmTitleByID[st.Worn]; t != nil {
		worn = fmTitleName(t, g.P.Name)
	}
	ls = append(ls, fmWrapAll(w, "Worn: {{Y|"+worn+"}}.  Fame by realm: "+g.fmFameLine())...)
	ls = append(ls, "")
	for i := range fmTitles {
		t := &fmTitles[i]
		if st.Titles[t.ID] > 0 {
			ls = append(ls, fmWrapAll(w, fmt.Sprintf("{{G|%s}}: %s ({{K|%s}})", t.Name, t.Desc, t.Perk))...)
			t := t
			lab := "Wear " + t.Name
			if st.Worn == t.ID {
				lab = "Stop wearing " + t.Name
			}
			acts = append(acts, fmAct{lab, func() {
				if st.Worn == t.ID {
					st.Worn = ""
				} else {
					st.Worn = t.ID
				}
			}})
		} else {
			ls = append(ls, fmWrapAll(w, fmt.Sprintf("{{K}}%s: %s", t.Name, t.Desc))...)
		}
	}
	return ls, acts
}

func (g *Game) fmFameLine() string {
	st := g.fm()
	var ids []int
	for id, v := range st.Fame {
		if v != 0 {
			ids = append(ids, id)
		}
	}
	sort.Ints(ids)
	var parts []string
	for _, id := range ids {
		parts = append(parts, fmt.Sprintf("%s %d", g.realmName(id), st.Fame[id]))
	}
	if len(parts) == 0 {
		return fmt.Sprintf("none yet (renown %d)", thFame(g))
	}
	return strings.Join(parts, ", ") + fmt.Sprintf(" (renown %d)", thFame(g))
}

func (g *Game) fmTabSongs(w int) ([]string, []fmAct) {
	st := g.fm()
	var ls []string
	ls = append(ls, fmWrapAll(w, fmt.Sprintf("Bards sing your major deeds in towns after some days; tips speed and grow a song. Tips given: %d. Songs sung: %d of %d.", st.Tips, st.sungCount(), len(st.Songs)))...)
	ls = append(ls, "")
	from := maxi(0, len(st.Songs)-7)
	bd := g.localBurg()
	for i := len(st.Songs) - 1; i >= from; i-- {
		s := st.Songs[i]
		txt := s.Text
		if len(txt) > 70 {
			txt = txt[:70] + "..."
		}
		known := ""
		if bd != nil {
			if g.Day() >= s.Day+g.fmSongDelay(s, bd) {
				known = " {{G|(known here)}}"
			} else {
				known = fmt.Sprintf(" {{K|(reaches here in %d days)}}", s.Day+g.fmSongDelay(s, bd)-g.Day())
			}
		}
		ls = append(ls, fmWrapAll(w, fmt.Sprintf("%s [boost %d, sung %d]%s", txt, s.Boost, s.Sung, known))...)
	}
	if len(st.Songs) == 0 {
		ls = append(ls, "{{K|No songs yet. Do something worth singing.}}")
	}
	return ls, nil
}

func (g *Game) fmTabDuels(w int) ([]string, []fmAct) {
	st := g.fm()
	var ls []string
	var acts []fmAct
	ls = append(ls, fmWrapAll(w, fmt.Sprintf("Duels won %d, lost %d; spared %d; tournaments won %d; oaths broken %d.", st.DuelW, st.DuelL, st.Spared, st.Champs, st.Oath))...)
	ls = append(ls, fmWrapAll(w, fmt.Sprintf("Custom here: duels up to %s.", fmRuleName[g.fmRuleMax()]))...)
	if bd, name, _, ok := g.fmTournToday(); ok {
		ls = append(ls, fmWrapAll(w, fmt.Sprintf("{{O|%s are held in %s today.}} Speak to a guard or notable to enter.", name, bd.Name))...)
	} else if bd := g.localBurg(); bd != nil && g.W.cityTier[bd.ID] >= 1 {
		d := g.Day()
		want := 6 + int(g.fmH("tday", bd.ID, d/SeasonDays)%16)
		next := d - d%SeasonDays + want
		if next < d {
			next = d - d%SeasonDays + SeasonDays + 6 + int(g.fmH("tday", bd.ID, d/SeasonDays+1)%16)
		}
		ls = append(ls, fmt.Sprintf("{{K}}The next tournament here is in %d days.", next-d))
	}
	if d := st.Duel; d != nil {
		ls = append(ls, "", fmt.Sprintf("{{R|In a duel with %s, %s.}}", d.OppName, fmRuleName[d.Rules]))
	}
	if c := st.Chal; c != nil {
		ls = append(ls, "", "{{R|Challenge:}}")
		ls = append(ls, fmWrapAll(w, c.Text+fmt.Sprintf(" Terms: %s, stake %s. Answer within %d days.", fmRuleName[c.Rules], c.Stake, maxi(0, c.Expires-g.Day())))...)
		acts = append(acts, fmAct{"Accept the challenge (in a town square)", g.fmAcceptChallenge})
		acts = append(acts, fmAct{"Decline (costs standing)", g.fmRefuseChallenge})
	}
	return ls, acts
}
