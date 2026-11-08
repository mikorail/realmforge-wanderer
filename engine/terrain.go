package engine

type Terrain uint8

const (
	TVoid Terrain = iota // open sky: you fall
	TFloor
	TGrass
	TTallGrass
	TSand
	TDirt
	TRoad
	TSnow
	TIce
	TAsh
	TMoss
	TFungus
	TCrystalFloor
	TTree
	TDeadTree
	TMushroom
	TBush
	TReeds
	TRock
	TWall
	TDoor
	TDoorOpen
	TCrystal
	TShallow
	TDeep
	TLava
	TDown
	TUp
	TUpdraft
	TWell
	TAltar
	TStall
	TSarcophagus
	TInscription
	TRubble
	TBridge
	TPillar
	TBed
	TShelf
	TStatue
	TRift
	TCloud
	TCoral
	TKelp
	TBones
	TCarpet
	TMachine
	TBrazier
	TTrunk // world tree
	TChasm
	TThrone
	TAnvil
	TDummy
	TCrate
	TCrop
	TBoat
	TLocked
	TOre
	TSeat
	TCobble
	TTrap
	TPortal // a rift to another planet (planets.go)
	// landforms (tglandforms.go)
	TCliff
	TFalls
	TSpray
	TGlacier
	TIceWall
	TPalm
	THotSpring
	TGeyser
	TFumarole
	TTar
	TQuicksand
	TSalt
	TObsidian
	TSulfur
	// architecture & unique materials
	TWoodWall
	TBrickWall
	TStoneWall
	TMarbleWall
	TObsidianWall
	TBoneWall
	TAdobeWall
	TBambooWall
	TCoralWall
	TMetalWall
	THedgeWall
	TWindow
	TGateArch
	TGateClosed
	TWoodFloor
	TTileFloor
	TScaffold
	// expanded tactical & ecological features
	TBarricade
	TBallista
	TLantern
	TBunting
	TSnowDrift
	TMonument
	TSkylight
	TChimney
	TBeehive
	TBarrel
	TChandelier
	TTent
	TAshBed
	TSilkwormRack
	TCoralDome
	NTerrain
)

type TDef struct {
	Name     string
	G        rune
	F, B     byte
	Walk     bool
	Opaque   bool
	Water    int // 0 dry, 1 shallow, 2 deep
	Burn     bool
	Desc     string
	Interact bool
}

