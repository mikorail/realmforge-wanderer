package engine

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"sync"
)

// Orders, cults and guilds: joinable societies generated deterministically from the world
// (religions, uprisings, great cities, leagues, academies). Membership lives in g.ext("orders").

type OrderReq struct {
	Kind    string `json:"kind"`    // rep attr task
	Faction string `json:"faction"` // rep: faction key
	Min     int    `json:"min"`
	Attr    int    `json:"attr"`
	Desc    string `json:"desc"`
}

type Order struct {
	ID        string
	Name      string
	Kind      string // holy monastic cult secret thieves merchant scholars
	Creed     string
	Burg      int
	WX, WY    int
	ZX, ZY    int
	District  string
	Faith     int // parent religion (0 none)
	State     int // realm of the home city
	Rival     string
	Secret    bool
	Req       OrderReq
	Ranks     [5]string
	Power     string
	ItemName  string
	Leader    string
	LeaderTit string
	TaskKinds []string
	Sign      string
	HomeName  string
}

func (o *Order) RepKey() string { return "order:" + o.ID }

var (
	orderCacheMu sync.Mutex
	orderCache   = map[*WorldData][]*Order{}
)

var orderRanks = map[string][5]string{
	"holy":              {"Novice", "Squire", "Knight", "Warden", "Grand Marshal"},
	"monastic":          {"Postulant", "Acolyte", "Brother", "Contemplative", "Abbot"},
	"cult":              {"Seeker", "Votary", "Herald", "Hierophant", "Prophet"},
	"thieves":           {"Footpad", "Cutpurse", "Shadow", "Fence-Lord", "Guildmaster"},
	"merchant":          {"Apprentice", "Journeyman", "Factor", "Master Merchant", "Guildmaster"},
	"scholars":          {"Auditor", "Scholar", "Fellow", "Magister", "Rector"},
	"secret:":           {"Initiate", "Adept", "Keeper", "Elder", "Grand Master"},
	"secret:fantasy":    {"Apprentice", "Adept", "Magister", "Arch-Magister", "Grand Archon"},
	"secret:mythic":     {"Whisperer", "Oracle", "Seer", "High Seer", "Voice of the Hidden"},
	"secret:steampunk":  {"Cog", "Piston", "Governor", "Engineer-Lord", "Master of the Lodge"},
	"secret:dieselpunk": {"Runner", "Enforcer", "Captain", "Underboss", "Don"},
	"secret:atompunk":   {"Recruit", "Technician", "Operative", "Director", "Chairman"},
	"secret:cyberpunk":  {"Script", "Runner", "Daemon", "Architect", "Root"},
	"secret:biopunk":    {"Graft", "Splicer", "Chimera", "Progenitor", "Prime Strain"},
	"secret:solarpunk":  {"Sprout", "Gardener", "Keeper", "Elder Keeper", "Heartwood"},
	"secret:wasteland":  {"Stray", "Scrounger", "Warden", "Overseer", "Deep Elder"},
	"secret:twilight":   {"Mourner", "Whisperer", "Hollow Keeper", "Pale Elder", "Eye-Bearer"},
}

var orderPowers = map[string][]string{
	"holy":     {"forcebubble", "stunforce", "heal"},
	"monastic": {"clairvoyance", "regen", "slumber"},
	"cult":     {"syphon", "leech", "slumber"},
	"secret":   {"arcanebolt", "phase", "psiblast", "glimmer"},
	"thieves":  {"clipping", "sprint", "glimmer"},
	"merchant": {"beguile", "sprint", "forcebubble"},
	"scholars": {"clairvoyance", "arcanebolt", "timedilation"},
}

var orderTaskKinds = map[string][]string{
	"holy":     {"kill", "pray", "tithe", "relic"},
	"monastic": {"pray", "tithe", "relic"},
	"cult":     {"kill", "pray", "relic", "tithe"},
	"secret":   {"kill", "relic", "tithe"},
	"thieves":  {"tithe", "relic", "kill"},
	"merchant": {"tithe", "kill", "relic"},
	"scholars": {"relic", "tithe", "kill"},
}

var orderRoles = map[string]string{
	"holy": "priest", "monastic": "scholar", "cult": "villager", "secret": "scholar", "thieves": "villager", "merchant": "merchant", "scholars": "scholar",
}

var orderDistrict = map[string]string{
	"holy": "temple", "monastic": "temple", "cult": "slums", "secret": "slums", "thieves": "slums", "merchant": "market", "scholars": "academy",
}

var orderLeaderTitle = map[string]string{
	"holy": "Grand Marshal", "monastic": "Abbot", "cult": "Prophet", "secret": "Grand Master", "thieves": "Guildmaster", "merchant": "Guildmaster", "scholars": "Rector",
}

var cultRe = regexp.MustCompile(`\b(Order|Children|Sisters|Brothers|Seekers|Disciples|Brotherhood|Sisterhood|Covenant|Circle|Heirs|Followers|Cult|Cabal|Congregation|Church)\s+of\s+(?:the\s+)?([A-Z][\w’']*(?:\s+[A-Z][\w’']*){0,2})`)

func orderSlug(s string) string {
	var b strings.Builder
	dash := false
	for _, c := range strings.ToLower(s) {
		if c >= 'a' && c <= 'z' || c >= '0' && c <= '9' {
			b.WriteRune(c)
			dash = false
		} else if !dash && b.Len() > 0 {
			b.WriteByte('-')
			dash = true
		}
	}
	return strings.Trim(b.String(), "-")
}

// Orders returns every order of the world (cached per world, never saved).
func (g *Game) Orders() []*Order {
	orderCacheMu.Lock()
	defer orderCacheMu.Unlock()
	if os, ok := orderCache[g.W]; ok {
		return os
	}
	os := g.genOrders()
	orderCache[g.W] = os
	return os
}

func (g *Game) OrderByID(id string) *Order {
	for _, o := range g.Orders() {
		if o.ID == id {
			return o
		}
	}
	return nil
}

func (g *Game) orderHome(o *Order, bd *BurgData, kind string) {
	o.Burg, o.State = bd.ID, bd.State
	o.WX, o.WY = bd.P%g.W.W, bd.P/g.W.W
	o.ZX, o.ZY, o.District = 1, 1, "plaza"
	o.HomeName = bd.Name
	cp := g.cityPlanFor(bd)
	var keys [][2]int
	for k := range cp.districts {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i][1] != keys[j][1] {
			return keys[i][1] < keys[j][1]
		}
		return keys[i][0] < keys[j][0]
	})
	for _, k := range keys {
		if cp.districts[k] == kind {
			o.ZX, o.ZY, o.District = k[0], k[1], kind
			return
		}
	}
	o.District = cp.districts[[2]int{1, 1}]
}

