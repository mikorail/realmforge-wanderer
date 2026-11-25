package engine

import (
	"fmt"
	"sort"
	"strings"
)

// Faiths in play: per-faith shrine blessings, offerings, pilgrimages to named holy sites,
// conversions and heresy trials (prefix fp). Complements rituals.go (prayers, liturgy,
// saints, scripture, confession), orders.go (joinable orders/cults), visions.go (omens/seers),
// realmstory.go (realm political crises) and saints_faith.go (divine oaths).
//
// State lives in g.ext("fp") (planet-local). All generation uses side rngs (seed:fp:...).

const (
	fpBlessAegis     = "fp_aegis"
	fpBlessClarity   = "fp_clarity"
	fpBlessVitality  = "fp_vitality"
	fpBlessZeal      = "fp_zeal"
	fpBlessGrace     = "fp_grace"
	fpBlessAncestors = "fp_ancestors"
)

var fpAllBlessings = []string{
	fpBlessAegis, fpBlessClarity, fpBlessVitality, fpBlessZeal, fpBlessGrace, fpBlessAncestors,
}

func init() {
	unLocalKeys = append(unLocalKeys, "fp")
	skAddStatus(&skStatusDef{ID: fpBlessAegis, Name: "Shrine Aegis", Glyph: '+', AV: 2, DV: 1})
	skAddStatus(&skStatusDef{ID: fpBlessClarity, Name: "Shrine Clarity", Glyph: '+', INT: 2, WIL: 2, MA: 1})
	skAddStatus(&skStatusDef{ID: fpBlessVitality, Name: "Shrine Vitality", Glyph: '+', DV: 1, Speed: 5, Tick: fpVitalityTick})
	skAddStatus(&skStatusDef{ID: fpBlessZeal, Name: "Shrine Zeal", Glyph: '+', Hit: 2, PV: 1})
	skAddStatus(&skStatusDef{ID: fpBlessGrace, Name: "Shrine Grace", Glyph: '+', EGO: 2, WIL: 1, DV: 1})
	skAddStatus(&skStatusDef{ID: fpBlessAncestors, Name: "Ancestral Vigil", Glyph: '+', AV: 1, WIL: 2, MA: 2})

	zoneHooks = append(zoneHooks, fpZone)
	enterHooks = append(enterHooks, fpEnter)
	talkHooks = append(talkHooks, fpTalk)
	sideHooks = append(sideHooks, fpSide)

	modeKeys["fp"] = fpKeys
	modeDraws["fp"] = func(g *Game) { g.fpDraw() }
	paEntries = append(paEntries, paEntry{
		Name:     "Faiths and shrines",
		Keywords: "faith religion shrine blessing offering pilgrimage heresy trial conversion deity",
		Cat:      "Screen",
		Label:    "screen",
		Func:     func(g *Game) { st := g.fp(); st.tab, st.sel = 0, 0; g.Mode = "fp" },
	})
}

func fpVitalityTick(g *Game, a *Actor) {
	if g != nil && a != nil && a.HP > 0 && a.HP < a.MaxHP && g.Turn%30 == 0 {
		a.HP = mini(a.MaxHP, a.HP+1)
	}
}

// ---- saved state ----

type fpTrialRec struct {
	FaithID int    `json:"f"`
	StateID int    `json:"s"`
	Day     int    `json:"d"`
	Outcome string `json:"o"` // acquitted, recanted, defiant, condemned
	Note    string `json:"n"`
}

type fpState struct {
	OfferDay   map[int]int     `json:"od,omitempty"` // faithID -> day+1 of last shrine offering
	ActiveVow  string          `json:"av,omitempty"` // holy site ID currently vowed
	Visited    map[string]bool `json:"vs,omitempty"` // holy site ID -> completed pilgrimage
	KnownSites map[string]bool `json:"ks,omitempty"` // holy site ID -> known to player
	Heretic    map[int]bool    `json:"hr,omitempty"` // faithID -> branded heretic
	Summoned   map[int]int     `json:"sm,omitempty"` // faithID -> day+1 of last summons notice
	Converted  []int           `json:"cv,omitempty"` // history of faith IDs converted to
	Trials     []fpTrialRec    `json:"tr,omitempty"`
	ActiveName string          `json:"an,omitempty"` // display name of current shrine blessing

	sites     []*fpSite
	sitesByP  map[int][]*fpSite
	sitesByID map[string]*fpSite
	sitesFor  string
	tab       int
	sel       int
}

func (g *Game) fp() *fpState {
	mk := func() any {
		return &fpState{
			OfferDay:   map[int]int{},
			Visited:    map[string]bool{},
			KnownSites: map[string]bool{},
			Heretic:    map[int]bool{},
			Summoned:   map[int]int{},
		}
	}
	s := g.ext("fp", mk).(*fpState)
	if s.OfferDay == nil {
		s.OfferDay = map[int]int{}
	}
	if s.Visited == nil {
		s.Visited = map[string]bool{}
	}
	if s.KnownSites == nil {
		s.KnownSites = map[string]bool{}
	}
	if s.Heretic == nil {
		s.Heretic = map[int]bool{}
	}
	if s.Summoned == nil {
		s.Summoned = map[int]int{}
	}
	return s
}

// ---- per-faith blessing profile ----

type fpBlessInfo struct {
	StatusID string
	Title    string
	Summary  string
	Quote    string
	Offer    string
}

