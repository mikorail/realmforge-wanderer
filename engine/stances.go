package engine

import (
	"fmt"
	"strings"
)

// Combat depth (prefix cs): stances (guard / aggressive / evasive / steady aim) toggled by abilities,
// weapon-class techniques unlocked by skill or practice, parry / riposte / dodge / shield bash,
// a momentum counter that unlocks a finisher, and stagger / off-balance / pinned / disarmed statuses.
// State lives in g.ext("cs"). Hooks (see report): csHitMod, csDefend, csDmgMod, csOnHit, csOnMiss,
// csOnShot, csOnHurt in combat.go; csAbilities / csUse in powers.go. Kill practice uses killHooks.

type csState struct {
	Stance string         `json:"st,omitempty"` // "", guard, aggr, evade, aim
	Combo  int            `json:"cb,omitempty"`
	LastT  int            `json:"lt,omitempty"` // g.Tick of the last landed hit
	Prac   map[string]int `json:"pr,omitempty"` // weapon class -> hits and kills landed
	Fin    int            `json:"fin,omitempty"`
	Parry  int            `json:"par,omitempty"`
	Temp   int            `json:"-"` // one-shot to-hit bonus for a technique in flight
	busy   bool
}

const csComboMax = 5

func (g *Game) cs() *csState {
	st := g.ext("cs", func() any { return &csState{} }).(*csState)
	if st.Prac == nil {
		st.Prac = map[string]int{}
	}
	return st
}

func init() {
	for _, d := range []*skStatusDef{
		{ID: "cs_stagger", Name: "Staggered", Glyph: 's', Hit: -3, DV: -3, Speed: -30},
		{ID: "cs_offbal", Name: "Off-balance", Glyph: 'o', Hit: -2, DV: -4},
		{ID: "cs_pinned", Name: "Pinned", Glyph: 'p', DV: -3, Speed: -100},
		{ID: "cs_disarmed", Name: "Disarmed", Glyph: 'd', Hit: -5},
	} {
		skAddStatus(d)
	}
	killHooks = append(killHooks, func(g *Game, d, src *Actor) {
		if src == g.P && g.P != nil {
			if cl := csClass(g.skWeaponSafe()); cl != "" {
				g.cs().Prac[cl] += 3
			}
		}
	})
}

func (g *Game) skWeaponSafe() *Item {
	if g.P == nil {
		return nil
	}
	if m := g.P.Missile(); m != nil && g.P.Weapons()[0].Name == "fist" {
		return m
	}
	return g.P.Weapons()[0]
}

// csClass maps a weapon to one of the technique classes.
func csClass(w *Item) string {
	if w == nil {
		return ""
	}
	switch w.Skill {
	case "longblade", "shortblade", "blade":
		return "blade"
	case "axe":
		return "axe"
	case "spear", "polearm":
		return "spear"
	case "staff", "cudgel", "flail", "whip":
		if w.Name == "fist" {
			return "unarmed"
		}
		return "staff"
	case "fist", "natural":
		return "unarmed"
	case "missile", "bow", "thrown":
		return "bow"
	case "firearm", "pistol":
		return "firearm"
	}
	return ""
}

func csRanged(w *Item) bool {
	c := csClass(w)
	return c == "bow" || c == "firearm"
}

// csTier is the player's technique tier (0-3) for a class: weapon skill or practice, whichever is higher.
func (g *Game) csTier(w *Item) int {
	cl := csClass(w)
	if cl == "" {
		return 0
	}
	sk, pr := g.Skills[w.Skill], g.cs().Prac[cl]
	switch {
	case sk >= 5 || pr >= 100:
		return 3
	case sk >= 3 || pr >= 40:
		return 2
	case sk >= 1 || pr >= 12:
		return 1
	}
	return 0
}

type csStanceDef struct {
	Key           string
	Hit, RHit, DV int
	Fat           float64
}

var csStances = []csStanceDef{
	{"guard", -1, -1, 3, 0},
	{"aggr", 2, 2, -3, 0.4},
	{"evade", -2, -2, 4, 0.25},
	{"aim", 1, 4, -4, 0.1},
}

func csStanceOf(a *Actor, g *Game) *csStanceDef {
	if a != g.P {
		return nil
	}
	k := g.cs().Stance
	for i := range csStances {
		if csStances[i].Key == k {
			return &csStances[i]
		}
	}
	return nil
}

