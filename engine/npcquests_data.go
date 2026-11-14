package engine

import (
	"fmt"
	"strings"
)

// Text and tables for npcquests.go (prefix nq): genre-skinned item names, letter bodies by goal, thanks lines.

// nqSkin names the things people hand you, per genre.
type nqSkin struct {
	Letter string
	Parcel string
	Gift   string
	Herbs  string
	Purse  string
	Heir   [3]string
	Book   string
}

var nqSkins = map[string]nqSkin{
	"fantasy":    {"sealed letter", "wrapped parcel", "posy of flowers", "healing herbs", "purse", [3]string{"silver locket", "grandmother's ring", "carved family seal"}, "old book of the household"},
	"mythic":     {"wax-sealed scroll", "bound bundle", "garland of sacred leaves", "sacred herbs", "offering-pouch", [3]string{"hearth-token", "ancestor's torque", "votive figure"}, "tablet of the old songs"},
	"steampunk":  {"pneumatic message-tube", "brass-bound box", "clockwork music box", "tincture of herbs", "coin-purse", [3]string{"pocket chronometer", "engraved signet", "brass locket"}, "ledger-bound journal"},
	"dieselpunk": {"telegram envelope", "crated package", "bottle of cologne", "medicine tin", "wallet", [3]string{"service lighter", "pocket watch", "wedding band"}, "dog-eared notebook"},
	"atompunk":   {"pink-slip envelope", "lead-lined packet", "boxed corsage", "wonder-tonic", "billfold", [3]string{"transistor radio", "charm bracelet", "atomic-age locket"}, "mimeographed manual"},
	"cyberpunk":  {"encrypted data-chip", "courier case", "synth-rose", "med-patch", "cred-stick", [3]string{"old photo-slate", "cracked implant core", "memory shard"}, "private data-slate"},
	"biopunk":    {"spore-sealed note", "vat jar", "glow-orchid", "healing culture", "seed-pouch", [3]string{"graft-bone charm", "living brooch", "seed-vial"}, "bound growth-journal"},
	"solarpunk":  {"woven message-leaf", "sun-dried bundle", "wreath of flowers", "garden herbs", "woven purse", [3]string{"carved seed-token", "glass prism pendant", "family loom-shuttle"}, "commons almanac"},
	"wasteland":  {"scrap-paper note", "oilcloth bundle", "bottle-cap bracelet", "clean-water herbs", "pouch", [3]string{"pre-war photograph", "dented locket", "rusted wedding ring"}, "water-stained diary"},
	"twilight":   {"ash-sealed letter", "candle-wax bundle", "dried night-bloom", "pale tonic of herbs", "pouch", [3]string{"smoke-dark locket", "last-light lantern charm", "tarnished ring"}, "ledger of the old days"},
}

func nqSkinFor(genre string) nqSkin {
	if s, ok := nqSkins[genre]; ok {
		return s
	}
	return nqSkins["fantasy"]
}

// nqLetterBody: the contents of a carried letter or parcel by the sender's goal. %s = sender.
var nqLetterBody = map[string]string{
	"love":      "I think of you every evening and cannot say it to your face. If the answer is no, burn this and I shall understand. Yours, %s.",
	"wealth":    "The goods are as agreed. Pay the bearer and I will count the matter closed. I trust the weights will be honest this time. %s.",
	"faith":     "A gift for the shrine, and a request for your prayers. The year has been hard and I would be lighter for it. %s.",
	"knowledge": "I have found the passage we argued over. Read it twice before you tell me I am wrong. %s.",
	"fame":      "Tell them what I did at the crossing, exactly as it happened, and no more than a little better. %s.",
	"justice":   "I set down here what I saw and who was there. If you have courage left, take it to someone with the power to act. %s.",
	"revenge":   "You know who is to blame. You know what I mean to do. Do not try to stop me, only be sure of your own side. %s.",
	"family":    "We are well, and the little ones ask after you. Come at the feast if the road allows. %s.",
	"safety":    "Things here grow worse. If the offer of a room still stands, I should like to take it, and soon. %s.",
}

// nqGoalLetter picks letter or parcel by goal: true = parcel.
var nqGoalParcel = map[string]bool{"wealth": true, "faith": true, "knowledge": true}

var nqThanks = []string{
	"\"I will not forget this.\"",
	"\"You have done more than I could ask of a stranger.\"",
	"\"Few would have bothered. Thank you.\"",
	"\"It is a weight off me. Truly.\"",
}

var nqLapse = []string{
	"\"I waited. You never came. I see how it is.\"",
	"\"Too late now. Do not trouble yourself on my account.\"",
}

var nqExtraWords = map[string]string{
	"item":   "a keepsake",
	"secret": "a confidence",
	"favour": "a favour owed",
}

