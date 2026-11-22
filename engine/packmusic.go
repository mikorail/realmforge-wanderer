package engine

import (
	"fmt"
	"strings"
)

// Culture pack music (prefix pm): Material.Music (instruments, forms, scale) in what a town's music shops sell, what
// its bards carry and play, and what a wanderer can learn from them. Everything is optional: no pack, no Music, no
// change. Nothing random (hashes of seed and names). State g.ext("pm") (planet-local): forms learned.
//
//   - pmSet(base)          the four instruments a music seller offers (hook in games.go gmShop)
//   - pmBuskTip(it)        busking tip bonus for a local instrument and learned forms (hook in games.go gmBusk)
//   - zone hook            bards and minstrels of a pack culture carry that culture's instruments
//   - talk option          "Ask about their music." / "Ask them to teach you a tune." for bards, minstrels, innkeepers
//
// The shell side (src/culturemusic.ts profileFor, wired in src/music.ts and src/garden.ts) reads frame.mstyle from
// musicstyle.go, which already carries the pack's instruments, forms and scale word.

type pmState struct {
	Learned map[string]bool `json:"l,omitempty"` // "culture:form" -> tune learned
	Day     map[string]int  `json:"d,omitempty"` // "actor:lesson" -> day+1 of the last lesson
}

func (g *Game) pm() *pmState {
	st := g.ext("pm", func() any { return &pmState{} }).(*pmState)
	if st.Learned == nil {
		st.Learned = map[string]bool{}
	}
	if st.Day == nil {
		st.Day = map[string]int{}
	}
	return st
}

func init() {
	unLocalKeys = append(unLocalKeys, "pm")
	zoneHooks = append(zoneHooks, pmZone)
	talkHooks = append(talkHooks, pmTalk)
}

// pmMusic returns the pack's Music of a culture id, or nil.
func (g *Game) pmMusic(culture int) *MusicStyle {
	if p := cgPack(g.W, culture); p != nil && p.Material != nil && p.Material.Music != nil && len(p.Material.Music.Instruments) > 0 {
		return p.Material.Music
	}
	return nil
}

var pmClassWords = [4][]string{
	0: {},
	1: {"flute", "pipe", "whistle", "shawm", "reed", "ney", "oboe", "ocarina", "clarinet", "duduk", "shaku", "recorder", "fife", "zurna", "bansuri"},
	2: {"drum", "tabla", "taiko", "djembe", "bodhran", "tambour", "bongo", "daf", "dhol", "timpani", "gong", "cymbal", "rattle", "clapper", "bell"},
	3: {"fiddle", "violin", "vielle", "rebec", "erhu", "horn", "trumpet", "conch", "lur", "khuur", "viol", "sarangi", "cello", "trombone", "organ", "bowed", "kamancheh", "rebab"},
}

func pmClass(name string) int {
	n := strings.ToLower(name)
	for _, c := range []int{2, 1, 3} {
		for _, w := range pmClassWords[c] {
			if strings.Contains(n, w) {
				return c
			}
		}
	}
	return 0 // plucked and struck strings, anything else
}

func pmTrim(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	for _, art := range []string{"a ", "an ", "the "} {
		s = strings.TrimPrefix(s, art)
	}
	return strings.TrimRight(s, ". ")
}

// pmSet: the string / wind / drum / bowed-or-horn set of a place's music seller, with the pack's instruments laid
// over base; slots the pack does not fill keep their old names.
func (g *Game) pmSet(base [4]string) [4]string {
	if g == nil || g.W == nil {
		return base
	}
	_, p := g.cgHere()
	if p == nil || p.Material == nil || p.Material.Music == nil {
		return base
	}
	var filled [4]bool
	var spare []string
	for _, in := range p.Material.Music.Instruments {
		in = pmTrim(in)
		if in == "" {
			continue
		}
		c := pmClass(in)
		if !filled[c] {
			filled[c], base[c] = true, in
		} else {
			spare = append(spare, in)
		}
	}
	for _, in := range spare { // extra instruments take the slots the pack left empty
		for c := 0; c < 4; c++ {
			if !filled[c] {
				filled[c], base[c] = true, in
				break
			}
		}
	}
	return base
}

// pmInstFor: the instrument a named performer carries in a culture, "" without a pack.
func (g *Game) pmInstFor(culture int, name string) string {
	m := g.pmMusic(culture)
	if m == nil {
		return ""
	}
	return pmTrim(m.Instruments[int(HashStr(g.W.Seed+":pm:"+name)%uint64(len(m.Instruments)))])
}

