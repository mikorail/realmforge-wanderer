package engine

import (
	"strings"
)

// Voice events for the web shell (prefix vx): VoxBegin snapshots the actors of the current zone and the talk
// window before a key press, VoxEnd diffs afterwards and returns what each creature or person would voice:
// pain, death, alert (a hostile newly noticing you), greet (a talk window opened), laugh and chatter (tavern
// barks), battlecry and growl (attackers, boss lines), sigh, and `speak` (a talk line appeared: speaker
// species, register, mood and text length for the speech blips). Read-only and deterministic: no g.R, no
// game state changed, at most 8 events per key. The snapshot lives in g.ext and holds only unexported fields.

// VoxEvent is one voice cue of the last key press.
type VoxEvent struct {
	Kind      string `json:"kind"` // pain death alert greet laugh growl chatter battlecry sigh speak
	Species   string `json:"species,omitempty"`
	Body      string `json:"body,omitempty"` // creature body plan (sprinfo)
	Tpl       string `json:"tpl,omitempty"`
	Size      int    `json:"size,omitempty"` // bestiary size 1..7
	X         int    `json:"x"`
	Y         int    `json:"y"`
	ID        int    `json:"id"`
	Player    bool   `json:"player,omitempty"`
	Role      string `json:"role,omitempty"`
	Caste     string `json:"caste,omitempty"`
	Boss      bool   `json:"boss,omitempty"`
	PitchSeed int    `json:"pitchSeed"`
	// speak only
	Register string `json:"register,omitempty"` // formal blunt flowery terse earthy courtly
	Mood     string `json:"mood,omitempty"`     // portrait mood
	Text     string `json:"text,omitempty"`     // the line (clipped to 200 runes)
	Len      int    `json:"len,omitempty"`
}

const vxMax = 8

type vxActor struct {
	id, x, y, hp int
	ai           string
	hostile      bool
	vis          bool
	name         string
	me           bool
	ghost        Actor // minimal copy, to describe the actor after it has left the zone
}

type vxState struct {
	zone     string
	list     []vxActor
	talkID   int
	talkText uint64
	mode     string
	nmsg     int
	last     string
	lastTurn int
	lastRep  int
	dead     bool
	valid    bool
}

func (g *Game) vxTake() vxState {
	s := vxState{dead: g.Dead, mode: g.Mode}
	if g.Talk != nil && g.Mode == "talk" {
		s.talkID = g.Talk.ID
		s.talkText = vxHashText(strings.Join(g.Popup, "\n"))
	}
	s.nmsg = len(g.Msgs)
	if s.nmsg > 0 {
		m := g.Msgs[s.nmsg-1]
		s.last, s.lastTurn, s.lastRep = m.T, m.Turn, m.Rep
	}
	if g.Cur == nil || g.P == nil || g.OnWorld {
		return s
	}
	s.valid, s.zone = true, g.Cur.Key
	s.list = make([]vxActor, 0, len(g.Cur.Actors)+1)
	add := func(a *Actor) {
		v := vxActor{id: a.ID, x: a.X, y: a.Y, hp: a.HP, ai: a.AI, name: a.Name, me: a == g.P,
			ghost: Actor{ID: a.ID, X: a.X, Y: a.Y, Tpl: a.Tpl, Species: a.Species, Role: a.Role, Caste: a.Caste, Boss: a.Boss, Name: a.Name}}
		if !v.me {
			v.hostile = g.Hostile(a)
			v.vis = g.visible(a.X, a.Y)
		} else {
			v.vis = true
		}
		s.list = append(s.list, v)
	}
	add(g.P)
	for _, a := range g.Cur.Actors {
		if a != nil && a != g.P {
			add(a)
		}
	}
	return s
}

func vxHashText(s string) uint64 {
	h := uint64(14695981039346656037)
	for i := 0; i < len(s); i++ {
		h = (h ^ uint64(s[i])) * 1099511628211
	}
	return h
}

// VoxBegin records the state just before a key is handled.
func (g *Game) VoxBegin() {
	st := g.ext("voxfx", func() any { return &vxState{} }).(*vxState)
	*st = g.vxTake()
}

