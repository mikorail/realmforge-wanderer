package engine

import (
	"fmt"
	"sort"
	"strings"
)

// Disguises, Forged Identities, Blackmail Leverage & Apprentices (prefix dg).
// Saved state lives in g.ext("dg"), planet-local ("dg" in unLocalKeys).
// Side RNGs seeded from g.Seed+":dg:..." are used for generation; g.R only for play-time rolls.

func init() {
	unLocalKeys = append(unLocalKeys, "dg")
	turnHooks = append(turnHooks, dgTurn)
	killHooks = append(killHooks, dgKill)
	zoneHooks = append(zoneHooks, dgZone)
	enterHooks = append(enterHooks, dgEnter)
	talkHooks = append(talkHooks, dgTalk)
	sideHooks = append(sideHooks, dgSide)
	modeKeys["dg"] = dgKeys
	modeDraws["dg"] = dgDraw
	paEntries = append(paEntries, paEntry{
		Name:     "Disguises, leverage & apprentices",
		Keywords: "disguise alias cover blackmail leverage scandal apprentice mentor",
		Cat:      "People",
		Func:     func(g *Game) { g.dgOpen() },
	})
}

type dgDisguiseDef struct {
	ID       string
	Name     string
	Role     string
	Cost     int
	MaxCover int
	Desc     string
}

var dgDisguises = []dgDisguiseDef{
	{
		ID:       "watch_tabard",
		Name:     "Watch Tabard",
		Role:     "guard",
		Cost:     60,
		MaxCover: 100,
		Desc:     "A stitched garrison surcoat and brass badge that let you pass checkpoints.",
	},
	{
		ID:       "priest_robes",
		Name:     "Priest Robes",
		Role:     "priest",
		Cost:     50,
		MaxCover: 95,
		Desc:     "Plain temple vestments and a prayer cord that deter rough questioning.",
	},
	{
		ID:       "merchant_papers",
		Name:     "Merchant Papers",
		Role:     "merchant",
		Cost:     75,
		MaxCover: 100,
		Desc:     "Stamped customs writs, wax seals, and a factor's coat for busy gates.",
	},
	{
		ID:       "guild_hood",
		Name:     "Guild Hood",
		Role:     "artisan",
		Cost:     55,
		MaxCover: 90,
		Desc:     "A dyed craft hood and tool sash worn by traveling journeymen.",
	},
}

var dgDisguiseByID = func() map[string]*dgDisguiseDef {
	m := map[string]*dgDisguiseDef{}
	for i := range dgDisguises {
		m[dgDisguises[i].ID] = &dgDisguises[i]
	}
	return m
}()

// dgAlias represents the player's active forged identity and cover integrity.
type dgAlias struct {
	Name           string `json:"n,omitempty"`
	Role           string `json:"r,omitempty"`
	Origin         string `json:"o,omitempty"`
	DisguiseID     string `json:"d,omitempty"`
	Cover          int    `json:"c,omitempty"`
	MaxCover       int    `json:"mc,omitempty"`
	Active         bool   `json:"a,omitempty"`
	Suspicion      int    `json:"s,omitempty"`
	BountiesEvaded int    `json:"be,omitempty"`
	CreatedDay     int    `json:"cd,omitempty"`
}

// dgLeverage holds active blackmail converted from a known secret.
type dgLeverage struct {
	ID          string `json:"id"`
	TargetName  string `json:"tn"`
	TargetID    int    `json:"tid,omitempty"`
	BurgID      int    `json:"bid,omitempty"`
	StateID     int    `json:"sid,omitempty"`
	SecretText  string `json:"st"`
	Strength    int    `json:"str"`
	Source      string `json:"src"` // "kn" or "secret"
	Used        bool   `json:"u,omitempty"`
	Leaked      bool   `json:"lk,omitempty"`
	GatheredDay int    `json:"gd,omitempty"`
}

// dgApprentice tracks a recruited or graduated apprentice.
type dgApprentice struct {
	ID          string         `json:"id"`
	ActorID     int            `json:"aid,omitempty"`
	Name        string         `json:"n"`
	HomeBurg    int            `json:"hb,omitempty"`
	HomeName    string         `json:"hn,omitempty"`
	Level       int            `json:"lv"`
	XP          int            `json:"xp"`
	Skills      map[string]int `json:"sk,omitempty"`
	Lessons     int            `json:"ls,omitempty"`
	Specialty   string         `json:"sp,omitempty"`
	Graduated   bool           `json:"gr,omitempty"`
	GradDay     int            `json:"gd,omitempty"`
	Encountered bool           `json:"en,omitempty"`
}

