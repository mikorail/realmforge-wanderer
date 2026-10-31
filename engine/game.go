package engine

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

type Msg struct {
	T    string `json:"t"`
	Turn int    `json:"n"`
	Rep  int    `json:"r,omitempty"`
}

type Secret struct {
	Text string `json:"t"`
	P    int    `json:"p"`
	Kind string `json:"k"`
	Year int    `json:"y"`
}

type Deed struct {
	Kind     string `json:"kind"` // slay relic quest death kinship
	Text     string `json:"text"`
	Monster  int    `json:"monster,omitempty"`
	Artifact int    `json:"artifact,omitempty"`
	Cell     int    `json:"cell"`
	Year     int    `json:"year"`
}

type Game struct {
	W  *WorldData `json:"-"`
	Sk *Skin      `json:"-"`
	sc *Screen

	Genre     string           `json:"genre"`
	Seed      string           `json:"seed"`
	P         *Actor           `json:"p"`
	Calling   string           `json:"calling"`
	Zones     map[string]*Zone `json:"zones"`
	CurKey    string           `json:"cur"`
	Cur       *Zone            `json:"-"`
	OnWorld   bool             `json:"onWorld"`
	WX        int              `json:"wx"`
	WY        int              `json:"wy"`
	StartX    int              `json:"sx"`
	StartY    int              `json:"sy"`
	StartBurg int              `json:"sb"`
	HomeState int              `json:"hs"`
	HomeFaith int              `json:"hf"`
	Turn      int              `json:"turn"`
	Tick      int              `json:"tick"`
	Msgs      []Msg            `json:"msgs"`
	Rep       map[string]int   `json:"rep"`
	Skills    map[string]int   `json:"skills"`
	XP        int              `json:"xp"`
	SP        int              `json:"sp"`
	MP        int              `json:"mp"`
	AP        int              `json:"ap"`
	Hunger    int              `json:"hunger"`
	Thirst    int              `json:"thirst"`
	Breath    int              `json:"breath"`
	Known     map[int]bool     `json:"known"`
	Secrets   []Secret         `json:"secrets"`
	SecretSet map[string]bool  `json:"secretSet"`
	Quests    []*Quest         `json:"quests"`
	Deeds     []Deed           `json:"deeds"`
	Slain     map[int]bool     `json:"slain"`
	Taken     map[int]bool     `json:"taken"`
	NextID    int              `json:"nid"`
	R         *Rng             `json:"rng"`
	Dead      bool             `json:"dead"`
	DeathText string           `json:"deathText"`
	Kills     int              `json:"kills"`
	Visited   map[string]bool  `json:"visited"`
	Read      map[string]bool  `json:"read"`
	DeedsSent int              `json:"deedsSent"`

	// ui state (not saved)
	Mode    string   `json:"-"`
	Sel     int      `json:"-"`
	Sel2    int      `json:"-"`
	Tab     int      `json:"-"`
	CX, CY  int      `json:"-"`
	Pend    string   `json:"-"` // pending action for targeting: fire:, throw:<id>, power:<id>
	Talk    *Actor   `json:"-"`
	Popup   []string `json:"-"`
	PopT    string   `json:"-"`
	ItemSel *Item    `json:"-"`
	Path    [][2]int `json:"-"`
	Opts    []Option `json:"-"`
	vis     []bool
	wmInfo  []string
	sensed  []bool
}

type Option struct {
	Text string
	Do   func()
}

func (g *Game) Msg(s string) {
	if n := len(g.Msgs); n > 0 && g.Msgs[n-1].T == s && g.Msgs[n-1].Turn == g.Turn {
		g.Msgs[n-1].Rep++
		return
	}
	g.Msgs = append(g.Msgs, Msg{T: s, Turn: g.Turn})
	if len(g.Msgs) > 300 {
		g.Msgs = g.Msgs[len(g.Msgs)-300:]
	}
}

func (g *Game) the(a *Actor) string {
	if a == g.P {
		return "you"
	}
	if a.Proper {
		return a.Name
	}
	return "the " + a.Name
}

func (g *Game) The(a *Actor) string {
	s := g.the(a)
	return strings.ToUpper(s[:1]) + s[1:]
}

