package engine

import (
	"fmt"
	"sort"
	"strings"
)

// Scene texts for es_delve.go. Every beat reads the site's real data (see evCtx) and fills a few lines.

func evDir(dx, dy int) string {
	ns, ew := "", ""
	if dy*2 < -abs(dx) {
		ns = "north"
	} else if dy*2 > abs(dx) {
		ns = "south"
	}
	if dx*2 < -abs(dy) {
		ew = "west"
	} else if dx*2 > abs(dy) {
		ew = "east"
	}
	switch {
	case ns != "" && ew != "":
		return ns + "-" + ew
	case ns != "":
		return ns
	case ew != "":
		return ew
	}
	return "right here"
}

func evSent(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.Index(s, ". "); i > 20 {
		s = s[:i]
	}
	if len(s) > 150 {
		s = s[:150]
		if i := strings.LastIndex(s, " "); i > 60 {
			s = s[:i]
		}
	}
	return strings.TrimRight(s, ".;:, ")
}

var evTrapNames = map[uint8]string{1: "a spiked floor", 2: "a dart trap", 3: "a fire jet", 4: "a false floor", 5: "a bell trap"}

// evTrapHint reads the level's hidden traps (read-only) as a warning from the spot (x, y).
func evTrapHint(zn *Zone, x, y int) string {
	type tr struct {
		d    int
		k    uint8
		x, y int
	}
	var ts []tr
	for i, k := range zn.Traps {
		tx, ty := i%MW, i/MW
		ts = append(ts, tr{abs(tx-x) + abs(ty-y), k, tx, ty})
	}
	if len(ts) == 0 {
		return ""
	}
	sort.Slice(ts, func(i, j int) bool {
		if ts[i].d != ts[j].d {
			return ts[i].d < ts[j].d
		}
		return ts[i].y*MW+ts[i].x < ts[j].y*MW+ts[j].x
	})
	t := ts[0]
	if t.d > 24 {
		return fmt.Sprintf("The floors of this level are trapped; I counted %d before I stopped counting.", len(ts))
	}
	return fmt.Sprintf("Mind %s about %d paces %s of here.", evTrapNames[t.k], maxi(abs(t.x-x), abs(t.y-y)), evDir(t.x-x, t.y-y))
}

// evLockHint reads the locked vault door and its key on the level (read-only).
func evLockHint(zn *Zone, x, y int) string {
	lx, ly, found := 0, 0, false
	for yy := 0; yy < MH && !found; yy++ {
		for xx := 0; xx < MW; xx++ {
			if zn.At(xx, yy) == TLocked {
				lx, ly, found = xx, yy, true
				break
			}
		}
	}
	if !found {
		return ""
	}
	s := fmt.Sprintf("A locked door waits %s of here.", evDir(lx-x, ly-y))
	if k := evSurvivorKey(zn); k != nil {
		s += fmt.Sprintf(" Its key lies %s.", evDir(k.X-x, k.Y-y))
	}
	return s
}

var evWeakWords = map[string]string{
	"slash": "steel cut deep", "pierce": "a thrust went in easily", "blunt": "a hammer cracked it", "fire": "fire made it scream",
	"cold": "cold slowed it", "shock": "lightning ran through it", "poison": "poison took hold", "acid": "acid ate into it",
	"holy": "blessed things hurt it", "dark": "darkness sapped it", "psychic": "it could be rattled in the mind",
}
var evResWords = map[string]string{
	"slash": "blades skidded off", "pierce": "arrows stuck and did nothing", "blunt": "blows were swallowed", "fire": "flame only fed it",
	"cold": "frost slid off", "shock": "sparks made it twitch and no more", "poison": "poison did nothing", "acid": "acid beaded and rolled off",
	"holy": "prayers meant nothing to it", "dark": "it drank the dark", "psychic": "there was nothing in it to frighten",
}

// evWeakness: the damage types a boss template is weak and resistant to (from foes.go, read-only).
func evWeakness(g *Game, tpl string) (weak, res []string) {
	if tplByID[tpl] == nil {
		return nil, nil
	}
	a := &Actor{Tpl: tpl, Name: "~", Flags: map[string]bool{"boss": true}}
	prof := g.bwBase(a)
	for _, t := range bwTypes {
		v := prof[t]
		if v < 0 {
			weak = append(weak, t)
		} else if v >= 40 {
			res = append(res, t)
		}
	}
	return
}

