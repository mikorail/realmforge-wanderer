package engine

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// Crossroads (prefix cr): Reigns-style dilemma cards. Templates (crossroads_data.go) are filled from real world data
// and picked by hash of (seed, day bucket, zone key, trigger), never g.R. Choices carry requirements, checks and
// effect strings; outcomes stack and queue follow-up cards days later. Karma and vows: karma.go.
// State g.ext("cr") is planet-local ("cr" in unLocalKeys). Card screen: mode `crd`.

type crPend struct {
	ID  string            `json:"id"`
	Due int               `json:"d"`
	Ctx map[string]string `json:"c"`
}

type crCur struct {
	TID  string            `json:"t"`
	Ctx  map[string]string `json:"c"`
	Back string            `json:"b"`
}

type crState struct {
	Last   int            `json:"l"`
	LastFu int            `json:"lf"`
	Seen   map[string]int `json:"s,omitempty"`
	Pend   []crPend       `json:"p,omitempty"`
	Cur    *crCur         `json:"cu,omitempty"`
	N      int            `json:"n"`
	Want   string         `json:"w,omitempty"`
	Amb    int            `json:"a,omitempty"`

	phase int
	res   []string
}

func crSt(g *Game) *crState {
	s := g.ext("cr", func() any { return &crState{} }).(*crState)
	if s.Seen == nil {
		s.Seen = map[string]int{}
	}
	return s
}

func init() {
	turnHooks = append(turnHooks, crTurn)
	enterHooks = append(enterHooks, crEnter)
	modeKeys["crd"] = crdKeys
	modeDraws["crd"] = func(g *Game) { g.crDrawCard() }
}

type crWhere struct {
	WX, WY, Burg int
	Key          string
}

func (g *Game) crHere() crWhere {
	if g.OnWorld || g.Cur == nil {
		return crWhere{g.WX, g.WY, 0, fmt.Sprintf("w%d.%d", g.WX, g.WY)}
	}
	return crWhere{g.Cur.WX, g.Cur.WY, g.Cur.Burg, g.Cur.Key}
}

var crLoaf = map[string]string{"mythic": "honey-cake", "steampunk": "meat pie", "dieselpunk": "ration tin", "atompunk": "can of beans", "cyberpunk": "protein brick",
	"biopunk": "vat-loaf", "solarpunk": "seed-loaf", "wasteland": "tin of scrap-meat", "twilight": "stale crust"}

