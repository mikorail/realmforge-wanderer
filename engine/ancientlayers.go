package engine

import (
	"fmt"
	"sort"
	"strings"
)

// Ruins and ancient layers: pre-collapse sites holding lore fragments that, once
// collected, reveal one real piece of the world's history from WorldData.Events
// (prefix al2). Links read-only to legendlore.go (g.lbArms / g.lb) without mutating it.
//
// State lives in g.ext("al2") (planet-local). All generation uses side rngs (seed:al2:...).

const al2FragsPerSite = 3

func init() {
	unLocalKeys = append(unLocalKeys, "al2")
	zoneHooks = append(zoneHooks, al2Zone)
	enterHooks = append(enterHooks, al2Enter)
	turnHooks = append(turnHooks, al2Turn)
	talkHooks = append(talkHooks, al2Talk)
	modeKeys["al2"] = al2Keys
	modeDraws["al2"] = func(g *Game) { g.al2Draw() }
	paEntries = append(paEntries, paEntry{
		Name:     "Ancient layers and ruins",
		Keywords: "ancient layers ruins pre-collapse fragments history archaeology stele",
		Cat:      "Screen",
		Label:    "screen",
		Func:     func(g *Game) { st := g.al2(); st.tab, st.sel = 0, 0; g.Mode = "al2" },
	})
}

type al2Site struct {
	ID         int
	Key        string
	Name       string
	Kind       string // ruined burg, shattered wonder, buried stratum
	P          int
	WX, WY     int
	ZX, ZY     int
	Depth      int // 0 = surface ruin, 1..2 = ancient layer depth
	EvIdx      int // index into WorldData.Events
	Year       int
	EvKind     string
	EvText     string
	Region     string
	FragNames  [al2FragsPerSite]string
	FragClues  [al2FragsPerSite]string
	LinkedArm  string // read-only key into legendlore.go lbArm.Key (optional)
	LinkedName string
}

type al2Prog struct {
	Known      bool                  `json:"k,omitempty"`
	Discovered bool                  `json:"d,omitempty"`
	Frags      [al2FragsPerSite]bool `json:"f,omitempty"`
	Revealed   bool                  `json:"r,omitempty"`
	Via        string                `json:"v,omitempty"`
}

type al2State struct {
	P   map[string]*al2Prog `json:"p,omitempty"`
	Log []string            `json:"l,omitempty"`

	sites      []*al2Site
	sitesByP   map[int][]*al2Site
	sitesFor   string
	lastInvSig uint64
	tab        int
	sel        int
}

func (g *Game) al2() *al2State {
	mk := func() any {
		return &al2State{P: map[string]*al2Prog{}}
	}
	s := g.ext("al2", mk).(*al2State)
	if s.P == nil {
		s.P = map[string]*al2Prog{}
	}
	return s
}

func (g *Game) al2Prog(site *al2Site) *al2Prog {
	st := g.al2()
	p := st.P[site.Key]
	if p == nil {
		p = &al2Prog{}
		st.P[site.Key] = p
	}
	return p
}

func (p *al2Prog) count() int {
	n := 0
	for _, ok := range p.Frags {
		if ok {
			n++
		}
	}
	return n
}

// ---- deterministic pre-collapse site generation ----

func (g *Game) al2Sites() []*al2Site {
	st := g.al2()
	if g.W == nil {
		return nil
	}
	if st.sitesFor == g.W.Seed && st.sites != nil {
		return st.sites
	}
	st.sitesFor = g.W.Seed
	st.sites = g.al2Gen()
	st.sitesByP = make(map[int][]*al2Site, len(st.sites))
	for _, s := range st.sites {
		st.sitesByP[s.P] = append(st.sitesByP[s.P], s)
	}
	return st.sites
}

var al2EraScore = map[string]int{
	"fall": 6, "disaster": 5, "eco_disaster": 5, "wonder": 5, "god": 4,
	"war": 4, "faith": 3, "monster": 3, "coup": 2, "uprising": 2, "peace": 1,
}

