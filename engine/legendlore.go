package engine

import (
	"fmt"
	"sort"
	"strings"
)

// Legendary arms and unique bosses tied to the world (prefix lb). Each planet gets 10 to 20 named
// legendary items bound to a real chronicle event and a real person (a dead king's sword, a saint's
// relic, a weapon of a famous battle). Everything is generated with side rngs (seed:lb:...), never g.R.
// An arm has a history chain (made by, carried by, lost at), 2 to 4 powers composed from channels the
// engine already reads (see lbFxs: Item.Elem, Item.Cat via categoryBonusDamage, Bonus attributes, AV/DS,
// the gearlore heirloom effects Extra["ggfx"] with gglv 3, and passive effects run here), sometimes a
// curse or a level condition, and a lead chain to find it: a rumour in a town of its home or a rival
// adventurer (read-only rv data) names a dungeon (S1); at S1 a dead bearer's note (a Kind "lore" item or
// a carved inscription on level 1) names the lair (S2); at S2 a named unique guardian, a strong version
// of the site's boss with a backstory from the same event and a signature trick (lbBossGimmick, consumed
// by delvemech_boss.go), carries it. Arms grow through three awakenings (kills of a kind, days carried
// and bosses slain, more kills and quests done). Screen: mode "lb" (palette "Legendary arms").
// State g.ext("lb"), planet-local. Only ever one arm per ~20 dungeons by chance; most need the leads.

func init() {
	unLocalKeys = append(unLocalKeys, "lb")
	zoneHooks = append(zoneHooks, lbZone)
	turnHooks = append(turnHooks, lbTurn)
	killHooks = append(killHooks, lbKill)
	talkHooks = append(talkHooks, lbTalk)
	modeKeys["lb"] = lbKeys
	modeDraws["lb"] = func(g *Game) { g.lbDraw() }
	paEntries = append(paEntries, paEntry{Name: "Legendary arms", Keywords: "legend legendary relic sword artifact awaken leads history unique boss", Cat: "Screen", Label: "screen",
		Func: func(g *Game) { g.Mode, g.lb().tab, g.lb().sel = "lb", 0, 0 }})
}

type lbSite struct {
	WX    int    `json:"wx"`
	WY    int    `json:"wy"`
	ZX    int    `json:"zx"`
	ZY    int    `json:"zy"`
	Name  string `json:"n"`
	Theme string `json:"th"`
	Depth int    `json:"d"`
}

func (s lbSite) key() string { return siteKey(s.WX, s.WY, s.ZX, s.ZY) }

type lbArm struct {
	ID      int
	Key     string
	Name    string
	Noun    string
	Class   string // weapon armor crown relic tome
	Skill   string
	Slot    string
	Tier    int
	Person  string
	Title   string
	Home    int
	HomeP   int
	EvKind  string
	EvText  string
	Year    int
	EvP     int
	Hist    []string
	Fx      []string
	Curse   string
	CondLv  int
	Boss    string
	BossTpl string
	Back    string
	Gim     string
	S1, S2  lbSite
	Kill    string
	KillN   int
	Variant int
	Lore    string
}

type lbProg struct {
	L1    bool   `json:"l1,omitempty"`
	L2    bool   `json:"l2,omitempty"`
	Found bool   `json:"f,omitempty"`
	Slain bool   `json:"s,omitempty"`
	Via   string `json:"v,omitempty"`
	Ins   int    `json:"in,omitempty"`
	InsZ  string `json:"iz,omitempty"`
}

type lbState struct {
	P   map[string]*lbProg `json:"p,omitempty"`
	Log []string           `json:"l,omitempty"`

	arms    []*lbArm
	armsFor string
	at      map[string][]*lbArm
	tab     int
	sel     int
}

func (g *Game) lb() *lbState {
	s := g.ext("lb", func() any { return &lbState{P: map[string]*lbProg{}} }).(*lbState)
	if s.P == nil {
		s.P = map[string]*lbProg{}
	}
	return s
}

func (g *Game) lbProg(a *lbArm) *lbProg {
	st := g.lb()
	if st.P[a.Key] == nil {
		st.P[a.Key] = &lbProg{}
	}
	return st.P[a.Key]
}

var lbGimmicks = map[string]string{}

// lbBossGimmick is the signature trick of a unique boss by name ("" when it is not one). Consumed by
// delvemech_boss.go; the guardian also carries the flag "lbg:<id>".
func lbBossGimmick(name string) string { return lbGimmicks[name] }

// ---------------------------------------------------------------- generation

func (g *Game) lbArms() []*lbArm {
	st := g.lb()
	if g.W == nil {
		return nil
	}
	if st.armsFor == g.W.Seed && st.arms != nil {
		return st.arms
	}
	st.armsFor = g.W.Seed
	st.arms = g.lbGen()
	st.at = map[string][]*lbArm{}
	for _, a := range st.arms {
		st.at[a.S1.key()] = append(st.at[a.S1.key()], a)
		if a.S2.key() != a.S1.key() {
			st.at[a.S2.key()] = append(st.at[a.S2.key()], a)
		}
		lbGimmicks[a.Boss] = a.Gim
	}
	return st.arms
}

func lbShort(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	s = s[:n]
	if i := strings.LastIndex(s, " "); i > n/2 {
		s = s[:i]
	}
	return strings.TrimRight(s, ",;: ") + "..."
}

