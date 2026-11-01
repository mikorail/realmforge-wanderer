package engine

import (
	"fmt"
	"sort"
	"strings"
)

// Murim, part 2 (prefix mr): membership, ranks, qi cultivation, techniques, missions, tournament, world
// presence and the G screen. Sect data and generation: murim_sects.go. State: g.ext("murim").

var mrRankCon = [5]int{0, 80, 240, 600, 1400} // contribution needed for each rank
var mrRankRealm = [5]int{0, 1, 2, 3, 5}       // cultivation realm needed for each rank
var mrBreakNeed = [7]int{40, 90, 160, 260, 400, 600, 900}

// mrRoll is the dice for trials and breakthroughs (tests replace it).
var mrRoll = func(g *Game) float64 { return g.R.Float() }

type mrApplied struct {
	Attr [6]int `json:"a"`
	HP   int    `json:"hp"`
	AV   int    `json:"av"`
}

type mrMission struct {
	Kind    string `json:"k"` // hunt courier manual
	Sect    string `json:"s"`
	Target  string `json:"t,omitempty"`
	Need    int    `json:"n"`
	Have    int    `json:"h"`
	Contrib int    `json:"c"`
	Gold    int    `json:"g"`
	WX      int    `json:"wx,omitempty"`
	WY      int    `json:"wy,omitempty"`
	Spawned bool   `json:"sp,omitempty"`
	Dest    int    `json:"d,omitempty"`
	Text    string `json:"tx"`
	Ready   bool   `json:"r,omitempty"`
}

type mrTourn struct {
	Ed      int  `json:"ed"`
	Entered bool `json:"in,omitempty"`
	Wins    int  `json:"w,omitempty"`
	Done    bool `json:"dn,omitempty"`
}

type mrState struct {
	Sect    string          `json:"sect,omitempty"`
	Rank    int             `json:"rank"`
	Contrib int             `json:"con"`
	Total   int             `json:"tot"`
	Done    int             `json:"done"`
	Realm   int             `json:"realm"`
	Prog    int             `json:"prog"`
	Qi      int             `json:"qi"`
	QiInit  bool            `json:"qii,omitempty"`
	Dev     int             `json:"dev"`
	Fav     int             `json:"fav"`
	FavSet  bool            `json:"favs,omitempty"`
	Techs   []string        `json:"techs,omitempty"`
	App     mrApplied       `json:"app"`
	Mission *mrMission      `json:"mis,omitempty"`
	Base    map[string]int  `json:"base,omitempty"`
	Seen    map[string]bool `json:"seen,omitempty"`
	Lock    map[string]int  `json:"lock,omitempty"`
	Tourn   mrTourn         `json:"tourn"`
	Raids   map[string]bool `json:"raids,omitempty"`
	PayDay  int             `json:"pay,omitempty"`
	Coach   map[string]int  `json:"coach,omitempty"`
	PillAt  int             `json:"pillAt,omitempty"`
	Hint    bool            `json:"hint,omitempty"`
}

func (g *Game) mr() *mrState {
	st := g.ext("murim", func() any { return &mrState{} }).(*mrState)
	if st.Seen == nil {
		st.Seen = map[string]bool{}
	}
	if st.Base == nil {
		st.Base = map[string]int{}
	}
	if st.Lock == nil {
		st.Lock = map[string]int{}
	}
	if st.Coach == nil {
		st.Coach = map[string]int{}
	}
	if st.Raids == nil {
		st.Raids = map[string]bool{}
	}
	return st
}

func (g *Game) mrSk() mrSkin { return mrSkinFor(g.W.Genre) }

func (g *Game) mrMine() *MrSect {
	st := g.mr()
	if st.Sect == "" {
		return nil
	}
	return g.SectByID(st.Sect)
}

func mrFlag(a *Actor, prefix string) string {
	for k, v := range a.Flags {
		if v && strings.HasPrefix(k, prefix) {
			return k[len(prefix):]
		}
	}
	return ""
}

func (g *Game) mrRealmName(r int) string { return g.mrSk().Realms[clampi(r, 0, 7)] }

func init() {
	zoneHooks = append(zoneHooks, mrZone)
	talkHooks = append(talkHooks, mrTalk)
	turnHooks = append(turnHooks, mrTurn)
	killHooks = append(killHooks, mrKill)
	enterHooks = append(enterHooks, mrEnter)
	sideHooks = append(sideHooks, mrSide)
	mainKeys["G"] = func(g *Game) { g.Mode, g.Sel, g.Tab = "murim", 0, 0 }
	modeKeys["murim"] = mrKeys
	modeDraws["murim"] = func(g *Game) { g.mrDraw() }
}

// ---- cultivation ----

func (g *Game) mrFav() int {
	st := g.mr()
	if !st.FavSet {
		best := 0
		for i := range g.P.Attr {
			if g.P.Attr[i] > g.P.Attr[best] {
				best = i
			}
		}
		return best
	}
	return st.Fav
}

func mrWant(realm, fav int) mrApplied {
	var w mrApplied
	if realm <= 0 {
		return w
	}
	for i := range w.Attr {
		w.Attr[i] = (realm + 1) / 3
	}
	w.Attr[fav] += (realm + 1) / 2
	w.HP = 8 * realm
	w.AV = realm / 2
	return w
}

// mrApply makes the player's attributes, HP and armour match the current realm (idempotent).
func (g *Game) mrApply() {
	st := g.mr()
	want := mrWant(st.Realm, g.mrFav())
	p := g.P
	for i := range want.Attr {
		p.Attr[i] += want.Attr[i] - st.App.Attr[i]
	}
	dh := want.HP - st.App.HP
	p.MaxHP += dh
	if dh > 0 {
		p.HP += dh
	}
	p.HP = clampi(p.HP, 1, p.MaxHP)
	p.AVNat += want.AV - st.App.AV
	st.App = want
}

func (g *Game) mrMaxQi() int {
	st := g.mr()
	return maxi(10, 20+12*st.Realm+2*g.P.M(WIL))
}

func (g *Game) mrQi() int {
	st := g.mr()
	if !st.QiInit {
		st.QiInit, st.Qi = true, g.mrMaxQi()
	}
	return st.Qi
}

func (g *Game) mrHere() *MrSect {
	if g.Cur == nil || g.Cur.Z != 0 || g.Cur.ZX != 1 || g.Cur.ZY != 1 {
		return nil
	}
	for _, s := range g.Sects() {
		if s.WX == g.Cur.WX && s.WY == g.Cur.WY {
			return s
		}
	}
	return nil
}

func (g *Game) mrBreakChance() float64 {
	st := g.mr()
	c := 0.75 - 0.06*float64(st.Realm) + 0.03*float64(g.P.M(WIL)) + 0.01*float64(g.P.M(INT)) - 0.02*float64(st.Dev)
	if g.mrHere() != nil {
		c += 0.10
	}
	if st.Rank >= 3 {
		c += 0.05
	}
	if c < 0.15 {
		c = 0.15
	}
	if c > 0.95 {
		c = 0.95
	}
	return c
}

// mrMeditate spends n turns gathering qi. Returns the progress gained.
func (g *Game) mrMeditate(n int) int {
	st := g.mr()
	if st.Realm >= 7 {
		g.Msg("There is no higher realm to reach by sitting.")
		return 0
	}
	if g.nearHostile() {
		g.Msg("You can't meditate with enemies in sight.")
		return 0
	}
	per := 8 + 2*g.P.M(WIL) + 2*st.Realm
	if per < 3 {
		per = 3
	}
	if g.mrHere() != nil {
		per *= 2
	}
	gain := 0
	for i := 0; i < n && !g.Dead; i++ {
		g.spend(1000)
		if g.nearHostile() {
			g.Msg("{{R|Your meditation is broken!}}")
			break
		}
		if i%10 == 9 {
			gain += per / 3 * 1
		}
	}
	gain = maxi(gain, 1)
	g.mrAddProg(gain)
	st.Qi = g.mrMaxQi()
	g.Msg(fmt.Sprintf("{{C|You gather %s. (+%d cultivation)}}", g.mrSk().Qi, gain))
	return gain
}

func (g *Game) mrAddProg(n int) {
	st := g.mr()
	need := mrBreakNeed[clampi(st.Realm, 0, 6)]
	st.Prog = mini(st.Prog+n, need*3/2)
}

// mrOnRest: resting inside a sect headquarters feeds cultivation a little.
func (g *Game) mrOnRest(turns int) {
	if g.mrHere() != nil && turns > 0 && g.mr().Realm < 7 {
		g.mrAddProg(maxi(1, turns/10))
	}
}

