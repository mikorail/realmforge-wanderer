package engine

import (
	"fmt"
	"sort"
	"strings"
)

// The Codex: bestiary, atlas, library and people of the journal. State lives in g.ext("codex").

func init() {
	turnHooks = append(turnHooks, codexTurn)
	killHooks = append(killHooks, codexKill)
	enterHooks = append(enterHooks, codexEnter)
}

type cxBeast struct {
	Name    string   `json:"n"`
	Tpl     string   `json:"t,omitempty"`
	SimMon  int      `json:"sm,omitempty"`
	Seen    int      `json:"s"`
	Kills   int      `json:"k"`
	Place   string   `json:"pl"`
	MinLv   int      `json:"lo"`
	MaxLv   int      `json:"hi"`
	Desc    string   `json:"d,omitempty"`
	Lore    string   `json:"lr,omitempty"`
	AV      int      `json:"av"`
	DV      int      `json:"dv"`
	HP      int      `json:"hp"`
	Flags   []string `json:"fl,omitempty"`
	Order   int      `json:"o"`
	Generic string   `json:"g,omitempty"` // species/genre name
}

type cxPlace struct {
	Name      string   `json:"n"`
	Kind      string   `json:"k"`
	P         int      `json:"p"`
	Realm     string   `json:"r"`
	Deepest   int      `json:"dp"`
	MaxDepth  int      `json:"md,omitempty"`
	Districts []string `json:"ds,omitempty"`
	Marks     []string `json:"lm,omitempty"`
	Order     int      `json:"o"`
}

type cxBook struct {
	Title string `json:"t"`
	Kind  string `json:"k"`
	Text  string `json:"x"`
	Read  bool   `json:"r"`
	Order int    `json:"o"`
}

type cxPerson struct {
	Name  string `json:"n"`
	Title string `json:"t,omitempty"`
	Realm string `json:"r"`
	Caste string `json:"c,omitempty"`
	House string `json:"h,omitempty"`
	Role  string `json:"ro,omitempty"`
	Place string `json:"pl"`
	Note  string `json:"nt,omitempty"`
	Order int    `json:"o"`
}

type codexState struct {
	Beasts  map[string]*cxBeast  `json:"beasts"`
	Places  map[string]*cxPlace  `json:"places"`
	Books   map[string]*cxBook   `json:"books"`
	People  map[string]*cxPerson `json:"people"`
	Bonus   map[string]bool      `json:"bonus"`
	Counter int                  `json:"ctr"`
	seen    map[int]bool
}

func (g *Game) codex() *codexState {
	c := g.ext("codex", func() any { return &codexState{} }).(*codexState)
	if c.Beasts == nil {
		c.Beasts = map[string]*cxBeast{}
	}
	if c.Places == nil {
		c.Places = map[string]*cxPlace{}
	}
	if c.Books == nil {
		c.Books = map[string]*cxBook{}
	}
	if c.People == nil {
		c.People = map[string]*cxPerson{}
	}
	if c.Bonus == nil {
		c.Bonus = map[string]bool{}
	}
	if c.seen == nil {
		c.seen = map[int]bool{}
	}
	return c
}

func (c *codexState) next() int { c.Counter++; return c.Counter }

// ---- recording ----

func (g *Game) cxRegion() string {
	if g.Cur == nil {
		return "the wilds"
	}
	return g.W.RegionName(g.W.Idx(g.Cur.WX, g.Cur.WY))
}

func cxFlags(a *Actor) []string {
	var f []string
	for _, k := range []string{"fire", "frost", "poison", "undead"} {
		if a.Has(k) {
			f = append(f, k)
		}
	}
	return f
}