// nqCrimeWord skins the debt/theft talk lightly.
var nqTradeStake = []string{"baker", "potter", "weaver", "chandler", "cobbler", "tanner", "cooper"}

// ---------------------------------------------------------------- template builders

func nqReport(p *lvPerson, burg int, opt, say string) nqStep {
	return nqStep{K: "talk", Who: p.Key, Burg: burg, Opt: opt, Say: say}
}

func nqCands(p *lvPerson, spouse, kin, friends bool) []string {
	var out []string
	if spouse && p.Spouse != "" {
		out = append(out, p.Spouse)
	}
	if kin {
		for _, k := range p.Kin {
			if k != p.Spouse {
				out = append(out, k)
			}
		}
	}
	if friends {
		out = append(out, p.Friends...)
	}
	return out
}

func nqMissing(x *nqX, rel string) *nqProp {
	g, p := x.g, x.p
	ls := lsSt(g)
	var cands []string
	switch rel {
	case "spouse":
		cands = nqCands(p, true, false, false)
	case "child":
		for _, c := range lsChildren(g, p) {
			cands = append(cands, c.Key)
		}
	default:
		cands = nqCands(p, false, true, true)
	}
	for _, k := range cands {
		q := lvGet(g, k)
		if q == nil || !q.Alive || !nqAdult(g, q) || q.Home != p.Home {
			continue
		}
		if _, j := ls.Jail[k]; j {
			continue
		}
		away := (q.Burg == 0 && q.Dest != 0) || (q.Burg > 0 && q.Burg != q.Home)
		tb := nqBurgOf(q)
		if !away || tb == x.gb || tb <= 0 || g.W.burgByID[tb] == nil {
			continue
		}
		d := x.days(x.gb, tb)
		word := nqRelWord(p, k)
		where := "was last heard of on the road to " + x.bname(tb)
		if q.Burg > 0 {
			where = "went to " + x.bname(tb) + " and has not come back"
		}
		eff := "word"
		if rel != "friend" {
			eff = "bringhome"
		}
		pr := &nqProp{Title: fmt.Sprintf("Find %s, %s's %s", q.Name, p.Name, word), Subj: q.Key, Days: clampi(d*2+6, 8, 40),
			Coin: nqCoin(g, 15+d*5), XP: 25, Rep: 8, Op: 12, Fx: "lostdie"}
		pr.Offer = fmt.Sprintf("\"%s, my %s, %s. It is long past the time they should have been home. I cannot leave the %s. If you are going that way, find them, and tell them to come home, or at the least to write.\"",
			q.Name, word, where, nonEmpty(lvLower(p.Job), "work"))
		pr.Steps = []nqStep{
			{K: "talk", Who: q.Key, Burg: tb, Opt: "Tell them " + p.Name + " is asking after them.", Eff: eff},
			nqReport(p, x.gb, "I found "+q.Name+".", "\"You found them? Are they well?\""),
		}
		return pr
	}
	return nil
}

func nqDebtCollect(x *nqX) *nqProp {
	g, p := x.g, x.p
	if p.Wealth > 200 {
		return nil
	}
	d := x.townsfolk(x.gb, func(q *lvPerson) bool {
		return q.Wealth >= 25 && !lsHas(p.Kin, q.Key) && !lsHas(p.Friends, q.Key)
	})
	if d == nil {
		return nil
	}
	amt := clampi(x.r.Int(15, 45), 10, d.Wealth-5)
	check := "persuasion"
	if p.Trait[3] < 45 {
		check = "intimidate"
	}
	pr := &nqProp{Title: fmt.Sprintf("Collect %s's debt from %s", p.Name, d.Name), Subj: d.Key, Num: amt, Days: 10,
		Coin: nqCoin(g, amt/3+8), XP: 20, Rep: 6, Op: 10, Fx: "debtjail"}
	pr.Offer = fmt.Sprintf("\"%s, the %s, borrowed %d from me and has been 'forgetting' since the last harvest. I cannot ask without a quarrel. You are a stranger: they may listen to you.\"", d.Name, nonEmpty(lvLower(d.Job), "neighbour"), amt)
	pr.Steps = []nqStep{
		{K: "talk", Who: d.Key, Burg: x.gb, Opt: fmt.Sprintf("Tell them %s wants the %d they owe.", p.Name, amt), Check: check, DC: dcMedium,
			Fail: d.Name + " scowls. \"Who are you to ask? Come back when you have manners.\"", Eff: "collect"},
		nqReport(p, x.gb, "I spoke to "+d.Name+" about the money.", "\"Truly? Every coin?\""),
	}
	return pr
}

