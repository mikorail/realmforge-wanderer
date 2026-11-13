package engine

import (
	"fmt"
	"strings"
)

// Genre signature: cyberpunk hacking (prefix gc; atompunk gets terminals, vending and doors lightly).
// Devices are tile interactions placed by a zone hook (side rng seed:gc:cy:zoneKey only, so other
// genres stay byte-identical): terminals/servers/ATMs/vending machines on TMachine, cameras on TPillar,
// ad boards on TInscription, security doors are any TLocked tile, turrets are real actors. Bumping one
// (gcBump, see the report for the one-line hook) opens the netrunning screen (mode "gch"): a layered
// node graph of ICE with strength, your Intrusion rating against it, a trace meter that rises every
// action. Breached nodes unlock payloads (doors, cameras and turrets, credits, data, virus, drone).
// Heat (state "gch", global) is corp and police attention: it decays by days and at thresholds spawns
// hostile corp security squads in towns; gcCyberHeatBounty feeds bhPlayerBounty.
// State: g.ext("gcc") planet-local (devices, blackouts), g.ext("gch") global (heat).

type gcDev struct {
	ID    string `json:"id"`
	K     string `json:"k"` // terminal server atm vend ad camera door
	X     int    `json:"x"`
	Y     int    `json:"y"`
	Tier  int    `json:"t"`
	Uses  int    `json:"u,omitempty"`
	Lock  int    `json:"l,omitempty"` // locked out until this game minute
	Spent bool   `json:"s,omitempty"` // payload drained (cash tray empty, camera looped)
}

type gcCyState struct {
	Devs map[string]*gcDev `json:"d"`
	Off  map[string]int    `json:"o"` // zone key -> cameras and turrets down until this game minute
	Seen map[string]bool   `json:"sn,omitempty"`
}

type gcHeatState struct {
	Heat   int             `json:"h"`
	Day    int             `json:"d"`
	Init   bool            `json:"i"`
	Bounty int             `json:"b"`
	Squad  int             `json:"sq"` // game minute of the last squad
	Tier   int             `json:"t"`
	Hacks  int             `json:"hk"`
	Firsts map[string]bool `json:"f"`
	Chip   int             `json:"c,omitempty"` // burner cartridge bonus for the next hack
	s      *gcSess
}

func (g *Game) gcCy() *gcCyState {
	st := g.ext("gcc", func() any { return &gcCyState{} }).(*gcCyState)
	if st.Devs == nil {
		st.Devs = map[string]*gcDev{}
	}
	if st.Off == nil {
		st.Off = map[string]int{}
	}
	if st.Seen == nil {
		st.Seen = map[string]bool{}
	}
	return st
}

func (g *Game) gcHt() *gcHeatState {
	st := g.ext("gch", func() any { return &gcHeatState{} }).(*gcHeatState)
	if st.Firsts == nil {
		st.Firsts = map[string]bool{}
	}
	return st
}

// gcCyGenre: 2 cyberpunk, 1 atompunk (terminals only, no corp squads), 0 elsewhere.
func gcCyGenre(g *Game) int {
	switch g.Genre {
	case "cyberpunk":
		return 2
	case "atompunk":
		return 1
	}
	return 0
}

// gcBumpers are tile-interaction handlers registered by the three gen_ files (see gcBump).
var gcBumpers []func(g *Game, x, y int) bool

// gcBump is the one hook for game.go MoveP: true when a device handled the bump.
func gcBump(g *Game, x, y int) bool {
	if g == nil || g.Cur == nil || g.Dead {
		return false
	}
	for _, f := range gcBumpers {
		if f(g, x, y) {
			return true
		}
	}
	return false
}

// gcUsers handle gcitem kinds from useItem.
var gcUsers []func(g *Game, it *Item) bool

// gcUseItem is the hook for powers.go useItem: true when it was one of our items.
func (g *Game) gcUseItem(it *Item) bool {
	if it == nil || it.Kind != "gcitem" {
		return false
	}
	for _, f := range gcUsers {
		if f(g, it) {
			return true
		}
	}
	return false
}

func gcID(zkey string, x, y int) string { return fmt.Sprintf("%s:%d", zkey, y*MW+x) }

func init() {
	unLocalKeys = append(unLocalKeys, "gcc")
	zoneHooks = append(zoneHooks, gcCyZone)
	turnHooks = append(turnHooks, gcCyTurn)
	sideHooks = append(sideHooks, gcCySide)
	talkHooks = append(talkHooks, gcCyTalk)
	gcBumpers = append(gcBumpers, gcCyBump)
	gcUsers = append(gcUsers, gcCyUse)
	modeKeys["gch"] = func(g *Game, k string) { g.keyMenu(k, func() { g.gcJackOut() }) }
	modeDraws["gch"] = func(g *Game) { g.gcMenuDraw() }
	modeKeys["gcm"] = func(g *Game, k string) { g.keyMenu(k, func() { g.gcClose() }) }
	modeDraws["gcm"] = func(g *Game) { g.gcMenuDraw() }
	for _, d := range []*skStatusDef{
		{ID: "gcdrone", Name: "Drone link", Glyph: 'd', Tick: func(g *Game, a *Actor) {
			if a.St("gcdrone") <= 1 && a.HP > 0 {
				g.hurt(a, a.HP+5, nil, "battery failure")
			}
		}},
	} {
		skAddStatus(d)
	}
}

// ---- shared menu screen (modes gch and gcm) ----

// gcMenu shows a titled text box with lettered options; the other gen_ files use it too.
func (g *Game) gcMenu(mode, title, text string, opts []Option) {
	g.PopT = title
	g.Popup = Wrap(text, 100)
	g.Opts = opts
	g.Sel = 0
	g.Mode = mode
}

func (g *Game) gcClose() {
	g.Mode, g.Talk, g.Popup, g.Opts = "main", nil, nil, nil
}

