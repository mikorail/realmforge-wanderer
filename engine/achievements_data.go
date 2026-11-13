package engine

import "strings"

// Achievement catalogue (state key "ac"; types and helpers use the prefix ach). Every entry unlocks when
// Get(g, st) >= Need. Getters read existing, stable state (kills, deeds, skills, family, alchemy,
// collections, quests, gold) and the counters this feature keeps from kill and enter hooks.

type achDef struct {
	ID, Name, Desc, Cat string
	Title               string // a title that comes with it ("" = badge only)
	Need                int
	Get                 func(g *Game, st *acState) int
}

func achB(b bool) int {
	if b {
		return 1
	}
	return 0
}

func achKinds(kinds ...string) func(g *Game, st *acState) int {
	return func(g *Game, st *acState) int {
		n := 0
		for _, k := range kinds {
			n += st.K[k]
		}
		return n
	}
}

func achC(key string) func(g *Game, st *acState) int {
	return func(g *Game, st *acState) int { return st.C[key] }
}

func achSkillMax(g *Game, st *acState) int {
	m := 0
	for _, v := range g.Skills {
		m = maxi(m, v)
	}
	return m
}

func achSkillSum(g *Game, st *acState) int {
	n := 0
	for _, v := range g.Skills {
		n += v
	}
	return n
}

func achRepMax(g *Game, st *acState) int {
	m := 0
	for _, v := range g.Rep {
		m = maxi(m, v)
	}
	return m
}

func achRepMin(g *Game, st *acState) int {
	m := 0
	for _, v := range g.Rep {
		m = mini(m, v)
	}
	return -m
}

func achQuestsClosed(g *Game, st *acState) int {
	n := 0
	for _, q := range g.Quests {
		if q.Closed && q.Done {
			n++
		}
	}
	return n
}

func achMatch(s string, words ...string) bool {
	s = strings.ToLower(s)
	for _, w := range words {
		if strings.Contains(s, w) {
			return true
		}
	}
	return false
}

