package engine

import (
	"fmt"
	"strings"
)

// Tribes, Clans & Colored/Subrace Race Variations (prefix tr).
// Uses deterministic side RNGs keyed by world seed + parasang/zone coordinates
// so existing RNG streams and tests remain completely untouched.

const (
	ZWild     = "surface"
	TTotem    = TAltar
	TCampfire = TBrazier
)

// TribeVariant defines a colored/subrace regional or biome variation for a species.
type TribeVariant struct {
	ID        string
	Species   string
	Name      string
	Color     byte
	SkinHex   string
	AccentHex string
	HP        int
	AV        int
	DV        int
	Speed     int
	Elem      string
	Totem     string
	Biomes    []int
}

// trTribe is a deterministically derived regional tribe for a parasang (wx, wy).
type trTribe struct {
	ID      string
	Name    string
	Species string
	Variant TribeVariant
	Totem   string
	WX      int
	WY      int
}

type trTribeState struct {
	Rep        map[string]int  `json:"rep,omitempty"`
	Paint      string          `json:"paint,omitempty"`
	PaintName  string          `json:"pname,omitempty"`
	PaintHex   string          `json:"phex,omitempty"`
	PaintElem  string          `json:"pelem,omitempty"`
	PaintBonus bool            `json:"pbon,omitempty"`
	ChiefOf    map[string]bool `json:"chief,omitempty"`
}

func (g *Game) trTribeSt() *trTribeState {
	st := g.ext("tr_tribes", func() any {
		return &trTribeState{
			Rep:     map[string]int{},
			ChiefOf: map[string]bool{},
		}
	}).(*trTribeState)
	if st.Rep == nil {
		st.Rep = map[string]int{}
	}
	if st.ChiefOf == nil {
		st.ChiefOf = map[string]bool{}
	}
	return st
}

// SeedU64 returns a deterministic uint64 seed from WorldData.
func (w *WorldData) SeedU64() uint64 {
	if w == nil || w.Seed == "" {
		return 0x9e3779b97f4a7c15
	}
	return HashStr(w.Seed)
}

