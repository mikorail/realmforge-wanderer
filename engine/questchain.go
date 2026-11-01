package engine

import (
	"fmt"
	"sort"
	"strings"
)

// Main quest spine and village quests (prefix qc). The spine is built once from the world's real
// history when the home elder is first asked about the omens, then saved with its concrete targets
// in g.ext("spine"). Village quests are a second layer offered by elders and notables.

type qcChap struct {
	Kind   string `json:"k"` // visit talk boss relic slay depth return
	Title  string `json:"t"`
	Text   string `json:"x"`
	Done   string `json:"d"` // line shown when finished
	Burg   int    `json:"b,omitempty"`
	P      int    `json:"p"`
	Key    string `json:"key,omitempty"`
	ArtID  int    `json:"a,omitempty"`
	SimMon int    `json:"m,omitempty"`
	XP     int    `json:"xp"`
}

type qcVQ struct {
	ID       int    `json:"id"`
	Kind     string `json:"k"` // escort inscription peace rescue stolen
	Title    string `json:"t"`
	Text     string `json:"x"`
	GiverID  int    `json:"g"`
	Giver    string `json:"gn"`
	Faction  string `json:"f"`
	Burg     int    `json:"b"`
	Target   string `json:"tk,omitempty"` // zone key of the site
	DestBurg int    `json:"db,omitempty"`
	DestName string `json:"dn,omitempty"`
	A        int    `json:"a,omitempty"` // actor id (ward, captive, first feuding head)
	B        int    `json:"b2,omitempty"`
	SpokeA   bool   `json:"sa,omitempty"`
	SpokeB   bool   `json:"sb,omitempty"`
	Placed   bool   `json:"pl,omitempty"`
	Stage    int    `json:"s"` // 0 active, 1 ready to report, 2 done, 3 failed
	Reward   int    `json:"r"`
	Found    string `json:"fo,omitempty"`
}

type qcSpine struct {
	Started  bool            `json:"started"`
	Finished bool            `json:"finished"`
	Step     int             `json:"step"`
	Chaps    []qcChap        `json:"chaps"`
	Flags    map[string]bool `json:"flags,omitempty"`
	VQ       []*qcVQ         `json:"vq,omitempty"`
}

func (g *Game) qcSpine() *qcSpine {
	return g.ext("spine", func() any { return &qcSpine{} }).(*qcSpine)
}

func init() {
	talkHooks = append(talkHooks, qcTalk)
	killHooks = append(killHooks, qcKill)
	enterHooks = append(enterHooks, qcEnter)
	turnHooks = append(turnHooks, qcTurn)
	sideHooks = append(sideHooks, qcSide)
	mainKeys["Q"] = func(g *Game) { g.Mode = "spine"; g.Sel = 0 }
	modeKeys["spine"] = func(g *Game, k string) {
		switch k {
		case "Escape", "Enter", "Q", "q":
			g.Mode = "main"
		}
	}
	modeDraws["spine"] = func(g *Game) { g.qcDraw() }
}

// ---- helpers ----

func qcPD(w *WorldData, a, b int) int {
	return maxi(abs(a%w.W-b%w.W), abs(a/w.W-b/w.W))
}

func qcRuinKey(w *WorldData, p int) string { return ZoneKey(p%w.W, p/w.W, 1, 1, 0) }

func qcPerson(w *WorldData, id int) *PersonData {
	for i := range w.People {
		if w.People[i].ID == id {
			return &w.People[i]
		}
	}
	return nil
}

func qcAtP(g *Game, p int) bool {
	return g.Cur != nil && g.Cur.WX == p%g.W.W && g.Cur.WY == p/g.W.W
}

func qcHasTile(zn *Zone, t Terrain) bool {
	for _, c := range zn.T {
		if c == t {
			return true
		}
	}
	return false
}

func qcReadIn(g *Game, key string) bool {
	for k, v := range g.Read {
		if v && strings.HasPrefix(k, key) {
			return true
		}
	}
	return false
}

func qcInvArt(g *Game, id int) *Item {
	for _, it := range g.P.Inv {
		if it.SimArt == id {
			return it
		}
	}
	return nil
}

func qcFindAny(g *Game, id int) (*Actor, *Zone) {
	keys := make([]string, 0, len(g.Zones))
	for k := range g.Zones {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		z := g.Zones[k]
		for _, a := range z.Actors {
			if a.ID == id {
				return a, z
			}
		}
	}
	return nil, nil
}

// burgsNear: living surface towns (not `not`) between lo and hi parasangs of p, nearest first.
func qcBurgsNear(g *Game, p, not, lo, hi int) []*BurgData {
	w := g.W
	var out []*BurgData
	for i := range w.Burgs {
		b := &w.Burgs[i]
		if !b.Alive || b.Layer != "surface" || b.ID == not {
			continue
		}
		if d := qcPD(w, p, b.P); d >= lo && d <= hi {
			out = append(out, b)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		di, dj := qcPD(w, p, out[i].P), qcPD(w, p, out[j].P)
		if di != dj {
			return di < dj
		}
		return out[i].ID < out[j].ID
	})
	return out
}

