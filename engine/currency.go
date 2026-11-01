package engine

import (
	"fmt"
	"math"
	"sort"
	"strings"
)

// Multi-currency world (prefix cu). Every realm (StateData) mints its own money, picked
// deterministically from genre, culture sphere, realm form/ideology and planet; each has a hidden
// value against a base unit (1 base = 1 old "gold") and a volatility that drifts by the day.
// The player's wallet is a map of holdings in cuState; g.P.Gold stays valid for every other
// system as the total in base units. Code that still does g.P.Gold += / -= is reconciled by
// cuSync (local coin for gains, local-first for losses), so nothing breaks while call sites are
// moved to cuEarn / cuSpend one by one.

type cuDef struct {
	ID    string
	Kind  string // coin scrip caps creds biomass dram salt shell bead bean ivory ichor chit barter
	Sing  string
	Plur  string
	Pre   string // realm prefix ("Qotusu "), empty for universal things
	Rate  float64
	Vol   float64
	Realm int
}

// Label is the display name for n units.
func (d *cuDef) Label(n int) string {
	if n == 1 {
		return d.Pre + d.Sing
	}
	return d.Pre + d.Plur
}

type cuState struct {
	Held   map[string]int `json:"held,omitempty"`
	Inited bool           `json:"init,omitempty"`
	Synced int            `json:"syn,omitempty"`
	Last   int            `json:"last,omitempty"` // realm id of the last place we announced money for
	defs   map[int]*cuDef
}

func (g *Game) cu() *cuState {
	st := g.ext("cu", func() any { return &cuState{} }).(*cuState)
	if st.Held == nil {
		st.Held = map[string]int{}
	}
	return st
}

// ---- naming tables ----

var cuCoinSphere = map[string][2]string{
	"sinic": {"cash", "cash"}, "japonic": {"mon", "mon"}, "korean": {"yeop", "yeop"},
	"norse": {"hacksilver", "hacksilver"}, "nordic": {"hacksilver", "hacksilver"}, "germanic": {"pfennig", "pfennigs"},
	"latin": {"denarius", "denarii"}, "greek": {"obol", "obols"}, "hellenic": {"obol", "obols"},
	"indic": {"rupaka", "rupakas"}, "arabic": {"dirham", "dirhams"}, "persian": {"daric", "darics"},
	"steppe": {"tamga", "tamgas"}, "celtic": {"torc-ring", "torc-rings"}, "slavic": {"grivna", "grivnas"},
	"maya": {"jade bead", "jade beads"}, "nahua": {"quachtli", "quachtli"}, "andean": {"silver tupu", "silver tupus"},
	"ethiopic": {"birr", "birr"}, "yoruba": {"cowrie", "cowries"}, "mande": {"cowrie", "cowries"},
	"polynesian": {"pearl", "pearls"}, "sea": {"pearl", "pearls"}, "turtle": {"jade tile", "jade tiles"},
	"inuit": {"ivory tally", "ivory tallies"}, "eldritch": {"ichor-mark", "ichor-marks"},
}

var cuBarterSphere = map[string]struct{ kind, s, p string }{
	"ethiopic": {"salt", "salt bar", "salt bars"}, "arabic": {"salt", "salt bar", "salt bars"},
	"yoruba": {"shell", "cowrie", "cowries"}, "mande": {"shell", "cowrie", "cowries"},
	"polynesian": {"shell", "pearl-shell", "pearl-shells"}, "sea": {"shell", "pearl-shell", "pearl-shells"}, "turtle": {"shell", "pearl-shell", "pearl-shells"},
	"maya": {"bean", "cacao bean", "cacao beans"}, "nahua": {"bean", "cacao bean", "cacao beans"},
	"andean": {"bead", "spondylus bead", "spondylus beads"}, "inuit": {"ivory", "ivory tally", "ivory tallies"},
}

var cuScrip = map[string][2]string{
	"fantasy": {"banknote", "banknotes"}, "mythic": {"votive token", "votive tokens"}, "steampunk": {"banknote", "banknotes"},
	"dieselpunk": {"script note", "script notes"}, "atompunk": {"pre-war bill", "pre-war bills"}, "cyberpunk": {"corp-scrip", "corp-scrip"},
	"biopunk": {"gene-note", "gene-notes"}, "solarpunk": {"hour-credit", "hour-credits"},
}