var trVariantTable = []TribeVariant{
	// Orcs
	{ID: "orc_crimson_bloodfang", Species: "orc", Name: "Crimson Bloodfang", Color: CRed, SkinHex: "#b83228", AccentHex: "#ffaa33", HP: 8, AV: 1, DV: 0, Speed: 5, Elem: "fire", Totem: "Blood-Boar", Biomes: []int{1, 2, 13, 16}},
	{ID: "orc_azure_frosttusk", Species: "orc", Name: "Azure Frost-Tusk", Color: CCyan, SkinHex: "#2e7d9b", AccentHex: "#a8e6ff", HP: 7, AV: 2, DV: 0, Speed: 0, Elem: "frost", Totem: "Rime-Wolf", Biomes: []int{9, 10, 11}},
	{ID: "orc_ashskin_magma", Species: "orc", Name: "Ash-Skin Magma", Color: COrange, SkinHex: "#6e473b", AccentHex: "#ff6622", HP: 9, AV: 2, DV: 0, Speed: 0, Elem: "fire", Totem: "Cinder-Drake", Biomes: []int{13, 16}},
	{ID: "orc_obsidian_blackhand", Species: "orc", Name: "Obsidian Blackhand", Color: CDarkGray, SkinHex: "#2b2b30", AccentHex: "#d4af37", HP: 8, AV: 2, DV: 1, Speed: 0, Elem: "", Totem: "Iron-Bear", Biomes: []int{2, 4, 10, 13}},
	{ID: "orc_emerald_mosstusk", Species: "orc", Name: "Emerald Moss-Tusk", Color: CGreen, SkinHex: "#2f6b3c", AccentHex: "#9be564", HP: 6, AV: 1, DV: 1, Speed: 5, Elem: "poison", Totem: "Bog-Boar", Biomes: []int{5, 6, 7, 8, 12}},

	// Goblins
	{ID: "gob_redcap_butcher", Species: "goblin", Name: "Redcap Butcher", Color: CRed, SkinHex: "#9e2a2b", AccentHex: "#ffccd5", HP: 4, AV: 1, DV: 1, Speed: 5, Elem: "", Totem: "Blood-Crow", Biomes: []int{4, 5, 6, 13}},
	{ID: "gob_frostbiter_blue", Species: "goblin", Name: "Frostbiter Blue", Color: CCyan, SkinHex: "#3a86ff", AccentHex: "#caf0f8", HP: 4, AV: 0, DV: 2, Speed: 5, Elem: "frost", Totem: "Ice-Weasel", Biomes: []int{9, 10, 11}},
	{ID: "gob_sunscorch_yellow", Species: "goblin", Name: "Sunscorch Yellow", Color: CYellow, SkinHex: "#d4a017", AccentHex: "#fff3b0", HP: 3, AV: 0, DV: 2, Speed: 10, Elem: "fire", Totem: "Sun-Scorpion", Biomes: []int{1, 2, 3}},
	{ID: "gob_sporegreen_creeper", Species: "goblin", Name: "Spore-Green Creeper", Color: CGreen, SkinHex: "#386641", AccentHex: "#a7c957", HP: 4, AV: 0, DV: 2, Speed: 5, Elem: "poison", Totem: "Spore-Spider", Biomes: []int{7, 8, 12, 15}},
	{ID: "gob_cinderear", Species: "goblin", Name: "Cinder-Ear", Color: COrange, SkinHex: "#bc4749", AccentHex: "#f4a261", HP: 5, AV: 1, DV: 1, Speed: 5, Elem: "fire", Totem: "Ash-Bat", Biomes: []int{13, 16}},

	// Elves (elf, wood_elf, dark_elf, void_elf)
	{ID: "elf_sungold_highkin", Species: "elf", Name: "Sun-Gold High-Kin", Color: CYellow, SkinHex: "#e9c46a", AccentHex: "#fff8d6", HP: 4, AV: 1, DV: 2, Speed: 5, Elem: "fire", Totem: "Sun-Hawk", Biomes: []int{3, 4, 5}},
	{ID: "elf_silvermoon_kin", Species: "elf", Name: "Silver-Moon Kin", Color: CWhite, SkinHex: "#cfd8dc", AccentHex: "#90e0ef", HP: 4, AV: 0, DV: 2, Speed: 5, Elem: "frost", Totem: "Moon-Owl", Biomes: []int{5, 6, 9, 14}},
	{ID: "elf_emerald_wildkin", Species: "elf", Name: "Emerald Wild-Kin", Color: CGreen, SkinHex: "#52b788", AccentHex: "#d8f3dc", HP: 5, AV: 0, DV: 2, Speed: 10, Elem: "poison", Totem: "Great-Stag", Biomes: []int{5, 6, 7, 8}},
	{ID: "elf_crimson_autumnkin", Species: "elf", Name: "Crimson Autumn-Kin", Color: CRed, SkinHex: "#c1121f", AccentHex: "#fdf0d5", HP: 5, AV: 1, DV: 1, Speed: 5, Elem: "fire", Totem: "Ember-Fox", Biomes: []int{4, 5, 6}},
	{ID: "elf_obsidian_duskkin", Species: "elf", Name: "Obsidian Dusk-Kin", Color: CMagenta, SkinHex: "#3c096c", AccentHex: "#e0aaff", HP: 4, AV: 0, DV: 3, Speed: 5, Elem: "shock", Totem: "Phase-Panther", Biomes: []int{13, 14, 15}},

	// Dwarves (dwarf, chaos_dwarf)
	{ID: "dwf_redbeard_forgeclan", Species: "dwarf", Name: "Red-Beard Forge-Clan", Color: CRed, SkinHex: "#b5654d", AccentHex: "#d62828", HP: 8, AV: 2, DV: 0, Speed: 0, Elem: "fire", Totem: "Forge-Ram", Biomes: []int{4, 10, 13, 16}},
	{ID: "dwf_slateskin_deepdelver", Species: "dwarf", Name: "Slate-Skin Deep-Delver", Color: CDarkGray, SkinHex: "#4a4e69", AccentHex: "#c9ada7", HP: 8, AV: 2, DV: 1, Speed: 0, Elem: "", Totem: "Blind-Mole", Biomes: []int{10, 13, 14}},
	{ID: "dwf_rimebeard_frosthold", Species: "dwarf", Name: "Rime-Beard Frost-Hold", Color: CCyan, SkinHex: "#8ecae6", AccentHex: "#ffffff", HP: 7, AV: 2, DV: 0, Speed: 0, Elem: "frost", Totem: "Glacier-Bear", Biomes: []int{9, 10, 11}},
	{ID: "dwf_bronzebrow_sunclan", Species: "dwarf", Name: "Bronze-Brow Sun-Clan", Color: COrange, SkinHex: "#cd7f32", AccentHex: "#ffd166", HP: 7, AV: 1, DV: 1, Speed: 0, Elem: "shock", Totem: "Storm-Roc", Biomes: []int{1, 2, 3, 10}},
	{ID: "dwf_jadevein_stonesinger", Species: "dwarf", Name: "Jade-Vein Stone-Singer", Color: CGreen, SkinHex: "#2a9d8f", AccentHex: "#e9c46a", HP: 7, AV: 2, DV: 0, Speed: 0, Elem: "", Totem: "Crystal-Tortoise", Biomes: []int{6, 10, 14}},

	// Lizardfolk / Saurian
	{ID: "liz_crimson_firescale", Species: "lizardfolk", Name: "Crimson Fire-Scale", Color: CRed, SkinHex: "#a4161a", AccentHex: "#ffba08", HP: 7, AV: 2, DV: 0, Speed: 5, Elem: "fire", Totem: "Cinder-Drake", Biomes: []int{1, 2, 13, 16}},
	{ID: "liz_azure_stormscale", Species: "lizardfolk", Name: "Azure Storm-Scale", Color: CCyan, SkinHex: "#0077b6", AccentHex: "#90e0ef", HP: 6, AV: 1, DV: 1, Speed: 5, Elem: "shock", Totem: "Thunder-Raptor", Biomes: []int{3, 7, 8, 12}},
	{ID: "liz_emerald_venomscale", Species: "lizardfolk", Name: "Emerald Venom-Scale", Color: CGreen, SkinHex: "#2d6a4f", AccentHex: "#95d5b2", HP: 6, AV: 1, DV: 1, Speed: 5, Elem: "poison", Totem: "Spitting-Cobra", Biomes: []int{7, 8, 12, 15}},
	{ID: "liz_golden_sunscale", Species: "lizardfolk", Name: "Golden Sun-Scale", Color: CYellow, SkinHex: "#ee9b00", AccentHex: "#fff3b0", HP: 7, AV: 2, DV: 0, Speed: 0, Elem: "fire", Totem: "Sun-Serpent", Biomes: []int{1, 2, 3, 7}},
	{ID: "liz_alabaster_rimescale", Species: "lizardfolk", Name: "Alabaster Rime-Scale", Color: CWhite, SkinHex: "#e5e5e5", AccentHex: "#a8dadc", HP: 7, AV: 2, DV: 0, Speed: 0, Elem: "frost", Totem: "Frost-Wyrm", Biomes: []int{9, 10, 11}},
	{ID: "liz_obsidian_shadowscale", Species: "lizardfolk", Name: "Obsidian Shadow-Scale", Color: CDarkGray, SkinHex: "#212529", AccentHex: "#9d4edd", HP: 6, AV: 0, DV: 2, Speed: 5, Elem: "poison", Totem: "Night-Basilisk", Biomes: []int{12, 13, 14, 15}},

	// Troll / Ogor
	{ID: "trl_mossback_river", Species: "troll", Name: "Moss-Back River", Color: CGreen, SkinHex: "#3a5a40", AccentHex: "#a3b18a", HP: 12, AV: 2, DV: 0, Speed: 0, Elem: "poison", Totem: "Mire-Toad", Biomes: []int{5, 6, 7, 8, 12}},
	{ID: "trl_glacierhide_blue", Species: "troll", Name: "Glacier-Hide Blue", Color: CCyan, SkinHex: "#457b9d", AccentHex: "#f1faee", HP: 12, AV: 2, DV: 0, Speed: 0, Elem: "frost", Totem: "Rime-Mammoth", Biomes: []int{9, 10, 11}},
	{ID: "trl_cinderhide_magma", Species: "troll", Name: "Cinder-Hide Magma", Color: COrange, SkinHex: "#9b2226", AccentHex: "#ee9b00", HP: 12, AV: 3, DV: 0, Speed: 0, Elem: "fire", Totem: "Magma-Behemoth", Biomes: []int{13, 16}},
	{ID: "trl_chalkbone_cave", Species: "troll", Name: "Chalk-Bone Cave", Color: CWhite, SkinHex: "#d6ccc2", AccentHex: "#8d99ae", HP: 10, AV: 2, DV: 1, Speed: 0, Elem: "", Totem: "Cave-Bear", Biomes: []int{10, 13, 14, 15}},

	// Beastman / Ratfolk
	{ID: "bst_whitefur_albinoseer", Species: "beastman", Name: "White-Fur Albino Seer", Color: CWhite, SkinHex: "#f8f9fa", AccentHex: "#e63946", HP: 5, AV: 0, DV: 2, Speed: 5, Elem: "frost", Totem: "Ghost-Hart", Biomes: []int{9, 10, 11, 14, 15}},
	{ID: "bst_redmane_bloodhorn", Species: "beastman", Name: "Red-Mane Blood-Horn", Color: CRed, SkinHex: "#780000", AccentHex: "#c1121f", HP: 8, AV: 1, DV: 0, Speed: 5, Elem: "fire", Totem: "Blood-Bull", Biomes: []int{2, 3, 4, 13}},
	{ID: "bst_brindle_steppeherd", Species: "beastman", Name: "Brindle Steppe-Herd", Color: CBrown, SkinHex: "#7f4f24", AccentHex: "#ddb892", HP: 6, AV: 0, DV: 1, Speed: 10, Elem: "", Totem: "Steppe-Wolf", Biomes: []int{2, 3, 4}},
	{ID: "bst_blackfur_plagueskulk", Species: "beastman", Name: "Black-Fur Plague-Skulk", Color: CDarkGray, SkinHex: "#2b2d42", AccentHex: "#8ac926", HP: 5, AV: 0, DV: 2, Speed: 10, Elem: "poison", Totem: "Plague-Rat", Biomes: []int{7, 8, 12, 15}},
	{ID: "bst_rustcoat_scrappack", Species: "beastman", Name: "Rust-Coat Scrap-Pack", Color: COrange, SkinHex: "#bb3e03", AccentHex: "#ca6702", HP: 6, AV: 1, DV: 1, Speed: 5, Elem: "shock", Totem: "Razor-Hyena", Biomes: []int{1, 2, 13}},

	// Merfolk
	{ID: "mer_coralred_reef", Species: "merfolk", Name: "Coral-Red Reef", Color: CRed, SkinHex: "#e63946", AccentHex: "#ffb703", HP: 6, AV: 1, DV: 1, Speed: 5, Elem: "fire", Totem: "Lionfish", Biomes: []int{0, 7, 12}},
	{ID: "mer_deepindigo_trench", Species: "merfolk", Name: "Deep-Indigo Trench", Color: CBlue, SkinHex: "#1d3557", AccentHex: "#48cae4", HP: 8, AV: 2, DV: 0, Speed: 0, Elem: "frost", Totem: "Abyssal-Kraken", Biomes: []int{0, 11, 12}},
	{ID: "mer_bioluminescent_glowshoal", Species: "merfolk", Name: "Bioluminescent Glow-Shoal", Color: CCyan, SkinHex: "#00b4d8", AccentHex: "#90e0ef", HP: 5, AV: 0, DV: 2, Speed: 5, Elem: "shock", Totem: "Lantern-Eel", Biomes: []int{0, 12, 14}},
	{ID: "mer_kelpgreen_tide", Species: "merfolk", Name: "Kelp-Green Tide", Color: CGreen, SkinHex: "#2a9d8f", AccentHex: "#a7c957", HP: 6, AV: 0, DV: 2, Speed: 5, Elem: "poison", Totem: "Sea-Serpent", Biomes: []int{0, 6, 8, 12}},

	// Human / Halfling
	{ID: "hum_redochre_sunrunner", Species: "human", Name: "Red-Ochre Sun-Runner", Color: CRed, SkinHex: "#bc4749", AccentHex: "#fcbf49", HP: 5, AV: 0, DV: 1, Speed: 5, Elem: "fire", Totem: "Sun-Lion", Biomes: []int{1, 2, 3}},
	{ID: "hum_woadpainted_bluepelt", Species: "human", Name: "Woad-Painted Blue-Pelt", Color: CCyan, SkinHex: "#1d4e89", AccentHex: "#7dcfb6", HP: 6, AV: 1, DV: 1, Speed: 0, Elem: "frost", Totem: "Rime-Wolf", Biomes: []int{4, 5, 6, 9, 10}},
	{ID: "hum_ashmarked_steppe", Species: "human", Name: "Ash-Marked Steppe", Color: CDarkGray, SkinHex: "#6c757d", AccentHex: "#e56b6f", HP: 6, AV: 1, DV: 0, Speed: 5, Elem: "", Totem: "Steppe-Stallion", Biomes: []int{2, 3, 4, 13}},
	{ID: "hum_ambercloaked_dune", Species: "human", Name: "Amber-Cloaked Dune", Color: CYellow, SkinHex: "#d4a373", AccentHex: "#fefae0", HP: 5, AV: 0, DV: 2, Speed: 5, Elem: "shock", Totem: "Storm-Roc", Biomes: []int{1, 2, 3}},
	{ID: "hum_pinegreen_warden", Species: "human", Name: "Pine-Green Warden", Color: CGreen, SkinHex: "#31572c", AccentHex: "#90a955", HP: 5, AV: 0, DV: 2, Speed: 5, Elem: "poison", Totem: "Forest-Hart", Biomes: []int{5, 6, 7, 8}},

	// Giant
	{ID: "gnt_frostrime_peak", Species: "giant", Name: "Frost-Rime Peak", Color: CCyan, SkinHex: "#a8dadc", AccentHex: "#f1faee", HP: 14, AV: 2, DV: 0, Speed: 0, Elem: "frost", Totem: "Rime-Wolf", Biomes: []int{9, 10, 11}},
	{ID: "gnt_stormcloud_bronze", Species: "giant", Name: "Storm-Cloud Bronze", Color: CYellow, SkinHex: "#ca6702", AccentHex: "#90e0ef", HP: 12, AV: 2, DV: 1, Speed: 0, Elem: "shock", Totem: "Storm-Roc", Biomes: []int{3, 4, 10, 14}},
	{ID: "gnt_magmacore_basalt", Species: "giant", Name: "Magma-Core Basalt", Color: COrange, SkinHex: "#370617", AccentHex: "#f48c06", HP: 14, AV: 3, DV: 0, Speed: 0, Elem: "fire", Totem: "Cinder-Drake", Biomes: []int{13, 16}},

	// Celestial
	{ID: "cel_dawngold_seraph", Species: "celestial", Name: "Dawn-Gold Seraph", Color: CYellow, SkinHex: "#ffd166", AccentHex: "#ffffff", HP: 6, AV: 1, DV: 2, Speed: 5, Elem: "fire", Totem: "Sun-Phoenix", Biomes: []int{1, 3, 4, 14}},
	{ID: "cel_starsapphire_astral", Species: "celestial", Name: "Star-Sapphire Astral", Color: CCyan, SkinHex: "#48cae4", AccentHex: "#caf0f8", HP: 6, AV: 0, DV: 2, Speed: 5, Elem: "shock", Totem: "Star-Swan", Biomes: []int{9, 10, 11, 14}},
	{ID: "cel_silverhalo_choir", Species: "celestial", Name: "Silver-Halo Choir", Color: CWhite, SkinHex: "#f8f9fa", AccentHex: "#ffd166", HP: 7, AV: 2, DV: 1, Speed: 0, Elem: "frost", Totem: "Silver-Eagle", Biomes: []int{4, 5, 10, 14}},

	// Tomb Undead
	{ID: "und_gilded_lapis", Species: "tomb_undead", Name: "Gilded Lapis Dynasty", Color: CYellow, SkinHex: "#d4af37", AccentHex: "#1d3557", HP: 8, AV: 2, DV: 0, Speed: 0, Elem: "fire", Totem: "Sun-Scarab", Biomes: []int{1, 2, 13}},
	{ID: "und_verdigris_bronze", Species: "tomb_undead", Name: "Verdigris Bronze-Crypt", Color: CGreen, SkinHex: "#40916c", AccentHex: "#b7e4c7", HP: 7, AV: 2, DV: 0, Speed: 0, Elem: "poison", Totem: "Crypt-Asp", Biomes: []int{4, 8, 12, 15}},
	{ID: "und_bleached_saltbarrow", Species: "tomb_undead", Name: "Bleached Salt-Barrow", Color: CWhite, SkinHex: "#e9ecef", AccentHex: "#adb5bd", HP: 7, AV: 1, DV: 1, Speed: 0, Elem: "frost", Totem: "Barrow-Hound", Biomes: []int{1, 2, 9, 10, 11}},

	// Warp Fiend
	{ID: "wrp_magenta_aether", Species: "warp_fiend", Name: "Magenta Aether-Flayed", Color: CMagenta, SkinHex: "#b5179e", AccentHex: "#4cc9f0", HP: 7, AV: 1, DV: 2, Speed: 5, Elem: "shock", Totem: "Void-Mantis", Biomes: []int{13, 14, 15, 16}},
	{ID: "wrp_brimstone_brasshorn", Species: "warp_fiend", Name: "Brimstone Brass-Horn", Color: CRed, SkinHex: "#9d0208", AccentHex: "#ffba08", HP: 9, AV: 2, DV: 0, Speed: 0, Elem: "fire", Totem: "Hell-Hound", Biomes: []int{1, 13, 16}},
	{ID: "wrp_viridian_bilespawn", Species: "warp_fiend", Name: "Viridian Bile-Spawn", Color: CGreen, SkinHex: "#55a630", AccentHex: "#d4d700", HP: 9, AV: 2, DV: 0, Speed: 0, Elem: "poison", Totem: "Blight-Fly", Biomes: []int{7, 8, 12, 15}},

	// Chitin Swarm
	{ID: "chi_amber_carapace", Species: "chitin_swarm", Name: "Amber-Carapace Locust", Color: CYellow, SkinHex: "#ee9b00", AccentHex: "#9b2226", HP: 6, AV: 2, DV: 0, Speed: 5, Elem: "fire", Totem: "Sun-Mantis", Biomes: []int{1, 2, 3}},
	{ID: "chi_iridescent_scarab", Species: "chitin_swarm", Name: "Iridescent Scarab-Caste", Color: CCyan, SkinHex: "#0096c7", AccentHex: "#72efdd", HP: 7, AV: 2, DV: 1, Speed: 0, Elem: "shock", Totem: "Jewel-Beetle", Biomes: []int{5, 7, 8, 14}},
	{ID: "chi_crimson_broodstalker", Species: "chitin_swarm", Name: "Crimson Brood-Stalker", Color: CRed, SkinHex: "#6a040f", AccentHex: "#e85d04", HP: 6, AV: 1, DV: 1, Speed: 5, Elem: "poison", Totem: "Spore-Spider", Biomes: []int{7, 8, 12, 15}},
}