// ruinsNear: dead surface towns within hi parasangs of p, nearest first.
func qcRuinsNear(g *Game, p, lo, hi int) []*BurgData {
	w := g.W
	var out []*BurgData
	for i := range w.Burgs {
		b := &w.Burgs[i]
		if b.Alive || b.Layer != "surface" || w.Burg[b.P] != b.ID {
			continue
		}
		if d := qcPD(w, p, b.P); d >= lo && d <= hi {
			out = append(out, b)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		di, dj := qcPD(w, p, out[i].P), qcPD(w, p, out[j].P)
		if di != dj {
			return di < dj
		}
		return out[i].ID < out[j].ID
	})
	return out
}

type qcSiteRef struct {
	P      int
	ZX, ZY int
	S      *Site
}

// sitesNear: surface dungeons within hi parasangs of p, nearest first.
func qcSitesNear(g *Game, p, lo, hi int, ok func(s *Site) bool) []qcSiteRef {
	w := g.W
	var out []qcSiteRef
	cx, cy := p%w.W, p/w.W
	for wy := maxi(0, cy-hi); wy <= mini(w.H-1, cy+hi); wy++ {
		for wx := maxi(0, cx-hi); wx <= mini(w.W-1, cx+hi); wx++ {
			q := w.Idx(wx, wy)
			if d := qcPD(w, p, q); d < lo || d > hi || w.IsWater(q) {
				continue
			}
			for zy := 0; zy < 3; zy++ {
				for zx := 0; zx < 3; zx++ {
					if s := g.siteAt(wx, wy, zx, zy); s != nil && ok(s) {
						out = append(out, qcSiteRef{q, zx, zy, s})
					}
				}
			}
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		di, dj := qcPD(w, p, out[i].P), qcPD(w, p, out[j].P)
		if di != dj {
			return di < dj
		}
		return out[i].P < out[j].P
	})
	return out
}

func qcDungeon(s *Site) bool {
	return s.Kind == "dungeon" && s.Theme != "trials" && s.Theme != "abyss"
}

// ---- building the spine ----

// qcCryptRelics mirrors genCrypt: the relics lying with the last eight dead rulers under a ruin.
func qcCryptRelics(g *Game, bd *BurgData) []*ArtifactData {
	w := g.W
	var dead []*PersonData
	for i := range w.People {
		pp := &w.People[i]
		if !pp.Alive && (pp.State == bd.State || pp.Burg == bd.ID) && (pp.Role == "ruler" || pp.Title != "") {
			dead = append(dead, pp)
		}
	}
	sort.Slice(dead, func(i, j int) bool { return dead[i].Died < dead[j].Died })
	if len(dead) > 8 {
		dead = dead[len(dead)-8:]
	}
	var out []*ArtifactData
	for _, pp := range dead {
		for i := range w.Artifacts {
			a := &w.Artifacts[i]
			if a.HolderType == "person" && a.HolderID == pp.ID && !g.Taken[a.ID] {
				out = append(out, a)
			}
		}
	}
	return out
}

