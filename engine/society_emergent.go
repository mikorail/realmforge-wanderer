package engine

import (
	"fmt"
)

// Living Society & Emergent Drama (prefix se):
// 1. Dynamic crime scenes, town guard murder investigations, and public courtroom trials with jury verdicts.
// 2. Royal wedding processions, funeral corteges, and triumphant army marches parade down city high streets.
// 3. Labor guild strikes, bread-line riots, and spontaneous philosophical tavern debates that boil over into brawls.
// 4. Travelling theatre troupes staging satiric plays and epic ballads recounting the player's own past exploits and deeds.
// 5. Seasonal livestock drives and regional harvest trade fairs bringing exotic merchant caravans to town squares.

type seState struct {
	SolvedCrimes   map[string]bool `json:"crimes,omitempty"`
	StrikesSettled map[string]bool `json:"strikes,omitempty"`
	RiotsCalmed    map[string]bool `json:"riots,omitempty"`
	TrialsAttended int             `json:"trials,omitempty"`
	PlaysSeen      int             `json:"plays,omitempty"`
	DebatesWon     int             `json:"debates,omitempty"`
}

func (g *Game) se() *seState {
	st := g.ext("se", func() any { return &seState{} }).(*seState)
	if st.SolvedCrimes == nil {
		st.SolvedCrimes = map[string]bool{}
	}
	if st.StrikesSettled == nil {
		st.StrikesSettled = map[string]bool{}
	}
	if st.RiotsCalmed == nil {
		st.RiotsCalmed = map[string]bool{}
	}
	return st
}

func init() {
	unLocalKeys = append(unLocalKeys, "se")
	zoneHooks = append(zoneHooks, seZoneHook)
	turnHooks = append(turnHooks, seTurnHook)
	talkHooks = append(talkHooks, seTalkHook)
}

func seTurnHook(g *Game) {
	// Ambient citizen drama and town reactions
}

func seZoneHook(g *Game, zn *Zone, r *Rng) {
	if g == nil || zn == nil || zn.Z != 0 || zn.Kind != "town" {
		return
	}
	sr := NewRng(fmt.Sprintf("%s:se:%s", g.Seed, zn.Key))

	// Roll for dramatic civic events in this town
	eventRoll := sr.Int(0, 100)
	switch {
	case eventRoll < 20:
		seSpawnCrimeScene(g, zn, sr)
	case eventRoll < 40:
		seSpawnCourtroomTrial(g, zn, sr)
	case eventRoll < 60:
		seSpawnProcession(g, zn, sr)
	case eventRoll < 75:
		seSpawnGuildStrikeOrRiot(g, zn, sr)
	case eventRoll < 90:
		seSpawnTheatreTroupe(g, zn, sr)
	default:
		seSpawnTradeFairOrLivestock(g, zn, sr)
	}

	// Always seed a philosophical debate table in the tavern
	seSpawnTavernDebate(g, zn, sr)
}

func seSpawnCrimeScene(g *Game, zn *Zone, sr *Rng) {
	// Find a secluded alleyway (e.g. y ~ 16, x ~ 18)
	cx, cy := 18, 16
	if !zn.In(cx, cy) || !zn.At(cx, cy).D().Walk {
		return
	}
	// Chalk outline and bloodstain
	zn.Set(cx, cy, TCobble)
	zn.Notes[cy*MW+cx] = "A chalk outline surrounds a dried puddle of dark crimson blood. Signs of a struggle."

	// Investigator guard
	inv := g.Spawn("human", zn.Level, sr)
	if inv != nil {
		inv.Name = "Chief Inquisitor Vane"
		inv.Role = "se_investigator"
		inv.Faction = "guards"
		inv.Desc = "Kneeling over the chalk outline with a brass loupe, examining scuff marks."
		inv.X, inv.Y = cx+1, cy
		zn.Actors = append(zn.Actors, inv)
	}

	// Cautious sentry cordoning off the alley
	sentry := g.Spawn("human", zn.Level, sr)
	if sentry != nil {
		sentry.Name = "Alley Guard"
		sentry.Role = "se_sentry"
		sentry.Faction = "guards"
		sentry.Desc = "Keeping bystanders and rubberneckers back from the cordoned crime scene."
		sentry.X, inv.Y = cx-1, cy
		zn.Actors = append(zn.Actors, sentry)
	}
}