var achDefs = []achDef{
	// ---- combat
	{"first_blood", "First Blood", "Slay your first creature.", "Combat", "", 1, func(g *Game, st *acState) int { return g.Kills }},
	{"slayer", "Slayer", "Slay 25 creatures.", "Combat", "", 25, func(g *Game, st *acState) int { return g.Kills }},
	{"butcher", "Butcher of Bones", "Slay 100 creatures.", "Combat", "the Butcher", 100, func(g *Game, st *acState) int { return g.Kills }},
	{"legion_bane", "Legion's Bane", "Slay 500 creatures.", "Combat", "Legion's Bane", 500, func(g *Game, st *acState) int { return g.Kills }},
	{"giant_killer", "Giant-Killer", "Bring down a boss.", "Combat", "", 1, achC("boss")},
	{"tyrant_dream", "Tyrant's Nightmare", "Bring down five bosses.", "Combat", "the Tyrant-Slayer", 5, achC("boss")},
	{"dragonbane", "Dragonbane", "Slay a dragon, wyrm or drake.", "Combat", "Dragonbane", 1, achC("dragon")},
	{"grave_quieter", "Grave-Quieter", "Put ten of the walking dead back down.", "Combat", "", 10, achC("undead")},
	{"beast_hunter", "Beast Hunter", "Slay 30 wild beasts.", "Combat", "", 30, achC("beast")},
	{"manhunter", "Manhunter", "Slay 15 people.", "Combat", "", 15, achC("people")},
	{"pacifist", "Pacifist", "Reach level 8 without taking a single life.", "Combat", "the Pacifist", 1, func(g *Game, st *acState) int { return achB(g.P.Level >= 8 && g.Kills == 0 && st.C["kills"] == 0) }},
	{"duelist", "Duelist", "Win three formal duels.", "Combat", "", 3, func(g *Game, st *acState) int { return g.fm().DuelW }},
	{"arena_champ", "Arena Champion", "Win three arena fights.", "Combat", "Champion", 3, func(g *Game, st *acState) int { return g.ArenaWins }},
	{"legend_killer", "Legend-Killer", "Slay a monster out of the chronicles.", "Combat", "", 1, achKinds("slay")},
	{"bounty_hunter", "Bounty Hunter", "Collect a bounty.", "Combat", "", 1, achKinds("bounty")},
	{"hideout_breaker", "Hideout Breaker", "Bring down a villain in their own hideout.", "Combat", "", 1, achKinds("villain")},
	{"egg_thief", "Egg Thief", "Rob a nest.", "Combat", "", 1, achKinds("nest")},
	{"sea_dog", "Sea Dog", "Win a fight at sea.", "Combat", "", 1, achKinds("ship")},
	{"nemesis_made", "Nemesis Made", "Make an enemy who remembers you.", "Combat", "", 1, achKinds("nemesis")},
	{"old_rival", "Old Rival", "Cross swords with rivals three times.", "Combat", "", 3, achKinds("rival")},
	{"murderer", "Blood on Your Hands", "Murder someone who did not deserve it.", "Combat", "", 1, achKinds("murder")},
	{"hard_target", "Hard Target", "Survive with a single hit point left.", "Combat", "", 1, achC("close_call")},

	// ---- travel and discovery
	{"wayfarer", "Wayfarer", "Discover 5 named places.", "Travel", "", 5, func(g *Game, st *acState) int { return len(g.Visited) }},
	{"far_wanderer", "Far Wanderer", "Discover 25 named places.", "Travel", "the Far Wanderer", 25, func(g *Game, st *acState) int { return len(g.Visited) }},
	{"world_walker", "World-Walker", "Discover 60 named places.", "Travel", "World-Walker", 60, func(g *Game, st *acState) int { return len(g.Visited) }},
	{"delver", "Delver", "Descend to the fifth level of a dungeon.", "Travel", "", 5, achC("depth")},
	{"deep_delver", "Deep Delver", "Descend ten levels.", "Travel", "the Deep Delver", 10, achC("depth")},
	{"tower_climber", "Tower-Climber", "Climb to the tenth floor of a tower.", "Travel", "", 10, achC("tower")},
	{"cloud_walker", "Cloud-Walker", "Walk on an island in the sky.", "Travel", "", 1, achC("sky")},
	{"drowned", "Drowned Depths", "Walk the sea floor.", "Travel", "", 1, achC("sea")},
	{"below_below", "Below the Below", "Enter the underdark.", "Travel", "", 1, achC("underdark")},
	{"rift_walker", "Rift-Walker", "Step through a rift to another world.", "Travel", "Rift-Walker", 2, func(g *Game, st *acState) int { return len(st.Planets) }},
	{"every_planet", "Every World", "Visit every planet of the universe.", "Travel", "the Planet-Hopper", 1, func(g *Game, st *acState) int {
		n := g.UniversePlanets()
		return achB(n >= 2 && len(st.Planets) >= n)
	}},
	{"tomb_robber", "Tomb-Robber", "Take a relic from a crypt.", "Travel", "Tomb-Robber", 1, achC("tomb")},
	{"grave_visitor", "Grave Visitor", "Enter three crypts.", "Travel", "", 3, achC("crypt")},
	{"wonder_seeker", "Wonder-Seeker", "Visit three wonders of the world.", "Travel", "", 3, achC("wonder")},
	{"ruin_delver", "Ruin-Delver", "Explore five ruins.", "Travel", "", 5, achC("ruin")},
	{"cartographer", "Cartographer", "Learn 200 regions of the map.", "Travel", "", 200, achC("known")},
	{"long_road", "The Long Road", "Travel 2,000 km overland.", "Travel", "", 2000, func(g *Game, st *acState) int { return int(g.fm().Km) }},
	{"endless_road", "Endless Road", "Travel 10,000 km overland.", "Travel", "the Endless Walker", 10000, func(g *Game, st *acState) int { return int(g.fm().Km) }},
	{"discoverer", "Discoverer", "Make a discovery worth writing down.", "Travel", "", 1, achKinds("discovery")},
	{"aeronaut", "Aeronaut", "Cross the world by air.", "Travel", "", 1, achKinds("journey")},
	{"mapmaker", "Mapmaker", "Mark three places of your own.", "Travel", "", 3, func(g *Game, st *acState) int { return len(g.wpl().List) }},

	// ---- craft and learning
	{"herb_wise", "Herb-Wise", "Reach the second rank of herbalism.", "Craft", "", 2, func(g *Game, st *acState) int { return g.al().Rank }},
	{"master_herbalist", "Master Herbalist", "Reach the fifth rank of herbalism.", "Craft", "Master Herbalist", 5, func(g *Game, st *acState) int { return g.al().Rank }},
	{"brewer", "Brewer", "Brew ten potions.", "Craft", "", 10, func(g *Game, st *acState) int { return g.al().Made }},
	{"cook", "Camp Cook", "Learn five dishes.", "Craft", "", 5, func(g *Game, st *acState) int { return len(g.Cookbook) }},
	{"gourmand", "Gourmand", "Learn fifteen dishes.", "Craft", "the Gourmand", 15, func(g *Game, st *acState) int { return len(g.Cookbook) }},
	{"magpie", "Magpie", "Complete a collection.", "Craft", "", 1, func(g *Game, st *acState) int { return clDoneCount(g) }},
	{"collector", "Collector", "Complete three collections.", "Craft", "the Collector", 3, func(g *Game, st *acState) int { return clDoneCount(g) }},
	{"secret_keeper", "Keeper of Secrets", "Learn ten secrets.", "Craft", "", 10, func(g *Game, st *acState) int { return len(g.Secrets) }},
	{"bookworm", "Bookworm", "Read ten books or inscriptions.", "Craft", "", 10, func(g *Game, st *acState) int { return len(g.Read) }},
	{"tinkerer", "Tinkerer", "Make something with your own hands.", "Craft", "", 1, achKinds("craft")},
	{"arcanist", "Arcanist", "Learn a power of the arcane.", "Craft", "", 1, achKinds("arcane")},
	{"alchemist", "Alchemist", "Make a discovery at the alembic.", "Craft", "", 1, achKinds("alchemy")},
	{"lockpick", "Light Fingers", "Open a lock that was not yours.", "Craft", "", 1, achKinds("lock")},
	{"adept", "Adept", "Raise a skill to rank 3.", "Craft", "", 3, achSkillMax},
	{"polymath", "Polymath", "Spend fifteen ranks across your skills.", "Craft", "the Polymath", 15, achSkillSum},
	{"relic_finder", "Relic-Finder", "Recover a lost relic.", "Craft", "", 1, achKinds("relic")},
	{"relic_hoard", "Relic-Hoarder", "Handle five relics.", "Craft", "", 5, achKinds("relic")},

	// ---- people
	{"married", "Married", "Wed.", "People", "", 1, func(g *Game, st *acState) int { return achB(g.fm().Spouse != nil) }},
	{"parent", "Parent", "Have a child.", "People", "", 1, func(g *Game, st *acState) int { return len(g.fm().Kids) }},
	{"big_family", "A Full House", "Raise three children.", "People", "the Patriarch", 3, func(g *Game, st *acState) int { return len(g.fm().Kids) }},
	{"never_alone", "Never Alone", "Win a companion.", "People", "", 1, achKinds("companion")},
	{"fellowship", "Fellowship", "Gather three companions' deeds.", "People", "", 3, achKinds("companion")},
	{"errand_runner", "Errand-Runner", "Finish five quests.", "People", "", 5, achQuestsClosed},
	{"hero_for_hire", "Hero for Hire", "Finish twenty quests.", "People", "the Reliable", 20, achQuestsClosed},
	{"sworn", "Sworn", "Join an order.", "People", "", 1, achKinds("order")},
	{"path_of_fist", "Path of the Fist", "Walk a martial path.", "People", "", 1, achKinds("murim")},
	{"beloved", "Beloved", "Earn great standing with a faction.", "People", "", 500, achRepMax},
	{"notorious", "Notorious", "Be hated by a faction.", "People", "the Notorious", 500, achRepMin},
	{"detective", "Consulting Detective", "Solve a murder.", "People", "", 1, achKinds("detective")},
	{"ghost_layer", "Ghost-Layer", "Lay a ghost town to rest.", "People", "", 1, achKinds("ghost")},
	{"gambler", "Gambler's Luck", "Play a game for stakes.", "People", "", 1, achKinds("game")},
	{"tradesman", "Tradesman", "Strike a trade worth a deed.", "People", "", 1, achKinds("trade")},
	{"plague_survivor", "Survived a Plague", "Come through a sickness that killed others.", "People", "", 1, func(g *Game, st *acState) int { return st.C["plague"] }},
	{"facing_end", "Facing the End", "Stand against the world's doom.", "People", "", 1, achKinds("endgame")},

	// ---- wealth and life
	{"comfortable", "Comfortable", "Carry 1,000 in coin.", "Life", "", 1000, func(g *Game, st *acState) int { return g.P.Gold }},
	{"wealthy", "Wealthy", "Carry 10,000 in coin.", "Life", "the Wealthy", 10000, func(g *Game, st *acState) int { return g.P.Gold }},
	{"dragon_hoard", "Dragon's Hoard", "Carry 100,000 in coin.", "Life", "", 100000, func(g *Game, st *acState) int { return g.P.Gold }},
	{"level10", "Veteran", "Reach level 10.", "Life", "", 10, func(g *Game, st *acState) int { return g.P.Level }},
	{"level20", "Champion of Many Roads", "Reach level 20.", "Life", "", 20, func(g *Game, st *acState) int { return g.P.Level }},
	{"level30", "Living Legend", "Reach level 30.", "Life", "the Legend", 30, func(g *Game, st *acState) int { return g.P.Level }},
	{"full_year", "A Full Year", "Survive a year (112 days).", "Life", "", 112, func(g *Game, st *acState) int { return g.Turn / DayLen }},
	{"five_winters", "Five Winters", "Survive five years.", "Life", "the Weathered", 560, func(g *Game, st *acState) int { return g.Turn / DayLen }},
	{"home_sweet", "Home Sweet Home", "Sleep in your own bed.", "Life", "", 1, achC("bed")},
	{"hard_ground", "Hard Ground", "Go a month without ever sleeping in a bed.", "Life", "the Sleepless", 1, func(g *Game, st *acState) int { return achB(g.Turn/DayLen >= 28 && st.C["bed"] == 0) }},
}

// achDefByID is built on first use.
var achIdx map[string]*achDef

func achByID(id string) *achDef {
	if achIdx == nil {
		achIdx = map[string]*achDef{}
		for i := range achDefs {
			achIdx[achDefs[i].ID] = &achDefs[i]
		}
	}
	return achIdx[id]
}
