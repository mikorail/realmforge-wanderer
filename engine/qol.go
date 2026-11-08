package engine

import (
	"fmt"
	"strings"
)

// Quality-of-life systems (prefix ql): auto-explore, auto-pickup rules, message log category
// filtering and search, and Markdown adventure diary export.
// State lives in g.ext("qol"). Nothing here draws from g.R.

type qlState struct {
	Gold        bool `json:"gold"`
	Ammo        bool `json:"ammo"`
	Tonics      bool `json:"tonics"`
	Ingredients bool `json:"ingredients"`
}

func (g *Game) ql() *qlState {
	return g.ext("qol", func() any {
		return &qlState{Gold: true, Ammo: true}
	}).(*qlState)
}

func init() {
	turnHooks = append(turnHooks, func(g *Game) { g.qlAutoPickup() })
	mainKeys["Tab"] = func(g *Game) { g.AutoExplore() }
	mainKeys["v"] = func(g *Game) { g.AutoExplore() }
}

// ---- auto-pickup ----

func (g *Game) qlMatchesRule(it *Item) bool {
	if it == nil || uwIsStolen(it) || (it.Extra != nil && it.Extra["uw:own"] > 0) {
		return false
	}
	st := g.ql()
	switch it.Kind {
	case "gold":
		return st.Gold
	case "ammo":
		return st.Ammo
	case "tonic":
		return st.Tonics
	case "ingredient":
		return st.Ingredients
	}
	return false
}

// qlAutoPickup collects items on the player's tile matching active rules during a safe turn.
func (g *Game) qlAutoPickup() {
	if g.P == nil || g.Dead || g.Cur == nil || g.OnWorld {
		return
	}
	if g.qlHostileVisible() || !g.canMoveLoad() {
		return
	}
	its := g.Cur.ItemsAt(g.P.X, g.P.Y)
	if len(its) == 0 {
		return
	}
	for _, it := range its {
		if !g.qlMatchesRule(it) {
			continue
		}
		if it.Kind != "gold" {
			curW, capW := g.burden()
			if curW+itemWeight(it) > capW {
				continue
			}
		}
		g.take(it)
	}
}

// qlToggleRule flips an auto-pickup category rule ("gold", "ammo", "tonics", "ingredients").
func (g *Game) qlToggleRule(kind string) {
	st := g.ql()
	k := strings.ToLower(strings.TrimSpace(kind))
	var label string
	var on bool
	switch k {
	case "gold":
		st.Gold = !st.Gold
		label, on = "gold", st.Gold
	case "ammo":
		st.Ammo = !st.Ammo
		label, on = "ammo", st.Ammo
	case "tonic", "tonics":
		st.Tonics = !st.Tonics
		label, on = "tonics", st.Tonics
	case "ingredient", "ingredients":
		st.Ingredients = !st.Ingredients
		label, on = "ingredients", st.Ingredients
	default:
		return
	}
	stateWord := "off"
	if on {
		stateWord = "on"
	}
	g.Msg(fmt.Sprintf("Auto-pickup for %s is now %s.", label, stateWord))
}

// ---- auto-explore ----

func (g *Game) qlHostileVisible() bool {
	if g.Cur == nil || g.P == nil {
		return false
	}
	for _, a := range g.Cur.Actors {
		if a != g.P && a.HP > 0 && g.Hostile(a) && g.visible(a.X, a.Y) {
			return true
		}
	}
	return false
}

func qlTileHazard(z *Zone, x, y int) bool {
	if !z.In(x, y) {
		return false
	}
	i := y*MW + x
	if z.Fire[i] > 0 || z.Liq[i] == 3 {
		return true
	}
	switch z.At(x, y) {
	case TTrap, TLava, TFumarole, TQuicksand, TTar, TGeyser:
		return true
	}
	return false
}

func (g *Game) qlHazardNear() bool {
	if g.Cur == nil || g.P == nil {
		return false
	}
	z, p := g.Cur, g.P
	if qlTileHazard(z, p.X, p.Y) {
		return true
	}
	for _, d := range dirs8 {
		nx, ny := p.X+d[0], p.Y+d[1]
		if qlTileHazard(z, nx, ny) {
			return true
		}
	}
	return false
}