func lbSiteOK(s *Site, second bool) bool {
	if s == nil || s.Kind != "dungeon" || s.Outlaw != nil {
		return false
	}
	if !second {
		return s.Depth >= 1
	}
	switch s.Theme {
	case "catacomb", "barrow", "sunken", "arcane", "chasm", "mine", "vault":
		return s.Depth >= 2
	}
	return false
}

// lbSitesNear lists dungeon sites in rings around parasang p, nearest first.
func (g *Game) lbSitesNear(p, maxR, want int, second bool) []lbSite {
	w := g.W
	cx, cy := p%w.W, p/w.W
	var out []lbSite
	for r := 0; r <= maxR && len(out) < want; r++ {
		for dy := -r; dy <= r; dy++ {
			for dx := -r; dx <= r; dx++ {
				if maxi(abs(dx), abs(dy)) != r {
					continue
				}
				x, y := cx+dx, cy+dy
				if !w.In(x, y) || w.IsWater(w.Idx(x, y)) {
					continue
				}
				for zy := 0; zy < 3; zy++ {
					for zx := 0; zx < 3; zx++ {
						s := g.siteAt(x, y, zx, zy)
						if lbSiteOK(s, second) {
							out = append(out, lbSite{WX: x, WY: y, ZX: zx, ZY: zy, Name: s.Name, Theme: s.Theme, Depth: s.Depth})
						}
					}
				}
			}
		}
	}
	return out
}

func (g *Game) lbBurgNear(p int) (int, int) {
	w := g.W
	cx, cy := p%w.W, p/w.W
	for r := 0; r <= 8; r++ {
		for dy := -r; dy <= r; dy++ {
			for dx := -r; dx <= r; dx++ {
				if maxi(abs(dx), abs(dy)) != r || !w.In(cx+dx, cy+dy) {
					continue
				}
				q := w.Idx(cx+dx, cy+dy)
				if id := w.Burg[q]; id > 0 && w.burgByID[id] != nil {
					return id, w.burgByID[id].P
				}
			}
		}
	}
	return 0, 0
}

var lbRoles = map[string][]string{
	"war": {"general", "hero", "ruler"}, "capture": {"general", "ruler"}, "peace": {"ruler", "general"}, "fall": {"ruler", "heir"},
	"coup": {"ruler", "heir", "general"}, "death": {"ruler", "heir"}, "monster": {"hero", "general"}, "disaster": {"hero", "heir", "priest"},
	"eco_disaster": {"hero", "inventor", "magician"}, "faith": {"prophet", "priest"}, "god": {"prophet", "priest", "magician"}, "wonder": {"hero", "magician", "ruler"},
}