// mrBreakthrough tries to reach the next realm. Failure risks a qi deviation.
func (g *Game) mrBreakthrough() bool {
	st := g.mr()
	if st.Realm >= 7 {
		g.Msg("You have reached the end of the path.")
		return false
	}
	need := mrBreakNeed[st.Realm]
	if st.Prog < need {
		g.Msg(fmt.Sprintf("Your %s is not yet full (%d/%d).", g.mrSk().Qi, st.Prog, need))
		return false
	}
	if g.nearHostile() {
		g.Msg("You can't attempt a breakthrough with enemies in sight.")
		return false
	}
	g.spend(1000)
	if mrRoll(g) < g.mrBreakChance() {
		st.Realm++
		st.Prog = 0
		st.Qi = 0
		st.QiInit = false
		if !st.FavSet {
			st.Fav, st.FavSet = g.mrFav(), true
		}
		g.mrApply()
		g.mrQi()
		g.Msg(fmt.Sprintf("{{W|Breakthrough! You enter the %s realm.}}", g.mrRealmName(st.Realm)))
		g.deed("murim", fmt.Sprintf("%s broke through to the %s realm.", g.P.Name, g.mrRealmName(st.Realm)), 0, 0)
		g.gainXP(30 * st.Realm)
		return true
	}
	// qi deviation
	st.Dev++
	st.Prog /= 2
	dmg := maxi(1, g.P.MaxHP*35/100)
	g.P.HP = maxi(1, g.P.HP-dmg)
	g.P.AddSt("stunned", 3)
	g.Msg(fmt.Sprintf("{{R|Qi deviation! Your %s runs wild (-%d HP).}}", g.mrSk().Qi, dmg))
	if mrRoll(g) < 0.2 && g.P.Attr[WIL] > 4 {
		g.P.Attr[WIL]--
		g.Msg("{{R|Your meridians are scarred: Willpower falls by 1.}}")
	}
	return false
}

// ---- items ----

func (g *Game) mrPill(tier int) *Item {
	tier = clampi(tier, 0, 2)
	return &Item{ID: g.newID(), Name: g.mrSk().Pills[tier], Glyph: '!', F: []byte{CGreen, CCyan, CYellow}[tier], Kind: "mrpill", Effect: "qi",
		Power: []int{30, 70, 140}[tier], Tier: tier, Value: []int{30, 90, 250}[tier], Weight: 1, Ident: true,
		Rarity: []string{"uncommon", "rare", "ultrarare"}[tier], Lore: "A cultivation pill. Taking several in a short while poisons the body."}
}

func (g *Game) mrManual(techID string) *Item {
	s, t := g.mrTech(techID)
	if t == nil {
		return nil
	}
	return &Item{ID: g.newID(), Name: fmt.Sprintf("%s (%s)", t.Name, g.mrSk().Manual), Glyph: '?', F: CYellow, Kind: "mrmanual", Effect: techID, Weight: 1,
		Value: 150 + 150*t.Pow, Ident: true, Rarity: "rare", Lore: fmt.Sprintf("A secret manual of %s: %s. Needs the %s realm.", s.Name, s.Art.Name, g.mrRealmName(t.Need))}
}

// mrUseItem handles pills and manuals; returns true when the item was handled.
func (g *Game) mrUseItem(it *Item) bool {
	st := g.mr()
	switch it.Kind {
	case "mrpill":
		if st.Realm >= 7 {
			g.Msg("It would do nothing for you now.")
			return true
		}
		toxic := st.PillAt != 0 && g.Turn-st.PillAt < 120
		st.PillAt = g.Turn
		g.mrAddProg(it.Power)
		g.Msg(fmt.Sprintf("{{G|You swallow the %s. (+%d cultivation)}}", it.Name, it.Power))
		if toxic {
			g.P.AddSt("poisoned", 6)
			g.P.HP = maxi(1, g.P.HP-maxi(2, g.P.MaxHP/10))
			g.Msg("{{G|The pills clash in your blood. You are poisoned.}}")
		}
		g.consume(it)
		g.spend(1000)
		return true
	case "mrmanual":
		_, t := g.mrTech(it.Effect)
		if t == nil {
			return true
		}
		if mrHas(st.Techs, it.Effect) {
			g.Msg("You already know this technique.")
			return true
		}
		g.mrAddProg(20)
		if st.Realm < t.Need {
			g.Msg(fmt.Sprintf("{{y|You can follow only a little of it. Reach the %s realm to use %s.}}", g.mrRealmName(t.Need), t.Name))
		}
		g.mrLearn(it.Effect)
		g.removeInv(it)
		g.spend(1000)
		return true
	}
	return false
}

func mrHas(a []string, s string) bool {
	for _, x := range a {
		if x == s {
			return true
		}
	}
	return false
}

// ---- techniques ----

func (g *Game) mrTech(id string) (*MrSect, *MrTech) {
	i := strings.LastIndex(id, ":")
	if i < 0 {
		return nil, nil
	}
	s := g.SectByID(id[:i])
	if s == nil || len(id) != i+2 || id[i+1] < '0' || id[i+1] > '2' {
		return nil, nil
	}
	return s, &s.Art.Tech[id[i+1]-'0']
}

func (g *Game) mrLearn(id string) bool {
	st := g.mr()
	s, t := g.mrTech(id)
	if t == nil || mrHas(st.Techs, id) {
		return false
	}
	st.Techs = append(st.Techs, id)
	g.Msg(fmt.Sprintf("{{W|You learn %s}} {{K|(%s)}}.", t.Name, s.Art.Name))
	return true
}

func mrAbID(techID string) string { return "mr_" + techID }

// mrAbilities lists learned techniques for Abilities().
func (g *Game) mrAbilities() []Ability {
	st := g.mr()
	var out []Ability
	for _, id := range st.Techs {
		_, t := g.mrTech(id)
		if t == nil {
			continue
		}
		tg := "tile"
		switch t.Kind {
		case "heal", "sweep", "fear":
			tg = "self"
		}
		ab := mrAbID(id)
		out = append(out, Ability{ID: ab, Name: fmt.Sprintf("%s [%d %s]", t.Name, t.Cost, g.mrSk().Qi), Target: tg, Cool: g.P.Cool[ab]})
	}
	return out
}

func (g *Game) mrFoeAt(tx, ty, r int) *Actor {
	d := g.Cur.ActorAt(tx, ty)
	if d == nil || d == g.P || d.HP <= 0 {
		g.Msg("There is nothing there to target.")
		return nil
	}
	if maxi(abs(tx-g.P.X), abs(ty-g.P.Y)) > r {
		g.Msg("{{K|Too far away.}}")
		return nil
	}
	return d
}

func (g *Game) mrSwing(d *Actor, extra int) bool {
	w := g.P.Weapons()[0]
	before := d.HP
	g.swing(g.P, d, w)
	hit := d.HP < before
	if hit && d.HP > 0 && extra > 0 {
		g.hurt(d, extra, g.P, "qi")
	}
	return hit
}

func (g *Game) mrAdjacent() []*Actor {
	var out []*Actor
	for _, a := range g.Cur.Actors {
		if a.HP > 0 && a != g.P && dist(a, g.P) == 1 && g.Hostile(a) {
			out = append(out, a)
		}
	}
	return out
}

func (g *Game) mrRush(tx, ty, n int) bool {
	moved := 0
	for _, pt := range line(g.P.X, g.P.Y, tx, ty) {
		if pt[0] == tx && pt[1] == ty {
			break
		}
		if moved >= n || !g.Cur.Walkable(pt[0], pt[1]) || g.Cur.ActorAt(pt[0], pt[1]) != nil {
			return false
		}
		g.P.X, g.P.Y = pt[0], pt[1]
		moved++
	}
	return maxi(abs(tx-g.P.X), abs(ty-g.P.Y)) <= 1
}

// mrUse runs the technique with ability id "mr_<sect>:<n>" (called from usePower).
func (g *Game) mrUse(id string, tx, ty int) {
	tid := strings.TrimPrefix(id, "mr_")
	st := g.mr()
	_, t := g.mrTech(tid)
	if t == nil || !mrHas(st.Techs, tid) {
		return
	}
	if g.P.Cool[id] > 0 {
		g.Msg(fmt.Sprintf("%s is recharging (%d turns).", t.Name, g.P.Cool[id]))
		return
	}
	if st.Realm < t.Need {
		g.Msg(fmt.Sprintf("{{y|%s needs the %s realm.}}", t.Name, g.mrRealmName(t.Need)))
		return
	}
	if g.mrQi() < t.Cost {
		g.Msg(fmt.Sprintf("{{y|Not enough %s (%d/%d).}}", g.mrSk().Qi, st.Qi, t.Cost))
		return
	}
	if !g.mrRun(t, tx, ty) {
		return
	}
	st.Qi -= t.Cost
	if g.P.Cool == nil {
		g.P.Cool = map[string]int{}
	}
	g.P.Cool[id] = t.Cool
	g.spend(1000)
}

