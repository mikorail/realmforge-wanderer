package engine

import (
	"fmt"
	"strings"
)

// Focused 30-Minute Challenge Scenarios (prefix scn).
// Generated deterministically from any world using side RNGs (g.Seed+":scn:...").
// Saved state lives in g.ext("scn"), planet-local ("scn" in unLocalKeys).

func init() {
	unLocalKeys = append(unLocalKeys, "scn")
	turnHooks = append(turnHooks, scnTurn)
	zoneHooks = append(zoneHooks, scnZone)
	enterHooks = append(enterHooks, scnEnter)
	killHooks = append(killHooks, scnKill)
	talkHooks = append(talkHooks, scnTalk)
	sideHooks = append(sideHooks, scnSide)
	modeKeys["scn"] = scnKeys
	modeDraws["scn"] = scnDraw
	paEntries = append(paEntries, paEntry{
		Name:     "Challenge scenarios",
		Keywords: "scenario challenge siege murder plague medal timer",
		Cat:      "Screen",
		Func:     func(g *Game) { g.scnOpen() },
	})
}

type scnDef struct {
	ID         string
	Title      string
	Summary    string
	LimitTurns int
}

var scnDefs = []scnDef{
	{
		ID:         "escape_siege",
		Title:      "Escape the Besieged Capital",
		Summary:    "Breach or slip past the siege cordons around the capital before the ring closes.",
		LimitTurns: 360,
	},
	{
		ID:         "festival_murder",
		Title:      "Solve the 3-Day Murder Before the Festival",
		Summary:    "Gather 3 clues and accuse the true culprit before the festival bells ring.",
		LimitTurns: 3 * DayLen,
	},
	{
		ID:         "plague_relic",
		Title:      "Deliver the Relic Across a Plague-Stricken Continent",
		Summary:    "Carry the cure relic across the realm to the distant target burg before time runs out.",
		LimitTurns: 720,
	},
}

func scnFindDef(id string) *scnDef {
	for i := range scnDefs {
		if scnDefs[i].ID == id {
			return &scnDefs[i]
		}
	}
	return nil
}

type scnObj struct {
	ID   string `json:"id"`
	Text string `json:"t"`
	Done bool   `json:"d,omitempty"`
}

type scnRun struct {
	ID              string   `json:"id"`
	Title           string   `json:"ti"`
	Status          string   `json:"st"` // active, won, failed
	StartTurn       int      `json:"s"`
	LimitTurns      int      `json:"lim"`
	ElapsedTurns    int      `json:"el"`
	BurgID          int      `json:"bid,omitempty"`
	BurgName        string   `json:"bn,omitempty"`
	StartWX         int      `json:"swx,omitempty"`
	StartWY         int      `json:"swy,omitempty"`
	TargetBurgID    int      `json:"tbid,omitempty"`
	TargetBurgName  string   `json:"tbn,omitempty"`
	TargetWX        int      `json:"twx,omitempty"`
	TargetWY        int      `json:"twy,omitempty"`
	Objectives      []scnObj `json:"obj,omitempty"`
	Score           int      `json:"sc,omitempty"`
	Rank            string   `json:"rk,omitempty"` // Bronze, Silver, Gold
	CordonsBreached int      `json:"cb,omitempty"`
	Victim          string   `json:"vic,omitempty"`
	Culprit         string   `json:"cul,omitempty"`
	Suspects        []string `json:"sus,omitempty"`
	Clues           []string `json:"clu,omitempty"`
	CluesFound      int      `json:"cf,omitempty"`
	WrongAccuses    int      `json:"wa,omitempty"`
	RelicItemID     int      `json:"rid,omitempty"`
}

type scnMedal struct {
	ScenarioID  string `json:"id"`
	Title       string `json:"ti"`
	BestScore   int    `json:"bs"`
	BestRank    string `json:"br"`
	Completions int    `json:"co"`
	BestTurns   int    `json:"bt"`
}

type scnState struct {
	Active  *scnRun              `json:"act,omitempty"`
	Medals  map[string]*scnMedal `json:"med,omitempty"`
	History []scnRun             `json:"his,omitempty"`

	tab int
	sel int
}

func scnSt(g *Game) *scnState {
	st := g.ext("scn", func() any { return &scnState{} }).(*scnState)
	if st.Medals == nil {
		st.Medals = map[string]*scnMedal{}
	}
	return st
}