// NewGame builds the player from the chargen spec and drops them at their home settlement.
func NewGame(w *WorldData, spec CharSpec) *Game {
	g := &Game{W: w, Sk: SkinFor(w.Genre), Genre: w.Genre, Seed: w.Seed, Zones: map[string]*Zone{}, Known: map[int]bool{}, SecretSet: map[string]bool{},
		Slain: map[int]bool{}, Taken: map[int]bool{}, Visited: map[string]bool{}, Read: map[string]bool{}, R: NewRng(w.Seed + ":play:" + spec.Name)}
	g.sc = NewScreen()
	sp := w.specByID[spec.Species]
	if sp == nil && len(w.Species) > 0 {
		sp = &w.Species[0]
	}
	plan := "humanoid"
	spID := "human"
	if sp != nil {
		plan, spID = BodyPlanForSpecies(sp.ID), sp.ID
	}
	p := &Actor{ID: 1, Name: validName(spec.Name), Proper: true, Glyph: '@', F: CWhite, Tpl: "player", Faction: "player", Level: 1, Speed: 100,
		Body: MakeBody(plan), Species: spID, AI: "player", Powers: map[string]int{}, Cool: map[string]int{}}
	g.NextID = 10
	g.Calling = spec.Calling
	var cd *CallingDef
	for i := range callingDefs {
		if callingDefs[i].ID == spec.Calling {
			cd = &callingDefs[i]
		}
	}
	if cd == nil {
		cd = &callingDefs[0]
		g.Calling = cd.ID
	}
	// attributes: chargen values are already validated host-side but clamp anyway
	for i := 0; i < 6; i++ {
		p.Attr[i] = clampi(spec.Attr[i], 8, 26)
	}
	if sp != nil {
		for k, v := range speciesAttr(sp) {
			p.Attr[k] += v
		}
	}
	for k, v := range cd.Attr {
		p.Attr[k] += v
	}
	g.Skills = map[string]int{}
	for k, v := range cd.Skills {
		g.Skills[k] = v
	}
	spent := 0
	for id, lv := range spec.Powers {
		pd := powerByID[id]
		if pd == nil || lv <= 0 {
			continue
		}
		if pd.Defect {
			p.Powers[id] = 1
			spent += pd.Cost
			continue
		}
		lv = clampi(lv, 1, 3)
		p.Powers[id] = lv
		spent += pd.Cost + (lv - 1)
	}
	g.MP = maxi(0, 12-spent)
	if g.Calling == "mystic" {
		g.MP++
	}
	if p.Pow("wings") > 0 && plan == "humanoid" {
		p.Body = append(p.Body, &BodyPart{Name: "Wings", Slot: "wings", Limb: true})
	}
	if p.Pow("stinger") > 0 {
		p.Body = append(p.Body, &BodyPart{Name: "Stinger", Slot: "tail", Limb: true, Natural: natural(g.Sk.PowerName("stinger"), 1, 6, 3, "poison")})
	}
	if spID == "merfolk" || spID == "saurian" || spID == "lizardfolk" {
		p.Set("swimmer")
	}
	p.MaxHP = p.Score(TOU) + 4 + g.Skills["endurance"]*3
	if sp != nil {
		p.MaxHP += sp.Health / 5
	}
	if p.Pow("brittle") > 0 {
		p.AVNat -= 2
	}
	p.HP = p.MaxHP
	g.P = p
	g.kit(cd.ID)
	g.Hunger, g.Thirst = 200, 200
	g.Breath = 30 + p.M(TOU)*5
	// start location
	b := w.burgByID[spec.Start]
	if b == nil || !b.Alive || b.Bandit {
		b = nil
		for i := range w.Burgs {
			if w.Burgs[i].Alive && !w.Burgs[i].Bandit && w.Burgs[i].Layer == "surface" && (b == nil || w.Burgs[i].Pop > b.Pop) {
				b = &w.Burgs[i]
			}
		}
	}
	if b != nil {
		g.StartBurg = b.ID
		g.WX, g.WY = b.P%w.W, b.P/w.W
		g.HomeState = b.State
		if st := w.stateByID[b.State]; st != nil {
			g.HomeFaith = st.Religion
		}
		g.Known[b.P] = true
	} else {
		g.WX, g.WY = w.W/2, w.H/2
	}
	g.StartX, g.StartY = g.WX, g.WY
	g.initRep()
	for _, bb := range w.Burgs {
		if bb.State == g.HomeState && bb.Alive && bb.Capital {
			g.Known[bb.P] = true
		}
	}
	g.enterZone(g.GetZone(g.WX, g.WY, 1, 1, 0), 40, 13, true)
	g.Msg(g.describeStart())
	g.Msg("Press {{W|?}} for help. Walk off the edge to explore; {{W|<}} on the surface opens the world map.")
	g.Mode = "main"
	g.recompute()
	return g
}

func (g *Game) give(it *Item) {
	if it.Stack() {
		for _, o := range g.P.Inv {
			if o.Kind == it.Kind && o.Name == it.Name && o.Effect == it.Effect {
				o.Count += maxi(it.Count, 1)
				return
			}
		}
	}
	if it.Count == 0 && it.Stack() {
		it.Count = 1
	}
	g.P.Inv = append(g.P.Inv, it)
}

func (g *Game) equip(it *Item) bool {
	a := g.P
	slot := it.Slot
	if it.Kind == "light" {
		slot = "float"
	}
	part := a.PartFor(slot)
	if part == nil {
		return false
	}
	if part.Item != nil {
		a.Inv = append(a.Inv, part.Item)
	}
	part.Item = it
	g.removeInv(it)
	return true
}