func (g *Game) cxBeastFor(a *Actor) *cxBeast {
	c := g.codex()
	key := a.Tpl
	if a.SimMon > 0 {
		key = fmt.Sprintf("sim:%d", a.SimMon)
	}
	if key == "" {
		return nil
	}
	b := c.Beasts[key]
	if b == nil {
		b = &cxBeast{Name: a.Name, Tpl: a.Tpl, SimMon: a.SimMon, Place: g.Cur.Name + ", " + g.cxRegion(), MinLv: a.Level, MaxLv: a.Level, Order: c.next()}
		if t := tplByID[a.Tpl]; t != nil && a.SimMon == 0 {
			b.Name = t.Name(g.Genre)
			b.Desc = t.Desc
		} else {
			b.Desc = a.Desc
		}
		if a.SimMon > 0 {
			for _, m := range g.W.Monsters {
				if m.ID == a.SimMon {
					b.Lore = fmt.Sprintf("%s %s (%s, power %d). %s", titleCase(m.Tier), m.Species, m.Cat, m.Power, m.Lore)
					if m.Trophy != "" {
						b.Lore += " Trophy: " + m.Trophy + "."
					}
					if m.Lair != "" {
						b.Lore += " Lair: " + m.Lair + "."
					}
				}
			}
		}
		c.Beasts[key] = b
	}
	b.MinLv = mini(b.MinLv, a.Level)
	if a.Level >= b.MaxLv {
		b.MaxLv = a.Level
		b.AV, b.DV, b.HP = a.AVNat, a.DVNat, a.MaxHP
		b.Flags = cxFlags(a)
	}
	return b
}

func codexTurn(g *Game) {
	if g.Cur == nil || g.P == nil || g.Dead {
		return
	}
	c := g.codex()
	for _, a := range g.Cur.Actors {
		if a == g.P || a.HP <= 0 {
			continue
		}
		if a.AI != "follower" && g.Hostile(a) && a.SimPer == 0 && g.visible(a.X, a.Y) {
			if b := g.cxBeastFor(a); b != nil && !c.seen[a.ID] {
				c.seen[a.ID] = true
				b.Seen++
			}
		}
		if (a.Talked || a.Kinship) && a.Name != "" && !g.Hostile(a) {
			g.cxPerson(a)
		}
	}
	g.cxScanBooks()
}

func codexKill(g *Game, d, src *Actor) {
	if d == nil || d == g.P || g.Cur == nil {
		return
	}
	if src != g.P && (src == nil || src.AI != "follower") {
		return
	}
	if d.SimPer > 0 || g.Hostile(d) == false && d.SimMon == 0 {
		return
	}
	c := g.codex()
	if b := g.cxBeastFor(d); b != nil {
		if !c.seen[d.ID] {
			c.seen[d.ID] = true
			b.Seen++
		}
		b.Kills++
	}
}

func (g *Game) cxPerson(a *Actor) {
	c := g.codex()
	realm := g.W.StateName(a.State)
	key := a.Name + "|" + realm
	if _, ok := c.People[key]; ok {
		return
	}
	p := &cxPerson{Name: a.Name, Title: a.Title, Realm: realm, Caste: a.Caste, House: a.House, Role: a.Role, Place: g.Cur.Name, Order: c.next()}
	if a.SimPer > 0 {
		for _, pd := range g.W.People {
			if pd.ID == a.SimPer {
				p.Note = fmt.Sprintf("%s, born Y%d; renown %d. %s", titleCase(pd.Role), pd.Born, pd.Renown, strings.Join(pd.Traits, ", "))
				if p.Title == "" {
					p.Title = pd.Title
				}
			}
		}
	}
	c.People[key] = p
}

