package engine

import (
	"fmt"
	"sort"
	"strings"
	"sync"
)

// Murim, part 1 (prefix mr): sect data, genre skins and deterministic generation. Sects are cross-border
// organisations, not countries: a remote headquarters (a walled compound in a mountain/forest parasang) and
// 2-5 branch halls in towns of different states. The game logic lives in murim.go.

const (
	mrOrthodox   = "orthodox"
	mrUnorthodox = "unorthodox"
	mrDemonic    = "demonic"
)

type MrTech struct {
	ID   string // sectID:index
	Name string
	Kind string // palm dash burst heal sweep drain fear flurry
	Need int    // cultivation realm required to use it
	Cool int
	Cost int // qi
	Pow  int
}

type MrArt struct {
	Name string
	Stat int
	Tech [3]MrTech
}

type MrHall struct {
	Burg   int
	State  int
	WX, WY int
	Town   string
}

type MrSect struct {
	ID        string
	Name      string
	Kind      string // sect clan cult guild
	Align     string
	Creed     string
	Culture   int
	WX, WY    int // headquarters parasang
	State     int // state holding the headquarters' land (0 = none)
	HQName    string
	Halls     []MrHall
	Art       MrArt
	Leader    string
	LeaderTit string
	Elders    []string
	Rivals    []string
}

func (s *MrSect) Short() string  { return strings.TrimPrefix(s.Name, "The ") }
func (s *MrSect) RepKey() string { return "mr:" + s.ID }

type mrSkin struct {
	Mount   string
	Sects   []string
	Clans   []string
	Black   []string
	Guilds  []string
	Cults   []string
	Realms  [8]string
	Ranks   [5]string
	Qi      string
	Pills   [3]string
	Manual  string
	ArtAdj  []string
	ArtForm []string
	Hall    string
	Union   string
	Lead    [3]string
	Trial   [6]string // entrance trial per favoured stat
}

var mrSkinBase = mrSkin{
	Mount:   "Mount %s",
	Sects:   []string{"Mount %s Sect", "%s Sword Pavilion", "%s Palm Temple", "%s Cloud Monastery"},
	Clans:   []string{"%s Clan", "%s Martial Clan", "House %s"},
	Black:   []string{"%s Blood Sect", "Black Lotus of %s", "%s Shadow Sect"},
	Guilds:  []string{"%s Merchant-Assassin Guild", "Beggars' Hall of %s", "%s Poison Guild", "%s Smugglers' Fellowship"},
	Cults:   []string{"Heavenly Demon Cult", "Blood Moon Cult", "Ten-Thousand Demons Cult"},
	Realms:  [8]string{"Mortal", "Third-rate", "Second-rate", "First-rate", "Peak", "Transcendent", "Profound", "Life-and-Death"},
	Ranks:   [5]string{"Outer Disciple", "Inner Disciple", "Core Disciple", "Elder", "Sect Leader"},
	Qi:      "qi",
	Pills:   [3]string{"Qi-Gathering Pill", "Marrow-Cleansing Pill", "Heaven-Dew Elixir"},
	Manual:  "secret manual",
	ArtAdj:  []string{"Azure", "Nine-Heaven", "Plum Blossom", "Thunder", "Hidden Dragon", "Crimson", "Iron", "Cloud-Treading", "Frost Lotus", "Mountain-Splitting", "Jade", "Void"},
	ArtForm: []string{"Sword Art", "Palm Art", "Fist Art", "Qi Art", "Step Art", "Spear Art"},
	Hall:    "branch hall",
	Union:   "Murim Alliance",
	Lead:    [3]string{"Sect Master", "Guildmaster", "Demon Lord"},
	Trial: [6]string{"Lift the iron urn up the hundred steps.", "Cross the plum-blossom posts without a fall.",
		"Stand unflinching beneath the waterfall for a stick of incense.", "Solve the sect's riddle of the nine paths.",
		"Sit the night in the cold hall and keep your breath even.", "Face the elders and hold their gaze."},
}