func (g *Game) scnPickBurgs(r *Rng) (capital, distant *BurgData) {
	var alive []*BurgData
	for i := range g.W.Burgs {
		b := &g.W.Burgs[i]
		if b.Alive && b.Name != "" {
			alive = append(alive, b)
			if b.Capital && capital == nil {
				capital = b
			}
		}
	}
	if capital == nil && len(alive) > 0 {
		capital = alive[0]
	}
	if capital != nil && len(alive) > 1 {
		cx, cy := capital.P%g.W.W, capital.P/g.W.W
		bestDist := -1
		for _, b := range alive {
			if b.ID == capital.ID {
				continue
			}
			bx, by := b.P%g.W.W, b.P/g.W.W
			d := abs(bx-cx) + abs(by-cy)
			if d > bestDist {
				bestDist = d
				distant = b
			}
		}
	}
	if distant == nil {
		distant = capital
	}
	return capital, distant
}

// scnStart launches one of the 3 focused challenge scenarios from the current world.
func (g *Game) scnStart(id string) (*scnRun, bool) {
	def := scnFindDef(id)
	if def == nil {
		return nil, false
	}
	st := scnSt(g)
	r := NewRng(g.Seed + ":scn:" + id)
	capBurg, farBurg := g.scnPickBurgs(r)

	run := &scnRun{
		ID:         def.ID,
		Title:      def.Title,
		Status:     "active",
		StartTurn:  g.Turn,
		LimitTurns: def.LimitTurns,
	}
	if capBurg != nil {
		run.BurgID = capBurg.ID
		run.BurgName = capBurg.Name
		run.StartWX = capBurg.P % g.W.W
		run.StartWY = capBurg.P / g.W.W
	} else {
		run.BurgName = "the Capital"
		run.StartWX = g.WX
		run.StartWY = g.WY
	}

	switch id {
	case "escape_siege":
		run.Objectives = []scnObj{
			{ID: "cordon1", Text: "Breach or slip past the inner siege cordon"},
			{ID: "cordon2", Text: "Breach or slip past the outer siege cordon"},
			{ID: "escape", Text: "Reach safety outside " + run.BurgName},
		}
	case "festival_murder":
		victims := []string{"Guildmaster Hal", "Cantor Sela", "Alderman Bram", "Magistrate Orin"}
		suspectPool := []string{"Vesper the Steward", "Maren the Apothecary", "Corvin the Captain", "Liora the Envoy"}
		run.Victim = Pick(r, victims)
		run.Suspects = append([]string{}, suspectPool[:3]...)
		run.Culprit = run.Suspects[r.Int(0, len(run.Suspects)-1)]
		run.Clues = []string{
			fmt.Sprintf("A torn ledger signed near %s's study pointing to %s.", run.Victim, run.Culprit),
			fmt.Sprintf("A vial of festival ink stained with blood matching %s's coat.", run.Culprit),
			fmt.Sprintf("A watchman saw %s slipping out before the bell struck.", run.Culprit),
		}
		run.Objectives = []scnObj{
			{ID: "clues", Text: "Gather 3 murder clues (0/3)"},
			{ID: "accuse", Text: "Accuse the culprit before the festival deadline"},
		}
	case "plague_relic":
		if farBurg != nil {
			run.TargetBurgID = farBurg.ID
			run.TargetBurgName = farBurg.Name
			run.TargetWX = farBurg.P % g.W.W
			run.TargetWY = farBurg.P / g.W.W
		} else {
			run.TargetBurgName = "the Distant Haven"
		}
		relic := &Item{
			ID:     g.newID(),
			Name:   "Relic of the Clean Breath",
			Glyph:  '&',
			F:      CCyan,
			Kind:   "artifact",
			Effect: "scn_plague_relic",
			Value:  500,
			Weight: 2,
			Ident:  true,
			Lore:   "A sealed censer carrying a living cure bound for " + run.TargetBurgName + ".",
		}
		g.give(relic)
		run.RelicItemID = relic.ID
		run.Objectives = []scnObj{
			{ID: "carry", Text: "Keep the Relic of the Clean Breath safe", Done: true},
			{ID: "deliver", Text: "Deliver the cure relic to " + run.TargetBurgName},
		}
	}

	st.Active = run
	g.Msg(fmt.Sprintf("{{Y|Challenge Scenario Started: %s (Time limit: %d turns).}}", run.Title, run.LimitTurns))
	return run, true
}

