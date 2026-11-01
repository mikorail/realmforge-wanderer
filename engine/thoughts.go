package engine

import (
	"fmt"
	"strings"
)

// The Thought Cabinet (prefix th), Disco-Elysium style. Thoughts are discovered through play,
// held in a limited number of slots, researched for in-game minutes (with a temporary penalty)
// and then internalised for a permanent effect, talk options and inner-voice lines. Forgetting
// frees the slot and removes the effect. Key J opens the cabinet screen. Traits and secondary
// stats live in traits.go. State: g.ext("th").
//
// Attribute effects are applied to g.P.Attr as tracked deltas (AppAt) so they revert exactly.

func init() {
	killHooks = append(killHooks, thKill)
	enterHooks = append(enterHooks, thEnter)
	turnHooks = append(turnHooks, thTurn)
	talkHooks = append(talkHooks, thTalk)
	sideHooks = append(sideHooks, thSide)
	mainKeys["J"] = func(g *Game) { g.Mode, g.Sel, g.Tab = "cabinet", 0, 0 }
	modeKeys["cabinet"] = thKeys
	modeDraws["cabinet"] = func(g *Game) { g.thDraw() }
}

type thSlot struct {
	ID   string `json:"id"`
	Left int    `json:"l"` // minutes of research remaining; 0 = internalised
}

type thState struct {
	C         map[string]int  `json:"c,omitempty"` // behaviour counters
	Found     []string        `json:"f,omitempty"`
	Slots     []thSlot        `json:"s,omitempty"`
	Traits    []string        `json:"t,omitempty"`
	AppAt     [6]int          `json:"aa"`
	Luck      int             `json:"lk"`
	Comp      int             `json:"cp"`
	CompMax   int             `json:"cm"`
	Fame      int             `json:"fa"`
	Infamy    int             `json:"in"`
	FameB     int             `json:"fb"`
	InfB      int             `json:"ib"`
	Perc      int             `json:"pe"`
	DeedsSeen int             `json:"ds"`
	Init      bool            `json:"i,omitempty"`
	Spared    map[int]bool    `json:"sp,omitempty"`
	Prayed    map[string]bool `json:"pr,omitempty"`
	LastHun   int             `json:"lh"`
	LastThi   int             `json:"lt"`
	LastTon   int             `json:"ltn"`
	LastP     int             `json:"lp"`
	LastState int             `json:"lst"`
	Chk       map[string]int  `json:"-"`
}

func thSt(g *Game) *thState {
	s := g.ext("th", func() any { return &thState{} }).(*thState)
	if s.C == nil {
		s.C = map[string]int{}
	}
	if s.Spared == nil {
		s.Spared = map[int]bool{}
	}
	if s.Prayed == nil {
		s.Prayed = map[string]bool{}
	}
	if !s.Init && g.P != nil {
		s.Init = true
		s.Luck = NewRng(g.Seed+":th:luck").Int(-2, 2)
		s.Comp, s.CompMax = 70, 70
		s.Perc = 5
		s.LastHun, s.LastThi = g.Hunger, g.Thirst
		s.LastP = -1
	}
	return s
}

type thOpt struct {
	Roles string // comma list of roles, "" = anyone
	Text  string
	Reply string
	Rep   int
}

type thDef struct {
	ID, Title string
	Cost      int // research minutes
	Fx        thFx
	Flav      string // {realm} {faith} {deity} {ruler} {town} {ritual}
	Voice     string
	VRole     string
	Opt       *thOpt
}

func thByRoles(r string) string { return r }

