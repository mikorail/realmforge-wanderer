package engine

import (
	"fmt"
	"strings"
)

// injuries.go (prefix ij): permanent injuries and scars. Effects ride on the status registry
// (statuses "ij_<id>", held at a huge duration and re-asserted each minute from g.ext("ij")), plus
// ijScoreMod for attributes (hook in Actor.Score). Never draws g.R outside play rolls.

const ijForever = 1 << 30

type ijInjury struct {
	ID       string `json:"id"`
	Name     string `json:"n"`
	Part     string `json:"p"`
	Effect   string `json:"e"`
	Healable bool   `json:"h,omitempty"`
	Until    int    `json:"u,omitempty"` // turn when it heals by itself (0 = never)
	Got      int    `json:"t,omitempty"`
}

type ijDef struct {
	ID, Name, Part, Effect, Scar, Slot string
	Days, Fee                          int
	Heal                               bool // a physician can mend it
	Speed, Hit, DV                     int
	Weight                             map[string]int // cause class -> weight
}

var ijDefs = []ijDef{
	{ID: "lame", Name: "Lame leg", Part: "leg", Effect: "slower on foot", Scar: "a leg that never healed straight", Slot: "feet", Fee: 120, Heal: true, Speed: -25,
		Weight: map[string]int{"fall": 5, "blow": 2, "poison": 1}},
	{ID: "blind", Name: "Blinded eye", Part: "eye", Effect: "worse aim at range, -1 Intelligence", Scar: "a milky, sightless eye", Slot: "face", Fee: 200, Heal: true, Hit: -1,
		Weight: map[string]int{"blow": 2, "fire": 2}},
	{ID: "ribs", Name: "Broken ribs", Part: "ribs", Effect: "-2 Toughness, slower", Scar: "a ridge of knitted ribs", Days: 14, Fee: 40, Heal: true, Speed: -5,
		Weight: map[string]int{"fall": 5, "blow": 3}},
	{ID: "scar", Name: "Scarred face", Part: "face", Effect: "-1 Ego; strangers are wary", Scar: "a long scar across the face", Fee: 0,
		Weight: map[string]int{"blow": 3, "fire": 5}},
	{ID: "shake", Name: "Shaking hand", Part: "hand", Effect: "worse aim, -1 Agility", Scar: "a hand that trembles", Slot: "hand", Fee: 150, Heal: true, Hit: -2,
		Weight: map[string]int{"poison": 4, "blow": 2}},
	{ID: "deaf", Name: "Deafness", Part: "ear", Effect: "you hear less gossip", Scar: "ears ringing with old thunder", Slot: "head", Fee: 180, Heal: true,
		Weight: map[string]int{"poison": 3, "blow": 1}},
	{ID: "finger", Name: "Lost finger", Part: "hand", Effect: "-1 to Tinkering", Scar: "a missing finger", Slot: "hand", Fee: 90,
		Weight: map[string]int{"blow": 2, "fire": 2}},
}

var ijDefByID = func() map[string]*ijDef {
	m := map[string]*ijDef{}
	for i := range ijDefs {
		m[ijDefs[i].ID] = &ijDefs[i]
	}
	return m
}()

func init() {
	for i := range ijDefs {
		d := &ijDefs[i]
		skAddStatus(&skStatusDef{ID: "ij_" + d.ID, Name: d.Name, Glyph: '+', Speed: d.Speed, Hit: d.Hit, DV: d.DV})
	}
	skAddStatus(&skStatusDef{ID: "ij_tired", Name: "Tired", Glyph: 'z', Hit: -1, Speed: -5})
	skAddStatus(&skStatusDef{ID: "ij_spent", Name: "Exhausted", Glyph: 'Z', Hit: -2, DV: -1, Speed: -15})
	skAddStatus(&skStatusDef{ID: "rested", Name: "Rested", Glyph: 'r'})
	turnHooks = append(turnHooks, ijInjuryTurn)
	talkHooks = append(talkHooks, ijTalk)
}