var cuChit = map[string][2]string{
	"fantasy": {"tally", "tallies"}, "mythic": {"tally", "tallies"}, "steampunk": {"ration ticket", "ration tickets"},
	"dieselpunk": {"ration stamp", "ration stamps"}, "atompunk": {"ration chit", "ration chits"}, "cyberpunk": {"ration credit", "ration credits"},
	"biopunk": {"spore-chit", "spore-chits"}, "solarpunk": {"share-token", "share-tokens"},
}

var cuChanger = map[string]string{
	"fantasy": "money-changer", "mythic": "temple banker", "steampunk": "exchange clerk", "dieselpunk": "exchange teller",
	"atompunk": "cap broker", "cyberpunk": "credit broker", "biopunk": "token broker", "solarpunk": "ledger keeper",
}

func cuHas(s string, subs ...string) bool {
	s = strings.ToLower(s)
	for _, x := range subs {
		if strings.Contains(s, x) {
			return true
		}
	}
	return false
}

func cuRange(r *Rng, lo, hi float64) float64 { return lo + (hi-lo)*r.Float() }

// ---- currency of a realm ----

func (g *Game) cuWild() *cuDef {
	name := g.Sk.Currency
	if name == "" {
		name = "scrap"
	}
	return &cuDef{ID: "wild", Kind: "barter", Sing: name, Plur: name, Rate: 1, Vol: 0}
}

var cuDram = &cuDef{ID: "dram", Kind: "dram", Sing: "dram", Plur: "drams", Rate: 1}

// cuDef returns the currency of a realm (0 = no realm: the neutral barter coin).
func (g *Game) cuDef(realm int) *cuDef {
	if realm <= 0 || g.W.stateByID[realm] == nil {
		return g.cuWild()
	}
	st := g.cu()
	if st.defs == nil {
		st.defs = map[int]*cuDef{}
	}
	if d, ok := st.defs[realm]; ok {
		return d
	}
	d := g.cuMake(g.W.stateByID[realm])
	st.defs[realm] = d
	return d
}

func (g *Game) cuCapitalBiome(s *StateData) int {
	w := g.W
	if b := w.burgByID[s.Capital]; b != nil && b.P >= 0 && b.P < len(w.Biome) {
		return w.Biome[b.P]
	}
	for i := range w.Burgs {
		if b := &w.Burgs[i]; b.State == s.ID && b.P >= 0 && b.P < len(w.Biome) {
			return w.Biome[b.P]
		}
	}
	return -1
}