func codexEnter(g *Game, zn *Zone) {
	if zn == nil {
		return
	}
	switch zn.Kind {
	case "town", "ruin", "lair", "wonder", "crypt", "dungeon", "tower", "camp":
	default:
		return
	}
	c := g.codex()
	p := g.W.Idx(zn.WX, zn.WY)
	name, key := zn.Name, zn.Kind+":"+zn.Name
	var site *Site
	if zn.Kind == "dungeon" || zn.Kind == "tower" {
		site = g.siteAt(zn.WX, zn.WY, zn.ZX, zn.ZY)
		if site != nil {
			name = site.Name
			key = zn.Kind + ":" + name
		}
	}
	var bd *BurgData
	if zn.Kind == "town" {
		bd = g.W.burgByID[zn.Burg]
		if bd != nil {
			name = bd.Name
			key = "town:" + bd.Name
		}
	}
	pl := c.Places[key]
	if pl == nil {
		pl = &cxPlace{Name: name, Kind: zn.Kind, P: p, Realm: g.W.StateName(g.W.State[p]), Order: c.next()}
		c.Places[key] = pl
		if site != nil {
			pl.MaxDepth = site.Depth
		}
		if bd != nil {
			pl.Realm = g.W.StateName(bd.State)
			var ds []string
			for k, v := range bd.Districts {
				if v > 0 {
					ds = append(ds, k)
				}
			}
			sort.Strings(ds)
			pl.Districts = ds
			pl.Marks = g.landmarkNames(bd)
		}
	}
	if bd != nil {
		if i := strings.Index(zn.Name, ", "); i >= 0 {
			d := zn.Name[i+2:]
			have := false
			for _, x := range pl.Districts {
				if x == d {
					have = true
				}
			}
			if !have {
				pl.Districts = append(pl.Districts, d)
			}
		}
	}
	if zn.Kind == "dungeon" && zn.Z > pl.Deepest {
		pl.Deepest = zn.Z
	}
	if zn.Kind == "tower" && towerBase-zn.Z > pl.Deepest {
		pl.Deepest = towerBase - zn.Z
	}
}

// ---- library ----

func cxBookKind(title string) string {
	switch {
	case strings.HasPrefix(title, "Annals of"):
		return "Annals"
	case strings.HasPrefix(title, "The Life of"):
		return "Biography"
	case strings.HasPrefix(title, "The Wars of"):
		return "War history"
	case strings.HasPrefix(title, "Beasts of"):
		return "Bestiary"
	case strings.HasPrefix(title, "Holy Writ of"):
		return "Sacred text"
	case strings.HasPrefix(title, "Travels in"):
		return "Travelogue"
	}
	return "Lore"
}

// cxScanBooks files every book in the pack into the library and notes which were read.
func (g *Game) cxScanBooks() {
	c := g.codex()
	for _, it := range g.P.Inv {
		if it.Kind != "book" || it.Text == "" {
			continue
		}
		b := c.Books[it.Name]
		if b == nil {
			if len(c.Books) >= 60 {
				continue
			}
			b = &cxBook{Title: it.Name, Kind: cxBookKind(it.Name), Text: it.Text, Order: c.next()}
			c.Books[it.Name] = b
		}
		if !b.Read && g.Read["book:"+it.Name] {
			b.Read = true
			if b.Kind != "Annals" && !c.Bonus[it.Name] {
				c.Bonus[it.Name] = true
				g.gainXP(5)
				g.Msg("{{c|Added to your library: " + it.Name + ". +5 XP}}")
			}
		}
	}
}

func (g *Game) cxP(p int) int {
	if p < 0 || p >= len(g.W.State) {
		return g.W.Idx(g.WX, g.WY)
	}
	return p
}

// RandomBook makes a procedural book for the region around parasang p.
// Hook it into loot, shelves (talk.go interact TShelf) and bookseller stock (faction.go).
func (g *Game) RandomBook(r *Rng, p int) *Item {
	return g.cxBookOfKind(r.Int(0, 5), r, p)
}

func (g *Game) cxBookOfKind(kind int, r *Rng, p int) *Item {
	p = g.cxP(p)
	w := g.W
	st := w.State[p]
	switch kind {
	case 1:
		if it := g.cxBiography(r, p, st); it != nil {
			return it
		}
	case 2:
		if it := g.cxWars(r, p, st); it != nil {
			return it
		}
	case 3:
		return g.cxBestiaryBook(r, p)
	case 4:
		if it := g.cxSacred(r, p); it != nil {
			return it
		}
	case 5:
		return g.cxTravelogue(r, p)
	}
	if len(w.States) == 0 {
		return g.MakeBook("Annals of the wilds", "{{Y|The Annals of the wilds}}\n\nNothing is recorded.")
	}
	if w.stateByID[st] == nil {
		st = w.States[r.Int(0, len(w.States)-1)].ID
	}
	return g.annalsBook(st)
}