func (g *Game) genOrders() []*Order {
	w := g.W
	var out []*Order
	used := map[string]bool{}
	add := func(o *Order, bd *BurgData, kind string) *Order {
		if o.ID == "" {
			o.ID = orderSlug(o.Name)
		}
		for used[o.ID] {
			o.ID += "-x"
		}
		used[o.ID] = true
		g.orderHome(o, bd, orderDistrict[kind])
		if o.Kind == "secret" {
			o.Ranks = orderRanks["secret:"+g.W.Genre]
			if o.Ranks[0] == "" {
				o.Ranks = orderRanks["secret:fantasy"]
			}
		} else {
			o.Ranks = orderRanks[o.Kind]
		}
		o.TaskKinds = orderTaskKinds[o.Kind]
		o.LeaderTit = orderLeaderTitle[o.Kind]
		if o.Kind == "secret" {
			o.LeaderTit = o.Ranks[4]
		}
		r := NewRng(w.Seed + ":order:" + o.ID)
		pw := orderPowers[o.Kind]
		o.Power = pw[r.Int(0, len(pw)-1)]
		o.Leader = g.cultureName(w.cultByID[bd.Culture], r)
		o.Sign = Pick(r, []string{"touches two fingers to the brow", "turns a ring thrice on the finger", "draws a slow circle in the air", "taps the left wrist twice", "folds a coin into the palm", "lets the sleeve slip to show a mark"})
		out = append(out, o)
		return o
	}
	burgs := func() []*BurgData {
		var bs []*BurgData
		for i := range w.Burgs {
			b := &w.Burgs[i]
			if b.Alive && b.Layer == "surface" {
				bs = append(bs, b)
			}
		}
		sort.SliceStable(bs, func(i, j int) bool {
			if bs[i].Pop != bs[j].Pop {
				return bs[i].Pop > bs[j].Pop
			}
			return bs[i].ID < bs[j].ID
		})
		return bs
	}()
	if len(burgs) == 0 {
		return nil
	}
	// religions: biggest city of each faith hosts its order
	faithBurg := map[int]*BurgData{}
	for _, b := range burgs {
		rid := w.Religion[clampi(b.P, 0, len(w.Religion)-1)]
		if w.relByID[rid] != nil && faithBurg[rid] == nil {
			faithBurg[rid] = b
		}
	}
	var rids []int
	for id := range faithBurg {
		rids = append(rids, id)
	}
	sort.Ints(rids)
	holyOf := map[int]*Order{}
	militant := map[string]bool{"militant": true, "evangelical": true, "liberation": true, "crusade": true}
	for _, rid := range rids {
		rel := w.relByID[rid]
		bd := faithBurg[rid]
		god := nonEmpty(rel.Deity, rel.Name)
		r := NewRng(w.Seed + fmt.Sprint(":holy", rid))
		o := &Order{Faith: rid}
		if militant[rel.Doctrine] {
			o.Kind = "holy"
			o.Name = Pick(r, []string{"Knights of ", "Sworn Blades of ", "Wardens of ", "The Hand of "}) + god
			o.Creed = fmt.Sprintf("A militant order sworn to defend the %s and carry the word of %s by the blade.", rel.Name, god)
			o.Req = OrderReq{Kind: "rep", Faction: fmt.Sprintf("faith:%d", rid), Min: 100, Desc: fmt.Sprintf("100 reputation with the %s", rel.Name)}
		} else {
			o.Kind = "monastic"
			o.Name = Pick(r, []string{"Cloister of ", "Contemplatives of ", "Silent Brothers of ", "Hermits of "}) + god
			o.Creed = fmt.Sprintf("Monks of the %s who withdraw from the world to keep vigil for %s.", rel.Name, god)
			o.Req = OrderReq{Kind: "rep", Faction: fmt.Sprintf("faith:%d", rid), Min: 50, Desc: fmt.Sprintf("50 reputation with the %s", rel.Name)}
		}
		holyOf[rid] = add(o, bd, o.Kind)
	}
	// heretic cults from the chronicle
	seenCult := map[string]bool{}
	ncult := 0
	for _, ev := range w.Events {
		if ev.Kind != "faith" && ev.Kind != "uprising" || ncult >= 8 {
			continue
		}
		m := cultRe.FindStringSubmatch(ev.Text)
		if m == nil {
			continue
		}
		name := strings.TrimSpace("The " + m[1] + " of " + m[2])
		if seenCult[name] {
			continue
		}
		var bd *BurgData
		if ev.P >= 0 && ev.P < len(w.Burg) {
			bd = w.burgByID[w.Burg[ev.P]]
		}
		if bd == nil || !bd.Alive || bd.Layer != "surface" {
			bd = nil
			for _, b := range burgs {
				if len(b.Name) >= 4 && strings.Contains(ev.Text, b.Name) {
					bd = b
					break
				}
			}
		}
		if bd == nil {
			continue
		}
		seenCult[name] = true
		ncult++
		rid := w.Religion[clampi(bd.P, 0, len(w.Religion)-1)]
		txt := ev.Text
		if len(txt) > 120 {
			txt = txt[:120] + "..."
		}
		o := &Order{Name: name, Kind: "cult", Faith: rid, Secret: true,
			Creed: fmt.Sprintf("A heretic movement of the year %d. \"%s\"", ev.Y, txt),
			Req:   OrderReq{Kind: "task", Desc: "pass the initiation rite"}}
		add(o, bd, "cult")
	}
	// secret society of the genre in the largest city
	{
		type sec struct{ name, creed string }
		secs := map[string]sec{
			"fantasy":    {"The Veiled Cabal", "An arcane cabal that hoards forbidden knowledge and rules through whispered favours."},
			"mythic":     {"The Hidden Pantheon", "Mortals who claim to speak for gods the temples have forgotten."},
			"steampunk":  {"The Brass Lodge", "A lodge of engineers and financiers who quietly set the course of the age."},
			"dieselpunk": {"The Iron Syndicate", "A syndicate that owns the unions, the ports and a good share of the police."},
			"atompunk":   {"The Atomic Vanguard", "A cell of technocrats preparing for a future they are sure is coming."},
			"cyberpunk":  {"The Ghost Collective", "Netrunners who live between the grid's cracks and sell what they find there."},
			"biopunk":    {"The Chimera Covenant", "A splice cult that believes flesh is a draft to be rewritten."},
			"solarpunk":  {"The Seed-Keepers", "Gardeners of forbidden seeds who plan the green overthrow of the old order."},
			"wasteland":  {"The Deep Listeners", "Bunker-born archivists who hoard sealed vaults, dead frequencies and the last working maps."},
			"twilight":   {"The Pale Choir", "Mourners who have seen past the veil and sing for the gods the lords buried."},
		}
		s, ok := secs[w.Genre]
		if !ok {
			s = secs["fantasy"]
		}
		o := &Order{Name: s.name, Kind: "secret", Secret: true, Creed: s.creed,
			Req: OrderReq{Kind: "task", Desc: "pass the initiation rite"}}
		add(o, burgs[0], "secret")
	}
	// thieves' guilds per great city, scholar societies in academies
	nsch := 0
	for _, bd := range burgs {
		if w.cityTier[bd.ID] != 2 {
			continue
		}
		var nm, cr string
		switch w.Genre {
		case "cyberpunk":
			nm, cr = bd.Name+" Street Cartel", "Fixers, couriers and cut-rate cutthroats of the lower levels."
		case "steampunk", "dieselpunk", "atompunk":
			nm, cr = bd.Name+" Underworld Syndicate", "Pickpockets, safecrackers and fences with a union card."
		case "biopunk", "solarpunk":
			nm, cr = bd.Name+" Black Market Ring", "Smugglers of seeds, organs and secrets."
		case "wasteland":
			nm, cr = bd.Name+" Tunnel Gang", "Scavengers, tollmen and knife-fighters who rule the dark passages."
		case "twilight":
			nm, cr = bd.Name+" Gutter Brotherhood", "Grave-robbers, relic-fences and cutthroats of the plague streets."
		default:
			nm, cr = bd.Name+" Thieves' Guild", "Cutpurses, burglars and fences bound by an oath of silence."
		}
		add(&Order{Name: nm, Kind: "thieves", Secret: true, Creed: cr,
			Req: OrderReq{Kind: "attr", Attr: AGI, Min: 14, Desc: "Agility 14 (nimble hands)"}}, bd, "thieves")
	}
	for _, bd := range burgs {
		if w.cityTier[bd.ID] == 0 || nsch >= 6 {
			continue
		}
		if hasDistrict(g.cityPlanFor(bd), "academy") {
			nsch++
			add(&Order{Name: "The " + bd.Name + " Society of Letters", Kind: "scholars",
				Creed: "Scholars who collect lore, relics and the truth behind the chronicles.",
				Req:   OrderReq{Kind: "attr", Attr: INT, Min: 14, Desc: "Intelligence 14"}}, bd, "scholars")
		}
	}
	// merchant guilds per league (or per great city when the world names none)
	leagues := map[string]*BurgData{}
	var lnames []string
	for _, b := range burgs {
		if b.League != "" && leagues[b.League] == nil {
			leagues[b.League] = b
			lnames = append(lnames, b.League)
		}
	}
	sort.Strings(lnames)
	merchReq := OrderReq{Kind: "attr", Attr: EGO, Min: 14, Desc: "Ego 14 (a persuasive tongue)"}
	for _, l := range lnames {
		add(&Order{Name: "The " + l + " Merchant League", Kind: "merchant", Creed: "A league of traders who guard the roads and the price of salt.", Req: merchReq}, leagues[l], "merchant")
	}
	if len(lnames) == 0 {
		for _, bd := range burgs {
			if w.cityTier[bd.ID] == 2 {
				add(&Order{Name: "The " + bd.Name + " Merchants' Guild", Kind: "merchant", Creed: "The counting-houses of " + bd.Name + ", rich enough to buy a regiment.", Req: merchReq}, bd, "merchant")
			}
		}
	}
	// rivalries
	first := func(pred func(o *Order) bool) *Order {
		for _, o := range out {
			if pred(o) {
				return o
			}
		}
		return nil
	}
	for _, o := range out {
		var rv *Order
		switch o.Kind {
		case "holy", "monastic":
			rv = first(func(c *Order) bool { return c.Kind == "cult" && c.Faith == o.Faith })
			if rv == nil {
				rv = first(func(c *Order) bool { return c.Kind == "cult" })
			}
		case "cult":
			rv = holyOf[o.Faith]
			if rv == nil {
				rv = first(func(c *Order) bool { return c.Kind == "holy" || c.Kind == "monastic" })
			}
		case "secret":
			rv = first(func(c *Order) bool { return c.Kind == "holy" })
			if rv == nil {
				rv = first(func(c *Order) bool { return c.Kind == "monastic" })
			}
		case "thieves":
			rv = first(func(c *Order) bool { return c.Kind == "merchant" && c.Burg == o.Burg })
			if rv == nil {
				rv = first(func(c *Order) bool { return c.Kind == "merchant" })
			}
		case "merchant":
			rv = first(func(c *Order) bool { return c.Kind == "thieves" && c.Burg == o.Burg })
			if rv == nil {
				rv = first(func(c *Order) bool { return c.Kind == "thieves" })
			}
		case "scholars":
			rv = first(func(c *Order) bool { return c.Kind == "cult" })
		}
		if rv != nil && rv != o {
			o.Rival = rv.ID
		}
	}
	for _, o := range out {
		if o.Rival != "" {
			if rv := first(func(c *Order) bool { return c.ID == o.Rival }); rv != nil && rv.Rival == "" {
				rv.Rival = o.ID
			}
		}
	}
	return out
}