var al2FragNouns = []string{
	"obsidian tablet", "bronze cylinder", "chipped stele shard", "sealed clay prism",
	"etched slate leaf", "weathered pillar rubbing", "engraved copper plate", "basalt seal",
}

func (g *Game) al2Gen() []*al2Site {
	w := g.W
	if w == nil || len(w.Events) == 0 {
		return nil
	}
	r := NewRng(w.Seed + ":al2:sites")

	// Collect candidate anchor locations: destroyed wonders, dead burgs, ancient dungeons/ruins.
	type anchor struct {
		name, kind string
		p          int
		zx, zy     int
		depth      int
	}
	var anchors []anchor
	usedP := map[int]bool{}

	for i := range w.Wonders {
		wd := &w.Wonders[i]
		if wd.Destroyed && wd.P >= 0 && wd.P < len(w.Biome) && !w.IsWater(wd.P) && !usedP[wd.P] {
			anchors = append(anchors, anchor{
				name:  "Ruins of " + wd.Name,
				kind:  "shattered wonder",
				p:     wd.P,
				zx:    1,
				zy:    1,
				depth: 0,
			})
			usedP[wd.P] = true
		}
	}
	for i := range w.Burgs {
		b := &w.Burgs[i]
		if !b.Alive && b.P >= 0 && b.P < len(w.Biome) && !w.IsWater(b.P) && !usedP[b.P] {
			anchors = append(anchors, anchor{
				name:  "Old " + b.Name + " Stratum",
				kind:  "ruined burg",
				p:     b.P,
				zx:    1,
				zy:    1,
				depth: 0,
			})
			usedP[b.P] = true
		}
	}
	// Supplement with ancient dungeon/ruin sites from siteAt so every planet gets 6 to 10 sites.
	want := 6 + int(HashStr(w.Seed+":al2:n")%5)
	if len(anchors) < want {
		for p := 0; p < len(w.Biome) && len(anchors) < want*3; p++ {
			if w.IsWater(p) || w.Burg[p] > 0 || usedP[p] {
				continue
			}
			wx, wy := p%w.W, p/w.W
			for zy := 0; zy < 3; zy++ {
				for zx := 0; zx < 3; zx++ {
					s := g.siteAt(wx, wy, zx, zy)
					if s != nil && s.Kind == "dungeon" && s.Depth >= 1 && s.Outlaw == nil {
						anchors = append(anchors, anchor{
							name:  "Pre-Collapse Layer of " + s.Name,
							kind:  "buried stratum",
							p:     p,
							zx:    zx,
							zy:    zy,
							depth: 1,
						})
						usedP[p] = true
						break
					}
				}
				if usedP[p] {
					break
				}
			}
		}
	}
	if len(anchors) == 0 {
		return nil
	}

	// Pick the oldest / most significant real events from WorldData.Events.
	evIdxs := make([]int, 0, len(w.Events))
	for i := range w.Events {
		if strings.TrimSpace(w.Events[i].Text) != "" {
			evIdxs = append(evIdxs, i)
		}
	}
	sort.SliceStable(evIdxs, func(i, j int) bool {
		ei, ej := &w.Events[evIdxs[i]], &w.Events[evIdxs[j]]
		si, sj := al2EraScore[ei.Kind], al2EraScore[ej.Kind]
		if si != sj {
			return si > sj
		}
		return ei.Y < ej.Y // earlier history first
	})

	// Read-only link list from legendlore.go.
	arms := g.lbArms()

	limit := mini(want, mini(len(anchors), len(evIdxs)))
	var out []*al2Site
	usedEv := map[int]bool{}

	for i := 0; i < limit; i++ {
		an := anchors[i]
		// Prefer an event near this anchor, else take the next oldest unused event.
		evIdx := -1
		for _, idx := range evIdxs {
			if usedEv[idx] {
				continue
			}
			e := &w.Events[idx]
			if e.P >= 0 && glDist(w, e.P, an.p) <= 14 {
				evIdx = idx
				break
			}
		}
		if evIdx < 0 {
			for _, idx := range evIdxs {
				if !usedEv[idx] {
					evIdx = idx
					break
				}
			}
		}
		if evIdx < 0 {
			break
		}
		usedEv[evIdx] = true
		ev := w.Events[evIdx]
		reg := w.RegionName(an.p)
		wx, wy := an.p%w.W, an.p/w.W

		site := &al2Site{
			ID:     len(out),
			Key:    fmt.Sprintf("%s:al2:%d", w.Seed, evIdx),
			Name:   an.name,
			Kind:   an.kind,
			P:      an.p,
			WX:     wx,
			WY:     wy,
			ZX:     an.zx,
			ZY:     an.zy,
			Depth:  an.depth,
			EvIdx:  evIdx,
			Year:   ev.Y,
			EvKind: ev.Kind,
			EvText: ev.Text,
			Region: reg,
		}

		// Split the real event into three archaeological fragment clues.
		words := strings.Fields(ev.Text)
		snip1, snip2, snip3 := ev.Text, "", ""
		if len(words) >= 6 {
			a := len(words) / 3
			b := (2 * len(words)) / 3
			snip1 = strings.Join(words[:a], " ") + "..."
			snip2 = "..." + strings.Join(words[a:b], " ") + "..."
			snip3 = "..." + strings.Join(words[b:], " ")
		} else {
			snip2 = fmt.Sprintf("dated to year %d", ev.Y)
			snip3 = fmt.Sprintf("recorded near %s", reg)
		}
		snips := [al2FragsPerSite]string{snip1, snip2, snip3}
		for f := 0; f < al2FragsPerSite; f++ {
			noun := Pick(r, al2FragNouns)
			site.FragNames[f] = fmt.Sprintf("%s (%s, %d/3)", noun, an.name, f+1)
			site.FragClues[f] = fmt.Sprintf("Fragment %d of 3 from %s (Y%d stratum): \"%s\"", f+1, an.name, ev.Y, snips[f])
		}

		// Read-only cross-link to nearest legendlore.go arm within 14 parasangs.
		bestDist := 15
		for _, arm := range arms {
			d1 := maxi(abs(arm.S1.WX-wx), abs(arm.S1.WY-wy))
			d2 := maxi(abs(arm.S2.WX-wx), abs(arm.S2.WY-wy))
			d := mini(d1, d2)
			if d < bestDist {
				bestDist = d
				site.LinkedArm = arm.Key
				site.LinkedName = arm.Name
			}
		}

		out = append(out, site)
	}
	return out
}