func (g *Game) cuMake(s *StateData) *cuDef {
	r := NewRng(fmt.Sprintf("%s:cu:%d", g.Seed, s.ID))
	w := g.W
	genre := g.Genre
	sphere := ""
	if cu := w.cultByID[s.Culture]; cu != nil {
		sphere = strings.ToLower(cu.Sphere)
	}
	d := &cuDef{ID: fmt.Sprintf("s%d", s.ID), Realm: s.ID, Pre: s.Name + " "}
	pk := w.PlanetKind + " " + w.Planet
	bio := g.cuCapitalBiome(s)
	dryPlanet := cuHas(pk, "desert", "arid", "dry", "waste", "barren", "dune", "scorch")
	dryLand := bio == 1 || bio == 2 || bio == 13
	comm := s.Ideology == "communal" || cuHas(s.Form, "commune", "collective")
	tribal := cuHas(s.Form, "chiefdom", "tribe", "clan", "band")
	commerce := s.Ideology == "oligarchy" || s.Ideology == "republic" || cuHas(s.Form, "commercial", "union", "league", "company", "corporat")
	auto := s.Ideology == "autocracy" || cuHas(s.Form, "empire", "junta", "regime")

	switch {
	case (dryPlanet && r.Chance(0.85)) || (dryLand && r.Chance(0.65)):
		d.Kind, d.Sing, d.Plur, d.Pre = "dram", "dram", "drams", ""
		d.Rate, d.Vol = cuRange(r, 0.95, 1.1), 0.05
	case cuHas(pk, "twilight", "tidal", "eclipse", "dusk") && r.Chance(0.6):
		d.Kind, d.Sing, d.Plur = "ichor", "ichor-mark", "ichor-marks"
		d.Rate, d.Vol = cuRange(r, 1.3, 2.0), 0.3
	case tribal:
		if b, ok := cuBarterSphere[sphere]; ok {
			d.Kind, d.Sing, d.Plur = b.kind, b.s, b.p
		} else {
			d.Kind, d.Sing, d.Plur = "bead", "glass bead", "glass beads"
		}
		d.Rate, d.Vol = cuBarterRate(d.Kind, r), 0.1
	case comm:
		c := cuChit[genre]
		if c[0] == "" {
			c = cuChit["fantasy"]
		}
		d.Kind, d.Sing, d.Plur = "chit", c[0], c[1]
		d.Rate, d.Vol = cuRange(r, 0.7, 1.0), 0.04
	case commerce && r.Chance(0.55) && genre != "atompunk":
		sc := cuScrip[genre]
		if sc[0] == "" {
			sc = cuScrip["fantasy"]
		}
		d.Kind, d.Sing, d.Plur = "scrip", sc[0], sc[1]
		d.Rate, d.Vol = cuRange(r, 0.8, 1.3), 0.14
	case genre == "atompunk" && commerce && r.Chance(0.4):
		sc := cuScrip[genre]
		d.Kind, d.Sing, d.Plur = "scrip", sc[0], sc[1]
		d.Rate, d.Vol = cuRange(r, 0.8, 1.2), 0.2
	default:
		switch genre {
		case "atompunk":
			d.Kind, d.Sing, d.Plur, d.Pre = "caps", "cap", "caps", s.Name+" "
			d.Rate, d.Vol = cuRange(r, 0.25, 0.6), 0.2
		case "cyberpunk":
			d.Kind, d.Sing, d.Plur = "creds", "cred", "creds"
			d.Rate, d.Vol = cuRange(r, 0.5, 1.5), 0.28
		case "biopunk":
			d.Kind, d.Sing, d.Plur = "biomass", "biomass token", "biomass tokens"
			d.Rate, d.Vol = cuRange(r, 0.4, 1.2), 0.18
		case "solarpunk":
			d.Kind, d.Sing, d.Plur = "scrip", "credit", "credits"
			d.Rate, d.Vol = cuRange(r, 0.8, 1.2), 0.1
		case "steampunk":
			d.Kind, d.Sing, d.Plur = "coin", "sovereign", "sovereigns"
			d.Rate, d.Vol = cuRange(r, 0.8, 1.4), 0.08
		case "dieselpunk":
			d.Kind, d.Sing, d.Plur = "coin", "mark", "marks"
			d.Rate, d.Vol = cuRange(r, 0.7, 1.3), 0.1
		default:
			if c, ok := cuCoinSphere[sphere]; ok {
				d.Kind, d.Sing, d.Plur = "coin", c[0], c[1]
			} else {
				d.Kind, d.Sing, d.Plur = "coin", "crown", "crowns"
			}
			d.Rate, d.Vol = cuRange(r, 0.7, 1.5), 0.07
			if s.Ideology == "theocracy" {
				d.Rate *= 1.1
			}
		}
		if auto { // a debased, nervous currency
			d.Rate *= 0.8
			d.Vol += 0.14
		}
	}
	return d
}

func cuBarterRate(kind string, r *Rng) float64 {
	switch kind {
	case "salt":
		return cuRange(r, 1.8, 2.5)
	case "shell":
		return cuRange(r, 0.25, 0.45)
	case "bean":
		return cuRange(r, 0.2, 0.35)
	case "ivory":
		return cuRange(r, 1.2, 1.8)
	}
	return cuRange(r, 0.3, 0.5)
}

// cuDefByID resolves a holdings key.
func (g *Game) cuDefByID(id string) *cuDef {
	switch id {
	case "wild":
		return g.cuWild()
	case "dram":
		return cuDram
	}
	var n int
	if _, err := fmt.Sscanf(id, "s%d", &n); err == nil {
		return g.cuDef(n)
	}
	return g.cuWild()
}

// cuRate: base value of one unit today (drifts with volatility, war and a fallen realm).
func (g *Game) cuRate(d *cuDef) float64 {
	rate := d.Rate
	if d.Vol > 0 {
		n := NewRng(fmt.Sprintf("%s:cuv:%s:%d", g.Seed, d.ID, g.Day()/3)).Float()*2 - 1
		rate *= 1 + d.Vol*n
	}
	if s := g.W.stateByID[d.Realm]; d.Realm > 0 && s != nil {
		if !s.Alive {
			rate *= 0.4
		} else if len(s.Wars) > 0 {
			rate *= 0.92
		}
	}
	return math.Max(rate, 0.05)
}

