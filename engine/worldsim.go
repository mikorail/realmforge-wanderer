package engine

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
)

// worldsim.go: a light, living world simulation advanced by game time. Travellers walk the
// roads parasang by parasang (caravans, pilgrims, minstrels, scholars, mercenaries, beast
// tamers, hirelings, armies, and evil slaver caravans); events (plague, invasion, desertion,
// famine, bandits, boom/bust, slave pens) spread over regions over days; commodity prices
// drift per region and react to events. Everything is rolled from seed + day buckets and
// kept in g.ext("ws"). Captives exist only to be freed, raided or reported: people are never
// for sale.

var wsKinds = []string{"food", "water", "ore", "cloth", "weapons", "medicine", "beasts"}

type wsTrav struct {
	ID     int    `json:"id"`
	Kind   string `json:"k"` // caravan pilgrim minstrel scholar merc tamer hireling army slaver
	Name   string `json:"n"`
	Path   []int  `json:"path"`
	Start  int    `json:"st"` // game hour the walk began
	Hpp    int    `json:"hpp"`
	Realm  int    `json:"rl"`
	Foe    int    `json:"foe,omitempty"`
	Size   int    `json:"sz"`
	Home   int    `json:"home"` // burg id the group hails from
	To     int    `json:"to"`   // destination burg id
	Done   bool   `json:"done,omitempty"`
	Broken bool   `json:"brk,omitempty"` // slavers destroyed or arrested
	Rep    int    `json:"rep,omitempty"` // hour reported to guards (+1)
}

type wsEvent struct {
	ID       int    `json:"id"`
	Kind     string `json:"k"` // plague invasion desertion famine bandits boom bust slavepen
	Name     string `json:"n"`
	P        int    `json:"p"`
	R        int    `json:"r"`
	Str      int    `json:"s"`
	Start    int    `json:"st"`
	End      int    `json:"end"` // 0 = until cleared
	Realm    int    `json:"rl,omitempty"`
	Foe      int    `json:"foe,omitempty"`
	Reported int    `json:"rep,omitempty"`
}

type wsState struct {
	Init   bool            `json:"init"`
	Hour   int             `json:"hour"`
	NextEv int             `json:"nev"`
	Trav   []*wsTrav       `json:"trav"`
	Ev     []*wsEvent      `json:"ev"`
	Seen   map[string]bool `json:"seen"`
	Freed  int             `json:"freed"`
	Log    []string        `json:"log"`
	burgs  []*BurgData
}

func init() {
	turnHooks = append(turnHooks, wsTurn)
	enterHooks = append(enterHooks, wsEnter)
	talkHooks = append(talkHooks, wsTalk)
	killHooks = append(killHooks, wsKill)
	sideHooks = append(sideHooks, wsSide)
	mainKeys["N"] = func(g *Game) { g.Mode = "wsnews"; g.Sel = 0 }
	modeKeys["wsnews"] = func(g *Game, k string) {
		switch k {
		case "Escape", "Enter", "N", " ", "q":
			g.Mode = "main"
		case "ArrowDown", "j", "N2":
			g.Sel++
		case "ArrowUp", "k", "N8":
			g.Sel--
		case "PageDown":
			g.Sel += 10
		case "PageUp":
			g.Sel -= 10
		}
	}
	modeDraws["wsnews"] = func(g *Game) { g.wsDrawNews() }
}

func (g *Game) ws() *wsState {
	s := g.ext("ws", func() any { return &wsState{} }).(*wsState)
	if s.Seen == nil {
		s.Seen = map[string]bool{}
	}
	return s
}

// ---- helpers ----

func wsHashF(parts ...any) float64 {
	h := HashStr(fmt.Sprint(parts...))
	return float64(h%2001)/1000 - 1
}

func (g *Game) wsHour() int { return g.Turn / 60 }

func wsDay(h int) int {
	if h < 0 {
		return -((-h + 23) / 24)
	}
	return h / 24
}

func wsCheb(w *WorldData, a, b int) int {
	return maxi(abs(a%w.W-b%w.W), abs(a/w.W-b/w.W))
}

func (s *wsState) burgList(g *Game) []*BurgData {
	if s.burgs == nil {
		for i := range g.W.Burgs {
			b := &g.W.Burgs[i]
			if b.Alive && (b.Layer == "" || b.Layer == "surface") && b.P >= 0 && b.P < len(g.W.Biome) && !g.W.IsWater(b.P) {
				s.burgs = append(s.burgs, b)
			}
		}
		if s.burgs == nil {
			s.burgs = []*BurgData{}
		}
	}
	return s.burgs
}

// wsPlace names a parasang: the town on it, or "near <nearest town>".
func (g *Game) wsPlace(p int) string {
	w := g.W
	if b := w.burgByID[w.Burg[p]]; b != nil {
		return b.Name
	}
	best, bn := 9, ""
	for _, b := range g.ws().burgList(g) {
		if d := wsCheb(w, b.P, p); d < best {
			best, bn = d, b.Name
		}
	}
	if bn != "" {
		return "the country near " + bn
	}
	return w.RegionName(p)
}

func (g *Game) wsDirDist(from, to int) string {
	w := g.W
	dx, dy := to%w.W-from%w.W, to/w.W-from/w.W
	d := maxi(abs(dx), abs(dy))
	if d == 0 {
		return "here"
	}
	ns, ew := "", ""
	if dy < -d/3 {
		ns = "N"
	} else if dy > d/3 {
		ns = "S"
	}
	if dx < -d/3 {
		ew = "W"
	} else if dx > d/3 {
		ew = "E"
	}
	return fmt.Sprintf("%d parasangs %s%s", d, ns, ew)
}

// route walks roads (and towns) from a to b; falls back to a straight line over land.
func (s *wsState) route(g *Game, a, b int) []int {
	w := g.W
	if a == b {
		return []int{a}
	}
	prev := map[int]int{a: -1}
	q := []int{a}
	for n := 0; len(q) > 0 && n < 6000; n++ {
		c := q[0]
		q = q[1:]
		if c == b {
			break
		}
		cx, cy := c%w.W, c/w.W
		for dy := -1; dy <= 1; dy++ {
			for dx := -1; dx <= 1; dx++ {
				nx, ny := cx+dx, cy+dy
				if (dx == 0 && dy == 0) || !w.In(nx, ny) {
					continue
				}
				np := w.Idx(nx, ny)
				if _, ok := prev[np]; ok || w.IsWater(np) {
					continue
				}
				if np != b && w.Road[np] == 0 && w.Burg[np] == 0 {
					continue
				}
				prev[np] = c
				q = append(q, np)
			}
		}
	}
	if _, ok := prev[b]; ok {
		var path []int
		for c := b; c != -1; c = prev[c] {
			path = append(path, c)
		}
		for i, j := 0, len(path)-1; i < j; i, j = i+1, j-1 {
			path[i], path[j] = path[j], path[i]
		}
		return path
	}
	// straight line
	ax, ay, bx, by := a%w.W, a/w.W, b%w.W, b/w.W
	n := maxi(abs(bx-ax), abs(by-ay))
	var path []int
	for i := 0; i <= n; i++ {
		x := ax + (bx-ax)*i/maxi(n, 1)
		y := ay + (by-ay)*i/maxi(n, 1)
		p := w.Idx(x, y)
		if w.IsWater(p) {
			return nil
		}
		if len(path) == 0 || path[len(path)-1] != p {
			path = append(path, p)
		}
	}
	return path
}