func (g *Game) vxEvent(kind string, a *Actor) VoxEvent {
	e := VoxEvent{Kind: kind, Species: a.Species, Tpl: a.Tpl, X: a.X, Y: a.Y, ID: a.ID, Player: a == g.P, Role: a.Role, Caste: a.Caste, Boss: a.Boss}
	if a.Species == "" {
		if t := tplByID[a.Tpl]; t != nil {
			e.Body = t.Body
		}
		if b := bsSpecies(a.Tpl); b != nil {
			e.Size = b.Size
			if e.Body == "" {
				e.Body = b.Body
			}
		}
	}
	e.PitchSeed = int((vxHashText(a.Tpl+"|"+a.Species) ^ uint64(a.ID)*2654435761) & 0x7fffffff)
	return e
}

// vxPlain strips {{C|...}} colour markup, keeping case.
func vxPlain(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == '{' && i+1 < len(s) && s[i+1] == '{' {
			j := strings.IndexByte(s[i:], '|')
			if j > 0 && j < 6 {
				i += j
				continue
			}
		}
		if s[i] == '}' && i+1 < len(s) && s[i+1] == '}' {
			i++
			continue
		}
		b.WriteByte(s[i])
	}
	return b.String()
}

// VoxEnd returns the voice cues caused since VoxBegin.
func (g *Game) VoxEnd() []VoxEvent {
	st := g.ext("voxfx", func() any { return &vxState{} }).(*vxState)
	old, now := *st, g.vxTake()
	*st = now
	if g.Cur == nil || g.P == nil {
		return nil
	}
	var out []VoxEvent
	push := func(e VoxEvent) {
		if len(out) < vxMax {
			out = append(out, e)
		}
	}
	actorOf := func(id int) *Actor {
		if id == g.P.ID {
			return g.P
		}
		for _, a := range g.Cur.Actors {
			if a != nil && a.ID == id {
				return a
			}
		}
		return nil
	}

	// the talk window: a greeting when it opens, a speak cue whenever the text changes
	if now.talkID != 0 {
		opened := old.talkID != now.talkID
		if opened || old.talkText != now.talkText {
			if a := g.Talk; a != nil {
				if opened {
					push(g.vxEvent("greet", a))
				}
				e := g.vxEvent("speak", a)
				if sp := g.dvVoice(a); sp != nil {
					e.Register = sp.Register
				}
				e.Mood = g.PortraitMood(a).Mood
				txt := vxPlain(strings.Join(g.Popup, " "))
				r := []rune(txt)
				e.Len = len(r)
				if len(r) > 200 {
					r = r[:200]
				}
				e.Text = string(r)
				push(e)
				if strings.Contains(strings.ToLower(txt), "sigh") {
					push(g.vxEvent("sigh", a))
				}
			}
		}
	}

	if !old.valid || !now.valid || old.zone != now.zone {
		return out
	}

	// fresh log lines
	k := now.nmsg - old.nmsg
	if k < 0 || k > len(g.Msgs) {
		k = 0
	}
	var lines []string
	for _, m := range g.Msgs[len(g.Msgs)-k:] {
		lines = append(lines, vxPlain(m.T))
	}
	if k == 0 && len(g.Msgs) > 0 && old.nmsg > 0 {
		m := g.Msgs[len(g.Msgs)-1]
		if m.Turn != old.lastTurn || m.Rep != old.lastRep || m.T != old.last {
			lines = append(lines, vxPlain(m.T))
		}
	}

	byNow := map[int]*vxActor{}
	for i := range now.list {
		byNow[now.list[i].id] = &now.list[i]
	}
	byName := map[string][]int{} // lower-case name -> actor ids visible now
	for i := range now.list {
		n := &now.list[i]
		if !n.me && n.vis {
			byName[strings.ToLower(n.name)] = append(byName[strings.ToLower(n.name)], n.id)
		}
	}

	// deaths: vanished actors matched to a death line, and the player
	if now.dead && !old.dead {
		e := g.vxEvent("death", g.P)
		push(e)
	}
	deathLine := false
	for _, ln := range lines {
		l := strings.ToLower(ln)
		if strings.HasSuffix(l, " dies.") || strings.HasSuffix(l, " is destroyed.") || strings.HasSuffix(l, " crumbles to dust.") {
			deathLine = true
		}
	}
	if deathLine {
		for _, o := range old.list {
			if o.me || !o.vis || byNow[o.id] != nil {
				continue
			}
			named := false
			for _, ln := range lines {
				l := strings.ToLower(ln)
				if strings.Contains(l, strings.ToLower(o.name)) && (strings.HasSuffix(l, " dies.") || strings.HasSuffix(l, " is destroyed.") || strings.HasSuffix(l, " crumbles to dust.")) {
					named = true
					break
				}
			}
			if !named {
				continue
			}
			gh := o.ghost
			push(g.vxEvent("death", &gh))
		}
	}

	// pain and alert
	for _, o := range old.list {
		n := byNow[o.id]
		if n == nil || !n.vis {
			continue
		}
		a := actorOf(o.id)
		if a == nil {
			continue
		}
		if n.hp < o.hp && n.hp > 0 {
			push(g.vxEvent("pain", a))
		}
		if !o.me && o.vis && !(o.ai == "hostile" || o.hostile) && (n.ai == "hostile" || n.hostile) && g.visible(n.x, n.y) {
			push(g.vxEvent("alert", a))
		}
	}

	// attackers (battlecry for people, growl for beasts), boss lines, laughter and chatter
	done := map[int]bool{}
	for _, ln := range lines {
		l := strings.ToLower(ln)
		switch {
		case strings.Contains(l, " damage") && strings.Contains(l, " for "):
			for nm, ids := range byName {
				if !strings.HasPrefix(l, "the "+nm+" ") && !strings.HasPrefix(l, nm+" ") && !strings.HasPrefix(l, nm) {
					continue
				}
				for _, id := range ids {
					a := actorOf(id)
					if a == nil || done[id] || !g.Hostile(a) {
						continue
					}
					done[id] = true
					if int(vxHashText(l)&3) == 0 { // a cry on about a quarter of the blows
						kind := "growl"
						if a.Species != "" || a.Boss {
							kind = "battlecry"
						}
						push(g.vxEvent(kind, a))
					}
				}
				break
			}
		default:
			// "Name: \"...\"" speech lines (tavern chatter, boss phase lines)
			i := strings.Index(ln, ": \"")
			if i <= 0 || i > 40 {
				continue
			}
			ids := byName[strings.ToLower(ln[:i])]
			if len(ids) == 0 {
				continue
			}
			a := actorOf(ids[0])
			if a == nil || done[a.ID] {
				continue
			}
			done[a.ID] = true
			switch {
			case a.Boss:
				push(g.vxEvent("battlecry", a))
			case strings.Contains(l, "laugh") || strings.Contains(l, "ha ha") || strings.Contains(l, "haha") || strings.Contains(l, "cackl"):
				push(g.vxEvent("laugh", a))
			case strings.Contains(l, "sigh"):
				push(g.vxEvent("sigh", a))
			default:
				push(g.vxEvent("chatter", a))
			}
		}
	}
	return out
}

// VoxJS converts events for the wasm frame (syscall/js cannot take structs).
func VoxJS(ev []VoxEvent) []any {
	out := make([]any, len(ev))
	for i, e := range ev {
		m := map[string]any{"kind": e.Kind, "x": e.X, "y": e.Y, "id": e.ID, "pitchSeed": e.PitchSeed}
		str := func(k, v string) {
			if v != "" {
				m[k] = v
			}
		}
		str("species", e.Species)
		str("body", e.Body)
		str("tpl", e.Tpl)
		str("role", e.Role)
		str("caste", e.Caste)
		str("register", e.Register)
		str("mood", e.Mood)
		str("text", e.Text)
		if e.Size > 0 {
			m["size"] = e.Size
		}
		if e.Len > 0 {
			m["len"] = e.Len
		}
		if e.Player {
			m["player"] = true
		}
		if e.Boss {
			m["boss"] = true
		}
		out[i] = m
	}
	return out
}
