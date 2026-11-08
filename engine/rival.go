package engine

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// Rival adventurers (prefix rv). 1-3 rival wanderers per planet, drawn from the world's notables
// (hero/general/magician/merchant/prophet) or generated with the culture's names. They live on the
// day clock: they travel between towns, hunt monsters, clear dungeons, level up, join orders and
// sects, fight each other and sometimes die; a replacement arrives after a while. They race the
// player for accepted bounties and relics learned as secrets (pure hash of seed + day, never g.R),
// can be met in towns, on roads and in dungeons, and a rivalry meter (-100..100) sets their mood
// (friendly, wary, hostile). Hostile rivals ambush with their gear. State: g.ext("rv"),
// planet-local ("rv" in unLocalKeys). Exports: rvDossier (Hunters screen), rvNews (wsNewsLines).

type rvRival struct {
	Slot    int      `json:"sl"`
	Gen     int      `json:"gn"`
	PID     int      `json:"pid,omitempty"`
	Name    string   `json:"n"`
	Title   string   `json:"ti"`
	Epithet string   `json:"ep,omitempty"`
	Species string   `json:"sp"`
	Calling string   `json:"ca"`
	Home    int      `json:"hm"`
	Level   int      `json:"lv"`
	XP      int      `json:"xp"`
	Gold    int      `json:"g"`
	Pos     int      `json:"pos"`
	Act     string   `json:"act"` // idle travel delve hunt ally
	From    int      `json:"fr"`
	Dest    int      `json:"de"`
	Depart  int      `json:"dp"`
	Arrive  int      `json:"ar"`
	End     int      `json:"end"`
	Wait    int      `json:"wt"`
	ZX      int      `json:"zx"`
	ZY      int      `json:"zy"`
	Target  string   `json:"tg,omitempty"`
	Rel     int      `json:"rel"`
	Group   string   `json:"grp,omitempty"`
	Wins    int      `json:"wins,omitempty"` // prizes taken from the player
	Kills   int      `json:"kills,omitempty"`
	Clears  int      `json:"clr,omitempty"`
	Relics  []int    `json:"rl,omitempty"`
	Alive   bool     `json:"al"`
	Died    int      `json:"dd,omitempty"`
	Killer  string   `json:"kl,omitempty"`
	Spawn   int      `json:"spn,omitempty"` // day the replacement arrives
	Bribed  int      `json:"bb,omitempty"`  // peace bought until this day
	Allied  bool     `json:"ally,omitempty"`
	AllyDay int      `json:"ad,omitempty"`
	AllyIn  bool     `json:"ai,omitempty"`
	Social  int      `json:"so,omitempty"` // last day of a news swap or meal (+1)
	Met     bool     `json:"met,omitempty"`
	Feud    int      `json:"fd,omitempty"` // slot+1 of a rival rival
	Notes   []string `json:"-"`
}

type rvLine struct {
	Day int    `json:"d"`
	T   string `json:"t"`
}

type rvState struct {
	Init    bool            `json:"init"`
	Day     int             `json:"day"`
	R       []*rvRival      `json:"r"`
	Used    map[int]bool    `json:"used,omitempty"`
	Act     map[int]string  `json:"act,omitempty"`
	Held    map[string]int  `json:"held,omitempty"`
	Cleared map[string]int  `json:"clr,omitempty"`
	Seen    map[string]bool `json:"seen,omitempty"`
	Log     []rvLine        `json:"log,omitempty"`
}

func init() {
	unLocalKeys = append(unLocalKeys, "rv")
	turnHooks = append(turnHooks, rvTurn)
	zoneHooks = append(zoneHooks, rvZone)
	killHooks = append(killHooks, rvKill)
	enterHooks = append(enterHooks, rvEnter)
	talkHooks = append(talkHooks, rvTalk)
	sideHooks = append(sideHooks, rvSide)
}

func (g *Game) rv() *rvState {
	st := g.ext("rv", func() any { return &rvState{} }).(*rvState)
	if st.Used == nil {
		st.Used = map[int]bool{}
	}
	if st.Act == nil {
		st.Act = map[int]string{}
	}
	if st.Held == nil {
		st.Held = map[string]int{}
	}
	if st.Cleared == nil {
		st.Cleared = map[string]int{}
	}
	if st.Seen == nil {
		st.Seen = map[string]bool{}
	}
	return st
}

// ---- helpers ----

func (g *Game) rvHash(parts ...any) uint64 {
	return HashStr(g.W.Seed + ":rv:" + fmt.Sprint(parts...))
}

// rvF: a stable float in [0,1) from the world seed and the parts.
func (g *Game) rvF(parts ...any) float64 { return float64(g.rvHash(parts...)%100000) / 100000 }

func (g *Game) rvToday() int { return DayOf(g.Turn) }

func (g *Game) rvLog(d int, text string) {
	st := g.rv()
	st.Log = append(st.Log, rvLine{d, text})
	if len(st.Log) > 24 {
		st.Log = st.Log[len(st.Log)-24:]
	}
}

func rvMood(rv *rvRival) string {
	switch {
	case rv.Rel >= 25:
		return "friendly"
	case rv.Rel <= -25:
		return "hostile"
	}
	return "wary"
}

func rvMoodTag(rv *rvRival) string {
	switch rvMood(rv) {
	case "friendly":
		return "{{G|friendly}}"
	case "hostile":
		return "{{R|hostile}}"
	}
	return "{{Y|wary}}"
}

func (g *Game) rvMeter(rv *rvRival) string {
	n := (rv.Rel + 100) / 10 // 0..20
	n = clampi(n, 0, 20)
	return "[" + strings.Repeat("=", n) + strings.Repeat("-", 20-n) + "]"
}

func (rv *rvRival) hostileNow(d int) bool { return rvMood(rv) == "hostile" && rv.Bribed < d }

func (g *Game) rvName(rv *rvRival) string {
	if rv.Epithet != "" {
		return rv.Name + " " + rv.Epithet
	}
	return rv.Name
}

func (g *Game) rvBurg(id int) *BurgData {
	if b := g.W.burgByID[id]; b != nil {
		return b
	}
	if l := g.ws().burgList(g); len(l) > 0 {
		return l[0]
	}
	return nil
}

func (g *Game) rvOf(a *Actor) *rvRival {
	if a == nil {
		return nil
	}
	st := g.rv()
	k, ok := st.Act[a.ID]
	if !ok {
		return nil
	}
	f := strings.Split(k, ":")
	if len(f) != 2 {
		return nil
	}
	sl, _ := strconv.Atoi(f[0])
	gn, _ := strconv.Atoi(f[1])
	for _, rv := range st.R {
		if rv.Slot == sl && rv.Gen == gn {
			return rv
		}
	}
	return nil
}

func (g *Game) rvGain(rv *rvRival, xp int) {
	rv.XP += xp
	for rv.XP >= rv.Level*40 && rv.Level < 45 {
		rv.XP -= rv.Level * 40
		rv.Level++
	}
}

func rvDays(w *WorldData, a, b int) int { return 1 + wsCheb(w, a, b)/6 }