func (g *Game) qlCanStep(z *Zone, x, y int) bool {
	if !z.In(x, y) || qlTileHazard(z, x, y) {
		return false
	}
	t := z.At(x, y)
	if t == TVoid || t == TChasm {
		return g.P.Pow("wings") > 0
	}
	if !t.D().Walk && t != TDoor {
		return false
	}
	if z.WaterAt(x, y) >= 2 && !z.Underwater && !g.canSwim() {
		return false
	}
	if a := z.ActorAt(x, y); a != nil && a != g.P && a.AI != "follower" {
		return false
	}
	return true
}

func qlTargetItemAt(z *Zone, x, y int) bool {
	for _, it := range z.Items {
		if it.X != x || it.Y != y {
			continue
		}
		if it.Kind == "clutter" || it.Kind == "nest" {
			continue
		}
		if uwIsStolen(it) || (it.Extra != nil && it.Extra["uw:own"] > 0) {
			continue
		}
		return true
	}
	return false
}

// qlNextExploreStep finds the first step along a shortest BFS path from (p.X, p.Y)
// to the nearest unseen walkable tile (adjacent to a seen walkable tile) or uncollected non-clutter item.
func (g *Game) qlNextExploreStep() (int, int, bool) {
	z, p := g.Cur, g.P
	start := p.Y*MW + p.X
	prev := make([]int, MW*MH)
	for i := range prev {
		prev[i] = -1
	}
	prev[start] = start
	q := []int{start}
	target := -1

	for len(q) > 0 && target < 0 {
		cur := q[0]
		q = q[1:]
		cx, cy := cur%MW, cur/MW
		for _, d := range dirs8 {
			nx, ny := cx+d[0], cy+d[1]
			if !z.In(nx, ny) {
				continue
			}
			ni := ny*MW + nx
			if prev[ni] != -1 {
				continue
			}
			if !g.qlCanStep(z, nx, ny) {
				continue
			}
			prev[ni] = cur
			if z.Seen[ni] == 0 {
				if z.Seen[cur] > 0 {
					target = ni
					break
				}
				continue
			}
			if qlTargetItemAt(z, nx, ny) {
				target = ni
				break
			}
			q = append(q, ni)
		}
	}

	if target < 0 {
		return 0, 0, false
	}
	step := target
	for prev[step] != start && prev[step] != -1 {
		step = prev[step]
	}
	return step % MW, step / MW, true
}

// AutoExplore steps along a BFS path toward the nearest unseen walkable tile or non-clutter floor item,
// up to 60 steps, stopping if a hostile is visible, HP drops below 40%, a trap/hazard is adjacent,
// or the zone is fully explored.
func (g *Game) AutoExplore() int {
	if g.P == nil || g.Dead || g.Cur == nil || g.OnWorld {
		return 0
	}
	g.recompute()
	if g.qlHostileVisible() {
		g.Msg("{{R|You cannot auto-explore with enemies in sight!}}")
		return 0
	}
	if g.P.HP*100 < g.P.MaxHP*40 {
		g.Msg("{{R|You are too badly hurt to auto-explore.}}")
		return 0
	}
	if g.qlHazardNear() {
		g.Msg("{{O|You stop near a hazard.}}")
		return 0
	}

	z := g.Cur
	steps := 0
	for steps < 60 && !g.Dead && g.Cur == z && g.Mode == "main" {
		nx, ny, ok := g.qlNextExploreStep()
		if !ok {
			break
		}
		hp0 := g.P.HP
		ox, oy := g.P.X, g.P.Y
		wasDoor := z.At(nx, ny) == TDoor
		hadItem := qlTargetItemAt(z, nx, ny)

		g.MoveP(nx-ox, ny-oy)
		moved := g.P.X != ox || g.P.Y != oy || (wasDoor && z.At(nx, ny) == TDoorOpen)
		if !moved {
			break
		}
		steps++
		g.qlAutoPickup()

		if g.Dead || g.Cur != z || g.Mode != "main" {
			break
		}
		if g.qlHostileVisible() {
			g.Msg("{{R|You spot a hostile creature and stop.}}")
			break
		}
		if g.P.HP < hp0 || g.P.HP*100 < g.P.MaxHP*40 {
			g.Msg("{{R|You are hurt and stop exploring.}}")
			break
		}
		if g.qlHazardNear() {
			g.Msg("{{O|You stop near a hazard.}}")
			break
		}
		if hadItem && g.P.X == nx && g.P.Y == ny && qlTargetItemAt(z, nx, ny) {
			break
		}
	}

	if steps == 0 {
		g.Msg("Nothing left to explore here.")
	} else {
		suf := "s"
		if steps == 1 {
			suf = ""
		}
		g.Msg(fmt.Sprintf("Explored %d step%s.", steps, suf))
	}
	return steps
}

