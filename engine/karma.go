package engine

import (
	"fmt"
	"sort"
	"strings"
)

// Karma, oaths and vows (prefix cr). Five axes -100..100 move with crossroads choices (crossroads.go) and with
// observable deeds (kills, spared foes, thefts, favour, gifts). Vows are sworn to priests and elders and give a
// bonus while kept and a lasting penalty when broken. State: g.ext("crk") (global: follows the wanderer).
// Ledger screen: mode `crk`, opened from the Abilities entry `cr_ledger` and from vow talk.

var crAxisNames = [5][2]string{{"Mercy", "Cruelty"}, {"Honesty", "Deceit"}, {"Courage", "Caution"}, {"Generosity", "Greed"}, {"Piety", "Defiance"}}
var crAxisKeys = map[string]int{"mer": 0, "hon": 1, "cou": 2, "gen": 3, "pie": 4}

type crLog struct {
	Day  int    `json:"d"`
	Text string `json:"t"`
}

type crVow struct {
	ID    string `json:"id"`
	Arg   string `json:"a,omitempty"`
	Since int    `json:"s"`
	Term  int    `json:"tm,omitempty"` // day it is fulfilled (0 = by deed)
	Dead  int    `json:"dl,omitempty"` // day it lapses unfulfilled (pilgrimage)
	Prog  int    `json:"p,omitempty"`
	Need  int    `json:"n,omitempty"`
	Over  int    `json:"o,omitempty"`
	To    string `json:"to,omitempty"`
	Faith int    `json:"f,omitempty"`
	Seen  []int  `json:"sn,omitempty"`
}

type crKState struct {
	Ax     [5]int          `json:"ax"`
	Log    []crLog         `json:"lg,omitempty"`
	Snap   map[string]int  `json:"sp,omitempty"`
	Vows   []*crVow        `json:"v,omitempty"`
	Kept   int             `json:"k,omitempty"`
	Broke  int             `json:"b,omitempty"`
	Op     map[string]int  `json:"op,omitempty"`
	Firsts map[string]bool `json:"fs,omitempty"`
	Hist   []crLog         `json:"h,omitempty"` // crossroads history
}

func crKs(g *Game) *crKState {
	s := g.ext("crk", func() any { return &crKState{} }).(*crKState)
	if s.Snap == nil {
		s.Snap = map[string]int{}
	}
	if s.Op == nil {
		s.Op = map[string]int{}
	}
	if s.Firsts == nil {
		s.Firsts = map[string]bool{}
	}
	return s
}

// crTrait is the karma value of an axis (0 mercy, 1 honesty, 2 courage, 3 generosity, 4 piety).
func (g *Game) crTrait(axis int) int {
	if g == nil || axis < 0 || axis > 4 {
		return 0
	}
	return crKs(g).Ax[axis]
}

// KarmaDivineMod shifts divine favour gains: -5..+5 (pious and merciful lean up, defiant and cruel down).
func (g *Game) KarmaDivineMod() int {
	if g == nil || g.P == nil {
		return 0
	}
	s := crKs(g)
	return clampi(s.Ax[4]/20+s.Ax[0]/50, -5, 5)
}

func (g *Game) crFirst(key, deed string) {
	s := crKs(g)
	if s.Firsts[key] || g.Cur == nil {
		return
	}
	s.Firsts[key] = true
	g.deed("event", deed, 0, 0)
}

// crKarma moves an axis; why is shown in the ledger log.
func (g *Game) crKarma(ax, n int, why string) {
	if n == 0 || ax < 0 || ax > 4 {
		return
	}
	s := crKs(g)
	old := s.Ax[ax]
	s.Ax[ax] = clampi(old+n, -100, 100)
	if why != "" {
		s.Log = append(s.Log, crLog{g.Day(), fmt.Sprintf("%s %+d: %s", crAxisNames[ax][boolInt(n < 0)], n, why)})
		if len(s.Log) > 30 {
			s.Log = s.Log[len(s.Log)-30:]
		}
	}
	if (old < 50 && s.Ax[ax] >= 50) || (old > -50 && s.Ax[ax] <= -50) {
		g.Msg("{{M|Your name is becoming known: " + g.crEpithet() + ".}}")
	}
}

var crEpi = [5][2][3]string{
	{{"the Gentle", "the Merciful", "Saint of Small Kindnesses"}, {"the Harsh", "the Cruel", "the Scourge"}},
	{{"the Candid", "the Honest", "Truth-Teller"}, {"the Smooth", "the Liar", "Father of Lies"}},
	{{"the Bold", "the Brave", "the Lionheart"}, {"the Careful", "the Cautious", "the Ghost of the Road"}},
	{{"the Open-Handed", "the Generous", "the Bountiful"}, {"the Close-Fisted", "the Grasping", "the Dragon-Hoarder"}},
	{{"the Devout", "the Pious", "the Faith-Burdened"}, {"the Doubter", "the Defiant", "the Godless"}},
}