// rvPosAt: the parasang a rival is on at the start of day d.
func (g *Game) rvPosAt(rv *rvRival, d int) int {
	w := g.W
	switch rv.Act {
	case "travel", "delve", "hunt":
		if d >= rv.Arrive || rv.Arrive <= rv.Depart {
			return rv.Dest
		}
		t := float64(d-rv.Depart) / float64(rv.Arrive-rv.Depart)
		fx, fy := rv.From%w.W, rv.From/w.W
		tx, ty := rv.Dest%w.W, rv.Dest/w.W
		x := fx + int(float64(tx-fx)*t+0.5)
		y := fy + int(float64(ty-fy)*t+0.5)
		if !w.In(x, y) {
			return rv.From
		}
		return w.Idx(x, y)
	}
	return rv.Pos
}

// ---- generation ----

func (g *Game) rvEnsure() {
	st := g.rv()
	if st.Init || g.W == nil {
		return
	}
	st.Init = true
	st.Day = g.rvToday()
	if len(g.ws().burgList(g)) == 0 {
		return
	}
	n := 1 + int(HashStr(g.W.Seed+":rv:n")%3)
	for slot := 0; slot < n; slot++ {
		st.R = append(st.R, g.rvGen(slot, 0, st.Day, 0))
	}
}

var rvPersonRoles = map[string]bool{"hero": true, "general": true, "magician": true, "merchant": true, "prophet": true}

func (g *Game) rvGen(slot, gen, day, baseLevel int) *rvRival {
	w := g.W
	st := g.rv()
	bl := g.ws().burgList(g)
	rv := &rvRival{Slot: slot, Gen: gen, Alive: true, Act: "idle", Rel: int(g.rvHash("rel", slot, gen)%41) - 20}
	var best *PersonData
	var bh uint64
	if g.rvF("usep", slot, gen) < 0.7 {
		for i := range w.People {
			p := &w.People[i]
			if !p.Alive || !rvPersonRoles[p.Role] || st.Used[p.ID] || p.Burg <= 0 {
				continue
			}
			bd := w.burgByID[p.Burg]
			if bd == nil || !bd.Alive || bd.Layer != "surface" || bd.P < 0 || bd.P >= len(w.Biome) || w.IsWater(bd.P) || bhCriminal(p) {
				continue
			}
			if h := g.rvHash("pick", slot, gen, p.ID); best == nil || h < bh {
				best, bh = p, h
			}
		}
	}
	var bd *BurgData
	if best != nil {
		st.Used[best.ID] = true
		bd = w.burgByID[best.Burg]
		rv.PID, rv.Name, rv.Species = best.ID, best.Name, best.Species
		cs := rvRoleCalling[best.Role]
		rv.Calling = cs[int(g.rvHash("call", slot, gen)%uint64(len(cs)))]
		rv.Level = g.zoneLevel(bd.P%w.W, bd.P/w.W, 0) + 1 + best.Martial/12
	} else {
		bd = bl[int(g.rvHash("burg", slot, gen)%uint64(len(bl)))]
		cu := w.cultByID[bd.Culture]
		rv.Name = g.cultureName(cu, NewRng(w.Seed+":rvname:"+strconv.Itoa(slot)+":"+strconv.Itoa(gen)))
		rv.Species = g.speciesOf(cu).ID
		rv.Calling = rvCallings[int(g.rvHash("call", slot, gen)%uint64(len(rvCallings)))]
		rv.Level = g.zoneLevel(bd.P%w.W, bd.P/w.W, 0) + 1 + int(g.rvHash("lv", slot, gen)%3)
	}
	if baseLevel > 0 {
		rv.Level = baseLevel
	}
	rv.Level = clampi(rv.Level, 2, 40)
	rv.Home, rv.Pos = bd.ID, bd.P
	rv.Title = g.Sk.Calling(rv.Calling)
	if best != nil && best.Title != "" && g.rvF("tt", slot, gen) < 0.4 {
		rv.Title = strings.ToUpper(best.Title[:1]) + best.Title[1:]
	}
	rv.Epithet = rvEpithets[int(g.rvHash("ep", slot, gen)%uint64(len(rvEpithets)))]
	rv.Gold = 40 + rv.Level*25
	rv.Wait = day + int(g.rvHash("w0", slot, gen)%3)
	if len(st.R) > 0 || slot > 0 {
		rv.Feud = 0
	}
	return rv
}

// ---- the daily sim ----

func (g *Game) rvAdvance() {
	if g.W == nil {
		return
	}
	g.rvEnsure()
	st := g.rv()
	today := g.rvToday()
	if today-st.Day > 40 {
		st.Day = today - 40
	}
	for st.Day < today {
		st.Day++
		g.rvDaily(st.Day)
	}
	if len(st.Seen) > 800 {
		st.Seen = map[string]bool{}
	}
}

func (g *Game) rvDaily(d int) {
	st := g.rv()
	for _, rv := range st.R {
		if !rv.Alive {
			if d >= rv.Spawn && rv.Spawn > 0 {
				g.rvReplace(rv, d)
			}
			continue
		}
		if rv.Allied || rv.Act == "ally" {
			continue
		}
		g.rvStep(rv, d)
		if d%6 == 0 && rv.Rel != 0 {
			if rv.Rel > 0 {
				rv.Rel--
			} else {
				rv.Rel++
			}
		}
	}
	g.rvDuels(d)
	g.rvRace(d)
}

func (g *Game) rvReplace(old *rvRival, d int) {
	st := g.rv()
	base := clampi(g.P.Level-1+int(g.rvHash("rb", old.Slot, old.Gen+1)%3), 2, 40)
	nr := g.rvGen(old.Slot, old.Gen+1, d, base)
	for i, r := range st.R {
		if r == old {
			st.R[i] = nr
		}
	}
	g.rvLog(d, fmt.Sprintf("A new wanderer, %s the %s, has taken up the road from %s.", g.rvName(nr), nr.Title, g.wsPlace(nr.Pos)))
}

// rvSites: dungeons and towers within radius of a parasang, away from towns.
type rvSite struct {
	P      int
	ZX, ZY int
	S      *Site
}

func (g *Game) rvSites(from, rad int) []rvSite {
	w := g.W
	cx, cy := from%w.W, from/w.W
	var out []rvSite
	for y := cy - rad; y <= cy+rad; y++ {
		for x := cx - rad; x <= cx+rad; x++ {
			if !w.In(x, y) {
				continue
			}
			p := w.Idx(x, y)
			if w.Burg[p] > 0 || w.IsWater(p) {
				continue
			}
			for zy := 0; zy < 3; zy++ {
				for zx := 0; zx < 3; zx++ {
					if s := g.siteAt(x, y, zx, zy); s != nil && (s.Kind == "dungeon" || s.Kind == "tower") && s.Depth > 0 {
						out = append(out, rvSite{p, zx, zy, s})
					}
				}
			}
		}
	}
	return out
}

func (g *Game) rvSiteLevel(s rvSite) int {
	return g.zoneLevel(s.P%g.W.W, s.P/g.W.W, 0) + s.S.Depth/2 + s.S.Bonus + 1
}