func (g *Game) removeInv(it *Item) {
	for i, o := range g.P.Inv {
		if o == it {
			g.P.Inv = append(g.P.Inv[:i], g.P.Inv[i+1:]...)
			return
		}
	}
}

func (g *Game) kit(calling string) {
	r := g.R
	switch calling {
	case "warrior":
		g.give(g.MakeWeapon(Pick(r, []string{"longblade", "axe"}), 1, r))
		g.give(g.MakeArmor("body", 1, r))
		g.give(g.MakeArmor("head", 0, r))
	case "scout":
		g.give(g.MakeRanged(1, r))
		g.give(g.MakeAmmo(40))
		g.give(g.MakeWeapon("shortblade", 0, r))
		g.give(g.MakeArmor("back", 1, r))
	case "scholar":
		g.give(g.MakeWeapon("shortblade", 0, r))
		g.give(g.MakeTonic("heal", 2))
		g.give(g.annalsBook(g.HomeState))
	case "tinker":
		g.give(g.MakeWeapon("cudgel", 1, r))
		g.give(g.MakeScrap(r, 12))
		g.give(g.MakeGrenade("fire", 3))
		g.give(g.MakeOddity(r, 1))
	case "mystic":
		g.give(g.MakeWeapon("cudgel", 0, r))
		g.give(g.MakeTonic("heal", 3))
	case "merchant":
		g.give(g.MakeWeapon("shortblade", 1, r))
		g.P.Gold += 250
	case "rogue":
		g.give(g.MakeWeapon("shortblade", 1, r))
		g.give(g.MakeWeapon("shortblade", 0, r))
		g.give(g.MakeGrenade("flash", 2))
	}
	g.give(g.MakeArmor("body", 0, r))
	g.give(g.MakeArmor("feet", 0, r))
	g.give(g.MakeLight(0))
	g.give(g.MakeFood(r, 4))
	g.give(g.MakeWater())
	g.give(g.MakeTonic("heal", 1))
	g.P.Gold += 40 + r.Int(0, 30)
	// auto-equip what fits
	inv := append([]*Item{}, g.P.Inv...)
	for _, it := range inv {
		switch it.Kind {
		case "weapon", "armor", "ranged", "light":
			p := g.P.PartFor(it.Slot)
			if it.Kind == "light" {
				p = g.P.PartFor("float")
			}
			if p != nil && p.Item == nil {
				g.equip(it)
			}
		}
	}
}

// ---- time ----

func (g *Game) actors() []*Actor {
	if g.Cur == nil {
		return nil
	}
	return g.Cur.Actors
}

// spend makes the player pay energy, then runs the world until the player can act again.
func (g *Game) spend(cost int) {
	g.P.Energy -= cost
	for g.P.Energy < 1000 && !g.Dead {
		g.Tick++
		for _, a := range append([]*Actor{}, g.Cur.Actors...) {
			if a.HP <= 0 {
				continue
			}
			a.Energy += a.Quick()
			for a.Energy >= 1000 && a.HP > 0 && !g.Dead {
				g.aiAct(a)
				a.Energy -= 1000
			}
		}
		g.P.Energy += g.P.Quick()
		if g.Tick%10 == 0 {
			g.turn()
		}
	}
	g.cleanup()
	g.recompute()
}

func (g *Game) cleanup() {
	alive := g.Cur.Actors[:0]
	for _, a := range g.Cur.Actors {
		if a.HP > 0 {
			alive = append(alive, a)
		}
	}
	g.Cur.Actors = alive
}

// turn: one game minute. Status effects, hunger, thirst, fire, regeneration.
func (g *Game) turn() {
	g.Turn++
	p := g.P
	hrate := 1
	if p.Pow("glutton") > 0 {
		hrate = 2
	}
	if g.Turn%(2+g.Skills["endurance"]) != 0 {
		g.Hunger += hrate
		g.Thirst++
	}
	if g.Hunger > 2400 && g.Turn%20 == 0 {
		g.hurt(p, 1, nil, "starvation")
	}
	if g.Thirst > 1800 && g.Turn%15 == 0 {
		g.hurt(p, 1, nil, "thirst")
	}
	if g.Hunger == 1200 {
		g.Msg("{{W|You are getting hungry.}}")
	}
	if g.Thirst == 900 {
		g.Msg("{{W|You are thirsty.}}")
	}
	// breath underwater
	if g.Cur.WaterAt(p.X, p.Y) >= 2 && g.Cur.Underwater && p.Pow("gills") == 0 && !p.Has("swimmer") {
		g.Breath--
		if g.Breath == 10 {
			g.Msg("{{R|Your lungs burn. Find air!}}")
		}
		if g.Breath <= 0 {
			g.hurt(p, 3, nil, "drowning")
		}
	} else {
		g.Breath = mini(g.Breath+3, 30+p.M(TOU)*5)
	}
	for _, a := range g.Cur.Actors {
		if a.HP <= 0 {
			continue
		}
		g.statusTick(a)
	}
	g.statusTick(p)
	// regeneration
	regen := 40 - p.M(TOU)*4 - p.Pow("regen")*8
	if p.Pow("albino") > 0 && g.Cur.Z == 0 && g.daylight() {
		regen += 30
	}
	if regen < 6 {
		regen = 6
	}
	if g.Turn%regen == 0 && p.HP < p.MaxHP && p.St("bleeding") == 0 && g.Hunger < 2400 && g.Thirst < 1800 {
		p.HP++
	}
	if p.Pow("regen") >= 3 && g.Turn%200 == 0 {
		for _, bp := range p.Body {
			if bp.Severed {
				bp.Severed = false
				g.Msg("{{G|Your " + strings.ToLower(bp.Name) + " grows back.}}")
				break
			}
		}
	}
	for k, v := range p.Cool {
		if v > 0 {
			p.Cool[k] = v - 1
		}
	}
	g.fireTick()
}