func scnRankFor(score int) string {
	switch {
	case score >= 850:
		return "Gold"
	case score >= 600:
		return "Silver"
	default:
		return "Bronze"
	}
}

func (g *Game) scnFinish(won bool, reason string) {
	st := scnSt(g)
	run := st.Active
	if run == nil || run.Status != "active" {
		return
	}
	run.ElapsedTurns = maxi(1, g.Turn-run.StartTurn)
	if !won {
		run.Status = "failed"
		run.Score = 0
		run.Rank = ""
		st.History = append(st.History, *run)
		g.Msg("{{R|Scenario Failed: " + run.Title + " - " + reason + "}}")
		return
	}
	run.Status = "won"
	rem := maxi(0, run.LimitTurns-run.ElapsedTurns)
	timeBonus := 0
	if run.LimitTurns > 0 {
		timeBonus = rem * 500 / run.LimitTurns
	}
	penalty := run.WrongAccuses * 150
	run.Score = maxi(100, 500+timeBonus-penalty)
	run.Rank = scnRankFor(run.Score)

	m := st.Medals[run.ID]
	if m == nil {
		m = &scnMedal{
			ScenarioID: run.ID,
			Title:      run.Title,
			BestTurns:  run.ElapsedTurns,
		}
		st.Medals[run.ID] = m
	}
	m.Completions++
	if run.Score > m.BestScore {
		m.BestScore = run.Score
		m.BestRank = run.Rank
	}
	if run.ElapsedTurns < m.BestTurns || m.BestTurns == 0 {
		m.BestTurns = run.ElapsedTurns
	}
	st.History = append(st.History, *run)
	g.deed("quest", fmt.Sprintf("%s completed scenario '%s' (%s medal, score %d).", g.P.Name, run.Title, run.Rank, run.Score), 0, 0)
	g.Msg(fmt.Sprintf("{{G|Scenario Complete: %s! Score: %d (%s Medal).}}", run.Title, run.Score, run.Rank))
}

// ---- Scenario Actions ----

// scnBreachCordon advances the escape_siege scenario when breaching/slipping a siege cordon.
func (g *Game) scnBreachCordon() bool {
	st := scnSt(g)
	run := st.Active
	if run == nil || run.Status != "active" || run.ID != "escape_siege" {
		return false
	}
	run.CordonsBreached++
	if run.CordonsBreached >= 1 && len(run.Objectives) > 0 {
		run.Objectives[0].Done = true
	}
	if run.CordonsBreached >= 2 && len(run.Objectives) > 1 {
		run.Objectives[1].Done = true
	}
	g.Msg(fmt.Sprintf("{{C|Siege cordon breached (%d/2).}}", mini(2, run.CordonsBreached)))
	if run.CordonsBreached >= 2 {
		if len(run.Objectives) > 2 {
			run.Objectives[2].Done = true
		}
		g.scnFinish(true, "Escaped the siege ring")
	}
	return true
}

// scnGatherClue collects one of the 3 murder clues in festival_murder.
func (g *Game) scnGatherClue(idx int) (string, bool) {
	st := scnSt(g)
	run := st.Active
	if run == nil || run.Status != "active" || run.ID != "festival_murder" {
		return "", false
	}
	if idx < 0 || idx >= len(run.Clues) {
		idx = run.CluesFound
	}
	if idx >= len(run.Clues) {
		return "", false
	}
	clue := run.Clues[idx]
	if run.CluesFound < len(run.Clues) {
		run.CluesFound++
	}
	if len(run.Objectives) > 0 {
		run.Objectives[0].Text = fmt.Sprintf("Gather 3 murder clues (%d/3)", run.CluesFound)
		if run.CluesFound >= 3 {
			run.Objectives[0].Done = true
		}
	}
	g.Msg("{{C|Clue gathered: " + clue + "}}")
	return clue, true
}

// scnAccuseCulprit accuses a suspect in festival_murder.
func (g *Game) scnAccuseCulprit(suspect string) (bool, string) {
	st := scnSt(g)
	run := st.Active
	if run == nil || run.Status != "active" || run.ID != "festival_murder" {
		return false, "No active murder scenario."
	}
	if strings.EqualFold(strings.TrimSpace(suspect), run.Culprit) {
		if len(run.Objectives) > 1 {
			run.Objectives[1].Done = true
		}
		g.scnFinish(true, "True culprit exposed")
		return true, fmt.Sprintf("%s confesses before the festival crowd!", run.Culprit)
	}
	run.WrongAccuses++
	msg := fmt.Sprintf("%s proves their alibi! That accusation cost precious time.", suspect)
	g.Msg("{{R|" + msg + "}}")
	return false, msg
}

