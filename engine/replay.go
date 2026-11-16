package engine

// Replay: an in-memory key log of the run (rp prefix). The engine is deterministic given world + CharSpec +
// key sequence + host prefs (its only randomness is the splitmix Rng seeded from the world seed and hero name;
// no wall clock, no math/rand; map iterations that matter are sorted), so a run is recorded as the CharSpec plus
// the keys, run-length compressed. Gameplay-affecting host prefs are logged as "@pref:" events between keys.
// Lightweight checkpoints let playback notice a desync instead of silently going wrong.
//
// The log is NOT part of Save(): it lives in a package-level slot keyed to the live *Game, and the shell stores
// ReplayData() separately (IndexedDB) and hands it back with ReplayRestore() after LoadGame.
// A game loaded without a restore starts a fresh log lazily, from a base save taken just before the next key.

import (
	"encoding/json"
	"fmt"
	"strings"
)

const (
	rpVersion   = 1
	rpMaxKeys   = 200000 // expanded keys kept; past this recording stops (Trunc) so the start stays intact
	rpMaxHl     = 400
	rpCpEvery   = 64
	rpPrefTag   = "@pref:"
	rpHlTextMax = 140
)

// rpPrefs are the host prefs that can change what the engine does (not only how it draws); changes are logged.
var rpPrefs = []string{"autoaim", "autodrink", "hints", "tu_on", "anim", "sprites"}

// RpHighlight marks a moment worth jumping to: K is the key index (count of keys fed) at which it had happened.
type RpHighlight struct {
	K    int    `json:"k"`
	T    int    `json:"t"`    // game turn
	Kind string `json:"kind"` // deed level boss death
	Text string `json:"text"`
}

// rpRun is one run-length entry, JSON [key, count, turn]. Pref events use key "@pref:<name>:<0|1>" with count 1.
type rpRun struct {
	K string
	N int
	T int
}

func (r rpRun) MarshalJSON() ([]byte, error) {
	if r.N <= 1 {
		return json.Marshal([]any{r.K, 1, r.T})
	}
	return json.Marshal([]any{r.K, r.N, r.T})
}

func (r *rpRun) UnmarshalJSON(b []byte) error {
	var a []json.RawMessage
	if err := json.Unmarshal(b, &a); err != nil || len(a) < 3 {
		return fmt.Errorf("bad replay entry")
	}
	if err := json.Unmarshal(a[0], &r.K); err != nil {
		return err
	}
	if err := json.Unmarshal(a[1], &r.N); err != nil {
		return err
	}
	return json.Unmarshal(a[2], &r.T)
}

// rpData is the serialised replay (ReplayData / ReplayRestore).
type rpData struct {
	V     int            `json:"v"`
	Seed  string         `json:"seed"`
	Genre string         `json:"genre"`
	Name  string         `json:"name"`
	Spec  *CharSpec      `json:"spec,omitempty"` // run began at NewGame
	Base  string         `json:"base,omitempty"` // or: began from this save (game was loaded without a replay)
	Keys  []rpRun        `json:"keys"`
	N     int            `json:"n"`  // keys fed (expanded)
	Cp    [][6]int       `json:"cp"` // n, turn, x, y, hp, zone hash
	End   *[6]int        `json:"end,omitempty"`
	Hl    []RpHighlight  `json:"hl"`
	Trunc bool           `json:"trunc,omitempty"`
	Pf    map[string]int `json:"pf,omitempty"` // last logged prefs
	Dead  bool           `json:"dead,omitempty"`
	Level int            `json:"level,omitempty"`
	HlSeq int            `json:"hlseq,omitempty"`
}

type rpLog struct {
	d        rpData
	deedSeen int
	level    int
	dead     bool
}

var (
	rpSlotG *Game
	rpSlotL *rpLog
	rpDepth int
	rpOff   = map[*Game]bool{} // playback games never record
)

func rpGet(g *Game) *rpLog {
	if rpSlotG == g {
		return rpSlotL
	}
	return nil
}

func rpDrop(g *Game) {
	if rpSlotG == g {
		rpSlotG, rpSlotL = nil, nil
	}
}