func (g *Game) cxBiography(r *Rng, p, st int) *Item {
	w := g.W
	var cand []*PersonData
	for i := range w.People {
		if w.People[i].State == st && w.People[i].Name != "" {
			cand = append(cand, &w.People[i])
		}
	}
	if len(cand) < 3 {
		cand = cand[:0]
		for i := range w.People {
			if w.People[i].Name != "" {
				cand = append(cand, &w.People[i])
			}
		}
	}
	if len(cand) == 0 {
		return nil
	}
	sort.SliceStable(cand, func(i, j int) bool { return cand[i].Renown > cand[j].Renown })
	pd := cand[r.Int(0, mini(len(cand), 12)-1)]
	title := pd.Title
	if title == "" {
		title = pd.Role
	}
	realm := w.StateName(pd.State)
	lines := []string{fmt.Sprintf("{{Y|The Life of %s}}", pd.Name), "",
		fmt.Sprintf("%s %s of %s, born in the year %d.", titleCase(title), pd.Name, realm, pd.Born)}
	if bd := w.burgByID[pd.Burg]; bd != nil {
		lines = append(lines, "Their seat was "+bd.Name+".")
	}
	if sp := w.specByID[pd.Species]; sp != nil {
		lines = append(lines, fmt.Sprintf("Of the %s; martial %d, diplomacy %d, stewardship %d, intrigue %d.", sp.Name, pd.Martial, pd.Diplo, pd.Steward, pd.Intrigue))
	}
	if len(pd.Traits) > 0 {
		lines = append(lines, "Chroniclers call them "+strings.ToLower(strings.Join(pd.Traits, ", "))+".")
	}
	n := 0
	for _, e := range w.Events {
		if lmContainsWord(e.Text, pd.Name) && n < 10 {
			lines = append(lines, fmt.Sprintf("{{K|Y%d}} %s", e.Y, e.Text))
			n++
		}
	}
	if n == 0 {
		lines = append(lines, "The records say little of their deeds.")
	}
	if !pd.Alive {
		end := "died"
		if pd.Cause != "" {
			end += " (" + pd.Cause + ")"
		}
		lines = append(lines, fmt.Sprintf("They %s in the year %d.", end, pd.Died))
	}
	return g.MakeBook("The Life of "+pd.Name, strings.Join(lines, "\n"))
}

func (g *Game) cxWars(r *Rng, p, st int) *Item {
	w := g.W
	var local, all []EventData
	for _, e := range w.Events {
		if e.Kind == "war" || e.Kind == "peace" {
			all = append(all, e)
			if e.St == st {
				local = append(local, e)
			}
		}
	}
	src := local
	if len(src) < 2 {
		src = all
	}
	if len(src) == 0 {
		return nil
	}
	name := "the world"
	if len(local) >= 2 {
		name = w.StateName(st)
	}
	start := 0
	if len(src) > 10 {
		start = r.Int(0, len(src)-10)
	}
	lines := []string{fmt.Sprintf("{{Y|The Wars of %s}}", name), ""}
	for _, e := range src[start:mini(len(src), start+10)] {
		lines = append(lines, fmt.Sprintf("{{K|Y%d}} %s", e.Y, e.Text))
	}
	return g.MakeBook("The Wars of "+name+": a history", strings.Join(lines, "\n"))
}

func cxWeak(f []string) string {
	var w []string
	for _, k := range f {
		switch k {
		case "fire":
			w = append(w, "weak to frost")
		case "frost":
			w = append(w, "weak to fire")
		case "undead":
			w = append(w, "weak to fire, immune to poison")
		case "poison":
			w = append(w, "poisonous, immune to poison")
		}
	}
	return strings.Join(w, "; ")
}