func fpBlessingFor(w *WorldData, rel *ReligionData) fpBlessInfo {
	if rel == nil {
		return fpBlessInfo{
			StatusID: fpBlessGrace,
			Title:    "Wayside Grace",
			Summary:  "+2 EGO, +1 WIL, +1 DV",
			Quote:    "Walk lightly and keep your word.",
			Offer:    "a small coin or travel bread",
		}
	}
	comb := strings.ToLower(rel.Type + " " + rel.Theology + " " + rel.Doctrine)
	god := nonEmpty(rel.Deity, rel.Name)
	var statusID, label, summary, offer string
	switch {
	case strings.Contains(comb, "militant") || strings.Contains(comb, "crusade") || strings.Contains(comb, "orthodox") || strings.Contains(comb, "guard"):
		statusID, label, summary, offer = fpBlessAegis, "Aegis of "+god, "+2 AV, +1 DV", "iron, coin, or rations"
	case strings.Contains(comb, "esoter") || strings.Contains(comb, "gnost") || strings.Contains(comb, "scholar") || strings.Contains(comb, "revel") || strings.Contains(comb, "light"):
		statusID, label, summary, offer = fpBlessClarity, "Clarity of "+god, "+2 INT, +2 WIL, +1 MA", "a book, scroll, or tonic"
	case strings.Contains(comb, "anim") || strings.Contains(comb, "panthe") || strings.Contains(comb, "nature") || strings.Contains(comb, "harvest") || strings.Contains(comb, "earth"):
		statusID, label, summary, offer = fpBlessVitality, "Vitality of "+god, "+1 DV, +5 Speed, slow healing", "herbs, food, or fresh water"
	case strings.Contains(comb, "evangel") || strings.Contains(comb, "war") || strings.Contains(comb, "fire") || strings.Contains(comb, "zeal") || strings.Contains(comb, "sun"):
		statusID, label, summary, offer = fpBlessZeal, "Zeal of "+god, "+2 Hit, +1 PV", "a trophy, tonic, or coin"
	case strings.Contains(comb, "ancest") || strings.Contains(comb, "monast") || strings.Contains(comb, "dead") || strings.Contains(comb, "myst") || strings.Contains(comb, "silent"):
		statusID, label, summary, offer = fpBlessAncestors, "Vigil of "+god, "+1 AV, +2 WIL, +2 MA", "incense, food, or coin"
	default:
		// Deterministically pick one of the 6 profiles from the world seed and religion id.
		seed := ""
		if w != nil {
			seed = w.Seed
		}
		h := HashStr(fmt.Sprintf("%s:fp:bless:%d", seed, rel.ID))
		switch h % 6 {
		case 0:
			statusID, label, summary, offer = fpBlessAegis, "Aegis of "+god, "+2 AV, +1 DV", "coin or rations"
		case 1:
			statusID, label, summary, offer = fpBlessClarity, "Clarity of "+god, "+2 INT, +2 WIL, +1 MA", "a book or tonic"
		case 2:
			statusID, label, summary, offer = fpBlessVitality, "Vitality of "+god, "+1 DV, +5 Speed, slow healing", "food or herbs"
		case 3:
			statusID, label, summary, offer = fpBlessZeal, "Zeal of "+god, "+2 Hit, +1 PV", "a trophy or coin"
		case 4:
			statusID, label, summary, offer = fpBlessGrace, "Grace of "+god, "+2 EGO, +1 WIL, +1 DV", "coin or food"
		default:
			statusID, label, summary, offer = fpBlessAncestors, "Vigil of "+god, "+1 AV, +2 WIL, +2 MA", "food or coin"
		}
	}
	quote := ""
	if rel.Lore != nil && len(rel.Lore.Blessings) > 0 {
		seed := ""
		if w != nil {
			seed = w.Seed
		}
		idx := int(HashStr(fmt.Sprintf("%s:fp:bq:%d", seed, rel.ID)) % uint64(len(rel.Lore.Blessings)))
		quote = rel.Lore.Blessings[idx]
	} else {
		th := nonEmpty(rel.Theology, "the old covenant")
		quote = fmt.Sprintf("May %s keep you in %s.", god, strings.ToLower(th))
	}
	return fpBlessInfo{
		StatusID: statusID,
		Title:    label,
		Summary:  summary,
		Quote:    quote,
		Offer:    offer,
	}
}

// ---- named holy sites ----

type fpSite struct {
	ID      string
	Name    string
	FaithID int
	BurgID  int
	P       int
	WX, WY  int
	Kind    string // temple, wonder, sanctuary
	Desc    string
}

func (g *Game) fpHolySites() []*fpSite {
	st := g.fp()
	if g.W == nil {
		return nil
	}
	if st.sitesFor == g.W.Seed && st.sites != nil {
		return st.sites
	}
	st.sitesFor = g.W.Seed
	st.sites = g.fpGenSites()
	st.sitesByP = make(map[int][]*fpSite, len(st.sites))
	st.sitesByID = make(map[string]*fpSite, len(st.sites))
	for _, s := range st.sites {
		st.sitesByP[s.P] = append(st.sitesByP[s.P], s)
		st.sitesByID[s.ID] = s
	}
	return st.sites
}

