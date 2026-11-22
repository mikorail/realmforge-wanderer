package engine

// Town reputation board (prefix rb). What a town says about the wanderer, built only from what the wanderer
// really did: the deeds list (kills of great beasts, quests, rescues, crimes), karma axes (karma.go), standing
// with the realm and the faith, any bounty on their head, and the count of the dead. State g.ext("rb"),
// planet-local.
//
//   board      a reputation board on a wall of each town (placed by the zone hook with a side rng
//              seed+":rb:"+zoneKey, always on a wall so it never blocks a path); its text is refreshed on entry
//   graffiti   a scrawl on another wall, plus a children's skipping rhyme heard in the street, after a big deed
//   signs      an inn (and, after much, a shop) sign repainted if the wanderer saved or ruined the town; the
//              new and old names are stored in g.ext("rb")
//   bard       a talk option on bards, and once per town a song that mentions the wanderer, when fame is high
//   screen     mode `rbs` (palette: "What do they say of me?"): per visited town, standing, best and worst line
//
// The rumour spreads: a deed done at parasang A is known in a town at distance d (parasangs) from A only after
// 2 days per parasang and only within a radius that grows with the deed's weight; the strength fades with
// distance. Nothing is stored for this, it is recomputed from the deed list and the clock. All text comes from
// templates; culture pack proverbs are added when the culture has a pack.

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

func init() {
	unLocalKeys = append(unLocalKeys, "rb")
	zoneHooks = append(zoneHooks, rbZone)
	enterHooks = append(enterHooks, rbEnter)
	turnHooks = append(turnHooks, rbTurn)
	killHooks = append(killHooks, rbKill)
	talkHooks = append(talkHooks, rbTalk)
	modeKeys["rbs"] = rbKeys
	modeDraws["rbs"] = func(g *Game) { g.rbDraw() }
	paEntries = append(paEntries, paEntry{Name: "What do they say of me?", Keywords: "reputation towns rumour gossip fame notices board name standing", Cat: "Screen",
		Func: func(g *Game) { g.Mode, g.Sel = "rbs", 0 }})
}

// ---------------------------------------------------------------- state

type rbDeed struct {
	Day  int    `json:"d"`
	P    int    `json:"p"`           // parasang where it happened
	Town int    `json:"t,omitempty"` // burg it happened in or beside
	Kind string `json:"k"`
	W    int    `json:"w"` // weight: + admired, - despised
	Do   string `json:"o"` // "slew the ash wyrm"
	Noun string `json:"n"`
}

type rbSign struct {
	Zone string `json:"z"`
	Tile int    `json:"t"`
}

type rbNames struct {
	Mood    int    `json:"m"` // 1 saved, -1 ruined
	Inn     string `json:"i"`
	InnOld  string `json:"io"`
	Shop    string `json:"s,omitempty"`
	ShopOld string `json:"so,omitempty"`
}

type rbState struct {
	Cur    int                `json:"c"`
	Deeds  []rbDeed           `json:"d,omitempty"`
	Kills  map[int]int        `json:"kl,omitempty"` // burg -> townsfolk the wanderer killed there
	Seen   map[int]int        `json:"sn,omitempty"` // burg -> last day the wanderer was there
	Board  map[int]string     `json:"b,omitempty"`  // burg -> zone key of its board
	BoardT map[string]int     `json:"bt,omitempty"` // zone key -> board tile
	Graf   map[string]int     `json:"gr,omitempty"` // zone key -> graffiti tile
	Signs  map[string]*rbSign `json:"sg,omitempty"` // "burg:inn" / "burg:shop"
	Names  map[int]*rbNames   `json:"nm,omitempty"`
	Sung   map[int]int        `json:"su,omitempty"` // burg -> day the bard first sang
	Rhyme  map[int]int        `json:"rh,omitempty"` // burg -> day children last chanted
	Last   string             `json:"l,omitempty"`
}

func (g *Game) rb() *rbState {
	s := g.ext("rb", func() any { return &rbState{} }).(*rbState)
	if s.Kills == nil {
		s.Kills = map[int]int{}
	}
	if s.Seen == nil {
		s.Seen = map[int]int{}
	}
	if s.Board == nil {
		s.Board = map[int]string{}
	}
	if s.BoardT == nil {
		s.BoardT = map[string]int{}
	}
	if s.Graf == nil {
		s.Graf = map[string]int{}
	}
	if s.Signs == nil {
		s.Signs = map[string]*rbSign{}
	}
	if s.Names == nil {
		s.Names = map[int]*rbNames{}
	}
	if s.Sung == nil {
		s.Sung = map[int]int{}
	}
	if s.Rhyme == nil {
		s.Rhyme = map[int]int{}
	}
	return s
}

func rbHash(g *Game, parts ...string) uint64 {
	return HashStr(g.Seed + ":rb:" + strings.Join(parts, ":"))
}