// at returns the parasang a traveller stands on at an hour.
func (t *wsTrav) at(hour int) (int, bool) {
	if hour < t.Start || t.Hpp <= 0 {
		return 0, false
	}
	i := (hour - t.Start) / t.Hpp
	if i >= len(t.Path) {
		return t.Path[len(t.Path)-1], false
	}
	return t.Path[i], true
}

func (t *wsTrav) end() int { return t.Start + len(t.Path)*t.Hpp }

func (t *wsTrav) active(hour int) bool {
	_, ok := t.at(hour)
	return ok && !t.Done && !t.Broken
}

func (s *wsState) travByID(id int) *wsTrav {
	for _, t := range s.Trav {
		if t.ID == id {
			return t
		}
	}
	return nil
}

func wsTag(id int) string { return "wst:" + strconv.Itoa(id) }

func wsTagOf(a *Actor) (int, bool) {
	for k := range a.Flags {
		if strings.HasPrefix(k, "wst:") {
			if id, err := strconv.Atoi(k[4:]); err == nil {
				return id, true
			}
		}
	}
	return 0, false
}

func (e *wsEvent) covers(w *WorldData, p int) bool {
	switch e.Kind {
	case "famine":
		return w.State[p] == e.Realm
	}
	return wsCheb(w, e.P, p) <= e.R
}

// ---- advancing time ----

func (g *Game) wsAdvance() {
	if g.W == nil {
		return
	}
	s := g.ws()
	now := g.wsHour()
	if !s.Init {
		s.Init = true
		s.Hour = now
		for d := wsDay(now) - 3; d <= wsDay(now); d++ {
			g.wsDaily(d)
		}
		return
	}
	if now-s.Hour > 24*20 {
		s.Hour = now - 24*20
	}
	for s.Hour < now {
		s.Hour++
		if s.Hour%24 == 0 {
			g.wsDaily(s.Hour / 24)
		}
		g.wsRetire(s.Hour)
	}
	if len(s.Seen) > 4000 {
		s.Seen = map[string]bool{}
	}
}

func (g *Game) wsRetire(hour int) {
	s := g.ws()
	keep := s.Trav[:0]
	for _, t := range s.Trav {
		if hour >= t.end() && !t.Done {
			t.Done = true
			if t.Kind == "army" && !t.Broken {
				g.wsArmyArrive(t, hour)
			}
		}
		if hour < t.end()+48 {
			keep = append(keep, t)
		}
	}
	s.Trav = keep
	ke := s.Ev[:0]
	for _, e := range s.Ev {
		if e.End > 0 && hour >= e.End {
			g.wsEventEnds(e, hour)
			continue
		}
		ke = append(ke, e)
	}
	s.Ev = ke
}

func (g *Game) wsLog(hour int, text string) {
	s := g.ws()
	s.Log = append(s.Log, fmt.Sprintf("Day %d: %s", hour/24+1, text))
	if len(s.Log) > 12 {
		s.Log = s.Log[len(s.Log)-12:]
	}
}

func (g *Game) wsAddEvent(e *wsEvent) *wsEvent {
	s := g.ws()
	if len(s.Ev) >= 18 {
		return nil
	}
	for _, o := range s.Ev {
		if o.Kind == e.Kind && o.P == e.P && o.Realm == e.Realm {
			return nil
		}
	}
	s.NextEv++
	e.ID = s.NextEv
	s.Ev = append(s.Ev, e)
	return e
}

func (g *Game) wsArmyArrive(t *wsTrav, hour int) {
	end := t.Path[len(t.Path)-1]
	if g.W.State[end] == t.Foe || g.W.State[t.Path[len(t.Path)/2]] == t.Foe {
		if e := g.wsAddEvent(&wsEvent{Kind: "invasion", P: end, R: 1, Str: 2 + t.Size/4, Start: hour, End: hour + 24*10, Realm: t.Realm, Foe: t.Foe,
			Name: fmt.Sprintf("%s occupy the outskirts of %s", g.realmName(t.Realm), g.wsPlace(end))}); e != nil {
			g.wsLog(hour, e.Name)
		}
	}
}

func (g *Game) wsEventEnds(e *wsEvent, hour int) {
	switch e.Kind {
	case "invasion":
		// a beaten or idle army sheds deserters
		if HashStr(fmt.Sprint(g.Seed, e.ID))%2 == 0 {
			if d := g.wsAddEvent(&wsEvent{Kind: "desertion", P: e.P, R: 2, Str: 2, Start: hour, End: hour + 24*8, Realm: e.Realm,
				Name: fmt.Sprintf("Deserters of %s roam near %s", g.realmName(e.Realm), g.wsPlace(e.P))}); d != nil {
				g.wsLog(hour, d.Name)
			}
		}
	case "desertion":
		if b := g.wsAddEvent(&wsEvent{Kind: "bandits", P: e.P, R: 1, Str: 3, Start: hour, Realm: e.Realm,
			Name: "Deserters turned bandit near " + g.wsPlace(e.P)}); b != nil {
			g.wsLog(hour, b.Name)
		}
	case "plague":
		g.wsLog(hour, "The plague around "+g.wsPlace(e.P)+" burns out")
	}
}

func (g *Game) wsLawless(p int) bool {
	w := g.W
	if w.State[p] == 0 {
		return true
	}
	for i := range w.Outlaws {
		if o := &w.Outlaws[i]; o.P >= 0 && o.P < len(w.State) && wsCheb(w, o.P, p) <= 5 {
			return true
		}
	}
	return false
}

