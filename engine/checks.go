package engine

import (
	"fmt"
	"sort"
	"strings"
)

// Skill checks and inner voices for conversations (Fallout / Disco Elysium style).
// A check is 2d6 + modifier against a difficulty; the modifier is an attribute modifier
// plus two per rank of the related skill.

const (
	dcTrivial     = 6
	dcEasy        = 8
	dcMedium      = 10
	dcChallenging = 12
	dcHeroic      = 14
	dcLegendary   = 16
)

func diffName(dc int) string {
	switch {
	case dc <= dcTrivial:
		return "Trivial"
	case dc <= dcEasy:
		return "Easy"
	case dc <= dcMedium:
		return "Medium"
	case dc <= dcChallenging:
		return "Challenging"
	case dc <= dcHeroic:
		return "Heroic"
	}
	return "Legendary"
}

// chanceFor is the exact probability that 2d6+mod >= dc.
func chanceFor(mod, dc int) float64 {
	n := 0
	for a := 1; a <= 6; a++ {
		for b := 1; b <= 6; b++ {
			if a+b+mod >= dc {
				n++
			}
		}
	}
	return float64(n) / 36
}

func (g *Game) roll2d6Check(mod, dc int) bool { return g.R.Dice(2, 6)+mod >= dc }

// checkKind describes how a named check derives its modifier.
type checkKind struct {
	label string
	attrs []int  // best attribute modifier among these is used
	skill string // optional skill id whose rank adds +2 each
}

var checkKinds = map[string]checkKind{
	"persuasion": {"Persuasion", []int{EGO}, "persuasion"},
	"lore":       {"Lore", []int{INT}, "lore"},
	"survival":   {"Survival", []int{WIL}, "survival"},
	"tinkering":  {"Tinkering", []int{INT}, "tinkering"},
	"acrobatics": {"Acrobatics", []int{AGI}, "acrobatics"},
	"endurance":  {"Endurance", []int{TOU}, "endurance"},
	"intimidate": {"Intimidate", []int{STR, EGO}, ""},
	"empathy":    {"Empathy", []int{WIL}, ""},
	"perception": {"Perception", []int{INT}, ""},
	"bluff":      {"Bluff", []int{EGO}, ""},
	"streetwise": {"Streetwise", []int{AGI}, ""},
}

func (g *Game) checkMod(kind string) int {
	k := checkKinds[kind]
	best := -99
	for _, at := range k.attrs {
		if m := g.P.M(at); m > best {
			best = m
		}
	}
	if k.skill != "" {
		best += 2 * g.Skills[k.skill]
	}
	return best + thCheckBonus(g, kind)
}

// checkSpec is one contextual check: what it tests, how hard it is, and what happens.
type checkSpec struct {
	id    string
	kind  string
	dc    int
	quote string
	win   func() string
	lose  func() string
}

func (g *Game) checkLabel(kind string, dc int) string {
	pct := int(chanceFor(g.checkMod(kind), dc)*100 + 0.5)
	return fmt.Sprintf("{{c|[%s · %s %d%%]}}", checkKinds[kind].label, diffName(dc), pct)
}

func (g *Game) checkOption(a *Actor, s checkSpec) Option {
	return Option{g.checkLabel(s.kind, s.dc) + " " + s.quote, func() {
		a.Set("chk:" + s.id)
		mod := g.checkMod(s.kind)
		if g.roll2d6Check(mod, s.dc) {
			g.gainXP(4 + s.dc)
			g.talkMenu("{{G|[" + checkKinds[s.kind].label + ": Success]}}\n\n" + s.win())
			return
		}
		g.talkMenu("{{R|[" + checkKinds[s.kind].label + ": Failure]}}\n\n" + s.lose())
	}}
}

func (g *Game) deadRulers(a *Actor) []PersonData {
	var out []PersonData
	for _, p := range g.W.People {
		if !p.Alive && p.Died > 0 && (a.State != 0 && p.State == a.State || a.State == 0 && p.Burg == g.Cur.Burg) {
			out = append(out, p)
		}
	}
	return out
}

func (g *Game) realmEvents(a *Actor) []EventData {
	var out []EventData
	st := a.State
	if st == 0 && g.W.In(g.Cur.WX, g.Cur.WY) {
		st = g.W.State[g.W.Idx(g.Cur.WX, g.Cur.WY)]
	}
	for _, e := range g.W.Events {
		if e.St == st && st != 0 {
			out = append(out, e)
		}
	}
	return out
}