func pmZone(g *Game, zn *Zone, _ *Rng) {
	if g == nil || g.W == nil || zn == nil || zn.Kind != "town" || zn.Z != 0 || zn.Burg <= 0 {
		return
	}
	bd := g.W.burgByID[zn.Burg]
	if bd == nil || g.pmMusic(bd.Culture) == nil {
		return
	}
	for _, a := range zn.Actors {
		if a == nil || (a.Role != "bard" && a.Role != "minstrel") {
			continue
		}
		in := g.pmInstFor(bd.Culture, a.Name)
		if in == "" {
			continue
		}
		if strings.Contains(a.Desc, " a lute ") {
			a.Desc = strings.Replace(a.Desc, " a lute ", " "+pmArticle(in)+" ", 1)
		} else if a.Desc != "" {
			a.Desc += " They carry " + pmArticle(in) + "."
		}
	}
}

func pmArticle(s string) string {
	if s == "" {
		return s
	}
	switch s[0] {
	case 'a', 'e', 'i', 'o', 'u':
		return "an " + s
	}
	return "a " + s
}

var pmRoles = map[string]bool{"bard": true, "minstrel": true, "innkeeper": true, "tavern": true}

func pmTalk(g *Game, a *Actor) []Option {
	if !cgOK(g, a) || !pmRoles[a.Role] || g.Cur == nil || g.Cur.Kind != "town" {
		return nil
	}
	cu, _ := g.cgHere()
	if cu == nil {
		return nil
	}
	m := g.pmMusic(cu.ID)
	if m == nil {
		return nil
	}
	out := []Option{{"Ask about their music.", func() { g.pmAbout(a, cu, m) }}}
	if len(m.Forms) > 0 && (a.Role == "bard" || a.Role == "minstrel") {
		out = append(out, Option{"Ask them to teach you a tune.", func() { g.pmTeach(a, cu, m) }})
	}
	return out
}

func (g *Game) pmAbout(a *Actor, cu *CultureData, m *MusicStyle) {
	in := g.pmInstFor(cu.ID, a.Name)
	var ls []string
	if a.Role == "bard" || a.Role == "minstrel" {
		ls = append(ls, fmt.Sprintf("%s lifts %s. \"This is what we play in %s.\"", g.the(a), pmArticle(in), g.Cur.Name))
	} else {
		ls = append(ls, "\"Music?\" "+g.the(a)+" nods toward the corner. \"There is always someone playing.\"")
	}
	var insts []string
	for _, i := range m.Instruments {
		insts = append(insts, pmTrim(i))
	}
	ls = append(ls, "{{Y|Instruments:}} "+strings.Join(insts, ", ")+".")
	if len(m.Forms) > 0 {
		ls = append(ls, "{{Y|Songs:}} "+strings.ToLower(strings.Join(m.Forms, ", "))+".")
	}
	if m.Scale != "" {
		ls = append(ls, "{{Y|The ear of it:}} "+pdClean(m.Scale)+".")
	}
	g.talkMenu(strings.Join(ls, "\n"))
}

func (g *Game) pmTeach(a *Actor, cu *CultureData, m *MusicStyle) {
	st := g.pm()
	day := g.Day()
	dk := fmt.Sprintf("%d:teach", a.ID)
	if st.Day[dk] == day+1 {
		g.talkMenu("\"Not twice in a day. My fingers are tired. Come back tomorrow.\"")
		return
	}
	idx := -1
	for i, f := range m.Forms { // the first form you do not know yet, from a hashed start
		j := (int(HashStr(g.W.Seed+":pmt:"+a.Name)%uint64(len(m.Forms))) + i) % len(m.Forms)
		if !st.Learned[fmt.Sprintf("%d:%s", cu.ID, m.Forms[j])] {
			idx = j
			_ = f
			break
		}
	}
	if idx < 0 {
		g.talkMenu("\"You know every tune I know. Play them well.\"")
		return
	}
	st.Day[dk] = day + 1
	form := m.Forms[idx]
	st.Learned[fmt.Sprintf("%d:%s", cu.ID, form)] = true
	g.gainXP(6)
	g.AddRep(a.Faction, 1, " for learning a local tune")
	g.Turn += 20
	txt := fmt.Sprintf("%s plays it through twice, slowly: a %s of %s. You work at it until your hands know it.", g.the(a), strings.ToLower(form), cu.Name)
	if g.gmInstrument() != nil {
		txt += "\n{{G|You can play it at a crowd now.}}"
	} else {
		txt += "\n{{W|You would need an instrument to play it. Music sellers here stock the local kinds.}}"
	}
	g.talkMenu(txt)
}

// pmBuskTip: a busking bonus for playing the local kind of instrument and for each local tune learned (up to 4).
func (g *Game) pmBuskTip(it *Item) int {
	cu, _ := g.cgHere()
	if cu == nil {
		return 0
	}
	m := g.pmMusic(cu.ID)
	if m == nil {
		return 0
	}
	n := 0
	if it != nil {
		low := strings.ToLower(it.Name)
		for _, in := range m.Instruments {
			if w := pmTrim(in); w != "" && strings.Contains(low, w) {
				n += 2
				break
			}
		}
	}
	st := g.pm()
	l := 0
	for _, f := range m.Forms {
		if st.Learned[fmt.Sprintf("%d:%s", cu.ID, f)] {
			l++
		}
	}
	return n + mini(l, 4)
}