// ijScoreMod is the attribute change from injuries and exhaustion (hook in Actor.Score).
func ijScoreMod(a *Actor, attr int) int {
	if a == nil || len(a.Status) == 0 {
		return 0
	}
	n := 0
	switch attr {
	case TOU:
		if a.St("ij_ribs") > 0 {
			n -= 2
		}
	case EGO:
		if a.St("ij_scar") > 0 {
			n--
		}
	case AGI:
		if a.St("ij_shake") > 0 {
			n--
		}
		if a.St("ij_spent") > 0 {
			n -= 2
		}
	case INT:
		if a.St("ij_blind") > 0 {
			n--
		}
		if a.St("ij_spent") > 0 {
			n--
		}
	case WIL:
		if a.St("ij_spent") > 0 {
			n--
		}
	}
	return n
}

func (g *Game) ijStatMod(attr int) int { return ijScoreMod(g.P, attr) }

// ijSpeedMod is the Quick() change from injuries and fatigue (already applied through the status table).
func (g *Game) ijSpeedMod() int {
	n := 0
	for _, in := range g.ijSt().Inj {
		if d := ijDefByID[in.ID]; d != nil {
			n += d.Speed
		}
	}
	if g.P != nil {
		if g.P.St("ij_spent") > 0 {
			n -= 15
		} else if g.P.St("ij_tired") > 0 {
			n -= 5
		}
	}
	return n
}

func (g *Game) ijHasInj(id string) bool {
	for _, in := range g.ijSt().Inj {
		if in.ID == id {
			return true
		}
	}
	return false
}

// ijDeaf: gossip heard less (gossip() may skip lines when true).
func (g *Game) ijDeaf() bool { return g.ijHasInj("deaf") }

// ijCheckMod is the check modifier from injuries, fatigue and rest (hook in checkMod).
func (g *Game) ijCheckMod(kind string) int {
	if g.P == nil {
		return 0
	}
	n := 0
	switch {
	case g.P.St("ij_spent") > 0:
		n -= 2
	case g.P.St("ij_tired") > 0:
		n--
	case g.P.St("rested") > 0:
		n++
	}
	if kind == "tinkering" && g.ijHasInj("finger") {
		n--
	}
	if kind == "perception" && g.ijHasInj("blind") {
		n -= 2
	}
	return n
}

// ijScars lists scar lore for the paper doll and the obituary.
func (g *Game) ijScars() []string {
	st := g.ijSt()
	out := append([]string{}, st.Scars...)
	return out
}

// ijScarGreeting is a stranger's remark about your scars ("" most of the time).
func (g *Game) ijScarGreeting(a *Actor) string {
	st := g.ijSt()
	if a == nil || len(st.Scars) == 0 || a.ID%3 != 0 {
		return ""
	}
	switch {
	case g.ijHasInj("scar"):
		return "\"That scar... I don't want to know. Keep your trouble out of my door.\""
	case g.ijHasInj("lame"):
		return "\"You walk like a veteran. Sit down, wanderer, rest that leg.\""
	case g.ijHasInj("blind"):
		return "\"The eye, wanderer... was it a blade or a beast?\""
	}
	return "\"You carry your old wounds like a coat. Speak.\""
}

func ijCause(cause string) string {
	c := strings.ToLower(cause)
	switch {
	case strings.Contains(c, "fall"):
		return "fall"
	case strings.Contains(c, "poison") || strings.Contains(c, "disease") || strings.Contains(c, "plague") || strings.Contains(c, "sick"):
		return "poison"
	case strings.Contains(c, "fire") || strings.Contains(c, "burn") || strings.Contains(c, "acid"):
		return "fire"
	}
	return "blow"
}

