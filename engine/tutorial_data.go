package engine

import (
	"encoding/json"
	"sort"
	"strconv"
	"strings"
)

// Tutorial step table (prefix tu). Each step is detected from real state in tutorial.go.
// "soft" steps cannot be observed from the hooks (journal screen) and finish by timer or by hook.

type tuStep struct {
	ID   string
	Obj  string // one-line objective for the sidebar
	Text string // message shown when the step begins
	Soft int    // >0: auto-complete after this many game minutes
}

var tuSteps = []tuStep{
	{ID: "move", Obj: "Walk about 5 tiles (arrows, numpad or click).",
		Text: "Welcome. Move with the arrow keys, numpad or hjklyubn, or click a tile to walk there. The town streets are safe ground: stay near them until you are stronger."},
	{ID: "look", Obj: "Press x (or L) to look around, Esc to stop.",
		Text: "Press x to look: move the cursor over anything to learn what it is. Esc closes the cursor."},
	{ID: "talk", Obj: "Walk into a townsperson to talk.",
		Text: "Walk into a person to talk. Choose an option with the arrows and Enter, or press its letter."},
	{ID: "buy", Obj: "Trade with a merchant and buy something.",
		Text: "Merchants have a trade option when you talk. Pick it and buy any one thing you can afford. Gold is shown on the sidebar."},
	{ID: "equip", Obj: "Press i, pick an item, and wear or wield it.",
		Text: "Press i for your pack. Select a weapon or armor and choose to equip it. Equipped gear is listed on the character screen (c)."},
	{ID: "fight", Obj: "Defeat one hostile creature (bump to attack).",
		Text: "When something hostile shows up, bump into it to melee, f to fire a ranged weapon, a for powers. Pick a weak target and press R to rest if you get hurt. Retreat with < or by walking away."},
	{ID: "rest", Obj: "Eat or drink (e / d), or rest with R.",
		Text: "Press e to eat, d to drink, or R to rest until healed (not with foes nearby). Hunger and thirst creep up with time."},
	{ID: "journal", Obj: "Press j to read your journal, Esc to close.", Soft: 25,
		Text: "Press j for the journal: quests, notes and what you have learned. Esc closes it."},
	{ID: "map", Obj: "Press w to open the world map, Esc to close.",
		Text: "Press w for the world map. Move the cursor to a destination and press Enter to travel; Esc goes back."},
	{ID: "quest", Obj: "Ask a person for work and accept a quest.",
		Text: "People with work offer quests in their talk options. Accept one; it shows up in your journal (j)."},
	{ID: "ability", Obj: "Press a to view your abilities and powers.",
		Text: "Press a to see the powers you can use. Pick one to use it; some need a target or a direction."},
	{ID: "palette", Obj: "Press ; to open the command palette.",
		Text: "Press ; for the command palette: type a few letters to find any screen, action or ability and run it."},
}

// hint ids that tutorial steps already teach; silenced while the guide is on so tips never double up.
var tuQuiet1 = []string{"town", "hostile", "lowhp", "hungry", "world"}
var tuQuiet2 = []string{"hungry", "tired"}

// ---------------------------------------------------------------------------------------------
// World editor-lite (mode "wed", ability wed_edit, console `wed`). Edits live in g.ext("wed") and are
// applied to the loaded world data in memory (species names/descriptions, culture names, town names),
// to the player's homes, to places.go waypoints (landmark notes) and to a house-tag table. Other
// systems can read them through g.wedName / g.wedHouseTag.

type wedSpec struct {
	N string `json:"n,omitempty"`
	D string `json:"d,omitempty"`
}

type wedMark struct {
	N string `json:"n"`
	T string `json:"t,omitempty"`
	X int    `json:"x"`
	Y int    `json:"y"`
	P int    `json:"p,omitempty"`
}

