package engine

import (
	"strings"
)

// Story-beat cues for the web shell's music stingers (prefix qcu): CueBegin snapshots a few facts before a key
// press, CueEnd diffs them afterwards (level, quests done, new deeds, mode changes, planet, zone kind, death,
// fresh message lines) and returns at most 4 cues {kind, intensity 1..3, genre, sphere}. CueScene reports the
// combat facts the layered combat music needs (hostile count, boss and a stable boss id). Read-only and
// cosmetic: no g.R, no game state changes; the snapshot lives in g.ext and holds no exported fields.

// Cue is one musical beat the shell should punctuate.
type Cue struct {
	Kind      string `json:"kind"`      // levelup quest discovery relic legend wedding funeral ceremony crossroads prophecy omen deadline rift city dungeon tower death ending
	Intensity int    `json:"intensity"` // 1 soft .. 3 grand
	Genre     string `json:"genre"`
	Sphere    string `json:"sphere"` // culture sphere of the land, may be empty
}

type qcuSnap struct {
	lv, quests, deeds, planet int
	mode, zoneKind, zoneKey   string
	dead                      bool
	msgs                      int
}

type qcuState struct{ s qcuSnap }

func (g *Game) qcuTake() qcuSnap {
	s := qcuSnap{deeds: len(g.Deeds), mode: g.Mode, dead: g.Dead, msgs: len(g.Msgs)}
	if g.W != nil {
		s.planet = g.W.uniIdx
	}
	if g.P != nil {
		s.lv = g.P.Level
	}
	if g.Cur != nil && !g.OnWorld {
		s.zoneKind, s.zoneKey = g.Cur.Kind, g.Cur.Key
	}
	for _, q := range g.Quests {
		if q != nil && q.Done {
			s.quests++
		}
	}
	return s
}

// CueBegin records the state just before a key is handled.
func (g *Game) CueBegin() {
	st := g.ext("qcu", func() any { return &qcuState{} }).(*qcuState)
	st.s = g.qcuTake()
}

func (g *Game) qcuSphere() string {
	if g.W == nil || g.OnWorld {
		return ""
	}
	if cu := g.W.cultByID[g.W.Culture[g.W.Idx(g.WX, g.WY)]]; cu != nil {
		return cu.Sphere
	}
	return ""
}

// CueEnd returns the cues caused since CueBegin, as []Cue (use CueJS for the wasm frame).
func (g *Game) CueEnd() []Cue {
	st := g.ext("qcu", func() any { return &qcuState{} }).(*qcuState)
	old, now := st.s, g.qcuTake()
	if g.P == nil || g.W == nil || g.Cur == nil {
		return nil
	}
	var out []Cue
	sphere := g.qcuSphere()
	add := func(kind string, n int) {
		for i := range out {
			if out[i].Kind == kind {
				return
			}
		}
		if len(out) < 4 {
			out = append(out, Cue{Kind: kind, Intensity: n, Genre: g.Genre, Sphere: sphere})
		}
	}
	if now.dead && !old.dead {
		add("death", 3)
	}
	if now.planet != old.planet {
		add("rift", 3)
		return out // a new world: the rest is arrival noise
	}
	if now.lv > old.lv {
		add("levelup", 2)
	}
	if now.quests > old.quests {
		add("quest", 2)
	}
	if now.deeds > old.deeds && now.deeds-old.deeds <= len(g.Deeds) {
		for _, d := range g.Deeds[old.deeds:] {
			switch d.Kind {
			case "relic":
				add("relic", 2)
			case "slay":
				if d.Monster != 0 {
					add("legend", 3)
				}
			case "discovery":
				add("discovery", 2)
			case "deadline":
				add("deadline", 2)
			case "win", "endgame":
				add("ending", 3)
			}
		}
	}
	if now.mode != old.mode {
		switch now.mode {
		case "crd":
			add("crossroads", 2)
		case "vs":
			add("prophecy", 2)
		case "egend":
			add("ending", 3)
		case "fwc":
			kind := "ceremony"
			if s, ok := g.ext("fw", func() any { return &fwState{} }).(*fwState); ok && s != nil {
				for _, c := range s.Cers {
					if c != nil && c.ID == s.Cur && (c.Kind == "wedding" || c.Kind == "funeral") {
						kind = c.Kind
					}
				}
			}
			add(kind, 2)
		}
	}
	if now.zoneKey != "" && now.zoneKey != old.zoneKey && old.zoneKey != "" && now.zoneKind != old.zoneKind {
		switch now.zoneKind {
		case "town":
			add("city", 1)
		case "dungeon", "crypt":
			add("dungeon", 1)
		case "tower":
			add("tower", 1)
		}
	}
	if k := now.msgs - old.msgs; k > 0 && k <= len(g.Msgs) {
		for _, m := range g.Msgs[len(g.Msgs)-k:] {
			low := strings.ToLower(m.T)
			switch {
			case strings.Contains(low, "omen") || strings.Contains(low, "portent"):
				add("omen", 2)
			case strings.Contains(low, "you discover") || strings.Contains(low, "you have discovered"):
				add("discovery", 1)
			case strings.Contains(low, "deadline") && strings.Contains(low, "days"):
				add("deadline", 1)
			}
		}
	}
	return out
}

// CueJS converts cues to plain maps for the wasm frame ("cues" field).
func CueJS(cs []Cue) []any {
	out := make([]any, len(cs))
	for i, c := range cs {
		out[i] = map[string]any{"kind": c.Kind, "intensity": c.Intensity, "genre": c.Genre, "sphere": c.Sphere}
	}
	return out
}

// CueScene reports the combat facts for the layered combat music ("cuex" field): visible hostile count, a boss
// in sight, a stable boss id (hash of its template, 0 when none), the player's HP fraction and a hash of the zone key
// (so the shell can tell a fled fight from the player leaving).
func (g *Game) CueScene() map[string]any {
	m := map[string]any{"hostiles": 0, "boss": false, "bossId": 0, "hpFrac": 1.0, "inCombat": false, "zone": 0}
	if g.Cur == nil || g.P == nil || g.OnWorld {
		return m
	}
	m["zone"] = int(HashStr(g.Cur.Key) & 0x7fffffff)
	m["hpFrac"] = float64(g.P.HP) / float64(maxi(g.P.MaxHP, 1))
	n, boss, bid := 0, false, 0
	for _, a := range g.Cur.Actors {
		if a != nil && a != g.P && a.HP > 0 && g.visible(a.X, a.Y) && g.Hostile(a) {
			n++
			if a.Boss && !boss {
				boss, bid = true, int(HashStr("boss:"+a.Tpl)&0x7fffffff)
			}
		}
	}
	m["hostiles"], m["boss"], m["bossId"], m["inCombat"] = n, boss, bid, n > 0
	return m
}