// per-genre overrides; empty fields fall back to mrSkinBase
var mrSkins = map[string]mrSkin{
	"mythic": {Mount: "Mount %s", Sects: []string{"%s Temple of the Fist", "%s Hero-School", "Mount %s Lodge"}, Qi: "pneuma",
		Realms: [8]string{"Mortal", "Initiate", "Hero-born", "Demigod-touched", "Heroic", "Titan-souled", "Olympian", "Deathless"},
		Cults:  []string{"Cult of the Devouring Gods", "Titans' Remnant"}, Pills: [3]string{"Nectar Draught", "Ambrosia Pill", "Ichor of the Gods"}},
	"steampunk": {Mount: "%s Crag", Sects: []string{"%s Pugilists' Lodge", "%s Engine-Dojo", "%s Boiler Academy"}, Clans: []string{"%s Works", "House %s Foundry"},
		Black: []string{"%s Cog-Breakers", "%s Smokestack Gang"}, Guilds: []string{"%s Brass Cartel", "%s Rag-and-Bone Union"}, Qi: "steam",
		Realms: [8]string{"Unwound", "Winding", "Geared", "Pressurised", "Master-Wrought", "Boiler-Hearted", "Perpetual", "Clockwork Transcendent"},
		Ranks:  [5]string{"Apprentice", "Journeyman", "Master", "Foreman", "Lodge Master"}, Lead: [3]string{"Lodge Master", "Guild Boss", "Engine Prophet"},
		Cults: []string{"Church of the Runaway Engine", "The Red Valve Cult"}, Pills: [3]string{"Steam Tonic", "Pressure Salts", "Quicksilver Draught"}, Manual: "blueprint folio",
		Union: "Pugilists' Concord", Hall: "lodge"},
	"dieselpunk": {Mount: "%s Ridge", Sects: []string{"%s Boxing Syndicate", "%s Iron-Fist Hall"}, Clans: []string{"%s Family", "The %s Outfit"},
		Black: []string{"%s Knuckle Gang", "%s Docks Crew"}, Guilds: []string{"%s Dockers' Union", "%s Racket"}, Qi: "drive",
		Realms: [8]string{"Rookie", "Journeyman", "Contender", "Heavyweight", "Champion", "Legend", "Iron Legend", "Untouchable"},
		Ranks:  [5]string{"Plug-Ugly", "Bruiser", "Lieutenant", "Underboss", "Don"}, Lead: [3]string{"Don", "Boss", "Black Don"},
		Cults: []string{"The Smoke Cult", "Brotherhood of the Black Engine"}, Pills: [3]string{"Pep Pill", "Smelling Salts", "Black Coffee Elixir"}, Manual: "fight ledger",
		Union: "Boxing Commission", Hall: "gym"},
	"atompunk": {Mount: "%s Bluff", Sects: []string{"%s Karate Institute", "%s Atomic Dojo"}, Clans: []string{"%s Family Dojo"}, Black: []string{"%s Isotope Gang"},
		Guilds: []string{"%s Courier Syndicate", "%s Salvage Union"}, Qi: "flux",
		Realms: [8]string{"Civilian", "Cadet", "Operative", "Vanguard", "Atomic", "Fusion", "Critical Mass", "Beyond the Bomb"},
		Ranks:  [5]string{"Trainee", "Practitioner", "Instructor", "Senior Instructor", "Director"}, Lead: [3]string{"Director", "Syndic", "Overlord"},
		Cults: []string{"The Glow Cult", "Children of the Second Sun"}, Pills: [3]string{"Rad-Tab", "Isotope Tonic", "Fission Elixir"}, Manual: "classified kata dossier",
		Union: "Dojo Federation", Hall: "dojo"},
	"cyberpunk": {Mount: "%s Heights", Sects: []string{"%s Dojo", "%s Kata Syndicate", "%s Neon Temple"}, Clans: []string{"%s Keiretsu Dojo", "%s Combine"},
		Black: []string{"%s Razor Cult", "%s Back-Alley Dojo"}, Guilds: []string{"%s Fixers' Syndicate", "%s Data-Courier Guild", "%s Street Cartel"}, Qi: "flux",
		Realms: [8]string{"Unmodded", "Street-grade", "Pro-grade", "Elite-grade", "Apex-grade", "Singular", "Ghost-grade", "Beyond the Grid"},
		Ranks:  [5]string{"Trainee", "Runner", "Core Operator", "Senior Operator", "Sensei"}, Lead: [3]string{"Sensei", "Fixer", "Null Prophet"},
		Cults: []string{"The Null Ascendancy", "Ghost Protocol Cult"}, Pills: [3]string{"Synapse Booster", "Marrow Flush", "Overclock Serum"}, Manual: "black-market datashard",
		ArtAdj:  []string{"Neon", "Chrome", "Static", "Overclocked", "Monofilament", "Kinetic", "Vector", "Razor", "Phantom", "Zero"},
		ArtForm: []string{"Kata", "Circuit-Fist", "Protocol", "Stance", "Cut"}, Union: "Dojo Compact", Hall: "dojo",
		Trial: [6]string{"Bench the hydraulic press.", "Run the rooftop gauntlet.", "Take the shock-rig for ten seconds.", "Break the daemon's puzzle-lock.",
			"Stay offline in the dark for a night.", "Out-stare the sensei's optics."}},
	"biopunk": {Mount: "%s Hive", Sects: []string{"%s Hive-Sect", "%s Marrow Order", "%s Sinew School"}, Clans: []string{"%s Strain", "%s Lineage"},
		Black: []string{"%s Rot-Hand Sect"}, Guilds: []string{"%s Splicers' Guild", "%s Flesh-Market Syndicate"}, Qi: "ichor",
		Realms: [8]string{"Baseline", "Budding", "Sinewed", "Chimeric", "Apex-Strain", "Prime", "Progenitor", "Beyond the Genome"},
		Ranks:  [5]string{"Graft", "Strain", "Chimera", "Progenitor", "Prime"}, Lead: [3]string{"Prime", "Broker", "Hive Tyrant"},
		Cults: []string{"The Writhing Covenant", "Church of the Open Wound"}, Pills: [3]string{"Growth Pellet", "Marrow Culture", "Prime Serum"}, Manual: "living codex", Hall: "nest"},
	"solarpunk": {Mount: "%s Terrace", Sects: []string{"%s Grove Sect", "%s Sun-Temple", "%s Greenhand Monastery"}, Clans: []string{"%s Orchard Clan", "House %s"},
		Black: []string{"%s Thornbound"}, Guilds: []string{"%s Seed-Smugglers", "%s Harvest Guild"}, Qi: "prana",
		Realms: [8]string{"Seedling", "Sprout", "Sapling", "Grown", "Ancient", "Heartwood", "Evergreen", "Everbloom"},
		Cults:  []string{"The Blight Choir", "Cult of the Devouring Bloom"}, Pills: [3]string{"Sunpetal Tea", "Root-Marrow Brew", "Heartwood Nectar"}, Manual: "pressed-leaf manual", Hall: "grove hall"},
	"wasteland": {Mount: "%s Bluff", Sects: []string{"%s Fist-Pack", "%s Iron Fists", "%s Bunker Dojo"}, Clans: []string{"%s Clan", "%s Kin"},
		Black: []string{"%s Reavers", "%s Skull-Crackers"}, Guilds: []string{"%s Scrap Cartel", "%s Tunnel Brawlers", "%s Water Barons"}, Qi: "grit",
		Realms: [8]string{"Soft", "Scrapper", "Hardened", "Veteran", "Warlord-grade", "Apex Survivor", "Legend of the Waste", "Wasteland Myth"},
		Ranks:  [5]string{"Stray", "Brawler", "Hardhand", "Warden", "Warlord"}, Lead: [3]string{"Warlord", "Boss", "Rad-King"},
		Cults: []string{"The Rad-Maw Cult", "Children of the Bomb"}, Pills: [3]string{"Grit Tonic", "Rad-Bone Draught", "Warlord's Blood"}, Manual: "oil-stained notebook", Union: "Fist Pact",
		Hall: "den"},
	"twilight": {Mount: "%s Fell", Sects: []string{"Order of %s", "%s Vigil", "Fallen Order of %s"}, Clans: []string{"%s Line", "House %s of the Last Light"},
		Black: []string{"%s Hollow Hand"}, Guilds: []string{"%s Gravewalkers' Guild", "%s Candle-Thieves"}, Qi: "dusk",
		Realms: [8]string{"Hollow", "Wan", "Pale", "Gloaming", "Dusk-Touched", "Nightbound", "Eclipsed", "Beyond the Veil"},
		Ranks:  [5]string{"Mourner", "Vigilant", "Hollow Knight", "Pale Elder", "Warden of the Last Light"}, Lead: [3]string{"Warden", "Gravemaster", "Dusk Lord"},
		Cults: []string{"Choir of the Black Sun", "The Hollow Crown Cult"}, Pills: [3]string{"Ash Draught", "Gloam Pill", "Eclipse Elixir"}, Manual: "grave-codex", Hall: "vigil hall"},
}