func (g *Game) statusTick(a *Actor) {
	if a.Status == nil {
		return
	}
	for k, v := range a.Status {
		if v <= 0 {
			delete(a.Status, k)
			continue
		}
		switch k {
		case "bleeding":
			g.hurt(a, 1, nil, "blood loss")
			if g.Cur.In(a.X, a.Y) {
				g.Cur.Liq[a.Y*MW+a.X] = 1
			}
			if a == g.P && a.Pow("hemophilia") > 0 {
				continue
			}
		case "burning":
			g.hurt(a, g.R.Int(1, 3), nil, "fire")
			if g.Cur.WaterAt(a.X, a.Y) > 0 {
				a.Status[k] = 0
				continue
			}
		case "poisoned":
			if g.Turn%3 == 0 {
				g.hurt(a, 1, nil, "poison")
			}
		case "regen":
			if a.HP < a.MaxHP {
				a.HP++
			}
		}
		if a.HP <= 0 {
			return
		}
		a.Status[k] = v - 1
		if a.Status[k] <= 0 {
			delete(a.Status, k)
			if a == g.P {
				g.Msg("You are no longer " + k + ".")
			}
		}
	}
	if a != g.P && a.Has("regen") && a.HP < a.MaxHP && g.Turn%3 == 0 {
		a.HP++
	}
}

func (g *Game) fireTick() {
	z := g.Cur
	changed := false
	for i := range z.Fire {
		if z.Fire[i] == 0 {
			continue
		}
		changed = true
		z.Fire[i]--
		x, y := i%MW, i/MW
		if a := z.ActorAt(x, y); a != nil {
			a.AddSt("burning", 3)
		}
		if g.P.X == x && g.P.Y == y {
			g.P.AddSt("burning", 3)
		}
		if z.Fire[i] == 0 {
			t := z.T[i]
			if t == TTree || t == TBush || t == TTallGrass || t == TDoor || t == TDoorOpen || t == TReeds || t == TShelf || t == TBridge {
				if t == TBridge {
					z.T[i] = TDeep
				} else {
					z.T[i] = TAsh
				}
			} else if t == TGrass {
				z.T[i] = TAsh
			}
			if z.Liq[i] == 2 {
				z.Liq[i] = 0
			}
		}
		if g.R.Chance(0.25) {
			d := dirs8[g.R.Int(0, 7)]
			nx, ny := x+d[0], y+d[1]
			if z.In(nx, ny) {
				ni := ny*MW + nx
				if z.Fire[ni] == 0 && (z.T[ni].D().Burn || z.Liq[ni] == 2) {
					z.Fire[ni] = uint8(g.R.Int(4, 9))
				}
			}
		}
	}
	if changed {
		g.computeLight(z)
	}
}

func (g *Game) daylight() bool {
	h := (g.Turn/60 + 8) % 24
	return h >= 6 && h < 20
}

func (g *Game) timeOfDay() string {
	h := (g.Turn/60 + 8) % 24
	names := []string{"Midnight", "Deep night", "Deep night", "Small hours", "Small hours", "Dawn", "Dawn", "Morning", "Morning", "Morning", "Late morning", "Noon", "Noon", "Afternoon", "Afternoon", "Afternoon", "Late afternoon", "Dusk", "Dusk", "Evening", "Evening", "Night", "Night", "Night"}
	day := (g.Turn/60+8)/24 + 1
	return fmt.Sprintf("%s, day %d", names[h], day)
}

// ---- zones & movement ----

