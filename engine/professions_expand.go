package engine

import (
	"fmt"
	"strings"
)

// Expanded guild professions, occupational barks, field-craft maneuvers and craft lore
// commentary (prefix pe). State lives in g.ext("pe"), planet-local ("pe" in unLocalKeys).
// Purely additive: town guild artisans are placed via zoneHooks with side rngs
// (`seed:pe:<zoneKey>`) and never mutate existing actors or draw from g.R.

type peState struct {
	Met       map[string]int  `json:"m,omitempty"`  // profession id -> conversations held
	Maneuvers map[string]bool `json:"mv,omitempty"` // field-craft maneuver id -> learned
	Lore      map[string]bool `json:"lo,omitempty"` // profession id -> guild lore recorded
	Drills    map[string]int  `json:"dr,omitempty"` // profession id -> day+1 of last drill
	sel       int
}

func (g *Game) pe() *peState {
	st := g.ext("pe", func() any { return &peState{} }).(*peState)
	if st.Met == nil {
		st.Met = map[string]int{}
	}
	if st.Maneuvers == nil {
		st.Maneuvers = map[string]bool{}
	}
	if st.Lore == nil {
		st.Lore = map[string]bool{}
	}
	if st.Drills == nil {
		st.Drills = map[string]int{}
	}
	return st
}

func init() {
	unLocalKeys = append(unLocalKeys, "pe")
	zoneHooks = append(zoneHooks, peZone)
	talkHooks = append(talkHooks, peTalk)
	modeKeys["pe"] = peKeys
	modeDraws["pe"] = func(g *Game) { g.peDraw() }
	paEntries = append(paEntries, paEntry{
		Name:     "Guilds and crafts",
		Keywords: "guilds crafts professions artisans field-craft maneuvers training lore",
		Cat:      "Screen",
		Label:    "screen",
		Func:     func(g *Game) { g.peOpen() },
	})
}

// ---- profession catalogue ----

type peProf struct {
	ID           string
	Title        string
	Guild        string
	Skill        string
	Districts    []string
	Desc         string
	Barks        []string
	LoreTitle    string
	LoreText     string
	ManeuverID   string
	ManeuverName string
	ManeuverDesc string
	Cost         int
}

