package engine

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"sync"
)

// Transport (prefix tr): Realmforge's rail lines and air routes, plus mounts and carts.
// Rail: stations in towns on a line (platform, timetable, ticket clerk); a ticket buys hours of
// travel instead of days, with a light event on the way. Air: dockmasters fly airships between
// docks and up to the sky isles. Mounts and carts: bought from the ostler in any surface town,
// chosen by biome, culture sphere and genre; faster overland pace (travel.go tvPace), feed, stabling,
// lameness and theft; carts and saddlebags add carrying capacity (inventory.go capacity).
// Old worlds have no Transport: mounts and carts still work, there are no stations or docks.
// State g.ext("tr"); all chance draws are side rngs (seed:tr:...), never g.R.

// TransportData mirrors Realmforge src/lib/adventure/transport.ts.
type TransportData struct {
	Rail  []TrRailData `json:"rail,omitempty"`
	Air   []TrAirData  `json:"air,omitempty"`
	Docks []int        `json:"docks,omitempty"`
}

type TrRailData struct {
	ID        int    `json:"id"`
	Name      string `json:"name,omitempty"`
	Kind      string `json:"kind"`
	Stations  []int  `json:"stations"`  // burg ids in line order
	Parasangs []int  `json:"parasangs"` // the track, parasang by parasang
	Owner     int    `json:"owner,omitempty"`
	Built     int    `json:"built,omitempty"`
	Cut       int    `json:"cut,omitempty"` // year the war cut the line; no trains run
}

type TrAirData struct {
	A    int    `json:"a"`
	B    int    `json:"b"`
	Kind string `json:"kind"`
}

// ---- derived lookups ----

type trIndex struct {
	railAt  map[int][]int // parasang → line indices
	station map[int][]int // burg → line indices
	dock    map[int]bool
	air     map[int][]int // burg → air route indices
}

var (
	trMu   sync.Mutex
	trIdxs = map[*WorldData]*trIndex{}
)

func trIdx(w *WorldData) *trIndex {
	trMu.Lock()
	defer trMu.Unlock()
	if ix := trIdxs[w]; ix != nil {
		return ix
	}
	ix := &trIndex{railAt: map[int][]int{}, station: map[int][]int{}, dock: map[int]bool{}, air: map[int][]int{}}
	if t := w.Transport; t != nil {
		for i, l := range t.Rail {
			for _, p := range l.Parasangs {
				if xs := ix.railAt[p]; len(xs) == 0 || xs[len(xs)-1] != i {
					ix.railAt[p] = append(xs, i)
				}
			}
			for _, b := range l.Stations {
				ix.station[b] = append(ix.station[b], i)
			}
		}
		for _, d := range t.Docks {
			ix.dock[d] = true
		}
		for i, r := range t.Air {
			ix.air[r.A] = append(ix.air[r.A], i)
			ix.air[r.B] = append(ix.air[r.B], i)
			ix.dock[r.A], ix.dock[r.B] = true, true
		}
	}
	trIdxs[w] = ix
	return ix
}

// trReindex drops the cached lookups (tests that inject transport data).
func trReindex(w *WorldData) {
	trMu.Lock()
	delete(trIdxs, w)
	trMu.Unlock()
}

// ---- vehicles ----

// trSpeed is km per hour by rail or air kind.
var trSpeed = map[string]float64{
	"railway": 60, "diesel line": 80, "atomic monorail": 110, "maglev": 280, "bio-tram": 35, "solar rail": 140, "salvage rail": 30,
	"airship": 80, "zeppelin": 110, "rocket ferry": 600, "aerodyne": 300, "sky-ray": 70, "solar glider": 90, "gyrocopter": 120,
	"skyliner": 400, "skyship": 60, "griffon roost": 70, "sky barge": 40,
}

// trVehicle names what you ride for a kind.
var trVehicle = map[string]string{
	"railway": "steam train", "diesel line": "diesel train", "atomic monorail": "monorail car", "maglev": "maglev", "bio-tram": "bio-tram",
	"solar rail": "sunline train", "salvage rail": "rust-cart train", "griffon roost": "griffon", "sky barge": "sky barge", "rocket ferry": "rocket ferry",
}

func trRideName(kind string) string {
	if v := trVehicle[kind]; v != "" {
		return v
	}
	return kind
}

func trKmh(kind string) float64 {
	if v := trSpeed[kind]; v > 0 {
		return v
	}
	return 50
}

// ---- mounts and carts ----

type trMountDef struct {
	ID, Name string
	Machine  bool
	Pace     float64 // multiplier on open ground and roads
	Rough    float64 // share of the bonus kept in forest, marsh and rough land
	Carry    int
	Price    int
	Feed     string // "" none, fodder, fuel, charge, nutrient
	Biomes   []int
	Spheres  []string
	Genres   []string // nil = any genre
	Looks    []string
}