func (g *Game) fpGenSites() []*fpSite {
	w := g.W
	if w == nil || len(w.Religions) == 0 {
		return nil
	}
	r := NewRng(w.Seed + ":fp:sites")
	var out []*fpSite
	usedP := map[int]bool{}

	// Index burgs by religion.
	relBurgs := map[int][]*BurgData{}
	for i := range w.Burgs {
		b := &w.Burgs[i]
		if !b.Alive || b.Layer != "surface" || b.P < 0 || b.P >= len(w.Religion) {
			continue
		}
		rid := w.Religion[b.P]
		if rid > 0 {
			relBurgs[rid] = append(relBurgs[rid], b)
		}
	}
	for rid := range relBurgs {
		sort.SliceStable(relBurgs[rid], func(i, j int) bool {
			if relBurgs[rid][i].Pop != relBurgs[rid][j].Pop {
				return relBurgs[rid][i].Pop > relBurgs[rid][j].Pop
			}
			return relBurgs[rid][i].ID < relBurgs[rid][j].ID
		})
	}

	rels := make([]*ReligionData, 0, len(w.Religions))
	for i := range w.Religions {
		if w.Religions[i].ID > 0 && w.Religions[i].Name != "" {
			rels = append(rels, &w.Religions[i])
		}
	}
	sort.Slice(rels, func(i, j int) bool { return rels[i].ID < rels[j].ID })

	sitePrefixes := []string{"Sanctum of ", "High Shrine of ", "Pilgrim Seat of ", "Reliquary of ", "Sacred Hearth of "}
	wildPrefixes := []string{"Hermitage of ", "Stone Cairn of ", "Way-Shrine of ", "Grove of ", "High Altar of "}

	// Pre-bucket wilderness land cells by religion in a single O(W*H) pass instead of O(R*W*H).
	landByRel := map[int][]int{}
	limP := len(w.Religion)
	if len(w.Height) < limP {
		limP = len(w.Height)
	}
	for p := 0; p < limP; p++ {
		rid := w.Religion[p]
		if rid > 0 && !w.IsWater(p) && w.Burg[p] == 0 {
			landByRel[rid] = append(landByRel[rid], p)
		}
	}

	for _, rel := range rels {
		god := nonEmpty(rel.Deity, rel.Name)
		saintName := ""
		if rel.Lore != nil && len(rel.Lore.Saints) > 0 {
			saintName = rel.Lore.Saints[r.Int(0, len(rel.Lore.Saints)-1)].Name
		}

		// Primary site: largest town of this faith (or a wonder in its territory).
		var primary *fpSite
		for i := range w.Wonders {
			wd := &w.Wonders[i]
			if !wd.Destroyed && wd.Layer == "surface" && wd.P >= 0 && wd.P < len(w.Religion) && w.Religion[wd.P] == rel.ID && !usedP[wd.P] {
				primary = &fpSite{
					ID:      fmt.Sprintf("fp:%d:w%d", rel.ID, wd.ID),
					Name:    wd.Name + " (" + god + ")",
					FaithID: rel.ID,
					P:       wd.P,
					WX:      wd.P % w.W,
					WY:      wd.P / w.W,
					Kind:    "wonder",
					Desc:    fmt.Sprintf("A revered %s sacred to followers of %s.", strings.ToLower(nonEmpty(wd.Kind, "monument")), rel.Name),
				}
				usedP[wd.P] = true
				break
			}
		}
		if primary == nil && len(relBurgs[rel.ID]) > 0 {
			bd := relBurgs[rel.ID][0]
			nm := Pick(r, sitePrefixes) + god
			if saintName != "" && r.Chance(0.45) {
				nm = "Shrine of " + saintName
			}
			primary = &fpSite{
				ID:      fmt.Sprintf("fp:%d:b%d", rel.ID, bd.ID),
				Name:    nm + " at " + bd.Name,
				FaithID: rel.ID,
				BurgID:  bd.ID,
				P:       bd.P,
				WX:      bd.P % w.W,
				WY:      bd.P / w.W,
				Kind:    "temple",
				Desc:    fmt.Sprintf("The chief pilgrimage shrine of %s in %s.", rel.Name, bd.Name),
			}
			usedP[bd.P] = true
		}
		if primary != nil {
			out = append(out, primary)
		}

		// Secondary wilderness sanctuary in the faith's territory.
		var landCands []int
		for _, p := range landByRel[rel.ID] {
			if !usedP[p] {
				landCands = append(landCands, p)
			}
		}
		if len(landCands) > 0 {
			p := landCands[r.Int(0, len(landCands)-1)]
			usedP[p] = true
			reg := w.RegionName(p)
			nm := Pick(r, wildPrefixes) + god
			if saintName != "" && r.Chance(0.5) {
				nm = Pick(r, wildPrefixes) + saintName
			}
			out = append(out, &fpSite{
				ID:      fmt.Sprintf("fp:%d:p%d", rel.ID, p),
				Name:    nm + " (" + reg + ")",
				FaithID: rel.ID,
				P:       p,
				WX:      p % w.W,
				WY:      p / w.W,
				Kind:    "sanctuary",
				Desc:    fmt.Sprintf("A lonely pilgrimage sanctuary of %s standing in %s.", rel.Name, reg),
			})
		}
	}
	return out
}

func (g *Game) fpSiteByID(id string) *fpSite {
	g.fpHolySites()
	return g.fp().sitesByID[id]
}

// ---- local faith & altar detection ----

func (g *Game) fpLocalFaith() *ReligionData {
	if g.W == nil {
		return nil
	}
	if b := g.localBurg(); b != nil {
		if b.P >= 0 && b.P < len(g.W.Religion) {
			if r := g.W.relByID[g.W.Religion[b.P]]; r != nil {
				return r
			}
		}
		if st := g.W.stateByID[b.State]; st != nil {
			if r := g.W.relByID[st.Religion]; r != nil {
				return r
			}
		}
	}
	if g.Cur != nil && g.W.In(g.Cur.WX, g.Cur.WY) {
		p := g.W.Idx(g.Cur.WX, g.Cur.WY)
		if p >= 0 && p < len(g.W.Religion) {
			if r := g.W.relByID[g.W.Religion[p]]; r != nil {
				return r
			}
		}
	}
	return g.W.relByID[g.HomeFaith]
}

