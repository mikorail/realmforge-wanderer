package engine

import (
	"fmt"
	"sort"
	"strings"
)

// Arcane screen (F8): tabs Research, Pacts, Masters.

type arRow struct {
	Text  string
	Do    func()
	Info  []string
	Focus bool
}

var arTabNames = []string{"Research", "Pacts", "Masters"}

func (g *Game) arSelectedSources() []arSrc {
	st := g.ar()
	have := map[string]arSrc{}
	for _, s := range g.arSources() {
		have[s.Key] = s
	}
	var out []arSrc
	for _, k := range st.picks {
		if s, ok := have[k]; ok {
			out = append(out, s)
		}
	}
	return out
}

func (g *Game) arSortedPowers() []*arPower {
	var ps []*arPower
	for _, p := range g.ar().Powers {
		ps = append(ps, p)
	}
	sort.Slice(ps, func(i, j int) bool {
		if ps[i].Name != ps[j].Name {
			return ps[i].Name < ps[j].Name
		}
		return ps[i].ID < ps[j].ID
	})
	return ps
}

func (g *Game) arBoonText(pat arPatron) []string {
	at := AttrShort[arElemAttr[pat.Elem]]
	return []string{
		fmt.Sprintf("Rank 1: +1 %s; call an emissary twice a day.", at),
		fmt.Sprintf("Rank 2: the gift of %s; three calls a day.", g.Sk.PowerName(arElemPower[pat.Elem])),
		fmt.Sprintf("Rank 3: +2 %s; a second pact may be taken.", at),
		"Rank 4: +3 stat, the gift grows, stronger emissaries.",
	}
}

func (g *Game) arResearchRows() []arRow {
	st := g.ar()
	var rows []arRow
	switch st.stage {
	case 0:
		rows = append(rows, arRow{Text: "{{Y|[ Devise a new " + arSkinFor(g.Genre).Power + " ]}}", Do: func() { st.stage, st.sel, st.picks, st.note = 1, 0, nil, "" },
			Info: []string{"Combine up to three sources with a focus element at a library, workshop, shrine, ley node, old stones or home.", "The same materials and focus always give the same result."}})
		for _, pw := range g.arSortedPowers() {
			pw := pw
			info := []string{arDescribe(pw), fmt.Sprintf("Range %d, cooldown %d, quality %s, refined %d times, cast %d times.", pw.Rng, pw.Cool, arQualNames[pw.Qual], pw.Refined, pw.Uses)}
			if pw.Flaw != "" {
				fl := arFlaws[pw.Flaw]
				info = append(info, fmt.Sprintf("{{R|Flaw: %s}} - %s (backlash check DC %d against Lore or Willpower).", fl.Label, fl.Desc, pw.DC))
			} else {
				info = append(info, "{{G|No flaw.}}")
			}
			rows = append(rows, arRow{Text: fmt.Sprintf("%s {{K|[%s] %s %s, magnitude %d}}", pw.Name, arQualNames[pw.Qual], pw.Elem, pw.Form, pw.Mag),
				Do: func() { st.stage, st.sel, st.cur, st.note = 2, 0, pw.ID, "" }, Info: append(info, "Enter: refine. Used from the ability list as ar_"+pw.ID+".")})
		}
	case 1:
		srcs := g.arSources()
		for _, s := range srcs {
			s := s
			mark := "[ ]"
			for _, k := range st.picks {
				if k == s.Key {
					mark = "{{G|[x]}}"
				}
			}
			rows = append(rows, arRow{Text: fmt.Sprintf("%s %s {{K|(%s, potency %d)}}", mark, s.Label, s.Elem, s.Pot), Do: func() {
				for i, k := range st.picks {
					if k == s.Key {
						st.picks = append(st.picks[:i], st.picks[i+1:]...)
						return
					}
				}
				if len(st.picks) >= 3 {
					st.note = "Three sources at most."
					return
				}
				st.picks = append(st.picks, s.Key)
			}, Info: []string{"Relics and books are studied, not destroyed (a relic loses a charge); ingredients and essences are consumed.", "Elements that match the focus add potency."}})
		}
		if len(srcs) == 0 {
			rows = append(rows, arRow{Text: "{{K|You carry nothing to work from: relics, ingredients, books, essences or powers.}}", Do: func() {}})
		}
		focus := arElems[st.focus%len(arElems)]
		rows = append(rows, arRow{Text: fmt.Sprintf("Focus: {{Y|< %s >}}  {{K|(Enter or arrows to change)}}", arCap(focus)), Focus: true, Do: func() { st.focus = (st.focus + 1) % len(arElems) },
			Info: []string{"The element the working is bent towards. It sets the likely forms: " + arForm4(focus)}})
		sel := g.arSelectedSources()
		begin := "Begin the work"
		info := []string{"Choose at least one source first."}
		if len(sel) > 0 {
			combo := arCombo(sel, focus)
			if id, ok := st.Combos[combo]; ok {
				info = []string{"{{Y|You already know this working: " + st.Powers[id].Name + ".}}"}
			} else {
				pw := g.arDesign(sel, focus)
				kind, label := g.arSite()
				if kind == "" {
					info = []string{"{{R|This is no place for it (" + label + ").}}"}
				} else {
					info = []string{fmt.Sprintf("At %s: %d hours and %s in reagents.", label, g.arHours(sel, pw, kind), g.cuFormat(g.arFee(len(sel), pw.Mag)))}
				}
			}
		}
		rows = append(rows, arRow{Text: "{{G|" + begin + "}}", Info: info, Do: func() {
			s2 := g.arSelectedSources()
			combo := arCombo(s2, focus)
			st.note = g.arDoResearch(s2, focus)
			if _, ok := st.Combos[combo]; ok {
				st.stage, st.sel = 0, 0
			}
		}})
		rows = append(rows, arRow{Text: "{{K|Back}}", Do: func() { st.stage, st.sel = 0, 0 }})
	case 2:
		pw := st.Powers[st.cur]
		if pw == nil {
			st.stage = 0
			return g.arResearchRows()
		}
		kind, _ := g.arSite()
		mul := arSiteMul(kind, pw.Elem)
		if pw.Flaw != "" {
			rows = append(rows, arRow{Text: "Purify: purge the flaw", Do: func() { st.note = g.arRefine(pw, "purify") },
				Info: []string{fmt.Sprintf("%d hours, %s and one essence or ingredient. A Lore check (DC %d) removes the flaw; a failure only eases it.", 12*mul/100, g.cuFormat(40+pw.Mag*10), pw.DC)}})
		}
		if pw.Refined < 3 && pw.Mag < 12 {
			rows = append(rows, arRow{Text: "Intensify: +1 magnitude, quicker recovery", Do: func() { st.note = g.arRefine(pw, "boost") },
				Info: []string{fmt.Sprintf("%d hours, %s and one essence or ingredient. Up to three times.", 16*mul/100, g.cuFormat(60+pw.Mag*15))}})
		}
		rows = append(rows, arRow{Text: "{{K|Back}}", Do: func() { st.stage, st.sel = 0, 0 }})
	}
	return rows
}

