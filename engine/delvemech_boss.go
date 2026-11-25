package engine

import (
	"fmt"
	"strings"
)

// Boss fight gimmicks (prefix dr, part of delvemech.go). An enterHook finds the deepest hall's boss and
// records a gimmick for it (the unique bosses of legendlore.go bring their own through lbBossGimmick);
// a turnHook arms it when the player closes in and ticks it. It reads foes.go only through its public
// data: the phase flags bwph0..bwph2 the phase system sets on a boss, the bwshield:/bwopen: resistance
// flags bwProfile reads, bwTypes, and bwSummon. Nothing in foes.go is edited. Gimmicks: pillars (warded
// by crystals that must be shattered), lava and ice (the floor spreads each phase), braziers (a ward
// until they are snuffed), adds on a timer, safezone (cursed ground, the ring tightens each phase),
// mirror (your reflection at half health), and weakness (what the delve story told you, from es_delve).

type drBoss struct {
	Gim   string `json:"g"`
	BID   int    `json:"b"`
	Name  string `json:"n,omitempty"`
	BX    int    `json:"bx"`
	BY    int    `json:"by"`
	Armed bool   `json:"ar,omitempty"`
	Done  bool   `json:"dn,omitempty"`
	Ph    int    `json:"ph,omitempty"`
	T0    int    `json:"t0,omitempty"`
	Last  int    `json:"ls,omitempty"`
	Told  bool   `json:"to,omitempty"`
	Mir   int    `json:"mi,omitempty"` // 1 duel running, 2 over
	Pts   []int  `json:"p,omitempty"`
	Aux   []int  `json:"a,omitempty"`
	Ring  []int  `json:"r,omitempty"`
	R     int    `json:"rr,omitempty"`
}

func (g *Game) drBossActor(b *drBoss) *Actor {
	for _, a := range g.Cur.Actors {
		if a.ID == b.BID && a.HP > 0 {
			return a
		}
	}
	return nil
}

func (g *Game) drBossFeat(b *drBoss) *drFeat {
	return &drFeat{ID: "boss:" + g.Cur.Key, Kind: "bossmech", Tag: g.Cur.Key}
}

func (g *Game) drBossProp(b *drBoss) (*drFeat, int, bool) {
	if b.Gim != "pillars" && b.Gim != "braziers" {
		return nil, 0, false
	}
	for i, p := range b.Pts {
		if maxi(abs(p%MW-g.P.X), abs(p/MW-g.P.Y)) <= 1 {
			return g.drBossFeat(b), i, true
		}
	}
	return nil, 0, false
}

func (g *Game) drPickGimmick(s *Site, boss *Actor, key string) string {
	for f := range boss.Flags {
		if strings.HasPrefix(f, "lbg:") && boss.Flags[f] {
			if _, ok := drGims[strings.TrimPrefix(f, "lbg:")]; ok {
				return strings.TrimPrefix(f, "lbg:")
			}
		}
	}
	if gm := lbBossGimmick(boss.Name); gm != "" {
		if _, ok := drGims[gm]; ok {
			return gm
		}
	}
	h := HashStr("drg:" + key)
	if gm, ok := drGimByTpl[boss.Tpl]; ok && h%2 == 0 {
		return gm
	}
	c := drGimByTheme[s.Theme]
	if len(c) == 0 {
		c = []string{"adds", "pillars"}
	}
	return c[int(h/2)%len(c)]
}

func (g *Game) drBossEnter(zn *Zone) {
	s, lv := evSiteLevel(g, zn)
	if s == nil || lv < s.Depth {
		return
	}
	st := g.dr()
	if st.B[zn.Key] != nil {
		if b := st.B[zn.Key]; !b.Done {
			if h := drGims[b.Gim].Hint; h != "" {
				g.Msg("{{c|" + h + "}}")
			}
		}
		return
	}
	var boss *Actor
	for _, a := range zn.Actors {
		if a.Boss && a.HP > 0 {
			boss = a
			break
		}
	}
	if boss == nil {
		return
	}
	b := &drBoss{Gim: g.drPickGimmick(s, boss, zn.Key), BID: boss.ID, Name: boss.Name, BX: boss.X, BY: boss.Y}
	st.B[zn.Key] = b
	boss.Set("drg")
	if h := drGims[b.Gim].Hint; h != "" {
		g.Msg("{{c|" + h + "}}")
	}
	if weak, _ := evWeakness(g, boss.Tpl); len(weak) > 0 && !g.ev().Done[siteKey(zn.WX, zn.WY, zn.ZX, zn.ZY)] {
		g.Msg("{{K|Perhaps someone wrote down how the master of this place can be hurt. The delve's story might say.}}")
	}
}