func rbPick(g *Game, list []string, parts ...string) string {
	if len(list) == 0 {
		return ""
	}
	return list[int(rbHash(g, parts...)%uint64(len(list)))]
}

// ---------------------------------------------------------------- reading deeds

var (
	rbSlew   = regexp.MustCompile(`slew (?:the )?(?:legendary )?(.+?)(?:,| at |\.|$)`)
	rbDown   = regexp.MustCompile(`brought down (.+?),`)
	rbHelp   = regexp.MustCompile(`helped at (.+?):`)
	rbRelic  = regexp.MustCompile(`relic (.+?)(?: from | to | for |\.|$)`)
	rbBounty = regexp.MustCompile(`bounty on (.+?) \(`)
	rbProper = regexp.MustCompile(`(?:for|of|at|to|in) ([A-Z][^,.:(]*)`)
	rbTitle  = regexp.MustCompile(`earned the name (.+?):`)
	rbVictim = regexp.MustCompile(`^(.+?) was slain`)
)

// rbRest is what the deed says the wanderer did ("slew the ash wyrm"), cut after their name.
func (g *Game) rbRest(text string) (string, bool) {
	me := g.P.Name
	i := strings.Index(text, me)
	if i < 0 || me == "" {
		return "", false
	}
	r := strings.TrimSpace(text[i+len(me):])
	if strings.HasPrefix(r, "'s") {
		return "", false
	}
	if g.Sk != nil {
		if c := g.Sk.Calling(g.Calling); c != "" && strings.HasPrefix(r, "the "+c+" ") {
			r = r[len("the "+c)+1:]
		}
	}
	r = strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(r), "."))
	if r == "" || r[0] < 'a' || r[0] > 'z' {
		return "", false
	}
	if len(r) > 90 {
		r = r[:90]
	}
	return r, true
}

// rbClassify turns a deed into a weight, a phrase and a noun. Weight 0 means the town would not talk of it.
func (g *Game) rbClassify(d Deed) (w int, do, noun string) {
	t := d.Text
	low := strings.ToLower(t)
	rest, ok := g.rbRest(t)
	pick := func(re *regexp.Regexp, def string) string {
		if m := re.FindStringSubmatch(t); m != nil {
			return strings.TrimSpace(m[1])
		}
		return def
	}
	other := func(def string) string {
		if m := rbProper.FindStringSubmatch(t); m != nil {
			return strings.TrimSpace(m[1])
		}
		return def
	}
	switch d.Kind {
	case "slay":
		noun = pick(rbSlew, "a beast")
		w = 3
		if strings.Contains(low, "legendary") {
			w = 6
		}
		return w, "slew " + noun, noun
	case "villain":
		noun = pick(rbDown, "a villain")
		return 6, "brought down " + noun, noun
	case "rescue":
		noun = pick(rbHelp, "the afflicted")
		return 3, "helped at " + noun, noun
	case "escort":
		return 3, "led refugees to safety", "refugees"
	case "relic":
		noun = pick(rbRelic, "a relic")
		switch {
		case strings.Contains(low, "gave the relic"):
			return 3, "gave the relic " + noun + " to the museum", noun
		case strings.Contains(low, "sold by"):
			return -1, "sold the relic " + noun, noun
		case strings.Contains(low, "set out bearing"):
			return 0, "", ""
		}
		return 2, "recovered the relic " + noun, noun
	case "bounty":
		noun = pick(rbBounty, "an outlaw")
		return 2, "collected the bounty on " + noun, noun
	case "quest":
		if strings.Contains(low, "took up") || strings.Contains(low, "has now") {
			return 0, "", ""
		}
		noun = other("the folk")
		switch {
		case strings.Contains(low, "failed") || strings.Contains(low, "struck from") || strings.Contains(low, "resigned"):
			return -1, "failed those who trusted them", noun
		case strings.Contains(low, "champion"):
			return 3, "became champion of the arena", noun
		case strings.Contains(low, "great errand"):
			return 3, "finished the elder's great errand", noun
		case strings.Contains(low, "enlisted") || strings.Contains(low, "raised to") || strings.Contains(low, "deputy"):
			return 1, "took service under " + noun, noun
		}
		if ok {
			return 1, rest, noun
		}
		return 1, "finished a task for " + noun, noun
	case "nemesis":
		switch {
		case strings.Contains(low, "ended the feud"):
			return 3, "ended a blood feud", "the feud"
		case strings.Contains(low, "made an end of the feud"):
			return 2, "made peace with an old enemy", "the feud"
		case strings.Contains(low, "drove off"):
			return 1, "drove off a sworn enemy", "the enemy"
		}
	case "rival":
		switch {
		case strings.Contains(low, "slew the rival"):
			return 2, "slew a rival adventurer", "the rival"
		case strings.Contains(low, "won a duel"):
			return 1, "won a duel and spared the loser", "the rival"
		case strings.Contains(low, "sabotag"):
			return -2, "sabotaged a rival adventurer", "the rival"
		}
	case "murder":
		noun = "the dead"
		if m := rbVictim.FindStringSubmatch(t); m != nil {
			noun = strings.TrimSpace(m[1])
		}
		switch {
		case strings.Contains(low, "had yielded"):
			return -6, "struck down one who had yielded", noun
		case strings.Contains(low, "broke the"):
			return -5, "broke a sacred rite and killed", noun
		}
		return -4, "murdered " + noun, noun
	case "crime":
		noun = other("the law")
		if strings.Contains(low, "slain on contract") {
			return -5, "killed for coin", noun
		}
		if ok {
			return -3, rest, noun
		}
		return -3, "is blamed for a crime", noun
	case "event":
		if m := rbTitle.FindStringSubmatch(t); m != nil {
			noun = strings.TrimSpace(m[1])
			return 2, "earned the name " + noun, noun
		}
	case "survived":
		return 2, "survived the plague", "the plague"
	}
	return 0, "", ""
}