var peProfs = []peProf{
	{
		ID:        "cartographer",
		Title:     "cartographer",
		Guild:     "Surveyors' Guild",
		Skill:     "lore",
		Districts: []string{"academy", "market", "harbor"},
		Desc:      "A guild cartographer with ink-stained thumbs and a brass divider tucked into their belt.",
		Barks: []string{
			"\"Every road bends around a hill the old map forgot to draw.\"",
			"\"Count the paces, sight the ridge, and never trust a tavern sketch.\"",
			"\"Rivers move slower than borders, yet both make liars of old charts.\"",
		},
		LoreTitle:    "Bearing and Scale",
		LoreText:     "Guild surveyors pace every league with a knotted cord and check their compass against noon shadow. A map without a date is only a rumour on vellum.",
		ManeuverID:   "pe_contour",
		ManeuverName: "Contour Reading",
		ManeuverDesc: "Read ridgelines, dry streambeds and sightlines at a glance to keep your bearings.",
		Cost:         45,
	},
	{
		ID:        "glassblower",
		Title:     "glassblower",
		Guild:     "Furnace-Makers' Guild",
		Skill:     "tinkering",
		Districts: []string{"artisans", "forge", "market"},
		Desc:      "A guild glassblower wearing scorched leather cuffs and carrying a long iron blowpipe.",
		Barks: []string{
			"\"Cool it too fast and the cleanest flask snaps in your pocket.\"",
			"\"White sand, kelp ash, and a steady breath: that is all a window asks.\"",
			"\"Mind the annealing kiln; the glass is still singing inside.\"",
		},
		LoreTitle:    "The Annealing Hearth",
		LoreText:     "Glass cooled in a draught shatters at a tap. Master blowers keep a second low hearth where flasks and lenses rest overnight until the inner strain settles.",
		ManeuverID:   "pe_anneal",
		ManeuverName: "Annealed Temper",
		ManeuverDesc: "Spot heat stress and hairline flaws in flasks, lenses and tempered gear.",
		Cost:         50,
	},
	{
		ID:        "clockmaker",
		Title:     "clockmaker",
		Guild:     "Horologists' Bench",
		Skill:     "tinkering",
		Districts: []string{"artisans", "academy", "market"},
		Desc:      "A guild clockmaker with a brass loupe on a cord and a pouch of tiny files.",
		Barks: []string{
			"\"One burr on a tooth and the whole tower loses ten minutes before noon.\"",
			"\"Oil too heavy slows the wheel; oil too thin leaves dust in the pivot.\"",
			"\"Hold your breath when you seat a hairspring, or you will hunt it till dusk.\"",
		},
		LoreTitle:    "The Escapement Beat",
		LoreText:     "Every spring wants to unwind in one violent rush. The escapement is the small stubborn catch that parcels force out one tick at a time.",
		ManeuverID:   "pe_escapement",
		ManeuverName: "Escapement Timing",
		ManeuverDesc: "Steady your fingers on fine catches, locks and tensioned springs.",
		Cost:         55,
	},
	{
		ID:        "lanternwright",
		Title:     "lantern-wright",
		Guild:     "Lamplitters' Union",
		Skill:     "survival",
		Districts: []string{"artisans", "gate", "harbor"},
		Desc:      "A lantern-wright carrying scraped horn panes, tin shears and coils of braided wick.",
		Barks: []string{
			"\"Horn bends where glass breaks; a watchman learns that in one rainstorm.\"",
			"\"Trim the wick flat, never pointed, if you want light instead of soot.\"",
			"\"A good shutter hides your flame faster than you can blow it out.\"",
		},
		LoreTitle:    "Horn Panes and Draughts",
		LoreText:     "A field lantern needs air from below and a baffled cap above, or the first gust off a pass leaves you in the dark. Boiled horn turns milky with age but survives a fall.",
		ManeuverID:   "pe_shutter",
		ManeuverName: "Shutter Hood",
		ManeuverDesc: "Shield a flame from wind and read night trails without ruining your dark sight.",
		Cost:         40,
	},
	{
		ID:        "herbalist",
		Title:     "herbalist-midwife",
		Guild:     "Apothecaries' Circle",
		Skill:     "endurance",
		Districts: []string{"temple", "market", "slums"},
		Desc:      "A herbalist-midwife with linen bundles of dried yarrow, willow bark and clean boiled bandages.",
		Barks: []string{
			"\"Boil the water first, then the linen, and keep your hands out of the dirt.\"",
			"\"Willow eases the ache; yarrow slows the bleed. Know which one you need.\"",
			"\"Fever takes the proud ones who refuse to rest when they still can.\"",
		},
		LoreTitle:    "Clean Cloth and Bitter Bark",
		LoreText:     "Most road wounds sour not from the blade but from dirty wool stuffed into them. Guild midwives keep linen wrapped in waxed paper and steep bitter bark before camp.",
		ManeuverID:   "pe_poultice",
		ManeuverName: "Poultice Folding",
		ManeuverDesc: "Bind cuts cleanly and brew field infusions that stretch your stamina on long marches.",
		Cost:         45,
	},
	{
		ID:        "bonecarver",
		Title:     "bone-carver",
		Guild:     "Ivory and Horn Guild",
		Skill:     "shortblade",
		Districts: []string{"artisans", "slums", "gate"},
		Desc:      "A bone-carver dusted with fine white shavings, turning a slim graving knife in their palm.",
		Barks: []string{
			"\"Carve with the grain and the bone yields; fight it and the blade slips.\"",
			"\"Boiled shin-bone makes finer pins and dice than half the soft metals in town.\"",
			"\"A short blade is honest: it goes where your wrist sends it, nowhere else.\"",
		},
		LoreTitle:    "Reading the Grain",
		LoreText:     "Antler, horn and marrow-bone each split along a hidden line. Carvers test a blank with a light score first so the final cut lands clean and true.",
		ManeuverID:   "pe_scoring",
		ManeuverName: "Grain Scoring",
		ManeuverDesc: "Find natural weak lines and place quick, controlled short-blade cuts.",
		Cost:         45,
	},
	{
		ID:        "bellfounder",
		Title:     "bell-founder",
		Guild:     "Founders' Yard",
		Skill:     "endurance",
		Districts: []string{"forge", "temple", "artisans"},
		Desc:      "A bell-founder with soot in their eyebrows and calluses thick as saddle leather.",
		Barks: []string{
			"\"Four parts copper to one of tin; skimp on tin and the bell dies flat.\"",
			"\"If the mould holds a drop of damp, the bronze spits back at the pour.\"",
			"\"Strike the rim and listen for the second hum; a crack always betrays itself.\"",
		},
		LoreTitle:    "The Sounding Rim",
		LoreText:     "Founders bury their clay moulds in packed earth pits before pouring molten bell-metal. A good casting rings true for three hundred winters; a hurried one cracks in the first frost.",
		ManeuverID:   "pe_pitch",
		ManeuverName: "Pitch Listening",
		ManeuverDesc: "Test metal, stone and armour by ear for hidden cracks and brace against heavy shock.",
		Cost:         50,
	},
	{
		ID:        "saltboiler",
		Title:     "salt-boiler",
		Guild:     "Brine-Pans Guild",
		Skill:     "survival",
		Districts: []string{"harbor", "market", "slums"},
		Desc:      "A salt-boiler whose boots and apron are crusted white from the evaporating pans.",
		Barks: []string{
			"\"Grain feeds a winter, salt keeps the meat from walking off before snow.\"",
			"\"Test the brine with an egg: when it floats, the pan is ready to draw.\"",
			"\"Every caravan buys salt before it buys wine. Remember that on the road.\"",
		},
		LoreTitle:    "Drawing the Brine",
		LoreText:     "Salt-boilers feed shallow iron pans day and night, raking white crystals as the steam rises. Coarse salt cures hides and fish; fine salt buys passage through any toll gate.",
		ManeuverID:   "pe_brine",
		ManeuverName: "Brine Curing",
		ManeuverDesc: "Pack and cure trail rations so food lasts longer through wet weather.",
		Cost:         40,
	},
	{
		ID:        "inkmaker",
		Title:     "ink-maker",
		Guild:     "Stationers' Hall",
		Skill:     "lore",
		Districts: []string{"academy", "artisans", "temple"},
		Desc:      "An ink-maker with black-stained knuckles and glass stoppers rattling in their coat.",
		Barks: []string{
			"\"Oak galls, green vitriol and gum: it dries pale, then turns black as pitch.\"",
			"\"Lampblack washes off in rain; iron-gall bites into the parchment for good.\"",
			"\"Show me a charter's ink and I will tell you if the seal is ten days old or ten years.\"",
		},
		LoreTitle:    "Iron-Gall and Lampblack",
		LoreText:     "Clerks favour iron-gall ink because it darkens as it ages and cannot be scraped clean without scarring the leaf. Forgers often give themselves away by the sheen of fresh soot.",
		ManeuverID:   "pe_gall",
		ManeuverName: "Gall Testing",
		ManeuverDesc: "Spot altered ledgers, fresh ink and faded inscriptions in old records.",
		Cost:         45,
	},
	{
		ID:        "astrologist",
		Title:     "astrologist",
		Guild:     "Almanac Watch",
		Skill:     "lore",
		Districts: []string{"academy", "temple", "gate"},
		Desc:      "An astrologist carrying a wooden cross-staff and a folded almanac covered in neat tables.",
		Barks: []string{
			"\"The stars do not care who wears the crown, but they keep honest time.\"",
			"\"Sight the pole star twice before midnight if you want a straight march.\"",
			"\"Tides follow the moon, and captains who forget that sleep on the shoals.\"",
		},
		LoreTitle:    "The Cross-Staff Almanac",
		LoreText:     "Town astrologists earn their bread charting eclipses, planting weeks and high tides. Travellers borrow their tables to hold a course across featureless flats after dark.",
		ManeuverID:   "pe_zenith",
		ManeuverName: "Zenith Sighting",
		ManeuverDesc: "Judge the night hour and keep a true compass bearing under open sky.",
		Cost:         50,
	},
	{
		ID:        "chandler",
		Title:     "chandler",
		Guild:     "Tallow and Wax Guild",
		Skill:     "persuasion",
		Districts: []string{"market", "harbor", "temple"},
		Desc:      "A chandler smelling faintly of beeswax, pine resin and rendered tallow.",
		Barks: []string{
			"\"Beeswax for the altar, mutton tallow for the kitchen, and pitch for the hull.\"",
			"\"A crooked dip burns crooked and drips half its weight on the table.\"",
			"\"Good light sells honest cloth; dim light sells yesterday's.\"",
		},
		LoreTitle:    "Weight and Pure Wax",
		LoreText:     "Chandlers weigh every cake twice because beeswax fetches four times the price of tallow. A thumbnail press tells a trained buyer whether a cake has been stretched with fat.",
		ManeuverID:   "pe_tallow",
		ManeuverName: "Tallow Grading",
		ManeuverDesc: "Judge wax, oil and trade goods accurately so merchants give you fairer terms.",
		Cost:         45,
	},
	{
		ID:        "ropemaker",
		Title:     "rope-maker",
		Guild:     "Cordwainers' Walk",
		Skill:     "acrobatics",
		Districts: []string{"harbor", "gate", "artisans"},
		Desc:      "A rope-maker with hemp strands tucked behind their ear and tarred twine at their hip.",
		Barks: []string{
			"\"Spin the yarns right, lay the strands left, and the rope holds itself together.\"",
			"\"A knot that jams under load is worse than a knot that slips.\"",
			"\"Tar keeps sea-rot out of a hawser, though it ruins every shirt you own.\"",
		},
		LoreTitle:    "The Long Rope-Walk",
		LoreText:     "Ropemakers walk backwards down a hundred-pace lane twisting hemp into balanced strands. A well-laid line takes a sudden jolt without kinking or snapping.",
		ManeuverID:   "pe_belay",
		ManeuverName: "Belay Hitch",
		ManeuverDesc: "Tie quick-release climbing and hauling hitches that keep your footing sure.",
		Cost:         45,
	},
}