func evWeakLine(c *evCtx) string {
	weak, res := evWeakness(c.g, c.boss)
	var parts []string
	if len(weak) > 0 {
		parts = append(parts, evWeakWords[weak[0]])
	}
	if len(res) > 0 {
		parts = append(parts, evResWords[res[0]])
	}
	if len(parts) == 0 {
		return "It has no tender side that I found. Bring steel and patience."
	}
	return "What I saw of it: " + strings.Join(parts, "; ") + "."
}

func evBossTitle(c *evCtx) string {
	if t := tplByID[c.boss]; t != nil {
		return strings.ToLower(t.Name(c.g.Genre))
	}
	return "the thing"
}

func evGift(c *evCtx, r *Rng) string {
	if c.pk != nil && c.pk.Customs != nil {
		for _, rt := range c.pk.Customs.Rites {
			if rt.ID == "funeral" && len(rt.Gifts) > 0 {
				return Pick(r, rt.Gifts)
			}
		}
	}
	return ""
}

func evRiteName(c *evCtx) string {
	if c.pk != nil && c.pk.Customs != nil {
		for _, rt := range c.pk.Customs.Rites {
			if rt.ID == "funeral" && rt.Name != "" {
				return rt.Name
			}
		}
	}
	return ""
}

func evCurse(c *evCtx, r *Rng) string {
	if c.pk != nil && c.pk.Voice != nil && len(c.pk.Voice.Curses) > 0 {
		return Pick(r, c.pk.Voice.Curses)
	}
	return Pick(r, []string{"Who lifts this lid carries the dead home.", "Leave the sleeper what the sleeper was given.", "Cold to the one who comes for gold."})
}

// evReign: the ruler's real chronicle lines, oldest first.
func evReign(c *evCtx, max int) []string {
	if c.ruler == nil {
		return nil
	}
	var out []string
	for _, e := range c.w.Events {
		if len(out) >= max {
			break
		}
		if strings.Contains(e.Text, c.ruler.Name) && (e.Kind == "war" || e.Kind == "capture" || e.Kind == "peace" || e.Kind == "found" || e.Kind == "coup" || e.Kind == "death" || e.Kind == "disaster" || e.Kind == "quest") {
			out = append(out, fmt.Sprintf("Year %d: %s.", e.Y, evSent(e.Text)))
		}
	}
	return out
}

func evRulerLabel(c *evCtx) string {
	if c.ruler == nil {
		return "a sleeper whose name was never cut"
	}
	return fmt.Sprintf("%s, %s", c.ruler.Name, nonEmpty(c.ruler.Title, "ruler"))
}

func evOrderName(c *evCtx, r *Rng) string {
	if c.order != nil {
		return c.order.Name
	}
	return Pick(r, []string{"the Hollow Choir", "the Quiet Hand", "the Lamp-Eaters"})
}