// ---- zone, enter, turn, talk hooks ----

func al2Zone(g *Game, zn *Zone, _ *Rng) {
	if g.W == nil || zn == nil || zn.Underwater {
		return
	}
	if !g.W.In(zn.WX, zn.WY) {
		return
	}
	p := g.W.Idx(zn.WX, zn.WY)
	g.al2Sites()
	for _, site := range g.al2().sitesByP[p] {
		if zn.Z != site.Depth || zn.ZX != site.ZX || zn.ZY != site.ZY {
			continue
		}
		r := NewRng(g.W.Seed + ":al2:zone:" + site.Key + ":" + zn.Key)
		// Place all 3 lore fragments across the site's zone, plus an ancient stele inscription.
		for f := 0; f < al2FragsPerSite; f++ {
			x, y, ok := zn.FreeSpot(r, func(x, y int) bool {
				t := zn.At(x, y)
				return (t == TFloor || t == TCobble || t == TDirt || t == TGrass || t == TSand) && x > 2 && x < MW-3 && y > 1 && y < MH-2
			})
			if !ok {
				continue
			}
			it := &Item{
				ID:     g.newID(),
				Kind:   "lore",
				Name:   site.FragNames[f],
				Glyph:  '?',
				F:      CCyan,
				Value:  12,
				Weight: 1,
				Ident:  true,
				Text:   site.FragClues[f],
				Lore:   fmt.Sprintf("A pre-collapse fragment from %s.", site.Name),
				Extra:  map[string]int{"al2": site.ID + 1, "al2:f": f + 1},
				X:      x,
				Y:      y,
			}
			zn.Items = append(zn.Items, it)
		}
		if x, y, ok := zn.FreeSpot(r, func(x, y int) bool {
			t := zn.At(x, y)
			return (t == TFloor || t == TCobble || t == TDirt || t == TGrass) && x > 3 && x < MW-4 && y > 2 && y < MH-3
		}); ok {
			zn.Set(x, y, TInscription)
			if zn.Notes == nil {
				zn.Notes = map[int]string{}
			}
			zn.Notes[y*MW+x] = fmt.Sprintf("Pre-collapse stele of %s: three fragments lie scattered in this stratum. Collect them to reconstruct the record of year %d.", site.Name, site.Year)
		}
	}
}