func peProfByID(id string) *peProf {
	for i := range peProfs {
		if peProfs[i].ID == id {
			return &peProfs[i]
		}
	}
	return nil
}

func peProfForActor(a *Actor) *peProf {
	if a == nil || !strings.HasPrefix(a.Role, "pe_") {
		return nil
	}
	return peProfByID(strings.TrimPrefix(a.Role, "pe_"))
}

// ---- additive zone placement (side rngs only) ----

func peZone(g *Game, zn *Zone, _ *Rng) {
	if g.W == nil || zn == nil || zn.Kind != "town" || zn.Z != 0 || zn.Burg < 0 {
		return
	}
	bd := g.W.burgByID[zn.Burg]
	if bd == nil {
		return
	}
	r := NewRng(g.Seed + ":pe:" + zn.Key)
	cp := g.cityPlanFor(bd)
	dist := ""
	if cp != nil {
		dist = cp.districts[[2]int{zn.ZX, zn.ZY}]
	}

	var cands []*peProf
	for i := range peProfs {
		p := &peProfs[i]
		if dist == "" {
			cands = append(cands, p)
			continue
		}
		for _, d := range p.Districts {
			if d == dist {
				cands = append(cands, p)
				break
			}
		}
	}
	if len(cands) == 0 {
		for i := range peProfs {
			cands = append(cands, &peProfs[i])
		}
	}

	count := 1
	if g.W.cityTier[bd.ID] >= 2 && len(cands) >= 2 && r.Chance(0.65) {
		count = 2
	}
	used := map[string]bool{}
	for i := 0; i < count; i++ {
		pick := cands[r.Int(0, len(cands)-1)]
		if used[pick.ID] {
			continue
		}
		used[pick.ID] = true
		x, y, ok := zn.FreeSpot(r, func(tx, ty int) bool {
			t := zn.At(tx, ty)
			return t == TCobble || t == TFloor
		})
		if !ok {
			continue
		}
		a := g.makeNPC("pe_"+pick.ID, bd, g.W.cultByID[bd.Culture], r)
		a.Title = pick.Title
		a.Desc = pick.Desc
		a.Teach = pick.Skill
		a.AI = "still"
		a.X, a.Y, a.HomeX, a.HomeY = x, y, x, y
		a.Gold = r.Int(60, 180) + a.Level*15
		a.Set("pe")
		zn.Actors = append(zn.Actors, a)
	}
}

