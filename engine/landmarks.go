package engine

import (
	"fmt"
	"sort"
	"strings"
)

// Landmarks: the famous places of a city (clock towers, palaces, universities, colosseums, columns
// to dead rulers). Everything is derived from the world data and an Rng seeded by the zone key, so
// names are stable and landmarkNames can list them without generating a single tile.

type lmSpec struct {
	Kind   string // tower palace mint gardens aqueduct bridge university observatory colosseum monument hero cathedral lighthouse bazaar guild
	Name   string
	Person *PersonData
	Year   int
	Unique bool
	Mech   bool   // tower: clockwork rather than bell
	Word   string // tower word
}

func lmRng(g *Game, zkey string) *Rng { return NewRng(g.W.Seed + ":landmark:" + zkey) }

func lmContainsWord(s, w string) bool {
	if len(w) < 3 {
		return false
	}
	for i := 0; ; {
		j := strings.Index(s[i:], w)
		if j < 0 {
			return false
		}
		a, b := i+j, i+j+len(w)
		isL := func(c byte) bool { return c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' || c >= 0x80 }
		if (a == 0 || !isL(s[a-1])) && (b >= len(s) || !isL(s[b])) {
			return true
		}
		i = a + 1
	}
}

func lmSurname(n string) string {
	f := strings.Fields(n)
	if len(f) == 0 {
		return n
	}
	return f[len(f)-1]
}

// lmDeadRulers are the realm's past rulers, most renowned first.
func (g *Game) lmDeadRulers(state int) []*PersonData {
	var out []*PersonData
	for i := range g.W.People {
		p := &g.W.People[i]
		if p.Role == "ruler" && !p.Alive && p.State == state && p.Name != "" {
			out = append(out, p)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Renown != out[j].Renown {
			return out[i].Renown > out[j].Renown
		}
		return out[i].ID < out[j].ID
	})
	if len(out) > 8 {
		out = out[:8]
	}
	return out
}

func (g *Game) lmLivingHeroes(bd *BurgData) []*PersonData {
	var out []*PersonData
	for i := range g.W.People {
		p := &g.W.People[i]
		if p.Role == "hero" && p.Alive && p.State == bd.State && p.Name != "" {
			out = append(out, p)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if (out[i].Burg == bd.ID) != (out[j].Burg == bd.ID) {
			return out[i].Burg == bd.ID
		}
		if out[i].Renown != out[j].Renown {
			return out[i].Renown > out[j].Renown
		}
		return out[i].ID < out[j].ID
	})
	if len(out) > 5 {
		out = out[:5]
	}
	return out
}

func (g *Game) lmLivingRuler(bd *BurgData) *PersonData {
	st := g.W.stateByID[bd.State]
	if st == nil {
		return nil
	}
	for i := range g.W.People {
		p := &g.W.People[i]
		if p.ID == st.RulerID && st.RulerID != 0 {
			return p
		}
	}
	for i := range g.W.People {
		p := &g.W.People[i]
		if p.Role == "ruler" && p.Alive && p.State == bd.State {
			return p
		}
	}
	return nil
}

func (g *Game) lmTowerWord(sphere string) (word string, mech bool) {
	switch g.Genre {
	case "steampunk", "dieselpunk", "atompunk", "cyberpunk":
		return "Clock Tower", true
	}
	switch sphere {
	case "sinic", "japonic", "sea":
		return "Pagoda", false
	case "arabic", "persian":
		return "Minaret", false
	case "nordic", "germanic":
		return "Stave Tower", false
	case "indic":
		return "Spire", false
	case "maya", "nahua":
		return "Step Pyramid", false
	case "yoruba", "polynesian", "ethiopic":
		return "Drum Tower", false
	}
	return "Bell Tower", false
}

func (g *Game) lmFaithWord(bd *BurgData, sphere string) string {
	deity := ""
	if st := g.W.stateByID[bd.State]; st != nil {
		if rl := g.W.relByID[st.Religion]; rl != nil {
			deity = rl.Deity
		}
	}
	of := bd.Name
	if deity != "" {
		of = deity
	}
	switch sphere {
	case "arabic", "persian":
		return "Great Mosque of " + bd.Name
	case "sinic", "japonic", "sea", "indic", "maya", "nahua":
		return "Great Temple of " + of
	}
	return "Cathedral of " + of
}

// lmSpecs decides which landmarks one district of a city has.
func (g *Game) lmSpecs(bd *BurgData, kind string, lr *Rng) []lmSpec {
	w := g.W
	tier := w.cityTier[bd.ID]
	sphere := ""
	if cu := w.cultByID[bd.Culture]; cu != nil {
		sphere = cu.Sphere
	}
	st := w.stateByID[bd.State]
	var out []lmSpec
	// roll always consumes the same numbers so names stay stable
	roll := func(p float64) bool { return lr.Float() < p }
	dead := g.lmDeadRulers(bd.State)
	pickDead := func() *PersonData {
		if len(dead) == 0 {
			lr.Int(0, 1)
			return nil
		}
		return dead[lr.Int(0, len(dead)-1)]
	}
	tp := func(a, b, c float64) float64 { return []float64{a, b, c}[clampi(tier, 0, 2)] }
	add := func(k, name string, p float64) *lmSpec {
		out = append(out, lmSpec{Kind: k, Name: name, Unique: roll(p)})
		return &out[len(out)-1]
	}
	monument := func(p float64) {
		d := pickDead()
		pre := Pick(lr, []string{"Column of ", "Obelisk of ", "Statue of "})
		if d == nil || !roll(p) {
			return
		}
		s := add("monument", pre+d.Name, 0.5)
		s.Person = d
		s.Word = strings.TrimSpace(strings.TrimSuffix(pre, " of "))
	}
	switch kind {
	case "plaza":
		if roll(tp(0.15, 0.6, 1)) {
			word, mech := g.lmTowerWord(sphere)
			s := add("tower", word+" of "+bd.Name, 0.55)
			s.Mech, s.Word = mech, word
			if mech {
				s.Name = "Great Clock of " + bd.Name
			} else if word == "Bell Tower" {
				s.Name = "Great Bell of " + bd.Name
			}
		}
		monument(tp(0.1, 0.5, 0.7))
		if hs := g.lmLivingHeroes(bd); len(hs) > 0 {
			h := hs[lr.Int(0, len(hs)-1)]
			if roll(0.4) {
				s := add("hero", "Statue of "+h.Name, 0.6)
				s.Person = h
			}
		} else {
			lr.Int(0, 1)
			lr.Float()
		}
	case "palace":
		ruler := g.lmLivingRuler(bd)
		house := bd.Name
		if ruler != nil {
			house = lmSurname(ruler.Name)
		} else if st != nil {
			house = st.Name
		}
		s := add("palace", "Palace of House "+house, 0.8)
		s.Person = ruler
		switch {
		case roll(0.45):
			add("gardens", "Royal Gardens of "+bd.Name, 0)
		case tier == 2 && roll(0.6):
			add("mint", "Royal Mint of "+bd.Name, 0.8)
		default:
			monument(1)
		}
	case "market":
		if roll(tp(0.2, 0.8, 1)) {
			add("bazaar", "Grand Bazaar of "+bd.Name, 0.5)
		}
		if tier == 2 && roll(0.5) {
			add("mint", "Treasury of "+bd.Name, 0.8)
		}
	case "temple":
		if roll(tp(0.3, 0.9, 1)) {
			add("cathedral", g.lmFaithWord(bd, sphere), 0.6)
		}
	case "academy":
		if roll(tp(0.5, 1, 1)) {
			s := add("university", "University of "+bd.Name, 0.6)
			s.Year = w.Year - lr.Int(80, 240)
			for _, ei := range w.eventsAt[bd.P] {
				if y := w.Events[ei].Y; y < s.Year || ei == w.eventsAt[bd.P][0] && false {
					s.Year = y
				}
			}
		}
		if roll(0.55) {
			add("observatory", "Observatory of "+bd.Name, 0.5)
		}
	case "arena":
		word := "Colosseum"
		switch sphere {
		case "maya", "nahua":
			word = "Great Ballcourt"
		case "arabic", "persian":
			word = "Hippodrome"
		case "steppe", "nordic", "germanic":
			word = "Great Ring"
		}
		add("colosseum", word+" of "+bd.Name, 0.55)
	case "harbor":
		add("lighthouse", "Lighthouse of "+bd.Name, 0.6)
	case "forge":
		if roll(0.75) {
			add("guild", bd.Name+" Forgemasters' Hall", 0.5)
		}
	case "artisans":
		if roll(0.75) {
			add("guild", "Guildhall of the "+bd.Name+" Artisans", 0.5)
		}
	case "homes":
		if tier >= 1 && roll(0.5) {
			if roll(0.5) {
				add("aqueduct", "Aqueduct of "+bd.Name, 0)
			} else {
				add("gardens", "Gardens of "+bd.Name, 0)
			}
		}
	}
	// a grand bridge where a river runs through the parasang
	if bd.P >= 0 && bd.P < len(w.River) && w.River[bd.P] > 0 {
		ok := kind == "homes" || kind == "slums" || kind == "artisans" || kind == "market" || kind == "plaza" && tier < 1
		d := pickDead()
		if ok && roll(0.6) && len(out) < 2 {
			name := "Old Bridge of " + bd.Name
			s := add("bridge", name, 0.4)
			if d != nil {
				s.Name = "Bridge of " + d.Name
				s.Person = d
			}
		}
	}
	if len(out) > 2 {
		out = out[:2]
	}
	return out
}

// landmarkNames lists the landmarks a city will have (towns of one zone have none).
func (g *Game) landmarkNames(bd *BurgData) []string {
	cp := g.cityPlanFor(bd)
	if cp.tier == 0 {
		return nil
	}
	var subs [][2]int
	for k := range cp.districts {
		subs = append(subs, k)
	}
	sort.Slice(subs, func(i, j int) bool {
		if subs[i][1] != subs[j][1] {
			return subs[i][1] < subs[j][1]
		}
		return subs[i][0] < subs[j][0]
	})
	wx, wy := bd.P%g.W.W, bd.P/g.W.W
	var out []string
	seen := map[string]bool{}
	for _, s := range subs {
		key := ZoneKey(wx, wy, s[0], s[1], 0)
		for _, sp := range g.lmSpecs(bd, cp.districts[s], lmRng(g, key)) {
			if !seen[sp.Name] {
				seen[sp.Name] = true
				out = append(out, sp.Name)
			}
		}
	}
	return out
}

// lmChronicleLine finds a real event mentioning a person.
func (g *Game) lmChronicleLine(p *PersonData) string {
	for _, e := range g.W.Events {
		if lmContainsWord(e.Text, p.Name) {
			t := e.Text
			if len(t) > 220 {
				t = t[:220] + "..."
			}
			return fmt.Sprintf("In the year %d: %s", e.Y, t)
		}
	}
	return ""
}

func (g *Game) lmInscription(sp lmSpec, bd *BurgData) string {
	w := g.W
	realm := bd.Name
	if st := w.stateByID[bd.State]; st != nil && st.ID > 0 {
		realm = st.FullName
	}
	head := "{{Y|" + sp.Name + "}}. "
	switch sp.Kind {
	case "monument", "hero":
		p := sp.Person
		if p == nil {
			return head
		}
		title := p.Title
		if title == "" {
			title = p.Role
		}
		s := head
		if sp.Kind == "hero" {
			s += fmt.Sprintf("Raised by the people of %s in honour of {{Y|%s}}, %s, who yet lives.", bd.Name, p.Name, title)
		} else {
			s += fmt.Sprintf("Erected by %s in memory of {{Y|%s}}, %s. Born in the year %d, died in %d%s.", realm, p.Name, title, p.Born, p.Died, causeText(p.Cause))
		}
		if c := g.lmChronicleLine(p); c != "" {
			s += "\n\n" + c
		}
		return s
	case "bridge":
		s := head + fmt.Sprintf("The great span over the river at %s.", bd.Name)
		if sp.Person != nil {
			s += fmt.Sprintf(" Built in the days of {{Y|%s}}, %s.", sp.Person.Name, sp.Person.Title)
		}
		return s
	case "palace":
		s := head + fmt.Sprintf("Seat of %s.", realm)
		if sp.Person != nil {
			s += fmt.Sprintf(" Home of {{Y|%s}}, %s, who rules it now.", sp.Person.Name, sp.Person.Title)
		}
		return s
	case "university":
		return head + fmt.Sprintf("Founded in the year %d. Here %s sends its scholars; the great hall holds the oldest books of the realm.", sp.Year, realm)
	case "tower":
		if sp.Mech {
			return head + "Its gears have kept the hours of " + bd.Name + " since before living memory."
		}
		return head + "Its voice calls the hours and the alarms of " + bd.Name + "."
	case "colosseum":
		return head + "Here the champions of " + realm + " fight for glory. Bring a weapon and a good reason."
	case "lighthouse":
		return head + "Its fire guides ships into the harbor of " + bd.Name + " through the dark."
	case "cathedral":
		return head + "Pilgrims come from every road to the high altar within."
	case "observatory":
		return head + "From this roof the scholars of " + bd.Name + " chart the wandering lights of the sky."
	case "mint":
		return head + "Every coin of " + realm + " is struck within. Guards watch the vault."
	case "bazaar":
		return head + "All the trade of " + bd.Name + " passes under this roof."
	case "guild":
		return head + "Masters of the craft gather here; their marks are carved on the lintel."
	case "gardens":
		return head + "Cool water, green walks, and a rest from the dust of the streets."
	case "aqueduct":
		return head + "On these arches fresh water reaches " + bd.Name + " from the far hills."
	}
	return head
}

// lmItem forges the landmark's unique treasure from an ordinary maker, then renames and strengthens it.
func (g *Game) lmItem(sp lmSpec, bd *BurgData, lr *Rng) *Item {
	var it *Item
	var name, lore string
	bonus := map[int]int{}
	city := bd.Name
	switch sp.Kind {
	case "tower":
		it = g.MakeOddity(lr, 5)
		if sp.Mech {
			name = "Mainspring of the Great Clock of " + city
			lore = "The master spring that drives the hours of " + city + ". Whoever holds it holds time."
		} else {
			name = "Key of the Bell of " + city
			lore = "The key to the great bell's mechanism, cast the same day as the bell itself."
		}
		bonus = map[int]int{WIL: 3, INT: 2}
	case "palace":
		it = g.MakeArmor("back", 5, lr)
		who := "the Crown"
		if sp.Person != nil {
			who = sp.Person.Name
		}
		name = "Chain of Office of " + who
		lore = "Heavy links of gold worn by every ruler who sat in the Palace of " + city + "."
		bonus = map[int]int{EGO: 4, WIL: 2}
	case "mint":
		it = g.MakeOddity(lr, 5)
		name = "Master Die of the Mint of " + city
		lore = "The die from which every coin of the realm is struck."
		bonus = map[int]int{EGO: 3, INT: 2}
	case "university":
		it = g.MakeOddity(lr, 5)
		name = "Founder's Codex of " + city
		lore = fmt.Sprintf("Set down in the year %d, the first book of the University of %s.", sp.Year, city)
		bonus = map[int]int{INT: 5, WIL: 1}
	case "observatory":
		it = g.MakeArmor("face", 5, lr)
		name = "Star-Glass of " + city
		lore = "A lens ground to find what the naked eye cannot."
		bonus = map[int]int{INT: 3, AGI: 2}
	case "colosseum":
		it = g.MakeWeapon("longblade", 5, lr)
		name = "Champion's Blade of " + city
		lore = "Awarded for one season and returned to the sand when its bearer falls."
		bonus = map[int]int{STR: 3, AGI: 2}
	case "lighthouse":
		it = g.MakeLight(2)
		name = "Beacon-Lantern of " + city
		lore = "The lamp that has never gone out on the harbor light."
		it.Light = 16
		bonus = map[int]int{AGI: 2, WIL: 1}
	case "cathedral":
		it = g.MakeOddity(lr, 5)
		name = "Reliquary of " + city
		lore = "A sealed casket of the founding faithful, kept behind the high altar."
		bonus = map[int]int{WIL: 4, TOU: 2}
	case "bazaar":
		it = g.MakeArmor("feet", 5, lr)
		name = "Spice-Prince's Boots of " + city
		lore = "Worn by the richest merchant of the bazaar on the day the market charter was signed."
		bonus = map[int]int{AGI: 3, EGO: 2}
	case "guild":
		it = g.MakeWeapon("cudgel", 5, lr)
		name = "Grandmaster's Hammer of " + city
		lore = "Passed from master to master of the guild for generations."
		bonus = map[int]int{STR: 3, TOU: 2}
	case "monument":
		it = g.MakeArmor("head", 5, lr)
		who := "a forgotten king"
		if sp.Person != nil {
			who = sp.Person.Name
		}
		name = "Crown of " + who
		lore = "Left at the foot of the " + strings.ToLower(sp.Word) + " when " + who + " was laid to rest."
		bonus = map[int]int{EGO: 3, WIL: 2}
	case "hero":
		it = g.MakeWeapon("axe", 5, lr)
		who := "the hero"
		if sp.Person != nil {
			who = sp.Person.Name
		}
		name = "Gift-Axe of " + who
		lore = "Offered at the statue of " + who + " by a grateful city."
		bonus = map[int]int{STR: 4, AGI: 2}
	case "bridge":
		it = g.MakeArmor("hands", 5, lr)
		name = "Toll-Keeper's Gauntlets of " + city
		lore = "Taken from the keystone chamber of the bridge, where the first toll was ever paid."
		bonus = map[int]int{STR: 3, TOU: 2}
	default:
		return nil
	}
	it.Name, it.Lore, it.Rarity, it.Tier, it.F, it.Ident = name, lore, "unique", 5, CYellow, true
	it.Value = 600 + lr.Int(0, 200)
	if it.Bonus == nil {
		it.Bonus = map[int]int{}
	}
	for k, v := range bonus {
		it.Bonus[k] += v
	}
	switch it.Kind {
	case "weapon":
		it.DS += 3
		it.PV += 2
	case "armor":
		it.AV += 3
	}
	return it
}

// placeLandmarks adds the district's landmarks to the finished zone.
func (g *Game) placeLandmarks(zn *Zone, r *Rng, bd *BurgData, kind string) {
	if bd == nil || zn == nil {
		return
	}
	lr := lmRng(g, zn.Key)
	specs := g.lmSpecs(bd, kind, lr)
	cu := g.W.cultByID[bd.Culture]
	named := false
	for _, sp := range specs {
		sp := sp
		// try the full size first, then a squeezed version
		try := func(wmin, wmax, hmin, hmax int) ([4]int, bool) {
			if rc, ok := g.lot(zn, lr, wmin, wmax, hmin, hmax); ok {
				return rc, true
			}
			return g.lot(zn, lr, wmin-3, wmin, hmin-1, hmin)
		}
		var rc [4]int
		var ok bool
		var ix, iy, px, py int // item spot, plaque spot
		plaque := func(rc [4]int, dx, dy int) {
			px, py = dx+2, dy+1
			if dy == rc[1] {
				py = dy - 1
			}
			if zn.At(px, py) != TCobble && zn.At(px, py) != TDirt {
				px, py = rc[0]-1, rc[1]+rc[3]/2
			}
		}
		// hall builds a walled room whose top row holds the decorations and mid row the treasure
		hall := func(wmin, wmax, hmin, hmax int, wall, floor, deco Terrain, step int) {
			if rc, ok = try(wmin, wmax, hmin, hmax); !ok {
				return
			}
			dx, dy := g.building(zn, rc[0], rc[1], rc[2], rc[3], wall, floor)
			for x := rc[0] + 2; x < rc[0]+rc[2]-2; x += step {
				if x != dx {
					zn.Set(x, rc[1]+1, deco)
				}
			}
			ix, iy = rc[0]+rc[2]-3, rc[1]+2
			plaque(rc, dx, dy)
		}
		switch sp.Kind {
		case "palace":
			hall(18, 24, 8, 10, TWall, TCarpet, TStatue, 4)
			if ok {
				for x := rc[0] + 3; x < rc[0]+rc[2]-3; x += 4 {
					zn.Set(x, rc[1]+rc[3]-3, TPillar)
				}
			}
		case "cathedral":
			hall(14, 18, 8, 10, TWall, TCarpet, TPillar, 3)
			if ok {
				zn.Set(rc[0]+rc[2]/2, rc[1]+1, TAltar)
			}
		case "university":
			hall(14, 18, 7, 9, TWall, TFloor, TShelf, 2)
		case "bazaar":
			hall(14, 20, 7, 9, TWall, TFloor, TStall, 3)
		case "guild":
			hall(10, 14, 6, 8, TWall, TFloor, TAnvil, 3)
		case "mint":
			hall(11, 14, 6, 8, TWall, TFloor, TCrate, 2)
		case "colosseum":
			if rc, ok = try(18, 24, 10, 11); ok {
				dx, dy := g.building(zn, rc[0], rc[1], rc[2], rc[3], TWall, TDirt)
				for x := rc[0] + 1; x < rc[0]+rc[2]-1; x++ {
					zn.Set(x, rc[1]+1, TSeat)
					zn.Set(x, rc[1]+rc[3]-2, TSeat)
				}
				zn.Set(dx, dy-1, TDirt)
				zn.Set(dx, dy, TDoor)
				ix, iy = rc[0]+rc[2]/2, rc[1]+rc[3]/2
				plaque(rc, dx, dy)
			}
		case "tower", "lighthouse", "observatory":
			if rc, ok = try(5, 7, 5, 7); ok {
				dx, dy := g.building(zn, rc[0], rc[1], rc[2], rc[3], TWall, TFloor)
				mid := TBrazier
				switch {
				case sp.Kind == "tower" && sp.Mech, sp.Kind == "observatory":
					mid = TMachine
				}
				zn.Set(rc[0]+rc[2]/2, rc[1]+rc[3]/2, mid)
				zn.Set(rc[0], rc[1], TPillar)
				zn.Set(rc[0]+rc[2]-1, rc[1], TPillar)
				ix, iy = rc[0]+1, rc[1]+1
				plaque(rc, dx, dy)
			}
		case "monument", "hero":
			if rc, ok = try(5, 5, 5, 5); ok {
				x0, y0 := rc[0], rc[1]
				for yy := y0; yy < y0+5; yy++ {
					for xx := x0; xx < x0+5; xx++ {
						zn.Set(xx, yy, TFloor)
					}
				}
				c := TPillar
				if sp.Word == "Statue" || sp.Kind == "hero" {
					c = TStatue
				}
				zn.Set(x0+2, y0+2, c)
				corner := TPillar
				if sp.Word == "Obelisk" {
					corner = TBrazier
				}
				for _, o := range [][2]int{{0, 0}, {4, 0}, {0, 4}, {4, 4}} {
					zn.Set(x0+o[0], y0+o[1], corner)
				}
				zn.Set(x0+2, y0+4, TInscription)
				px, py = x0+2, y0+4
				ix, iy = x0+1, y0+2
			}
		case "gardens":
			if rc, ok = try(10, 13, 6, 8); ok {
				for yy := rc[1]; yy < rc[1]+rc[3]; yy++ {
					for xx := rc[0]; xx < rc[0]+rc[2]; xx++ {
						t := TGrass
						switch {
						case (xx+yy)%7 == 0 && yy != rc[1]+rc[3]/2 && xx != rc[0]+rc[2]/2:
							t = TTree
						case xx == rc[0] || xx == rc[0]+rc[2]-1 || yy == rc[1] || yy == rc[1]+rc[3]-1:
							t = TBush
						}
						zn.Set(xx, yy, t)
					}
				}
				zn.Set(rc[0]+rc[2]/2, rc[1]+rc[3]/2, TWell)
				zn.Set(rc[0]+2, rc[1]+rc[3]/2, TStatue)
				// open gates on the cross-axes
				zn.Set(rc[0]+rc[2]/2, rc[1], TGrass)
				zn.Set(rc[0]+rc[2]/2, rc[1]+rc[3]-1, TGrass)
				zn.Set(rc[0], rc[1]+rc[3]/2, TGrass)
				zn.Set(rc[0]+rc[2]-1, rc[1]+rc[3]/2, TGrass)
				px, py = rc[0]+rc[2]/2+2, rc[1]+rc[3]/2
				zn.Set(px, py, TInscription)
			}
		case "aqueduct":
			if rc, ok = try(13, 17, 3, 4); ok {
				for xx := rc[0]; xx < rc[0]+rc[2]; xx++ {
					if (xx-rc[0])%2 == 0 {
						zn.Set(xx, rc[1], TPillar)
						zn.Set(xx, rc[1]+2, TPillar)
					}
					zn.Set(xx, rc[1]+1, TShallow)
				}
				px, py = rc[0]+rc[2]-1, rc[1]+rc[3]-1
				zn.Set(px, py, TInscription)
			}
		case "bridge":
			if rc, ok = try(11, 13, 6, 7); ok {
				cx := rc[0] + rc[2]/2
				for yy := rc[1] + 1; yy < rc[1]+rc[3]-1; yy++ {
					for xx := rc[0]; xx < rc[0]+rc[2]; xx++ {
						zn.Set(xx, yy, TShallow)
					}
				}
				for yy := rc[1]; yy < rc[1]+rc[3]; yy++ {
					zn.Set(cx, yy, TBridge)
				}
				for _, yy := range []int{rc[1], rc[1] + rc[3] - 1} {
					zn.Set(cx-1, yy, TPillar)
					zn.Set(cx+1, yy, TPillar)
				}
				ix, iy = cx, rc[1]+rc[3]/2
				px, py = cx+2, rc[1]
				zn.Set(px, py, TInscription)
			}
		}
		if !ok {
			continue
		}
		if zn.At(px, py) != TInscription && zn.At(px, py) != TDoor && (zn.At(px, py) == TCobble || zn.At(px, py) == TDirt) {
			zn.Set(px, py, TInscription)
		}
		if zn.At(px, py) == TInscription {
			zn.Notes[py*MW+px] = g.lmInscription(sp, bd)
		}
		if !named {
			named = true
			zn.Name = bd.Name + ", " + sp.Name
		}
		if sp.Unique && zn.Walkable(ix, iy) && len(zn.ItemsAt(ix, iy)) == 0 {
			if it := g.lmItem(sp, bd, lr); it != nil {
				it.X, it.Y = ix, iy
				zn.Items = append(zn.Items, it)
				if (sp.Kind == "palace" || sp.Kind == "mint") && cu != nil {
					for i := 0; i < 1+lr.Int(0, 1); i++ {
						a := g.makeNPC("guard", bd, cu, r)
						a.X, a.Y = ix-1-i, iy+1
						a.HomeX, a.HomeY = a.X, a.Y
						if !zn.Walkable(a.X, a.Y) || zn.ActorAt(a.X, a.Y) != nil {
							g.nudge(zn, a)
							a.HomeX, a.HomeY = a.X, a.Y
						}
						zn.Actors = append(zn.Actors, a)
					}
				}
			}
		}
	}
}