type wedState struct {
	Spec   map[string]wedSpec `json:"sp,omitempty"`
	Cult   map[string]string  `json:"cu,omitempty"`
	Town   map[string]string  `json:"tn,omitempty"`
	Home   map[string]string  `json:"hm,omitempty"`
	Houses map[string]string  `json:"ho,omitempty"`
	Marks  []wedMark          `json:"mk,omitempty"`

	applied  *WorldData
	lastTown map[int]string
	tab, sel int
	prompt   bool
	fields   []string
	vals     []string
	fi       int
	rows     []wedRow
	text     []string
	top      int
	note     string
}

type wedRow struct {
	Label string
	ID    string
	Cur   []string // current values per field (prefill)
}

func (g *Game) wed() *wedState {
	return g.ext("wed", func() any { return &wedState{} }).(*wedState)
}

var wedTabs = []string{"Species", "Cultures", "Towns", "Homes", "Landmarks", "Houses"}

// wedName is the lookup other systems call: kind is species, species_desc, culture, town or home;
// id is the species id, culture id, burg id or zone key as a string. It returns orig when unedited.
func (g *Game) wedName(kind, id, orig string) string {
	st := g.wed()
	switch kind {
	case "species":
		if v := st.Spec[id].N; v != "" {
			return v
		}
	case "species_desc":
		if v := st.Spec[id].D; v != "" {
			return v
		}
	case "culture":
		if v := st.Cult[id]; v != "" {
			return v
		}
	case "town":
		if v := st.Town[id]; v != "" {
			return v
		}
	case "home":
		if v := st.Home[id]; v != "" {
			return v
		}
	}
	return orig
}

// wedHouseTag returns the tag you gave a house (family) name, or "".
func (g *Game) wedHouseTag(house string) string { return g.wed().Houses[house] }

type wedOrigs struct {
	spN, spD map[string]string
	cu, bu   map[int]string
}

var wedOrig = map[*WorldData]*wedOrigs{}

func wedClean(s string) string {
	s = strings.Map(func(r rune) rune {
		if r == '{' || r == '}' || r == '|' || r < 32 {
			return -1
		}
		return r
	}, s)
	return strings.TrimSpace(s)
}

// wedApply pushes the edits into the in-memory world data (names only; originals are remembered
// per world so reverting an edit or loading another save restores them).
func (g *Game) wedApply() {
	w := g.W
	if w == nil {
		return
	}
	st := g.wed()
	o := wedOrig[w]
	if o == nil {
		o = &wedOrigs{spN: map[string]string{}, spD: map[string]string{}, cu: map[int]string{}, bu: map[int]string{}}
		for _, s := range w.Species {
			o.spN[s.ID], o.spD[s.ID] = s.Name, s.Desc
		}
		for _, c := range w.Cultures {
			o.cu[c.ID] = c.Name
		}
		for _, b := range w.Burgs {
			o.bu[b.ID] = b.Name
		}
		wedOrig[w] = o
	}
	for i := range w.Species {
		s := &w.Species[i]
		s.Name = g.wedName("species", s.ID, o.spN[s.ID])
		s.Desc = g.wedName("species_desc", s.ID, o.spD[s.ID])
	}
	for i := range w.Cultures {
		c := &w.Cultures[i]
		c.Name = g.wedName("culture", strconv.Itoa(c.ID), o.cu[c.ID])
	}
	if st.lastTown == nil {
		st.lastTown = map[int]string{}
	}
	for i := range w.Burgs {
		b := &w.Burgs[i]
		nn := g.wedName("town", strconv.Itoa(b.ID), o.bu[b.ID])
		b.Name = nn
		if _, had := st.lastTown[b.ID]; had || nn != o.bu[b.ID] {
			for _, z := range g.Zones {
				if z.Burg == b.ID && z.Kind == "town" && (z.Name == o.bu[b.ID] || z.Name == st.lastTown[b.ID]) {
					z.Name = nn
				}
			}
		}
		if nn != o.bu[b.ID] {
			st.lastTown[b.ID] = nn
		} else {
			delete(st.lastTown, b.ID)
		}
	}
	if g.Cur != nil && g.Cur.Kind == "town" && g.Cur.Burg > 0 {
		if b := w.burgByID[g.Cur.Burg]; b != nil && st.Town[strconv.Itoa(b.ID)] != "" {
			g.Cur.Name = b.Name
		}
	}
	hs := g.homeSt()
	for i := range hs.Homes {
		if v := st.Home[hs.Homes[i].Key]; v != "" {
			hs.Homes[i].Name = v
		}
	}
	st.applied = w
}