var trMounts = []trMountDef{
	{ID: "horse", Name: "horse", Pace: 1.7, Rough: 0.5, Carry: 25, Price: 150, Feed: "fodder", Biomes: []int{3, 4, 5, 6, 9, 17, 18}, Looks: []string{"bay", "grey", "dun", "chestnut", "black", "piebald"}},
	{ID: "pony", Name: "pony", Pace: 1.4, Rough: 0.6, Carry: 20, Price: 80, Feed: "fodder", Looks: []string{"shaggy", "stubborn", "dappled", "fat"}},
	{ID: "camel", Name: "camel", Pace: 1.6, Rough: 0.5, Carry: 45, Price: 160, Feed: "fodder", Biomes: []int{1, 2, 19}, Spheres: []string{"arabic", "persian", "mande", "steppe"}, Looks: []string{"one-humped", "two-humped", "sand-coloured", "haughty"}},
	{ID: "yak", Name: "yak", Pace: 1.35, Rough: 0.8, Carry: 50, Price: 120, Feed: "fodder", Biomes: []int{9, 10, 11, 21}, Spheres: []string{"sinic", "steppe"}, Looks: []string{"shaggy", "black", "white-faced"}},
	{ID: "elephant", Name: "elephant", Pace: 1.3, Rough: 0.7, Carry: 90, Price: 420, Feed: "fodder", Biomes: []int{3, 7, 8}, Spheres: []string{"indic", "sea", "yoruba", "bantu"}, Looks: []string{"old", "painted", "tusked", "young"}},
	{ID: "reindeer", Name: "reindeer", Pace: 1.45, Rough: 0.7, Carry: 30, Price: 110, Feed: "fodder", Biomes: []int{9, 10}, Spheres: []string{"inuit", "norse"}, Looks: []string{"antlered", "grey", "belled"}},
	{ID: "llama", Name: "llama", Pace: 1.25, Rough: 0.8, Carry: 35, Price: 90, Feed: "fodder", Spheres: []string{"andean"}, Looks: []string{"woolly", "tasselled", "spitting"}},
	{ID: "lizard", Name: "giant lizard", Pace: 1.55, Rough: 0.8, Carry: 35, Price: 200, Feed: "fodder", Biomes: []int{7, 12, 13, 15, 16}, Genres: []string{"fantasy", "mythic", "twilight", "wasteland", "biopunk"}, Looks: []string{"green-scaled", "frilled", "basking", "striped"}},
	{ID: "velocipede", Name: "velocipede", Machine: true, Pace: 1.6, Rough: 0.2, Carry: 8, Price: 90, Genres: []string{"steampunk", "dieselpunk", "solarpunk"}, Looks: []string{"brass", "high-wheeled", "rattling"}},
	{ID: "strider", Name: "clockwork strider", Machine: true, Pace: 1.9, Rough: 0.7, Carry: 30, Price: 400, Genres: []string{"steampunk"}, Looks: []string{"ticking", "copper", "long-legged"}},
	{ID: "motorbike", Name: "motorbike", Machine: true, Pace: 2.3, Rough: 0.4, Carry: 20, Price: 280, Feed: "fuel", Genres: []string{"dieselpunk", "atompunk", "wasteland", "cyberpunk"}, Looks: []string{"rusted", "sidecar", "chromed", "growling"}},
	{ID: "skiff", Name: "hover-skiff", Machine: true, Pace: 2.7, Rough: 1, Carry: 30, Price: 650, Feed: "charge", Genres: []string{"cyberpunk", "atompunk", "solarpunk"}, Looks: []string{"humming", "matte", "neon-trimmed"}},
	{ID: "biostrider", Name: "bio-strider", Pace: 2.0, Rough: 0.8, Carry: 40, Price: 480, Feed: "nutrient", Genres: []string{"biopunk"}, Looks: []string{"vat-grown", "chitinous", "purring"}},
}

type trCartDef struct {
	ID, Name string
	Carry    int
	Pace     float64
	Draft    bool // needs a mount to pull it at full pace
	Price    int
	Genres   []string
}

var trCarts = []trCartDef{
	{ID: "handcart", Name: "handcart", Carry: 20, Pace: 0.95, Price: 30},
	{ID: "wagon", Name: "wagon", Carry: 60, Pace: 0.85, Draft: true, Price: 110},
	{ID: "trailer", Name: "cargo trailer", Carry: 45, Pace: 0.9, Draft: true, Price: 140, Genres: []string{"dieselpunk", "atompunk", "cyberpunk", "solarpunk", "wasteland"}},
}

func trMountByID(id string) *trMountDef {
	for i := range trMounts {
		if trMounts[i].ID == id {
			return &trMounts[i]
		}
	}
	return nil
}

func trCartByID(id string) *trCartDef {
	for i := range trCarts {
		if trCarts[i].ID == id {
			return &trCarts[i]
		}
	}
	return nil
}

func trFeedWord(f string) string {
	switch f {
	case "fuel":
		return "fuel"
	case "charge":
		return "charge cells"
	case "nutrient":
		return "nutrient paste"
	}
	return "fodder"
}

type trMount struct {
	Def    string  `json:"d"`
	Name   string  `json:"n"`
	Feed   float64 `json:"f"`           // days of feed carried
	Starve float64 `json:"s,omitempty"` // days without
}

type trStable struct {
	Burg   int    `json:"b"`
	Planet int    `json:"p"`
	Since  int    `json:"t"`
	Town   string `json:"n"`
}

type trState struct {
	Mount   *trMount  `json:"m,omitempty"`
	Cart    string    `json:"c,omitempty"`
	Stable  *trStable `json:"st,omitempty"`
	Rides   int       `json:"r,omitempty"`
	Flights int       `json:"f,omitempty"`
}