var thDefs = []*thDef{
	{ID: "blood_on_blade", Title: "Blood on the Blade", Cost: 240, Fx: thFx{At: thA(STR, 1, EGO, -1)}, Flav: "The first one is always louder than you expect. {realm} will never know his name, and you will not forget it.", Voice: "You have done this before. Your hands remember how.", VRole: "guard"},
	{ID: "butchers_arithmetic", Title: "Butcher's Arithmetic", Cost: 480, Fx: thFx{At: thA(STR, 1), Comp: 6, Chk: map[string]int{"intimidate": 1}}, Flav: "Twenty-five. You have started counting like a ledger clerk. The number does not make you sad, and that is the sad part."},
	{ID: "hecatomb", Title: "Hecatomb", Cost: 720, Fx: thFx{At: thA(WIL, -1, STR, 1), Comp: 12, Chk: map[string]int{"intimidate": 2, "empathy": -1}}, Flav: "A hundred offerings, and no altar. {deity} has not asked, and still you pile them up.", Voice: "They can smell it on you. The ones who know what it means step back."},
	{ID: "dead_dont_rest", Title: "The Dead Do Not Rest", Cost: 360, Fx: thFx{At: thA(WIL, 1), Comp: 5}, Flav: "It stood up again after you had decided it was finished. Whatever {faith} says about the grave, it was optimistic.", Voice: "Burial is a suggestion. Be wary of quiet crypts.", VRole: "priest", Opt: &thOpt{"priest", "[Thought] \"The dead walk in {town}'s ruins. What does your doctrine say of that?\"", "They go grey. \"We say nothing aloud. But we keep the lamps lit all night now.\"", 5}},
	{ID: "hubris_giants", Title: "Hubris of Giants", Cost: 600, Fx: thFx{At: thA(EGO, 1), Comp: 8, Luck: -1}, Flav: "The great thing fell. For a moment the whole room was yours. Then it was a corpse and you were tired.", Voice: "Tall things fall the loudest, friend. Mind the roof."},
	{ID: "beyond_veil", Title: "Things Beyond the Veil", Cost: 720, Fx: thFx{At: thA(INT, 1, WIL, -1), Perc: 1, Comp: -6}, Flav: "You looked at it for too long. {deity} is a candle; that was the dark the candle was for.", Voice: "Do not look straight at it. Do not look away either.", Opt: &thOpt{"priest,elder", "[Thought] \"I have seen something that should not be, and I did not look away.\"", "They do not answer for a long time. \"Then you know what we only guess.\"", 10}},
	{ID: "murder_weight", Title: "The Weight of a Murder", Cost: 600, Fx: thFx{At: thA(WIL, 1), Comp: -8, Chk: map[string]int{"empathy": 1, "persuasion": -1}}, Flav: "They were not hunting you. They were going home. {realm} has a law for this, and you would not like to hear it recited.", Voice: "Everyone here has someone. You are good at forgetting that."},
	{ID: "absent_friend", Title: "Absent Friend", Cost: 480, Fx: thFx{At: thA(WIL, 1), Comp: -4, Chk: map[string]int{"empathy": 1}}, Flav: "You catch yourself leaving room on the road for someone who is not there. The road is very quiet.", Voice: "You still say two names at the fire out of habit.", Opt: &thOpt{"villager,elder", "[Thought] \"I travelled with someone once. They did not come back.\"", "A hand on your sleeve, brief. \"Then you carry them. That is how it is done here.\"", 5}},
	{ID: "dead_king", Title: "The Dead Ruler's Shadow", Cost: 600, Fx: thFx{At: thA(INT, 1), Chk: map[string]int{"lore": 1}}, Flav: "{ruler} is dead, and the banners still hang in {realm}. Everyone is waiting for someone to say what comes next.", Voice: "Their banner is older than the man holding it. The dead ruler still governs.", VRole: "guard", Opt: &thOpt{"elder,notable,guard", "[Thought] \"{ruler} is in everyone's mouths here. Say what you really think of them.\"", "They lower their voice. \"Better than what came after. Don't repeat that.\"", 5}},
	{ID: "old_dust", Title: "Smell of Old Dust", Cost: 300, Fx: thFx{At: thA(INT, 1), Perc: 1}, Flav: "The crypt breathes out as you enter. It has been waiting a very long time to be disappointed in someone."},
	{ID: "lair_breath", Title: "Breath of the Lair", Cost: 300, Fx: thFx{At: thA(AGI, 1), Perc: 1, Comp: -2}, Flav: "Warm air, bones, patience. Something lives here that has never needed to hurry."},
	{ID: "quiet_prayers", Title: "The Quiet Between Prayers", Cost: 360, Fx: thFx{At: thA(WIL, 1), Comp: 8}, Flav: "No one answers in the silence after the {faith} words. That is where the praying actually happens.", Voice: "Be still. Your own breathing is the most honest prayer in the room.", VRole: "priest", Opt: &thOpt{"priest", "[Thought] \"I find {deity} in the quiet between the words, not in the words.\"", "A slow nod. \"You would be welcome to sit with us a while.\"", 10}},
	{ID: "below_daylight", Title: "Deeper Than Daylight", Cost: 360, Fx: thFx{At: thA(TOU, 1), Perc: 1, Comp: -3}, Flav: "You have stopped counting stairs. The sun is a story told on the surface."},
	{ID: "drowned_hush", Title: "The Drowned Hush", Cost: 300, Fx: thFx{At: thA(TOU, 1, AGI, -1), Luck: 1}, Flav: "Under the water, every sound is a rumour. You are beginning to prefer rumours."},
	{ID: "crowd_hum", Title: "The Crowd's Hum", Cost: 240, Fx: thFx{At: thA(EGO, 1), Chk: map[string]int{"streetwise": 1}}, Flav: "{town} sings with a thousand small agreements. You are learning where the wrong notes live.", Voice: "Everyone here is lying a little. Find the one lying the most.", VRole: "villager"},
	{ID: "ruin_remembers", Title: "What the Ruins Remember", Cost: 420, Fx: thFx{At: thA(INT, 1, WIL, 1), Comp: -4}, Flav: "Walls without roofs. Stairs that go up into weather. Whoever lived here is the reason {realm} still has a map."},
	{ID: "border_lines", Title: "Borders Are Lines on Paper", Cost: 360, Fx: thFx{At: thA(EGO, 1), Chk: map[string]int{"bluff": 1}}, Flav: "You crossed out of one country and the dirt did not change. The people did. The paper says it matters."},
	{ID: "armies_weather", Title: "Armies Are Weather", Cost: 420, Fx: thFx{At: thA(TOU, 1), Comp: 4, Chk: map[string]int{"survival": 1}}, Flav: "{realm}'s war moves across the hills like rain: no one decides it, everyone gets wet.", Voice: "Armies hate being watched. Say as little as you can.", VRole: "guard"},
	{ID: "empty_stomach", Title: "The Empty Stomach", Cost: 240, Fx: thFx{At: thA(TOU, 1, INT, -1), Chk: map[string]int{"survival": 1}}, Flav: "Hunger has opinions about everything and none of them are kind. Slowly you learn to let it talk."},
	{ID: "dry_tongue", Title: "Dry Tongue", Cost: 240, Fx: thFx{At: thA(WIL, 1), Chk: map[string]int{"endurance": 1}}, Flav: "A mouth like old leather. Every well on the road is now a small cathedral."},
	{ID: "full_belly", Title: "A Full Belly Is a Philosophy", Cost: 240, Fx: thFx{At: thA(TOU, 1, AGI, -1), Comp: 4}, Flav: "Bread has taught you more patience than any book. You trust people who feed you."},
	{ID: "weight_coin", Title: "The Weight of Coin", Cost: 360, Fx: thFx{Luck: 1, At: thA(WIL, -1), Chk: map[string]int{"persuasion": 1}}, Flav: "A heavy purse changes how you walk. {realm}'s coin has a king's face on it, and you are not sure he approves.", Voice: "Their eyes keep going to your belt. So do yours.", Opt: &thOpt{"merchant,smith,apothecary,exotic", "[Thought] \"Money talks to me. Let us hear what it says to you.\"", "A thin smile. \"Then we speak the same language. I will keep my prices... considerate.\"", 5}},
	{ID: "light_purse", Title: "Light Purse, Lighter Step", Cost: 300, Fx: thFx{At: thA(AGI, 1), Luck: 1, Comp: 3}, Flav: "You have nothing a thief would want. It is a surprisingly good feeling."},
	{ID: "nearly_over", Title: "Nearly Over", Cost: 480, Fx: thFx{At: thA(TOU, 1), Luck: 1, Comp: -5}, Flav: "The world went thin and bright, like paper held to a flame. You came back with a better understanding of how thin it is."},
	{ID: "rain_thought", Title: "Rain on the Road", Cost: 240, Fx: thFx{At: thA(WIL, 1), Chk: map[string]int{"survival": 1}}, Flav: "Water finds every seam. You stop minding. The road keeps going either way."},
	{ID: "fog_thought", Title: "Fog Has No Opinions", Cost: 240, Fx: thFx{Perc: 1, At: thA(INT, 1, AGI, -1)}, Flav: "Everything is closer than it looks and farther than it sounds. You listen, which the fog prefers."},
	{ID: "snow_thought", Title: "Snow Keeps Secrets", Cost: 240, Fx: thFx{At: thA(TOU, 1), Comp: 3, Luck: -1}, Flav: "Footprints are the only honest document. You walk on the page and write yourself into it."},
	{ID: "heat_thought", Title: "Dust in the Teeth", Cost: 240, Fx: thFx{At: thA(TOU, 1, EGO, -1), Chk: map[string]int{"endurance": 1}}, Flav: "The land has been burned to a single idea. You are learning to share it."},
	{ID: "hour_wolf", Title: "Hour of the Wolf", Cost: 300, Fx: thFx{At: thA(AGI, 1), Perc: 1, Comp: -3}, Flav: "Past midnight the {town} mind lets go of its manners. Things are said to the dark that daylight would not hear."},
	{ID: "margin_notes", Title: "The Margin Notes", Cost: 360, Fx: thFx{At: thA(INT, 1), Chk: map[string]int{"lore": 1}}, Flav: "Someone argued with this book before you did, in a tiny hand. They were wrong, but kindly."},
	{ID: "library_hunger", Title: "Library Hunger", Cost: 600, Fx: thFx{At: thA(INT, 2, STR, -1), Chk: map[string]int{"lore": 1}}, Flav: "You have started to see the world as a catalogue with missing pages."},
	{ID: "doctrine_doubt", Title: "Doctrine and Doubt", Cost: 420, Fx: thFx{At: thA(WIL, 1), Chk: map[string]int{"lore": 1, "empathy": 1}}, Flav: "The priest believed it, and also did not. {faith} is held up by that gap between the two.", Voice: "They say the words. They watch your face while they say them.", VRole: "priest", Opt: &thOpt{"priest", "[Thought] \"Faith is the part that remains when you stop asking. How long since you stopped?\"", "A brittle laugh. \"Some mornings I wish you had not asked.\"", 8}},
	{ID: "own_faith", Title: "The Taste of {faith}", Cost: 420, Fx: thFx{At: thA(WIL, 1), Comp: 6}, Flav: "You were raised on {deity}'s name like bread. Away from home you notice how foreign other people's bread tastes.", Voice: "They wear the old symbols. Their prayers are not yours.", VRole: "priest"},
	{ID: "caste_ladder", Title: "The Ladder Has No Top", Cost: 360, Fx: thFx{At: thA(EGO, 1), Chk: map[string]int{"empathy": 1, "bluff": 1}}, Flav: "High or low, they all look up to someone. You notice the one who looks nowhere."},
	{ID: "merchant_smile", Title: "The Merchant's Smile", Cost: 300, Fx: thFx{At: thA(EGO, 1, WIL, -1), Chk: map[string]int{"persuasion": 1}}, Flav: "It is exactly as wide as the profit. You are learning to wear one.", Voice: "The smile is a price. Find out what it costs.", VRole: "merchant"},
	{ID: "guard_gaze", Title: "The Warden's Gaze", Cost: 300, Fx: thFx{At: thA(WIL, 1), Chk: map[string]int{"intimidate": 1}}, Flav: "Years of looking at crowds has turned their eyes into fences. You learn to be a gap.", Voice: "He wants you to move along. He also wants to be asked why not.", VRole: "guard"},
	{ID: "elder_memory", Title: "An Elder's Memory", Cost: 420, Fx: thFx{At: thA(INT, 1), Chk: map[string]int{"lore": 1, "persuasion": 1}}, Flav: "Old people remember the world as it was lived, not as it was written. {realm}'s chroniclers should be a little afraid of them.", Voice: "They know more than they say. Be quiet and be patient.", VRole: "elder"},
	{ID: "water_remembers", Title: "Water Remembers", Cost: 480, Fx: thFx{At: thA(EGO, 1), Luck: 1}, Flav: "The {ritual} binds people strangely. You can feel the thread of it long after the water has dried."},
	{ID: "long_road", Title: "The Long Road", Cost: 480, Fx: thFx{At: thA(TOU, 1), Perc: 1, Chk: map[string]int{"survival": 1}}, Flav: "Your boots have a different shape from your feet now. You can no longer say where you started."},
	{ID: "road_home", Title: "Which Way Is Home", Cost: 480, Fx: thFx{At: thA(WIL, 1), Comp: -3, Luck: 1}, Flav: "You have gone far from {realm}. Home is sharper in memory than it ever was in front of you."},
	{ID: "weight_experience", Title: "The Weight of Experience", Cost: 600, Fx: thFx{At: thA(INT, 1, WIL, 1), Comp: 6}, Flav: "You are no longer the person who left. Whoever that was, you owe them a small apology."},
	{ID: "stories_told", Title: "Someone Is Telling Stories", Cost: 480, Fx: thFx{At: thA(EGO, 1), Comp: 5, Chk: map[string]int{"persuasion": 1}}, Flav: "In {town}, a stranger repeated your name as if it were weather. It sounded better than you remembered it.", Voice: "They know the story. They want to see if you match it.", Opt: &thOpt{"villager,merchant,guard", "[Thought] \"I hear a few of you have been telling stories about me.\"", "They go pink, then grin. \"Only the good parts. Most of them.\"", 8}},
	{ID: "name_whispered", Title: "A Name Said Quietly", Cost: 480, Fx: thFx{At: thA(EGO, 1, WIL, -1), Chk: map[string]int{"intimidate": 2}}, Flav: "Doors close a little softer when you arrive. {realm} has begun to measure you against its nightmares.", Voice: "They are afraid of you. Do not enjoy it too much.", Opt: &thOpt{"villager,merchant,guard", "[Thought] \"You know who I am. Say it.\"", "A tight swallow. \"I would rather not. Please.\"", -5}},
	{ID: "shaking_hands", Title: "Hands That Shake", Cost: 420, Fx: thFx{At: thA(WIL, 1), Comp: -5, Perc: 1}, Flav: "You put your sword down to prove you could. It took you longer to pick it up than it should."},
	{ID: "taste_smoke", Title: "The Taste of Smoke", Cost: 300, Fx: thFx{At: thA(TOU, 1), Comp: -2, Chk: map[string]int{"endurance": 1}}, Flav: "Fire is honest about what it wants. You've started thinking in its sentences."},
	{ID: "red_thread", Title: "The Red Thread", Cost: 300, Fx: thFx{At: thA(TOU, 1, AGI, -1), Luck: 1}, Flav: "A thin line of warmth running out of you. You are a closed book, and someone has been reading."},
	{ID: "two_footprints", Title: "Two Sets of Footprints", Cost: 360, Fx: thFx{At: thA(EGO, 1), Comp: 6}, Flav: "You haven't spoken aloud to yourself in weeks. Someone answers now. It is a better habit than the old one."},
}

