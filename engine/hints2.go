package engine

// More first-time tips (prefix h2, state g.ext("h2")). Events are detected by polling saved state
// and world state once per game minute, so no other file hooks in. Each tip shows once, only while
// the "hints" pref is on, never with a foe at arm's length, at most one per three game minutes;
// extras wait in a queue. Display goes through the same message line as hints.go.

type h2State struct {
	Shown map[string]bool `json:"s,omitempty"`
	Queue []string        `json:"q,omitempty"`
	Next  int             `json:"n,omitempty"`
}

func (g *Game) h2() *h2State {
	s := g.ext("h2", func() any { return &h2State{} }).(*h2State)
	if s.Shown == nil {
		s.Shown = map[string]bool{}
	}
	return s
}

func init() {
	turnHooks = append(turnHooks, h2Turn)
	enterHooks = append(enterHooks, h2Enter)
}

// h2Want queues a tip once.
func (g *Game) h2Want(id string) {
	s := g.h2()
	if s.Shown[id] || len(s.Queue) >= 8 {
		return
	}
	for _, q := range s.Queue {
		if q == id {
			return
		}
	}
	s.Queue = append(s.Queue, id)
}

func (g *Game) h2Melee() bool {
	for _, a := range g.Cur.Actors {
		if a != g.P && a.HP > 0 && a.AI != "follower" && (g.Hostile(a) || a.Angry) && dist(a, g.P) <= 2 {
			return true
		}
	}
	return false
}

func h2Enter(g *Game, zn *Zone) {
	if !g.Pref("hints") || g.P == nil {
		return
	}
	switch zn.Kind {
	case "dungeon", "crypt", "tower", "ruin", "lair", "wonder", "town":
		if len(g.wpl().List) == 0 {
			g.h2Want("place")
		}
	}
}

func h2Turn(g *Game) {
	if !g.Pref("hints") || g.P == nil || g.Cur == nil || g.Dead || g.Mode == "world" || g.OnWorld {
		return
	}
	s := g.h2()
	g.h2Detect(s)
	if len(s.Queue) == 0 || g.Turn < s.Next || g.h2Melee() {
		return
	}
	id := s.Queue[0]
	s.Queue = s.Queue[1:]
	if s.Shown[id] {
		return
	}
	s.Shown[id] = true
	msg := h2Tips[id]
	if msg == "" {
		return
	}
	if len(s.Shown) == 1 && mainKeys[";"] != nil {
		msg += " Press ; for the palette."
	}
	g.Msg("{{c|Tip:}} " + msg)
	s.Next = g.Turn + 3
}

func (g *Game) h2Detect(s *h2State) {
	p, zn := g.P, g.Cur
	want := func(id string, ok bool) {
		if ok && !s.Shown[id] {
			g.h2Want(id)
		}
	}
	// level milestones and plain state
	want("lv2", p.Level >= 2)
	want("lv5", p.Level >= 5)
	want("lv10", p.Level >= 10)
	if g.Hunger >= 1200 && !s.Shown["hungry"] {
		if g.hintState().Shown["hungry"] {
			s.Shown["hungry"] = true
		} else {
			g.h2Want("hungry")
		}
	}
	ij := g.ijSt()
	want("tired", ij.Fat >= ijTiredAt)
	want("injury", len(ij.Inj) > 0)
	want("achieve", len(g.ac().Done) > 0)
	want("tame", g.tm().Tamed > 0)
	want("cr", crSt(g).N > 0)

	// pack contents
	missile := p.Missile() != nil
	for _, it := range p.Inv {
		switch it.Kind {
		case "egg":
			want("egg", true)
		case "goods":
			want("goods", true)
		case "clutter":
			want("collect", true)
		}
	}
	want("brew", !s.Shown["brew"] && len(g.alIngredients()) > 0)

	// locks within two tiles
	if !s.Shown["lk"] {
	scan:
		for dy := -2; dy <= 2; dy++ {
			for dx := -2; dx <= 2; dx++ {
				x, y := p.X+dx, p.Y+dy
				if zn.In(x, y) && zn.At(x, y) == TLocked {
					g.h2Want("lk")
					break scan
				}
			}
		}
		if !s.Shown["lk"] {
			for _, L := range g.lkSt().Locks {
				if L.Kind == "chest" && !L.Open && !L.Broken && abs(L.X-p.X) <= 2 && abs(L.Y-p.Y) <= 2 && L.ID == lkID(zn.Key, L.X, L.Y) {
					g.h2Want("lk")
					break
				}
			}
		}
	}

	// secrets: a hint given in this zone
	if !s.Shown["secret"] {
		if z := sxSt(g).Z[zn.Key]; z != nil {
			for _, sc := range z.S {
				if sc.Hint && !sc.Found {
					g.h2Want("secret")
					break
				}
			}
		}
	}

	// genre devices
	switch g.Genre {
	case "cyberpunk":
		if !s.Shown["gchack"] {
			for _, d := range g.gcCy().Devs {
				if d.K != "ad" && abs(d.X-p.X) <= 3 && abs(d.Y-p.Y) <= 3 && d.ID == gcID(zn.Key, d.X, d.Y) {
					g.h2Want("gchack")
					break
				}
			}
		}
	case "wasteland":
		want("geiger", g.gcRadAt(zn) >= 6)
	case "twilight":
		if !s.Shown["gcbeacon"] {
			for id, b := range g.gcTw().Beacons {
				if id == gcID(zn.Key, b.X, b.Y) && abs(b.X-p.X) <= 8 && abs(b.Y-p.Y) <= 8 {
					g.h2Want("gcbeacon")
					break
				}
			}
		}
	}
	if !s.Shown["gd"] && gdGenre(g) {
		for _, it := range zn.Items {
			if it.Kind == "gddev" && abs(it.X-p.X) <= 3 && abs(it.Y-p.Y) <= 3 {
				g.h2Want("gd")
				break
			}
		}
		if !s.Shown["gd"] {
			for dy := -2; dy <= 2 && !s.Shown["gd"]; dy++ {
				for dx := -2; dx <= 2; dx++ {
					if zn.In(p.X+dx, p.Y+dy) && zn.At(p.X+dx, p.Y+dy) == TAltar {
						g.h2Want("gd")
						break
					}
				}
			}
		}
	}

	// creatures in sight
	sneakOK := len(g.ssAbilities()) > 0
	for _, a := range zn.Actors {
		if a == p || a.HP <= 0 || !g.visible(a.X, a.Y) {
			continue
		}
		d := dist(a, p)
		hostile := g.Hostile(a) || a.Angry
		if a.Has("nx") {
			g.h2Want("nemesis")
		} else if g.rvOf(a) != nil {
			g.h2Want("rival")
		}
		if hostile && a.AI != "follower" {
			if missile && d >= 3 && g.aaClear(a, "fire") {
				g.h2Want("aim")
			}
			if sneakOK && !a.Angry && d >= 4 && d <= 12 {
				g.h2Want("sneak")
			}
			continue
		}
		if a.AI == "follower" || d > 6 {
			continue
		}
		if !s.Shown["lx"] {
			if _, f, foreign := g.lxForeign(a); foreign && f < 25 {
				g.h2Want("lx")
			}
		}
		if !s.Shown["person"] && lvPersonOf != nil {
			if lp := lvPersonOf(g, a); lp != nil && lp.Met {
				g.h2Want("person")
			}
		}
	}
}