func (g *Game) tr() *trState { return g.ext("tr", func() any { return &trState{} }).(*trState) }

// trWithYou: the mount travels with you (owned and not left in a stable).
func (g *Game) trWithYou() *trMountDef {
	st := g.tr()
	if st.Mount == nil || st.Stable != nil {
		return nil
	}
	return trMountByID(st.Mount.Def)
}

func trOpenBiome(b int) bool {
	switch b {
	case 1, 2, 3, 4, 10, 17, 18, 19:
		return true
	}
	return false
}

// trPaceMul is the mount and cart multiplier on overland pace across parasang p (tvPace hook).
func (g *Game) trPaceMul(p int) float64 {
	st := g.tr()
	mul := 1.0
	d := g.trWithYou()
	if d != nil && (d.Feed == "" || !d.Machine || st.Mount.Feed > 0) {
		share := 1.0
		road := p >= 0 && p < len(g.W.Road) && g.W.Road[p] > 0
		if !road {
			b := 0
			if p >= 0 && p < len(g.W.Biome) {
				b = g.W.Biome[p]
			}
			if !trOpenBiome(b) {
				share = d.Rough
			} else if d.Machine && d.Rough < 1 {
				share = 0.6 + 0.4*d.Rough
			}
			if p >= 0 && p < len(g.W.Height) && g.W.Height[p] >= 70 {
				share = math.Min(share, d.Rough*0.5)
			}
			if d.ID == "camel" && (b == 1 || b == 2 || b == 19) || d.ID == "yak" && (b == 10 || b == 11 || b == 21) {
				share = 1
			}
		}
		if d.Machine && st.Mount.Feed <= 0 && d.Feed != "" {
			share = 0
		}
		mul = 1 + (d.Pace-1)*share
		mul = g.tmAdjust(p, d, mul) // stamina, river swimmers, mount level (taming.go)
		mul = g.bcMount(mul)        // saddle quality (beastcraft.go)
	}
	if c := trCartByID(st.Cart); c != nil && st.Stable == nil {
		if c.Draft && d == nil {
			mul *= 0.6
		} else {
			mul *= c.Pace
		}
	}
	return mul
}

// trCarry is extra carrying capacity from saddlebags and a cart (inventory.go capacity hook).
func (g *Game) trCarry() int {
	st := g.tr()
	if st.Stable != nil {
		return 0
	}
	n := 0
	if d := g.trWithYou(); d != nil {
		n += d.Carry
	}
	if c := trCartByID(st.Cart); c != nil {
		n += c.Carry
	}
	return n
}

// trRiding is the travel message's ", riding your bay horse" suffix.
func (g *Game) trRiding() string {
	if d := g.trWithYou(); d != nil {
		return " riding your " + g.tr().Mount.Name
	}
	return ""
}

// trHow is "on foot" / "riding" for the world map's time estimates.
func (g *Game) trHow() string {
	if d := g.trWithYou(); d != nil {
		return "riding"
	}
	return "on foot"
}

// trJourney: after an overland crossing of cost minutes the mount eats, may go lame, may be stolen.
func (g *Game) trJourney(cost int) {
	st := g.tr()
	d := g.trWithYou()
	if d == nil {
		return
	}
	m := st.Mount
	g.tmJourney(cost) // flying stamina, mount XP and loyalty (taming.go)
	days := float64(cost) / DayLen
	p := g.W.Idx(g.WX, g.WY)
	b := g.W.Biome[p]
	graze := !d.Machine && d.Feed == "fodder" && (b == 3 || b == 4 || b == 17 || b == 18 || (b == 10 && (d.ID == "yak" || d.ID == "reindeer")))
	if d.Feed != "" && !graze {
		m.Feed -= days
		if m.Feed < 0 {
			if !d.Machine {
				m.Starve += -m.Feed
			}
			m.Feed = 0
		}
	} else if graze {
		m.Starve = 0
	}
	if d.Machine && d.Feed != "" && m.Feed <= 0 {
		g.Msg(fmt.Sprintf("{{y|Your %s is out of %s; you push it along.}}", m.Name, trFeedWord(d.Feed)))
	} else if !d.Machine && m.Feed <= 0 && !graze && d.Feed != "" {
		g.Msg(fmt.Sprintf("{{y|Your %s is hungry. Buy %s from an ostler.}}", m.Name, trFeedWord(d.Feed)))
	}
	if !d.Machine && m.Starve >= 4 {
		g.Msg(fmt.Sprintf("{{R|Your %s collapses from hunger and does not get up.}}", m.Name))
		st.Mount = nil
		return
	}
	r := NewRng(fmt.Sprintf("%s:tr:road:%d", g.Seed, g.Turn))
	if !d.Machine && g.W.Height[p] >= 70 && r.Chance(0.03*days) {
		g.Msg(fmt.Sprintf("{{R|Your %s stumbles on the scree and breaks a leg. You have to leave it.}}", m.Name))
		st.Mount = nil
		return
	}
	theft := 0.01 * days
	if g.W.State[p] == 0 {
		theft += 0.04
	}
	if g.Genre == "wasteland" {
		theft *= 2
	}
	theft -= 0.01 * float64(g.Skills["survival"])
	if r.Chance(math.Min(0.2, theft)) {
		g.Msg(fmt.Sprintf("{{R|In the night thieves cut your %s loose and ride off with it.}}", m.Name))
		st.Mount = nil
		if c := trCartByID(st.Cart); c != nil && c.Draft {
			g.Msg(fmt.Sprintf("{{y|Your %s is useless without something to pull it.}}", c.Name))
		}
	}
}