func (g *Game) rvStep(rv *rvRival, d int) {
	w := g.W
	st := g.rv()
	if rv.Wait > d {
		return
	}
	tag := fmt.Sprint(rv.Slot, ":", rv.Gen, ":", d)
	switch rv.Act {
	case "travel":
		if d >= rv.Arrive {
			rv.Pos, rv.Act = rv.Dest, "idle"
			rv.Wait = d + int(g.rvHash("rest", tag)%2)
		}
		return
	case "delve":
		if d < rv.End {
			return
		}
		g.rvResolveDelve(rv, d)
		return
	case "hunt":
		if d < rv.End {
			return
		}
		g.rvResolveHunt(rv, d)
		return
	case "hoard":
		if d < rv.End {
			return
		}
		g.rvResolveHoard(rv, d)
		return
	}
	u := int(g.rvHash("act", tag) % 100)
	// hostile rivals stalk a player who is close
	if rvMood(rv) == "hostile" && rv.Bribed < d && g.W.In(g.WX, g.WY) {
		pp := w.Idx(g.WX, g.WY)
		if wsCheb(w, rv.Pos, pp) <= 22 && wsCheb(w, rv.Pos, pp) > 0 && g.rvF("stalk", tag) < 0.4 {
			rv.Act, rv.From, rv.Dest, rv.Depart = "travel", rv.Pos, pp, d
			rv.Arrive = d + rvDays(w, rv.Pos, pp)
			rv.Target = "your trail"
			return
		}
	}
	switch {
	case u < 30:
		if !g.rvTravel(rv, d, tag) {
			rv.Wait = d + 1
		}
	case u < 62:
		var cs []rvSite
		for _, s := range g.rvSites(rv.Pos, 12) {
			if last, ok := st.Cleared[siteKey(s.P%w.W, s.P/w.W, s.ZX, s.ZY)]; ok && d-last < 25 {
				continue
			}
			if g.rvSiteLevel(s) > rv.Level+6 {
				continue
			}
			cs = append(cs, s)
		}
		if len(cs) == 0 {
			if !g.rvTravel(rv, d, tag) {
				rv.Wait = d + 1
			}
			return
		}
		s := cs[int(g.rvHash("site", tag)%uint64(len(cs)))]
		rv.Act, rv.From, rv.Dest, rv.Depart = "delve", rv.Pos, s.P, d
		rv.Arrive = d + rvDays(w, rv.Pos, s.P)
		rv.End = rv.Arrive + 1 + s.S.Depth/2
		rv.ZX, rv.ZY = s.ZX, s.ZY
		rv.Target = s.S.Name
		g.rvLog(d, fmt.Sprintf("%s set out for %s.", g.rvName(rv), s.S.Name))
	case u < 76:
		var cs []*MonsterData
		for i := range w.Monsters {
			m := &w.Monsters[i]
			if m.P < 0 || m.P >= len(w.State) || m.Layer != "surface" || m.Slain || g.Slain[m.ID] || wsCheb(w, rv.Pos, m.P) > 22 || 2+m.Power/8 > rv.Level+5 {
				continue
			}
			cs = append(cs, m)
		}
		if len(cs) == 0 {
			rv.Wait = d + 1
			return
		}
		m := cs[int(g.rvHash("mon", tag)%uint64(len(cs)))]
		rv.Act, rv.From, rv.Dest, rv.Depart = "hunt", rv.Pos, m.P, d
		rv.Arrive = d + rvDays(w, rv.Pos, m.P)
		rv.End = rv.Arrive + 1
		rv.Target = m.Name
		rv.ZX = m.ID
		g.rvLog(d, fmt.Sprintf("%s is hunting %s.", g.rvName(rv), m.Name))
	case u < 86:
		var cs []*bhBounty
		for _, b := range bhSt(g).Bounties {
			if b.Taken || b.Done || b.Dead || b.Lvl > rv.Level+3 || wsCheb(w, rv.Pos, w.Idx(clampi(b.WX, 0, w.W-1), clampi(b.WY, 0, w.H-1))) > 30 {
				continue
			}
			cs = append(cs, b)
		}
		if len(cs) == 0 {
			rv.Wait = d + 1
			return
		}
		sort.Slice(cs, func(i, j int) bool { return cs[i].ID < cs[j].ID })
		b := cs[int(g.rvHash("board", tag)%uint64(len(cs)))]
		b.Done, b.Dead = true, true
		rv.Gold += b.Reward
		g.rvGain(rv, 30+b.Reward/10)
		rv.Kills++
		rv.Wait = d + 2
		g.rvLog(d, fmt.Sprintf("%s collected the bounty on %s.", g.rvName(rv), b.Name))
	case u < 90:
		var hs []int
		for _, hp := range g.buDigSites() {
			if wsCheb(w, rv.Pos, hp) <= 28 {
				hs = append(hs, hp)
			}
		}
		if len(hs) > 0 {
			hp := hs[int(g.rvHash("hoard", tag)%uint64(len(hs)))]
			rv.Act, rv.From, rv.Dest, rv.Depart = "hoard", rv.Pos, hp, d
			rv.Arrive = d + rvDays(w, rv.Pos, hp)
			rv.End = rv.Arrive + 1
			rv.Target = "a buried hoard near " + g.wsPlace(hp)
			g.rvLog(d, fmt.Sprintf("%s is racing toward a buried hoard near %s.", g.rvName(rv), g.wsPlace(hp)))
			return
		}
		rv.Wait = d + 1
	case u < 95:
		if rv.Group == "" && rv.Level >= 4 {
			var names []string
			for _, o := range g.Orders() {
				if !o.Secret {
					names = append(names, o.Name)
				}
			}
			for _, s := range g.Sects() {
				names = append(names, s.Name)
			}
			if len(names) > 0 {
				rv.Group = names[int(g.rvHash("grp", rv.Slot, rv.Gen)%uint64(len(names)))]
				g.rvGain(rv, 25)
				g.rvLog(d, fmt.Sprintf("%s joined %s.", g.rvName(rv), rv.Group))
				return
			}
		}
		rv.Wait = d + 1 + int(g.rvHash("rst", tag)%3)
	default:
		rv.Wait = d + 1 + int(g.rvHash("rst", tag)%3)
	}
}

func (g *Game) rvTravel(rv *rvRival, d int, tag string) bool {
	w := g.W
	var cs []*BurgData
	for _, b := range g.ws().burgList(g) {
		if dd := wsCheb(w, rv.Pos, b.P); dd >= 3 && dd <= 34 {
			cs = append(cs, b)
		}
	}
	if len(cs) == 0 {
		return false
	}
	b := cs[int(g.rvHash("dest", tag)%uint64(len(cs)))]
	rv.Act, rv.From, rv.Dest, rv.Depart = "travel", rv.Pos, b.P, d
	rv.Arrive = d + rvDays(w, rv.Pos, b.P)
	rv.Target = b.Name
	return true
}