func trCanonicalSpecies(sp string) string {
	sp = strings.ToLower(strings.TrimSpace(sp))
	switch sp {
	case "wood_elf", "dark_elf", "void_elf", "high_elf":
		return "elf"
	case "chaos_dwarf", "duergar":
		return "dwarf"
	case "saurian", "drake", "reptilian":
		return "lizardfolk"
	case "ogor", "ogre":
		return "troll"
	case "ratfolk", "skaven":
		return "beastman"
	case "halfling", "gnome":
		return "human"
	case "undead", "skeleton", "wight", "ghoul":
		return "tomb_undead"
	case "insect", "mantis", "crawler":
		return "chitin_swarm"
	case "eldritch", "demon", "fiend":
		return "warp_fiend"
	}
	return sp
}

// trVariantsForSpecies returns all colored/subrace TribeVariants for a species.
func trVariantsForSpecies(sp string) []TribeVariant {
	orig := strings.ToLower(strings.TrimSpace(sp))
	canon := trCanonicalSpecies(orig)
	var out []TribeVariant
	for _, v := range trVariantTable {
		if v.Species == canon {
			vc := v
			if orig != "" {
				vc.Species = orig
			}
			out = append(out, vc)
		}
	}
	if len(out) == 0 {
		for _, v := range trVariantTable {
			if v.Species == "human" {
				vc := v
				if orig != "" {
					vc.Species = orig
				}
				out = append(out, vc)
			}
		}
	}
	return out
}