// nearestSite finds the closest dungeon/tower or ruined burg to the current parasang.
func (g *Game) nearestSite() (text string, p int, ok bool) {
	w := g.W
	cx, cy := g.Cur.WX, g.Cur.WY
	best := 1 << 30
	for dy := -10; dy <= 10; dy++ {
		for dx := -12; dx <= 12; dx++ {
			if !w.In(cx+dx, cy+dy) {
				continue
			}
			pp := w.Idx(cx+dx, cy+dy)
			if w.Burg[pp] > 0 {
				continue
			}
			d := abs(dx) + abs(dy)
			if s := g.parasangSite(pp); s != nil && d < best && !g.SecretSet["dungeon:"+s.Name+" lies near "+w.RegionName(pp)+"."] {
				best, p, ok = d, pp, true
				what := "a dungeon"
				if s.Kind == "tower" {
					what = fmt.Sprintf("a tower of %d floors", s.Depth)
				}
				text = fmt.Sprintf("%s is %s lying near %s.", s.Name, what, w.RegionName(pp))
			}
		}
	}
	for _, b := range w.Burgs {
		if b.Alive || b.Layer != "surface" {
			continue
		}
		d := abs(b.P%w.W-cx) + abs(b.P/w.W-cy)
		if d < best && d <= 14 {
			best, p, ok = d, b.P, true
			text = fmt.Sprintf("The ruins of %s lie near %s. Its cellars were never emptied.", b.Name, w.RegionName(b.P))
		}
	}
	return
}