// ---- message log search & category filter ----

var (
	qlRumorWords = []string{
		"rumor", "rumour", "gossip", "heard that", "they say", "word is", "whispers",
		"news:", "rival:", "bounty", "reports that", "sing that", "bard ", "last seen near",
	}
	qlTradeWords = []string{
		"buy", "bought", "sell", "sold", "merchant", "shop", "trade", "coins", "gold",
		"drachm", "scrip", "obols", "shells", "barter", "paid", "pay ", "price",
		"stipend", "wage", "tips", "curio",
	}
	qlLoreWords = []string{
		"discover", "secret", "chronicle", "inscri", "shrine", "altar", "relic", "artifact",
		"ancient", "history", "legend", "read", "book", "scroll", "annals", "codex",
		"pact", "rite", "quest", "journal", "ruin", "epitaph", "grave",
	}
	qlCombatWords = []string{
		"hit", "hits", "miss", "misses", "slay", "slew", "slain", "kill", "killed",
		"dies", "died", "attack", "attacks", "strike", "strikes", "swing", "swings",
		"shoot", "shoots", "shot", "stab", "stabs", "slash", "slashes", "cleave", "bash",
		"bite", "bites", "claw", "claws", "damage", "wounded", "bleed", "bleeding",
		"poison", "burn", "burning", "stun", "stunned", "parry", "dodge", "penetrat",
		"sever", "ambush", "hostile", "sneak attack", "flees", "perished", "hurt",
	}
)

// qlMsgCat classifies a log message into "rumor", "trade", "lore", "combat", or "general".
func qlMsgCat(s string) string {
	low := strings.ToLower(Plain(s))
	for _, w := range qlRumorWords {
		if strings.Contains(low, w) {
			return "rumor"
		}
	}
	for _, w := range qlTradeWords {
		if strings.Contains(low, w) {
			return "trade"
		}
	}
	for _, w := range qlLoreWords {
		if strings.Contains(low, w) {
			return "lore"
		}
	}
	for _, w := range qlCombatWords {
		if strings.Contains(low, w) {
			return "combat"
		}
	}
	return "general"
}

// FilterMsgs filters g.Msgs by category ("all", "combat", "lore", "trade", "rumor") and case-insensitive substring query.
func (g *Game) FilterMsgs(cat, query string) []Msg {
	c := strings.ToLower(strings.TrimSpace(cat))
	if c == "rumours" || c == "rumors" || c == "rumour" {
		c = "rumor"
	}
	q := strings.ToLower(strings.TrimSpace(query))
	var out []Msg
	for _, m := range g.Msgs {
		if c != "" && c != "all" && qlMsgCat(m.T) != c {
			continue
		}
		if q != "" && !strings.Contains(strings.ToLower(Plain(m.T)), q) {
			continue
		}
		out = append(out, m)
	}
	return out
}

// ---- adventure diary export ----