func init() {
	modeKeys["wed"] = wedKey
	modeDraws["wed"] = wedDraw
	modeKeys["wedx"] = wedTextKey
	modeDraws["wedx"] = wedTextDraw
	turnHooks = append(turnHooks, func(g *Game) {
		if g.W != nil && g.wed().applied != g.W {
			g.wedApply()
		}
	})
	enterHooks = append(enterHooks, func(g *Game, zn *Zone) {
		if g.W != nil && g.wed().applied != g.W {
			g.wedApply()
		}
	})
	paEntries = append(paEntries, paEntry{Name: "World editor", Keywords: "wed edit rename species culture town home landmark house export", Cat: "Notes", Func: func(g *Game) { g.wedOpen() }})
	cnCmds = append(cnCmds, cnCmd{"wed", []string{"worldedit"}, "wed [list|species|culture|town|home|mark|house|clear|export]", "tweak names, add landmark notes, tag houses (try `wed help`)", false, wedCmd})
}

// wedAbilities lists the editor ability (hook: powers.go Abilities).
func (g *Game) wedAbilities() []Ability {
	return []Ability{{ID: "wed_edit", Name: "Edit the world (names, notes)", Target: "self"}}
}

// wedUse runs wed_* abilities (hook: powers.go usePower).
func (g *Game) wedUse(id string) {
	if id == "wed_edit" {
		g.wedOpen()
	}
}

func (g *Game) wedOpen() {
	st := g.wed()
	if g.W != nil && st.applied != g.W {
		g.wedApply()
	}
	st.sel, st.prompt, st.note = 0, false, ""
	g.Mode = "wed"
	g.wedRebuild()
}

// ---- rows

func (g *Game) wedBurgs() []*BurgData {
	var l []*BurgData
	for i := range g.W.Burgs {
		if b := &g.W.Burgs[i]; b.Alive && b.Layer == "surface" {
			l = append(l, b)
		}
	}
	sort.SliceStable(l, func(i, j int) bool {
		hi, hj := l[i].ID == g.StartBurg, l[j].ID == g.StartBurg
		if hi != hj {
			return hi
		}
		if l[i].Pop != l[j].Pop {
			return l[i].Pop > l[j].Pop
		}
		return l[i].ID < l[j].ID
	})
	if len(l) > 300 {
		l = l[:300]
	}
	return l
}

func (g *Game) wedHouseList() []string {
	seen := map[string]bool{}
	var l []string
	add := func(cid int) {
		if c := g.W.cultByID[cid]; c != nil {
			for _, h := range c.Houses {
				if !seen[h] {
					seen[h] = true
					l = append(l, h)
				}
			}
		}
	}
	if b := g.W.burgByID[g.StartBurg]; b != nil {
		add(b.Culture)
	}
	if g.Cur != nil && g.Cur.Burg > 0 {
		if b := g.W.burgByID[g.Cur.Burg]; b != nil {
			add(b.Culture)
		}
	}
	for h := range g.wed().Houses {
		if !seen[h] {
			seen[h] = true
			l = append(l, h)
		}
	}
	return l
}