var thByID = func() map[string]*thDef {
	m := map[string]*thDef{}
	for _, d := range thDefs {
		m[d.ID] = d
	}
	return m
}()

// thFill expands {placeholders} against the live world.
func thFill(g *Game, s string) string {
	if !strings.Contains(s, "{") {
		return s
	}
	realm := "the Free Folk"
	if g.HomeState > 0 {
		realm = g.realmName(g.HomeState)
	}
	faith, deity := "the old faith", "the unnamed god"
	if r := g.W.relByID[g.HomeFaith]; r != nil {
		faith = r.Name
		deity = nonEmpty(r.Deity, deity)
	}
	ruler := "the last ruler"
	for _, p := range g.W.People {
		if !p.Alive && p.Died > 0 {
			ruler = p.Name
			break
		}
	}
	town := "this place"
	if g.Cur != nil && g.Cur.Name != "" {
		town = g.Cur.Name
	}
	return strings.NewReplacer("{realm}", realm, "{faith}", faith, "{deity}", deity, "{ruler}", ruler, "{town}", town, "{ritual}", strings.ToLower(g.Sk.Ritual)).Replace(s)
}

func (d *thDef) title(g *Game) string { return thFill(g, d.Title) }

func thFound(g *Game, id string) bool {
	for _, f := range thSt(g).Found {
		if f == id {
			return true
		}
	}
	return false
}