// ---- rail and air trips ----

type trTrip struct {
	To      *BurgData
	Kind    string
	Name    string
	Km      float64
	Minutes int
	Fare    int
	Air     bool
	Via     []int // parasangs crossed
}

// trVia is the track between parasangs a and b: the closest pair of occurrences, loops cut out.
func trVia(ps []int, a, b int) []int {
	bi, bj := -1, -1
	for i, p := range ps {
		if p != a {
			continue
		}
		for j, q := range ps {
			if q == b && (bi < 0 || abs(i-j) < abs(bi-bj)) {
				bi, bj = i, j
			}
		}
	}
	if bi < 0 {
		return nil
	}
	var seg []int
	if bi <= bj {
		seg = append(seg, ps[bi:bj+1]...)
	} else {
		for k := bi; k >= bj; k-- {
			seg = append(seg, ps[k])
		}
	}
	var out []int
	at := map[int]int{}
	for _, p := range seg {
		if k, ok := at[p]; ok {
			for _, q := range out[k+1:] {
				delete(at, q)
			}
			out = out[:k+1]
			continue
		}
		at[p] = len(out)
		out = append(out, p)
	}
	return out
}

// trRailTrips lists the train rides from burg bid (running lines only).
func (g *Game) trRailTrips(bid int) (trips []trTrip, cut []TrRailData) {
	w := g.W
	from := w.burgByID[bid]
	if from == nil || w.Transport == nil {
		return
	}
	for _, li := range trIdx(w).station[bid] {
		l := w.Transport.Rail[li]
		if l.Cut > 0 {
			cut = append(cut, l)
			continue
		}
		for _, sid := range l.Stations {
			to := w.burgByID[sid]
			if sid == bid || to == nil || !to.Alive {
				continue
			}
			steps := 0.0
			var via []int
			if via = trVia(l.Parasangs, from.P, to.P); len(via) > 1 {
				steps = float64(len(via) - 1)
			}
			if steps <= 0 {
				steps = math.Hypot(float64(to.P%w.W-from.P%w.W), float64(to.P/w.W-from.P/w.W)) * 1.2
			}
			km := math.Max(20, steps*g.tvKm())
			trips = append(trips, trTrip{To: to, Kind: l.Kind, Name: l.Name, Km: km, Minutes: maxi(45, int(km/trKmh(l.Kind)*60)+30), Fare: 3 + int(km/25), Via: via})
		}
	}
	sort.Slice(trips, func(i, j int) bool { return trips[i].Km < trips[j].Km })
	return
}

// trAirTrips lists the flights from dock bid.
func (g *Game) trAirTrips(bid int) (trips []trTrip) {
	w := g.W
	from := w.burgByID[bid]
	if from == nil || w.Transport == nil {
		return
	}
	for _, ri := range trIdx(w).air[bid] {
		r := w.Transport.Air[ri]
		oid := r.B
		if oid == bid {
			oid = r.A
		}
		to := w.burgByID[oid]
		if to == nil || !to.Alive {
			continue
		}
		km := math.Max(30, math.Hypot(float64(to.P%w.W-from.P%w.W), float64(to.P/w.W-from.P/w.W))*g.tvKm())
		trips = append(trips, trTrip{To: to, Kind: r.Kind, Km: km, Minutes: maxi(60, int(km/trKmh(r.Kind)*60)+45), Fare: 6 + int(km/12), Air: true})
	}
	sort.Slice(trips, func(i, j int) bool { return trips[i].Km < trips[j].Km })
	return
}

func (g *Game) trFare(t trTrip) int {
	f := t.Fare * (100 - g.Skills["persuasion"]*8) / 100
	if g.trWithYou() != nil {
		f += maxi(1, f/3) // a stall or a hold for the mount
	}
	return maxi(1, f)
}

var trPassengers = []string{"a travelling clerk", "a soldier on leave", "a pilgrim with a bundle", "a widow with a basket of eggs", "a pedlar", "a student going home", "a retired engineer", "a priest reading", "two quarrelling merchants"}