func (g *Game) gcMenuDraw() {
	s := g.sc
	h := len(g.Popup) + len(g.Opts) + 4
	y0 := maxi(1, (MH-h)/2)
	col := CBrown
	if g.Mode == "gch" {
		col = CCyan
	}
	s.Box(2, y0, SW-4, mini(h, SH-2), col, g.PopT)
	y := y0 + 1
	for _, l := range g.Popup {
		s.TextB(6, y, l, CGray, CBgDark, SW-3)
		y++
	}
	y++
	for i, o := range g.Opts {
		bg := CBgDark
		if i == g.Sel {
			bg = CBgSel
		}
		s.Fill(4, y, SW-8, 1, bg)
		s.TextB(4, y, fmt.Sprintf("{{W|%c)}} %s", 'a'+i, o.Text), CGray, bg, SW-3)
		y++
	}
}

func (g *Game) gcDeed(id, text string) {
	st := g.gcHt()
	if st.Firsts[id] {
		return
	}
	st.Firsts[id] = true
	g.deed("gc", fmt.Sprintf("The wanderer %s %s.", g.P.Name, text), 0, 0)
}

func gcFree(zn *Zone, r *Rng, near func(int, int) bool) (int, int, bool) {
	return zn.FreeSpot(r, func(x, y int) bool {
		if near != nil && !near(x, y) {
			return false
		}
		for _, it := range zn.Items {
			if it.X == x && it.Y == y {
				return false
			}
		}
		for _, d := range dirs8 {
			if t := zn.At(x+d[0], y+d[1]); t == TUp || t == TDown || t == TLocked || t == TDoor || t == TMachine || t == TInscription {
				return false
			}
		}
		return true
	})
}

// gcWallside: a free tile with a solid wall next to it (devices hug walls and stay out of the street).
func gcWallside(zn *Zone) func(int, int) bool {
	return func(x, y int) bool {
		for _, d := range dirs8 {
			if t := zn.At(x+d[0], y+d[1]); !t.D().Walk && t.D().Opaque {
				return true
			}
		}
		return false
	}
}

// ---- placement ----

var gcAds = []string{
	"{c} PRESENTS: SMILE.TM. A face worth ten thousand creds. Subscribe to yours.",
	"{c} CARES. Sleep is a productivity gap. Ask your cyberdoc about wakefulness chrome.",
	"BUY NOW, OWN NEVER. {c} leasing: your body, your flat, your debt.",
	"Citizen, your attention is valued at 0.4 cred a minute. Thank you for being watched.",
	"{c} SECURITY: if you see something, delete it.",
	"Loyalty is a feature. Upgrade to {c} PREMIUM.",
}

func (g *Game) gcCorpFor(bd *BurgData) string {
	if g.W.Forces == nil || len(g.W.Forces.Corps) == 0 {
		return "the Combine"
	}
	best := &g.W.Forces.Corps[0]
	for i := range g.W.Forces.Corps {
		c := &g.W.Forces.Corps[i]
		if bd != nil && c.HQ == bd.ID {
			return c.Name
		}
		if bd != nil && c.Host == bd.State && c.Influence > best.Influence {
			best = c
		}
	}
	return best.Name
}

func (g *Game) gcPlaceDev(zn *Zone, r *Rng, kind string, near func(int, int) bool, tierBonus int) *gcDev {
	x, y, ok := gcFree(zn, r, near)
	if !ok {
		return nil
	}
	return g.gcPutDev(zn, r, kind, x, y, tierBonus)
}

func (g *Game) gcPutDev(zn *Zone, r *Rng, kind string, x, y, tierBonus int) *gcDev {
	t := TMachine
	switch kind {
	case "camera":
		t = TPillar
	case "ad":
		t = TInscription
	}
	zn.Set(x, y, t)
	d := &gcDev{ID: gcID(zn.Key, x, y), K: kind, X: x, Y: y}
	d.Tier = clampi(1+zn.Level/6+tierBonus+r.Int(0, 1), 1, 5)
	if kind == "ad" || kind == "vend" {
		d.Tier = clampi(d.Tier-1, 1, 3)
	}
	if kind == "ad" {
		bd := g.W.burgByID[zn.Burg]
		if zn.Notes == nil {
			zn.Notes = map[int]string{}
		}
		zn.Notes[y*MW+x] = strings.ReplaceAll(Pick(r, gcAds), "{c}", g.gcCorpFor(bd))
	}
	g.gcCy().Devs[d.ID] = d
	return d
}

func gcCyZone(g *Game, zn *Zone, _ *Rng) {
	lv := gcCyGenre(g)
	if lv == 0 || g.W == nil || zn == nil {
		return
	}
	r := NewRng(g.Seed + ":gc:cy:" + zn.Key)
	switch {
	case zn.Kind == "town" && zn.Z == 0 && zn.Burg >= 0:
		g.gcCyTown(zn, r, lv)
	case zn.Kind == "dungeon" && zn.Z >= 1 && lv == 2:
		g.gcCyDungeon(zn, r)
	}
}

func (g *Game) gcCyTown(zn *Zone, r *Rng, lv int) {
	bd := g.W.burgByID[zn.Burg]
	if bd == nil || !bd.Alive {
		return
	}
	tier := g.W.cityTier[bd.ID]
	center := zn.ZX == 1 && zn.ZY == 1
	near := gcWallside(zn)
	if center {
		g.gcPlaceDev(zn, r, "terminal", near, tier/2)
		for i := 0; i < tier && lv == 2; i++ {
			g.gcPlaceDev(zn, r, "terminal", near, tier/2)
		}
		g.gcPlaceDev(zn, r, "vend", near, 0)
		if lv == 2 {
			if tier >= 1 || r.Chance(0.5) {
				g.gcPlaceDev(zn, r, "atm", near, tier/2)
			}
			for i := 0; i < 1+tier; i++ {
				g.gcPlaceDev(zn, r, "camera", nil, tier/2)
				g.gcPlaceDev(zn, r, "ad", near, 0)
			}
		}
		g.gcCyHQ(zn, r, bd)
		return
	}
	if r.Chance(0.3) {
		g.gcPlaceDev(zn, r, "vend", near, 0)
	}
	if lv == 2 {
		if r.Chance(0.3) {
			g.gcPlaceDev(zn, r, "ad", near, 0)
		}
		if r.Chance(0.25) {
			g.gcPlaceDev(zn, r, "camera", nil, tier/2)
		}
		if r.Chance(0.15) {
			g.gcPlaceDev(zn, r, "terminal", near, tier/2)
		}
	}
}

