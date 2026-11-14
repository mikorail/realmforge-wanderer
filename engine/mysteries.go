package engine

// Mystery generator (prefix mx): whodunits built from real life-simulation events. State g.ext("mx"),
// planet-local. Case generation uses side rngs only (NewRng(seed+":mx:"+key)); g.R is used by play-time checks.
//
// A case: a victim and a crime (found dead, a theft, an arson, a kidnapping, a forged will, a missing shipment),
// 4-6 suspects built from the real people of the burg (kin, rivals, a lover, a servant, a debtor, an outsider), a
// culprit with motive, means and a false alibi, a scene of the crime with props, 6-10 clues (physical, testimonial,
// documentary) and 1-2 red herrings. The Casebook (mode "mxb") shows it all; accusing with enough consistent
// evidence ends it, a bluff may work on a nervous suspect, a wrong accusation costs reputation.

import (
	"fmt"
	"sort"
	"strings"
)

const (
	mxMaxOpen = 2  // open cases at once
	mxColdDay = 21 // days before an unsolved case goes cold
	mxNearP   = 2  // parasangs from the player (about three days of travel)
)

type mxSusp struct {
	Key       string `json:"k"`
	Name      string `json:"n"`
	Rel       string `json:"r"`
	Why       string `json:"y"`            // the public reason they could have done it
	Alibi     string `json:"a"`            // what they say
	AlibiOK   bool   `json:"ao,omitempty"` // the alibi is true
	Witness   string `json:"w,omitempty"`  // person key who can confirm or break the alibi
	Secret    string `json:"s,omitempty"`
	Herring   bool   `json:"h,omitempty"`
	Asked     bool   `json:"as,omitempty"`
	SecretOut bool   `json:"so,omitempty"`
	AskT      int    `json:"at,omitempty"` // turn of the last failed question
	Nerves    bool   `json:"nv,omitempty"` // pressed hard and sweating
}

type mxClue struct {
	ID      int    `json:"i"`
	Kind    string `json:"k"`            // phys test doc
	Method  string `json:"m"`            // vis search body tox papers tracks talk press
	Src     string `json:"s,omitempty"`  // person key (talk, press)
	Text    string `json:"t"`            // what you learn
	For     string `json:"f,omitempty"`  // suspect it implicates
	Clear   string `json:"c,omitempty"`  // suspect it clears
	Aspect  string `json:"a,omitempty"`  // id motive means opp
	Rel     int    `json:"r"`            // reliability 1..3
	Lie     bool   `json:"l,omitempty"`  // false testimony (the player is not told)
	DC      int    `json:"dc"`           // difficulty
	X       int    `json:"x,omitempty"`  // where it lies (scene zone)
	Y       int    `json:"y,omitempty"`  //
	Found   bool   `json:"fd,omitempty"` //
	TL      string `json:"tl,omitempty"` // timeline line
	TH      int    `json:"th,omitempty"` // timeline hour (shifted past midnight)
	TryT    int    `json:"tt,omitempty"` // turn of the last failed attempt
	Herring bool   `json:"hr,omitempty"`
}

type mxScene struct {
	Name    string  `json:"n"`
	Kind    int     `json:"k"`
	ZoneKey string  `json:"z,omitempty"`
	Placed  bool    `json:"p,omitempty"`
	Clean   bool    `json:"cl,omitempty"`
	X       int     `json:"x,omitempty"`
	Y       int     `json:"y,omitempty"`
	Under   Terrain `json:"u,omitempty"`
}

type mxCase struct {
	ID      int            `json:"i"`
	Kind    string         `json:"k"`
	Title   string         `json:"t"`
	Burg    int            `json:"b"`
	Day     int            `json:"d"`
	Victim  string         `json:"v"`
	VName   string         `json:"vn"`
	Culprit string         `json:"c"`
	Motive  string         `json:"mo"`
	MotiveT string         `json:"mt"`
	Means   string         `json:"me"`
	Hour    int            `json:"h"`
	Status  string         `json:"st"` // open solved cold failed
	Outcome string         `json:"o,omitempty"`
	Wrong   int            `json:"w,omitempty"`
	Blamed  bool           `json:"bl,omitempty"`
	Suspect []*mxSusp      `json:"su"`
	Clues   []*mxClue      `json:"cl"`
	Scene   mxScene        `json:"sc"`
	Tried   map[string]int `json:"tr,omitempty"` // method -> turn of last failed attempt
	Reward  string         `json:"rw,omitempty"`
}

type mxUI struct {
	page     string // book scene acc
	tab, off int
	cur      int
	fromTalk bool
	step     int
	pick     int
	picks    map[int]bool
	res      []string
	sus      int
}

type mxState struct {
	Cases   []*mxCase       `json:"c,omitempty"`
	Seen    map[string]bool `json:"s,omitempty"`
	Next    int             `json:"n,omitempty"`
	Base    bool            `json:"b,omitempty"`
	LastDay int             `json:"ld,omitempty"`
	ScanT   int             `json:"sc,omitempty"`
	Solved  int             `json:"so,omitempty"`
	Firsts  map[string]bool `json:"f,omitempty"`

	ui mxUI
}

func init() {
	unLocalKeys = append(unLocalKeys, "mx")
	turnHooks = append(turnHooks, mxTurn)
	enterHooks = append(enterHooks, mxEnter)
	talkHooks = append(talkHooks, mxTalk)
	modeKeys["mxb"] = mxKeys
	modeDraws["mxb"] = func(g *Game) { g.mxDraw() }
}

func mxSt(g *Game) *mxState {
	s := g.ext("mx", func() any { return &mxState{} }).(*mxState)
	if s.Seen == nil {
		s.Seen = map[string]bool{}
	}
	if s.Firsts == nil {
		s.Firsts = map[string]bool{}
	}
	return s
}

func mxFirst(g *Game, key, text string) {
	st := mxSt(g)
	if st.Firsts[key] {
		return
	}
	st.Firsts[key] = true
	if g.Cur != nil && g.W != nil {
		g.deed("mx", text, 0, 0)
	}
}

func (c *mxCase) open() bool { return c.Status == "open" }

func (c *mxCase) susp(key string) *mxSusp {
	for _, s := range c.Suspect {
		if s.Key == key {
			return s
		}
	}
	return nil
}

func (c *mxCase) clue(id int) *mxClue {
	for _, cl := range c.Clues {
		if cl.ID == id {
			return cl
		}
	}
	return nil
}

func mxOpenCount(st *mxState) int {
	n := 0
	for _, c := range st.Cases {
		if c.open() {
			n++
		}
	}
	return n
}

func mxName(g *Game, key string) string {
	if p := lvGet(g, key); p != nil {
		return p.Name
	}
	return "someone"
}

// ---------------------------------------------------------------- watching the world

func mxTurn(g *Game) {
	if g == nil || g.W == nil || g.P == nil || g.Cur == nil {
		return
	}
	st := mxSt(g)
	mxCollect(g, st)
	if st.ScanT != 0 && g.Turn-st.ScanT < 60 && g.Turn >= st.ScanT {
		return
	}
	st.ScanT = g.Turn
	mxScan(g, st)
}

// mxCollect turns picked-up clue props into casebook entries.
func mxCollect(g *Game, st *mxState) {
	if len(st.Cases) == 0 {
		return
	}
	for i := 0; i < len(g.P.Inv); i++ {
		it := g.P.Inv[i]
		if it.Extra == nil || it.Extra["mx"] == 0 {
			continue
		}
		g.P.Inv = append(g.P.Inv[:i], g.P.Inv[i+1:]...)
		i--
		for _, c := range st.Cases {
			if c.ID == it.Extra["mx"] {
				if cl := c.clue(it.Extra["mxc"]); cl != nil && !cl.Found {
					mxFound(g, c, cl, "You pocket it and add it to the casebook.")
				}
			}
		}
	}
}