type dgState struct {
	Owned      map[string]bool `json:"ow,omitempty"`
	Alias      dgAlias         `json:"al,omitempty"`
	Leverage   []*dgLeverage   `json:"lv,omitempty"`
	UsedSec    map[string]bool `json:"us,omitempty"`
	Discounts  map[int]int     `json:"dc,omitempty"` // burgID -> percent discount
	Scandals   map[string]int  `json:"sc,omitempty"` // targetName -> scandal count
	CrierLog   []string        `json:"cl,omitempty"`
	Apprentice *dgApprentice   `json:"ap,omitempty"`
	Graduates  []*dgApprentice `json:"gr,omitempty"`
	GradZones  map[string]bool `json:"gz,omitempty"`
	NextSeq    int             `json:"sq,omitempty"`

	tab int
	sel int
}

func dgSt(g *Game) *dgState {
	st := g.ext("dg", func() any { return &dgState{} }).(*dgState)
	if st.Owned == nil {
		st.Owned = map[string]bool{}
	}
	if st.UsedSec == nil {
		st.UsedSec = map[string]bool{}
	}
	if st.Discounts == nil {
		st.Discounts = map[int]int{}
	}
	if st.Scandals == nil {
		st.Scandals = map[string]int{}
	}
	if st.GradZones == nil {
		st.GradZones = map[string]bool{}
	}
	if st.Apprentice != nil && st.Apprentice.Skills == nil {
		st.Apprentice.Skills = map[string]int{}
	}
	for _, gr := range st.Graduates {
		if gr != nil && gr.Skills == nil {
			gr.Skills = map[string]int{}
		}
	}
	return st
}

// ---- Disguises, Forged Identities & Double Lives ----

func (g *Game) dgGenAliasName(disguiseID string) (string, string) {
	r := NewRng(fmt.Sprintf("%s:dg:alias:%s:%d", g.Seed, disguiseID, g.Day()))
	firsts := []string{"Maren", "Tovin", "Bram", "Sera", "Oren", "Kael", "Nira", "Davan", "Liora", "Halen"}
	lasts := []string{"Vane", "Marsh", "Redfern", "Carrow", "Thorne", "Linden", "Wick", "Sable", "Alder", "Fenn"}
	origins := []string{"the river towns", "the salt road", "the border marches", "the high market", "the old abbey"}
	return Pick(r, firsts) + " " + Pick(r, lasts), Pick(r, origins)
}

// dgAcquireDisguise purchases or acquires a disguise kit by ID.
func (g *Game) dgAcquireDisguise(id string) (bool, string) {
	def := dgDisguiseByID[id]
	if def == nil {
		return false, "Unknown disguise kit."
	}
	st := dgSt(g)
	if st.Owned[id] {
		return true, "You already own " + def.Name + "."
	}
	if g.P.Gold < def.Cost {
		msg := fmt.Sprintf("You need %d %s for %s.", def.Cost, g.Sk.Currency, def.Name)
		g.Msg(msg)
		return false, msg
	}
	g.P.Gold -= def.Cost
	st.Owned[id] = true
	msg := fmt.Sprintf("You acquire %s for %d %s.", def.Name, def.Cost, g.Sk.Currency)
	g.Msg("{{G|" + msg + "}}")
	return true, msg
}

// dgDonDisguise puts on a faction/role disguise and establishes an active alias identity.
func (g *Game) dgDonDisguise(id, aliasName string) (bool, string) {
	def := dgDisguiseByID[id]
	if def == nil {
		return false, "Unknown disguise kit."
	}
	st := dgSt(g)
	st.Owned[id] = true
	genName, origin := g.dgGenAliasName(id)
	if strings.TrimSpace(aliasName) == "" {
		aliasName = genName
	}
	st.Alias = dgAlias{
		Name:           aliasName,
		Role:           def.Role,
		Origin:         origin,
		DisguiseID:     def.ID,
		Cover:          def.MaxCover,
		MaxCover:       def.MaxCover,
		Active:         true,
		Suspicion:      0,
		BountiesEvaded: st.Alias.BountiesEvaded,
		CreatedDay:     g.Day(),
	}
	msg := fmt.Sprintf("You don %s and assume the alias %s (%s from %s). Cover: %d%%.", def.Name, aliasName, def.Role, origin, def.MaxCover)
	g.Msg("{{C|" + msg + "}}")
	return true, msg
}