func rbDist(w *WorldData, a, b int) int {
	if w == nil || w.W <= 0 {
		return 0
	}
	dx, dy := abs(a%w.W-b%w.W), abs(a/w.W-b/w.W)
	return maxi(dx, dy) + mini(dx, dy)/2
}

func (g *Game) rbBurgAt(p int) int {
	best, bd := 2, 0
	for i := range g.W.Burgs {
		b := &g.W.Burgs[i]
		if !b.Alive {
			continue
		}
		if d := rbDist(g.W, b.P, p); d < best {
			best, bd = d, b.ID
		}
	}
	return bd
}

// rbScan folds new deeds into the state. New deeds happened where the wanderer stands.
func (g *Game) rbScan() {
	if g.W == nil || g.Cur == nil || g.P == nil {
		return
	}
	st := g.rb()
	if st.Cur > len(g.Deeds) {
		st.Cur = len(g.Deeds)
	}
	if st.Cur == len(g.Deeds) {
		return
	}
	batch := len(g.Deeds) - st.Cur
	here := g.W.Idx(g.Cur.WX, g.Cur.WY)
	var cellP map[int]int
	for ; st.Cur < len(g.Deeds); st.Cur++ {
		d := g.Deeds[st.Cur]
		if d.Planet != g.W.uniIdx {
			continue
		}
		w, do, noun := g.rbClassify(d)
		if w == 0 || do == "" {
			continue
		}
		p, day := here, lvDay(g)
		if batch > 3 { // an old save or a planet swap: place old deeds by their cell and year
			if cellP == nil {
				cellP = map[int]int{}
				for i, c := range g.W.Cell {
					if _, ok := cellP[c]; !ok {
						cellP[c] = i
					}
				}
			}
			if i, ok := cellP[d.Cell]; ok && d.Cell >= 0 {
				p = i
			}
			day = maxi(0, (d.Year-g.W.Year)*112)
		}
		st.Deeds = append(st.Deeds, rbDeed{Day: day, P: p, Town: g.rbBurgAt(p), Kind: d.Kind, W: w, Do: do, Noun: noun})
	}
	if len(st.Deeds) > 160 {
		st.Deeds = st.Deeds[len(st.Deeds)-160:]
	}
}

// ---------------------------------------------------------------- the rumour

type rbHeard struct {
	D    *rbDeed
	Str  float64
	Dist int
}

// rbHeardAt lists the deeds a town has heard of by the given day, strongest first.
func (g *Game) rbHeardAt(bd *BurgData, day int) []rbHeard {
	st := g.rb()
	var out []rbHeard
	for i := range st.Deeds {
		d := &st.Deeds[i]
		dist := rbDist(g.W, bd.P, d.P)
		if d.Town == bd.ID {
			dist = 0
		}
		if day < d.Day+dist*2 {
			continue
		}
		radius := mini(14, 5+2*abs(d.W))
		if dist > radius {
			continue
		}
		s := float64(clampi(abs(d.W), 1, 6)) / 6 * (1 - float64(dist)/float64(radius+1))
		if s < 0.05 {
			continue
		}
		out = append(out, rbHeard{D: d, Str: s, Dist: dist})
	}
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].Str*float64(abs(out[i].D.W)) > out[j].Str*float64(abs(out[j].D.W))
	})
	return out
}

func (g *Game) rbFame() int {
	n := g.Kills / 8
	for _, d := range g.rb().Deeds {
		if d.W > 0 {
			n += d.W
		}
	}
	return n
}

// ---------------------------------------------------------------- lines

type rbLine struct {
	S float64 // signed: + admired, - despised
	T string
}

func (g *Game) rbRegister(bd *BurgData) string {
	reg := ""
	if cp := g.W.cpOf(bd.Culture); cp != nil && cp.Voice != nil {
		reg = cp.Voice.Register
	}
	if reg == "" {
		reg = dvRegisterFor(g.W.cultByID[bd.Culture])
	}
	switch reg {
	case "formal", "courtly":
		return "f"
	case "flowery":
		return "o"
	}
	return "p"
}

