package engine

// Genre skins for the court and its factions (politics.go). Office order everywhere:
// 0 chancellor (diplomacy), 1 treasurer (stewardship), 2 marshal (martial), 3 spymaster (intrigue), 4 prelate (faith).
// Faction order: crown, sword, purse, altar, street.

type poSkin struct {
	Council string
	Ruler   string
	Offices [5]string
	Fac     [5]string
}

var poSkins = map[string]poSkin{
	"fantasy": {"the Royal Court", "Sovereign", [5]string{"Chancellor", "Exchequer", "Marshal", "Spymaster", "High Prelate"},
		[5]string{"The Crown Party", "The Sword Lords", "The Merchant Guilds", "The Temple Party", "The Commons"}},
	"mythic": {"the Hall of Heroes", "Basileus", [5]string{"Herald", "Steward of Tribute", "Strategos", "Keeper of Whispers", "Oracle"},
		[5]string{"The Hearth of Kings", "The Spear-Friends", "The Tribute Houses", "The Oracle's Voice", "The Folk of the Fields"}},
	"steampunk": {"the Cabinet", "Head of State", [5]string{"Minister of State", "Chancellor of the Exchequer", "Field Marshal", "Director of Intelligence", "Lord Bishop"},
		[5]string{"The Court Party", "The Officers' Club", "The Chamber of Commerce", "The Established Church", "The Chartist Leagues"}},
	"dieselpunk": {"the Cabinet", "Premier", [5]string{"Foreign Secretary", "Treasury Secretary", "Chief of Staff", "Bureau Chief", "Chaplain-General"},
		[5]string{"The Party Machine", "The General Staff", "The Syndicates", "The Pulpit", "The Union Halls"}},
	"atompunk": {"the Presidium", "First Secretary", [5]string{"Foreign Minister", "Planning Commissar", "Strategic Commander", "Security Director", "Chief Ideologue"},
		[5]string{"The Presidium Clique", "The Strategic Command", "The Planning Bureau", "The Ideological Bureau", "The Worker Councils"}},
	"cyberpunk": {"the Board", "Chair", [5]string{"Chief Executive", "Chief Financial Officer", "Head of Security", "Head of Black Ops", "Compliance Officer"},
		[5]string{"The Executive Floor", "Security Division", "The Shareholders", "Compliance & Ethics", "The Undercity Collectives"}},
	"biopunk": {"the Conclave of Strains", "Prime Strain", [5]string{"Prime Envoy", "Vat Steward", "War Cultivar", "Silent Gardener", "Geneticist-Priest"},
		[5]string{"The Prime Lineage", "The War Cultivars", "The Vat Consortium", "The Splice Priesthood", "The Unmodified"}},
	"solarpunk": {"the Assembly", "Speaker", [5]string{"Envoy of the Commons", "Steward of the Commons", "Warden of the Watch", "Keeper of Quiet Words", "Grove Keeper"},
		[5]string{"The Speaker's Circle", "The Watch Wardens", "The Cooperatives", "The Grove Keepers", "The Open Door Movement"}},
	"wasteland": {"the Elders' Council", "Warchief", [5]string{"Elder of Words", "Keeper of Stores", "War Boss", "Eyes of the Elders", "Rite Keeper"},
		[5]string{"The Old Blood", "The Raiders", "The Cache Keepers", "The Rite Keepers", "The Scavengers"}},
	"twilight": {"the Dusk Court", "Sovereign of Dusk", [5]string{"Seneschal", "Keeper of the Tithe", "Warden of the Dusk", "Whisperer", "Sepulchral Priest"},
		[5]string{"The Pale Throne", "The Dusk Wardens", "The Tithe Houses", "The Sepulchral Order", "The Hollow Folk"}},
}

func poSkinFor(genre string) poSkin {
	if s, ok := poSkins[genre]; ok {
		return s
	}
	return poSkins["fantasy"]
}

// poCouncilByIdeology renames the council in genres where the skin's is generic.
var poCouncilByIdeology = map[string]string{
	"theocracy": "the Synod", "republic": "the Senate", "oligarchy": "the Council of Houses",
	"communal": "the Moot", "technocracy": "the Directorate", "autocracy": "the Inner Circle",
}

var poOfficeFac = [5]int{0, 2, 1, 0, 3} // office index → faction kind index

var poOfficeAbout = [5]string{
	"treaties, embassies and the realm's good name",
	"the levy, the tax rolls and the treasury",
	"the levy of soldiers and the conduct of war",
	"informers, plots and what is whispered",
	"doctrine, rites and the quarrels of the faithful",
}

var poSecrets = []string{
	"%s keeps a second set of books.",
	"%s has a child nobody at court has been told of.",
	"%s sold a pass through the frontier to the enemy.",
	"%s owes a ruinous debt to a moneylender in the undercity.",
	"%s forged the seal on a famous decree.",
	"%s meets a foreign envoy in secret every new moon.",
	"%s hides a vice that would end a career.",
}

var poRankName = []string{"Deputy", "Senior Deputy", "Second"}

var poFactionKinds = [5]string{"crown", "sword", "purse", "altar", "street"}

var poFactionBent = map[string]string{
	"crown":  "wants order and an unbroken line of rule",
	"sword":  "wants war, levies and a free hand for the officers",
	"purse":  "wants low tolls, open roads and quiet",
	"altar":  "wants doctrine obeyed and the faithful protected",
	"street": "wants the old order broken and the people heard",
	"heir":   "wants its claimant on the seat",
}