func (g *Game) lbGen() []*lbArm {
	w := g.W
	r := NewRng(w.Seed + ":lb:gen")
	n := 10 + int(HashStr(w.Seed+":lb:n")%11)
	var cand []int
	for i := range w.Events {
		e := &w.Events[i]
		if _, ok := lbEvClasses[e.Kind]; ok && e.P >= 0 && e.P < len(w.Biome) && !w.IsWater(e.P) {
			cand = append(cand, i)
		}
	}
	if len(cand) == 0 {
		return nil
	}
	var arms []*lbArm
	usedEv := map[int]bool{}
	usedS := map[string]bool{}
	usedName := map[string]bool{}
	for try := 0; try < n*12 && len(arms) < n; try++ {
		ei := cand[r.Int(0, len(cand)-1)]
		if usedEv[ei] {
			continue
		}
		e := &w.Events[ei]
		a := &lbArm{ID: len(arms), EvKind: e.Kind, EvText: e.Text, Year: e.Y, EvP: e.P}
		a.Key = fmt.Sprintf("%s:%d", w.Seed, ei)
		// the person
		roles := lbRoles[e.Kind]
		var pool, any []*PersonData
		for i := range w.People {
			pp := &w.People[i]
			ok := false
			for _, ro := range roles {
				if pp.Role == ro {
					ok = true
				}
			}
			if !ok {
				continue
			}
			any = append(any, pp)
			if e.St > 0 && pp.State == e.St {
				pool = append(pool, pp)
			}
		}
		if len(pool) == 0 {
			pool = any
		}
		var pp *PersonData
		if len(pool) > 0 {
			pp = pool[r.Int(0, len(pool)-1)]
		}
		home, homeP := 0, 0
		if pp != nil && pp.Burg > 0 && w.burgByID[pp.Burg] != nil {
			home, homeP = pp.Burg, w.burgByID[pp.Burg].P
		} else {
			home, homeP = g.lbBurgNear(e.P)
		}
		if home == 0 {
			continue
		}
		a.Home, a.HomeP = home, homeP
		if pp != nil {
			a.Person, a.Title = pp.Name, nonEmpty(pp.Title, pp.Role)
		} else {
			a.Person, a.Title = "a nameless captain", "captain"
		}
		// sites
		prefer := lbSitePrefer[e.Kind]
		s2c := g.lbSitesNear(e.P, 14, 8, true)
		var s2 *lbSite
		for i := range s2c {
			if usedS[s2c[i].key()] {
				continue
			}
			for _, th := range prefer {
				if s2c[i].Theme == th && s2 == nil {
					s2 = &s2c[i]
				}
			}
		}
		if s2 == nil {
			for i := range s2c {
				if !usedS[s2c[i].key()] {
					s2 = &s2c[i]
					break
				}
			}
		}
		if s2 == nil {
			continue
		}
		var s1 *lbSite
		s1c := g.lbSitesNear(homeP, 12, 6, false)
		for i := range s1c {
			if !usedS[s1c[i].key()] && s1c[i].key() != s2.key() && s1 == nil {
				s1 = &s1c[i]
			}
		}
		if s1 == nil {
			continue
		}
		a.S1, a.S2 = *s1, *s2
		usedS[s1.key()], usedS[s2.key()], usedEv[ei] = true, true, true
		// name and class
		for k := 0; k < 30; k++ {
			a.Name = Pick(r, lbNamePre) + Pick(r, lbNameSuf)
			if !usedName[a.Name] {
				break
			}
		}
		usedName[a.Name] = true
		a.Class = Pick(r, lbEvClasses[e.Kind])
		a.Tier = 4 + r.Int(0, 1)
		switch a.Class {
		case "weapon":
			a.Skill = Pick(r, []string{"longblade", "shortblade", "axe", "cudgel"})
		case "armor":
			a.Slot = Pick(r, []string{"body", "hands", "back", "feet"})
		case "relic":
			a.Noun = Pick(r, lbNounRelic)
		case "tome":
			a.Noun = Pick(r, lbNounTome)
		}
		// powers
		code := map[string]string{"weapon": "w", "armor": "a", "crown": "a", "relic": "r", "tome": "r"}[a.Class]
		var fxs []lbFx
		for _, f := range lbFxs {
			if strings.Contains(f.For, code) {
				fxs = append(fxs, f)
			}
		}
		cnt := 2 + r.Int(0, 2)
		hasGG := false
		for len(a.Fx) < cnt && len(fxs) > 0 {
			j := r.Int(0, len(fxs)-1)
			f := fxs[j]
			fxs = append(fxs[:j], fxs[j+1:]...)
			if f.GG > 0 {
				if hasGG {
					continue
				}
				hasGG = true
			}
			a.Fx = append(a.Fx, f.ID)
		}
		if r.Chance(0.4) {
			a.Curse = Pick(r, lbCurseIDs)
		}
		if r.Chance(0.3) {
			a.CondLv = 6 + r.Int(0, 6)
		}
		ks := lbKillByEv[e.Kind]
		a.Kill, a.KillN = ks[r.Int(0, len(ks)-1)], 12
		a.Variant = r.Int(0, 1)
		// the history chain
		maker := ""
		var makers []*PersonData
		for i := range w.People {
			if q := &w.People[i]; q.Role == "inventor" || q.Role == "magician" || q.Role == "artist" {
				makers = append(makers, q)
			}
		}
		if len(makers) > 0 && r.Chance(0.5) {
			maker = makers[r.Int(0, len(makers)-1)].Name
		} else if cu := w.cultByID[w.Culture[homeP]]; cu != nil && len(cu.Given) > 0 {
			maker = Pick(r, cu.Given) + " " + Pick(r, lbMakerTitle)
		} else {
			maker = "an unnamed smith"
		}
		place := w.RegionName(e.P)
		if b := w.burgByID[w.Burg[e.P]]; b != nil {
			place = b.Name
		}
		homeName := w.burgByID[home].Name
		made := e.Y - r.Int(25, 190)
		a.Hist = []string{
			fmt.Sprintf("Y%d: made by %s at %s.", made, maker, homeName),
			fmt.Sprintf("Y%d: carried by %s, %s, through %s.", e.Y, a.Person, a.Title, lbEvPhrase[e.Kind]),
			fmt.Sprintf("Y%d: lost at %s when %s %s. It was never seen again.", e.Y+r.Int(0, 3), place, a.Person, lbFate[e.Kind]),
		}
		// guardian
		a.BossTpl = themes[a.S2.Theme].boss
		a.Gim = lbGimByEv[e.Kind]
		a.Boss = a.Person + " " + Pick(r, lbBossEpithet)
		if e.Kind == "monster" {
			var ms []*MonsterData
			for i := range w.Monsters {
				if !w.Monsters[i].Slain {
					ms = append(ms, &w.Monsters[i])
				}
			}
			if len(ms) > 0 {
				a.Boss = ms[r.Int(0, len(ms)-1)].Name
			}
		}
		if a.Boss == "" {
			a.Boss = "the Last Bearer"
		}
		a.Back = fmt.Sprintf(lbBossBack[e.Kind], a.Person, a.Name)
		arms = append(arms, a)
	}
	sort.SliceStable(arms, func(i, j int) bool { return arms[i].Key < arms[j].Key })
	for i, a := range arms {
		a.ID = i
	}
	return arms
}

// ---------------------------------------------------------------- items

func lbClassCode(class string) string {
	return map[string]string{"weapon": "w", "armor": "a", "crown": "a", "relic": "r", "tome": "r"}[class]
}

func lbFxByID(id string) *lbFx {
	for i := range lbFxs {
		if lbFxs[i].ID == id {
			return &lbFxs[i]
		}
	}
	return nil
}

func lbApplyFx(it *Item, id string) {
	f := lbFxByID(id)
	if f == nil {
		return
	}
	ex := lbEx(it)
	if ex["lbfx:"+id] > 0 {
		return
	}
	ex["lbfx:"+id] = 1
	if f.Apply != nil {
		f.Apply(it)
	}
	if f.GG > 0 {
		ex["ggfx"] = f.GG
		if ex["gglv"] < 3 {
			ex["gglv"], ex["ggxp"] = 3, 200
		}
	}
}