func (g *Game) wsDaily(d int) {
	s := g.ws()
	w := g.W
	hour := d * 24
	r := NewRng(fmt.Sprintf("%s:ws:%d", g.Seed, d))
	bl := s.burgList(g)
	if len(bl) == 0 {
		return
	}
	// age events: spread
	for _, e := range s.Ev {
		switch e.Kind {
		case "plague":
			if e.R < 5 && r.Chance(0.6) {
				e.R++
			}
			if e.Str >= 2 && (hour-e.Start) > 48 && r.Chance(0.2) {
				b := bl[r.Int(0, len(bl)-1)]
				if wsCheb(w, b.P, e.P) <= 10 && wsCheb(w, b.P, e.P) > e.R {
					g.wsAddEvent(&wsEvent{Kind: "plague", P: b.P, R: 1, Str: e.Str - 1, Start: hour, End: hour + 24*10,
						Name: "Plague spreads to " + b.Name})
				}
			}
		case "bandits":
			if r.Chance(0.5) && e.Str < 12 {
				e.Str++
				if e.Str%3 == 0 && e.R < 4 {
					e.R++
				}
			}
			if e.Str >= 6 && r.Chance(0.15) {
				b := bl[r.Int(0, len(bl)-1)]
				if wsCheb(w, b.P, e.P) <= 8 && b.P != e.P {
					g.wsAddEvent(&wsEvent{Kind: "bandits", P: b.P, R: 1, Str: 2, Start: hour, Realm: e.Realm,
						Name: "Bandit camps spread toward " + b.Name})
				}
			}
		}
	}
	// new events
	if r.Chance(0.04) {
		b := bl[r.Int(0, len(bl)-1)]
		if e := g.wsAddEvent(&wsEvent{Kind: "plague", P: b.P, R: 1, Str: 3 + r.Int(0, 2), Start: hour, End: hour + 24*18, Name: "Plague in " + b.Name}); e != nil {
			g.wsLog(hour, e.Name)
		}
	}
	if r.Chance(0.03) {
		st := w.States[r.Int(0, len(w.States)-1)]
		if st.Alive && st.ID > 0 {
			if e := g.wsAddEvent(&wsEvent{Kind: "famine", P: w.Burgs[0].P, Realm: st.ID, Str: 2, Start: hour, End: hour + 24*25, Name: "Famine in " + st.Name}); e != nil {
				g.wsLog(hour, e.Name)
			}
		}
	}
	if r.Chance(0.05) {
		b := bl[r.Int(0, len(bl)-1)]
		k, word := "boom", "Trade booms in "
		if r.Chance(0.5) {
			k, word = "bust", "Trade collapses in "
		}
		if e := g.wsAddEvent(&wsEvent{Kind: k, P: b.P, R: 3, Str: 2, Start: hour, End: hour + 24*14, Realm: b.State, Name: word + b.Name}); e != nil {
			g.wsLog(hour, e.Name)
		}
	}
	if r.Chance(0.06) {
		var p int
		if len(w.Outlaws) > 0 && r.Chance(0.6) {
			p = w.Outlaws[r.Int(0, len(w.Outlaws)-1)].P
		} else {
			p = bl[r.Int(0, len(bl)-1)].P
		}
		if p >= 0 && p < len(w.Biome) && !w.IsWater(p) && w.Burg[p] == 0 {
			if e := g.wsAddEvent(&wsEvent{Kind: "bandits", P: p, R: 1, Str: 2, Start: hour, Name: "Bandits camp near " + g.wsPlace(p)}); e != nil {
				g.wsLog(hour, e.Name)
			}
		}
	}
	if r.Chance(0.03) && len(w.Outlaws) > 0 {
		p := w.Outlaws[r.Int(0, len(w.Outlaws)-1)].P
		if p >= 0 && p < len(w.Biome) && !w.IsWater(p) && w.Burg[p] == 0 {
			if e := g.wsAddEvent(&wsEvent{Kind: "slavepen", P: p, Str: 3, Start: hour, End: hour + 24*14, Name: "Slavers keep a pen near " + g.wsPlace(p)}); e != nil {
				g.wsLog(hour, e.Name)
			}
		}
	}
	// travellers
	n := mini(8, 2+len(bl)/25)
	for i := 0; i < n && len(s.Trav) < 45; i++ {
		g.wsSpawnTrav(d, i, NewRng(fmt.Sprintf("%s:wst:%d:%d", g.Seed, d, i)), hour+r.Int(0, 23))
	}
	active := map[int]int{}
	for _, t := range s.Trav {
		if t.Kind == "army" && !t.Done {
			active[t.Realm]++
		}
	}
	for _, st := range w.States {
		if !st.Alive || st.ID == 0 || active[st.ID] >= 2 {
			continue
		}
		if foes := g.warsOf(st.ID); len(foes) > 0 && r.Chance(0.3) {
			g.wsSpawnArmy(d, st.ID, foes[r.Int(0, len(foes)-1)], r, hour+r.Int(0, 23))
		}
	}
}

var wsTravKinds = []struct {
	k string
	w int
}{{"caravan", 30}, {"pilgrim", 14}, {"minstrel", 8}, {"scholar", 8}, {"merc", 12}, {"tamer", 8}, {"hireling", 7}, {"slaver", 7}}

var wsHpp = map[string]int{"caravan": 6, "pilgrim": 7, "minstrel": 6, "scholar": 6, "merc": 4, "tamer": 6, "hireling": 5, "slaver": 6, "army": 8}

func (g *Game) wsSpawnTrav(d, i int, r *Rng, start int) {
	s := g.ws()
	w := g.W
	bl := s.burgList(g)
	tot := 0
	for _, k := range wsTravKinds {
		tot += k.w
	}
	roll := r.Int(0, tot-1)
	kind := "caravan"
	for _, k := range wsTravKinds {
		if roll < k.w {
			kind = k.k
			break
		}
		roll -= k.w
	}
	a := bl[r.Int(0, len(bl)-1)]
	startP := a.P
	if kind == "slaver" {
		if len(w.Outlaws) == 0 {
			return
		}
		o := w.Outlaws[r.Int(0, len(w.Outlaws)-1)]
		if o.P < 0 || o.P >= len(w.Biome) || w.IsWater(o.P) {
			return
		}
		startP = o.P
	}
	var b *BurgData
	var path []int
	for try := 0; try < 6 && path == nil; try++ {
		c := bl[r.Int(0, len(bl)-1)]
		if c.P == startP || wsCheb(w, c.P, startP) > 16 {
			continue
		}
		if p := s.route(g, startP, c.P); len(p) >= 4 && len(p) <= 40 {
			b, path = c, p
		}
	}
	if path == nil {
		return
	}
	hs := a
	if kind == "slaver" {
		hs = b
	}
	t := &wsTrav{ID: (d+10)*100 + i, Kind: kind, Path: path, Start: start, Hpp: wsHpp[kind], Realm: hs.State, Home: hs.ID, To: b.ID, Size: r.Int(2, 4)}
	t.Name = g.wsTravName(t, hs)
	s.Trav = append(s.Trav, t)
}

func (g *Game) wsTravName(t *wsTrav, hs *BurgData) string {
	to := ""
	if b := g.W.burgByID[t.To]; b != nil {
		to = b.Name
	}
	switch t.Kind {
	case "caravan":
		return fmt.Sprintf("caravan of %s bound for %s", hs.Name, to)
	case "pilgrim":
		return fmt.Sprintf("pilgrims from %s bound for %s", hs.Name, to)
	case "minstrel":
		return "wandering minstrel bound for " + to
	case "scholar":
		return "travelling scholar bound for " + to
	case "merc":
		return "mercenary band bound for " + to
	case "tamer":
		return "beast tamer bound for " + to
	case "hireling":
		return "sellsword bound for " + to
	case "slaver":
		return "slavers' coffle bound for " + to
	}
	return t.Kind
}