func (g *Game) wedRebuild() {
	st := g.wed()
	st.rows = nil
	if g.W == nil {
		return
	}
	mark := func(on bool) string {
		if on {
			return " {{G|*}}"
		}
		return ""
	}
	switch st.tab {
	case 0:
		for _, s := range g.W.Species {
			e := st.Spec[s.ID]
			st.rows = append(st.rows, wedRow{Label: s.Name + mark(e.N != "" || e.D != ""), ID: s.ID, Cur: []string{s.Name, s.Desc}})
		}
	case 1:
		for _, c := range g.W.Cultures {
			id := strconv.Itoa(c.ID)
			st.rows = append(st.rows, wedRow{Label: c.Name + " {{K|(" + c.Species + ")}}" + mark(st.Cult[id] != ""), ID: id, Cur: []string{c.Name}})
		}
	case 2:
		for _, b := range g.wedBurgs() {
			id := strconv.Itoa(b.ID)
			l := b.Name
			if b.ID == g.StartBurg {
				l += " {{K|(home town)}}"
			}
			st.rows = append(st.rows, wedRow{Label: l + mark(st.Town[id] != ""), ID: id, Cur: []string{b.Name}})
		}
	case 3:
		for _, h := range g.homeSt().Homes {
			st.rows = append(st.rows, wedRow{Label: h.Name + " {{K|(" + h.Kind + ")}}" + mark(st.Home[h.Key] != ""), ID: h.Key, Cur: []string{h.Name}})
		}
	case 4:
		st.rows = append(st.rows, wedRow{Label: "{{G|[ place a new landmark at your current spot ]}}", ID: "new", Cur: []string{"", ""}})
		for i, m := range st.Marks {
			st.rows = append(st.rows, wedRow{Label: m.N + " {{K|(" + strconv.Itoa(m.X) + "," + strconv.Itoa(m.Y) + ")}}", ID: strconv.Itoa(i), Cur: []string{m.N, m.T}})
		}
	case 5:
		for _, h := range g.wedHouseList() {
			t := st.Houses[h]
			l := h
			if t != "" {
				l += " {{K|-}} {{Y|" + t + "}}"
			}
			st.rows = append(st.rows, wedRow{Label: l, ID: h, Cur: []string{t}})
		}
	}
	if st.sel >= len(st.rows) {
		st.sel = maxi(0, len(st.rows)-1)
	}
}

func wedFieldNames(tab int) []string {
	switch tab {
	case 0:
		return []string{"Name", "Description"}
	case 4:
		return []string{"Landmark name", "Note"}
	case 5:
		return []string{"Tag (empty clears)"}
	}
	return []string{"New name"}
}

// ---- mutations (used by the screen and the console)

func (g *Game) wedSetSpecies(id, name, desc string) bool {
	if g.W.specByID[id] == nil {
		return false
	}
	st := g.wed()
	if st.Spec == nil {
		st.Spec = map[string]wedSpec{}
	}
	e := st.Spec[id]
	if name = wedClean(name); name != "" {
		e.N = name
	}
	if desc = wedClean(desc); desc != "" {
		e.D = desc
	}
	st.Spec[id] = e
	g.wedApply()
	return true
}

func (g *Game) wedSetCulture(id, name string) bool {
	n, err := strconv.Atoi(id)
	if err != nil || g.W.cultByID[n] == nil || wedClean(name) == "" {
		return false
	}
	st := g.wed()
	if st.Cult == nil {
		st.Cult = map[string]string{}
	}
	st.Cult[id] = wedClean(name)
	g.wedApply()
	return true
}

func (g *Game) wedSetTown(id, name string) bool {
	n, err := strconv.Atoi(id)
	if err != nil || g.W.burgByID[n] == nil || wedClean(name) == "" {
		return false
	}
	st := g.wed()
	if st.Town == nil {
		st.Town = map[string]string{}
	}
	st.Town[id] = wedClean(name)
	g.wedApply()
	return true
}