// ijOnHurt: called from hurt() after the player takes n damage (hook).
func (g *Game) ijOnHurt(d *Actor, n int, src *Actor, cause string) {
	if d != g.P || g.P == nil || g.P.MaxHP <= 0 {
		return
	}
	st := g.ijSt()
	st.Hook = true
	cls := ijCause(cause)
	perm := g.ijHas("permanent")
	mul := 1.0
	if perm {
		mul = 1.5
	}
	pct := n * 100 / g.P.MaxHP
	switch {
	case cls == "fall" && pct >= 20:
		g.ijMaybe(0.35*mul, cls)
	case cls == "poison":
		if g.P.HP > 0 && g.P.HP*100 < 25*g.P.MaxHP {
			g.ijMaybe(0.06*mul, cls)
		}
	case pct >= 40:
		g.ijMaybe(0.4*mul, cls)
	}
}

func (g *Game) ijMaybe(chance float64, cls string) {
	if g.R.Chance(chance) {
		g.ijGive(cls)
	}
}

func (g *Game) ijFitted(d *ijDef) bool {
	if d.Slot == "" || g.P == nil {
		return false
	}
	for _, bp := range g.P.Body {
		if bp.Slot == d.Slot && bp.Aug != nil && !bp.Severed {
			return true
		}
	}
	return false
}

// ijGive picks a fitting injury for the cause class (in-play roll) and applies it.
func (g *Game) ijGive(cls string) {
	var pool []*ijDef
	total := 0
	for i := range ijDefs {
		d := &ijDefs[i]
		w := d.Weight[cls]
		if w == 0 || g.ijHasInj(d.ID) || g.ijFitted(d) {
			continue
		}
		pool = append(pool, d)
		total += w
	}
	if total == 0 {
		return
	}
	r := g.R.Int(1, total)
	for _, d := range pool {
		r -= d.Weight[cls]
		if r <= 0 {
			g.ijAdd(d)
			return
		}
	}
}

func (g *Game) ijAdd(d *ijDef) {
	st := g.ijSt()
	perm := g.ijHas("permanent")
	in := ijInjury{ID: d.ID, Name: d.Name, Part: d.Part, Effect: d.Effect, Healable: d.Heal && !perm, Got: g.Turn}
	if d.Days > 0 && !perm {
		in.Until = g.Turn + d.Days*DayLen
	}
	st.Inj = append(st.Inj, in)
	if len(st.Scars) < 12 {
		st.Scars = append(st.Scars, fmt.Sprintf("%s (day %d, near %s)", d.Scar, g.Turn/DayLen+1, g.Cur.Name))
	}
	g.P.AddSt("ij_"+d.ID, ijForever)
	g.Msg(fmt.Sprintf("{{R|Injury: %s.}} {{K|(%s)}}", d.Name, d.Effect))
	g.ijFirst("injury", fmt.Sprintf("%s was maimed for life: %s.", g.P.Name, strings.ToLower(d.Name)))
}

// ijCure removes an injury; the scar lore stays.
func (g *Game) ijCure(id, msg string) {
	st := g.ijSt()
	for i, in := range st.Inj {
		if in.ID == id {
			st.Inj = append(st.Inj[:i], st.Inj[i+1:]...)
			delete(g.P.Status, "ij_"+id)
			if msg != "" {
				g.Msg("{{G|" + msg + "}}")
			}
			g.ijFirst("cured", fmt.Sprintf("%s was mended after a lasting injury.", g.P.Name))
			return
		}
	}
}

// ijAugmentCures: a graft in a body part (slot name) replaces what the injury took.
func (g *Game) ijAugmentCures(part string) {
	st := g.ijSt()
	for i := len(st.Inj) - 1; i >= 0; i-- {
		d := ijDefByID[st.Inj[i].ID]
		if d != nil && d.Slot != "" && d.Slot == part {
			g.ijCure(d.ID, fmt.Sprintf("Your new %s replaces the damage: %s is cured.", g.augSkin().Noun, strings.ToLower(d.Name)))
		}
	}
}

