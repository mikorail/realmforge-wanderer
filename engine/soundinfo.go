package engine

import "strings"

// Location soundscape facts for the web shell (prefix sdi): SoundInfo reads the zone, the terrain around
// the player (within sdiRadius tiles), the district kind, the dungeon theme and the NPC roles in view, and
// returns a small struct the shell turns into ambient beds. Read-only, deterministic, no RNG, no state,
// no g.R; the scan is a 25x25 window so it is cheap enough to run on every key. SoundJS makes it JS-safe.

const sdiRadius = 12

// SoundScene is what the soundscape layer needs to know about where the player is.
type SoundScene struct {
	Place   string          // street market tavern temple forge harbor farm wilds-forest wilds-grass wilds-desert wilds-mountain wilds-swamp wilds-snow coast cave dungeon crypt mine tower sky underwater ship-deck rail-station camp; "" on the world map
	Crowd   int             // 0..3: visible non-hostile NPC bands (0, 1-2, 3-6, 7+)
	Indoors bool            // under a roof, or inside a dungeon, tower or cave
	Depth   int             // zone z (0 surface, >0 underground, negative for towers and ships)
	Night   bool            // not daylight
	Weather string          // active weather id when outdoors, else ""
	Near    map[string]bool // forge well fountain bell altar stall tavern harbour water fire waterfall lava machine
	Culture string          // sphere of the land (festival and crowd murmur flavour)
	Genre   string
	Event   string // feast festival war plague curfew quiet
}

func sdiBiomeClass(b int) string {
	switch b {
	case 1, 2:
		return "wilds-desert"
	case 3, 4, 13:
		return "wilds-grass"
	case 5, 6, 7, 8, 9:
		return "wilds-forest"
	case 10, 11:
		return "wilds-snow"
	case 12, 15:
		return "wilds-swamp"
	case 14, 16:
		return "wilds-mountain"
	}
	return "wilds-grass"
}

func sdiDistrictPlace(k string) string {
	switch k {
	case "market":
		return "market"
	case "temple":
		return "temple"
	case "forge":
		return "forge"
	case "harbor":
		return "harbor"
	case "farms":
		return "farm"
	case "mines":
		return "mine"
	}
	return "street"
}