func seSpawnCourtroomTrial(g *Game, zn *Zone, sr *Rng) {
	// Inside town hall / courthouse chamber (e.g., around x=40, y=6)
	mx, my := 40, 6
	if !zn.In(mx, my) {
		return
	}
	zn.Set(mx, my, TThrone)
	zn.Notes[my*MW+mx] = "The Magistrate's Oaken Bench of High Justice."

	magistrate := g.Spawn("human", zn.Level+1, sr)
	if magistrate != nil {
		magistrate.Name = "Magistrate Corvo"
		magistrate.Role = "se_magistrate"
		magistrate.Faction = "citizens"
		magistrate.Desc = "Clad in ermine-trimmed robes, holding a polished gavel and an ancient codex of law."
		magistrate.X, magistrate.Y = mx, my
		zn.Actors = append(zn.Actors, magistrate)
	}

	// Defendant in the dock
	def := g.Spawn("human", zn.Level, sr)
	if def != nil {
		def.Name = "Aldo the Accused"
		def.Role = "se_defendant"
		def.Faction = "citizens"
		def.Desc = "Pale and shackled at the wrists, pleading innocence to charges of grain smuggling."
		def.X, def.Y = mx, my+2
		zn.Actors = append(zn.Actors, def)
	}

	// Jurors in stands
	for k := -2; k <= 2; k++ {
		jx, jy := mx+k, my+4
		if zn.In(jx, jy) {
			zn.Set(jx, jy, TSeat)
			juror := g.Spawn("human", zn.Level, sr)
			if juror != nil {
				juror.Name = "Citizen Juror"
				juror.Role = "se_juror"
				juror.Faction = "citizens"
				juror.Desc = "A grim townsfolk weighing the evidence and whispering among peers."
				juror.X, juror.Y = jx, jy
				zn.Actors = append(zn.Actors, juror)
			}
		}
	}
}

func seSpawnProcession(g *Game, zn *Zone, sr *Rng) {
	// Grand procession along high street y=12
	procKind := Pick(sr, []string{"wedding", "funeral", "triumph"})

	startX := 15
	switch procKind {
	case "wedding":
		roles := []struct {
			name, role, desc string
			glyph            rune
			f                byte
		}{
			{"Royal Herald", "se_herald", "Blowing a silver trumpet adorned with nuptial ribbons.", '@', CYellow},
			{"Noble Groom", "se_groom", "Adorned in embroidered velvet and jewels.", '@', CWhite},
			{"Veiled Bride", "se_bride", "Shimmering in gossamer silk and carrying a bouquet of white mountain lilies.", '@', CWhite},
			{"Flower Maiden", "se_flower_girl", "Strewing fresh rose and jasmine petals over the cobblestones.", '@', CMagenta},
			{"Wedding Minstrel", "se_minstrel", "Strumming a gilded lute and singing merry court romances.", '@', CCyan},
		}
		for i, rdef := range roles {
			px := startX + i*3
			a := g.Spawn("human", zn.Level, sr)
			if a != nil {
				a.Name = rdef.name
				a.Role = rdef.role
				a.Desc = rdef.desc
				a.Glyph = rdef.glyph
				a.F = rdef.f
				a.X, a.Y = px, 12
				zn.Actors = append(zn.Actors, a)
			}
		}

	case "funeral":
		roles := []struct {
			name, role, desc string
			glyph            rune
			f                byte
		}{
			{"Solemn Friar", "se_friar", "Swinging a silver thurible of fragrant Frankincense.", '@', CDarkGray},
			{"Pallbearer Guildsman", "se_pallbearer", "Carrying the heavy black-draped bier of a departed lord.", '@', CBlack},
			{"Veiled Mourner", "se_mourner", "Draped in black mourning weeds, weeping softly.", '@', CDarkGray},
			{"Bellringer", "se_bellringer", "Tolling a heavy bronze passing bell in slow cadence.", '@', CBrown},
		}
		for i, rdef := range roles {
			px := startX + i*4
			a := g.Spawn("human", zn.Level, sr)
			if a != nil {
				a.Name = rdef.name
				a.Role = rdef.role
				a.Desc = rdef.desc
				a.Glyph = rdef.glyph
				a.F = rdef.f
				a.X, a.Y = px, 12
				zn.Actors = append(zn.Actors, a)
			}
		}

	case "triumph":
		roles := []struct {
			name, role, desc string
			glyph            rune
			f                byte
		}{
			{"Legion Standard-Bearer", "se_standard_bearer", "Holding high an eagle standard gilded with battle honors.", '@', CYellow},
			{"Triumphant Centurion", "se_centurion", "Battle-scarred veteran crowned with laurel leaves.", '@', CRed},
			{"War Drummer", "se_drummer", "Beating a deep resonant march on a bronze kettledrum.", '@', COrange},
			{"Loot Wagon Driver", "se_spoils", "Leading a wagon piled with captured enemy banners and spoils of war.", '@', CBrown},
		}
		for i, rdef := range roles {
			px := startX + i*4
			a := g.Spawn("human", zn.Level+1, sr)
			if a != nil {
				a.Name = rdef.name
				a.Role = rdef.role
				a.Desc = rdef.desc
				a.Glyph = rdef.glyph
				a.F = rdef.f
				a.X, a.Y = px, 12
				zn.Actors = append(zn.Actors, a)
			}
		}
	}
}

