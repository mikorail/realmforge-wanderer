package engine

import (
	"fmt"
	"strings"
)

// Deep Murim systems (prefix md):
// 1. Hidden masters disguised as beggars in towns (mdSpawnBeggarMaster, mdOfferAlms, mdUnmaskMaster).
// 2. Sect grudges when killing disciples, escalating through 3 avenger stages (mdOnKill, mdCheckGrudgeHunter).
// 3. Martial lineage recorded and passed to an heir (mdRecordLineage, mdApplyLineageToHeir).
// 4. Closed-door seclusion for a week/season to heal deviation and break bottlenecks (mdSeclude).
// 5. Forbidden demonic arts with life drain, demonic aura, and qi deviation (mdLearnDemonic, mdUseDemonic, mdQiDeviation).

type mdGrudge struct {
	SectName   string `json:"sect"`
	VictimName string `json:"victim"`
	DueDay     int    `json:"due"`
	Stage      int    `json:"stage"` // 1 = Senior Martial Brother, 2 = Sect Elder, 3 = Grandmaster Patriarch
}

type mdLineage struct {
	ParentName string   `json:"parent"`
	Sect       string   `json:"sect,omitempty"`
	Rank       int      `json:"rank"`
	Realm      int      `json:"realm"`
	MaxAttr    int      `json:"maxAttr"`
	SecretTech string   `json:"secretTech"`
	Techs      []string `json:"techs,omitempty"`
	Applied    bool     `json:"applied,omitempty"`
}

type mdState struct {
	Grudges       []*mdGrudge `json:"grudges,omitempty"`
	Lineage       *mdLineage  `json:"lineage,omitempty"`
	Breakthroughs int         `json:"breakthroughs,omitempty"`
	Corruption    int         `json:"corruption,omitempty"`
	DemonicArts   []string    `json:"demonic,omitempty"`
	Techs         []string    `json:"techs,omitempty"`
	Blessed       bool        `json:"blessed,omitempty"`
	AuraApplied   bool        `json:"aura,omitempty"`
}

var mdSecretTechNames = map[string]string{
	"md_palm":   "Heaven-Shattering Palm",
	"md_step":   "Cloud-Treading Step",
	"md_finger": "Meridian-Sealing Finger",
}

var mdDemonicArtNames = map[string]string{
	"blood_sutra": "Blood Sea Sutra",
	"soul_devour": "Soul-Devouring Claw",
	"corpse_qi":   "Corpse-Refining Qi",
}

func (g *Game) md() *mdState {
	return g.ext("md", func() any { return &mdState{} }).(*mdState)
}

func init() {
	unLocalKeys = append(unLocalKeys, "md")
	zoneHooks = append(zoneHooks, mdZoneHook)
	talkHooks = append(talkHooks, mdTalkHook)
	killHooks = append(killHooks, mdOnKill)
	turnHooks = append(turnHooks, func(g *Game) {
		if g.Cur != nil && !g.OnWorld && g.Turn%30 == 0 {
			g.mdCheckGrudgeHunter()
		}
	})
	enterHooks = append(enterHooks, func(g *Game, zn *Zone) {
		if zn != nil && !g.OnWorld {
			g.mdCheckGrudgeHunter()
		}
	})
	sideHooks = append(sideHooks, mdSideHook)
}

func mdZoneHook(g *Game, zn *Zone, r *Rng) {
	if zn == nil || zn.Kind != "town" || zn.Z != 0 {
		return
	}
	if zn.ZX == 1 && zn.ZY == 1 {
		mdSpawnBeggarMaster(g, zn, r)
	}
}

// ---- 1. Hidden Masters Disguised as Beggars --------------------------------