// trPickVariant deterministically chooses a TribeVariant for a species and biome.
func trPickVariant(sp string, biome int, r *Rng) TribeVariant {
	vars := trVariantsForSpecies(sp)
	if r == nil {
		r = NewRng(fmt.Sprintf("tr_pick:%s:%d", sp, biome))
	}
	var fits []TribeVariant
	for _, v := range vars {
		for _, b := range v.Biomes {
			if b == biome {
				fits = append(fits, v)
				break
			}
		}
	}
	if len(fits) > 0 {
		return Pick(r, fits)
	}
	return Pick(r, vars)
}

func trHasVar(a *Actor) bool {
	if a == nil || a.Flags == nil {
		return false
	}
	for k, v := range a.Flags {
		if v && strings.HasPrefix(k, "tr_var:") {
			return true
		}
	}
	return false
}

// trApplyVariant tints and styles an Actor with a TribeVariant's color, stats, hex tags, and totem.
func trApplyVariant(a *Actor, v TribeVariant) {
	if a == nil || v.ID == "" {
		return
	}
	a.F = v.Color
	a.MaxHP = maxi(1, a.MaxHP+v.HP)
	a.HP = maxi(1, a.HP+v.HP)
	a.AVNat += v.AV
	a.DVNat += v.DV
	a.Speed += v.Speed
	if v.Elem != "" {
		a.Set(v.Elem)
		a.Set("tr_elem:" + v.Elem)
		for _, bp := range a.Body {
			if bp.Natural != nil && bp.Natural.Elem == "" {
				bp.Natural.Elem = v.Elem
			}
			if bp.Item != nil && (bp.Item.Kind == "weapon" || bp.Item.Kind == "ranged") && bp.Item.Elem == "" {
				bp.Item.Elem = v.Elem
			}
		}
	}
	a.Set("tr_var:" + v.ID)
	a.Set("tr_hex:" + v.SkinHex + ":" + v.AccentHex)
	a.Set("tr_totem:" + v.Totem)
	if a.Species == "" && v.Species != "" {
		a.Species = v.Species
	}
	if !strings.Contains(a.Name, v.Name) {
		if a.Name == "" {
			a.Name = v.Name
		} else {
			a.Name = v.Name + " " + a.Name
		}
	}
	tag := fmt.Sprintf("Bearing %s war-paint (%s totem).", v.Name, v.Totem)
	if !strings.Contains(a.Desc, v.Totem) {
		if a.Desc == "" {
			a.Desc = tag
		} else {
			a.Desc = strings.TrimSpace(a.Desc) + " " + tag
		}
	}
}