// ---- where am I ----

func (g *Game) cuRealm() int {
	zn := g.Cur
	if zn == nil || g.W == nil {
		return 0
	}
	if zn.Burg > 0 {
		if bd := g.W.burgByID[zn.Burg]; bd != nil {
			return bd.State
		}
	}
	if g.W.In(zn.WX, zn.WY) {
		return g.W.State[g.W.Idx(zn.WX, zn.WY)]
	}
	return 0
}

func (g *Game) cuLocal() *cuDef { return g.cuDef(g.cuRealm()) }

// cuLocalName: plural name of the local money, for any "%d %s" text.
func (g *Game) cuLocalName() string { return g.cuLocal().Plur }

func (g *Game) cuChangerHere() bool {
	if g.Cur == nil {
		return false
	}
	for _, a := range g.Cur.Actors {
		if a.Role == "changer" && a.HP > 0 {
			return true
		}
	}
	return false
}

func cuClamp(v, lo, hi float64) float64 { return math.Min(hi, math.Max(lo, v)) }

// cuSpread: the changer's cut; Persuasion and Ego shave it.
func (g *Game) cuSpread() float64 {
	return cuClamp(0.16-0.025*float64(g.Skills["persuasion"])-0.006*float64(g.P.M(EGO)), 0.03, 0.2)
}

// cuForeign says whether foreign money is taken here and at what penalty (fraction lost).
func (g *Game) cuForeign() (bool, float64) {
	zn := g.Cur
	best, ok := 1.0, false
	take := func(p float64) {
		if p < best {
			best, ok = p, true
		}
	}
	realm := g.cuRealm()
	if realm == 0 {
		take(0.3)
	}
	if g.cuChangerHere() {
		take(g.cuSpread() + 0.03)
	}
	if zn != nil && zn.Burg > 0 {
		if bd := g.W.burgByID[zn.Burg]; bd != nil {
			if bd.Bandit {
				take(0.3)
			}
			if bd.Port {
				take(0.18)
			}
			if g.cuBorder(zn.WX, zn.WY, bd.State) {
				take(0.2)
			}
		}
	}
	if zn != nil && (strings.Contains(zn.Name, "Warrens") || (g.Sk.Bandits != "" && strings.Contains(zn.Name, g.Sk.Bandits+" camp"))) {
		take(0.25)
	}
	return ok, best
}

func (g *Game) cuBorder(wx, wy, st int) bool {
	if st <= 0 || !g.W.In(wx, wy) {
		return false
	}
	for dy := -2; dy <= 2; dy++ {
		for dx := -2; dx <= 2; dx++ {
			if x, y := wx+dx, wy+dy; g.W.In(x, y) {
				if o := g.W.State[g.W.Idx(x, y)]; o > 0 && o != st {
					return true
				}
			}
		}
	}
	return false
}

// ---- wallet ----

func (g *Game) cuUnits(id string) int {
	if id == "dram" {
		return g.lqDrams()
	}
	return g.cu().Held[id]
}

// cuIDs: every key the player holds something under, drams included.
func (g *Game) cuIDs() []string {
	st := g.cu()
	var ids []string
	for id, n := range st.Held {
		if n > 0 {
			ids = append(ids, id)
		}
	}
	if g.lqDrams() > 0 {
		ids = append(ids, "dram")
	}
	sort.Slice(ids, func(i, j int) bool {
		vi := float64(g.cuUnits(ids[i])) * g.cuRate(g.cuDefByID(ids[i]))
		vj := float64(g.cuUnits(ids[j])) * g.cuRate(g.cuDefByID(ids[j]))
		if vi != vj {
			return vi > vj
		}
		return ids[i] < ids[j]
	})
	return ids
}

// cuTotalF: the wallet in base units.
func (g *Game) cuTotalF() float64 {
	t := 0.0
	for _, id := range g.cuIDs() {
		t += float64(g.cuUnits(id)) * g.cuRate(g.cuDefByID(id))
	}
	return t
}

func (g *Game) cuTotal() int { return int(math.Round(g.cuTotalF())) }

func (g *Game) cuFinish() {
	g.P.Gold = g.cuTotal()
	g.cu().Synced = g.P.Gold
}