func (g *Game) rvResolveDelve(rv *rvRival, d int) {
	w := g.W
	st := g.rv()
	s := g.siteAt(rv.Dest%w.W, rv.Dest/w.W, rv.ZX, rv.ZY)
	rv.Act, rv.Pos = "idle", rv.Dest
	if s == nil {
		rv.Wait = d + 1
		return
	}
	lvl := g.zoneLevel(rv.Dest%w.W, rv.Dest/w.W, 0) + s.Depth/2 + s.Bonus + 1
	tag := fmt.Sprint(rv.Slot, ":", rv.Gen, ":", d)
	prob := rvClampf(0.6+float64(rv.Level-lvl)*0.08, 0.1, 0.95)
	if g.rvF("delve", tag) < prob {
		rv.Clears++
		g.rvGain(rv, 40+lvl*12)
		rv.Gold += 20 + lvl*15
		st.Cleared[siteKey(rv.Dest%w.W, rv.Dest/w.W, rv.ZX, rv.ZY)] = d
		rv.Wait = d + 1 + int(g.rvHash("rs", tag)%2)
		g.rvLog(d, fmt.Sprintf("%s cleared %s (level %d).", g.rvName(rv), s.Name, rv.Level))
		return
	}
	dieP := rvClampf(0.25+float64(lvl-rv.Level)*0.06, 0.05, 0.8)
	if g.rvF("die", tag) < dieP {
		g.rvDie(rv, d, "slain by "+rvMisfortune[int(g.rvHash("mf", tag)%uint64(len(rvMisfortune)))]+" in "+s.Name, false)
		return
	}
	g.rvGain(rv, 10)
	rv.Wait = d + 3
	g.rvLog(d, fmt.Sprintf("%s fled %s, wounded.", g.rvName(rv), s.Name))
}

func (g *Game) rvResolveHoard(rv *rvRival, d int) {
	rv.Act, rv.Pos = "idle", rv.Dest
	if g.buRivalLoot(rv.Dest) {
		rv.Clears++
		rv.Gold += 120 + rv.Level*25
		g.rvGain(rv, 45+rv.Level*10)
		rv.Wait = d + 2
		g.rvLog(d, fmt.Sprintf("%s dug up the buried hoard near %s before you could.", g.rvName(rv), g.wsPlace(rv.Dest)))
		return
	}
	rv.Wait = d + 1
}

func (g *Game) rvResolveHunt(rv *rvRival, d int) {
	w := g.W
	rv.Act, rv.Pos = "idle", rv.Dest
	var m *MonsterData
	for i := range w.Monsters {
		if w.Monsters[i].ID == rv.ZX {
			m = &w.Monsters[i]
		}
	}
	rv.ZX = 0
	if m == nil {
		rv.Wait = d + 1
		return
	}
	lvl := 2 + m.Power/8
	tag := fmt.Sprint(rv.Slot, ":", rv.Gen, ":", d)
	if g.rvF("hunt", tag) < rvClampf(0.5+float64(rv.Level-lvl)*0.08, 0.1, 0.92) {
		rv.Kills++
		g.rvGain(rv, 30+lvl*14)
		rv.Gold += 30 + lvl*10
		rv.Wait = d + 2
		g.rvLog(d, fmt.Sprintf("%s hunted %s to ground (it still prowls; the beast's kin say otherwise).", g.rvName(rv), m.Name))
		return
	}
	if g.rvF("hdie", tag) < rvClampf(0.2+float64(lvl-rv.Level)*0.07, 0.04, 0.7) {
		g.rvDie(rv, d, "killed by "+m.Name, false)
		return
	}
	rv.Wait = d + 3
	g.rvLog(d, fmt.Sprintf("%s was driven off by %s.", g.rvName(rv), m.Name))
}