func mdSpawnBeggarMaster(g *Game, zn *Zone, _ *Rng) *Actor {
	if g == nil || zn == nil || zn.Kind != "town" {
		return nil
	}
	for _, a := range zn.Actors {
		if a.HP > 0 && (a.Role == "md_beggar" || a.Has("md_hidden_master")) {
			return a
		}
	}
	sr := NewRng(g.W.Seed + ":md:beggar:" + zn.Key)
	a := &Actor{
		ID:      g.newID(),
		Name:    "ragged beggar",
		Proper:  false,
		Glyph:   '@',
		F:       CGray,
		Faction: nonEmpty(zn.Faction, "state:0"),
		Level:   1,
		Attr:    [6]int{12, 12, 12, 14, 16, 12},
		HP:      24,
		MaxHP:   24,
		AVNat:   1,
		DVNat:   2,
		Speed:   100,
		Body:    MakeBody("humanoid"),
		AI:      "still",
		Role:    "md_beggar",
		Species: "human",
		Desc:    "An unassuming beggar in patched hemp robes, cradling a chipped alms bowl with strangely steady hands.",
	}
	a.Set("md_hidden_master")
	x, y, ok := zn.FreeSpot(sr, nil)
	if !ok {
		x, y = clampi(MW/2, 1, MW-2), clampi(MH/2, 1, MH-2)
	}
	a.X, a.Y, a.HomeX, a.HomeY = x, y, x, y
	zn.Actors = append(zn.Actors, a)
	return a
}

func mdTalkHook(g *Game, a *Actor) []Option {
	if a == nil || (!a.Has("md_hidden_master") && a.Role != "md_beggar") {
		return nil
	}
	if a.Has("md_unmasked") || a.Has("md_master_blessed") {
		return nil
	}
	return []Option{
		{
			Text: "{{G|Offer alms of food or coin to the ragged beggar.}}",
			Do: func() {
				g.mdOfferAlms(a)
				g.Mode, g.Talk = "main", nil
			},
		},
		{
			Text: "{{R|Kick the beggar's alms bowl aside and mock them.}}",
			Do: func() {
				g.mdUnmaskMaster(a)
				g.Mode, g.Talk = "main", nil
			},
		},
	}
}

func (g *Game) mdOfferAlms(beggar *Actor) string {
	if g == nil || g.P == nil {
		return ""
	}
	// Deduct a food item if carried, else up to 5 gold if available.
	spent := false
	for _, it := range g.P.Inv {
		if it.Kind == "food" || it.Kind == "dish" {
			g.takeOne(it)
			spent = true
			break
		}
	}
	if !spent && g.P.Gold > 0 {
		g.P.Gold -= mini(g.P.Gold, 5)
	}

	techPool := []string{"md_palm", "md_step", "md_finger"}
	sr := NewRng(fmt.Sprintf("%s:md:alms:%d", g.W.Seed, g.Turn))
	tech := techPool[sr.Int(0, len(techPool)-1)]

	st := g.md()
	if !mrHas(st.Techs, tech) {
		st.Techs = append(st.Techs, tech)
	}
	if !mrHas(g.mr().Techs, tech) {
		g.mr().Techs = append(g.mr().Techs, tech)
	}
	st.Blessed = true
	g.P.Set("md_master_blessed")
	g.P.Set(tech)

	if beggar != nil {
		beggar.Name = "Hidden Grandmaster"
		beggar.Title = "recluse of the jianghu"
		beggar.F = CYellow
		beggar.Set("md_master_blessed")
	}

	g.gainXP(80)
	techName := mdSecretTechNames[tech]
	g.Msg(fmt.Sprintf("{{Y|The beggar's eyes flash with ancient clarity! Revealing themselves as a Hidden Grandmaster, they impart %s (+80 XP).}}", techName))
	return tech
}

func (g *Game) mdUnmaskMaster(beggar *Actor) {
	if g == nil || beggar == nil {
		return
	}
	beggar.Name = "Hidden Grandmaster"
	beggar.Title = "unmasked master"
	beggar.Role = "md_grandmaster"
	beggar.Level = 12
	beggar.MaxHP = 120
	beggar.HP = 120
	beggar.AVNat = 8
	beggar.DVNat = 8
	beggar.Attr = [6]int{22, 22, 20, 18, 22, 18}
	beggar.AI = "hostile"
	beggar.Faction = "outlaws"
	beggar.Angry = true
	beggar.F = CRed
	beggar.Set("Hostile")
	beggar.Set("md_unmasked")
	if len(beggar.Body) > 0 && beggar.Body[0].Natural != nil {
		beggar.Body[0].Natural.DN = 2
		beggar.Body[0].Natural.DS = 8
		beggar.Body[0].Natural.PV = 7
	}
	g.Msg("{{R|The beggar rises with a thunderous burst of qi, unmasking as a Level 12 Grandmaster!}}")
}