// cuAddLocal pays base units into the wallet in the local money.
func (g *Game) cuAddLocal(base int) string {
	if base <= 0 {
		return ""
	}
	d := g.cuLocal()
	n := maxi(1, int(math.Round(float64(base)/g.cuRate(d))))
	st := g.cu()
	if d.Kind == "dram" {
		got := g.lqGain(n)
		if left := n - got; left > 0 {
			st.Held[d.ID] += left // receipts for water you have no flask for
		}
	} else {
		st.Held[d.ID] += n
	}
	return fmt.Sprintf("%d %s", n, d.Label(n))
}

type cuTake struct {
	id    string
	units int
}

// cuPlan picks what to hand over for base units: local money, then water (universal), then
// foreign money at the given penalty when allowed. ok=false means it does not cover; the plan
// then holds everything that could be used.
func (g *Game) cuPlan(base int, pen float64, foreign bool) ([]cuTake, bool) {
	loc := g.cuLocal()
	ids := g.cuIDs()
	var order []string
	pn := map[string]float64{}
	has := map[string]bool{}
	for _, id := range ids {
		has[id] = true
	}
	if has[loc.ID] {
		order = append(order, loc.ID)
	}
	if has["dram"] && loc.ID != "dram" {
		order = append(order, "dram")
	}
	for _, id := range ids {
		if id == loc.ID || id == "dram" {
			continue
		}
		if foreign {
			order = append(order, id)
			pn[id] = pen
		}
	}
	rem := float64(base)
	var out []cuTake
	for _, id := range order {
		if rem <= 1e-6 {
			break
		}
		eff := g.cuRate(g.cuDefByID(id)) * (1 - pn[id])
		use := mini(g.cuUnits(id), int(math.Ceil(rem/eff-1e-9)))
		if use <= 0 {
			continue
		}
		out = append(out, cuTake{id, use})
		rem -= float64(use) * eff
	}
	return out, rem <= 1e-6
}

func (g *Game) cuApply(plan []cuTake) {
	st := g.cu()
	for _, t := range plan {
		if t.id == "dram" {
			g.lqPay(t.units)
			continue
		}
		st.Held[t.id] -= t.units
		if st.Held[t.id] <= 0 {
			delete(st.Held, t.id)
		}
	}
}

// cuSync reconciles g.P.Gold with the wallet: edits other systems made to Gold since the last
// sync become local-currency gains or local-first losses, then Gold is set to the wallet total.
func (g *Game) cuSync() {
	if g.P == nil || g.W == nil || g.Cur == nil {
		return
	}
	st := g.cu()
	if !st.Inited {
		st.Inited = true
		g.cuAddLocal(g.P.Gold)
		g.cuFinish()
		return
	}
	if d := g.P.Gold - st.Synced; d > 0 {
		g.cuAddLocal(d)
	} else if d < 0 {
		plan, _ := g.cuPlan(-d, 0, true)
		g.cuApply(plan)
	}
	g.cuFinish()
}

// cuEarn pays base units in the money of the place you stand in; it returns "12 Qotusu cash".
func (g *Game) cuEarn(base int) string {
	g.cuSync()
	s := g.cuAddLocal(base)
	g.cuFinish()
	return s
}

// cuSpend pays base units: local money first, then water, then foreign money at a penalty but
// only where it is taken (money changer in the zone, port, border town, black market, neutral
// land, outlaws). False and nothing taken when you cannot pay; it says why when foreign money
// would have covered it.
func (g *Game) cuSpend(base int) bool {
	if base <= 0 {
		return true
	}
	g.cuSync()
	ok, pen := g.cuForeign()
	plan, good := g.cuPlan(base, pen, ok)
	if !good {
		if !ok {
			if _, all := g.cuPlan(base, 0, true); all {
				g.Msg("{{y|They will not take foreign money here. Find a money changer, a port or a border town.}}")
			}
		}
		return false
	}
	g.cuApply(plan)
	g.cuFinish()
	return true
}

// cuCanSpend: like cuSpend without paying.
func (g *Game) cuCanSpend(base int) bool {
	g.cuSync()
	ok, pen := g.cuForeign()
	_, good := g.cuPlan(base, pen, ok)
	return good
}

// cuFormat prints a base amount in the local money: "42 Qotusu cash".
func (g *Game) cuFormat(base int) string {
	d := g.cuLocal()
	n := maxi(0, int(math.Ceil(float64(base)/g.cuRate(d)-1e-9)))
	if base > 0 && n == 0 {
		n = 1
	}
	return fmt.Sprintf("%d %s", n, d.Label(n))
}