// dgDropDisguise sheds the active disguise cleanly.
func (g *Game) dgDropDisguise() {
	st := dgSt(g)
	if !st.Alias.Active {
		return
	}
	st.Alias.Active = false
	st.Alias.Cover = 0
	g.Msg("You shed your disguise and resume your own name.")
}

// dgHasCover reports whether the player currently maintains an intact disguise cover.
func (g *Game) dgHasCover() bool {
	st := dgSt(g)
	return st.Alias.Active && st.Alias.Cover > 0
}

// dgDegradeCover reduces cover integrity on suspicious acts and blows cover at 0.
func (g *Game) dgDegradeCover(amount int, reason string) bool {
	st := dgSt(g)
	if !st.Alias.Active || amount <= 0 {
		return false
	}
	st.Alias.Cover = maxi(0, st.Alias.Cover-amount)
	st.Alias.Suspicion += amount
	if st.Alias.Cover == 0 {
		st.Alias.Active = false
		if g.Cur != nil {
			for _, a := range g.Cur.Actors {
				if a != nil && a.HP > 0 && a.Role == "guard" && abs(a.X-g.P.X)+abs(a.Y-g.P.Y) <= 10 {
					a.Angry = true
				}
			}
		}
		if reason != "" {
			g.Msg("{{R|Your cover is blown (" + reason + ")! Nearby guards recognize you.}}")
		} else {
			g.Msg("{{R|Your cover is blown! Nearby guards recognize you.}}")
		}
		return true
	}
	if reason != "" {
		g.Msg(fmt.Sprintf("{{O|Suspicious act (%s): alias cover drops to %d%%.}}", reason, st.Alias.Cover))
	}
	return false
}

// dgSlipPastGuards checks if an active disguise lets a fugitive slip past hostile guards/bounties.
func (g *Game) dgSlipPastGuards() bool {
	st := dgSt(g)
	if !st.Alias.Active || st.Alias.Cover <= 0 {
		return false
	}
	calmed := false
	if g.Cur != nil {
		for _, a := range g.Cur.Actors {
			if a == nil || a.HP <= 0 {
				continue
			}
			if a.Role == "guard" || a.AI == "guard" {
				dist := abs(a.X-g.P.X) + abs(a.Y-g.P.Y)
				if dist <= 8 && (a.Angry || g.Hostile(a)) {
					a.Angry = false
					calmed = true
				}
			}
		}
	}
	// Check if player has an active bounty in the local state.
	hasBounty := false
	if bd := g.localBurg(); bd != nil && g.uwSt().Bounty[bd.State] > 0 {
		hasBounty = true
	}
	if calmed || hasBounty {
		st.Alias.BountiesEvaded++
		if st.Alias.Cover > 5 {
			st.Alias.Cover = maxi(1, st.Alias.Cover-3)
		}
		return true
	}
	return true
}

// ---- Blackmail, Leverage & Court Scandals ----

func (g *Game) dgPickSecret(st *dgState) (string, string, bool) {
	// First check knowledge.go facts (learned facts or recorded secrets/crimes).
	kn := knSt(g)
	for _, lr := range kn.Learned {
		f := kn.Facts[lr.Fact]
		if f == nil || f.Text == "" {
			continue
		}
		key := fmt.Sprintf("kn:%d", f.ID)
		if st.UsedSec[key] {
			continue
		}
		if f.Kind == "secret" || f.Kind == "crime" || f.Kind == "rumour" {
			st.UsedSec[key] = true
			return f.Text, "kn", true
		}
	}
	if len(kn.Facts) > 0 {
		ids := make([]int, 0, len(kn.Facts))
		for id := range kn.Facts {
			ids = append(ids, id)
		}
		sort.Ints(ids)
		for _, id := range ids {
			f := kn.Facts[id]
			if f == nil || f.Text == "" {
				continue
			}
			key := fmt.Sprintf("kn:%d", f.ID)
			if st.UsedSec[key] {
				continue
			}
			if f.Kind == "secret" || f.Kind == "crime" {
				st.UsedSec[key] = true
				return f.Text, "kn", true
			}
		}
	}
	// Next check g.Secrets.
	for i, sec := range g.Secrets {
		if sec.Text == "" {
			continue
		}
		key := fmt.Sprintf("sec:%d:%s", i, sec.Text)
		if st.UsedSec[key] {
			continue
		}
		st.UsedSec[key] = true
		return sec.Text, "secret", true
	}
	return "", "", false
}

