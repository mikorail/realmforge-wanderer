package engine

import (
	"encoding/json"
	"strings"
)

// WorldData is the snapshot Realmforge exports from its live simulation (src/lib/adventure/export.ts).
// The overworld is rasterised into W×H parasangs; every per-parasang array is row-major.
type WorldData struct {
	Seed       string            `json:"seed"`
	Name       string            `json:"name"`
	Genre      string            `json:"genre"`
	Planet     string            `json:"planet"`
	PlanetKind string            `json:"planetKind"`
	Year       int               `json:"year"`
	Era        string            `json:"era"`
	W          int               `json:"w"`
	H          int               `json:"h"`
	Biome      []int             `json:"biome"`
	Height     []int             `json:"height"`
	Temp       []int             `json:"temp"`
	River      []int             `json:"river"`
	Road       []int             `json:"road"`
	State      []int             `json:"state"`
	Culture    []int             `json:"culture"`
	Religion   []int             `json:"religion"`
	Burg       []int             `json:"burg"`
	Cell       []int             `json:"cell"`
	Layers     []LayerData       `json:"layers"`
	Burgs      []BurgData        `json:"burgs"`
	States     []StateData       `json:"states"`
	Cultures   []CultureData     `json:"cultures"`
	Religions  []ReligionData    `json:"religions"`
	People     []PersonData      `json:"people"`
	Events     []EventData       `json:"events"`
	Monsters   []MonsterData     `json:"monsters"`
	Artifacts  []ArtifactData    `json:"artifacts"`
	Wonders    []WonderData      `json:"wonders"`
	Species    []SpeciesData     `json:"species"`
	Terms      map[string]string `json:"terms"`
	LayerNames map[string]string `json:"layerNames"`

	burgByID    map[int]*BurgData
	stateByID   map[int]*StateData
	cultByID    map[int]*CultureData
	relByID     map[int]*ReligionData
	specByID    map[string]*SpeciesData
	eventsAt    map[int][]int
	monstersAt  map[int][]int
	wondersAt   map[int][]int
	deepOpen    map[int][]bool // depth → open mask
	deepBurg    map[int]map[int]int
	skyOpen     []bool
	seaOpen     []bool
	skyBurg     map[int]int
	seaBurg     map[int]int
	maxDepth    int
	peopleByBrg map[int][]int
}

type LayerData struct {
	Kind  string      `json:"kind"`
	Depth int         `json:"depth"`
	Name  string      `json:"name"`
	Open  string      `json:"open"` // '0'/'1' per parasang
	Burgs map[int]int `json:"burgs"`
}

type BurgData struct {
	ID        int            `json:"id"`
	Name      string         `json:"name"`
	Kind      string         `json:"kind"`
	Layer     string         `json:"layer"`
	Depth     int            `json:"depth"`
	State     int            `json:"state"`
	Culture   int            `json:"culture"`
	Pop       float64        `json:"pop"`
	Capital   bool           `json:"capital"`
	Port      bool           `json:"port"`
	Alive     bool           `json:"alive"`
	P         int            `json:"p"` // parasang index
	Districts map[string]int `json:"districts"`
	Governor  string         `json:"governor"`
	Bandit    bool           `json:"bandit"`
	Original  string         `json:"original"`
}

type StateData struct {
	ID        int         `json:"id"`
	Name      string      `json:"name"`
	FullName  string      `json:"fullName"`
	Form      string      `json:"form"`
	Color     string      `json:"color"`
	Culture   int         `json:"culture"`
	Religion  int         `json:"religion"`
	Alive     bool        `json:"alive"`
	Ruler     string      `json:"ruler"`
	RulerID   int         `json:"rulerId"`
	Species   string      `json:"species"`
	Ideology  string      `json:"ideology"`
	Relations map[int]int `json:"relations"`
	Wars      []int       `json:"wars"`
	Capital   int         `json:"capital"`
	Founded   int         `json:"founded"`
	Fell      int         `json:"fell"`
}

type CultureData struct {
	ID      int      `json:"id"`
	Name    string   `json:"name"`
	Sphere  string   `json:"sphere"`
	Species string   `json:"species"`
	Layer   string   `json:"layer"`
	Given   []string `json:"given"`
	Houses  []string `json:"houses"`
	Places  []string `json:"places"`
	Food    string   `json:"food"`
	Dress   string   `json:"dress"`
}

