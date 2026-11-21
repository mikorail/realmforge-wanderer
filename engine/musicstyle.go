package engine

import (
	"math"
	"strings"
)

// Culture-specific music data for the shell (prefix mst): the instruments, song forms and scale word of the land's
// culture, today's feast, the nearest visible performer, and the ceremony in progress. Read-only and cosmetic: no
// g.R, no state is changed. The shell's culturemusic.ts turns it into scales, timbres and a positional phrase.

// MstPerformer is a visible NPC who is making music (or a feast crowd that is).
type MstPerformer struct {
	ID         int     `json:"id"`
	Kind       string  `json:"kind"` // bard minstrel innkeeper temple choir busker drummers at a feast street band
	X          int     `json:"x"`
	Y          int     `json:"y"`
	Distance   float64 `json:"distance"` // tiles from the player
	DX         int     `json:"dx"`       // performer minus player, for panning
	DY         int     `json:"dy"`
	Instrument string  `json:"instrument"`
}

// MstStyle is the culture and performer context of the current place; nil on the world map or when dead.
type MstStyle struct {
	Culture     int           `json:"culture"` // -1 when unknown
	Sphere      string        `json:"sphere"`
	Instruments []string      `json:"instruments"`
	Forms       []string      `json:"forms"`
	ScaleWord   string        `json:"scaleWord"`
	TempoHint   string        `json:"tempoHint"` // slow moderate lively
	Feast       string        `json:"feast"`     // name of today's feast in this town, "" if none
	FeastKind   string        `json:"feastKind"`
	Song        string        `json:"song"`  // the pack's song text for today's feast or the ceremony, "" if none
	Rite        string        `json:"rite"`  // ceremony kind while the ceremony screen is up: wedding funeral birth age feast
	Genre       string        `json:"genre"` // lower case world genre
	Performer   *MstPerformer `json:"performer,omitempty"`
}

const mstRange = 20.0

// mstInstWords are the keywords worth keeping from free text, longest first so "throat singing" beats "singing".
var mstInstWords = []string{"throat singing", "didgeridoo", "frame drum", "bagpipe", "balafon", "panpipe", "harpsichord", "hurdy-gurdy", "hurdy gurdy",
	"concertina", "accordion", "lute", "oud", "harp", "lyre", "flute", "pipe", "fiddle", "violin", "drum", "gong", "koto", "sitar", "kora", "choir", "chant", "bell",
	"horn", "organ", "synth", "shawm", "zither", "mandolin", "guitar", "marimba", "kalimba", "tabla", "djembe", "cymbal", "rattle", "whistle", "trumpet", "erhu",
	"gamelan", "bodhran", "tambour", "lamellaphone", "theremin", "banjo", "dulcimer"}

func mstInstruments(cu *CultureData) []string {
	var out []string
	seen := map[string]bool{}
	add := func(s string) {
		s = strings.ToLower(strings.TrimSpace(s))
		if s != "" && !seen[s] && len(out) < 12 {
			seen[s] = true
			out = append(out, s)
		}
	}
	if cu == nil {
		return nil
	}
	if cu.Pack != nil && cu.Pack.Material != nil && cu.Pack.Material.Music != nil {
		for _, s := range cu.Pack.Material.Music.Instruments {
			add(s)
		}
	}
	if len(out) == 0 && cu.Music != "" {
		t := strings.NewReplacer(" and ", ",", ";", ",", "/", ",").Replace(strings.ToLower(cu.Music))
		for _, s := range strings.Split(t, ",") {
			add(s)
		}
	}
	return out
}

func mstForms(cu *CultureData) []string {
	var out []string
	if cu == nil || cu.Pack == nil {
		return nil
	}
	if m := cu.Pack.Material; m != nil && m.Music != nil {
		for _, s := range m.Music.Forms {
			if len(out) < 10 {
				out = append(out, strings.ToLower(s))
			}
		}
	}
	if l := cu.Pack.Literature; l != nil {
		for _, f := range l.Forms {
			if len(out) < 14 {
				out = append(out, strings.ToLower(f.Name))
			}
		}
	}
	return out
}

// mstSong finds the pack's song for a feast (by name) or a rite (by id).
func mstSong(cu *CultureData, feast, rite string) string {
	if cu == nil || cu.Pack == nil || cu.Pack.Customs == nil {
		return ""
	}
	c := cu.Pack.Customs
	if feast != "" {
		for _, f := range c.Festivals {
			if f.Feast == feast && f.Song != "" {
				return f.Song
			}
		}
	}
	if rite != "" {
		id := map[string]string{"age": "comingofage"}[rite]
		if id == "" {
			id = rite
		}
		for _, r := range c.Rites {
			if r.ID == id && r.Song != "" {
				return r.Song
			}
		}
	}
	return ""
}

func mstPickInst(inst []string, id int) string {
	if len(inst) == 0 {
		return ""
	}
	i := id % len(inst)
	if i < 0 {
		i = -i
	}
	return inst[i]
}

