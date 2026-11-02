package engine

// Plant species catalogue (prefix fl). Each entry sits on one terrain slot (class), fits biomes,
// a base-temperature band (°C, the parasang's `Temp`) and an elevation band (`Height`, sea 20,
// mountains 60+). Biomes: 1 hot desert, 2 cold desert, 3 savanna, 4 grassland, 5 tropical seasonal,
// 6 temperate deciduous, 7 tropical rainforest, 8 temperate rainforest, 9 taiga, 10 tundra,
// 11 glacier, 12 wetland, 13 ashland, 14 crystal, 15 fungal, 16 lava.

// plant classes = the terrain slot a species grows on
const (
	flNone     uint8 = iota
	flTree           // TTree
	flSnag           // TDeadTree: cacti, dead snags, ash-trees
	flShrub          // TBush
	flGrass          // TTallGrass (and a few TGrass)
	flFlower         // sparse on TGrass / TTallGrass
	flReed           // TReeds
	flMoss           // TMoss
	flGiant          // TMushroom
	flMat            // TFungus (and forest-floor mushrooms on grass/moss)
	flKelp           // TKelp
	flLily           // fresh TShallow
	flSeagrass       // TShallow in the sea
	flCrop           // TCrop
	flCrystal        // TCrystal
	flNClass
)

var flClassName = [flNClass]string{"", "tree", "tree", "shrub", "grass", "flower", "reed", "moss", "giant fungus", "fungus", "seaweed", "water plant", "seagrass", "crop", "crystal growth"}

// season bits
const (
	sSp uint8 = 1 << iota
	sSu
	sAu
	sWi
	sAll = sSp | sSu | sAu | sWi
)

// layers
const (
	flSurf uint8 = iota
	flDeep
	flSky
	flSea
)

type flSp struct {
	Name  string
	Cl    uint8
	G     rune
	F     byte
	Bio   []int
	TLo   int
	THi   int
	HLo   int
	HHi   int
	Lay   uint8
	Gen   []string
	Wt    int
	Decid bool  // drops its leaves in winter
	Aut   byte  // autumn colour (0 = none)
	Blo   byte  // blossom / flower colour
	Bloom uint8 // seasons in flower
	Size  uint8 // 0 low, 1 small, 2 tall, 3 giant
	Desc  string
	Wood  string
	WoodV int
	Prod  string
	PTag  string
	PSeas uint8
	PN    int
}

func flS(cl uint8, name string, g rune, f byte, desc string) flSp {
	sz := uint8(1)
	if cl == flTree || cl == flSnag || cl == flGiant || cl == flCrystal {
		sz = 2
	}
	return flSp{Name: name, Cl: cl, G: g, F: f, TLo: -99, THi: 99, HLo: 0, HHi: 100, Wt: 4, Size: sz, Desc: desc}
}
func (s flSp) b(bio ...int) flSp     { s.Bio = bio; return s }
func (s flSp) t(lo, hi int) flSp     { s.TLo, s.THi = lo, hi; return s }
func (s flSp) h(lo, hi int) flSp     { s.HLo, s.HHi = lo, hi; return s }
func (s flSp) w(n int) flSp          { s.Wt = n; return s }
func (s flSp) dec(aut byte) flSp     { s.Decid, s.Aut = true, aut; return s }
func (s flSp) aut(c byte) flSp       { s.Aut = c; return s }
func (s flSp) big(n uint8) flSp      { s.Size = n; return s }
func (s flSp) lay(l uint8) flSp      { s.Lay = l; return s }
func (s flSp) only(g ...string) flSp { s.Gen = g; return s }
func (s flSp) blo(c byte, seas uint8) flSp {
	s.Blo, s.Bloom = c, seas
	return s
}
func (s flSp) wood(name string, v int) flSp { s.Wood, s.WoodV = name, v; return s }
func (s flSp) fruit(name, tag string, seas uint8, n int) flSp {
	s.Prod, s.PTag, s.PSeas, s.PN = name, tag, seas, n
	return s
}

const (
	flGBroad  = '♣'
	flGConif  = '↑'
	flGPalm   = 'Γ'
	flGCactus = 'φ'
	flGBamboo = '‼'
	flGShrub  = '♠'
	flGGrass  = '"'
	flGFlower = '°'
	flGMoss   = ','
	flGSnag   = 'τ'
	flGCrys   = '*'
)