func (g *Game) qcBuildSpine() []qcChap {
	w := g.W
	home := w.Idx(g.StartX, g.StartY)
	r := NewRng(w.Seed + ":qc:spine")
	hb := w.burgByID[g.StartBurg]
	homeName := "this town"
	if hb != nil {
		homeName = hb.Name
	}
	var ch []qcChap

	// 1. the ruin whose stone remembers
	ruins := qcRuinsNear(g, home, 1, 30)
	var ruin *BurgData
	if len(ruins) > 0 {
		ruin = ruins[0]
		line := "Its last words are lost."
		if evs := w.eventsAt[ruin.P]; len(evs) > 0 {
			e := w.Events[evs[len(evs)-1]]
			line = fmt.Sprintf("Year %d: \"%s\"", e.Y, e.Text)
		}
		ch = append(ch, qcChap{Kind: "visit", Title: "The Silent Stone", P: ruin.P, Key: qcRuinKey(w, ruin.P), XP: 150,
			Text: fmt.Sprintf("\"Before the troubles, a stone was raised in the ruins of %s, in %s. It is said to name what was lost. Go there and read it.\"", ruin.Name, w.RegionName(ruin.P)),
			Done: "The weathered letters give up their meaning. " + line})
	} else {
		// no ruins in this world: an old shrine in a living town
		var shrine *BurgData
		if bs := qcBurgsNear(g, home, g.StartBurg, 3, 25); len(bs) > 0 {
			shrine = bs[r.Int(0, mini(2, len(bs)-1))]
		}
		if shrine != nil {
			ch = append(ch, qcChap{Kind: "visit", Title: "The Old Shrine of " + shrine.Name, P: shrine.P, XP: 150,
				Text: fmt.Sprintf("\"Before the troubles, a shrine in %s, in %s, kept the account of what was lost. Go there and stand where the pilgrims stood.\"", shrine.Name, w.RegionName(shrine.P)),
				Done: "The shrine is small and old, and someone still keeps a lamp lit in it."})
		}
	}

	// the relic and the heir (used by the next two chapters)
	var relic *ArtifactData
	var crypt *BurgData
	var owner *PersonData
	for _, bd := range qcRuinsNear(g, home, 0, 60) {
		if rs := qcCryptRelics(g, bd); len(rs) > 0 {
			relic, crypt = rs[0], bd
			owner = qcPerson(w, relic.HolderID)
			break
		}
	}
	heir := "the last of the line"
	if owner != nil {
		for i := range w.People {
			pp := &w.People[i]
			if pp.Alive && pp.State == owner.State && pp.Born > owner.Born && pp.ID != owner.ID {
				heir = pp.Name
				break
			}
		}
	}

	// 2. the sage
	var sage *BurgData
	if bs := qcBurgsNear(g, home, g.StartBurg, 4, 22); len(bs) > 0 {
		sort.SliceStable(bs, func(i, j int) bool { return bs[i].Pop > bs[j].Pop })
		sage = bs[r.Int(0, mini(2, len(bs)-1))]
	} else if bs := qcBurgsNear(g, home, g.StartBurg, 1, 40); len(bs) > 0 {
		sage = bs[0]
	}
	if sage != nil {
		tale := "an old tale"
		if owner != nil && relic != nil {
			tale = fmt.Sprintf("%s, %s, whose heir %s vanished", owner.Name, nonEmpty(owner.Title, "a ruler"), heir)
		}
		ch = append(ch, qcChap{Kind: "talk", Title: "The Elder of " + sage.Name, Burg: sage.ID, P: sage.P, XP: 200,
			Text: fmt.Sprintf("\"The stone points to %s. Their elder keeps the old accounts and will know the rest. Ask them about %s.\"", sage.Name, tale),
			Done: "The elder lowers their voice and tells it plainly."})
	}

	// 3. a lair nearby
	lairName, lairP := "", 0
	from := home
	if sage != nil {
		from = sage.P
	}
	if ss := qcSitesNear(g, from, 0, 14, qcDungeon); len(ss) > 0 {
		s := ss[0]
		lairName, lairP = s.S.Name, s.P
		ch = append(ch, qcChap{Kind: "boss", Title: "Clear " + s.S.Name, P: s.P, XP: 400,
			Text: fmt.Sprintf("\"Whatever holds the old paths holds them from %s, in %s. Break its master and the road opens.\"", s.S.Name, w.RegionName(s.P)),
			Done: "The master of the place falls. The road is open."})
	}

	// 4. the relic in a crypt
	if relic != nil && crypt != nil {
		ch = append(ch, qcChap{Kind: "relic", Title: "The Lost " + relic.Name, P: crypt.P, ArtID: relic.ID, Key: ZoneKey(crypt.P%w.W, crypt.P/w.W, 1, 1, 1), XP: 600,
			Text: fmt.Sprintf("\"%s was buried with %s in the crypt beneath the ruins of %s, in %s. Bring it out.\"", nonEmpty(ownerName(owner), "The last ruler"), relic.Name, crypt.Name, w.RegionName(crypt.P)),
			Done: fmt.Sprintf("You hold %s. It is heavier than it looks.", relic.Name)})
	}

	if relic == nil {
		// no crypt in this world: the master of the lair carries a famous relic
		var best *ArtifactData
		for i := range w.Artifacts {
			x := &w.Artifacts[i]
			if !g.Taken[x.ID] && (best == nil || x.Fame > best.Fame || x.Fame == best.Fame && x.ID < best.ID) {
				best = x
			}
		}
		if best != nil && lairName != "" {
			ch = append(ch, qcChap{Kind: "relic", Title: "The Lost " + best.Name, ArtID: best.ID, P: lairP, XP: 600,
				Text: fmt.Sprintf("\"%s was carried off long ago. The master of %s wears it as a trophy. Take it from the hoard.\"", best.Name, lairName),
				Done: fmt.Sprintf("You hold %s. It is heavier than it looks.", best.Name)})
		}
	}

	// 5. a legend of the sim
	var mon *MonsterData
	for i := range w.Monsters {
		m := &w.Monsters[i]
		if m.Immortal || g.Slain[m.ID] || m.Slain {
			continue
		}
		if mon == nil || m.Power > mon.Power {
			mon = m
		}
	}
	if mon != nil {
		ch = append(ch, qcChap{Kind: "slay", Title: "Slay " + mon.Name, P: mon.P, SimMon: mon.ID, XP: 900,
			Text: fmt.Sprintf("\"Last, the one the chronicles fear: %s, which lairs at %s. The way to the spire runs through its death.\"", mon.Name, nonEmpty(mon.Lair, w.RegionName(mon.P))),
			Done: mon.Name + " is dead. The chroniclers will have to find new words."})
	}

	if mon == nil {
		if ss := qcSitesNear(g, from, 0, 26, qcDungeon); len(ss) > 0 {
			for _, s := range ss {
				if s.P == lairP && lairName != "" {
					continue
				}
				ch = append(ch, qcChap{Kind: "boss", Title: "Break the master of " + s.S.Name, P: s.P, XP: 900,
					Text: fmt.Sprintf("\"Last, the one the chronicles fear, who holds %s, in %s. The way to the spire runs through its death.\"", s.S.Name, w.RegionName(s.P)),
					Done: "The last of the old masters is dead."})
				break
			}
		}
	}

	// 6. the spire or the abyss
	var mega *MegaData
	for i := range w.Megas {
		m := &w.Megas[i]
		if mega == nil || m.Kind == "tower" {
			mega = m
		}
	}
	if mega != nil {
		verb, how := "Climb", "up"
		if mega.Kind == "abyss" {
			verb, how = "Descend into", "down"
		}
		ch = append(ch, qcChap{Kind: "depth", Title: verb + " " + mega.Name, P: mega.P, XP: 1200,
			Text: fmt.Sprintf("\"%s waits in %s. Go %s until you cannot be turned back, or until its keeper is dead.\"", mega.Name, w.RegionName(mega.P), how),
			Done: "You have gone deeper than most who tried."})
	}

	// 7. home
	ch = append(ch, qcChap{Kind: "return", Title: "Report to the elder of " + homeName, P: home, XP: 1500,
		Text: "\"Come back to me when it is done. I will want to hear all of it.\"",
		Done: "The elder listens to the end."})
	return ch
}