func al2Enter(g *Game, zn *Zone) {
	if g.W == nil || g.P == nil || zn == nil || !g.W.In(zn.WX, zn.WY) {
		return
	}
	p := g.W.Idx(zn.WX, zn.WY)
	g.al2Sites()
	for _, site := range g.al2().sitesByP[p] {
		if zn.Z != site.Depth {
			continue
		}
		pr := g.al2Prog(site)
		if !pr.Discovered {
			pr.Known = true
			pr.Discovered = true
			g.Known[site.P] = true
			g.al2Log(fmt.Sprintf("Entered pre-collapse site: %s (%s).", site.Name, site.Region))
			g.Msg(fmt.Sprintf("{{C|Ancient layer: %s.}} Weathered pre-collapse masonry marks this stratum; search for lore fragments.", site.Name))
		}
	}
}

func (g *Game) al2Log(s string) {
	st := g.al2()
	st.Log = append(st.Log, fmt.Sprintf("Day %d: %s", DayOf(g.Turn)+1, s))
	if len(st.Log) > 24 {
		st.Log = st.Log[len(st.Log)-24:]
	}
}

func al2Turn(g *Game) {
	if g.W == nil || g.P == nil || g.Dead {
		return
	}
	var sig uint64
	hasFrag := false
	for _, it := range g.P.Inv {
		if it != nil && it.Kind == "lore" && it.Extra != nil && it.Extra["al2"] > 0 {
			hasFrag = true
			sig = sig*31 + uint64(it.ID) + 1
		}
	}
	if !hasFrag {
		return
	}
	st := g.al2()
	if sig == st.lastInvSig {
		return
	}
	st.lastInvSig = sig
	sites := g.al2Sites()
	if len(sites) == 0 {
		return
	}
	for _, it := range g.P.Inv {
		if it == nil || it.Kind != "lore" || it.Extra == nil || it.Extra["al2"] <= 0 {
			continue
		}
		sid := it.Extra["al2"] - 1
		fid := it.Extra["al2:f"] - 1
		if sid < 0 || sid >= len(sites) || fid < 0 || fid >= al2FragsPerSite {
			continue
		}
		site := sites[sid]
		pr := g.al2Prog(site)
		if pr.Frags[fid] {
			continue
		}
		pr.Known = true
		pr.Discovered = true
		pr.Frags[fid] = true
		g.gainXP(10)
		g.al2Log(fmt.Sprintf("Recovered fragment %d/3 at %s.", fid+1, site.Name))
		g.Msg(fmt.Sprintf("{{C|Fragment %d of 3 catalogued (%s):}} %s", pr.count(), site.Name, site.FragClues[fid]))
		if pr.count() >= al2FragsPerSite && !pr.Revealed {
			g.al2Reveal(site, pr)
		}
	}
}