func (g *Game) fpNearAltar() bool {
	if g.Cur == nil || g.P == nil {
		return false
	}
	for dy := -1; dy <= 1; dy++ {
		for dx := -1; dx <= 1; dx++ {
			x, y := g.P.X+dx, g.P.Y+dy
			if g.Cur.In(x, y) && g.Cur.At(x, y) == TAltar {
				return true
			}
		}
	}
	return false
}

func (g *Game) fpNearClergy() *Actor {
	if g.Cur == nil || g.P == nil {
		return nil
	}
	for _, a := range g.Cur.Actors {
		if a != nil && a.HP > 0 && rtIsClergy(a) && !g.Hostile(a) && maxi(abs(a.X-g.P.X), abs(a.Y-g.P.Y)) <= 4 {
			return a
		}
	}
	return nil
}

// ---- zone, enter, side hooks ----

func fpZone(g *Game, zn *Zone, _ *Rng) {
	if g.W == nil || zn == nil || zn.Z != 0 || zn.Underwater {
		return
	}
	if !g.W.In(zn.WX, zn.WY) {
		return
	}
	p := g.W.Idx(zn.WX, zn.WY)
	if p < 0 || p >= len(g.W.Religion) {
		return
	}

	// Ensure named holy sites always have a grand shrine altar in their central zone.
	if zn.ZX == 1 && zn.ZY == 1 {
		g.fpHolySites()
		if sites := g.fp().sitesByP[p]; len(sites) > 0 {
			site := sites[0]
			rel := g.W.relByID[site.FaithID]
			bl := fpBlessingFor(g.W, rel)
			r := NewRng(g.W.Seed + ":fp:holy:" + site.ID + ":" + zn.Key)
			if x, y, ok := zn.FreeSpot(r, func(x, y int) bool {
				t := zn.At(x, y)
				return (t == TFloor || t == TCobble || t == TGrass || t == TDirt) && x > 3 && x < MW-4 && y > 2 && y < MH-3
			}); ok {
				zn.Set(x, y, TAltar)
				if zn.Notes == nil {
					zn.Notes = map[int]string{}
				}
				zn.Notes[y*MW+x] = fmt.Sprintf("%s. %s (Open Faiths & Shrines to make an offering for %s.)", site.Name, bl.Quote, bl.Title)
			}
			return
		}
	}

	// Wayside shrines in ~12% of non-town surface land zones belonging to a faith.
	if zn.Kind == "town" || g.W.IsWater(p) {
		return
	}
	rid := g.W.Religion[p]
	rel := g.W.relByID[rid]
	if rel == nil {
		return
	}
	if HashStr(g.W.Seed+":fp:ws:"+zn.Key)%100 >= 12 {
		return
	}
	r := NewRng(g.W.Seed + ":fp:shrine:" + zn.Key)
	x, y, ok := zn.FreeSpot(r, func(x, y int) bool {
		t := zn.At(x, y)
		return (t == TGrass || t == TDirt || t == TSand || t == TRoad) && x > 4 && x < MW-5 && y > 3 && y < MH-4
	})
	if !ok {
		return
	}
	bl := fpBlessingFor(g.W, rel)
	zn.Set(x, y, TAltar)
	if zn.Notes == nil {
		zn.Notes = map[int]string{}
	}
	zn.Notes[y*MW+x] = fmt.Sprintf("A wayside shrine of %s (%s). \"%s\" (Open Faiths & Shrines beside it to make an offering.)", rel.Name, bl.Title, bl.Quote)
}

func fpEnter(g *Game, zn *Zone) {
	if g.W == nil || g.P == nil || zn == nil || zn.Z != 0 {
		return
	}
	if !g.W.In(zn.WX, zn.WY) {
		return
	}
	p := g.W.Idx(zn.WX, zn.WY)
	st := g.fp()

	// Check pilgrimage arrival and discovery of holy sites.
	g.fpHolySites()
	for _, site := range st.sitesByP[p] {
		st.KnownSites[site.ID] = true
		if st.ActiveVow == site.ID && !st.Visited[site.ID] {
			g.fpCompletePilgrimage(site)
		} else if !st.Visited[site.ID] {
			g.Msg(fmt.Sprintf("{{Y|You arrive at %s, a holy site of %s.}}", site.Name, g.fpRelName(site.FaithID)))
		}
	}

	// Check for heresy summons in towns whose state religion views the player as hostile/heretic.
	if zn.Kind == "town" && zn.Burg > 0 {
		bd := g.W.burgByID[zn.Burg]
		if bd == nil || bd.State <= 0 {
			return
		}
		state := g.W.stateByID[bd.State]
		if state == nil || state.Religion <= 0 {
			return
		}
		rel := g.W.relByID[state.Religion]
		if rel == nil {
			return
		}
		if g.fpUnderSuspicion(state, rel) && st.Summoned[rel.ID] != g.Day()+1 {
			st.Summoned[rel.ID] = g.Day() + 1
			g.Msg(fmt.Sprintf("{{R|Clerics of %s watch you closely in %s. Speak to a priest to answer a heresy summons, or keep your head down.}}", rel.Name, bd.Name))
		}
	}
}

func fpSide(g *Game) []string {
	if g.W == nil || g.P == nil || g.OnWorld {
		return nil
	}
	st := g.fp()
	var out []string
	if st.ActiveVow != "" {
		if site := g.fpSiteByID(st.ActiveVow); site != nil && !st.Visited[site.ID] {
			out = append(out, fmt.Sprintf("{{Y|Pilgrimage:}} %s {{K|(%s)}}", fpShort(site.Name, 18), g.cxDir(site.P)))
		}
	}
	return out
}