func (g *Game) mrRun(t *MrTech, tx, ty int) bool {
	p := g.P
	st := g.mr()
	lv := p.Level
	rl := st.Realm
	switch t.Kind {
	case "palm":
		d := g.mrFoeAt(tx, ty, 1)
		if d == nil {
			return false
		}
		g.Msg(fmt.Sprintf("{{W|%s!}}", t.Name))
		if g.mrSwing(d, 3+rl+lv/4+t.Pow) && d.HP > 0 && mrRoll(g) < 0.5 {
			d.AddSt("stunned", 2)
		}
	case "dash":
		d := g.mrFoeAt(tx, ty, 6)
		if d == nil {
			return false
		}
		if !g.mrRush(d.X, d.Y, 5) {
			g.Msg("{{K|The way is blocked or too far.}}")
			return false
		}
		g.Msg(fmt.Sprintf("{{W|%s!}}", t.Name))
		g.mrSwing(d, 2+rl+lv/5+t.Pow)
	case "burst":
		g.Msg(fmt.Sprintf("{{C|%s!}}", t.Name))
		for i, pt := range line(p.X, p.Y, tx, ty) {
			if i > 6+rl || g.Cur.Opaque(pt[0], pt[1]) {
				break
			}
			if d := g.Cur.ActorAt(pt[0], pt[1]); d != nil && d != p {
				dmg := g.R.Dice(2+rl/2+t.Pow, 6)
				g.Msg(fmt.Sprintf("The blast strikes %s for {{C|%d}}.", g.the(d), dmg))
				g.hurt(d, dmg, p, "qi")
				break
			}
		}
	case "heal":
		h := 10 + 2*lv + 5*rl + 4*t.Pow
		p.HP = mini(p.MaxHP, p.HP+h)
		delete(p.Status, "bleeding")
		delete(p.Status, "poisoned")
		g.Msg(fmt.Sprintf("{{G|%s. (+%d HP)}}", t.Name, h))
	case "sweep":
		foes := g.mrAdjacent()
		if len(foes) == 0 {
			g.Msg("No foes beside you.")
			return false
		}
		g.Msg(fmt.Sprintf("{{W|%s!}}", t.Name))
		for _, d := range foes {
			if g.mrSwing(d, 1+rl/2) && d.HP > 0 {
				d.AddSt("dazed", 2)
			}
		}
	case "drain":
		d := g.mrFoeAt(tx, ty, 1)
		if d == nil {
			return false
		}
		dmg := 4 + 2*rl + 2*t.Pow + lv/3
		g.Msg(fmt.Sprintf("{{R|%s!}}", t.Name))
		g.hurt(d, dmg, p, "qi")
		p.HP = mini(p.MaxHP, p.HP+dmg/2)
	case "fear":
		n := 0
		for _, a := range g.Cur.Actors {
			if a.HP > 0 && a != p && dist(a, p) <= 3 && g.Hostile(a) {
				a.AddSt("dazed", 3)
				if a.Level <= lv+rl {
					a.AddSt("stunned", 2)
				}
				n++
			}
		}
		if n == 0 {
			g.Msg("No foes close enough to feel it.")
			return false
		}
		g.Msg(fmt.Sprintf("{{M|%s! Your foes falter.}}", t.Name))
	case "flurry":
		d := g.mrFoeAt(tx, ty, 1)
		if d == nil {
			return false
		}
		g.Msg(fmt.Sprintf("{{W|%s!}}", t.Name))
		for i := 0; i < 3 && d.HP > 0; i++ {
			g.mrSwing(d, 0)
		}
	default:
		return false
	}
	return true
}

// ---- membership ----

func (g *Game) mrJoin(s *MrSect) {
	st := g.mr()
	st.Sect, st.Rank, st.Contrib, st.Mission = s.ID, 0, 0, nil
	st.Seen[s.ID] = true
	if !st.FavSet || st.Realm == 0 {
		st.Fav, st.FavSet = s.Art.Stat, true
		g.mrApply()
	}
	g.mrLearn(s.Art.Tech[0].ID)
	g.mrSyncRep()
	g.Msg(fmt.Sprintf("{{G|You are accepted into %s as %s.}}", s.Name, g.mrSk().Ranks[0]))
	g.deed("murim", fmt.Sprintf("%s joined %s.", g.P.Name, s.Name), 0, 0)
}

func (g *Game) mrLeave(s *MrSect) {
	st := g.mr()
	st.Sect, st.Rank, st.Contrib, st.Mission = "", 0, 0, nil
	st.Lock[s.ID] = g.Turn + 3*DayLen
	g.mrSyncRep()
	g.Rep[s.RepKey()] -= 150
	g.Msg("{{R|You are cast out. Your former brothers will remember.}}")
}

// mrPromote raises the rank by one if contribution and realm allow. Rank 4 needs a duel (mrChallenge).
func (g *Game) mrPromote(s *MrSect) (bool, string) {
	st := g.mr()
	if st.Rank >= 3 {
		return false, "Only a duel with the leader can take the seat."
	}
	nr := st.Rank + 1
	if st.Contrib < mrRankCon[nr] {
		return false, fmt.Sprintf("You need %d contribution (you have %d).", mrRankCon[nr], st.Contrib)
	}
	if st.Realm < mrRankRealm[nr] {
		return false, fmt.Sprintf("You need the %s realm.", g.mrRealmName(mrRankRealm[nr]))
	}
	st.Rank = nr
	if nr <= 2 {
		g.mrLearn(s.Art.Tech[nr].ID)
	}
	g.gainXP(60 * nr)
	g.deed("murim", fmt.Sprintf("%s became %s of %s.", g.P.Name, g.mrSk().Ranks[nr], s.Name), 0, 0)
	return true, fmt.Sprintf("You are raised to %s.", g.mrSk().Ranks[nr])
}

func (g *Game) mrAddContrib(n int, why string) {
	st := g.mr()
	if st.Sect == "" || n <= 0 {
		return
	}
	st.Contrib += n
	st.Total += n
	g.Msg(fmt.Sprintf("{{G|+%d contribution}}%s.", n, why))
}

// mrSyncRep keeps g.Rep["mr:<sect>"] at the baseline implied by who you serve (hostility follows from it).
func (g *Game) mrSyncRep() {
	st := g.mr()
	mine := g.mrMine()
	for _, s := range g.Sects() {
		b := 0
		switch {
		case mine != nil && mine.ID == s.ID:
			b = 400
		case s.Align == mrDemonic:
			b = -400
		case mine != nil && mine.Align == mrDemonic && s.Align == mrOrthodox:
			b = -300
		case mine != nil && mrHas(mine.Rivals, s.ID):
			b = -300
		}
		k := s.RepKey()
		g.Rep[k] += b - st.Base[k]
		st.Base[k] = b
	}
}

// ---- NPCs ----

func mrColor(s *MrSect) byte {
	switch s.Align {
	case mrOrthodox:
		return CCyan
	case mrUnorthodox:
		return CYellow
	}
	return CRed
}

func (g *Game) mrDisciple(s *MrSect, lvl int, r *Rng, rank int) *Actor {
	a := g.Spawn("bandit", lvl, r)
	a.Name, a.Proper = g.cultureName(g.W.cultByID[s.Culture], r), true
	a.Title = strings.ToLower(g.mrSk().Ranks[clampi(rank, 0, 3)]) + " of " + s.Short()
	a.Desc = fmt.Sprintf("A %s of %s, trained in the %s.", strings.ToLower(g.mrSk().Ranks[clampi(rank, 0, 3)]), s.Name, s.Art.Name)
	a.Faction, a.AI, a.F = s.RepKey(), "wander", mrColor(s)
	a.Set("mrd:" + s.ID)
	a.Attr[s.Art.Stat] += 2 + rank
	a.MaxHP += 4 * rank
	a.HP = a.MaxHP
	return a
}

func (g *Game) mrTalker(role string, s *MrSect, bd *BurgData, r *Rng, title string) *Actor {
	cu := g.W.cultByID[bd.Culture]
	a := g.makeNPC(role, bd, cu, r)
	a.Faction = s.RepKey()
	a.Title = title
	a.F = mrColor(s)
	a.AI = "still"
	return a
}