func (g *Game) cxBestiaryBook(r *Rng, p int) *Item {
	w := g.W
	biome := w.Biome[p]
	var cand []*CTpl
	for i := range Templates {
		if Templates[i].Fits(biome, 's') && Templates[i].Lvl <= 8 {
			cand = append(cand, &Templates[i])
		}
	}
	if len(cand) < 3 {
		cand = cand[:0]
		for i := range Templates {
			cand = append(cand, &Templates[i])
		}
	}
	region := w.RegionName(p)
	lines := []string{fmt.Sprintf("{{Y|Beasts of %s}}", region), "{{K|A pamphlet of hunters' lore.}}", ""}
	used := map[string]bool{}
	for len(used) < 5 && len(used) < len(cand) {
		t := cand[r.Int(0, len(cand)-1)]
		if used[t.ID] {
			continue
		}
		used[t.ID] = true
		l := fmt.Sprintf("{{W|%s}} (level %d+): %s", titleCase(t.Name(g.Genre)), t.Lvl, t.Desc)
		var f []string
		for _, k := range []string{"fire", "frost", "poison", "undead"} {
			if strings.Contains(t.Flags, k) {
				f = append(f, k)
			}
		}
		if wk := cxWeak(f); wk != "" {
			l += " Hunters say it is " + wk + "."
		}
		lines = append(lines, l)
	}
	for _, m := range w.Monsters {
		if abs(m.P%w.W-p%w.W) <= 10 && abs(m.P/w.W-p/w.W) <= 10 && m.Lore != "" {
			lines = append(lines, fmt.Sprintf("{{R|%s}}: %s", m.Name, m.Lore))
			break
		}
	}
	return g.MakeBook("Beasts of "+region+": a pamphlet", strings.Join(lines, "\n"))
}

func (g *Game) cxSacred(r *Rng, p int) *Item {
	w := g.W
	var rel *ReligionData
	if w.Religion != nil && p < len(w.Religion) {
		if rr := w.relByID[w.Religion[p]]; rr != nil && rr.Deity != "" {
			rel = rr
		}
	}
	if rel == nil {
		var all []*ReligionData
		for i := range w.Religions {
			if w.Religions[i].Deity != "" {
				all = append(all, &w.Religions[i])
			}
		}
		if len(all) == 0 {
			return nil
		}
		rel = all[r.Int(0, len(all)-1)]
	}
	text := fmt.Sprintf("{{Y|Holy Writ of %s}}\n\n{{W|%s}}, %s\n\nTheology: %s\n\nDoctrine: %s", rel.Name, rel.Deity, strings.ToLower(rel.Type)+" faith", nonEmpty(rel.Theology, "It is not written."), nonEmpty(rel.Doctrine, "The faithful keep it by memory."))
	return g.MakeBook("Holy Writ of "+rel.Name, text)
}

func (g *Game) cxTravelogue(r *Rng, p int) *Item {
	w := g.W
	region := w.RegionName(p)
	px, py := p%w.W, p/w.W
	lines := []string{fmt.Sprintf("{{Y|Travels in %s}}", region), ""}
	author := "A wanderer"
	if len(w.People) > 0 {
		author = w.People[r.Int(0, len(w.People)-1)].Name
	}
	lines = append(lines, fmt.Sprintf("Set down by %s in the year %d.", author, w.Year), "")
	lines = append(lines, fmt.Sprintf("The country about is %s.", BiomeName(w.Biome[p])))
	n := 0
	for _, b := range w.Burgs {
		if b.Alive && abs(b.P%w.W-px) <= 8 && abs(b.P/w.W-py) <= 8 && n < 5 {
			lines = append(lines, fmt.Sprintf("%s, a %s of %s, lies %d parasangs hence.", b.Name, strings.ToLower(nonEmpty(b.Kind, "town")), w.StateName(b.State), maxi(abs(b.P%w.W-px), abs(b.P/w.W-py))))
			n++
		}
	}
	for _, wd := range w.Wonders {
		if abs(wd.P%w.W-px) <= 10 && abs(wd.P/w.W-py) <= 10 {
			lines = append(lines, fmt.Sprintf("Travellers speak of %s, a %s%s.", wd.Name, wd.Kind, map[bool]string{true: " now ruined", false: ""}[wd.Destroyed]))
			break
		}
	}
	if n == 0 {
		lines = append(lines, "I met no soul on the road.")
	}
	return g.MakeBook("Travels in "+region, strings.Join(lines, "\n"))
}