func (g *Game) fpRelName(id int) string {
	if g.W != nil {
		if r := g.W.relByID[id]; r != nil && r.Name != "" {
			return r.Name
		}
	}
	return "an old faith"
}

func fpShort(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n-1]) + "…"
}

// ---- blessings & offerings ----

func (g *Game) fpClearBlessings() {
	if g.P == nil || g.P.Status == nil {
		return
	}
	for _, id := range fpAllBlessings {
		delete(g.P.Status, id)
	}
}

func (g *Game) fpApplyBlessing(rel *ReligionData, dur int) fpBlessInfo {
	bl := fpBlessingFor(g.W, rel)
	g.fpClearBlessings()
	g.P.AddSt(bl.StatusID, dur)
	g.fp().ActiveName = bl.Title
	return bl
}

func (g *Game) fpOfferCoin(rel *ReligionData, cost int) (bool, string) {
	if rel == nil {
		return false, "There is no named faith here."
	}
	st := g.fp()
	if st.OfferDay[rel.ID] == g.Day()+1 {
		return false, fmt.Sprintf("You have already made an offering to %s today.", rel.Name)
	}
	if st.Heretic[rel.ID] {
		cost *= 2
	}
	if !g.cuSpend(cost) {
		return false, fmt.Sprintf("You need %s for the offering.", g.cuFormat(cost))
	}
	st.OfferDay[rel.ID] = g.Day() + 1
	dur := 720
	if g.HomeFaith == rel.ID {
		dur = 1080
	}
	bl := g.fpApplyBlessing(rel, dur)
	g.AddRep(fmt.Sprintf("faith:%d", rel.ID), 5, " for a shrine offering")
	g.crKarma(4, 1, "made an offering at a shrine")
	g.spendTurns(5)
	msg := fmt.Sprintf("You leave %s at the shrine of %s. {{G|%s (%s) settles upon you.}}", g.cuFormat(cost), rel.Name, bl.Title, bl.Summary)
	g.Msg(msg)
	return true, msg
}

func (g *Game) fpFindOfferItem() *Item {
	for _, it := range g.P.Inv {
		if it == nil || uwIsStolen(it) || it.Value < 0 {
			continue
		}
		switch it.Kind {
		case "food", "dish", "ingredient", "tonic", "trophy":
			return it
		}
	}
	return nil
}

func (g *Game) fpOfferItem(rel *ReligionData, it *Item) (bool, string) {
	if rel == nil || it == nil {
		return false, "You have nothing suitable to offer."
	}
	st := g.fp()
	if st.OfferDay[rel.ID] == g.Day()+1 {
		return false, fmt.Sprintf("You have already made an offering to %s today.", rel.Name)
	}
	name := it.Name
	rtTake(g, it)
	st.OfferDay[rel.ID] = g.Day() + 1
	dur := 960
	if g.HomeFaith == rel.ID {
		dur = 1440
	}
	bl := g.fpApplyBlessing(rel, dur)
	g.AddRep(fmt.Sprintf("faith:%d", rel.ID), 8, " for a devoted offering")
	g.crKarma(4, 2, "offered goods at a shrine")
	g.spendTurns(5)
	msg := fmt.Sprintf("You lay %s upon the shrine of %s. {{G|%s (%s) warms your spirit.}}", name, rel.Name, bl.Title, bl.Summary)
	g.Msg(msg)
	return true, msg
}

// ---- pilgrimages ----

func (g *Game) fpVowPilgrimage(site *fpSite) string {
	if site == nil {
		return "No holy site is named."
	}
	st := g.fp()
	if st.Visited[site.ID] {
		return fmt.Sprintf("You have already completed the pilgrimage to %s.", site.Name)
	}
	st.ActiveVow = site.ID
	st.KnownSites[site.ID] = true
	if g.W.In(site.WX, site.WY) {
		g.Known[site.P] = true
	}
	msg := fmt.Sprintf("You vow a pilgrimage to %s (%s).", site.Name, g.cxDir(site.P))
	g.Msg("{{Y|" + msg + "}}")
	return msg
}

func (g *Game) fpCompletePilgrimage(site *fpSite) {
	st := g.fp()
	st.Visited[site.ID] = true
	if st.ActiveVow == site.ID {
		st.ActiveVow = ""
	}
	rel := g.W.relByID[site.FaithID]
	bl := g.fpApplyBlessing(rel, DayLen*2)
	g.AddRep(fmt.Sprintf("faith:%d", site.FaithID), 25, " for completing a pilgrimage")
	if site.P >= 0 && site.P < len(g.W.State) {
		if sid := g.W.State[site.P]; sid > 0 {
			if sd := g.W.stateByID[sid]; sd != nil && sd.Religion == site.FaithID {
				g.AddRep(fmt.Sprintf("state:%d", sid), 10, " for a pious pilgrimage")
			}
		}
	}
	g.gainXP(35 + g.P.Level*5)
	g.crKarma(4, 3, "completed a pilgrimage")
	g.deed("faith", fmt.Sprintf("%s completed a pilgrimage to %s.", g.P.Name, site.Name), 0, 0)
	g.Msg(fmt.Sprintf("{{G|Pilgrimage complete: %s!}} {{Y|%s (%s) blesses you for two days.}}", site.Name, bl.Title, bl.Summary))
}

// ---- conversions & heresy trials ----