func (g *Game) scnHasRelic(run *scnRun) *Item {
	if g.P == nil || run == nil {
		return nil
	}
	for _, it := range g.P.Inv {
		if it != nil && (it.ID == run.RelicItemID || it.Effect == "scn_plague_relic") {
			return it
		}
	}
	return nil
}

// scnDeliverRelic completes plague_relic when the player holds the cure relic.
func (g *Game) scnDeliverRelic() (bool, string) {
	st := scnSt(g)
	run := st.Active
	if run == nil || run.Status != "active" || run.ID != "plague_relic" {
		return false, "No active plague relic scenario."
	}
	rel := g.scnHasRelic(run)
	if rel == nil {
		return false, "You are not carrying the Relic of the Clean Breath."
	}
	g.removeInv(rel)
	if len(run.Objectives) > 1 {
		run.Objectives[1].Done = true
	}
	g.scnFinish(true, "Cure relic delivered to "+run.TargetBurgName)
	return true, "The healers of " + run.TargetBurgName + " unseal the cure relic!"
}

// ---- Hooks ----

func scnTurn(g *Game) {
	st := scnSt(g)
	run := st.Active
	if run == nil || run.Status != "active" {
		return
	}
	run.ElapsedTurns = maxi(0, g.Turn-run.StartTurn)
	if run.ElapsedTurns > run.LimitTurns {
		g.scnFinish(false, "Time limit expired")
		return
	}
	// Check automatic distance escape for escape_siege.
	if run.ID == "escape_siege" {
		dist := abs(g.WX-run.StartWX) + abs(g.WY-run.StartWY)
		if dist >= 3 {
			run.CordonsBreached = maxi(run.CordonsBreached, 2)
			for i := range run.Objectives {
				run.Objectives[i].Done = true
			}
			g.scnFinish(true, "Outran the siege perimeter")
		}
	}
}

func scnEnter(g *Game, zn *Zone) {
	if zn == nil {
		return
	}
	st := scnSt(g)
	run := st.Active
	if run == nil || run.Status != "active" {
		return
	}
	if run.ID == "plague_relic" && run.TargetBurgID > 0 && zn.Burg == run.TargetBurgID {
		g.scnDeliverRelic()
	}
	if run.ID == "escape_siege" && abs(zn.WX-run.StartWX)+abs(zn.WY-run.StartWY) >= 3 {
		run.CordonsBreached = maxi(run.CordonsBreached, 2)
		for i := range run.Objectives {
			run.Objectives[i].Done = true
		}
		g.scnFinish(true, "Reached open country beyond the siege")
	}
}

func scnKill(g *Game, d, src *Actor) {
	if d == nil || src != g.P {
		return
	}
	st := scnSt(g)
	run := st.Active
	if run != nil && run.Status == "active" && run.ID == "escape_siege" && (d.Role == "guard" || d.Has("scn_cordon")) {
		g.scnBreachCordon()
	}
}

func scnZone(g *Game, zn *Zone, _ *Rng) {
	if zn == nil {
		return
	}
	st := scnSt(g)
	run := st.Active
	if run == nil || run.Status != "active" || run.ID != "escape_siege" {
		return
	}
	if abs(zn.WX-run.StartWX)+abs(zn.WY-run.StartWY) <= 1 {
		r := NewRng(g.Seed + ":scn:cordon:" + zn.Key)
		if x, y, ok := zn.FreeSpot(r, nil); ok {
			guard := g.Spawn("guard", maxi(1, g.P.Level), r)
			if guard != nil {
				guard.Name = "Siege Cordon Sentry"
				guard.Role = "guard"
				guard.X, guard.Y = x, y
				guard.Set("scn_cordon")
				zn.Actors = append(zn.Actors, guard)
			}
		}
	}
}