// ---- saved state ----

type OrderTask struct {
	Kind    string `json:"kind"` // kill pray tithe relic
	Text    string `json:"text"`
	Need    int    `json:"need"`
	Have    int    `json:"have"`
	Target  string `json:"target"` // kill: creature faction or "agent" (rival members)
	Ready   bool   `json:"ready"`
	Init    bool   `json:"init"`
	Reward  int    `json:"reward"`
	Gold    int    `json:"gold"`
	Zone    string `json:"zone"` // relic: dungeon zone key
	Site    string `json:"site"`
	Spawned bool   `json:"spawned"`
	LastAct int    `json:"lastAct"`
}

type OrderMember struct {
	Rank      int        `json:"rank"` // 0 = on probation (initiation pending)
	Done      int        `json:"done"`
	Task      *OrderTask `json:"task,omitempty"`
	LastPay   int        `json:"lastPay"`
	LastCall  int        `json:"lastCall"`
	GotPower  string     `json:"gotPower,omitempty"`
	GotItem   bool       `json:"gotItem,omitempty"`
	JoinedYr  int        `json:"joinedYr"`
	TaskCount int        `json:"taskCount"`
}

type orderState struct {
	Mem  map[string]*OrderMember `json:"mem"`
	Seen map[string]bool         `json:"seen"`
}

