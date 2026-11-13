package engine

import (
	"fmt"
	"strings"
)

// Achievements and titles (prefix ach, state key "ac"). About seventy badges evaluated once every few
// game minutes (and on kills and zone entries) from existing state and a few counters kept here:
// deed kinds, kills by sort, deepest level, planets, crypts, beds. Unlocking shows a highlighted message
// and, for the ones that grant a title, a deed. A title can be worn from the Journal+ screen (F10, tab
// Achievements, key T). `acObituary` writes the death screen's short life story. Reads only; never g.R.

type acState struct {
	Done    map[string]int  `json:"d,omitempty"` // id -> turn unlocked + 1
	C       map[string]int  `json:"c,omitempty"` // counters (kills, depth, tomb, ...)
	K       map[string]int  `json:"k,omitempty"` // deed kind -> count
	Planets map[int]bool    `json:"pl,omitempty"`
	Zones   map[string]bool `json:"zn,omitempty"` // crypt / wonder / ruin zones entered
	Seen    int             `json:"ds,omitempty"` // deeds processed
	Worn    string          `json:"wn,omitempty"` // worn title
	Last    int             `json:"lt,omitempty"`

	sel, top int
}

func (g *Game) ac() *acState {
	st := g.ext("ac", func() any { return &acState{} }).(*acState)
	if st.Done == nil {
		st.Done, st.C, st.K = map[string]int{}, map[string]int{}, map[string]int{}
		st.Planets, st.Zones = map[int]bool{}, map[string]bool{}
	}
	return st
}

func init() {
	turnHooks = append(turnHooks, achTurn)
	killHooks = append(killHooks, achKill)
	enterHooks = append(enterHooks, achEnter)
}

func achTurn(g *Game) {
	if g.P == nil || g.W == nil || g.Cur == nil {
		return
	}
	st := g.ac()
	if g.Turn-st.Last < 5 && st.Last != 0 {
		return
	}
	st.Last = g.Turn
	st.Planets[g.PlanetIndex()] = true
	if g.Turn%60 < 5 || st.C["known"] == 0 {
		n := 0
		for _, k := range g.Known {
			if k {
				n++
			}
		}
		st.C["known"] = maxi(st.C["known"], n)
	}
	if g.P.HP == 1 && !g.Dead {
		st.C["close_call"] = 1
	}
	// beds: the log says when you slept in one
	for i := len(g.Msgs) - 1; i >= 0 && i >= len(g.Msgs)-8; i-- {
		if g.Msgs[i].Turn < g.Turn-6 {
			break
		}
		if strings.Contains(g.Msgs[i].T, "in your own bed") {
			st.C["bed"]++
			break
		}
	}
	achDeeds(g, st)
	achCheck(g, st)
}

func achKill(g *Game, d, src *Actor) {
	if d == nil || g.P == nil || (src != g.P && (src == nil || src.AI != "follower")) {
		return
	}
	st := g.ac()
	st.C["kills"]++
	if d.Boss {
		st.C["boss"]++
	}
	id := d.Tpl + " " + d.Name + " " + d.Faction
	switch {
	case achMatch(id, "dragon", "wyrm", "drake", "wyvern"):
		st.C["dragon"]++
	case d.Has("undead") || achMatch(id, "undead", "skeleton", "zombie", "wraith", "ghoul", "lich", "ghost", "revenant", "mummy", "wight"):
		st.C["undead"]++
	}
	if d.Species != "" || d.SimPer > 0 {
		st.C["people"]++
	} else {
		st.C["beast"]++
	}
	achCheck(g, st)
}

func achEnter(g *Game, zn *Zone) {
	if g.P == nil || zn == nil {
		return
	}
	st := g.ac()
	st.Planets[g.PlanetIndex()] = true
	if zn.Z > 0 {
		st.C["depth"] = maxi(st.C["depth"], zn.Z)
	}
	if zn.Z <= -100 {
		st.C["tower"] = maxi(st.C["tower"], -100-zn.Z)
	}
	if zn.Sky {
		st.C["sky"] = 1
	}
	if zn.Underwater {
		st.C["sea"] = 1
	}
	if zn.Kind == "underdark" {
		st.C["underdark"] = 1
	}
	switch zn.Kind {
	case "crypt", "wonder", "ruin":
		if !st.Zones[zn.Key] {
			st.Zones[zn.Key] = true
			st.C[zn.Kind]++
		}
	}
	achCheck(g, st)
}