func scnTalk(g *Game, a *Actor) []Option {
	if a == nil {
		return nil
	}
	st := scnSt(g)
	run := st.Active
	if run == nil || run.Status != "active" {
		return nil
	}
	var out []Option
	switch run.ID {
	case "festival_murder":
		if run.CluesFound < 3 {
			out = append(out, Option{"{{C|Question them about the festival murder.}}", func() {
				if clue, ok := g.scnGatherClue(run.CluesFound); ok {
					g.talkMenu("\"Keep your voice down...\" " + clue)
				}
			}})
		}
		for _, sus := range run.Suspects {
			sus := sus
			out = append(out, Option{"{{R|Accuse " + sus + " of the murder.}}", func() {
				_, msg := g.scnAccuseCulprit(sus)
				g.talkMenu(msg)
			}})
		}
	case "plague_relic":
		if g.scnHasRelic(run) != nil && (a.Role == "priest" || a.Role == "mayor" || a.Role == "herbalist") {
			out = append(out, Option{"{{G|Hand over the Relic of the Clean Breath.}}", func() {
				_, msg := g.scnDeliverRelic()
				g.talkMenu(msg)
			}})
		}
	}
	return out
}

func scnSide(g *Game) []string {
	st := scnSt(g)
	run := st.Active
	if run == nil || run.Status != "active" {
		return nil
	}
	rem := maxi(0, run.LimitTurns-run.ElapsedTurns)
	done := 0
	for _, o := range run.Objectives {
		if o.Done {
			done++
		}
	}
	return []string{
		fmt.Sprintf("{{Y|Scenario: %s}}", run.ID),
		fmt.Sprintf("{{C|Time: %d turns  Obj: %d/%d}}", rem, done, len(run.Objectives)),
	}
}

// ---- Interactive Mode "scn" Screen ----

func (g *Game) scnOpen() {
	st := scnSt(g)
	st.tab = 0
	st.sel = 0
	g.Mode = "scn"
}

func scnKeys(g *Game, k string) {
	st := scnSt(g)
	switch k {
	case "Escape", "q":
		g.Mode = "main"
		return
	case "Tab", "ArrowRight":
		st.tab = (st.tab + 1) % 2
		st.sel = 0
		return
	case "ArrowLeft":
		st.tab = (st.tab + 1) % 2
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
	case 0:
		st.sel = clampi(st.sel, 0, len(scnDefs)-1)
		if k == "Enter" || k == "s" {
			g.scnStart(scnDefs[st.sel].ID)
		}
	case 1:
		if k == "x" && st.Active != nil && st.Active.Status == "active" {
			g.scnFinish(false, "Abandoned by player")
		}
	}
}

func scnDraw(g *Game) {
	s := g.sc
	st := scnSt(g)
	x, y, w, h := g.panel("Challenge Scenarios (30-Min Runs)")
	bottom := y + h - 1
	tabs := []string{"Scenarios", "Active & Medals"}
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
		st.sel = clampi(st.sel, 0, len(scnDefs)-1)
		for i, d := range scnDefs {
			medal := "no medal"
			if m := st.Medals[d.ID]; m != nil && m.BestRank != "" {
				medal = fmt.Sprintf("%s (%d)", m.BestRank, m.BestScore)
			}
			row(st.sel == i, fmt.Sprintf("{{Y|%-46s}} {{K|[%s, %d turns]}}", d.Title, medal, d.LimitTurns))
		}
		y++
		put("{{W|Brief:}} " + scnDefs[st.sel].Summary)
		y++
		put("{{K|Enter/s: start scenario  Tab: active & medals  Esc: close}}")
	case 1:
		if run := st.Active; run != nil {
			rem := maxi(0, run.LimitTurns-run.ElapsedTurns)
			put(fmt.Sprintf("{{Y|Active: %s}} {{K|(%s, %d turns left)}}", run.Title, run.Status, rem))
			for _, o := range run.Objectives {
				mark := "[ ]"
				if o.Done {
					mark = "[x]"
				}
				put("  " + mark + " " + o.Text)
			}
			if run.Status == "won" {
				put(fmt.Sprintf("{{G|Final Score: %d (%s Medal)}}", run.Score, run.Rank))
			}
		} else {
			put("{{K|No active challenge scenario.}}")
		}
		y++
		put("{{Y|Earned Medals:}}")
		if len(st.Medals) == 0 {
			put("  {{K|None yet.}}")
		} else {
			for _, d := range scnDefs {
				if m := st.Medals[d.ID]; m != nil {
					put(fmt.Sprintf("  {{G|%-8s}} %-40s Best: %d (%d runs)", m.BestRank, m.Title, m.BestScore, m.Completions))
				}
			}
		}
		y++
		put("{{K|x: abandon active scenario  Tab: scenarios  Esc: close}}")
	}
}