// evBuild makes the scene of one beat, or nil.
func evBuild(c *evCtx, id string, r *Rng, zn *Zone, x, y, lv int) *evScene {
	s := c.s
	site := s.Name
	town := c.place
	sc := &evScene{}
	switch id {
	case "camp":
		sc.Anchor = "stair"
		sc.Decor = []evDecor{{-1, 0, TBed}, {1, 1, TBed}, {0, 1, TRubble}}
		what := "the dead of " + town
		if s.Ruin && c.fall != nil {
			what = fmt.Sprintf("what the chronicle keeps of %s: \"%s\" (year %d)", town, evSent(c.fall.Text), c.fall.Y)
		}
		switch {
		case s.Kind == "tower" && s.Theme == "magic":
			sc.Name = "apprentice's logbook"
			sc.Look = "A reading desk at the foot of the stair, a cold lamp, a coat folded as if its owner stepped out."
			sc.Lines = []string{
				fmt.Sprintf("Logbook of %s, apprentice, year %d. I climb to see the work of the one who built %s.", c.leaderName, c.expYear, site),
				"Floor by floor the wards get older. The lower ones only bark; the upper ones keep count.",
				"Rule from the margin of the first page: touch nothing that hums.",
			}
		case s.Theme == "abyss" || s.Theme == "chasm":
			sc.Name = "descent ledger"
			sc.Look = "Pitons, a rope coiled on a stone and a ledger weighed down with a boot."
			sc.Lines = []string{
				fmt.Sprintf("Descent ledger of %s, year %d, out of %s. %d fathoms of line, %d volunteers.", c.leaderName, c.expYear, town, 40+r.Int(0, 80), 3+r.Int(0, 4)),
				"Rule one: the rope is the road. Rule two: when the lamp gutters, stand still.",
				"Everyone who goes down says the same: it gets quieter the deeper, and the quiet is listening.",
			}
		case s.Theme == "vault" || s.Theme == "arcane" || s.Theme == "sunken":
			sc.Name = "survey crew's field book"
			sc.Look = "A survey crew's table, still set with instruments, dust over everything but the ledger."
			sc.Lines = []string{
				fmt.Sprintf("Field book of %s, year %d. Sent from %s to map %s and bring up what is safe to carry.", c.leaderName, c.expYear, town, site),
				"We chalked arrows on the safe floors. Where the chalk stops, we stopped.",
				"Standing order: no one past the second room alone, and no one eats anything found.",
			}
		default:
			sc.Name = "expedition ledger"
			sc.Look = "A dead camp beside the way down: two bedrolls, a fire pit of burnt lamp crates, a satchel under a stone."
			sc.Lines = []string{
				fmt.Sprintf("Ledger of %s's company, year %d. Hired at %s: %d hands, %d fathoms of rope, oil for %d days.", c.leaderName, c.expYear, town, 4+r.Int(0, 5), 30+r.Int(0, 40), 5+r.Int(0, 6)),
				"We came for " + what + ".",
				fmt.Sprintf("Day 4: the rooms below are not empty. We hear it below, at night, regular as a mallet."),
			}
		}
		if h := evTrapHint(zn, x, y); strings.HasPrefix(h, "Mind") {
			sc.Lines = append(sc.Lines, "Foreman's count of traps so far. "+h)
		}
		sc.Sum = fmt.Sprintf("The camp of %s's company: they came for %s and heard something below.", c.leaderName, evSent(what))
	case "lid":
		sc.Anchor = "sarc"
		ru := evRulerLabel(c)
		sc.Name = "rubbing of a coffin lid"
		sc.Look = fmt.Sprintf("The lid is carved with a crowned figure and columns of cut letters: %s.", ru)
		sc.Lines = append(sc.Lines, fmt.Sprintf("%s.", ru))
		if c.ruler != nil {
			sc.Lines = append(sc.Lines, fmt.Sprintf("Born %d, died %d%s.", c.ruler.Born, c.ruler.Died, causeText(c.ruler.Cause)))
		}
		sc.Lines = append(sc.Lines, evReign(c, 2)...)
		g := evGift(c, r)
		if g != "" {
			rn := evRiteName(c)
			if rn == "" {
				rn = "burial rite"
			}
			sc.Lines = append(sc.Lines, fmt.Sprintf("Laid in as the %s requires: %s.", rn, g))
			sc.Goods = g
		} else {
			sc.Goods = Pick(r, []string{"funeral coin", "grave bead", "bone comb"})
		}
		sc.Lines = append(sc.Lines, "Warding line: \""+evCurse(c, r)+"\"")
		if len(sc.Lines) > 5 {
			sc.Lines = sc.Lines[:5]
		}
		sc.Note = fmt.Sprintf("Cut into the wall: %s. The letters are deep and meant to be read by torchlight.", ru)
		sc.Sum = "The lid of " + ru + " tells the reign and carries a warding line."
	case "thief":
		sc.Decor = []evDecor{{0, 1, TBones}, {1, 1, TRubble}, {-1, 0, TBones}}
		sc.Loot = lv/2 + 2
		sc.Name = "dead thief's diary"
		sc.Look = "A dead man sits against the wall with a rope still looped round his belt, one hand on a sack."
		sc.Lines = []string{
			fmt.Sprintf("%s of %s. Third night. I came in by rope; the rope is where I left it, for the climb out.", c.thief, town),
			"Prised the lid of the old ruler and took what the dead were given. It was not as heavy as the story.",
		}
		if h := evTrapHint(zn, x, y); h != "" {
			sc.Lines = append(sc.Lines, "I paid for this. "+h)
		}
		sc.Lines = append(sc.Lines, "Something answered my mallet. The sack stays; take it if you can carry it past what I heard.")
		sc.Sum = fmt.Sprintf("%s the thief broke in before you and did not leave.", c.thief)
	case "wrong":
		sc.Anchor = "sarc"
		sc.Decor = []evDecor{{1, 0, TBones}}
		sc.Name = "slab of scratched stone"
		sc.Look = "On this coffin the name has been chiselled out, the groove still sharp, and a second, smaller coffin stands against it."
		ru := "the one the court would not name"
		born, died := 0, 0
		if c.ruler != nil {
			ru, born, died = c.ruler.Name, c.ruler.Born, c.ruler.Died
		}
		sc.Lines = []string{
			fmt.Sprintf("The lid should read %s. The chisel took the name and left the dates: %d to %d.", ru, born, died),
			"Under the gap, scratched later with a knife: \"not him\".",
			"The small coffin has no lid at all. Whatever was in it is lighter than a sleeper should be, and the dust round it has been walked through.",
		}
		sc.Note = "A name chiselled out of the stone. Somebody wanted this grave nameless, and wanted it known."
		sc.Sum = "A coffin has its name chiselled out, and a second coffin stands beside it, empty."
	case "ritual":
		on := evOrderName(c, r)
		sc.Decor = []evDecor{{-2, 0, TBones}, {-2, -1, TRubble}, {-1, -2, TBones}, {0, -2, TRubble}, {1, -2, TBones}, {2, -1, TRubble}}
		sc.Name = "rule of " + on
		sc.Look = "A sigil chalked on the floor in a ring, drawn two thirds round and left. The chalk is still in a dead hand."
		leader, rank := "the Keeper", "Seeker"
		sign := ""
		if c.order != nil {
			leader = nonEmpty(c.order.LeaderTit+" "+c.order.Leader, leader)
			rank = c.order.Ranks[0]
			sign = c.order.Sign
		}
		sc.Lines = []string{
			fmt.Sprintf("%s hold the lower rooms. The %s carries the lamp and says nothing to anyone.", strings.ToUpper(on[:1])+on[1:], rank),
			fmt.Sprintf("The ring is drawn by the third rank and closed only at the word of %s.", leader),
		}
		if sign != "" {
			sc.Lines = append(sc.Lines, "Sign of the order: "+sign)
		}
		sc.Lines = append(sc.Lines, "The ring was not closed. Whoever the chalk belonged to ran, and what the ring was meant to hold is not held.")
		sc.Sum = fmt.Sprintf("%s used these rooms for a rite that was never finished.", on)
	case "trail":
		sc.Decor = []evDecor{{1, 0, TRubble}, {-1, 1, TRubble}}
		sc.Name = "chalked field map"
		sc.Look = "Chalk arrows run along the floor and stop at a heap of fallen stone with a boot print on top."
		sc.Lines = []string{"A scrap of field map held down with a stone: the safe floor is marked, the rest is crossed."}
		if h := evTrapHint(zn, x, y); h != "" {
			sc.Lines = append(sc.Lines, h)
		}
		if h := evLockHint(zn, x, y); h != "" {
			sc.Lines = append(sc.Lines, h)
		}
		sc.Lines = append(sc.Lines, fmt.Sprintf("Margin note: the master of %s sleeps in the widest hall. Do not wake it for the sake of a map.", site))
		sc.Sum = "A chalked map marks the safe floor and the widest hall where the master sleeps."
	case "survivor":
		sc.Anchor = "key"
		sc.Decor = []evDecor{{0, 1, TBones}, {1, 0, TBones}}
		sc.Name = "survivor's last page"
		sc.Look = "A body lies curled round its right hand. The fingers are closed on something, and a page is pinned under the knee."
		dn := evName(c, r)
		sc.Lines = []string{fmt.Sprintf("%s, of %s's company. The others went on ahead and I was the one who stayed with the lamp.", dn, town)}
		if h := evLockHint(zn, x, y); h != "" {
			sc.Lines = append(sc.Lines, h)
		} else {
			sc.Lines = append(sc.Lines, "There was a key on me when I fell. I threw it down the stair so it would not be found with me.")
		}
		if h := evTrapHint(zn, x, y); h != "" {
			sc.Lines = append(sc.Lines, h)
		}
		sc.Lines = append(sc.Lines, "Below is the last hall. It is waiting, and it does not tire.")
		sc.Sum = dn + " died holding what the level is locked with and warned of what waits below."
	case "lair":
		sc.Anchor = "boss"
		sc.Book = true
		sc.Name = "record of the master of " + site
		sc.Look = "The last hall. Offerings sit in rows, some of them old enough to have turned to dust in their dishes."
		sc.Lines = append(sc.Lines, evOrigin(c, r))
		sc.Lines = append(sc.Lines, evWeakLine(c))
		sc.Lines = append(sc.Lines, "Whoever reads this and still goes in: it is a "+evBossTitle(c)+" and it does not yield.")
		sc.Sum = "The record of the master of the deepest hall: where it came from and what it cannot bear."
	case "strike":
		sc.Anchor = "stair"
		w := 2 + lv + zn.Level/3
		sc.Name = "strike notice"
		sc.Look = "A notice nailed to the first prop, torn at the corners, with a muddy handprint over the signature."
		sc.Lines = []string{
			fmt.Sprintf("NOTICE TO THE HANDS OF %s: from year %d the shift pays %d %s, down from %d.", strings.ToUpper(site), c.w.Year-r.Int(1, 5), w, c.cur, w+2),
			"Those who lay down tools will not be paid for the days they lay them down.",
			"Written below in pencil: we are not going back down while the knocking goes on.",
		}
		sc.Sum = fmt.Sprintf("A strike notice at %s: wages cut to %d %s a shift, and the men refuse to go down.", site, w, c.cur)
	case "ledger":
		w := 2 + zn.Level/3
		sc.Decor = []evDecor{{0, 1, TRubble}}
		sc.Name = "company ledger"
		sc.Look = "A clerk's table, one leg propped on ore sacks, a ledger open to the last full week."
		sc.Lines = []string{
			fmt.Sprintf("Wages by the shift at %s: hewers %d %s, haulers %d, lampmen %d. Owed to the hands this week: %d %s.", site, w+2, c.cur, w, w-1, (w+2)*(30+r.Int(0, 20)), c.cur),
			fmt.Sprintf("Foreman %s, in the margin: the third level has put out three lamps this week and men will not go back to it.", evName(c, r)),
			"Reply in another hand: the company is not paying for lamps that go out on their own. Send them down.",
		}
		sc.Sum = "The company ledger shows the wages and a foreman's complaint about the third level."
	case "collapse":
		sc.Decor = []evDecor{{-1, 0, TRubble}, {1, 0, TRubble}, {0, -1, TRubble}, {0, 1, TRubble}, {1, 1, TBones}}
		days := 20 + r.Int(0, 30)
		sc.Name = "tally stick from a prop"
		sc.Look = "A timber prop split under a fall of stone. Tally marks run up the splinters, then one long scratch."
		sc.Lines = []string{
			fmt.Sprintf("%d marks, one for each time the water came and went.", days),
			"Scratched beside the last: we hear it below.",
			"Bread gone on the fifteenth. Dig toward the lamp. Do not dig toward the sound.",
		}
		sc.Note = fmt.Sprintf("Tally marks, %d of them, and the words \"we hear it below\".", days)
		sc.Sum = fmt.Sprintf("Trapped miners counted %d days under the fall and heard something below.", days)
	case "vein":
		sc.Anchor = "ore"
		sc.Name = "marked assay stick"
		sc.Look = "A chalk cross on the rock beside a rich seam, and a split stick wedged in the crack."
		sc.Lines = []string{
			"The miners' mark: a cross for good ore, a circle for water, a bar for rot in the roof.",
			fmt.Sprintf("This seam assayed rich in year %d. The company never worked it, because the men would not go farther than the cross.", c.w.Year-r.Int(1, 6)),
			"Take what you can swing at. The roof beyond it is bad.",
		}
		sc.Loot = lv/2 + 1
		sc.Sum = "A miners' mark on a rich seam the company never worked."
	case "poster":
		sc.Anchor = "stair"
		t := c.target
		sc.Name = "wanted poster"
		sc.Look = "A poster pinned to the wall with a knife, someone's thumb pressed over the face."
		if t != nil {
			sc.Lines = []string{
				fmt.Sprintf("WANTED: %s, once %s, for %s.", t.Name, nonEmpty(t.Title, t.Role), bhCrime(t)),
				fmt.Sprintf("Reward %d %s. Dead or alive; alive pays half again.", 180+t.Renown*3+t.Martial*10, c.cur),
				fmt.Sprintf("Underneath, in the hand of whoever lives here: %s is welcome, and the poster stays as a joke.", t.Name),
			}
			sc.Sum = fmt.Sprintf("A poster of %s, wanted for %s, hangs in the hideout.", t.Name, bhCrime(t))
		} else {
			sc.Lines = []string{"WANTED: the leader of this place, for banditry.", fmt.Sprintf("Reward %d %s.", 200+lv*50, c.cur), "Someone has drawn a crown on the face."}
			sc.Sum = "A poster of the hideout's own leader, with a crown drawn on it."
		}
	case "ledger_h":
		return nil
	case "fence":
		sc.Name = "fence's price list"
		sc.Look = "A shelf of bundles with chalked tags, and a price list nailed beside it."
		b := 8 + zn.Level*2
		sc.Lines = []string{
			fmt.Sprintf("Blades %d %s, a jewelled piece %d, any relic %d, no questions on a Thursday.", b, c.cur, b*3, b*5),
			"Cut for the house: a third of whatever you bring and silence about where it came from.",
		}
		if c.order != nil {
			sc.Lines = append(sc.Lines, fmt.Sprintf("Members of %s pay half. Ranks of the order: %s to %s.", c.order.Name, c.order.Ranks[0], c.order.Ranks[4]))
		}
		sc.Sum = "A fence's price list, with the house cut."
	case "map":
		sc.Name = "map of the cellars"
		sc.Look = "A map drawn on a flour sack, creased from being carried, with a thumbprint at one corner."
		var kinds []string
		if c.bd != nil && c.bd.Alive {
			cp := c.g.cityPlanFor(c.bd)
			seen := map[string]bool{}
			for _, k := range cp.districts {
				if !seen[k] {
					seen[k] = true
					kinds = append(kinds, districtTitle(k))
				}
			}
			sort.Strings(kinds)
		}
		a, b := "the lower cellar", "the old drain"
		if len(kinds) > 1 {
			a, b = "the cellar of the "+kinds[r.Int(0, len(kinds)-1)], "the drain under the "+kinds[r.Int(0, len(kinds)-1)]
		}
		sc.Lines = []string{
			fmt.Sprintf("The ways under %s: from this room, %s, then %s, then up a ladder to the street.", town, a, b),
			"Two crosses mark where the watch has found us. Don't use those.",
			"A third mark has been scraped out: the one way to the deepest room nobody uses twice.",
		}
		sc.Sum = "A map of the cellars and drains under " + town + "."
	case "graffiti":
		sc.Anchor = "stair"
		a, b := evName(c, r), evName(c, r)
		sc.Name = "ranker graffiti"
		sc.Look = "The wall by the stair is covered in scratched names and tallies, several struck through."
		sc.Lines = []string{
			fmt.Sprintf("%s: floor %d. %s: floor %d, and then stopped writing.", a, 5+r.Int(0, 10), b, 2+r.Int(0, 7)),
			"Someone's drawn a ladder with a gap and written: guardians every fifth floor.",
			"Below it, in a steady hand: the keeper hits hardest right after you think it is done.",
		}
		sc.Note = "Names scratched and struck through. A tally runs to the ceiling."
		sc.Sum = "The rankers' wall: names, floors and a warning about the guardians."
	case "log":
		sc.Book = true
		sc.Name = "register of climbers"
		sc.Look = "A bound register on a lectern, the pages swollen, the last entry unfinished."
		lg := c.g.hf().Legends
		var ls []string
		for i := range lg {
			if len(ls) >= 3 {
				break
			}
			ls = append(ls, fmt.Sprintf("Year %d: %s, %s %s, level %d. %s.", lg[i].Year, lg[i].Name, strings.ToLower(lg[i].Species), strings.ToLower(lg[i].Calling), lg[i].Level, nonEmpty(Plain(lg[i].Cause), "retired")))
		}
		if len(ls) == 0 {
			ls = append(ls, evPeopleLog(c, 3)...)
		}
		sc.Lines = ls
		sc.Lines = append(sc.Lines, "The last line is a name begun and never ended.")
		sc.Sum = "A register of earlier climbers of " + site + "."
	case "legend":
		lg := c.g.hf().Legends
		sc.Name = "climber's last note"
		sc.Look = "A scrap of cloth is tied to a rail, written on in soot, flapping in the draught."
		if len(lg) > 0 {
			l := lg[r.Int(0, len(lg)-1)]
			sc.Name = l.Name + "'s note"
			deed := "I did what I could."
			if len(l.Deeds) > 0 {
				deed = evSent(Plain(l.Deeds[0])) + "."
			}
			sc.Lines = []string{
				fmt.Sprintf("%s, %s %s, year %d, level %d. %s", l.Name, strings.ToLower(l.Species), strings.ToLower(l.Calling), l.Year, l.Level, deed),
				fmt.Sprintf("It ended with %s.", lowerFirst(nonEmpty(Plain(l.Cause), "retiring"))),
				"Whoever reads this: I was wrong about the rest. The way up goes slower than the way down.",
			}
			sc.Sum = fmt.Sprintf("%s, an earlier wanderer of year %d, left a note here.", l.Name, l.Year)
		} else {
			ls := evPeopleLog(c, 1)
			sc.Lines = append(ls, "Whoever reads this: I was wrong about the rest. The way up goes slower than the way down.")
			sc.Sum = "An earlier climber left a note here."
		}
	default:
		return nil
	}
	return sc
}