// gcCyHQ builds a corp server room (locked door, server, turrets, camera) in a corp's headquarters town.
// It keeps clear of the regional office socialworld.go builds at (4,2)-(12,6).
func (g *Game) gcCyHQ(zn *Zone, r *Rng, bd *BurgData) {
	if g.W.Forces == nil {
		return
	}
	var corp *CorpData
	for i := range g.W.Forces.Corps {
		if g.W.Forces.Corps[i].HQ == bd.ID {
			corp = &g.W.Forces.Corps[i]
		}
	}
	if corp == nil {
		return
	}
	const w, h = 11, 7
	for try := 0; try < 24; try++ {
		x0, y0 := r.Int(16, MW-w-3), r.Int(3, MH-h-3)
		ok := true
		for y := y0 - 1; y <= y0+h && ok; y++ {
			for x := x0 - 1; x <= x0+w && ok; x++ {
				t := zn.At(x, y)
				if !t.D().Walk || t.D().Water > 0 || t == TUp || t == TDown || t == TMachine || zn.ActorAt(x, y) != nil {
					ok = false
				}
			}
		}
		for _, it := range zn.Items {
			if it.X >= x0-1 && it.X <= x0+w && it.Y >= y0-1 && it.Y <= y0+h {
				ok = false
			}
		}
		if !ok {
			continue
		}
		for y := y0; y < y0+h; y++ {
			for x := x0; x < x0+w; x++ {
				if y == y0 || y == y0+h-1 || x == x0 || x == x0+w-1 {
					zn.Set(x, y, TMetalWall)
				} else {
					zn.Set(x, y, TTileFloor)
				}
			}
		}
		dx, dy := x0+w/2, y0+h-1
		zn.Set(dx, dy, TLocked)
		sx, sy := x0+w/2, y0+2
		d := g.gcPutDev(zn, r, "server", sx, sy, 2)
		if d != nil {
			d.Tier = clampi(d.Tier+1, 3, 5)
			zn.Notes[sy*MW+sx] = corp.Name + " root server. Tier-" + fmt.Sprint(d.Tier) + " ICE. Unauthorised access is a performance issue."
		}
		g.gcPutDev(zn, r, "camera", dx+2, dy+1, 1)
		for i := 0; i < 2; i++ {
			a := g.Spawn("turret", zn.Level+4, r)
			a.Set("gcsec")
			a.X, a.Y, a.HomeX, a.HomeY = x0+2+i*(w-5), y0+2+r.Int(0, 2), 0, 0
			a.HomeX, a.HomeY = a.X, a.Y
			if zn.At(a.X, a.Y) == TTileFloor && zn.ActorAt(a.X, a.Y) == nil {
				zn.Actors = append(zn.Actors, a)
			}
		}
		return
	}
}

func (g *Game) gcCyDungeon(zn *Zone, r *Rng) {
	s := g.siteAt(zn.WX, zn.WY, zn.ZX, zn.ZY)
	if s == nil || (s.Theme != "hideout" && s.Theme != "vault") {
		return
	}
	deep := zn.Z >= s.Depth
	if r.Chance(0.55) || deep {
		g.gcPlaceDev(zn, r, "terminal", gcWallside(zn), 1+zn.Z/3)
	}
	if r.Chance(0.5) {
		g.gcPlaceDev(zn, r, "camera", nil, 1)
	}
	if r.Chance(0.45) || deep {
		if x, y, ok := gcFree(zn, r, nil); ok {
			a := g.Spawn("turret", zn.Level+2, r)
			a.Set("gcsec")
			a.X, a.Y, a.HomeX, a.HomeY = x, y, x, y
			zn.Actors = append(zn.Actors, a)
		}
	}
}

// ---- finding the device under a bump ----

func (g *Game) gcDevAt(zn *Zone, x, y int) *gcDev {
	st := g.gcCy()
	id := gcID(zn.Key, x, y)
	if d := st.Devs[id]; d != nil {
		return d
	}
	if gcCyGenre(g) == 0 {
		return nil
	}
	kind := ""
	switch zn.At(x, y) {
	case TMachine:
		kind = "terminal"
		if strings.Contains(zn.Notes[y*MW+x], "regional office") {
			kind = "server"
		}
	case TLocked:
		kind = "door"
	default:
		return nil
	}
	h := NewRng(g.Seed + ":gc:dev:" + id)
	d := &gcDev{ID: id, K: kind, X: x, Y: y, Tier: clampi(1+zn.Level/6+h.Int(0, 1), 1, 5)}
	if kind == "server" {
		d.Tier = clampi(d.Tier+2, 3, 5)
	}
	st.Devs[id] = d
	return d
}

var gcDevName = map[string]string{
	"terminal": "data terminal", "server": "corporate server", "atm": "ATM", "vend": "vending machine",
	"ad": "holo-ad board", "camera": "security camera", "door": "security door",
}

func (g *Game) gcDevLabel(d *gcDev) string {
	n := gcDevName[d.K]
	if gcCyGenre(g) == 1 {
		switch d.K {
		case "terminal":
			n = "teletype console"
		case "vend":
			n = "automat"
		case "door":
			n = "vault door"
		}
	}
	return n
}

// ---- bump: the device menu ----

func gcCyBump(g *Game, x, y int) bool {
	zn := g.Cur
	if gcCyGenre(g) == 0 || zn == nil {
		return false
	}
	d := g.gcDevAt(zn, x, y)
	if d == nil {
		return false
	}
	g.gcDevMenu(d)
	return true
}