func (g *Game) wsSpawnArmy(d, realm, foe int, r *Rng, start int) {
	s := g.ws()
	from := g.warBurg(realm)
	to := g.warBurg(foe)
	if from == nil || to == nil || from.P == to.P {
		return
	}
	path := s.route(g, from.P, to.P)
	if len(path) < 5 {
		return
	}
	if len(path) > 45 {
		path = path[:45]
	}
	t := &wsTrav{ID: (d+10)*100 + 90 + realm%9, Kind: "army", Path: path, Start: start, Hpp: wsHpp["army"], Realm: realm, Foe: foe, Home: from.ID, To: to.ID, Size: r.Int(5, 10)}
	t.Name = fmt.Sprintf("army of %s marching on %s", g.realmName(realm), g.realmName(foe))
	s.Trav = append(s.Trav, t)
	g.wsLog(start, "The "+t.Name)
}

// ---- prices ----

func (g *Game) wsRegion(p int) int {
	w := g.W
	if p < 0 || p >= len(w.State) {
		return 0
	}
	if w.State[p] > 0 {
		return w.State[p]
	}
	return 1000 + (p%w.W/6)*100 + p/w.W/6
}

// wsPrice is the market multiplier for a commodity kind at a parasang: slow regional drift
// times what the active events do to it.
func (g *Game) wsPrice(p int, kind string) float64 {
	if g.W == nil || p < 0 || p >= len(g.W.State) {
		return 1
	}
	day := float64(g.Turn) / 1440
	reg := g.wsRegion(p)
	b := math.Floor(day / 3)
	f := day/3 - b
	v := wsHashF(g.Seed, reg, kind, int(b))*(1-f) + wsHashF(g.Seed, reg, kind, int(b)+1)*f
	m := 1 + 0.15*v
	for _, e := range g.ws().Ev {
		if !e.covers(g.W, p) {
			continue
		}
		st := float64(e.Str)
		switch e.Kind {
		case "plague":
			switch kind {
			case "medicine":
				m *= 1 + 0.3*st
			case "food":
				m *= 1.1
			}
		case "famine":
			switch kind {
			case "food":
				m *= 1 + 0.35*st
			case "water":
				m *= 1.2
			}
		case "invasion":
			switch kind {
			case "weapons":
				m *= 1.35
			case "food", "medicine":
				m *= 1.2
			}
		case "desertion":
			if kind == "weapons" {
				m *= 0.9
			}
		case "bandits":
			if kind == "ore" || kind == "cloth" || kind == "food" {
				m *= 1.08
			}
		case "boom":
			if kind != "food" {
				m *= 1.15
			}
		case "bust":
			m *= 0.85
		}
	}
	return math.Min(4, math.Max(0.4, m))
}

func wsItemKind(it *Item) string {
	switch it.Kind {
	case "food":
		return "food"
	case "water":
		return "water"
	case "tonic":
		return "medicine"
	case "weapon", "ranged", "ammo", "grenade":
		return "weapons"
	case "armor":
		return "cloth"
	case "scrap", "ingredient":
		return "ore"
	}
	return ""
}

// wsItemMul is the factor price() applies for the local market.
func (g *Game) wsItemMul(it *Item) float64 {
	if g.Cur == nil || it == nil || !g.W.In(g.Cur.WX, g.Cur.WY) {
		return 1
	}
	k := wsItemKind(it)
	if k == "" {
		return 1
	}
	return g.wsPrice(g.W.Idx(g.Cur.WX, g.Cur.WY), k)
}

// ---- hooks: time ----

func wsTurn(g *Game) {
	if g.Turn%60 != 0 && g.ws().Init {
		wsPlagueTick(g)
		return
	}
	g.wsAdvance()
	wsPlagueTick(g)
	if g.Turn%60 == 0 {
		wsInfectCheck(g)
	}
}

// wsOnMove: one-line hook for WorldMove, after the travel turns pass.
func wsOnMove(g *Game) {
	g.wsAdvance()
	if !g.W.In(g.WX, g.WY) {
		return
	}
	p := g.W.Idx(g.WX, g.WY)
	h := g.wsHour()
	for _, t := range g.ws().Trav {
		if q, ok := t.at(h); ok && q == p && t.active(h) {
			g.Msg("{{Y|You pass a " + t.Name + " on the road.}}")
		}
	}
	for _, e := range g.ws().Ev {
		if e.Kind != "slavepen" && e.covers(g.W, p) && e.Kind != "famine" && e.Kind != "boom" && e.Kind != "bust" {
			g.Msg("{{R|" + e.Name + ".}}")
		}
	}
	wsInfectCheck(g)
}

func wsPlagueTick(g *Game) {
	if g.P.St("plague") > 0 && g.Turn%25 == 0 {
		g.hurt(g.P, 1, nil, "the plague")
		if g.Turn%100 == 0 {
			g.Msg("{{R|Fever wracks you.}}")
		}
	}
}

func wsInfectCheck(g *Game) {
	if g.Cur == nil || g.P.St("plague") > 0 || !g.W.In(g.Cur.WX, g.Cur.WY) {
		return
	}
	p := g.W.Idx(g.Cur.WX, g.Cur.WY)
	chance := 0.0
	for _, e := range g.ws().Ev {
		if e.Kind == "plague" && e.covers(g.W, p) {
			chance += 0.04 * float64(e.Str)
		}
	}
	for _, a := range g.Cur.Actors {
		if a.Has("wssick") && a.HP > 0 && abs(a.X-g.P.X) <= 3 && abs(a.Y-g.P.Y) <= 3 {
			chance += 0.1
		}
	}
	chance *= 1 - 0.2*float64(g.Skills["survival"])
	if chance > 0 && g.R.Chance(chance) {
		g.P.AddSt("plague", 2000)
		g.Msg("{{R|You feel feverish: the plague has taken you.}} {{K|(A healer or apothecary can treat it.)}}")
	}
}

// ---- hooks: entering a zone ----

func (g *Game) wsAt(zn *Zone, a *Actor, cx, cy, rad int, r *Rng) bool {
	x, y, ok := zn.FreeSpot(r, func(x, y int) bool { return abs(x-cx) <= rad && abs(y-cy) <= rad })
	if !ok {
		return false
	}
	a.X, a.Y, a.HomeX, a.HomeY = x, y, x, y
	zn.Actors = append(zn.Actors, a)
	return true
}

func wsBurn(zn *Zone, r *Rng, pct float64) {
	for i, t := range zn.T {
		if !r.Chance(pct) {
			continue
		}
		switch t {
		case TGrass, TTallGrass, TBush, TCrop, TMoss, TReeds:
			zn.T[i] = TAsh
		case TTree:
			zn.T[i] = TDeadTree
		}
	}
}