// dgGatherLeverage converts a secret from knowledge.go (kn) or g.Secrets into active leverage.
func (g *Game) dgGatherLeverage(target *Actor) (*dgLeverage, bool) {
	st := dgSt(g)
	secText, src, ok := g.dgPickSecret(st)
	if !ok {
		g.Msg("You hold no unused secrets to turn into leverage.")
		return nil, false
	}
	targetName := "Local Notable"
	targetID := 0
	stateID := g.HomeState
	burgID := g.StartBurg
	if bd := g.localBurg(); bd != nil {
		burgID = bd.ID
		stateID = bd.State
		if bd.Governor != "" {
			targetName = bd.Governor
		}
	}
	if target != nil {
		targetName = target.Name
		targetID = target.ID
		if target.State > 0 {
			stateID = target.State
		}
	} else if g.Cur != nil {
		for _, a := range g.Cur.Actors {
			if a != nil && a.HP > 0 && (a.Role == "merchant" || a.Role == "mayor" || a.Role == "priest" || a.Proper) {
				targetName = a.Name
				targetID = a.ID
				if a.State > 0 {
					stateID = a.State
				}
				break
			}
		}
	}
	st.NextSeq++
	strength := 2
	if src == "kn" {
		strength = 3
	}
	lev := &dgLeverage{
		ID:          fmt.Sprintf("lev_%d", st.NextSeq),
		TargetName:  targetName,
		TargetID:    targetID,
		BurgID:      burgID,
		StateID:     stateID,
		SecretText:  secText,
		Strength:    strength,
		Source:      src,
		GatheredDay: g.Day(),
	}
	st.Leverage = append(st.Leverage, lev)
	g.Msg(fmt.Sprintf("{{Y|You compile leverage against %s from a secret (%s).}}", targetName, src))
	return lev, true
}

func (g *Game) dgFindLeverage(id string) *dgLeverage {
	st := dgSt(g)
	for _, l := range st.Leverage {
		if l.ID == id {
			return l
		}
	}
	return nil
}

// dgExtort uses active leverage to demand gold and a local merchant discount.
func (g *Game) dgExtort(levID string) (int, bool) {
	st := dgSt(g)
	lev := g.dgFindLeverage(levID)
	if lev == nil || lev.Used || lev.Leaked {
		return 0, false
	}
	payout := 35*lev.Strength + g.R.Int(5, 25)
	g.P.Gold += payout
	disc := 10 + 5*lev.Strength
	if st.Discounts[lev.BurgID] < disc {
		st.Discounts[lev.BurgID] = disc
	}
	lev.Used = true
	if st.Alias.Active {
		g.dgDegradeCover(12, "extortion")
	}
	g.Msg(fmt.Sprintf("{{G|%s pays %d %s in hush money and grants a %d%% local discount.}}", lev.TargetName, payout, g.Sk.Currency, disc))
	return payout, true
}

// dgSilenceWitness spends leverage to silence witnesses and reduce local state bounty.
func (g *Game) dgSilenceWitness(levID string) (bool, string) {
	st := dgSt(g)
	lev := g.dgFindLeverage(levID)
	if lev == nil || lev.Used || lev.Leaked {
		return false, "No active leverage selected."
	}
	lev.Used = true
	reduction := 80 * lev.Strength
	uw := g.uwSt()
	if cur := uw.Bounty[lev.StateID]; cur > 0 {
		uw.Bounty[lev.StateID] = maxi(0, cur-reduction)
	}
	if g.Cur != nil {
		for _, a := range g.Cur.Actors {
			if a != nil && a.HP > 0 && a.Angry && (a.AI == "town" || a.Role == "guard") {
				a.Angry = false
			}
		}
	}
	if st.Alias.Active {
		g.dgDegradeCover(8, "silencing a witness")
	}
	msg := fmt.Sprintf("You lean on %s to silence witnesses; local bounty drops by up to %d.", lev.TargetName, reduction)
	g.Msg("{{C|" + msg + "}}")
	return true, msg
}

// dgLeakScandal leaks leverage to the town crier, toppling the notable's standing.
func (g *Game) dgLeakScandal(levID string) (bool, string) {
	st := dgSt(g)
	lev := g.dgFindLeverage(levID)
	if lev == nil || lev.Leaked {
		return false, "No un-leaked leverage available."
	}
	lev.Leaked = true
	lev.Used = true
	st.Scandals[lev.TargetName]++
	if lev.StateID > 0 {
		po := g.po()
		po.Stand[lev.StateID] = maxi(-100, po.Stand[lev.StateID]-15*lev.Strength)
	}
	headline := fmt.Sprintf("Town Crier: Scandal topples %s! \"%s\"", lev.TargetName, lev.SecretText)
	st.CrierLog = append(st.CrierLog, headline)
	if len(st.CrierLog) > 12 {
		st.CrierLog = st.CrierLog[len(st.CrierLog)-12:]
	}
	g.deed("event", fmt.Sprintf("%s leaked a scandal to the town crier that toppled %s.", g.P.Name, lev.TargetName), 0, 0)
	g.gainXP(25 * lev.Strength)
	g.Msg("{{M|" + headline + "}}")
	return true, headline
}

