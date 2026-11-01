package engine

import (
	"fmt"
	"strings"
)

// Insight (Bloodborne-like) and Hearts (Zelda-like heart pieces). Insight is gained by beholding
// eldritch or legendary bosses, descending into deep levels, reading forbidden books and communing
// at shrines; it is spent at eldritch shrines and odd merchants. At 5/15/30/50 the world changes:
// glyphs on walls, veiled horrors become visible, mad prophets say more, secret doors show, but
// horrors also gather and the mind frays (status "frenzied"). Hearts: 4 pieces = +HP and a key
// for Heart Seals (gates.go). Everything hangs off hooks from init(); state is g.ext("in").

func init() {
	zoneHooks = append(zoneHooks, inZone)
	enterHooks = append(enterHooks, inEnter)
	turnHooks = append(turnHooks, inTurn)
	killHooks = append(killHooks, inKill)
	talkHooks = append(talkHooks, inTalk)
	sideHooks = append(sideHooks, inSide)
	mainKeys["U"] = inKeyU
}

// inComposureHit is wired by the coordinator to the Composure system (thoughts.go); negative heals.
var inComposureHit = func(g *Game, n int) {}

// inSightN mirrors the player's current insight so inSeesHidden needs no Game.
var inSightN int

var inTiers = [4]int{5, 15, 30, 50}

const (
	inCostRelic = 5
	inCostHeart = 8
	inCostCalm  = 2
	inHeartHP   = 6
)

type inSpot struct {
	Zone   string `json:"z"`
	Idx    int    `json:"i"`
	K      string `json:"k"` // gate secret glyph eldritch heart
	Req    string `json:"rq,omitempty"`
	N      int    `json:"n,omitempty"`
	Skill  string `json:"sk,omitempty"`
	Reward string `json:"rw,omitempty"` // loot lore heart shortcut gold
	Text   string `json:"tx,omitempty"`
	Direct bool   `json:"d,omitempty"` // freestanding seal: reward is handed over on opening
	Done   bool   `json:"dn,omitempty"`
}

type inState struct {
	Ins     int                `json:"ins,omitempty"`
	Peak    int                `json:"pk,omitempty"`
	Pieces  int                `json:"pc,omitempty"`
	Applied int                `json:"ap,omitempty"` // hearts already added to max HP
	Beheld  map[int]bool       `json:"bh,omitempty"`
	Zones   map[string]bool    `json:"zn,omitempty"`
	Books   map[string]bool    `json:"bk,omitempty"`
	Enc     map[string]bool    `json:"en,omitempty"`
	Prophet map[string]bool    `json:"pr,omitempty"`
	Feats   map[string]bool    `json:"ft,omitempty"`
	Spots   map[string]*inSpot `json:"sp,omitempty"`
	Deep    int                `json:"dp,omitempty"`
	Buys    int                `json:"by,omitempty"`
}

func inSt(g *Game) *inState {
	s := g.ext("in", func() any { return &inState{} }).(*inState)
	if s.Beheld == nil {
		s.Beheld = map[int]bool{}
	}
	if s.Zones == nil {
		s.Zones = map[string]bool{}
	}
	if s.Books == nil {
		s.Books = map[string]bool{}
	}
	if s.Enc == nil {
		s.Enc = map[string]bool{}
	}
	if s.Prophet == nil {
		s.Prophet = map[string]bool{}
	}
	if s.Feats == nil {
		s.Feats = map[string]bool{}
	}
	if s.Spots == nil {
		s.Spots = map[string]*inSpot{}
	}
	inSightN = s.Ins
	return s
}

func inTier(n int) int {
	t := 0
	for _, v := range inTiers {
		if n >= v {
			t++
		}
	}
	return t
}

// Insight is the player's current insight; Hearts the whole hearts carried.
func inInsight(g *Game) int { return inSt(g).Ins }
func inHearts(g *Game) int  { return inSt(g).Pieces / 4 }

// inSeesHidden reports whether the player can see a veiled eldritch creature (render/visibility hook).
func inSeesHidden(a *Actor) bool {
	return a == nil || !a.Has("in_veil") || inSightN >= inTiers[1]
}

func inEldritch(a *Actor) bool {
	return a != nil && (a.Faction == "eldritch" || a.Tpl == "horror" || a.Has("in_eldr"))
}

var inTierText = [5]string{
	"",
	"{{M|Faint glyphs surface on the walls around you.}}",
	"{{M|Shapes that were never there stand revealed. Something has noticed you noticing.}}",
	"{{M|Seams open in stone. The deep things lean closer. Your thoughts begin to fray.}}",
	"{{M|You see the whole of it now, and it sees you.}}",
}