func (g *Game) csFat(x float64) {
	ij := g.ijSt()
	ij.Fat = minf(100, ij.Fat+x)
}

// csHitMod is added to a swing's or shot's to-hit: attacker stance, defender stance, one-shot bonus, weariness.
func (g *Game) csHitMod(a, d *Actor, w *Item) int {
	m := 0
	if s := csStanceOf(a, g); s != nil {
		if csRanged(w) {
			m += s.RHit
		} else {
			m += s.Hit
		}
		g.csFat(s.Fat)
		if g.ijSt().Fat >= 85 && s.Key == "aggr" {
			m -= 4
		}
	}
	if a == g.P {
		m += g.cs().Temp
	}
	if s := csStanceOf(d, g); s != nil {
		m -= s.DV
	}
	return m
}

// csDmgMod is extra damage on a landed melee hit.
func (g *Game) csDmgMod(a, d *Actor, w *Item) int {
	n := 0
	if a == g.P {
		st := g.cs()
		if st.Stance == "aggr" && g.ijSt().Fat < 85 {
			n += 1 + boolInt(g.csTier(w) >= 2)
		}
		if st.Combo >= 5 {
			n += 2
		} else if st.Combo >= 3 {
			n++
		}
	}
	if d.St("cs_offbal") > 0 || d.St("cs_stagger") > 0 {
		n++
	}
	return n
}

func (g *Game) csSee(a, d *Actor) bool { return a == g.P || d == g.P || g.visible(d.X, d.Y) }

// csDefend is called once an attack would land: true means it was dodged or parried (and may be riposted).
func (g *Game) csDefend(d, a *Actor, w *Item, missile bool) bool {
	if d == nil || a == nil || d.HP <= 0 || d.St("stunned") > 0 || d.St("prone") > 0 || d.St("frozen") > 0 ||
		d.St("asleep") > 0 || d.St("cs_stagger") > 0 || d.St("cs_pinned") > 0 {
		return false
	}
	s := csStanceOf(d, g)
	key := ""
	if s != nil {
		key = s.Key
	}
	// dodge: agility and acrobatics, hurt by heavy armour, helped by an evasive stance
	acro := d.Level / 3
	if d == g.P {
		acro = g.Skills["acrobatics"]
	}
	dodge := 3 + 3*maxi(0, d.M(AGI)) + 4*acro - 2*maxi(0, d.AV()-3)
	switch key {
	case "evade":
		dodge += 14
	case "aim":
		dodge -= 8
	case "aggr":
		dodge -= 4
	}
	if d.St("cs_offbal") > 0 {
		dodge /= 2
	}
	if missile {
		dodge /= 2
	}
	dodge = mini(maxi(dodge, 0), 55)
	sk := csSkinFor(g.Genre)
	if g.R.Int(1, 100) <= dodge {
		if d == g.P {
			g.csFat(0.4)
		}
		if g.csSee(a, d) {
			g.Msg(fmt.Sprintf("%s %s %s's attack.", g.The(d), verb(d == g.P, "dodge", "dodges"), g.the(a)))
		}
		return true
	}
	if missile {
		return false
	}
	// shield bash as a reaction
	if sh := wpShieldOf(d); sh != nil && !g.cs().busy {
		ch := 12 + 4*g.skillLv(d, "shield")
		if key == "guard" {
			ch += 18
		}
		if g.R.Int(1, 100) <= ch {
			a.AddSt("cs_stagger", 2)
			if g.csSee(a, d) {
				g.Msg(fmt.Sprintf("{{Y|%s %s %s with %s!}}", g.The(d), verb(d == g.P, "bash", "bashes"), g.the(a), sh.Name))
			}
			g.hurt(a, g.R.Int(1, 3), d, "a "+strings.ToLower(sk.Bash))
			if a.HP <= 0 {
				return true
			}
		}
	}
	// parry needs a real melee weapon
	dw := d.Weapons()[0]
	if dw == nil || dw.Skill == "natural" || dw.Name == "fist" || csRanged(dw) || w == nil || csRanged(w) {
		return false
	}
	parry := 4 + 3*g.skillLv(d, dw.Skill)
	switch key {
	case "guard":
		parry += 18
	case "evade":
		parry += 4
	case "aggr":
		parry -= 4
	}
	if d.St("cs_offbal") > 0 {
		parry /= 2
	}
	parry = mini(maxi(parry, 0), 55)
	if g.R.Int(1, 100) > parry {
		return false
	}
	if g.csSee(a, d) {
		g.Msg(fmt.Sprintf("%s %s the blow aside with %s.", g.The(d), verb(d == g.P, "parry", "parries"), dw.Name))
	}
	if d == g.P {
		g.csFat(0.5)
	}
	st := g.cs()
	if !st.busy && a.HP > 0 && dist(a, d) <= 1 {
		ch := 0.3
		if d == g.P {
			ch = 0.1 + 0.1*float64(g.csTier(dw))
			if key == "guard" {
				ch += 0.3
			}
		}
		if g.R.Chance(ch) {
			st.busy = true
			if g.csSee(a, d) {
				g.Msg("{{G|" + sk.Riposte + "!}}")
			}
			g.swing(d, a, dw)
			st.busy = false
		}
	}
	return true
}