func (g *Game) fpConvert(rel *ReligionData) string {
	if rel == nil || rel.ID == g.HomeFaith {
		return "You already keep this faith."
	}
	oldID := g.HomeFaith
	g.HomeFaith = rel.ID
	st := g.fp()
	st.Converted = append(st.Converted, rel.ID)
	if len(st.Converted) > 24 {
		st.Converted = st.Converted[len(st.Converted)-24:]
	}
	delete(st.Heretic, rel.ID)

	g.AddRep(fmt.Sprintf("faith:%d", rel.ID), 40, " for embracing "+rel.Name)
	if oldID > 0 {
		g.AddRep(fmt.Sprintf("faith:%d", oldID), -35, " for leaving "+g.fpRelName(oldID))
	}
	if b := g.localBurg(); b != nil && b.State > 0 {
		if sd := g.W.stateByID[b.State]; sd != nil {
			if sd.Religion == rel.ID {
				g.AddRep(fmt.Sprintf("state:%d", sd.ID), 15, " for joining the realm's faith")
			} else if sd.Religion == oldID {
				g.AddRep(fmt.Sprintf("state:%d", sd.ID), -20, " for forsaking the realm's faith")
			}
		}
	}
	bl := g.fpApplyBlessing(rel, DayLen)
	g.deed("faith", fmt.Sprintf("%s converted to %s.", g.P.Name, rel.Name), 0, 0)
	msg := fmt.Sprintf("You take the vows of %s. {{G|%s (%s) marks your conversion.}}", rel.Name, bl.Title, bl.Summary)
	g.Msg(msg)
	return msg
}

func (g *Game) fpUnderSuspicion(state *StateData, rel *ReligionData) bool {
	if rel == nil {
		return false
	}
	st := g.fp()
	if st.Heretic[rel.ID] {
		return true
	}
	if g.Rep[fmt.Sprintf("faith:%d", rel.ID)] <= -40 {
		return true
	}
	if state != nil {
		for _, p := range state.Persecutions {
			if p.Ended == 0 && p.Axis == "faith" && p.Target == g.HomeFaith && rel.ID != g.HomeFaith {
				return true
			}
		}
	}
	return false
}

func (g *Game) fpBeginTrial(a *Actor, rel *ReligionData) {
	if rel == nil {
		return
	}
	stateID := 0
	if b := g.localBurg(); b != nil {
		stateID = b.State
	}
	charge := fmt.Sprintf("holding doctrines contrary to %s", rel.Name)
	if rel.Lore != nil && len(rel.Lore.Heresies) > 0 {
		h := rel.Lore.Heresies[int(HashStr(fmt.Sprintf("%s:fp:her:%d:%d", g.W.Seed, rel.ID, g.Day()))%uint64(len(rel.Lore.Heresies)))]
		charge = fmt.Sprintf("sympathy with %s (%s)", h.Name, rtFirstSentence(h.Claim))
	}
	fine := 35 + g.P.Level*8
	intro := fmt.Sprintf("{{R|Heresy Trial of %s}}\n%s reads the charge before witnesses: you are accused of %s.\n\nHow do you answer?", rel.Name, g.The(a), charge)
	g.rtScene(a, intro,
		Option{
			Text: fmt.Sprintf("Recant and pay penance (%s).", g.cuFormat(fine)),
			Do:   func() { g.fpResolveTrial(a, rel, stateID, "recant", fine) },
		},
		Option{
			Text: fmt.Sprintf("%s Argue your orthodoxy from scripture and law.", g.checkLabel("lore", dcMedium+2)),
			Do:   func() { g.fpResolveTrial(a, rel, stateID, "lore", 0) },
		},
		Option{
			Text: fmt.Sprintf("%s Appeal to the crowd's mercy and good sense.", g.checkLabel("persuasion", dcMedium+1)),
			Do:   func() { g.fpResolveTrial(a, rel, stateID, "persuade", 0) },
		},
		Option{
			Text: "{{R|Stand defiant and reject the synod's authority.}}",
			Do:   func() { g.fpResolveTrial(a, rel, stateID, "defy", 0) },
		},
	)
}