func (g *Game) crEpithet() string {
	s := crKs(g)
	best, bi := 0, -1
	for i, v := range s.Ax {
		if abs(v) > best {
			best, bi = abs(v), i
		}
	}
	if bi < 0 || best < 15 {
		return "the Unwritten"
	}
	lv := 0
	if best >= 40 {
		lv = 1
	}
	if best >= 75 {
		lv = 2
	}
	return crEpi[bi][boolInt(s.Ax[bi] < 0)][lv]
}

// ---- statuses (passive perks at the extremes, vow bonuses) ----

type crPerk struct {
	ax   int
	neg  bool
	def  *skStatusDef
	desc string
}

var crPerks []crPerk
var crVowDefs []*crVowDef

type crVowDef struct {
	ID, Name, Desc, Perk string
	Days                 int
	St                   *skStatusDef
}

func init() {
	skAddStatus(&skStatusDef{ID: "crvow", Name: "Sworn", Glyph: 'o', WIL: 1})
	for _, p := range []crPerk{
		{0, false, &skStatusDef{ID: "crx0p", Name: "Gentle soul", Glyph: 'g', MA: 2, Hit: -1}, "Mental armor +2, to-hit -1"},
		{0, true, &skStatusDef{ID: "crx0n", Name: "Dread", Glyph: 'd', EGO: 3, WIL: -2}, "Ego +3, will -2"},
		{1, false, &skStatusDef{ID: "crx1p", Name: "Candour", Glyph: 'c', EGO: 2}, "Ego +2"},
		{1, true, &skStatusDef{ID: "crx1n", Name: "Slippery", Glyph: 's', AGI: 2, WIL: -2}, "Agility +2, will -2"},
		{2, false, &skStatusDef{ID: "crx2p", Name: "Lionheart", Glyph: 'l', Hit: 2, MA: 2}, "To-hit +2, mental armor +2"},
		{2, true, &skStatusDef{ID: "crx2n", Name: "Wary", Glyph: 'w', DV: 2, Speed: -5}, "DV +2, speed -5"},
		{3, false, &skStatusDef{ID: "crx3p", Name: "Open hand", Glyph: 'h', EGO: 2}, "Ego +2"},
		{3, true, &skStatusDef{ID: "crx3n", Name: "Grasping", Glyph: 'r', Hit: 1, EGO: -1}, "To-hit +1, ego -1"},
		{4, false, &skStatusDef{ID: "crx4p", Name: "Blessed", Glyph: 'b', MA: 3}, "Mental armor +3"},
		{4, true, &skStatusDef{ID: "crx4n", Name: "Godless", Glyph: 'u', WIL: 2, MA: -2}, "Will +2, mental armor -2"},
	} {
		skAddStatus(p.def)
		crPerks = append(crPerks, p)
	}
	crVowDefs = []*crVowDef{
		{"nv", "Non-violence", "Never take a life: no person may die by your hand.", "Mental armor +3, will +1; faith rep creeps up", 40, &skStatusDef{ID: "crvow_nv", Name: "Vow of Non-violence", Glyph: 'n', WIL: 1, MA: 3}},
		{"po", "Poverty", "Never carry more than a few coins.", "Speed +10, DV +1", 40, &skStatusDef{ID: "crvow_po", Name: "Vow of Poverty", Glyph: 'p', Speed: 10, DV: 1}},
		{"tr", "Truth", "Never lie: no deceit options at all.", "Ego +2, mental armor +1", 60, &skStatusDef{ID: "crvow_tr", Name: "Vow of Truth", Glyph: 't', EGO: 2, MA: 1}},
		{"ho", "Hospitality", "Never turn away one in need: aid five people, refuse none.", "Ego +1, will +1", 0, &skStatusDef{ID: "crvow_ho", Name: "Vow of Hospitality", Glyph: 'H', EGO: 1, WIL: 1}},
		{"ve", "Vengeance", "Hunt a named enemy to the end. No truce, no mercy.", "To-hit +2, will +1", 0, &skStatusDef{ID: "crvow_ve", Name: "Vow of Vengeance", Glyph: 'V', Hit: 2, WIL: 1}},
		{"pi", "Pilgrimage", "Visit several towns before time runs out.", "Speed +15", 0, &skStatusDef{ID: "crvow_pi", Name: "Vow of Pilgrimage", Glyph: 'P', Speed: 15}},
		{"si", "Silence", "Keep silent on a topic: no talk options about it.", "Will +2, mental armor +2", 40, &skStatusDef{ID: "crvow_si", Name: "Vow of Silence", Glyph: 'q', WIL: 2, MA: 2}},
	}
	for _, d := range crVowDefs {
		skAddStatus(d.St)
	}
	unLocalKeys = append(unLocalKeys, "cr")
	killHooks = append(killHooks, crKillHook)
	turnHooks = append(turnHooks, crKarmaTurn)
	enterHooks = append(enterHooks, crVowEnter)
	talkHooks = append(talkHooks, crKarmaTalk)
	modeKeys["crk"] = crKeys
	modeDraws["crk"] = func(g *Game) { g.crDrawLedger() }
}