// achDeeds folds new deeds into counters.
func achDeeds(g *Game, st *acState) {
	for ; st.Seen < len(g.Deeds); st.Seen++ {
		d := g.Deeds[st.Seen]
		st.K[d.Kind]++
		switch {
		case d.Kind == "survived":
			st.C["plague"]++
		case d.Kind == "relic" && strings.Contains(d.Text, "recovered the relic") && g.Cur != nil && g.Cur.Kind == "crypt":
			st.C["tomb"]++
		}
	}
}

func achCheck(g *Game, st *acState) {
	unlocked := 0
	for i := range achDefs {
		d := &achDefs[i]
		if st.Done[d.ID] != 0 || d.Get(g, st) < d.Need {
			continue
		}
		st.Done[d.ID] = g.Turn + 1
		unlocked++
		if unlocked <= 3 {
			g.Msg(fmt.Sprintf("{{Y|★ Achievement:}} {{W|%s}} {{K|%s}}", d.Name, d.Desc))
		}
		if d.Title != "" {
			g.deed("event", fmt.Sprintf("%s earned the name %s: %s", g.P.Name, d.Title, strings.TrimSuffix(d.Desc, ".")+"."), 0, 0)
			if st.Worn == "" {
				st.Worn = d.ID
			}
		}
	}
	if unlocked > 3 {
		g.Msg(fmt.Sprintf("{{Y|★ %d more achievements unlocked.}} {{K|(F10 Journal+)}}", unlocked-3))
	}
}

// acCount is how many badges are earned and how many exist.
func (g *Game) acCount() (int, int) { return len(g.ac().Done), len(achDefs) }

// acTitle is the worn title ("" = none), such as "Dragonbane".
func (g *Game) acTitle() string {
	st := g.ac()
	if d := achByID(st.Worn); d != nil && st.Done[d.ID] != 0 {
		return d.Title
	}
	return ""
}

func achProgress(g *Game, st *acState, d *achDef) string {
	if d.Need <= 1 {
		return ""
	}
	return fmt.Sprintf("%d/%d", mini(d.Get(g, st), d.Need), d.Need)
}

// ---- Journal+ tab

func (g *Game) achKey(k string) {
	st := g.ac()
	n := len(achDefs)
	switch k {
	case "ArrowDown", "Down", "j":
		st.sel = mini(st.sel+1, n-1)
	case "ArrowUp", "Up", "k":
		st.sel = maxi(st.sel-1, 0)
	case "PageDown":
		st.sel = mini(st.sel+10, n-1)
	case "PageUp":
		st.sel = maxi(st.sel-10, 0)
	case "Home":
		st.sel = 0
	case "End":
		st.sel = n - 1
	case "T", "t", "Enter":
		d := &achDefs[clampi(st.sel, 0, n-1)]
		switch {
		case st.Done[d.ID] == 0:
			g.Msg("{{K|Not earned yet.}}")
		case d.Title == "":
			g.Msg("{{K|That badge carries no title.}}")
		case st.Worn == d.ID:
			st.Worn = ""
			g.Msg("You set your title aside.")
		default:
			st.Worn = d.ID
			g.Msg(fmt.Sprintf("You are now {{Y|%s}}.", fmtTitleLine(g.P.Name, d.Title)))
		}
	}
}

func fmtTitleLine(name, title string) string {
	if strings.HasPrefix(title, "the ") || strings.HasPrefix(title, "The ") {
		return name + " " + title
	}
	return name + ", " + title
}

func (g *Game) achDraw(x, y, w, h int) {
	s := g.sc
	st := g.ac()
	got, all := g.acCount()
	tl := "none"
	if t := g.acTitle(); t != "" {
		tl = fmtTitleLine(g.P.Name, t)
	}
	s.TextB(x, y, fmt.Sprintf("{{W|%d}} of {{W|%d}} earned   {{K|title:}} {{Y|%s}}", got, all, tl), CGray, CBgDark, x+w)
	y += 2
	rows := h - 5
	if rows < 3 {
		return
	}
	st.sel = clampi(st.sel, 0, len(achDefs)-1)
	if st.sel < st.top {
		st.top = st.sel
	}
	if st.sel >= st.top+rows {
		st.top = st.sel - rows + 1
	}
	for i := st.top; i < len(achDefs) && i < st.top+rows; i++ {
		d := &achDefs[i]
		bg := CBgDark
		if i == st.sel {
			bg = CBgSel
		}
		s.Fill(x, y, w, 1, bg)
		mark, col := "{{K|[ ]}}", "K"
		if st.Done[d.ID] != 0 {
			mark, col = "{{G|[★]}}", "W"
		}
		pr := ""
		if st.Done[d.ID] == 0 {
			if p := achProgress(g, st, d); p != "" {
				pr = " {{K|(" + p + ")}}"
			}
		}
		tt := ""
		if d.Title != "" {
			tt = " {{Y|*}}"
		}
		s.TextB(x, y, fmt.Sprintf("%s {{%s|%-22s}}%s {{K|%s}}%s", mark, col, d.Name, tt, d.Desc, pr), CGray, bg, x+w)
		y++
	}
	d := &achDefs[st.sel]
	s.TextB(x, y+1, "{{K|["+d.Cat+"] ↑↓ scroll · T wear or set aside a title (* = grants one) · Tab switch · Esc close}}", CGray, CBgDark, x+w)
}