func (g *Game) mrSpawnNear(zn *Zone, a *Actor, minD, maxD int) bool {
	ok := func(x, y int) bool {
		d := abs(x-g.P.X) + abs(y-g.P.Y)
		return d >= minD && d <= maxD
	}
	x, y, found := zn.FreeSpot(g.R, ok)
	if !found {
		x, y, found = zn.FreeSpot(g.R, nil)
	}
	if !found {
		return false
	}
	a.X, a.Y, a.HomeX, a.HomeY = x, y, x, y
	zn.Actors = append(zn.Actors, a)
	return true
}

// ---- zone hooks ----

func mrZone(g *Game, zn *Zone, _ *Rng) {
	if zn.Z != 0 {
		return
	}
	ss := g.Sects()
	if len(ss) == 0 {
		return
	}
	w := g.W
	key := w.Seed + ":murim:z:" + zn.Key
	switch zn.Kind {
	case "town":
		bd := w.burgByID[zn.Burg]
		if bd == nil {
			return
		}
		for _, s := range ss {
			for _, h := range s.Halls {
				if h.Burg != bd.ID || zn.ZX != 1 || zn.ZY != 1 {
					continue
				}
				r := NewRng(key + ":hall:" + s.ID)
				a := g.mrTalker("scholar", s, bd, r, "hall master of "+s.Short())
				a.Desc = fmt.Sprintf("The keeper of the %s of %s in %s. %s", g.mrSk().Hall, s.Name, bd.Name, s.Creed)
				a.Level += 2
				a.Set("mrhall:" + s.ID)
				mrPlace(zn, a, r)
			}
		}
		if hb, zx, zy := g.mrHostZone(); hb != nil && hb.ID == bd.ID && zn.ZX == zx && zn.ZY == zy {
			r := NewRng(key + ":marshal")
			a := g.mrTalker("guard", ss[0], bd, r, "marshal of the "+g.mrSk().Union)
			a.Faction = fmt.Sprintf("state:%d", bd.State)
			a.Desc = "The marshal who keeps the " + g.mrSk().Union + " tournament rolls."
			a.Set("mrmarshal")
			mrPlace(zn, a, r)
		}
	case "surface":
		for _, s := range ss {
			if s.WX == zn.WX && s.WY == zn.WY && zn.ZX == 1 && zn.ZY == 1 {
				g.mrBuildHQ(zn, s, key)
				return
			}
		}
		r := NewRng(key + ":wander")
		if !r.Chance(0.07) {
			return
		}
		// prefer sects with a presence nearby
		var near []*MrSect
		for _, s := range ss {
			if mrCheb(w, w.Idx(zn.WX, zn.WY), w.Idx(s.WX, s.WY)) <= 12 {
				near = append(near, s)
			}
		}
		if len(near) == 0 {
			return
		}
		s := near[r.Int(0, len(near)-1)]
		n := 2 + r.Int(0, 2)
		if s.Align == mrDemonic {
			n++
		}
		lvl := g.zoneLevel(zn.WX, zn.WY, 0) + r.Int(0, 2)
		for i := 0; i < n; i++ {
			a := g.mrDisciple(s, lvl, r, 0)
			mrPlace(zn, a, r)
		}
	}
}

func mrPlace(zn *Zone, a *Actor, r *Rng) {
	x, y, ok := zn.FreeSpot(r, nil)
	if !ok {
		return
	}
	a.X, a.Y, a.HomeX, a.HomeY = x, y, x, y
	zn.Actors = append(zn.Actors, a)
}

func (g *Game) mrBuildHQ(zn *Zone, s *MrSect, key string) {
	mrBuildCompound(zn)
	zn.Name = s.HQName
	var keep []*Actor
	for _, a := range zn.Actors {
		if !mrInCompound(a.X, a.Y) {
			keep = append(keep, a)
		}
	}
	zn.Actors = keep
	var its []*Item
	for _, it := range zn.Items {
		if !mrInCompound(it.X, it.Y) {
			its = append(its, it)
		}
	}
	zn.Items = its
	p := g.W.Idx(zn.WX, zn.WY)
	bd := &BurgData{ID: -1, Name: s.HQName, State: s.State, Culture: s.Culture, P: p, Alive: true}
	r := NewRng(key + ":hq:" + s.ID)
	lead := g.mrTalker("elder", s, bd, r, strings.ToLower(s.LeaderTit)+" of "+s.Short())
	lead.Name, lead.Proper = s.Leader, true
	lead.Level += 8
	lead.MaxHP += 60
	lead.HP = lead.MaxHP
	lead.Desc = fmt.Sprintf("%s, %s of %s. Master of the %s.", s.Leader, strings.ToLower(s.LeaderTit), s.Name, s.Art.Name)
	lead.Set("mrlead:" + s.ID)
	lead.X, lead.Y, lead.HomeX, lead.HomeY = 40, 6, 40, 6
	zn.Actors = append(zn.Actors, lead)
	for i, el := range s.Elders {
		e := g.mrTalker("scholar", s, bd, r, "elder of "+s.Short())
		e.Name, e.Proper = el, true
		e.Level += 4
		e.MaxHP += 30
		e.HP = e.MaxHP
		e.Desc = fmt.Sprintf("%s, an elder of %s.", el, s.Name)
		e.Set("mrelder:" + s.ID)
		e.X, e.Y = 34+i*6, 8
		e.HomeX, e.HomeY = e.X, e.Y
		zn.Actors = append(zn.Actors, e)
	}
	lvl := g.zoneLevel(zn.WX, zn.WY, 0) + 2
	for i := 0; i < 6; i++ {
		a := g.mrDisciple(s, lvl, r, i%3)
		x, y, ok := zn.FreeSpot(r, func(x, y int) bool { return mrInCompound(x, y) && y >= 12 && y <= 20 })
		if !ok {
			continue
		}
		a.X, a.Y, a.HomeX, a.HomeY = x, y, x, y
		zn.Actors = append(zn.Actors, a)
	}
	// vault: a secret manual of the house, plus a pill
	for k, t := range []int{1, 2} {
		if m := g.mrManual(s.Art.Tech[t].ID); m != nil {
			m.X, m.Y = 62, 15+k*2
			zn.Items = append(zn.Items, m)
		}
	}
	pl := g.mrPill(1 + r.Int(0, 1))
	pl.X, pl.Y = 62, 17
	zn.Items = append(zn.Items, pl)
}

// ---- enter / turn / side ----

func mrEnter(g *Game, zn *Zone) {
	ss := g.Sects()
	if len(ss) == 0 || g.P == nil {
		return
	}
	st := g.mr()
	g.mrSyncRep()
	if zn.Z == 0 && zn.Kind == "town" {
		for _, s := range ss {
			for _, h := range s.Halls {
				if h.Burg == zn.Burg && !st.Seen[s.ID] {
					st.Seen[s.ID] = true
					g.Msg(fmt.Sprintf("{{c|You notice the banner of %s over a %s.}}", s.Name, g.mrSk().Hall))
				}
			}
		}
		// demonic cult raid
		day := DayOf(g.Turn)
		if rd := g.mrRaidAt(day); rd != nil && rd.Burg == zn.Burg && !st.Raids[rd.ID] {
			st.Raids[rd.ID] = true
			if c := g.mrCult(); c != nil {
				g.Msg(fmt.Sprintf("{{R|Raiders of the %s storm the town!}}", c.Name))
				r := NewRng(g.W.Seed + ":murim:raidmen:" + rd.ID)
				lvl := g.zoneLevel(zn.WX, zn.WY, 0) + 2
				for i := 0; i < 4; i++ {
					a := g.mrDisciple(c, lvl, r, 1)
					a.Angry = true
					g.mrSpawnNear(zn, a, 6, 20)
				}
			}
		}
	}
	if s := g.mrHere(); s != nil && zn.Z == 0 {
		if !st.Seen[s.ID] {
			st.Seen[s.ID] = true
		}
		g.Msg(fmt.Sprintf("{{Y|%s: the walls of %s rise before you.}}", s.HQName, s.Name))
	}
	// missions
	m := st.Mission
	if m == nil || m.Ready || zn.Z != 0 {
		return
	}
	switch {
	case m.Kind == "manual" && !m.Spawned && zn.WX == m.WX && zn.WY == m.WY && zn.ZX == 1 && zn.ZY == 1:
		m.Spawned = true
		if x, y, ok := zn.FreeSpot(g.R, func(x, y int) bool { return abs(x-g.P.X)+abs(y-g.P.Y) > 8 }); ok {
			zn.Items = append(zn.Items, &Item{ID: g.newID(), Kind: "trophy", Name: "stolen manual", Glyph: '?', F: CYellow, X: x, Y: y,
				Text: "mr-manual:" + m.Sect, Lore: "The sect's stolen manual.", Value: 40, Weight: 1, Ident: true})
			g.Msg("{{Y|A thief's camp... and a familiar manual among the loot.}}")
			for i := 0; i < 3; i++ {
				b := g.Spawn("bandit", maxi(zn.Level, 3), g.R)
				b.Angry = true
				g.mrSpawnNear(zn, b, 5, 14)
			}
		}
	case (m.Kind == "hunt" || m.Kind == "courier") && zn.Kind == "surface" && m.Have < m.Need && g.R.Chance(0.5):
		tid := m.Target
		if m.Kind == "courier" {
			tid = ""
			if s := g.SectByID(m.Sect); s != nil && len(s.Rivals) > 0 {
				tid = s.Rivals[0]
			}
		}
		rv := g.SectByID(tid)
		if rv == nil {
			return
		}
		n := 0
		for _, a := range zn.Actors {
			if a.HP > 0 && a.Has("mrd:"+rv.ID) {
				n++
			}
		}
		if n >= 2 {
			return
		}
		lvl := maxi(zn.Level, g.P.Level-1)
		for i := 0; i < 2; i++ {
			a := g.mrDisciple(rv, lvl, g.R, 1)
			a.Angry = true
			g.mrSpawnNear(zn, a, 8, 22)
		}
		g.Msg(fmt.Sprintf("{{R|Disciples of %s lie in wait!}}", rv.Name))
	}
}