// SoundInfo returns the soundscape facts for the current zone, or a scene with an empty Place on the world map.
func (g *Game) SoundInfo() *SoundScene {
	s := &SoundScene{Near: map[string]bool{}, Genre: g.Genre}
	if g.P == nil || g.Cur == nil || g.OnWorld || g.Dead {
		return s
	}
	z := g.Cur
	s.Depth, s.Night = z.Z, !g.daylight()
	if w := g.activeWeather(); w != nil && w.Kind != nil {
		s.Weather = w.Kind.ID
	}
	if g.W != nil && g.W.In(z.WX, z.WY) {
		if cu := g.W.cultByID[g.W.Culture[g.W.Idx(z.WX, z.WY)]]; cu != nil {
			s.Culture = cu.Sphere
		}
	}
	px, py := g.P.X, g.P.Y
	under := z.At(px, py)
	s.Indoors = roofed(under) && z.Z == 0

	// terrain window
	var water, lava, crop, rock, snow, sand, tree, wood int
	near := s.Near
	machineP := g.Genre == "cyberpunk" || g.Genre == "steampunk" || g.Genre == "dieselpunk" || g.Genre == "atompunk"
	for y := maxi(0, py-sdiRadius); y <= mini(MH-1, py+sdiRadius); y++ {
		for x := maxi(0, px-sdiRadius); x <= mini(MW-1, px+sdiRadius); x++ {
			switch t := z.T[y*MW+x]; t {
			case TShallow, TDeep, THotSpring:
				water++
			case TLava:
				lava++
			case TAnvil:
				near["forge"] = true
			case TWell:
				near["well"] = true
			case TSpray, TGeyser:
				near["fountain"] = true
			case TFalls:
				near["waterfall"] = true
				water++
			case TAltar:
				near["altar"] = true
			case TStall:
				near["stall"] = true
			case TBoat:
				near["harbour"] = true
			case TBrazier, TChimney:
				near["fire"] = true
			case TMachine:
				near["machine"] = true
			case TPillar:
				if machineP {
					near["machine"] = true
				}
			case TCrop:
				crop++
			case TRock, TCliff, TWall:
				rock++
			case TSnow, TIce, TSnowDrift, TGlacier:
				snow++
			case TSand, TSalt:
				sand++
			case TTree, TPalm:
				tree++
			case TWoodFloor, TBridge:
				wood++
			}
			if z.Fire != nil && z.Fire[y*MW+x] > 0 {
				near["fire"] = true
			}
		}
	}
	near["water"] = water >= 6
	near["lava"] = lava >= 2
	if wood > 0 && water >= 6 && z.Kind == "town" {
		near["harbour"] = true
	}

	// NPCs in view
	n := 0
	tav := false
	for _, a := range z.Actors {
		if a.HP <= 0 || a.AI == "follower" || !g.visible(a.X, a.Y) || g.Hostile(a) {
			continue
		}
		n++
		if maxi(abs(a.X-px), abs(a.Y-py)) > sdiRadius {
			continue
		}
		switch a.Role {
		case "smith":
			near["forge"] = true
		case "priest":
			near["bell"] = near["bell"] || z.Kind == "town"
		case "merchant", "exotic", "apothecary":
			near["stall"] = true
		case "tavern", "innkeeper", "bard":
			tav = true
		case "ferry", "dockmaster", "shipwright", "captain":
			near["harbour"] = true
		}
	}
	switch {
	case n == 0:
		s.Crowd = 0
	case n <= 2:
		s.Crowd = 1
	case n <= 6:
		s.Crowd = 2
	default:
		s.Crowd = 3
	}

	// place
	district := ""
	var bd *BurgData
	if z.Kind == "town" {
		district, bd = g.districtAt(z.WX, z.WY, z.ZX, z.ZY)
		if district == "temple" || district == "palace" {
			near["bell"] = true
		}
	}
	theme := ""
	if z.Z != 0 && z.Z != shDeckZ || z.Kind == "dungeon" || z.Kind == "tower" {
		if st := g.siteAt(z.WX, z.WY, z.ZX, z.ZY); st != nil {
			theme = st.Theme
		}
	}
	low := strings.ToLower(z.Name)
	switch {
	case z.Z == shDeckZ:
		s.Place, s.Indoors = "ship-deck", false
	case z.Underwater:
		s.Place = "underwater"
	case z.Sky:
		s.Place = "sky"
	case z.Kind == "sea":
		s.Place = "coast"
	case z.Kind == "tower":
		s.Place, s.Indoors = "tower", true
	case z.Kind == "crypt" || theme == "catacomb" || theme == "barrow":
		s.Place, s.Indoors = "crypt", true
	case theme == "mine":
		s.Place, s.Indoors = "mine", true
	case z.Kind == "dungeon":
		s.Place, s.Indoors = "dungeon", true
	case z.Z > 0 || z.Dark && z.Z == 0 && z.Kind != "town" && z.Kind != "surface" && z.Kind != "outskirts":
		s.Place, s.Indoors = "cave", true
	case z.Kind == "camp":
		s.Place = "camp"
		near["fire"] = true
	case z.Kind == "town":
		switch {
		case strings.Contains(low, "station") && machineP:
			s.Place = "rail-station"
		case s.Indoors && tav:
			s.Place = "tavern"
		case s.Indoors && near["altar"]:
			s.Place = "temple"
		case s.Indoors && near["forge"]:
			s.Place = "forge"
		default:
			s.Place = sdiDistrictPlace(district)
		}
	default:
		cls := sdiBiomeClass(z.Biome)
		switch {
		case water >= 120 && (tree < 40 || z.Biome == 0):
			s.Place = "coast"
		case crop >= 8:
			s.Place = "farm"
		case snow >= 60 || under == TSnow || under == TIce:
			s.Place = "wilds-snow"
		case rock >= 90:
			s.Place = "wilds-mountain"
		case sand >= 80:
			s.Place = "wilds-desert"
		default:
			s.Place = cls
		}
	}
	if s.Place == "tavern" {
		near["tavern"] = true
	}

	// events
	s.Event = "quiet"
	if g.W != nil && g.W.In(z.WX, z.WY) {
		st := g.uwStateHere()
		switch {
		case z.Kind == "town" && bd != nil && g.W.hzOutbreakAt(bd.P) != nil:
			s.Event = "plague"
		case st > 0 && len(g.warsOf(st)) > 0:
			s.Event = "war"
		}
		if s.Event == "quiet" && z.Kind == "town" {
			if _, fd := g.feastToday(); fd != nil {
				s.Event = "feast"
				if fd.Kind == "spring" || fd.Kind == "harvest" || fd.Kind == "sun" || fd.Kind == "midwinter" {
					s.Event = "festival"
				}
			} else if s.Night && bd != nil && g.W.cityTier[bd.ID] >= 1 {
				s.Event = "curfew"
			}
		}
	}
	return s
}

// SoundJS converts a scene to plain maps and scalars for syscall/js (frame field "sound").
func SoundJS(s *SoundScene) map[string]any {
	near := map[string]any{}
	for k, v := range s.Near {
		near[k] = v
	}
	return map[string]any{"place": s.Place, "crowd": s.Crowd, "indoors": s.Indoors, "depth": s.Depth, "night": s.Night,
		"weather": s.Weather, "near": near, "culture": s.Culture, "genre": s.Genre, "event": s.Event}
}