// crBuild fills the placeholder context from the real world around w.
func (g *Game) crBuild(w crWhere, r *Rng) map[string]string {
	c := map[string]string{}
	if !g.W.In(w.WX, w.WY) {
		return c
	}
	p := g.W.Idx(w.WX, w.WY)
	bd := g.W.burgByID[w.Burg]
	if bd == nil {
		bd = g.W.burgByID[g.W.Burg[p]]
	}
	state := g.W.State[p]
	if bd != nil {
		state = bd.State
		c["T"], c["Tn"], c["Bid"] = bd.Name, "1", strconv.Itoa(bd.ID)
	}
	// nearest other town
	best, bn := 1<<30, ""
	for i := range g.W.Burgs {
		b := &g.W.Burgs[i]
		if !b.Alive || (bd != nil && b.ID == bd.ID) {
			continue
		}
		d := abs(b.P%g.W.W-w.WX) + abs(b.P/g.W.W-w.WY)
		if d < best || d == best && b.Name < bn {
			best, bn = d, b.Name
		}
	}
	c["N"] = bn
	if c["T"] == "" {
		c["T"] = bn
	}
	c["Rid"] = strconv.Itoa(state)
	if st := g.W.stateByID[state]; st != nil && state > 0 {
		c["R"], c["Ruler"], c["I"] = st.Name, st.Ruler, st.Ideology
		if st.Famine != nil {
			c["Fam"] = "1"
		}
		if len(st.Wars) > 0 {
			if e := g.W.stateByID[st.Wars[0]]; e != nil {
				c["W"] = e.Name
			}
		}
		if st.Reforms != nil && len(st.Reforms.Movements) > 0 {
			c["Mv"] = st.Reforms.Movements[0].Name
		}
		if st.Religion > 0 {
			c["Fid"] = strconv.Itoa(st.Religion)
		}
	}
	if rl := g.W.Religion[p]; rl > 0 {
		c["Fid"] = strconv.Itoa(rl)
	}
	if id, _ := strconv.Atoi(c["Fid"]); id > 0 {
		if rel := g.W.relByID[id]; rel != nil {
			c["F"], c["D"] = rel.Name, rel.Deity
		}
	}
	if h := g.W.Hazards; h != nil && len(h.Outbreaks) > 0 {
		o := h.Outbreaks[0]
		for _, x := range h.Outbreaks {
			if hzHas(x.Parasangs, p) {
				o = x
				break
			}
		}
		c["H"] = o.Name
	}
	if bd != nil {
		if f := g.W.FeastOn(bd, g.Day()); f != nil {
			c["X"] = f.Name
		} else if f, d := g.W.nextFeast(bd, g.Day()); f != nil && d < 20 {
			c["X"] = f.Name
		}
		lvEnsure(g, bd.ID)
		ps := lvPresent(g, bd.ID)
		sort.Slice(ps, func(i, j int) bool { return ps[i].Key < ps[j].Key })
		if len(ps) > 0 {
			pp := ps[r.Int(0, len(ps)-1)]
			c["P"], c["PK"], c["Pr"], c["PS"] = pp.Name, pp.Key, pp.Role, pp.Secret
			var oth *lvPerson
			for _, k := range pp.Rivals {
				if q := lvGet(g, k); q != nil && q.Alive {
					oth = q
					break
				}
			}
			if oth == nil && len(ps) > 1 {
				for _, q := range ps {
					if q.Key != pp.Key {
						oth = q
						break
					}
				}
			}
			if oth != nil {
				c["O"], c["OK"] = oth.Name, oth.Key
			}
		}
	}
	for _, id := range g.W.monstersAt[p] {
		if id >= 0 && id < len(g.W.Monsters) {
			c["M"] = g.W.Monsters[id].Name
			break
		}
	}
	if g.Cur != nil {
		for _, a := range g.Cur.Actors {
			if isFollower(a) {
				c["C"] = a.Name
				break
			}
		}
	}
	c["loaf"] = nonEmptyCr(crLoaf[g.Genre], "loaf of bread")
	c["guard"], c["priest"], c["elder"] = g.roleTitle("guard"), g.roleTitle("priest"), g.roleTitle("elder")
	c["merchant"], c["doc"], c["smith"] = g.roleTitle("merchant"), g.roleTitle("apothecary"), g.roleTitle("smith")
	c["coin"], c["bandit"], c["undead"] = g.Sk.Currency, g.Sk.Bandits, strings.TrimPrefix(g.Sk.Undead, "the ")
	for k, v := range map[string]string{"T": "town", "N": "the next town", "R": "the realm", "F": "the faith", "D": "the gods", "H": "the sickness", "X": "the feast",
		"M": "a beast", "C": "your companion", "P": "a local", "O": "a rival", "W": "the enemy", "Ruler": "the ruler", "I": "the regime", "Mv": "the reformers", "PS": "something shameful"} {
		if c[k] == "" {
			c["~"+k] = "1"
			c[k] = v
		}
	}
	return c
}

var crNeedKey = map[string]string{"person": "P", "rival": "O", "plague": "H", "famine": "Fam", "war": "W", "faith": "F", "feast": "X", "monster": "M",
	"move": "Mv", "realm": "R", "secret": "PS", "comp": "C", "town": "Tn"}

func crNeedsOK(t *crTpl, c map[string]string) bool {
	if t.Need == "" {
		return true
	}
	for _, n := range strings.Split(t.Need, ",") {
		k := crNeedKey[n]
		if c[k] == "" || c["~"+k] != "" {
			return false
		}
	}
	return true
}

func crFill(s string, c map[string]string) string {
	if !strings.Contains(s, "{") {
		return s
	}
	for k, v := range c {
		if k[0] != '~' {
			s = strings.ReplaceAll(s, "{"+k+"}", v)
		}
	}
	return s
}

func crTplByID(id string) *crTpl {
	for i := range crTpls {
		if crTpls[i].ID == id {
			return &crTpls[i]
		}
	}
	return nil
}

// ---- triggers ----