var rbFrames = map[string][2][]string{
	"p": {
		{"Word is that %s. Folk here think well of it.", "Heard in the market: %s. No complaints.", "%s. The town nods."},
		{"%s. Bar your door.", "Watch your purse: %s.", "They say %s, and they do not say it kindly."},
	},
	"f": {
		{"Be it known in {town} that %s, to the credit of all.", "Notice, with the thanks of the council: %s.", "Let it be entered in the record that %s."},
		{"Let it be recorded that %s. The watch has been told.", "Notice: %s. Report any sighting to the magistrate.", "The council notes with grave concern that %s."},
	},
	"o": {
		{"Sing it at every door: %s!", "They say %s, and the lamps of {town} burn brighter for it.", "Hear it told: %s, and the whole street cheered."},
		{"Alas, %s, and the lamps of {town} burn low.", "A shadow lies on the road: %s.", "Whisper it, for it is ill luck to say aloud: %s."},
	},
}

var rbSecondhand = []string{"Rumour only: ", "Heard secondhand: ", "A carter swears that this is so. ", "Passed along from the next town: "}

func (g *Game) rbFrame(bd *BurgData, pos bool, x string, str float64, key string) string {
	f := rbFrames[g.rbRegister(bd)]
	list := f[boolInt(!pos)]
	s := fmt.Sprintf(rbPick(g, list, "frame", key, bd.Name), x)
	s = strings.ReplaceAll(s, "{town}", bd.Name)
	if str < 0.4 {
		s = rbPick(g, rbSecondhand, "sec", key, bd.Name) + s
	}
	return s
}

var rbKarma = [5][2]string{
	{"spares the beaten and the begging", "leaves no foe breathing"},
	{"keeps their word to the last copper", "has lied across half the realm"},
	{"walks into places others flee", "is never first through a door"},
	{"leaves coin where it is needed", "takes the last coin from a poor table"},
	{"keeps the old rites", "spits at the shrines"},
}

// rbLines builds every line the town could post about the wanderer.
func (g *Game) rbLines(bd *BurgData, day int) []rbLine {
	var out []rbLine
	me := g.P.Name
	for _, h := range g.rbHeardAt(bd, day) {
		pos := h.D.W > 0
		key := fmt.Sprintf("d%d:%s", h.D.Day, h.D.Kind)
		out = append(out, rbLine{float64(h.D.W) * h.Str, g.rbFrame(bd, pos, me+" "+h.D.Do, h.Str, key)})
	}
	if b := g.uwBounty(bd.State); bd.State > 0 && b > 0 {
		out = append(out, rbLine{-9, fmt.Sprintf("{{R|WANTED}} %s, %d %s, for crimes against %s.", me, b, g.Sk.Currency, g.realmName(bd.State))})
	}
	if bd.State > 0 {
		r := g.Rep[fmt.Sprintf("state:%d", bd.State)]
		switch {
		case r >= 250:
			out = append(out, rbLine{2.5, g.rbFrame(bd, true, me+" is counted among the friends of "+g.realmName(bd.State), 1, "realm")})
		case r <= -250:
			out = append(out, rbLine{-2.5, g.rbFrame(bd, false, me+" is counted among the enemies of "+g.realmName(bd.State), 1, "realm")})
		}
	}
	if n := len(g.W.Religion); n > 0 {
		if id := g.W.Religion[clampi(bd.P, 0, n-1)]; id > 0 {
			if rel := g.W.relByID[id]; rel != nil {
				r := g.Rep[fmt.Sprintf("faith:%d", id)]
				switch {
				case r >= 250:
					out = append(out, rbLine{2, g.rbFrame(bd, true, "the priests of "+rel.Name+" keep a place at table for "+me, 1, "faith")})
				case r <= -250:
					out = append(out, rbLine{-2, g.rbFrame(bd, false, "the priests of "+rel.Name+" have struck "+me+" from their book", 1, "faith")})
				}
			}
		}
	}
	if g.rbFame() >= 6 { // karma travels with the name, so only the known are judged by it
		for ax := 0; ax < 5; ax++ {
			v := g.crTrait(ax)
			switch {
			case v >= 35:
				out = append(out, rbLine{2 + float64(v)/50, g.rbFrame(bd, true, fmt.Sprintf("%s %s", me, rbKarma[ax][0]), 1, fmt.Sprint("k", ax))})
			case v <= -35:
				out = append(out, rbLine{-2 + float64(v)/50, g.rbFrame(bd, false, fmt.Sprintf("%s %s", me, rbKarma[ax][1]), 1, fmt.Sprint("k", ax))})
			}
		}
	}
	if g.Kills >= 30 {
		out = append(out, rbLine{-1, g.rbFrame(bd, false, fmt.Sprintf("%s has put %d to the sword and walks on", me, g.Kills), 0.5, "kills")})
	}
	if k := g.rb().Kills[bd.ID]; k > 0 {
		out = append(out, rbLine{-float64(2 + k*2), g.rbFrame(bd, false, fmt.Sprintf("%s killed %d of our own in the streets of %s", me, k, bd.Name), 1, "ownkills")})
	}
	sort.SliceStable(out, func(i, j int) bool { return abs(int(out[i].S*100)) > abs(int(out[j].S*100)) })
	return out
}