func wsEnter(g *Game, zn *Zone) {
	if g.W == nil || !g.W.In(zn.WX, zn.WY) || zn.Z != 0 {
		return
	}
	g.wsAdvance()
	s := g.ws()
	w := g.W
	p := w.Idx(zn.WX, zn.WY)
	h := g.wsHour()
	// travellers who have moved on leave
	keep := zn.Actors[:0]
	for _, a := range zn.Actors {
		if id, ok := wsTagOf(a); ok && a.AI != "follower" && !a.Has("wskeep") {
			if t := s.travByID(id); t == nil || !t.active(h) || func() bool { q, _ := t.at(h); return q != p }() {
				continue
			}
		}
		keep = append(keep, a)
	}
	zn.Actors = keep
	if zn.Kind == "town" {
		g.wsEnterTown(zn, p)
		return
	}
	if zn.Kind != "surface" || w.Burg[p] != 0 {
		return
	}
	for _, t := range s.Trav {
		q, ok := t.at(h)
		if !ok || q != p || t.Done || t.Broken {
			continue
		}
		key := fmt.Sprintf("t%d@%d", t.ID, (h-t.Start)/t.Hpp)
		if s.Seen[key] {
			continue
		}
		s.Seen[key] = true
		g.wsSpawnGroup(zn, t, NewRng(g.Seed+":wsg:"+strconv.Itoa(t.ID)+":"+zn.Key))
	}
	day := wsDay(h)
	for _, e := range s.Ev {
		if !e.covers(w, p) {
			continue
		}
		key := fmt.Sprintf("e%d:%s:%d", e.ID, zn.Key, day)
		if s.Seen[key] {
			continue
		}
		s.Seen[key] = true
		g.wsEventZone(zn, e, NewRng(g.Seed+":wse:"+key))
	}
}

func (g *Game) wsEnterTown(zn *Zone, p int) {
	s := g.ws()
	for _, e := range s.Ev {
		if e.Kind != "plague" || !e.covers(g.W, p) {
			continue
		}
		key := fmt.Sprintf("tp%d:%s", e.ID, zn.Key)
		if s.Seen[key] {
			return
		}
		s.Seen[key] = true
		r := NewRng(g.Seed + ":wsp:" + key)
		for _, a := range zn.Actors {
			if (a.Role == "villager" || a.Role == "") && a.AI != "follower" && r.Chance(0.3) {
				a.Set("wssick")
				a.Desc += " They are pale and feverish."
				a.F = CGreen
			}
		}
		if e.Str >= 3 {
			g.Msg("{{R|The gates are barred against the plague; wardens in rags and masks watch you pass.}}")
			if bd := g.W.burgByID[zn.Burg]; bd != nil {
				for i := 0; i < 2; i++ {
					a := g.makeNPC("guard", bd, g.W.cultByID[bd.Culture], r)
					a.Title, a.Desc = "plague warden", "A masked warden, keeping the quarantine."
					a.Set("wswarden")
					g.wsAt(zn, a, g.P.X, g.P.Y, 5, r)
				}
			}
		}
		return
	}
}

func (g *Game) wsHome(t *wsTrav) *BurgData {
	if b := g.W.burgByID[t.Home]; b != nil {
		return b
	}
	return g.ws().burgList(g)[0]
}

func (g *Game) wsPerson(t *wsTrav, role, title string, r *Rng) *Actor {
	bd := g.wsHome(t)
	a := g.makeNPC(role, bd, g.W.cultByID[bd.Culture], r)
	a.Title = title
	a.Desc = fmt.Sprintf("A %s of %s, on the road.", title, bd.Name)
	a.Set(wsTag(t.ID))
	return a
}

func (g *Game) wsSpawnGroup(zn *Zone, t *wsTrav, r *Rng) {
	cx, cy, ok := zn.FreeSpot(r, func(x, y int) bool { return x > 12 && x < MW-12 && y > 5 && y < MH-5 })
	if !ok {
		return
	}
	add := func(a *Actor, rad int) *Actor {
		if a != nil && g.wsAt(zn, a, cx, cy, rad, r) {
			return a
		}
		return nil
	}
	guards := func(n int, title string) {
		for i := 0; i < n; i++ {
			a := g.wsPerson(t, "guard", title, r)
			a.AI = "wander"
			add(a, 4)
		}
	}
	g.Msg("{{Y|You come upon a " + t.Name + ".}}")
	switch t.Kind {
	case "caravan":
		m := g.wsPerson(t, "merchant", "caravan master", r)
		if g.hasEventAt(zn, "plague") {
			m.Inv = append(m.Inv, g.MakeTonic("heal", 2), g.MakeTonic("antidote", 2))
		}
		add(m, 3)
		guards(r.Int(2, 3), "caravan guard")
	case "pilgrim":
		for i := 0; i < t.Size; i++ {
			a := g.wsPerson(t, "villager", "pilgrim", r)
			a.AI = "wander"
			a.Set("wspilgrim")
			add(a, 4)
		}
	case "minstrel":
		a := g.wsPerson(t, "villager", "minstrel", r)
		a.AI = "wander"
		a.Set("wsminstrel")
		add(a, 3)
	case "scholar":
		a := g.wsPerson(t, "villager", "scholar", r)
		a.AI = "wander"
		a.Set("wsscholar")
		a.Inv = append(a.Inv, g.RandomBook(r, g.W.Idx(zn.WX, zn.WY)))
		add(a, 3)
	case "merc":
		for i := 0; i < t.Size; i++ {
			title := "sellsword"
			if i == 0 {
				title = "mercenary captain"
			}
			a := g.wsPerson(t, "guard", title, r)
			a.AI = "wander"
			a.Level += 1
			a.Set("wshire")
			add(a, 4)
		}
	case "hireling":
		a := g.wsPerson(t, "guard", Pick(r, []string{"sellsword", "scout", "hedge-knight"}), r)
		a.AI = "wander"
		a.Set("wshire")
		add(a, 3)
	case "tamer":
		a := g.wsPerson(t, "villager", "beast tamer", r)
		a.AI = "still"
		a.Set("wstamer")
		if add(a, 3) == nil {
			return
		}
		for i := 0; i < r.Int(1, 2); i++ {
			tp := Pick(r, []string{"boar", "wolf", "bear", "jaguar"})
			b := g.Spawn(tp, maxi(2, zn.Level), r)
			b.Faction, b.AI, b.Angry = fmt.Sprintf("state:%d", t.Realm), "still", false
			b.Name = "trained " + b.Name
			b.Set("wsbeast")
			b.Set(wsTag(t.ID))
			add(b, 4)
		}
	case "slaver":
		g.wsSlaverGroup(zn, t, cx, cy, r)
	case "army":
		g.wsArmyGroup(zn, t, cx, cy, r)
	}
}

func (g *Game) hasEventAt(zn *Zone, kind string) bool {
	p := g.W.Idx(zn.WX, zn.WY)
	for _, e := range g.ws().Ev {
		if e.Kind == kind && e.covers(g.W, p) {
			return true
		}
	}
	return false
}

func (g *Game) wsSlaverGroup(zn *Zone, t *wsTrav, cx, cy int, r *Rng) {
	for i := 0; i < 3; i++ {
		a := g.Spawn(Pick(r, []string{"bandit", "bandit", "archer"}), maxi(3, zn.Level), r)
		a.Name, a.Title = "slaver", "slaver"
		a.Set("slaver")
		a.Set(wsTag(t.ID))
		g.wsAt(zn, a, cx, cy, 4, r)
	}
	g.wsCaptives(zn, t.ID, cx, cy, r.Int(2, 4), r)
}

