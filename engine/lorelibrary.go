package engine

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// Lore library: readable in-world books and scrolls built from real WorldData.Events,
// with a collection log screen (prefix ll). Reuses literature.go (GenBook, glSt) and
// codex.go (codex, cxScanBooks) rather than duplicating them.
//
// State lives in g.ext("ll") (planet-local). All generation uses side rngs (seed:ll:...).

const llVis = MH - 5

func init() {
	unLocalKeys = append(unLocalKeys, "ll")
	zoneHooks = append(zoneHooks, llZone)
	turnHooks = append(turnHooks, llTurn)
	talkHooks = append(talkHooks, llTalk)
	modeKeys["ll"] = llKeys
	modeDraws["ll"] = func(g *Game) { g.llDraw() }
	paEntries = append(paEntries, paEntry{
		Name:     "Lore library",
		Keywords: "lore library books scrolls reading collection archive chronicle codex",
		Cat:      "Screen",
		Label:    "screen",
		Func:     func(g *Game) { g.llOpen() },
	})
}

type llEntry struct {
	Key      string `json:"k"`
	Title    string `json:"t"`
	Category string `json:"c"` // Scroll, Literature, Annals, Biography, War history, Bestiary, Sacred text, Travelogue, Lore
	Author   string `json:"a,omitempty"`
	Text     string `json:"x"`
	EvIdx    int    `json:"e,omitempty"` // WorldData.Events index + 1 (0 if none)
	Year     int    `json:"y,omitempty"`
	Read     bool   `json:"r,omitempty"`
	Order    int    `json:"o"`
}

type llState struct {
	Entries    map[string]*llEntry `json:"en,omitempty"`
	EventsRead map[int]bool        `json:"ev,omitempty"`
	Milestones map[int]bool        `json:"ms,omitempty"`
	Bought     map[string]bool     `json:"bo,omitempty"`
	Counter    int                 `json:"ct,omitempty"`

	catIdx      int
	sel         int
	reading     string
	scroll      int
	lines       []string
	lastSyncSig uint64
	synced      bool
}

func (g *Game) ll() *llState {
	mk := func() any {
		return &llState{
			Entries:    map[string]*llEntry{},
			EventsRead: map[int]bool{},
			Milestones: map[int]bool{},
			Bought:     map[string]bool{},
		}
	}
	s := g.ext("ll", mk).(*llState)
	if s.Entries == nil {
		s.Entries = map[string]*llEntry{}
	}
	if s.EventsRead == nil {
		s.EventsRead = map[int]bool{}
	}
	if s.Milestones == nil {
		s.Milestones = map[int]bool{}
	}
	if s.Bought == nil {
		s.Bought = map[string]bool{}
	}
	return s
}

var llCategories = []string{"All", "Scroll", "Literature", "Annals", "Biography", "War history", "Sacred text", "Travelogue", "Bestiary", "Lore"}

// ---- event scrolls built from WorldData.Events ----

var llScrollForms = map[string][]string{
	"war":          {"Campaign Dispatch", "Field Roll", "Siege Account"},
	"peace":        {"Treaty Copy", "Accord Scroll", "Truce Charter"},
	"fall":         {"Lamentation Scroll", "Last Witness Leaf", "Chronicle of Collapse"},
	"coup":         {"Palace Broadsheet", "Usurpation Record", "Sealed Deposition"},
	"death":        {"Funeral Oration", "Court Obituary", "Mourning Leaf"},
	"monster":      {"Beast-Warden Report", "Hunter's Tally", "Warning Scroll"},
	"disaster":     {"Calamity Ledger", "Relief Roll", "Eyewitness Scroll"},
	"eco_disaster": {"Blight Survey", "Warden's Log", "Fallow Roll"},
	"faith":        {"Synod Epistle", "Heresy Tract", "Temple Scroll"},
	"god":          {"Miracle Account", "Oracle Leaf", "Prophet's Scroll"},
	"wonder":       {"Builder's Charter", "Monument Roll", "Surveyor's Leaf"},
	"uprising":     {"Rebel Manifesto", "Garrison Report", "Street Broadsheet"},
}