// ---- Apprentices & Mentorship ----

func (g *Game) dgApprenticeActor() *Actor {
	st := dgSt(g)
	if st.Apprentice == nil || st.Apprentice.Graduated || g.Cur == nil {
		return nil
	}
	for _, a := range g.Cur.Actors {
		if a != nil && a.HP > 0 && a.ID == st.Apprentice.ActorID {
			return a
		}
	}
	return nil
}

// dgTakeApprentice recruits a young villager as the player's apprentice follower.
func (g *Game) dgTakeApprentice(a *Actor) (*dgApprentice, bool) {
	st := dgSt(g)
	if st.Apprentice != nil && !st.Apprentice.Graduated {
		g.Msg("You already mentor " + st.Apprentice.Name + ".")
		return nil, false
	}
	if a == nil {
		if g.Cur == nil {
			return nil, false
		}
		r := NewRng(fmt.Sprintf("%s:dg:app:%d", g.Seed, g.Turn))
		a = g.Spawn("villager", maxi(1, g.P.Level-1), r)
		if a == nil {
			return nil, false
		}
		names := []string{"Pip", "Tamsin", "Jori", "Bess", "Cael", "Wren", "Renn", "Milo"}
		a.Name = Pick(r, names)
		a.Proper = true
		x, y, ok := g.Cur.FreeSpot(r, func(x, y int) bool { return abs(x-g.P.X)+abs(y-g.P.Y) <= 4 })
		if !ok {
			x, y = clampi(g.P.X+1, 1, MW-2), g.P.Y
		}
		a.X, a.Y = x, y
		g.Cur.Actors = append(g.Cur.Actors, a)
	}
	homeBurg := g.StartBurg
	homeName := "the village"
	if bd := g.localBurg(); bd != nil {
		homeBurg = bd.ID
		homeName = bd.Name
	}
	a.AI = "follower"
	a.Faction = "player"
	a.Role = "apprentice"
	a.Title = "Apprentice"
	a.Angry = false
	a.Comp = &Companion{
		Faction: "villagers",
		AI:      "town",
		HomeKey: "",
		Town:    homeName,
		Loyalty: 85,
		Since:   g.Turn,
	}
	if g.Cur != nil {
		a.Comp.HomeKey = g.Cur.Key
	}
	st.NextSeq++
	app := &dgApprentice{
		ID:       fmt.Sprintf("app_%d", st.NextSeq),
		ActorID:  a.ID,
		Name:     a.Name,
		HomeBurg: homeBurg,
		HomeName: homeName,
		Level:    maxi(1, a.Level),
		XP:       0,
		Skills:   map[string]int{},
	}
	st.Apprentice = app
	g.Msg(fmt.Sprintf("{{G|%s joins you as your apprentice!}}", app.Name))
	g.deed("event", fmt.Sprintf("%s took %s of %s as an apprentice.", g.P.Name, app.Name, homeName), 0, 0)
	return app, true
}

func (g *Game) dgAwardApprenticeXP(xp int) {
	st := dgSt(g)
	app := st.Apprentice
	if app == nil || app.Graduated || xp <= 0 {
		return
	}
	app.XP += xp
	for app.XP >= app.Level*40 {
		app.XP -= app.Level * 40
		app.Level++
		if act := g.dgApprenticeActor(); act != nil {
			act.Level = app.Level
			act.MaxHP += 6
			act.HP = act.MaxHP
			act.AVNat++
		}
		g.Msg(fmt.Sprintf("{{G|Your apprentice %s advances to level %d!}}", app.Name, app.Level))
	}
}

// dgTrainApprentice instructs the active apprentice in one of your skills.
func (g *Game) dgTrainApprentice(skill string) (bool, string) {
	st := dgSt(g)
	app := st.Apprentice
	if app == nil || app.Graduated {
		return false, "You have no active apprentice to train."
	}
	if skill == "" {
		skill = "survival"
		for k := range g.Skills {
			skill = k
			break
		}
	}
	app.Skills[skill]++
	app.Lessons++
	app.Specialty = skill
	g.dgAwardApprenticeXP(25)
	msg := fmt.Sprintf("You drill %s in %s (rank %d, level %d).", app.Name, skill, app.Skills[skill], app.Level)
	g.Msg("{{C|" + msg + "}}")
	return true, msg
}