// wsCaptives chains up bound prisoners from real realms. They are never for sale: they can only be freed.
func (g *Game) wsCaptives(zn *Zone, tag, cx, cy, n int, r *Rng) {
	var realms []int
	for _, st := range g.W.States {
		if st.Alive && st.ID > 0 && g.warBurg(st.ID) != nil {
			realms = append(realms, st.ID)
		}
	}
	if len(realms) == 0 {
		return
	}
	for i := 0; i < n; i++ {
		realm := realms[r.Int(0, len(realms)-1)]
		bd := g.warBurg(realm)
		a := g.makeNPC("villager", bd, g.W.cultByID[bd.Culture], r)
		a.Title = "captive"
		a.Desc = fmt.Sprintf("A bound captive from %s, wrists raw from the shackles.", bd.Name)
		a.AI = "still"
		a.Inv = nil
		a.Gold = 0
		a.Set("captive")
		a.Set(wsTag(tag))
		a.Set("wskeep")
		g.wsAt(zn, a, cx, cy, 2, r)
	}
}

func (g *Game) wsArmyGroup(zn *Zone, t *wsTrav, cx, cy int, r *Rng) {
	p := g.W.Idx(zn.WX, zn.WY)
	n := mini(4+t.Size/2, 9)
	for i := 0; i < n; i++ {
		a := g.soldier(t.Realm, r)
		if a == nil {
			return
		}
		a.Set(wsTag(t.ID))
		if i == 0 {
			a.Title, a.Level = "captain", a.Level+2
			a.Set("wscaptain")
		}
		g.wsAt(zn, a, cx, cy, 5, r)
	}
	if bd := g.wsHome(t); bd != nil {
		sut := g.makeNPC("smith", bd, g.W.cultByID[bd.Culture], r)
		sut.Title, sut.Faction = "army sutler", stateFaction(t.Realm)
		sut.Set(wsTag(t.ID))
		g.wsAt(zn, sut, cx, cy, 4, r)
	}
	if g.W.State[p] != t.Realm {
		wsBurn(zn, r, 0.22)
	}
}

func (g *Game) wsEventZone(zn *Zone, e *wsEvent, r *Rng) {
	p := g.W.Idx(zn.WX, zn.WY)
	switch e.Kind {
	case "invasion":
		n := r.Int(3, 5) + e.Str/2
		for i := 0; i < n; i++ {
			if a := g.soldier(e.Realm, r); a != nil {
				a.Set("wsinvader")
				g.wsAt(zn, a, 40, 12, 20, r)
			}
		}
		wsBurn(zn, r, 0.3)
		for i := 0; i < 4; i++ {
			if x, y, ok := zn.FreeSpot(r, nil); ok {
				zn.Set(x, y, TRubble)
			}
		}
	case "desertion":
		for i := 0; i < r.Int(3, 5); i++ {
			if a := g.soldier(e.Realm, r); a != nil {
				a.Faction, a.Title = "outlaws", "deserter"
				a.Set("wsband")
				a.Set("wsdes")
				g.wsAt(zn, a, 40, 12, 20, r)
			}
		}
	case "bandits":
		n := mini(1+e.Str/2, 7)
		cx, cy, ok := zn.FreeSpot(r, func(x, y int) bool { return x > 12 && x < MW-12 && y > 5 && y < MH-5 })
		if !ok {
			return
		}
		for i := 0; i < n; i++ {
			a := g.Spawn(Pick(r, []string{"bandit", "bandit", "archer"}), maxi(2, zn.Level+e.Str/4), r)
			a.Set("wsband")
			g.wsAt(zn, a, cx, cy, 5, r)
		}
		if e.Str >= 4 {
			zn.Set(cx, cy, TBrazier)
			for i := 0; i < e.Str/2; i++ {
				if x, y, ok := zn.FreeSpot(r, func(x, y int) bool { return abs(x-cx) <= 4 && abs(y-cy) <= 4 }); ok {
					zn.Set(x, y, TCrate)
				}
			}
		}
		if e.Str >= 7 {
			c := g.Spawn("chief", maxi(5, zn.Level+2), r)
			c.Set("wsband")
			g.wsAt(zn, c, cx, cy, 4, r)
		}
	case "slavepen":
		if e.P != p {
			return
		}
		cx, cy, ok := zn.FreeSpot(r, func(x, y int) bool { return x > 12 && x < MW-12 && y > 5 && y < MH-5 })
		if !ok {
			return
		}
		for i := 0; i < 3; i++ {
			a := g.Spawn(Pick(r, []string{"bandit", "archer"}), maxi(3, zn.Level), r)
			a.Name, a.Title = "slaver", "slaver"
			a.Set("slaver")
			a.Set("wspen")
			g.wsAt(zn, a, cx, cy, 5, r)
		}
		for _, o := range [][2]int{{-3, -2}, {3, -2}, {-3, 2}, {3, 2}} {
			zn.Set(cx+o[0], cy+o[1], TCrate)
		}
		g.wsCaptives(zn, 9000+e.ID, cx, cy, r.Int(2, 4), r)
		g.Msg("{{R|A slavers' pen: bound people huddle behind crates.}}")
	}
}

// ---- hooks: talk ----

func (g *Game) wsNearSlavers(a *Actor, rad int) bool {
	for _, o := range g.Cur.Actors {
		if o.HP > 0 && o.Has("slaver") && abs(o.X-a.X) <= rad && abs(o.Y-a.Y) <= rad {
			return true
		}
	}
	return false
}

func (g *Game) wsKnownSlavers() bool {
	p := g.W.Idx(g.Cur.WX, g.Cur.WY)
	h := g.wsHour()
	for _, t := range g.ws().Trav {
		if t.Kind == "slaver" && t.active(h) && t.Rep == 0 {
			if q, _ := t.at(h); wsCheb(g.W, q, p) <= 14 {
				return true
			}
		}
	}
	for _, e := range g.ws().Ev {
		if e.Kind == "slavepen" && e.Reported == 0 && wsCheb(g.W, e.P, p) <= 14 {
			return true
		}
	}
	return false
}

func (g *Game) wsReport(realmFaction string) {
	p := g.W.Idx(g.Cur.WX, g.Cur.WY)
	h := g.wsHour()
	n := 0
	for _, t := range g.ws().Trav {
		if t.Kind == "slaver" && t.active(h) && t.Rep == 0 {
			if q, _ := t.at(h); wsCheb(g.W, q, p) <= 14 {
				t.Rep = h + 1
				n++
			}
		}
	}
	for _, e := range g.ws().Ev {
		if e.Kind == "slavepen" && e.Reported == 0 && wsCheb(g.W, e.P, p) <= 14 {
			e.Reported = h + 1
			n++
		}
	}
	g.AddRep(realmFaction, 20, " for reporting slavers")
	g.gainXP(25)
	g.Msg(fmt.Sprintf("{{G|The guards take your report of %d slaving band(s) and promise riders within two days.}}", n))
}

