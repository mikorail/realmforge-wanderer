package engine

// Stealth data tables (prefix ss).

// ssMetal: armour names that clank when you move.
var ssMetal = []string{"mail", "plate", "iron", "steel", "bronze", "chain", "scale", "brigandine", "cuirass", "greave"}

// ssCoverTiles: terrain that hides a crouching figure (bonus to the Stealth rating).
var ssCoverTiles = map[Terrain]int{
	TTallGrass: 3, TBush: 2, TReeds: 2, TTree: 2, TMushroom: 2, TKelp: 2, TCoral: 2, TFungus: 1, TDeadTree: 1, TPillar: 1, TStatue: 1,
}

// ssSoft: ground that holds footprints.
var ssSoft = map[Terrain]bool{
	TGrass: true, TTallGrass: true, TSand: true, TDirt: true, TSnow: true, TAsh: true, TMoss: true, TFungus: true, TSalt: true,
}

// ssAccidentFeatures: fixtures a killer can tamper with to stage an accident.
var ssAccidentFeatures = map[Terrain]string{
	TPillar: "a pillar is chipped loose", TStatue: "a statue is levered off its plinth", TBridge: "a bridge plank is sawn half through",
	TShelf: "a laden shelf is pried from its pegs", TWell: "the well rope is frayed", TCliff: "the cliff edge is undercut",
	TStall: "a stall awning is cut loose", TCrate: "a crate stack is propped to topple", TAnvil: "the anvil bench is sabotaged",
	TBrazier: "a brazier is set to tip", TDown: "the stair rail is loosened", TUp: "the stair rail is loosened",
}

var ssBlameLines = []string{
	"Whispers say %s was seen near the body.", "The watch drags in %s for questioning.", "Suspicion falls on %s.",
}

var ssHideLines = []string{
	"You haul the body into the shadows.", "You roll the body out of sight.", "You stuff the body under cover.",
}