func (g *Game) enterZone(zn *Zone, x, y int, place bool) {
	if g.Cur != nil && g.Cur != zn {
		// followers come along
		var fol []*Actor
		rest := g.Cur.Actors[:0]
		for _, a := range g.Cur.Actors {
			if a.AI == "follower" && a.HP > 0 && abs(a.X-g.P.X) <= 8 && abs(a.Y-g.P.Y) <= 8 {
				fol = append(fol, a)
			} else {
				rest = append(rest, a)
			}
		}
		g.Cur.Actors = rest
		for _, a := range fol {
			zn.Actors = append(zn.Actors, a)
			a.X, a.Y = x, y
			g.nudge(zn, a)
		}
	}
	g.Cur = zn
	g.CurKey = zn.Key
	g.OnWorld = false
	g.WX, g.WY = zn.WX, zn.WY
	g.P.X, g.P.Y = clampi(x, 0, MW-1), clampi(y, 0, MH-1)
	if place {
		if !zn.Walkable(g.P.X, g.P.Y) || zn.ActorAt(g.P.X, g.P.Y) != nil || zn.WaterAt(g.P.X, g.P.Y) >= 2 && !zn.Underwater {
			if fx, fy, ok := g.nearestFree(zn, g.P.X, g.P.Y); ok {
				g.P.X, g.P.Y = fx, fy
			}
		}
	}
	for _, a := range zn.Actors {
		if a.AI == "follower" {
			g.nudge(zn, a)
		}
	}
	if !zn.Visited {
		zn.Visited = true
		g.discover(zn)
	}
	g.questSpawn(zn)
	p := g.W.Idx(zn.WX, zn.WY)
	g.Known[p] = true
	g.Msg("{{C|" + zn.Name + "}}")
}

func (g *Game) nudge(zn *Zone, a *Actor) {
	a.X, a.Y = clampi(a.X, 0, MW-1), clampi(a.Y, 0, MH-1)
	if zn.Walkable(a.X, a.Y) && (zn.ActorAt(a.X, a.Y) == nil || zn.ActorAt(a.X, a.Y) == a) && !(a.X == g.P.X && a.Y == g.P.Y) {
		return
	}
	for r := 1; r < 6; r++ {
		for dy := -r; dy <= r; dy++ {
			for dx := -r; dx <= r; dx++ {
				nx, ny := a.X+dx, a.Y+dy
				if zn.Walkable(nx, ny) && zn.ActorAt(nx, ny) == nil && !(nx == g.P.X && ny == g.P.Y) && zn.WaterAt(nx, ny) < 2 {
					a.X, a.Y = nx, ny
					return
				}
			}
		}
	}
}

func (g *Game) nearestFree(zn *Zone, x, y int) (int, int, bool) {
	for r := 0; r < 40; r++ {
		for dy := -r; dy <= r; dy++ {
			for dx := -r; dx <= r; dx++ {
				if abs(dx) != r && abs(dy) != r {
					continue
				}
				nx, ny := x+dx, y+dy
				t := zn.At(nx, ny)
				if zn.Walkable(nx, ny) && zn.ActorAt(nx, ny) == nil && (zn.WaterAt(nx, ny) < 2 || zn.Underwater) && t != TLava && t != TChasm && (t != TVoid || !zn.Sky) {
					return nx, ny, true
				}
			}
		}
	}
	return x, y, false
}

func (g *Game) discover(zn *Zone) {
	k := zn.Kind
	if k == "town" || k == "ruin" || k == "lair" || k == "wonder" || k == "crypt" || k == "underdark" && zn.ZX == 1 && zn.ZY == 1 {
		key := zn.Name
		if !g.Visited[key] {
			g.Visited[key] = true
			xp := 25 + zn.Level*5
			g.Msg(fmt.Sprintf("You discover {{Y|%s}}! {{c|+%d XP}}", zn.Name, xp))
			g.gainXP(xp)
		}
	}
}

func (g *Game) canSwim() bool {
	return g.P.Pow("gills") > 0 || g.P.Has("swimmer") || g.P.Pow("wings") > 0
}

// MoveP is a bump: attack, talk, open, interact or step.
func (g *Game) MoveP(dx, dy int) {
	p := g.P
	if p.St("frozen") > 0 || p.St("stunned") > 0 {
		g.Msg("You can't move!")
		g.spend(1000)
		return
	}
	if p.St("confused") > 0 && g.R.Chance(0.5) {
		d := dirs8[g.R.Int(0, 7)]
		dx, dy = d[0], d[1]
	}
	nx, ny := p.X+dx, p.Y+dy
	z := g.Cur
	if !z.In(nx, ny) {
		g.crossEdge(nx, ny)
		return
	}
	if a := z.ActorAt(nx, ny); a != nil {
		if g.Hostile(a) {
			g.melee(p, a)
			g.spend(1000)
			return
		}
		if a.AI == "follower" {
			a.X, a.Y, p.X, p.Y = p.X, p.Y, nx, ny
			g.spend(1000)
			return
		}
		g.openTalk(a)
		return
	}
	t := z.At(nx, ny)
	if t == TDoor {
		z.Set(nx, ny, TDoorOpen)
		g.Msg("You open the door.")
		g.spend(1000)
		return
	}
	if t.D().Interact || t == TStatue {
		g.interact(nx, ny)
		return
	}
	phasing := p.St("phased") > 0
	if !t.D().Walk && !phasing {
		if t == TRock || t == TWall {
			if p.Pow("burrow") > 0 {
				g.dig(nx, ny)
				return
			}
		}
		return
	}
	if t == TLava && p.Pow("wings") == 0 {
		g.Popup = []string{"That's {{R|lava}}. Step in anyway? (y/n)"}
		g.PopT = "Danger"
		g.Mode = "confirm"
		g.Pend = fmt.Sprintf("step:%d,%d", nx, ny)
		return
	}
	g.stepTo(nx, ny)
}