func mrSkinFor(genre string) mrSkin {
	o, ok := mrSkins[genre]
	b := mrSkinBase
	if !ok {
		return b
	}
	if o.Mount != "" {
		b.Mount = o.Mount
	}
	if len(o.Sects) > 0 {
		b.Sects = o.Sects
	}
	if len(o.Clans) > 0 {
		b.Clans = o.Clans
	}
	if len(o.Black) > 0 {
		b.Black = o.Black
	}
	if len(o.Guilds) > 0 {
		b.Guilds = o.Guilds
	}
	if len(o.Cults) > 0 {
		b.Cults = o.Cults
	}
	if o.Realms[0] != "" {
		b.Realms = o.Realms
	}
	if o.Ranks[0] != "" {
		b.Ranks = o.Ranks
	}
	if o.Qi != "" {
		b.Qi = o.Qi
	}
	if o.Pills[0] != "" {
		b.Pills = o.Pills
	}
	if o.Manual != "" {
		b.Manual = o.Manual
	}
	if len(o.ArtAdj) > 0 {
		b.ArtAdj = o.ArtAdj
	}
	if len(o.ArtForm) > 0 {
		b.ArtForm = o.ArtForm
	}
	if o.Hall != "" {
		b.Hall = o.Hall
	}
	if o.Union != "" {
		b.Union = o.Union
	}
	if o.Lead[0] != "" {
		b.Lead = o.Lead
	}
	if o.Trial[0] != "" {
		b.Trial = o.Trial
	}
	return b
}

