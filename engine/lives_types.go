package engine

// Shared model of the NPC life simulation (prefix lv). Three files build on it, concurrently:
//   lives.go     (A) who people are: generation, bios, personality, relationships, memory of you, talk, People screen
//   lifesim.go   (B) what they do: the day-clock agent simulation, events, news, visible changes to towns
//   knowledge.go (C) what they know: facts, topics, questioning, cross-checking, gossip
// This file only holds the data model and the few helpers all three need. Do not rename fields: they are used
// across files. Add fields only at the end of a struct, always omitempty.

import "fmt"

// lvPerson is one person in the simulation: a Realmforge notable ("s<id>") or a generated townsperson ("g<burg>:<n>").
type lvPerson struct {
	Key     string            `json:"k"`
	Name    string            `json:"n"`
	Role    string            `json:"r"`
	Species string            `json:"sp,omitempty"`
	Culture int               `json:"cu,omitempty"`
	State   int               `json:"st,omitempty"`
	Faith   int               `json:"fa,omitempty"`
	Female  bool              `json:"f,omitempty"`
	Born    int               `json:"bo"`           // calendar year (the world's year numbering)
	Alive   bool              `json:"al"`           // false once dead
	Died    int               `json:"di,omitempty"` // year
	Cause   string            `json:"ca,omitempty"` // how they died
	Home    int               `json:"h"`            // home burg id
	Burg    int               `json:"b"`            // burg they are in now; 0 = on the road
	Dest    int               `json:"de,omitempty"` // burg they are travelling to (0 = none)
	ArriveD int               `json:"ar,omitempty"` // day they arrive (lvDay counter)
	Job     string            `json:"j,omitempty"`  // trade or post
	Wealth  int               `json:"w,omitempty"`  // coin, rough
	Mood    int               `json:"mo,omitempty"` // -100 grim .. 100 elated
	Trait   [5]int            `json:"t"`            // openness, conscientiousness, extraversion, agreeableness, neuroticism: 0..100
	Goal    string            `json:"g,omitempty"`  // current aim, one short line
	GoalID  string            `json:"gi,omitempty"` // machine id of the goal kind
	GoalPct int               `json:"gp,omitempty"` // progress 0..100
	Fear    string            `json:"fe,omitempty"`
	Secret  string            `json:"se,omitempty"`  // something they hide; may be revealed
	Spouse  string            `json:"sps,omitempty"` // key
	Kin     []string          `json:"ki,omitempty"`
	Friends []string          `json:"fr,omitempty"`
	Rivals  []string          `json:"rv,omitempty"`
	Boss    string            `json:"bs,omitempty"` // key of employer or patron
	Life    []lvEvent         `json:"li,omitempty"` // bio lines, oldest first
	Mem     []lvMemory        `json:"me,omitempty"` // what they remember about the player
	Opinion int               `json:"op,omitempty"` // of the player, -100 .. 100
	Knows   []int             `json:"kn,omitempty"` // knowledge fact ids (knowledge.go owns the facts)
	Zone    string            `json:"z,omitempty"`  // zone key where they last stood when the player saw them
	Met     bool              `json:"mt,omitempty"` // the player has talked to them
	Rel     map[string]string `json:"rl,omitempty"` // key -> what that person is to this one (mother, son, sister...), set by lives.go
}

// lvEvent is one line of a life: a birth, a wedding, a promotion, a crime, a death.
type lvEvent struct {
	Y    int    `json:"y"`
	Kind string `json:"k"` // birth marriage child job move crime feud friend illness death goal secret
	Text string `json:"t"`
}

// lvMemory is something a person remembers the player doing to them or near them.
type lvMemory struct {
	Day    int    `json:"d"` // lvDay counter
	Kind   string `json:"k"` // gift help insult theft attack kill spared trade quest saved witnessed
	Text   string `json:"t"`
	Weight int    `json:"w"` // how strongly: -20 .. 20, added to Opinion
}