func (g *Game) crQuiet() bool {
	if g.P == nil || g.Dead || g.Cur == nil || g.W == nil {
		return false
	}
	if g.Mode != "main" && g.Mode != "world" {
		return false
	}
	if !g.OnWorld && g.nearHostile() {
		return false
	}
	return true
}

func (g *Game) crTry(trig string, force bool) bool {
	s := crSt(g)
	if !g.crQuiet() || s.Cur != nil {
		return false
	}
	if s.Last > 0 && g.Turn < s.Last-1+2*DayLen {
		return false
	}
	w := g.crHere()
	day := g.Day()
	r := NewRng(fmt.Sprintf("%s:cr:%d:%s:%s", g.Seed, day/2, w.Key, trig))
	pr := map[string]float64{"t": 0.55, "r": 0.45, "c": 0.4, "d": 0.9}[trig]
	if !force && !r.Chance(pr) {
		return false
	}
	c := g.crBuild(w, r)
	var cand []*crTpl
	tot := 0
	for i := range crTpls {
		t := &crTpls[i]
		if t.Fu || !strings.Contains(t.Trig, trig) || t.Gn != "" && t.Gn != g.Genre || !crNeedsOK(t, c) {
			continue
		}
		if d, ok := s.Seen[t.ID]; ok && day-d < 14 {
			continue
		}
		cand = append(cand, t)
		tot += 1 + 2*boolInt(t.Gn != "")
	}
	if len(cand) == 0 {
		return false
	}
	x := r.Int(0, tot-1)
	for _, t := range cand {
		x -= 1 + 2*boolInt(t.Gn != "")
		if x < 0 {
			g.crOpen(t, c)
			return true
		}
	}
	return false
}

func crEnter(g *Game, zn *Zone) {
	if g.P == nil || zn == nil {
		return
	}
	s := crSt(g)
	if zn.Kind == "town" && zn.Burg > 0 {
		s.Want = "t"
	}
	if s.Amb > 0 && !g.OnWorld {
		n := s.Amb
		s.Amb = 0
		r := NewRng(fmt.Sprintf("%s:cr:amb:%d", g.Seed, g.Turn))
		for i := 0; i < n; i++ {
			a := g.Spawn("bandit", maxi(1, zn.Level), r)
			a.X, a.Y = g.P.X+r.Int(-6, 6), g.P.Y+r.Int(-3, 3)
			g.nudge(zn, a)
			zn.Actors = append(zn.Actors, a)
		}
		g.Msg("{{R|Armed strangers close in. You have been followed.}}")
	}
}

// crOnMove: call at the end of WorldMove. A card may open at a road junction.
func crOnMove(g *Game) {
	if g.W == nil || !g.W.In(g.WX, g.WY) || g.W.IsWater(g.W.Idx(g.WX, g.WY)) {
		return
	}
	p := g.W.Idx(g.WX, g.WY)
	if g.W.Road[p] <= 0 {
		return
	}
	n := 0
	for _, d := range dirs8 {
		if g.W.In(g.WX+d[0], g.WY+d[1]) && g.W.Road[g.W.Idx(g.WX+d[0], g.WY+d[1])] > 0 {
			n++
		}
	}
	if n < 3 {
		return
	}
	if crSt(g).Cur == nil && g.crQuiet() && g.Mode == "world" {
		g.crTry("r", false)
	}
}

func crTurn(g *Game) {
	if g.P == nil || g.W == nil || g.Cur == nil || g.Dead {
		return
	}
	s := crSt(g)
	if s.Cur != nil {
		if g.crQuiet() {
			if t := crTplByID(s.Cur.TID); t != nil {
				g.crShow()
			} else {
				s.Cur = nil
			}
		}
		return
	}
	if !g.crQuiet() {
		return
	}
	day := g.Day()
	if len(s.Pend) > 0 && g.Turn-s.LastFu >= 360 {
		sort.SliceStable(s.Pend, func(i, j int) bool { return s.Pend[i].Due < s.Pend[j].Due })
		if s.Pend[0].Due <= day {
			pd := s.Pend[0]
			s.Pend = s.Pend[1:]
			if t := crTplByID(pd.ID); t != nil {
				s.LastFu = g.Turn
				g.crOpen(t, pd.Ctx)
			}
			return
		}
	}
	if s.Want != "" {
		w := s.Want
		s.Want = ""
		g.crTry(w, false)
		return
	}
	k := crKs(g)
	if n := len(g.Deeds); k.Snap["crdeeds"] == 0 {
		k.Snap["crdeeds"] = n + 1
	} else if n+1 > k.Snap["crdeeds"] {
		big := false
		for i := k.Snap["crdeeds"] - 1; i < n; i++ {
			switch g.Deeds[i].Kind {
			case "slay", "relic", "quest", "endgame", "murder":
				big = true
			}
		}
		k.Snap["crdeeds"] = n + 1
		if big {
			g.crTry("d", false)
			return
		}
	}
	if h := g.Hour(); (h >= 20 || h < 5) && g.Turn%45 == 0 && g.Cur.Z == 0 && g.Cur.Kind != "town" && outdoors(g.Cur) {
		g.crTry("c", false)
	}
}