var trDefaultSpecies = []string{
	"orc", "goblin", "elf", "dwarf", "lizardfolk", "troll", "beastman", "human",
}

// trTribeForAt deterministically derives a regional tribe for a parasang (wx, wy).
func trTribeForAt(w *WorldData, wx, wy int) trTribe {
	seed := uint64(wx*73856093^wy*19349663) ^ w.SeedU64() ^ 0x72696265
	r := &Rng{S: seed | 1}
	biome := 4
	sp := ""
	if w != nil && w.In(wx, wy) {
		p := w.Idx(wx, wy)
		if p >= 0 && p < len(w.Biome) {
			biome = w.Biome[p]
		}
		if p >= 0 && p < len(w.Culture) {
			if cu := w.cultByID[w.Culture[p]]; cu != nil && cu.Species != "" {
				sp = cu.Species
			}
		}
		if sp == "" && len(w.Species) > 0 {
			sp = w.Species[r.Int(0, len(w.Species)-1)].ID
		}
	}
	if sp == "" {
		switch biome {
		case 0:
			sp = "merfolk"
		case 1, 2, 7, 8, 12:
			sp = Pick(r, []string{"lizardfolk", "orc", "goblin", "beastman"})
		case 9, 10, 11:
			sp = Pick(r, []string{"dwarf", "orc", "troll", "giant"})
		case 13, 16:
			sp = Pick(r, []string{"orc", "goblin", "dwarf", "warp_fiend"})
		default:
			sp = Pick(r, trDefaultSpecies)
		}
	}
	v := trPickVariant(sp, biome, r)
	id := fmt.Sprintf("%s_%d_%d", v.ID, wx, wy)
	name := fmt.Sprintf("%s %s Clan", v.Name, v.Totem)
	return trTribe{
		ID:      id,
		Name:    name,
		Species: sp,
		Variant: v,
		Totem:   v.Totem,
		WX:      wx,
		WY:      wy,
	}
}

