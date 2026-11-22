package engine

import (
	"fmt"
	"strings"
)

// Culture leftovers (prefix lo): callers for the dvSay kinds that had none (apology, praise, insult, curse,
// oath, gossipOpen), and the culture pack's trade goods as extra goods in the tq ledger.
//
// Existing files call these through one-line hooks (see the report of the wave that added this file).
// Nothing here draws g.R; the only saved thing is the Flag "lobark" on an actor.

func init() {
	turnHooks = append(turnHooks, loTurn)
	enterHooks = append(enterHooks, func(g *Game, zn *Zone) { g.loTqGoods() })
}

// ---------------------------------------------------------------- voice callers

func loSpeaker(a *Actor) bool { return a != nil && a.Species != "" || a != nil && a.Tpl == "npc" }

// loApology: ` "line"` (leading space) for appending to an existing reply, "" when there is no voice.
func (g *Game) loApology(a *Actor) string {
	if !loSpeaker(a) {
		return ""
	}
	if s := g.dvSayQ(a, "apology", nil); s != "" {
		return " " + s
	}
	return ""
}

// loGossipOpen: `"line" ` to put in front of a gossip answer, "" when there is no voice.
func (g *Game) loGossipOpen(a *Actor) string {
	if !loSpeaker(a) {
		return ""
	}
	if s := g.dvSayQ(a, "gossipOpen", nil); s != "" {
		return s + " "
	}
	return ""
}

func loName(a *Actor) string { return "{{c|" + givenOf(a.Name) + "}}" }

// loOath: the NPC answers a vow, a promise or a rite with an oath in their own voice (log line).
func (g *Game) loOath(a *Actor) {
	if !loSpeaker(a) {
		return
	}
	if s := g.dvSayQ(a, "oath", nil); s != "" {
		g.Msg(fmt.Sprintf("%s says: %s", loName(a), s))
	}
}

// loPraise: the NPC praises the player (quest finished).
func (g *Game) loPraise(a *Actor) {
	if !loSpeaker(a) {
		return
	}
	if s := g.dvSayQ(a, "praise", nil); s != "" {
		g.Msg(fmt.Sprintf("%s says: %s", loName(a), s))
	}
}

// loCrowdPraise: after a won bout somebody nearby (the nearest townsperson or guard in sight) cheers in their voice.
func (g *Game) loCrowdPraise() {
	if g.Cur == nil || g.P == nil {
		return
	}
	var best *Actor
	bd := 1 << 30
	for _, a := range g.Cur.Actors {
		if a == g.P || a.HP <= 0 || !loSpeaker(a) || a.AI == "hostile" || g.Hostile(a) {
			continue
		}
		d := maxi(absi(a.X-g.P.X), absi(a.Y-g.P.Y))
		if d < bd || d == bd && best != nil && a.ID < best.ID {
			best, bd = a, d
		}
	}
	if best != nil && bd <= 20 {
		g.loPraise(best)
	}
}

// loHurt: somebody who was friendly and is struck begs off before the fight turns.
func (g *Game) loHurt(d *Actor) {
	if d == nil || !loSpeaker(d) || d.AI == "guard" || d.AI == "hostile" || d.Role == "guard" {
		return
	}
	if s := g.dvSayQ(d, "apology", nil); s != "" {
		g.Msg(fmt.Sprintf("%s cries: %s", loName(d), s))
	}
}

// loTurn: hostile people curse or insult as a fight starts (once each, one per few turns).
func loTurn(g *Game) {
	if g.Cur == nil || g.P == nil || g.Dead || !g.Pref("barks") {
		return
	}
	st := ct2St(g)
	if g.Turn-st.LastBark < 6 {
		return
	}
	for _, a := range g.Cur.Actors {
		if a == g.P || a.HP <= 0 || a.Species == "" || a.Has("lobark") || a.AI == "follower" || !g.Hostile(a) {
			continue
		}
		if !g.visible(a.X, a.Y) || maxi(absi(a.X-g.P.X), absi(a.Y-g.P.Y)) > 9 {
			continue
		}
		a.Set("lobark")
		kind := "insult"
		if HashStr(fmt.Sprintf("%s:lobark:%d", g.W.Seed, a.ID))%2 == 0 {
			kind = "curse"
		}
		s := g.dvSay(a, kind, nil)
		if s == "" {
			s = g.dvSay(a, "insult", nil)
		}
		if s == "" {
			continue
		}
		st.LastBark = g.Turn
		g.Msg(fmt.Sprintf("{{R|%s}} snarls: \"%s\"", givenOf(a.Name), s))
		return
	}
}