func nqCreditor(x *nqX) (*lvPerson, int) {
	p := x.p
	if p.Wealth >= 80 {
		return nil, 0
	}
	c := x.townsfolk(x.gb, func(q *lvPerson) bool { return q.Wealth >= 80 && !lsHas(p.Kin, q.Key) })
	if c == nil {
		return nil, 0
	}
	return c, x.r.Int(20, 60)
}

func nqDebtForgive(x *nqX) *nqProp {
	g, p := x.g, x.p
	c, amt := nqCreditor(x)
	if c == nil {
		return nil
	}
	pr := &nqProp{Title: fmt.Sprintf("Ask %s to forgive %s's debt", c.Name, p.Name), Subj: c.Key, Num: amt, Days: 10,
		Coin: nqCoin(g, 12), XP: 20, Rep: 6, Op: 12, Fx: "debtjail"}
	pr.Offer = fmt.Sprintf("\"I owe %s %d and I do not have it. They will have me in the lockup by the new moon. A word from someone with no quarrel with them might soften them. I would be in your debt.\"", c.Name, amt)
	pr.Steps = []nqStep{
		{K: "talk", Who: c.Key, Burg: x.gb, Opt: fmt.Sprintf("Speak for %s and ask them to forgive the debt.", p.Name), Check: "persuasion", DC: dcChallenging,
			Fail: c.Name + " folds their arms. \"Debts are debts. Do not waste my morning.\"", Eff: "forgive"},
		nqReport(p, x.gb, "I spoke to "+c.Name+" for you.", "\"And?\""),
	}
	return pr
}

func nqDebtPay(x *nqX) *nqProp {
	g, p := x.g, x.p
	c, amt := nqCreditor(x)
	if c == nil {
		return nil
	}
	pr := &nqProp{Title: fmt.Sprintf("Settle %s's debt to %s", p.Name, c.Name), Subj: c.Key, Num: amt, Days: 10,
		Coin: nqCoin(g, amt/6), XP: 20, Rep: 8, Op: 22, Extra: "favour", Fx: "debtjail"}
	pr.Offer = fmt.Sprintf("\"I owe %s %d and cannot raise it. They will have me jailed. If you could settle it... I cannot repay you in coin, but I do not forget a kindness.\"", c.Name, amt)
	pr.Steps = []nqStep{
		{K: "pay", Who: c.Key, Burg: x.gb, Cost: amt, Opt: fmt.Sprintf("Hand over the money for %s's debt.", p.Name), Eff: "debtpay"},
		nqReport(p, x.gb, "I have settled your debt.", "\"All of it? I... thank you.\""),
	}
	return pr
}

func nqLetter(x *nqX, parcel bool) *nqProp {
	g, p := x.g, x.p
	goal := p.GoalID
	if goal == "" {
		goal = "family"
	}
	if nqGoalParcel[goal] != parcel {
		return nil
	}
	var rc *lvPerson
	for _, k := range nqCands(p, true, true, true) {
		if q := lvGet(g, k); q != nil && q.Alive && q.Burg > 0 && q.Burg != x.gb && x.days(x.gb, q.Burg) <= 10 {
			rc = q
			break
		}
	}
	if rc == nil {
		near := x.ctx().nearBurgs(x.gb)
		if len(near) == 0 {
			return nil
		}
		b2 := near[x.r.Int(0, mini(len(near), 4)-1)]
		pred := func(q *lvPerson) bool { return q.Job != "" }
		if goal == "faith" {
			pred = func(q *lvPerson) bool {
				return q.Job == "priest" || q.Job == "acolyte" || strings.Contains(q.Role, "priest")
			}
		}
		if rc = x.townsfolk(b2, pred); rc == nil {
			rc = x.townsfolk(b2, func(*lvPerson) bool { return true })
		}
	}
	if rc == nil || rc.Burg <= 0 {
		return nil
	}
	tb := rc.Burg
	d := x.days(x.gb, tb)
	sk := nqSkinFor(g.Genre)
	name, kind, glyph := sk.Letter, "letter", '?'
	if parcel {
		name, kind, glyph = sk.Parcel, "trophy", '■'
	}
	title := fmt.Sprintf("Carry a %s to %s in %s", name, rc.Name, x.bname(tb))
	pr := &nqProp{Title: title, Subj: rc.Key, Days: clampi(d*2+4, 6, 40), Grace: 6, Coin: nqCoin(g, 10+d*6), XP: 20, Rep: 6, Op: 8, Item: name}
	body := fmt.Sprintf(nqLetterBody[goal], p.Name)
	pr.Give = &nqGive{Kind: kind, Name: name + " for " + rc.Name, Glyph: glyph, Col: CMagenta, Text: body, Lore: "From " + p.Name + " of " + x.bname(p.Home) + ", for " + rc.Name + " in " + x.bname(tb) + "."}
	what := "a " + name
	pr.Offer = fmt.Sprintf("\"I have %s for %s in %s, and nobody I trust who is going that way. Take it into their own hands, unopened, and they will pay you for the trouble. Do not be slow about it.\"", what, rc.Name, x.bname(tb))
	pr.Steps = []nqStep{{K: "talk", Who: rc.Key, Burg: tb, Need: "obj", Opt: "Hand over the " + name + " for " + rc.Name + ".", Eff: "deliver"}}
	return pr
}