// csOnMiss: a whiff breaks the player's momentum.
func (g *Game) csOnMiss(a, d *Actor) {
	if a == g.P && !g.cs().busy {
		st := g.cs()
		if st.Combo > 1 {
			st.Combo--
		}
	}
}

// csOnHurt: being struck rattles the player's rhythm.
func (g *Game) csOnHurt(d *Actor, n int, src *Actor) {
	if d == g.P && n >= 2 {
		st := g.cs()
		st.Combo /= 2
	}
}

func (g *Game) csBump(a, d *Actor, w *Item) {
	st := g.cs()
	if g.Tick-st.LastT > 12 {
		st.Combo = 0
	}
	st.LastT = g.Tick
	if st.Combo < 9 {
		st.Combo++
	}
	if cl := csClass(w); cl != "" {
		st.Prac[cl]++
	}
	if st.Combo == csComboMax {
		g.Msg("{{W|" + csSkinFor(g.Genre).Combo + " builds: your finisher is ready.}}")
	}
}

func (g *Game) csApply(d *Actor, status string, n int, text string) {
	d.AddSt(status, n)
	if g.visible(d.X, d.Y) {
		g.Msg(fmt.Sprintf("{{Y|%s %s %s.}}", g.The(d), verb(d == g.P, "are", "is"), text))
	}
}

// csOnHit runs after a melee hit lands (after onHit): momentum, practice, class procs.
func (g *Game) csOnHit(a, d *Actor, w *Item, crit bool, sk int) {
	if a != g.P || d.HP <= 0 || g.cs().busy {
		return
	}
	g.csBump(a, d, w)
	t := g.csTier(w)
	if t == 0 {
		return
	}
	ch := 0.12 + 0.06*float64(t)
	if crit {
		ch = 0.6
	}
	if !g.R.Chance(ch) {
		return
	}
	pn := csSkinFor(g.Genre).Passive
	cl := csClass(w)
	switch cl {
	case "blade":
		if !d.Has("undead") && d.Tpl != "golem" {
			g.csApply(d, "bleeding", 6, "bleeding ("+strings.ToLower(pn[cl])+")")
		}
	case "axe":
		st := g.cs()
		for _, o := range g.Cur.Actors {
			if o != d && o.HP > 0 && o != g.P && dist(o, g.P) == 1 && g.Hostile(o) {
				st.busy = true
				g.Msg("{{Y|" + pn[cl] + "!}}")
				g.swing(g.P, o, w)
				st.busy = false
				break
			}
		}
	case "spear":
		g.csApply(d, "cs_pinned", 2+t, "pinned ("+strings.ToLower(pn[cl])+")")
	case "staff":
		g.csApply(d, "cs_offbal", 3+t, "thrown off balance")
	case "unarmed":
		g.csApply(d, "cs_stagger", 2, "staggered")
	}
	if t >= 3 && crit {
		d.AddSt("cs_stagger", 2)
	}
}

// csOnShot runs after a missile or firearm shot wounds a target.
func (g *Game) csOnShot(a, d *Actor, w *Item) {
	if a != g.P || d.HP <= 0 {
		return
	}
	g.csBump(a, d, w)
	if t := g.csTier(w); t > 0 && g.R.Chance(0.1+0.06*float64(t)) {
		g.csApply(d, "cs_pinned", 2+t, "pinned down ("+strings.ToLower(csSkinFor(g.Genre).Passive[csClass(w)])+")")
	}
}

// ---- abilities ----