func arForm4(focus string) string {
	type kv struct {
		f string
		n int
	}
	var l []kv
	for _, f := range arForms {
		if n := arFormW[focus][f]; n > 0 {
			l = append(l, kv{f, n})
		}
	}
	sort.SliceStable(l, func(i, j int) bool { return l[i].n > l[j].n })
	var out []string
	for i := 0; i < len(l) && i < 4; i++ {
		out = append(out, l[i].f)
	}
	return strings.Join(out, ", ") + "."
}

func (g *Game) arPactRows() []arRow {
	st := g.ar()
	var rows []arRow
	pats := g.arPatrons()
	switch st.stage {
	case 0:
		loc := g.arl()
		for _, pr := range st.Pacts {
			pr := pr
			info := []string{fmt.Sprintf("Terms: %s.", pr.TText), fmt.Sprintf("Standing %d of 100; kept %d days, %d breaches. Gifts are withheld below 20.", pr.Standing, pr.Kept, pr.Breaches)}
			if t := loc.Tasks[pr.PID]; t != nil {
				info = append(info, "{{Y|Task: "+t.Text+"}}")
			}
			rows = append(rows, arRow{Text: fmt.Sprintf("{{M|%s}} {{K|(%s) rank %d, standing %d}}", pr.PName, g.arKindName(pr.PKind), pr.Rank, pr.Standing),
				Do: func() { st.stage, st.sel, st.cur, st.note = 3, 0, pr.PID, "" }, Info: info})
		}
		if len(st.Pacts) < st.maxPacts() {
			rows = append(rows, arRow{Text: "{{Y|[ Seek a " + arSkinFor(g.Genre).Pact + " ]}}", Do: func() { st.stage, st.sel, st.note = 1, 0, "" },
				Info: []string{"A ceremony at a shrine, old stones or a ley node, with an offering of " + g.cuFormat(g.arSealCost()) + ".", "Only one pact at a time, two once a pact reaches rank 3."}})
		}
		rows = append(rows, arRow{Text: "Rite of attunement", Do: func() { st.note = g.arAttune() },
			Info: []string{"Three hours at a holy place or library: for a day research is quicker, workings hit harder and slip less."}})
	case 1:
		for _, pat := range pats {
			pat := pat
			tag := ""
			if st.Grudge[pat.ID] {
				tag = " {{R|(wrathful)}}"
			}
			rows = append(rows, arRow{Text: fmt.Sprintf("{{M|%s}} {{K|%s, domain of %s}}%s", pat.Name, g.arKindName(pat.Kind), pat.Elem, tag),
				Do: func() { st.stage, st.sel, st.cur, st.term = 2, 0, pat.ID, 0 }, Info: append([]string{pat.Lore}, g.arBoonText(pat)...)})
		}
		rows = append(rows, arRow{Text: "{{K|Back}}", Do: func() { st.stage, st.sel = 0, 0 }})
	case 2:
		var pat arPatron
		for _, p := range pats {
			if p.ID == st.cur {
				pat = p
			}
		}
		if pat.ID == "" {
			st.stage = 0
			return g.arPactRows()
		}
		terms := g.arTermsFor(pat)
		for i, tm := range terms {
			i, tm := i, tm
			mark := "( )"
			if st.term == i {
				mark = "{{G|(x)}}"
			}
			rows = append(rows, arRow{Text: fmt.Sprintf("%s Terms: %s", mark, tm.Text), Do: func() { st.term = i },
				Info: []string{"The price of the gifts. Keep it and standing rises; break it and the patron grows angry."}})
		}
		bonus, why := g.arRiteBonus()
		info := []string{fmt.Sprintf("Needs a holy place; an offering of %s; three hours; Willpower or Ego check (DC %d).", g.cuFormat(g.arSealCost()), 9-bonus)}
		if len(why) > 0 {
			info = append(info, "{{G|Favoured by: "+strings.Join(why, ", ")+".}}")
		}
		info = append(info, "The moon is a "+g.arMoonName()+".")
		rows = append(rows, arRow{Text: "{{G|Perform the sealing rite}}", Info: info, Do: func() {
			st.note = g.arSeal(pat, terms[clampi(st.term, 0, len(terms)-1)])
			if g.arPactByID(pat.ID) != nil {
				st.stage, st.sel = 0, 0
			}
		}})
		rows = append(rows, arRow{Text: "{{K|Back}}", Do: func() { st.stage, st.sel = 1, 0 }})
	case 3:
		pr := g.arPactByID(st.cur)
		if pr == nil {
			st.stage = 0
			return g.arPactRows()
		}
		rows = append(rows, arRow{Text: "Make an offering (rite)", Do: func() { st.note = g.arOffering(pr) },
			Info: []string{fmt.Sprintf("%s at a holy place, two hours. Raises standing by 8, more under a full moon, a feast, the turn of a season or on a ley node.", g.cuFormat(40*pr.Rank+g.P.Level*5))}})
		if t := g.arl().Tasks[pr.PID]; t != nil {
			rows = append(rows, arRow{Text: "{{Y|Task:}} " + t.Text, Do: func() {}, Info: []string{fmt.Sprintf("Progress %d of %d. Deadline in %d days.", t.Have, maxi(t.Need, 1), maxi(0, (t.Deadline-g.Turn)/DayLen))}})
			if t.Kind == "offer" {
				rows = append(rows, arRow{Text: "Make the demanded offering", Do: func() { st.note = g.arPayOffering(pr); st.stage = 0 }})
			}
			rows = append(rows, arRow{Text: "{{R|Refuse the task}}", Do: func() { st.note = g.arRefuseTask(pr); st.stage = 0 }, Info: []string{"Costs 12 standing and the patron's temper."}})
		}
		rows = append(rows, arRow{Text: fmt.Sprintf("Leave on fair terms (%s)", g.cuFormat(g.arLeaveCost(pr))), Do: func() { st.note = g.arLeave(pr, true); st.stage, st.sel = 0, 0 },
			Info: []string{"Gifts end. The patron lets you go."}})
		rows = append(rows, arRow{Text: "{{R|Walk away without paying}}", Do: func() { st.note = g.arLeave(pr, false); st.stage, st.sel = 0, 0 },
			Info: []string{"Gifts end, you are marked with a lasting curse and the patron will never hear you again."}})
		rows = append(rows, arRow{Text: "{{K|Back}}", Do: func() { st.stage, st.sel = 0, 0 }})
	}
	return rows
}