func (g *Game) lbBurgName(id int) string {
	if b := g.W.burgByID[id]; b != nil {
		return b.Name
	}
	return "town"
}

func (g *Game) lbWho(a *lbArm) string {
	return strings.Fields(a.Person + " ")[0]
}

func (a *lbArm) fxNames() []string {
	var out []string
	for _, id := range a.Fx {
		if f := lbFxByID(id); f != nil {
			out = append(out, f.Name)
		}
	}
	return out
}

func (g *Game) lbLore(a *lbArm, aw int) string {
	var parts []string
	parts = append(parts, a.Hist...)
	parts = append(parts, "The chronicle of Y"+fmt.Sprint(a.Year)+": "+lbShort(a.EvText, 130))
	have := mini(len(a.Fx), 2+aw)
	var pw []string
	for i := 0; i < have; i++ {
		if f := lbFxByID(a.Fx[i]); f != nil {
			pw = append(pw, f.Name)
		}
	}
	parts = append(parts, "Powers: "+strings.Join(pw, "; ")+".")
	if have < len(a.Fx) {
		parts = append(parts, fmt.Sprintf("%d more sleep in it.", len(a.Fx)-have))
	}
	if a.Curse != "" {
		parts = append(parts, "Curse: "+lbCurseDefs[a.Curse])
	}
	if a.CondLv > 0 {
		parts = append(parts, fmt.Sprintf("Condition: its gifts sleep in the hands of anyone below level %d.", a.CondLv))
	}
	return strings.Join(parts, " ")
}

func (g *Game) lbMakeItem(a *lbArm) *Item {
	r := NewRng(g.W.Seed + ":lb:item:" + a.Key)
	var base *Item
	switch a.Class {
	case "weapon":
		base = g.MakeWeapon(a.Skill, a.Tier, r)
	case "armor":
		base = g.MakeArmor(a.Slot, a.Tier, r)
	case "crown":
		base = g.MakeArmor("head", a.Tier, r)
	default:
		base = g.MakeOddity(r, a.Tier)
		if a.Class == "tome" {
			base.Name = "codex"
		}
	}
	noun := a.Noun
	if noun == "" {
		f := strings.Fields(base.Name)
		noun = f[len(f)-1]
	}
	name := fmt.Sprintf("%s, %s of %s", a.Name, noun, a.Person)
	it := g.MakeUnique(base, name, g.lbLore(a, 0))
	it.Kind = map[bool]string{true: "artifact", false: base.Kind}[a.Class == "relic" || a.Class == "tome"]
	ex := lbEx(it)
	ex["lb"] = a.ID + 1
	ex["lbaw"] = 0
	for i := 0; i < mini(2, len(a.Fx)); i++ {
		lbApplyFx(it, a.Fx[i])
	}
	if it.Charges > 0 {
		ex["lbcap"] = it.Charges
	}
	it.Text = "Bound to " + a.Person + " and " + lbEvPhrase[a.EvKind] + "."
	return it
}

func (g *Game) lbCarried() []*Item {
	var out []*Item
	for _, it := range g.P.Inv {
		if it.Extra != nil && it.Extra["lb"] > 0 {
			out = append(out, it)
		}
	}
	for _, bp := range g.P.Body {
		if bp.Item != nil && bp.Item.Extra != nil && bp.Item.Extra["lb"] > 0 {
			out = append(out, bp.Item)
		}
	}
	return out
}

func (g *Game) lbArmOf(it *Item) *lbArm {
	arms := g.lbArms()
	i := it.Extra["lb"] - 1
	if i < 0 || i >= len(arms) {
		return nil
	}
	return arms[i]
}

// ---------------------------------------------------------------- placing: zone hook

func lbZone(g *Game, zn *Zone, _ *Rng) {
	if g.W == nil || zn == nil || zn.Z < 1 || zn.Underwater {
		return
	}
	arms := g.lbArms()
	if len(arms) == 0 {
		return
	}
	for _, a := range g.lb().at[siteKey(zn.WX, zn.WY, zn.ZX, zn.ZY)] {
		r := NewRng(g.W.Seed + ":lb:zone:" + a.Key + ":" + zn.Key)
		if zn.Z == 1 && a.S1.key() == siteKey(zn.WX, zn.WY, zn.ZX, zn.ZY) {
			g.lbPlaceTrail(zn, a, r)
		}
		if zn.Z == a.S2.Depth && a.S2.key() == siteKey(zn.WX, zn.WY, zn.ZX, zn.ZY) {
			g.lbPlaceGuardian(zn, a, r)
		}
	}
}

func (g *Game) lbTrailText(a *lbArm) string {
	return fmt.Sprintf("...%s did not come back from %s. The last of us went on, down to %s, %s. %s still holds the way, and %s is with them.",
		a.Person, a.S1.Name, a.S2.Name, "where the old hall is", a.Boss, a.Name)
}