func ownerName(p *PersonData) string {
	if p == nil {
		return ""
	}
	return p.Name
}

// ---- progress ----

func (g *Game) qcCur() *qcChap {
	st := g.qcSpine()
	if !st.Started || st.Finished || st.Step < 0 || st.Step >= len(st.Chaps) {
		return nil
	}
	return &st.Chaps[st.Step]
}

// qcChapter is the sidebar line: the title of the current main-arc chapter, or "".
func (g *Game) qcChapter() string {
	if c := g.qcCur(); c != nil {
		return c.Title
	}
	return ""
}

func (g *Game) qcComplete(extra string) {
	st := g.qcSpine()
	c := g.qcCur()
	if c == nil {
		return
	}
	g.gainXP(c.XP)
	g.AddRep(fmt.Sprintf("state:%d", g.HomeState), 30, " for the old errand")
	g.Msg(fmt.Sprintf("{{G|Chapter %d of %d done: %s.}} {{c|+%d XP}}", st.Step+1, len(st.Chaps), c.Title, c.XP))
	if c.Done != "" {
		g.Msg("{{K|" + c.Done + "}}")
	}
	if c.Kind == "visit" {
		if it := g.MakeBook("Rubbing of the Silent Stone", "{{Y|Rubbing of the Silent Stone}}\n\n"+c.Done); it != nil {
			g.give(it)
		}
	}
	st.Step++
	if st.Step < len(st.Chaps) {
		n := st.Chaps[st.Step]
		g.Msg("{{W|Next: " + n.Title + "}} {{K|(Q: journal of the great errand)}}")
	}
	_ = extra
}

// qcPoll checks whether the current chapter's condition holds. Cheap; called from every hook.
func (g *Game) qcPoll() {
	c := g.qcCur()
	if c == nil || g.Cur == nil || g.Dead {
		return
	}
	switch c.Kind {
	case "visit":
		if c.Key == "" {
			if qcAtP(g, c.P) && g.Cur.Z == 0 {
				g.qcComplete("")
			}
		} else if g.Cur.Key == c.Key && (qcReadIn(g, c.Key) || !qcHasTile(g.Cur, TInscription)) {
			g.qcComplete("")
		}
	case "boss":
		if g.qcSpine().Flags[fmt.Sprintf("boss:%d", c.P)] {
			g.qcComplete("")
		}
	case "relic":
		if qcInvArt(g, c.ArtID) != nil {
			g.qcComplete("")
		}
	case "slay":
		if g.Slain[c.SimMon] {
			g.qcComplete("")
		}
	case "depth":
		if qcAtP(g, c.P) && (g.Cur.Z >= 3 || g.Cur.Z <= towerBase-2) {
			g.qcComplete("")
		}
	}
}

func qcKill(g *Game, d, src *Actor) {
	sp0 := g.qcSpine()
	if d != nil && d.Boss && g.Cur != nil && sp0.Started {
		if sp0.Flags == nil {
			sp0.Flags = map[string]bool{}
		}
		sp0.Flags[fmt.Sprintf("boss:%d", g.W.Idx(g.Cur.WX, g.Cur.WY))] = true
		for _, c := range sp0.Chaps {
			// no crypt in this world: the lair master's hoard holds the relic
			if c.Kind == "relic" && c.Key == "" && c.P == g.W.Idx(g.Cur.WX, g.Cur.WY) && !sp0.Flags["dropped"] {
				for i := range g.W.Artifacts {
					if a := &g.W.Artifacts[i]; a.ID == c.ArtID {
						sp0.Flags["dropped"] = true
						it := g.RelicItem(a)
						it.X, it.Y = d.X, d.Y
						g.Cur.Items = append(g.Cur.Items, it)
						g.Msg("{{Y|Among the hoard lies " + a.Name + ".}}")
					}
				}
			}
		}
	}
	if c := g.qcCur(); c != nil && d != nil {
		switch c.Kind {
		case "boss", "depth":
			if d.Boss && qcAtP(g, c.P) {
				g.qcComplete("")
			}
		case "slay":
			if d.SimMon == c.SimMon {
				g.qcComplete("")
			}
		}
	}
	g.qcPoll()
	sp := g.qcSpine()
	for _, q := range sp.VQ {
		if q.Stage == 0 && q.Kind == "peace" && (d.ID == q.A || d.ID == q.B) {
			q.Stage = 3
			g.Msg("{{R|Quest failed: " + q.Title + ".}}")
		}
		if q.Stage == 0 && q.Kind == "rescue" && d.ID == q.A {
			q.Stage = 3
			g.Msg("{{R|Quest failed: " + q.Title + ".}}")
		}
	}
}

func qcEnter(g *Game, zn *Zone) {
	g.qcPoll()
	sp := g.qcSpine()
	for _, q := range sp.VQ {
		if q.Stage == 0 && !q.Placed && q.Target == zn.Key {
			g.qcPlaceVQ(q, zn)
		}
	}
	g.qcVQPoll()
}

func qcTurn(g *Game) {
	g.qcPoll()
	g.qcVQPoll()
}