func (g *Game) arMasterRows() []arRow {
	loc := g.arl()
	st := g.ar()
	var rows []arRow
	for _, m := range g.arMasters() {
		m := m
		rec := loc.Masters[m.ID]
		met := rec != nil && rec.Met
		if !loc.Heard[m.ID] && !met {
			continue
		}
		status := "a rumour"
		switch {
		case st.Taught[m.ID]:
			status = "{{G|taught you}}"
		case met && rec.Started:
			status = "{{Y|your task is set}}"
		case met:
			status = "met"
		}
		info := []string{m.Story, fmt.Sprintf("Their %s lies %s of here, near %s.", m.Place, g.arWhere(m.P), g.W.RegionName(m.P))}
		if met {
			info = append(info, "Teaches: "+m.Teach.Name+".", "Price: "+m.Req.Text+".")
			if rec != nil && !st.Taught[m.ID] {
				info = append(info, "Progress: "+g.arReqProgress(m, rec)+".")
			}
		} else {
			info = append(info, "What they teach, and at what price, only they will say.")
		}
		rows = append(rows, arRow{Text: fmt.Sprintf("{{M|%s}} {{K|the %s, %s}} - %s", m.Name, strings.ToLower(m.Title), g.arWhere(m.P), status), Do: func() {}, Info: info})
	}
	if len(rows) == 0 {
		rows = append(rows, arRow{Text: "{{K|No leads yet. Masters hide in remote places; townsfolk sometimes speak of them.}}", Do: func() {}})
	}
	return rows
}

