package engine

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// Rich tooltips (prefix ti). TipInfo is the frame field "tip": detail text for the thing the player is pointing at:
// the look or target cursor (a creature: disposition, hit chance and damage range of the wielded weapon against it,
// statuses with what they do and turns left, what a trader sells and wants; an item on the floor), the selected row of
// the inventory or the trade window (stat deltas against what is worn, value, price), and on the character sheet the
// player's own conditions. TipAt(x, y) gives the same for any map cell so the shell can use it for mouse hover.
// Pure reads: no g.R draws, no writes (the trader price probe restores g.Talk). Lines starting "+" are good, "-" bad,
// "!" a warning; the shell colours by that first character.

type TipInfo struct {
	Title string   `json:"t"`
	Tone  string   `json:"tone,omitempty"` // hostile friendly neutral
	Lines []string `json:"l"`
	X     int      `json:"x,omitempty"` // map cell for cursor tips
	Y     int      `json:"y,omitempty"`
}

// tiStatusText explains the core statuses that the skStatuses table (statuses.go) does not describe.
var tiStatusText = map[string]string{
	"bleeding": "loses health every turn until it stops", "poisoned": "loses health every turn, may end in sickness",
	"burning": "takes fire damage every turn until it burns out or is doused", "frozen": "cannot act well: -6 DV, +2 AV, slowed",
	"stunned": "cannot act, -6 DV", "confused": "-4 to hit, moves at random", "hasted": "+50 speed", "slowed": "-40 speed",
	"mighty": "+6 Strength, +2 damage", "sprint": "+100 speed", "phased": "attacks pass through, cannot be hit by the unphased",
	"blessed": "favoured by the divine", "fed": "carries a meal's bonus", "hzsick": "carries a sickness", "winded": "thin air: half natural healing, more thirst",
	"altsick": "altitude sickness", "heatstroke": "overheated, losing strength", "mired": "stuck in mud or sand, slow to leave", "scalded": "burned by steam",
	"asleep": "defenceless: attacks against it penetrate more", "prone": "knocked down, slow to rise",
}

func tiStatusLine(name string, turns int) string {
	if d := skStatuses[name]; d != nil {
		var p []string
		add := func(v int, w string) {
			if v != 0 {
				p = append(p, fmt.Sprintf("%+d %s", v, w))
			}
		}
		add(d.DV, "DV")
		add(d.Hit, "to hit")
		add(d.AV, "AV")
		add(d.AGI, "Agility")
		add(d.INT, "Intelligence")
		add(d.WIL, "Willpower")
		add(d.EGO, "Ego")
		add(d.MA, "mental armor")
		add(d.Speed, "speed")
		add(d.PV, "penetration against it")
		eff := strings.Join(p, ", ")
		if eff == "" {
			eff = tiStatusText[name]
		}
		if eff == "" {
			eff = "an ongoing condition"
		}
		return fmt.Sprintf("%s (%d turns left): %s", d.Name, turns, eff)
	}
	eff := tiStatusText[name]
	if eff == "" {
		eff = "an ongoing condition"
	}
	return fmt.Sprintf("%s (%d turns left): %s", name, turns, eff)
}