func (g *Game) stepTo(nx, ny int) {
	p := g.P
	z := g.Cur
	cost := 1000
	w := z.WaterAt(nx, ny)
	if w >= 2 && !z.Underwater && !g.canSwim() {
		cost = 2000
		if p.X == nx && p.Y == ny {
			return
		}
		if z.WaterAt(p.X, p.Y) < 2 {
			g.Msg("You wade into deep water and start to swim.")
		}
	}
	if z.At(nx, ny) == TBush || z.At(nx, ny) == TReeds {
		cost = 1300
	}
	p.X, p.Y = nx, ny
	t := z.At(nx, ny)
	switch t {
	case TLava:
		if p.Pow("wings") == 0 {
			p.AddSt("burning", 6)
			g.hurt(p, g.R.Int(10, 20), nil, "lava")
		}
	case TVoid, TChasm:
		if p.Pow("wings") == 0 && (z.Sky || t == TChasm) {
			g.fall()
			return
		}
	}
	if its := z.ItemsAt(nx, ny); len(its) > 0 {
		if len(its) == 1 {
			g.Msg("You see here " + its[0].Display() + ".")
		} else {
			g.Msg(fmt.Sprintf("There are %d items here. (g to pick up)", len(its)))
		}
	}
	if note, ok := z.Notes[ny*MW+nx]; ok && (t == TUpdraft || t == TUp || t == TDown) {
		g.Msg(note)
	}
	g.spend(cost)
}

func (g *Game) fall() {
	z := g.Cur
	g.Msg("{{R|You plummet!}}")
	if z.Sky {
		below := g.GetZone(z.WX, z.WY, z.ZX, z.ZY, 0)
		g.enterZone(below, g.P.X, g.P.Y, true)
		g.hurt(g.P, g.R.Int(15, 30), nil, "a fall from the sky")
	} else {
		below := g.GetZone(z.WX, z.WY, z.ZX, z.ZY, z.Z+1)
		g.enterZone(below, g.P.X, g.P.Y, true)
		g.hurt(g.P, g.R.Int(6, 14), nil, "a fall")
	}
	g.spend(1000)
}

func (g *Game) crossEdge(nx, ny int) {
	z := g.Cur
	wx, wy, zx, zy := z.WX, z.WY, z.ZX, z.ZY
	x, y := nx, ny
	if nx < 0 {
		zx--
		x = MW - 1
	} else if nx >= MW {
		zx++
		x = 0
	}
	if ny < 0 {
		zy--
		y = MH - 1
	} else if ny >= MH {
		zy++
		y = 0
	}
	if zx < 0 {
		zx, wx = 2, wx-1
	} else if zx > 2 {
		zx, wx = 0, wx+1
	}
	if zy < 0 {
		zy, wy = 2, wy-1
	} else if zy > 2 {
		zy, wy = 0, wy+1
	}
	if !g.W.In(wx, wy) {
		g.Msg("The world ends here.")
		return
	}
	g.enterZone(g.GetZone(wx, wy, zx, zy, z.Z), x, y, true)
	g.spend(1000)
}

func (g *Game) Ascend() {
	z := g.Cur
	t := z.At(g.P.X, g.P.Y)
	switch {
	case t == TUp && z.Z > 0:
		g.enterZone(g.GetZone(z.WX, z.WY, z.ZX, z.ZY, z.Z-1), g.P.X, g.P.Y, true)
		g.spend(1000)
	case t == TUpdraft || (g.P.Pow("wings") > 0 && z.Z == 0 && g.W.skyOpen[g.W.Idx(z.WX, z.WY)]):
		g.Msg("You rise on the wind toward the " + g.W.LayerName("sky", "sky") + ".")
		g.enterZone(g.GetZone(z.WX, z.WY, z.ZX, z.ZY, -1), g.P.X, g.P.Y, true)
		g.spend(1000)
	case z.Z == 0:
		g.openWorldMap()
	case z.Z < 0:
		g.Msg("There is nothing higher than this.")
	default:
		g.Msg("There is no way up here.")
	}
}