func (g *Game) orderState() *orderState {
	st := g.ext("orders", func() any { return &orderState{} }).(*orderState)
	if st.Mem == nil {
		st.Mem = map[string]*OrderMember{}
	}
	if st.Seen == nil {
		st.Seen = map[string]bool{}
	}
	return st
}

// OrderMembership returns the player's standing in an order (nil if not a member).
func (g *Game) OrderMembership(id string) *OrderMember { return g.orderState().Mem[id] }

var orderRankDone = [5]int{0, 2, 4, 7, 11}

func orderRankFor(done int) int {
	r := 1
	for i, n := range orderRankDone {
		if done >= n {
			r = i + 1
		}
	}
	return r
}

func (g *Game) orderMoney(rank int) int { return 25 + rank*20 + g.P.Level*4 }

// ---- joining ----

func (g *Game) orderMeets(o *Order) (bool, string) {
	switch o.Req.Kind {
	case "rep":
		if g.Rep[o.Req.Faction] >= o.Req.Min {
			return true, ""
		}
		return false, "Requires " + o.Req.Desc + "."
	case "attr":
		if g.P.Score(o.Req.Attr) >= o.Req.Min {
			return true, ""
		}
		return false, "Requires " + o.Req.Desc + "."
	}
	return true, ""
}

func (g *Game) orderJoin(o *Order) {
	st := g.orderState()
	if st.Mem[o.ID] != nil {
		return
	}
	m := &OrderMember{JoinedYr: g.W.Year}
	st.Mem[o.ID] = m
	st.Seen[o.ID] = true
	if o.Req.Kind == "task" {
		m.Task = g.newOrderTask(o, m, true)
		g.Msg("{{Y|" + o.Name + "}}: the leader sets you a rite of initiation.")
		return
	}
	g.orderAdmit(o, m)
}

func (g *Game) orderAdmit(o *Order, m *OrderMember) {
	m.Rank = 1
	m.Task = nil
	if g.Rep[o.RepKey()] < 50 {
		g.Rep[o.RepKey()] = 50
	}
	g.Msg(fmt.Sprintf("{{G|You are sworn into %s as a %s.}}", o.Name, o.Ranks[0]))
	g.deed("order", fmt.Sprintf("%s was sworn into %s.", g.P.Name, o.Name), 0, 0)
	if rv := g.OrderByID(o.Rival); rv != nil {
		g.AddRep(rv.RepKey(), -80, " (rival order)")
	}
	switch o.Kind {
	case "holy", "monastic":
		g.AddRep(fmt.Sprintf("faith:%d", o.Faith), 60, "")
	case "cult":
		if o.Faith > 0 {
			g.AddRep(fmt.Sprintf("faith:%d", o.Faith), -150, " for consorting with heretics")
		}
		g.AddRep(fmt.Sprintf("state:%d", o.State), -30, "")
	case "secret", "thieves":
		g.AddRep(fmt.Sprintf("state:%d", o.State), -25, "")
	case "merchant":
		g.AddRep(fmt.Sprintf("state:%d", o.State), 40, "")
	}
}

// ---- tasks ----

func (g *Game) newOrderTask(o *Order, m *OrderMember, init bool) *OrderTask {
	r := NewRng(fmt.Sprintf("%s:%s:task:%d:%d", g.W.Seed, o.ID, m.TaskCount, m.Rank))
	m.TaskCount++
	rank := maxi(m.Rank, 1)
	t := &OrderTask{Init: init, Reward: g.orderMoney(rank)}
	kinds := o.TaskKinds
	kind := kinds[r.Int(0, len(kinds)-1)]
	if init && kind == "tithe" {
		kind = kinds[0]
	}
	rv := g.OrderByID(o.Rival)
	switch kind {
	case "kill":
		t.Kind, t.Need = "kill", r.Int(3, 4+rank)
		switch {
		case rv != nil && o.Kind != "holy" && o.Kind != "merchant" && o.Kind != "scholars" || rv != nil && r.Chance(0.5):
			t.Target = "agent"
			t.Text = fmt.Sprintf("Slay %d agents of %s (they will find you in the wilds).", t.Need, rv.Name)
		case o.Kind == "holy" || o.Kind == "scholars":
			t.Target = "undead"
			t.Text = fmt.Sprintf("Destroy %d of the restless dead.", t.Need)
		default:
			t.Target = "outlaws"
			t.Text = fmt.Sprintf("Slay %d outlaws preying on the roads.", t.Need)
		}
	case "pray":
		t.Kind, t.Need = "pray", r.Int(2, 3)
		t.Text = fmt.Sprintf("Pray at shrines (altars) %d times, a while apart.", t.Need)
	case "tithe":
		t.Kind, t.Need = "tithe", 1
		t.Gold = g.orderMoney(rank) + 10
		t.Text = fmt.Sprintf("Pay a tithe of %d %s to the order.", t.Gold, g.Sk.Currency)
	case "relic":
		z, name, ok := g.orderRelicSite(o, r)
		if !ok {
			t.Kind, t.Need, t.Target = "kill", 3, "outlaws"
			t.Text = "Slay 3 outlaws preying on the roads."
			break
		}
		t.Kind, t.Need, t.Zone, t.Site = "relic", 1, z, name
		t.Text = fmt.Sprintf("Recover a relic of the order from the depths of %s.", name)
	}
	return t
}

// orderRelicSite picks a dungeon near the order's home and returns its first-level zone key.
func (g *Game) orderRelicSite(o *Order, r *Rng) (string, string, bool) {
	type cand struct {
		key, name string
		d         int
	}
	var cs []cand
	for rad := 0; rad <= 14 && len(cs) < 5; rad++ {
		for dy := -rad; dy <= rad; dy++ {
			for dx := -rad; dx <= rad; dx++ {
				if maxi(abs(dx), abs(dy)) != rad {
					continue
				}
				wx, wy := o.WX+dx, o.WY+dy
				if !g.W.In(wx, wy) {
					continue
				}
				for zy := 0; zy < 3; zy++ {
					for zx := 0; zx < 3; zx++ {
						if s := g.siteAt(wx, wy, zx, zy); s != nil && s.Kind == "dungeon" && s.Depth >= 1 {
							cs = append(cs, cand{ZoneKey(wx, wy, zx, zy, 1), s.Name, rad})
						}
					}
				}
			}
		}
	}
	if len(cs) == 0 {
		return "", "", false
	}
	c := cs[r.Int(0, len(cs)-1)]
	return c.key, c.name, true
}