func (g *Game) gcDevMenu(d *gcDev) {
	zn := g.Cur
	name := g.gcDevLabel(d)
	text := fmt.Sprintf("A %s, security tier %d.", name, d.Tier)
	var opts []Option
	if d.Lock > g.Turn {
		text += " It has locked you out and glares red for another " + fmt.Sprint(d.Lock-g.Turn) + " minutes."
	} else if d.Spent && d.K != "door" {
		text += " Its cache is drained; nothing more to take."
	} else {
		opts = append(opts, Option{"{{C|Jack in and hack it.}} {{K|(Intrusion rating " + fmt.Sprint(g.gcRating()) + ")}}", func() { g.gcStart(d) }})
	}
	if n := zn.Notes[d.Y*MW+d.X]; n != "" {
		opts = append(opts, Option{"Read the screen.", func() { g.showPopup(name, n) }})
	}
	if d.K == "door" {
		opts = append(opts, Option{"Try the lock the old way.", func() {
			g.gcClose()
			if !g.lkBumpLocked(d.X, d.Y) {
				g.unlock(d.X, d.Y)
			}
		}})
	}
	opts = append(opts, Option{"Walk away.", func() { g.gcClose() }})
	g.gcMenu("gcm", strings.ToUpper(name), text, opts)
}

// ---- intrusion rating ----

func (g *Game) gcHasAug(id string) bool {
	for _, p := range g.P.Body {
		if p.Aug != nil && !p.Severed && p.Aug.Effect == id {
			return true
		}
	}
	return false
}

func (g *Game) gcDeck() *Item {
	var best *Item
	for _, it := range g.P.Inv {
		if it.Kind == "gcitem" && it.Effect == "deck" && it.Charges > 0 && (best == nil || it.Power > best.Power) {
			best = it
		}
	}
	return best
}

// gcRating: tinkering check modifier (INT + 2 per rank) + deck quality + head augments + burner chip.
func (g *Game) gcRating() int {
	n := g.checkMod("tinkering")
	if d := g.gcDeck(); d != nil {
		n += d.Power
	} else if gcCyGenre(g) == 2 {
		n -= 2 // bare-handed jack through a socket
	}
	if g.gcHasAug("cortex") {
		n++
	}
	if g.gcHasAug("reflex") {
		n++
	}
	if g.gcHasAug("mindward") {
		n++
	}
	return n + g.gcHt().Chip
}

// ---- the node graph ----

type gcNode struct {
	Name string
	Kind string // gate fw ice tracer data ctrl bank core
	Str  int
	Lvl  int
	Open bool
	Seen bool
}

type gcSess struct {
	D      *gcDev
	N      []*gcNode
	Trace  int
	Spoofs int
	Acts   int
	Log    []string
	Used   map[string]bool
	Over   bool
	Ghost  bool
}

var gcTpl = map[string][][]string{
	"ad":       {{"gate"}, {"bank"}},
	"vend":     {{"gate"}, {"fw"}, {"bank"}},
	"atm":      {{"gate"}, {"fw"}, {"ice", "tracer"}, {"bank"}},
	"door":     {{"gate"}, {"ctrl"}},
	"camera":   {{"gate"}, {"ctrl"}},
	"terminal": {{"gate"}, {"fw", "tracer"}, {"ctrl", "data"}, {"core"}},
	"server":   {{"gate"}, {"fw", "ice"}, {"tracer", "fw"}, {"data", "bank", "ctrl"}, {"core"}},
}

var gcKindShort = map[string]string{"gate": "GATE", "fw": "FWLL", "ice": "BICE", "tracer": "TRAC", "data": "DATA", "ctrl": "CTRL", "bank": "BANK", "core": "CORE"}
var gcKindLong = map[string]string{"gate": "access gate", "fw": "firewall", "ice": "black ICE", "tracer": "trace daemon", "data": "data cache", "ctrl": "control bus", "bank": "ledger", "core": "root core"}
var gcKindOff = map[string]int{"gate": -1, "fw": 0, "ice": 1, "tracer": 0, "data": 0, "ctrl": 0, "bank": 1, "core": 1}

func (g *Game) gcStart(d *gcDev) {
	ht := g.gcHt()
	r := NewRng(fmt.Sprintf("%s:gc:hack:%s:%d", g.Seed, d.ID, d.Uses))
	tpl := gcTpl[d.K]
	if tpl == nil {
		tpl = gcTpl["terminal"]
	}
	s := &gcSess{D: d, Used: map[string]bool{}, Spoofs: 2 + maxi(0, g.checkMod("tinkering")/3)}
	for lv, kinds := range tpl {
		for _, k := range kinds {
			n := &gcNode{Kind: k, Lvl: lv, Name: gcKindLong[k]}
			n.Str = clampi(d.Tier+gcKindOff[k]+r.Int(-1, 1), 1, 9)
			s.N = append(s.N, n)
		}
	}
	s.N[0].Open = true
	s.N[0].Seen = true
	s.Log = []string{"You jack in. The " + g.gcDevLabel(d) + " wakes: a lattice of ICE."}
	ht.s = s
	d.Uses++
	g.gcShow()
}

func (s *gcSess) reachable(n *gcNode) bool {
	if n.Open {
		return false
	}
	if n.Lvl == 0 {
		return true
	}
	for _, m := range s.N {
		if m.Lvl == n.Lvl-1 && m.Open {
			return true
		}
	}
	return false
}

func (s *gcSess) cols() int {
	c := 0
	for _, n := range s.N {
		c = maxi(c, n.Lvl+1)
	}
	return c
}