func (g *Game) lbPlaceTrail(zn *Zone, a *lbArm, r *Rng) {
	if a.Variant == 1 {
		if x, y, ok := drSpot(zn, r, true, nil, 0); ok {
			zn.Set(x, y, TInscription)
			zn.Notes[drPack(x, y)] = "Scratched in a hurry: " + g.lbTrailText(a)
			p := g.lbProg(a)
			p.Ins, p.InsZ = drPack(x, y), zn.Key
			return
		}
	}
	x, y, ok := zn.FreeSpot(r, func(x, y int) bool { return evPlain(zn, x, y) && !evNearStair(zn, x, y, 2) })
	if !ok {
		return
	}
	it := &Item{ID: g.newID(), Kind: "lore", Name: Pick(r, lbLoreNames) + " (" + a.Name + ")", Glyph: '?', F: CYellow, Value: 8, Ident: true,
		Text: g.lbTrailText(a), Extra: map[string]int{"lbl": a.ID + 1, "lbs": 2}}
	g.drAddItem(zn, it, x, y)
}

func (g *Game) lbPlaceGuardian(zn *Zone, a *lbArm, r *Rng) {
	var boss *Actor
	for _, o := range zn.Actors {
		if o.Boss && o.HP > 0 && !o.Has("lbboss") {
			boss = o
			break
		}
	}
	if boss == nil || boss.Has("legend") {
		base := g.Spawn(a.BossTpl, zn.Level+3, r)
		base.Boss = true
		base.Set("boss")
		base.MaxHP *= 2
		base.HP = base.MaxHP
		if boss != nil {
			base.X, base.Y = boss.X+1, boss.Y
		} else {
			x, y, ok := zn.FreeSpot(r, nil)
			if !ok {
				return
			}
			base.X, base.Y = x, y
		}
		g.nudge(zn, base)
		zn.Actors = append(zn.Actors, base)
		boss = base
	}
	for i := range boss.Attr {
		boss.Attr[i]++
	}
	boss.Level += 2
	boss.MaxHP = boss.MaxHP * 3 / 2
	boss.HP = boss.MaxHP
	boss.XP = boss.XP * 2
	boss.AVNat++
	boss.Name, boss.Proper = a.Boss, true
	boss.Desc = a.Back
	boss.F = CMagenta
	boss.Set("lbboss")
	boss.Set("lbg:" + a.Gim)
	boss.Inv = append(boss.Inv, g.lbMakeItem(a))
	boss.Gold += 80 * zn.Level
	if zn.Notes != nil && zn.In(boss.X, boss.Y) {
		zn.Notes[boss.Y*MW+boss.X] = "Here " + a.Boss + " keeps " + a.Name + "."
	}
	lbGimmicks[a.Boss] = a.Gim
}

// ---------------------------------------------------------------- leads

func lbDir(dx, dy int) string {
	if abs(dx) <= abs(dy)/2 {
		dx = 0
	}
	if abs(dy) <= abs(dx)/2 {
		dy = 0
	}
	n := map[[2]int]string{{0, -1}: "north", {1, -1}: "north-east", {1, 0}: "east", {1, 1}: "south-east", {0, 1}: "south", {-1, 1}: "south-west", {-1, 0}: "west", {-1, -1}: "north-west", {0, 0}: "right here"}
	return n[[2]int{sign(dx), sign(dy)}]
}

func (g *Game) lbWhere(s lbSite) string {
	wx, wy := g.WX, g.WY
	if g.Cur != nil {
		wx, wy = g.Cur.WX, g.Cur.WY
	}
	dx, dy := s.WX-wx, s.WY-wy
	d := maxi(abs(dx), abs(dy))
	km := g.W.KmPer
	if km <= 0 {
		km = 160
	}
	if d == 0 {
		return "It is in this very parasang"
	}
	return fmt.Sprintf("It lies about %d parasangs %s (around %d km)", d, lbDir(dx, dy), int(float64(d)*km))
}

func (g *Game) lbMarkKnown(s lbSite) {
	if g.W.In(s.WX, s.WY) {
		g.Known[g.W.Idx(s.WX, s.WY)] = true
	}
}

func (g *Game) lbLog(s string) {
	st := g.lb()
	st.Log = append(st.Log, fmt.Sprintf("Day %d: %s", DayOf(g.Turn)+1, s))
	if len(st.Log) > 20 {
		st.Log = st.Log[len(st.Log)-20:]
	}
}

func (g *Game) lbLearn1(a *lbArm, via string) string {
	p := g.lbProg(a)
	if p.L1 {
		return ""
	}
	p.L1, p.Via = true, via
	g.lbMarkKnown(a.S1)
	line := fmt.Sprintf(lbRumourLines[int(HashStr(a.Key))%len(lbRumourLines)], a.Name, a.Person, a.S1.Name, g.lbWhere(a.S1))
	g.addSecret(line, "legend", g.W.Idx(a.S1.WX, a.S1.WY), g.W.Year)
	g.lbLog(fmt.Sprintf("Heard of %s: it went down into %s.", a.Name, a.S1.Name))
	g.give(&Item{ID: g.newID(), Kind: "lore", Name: "rumour of " + a.Name, Glyph: '?', F: CYellow, Value: 2, Ident: true, Text: line, Extra: map[string]int{"lbl": a.ID + 1, "lbs": 1, "lbr": 1}})
	return line
}

func (g *Game) lbLearn2(a *lbArm) {
	p := g.lbProg(a)
	if p.L2 {
		return
	}
	p.L2 = true
	p.L1 = true
	g.lbMarkKnown(a.S2)
	line := fmt.Sprintf("The trail of %s ends at %s. %s. Its guardian is %s.", a.Name, a.S2.Name, g.lbWhere(a.S2), a.Boss)
	g.addSecret(line, "legend", g.W.Idx(a.S2.WX, a.S2.WY), g.W.Year)
	g.lbLog(fmt.Sprintf("Found the trail of %s: %s.", a.Name, a.S2.Name))
	g.Msg("{{Y|" + line + "}}")
	g.deed("quest", fmt.Sprintf("The wanderer %s followed the trail of %s to its lair.", g.P.Name, a.Name), 0, 0)
}