// ---- 2. Sect Grudges -------------------------------------------------------

func mdActorSectInfo(g *Game, d *Actor) (bool, string) {
	if d == nil {
		return false, ""
	}
	if strings.HasPrefix(d.Faction, "mr:") {
		sid := strings.TrimPrefix(d.Faction, "mr:")
		if g != nil {
			if s := g.SectByID(sid); s != nil {
				return true, s.Name
			}
		}
		return true, sid
	}
	if strings.HasPrefix(d.Faction, "sect_") {
		return true, strings.TrimPrefix(d.Faction, "sect_")
	}
	for k, v := range d.Flags {
		if !v {
			continue
		}
		if strings.HasPrefix(k, "mrd:") || strings.HasPrefix(k, "mre:") || strings.HasPrefix(k, "mrl:") || strings.HasPrefix(k, "mrhall:") {
			parts := strings.SplitN(k, ":", 2)
			if len(parts) == 2 && g != nil {
				if s := g.SectByID(parts[1]); s != nil {
					return true, s.Name
				}
				return true, parts[1]
			}
		}
		if strings.Contains(k, "mr_") {
			return true, nonEmpty(d.Faction, "Iron Lotus Sect")
		}
	}
	if d.Role == "disciple" {
		return true, nonEmpty(d.Faction, "Iron Lotus Sect")
	}
	return false, ""
}

func mdOnKill(g *Game, d, src *Actor) {
	if g == nil || d == nil || src != g.P {
		return
	}
	ok, sectName := mdActorSectInfo(g, d)
	if !ok {
		return
	}
	st := g.md()
	for _, gr := range st.Grudges {
		if gr.SectName == sectName {
			gr.VictimName = d.Name
			if gr.Stage < 1 {
				gr.Stage = 1
			}
			gr.DueDay = g.Day() + 3
			g.Msg(fmt.Sprintf("{{R|%s deepens their blood grudge over %s!}}", sectName, d.Name))
			return
		}
	}
	st.Grudges = append(st.Grudges, &mdGrudge{
		SectName:   sectName,
		VictimName: d.Name,
		DueDay:     g.Day() + 3,
		Stage:      1,
	})
	g.Msg(fmt.Sprintf("{{R|You have slain %s of %s! Their sect swears a blood grudge.}}", d.Name, sectName))
}

func (g *Game) mdCheckGrudgeHunter() *Actor {
	if g == nil || g.Cur == nil || g.P == nil {
		return nil
	}
	st := g.md()
	day := g.Day()
	for _, gr := range st.Grudges {
		if gr.Stage < 1 || gr.Stage > 3 || day < gr.DueDay {
			continue
		}
		var title string
		var lvl, hp, av, dv int
		switch gr.Stage {
		case 1:
			title = "Senior Martial Brother"
			lvl = maxi(g.P.Level+1, 5)
			hp, av, dv = 55, 4, 4
		case 2:
			title = "Sect Elder"
			lvl = maxi(g.P.Level+3, 8)
			hp, av, dv = 85, 6, 6
		default:
			title = "Grandmaster Patriarch"
			lvl = maxi(g.P.Level+5, 12)
			hp, av, dv = 130, 8, 8
		}
		sr := NewRng(fmt.Sprintf("%s:md:avenger:%s:%d", g.W.Seed, gr.SectName, gr.Stage))
		avActor := g.Spawn("bandit", lvl, sr)
		avActor.Name = fmt.Sprintf("%s of %s", title, gr.SectName)
		avActor.Proper = true
		avActor.Role = "md_avenger"
		avActor.AI = "hostile"
		avActor.Faction = "outlaws"
		avActor.Angry = true
		avActor.F = CRed
		avActor.Level = lvl
		avActor.MaxHP = hp
		avActor.HP = hp
		avActor.AVNat = av
		avActor.DVNat = dv
		avActor.Set("Hostile")
		avActor.Set("md_avenger")

		x, y := clampi(g.P.X+1, 1, MW-2), clampi(g.P.Y, 1, MH-2)
		if !g.Cur.Walkable(x, y) || g.Cur.ActorAt(x, y) != nil {
			if fx, fy, ok := g.Cur.FreeSpot(sr, func(px, py int) bool {
				d := abs(px-g.P.X) + abs(py-g.P.Y)
				return d >= 1 && d <= 6
			}); ok {
				x, y = fx, fy
			}
		}
		avActor.X, avActor.Y = x, y
		g.Cur.Actors = append(g.Cur.Actors, avActor)

		g.Msg(fmt.Sprintf("{{R|\"%s! Pay with your life for slaying %s!\" cries the %s!}}", g.P.Name, gr.VictimName, avActor.Name))
		gr.Stage++
		gr.DueDay = day + 5
		return avActor
	}
	return nil
}