type ReligionData struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Deity    string `json:"deity"`
	Theology string `json:"theology"`
	Doctrine string `json:"doctrine"`
	Type     string `json:"type"`
}

type PersonData struct {
	ID       int      `json:"id"`
	Name     string   `json:"name"`
	Role     string   `json:"role"`
	Burg     int      `json:"burg"`
	State    int      `json:"state"`
	Culture  int      `json:"culture"`
	Religion int      `json:"religion"`
	Species  string   `json:"species"`
	Martial  int      `json:"martial"`
	Diplo    int      `json:"diplomacy"`
	Steward  int      `json:"stewardship"`
	Intrigue int      `json:"intrigue"`
	Born     int      `json:"born"`
	Died     int      `json:"died"`
	Cause    string   `json:"cause"`
	Alive    bool     `json:"alive"`
	Traits   []string `json:"traits"`
	Renown   int      `json:"renown"`
	P        int      `json:"p"`
	Title    string   `json:"title"`
}

type EventData struct {
	Y    int    `json:"y"`
	Kind string `json:"kind"`
	Text string `json:"text"`
	P    int    `json:"p"` // -1 = nowhere in particular
	St   int    `json:"st"`
}

type MonsterData struct {
	ID       int     `json:"id"`
	Name     string  `json:"name"`
	Species  string  `json:"species"`
	Tier     string  `json:"tier"`
	Cat      string  `json:"cat"`
	Power    int     `json:"power"`
	HP       int     `json:"hp"`
	Armor    float64 `json:"armor"`
	Lore     string  `json:"lore"`
	Trophy   string  `json:"trophy"`
	P        int     `json:"p"`
	Layer    string  `json:"layer"`
	Depth    int     `json:"depth"`
	Lair     string  `json:"lair"`
	Immortal bool    `json:"immortal"`
}

type ArtifactData struct {
	ID         int    `json:"id"`
	Name       string `json:"name"`
	Kind       string `json:"kind"`
	Fame       int    `json:"fame"`
	Year       int    `json:"year"`
	Origin     string `json:"origin"`
	Stat       string `json:"stat"`
	Val        int    `json:"val"`
	HolderType string `json:"holderType"`
	HolderID   int    `json:"holderId"`
	Holder     string `json:"holder"`
	Lore       string `json:"lore"`
	P          int    `json:"p"`
}

type WonderData struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	Kind      string `json:"kind"`
	P         int    `json:"p"`
	Layer     string `json:"layer"`
	Destroyed bool   `json:"destroyed"`
	Lore      string `json:"lore"`
	Builder   string `json:"builder"`
	Year      int    `json:"year"`
	Effect    string `json:"effect"`
}

type SpeciesData struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	Plural   string   `json:"plural"`
	Adj      string   `json:"adjective"`
	Desc     string   `json:"desc"`
	Martial  int      `json:"martial"`
	Diplo    int      `json:"diplomacy"`
	Steward  int      `json:"stewardship"`
	Intrigue int      `json:"intrigue"`
	Health   int      `json:"health"`
	Traits   []string `json:"traits"`
	Layers   []string `json:"layers"`
	Common   int      `json:"common"` // how many cultures in this world are of this species
}

func ParseWorld(s string) (*WorldData, error) {
	w := &WorldData{}
	if err := json.Unmarshal([]byte(s), w); err != nil {
		return nil, err
	}
	w.index()
	return w, nil
}

func mask(s string, n int) []bool {
	m := make([]bool, n)
	for i := 0; i < n && i < len(s); i++ {
		m[i] = s[i] == '1'
	}
	return m
}