// ---- journal tabs ----

var codexTabNames = []string{"Quests", "Secrets", "Chronicle", "Bestiary", "Atlas", "Library", "People", "Cookbook", "Calendar"}

func (g *Game) keyJournal(k string) {
	n := len(codexTabNames)
	switch k {
	case "Escape", "Enter":
		g.Mode = "main"
		if g.OnWorld {
			g.Mode = "world"
		}
	case "Tab", "ArrowRight":
		g.Tab = (g.Tab + 1) % n
		g.Sel = 0
	case "ArrowLeft":
		g.Tab = (g.Tab + n - 1) % n
		g.Sel = 0
	case "ArrowUp", "PageUp":
		g.Sel = maxi(0, g.Sel-1-boolInt(k == "PageUp")*10)
	case "ArrowDown", "PageDown":
		g.Sel += 1 + boolInt(k == "PageDown")*10
	case "Home":
		g.Sel = 0
	}
}

func cxPad(s string, n int) string {
	r := []rune(s)
	if len(r) > n {
		return string(r[:n-1]) + "…"
	}
	return s + strings.Repeat(" ", n-len(r))
}

func (g *Game) cxDir(p int) string {
	dx, dy := p%g.W.W-g.WX, p/g.W.W-g.WY
	d := maxi(abs(dx), abs(dy))
	if d == 0 {
		return "here"
	}
	v, h := "", ""
	if dy < -abs(dx)/2 {
		v = "N"
	} else if dy > abs(dx)/2 {
		v = "S"
	}
	if dx < -abs(dy)/2 {
		h = "W"
	} else if dx > abs(dy)/2 {
		h = "E"
	}
	return fmt.Sprintf("%d ps %s%s", d, v, h)
}