func trSideRng(w *WorldData, wx, wy, z int) *Rng {
	s := uint64(wx*73856093^wy*19349663^(z+17)*83492791) ^ w.SeedU64() ^ 0x72696265
	return &Rng{S: s | 1}
}

func trSpeciesForActor(a *Actor, fallback string) string {
	if a.Species != "" {
		return a.Species
	}
	switch a.Tpl {
	case "bandit", "archer", "chief", "cultist", "npc":
		if fallback != "" {
			return fallback
		}
		return "human"
	case "troglodyte":
		return "goblin"
	case "skeleton", "ghoul", "wight", "wraith":
		return "tomb_undead"
	case "crawler", "mantis", "centipede", "scorpion":
		return "chitin_swarm"
	case "horror":
		return "warp_fiend"
	}
	return ""
}

// trSpawnCamp spawns a Tribal Encampment in zn with a central Totem, 3-4 tribal warriors,
// a Tribal Shaman, and a Tribal Chieftain styled with the regional TribeVariant.
func trSpawnCamp(g *Game, zn *Zone, r2 *Rng) {
	if g == nil || zn == nil {
		return
	}
	if r2 == nil {
		r2 = trSideRng(g.W, zn.WX, zn.WY, zn.Z)
	}
	tribe := trTribeForAt(g.W, zn.WX, zn.WY)
	cx := clampi(MW/2+r2.Int(-12, 12), 8, MW-9)
	cy := clampi(MH/2+r2.Int(-4, 4), 5, MH-6)

	for dy := -2; dy <= 2; dy++ {
		for dx := -3; dx <= 3; dx++ {
			nx, ny := cx+dx, cy+dy
			if !zn.In(nx, ny) {
				continue
			}
			t := zn.At(nx, ny)
			if t == TDown || t == TUp || t == TUpdraft || t == TCliff || t == TLava {
				continue
			}
			zn.Set(nx, ny, TDirt)
		}
	}
	zn.Set(cx, cy, TTotem)
	if zn.Notes == nil {
		zn.Notes = map[int]string{}
	}
	zn.Notes[cy*MW+cx] = fmt.Sprintf("Tribal Totem of the %s (%s). Carved with %s symbols.", tribe.Name, tribe.Totem, tribe.Variant.Name)
	if zn.In(cx+1, cy) && zn.At(cx+1, cy) == TDirt {
		zn.Set(cx+1, cy, TCampfire)
	}

	lvl := maxi(1, zn.Level)
	fac := "tribe:" + tribe.ID
	friendly := g.trTribeSt().Rep[tribe.ID] > 0 || g.Rep[tribe.ID] > 0 || g.Rep[fac] > 0

	placeMember := func(tpl, role, baseName string, boss bool, ox, oy int) *Actor {
		a := g.Spawn(tpl, lvl, r2)
		a.Role = role
		a.Boss = boss
		a.Species = tribe.Species
		a.Faction = fac
		if friendly {
			a.AI = "town"
		} else {
			a.AI = "wander"
		}
		a.Name = baseName
		a.Set("tr_member:" + tribe.ID)
		a.Set("pack")
		trApplyVariant(a, tribe.Variant)
		tx, ty := clampi(cx+ox, 2, MW-3), clampi(cy+oy, 2, MH-3)
		if !zn.Walkable(tx, ty) || zn.ActorAt(tx, ty) != nil {
			if fx, fy, ok := zn.FreeSpot(r2, func(x, y int) bool { return abs(x-cx) <= 5 && abs(y-cy) <= 3 }); ok {
				tx, ty = fx, fy
			}
		}
		a.X, a.Y = tx, ty
		a.HomeX, a.HomeY = tx, ty
		zn.Actors = append(zn.Actors, a)
		return a
	}

	nWarriors := r2.Int(3, 4)
	offsets := [][2]int{{-2, -1}, {2, -1}, {-2, 1}, {2, 1}}
	for i := 0; i < nWarriors; i++ {
		tp := "bandit"
		if i%2 == 1 {
			tp = "archer"
		}
		off := offsets[i%len(offsets)]
		placeMember(tp, "tr_warrior", "Tribal Warrior", false, off[0], off[1])
	}

	shaman := placeMember("cultist", "tr_shaman", "Tribal Shaman", false, -1, -1)
	if shaman != nil {
		shaman.Desc += " Keeper of the " + tribe.Totem + " rites and sacred pigments."
	}

	chief := placeMember("chief", "tr_chief", "Tribal Chieftain", true, 0, -1)
	if chief != nil {
		chief.MaxHP = maxi(chief.MaxHP, 36+lvl*5)
		chief.HP = chief.MaxHP
		chief.Desc += " Warlord of the " + tribe.Name + "."
		totemGear := &Item{
			ID:     g.newID(),
			Name:   tribe.Totem + " Totem-Axe",
			Glyph:  ')',
			F:      tribe.Variant.Color,
			Kind:   "weapon",
			Slot:   "hand",
			Skill:  "axe",
			DN:     2,
			DS:     6,
			PV:     5 + lvl/3,
			MaxStr: 4,
			Elem:   tribe.Variant.Elem,
			Value:  120 + lvl*25,
			Rarity: "rare",
		}
		if bp := chief.PartFor("hand"); bp != nil {
			bp.Item = totemGear
		}
	}
}