func nqRevenge(x *nqX, kind string) *nqProp {
	g, p := x.g, x.p
	var rv *lvPerson
	for _, k := range p.Rivals {
		if q := lvGet(g, k); q != nil && q.Alive && nqAdult(g, q) {
			rv = q
			break
		}
	}
	if rv == nil {
		return nil
	}
	tb := nqBurgOf(rv)
	if tb <= 0 || g.W.burgByID[tb] == nil {
		return nil
	}
	d := x.days(x.gb, tb)
	pr := &nqProp{Subj: rv.Key, Days: clampi(d*2+8, 10, 40), XP: 30, Rep: 4, Op: 12, Fx: "feudup"}
	where := ""
	if tb != x.gb {
		where = " in " + x.bname(tb)
	}
	switch kind {
	case "kill":
		if p.Trait[3] >= 55 {
			return nil
		}
		pr.Title = "Be rid of " + rv.Name
		pr.Coin = nqCoin(g, 45)
		pr.Offer = fmt.Sprintf("\"%s, the %s%s, has taken everything from me that matters. I want them gone: dead, if it must be. No names, no witnesses. Do this and I will see you paid.\"", rv.Name, nonEmpty(lvLower(rv.Job), "scoundrel"), where)
		pr.Steps = []nqStep{{K: "kill", Who: rv.Key, Burg: tb}, nqReport(p, x.gb, "I have done what you asked of me.", "\"It is true? Then it is over. Take this.\"")}
	case "scare":
		pr.Title = "Warn off " + rv.Name
		pr.Coin = nqCoin(g, 22)
		pr.Offer = fmt.Sprintf("\"%s%s will not leave me be. I do not want blood. I want them frightened enough to stay away. You look as if you could manage that.\"", rv.Name, where)
		pr.Steps = []nqStep{{K: "talk", Who: rv.Key, Burg: tb, Check: "intimidate", DC: dcMedium, Opt: "Tell them to stay away from " + p.Name + ", or else.",
			Fail: rv.Name + " laughs. \"Run along, stranger. You do not scare me.\"", Eff: "scare"}, nqReport(p, x.gb, "I have given "+rv.Name+" a warning.", "\"Good. Good.\"")}
	default:
		if rv.Secret == "" {
			return nil
		}
		pr.Title = "Expose " + rv.Name
		pr.Coin = nqCoin(g, 26)
		pr.Offer = fmt.Sprintf("\"I know what %s%s is hiding: %s. Nobody will believe me. They might believe a stranger who tells it plain to the right ear: an elder, a priest, a guard. Spare nothing.\"", rv.Name, where, lvLower(rv.Secret))
		pr.Steps = []nqStep{{K: "talk", Role: "elder|priest|guard", Burg: tb, Opt: "Tell them what " + rv.Name + " has been hiding.", Eff: "expose"},
			nqReport(p, x.gb, "I have spread the word about "+rv.Name+".", "\"Let them choke on it.\"")}
	}
	return pr
}

func nqProtect(x *nqX, kind string) *nqProp {
	g, p := x.g, x.p
	ls := lsSt(g)
	pr := &nqProp{Days: 3, Coin: nqCoin(g, 25), XP: 20, Rep: 6, Op: 15}
	switch kind {
	case "child":
		var ch *lvPerson
		for _, c := range lsChildren(g, p) {
			if c.Alive && c.Burg == x.gb && !nqAdult(g, c) {
				ch = c
				break
			}
		}
		if ch == nil {
			return nil
		}
		pr.Title = "Keep watch over " + ch.Name
		pr.Offer = fmt.Sprintf("\"Someone has been watching my %s, %s. I cannot be everywhere, and I must go to work. Stay in town and keep an eye on them for the day.\"", nqRelWord(p, ch.Key), ch.Name)
	case "shop":
		if !lsShopJobs[p.Job] {
			return nil
		}
		pr.Title = "Guard " + p.Name + "'s " + p.Job + " shop"
		pr.Fx = "robbed"
		pr.Offer = "\"There have been men sniffing round my shop and I hear they mean to try a night market. Stand watch in town for the day and keep them honest.\""
	default:
		var k, m *lvPerson
		for _, key := range append([]string{p.Key}, p.Kin...) {
			q := lvGet(g, key)
			if q == nil || !q.Alive || !nqAdult(g, q) || q.Spouse != "" || q.Burg != x.gb {
				continue
			}
			if pk := ls.Wooing[key]; pk != "" {
				if o := lvGet(g, pk); o != nil && o.Alive && o.Spouse == "" && o.Burg == x.gb {
					k, m = q, o
					break
				}
			}
		}
		if k == nil {
			return nil
		}
		pr.Subj, pr.Subj2 = k.Key, m.Key
		pr.Title = fmt.Sprintf("Guard the wedding of %s and %s", k.Name, m.Name)
		pr.Offer = fmt.Sprintf("\"%s and %s are to be wed, and there are those who would like to see it spoiled. Stay in town through the day and see that nothing is spoilt.\"", k.Name, m.Name)
		pr.Op = 20
	}
	pr.Steps = []nqStep{{K: "stay", Burg: x.gb, Mins: 360}, nqReport(p, x.gb, "Everything went quietly.", "\"Thank the stars. Take this.\"")}
	return pr
}