func (w *WorldData) index() {
	n := w.W * w.H
	w.burgByID = map[int]*BurgData{}
	for i := range w.Burgs {
		w.burgByID[w.Burgs[i].ID] = &w.Burgs[i]
	}
	w.stateByID = map[int]*StateData{}
	for i := range w.States {
		w.stateByID[w.States[i].ID] = &w.States[i]
	}
	w.cultByID = map[int]*CultureData{}
	for i := range w.Cultures {
		w.cultByID[w.Cultures[i].ID] = &w.Cultures[i]
	}
	w.relByID = map[int]*ReligionData{}
	for i := range w.Religions {
		w.relByID[w.Religions[i].ID] = &w.Religions[i]
	}
	w.specByID = map[string]*SpeciesData{}
	for i := range w.Species {
		w.specByID[w.Species[i].ID] = &w.Species[i]
	}
	w.eventsAt = map[int][]int{}
	for i, e := range w.Events {
		if e.P >= 0 {
			w.eventsAt[e.P] = append(w.eventsAt[e.P], i)
		}
	}
	w.monstersAt = map[int][]int{}
	for i, m := range w.Monsters {
		w.monstersAt[m.P] = append(w.monstersAt[m.P], i)
	}
	w.wondersAt = map[int][]int{}
	for i, m := range w.Wonders {
		w.wondersAt[m.P] = append(w.wondersAt[m.P], i)
	}
	w.peopleByBrg = map[int][]int{}
	for i, p := range w.People {
		if p.Alive && p.Burg > 0 {
			w.peopleByBrg[p.Burg] = append(w.peopleByBrg[p.Burg], i)
		}
	}
	w.deepOpen = map[int][]bool{}
	w.deepBurg = map[int]map[int]int{}
	w.skyBurg = map[int]int{}
	w.seaBurg = map[int]int{}
	for _, L := range w.Layers {
		switch L.Kind {
		case "deep":
			w.deepOpen[L.Depth] = mask(L.Open, n)
			w.deepBurg[L.Depth] = L.Burgs
			if L.Depth > w.maxDepth {
				w.maxDepth = L.Depth
			}
		case "sky":
			w.skyOpen = mask(L.Open, n)
			w.skyBurg = L.Burgs
		case "sea":
			w.seaOpen = mask(L.Open, n)
			w.seaBurg = L.Burgs
		}
	}
	if w.skyOpen == nil {
		w.skyOpen = make([]bool, n)
	}
	if w.seaOpen == nil {
		w.seaOpen = make([]bool, n)
	}
	if w.Terms == nil {
		w.Terms = map[string]string{}
	}
	if w.LayerNames == nil {
		w.LayerNames = map[string]string{}
	}
}

func (w *WorldData) In(x, y int) bool { return x >= 0 && y >= 0 && x < w.W && y < w.H }
func (w *WorldData) Idx(x, y int) int { return y*w.W + x }

func (w *WorldData) IsWater(p int) bool { return w.Biome[p] == 0 }

func (w *WorldData) StateName(id int) string {
	if s := w.stateByID[id]; s != nil {
		return s.Name
	}
	return "the wilds"
}

func (w *WorldData) Term(k, def string) string {
	if v, ok := w.Terms[k]; ok && v != "" {
		return v
	}
	return def
}

func (w *WorldData) LayerName(k, def string) string {
	if v, ok := w.LayerNames[k]; ok && v != "" {
		return v
	}
	return def
}

// DeepOpen reports whether a Realmforge deep layer has caverns under this parasang at depth d.
func (w *WorldData) DeepOpen(p, d int) bool {
	m, ok := w.deepOpen[d]
	return ok && m[p]
}

func (w *WorldData) RegionName(p int) string {
	if p < 0 || p >= len(w.State) {
		return "the wilds"
	}
	if b := w.burgByID[w.Burg[p]]; b != nil {
		return b.Name
	}
	if s := w.stateByID[w.State[p]]; s != nil && s.ID > 0 {
		return "the lands of " + s.Name
	}
	return "the wilds"
}

func BiomeName(b int) string {
	names := []string{"open sea", "hot desert", "cold desert", "savanna", "grassland", "tropical forest", "deciduous forest", "rainforest", "temperate rainforest", "taiga", "tundra", "glacier", "wetland", "ashland", "crystal fields", "fungal forest", "lava flats"}
	if b >= 0 && b < len(names) {
		return names[b]
	}
	return "wilds"
}

func titleCase(s string) string {
	ws := strings.Fields(s)
	for i, w := range ws {
		if len(w) > 0 && w != "of" && w != "the" {
			ws[i] = strings.ToUpper(w[:1]) + w[1:]
		}
	}
	return strings.Join(ws, " ")
}