// ---- 3. Martial Lineage Passed to an Heir ----------------------------------

func (g *Game) mdRecordLineage() *mdLineage {
	if g == nil || g.P == nil {
		return nil
	}
	martialAttrs := []int{STR, AGI, TOU, WIL}
	bestAttr := STR
	for _, a := range martialAttrs {
		if g.P.Attr[a] > g.P.Attr[bestAttr] {
			bestAttr = a
		}
	}
	var allTechs []string
	for _, t := range g.mr().Techs {
		if !mrHas(allTechs, t) {
			allTechs = append(allTechs, t)
		}
	}
	for _, t := range g.md().Techs {
		if !mrHas(allTechs, t) {
			allTechs = append(allTechs, t)
		}
	}
	secret := "md_palm"
	if len(g.md().Techs) > 0 {
		secret = g.md().Techs[0]
	} else if len(allTechs) > 0 {
		secret = allTechs[0]
	}
	lin := &mdLineage{
		ParentName: g.P.Name,
		Sect:       g.mr().Sect,
		Rank:       g.mr().Rank,
		Realm:      g.mr().Realm,
		MaxAttr:    bestAttr,
		SecretTech: secret,
		Techs:      allTechs,
	}
	g.md().Lineage = lin
	return lin
}

func (g *Game) mdApplyLineageToHeir() bool {
	if g == nil || g.P == nil {
		return false
	}
	lin := g.md().Lineage
	if lin == nil {
		return false
	}
	attr := clampi(lin.MaxAttr, 0, 5)
	g.P.Attr[attr] += 2
	tech := nonEmpty(lin.SecretTech, "md_palm")
	if !mrHas(g.md().Techs, tech) {
		g.md().Techs = append(g.md().Techs, tech)
	}
	if !mrHas(g.mr().Techs, tech) {
		g.mr().Techs = append(g.mr().Techs, tech)
	}
	g.P.Set(tech)
	g.P.Set("Lineage Disciple")
	g.P.Set("md_lineage_disciple")
	if lin.Realm > 0 && g.mr().Realm < 1 {
		g.mr().Realm = 1
		g.mrApply()
	}
	lin.Applied = true
	g.Msg(fmt.Sprintf("{{C|You inherit %s's martial foundation (+2 %s, %s, Lineage Disciple).}}", lin.ParentName, AttrShort[attr], nonEmpty(mdSecretTechNames[tech], tech)))
	return true
}

// ---- 4. Closed-Door Seclusion ----------------------------------------------