// ExportDiary formats a Markdown chronicle of the character's life and run statistics.
func (g *Game) ExportDiary() string {
	var b strings.Builder
	p := g.P
	if p == nil {
		return "# Adventure Diary\n"
	}

	spName := p.Species
	if g.W != nil {
		if sp := g.W.specByID[p.Species]; sp != nil {
			spName = sp.Name
		}
	}
	callName := g.Calling
	if g.Sk != nil {
		callName = g.Sk.Calling(g.Calling)
	}
	planet := g.unPlanetName()
	days := g.Turn / DayLen
	year := 0
	if g.W != nil {
		year = g.W.Year + days/YearDays
	}
	km := g.fm().Km
	score := g.hfScore()

	fmt.Fprintf(&b, "# Chronicle of %s\n\n", p.Name)
	fmt.Fprintf(&b, "- **Name**: %s\n", p.Name)
	fmt.Fprintf(&b, "- **Species**: %s\n", spName)
	fmt.Fprintf(&b, "- **Calling**: %s\n", callName)
	fmt.Fprintf(&b, "- **Genre**: %s\n", g.Genre)
	fmt.Fprintf(&b, "- **Planet**: %s\n", planet)
	fmt.Fprintf(&b, "- **Year**: %d\n", year)
	fmt.Fprintf(&b, "- **Level**: %d\n", p.Level)
	fmt.Fprintf(&b, "- **Score**: %d\n", score)
	fmt.Fprintf(&b, "- **Turns / Days**: %d turns (%d days)\n", g.Turn, days)
	fmt.Fprintf(&b, "- **Kilometres Walked**: %.1f km\n", km)
	fmt.Fprintf(&b, "- **Kills**: %d\n\n", g.Kills)

	b.WriteString("## Equipped Gear\n\n")
	equipped := 0
	for _, bp := range p.Body {
		if bp.Item != nil && !bp.Severed {
			fmt.Fprintf(&b, "- **%s**: %s\n", bp.Name, Plain(bp.Item.Display()))
			equipped++
		}
	}
	if equipped == 0 {
		b.WriteString("- None\n")
	}
	b.WriteString("\n")

	b.WriteString("## Family & Fame\n\n")
	fmt.Fprintf(&b, "- **Summary**: %s\n", g.fmSummary())
	fmt.Fprintf(&b, "- **Secrets Learned**: %d\n", len(g.Secrets))
	fmt.Fprintf(&b, "- **Quests Undertaken**: %d\n", len(g.Quests))
	fmt.Fprintf(&b, "- **Arena Wins**: %d\n\n", g.ArenaWins)

	b.WriteString("## Major Deeds\n\n")
	if len(g.Deeds) == 0 {
		b.WriteString("- No major deeds recorded yet.\n")
	} else {
		for _, d := range g.Deeds {
			fmt.Fprintf(&b, "- [Year %d, %s] %s\n", d.Year, d.Kind, Plain(d.Text))
		}
	}
	b.WriteString("\n")

	b.WriteString("## Feasts & Songs\n\n")
	if len(g.Feasts) == 0 && len(g.fm().Songs) == 0 {
		b.WriteString("- No feasts joined or songs composed yet.\n")
	} else {
		for _, f := range g.Feasts {
			fmt.Fprintf(&b, "- **Feast**: %s at %s (Year %d): %s\n", f.Name, f.Town, f.Year, Plain(f.Text))
		}
		for _, s := range g.fm().Songs {
			fmt.Fprintf(&b, "- **Song**: %s (sung %d times)\n", Plain(s.Text), s.Sung)
		}
	}
	b.WriteString("\n")

	b.WriteString("## Final Fate\n\n")
	place := "the wilds"
	if g.Cur != nil && g.Cur.Name != "" {
		place = g.Cur.Name
	}
	if g.Dead {
		cause := strings.TrimSpace(Plain(g.DeathText))
		if cause == "" {
			cause = "Perished in " + place + "."
		}
		fmt.Fprintf(&b, "- Fallen in %s: %s\n", place, cause)
	} else {
		fmt.Fprintf(&b, "- Still wandering in %s (HP %d/%d).\n", place, p.HP, p.MaxHP)
	}

	return b.String()
}