func qcSide(g *Game) []string {
	var out []string
	if t := g.qcChapter(); t != "" {
		out = append(out, "{{Y|Quest:}} "+shorten(t, 24))
	}
	for _, q := range g.qcSpine().VQ {
		if q.Stage <= 1 {
			tag := "{{y|Task:}} "
			if q.Stage == 1 {
				tag = "{{G|Report:}} "
			}
			out = append(out, tag+shorten(q.Title, 22))
			break
		}
	}
	return out
}

// ---- talk ----

func qcTalk(g *Game, a *Actor) []Option {
	var out []Option
	st := g.qcSpine()
	home := a.Role == "elder" && g.Cur.Burg == g.StartBurg && g.StartBurg > 0
	if home && !st.Started {
		out = append(out, Option{"{{W|Ask about the old omens.}}", func() { g.qcStart(a) }})
	}
	if c := g.qcCur(); c != nil {
		switch {
		case c.Kind == "talk" && a.Role == "elder" && g.Cur.Burg == c.Burg:
			out = append(out, Option{"{{W|Ask about the old tale.}}", func() {
				txt := c.Done
				g.qcComplete("")
				g.talkMenu("\"" + strings.Trim(txt, "\"") + "\"")
			}})
		case c.Kind == "return" && home:
			out = append(out, Option{"{{W|Report on the great errand.}}", func() { g.qcFinishMenu(a) }})
		case home:
			out = append(out, Option{"{{c|Remind me of the charge.}}", func() { g.talkMenu(c.Text) }})
		}
	}
	out = append(out, g.qcVQTalk(a)...)
	return out
}

func (g *Game) qcStart(a *Actor) {
	st := g.qcSpine()
	st.Chaps = g.qcBuildSpine()
	st.Started = true
	st.Step = 0
	st.Flags = map[string]bool{}
	g.deed("quest", fmt.Sprintf("The wanderer %s took up the elder's great errand.", g.P.Name), 0, 0)
	c := st.Chaps[0]
	g.Msg("{{W|The great errand begins: " + c.Title + ".}} {{K|(Q: journal)}}")
	g.talkMenu("\"I have waited for someone who asks. The old omens are not stories; they are an account that has not been paid.\"\n\n" + c.Text + "\n\n{{K|(The great errand is tracked in the sidebar and on Q.)}}")
	g.qcPoll()
}

func (g *Game) qcFinishMenu(a *Actor) {
	st := g.qcSpine()
	var relicID int
	for _, c := range st.Chaps {
		if c.Kind == "relic" {
			relicID = c.ArtID
		}
	}
	it := qcInvArt(g, relicID)
	text := "\"So. You went where the rest of us only talked. Tell me what to do with what you carry.\""
	if relicID == 0 || it == nil {
		text = "\"So. You went where the rest of us only talked. That is enough.\""
		g.talkMenu(text)
		g.Opts = []Option{{"{{G|It is done.}}", func() { g.qcFinish(a, "") }}}
		g.Sel = 0
		return
	}
	g.talkMenu(text)
	g.Opts = []Option{
		{"{{Y|Give the relic to the elder, to be returned to the people.}}", func() { g.qcFinish(a, "gave") }},
		{"{{O|Keep the relic. I earned it.}}", func() { g.qcFinish(a, "kept") }},
	}
	g.Sel = 0
}

func (g *Game) qcFinish(a *Actor, choice string) {
	st := g.qcSpine()
	c := g.qcCur()
	if c == nil {
		return
	}
	var line string
	switch choice {
	case "gave":
		for _, c2 := range st.Chaps {
			if c2.Kind == "relic" {
				if it := qcInvArt(g, c2.ArtID); it != nil {
					g.removeInv(it)
				}
			}
		}
		st.Flags["gave"] = true
		g.AddRep(fmt.Sprintf("state:%d", g.HomeState), 200, " for returning the relic")
		line = "The elder takes it in both hands. \"It will be kept where all can see it. You have given back a piece of who we were.\""
	case "kept":
		st.Flags["kept"] = true
		line = "The elder looks at it a long time. \"Then carry it well. May it be remembered that you did.\""
	default:
		line = "\"The account is paid. That is all I wanted.\""
	}
	g.qcComplete("")
	st.Finished = true
	r := NewRng(g.W.Seed + ":qc:reward")
	it := g.Loot(r, 5)
	if it != nil {
		it.Rarity, it.Ident = "unique", true
		it.Name = a.Name + "'s gift: " + it.Name
		g.give(it)
		line += fmt.Sprintf("\n\nThey press %s into your hands.", it.Display())
	}
	g.P.Gold += 300
	verdict := "kept the relic"
	if choice == "gave" {
		verdict = "returned the relic to the people"
	}
	g.deed("quest", fmt.Sprintf("The wanderer %s finished the great errand and %s.", g.P.Name, verdict), 0, 0)
	g.talkMenu(line)
}

// ---- village quests ----