func ijInjuryTurn(g *Game) {
	st := g.ijSt()
	p := g.P
	if p == nil {
		return
	}
	// big single-minute drop fallback while hurt() is not wired
	if !st.Hook {
		if st.LastHP > 0 && p.MaxHP > 0 && (st.LastHP-p.HP)*100 >= 40*p.MaxHP && p.HP > 0 {
			g.ijMaybe(0.4, "blow")
		}
		st.LastHP = p.HP
	}
	if len(st.Inj) == 0 {
		return
	}
	for _, bp := range p.Body {
		if bp.Aug != nil && !bp.Severed && g.Turn%60 == 0 {
			g.ijAugmentCures(bp.Slot)
		}
	}
	for i := len(st.Inj) - 1; i >= 0; i-- {
		in := st.Inj[i]
		if in.Until > 0 && g.Turn >= in.Until {
			g.ijCure(in.ID, "Your "+strings.ToLower(in.Name)+" has healed with time.")
			continue
		}
		if p.St("ij_"+in.ID) <= 0 {
			p.AddSt("ij_"+in.ID, ijForever)
		}
	}
}

// ---- healers ----

func ijIsPhysician(a *Actor) bool {
	r := strings.ToLower(a.Role)
	return strings.Contains(r, "heal") || r == "apothecary" || strings.Contains(r, "physician") || strings.Contains(r, "doctor")
}

func ijTalk(g *Game, a *Actor) []Option {
	var out []Option
	st := g.ijSt()
	if a.Role == "merchant" {
		cost := maxi(1, int(60*g.ijPriceMod()))
		out = append(out, Option{fmt.Sprintf("{{W|Buy a travelling tent.}} {{K|(%d %s)}}", cost, g.Sk.Currency), func() {
			if g.P.Gold < cost {
				g.talkMenu("\"You can't afford it.\"")
				return
			}
			g.P.Gold -= cost
			g.give(g.ijMakeTent())
			g.talkMenu("\"Oiled canvas and good poles. It will keep the rain off.\"")
		}})
	}
	if strings.Contains(a.Role, "inn") {
		out = append(out, g.ijRoomOption(a))
	}
	if len(st.Inj) > 0 && (ijIsPhysician(a) || a.Role == "priest") {
		priest := a.Role == "priest"
		label := "{{G|Have a physician look at my old injuries.}}"
		if priest {
			label = "{{G|Pray for a miracle on my body.}}"
		}
		out = append(out, Option{label, func() { g.ijHealMenu(a, priest) }})
	}
	return out
}

func (g *Game) ijHealMenu(a *Actor, miracle bool) {
	st := g.ijSt()
	g.Popup = Wrap("\"Which hurt is it?\"", 100)
	g.Opts = nil
	for _, in := range st.Inj {
		d := ijDefByID[in.ID]
		if d == nil {
			continue
		}
		in := in
		fee := d.Fee
		ok := in.Healable
		if miracle {
			fee = 150 + g.P.Level*20 + d.Fee*2
			ok = true
		} else if d.Fee == 0 {
			ok = false
		}
		fee = int(float64(fee) * g.ijPriceMod())
		if !ok {
			g.Opts = append(g.Opts, Option{fmt.Sprintf("{{K|%s: beyond a physician's craft.}}", in.Name), func() { g.ijHealMenu(a, miracle) }})
			continue
		}
		g.Opts = append(g.Opts, Option{fmt.Sprintf("Mend %s. {{K|(%d %s)}}", strings.ToLower(in.Name), fee, g.Sk.Currency), func() {
			if g.P.Gold < fee {
				g.talkMenu("\"You can't afford it.\"")
				return
			}
			g.P.Gold -= fee
			g.ijCure(in.ID, "The "+strings.ToLower(in.Name)+" is mended.")
			g.talkMenu("\"It is done.\"")
		}})
	}
	g.Opts = append(g.Opts, Option{"Never mind.", func() { g.talkMenu(g.greeting(a)) }})
	g.Mode = "talk"
}