func thSlotOf(g *Game, id string) *thSlot {
	s := thSt(g)
	for i := range s.Slots {
		if s.Slots[i].ID == id {
			return &s.Slots[i]
		}
	}
	return nil
}

// thDone reports whether a thought is fully internalised.
func thDone(g *Game, id string) bool {
	sl := thSlotOf(g, id)
	return sl != nil && sl.Left <= 0
}

// thFind discovers a thought (idempotent) and announces it.
func thFind(g *Game, id string) bool {
	d := thByID[id]
	if d == nil || thFound(g, id) {
		return false
	}
	s := thSt(g)
	s.Found = append(s.Found, id)
	g.Msg(fmt.Sprintf("{{M|A thought occurs to you: %s.}} {{K|(J to open the cabinet)}}", d.title(g)))
	return true
}

func thSlotCount(g *Game) int {
	return clampi(2+(g.P.Level-1)/4+maxi(0, g.P.M(INT)/2), 2, 8)
}

// thStart begins researching a found thought. Returns a message.
func thStart(g *Game, id string) string {
	s := thSt(g)
	d := thByID[id]
	switch {
	case d == nil || !thFound(g, id):
		return "You have not had that thought yet."
	case thSlotOf(g, id) != nil:
		return "Already in the cabinet."
	case len(s.Slots) >= thSlotCount(g):
		return "The cabinet is full. Forget a thought first."
	}
	cost := d.Cost * (100 - 5*clampi(g.P.M(INT), 0, 6)) / 100
	s.Slots = append(s.Slots, thSlot{ID: id, Left: maxi(30, cost)})
	thSync(g)
	return fmt.Sprintf("You turn %s over in your mind. It will take about %s.", d.title(g), thHours(maxi(30, cost)))
}