func (g *Game) csAbilities() []Ability {
	if g.P == nil || g.Cur == nil {
		return nil
	}
	st, sk := g.cs(), csSkinFor(g.Genre)
	var out []Ability
	for i, s := range csStances {
		n := sk.Stance[i]
		if st.Stance == s.Key {
			n += " (on)"
		}
		out = append(out, Ability{ID: "cs_st_" + s.Key, Name: n, Target: "self"})
	}
	if w := g.skWeaponSafe(); w != nil {
		if t := g.csTier(w); t >= 2 {
			out = append(out, Ability{ID: "cs_tech", Name: sk.Tech[csClass(w)], Target: "tile", Cool: g.P.Cool["cs_tech"]})
		}
	}
	out = append(out,
		Ability{ID: "cs_finish", Name: fmt.Sprintf("%s (%s %d/%d)", sk.Finish, sk.Combo, mini(st.Combo, csComboMax), csComboMax), Target: "tile"},
		Ability{ID: "cs_view", Name: "Combat stances and techniques", Target: "self"})
	return out
}

func (g *Game) csUse(id string, tx, ty int) {
	st, sk := g.cs(), csSkinFor(g.Genre)
	switch {
	case strings.HasPrefix(id, "cs_st_"):
		key := id[6:]
		for i, s := range csStances {
			if s.Key != key {
				continue
			}
			if st.Stance == key {
				st.Stance = ""
				g.Msg("You ease out of " + strings.ToLower(sk.Stance[i]) + ".")
			} else {
				st.Stance = key
				g.Msg("{{C|" + sk.Stance[i] + ":}} " + csStanceBlurb[key])
			}
			g.spend(500)
		}
	case id == "cs_view":
		g.csShow()
	case id == "cs_tech":
		g.csTech(tx, ty)
	case id == "cs_finish":
		g.csFinish(tx, ty)
	}
}

var csStanceBlurb = map[string]string{
	"guard": "defence up, parry and shield bash far likelier, a little less accurate.",
	"aggr":  "more accurate and harder hitting, but open to blows, and it tires you.",
	"evade": "dodge sharply up and a mild accuracy cost; heavy armour blunts it.",
	"aim":   "steady, accurate (greatly with missiles) but planted and easy to hit.",
}

func (g *Game) csShow() {
	st, sk := g.cs(), csSkinFor(g.Genre)
	cur := "none"
	for i, s := range csStances {
		if s.Key == st.Stance {
			cur = sk.Stance[i]
		}
	}
	lines := []string{fmt.Sprintf("Stance: %s.  %s: %d/%d%s.  Weariness: %s.", cur, sk.Combo, mini(st.Combo, csComboMax), csComboMax,
		map[bool]string{true: " (finisher ready)", false: ""}[st.Combo >= csComboMax], g.ijFatigueWord()), ""}
	for _, cl := range csClassOrder {
		w := &Item{Skill: map[string]string{"blade": "longblade", "axe": "axe", "spear": "spear", "staff": "staff",
			"unarmed": "fist", "bow": "missile", "firearm": "firearm"}[cl]}
		t := g.csTier(w)
		if cl == "unarmed" {
			t = g.csTier(&Item{Skill: "cudgel", Name: "fist"})
		}
		lines = append(lines, fmt.Sprintf("%-11s tier %d  (%d practice)  proc: %s; active: %s", csClassNames[cl], t, st.Prac[cl], sk.Passive[cl], sk.Tech[cl]))
	}
	lines = append(lines, "", "Tier 1: on-hit technique. Tier 2: active technique. Tier 3: stronger finisher and stagger on crits.",
		"Skill 1/3/5 or 12/40/100 hits and kills unlock tiers.", sk.Riposte+" follows parries; "+sk.Bash+" needs a shield.")
	g.showPopup("Combat", strings.Join(lines, "\n"))
}

func (g *Game) csFoe(tx, ty, r int) *Actor {
	return g.skFoeAt(tx, ty, r)
}

func (g *Game) csUseAmmo(m *Item) bool {
	if !g.wpUseAmmo(m) {
		g.Msg("You're out of " + g.wpAmmoLabel(m) + ".")
		return false
	}
	return true
}