func (g *Game) qcVQTalk(a *Actor) []Option {
	st := g.qcSpine()
	var out []Option
	given := false
	for _, q := range st.VQ {
		q := q
		if q.GiverID == a.ID {
			given = true
			if q.Stage == 1 {
				out = append(out, Option{"{{G|I have done what you asked. (" + shorten(q.Title, 30) + ")}}", func() { g.qcVQTurnIn(q, a) }})
			}
		}
		if q.Stage != 0 {
			continue
		}
		switch q.Kind {
		case "peace":
			if a.ID == q.A && !q.SpokeA || a.ID == q.B && !q.SpokeB {
				out = append(out, Option{"{{c|Speak for the peace between the houses.}}", func() { g.qcPeace(q, a) }})
			}
		case "rescue":
			if a.ID == q.A {
				out = append(out, Option{"{{G|Cut their bonds.}}", func() { g.qcFreeCaptive(q, a) }})
			}
		}
	}
	if (a.Role == "elder" || a.Role == "notable") && !given && !g.qcBusy(a) {
		out = append(out, Option{"Is there trouble here I could settle?", func() { g.qcOfferVQ(a) }})
	}
	return out
}

// qcBusy: only one open village task per town at a time.
func (g *Game) qcBusy(a *Actor) bool {
	for _, q := range g.qcSpine().VQ {
		if q.Stage <= 1 && q.Burg == g.Cur.Burg {
			return true
		}
	}
	return false
}

func (g *Game) qcOfferVQ(a *Actor) {
	st := g.qcSpine()
	r := NewRng(fmt.Sprintf("%s:qc:vq:%d", g.W.Seed, a.ID))
	kinds := []string{"escort", "inscription", "peace", "rescue", "stolen"}
	for i := len(kinds) - 1; i > 0; i-- {
		j := r.Int(0, i)
		kinds[i], kinds[j] = kinds[j], kinds[i]
	}
	var q *qcVQ
	for _, k := range kinds {
		if q = g.qcMakeVQ(k, a, r); q != nil {
			break
		}
	}
	if q == nil {
		g.talkMenu("\"Nothing I can name. The town is quiet, for once.\"")
		return
	}
	q.ID = g.newID()
	q.GiverID, q.Giver, q.Faction, q.Burg = a.ID, a.Name, a.Faction, g.Cur.Burg
	q.Reward = 70 + g.P.Level*20
	st.VQ = append(st.VQ, q)
	if z := g.Zones[q.Target]; z != nil && q.Target != "" {
		g.qcPlaceVQ(q, z)
	}
	g.Msg("{{W|New task: " + q.Title + "}}")
	g.talkMenu(q.Text + "\n\n{{K|(Task added; see Q.)}}")
}

func (g *Game) qcMakeVQ(kind string, a *Actor, r *Rng) *qcVQ {
	w := g.W
	here := w.Idx(g.Cur.WX, g.Cur.WY)
	bd := w.burgByID[g.Cur.Burg]
	town := "the town"
	if bd != nil {
		town = bd.Name
	}
	switch kind {
	case "escort":
		var cands []*Actor
		for _, o := range g.Cur.Actors {
			if o.Role == "villager" && o.AI != "follower" && o != a && !g.Hostile(o) && o.HP > 0 {
				cands = append(cands, o)
			}
		}
		bs := qcBurgsNear(g, here, g.Cur.Burg, 2, 14)
		if len(cands) == 0 || len(bs) == 0 {
			return nil
		}
		ward := Pick(r, cands)
		dest := bs[r.Int(0, mini(2, len(bs)-1))]
		g.joinCompanion(ward)
		ward.Set("qcward")
		return &qcVQ{Kind: "escort", Title: "Escort " + givenOf(ward.Name) + " to " + dest.Name, A: ward.ID, DestBurg: dest.ID, DestName: dest.Name,
			Text: fmt.Sprintf("\"%s must reach %s, in %s, and the roads are not safe. Take them there alive. They are waiting at your side.\"", ward.Name, dest.Name, w.RegionName(dest.P))}
	case "inscription":
		rs := qcRuinsNear(g, here, 1, 16)
		if len(rs) == 0 {
			return nil
		}
		rb := rs[r.Int(0, mini(2, len(rs)-1))]
		return &qcVQ{Kind: "inscription", Title: "Read the stone at " + rb.Name, Target: qcRuinKey(w, rb.P), DestName: rb.Name,
			Text: fmt.Sprintf("\"There is writing among the ruins of %s, in %s. Our scholars argue over what it says. Go and read it, and tell me.\"", rb.Name, w.RegionName(rb.P))}
	case "peace":
		byHouse := map[string]*Actor{}
		var hs []string
		for _, o := range g.Cur.Actors {
			if o.House != "" && o != a && o.AI != "follower" && !g.Hostile(o) && o.HP > 0 && byHouse[o.House] == nil {
				byHouse[o.House] = o
				hs = append(hs, o.House)
			}
		}
		sort.Strings(hs)
		if len(hs) < 2 {
			return nil
		}
		i := r.Int(0, len(hs)-1)
		j := (i + 1 + r.Int(0, len(hs)-2)) % len(hs)
		x, y := byHouse[hs[i]], byHouse[hs[j]]
		return &qcVQ{Kind: "peace", Title: fmt.Sprintf("Peace between House %s and House %s", hs[i], hs[j]), A: x.ID, B: y.ID,
			Text: fmt.Sprintf("\"House %s and House %s have not spoken since a boundary was moved by night. Blood is coming. Speak to %s and to %s, and see if the two can be brought to terms.\"", hs[i], hs[j], x.Name, y.Name)}
	case "rescue":
		ss := qcSitesNear(g, here, 2, 14, qcDungeon)
		if len(ss) == 0 {
			return nil
		}
		s := ss[r.Int(0, mini(2, len(ss)-1))]
		who := "a daughter"
		if bd != nil {
			who = Pick(r, []string{"a daughter", "a brother", "a cousin", "a husband", "an apprentice"})
		}
		return &qcVQ{Kind: "rescue", Title: "Rescue the missing from " + s.S.Name, Target: ZoneKey(s.P%w.W, s.P/w.W, s.ZX, s.ZY, 1), DestName: s.S.Name,
			Text: fmt.Sprintf("\"%s of %s went into %s, in %s, and did not come back. Someone saw them taken alive. Bring them out.\"", titleCase(who), town, s.S.Name, w.RegionName(s.P))}
	case "stolen":
		var tgt *qcSiteRef
		for _, o := range w.Outlaws {
			if ob := w.burgByID[o.Burg]; ob != nil && ob.Alive && qcPD(w, here, ob.P) <= 18 {
				zz := g.outlawZone(ob)
				if zz[0] < 0 {
					continue
				}
				if s := g.siteAt(ob.P%w.W, ob.P/w.W, zz[0], zz[1]); s != nil && s.Kind == "dungeon" {
					if tgt == nil || qcPD(w, here, ob.P) < qcPD(w, here, tgt.P) {
						tgt = &qcSiteRef{ob.P, zz[0], zz[1], s}
					}
				}
			}
		}
		if tgt == nil {
			if ss := qcSitesNear(g, here, 2, 14, qcDungeon); len(ss) > 0 {
				tgt = &ss[0]
			}
		}
		if tgt == nil {
			return nil
		}
		return &qcVQ{Kind: "stolen", Title: "Recover the stolen goods of " + town, Target: ZoneKey(tgt.P%w.W, tgt.P/w.W, tgt.ZX, tgt.ZY, 1), DestName: tgt.S.Name,
			Text: fmt.Sprintf("\"Thieves took the town's strongbox and ran for %s, in %s. The goods are below, on the first level. Bring them home.\"", tgt.S.Name, w.RegionName(tgt.P))}
	}
	return nil
}