func (g *Game) orderTaskDone(o *Order, m *OrderMember) {
	t := m.Task
	if t == nil || t.Ready {
		return
	}
	t.Ready = true
	g.Msg(fmt.Sprintf("{{G|Task fulfilled for %s: report to %s.}}", o.Name, o.Leader))
}

func (g *Game) orderTaskProgress(o *Order, m *OrderMember, n int) {
	t := m.Task
	if t == nil || t.Ready {
		return
	}
	t.Have = mini(t.Need, t.Have+n)
	if t.Have >= t.Need {
		g.orderTaskDone(o, m)
	} else {
		g.Msg(fmt.Sprintf("{{c|%s: %d/%d.}}", o.Name, t.Have, t.Need))
	}
}

func (g *Game) hasRelic(o *Order) *Item {
	for _, it := range g.P.Inv {
		if it.Text == "order-relic:"+o.ID {
			return it
		}
	}
	return nil
}

// orderReport hands in a finished task and applies initiation or rank-ups.
func (g *Game) orderReport(o *Order, m *OrderMember) string {
	t := m.Task
	if t == nil || !t.Ready && t.Kind != "relic" {
		return "\"You have nothing to report.\""
	}
	if t.Kind == "relic" && !t.Ready {
		if it := g.hasRelic(o); it != nil {
			t.Ready = true
		} else {
			return "\"Where is the relic?\""
		}
	}
	if t.Kind == "relic" {
		for i, it := range g.P.Inv {
			if it.Text == "order-relic:"+o.ID {
				g.P.Inv = append(g.P.Inv[:i], g.P.Inv[i+1:]...)
				break
			}
		}
	}
	init := t.Init
	m.Task = nil
	g.P.Gold += t.Reward
	g.AddRep(o.RepKey(), 30, "")
	if init {
		g.orderAdmit(o, m)
		return fmt.Sprintf("\"The rite is done. You are one of us now, %s.\" (+%d %s)", o.Ranks[0], t.Reward, g.Sk.Currency)
	}
	m.Done++
	text := fmt.Sprintf("\"Well done.\" (+%d %s)", t.Reward, g.Sk.Currency)
	for nr := orderRankFor(m.Done); m.Rank < nr && m.Rank < 5; {
		m.Rank++
		text += "\n\n" + g.orderPromote(o, m)
	}
	return text
}

func (g *Game) orderPromote(o *Order, m *OrderMember) string {
	title := o.Ranks[m.Rank-1]
	g.Msg(fmt.Sprintf("{{W|You rise to %s in %s.}}", title, o.Name))
	g.deed("order", fmt.Sprintf("%s rose to %s of %s.", g.P.Name, title, o.Name), 0, 0)
	txt := fmt.Sprintf("{{Y|Promoted to %s.}}", title)
	switch m.Rank {
	case 2:
		txt += " You may now draw a stipend and requisition from the order's stores."
	case 3:
		if pd := powerByID[o.Power]; pd != nil && m.GotPower == "" {
			if g.P.Powers == nil {
				g.P.Powers = map[string]int{}
			}
			g.P.Powers[o.Power] = clampi(g.P.Powers[o.Power]+1, 1, 3)
			m.GotPower = o.Power
			txt += fmt.Sprintf(" The order teaches you {{M|%s}}.", g.Sk.PowerName(o.Power))
			g.Msg(fmt.Sprintf("{{M|You learn %s from %s.}}", g.Sk.PowerName(o.Power), o.Name))
		}
	case 4:
		txt += " Retainers of the order will follow you, and the armoury prices fall."
	case 5:
		if !m.GotItem {
			m.GotItem = true
			it := g.orderUniqueItem(o)
			g.give(it)
			txt += fmt.Sprintf(" You are given {{Y|%s}}.", it.Name)
			g.Msg(fmt.Sprintf("{{Y|%s}} is entrusted to you.", it.Name))
		}
	}
	return txt
}

func (g *Game) orderUniqueItem(o *Order) *Item {
	r := NewRng(g.W.Seed + ":orderitem:" + o.ID)
	var base *Item
	switch o.Kind {
	case "holy":
		base = g.MakeWeapon("longblade", 4, r)
	case "monastic":
		base = g.MakeArmor("head", 4, r)
	case "cult":
		base = g.MakeWeapon("cudgel", 4, r)
	case "thieves":
		base = g.MakeWeapon("shortblade", 4, r)
	case "merchant":
		base = g.MakeArmor("back", 4, r)
	default:
		base = g.MakeOddity(r, 3)
	}
	short := strings.TrimPrefix(o.Name, "The ")
	name := map[string]string{"holy": "Oathblade of ", "monastic": "Cowl of ", "cult": "Scepter of ", "thieves": "Shadowfang of ", "merchant": "Mantle of ", "scholars": "Codex-Lens of ", "secret": "Signet of "}[o.Kind] + short
	o.ItemName = name
	return g.MakeUnique(base, name, fmt.Sprintf("Given to the %s of %s for faithful service.", strings.ToLower(o.Ranks[4]), o.Name))
}

func (g *Game) orderDiscount(m *OrderMember) int {
	if m == nil {
		return 0
	}
	d := 5 * m.Rank
	if m.Rank >= 4 {
		d += 10
	}
	return d
}

// ---- hooks ----

func init() {
	zoneHooks = append(zoneHooks, orderZoneHook)
	talkHooks = append(talkHooks, orderTalkHook)
	killHooks = append(killHooks, orderKillHook)
	enterHooks = append(enterHooks, orderEnterHook)
	turnHooks = append(turnHooks, orderTurnHook)
	sideHooks = append(sideHooks, orderSide)
	mainKeys["O"] = func(g *Game) { g.Mode = "orders"; g.Sel = 0 }
	modeKeys["orders"] = orderKeys
	modeDraws["orders"] = func(g *Game) { g.drawOrders() }
}

func orderZoneHook(g *Game, zn *Zone, _ *Rng) {
	if zn.Kind != "town" || zn.Z != 0 || zn.Burg < 0 {
		return
	}
	bd := g.W.burgByID[zn.Burg]
	if bd == nil {
		return
	}
	cu := g.W.cultByID[bd.Culture]
	for _, o := range g.Orders() {
		if o.Burg != zn.Burg {
			continue
		}
		r := NewRng(g.W.Seed + ":ordnpc:" + o.ID + zn.Key)
		if zn.ZX == o.ZX && zn.ZY == o.ZY {
			a := g.makeNPC(orderRoles[o.Kind], bd, cu, r)
			a.Name, a.Proper = o.Leader, true
			a.Title = strings.ToLower(o.LeaderTit) + " of " + strings.TrimPrefix(o.Name, "The ")
			a.Desc = fmt.Sprintf("%s, %s of %s. %s", o.Leader, strings.ToLower(o.LeaderTit), o.Name, o.Creed)
			a.Level += 3
			a.MaxHP += 20
			a.HP = a.MaxHP
			a.AI = "still"
			a.Set("ord:" + o.ID)
			g.placeOrderNPC(zn, a, r)
		} else if o.Secret && o.Kind != "cult" {
			// a rank-and-file member lurks in the other districts of the city
			a := g.makeNPC("villager", bd, cu, r)
			a.Title = "member"
			a.Set("ordm:" + o.ID)
			g.placeOrderNPC(zn, a, r)
		}
	}
}