func seSpawnGuildStrikeOrRiot(g *Game, zn *Zone, sr *Rng) {
	// Plaza picket or granary crowd (around x=48, y=10)
	px, py := 48, 10
	if !zn.In(px, py) {
		return
	}

	isRiot := sr.Chance(0.5)
	if isRiot {
		// Bread-line Riot
		zn.Notes[py*MW+px] = "Protestors gather outside the civic granary demanding bread rations!"
		for k := 0; k < 5; k++ {
			a := g.Spawn("human", zn.Level, sr)
			if a != nil {
				a.Name = Pick(sr, []string{"Hungry Townsman", "Desperate Weaver", "Angry Laborer"})
				a.Role = "se_rioter"
				a.Faction = "citizens"
				a.Desc = "Shouting for lowered bread prices and waving an empty basket."
				a.X, a.Y = px+k%3, py+k/3
				zn.Actors = append(zn.Actors, a)
			}
		}
	} else {
		// Labor Guild Strike
		zn.Notes[py*MW+px] = "Stonemasons Guild picket line demanding fair wages and safety!"
		for k := 0; k < 4; k++ {
			a := g.Spawn("human", zn.Level, sr)
			if a != nil {
				a.Name = Pick(sr, []string{"Striking Stonemason", "Picket Captain", "Apprentice Striker"})
				a.Role = "se_striker"
				a.Faction = "citizens"
				a.Desc = "Refusing to pick up the chisel until guild demands are met. Holds a painted sign."
				a.X, a.Y = px+k%2, py+k/2
				zn.Actors = append(zn.Actors, a)
			}
		}
	}
}

func seSpawnTheatreTroupe(g *Game, zn *Zone, sr *Rng) {
	// Theatre stage in town plaza (x=38, y=14)
	tx, ty := 38, 14
	if !zn.In(tx, ty) {
		return
	}
	zn.Set(tx, ty, TScaffold)
	zn.Set(tx+1, ty, TScaffold)
	zn.Set(tx+2, ty, TScaffold)
	zn.Notes[ty*MW+tx] = "The Gilded Wagon Theatre Troupe stage. Satiric farces and heroic ballads daily!"

	actorRoles := []struct {
		name, role, desc string
		f                byte
	}{
		{"Master Playwright Jean", "se_troupe_leader", "Dressed in peacock feathers, orchestrating the dramatic performance.", CMagenta},
		{"The Great Harlequin", "se_harlequin", "Performing acrobatic leaps in a checkered motley suit.", CYellow},
		{"The Tragic Queen", "se_actress", "Dramatically reciting soliloquies of star-crossed love and dynastic ruin.", CWhite},
	}
	for i, rdef := range actorRoles {
		a := g.Spawn("human", zn.Level, sr)
		if a != nil {
			a.Name = rdef.name
			a.Role = rdef.role
			a.Desc = rdef.desc
			a.F = rdef.f
			a.X, a.Y = tx+i, ty
			zn.Actors = append(zn.Actors, a)
		}
	}
}

func seSpawnTradeFairOrLivestock(g *Game, zn *Zone, sr *Rng) {
	// Harvest Fair & Livestock drive on east common (x=60, y=10)
	fx, fy := 60, 10
	if !zn.In(fx, fy) {
		return
	}

	// Sheep / Cattle with sheepdog and drover
	drover := g.Spawn("human", zn.Level, sr)
	if drover != nil {
		drover.Name = "Highland Herd Drover"
		drover.Role = "se_drover"
		drover.Faction = "citizens"
		drover.Desc = "Armed with a long ash crook and a woollen cloak, driving seasonal livestock."
		drover.X, drover.Y = fx, fy
		zn.Actors = append(zn.Actors, drover)
	}

	dog := g.Spawn("dog", zn.Level, sr)
	if dog != nil {
		dog.Name = "Loyal Sheepdog"
		dog.Role = "se_sheepdog"
		dog.Faction = "citizens"
		dog.Desc = "Ears perked, circling the herd and keeping stragglers in line."
		dog.X, dog.Y = fx+1, fy
		zn.Actors = append(zn.Actors, dog)
	}

	// Flock of livestock
	for k := 0; k < 3; k++ {
		sheep := g.Spawn("goat", 1, sr)
		if sheep != nil {
			sheep.Name = "Highland Wool-Sheep"
			sheep.Faction = "citizens"
			sheep.X, sheep.Y = fx-1+k, fy+1
			zn.Actors = append(zn.Actors, sheep)
		}
	}
}