// inGain adds insight and fires the world changes at the thresholds.
func inGain(g *Game, n int, why string) {
	if n == 0 {
		return
	}
	st := inSt(g)
	old := inTier(st.Ins)
	st.Ins = maxi(0, st.Ins+n)
	if st.Ins > st.Peak {
		st.Peak = st.Ins
	}
	inSightN = st.Ins
	if n > 0 {
		g.Msg(fmt.Sprintf("{{M|+%d insight}}%s", n, why))
	}
	if nt := inTier(st.Ins); nt > old && n > 0 {
		g.Msg(inTierText[nt])
		if g.Cur != nil {
			inReveal(g, g.Cur)
		}
	}
}

// inHeartPiece grants one heart piece; every fourth makes a heart (+max HP).
func inHeartPiece(g *Game, why string) {
	st := inSt(g)
	st.Pieces++
	g.Msg(fmt.Sprintf("{{R|♥ Heart piece}} (%d/4)%s", st.Pieces%4, why))
	for st.Pieces/4 > st.Applied {
		st.Applied++
		g.P.MaxHP += inHeartHP
		g.P.HP += inHeartHP
		g.Msg(fmt.Sprintf("{{R|♥ The pieces join: a heart. +%d max HP.}} You carry %d.", inHeartHP, st.Applied))
	}
}

func inFeat(g *Game, id, text string) {
	st := inSt(g)
	if st.Feats[id] {
		return
	}
	st.Feats[id] = true
	g.Msg("{{Y|Feat: " + text + "}}")
	inHeartPiece(g, "")
}

var inBookWords = []string{"forbidden", "eldritch", "necro", "occult", "void", "abyss", "unnamed", "whisper", "black", "heresy", "heretic", "unspeakable", "hidden", "madness", "star", "mourning"}

func inForbidden(name string) bool {
	n := strings.ToLower(name)
	for _, w := range inBookWords {
		if strings.Contains(n, w) {
			return true
		}
	}
	return false
}

func inEnter(g *Game, zn *Zone) {
	st := inSt(g)
	if zn.Kind == "dungeon" || zn.Kind == "tower" {
		depth := zn.Z
		if zn.Kind == "tower" {
			depth = -100 - zn.Z
		}
		if depth > st.Deep {
			st.Deep = depth
		}
		if !st.Zones[zn.Key] {
			if zn.Kind == "dungeon" && zn.Z >= 1 {
				if s := g.siteAt(zn.WX, zn.WY, zn.ZX, zn.ZY); s != nil && s.Theme == "abyss" {
					st.Zones[zn.Key] = true
					inGain(g, 2, ": the abyss looks back.")
				} else if zn.Z >= 6 {
					st.Zones[zn.Key] = true
					inGain(g, 1, ": the deep places teach.")
				}
			} else if zn.Kind == "tower" && depth >= 5 {
				st.Zones[zn.Key] = true
				inGain(g, 1, ": the upper air is thin and strange.")
			}
		}
		inEncounters(g, zn)
	}
	if st.Deep >= 5 {
		inFeat(g, "deep5", "reached the fifth depth")
	}
	if st.Deep >= 10 {
		inFeat(g, "deep10", "reached the tenth depth")
	}
	inReveal(g, zn)
}

// inEncounters: with insight the deep draws horrors. Placed with a side rng per zone, once.
func inEncounters(g *Game, zn *Zone) {
	st := inSt(g)
	tier := inTier(st.Ins)
	if tier < 2 || st.Enc[zn.Key] {
		return
	}
	st.Enc[zn.Key] = true
	r := NewRng(g.Seed + ":in:enc:" + zn.Key)
	for i := 0; i < tier-1; i++ {
		id := "wisp"
		if tier >= 3 && i == 0 {
			id = "horror"
		}
		for tries := 0; tries < 60; tries++ {
			x, y := r.Int(1, MW-2), r.Int(1, MH-2)
			if !zn.At(x, y).D().Walk || zn.At(x, y).D().Water > 0 || zn.ActorAt(x, y) != nil || abs(x-g.P.X)+abs(y-g.P.Y) < 12 {
				continue
			}
			a := g.Spawn(id, maxi(zn.Level, 4), r)
			a.X, a.Y = x, y
			a.Set("in_eldr")
			a.Set("in_veil")
			zn.Actors = append(zn.Actors, a)
			break
		}
	}
}