// nqWild picks a wild parasang near a burg, or a ruined town within 14 when ruins is set.
func nqWild(g *Game, hx, hy int, r *Rng, ruins bool) (int, int, string, bool) {
	w := g.W
	if ruins {
		bestD, bx, by, name := 99, 0, 0, ""
		for i := range w.Burgs {
			b := &w.Burgs[i]
			if b.Alive || b.Layer != "surface" {
				continue
			}
			if d := maxi(abs(b.P%w.W-hx), abs(b.P/w.W-hy)); d <= 14 && d < bestD {
				bestD, bx, by, name = d, b.P%w.W, b.P/w.W, "the ruins of "+b.Name
			}
		}
		if bestD < 99 {
			return bx, by, name, true
		}
	}
	off := r.Int(0, 7)
	for i := 0; i < 8; i++ {
		dx, dy := dirs8[(off+i)%8][0], dirs8[(off+i)%8][1]
		for k := 2; k <= 4; k++ {
			x, y := hx+dx*k, hy+dy*k
			if !w.In(x, y) {
				break
			}
			p := w.Idx(x, y)
			if !w.IsWater(p) && w.Burg[p] == 0 && len(w.wondersAt[p]) == 0 {
				return x, y, "the wilds " + cqDir(hx, hy, x, y) + " of town", true
			}
		}
	}
	return 0, 0, "", false
}

func nqHeir(x *nqX, book bool) *nqProp {
	g, p := x.g, x.p
	b := g.W.burgByID[x.gb]
	if b == nil {
		return nil
	}
	hx, hy := b.P%g.W.W, b.P/g.W.W
	wx, wy, where, ok := nqWild(g, hx, hy, x.r, x.r.Chance(0.5))
	if !ok {
		return nil
	}
	sk := nqSkinFor(g.Genre)
	name := sk.Heir[x.r.Int(0, 2)]
	place, noun := "heir", name
	if book {
		place, noun = "book", sk.Book
	}
	full := p.Name + "'s " + noun
	dist := maxi(abs(wx-hx), abs(wy-hy))
	pr := &nqProp{Title: "Recover " + full, Days: clampi(dist*3+8, 10, 45), Coin: nqCoin(g, 25+dist*4), XP: 30, Rep: 6, Op: 15,
		Item: full, Place: place, Zone: ZoneKey(wx, wy, 1, 1, 0), TP: g.W.Idx(wx, wy)}
	pr.Offer = fmt.Sprintf("\"My %s was lost, or taken, and the trail leads to %s. It means more to me than I can say. Bring it back and I will not forget it.\"", noun, where)
	pr.Steps = []nqStep{{K: "place"}, {K: "talk", Who: p.Key, Burg: x.gb, Need: "obj", Opt: "Hand over " + full + ".", Eff: "heirdone"}}
	return pr
}