func (g *Game) csTech(tx, ty int) {
	p, st, sk := g.P, g.cs(), csSkinFor(g.Genre)
	w := g.skWeaponSafe()
	if w == nil || g.csTier(w) < 2 {
		g.Msg("You have not mastered a technique with this weapon.")
		return
	}
	cl := csClass(w)
	name := sk.Tech[cl]
	reach := 1
	if cl == "spear" || cl == "blade" {
		reach = 2
	}
	switch cl {
	case "bow", "firearm":
		d := g.skFoeAt(tx, ty, 40)
		if d == nil || !g.csUseAmmo(w) {
			return
		}
		shots := 1 + boolInt(cl == "firearm")
		st.Temp = 2
		for i := 0; i < shots && d.HP > 0; i++ {
			before := d.HP
			g.shoot(p, tx, ty, w)
			if d.HP > 0 && d.HP < before {
				g.csApply(d, "cs_pinned", 4, "pinned ("+strings.ToLower(name)+")")
			}
		}
		st.Temp = 0
	case "axe", "staff":
		foes := g.skAdjacentFoes()
		if len(foes) == 0 {
			g.Msg("No foe is close enough.")
			return
		}
		g.Msg("{{Y|" + name + "!}}")
		st.busy = true
		for _, d := range foes {
			before := d.HP
			g.swing(p, d, w)
			if d.HP > 0 && d.HP < before && cl == "staff" {
				g.csApply(d, "cs_offbal", 5, "knocked off balance")
				d.AddSt("cs_stagger", 1)
			}
		}
		st.busy = false
	default:
		d := g.skFoeAt(tx, ty, reach)
		if d == nil {
			return
		}
		g.Msg("{{Y|" + name + "!}}")
		before := d.HP
		st.Temp, st.busy = 4, true
		g.swing(p, d, w)
		st.Temp, st.busy = 0, false
		if d.HP > 0 && d.HP < before {
			switch cl {
			case "blade":
				g.hurt(d, 2, p, w.Name)
				g.csApply(d, "bleeding", 8, "bleeding")
			case "spear":
				g.csApply(d, "cs_pinned", 5, "pinned")
			case "unarmed":
				g.csApply(d, "cs_stagger", 3, "staggered")
				g.csApply(d, "cs_disarmed", 8, "disarmed")
			}
		}
	}
	p.Cool["cs_tech"] = 6
	g.csFat(1.5)
	g.spend(1000)
}

func (g *Game) csFinish(tx, ty int) {
	p, st, sk := g.P, g.cs(), csSkinFor(g.Genre)
	if st.Combo < csComboMax {
		g.Msg(fmt.Sprintf("{{K|%s is at %d/%d; keep landing hits.}}", sk.Combo, st.Combo, csComboMax))
		return
	}
	w := g.skWeaponSafe()
	tier := 0
	cl := csClass(w)
	d := g.Cur.ActorAt(tx, ty)
	if d == nil || d == p || d.HP <= 0 {
		g.Msg("There is nothing there to target.")
		return
	}
	if dist(d, p) > 1 {
		m := p.Missile()
		if m == nil || dist(d, p) > m.Range+g.Skills["missile"]*2 {
			g.Msg("{{K|Too far away.}}")
			return
		}
		w, cl = m, csClass(m)
		if !g.csUseAmmo(m) {
			return
		}
		tier = g.csTier(m)
		before := d.HP
		st.Temp = 4
		g.shoot(p, tx, ty, m)
		st.Temp = 0
		if d.HP > 0 && d.HP <= before {
			g.hurt(d, g.R.Dice(2, 6)+2*tier, p, m.Name)
		}
	} else {
		if w == nil {
			w = fist()
		}
		tier = g.csTier(w)
		dmg := g.R.Dice(2, 6) + 2*tier + maxi(0, p.M(STR)) + st.Combo
		g.Msg(fmt.Sprintf("{{R|%s! %s %s %s for %d.}}", sk.Finish, g.The(p), "strike", g.the(d), dmg))
		g.hurt(d, dmg, p, w.Name)
		if d.HP > 0 {
			switch cl {
			case "blade":
				g.csApply(d, "bleeding", 10, "bleeding badly")
			case "axe":
				if tier >= 3 && g.R.Chance(0.35) {
					g.dismember(d)
				}
			case "spear":
				g.csApply(d, "cs_pinned", 5, "impaled in place")
			case "staff":
				g.csApply(d, "stunned", 3, "stunned")
			case "unarmed":
				g.csApply(d, "stunned", 3, "stunned")
				g.csApply(d, "cs_disarmed", 10, "disarmed")
			}
		}
	}
	if tier >= 3 && d.HP > 0 {
		d.AddSt("cs_stagger", 3)
	}
	st.Combo, st.Fin = 0, st.Fin+1
	g.csFat(2)
	g.spend(1000)
}
