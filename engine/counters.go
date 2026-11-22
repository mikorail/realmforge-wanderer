package engine

// Rivals that read your habits (prefix ct). The wanderer's way of fighting is recorded over time in
// g.ext("ct") (planet-local): melee or ranged, hit and run, the damage types of the kills, spells,
// stealth, companions and summons, healing items drunk when hurt, kills at night, sleeping rough. Hostile
// parties that hold a grudge (the nemesis, a realm with a bounty on the wanderer, outlaws who hate them)
// study the habits and send a counter: a warded squad against a favourite element (the bw resistance
// flags `bwshield:<type>`), archers and shield-bearers against a melee fighter, fast skirmishers against
// a kiter, hounds that follow the scent trail (senses.go) against a sneak, wards against casters,
// numbers against companions, venom against a drinker of tonics, an ambush at night for someone who
// sleeps rough. Ambushes come from the enter hook, are seeded by zone and day (never g.R), rare, and
// follow a cooldown in days. The dossier lines say "what they know about you"; bounty text mentions
// the habit ("known to hide in shadows").

import (
	"fmt"
	"sort"
	"strings"
)

func init() {
	unLocalKeys = append(unLocalKeys, "ct")
	killHooks = append(killHooks, ctKill)
	turnHooks = append(turnHooks, ctTurn)
	enterHooks = append(enterHooks, ctEnter)
	paEntries = append(paEntries, paEntry{Name: "What they know about you", Keywords: "hunters habits counters bounty nemesis rivals dossier", Cat: "Screen",
		Func: func(g *Game) {
			l := g.ctDossier()
			if len(l) == 0 {
				l = []string{"{{K|No one has watched you long enough to know your ways. Fight a while longer.}}"}
			}
			g.showPopup("What they know about you", strings.Join(l, "\n"))
		}})
}

type ctState struct {
	Style  map[string]int `json:"s,omitempty"`
	Dmg    map[string]int `json:"d,omitempty"`
	Fights int            `json:"f,omitempty"`
	Night  int            `json:"n,omitempty"`
	Heal   int            `json:"h,omitempty"`
	SneakT int            `json:"sk,omitempty"`
	Rest   int            `json:"r,omitempty"`
	PrevH  int            `json:"ph,omitempty"`
	LX     int            `json:"lx,omitempty"`
	LY     int            `json:"ly,omitempty"`
	LKey   string         `json:"lk,omitempty"`
	Still  int            `json:"st,omitempty"`
	Next   int            `json:"nx,omitempty"` // first day another ambush may come
	Sent   int            `json:"se,omitempty"`
	Beat   int            `json:"be,omitempty"`
	Last   string         `json:"la,omitempty"`
}

func (g *Game) ct() *ctState {
	s := g.ext("ct", func() any { return &ctState{} }).(*ctState)
	if s.Style == nil {
		s.Style = map[string]int{}
	}
	if s.Dmg == nil {
		s.Dmg = map[string]int{}
	}
	return s
}

// ---------------------------------------------------------------- recording

var ctElems = map[string]bool{"fire": true, "cold": true, "shock": true, "poison": true, "acid": true}
var ctArcane = map[string]bool{"psychic": true, "dark": true, "holy": true}

func ctKill(g *Game, d, src *Actor) {
	if g.W == nil || g.P == nil || d == nil || d == g.P || src == nil {
		return
	}
	st := g.ct()
	if d.Has("ctx") && src == g.P {
		st.Beat++
		g.Msg("{{G|One of the hunters falls. Their orders will not be carried out.}}")
	}
	byP := src == g.P
	byComp := src != g.P && src.AI == "follower"
	if !byP && !byComp {
		return
	}
	if d.AI != "hostile" && !d.Angry {
		return
	}
	st.Fights++
	if h := g.Hour(); h >= 20 || h < 5 {
		st.Night++
	}
	if byComp {
		if src.Comp == nil {
			st.Style["summ"]++
		} else {
			st.Style["comp"]++
		}
		return
	}
	typ := g.bw().Last[d.ID]
	ws := g.P.Weapons()
	var used *Item
	if len(ws) > 0 {
		used = ws[0]
	}
	d2 := dist(g.P, d)
	if m := g.P.Missile(); m != nil && d2 >= 2 {
		st.Style["ranged"]++
		if d2 >= 4 {
			st.Style["kite"]++
		}
		used = m
		if typ == "" {
			typ = bwTypeOf(m, "")
		}
	} else {
		st.Style["melee"]++
		if typ == "" && used != nil {
			typ = bwTypeOf(used, "")
		}
	}
	if typ != "" {
		st.Dmg[typ]++
		if ctArcane[typ] || ctElems[typ] && (used == nil || used.Elem == "") {
			st.Style["magic"]++
		}
	}
	if g.ss().Sneak {
		st.Style["stealth"]++
	}
}