func nqJailed(x *nqX, innocence bool) *nqProp {
	g, p := x.g, x.p
	ls := lsSt(g)
	minLeft := 2
	if innocence {
		minLeft = 4
	}
	var j *lvPerson
	left := 0
	for _, k := range nqCands(p, true, true, true) {
		q := lvGet(g, k)
		if q == nil || !q.Alive {
			continue
		}
		if rel, ok := ls.Jail[k]; ok && ls.From[k] == x.gb && rel-x.day >= minLeft {
			j, left = q, rel-x.day
			break
		}
	}
	if j == nil {
		return nil
	}
	word := nqRelWord(p, j.Key)
	pr := &nqProp{Subj: j.Key, Days: left, XP: 30, Rep: 8, Op: 20, Fx: "jaildie"}
	if innocence {
		w := x.townsfolk(x.gb, func(q *lvPerson) bool { return q.Key != j.Key && !lsHas(j.Kin, q.Key) })
		if w == nil {
			return nil
		}
		pr.Subj2 = w.Key
		pr.Title = fmt.Sprintf("Prove %s innocent", j.Name)
		pr.Coin = nqCoin(g, 30)
		pr.Offer = fmt.Sprintf("\"They have locked up my %s, %s, for something they did not do. %s saw where they were that night, but will not speak to me. Get it out of them in writing, and take it to the watch.\"", word, j.Name, w.Name)
		pr.Steps = []nqStep{
			{K: "talk", Who: w.Key, Burg: x.gb, Check: "persuasion", DC: dcMedium, Opt: fmt.Sprintf("Ask %s what they saw the night %s was taken.", w.Name, j.Name),
				Fail: w.Name + " looks away. \"I saw nothing. Leave me out of it.\"", Eff: "witness"},
			{K: "talk", Role: "guard|elder", Burg: x.gb, Need: "obj", Opt: "Hand over the witness statement.", Eff: "release"},
			nqReport(p, x.gb, "I got "+j.Name+" out.", "\"They are free? Truly?\""),
		}
		return pr
	}
	fine := 15 + left*3
	pr.Title = fmt.Sprintf("Pay bail for %s", j.Name)
	pr.Coin = nqCoin(g, fine/4)
	pr.Offer = fmt.Sprintf("\"They have locked up my %s, %s, and I do not have the bail. %d would see them out. I would be in your debt more than I can say.\"", word, j.Name, fine)
	pr.Steps = []nqStep{
		{K: "pay", Role: "guard|elder", Burg: x.gb, Cost: fine, Opt: "Pay the bail for " + j.Name + ".", Eff: "bail"},
		nqReport(p, x.gb, "I paid "+j.Name+"'s bail.", "\"You did? Heaven keep you.\""),
	}
	return pr
}

func nqVouch(x *nqX) *nqProp {
	g, p := x.g, x.p
	b := lvGet(g, p.Boss)
	if b == nil || !b.Alive || b.Burg <= 0 || b.Key == p.Key {
		return nil
	}
	d := x.days(x.gb, b.Burg)
	if b.Burg == x.gb {
		d = 0
	}
	pr := &nqProp{Title: fmt.Sprintf("Put in a word for %s with %s", p.Name, b.Name), Subj: b.Key, Days: clampi(d*2+6, 8, 30),
		Coin: nqCoin(g, 18), XP: 20, Rep: 6, Op: 14}
	pr.Offer = fmt.Sprintf("\"%s answers for my place, and I think %s has forgotten what I am worth. I cannot beg. A stranger speaking well of me might do what my work has not.\"", b.Name, b.Name)
	pr.Steps = []nqStep{
		{K: "talk", Who: b.Key, Burg: b.Burg, Check: "persuasion", DC: dcMedium, Opt: "Put in a good word for " + p.Name + ".",
			Fail: b.Name + " frowns. \"I have my own opinion of them. Thank you, but no.\"", Eff: "vouch"},
		nqReport(p, x.gb, "I spoke to "+b.Name+" for you.", "\"And what did they say?\""),
	}
	return pr
}

func nqMatchmake(x *nqX) *nqProp {
	g, p := x.g, x.p
	if p.Spouse != "" {
		return nil
	}
	c := x.ctx()
	t := x.townsfolk(x.gb, func(q *lvPerson) bool { return lsEligible(c, p, q, p.lvAge(c.yr)) })
	if t == nil {
		return nil
	}
	sk := nqSkinFor(g.Genre)
	pr := &nqProp{Title: fmt.Sprintf("Carry %s's courtship gift to %s", p.Name, t.Name), Subj: t.Key, Days: 8, Coin: nqCoin(g, 18), XP: 20, Rep: 4, Op: 14, Item: sk.Gift}
	pr.Give = &nqGive{Kind: "trophy", Name: sk.Gift + " from " + p.Name, Glyph: '♥', Col: CMagenta, Lore: "A gift meant for " + t.Name + "."}
	pr.Offer = fmt.Sprintf("\"I have admired %s for a long time and I cannot get the words out. A %s, from a stranger, with my name on it: that I can manage. Will you take it to them?\"", t.Name, sk.Gift)
	pr.Steps = []nqStep{
		{K: "talk", Who: t.Key, Burg: x.gb, Need: "obj", Opt: "Hand over " + p.Name + "'s gift to " + t.Name + ".", Eff: "court"},
		nqReport(p, x.gb, "I gave "+t.Name+" your gift.", "\"And did they smile? Tell me everything.\""),
	}
	return pr
}