func mrTurn(g *Game) {
	if g.Cur == nil || g.P == nil || len(g.Sects()) == 0 {
		return
	}
	st := g.mr()
	if st.Realm > 0 || st.Sect != "" || len(st.Techs) > 0 {
		if g.Turn%3 == 0 && g.mrQi() < g.mrMaxQi() {
			st.Qi++
		}
	}
	day := DayOf(g.Turn)
	if st.Sect != "" && st.Rank >= 3 && st.PayDay != day {
		st.PayDay = day
		g.P.Gold += 15 * st.Rank
	}
	if m := st.Mission; m != nil && m.Kind == "manual" && !m.Ready && g.Turn%10 == 0 {
		for _, it := range g.P.Inv {
			if it.Kind == "trophy" && it.Text == "mr-manual:"+m.Sect {
				m.Have, m.Ready = m.Need, true
				g.Msg("{{G|You have recovered the manual. Return it to the hall.}}")
				break
			}
		}
	}
}

func mrSide(g *Game) []string {
	if len(g.Sects()) == 0 {
		return nil
	}
	st := g.mr()
	var out []string
	if st.Sect != "" || st.Realm > 0 || len(st.Techs) > 0 {
		sk := g.mrSk()
		s := fmt.Sprintf("{{c|%s}} %s %d/%d", g.mrRealmName(st.Realm), sk.Qi, g.mrQi(), g.mrMaxQi())
		out = append(out, s)
	}
	if m := st.Mission; m != nil {
		out = append(out, fmt.Sprintf("{{y|Sect task:}} %d/%d", m.Have, m.Need))
	}
	if g.Cur != nil && g.Cur.Kind == "town" {
		if hb, _, _ := g.mrHostZone(); hb != nil && hb.ID == g.Cur.Burg {
			if _, act, _ := mrTournament(DayOf(g.Turn)); act {
				out = append(out, "{{R|Alliance tournament is open}}")
			}
		}
	}
	return out
}

// ---- kills ----

func mrKill(g *Game, d, src *Actor) {
	if len(g.Sects()) == 0 || g.P == nil {
		return
	}
	st := g.mr()
	byP := src == g.P || (src != nil && (src.Faction == "player" || src.AI == "follower"))
	// tournament bout
	if d.Has("mrbout") && byP {
		g.mrBoutWon()
		return
	}
	if id := mrFlag(d, "mrduel:"); id != "" && byP {
		if s := g.SectByID(id); s != nil && st.Sect == "" {
			g.mrJoin(s)
		}
		return
	}
	if id := mrFlag(d, "mrlead:"); id != "" && d.Angry && byP {
		s := g.SectByID(id)
		if s != nil && st.Sect == id && st.Rank == 3 {
			st.Rank = 4
			g.Msg(fmt.Sprintf("{{W|You are now %s of %s!}}", g.mrSk().Ranks[4], s.Name))
			g.deed("murim", fmt.Sprintf("%s seized the leadership of %s.", g.P.Name, s.Name), 0, 0)
			g.gainXP(300)
		}
		return
	}
	id := mrFlag(d, "mrd:")
	if id == "" || !byP {
		return
	}
	s := g.SectByID(id)
	if s == nil {
		return
	}
	mine := g.mrMine()
	if mine == nil || mine.ID != id {
		g.Rep[s.RepKey()] -= 40
	}
	if mine != nil && mine.ID != id && (mrHas(mine.Rivals, id) || (s.Align == mrDemonic && mine.Align == mrOrthodox)) {
		g.mrAddContrib(4+st.Rank, " for striking a foe of the sect")
	}
	if m := st.Mission; m != nil && m.Kind == "hunt" && !m.Ready && m.Target == id {
		m.Have++
		if m.Have >= m.Need {
			m.Ready = true
			g.Msg("{{G|The hunt is done. Report to the hall.}}")
		} else {
			g.Msg(fmt.Sprintf("{{y|Hunt: %d/%d}}", m.Have, m.Need))
		}
	}
	// loot
	var drop *Item
	switch {
	case g.R.Chance(0.04):
		drop = g.mrManual(s.Art.Tech[g.R.Int(0, 2)].ID)
	case g.R.Chance(0.12):
		drop = g.mrPill(g.R.Int(0, 1))
	}
	if drop != nil {
		drop.X, drop.Y = d.X, d.Y
		g.Cur.Items = append(g.Cur.Items, drop)
		g.Msg("{{Y|Something falls from the body: " + drop.Name + ".}}")
	}
}

// ---- missions ----

func (g *Game) mrNewMission(s *MrSect) *mrMission {
	st := g.mr()
	r := NewRng(fmt.Sprintf("%s:murim:mission:%s:%d:%d", g.W.Seed, s.ID, DayOf(g.Turn), st.Done))
	kinds := []string{"hunt", "courier", "manual"}
	k := kinds[r.Int(0, 2)]
	base := 25 + 10*st.Rank + r.Int(0, 15)
	m := &mrMission{Sect: s.ID, Contrib: base, Gold: 20 + 15*st.Rank + r.Int(0, 20)}
	if k == "courier" {
		var dests []int
		for _, h := range s.Halls {
			if g.Cur == nil || h.Burg != g.Cur.Burg {
				dests = append(dests, h.Burg)
			}
		}
		if len(dests) == 0 {
			k = "hunt"
		} else {
			m.Dest = dests[r.Int(0, len(dests)-1)]
			bd := g.W.burgByID[m.Dest]
			m.Need, m.Contrib = 1, base+10
			m.Text = fmt.Sprintf("Carry the sealed message to our %s in %s. Rivals will try to stop you.", g.mrSk().Hall, bd.Name)
		}
	}
	if k == "manual" {
		var cs [][2]int
		cx, cy := g.Cur.WX, g.Cur.WY
		for dy := -7; dy <= 7; dy++ {
			for dx := -7; dx <= 7; dx++ {
				d := maxi(abs(dx), abs(dy))
				x, y := cx+dx, cy+dy
				if d < 3 || !g.W.In(x, y) || g.W.IsWater(g.W.Idx(x, y)) || g.W.Burg[g.W.Idx(x, y)] > 0 {
					continue
				}
				cs = append(cs, [2]int{x, y})
			}
		}
		if len(cs) == 0 {
			k = "hunt"
		} else {
			c := cs[r.Int(0, len(cs)-1)]
			m.WX, m.WY, m.Need = c[0], c[1], 1
			m.Contrib += 15
			m.Text = fmt.Sprintf("A thief fled with our %s. Find the camp %s and bring it back.", g.mrSk().Manual, g.mrCompass(c[0], c[1]))
		}
	}
	if k == "hunt" {
		tgt := ""
		if len(s.Rivals) > 0 {
			tgt = s.Rivals[r.Int(0, len(s.Rivals)-1)]
		}
		rv := g.SectByID(tgt)
		if rv == nil {
			rv = g.mrCult()
		}
		m.Target, m.Need = rv.ID, mini(4, 2+st.Rank/2)
		m.Text = fmt.Sprintf("Defeat %d disciples of %s on the roads.", m.Need, rv.Name)
	}
	m.Kind = k
	return m
}