func (g *Game) arRows() []arRow {
	switch g.ar().tab {
	case 1:
		return g.arPactRows()
	case 2:
		return g.arMasterRows()
	}
	return g.arResearchRows()
}

func arKeys(g *Game, k string) {
	st := g.ar()
	rows := g.arRows()
	if st.sel >= len(rows) {
		st.sel = maxi(0, len(rows)-1)
	}
	switch k {
	case "Escape":
		if st.stage != 0 {
			if st.tab == 1 && st.stage == 2 {
				st.stage = 1
			} else {
				st.stage = 0
			}
			st.sel = 0
			return
		}
		g.Mode = "main"
	case "ArrowUp":
		st.sel = maxi(0, st.sel-1)
	case "ArrowDown":
		st.sel = mini(maxi(0, len(rows)-1), st.sel+1)
	case "Tab", "ArrowRight", "ArrowLeft":
		if len(rows) > 0 && rows[st.sel].Focus {
			if k == "ArrowLeft" {
				st.focus = (st.focus + len(arElems) - 1) % len(arElems)
			} else {
				st.focus = (st.focus + 1) % len(arElems)
			}
			return
		}
		if st.stage == 0 {
			if k == "ArrowLeft" {
				st.tab = (st.tab + 2) % 3
			} else {
				st.tab = (st.tab + 1) % 3
			}
			st.sel, st.note = 0, ""
		}
	case "Enter", " ":
		if st.sel < len(rows) && rows[st.sel].Do != nil {
			rows[st.sel].Do()
		}
	}
}

func arDraw(g *Game) {
	st := g.ar()
	x, y, w, h := g.panel(arSkinFor(g.Genre).School)
	s := g.sc
	tx := x
	for i, t := range arTabNames {
		c := "K"
		if st.tab == i {
			c = "Y"
		}
		tx = s.TextB(tx, y, fmt.Sprintf("{{%s|[ %s ]}} ", c, t), CGray, CBgDark, x+w)
	}
	_, label := g.arSite()
	bonus, why := g.arRiteBonus()
	line2 := fmt.Sprintf("{{C|Here: %s}}  {{K|%s, day %d}}", label, g.arMoonName(), g.Day()%SeasonDays+1)
	if bonus > 0 {
		line2 += "  {{G|rites favoured: " + strings.Join(why, ", ") + "}}"
	}
	s.TextB(x, y+1, line2, CGray, CBgDark, x+w)
	rows := g.arRows()
	if st.sel >= len(rows) {
		st.sel = maxi(0, len(rows)-1)
	}
	n := maxi(3, h-13)
	top := maxi(0, st.sel-n+1)
	for i := top; i < len(rows) && i-top < n; i++ {
		pre := "  "
		if i == st.sel {
			pre = "{{Y|> }}"
		}
		s.TextB(x, y+3+i-top, pre+rows[i].Text, CGray, CBgDark, x+w)
	}
	iy := y + 3 + n + 1
	if st.sel < len(rows) {
		for _, l := range rows[st.sel].Info {
			for _, wl := range Wrap(l, w-2) {
				if iy < y+h-3 {
					s.TextB(x, iy, wl, CGray, CBgDark, x+w)
					iy++
				}
			}
		}
	}
	if st.note != "" {
		s.TextB(x, y+h-3, "{{W|"+st.note+"}}", CGray, CBgDark, x+w)
	}
	s.TextB(x, y+h-1, "{{K|arrows: move and switch tabs   Enter: choose   Esc: back}}", CGray, CBgDark, x+w)
}