func (g *Game) al2Reveal(site *al2Site, pr *al2Prog) {
	pr.Revealed = true
	line := fmt.Sprintf("Pre-collapse record from %s (Y%d): %s", site.Name, site.Year, site.EvText)
	g.addSecret(line, "history", site.P, site.Year)
	g.gainXP(35 + g.Skills["lore"]*5)
	g.deed("lore", fmt.Sprintf("%s reconstructed the pre-collapse record of %s (Y%d).", g.P.Name, site.Name, site.Year), 0, 0)
	g.al2Log(fmt.Sprintf("Reconstructed Y%d record at %s.", site.Year, site.Name))
	g.Msg(fmt.Sprintf("{{Y|Ancient record restored (%s, Y%d):}} %s", site.Name, site.Year, site.EvText))
	if site.LinkedArm != "" {
		g.Msg(fmt.Sprintf("{{c|Cross-reference: the fragments also mention %s (see Ancient layers or Legendary arms).}}", site.LinkedName))
	}
}

func al2Talk(g *Game, a *Actor) []Option {
	if g.W == nil || g.P == nil || g.Cur == nil || a == nil || a.HP <= 0 || a.AI == "follower" || g.Hostile(a) {
		return nil
	}
	switch a.Role {
	case "scholar", "chronicler", "archivist", "bookseller", "explorer":
	default:
		return nil
	}
	var pick *al2Site
	for _, s := range g.al2Sites() {
		if !g.al2Prog(s).Known {
			pick = s
			break
		}
	}
	if pick == nil {
		return nil
	}
	s := pick
	return []Option{{
		Text: "Ask about pre-collapse ruins and ancient layers.",
		Do: func() {
			pr := g.al2Prog(s)
			pr.Known = true
			pr.Via = "spoken of by " + a.Name
			if g.W.In(s.WX, s.WY) {
				g.Known[s.P] = true
			}
			g.al2Log(fmt.Sprintf("Heard of %s in %s.", s.Name, s.Region))
			depthNote := "on the surface"
			if s.Depth > 0 {
				depthNote = fmt.Sprintf("at depth %d", s.Depth)
			}
			g.talkMenu(fmt.Sprintf("\"Seek %s, a %s in %s (%s, %s). Old fragments from year %d still lie there.\"", s.Name, s.Kind, s.Region, g.cxDir(s.P), depthNote, s.Year))
		},
	}}
}

// ---- read-only legendlore.go link lookup ----

func (g *Game) al2LegendNote(site *al2Site) string {
	if site == nil || site.LinkedArm == "" {
		return ""
	}
	var arm *lbArm
	for _, a := range g.lbArms() {
		if a.Key == site.LinkedArm {
			arm = a
			break
		}
	}
	if arm == nil {
		return ""
	}
	// Strictly read-only inspection of lb state.
	st := g.lb()
	var prog *lbProg
	if st != nil && st.P != nil {
		prog = st.P[arm.Key]
	}
	status := "untraced"
	if prog != nil {
		switch {
		case prog.Found:
			status = "found"
		case prog.L2:
			status = "lair known (" + arm.S2.Name + ")"
		case prog.L1:
			status = "trail leads to " + arm.S1.Name
		}
	}
	return fmt.Sprintf("Linked legend (read-only): %s, bound to %s (%s; %s).", arm.Name, arm.Person, status, g.cxDir(g.W.Idx(arm.S1.WX, arm.S1.WY)))
}

// ---- screen ----

func al2Keys(g *Game, k string) {
	st := g.al2()
	sites := g.al2Sites()
	switch k {
	case "Escape", "q":
		g.Mode = "main"
	case "Tab", "ArrowRight", "ArrowLeft":
		st.tab = 1 - st.tab
		st.sel = 0
	case "ArrowUp", "N8", "k":
		st.sel = maxi(0, st.sel-1)
	case "ArrowDown", "N2", "j":
		st.sel = mini(maxi(0, len(sites)-1), st.sel+1)
	}
}