// thForget drops a thought from the cabinet, freeing the slot and removing its effect.
func thForget(g *Game, id string) bool {
	s := thSt(g)
	for i := range s.Slots {
		if s.Slots[i].ID == id {
			s.Slots = append(s.Slots[:i], s.Slots[i+1:]...)
			thSync(g)
			return true
		}
	}
	return false
}

func thHours(m int) string {
	if m >= 120 {
		return fmt.Sprintf("%d hours", (m+30)/60)
	}
	return fmt.Sprintf("%d minutes", m)
}

// thSync recomputes every derived effect and moves g.P.Attr by the difference from last time.
func thSync(g *Game) {
	if g.P == nil {
		return
	}
	s := thSt(g)
	var want [6]int
	chk := map[string]int{}
	luck, comp, perc := 0, 0, 0
	add := func(f thFx) {
		for i, v := range f.At {
			want[i] += v
		}
		luck += f.Luck
		comp += f.Comp
		perc += f.Perc
		for k, v := range f.Chk {
			chk[k] += v
		}
	}
	for _, id := range s.Traits {
		if t := thTraitByID[id]; t != nil {
			add(t.Fx)
		}
	}
	for _, sl := range s.Slots {
		d := thByID[sl.ID]
		if d == nil {
			continue
		}
		if sl.Left > 0 {
			want[INT]--
			want[WIL]--
		} else {
			add(d.Fx)
		}
	}
	if s.Comp < 25 {
		want[WIL]--
	}
	if s.Comp < 10 {
		want[WIL]--
		want[AGI]--
	}
	for i := range want {
		g.P.Attr[i] += want[i] - s.AppAt[i]
		s.AppAt[i] = want[i]
	}
	s.Chk = chk
	base := NewRng(g.Seed+":th:luck").Int(-2, 2)
	s.Luck = clampi(base+luck, -5, 10)
	s.CompMax = clampi(60+5*g.P.M(WIL)+comp, 20, 100)
	s.Comp = clampi(s.Comp, 0, s.CompMax)
	s.Perc = clampi(5+g.P.M(INT)+g.P.M(WIL)/2+perc, 1, 20)
	thFameFromDeeds(g)
}

// ---- hooks ----

func thKill(g *Game, d, src *Actor) {
	if g.P == nil || g.W == nil {
		return
	}
	s := thSt(g)
	if d.AI == "follower" {
		s.C["compLost"]++
		thFind(g, "absent_friend")
		return
	}
	if src != g.P {
		return
	}
	s.C["kills"]++
	k := s.C["kills"]
	thFind(g, "blood_on_blade")
	if k >= 25 {
		thFind(g, "butchers_arithmetic")
	}
	if k >= 100 {
		thFind(g, "hecatomb")
	}
	if d.Has("undead") || d.Faction == "undead" {
		thFind(g, "dead_dont_rest")
	}
	if d.Boss || d.Title != "" && d.SimMon == 0 && d.Level >= g.P.Level {
		s.C["bossKills"]++
		thFind(g, "hubris_giants")
	}
	if d.Faction == "eldritch" {
		thFind(g, "beyond_veil")
	}
	if d.Role == "priest" {
		s.C["priestKills"]++
	}
	if (d.AI == "town" || d.SimPer > 0) && !d.Angry {
		s.C["innocents"]++
		thFind(g, "murder_weight")
	}
	thEarn(g)
	thSync(g)
}