func nqStolen(x *nqX) *nqProp {
	g, p := x.g, x.p
	for _, k := range lsCrimKeys(x.ctx()) {
		c := lvGet(g, k)
		if c == nil || !c.Alive || c.Burg <= 0 || !lsHas(p.Rivals, k) {
			continue
		}
		amt := x.r.Int(20, 35)
		d := x.days(x.gb, c.Burg)
		if c.Burg == x.gb {
			d = 0
		}
		check := "persuasion"
		if p.Trait[3] < 50 {
			check = "intimidate"
		}
		pr := &nqProp{Title: fmt.Sprintf("Get back what %s stole", c.Name), Subj: c.Key, Num: amt, Days: clampi(d*2+8, 8, 30), Coin: nqCoin(g, 15), XP: 25, Rep: 6, Op: 15}
		pr.Offer = fmt.Sprintf("\"%s robbed me, everyone knows it, and the watch has done nothing. Find them and make them give back what is mine: about %d in coin. Do it however you must.\"", c.Name, amt)
		pr.Steps = []nqStep{
			{K: "talk", Who: c.Key, Burg: c.Burg, Check: check, DC: dcChallenging, Opt: "Demand what " + c.Name + " took from " + p.Name + ".",
				Fail: c.Name + " sneers. \"Prove it, and then try that again.\"", Eff: "stolen"},
			{K: "talk", Who: p.Key, Burg: x.gb, Need: "obj", Opt: "Hand over what " + c.Name + " took.", Eff: "gotback"},
		}
		return pr
	}
	return nil
}

func nqFeud(x *nqX) *nqProp {
	g, p := x.g, x.p
	var r *lvPerson
	for _, k := range p.Rivals {
		if q := lvGet(g, k); q != nil && q.Alive && q.Burg == x.gb && nqAdult(g, q) {
			r = q
			break
		}
	}
	if r == nil {
		return nil
	}
	pr := &nqProp{Title: fmt.Sprintf("Make peace between %s and %s", p.Name, r.Name), Subj: r.Key, Days: 8, Coin: nqCoin(g, 24), XP: 30, Rep: 8, Op: 14, Fx: "feudup"}
	pr.Offer = fmt.Sprintf("\"%s and I have been at each other's throats for years, and it will end with somebody hurt. I cannot be the one to hold out a hand. Someone from outside might.\"", r.Name)
	pr.Steps = []nqStep{
		{K: "talk", Who: r.Key, Burg: x.gb, Check: "persuasion", DC: dcChallenging, Opt: "Ask " + r.Name + " to make peace with " + p.Name + ".",
			Fail: r.Name + " bristles. \"After what they said? Never.\"", Eff: "peace"},
		nqReport(p, x.gb, "I talked to "+r.Name+".", "\"What did they say?\""),
	}
	return pr
}

func nqFuneral(x *nqX) *nqProp {
	g, p := x.g, x.p
	var dead *lvPerson
	for _, k := range nqCands(p, true, true, false) {
		if q := lvGet(g, k); q != nil && !q.Alive && q.Died >= lvYear(g)-1 {
			dead = q
			break
		}
	}
	if dead == nil {
		return nil
	}
	near := x.ctx().nearBurgs(x.gb)
	if len(near) == 0 {
		return nil
	}
	b2 := near[0]
	d := x.days(x.gb, b2)
	pr := &nqProp{Title: "Fetch a priest for " + dead.Name, Days: d*2 + 4, Coin: nqCoin(g, 20), XP: 20, Rep: 6, Op: 14}
	pr.Offer = fmt.Sprintf("\"%s is dead, and we have no priest to see them off properly. There is one in %s. Ask them to come. The dead should not wait.\"", dead.Name, x.bname(b2))
	pr.Steps = []nqStep{
		{K: "talk", Role: "priest", Burg: b2, Opt: "Ask the priest to come and bury " + dead.Name + ".", Eff: "priest"},
		nqReport(p, x.gb, "A priest is coming for "+dead.Name+".", "\"Thank you. Truly.\""),
	}
	return pr
}

func nqSick(x *nqX) *nqProp {
	g, p := x.g, x.p
	ls := lsSt(g)
	for _, k := range nqCands(p, true, true, true) {
		q := lvGet(g, k)
		sk, ok := ls.Sick[k]
		if q == nil || !q.Alive || !ok || q.Burg != x.gb {
			continue
		}
		pr := &nqProp{Title: "Bring a remedy for " + q.Name, Subj: q.Key, Days: clampi(sk.Until-x.day, 4, 14), Coin: nqCoin(g, 22), XP: 25, Rep: 8, Op: 25, Fx: "sickdie"}
		pr.Offer = fmt.Sprintf("\"%s is ill, and getting worse. Herbs of healing, a tonic, anything that mends: I cannot afford the apothecary. If you have such a thing, I will take it gladly.\"", q.Name)
		pr.Steps = []nqStep{{K: "item", Who: p.Key, Burg: x.gb, Need: "herb", Opt: "Hand over a remedy for " + q.Name + ".", Eff: "cure"}}
		return pr
	}
	return nil
}