// ---- opening and showing ----

func (g *Game) crOpen(t *crTpl, c map[string]string) {
	s := crSt(g)
	back := "main"
	if g.OnWorld || g.Mode == "world" {
		back = "world"
	}
	s.Cur = &crCur{TID: t.ID, Ctx: c, Back: back}
	s.Seen[t.ID] = g.Day()
	s.Last = g.Turn + 1
	s.N++
	g.crShow()
	if s.N == 1 {
		g.crFirst("card1", fmt.Sprintf("%s came to a first crossroads of choice.", g.P.Name))
	}
}

func (g *Game) crShow() {
	s := crSt(g)
	s.phase, s.res = 0, nil
	g.Mode, g.Sel = "crd", 0
}

type crOpt struct {
	label string
	ok    bool
	why   string
}

// crReqOK checks requirement tokens; returns false and a short reason.
func (g *Game) crReqOK(req string, c map[string]string) (bool, string) {
	if req == "" {
		return true, ""
	}
	for _, tk := range strings.Split(req, ",") {
		f := strings.Split(tk, ":")
		switch f[0] {
		case "skill":
			n, _ := strconv.Atoi(f[2])
			if g.Skills[f[1]] < n {
				return false, "needs " + f[1] + " " + f[2]
			}
		case "stat":
			n, _ := strconv.Atoi(f[2])
			if g.P.Score(crStatIdx(f[1])) < n {
				return false, "needs " + f[1] + " " + f[2]
			}
		case "gold":
			n, _ := strconv.Atoi(f[1])
			if g.P.Gold < n {
				return false, fmt.Sprintf("needs %d %s", n, g.Sk.Currency)
			}
		case "item":
			ok := false
			for _, it := range g.P.Inv {
				if it.Kind == f[1] {
					ok = true
				}
			}
			if !ok {
				return false, "needs " + f[1]
			}
		case "comp":
			if c["~C"] != "" || c["C"] == "" {
				return false, "needs a companion"
			}
		case "karma":
			n, _ := strconv.Atoi(f[2])
			v := g.crTrait(crAxisKeys[f[1]])
			if n > 0 && v < n || n < 0 && v > n {
				return false, "needs " + crAxisNames[crAxisKeys[f[1]]][boolInt(n < 0)] + " " + strconv.Itoa(abs(n))
			}
		case "rep":
			n, _ := strconv.Atoi(f[2])
			key := crRepKey(f[1], c)
			if key != "" {
				v := g.Rep[key]
				if n > 0 && v < n || n < 0 && v > n {
					return false, "needs standing"
				}
			}
		}
	}
	return true, ""
}

func crStatIdx(s string) int {
	return map[string]int{"STR": STR, "AGI": AGI, "TOU": TOU, "INT": INT, "WIL": WIL, "EGO": EGO}[s]
}

func crRepKey(k string, c map[string]string) string {
	switch k {
	case "realm":
		if c["Rid"] != "" && c["Rid"] != "0" {
			return "state:" + c["Rid"]
		}
	case "faith":
		if c["Fid"] != "" {
			return "faith:" + c["Fid"]
		}
	case "out":
		return "outlaws"
	}
	return ""
}