// dgGraduateApprentice sends a trained apprentice out as an independent wandering adventurer.
func (g *Game) dgGraduateApprentice() (*dgApprentice, bool) {
	st := dgSt(g)
	app := st.Apprentice
	if app == nil || app.Graduated {
		g.Msg("You have no active apprentice ready to graduate.")
		return nil, false
	}
	if app.Level < 2 && app.Lessons < 2 {
		g.Msg(fmt.Sprintf("%s needs more training first (level 2+ or 2 lessons).", app.Name))
		return nil, false
	}
	if act := g.dgApprenticeActor(); act != nil {
		act.AI = "wander"
		act.Faction = "villagers"
		act.Role = "adventurer"
		act.Title = "Wandering Adventurer"
		act.Comp = nil
	}
	app.Graduated = true
	app.GradDay = g.Day()
	st.Graduates = append(st.Graduates, app)
	st.Apprentice = nil
	g.deed("event", fmt.Sprintf("%s graduated %s into an independent wandering adventurer.", g.P.Name, app.Name), 0, 0)
	g.Msg(fmt.Sprintf("{{Y|%s bows in gratitude and sets off as an independent wandering adventurer!}}", app.Name))
	return app, true
}

// ---- Hooks ----

func dgTurn(g *Game) {
	st := dgSt(g)
	if st.Alias.Active && st.Alias.Cover > 0 {
		// Passive guard check if a hostile guard is adjacent.
		if g.Cur != nil {
			for _, a := range g.Cur.Actors {
				if a != nil && a.HP > 0 && (a.Role == "guard" || a.AI == "guard") {
					if abs(a.X-g.P.X)+abs(a.Y-g.P.Y) <= 3 && a.Angry {
						g.dgSlipPastGuards()
						break
					}
				}
			}
		}
	}
	// Apprentice gains steady experience while traveling with you.
	if st.Apprentice != nil && !st.Apprentice.Graduated && g.Turn > 0 && g.Turn%20 == 0 {
		g.dgAwardApprenticeXP(5)
	}
}

func dgKill(g *Game, d, src *Actor) {
	if d == nil {
		return
	}
	st := dgSt(g)
	// Killing in front of witnesses degrades disguise cover.
	if st.Alias.Active && src == g.P && g.Cur != nil && g.Cur.Kind == "town" {
		g.dgDegradeCover(35, "violence in public")
	}
	// Active apprentice learns from combat nearby.
	if st.Apprentice != nil && !st.Apprentice.Graduated && (src == g.P || (src != nil && src.ID == st.Apprentice.ActorID)) {
		g.dgAwardApprenticeXP(15 + d.Level*5)
	}
}

func dgEnter(g *Game, zn *Zone) {
	if zn == nil {
		return
	}
	st := dgSt(g)
	if st.Alias.Active && st.Alias.Cover > 0 && zn.Kind == "town" {
		g.dgSlipPastGuards()
	}
}

func dgZone(g *Game, zn *Zone, _ *Rng) {
	if zn == nil || (zn.Kind != "town" && zn.Kind != "road") {
		return
	}
	st := dgSt(g)
	if len(st.Graduates) == 0 || st.GradZones[zn.Key] {
		return
	}
	r := NewRng(g.Seed + ":dg:grad:" + zn.Key)
	if !r.Chance(0.25) {
		return
	}
	grad := Pick(r, st.Graduates)
	x, y, ok := zn.FreeSpot(r, nil)
	if !ok {
		return
	}
	act := g.Spawn("villager", maxi(2, grad.Level), r)
	if act == nil {
		return
	}
	act.Name = grad.Name
	act.Proper = true
	act.Role = "dg_graduate"
	act.Title = "Wandering Adventurer"
	act.AI = "wander"
	act.Faction = "villagers"
	act.X, act.Y = x, y
	act.Set("dg_grad")
	zn.Actors = append(zn.Actors, act)
	st.GradZones[zn.Key] = true
}

func dgSide(g *Game) []string {
	st := dgSt(g)
	var out []string
	if st.Alias.Active && st.Alias.Cover > 0 {
		out = append(out, fmt.Sprintf("{{C|Alias: %s (%d%%)}}", st.Alias.Name, st.Alias.Cover))
	}
	if st.Apprentice != nil && !st.Apprentice.Graduated {
		out = append(out, fmt.Sprintf("{{G|Apprentice: %s Lv%d}}", st.Apprentice.Name, st.Apprentice.Level))
	}
	return out
}