func wsTalk(g *Game, a *Actor) []Option {
	if a.AI == "follower" || a.HP <= 0 || g.Cur == nil {
		return nil
	}
	var out []Option
	cur := g.Sk.Currency
	// the road gossip, from anyone who is not an enemy
	if !g.Hostile(a) && !a.Has("captive") && !a.Has("slaver") {
		out = append(out, Option{"What's happening on the roads?", func() { g.talkMenu(g.wsGossip()) }})
	}
	if a.Has("captive") {
		a := a
		out = append(out, Option{"{{G|Cut " + a.Name + "'s bonds.}}", func() { g.wsFree(a) }})
	}
	if a.Has("wsfreed") && a.Comp == nil {
		a := a
		if HashStr(fmt.Sprint(g.Seed, a.ID))%2 == 0 {
			out = append(out, Option{"{{G|Invite " + a.Name + " to travel with you.}}", func() {
				if g.companionCount() >= maxCompanions {
					g.talkMenu("\"I'd come, but you have company enough.\"")
					return
				}
				g.joinCompanion(a)
				g.Mode = "main"
			}})
		} else {
			out = append(out, Option{"Send them safely home.", func() {
				g.Msg(a.Name + " thanks you and heads for home.")
				a.AI = "wander"
				g.Mode = "main"
			}})
		}
	}
	if (a.Role == "guard" || isSoldier(a)) && !a.Has("slaver") && g.wsKnownSlavers() {
		fac := a.Faction
		out = append(out, Option{"{{Y|Report slavers on the roads.}}", func() {
			g.wsReport(fac)
			g.talkMenu("\"Slavers? In our lands? We'll see to it.\"")
		}})
	}
	if a.Has("wshire") {
		a := a
		fee := 60 + a.Level*20
		out = append(out, Option{fmt.Sprintf("{{W|Hire %s as a companion (%d %s).}}", a.Name, fee, cur), func() {
			switch {
			case g.companionCount() >= maxCompanions:
				g.talkMenu("\"You have company enough.\"")
			case g.P.Gold < fee:
				g.talkMenu("\"Come back with coin.\"")
			default:
				g.P.Gold -= fee
				a.Flags["wshire"] = false
				g.joinCompanion(a)
				g.Mode = "main"
			}
		}})
	}
	if a.Has("wstamer") {
		p := g.W.Idx(g.Cur.WX, g.Cur.WY)
		for _, b := range g.Cur.Actors {
			if b.HP > 0 && b.Has("wsbeast") {
				b := b
				fee := int(float64(40+b.Level*15) * g.wsPrice(p, "beasts"))
				out = append(out, Option{fmt.Sprintf("{{W|Buy the %s, trained to follow (%d %s).}}", strings.TrimPrefix(b.Name, "trained "), fee, cur), func() {
					switch {
					case g.companionCount() >= maxCompanions:
						g.talkMenu("\"You cannot keep another beast with you.\"")
					case g.P.Gold < fee:
						g.talkMenu("\"A good beast costs what it costs.\"")
					default:
						g.P.Gold -= fee
						b.Flags["wsbeast"] = false
						b.Set("wskeep")
						g.joinCompanion(b)
						g.talkMenu("\"Treat her well.\"")
					}
				}})
			}
		}
	}
	if a.Has("wspilgrim") && !a.Has("wsdone") {
		a := a
		out = append(out, Option{"{{C|Walk a while with the pilgrims.}}", func() {
			a.Set("wsdone")
			if a.Religion > 0 {
				g.AddRep(fmt.Sprintf("faith:%d", a.Religion), 15, " for sharing the road")
			}
			g.P.HP = mini(g.P.MaxHP, g.P.HP+g.P.MaxHP/5)
			g.talkMenu("They share bread and a blessing, and you set out lighter.")
		}})
	}
	if a.Has("wsminstrel") && !a.Has("wsdone") {
		a := a
		out = append(out, Option{"{{C|Hear a song.}}", func() {
			a.Set("wsdone")
			g.gainXP(15)
			g.talkMenu("A road song, wry and sad. " + g.wsGossip())
		}})
	}
	if a.Has("wsscholar") && !a.Has("wsdone") {
		a := a
		out = append(out, Option{"{{C|Ask the scholar about the lands round about.}}", func() {
			a.Set("wsdone")
			n := 0
			for _, b := range g.ws().burgList(g) {
				if !g.Known[b.P] && wsCheb(g.W, b.P, g.W.Idx(g.Cur.WX, g.Cur.WY)) <= 12 && n < 3 {
					g.Known[b.P] = true
					n++
				}
			}
			g.talkMenu(fmt.Sprintf("They sketch %d places you did not know on your map.", n))
		}})
	}
	if a.Has("wscaptain") {
		for _, t := range g.ws().Trav {
			if id, _ := wsTagOf(a); id == t.ID {
				t := t
				out = append(out, Option{"What is this army's purpose?", func() {
					g.talkMenu(fmt.Sprintf("\"We are the %s. Our road runs to %s.\"", t.Name, g.wsPlace(t.Path[len(t.Path)-1])))
				}})
			}
		}
	}
	if a.Has("wssick") {
		a := a
		out = append(out, Option{"{{G|Tend the sick with a tonic.}}", func() {
			for _, it := range g.P.Inv {
				if it.Kind == "tonic" && (it.Effect == "heal" || it.Effect == "antidote" || it.Effect == "salve") {
					g.takeOne(it)
					a.Flags["wssick"] = false
					a.F = roleColor[a.Role]
					g.AddRep(a.Faction, 15, " for tending the sick")
					g.talkMenu("The fever breaks. \"May the road be kind to you.\"")
					return
				}
			}
			g.talkMenu("You have no tonic to give.")
		}})
		if g.R.Chance(0.2) && g.P.St("plague") == 0 {
			g.P.AddSt("plague", 2000)
			g.Msg("{{R|Their breath is hot on your face; you feel the fever take you.}}")
		}
	}
	if g.P.St("plague") > 0 && (a.Role == "apothecary" || a.Role == "priest") {
		cost := 40 + g.P.Level*4
		out = append(out, Option{fmt.Sprintf("{{G|Treat the plague (%d %s).}}", cost, cur), func() {
			if g.P.Gold < cost {
				g.talkMenu("\"You can't afford the treatment.\"")
				return
			}
			g.P.Gold -= cost
			delete(g.P.Status, "plague")
			g.talkMenu("The fever leaves you, weak but clean.")
		}})
	}
	if a.Has("wswarden") {
		out = append(out, Option{"What of the quarantine?", func() {
			g.talkMenu("\"Keep clear of the sick. Their breath carries it. A healing tonic or an apothecary can ease the fever.\"")
		}})
	}
	return out
}

func (g *Game) wsFree(a *Actor) {
	if g.wsNearSlavers(a, 8) {
		g.talkMenu("\"Their keepers are watching! Deal with them first!\"")
		return
	}
	delete(a.Flags, "captive")
	a.Set("wsfreed")
	a.AI = "wander"
	a.Desc += " Freed."
	g.ws().Freed++
	g.AddRep(a.Faction, 40, " for freeing one of theirs")
	if a.Religion > 0 {
		g.AddRep(fmt.Sprintf("faith:%d", a.Religion), 20, " for freeing a believer")
	}
	g.gainXP(20)
	g.talkMenu("\"You've given me my life back. I will not forget you.\"")
}

// ---- hooks: kills ----