func (g *Game) Descend() {
	z := g.Cur
	t := z.At(g.P.X, g.P.Y)
	switch {
	case t == TDown:
		g.enterZone(g.GetZone(z.WX, z.WY, z.ZX, z.ZY, z.Z+1), g.P.X, g.P.Y, true)
		g.spend(1000)
	case t == TUp && z.Z < 0:
		g.enterZone(g.GetZone(z.WX, z.WY, z.ZX, z.ZY, 0), g.P.X, g.P.Y, true)
		g.spend(1000)
	case z.Z < 0 && g.P.Pow("wings") > 0:
		g.Msg("You glide down.")
		g.enterZone(g.GetZone(z.WX, z.WY, z.ZX, z.ZY, 0), g.P.X, g.P.Y, true)
		g.spend(1000)
	default:
		g.Msg("There is no way down here.")
	}
}

// ---- world map ----

func (g *Game) openWorldMap() {
	if g.Cur.Z != 0 {
		g.Msg("You can only travel overland from the surface.")
		return
	}
	if g.nearHostile() {
		g.Msg("{{R|You can't travel with enemies nearby!}}")
		return
	}
	g.OnWorld = true
	g.Mode = "world"
	g.CX, g.CY = g.WX, g.WY
}

func (g *Game) nearHostile() bool {
	for _, a := range g.Cur.Actors {
		if a.HP > 0 && g.Hostile(a) && g.visible(a.X, a.Y) && abs(a.X-g.P.X)+abs(a.Y-g.P.Y) < 20 {
			return true
		}
	}
	return false
}

func (g *Game) WorldMove(dx, dy int) {
	nx, ny := g.WX+dx, g.WY+dy
	if !g.W.In(nx, ny) {
		return
	}
	np := g.W.Idx(nx, ny)
	if g.W.IsWater(np) && !g.canSwim() {
		g.Msg("The sea stretches away. You need " + g.Sk.Gills() + ", or " + g.Sk.Ferry + " at a port.")
		return
	}
	g.WX, g.WY = nx, ny
	g.CX, g.CY = nx, ny
	cost := 300
	if g.P.Pow("wings") > 0 {
		cost = 180
	}
	cost -= g.Skills["survival"] * 40
	if g.W.Road[np] > 0 {
		cost = cost * 2 / 3
	}
	if g.W.Height[np] >= 70 {
		cost = cost * 3 / 2
	}
	for i := 0; i < cost; i++ {
		g.Turn++
		if g.Turn%(2+g.Skills["endurance"]) != 0 {
			g.Hunger++
			g.Thirst++
		}
		if g.Turn%30 == 0 && g.P.HP < g.P.MaxHP {
			g.P.HP++
		}
	}
	g.Known[np] = true
	for _, d := range dirs8 {
		if g.W.In(nx+d[0], ny+d[1]) {
			q := g.W.Idx(nx+d[0], ny+d[1])
			if g.W.Burg[q] > 0 {
				g.Known[q] = true
			}
		}
	}
	if g.Hunger > 2400 || g.Thirst > 1800 {
		g.hurt(g.P, 3, nil, "privation on the road")
		g.Msg("{{R|The road is hard without food and water.}}")
	}
	// ambushes
	chance := 0.12 - 0.03*float64(g.Skills["survival"])
	if g.W.State[np] == 0 {
		chance += 0.06
	}
	if g.W.Burg[np] > 0 {
		chance = 0
	}
	if g.R.Chance(chance) {
		g.Msg("{{R|You are waylaid on the road!}}")
		zx, zy := g.R.Int(0, 2), g.R.Int(0, 2)
		zn := g.GetZone(nx, ny, zx, zy, 0)
		lvl := zn.Level
		tp := pickCreature(g.R, g.W.Biome[np], 's', lvl)
		if g.R.Chance(0.4) {
			tp = "bandit"
		}
		n := g.R.Int(2, 4)
		g.enterZone(zn, 40, 12, true)
		for i := 0; i < n; i++ {
			a := g.Spawn(tp, lvl, g.R)
			a.X, a.Y = g.P.X+g.R.Int(-8, 8), g.P.Y+g.R.Int(-4, 4)
			g.nudge(zn, a)
			zn.Actors = append(zn.Actors, a)
		}
		g.Mode = "main"
		g.recompute()
	}
}

func (g *Game) EnterFromWorld() {
	g.enterZone(g.GetZone(g.WX, g.WY, 1, 1, 0), 40, 12, true)
	g.Mode = "main"
	g.recompute()
}

// ---- xp ----

func XPFor(level int) int { return level*level*60 + 40 }

func (g *Game) gainXP(n int) {
	g.XP += n
	for g.XP >= XPFor(g.P.Level) {
		g.XP -= XPFor(g.P.Level)
		g.P.Level++
		hp := maxi(1, 3+g.P.M(TOU)) + g.Skills["endurance"]*3
		g.P.MaxHP += hp
		g.P.HP += hp
		g.SP++
		if g.P.Score(INT) >= 18 {
			g.SP++
		}
		g.MP++
		if g.P.Level%3 == 0 {
			g.AP++
		}
		g.Msg(fmt.Sprintf("{{W|You reach level %d!}} +%d HP. Spend points on the character screen (c).", g.P.Level, hp))
	}
}