var flCat = []flSp{
	// ---------------- temperate broadleaf trees ----------------
	flS(flTree, "sessile oak", flGBroad, CGreen, "its bark deeply furrowed").b(6, 8, 4).t(4, 22).h(0, 70).dec(CBrown).w(8).wood("oak", 14).fruit("acorns", "lasting", sAu, 3),
	flS(flTree, "pedunculate oak", flGBroad, CDarkGreen, "broad-crowned, its lower boughs spreading low").b(6, 4).t(4, 22).h(0, 60).dec(CBrown).w(6).wood("oak", 14).fruit("acorns", "lasting", sAu, 3),
	flS(flTree, "copper beech", flGBroad, CDarkRed, "smooth grey trunk under purple leaves").b(6).t(4, 20).h(0, 65).dec(COrange).w(3).wood("beech", 10).fruit("beechnuts", "hearty", sAu, 2),
	flS(flTree, "common beech", flGBroad, CGreen, "its smooth grey bark carved with old initials").b(6, 8).t(4, 20).h(0, 70).dec(COrange).w(7).wood("beech", 10).fruit("beechnuts", "hearty", sAu, 2),
	flS(flTree, "sugar maple", flGBroad, CGreen, "five-pointed leaves, sap rising in its veins").b(6, 9).t(2, 18).h(0, 70).dec(CRed).w(6).wood("maple", 11).fruit("maple sap", "mending", sSp, 1),
	flS(flTree, "sycamore maple", flGBroad, CDarkGreen, "winged seeds twirling down from it").b(6, 8).t(4, 20).h(0, 70).dec(CYellow).w(4).wood("maple", 10),
	flS(flTree, "red maple", flGBroad, CGreen, "red-stemmed, quick to colour").b(6, 12).t(4, 22).h(0, 60).dec(CRed).w(3).wood("maple", 9),
	flS(flTree, "silver birch", flGBroad, CGreen, "white bark peeling in papery strips").b(6, 9, 4).t(-4, 18).h(0, 75).dec(CYellow).w(7).wood("birch", 7).fruit("birch sap", "spry", sSp, 1),
	flS(flTree, "paper birch", flGBroad, CDarkGreen, "chalk-white, ringed with black scars").b(9, 6, 10).t(-8, 14).h(0, 80).dec(CYellow).w(5).wood("birch", 7),
	flS(flTree, "quaking aspen", flGBroad, CGreen, "its leaves trembling in the faintest breeze").b(9, 6, 4).t(-6, 18).h(20, 85).dec(CYellow).w(5).wood("aspen", 5),
	flS(flTree, "black alder", flGBroad, CDarkGreen, "dark-barked, its roots in wet ground").b(12, 6, 8).t(0, 22).h(0, 50).dec(CBrown).w(5).wood("alder", 6),
	flS(flTree, "weeping willow", flGBroad, CGreen, "its long fronds trailing to the ground").b(12, 6, 4).t(4, 24).h(0, 45).dec(CYellow).w(5).wood("willow", 5).fruit("willow bark", "mending", sAll, 1),
	flS(flTree, "crack willow", flGBroad, CDarkGreen, "split and leaning, sprouting anew from the break").b(12, 6).t(2, 22).h(0, 50).dec(CYellow).w(3).wood("willow", 5),
	flS(flTree, "black poplar", flGBroad, CGreen, "tall and ragged, shedding cotton seed").b(6, 4, 12).t(4, 24).h(0, 50).dec(CYellow).w(4).wood("poplar", 5),
	flS(flTree, "eastern cottonwood", flGBroad, CGreen, "a river tree, its leaves rattling").b(4, 2, 12).t(2, 24).h(0, 55).dec(CYellow).w(5).wood("cottonwood", 4),
	flS(flTree, "common ash", flGBroad, CGreen, "grey-barked, black buds on its twigs").b(6, 8).t(2, 20).h(0, 65).dec(CYellow).w(5).wood("ash", 12),
	flS(flTree, "small-leaved linden", flGBroad, CGreen, "heart-shaped leaves, humming with bees").b(6).t(2, 20).h(0, 60).dec(CYellow).blo(CYellow, sSu).w(4).wood("linden", 8).fruit("linden blossom", "mending", sSu, 2),
	flS(flTree, "sweet chestnut", flGBroad, CDarkGreen, "its bark twisted in a slow spiral").b(6, 4).t(8, 22).h(0, 60).dec(CYellow).w(4).wood("chestnut", 10).fruit("chestnuts", "hearty", sAu, 3),
	flS(flTree, "horse chestnut", flGBroad, CGreen, "candles of white blossom on every bough").b(6).t(4, 20).h(0, 55).dec(COrange).blo(CWhite, sSp).w(3).wood("chestnut", 6),
	flS(flTree, "common walnut", flGBroad, CGreen, "broad and aromatic, its leaves bitter").b(6, 4).t(8, 22).h(0, 60).dec(CBrown).w(3).wood("walnut", 16).fruit("walnuts", "keen", sAu, 3),
	flS(flTree, "black walnut", flGBroad, CDarkGreen, "dark-barked, little grows beneath it").b(6, 4).t(6, 22).h(0, 55).dec(CYellow).w(2).wood("walnut", 18).fruit("black walnuts", "keen", sAu, 2),
	flS(flTree, "wild cherry", flGBroad, CGreen, "its bark banded and glossy").b(6, 4).t(4, 20).h(0, 60).dec(CRed).blo(CWhite, sSp).w(3).wood("cherry", 13).fruit("wild cherries", "spry", sSu, 3),
	flS(flTree, "crab apple", flGBroad, CGreen, "gnarled and low, knobbed with spurs").b(6, 4).t(2, 20).h(0, 60).dec(CYellow).blo(CWhite, sSp).w(3).wood("applewood", 9).fruit("crab apples", "spry", sAu, 3),
	flS(flTree, "wild pear", flGBroad, CDarkGreen, "tall and narrow, thorned on its spurs").b(6, 4).t(6, 22).h(0, 55).dec(CRed).blo(CWhite, sSp).w(2).wood("pearwood", 11).fruit("wild pears", "spry", sAu, 2),
	flS(flTree, "rowan", flGBroad, CGreen, "slender, hung with scarlet berries").b(6, 9, 8).t(-4, 18).h(20, 90).dec(CRed).w(4).wood("rowan", 6).fruit("rowan berries", "rime", sAu, 3),
	flS(flTree, "hornbeam", flGBroad, CGreen, "its trunk fluted like a muscle").b(6).t(4, 20).h(0, 55).dec(CYellow).w(3).wood("hornbeam", 12),
	flS(flTree, "field elm", flGBroad, CDarkGreen, "rough-leaved, suckering along the hedges").b(6, 4).t(4, 22).h(0, 55).dec(CYellow).w(3).wood("elm", 9),
	flS(flTree, "holly", flGBroad, CDarkGreen, "glossy and spined, berried red").b(6, 8).t(4, 20).h(0, 60).w(3).wood("holly", 9).fruit("holly berries", "venom", sWi, 2),
	flS(flTree, "southern magnolia", flGBroad, CDarkGreen, "leathery leaves, plate-sized white blooms").b(6, 5).t(14, 26).h(0, 45).blo(CWhite, sSu).w(2).wood("magnolia", 8),
	flS(flTree, "tulip tree", flGBroad, CGreen, "ramrod straight, its leaves square-tipped").b(6).t(8, 22).h(0, 55).dec(CYellow).blo(COrange, sSu).w(2).wood("tulipwood", 7),
	flS(flTree, "sweetgum", flGBroad, CGreen, "star-shaped leaves, spiky seed balls beneath").b(6, 12).t(10, 24).h(0, 50).dec(CRed).w(2).wood("sweetgum", 7).fruit("sweetgum resin", "mending", sSu, 1),
	flS(flTree, "shagbark hickory", flGBroad, CDarkGreen, "its bark curling off in long plates").b(6, 4).t(6, 22).h(0, 55).dec(CYellow).w(2).wood("hickory", 15).fruit("hickory nuts", "hearty", sAu, 2),
	flS(flTree, "ginkgo", flGBroad, CGreen, "fan-shaped leaves on an ancient lineage").b(6, 8).t(6, 22).h(0, 60).dec(CYellow).w(1).wood("ginkgo", 12).fruit("ginkgo nuts", "keen", sAu, 2),
	flS(flTree, "bur oak", flGBroad, CDarkGreen, "corky-barked, standing alone in the grass").b(4, 6).t(2, 22).h(0, 60).dec(CBrown).w(5).wood("oak", 14).fruit("acorns", "lasting", sAu, 3),
	flS(flTree, "honey locust", flGBroad, CGreen, "its trunk bristling with branched thorns").b(4, 6).t(6, 24).h(0, 55).dec(CYellow).w(2).wood("locust", 11).fruit("locust pods", "spry", sAu, 2),
	flS(flTree, "osage orange", flGBroad, CDarkGreen, "thorny, dropping green brain-like fruit").b(4).t(10, 24).h(0, 50).dec(CYellow).w(2).wood("osage", 15),
	flS(flTree, "hackberry", flGBroad, CGreen, "warty-barked, small purple fruit").b(4, 6).t(6, 24).h(0, 55).dec(CYellow).w(2).wood("hackberry", 6).fruit("hackberries", "spry", sAu, 2),
	flS(flTree, "persimmon", flGBroad, CGreen, "blocky bark, orange fruit hanging after the leaves fall").b(6, 5).t(12, 26).h(0, 50).dec(COrange).w(2).wood("persimmon", 13).fruit("persimmons", "spry", sAu, 2),
	flS(flTree, "Japanese cherry", flGBroad, CGreen, "trained and graceful").b(6, 8).t(6, 20).h(0, 60).dec(CRed).blo(CMagenta, sSp).w(2).wood("cherry", 13),
	flS(flTree, "almond", flGBroad, CGreen, "its pale blossom out before the leaves").b(4, 6, 2).t(14, 26).h(0, 60).dec(CYellow).blo(CWhite, sSp).w(2).wood("almond", 12).fruit("almonds", "hearty", sSu, 3),
	// ---------------- Mediterranean / warm dry ----------------
	flS(flTree, "olive", flGBroad, CGray, "grey-green leaves on a hollow, twisted trunk").b(4, 6, 1, 2).t(14, 28).h(0, 60).w(6).wood("olive", 15).fruit("olives", "thick", sAu, 3),
	flS(flTree, "common fig", flGBroad, CGreen, "broad lobed leaves, milky sap in the stems").b(4, 6, 1, 5).t(14, 30).h(0, 55).dec(CYellow).w(4).wood("fig", 5).fruit("figs", "spry", sSu, 3),
	flS(flTree, "cork oak", flGBroad, CDarkGreen, "its thick bark stripped once a decade").b(4, 6).t(14, 26).h(0, 55).w(3).wood("cork oak", 9).fruit("cork bark", "thick", sAll, 1),
	flS(flTree, "holm oak", flGBroad, CDarkGreen, "dark evergreen, holly-leaved").b(4, 6, 2).t(12, 26).h(0, 65).w(4).wood("oak", 14).fruit("sweet acorns", "lasting", sAu, 3),
	flS(flTree, "carob", flGBroad, CDarkGreen, "glossy leaves, long dark pods").b(4, 1, 3).t(16, 30).h(0, 50).w(2).wood("carob", 10).fruit("carob pods", "lasting", sAu, 3),
	flS(flTree, "pomegranate", flGBroad, CGreen, "small and spiny, flowering scarlet").b(4, 1, 2, 5).t(16, 30).h(0, 55).dec(CYellow).blo(CRed, sSu).w(2).wood("pomegranate", 8).fruit("pomegranates", "mending", sAu, 2),
	flS(flTree, "Italian cypress", flGConif, CDarkGreen, "a dark flame of a tree, pointing at the sky").b(4, 6, 1).t(14, 28).h(0, 60).w(4).wood("cypress", 12),
	flS(flTree, "stone pine", flGConif, CDarkGreen, "an umbrella of needles on a bare trunk").b(4, 6, 1).t(14, 28).h(0, 55).w(4).wood("pine", 7).fruit("pine nuts", "hearty", sAu, 2),
	flS(flTree, "Aleppo pine", flGConif, CGreen, "sparse, leaning, smelling of resin in the heat").b(4, 1, 2).t(14, 30).h(0, 60).w(3).wood("pine", 6).fruit("pine resin", "mending", sSu, 1),
	flS(flTree, "cedar of Lebanon", flGConif, CDarkGreen, "its boughs laid out in flat, layered tiers").b(6, 2, 4).t(6, 22).h(40, 85).w(4).wood("cedar", 18).fruit("cedar resin", "mending", sSu, 1),
	flS(flTree, "Atlas cedar", flGConif, CCyan, "blue-needled, broad and stately").b(2, 6, 4).t(6, 22).h(45, 85).w(3).wood("cedar", 17),
	flS(flTree, "laurel", flGBroad, CDarkGreen, "aromatic leaves, the victor's wreath").b(4, 6).t(12, 26).h(0, 55).w(2).wood("laurel", 8).fruit("bay leaves", "keen", sAll, 2),
	// ---------------- conifers: boreal, montane, rainforest ----------------
	flS(flTree, "Scots pine", flGConif, CDarkGreen, "its upper trunk flaking orange").b(9, 6, 4).t(-8, 18).h(0, 80).w(8).wood("pine", 7).fruit("pine resin", "mending", sSu, 1),
	flS(flTree, "Norway spruce", flGConif, CDarkGreen, "a dark spire hung with long cones").b(9, 6, 8).t(-10, 16).h(0, 85).w(8).wood("spruce", 6).fruit("spruce tips", "rime", sSp, 2),
	flS(flTree, "black spruce", flGConif, CDarkGray, "thin and ragged, a bog tree").b(9, 12, 10).t(-14, 8).h(0, 70).w(5).wood("spruce", 5),
	flS(flTree, "white spruce", flGConif, CDarkGreen, "dense and conical, smelling of cat").b(9).t(-14, 10).h(0, 80).w(4).wood("spruce", 6),
	flS(flTree, "balsam fir", flGConif, CDarkGreen, "blistered with fragrant resin").b(9, 6).t(-10, 14).h(0, 80).w(6).wood("fir", 6).fruit("fir balsam", "mending", sSu, 1),
	flS(flTree, "silver fir", flGConif, CGreen, "needles white-striped beneath").b(6, 8, 9).t(-4, 16).h(30, 85).w(4).wood("fir", 7),
	flS(flTree, "Siberian larch", flGConif, CGreen, "a conifer that turns gold and drops its needles").b(9, 10).t(-16, 10).h(0, 85).dec(CYellow).w(5).wood("larch", 10),
	flS(flTree, "tamarack", flGConif, CGreen, "a bog larch, soft-needled").b(9, 12).t(-14, 10).h(0, 60).dec(CYellow).w(3).wood("larch", 9),
	flS(flTree, "European larch", flGConif, CGreen, "soft tufts of needles, rose-red young cones").b(6, 9).t(-6, 16).h(35, 90).dec(COrange).w(3).wood("larch", 10),
	flS(flTree, "Siberian stone pine", flGConif, CDarkGreen, "five needles to a bundle, cones full of nuts").b(9).t(-16, 8).h(0, 85).w(3).wood("pine", 8).fruit("cedar nuts", "hearty", sAu, 3),
	flS(flTree, "Swiss stone pine", flGConif, CDarkGreen, "wind-shaped, clinging to the high slopes").b(9, 6, 2, 10).t(-12, 14).h(60, 100).w(3).wood("pine", 9).fruit("pine nuts", "hearty", sAu, 2),
	flS(flTree, "mountain pine", flGConif, CDarkGreen, "low and many-stemmed above the treeline").b(9, 6, 10, 2).t(-14, 14).h(62, 100).w(3).wood("pine", 6),
	flS(flTree, "bristlecone pine", flGConif, CBrown, "half-dead and thousands of years old, twisted by wind").b(2, 10, 9).t(-12, 14).h(65, 100).w(2).big(1).wood("bristlecone", 20),
	flS(flTree, "krummholz spruce", flGConif, CDarkGray, "stunted and flagged, bent flat by the wind").b(10, 9).t(-20, 6).h(0, 100).w(4).big(1).wood("spruce", 4),
	flS(flTree, "Douglas fir", flGConif, CDarkGreen, "thick-barked, cones with three-pointed bracts").b(8, 6).t(2, 18).h(0, 85).w(7).wood("fir", 11),
	flS(flTree, "western red cedar", flGConif, CGreen, "fibrous bark, scale-leaved and fragrant").b(8).t(2, 18).h(0, 75).w(6).wood("cedar", 14),
	flS(flTree, "western hemlock", flGConif, CDarkGreen, "its leading shoot drooping").b(8, 9).t(0, 16).h(0, 75).w(5).wood("hemlock", 6),
	flS(flTree, "mountain hemlock", flGConif, CDarkCyan, "blue-green and shaggy near the snows").b(8, 9).t(-6, 12).h(55, 95).w(3).wood("hemlock", 6),
	flS(flTree, "coast redwood", flGConif, CDarkGreen, "its red trunk rising beyond sight into fog").b(8).t(6, 18).h(0, 45).w(4).big(3).wood("redwood", 16),
	flS(flTree, "giant sequoia", flGConif, CDarkGreen, "a trunk wider than a house, its bark soft and red").b(8, 6).t(2, 18).h(40, 75).w(2).big(3).wood("sequoia", 16),
	flS(flTree, "monkey puzzle", flGConif, CDarkGreen, "armoured in hard, spiralled scales").b(8, 6).t(2, 18).h(35, 85).w(2).wood("araucaria", 12).fruit("piñones", "hearty", sAu, 2),
	flS(flTree, "European yew", flGConif, CDarkGreen, "dark and ancient, every part of it poison").b(6, 8).t(2, 20).h(0, 70).w(3).wood("yew", 17).fruit("yew berries", "venom", sAu, 2),
	flS(flTree, "Japanese black pine", flGConif, CDarkGreen, "salt-hardy, leaning seaward").b(6, 8, 5).t(8, 22).h(0, 40).w(2).wood("pine", 7),
	flS(flTree, "eastern white pine", flGConif, CGreen, "soft long needles in fives").b(6, 9).t(0, 18).h(0, 70).w(4).wood("pine", 8),
	flS(flTree, "Rocky juniper", flGConif, CCyan, "shaggy-barked, berried blue").b(2, 4, 1).t(0, 24).h(25, 85).w(4).big(1).wood("juniper", 12).fruit("juniper berries", "keen", sAu, 2),
	flS(flTree, "bald cypress", flGConif, CGreen, "standing in black water on buttressed knees").b(12, 6).t(10, 26).h(0, 30).dec(COrange).w(6).wood("cypress", 12),
	// ---------------- desert ----------------
	flS(flTree, "date palm", flGPalm, CGreen, "its crown of fronds over a scaly trunk").b(1, 3).t(18, 40).h(0, 50).w(8).wood("palm", 4).fruit("dates", "lasting", sAu, 4),
	flS(flTree, "doum palm", flGPalm, CDarkGreen, "a palm that forks, and forks again").b(1, 3).t(20, 40).h(0, 45).w(3).wood("palm", 4).fruit("doum nuts", "lasting", sAu, 2),
	flS(flTree, "desert ironwood", flGBroad, CGray, "its wood so dense it sinks").b(1).t(18, 40).h(0, 55).w(4).big(1).wood("ironwood", 16),
	flS(flTree, "honey mesquite", flGBroad, CGreen, "thorny and deep-rooted, hung with sweet pods").b(1, 2, 3).t(14, 38).h(0, 60).w(6).big(1).wood("mesquite", 10).fruit("mesquite pods", "lasting", sSu, 3),
	flS(flTree, "blue palo verde", flGBroad, CGreen, "green-barked, leafless most of the year").b(1).t(18, 40).h(0, 55).blo(CYellow, sSp).w(4).big(1).wood("palo verde", 5),
	flS(flTree, "tamarisk", flGBroad, CGray, "feathery and salt-crusted").b(1, 2, 12).t(10, 38).h(0, 50).blo(CMagenta, sSp).w(4).big(1).wood("tamarisk", 6),
	flS(flTree, "argan", flGBroad, CDarkGreen, "thorny and low, goats standing in its branches").b(1, 4).t(18, 36).h(20, 55).w(2).big(1).wood("argan", 12).fruit("argan kernels", "thick", sSu, 2),
	flS(flTree, "frankincense tree", flGBroad, CGray, "papery bark weeping white tears of resin").b(1, 3).t(20, 40).h(25, 65).w(2).big(1).wood("boswellia", 8).fruit("frankincense", "keen", sAll, 1),
	flS(flTree, "myrrh tree", flGBroad, CBrown, "spiny and stunted, oozing red-brown gum").b(1, 3).t(20, 40).h(20, 60).w(2).big(1).wood("commiphora", 6).fruit("myrrh", "mending", sAll, 1),
	flS(flTree, "dragon blood tree", flGBroad, CDarkGreen, "an upturned umbrella of branches, bleeding red resin").b(1, 2, 3).t(16, 36).h(30, 75).w(2).wood("dragonwood", 14).fruit("dragon's blood resin", "ember", sAll, 1),
	// ---------------- savanna ----------------
	flS(flTree, "baobab", flGBroad, CBrown, "a vast bottle-trunk, branches like roots in the air").b(3, 1, 5).t(20, 40).h(0, 50).dec(CYellow).w(4).big(3).wood("baobab", 3).fruit("baobab fruit", "spry", sAu, 3),
	flS(flTree, "umbrella thorn acacia", flGBroad, CGreen, "flat-topped, every twig set with paired thorns").b(3, 1).t(18, 40).h(0, 60).w(9).wood("acacia", 11).fruit("acacia gum", "mending", sSu, 1),
	flS(flTree, "fever tree", flGBroad, CYellow, "powdery yellow bark by the waterholes").b(3, 12).t(20, 38).h(0, 45).w(3).wood("acacia", 9),
	flS(flTree, "whistling thorn", flGBroad, CDarkGreen, "swollen thorns that hum when the wind blows").b(3).t(20, 38).h(0, 55).w(3).big(1).wood("acacia", 8),
	flS(flTree, "marula", flGBroad, CGreen, "dropping yellow fruit the elephants love").b(3).t(20, 38).h(0, 55).dec(CYellow).w(4).wood("marula", 7).fruit("marula fruit", "spry", sSu, 3),
	flS(flTree, "shea tree", flGBroad, CDarkGreen, "cork-barked, its nuts rich with fat").b(3, 5).t(22, 38).h(0, 50).w(4).wood("shea", 9).fruit("shea nuts", "thick", sSu, 3),
	flS(flTree, "sausage tree", flGBroad, CGreen, "hung with heavy grey fruit on long cords").b(3, 5).t(20, 38).h(0, 50).blo(CDarkRed, sSp).w(2).wood("kigelia", 7),
	flS(flTree, "mopane", flGBroad, CGreen, "butterfly-shaped leaves folding in the heat").b(3).t(22, 40).h(0, 50).dec(COrange).w(4).wood("mopane", 15),
	flS(flTree, "jackalberry", flGBroad, CDarkGreen, "a termite mound wrapped about its roots").b(3, 5).t(20, 38).h(0, 50).w(2).wood("ebony", 18).fruit("jackalberries", "spry", sWi, 2),
	flS(flTree, "eucalyptus", flGBroad, CCyan, "its bark peeling in coloured ribbons, the air smelling of menthol").b(3, 4, 5, 1).t(12, 34).h(0, 70).w(5).wood("eucalyptus", 9).fruit("eucalyptus leaves", "mending", sAll, 2),
	flS(flTree, "river red gum", flGBroad, CDarkCyan, "white-trunked, a widow-maker dropping limbs").b(3, 12, 4).t(14, 34).h(0, 45).w(3).wood("red gum", 12),
	// ---------------- tropical seasonal ----------------
	flS(flTree, "teak", flGBroad, CGreen, "huge rough leaves, wood that never rots").b(5).t(20, 36).h(0, 55).dec(CBrown).w(6).wood("teak", 18),
	flS(flTree, "sal", flGBroad, CDarkGreen, "tall and straight, the forest's pillar").b(5).t(20, 34).h(0, 60).w(5).wood("sal", 14).fruit("sal resin", "keen", sSu, 1),
	flS(flTree, "neem", flGBroad, CGreen, "bitter-leaved, a village medicine tree").b(5, 3, 1).t(20, 38).h(0, 55).w(4).wood("neem", 8).fruit("neem leaves", "mending", sAll, 2),
	flS(flTree, "mango", flGBroad, CDarkGreen, "dense and dark, heavy with fruit in the hot months").b(5, 7).t(20, 36).h(0, 50).blo(CYellow, sSp).w(5).wood("mango", 7).fruit("mangoes", "spry", sSu, 3),
	flS(flTree, "tamarind", flGBroad, CGreen, "feathery leaves, brown pods sour and sticky").b(5, 3).t(20, 38).h(0, 50).w(3).wood("tamarind", 12).fruit("tamarind pods", "keen", sWi, 3),
	flS(flTree, "Indian rosewood", flGBroad, CGreen, "its heartwood dark and scented").b(5).t(20, 36).h(0, 55).w(3).wood("rosewood", 20),
	flS(flTree, "sandalwood", flGBroad, CDarkGreen, "a small half-parasite, its wood fragrant for centuries").b(5, 3).t(20, 36).h(20, 60).w(2).big(1).wood("sandalwood", 24),
	flS(flTree, "banyan", flGBroad, CDarkGreen, "a whole grove grown from one tree, its aerial roots become trunks").b(5, 7).t(20, 36).h(0, 50).w(3).big(3).wood("banyan", 5).fruit("banyan figs", "spry", sSu, 2),
	flS(flTree, "sacred fig", flGBroad, CGreen, "heart-shaped leaves with long drip tips").b(5, 7).t(20, 36).h(0, 55).w(3).wood("fig", 6),
	flS(flTree, "flame tree", flGBroad, CGreen, "ablaze with scarlet blossom before the rains").b(5, 3, 7).t(20, 36).h(0, 50).dec(CYellow).blo(CRed, sSp).w(3).wood("gulmohar", 7),
	flS(flTree, "jacaranda", flGBroad, CGreen, "a haze of fern-like leaves").b(5, 3).t(18, 34).h(0, 60).blo(CBlue, sSp).w(2).wood("jacaranda", 9),
	flS(flTree, "cashew", flGBroad, CGreen, "each nut hanging under a swollen red apple").b(5, 3).t(22, 38).h(0, 45).w(2).wood("cashew", 6).fruit("cashew apples", "spry", sSp, 2),
	flS(flTree, "ebony", flGBroad, CDarkGreen, "its heartwood black as pitch").b(5, 7).t(22, 36).h(0, 50).w(2).wood("ebony", 22),
	flS(flTree, "giant bamboo", flGBamboo, CGreen, "a forest of jointed green stems, creaking").b(5, 7, 8).t(14, 34).h(0, 70).w(5).wood("bamboo", 4).fruit("bamboo shoots", "spry", sSp, 2),
	flS(flTree, "black bamboo", flGBamboo, CDarkGray, "its canes darkening to ink").b(5, 8, 6).t(10, 28).h(0, 65).w(2).wood("bamboo", 6),
	flS(flTree, "cycad", flGPalm, CDarkGreen, "stiff, ancient fronds on a stout trunk").b(5, 7, 3).t(20, 38).h(0, 55).w(2).big(1).wood("cycad", 3).fruit("cycad seeds", "venom", sAu, 2),
	// ---------------- tropical rainforest ----------------
	flS(flTree, "kapok", flGBroad, CGreen, "an emergent giant on plank buttresses, pods of silk cotton").b(7, 5).t(22, 38).h(0, 50).w(4).big(3).wood("kapok", 3).fruit("kapok floss", "lasting", sSp, 2),
	flS(flTree, "big-leaf mahogany", flGBroad, CDarkGreen, "red-barked, its seeds spinning down").b(7, 5).t(22, 36).h(0, 50).w(5).wood("mahogany", 20),
	flS(flTree, "rubber tree", flGBroad, CDarkGreen, "scored with old tapping cuts").b(7).t(22, 36).h(0, 45).w(4).wood("rubberwood", 6).fruit("latex", "thick", sAll, 1),
	flS(flTree, "strangler fig", flGBroad, CGreen, "a lattice of roots wrapped about a dead host").b(7, 5).t(22, 36).h(0, 50).w(5).big(3).wood("fig", 5).fruit("wild figs", "spry", sAll, 2),
	flS(flTree, "Brazil nut", flGBroad, CDarkGreen, "towering, dropping cannonball pods").b(7).t(22, 36).h(0, 40).w(3).big(3).wood("castanha", 12).fruit("brazil nuts", "hearty", sWi, 3),
	flS(flTree, "cacao", flGBroad, CDarkGreen, "pods sprouting straight from its trunk").b(7).t(22, 36).h(0, 45).w(3).big(1).wood("cacao", 5).fruit("cacao pods", "keen", sAu, 2),
	flS(flTree, "breadfruit", flGBroad, CGreen, "huge lobed leaves, starchy green fruit").b(7, 5).t(22, 36).h(0, 40).w(3).wood("breadfruit", 6).fruit("breadfruit", "lasting", sSu, 2),
	flS(flTree, "durian", flGBroad, CGreen, "spiked fruit that reek like a sewer").b(7).t(24, 36).h(0, 45).w(2).wood("durian", 7).fruit("durian", "hearty", sSu, 1),
	flS(flTree, "ironwood", flGBroad, CDarkGreen, "buttressed and dark, blunting every blade").b(7).t(22, 36).h(0, 50).w(3).wood("ironwood", 17),
	flS(flTree, "tree fern", flGPalm, CGreen, "a fern raised on a shaggy trunk").b(7, 8, 5).t(10, 32).h(0, 75).w(5).big(1).wood("fern trunk", 3).fruit("fern fiddleheads", "spry", sSp, 2),
	flS(flTree, "coconut palm", flGPalm, CGreen, "leaning seaward, nuts clustered under its fronds").b(7, 5, 3, 12).t(22, 40).h(0, 28).w(6).wood("coconut wood", 5).fruit("coconuts", "lasting", sAll, 2),
	flS(flTree, "oil palm", flGPalm, CDarkGreen, "spiny fronds over bunches of red fruit").b(7, 5).t(22, 38).h(0, 40).w(3).wood("palm", 4).fruit("palm fruit", "thick", sAll, 2),
	flS(flTree, "açaí palm", flGPalm, CGreen, "slender stems in clumps, purple berries").b(7, 12).t(22, 38).h(0, 35).w(3).wood("palm", 3).fruit("açaí berries", "spry", sSu, 3),
	flS(flTree, "sago palm", flGPalm, CDarkGreen, "a swamp palm, its pith full of starch").b(12, 7).t(22, 38).h(0, 28).w(3).wood("palm", 3).fruit("sago pith", "lasting", sAll, 2),
	flS(flTree, "nipa palm", flGPalm, CGreen, "fronds straight from the mud").b(12, 7, 5).t(22, 38).h(0, 24).w(3).big(1).wood("palm", 2).fruit("nipa sap", "spry", sAll, 1),
	flS(flTree, "red mangrove", flGBroad, CDarkGreen, "standing above the tide on arching stilt roots").b(12, 7, 5).t(20, 40).h(0, 24).w(7).wood("mangrove", 9),
	flS(flTree, "black mangrove", flGBroad, CDarkGreen, "breathing roots poking up from the mud like spikes").b(12, 7, 5, 1).t(18, 40).h(0, 24).w(4).wood("mangrove", 9),
	flS(flTree, "swamp tupelo", flGBroad, CGreen, "swollen-based in the black water").b(12).t(10, 28).h(0, 30).dec(CRed).w(3).wood("tupelo", 6),
	flS(flTree, "paperbark", flGBroad, CGray, "wrapped in layer on layer of soft bark").b(12, 3).t(16, 36).h(0, 30).blo(CWhite, sSp).w(3).wood("melaleuca", 8),
	// ---------------- exotic biomes ----------------
	flS(flTree, "cinder oak", flGBroad, CDarkRed, "black-barked, its leaves glowing like coals at the veins").b(13, 16).w(5).wood("cinderwood", 14).fruit("ember-acorns", "ember", sAu, 2),
	flS(flTree, "ashbark pine", flGConif, CDarkGray, "grey needles that sift ash with every gust").b(13).w(5).wood("ashbark", 8).fruit("smoke-resin", "ember", sSu, 1),
	flS(flTree, "smoke willow", flGBroad, CGray, "its fronds trailing like drifting smoke").b(13, 12).w(3).wood("smokewood", 6),
	flS(flTree, "glass birch", flGBroad, CCyan, "its bark gone to clear crystal").b(14).w(4).wood("glasswood", 16),
	flS(flTree, "spore oak", flGBroad, CDarkMagenta, "lumpy with shelf fungi, breathing out spores").b(15).w(3).wood("sporewood", 6),
	// ---------------- sky isles ----------------
	flS(flTree, "cloud pine", flGConif, CCyan, "its needles beaded with cloud").lay(flSky).w(6).wood("cloud pine", 10).fruit("sky resin", "spry", sSu, 1),
	flS(flTree, "wind-harp ash", flGBroad, CWhite, "hollow-twigged, sounding chords in the gale").lay(flSky).w(4).wood("harpwood", 14),
	flS(flTree, "sky willow", flGBroad, CCyan, "its fronds streaming upward on the updrafts").lay(flSky).w(4).wood("willow", 6),
	flS(flTree, "stormbark oak", flGBroad, CDarkCyan, "lightning-scarred a hundred times").lay(flSky).w(3).wood("stormoak", 16).fruit("charged acorns", "keen", sAu, 2),
	// ---------------- genre trees ----------------
	flS(flTree, "rust tree", flGBroad, CBrown, "its bark flaking like corroded iron").only("wasteland", "dieselpunk").w(5).wood("rustwood", 6),
	flS(flTree, "soot poplar", flGBroad, CDarkGray, "black-leaved from the stacks upwind").only("steampunk", "dieselpunk").w(4).wood("poplar", 5),
	flS(flTree, "isotope elm", flGBroad, CCyan, "it glows faintly after dusk").only("atompunk").w(4).wood("glowwood", 9).fruit("hot seeds", "ember", sAu, 1),
	flS(flTree, "lumen palm", flGPalm, CCyan, "fronds spliced with bioluminescent genes").only("cyberpunk", "biopunk").w(4).wood("palm", 5),
	flS(flTree, "neon birch", flGBroad, CMagenta, "tagged at the root, leaves faintly fluorescent").only("cyberpunk").w(3).wood("birch", 7),
	flS(flTree, "flesh oak", flGBroad, CRed, "warm to the touch, its bark scarred like skin").only("biopunk").w(3).wood("fleshwood", 6).fruit("blood-acorns", "mending", sAu, 2),
	flS(flTree, "solar linden", flGBroad, CYellow, "its leaves faceted to drink the sun").only("solarpunk").w(4).wood("linden", 9).fruit("sun-blossom", "mending", sSu, 2),
	flS(flTree, "sacred laurel", flGBroad, CGreen, "votive ribbons tied about its trunk").only("mythic").w(3).wood("laurel", 10).fruit("laurel leaves", "keen", sAll, 2),
	flS(flTree, "gilt ash", flGBroad, CYellow, "leaves like beaten old gold").only("twilight").w(3).wood("ash", 12),
	flS(flTree, "bone-white sycamore", flGBroad, CWhite, "a pale relic of an older forest").only("twilight", "wasteland").w(2).wood("sycamore", 8),

	// ---------------- snags, cacti, dead trees (TDeadTree) ----------------
	flS(flSnag, "saguaro", flGCactus, CGreen, "a fluted green column raising its arms").b(1).t(12, 40).h(0, 55).w(7).wood("saguaro ribs", 4).fruit("saguaro fruit", "spry", sSu, 2),
	flS(flSnag, "cardón", flGCactus, CDarkGreen, "a giant many-armed cactus").b(1).t(12, 40).h(0, 50).w(4).wood("cactus ribs", 4),
	flS(flSnag, "Joshua tree", flGCactus, CGray, "shaggy-limbed, each arm ending in a spiked fist").b(1, 2).t(8, 32).h(30, 70).w(5).wood("yucca wood", 3),
	flS(flSnag, "quiver tree", flGCactus, CYellow, "a forking aloe with golden, powdery bark").b(1, 3).t(16, 38).h(20, 60).w(3).wood("aloe wood", 3),
	flS(flSnag, "candelabra euphorbia", flGCactus, CDarkGreen, "a spurge of ribbed arms, its milk caustic").b(3, 1).t(18, 38).h(0, 60).w(3).fruit("euphorbia latex", "venom", sAll, 1),
	flS(flSnag, "boojum", flGCactus, CBrown, "an upside-down carrot of a tree, fuzzed with twigs").b(1).t(16, 36).h(0, 45).w(2),
	flS(flSnag, "charred snag", flGSnag, CDarkGray, "burned hollow, still smoking at its heart").b(13, 16).w(6).wood("charcoal", 3),
	flS(flSnag, "petrified trunk", flGSnag, CGray, "turned to stone long ago").b(13, 1, 2, 16).w(2),
	flS(flSnag, "obsidian thorn", flGSnag, CDarkMagenta, "a tree-shaped spike of volcanic glass").b(16, 13).w(3),
	flS(flSnag, "ghost tree", flGSnag, CWhite, "a bleached snag, barkless and silver").b(2, 3, 4, 5, 6, 7, 8, 9, 10, 12).w(2).wood("deadwood", 3),
	flS(flSnag, "lightning-split snag", flGSnag, CBrown, "split to the root by an old strike").b(3, 4, 5, 6, 7, 8, 9, 12).w(2).wood("deadwood", 3),
	flS(flSnag, "wired snag", flGSnag, CDarkCyan, "strung with dead cable").only("cyberpunk", "dieselpunk", "atompunk").w(4).wood("deadwood", 3),

	// ---------------- shrubs (TBush) ----------------
	flS(flShrub, "heather", flGShrub, CDarkMagenta, "low and wiry, a sea of tiny bells").b(4, 6, 8, 9, 10).t(-6, 18).h(0, 90).blo(CMagenta, sSu|sAu).w(6).fruit("heather tips", "lasting", sSu, 2),
	flS(flShrub, "gorse", flGShrub, CDarkGreen, "a spiny mound, its flowers smelling of coconut").b(4, 6, 8).t(2, 20).h(0, 70).blo(CYellow, sSp|sSu).w(5),
	flS(flShrub, "broom", flGShrub, CGreen, "whippy green stems").b(4, 6).t(4, 22).h(0, 65).blo(CYellow, sSp).w(3),
	flS(flShrub, "common juniper", flGShrub, CDarkCyan, "prickly, berried blue-black").b(4, 6, 9, 10, 2).t(-12, 20).h(0, 95).w(5).fruit("juniper berries", "keen", sAu, 2),
	flS(flShrub, "big sagebrush", flGShrub, CGray, "silver-grey, pungent after rain").b(2, 1, 4).t(-4, 24).h(20, 85).w(9).fruit("sage", "keen", sAll, 1),
	flS(flShrub, "creosote bush", flGShrub, CDarkGreen, "varnished leaves, smelling of rain").b(1).t(16, 40).h(0, 60).w(8).fruit("creosote leaves", "mending", sAll, 1),
	flS(flShrub, "saltbush", flGShrub, CGray, "grey and scurfy, tasting of salt").b(1, 2, 12).t(6, 38).h(0, 50).w(5).fruit("saltbush leaves", "thick", sAll, 2),
	flS(flShrub, "rhododendron", flGShrub, CDarkGreen, "leathery-leaved and dense").b(8, 6, 9).t(0, 20).h(30, 90).blo(CMagenta, sSp).w(4),
	flS(flShrub, "azalea", flGShrub, CGreen, "thin-twigged").b(6, 8).t(4, 22).h(20, 80).blo(COrange, sSp).w(2),
	flS(flShrub, "hawthorn", flGShrub, CGreen, "thorny, a hedge of it").b(6, 4, 8).t(0, 22).h(0, 70).blo(CWhite, sSp).w(6).fruit("haws", "mending", sAu, 3),
	flS(flShrub, "blackthorn", flGShrub, CDarkGray, "black-twigged and fiercely thorned").b(6, 4).t(2, 22).h(0, 60).blo(CWhite, sSp).w(4).fruit("sloes", "keen", sAu, 3),
	flS(flShrub, "bramble", flGShrub, CDarkGreen, "an arching tangle of hooked canes").b(6, 8, 4).t(2, 24).h(0, 65).blo(CWhite, sSu).w(7).fruit("blackberries", "spry", sAu, 4),
	flS(flShrub, "elder", flGShrub, CGreen, "pithy-stemmed, smelling of cats and summer").b(6, 4, 8).t(2, 22).h(0, 60).blo(CWhite, sSu).w(4).fruit("elderberries", "mending", sAu, 3),
	flS(flShrub, "hazel", flGShrub, CGreen, "many-stemmed, catkins in late winter").b(6, 8, 4).t(2, 20).h(0, 65).w(5).fruit("hazelnuts", "hearty", sAu, 3),
	flS(flShrub, "dog rose", flGShrub, CGreen, "scrambling, its hips scarlet").b(6, 4, 2).t(0, 24).h(0, 70).blo(CMagenta, sSu).w(4).fruit("rose hips", "mending", sAu, 3),
	flS(flShrub, "myrtle", flGShrub, CDarkGreen, "glossy and aromatic").b(4, 6).t(14, 28).h(0, 55).blo(CWhite, sSu).w(3).fruit("myrtle berries", "keen", sAu, 2),
	flS(flShrub, "rosemary", flGShrub, CDarkCyan, "needle-leaved, blue-flowered, rich-scented").b(4, 1, 2).t(14, 30).h(0, 60).w(4).fruit("rosemary", "keen", sAll, 2),
	flS(flShrub, "oleander", flGShrub, CDarkGreen, "lovely and wholly poisonous").b(4, 1, 5, 3).t(16, 34).h(0, 50).blo(CMagenta, sSu).w(3).fruit("oleander leaves", "venom", sAll, 1),
	flS(flShrub, "tea bush", flGShrub, CDarkGreen, "kept low, its young leaves plucked").b(5, 8, 7).t(14, 30).h(30, 80).w(3).fruit("tea leaves", "keen", sSp|sSu, 3),
	flS(flShrub, "wild coffee", flGShrub, CDarkGreen, "glossy leaves, red cherries").b(7, 5).t(18, 32).h(30, 75).w(3).fruit("coffee cherries", "keen", sAu, 3),
	flS(flShrub, "bilberry", flGShrub, CGreen, "knee-high, its berries staining the fingers").b(9, 6, 8).t(-8, 16).h(0, 85).dec(CRed).w(6).fruit("bilberries", "spry", sSu, 4),
	flS(flShrub, "lingonberry", flGShrub, CDarkGreen, "a mat of little glossy leaves").b(9, 10).t(-14, 10).h(0, 90).w(5).fruit("lingonberries", "rime", sAu, 4),
	flS(flShrub, "crowberry", flGShrub, CDarkGreen, "a heath-like mat, berries black").b(10, 9).t(-18, 6).h(0, 100).w(5).fruit("crowberries", "rime", sAu, 3),
	flS(flShrub, "dwarf birch", flGShrub, CGreen, "a birch no higher than your knee").b(10, 9).t(-20, 6).h(0, 100).dec(CRed).w(6),
	flS(flShrub, "arctic willow", flGShrub, CGreen, "a creeping willow, catkins upright").b(10, 11).t(-24, 4).h(0, 100).dec(CYellow).w(5),
	flS(flShrub, "Labrador tea", flGShrub, CDarkGreen, "rusty-felted beneath its leaves").b(9, 10, 12).t(-16, 10).h(0, 85).blo(CWhite, sSu).w(3).fruit("Labrador tea", "rime", sAll, 2),
	flS(flShrub, "bog myrtle", flGShrub, CDarkGreen, "resin-dotted, keeping midges away").b(12, 9).t(-4, 18).h(0, 50).w(4).fruit("gale leaves", "keen", sSu, 2),
	flS(flShrub, "buttonbush", flGShrub, CGreen, "white globes of flowers over standing water").b(12).t(6, 30).h(0, 35).blo(CWhite, sSu).w(3),
	flS(flShrub, "acacia scrub", flGShrub, CDarkGreen, "a thorn thicket").b(3, 1).t(18, 40).h(0, 60).w(6),
	flS(flShrub, "king protea", flGShrub, CGreen, "leathery leaves, a great crowned bloom").b(3, 4).t(14, 30).h(20, 70).blo(CMagenta, sWi|sSp).w(3).fruit("protea nectar", "spry", sSp, 1),
	flS(flShrub, "manzanita", flGShrub, CDarkRed, "red bark smooth as polished wood").b(2, 4, 1).t(10, 28).h(20, 75).w(3).fruit("manzanita berries", "spry", sSu, 2),
	flS(flShrub, "jojoba", flGShrub, CGray, "grey-green, its seeds full of wax").b(1).t(16, 40).h(0, 55).w(3).fruit("jojoba seeds", "thick", sSu, 2),
	flS(flShrub, "hibiscus", flGShrub, CGreen, "glossy, blooms open for a single day").b(7, 5).t(20, 36).h(0, 50).blo(CRed, sAll).w(3).fruit("hibiscus calyces", "mending", sAll, 2),
	flS(flShrub, "chilli bush", flGShrub, CGreen, "hung with fiery pods").b(5, 7, 3).t(20, 36).h(0, 55).w(2).fruit("wild chillies", "ember", sSu|sAu, 3),
	flS(flShrub, "prickly pear", flGCactus, CGreen, "flat green pads armed with spines").b(1, 2, 3, 4).t(10, 40).h(0, 60).blo(CYellow, sSp).w(6).fruit("tunas", "spry", sSu, 3),
	flS(flShrub, "teddy-bear cholla", flGCactus, CYellow, "fuzzy-looking, its joints leaping onto passers-by").b(1).t(16, 40).h(0, 55).w(4),
	flS(flShrub, "barrel cactus", flGCactus, CDarkGreen, "a ribbed barrel crowned with hooked spines").b(1, 2).t(12, 40).h(0, 60).blo(COrange, sSu).w(4).fruit("barrel cactus pulp", "lasting", sAll, 1),
	flS(flShrub, "agave", flGCactus, CCyan, "a rosette of sword-leaves, flowering once and dying").b(1, 3, 2).t(14, 38).h(0, 65).w(5).fruit("agave heart", "lasting", sAll, 1),
	flS(flShrub, "aloe", flGCactus, CGreen, "fleshy, toothed leaves full of cool gel").b(1, 3, 5).t(16, 38).h(0, 60).blo(COrange, sWi).w(4).fruit("aloe gel", "mending", sAll, 2),
	flS(flShrub, "yucca", flGCactus, CGray, "a spiked rosette with a tall bell spike").b(1, 2, 4).t(8, 34).h(0, 70).blo(CWhite, sSu).w(4).fruit("yucca root", "hearty", sAll, 1),
	flS(flShrub, "ocotillo", flGCactus, CBrown, "a bundle of thorny whips, tipped red after rain").b(1).t(16, 40).h(0, 55).blo(CRed, sSp).w(3),
	flS(flShrub, "ember briar", flGShrub, CRed, "thorns that stay hot").b(13, 16).w(5).fruit("ember hips", "ember", sAu, 2),
	flS(flShrub, "ash thistle", flGShrub, CGray, "grey spines drinking the ash").b(13).w(4),
	flS(flShrub, "geode bramble", flGShrub, CMagenta, "a thicket of brittle crystal canes").b(14).w(5),
	flS(flShrub, "mycelium thicket", flGShrub, CDarkMagenta, "ropes of fungal threads").b(15).w(4).fruit("thread-caps", "mending", sAll, 2),
	flS(flShrub, "cloudberry briar", flGShrub, CWhite, "white berries frosted with rime").lay(flSky).w(4).fruit("cloudberries", "spry", sSu, 3),
	flS(flShrub, "barbed scrub", flGShrub, CBrown, "a tangle of thorn and old wire").only("wasteland", "dieselpunk").w(4),
	flS(flShrub, "spliced berry hedge", flGShrub, CDarkCyan, "berries in colours no hedge should have").only("biopunk", "cyberpunk", "solarpunk").w(3).fruit("splice-berries", "spry", sAll, 3),

	// ---------------- grasses, sedges, ferns (TTallGrass) ----------------
	flS(flGrass, "big bluestem", flGGrass, CCyan, "taller than a rider, turkey-foot seed heads").b(4).t(2, 26).h(0, 60).aut(COrange).w(7),
	flS(flGrass, "switchgrass", flGGrass, CGreen, "airy-headed prairie grass").b(4, 12).t(2, 26).h(0, 55).aut(CYellow).w(5),
	flS(flGrass, "feather grass", flGGrass, CGray, "silver plumes streaming in the steppe wind").b(2, 4).t(-4, 22).h(20, 80).aut(CYellow).w(7),
	flS(flGrass, "pampas grass", flGGrass, CGreen, "great creamy plumes").b(4, 3).t(10, 30).h(0, 60).aut(CWhite).w(4),
	flS(flGrass, "elephant grass", flGGrass, CGreen, "head-high and razor-edged").b(3, 5).t(20, 38).h(0, 55).aut(CYellow).w(7),
	flS(flGrass, "red oat grass", flGGrass, CDarkRed, "rust-red seed heads nodding").b(3).t(18, 38).h(0, 60).aut(CBrown).w(6),
	flS(flGrass, "spinifex", flGGrass, CYellow, "a hummock of needle-sharp leaves").b(1, 3).t(18, 40).h(0, 60).w(6),
	flS(flGrass, "marram grass", flGGrass, CGreen, "binding the dunes").b(1, 4, 12).t(4, 30).h(0, 26).aut(CYellow).w(3),
	flS(flGrass, "tussock grass", flGGrass, CBrown, "dense, hard tussocks").b(10, 4, 2).t(-16, 14).h(0, 100).w(6),
	flS(flGrass, "cottongrass", flGGrass, CWhite, "tufts of white cotton bobbing over the bog").b(10, 12, 9).t(-18, 12).h(0, 90).w(5),
	flS(flGrass, "meadow fescue", flGGrass, CGreen, "soft and lush").b(4, 6, 8).t(0, 22).h(0, 75).aut(CYellow).w(5),
	flS(flGrass, "wild oats", flGGrass, CYellow, "dangling spikelets").b(4, 6, 1, 2).t(6, 28).h(0, 60).w(3).fruit("wild oats", "lasting", sSu, 2),
	flS(flGrass, "tufted sedge", flGGrass, CDarkGreen, "three-edged stems in wet ground").b(12, 8, 9, 6).t(-6, 24).h(0, 70).w(5),
	flS(flGrass, "bracken", flGGrass, CGreen, "fronds waist-high, curling brown in autumn").b(6, 8, 9).t(-2, 22).h(0, 75).dec(CBrown).w(7),
	flS(flGrass, "ostrich fern", flGGrass, CGreen, "shuttlecock fronds").b(6, 9, 8).t(-4, 18).h(0, 65).dec(CBrown).w(3).fruit("fiddleheads", "spry", sSp, 2),
	flS(flGrass, "sword fern", flGGrass, CDarkGreen, "evergreen fronds in fountains").b(8).t(2, 18).h(0, 70).w(6),
	flS(flGrass, "lemongrass", flGGrass, CGreen, "citrus-scented blades").b(5, 7).t(20, 36).h(0, 50).w(3).fruit("lemongrass", "keen", sAll, 2),
	flS(flGrass, "giant reed fern", flGGrass, CDarkGreen, "fronds as long as a man").b(7, 5).t(20, 36).h(0, 55).w(5),
	flS(flGrass, "ashgrass", flGGrass, CGray, "blades powdered grey").b(13).w(5),
	flS(flGrass, "glass grass", flGGrass, CCyan, "chiming as you push through").b(14).w(5),
	flS(flGrass, "windgrass", flGGrass, CWhite, "flattened by the endless wind").lay(flSky).w(5),
	flS(flGrass, "fibre-optic sedge", flGGrass, CCyan, "glowing at its tips").only("cyberpunk").w(3),
	flS(flGrass, "rad-wheat", flGGrass, CYellow, "feral grain gone strange").only("wasteland", "atompunk").w(3).fruit("rad-grain", "lasting", sSu, 2),

	// ---------------- flowers ----------------
	flS(flFlower, "corn poppy", flGFlower, CGreen, "a papery scarlet cup").b(4, 6, 2).t(4, 26).h(0, 60).blo(CRed, sSu).w(6).fruit("poppy seeds", "lasting", sAu, 2),
	flS(flFlower, "lupin", flGFlower, CGreen, "a spire of pea flowers").b(4, 8, 9, 6).t(-4, 20).h(0, 80).blo(CBlue, sSp|sSu).w(5),
	flS(flFlower, "edelweiss", flGFlower, CGray, "woolly white stars on the high scree").b(2, 6, 9, 10, 4).t(-12, 16).h(62, 100).blo(CWhite, sSu).w(6),
	flS(flFlower, "trumpet gentian", flGFlower, CGreen, "a deep blue trumpet").b(2, 6, 9, 10, 4, 8).t(-12, 16).h(55, 100).blo(CBlue, sSu).w(5).fruit("gentian root", "mending", sAu, 1),
	flS(flFlower, "alpine aster", flGFlower, CGreen, "violet rays around a golden eye").b(2, 9, 10, 6).t(-12, 16).h(55, 100).blo(CMagenta, sSu).w(3),
	flS(flFlower, "wild orchid", flGFlower, CGreen, "an intricate, waxy bloom").b(7, 5, 8, 6).t(4, 36).h(0, 75).blo(CMagenta, sSp|sSu).w(4),
	flS(flFlower, "sunflower", flGFlower, CGreen, "a face that follows the sun").b(4, 3).t(10, 32).h(0, 60).blo(CYellow, sSu|sAu).w(4).fruit("sunflower seeds", "hearty", sAu, 3),
	flS(flFlower, "lavender", flGFlower, CCyan, "purple spikes, humming with bees").b(4, 1, 2).t(12, 30).h(0, 70).blo(CBlue, sSu).w(5).fruit("lavender", "mending", sSu, 2),
	flS(flFlower, "bluebell", flGFlower, CGreen, "nodding blue bells in the shade").b(6, 8).t(2, 18).h(0, 60).blo(CBlue, sSp).w(6),
	flS(flFlower, "snowdrop", flGFlower, CGreen, "a white drop hanging over the snow").b(6, 8).t(0, 18).h(0, 65).blo(CWhite, sWi).w(3),
	flS(flFlower, "crocus", flGFlower, CGreen, "a goblet opening on the thaw").b(6, 4, 2).t(0, 22).h(0, 85).blo(CMagenta, sSp).w(3).fruit("saffron threads", "keen", sAu, 1),
	flS(flFlower, "wild daffodil", flGFlower, CGreen, "a yellow trumpet").b(6, 4).t(2, 20).h(0, 60).blo(CYellow, sSp).w(3),
	flS(flFlower, "cornflower", flGFlower, CGreen, "ragged blue stars among the wheat").b(4, 6).t(4, 24).h(0, 55).blo(CBlue, sSu).w(3),
	flS(flFlower, "foxglove", flGFlower, CGreen, "a tall spike of speckled bells; heart poison").b(6, 8).t(2, 20).h(0, 70).blo(CMagenta, sSu).w(4).fruit("foxglove leaves", "venom", sSu, 1),
	flS(flFlower, "marigold", flGFlower, CGreen, "bright and bitter-scented").b(5, 3, 7).t(16, 34).h(0, 60).blo(COrange, sAll).w(3).fruit("marigold petals", "mending", sAll, 2),
	flS(flFlower, "desert marigold", flGFlower, CGray, "woolly-leaved, yellow after rain").b(1, 2).t(10, 38).h(0, 60).blo(CYellow, sSp).w(5),
	flS(flFlower, "desert lily", flGFlower, CGray, "a white trumpet from a buried bulb").b(1).t(16, 40).h(0, 50).blo(CWhite, sSp).w(3).fruit("lily bulb", "lasting", sAll, 1),
	flS(flFlower, "fireweed", flGFlower, CGreen, "magenta spires on burned ground").b(9, 6, 8, 10, 13).t(-10, 18).h(0, 85).blo(CMagenta, sSu).w(5).fruit("fireweed shoots", "spry", sSp, 2),
	flS(flFlower, "arctic poppy", flGFlower, CGreen, "a yellow cup turned to the low sun").b(10, 11).t(-24, 4).h(0, 100).blo(CYellow, sSu).w(5),
	flS(flFlower, "purple saxifrage", flGFlower, CDarkGreen, "a cushion of tiny purple stars on bare rock").b(10, 11, 2).t(-24, 8).h(40, 100).blo(CMagenta, sSp).w(4),
	flS(flFlower, "chamomile", flGFlower, CGreen, "little daisies smelling of apples").b(4, 6).t(4, 26).h(0, 60).blo(CWhite, sSu).w(4).fruit("chamomile", "mending", sSu, 2),
	flS(flFlower, "yarrow", flGFlower, CGreen, "flat white heads over feathery leaves").b(4, 6, 2, 9).t(-4, 24).h(0, 80).blo(CWhite, sSu).w(4).fruit("yarrow", "mending", sSu, 2),
	flS(flFlower, "St John's wort", flGFlower, CGreen, "five yellow petals, pricked with oil glands").b(6, 4).t(2, 24).h(0, 60).blo(CYellow, sSu).w(3).fruit("St John's wort", "keen", sSu, 1),
	flS(flFlower, "wild garlic", flGFlower, CGreen, "white stars over a smell of onions").b(6, 8).t(2, 20).h(0, 55).blo(CWhite, sSp).w(4).fruit("ramsons", "hearty", sSp, 3),
	flS(flFlower, "flame lily", flGFlower, CGreen, "a climbing lily, its petals like fire").b(3, 5).t(18, 36).h(0, 55).blo(CRed, sSu).w(4),
	flS(flFlower, "ember poppy", flGFlower, CDarkRed, "petals that smoulder").b(13, 16).blo(COrange, sAll).w(5).fruit("ember seeds", "ember", sAu, 1),
	flS(flFlower, "glass rose", flGFlower, CCyan, "a rose of clear crystal").b(14).blo(CWhite, sAll).w(4),
	flS(flFlower, "zephyr bell", flGFlower, CCyan, "ringing faintly in the wind").lay(flSky).blo(CWhite, sSp|sSu).w(5),
	flS(flFlower, "smog lily", flGFlower, CDarkGray, "blooming grey and still alive").only("steampunk", "dieselpunk", "cyberpunk").blo(CGray, sSu).w(3),

	// ---------------- reeds (TReeds) ----------------
	flS(flReed, "common reed", flGGrass, CDarkCyan, "plumed canes rustling").b(12, 4, 6, 2).t(-4, 30).h(0, 60).aut(CBrown).w(8),
	flS(flReed, "papyrus", flGGrass, CGreen, "green stems topped with starbursts").b(12, 5, 1, 3, 7).t(18, 40).h(0, 40).w(6).fruit("papyrus pith", "lasting", sAll, 1),
	flS(flReed, "bulrush", flGGrass, CBrown, "brown velvet heads on tall stems").b(12, 6, 4, 9).t(-4, 28).h(0, 55).aut(CBrown).w(7).fruit("cattail root", "hearty", sAll, 2),
	flS(flReed, "wild rice", flGGrass, CGreen, "grain heads leaning over the water").b(12, 9, 4).t(-2, 24).h(0, 45).w(4).fruit("wild rice", "lasting", sAu, 3),
	flS(flReed, "soft rush", flGGrass, CDarkGreen, "glossy green spears").b(12, 8, 6).t(0, 24).h(0, 60).w(4),
	flS(flReed, "giant horsetail", flGGrass, CGreen, "jointed and ancient").b(12, 8, 7).t(4, 30).h(0, 55).w(3),
	flS(flReed, "glass reed", flGGrass, CCyan, "hollow crystal stems").b(14).w(4),
	flS(flReed, "sump reed", flGGrass, CDarkGray, "thriving on runoff").only("dieselpunk", "cyberpunk", "wasteland").w(3),

	// ---------------- mosses and lichens (TMoss) ----------------
	flS(flMoss, "sphagnum moss", flGMoss, CGreen, "a sodden red-green carpet").b(12, 9, 8, 10).t(-12, 22).h(0, 75).w(6).fruit("sphagnum", "mending", sAll, 1),
	flS(flMoss, "haircap moss", flGMoss, CDarkGreen, "a forest of tiny stars").b(6, 8, 9).t(-6, 22).h(0, 80).w(5),
	flS(flMoss, "feather moss", flGMoss, CGreen, "fern-like and springy").b(9, 8).t(-10, 18).h(0, 80).w(5),
	flS(flMoss, "cushion moss", flGMoss, CGreen, "domed and soft").b(6, 8, 7).t(0, 26).h(0, 70).w(4),
	flS(flMoss, "reindeer lichen", flGMoss, CGray, "pale, coral-like tufts").b(10, 9, 11).t(-24, 8).h(0, 100).w(6).fruit("reindeer lichen", "lasting", sAll, 1),
	flS(flMoss, "map lichen", flGMoss, CYellow, "yellow-green patches like a map").b(10, 11, 2).t(-24, 10).h(40, 100).w(3),
	flS(flMoss, "liverwort", flGMoss, CDarkGreen, "flat lobed sheets").b(7, 8, 12).t(4, 34).h(0, 60).w(4),
	flS(flMoss, "club moss", flGMoss, CGreen, "creeping, tiny fir trees").b(8, 9, 7).t(-4, 28).h(0, 80).w(3),
	flS(flMoss, "peat moss", flGMoss, CBrown, "deep and spongy").b(12, 10).t(-10, 20).h(0, 50).w(4),
	flS(flMoss, "ember moss", flGMoss, CDarkRed, "warm and faintly glowing").b(13, 16).w(5),
	flS(flMoss, "spore moss", flGMoss, CDarkMagenta, "puffing at a touch").b(15).w(5),
	flS(flMoss, "prism lichen", flGMoss, CMagenta, "splitting light into colours").b(14).w(4),
	flS(flMoss, "luminous cave moss", flGMoss, CCyan, "glowing a soft green-blue").lay(flDeep).w(6),
	flS(flMoss, "nitre lichen", flGMoss, CWhite, "crusting the damp stone with salts").lay(flDeep).w(4).fruit("nitre crust", "ember", sAll, 1),
	flS(flMoss, "cave pearl moss", flGMoss, CGray, "beaded with tiny limestone pearls").lay(flDeep).w(3),
	flS(flMoss, "cloud moss", flGMoss, CWhite, "wet and silver-grey").lay(flSky).w(5),
	flS(flMoss, "oil-slick moss", flGMoss, CDarkCyan, "iridescent with spilled fuel").only("dieselpunk", "wasteland", "steampunk").w(3),

	// ---------------- giant fungi (TMushroom) ----------------
	flS(flGiant, "towercap", flGShrub, CMagenta, "a pale stalk as thick as a tower, its cap lost in gloom").b(15).lay(flDeep).w(6).wood("towercap", 7),
	flS(flGiant, "violet ghostcap", flGShrub, CDarkMagenta, "a translucent giant, faintly lit from within").b(15).w(5).wood("ghostcap", 6).fruit("ghostcap gills", "keen", sAll, 2),
	flS(flGiant, "shelf giant", flGShrub, CBrown, "tier on tier of hard brackets").b(15).w(4).wood("shelf-wood", 8),
	flS(flGiant, "puffball tree", flGShrub, CWhite, "swollen globes on stalks, bursting into smoke").b(15).w(4).fruit("puffball flesh", "hearty", sAll, 2),
	flS(flGiant, "lantern morel", flGShrub, CYellow, "honeycombed and glowing amber").b(15).w(3).fruit("lantern morel", "spry", sAll, 2),
	flS(flGiant, "stinkhorn spire", flGShrub, CRed, "a slimy spire that reeks of carrion").b(15).w(2).fruit("stinkhorn egg", "venom", sAll, 1),
	flS(flGiant, "blood-gill", flGShrub, CDarkRed, "dripping red under its cap").b(15).w(2).fruit("blood-gill", "mending", sAll, 1),
	flS(flGiant, "glowcap", flGShrub, CCyan, "its gills glowing cold green").lay(flDeep).w(6).fruit("glowcap", "keen", sAll, 2),
	flS(flGiant, "blindcap", flGShrub, CWhite, "white and eyeless, grown in the dark").lay(flDeep).w(4).wood("blindcap", 5).fruit("blindcap flesh", "lasting", sAll, 2),
	flS(flGiant, "cave-ear", flGShrub, CBrown, "brown ear-shaped brackets climbing a stalk").lay(flDeep).w(3).fruit("cave-ear", "thick", sAll, 2),
	flS(flGiant, "nitre-cap", flGShrub, CGray, "crusted with salts").lay(flDeep).w(2).fruit("nitre-cap", "ember", sAll, 1),
	flS(flGiant, "cloud-cap", flGShrub, CWhite, "a giant puff of a mushroom").lay(flSky).w(3),

	// ---------------- small fungi (TFungus, forest floor) ----------------
	flS(flMat, "spore mat", flGMoss, CDarkMagenta, "a felt of threads, breathing spores").b(15).w(5),
	flS(flMat, "slime mould bloom", flGMoss, CYellow, "a yellow net, slowly creeping").b(15, 12, 7).w(3),
	flS(flMat, "fly agaric", flGShrub, CRed, "a scarlet cap flecked with white").b(9, 6, 8).t(-8, 20).h(0, 80).blo(CRed, sSu|sAu).w(5).fruit("fly agaric", "venom", sSu|sAu, 1),
	flS(flMat, "chanterelle", flGShrub, CYellow, "a golden funnel smelling of apricots").b(6, 8, 9).t(-2, 22).h(0, 70).blo(CYellow, sSu|sAu).w(5).fruit("chanterelles", "spry", sSu|sAu, 2),
	flS(flMat, "morel", flGShrub, CBrown, "a honeycombed cone").b(6, 4, 9).t(0, 22).h(0, 70).blo(CBrown, sSp).w(4).fruit("morels", "hearty", sSp, 2),
	flS(flMat, "giant puffball", flGShrub, CWhite, "a white ball the size of a head").b(4, 6).t(2, 24).h(0, 60).blo(CWhite, sSu|sAu).w(3).fruit("puffball", "hearty", sSu|sAu, 2),
	flS(flMat, "porcini", flGShrub, CBrown, "a fat brown cap on a bulging stem").b(6, 9, 8).t(-2, 22).h(0, 75).blo(CBrown, sAu).w(4).fruit("porcini", "hearty", sAu, 2),
	flS(flMat, "death cap", flGShrub, CGreen, "olive-capped and deadly").b(6, 4).t(4, 24).h(0, 60).blo(CGreen, sAu).w(2).fruit("death cap", "venom", sAu, 1),
	flS(flMat, "oyster mushroom", flGShrub, CGray, "grey fans on rotting wood").b(6, 8, 7, 5).t(2, 32).h(0, 65).blo(CGray, sAu|sWi).w(3).fruit("oyster mushrooms", "thick", sAu|sWi, 2),
	flS(flMat, "shaggy inkcap", flGShrub, CWhite, "melting into black ink").b(4, 6).t(2, 24).h(0, 60).blo(CWhite, sAu).w(2),
	flS(flMat, "termite mushroom", flGShrub, CGray, "a huge cap from a termite mound").b(3, 5, 7).t(20, 36).h(0, 50).blo(CGray, sSu).w(3).fruit("termite mushroom", "hearty", sSu, 2),
	flS(flMat, "glowcap patch", flGMoss, CCyan, "a scatter of little glowing caps").lay(flDeep).w(6).fruit("glowcaps", "keen", sAll, 2),
	flS(flMat, "cave puffballs", flGMoss, CGray, "grey puffs that burst underfoot").lay(flDeep).w(3),

	// ---------------- aquatic ----------------
	flS(flKelp, "giant kelp", flGGrass, CDarkGreen, "fronds rising from the deep on gas bladders").lay(flSea).t(-99, 22).w(7).fruit("kelp", "deep", sAll, 2),
	flS(flKelp, "bull kelp", flGGrass, CBrown, "a whip-like stipe with a single float").lay(flSea).t(-99, 18).w(4).fruit("bull kelp", "deep", sAll, 2),
	flS(flKelp, "sugar kelp", flGGrass, CYellow, "a crinkled golden ribbon").lay(flSea).t(-99, 20).w(4).fruit("sugar kelp", "deep", sAll, 2),
	flS(flKelp, "oarweed", flGGrass, CDarkGreen, "fingered fronds").lay(flSea).t(-99, 18).w(4),
	flS(flKelp, "eelgrass", flGGrass, CGreen, "ribbon leaves combed by the current").lay(flSea).w(5),
	flS(flKelp, "turtle grass", flGGrass, CGreen, "flat green blades on the sand").lay(flSea).t(18, 99).w(5),
	flS(flKelp, "Neptune grass", flGGrass, CDarkGreen, "old meadows of the sea").lay(flSea).t(14, 26).w(4).fruit("sea olives", "deep", sAu, 2),
	flS(flKelp, "sargassum", flGGrass, CBrown, "berried with air sacs").lay(flSea).t(18, 99).w(4),
	flS(flKelp, "sea lettuce", flGGrass, CGreen, "thin bright sheets").lay(flSea).w(3).fruit("sea lettuce", "deep", sAll, 2),
	flS(flKelp, "glow kelp", flGGrass, CCyan, "lit along its edges").lay(flSea).only("cyberpunk", "biopunk", "atompunk").w(3).fruit("glow kelp", "deep", sAll, 2),
	flS(flLily, "white water lily", flGFlower, CGreen, "pads floating flat on the water").b(6, 4, 12, 8, 9).t(-2, 26).h(0, 60).blo(CWhite, sSu).w(6).fruit("lily root", "lasting", sAu, 1),
	flS(flLily, "sacred lotus", flGFlower, CGreen, "standing out of the mud unsoiled").b(5, 7, 12, 3).t(16, 38).h(0, 50).blo(CMagenta, sSu).w(5).fruit("lotus seeds", "mending", sAu, 2),
	flS(flLily, "water hyacinth", flGFlower, CGreen, "swollen floats, choking the water").b(7, 5, 12).t(18, 38).h(0, 45).blo(CBlue, sAll).w(4),
	flS(flLily, "duckweed", flGFlower, CGreen, "a green skin on still water").w(2),
	flS(flLily, "giant water lily", flGFlower, CGreen, "pads wide enough to hold a child").b(7).t(22, 38).h(0, 40).blo(CWhite, sSu).w(2),
	flS(flLily, "water chestnut", flGFlower, CDarkGreen, "a rosette of toothed leaves").b(6, 5, 12).t(8, 30).h(0, 45).w(2).fruit("water chestnuts", "hearty", sAu, 2),
	flS(flSeagrass, "shoal eelgrass", flGGrass, CGreen, "a ribbon meadow in the shallows").b(0).w(5).fruit("eelgrass seeds", "deep", sSu, 1),
	flS(flSeagrass, "shoal turtle grass", flGGrass, CGreen, "flat blades under clear water").b(0).t(18, 99).w(4),
	flS(flSeagrass, "bladderwrack", flGGrass, CBrown, "paired air bladders popping underfoot").b(0).t(-99, 20).w(4).fruit("bladderwrack", "deep", sAll, 2),

	// ---------------- crops (TCrop) ----------------
	flS(flCrop, "wheat", flGGrass, CYellow, "golden heads of grain").b(4, 6, 2).t(2, 26).h(0, 70).w(8).fruit("wheat", "lasting", sSu, 3),
	flS(flCrop, "barley", flGGrass, CYellow, "bearded, nodding heads").b(2, 6, 9, 4).t(-4, 24).h(0, 85).w(6).fruit("barley", "lasting", sSu, 3),
	flS(flCrop, "rye", flGGrass, CBrown, "tall grey-green grain").b(9, 6, 2).t(-8, 18).h(0, 80).w(4).fruit("rye", "lasting", sSu, 3),
	flS(flCrop, "oats", flGGrass, CYellow, "loose drooping spikelets").b(6, 8).t(0, 20).h(0, 70).w(4).fruit("oats", "lasting", sSu, 3),
	flS(flCrop, "millet", flGGrass, CBrown, "brushy seed heads").b(3, 1, 2, 5).t(14, 38).h(0, 60).w(6).fruit("millet", "lasting", sAu, 3),
	flS(flCrop, "sorghum", flGGrass, CDarkRed, "red grain heads on cane-like stalks").b(3, 5, 1).t(18, 38).h(0, 60).w(5).fruit("sorghum", "lasting", sAu, 3),
	flS(flCrop, "rice paddy", flGGrass, CGreen, "flooded rows of green").b(5, 7, 12).t(16, 38).h(0, 60).w(8).fruit("rice", "lasting", sAu, 3),
	flS(flCrop, "maize", flGGrass, CGreen, "tall stalks, tasselled").b(4, 5, 3, 6).t(12, 34).h(0, 70).w(6).fruit("maize", "lasting", sAu, 3),
	flS(flCrop, "potatoes", flGGrass, CDarkGreen, "rows of leafy mounds").b(6, 9, 2, 8).t(-4, 20).h(30, 95).w(4).fruit("potatoes", "hearty", sAu, 3),
	flS(flCrop, "flax", flGGrass, CBlue, "slender stems, sky-blue flowers").b(6, 4, 9).t(0, 22).h(0, 60).w(3).fruit("linseed", "thick", sSu, 2),
	flS(flCrop, "cotton", flGGrass, CWhite, "bolls splitting into white fluff").b(3, 5, 1).t(18, 38).h(0, 55).w(4),
	flS(flCrop, "terraced tea", flGShrub, CDarkGreen, "clipped hedges contouring the hill").b(5, 8, 7).t(14, 30).h(35, 85).w(4).fruit("tea leaves", "keen", sSp|sSu, 3),
	flS(flCrop, "sugar cane", flGGrass, CGreen, "thick jointed canes").b(7, 5).t(20, 38).h(0, 50).w(5).fruit("sugar cane", "spry", sWi, 3),
	flS(flCrop, "cassava", flGShrub, CGreen, "hand-shaped leaves over buried roots").b(7, 5, 3).t(20, 38).h(0, 55).w(4).fruit("cassava", "hearty", sAll, 3),
	flS(flCrop, "taro", flGShrub, CDarkGreen, "elephant-ear leaves in wet beds").b(12, 7, 5).t(20, 38).h(0, 50).w(4).fruit("taro", "hearty", sAll, 3),
	flS(flCrop, "grapevines", flGShrub, CGreen, "trained along stakes").b(4, 6, 2).t(12, 26).h(0, 65).dec(CRed).w(4).fruit("grapes", "spry", sAu, 3),
	flS(flCrop, "quinoa", flGGrass, CRed, "red and gold seed heads").b(2, 10, 4).t(-4, 20).h(55, 100).w(3).fruit("quinoa", "lasting", sAu, 3),
	flS(flCrop, "buckwheat", flGGrass, CWhite, "pink-white sprays over red stems").b(9, 6, 2).t(-6, 18).h(0, 80).w(3).fruit("buckwheat", "lasting", sAu, 3),
	flS(flCrop, "hops", flGShrub, CGreen, "bines climbing tall poles").b(6, 8).t(2, 18).h(0, 60).w(2).fruit("hops", "keen", sAu, 2),
	flS(flCrop, "beans", flGShrub, CGreen, "twining up their poles").b(4, 6, 5, 3).t(8, 30).h(0, 70).w(4).fruit("beans", "thick", sSu, 3),
	flS(flCrop, "ash-barley", flGGrass, CGray, "a grey grain bred for ash soils").b(13, 16).w(5).fruit("ash-barley", "lasting", sSu, 3),
	flS(flCrop, "cavern cap beds", flGShrub, CDarkMagenta, "rows of farmed fungus").b(15).lay(flDeep).w(5).fruit("farmed caps", "hearty", sAll, 3),
	flS(flCrop, "vat algae beds", flGGrass, CCyan, "green slurry in clear troughs").only("cyberpunk", "biopunk", "solarpunk").w(4).fruit("algae cake", "lasting", sAll, 3),

	// ---------------- crystal growths (TCrystal) ----------------
	flS(flCrystal, "singing quartz spire", flGCrys, CMagenta, "a spire of singing crystal").b(14).w(6).wood("quartz shards", 12),
	flS(flCrystal, "prism tree", flGCrys, CCyan, "a crystal trunk branching into glassy leaves that split the light").b(14).w(5).wood("prism shards", 16),
	flS(flCrystal, "amethyst palm", flGCrys, CMagenta, "violet crystal fronds").b(14).w(3).wood("amethyst shards", 18),
	flS(flCrystal, "rime-crystal fir", flGCrys, CWhite, "a cone of frosted crystal").b(14).w(3).wood("rime shards", 14).fruit("crystal dew", "rime", sAll, 1),
	flS(flCrystal, "chime willow", flGCrys, CCyan, "hanging strings of crystal that chime").b(14).w(3).wood("chime shards", 15),
	flS(flCrystal, "geode stump", flGCrys, CMagenta, "a split boulder lined with crystal teeth").w(2).wood("geode shards", 10),
}