// cuHolding: "312 Qotusu cash".
func (g *Game) cuHolding(id string) string {
	n := g.cuUnits(id)
	return fmt.Sprintf("%d %s", n, g.cuDefByID(id).Label(n))
}

// cuWallet lists the holdings, richest first: "312 Hearth-cash · 40 caps · 6 drams".
func (g *Game) cuWallet() string {
	g.cuSync()
	var parts []string
	for _, id := range g.cuIDs() {
		parts = append(parts, g.cuHolding(id))
	}
	if len(parts) == 0 {
		return "empty purse"
	}
	return strings.Join(parts, " · ")
}

// ---- sidebar ----

func cuSide(g *Game) []string {
	g.cuSync()
	ids := g.cuIDs()
	if len(ids) == 0 {
		return nil
	}
	const width, maxLines = 30, 3
	var lines []string
	cur, plain := "", 0
	shown := 0
	for i, id := range ids {
		n := g.cuUnits(id)
		name := g.cuDefByID(id).Label(n)
		if r := []rune(name); len(r) > 16 {
			name = string(r[:15]) + "."
		}
		txt := fmt.Sprintf("%d %s", n, name)
		seg := fmt.Sprintf("{{W|%d}} %s", n, name)
		w := len([]rune(txt))
		if plain > 0 && plain+3+w > width {
			lines = append(lines, cur)
			cur, plain = "", 0
			if len(lines) == maxLines {
				break
			}
		}
		if plain > 0 {
			cur += " {{K|·}} "
			plain += 3
		}
		cur += seg
		plain += w
		shown = i + 1
	}
	if cur != "" && len(lines) < maxLines {
		lines = append(lines, cur)
	}
	if more := len(ids) - shown; more > 0 && len(lines) > 0 {
		lines[len(lines)-1] += fmt.Sprintf(" {{K|+%d}}", more)
	}
	return lines
}

// ---- money changers ----

func cuZone(g *Game, zn *Zone, _ *Rng) {
	if zn.Z != 0 || zn.Kind != "town" || zn.Burg <= 0 {
		return
	}
	bd := g.W.burgByID[zn.Burg]
	if bd == nil {
		return
	}
	cp := g.cityPlanFor(bd)
	if cp.tier == 0 {
		return
	}
	k := cp.districts[[2]int{zn.ZX, zn.ZY}]
	if k != "market" && k != "harbor" {
		return
	}
	for _, a := range zn.Actors {
		if a.Role == "changer" {
			return
		}
	}
	r := NewRng(g.Seed + ":cuchg:" + zn.Key)
	x, y, ok := zn.FreeSpot(r, func(x, y int) bool { return zn.At(x, y) == TCobble || zn.At(x, y) == TFloor })
	if !ok {
		return
	}
	a := g.makeNPC("changer", bd, g.W.cultByID[bd.Culture], r)
	t := cuChanger[g.Genre]
	if t == "" {
		t = "money changer"
	}
	a.Title, a.F, a.AI = t, CYellow, "still"
	a.Desc = fmt.Sprintf("A %s of %s, weighing foreign money on a small scale.", t, bd.Name)
	a.X, a.Y, a.HomeX, a.HomeY = x, y, x, y
	a.Gold = r.Int(300, 700)
	a.Inv = nil
	zn.Actors = append(zn.Actors, a)
}

// cuConvert swaps units of one holding into another at the changer's spread.
func (g *Game) cuConvert(from, to string, units int) (int, bool) {
	if units <= 0 || g.cuUnits(from) < units || from == "dram" || to == "dram" || from == to {
		return 0, false
	}
	base := float64(units) * g.cuRate(g.cuDefByID(from)) * (1 - g.cuSpread())
	out := int(math.Floor(base / g.cuRate(g.cuDefByID(to))))
	if out <= 0 {
		return 0, false
	}
	st := g.cu()
	st.Held[from] -= units
	if st.Held[from] <= 0 {
		delete(st.Held, from)
	}
	st.Held[to] += out
	g.cuFinish()
	return out, true
}

