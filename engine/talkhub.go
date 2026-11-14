package engine

import (
	"fmt"
	"regexp"
	"strings"
)

// Talk menu hub: when talkMenu would list more than th2Flat options, the top level keeps the
// core conversation, the person's central services and quest turn-ins, and opens the rest as
// grouped submenus ("Ask about...", "Social...", ...). Options are classified by their text, so
// the many systems that add options through talkHooks need no changes.

const th2Flat = 10 // at most this many options: keep the flat list

// Group names (also the submenu entry labels).
const (
	th2Ask    = "ask"
	th2Social = "social"
	th2Serv   = "services"
	th2Work   = "work"
	th2Shady  = "shady"
	th2More   = "more"
)

var th2Titles = []struct{ id, label, color string }{
	{th2Ask, "Ask about…", "c"},
	{th2Social, "Social…", "M"},
	{th2Serv, "Services…", "W"},
	{th2Work, "Work and quests…", "G"},
	{th2Shady, "Shady…", "R"},
	{th2More, "More…", "K"},
}

type th2Rule struct {
	group string
	pre   []string // option text starts with
	has   []string // option text contains
}

// th2Rules run in order, first match wins ("" = stays on the top level). Text is lower-cased,
// markup and a leading [check label] removed.
var th2Rules = []th2Rule{
	{th2Shady, nil, []string{"pick their pocket", "pick the pocket", "lift something", "unattended", "slip something", "slip poison", "spike their", "sleeping draught", "forged", "contraband", "quiet work", "quietly", "fence", "press them", "threaten", "blackmail", "bribe", "skim a", "against the law", "back room", "join the plot", "throw the bout", "surrender", "weighted dice", "spoof", "i need papers", "keep out of your way", "treasure map"}},
	{"", []string{"who are you", "what news", "let's trade", "heal my wounds", "i want passage", "i want a ticket", "join the", "perform the", "share a secret", "swap recipes", "travel with me", "goodbye", "farewell", "i have done what", "i bring a letter", "i found what", "hand in", "hand over", "present the dispatch", "report on", "something is on your mind", "do you have any work", "have a garment", "install or remove"}, nil},
	{th2Work, nil, []string{"work for me", "trouble here", "any work", "caravan", "contract", "petition", "errand", "request", "bounty", "accuse", "enlist", "salvage", "file a claim", "slavers", "settle the claim", "clear a dungeon", "charter", "sign on", "charge", "the court needs", "is something troubling", "break the seal", "ask about the investigation"}},
	{th2Serv, nil, []string{"trade in bulk", "reforge", "install", "remade", "garment", "brew", "hearth and still", "herb", "tanner", "lockpick", "cut a key", "rigger", "ostler", "stable", "saddle", "mount", "shipwright", "ships for sale", "hull", "harbour", "sailors", "diving gear", "ticket", "passage", "poultice", "tonic", "remedy", "treatment", "plague", "decontaminate", "chelation", "rent a room", "sell water", "buy water", "dirty water", "appraise", "mend ", "repair", "refill", "vat sample", "pays less", "see to my beast", "instruments", "see ships", "buy a house", "tent", "fuel it", "patch it", "rates today", "change my", "change half", "stakes: none"}},
	{th2Social, nil, []string{"released from", "to be taught", "vow", "swear", "court", "duel", "challenge", "play ", "game", "wager", "bet on", "stake", "gift", "family", "wedding", "tongue", "few words", "teach me", "teach them", "compare notes", "trade news", "tell them about", "gesture", "hear a song", "make up", "quarrel", "speak for", "speak as kin", "take them along", "come along", "companion", "invite", "part ways", "share a meal", "tip ", "buy a round", "pray", "walk a while", "ponder", "who do i know", "bout"}},
	{th2Ask, []string{"ask", "what", "heard", "tell me", "where is", "why are", "read the", "listen", "remind me", "look at", "is there", "you look", "about your", "recite", "\""}, []string{"omens", "rumour", "lore", "news", "tale"}},
	{th2Serv, []string{"buy ", "sell ", "bring out", "stable "}, nil},
}

var th2LabelRe = regexp.MustCompile(`^\s*\[[^\]]*\]\s*`)

// th2Group classifies an option text: one of the th2* group ids, or "" for the top level.
func th2Group(text string) string {
	t := strings.ToLower(strings.TrimSpace(Plain(text)))
	if r := strings.TrimSpace(th2LabelRe.ReplaceAllString(t, "")); r != "" && !strings.HasPrefix(r, "(") {
		t = r
	}
	for _, ru := range th2Rules {
		for _, p := range ru.pre {
			if strings.HasPrefix(t, p) {
				return ru.group
			}
		}
		for _, h := range ru.has {
			if strings.Contains(t, h) {
				return ru.group
			}
		}
	}
	return th2More
}

// th2Back remembers the open submenu (per game) so Escape can return to the hub.
var th2Back = map[*Game]*th2Sub{}

type th2Sub struct {
	first *Option
	n     int
	back  func()
}

// th2Esc is called by the talk-mode Escape: leaves a submenu back to the hub, true if it did.
func (g *Game) th2Esc() bool {
	s := th2Back[g]
	if s == nil {
		return false
	}
	if len(g.Opts) != s.n || &g.Opts[0] != s.first {
		delete(th2Back, g)
		return false
	}
	s.back()
	return true
}

// th2Apply turns g.Opts (already complete, Goodbye last) into the hub when it is long.
func (g *Game) th2Apply() {
	delete(th2Back, g)
	if len(g.Opts) <= th2Flat {
		return
	}
	var bye *Option
	opts := g.Opts
	if n := len(opts); n > 0 && th2Group(opts[n-1].Text) == "" && strings.Contains(strings.ToLower(Plain(opts[n-1].Text)), "goodbye") {
		bye = &opts[n-1]
		opts = opts[:n-1]
	}
	groups := map[string][]Option{}
	for _, o := range opts {
		groups[th2Group(o.Text)] = append(groups[th2Group(o.Text)], o)
	}
	var hub []Option
	pos := map[string]int{}
	for _, o := range groups[""] {
		hub = append(hub, o)
	}
	for _, t := range th2Titles {
		list := groups[t.id]
		switch len(list) {
		case 0:
			continue
		case 1:
			hub = append(hub, list[0])
			continue
		}
		pos[t.id] = len(hub)
		id, label, color := t.id, t.label, t.color
		hub = append(hub, Option{fmt.Sprintf("{{%s|%s}} {{K|(%d)}}", color, label, len(list)), func() { g.th2Open(hub, id, pos[id], list) }})
	}
	if bye != nil {
		hub = append(hub, *bye)
	}
	g.Opts = hub
}

func (g *Game) th2Open(hub []Option, id string, idx int, list []Option) {
	sub := make([]Option, 0, len(list)+1)
	sub = append(sub, list...)
	back := func() {
		delete(th2Back, g)
		g.Opts = hub
		g.Sel = idx
	}
	sub = append(sub, Option{"{{K|Back.}}", back})
	g.Opts = sub
	g.Sel = 0
	th2Back[g] = &th2Sub{first: &sub[0], n: len(sub), back: back}
}