func (g *Game) checkOptions(a *Actor) []Option {
	if a == nil {
		return nil
	}
	var specs []checkSpec
	faction := g.FactionName(a.Faction)
	add := func(s checkSpec) {
		if !a.Has("chk:" + s.id) {
			specs = append(specs, s)
		}
	}

	if a.Angry && !a.Has("chk:calm") {
		add(checkSpec{"calm", "persuasion", dcChallenging, "\"Easy now. I didn't come here for blood.\"",
			func() string {
				a.Angry = false
				return a.Name + " lowers their guard, breathing hard. \"...Fine. Fine. Say your piece.\""
			},
			func() string {
				g.AddRep(a.Faction, -20, " for goading them")
				return a.Name + " spits. \"Pretty words. They won't save you.\""
			}})
	}

	switch a.Role {
	case "merchant", "smith", "apothecary", "bookseller", "exotic":
		add(checkSpec{"haggle", "persuasion", dcMedium, "\"Surely a friend of the realm pays less?\"",
			func() string {
				a.Set("disc")
				return "They grumble, then wave a hand. \"Fine. Trade-friend's prices. Don't tell the others.\"\n{{G|Their prices for you are lowered.}}"
			},
			func() string {
				a.Set("markup")
				g.AddRep(a.Faction, -10, " for haggling badly")
				return "They snort. \"Insulting. My prices just went up for your trouble.\"\n{{R|Their prices for you are raised.}}"
			}})
	case "guard":
		add(checkSpec{"patrol", "intimidate", dcMedium, "\"Tell me where the patrols have been hunting. Now.\"",
			func() string {
				if t, p, ok := g.nearestSite(); ok && g.addSecret(t, "dungeon", p, g.W.Year) {
					return "They swallow and lower their voice. \"The watch avoids it. " + t + "\"\n\n{{W|(Recorded in your journal.)}}"
				}
				return "\"All quiet,\" they mutter. \"I've told you everything I know.\""
			},
			func() string {
				g.AddRep(a.Faction, -25, " for threatening the watch")
				return "\"Threatening a warden?\" Their hand finds their hilt. \"Move along.\""
			}})
		add(checkSpec{"officer", "bluff", dcEasy, "\"Captain sent me. Stand aside, I'm expected.\"",
			func() string {
				g.AddRep(a.Faction, 30, " for being waved through")
				return "They straighten and salute out of habit. \"Of course. My apologies.\""
			},
			func() string {
				g.AddRep(a.Faction, -20, " for a clumsy lie")
				return "They squint. \"The captain is on the wall. And you aren't anyone's officer.\""
			}})
	case "elder", "notable":
		add(checkSpec{"chronicle", "lore", dcMedium, "\"I have read of this realm. Correct me where I am wrong.\"",
			func() string {
				evs := g.realmEvents(a)
				if len(evs) == 0 {
					return "They nod, impressed, but there is little to add."
				}
				e := evs[g.R.Int(0, len(evs)-1)]
				g.addSecret(fmt.Sprintf("In the year %d, %s.", e.Y, lowerFirst(e.Text)), "history", e.P, e.Y)
				return fmt.Sprintf("Their eyes brighten. \"You know your history. Then you will want the part the books skip: in the year %d, %s.\"\n\n{{W(Recorded in your journal.)}}", e.Y, lowerFirst(e.Text))
			},
			func() string {
				g.AddRep(a.Faction, -10, " for misquoting their history")
				return "They correct you three times, each more coldly than the last."
			}})
		add(checkSpec{"grief", "empathy", dcMedium, "[Watch their face. Something is held back.]",
			func() string {
				d := g.deadRulers(a)
				if len(d) == 0 {
					return "You see only tiredness, the plain weight of years."
				}
				p := d[g.R.Int(0, len(d)-1)]
				cause := nonEmpty(p.Cause, "an unnamed end")
				g.AddRep(a.Faction, 30, " for your compassion")
				g.addSecret(fmt.Sprintf("%s, %s, died in the year %d (%s). The elders of %s still grieve.", p.Name, p.Role, p.Died, cause, faction), "grief", -1, p.Died)
				return fmt.Sprintf("You say nothing. After a while they do. \"%s. %s. Gone in the year %d: %s. I was there.\"", p.Name, titleCase(p.Role), p.Died, cause)
			},
			func() string {
				g.AddRep(a.Faction, -15, " for prying")
				return "They turn their face away. \"That is not yours to ask.\""
			}})
	case "priest":
		add(checkSpec{"theology", "lore", dcMedium, "\"Walk me through your doctrine. I know the old arguments.\"",
			func() string {
				rel := g.W.relByID[a.Religion]
				if rel == nil {
					return "They smile. \"Our faith is older than words.\""
				}
				g.Rep[fmt.Sprintf("faith:%d", rel.ID)] += 40
				t := fmt.Sprintf("The %s holds that %s Its deity is %s.", rel.Name, lowerFirst(nonEmpty(rel.Doctrine, rel.Theology)), nonEmpty(rel.Deity, "unnamed"))
				g.addSecret(Plain(t), "faith", -1, g.W.Year)
				return "They light up. \"A rare traveller!\" " + t + "\n\n{{W(Recorded in your journal.)}}"
			},
			func() string {
				if a.Religion > 0 {
					g.Rep[fmt.Sprintf("faith:%d", a.Religion)] -= 30
				}
				return "\"Heresy,\" they sigh, \"born of half-reading.\""
			}})
	case "villager":
		add(checkSpec{"rumor", "streetwise", dcEasy, "\"Come on. Everyone knows a place nobody goes. Where?\"",
			func() string {
				if t, p, ok := g.nearestSite(); ok && g.addSecret(t, "dungeon", p, g.W.Year) {
					return "They glance around and lower their voice. \"" + t + " Folks don't say it twice.\"\n\n{{W(Recorded in your journal; marked on the map.)}}"
				}
				return "They shrug. \"Nothing nearby that I'd tell a stranger.\""
			},
			func() string {
				g.AddRep(a.Faction, -5, " for pushing")
				return "They clam up. \"Don't know what you mean.\""
			}})
	}

	if len(specs) < 3 {
		dc := dcEasy
		if len(a.Inv) > 0 {
			dc = dcMedium
		}
		add(checkSpec{"notice", "perception", dc, "[Study them closely.]",
			func() string {
				var bits []string
				if a.Caste != "" {
					bits = append(bits, fmt.Sprintf("the bearing of the %s caste", a.Caste))
				}
				if a.House != "" {
					bits = append(bits, "the signet of House "+a.House)
				}
				if len(a.Inv) > 0 {
					bits = append(bits, "a concealed "+a.Inv[g.R.Int(0, len(a.Inv)-1)].Name)
				}
				if len(bits) == 0 {
					return "Nothing remarkable. That, too, tells you something."
				}
				return "You notice " + strings.Join(bits, ", ") + "."
			},
			func() string {
				g.AddRep(a.Faction, -5, " for staring")
				return "They catch you staring. \"Something wrong?\""
			}})
	}
	if len(specs) > 3 {
		specs = specs[:3]
	}
	opts := make([]Option, 0, len(specs)+1)
	for _, s := range specs {
		opts = append(opts, g.checkOption(a, s))
	}
	// non-check: bribe merchants (ranked after the checks so the cap above never drops it)
	switch a.Role {
	case "merchant", "smith", "apothecary", "bookseller", "exotic":
		if !a.Has("chk:bribe") && !a.Has("disc") {
			cost := 25 + g.P.Level*10
			opts = append(opts, Option{fmt.Sprintf("{{c|[Bribe · %d %s]}} (slides coins across) \"For your trouble.\"", cost, g.Sk.Currency), func() {
				a.Set("chk:bribe")
				if g.P.Gold < cost {
					g.talkMenu("You pat your purse. It is lighter than your ambition.")
					return
				}
				g.P.Gold -= cost
				a.Gold += cost
				a.Set("disc")
				g.talkMenu("The coins vanish. \"Ah. A reasonable " + g.Sk.Ritual + "-friend.\"\n{{G|Their prices for you are lowered.}}")
			}})
		}
	}
	return opts
}