// ---- dialogue hooks ----

func peDiscount(g *Game, base int) int {
	return maxi(10, base*(100-g.Skills["persuasion"]*8)/100)
}

func peTalk(g *Game, a *Actor) []Option {
	if a == nil || a.HP <= 0 || a.Angry || a.AI == "follower" {
		return nil
	}
	if p := peProfForActor(a); p != nil {
		return g.peArtisanOpts(a, p)
	}
	// Light craft-lore commentary for existing guild trainers/artisans without mutating them.
	if a.Teach != "" || a.Role == "smith" || a.Role == "merchant" || a.Role == "scholar" {
		p := peProfForExisting(g, a)
		if p != nil {
			return []Option{{
				Text: fmt.Sprintf("{{c|Ask about %s craft notes.}}", p.Guild),
				Do:   func() { g.peDiscussCraft(a, p) },
			}}
		}
	}
	return nil
}

func peProfForExisting(g *Game, a *Actor) *peProf {
	r := NewRng(fmt.Sprintf("%s:pe:ex:%d", g.Seed, a.ID))
	var matches []*peProf
	for i := range peProfs {
		if a.Teach != "" && strings.Contains(a.Teach, peProfs[i].Skill) {
			matches = append(matches, &peProfs[i])
		}
	}
	if len(matches) > 0 {
		return matches[r.Int(0, len(matches)-1)]
	}
	return &peProfs[r.Int(0, len(peProfs)-1)]
}