func (s *gcSess) graph() []string {
	var out []string
	rows := 0
	for l := 0; l < s.cols(); l++ {
		k := 0
		for _, n := range s.N {
			if n.Lvl == l {
				k++
			}
		}
		rows = maxi(rows, k)
	}
	for row := 0; row < rows; row++ {
		line := ""
		for l := 0; l < s.cols(); l++ {
			var n *gcNode
			i := 0
			for _, m := range s.N {
				if m.Lvl == l {
					if i == row {
						n = m
					}
					i++
				}
			}
			if n == nil {
				line += "             "
				continue
			}
			st := "?"
			if n.Seen {
				st = fmt.Sprint(n.Str)
			}
			col := "K"
			switch {
			case n.Open:
				col, st = "G", "ok"
			case s.reachable(n):
				col = "Y"
				if n.Kind == "tracer" || n.Kind == "ice" {
					col = "R"
				}
			}
			cell := fmt.Sprintf("[%s %-2s]", gcKindShort[n.Kind], st)
			line += "{{" + col + "|" + cell + "}}"
			if l < s.cols()-1 {
				line += "{{K|─▶}} "
			}
		}
		out = append(out, line)
	}
	return out
}

func (s *gcSess) bar() string {
	n := clampi(s.Trace/5, 0, 20)
	col := "G"
	if s.Trace >= 40 {
		col = "Y"
	}
	if s.Trace >= 70 {
		col = "R"
	}
	return fmt.Sprintf("Trace {{%s|[%s%s] %d%%}}", col, strings.Repeat("█", n), strings.Repeat("░", 20-n), s.Trace)
}

func (g *Game) gcShow() {
	ht := g.gcHt()
	s := ht.s
	if s == nil {
		g.gcClose()
		return
	}
	lines := []string{"{{C|Rating " + fmt.Sprint(g.gcRating()) + "}}   " + s.bar() + fmt.Sprintf("   {{K|spoofs %d}}", s.Spoofs), ""}
	lines = append(lines, s.graph()...)
	lines = append(lines, "")
	from := maxi(0, len(s.Log)-4)
	for _, l := range s.Log[from:] {
		lines = append(lines, "{{K|> "+l+"}}")
	}
	var opts []Option
	for _, n := range s.N {
		n := n
		if !s.reachable(n) {
			continue
		}
		if !n.Seen {
			opts = append(opts, Option{fmt.Sprintf("Probe the %s {{K|(+2 trace)}}", n.Name), func() { g.gcAct(n, "probe") }})
		}
		opts = append(opts, Option{fmt.Sprintf("Bypass the %s {{K|(quiet)}}", n.Name), func() { g.gcAct(n, "bypass") }})
		opts = append(opts, Option{fmt.Sprintf("Crash the %s {{R|(loud, feedback on failure)}}", n.Name), func() { g.gcAct(n, "crash") }})
	}
	if s.Spoofs > 0 && s.Trace > 0 {
		opts = append(opts, Option{"{{c|Spoof the trace: scrub the route.}}", func() { g.gcAct(nil, "spoof") }})
	}
	opts = append(opts, g.gcPayloads(s)...)
	opts = append(opts, Option{"{{K|Jack out.}}", func() { g.gcJackOut() }})
	g.gcMenu("gch", "NETRUN: "+strings.ToUpper(g.gcDevLabel(s.D)), strings.Join(lines, "\n"), opts)
}

func (s *gcSess) log(t string) { s.Log = append(s.Log, t) }

func (g *Game) gcAct(n *gcNode, act string) {
	s := g.gcHt().s
	if s == nil {
		return
	}
	s.Acts++
	rt := g.gcRating()
	switch act {
	case "probe":
		n.Seen = true
		s.Trace += 2
		s.log(fmt.Sprintf("Probe: the %s is strength %d.", n.Name, n.Str))
	case "bypass":
		roll := g.R.Dice(2, 6) + rt
		if roll >= 6+n.Str {
			n.Open, n.Seen = true, true
			s.Trace += 3
			s.log(fmt.Sprintf("You slip past the %s without a ripple.", n.Name))
		} else {
			s.Trace += 7
			s.log(fmt.Sprintf("The %s throws you back and pings the trace.", n.Name))
			g.gcBite(s, n, false)
		}
	case "crash":
		roll := g.R.Dice(2, 6) + rt + 2
		if roll >= 4+n.Str {
			n.Open, n.Seen = true, true
			s.Trace += 9
			s.log(fmt.Sprintf("You smash the %s; alarms bloom down the line.", n.Name))
		} else {
			s.Trace += 12
			s.log(fmt.Sprintf("The %s holds and bites back.", n.Name))
			g.gcBite(s, n, true)
		}
	case "spoof":
		s.Spoofs--
		if g.R.Dice(2, 6)+rt >= 8 {
			s.Trace = maxi(0, s.Trace-14)
			s.log("Spoof: the route scrambles and the trace loses your scent.")
		} else {
			s.Trace += 3
			s.log("The spoof is sloppy; the trace tightens.")
		}
	}
	for _, m := range s.N {
		if m.Kind == "tracer" && !m.Open && (m.Lvl == 0 || s.reachable(m)) {
			s.Trace += 3
			s.log("The trace daemon ticks up.")
			break
		}
	}
	if s.Trace >= 100 {
		s.Trace = 100
		g.gcTraced(s)
		return
	}
	if g.Dead {
		return
	}
	g.gcShow()
}

// gcBite: ICE feedback and a worn deck after a failed breach.
func (g *Game) gcBite(s *gcSess, n *gcNode, loud bool) {
	dmg := 0
	if n.Kind == "ice" {
		dmg = g.R.Int(1, 4) + n.Str/2
	} else if loud {
		dmg = g.R.Int(1, 3)
	}
	if dmg > 0 {
		g.hurt(g.P, dmg, nil, "neural feedback")
		s.log(fmt.Sprintf("Feedback burns through your skull (-%d).", dmg))
	}
	if d := g.gcDeck(); d != nil && (loud || n.Kind == "ice") && g.R.Chance(0.45) {
		d.Charges--
		if d.Charges <= 0 {
			g.removeInv(d)
			g.Msg("{{R|Your cyberdeck cooks itself: burnt out.}}")
			s.log("Your deck is burnt out.")
		}
	}
}

// ---- payloads ----