var mrTechWords = map[string][]string{
	"palm":   {"Thunder Palm", "Mountain-Cleaving Palm", "Iron Palm", "Shattering Fist", "Dragon-Taming Palm", "Falling Star Strike"},
	"dash":   {"Wind-Riding Step", "Lightning Lunge", "Swallow Dive", "Shadowless Step", "Piercing Rush", "Cloud-Cutting Leap"},
	"burst":  {"Finger of Heaven", "Qi Wave", "Heaven-Rending Beam", "Sword Qi", "Void Shot", "Roaring Tide Blast"},
	"heal":   {"Jade Circulation", "Heart-Mending Breath", "Golden Bell Guard", "Marrow Recovery", "Still-Water Mind", "Lotus Renewal"},
	"sweep":  {"Whirling Lotus", "Hundred-Blade Storm", "Ring of Falling Petals", "Dragon's Tail", "Cyclone Stance", "Eight-Direction Sweep"},
	"drain":  {"Blood Siphon", "Soul-Devouring Hand", "Marrow-Draining Claw", "Crimson Tide", "Heart-Eater Palm", "Vampiric Grasp"},
	"fear":   {"Lion's Roar", "Overbearing Aura", "Heaven-Shaking Shout", "Killing Intent", "Dread Presence", "Tyrant's Gaze"},
	"flurry": {"Thousand-Hand Strike", "Rain of Fists", "Twin Dragon Dance", "Seven-Star Jab", "Phantom Barrage", "Hundred-Fold Pierce"},
}

var mrStatKinds = [6][3]string{
	STR: {"palm", "sweep", "fear"},
	AGI: {"dash", "flurry", "sweep"},
	TOU: {"palm", "heal", "fear"},
	INT: {"burst", "dash", "heal"},
	WIL: {"burst", "heal", "fear"},
	EGO: {"fear", "drain", "flurry"},
}

var mrTechNeed = [3]int{1, 2, 4}
var mrTechCool = [3]int{8, 12, 20}
var mrTechCost = [3]int{8, 14, 22}

var mrCreeds = map[string][]string{
	mrOrthodox: {"Uphold righteousness, protect the weak, and keep the martial world honest.", "Discipline of body and mind; the sword serves the way, never the purse.",
		"An old school of rigorous forms. Its disciples are taught that restraint is the highest technique."},
	mrUnorthodox: {"Neither righteous nor wicked: they sell what the highest bidder needs and keep their own ledger of favours.", "Coin, secrets and knives. The rules of the others do not bind them.",
		"A network of informants and hired blades that thrives in the cracks between the great houses."},
	mrDemonic: {"Power at any price. The strong feed on the weak, and the whole world is theirs to be consumed.", "Forbidden arts that burn life for strength. The righteous call them a plague; they call it freedom.",
		"A dark creed of blood and devotion. Every orthodox sect has sworn to see it destroyed."},
}

var (
	mrMu    sync.Mutex
	mrCache = map[*WorldData][]*MrSect{}
)

// Sects returns every sect of the world (cached per world, never saved).
func (g *Game) Sects() []*MrSect {
	mrMu.Lock()
	defer mrMu.Unlock()
	if s, ok := mrCache[g.W]; ok {
		return s
	}
	s := g.mrGenSects()
	mrCache[g.W] = s
	return s
}