func lbTalk(g *Game, a *Actor) []Option {
	if g.W == nil || g.Cur == nil || g.P == nil || a == nil || a.HP <= 0 || a.Angry || a.AI == "hostile" || a.AI == "follower" {
		return nil
	}
	arms := g.lbArms()
	if len(arms) == 0 {
		return nil
	}
	var out []Option
	if g.Cur.Kind == "town" && g.Cur.Burg > 0 {
		var pick *lbArm
		for _, m := range arms {
			if g.lbProg(m).L1 {
				continue
			}
			here := m.Home == g.Cur.Burg
			if !here {
				if b := g.W.burgByID[g.Cur.Burg]; b != nil && HashStr(m.Key+fmt.Sprint(g.Cur.Burg))%6 == 0 && maxi(abs(b.P%g.W.W-m.HomeP%g.W.W), abs(b.P/g.W.W-m.HomeP/g.W.W)) <= 20 {
					here = true
				}
			}
			if here {
				pick = m
				break
			}
		}
		if pick != nil {
			m := pick
			out = append(out, Option{"Ask whether they know any tales of lost, famous arms.", func() {
				line := g.lbLearn1(m, "a rumour in "+g.lbBurgName(g.Cur.Burg))
				g.talkMenu("\"Old songs, mostly. But there is one the grandmothers tell.\" " + line)
			}})
		}
	}
	if rv := g.rvOf(a); rv != nil {
		for _, m := range arms {
			if !g.lbProg(m).L1 && HashStr(m.Key+rv.Name)%3 == 0 {
				m := m
				out = append(out, Option{"Ask what the other wanderers are after.", func() {
					line := g.lbLearn1(m, "the rival "+rv.Name)
					g.talkMenu("\"I will tell you, because the road is long and you may need a friend.\" " + line + " \"I am after it myself.\"")
				}})
				break
			}
		}
	}
	return out
}

// ---------------------------------------------------------------- turn

func lbTurn(g *Game) {
	if g.P == nil || g.Cur == nil || g.W == nil || g.Dead {
		return
	}
	arms := g.lbArms()
	if len(arms) == 0 {
		return
	}
	st := g.lb()
	// lore items read on pickup, carved notes read by bumping
	for _, it := range g.P.Inv {
		if it.Kind == "lore" && it.Extra != nil && it.Extra["lbl"] > 0 && it.Extra["lbr"] == 0 {
			it.Extra["lbr"] = 1
			if i := it.Extra["lbl"] - 1; i >= 0 && i < len(arms) {
				if it.Extra["lbs"] == 2 {
					g.lbLearn2(arms[i])
				} else {
					g.lbLearn1(arms[i], "a lore page")
				}
			}
		}
	}
	if g.Cur.Z == 1 {
		for _, a := range st.at[siteKey(g.Cur.WX, g.Cur.WY, g.Cur.ZX, g.Cur.ZY)] {
			p := g.lbProg(a)
			if p.Ins > 0 && !p.L2 && p.InsZ == g.Cur.Key && g.Read[g.Cur.Key+fmt.Sprint(p.Ins)] {
				g.lbLearn2(a)
			}
		}
	}
	for _, it := range g.lbCarried() {
		a := g.lbArmOf(it)
		if a == nil {
			continue
		}
		ex := lbEx(it)
		if ex["lbt0"] == 0 {
			ex["lbt0"] = g.Turn + 1
			q := 0
			for _, qq := range g.Quests {
				if qq.Done {
					q++
				}
			}
			ex["lbq0"] = q + 1
			if p := g.lbProg(a); !p.Found {
				p.Found = true
				p.L1, p.L2 = true, true
				g.lbLog(fmt.Sprintf("Took up %s.", a.Name))
				g.Msg("{{Y|You hold " + a.Name + ", the arm of " + a.Person + ". " + a.Hist[2] + "}}")
				g.deed("relic", fmt.Sprintf("The wanderer %s took up %s, once the arm of %s.", g.P.Name, a.Name, a.Person), 0, 0)
			}
		}
		g.lbPassives(it, a)
		g.lbCheckAwaken(it, a)
	}
}

func (g *Game) lbPassives(it *Item, a *lbArm) {
	p := g.P
	ex := lbEx(it)
	dormant := a.CondLv > 0 && p.Level < a.CondLv
	if !dormant {
		if ex["lbp:wind"] > 0 && p.HP*4 < p.MaxHP && g.Turn >= ex["lbsw"] {
			ex["lbsw"] = g.Turn + 180
			p.HP = mini(p.MaxHP, p.HP+p.MaxHP*30/100)
			g.Msg("{{G|" + a.Name + " rallies you.}}")
		}
		if ex["lbp:recharge"] > 0 && g.Turn%150 == 0 && it.Charges < ex["lbcap"] {
			it.Charges++
			g.Msg("{{c|" + a.Name + " gathers a charge.}}")
		}
	}
	switch a.Curse {
	case "thirst":
		if g.Turn%3 == 0 {
			g.Thirst++
		}
	case "blood":
		if g.Turn%30 == 0 && p.HP > 5 {
			g.Msg("{{r|" + a.Name + " takes a little blood.}}")
			g.hurt(p, 1+p.Level/10, nil, "a cursed arm")
		}
	case "unquiet":
		if g.Turn%90 == 0 && NewRng(fmt.Sprintf("%s:lb:uq:%d", g.W.Seed, g.Turn)).Chance(0.35) && g.Cur.Z > 0 {
			rr := NewRng(fmt.Sprintf("%s:lb:uqs:%d", g.W.Seed, g.Turn))
			s := g.Spawn("wisp", g.Cur.Level, rr)
			s.Name = "shade of " + a.Person
			s.X, s.Y = p.X+rr.Int(-4, 4), p.Y+rr.Int(-3, 3)
			g.nudge(g.Cur, s)
			g.Cur.Actors = append(g.Cur.Actors, s)
			g.Msg("{{R|A cold shape of " + a.Person + " drifts up out of the dark, looking for its own.}}")
		}
	}
}