// trRide pays the fare, passes the hours, rolls one light event and sets you down in the destination town.
func (g *Game) trRide(from *BurgData, t trTrip) bool {
	if g.ijNoFastTravel() { // a vow against fast travel (challenges.go)
		g.Msg("Not in this vow.")
		return false
	}
	fare := g.trFare(t)
	if !g.cuSpend(fare) {
		g.Msg("{{y|\"No ticket, no ride.\" You cannot pay the fare.}}")
		return false
	}
	st := g.tr()
	r := NewRng(fmt.Sprintf("%s:tr:ride:%d:%d", g.Seed, g.Turn, t.To.ID))
	ate, drank, _ := g.tvPass(t.Minutes)
	verb := fmt.Sprintf("You ride the %s", trRideName(t.Kind))
	if t.Name != "" && !t.Air {
		verb = fmt.Sprintf("You ride the %s on the %s", trRideName(t.Kind), t.Name)
	}
	if t.Air {
		verb = fmt.Sprintf("You fly by %s", trRideName(t.Kind))
		st.Flights++
	} else {
		st.Rides++
	}
	msg := fmt.Sprintf("%s from %s to {{Y|%s}}: %.0f km in %s.", verb, from.Name, t.To.Name, t.Km, tvDuration(t.Minutes))
	if ate > 0 || drank > 0 {
		msg += fmt.Sprintf(" {{K|(%d meals, %d drams on the way)}}", ate, drank)
	}
	g.Msg(msg)
	g.trEvent(t, r)
	w := g.W
	wx, wy, z := unTownZone(w, t.To)
	g.WX, g.WY, g.CX, g.CY = wx, wy, wx, wy
	g.Known[t.To.P] = true
	for _, p := range t.Via {
		g.Known[p] = true
	}
	g.Mode = "main"
	g.Talk = nil
	g.enterZone(g.GetZone(wx, wy, 1, 1, z), 40, 13, true)
	g.recompute()
	return true
}

func (g *Game) trEvent(t trTrip, r *Rng) {
	w := g.W
	lawless := false
	for _, p := range t.Via {
		if p >= 0 && p < len(w.State) && w.State[p] == 0 {
			lawless = true
		}
	}
	rob := 0.0
	switch {
	case g.Genre == "wasteland":
		rob = 0.3
	case t.Air && g.Genre == "steampunk":
		rob = 0.06
	case lawless:
		rob = 0.08
	}
	if r.Chance(rob) {
		who := "Raiders"
		if t.Air {
			who = "Sky pirates"
		}
		if g.P.Score(AGI)+g.P.Level+r.Int(1, 20) >= 28 {
			g.Msg(fmt.Sprintf("{{R|%s board between stops!}} You and the guards drive them off.", who))
			g.gainXP(10 + g.P.Level*2)
		} else {
			take := maxi(5, t.Fare*2)
			if g.cuSpend(take) {
				g.Msg(fmt.Sprintf("{{R|%s board between stops and rob the passengers.}} They take %s from you.", who, g.cuFormat(take)))
			} else {
				g.Msg(fmt.Sprintf("{{R|%s board between stops,}} but your purse is too thin to bother with.", who))
			}
		}
		return
	}
	if len(w.Events) > 0 && r.Chance(0.55) {
		// gossip about something that really happened near the destination
		var near []int
		for i, e := range w.Events {
			if e.P >= 0 && abs(e.P%w.W-t.To.P%w.W) <= 6 && abs(e.P/w.W-t.To.P/w.W) <= 6 {
				near = append(near, i)
			}
		}
		e := w.Events[r.Int(0, len(w.Events)-1)]
		if len(near) > 0 {
			e = w.Events[near[r.Int(0, len(near)-1)]]
		}
		text := e.Text
		if len(text) > 180 {
			text = text[:177] + "…"
		}
		g.Msg(fmt.Sprintf("%s beside you talks the whole way: {{K|\"In Y%d — %s\"}}", trCap(Pick(r, trPassengers)), e.Y, text))
		return
	}
	if t.Air {
		g.Msg(Pick(r, []string{"{{C|Clouds part below you, and the land lies spread out like a map.}}", "{{C|The deck hums; gulls race the hull for a while.}}", "{{C|You doze in a wicker seat as the ground slides past.}}"}))
		return
	}
	g.Msg(Pick(r, []string{"{{K|Fields and villages flicker past the window.}}", "{{K|The carriage rocks you half asleep.}}", "{{K|At a halt a girl sells hot tea through the window.}}", "{{K|Smoke and sparks stream past the window.}}"}))
}

// ---- talk ----

func trTalk(g *Game, a *Actor) []Option {
	switch a.Role {
	case "stationmaster":
		return g.trStationTalk(a)
	case "dockmaster":
		return g.trDockTalk(a)
	case "ostler":
		return g.trOstlerTalk(a)
	}
	return nil
}

func (g *Game) trHere() *BurgData {
	if g.Cur == nil {
		return nil
	}
	return g.W.burgByID[g.Cur.Burg]
}

func (g *Game) trTripOption(from *BurgData, t trTrip) Option {
	label := "Ticket"
	if t.Air {
		label = "Passage"
	}
	return Option{fmt.Sprintf("{{B|%s to %s}} {{K|(%s, %.0f km, %s)}} — %s", label, t.To.Name, trRideName(t.Kind), t.Km, tvDuration(t.Minutes), g.cuFormat(g.trFare(t))), func() { g.trRide(from, t) }}
}

func (g *Game) trStationTalk(a *Actor) []Option {
	from := g.trHere()
	if from == nil {
		return nil
	}
	trips, cut := g.trRailTrips(from.ID)
	var out []Option
	if len(trips) > 0 {
		out = append(out, Option{"{{B|I want a ticket.}}", func() {
			g.Talk = a
			g.Popup = Wrap("\"Trains leave from this platform. Where to?\" {{K|(walking would take days; the train takes hours)}}", 100)
			g.Opts = nil
			for i, t := range trips {
				if i >= 10 {
					break
				}
				g.Opts = append(g.Opts, g.trTripOption(from, t))
			}
			g.Opts = append(g.Opts, Option{"Never mind.", func() { g.talkMenu(g.greeting(a)) }})
			g.Sel = 0
		}})
	}
	for _, l := range cut {
		l := l
		out = append(out, Option{"Why are the trains not running?", func() {
			g.talkMenu(fmt.Sprintf("\"The %s has been closed since Y%d. The war cut the track and nobody has relaid it yet.\"", trLineName(l), l.Cut))
		}})
	}
	return out
}