// innerVoices: passive skill/attribute voices. A voice speaks when your passive score
// (5 + modifier) reaches its difficulty.
func (g *Game) innerVoices(a *Actor) []string {
	if a == nil {
		return nil
	}
	type voice struct {
		name string
		kind string
		dc   int
		text string
	}
	var vs []voice
	w := g.W
	faction := g.FactionName(a.Faction)
	if d := g.deadRulers(a); len(d) > 0 {
		p := d[int(NewRng(fmt.Sprint("iv", a.ID)).Int(0, len(d)-1))]
		vs = append(vs, voice{"LORE", "lore", dcEasy, fmt.Sprintf("The banner behind them is %d years out of date. They still fly it for %s, who died in the year %d.", maxi(1, w.Year-p.Died), p.Name, p.Died)})
	}
	if rel := w.relByID[a.Religion]; rel != nil {
		vs = append(vs, voice{"LORE", "lore", dcMedium, fmt.Sprintf("The charm at their throat is a sign of the %s. They pray to %s, and they hide a doubt.", rel.Name, nonEmpty(rel.Deity, "no one in particular"))})
	}
	if a.Title != "" {
		vs = append(vs, voice{"EMPATHY", "empathy", dcMedium, fmt.Sprintf("They wear the title of %s like a coat a size too big. Someone is watching how they carry it.", a.Title)})
	}
	if a.House != "" || a.Caste != "" {
		who := strings.TrimSpace(a.Caste + " " + a.House)
		vs = append(vs, voice{"PERCEPTION", "perception", dcEasy, fmt.Sprintf("Their hands and manner say %s. They are used to being obeyed, or to not being noticed.", who)})
	}
	if g.Rep[a.Faction] <= -100 {
		vs = append(vs, voice{"EMPATHY", "empathy", dcTrivial, fmt.Sprintf("They have not forgotten what you did to %s. Tread softly.", faction)})
	}
	if a.Gold > 0 {
		vs = append(vs, voice{"STREETWISE", "streetwise", dcEasy, fmt.Sprintf("A purse of about %d %s hangs at their belt, and they keep touching it.", a.Gold, g.Sk.Currency)})
	}
	if g.Cur != nil && g.Cur.Name != "" {
		vs = append(vs, voice{"SURVIVAL", "survival", dcTrivial, fmt.Sprintf("%s keeps its wells and woodpiles where a practised eye can find them; you could live off this land.", g.Cur.Name)})
	}
	var out []string
	r := NewRng(fmt.Sprintf("voices:%d:%d", a.ID, g.Turn/600))
	idx := make([]int, len(vs))
	for i := range idx {
		idx[i] = i
	}
	sort.Slice(idx, func(i, j int) bool { return vs[idx[i]].dc > vs[idx[j]].dc })
	for _, i := range idx {
		v := vs[i]
		if 5+g.checkMod(v.kind) < v.dc || !r.Chance(0.8) {
			continue
		}
		out = append(out, fmt.Sprintf("{{m|%s [%s: Success]}} — %s", v.name, diffName(v.dc), v.text))
		if len(out) == 2 {
			break
		}
	}
	return out
}