func (g *Game) lbDays(it *Item) int {
	t0 := lbEx(it)["lbt0"]
	if t0 == 0 {
		return 0
	}
	return (g.Turn - t0) / DayLen
}

func (g *Game) lbQuests(it *Item) int {
	q := 0
	for _, qq := range g.Quests {
		if qq.Done {
			q++
		}
	}
	return q - (lbEx(it)["lbq0"] - 1)
}

func (g *Game) lbCheckAwaken(it *Item, a *lbArm) {
	ex := lbEx(it)
	switch ex["lbaw"] {
	case 0:
		if ex["lbk"] >= a.KillN {
			g.lbAwaken(it, a, 1)
		}
	case 1:
		if g.lbDays(it) >= 40 && ex["lbb"] >= 2 {
			g.lbAwaken(it, a, 2)
		}
	case 2:
		if ex["lbk"] >= a.KillN*3 && g.lbQuests(it) >= 3 {
			g.lbAwaken(it, a, 3)
		}
	}
}

func (g *Game) lbAwaken(it *Item, a *lbArm, n int) {
	ex := lbEx(it)
	ex["lbaw"] = n
	switch n {
	case 1, 2:
		if 1+n < len(a.Fx) {
			lbApplyFx(it, a.Fx[1+n])
		} else {
			addBonus(it, []int{STR, AGI, TOU, INT, WIL, EGO}[int(HashStr(a.Key))%6], 1)
		}
		if n == 2 {
			it.DS++
			it.AV++
		}
	case 3:
		for _, k := range []int{int(HashStr(a.Key)) % 6, int(HashStr(a.Key+"b")) % 6} {
			addBonus(it, k, 2)
		}
		if it.DN > 0 {
			it.DS += 2
			it.PV++
		}
		if it.AV > 0 {
			it.AV += 2
		}
		if it.Charges > 0 {
			it.Charges += 3
			ex["lbcap"] += 3
		}
		f := strings.Fields(it.Name)
		_ = f
		it.Name = fmt.Sprintf("%s, Awake", a.Name)
	}
	it.Lore = g.lbLore(a, n)
	g.Msg(fmt.Sprintf("{{Y|%s %s. (%s)}}", a.Name, map[int]string{1: "stirs in your hand", 2: "wakes further", 3: "wakes fully"}[n], lbAwakeNames[n-1]))
	g.deed("relic", fmt.Sprintf("%s woke (%s) in the hands of %s.", a.Name, strings.ToLower(lbAwakeNames[n-1]), g.P.Name), 0, 0)
	g.lbLog(fmt.Sprintf("%s: awakening %d.", a.Name, n))
}

func lbKill(g *Game, d, src *Actor) {
	if g.P == nil || d == nil || g.W == nil {
		return
	}
	if d.Has("lbboss") {
		for _, a := range g.lbArms() {
			if a.Boss == d.Name {
				p := g.lbProg(a)
				p.Slain = true
				g.lbLog(fmt.Sprintf("%s fell.", a.Boss))
				if src == g.P {
					g.Msg("{{M|" + a.Boss + " is done. " + a.Name + " lies among their things.}}")
				}
			}
		}
	}
	if src != g.P {
		return
	}
	tags := g.ggTags(d)
	for _, it := range g.lbCarried() {
		a := g.lbArmOf(it)
		if a == nil {
			continue
		}
		ex := lbEx(it)
		for _, t := range tags {
			if t == a.Kill {
				ex["lbk"]++
				break
			}
		}
		if d.Boss || d.SimMon > 0 {
			ex["lbb"]++
		}
		if a.CondLv > 0 && g.P.Level < a.CondLv {
			continue
		}
		if ex["lbp:reaper"] > 0 && g.P.HP < g.P.MaxHP {
			g.P.HP = mini(g.P.MaxHP, g.P.HP+1+g.P.Level/6)
		}
		if ex["lbp:fortune"] > 0 {
			g.P.Gold += 2 + d.Level
		}
		if ex["lbp:scholar"] > 0 {
			g.gainXP(1 + d.Level/3)
		}
		g.lbCheckAwaken(it, a)
	}
}

// ---------------------------------------------------------------- the screen

func lbKeys(g *Game, k string) {
	st := g.lb()
	rows := g.lbRows()
	switch k {
	case "Escape", "q":
		g.Mode = "main"
	case "Tab", "ArrowRight", "ArrowLeft":
		st.tab = 1 - st.tab
		st.sel = 0
	case "ArrowUp", "N8", "k":
		st.sel = maxi(0, st.sel-1)
	case "ArrowDown", "N2", "j":
		st.sel = mini(maxi(0, len(rows)-1), st.sel+1)
	}
}