func trZoneHook(g *Game, zn *Zone, _ *Rng) {
	if g == nil || zn == nil || g.W == nil {
		return
	}
	r2 := trSideRng(g.W, zn.WX, zn.WY, zn.Z)
	tribe := trTribeForAt(g.W, zn.WX, zn.WY)

	for i, a := range zn.Actors {
		if a == nil || trHasVar(a) {
			continue
		}
		if a.Proper || a.Boss || a.Role != "" || a.SimMon > 0 || a.SimPer > 0 || a.Quest > 0 || strings.HasPrefix(a.Name, "Brood of ") || strings.HasPrefix(a.Tpl, "bs_") {
			continue
		}
		sp := trSpeciesForActor(a, tribe.Species)
		if sp == "" {
			continue
		}
		ar := &Rng{S: (r2.S ^ uint64(i+1)*0x9e3779b97f4a7c15) | 1}
		v := trPickVariant(sp, zn.Biome, ar)
		trApplyVariant(a, v)
	}

	if (zn.Kind == ZWild || zn.Kind == "wild") && zn.Z == 0 && !zn.Underwater && !zn.Sky && g.W.In(zn.WX, zn.WY) {
		p := g.W.Idx(zn.WX, zn.WY)
		if g.W.Burg[p] == 0 && g.W.Road[p] == 0 && g.W.River[p] == 0 && !g.W.IsWater(p) && !strings.Contains(zn.Name, "Crater") {
			if (zn.ZX != 1 || zn.ZY != 1) && (r2.S%23) == 0 {
				trSpawnCamp(g, zn, r2)
			}
		}
	}
}

// trTribeOfActor returns the tribe ID of a tribal actor, or "" if none.
func trTribeOfActor(a *Actor) string {
	if a == nil {
		return ""
	}
	for k, v := range a.Flags {
		if v && strings.HasPrefix(k, "tr_member:") {
			return strings.TrimPrefix(k, "tr_member:")
		}
	}
	if strings.HasPrefix(a.Faction, "tribe:") {
		return strings.TrimPrefix(a.Faction, "tribe:")
	}
	return ""
}

func trVariantOfActor(g *Game, a *Actor) TribeVariant {
	if a != nil {
		for k, v := range a.Flags {
			if v && strings.HasPrefix(k, "tr_var:") {
				id := strings.TrimPrefix(k, "tr_var:")
				for _, tv := range trVariantTable {
					if tv.ID == id {
						return tv
					}
				}
			}
		}
		if a.Species != "" {
			vs := trVariantsForSpecies(a.Species)
			if len(vs) > 0 {
				return vs[0]
			}
		}
	}
	if g != nil && g.Cur != nil {
		return trTribeForAt(g.W, g.Cur.WX, g.Cur.WY).Variant
	}
	return trVariantTable[0]
}

// trParley offers 25 gold tribute to parley with a tribe (+25 standing, makes tribe friendly in zone).
func (g *Game) trParley(a *Actor) bool {
	if g == nil || g.P == nil || a == nil {
		return false
	}
	tid := trTribeOfActor(a)
	if tid == "" {
		tid = "local_tribe"
	}
	if g.P.Gold < 25 {
		g.Msg("You need 25 gold to offer tribute.")
		return false
	}
	g.P.Gold -= 25
	st := g.trTribeSt()
	st.Rep[tid] += 25
	if g.Rep == nil {
		g.Rep = map[string]int{}
	}
	g.Rep[tid] += 25
	if a.Faction != "" {
		g.Rep[a.Faction] += 25
	}
	a.Angry = false
	a.AI = "town"
	if g.Cur != nil {
		for _, o := range g.Cur.Actors {
			if o != nil && (trTribeOfActor(o) == tid || (a.Faction != "" && o.Faction == a.Faction)) {
				o.Angry = false
				o.AI = "town"
			}
		}
	}
	g.Msg(fmt.Sprintf("{{G|%s accepts your tribute of 25 gold. The clan stands at peace with you (+25 standing).}}", g.The(a)))
	return true
}