// codexLines builds the body of journal tab 3.. (Bestiary, Atlas, Library, People).
func (g *Game) codexLines(tab, w int) []string {
	g.cxScanBooks()
	c := g.codex()
	var lines []string
	switch tab {
	case 3:
		var bs []*cxBeast
		for _, b := range c.Beasts {
			bs = append(bs, b)
		}
		sort.Slice(bs, func(i, j int) bool { return bs[i].Order < bs[j].Order })
		lines = append(lines, fmt.Sprintf("{{K|%d creatures recorded. Slay three of a kind to learn its anatomy.}}", len(bs)), "")
		for _, b := range bs {
			head := fmt.Sprintf("{{Y|%s}} {{K|seen %d · slain %d · level %s}}", titleCase(b.Name), b.Seen, b.Kills, cxRange(b.MinLv, b.MaxLv))
			if b.SimMon > 0 {
				head = "{{R|" + titleCase(b.Name) + "}} {{O|legend}} {{K|" + fmt.Sprintf("seen %d · slain %d", b.Seen, b.Kills) + "}}"
			}
			lines = append(lines, head)
			lines = append(lines, Wrap("  {{K|first seen:}} "+b.Place, w)...)
			if b.Lore != "" {
				lines = append(lines, Wrap("  "+b.Lore, w)...)
			}
			if b.Desc != "" {
				lines = append(lines, Wrap("  "+b.Desc, w)...)
			}
			if b.Kills >= 3 {
				st := fmt.Sprintf("  {{C|◆}}AV %d  {{c|DV}} %d  {{R|♥}}HP %d", b.AV, b.DV, b.HP)
				if wk := cxWeak(b.Flags); wk != "" {
					st += "  {{G|" + wk + "}}"
				}
				lines = append(lines, st)
			} else {
				lines = append(lines, fmt.Sprintf("  {{K|anatomy unknown (%d more kills)}}", 3-b.Kills))
			}
			lines = append(lines, "")
		}
		if len(bs) == 0 {
			lines = []string{"{{K|You have not yet recorded any creatures.}}"}
		}
	case 4:
		var ps []*cxPlace
		for _, p := range c.Places {
			ps = append(ps, p)
		}
		sort.Slice(ps, func(i, j int) bool { return ps[i].Order < ps[j].Order })
		lines = append(lines, "{{K}}"+"{{K|"+cxPad("Place", 36)+cxPad("Kind", 9)+cxPad("Parasang", 11)+cxPad("Realm", 26)+"Distance}}", "")
		for _, p := range ps {
			x, y := p.P%g.W.W, p.P/g.W.W
			lines = append(lines, fmt.Sprintf("{{Y|%s}}{{c|%s}}%s{{K|%s}}{{W|%s}}", cxPad(p.Name, 36), cxPad(p.Kind, 9), cxPad(fmt.Sprintf("(%d,%d)", x, y), 11), cxPad(p.Realm, 26), g.cxDir(p.P)))
			switch p.Kind {
			case "dungeon":
				lines = append(lines, fmt.Sprintf("  {{K|deepest level reached: %d of %d}}", p.Deepest, p.MaxDepth))
			case "tower":
				lines = append(lines, fmt.Sprintf("  {{K|highest floor reached: %d of %d}}", p.Deepest, p.MaxDepth))
			}
			if len(p.Districts) > 0 {
				lines = append(lines, Wrap("  {{K|districts:}} "+strings.Join(p.Districts, ", "), w)...)
			}
			if len(p.Marks) > 0 {
				lines = append(lines, Wrap("  {{K|landmarks:}} "+strings.Join(p.Marks, ", "), w)...)
			}
		}
		if len(ps) == 0 {
			lines = []string{"{{K|No places charted yet.}}"}
		}
	case 5:
		var bs []*cxBook
		for _, b := range c.Books {
			bs = append(bs, b)
		}
		sort.Slice(bs, func(i, j int) bool { return bs[i].Order > bs[j].Order })
		for _, b := range bs {
			st := "{{K|unread}}"
			if b.Read {
				st = "{{G|read}}"
			}
			lines = append(lines, fmt.Sprintf("{{Y|%s}} {{c|[%s]}} %s", b.Title, b.Kind, st))
			if b.Read {
				lines = append(lines, Wrap(b.Text, w-2)...)
			}
			lines = append(lines, "")
		}
		if len(bs) == 0 {
			lines = []string{"{{K|You carry no books yet. Shelves and booksellers have them.}}"}
		}
	case 6:
		var ps []*cxPerson
		for _, p := range c.People {
			ps = append(ps, p)
		}
		sort.Slice(ps, func(i, j int) bool { return ps[i].Order > ps[j].Order })
		for _, p := range ps {
			head := "{{Y|" + p.Name + "}}"
			if p.Title != "" {
				head += ", " + p.Title
			}
			lines = append(lines, head+" {{K|of "+p.Realm+"}}")
			var bits []string
			if p.Caste != "" {
				bits = append(bits, "caste "+p.Caste)
			}
			if p.House != "" {
				bits = append(bits, "house "+p.House)
			}
			bits = append(bits, "met at "+p.Place)
			lines = append(lines, Wrap("  {{K|"+strings.Join(bits, " · ")+"}}", w)...)
			if p.Note != "" {
				lines = append(lines, Wrap("  "+p.Note, w)...)
			}
			lines = append(lines, "")
		}
		if len(ps) == 0 {
			lines = []string{"{{K|You have not yet spoken with anyone of note.}}"}
		}
	}
	return lines
}

func cxRange(a, b int) string {
	if a == b {
		return fmt.Sprint(a)
	}
	return fmt.Sprintf("%d-%d", a, b)
}