// qcPlaceVQ puts the captive or the stolen goods (and their guards) into the target zone, once.
func (g *Game) qcPlaceVQ(q *qcVQ, zn *Zone) {
	if q.Placed || q.Stage != 0 {
		return
	}
	q.Placed = true
	r := NewRng(fmt.Sprintf("%s:qc:place:%d", g.W.Seed, q.ID))
	spot := func(near func(x, y int) bool) (int, int, bool) { return zn.FreeSpot(r, near) }
	switch q.Kind {
	case "rescue":
		bd := g.W.burgByID[q.Burg]
		if bd == nil {
			return
		}
		c := g.makeNPC("villager", bd, g.W.cultByID[bd.Culture], r)
		c.AI, c.Faction, c.Desc = "still", fmt.Sprintf("state:%d", bd.State), "Bound and gagged, but alive."
		c.Set("qccaptive")
		if x, y, ok := spot(nil); ok {
			c.X, c.Y = x, y
			zn.Actors = append(zn.Actors, c)
			q.A = c.ID
			for i := 0; i < 2; i++ {
				m := g.Spawn(pickCreature(r, zn.Biome, 'd', zn.Level), zn.Level+1, r)
				if mx, my, ok := spot(func(px, py int) bool { return abs(px-c.X) < 7 && abs(py-c.Y) < 5 }); ok {
					m.X, m.Y = mx, my
					zn.Actors = append(zn.Actors, m)
				}
			}
		}
	case "stolen":
		it := &Item{ID: g.newID(), Kind: "trophy", Name: "stolen strongbox of " + g.qcTownName(q.Burg), Glyph: '■', F: CYellow, Quest: q.ID, Weight: 6, Value: 100 + g.P.Level*20}
		if x, y, ok := spot(nil); ok {
			it.X, it.Y = x, y
			zn.Items = append(zn.Items, it)
			for i := 0; i < 2; i++ {
				m := g.Spawn(Pick(r, []string{"bandit", "archer"}), zn.Level+1, r)
				if mx, my, ok := spot(func(px, py int) bool { return abs(px-it.X) < 7 && abs(py-it.Y) < 5 }); ok {
					m.X, m.Y = mx, my
					m.Faction = "outlaws"
					zn.Actors = append(zn.Actors, m)
				}
			}
		}
	}
}

func (g *Game) qcTownName(id int) string {
	if bd := g.W.burgByID[id]; bd != nil {
		return bd.Name
	}
	return "the town"
}

func (g *Game) qcPeace(q *qcVQ, a *Actor) {
	roll := g.R.Int(1, 20) + g.P.M(EGO) + g.Skills["persuasion"]*2 + g.P.Level/4
	if roll < 12 {
		g.talkMenu("\"Words. The last man who came with words took a purse and went away. Try again when you have something that sounds like a promise.\"")
		return
	}
	if a.ID == q.A {
		q.SpokeA = true
	} else {
		q.SpokeB = true
	}
	if q.SpokeA && q.SpokeB {
		q.Stage = 1
		g.Msg("{{G|Both houses agree to meet. Report to " + q.Giver + ".}}")
	}
	g.talkMenu("\"...Fine. If they will sit and talk, so will we. But not before the other side asks.\"")
}

func (g *Game) qcFreeCaptive(q *qcVQ, a *Actor) {
	q.Stage = 1
	g.Cur.Actors = removeActor(g.Cur.Actors, a)
	g.Msg("{{G|You cut " + a.Name + " loose. They run for home. Report to " + q.Giver + ".}}")
	g.Mode = "main"
	g.Talk = nil
}