// ---- obituary

// acObituary is the short life story shown on the death screen (each line at most 60 wide; markup).
// Hook: drawDead, after the DeathText wrap: `lines = append(lines, g.acObituary()...)`.
func (g *Game) acObituary() []string {
	if g.P == nil || g.W == nil {
		return nil
	}
	st := g.ac()
	days := g.Turn / DayLen
	who := g.P.Name
	if t := g.acTitle(); t != "" {
		who = fmtTitleLine(g.P.Name, t)
	}
	kind := strings.TrimSpace(titleCase(g.P.Species) + " " + g.Sk.Calling(g.Calling))
	place := "the wilds"
	if g.Cur != nil {
		place = g.Cur.Name
	}
	cause := strings.TrimSpace(Plain(g.DeathText))
	cause = strings.TrimPrefix(cause, "You were killed by ")
	cause = strings.TrimSuffix(cause, ".")
	var out []string
	out = append(out, "", "{{Y|Obituary}}")
	head := fmt.Sprintf("{{W|%s}}, %s, fell in %s in the year %d after %s of wandering.", who, kind, place, g.W.Year, achDays(days))
	out = append(out, Wrap(head, 60)...)
	if cause != "" && cause != Plain(g.DeathText) {
		out = append(out, Wrap("Cause: killed by "+cause+".", 60)...)
	}
	out = append(out, Wrap(fmt.Sprintf("Reached level %d; %d slain; %d places discovered; %d quests done.", g.P.Level, g.Kills, len(g.Visited), achQuestsClosed(g, st)), 60)...)
	if top := achTopDeeds(g, 4); len(top) > 0 {
		out = append(out, "{{K|Remembered for:}}")
		for _, t := range top {
			out = append(out, Wrap("{{K|-}} "+t, 58)...)
		}
	}
	if sm := g.fmSummary(); strings.TrimSpace(sm) != "" {
		out = append(out, Wrap("{{K|Family:}} "+sm, 60)...)
	}
	got, all := g.acCount()
	var names []string
	for i := range achDefs {
		if st.Done[achDefs[i].ID] != 0 && achDefs[i].Title != "" && len(names) < 4 {
			names = append(names, achDefs[i].Name)
		}
	}
	line := fmt.Sprintf("Badges: %d of %d", got, all)
	if len(names) > 0 {
		line += " (" + strings.Join(names, ", ") + ")"
	}
	out = append(out, Wrap("{{K|"+line+".}}", 60)...)
	return out
}

func achDays(d int) string {
	switch {
	case d == 1:
		return "one day"
	case d >= 112:
		return fmt.Sprintf("%d days (%d years)", d, d/112)
	}
	return fmt.Sprintf("%d days", d)
}

// achTopDeeds picks the weightiest deeds of the run, most important kinds first, newest first within a kind.
func achTopDeeds(g *Game, n int) []string {
	rank := map[string]int{"slay": 0, "endgame": 0, "villain": 1, "relic": 1, "quest": 2, "kinship": 2, "murim": 3, "bounty": 3, "order": 3, "ship": 4, "companion": 4, "survived": 4, "ghost": 5, "detective": 5, "discovery": 5}
	var out []string
	for r := 0; r <= 5 && len(out) < n; r++ {
		for i := len(g.Deeds) - 1; i >= 0 && len(out) < n; i-- {
			d := g.Deeds[i]
			if v, ok := rank[d.Kind]; ok && v == r && d.Text != "" {
				out = append(out, d.Text)
			}
		}
	}
	return out
}