func drPopCount(n int) int {
	c := 0
	for ; n > 0; n >>= 1 {
		c += n & 1
	}
	return c
}

// drBossArena: walkable tiles within rad of the boss's start, joined to it.
func drBossArena(zn *Zone, bx, by, rad int) []int {
	seen := map[int]bool{drPack(bx, by): true}
	q := []int{drPack(bx, by)}
	var out []int
	for len(q) > 0 {
		i := q[0]
		q = q[1:]
		out = append(out, i)
		x, y := i%MW, i/MW
		for _, d := range dirs8 {
			nx, ny := x+d[0], y+d[1]
			ni := drPack(nx, ny)
			if !zn.In(nx, ny) || seen[ni] || !zn.Walkable(nx, ny) || maxi(abs(nx-bx), abs(ny-by)) > rad {
				continue
			}
			seen[ni] = true
			q = append(q, ni)
		}
	}
	return out
}

func (g *Game) drFreeFloor(zn *Zone, x, y int) bool {
	if !evFloors[zn.At(x, y)] || zn.At(x, y) == TRubble || zn.ActorAt(x, y) != nil || len(zn.ItemsAt(x, y)) > 0 || (x == g.P.X && y == g.P.Y) {
		return false
	}
	return !evNearStair(zn, x, y, 3)
}

func (g *Game) drBossTick() {
	zn, p := g.Cur, g.P
	b := g.dr().B[zn.Key]
	if b == nil || b.Done {
		return
	}
	boss := g.drBossActor(b)
	if boss == nil {
		g.drBossEnd(b, "")
		return
	}
	if !b.Armed {
		d := dist(boss, p)
		if d > 9 || (d > 5 && !g.visible(boss.X, boss.Y)) {
			return
		}
		g.drBossArm(b, boss)
	}
	ph := 0
	for i := 0; i < 3; i++ {
		if boss.Has(fmt.Sprintf("bwph%d", i)) {
			ph = i + 1
		}
	}
	newPhase := ph > b.Ph
	if newPhase {
		b.Ph = ph
	}
	rr := NewRng(fmt.Sprintf("%s:dr:boss:%s:%d:%d", g.W.Seed, zn.Key, b.Ph, g.Turn))
	switch b.Gim {
	case "pillars":
		if len(b.Pts) > 0 {
			boss.AddSt("shielded", 3)
			if boss.HP < boss.MaxHP {
				boss.HP = mini(boss.MaxHP, boss.HP+maxi(1, boss.MaxHP*len(b.Pts)/150))
			}
		} else {
			delete(boss.Status, "shielded")
		}
	case "lava", "ice":
		if newPhase {
			g.drSpread(b, boss, rr)
		}
		if b.Gim == "lava" && g.Turn%3 == 0 {
			// the heat is felt near the edge of the glow
			if zn.At(p.X, p.Y) != TLava {
				for _, d := range dirs8 {
					if zn.At(p.X+d[0], p.Y+d[1]) == TLava && rr.Chance(0.2) {
						p.AddSt("burning", 1)
						break
					}
				}
			}
		}
	case "adds":
		per := maxi(6, 16-4*b.Ph)
		if !b.Told && g.Turn-b.Last >= per-1 {
			b.Told = true
			g.Msg("{{O|" + Pick(rr, drAddsTell) + "}}")
		}
		if g.Turn-b.Last >= per || newPhase && b.Ph > 0 {
			b.Last, b.Told = g.Turn, false
			alive := 0
			for _, v := range g.bw().Guard {
				if v == boss.ID {
					alive++
				}
			}
			if alive < 6 {
				g.bwSummon(boss, 2, rr)
				g.Msg("{{R|More of them pour in.}}")
			}
		}
	case "safezone":
		if newPhase {
			b.R = maxi(3, b.R-2)
			g.drBossCurse(b, boss)
			g.Msg("{{R|The ring of safe ground draws in.}}")
		}
		here := drPack(p.X, p.Y)
		for _, c := range b.Aux {
			if c == here {
				if g.Turn%2 == 0 {
					g.Msg("{{R|The cursed ground drains you.}}")
				}
				g.drHurt(1+zn.Level/6, "cursed ground")
				break
			}
		}
	case "braziers":
		if len(b.Pts) == 0 {
			for _, t := range bwTypes {
				delete(boss.Flags, "bwshield:"+t)
			}
		}
	case "mirror":
		if b.Mir == 0 && (boss.HP*2 <= boss.MaxHP || b.Ph >= 2) {
			b.Mir = 1
			g.drBossShade(b, boss, rr)
		} else if b.Mir == 1 {
			if g.drFlagged("drbm") > 0 {
				boss.AddSt("shielded", 3)
			} else {
				b.Mir = 2
				delete(boss.Status, "shielded")
				g.Msg("{{G|Your reflection shatters. The master is bare again.}}")
			}
		}
	}
}