func dgTalk(g *Game, a *Actor) []Option {
	if a == nil {
		return nil
	}
	st := dgSt(g)
	var out []Option
	if a.Has("dg_grad") || a.Role == "dg_graduate" {
		out = append(out, Option{"{{G|Greet your former apprentice.}}", func() {
			gift := 20 + g.R.Int(5, 25)
			g.P.Gold += gift
			for _, gr := range st.Graduates {
				if gr.Name == a.Name {
					gr.Encountered = true
				}
			}
			g.talkMenu(fmt.Sprintf("\"Mentor! The road taught me much since we parted. Take %d %s for your pack.\"", gift, g.Sk.Currency))
		}})
		return out
	}
	if (a.Role == "villager" || a.Role == "" || a.Tpl == "villager") && a.AI != "follower" && st.Apprentice == nil {
		out = append(out, Option{"{{G|Take them on as your apprentice.}}", func() {
			if _, ok := g.dgTakeApprentice(a); ok {
				g.talkMenu("\"I will carry my weight and learn every lesson you teach.\"")
			}
		}})
	}
	if a.Role == "merchant" || a.Role == "mayor" || a.Role == "priest" || a.Proper {
		out = append(out, Option{"{{O|Compile blackmail leverage on them.}}", func() {
			if lev, ok := g.dgGatherLeverage(a); ok {
				g.talkMenu(fmt.Sprintf("You quietly note leverage on %s: \"%s\"", lev.TargetName, lev.SecretText))
			} else {
				g.talkMenu("You have no unused secrets to hold over them right now.")
			}
		}})
		for _, lev := range st.Leverage {
			lev := lev
			if lev.Used || lev.Leaked || lev.TargetName != a.Name {
				continue
			}
			out = append(out, Option{"{{R|Extort coin and discounts using leverage.}}", func() {
				if gold, ok := g.dgExtort(lev.ID); ok {
					g.talkMenu(fmt.Sprintf("They pale and slide %d %s across the counter.", gold, g.Sk.Currency))
				}
			}})
			out = append(out, Option{"{{M|Leak their scandal to the town crier.}}", func() {
				if _, msg := g.dgLeakScandal(lev.ID); msg != "" {
					g.talkMenu(msg)
				}
			}})
		}
	}
	if a.Role == "merchant" || a.Role == "tailor" || a.Role == "fence" {
		out = append(out, Option{"{{C|Browse disguises & forged papers.}}", func() {
			g.dgOpen()
		}})
	}
	return out
}

// ---- Interactive Mode "dg" Screen ----

func (g *Game) dgOpen() {
	st := dgSt(g)
	st.tab = 0
	st.sel = 0
	g.Mode = "dg"
}

func dgKeys(g *Game, k string) {
	st := dgSt(g)
	switch k {
	case "Escape", "q":
		g.Mode = "main"
		return
	case "Tab", "ArrowRight":
		st.tab = (st.tab + 1) % 3
		st.sel = 0
		return
	case "ArrowLeft":
		st.tab = (st.tab + 2) % 3
		st.sel = 0
		return
	case "ArrowUp", "N8", "k":
		st.sel = maxi(0, st.sel-1)
		return
	case "ArrowDown", "N2", "j":
		st.sel++
		return
	}
	switch st.tab {
	case 0: // Disguise & Alias
		st.sel = clampi(st.sel, 0, len(dgDisguises)-1)
		def := dgDisguises[st.sel]
		switch k {
		case "Enter", "e":
			if !st.Owned[def.ID] {
				if ok, _ := g.dgAcquireDisguise(def.ID); !ok {
					return
				}
			}
			g.dgDonDisguise(def.ID, "")
		case "x":
			g.dgDropDisguise()
		}
	case 1: // Leverage & Scandals
		if k == "c" {
			g.dgGatherLeverage(nil)
			return
		}
		if len(st.Leverage) == 0 {
			return
		}
		st.sel = clampi(st.sel, 0, len(st.Leverage)-1)
		lev := st.Leverage[st.sel]
		switch k {
		case "Enter", "e":
			g.dgExtort(lev.ID)
		case "s":
			g.dgSilenceWitness(lev.ID)
		case "l":
			g.dgLeakScandal(lev.ID)
		}
	case 2: // Apprentice & Graduates
		switch k {
		case "r":
			g.dgTakeApprentice(nil)
		case "t", "Enter":
			g.dgTrainApprentice("")
		case "g":
			g.dgGraduateApprentice()
		}
	}
}