// lbRows: the arms the player knows of (heard, followed, or carried).
func (g *Game) lbRows() []*lbArm {
	var out []*lbArm
	for _, a := range g.lbArms() {
		if p := g.lb().P[a.Key]; p != nil && (p.L1 || p.L2 || p.Found) {
			out = append(out, a)
		}
	}
	return out
}

func (g *Game) lbDraw() {
	st := g.lb()
	sc := g.sc
	x, y, w, h := g.panel("Legendary arms")
	bottom := y + h - 1
	arms := g.lbArms()
	rows := g.lbRows()
	tab := func(on bool) string { return map[bool]string{true: "W", false: "K"}[on] }
	sc.TextB(x, y, fmt.Sprintf("{{%s|[ Arms %d/%d ]}}  {{%s|[ Leads and log ]}}", tab(st.tab == 0), len(rows), len(arms), tab(st.tab == 1)), CGray, CBgDark, x+w)
	y += 2
	put := func(s string) {
		for _, ln := range Wrap(s, w) {
			if y < bottom {
				sc.TextB(x, y, ln, CGray, CBgDark, x+w)
				y++
			}
		}
	}
	if st.tab == 1 {
		if len(st.Log) == 0 {
			put("Nothing yet. Ask in towns about lost and famous arms, or follow what you find in old halls.")
		}
		for i := len(st.Log) - 1; i >= 0; i-- {
			put(st.Log[i])
		}
		sc.TextB(x, bottom, "{{K|Tab arms   Esc close}}", CGray, CBgDark, x+w)
		return
	}
	if len(rows) == 0 {
		put(fmt.Sprintf("The world holds %d legendary arms, each bound to a real event and a real name. You know of none yet. Ask in towns, or read what the dead left in old halls.", len(arms)))
		sc.TextB(x, bottom, "{{K|Tab leads   Esc close}}", CGray, CBgDark, x+w)
		return
	}
	st.sel = clampi(st.sel, 0, len(rows)-1)
	for i, a := range rows {
		mark := "  "
		if i == st.sel {
			mark = "> "
		}
		state := "heard of"
		if p := g.lb().P[a.Key]; p != nil && p.Found {
			state = "{{G|yours or found}}"
		} else if p != nil && p.L2 {
			state = "{{Y|lair known}}"
		}
		put(fmt.Sprintf("%s{{W|%s}} (%s) {{K|%s}}", mark, a.Name, a.Class, state))
	}
	y++
	a := rows[st.sel]
	p := g.lbProg(a)
	put("{{Y|" + a.Name + "}}, bound to " + a.Person + ", " + a.Title + ", and " + lbEvPhrase[a.EvKind] + ".")
	put(a.Hist[0])
	if p.L1 || p.Found {
		put(a.Hist[1])
	}
	if p.L2 || p.Found {
		put(a.Hist[2])
	}
	y++
	var held *Item
	for _, it := range g.lbCarried() {
		if it.Extra["lb"] == a.ID+1 {
			held = it
		}
	}
	if p.L1 && !p.L2 {
		put("Lead: it went down into " + a.S1.Name + ". " + g.lbWhere(a.S1) + ". Something there may say where it went next.")
	}
	if p.L2 && !p.Found {
		put("Lead: its lair is " + a.S2.Name + ", " + strings.ToLower(g.lbWhere(a.S2)[:1]) + g.lbWhere(a.S2)[1:] + ". Guardian: " + a.Boss + " (" + drGims[a.Gim].Name + ").")
		put(a.Back)
	}
	if held != nil {
		aw := held.Extra["lbaw"]
		put(fmt.Sprintf("Awakening %d of 3 (%s).", aw, map[bool]string{true: lbAwakeNames[maxi(aw-1, 0)], false: "dormant"}[aw > 0]))
		switch aw {
		case 0:
			put(fmt.Sprintf("Next: slay %d of %s (%d so far).", a.KillN, lbKillWord[a.Kill], held.Extra["lbk"]))
		case 1:
			put(fmt.Sprintf("Next: carry it 40 days (%d) and fell 2 bosses (%d).", g.lbDays(held), held.Extra["lbb"]))
		case 2:
			put(fmt.Sprintf("Next: slay %d of %s (%d), and finish 3 quests with it (%d).", a.KillN*3, lbKillWord[a.Kill], held.Extra["lbk"], maxi(0, g.lbQuests(held))))
		default:
			put("It is fully awake.")
		}
		var pw []string
		for i := 0; i < mini(len(a.Fx), 2+aw); i++ {
			if f := lbFxByID(a.Fx[i]); f != nil {
				pw = append(pw, f.Name+" ("+f.Desc+")")
			}
		}
		put("Powers: " + strings.Join(pw, "; "))
		if a.Curse != "" {
			put("Curse: " + lbCurseDefs[a.Curse])
		}
		if a.CondLv > 0 {
			put(fmt.Sprintf("Condition: gifts dormant below level %d.", a.CondLv))
		}
	} else if p.Found {
		put("You have found it, but it is not on you now.")
	}
	sc.TextB(x, bottom, "{{K|Up/Down choose   Tab leads   Esc close}}", CGray, CBgDark, x+w)
}