func (g *Game) llMakeScroll(r *Rng, evIdx int) *Item {
	w := g.W
	if w == nil || evIdx < 0 || evIdx >= len(w.Events) {
		return nil
	}
	e := w.Events[evIdx]
	forms := llScrollForms[e.Kind]
	if len(forms) == 0 {
		forms = []string{"Chronicle Scroll", "Archive Leaf", "Dated Account"}
	}
	form := Pick(r, forms)
	realm := w.StateName(e.St)
	place := realm
	if e.P >= 0 && e.P < len(w.Biome) {
		place = w.RegionName(e.P)
	}
	title := fmt.Sprintf("%s of %s (Y%d)", form, place, e.Y)

	author := "an unnamed clerk"
	if e.St > 0 {
		if st := w.stateByID[e.St]; st != nil {
			if cu := w.cultByID[st.Culture]; cu != nil {
				author = g.cultureName(cu, r)
			}
		}
	}

	var lines []string
	lines = append(lines, "{{Y|"+title+"}}")
	lines = append(lines, fmt.Sprintf("{{K|Set down by %s, touching %s in the year %d.}}", author, realm, e.Y), "")
	lines = append(lines, e.Text, "")
	if e.P >= 0 && e.P < len(w.Biome) {
		lines = append(lines, fmt.Sprintf("Region noted: %s (%s).", place, BiomeName(w.Biome[e.P])))
	}
	if e.St > 0 {
		if st := w.stateByID[e.St]; st != nil && st.Ruler != "" {
			lines = append(lines, fmt.Sprintf("Realm record: %s (%s).", st.Name, nonEmpty(st.Form, "realm")))
		}
	}
	lines = append(lines, fmt.Sprintf("{{K|Catalogued as event #%d of the world chronicle.}}", evIdx+1))

	it := g.MakeBook(title, strings.Join(lines, "\n"))
	it.Glyph = '?'
	it.F = CYellow
	it.Value = 16 + r.Int(0, 12)
	it.Lore = fmt.Sprintf("A %s from year %d.", strings.ToLower(form), e.Y)
	it.Extra = map[string]int{"ll": 1, "ll:ev": evIdx + 1}
	return it
}

func (g *Game) llPickEvent(r *Rng, p int) int {
	w := g.W
	if w == nil || len(w.Events) == 0 {
		return -1
	}
	st := 0
	if p >= 0 && p < len(w.State) {
		st = w.State[p]
	}
	var local, all []int
	for i := range w.Events {
		e := &w.Events[i]
		if strings.TrimSpace(e.Text) == "" {
			continue
		}
		all = append(all, i)
		if (st > 0 && e.St == st) || (e.P >= 0 && p >= 0 && glDist(w, e.P, p) <= 18) {
			local = append(local, i)
		}
	}
	if len(local) > 0 && r.Chance(0.75) {
		return local[r.Int(0, len(local)-1)]
	}
	if len(all) == 0 {
		return -1
	}
	return all[r.Int(0, len(all)-1)]
}

// ---- zone placement ----

func llZone(g *Game, zn *Zone, _ *Rng) {
	if g.W == nil || zn == nil || len(g.W.Events) == 0 || zn.Underwater {
		return
	}
	eligible := false
	switch zn.Kind {
	case "ruin", "crypt", "wonder":
		eligible = true
	case "town":
		nm := strings.ToLower(zn.Name)
		if strings.Contains(nm, "academy") || strings.Contains(nm, "temple") || strings.Contains(nm, "palace") || strings.Contains(nm, "plaza") {
			eligible = true
		}
	}
	if !eligible {
		return
	}
	if HashStr(g.W.Seed+":ll:z:"+zn.Key)%100 >= 55 {
		return
	}
	r := NewRng(g.W.Seed + ":ll:zone:" + zn.Key)
	p := g.W.Idx(zn.WX, zn.WY)

	x, y, ok := zn.FreeSpot(r, func(x, y int) bool {
		t := zn.At(x, y)
		return (t == TFloor || t == TCobble || t == TDirt || t == TGrass) && x > 2 && x < MW-3 && y > 1 && y < MH-2
	})
	if !ok {
		return
	}
	var it *Item
	if r.Chance(0.6) {
		if ev := g.llPickEvent(r, p); ev >= 0 {
			it = g.llMakeScroll(r, ev)
		}
	} else {
		// Reuse literature.go's GenBook.
		it = g.GenBook(r, p)
	}
	if it != nil {
		it.X, it.Y = x, y
		zn.Items = append(zn.Items, it)
	}
}