// trApplyWarPaint applies tribal war-paint (15 gold): grants "tr_paint:"+v.ID, +1 AV, +1 DV,
// "tr_hex:" tint on g.P, and elemental resistance tag.
func (g *Game) trApplyWarPaint(v TribeVariant) bool {
	if g == nil || g.P == nil || v.ID == "" {
		return false
	}
	if g.P.Gold >= 15 {
		g.P.Gold -= 15
	}
	st := g.trTribeSt()
	if g.P.Flags != nil {
		for k := range g.P.Flags {
			if strings.HasPrefix(k, "tr_paint:") || strings.HasPrefix(k, "tr_hex:") || strings.HasPrefix(k, "tr_res:") {
				delete(g.P.Flags, k)
			}
		}
	}
	if !st.PaintBonus {
		g.P.AVNat++
		g.P.DVNat++
		st.PaintBonus = true
	}
	g.P.Set("tr_paint:" + v.ID)
	g.P.Set("tr_hex:" + v.SkinHex + ":" + v.AccentHex)
	if v.Elem != "" {
		g.P.Set("tr_res:" + v.Elem)
		g.P.Set(v.Elem)
	} else {
		g.P.Set("tr_res:physical")
	}
	st.Paint = v.ID
	st.PaintName = v.Name
	st.PaintHex = v.SkinHex + ":" + v.AccentHex
	st.PaintElem = v.Elem
	g.Msg(fmt.Sprintf("{{Y|Shamanic pigments of %s (%s) are painted across your skin (+1 AV, +1 DV).}}", v.Name, v.Totem))
	return true
}

// trChallengeChief challenges a Tribal Chieftain to a Rite of Strength.
func (g *Game) trChallengeChief(a *Actor) bool {
	if g == nil || g.P == nil || a == nil {
		return false
	}
	tid := trTribeOfActor(a)
	if tid == "" {
		tid = "local_tribe"
	}
	v := trVariantOfActor(g, a)
	totem := v.Totem
	if totem == "" {
		totem = "Blood-Boar"
	}
	score := g.P.Score(STR) + g.P.Level*2 + g.Skills["longblade"]*2 + g.Skills["axe"]*2 + g.Skills["cudgel"]*2
	dc := maxi(10, a.Level+8)
	if score < dc && g.R != nil && g.R.Int(1, 20)+g.P.M(STR) < 10 {
		g.Msg(fmt.Sprintf("{{r|%s overpowers you in the Rite of Strength!}}", g.The(a)))
		return false
	}
	st := g.trTribeSt()
	st.ChiefOf[tid] = true
	st.Rep[tid] += 100
	if g.Rep == nil {
		g.Rep = map[string]int{}
	}
	g.Rep[tid] += 100
	if a.Faction != "" {
		g.Rep[a.Faction] += 100
	}
	g.P.Set("tr_bloodchief:" + tid)
	a.Angry = false
	a.AI = "town"
	if g.Cur != nil {
		for _, o := range g.Cur.Actors {
			if o != nil && trTribeOfActor(o) == tid {
				o.Angry = false
				o.AI = "town"
			}
		}
	}
	relic := &Item{
		ID:     g.newID(),
		Name:   fmt.Sprintf("Totem Relic of the %s", totem),
		Glyph:  '*',
		F:      v.Color,
		Kind:   "artifact",
		Slot:   "back",
		AV:     2,
		DV:     2,
		Value:  350,
		Weight: 2,
		Tier:   4,
		Ident:  true,
		Rarity: "unique",
		Elem:   v.Elem,
		Bonus:  map[int]int{STR: 2, WIL: 2},
		Lore:   fmt.Sprintf("Sacred clan totem won in the Rite of Strength from %s.", a.Name),
	}
	g.give(relic)
	g.Msg(fmt.Sprintf("{{G|You triumph in the Rite of Strength! You are hailed as Blood-Chief (+100 standing) and receive the %s!}}", relic.Name))
	return true
}

func trTalkHook(g *Game, a *Actor) []Option {
	if g == nil || a == nil {
		return nil
	}
	tid := trTribeOfActor(a)
	if tid == "" && a.Role != "tr_chief" && a.Role != "tr_shaman" && a.Role != "tr_warrior" {
		return nil
	}
	v := trVariantOfActor(g, a)
	opts := []Option{
		{
			Text: "Offer tribute & parley (25 gold)",
			Do: func() {
				g.trParley(a)
				g.Mode = "main"
			},
		},
		{
			Text: "Receive tribal war-paint (15 gold)",
			Do: func() {
				g.trApplyWarPaint(v)
				g.Mode = "main"
			},
		},
	}
	if a.Role == "tr_chief" {
		opts = append(opts, Option{
			Text: "Challenge the Chieftain to a Rite of Strength",
			Do: func() {
				g.trChallengeChief(a)
				g.Mode = "main"
			},
		})
	}
	return opts
}

func trSideHook(g *Game) []string {
	if g == nil || g.P == nil {
		return nil
	}
	st := g.trTribeSt()
	var out []string
	if st.Paint != "" {
		label := st.PaintName
		if label == "" {
			label = st.Paint
		}
		out = append(out, "{{Y|War-paint: "+label+"}}")
	} else if g.P.Flags != nil {
		for k, v := range g.P.Flags {
			if v && strings.HasPrefix(k, "tr_paint:") {
				out = append(out, "{{Y|War-paint: "+strings.TrimPrefix(k, "tr_paint:")+"}}")
				break
			}
		}
	}
	if len(st.ChiefOf) > 0 {
		out = append(out, "{{R|Title: Blood-Chief}}")
	} else if g.P.Flags != nil {
		for k, v := range g.P.Flags {
			if v && strings.HasPrefix(k, "tr_bloodchief:") {
				out = append(out, "{{R|Title: Blood-Chief}}")
				break
			}
		}
	}
	return out
}

func init() {
	unLocalKeys = append(unLocalKeys, "tr_tribes")
	zoneHooks = append(zoneHooks, trZoneHook)
	talkHooks = append(talkHooks, trTalkHook)
	sideHooks = append(sideHooks, trSideHook)
}