func (g *Game) gcPayloads(s *gcSess) []Option {
	open := map[string]bool{}
	for _, n := range s.N {
		if n.Open {
			open[n.Kind] = true
		}
	}
	d := s.D
	var out []Option
	add := func(id, text string, f func()) {
		if s.Used[id] {
			return
		}
		out = append(out, Option{"{{G|" + text + "}}", func() {
			s.Used[id] = true
			s.Trace += 5
			s.Acts++
			f()
			if g.Dead {
				return
			}
			if s.Trace >= 100 {
				s.Trace = 100
				g.gcTraced(s)
				return
			}
			g.gcShow()
		}})
	}
	if open["ctrl"] {
		switch d.K {
		case "camera":
			add("cam", "Loop the feed: blind cameras and stun turrets here for 45 minutes.", func() { g.gcSecurityOff(45) })
		case "door":
			add("doors", "Unseal the door and the locks near it.", g.gcUnseal)
		default:
			add("doors", "Unseal every lock near here.", g.gcUnseal)
			add("cam", "Blind cameras and stun turrets here for 45 minutes.", func() { g.gcSecurityOff(45) })
		}
	}
	if open["data"] {
		add("data", "Download the data cache.", func() { g.gcDownload() })
	}
	if open["bank"] {
		add("cash", "Siphon credits.", func() { g.gcSiphon() })
	}
	if open["core"] {
		add("virus", "Plant a virus: scrub your logs, blind security for 12 hours.", func() { g.gcVirus() })
		add("drone", "Seize a maintenance drone as an ally.", func() { g.gcDrone() })
	}
	return out
}

func (g *Game) gcSecurityOff(mins int) {
	st := g.gcCy()
	st.Off[g.Cur.Key] = g.Turn + mins
	n := 0
	for _, a := range g.Cur.Actors {
		if a.HP > 0 && (a.Has("gcsec") || a.Tpl == "turret") {
			a.AddSt("stunned", mini(mins, 90))
			n++
		}
	}
	g.gcHt().s.log(fmt.Sprintf("Cameras dark, %d turrets stunned.", n))
	g.gcAddHeat(4, "")
	g.gcDeed("secoff", "blinded a zone's cameras and turrets with a hack")
}

func (g *Game) gcUnseal() {
	zn := g.Cur
	d := g.gcHt().s.D
	n := 0
	for y := d.Y - 12; y <= d.Y+12; y++ {
		for x := d.X - 12; x <= d.X+12; x++ {
			if zn.At(x, y) == TLocked {
				zn.Set(x, y, TDoorOpen)
				n++
			}
		}
	}
	if d.K == "door" {
		d.Spent = true
	}
	g.gcHt().s.log(fmt.Sprintf("%d locks slide open.", n))
	g.gcAddHeat(3, "")
	g.gcDeed("unseal", "talked a security door open by hand")
}

func (g *Game) gcSiphon() {
	s := g.gcHt().s
	d := s.D
	base := map[string]int{"ad": 8, "vend": 12, "atm": 55, "terminal": 30, "server": 90, "camera": 5, "door": 5}[d.K]
	amt := base * (1 + d.Tier) / 2
	amt += g.R.Int(0, maxi(2, amt/3))
	got := g.cuEarn(amt)
	d.Spent = true
	s.log("The ledger coughs up " + got + ".")
	g.Msg("{{Y|You siphon " + got + ".}}")
	g.gcAddHeat(4+d.Tier*2, "")
	g.gcDeed("siphon", "siphoned credits out of a machine")
	g.gainXP(6 + 6*d.Tier)
}

func (g *Game) gcDownload() {
	s := g.gcHt().s
	d := s.D
	r := NewRng(fmt.Sprintf("%s:gc:data:%s:%d", g.Seed, d.ID, d.Uses))
	text := "Encrypted payroll and nothing else. A fence might still pay for it."
	title := "datashard: payroll dump"
	w := g.W
	if len(w.Burgs) > 0 {
		for try := 0; try < 12; try++ {
			bd := &w.Burgs[r.Int(0, len(w.Burgs)-1)]
			if !bd.Alive || bd.Layer != "surface" || g.Known[bd.P] {
				continue
			}
			g.Known[bd.P] = true
			text = fmt.Sprintf("Courier manifest, route plans and a map tile: the settlement of %s lies under %s. It is now marked on your world map.", bd.Name, g.FactionName(uwFac(bd.State)))
			title = "datashard: courier manifest"
			break
		}
	}
	if w.Forces != nil && len(w.Forces.Corps) > 0 && r.Chance(0.5) {
		c := w.Forces.Corps[r.Int(0, len(w.Forces.Corps)-1)]
		bn := ""
		if b := w.burgByID[c.HQ]; b != nil {
			bn = b.Name
		}
		text += fmt.Sprintf(" Memo: %s keeps its servers in %s; influence index %d.", c.Name, bn, c.Influence)
		title = "datashard: " + c.Name + " memo"
	}
	it := &Item{ID: g.newID(), Kind: "book", Name: title, Glyph: '■', F: CCyan, Text: text, Value: 25 + 22*d.Tier + r.Int(0, 20), Weight: 0, Ident: true,
		Lore: "A shard of stolen data. Read it, or sell it to a fence."}
	g.give(it)
	d.Spent = true
	s.log("The cache decrypts onto a shard.")
	g.Msg("{{C|You pull a " + title + ".}}")
	g.gcAddHeat(5, "")
	g.gcDeed("download", "stole a datashard from a machine")
	g.gainXP(5 + 5*d.Tier)
}

func (g *Game) gcVirus() {
	st := g.gcCy()
	st.Off[g.Cur.Key] = g.Turn + 720
	for _, a := range g.Cur.Actors {
		if a.HP > 0 && (a.Has("gcsec") || a.Tpl == "turret") {
			a.AddSt("stunned", 90)
		}
	}
	ht := g.gcHt()
	ht.Heat = maxi(0, ht.Heat-12)
	ht.s.log("The virus eats the audit logs. Security stays blind.")
	g.gcDeed("virus", "planted a virus in a machine")
	g.gainXP(10 + 8*ht.s.D.Tier)
}