func (g *Game) peArtisanOpts(a *Actor, p *peProf) []Option {
	st := g.pe()
	var opts []Option
	opts = append(opts, Option{
		Text: fmt.Sprintf("{{c|Ask about their trade (%s).}}", p.Title),
		Do:   func() { g.peDiscussCraft(a, p) },
	})
	cost := peDiscount(g, p.Cost)
	if !st.Maneuvers[p.ManeuverID] {
		opts = append(opts, Option{
			Text: fmt.Sprintf("{{O|Learn field-craft: %s (%d %s).}}", p.ManeuverName, cost, g.Sk.Currency),
			Do:   func() { g.peLearnManeuver(a, p) },
		})
	} else {
		day := g.Turn/1440 + 1
		if st.Drills[p.ID] < day {
			drillCost := maxi(10, cost/2)
			opts = append(opts, Option{
				Text: fmt.Sprintf("{{y|Run a %s guild drill (%d %s).}}", skillName(p.Skill), drillCost, g.Sk.Currency),
				Do:   func() { g.peRunDrill(a, p) },
			})
		}
	}
	opts = append(opts, Option{
		Text: "{{K|Open guilds and crafts ledger.}}",
		Do:   func() { g.peOpen() },
	})
	return opts
}

func (g *Game) peDiscussCraft(a *Actor, p *peProf) {
	st := g.pe()
	idx := st.Met[p.ID]
	st.Met[p.ID] = idx + 1
	bark := p.Barks[idx%len(p.Barks)]
	firstLore := !st.Lore[p.ID]
	st.Lore[p.ID] = true
	if firstLore {
		g.gainXP(6)
		g.Msg(fmt.Sprintf("{{C|Craft note recorded: %s (%s).}}", p.LoreTitle, p.Guild))
	}
	text := fmt.Sprintf("%s\n\n%s (%s): %s", bark, p.LoreTitle, p.Guild, p.LoreText)
	g.talkMenu(text)
}

func (g *Game) peLearnManeuver(a *Actor, p *peProf) {
	st := g.pe()
	cost := peDiscount(g, p.Cost)
	if g.P.Gold < cost {
		g.talkMenu(fmt.Sprintf("\"A guild lesson takes tools and time. Bring %d %s and we will start.\"", cost, g.Sk.Currency))
		return
	}
	g.P.Gold -= cost
	a.Gold += cost
	g.Turn += 120
	st.Maneuvers[p.ManeuverID] = true
	st.Lore[p.ID] = true
	g.gainXP(14)
	extra := ""
	if g.Skills != nil && g.Skills[p.Skill] < 1 {
		g.Skills[p.Skill] = 1
		extra = fmt.Sprintf(" Your %s rises to rank 1.", skillName(p.Skill))
	}
	g.Msg(fmt.Sprintf("{{G|Field-craft learned: %s (%s).%s}}", p.ManeuverName, p.Guild, extra))
	g.talkMenu(fmt.Sprintf("\"Watch the hands once more, then try it yourself.\"\n\nYou learn %s: %s%s", p.ManeuverName, p.ManeuverDesc, extra))
}