func mxScan(g *Game, st *mxState) {
	day := lvDay(g)
	pp, pb := lsPlayerLoc(g)
	if pp < 0 {
		return
	}
	catch := st.LastDay != 0 && day-st.LastDay > 3
	st.LastDay = day
	ls := lsSt(g)
	people := lvSt(g).People
	keys := make([]string, 0, len(people))
	for k := range people {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	// age out old cases
	for _, c := range st.Cases {
		if c.open() && day-c.Day > mxColdDay {
			mxGoCold(g, c)
		}
	}
	near := func(burg int) bool {
		if burg <= 0 {
			return false
		}
		if burg == pb {
			return true
		}
		b := g.W.burgByID[burg]
		return b != nil && b.Alive && wsCheb(g.W, b.P, pp) <= mxNearP
	}
	var made *mxCase
	try := func(kind string, victim, culprit *lvPerson, burg int, tag string) {
		if made != nil || mxOpenCount(st) >= mxMaxOpen || victim == nil {
			return
		}
		made = mxBuild(g, kind, victim, culprit, burg, tag)
	}
	// 1. people who died: baseline on the first look, then fresh deaths near the player
	for _, k := range keys {
		p := people[k]
		if p.Alive || st.Seen["d:"+k] {
			continue
		}
		st.Seen["d:"+k] = true
		if !st.Base || catch || p.Died != lvYear(g) || !near(p.Burg) || p.Role == "child" {
			continue
		}
		cause := strings.ToLower(p.Cause)
		switch {
		case strings.Contains(cause, "murdered"):
			if kl := lvGet(g, ls.Avenge[p.Key]); kl != nil && kl.Alive && kl.Burg == p.Burg {
				try("murder", p, kl, p.Burg, k)
			}
		case strings.Contains(cause, "duel"), strings.Contains(cause, "gallows"), strings.Contains(cause, "war"), strings.Contains(cause, "old age"), strings.Contains(cause, "slain"):
		default:
			age, life := p.lvAge(lvYear(g)), lsLifespan(p.Species)
			r := NewRng(g.Seed + ":mx:dead:" + k)
			if age*10 < life*8 && (len(p.Rivals) > 0 || p.Wealth >= 0) && r.Chance(0.45) {
				kind := "poison"
				if strings.Contains(cause, "accident") {
					kind = "accident"
				} else if r.Chance(0.25) {
					kind = "murder"
				}
				try(kind, p, nil, p.Burg, k)
				if made != nil && made.Victim == p.Key {
					lvSt(g).addNews(day, p.Burg, "Whispers in %s: the death of %s was no natural thing.", g.knBurgName(p.Burg), p.Name)
				}
			}
		}
	}
	// 2. crimes at large
	if st.Base && !catch {
		ckeys := make([]string, 0, len(ls.Crim))
		for k := range ls.Crim {
			ckeys = append(ckeys, k)
		}
		sort.Strings(ckeys)
		for _, k := range ckeys {
			cd := ls.Crim[k]
			tag := fmt.Sprintf("c:%s:%d", k, cd)
			if st.Seen[tag] || day-cd > 3 {
				continue
			}
			st.Seen[tag] = true
			perp := lvGet(g, k)
			if perp == nil || !perp.Alive || !near(perp.Burg) {
				continue
			}
			for _, vk := range keys {
				v := people[vk]
				if v.Alive && v.Burg == perp.Burg && lsHas(v.Rivals, perp.Key) && len(v.Life) > 0 && strings.Contains(v.Life[len(v.Life)-1].Text, perp.Name) {
					try("theft", v, perp, v.Burg, tag)
					break
				}
			}
		}
	}
	// 3. events that happen of themselves in the burgs around the player
	if st.Base && !catch && made == nil && mxOpenCount(st) < mxMaxOpen {
		var burgs []int
		if pb > 0 {
			burgs = append(burgs, pb)
		}
		for _, b := range g.ws().burgList(g) {
			if b.ID != pb && b.Alive && wsCheb(g.W, b.P, pp) <= mxNearP && len(burgs) < 6 {
				burgs = append(burgs, b.ID)
			}
		}
		for _, bid := range burgs {
			tag := fmt.Sprintf("r:%d:%d", bid, day)
			if st.Seen[tag] {
				continue
			}
			st.Seen[tag] = true
			r := NewRng(g.Seed + ":mx:" + tag)
			if !r.Chance(0.04) {
				continue
			}
			lvEnsureTown(g, bid)
			kind := []string{"arson", "kidnap", "forgery", "shipment", "theft", "poison"}[r.Int(0, 5)]
			v := mxPickVictim(g, kind, bid, r)
			if v == nil {
				continue
			}
			try(kind, v, nil, bid, tag)
			if made != nil {
				break
			}
		}
	}
	st.Base = true
	if made != nil {
		g.Msg("{{R|A case has opened in " + g.knBurgName(made.Burg) + ": " + made.Title + ".}} {{K|(Casebook)}}")
	}
	for k := range st.Seen {
		if strings.HasPrefix(k, "r:") && len(st.Seen) > 400 {
			delete(st.Seen, k)
		}
	}
}

func mxPickVictim(g *Game, kind string, burg int, r *Rng) *lvPerson {
	var pool []*lvPerson
	for _, p := range lvPresent(g, burg) {
		if p.Role == "child" || p.lvAge(lvYear(g)) < lsAdultAge(p.Species) {
			continue
		}
		switch kind {
		case "arson", "theft", "shipment":
			if lsShopJobs[p.Job] || p.Role == "merchant" {
				pool = append(pool, p)
			}
		case "forgery":
			if lsIsNotable(p) || p.Wealth >= 60 {
				pool = append(pool, p)
			}
		default:
			pool = append(pool, p)
		}
	}
	if len(pool) == 0 {
		return nil
	}
	return pool[r.Int(0, len(pool)-1)]
}

func mxGoCold(g *Game, c *mxCase) {
	c.Status = "cold"
	c.Outcome = "The trail went cold."
	day := lvDay(g)
	k := lvGet(g, c.Culprit)
	if k != nil && k.Alive {
		k.addLife(lvYear(g), "crime", "Was never called to account for the "+mxKinds[c.Kind].Noun+" of "+c.VName+".")
		lvSt(g).addNews(day, c.Burg, "The %s of %s was never solved; whispers say %s knows more than %s lets on.", mxKinds[c.Kind].Noun, c.VName, k.Name, mxHe(k))
	} else {
		lvSt(g).addNews(day, c.Burg, "The %s of %s was never solved.", mxKinds[c.Kind].Noun, c.VName)
	}
	if v := lvGet(g, c.Victim); v != nil && c.Kind == "kidnap" && v.Alive {
		v.addLife(lvYear(g), "crime", "Escaped from captivity.")
	}
	g.Msg("{{K|The case \"" + c.Title + "\" has gone cold.}}")
}

func mxHe(p *lvPerson) string {
	if p.Female {
		return "she"
	}
	return "he"
}

// ---------------------------------------------------------------- building a case

type mxCand struct {
	p    *lvPerson
	rel  string
	why  string
	mot  string
	cat  string
	w    int
	used bool
}

// mxRelOf says what p is to the victim and the motive that fits.
func mxRelOf(v, p *lvPerson, burg int) (rel, motive string) {
	switch {
	case v.Spouse == p.Key:
		return "spouse", "love"
	case lsHas(v.Kin, p.Key):
		if l := v.Rel[p.Key]; l != "" {
			return "kin", "greed"
		}
		return "kin", "greed"
	case lsHas(v.Rivals, p.Key) || lsHas(p.Rivals, v.Key) || p.GoalID == "revenge":
		return "rival", "revenge"
	case p.Boss == v.Key:
		return "servant", "fear"
	case v.Boss == p.Key:
		return "employer", "silence"
	case lsHas(v.Friends, p.Key):
		return "friend", "love"
	case p.Home != burg:
		return "outsider", "silence"
	case p.Wealth < 25:
		return "debtor", "greed"
	}
	return "acquaintance", "silence"
}

var mxRelWeight = map[string]int{"rival": 6, "spouse": 3, "kin": 3, "servant": 3, "employer": 2, "friend": 2, "debtor": 3, "outsider": 2, "acquaintance": 1}

// mxBuild makes a case from a real victim, an optional real culprit, in a burg.
func mxBuild(g *Game, kind string, victim, culprit *lvPerson, burg int, tag string) *mxCase {
	kd := mxKinds[kind]
	if kd == nil || victim == nil || burg <= 0 {
		return nil
	}
	st := mxSt(g)
	r := NewRng(fmt.Sprintf("%s:mx:%s:%s:%d", g.Seed, kind, victim.Key, lvDay(g)))
	sk := mxSkinFor(g)
	yr := lvYear(g)
	var cands []*mxCand
	for _, p := range lvPresent(g, burg) {
		if p.Key == victim.Key || p.Role == "child" || p.lvAge(yr) < lsAdultAge(p.Species) {
			continue
		}
		rel, mot := mxRelOf(victim, p, burg)
		w := mxRelWeight[rel]
		if p.GoalID == "revenge" && lsHas(p.Rivals, victim.Key) {
			w += 4
		}
		if kind == "poison" && (p.Job == "apothecary" || p.Job == "herbalist") {
			w += 3
		}
		cands = append(cands, &mxCand{p: p, rel: rel, mot: mot, w: w})
	}
	// a stranger passing through can also be a suspect: people on the road are not present, so use outsiders by Home
	if len(cands) < 5 {
		return nil
	}
	sort.Slice(cands, func(i, j int) bool { return cands[i].p.Key < cands[j].p.Key })
	var cc *mxCand
	if culprit != nil {
		for _, c := range cands {
			if c.p.Key == culprit.Key {
				cc = c
			}
		}
		if cc == nil {
			rel, mot := mxRelOf(victim, culprit, burg)
			cc = &mxCand{p: culprit, rel: rel, mot: mot}
			cands = append(cands, cc)
		}
	} else {
		tot := 0
		for _, c := range cands {
			tot += c.w
		}
		n := r.Int(0, maxi(tot-1, 0))
		for _, c := range cands {
			if n < c.w {
				cc = c
				break
			}
			n -= c.w
		}
		if cc == nil {
			cc = cands[0]
		}
	}
	cc.used = true
	c := &mxCase{ID: st.Next + 1, Kind: kind, Burg: burg, Day: lvDay(g), Victim: victim.Key, VName: victim.Name, Culprit: cc.p.Key, Status: "open",
		Hour: (r.Int(20, 26)) % 24, Title: fmt.Sprintf(kd.Title, victim.Name), Tried: map[string]int{}}
	// motive
	c.Motive = cc.mot
	if r.Chance(0.2) && victim.Secret != "" {
		c.Motive = "silence"
	}
	if kind == "forgery" || kind == "shipment" || kind == "theft" {
		if r.Chance(0.7) {
			c.Motive = "greed"
		}
	}
	mt := mxMotiveTxt[c.Motive]
	c.MotiveT = fmt.Sprintf(mt[r.Int(0, len(mt)-1)], cc.p.Name, victim.Name)
	// means
	switch kind {
	case "poison":
		bad := []string{"poison", "paralysis", "sleep", "hallucination"}
		c.Means = bad[r.Int(0, len(bad)-1)]
	case "murder":
		c.Means = sk.Weapons[r.Int(0, 2)]
	case "accident":
		c.Means = "a tampered " + []string{"beam", "stair", "rope", "cart axle", "scaffold"}[r.Int(0, 4)]
	case "theft":
		c.Means = "a picked lock"
	case "arson":
		c.Means = "lamp oil"
	case "kidnap":
		c.Means = "a sack and a waiting cart"
	case "forgery":
		c.Means = "a copied hand"
	case "shipment":
		c.Means = "an altered manifest"
	}
	// suspects: the culprit, then others of different sorts
	want := r.Int(4, 6)
	if want > len(cands) {
		want = len(cands)
	}
	var picked []*mxCand
	picked = append(picked, cc)
	cats := []string{"kin", "spouse", "rival", "friend", "servant", "employer", "debtor", "outsider", "acquaintance"}
	mxShuffle(r, cats)
	for _, cat := range cats {
		if len(picked) >= want {
			break
		}
		for _, cd := range cands {
			if !cd.used && cd.rel == cat {
				cd.used = true
				picked = append(picked, cd)
				break
			}
		}
	}
	for _, cd := range cands {
		if len(picked) >= want {
			break
		}
		if !cd.used {
			cd.used = true
			picked = append(picked, cd)
		}
	}
	// witnesses: people who are not suspects
	var wit []*lvPerson
	for _, cd := range cands {
		if !cd.used {
			wit = append(wit, cd.p)
		}
	}
	if len(wit) < 3 {
		return nil
	}
	mxShuffle(r, wit)
	witE, witA, witL := wit[0], wit[1], wit[2]
	var wx []*lvPerson
	if len(wit) > 3 {
		wx = wit[3:]
	}
	order := make([]int, len(picked))
	for i := range order {
		order[i] = i
	}
	mxShuffle(r, order)
	var herr []*mxSusp
	for _, i := range order {
		cd := picked[i]
		s := &mxSusp{Key: cd.p.Key, Name: cd.p.Name, Rel: cd.rel}
		tx := mxRelTxt[cd.rel]
		s.Why = fmt.Sprintf(tx[r.Int(0, len(tx)-1)], cd.p.Name)
		place := mxAlibiPlaces[r.Int(0, len(mxAlibiPlaces)-1)]
		if cd == cc {
			s.Why = c.MotiveT
			if r.Chance(0.5) { // the public reason is vaguer than the truth
				tx := mxRelTxt[cd.rel]
				s.Why = fmt.Sprintf(tx[r.Int(0, len(tx)-1)], cd.p.Name)
			}
			s.Alibi = fmt.Sprintf("was %s with %s until well past %s", place, witA.Name, mxHourWord(c.Hour+1))
			s.Witness = witA.Key
		} else {
			s.AlibiOK = r.Chance(0.5)
			if s.AlibiOK && len(wx) > 0 {
				w := wx[r.Int(0, len(wx)-1)]
				s.Witness = w.Key
				s.Alibi = fmt.Sprintf("was %s, where %s saw them", place, w.Name)
			} else if s.AlibiOK {
				s.Alibi = "was " + place + " among a crowd"
				s.Witness = ""
			} else {
				s.Alibi = "was " + place + " alone, and can prove nothing"
			}
		}
		c.Suspect = append(c.Suspect, s)
	}
	// red herrings: 1-2 innocent suspects with an unrelated secret and a shaky alibi
	nh := r.Int(1, 2)
	for _, s := range c.Suspect {
		if nh == 0 {
			break
		}
		if s.Key == c.Culprit {
			continue
		}
		p := lvGet(g, s.Key)
		s.Herring = true
		s.Secret = "has been secretly " + strings.TrimPrefix(mxSecrets[r.Int(0, len(mxSecrets)-1)], "has been ")
		if p != nil && p.Secret != "" && r.Chance(0.5) {
			s.Secret = p.Secret
		}
		s.AlibiOK = false
		s.Alibi = "was at home, and will say no more"
		s.Witness = ""
		herr = append(herr, s)
		nh--
	}
	kp := cc.p
	vn := victim.Name
	hour := c.Hour
	th := func(h int) int {
		if h < 12 {
			return h + 24
		}
		return h
	}
	add := func(cl *mxClue) *mxClue {
		cl.ID = len(c.Clues) + 1
		c.Clues = append(c.Clues, cl)
		return cl
	}
	build := "a broad figure"
	if kp.Female {
		build = "a slight figure"
	}
	// ---- the core clues the culprit cannot escape: means, opportunity (broken alibi), motive
	meansTxt := mxMeansText(kind, sk, c, kp.Name, vn, mxPoisonName(c.Means), mxJobWord(kp))
	if kind == "poison" {
		add(&mxClue{Kind: "phys", Method: "tox", Text: meansTxt, For: kp.Key, Aspect: "means", Rel: 2, DC: 10})
	} else {
		add(&mxClue{Kind: "phys", Method: "search", Text: meansTxt, For: kp.Key, Aspect: "means", Rel: 2, DC: 9})
	}
	add(&mxClue{Kind: "test", Method: "talk", Src: witA.Key, Text: fmt.Sprintf("%s admits it: %s left well before %s, and was asked to say otherwise.", witA.Name, kp.Name, mxHourWord(hour)),
		For: kp.Key, Aspect: "opp", Rel: 3, DC: 9, TL: fmt.Sprintf("%s: %s slipped away from %s.", mxHourWord(hour-1), kp.Name, witA.Name), TH: th(hour - 1)})
	docs := mxDocTxt[c.Motive]
	add(&mxClue{Kind: "doc", Method: "papers", Text: fmt.Sprintf(docs[r.Int(0, len(docs)-1)], kp.Name, vn, kd.Noun), For: kp.Key, Aspect: "motive", Rel: 3, DC: 10})
	// ---- supporting clues
	n := 0
	if r.Chance(0.75) {
		add(&mxClue{Kind: "test", Method: "talk", Src: witE.Key, Text: fmt.Sprintf("%s saw %s hurry from the place near %s, face hidden.", witE.Name, build, mxHourWord(hour)),
			For: kp.Key, Aspect: "id", Rel: 2, DC: 9, TL: fmt.Sprintf("%s: %s seen leaving the scene.", mxHourWord(hour), build), TH: th(hour)})
		n++
	}
	if r.Chance(0.8) || n == 0 {
		add(&mxClue{Kind: "phys", Method: "vis", Text: fmt.Sprintf("A %s, dropped near the door; it bears %s's mark.", sk.Mark, kp.Name), For: kp.Key, Aspect: "id", Rel: 2, DC: 0})
	}
	if r.Chance(0.6) {
		tread := "broad"
		if kp.Female {
			tread = "narrow"
		}
		add(&mxClue{Kind: "phys", Method: "tracks", Text: "A " + tread + " tread leads away from the scene toward the quarter where " + kp.Name + " lives.", For: kp.Key, Aspect: "id", Rel: 2, DC: 10})
	}
	// time of death or event: clears one suspect with a true alibi
	var safe *mxSusp
	for _, s := range c.Suspect {
		if s.AlibiOK && s.Witness != "" {
			safe = s
			break
		}
	}
	timeTxt := fmt.Sprintf("By the stiffness and the cold, the %s was about %s.", kd.Noun, mxHourWord(hour))
	if !kd.Body {
		timeTxt = fmt.Sprintf("A stopped clock, a cooling hearth: the %s happened about %s.", kd.Noun, mxHourWord(hour))
	}
	tc := &mxClue{Kind: "phys", Method: "body", Text: timeTxt, Aspect: "opp", Rel: 2, DC: 9, TL: mxHourWord(hour) + ": the " + kd.Noun + " itself.", TH: th(hour)}
	if safe != nil {
		tc.Clear = safe.Key
		tc.Text += " " + safe.Name + " was elsewhere then, in plain view."
	}
	add(tc)
	// red herring clues
	for _, h := range herr {
		hn := h.Name
		add(&mxClue{Kind: "doc", Method: "papers", Text: fmt.Sprintf("A torn note, \"...tell no one. Come by the back way after dark. -%s\".", string([]rune(hn)[:1])), For: h.Key, Aspect: "id", Rel: 1, DC: 9, Herring: true})
		add(&mxClue{Kind: "test", Method: "press", Src: h.Key, Text: fmt.Sprintf("Pressed, %s admits they %s, and were nowhere near the %s.", hn, strings.TrimPrefix(h.Secret, "has been "), kd.Noun), Clear: h.Key, Aspect: "opp", Rel: 3, DC: 10, Herring: true,
			TL: fmt.Sprintf("%s: %s was busy with a secret of their own.", mxHourWord(hour), hn)})
		break
	}
	// a malicious witness blames an innocent
	var victimOfLie *mxSusp
	for _, s := range c.Suspect {
		if s.Key != c.Culprit && !s.Herring {
			victimOfLie = s
			break
		}
	}
	if victimOfLie == nil && len(herr) > 0 {
		victimOfLie = herr[0]
	}
	if victimOfLie != nil && r.Chance(0.7) {
		add(&mxClue{Kind: "test", Method: "talk", Src: witL.Key, Text: fmt.Sprintf("%s swears they saw %s near the place that night, looking guilty.", witL.Name, victimOfLie.Name), For: victimOfLie.Key, Aspect: "id", Rel: 1, Lie: true, DC: 8})
	}
	// a testimony that clears another suspect for the record
	if safe != nil && len(c.Clues) < 9 {
		if w := lvGet(g, safe.Witness); w != nil {
			add(&mxClue{Kind: "test", Method: "talk", Src: w.Key, Text: fmt.Sprintf("%s vouches for %s: they were together at the time, and %s saw it too.", w.Name, safe.Name, mxHe(w)), Clear: safe.Key, Aspect: "opp", Rel: 3, DC: 8})
		}
	}
	for len(c.Clues) > 10 {
		c.Clues = c.Clues[:len(c.Clues)-1]
	}
	for i, cl := range c.Clues {
		cl.ID = i + 1
	}
	// scene
	sc := kd.Scenes[r.Int(0, len(kd.Scenes)-1)]
	c.Scene = mxScene{Name: "the " + sk.Scenes[sc], Kind: sc}
	st.Next = c.ID
	st.Cases = append(st.Cases, c)
	// news and the victim's life
	day := lvDay(g)
	switch kind {
	case "arson":
		lvSt(g).addNews(day, burg, "Fire gutted the property of %s in %s; arson is suspected.", vn, g.knBurgName(burg))
	case "theft":
		lvSt(g).addNews(day, burg, "%s of %s was robbed in the night; %s has no suspect yet.", vn, g.knBurgName(burg), mxSkinFor(g).Watch)
	case "shipment":
		lvSt(g).addNews(day, burg, "A shipment bound for %s of %s has gone missing.", vn, g.knBurgName(burg))
	case "forgery":
		lvSt(g).addNews(day, burg, "A will naming %s's heirs is called a forgery in %s.", vn, g.knBurgName(burg))
	case "kidnap":
		lvSt(g).addNews(day, burg, "%s of %s has vanished; a ransom is feared.", vn, g.knBurgName(burg))
		ls := lsSt(g)
		victim.Burg, victim.Dest, victim.ArriveD = 0, victim.Home, day+28
		ls.Jail[victim.Key] = victim.ArriveD
		ls.From[victim.Key] = burg
		victim.addLife(yr, "crime", "Was carried off by an unknown hand.")
	}
	if kd.Body {
		victim.addLife(yr, "crime", "Died under suspicious circumstances.")
	}
	mxFirst(g, "open", "The wanderer took on the case of \""+c.Title+"\".")
	return c
}

func mxPoisonName(prop string) string {
	if d := alPropByID[prop]; d != nil {
		return d.Name
	}
	return prop
}

func mxJobWord(p *lvPerson) string {
	if p.Job != "" {
		return p.Job
	}
	return "townsfolk"
}

func mxMeansText(kind string, sk mxSkin, c *mxCase, k, v, poison, job string) string {
	switch kind {
	case "murder":
		return fmt.Sprintf("A %s wiped but not clean, thrust under %s. The fittings are the sort a %s keeps, and %s is the %s hereabouts.", c.Means, mxSpots[c.ID%len(mxSpots)], job, k, job)
	case "poison":
		return fmt.Sprintf("In the %s, traces of %s: a brewer's work, bought from a herbalist. %s was seen at the herbalist's stall this week.", sk.Poison, poison, k)
	case "accident":
		return fmt.Sprintf("The \"accident\" was staged: %s, cut through. Sawdust on %s's sleeve fits the saw in their shed.", c.Means, k)
	case "theft":
		return fmt.Sprintf("The lock was opened with a %s, not forced: practised work, such as %s is known for.", sk.Tool, k)
	case "arson":
		return fmt.Sprintf("Shards of a lamp-oil flask lie in the ash; a flask of that make was sold to %s.", k)
	case "kidnap":
		return fmt.Sprintf("Drag marks, a spilled sack and wheel-ruts from a cart of the sort %s drives.", k)
	case "forgery":
		return fmt.Sprintf("The ink on the will is fresher than the parchment, and the flourish matches %s's chancery hand.", k)
	}
	return fmt.Sprintf("The manifest was altered after sealing, in a hand like %s's.", k)
}

// mxSkinFor picks the genre skin.
func mxSkinFor(g *Game) mxSkin {
	if s, ok := mxSkins[g.Genre]; ok {
		return s
	}
	return mxSkins["fantasy"]
}

// ---------------------------------------------------------------- evidence arithmetic

func mxFoundClues(c *mxCase) []*mxClue {
	var out []*mxClue
	for _, cl := range c.Clues {
		if cl.Found {
			out = append(out, cl)
		}
	}
	return out
}

// mxStrength counts found (or presented) clues pointing at a suspect: how many, their aspects, the weight, the clearing weight.
func mxStrength(c *mxCase, key string, only map[int]bool) (n int, asp map[string]bool, weight, clear int) {
	asp = map[string]bool{}
	for _, cl := range c.Clues {
		if !cl.Found || (only != nil && !only[cl.ID]) {
			continue
		}
		if cl.For == key {
			n++
			weight += cl.Rel
			if !cl.Lie {
				asp[cl.Aspect] = true
			}
		}
		if cl.Clear == key {
			clear += cl.Rel
		}
	}
	return
}

// mxSufficient: at least three consistent clues (four after a deflected accusation) covering motive, means and opportunity.
func mxSufficient(c *mxCase, key string, only map[int]bool) bool {
	n, asp, _, clear := mxStrength(c, key, only)
	need := 3
	if c.Blamed {
		need = 4
	}
	return n >= need && asp["motive"] && asp["means"] && asp["opp"] && clear < 3
}

func mxStatus(c *mxCase, s *mxSusp) string {
	_, _, w, clear := mxStrength(c, s.Key, nil)
	switch {
	case clear >= 3 && clear > w:
		return "cleared"
	case mxSufficient(c, s.Key, nil) || w >= 6:
		return "likely"
	}
	return "suspect"
}

// mxFound records a clue as found, with a message.
func mxFound(g *Game, c *mxCase, cl *mxClue, how string) {
	if cl.Found {
		return
	}
	cl.Found = true
	g.Msg("{{c|Clue: " + cl.Text + "}}")
	if how != "" {
		g.Msg("{{K|" + how + "}}")
	}
	g.gainXP(3 + cl.DC)
	if c.Kind == "poison" && cl.Method == "tox" {
		mxFirst(g, "poison", "The wanderer named a poison from a corpse and began to see how it was done.")
	}
	mxFirst(g, "clue", "The wanderer found a first clue in the case of \""+c.Title+"\".")
}

func mxShuffle[T any](r *Rng, a []T) {
	for i := len(a) - 1; i > 0; i-- {
		j := r.Int(0, i)
		a[i], a[j] = a[j], a[i]
	}
}

// ---------------------------------------------------------------- the scene of the crime

func mxEnter(g *Game, zn *Zone) {
	if g == nil || zn == nil || g.W == nil || zn.Kind != "town" || zn.Burg <= 0 || zn.Z != 0 {
		return
	}
	st := mxSt(g)
	for _, c := range st.Cases {
		if c.Scene.Placed && c.Scene.ZoneKey == zn.Key && !c.open() && !c.Scene.Clean {
			mxClean(zn, c)
			continue
		}
		if !c.open() || c.Burg != zn.Burg || c.Scene.Placed {
			continue
		}
		mxPlace(g, zn, c)
	}
}

func mxFree(zn *Zone, x, y int) bool {
	if !zn.In(x, y) || !zn.At(x, y).D().Walk || zn.At(x, y).D().Water > 0 || zn.At(x, y) == TDown || zn.At(x, y) == TUp {
		return false
	}
	for _, a := range zn.Actors {
		if a.X == x && a.Y == y {
			return false
		}
	}
	for _, it := range zn.Items {
		if it.X == x && it.Y == y {
			return false
		}
	}
	return true
}

func mxIndoor(t Terrain) bool {
	return t == TFloor || t == TWoodFloor || t == TTileFloor || t == TCarpet
}

func mxNearFree(zn *Zone, x, y, rad int, r *Rng) (int, int, bool) {
	var c [][2]int
	for dy := -rad; dy <= rad; dy++ {
		for dx := -rad; dx <= rad; dx++ {
			if mxFree(zn, x+dx, y+dy) {
				c = append(c, [2]int{x + dx, y + dy})
			}
		}
	}
	if len(c) == 0 {
		return 0, 0, false
	}
	p := c[r.Int(0, len(c)-1)]
	return p[0], p[1], true
}

// mxPlace builds the scene in the first town zone of the burg the player enters: a body or the marks of the crime,
// blood, visible clue props. Side rng only.
func mxPlace(g *Game, zn *Zone, c *mxCase) {
	r := NewRng(fmt.Sprintf("%s:mx:scene:%d:%s", g.Seed, c.ID, zn.Key))
	indoor := c.Scene.Kind != mxAlley
	var cand [][2]int
	for y := 2; y < MH-2; y++ {
		for x := 2; x < MW-2; x++ {
			t := zn.At(x, y)
			if !mxFree(zn, x, y) {
				continue
			}
			ok := false
			if indoor {
				ok = mxIndoor(t)
			} else {
				ok = (t == TCobble || t == TRoad || t == TDirt) && (!zn.At(x-1, y).D().Walk || !zn.At(x+1, y).D().Walk || !zn.At(x, y-1).D().Walk || !zn.At(x, y+1).D().Walk)
			}
			if ok {
				free := 0
				for _, d := range [][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
					if mxFree(zn, x+d[0], y+d[1]) {
						free++
					}
				}
				if free >= 2 {
					cand = append(cand, [2]int{x, y})
				}
			}
		}
	}
	if len(cand) == 0 {
		for y := 2; y < MH-2; y++ {
			for x := 2; x < MW-2; x++ {
				if mxFree(zn, x, y) && zn.At(x, y).D().Walk {
					cand = append(cand, [2]int{x, y})
				}
			}
		}
	}
	if len(cand) == 0 {
		c.Scene.Placed, c.Scene.ZoneKey = true, zn.Key
		return
	}
	pt := cand[r.Int(0, len(cand)-1)]
	cx, cy := pt[0], pt[1]
	c.Scene.Placed, c.Scene.ZoneKey, c.Scene.X, c.Scene.Y = true, zn.Key, cx, cy
	c.Scene.Under = zn.At(cx, cy)
	kd := mxKinds[c.Kind]
	idx := cy*MW + cx
	if zn.Notes == nil {
		zn.Notes = map[int]string{}
	}
	switch {
	case kd.Body:
		zn.Set(cx, cy, TBones)
		zn.Notes[idx] = "The body of " + c.VName + " lies here, grown cold. Something was done to them, and not by chance."
	case c.Kind == "arson":
		zn.Set(cx, cy, TAsh)
		zn.Notes[idx] = "Scorched boards and ash. The fire started here, and not by chance."
	case c.Kind == "theft", c.Kind == "shipment":
		zn.Set(cx, cy, TRubble)
		zn.Notes[idx] = "A broken lock, a ransacked strongbox: " + c.VName + "'s goods are gone."
	case c.Kind == "kidnap":
		zn.Set(cx, cy, TRubble)
		zn.Notes[idx] = "An overturned chair and torn cloth: there was a struggle here."
	default:
		zn.Set(cx, cy, TRubble)
		zn.Notes[idx] = "A writing desk, ransacked: the will was swapped here."
	}
	// blood or soot around
	if kd.Body || c.Kind == "kidnap" {
		for _, d := range [][2]int{{0, 0}, {1, 0}, {0, 1}, {-1, 0}, {1, 1}} {
			x, y := cx+d[0], cy+d[1]
			if zn.In(x, y) && zn.At(x, y).D().Walk && len(zn.Liq) > y*MW+x && r.Chance(0.8) {
				zn.Liq[y*MW+x] = 1
			}
		}
	}
	sk := mxSkinFor(g)
	for _, cl := range c.Clues {
		x, y, ok := mxNearFree(zn, cx, cy, 3, r)
		if !ok {
			x, y = cx, cy
		}
		cl.X, cl.Y = x, y
		if cl.Method != "vis" || cl.Found {
			continue
		}
		name := "a " + sk.Mark
		zn.Items = append(zn.Items, &Item{ID: g.newID(), Name: name, Glyph: '?', F: CYellow, Kind: "letter", Text: cl.Text, X: x, Y: y, Weight: 0,
			Lore: "Left at the scene of " + c.Title + ".", Extra: map[string]int{"mx": c.ID, "mxc": cl.ID}})
	}
	g.Msg("{{R|You come upon the scene of a crime:}} " + c.Scene.Name + ". {{K|(Examine the scene from the Abilities list.)}}")
}

// mxClean removes the props once a case is closed.
func mxClean(zn *Zone, c *mxCase) {
	c.Scene.Clean = true
	if zn.At(c.Scene.X, c.Scene.Y) != c.Scene.Under && c.Scene.Under != 0 {
		zn.Set(c.Scene.X, c.Scene.Y, c.Scene.Under)
	}
	delete(zn.Notes, c.Scene.Y*MW+c.Scene.X)
	for dy := -1; dy <= 1; dy++ {
		for dx := -1; dx <= 1; dx++ {
			x, y := c.Scene.X+dx, c.Scene.Y+dy
			if zn.In(x, y) && len(zn.Liq) > y*MW+x && zn.Liq[y*MW+x] == 1 {
				zn.Liq[y*MW+x] = 0
			}
		}
	}
	var keep []*Item
	for _, it := range zn.Items {
		if it.Extra != nil && it.Extra["mx"] == c.ID {
			continue
		}
		keep = append(keep, it)
	}
	zn.Items = keep
}

// mxSceneCase: the open case whose scene the player stands in.
func mxSceneCase(g *Game) *mxCase {
	if g == nil || g.Cur == nil || g.W == nil {
		return nil
	}
	for _, c := range mxSt(g).Cases {
		if c.open() && c.Scene.Placed && c.Scene.ZoneKey == g.Cur.Key {
			return c
		}
	}
	return nil
}

type mxMethod struct {
	ID, Label, Check string
}

func mxMethods(c *mxCase) []mxMethod {
	body := "Examine the body for how and when"
	if !mxKinds[c.Kind].Body {
		body = "Date the event: the cold hearth, the clock"
	}
	m := []mxMethod{
		{"search", "Search the room for what was left", "perception"},
		{"body", body, "lore"},
		{"papers", "Go through the papers and letters", "perception"},
		{"tracks", "Read the tracks and blood trail", "survival"},
	}
	if c.Kind == "poison" {
		m = append(m, mxMethod{"tox", "Test the dregs and the dead (herbalism, Lore, Tinkering)", "lore"})
	} else if c.Kind == "arson" {
		m = append(m, mxMethod{"tox", "Smell and test the ash for an accelerant", "tinkering"})
	}
	return m
}

// mxMod: the check modifier for a method (the toxicology one takes the best of three).
func mxMod(g *Game, m mxMethod) int {
	mod := g.checkMod(m.Check)
	if m.ID == "tox" {
		if t := g.checkMod("tinkering"); t > mod {
			mod = t
		}
		if h := g.checkMod("lore") + 2*g.al().Rank; h > mod {
			mod = h
		}
	}
	return mod
}

// mxExamine runs one examination method at the scene.
func (g *Game) mxExamine(c *mxCase, m mxMethod) []string {
	if t, ok := c.Tried[m.ID]; ok && g.Turn-t < 30 {
		return []string{"{{K|You have just tried this. Give it half an hour, or come at it differently.}}"}
	}
	near := func(cl *mxClue) bool {
		return cl.X == 0 && cl.Y == 0 || absi(cl.X-g.P.X) <= 9 && absi(cl.Y-g.P.Y) <= 9
	}
	if absi(c.Scene.X-g.P.X) > 12 || absi(c.Scene.Y-g.P.Y) > 12 {
		return []string{"{{K|You are too far from the scene to see anything. Go to the place itself.}}"}
	}
	var pend []*mxClue
	for _, cl := range c.Clues {
		if !cl.Found && cl.Method == m.ID && near(cl) {
			pend = append(pend, cl)
		}
	}
	// visible props within reach come along with a search
	if m.ID == "search" {
		var keep []*Item
		for _, it := range g.Cur.Items {
			if it.Extra != nil && it.Extra["mx"] == c.ID && absi(it.X-g.P.X) <= 4 && absi(it.Y-g.P.Y) <= 4 {
				if cl := c.clue(it.Extra["mxc"]); cl != nil && !cl.Found {
					mxFound(g, c, cl, "")
				}
				continue
			}
			keep = append(keep, it)
		}
		g.Cur.Items = keep
	}
	if len(pend) == 0 {
		c.Tried[m.ID] = g.Turn
		return []string{"{{K|You find nothing more this way.}}"}
	}
	mod := mxMod(g, m)
	roll := g.R.Dice(2, 6) + mod
	var out []string
	got := 0
	for _, cl := range pend {
		if roll >= cl.DC {
			mxFound(g, c, cl, "")
			out = append(out, "{{G|"+cl.Text+"}}")
			got++
		}
	}
	if got == 0 {
		c.Tried[m.ID] = g.Turn
		out = []string{fmt.Sprintf("{{R|[%s: failure]}} You sense something is here, but cannot find it. Try again later.", checkKinds[m.Check].label)}
	} else {
		out = append([]string{fmt.Sprintf("{{G|[%s: success]}}", checkKinds[m.Check].label)}, out...)
	}
	return out
}

func absi(a int) int {
	if a < 0 {
		return -a
	}
	return a
}

// ---------------------------------------------------------------- talk: witnesses, suspects, magistrates

func (g *Game) mxShow(text string, opts []Option) {
	g.Popup = Wrap(text, 100)
	g.Opts = append(opts, Option{"Back.", func() { g.talkMenu("\"Anything else?\"") }})
	g.Sel = 0
}

func mxTalk(g *Game, a *Actor) []Option {
	if g == nil || a == nil || g.W == nil || g.Cur == nil || a.AI == "hostile" || a.AI == "follower" {
		return nil
	}
	st := mxSt(g)
	var opts []Option
	var openHere []*mxCase
	for _, c := range st.Cases {
		if c.open() {
			openHere = append(openHere, c)
		}
	}
	if len(openHere) == 0 {
		if mxPlaceRoles[a.Role] && len(st.Cases) > 0 {
			opts = append(opts, Option{"{{c|Ask about the closed cases.}}", func() { g.mxOpen(true) }})
		}
		return opts
	}
	if mxPlaceRoles[a.Role] {
		opts = append(opts, Option{"{{c|Ask about the investigation: open the casebook.}}", func() { g.mxOpen(true) }})
	}
	p := lvPersonOf(g, a)
	if p == nil {
		return opts
	}
	day := lvDay(g)
	for _, c := range openHere {
		c := c
		if c.Burg != g.Cur.Burg {
			continue
		}
		if s := c.susp(p.Key); s != nil {
			s := s
			opts = append(opts, g.mxAskOption(a, p, c, s))
			if !s.SecretOut && s.Secret != "" || s.Key == c.Culprit {
				dc := 8 + p.Trait[1]/25
				opts = append(opts, Option{g.checkLabel("intimidate", dc) + " Lean on them about " + c.VName + ".", func() { g.mxLeanOn(a, p, c, s, dc, day) }})
			}
			if nf := len(mxFoundClues(c)); nf > 0 {
				opts = append(opts, Option{fmt.Sprintf("{{R|Accuse %s of the %s of %s.}}", p.Name, mxKinds[c.Kind].Noun, c.VName), func() { g.mxOpenAccuse(c, s.Key, true) }})
			}
		}
		for _, cl := range c.Clues {
			cl := cl
			if cl.Found || cl.Method != "talk" || cl.Src != p.Key {
				continue
			}
			opts = append(opts, g.mxWitnessOption(a, p, c, cl, day))
		}
	}
	return opts
}

func (g *Game) mxAskOption(a *Actor, p *lvPerson, c *mxCase, s *mxSusp) Option {
	dc := clampi(9-p.Opinion/15+p.Trait[1]/30, 6, 13)
	label := fmt.Sprintf("Ask where they were the night %s died.", c.VName)
	if !mxKinds[c.Kind].Body {
		label = fmt.Sprintf("Ask where they were the night of %s.", mxKinds[c.Kind].Noun)
	}
	if s.Asked {
		return Option{"{{K|Ask again where they were.}}", func() {
			g.mxShow(fmt.Sprintf("\"I told you: I %s.\" %s folds their arms.", s.Alibi, p.Name), nil)
		}}
	}
	return Option{g.checkLabel("persuasion", dc) + " " + label, func() {
		if s.AskT != 0 && g.Turn-s.AskT < 30 {
			g.mxShow("\"I have said all I will. Come back later.\"", nil)
			return
		}
		if g.roll2d6Check(g.checkMod("persuasion"), dc) {
			s.Asked = true
			g.gainXP(4 + dc)
			txt := fmt.Sprintf("\"Very well. I %s.\"", s.Alibi)
			if s.Witness != "" {
				txt += fmt.Sprintf(" They name %s as the one who can say so.", mxName(g, s.Witness))
			}
			if s.Herring && p.Trait[4] > 60 {
				txt += " {{y|They will not meet your eye.}}"
			}
			g.mxShow("{{G|[Persuasion: success]}}\n\n"+txt, nil)
			return
		}
		s.AskT = g.Turn
		p.remember(lvDay(g), "insult", "pried into a death", -3)
		g.mxShow("{{R|[Persuasion: failure]}}\n\n\"It is none of your business where I was.\"", nil)
	}}
}

func (g *Game) mxLeanOn(a *Actor, p *lvPerson, c *mxCase, s *mxSusp, dc, day int) {
	if s.AskT != 0 && g.Turn-s.AskT < 30 {
		g.mxShow("\"Leave me be. I have said what I will say.\"", nil)
		return
	}
	if g.roll2d6Check(g.checkMod("intimidate"), dc) {
		g.gainXP(5 + dc)
		p.remember(day, "insult", "leaned on me hard over a death", -6)
		s.Asked, s.Nerves = true, true
		txt := fmt.Sprintf("{{G|[Intimidate: success]}}\n\n%s sweats. \"All right! I %s.\"", p.Name, s.Alibi)
		if s.Secret != "" && !s.SecretOut {
			s.SecretOut = true
			txt += fmt.Sprintf("\n\n\"And yes: I %s. That is why I was out. But I never touched %s!\"", strings.TrimPrefix(s.Secret, "has been "), c.VName)
			for _, cl := range c.Clues {
				if cl.Method == "press" && cl.Src == s.Key && !cl.Found {
					mxFound(g, c, cl, "")
				}
			}
		} else if s.Key == c.Culprit {
			txt += "\n\n{{y|Their hands shake. They are hiding something, and it is not small.}}"
		}
		g.mxShow(txt, nil)
		return
	}
	s.AskT = g.Turn
	p.remember(day, "insult", "threatened me over a death", -14)
	a.Angry = a.Angry || p.Trait[3] < 30
	g.mxShow("{{R|[Intimidate: failure]}}\n\n\"Get out of my sight, or I call the watch.\"", nil)
}

func (g *Game) mxWitnessOption(a *Actor, p *lvPerson, c *mxCase, cl *mxClue, day int) Option {
	chk := "persuasion"
	label := fmt.Sprintf("Ask what they saw the night of %s.", mxKinds[c.Kind].Noun)
	if cl.Aspect == "opp" && cl.Clear == "" {
		label = "Ask whether they were with the one who says so that night."
		chk = "empathy"
	} else if cl.Clear != "" {
		label = "Ask whether they can vouch for someone that night."
	}
	dc := cl.DC
	return Option{g.checkLabel(chk, dc) + " " + label, func() {
		if cl.TryT != 0 && g.Turn-cl.TryT < 30 {
			g.mxShow("\"I told you already. Give me time to think.\"", nil)
			return
		}
		if g.roll2d6Check(g.checkMod(chk), dc) {
			p.remember(day, "help", "asked after the truth of a crime", 2)
			mxFound(g, c, cl, "")
			g.mxShow("{{G|["+checkKinds[chk].label+": success]}}\n\n"+cl.Text, nil)
			return
		}
		cl.TryT = g.Turn
		g.mxShow("{{R|["+checkKinds[chk].label+": failure]}}\n\n\"I... would rather not say. It is none of my affair.\"", nil)
	}}
}

// ---------------------------------------------------------------- abilities and screens

func (g *Game) mxAbilities() []Ability {
	if g.Cur == nil || g.P == nil || g.W == nil {
		return nil
	}
	st := mxSt(g)
	if len(st.Cases) == 0 {
		return nil
	}
	out := []Ability{{ID: "mx_casebook", Name: "Casebook", Target: "self"}}
	if mxSceneCase(g) != nil {
		out = append([]Ability{{ID: "mx_examine", Name: "Examine the scene", Target: "self"}}, out...)
	}
	return out
}

func (g *Game) mxAbUse(id string) {
	switch id {
	case "mx_casebook":
		g.mxOpen(false)
	case "mx_examine":
		if c := mxSceneCase(g); c != nil {
			st := mxSt(g)
			st.ui = mxUI{page: "scene", cur: indexOfCase(st, c)}
			g.Mode = "mxb"
		} else {
			g.Msg("There is no scene to examine here.")
		}
	}
}

func indexOfCase(st *mxState, c *mxCase) int {
	for i, x := range st.Cases {
		if x == c {
			return i
		}
	}
	return 0
}

func (g *Game) mxOpen(fromTalk bool) {
	st := mxSt(g)
	cur := 0
	for i, c := range st.Cases {
		if c.open() {
			cur = i
		}
	}
	st.ui = mxUI{page: "book", cur: cur, fromTalk: fromTalk}
	g.Mode = "mxb"
}

func (g *Game) mxOpenAccuse(c *mxCase, key string, fromTalk bool) {
	st := mxSt(g)
	st.ui = mxUI{page: "acc", cur: indexOfCase(st, c), fromTalk: fromTalk, picks: map[int]bool{}}
	for i, s := range c.Suspect {
		if s.Key == key {
			st.ui.sus = i
			st.ui.step = 1
		}
	}
	g.Mode = "mxb"
}

func (g *Game) mxClose() {
	st := mxSt(g)
	ft := st.ui.fromTalk
	st.ui = mxUI{}
	if ft && g.Talk != nil && g.Cur != nil {
		for _, a := range g.Cur.Actors {
			if a == g.Talk {
				g.Mode = "talk"
				g.talkMenu("\"Anything else?\"")
				return
			}
		}
	}
	g.Mode = "main"
}

func (g *Game) mxCur() *mxCase {
	st := mxSt(g)
	if st.ui.cur < 0 || st.ui.cur >= len(st.Cases) {
		return nil
	}
	return st.Cases[st.ui.cur]
}

func mxKeys(g *Game, k string) {
	st := mxSt(g)
	ui := &st.ui
	c := g.mxCur()
	if c == nil {
		g.mxClose()
		return
	}
	switch ui.page {
	case "scene":
		if k == "Escape" {
			ui.page, ui.res = "book", nil
			return
		}
		ms := mxMethods(c)
		if len(k) == 1 && k[0] >= 'a' && int(k[0]-'a') < len(ms) {
			ui.res = g.mxExamine(c, ms[k[0]-'a'])
			g.spendTurns(10)
		}
	case "acc":
		g.mxAccKeys(c, k)
	default:
		switch k {
		case "Escape":
			g.mxClose()
		case "Tab", "ArrowRight":
			ui.tab, ui.off = (ui.tab+1)%5, 0
		case "ArrowLeft":
			ui.tab, ui.off = (ui.tab+4)%5, 0
		case "ArrowUp":
			ui.off = maxi(0, ui.off-1)
		case "ArrowDown":
			ui.off++
		case "PageUp":
			ui.off = maxi(0, ui.off-8)
		case "PageDown":
			ui.off += 8
		case "[", ",":
			if len(st.Cases) > 0 {
				ui.cur, ui.off = (ui.cur+len(st.Cases)-1)%len(st.Cases), 0
			}
		case "]", ".":
			if len(st.Cases) > 0 {
				ui.cur, ui.off = (ui.cur+1)%len(st.Cases), 0
			}
		case "x":
			if mxSceneCase(g) == c {
				ui.page, ui.res = "scene", nil
			}
		case "a":
			if c.open() {
				ui.page, ui.step, ui.picks, ui.res = "acc", 0, map[int]bool{}, nil
			}
		}
	}
}

// mxTabLines: the casebook pages.
func (g *Game) mxTabLines(c *mxCase, tab int) []string {
	var l []string
	add := func(f string, a ...any) { l = append(l, fmt.Sprintf(f, a...)) }
	kd := mxKinds[c.Kind]
	switch tab {
	case 0: // overview
		col := map[string]string{"open": "Y", "solved": "G", "cold": "K", "failed": "R"}[c.Status]
		add("{{W|%s}}  {{%s|[%s]}}", c.Title, col, c.Status)
		add("{{K|Opened day %d in %s. %s}}", c.Day, g.knBurgName(c.Burg), mxDaysLeft(g, c))
		add("")
		if kd.Body {
			add("Victim: {{W|%s}}, found dead.", c.VName)
		} else {
			add("Wronged: {{W|%s}} (%s).", c.VName, kd.Noun)
		}
		add("Scene: %s. %s", c.Scene.Name, mxSceneLine(g, c))
		found := mxFoundClues(c)
		add("Clues in the casebook: {{W|%d}}.   Suspects: {{W|%d}}.", len(found), len(c.Suspect))
		add("")
		if c.Outcome != "" {
			add("{{Y|%s}}", c.Outcome)
			add("")
		}
		// what is known so far about means and motive
		var haveM, haveMo, haveO bool
		for _, cl := range found {
			switch cl.Aspect {
			case "means":
				haveM = cl.For != "" || haveM
			case "motive":
				haveMo = true
			case "opp":
				haveO = true
			}
		}
		add("Means:       %s", mxTick(haveM, "a way has been found"))
		add("Motive:      %s", mxTick(haveMo, "a reason has been found"))
		add("Opportunity: %s", mxTick(haveO, "a time and place have been found"))
		add("")
		add("{{K|Three consistent clues covering motive, means and opportunity make a case you can put to someone.}}")
		for _, m := range mxMethods(c) {
			if _, ok := c.Tried[m.ID]; ok {
				add("{{K}}Tried: %s", m.Label)
			}
		}
	case 1: // suspects
		for _, s := range c.Suspect {
			stt := mxStatus(c, s)
			col := map[string]string{"cleared": "G", "suspect": "Y", "likely": "R"}[stt]
			add("{{%s|[%s]}} {{W|%s}} ({{c|%s}})", col, stt, s.Name, s.Rel)
			add("   %s", s.Why)
			if s.Asked {
				add("   Says: they %s.", s.Alibi)
			} else {
				add("   {{K|Alibi not yet asked.}}")
			}
			if s.SecretOut {
				add("   {{y|Secret: they %s.}}", strings.TrimPrefix(s.Secret, "has been "))
			}
			n, _, w, clr := mxStrength(c, s.Key, nil)
			add("   {{K|Clues against: %d (weight %d)   clearing: %d}}", n, w, clr)
		}
	case 2: // clues
		found := mxFoundClues(c)
		if len(found) == 0 {
			add("{{K|Nothing yet. Examine the scene (Abilities), and ask people what they saw.}}")
		}
		for _, cl := range found {
			ic := map[string]string{"phys": "P", "test": "T", "doc": "D"}[cl.Kind]
			add("{{c|#%d [%s]}} %s", cl.ID, ic, cl.Text)
			tg := "   " + mxStars(cl.Rel)
			if cl.For != "" {
				tg += "  {{R|points to " + mxName(g, cl.For) + "}}"
			}
			if cl.Clear != "" {
				tg += "  {{G|clears " + mxName(g, cl.Clear) + "}}"
			}
			l = append(l, tg)
		}
	case 3: // timeline
		type ev struct {
			h int
			t string
		}
		var evs []ev
		evs = append(evs, ev{99, "{{Y|Day " + fmt.Sprint(c.Day) + ": the " + kd.Noun + " is discovered.}}"})
		for _, cl := range mxFoundClues(c) {
			if cl.TL != "" {
				evs = append(evs, ev{cl.TH, "{{c|" + cl.TL + "}}"})
			}
		}
		for _, s := range c.Suspect {
			if s.Asked {
				hh := c.Hour
				if hh < 12 {
					hh += 24
				}
				evs = append(evs, ev{hh - 1, fmt.Sprintf("{{K|Claimed:}} %s %s.", s.Name, s.Alibi)})
			}
		}
		sort.SliceStable(evs, func(i, j int) bool { return evs[i].h < evs[j].h })
		for _, e := range evs {
			l = append(l, e.t)
		}
		if len(evs) < 3 {
			add("")
			add("{{K|Date the event, ask about alibis and witnesses to fill the night in.}}")
		}
	default: // contradictions
		any := false
		for _, s := range c.Suspect {
			var imp, clr []*mxClue
			for _, cl := range mxFoundClues(c) {
				if cl.For == s.Key {
					imp = append(imp, cl)
				}
				if cl.Clear == s.Key {
					clr = append(clr, cl)
				}
			}
			if len(imp) > 0 && len(clr) > 0 {
				any = true
				add("{{R|Conflict:}} clue #%d points to {{W|%s}} but #%d clears them. One of them is wrong.", imp[0].ID, s.Name, clr[0].ID)
			}
			if s.Asked {
				for _, cl := range imp {
					if cl.Aspect == "opp" && s.Witness != "" && cl.Src == s.Witness {
						any = true
						add("{{R|False alibi:}} {{W|%s}} says they %s, but %s says otherwise (#%d).", s.Name, s.Alibi, mxName(g, cl.Src), cl.ID)
					}
				}
			}
		}
		for _, cl := range mxFoundClues(c) {
			if cl.Herring && cl.For != "" {
				for _, c2 := range mxFoundClues(c) {
					if c2.Clear == cl.For && c2.Herring {
						any = true
						add("{{y|Resolved:}} the lead in #%d (%s) was a secret, not a crime (#%d).", cl.ID, mxName(g, cl.For), c2.ID)
					}
				}
			}
		}
		if !any {
			add("{{K|No contradictions yet. Compare alibis with what witnesses say.}}")
		}
	}
	return l
}

func mxTick(ok bool, txt string) string {
	if ok {
		return "{{G|" + txt + "}}"
	}
	return "{{K|unknown}}"
}

func mxStars(n int) string {
	return "{{Y|" + strings.Repeat("*", clampi(n, 1, 3)) + "}}" + "{{K|" + strings.Repeat("*", 3-clampi(n, 1, 3)) + "}}"
}

func mxDaysLeft(g *Game, c *mxCase) string {
	if !c.open() {
		return ""
	}
	return fmt.Sprintf("The trail grows cold in %d days.", maxi(0, mxColdDay-(lvDay(g)-c.Day)))
}

func mxSceneLine(g *Game, c *mxCase) string {
	if !c.Scene.Placed {
		return "{{K}}You have not been to the scene yet."
	}
	if mxSceneCase(g) == c {
		return "{{G|You are at the scene: press x to examine.}}"
	}
	return "{{K|It lies in " + g.knBurgName(c.Burg) + ".}}"
}

func (g *Game) mxDraw() {
	st := mxSt(g)
	c := g.mxCur()
	if c == nil {
		return
	}
	x, y, w, h := g.panel("Casebook")
	s := g.sc
	ui := &st.ui
	if ui.page == "scene" {
		g.mxDrawScene(c, x, y, w, h)
		return
	}
	if ui.page == "acc" {
		g.mxDrawAcc(c, x, y, w, h)
		return
	}
	tabs := []string{"Overview", "Suspects", "Clues", "Timeline", "Contradictions"}
	tx := x
	for i, t := range tabs {
		col := "K"
		if i == ui.tab {
			col = "W"
		}
		tx = s.Text(tx, y, "{{"+col+"|"+t+"}}  ", CGray)
	}
	s.Text(x, y+1, fmt.Sprintf("{{K|Case %d of %d: [ ] to switch}}", ui.cur+1, len(st.Cases)), CGray)
	var lines []string
	for _, l := range g.mxTabLines(c, ui.tab) {
		lines = append(lines, Wrap(l, w-2)...)
	}
	room := h - 5
	if ui.off > maxi(0, len(lines)-room) {
		ui.off = maxi(0, len(lines)-room)
	}
	for i := 0; i < room && ui.off+i < len(lines); i++ {
		s.Text(x, y+3+i, lines[ui.off+i], CGray)
	}
	foot := "Tab: pages · arrows: scroll · [ ]: cases"
	if c.open() {
		foot += " · a: accuse"
	}
	if mxSceneCase(g) == c {
		foot += " · x: examine the scene"
	}
	s.Text(x, y+h-1, "{{K|"+foot+" · Esc: close}}", CGray)
}

func (g *Game) mxDrawScene(c *mxCase, x, y, w, h int) {
	s := g.sc
	st := mxSt(g)
	s.Text(x, y, "{{W|Examine the scene}}: "+c.Scene.Name, CGray)
	for i, m := range mxMethods(c) {
		dc := 9
		note := ""
		if t, ok := c.Tried[m.ID]; ok && g.Turn-t < 30 {
			note = " {{K|(just tried)}}"
		}
		s.Text(x, y+2+i, fmt.Sprintf("{{Y|%c}}) %s {{K|[%s %d%%]}}%s", 'a'+i, m.Label, checkKinds[m.Check].label, int(chanceFor(mxMod(g, m), dc)*100+0.5), note), CGray)
	}
	row := y + 3 + len(mxMethods(c))
	for _, l := range st.ui.res {
		for _, wl := range Wrap(l, w-2) {
			if row < y+h-2 {
				s.Text(x, row, wl, CGray)
				row++
			}
		}
	}
	s.Text(x, y+h-1, "{{K|Letter: try a method (10 minutes) · Esc: back to the casebook}}", CGray)
}

// ---------------------------------------------------------------- the accusation scene

func mxActorOf(g *Game, key string) *Actor {
	if g.Cur == nil {
		return nil
	}
	b := lvb(g)
	for _, a := range g.Cur.Actors {
		if k, ok := b.Bind[a.ID]; ok && k == key && a.HP > 0 {
			return a
		}
	}
	return nil
}

func mxDropActor(g *Game, a *Actor) {
	if g.Cur == nil || a == nil {
		return
	}
	var keep []*Actor
	for _, x := range g.Cur.Actors {
		if x != a {
			keep = append(keep, x)
		}
	}
	g.Cur.Actors = keep
}

func (g *Game) mxAccKeys(c *mxCase, k string) {
	st := mxSt(g)
	ui := &st.ui
	found := mxFoundClues(c)
	switch ui.step {
	case 0:
		if k == "Escape" {
			if ui.fromTalk {
				g.mxClose()
			} else {
				ui.page = "book"
			}
			return
		}
		if len(k) == 1 && k[0] >= 'a' && int(k[0]-'a') < len(c.Suspect) {
			ui.sus, ui.step, ui.picks = int(k[0]-'a'), 1, map[int]bool{}
		}
	case 1:
		switch {
		case k == "Escape":
			ui.step = 0
		case k == "Enter":
			ui.res = g.mxConfront(c, c.Suspect[ui.sus], ui.picks, false)
			ui.step = 2
		case k == "!":
			ui.res = g.mxConfront(c, c.Suspect[ui.sus], ui.picks, true)
			ui.step = 2
		case len(k) == 1 && k[0] >= 'a' && int(k[0]-'a') < len(found):
			cl := found[k[0]-'a']
			if ui.picks[cl.ID] {
				delete(ui.picks, cl.ID)
			} else if len(ui.picks) < 6 {
				ui.picks[cl.ID] = true
			}
		}
	default:
		if k == "Escape" || k == "Enter" {
			if !c.open() || ui.fromTalk {
				g.mxClose()
			} else {
				ui.page, ui.step, ui.res = "book", 0, nil
			}
		}
	}
}

func (g *Game) mxDrawAcc(c *mxCase, x, y, w, h int) {
	s := g.sc
	ui := &g.mxStUI().ui
	s.Text(x, y, "{{R|Accusation}}: "+c.Title, CGray)
	row := y + 2
	switch ui.step {
	case 0:
		s.Text(x, row, "Whom do you accuse?", CGray)
		for i, su := range c.Suspect {
			stt := mxStatus(c, su)
			col := map[string]string{"cleared": "G", "suspect": "Y", "likely": "R"}[stt]
			s.Text(x, row+2+i, fmt.Sprintf("{{Y|%c}}) %s ({{c|%s}}) {{%s|[%s]}}", 'a'+i, su.Name, su.Rel, col, stt), CGray)
		}
		s.Text(x, y+h-1, "{{K|Letter: choose · Esc: back}}", CGray)
	case 1:
		su := c.Suspect[ui.sus]
		s.Text(x, row, "You face {{W|"+su.Name+"}}. Present the evidence you want to lay out (up to 6):", CGray)
		found := mxFoundClues(c)
		if len(found) == 0 {
			s.Text(x, row+2, "{{K|You have nothing at all. Only a bluff remains.}}", CGray)
		}
		r := row + 2
		for i, cl := range found {
			mark := " "
			if ui.picks[cl.ID] {
				mark = "{{G|x}}"
			}
			tg := ""
			if cl.For != "" {
				tg = " {{R|-> " + mxName(g, cl.For) + "}}"
			}
			if cl.Clear != "" {
				tg += " {{G|clears " + mxName(g, cl.Clear) + "}}"
			}
			for j, wl := range Wrap(fmt.Sprintf("{{Y|%c}}) [%s] %s%s", 'a'+i, mark, cl.Text, tg), w-2) {
				if r < y+h-6 {
					pre := ""
					if j > 0 {
						pre = "    "
					}
					s.Text(x, r, pre+wl, CGray)
					r++
				}
			}
		}
		n, asp, _, clr := mxStrength(c, su.Key, ui.picks)
		tk := func(b bool) string {
			if b {
				return "{{G|yes}}"
			}
			return "{{R|no}}"
		}
		s.Text(x, y+h-5, fmt.Sprintf("Presented against them: {{W|%d}}   motive %s  means %s  opportunity %s", n, tk(asp["motive"]), tk(asp["means"]), tk(asp["opp"])), CGray)
		if clr > 0 {
			s.Text(x, y+h-4, "{{y|Something you present clears them.}}", CGray)
		}
		s.Text(x, y+h-2, "{{K|Letters: toggle evidence · Enter: confront · !: bluff · Esc: back}}", CGray)
	default:
		r := row
		for _, l := range ui.res {
			for _, wl := range Wrap(l, w-2) {
				if r < y+h-2 {
					s.Text(x, r, wl, CGray)
					r++
				}
			}
		}
		s.Text(x, y+h-1, "{{K|Enter: continue}}", CGray)
	}
}

func (g *Game) mxStUI() *mxState { return mxSt(g) }

// mxConfront plays the scene and returns the text to show.
func (g *Game) mxConfront(c *mxCase, s *mxSusp, picks map[int]bool, bluff bool) []string {
	p := lvGet(g, s.Key)
	a := mxActorOf(g, s.Key)
	if p == nil || !p.Alive {
		return []string{"They are gone."}
	}
	if a == nil {
		return []string{p.Name + " is not here. Find them first, then put it to them face to face."}
	}
	day := lvDay(g)
	n, _, _, clr := mxStrength(c, s.Key, picks)
	suff := mxSufficient(c, s.Key, picks)
	isC := s.Key == c.Culprit
	N := p.Trait[4]
	head := fmt.Sprintf("You lay it out before %s.", p.Name)
	if n == 0 {
		head = fmt.Sprintf("You look %s in the eye and say you know.", p.Name)
	}
	if !isC {
		return g.mxWrong(c, s, p, a, clr, bluff, head)
	}
	if !suff && !bluff {
		p.remember(day, "insult", "accused me with too little proof", -6)
		return []string{head, "", fmt.Sprintf("\"Hearsay and guesses,\" %s says, steady. \"Bring me proof, not stories.\"", p.Name), "", fmt.Sprintf("{{y|You have %d clue(s) against them; they do not yet cover motive, means and opportunity.}}", n)}
	}
	how := ""
	if !suff && bluff {
		dc := clampi(13-N/10-2*n, 5, 14)
		if !g.roll2d6Check(g.checkMod("bluff"), dc) {
			p.remember(day, "insult", "tried to bluff me over a crime", -10)
			s.Nerves = true
			return []string{head, "", "{{R|[Bluff: failure]}}", fmt.Sprintf("\"You know nothing,\" %s says, with a thin smile. \"Go and find something.\"", p.Name)}
		}
		g.gainXP(10 + dc)
		how = "{{G|[Bluff: success]}} Your certainty breaks something in them.\n"
		mxFirst(g, "bluff", "The wanderer bluffed a nervous culprit into the open.")
		return g.mxResolve(c, s, p, a, "confess", n, how, head)
	}
	r := mxReaction(g, c, p, n)
	if r == "blame" {
		c.Blamed = true
		var other string
		for _, o := range c.Suspect {
			if o.Key != c.Culprit {
				other = o.Name
				break
			}
		}
		return []string{head, "", fmt.Sprintf("\"No!\" %s throws a hand toward %s. \"Ask them where they were that night! Ask!\"", p.Name, other),
			"", "{{y|They have turned it back on another. You will need stronger proof: four consistent clues now.}}"}
	}
	return g.mxResolve(c, s, p, a, r, n, how, head)
}

// mxReaction: how the real culprit takes sufficient proof.
func mxReaction(g *Game, c *mxCase, p *lvPerson, n int) string {
	N, A, O, E := p.Trait[4], p.Trait[3], p.Trait[0], p.Trait[2]
	if !c.Blamed && n == 3 && N < 45 {
		for _, o := range c.Suspect {
			if o.Key != c.Culprit {
				return "blame"
			}
		}
	}
	switch {
	case N >= 55 || A >= 55:
		return "confess"
	case A < 35:
		return "fight"
	case O >= 55 || E >= 60:
		return "flee"
	}
	return "arrest"
}

func (g *Game) mxWrong(c *mxCase, s *mxSusp, p *lvPerson, a *Actor, clr int, bluff bool, head string) []string {
	day := lvDay(g)
	c.Wrong++
	p.remember(day, "insult", "accused me falsely of a crime", -25)
	state := 0
	if b := g.W.burgByID[c.Burg]; b != nil {
		state = b.State
	}
	g.AddRep(uwFac(state), -10, " for a false accusation")
	g.fmFame(g.fmRealm(), -4)
	mxFirst(g, "wrong", "The wanderer named the wrong person for \""+c.Title+"\".")
	out := []string{head, ""}
	if clr > 0 {
		out = append(out, fmt.Sprintf("\"You hold up the very proof that clears me!\" %s is red with anger.", p.Name))
	} else {
		out = append(out, fmt.Sprintf("\"Me? On whose word?\" %s stares. The bystanders mutter against you.", p.Name))
	}
	if s.Herring && s.Secret != "" && !s.SecretOut {
		s.SecretOut = true
		out = append(out, fmt.Sprintf("\"All right! I %s. That is where I was. Satisfied?\"", strings.TrimPrefix(s.Secret, "has been ")))
	}
	out = append(out, "", "{{R|A wrong accusation: your name suffers, and so does their opinion of you.}}")
	esc := NewRng(fmt.Sprintf("%s:mx:esc:%d:%d", g.Seed, c.ID, c.Wrong)).Chance(0.35)
	if esc || c.Wrong >= 3 {
		k := lvGet(g, c.Culprit)
		c.Status = "failed"
		c.Outcome = "You accused the wrong person and the true culprit slipped away."
		mxEscape(g, c, k)
		out = append(out, "{{R|While you were looking the wrong way, the real culprit slipped out of town. The case is lost.}}")
	}
	return out
}

func mxEscape(g *Game, c *mxCase, k *lvPerson) {
	if k == nil || !k.Alive || k.Burg <= 0 {
		return
	}
	ctx := lsMakeCtx(g)
	ctx.rebuild()
	r := ctx.rng(k.Key + ":mxesc")
	lsExile(ctx, k, r, "fleeing the "+mxKinds[c.Kind].Noun+" of "+c.VName)
	if a := mxActorOf(g, k.Key); a != nil {
		mxDropActor(g, a)
	}
	lvSt(g).addNews(lvDay(g), c.Burg, "%s has fled %s; folk say they knew something of the %s of %s.", k.Name, g.knBurgName(c.Burg), mxKinds[c.Kind].Noun, c.VName)
}

// mxResolve: the culprit is named. Applies the outcome to the simulation and pays the rewards.
func (g *Game) mxResolve(c *mxCase, s *mxSusp, p *lvPerson, a *Actor, how string, n int, pre, head string) []string {
	st := mxSt(g)
	day, yr := lvDay(g), lvYear(g)
	kd := mxKinds[c.Kind]
	c.Status = "solved"
	st.Solved++
	var out []string
	out = append(out, head, "")
	if pre != "" {
		out = append(out, strings.Split(strings.TrimRight(pre, "\n"), "\n")...)
	}
	switch how {
	case "confess":
		c.Outcome = "Confession"
		out = append(out, fmt.Sprintf("\"...Enough.\" %s sags. \"Yes. It was me. %s\"", p.Name, c.MotiveT))
		p.addLife(yr, "crime", "Confessed to the "+kd.Noun+" of "+c.VName+" when the wanderer laid out the proof.")
		mxFirst(g, "confess", "The wanderer drew a confession from the culprit of \""+c.Title+"\".")
		out = append(out, "", mxLaw(g, c, p, a))
	case "arrest":
		c.Outcome = "Arrest"
		out = append(out, fmt.Sprintf("%s does not move or speak as the watch is called. %s is led away.", p.Name, strings.ToUpper(mxHe(p)[:1])+mxHe(p)[1:]))
		p.addLife(yr, "crime", "Was arrested for the "+kd.Noun+" of "+c.VName+".")
		out = append(out, "", mxLaw(g, c, p, a))
	case "flee":
		c.Outcome = "Fled"
		out = append(out, fmt.Sprintf("%s goes white, then runs. The alarm is raised, but %s is out of the gate before the watch can close it.", p.Name, mxHe(p)))
		p.addLife(yr, "crime", "Fled "+g.knBurgName(c.Burg)+" when named for the "+kd.Noun+" of "+c.VName+".")
		mxEscape(g, c, p)
	default: // fight
		c.Outcome = "Duel"
		out = append(out, fmt.Sprintf("\"You will not take me!\" %s draws steel.", p.Name))
		a.Angry, a.AI = true, "hostile"
		p.addLife(yr, "crime", "Fought when named for the "+kd.Noun+" of "+c.VName+".")
	}
	// kin and friends of the victim are grateful
	if v := lvGet(g, c.Victim); v != nil {
		ths := append(append([]string{}, v.Kin...), v.Friends...)
		if v.Spouse != "" {
			ths = append(ths, v.Spouse)
		}
		for _, k := range ths {
			if q := lvGet(g, k); q != nil && q.Alive && q.Key != p.Key {
				q.remember(day, "help", "brought the truth of "+c.VName+" to light", 15)
				lsMood(q, 6)
			}
		}
		if c.Kind == "kidnap" && v.Alive {
			ls := lsSt(g)
			v.ArriveD = day + 1
			ls.Jail[v.Key] = v.ArriveD
			out = append(out, "", fmt.Sprintf("%s is found alive, and will be home within the day.", v.Name))
		}
	}
	state := 0
	if b := g.W.burgByID[c.Burg]; b != nil {
		state = b.State
	}
	found := len(mxFoundClues(c))
	coin := 40 + 12*found + 8*len(c.Suspect)
	rew := g.cuEarn(coin)
	c.Reward = rew
	g.gainXP(80 + 15*found)
	g.AddRep(uwFac(state), 8, " for solving a crime")
	g.fmFame(g.fmRealm(), 5)
	lvSt(g).addNews(day, c.Burg, "A wanderer, %s, has solved the %s of %s: %s was the one.", g.P.Name, kd.Noun, c.VName, p.Name)
	g.deed("mx", fmt.Sprintf("The wanderer %s solved \"%s\" and named %s.", g.P.Name, c.Title, p.Name), 0, 0)
	mxFirst(g, "solved", "The wanderer solved a first case.")
	if c.Kind == "poison" {
		mxFirst(g, "poison_solved", "The wanderer unmasked a poisoner.")
	}
	out = append(out, "", fmt.Sprintf("{{G|Case solved: %s.}} The magistrate pays you %s, and the town thinks well of you.", c.Title, rew))
	return out
}

// mxLaw puts the culprit through the realm's law, in the life simulation.
func mxLaw(g *Game, c *mxCase, p *lvPerson, a *Actor) string {
	if p.Burg <= 0 {
		return ""
	}
	ideo := ""
	if b := g.W.burgByID[c.Burg]; b != nil {
		if sd := g.W.stateByID[b.State]; sd != nil {
			ideo = sd.Ideology
		}
	}
	crime := mxKinds[c.Kind].Crime
	ctx := lsMakeCtx(g)
	ctx.rebuild()
	r := ctx.rng(p.Key + ":mxlaw")
	why := "the " + mxKinds[c.Kind].Noun + " of " + c.VName
	switch {
	case crime == "murder" && (ideo == "autocracy" || ideo == "monarchy" || ideo == "theocracy" || ideo == ""):
		lsDie(ctx, p, "on the gallows for "+why, 1, r, nil)
		mxDropActor(g, a)
		return "The law of this land is hard: " + p.Name + " is to hang."
	case crime == "murder" && (ideo == "republic" || ideo == "communal"):
		lsExile(ctx, p, r, why)
		mxDropActor(g, a)
		return p.Name + " is stripped of all rights and exiled from " + g.knBurgName(c.Burg) + "."
	}
	days := r.Int(10, 30)
	if crime == "murder" || crime == "kidnapping" {
		days = r.Int(40, 80)
	}
	ls := lsSt(g)
	p.Burg, p.Dest, p.ArriveD = 0, p.Home, lvDay(g)+days
	ls.Jail[p.Key] = p.ArriveD
	ls.From[p.Key] = c.Burg
	lsMood(p, -25)
	p.addLife(lvYear(g), "crime", fmt.Sprintf("Jailed in %s for %s (%d days).", g.knBurgName(c.Burg), why, days))
	lvSt(g).addNews(lvDay(g), c.Burg, "%s is jailed for %s.", p.Name, why)
	mxDropActor(g, a)
	return fmt.Sprintf("%s is taken to the cells for %d days.", p.Name, days)
}