func ctTurn(g *Game) {
	if g.W == nil || g.P == nil || g.Cur == nil {
		return
	}
	st := g.ct()
	if g.ss().Sneak {
		st.SneakT++
	}
	n := 0
	for _, it := range g.P.Inv {
		if it != nil && it.Kind == "tonic" && it.Effect == "heal" {
			n += maxi(1, it.Count)
		}
	}
	if n < st.PrevH && g.P.HP*5 < g.P.MaxHP*4 {
		st.Heal += st.PrevH - n
	}
	st.PrevH = n
	if g.P.X == st.LX && g.P.Y == st.LY && g.Cur.Key == st.LKey {
		st.Still++
	} else {
		st.Still, st.LX, st.LY, st.LKey = 0, g.P.X, g.P.Y, g.Cur.Key
	}
	wild := g.Cur.Kind != "town" && g.Cur.Kind != "outskirts" && outdoors(g.Cur)
	if st.Still == 8 && wild {
		st.Rest++
	}
	if st.Still == 10 && wild {
		if h := g.Hour(); h >= 21 || h < 5 {
			g.ctAmbush(g.Cur, "rest")
		}
	}
}

func ctEnter(g *Game, zn *Zone) {
	if g.W == nil || g.P == nil {
		return
	}
	g.ct().Still = 0
	g.ctAmbush(zn, "enter")
}

// ---------------------------------------------------------------- what they know

type ctHabit struct {
	ID    string
	Score int
	Bount string // for bounty text
	Know  string // "you ..."
}

var ctElemWords = map[string]string{"fire": "fire", "cold": "frost", "shock": "lightning", "poison": "venom", "acid": "acid"}

func (g *Game) ctHabits() []ctHabit {
	st := g.ct()
	var hs []ctHabit
	add := func(id string, score int, bount, know string) {
		hs = append(hs, ctHabit{id, score, bount, know})
	}
	if f := st.Fights; f >= 6 {
		pc := func(n int) int { return n * 100 / f }
		if v := pc(st.Style["melee"]); v >= 60 {
			add("melee", v, "known to close with the blade", "you fight at arm's length")
		}
		if v := pc(st.Style["ranged"]); v >= 50 {
			add("ranged", v, "known to shoot from afar", "you shoot from afar")
		}
		if v := pc(st.Style["kite"]); v >= 25 {
			add("kite", v+20, "known to strike and fall back", "you strike and fall back")
		}
		for _, e := range []string{"fire", "cold", "shock", "poison", "acid"} {
			if v := pc(st.Dmg[e]); v >= 35 {
				add("elem:"+e, v+25, "known to fight with "+ctElemWords[e], "you fight with "+ctElemWords[e])
			}
		}
		if v := pc(st.Style["magic"]); v >= 35 {
			add("magic", v+15, "known to work spells", "you work spells")
		}
		if v := pc(st.Style["stealth"]); v >= 25 {
			add("stealth", v+20, "known to hide in shadows", "you creep through the shadows")
		}
		if v := pc(st.Style["comp"] + st.Style["summ"]); v >= 35 {
			add("comp", v, "known to let others do the fighting", "you let companions and summons fight for you")
		}
		if v := pc(st.Night); v >= 60 {
			add("night", v-10, "known to strike by night", "you strike by night")
		}
	}
	if st.SneakT >= 80 && st.Style["stealth"]*100 < 25*maxi(1, st.Fights) {
		add("stealth", 40, "known to hide in shadows", "you creep through the shadows")
	}
	if st.Heal >= 5 {
		add("heal", 30+st.Heal, "known to drink deep when hurt", "you drink deep when hurt")
	}
	if st.Rest >= 3 {
		add("rest", 30+st.Rest*3, "known to sleep rough on the open road", "you sleep rough on the open road")
	}
	sort.SliceStable(hs, func(i, j int) bool { return hs[i].Score > hs[j].Score })
	return hs
}