func removeActor(l []*Actor, a *Actor) []*Actor {
	out := l[:0]
	for _, o := range l {
		if o != a {
			out = append(out, o)
		}
	}
	return out
}

func (g *Game) qcVQTurnIn(q *qcVQ, a *Actor) {
	q.Stage = 2
	switch q.Kind {
	case "stolen":
		for _, it := range g.P.Inv {
			if it.Quest == q.ID {
				g.removeInv(it)
				break
			}
		}
	}
	g.qcPay(q)
	extra := ""
	if q.Found != "" {
		extra = "\n\n" + q.Found
	}
	g.talkMenu("\"You have done right by us.\" They count out " + fmt.Sprint(q.Reward) + " " + g.Sk.Currency + "." + extra)
}

func (g *Game) qcPay(q *qcVQ) {
	g.P.Gold += q.Reward
	g.gainXP(q.Reward)
	g.AddRep(q.Faction, 100, " for a village task")
	g.deed("quest", fmt.Sprintf("The wanderer %s finished a task for %s: %s.", g.P.Name, q.Giver, q.Title), 0, 0)
	g.Msg("{{G|Task done: " + q.Title + ".}}")
}

// qcVQPoll advances open village tasks: reading the stone, goods in the pack, the ward arriving.
func (g *Game) qcVQPoll() {
	if g.Cur == nil || g.Dead {
		return
	}
	for _, q := range g.qcSpine().VQ {
		if q.Stage != 0 {
			continue
		}
		switch q.Kind {
		case "inscription":
			if g.Cur.Key == q.Target && (qcReadIn(g, q.Target) || !qcHasTile(g.Cur, TInscription)) {
				q.Stage = 1
				q.Found = "You tell them what the stone said."
				for k, v := range g.Read {
					if v && strings.HasPrefix(k, q.Target) {
						var idx int
						fmt.Sscanf(strings.TrimPrefix(k, q.Target), "%d", &idx)
						if n := g.Cur.Notes[idx]; n != "" {
							q.Found = "You tell them what the stone said: " + Plain(n)
							break
						}
					}
				}
				g.Msg("{{G|You have read the stone. Report to " + q.Giver + ".}}")
			}
		case "stolen":
			for _, it := range g.P.Inv {
				if it.Quest == q.ID {
					q.Stage = 1
					g.Msg("{{G|You have the goods. Report to " + q.Giver + ".}}")
					break
				}
			}
		case "escort":
			ward, _ := qcFindAny(g, q.A)
			switch {
			case ward == nil || ward.HP <= 0:
				q.Stage = 3
				g.Msg("{{R|Quest failed: " + q.Title + ". Your charge is gone.}}")
			case ward.AI != "follower":
				q.Stage = 3
				g.Msg("{{R|Quest failed: " + q.Title + ". You parted ways.}}")
			case g.Cur.Burg == q.DestBurg && g.Cur.Burg > 0 && g.Cur.ActorAt(ward.X, ward.Y) == ward:
				c := ward.Comp
				ward.AI, ward.Comp, ward.Order = "town", nil, ""
				if c != nil {
					ward.Faction, ward.F = c.Faction, c.F
				}
				delete(ward.Flags, "qcward")
				q.Stage = 2
				g.qcPay(q)
				g.Msg("{{G|" + ward.Name + " thanks you and goes about their business.}}")
			}
		}
	}
}

// ---- journal screen ----

func (g *Game) qcDraw() {
	s := g.sc
	x, y, w, h := g.panel("The great errand & village tasks")
	st := g.qcSpine()
	ly := y
	put := func(str string) {
		for _, l := range Wrap(str, w) {
			if ly < y+h-2 {
				s.TextB(x, ly, l, CGray, CBgDark, x+w)
				ly++
			}
		}
	}
	if !st.Started {
		put("{{K|The elder of your home town speaks of old omens. Ask about them.}}")
	} else {
		for i, c := range st.Chaps {
			switch {
			case i < st.Step || st.Finished:
				put("{{G|[x] " + c.Title + "}}")
			case i == st.Step:
				put("{{Y|[>] " + c.Title + "}}")
				put("    {{c|" + c.Text + "}}")
			default:
				put("{{K|[ ] " + c.Title + "}}")
			}
		}
		if st.Finished {
			put("")
			if st.Flags["gave"] {
				put("{{G|You returned the relic to the people.}}")
			} else if st.Flags["kept"] {
				put("{{G|You kept the relic.}}")
			}
		}
	}
	put("")
	put("{{W|Village tasks}}")
	n := 0
	for _, q := range st.VQ {
		mark := "{{y|[ ]}}"
		switch q.Stage {
		case 1:
			mark = "{{G|[!]}}"
		case 2:
			mark = "{{G|[x]}}"
		case 3:
			mark = "{{R|[-]}}"
		}
		put(mark + " " + q.Title + " {{K|(" + q.Giver + ")}}")
		if q.Stage <= 1 {
			put("    {{K|" + q.Text + "}}")
		}
		n++
	}
	if n == 0 {
		put("{{K|None yet. Ask elders and notables whether there is trouble to settle.}}")
	}
	s.TextB(x, y+h-1, "{{K|[Esc] close}}", CGray, CBgDark, x+w)
}