func seSpawnTavernDebate(g *Game, zn *Zone, sr *Rng) {
	// Inside the tavern (x=24, y=6)
	dx, dy := 24, 6
	if !zn.In(dx, dy) || !zn.At(dx, dy).D().Walk {
		return
	}

	debaterA := g.Spawn("human", zn.Level, sr)
	if debaterA != nil {
		debaterA.Name = "Scholar Theophilus"
		debaterA.Role = "se_debater_a"
		debaterA.Faction = "citizens"
		debaterA.Desc = "Pounding the tavern table, arguing that destiny is predetermined by celestial spheres."
		debaterA.X, debaterA.Y = dx, dy
		zn.Actors = append(zn.Actors, debaterA)
	}

	debaterB := g.Spawn("human", zn.Level, sr)
	if debaterB != nil {
		debaterB.Name = "Philosopher Kassia"
		debaterB.Role = "se_debater_b"
		debaterB.Faction = "citizens"
		debaterB.Desc = "Slamming her tankard down, arguing with fiery passion that mortal will shapes all fate."
		debaterB.X, debaterB.Y = dx+1, dy
		zn.Actors = append(zn.Actors, debaterB)
	}
}

func seTalkHook(g *Game, a *Actor) []Option {
	var opts []Option
	if g == nil || a == nil || g.Cur == nil {
		return opts
	}

	st := g.se()
	townKey := fmt.Sprintf("%d,%d", g.Cur.WX, g.Cur.WY)

	switch a.Role {
	case "se_investigator":
		opts = append(opts, Option{
			Text: "{{C|Examine the crime scene clues and assist the investigation.}}",
			Do: func() {
				if st.SolvedCrimes[townKey] {
					g.talkMenu("\"Thanks to your keen eye, the culprit is already behind iron bars!\"")
					return
				}
				g.gainXP(45)
				st.SolvedCrimes[townKey] = true
				g.give(&Item{ID: g.newID(), Kind: "currency", Name: "investigator reward pouch", Value: 60, Weight: 1, Glyph: '$', F: CYellow})
				g.talkMenu("\"Brilliant deduction! The footprint mud came from the tannery wharf. The watch will seize them at once. Here is your bounty!\"")
			},
		})

	case "se_magistrate":
		opts = append(opts, Option{
			Text: "{{Y|Testify as a character witness for Aldo the Accused.}}",
			Do: func() {
				st.TrialsAttended++
				g.gainXP(40)
				g.talkMenu("\"Upon the sworn testimony of this respected Wanderer, the tribunal finds reasonable doubt! The charges are dismissed!\"")
				for _, act := range g.Cur.Actors {
					if act.Role == "se_defendant" {
						act.Name = "Aldo (Exonerated)"
						act.Desc = "Freed by your decisive testimony; bows deeply in gratitude."
					}
				}
			},
		})

	case "se_striker":
		opts = append(opts, Option{
			Text: "{{G|Contribute 25 gold to the guild strike relief fund.}}",
			Do: func() {
				if !g.cuSpend(25) {
					g.talkMenu("\"Even a few coppers help keep our families fed during the walkout.\"")
					return
				}
				st.StrikesSettled[townKey] = true
				g.gainXP(35)
				g.fmFame(g.fmRealm(), 3)
				g.talkMenu("\"Bless you, Wanderer! With this support, the guild masters will be forced to negotiate fair wages!\"")
			},
		})

	case "se_rioter":
		opts = append(opts, Option{
			Text: "{{G|Donate 5 sacks of grain and calm the rioting crowd.}}",
			Do: func() {
				st.RiotsCalmed[townKey] = true
				g.gainXP(40)
				g.fmFame(g.fmRealm(), 4)
				g.talkMenu("\"Bread! There is bread! The Wanderer has provided for the children! The shouting turns to joyful cheers.\"")
			},
		})

	case "se_troupe_leader":
		opts = append(opts, Option{
			Text: "{{M|Listen to the Troupe perform an Epic Ballad of your deeds (10 gold).}}",
			Do: func() {
				if !g.cuSpend(10) {
					g.talkMenu("\"Ten coppers to hear the grand recounting of legends!\"")
					return
				}
				st.PlaysSeen++
				g.gainXP(30)
				g.fmFame(g.fmRealm(), 5)
				ballad := fmt.Sprintf("\"Hark to the song of %s! Slaying beasts, wandering across rifts, fearless under tempest skies!\" The crowd erupts in thunderous applause!", g.P.Name)
				g.showPopup("Epic Ballad of the Wanderer", ballad)
			},
		})

	case "se_debater_a", "se_debater_b":
		opts = append(opts, Option{
			Text: "{{C|Intervene with a decisive philosophical argument!}}",
			Do: func() {
				st.DebatesWon++
				g.gainXP(35)
				g.talkMenu("\"By the gods... a masterstroke of synthesis! Neither pure fate nor pure chance, but will carving destiny! The scholars raise their flagons in salute!\"")
			},
		})
	}

	return opts
}