func (g *Game) placeOrderNPC(zn *Zone, a *Actor, r *Rng) {
	x, y, ok := zn.FreeSpot(r, nil)
	if !ok {
		return
	}
	a.X, a.Y, a.HomeX, a.HomeY = x, y, x, y
	zn.Actors = append(zn.Actors, a)
}

func orderFlag(a *Actor, prefix string) string {
	for k, v := range a.Flags {
		if v && strings.HasPrefix(k, prefix) {
			return k[len(prefix):]
		}
	}
	return ""
}

func orderTalkHook(g *Game, a *Actor) []Option {
	if id := orderFlag(a, "ord:"); id != "" {
		o := g.OrderByID(id)
		if o == nil {
			return nil
		}
		return []Option{{fmt.Sprintf("{{M|Speak of %s.}}", o.Name), func() {
			g.orderSeen(o)
			g.orderMenu(a, o, "\""+o.Creed+"\"")
		}}}
	}
	if id := orderFlag(a, "ordm:"); id != "" {
		o := g.OrderByID(id)
		if o == nil {
			return nil
		}
		m := g.OrderMembership(o.ID)
		if m == nil || m.Rank < 1 {
			return nil
		}
		if !a.Has("signed") {
			a.Set("signed")
			g.Popup = Wrap(fmt.Sprintf("%s %s. \"Ah. %s of %s. The walls here have ears, but not for us.\"", a.Name, o.Sign, o.Ranks[m.Rank-1], o.Name), 100)
		}
		return []Option{{fmt.Sprintf("{{c|Give the sign of %s.}}", strings.TrimPrefix(o.Name, "The ")), func() {
			g.orderSeen(o)
			if g.Turn-m.LastCall >= 1440 || m.LastCall == 0 {
				m.LastCall = g.Turn
				tip := 5 * m.Rank
				g.P.Gold += tip
				g.talkMenu(fmt.Sprintf("They %s and lean close. \"%s\" (+%d %s)", o.Sign, g.gossip(a), tip, g.Sk.Currency))
				return
			}
			g.talkMenu("\"I told you all I know today, brother.\"")
		}}}
	}
	return nil
}

func (g *Game) orderSeen(o *Order) { g.orderState().Seen[o.ID] = true }

func orderKillHook(g *Game, d, src *Actor) {
	if src != nil && src != g.P && src.Faction != "player" {
		return
	}
	st := g.orderState()
	for id, m := range st.Mem {
		t := m.Task
		if t == nil || t.Kind != "kill" || t.Ready {
			continue
		}
		o := g.OrderByID(id)
		if o == nil {
			continue
		}
		hit := false
		if t.Target == "agent" {
			hit = d.Has("ordm:" + o.Rival)
		} else {
			hit = d.Faction == t.Target
		}
		if hit {
			g.orderTaskProgress(o, m, 1)
		}
	}
}

func orderEnterHook(g *Game, zn *Zone) {
	st := g.orderState()
	for _, o := range g.Orders() {
		if !o.Secret && zn.Kind == "town" && zn.Burg == o.Burg && zn.Z == 0 {
			st.Seen[o.ID] = true
		}
	}
	for id, m := range st.Mem {
		t := m.Task
		if t == nil || t.Ready {
			continue
		}
		o := g.OrderByID(id)
		if o == nil {
			continue
		}
		switch {
		case t.Kind == "relic" && !t.Spawned && zn.Key == t.Zone:
			if x, y, ok := zn.FreeSpot(g.R, nil); ok {
				t.Spawned = true
				zn.Items = append(zn.Items, &Item{ID: g.newID(), Kind: "trophy", Name: "relic of " + strings.TrimPrefix(o.Name, "The "), Glyph: '♦', F: CYellow, X: x, Y: y,
					Text: "order-relic:" + o.ID, Lore: "A sacred relic that " + o.Name + " lost long ago.", Value: 80, Weight: 1, Ident: true})
				g.Msg("{{Y|Something sacred glimmers in the dark...}}")
			}
		case t.Kind == "kill" && t.Target == "agent" && zn.Kind != "town" && t.Have+g.orderAgents(zn, o) < t.Need && g.R.Chance(0.6):
			rv := g.OrderByID(o.Rival)
			if rv == nil {
				continue
			}
			a := g.Spawn("bandit", maxi(zn.Level, 2), g.R)
			if x, y, ok := zn.FreeSpot(g.R, func(x, y int) bool { return abs(x-g.P.X)+abs(y-g.P.Y) > 6 }); ok {
				a.X, a.Y = x, y
				a.Name, a.Proper = "Agent of "+strings.TrimPrefix(rv.Name, "The "), true
				a.Faction, a.Angry = rv.RepKey(), true
				a.Set("ordm:" + rv.ID)
				zn.Actors = append(zn.Actors, a)
			}
		}
	}
}

func (g *Game) orderAgents(zn *Zone, o *Order) int {
	n := 0
	for _, a := range zn.Actors {
		if a.HP > 0 && a.Has("ordm:"+o.Rival) {
			n++
		}
	}
	return n
}

func orderTurnHook(g *Game) {
	if g.Cur == nil {
		return
	}
	st := g.orderState()
	if len(st.Mem) == 0 {
		return
	}
	near := false
	for _, d := range append(dirs8[:], [2]int{0, 0}) {
		if g.Cur.At(g.P.X+d[0], g.P.Y+d[1]) == TAltar {
			near = true
		}
	}
	for id, m := range st.Mem {
		t := m.Task
		if t == nil || t.Ready {
			continue
		}
		o := g.OrderByID(id)
		if o == nil {
			continue
		}
		switch t.Kind {
		case "pray":
			if near && (t.LastAct == 0 || g.Turn-t.LastAct >= 30) {
				t.LastAct = g.Turn
				g.Msg("{{C|You kneel and pray for the order.}}")
				g.orderTaskProgress(o, m, 1)
			}
		case "relic":
			if g.Turn%10 == 0 && g.hasRelic(o) != nil {
				t.Have = 1
				g.orderTaskDone(o, m)
			}
		}
	}
}

