package engine

// Trade goods table (prefix tq). Prod: the Realmforge sector whose towns make the good cheap;
// Dem: the sector whose towns pay dear for it. Perish: days until a unit may spoil (0 = never).
// Ban: the statute (crime.go uwLaws field) that makes it contraband: drugs occult heresy arms augment.
type tqGood struct {
	ID, Name string
	Base     int // value per unit, base money
	Wt       int // weight per unit
	Perish   int
	Prod     string
	Dem      string
	Ban      string
	Ws       string   // worldsim price kind that also moves it: food medicine weapons cloth ore
	Port     bool     // sea goods: cheaper in ports
	Genres   []string // nil = every genre
}

var tqGoods = []tqGood{
	{ID: "grain", Name: "sacks of grain", Base: 6, Wt: 2, Perish: 40, Prod: "farming", Dem: "military", Ws: "food", Genres: tqOld},
	{ID: "salt", Name: "salt blocks", Base: 9, Wt: 2, Prod: "mining", Dem: "seafaring", Genres: tqOld},
	{ID: "spice", Name: "bales of spice", Base: 40, Wt: 1, Perish: 90, Prod: "seafaring", Dem: "trade", Port: true, Genres: tqOld},
	{ID: "silk", Name: "bolts of silk", Base: 55, Wt: 1, Perish: 0, Prod: "crafts", Dem: "faith", Ws: "cloth", Genres: tqOld},
	{ID: "tin", Name: "tin ingots", Base: 14, Wt: 3, Prod: "mining", Dem: "crafts", Ws: "ore", Genres: tqOld},
	{ID: "iron", Name: "iron bars", Base: 16, Wt: 3, Prod: "industry", Dem: "military", Ws: "ore"},
	{ID: "timber", Name: "timber logs", Base: 7, Wt: 4, Prod: "farming", Dem: "industry"},
	{ID: "wine", Name: "casks of wine", Base: 24, Wt: 3, Perish: 120, Prod: "farming", Dem: "trade", Ban: "drugs", Genres: tqOld},
	{ID: "dye", Name: "pots of dye", Base: 30, Wt: 1, Prod: "crafts", Dem: "trade", Genres: tqOld},
	{ID: "glass", Name: "crates of glass", Base: 22, Wt: 2, Prod: "industry", Dem: "scholarship"},
	{ID: "medicine", Name: "medicine chests", Base: 45, Wt: 1, Perish: 150, Prod: "scholarship", Dem: "military", Ws: "medicine"},
	{ID: "books", Name: "crates of books", Base: 35, Wt: 2, Prod: "scholarship", Dem: "faith"},
	{ID: "tomes", Name: "forbidden tomes", Base: 80, Wt: 1, Prod: "scholarship", Dem: "faith", Ban: "heresy"},
	{ID: "tools", Name: "crates of tools", Base: 26, Wt: 3, Prod: "industry", Dem: "farming"},
	{ID: "arms", Name: "crates of weapons", Base: 60, Wt: 4, Prod: "military", Dem: "trade", Ban: "arms", Ws: "weapons"},
	{ID: "beasts", Name: "caged exotic beasts", Base: 70, Wt: 5, Perish: 14, Prod: "seafaring", Dem: "faith", Port: true, Genres: tqOld},
	{ID: "fish", Name: "salted fish", Base: 8, Wt: 2, Perish: 20, Prod: "seafaring", Dem: "farming", Ws: "food", Port: true},
	{ID: "relics", Name: "ritual ash and relics", Base: 90, Wt: 1, Prod: "faith", Dem: "scholarship", Ban: "occult"},
	{ID: "narcotic", Name: "poppy resin", Base: 65, Wt: 1, Perish: 100, Prod: "farming", Dem: "trade", Ban: "drugs"},
	// genre skins
	{ID: "scrap", Name: "scrap bundles", Base: 10, Wt: 3, Prod: "industry", Dem: "crafts", Ws: "ore", Genres: tqNew},
	{ID: "fuel", Name: "fuel drums", Base: 20, Wt: 4, Prod: "industry", Dem: "military", Genres: tqNew},
	{ID: "circuits", Name: "circuit boards", Base: 48, Wt: 1, Prod: "industry", Dem: "scholarship", Genres: tqNew},
	{ID: "vatmeat", Name: "vat-meat blocks", Base: 12, Wt: 2, Perish: 15, Prod: "farming", Dem: "trade", Ws: "food", Genres: tqNew},
	{ID: "datacore", Name: "data cores", Base: 95, Wt: 1, Prod: "scholarship", Dem: "trade", Genres: tqNew},
	{ID: "implants", Name: "crated implants", Base: 120, Wt: 1, Prod: "industry", Dem: "scholarship", Ban: "augment", Genres: tqNew},
}

var tqOld = []string{"fantasy", "mythic", "twilight", "steampunk", "solarpunk"}
var tqNew = []string{"cyberpunk", "biopunk", "atompunk", "dieselpunk", "wasteland", "steampunk", "solarpunk"}

// genre renames of a few goods
var tqSkin = map[string]map[string]string{
	"cyberpunk":  {"grain": "protein sacks", "tools": "toolkits", "medicine": "med-gel cases", "books": "data-slabs", "arms": "crated firearms", "spice": "flavour vials", "wine": "synth-liquor", "silk": "synth-fibre", "relics": "cult hardware"},
	"biopunk":    {"grain": "spore-meal sacks", "medicine": "antitoxin vials", "arms": "bone-blade crates", "wine": "ferment casks", "silk": "spun chitin"},
	"atompunk":   {"grain": "ration sacks", "medicine": "rad-away cases", "arms": "crated rifles", "wine": "moonshine kegs", "silk": "nylon bolts"},
	"dieselpunk": {"grain": "ration sacks", "arms": "crated rifles", "wine": "gin kegs", "silk": "parachute silk"},
	"wasteland":  {"grain": "ration sacks", "medicine": "stim-kit cases", "arms": "crated guns", "wine": "jugs of hooch", "silk": "salvaged cloth", "salt": "salt cakes"},
	"steampunk":  {"arms": "crated carbines", "glass": "crates of lenses"},
}

func (g *Game) tqGoodsHere() []*tqGood {
	var out []*tqGood
	for i := range tqGoods {
		d := &tqGoods[i]
		if d.Genres != nil {
			ok := false
			for _, ge := range d.Genres {
				if ge == g.Genre {
					ok = true
				}
			}
			if !ok {
				continue
			}
		}
		out = append(out, d)
	}
	return out
}

func (g *Game) tqName(d *tqGood) string {
	if m := tqSkin[g.Genre]; m != nil {
		if n, ok := m[d.ID]; ok {
			return n
		}
	}
	return d.Name
}

func tqGoodByID(id string) *tqGood {
	for i := range tqGoods {
		if tqGoods[i].ID == id {
			return &tqGoods[i]
		}
	}
	return nil
}

var tqRanks = []struct {
	Pts   int
	Title string
}{{0, "Pedlar"}, {60, "Hawker"}, {200, "Trader"}, {500, "Factor"}, {1200, "Merchant Prince"}, {3000, "Caravan Lord"}}