func (g *Game) rpNewLog() *rpLog {
	l := &rpLog{d: rpData{V: rpVersion, Seed: g.unHome(), Genre: g.Genre, Name: g.P.Name, Pf: map[string]int{}}}
	l.deedSeen, l.level, l.dead = len(g.Deeds), g.P.Level, g.Dead
	rpSlotG, rpSlotL = g, l
	return l
}

// rpBegin is called at the end of NewGame: the run's start state is fully defined by world + spec.
func (g *Game) rpBegin(spec CharSpec) {
	if !g.Pref("recordReplay") {
		return
	}
	l := g.rpNewLog()
	s := spec
	l.d.Spec = &s
	l.deedSeen = 0 // deeds made by NewGame itself count as the opening
	l.level = 1
}

// rpEnter wraps Key(): `defer g.rpEnter(k)()`. Only the outermost Key call is recorded (the command palette
// calls Key recursively).
func (g *Game) rpEnter(k string) func() {
	rpDepth++
	depth := rpDepth
	if depth == 1 && !rpOff[g] {
		if !g.Pref("recordReplay") {
			rpDrop(g)
		} else if rpGet(g) == nil {
			l := g.rpNewLog() // loaded without a replay: begin from a base save
			l.d.Base = g.Save()
		}
	}
	turn := g.Turn
	return func() {
		rpDepth--
		if depth == 1 {
			if l := rpGet(g); l != nil {
				g.rpRecord(l, k, turn)
			}
		}
	}
}

func rpProbe(g *Game, n int) [6]int {
	x, y := 0, 0
	hp := 0
	if g.P != nil {
		x, y, hp = g.P.X, g.P.Y, g.P.HP
	}
	return [6]int{n, g.Turn, x, y, hp, int(HashStr(g.CurKey) & 0xffffffff)}
}

func (g *Game) rpRecord(l *rpLog, k string, turn int) {
	d := &l.d
	if d.Trunc {
		return
	}
	if d.N >= rpMaxKeys {
		d.Trunc = true
		return
	}
	// prefs that change play: log changes before the key they precede
	for _, name := range rpPrefs {
		v := 0
		if g.Pref(name) {
			v = 1
		}
		old, ok := d.Pf[name]
		if !ok {
			old = 1 // a fresh game has every pref on
		}
		if old != v {
			d.Keys = append(d.Keys, rpRun{K: fmt.Sprintf("%s%s:%d", rpPrefTag, name, v), N: 1, T: turn})
		}
		d.Pf[name] = v
	}
	if n := len(d.Keys); n > 0 && d.Keys[n-1].K == k && !strings.HasPrefix(k, rpPrefTag) {
		d.Keys[n-1].N++
	} else {
		d.Keys = append(d.Keys, rpRun{K: k, N: 1, T: turn})
	}
	d.N++
	if d.N%rpCpEvery == 0 {
		d.Cp = append(d.Cp, rpProbe(g, d.N))
	}
	g.rpHighlights(l)
}

func rpClip(s string) string {
	r := []rune(strings.TrimSpace(s))
	if len(r) > rpHlTextMax {
		return string(r[:rpHlTextMax-1]) + "…"
	}
	return string(r)
}

func (l *rpLog) addHl(h RpHighlight) {
	d := &l.d
	d.HlSeq++
	d.Hl = append(d.Hl, h)
	if len(d.Hl) <= rpMaxHl {
		return
	}
	prio := func(k string) int {
		switch k {
		case "death", "boss":
			return 2
		case "level":
			return 1
		}
		return 0
	}
	// drop the oldest of the lowest priority; never the first
	for p := 0; p <= 2; p++ {
		for i := 1; i < len(d.Hl); i++ {
			if prio(d.Hl[i].Kind) == p {
				d.Hl = append(d.Hl[:i], d.Hl[i+1:]...)
				return
			}
		}
	}
}