// crHint summarises costs and risks from the effect strings.
func (g *Game) crHint(ch *crCh) string {
	var parts []string
	scan := func(eff, pre string) {
		for _, tk := range strings.Split(eff, ";") {
			f := strings.Split(tk, ":")
			if len(f) < 2 {
				continue
			}
			n, _ := strconv.Atoi(f[1])
			switch f[0] {
			case "gold":
				if n < 0 {
					parts = append(parts, fmt.Sprintf("%s%d %s", pre, n, g.Sk.Currency))
				}
			case "hp":
				if n < 0 {
					parts = append(parts, fmt.Sprintf("%s%d HP", pre, n))
				}
			case "time":
				parts = append(parts, fmt.Sprintf("%s%dh", pre, maxi(1, n/60)))
			case "bounty":
				parts = append(parts, pre+"bounty")
			case "spawn":
				parts = append(parts, pre+"a fight")
			case "lose":
				parts = append(parts, pre+"a "+f[1])
			}
		}
	}
	if ch.Chk == "" {
		scan(ch.Win, "")
	} else {
		scan(ch.Win, "")
		scan(ch.Lose, "risk ")
	}
	if len(parts) == 0 {
		return ""
	}
	return " {{K|(" + strings.Join(parts, ", ") + ")}}"
}

func (g *Game) crOptions() (*crTpl, []crOpt) {
	s := crSt(g)
	t := crTplByID(s.Cur.TID)
	var out []crOpt
	for i := range t.Ch {
		ch := &t.Ch[i]
		ok, why := g.crReqOK(ch.Req, s.Cur.Ctx)
		if strings.Contains(ch.Fl, "lie") && g.crVow("tr") != nil {
			ok, why = false, "vow of Truth"
		}
		if strings.Contains(ch.Fl, "tell") && g.crVow("si") != nil {
			ok, why = false, "vow of Silence"
		}
		lab := crFill(ch.T, s.Cur.Ctx)
		if ch.Chk != "" {
			f := strings.Split(ch.Chk, ":")
			dc, _ := strconv.Atoi(f[1])
			lab = g.checkLabel(f[0], dc) + " " + lab
		}
		lab += g.crHint(ch)
		if ch.Req != "" {
			lab += " {{K|[" + strings.ReplaceAll(strings.ReplaceAll(ch.Req, ",", ", "), ":", " ") + "]}}"
		}
		out = append(out, crOpt{lab, ok, why})
	}
	out = append(out, crOpt{"{{K|Decline to decide.}}", true, ""})
	return t, out
}

func crdKeys(g *Game, k string) {
	s := crSt(g)
	if s.Cur == nil {
		g.Mode = "main"
		return
	}
	if s.phase == 1 {
		if k == "Enter" || k == " " || k == "Escape" {
			g.Mode = s.Cur.Back
			s.Cur = nil
			if g.Mode == "world" {
				g.OnWorld = true
			}
		}
		return
	}
	t, opts := g.crOptions()
	pick := func(i int) {
		if i < 0 || i >= len(opts) {
			return
		}
		if !opts[i].ok {
			g.Msg("{{K|You cannot choose that: " + opts[i].why + ".}}")
			return
		}
		g.crChoose(t, i)
	}
	if i := letterIndex(k); i >= 0 {
		pick(i)
		return
	}
	switch k {
	case "ArrowUp":
		g.Sel = maxi(0, g.Sel-1)
	case "ArrowDown":
		g.Sel = mini(len(opts)-1, g.Sel+1)
	case "Enter", " ":
		pick(g.Sel)
	}
}