func (g *Game) addSecret(text, kind string, p, year int) bool {
	key := kind + ":" + text
	if g.SecretSet[key] {
		return false
	}
	g.SecretSet[key] = true
	g.Secrets = append(g.Secrets, Secret{Text: text, P: p, Kind: kind, Year: year})
	if p >= 0 {
		g.Known[p] = true
	}
	return true
}

// ---- save ----

func (g *Game) Save() string {
	b, err := json.Marshal(g)
	if err != nil {
		return ""
	}
	return string(b)
}

func LoadGame(w *WorldData, s string) (*Game, error) {
	g := &Game{}
	if err := json.Unmarshal([]byte(s), g); err != nil {
		return nil, err
	}
	g.W = w
	g.Sk = SkinFor(g.Genre)
	g.sc = NewScreen()
	for _, z := range g.Zones {
		if z.Lit == nil {
			z.Lit = make([]bool, MW*MH)
		}
		if z.Notes == nil {
			z.Notes = map[int]string{}
		}
		g.computeLight(z)
	}
	g.Cur = g.Zones[g.CurKey]
	if g.Cur == nil {
		g.Cur = g.GetZone(g.WX, g.WY, 1, 1, 0)
	}
	if g.R == nil {
		g.R = NewRng(g.Seed)
	}
	g.Mode = "main"
	if g.OnWorld {
		g.Mode = "world"
	}
	if g.Dead {
		g.Mode = "dead"
	}
	g.recompute()
	return g, nil
}

// PendingDeeds returns deeds not yet handed back to Realmforge's chronicle.
func (g *Game) PendingDeeds() string {
	d := g.Deeds[g.DeedsSent:]
	g.DeedsSent = len(g.Deeds)
	if len(d) == 0 {
		return "[]"
	}
	b, _ := json.Marshal(d)
	return string(b)
}

func (g *Game) deed(kind, text string, mon, art int) {
	cell := -1
	p := g.W.Idx(g.Cur.WX, g.Cur.WY)
	if p >= 0 && p < len(g.W.Cell) {
		cell = g.W.Cell[p]
	}
	g.Deeds = append(g.Deeds, Deed{Kind: kind, Text: text, Monster: mon, Artifact: art, Cell: cell, Year: g.W.Year})
}

func sortedKeys(m map[string]int) []string {
	var k []string
	for s := range m {
		k = append(k, s)
	}
	sort.Strings(k)
	return k
}

func (s *Skin) Gills() string { return s.PowerName("gills") }

// Warp jumps straight into a zone (debug and fuzzing).
func (g *Game) Warp(wx, wy, zx, zy, z int) {
	g.enterZone(g.GetZone(clampi(wx, 0, g.W.W-1), clampi(wy, 0, g.W.H-1), zx, zy, z), 40, 12, true)
	g.Mode = "main"
	g.recompute()
}

// DebugTalkNearest opens dialogue with the nearest friendly NPC (dev tools).
func (g *Game) DebugTalkNearest() {
	var best *Actor
	for _, a := range g.Cur.Actors {
		if a.HP > 0 && !g.Hostile(a) && (best == nil || dist(a, g.P) < dist(best, g.P)) && (a.Role == "elder" || a.Role == "notable" || best == nil) {
			best = a
		}
	}
	if best != nil {
		g.openTalk(best)
	}
}

// DebugGotoStairs teleports onto the nearest way down, searching neighbouring zones (dev tools).
func (g *Game) DebugGotoStairs() {
	for r := 0; r < 6; r++ {
		for dy := -r; dy <= r; dy++ {
			for dx := -r; dx <= r; dx++ {
				for zx := 0; zx < 3; zx++ {
					for zy := 0; zy < 3; zy++ {
						wx, wy := g.WX+dx, g.WY+dy
						if !g.W.In(wx, wy) || g.W.IsWater(g.W.Idx(wx, wy)) {
							continue
						}
						for _, s := range g.stairsDown(wx, wy, zx, zy, g.Cur.Z) {
							g.enterZone(g.GetZone(wx, wy, zx, zy, g.Cur.Z), s[0], s[1], false)
							g.recompute()
							return
						}
					}
				}
			}
		}
	}
}

// Meta is the small summary a save list shows.
func (g *Game) Meta() string {
	b, _ := json.Marshal(map[string]any{"name": g.P.Name, "level": g.P.Level, "calling": g.Sk.Calling(g.Calling), "species": g.P.Species,
		"world": g.W.Name, "genre": g.Genre, "year": g.W.Year, "place": g.Cur.Name, "dead": g.Dead, "seed": g.Seed})
	return string(b)
}