func thEnter(g *Game, zn *Zone) {
	if g.P == nil || g.W == nil || zn == nil {
		return
	}
	s := thSt(g)
	if zn.WX != s.LastP && s.LastP >= 0 {
		s.C["steps"]++
	}
	s.LastP = zn.WX
	if st := g.W.State[g.W.Idx(zn.WX, zn.WY)]; st != s.LastState {
		if s.LastState != 0 && st != 0 {
			thFind(g, "border_lines")
		}
		s.LastState = st
	}
	switch zn.Kind {
	case "crypt":
		thFind(g, "old_dust")
	case "lair":
		thFind(g, "lair_breath")
	case "sea":
		s.C["sea"]++
	case "holy", "monastic":
		thFind(g, "quiet_prayers")
	case "ruin":
		thFind(g, "ruin_remembers")
	case "town":
		thFind(g, "crowd_hum")
		if b := g.W.burgByID[zn.Burg]; b != nil && !b.Alive {
			thFind(g, "ruin_remembers")
		}
		for _, p := range g.W.People {
			if !p.Alive && p.Died > 0 {
				thFind(g, "dead_king")
				break
			}
		}
	}
	if zn.Underwater {
		thFind(g, "drowned_hush")
		s.C["sea"]++
	}
	if zn.Z >= 3 || zn.Z <= -100 {
		s.C["deep"]++
		thFind(g, "below_daylight")
	}
	if _, _, ok := g.frontierAt(zn.WX, zn.WY); ok {
		thFind(g, "armies_weather")
	}
	if s.C["steps"] >= 100 {
		thFind(g, "long_road")
	}
	if s.C["steps"] >= 30 && g.HomeState > 0 && abs(zn.WX-g.StartX)+abs(zn.WY-g.StartY) >= 20 {
		thFind(g, "road_home")
	}
	thEarn(g)
}

func thTurn(g *Game) {
	if g.P == nil || g.W == nil || g.Cur == nil || g.Dead {
		return
	}
	s := thSt(g)
	// research
	changed := false
	for i := range s.Slots {
		sl := &s.Slots[i]
		if sl.Left > 0 {
			sl.Left--
			if sl.Left == 0 {
				changed = true
				d := thByID[sl.ID]
				g.Msg(fmt.Sprintf("{{G|You have internalised %s.}} {{K|(%s)}}", d.title(g), d.Fx.text()))
			}
		}
	}
	// counters: appetite, thirst, tonics
	if g.Hunger < s.LastHun-300 {
		s.C["meals"]++
	}
	if g.Thirst < s.LastThi-200 {
		s.C["drinks"]++
	}
	s.LastHun, s.LastThi = g.Hunger, g.Thirst
	tonics := 0
	for _, it := range g.P.Inv {
		if it.Kind == "tonic" {
			tonics += maxi(1, it.Count)
		}
	}
	if tonics < s.LastTon {
		s.C["tonics"] += s.LastTon - tonics
	}
	s.LastTon = tonics
	if g.P.Gold >= 300 {
		s.C["hoardMin"]++
	}
	if g.P.Level >= 6 && g.P.Gold < 20 {
		s.C["poorMin"]++
	}
	outdoor := g.Cur.Z == 0 && !g.Cur.Dark && !g.Cur.Underwater
	if !g.daylight() && outdoor {
		s.C["nightMin"]++
	}
	hasFollower := false
	for _, a := range g.Cur.Actors {
		if a.AI == "follower" && a.HP > 0 {
			hasFollower = true
		}
		if a.Has("chk:calm") && !a.Angry && !s.Spared[a.ID] {
			s.Spared[a.ID] = true
			s.C["spared"]++
		}
	}
	if hasFollower {
		s.C["followerMin"]++
		if s.C["followerMin"] >= 600 {
			thFind(g, "two_footprints")
		}
	}
	lowNow := g.P.HP*4 < g.P.MaxHP
	if lowNow && !g.P.Has("th:low") {
		g.P.Set("th:low")
		s.C["lowHP"]++
		thFind(g, "nearly_over")
	} else if !lowNow && g.P.HP*2 > g.P.MaxHP && g.P.Has("th:low") {
		delete(g.P.Flags, "th:low")
	}
	thComposureTick(g, s)
	if g.Turn%5 == 0 {
		thWatch(g, s, outdoor)
	}
	if changed || g.Turn%30 == 0 {
		thSync(g)
	}
}

func thComposureTick(g *Game, s *thState) {
	drain := 0
	for _, a := range g.Cur.Actors {
		if a.HP <= 0 || a == g.P || a.AI == "follower" || abs(a.X-g.P.X) > 9 || abs(a.Y-g.P.Y) > 9 || !g.visible(a.X, a.Y) {
			continue
		}
		switch {
		case a.Faction == "eldritch":
			drain += 2
		case a.Boss || a.Title != "" && a.AI == "hostile" && a.Level > g.P.Level:
			drain++
		case a.Has("undead") && g.Turn%3 == 0:
			drain++
		}
	}
	if g.P.HP*4 < g.P.MaxHP && g.Turn%2 == 0 {
		drain++
	}
	switch {
	case drain > 0:
		before := s.Comp
		s.Comp = maxi(0, s.Comp-drain)
		if before >= 25 && s.Comp < 25 {
			g.Msg("{{R|Your hands start to shake. You are losing your composure.}}")
			thFind(g, "shaking_hands")
			thSync(g)
		}
	case g.Turn%10 == 0 && s.Comp < s.CompMax:
		s.Comp++
		if g.Cur.Kind == "town" && g.Turn%20 == 0 {
			s.Comp++
		}
		if s.Comp == 25 {
			thSync(g)
		}
	}
	if s.Comp < 25 {
		s.C["lowCompMin"]++
	}
}