func (g *Game) crChoose(t *crTpl, i int) {
	s := crSt(g)
	c := s.Cur.Ctx
	var out, eff string
	var ch *crCh
	crDelta = [5]int{}
	if i >= len(t.Ch) {
		out = "You walk on without deciding. Silence is also an answer."
		eff = nonEmptyCr(t.Dec, "cou:-2")
		g.crFirst("decline", fmt.Sprintf("%s refused to decide.", g.P.Name))
		switch t.Th {
		case "mercy", "family", "plague", "war", "famine":
			g.crRefuse()
		}
	} else {
		ch = &t.Ch[i]
		eff, out = ch.Win, ch.WT
		if ch.Chk != "" {
			f := strings.Split(ch.Chk, ":")
			dc, _ := strconv.Atoi(f[1])
			if g.roll2d6Check(g.checkMod(f[0]), dc) {
				g.gainXP(4 + dc)
				out = "{{G|[" + checkKinds[f[0]].label + ": Success]}} " + ch.WT
			} else {
				eff, out = ch.Lose, "{{R|["+checkKinds[f[0]].label+": Failure]}} "+ch.LT
			}
		}
	}
	lines := g.crApply(eff, c)
	if ch != nil {
		if crDelta[0] <= -4 || crDelta[3] <= -4 {
			g.crRefuse()
		} else if crDelta[0] >= 3 || crDelta[3] >= 3 {
			g.crAid()
		}
	}
	s.Cur.Ctx = c
	hist := fmt.Sprintf("{{Y|%s}}: ", crFill(t.Title, c))
	if ch != nil {
		hist += crFill(ch.T, c)
	} else {
		hist += "declined"
	}
	k := crKs(g)
	k.Hist = append(k.Hist, crLog{g.Day(), hist})
	if len(k.Hist) > 40 {
		k.Hist = k.Hist[len(k.Hist)-40:]
	}
	s.res = append(Wrap(crFill(out, c), 96), "")
	for _, l := range lines {
		s.res = append(s.res, l)
	}
	s.phase = 1
}

var crDelta [5]int

func crSign(n int) string { return fmt.Sprintf("%+d", n) }

// crApply runs an effect string and returns display lines.
func (g *Game) crApply(eff string, c map[string]string) []string {
	var lines []string
	if eff == "" {
		return nil
	}
	day := g.Day()
	r := NewRng(fmt.Sprintf("%s:cr:fx:%d:%d", g.Seed, g.Turn, g.Day()))
	for _, tk := range strings.Split(eff, ";") {
		tk = strings.TrimSpace(tk)
		if tk == "" {
			continue
		}
		f := strings.SplitN(tk, ":", 3)
		num := func(i int) int {
			if i < len(f) {
				n, _ := strconv.Atoi(strings.TrimPrefix(f[i], "+"))
				return n
			}
			return 0
		}
		if ax, ok := crAxisKeys[f[0]]; ok {
			n := num(1)
			crDelta[ax] += n
			g.crKarma(ax, n, crFill("a choice about {T}", c))
			lines = append(lines, fmt.Sprintf("{{K|%s %s}}", crAxisNames[ax][boolInt(n < 0)], crSign(n)))
			continue
		}
		switch f[0] {
		case "gold":
			n := num(1)
			if n < 0 {
				n = -mini(-n, g.P.Gold)
			}
			g.P.Gold += n
			lines = append(lines, fmt.Sprintf("{{y|%s %s}}", crSign(n), g.Sk.Currency))
		case "hp":
			n := num(1)
			if n < 0 {
				g.hurt(g.P, -n, nil, "a hard choice")
			} else {
				g.P.HP = mini(g.P.MaxHP, g.P.HP+n)
			}
			lines = append(lines, fmt.Sprintf("{{r|%s HP}}", crSign(n)))
		case "time":
			g.Turn += num(1)
			lines = append(lines, fmt.Sprintf("{{K|%d hours pass}}", maxi(1, num(1)/60)))
		case "rep":
			if k := crRepKey(f[1], c); k != "" {
				g.AddRep(k, num(2), "")
			} else if f[1] == "home" && g.HomeState > 0 {
				g.AddRep(fmt.Sprintf("state:%d", g.HomeState), num(2), "")
			}
		case "fav":
			if id, _ := strconv.Atoi(c["Fid"]); id > 0 {
				g.gdAddFav(id, num(1), "")
				lines = append(lines, fmt.Sprintf("{{C|%s favour %s}}", c["D"], crSign(num(1))))
			}
		case "op":
			key := c["PK"]
			if f[1] == "O" {
				key = c["OK"]
			}
			if p := lvGet(g, key); p != nil {
				n := num(2)
				kind := "help"
				if n < 0 {
					kind = "insult"
				}
				g.lvMemoryAdd(p, kind, "You made a choice that touched them.", n)
				lines = append(lines, fmt.Sprintf("{{K|%s thinks %s of you}}", p.Name, map[bool]string{true: "better", false: "worse"}[n > 0]))
			}
		case "loy":
			for _, a := range g.localCompanions() {
				g.addLoyalty(a, num(1))
			}
			lines = append(lines, fmt.Sprintf("{{K|Companion loyalty %s}}", crSign(num(1))))
		case "bounty":
			if st, _ := strconv.Atoi(c["Rid"]); st > 0 && !g.OnWorld {
				g.uwAddBounty(st, num(1), " from a rash choice")
			}
		case "item":
			var it *Item
			switch f[1] {
			case "food":
				it = g.MakeFood(r, 1)
			case "tonic":
				it = g.MakeTonic("heal", 1)
			case "loot":
				it = g.Loot(r, 1+g.P.Level/4)
			case "oddity":
				it = g.MakeOddity(r, 1+g.P.Level/5)
			case "scrap":
				it = g.MakeScrap(r, 2)
			case "weapon":
				it = g.MakeWeapon("shortblade", 1+g.P.Level/4, r)
			case "book":
				it = g.MakeBook("A stranger's ledger", crFill("Kept by {P} of {T}: names, debts and a few things nobody was meant to read.", c))
			}
			if it != nil {
				g.give(it)
				lines = append(lines, "{{G|You gain "+Plain(it.Display())+"}}")
			}
		case "lose":
			for i, it := range g.P.Inv {
				if it.Kind == f[1] {
					if it.Count > 1 {
						it.Count--
					} else {
						g.P.Inv = append(g.P.Inv[:i], g.P.Inv[i+1:]...)
					}
					lines = append(lines, "{{r|You give up a "+f[1]+"}}")
					break
				}
			}
		case "xp":
			g.gainXP(num(1))
		case "news":
			bid, _ := strconv.Atoi(c["Bid"])
			lvSt(g).addNews(day, bid, "%s", Plain(crFill(strings.Join(f[1:], ":"), c)))
		case "deed":
			if g.Cur != nil {
				g.deed("event", crFill(strings.Join(f[1:], ":"), c), 0, 0)
			}
		case "spawn":
			crSt(g).Amb += maxi(1, num(2))
			lines = append(lines, "{{R|Trouble will find you.}}")
		case "fu":
			cc := map[string]string{}
			for k, v := range c {
				cc[k] = v
			}
			d := num(2) + int(HashStr(fmt.Sprint(g.Seed, f[1], g.Turn))%3)
			crSt(g).Pend = append(crSt(g).Pend, crPend{f[1], day + d, cc})
		}
	}
	return lines
}