func (g *Game) gcDrone() {
	ht := g.gcHt()
	n := 0
	for _, a := range g.Cur.Actors {
		if a.St("gcdrone") > 0 && a.HP > 0 {
			n++
		}
	}
	if n >= 2 {
		ht.s.log("You already control two drones; the link has no bandwidth for a third.")
		return
	}
	p := g.P
	x, y, ok := g.nearestFree(g.Cur, p.X+1, p.Y)
	if !ok {
		ht.s.log("No room for a drone to land.")
		return
	}
	lvl := maxi(2, p.Level-1+ht.s.D.Tier/2)
	c := &Actor{ID: g.newID(), Name: "hijacked drone", Glyph: 'd', F: CCyan, Tpl: "gcdrone", Faction: "player", Level: lvl,
		Attr: [6]int{12, 16, 12, 6, 10, 6}, MaxHP: 14 + lvl*4, AVNat: 2 + lvl/3, DVNat: 4 + lvl/2, Speed: 110, AI: "follower",
		Body: MakeBody("humanoid"), X: x, Y: y, Desc: "A maintenance drone with your handshake burned into it."}
	c.HP = c.MaxHP
	c.Set("armed")
	if hp := c.PartFor("hand"); hp != nil {
		hp.Natural = natural("arc cutter", 1, 6, 2+lvl/4, "shock")
	}
	c.Status = map[string]int{"gcdrone": 240}
	g.Cur.Actors = append(g.Cur.Actors, c)
	ht.s.log("A drone rotors up and falls in behind you.")
	g.gcDeed("drone", "seized a drone as an ally by hacking")
	g.gainXP(8 + 6*ht.s.D.Tier)
}

// ---- ending a run ----

func (g *Game) gcJackOut() {
	ht := g.gcHt()
	s := ht.s
	ht.s = nil
	g.gcClose()
	if s == nil {
		return
	}
	ht.Chip = 0
	ht.Hacks++
	acts := maxi(1, s.Acts)
	if !s.Over {
		g.gcAddHeat(s.Trace/12, "")
		if s.Trace < 30 && acts > 1 {
			g.gcDeed("ghost", "jacked out of a hack before the trace found a thing")
		}
		g.Msg("{{c|You jack out, eyes watering.}}")
		g.gcDeed("hack", "broke into a machine for the first time")
	}
	g.spend(1000 * mini(acts, 8))
}

func (g *Game) gcTraced(s *gcSess) {
	ht := g.gcHt()
	ht.s = nil
	s.Over = true
	d := s.D
	d.Lock = g.Turn + 240
	zn := g.Cur
	g.gcClose()
	g.Msg("{{R|TRACED. The system screams your name into every channel.}}")
	dmg := g.R.Dice(2, 6) + d.Tier
	g.hurt(g.P, dmg, nil, "neural feedback")
	if g.Dead {
		return
	}
	if dk := g.gcDeck(); dk != nil {
		dk.Charges -= 3
		if dk.Charges <= 0 {
			g.removeInv(dk)
			g.Msg("{{R|Your cyberdeck burns out in your hands.}}")
		}
	} else {
		g.P.AddSt("stunned", 3)
	}
	g.gcAddHeat(18+d.Tier*5, "")
	state := g.uwStateHere()
	g.uwAddBounty(state, 6+d.Tier*3, " (intrusion)")
	if gcCyGenre(g) == 2 && zn != nil {
		if zn.Kind == "town" {
			g.gcSquad(1+d.Tier/3, false)
		} else {
			for i := 0; i < 1+d.Tier/3; i++ {
				if x, y, ok := zn.FreeSpot(g.R, func(x, y int) bool { return abs(x-g.P.X) >= 5 && abs(x-g.P.X) <= 12 }); ok {
					a := g.Spawn("turret", zn.Level+d.Tier, g.R)
					a.Set("gcsec")
					a.X, a.Y, a.HomeX, a.HomeY = x, y, x, y
					zn.Actors = append(zn.Actors, a)
					g.Msg("{{R|A defence turret drops out of the ceiling.}}")
				}
			}
		}
	}
	g.spend(3000)
}

// ---- heat ----

func gcHeatName(h int) string {
	switch {
	case h >= 70:
		return "Priority target"
	case h >= 40:
		return "Flagged"
	case h >= 15:
		return "Noticed"
	}
	return "Quiet"
}

func gcHeatTier(h int) int {
	switch {
	case h >= 70:
		return 3
	case h >= 40:
		return 2
	case h >= 15:
		return 1
	}
	return 0
}

func (g *Game) gcAddHeat(n int, _ string) {
	if gcCyGenre(g) == 0 || n == 0 {
		return
	}
	ht := g.gcHt()
	ht.Heat = clampi(ht.Heat+n, 0, 100)
	if t := gcHeatTier(ht.Heat); t > ht.Tier && n > 0 {
		g.Msg("{{R|Heat: " + gcHeatName(ht.Heat) + ". The corporations have your handle.}}")
	}
	ht.Tier = gcHeatTier(ht.Heat)
}

// gcCyberHeatBounty: heat as bounty units for bhPlayerBounty (hunts.go): hunters track you while it burns.
func gcCyberHeatBounty(g *Game) int {
	if g == nil || g.Genre != "cyberpunk" {
		return 0
	}
	h := g.gcHt().Heat
	if h < 25 {
		return 0
	}
	return (h - 20) * 3
}