func (g *Game) SectByID(id string) *MrSect {
	for _, s := range g.Sects() {
		if s.ID == id {
			return s
		}
	}
	return nil
}

func mrSlug(s string) string {
	var b strings.Builder
	dash := false
	for _, c := range strings.ToLower(s) {
		if c >= 'a' && c <= 'z' || c >= '0' && c <= '9' {
			b.WriteRune(c)
			dash = false
		} else if !dash && b.Len() > 0 {
			b.WriteByte('-')
			dash = true
		}
	}
	return strings.Trim(b.String(), "-")
}

func mrCheb(w *WorldData, a, b int) int {
	return maxi(abs(a%w.W-b%w.W), abs(a/w.W-b/w.W))
}

func mrShuffle[T any](r *Rng, a []T) {
	for i := len(a) - 1; i > 0; i-- {
		j := r.Int(0, i)
		a[i], a[j] = a[j], a[i]
	}
}

func (g *Game) mrBurgs() []*BurgData {
	w := g.W
	var bs []*BurgData
	for i := range w.Burgs {
		b := &w.Burgs[i]
		if b.Alive && b.Layer == "surface" {
			bs = append(bs, b)
		}
	}
	sort.SliceStable(bs, func(i, j int) bool {
		if bs[i].Pop != bs[j].Pop {
			return bs[i].Pop > bs[j].Pop
		}
		return bs[i].ID < bs[j].ID
	})
	return bs
}

func (g *Game) mrPlaceWord(cu *CultureData, r *Rng) string {
	if cu != nil && len(cu.Places) > 0 {
		return Pick(r, cu.Places)
	}
	if cu != nil && len(cu.Given) > 0 {
		return Pick(r, cu.Given)
	}
	return Pick(r, []string{"Hua", "Song", "Wudang", "Emei", "Kunlun", "Tai", "Heng"})
}