func trLineName(l TrRailData) string {
	if l.Name != "" {
		return l.Name
	}
	return l.Kind
}

func (g *Game) trDockTalk(a *Actor) []Option {
	from := g.trHere()
	if from == nil {
		return nil
	}
	trips := g.trAirTrips(from.ID)
	if len(trips) == 0 {
		return nil
	}
	return []Option{{"{{B|I want passage by air.}}", func() {
		g.Talk = a
		g.Popup = Wrap("\"Moorings are cast off at the turn of the hour. Where to?\"", 100)
		g.Opts = nil
		for i, t := range trips {
			if i >= 10 {
				break
			}
			g.Opts = append(g.Opts, g.trTripOption(from, t))
		}
		g.Opts = append(g.Opts, Option{"Never mind.", func() { g.talkMenu(g.greeting(a)) }})
		g.Sel = 0
	}}}
}

// trA is "a horse" / "an elephant".
func trA(s string) string {
	if s != "" && strings.ContainsRune("aeiouAEIOU", rune(s[0])) {
		return "an " + s
	}
	return "a " + s
}

func trCap(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func trHas(xs []string, s string) bool {
	for _, x := range xs {
		if x == s {
			return true
		}
	}
	return false
}

func trHasInt(xs []int, n int) bool {
	for _, x := range xs {
		if x == n {
			return true
		}
	}
	return false
}

// trOffer is what an ostler in bd sells: up to 3 mounts fitting the land, the people and the age.
func (g *Game) trOffer(bd *BurgData) []*trMountDef {
	w := g.W
	b := w.Biome[bd.P]
	sphere := ""
	if cu := w.cultByID[bd.Culture]; cu != nil {
		sphere = cu.Sphere
	}
	type sc struct {
		d *trMountDef
		s int
	}
	var xs []sc
	for i := range trMounts {
		d := &trMounts[i]
		if d.Genres != nil && !trHas(d.Genres, g.Genre) {
			continue
		}
		s := 1
		if d.Genres != nil {
			s += 3
		}
		if trHasInt(d.Biomes, b) {
			s += 3
		}
		if trHas(d.Spheres, sphere) {
			s += 4
		}
		if d.Biomes != nil && !trHasInt(d.Biomes, b) && !trHas(d.Spheres, sphere) && d.Genres == nil {
			continue // a regional beast far from home
		}
		xs = append(xs, sc{d, s})
	}
	sort.SliceStable(xs, func(i, j int) bool { return xs[i].s > xs[j].s })
	var out []*trMountDef
	for _, x := range xs {
		if len(out) < 3 {
			out = append(out, x.d)
		}
	}
	// the genre's own mount (motorbike, hover-skiff, clockwork strider...) always gets a slot,
	// even where local beasts outscore it
	has := false
	for _, d := range out {
		has = has || d.Genres != nil
	}
	if !has && len(out) == 3 {
		for _, x := range xs {
			if x.d.Genres != nil {
				out[2] = x.d
				break
			}
		}
	}
	return out
}

func (g *Game) trCartOffer() []*trCartDef {
	var out []*trCartDef
	for i := range trCarts {
		c := &trCarts[i]
		if c.Genres == nil || trHas(c.Genres, g.Genre) {
			out = append(out, c)
		}
	}
	return out
}

func trStableFee() int { return 2 }

// trBuyMount buys d and names it; the old mount (if any) is sold back at a third.
func (g *Game) trBuyMount(d *trMountDef, price int) bool {
	if !g.cuSpend(price) {
		return false
	}
	st := g.tr()
	r := NewRng(fmt.Sprintf("%s:tr:buy:%d:%s", g.Seed, g.Turn, d.ID))
	look := ""
	if len(d.Looks) > 0 {
		look = Pick(r, d.Looks) + " "
	}
	st.Mount = &trMount{Def: d.ID, Name: look + d.Name, Feed: 5}
	if d.Feed == "" {
		st.Mount.Feed = 0
	}
	st.Stable = nil
	return true
}

func (g *Game) trOstlerTalk(a *Actor) []Option {
	bd := g.trHere()
	if bd == nil {
		return nil
	}
	st := g.tr()
	again := func() { g.talkMenu(g.greeting(a)) }
	var out []Option
	disc := func(n int) int { return maxi(1, n*(100-g.Skills["persuasion"]*8)/100) }
	if st.Stable != nil && st.Stable.Burg == bd.ID && st.Stable.Planet == g.W.uniIdx {
		days := maxi(1, (g.Turn-st.Stable.Since)/DayLen)
		fee := days * trStableFee()
		out = append(out, Option{fmt.Sprintf("{{B|Bring out my %s.}} {{K|(%d days stabled)}} — %s", st.Mount.Name, days, g.cuFormat(fee)), func() {
			if !g.cuSpend(fee) {
				g.talkMenu("\"Settle the stable bill first.\"")
				return
			}
			st.Stable = nil
			if d := trMountByID(st.Mount.Def); d != nil && d.Feed != "" {
				st.Mount.Feed = math.Max(st.Mount.Feed, 3)
			}
			st.Mount.Starve = 0
			g.Msg(fmt.Sprintf("Your %s is brought round, groomed and fed.", st.Mount.Name))
			again()
		}})
	} else if st.Stable != nil {
		out = append(out, Option{"Where is my mount?", func() {
			g.talkMenu(fmt.Sprintf("\"Not with me. You left your %s at the stable in %s.\"", st.Mount.Name, st.Stable.Town))
		}})
	}
	if d := g.trWithYou(); d != nil {
		m := st.Mount
		out = append(out, Option{fmt.Sprintf("Stable my %s here. {{K|(%s a day, paid when you collect)}}", m.Name, g.cuFormat(trStableFee())), func() {
			st.Stable = &trStable{Burg: bd.ID, Planet: g.W.uniIdx, Since: g.Turn, Town: bd.Name}
			g.Msg(fmt.Sprintf("You leave your %s in the stable at %s.", m.Name, bd.Name))
			again()
		}})
		if d.Feed != "" {
			price := disc(5 * maxi(2, d.Price/60))
			out = append(out, Option{fmt.Sprintf("Buy 5 days of %s. {{K|(carrying %.0f)}} — %s", trFeedWord(d.Feed), m.Feed, g.cuFormat(price)), func() {
				if g.cuSpend(price) {
					m.Feed += 5
					m.Starve = 0
					g.Msg(fmt.Sprintf("You load 5 days of %s for your %s.", trFeedWord(d.Feed), m.Name))
				}
				again()
			}})
		}
		sell := d.Price / 3
		out = append(out, Option{fmt.Sprintf("Sell my %s. — %s", m.Name, g.cuFormat(sell)), func() {
			got := g.cuEarn(sell)
			g.Msg(fmt.Sprintf("You sell your %s for %s.", m.Name, got))
			st.Mount = nil
			again()
		}})
	}
	if st.Stable == nil || st.Stable.Burg == bd.ID {
		for _, d := range g.trOffer(bd) {
			d := d
			price := disc(d.Price)
			note := fmt.Sprintf("faster on open land, carries %d", d.Carry)
			if d.Machine && d.Rough < 0.5 {
				note = fmt.Sprintf("fast on roads, carries %d", d.Carry)
			}
			out = append(out, Option{fmt.Sprintf("Buy %s. {{K|(%s)}} — %s", trA(d.Name), note, g.cuFormat(price)), func() {
				if st.Mount != nil && st.Stable == nil {
					if od := trMountByID(st.Mount.Def); od != nil {
						g.cuEarn(od.Price / 3)
					}
				}
				if g.trBuyMount(d, price) {
					g.Msg(fmt.Sprintf("You buy %s. It eyes you warily.", trA(st.Mount.Name)))
				} else {
					g.Msg("{{y|You cannot afford it.}}")
				}
				again()
			}})
		}
	}
	for _, c := range g.trCartOffer() {
		c := c
		if st.Cart == c.ID {
			continue
		}
		price := disc(c.Price)
		need := ""
		if c.Draft {
			need = ", needs a mount to pull it"
		}
		out = append(out, Option{fmt.Sprintf("Buy %s. {{K|(+%d carrying%s)}} — %s", trA(c.Name), c.Carry, need, g.cuFormat(price)), func() {
			if g.cuSpend(price) {
				if old := trCartByID(st.Cart); old != nil {
					g.cuEarn(old.Price / 3)
				}
				st.Cart = c.ID
				g.Msg(fmt.Sprintf("You buy %s.", trA(c.Name)))
			}
			again()
		}})
	}
	return out
}

// ---- zones: stations, docks, stables ----

func trZone(g *Game, zn *Zone, _ *Rng) {
	if zn.Kind != "town" || zn.ZX != 1 || zn.ZY != 1 || zn.Burg <= 0 {
		return
	}
	bd := g.W.burgByID[zn.Burg]
	if bd == nil || !bd.Alive {
		return
	}
	if _, _, z := unTownZone(g.W, bd); z != zn.Z {
		return
	}
	ix := trIdx(g.W)
	if zn.Z == 0 && bd.Layer == "surface" {
		g.trPlace(zn, bd, "ostler", "ostler", fmt.Sprintf("An ostler of %s, who sells and stables mounts and carts.", bd.Name), "")
	}
	if len(ix.station[bd.ID]) > 0 {
		trips, _ := g.trRailTrips(bd.ID)
		var lines []string
		for _, li := range ix.station[bd.ID] {
			l := g.W.Transport.Rail[li]
			s := trLineName(l)
			if l.Cut > 0 {
				s += fmt.Sprintf(" — CLOSED since Y%d", l.Cut)
			}
			lines = append(lines, s)
		}
		note := "PLATFORM — " + strings.Join(lines, "; ") + "."
		for i, t := range trips {
			if i >= 6 {
				break
			}
			note += fmt.Sprintf(" To %s: %s.", t.To.Name, tvDuration(t.Minutes))
		}
		g.trPlace(zn, bd, "stationmaster", g.trClerk(), fmt.Sprintf("The %s at %s station, selling tickets.", g.trClerk(), bd.Name), note)
	}
	if ix.dock[bd.ID] {
		note := "AIR DOCK — mooring mast and gangway."
		for i, t := range g.trAirTrips(bd.ID) {
			if i >= 6 {
				break
			}
			note += fmt.Sprintf(" To %s by %s: %s.", t.To.Name, trRideName(t.Kind), tvDuration(t.Minutes))
		}
		g.trPlace(zn, bd, "dockmaster", "dockmaster", fmt.Sprintf("The dockmaster of %s, who books passage by air.", bd.Name), note)
	}
}

func (g *Game) trClerk() string {
	switch g.Genre {
	case "cyberpunk", "solarpunk", "atompunk":
		return "ticket agent"
	case "wasteland":
		return "rail boss"
	}
	return "ticket clerk"
}

// trPlace adds an NPC on a free street tile (side rng) and, with note, a timetable plaque beside them.
func (g *Game) trPlace(zn *Zone, bd *BurgData, role, title, desc, note string) {
	for _, a := range zn.Actors {
		if a.Role == role {
			return
		}
	}
	r := NewRng(g.Seed + ":tr:" + role + ":" + zn.Key)
	x, y, ok := zn.FreeSpot(r, func(x, y int) bool {
		t := zn.At(x, y)
		return (t == TCobble || t == TRoad || t == TDirt || t == TGrass) && x > 2 && x < MW-3 && y > 1 && y < MH-2
	})
	if !ok {
		if x, y, ok = zn.FreeSpot(r, nil); !ok {
			return
		}
	}
	a := g.makeNPC(role, bd, g.W.cultByID[bd.Culture], r)
	a.Title, a.Desc, a.AI = title, desc, "still"
	a.F = CCyan
	a.X, a.Y, a.HomeX, a.HomeY = x, y, x, y
	zn.Actors = append(zn.Actors, a)
	if note == "" {
		return
	}
	for _, d := range [][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
		px, py := x+d[0], y+d[1]
		if px < 1 || py < 1 || px >= MW-1 || py >= MH-1 || zn.ActorAt(px, py) != nil || !zn.Walkable(px, py) {
			continue
		}
		zn.Set(px, py, TInscription)
		if zn.Notes == nil {
			zn.Notes = map[int]string{}
		}
		zn.Notes[py*MW+px] = note
		// a short platform edge of pillars
		for _, e := range [][2]int{{px - 2, py}, {px + 2, py}} {
			if e[0] > 0 && e[0] < MW-1 && zn.ActorAt(e[0], e[1]) == nil && (zn.At(e[0], e[1]) == TCobble || zn.At(e[0], e[1]) == TRoad) {
				zn.Set(e[0], e[1], TPillar)
			}
		}
		return
	}
}

// ---- world map and sidebar ----

// trWorldGlyph draws rail track across known parasangs (render.go drawWorld).
func (g *Game) trWorldGlyph(p int, gl rune, f byte) (rune, byte) {
	if g.W.Transport == nil {
		return gl, f
	}
	lis := trIdx(g.W).railAt[p]
	if len(lis) == 0 {
		return gl, f
	}
	if g.W.Transport.Rail[lis[0]].Cut > 0 {
		return '=', CDarkGray
	}
	return '=', CGray
}

// trInfo is the world-map cursor's transport lines.
func (g *Game) trInfo(p int) []string {
	w := g.W
	if w.Transport == nil {
		return nil
	}
	ix := trIdx(w)
	var out []string
	for _, li := range ix.railAt[p] {
		l := w.Transport.Rail[li]
		s := "{{W|" + trLineName(l) + "}} (" + l.Kind + ")"
		if l.Cut > 0 {
			s += fmt.Sprintf(" {{R|cut Y%d}}", l.Cut)
		}
		out = append(out, s)
	}
	if bid := w.Burg[p]; bid > 0 {
		if len(ix.station[bid]) > 0 {
			out = append(out, "{{C|Railway station}}")
		}
		if ix.dock[bid] {
			out = append(out, "{{C|Air dock}}")
		}
	}
	return out
}

func trSide(g *Game) []string {
	st := g.tr()
	if st.Mount == nil && st.Cart == "" {
		return nil
	}
	var parts []string
	if st.Mount != nil {
		d := trMountByID(st.Mount.Def)
		s := "{{y|" + titleCase(st.Mount.Name) + "}}"
		switch {
		case st.Stable != nil:
			s += " {{K|stabled in " + st.Stable.Town + "}}"
		case d != nil && d.Feed != "":
			s += fmt.Sprintf(" {{K|%.0fd %s}}", st.Mount.Feed, trFeedWord(d.Feed))
		}
		parts = append(parts, s)
	}
	if c := trCartByID(st.Cart); c != nil {
		parts = append(parts, "{{K|"+c.Name+"}}")
	}
	return []string{strings.Join(parts, ", ")}
}

func init() {
	zoneHooks = append(zoneHooks, trZone)
	talkHooks = append(talkHooks, trTalk)
	sideHooks = append(sideHooks, trSide)
}