// ---------------------------------------------------------------- culture trade goods in the tq ledger

const loTqBase = 1000

var (
	loTqSeed  string
	loTqList  []tqGood
	loTqState *Game
)

var loTqProd = map[string]string{"food": "farming", "drug": "farming", "timber": "farming", "tool": "industry", "ore": "mining", "metal": "mining",
	"craft": "crafts", "cloth": "crafts", "luxury": "trade"}

func loSlug(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			b.WriteRune(r)
		} else if b.Len() > 0 && !strings.HasSuffix(b.String(), "-") {
			b.WriteByte('-')
		}
	}
	return strings.Trim(b.String(), "-")
}

// loTqBuild lists every culture-pack trade good of the world, in a stable order (the index goes into saved items).
func loTqBuild(w *WorldData) []tqGood {
	var out []tqGood
	for i := range w.Cultures {
		cu := &w.Cultures[i]
		p := w.cpOf(i)
		if p == nil || p.Material == nil {
			continue
		}
		for _, tg := range p.Material.TradeGoods {
			if tg.Name == "" {
				continue
			}
			wt := cgGoodWeight[tg.Kind]
			if wt == 0 {
				wt = 2
			}
			prod := loTqProd[tg.Kind]
			if prod == "" {
				prod = "crafts"
			}
			per := 0
			if tg.Kind == "food" {
				per = 60
			}
			out = append(out, tqGood{ID: fmt.Sprintf("cg:%d:%s", cu.ID, loSlug(tg.Name)), Name: tg.Name, Base: maxi(2, tg.Value), Wt: wt,
				Perish: per, Prod: prod, Dem: "trade", Ws: ""})
		}
	}
	return out
}

func loTqEnsure(g *Game) {
	if g == nil || g.W == nil {
		return
	}
	loTqState = g
	if loTqSeed == g.W.Seed && loTqList != nil {
		return
	}
	loTqSeed = g.W.Seed
	loTqList = loTqBuild(g.W)
	if loTqList == nil {
		loTqList = []tqGood{}
	}
}

// loTqCulture: the culture id in a culture good's ID ("cg:<culture>:<slug>"), or -1.
func loTqCulture(d *tqGood) int {
	var c int
	if _, err := fmt.Sscanf(d.ID, "cg:%d:", &c); err != nil {
		return -1
	}
	return c
}

// loTqGoods: the culture goods on offer here: the local culture's, plus any culture good you carry.
func (g *Game) loTqGoods() []*tqGood {
	loTqEnsure(g)
	if len(loTqList) == 0 {
		return nil
	}
	want := map[int]bool{}
	if bd := g.localBurg(); bd != nil {
		want[bd.Culture] = true
	}
	held := map[int]bool{}
	scan := func(inv []*Item) {
		for _, it := range inv {
			if it != nil && it.Extra != nil && it.Extra["tq"] == 1 && it.Extra["tq:i"] >= loTqBase {
				held[it.Extra["tq:i"]-loTqBase] = true
			}
		}
	}
	if g.P != nil {
		scan(g.P.Inv)
	}
	if sp := g.sh().Ship; sp != nil {
		scan(sp.Hold)
	}
	var out []*tqGood
	for i := range loTqList {
		d := &loTqList[i]
		if want[loTqCulture(d)] || held[i] {
			out = append(out, d)
		}
	}
	return out
}

// loTqIdx: the saved index of a culture good, or -1 when d is not one.
func loTqIdx(d *tqGood) int {
	for i := range loTqList {
		if &loTqList[i] == d {
			return loTqBase + i
		}
	}
	return -1
}

func loTqAt(i int) *tqGood {
	if i < 0 || i >= len(loTqList) {
		return nil
	}
	return &loTqList[i]
}

func loTqByID(id string) *tqGood {
	if !strings.HasPrefix(id, "cg:") {
		return nil
	}
	if loTqList == nil && loTqState != nil {
		loTqEnsure(loTqState)
	}
	for i := range loTqList {
		if loTqList[i].ID == id {
			return &loTqList[i]
		}
	}
	return nil
}