func crVowDefOf(id string) *crVowDef {
	for _, d := range crVowDefs {
		if d.ID == id {
			return d
		}
	}
	return nil
}

func (g *Game) crVow(id string) *crVow {
	for _, v := range crKs(g).Vows {
		if v.ID == id {
			return v
		}
	}
	return nil
}

func (g *Game) crVowName(v *crVow) string {
	d := crVowDefOf(v.ID)
	n := d.Name
	switch v.ID {
	case "ve":
		n += " on " + v.Arg
	case "si":
		n += " on " + v.Arg
	case "po":
		n += fmt.Sprintf(" (max %s %s)", v.Arg, g.Sk.Currency)
	case "pi":
		n += fmt.Sprintf(" (%d/%d)", v.Prog, v.Need)
	case "ho":
		n += fmt.Sprintf(" (%d/%d)", v.Prog, v.Need)
	}
	return n
}

// ---- breaking and fulfilling ----

func (g *Game) crRemoveVow(v *crVow) {
	s := crKs(g)
	for i, x := range s.Vows {
		if x == v {
			s.Vows = append(s.Vows[:i], s.Vows[i+1:]...)
			break
		}
	}
	if g.P != nil {
		delete(g.P.Status, crVowDefOf(v.ID).St.ID)
	}
}

func (g *Game) crBreakVow(v *crVow, why string) {
	if v == nil {
		return
	}
	s := crKs(g)
	name := g.crVowName(v)
	g.crRemoveVow(v)
	s.Broke++
	g.Msg("{{R|You broke your vow: " + name + ". " + why + "}}")
	g.crKarma(4, -12, "broke the vow of "+crVowDefOf(v.ID).Name)
	g.crKarma(1, -6, "")
	if v.Faith > 0 {
		g.AddRep(fmt.Sprintf("faith:%d", v.Faith), -25, " for a broken vow")
		g.gdAddFav(v.Faith, -10, " for a broken vow")
	}
	if g.Cur != nil {
		// family.go reads "broke the" in deed text: Oathbreaker title
		g.deed("event", fmt.Sprintf("%s broke the vow of %s before %s: %s", g.P.Name, strings.ToLower(crVowDefOf(v.ID).Name), nonEmptyCr(v.To, "the gods"), why), 0, 0)
	}
	g.crFirst("oathbroken", fmt.Sprintf("%s broke the first sworn oath of their life.", g.P.Name))
}

