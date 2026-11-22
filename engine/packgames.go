package engine

import (
	"fmt"
	"strings"
)

// Culture pack games (prefix pg): Material.Games (name, rule) shown as the local names and rules of the six table
// games of games.go. A pack game takes the table game it most resembles (by words in its name and rule: cards,
// bluff, dice, tiles, board, shell game); the next free table takes the next pack game, and the tables a pack does
// not name keep their old names. The mechanics are the table game's own; the name, the rule text in the stake menu
// and the "Ask about the games" talk option are the pack's. No state, nothing random, nil-safe.
//
//   - pgNames()      the six names for this place (hooks in games.go gmGameName and gmStart)
//   - pgRule(kind)   rule text appended to the stake menu (hook in games.go gmBetMenu)
//   - talk option    "Ask about the games played here." for players of table games, innkeepers and bards

func init() {
	talkHooks = append(talkHooks, pgTalk)
}

type pgEntry struct {
	Name, Rule string
	Kind       int // the table game (0..5) it plays as
}

// pgKinds lists the keywords of each table game, in the order they are tried.
var pgKinds = [gmGameCount][]string{
	0: {"card", "trick", "leaf", "tarot", "suit", "trump"},
	1: {"bluff", "liar", " lie", "cup"},
	2: {"dice", "die ", "bones", "knuckle", "cowrie", "cast", "lot ", "throw", "hazard"},
	3: {"tile", "domino", "mahjong", "tablet", "counter"},
	4: {"board", "siege", "chase", "capture", "stones", "king ", "sow", "mancala", "hunt", "jump", "piece", "fort", "castle"},
	5: {"shell", "pea ", "hidden", "guess", "thimble", "conceal"},
}

func pgGuess(name, rule string) int {
	for _, hay := range []string{" " + strings.ToLower(name) + " ", " " + strings.ToLower(rule) + " "} {
		for k := 0; k < gmGameCount; k++ {
			for _, w := range pgKinds[k] {
				if strings.Contains(hay, w) {
					return k
				}
			}
		}
	}
	return -1
}

// pgGames: the pack games of the place you are in, each with the table game it plays as. Nil without a pack.
func (g *Game) pgGames() []pgEntry {
	_, p := g.cgHere()
	if p == nil || p.Material == nil || len(p.Material.Games) == 0 {
		return nil
	}
	var out []pgEntry
	taken := [gmGameCount]bool{}
	for i, gr := range p.Material.Games {
		if i >= gmGameCount || strings.TrimSpace(gr.Name) == "" {
			continue
		}
		k := pgGuess(gr.Name, gr.Rule)
		if k < 0 {
			k = int(HashStr(g.W.Seed+":pg:"+gr.Name) % gmGameCount)
		}
		for n := 0; n < gmGameCount && taken[k]; n++ {
			k = (k + 1) % gmGameCount
		}
		if taken[k] {
			continue
		}
		taken[k] = true
		out = append(out, pgEntry{Name: strings.TrimSpace(gr.Name), Rule: strings.TrimSpace(gr.Rule), Kind: k})
	}
	return out
}

// pgNames: gmNames with the pack's local names laid over it.
func (g *Game) pgNames() [gmGameCount]string {
	names := gmNames(g.Genre, g.gmSphere())
	for _, e := range g.pgGames() {
		names[e.Kind] = strings.ToLower(e.Name)
	}
	return names
}

// pgRule: the rule text for a table game of this place, "" when the pack does not name it.
func (g *Game) pgRule(kind int) string {
	for _, e := range g.pgGames() {
		if e.Kind == kind && e.Rule != "" {
			return "\n{{K|Local rule: " + e.Rule + "}}"
		}
	}
	return ""
}

var pgRoles = map[string]bool{"innkeeper": true, "tavern": true, "gambler": true, "bard": true, "minstrel": true}

func pgTalk(g *Game, a *Actor) []Option {
	if !cgOK(g, a) || g.Cur == nil || g.Cur.Kind != "town" || (!gmPlayRoles[a.Role] && !pgRoles[a.Role]) {
		return nil
	}
	games := g.pgGames()
	if len(games) == 0 {
		return nil
	}
	return []Option{{"Ask about the games played here.", func() {
		base := gmNames(g.Genre, g.gmSphere())
		lines := []string{"\"Folk here play these,\" " + g.the(a) + " says."}
		for _, e := range games {
			l := fmt.Sprintf("{{Y|%s}}", titleCase(e.Name))
			if e.Rule != "" {
				l += ": " + e.Rule
			}
			if base[e.Kind] != strings.ToLower(e.Name) {
				l += fmt.Sprintf(" {{K|(At the table it plays like %s.)}}", base[e.Kind])
			}
			lines = append(lines, l)
		}
		g.talkMenu(strings.Join(lines, "\n"))
	}}}
}