func (g *Game) wedSetHome(key, name string) bool {
	if wedClean(name) == "" {
		return false
	}
	ok := false
	for _, h := range g.homeSt().Homes {
		if h.Key == key {
			ok = true
		}
	}
	if !ok {
		return false
	}
	st := g.wed()
	if st.Home == nil {
		st.Home = map[string]string{}
	}
	st.Home[key] = wedClean(name)
	g.wedApply()
	return true
}

func (g *Game) wedSetHouse(house, tag string) {
	st := g.wed()
	if st.Houses == nil {
		st.Houses = map[string]string{}
	}
	if tag = wedClean(tag); tag == "" {
		delete(st.Houses, house)
	} else {
		st.Houses[house] = tag
	}
}

// wedAddMark places a landmark (a places.go waypoint, F10 list) at the current world spot.
func (g *Game) wedAddMark(name, note string) string {
	name, note = wedClean(name), wedClean(note)
	if name == "" {
		return "A landmark needs a name."
	}
	pl := g.wpl()
	if len(pl.List) >= wplMax {
		return "Your places list is full."
	}
	st := g.wed()
	m := wedMark{N: name, T: note, X: g.WX, Y: g.WY, P: g.PlanetIndex()}
	st.Marks = append(st.Marks, m)
	pl.List = append(pl.List, wpPlace{Name: name, WX: m.X, WY: m.Y, Note: note, Icon: "★", Planet: m.P, Turn: g.Turn})
	pl.touch()
	return "Landmark placed: " + name + "."
}

func (g *Game) wedDelMark(i int) {
	st := g.wed()
	if i < 0 || i >= len(st.Marks) {
		return
	}
	m := st.Marks[i]
	st.Marks = append(st.Marks[:i], st.Marks[i+1:]...)
	pl := g.wpl()
	for j, p := range pl.List {
		if p.Name == m.N && p.WX == m.X && p.WY == m.Y && p.Planet == m.P {
			pl.List = append(pl.List[:j], pl.List[j+1:]...)
			pl.touch()
			break
		}
	}
}

func (g *Game) wedClear(tab int, id string) {
	st := g.wed()
	switch tab {
	case 0:
		delete(st.Spec, id)
	case 1:
		delete(st.Cult, id)
	case 2:
		delete(st.Town, id)
	case 3:
		delete(st.Home, id)
	case 4:
		if i, err := strconv.Atoi(id); err == nil {
			g.wedDelMark(i)
		}
	case 5:
		delete(st.Houses, id)
	}
	g.wedApply()
}

func (g *Game) wedCommit() {
	st := g.wed()
	if st.sel >= len(st.rows) {
		return
	}
	r := st.rows[st.sel]
	v := st.vals
	st.note = ""
	switch st.tab {
	case 0:
		g.wedSetSpecies(r.ID, v[0], v[1])
	case 1:
		if !g.wedSetCulture(r.ID, v[0]) {
			st.note = "Name cannot be empty."
		}
	case 2:
		if !g.wedSetTown(r.ID, v[0]) {
			st.note = "Name cannot be empty."
		}
	case 3:
		if !g.wedSetHome(r.ID, v[0]) {
			st.note = "Name cannot be empty."
		}
	case 4:
		if r.ID == "new" {
			st.note = g.wedAddMark(v[0], v[1])
		} else if i, err := strconv.Atoi(r.ID); err == nil && i < len(st.Marks) {
			g.wedDelMark(i)
			st.note = g.wedAddMark(v[0], v[1])
		}
	case 5:
		g.wedSetHouse(r.ID, v[0])
	}
	g.wedRebuild()
}

// ---- export