func (g *Game) fpResolveTrial(a *Actor, rel *ReligionData, stateID int, choice string, fine int) {
	st := g.fp()
	g.spendTurns(20)
	fk := fmt.Sprintf("faith:%d", rel.ID)
	sk := ""
	if stateID > 0 {
		sk = fmt.Sprintf("state:%d", stateID)
	}
	record := func(outcome, note string) {
		st.Trials = append(st.Trials, fpTrialRec{FaithID: rel.ID, StateID: stateID, Day: g.Day() + 1, Outcome: outcome, Note: note})
		if len(st.Trials) > 20 {
			st.Trials = st.Trials[len(st.Trials)-20:]
		}
	}

	switch choice {
	case "recant":
		if !g.cuSpend(fine) {
			g.talkMenu("\"You cannot even pay the penance. Return when your purse can back your contrition.\"")
			return
		}
		delete(st.Heretic, rel.ID)
		g.AddRep(fk, 20, " for recanting before the synod")
		if sk != "" {
			g.AddRep(sk, 10, " for submitting to the court")
		}
		record("recanted", fmt.Sprintf("Recanted before %s and paid %s.", rel.Name, g.cuFormat(fine)))
		g.talkMenu(fmt.Sprintf("You sign the recantation and set down %s. The clerics strike your name from the heresy roll.", g.cuFormat(fine)))
	case "lore", "persuade":
		kind, dc := "lore", dcMedium+2
		if choice == "persuade" {
			kind, dc = "persuasion", dcMedium+1
		}
		if g.roll2d6Check(g.checkMod(kind), dc) {
			delete(st.Heretic, rel.ID)
			g.AddRep(fk, 15, " for winning acquittal at trial")
			if sk != "" {
				g.AddRep(sk, 8, " for an eloquent defence")
			}
			g.gainXP(25)
			g.deed("faith", fmt.Sprintf("%s was acquitted at a heresy trial of %s.", g.P.Name, rel.Name), 0, 0)
			record("acquitted", fmt.Sprintf("Acquitted by %s before %s.", rel.Name, a.Name))
			g.talkMenu(fmt.Sprintf("{{G|[%s: Success]}} Your answer silences the accusers. The synod clears you of heresy.", checkKinds[kind].label))
			return
		}
		st.Heretic[rel.ID] = true
		g.AddRep(fk, -30, " for failing a heresy trial")
		if sk != "" {
			g.AddRep(sk, -20, " for condemnation as a heretic")
			if g.uwLawsFor(stateID).Heresy {
				g.uwAddBounty(stateID, 40, " for heresy")
			}
		}
		record("condemned", fmt.Sprintf("Condemned as a heretic by %s.", rel.Name))
		g.talkMenu(fmt.Sprintf("{{R|[%s: Failure]}} The synod finds your defence wanting and brands you a heretic of %s.", checkKinds[kind].label, rel.Name))
	case "defy":
		st.Heretic[rel.ID] = true
		g.AddRep(fk, -40, " for defying the synod")
		if sk != "" {
			g.AddRep(sk, -25, " for defying the state faith")
			if g.uwLawsFor(stateID).Heresy {
				g.uwAddBounty(stateID, 55, " for open heresy")
			}
		}
		// Rival cults and opposing faiths respect defiance.
		for _, o := range g.Orders() {
			if o.Kind == "cult" && (o.Faith == rel.ID || o.Faith == 0) {
				g.AddRep(o.RepKey(), 25, " for defying the orthodox synod")
				break
			}
		}
		g.gainXP(20)
		g.deed("faith", fmt.Sprintf("%s defied a heresy trial of %s.", g.P.Name, rel.Name), 0, 0)
		record("defiant", fmt.Sprintf("Openly defied the synod of %s.", rel.Name))
		g.talkMenu(fmt.Sprintf("{{R|You reject the court outright.}} The clerics pronounce you heretic, though dissidents in the crowd remember your name."))
	}
}

// ---- talk hook ----

func fpTalk(g *Game, a *Actor) []Option {
	if g.W == nil || g.P == nil || a == nil || a.HP <= 0 || a.AI == "follower" || g.Hostile(a) || !rtIsClergy(a) {
		return nil
	}
	rel := g.rtFaithOf(a)
	if rel == nil {
		return nil
	}
	var out []Option
	bl := fpBlessingFor(g.W, rel)
	cost := 15 + g.P.Level*3

	out = append(out, Option{
		Text: fmt.Sprintf("Make a shrine offering to %s for %s (%s).", rel.Name, bl.Title, g.cuFormat(cost)),
		Do: func() {
			_, msg := g.fpOfferCoin(rel, cost)
			g.talkMenu(msg)
		},
	})

	// Pilgrimage lead for this faith.
	for _, site := range g.fpHolySites() {
		if site.FaithID == rel.ID && !g.fp().Visited[site.ID] {
			s := site
			out = append(out, Option{
				Text: fmt.Sprintf("Vow a pilgrimage to %s.", s.Name),
				Do: func() {
					msg := g.fpVowPilgrimage(s)
					g.talkMenu(msg + " " + s.Desc)
				},
			})
			break
		}
	}

	if rel.ID != g.HomeFaith {
		out = append(out, Option{
			Text: fmt.Sprintf("Ask to convert to %s.", rel.Name),
			Do: func() {
				msg := g.fpConvert(rel)
				g.talkMenu(msg)
			},
		})
	}

	var state *StateData
	if b := g.localBurg(); b != nil {
		state = g.W.stateByID[b.State]
	}
	if g.fpUnderSuspicion(state, rel) {
		out = append(out, Option{
			Text: "{{R|Answer the synod's heresy summons.}}",
			Do:   func() { g.fpBeginTrial(a, rel) },
		})
	} else {
		out = append(out, Option{
			Text: "Challenge their orthodoxy (risk a heresy trial).",
			Do:   func() { g.fpBeginTrial(a, rel) },
		})
	}
	return out
}

// ---- screen ----