func evPeopleLog(c *evCtx, n int) []string {
	var ps []*PersonData
	for i := range c.w.People {
		if p := &c.w.People[i]; p.Alive && p.Martial >= 8 {
			ps = append(ps, p)
		}
	}
	sort.Slice(ps, func(i, j int) bool { return ps[i].Martial*1000+ps[i].Renown > ps[j].Martial*1000+ps[j].Renown })
	var out []string
	for i := 0; i < len(ps) && i < n; i++ {
		out = append(out, fmt.Sprintf("Year %d: %s, %s, martial %d, passed this way and kept going.", c.w.Year-i-1, ps[i].Name, nonEmpty(ps[i].Title, ps[i].Role), ps[i].Martial))
	}
	if len(out) == 0 {
		out = []string{"Year unknown: a name that was never finished."}
	}
	return out
}

// evOrigin: where the master of the deepest hall came from, by theme, with a real fact when there is one.
func evOrigin(c *evCtx, r *Rng) string {
	s := c.s
	fact := ""
	if c.fall != nil && s.Ruin {
		fact = fmt.Sprintf(" The chronicle of %s says: \"%s\" (year %d).", c.place, evSent(c.fall.Text), c.fall.Y)
	}
	switch s.Theme {
	case "catacomb", "barrow":
		who := "the last of the court"
		if c.ruler != nil {
			who = c.ruler.Name
		}
		return fmt.Sprintf("When %s was laid down, a keeper was sworn to stay until relieved. Nobody came to relieve it.%s", who, fact)
	case "mine":
		return "The miners cut into something that was already there. The foreman's last entry: it was not a seam."
	case "hideout":
		if c.vl != nil {
			return fmt.Sprintf("%s took the place after the watch of %s stopped coming. The cutthroats are sworn; they will not run.", c.vl.P.Name, c.bd.Name)
		}
		if s.Outlaw != nil {
			return fmt.Sprintf("%s was built on stolen coin and %s has kept it since. They hold %d %s of it, and will not share.", s.Outlaw.Name, "its chief", s.Gold, c.cur)
		}
		return "The warren's chief took it from the previous chief, and has not slept since."
	case "sunken":
		return "The vault was sunk on purpose when the town above would not pay its keeper. What was left in the water grew."
	case "arcane":
		return "The maze was built to hold one thing, and the singing is the thing keeping time."
	case "chasm":
		return "The rift opened and something came up with the cold. It is still coming up."
	case "vault":
		return "The warden was told to hold the door until relief came. Its clock stopped counting years a long time ago."
	case "abyss":
		return "The ones who went down and did not come back did not die. They stayed, and became what is below."
	case "trials":
		return "The keeper tests every climber and has never been bested twice by the same hand."
	case "magic":
		return "The familiar was left to guard the work when the mage did not come down. It does not know the mage is dead."
	}
	return "Nobody wrote down where it came from."
}