func (g *Game) wedExport() []string {
	st := g.wed()
	var out []string
	emit := func(m map[string]any) {
		if b, err := json.Marshal(m); err == nil {
			out = append(out, string(b))
		}
	}
	var ids []string
	for id := range st.Spec {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		e := st.Spec[id]
		emit(map[string]any{"kind": "species", "id": id, "name": e.N, "desc": e.D})
	}
	sortedNum := func(m map[string]string) []string {
		var k []string
		for id := range m {
			k = append(k, id)
		}
		sort.Slice(k, func(i, j int) bool {
			a, e1 := strconv.Atoi(k[i])
			b, e2 := strconv.Atoi(k[j])
			if e1 == nil && e2 == nil {
				return a < b
			}
			return k[i] < k[j]
		})
		return k
	}
	for _, id := range sortedNum(st.Cult) {
		emit(map[string]any{"kind": "culture", "id": id, "name": st.Cult[id]})
	}
	for _, id := range sortedNum(st.Town) {
		emit(map[string]any{"kind": "town", "id": id, "name": st.Town[id]})
	}
	for _, id := range sortedNum(st.Home) {
		emit(map[string]any{"kind": "home", "id": id, "name": st.Home[id]})
	}
	for _, m := range st.Marks {
		emit(map[string]any{"kind": "landmark", "name": m.N, "note": m.T, "x": m.X, "y": m.Y, "planet": m.P})
	}
	for _, id := range sortedNum(st.Houses) {
		emit(map[string]any{"kind": "house", "id": id, "tag": st.Houses[id]})
	}
	return out
}

func (g *Game) wedOpenExport() {
	st := g.wed()
	lines := g.wedExport()
	if len(lines) == 0 {
		lines = []string{"(no edits yet)"}
	}
	st.text = nil
	for _, l := range lines {
		st.text = append(st.text, Wrap(l, SW-8)...)
	}
	st.top = 0
	g.Mode = "wedx"
}

func wedTextKey(g *Game, k string) {
	st := g.wed()
	rows := SH - 8
	switch k {
	case "Escape", "Enter", "q":
		g.Mode = "wed"
	case "ArrowUp", "N8":
		st.top = maxi(0, st.top-1)
	case "ArrowDown", "N2":
		st.top = maxi(0, mini(st.top+1, len(st.text)-rows))
	case "PageUp":
		st.top = maxi(0, st.top-rows)
	case "PageDown":
		st.top = maxi(0, mini(st.top+rows, len(st.text)-rows))
	}
}

func wedTextDraw(g *Game) {
	s, st := g.sc, g.wed()
	s.Box(2, 1, SW-4, SH-3, CDarkCyan, "Export my edits (copy these JSON lines)")
	rows := SH - 8
	for i := 0; i < rows && st.top+i < len(st.text); i++ {
		s.TextB(4, 3+i, st.text[st.top+i], CGray, CBgDark, SW-4)
	}
	s.TextB(4, SH-4, "{{K|Up/Down/PageUp/PageDown scroll, Esc back}}", CGray, CBgDark, SW-4)
}

// ---- screen

func wedKey(g *Game, k string) {
	st := g.wed()
	if st.prompt {
		wedPromptKey(g, k)
		return
	}
	switch k {
	case "Escape":
		g.Mode = "main"
	case "ArrowLeft", "N4":
		st.tab = (st.tab + len(wedTabs) - 1) % len(wedTabs)
		st.sel, st.note = 0, ""
		g.wedRebuild()
	case "ArrowRight", "N6", "Tab":
		st.tab = (st.tab + 1) % len(wedTabs)
		st.sel, st.note = 0, ""
		g.wedRebuild()
	case "ArrowUp", "N8":
		st.sel = maxi(0, st.sel-1)
	case "ArrowDown", "N2":
		st.sel = maxi(0, mini(st.sel+1, len(st.rows)-1))
	case "PageUp":
		st.sel = maxi(0, st.sel-10)
	case "PageDown":
		st.sel = maxi(0, mini(st.sel+10, len(st.rows)-1))
	case "x", "X":
		g.wedOpenExport()
	case "d", "Delete", "r":
		if st.sel < len(st.rows) && st.rows[st.sel].ID != "new" {
			g.wedClear(st.tab, st.rows[st.sel].ID)
			g.wedRebuild()
			st.note = "Edit cleared."
		}
	case "Enter":
		if st.sel < len(st.rows) {
			r := st.rows[st.sel]
			st.fields = wedFieldNames(st.tab)
			st.vals = append([]string{}, r.Cur...)
			for len(st.vals) < len(st.fields) {
				st.vals = append(st.vals, "")
			}
			st.fi, st.prompt, st.note = 0, true, ""
		}
	}
}