func (g *Game) fpRelList() []*ReligionData {
	if g.W == nil {
		return nil
	}
	var out []*ReligionData
	for i := range g.W.Religions {
		if g.W.Religions[i].ID > 0 && g.W.Religions[i].Name != "" {
			out = append(out, &g.W.Religions[i])
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func fpKeys(g *Game, k string) {
	st := g.fp()
	switch k {
	case "Escape", "q":
		g.Mode = "main"
		return
	case "Tab", "ArrowRight":
		st.tab = (st.tab + 1) % 3
		st.sel = 0
		return
	case "ArrowLeft":
		st.tab = (st.tab + 2) % 3
		st.sel = 0
		return
	case "ArrowUp", "N8", "k":
		st.sel = maxi(0, st.sel-1)
		return
	case "ArrowDown", "N2", "j":
		st.sel++
		return
	}
	switch st.tab {
	case 0:
		if k == "Enter" || k == "o" {
			if !g.fpNearAltar() && g.fpNearClergy() == nil {
				g.Msg("Stand beside a shrine altar or speak with clergy to make an offering.")
				return
			}
			rel := g.fpLocalFaith()
			cost := 15 + g.P.Level*3
			g.fpOfferCoin(rel, cost)
		} else if k == "i" {
			if !g.fpNearAltar() && g.fpNearClergy() == nil {
				g.Msg("Stand beside a shrine altar or speak with clergy to offer an item.")
				return
			}
			rel := g.fpLocalFaith()
			it := g.fpFindOfferItem()
			if it == nil {
				g.Msg("You carry no food, tonic, or trophy fit for an offering.")
				return
			}
			g.fpOfferItem(rel, it)
		}
	case 1:
		sites := g.fpHolySites()
		if len(sites) > 0 && (k == "Enter" || k == "v") {
			idx := clampi(st.sel, 0, len(sites)-1)
			g.fpVowPilgrimage(sites[idx])
		}
	}
}

func (g *Game) fpDraw() {
	st := g.fp()
	sc := g.sc
	x, y, w, h := g.panel("Faiths and shrines")
	bottom := y + h - 1
	tabCol := func(on bool) string {
		if on {
			return "W"
		}
		return "K"
	}
	sc.TextB(x, y, fmt.Sprintf("{{%s|[ Shrines & Faiths ]}}  {{%s|[ Pilgrimages ]}}  {{%s|[ Trials & Standing ]}}",
		tabCol(st.tab == 0), tabCol(st.tab == 1), tabCol(st.tab == 2)), CGray, CBgDark, x+w)
	y += 2

	put := func(s string) {
		for _, ln := range Wrap(s, w) {
			if y < bottom {
				sc.TextB(x, y, ln, CGray, CBgDark, x+w)
				y++
			}
		}
	}

	switch st.tab {
	case 0:
		rels := g.fpRelList()
		local := g.fpLocalFaith()
		if g.fpNearAltar() && local != nil {
			bl := fpBlessingFor(g.W, local)
			put(fmt.Sprintf("{{G|At a shrine of %s:}} %s (%s). Press {{W|Enter}} to offer coin or {{W|i}} to offer goods.", local.Name, bl.Title, bl.Summary))
			y++
		}
		if len(rels) == 0 {
			put("No organised faiths are recorded in this world.")
			break
		}
		st.sel = clampi(st.sel, 0, len(rels)-1)
		maxShow := mini(len(rels), 8)
		start := 0
		if st.sel >= maxShow {
			start = st.sel - maxShow + 1
		}
		for i := start; i < mini(len(rels), start+maxShow); i++ {
			r := rels[i]
			mark := "  "
			if i == st.sel {
				mark = "> "
			}
			tag := ""
			if r.ID == g.HomeFaith {
				tag = " {{G|[your faith]}}"
			}
			if st.Heretic[r.ID] {
				tag += " {{R|[heretic]}}"
			}
			rep := g.Rep[fmt.Sprintf("faith:%d", r.ID)]
			put(fmt.Sprintf("%s{{W|%s}} {{K|(rep %d)}}%s", mark, r.Name, rep, tag))
		}
		y++
		sel := rels[st.sel]
		bl := fpBlessingFor(g.W, sel)
		put(fmt.Sprintf("{{Y|%s}} - Deity: %s (%s)", sel.Name, nonEmpty(sel.Deity, "unnamed"), nonEmpty(sel.Type, "folk faith")))
		if sel.Theology != "" || sel.Doctrine != "" {
			put(fmt.Sprintf("Theology: %s. Doctrine: %s.", nonEmpty(sel.Theology, "unwritten"), nonEmpty(sel.Doctrine, "kept by memory")))
		}
		put(fmt.Sprintf("Shrine blessing: {{G|%s}} (%s). Preferred offering: %s.", bl.Title, bl.Summary, bl.Offer))
		put("{{K|\"" + bl.Quote + "\"}}")
	case 1:
		sites := g.fpHolySites()
		if len(sites) == 0 {
			put("No pilgrimage sites exist on this planet.")
			break
		}
		st.sel = clampi(st.sel, 0, len(sites)-1)
		maxShow := mini(len(sites), 10)
		start := 0
		if st.sel >= maxShow {
			start = st.sel - maxShow + 1
		}
		for i := start; i < mini(len(sites), start+maxShow); i++ {
			s := sites[i]
			mark := "  "
			if i == st.sel {
				mark = "> "
			}
			status := "{{K|unvisited}}"
			if st.Visited[s.ID] {
				status = "{{G|completed}}"
			} else if st.ActiveVow == s.ID {
				status = "{{Y|vowed}}"
			}
			put(fmt.Sprintf("%s{{W|%s}} %s {{K|%s}}", mark, s.Name, status, g.cxDir(s.P)))
		}
		y++
		s := sites[st.sel]
		put("{{Y|" + s.Name + "}} (" + g.fpRelName(s.FaithID) + ")")
		put(s.Desc)
		put("Press {{W|Enter}} to vow a pilgrimage here (reveals its parasang on the world map).")
	case 2:
		put(fmt.Sprintf("Current faith: {{Y|%s}}", g.fpRelName(g.HomeFaith)))
		if st.ActiveName != "" {
			put("Last shrine blessing: {{G|" + st.ActiveName + "}}")
		}
		if len(st.Converted) > 0 {
			put(fmt.Sprintf("Conversions made: %d", len(st.Converted)))
		}
		y++
		put("{{W|Heresy trials & synod records}}")
		if len(st.Trials) == 0 {
			put("{{K|No trials recorded. Speak with clergy to answer a summons or challenge orthodoxy.}}")
		} else {
			for i := len(st.Trials) - 1; i >= 0; i-- {
				tr := st.Trials[i]
				put(fmt.Sprintf("Day %d [%s]: %s", tr.Day, tr.Outcome, tr.Note))
			}
		}
	}
	sc.TextB(x, bottom, "{{K|Up/Down choose   Tab switch   Enter act   Esc close}}", CGray, CBgDark, x+w)
}