func rbStanding(s float64) string {
	switch {
	case s >= 8:
		return "{{G|Adored}}"
	case s >= 4:
		return "{{g|Respected}}"
	case s >= 1.5:
		return "{{g|Spoken of well}}"
	case s <= -8:
		return "{{R|Feared and loathed}}"
	case s <= -4:
		return "{{r|Distrusted}}"
	case s <= -1.5:
		return "{{y|Spoken of ill}}"
	}
	return "{{K|Hardly known}}"
}

func rbSum(ls []rbLine) float64 {
	s := 0.0
	for _, l := range ls {
		s += l.S
	}
	return s
}

// ---------------------------------------------------------------- props

func rbTownKind(g *Game, zn *Zone) (string, *BurgData) {
	if g.W == nil || zn == nil {
		return "", nil
	}
	return ehKind(g, zn)
}

func (g *Game) rbBoardText(bd *BurgData, day int) string {
	var b strings.Builder
	b.WriteString("{{c|The reputation board of " + bd.Name + ", papers pinned three deep.}}\n")
	ls := g.rbLines(bd, day)
	n := 0
	for _, l := range ls {
		if n >= 5 {
			break
		}
		n++
		b.WriteString("\n- " + l.T)
	}
	if n == 0 {
		b.WriteString("\nNothing here concerns you. The board talks of harvests, strays and a lost goat.")
	}
	if cp := g.W.cpOf(bd.Culture); cp != nil && cp.Voice != nil && len(cp.Voice.Proverbs) > 0 {
		b.WriteString("\n\n{{K|Nailed under it, an old saying: \"" + cpFill(g.W.cpChoose(cp.Voice.Proverbs, "rb:prov", bd.Name), nil) + "\"}}")
	}
	return b.String()
}

var rbGrafHero = []string{"%s!", "%s. REMEMBER.", "%s. WELL DONE."}
var rbGrafShame = []string{"%s. WE HAVE NOT FORGOTTEN.", "%s. SHAME.", "%s. MAY THEY CHOKE ON IT."}

var rbRhymeHero = []string{
	"One for the {n}, two for the door,|three for {P}, who walks the floor.",
	"Who's afraid of the {n} tonight?|Not {P}, who held on to the light!",
	"{P} faced the {n} alone,|and now it's nothing but a bone.",
}
var rbRhymeShame = []string{
	"Hush, little one, bolt the gate,|{P} came for {n} and will not wait.",
	"One, two, {P} at the door,|three, four, {n} on the floor.",
	"Don't go out when the lamps burn low,|{P} and the {n} are walking slow.",
}

// rbBig is the deed the town talks of most, if it is a big one.
func (g *Game) rbBig(bd *BurgData, day int) *rbHeard {
	for _, h := range g.rbHeardAt(bd, day) {
		h := h
		if abs(h.D.W) >= 3 && h.Str >= 0.3 {
			return &h
		}
	}
	return nil
}

func (g *Game) rbGrafText(bd *BurgData, day int) string {
	h := g.rbBig(bd, day)
	if h == nil {
		return "Old whitewash and chalk smudges. Nothing about you, yet."
	}
	me := strings.ToUpper(g.P.Name)
	x := me + " " + strings.ToUpper(h.D.Do)
	if h.D.W > 0 {
		return "Chalked large on the wall: {{W|" + fmt.Sprintf(rbPick(g, rbGrafHero, "gh", bd.Name, h.D.Do), x) + "}}"
	}
	return "Scratched deep into the wall: {{R|" + fmt.Sprintf(rbPick(g, rbGrafShame, "gs", bd.Name, h.D.Do), x) + "}}"
}

func (g *Game) rbRhymeText(bd *BurgData, h *rbHeard) string {
	list := rbRhymeHero
	if h.D.W < 0 {
		list = rbRhymeShame
	}
	s := rbPick(g, list, "rh", bd.Name, h.D.Do)
	noun := strings.TrimPrefix(strings.TrimPrefix(h.D.Noun, "the "), "a ")
	s = strings.ReplaceAll(strings.ReplaceAll(s, "{n}", noun), "{P}", g.P.Name)
	return strings.ReplaceAll(s, "|", " / ")
}