func (g *Game) crDrawCard() {
	sc := g.sc
	s := crSt(g)
	if s.Cur == nil {
		return
	}
	t := crTplByID(s.Cur.TID)
	if t == nil {
		return
	}
	c := s.Cur.Ctx
	x, y, w, _ := g.panel("Crossroads")
	sc.TextB(x, y, "{{M|"+crFill(t.Title, c)+"}}", CGray, CBgDark, x+w)
	y += 2
	if s.phase == 1 {
		for _, l := range s.res {
			sc.TextB(x, y, l, CGray, CBgDark, x+w)
			y++
		}
		y++
		sc.TextB(x, y, "{{K|Enter to go on.}}", CGray, CBgDark, x+w)
		return
	}
	for _, l := range Wrap(crFill(t.Text, c), w-2) {
		sc.TextB(x, y, l, CGray, CBgDark, x+w)
		y++
	}
	y++
	_, opts := g.crOptions()
	g.Sel = clampi(g.Sel, 0, len(opts)-1)
	for i, o := range opts {
		bg := CBgDark
		if i == g.Sel {
			bg = CBgSel
		}
		col := "W"
		if !o.ok {
			col = "K"
		}
		ls := Wrap(fmt.Sprintf("{{%s|%c)}} %s", col, 'a'+i, o.label), w-4)
		if !o.ok {
			ls = Wrap(fmt.Sprintf("{{K|%c) %s (%s)}}", 'a'+i, Plain(o.label), o.why), w-4)
		}
		for j, l := range ls {
			sc.Fill(x, y, w, 1, bg)
			if j > 0 {
				l = "   " + l
			}
			sc.TextB(x, y, l, CGray, bg, x+w)
			y++
		}
	}
	sc.TextB(x, y+1, "{{K|Letter or Enter to choose. The consequences stack.}}", CGray, CBgDark, x+w)
}