func nqRescue(x *nqX) *nqProp {
	g, p := x.g, x.p
	ls := lsSt(g)
	b := g.W.burgByID[x.gb]
	if b == nil {
		return nil
	}
	var q *lvPerson
	for _, k := range nqCands(p, true, true, false) {
		o := lvGet(g, k)
		if _, jailed := ls.Jail[k]; o != nil && o.Alive && o.Burg == x.gb && !jailed && o.Home == p.Home {
			if _, sick := ls.Sick[k]; !sick {
				q = o
				break
			}
		}
	}
	if q == nil {
		return nil
	}
	hx, hy := b.P%g.W.W, b.P/g.W.W
	wx, wy, where, ok := nqWild(g, hx, hy, x.r, false)
	if !ok {
		return nil
	}
	pr := &nqProp{Title: "Rescue " + q.Name, Subj: q.Key, Days: 8, Grace: 0, Coin: nqCoin(g, 50), XP: 50, Rep: 12, Op: 30, Fx: "captivedie",
		Place: "captive", Zone: ZoneKey(wx, wy, 1, 1, 0), TP: g.W.Idx(wx, wy)}
	pr.Offer = fmt.Sprintf("\"%s was taken on the road this morning. Bandits: they hold them in %s. Please. I am no fighter, and the ransom would ruin us. Bring them out alive.\"", q.Name, where)
	pr.Steps = []nqStep{
		{K: "talk", Role: "captive", Opt: "Cut " + q.Name + "'s bonds.", Eff: "rescue"},
		nqReport(p, x.gb, "I brought "+q.Name+" out of the bandits' hands.", "\"Alive? They are alive?\""),
	}
	return pr
}

func nqStake(x *nqX) *nqProp {
	g, p := x.g, x.p
	ls := lsSt(g)
	if len(ls.Vacant[x.gb]) == 0 || lsShopJobs[p.Job] {
		return nil
	}
	trade := ls.Vacant[x.gb][0]
	cost := 40
	pr := &nqProp{Title: fmt.Sprintf("Stake %s's new %s shop", p.Name, trade), Days: 6, Coin: cost * 3 / 4, XP: 25, Rep: 8, Op: 25, Extra: "favour"}
	pr.Offer = fmt.Sprintf("\"The old %s's shop stands empty, and I have been saving for years, and I am still %d short. Stake me, and you will have your share back before the year is out, and a friend in this town.\"", trade, cost)
	pr.Steps = []nqStep{{K: "pay", Who: p.Key, Burg: x.gb, Cost: cost, Opt: "Hand over the coin to stake " + p.Name + "'s shop.", Eff: "stake"}}
	return pr
}

func nqWatch(x *nqX) *nqProp {
	g, p := x.g, x.p
	for _, k := range lsCrimKeys(x.ctx()) {
		c := lvGet(g, k)
		if c == nil || !c.Alive || c.Burg != x.gb {
			continue
		}
		pr := &nqProp{Title: "Report " + c.Name + " to the watch", Subj: c.Key, Days: 6, Coin: nqCoin(g, 15), XP: 20, Rep: 10, Op: 12}
		pr.Offer = fmt.Sprintf("\"Everyone in this street knows %s is behind the thefts. Nobody will say it to the watch: they are afraid. Will you?\"", c.Name)
		pr.Steps = []nqStep{
			{K: "talk", Role: "guard", Burg: x.gb, Opt: "Report " + c.Name + " to the watch.", Eff: "watch"},
			nqReport(p, x.gb, "I reported "+c.Name+" to the guard.", "\"At last. Thank you.\""),
		}
		return pr
	}
	return nil
}

func nqFame(x *nqX) *nqProp {
	g, p := x.g, x.p
	if p.GoalID != "fame" {
		return nil
	}
	near := x.ctx().nearBurgs(x.gb)
	if len(near) < 2 {
		return nil
	}
	b1, b2 := near[0], near[1]
	pr := &nqProp{Title: fmt.Sprintf("Spread the name of %s", p.Name), Days: 14, Coin: nqCoin(g, 16), XP: 20, Rep: 4, Op: 12}
	pr.Offer = fmt.Sprintf("\"I want my name known, and I am done waiting for it to travel on its own. Tell people in %s and in %s what I have done: the more the better, and a little polish is no sin.\"", x.bname(b1), x.bname(b2))
	pr.Steps = []nqStep{
		{K: "talk", Role: "*", Burg: b1, Opt: "Tell them of " + p.Name + "'s name and deeds.", Eff: "spread"},
		{K: "talk", Role: "*", Burg: b2, Opt: "Tell them of " + p.Name + "'s name and deeds.", Eff: "spread"},
		nqReport(p, x.gb, "I have spread word of you.", "\"Splendid.\""),
	}
	return pr
}