func orderSide(g *Game) []string {
	st := g.orderState()
	var ids []string
	for id, m := range st.Mem {
		if m.Task != nil {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	var out []string
	for _, id := range ids {
		o, m := g.OrderByID(id), st.Mem[id]
		if o == nil || len(out) >= 2 {
			continue
		}
		if m.Task.Ready {
			out = append(out, "{{G|"+shorten(o.Name, 22)+": report}}")
		} else {
			out = append(out, fmt.Sprintf("{{y|%s}} {{K|%d/%d}}", shorten(strings.TrimPrefix(o.Name, "The "), 20), m.Task.Have, m.Task.Need))
		}
	}
	return out
}

func shorten(s string, n int) string {
	if r := []rune(s); len(r) > n {
		return string(r[:n-1]) + "."
	}
	return s
}

// ---- leader dialogue ----

func (g *Game) orderMenu(a *Actor, o *Order, text string) {
	g.Popup = Wrap(text, 100)
	g.Opts = nil
	add := func(t string, f func()) { g.Opts = append(g.Opts, Option{t, f}) }
	m := g.OrderMembership(o.ID)
	cur := g.Sk.Currency
	back := func() { g.orderMenu(a, o, "\""+o.Creed+"\"") }
	switch {
	case m == nil:
		ok, why := g.orderMeets(o)
		add("{{G|Seek admission.}}", func() {
			if ok2, why2 := g.orderMeets(o); !ok2 {
				g.orderMenu(a, o, "\"Not yet. "+why2+"\"")
				return
			}
			g.orderJoin(o)
			if m2 := g.OrderMembership(o.ID); m2 != nil && m2.Task != nil {
				g.orderMenu(a, o, "\"Prove yourself first.\" "+m2.Task.Text)
			} else {
				g.orderMenu(a, o, fmt.Sprintf("\"Welcome, %s.\"", o.Ranks[0]))
			}
		})
		if o.Req.Kind == "rep" {
			add(fmt.Sprintf("Make an offering to the faith (50 %s, +40 reputation).", cur), func() {
				if g.P.Gold < 50 {
					g.orderMenu(a, o, "\"The faith is not moved by empty purses.\"")
					return
				}
				g.P.Gold -= 50
				g.AddRep(o.Req.Faction, 40, " for your offering")
				g.orderMenu(a, o, "\"Your gift is noted.\"")
			})
		}
		req := "an initiation rite"
		if o.Req.Kind != "task" {
			req = o.Req.Desc
		}
		add("What do you ask of those who join?", func() {
			s := "\"We ask " + req + "."
			if !ok {
				s += " " + why
			}
			if rv := g.OrderByID(o.Rival); rv != nil {
				s += " Know that we are enemies of " + rv.Name + "."
			}
			g.orderMenu(a, o, s+"\"")
		})
	case m.Task != nil:
		t := m.Task
		add("What is my task?", func() { g.orderMenu(a, o, fmt.Sprintf("\"%s\" (%d/%d)", t.Text, t.Have, t.Need)) })
		if t.Kind == "tithe" && !t.Ready {
			add(fmt.Sprintf("{{Y|Pay the tithe (%d %s).}}", t.Gold, cur), func() {
				if g.P.Gold < t.Gold {
					g.orderMenu(a, o, "\"Come back with the full sum.\"")
					return
				}
				g.P.Gold -= t.Gold
				t.Have, t.Ready = 1, true
				g.orderMenu(a, o, "\""+g.orderReport(o, m)+"\"")
			})
		}
		if t.Ready || t.Kind == "relic" {
			add("{{G|I have done what you asked.}}", func() { g.orderMenu(a, o, g.orderReport(o, m)) })
		}
		add("{{K|Abandon this task.}}", func() { m.Task = nil; g.AddRep(o.RepKey(), -10, " for failing the order"); back() })
	default:
		add("{{Y|Give me a task.}}", func() {
			m.Task = g.newOrderTask(o, m, false)
			g.orderMenu(a, o, "\""+m.Task.Text+"\"")
		})
	}
	if m != nil && m.Rank >= 1 {
		add("{{g|Tend my wounds.}}", func() {
			g.P.HP = g.P.MaxHP
			delete(g.P.Status, "bleeding")
			delete(g.P.Status, "poisoned")
			g.orderMenu(a, o, "\"Rest, "+o.Ranks[m.Rank-1]+". The order looks after its own.\"")
		})
	}
	if m != nil && m.Rank >= 2 {
		add("{{W|Draw my stipend.}}", func() {
			if m.LastPay != 0 && g.Turn-m.LastPay < 1440 {
				g.orderMenu(a, o, "\"You were paid today.\"")
				return
			}
			m.LastPay = g.Turn
			pay := 15*m.Rank + 2*g.P.Level
			g.P.Gold += pay
			g.orderMenu(a, o, fmt.Sprintf("\"For your service.\" (+%d %s)", pay, cur))
		})
		add("{{O|Requisition from the order's stores.}}", func() { g.orderStores(a, o, m) })
	}
	if m != nil && m.Rank >= 4 {
		add("{{G|Call a retainer to my side.}}", func() { g.orderRetainer(a, o, m) })
	}
	if m != nil {
		add("{{R|Renounce the order.}}", func() {
			delete(g.orderState().Mem, o.ID)
			g.AddRep(o.RepKey(), -100, " for oath-breaking")
			g.orderMenu(a, o, "\"Then you are dead to us.\"")
		})
	}
	add("Enough about that.", func() { g.talkMenu(g.greeting(a)) })
	g.Sel = 0
}

func (g *Game) orderStores(a *Actor, o *Order, m *OrderMember) {
	r := NewRng(fmt.Sprintf("%s:stores:%s:%d", g.W.Seed, o.ID, m.Rank))
	tier := clampi(m.Rank, 1, 4)
	stock := []*Item{g.MakeWeapon(Pick(r, []string{"longblade", "axe", "shortblade", "cudgel"}), tier, r), g.MakeArmor("body", tier, r), g.MakeArmor("head", tier, r), g.MakeTonic("heal", 1), g.MakeTonic("antidote", 1)}
	disc := g.orderDiscount(m)
	g.Popup = Wrap(fmt.Sprintf("\"The stores are open to you.\" (%d%% off)", disc), 100)
	g.Opts = nil
	for _, it := range stock {
		it := it
		price := maxi(1, it.Value*(100-disc)/100)
		g.Opts = append(g.Opts, Option{fmt.Sprintf("%s - %d %s", it.Name, price, g.Sk.Currency), func() {
			if g.P.Gold < price {
				g.orderMenu(a, o, "\"Not enough coin.\"")
				return
			}
			g.P.Gold -= price
			g.give(it)
			g.orderMenu(a, o, "\"Use it well.\"")
		}})
	}
	g.Opts = append(g.Opts, Option{"Enough.", func() { g.orderMenu(a, o, "\""+o.Creed+"\"") }})
	g.Sel = 0
}

func (g *Game) orderRetainer(a *Actor, o *Order, m *OrderMember) {
	n := 0
	for _, x := range g.Cur.Actors {
		if x.AI == "follower" {
			n++
		}
	}
	if n >= 3 {
		g.orderMenu(a, o, "\"You already travel with company enough.\"")
		return
	}
	bd := g.W.burgByID[o.Burg]
	if bd == nil {
		return
	}
	r := NewRng(fmt.Sprintf("%s:retainer:%s:%d", g.W.Seed, o.ID, g.Turn))
	f := g.makeNPC("guard", bd, g.W.cultByID[bd.Culture], r)
	f.Title = "retainer of " + strings.TrimPrefix(o.Name, "The ")
	f.AI, f.Faction, f.F = "follower", "player", CGreen
	f.X, f.Y = g.P.X, g.P.Y
	g.nudge(g.Cur, f)
	g.Cur.Actors = append(g.Cur.Actors, f)
	g.Msg(fmt.Sprintf("{{G|%s of the order joins you.}}", f.Name))
	g.orderMenu(a, o, "\"A good blade, sworn to our cause. Keep them alive.\"")
}

// ---- screen ----

func (g *Game) orderKnown() []*Order {
	st := g.orderState()
	var out []*Order
	for _, o := range g.Orders() {
		if st.Mem[o.ID] != nil || st.Seen[o.ID] {
			out = append(out, o)
			continue
		}
		if bd := g.W.burgByID[o.Burg]; bd != nil && !o.Secret && g.Known[bd.P] {
			out = append(out, o)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		mi, mj := st.Mem[out[i].ID] != nil, st.Mem[out[j].ID] != nil
		return mi && !mj
	})
	return out
}

func orderKeys(g *Game, k string) {
	known := g.orderKnown()
	switch k {
	case "Escape", "Enter", "O", "q":
		g.Mode = "main"
	case "ArrowUp", "N8", "k":
		g.Sel = maxi(0, g.Sel-1)
	case "ArrowDown", "N2", "j":
		g.Sel = mini(maxi(len(known)-1, 0), g.Sel+1)
	case "PageUp":
		g.Sel = maxi(0, g.Sel-10)
	case "PageDown":
		g.Sel = mini(maxi(len(known)-1, 0), g.Sel+10)
	}
}

func (g *Game) drawOrders() {
	s := g.sc
	x, y, w, h := g.panel("Orders, cults & guilds")
	known := g.orderKnown()
	st := g.orderState()
	if len(known) == 0 {
		s.TextB(x, y, "{{K|You know of no orders yet. Visit temples, academies and slums; ask around.}}", CGray, CBgDark, x+w)
		return
	}
	g.Sel = clampi(g.Sel, 0, len(known)-1)
	lw := 38
	rows := h - 4
	top := maxi(0, g.Sel-rows+1)
	for i := top; i < len(known) && i-top < rows; i++ {
		o := known[i]
		bg := CBgDark
		if i == g.Sel {
			bg = CBgSel
		}
		s.Fill(x, y+i-top, lw, 1, bg)
		mark := " "
		if m := st.Mem[o.ID]; m != nil {
			mark = "{{Y|*}}"
		}
		s.TextB(x, y+i-top, fmt.Sprintf("%s %s", mark, shorten(o.Name, lw-3)), CGray, bg, x+lw)
	}
	o := known[g.Sel]
	dx := x + lw + 2
	dw := w - lw - 2
	ly := y
	line := func(str string) {
		for _, l := range Wrap(str, dw) {
			if ly < y+h-3 {
				s.TextB(dx, ly, l, CGray, CBgDark, x+w)
				ly++
			}
		}
	}
	line("{{W|" + o.Name + "}}  {{K|(" + orderKindName(o.Kind) + ")}}")
	line("{{c|" + o.Creed + "}}")
	line(fmt.Sprintf("Seat: {{y|%s}} ({{K|%s}})", o.HomeName, districtNames[o.District]))
	if rv := g.OrderByID(o.Rival); rv != nil {
		line("Rival: {{r|" + rv.Name + "}}")
	}
	if o.Secret {
		line("{{K|A secret order: its members know a sign.}}")
	}
	req := o.Req.Desc
	if o.Req.Kind == "task" {
		req = "an initiation rite set by the leader"
	}
	line(fmt.Sprintf("Entry: {{y|%s}}   Leader: {{y|%s, %s}}", req, o.Leader, o.LeaderTit))
	ly++
	m := st.Mem[o.ID]
	rank := 0
	if m != nil {
		rank = m.Rank
	}
	for i := 0; i < 5; i++ {
		c := "K"
		if rank > i {
			c = "G"
		}
		perk := [5]string{"Healing, the sign", "Stipend, stores", "Power: " + g.Sk.PowerName(o.Power), "Retainers, discounts", "A unique relic of office"}[i]
		line(fmt.Sprintf("{{%s|%d. %s}} {{K|- %s}}", c, i+1, o.Ranks[i], perk))
	}
	ly++
	if m == nil {
		line("{{K|Not a member. Find the leader and ask for admission.}}")
	} else {
		line(fmt.Sprintf("{{G|You are a %s}}  tasks done: %d  reputation: %d", maxStr(rank, o.Ranks), m.Done, g.Rep[o.RepKey()]))
		if t := m.Task; t != nil {
			if t.Ready {
				line("{{G|Task fulfilled - report to " + o.Leader + ".}} " + t.Text)
			} else {
				line(fmt.Sprintf("{{y|Task: %s (%d/%d)}}", t.Text, t.Have, t.Need))
			}
		} else {
			line("{{K|No task: ask your leader.}}")
		}
	}
	s.TextB(x, y+h-3, "{{K|Up/Down: select   Esc: close}}", CGray, CBgDark, x+w)
}

func maxStr(rank int, ranks [5]string) string {
	if rank < 1 {
		return "candidate"
	}
	return ranks[rank-1]
}

func orderKindName(k string) string {
	return map[string]string{"holy": "militant order", "monastic": "monastic order", "cult": "heretic cult", "secret": "secret society", "thieves": "thieves' guild", "merchant": "merchant guild", "scholars": "scholars' society"}[k]
}
