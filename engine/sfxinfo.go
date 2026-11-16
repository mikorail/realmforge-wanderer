package engine

import "strings"

// Sound-event detection for the web shell (prefix sfx): SfxBegin snapshots a few facts before a key press,
// SfxEnd diffs them afterwards and reads the new message-log lines, returning short event ids such as
// "step:grass", "hit:blade", "coin", "door", "levelup". Read-only and cosmetic: it never draws g.R and
// changes no game state (the snapshot lives in g.ext and is harmless in saves).

type sfxSnap struct {
	n, rep, turn int
	last         string
	hp, gold, lv int
	inv, quests  int
	x, y         int
	zone         string
	dead         bool
}

type sfxState struct{ s sfxSnap }

func (g *Game) sfxTake() sfxSnap {
	s := sfxSnap{n: len(g.Msgs), dead: g.Dead}
	if s.n > 0 {
		m := g.Msgs[s.n-1]
		s.last, s.turn, s.rep = m.T, m.Turn, m.Rep
	}
	if g.P != nil {
		s.hp, s.gold, s.lv, s.inv, s.x, s.y = g.P.HP, g.P.Gold, g.P.Level, len(g.P.Inv), g.P.X, g.P.Y
	}
	if g.Cur != nil {
		s.zone = g.Cur.Key
	}
	for _, q := range g.Quests {
		if q != nil && q.Done {
			s.quests++
		}
	}
	return s
}

// SfxBegin records the state just before a key is handled.
func (g *Game) SfxBegin() {
	st := g.ext("sfx", func() any { return &sfxState{} }).(*sfxState)
	st.s = g.sfxTake()
}

func sfxSurface(t Terrain) string {
	switch t {
	case TShallow, TDeep, TFalls, TSpray, THotSpring:
		return "water"
	case TSand, TAsh, TSalt, TQuicksand, TTar:
		return "sand"
	case TSnow, TIce, TSnowDrift, TGlacier:
		return "snow"
	case TWoodFloor, TBridge, TBoat, TScaffold, TBed, TCarpet, TStall:
		return "wood"
	case TMachine, TAnvil:
		return "metal"
	case TGrass, TTallGrass, TMoss, TBush, TReeds, TCrop, TFungus, TDirt, TMushroom, TPalm:
		return "grass"
	}
	return "stone"
}

func sfxWeaponClass(name string) string {
	n := strings.ToLower(name)
	has := func(ws ...string) bool {
		for _, w := range ws {
			if strings.Contains(n, w) {
				return true
			}
		}
		return false
	}
	switch {
	case has("pistol", "rifle", "musket", "gun", "cannon", "blaster", "carbine", "arquebus", "revolver", "shotgun"):
		return "gun"
	case has("bow", "arrow", "sling", "crossbow", "javelin"):
		return "bow"
	case has("spear", "lance", "pike", "dagger", "rapier", "stiletto", "needle", "dirk", "trident", "fang", "claw", "sting", "bite", "tooth"):
		return "pierce"
	case has("club", "mace", "hammer", "cudgel", "staff", "flail", "maul", "fist", "knuckle", "stone", "bone", "tail", "horn", "slam"):
		return "blunt"
	}
	return "blade"
}

func sfxElement(low string) string {
	for _, e := range [][2]string{{"fire", "fire"}, {"flame", "fire"}, {"burn", "fire"}, {"frost", "frost"}, {"ice", "frost"}, {"freez", "frost"},
		{"shock", "shock"}, {"lightning", "shock"}, {"spark", "shock"}, {"acid", "acid"}, {"poison", "poison"}, {"venom", "poison"},
		{"heal", "holy"}, {"bless", "holy"}, {"shadow", "dark"}, {"curse", "dark"}} {
		if strings.Contains(low, e[0]) {
			return e[1]
		}
	}
	return "arcane"
}