func (g *Game) rpHighlights(l *rpLog) {
	d := &l.d
	if len(g.Deeds) < l.deedSeen {
		l.deedSeen = len(g.Deeds)
	}
	for ; l.deedSeen < len(g.Deeds); l.deedSeen++ {
		dd := g.Deeds[l.deedSeen]
		kind := "deed"
		if dd.Kind == "slay" && strings.Contains(strings.ToLower(dd.Text), "legendary") {
			kind = "boss"
		}
		l.addHl(RpHighlight{K: d.N, T: g.Turn, Kind: kind, Text: rpClip(dd.Text)})
	}
	if g.P != nil {
		if g.P.Level > l.level {
			l.addHl(RpHighlight{K: d.N, T: g.Turn, Kind: "level", Text: fmt.Sprintf("Reached level %d", g.P.Level)})
		}
		l.level = g.P.Level
	}
	if g.Dead && !l.dead {
		t := g.DeathText
		if t == "" {
			t = g.P.Name + " died."
		}
		l.addHl(RpHighlight{K: d.N, T: g.Turn, Kind: "death", Text: rpClip(t)})
		d.Cp = append(d.Cp, rpProbe(g, d.N))
	}
	l.dead = g.Dead
}

// ReplayHL counts highlights added so far (monotonic); the shell watches it in each frame to cut a frame clip.
func (g *Game) ReplayHL() int {
	if l := rpGet(g); l != nil {
		return l.d.HlSeq
	}
	return 0
}

// ReplayData returns the replay as JSON, or "" when none is being recorded.
func (g *Game) ReplayData() string {
	l := rpGet(g)
	if l == nil {
		return ""
	}
	d := l.d
	d.Dead = g.Dead
	d.Level = g.P.Level
	if !d.Trunc {
		e := rpProbe(g, d.N)
		d.End = &e
	}
	b, err := json.Marshal(d)
	if err != nil {
		return ""
	}
	return string(b)
}

// ReplayRestore adopts a stored replay for a game just made by LoadGame, so recording continues the same log.
// It refuses (and the game then starts a fresh base-save log) when the replay does not end where this game is.
func (g *Game) ReplayRestore(s string) error {
	if !g.Pref("recordReplay") {
		return nil
	}
	var d rpData
	if err := json.Unmarshal([]byte(s), &d); err != nil {
		return err
	}
	if d.V != rpVersion {
		return fmt.Errorf("replay version %d", d.V)
	}
	if d.Seed != g.unHome() {
		return fmt.Errorf("replay belongs to another world")
	}
	if d.Trunc {
		return fmt.Errorf("replay was capped")
	}
	if d.End == nil || *d.End != rpProbe(g, d.N) {
		return fmt.Errorf("replay does not end where the save is")
	}
	if d.Pf == nil {
		d.Pf = map[string]int{}
	}
	d.End = nil
	l := &rpLog{d: d, deedSeen: len(g.Deeds), level: g.P.Level, dead: g.Dead}
	rpSlotG, rpSlotL = g, l
	return nil
}

// ---- playback ----

// RpPlayer feeds a stored replay to a fresh game of its own (its own parsed world, so nothing is shared with the
// live run) and checks the checkpoints on the way.
type RpPlayer struct {
	text string
	d    rpData
	g    *Game
	w    *WorldData
	cps  map[int][6]int
	ei   int // entry index
	sub  int // keys already fed from the current run
	n    int
	Div  string // "replay diverged at turn N" once a checkpoint fails
	last Out
}

// NewReplayPlayer parses worldText itself so each (re)start is a pristine world.
func NewReplayPlayer(worldText, data string) (*RpPlayer, error) {
	p := &RpPlayer{text: worldText}
	if err := json.Unmarshal([]byte(data), &p.d); err != nil {
		return nil, err
	}
	if p.d.V != rpVersion {
		return nil, fmt.Errorf("unsupported replay version %d", p.d.V)
	}
	if p.d.Spec == nil && p.d.Base == "" {
		return nil, fmt.Errorf("replay has no start")
	}
	p.cps = map[int][6]int{}
	for _, c := range p.d.Cp {
		p.cps[c[0]] = c
	}
	if p.d.End != nil {
		p.cps[p.d.End[0]] = *p.d.End
	}
	return p, p.Restart()
}