func rvClampf(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// rvDie: the rival dies away from or in front of the player; a replacement is scheduled.
func (g *Game) rvDie(rv *rvRival, d int, how string, byPlayer bool) {
	rv.Alive = false
	rv.Died = d
	rv.Killer = how
	rv.Act = "idle"
	rv.Allied = false
	rv.Spawn = d + 18 + int(g.rvHash("repl", rv.Slot, rv.Gen)%18)
	if !byPlayer {
		for _, id := range rv.Relics {
			g.Taken[id] = false
		}
		rv.Relics = nil
		g.rvLog(d, fmt.Sprintf("{{R|%s is dead:}} %s.", g.rvName(rv), how))
		if g.Cur != nil && g.rvKnows(rv) {
			g.Msg(fmt.Sprintf("{{y|Word reaches you that your rival %s is dead: %s.}}", g.rvName(rv), how))
		}
	}
}

func (g *Game) rvKnows(rv *rvRival) bool { return rv.Met }

// rvDuels: rivals on the same parasang may fight each other.
func (g *Game) rvDuels(d int) {
	st := g.rv()
	for i := 0; i < len(st.R); i++ {
		for j := i + 1; j < len(st.R); j++ {
			a, b := st.R[i], st.R[j]
			if !a.Alive || !b.Alive || a.Allied || b.Allied || a.Act != "idle" || b.Act != "idle" || a.Pos != b.Pos {
				continue
			}
			if g.rvF("duel", a.Slot, b.Slot, d) >= 0.3 {
				continue
			}
			sa := float64(a.Level)*10 + g.rvF("sa", a.Slot, b.Slot, d)*30
			sb := float64(b.Level)*10 + g.rvF("sb", a.Slot, b.Slot, d)*30
			win, lose := a, b
			if sb > sa {
				win, lose = b, a
			}
			g.rvGain(win, 20+lose.Level*6)
			win.Kills++
			if g.rvF("dk", a.Slot, b.Slot, d) < 0.3 {
				g.rvDie(lose, d, "cut down by "+g.rvName(win)+" in a duel at "+g.wsPlace(a.Pos), false)
			} else {
				lose.Wait = d + 5
				lose.XP = maxi(0, lose.XP-20)
				g.rvLog(d, fmt.Sprintf("%s bested %s in a duel at %s.", g.rvName(win), g.rvName(lose), g.wsPlace(a.Pos)))
			}
			win.Wait = maxi(win.Wait, d+2)
		}
	}
}

// ---- racing the player for prizes ----

type rvPrize struct {
	Key  string
	Name string
	P    int
	Lvl  int
	B    *bhBounty
	Art  *ArtifactData
}

func (g *Game) rvPrizes() []rvPrize {
	w := g.W
	var out []rvPrize
	for _, b := range bhActive(g) {
		if b.Dead || b.Done {
			continue
		}
		wx, wy := clampi(b.WX, 0, w.W-1), clampi(b.WY, 0, w.H-1)
		out = append(out, rvPrize{Key: "b" + strconv.Itoa(b.ID), Name: "the bounty on " + b.Name, P: w.Idx(wx, wy), Lvl: maxi(b.Lvl, g.zoneLevel(wx, wy, 0)+1), B: b})
	}
	for _, sc := range g.Secrets {
		if sc.Kind != "relic" {
			continue
		}
		var best *ArtifactData
		for i := range w.Artifacts {
			a := &w.Artifacts[i]
			if !g.Taken[a.ID] && a.P >= 0 && a.P < len(w.Biome) && strings.Contains(sc.Text, "relic "+a.Name) && (best == nil || len(a.Name) > len(best.Name)) {
				best = a
			}
		}
		if best != nil {
			out = append(out, rvPrize{Key: "r" + strconv.Itoa(best.ID), Name: "the relic " + best.Name, P: best.P, Lvl: g.zoneLevel(best.P%w.W, best.P/w.W, 0) + 3, Art: best})
		}
	}
	return out
}

func (g *Game) rvRace(d int) {
	st := g.rv()
	w := g.W
	here := -1
	if g.Cur != nil && w.In(g.Cur.WX, g.Cur.WY) {
		here = w.Idx(g.Cur.WX, g.Cur.WY)
	}
	for _, pz := range g.rvPrizes() {
		if _, ok := st.Held[pz.Key]; !ok {
			st.Held[pz.Key] = d
			continue
		}
		if pz.P == here {
			continue
		}
		age := d - st.Held[pz.Key]
		for _, rv := range st.R {
			if !rv.Alive || rv.Allied || rv.Bribed >= d || rv.Rel >= 55 {
				continue
			}
			if age < 2+wsCheb(w, rv.Pos, pz.P)/9 {
				continue
			}
			p := 0.05
			if rvMood(rv) == "hostile" {
				p += 0.04
			}
			p += rvClampf(float64(rv.Level-pz.Lvl)*0.01, -0.03, 0.06)
			if g.rvF("race", rv.Slot, rv.Gen, pz.Key, d) >= p {
				continue
			}
			g.rvTake(rv, pz, d)
			break
		}
	}
}

func (g *Game) rvTake(rv *rvRival, pz rvPrize, d int) {
	w := g.W
	rv.Wins++
	rv.Rel = clampi(rv.Rel-8, -100, 100)
	g.rvGain(rv, 40+pz.Lvl*8)
	nm := g.rvName(rv)
	if b := pz.B; b != nil {
		b.Done, b.Dead = true, true
		if b.Kind == "monster" {
			g.Slain[b.Mon] = true
		}
		rv.Gold += b.Reward
		for _, zn := range g.Zones {
			keep := zn.Actors[:0]
			for _, a := range zn.Actors {
				if a.AI != "follower" && (b.ActorID != 0 && a.ID == b.ActorID || b.Kind == "monster" && a.SimMon == b.Mon && b.Mon != 0) {
					continue
				}
				keep = append(keep, a)
			}
			zn.Actors = keep
		}
	}
	if a := pz.Art; a != nil {
		g.Taken[a.ID] = true
		rv.Relics = append(rv.Relics, a.ID)
		for _, c := range bhSt(g).Caches {
			if c.Art == a.ID && !c.Dug {
				c.Dug = true
			}
		}
		for _, zn := range g.Zones {
			keep := zn.Items[:0]
			for _, it := range zn.Items {
				if it.SimArt == a.ID {
					continue
				}
				keep = append(keep, it)
			}
			zn.Items = keep
		}
	}
	text := fmt.Sprintf("%s the %s took %s before you could.", nm, rv.Title, pz.Name)
	g.rvLog(d, text)
	rv.Met = true
	if g.Cur != nil {
		g.Msg("{{R|Rival: " + text + " Someone else has it now.}}")
		g.deed("rival", fmt.Sprintf("%s the %s lost %s to the rival %s.", g.P.Name, g.Sk.Calling(g.Calling), pz.Name, nm), 0, 0)
	}
	_ = w
}

// ---- actors ----

func (g *Game) rvAvatar(rv *rvRival) *Actor {
	w := g.W
	bd := g.rvBurg(rv.Home)
	cu := w.cultByID[bd.Culture]
	r := NewRng(w.Seed + ":rvmk:" + strconv.Itoa(rv.Slot) + ":" + strconv.Itoa(rv.Gen))
	a := g.makeNPC("guard", bd, cu, r)
	a.Species = nonEmpty(rv.Species, a.Species)
	a.Body = MakeBody(BodyPlanForSpecies(a.Species))
	tier := clampi(rv.Level/4+1, 1, 5)
	if p := a.PartFor("hand"); p != nil {
		p.Item = g.MakeWeapon(Pick(r, rvWeapon[rv.Calling]), tier, r)
	}
	if p := a.PartFor("body"); p != nil {
		p.Item = g.MakeArmor("body", tier, r)
	}
	if p := a.PartFor("head"); p != nil {
		p.Item = g.MakeArmor("head", maxi(0, tier-1), r)
	}
	if p := a.PartFor("feet"); p != nil {
		p.Item = g.MakeArmor("feet", maxi(0, tier-1), r)
	}
	a.Name, a.Proper, a.Role, a.Title = g.rvName(rv), true, "", rv.Title
	a.Faction, a.State = "rivals", 0
	a.Level = rv.Level
	a.MaxHP = 24 + rv.Level*5
	if rv.Calling == "warrior" {
		a.MaxHP += 12
	}
	a.HP = a.MaxHP
	a.Attr[STR] += rv.Level / 5
	a.Attr[AGI] += rv.Level / 6
	a.Attr[rvAttr[rv.Calling]] += 2 + rv.Level/4
	a.AVNat = rv.Level / 6
	a.XP = 40 + rv.Level*rv.Level*3
	a.Gold = rv.Gold
	a.SimPer = rv.PID
	a.F = CMagenta
	a.Inv = nil
	a.Inv = append(a.Inv, g.MakeTonic("heal", 1+rv.Level/6))
	for _, id := range rv.Relics {
		for i := range w.Artifacts {
			if w.Artifacts[i].ID == id {
				it := g.RelicItem(&w.Artifacts[i])
				a.Inv = append(a.Inv, it)
			}
		}
	}
	sp := ""
	if s := w.specByID[a.Species]; s != nil {
		sp = strings.ToLower(s.Name) + " "
	}
	a.Desc = fmt.Sprintf("%s, a %s%s of %s, wandering the same roads for the same prizes. Level %d. %s", g.rvName(rv), sp, strings.ToLower(rv.Title), bd.Name, rv.Level, rvGroupText(rv))
	a.Set("rvr")
	g.rv().Act[a.ID] = fmt.Sprint(rv.Slot, ":", rv.Gen)
	return a
}

func rvGroupText(rv *rvRival) string {
	if rv.Group == "" {
		return ""
	}
	return "Sworn to " + rv.Group + "."
}

// rvZone: a notable who wanders as a rival does not also stand in their old town.
func rvZone(g *Game, zn *Zone, _ *Rng) {
	if g.W == nil || zn.Z != 0 || zn.Burg <= 0 {
		return
	}
	g.rvEnsure()
	st := g.rv()
	if len(st.Used) == 0 {
		return
	}
	keep := zn.Actors[:0]
	for _, a := range zn.Actors {
		if a.SimPer > 0 && st.Used[a.SimPer] && !a.Has("rvr") {
			continue
		}
		keep = append(keep, a)
	}
	zn.Actors = keep
}

// rvHere: should this rival be physically in this zone right now?
func (g *Game) rvHere(rv *rvRival, zn *Zone, d int) bool {
	w := g.W
	if !rv.Alive || rv.Allied || !w.In(zn.WX, zn.WY) {
		return false
	}
	p := w.Idx(zn.WX, zn.WY)
	switch zn.Kind {
	case "town":
		return zn.Z == 0 && zn.Burg > 0 && rv.Act == "idle" && rv.Pos == p
	case "dungeon", "tower":
		if rv.Act != "delve" || rv.Dest != p || rv.ZX != zn.ZX || rv.ZY != zn.ZY || d < rv.Arrive || d >= rv.End {
			return false
		}
		s := g.siteAt(zn.WX, zn.WY, zn.ZX, zn.ZY)
		if s == nil || s.Depth <= 0 {
			return false
		}
		tgt := int(g.rvHash("lvl", rv.Slot, rv.Gen, rv.End) % uint64(s.Depth))
		if zn.Kind == "dungeon" {
			return zn.Z >= 1 && (zn.Z-1)%s.Depth == tgt
		}
		return (towerBase-zn.Z)%s.Depth == tgt
	case "surface":
		if zn.Z != 0 || w.Burg[p] != 0 {
			return false
		}
		q := g.rvPosAt(rv, d)
		if rvMood(rv) == "hostile" && rv.Bribed < d {
			return wsCheb(w, q, p) <= 1
		}
		return q == p
	}
	return false
}

func rvEnter(g *Game, zn *Zone) {
	if g.W == nil || !g.W.In(zn.WX, zn.WY) {
		return
	}
	g.rvAdvance()
	g.rvAllyEnter(zn)
	st := g.rv()
	d := g.rvToday()
	// stale rival actors leave
	present := map[int]bool{}
	keep := zn.Actors[:0]
	for _, a := range zn.Actors {
		if a.Has("rvr") && a.AI != "follower" {
			rv := g.rvOf(a)
			if rv == nil || !g.rvHere(rv, zn, d) {
				continue
			}
			present[rv.Slot] = true
		}
		keep = append(keep, a)
	}
	zn.Actors = keep
	for _, rv := range st.R {
		if present[rv.Slot] || !g.rvHere(rv, zn, d) {
			continue
		}
		if zn.Kind == "surface" {
			key := fmt.Sprint("road:", rv.Slot, ":", rv.Gen, ":", d)
			if st.Seen[key] {
				continue
			}
			st.Seen[key] = true
		}
		g.rvSpawn(zn, rv, d)
	}
	// a dungeon a rival has just emptied
	if zn.Kind == "dungeon" || zn.Kind == "tower" {
		key := siteKey(zn.WX, zn.WY, zn.ZX, zn.ZY)
		if last, ok := st.Cleared[key]; ok && d-last < 30 && !st.Seen["cl:"+zn.Key] {
			st.Seen["cl:"+zn.Key] = true
			n := 0
			out := zn.Actors[:0]
			for _, a := range zn.Actors {
				if a.AI == "hostile" && !a.Boss && !a.Has("rvr") && a.AI != "follower" && g.rvF("clr", zn.Key, a.ID) < 0.35 {
					n++
					continue
				}
				out = append(out, a)
			}
			zn.Actors = out
			if n > 0 {
				g.Msg("{{y|Fresh kills and trampled dust: someone has been through here ahead of you.}}")
			}
		}
	}
}

func (g *Game) rvSpawn(zn *Zone, rv *rvRival, d int) {
	a := g.rvAvatar(rv)
	r := NewRng(fmt.Sprint(g.W.Seed, ":rvsp:", rv.Slot, ":", rv.Gen, ":", zn.Key, ":", d))
	x, y, ok := zn.FreeSpot(r, func(x, y int) bool { return abs(x-g.P.X) > 5 || abs(y-g.P.Y) > 4 })
	if !ok {
		return
	}
	a.X, a.Y, a.HomeX, a.HomeY = x, y, x, y
	nm := g.rvName(rv)
	switch {
	case zn.Kind == "town":
		a.AI = "wander"
		g.Msg("{{Y|Your rival " + nm + " is in town.}}")
	case rv.hostileNow(d):
		a.AI, a.Angry = "hostile", true
		g.Msg("{{R|" + nm + " steps into your path.}} " + rvAmbush[int(g.rvHash("amb", rv.Slot, d)%uint64(len(rvAmbush)))])
	default:
		a.AI = "wander"
		if rvMood(rv) == "hostile" {
			g.Msg("{{Y|" + nm + " eyes you from a distance but has been paid to keep the peace.}}")
		} else {
			g.Msg("{{Y|You meet your rival " + nm + " on the way.}}")
		}
	}
	zn.Actors = append(zn.Actors, a)
	rv.Met = true
}

// ---- turn: duels and ally timeouts ----

func rvTurn(g *Game) {
	if g.W == nil || g.Cur == nil {
		return
	}
	for _, a := range g.Cur.Actors {
		if !a.Has("rvduel") || a.HP <= 0 || a.HP*10 > a.MaxHP*3 {
			continue
		}
		rv := g.rvOf(a)
		delete(a.Flags, "rvduel")
		a.AI, a.Angry = "still", false
		a.Set("rvyield")
		if rv != nil {
			rv.Rel = clampi(rv.Rel+10, -100, 100)
			g.Msg("{{G|" + a.Name + " lowers the blade: \"Enough. You have earned my respect.\"}}")
			g.deed("rival", fmt.Sprintf("%s the %s won a duel against the rival %s, who yielded.", g.P.Name, g.Sk.Calling(g.Calling), a.Name), 0, 0)
		}
	}
	if g.Turn%60 != 0 {
		return
	}
	g.rvAdvance()
	d := g.rvToday()
	st := g.rv()
	for _, rv := range st.R {
		if rv.Allied && d-rv.AllyDay >= 4 {
			g.rvAllyEnd(rv, "time")
		}
	}
	// stalkers who reach the player
	zn := g.Cur
	if zn.Kind == "surface" && zn.Z == 0 && g.W.In(zn.WX, zn.WY) {
		for _, rv := range st.R {
			if rv.Alive && rv.hostileNow(d) && g.rvHere(rv, zn, d) {
				key := fmt.Sprint("road:", rv.Slot, ":", rv.Gen, ":", d)
				if st.Seen[key] {
					continue
				}
				st.Seen[key] = true
				g.rvSpawn(zn, rv, d)
			}
		}
	}
}

// ---- death ----

func rvKill(g *Game, d, src *Actor) {
	if g.W == nil || !d.Has("rvr") {
		return
	}
	rv := g.rvOf(d)
	if rv == nil || !rv.Alive {
		return
	}
	day := g.rvToday()
	byP := src == g.P || src != nil && src.AI == "follower" && !src.Has("rvally")
	if byP {
		g.rvDie(rv, day, "slain by "+g.P.Name, true)
		rv.Killer = "you"
		g.rvLog(day, fmt.Sprintf("{{R|%s was slain by %s.}}", g.rvName(rv), g.P.Name))
		g.Msg("{{W|Your rival " + g.rvName(rv) + " is dead.}}")
		g.deed("rival", fmt.Sprintf("%s the %s slew the rival %s, %s, at %s.", g.P.Name, g.Sk.Calling(g.Calling), g.rvName(rv), strings.ToLower(rv.Title), g.rvWhere()), 0, 0)
		return
	}
	how := "killed in the wilds"
	if src != nil && src.Name != "" {
		how = "killed by " + src.Name
	}
	if d.Has("rvally") {
		how += ", fighting beside " + g.P.Name
		g.deed("rival", fmt.Sprintf("%s fell fighting beside %s the %s.", g.rvName(rv), g.P.Name, g.Sk.Calling(g.Calling)), 0, 0)
	}
	g.rvDie(rv, day, how, true)
	rv.Killer = how
	g.rvLog(day, fmt.Sprintf("{{R|%s is dead:}} %s.", g.rvName(rv), how))
}

func (g *Game) rvWhere() string {
	if g.Cur == nil {
		return "an unknown place"
	}
	return nonEmpty(g.Cur.Name, g.wsPlace(g.W.Idx(g.Cur.WX, g.Cur.WY)))
}

// rvCull: for the endgame threat or other disasters; each living rival within rad of p (rad<0: anywhere)
// dies with probability frac, by pure hash.
func (g *Game) rvCull(p, rad int, frac float64, why string) {
	g.rvEnsure()
	d := g.rvToday()
	for _, rv := range g.rv().R {
		if !rv.Alive || rv.Allied || rad >= 0 && wsCheb(g.W, g.rvPosAt(rv, d), p) > rad {
			continue
		}
		if g.rvF("cull", rv.Slot, rv.Gen, d, why) < frac {
			g.rvDie(rv, d, why, false)
		}
	}
}

// ---- allies ----

func (g *Game) rvAllyActor() *Actor {
	for _, zn := range g.Zones {
		for _, a := range zn.Actors {
			if a.Has("rvally") && a.AI == "follower" && a.HP > 0 {
				return a
			}
		}
	}
	return nil
}

func (g *Game) rvAllyEnter(zn *Zone) {
	st := g.rv()
	for _, rv := range st.R {
		if !rv.Allied {
			continue
		}
		if zn.Kind == "dungeon" || zn.Kind == "tower" {
			rv.AllyIn = true
		} else if rv.AllyIn {
			g.rvAllyEnd(rv, "done")
		}
	}
}

func (g *Game) rvAllyEnd(rv *rvRival, why string) {
	if !rv.Allied {
		return
	}
	d := g.rvToday()
	for _, zn := range g.Zones {
		keep := zn.Actors[:0]
		for _, a := range zn.Actors {
			if a.Has("rvally") && g.rvOf(a) == rv {
				continue
			}
			keep = append(keep, a)
		}
		zn.Actors = keep
	}
	rv.Allied, rv.AllyIn = false, false
	rv.Act = "idle"
	rv.Wait = d + 2
	if g.W.In(g.WX, g.WY) {
		rv.Pos = g.W.Idx(g.WX, g.WY)
	}
	nm := g.rvName(rv)
	if why == "done" {
		rv.Rel = clampi(rv.Rel+12, -100, 100)
		g.rvGain(rv, 60+rv.Level*6)
		g.P.Gold += 0
		g.Msg("{{G|" + nm + " claps your shoulder at the dungeon mouth: \"Fair split. Next time, we race.\" They go their own way.}}")
		g.deed("rival", fmt.Sprintf("%s the %s and the rival %s cleared a dungeon together, then parted as equals.", g.P.Name, g.Sk.Calling(g.Calling), nm), 0, 0)
		g.rvLog(d, fmt.Sprintf("%s fought beside %s for a dungeon.", nm, g.P.Name))
	} else {
		g.Msg("{{y|" + nm + " goes their own way: the agreement was for one dungeon.}}")
	}
}

func (g *Game) rvMakeAlly(a *Actor, rv *rvRival) {
	bd := g.W.burgByID[g.Cur.Burg]
	town := g.Cur.Name
	if bd != nil {
		town = bd.Name
	}
	a.Comp = &Companion{Faction: a.Faction, AI: "wander", F: a.F, HomeKey: g.Cur.Key, HomeX: a.HomeX, HomeY: a.HomeY, Town: town, Loyalty: 70 + rv.Rel/2, Since: g.Turn, Summoned: true}
	a.Order, a.AI, a.Faction, a.Angry = OrdFollow, "follower", "player", false
	a.F = CGreen
	a.Set("rvally")
	rv.Allied, rv.AllyDay, rv.AllyIn, rv.Act = true, g.rvToday(), false, "ally"
	g.Msg("{{G|" + a.Name + " joins you for one dungeon.}} {{K|(o: companion orders)}}")
}

// ---- talk ----

func rvTalk(g *Game, a *Actor) []Option {
	if g.W == nil || !a.Has("rvr") || a.HP <= 0 || g.Cur == nil {
		return nil
	}
	rv := g.rvOf(a)
	if rv == nil || !rv.Alive {
		return nil
	}
	d := g.rvToday()
	cur := g.Sk.Currency
	var out []Option
	if a.AI == "follower" {
		return []Option{{"{{Y|Part ways with " + a.Name + ".}}", func() { g.rvAllyEnd(rv, "time"); g.Mode = "main" }}}
	}
	rv.Met = true
	mood := rvMood(rv)
	greet := rvGreetWary
	switch mood {
	case "friendly":
		greet = rvGreetFriendly
	case "hostile":
		greet = rvGreetHostile
	}
	say := greet[int(g.rvHash("greet", rv.Slot, d)%uint64(len(greet)))]
	info := fmt.Sprintf("%s\n\n%s the %s, level %d. Mood: %s %s", say, g.rvName(rv), rv.Title, rv.Level, rvMoodTag(rv), g.rvMeter(rv))
	out = append(out, Option{"What are they after?", func() {
		var t string
		switch rv.Act {
		case "travel":
			t = fmt.Sprintf(rvAimLines["travel"], rv.Target)
		case "delve":
			t = fmt.Sprintf(rvAimLines["delve"], rv.Target)
		case "hunt":
			t = fmt.Sprintf(rvAimLines["hunt"], rv.Target)
		default:
			t = rvAimLines["idle"]
		}
		if mood == "hostile" {
			t = "\"Why would I tell you?\" " + t
		}
		g.talkMenu(info + "\n\n" + t)
	}})
	if mood != "hostile" && rv.Social != d+1 {
		out = append(out, Option{"Trade news.", func() {
			rv.Social = d + 1
			rv.Rel = clampi(rv.Rel+3, -100, 100)
			lines := g.rvNews()
			txt := "\"Here is what I have heard.\""
			for i, l := range lines {
				if i >= 3 {
					break
				}
				txt += "\n\n" + l
			}
			txt += "\n\n" + g.wsGossip()
			g.talkMenu(txt)
		}})
		cost := 8 + rv.Level*2
		if g.P.Gold >= cost {
			out = append(out, Option{fmt.Sprintf("Share a meal (%d %s).", cost, cur), func() {
				if g.P.Gold < cost {
					return
				}
				g.P.Gold -= cost
				rv.Social = d + 1
				rv.Rel = clampi(rv.Rel+6, -100, 100)
				txt := "You eat and drink at their table. The rivalry softens, a little."
				var cs []rvSite
				for _, s := range g.rvSites(rv.Pos, 10) {
					cs = append(cs, s)
				}
				if len(cs) > 0 && rv.Rel >= 0 {
					s := cs[int(g.rvHash("meal", rv.Slot, d)%uint64(len(cs)))]
					if g.addSecret(fmt.Sprintf("%s is a place near %s. %s says it is worth the risk.", s.S.Name, g.W.RegionName(s.P), rv.Name), "dungeon", s.P, g.W.Year) {
						txt += "\n\nThey let slip: " + s.S.Name + " near " + g.W.RegionName(s.P) + ". (Recorded in your journal.)"
					}
				}
				g.deed("rival", fmt.Sprintf("%s the %s broke bread with the rival %s.", g.P.Name, g.Sk.Calling(g.Calling), g.rvName(rv)), 0, 0)
				g.talkMenu(txt)
			}})
		}
	}
	if mood != "hostile" && rv.Rel >= 10 && g.companionCount() < maxCompanions {
		out = append(out, Option{"{{G|Propose to clear a dungeon together.}}", func() {
			for _, o := range g.Cur.Actors {
				if o == a {
					g.rvMakeAlly(a, rv)
				}
			}
			g.Mode = "main"
			g.Msg(rvAllyQuips[int(g.rvHash("ally", rv.Slot, d)%uint64(len(rvAllyQuips)))])
			var cs []rvSite
			for _, s := range g.rvSites(g.W.Idx(g.Cur.WX, g.Cur.WY), 12) {
				cs = append(cs, s)
			}
			if len(cs) > 0 {
				s := cs[int(g.rvHash("allysite", rv.Slot, d)%uint64(len(cs)))]
				g.addSecret(fmt.Sprintf("%s is a place near %s, where %s proposes to fight beside you.", s.S.Name, g.W.RegionName(s.P), rv.Name), "dungeon", s.P, g.W.Year)
			}
		}})
	}
	bribe := 60 + rv.Level*25
	if rv.Bribed < d && g.P.Gold >= bribe {
		out = append(out, Option{fmt.Sprintf("Pay %d %s to keep out of your way for ten days.", bribe, cur), func() {
			if g.P.Gold < bribe {
				return
			}
			g.P.Gold -= bribe
			rv.Gold += bribe
			rv.Bribed = d + 10
			rv.Rel = clampi(rv.Rel+5, -100, 100)
			g.deed("rival", fmt.Sprintf("%s the %s paid the rival %s to stay out of the way.", g.P.Name, g.Sk.Calling(g.Calling), g.rvName(rv)), 0, 0)
			g.talkMenu("They pocket the coin. \"Ten days. After that, the road is the road.\"")
		}})
	}
	out = append(out, Option{"{{c|Slip something into their pack (sabotage).}}", func() {
		if g.roll2d6Check(g.checkMod("streetwise"), dcMedium+rv.Level/4) {
			rv.Wait = maxi(rv.Wait, d+4)
			rv.XP = maxi(0, rv.XP-30)
			g.rvLog(d, fmt.Sprintf("%s's gear failed them on the road.", g.rvName(rv)))
			g.deed("rival", fmt.Sprintf("%s the %s sabotaged the rival %s unseen.", g.P.Name, g.Sk.Calling(g.Calling), g.rvName(rv)), 0, 0)
			g.talkMenu("Your hands are quick. They will lose days to a cut strap and a spoiled ration, and never know why.")
			return
		}
		rv.Rel = clampi(rv.Rel-35, -100, 100)
		a.Angry, a.AI = true, "hostile"
		g.deed("rival", fmt.Sprintf("%s the %s was caught sabotaging the rival %s.", g.P.Name, g.Sk.Calling(g.Calling), g.rvName(rv)), 0, 0)
		g.Msg("{{R|" + a.Name + " catches your wrist. \"Thief!\"}}")
		g.Mode = "main"
	}})
	out = append(out, Option{"{{R|Challenge to a duel.}}", func() {
		a.Angry, a.AI = true, "hostile"
		if mood != "hostile" {
			a.Set("rvduel")
		}
		g.Mode = "main"
		g.Msg("{{R|" + a.Name + " draws. \"To first blood, then.\"}}")
	}})
	return out
}

// ---- sidebar ----

func rvSide(g *Game) []string {
	if g.W == nil || g.Cur == nil {
		return nil
	}
	var out []string
	for _, a := range g.Cur.Actors {
		if !a.Has("rvr") || a.HP <= 0 {
			continue
		}
		if rv := g.rvOf(a); rv != nil {
			out = append(out, fmt.Sprintf("Rival: %s L%d %s", a.Name, rv.Level, rvMoodTag(rv)))
		}
	}
	return out
}

// ---- exports ----

// rvDossier: lines for the Hunters screen.
func (g *Game) rvDossier() []string {
	if g.W == nil {
		return nil
	}
	g.rvAdvance()
	d := g.rvToday()
	var out []string
	for _, rv := range g.rv().R {
		out = append(out, fmt.Sprintf("{{W|%s}}, %s, level %d", g.rvName(rv), rv.Title, rv.Level))
		if !rv.Alive {
			out = append(out, fmt.Sprintf("  {{R|Dead}} (%s). A new rival arrives in about %d days.", rv.Killer, maxi(0, rv.Spawn-d)))
			out = append(out, "")
			continue
		}
		bd := g.rvBurg(rv.Home)
		sp := rv.Species
		if s := g.W.specByID[rv.Species]; s != nil {
			sp = s.Name
		}
		out = append(out, fmt.Sprintf("  %s of %s. Mood: %s %s", sp, bd.Name, rvMoodTag(rv), g.rvMeter(rv)))
		if rv.Group != "" {
			out = append(out, "  Sworn to "+rv.Group+".")
		}
		out = append(out, "  "+g.rvDoing(rv, d))
		out = append(out, fmt.Sprintf("  Dungeons cleared %d, beasts and bounties %d, prizes taken from you %d.", rv.Clears, rv.Kills, rv.Wins))
		for _, id := range rv.Relics {
			for i := range g.W.Artifacts {
				if g.W.Artifacts[i].ID == id {
					out = append(out, "  Carries the relic "+g.W.Artifacts[i].Name+".")
				}
			}
		}
		if rv.Bribed >= d {
			out = append(out, fmt.Sprintf("  Paid to keep the peace for %d more days.", rv.Bribed-d+1))
		}
		out = append(out, "")
	}
	if len(out) == 0 {
		out = append(out, "No rival adventurers are known on this world.")
	}
	return out
}

func (g *Game) rvDoing(rv *rvRival, d int) string {
	switch rv.Act {
	case "travel":
		return "On the road to " + rv.Target + ", near " + g.wsPlace(g.rvPosAt(rv, d)) + "."
	case "delve":
		return "Delving " + rv.Target + " (" + g.wsPlace(rv.Dest) + ")."
	case "hunt":
		return "Hunting " + rv.Target + " near " + g.wsPlace(rv.Dest) + "."
	case "ally":
		return "Fighting beside you."
	}
	return "Resting at " + g.wsPlace(rv.Pos) + "."
}

// rvNews: lines for the news screen and gossip: recent rival doings.
func (g *Game) rvNews() []string {
	if g.W == nil {
		return nil
	}
	g.rvAdvance()
	st := g.rv()
	d := g.rvToday()
	var out []string
	for i := len(st.Log) - 1; i >= 0 && len(out) < 5; i-- {
		l := st.Log[i]
		if d-l.Day > 30 {
			continue
		}
		out = append(out, fmt.Sprintf("{{m|Rival}}: %s", l.T))
	}
	for _, rv := range st.R {
		if rv.Alive && rv.Act != "ally" {
			out = append(out, fmt.Sprintf("{{m|Rival}}: %s the %s was last seen near %s.", g.rvName(rv), strings.ToLower(rv.Title), g.wsPlace(g.rvPosAt(rv, d))))
		}
	}
	return out
}