// ctBountyText is the line a bounty notice can add about the wanderer's habits ("" = none known yet).
func (g *Game) ctBountyText() string {
	hs := g.ctHabits()
	if len(hs) == 0 {
		return ""
	}
	var parts []string
	for i, h := range hs {
		if i >= 2 {
			break
		}
		parts = append(parts, strings.TrimPrefix(h.Bount, "known to "))
	}
	return "Known to " + strings.Join(parts, " and ") + "."
}

// ctDossier lines for the Hunters screen (add after the rival section) or the palette popup.
func (g *Game) ctDossier() []string {
	hs := g.ctHabits()
	st := g.ct()
	if len(hs) == 0 {
		return nil
	}
	L := []string{"{{C|What they know about you}}"}
	for i, h := range hs {
		if i >= 4 {
			break
		}
		L = append(L, "  They have seen that "+h.Know+".")
	}
	if gr := g.ctGrudge(); gr != nil {
		L = append(L, "{{W|Holding the grudge}} "+gr.Name)
	} else {
		L = append(L, "{{K|No one holds a grudge hard enough to act on it.}}")
	}
	L = append(L, fmt.Sprintf("{{W|Hunters sent}} %d   {{W|Beaten}} %d", st.Sent, st.Beat))
	if t := g.ctBountyText(); t != "" {
		L = append(L, "{{K|On the posters: "+t+"}}")
	}
	return L
}

// ---------------------------------------------------------------- the grudge

type ctGrudge struct {
	Fac  string
	Name string
}

func (g *Game) ctGrudge() *ctGrudge {
	if n := nxSt(g).Nem; n != nil && n.Status == "hunt" {
		return &ctGrudge{"nemesis", n.Name + "'s hunters"}
	}
	best, bs := 0, 0
	for _, s := range g.W.States {
		if !s.Alive {
			continue
		}
		if b := g.uwBounty(s.ID); b > best {
			best, bs = b, s.ID
		}
	}
	if best > 0 {
		return &ctGrudge{"outlaws", "Bounty hunters hired by " + g.realmName(bs)}
	}
	if g.Rep["outlaws"] <= -600 {
		return &ctGrudge{"outlaws", "A band of " + strings.ToLower(g.Sk.Bandits) + " with a score to settle"}
	}
	return nil
}

// ---------------------------------------------------------------- the counter

type ctMember struct {
	tpl, name string
	flags     []string
	speed     int
	lead      bool
}