// SfxEnd returns the sound events caused since SfxBegin, as []any for the wasm frame ("sfx" field).
func (g *Game) SfxEnd() []any {
	st := g.ext("sfx", func() any { return &sfxState{} }).(*sfxState)
	old, now := st.s, g.sfxTake()
	var ev []string
	add := func(e string) {
		for _, x := range ev {
			if x == e {
				return
			}
		}
		if len(ev) < 8 {
			ev = append(ev, e)
		}
	}
	// new message lines
	var fresh []Msg
	if k := now.n - old.n; k > 0 && k <= len(g.Msgs) {
		fresh = g.Msgs[len(g.Msgs)-k:]
	} else if now.n > 0 && (now.last != old.last || now.turn != old.turn || now.rep != old.rep) {
		fresh = g.Msgs[now.n-1:]
	}
	for _, m := range fresh {
		low := strings.ToLower(m.T)
		mine := strings.HasPrefix(low, "you ") || strings.HasPrefix(low, "{{")
		switch {
		case strings.Contains(low, "fail to penetrate"):
			add("block")
		case strings.Contains(low, "parr"):
			add("parry")
		case strings.Contains(low, " miss"):
			add("miss")
		case strings.Contains(low, " for ") && strings.Contains(low, " damage"):
			cls := "blade"
			if i := strings.Index(low, " with "); i >= 0 {
				cls = sfxWeaponClass(low[i+6:])
			} else if strings.Contains(low, "shoot") {
				cls = "bow"
			}
			if strings.HasPrefix(low, "you shoot") && g.P != nil {
				for _, w := range g.P.Weapons() {
					if w.Kind == "ranged" {
						cls = sfxWeaponClass(w.Name)
					}
				}
			}
			if cls == "gun" || cls == "bow" {
				add("shot:" + cls)
			} else {
				add("hit:" + cls)
			}
			if strings.Contains(low, "critical!") {
				add("crit")
			}
		case strings.Contains(low, "you open the door"):
			add("door")
		case strings.Contains(low, "you close the door") || strings.Contains(low, "door closes") || strings.Contains(low, "slams shut"):
			add("doorclose")
		case strings.Contains(low, "locked") || strings.Contains(low, "unlock") || strings.Contains(low, "lockpick"):
			add("lock")
		case strings.Contains(low, "alarm") || strings.Contains(low, "sound the horn") || strings.Contains(low, "bell tolls"):
			add("alarm")
		case mine && (strings.Contains(low, "you eat") || strings.Contains(low, "you devour") || strings.Contains(low, "you swallow")):
			add("eat")
		case mine && strings.Contains(low, "you drink"):
			add("drink")
		case mine && (strings.HasPrefix(low, "you wear") || strings.HasPrefix(low, "you wield") || strings.HasPrefix(low, "you equip") || strings.HasPrefix(low, "you put on") || strings.HasPrefix(low, "you don ")):
			add("equip")
		case mine && (strings.HasPrefix(low, "you read") || strings.Contains(low, "turn the page") || strings.Contains(low, "you open the book")):
			add("page")
		case strings.Contains(low, "you cast") || strings.Contains(low, "you invoke") || strings.Contains(low, "you channel") || strings.Contains(low, "you chant"):
			add("cast:" + sfxElement(low))
		case strings.Contains(low, "quest") && (strings.Contains(low, "complete") || strings.Contains(low, "fulfilled")):
			add("quest")
		}
	}
	if g.P != nil {
		if now.hp < old.hp && !now.dead {
			add("hurt")
		}
		if now.gold > old.gold {
			add("coin")
		}
		if now.lv > old.lv {
			add("levelup")
		}
		if now.quests > old.quests {
			add("quest")
		}
		if now.inv > old.inv && now.gold <= old.gold && old.zone == now.zone {
			add("pickup")
		}
		if (now.x != old.x || now.y != old.y) && old.zone == now.zone && g.Cur != nil && g.Mode != "world" && !g.OnWorld {
			add("step:" + sfxSurface(g.Cur.At(now.x, now.y)))
		}
	}
	if now.dead && !old.dead {
		add("death")
	}
	out := make([]any, len(ev))
	for i, e := range ev {
		out[i] = e
	}
	return out
}

// SfxScene reports the adaptive-audio facts the music frame lacks, as a map for the wasm frame ("sfxs").
func (g *Game) SfxScene() map[string]any {
	m := map[string]any{"cave": false, "uw": false, "deaf": false, "boss": false, "n": 0}
	if g.Cur == nil || g.P == nil || g.OnWorld {
		return m
	}
	z := g.Cur
	m["uw"] = z.Underwater
	m["cave"] = z.Z > 0 || z.Kind == "dungeon" || z.Kind == "crypt"
	m["deaf"] = g.ijDeaf()
	n, boss := 0, false
	for _, a := range z.Actors {
		if a.HP > 0 && g.visible(a.X, a.Y) && g.Hostile(a) {
			n++
			boss = boss || a.Boss
		}
	}
	m["n"], m["boss"] = n, boss
	return m
}