func thWatch(g *Game, s *thState, outdoor bool) {
	if g.Hunger >= 1500 {
		thFind(g, "empty_stomach")
	}
	if g.Thirst >= 1200 {
		thFind(g, "dry_tongue")
	}
	if s.C["meals"] >= 10 {
		thFind(g, "full_belly")
	}
	if g.P.Gold >= 400 {
		thFind(g, "weight_coin")
	}
	if g.P.Gold < 5 && g.P.Level >= 4 {
		thFind(g, "light_purse")
	}
	if outdoor {
		if !g.daylight() {
			thFind(g, "hour_wolf")
		}
		if w := g.activeWeather(); w != nil && w.Kind != nil {
			switch w.Kind.ID {
			case "rain", "storm", "tempest", "acidrain", "sporerain":
				thFind(g, "rain_thought")
				if w.Kind.ID == "storm" || w.Kind.ID == "tempest" {
					s.C["stormMin"] += 5
				}
			case "fog", "steamfog", "smog":
				thFind(g, "fog_thought")
			case "snow", "blizzard", "frost":
				thFind(g, "snow_thought")
				if w.Kind.ID == "blizzard" {
					s.C["stormMin"] += 5
				}
			case "sandstorm", "heat", "ashfall", "fallout":
				thFind(g, "heat_thought")
			}
		}
	}
	books := 0
	for k, v := range g.Read {
		if v && strings.HasPrefix(k, "book:") {
			books++
		}
	}
	s.C["books"] = books
	if books >= 3 {
		thFind(g, "margin_notes")
	}
	if books >= 10 {
		thFind(g, "library_hunger")
	}
	if g.P.St("burning") > 0 {
		thFind(g, "taste_smoke")
	}
	if g.P.St("bleeding") > 0 {
		thFind(g, "red_thread")
	}
	if g.P.Level >= 8 {
		thFind(g, "weight_experience")
	}
	if g.HomeFaith > 0 && g.P.Level >= 3 {
		thFind(g, "own_faith")
	}
	if rit := lgSt(g).Rit; len(rit) > 0 {
		thFind(g, "water_remembers")
	}
	if s.Fame >= 40 {
		thFind(g, "stories_told")
	}
	if s.Infamy >= 40 {
		thFind(g, "name_whispered")
	}
	// prayer at altars, once per altar-zone per day
	for _, d := range dirs8 {
		if g.Cur.At(g.P.X+d[0], g.P.Y+d[1]) == TAltar {
			k := fmt.Sprintf("%s:%d", g.Cur.Key, g.Day())
			if !s.Prayed[k] {
				s.Prayed[k] = true
				s.C["prayers"]++
				thFind(g, "quiet_prayers")
			}
			break
		}
	}
	thEarn(g)
}

func thTalk(g *Game, a *Actor) []Option {
	if a == nil || g.P == nil {
		return nil
	}
	s := thSt(g)
	if !a.Has("th:talked") {
		a.Set("th:talked")
		s.C["talks"]++
		switch a.Role {
		case "priest":
			thFind(g, "doctrine_doubt")
		case "merchant", "smith", "apothecary", "bookseller", "exotic":
			thFind(g, "merchant_smile")
		case "guard":
			thFind(g, "guard_gaze")
		case "elder", "notable":
			thFind(g, "elder_memory")
		}
		if a.Caste != "" {
			thFind(g, "caste_ladder")
		}
		thEarn(g)
	}
	// inner voices from internalised thoughts
	if vs := thVoices(g, a); len(vs) > 0 {
		g.Popup = append(g.Popup, "")
		for _, v := range vs {
			g.Popup = append(g.Popup, Wrap(v, 100)...)
		}
	}
	var opts []Option
	if !a.Has("th:rep") {
		if line := thTraitLine(g, a); line != "" {
			rep := thTraitRep(g, a)
			opts = append(opts, Option{"{{M|[Your reputation]}} (let them look you over)", func() {
				a.Set("th:rep")
				if rep != 0 {
					g.AddRep(a.Faction, rep*5, " for the kind of person you are")
				}
				g.talkMenu(line)
			}})
		}
	}
	for _, d := range thDefs {
		if d.Opt == nil || !thDone(g, d.ID) || a.Has("th:o:"+d.ID) {
			continue
		}
		if d.Opt.Roles != "" && !strings.Contains(","+d.Opt.Roles+",", ","+a.Role+",") {
			continue
		}
		d := d
		opts = append(opts, Option{"{{M|" + thFill(g, d.Opt.Text) + "}}", func() {
			a.Set("th:o:" + d.ID)
			if d.Opt.Rep != 0 {
				g.AddRep(a.Faction, d.Opt.Rep, " for a thoughtful word")
			}
			g.gainXP(5)
			g.talkMenu(thFill(g, d.Opt.Reply))
		}})
	}
	return opts
}

// thVoices are the inner-voice lines of internalised thoughts that fit this person.
func thVoices(g *Game, a *Actor) []string {
	if a == nil {
		return nil
	}
	var pool []string
	for _, d := range thDefs {
		if d.Voice == "" || !thDone(g, d.ID) {
			continue
		}
		if d.VRole != "" && d.VRole != a.Role {
			continue
		}
		pool = append(pool, fmt.Sprintf("{{m|%s}} — %s", strings.ToUpper(d.title(g)), d.Voice))
	}
	r := NewRng(fmt.Sprintf("thv:%d:%d", a.ID, g.Turn/600))
	var out []string
	for len(pool) > 0 && len(out) < 2 {
		i := r.Int(0, len(pool)-1)
		out = append(out, pool[i])
		pool = append(pool[:i], pool[i+1:]...)
	}
	return out
}

func thSide(g *Game) []string {
	if g.P == nil {
		return nil
	}
	s := thSt(g)
	out := []string{fmt.Sprintf("{{K|Luck %+d  Comp %d  Fame %d}}", s.Luck, s.Comp, s.Fame)}
	for _, sl := range s.Slots {
		if sl.Left > 0 {
			out = append(out, fmt.Sprintf("{{m|Thinking: %s}} {{K|%s}}", Plain(thByID[sl.ID].title(g)), thHours(sl.Left)))
			break
		}
	}
	return out
}