// ---- syncing with inventory, codex.go and literature.go ----

func (g *Game) llSync() {
	if g.W == nil || g.P == nil {
		return
	}
	st := g.ll()
	g.cxScanBooks()

	addEntry := func(key, title, cat, author, text string, evIdx, year int, read bool) {
		if title == "" || text == "" {
			return
		}
		e := st.Entries[key]
		if e == nil {
			if len(st.Entries) >= 120 {
				return
			}
			st.Counter++
			e = &llEntry{
				Key:      key,
				Title:    title,
				Category: cat,
				Author:   author,
				Text:     text,
				EvIdx:    evIdx,
				Year:     year,
				Order:    st.Counter,
			}
			st.Entries[key] = e
		}
		if read && !e.Read {
			g.llMarkRead(e)
		}
	}

	// 1. Scan player pack for books and scrolls.
	for _, it := range g.P.Inv {
		if it == nil || it.Kind != "book" || it.Text == "" {
			continue
		}
		cat := cxBookKind(it.Name)
		evIdx := 0
		year := 0
		if it.Extra != nil {
			if it.Extra["ll"] > 0 {
				cat = "Scroll"
				evIdx = it.Extra["ll:ev"]
			} else if it.Extra["gl"] > 0 {
				cat = "Literature"
				evIdx = it.Extra["gl:ev"]
			}
		}
		if evIdx > 0 && evIdx <= len(g.W.Events) {
			year = g.W.Events[evIdx-1].Y
		}
		key := "b:" + it.Name
		addEntry(key, it.Name, cat, it.Lore, it.Text, evIdx, year, g.Read["book:"+it.Name])
	}

	// 2. Reuse codex.go library entries.
	for _, cb := range g.codex().Books {
		if cb == nil {
			continue
		}
		addEntry("b:"+cb.Title, cb.Title, cb.Kind, "", cb.Text, 0, 0, cb.Read)
	}

	// 3. Reuse literature.go shelf entries.
	for _, sr := range g.glSt().Shelf {
		key := "b:" + sr.Title
		if e := st.Entries[key]; e != nil {
			if e.Author == "" {
				e.Author = sr.Author
			}
			if !e.Read {
				g.llMarkRead(e)
			}
		} else {
			summary := fmt.Sprintf("{{Y|%s}}\n\nA %s by %s (leaning %s), recorded on your reading shelf.", sr.Title, sr.Form, sr.Author, sr.Bias)
			addEntry(key, sr.Title, "Literature", sr.Author, summary, 0, 0, true)
		}
	}
}

func (g *Game) llMarkRead(e *llEntry) {
	if e == nil || e.Read {
		return
	}
	st := g.ll()
	e.Read = true
	if g.Read != nil {
		g.Read["book:"+e.Title] = true
	}
	if e.EvIdx > 0 && e.EvIdx <= len(g.W.Events) {
		ev := g.W.Events[e.EvIdx-1]
		if !st.EventsRead[e.EvIdx-1] {
			st.EventsRead[e.EvIdx-1] = true
			g.addSecret(fmt.Sprintf("Year %d: %s (from %s)", ev.Y, ev.Text, e.Title), "history", ev.P, ev.Y)
		}
	}
	g.gainXP(6 + g.Skills["lore"]*2)

	// Check collection milestones.
	readCount := 0
	for _, en := range st.Entries {
		if en.Read {
			readCount++
		}
	}
	for _, ms := range []int{5, 12, 25} {
		if readCount >= ms && !st.Milestones[ms] {
			st.Milestones[ms] = true
			g.gainXP(ms * 4)
			g.deed("lore", fmt.Sprintf("%s catalogued %d works in their lore library.", g.P.Name, ms), 0, 0)
			if ms == 12 {
				g.Skills["lore"]++
				g.Msg("{{G|Library milestone (12 works): your Lore skill rises by 1!}}")
			} else {
				g.Msg(fmt.Sprintf("{{G|Library milestone: %d works read and catalogued! +%d XP}}", ms, ms*4))
			}
		}
	}
}