func dgDraw(g *Game) {
	s := g.sc
	st := dgSt(g)
	x, y, w, h := g.panel("Disguises, Leverage & Apprentices")
	bottom := y + h - 1
	tabs := []string{"Disguise & Alias", "Leverage & Scandals", "Apprentice & Graduates"}
	tx := x
	for i, t := range tabs {
		c := "K"
		if st.tab == i {
			c = "Y"
		}
		tx = s.TextB(tx, y, fmt.Sprintf("{{%s|[ %s ]}} ", c, t), CGray, CBgDark, x+w)
	}
	y += 2
	put := func(str string) {
		for _, l := range Wrap(str, w) {
			if y < bottom {
				s.TextB(x, y, l, CGray, CBgDark, x+w)
				y++
			}
		}
	}
	row := func(sel bool, str string) {
		if y >= bottom {
			return
		}
		bg := CBgDark
		if sel {
			bg = CBgSel
		}
		s.Fill(x, y, w, 1, bg)
		s.TextB(x, y, str, CGray, bg, x+w)
		y++
	}
	switch st.tab {
	case 0:
		if st.Alias.Active {
			put(fmt.Sprintf("{{C|Active Alias:}} %s (%s)  {{G|Cover: %d/%d%%}}  {{K|Bounties evaded: %d}}",
				st.Alias.Name, st.Alias.Role, st.Alias.Cover, st.Alias.MaxCover, st.Alias.BountiesEvaded))
		} else {
			put("{{K|No active disguise. Traveling under your own name.}}")
		}
		y++
		st.sel = clampi(st.sel, 0, len(dgDisguises)-1)
		for i, d := range dgDisguises {
			tag := fmt.Sprintf("%d %s", d.Cost, g.Sk.Currency)
			if st.Owned[d.ID] {
				tag = "owned"
			}
			row(st.sel == i, fmt.Sprintf("{{Y|%-18s}} {{K|[%s]}} %s", d.Name, tag, d.Desc))
		}
		y++
		put("{{K|Enter: buy/don disguise  x: drop disguise  Tab: switch tab  Esc: close}}")
	case 1:
		put(fmt.Sprintf("{{Y|Active Leverage: %d}}  {{K|Unused world secrets: %d}}", len(st.Leverage), len(g.Secrets)))
		y++
		if len(st.Leverage) == 0 {
			put("{{K|No compiled leverage yet. Press c to compile from known secrets.}}")
		} else {
			st.sel = clampi(st.sel, 0, len(st.Leverage)-1)
			for i, l := range st.Leverage {
				status := "active"
				if l.Leaked {
					status = "leaked"
				} else if l.Used {
					status = "spent"
				}
				row(st.sel == i, fmt.Sprintf("{{W|%-18s}} {{K|[%s, str %d]}} %s", l.TargetName, status, l.Strength, l.SecretText))
			}
		}
		if len(st.CrierLog) > 0 {
			y++
			put("{{M|Recent Town Crier:}} " + st.CrierLog[len(st.CrierLog)-1])
		}
		y++
		put("{{K|c: compile secret  e: extort  s: silence witness  l: leak scandal  Esc: close}}")
	case 2:
		if st.Apprentice != nil {
			ap := st.Apprentice
			put(fmt.Sprintf("{{G|Current Apprentice:}} %s of %s  Lv %d (XP %d/%d)  Lessons: %d",
				ap.Name, ap.HomeName, ap.Level, ap.XP, ap.Level*40, ap.Lessons))
			if len(ap.Skills) > 0 {
				var sks []string
				for k, v := range ap.Skills {
					sks = append(sks, fmt.Sprintf("%s %d", k, v))
				}
				sort.Strings(sks)
				put("{{C|Trained Skills:}} " + strings.Join(sks, ", "))
			}
		} else {
			put("{{K|No active apprentice. Talk to a villager or press r to recruit nearby.}}")
		}
		y++
		put(fmt.Sprintf("{{Y|Graduated Adventurers (%d):}}", len(st.Graduates)))
		for _, gr := range st.Graduates {
			seen := "wandering"
			if gr.Encountered {
				seen = "met on the road"
			}
			put(fmt.Sprintf("  %s (Lv %d, %s) - %s", gr.Name, gr.Level, nonEmpty(gr.Specialty, "all-rounder"), seen))
		}
		y++
		put("{{K|r: recruit apprentice  t: train skill  g: graduate apprentice  Esc: close}}")
	}
}