func (g *Game) mrGenSects() []*MrSect {
	w := g.W
	if w.W == 0 || len(w.Biome) == 0 {
		return nil
	}
	sk := mrSkinFor(w.Genre)
	r := NewRng(w.Seed + ":murim:sects")
	burgs := g.mrBurgs()
	if len(burgs) < 2 {
		return nil
	}
	n := 6 + r.Int(0, 4)
	// headquarters candidates: remote land parasangs, high ground first
	type cand struct {
		p  int
		sc float64
	}
	var cands []cand
	for p := 0; p < w.W*w.H && p < len(w.Biome); p++ {
		if w.IsWater(p) || w.Burg[p] > 0 {
			continue
		}
		sc := float64(w.Height[p]) + r.Float()*6
		switch w.Biome[p] {
		case 5, 6, 7, 8, 9, 13, 14, 15, 16:
			sc += 6
		}
		cands = append(cands, cand{p, sc})
	}
	sort.SliceStable(cands, func(i, j int) bool {
		if cands[i].sc != cands[j].sc {
			return cands[i].sc > cands[j].sc
		}
		return cands[i].p < cands[j].p
	})
	if len(cands) == 0 {
		return nil
	}
	pool := cands
	if lim := maxi(n*8, len(cands)/6); lim < len(pool) {
		pool = pool[:lim]
	}
	var taken []int
	pickHQ := func(minD int, wantWild bool) int {
		for pass := 0; pass < 2; pass++ {
			idx := make([]int, len(pool))
			for i := range idx {
				idx[i] = i
			}
			mrShuffle(r, idx)
			for _, i := range idx {
				p := pool[i].p
				ok := true
				for _, q := range taken {
					if mrCheb(w, p, q) < minD {
						ok = false
						break
					}
				}
				if !ok {
					continue
				}
				if pass == 0 && wantWild {
					switch w.Biome[p] {
					case 10, 11, 13, 14, 15, 16:
					default:
						continue
					}
				}
				if g.siteAt(p%w.W, p/w.W, 1, 1) != nil {
					continue
				}
				taken = append(taken, p)
				return p
			}
			if !wantWild {
				break
			}
		}
		if minD > 1 {
			return -1
		}
		return -1
	}
	type slot struct{ kind, align string }
	pattern := []slot{{"sect", mrOrthodox}, {"clan", mrOrthodox}, {"guild", mrUnorthodox}, {"sect", mrOrthodox}, {"sect", mrUnorthodox},
		{"clan", mrOrthodox}, {"guild", mrUnorthodox}, {"sect", mrOrthodox}, {"guild", mrUnorthodox}}
	slots := []slot{{"cult", mrDemonic}}
	slots = append(slots, pattern[:n-1]...)
	used := map[string]bool{}
	var out []*MrSect
	for i, sl := range slots {
		hq := pickHQ(4, sl.align == mrDemonic)
		if hq < 0 {
			hq = pickHQ(2, false)
		}
		if hq < 0 {
			continue
		}
		cu := w.cultByID[w.Culture[hq]]
		s := &MrSect{Kind: sl.kind, Align: sl.align, WX: hq % w.W, WY: hq / w.W, State: w.State[hq], Culture: w.Culture[hq]}
		sr := NewRng(fmt.Sprintf("%s:murim:sect:%d", w.Seed, i))
		var place, name string
		for try := 0; try < 10; try++ {
			place = g.mrPlaceWord(cu, sr)
			switch {
			case sl.kind == "cult":
				name = Pick(sr, sk.Cults)
				if try > 0 {
					name = fmt.Sprintf("%s of %s", name, place)
				}
			case sl.kind == "clan":
				house := place
				if cu != nil && len(cu.Houses) > 0 {
					house = Pick(sr, cu.Houses)
				}
				name = fmt.Sprintf(Pick(sr, sk.Clans), house)
			case sl.kind == "guild":
				name = fmt.Sprintf(Pick(sr, sk.Guilds), place)
			case sl.align == mrUnorthodox:
				name = fmt.Sprintf(Pick(sr, sk.Black), place)
			default:
				name = fmt.Sprintf(Pick(sr, sk.Sects), place)
			}
			if !used[name] {
				break
			}
		}
		for used[name] {
			name += " II"
		}
		used[name] = true
		s.Name, s.ID = name, mrSlug(name)
		s.HQName = fmt.Sprintf(sk.Mount, place)
		if sl.kind == "clan" {
			s.HQName = place + " Manor"
		}
		if sl.kind == "guild" {
			s.HQName = place + " Hideout"
		}
		s.Creed = Pick(sr, mrCreeds[sl.align])
		s.Leader = g.cultureName(cu, sr)
		s.LeaderTit = sk.Lead[map[string]int{mrOrthodox: 0, mrUnorthodox: 1, mrDemonic: 2}[sl.align]]
		if sl.kind == "clan" {
			s.LeaderTit = "Clan Head"
		}
		for k := 0; k < 3; k++ {
			s.Elders = append(s.Elders, g.cultureName(cu, sr))
		}
		// signature art
		stat := sr.Int(0, 5)
		if sl.align == mrDemonic {
			stat = Pick(sr, []int{WIL, EGO})
		}
		adj := Pick(sr, sk.ArtAdj)
		s.Art = MrArt{Name: adj + " " + Pick(sr, sk.ArtForm), Stat: stat}
		kinds := mrStatKinds[stat]
		if sl.align == mrDemonic {
			kinds[1] = "drain"
		}
		usedWord := map[string]bool{}
		for k := 0; k < 3; k++ {
			words := mrTechWords[kinds[k]]
			wd := Pick(sr, words)
			for t := 0; t < 6 && usedWord[wd]; t++ {
				wd = Pick(sr, words)
			}
			usedWord[wd] = true
			s.Art.Tech[k] = MrTech{ID: fmt.Sprintf("%s:%d", s.ID, k), Name: wd, Kind: kinds[k], Need: mrTechNeed[k],
				Cool: mrTechCool[k], Cost: mrTechCost[k], Pow: k + 1}
		}
		out = append(out, s)
	}
	if len(out) < 2 {
		return nil
	}
	// branch halls: different states, orthodox in big cities, the cult in small hidden towns
	hallCount := map[int]int{}
	for i, s := range out {
		hr := NewRng(fmt.Sprintf("%s:murim:halls:%d", w.Seed, i))
		var pref, rest []*BurgData
		for _, b := range burgs {
			big := w.cityTier[b.ID] >= 1
			if (s.Align == mrOrthodox && big) || (s.Align == mrUnorthodox) || (s.Align == mrDemonic && !big) {
				pref = append(pref, b)
			} else {
				rest = append(rest, b)
			}
		}
		mrShuffle(hr, pref)
		mrShuffle(hr, rest)
		order := append(pref, rest...)
		want := 2 + hr.Int(0, 3)
		if s.Align == mrDemonic {
			want = 2 + hr.Int(0, 1)
		}
		if s.Kind == "clan" {
			want = mini(want, 3)
		}
		states := map[int]bool{}
		add := func(b *BurgData, requireNewState bool) {
			if len(s.Halls) >= want || hallCount[b.ID] >= 2 {
				return
			}
			if requireNewState && states[b.State] {
				return
			}
			for _, h := range s.Halls {
				if h.Burg == b.ID {
					return
				}
			}
			states[b.State] = true
			hallCount[b.ID]++
			s.Halls = append(s.Halls, MrHall{Burg: b.ID, State: b.State, WX: b.P % w.W, WY: b.P / w.W, Town: b.Name})
		}
		for _, b := range order {
			add(b, true)
		}
		for _, b := range order {
			add(b, false)
		}
	}
	// rivalries
	rival := func(a, b *MrSect) {
		if a == b {
			return
		}
		for _, x := range a.Rivals {
			if x == b.ID {
				return
			}
		}
		a.Rivals = append(a.Rivals, b.ID)
		b.Rivals = append(b.Rivals, a.ID)
	}
	rr := NewRng(w.Seed + ":murim:rivals")
	var orth, unor []*MrSect
	var cult *MrSect
	for _, s := range out {
		switch s.Align {
		case mrOrthodox:
			orth = append(orth, s)
		case mrUnorthodox:
			unor = append(unor, s)
		default:
			cult = s
		}
	}
	for _, o := range orth {
		if cult != nil {
			rival(o, cult)
		}
	}
	for _, u := range unor {
		if len(orth) > 0 {
			rival(u, orth[rr.Int(0, len(orth)-1)])
		}
	}
	if len(orth) >= 2 {
		rival(orth[0], orth[len(orth)-1])
	}
	if len(unor) >= 2 {
		rival(unor[0], unor[1])
	}
	for _, s := range out {
		sort.Strings(s.Rivals)
	}
	return out
}