func llTurn(g *Game) {
	if g.W == nil || g.P == nil || g.Dead || g.Turn%15 != 0 {
		return
	}
	st := g.ll()
	var sig uint64 = uint64(len(g.Read))*1315423911 ^ uint64(len(g.codex().Books))*2654435761 ^ uint64(len(g.glSt().Shelf))*40503
	for _, it := range g.P.Inv {
		if it != nil && it.Kind == "book" {
			sig = sig*31 + uint64(it.ID) + 1
		}
	}
	if st.synced && sig == st.lastSyncSig {
		return
	}
	st.synced = true
	st.lastSyncSig = sig
	g.llSync()
}

// ---- talk hook for scholars, chroniclers, booksellers ----

func llTalk(g *Game, a *Actor) []Option {
	if g.W == nil || g.P == nil || g.Cur == nil || a == nil || a.HP <= 0 || a.AI == "follower" || g.Hostile(a) {
		return nil
	}
	switch a.Role {
	case "bookseller", "scholar", "chronicler", "archivist":
	default:
		return nil
	}
	return []Option{
		{
			Text: "Ask for a local chronicle scroll or book for your library.",
			Do:   func() { g.llOfferScroll(a) },
		},
		{
			Text: "Open your lore library and collection log.",
			Do:   func() { g.llOpen() },
		},
	}
}

func (g *Game) llOfferScroll(a *Actor) {
	st := g.ll()
	p := g.W.Idx(g.Cur.WX, g.Cur.WY)
	slot := g.Day() / 10
	key := fmt.Sprintf("%d:%d:%d", a.ID, p, slot)
	if st.Bought[key] {
		g.talkMenu("\"I have already copied the best leaf from our shelves for you this tenday.\"")
		return
	}
	r := NewRng(fmt.Sprintf("%s:ll:shop:%s", g.W.Seed, key))
	var it *Item
	if r.Chance(0.65) {
		if ev := g.llPickEvent(r, p); ev >= 0 {
			it = g.llMakeScroll(r, ev)
		}
	}
	if it == nil {
		it = g.GenBook(r, p)
	}
	if it == nil {
		g.talkMenu("\"Our shelves are bare of fresh copies today.\"")
		return
	}
	price := maxi(14, it.Value)
	g.rtScene(a, fmt.Sprintf("%s lays out {{C|%s}}.\n%s\n\n\"Yours for %s.\"", g.The(a), it.Name, it.Lore, g.cuFormat(price)),
		Option{
			Text: fmt.Sprintf("Buy it (%s).", g.cuFormat(price)),
			Do: func() {
				if !g.cuSpend(price) {
					g.talkMenu("\"Come back when you have the coin.\"")
					return
				}
				st.Bought[key] = true
				if len(st.Bought) > 32 {
					curSlot := strconv.Itoa(slot)
					for k := range st.Bought {
						if !strings.HasSuffix(k, ":"+curSlot) {
							delete(st.Bought, k)
						}
					}
				}
				g.give(it)
				g.llSync()
				g.talkMenu(fmt.Sprintf("You tuck {{C|%s}} into your pack and note it in your lore library.", it.Name))
			},
		},
		g.rtBack(a),
	)
}

// ---- collection log & reader screen ----

func (g *Game) llOpen() {
	g.llSync()
	st := g.ll()
	st.reading = ""
	st.scroll = 0
	st.lines = nil
	g.Mode = "ll"
}

func (g *Game) llFiltered() []*llEntry {
	st := g.ll()
	cat := llCategories[clampi(st.catIdx, 0, len(llCategories)-1)]
	var out []*llEntry
	for _, e := range st.Entries {
		if cat == "All" || e.Category == cat {
			out = append(out, e)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Order > out[j].Order })
	return out
}