func tiStatuses(a *Actor, prefix string) []string {
	var keys []string
	for k, v := range a.Status {
		if v > 0 {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	var out []string
	for _, k := range keys {
		out = append(out, prefix+tiStatusLine(k, a.Status[k]))
	}
	return out
}

// TipInfo is the tip for the current mode, nil when there is nothing to say.
func (g *Game) TipInfo() (ti *TipInfo) {
	if g.P == nil || g.Cur == nil || g.Dead {
		return nil
	}
	defer func() {
		if recover() != nil {
			ti = nil
		}
	}()
	switch g.Mode {
	case "look", "target":
		return g.TipAt(g.CX, g.CY)
	case "inv":
		list := g.invVisible()
		if len(list) == 0 {
			return nil
		}
		return g.tiItem(list[clampi(g.Sel, 0, len(list)-1)], false)
	case "trade":
		if g.Talk == nil {
			return nil
		}
		list := g.P.Inv
		if g.Tab == 0 {
			list = g.Talk.Inv
		}
		if len(list) == 0 || g.Sel < 0 || g.Sel >= len(list) {
			return nil
		}
		return g.tiItem(list[g.Sel], true)
	case "char":
		if g.Tab != 0 {
			return nil
		}
		l := tiStatuses(g.P, "")
		if len(l) == 0 {
			return nil
		}
		return &TipInfo{Title: "Conditions", Tone: "neutral", Lines: l}
	}
	return nil
}

// TipAt describes map cell (x, y) of the current zone: a visible creature, else an item lying there.
func (g *Game) TipAt(x, y int) (ti *TipInfo) {
	if g.P == nil || g.Cur == nil || !g.Cur.In(x, y) || g.Cur.Seen[y*MW+x] == 0 {
		return nil
	}
	defer func() {
		if recover() != nil {
			ti = nil
		}
	}()
	if a := g.Cur.ActorAt(x, y); a != nil && g.visible(x, y) {
		t := g.tiActor(a)
		t.X, t.Y = x, y
		return t
	}
	if its := g.Cur.ItemsAt(x, y); len(its) > 0 && g.visible(x, y) {
		t := g.tiItem(its[0], false)
		if len(its) > 1 {
			t.Lines = append(t.Lines, fmt.Sprintf("%d more items lie here", len(its)-1))
		}
		t.X, t.Y = x, y
		return t
	}
	return nil
}

// ---- creatures ----

// tiPrimary is the weapon a melee attack would use first, or the missile when the target cursor is for a shot.
func (g *Game) tiPrimary() *Item {
	ws := g.P.Weapons()
	if len(ws) == 0 {
		return nil
	}
	return ws[0]
}

// tiHit is the chance a swing of w by a lands on d, from the same terms as swing().
func (g *Game) tiHit(a, d *Actor, w *Item) int {
	if d.St("phased") > 0 && a.St("phased") == 0 {
		return 0
	}
	bonus := a.M(AGI) + g.skillLv(a, w.Skill) + skStatusToHit(a) + wpHitMod(a, d, w) + g.tcFlankBonus(a, d) + g.tcHighGroundBonus(a, d) + g.ggHitBonus(a, w) + g.csHitMod(a, d, w)
	if a.St("confused") > 0 {
		bonus -= 4
	}
	if a == g.P {
		bonus += thToHitBonus(g)
	}
	n := 0
	for roll := 2; roll <= 19; roll++ {
		if roll+bonus >= d.DV() {
			n++
		}
	}
	return (n + 1) * 100 / 20 // a natural 20 always hits, a 1 never
}

func tiAvg(w *Item) float64 {
	return float64(maxi(w.DN, 1))*float64(maxi(w.DS, 1)+1)/2 + float64(w.DB)
}

func (g *Game) tiActor(a *Actor) *TipInfo {
	t := &TipInfo{Title: Plain(g.the(a))}
	l := &t.Lines
	if a == g.P {
		t.Tone = "friendly"
		*l = append(*l, fmt.Sprintf("Health %d/%d, AV %d, DV %d", a.HP, a.MaxHP, a.AV(), a.DV()))
		*l = append(*l, tiStatuses(a, "")...)
		return t
	}
	hostile := g.Hostile(a)
	switch {
	case hostile:
		t.Tone = "hostile"
	case isFollower(a):
		t.Tone = "friendly"
	default:
		t.Tone = "neutral"
	}
	var who []string
	if a.Title != "" {
		who = append(who, a.Title)
	} else if a.Role != "" {
		who = append(who, a.Role)
	}
	if a.Faction != "" {
		who = append(who, g.FactionName(a.Faction))
	}
	if len(who) > 0 {
		*l = append(*l, strings.Join(who, ", "))
	}
	disp := "neutral"
	switch {
	case isFollower(a):
		disp = "your companion"
	case hostile:
		disp = "hostile, will attack"
	case g.Rep[a.Faction] >= 250:
		disp = "friendly"
	case g.Rep[a.Faction] <= -250:
		disp = "unfriendly"
	}
	*l = append(*l, fmt.Sprintf("Disposition: %s (%s by their people)", disp, strings.ToLower(Plain(RepWord(g.Rep[a.Faction])))))
	*l = append(*l, fmt.Sprintf("Health %d/%d, AV %d, DV %d", a.HP, a.MaxHP, a.AV(), a.DV()))
	if hostile || a.Angry {
		if w := g.tiPrimary(); w != nil {
			dn, ds := maxi(w.DN, 1), maxi(w.DS, 1)
			pv := w.PV + mini(g.P.M(STR), w.MaxStr) + skStatusPV(a)
			need := a.AV() - pv + 2 // a roll of 1..10 must beat this to penetrate
			stop := clampi(need*10, 0, 100)
			hit := g.tiHit(g.P, a, w)
			tone := "+"
			if hit < 50 {
				tone = "-"
			}
			*l = append(*l, fmt.Sprintf("%sHit chance %d%% with %s", tone, hit, Plain(w.Name)))
			*l = append(*l, fmt.Sprintf("Damage %d to %d per penetration (average %.1f), its armor stops about %d%% of rolls", dn+w.DB, dn*ds+w.DB, tiAvg(w), clampi(stop, 0, 100)))
		}
		if len(g.P.Weapons()) > 1 {
			*l = append(*l, "You also strike with your other hands by chance")
		}
	}
	*l = append(*l, tiStatuses(a, "")...)
	if hostile {
		// what it can do to you
		if ws := a.Weapons(); len(ws) > 0 {
			w := ws[0]
			bonus := a.M(AGI) + a.Level/5 + skStatusToHit(a)
			n := 0
			for roll := 2; roll <= 19; roll++ {
				if roll+bonus >= g.P.DV() {
					n++
				}
			}
			*l = append(*l, fmt.Sprintf("!It hits you about %d%% of the time for %d to %d per penetration", (n+1)*5, maxi(w.DN, 1)+w.DB, maxi(w.DN, 1)*maxi(w.DS, 1)+w.DB))
		}
	} else if !isFollower(a) {
		*l = append(*l, g.tiTrader(a)...)
	}
	return t
}

// tiTrader: what a seller carries and what they would pay well for.
func (g *Game) tiTrader(a *Actor) []string {
	var out []string
	if len(a.Inv) > 0 {
		inv := append([]*Item(nil), a.Inv...)
		sort.SliceStable(inv, func(i, j int) bool { return inv[i].Value*maxi(inv[i].Count, 1) > inv[j].Value*maxi(inv[j].Count, 1) })
		var n []string
		for i, it := range inv {
			if i >= 3 {
				n = append(n, fmt.Sprintf("%d more", len(inv)-3))
				break
			}
			n = append(n, Plain(it.Display()))
		}
		out = append(out, fmt.Sprintf("Sells (%d items): %s", len(a.Inv), strings.Join(n, ", ")))
	}
	if len(a.Inv) > 0 || a.Role == "merchant" || a.Role == "smith" || a.Role == "apothecary" || a.Role == "bookseller" || a.Role == "exotic" {
		talk := g.Talk
		defer func() { g.Talk = talk }()
		g.Talk = a
		best, bestRatio := (*Item)(nil), 0.0
		for _, it := range g.P.Inv {
			if g.P.Equipped(it) != nil || it.Value <= 0 {
				continue
			}
			r := float64(g.price(it, false)) / float64(it.Value*maxi(it.Count, 1))
			if r > bestRatio {
				best, bestRatio = it, r
			}
		}
		if best != nil {
			out = append(out, fmt.Sprintf("Pays best for your %s (%d, about %d%% of its worth)", Plain(best.ShownName()), g.price(best, false), int(bestRatio*100)))
		}
	}
	if a.Teach != "" {
		out = append(out, "Teaches "+a.Teach)
	}
	for _, q := range g.Quests {
		if q != nil && !q.Closed && q.GiverID == a.ID {
			out = append(out, "Quest giver: "+Plain(q.Title))
		}
	}
	return out
}

// ---- items ----

func (g *Game) tiEquippedFor(it *Item) *Item {
	switch it.Kind {
	case "weapon":
		if ws := g.P.Weapons(); len(ws) > 0 && ws[0].Kind == "weapon" {
			return ws[0]
		}
		return nil
	case "ranged":
		return g.P.Missile()
	}
	if it.Slot == "" {
		return nil
	}
	for _, bp := range g.P.Body {
		if bp.Slot == it.Slot && !bp.Severed && bp.Item != nil && bp.Item != it {
			return bp.Item
		}
	}
	return nil
}

func tiDelta(name string, v int, goodWhenUp bool) string {
	if v == 0 {
		return ""
	}
	sign := "+"
	if v < 0 == goodWhenUp {
		sign = "-"
	}
	return fmt.Sprintf("%s%s %+d", sign, name, v)
}

func (g *Game) tiItem(it *Item, trade bool) *TipInfo {
	t := &TipInfo{Title: Plain(it.Display()), Tone: "neutral"}
	l := &t.Lines
	if d := icDesc(g, it); d != "" {
		*l = append(*l, d)
	}
	if eq := g.tiEquippedFor(it); eq != nil && g.P.Equipped(it) == nil {
		en := Plain(eq.ShownName())
		*l = append(*l, "Compared with your "+en+":")
		add := func(s string) {
			if s != "" {
				*l = append(*l, s)
			}
		}
		switch it.Kind {
		case "weapon", "ranged":
			avg, eavg := tiAvg(it), tiAvg(eq)
			if d := avg - eavg; d > 0.049 || d < -0.049 {
				sg := "+"
				if d < 0 {
					sg = "-"
				}
				add(fmt.Sprintf("%sdamage per hit %+.1f (average %.1f)", sg, d, avg))
			}
			add(tiDelta("penetration", it.PV-eq.PV, true))
			if it.Kind == "ranged" {
				add(tiDelta("range", it.Range-eq.Range, true))
			}
		default:
			add(tiDelta("armor (AV)", it.AV-eq.AV, true))
			add(tiDelta("defense (DV)", it.DV-eq.DV, true))
			for a := 0; a < 6; a++ {
				add(tiDelta(AttrNames[a], it.Bonus[a]-eq.Bonus[a], true))
			}
			add(tiDelta("weight", itemWeight(it)-itemWeight(eq), false))
		}
	} else if g.P.Equipped(it) != nil {
		*l = append(*l, "Currently equipped")
	} else if (it.Kind == "weapon" || it.Kind == "armor" || it.Kind == "ranged") && it.Slot != "" || it.Kind == "weapon" {
		*l = append(*l, "Nothing worn there: it fills an empty slot")
	}
	*l = append(*l, fmt.Sprintf("Weight %d, worth about %s", itemWeight(it), g.cuFormat(it.Value*maxi(it.Count, 1))))
	if trade && g.Talk != nil {
		buy := g.Tab == 0
		verb := "They would pay"
		if buy {
			verb = "They ask"
		}
		*l = append(*l, fmt.Sprintf("%s %s", verb, g.cuFormat(g.price(it, buy))))
	}
	return t
}

// ---- bridge ----

// TipJSON is the frame field "tip" as a JSON string ("" when there is no tip).
func (g *Game) TipJSON() string {
	t := g.TipInfo()
	if t == nil {
		return ""
	}
	b, err := json.Marshal(t)
	if err != nil {
		return ""
	}
	return string(b)
}

// TipAtJSON is TipAt for mouse hover, exposed to the shell as qud.tipAt(x, y).
func (g *Game) TipAtJSON(x, y int) string {
	t := g.TipAt(x, y)
	if t == nil {
		return ""
	}
	b, err := json.Marshal(t)
	if err != nil {
		return ""
	}
	return string(b)
}