func (g *Game) drBossArm(b *drBoss, boss *Actor) {
	zn := g.Cur
	b.Armed, b.T0, b.Last = true, g.Turn, g.Turn
	b.Ring = drBossArena(zn, b.BX, b.BY, 8)
	rr := NewRng(g.W.Seed + ":dr:arm:" + zn.Key)
	gm := drGims[b.Gim]
	g.Msg("{{R|" + gm.Start + "}}")
	switch b.Gim {
	case "pillars", "braziers":
		n, hp := 3, 3
		t := TCrystal
		if b.Gim == "braziers" {
			hp, t = 1, TBrazier
		}
		for try := 0; try < 200 && len(b.Pts) < n; try++ {
			c := b.Ring[rr.Int(0, len(b.Ring)-1)]
			x, y := c%MW, c/MW
			d := maxi(abs(x-b.BX), abs(y-b.BY))
			if d < 3 || d > 6 || !g.drFreeFloor(zn, x, y) || !evSafeBlock(zn, x, y) {
				continue
			}
			zn.Set(x, y, t)
			zn.Notes[c] = map[bool]string{true: "A pillar of humming crystal, bright with stolen strength.", false: "A tall brazier. While it burns the master cannot be harmed."}[b.Gim == "pillars"]
			b.Pts = append(b.Pts, c)
			b.Aux = append(b.Aux, hp)
		}
		if b.Gim == "braziers" && len(b.Pts) > 0 {
			for _, ty := range bwTypes {
				boss.Set("bwshield:" + ty)
			}
		}
		g.computeLight(zn)
	case "safezone":
		b.R = 8
		g.drBossCurse(b, boss)
	}
	// the delve's story: what the player read tells where the master is weak
	key := siteKey(zn.WX, zn.WY, zn.ZX, zn.ZY)
	weak, _ := evWeakness(g, boss.Tpl)
	if len(weak) > 0 {
		ev := g.ev()
		got := drPopCount(ev.Found[key])
		if ev.Done[key] {
			for _, t := range weak {
				boss.Set("bwopen:" + t)
				if g.bw().Known[boss.Tpl] == nil {
					g.bw().Known[boss.Tpl] = map[string]bool{}
				}
				g.bw().Known[boss.Tpl][t] = true
			}
			g.Msg("{{G|You remember the whole story of this place: the master is open to " + strings.Join(weak, ", ") + ". It shows.}}")
		} else if got >= 2 {
			if g.bw().Known[boss.Tpl] == nil {
				g.bw().Known[boss.Tpl] = map[string]bool{}
			}
			g.bw().Known[boss.Tpl][weak[0]] = true
			g.Msg("{{G|What you read of this place comes back to you: it is weak to " + weak[0] + ".}}")
		}
	}
}

// drBossCurse: the tiles of the arena outside the safe ring go foul.
func (g *Game) drBossCurse(b *drBoss, boss *Actor) {
	zn := g.Cur
	have := map[int]bool{}
	for _, c := range b.Aux {
		have[c] = true
	}
	for _, c := range b.Ring {
		x, y := c%MW, c/MW
		if have[c] || maxi(abs(x-b.BX), abs(y-b.BY)) <= b.R || evNearStair(zn, x, y, 2) || !evFloors[zn.At(x, y)] {
			continue
		}
		b.Aux = append(b.Aux, c)
		have[c] = true
		zn.Set(x, y, TAsh)
	}
}

func (g *Game) drSpread(b *drBoss, boss *Actor, rr *Rng) {
	zn := g.Cur
	t := TLava
	word := "The floor splits and runs with fire."
	if b.Gim == "ice" {
		t, word = TIce, "Frost sweeps over the floor."
	}
	limit := len(b.Ring) * 35 / 100
	n := maxi(4, len(b.Ring)/8)
	placed := 0
	have := map[int]bool{}
	for _, c := range b.Pts {
		have[c] = true
	}
	for try := 0; try < 500 && placed < n && len(b.Pts) < limit; try++ {
		c := b.Ring[rr.Int(0, len(b.Ring)-1)]
		x, y := c%MW, c/MW
		if have[c] || !g.drFreeFloor(zn, x, y) || maxi(abs(x-g.P.X), abs(y-g.P.Y)) <= 1 || maxi(abs(x-b.BX), abs(y-b.BY)) < 2 {
			continue
		}
		if len(b.Pts) > 0 {
			near := false
			for _, d := range dirs8 {
				if have[drPack(x+d[0], y+d[1])] {
					near = true
					break
				}
			}
			if !near && try < 300 {
				continue
			}
		}
		zn.Set(x, y, t)
		b.Pts = append(b.Pts, c)
		have[c] = true
		placed++
	}
	g.computeLight(zn)
	g.Msg("{{R|" + word + "}}")
}