func inTurn(g *Game) {
	if g.Cur == nil || g.P == nil || g.Dead {
		return
	}
	st := inSt(g)
	for _, a := range g.Cur.Actors {
		if a.HP <= 0 || st.Beheld[a.ID] || !g.visible(a.X, a.Y) || !inSeesHidden(a) {
			continue
		}
		gain := 0
		if a.Has("legend") {
			gain += 2
		}
		if inEldritch(a) && (a.Boss || a.Has("legend")) {
			gain += 2
		}
		if a.Boss && gain > 0 {
			gain++
		}
		if a.Has("in_veil") {
			gain = maxi(gain, 1)
		}
		if gain == 0 {
			continue
		}
		st.Beheld[a.ID] = true
		inGain(g, gain, ": you behold "+g.the(a)+".")
		inComposureHit(g, gain)
	}
	if g.Turn%20 == 0 {
		for k := range g.Read {
			if strings.HasPrefix(k, "book:") && inForbidden(k[5:]) && !st.Books[k] {
				st.Books[k] = true
				inGain(g, 2, ": the pages should not have been read.")
				inComposureHit(g, 1)
			}
		}
		if g.Kills >= 50 {
			inFeat(g, "kills50", "fifty foes fallen")
		}
		if g.Kills >= 200 {
			inFeat(g, "kills200", "two hundred foes fallen")
		}
		if len(lgSt(g).Slain) >= 3 {
			inFeat(g, "slay3", "three legends slain")
		}
		if st.Peak >= inTiers[2] {
			inFeat(g, "ins30", "stared into the deep and kept your name")
		}
	}
	// frenzy: high insight near horrors
	p := g.P
	if inTier(st.Ins) >= 3 {
		for _, a := range g.Cur.Actors {
			if a.HP > 0 && inEldritch(a) && g.Hostile(a) && g.visible(a.X, a.Y) && abs(a.X-p.X)+abs(a.Y-p.Y) < 9 {
				if p.St("frenzied") == 0 && g.R.Chance(0.04+float64(st.Ins-inTiers[2])/400) {
					p.AddSt("frenzied", 10)
					g.Msg("{{R|The sight of it unmakes your calm. You are frenzied!}}")
					inComposureHit(g, 3)
				}
				break
			}
		}
	}
	if p.St("frenzied") > 0 && p.HP > 3 {
		g.hurt(p, 1, nil, "frenzy")
	}
}

func inKill(g *Game, d, src *Actor) {
	if d == nil || d.Has("in_gone") {
		return
	}
	mine := src == g.P || src != nil && src.AI == "follower"
	if !mine {
		return
	}
	if d.Has("legend") {
		inHeartPiece(g, ": a legend's last gift.")
	}
	if inEldritch(d) && (d.Boss || d.Has("legend")) {
		inGain(g, 2, ": its death leaves a lesson.")
	}
}

func inSide(g *Game) []string {
	st := inSt(g)
	if st.Ins == 0 && st.Pieces == 0 {
		return nil
	}
	l := fmt.Sprintf("{{M|Insight %d}}", st.Ins)
	if st.Pieces > 0 || st.Applied > 0 {
		l += fmt.Sprintf("  {{R|♥%d}}{{K|+%d/4}}", st.Pieces/4, st.Pieces%4)
	}
	if g.P.St("frenzied") > 0 {
		l += "  {{R|frenzied}}"
	}
	return []string{l}
}

var inProphetLines = [5][]string{
	nil,
	{"\"The walls write when no one reads them. Have you read them?\"", "\"There is a second staircase under the stair. It counts you going down.\""},
	{"\"It stood beside you in the passage. You walked through it. It forgave you.\"", "\"Eyes in the dark are not eyes. They are the dark, being polite.\""},
	{"\"Cut the seam and the room behind the room breathes out. Do not hold your breath. It notices.\"", "\"I have seen the thing beneath the tower. It was a mercy that it was asleep. It is not asleep.\""},
	{"\"You are almost the right shape. Almost. Come closer to the dark and let it finish.\"", "\"Everything you have learned, the world has learned of you. We are even.\""},
}

func inProphet(a *Actor) bool {
	if a == nil || a.AI == "hostile" || a.AI == "follower" {
		return false
	}
	if a.Role != "priest" && a.Role != "elder" && a.Role != "hermit" {
		return false
	}
	return HashStr(fmt.Sprint(a.ID))%2 == 0
}