// ---- screen ----

func thRows(g *Game) []string {
	s := thSt(g)
	var rows []string
	for _, sl := range s.Slots {
		rows = append(rows, sl.ID)
	}
	for _, f := range s.Found {
		if thSlotOf(g, f) == nil {
			rows = append(rows, f)
		}
	}
	return rows
}

func thKeys(g *Game, k string) {
	rows := thRows(g)
	g.Sel = clampi(g.Sel, 0, maxi(0, len(rows)-1))
	switch k {
	case "Escape", "q", "J":
		g.Mode = "main"
	case "Tab", "ArrowRight", "ArrowLeft":
		g.Tab = (g.Tab + 1) % 2
		g.Sel = 0
	case "ArrowUp", "N8", "k":
		g.Sel = maxi(0, g.Sel-1)
	case "ArrowDown", "N2", "j":
		g.Sel = mini(maxi(0, len(rows)-1), g.Sel+1)
	case "Enter", " ":
		if g.Tab == 0 && len(rows) > 0 {
			g.Msg(thStart(g, rows[g.Sel]))
		}
	case "f", "Delete", "Backspace":
		if g.Tab == 0 && len(rows) > 0 && thForget(g, rows[g.Sel]) {
			g.Msg("You let the thought go. The slot is free.")
		}
	}
}

func (g *Game) thDraw() {
	s := g.sc
	x, y, w, h := g.panel("Thought Cabinet")
	st := thSt(g)
	thSync(g)
	tabs := []string{"Thoughts", "Traits"}
	tx := x
	for i, t := range tabs {
		c := "K"
		if i == g.Tab {
			c = "W"
		}
		tx = s.TextB(tx, y, fmt.Sprintf("{{%s|[%s]}} ", c, t), CGray, CBgDark, x+w)
	}
	s.TextB(tx+2, y, fmt.Sprintf("{{y|Slots %d/%d}}  {{K|Enter research · f forget · Tab · Esc}}", len(st.Slots), thSlotCount(g)), CGray, CBgDark, x+w)
	lw := 42
	rows := thRows(g)
	g.Sel = clampi(g.Sel, 0, maxi(0, len(rows)-1))
	listH := h - 11
	if g.Tab == 0 {
		top := maxi(0, g.Sel-listH+1)
		for i := top; i < len(rows) && i-top < listH; i++ {
			bg := CBgDark
			if i == g.Sel {
				bg = CBgSel
			}
			ry := y + 2 + i - top
			s.Fill(x, ry, lw, 1, bg)
			d := thByID[rows[i]]
			tag := "{{K|found}}"
			if sl := thSlotOf(g, d.ID); sl != nil {
				if sl.Left > 0 {
					tag = "{{y|thinking " + thHours(sl.Left) + "}}"
				} else {
					tag = "{{G|internalised}}"
				}
			}
			s.TextB(x, ry, fmt.Sprintf("%s %s", tag, Plain(d.title(g))), CGray, bg, x+lw)
		}
		if len(rows) == 0 {
			s.TextB(x, y+2, "{{K|No thoughts yet. Kill, travel, read, pray, talk. They will come.}}", CGray, CBgDark, x+w)
		} else {
			d := thByID[rows[g.Sel]]
			dy := y + 2
			for _, l := range Wrap("{{M|"+d.title(g)+"}}\n"+thFill(g, d.Flav), w-lw-3) {
				s.TextB(x+lw+2, dy, l, CGray, CBgDark, x+w)
				dy++
			}
			dy++
			s.TextB(x+lw+2, dy, fmt.Sprintf("{{G|Effect:}} %s", d.Fx.text()), CGray, CBgDark, x+w)
			s.TextB(x+lw+2, dy+1, fmt.Sprintf("{{R|While researching:}} INT -1, WIL -1 · %s", thHours(d.Cost)), CGray, CBgDark, x+w)
		}
	} else {
		for i, id := range st.Traits {
			if i >= listH {
				break
			}
			t := thTraitByID[id]
			s.TextB(x, y+2+i, fmt.Sprintf("{{G|%s}} {{K|%s}}", t.Name, t.Fx.text()), CGray, CBgDark, x+w)
		}
		if len(st.Traits) == 0 {
			s.TextB(x, y+2, "{{K|No traits earned yet. They come from how you behave.}}", CGray, CBgDark, x+w)
		}
	}
	// stats section
	sy := y + h - 7
	s.TextB(x, sy, "{{c|Secondary stats}}", CGray, CBgDark, x+w)
	s.TextB(x, sy+1, fmt.Sprintf("Luck {{W|%+d}}    Composure {{W|%d/%d}}    Perception {{W|%d}}", st.Luck, st.Comp, st.CompMax, st.Perc), CGray, CBgDark, x+w)
	s.TextB(x, sy+2, fmt.Sprintf("Fame {{W|%d}}    Infamy {{W|%d}}    {{K|%s}}", st.Fame, st.Infamy, thFameTitle(g)), CGray, CBgDark, x+w)
	if st.Comp < 25 {
		s.TextB(x, sy+3, "{{R|Shaken: -1 WIL, -1 to checks.}}", CGray, CBgDark, x+w)
	}
}