func wedPromptKey(g *Game, k string) {
	st := g.wed()
	switch k {
	case "Escape":
		st.prompt = false
	case "Enter":
		if st.fi+1 < len(st.fields) {
			st.fi++
			return
		}
		st.prompt = false
		g.wedCommit()
	case "Backspace":
		if rs := []rune(st.vals[st.fi]); len(rs) > 0 {
			st.vals[st.fi] = string(rs[:len(rs)-1])
		}
	default:
		if rs := []rune(k); len(rs) == 1 && rs[0] >= 32 && rs[0] != '{' && rs[0] != '}' && rs[0] != '|' && len(st.vals[st.fi]) < 300 {
			st.vals[st.fi] += k
		}
	}
}

func wedDraw(g *Game) {
	s, st := g.sc, g.wed()
	s.Box(2, 1, SW-4, SH-3, CDarkCyan, "World editor-lite")
	x := 4
	for i, t := range wedTabs {
		if i == st.tab {
			x += s.TextB(x, 2, "{{W|["+t+"]}} ", CGray, CBgDark, SW-4)
		} else {
			x += s.TextB(x, 2, "{{K|"+t+"}} ", CGray, CBgDark, SW-4)
		}
	}
	rows := SH - 12
	top := maxi(0, st.sel-rows+1)
	if len(st.rows) == 0 {
		msg := "Nothing here."
		if st.tab == 3 {
			msg = "You have no homes yet (claim one with H). Rename your home town on the Towns tab."
		}
		s.TextB(4, 4, "{{K|"+msg+"}}", CGray, CBgDark, SW-4)
	}
	for i := 0; i < rows && top+i < len(st.rows); i++ {
		bg := CBgDark
		if top+i == st.sel {
			bg = CBgSel
		}
		s.Fill(4, 4+i, SW-8, 1, bg)
		s.TextB(5, 4+i, st.rows[top+i].Label, CGray, bg, SW-5)
	}
	y := SH - 7
	if st.prompt {
		for i, f := range st.fields {
			cur := ""
			if i == st.fi {
				cur = "{{Y|_}}"
			}
			s.TextB(4, y+i, "{{W|"+f+":}} "+st.vals[i]+cur, CGray, CBgDark, SW-4)
		}
		s.TextB(4, SH-4, "{{K|Enter next/accept, Esc cancel}}", CGray, CBgDark, SW-4)
		return
	}
	if st.note != "" {
		s.TextB(4, y, "{{Y|"+st.note+"}}", CGray, CBgDark, SW-4)
	}
	if st.tab == 0 && st.sel < len(st.rows) {
		if sp := g.W.specByID[st.rows[st.sel].ID]; sp != nil {
			for i, l := range Wrap(sp.Desc, SW-10) {
				if i < 2 {
					s.TextB(4, y+1+i, "{{K|"+l+"}}", CGray, CBgDark, SW-4)
				}
			}
		}
	}
	if st.tab == 4 && st.sel > 0 && st.sel-1 < len(st.Marks) {
		s.TextB(4, y+1, "{{K|"+st.Marks[st.sel-1].T+"}}", CGray, CBgDark, SW-4)
	}
	s.TextB(4, SH-4, "{{K|Left/Right tab, Up/Down pick, Enter edit, d clear edit, x export, Esc close}}", CGray, CBgDark, SW-4)
}

// ---- console

