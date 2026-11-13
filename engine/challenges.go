package engine

import (
	"fmt"
	"strings"
)

// challenges.go (prefix ij): challenge modifiers chosen at creation (CharSpec.Mods), plus the shared
// saved state of the ij feature set (injuries.go, fatigue.go). State lives in g.ext("ij"); nothing in it
// is keyed by zone, burg or person ids, so it is global (no unLocalKeys entry).

type ijChallenge struct {
	ID, Name, Desc string
	Mul            float64
}

var ijChallenges = []ijChallenge{
	{"nohealing", "No natural healing", "Wounds only close with tonics, healers and prayer.", 1.25},
	{"permanent", "Permanent injuries", "Injuries never heal with time and physicians cannot mend them. Easier to get.", 1.15},
	{"onecompanion", "One companion only", "Only one companion may travel with you.", 1.1},
	{"doubleprices", "Double prices", "Merchants, healers and innkeepers charge twice as much.", 1.15},
	{"nofast", "No fast travel", "No trains, ships, balloons, auto-travel or travel between places.", 1.2},
	{"ironman", "Iron man", "No manual saving. The game saves only on stairs and in towns; death ends the run.", 1.15},
	{"hungry", "Hungry world", "Hunger and thirst come twice as fast.", 1.2},
	{"fragile", "Fragile", "Half the hit points.", 1.3},
	{"frailgear", "Frail gear", "Items break three times faster (needs item durability: not active yet).", 1},
	{"famous", "Famous", "Bounties and reputation swings are doubled.", 1.1},
	{"quiet", "Quiet world", "No hints, no map overlay tips.", 1.05},
}

var ijChallengeByID = func() map[string]*ijChallenge {
	m := map[string]*ijChallenge{}
	for i := range ijChallenges {
		m[ijChallenges[i].ID] = &ijChallenges[i]
	}
	return m
}()

type ijState struct {
	Mods     []string        `json:"mods,omitempty"`
	Inj      []ijInjury      `json:"inj,omitempty"`
	Scars    []string        `json:"scars,omitempty"`
	Fat      float64         `json:"fat,omitempty"`
	LastHP   int             `json:"lhp,omitempty"`
	LastMax  int             `json:"lmx,omitempty"`
	Hook     bool            `json:"hk,omitempty"` // ijOnHurt is wired: skip the HP-delta fallback
	Sleep    string          `json:"sl,omitempty"` // kind while a sleep command runs
	Down     bool            `json:"dn,omitempty"` // collapsed from exhaustion
	Tier     int             `json:"tr,omitempty"` // last announced fatigue tier
	Firsts   map[string]bool `json:"fs,omitempty"`
	WantSave bool            `json:"ws,omitempty"`
	LastZ    int             `json:"lz,omitempty"`
}

func (g *Game) ijSt() *ijState { return g.ext("ij", func() any { return &ijState{} }).(*ijState) }

func (g *Game) ijHas(id string) bool {
	for _, m := range g.ijSt().Mods {
		if m == id {
			return true
		}
	}
	return false
}

func (g *Game) ijFirst(key, deed string) {
	st := g.ijSt()
	if st.Firsts == nil {
		st.Firsts = map[string]bool{}
	}
	if st.Firsts[key] {
		return
	}
	st.Firsts[key] = true
	g.deed("event", deed, 0, 0)
}

func ijChallengeList() []ijChallenge { return ijChallenges }