func (g *Game) al2Draw() {
	st := g.al2()
	sc := g.sc
	x, y, w, h := g.panel("Ancient layers and ruins")
	bottom := y + h - 1
	sites := g.al2Sites()

	revCount := 0
	for _, s := range sites {
		if g.al2Prog(s).Revealed {
			revCount++
		}
	}
	tabCol := func(on bool) string {
		if on {
			return "W"
		}
		return "K"
	}
	sc.TextB(x, y, fmt.Sprintf("{{%s|[ Pre-Collapse Sites %d/%d ]}}  {{%s|[ Revealed History & Log ]}}",
		tabCol(st.tab == 0), revCount, len(sites), tabCol(st.tab == 1)), CGray, CBgDark, x+w)
	y += 2

	put := func(s string) {
		for _, ln := range Wrap(s, w) {
			if y < bottom {
				sc.TextB(x, y, ln, CGray, CBgDark, x+w)
				y++
			}
		}
	}

	if st.tab == 1 {
		shown := false
		for _, s := range sites {
			if !g.al2Prog(s).Revealed {
				continue
			}
			shown = true
			put(fmt.Sprintf("{{Y|Y%d - %s}} (%s)", s.Year, s.Name, s.Region))
			put("  " + s.EvText)
			if ln := g.al2LegendNote(s); ln != "" {
				put("  {{c|" + ln + "}}")
			}
			y++
		}
		if !shown {
			put("No pre-collapse records have been fully reconstructed yet. Collect all 3 fragments at an ancient site to restore its history.")
		}
		if len(st.Log) > 0 && y < bottom-2 {
			put("{{W|Field log}}")
			for i := len(st.Log) - 1; i >= 0 && y < bottom; i-- {
				put("{{K|" + st.Log[i] + "}}")
			}
		}
		sc.TextB(x, bottom, "{{K|Tab sites   Esc close}}", CGray, CBgDark, x+w)
		return
	}

	var known []*al2Site
	for _, s := range sites {
		if g.al2Prog(s).Known || g.al2Prog(s).Discovered || g.al2Prog(s).Revealed {
			known = append(known, s)
		}
	}
	if len(known) == 0 {
		put(fmt.Sprintf("This planet holds %d pre-collapse ruin sites and buried strata. Ask scholars or explorers in town, or explore ruins and deep layers to locate them.", len(sites)))
		sc.TextB(x, bottom, "{{K|Tab history   Esc close}}", CGray, CBgDark, x+w)
		return
	}

	st.sel = clampi(st.sel, 0, len(known)-1)
	maxShow := mini(len(known), 8)
	start := 0
	if st.sel >= maxShow {
		start = st.sel - maxShow + 1
	}
	for i := start; i < mini(len(known), start+maxShow); i++ {
		s := known[i]
		pr := g.al2Prog(s)
		mark := "  "
		if i == st.sel {
			mark = "> "
		}
		badge := fmt.Sprintf("{{c|%d/3 frags}}", pr.count())
		if pr.Revealed {
			badge = "{{G|restored}}"
		}
		put(fmt.Sprintf("%s{{W|%s}} %s {{K|%s}}", mark, fpShort(s.Name, 40), badge, g.cxDir(s.P)))
	}
	y++

	s := known[st.sel]
	pr := g.al2Prog(s)
	depthNote := "surface"
	if s.Depth > 0 {
		depthNote = fmt.Sprintf("depth %d", s.Depth)
	}
	put(fmt.Sprintf("{{Y|%s}} (%s, %s, %s)", s.Name, s.Kind, s.Region, depthNote))
	if pr.Revealed {
		put(fmt.Sprintf("{{G|Restored record (Y%d):}} %s", s.Year, s.EvText))
	} else {
		put(fmt.Sprintf("Fragments recovered: %d of %d.", pr.count(), al2FragsPerSite))
		for f := 0; f < al2FragsPerSite; f++ {
			if pr.Frags[f] {
				put("  " + s.FragClues[f])
			}
		}
	}
	if ln := g.al2LegendNote(s); ln != "" {
		put("{{c|" + ln + "}}")
	}
	sc.TextB(x, bottom, "{{K|Up/Down choose   Tab history   Esc close}}", CGray, CBgDark, x+w)
}