// cuNeighbours: currencies of the realms around here, as holdings keys.
func (g *Game) cuNeighbours() []string {
	zn := g.Cur
	seen := map[int]bool{g.cuRealm(): true, 0: true}
	var out []string
	for r := 1; r <= 4 && len(out) < 3; r++ {
		for dy := -r; dy <= r; dy++ {
			for dx := -r; dx <= r; dx++ {
				if x, y := zn.WX+dx, zn.WY+dy; g.W.In(x, y) {
					if s := g.W.State[g.W.Idx(x, y)]; !seen[s] && g.W.stateByID[s] != nil && g.W.stateByID[s].Alive && len(out) < 3 {
						seen[s] = true
						out = append(out, g.cuDef(s).ID)
					}
				}
			}
		}
	}
	return out
}

func cuTalk(g *Game, a *Actor) []Option {
	if a.Role != "changer" {
		return nil
	}
	g.Talk = a
	g.cuSync()
	loc := g.cuLocal()
	again := func() { g.cuSync(); g.talkMenu(g.greeting(a)) }
	var out []Option
	pct := int(math.Round(g.cuSpread() * 100))
	for _, id := range g.cuIDs() {
		id := id
		if id == loc.ID || id == "dram" {
			continue
		}
		n := g.cuUnits(id)
		fd := g.cuDefByID(id)
		get := int(math.Floor(float64(n) * g.cuRate(fd) * (1 - g.cuSpread()) / g.cuRate(loc)))
		if get <= 0 {
			continue
		}
		out = append(out, Option{fmt.Sprintf("{{B|Change my %d %s into %s.}} {{K|(get %d, %d%% cut)}}", n, fd.Label(n), loc.Plur, get, pct), func() {
			if got, ok := g.cuConvert(id, loc.ID, n); ok {
				g.Msg(fmt.Sprintf("You change %d %s for %d %s.", n, fd.Label(n), got, loc.Label(got)))
			}
			again()
		}})
	}
	if n := g.cuUnits(loc.ID); n > 1 {
		for _, id := range g.cuNeighbours() {
			id := id
			td := g.cuDefByID(id)
			half := n / 2
			get := int(math.Floor(float64(half) * g.cuRate(loc) * (1 - g.cuSpread()) / g.cuRate(td)))
			if get <= 0 {
				continue
			}
			out = append(out, Option{fmt.Sprintf("{{B|Change half my %s into %s.}} {{K|(get %d, %d%% cut)}}", loc.Plur, td.Plur, get, pct), func() {
				if got, ok := g.cuConvert(loc.ID, id, half); ok {
					g.Msg(fmt.Sprintf("You change %d %s for %d %s.", half, loc.Label(half), got, td.Label(got)))
				}
				again()
			}})
		}
	}
	out = append(out, Option{"What are the rates today?", func() { g.talkMenu(g.cuRatesText(a)) }})
	return out
}

func (g *Game) cuRatesText(a *Actor) string {
	loc := g.cuLocal()
	lr := g.cuRate(loc)
	var ids []string
	for _, id := range g.cuNeighbours() {
		ids = append(ids, id)
	}
	for _, id := range g.cuIDs() {
		dup := false
		for _, o := range ids {
			dup = dup || o == id
		}
		if !dup && id != loc.ID {
			ids = append(ids, id)
		}
	}
	var sb strings.Builder
	sb.WriteString("\"Today, against " + loc.Plur + ":")
	for i, id := range ids {
		if i >= 5 {
			break
		}
		d := g.cuDefByID(id)
		sb.WriteString(fmt.Sprintf(" one %s is %.2f;", d.Label(1), g.cuRate(d)/lr))
	}
	sb.WriteString(fmt.Sprintf(" my cut is %d%%.\"", int(math.Round(g.cuSpread()*100))))
	return sb.String()
}

// ---- announcements ----

func cuEnter(g *Game, zn *Zone) {
	g.cuSync()
	st := g.cu()
	realm := g.cuRealm()
	if realm == st.Last {
		return
	}
	prev := st.Last
	st.Last = realm
	if prev == 0 && realm == 0 {
		return
	}
	d := g.cuLocal()
	if realm == 0 {
		g.Msg("{{K|No realm's money rules here; every purse is taken at a poor rate.}}")
		return
	}
	g.Msg(fmt.Sprintf("{{y|Here they trade in %s.}}", d.Plur))
}

func init() {
	zoneHooks = append(zoneHooks, cuZone)
	talkHooks = append(talkHooks, cuTalk)
	sideHooks = append(sideHooks, cuSide)
	turnHooks = append(turnHooks, func(g *Game) { g.cuSync() })
	enterHooks = append(enterHooks, cuEnter)
}