// ---- world events: wars, raids, the Alliance tournament (pure functions of the day) ----

type MrWar struct {
	A, B       string
	Start, End int
}

// mrWarsAt returns the sect wars burning on a given day (10-day buckets, ~70% hold one).
func (g *Game) mrWarsAt(day int) []MrWar {
	var out []MrWar
	ss := g.Sects()
	if len(ss) == 0 {
		return nil
	}
	b := day / 10
	for _, bk := range []int{b - 1, b} {
		if bk < 0 {
			continue
		}
		r := NewRng(fmt.Sprintf("%s:murim:war:%d", g.W.Seed, bk))
		if !r.Chance(0.7) {
			continue
		}
		a := ss[r.Int(0, len(ss)-1)]
		if len(a.Rivals) == 0 {
			continue
		}
		bID := a.Rivals[r.Int(0, len(a.Rivals)-1)]
		start := bk*10 + r.Int(0, 4)
		end := start + 5 + r.Int(0, 6)
		if day >= start && day < end {
			out = append(out, MrWar{a.ID, bID, start, end})
		}
	}
	return out
}

type MrRaid struct {
	ID         string
	Burg       int
	Start, End int
}

// mrRaidAt: the Heavenly-Demon-style cult raids a small town for three days in each 21-day bucket.
func (g *Game) mrRaidAt(day int) *MrRaid {
	bs := g.mrBurgs()
	if len(bs) == 0 || g.mrCult() == nil {
		return nil
	}
	b := day / 21
	r := NewRng(fmt.Sprintf("%s:murim:raid:%d", g.W.Seed, b))
	start := b*21 + r.Int(0, 17)
	if day < start || day >= start+3 {
		return nil
	}
	// raids favour small towns: last two thirds of the list
	lo := len(bs) / 3
	t := bs[lo+r.Int(0, len(bs)-lo-1)]
	return &MrRaid{ID: fmt.Sprintf("raid%d", b), Burg: t.ID, Start: start, End: start + 3}
}

func (g *Game) mrCult() *MrSect {
	for _, s := range g.Sects() {
		if s.Align == mrDemonic {
			return s
		}
	}
	return nil
}

const mrTournPeriod = 56

// mrTournament: edition number and whether the Alliance tournament is open today, plus days to the next one.
func mrTournament(day int) (ed int, active bool, daysTo int) {
	d := day + 36
	ed = d / mrTournPeriod
	if d%mrTournPeriod < 7 {
		return ed, true, 0
	}
	return ed + 1, false, mrTournPeriod - d%mrTournPeriod
}

// mrHost: the great city that hosts the Alliance tournament (largest tier, then population).
func (g *Game) mrHost() *BurgData {
	bs := g.mrBurgs()
	var best *BurgData
	for _, b := range bs {
		if best == nil || g.W.cityTier[b.ID] > g.W.cityTier[best.ID] {
			best = b
		}
	}
	return best
}