// MusicStyle describes the culture's music and the nearest performer; nil on the world map, in a dead state or with
// no zone.
func (g *Game) MusicStyle() *MstStyle {
	if g == nil || g.Cur == nil || g.P == nil || g.W == nil || g.OnWorld || g.Mode == "world" || g.Dead {
		return nil
	}
	z := g.Cur
	cu := g.lxCultureAt()
	s := &MstStyle{Culture: -1, Genre: strings.ToLower(g.Genre), TempoHint: "moderate"}
	if cu != nil {
		s.Culture, s.Sphere = cu.ID, strings.ToLower(cu.Sphere)
		s.Instruments = mstInstruments(cu)
		s.Forms = mstForms(cu)
		if cu.Pack != nil && cu.Pack.Material != nil && cu.Pack.Material.Music != nil {
			s.ScaleWord = strings.ToLower(cu.Pack.Material.Music.Scale)
		}
	}
	if z.Kind == "town" {
		if _, f := g.feastToday(); f != nil {
			s.Feast, s.FeastKind = f.Name, f.Kind
			s.Song = mstSong(cu, f.Name, "")
			s.TempoHint = "lively"
			if f.Kind == "dead" || f.Kind == "midwinter" {
				s.TempoHint = "slow"
			}
		}
	}
	if g.Mode == "fwc" {
		if st := fwSt(g); st != nil {
			if c := fwCerByID(st, st.Cur); c != nil {
				s.Rite = c.Kind
				if sg := mstSong(cu, "", c.Kind); sg != "" {
					s.Song = sg
				}
				switch c.Kind {
				case "funeral":
					s.TempoHint = "slow"
				case "wedding", "age", "birth":
					s.TempoHint = "lively"
				}
			}
		}
	}

	// the nearest visible performer
	best, bestScore := (*Actor)(nil), 1e9
	bestKind := ""
	for _, a := range z.Actors {
		if a == nil || a.HP <= 0 || a.ID == g.P.ID || g.Hostile(a) || a.AI == "follower" {
			continue
		}
		dx, dy := float64(a.X-g.P.X), float64(a.Y-g.P.Y)
		d := math.Hypot(dx, dy)
		if d > mstRange {
			continue
		}
		kind, pri := "", 0.0
		day := g.Day()
		switch {
		case a.Role == "bard":
			kind, pri = "bard", 0
		case a.Role == "minstrel":
			kind, pri = "minstrel", 0
		case s.Feast != "" && a.Role != "guard" && a.Role != "" && HashStr(a.Name+":mstf")%5 == 0:
			kind, pri = "street band", 6
			if s.FeastKind == "war" || s.FeastKind == "harvest" || s.FeastKind == "sun" {
				kind = "drummers at a feast"
			}
		case s.Feast != "" && a.AI == "town" && a.Role == "" && a.ID%6 == 0:
			kind, pri = "street band", 8
		case (a.Role == "innkeeper" || a.Role == "tavern") && HashStr(a.Name+":msti")%3 == 0 && !g.daylight():
			kind, pri = "innkeeper", 4
		case a.Role == "priest" && HashStr(a.Name+":mstp")%2 == 0 && day%2 == 0:
			kind, pri = "temple choir", 5
		}
		if kind == "" || !g.visible(a.X, a.Y) {
			continue
		}
		if sc := d + pri; sc < bestScore {
			best, bestScore, bestKind = a, sc, kind
		}
	}
	if best != nil {
		s.Performer = &MstPerformer{ID: best.ID, Kind: bestKind, X: best.X, Y: best.Y, Instrument: mstPickInst(s.Instruments, best.ID), DX: best.X - g.P.X, DY: best.Y - g.P.Y,
			Distance: math.Round(math.Hypot(float64(best.X-g.P.X), float64(best.Y-g.P.Y))*10) / 10}
	}
	return s
}

// MusicStyleJS converts a MusicStyle to a JS-friendly value (the wasm host's fr["mstyle"]).
func MusicStyleJS(s *MstStyle) any {
	if s == nil {
		return nil
	}
	strs := func(a []string) []any {
		o := make([]any, len(a))
		for i, v := range a {
			o[i] = v
		}
		return o
	}
	m := map[string]any{"culture": s.Culture, "sphere": s.Sphere, "instruments": strs(s.Instruments), "forms": strs(s.Forms),
		"scaleWord": s.ScaleWord, "tempoHint": s.TempoHint, "feast": s.Feast, "feastKind": s.FeastKind, "song": s.Song, "rite": s.Rite, "genre": s.Genre}
	if p := s.Performer; p != nil {
		m["performer"] = map[string]any{"id": p.ID, "kind": p.Kind, "x": p.X, "y": p.Y, "distance": p.Distance, "dx": p.DX, "dy": p.DY, "instrument": p.Instrument}
	}
	return m
}