func (g *Game) mdSeclude(days int) string {
	if g == nil || g.P == nil {
		return ""
	}
	if days <= 0 {
		days = 7
	}
	days = clampi(days, 7, 60)
	if g.nearHostile() {
		msg := "You cannot enter closed-door seclusion while enemies are near."
		g.Msg(msg)
		return msg
	}

	g.Turn += days * DayLen
	st := g.md()

	// Clear qi deviation and calm corruption.
	delete(g.P.Flags, "md_deviation")
	delete(g.P.Status, "confused")
	delete(g.P.Status, "stunned")
	g.mr().Dev = 0
	st.Corruption = maxi(0, st.Corruption-days*4)
	if st.Corruption < 50 && st.AuraApplied {
		delete(g.P.Flags, "md_demonic_aura")
		g.P.Attr[STR] -= 2
		g.P.Attr[EGO] += 2
		st.AuraApplied = false
	}

	bonusText := ""
	if st.Breakthroughs < 3 {
		st.Breakthroughs++
		g.P.Attr[WIL]++
		g.P.Attr[STR]++
		g.P.MaxHP += 5
		bonusText = " Bottleneck broken (+1 WIL, +1 STR, +5 MaxHP)!"
	}
	g.P.HP = g.P.MaxHP
	g.mr().Qi = g.mrMaxQi()

	msg := fmt.Sprintf("You emerge from %d days of closed-door seclusion with meridians cleansed and full vitality.%s", days, bonusText)
	g.Msg("{{C|" + msg + "}}")
	return msg
}

// ---- 5. Forbidden Demonic Arts ---------------------------------------------

func (g *Game) mdLearnDemonic(artID string) bool {
	if g == nil || g.P == nil || artID == "" {
		return false
	}
	st := g.md()
	if !mrHas(st.DemonicArts, artID) {
		st.DemonicArts = append(st.DemonicArts, artID)
	}
	g.P.Set(artID)
	g.P.Set("md_art:" + artID)
	name := nonEmpty(mdDemonicArtNames[artID], artID)
	g.Msg(fmt.Sprintf("{{M|You inscrive the forbidden %s into your meridians.}}", name))
	return true
}

func (g *Game) mdUseDemonic(artID string, tgt *Actor) int {
	if g == nil || g.P == nil {
		return 0
	}
	st := g.md()
	if !mrHas(st.DemonicArts, artID) {
		g.mdLearnDemonic(artID)
	}
	sr := NewRng(fmt.Sprintf("%s:md:demonic:%d:%s", g.W.Seed, g.Turn, artID))
	dmg := sr.Dice(2, 8) + maxi(2, g.P.M(WIL)+6)
	if tgt != nil && tgt.HP > 0 {
		before := tgt.HP
		g.hurt(tgt, dmg, g.P, "demonic")
		dealt := maxi(1, before-maxi(0, tgt.HP))
		if dealt > dmg {
			dmg = dealt
		}
	}
	drain := maxi(1, dmg/2)
	g.P.HP = mini(g.P.MaxHP, g.P.HP+drain)

	st.Corruption += 15
	if st.Corruption >= 50 {
		g.P.Set("md_demonic_aura")
		if !st.AuraApplied {
			g.P.Attr[STR] += 2
			g.P.Attr[EGO] -= 2
			st.AuraApplied = true
			g.Msg("{{M|A thick demonic aura coils around you (+2 STR, -2 EGO).}}")
		}
	}
	if st.Corruption >= 100 {
		mdQiDeviation(g)
	}
	return dmg
}

func mdQiDeviation(g *Game) {
	if g == nil || g.P == nil {
		return
	}
	g.P.Set("md_deviation")
	g.P.AddSt("confused", 8)
	loss := maxi(2, g.P.MaxHP/5)
	g.P.HP = maxi(1, g.P.HP-loss)
	g.mr().Dev++
	g.Msg(fmt.Sprintf("{{R|Qi deviation! Demonic qi tears through your mind and meridians (-%d HP, Confused)!}}", loss))
}

func mdSideHook(g *Game) []string {
	st := g.md()
	var out []string
	if st.Corruption > 0 {
		out = append(out, fmt.Sprintf("{{M|Corruption: %d%%}}", st.Corruption))
	}
	if g.P != nil && g.P.Has("md_deviation") {
		out = append(out, "{{R|Qi Deviation}}")
	}
	return out
}