func wsKill(g *Game, d, src *Actor) {
	if g.Cur == nil || !g.W.In(g.Cur.WX, g.Cur.WY) {
		return
	}
	s := g.ws()
	p := g.W.Idx(g.Cur.WX, g.Cur.WY)
	if d.Has("wsband") && src == g.P {
		for i, e := range s.Ev {
			if e.Kind == "bandits" && e.covers(g.W, p) {
				e.Str--
				if e.Str <= 0 {
					s.Ev = append(s.Ev[:i], s.Ev[i+1:]...)
					g.Msg("{{G|The bandits of this country are broken.}}")
					g.wsLog(g.wsHour(), "Bandits near "+g.wsPlace(e.P)+" were cleared")
				}
				break
			}
		}
	}
	if d.Has("slaver") {
		for _, o := range g.Cur.Actors {
			if o != d && o.HP > 0 && o.Has("slaver") {
				return
			}
		}
		g.Msg("{{G|The slavers are dead. The captives can be freed.}}")
		if id, ok := wsTagOf(d); ok {
			if t := s.travByID(id); t != nil {
				t.Broken = true
			}
		}
		for _, e := range s.Ev {
			if e.Kind == "slavepen" && e.P == p {
				e.End = g.wsHour() + 1
			}
		}
		g.gainXP(40)
	}
}

// ---- gossip & news ----

type wsLine struct {
	P    int
	Text string
}

func (g *Game) wsNewsLines() []wsLine {
	s := g.ws()
	g.wsAdvance()
	h := g.wsHour()
	here := 0
	if g.Cur != nil && g.W.In(g.Cur.WX, g.Cur.WY) {
		here = g.W.Idx(g.Cur.WX, g.Cur.WY)
	}
	var out []wsLine
	for _, e := range s.Ev {
		var txt string
		switch e.Kind {
		case "plague":
			txt = fmt.Sprintf("{{R|Plague}} at %s, spreading (reach %d). The gates are shut and medicine is dear.", g.wsPlace(e.P), e.R)
		case "invasion":
			txt = fmt.Sprintf("{{R|Invasion}}: %s.", e.Name)
		case "desertion":
			txt = fmt.Sprintf("{{O|Deserters}}: %s.", e.Name)
		case "famine":
			txt = fmt.Sprintf("{{O|Famine}} grips %s; food and water are dear.", g.W.StateName(e.Realm))
		case "bandits":
			txt = fmt.Sprintf("{{O|Bandits}} thrive near %s (%d strong). Unchecked, they spread.", g.wsPlace(e.P), e.Str)
		case "boom":
			txt = fmt.Sprintf("{{G|Boom}}: %s.", e.Name)
		case "bust":
			txt = fmt.Sprintf("{{y|Bust}}: %s.", e.Name)
		case "slavepen":
			txt = fmt.Sprintf("{{R|Slavers}} are said to keep a pen near %s. Free the captives or tell the guards.", g.wsPlace(e.P))
		}
		out = append(out, wsLine{e.P, txt})
	}
	for _, t := range s.Trav {
		q, ok := t.at(h)
		if !ok || t.Done || t.Broken {
			continue
		}
		near := here >= 0 && wsCheb(g.W, q, here) <= 10
		switch {
		case t.Kind == "army":
			out = append(out, wsLine{q, fmt.Sprintf("{{R|An army}} of %s marches on %s, now near %s.", g.realmName(t.Realm), g.realmName(t.Foe), g.wsPlace(q))})
		case t.Kind == "slaver" && near:
			out = append(out, wsLine{q, fmt.Sprintf("{{R|A slavers' coffle}} was seen near %s.", g.wsPlace(q))})
		case near && (t.Kind == "caravan" || t.Kind == "merc" || t.Kind == "tamer"):
			out = append(out, wsLine{q, fmt.Sprintf("A %s, near %s.", t.Name, g.wsPlace(q))})
		}
	}
	return out
}

func (g *Game) wsGossip() string {
	lines := g.wsNewsLines()
	if len(lines) == 0 {
		return "\"The roads are quiet. Merchants complain of nothing but the weather.\""
	}
	here := g.W.Idx(g.Cur.WX, g.Cur.WY)
	sort.SliceStable(lines, func(i, j int) bool { return wsCheb(g.W, lines[i].P, here) < wsCheb(g.W, lines[j].P, here) })
	var out []string
	for i, l := range lines {
		if i >= 3 {
			break
		}
		out = append(out, l.Text+" ("+g.wsDirDist(here, l.P)+")")
	}
	return "\"What I hear on the roads:\"\n\n" + strings.Join(out, "\n")
}

func wsSide(g *Game) []string {
	if g.Cur == nil || !g.W.In(g.Cur.WX, g.Cur.WY) {
		return nil
	}
	var out []string
	if g.P.St("plague") > 0 {
		out = append(out, "{{R|Plague fever}}")
	}
	p := g.W.Idx(g.Cur.WX, g.Cur.WY)
	for _, e := range g.ws().Ev {
		if e.Kind == "plague" && e.covers(g.W, p) {
			out = append(out, "{{r|Plague in the land}}")
			break
		}
	}
	return out
}

func (g *Game) wsDrawNews() {
	s := g.sc
	x, y, w, h := g.panel("World news")
	lines := g.wsNewsLines()
	here := 0
	if g.Cur != nil && g.W.In(g.Cur.WX, g.Cur.WY) {
		here = g.W.Idx(g.Cur.WX, g.Cur.WY)
	}
	sort.SliceStable(lines, func(i, j int) bool { return wsCheb(g.W, lines[i].P, here) < wsCheb(g.W, lines[j].P, here) })
	var rows []string
	for _, l := range lines {
		rows = append(rows, Wrap(l.Text+" {{K|("+g.wsDirDist(here, l.P)+")}}", w)...)
	}
	if len(rows) == 0 {
		rows = append(rows, "{{K|Nothing stirs on the roads.}}")
	}
	rows = append(rows, "", "{{W|Markets here:}}")
	var mk []string
	for _, k := range wsKinds {
		m := g.wsPrice(here, k)
		col := "K"
		if m >= 1.15 {
			col = "R"
		} else if m <= 0.88 {
			col = "G"
		}
		mk = append(mk, fmt.Sprintf("{{%s|%s x%.2f}}", col, k, m))
	}
	rows = append(rows, strings.Join(mk, "  "))
	if lg := g.ws().Log; len(lg) > 0 {
		rows = append(rows, "", "{{W|Recently:}}")
		for i := len(lg) - 1; i >= 0 && i >= len(lg)-6; i-- {
			rows = append(rows, Wrap("{{K|"+lg[i]+"}}", w)...)
		}
	}
	vis := h - 2
	if g.Sel > len(rows)-vis {
		g.Sel = len(rows) - vis
	}
	if g.Sel < 0 {
		g.Sel = 0
	}
	for i := g.Sel; i < len(rows) && i < g.Sel+vis; i++ {
		s.TextB(x, y, rows[i], CGray, CBgDark, x+w)
		y++
	}
	s.TextB(x, y+0, "{{K|Up/Down scrolls, Esc closes}}", CGray, CBgDark, x+w)
}