// ijApply runs once at the end of NewGame (after stApply).
func (g *Game) ijApply(spec CharSpec) {
	if g.P == nil {
		return
	}
	st := g.ijSt()
	st.LastHP, st.LastMax = g.P.HP, g.P.MaxHP
	seen := map[string]bool{}
	var names []string
	for _, m := range spec.Mods {
		c := ijChallengeByID[m]
		if c == nil || seen[m] {
			continue
		}
		seen[m] = true
		st.Mods = append(st.Mods, m)
		names = append(names, c.Name)
	}
	if len(st.Mods) == 0 {
		return
	}
	if g.ijHas("fragile") {
		g.P.MaxHP = maxi(2, g.P.MaxHP/2)
		g.P.HP = g.P.MaxHP
		st.LastHP, st.LastMax = g.P.HP, g.P.MaxHP
	}
	if g.ijHas("quiet") {
		g.SetPref("hints", false)
	}
	if g.ijHas("ironman") {
		g.GameMode = "classic"
	}
	g.Msg("{{Y|Challenge:}} " + strings.Join(names, ", ") + fmt.Sprintf(" {{K|(score x%.2f)}}", g.ijScoreMul()))
	g.deed("event", fmt.Sprintf("%s set out under a vow: %s.", g.P.Name, strings.Join(names, ", ")), 0, 0)
}

// ijScoreMul multiplies the hall-of-fame score (halloffame.go hfScore).
func (g *Game) ijScoreMul() float64 {
	if g.P == nil {
		return 1
	}
	m := 1.0
	for _, id := range g.ijSt().Mods {
		if c := ijChallengeByID[id]; c != nil {
			m *= c.Mul
		}
	}
	if m > 3 {
		m = 3
	}
	return m
}

// ijPriceMod multiplies BUY prices (talk.go price(), buy branch).
func (g *Game) ijPriceMod() float64 {
	if g.ijHas("doubleprices") {
		return 2
	}
	return 1
}

// ijMaxCompanions replaces maxCompanions at its use sites.
func (g *Game) ijMaxCompanions(def int) int {
	if g.ijHas("onecompanion") {
		return mini(def, 1)
	}
	return def
}

// ijHungerMul is 2 in a hungry world (the hunger tick itself is applied by ijTurn).
func (g *Game) ijHungerMul() int {
	if g.ijHas("hungry") {
		return 2
	}
	return 1
}

// ijRepMul multiplies reputation swings and bounties.
func (g *Game) ijRepMul() float64 {
	if g.ijHas("famous") {
		return 2
	}
	return 1
}

// ijNoRegen reports "no natural healing" (turn() regen and tvPass).
func (g *Game) ijNoRegen() bool { return g.ijHas("nohealing") }

// ijNoFastTravel: transport.go, sailing.go, journeys.go, places.go, auto-travel should refuse when true.
func (g *Game) ijNoFastTravel() bool { return g.ijHas("nofast") }

// ijNoManualSave: the S key should refuse when true.
func (g *Game) ijNoManualSave() bool { return g.ijHas("ironman") }

// ijWantsSave returns true once after entering a town or taking stairs in iron man (Key(): out.Save = true).
func (g *Game) ijWantsSave() bool {
	st := g.ijSt()
	if st.WantSave {
		st.WantSave = false
		return true
	}
	return false
}

func init() {
	turnHooks = append(turnHooks, ijChallengeTurn)
	enterHooks = append(enterHooks, ijChallengeEnter)
}

func ijChallengeEnter(g *Game, zn *Zone) {
	st := g.ijSt()
	if g.ijHas("quiet") {
		g.SetPref("hints", false)
	}
	if g.ijHas("ironman") && (zn.Kind == "town" || zn.Z != st.LastZ) {
		st.WantSave = true
	}
	st.LastZ = zn.Z
}

func ijChallengeTurn(g *Game) {
	st := g.ijSt()
	p := g.P
	if len(st.Mods) == 0 || p == nil {
		return
	}
	if g.ijHas("hungry") && g.Turn%(2+g.Skills["endurance"]) != 0 && !g.slowHunger() {
		g.Hunger++
		g.Thirst++
	}
	if g.ijHas("fragile") {
		if st.LastMax > 0 && p.MaxHP > st.LastMax {
			d := p.MaxHP - st.LastMax
			p.MaxHP = st.LastMax + d/2
			p.HP = mini(p.HP, p.MaxHP)
		}
		st.LastMax = p.MaxHP
	}
}