// lvState is everything saved: g.ext("lv"). It is planet-local (see unLocalKeys) because people belong to a planet.
type lvState struct {
	People map[string]*lvPerson `json:"p,omitempty"`
	Day    int                  `json:"d"`           // last simulated day (lifesim)
	Seeded map[int]bool         `json:"s,omitempty"` // burg ids whose people exist
	News   []lvNews             `json:"n,omitempty"` // recent happenings, newest last, capped at 60
	Next   int                  `json:"x,omitempty"` // counter for generated keys
}

// lvNews is one happening worth telling: shown in the news and gossip.
type lvNews struct {
	Day  int    `json:"d"`
	Burg int    `json:"b"`
	Text string `json:"t"`
}

func init() { unLocalKeys = append(unLocalKeys, "lv") }

// lvSt is the saved state, created on first use.
func lvSt(g *Game) *lvState {
	s := g.ext("lv", func() any { return &lvState{People: map[string]*lvPerson{}, Seeded: map[int]bool{}} }).(*lvState)
	if s.People == nil {
		s.People = map[string]*lvPerson{}
	}
	if s.Seeded == nil {
		s.Seeded = map[int]bool{}
	}
	return s
}

// lvDay is the number of whole game days since the start of the run (day 0 = the first morning).
func lvDay(g *Game) int { return DayOf(g.Turn) }

// lvYear is the world's calendar year now: the world year at the start plus whole 112-day years played.
func lvYear(g *Game) int { return g.W.Year + lvDay(g)/112 }

// lvGet returns a person by key, or nil.
func lvGet(g *Game, key string) *lvPerson { return lvSt(g).People[key] }

// addLife appends a bio line.
func (p *lvPerson) addLife(y int, kind, text string) {
	p.Life = append(p.Life, lvEvent{Y: y, Kind: kind, Text: text})
	if len(p.Life) > 40 {
		p.Life = p.Life[len(p.Life)-40:]
	}
}

// remember records something about the player and shifts the opinion; capped history.
func (p *lvPerson) remember(day int, kind, text string, weight int) {
	p.Mem = append(p.Mem, lvMemory{Day: day, Kind: kind, Text: text, Weight: weight})
	if len(p.Mem) > 24 {
		p.Mem = p.Mem[len(p.Mem)-24:]
	}
	p.Opinion = clampi(p.Opinion+weight, -100, 100)
}

// lvAge is the person's age in years at the given world year.
func (p *lvPerson) lvAge(year int) int {
	end := year
	if !p.Alive && p.Died > 0 {
		end = p.Died
	}
	return maxi(0, end-p.Born)
}

// addNews records a happening (capped) and returns the text.
func (s *lvState) addNews(day, burg int, format string, a ...any) string {
	t := fmt.Sprintf(format, a...)
	s.News = append(s.News, lvNews{Day: day, Burg: burg, Text: t})
	if len(s.News) > 60 {
		s.News = s.News[len(s.News)-60:]
	}
	return t
}

// lvPresent lists the living people who are in a burg right now (not away on the road, not dead), in stable order.
// lives.go binds the actors of a town zone to this list when the zone is entered, so what the simulation
// decides (who travelled, who died, who was born) is what you see.
func lvPresent(g *Game, burg int) []*lvPerson {
	var out []*lvPerson
	for _, p := range lvSt(g).People {
		if p.Alive && p.Burg == burg {
			out = append(out, p)
		}
	}
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j].Key < out[j-1].Key; j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
	return out
}

// Function variables each file fills from its own init(), so the three files compile independently.
var (
	// lvEnsureTown (lives.go) makes sure the people of a burg exist: its Realmforge notables and a dozen or so
	// generated townsfolk with bios, families and relationships. Safe to call repeatedly.
	lvEnsureTown = func(g *Game, burg int) {}
	// lvPersonOf (lives.go) finds or lazily creates the person behind a talking actor, or nil.
	lvPersonOf = func(g *Game, a *Actor) *lvPerson { return nil }
	// lvBioText (lives.go) renders a person's page as lines of text with {{X|..}} markup.
	lvBioText = func(g *Game, p *lvPerson) []string { return nil }
	// lvKnowledgeOf (knowledge.go) returns fact ids a person would know at creation (called once by lives.go if set).
	lvKnowledgeOf = func(g *Game, p *lvPerson) []int { return nil }
)