func (g *Game) mrFinish(s *MrSect) string {
	st := g.mr()
	m := st.Mission
	if m == nil || !m.Ready {
		return ""
	}
	st.Mission = nil
	st.Done++
	g.P.Gold += m.Gold
	g.mrAddContrib(m.Contrib, "")
	g.gainXP(25 + 10*st.Rank)
	for i, it := range g.P.Inv {
		if it.Kind == "trophy" && it.Text == "mr-manual:"+m.Sect {
			g.P.Inv = append(g.P.Inv[:i], g.P.Inv[i+1:]...)
			break
		}
	}
	return fmt.Sprintf("Well done. The sect remembers. (+%d contribution, +%d %s)", m.Contrib, m.Gold, g.Sk.Currency)
}

// ---- tournament ----

func (g *Game) mrBoutFoe(round int) {
	ss := g.Sects()
	st := g.mr()
	r := NewRng(fmt.Sprintf("%s:murim:bout:%d:%d", g.W.Seed, st.Tourn.Ed, round))
	s := ss[r.Int(0, len(ss)-1)]
	a := g.mrDisciple(s, g.P.Level+round, r, 2)
	a.Title = "tournament champion of " + s.Short()
	a.Angry, a.AI = true, "hostile"
	a.Faction = "mrbout"
	a.Set("mrbout")
	g.mrSpawnNear(g.Cur, a, 4, 12)
	g.Msg(fmt.Sprintf("{{R|Round %d: %s of %s enters the ring!}}", round+1, a.Name, s.Name))
}

func (g *Game) mrBoutWon() {
	st := g.mr()
	st.Tourn.Wins++
	if st.Tourn.Wins < 3 {
		g.Msg("{{G|Your opponent yields. The crowd roars.}}")
		g.mrBoutFoe(st.Tourn.Wins)
		return
	}
	st.Tourn.Done = true
	g.Msg("{{W|You are champion of the tournament!}}")
	g.P.Gold += 100
	g.mrAddContrib(40, " for the tournament crown")
	g.gainXP(120)
	var prize *Item
	if ss := g.Sects(); len(ss) > 0 {
		s := ss[NewRng(fmt.Sprintf("%s:murim:prize:%d", g.W.Seed, st.Tourn.Ed)).Int(0, len(ss)-1)]
		prize = g.mrManual(s.Art.Tech[1].ID)
	}
	if prize != nil {
		g.give(prize)
		g.Msg("{{Y|The Alliance awards you " + prize.Name + ".}}")
	}
	g.deed("murim", g.P.Name+" won the martial tournament.", 0, 0)
}

// ---- rumors ----

func (g *Game) mrRumors() []string {
	var out []string
	ss := g.Sects()
	if len(ss) == 0 {
		return nil
	}
	day := DayOf(g.Turn)
	for _, w := range g.mrWarsAt(day) {
		a, b := g.SectByID(w.A), g.SectByID(w.B)
		if a != nil && b != nil {
			out = append(out, fmt.Sprintf("{{R|War}}: %s and %s are at blades' point near %s (ends in about %d days).", a.Name, b.Name, g.mrCompass(a.WX, a.WY), maxi(1, w.End-day)))
		}
	}
	if rd := g.mrRaidAt(day); rd != nil {
		if bd := g.W.burgByID[rd.Burg]; bd != nil {
			out = append(out, fmt.Sprintf("{{R|Raid}}: the %s is burning %s.", g.mrCult().Name, bd.Name))
		}
	}
	ed, act, to := mrTournament(day)
	_ = ed
	if hb := g.mrHost(); hb != nil {
		if act {
			out = append(out, fmt.Sprintf("{{Y|The %s tournament is under way in %s.}}", g.mrSk().Union, hb.Name))
		} else {
			out = append(out, fmt.Sprintf("The %s tournament opens in %s in %d days.", g.mrSk().Union, hb.Name, to))
		}
	}
	return out
}

// mrGossip returns a sect rumor for a gossiping NPC or "".
func (g *Game) mrGossip(a *Actor) string {
	rs := g.mrRumors()
	if len(rs) == 0 {
		return ""
	}
	r := NewRng(fmt.Sprintf("%d:murim:%d", a.ID, g.Turn/600))
	if !r.Chance(0.35) {
		return ""
	}
	return "\"" + Plain(Pick(r, rs)) + "\""
}

// ---- talk ----

func mrTalk(g *Game, a *Actor) []Option {
	if len(g.Sects()) == 0 {
		return nil
	}
	if id := mrFlag(a, "mrhall:"); id != "" {
		if s := g.SectByID(id); s != nil {
			return []Option{{fmt.Sprintf("{{M|Speak of %s.}}", s.Name), func() {
				g.mr().Seen[s.ID] = true
				g.mrHallMenu(a, s, "\""+s.Creed+"\"")
			}}}
		}
	}
	if id := mrFlag(a, "mrlead:"); id != "" {
		if s := g.SectByID(id); s != nil {
			return []Option{{fmt.Sprintf("{{M|Address the %s.}}", strings.ToLower(s.LeaderTit)), func() { g.mrLeadMenu(a, s, "\"State your business.\"") }}}
		}
	}
	if id := mrFlag(a, "mrelder:"); id != "" {
		if s := g.SectByID(id); s != nil {
			return []Option{{"{{M|Seek an elder's instruction.}}", func() { g.mrElderMenu(a, s, "\"The way is long. What do you wish to learn?\"") }}}
		}
	}
	if a.Has("mrmarshal") {
		return []Option{{fmt.Sprintf("{{M|Ask about the %s tournament.}}", g.mrSk().Union), func() { g.mrMarshalMenu(a, "\"The rolls are open to any fighter with nerve.\"") }}}
	}
	return nil
}

func (g *Game) mrMenuBase(text string) func(string, func()) {
	g.Popup = Wrap(text, 100)
	g.Opts = nil
	return func(t string, f func()) { g.Opts = append(g.Opts, Option{t, f}) }
}

func (g *Game) mrJianghu(a *Actor) string {
	rs := g.mrRumors()
	ss := g.Sects()
	st := g.mr()
	r := NewRng(fmt.Sprintf("%s:murim:learn:%d", g.W.Seed, DayOf(g.Turn)))
	for i := 0; i < 2 && len(ss) > 0; i++ {
		s := ss[r.Int(0, len(ss)-1)]
		if s.Align != mrDemonic || r.Chance(0.3) {
			st.Seen[s.ID] = true
		}
	}
	if len(rs) == 0 {
		return "\"The martial world is quiet. Too quiet.\""
	}
	var parts []string
	for _, x := range rs {
		parts = append(parts, Plain(x))
	}
	return "\"" + strings.Join(parts, " ") + "\" (Press G to review the sects you know.)"
}

func (g *Game) mrShop(a *Actor, s *MrSect, mult float64) {
	add := g.mrMenuBase("\"Pills to open the meridians. Do not gulp them.\"")
	for i := 0; i < 3; i++ {
		i := i
		price := int(float64([]int{30, 90, 250}[i]) * mult)
		add(fmt.Sprintf("%s ({{Y|%d %s}})", g.mrSk().Pills[i], price, g.Sk.Currency), func() {
			if g.P.Gold < price {
				g.mrHallMenu(a, s, "\"Come back when your purse is heavier.\"")
				return
			}
			g.P.Gold -= price
			g.give(g.mrPill(i))
			g.mrHallMenu(a, s, "\"Use it wisely.\"")
		})
	}
	add("Back.", func() { g.mrHallMenu(a, s, "\""+s.Creed+"\"") })
	g.Sel = 0
}

func (g *Game) mrTrialChance(s *MrSect) float64 {
	st := g.mr()
	c := 0.45 + 0.06*float64(g.P.M(s.Art.Stat)) + 0.02*float64(g.P.Level) + 0.05*float64(st.Realm)
	if s.Kind == "clan" {
		c -= 0.1
	}
	if s.Align == mrUnorthodox {
		c += 0.1
	}
	return float64(clampi(int(c*100), 10, 95)) / 100
}