func (g *Game) peRunDrill(a *Actor, p *peProf) {
	st := g.pe()
	day := g.Turn/1440 + 1
	if st.Drills[p.ID] >= day {
		g.talkMenu("\"Enough bench work for today. Rest your hands and come back tomorrow.\"")
		return
	}
	cost := maxi(10, peDiscount(g, p.Cost)/2)
	if g.P.Gold < cost {
		g.talkMenu(fmt.Sprintf("\"Bench stock is not free. Come back with %d %s.\"", cost, g.Sk.Currency))
		return
	}
	g.P.Gold -= cost
	a.Gold += cost
	g.Turn += 90
	st.Drills[p.ID] = day
	g.gainXP(10 + g.Skills[p.Skill]*3)
	g.Msg(fmt.Sprintf("{{G|You complete a %s drill with the %s.}}", skillName(p.Skill), p.Title))
	g.talkMenu(fmt.Sprintf("\"Cleaner work than yesterday. Keep %s sharp on the road.\"", p.ManeuverName))
}

// ---- Guilds and crafts screen ----

func (g *Game) peOpen() {
	g.pe()
	g.Talk = nil
	g.Mode = "pe"
}

func peKeys(g *Game, k string) {
	st := g.pe()
	switch k {
	case "Escape", "q":
		g.Mode = "main"
	case "ArrowUp", "N8", "k":
		st.sel = maxi(0, st.sel-1)
	case "ArrowDown", "N2", "j":
		st.sel = mini(len(peProfs)-1, st.sel+1)
	}
}

func (g *Game) peDraw() {
	st := g.pe()
	sc := g.sc
	x, y, w, h := g.panel("Guilds and crafts")
	bottom := y + h - 1

	learnedN := len(st.Maneuvers)
	loreN := len(st.Lore)
	sc.TextB(x, y, fmt.Sprintf("{{W|Guild trades: %d}}  {{G|Field-craft learned: %d/%d}}  {{C|Lore notes: %d/%d}}",
		len(peProfs), learnedN, len(peProfs), loreN, len(peProfs)), CGray, CBgDark, x+w)
	y += 2

	st.sel = clampi(st.sel, 0, len(peProfs)-1)
	listW := 34
	if listW > w/2 {
		listW = w / 2
	}
	for i, p := range peProfs {
		if y+i >= bottom-1 {
			break
		}
		mark := " "
		if st.Maneuvers[p.ManeuverID] {
			mark = "*"
		} else if st.Lore[p.ID] {
			mark = "+"
		}
		prefix := "  "
		color := CGray
		if i == st.sel {
			prefix = "> "
			color = CYellow
		}
		line := fmt.Sprintf("%s[%s] %s", prefix, mark, titleCase(p.Title))
		sc.TextB(x, y+i, line, color, CBgDark, x+listW)
	}

	p := peProfs[st.sel]
	dx := x + listW + 2
	dw := w - listW - 2
	if dw > 16 {
		dy := y
		sc.TextB(dx, dy, fmt.Sprintf("{{Y|%s}} {{K|(%s)}}", titleCase(p.Title), p.Guild), CGray, CBgDark, dx+dw)
		dy++
		sc.TextB(dx, dy, fmt.Sprintf("{{W|Skill:}} %s   {{W|Districts:}} %s", skillName(p.Skill), strings.Join(p.Districts, ", ")), CGray, CBgDark, dx+dw)
		dy += 2
		status := "Not yet learned"
		if st.Maneuvers[p.ManeuverID] {
			status = "Learned"
		}
		sc.TextB(dx, dy, fmt.Sprintf("{{O|Field-craft: %s}} {{K|[%s]}}", p.ManeuverName, status), CGray, CBgDark, dx+dw)
		dy++
		for _, ln := range Wrap(p.ManeuverDesc, dw) {
			if dy >= bottom-1 {
				break
			}
			sc.TextB(dx, dy, ln, CGray, CBgDark, dx+dw)
			dy++
		}
		dy++
		if st.Lore[p.ID] {
			if dy < bottom-1 {
				sc.TextB(dx, dy, fmt.Sprintf("{{C|%s}}", p.LoreTitle), CGray, CBgDark, dx+dw)
				dy++
			}
			for _, ln := range Wrap(p.LoreText, dw) {
				if dy >= bottom-1 {
					break
				}
				sc.TextB(dx, dy, ln, CGray, CBgDark, dx+dw)
				dy++
			}
		} else if dy < bottom-1 {
			sc.TextB(dx, dy, "{{K|Speak with a guild artisan in town to record their craft notes.}}", CGray, CBgDark, dx+dw)
		}
	}

	sc.TextB(x, bottom, "{{K|Up/Down select   [*] maneuver learned   [+] lore noted   Esc close}}", CGray, CBgDark, x+w)
}