// rbNamesFor decides whether this town renamed its signs for the wanderer (mood +1 saved, -1 ruined).
func (g *Game) rbNamesFor(bd *BurgData) *rbNames {
	st := g.rb()
	pos, neg := 0, 0
	for _, d := range st.Deeds {
		if d.Town != bd.ID {
			continue
		}
		if d.W > 0 {
			pos += d.W
		} else {
			neg += -d.W
		}
	}
	neg += st.Kills[bd.ID] * 3
	mood := 0
	switch {
	case pos >= 5 && pos >= neg:
		mood = 1
	case neg >= 6 && neg > pos:
		mood = -1
	}
	n := st.Names[bd.ID]
	if mood == 0 {
		return n // an old renaming stands, even if the deeds later balance out
	}
	if n != nil && n.Mood == mood {
		if mood == 1 && n.Shop == "" && pos >= 10 {
			n.Shop, n.ShopOld = rbShopNames(g, bd, mood)
		}
		if mood == -1 && n.Shop == "" && neg >= 10 {
			n.Shop, n.ShopOld = rbShopNames(g, bd, mood)
		}
		return n
	}
	first := "The Wanderer"
	if f := strings.Fields(g.P.Name); len(f) > 0 {
		first = f[0]
	}
	inns := []string{"The Plough and Lantern", "The Crooked Lamp", "The Green Kettle", "The Salted Hare", "The Three Barrels", "The Quiet Ox"}
	var nn string
	if mood == 1 {
		nn = rbPick(g, []string{"%s's Rest", "The Wanderer's Welcome", "%s's Hearth"}, "inn+", bd.Name)
	} else {
		nn = rbPick(g, []string{"The Bolted Door", "The Last Candle", "The Weeping Lamp"}, "inn-", bd.Name)
	}
	nn = strings.ReplaceAll(nn, "%s", first)
	n = &rbNames{Mood: mood, Inn: nn, InnOld: rbPick(g, inns, "innold", bd.Name)}
	st.Names[bd.ID] = n
	return n
}

func rbShopNames(g *Game, bd *BurgData, mood int) (string, string) {
	old := rbPick(g, []string{"Hale and Daughters, Chandlers", "The Brass Scale", "Orrin's Dry Goods", "The Fair Measure"}, "shopold", bd.Name)
	if mood == 1 {
		return rbPick(g, []string{"The Open Hand", "Honest Weight", "The Wanderer's Bargain"}, "shop+", bd.Name), old
	}
	return rbPick(g, []string{"Shuttered, Gone Away", "Closed Since the Trouble", "Nothing Left to Sell"}, "shop-", bd.Name), old
}

func (g *Game) rbSignText(n *rbNames, bd *BurgData, kind string) string {
	name, old, thing := n.Inn, n.InnOld, "inn"
	if kind == "shop" {
		name, old, thing = n.Shop, n.ShopOld, "shop"
	}
	why := fmt.Sprintf("%s has not forgotten what %s did here.", bd.Name, g.P.Name)
	if n.Mood < 0 {
		why = fmt.Sprintf("%s has not forgiven what %s did here.", bd.Name, g.P.Name)
	}
	return fmt.Sprintf("The sign over the %s door has been repainted: {{Y|%s}}. Under the new paint the old name still shows: {{K|%s}}.\n\n{{K|%s}}", thing, name, old, why)
}

// rbPlace turns a free wall tile into a prop with a note (walls never carry the path, so the zone stays connected).
func (g *Game) rbPlace(zn *Zone, r *Rng, note string) (int, bool) {
	c := &ehCtx{g: g, zn: zn}
	x, y, ok := c.wallSpot(r, 1)
	if !ok {
		return 0, false
	}
	zn.Set(x, y, TInscription)
	zn.Notes[y*MW+x] = note
	return y*MW + x, true
}

var rbBoardKinds = map[string]bool{"village": true, "plaza": true, "market": true, "palace": true}
var rbGrafKinds = map[string]bool{"village": true, "homes": true, "slums": true, "market": true, "barracks": true, "artisans": true, "harbor": true}

func rbZone(g *Game, zn *Zone, _ *Rng) {
	kind, bd := rbTownKind(g, zn)
	if bd == nil {
		return
	}
	st := g.rb()
	if rbBoardKinds[kind] {
		if _, has := st.Board[bd.ID]; !has {
			if idx, ok := g.rbPlace(zn, NewRng(g.Seed+":rb:board:"+zn.Key), "{{c|A reputation board, its papers pinned three deep.}}"); ok {
				st.Board[bd.ID] = zn.Key
				st.BoardT[zn.Key] = idx
			}
		}
	}
	if rbGrafKinds[kind] && HashStr(g.Seed+":rb:grafgate:"+zn.Key)%100 < 45 {
		if _, has := st.Graf[zn.Key]; !has {
			if idx, ok := g.rbPlace(zn, NewRng(g.Seed+":rb:graf:"+zn.Key), "Old whitewash and chalk smudges."); ok {
				st.Graf[zn.Key] = idx
			}
		}
	}
}