func (g *Game) mrHallMenu(a *Actor, s *MrSect, text string) {
	add := g.mrMenuBase(text)
	st := g.mr()
	sk := g.mrSk()
	mine := g.mrMine()
	back := func() { g.mrHallMenu(a, s, "\""+s.Creed+"\"") }
	switch {
	case mine == nil:
		add(fmt.Sprintf("{{G|Seek admission.}} {{K|(trial of %s)}}", AttrNames[s.Art.Stat]), func() {
			if st.Lock[s.ID] > g.Turn {
				g.mrHallMenu(a, s, "\"You failed us not long ago. Return later.\"")
				return
			}
			if mrRoll(g) < g.mrTrialChance(s) {
				g.mrJoin(s)
				g.mrHallMenu(a, s, fmt.Sprintf("\"%s Welcome, %s.\"", sk.Trial[s.Art.Stat], sk.Ranks[0]))
				return
			}
			st.Lock[s.ID] = g.Turn + DayLen/2
			g.mrHallMenu(a, s, "\""+sk.Trial[s.Art.Stat]+" ... You fall short. Train, and come back tomorrow.\"")
		})
		add("{{R|Challenge a senior disciple to a duel.}}", func() {
			if st.Lock[s.ID] > g.Turn {
				g.mrHallMenu(a, s, "\"Not so soon after your failure.\"")
				return
			}
			d := g.mrDisciple(s, g.P.Level+1, g.R, 1)
			d.Faction = "mrduel"
			d.Angry, d.AI = true, "hostile"
			d.Set("mrduel:" + s.ID)
			if g.mrSpawnNear(g.Cur, d, 3, 10) {
				g.Mode, g.Talk = "main", nil
				g.Msg("{{R|A senior disciple steps forward to test you.}}")
			} else {
				back()
			}
		})
		add("What does the sect teach?", func() {
			g.mrHallMenu(a, s, fmt.Sprintf("\"The %s: %s, %s and %s. Leader: %s. Our rivals: %s.\"", s.Art.Name, s.Art.Tech[0].Name, s.Art.Tech[1].Name, s.Art.Tech[2].Name, s.Leader, g.mrRivalNames(s)))
		})
		add("Buy cultivation pills.", func() { g.mrShop(a, s, 1.5) })
	case mine.ID == s.ID:
		m := st.Mission
		switch {
		case m == nil:
			add("{{Y|Ask for a sect task.}}", func() {
				st.Mission = g.mrNewMission(s)
				if st.Mission.Kind == "courier" {
					g.give(&Item{ID: g.newID(), Kind: "letter", Name: "sealed letter of " + s.Short(), Glyph: '=', F: CWhite, Text: fmt.Sprintf("mr-letter:%s:%d", s.ID, st.Mission.Dest),
						Weight: 1, Ident: true})
				}
				g.mrHallMenu(a, s, "\""+st.Mission.Text+"\"")
			})
		case m.Ready && m.Kind != "courier":
			add("{{G|I have done what you asked.}}", func() { g.mrHallMenu(a, s, "\""+g.mrFinish(s)+"\"") })
			fallthrough
		default:
			if m != nil {
				add("What is my task?", func() { g.mrHallMenu(a, s, fmt.Sprintf("\"%s\" (%d/%d)", m.Text, m.Have, m.Need)) })
				add("{{K|Abandon this task.}}", func() { st.Mission = nil; g.Rep[s.RepKey()] -= 10; back() })
			}
		}
		if m != nil && m.Kind == "courier" && !m.Ready && g.Cur.Burg == m.Dest {
			add("{{G|Deliver the sealed message.}}", func() {
				for i, it := range g.P.Inv {
					if it.Kind == "letter" && strings.HasPrefix(it.Text, "mr-letter:"+s.ID+":") {
						g.P.Inv = append(g.P.Inv[:i], g.P.Inv[i+1:]...)
						m.Have, m.Ready = 1, true
						g.mrHallMenu(a, s, "\""+g.mrFinish(s)+"\"")
						return
					}
				}
				g.mrHallMenu(a, s, "\"You carry no message of ours.\"")
			})
		}
		add("{{O|Petition for promotion.}}", func() {
			ok, msg := g.mrPromote(s)
			if ok {
				g.mrHallMenu(a, s, "\""+msg+"\"")
			} else {
				g.mrHallMenu(a, s, "\"Not yet. "+msg+"\"")
			}
		})
		mult := 1.0
		if st.Rank >= 2 {
			mult = 0.7
		}
		add("Buy cultivation pills.", func() { g.mrShop(a, s, mult) })
		add("{{R|Renounce the sect.}}", func() { g.mrLeave(s); g.mrHallMenu(a, s, "\"Then you are nothing to us.\"") })
	default:
		add(fmt.Sprintf("{{K|(You wear the colours of %s.)}}", mine.Short()), func() {
			g.mrHallMenu(a, s, "\"We know whose disciple you are. Leave, or prove yourself a friend.\"")
		})
		add("Buy cultivation pills.", func() { g.mrShop(a, s, 2) })
	}
	add("Ask about the jianghu.", func() { g.mrHallMenu(a, s, g.mrJianghu(a)) })
	add("Enough about that.", func() { g.talkMenu(g.greeting(a)) })
	g.Sel = 0
}

func (g *Game) mrRivalNames(s *MrSect) string {
	var n []string
	for _, id := range s.Rivals {
		if r := g.SectByID(id); r != nil {
			n = append(n, r.Short())
		}
	}
	if len(n) == 0 {
		return "none"
	}
	return strings.Join(n, ", ")
}

func (g *Game) mrLeadMenu(a *Actor, s *MrSect, text string) {
	add := g.mrMenuBase(text)
	st := g.mr()
	back := func() { g.mrLeadMenu(a, s, "\"Speak.\"") }
	if st.Sect == s.ID {
		add("Ask for guidance.", func() {
			if DayOf(g.Turn)-st.Coach[a.Name] < 7 && st.Coach[a.Name] != 0 {
				g.mrLeadMenu(a, s, "\"I taught you this week. Practise.\"")
				return
			}
			st.Coach[a.Name] = DayOf(g.Turn)
			g.mrAddProg(25 + 10*st.Realm)
			g.mrLeadMenu(a, s, "\"Breathe from the belly. Again.\" (+cultivation)")
		})
		if st.Rank == 3 {
			add("{{R|Challenge you for the seat.}}", func() {
				if st.Contrib < mrRankCon[4] || st.Realm < mrRankRealm[4] {
					g.mrLeadMenu(a, s, fmt.Sprintf("\"You are not ready. %d contribution and the %s realm.\"", mrRankCon[4], g.mrRealmName(mrRankRealm[4])))
					return
				}
				a.Angry, a.AI = true, "hostile"
				a.Faction = "mrchal"
				g.Mode, g.Talk = "main", nil
				g.Msg("{{R|The leader draws a breath. \"Then come.\"}}")
			})
		}
	}
	add("Who are you?", func() {
		g.mrLeadMenu(a, s, fmt.Sprintf("\"%s, %s of %s. We keep the %s.\"", s.Leader, strings.ToLower(s.LeaderTit), s.Name, s.Art.Name))
	})
	add("Enough about that.", func() { g.talkMenu(g.greeting(a)) })
	_ = back
	g.Sel = 0
}

func (g *Game) mrElderMenu(a *Actor, s *MrSect, text string) {
	add := g.mrMenuBase(text)
	st := g.mr()
	if st.Sect == s.ID {
		add("Receive instruction.", func() {
			if st.Coach[a.Name] != 0 && DayOf(g.Turn)-st.Coach[a.Name] < 3 {
				g.mrElderMenu(a, s, "\"Rest. The lesson needs time to settle.\"")
				return
			}
			st.Coach[a.Name] = DayOf(g.Turn)
			g.mrAddProg(15 + 6*st.Realm)
			g.mrElderMenu(a, s, "\"Again, with the hips.\" (+cultivation)")
		})
	} else {
		for i := range s.Art.Tech {
			i := i
			t := s.Art.Tech[i]
			price := 150 * (i + 1) * (i + 1)
			if mrHas(st.Techs, t.ID) {
				continue
			}
			add(fmt.Sprintf("Learn %s. {{Y|(%d %s)}}", t.Name, price, g.Sk.Currency), func() {
				if g.P.Gold < price {
					g.mrElderMenu(a, s, "\"Knowledge is not free.\"")
					return
				}
				g.P.Gold -= price
				g.mrLearn(t.ID)
				g.mrElderMenu(a, s, "\"Do not teach it to the wrong people.\"")
			})
		}
	}
	add("Enough about that.", func() { g.talkMenu(g.greeting(a)) })
	g.Sel = 0
}

func (g *Game) mrMarshalMenu(a *Actor, text string) {
	add := g.mrMenuBase(text)
	st := g.mr()
	day := DayOf(g.Turn)
	ed, act, to := mrTournament(day)
	if st.Tourn.Ed != ed {
		st.Tourn = mrTourn{Ed: ed}
	}
	switch {
	case !act:
		add(fmt.Sprintf("{{K|(The next tournament opens in %d days.)}}", to), func() { g.mrMarshalMenu(a, "\"Come back then.\"") })
	case st.Tourn.Done:
		add("{{K|(You have already won this edition.)}}", func() { g.mrMarshalMenu(a, "\"Champion. Rest.\"") })
	case st.Tourn.Entered:
		add("{{K|(Your bouts are under way.)}}", func() { g.mrMarshalMenu(a, "\"Finish what you started.\"") })
	default:
		add("{{R|Enter the tournament (3 bouts).}}", func() {
			st.Tourn = mrTourn{Ed: ed, Entered: true}
			g.Mode, g.Talk = "main", nil
			g.mrBoutFoe(0)
		})
	}
	add("Ask about the jianghu.", func() { g.mrMarshalMenu(a, g.mrJianghu(a)) })
	add("Enough about that.", func() { g.talkMenu(g.greeting(a)) })
	g.Sel = 0
}

