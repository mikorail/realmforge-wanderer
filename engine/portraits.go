package engine

import (
	"fmt"
	"strings"
)

// Portraits (prefix pt): a rectangle of grid cells reserved (blanked) on the talk, trade and
// character screens where the web shell draws a large sprite: the speaker in dialogue and trade, the
// player with their gear as a paper doll on the first character tab. The engine only reserves the
// cells and says who goes in them (`Game.Portrait`, read by cmd/wasm frame()); the shell draws it in
// sprite mode. Text mode never reserves anything: the shell sets pref "sprites" false there.
// Read-only, never draws g.R, nothing is saved.

// PortraitInfo is one reserved rectangle (grid cells) and the actor to paint in it.
type PortraitInfo struct {
	X, Y, W, H int
	Kind       string // talk | trade | sheet
	Actor      SprActor
}

type ptState struct {
	X, Y, W, H int    `json:"-"`
	Kind       string `json:"-"`
	Mode       string `json:"-"`
	Tab        int    `json:"-"`
	Stamp      int    `json:"-"` // g.Tick of the frame that reserved it
}

func (g *Game) pt() *ptState {
	return g.ext("pt", func() any { return &ptState{} }).(*ptState)
}

const (
	ptTalkW = 12
	ptTalkH = 6
)

// ptOn: the shell is in sprite mode (it sets pref "sprites"; unset counts as on).
func (g *Game) ptOn() bool { return g.Pref("sprites") }

// ptGear fills the worn-item fields from a's body (the player: the paper doll).
func ptGear(a *Actor, s *SprActor) {
	s.Gear = true
	for _, p := range a.Body {
		if p.Severed || p.Item == nil {
			continue
		}
		it := p.Item
		switch p.Slot {
		case "head":
			s.Head = it.Name
		case "face":
			if s.Head == "" && it.Kind == "armor" {
				s.Head = it.Name
			}
		case "back":
			s.Back = it.Name
		case "feet":
			s.Feet = it.Name
		case "hands":
			s.Hands = it.Name
		case "body":
			s.Torso = it.Name
		case "hand":
			if it.Kind == "armor" {
				s.Shield = true
			}
		}
	}
}

// ptActor is the speaker's sprite description.
func (g *Game) ptActor(a *Actor) SprActor {
	s := g.sprOf(a)
	s.X, s.Y = 0, 0
	if a == g.P {
		s.Player, s.Calling = true, g.Calling
		ptGear(a, &s)
	}
	return s
}

// ptBandH is the extra height talk needs for the portrait band (0 in text mode).
// Hook: drawTalk, `h := len(g.Popup) + len(g.Opts) + 5 + g.ptBandH()`.
func (g *Game) ptBandH() int {
	if g.Talk == nil || !g.ptOn() {
		return 0
	}
	if len(g.Popup)+len(g.Opts)+5+ptTalkH+1 > SH-3 { // a crowded menu keeps its whole box on screen: no portrait
		return 0
	}
	return ptTalkH + 1
}

// ptTalkBand reserves and blanks the portrait rectangle at the top left of the talk box, writes the
// speaker's details beside it, and returns how many rows the text below must move down.
// Hook: drawTalk, `y := y0 + 1 + g.ptTalkBand(y0)`.
func (g *Game) ptTalkBand(y0 int) int {
	a := g.Talk
	if a == nil || !g.ptOn() || g.ptBandH() == 0 {
		return 0
	}
	s := g.sc
	st := g.pt()
	st.X, st.Y, st.W, st.H, st.Kind, st.Mode, st.Tab, st.Stamp = 4, y0+1, ptTalkW, ptTalkH, "talk", g.Mode, g.Tab, g.Tick
	s.Fill(st.X, st.Y, st.W, st.H, CBgDark)
	x := st.X + st.W + 2
	y := st.Y
	put := func(l string) {
		if l != "" && y < st.Y+st.H {
			s.TextB(x, y, l, CGray, CBgDark, SW-4)
			y++
		}
	}
	put("{{W|" + a.Name + "}}")
	if a.Title != "" {
		put("{{Y|" + a.Title + "}}")
	}
	var who []string
	if a.Species != "" {
		who = append(who, titleCase(a.Species))
	}
	if a.Role != "" {
		who = append(who, titleCase(a.Role))
	}
	put("{{c|" + strings.Join(who, " · ") + "}}")
	if a.Caste != "" {
		put("{{K|" + titleCase(a.Caste) + "}}")
	}
	put(fmt.Sprintf("{{K|level %d · %d/%d HP}}", a.Level, a.HP, a.MaxHP))
	if g.Hostile(a) {
		put("{{R|hostile}}")
	} else if a.Faction != "" {
		put("{{K|" + a.Faction + "}}")
	}
	return ptTalkH + 1
}

// ptTradeRect reserves a small portrait over the top right corner of the trade panel.
// Hook: drawTrade, right after `x, y, w, _ := g.panel(...)`: `g.ptTradeRect()`.
func (g *Game) ptTradeRect() {
	if g.Talk == nil || !g.ptOn() {
		return
	}
	st := g.pt()
	st.X, st.Y, st.W, st.H, st.Kind, st.Mode, st.Tab, st.Stamp = MW-14, 0, 9, 4, "trade", g.Mode, g.Tab, g.Tick
	g.sc.Fill(st.X, st.Y, st.W, st.H, CBgDark)
}

// ptSheet reserves the paper doll rectangle on the first character tab.
// Hook: drawChar, inside `if g.Tab == 0 {` after the first `y += 2`: `g.ptSheet(x+w-14, y, 14, 8)`.
func (g *Game) ptSheet(x, y, w, h int) {
	if g.P == nil || !g.ptOn() {
		return
	}
	st := g.pt()
	st.X, st.Y, st.W, st.H, st.Kind, st.Mode, st.Tab, st.Stamp = x, y, w, h, "sheet", g.Mode, g.Tab, g.Tick
	g.sc.Fill(x, y, w, h, CBgDark)
}

// Portrait is the rectangle reserved by the frame just drawn, or nil (read by cmd/wasm frame()).
func (g *Game) Portrait() *PortraitInfo {
	if g.P == nil || !g.ptOn() {
		return nil
	}
	st := g.pt()
	if st.Kind == "" || st.W == 0 || st.Mode != g.Mode || st.Tab != g.Tab || st.Stamp != g.Tick {
		return nil
	}
	switch st.Kind {
	case "talk", "trade":
		if g.Talk == nil || (g.Mode != "talk" && g.Mode != "trade") {
			return nil
		}
		return &PortraitInfo{X: st.X, Y: st.Y, W: st.W, H: st.H, Kind: st.Kind, Actor: g.ptActor(g.Talk)}
	case "sheet":
		if g.Mode != "char" || g.Tab != 0 {
			return nil
		}
		return &PortraitInfo{X: st.X, Y: st.Y, W: st.W, H: st.H, Kind: st.Kind, Actor: g.ptActor(g.P)}
	}
	return nil
}