// rbRefresh rewrites the notes of the board, graffiti and signs of the zone the wanderer just entered.
func (g *Game) rbRefresh(zn *Zone, bd *BurgData) {
	st := g.rb()
	day := lvDay(g)
	if idx, ok := st.BoardT[zn.Key]; ok && zn.T[idx] == TInscription && st.Board[bd.ID] == zn.Key {
		zn.Notes[idx] = g.rbBoardText(bd, day)
	}
	if idx, ok := st.Graf[zn.Key]; ok && zn.T[idx] == TInscription {
		zn.Notes[idx] = g.rbGrafText(bd, day)
	}
	kind, _ := rbTownKind(g, zn)
	n := g.rbNamesFor(bd)
	if n == nil || kind == "outskirts" || zn.Z != 0 {
		return
	}
	for _, sk := range []string{"inn", "shop"} {
		if sk == "shop" && n.Shop == "" {
			continue
		}
		key := fmt.Sprintf("%d:%s", bd.ID, sk)
		sg := st.Signs[key]
		if sg == nil {
			if !rbBoardKinds[kind] && kind != "homes" && kind != "harbor" {
				continue
			}
			idx, ok := g.rbPlace(zn, NewRng(g.Seed+":rb:sign:"+key+":"+zn.Key), g.rbSignText(n, bd, sk))
			if !ok {
				continue
			}
			st.Signs[key] = &rbSign{Zone: zn.Key, Tile: idx}
			continue
		}
		if sg.Zone == zn.Key && zn.T[sg.Tile] == TInscription {
			zn.Notes[sg.Tile] = g.rbSignText(n, bd, sk)
		}
	}
}

// ---------------------------------------------------------------- hooks

func rbEnter(g *Game, zn *Zone) {
	if g.W == nil || g.P == nil {
		return
	}
	g.rbScan()
	_, bd := rbTownKind(g, zn)
	if bd == nil {
		return
	}
	g.rb().Seen[bd.ID] = lvDay(g)
	g.rbRefresh(zn, bd)
}

func rbTurn(g *Game) {
	if g.W == nil || g.P == nil || g.Cur == nil {
		return
	}
	g.rbScan()
	zn := g.Cur
	if zn.Kind != "town" && zn.Kind != "outskirts" {
		return
	}
	_, bd := rbTownKind(g, zn)
	if bd == nil {
		return
	}
	st := g.rb()
	day := lvDay(g)
	st.Seen[bd.ID] = day
	if g.Turn%120 == 0 {
		g.rbRefresh(zn, bd)
	}
	h := g.Hour()
	if g.Turn%50 != 0 || h < 8 || h > 18 || zn.Kind != "town" || g.Mode != "main" {
		if g.Turn%40 == 0 && zn.Kind == "town" && g.Mode == "main" {
			rbBardSings(g, bd, day)
		}
		return
	}
	if last, ok := st.Rhyme[bd.ID]; ok && day-last < 3 {
		return
	}
	big := g.rbBig(bd, day)
	if big == nil || HashStr(fmt.Sprintf("%s:rb:rhymegate:%d:%d", g.Seed, bd.ID, g.Turn))%100 >= 35 {
		return
	}
	st.Rhyme[bd.ID] = day
	g.Msg("{{c|Children skip rope nearby, chanting:}} {{W|" + g.rbRhymeText(bd, big) + "}}")
}

func rbKill(g *Game, d, src *Actor) {
	if g.W == nil || g.Cur == nil || src != g.P || d == nil || g.Cur.Burg <= 0 {
		return
	}
	if g.Cur.Kind != "town" && g.Cur.Kind != "outskirts" {
		return
	}
	if d.AI == "hostile" || d.Angry && d.Faction == "outlaws" || !crIsPerson(d) {
		return
	}
	g.rb().Kills[g.Cur.Burg]++
}

// ---------------------------------------------------------------- the bard

func (g *Game) rbPosScore(bd *BurgData, day int) float64 {
	s := 0.0
	for _, h := range g.rbHeardAt(bd, day) {
		if h.D.W > 0 {
			s += h.Str * float64(h.D.W)
		}
	}
	return s
}

func (g *Game) rbSong(bd *BurgData, day int) string {
	me := g.P.Name
	var good, bad []rbHeard
	for _, h := range g.rbHeardAt(bd, day) {
		if h.D.W > 0 {
			good = append(good, h)
		} else if h.D.W <= -3 {
			bad = append(bad, h)
		}
	}
	if len(good) == 0 {
		return ""
	}
	form := "ballad"
	if cp := g.W.cpOf(bd.Culture); cp != nil && cp.Literature != nil && len(cp.Literature.Forms) > 0 {
		form = strings.ToLower(cp.Literature.Forms[g.W.cpIdx(len(cp.Literature.Forms), "rb:form", bd.Name)].Name)
	}
	var b strings.Builder
	b.WriteString("{{W|The " + form + " of " + me + "}}\n\n")
	b.WriteString("Oh gather round and hush your tongue,\na song of " + me + " must be sung:\n")
	b.WriteString("how " + me + " " + good[0].D.Do + ",\nand " + bd.Name + " thanks them, old and young.\n")
	if len(good) > 1 {
		b.WriteString("\nAnd more: " + me + " " + good[1].D.Do + ".\n")
	}
	if len(bad) > 0 {
		b.WriteString("\n{{K|(Some sing another verse, and it is not kind: " + me + " " + bad[0].D.Do + ".)}}\n")
	}
	return b.String()
}