var TDefs = [NTerrain]TDef{
	TVoid:         {"open sky", ' ', CBlue, CBgDark, true, false, 0, false, "Nothing but wind and a long, long fall.", false},
	TFloor:        {"floor", '.', CDarkGray, CBg, true, false, 0, false, "", false},
	TGrass:        {"grass", '.', CDarkGreen, CBg, true, false, 0, true, "", false},
	TTallGrass:    {"tall grass", '"', CGreen, CBg, true, false, 0, true, "Swaying grass, waist-high.", false},
	TSand:         {"sand", '.', CYellow, CBg, true, false, 0, false, "", false},
	TDirt:         {"dirt", '.', CBrown, CBg, true, false, 0, false, "", false},
	TRoad:         {"road", '∙', CBrown, CBg, true, false, 0, false, "Worn by many feet.", false},
	TSnow:         {"snow", '.', CWhite, CBg, true, false, 0, false, "", false},
	TIce:          {"ice", '░', CCyan, CBg, true, false, 0, false, "Slick ice.", false},
	TAsh:          {"ash", '.', CDarkGray, CBg, true, false, 0, false, "Grey ash, still warm in places.", false},
	TMoss:         {"moss", ',', CDarkGreen, CBg, true, false, 0, false, "", false},
	TFungus:       {"spore mat", ',', CDarkMagenta, CBg, true, false, 0, false, "", false},
	TCrystalFloor: {"crystal sand", ':', CMagenta, CBg, true, false, 0, false, "", false},
	TTree:         {"tree", '♣', CGreen, CBg, false, true, 0, true, "", false},
	TDeadTree:     {"dead tree", 'τ', CBrown, CBg, false, false, 0, true, "", false},
	TMushroom:     {"giant mushroom", '♠', CMagenta, CBg, false, true, 0, false, "A mushroom as tall as a house.", false},
	TBush:         {"brush", '♠', CDarkGreen, CBg, true, false, 0, true, "Thorny brush slows you.", false},
	TReeds:        {"reeds", '"', CDarkCyan, CBg, true, true, 1, true, "", false},
	TRock:         {"rock", '#', CDarkGray, CBgDark, false, true, 0, false, "Living rock.", false},
	TWall:         {"wall", '#', CGray, CBgLight, false, true, 0, false, "", false},
	TDoor:         {"door", '+', CBrown, CBg, true, true, 0, true, "", true},
	TDoorOpen:     {"open door", '/', CBrown, CBg, true, false, 0, true, "", false},
	TCrystal:      {"crystal spire", '*', CMagenta, CBg, false, false, 0, false, "A spire of singing crystal.", false},
	TShallow:      {"shallow water", '~', CBlue, CDarkBlue, true, false, 1, false, "Fresh water. You could drink here (d).", true},
	TDeep:         {"deep water", '≈', CBlue, CDarkBlue, true, false, 2, false, "Too deep to wade. You'd have to swim.", false},
	TLava:         {"lava", '≈', COrange, CDarkRed, true, false, 0, false, "Molten rock. Do not.", false},
	TDown:         {"way down", '>', CWhite, CBg, true, false, 0, false, "A way down into darkness.", false},
	TUp:           {"way up", '<', CWhite, CBg, true, false, 0, false, "A way back up.", false},
	TUpdraft:      {"updraft", '↑', CCyan, CBg, true, false, 0, false, "Ascend with <.", false},
	TWell:         {"well", 'o', CBlue, CBg, false, false, 0, false, "A well of fresh water. Bump to drink and fill.", true},
	TAltar:        {"altar", '╥', CYellow, CBg, false, false, 0, false, "A shrine to the local faith.", true},
	TStall:        {"market stall", 'π', COrange, CBg, false, false, 0, false, "", false},
	TSarcophagus:  {"sarcophagus", '▬', CGray, CBg, false, false, 0, false, "A sealed stone coffin.", true},
	TInscription:  {"inscribed stone", '¶', CYellow, CBg, false, false, 0, false, "Words are carved here. Bump to read.", true},
	TRubble:       {"rubble", '%', CDarkGray, CBg, true, false, 0, false, "", false},
	TBridge:       {"bridge", '=', CBrown, CDarkBlue, true, false, 0, true, "", false},
	TPillar:       {"pillar", 'O', CGray, CBg, false, true, 0, false, "", false},
	TBed:          {"bedroll", '=', CDarkRed, CBg, true, false, 0, true, "", false},
	TShelf:        {"shelf", '≡', CBrown, CBg, false, false, 0, true, "", true},
	TStatue:       {"statue", 'Ω', CWhite, CBg, false, false, 0, false, "", true},
	TRift:         {"rift", '◊', CMagenta, CDarkMagenta, true, false, 0, false, "A tear in the world.", false},
	TCloud:        {"cloud bank", '░', CWhite, CBgLight, true, false, 0, false, "Dense cloud you can stand on, barely.", false},
	TCoral:        {"coral", '¥', CMagenta, CDarkBlue, false, false, 0, false, "", false},
	TKelp:         {"kelp", '"', CGreen, CDarkBlue, true, true, 0, false, "", false},
	TBones:        {"bones", ',', CWhite, CBg, true, false, 0, false, "Old bones.", false},
	TCarpet:       {"carpet", '.', CDarkRed, CBg, true, false, 0, true, "", false},
	TMachine:      {"machinery", 'Φ', CDarkCyan, CBg, false, false, 0, false, "Humming, ancient machinery.", false},
	TBrazier:      {"brazier", 'Ψ', COrange, CBg, false, false, 0, false, "A fire burns here, giving light.", false},
	TTrunk:        {"world-tree trunk", '█', CBrown, CBg, false, true, 0, false, "Bark like a cliff face, rising out of sight.", false},
	TChasm:        {"chasm", ' ', CDarkGray, CBgDark, true, false, 0, false, "A drop into the dark below.", false},
	TThrone:       {"throne", 'Ж', CYellow, CBg, false, false, 0, false, "A throne of state.", false},
	TAnvil:        {"anvil", 'π', CGray, CBg, false, false, 0, false, "A smith's anvil, scarred by work.", false},
	TDummy:        {"training dummy", 'Ŧ', CBrown, CBg, false, false, 0, true, "Straw and sackcloth, hacked to ribbons. Bump to practise.", false},
	TCrate:        {"crates", '■', CBrown, CBg, false, false, 0, true, "Stacked crates and barrels.", false},
	TCrop:         {"crops", '"', CYellow, CBg, true, false, 0, true, "Rows of crops.", false},
	TBoat:         {"boat deck", '=', CBrown, CDarkBlue, true, false, 0, true, "Planks of a moored ship.", false},
	TLocked:       {"locked door", '+', CRed, CBg, false, true, 0, false, "Locked. You need a key.", false},
	TOre:          {"ore vein", '%', COrange, CBgDark, false, true, 0, false, "A seam of ore. Bump to mine it.", false},
	TSeat:         {"stands", '≡', CDarkGray, CBg, false, false, 0, false, "Tiered seating for spectators.", false},
	TCobble:       {"cobbles", '·', CGray, CBg, true, false, 0, false, "", false},
	TTrap:         {"trap", '^', CMagenta, CBg, true, false, 0, false, "A trap you spotted. Step around it.", false},
	TPortal:       {"rift gate", 'Ø', CMagenta, CDarkMagenta, false, false, 0, false, "A shimmering tear onto another world. Bump to step through.", true},
	TCliff:        {"cliff", '▲', CBrown, CBgDark, false, true, 0, false, "A sheer rock face. Find a way round, or climb it.", false},
	TFalls:        {"waterfall", '║', CWhite, CBlue, true, true, 1, false, "Water thunders over the lip. You could slip behind the curtain.", false},
	TSpray:        {"spray", '░', CCyan, CDarkBlue, true, false, 1, false, "Cold mist off the falls. It soaks you through.", false},
	TGlacier:      {"glacier ice", '▒', CWhite, CDarkCyan, true, false, 0, false, "Old blue ice, ridged and slick. Mind the crevasses.", false},
	TIceWall:      {"ice wall", '#', CCyan, CBgLight, false, true, 0, false, "Walls of glassy blue ice.", false},
	TPalm:         {"palm", '♣', CYellow, CBg, false, false, 0, true, "A date palm, its fronds rattling in the wind.", false},
	THotSpring:    {"hot spring", '~', CCyan, CDarkCyan, true, false, 1, false, "Steaming mineral water. A long soak mends aches.", false},
	TGeyser:       {"geyser", '¤', CWhite, CDarkCyan, false, false, 0, false, "A crusted vent that hisses and gurgles. It erupts from time to time; keep clear.", false},
	TFumarole:     {"fumarole", '∴', CYellow, CDarkRed, true, false, 0, false, "A crack venting hot, foul gas.", false},
	TTar:          {"tar pit", '≈', CDarkGray, CBlack, true, false, 0, true, "Black, sucking tar. Slow going, and it burns.", false},
	TQuicksand:    {"quicksand", ':', CBrown, CBgDark, true, false, 0, false, "Wet sand that gives way underfoot. Keep moving.", false},
	TSalt:         {"salt flat", '.', CWhite, CBgLight, true, false, 0, false, "A blinding crust of salt, cracked into plates.", false},
	TObsidian:     {"obsidian", '%', CDarkMagenta, CBgDark, false, true, 0, false, "Black volcanic glass. Bump to knap shards.", true},
	TSulfur:       {"sulfur crust", '%', CYellow, CBgDark, false, true, 0, false, "A crust of yellow sulfur, reeking of rotten eggs. Bump to scrape some.", true},
	TWoodWall:     {"timber wall", '#', CBrown, CBgDark, false, true, 0, true, "Hewn timber logs notched at the corners.", false},
	TBrickWall:    {"brick wall", '#', CRed, CBgDark, false, true, 0, false, "Kiln-fired clay bricks laid in mortar.", false},
	TStoneWall:    {"stone wall", '#', CGray, CBgLight, false, true, 0, false, "Courses of dressed ashlar stone.", false},
	TMarbleWall:   {"marble wall", '#', CWhite, CBgLight, false, true, 0, false, "Polished marble veined with mineral grain.", false},
	TObsidianWall: {"obsidian wall", '#', CDarkMagenta, CBgDark, false, true, 0, false, "Glassy black volcanic stone.", false},
	TBoneWall:     {"bone wall", '#', CYellow, CBgDark, false, true, 0, false, "Ribs, tusks, and hides lashed together.", false},
	TAdobeWall:    {"adobe wall", '#', COrange, CBgDark, false, true, 0, false, "Thick sun-baked mudbrick and ochre plaster.", false},
	TBambooWall:   {"bamboo wall", '#', CGreen, CBgDark, false, true, 0, true, "Joined bamboo canes and lacquered timber.", false},
	TCoralWall:    {"coral wall", '#', CCyan, CBgDark, false, true, 0, false, "Living reef-stone and nacreous shell.", false},
	TMetalWall:    {"riveted wall", '#', CDarkCyan, CBgDark, false, true, 0, false, "Heavy riveted iron plates.", false},
	THedgeWall:    {"thorn hedge", '#', CDarkGreen, CBg, false, true, 0, true, "Interwoven boughs and iron-hard briars.", false},
	TWindow:       {"latticed window", '⊞', CCyan, CBg, false, false, 0, false, "A casement window letting light and sight through.", false},
	TGateArch:     {"gatehouse arch", '∏', CYellow, CBgLight, true, false, 0, false, "A towering stone archway beneath the gatehouse.", false},
	TGateClosed:   {"portcullis", '╫', CDarkGray, CBgDark, false, false, 0, false, "An iron-shod portcullis gate. Bump to open.", true},
	TWoodFloor:    {"plank floor", '=', CBrown, CBg, true, false, 0, true, "Warm timber floorboards.", false},
	TTileFloor:    {"tile floor", '·', CCyan, CBg, true, false, 0, false, "Glazed ceramic and mosaic tiles.", false},
	TScaffold:     {"scaffold", '≡', CBrown, CBg, true, false, 0, true, "Raised timber platform and beams.", false},
	TBarricade:    {"barricade", 'X', CBrown, CBgDark, false, false, 0, true, "A spiked timber barricade. Bump to assault or dismantle.", true},
	TBallista:     {"siege ballista", 'Ψ', CDarkGray, CBgDark, false, false, 0, false, "A mounted siege ballista loaded with iron-tipped bolts. Bump to fire.", true},
	TLantern:      {"paper lantern", '*', CYellow, CBg, true, false, 0, true, "A glowing paper lantern swaying on a cord.", true},
	TBunting:      {"festival bunting", '~', CRed, CBg, true, false, 0, true, "Festive silk ribbons and autumn harvest bunting.", false},
	TSnowDrift:    {"snow drift", '%', CWhite, CBgLight, true, false, 0, false, "Deep banked winter snow.", false},
	TMonument:     {"victory monument", 'Ω', CYellow, CBgLight, false, false, 0, false, "A grand carved monument commemorating legendary deeds. Bump to inspect.", true},
	TSkylight:     {"skylight", '◊', CCyan, CBgLight, true, false, 0, false, "A rooftop glass skylight. Bump to peer down or descend.", true},
	TChimney:      {"chimney", '∩', CBrown, CBgDark, false, false, 0, false, "A soot-stained stone chimney. Bump to shimmy down.", true},
	TBeehive:      {"wild beehive", 'δ', CYellow, CBg, false, false, 0, true, "A humming hive swollen with honey and stinging bees. Bump to knock down.", true},
	TBarrel:       {"cask of oil & wine", 'Θ', CBrown, CBg, false, false, 0, true, "A heavy oak cask sloshing with liquid. Bump to puncture.", true},
	TChandelier:   {"iron chandelier", '¤', CYellow, CBgDark, true, false, 0, false, "A tiered iron chandelier suspended by chains. Bump to leap and swing.", true},
	TTent:         {"yurt felt wall", '#', COrange, CBgDark, false, true, 0, true, "Thick pressed felt and leather wall of a nomad yurt.", false},
	TAshBed:       {"fertile ash bed", '.', CDarkGray, CBgDark, true, false, 0, false, "Deep volcanic and brush ash, rich with soil nutrients.", false},
	TSilkwormRack: {"fungal silk rack", '§', CCyan, CBgDark, false, false, 0, false, "Trays of bioluminescent cavern silkworms spinning glowing thread. Bump to harvest.", true},
	TCoralDome:    {"coral dome", 'O', CCyan, CDarkBlue, false, false, 1, false, "A living bioluminescent coral cultivation dome. Bump to harvest.", true},
}

func (t Terrain) D() *TDef { return &TDefs[t] }