// ---- screen ----

func (g *Game) mrKnown() []*MrSect {
	st := g.mr()
	var out []*MrSect
	for _, s := range g.Sects() {
		if st.Seen[s.ID] || st.Sect == s.ID {
			out = append(out, s)
		}
	}
	return out
}

func (g *Game) mrPackItems() []*Item {
	var out []*Item
	for _, it := range g.P.Inv {
		if it.Kind == "mrpill" || it.Kind == "mrmanual" {
			out = append(out, it)
		}
	}
	return out
}

func mrKeys(g *Game, k string) {
	switch k {
	case "Escape", "G", "q":
		g.Mode = "main"
	case "Tab", "ArrowRight":
		g.Tab, g.Sel = (g.Tab+1)%4, 0
	case "ArrowLeft":
		g.Tab, g.Sel = (g.Tab+3)%4, 0
	case "ArrowUp", "N8", "k":
		g.Sel = maxi(0, g.Sel-1)
	case "ArrowDown", "N2", "j":
		g.Sel++
	case "z":
		g.mrMeditate(30)
	case "b":
		g.mrBreakthrough()
	case "Enter", "e":
		if g.Tab == 2 {
			if its := g.mrPackItems(); g.Sel >= 0 && g.Sel < len(its) {
				g.mrUseItem(its[g.Sel])
			}
		}
	}
}

func (g *Game) mrDraw() {
	s := g.sc
	x, y, w, h := g.panel("Murim")
	sk := g.mrSk()
	st := g.mr()
	tabs := []string{"Sects", "Cultivation", "Pack", "Rumors"}
	tx := x
	for i, t := range tabs {
		c := "K"
		if g.Tab == i {
			c = "Y"
		}
		tx = s.TextB(tx, y, fmt.Sprintf("{{%s|[ %s ]}} ", c, t), CGray, CBgDark, x+w)
	}
	y += 2
	line := func(yy *int, lx, lw int, str string) {
		for _, l := range Wrap(str, lw) {
			if *yy < y+h-5 {
				s.TextB(lx, *yy, l, CGray, CBgDark, lx+lw)
				*yy++
			}
		}
	}
	switch g.Tab {
	case 0:
		known := g.mrKnown()
		if len(known) == 0 {
			s.TextB(x, y, "{{K|You know of no sects yet. Look for banners in towns and ask about the jianghu.}}", CGray, CBgDark, x+w)
			return
		}
		g.Sel = clampi(g.Sel, 0, len(known)-1)
		lw := 40
		for i, sc := range known {
			if i >= h-6 {
				break
			}
			bg := CBgDark
			if i == g.Sel {
				bg = CBgSel
			}
			s.Fill(x, y+i, lw, 1, bg)
			mark := " "
			if st.Sect == sc.ID {
				mark = "{{Y|*}}"
			}
			s.TextB(x, y+i, fmt.Sprintf("%s %s", mark, shorten(sc.Name, lw-3)), CGray, bg, x+lw)
		}
		sc := known[g.Sel]
		dx, dw := x+lw+2, w-lw-2
		ly := y
		col := map[string]string{mrOrthodox: "C", mrUnorthodox: "Y", mrDemonic: "R"}[sc.Align]
		line(&ly, dx, dw, fmt.Sprintf("{{W|%s}} {{%s|(%s %s)}}", sc.Name, col, sc.Align, sc.Kind))
		line(&ly, dx, dw, "{{c|"+sc.Creed+"}}")
		line(&ly, dx, dw, fmt.Sprintf("Seat: {{y|%s}}, %s", sc.HQName, g.mrCompass(sc.WX, sc.WY)))
		var hs []string
		for _, hl := range sc.Halls {
			hs = append(hs, fmt.Sprintf("%s (%s)", hl.Town, g.W.StateName(hl.State)))
		}
		line(&ly, dx, dw, "Halls: "+strings.Join(hs, ", "))
		line(&ly, dx, dw, fmt.Sprintf("Art: {{M|%s}} (favours %s)", sc.Art.Name, AttrNames[sc.Art.Stat]))
		for _, t := range sc.Art.Tech {
			own := " "
			if mrHas(st.Techs, t.ID) {
				own = "{{G|+}}"
			}
			line(&ly, dx, dw, fmt.Sprintf(" %s %s {{K|[%s, %d %s, cd %d]}}", own, t.Name, g.mrRealmName(t.Need), t.Cost, sk.Qi, t.Cool))
		}
		line(&ly, dx, dw, fmt.Sprintf("%s: %s. Elders: %s.", sc.LeaderTit, sc.Leader, strings.Join(sc.Elders, ", ")))
		line(&ly, dx, dw, "Rivals: {{r|"+g.mrRivalNames(sc)+"}}")
		if st.Sect == sc.ID {
			line(&ly, dx, dw, fmt.Sprintf("{{G|You are %s. Contribution %d.}}", sk.Ranks[st.Rank], st.Contrib))
		}
	case 1:
		ly := y
		line(&ly, x, w, fmt.Sprintf("Realm: {{W|%s}} (%d/7)   %s: %d/%d   Deviations: %d", g.mrRealmName(st.Realm), st.Realm, strings.Title(sk.Qi), g.mrQi(), g.mrMaxQi(), st.Dev))
		if st.Realm < 7 {
			need := mrBreakNeed[st.Realm]
			bar := strings.Repeat("#", clampi(st.Prog*30/need, 0, 30)) + strings.Repeat(".", 30-clampi(st.Prog*30/need, 0, 30))
			line(&ly, x, w, fmt.Sprintf("Cultivation [%s] %d/%d   breakthrough chance %d%%", bar, st.Prog, need, int(g.mrBreakChance()*100)))
			line(&ly, x, w, fmt.Sprintf("Next: %s", g.mrRealmName(st.Realm+1)))
		}
		wnt := mrWant(st.Realm, g.mrFav())
		line(&ly, x, w, fmt.Sprintf("Realm bonus: +%d HP, +%d armour, favoured %s", wnt.HP, wnt.AV, AttrNames[g.mrFav()]))
		ly++
		if sc := g.mrMine(); sc != nil {
			line(&ly, x, w, fmt.Sprintf("Sect: {{C|%s}} - %s, contribution %d", sc.Name, sk.Ranks[st.Rank], st.Contrib))
			if st.Rank < 3 {
				line(&ly, x, w, fmt.Sprintf("Next rank: %d contribution, %s realm", mrRankCon[st.Rank+1], g.mrRealmName(mrRankRealm[st.Rank+1])))
			}
			if m := st.Mission; m != nil {
				line(&ly, x, w, fmt.Sprintf("{{y|Task:}} %s (%d/%d)", m.Text, m.Have, m.Need))
			}
		} else {
			line(&ly, x, w, "{{K|You follow no sect.}}")
		}
		ly++
		line(&ly, x, w, "Techniques (use from the abilities screen):")
		for _, id := range st.Techs {
			if _, t := g.mrTech(id); t != nil {
				line(&ly, x, w, fmt.Sprintf("  %s [%s, %d %s, cd %d]", t.Name, g.mrRealmName(t.Need), t.Cost, sk.Qi, t.Cool))
			}
		}
		s.TextB(x, y+h-4, "{{Y|z}} meditate (30 turns, twice as good in a headquarters)   {{Y|b}} attempt breakthrough   {{Y|Tab}} next tab", CGray, CBgDark, x+w)
	case 2:
		its := g.mrPackItems()
		if len(its) == 0 {
			s.TextB(x, y, "{{K|You carry no pills or manuals.}}", CGray, CBgDark, x+w)
			return
		}
		g.Sel = clampi(g.Sel, 0, len(its)-1)
		for i, it := range its {
			bg := CBgDark
			if i == g.Sel {
				bg = CBgSel
			}
			s.Fill(x, y+i, w, 1, bg)
			s.TextB(x, y+i, fmt.Sprintf("%s  {{K|%s}}", it.Name, it.Lore), CGray, bg, x+w)
		}
		s.TextB(x, y+h-4, "{{Y|Enter}} use", CGray, CBgDark, x+w)
	case 3:
		ly := y
		rs := g.mrRumors()
		if len(rs) == 0 {
			rs = []string{"{{K|Nothing is stirring in the martial world.}}"}
		}
		for _, r := range rs {
			line(&ly, x, w, r)
		}
	}
}

// keep sort imported for future use in screens
var _ = sort.Strings