func (g *Game) drBossShade(b *drBoss, boss *Actor, rr *Rng) {
	zn := g.Cur
	a := g.Spawn("skeleton", maxi(g.P.Level, 2), rr)
	a.Name, a.Proper, a.Glyph, a.F = "your reflection", true, '@', CDarkGray
	a.Attr = g.P.Attr
	a.MaxHP = maxi(20, g.P.MaxHP*70/100)
	a.HP = a.MaxHP
	a.XP = 60 + 10*g.P.Level
	a.AI, a.Faction = "hostile", "undead"
	a.Set("drbm")
	a.X, a.Y = boss.X+sgn(g.P.X-boss.X), boss.Y+sgn(g.P.Y-boss.Y)
	g.nudge(zn, a)
	zn.Actors = append(zn.Actors, a)
	g.Msg("{{M|The master looks at you, and a copy of you steps out of the glint on its blade. Until the reflection falls, the master cannot be hurt.}}")
}

func (g *Game) drBossKill(d *Actor) {
	if g.Cur == nil || d == nil {
		return
	}
	st := g.dr()
	b := st.B[g.Cur.Key]
	if b == nil || b.Done {
		return
	}
	if d.ID == b.BID {
		g.drBossEnd(b, "The master falls. The hall goes quiet.")
	}
}

func (g *Game) drBossEnd(b *drBoss, line string) {
	zn := g.Cur
	b.Done = true
	switch b.Gim {
	case "lava", "ice":
		for _, c := range b.Pts {
			zn.Set(c%MW, c/MW, TRubble)
		}
	case "pillars", "braziers":
		for _, c := range b.Pts {
			zn.Set(c%MW, c/MW, g.floorFor(zn))
			delete(zn.Notes, c)
		}
	}
	b.Pts = nil
	for _, a := range zn.Actors {
		if a.HP > 0 && a.Has("drbm") {
			a.HP = 0
		}
	}
	g.computeLight(zn)
	if line != "" && b.Armed {
		g.Msg("{{G|" + line + "}}")
	}
}

func (g *Game) drBossMenu(f *drFeat, idx int) (string, []string, []drOpt) {
	b := g.dr().B[f.Tag]
	leave := drOpt{"Step away.", func() bool { return false }}
	if b == nil || b.Done || idx >= len(b.Pts) {
		return "The hall", []string{"Nothing here answers."}, []drOpt{leave}
	}
	zn := g.Cur
	name := drGims[b.Gim].Name
	var lines []string
	var opts []drOpt
	if b.Gim == "pillars" {
		lines = []string{fmt.Sprintf("A crystal pillar, bright with stolen strength (about %d solid blows left). It feeds the master of the hall.", b.Aux[idx])}
		opts = append(opts, drOpt{"Strike the crystal with all you have.", func() bool {
			dmg := 1
			if g.P.M(STR) >= 2 {
				dmg = 2
			}
			g.spend(1000)
			if idx >= len(b.Pts) {
				return false
			}
			b.Aux[idx] -= dmg
			if b.Aux[idx] > 0 {
				g.drSay("Cracks run through it, and it hums higher.")
				return true
			}
			c := b.Pts[idx]
			zn.Set(c%MW, c/MW, g.floorFor(zn))
			delete(zn.Notes, c)
			b.Pts = append(b.Pts[:idx], b.Pts[idx+1:]...)
			b.Aux = append(b.Aux[:idx], b.Aux[idx+1:]...)
			g.computeLight(zn)
			if len(b.Pts) == 0 {
				g.drSay("{{G|The last pillar bursts. The master is bare to your blows.}}")
			} else {
				g.drSay("The pillar bursts in a spray of light.")
			}
			return false
		}})
	} else {
		lines = []string{"A tall brazier, burning with an unnatural light. While any of them burn, the master takes little harm."}
		opts = append(opts, drOpt{"Snuff it (smother the flame).", func() bool {
			g.spend(1000)
			if idx >= len(b.Pts) {
				return false
			}
			c := b.Pts[idx]
			zn.Set(c%MW, c/MW, TPillar)
			delete(zn.Notes, c)
			b.Pts = append(b.Pts[:idx], b.Pts[idx+1:]...)
			b.Aux = append(b.Aux[:idx], b.Aux[idx+1:]...)
			g.computeLight(zn)
			if len(b.Pts) == 0 {
				g.drSay("{{G|The last flame dies. The ward on the master fails.}}")
			} else {
				g.drSay("The flame dies. The ward flickers but holds.")
			}
			return false
		}})
	}
	return name, lines, append(opts, leave)
}