func llKeys(g *Game, k string) {
	st := g.ll()
	if st.reading != "" {
		maxSc := maxi(0, len(st.lines)-llVis)
		switch k {
		case "Escape", "q", "Backspace":
			st.reading = ""
			st.lines = nil
			st.scroll = 0
		case "ArrowDown", "N2", "j":
			st.scroll = mini(maxSc, st.scroll+1)
		case "ArrowUp", "N8", "k":
			st.scroll = maxi(0, st.scroll-1)
		case "PageDown", " ":
			st.scroll = mini(maxSc, st.scroll+llVis-2)
		case "PageUp":
			st.scroll = maxi(0, st.scroll-(llVis-2))
		}
		if st.scroll >= maxSc {
			if e := st.Entries[st.reading]; e != nil {
				g.llMarkRead(e)
			}
		}
		return
	}

	rows := g.llFiltered()
	switch k {
	case "Escape", "q":
		g.Mode = "main"
	case "Tab", "ArrowRight":
		st.catIdx = (st.catIdx + 1) % len(llCategories)
		st.sel = 0
	case "ArrowLeft":
		st.catIdx = (st.catIdx + len(llCategories) - 1) % len(llCategories)
		st.sel = 0
	case "ArrowUp", "N8", "k":
		st.sel = maxi(0, st.sel-1)
	case "ArrowDown", "N2", "j":
		st.sel = mini(maxi(0, len(rows)-1), st.sel+1)
	case "Enter":
		if len(rows) > 0 {
			e := rows[clampi(st.sel, 0, len(rows)-1)]
			st.reading = e.Key
			st.scroll = 0
			st.lines = Wrap(e.Text, MW-14)
			if len(st.lines) <= llVis {
				g.llMarkRead(e)
			}
		}
	}
}

func (g *Game) llDraw() {
	st := g.ll()
	sc := g.sc
	x, y, w, h := g.panel("Lore library")
	bottom := y + h - 1

	if st.reading != "" {
		e := st.Entries[st.reading]
		if e == nil {
			st.reading = ""
		} else {
			sc.TextB(x, y, fmt.Sprintf("{{Y|%s}} {{K|[%s] Line %d/%d}}", e.Title, e.Category, mini(st.scroll+llVis, len(st.lines)), len(st.lines)), CGray, CBgDark, x+w)
			for i := 0; i < llVis && st.scroll+i < len(st.lines); i++ {
				sc.TextB(x, y+2+i, st.lines[st.scroll+i], CGray, CBgDark, x+w)
			}
			sc.TextB(x, bottom, "{{K|Up/Down/PgUp/PgDn scroll   Esc back to library}}", CGray, CBgDark, x+w)
			return
		}
	}

	total := len(st.Entries)
	readN := 0
	for _, e := range st.Entries {
		if e.Read {
			readN++
		}
	}
	evTot := 0
	if g.W != nil {
		evTot = len(g.W.Events)
	}
	cat := llCategories[clampi(st.catIdx, 0, len(llCategories)-1)]
	sc.TextB(x, y, fmt.Sprintf("{{W|Filter: [%s]}}  {{K|Collected: %d (%d read)  Events attested: %d/%d}}",
		cat, total, readN, len(st.EventsRead), evTot), CGray, CBgDark, x+w)
	y += 2

	rows := g.llFiltered()
	if len(rows) == 0 {
		sc.TextB(x, y, "{{K|No works in this section yet. Search shelves, ruins, or speak with scholars and booksellers.}}", CGray, CBgDark, x+w)
		sc.TextB(x, bottom, "{{K|Tab filter   Esc close}}", CGray, CBgDark, x+w)
		return
	}

	st.sel = clampi(st.sel, 0, len(rows)-1)
	maxRows := h - 6
	if maxRows < 4 {
		maxRows = 4
	}
	start := 0
	if st.sel >= maxRows {
		start = st.sel - maxRows + 1
	}
	for i := start; i < mini(len(rows), start+maxRows) && y < bottom-2; i++ {
		e := rows[i]
		mark := "  "
		if i == st.sel {
			mark = "> "
		}
		rd := "{{K|unread}}"
		if e.Read {
			rd = "{{G|read}}"
		}
		yr := ""
		if e.Year > 0 {
			yr = " Y" + strconv.Itoa(e.Year)
		}
		sc.TextB(x, y, fmt.Sprintf("%s{{W|%s}} {{c|[%s%s]}} %s", mark, fpShort(e.Title, 52), e.Category, yr, rd), CGray, CBgDark, x+w)
		y++
	}
	sc.TextB(x, bottom, "{{K|Up/Down choose   Enter read   Tab category   Esc close}}", CGray, CBgDark, x+w)
}