func (g *Game) gcSquad(n int, captain bool) {
	zn := g.Cur
	bd := g.localBurg()
	if zn == nil || bd == nil {
		return
	}
	cu := g.W.cultByID[bd.Culture]
	corp := g.gcCorpFor(bd)
	have := 0
	for _, a := range zn.Actors {
		if a.HP > 0 && a.Has("gcsquad") {
			have++
		}
	}
	if have >= 6 {
		return
	}
	made := 0
	for i := 0; i < n; i++ {
		x, y, ok := zn.FreeSpot(g.R, func(x, y int) bool {
			d := maxi(abs(x-g.P.X), abs(y-g.P.Y))
			return d >= 7 && d <= 22
		})
		if !ok {
			break
		}
		a := g.makeNPC("guard", bd, cu, g.R)
		a.Faction = "corp"
		a.Name, a.Proper = corp+" security", false
		a.Title = "corporate security"
		a.Glyph, a.F = 'S', CCyan
		a.AI, a.Angry, a.State = "hostile", true, 0
		a.Desc = "Corporate security in visored riot armour, a company sigil on the shoulder, a tracker pinging your face."
		a.Set("gcsquad")
		a.Set("gcsec")
		if captain && i == 0 {
			a.Name = corp + " response captain"
			a.Level += 3
			a.MaxHP += 25
			a.Boss = true
		}
		a.HP = a.MaxHP
		a.X, a.Y, a.HomeX, a.HomeY = x, y, x, y
		zn.Actors = append(zn.Actors, a)
		made++
	}
	if made > 0 {
		g.Msg("{{R|" + corp + " security squad moves in on you.}}")
		g.gcDeed("squad", "drew a corporate security squad down on their head")
	}
}

func gcCyTurn(g *Game) {
	lv := gcCyGenre(g)
	if lv == 0 || g.Dead || g.Cur == nil || g.Mode == "world" {
		return
	}
	ht := g.gcHt()
	if d := g.Day(); !ht.Init || d != ht.Day {
		if ht.Init && d > ht.Day {
			ht.Heat = maxi(0, ht.Heat-5*mini(d-ht.Day, 10))
			ht.Tier = gcHeatTier(ht.Heat)
		}
		ht.Init, ht.Day = true, d
	}
	zn := g.Cur
	// crimes the law notes also burn heat (bounty units rose since the last look)
	if lv == 2 && g.Turn%5 == 0 {
		sum := 0
		for _, b := range g.uwSt().Bounty {
			sum += b
		}
		if sum > ht.Bounty {
			g.gcAddHeat((sum-ht.Bounty)/6+1, "")
		}
		ht.Bounty = sum
	}
	// cameras watch a flagged wanderer
	st := g.gcCy()
	if g.Turn%10 == 0 && ht.Heat >= 10 && st.Off[zn.Key] <= g.Turn {
		for _, d := range st.Devs {
			if d.K == "camera" && strings.HasPrefix(d.ID, zn.Key+":") && g.visible(d.X, d.Y) && maxi(abs(d.X-g.P.X), abs(d.Y-g.P.Y)) <= 12 {
				if !st.Seen[zn.Key] {
					st.Seen[zn.Key] = true
					g.Msg("{{R|A camera swivels and holds on your face.}}")
				}
				g.gcAddHeat(1, "")
				break
			}
		}
	}
	// squads in towns
	if lv == 2 && zn.Kind == "town" && zn.Z == 0 && ht.Heat >= 25 && g.Turn%7 == 0 {
		t := gcHeatTier(ht.Heat)
		cool := []int{0, 240, 150, 90}[t]
		if g.Turn-ht.Squad >= cool && g.R.Chance(0.25) {
			ht.Squad = g.Turn
			g.gcSquad(1+t, t >= 3)
		}
	}
}

func gcCySide(g *Game) []string {
	if gcCyGenre(g) != 2 {
		return nil
	}
	ht := g.gcHt()
	if ht.Heat < 5 {
		return nil
	}
	col := "Y"
	if ht.Heat >= 40 {
		col = "R"
	}
	return []string{fmt.Sprintf("{{%s|Heat %d %s}}", col, ht.Heat, gcHeatName(ht.Heat))}
}

// ---- items and merchants ----

func (g *Game) gcMakeDeck(q int) *Item {
	q = clampi(q, 1, 4)
	names := []string{"", "Kiwi-3 cyberdeck", "Hosaka Ono-Sendai deck", "Zetatech Blackdeck", "Mil-spec ghost rig"}
	return &Item{ID: g.newID(), Kind: "gcitem", Effect: "deck", Name: names[q], Glyph: '▫', F: CCyan, Power: q, Charges: 8 + 3*q, Value: 90 * q * q, Weight: 2, Ident: true,
		Lore: fmt.Sprintf("A netrunner's deck. Adds %d to your Intrusion rating while it has integrity left (%d).", q, 8+3*q)}
}

func (g *Game) gcMakeChip() *Item {
	return &Item{ID: g.newID(), Kind: "gcitem", Effect: "chip", Name: "burner ICE-breaker", Glyph: '▪', F: CRed, Power: 3, Value: 45, Weight: 0, Ident: true,
		Lore: "A one-shot cartridge. Use it before a hack for +3 Intrusion rating on the next run."}
}

func gcCyUse(g *Game, it *Item) bool {
	switch it.Effect {
	case "deck":
		g.Msg("The deck needs no fussing; it is used automatically when you jack into a machine.")
		return true
	case "chip":
		g.gcHt().Chip = 3
		g.removeInv(it)
		g.Msg("{{C|You slot the burner. Your next hack runs hot.}}")
		g.Mode = "main"
		g.spend(1000)
		return true
	}
	return false
}

func gcCyTalk(g *Game, a *Actor) []Option {
	if gcCyGenre(g) != 2 || a == nil {
		return nil
	}
	if (a.Role == "merchant" || a.Role == "exotic" || a.Role == "smith") && !a.Has("gcdeck") {
		a.Set("gcdeck")
		r := NewRng(fmt.Sprintf("%s:gc:stock:%d", g.Seed, a.ID))
		if r.Chance(0.55) {
			a.Inv = append(a.Inv, g.gcMakeDeck(1+r.Int(0, 2)))
		}
		if r.Chance(0.5) {
			a.Inv = append(a.Inv, g.gcMakeChip())
		}
	}
	return nil
}