// mrHostZone returns the zone coordinates of the host city's arena (or its centre).
func (g *Game) mrHostZone() (bd *BurgData, zx, zy int) {
	bd = g.mrHost()
	if bd == nil {
		return nil, 1, 1
	}
	cp := g.cityPlanFor(bd)
	var keys [][2]int
	for k, v := range cp.districts {
		if v == "arena" {
			keys = append(keys, k)
		}
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i][1] != keys[j][1] {
			return keys[i][1] < keys[j][1]
		}
		return keys[i][0] < keys[j][0]
	})
	if len(keys) > 0 {
		return bd, keys[0][0], keys[0][1]
	}
	return bd, 1, 1
}

// ---- compound builder ----

func mrBox(zn *Zone, x0, y0, x1, y1 int, wall, floor Terrain) {
	for y := y0; y <= y1; y++ {
		for x := x0; x <= x1; x++ {
			if x == x0 || x == x1 || y == y0 || y == y1 {
				zn.Set(x, y, wall)
			} else {
				zn.Set(x, y, floor)
			}
		}
	}
}

const (
	mrCX0, mrCX1 = 12, 67
	mrCY0, mrCY1 = 2, 22
)

// mrBuildCompound carves the walled headquarters into the centre zone of the sect's parasang.
func mrBuildCompound(zn *Zone) {
	for y := mrCY0; y <= mrCY1; y++ {
		for x := mrCX0; x <= mrCX1; x++ {
			zn.Set(x, y, TCobble)
		}
	}
	mrBox(zn, mrCX0, mrCY0, mrCX1, mrCY1, TWall, TCobble)
	zn.Set(40, mrCY1, TDoorOpen)
	for y := mrCY1 + 1; y < MH; y++ {
		zn.Set(40, y, TRoad)
	}
	// great hall with throne
	mrBox(zn, 28, 4, 52, 10, TWall, TFloor)
	for y := 5; y <= 9; y++ {
		zn.Set(40, y, TCarpet)
	}
	zn.Set(40, 5, TThrone)
	zn.Set(30, 5, TBrazier)
	zn.Set(50, 5, TBrazier)
	zn.Set(40, 10, TDoorOpen)
	// dormitories
	mrBox(zn, 14, 4, 24, 9, TWall, TFloor)
	zn.Set(19, 9, TDoorOpen)
	mrBox(zn, 56, 4, 65, 9, TWall, TFloor)
	zn.Set(60, 9, TDoorOpen)
	for _, x := range []int{16, 18, 20, 22} {
		zn.Set(x, 5, TBed)
		zn.Set(x, 7, TBed)
	}
	for _, x := range []int{58, 60, 62, 64} {
		zn.Set(x, 5, TBed)
	}
	// training yard, altar, well, statue
	for x := 18; x <= 28; x += 2 {
		zn.Set(x, 16, TDummy)
	}
	zn.Set(40, 14, TStatue)
	zn.Set(32, 13, TWell)
	zn.Set(56, 16, TAltar)
	zn.Set(55, 16, TBrazier)
	zn.Set(57, 16, TBrazier)
	for _, x := range []int{14, 65} {
		for _, y := range []int{12, 18} {
			zn.Set(x, y, TPillar)
		}
	}
	// vault
	mrBox(zn, 56, 13, 65, 19, TWall, TFloor)
	zn.Set(56, 14, TDoorOpen)
	zn.Set(63, 14, TCrate)
	zn.Set(63, 18, TCrate)
	zn.Set(64, 16, TShelf)
}

func mrInCompound(x, y int) bool { return x >= mrCX0 && x <= mrCX1 && y >= mrCY0 && y <= mrCY1 }

// mrCompass names the direction and distance (parasangs) from the player to a parasang.
func (g *Game) mrCompass(wx, wy int) string {
	cx, cy := g.WX, g.WY
	if g.Cur != nil {
		cx, cy = g.Cur.WX, g.Cur.WY
	}
	dx, dy := wx-cx, wy-cy
	d := maxi(abs(dx), abs(dy))
	if d == 0 {
		return "right here"
	}
	var dir string
	if dy < -d/3 {
		dir += "north"
	} else if dy > d/3 {
		dir += "south"
	}
	if dx > d/3 {
		dir += "east"
	} else if dx < -d/3 {
		dir += "west"
	}
	return fmt.Sprintf("%d parasangs %s", d, dir)
}