// Restart rebuilds the game at key 0.
func (p *RpPlayer) Restart() error {
	if p.g != nil {
		delete(rpOff, p.g)
	}
	w, err := ParseWorld(p.text)
	if err != nil {
		return err
	}
	p.w = w
	if p.d.Spec != nil {
		p.g = NewGame(w, *p.d.Spec)
		rpDrop(p.g)
	} else {
		g, err := LoadGame(w, p.d.Base)
		if err != nil {
			return err
		}
		p.g = g
	}
	rpOff[p.g] = true
	p.ei, p.sub, p.n, p.Div = 0, 0, 0, ""
	p.last = Out{}
	return nil
}

// Close releases the player's game.
func (p *RpPlayer) Close() {
	if p.g != nil {
		delete(rpOff, p.g)
	}
}

func (p *RpPlayer) Game() *Game { return p.g }
func (p *RpPlayer) Pos() int    { return p.n }
func (p *RpPlayer) Total() int  { return p.d.N }

// Finished is true at the end of the log or after a divergence.
func (p *RpPlayer) Finished() bool { return p.Div != "" || p.n >= p.d.N }

// Step feeds one key (applying any pref events before it). It reports false when nothing more can be fed.
func (p *RpPlayer) Step() bool {
	if p.Finished() {
		return false
	}
	for p.ei < len(p.d.Keys) {
		e := p.d.Keys[p.ei]
		if strings.HasPrefix(e.K, rpPrefTag) {
			parts := strings.Split(e.K[len(rpPrefTag):], ":")
			if len(parts) == 2 {
				p.g.SetPref(parts[0], parts[1] == "1")
			}
			p.ei++
			continue
		}
		if e.N < 1 {
			p.ei++
			continue
		}
		break
	}
	if p.ei >= len(p.d.Keys) {
		p.n = p.d.N
		return false
	}
	e := p.d.Keys[p.ei]
	p.last = p.g.Key(e.K)
	p.n++
	p.sub++
	if p.sub >= e.N {
		p.ei++
		p.sub = 0
	}
	if want, ok := p.cps[p.n]; ok {
		if got := rpProbe(p.g, p.n); got != want {
			p.Div = fmt.Sprintf("replay diverged at turn %d (key %d of %d)", p.g.Turn, p.n, p.d.N)
		}
	}
	return !p.Finished()
}

// StepTo feeds keys until n keys have been fed; going backwards restarts and fast-forwards.
func (p *RpPlayer) StepTo(n int) error {
	if n > p.d.N {
		n = p.d.N
	}
	if n < p.n {
		if err := p.Restart(); err != nil {
			return err
		}
	}
	for p.n < n && p.Step() {
	}
	return nil
}

// NextHighlight returns the index into Highlights of the first highlight after the current position, or -1.
func (p *RpPlayer) NextHighlight() int {
	for i, h := range p.d.Hl {
		if h.K > p.n {
			return i
		}
	}
	return -1
}

// PrevHighlight returns the last highlight before the current position, or -1.
func (p *RpPlayer) PrevHighlight() int {
	r := -1
	for i, h := range p.d.Hl {
		if h.K < p.n {
			r = i
		}
	}
	return r
}

// Info is the player's state as JSON for the shell.
func (p *RpPlayer) Info() string {
	hl := p.d.Hl
	if hl == nil {
		hl = []RpHighlight{}
	}
	b, _ := json.Marshal(map[string]any{"n": p.n, "total": p.d.N, "turn": p.g.Turn, "diverged": p.Div, "done": p.Finished(),
		"hl": hl, "name": p.d.Name, "genre": p.d.Genre, "trunc": p.d.Trunc, "dead": p.d.Dead, "level": p.d.Level,
		"exit": p.last.Exit, "mode": p.g.Mode})
	return string(b)
}

// ReplayLastHl returns the newest highlight as JSON ("" when none).
func (g *Game) ReplayLastHl() string {
	l := rpGet(g)
	if l == nil || len(l.d.Hl) == 0 {
		return ""
	}
	b, _ := json.Marshal(l.d.Hl[len(l.d.Hl)-1])
	return string(b)
}

// JumpHighlight fast-forwards to the next (dir > 0) or previous highlight; false when there is none.
func (p *RpPlayer) JumpHighlight(dir int) bool {
	i := p.NextHighlight()
	if dir < 0 {
		i = p.PrevHighlight()
	}
	if i < 0 {
		return false
	}
	return p.StepTo(p.d.Hl[i].K) == nil
}