// per-genre reskins applied to some tiles (name prefix, extra look line, colour)
type flSkin struct {
	Adj    string
	Note   string
	F      byte
	Chance int // percent of eligible plant tiles
}

var flGenreSkins = map[string][]flSkin{
	"steampunk":  {{"soot-streaked", "Its leaves are filmed with coal soot.", CDarkGray, 22}, {"brass-tagged", "A brass survey tag is nailed to it.", 0, 6}},
	"dieselpunk": {{"oil-stained", "Black oil has soaked into it from the ground.", CBrown, 20}, {"smog-blighted", "Grey film coats every leaf.", CDarkGray, 12}},
	"atompunk":   {{"glowing", "It hums faintly; its green is a little too green.", CCyan, 10}, {"fallout-dusted", "Fine grey dust clings to it.", CGray, 10}},
	"cyberpunk":  {{"smog-blighted", "Grey film coats every leaf.", CDarkGray, 22}, {"gene-tagged", "A barcode is laser-burned into it.", 0, 10}},
	"biopunk":    {{"gene-modified", "Its veins pulse faintly when you touch it.", CDarkCyan, 25}, {"tumorous", "Swollen galls bulge from it.", CDarkRed, 8}},
	"solarpunk":  {{"tended", "A seed-bank ribbon is tied to it.", 0, 15}, {"grafted", "Several kinds have been grafted onto one stock.", 0, 6}},
	"wasteland":  {{"rust-leaved", "Its leaves are spotted orange like rust.", CBrown, 22}, {"blighted", "Half of it is dead.", CDarkGray, 16}},
	"twilight":   {{"withering", "It looks as though it has stopped trying.", CBrown, 14}, {"pale", "It has the sheen of old gilt.", CYellow, 6}},
}