func ctSquad(h ctHabit) []ctMember {
	switch {
	case h.ID == "melee":
		return []ctMember{{"archer", "Counter-archer", nil, 0, true}, {"archer", "Counter-archer", nil, 0, false}, {"bandit", "Shield-bearer", []string{"bwshield:slash", "bwshield:blunt"}, 0, false}}
	case h.ID == "ranged" || h.ID == "kite":
		return []ctMember{{"bandit", "Fleet runner", nil, 140, true}, {"bandit", "Fleet runner", nil, 140, false}, {"bandit", "Fleet runner", nil, 140, false}}
	case strings.HasPrefix(h.ID, "elem:"):
		e := strings.TrimPrefix(h.ID, "elem:")
		nm := "Warded " + ctElemWords[e] + "-breaker"
		return []ctMember{{"bandit", nm, []string{"bwshield:" + e}, 0, true}, {"bandit", nm, []string{"bwshield:" + e}, 0, false}, {"archer", nm, []string{"bwshield:" + e}, 0, false}}
	case h.ID == "magic":
		return []ctMember{{"bandit", "Spellbreaker", []string{"bwaff_warded"}, 0, true}, {"bandit", "Spellbreaker", []string{"bwaff_warded"}, 0, false}, {"archer", "Spellbreaker", []string{"bwaff_warded"}, 0, false}}
	case h.ID == "stealth":
		return []ctMember{{"bandit", "Tracker", nil, 0, true}, {"wolf", "hunting hound", nil, 0, false}, {"wolf", "hunting hound", nil, 0, false}}
	case h.ID == "comp":
		return []ctMember{{"bandit", "Line-breaker", nil, 0, true}, {"bandit", "Line-breaker", nil, 0, false}, {"bandit", "Line-breaker", nil, 0, false}, {"archer", "Line-breaker", nil, 0, false}}
	case h.ID == "heal":
		return []ctMember{{"bandit", "Poisoner", []string{"bwaff_venom"}, 0, true}, {"bandit", "Poisoner", []string{"bwaff_venom"}, 0, false}, {"archer", "Poisoner", []string{"bwaff_venom"}, 0, false}}
	case h.ID == "night" || h.ID == "rest":
		return []ctMember{{"bandit", "Night stalker", nil, 0, true}, {"bandit", "Night stalker", nil, 0, false}, {"archer", "Night stalker", nil, 0, false}}
	}
	return nil
}

// ctAmbush may send a counter squad into zn; true when it did.
func (g *Game) ctAmbush(zn *Zone, why string) bool {
	st := g.ct()
	day := DayOf(g.Turn)
	if g.P.Level < 3 || day < st.Next || !nxCanHost(zn) || zn.Z != 0 {
		return false
	}
	gr := g.ctGrudge()
	hs := g.ctHabits()
	if gr == nil || len(hs) == 0 {
		return false
	}
	chance := uint64(10)
	if why == "rest" {
		chance = 30
	}
	roll := HashStr(fmt.Sprintf("%s:ct:amb:%s:%d:%s", g.Seed, zn.Key, day, why))
	if roll%100 >= chance {
		return false
	}
	h := hs[int((roll/100)%uint64(mini(3, len(hs))))]
	if why == "rest" {
		h = ctHabit{ID: "rest", Know: "you sleep rough on the open road"}
		for _, x := range hs {
			if x.ID == "rest" {
				h = x
			}
		}
	}
	squad := ctSquad(h)
	if len(squad) == 0 {
		return false
	}
	r := NewRng(fmt.Sprintf("%s:ct:sq:%s:%d", g.Seed, zn.Key, day))
	far := func(x, y int) bool { return abs(x-g.P.X) >= 8 || abs(y-g.P.Y) >= 6 }
	lx, ly, ok := zn.FreeSpot(r, far)
	if !ok {
		return false
	}
	lvl := clampi(g.P.Level-1, 2, 25)
	for i, m := range squad {
		a := g.Spawn(m.tpl, lvl+boolInt(m.lead), r)
		a.Name, a.Faction, a.AI, a.Angry = m.name, gr.Fac, "hostile", true
		a.Desc = gr.Name + ". They have learned that " + h.Know + "."
		a.Set("ctx")
		for _, f := range m.flags {
			a.Set(f)
		}
		if m.speed > 0 {
			a.Speed = m.speed
		}
		a.X, a.Y = lx+r.Int(-2, 2), ly+r.Int(-2, 2)
		g.nudge(zn, a)
		if i == 0 {
			a.Inv = append(a.Inv, &Item{ID: g.newID(), Kind: "book", Name: "Hunter's brief", Glyph: '■', F: CWhite, Value: 2, Weight: 1,
				Text: fmt.Sprintf("Orders to %s.\n\nThe quarry is %s. %s\nDo not take them in the open the way they like. Take them the way we have planned.", gr.Name, strings.TrimPrefix(h.Bount, "known to "), nonEmpty(g.ctBountyText(), ""))})
		}
		zn.Actors = append(zn.Actors, a)
	}
	st.Next = day + 4 + int(roll%4)
	st.Sent++
	st.Last = h.ID
	g.Msg(fmt.Sprintf("{{R|%s step out of the land and into your road. They have learned that %s.}}", gr.Name, h.Know))
	return true
}