func inTalk(g *Game, a *Actor) []Option {
	st := inSt(g)
	tier := inTier(st.Ins)
	var out []Option
	if tier >= 1 && inProphet(a) {
		out = append(out, Option{"{{M|Listen to the ravings.}}", func() {
			lines := inProphetLines[tier]
			r := NewRng(fmt.Sprint(g.Seed, ":in:prophet:", a.ID, ":", tier))
			text := Pick(r, lines)
			k := fmt.Sprint(a.ID, ":", tier)
			if !st.Prophet[k] {
				st.Prophet[k] = true
				g.Popup = Wrap(text+"\n\n{{M|It makes a terrible kind of sense. +1 insight}}", 100)
				inGain(g, 1, "")
			} else {
				g.Popup = Wrap(text, 100)
			}
			g.Opts = []Option{{"Back.", func() { g.openTalk(a) }}}
			g.Sel = 0
		}})
	}
	if tier >= 2 && a.Role == "merchant" && a.AI != "hostile" && a.AI != "follower" && HashStr(fmt.Sprint(a.ID, "in"))%6 == 0 {
		out = append(out, Option{"{{M|Ask about the thing under the black cloth.}}", func() {
			inShrineMenu(g, a, "\"Not for coin. For what you have seen.\"", func() { g.openTalk(a) })
		}})
	}
	return out
}

// inShrineMenu is the insight shop, used by eldritch shrines and odd merchants.
func inShrineMenu(g *Game, who *Actor, text string, back func()) {
	st := inSt(g)
	g.Talk = who
	g.Mode = "talk"
	g.Popup = Wrap(text+fmt.Sprintf("\n\n{{M|Insight: %d}}   {{R|Hearts: %d (+%d/4)}}", st.Ins, st.Pieces/4, st.Pieces%4), 100)
	g.Opts = nil
	g.Sel = 0
	add := func(t string, f func()) { g.Opts = append(g.Opts, Option{t, f}) }
	cost := func(label string, c int, f func()) {
		if st.Ins < c {
			add("{{K|"+label+fmt.Sprintf(" (needs %d insight)}}", c), func() { inShrineMenu(g, who, "You have not seen enough.", back) })
			return
		}
		warn := ""
		if inTier(st.Ins-c) < inTier(st.Ins) {
			warn = " {{R|(the world will dim for you)}}"
		}
		add(fmt.Sprintf("%s {{M|(%d insight)}}%s", label, c, warn), func() {
			inGain(g, -c, "")
			f()
		})
	}
	cost("Trade a vision for a relic.", inCostRelic, func() {
		st.Buys++
		r := NewRng(fmt.Sprint(g.Seed, ":in:buy:", st.Buys))
		var it *Item
		for i := 0; i < 24; i++ {
			it = g.bossDrop(r, maxi(g.Cur.Level, 8))
			if it.Cat == "eldritch" {
				break
			}
		}
		g.give(it)
		g.Msg("It is pressed into your hands: " + it.Display() + ".")
		inShrineMenu(g, who, "Taken. Do not ask what it was before.", back)
	})
	cost("Trade a vision for a heart piece.", inCostHeart, func() {
		inHeartPiece(g, "")
		inShrineMenu(g, who, "A small warm thing, freely given. Suspicious.", back)
	})
	cost("Have the voices quieted.", inCostCalm, func() {
		delete(g.P.Status, "frenzied")
		inComposureHit(g, -3)
		inShrineMenu(g, who, "Hush. Better?", back)
	})
	add("Leave.", func() {
		if back != nil {
			back()
			return
		}
		g.Mode, g.Talk = "main", nil
	})
}

func inKeyU(g *Game) {
	for dy := -1; dy <= 1; dy++ {
		for dx := -1; dx <= 1; dx++ {
			if (dx != 0 || dy != 0) && inBump(g, g.P.X+dx, g.P.Y+dy) {
				return
			}
		}
	}
	st := inSt(g)
	tier := inTier(st.Ins)
	next := "{{K|nothing further}}"
	if tier < 4 {
		next = fmt.Sprintf("%d insight", inTiers[tier])
	}
	lines := []string{
		fmt.Sprintf("{{M|Insight %d}}  next change at %s.", st.Ins, next),
		fmt.Sprintf("{{R|Hearts %d}}, %d/4 toward the next. Heart seals open for %d or more.", st.Pieces/4, st.Pieces%4, 1),
		"",
		"{{K|5: glyphs show on walls.   15: veiled horrors visible.}}",
		"{{K|30: secret doors, frenzy near horrors.   50: all of it.}}",
		"",
		"Face a seal or shrine and press U, or bump it.",
	}
	g.showPopup("Insight", strings.Join(lines, "\n"))
}