func nonEmptyCr(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

func (g *Game) crFulfilVow(v *crVow) {
	s := crKs(g)
	name := g.crVowName(v)
	g.crRemoveVow(v)
	s.Kept++
	g.Msg("{{G|Vow fulfilled: " + name + ".}}")
	g.gainXP(60 + 10*g.P.Level)
	g.crKarma(4, 8, "kept the vow of "+crVowDefOf(v.ID).Name)
	g.crKarma(1, 4, "")
	if v.Faith > 0 {
		g.AddRep(fmt.Sprintf("faith:%d", v.Faith), 25, " for a kept vow")
		g.gdAddFav(v.Faith, 8, " for a kept vow")
	}
	if g.Cur != nil {
		g.deed("event", fmt.Sprintf("%s kept the vow of %s.", g.P.Name, strings.ToLower(crVowDefOf(v.ID).Name)), 0, 0)
	}
	g.crFirst("vowkept", fmt.Sprintf("%s kept a vow to the end.", g.P.Name))
}

// crAid counts one act of aid for a hospitality vow; crRefuse breaks it.
func (g *Game) crAid() {
	if v := g.crVow("ho"); v != nil {
		v.Prog++
		if v.Prog >= v.Need {
			g.crFulfilVow(v)
		}
	}
}

func (g *Game) crRefuse() {
	if v := g.crVow("ho"); v != nil {
		g.crBreakVow(v, "you turned away one in need.")
	}
}

// crLieBlocked: true when a Truth vow forbids this option label (use from checkOptions-like filters).
func (g *Game) crLieBlocked(label string) bool {
	return g.crVow("tr") != nil && strings.Contains(label, "[Bluff")
}

var crSilence = []struct{ Topic, Match string }{
	{"gossip", "What news"}, {"your name", "Who are you"}, {"secrets", "secret"}, {"haggling", "trade"},
}

// ---- hooks ----

func crIsPerson(d *Actor) bool {
	return d.Tpl == "npc" || d.SimPer > 0 || strings.HasPrefix(d.Faction, "state:") || d.Faction == "outlaws" || d.AI == "guard" || d.AI == "town"
}

func crKillHook(g *Game, d, src *Actor) {
	if g.P == nil || d == nil || src != g.P || d.AI == "follower" {
		return
	}
	s := crKs(g)
	innocent := !g.Hostile(d) && !d.Angry && d.AI != "hostile" && crIsPerson(d)
	if innocent {
		g.crKarma(0, -6, "killed "+d.Name+", who meant no harm")
		if d.Role == "priest" {
			g.crKarma(4, -5, "killed a priest")
		}
	} else {
		s.Snap["hk"]++
		if s.Snap["hk"]%8 == 0 {
			g.crKarma(0, -1, "a long road of killing")
		}
		if d.Boss || d.Level >= g.P.Level+3 {
			g.crKarma(2, 2, "faced down "+d.Name)
		}
	}
	if v := g.crVow("nv"); v != nil && crIsPerson(d) {
		g.crBreakVow(v, "you killed "+d.Name+".")
	}
	if v := g.crVow("ve"); v != nil && v.Arg != "" && strings.Contains(d.Name, v.Arg) {
		g.crFulfilVow(v)
	}
}

func crVowEnter(g *Game, zn *Zone) {
	if g.P == nil || zn == nil {
		return
	}
	if v := g.crVow("pi"); v != nil && zn.Kind == "town" && zn.Burg > 0 && zn.Burg != g.StartBurg {
		for _, b := range v.Seen {
			if b == zn.Burg {
				return
			}
		}
		v.Seen = append(v.Seen, zn.Burg)
		v.Prog++
		g.Msg(fmt.Sprintf("{{Y|Pilgrimage: %d of %d places.}}", v.Prog, v.Need))
		if v.Prog >= v.Need {
			g.crFulfilVow(v)
		}
	}
}

func (g *Game) crSnapDelta(key string, now int) int {
	s := crKs(g)
	old, ok := s.Snap[key]
	s.Snap[key] = now
	if !ok {
		return 0
	}
	return now - old
}

func crKarmaTurn(g *Game) {
	if g.P == nil || g.W == nil || g.Cur == nil || g.Dead {
		return
	}
	s := crKs(g)
	// vows tick every minute (cheap)
	if v := g.crVow("po"); v != nil {
		var lim int
		fmt.Sscanf(v.Arg, "%d", &lim)
		if g.P.Gold > lim {
			v.Over++
			if v.Over == 1 {
				g.Msg(fmt.Sprintf("{{R|You carry more than your vow of Poverty allows (%d %s). Give it away within hours.}}", lim, g.Sk.Currency))
			}
			if v.Over >= 180 {
				g.crBreakVow(v, "you kept your purse full.")
			}
		} else {
			v.Over = 0
		}
	}
	if v := g.crVow("ve"); v != nil && g.Turn%60 == 0 {
		if nem := nxSt(g).Nem; nem != nil && strings.Contains(nem.Name, v.Arg) || nem != nil && strings.Contains(v.Arg, nem.Name) {
			switch nem.Status {
			case "slain":
				g.crFulfilVow(v)
			case "spared", "truce":
				g.crBreakVow(v, "you made peace with "+v.Arg+".")
			}
		}
	}
	if g.Turn%20 == 0 {
		g.crApplyStatuses()
	}
	if g.Turn%10 != 3 {
		return
	}
	day := g.Day()
	if d := g.crSnapDelta("day", day); d != 0 {
		g.crDaily(day)
	}
	th := thSt(g)
	if d := g.crSnapDelta("spared", th.C["spared"]); d > 0 {
		g.crKarma(0, mini(d, 3)*4, "spared a beaten foe")
	}
	if d := g.crSnapDelta("fmsp", g.fm().Spared); d > 0 {
		g.crKarma(0, mini(d, 3)*4, "spared a yielded foe")
	}
	if d := g.crSnapDelta("lowhp", th.C["lowHP"]); d > 0 {
		g.crKarma(2, 1, "stood on after a near death")
	}
	if d := g.crSnapDelta("boss", th.C["bossKills"]); d > 0 {
		g.crKarma(2, 3*mini(d, 2), "slew a mighty foe")
	}
	uw := g.uwSt()
	if d := g.crSnapDelta("stolen", uw.Stolen); d > 0 {
		g.crKarma(1, -3*mini(d, 3), "theft")
		g.crKarma(3, -2*mini(d, 3), "")
	}
	if d := g.crSnapDelta("jailed", uw.Jailed); d > 0 {
		g.crKarma(1, -2, "was jailed")
	}
	bs := 0
	for _, b := range uw.Bounty {
		bs += b
	}
	if d := g.crSnapDelta("bounty", bs); d > 0 {
		g.crKarma(1, -2, "a witnessed crime")
	}
	fav := 0
	for _, f := range g.gd().Faith {
		fav += f.Fav
	}
	if d := g.crSnapDelta("fav", fav); d < 0 {
		g.crKarma(4, maxi(-8, d/2-1), "offended the gods")
	} else if d > 0 {
		g.crKarma(4, mini(4, d/2+1), "pleased the gods")
	}
	if sfSt(g).IsApostate && !s.Firsts["apo"] {
		s.Firsts["apo"] = true
		g.crKarma(4, -15, "branded an apostate")
	}
	// new deeds
	from, ok := s.Snap["deeds"]
	if !ok {
		from = len(g.Deeds)
	}
	for i := from; i < len(g.Deeds); i++ {
		d := g.Deeds[i]
		low := strings.ToLower(d.Text)
		switch {
		case strings.Contains(low, "robbed the grave"):
			g.crKarma(4, -6, "robbed a grave")
			g.crKarma(1, -2, "")
		case d.Kind == "murder":
			g.crKarma(0, -6, "murder")
		case strings.Contains(low, "gave a gift") || strings.Contains(low, "donat") || strings.Contains(low, "alms") || strings.Contains(low, "funded") || strings.Contains(low, "laid a gift"):
			g.crKarma(3, 3, "a gift freely given")
		}
	}
	s.Snap["deeds"] = len(g.Deeds)
}

func (g *Game) crApplyStatuses() {
	s := crKs(g)
	if g.P.Status == nil {
		g.P.Status = map[string]int{}
	}
	for _, p := range crPerks {
		v := s.Ax[p.ax]
		if (!p.neg && v >= 80) || (p.neg && v <= -80) {
			g.P.Status[p.def.ID] = 60
		} else {
			delete(g.P.Status, p.def.ID)
		}
	}
	any := false
	for _, d := range crVowDefs {
		if g.crVow(d.ID) != nil {
			g.P.Status[d.St.ID] = 60
			any = true
		} else {
			delete(g.P.Status, d.St.ID)
		}
	}
	if any {
		g.P.Status["crvow"] = 60
	} else {
		delete(g.P.Status, "crvow")
	}
}

// crDaily: vow terms and slow drift.
func (g *Game) crDaily(day int) {
	s := crKs(g)
	for _, v := range append([]*crVow{}, s.Vows...) {
		if v.Term > 0 && day >= v.Term {
			g.crFulfilVow(v)
			continue
		}
		if v.Dead > 0 && day >= v.Dead {
			g.crBreakVow(v, "time ran out.")
			continue
		}
		if v.Faith > 0 && day%5 == 0 {
			g.AddRep(fmt.Sprintf("faith:%d", v.Faith), 1, " for a kept vow")
		}
		switch v.ID {
		case "tr":
			if day%10 == 0 {
				g.crKarma(1, 1, "keeping the vow of Truth")
			}
		case "nv":
			if day%10 == 0 {
				g.crKarma(0, 1, "keeping the vow of Non-violence")
			}
		case "po":
			if day%10 == 0 {
				g.crKarma(3, 1, "keeping the vow of Poverty")
			}
		}
	}
}

// ---- talk ----

func crKarmaTalk(g *Game, a *Actor) []Option {
	if a == nil || g.P == nil || a.AI == "follower" {
		return nil
	}
	s := crKs(g)
	day := g.Day()
	// deceit/cruelty observed in earlier checks
	if a.Has("chk:officer") && !a.Has("cr:lie") {
		a.Set("cr:lie")
		g.crKarma(1, -3, "lied to "+a.Name)
		if v := g.crVow("tr"); v != nil {
			g.crBreakVow(v, "you lied.")
		}
	}
	if a.Has("chk:patrol") && !a.Has("cr:thr0") {
		a.Set("cr:thr0")
		g.crKarma(0, -2, "bullied "+a.Name)
	}
	// vow filters
	var keep []Option
	for _, o := range g.Opts {
		drop := false
		if g.crLieBlocked(o.Text) {
			drop = true
		}
		if v := g.crVow("si"); v != nil {
			for _, t := range crSilence {
				if t.Topic == v.Arg && strings.Contains(Plain(o.Text), t.Match) {
					drop = true
				}
			}
		}
		if !drop {
			keep = append(keep, o)
		}
	}
	g.Opts = keep
	// how they see you
	if p := lvPersonFor(g, a); p != nil && s.Op[p.Key] < day-8 {
		w, line := crReact(g, p, a)
		if w != 0 {
			s.Op[p.Key] = day
			g.lvMemoryAdd(p, "witnessed", "They have heard what kind of person you are.", w)
			g.Popup = append(g.Popup, "", "{{K|"+line+"}}")
		}
	}
	var opts []Option
	if !a.Has("cr:thr") && a.Role != "guard" && a.Role != "priest" && (s.Ax[0] <= -20 || (s.Ax[0] <= -8 && HashStr(fmt.Sprint(a.ID, ":", day))%2 == 0)) {
		opts = append(opts, g.checkOptionCr(a))
	}
	if !a.Has("cr:vch") && s.Ax[1] >= 35 {
		opts = append(opts, Option{"{{G|[Honest name]}} \"You know what I am. I have never lied to your kind.\"", func() {
			a.Set("cr:vch")
			g.AddRep(a.Faction, 6, " for your honest name")
			g.talkMenu("They nod slowly. Word of your honesty has travelled.")
		}})
	}
	if a.Role == "priest" || a.Role == "elder" || a.Role == "notable" {
		if len(s.Vows) < 3 {
			opts = append(opts, Option{"{{Y|Swear a vow.}}", func() { g.crVowMenu(a) }})
		}
		if len(s.Vows) > 0 && a.Role == "priest" {
			opts = append(opts, Option{"{{y|Ask to be released from a vow.}}", func() { g.crReleaseMenu(a) }})
		}
	}
	return opts
}

func (g *Game) checkOptionCr(a *Actor) Option {
	dc := 9
	return Option{g.checkLabel("intimidate", dc) + " {{R|Threaten them.}}", func() {
		a.Set("cr:thr")
		if g.roll2d6Check(g.checkMod("intimidate")-crKs(g).Ax[0]/40, dc) {
			take := 8 + g.P.Level*4
			if a.Gold > 0 {
				take = mini(take, a.Gold)
				a.Gold -= take
			}
			g.P.Gold += take
			g.crKarma(0, -3, "threatened "+a.Name)
			g.lvWitness("insult", "You threatened them.", -8)
			g.talkMenu(fmt.Sprintf("{{G|[Intimidate: Success]}}\n\nThey fumble out %d %s and look at the ground.", take, g.Sk.Currency))
			return
		}
		g.AddRep(a.Faction, -6, " for threatening one of theirs")
		g.crKarma(0, -2, "failed threat")
		g.talkMenu("{{R|[Intimidate: Failure]}}\n\n\"Get away from me!\" Heads turn.")
	}}
}

// crReact: opinion shift and line for a person by their own traits (agreeableness, conscientiousness, neuroticism).
func crReact(g *Game, p *lvPerson, a *Actor) (int, string) {
	s := crKs(g)
	agree, cons, neur := p.Trait[3], p.Trait[1], p.Trait[4]
	switch {
	case s.Ax[0] >= 30 && agree >= 55:
		return 3, p.Name + " warms to you. Word of your mercy has reached them."
	case s.Ax[0] <= -30 && agree >= 55:
		return -4, p.Name + " keeps a wary distance. They have heard what you do to the helpless."
	case s.Ax[0] <= -30 && neur >= 60:
		return -2, p.Name + " flinches when you step near."
	case s.Ax[0] <= -30 && agree < 40:
		return 2, p.Name + " eyes you with a hard sort of respect."
	case s.Ax[1] >= 30 && cons >= 55:
		return 2, p.Name + " trusts a person known for plain dealing."
	case s.Ax[1] <= -30 && cons >= 55:
		return -3, p.Name + " counts their coin twice, thinking of your reputation."
	case s.Ax[4] >= 30 && p.Role == "priest":
		return 3, p.Name + " blesses you, having heard of your devotion."
	case s.Ax[4] <= -30 && p.Role == "priest":
		return -3, p.Name + " makes a warding sign: they know you mock the gods."
	case s.Ax[3] >= 30:
		return 1, p.Name + " has heard you are free with your coin."
	case s.Ax[3] <= -30 && agree >= 55:
		return -2, p.Name + " has heard you grasp at every coin."
	}
	return 0, ""
}

// ---- vow menus ----

func (g *Game) crVengTarget() (string, bool) {
	if nem := nxSt(g).Nem; nem != nil && nxActive(nem) {
		return nem.Name, true
	}
	var keys []string
	for k, p := range lvSt(g).People {
		if p.Alive && p.Opinion <= -15 && p.Met {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	if len(keys) > 0 {
		return lvSt(g).People[keys[0]].Name, true
	}
	if st := g.W.stateByID[g.HomeState]; st != nil && g.HomeState > 0 {
		for _, w := range st.Wars {
			if e := g.W.stateByID[w]; e != nil && e.Ruler != "" && e.Alive {
				return e.Ruler, true
			}
		}
	}
	return "", false
}

func (g *Game) crVowMenu(a *Actor) {
	s := crKs(g)
	g.Popup = Wrap("\"A vow binds the one who makes it. Choose with care: the breaking of it is remembered.\"", 100)
	g.Opts = nil
	for _, d := range crVowDefs {
		d := d
		if g.crVow(d.ID) != nil {
			continue
		}
		label := fmt.Sprintf("{{Y|%s}}: %s {{K|(%s)}}", d.Name, d.Desc, d.Perk)
		switch d.ID {
		case "ve":
			t, ok := g.crVengTarget()
			if !ok {
				continue
			}
			label = fmt.Sprintf("{{Y|Vengeance on %s}}: hunt them to the end. {{K|(%s)}}", t, d.Perk)
		case "si":
			g.Opts = append(g.Opts, Option{fmt.Sprintf("{{Y|%s}}: %s {{K|(%s)}}", d.Name, d.Desc, d.Perk), func() { g.crSilenceMenu(a) }})
			continue
		}
		g.Opts = append(g.Opts, Option{label, func() { g.crSwear(a, d.ID, "") }})
	}
	g.Opts = append(g.Opts, Option{"Never mind.", func() { g.talkMenu(g.greeting(a)) }})
	g.Sel = 0
	_ = s
}

func (g *Game) crSilenceMenu(a *Actor) {
	g.Popup = Wrap("\"On what will you keep silent?\"", 100)
	g.Opts = nil
	for _, t := range crSilence {
		t := t
		g.Opts = append(g.Opts, Option{"Silence on " + t.Topic + ".", func() { g.crSwear(a, "si", t.Topic) }})
	}
	g.Opts = append(g.Opts, Option{"Never mind.", func() { g.crVowMenu(a) }})
	g.Sel = 0
}

func (g *Game) crSwear(a *Actor, id, arg string) {
	s := crKs(g)
	day := g.Day()
	d := crVowDefOf(id)
	v := &crVow{ID: id, Arg: arg, Since: day, To: a.Name, Faith: a.Religion}
	if d.Days > 0 {
		v.Term = day + d.Days
	}
	switch id {
	case "po":
		v.Arg = fmt.Sprint(30 + 5*g.P.Level)
	case "ho":
		v.Need = 5
	case "ve":
		v.Arg, _ = g.crVengTarget()
	case "pi":
		v.Need = 4 + int(HashStr(g.Seed+":crpi:"+fmt.Sprint(day))%3)
		v.Dead = day + 14*v.Need
	}
	s.Vows = append(s.Vows, v)
	g.crApplyStatuses()
	g.crFirst("vowsworn", fmt.Sprintf("%s swore a first vow before %s.", g.P.Name, a.Name))
	g.crKarma(4, 3, "swore the vow of "+d.Name)
	g.AddRep(a.Faction, 3, " for a sworn vow")
	g.talkMenu(fmt.Sprintf("You swear the vow of %s. {{K|%s}}\n\n\"Then it is so. It will be remembered, kept or broken.\"", g.crVowName(v), d.Perk))
}

func (g *Game) crReleaseMenu(a *Actor) {
	s := crKs(g)
	cost := 50 + 10*g.P.Level
	g.Popup = Wrap(fmt.Sprintf("\"I can unbind you, for an offering of %d %s. The gods note it, but it is no breaking.\"", cost, g.Sk.Currency), 100)
	g.Opts = nil
	for _, v := range s.Vows {
		v := v
		g.Opts = append(g.Opts, Option{"Release me from: " + g.crVowName(v), func() {
			if g.P.Gold < cost {
				g.talkMenu("\"You cannot afford the offering.\"")
				return
			}
			g.P.Gold -= cost
			g.crRemoveVow(v)
			g.crKarma(4, -3, "asked to be released from a vow")
			g.talkMenu("The vow is lifted. You feel lighter, and slightly smaller.")
		}})
	}
	g.Opts = append(g.Opts, Option{"Never mind.", func() { g.talkMenu(g.greeting(a)) }})
	g.Sel = 0
}

// ---- ledger screen ----

func (g *Game) crAbilities() []Ability {
	if g.P == nil {
		return nil
	}
	return []Ability{{ID: "cr_ledger", Name: "Karma and vows", Target: "self"}}
}

func (g *Game) crAbUse(id string) {
	if id == "cr_ledger" {
		g.Mode, g.Sel, g.Tab = "crk", 0, 0
	}
}

func crKeys(g *Game, k string) {
	s := crKs(g)
	switch k {
	case "Escape", "q":
		g.Mode = "main"
	case "Tab", "ArrowRight", "ArrowLeft":
		g.Tab = (g.Tab + 1) % 3
		g.Sel = 0
	case "ArrowUp":
		g.Sel = maxi(0, g.Sel-1)
	case "ArrowDown":
		g.Sel = mini(maxi(0, len(s.Vows)-1), g.Sel+1)
	case "r":
		if g.Tab == 1 && g.Sel < len(s.Vows) {
			g.crBreakVow(s.Vows[g.Sel], "you renounced it.")
			g.Sel = 0
		}
	}
}

func (g *Game) crDrawLedger() {
	sc := g.sc
	s := crKs(g)
	x, y, w, h := g.panel("Karma and Vows")
	tabs := []string{"Karma", "Vows", "Crossroads"}
	tx := x
	for i, t := range tabs {
		c := "K"
		if i == g.Tab {
			c = "W"
		}
		tx = sc.TextB(tx, y, fmt.Sprintf("{{%s|[%s]}} ", c, t), CGray, CBgDark, x+w)
	}
	sc.TextB(tx+2, y, "{{K|Tab · r renounce vow (counts as broken) · Esc}}", CGray, CBgDark, x+w)
	y += 2
	line := func(t string) {
		for _, l := range Wrap(t, w) {
			if y < sc2Bottom(x, h, g) {
				sc.TextB(x, y, l, CGray, CBgDark, x+w)
				y++
			}
		}
	}
	switch g.Tab {
	case 0:
		line("You are known as {{M|" + g.crEpithet() + "}}.")
		y++
		for i := 0; i < 5; i++ {
			v := s.Ax[i]
			pos := (v + 100) * 24 / 200
			bar := []rune(strings.Repeat("-", 25))
			bar[12] = '|'
			lo, hi := 12, pos
			if pos < 12 {
				lo, hi = pos, 12
			}
			for j := lo; j <= hi; j++ {
				if j != 12 {
					bar[j] = '='
				}
			}
			col := "G"
			if v < 0 {
				col = "R"
			}
			line(fmt.Sprintf("{{G|%-10s}} %s {{R|%-10s}} {{%s|%+d}}", crAxisNames[i][0], string(bar), crAxisNames[i][1], col, v))
		}
		y++
		for _, p := range crPerks {
			if st := g.P.Status[p.def.ID]; st > 0 {
				line("{{Y|" + p.def.Name + "}}: " + p.desc)
			}
		}
		line("{{K|Divine favour shifts by " + fmt.Sprintf("%+d", g.KarmaDivineMod()) + ". Recent:}}")
		for i := len(s.Log) - 1; i >= 0 && i >= len(s.Log)-8; i-- {
			line(fmt.Sprintf("{{K|day %d: %s}}", s.Log[i].Day+1, s.Log[i].Text))
		}
	case 1:
		line(fmt.Sprintf("Vows kept {{G|%d}}, broken {{R|%d}}. Swear vows to priests, elders and notables in talk.", s.Kept, s.Broke))
		y++
		if len(s.Vows) == 0 {
			line("{{K|You are bound by no vow.}}")
		}
		g.Sel = clampi(g.Sel, 0, maxi(0, len(s.Vows)-1))
		for i, v := range s.Vows {
			d := crVowDefOf(v.ID)
			bg := CBgDark
			if i == g.Sel {
				bg = CBgSel
			}
			sc.Fill(x, y, w, 1, bg)
			left := ""
			if v.Term > 0 {
				left = fmt.Sprintf(" · %d days left", maxi(0, v.Term-g.Day()))
			} else if v.Dead > 0 {
				left = fmt.Sprintf(" · %d days left", maxi(0, v.Dead-g.Day()))
			}
			sc.TextB(x, y, fmt.Sprintf("{{Y|%s}}%s {{K|(%s)}}", g.crVowName(v), left, d.Perk), CGray, bg, x+w)
			y++
			if i == g.Sel {
				for _, l := range Wrap("{{K|"+d.Desc+" Sworn to "+nonEmptyCr(v.To, "the gods")+".}}", w-2) {
					sc.TextB(x+2, y, l, CGray, CBgDark, x+w)
					y++
				}
			}
		}
	default:
		if len(s.Hist) == 0 {
			line("{{K|No crossroads yet. They find you in towns, at junctions and at camp.}}")
		}
		for i := len(s.Hist) - 1; i >= 0; i-- {
			line(fmt.Sprintf("{{K|day %d}} %s", s.Hist[i].Day+1, s.Hist[i].Text))
		}
	}
}

func sc2Bottom(x, h int, g *Game) int { return 2 + 1 + h - 1 }