func wedCmd(g *Game, a []string) string {
	if g.W == nil {
		return "no world"
	}
	if len(a) == 0 || a[0] == "open" {
		g.cn().out = append(g.cn().out, "{{K|Opening the editor (Esc returns to the game).}}")
		g.wedOpen()
		return ""
	}
	st := g.wed()
	rest := func(from int) string {
		if len(a) <= from {
			return ""
		}
		return strings.Join(a[from:], " ")
	}
	switch strings.ToLower(a[0]) {
	case "help":
		return "wed species <id> <name> [| <desc>] ; wed culture <id> <name> ; wed town <burg id> <name> ; wed home <n> <name> ; wed mark <name> [| <note>] ; wed house <house> <tag> ; wed clear <species|culture|town|home|house> <id> ; wed list <species|culture|town|home|mark|house> ; wed export ; wed (opens the screen)"
	case "export":
		lines := g.wedExport()
		if len(lines) == 0 {
			return "no edits yet"
		}
		return strings.Join(lines, "\n")
	case "species":
		if len(a) < 3 {
			return "usage: wed species <id> <name> [| <description>]"
		}
		name, desc, _ := strings.Cut(rest(2), "|")
		if !g.wedSetSpecies(a[1], name, desc) {
			return "unknown species id (wed list species)"
		}
		return "Species updated."
	case "culture":
		if len(a) < 3 || !g.wedSetCulture(a[1], rest(2)) {
			return "usage: wed culture <id> <name> (wed list culture)"
		}
		return "Culture renamed."
	case "town":
		if len(a) < 3 || !g.wedSetTown(a[1], rest(2)) {
			return "usage: wed town <burg id> <name> (wed list town)"
		}
		return "Town renamed."
	case "home":
		hs := g.homeSt().Homes
		n, err := strconv.Atoi(wedStrOr(a, 1))
		if len(a) < 3 || err != nil || n < 1 || n > len(hs) {
			return "usage: wed home <1.." + strconv.Itoa(len(hs)) + "> <name>"
		}
		g.wedSetHome(hs[n-1].Key, rest(2))
		return "Home renamed."
	case "mark", "landmark":
		name, note, _ := strings.Cut(rest(1), "|")
		return g.wedAddMark(name, note)
	case "house":
		if len(a) < 3 {
			return "usage: wed house <house name> <tag> (a house name with spaces: use underscores)"
		}
		g.wedSetHouse(strings.ReplaceAll(a[1], "_", " "), rest(2))
		return "House tagged."
	case "clear":
		if len(a) < 3 {
			return "usage: wed clear <species|culture|town|home|house> <id>"
		}
		tab := map[string]int{"species": 0, "culture": 1, "town": 2, "home": 3, "house": 5}
		t, ok := tab[strings.ToLower(a[1])]
		if !ok {
			return "unknown kind"
		}
		id := strings.ReplaceAll(a[2], "_", " ")
		g.wedClear(t, id)
		return "Edit cleared."
	case "list":
		kind := strings.ToLower(wedStrOr(a, 1))
		var out []string
		switch kind {
		case "species":
			for _, s := range g.W.Species {
				out = append(out, s.ID+"="+s.Name)
			}
		case "culture":
			for _, c := range g.W.Cultures {
				out = append(out, strconv.Itoa(c.ID)+"="+c.Name)
			}
		case "town":
			for i, b := range g.wedBurgs() {
				if i >= 40 {
					break
				}
				out = append(out, strconv.Itoa(b.ID)+"="+b.Name)
			}
		case "home":
			for i, h := range g.homeSt().Homes {
				out = append(out, strconv.Itoa(i+1)+"="+h.Name)
			}
		case "mark":
			for _, m := range st.Marks {
				out = append(out, m.N)
			}
		case "house":
			out = g.wedHouseList()
		default:
			return "usage: wed list <species|culture|town|home|mark|house>"
		}
		if len(out) == 0 {
			return "(none)"
		}
		return strings.Join(out, ", ")
	}
	return "unknown subcommand (wed help)"
}

func wedStrOr(a []string, i int) string {
	if i < len(a) {
		return a[i]
	}
	return ""
}