func rbBardSings(g *Game, bd *BurgData, day int) {
	st := g.rb()
	if _, done := st.Sung[bd.ID]; done || g.rbPosScore(bd, day) < 7 {
		return
	}
	for _, a := range g.Cur.Actors {
		if a.HP > 0 && a.Role == "bard" && g.visible(a.X, a.Y) {
			st.Sung[bd.ID] = day
			g.Msg("{{M|A bard in the square strikes up a song, and you catch your own name in it.}} {{K|(Talk to the bard to hear it.)}}")
			return
		}
	}
}

func rbTalk(g *Game, a *Actor) []Option {
	if g.W == nil || a == nil || a.Role != "bard" || g.Cur == nil {
		return nil
	}
	_, bd := rbTownKind(g, g.Cur)
	if bd == nil {
		return nil
	}
	day := lvDay(g)
	if g.rbPosScore(bd, day) < 7 {
		return nil
	}
	return []Option{{"Ask the bard to sing the song they have made about you.", func() {
		g.rb().Sung[bd.ID] = day
		g.showPopup("The bard's song", g.rbSong(bd, day))
	}}}
}

// ---------------------------------------------------------------- the screen

func rbKeys(g *Game, k string) {
	switch k {
	case "Escape", "q", "Enter":
		g.Mode = "main"
	case "ArrowDown", "j":
		g.Sel++
	case "ArrowUp", "k":
		g.Sel = maxi(0, g.Sel-1)
	case "PageDown":
		g.Sel += 10
	case "PageUp":
		g.Sel = maxi(0, g.Sel-10)
	}
}

// rbReport is the screen text: one block per town the wanderer has been to.
func (g *Game) rbReport() []string {
	g.rbScan()
	st := g.rb()
	day := lvDay(g)
	L := []string{fmt.Sprintf("They call you {{Y|%s}}. Fame: {{W|%d}}. The dead you have left: %d.", g.crEpithet(), g.rbFame(), g.Kills), ""}
	var ids []int
	for id := range st.Seen {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool {
		if st.Seen[ids[i]] != st.Seen[ids[j]] {
			return st.Seen[ids[i]] > st.Seen[ids[j]]
		}
		return ids[i] < ids[j]
	})
	if len(ids) == 0 {
		return append(L, "{{K|You have not walked a town street yet. No one has anything to say.}}")
	}
	for _, id := range ids {
		bd := g.W.burgByID[id]
		if bd == nil {
			continue
		}
		ls := g.rbLines(bd, day)
		L = append(L, fmt.Sprintf("{{Y|%s}} {{K|(%s)}}   %s", bd.Name, g.realmName(bd.State), rbStanding(rbSum(ls))))
		var best, worst *rbLine
		for i := range ls {
			if ls[i].S > 0 && (best == nil || ls[i].S > best.S) {
				best = &ls[i]
			}
			if ls[i].S < 0 && (worst == nil || ls[i].S < worst.S) {
				worst = &ls[i]
			}
		}
		if best != nil {
			L = append(L, "  {{G|Best:}}  "+best.T)
		} else {
			L = append(L, "  {{K|Best:  nothing good is said.}}")
		}
		if worst != nil {
			L = append(L, "  {{R|Worst:}} "+worst.T)
		} else {
			L = append(L, "  {{K|Worst: nothing bad is said.}}")
		}
		if n := st.Names[id]; n != nil {
			L = append(L, fmt.Sprintf("  {{K|The inn is now called %s (once %s).}}", n.Inn, n.InnOld))
		}
		L = append(L, "")
	}
	return L
}

func (g *Game) rbDraw() {
	s := g.sc
	x, y, w, h := g.panel("What they say of you")
	var rows []string
	for _, l := range g.rbReport() {
		rows = append(rows, Wrap(l, w)...)
	}
	maxOff := maxi(0, len(rows)-(h-3))
	g.Sel = clampi(g.Sel, 0, maxOff)
	for i := g.Sel; i < len(rows) && i-g.Sel < h-3; i++ {
		s.TextB(x, y+i-g.Sel, rows[i], CGray, CBgDark, x+w)
	}
	s.TextB(x, y+h-2, "{{K|Esc closes   up/down scrolls}}", CGray, CBgDark, x+w)
}
