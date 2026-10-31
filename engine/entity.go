package engine

import "fmt"

const (
	STR = iota
	AGI
	TOU
	INT
	WIL
	EGO
)

var AttrNames = [6]string{"Strength", "Agility", "Toughness", "Intelligence", "Willpower", "Ego"}
var AttrShort = [6]string{"STR", "AGI", "TOU", "INT", "WIL", "EGO"}

func Mod(score int) int {
	if score >= 16 {
		return (score - 16) / 2
	}
	return -((17 - score) / 2)
}

// BodyPart is a limb with its own equipment slot; severed limbs drop what they held (Qud-style dismemberment).
type BodyPart struct {
	Name    string `json:"n"`
	Slot    string `json:"s"` // hand, head, face, body, back, feet, hands, tail, wings, float
	Item    *Item  `json:"i,omitempty"`
	Natural *Item  `json:"nat,omitempty"`
	Severed bool   `json:"sev,omitempty"`
	Limb    bool   `json:"l,omitempty"` // can be severed
}

type Item struct {
	ID      int            `json:"id"`
	Name    string         `json:"n"`
	Glyph   rune           `json:"g"`
	F       byte           `json:"f"`
	Kind    string         `json:"k"` // weapon armor ranged ammo light food water tonic grenade book artifact scrap letter trophy
	Slot    string         `json:"sl,omitempty"`
	Skill   string         `json:"sk,omitempty"`
	DN      int            `json:"dn,omitempty"`
	DS      int            `json:"ds,omitempty"`
	DB      int            `json:"db,omitempty"`
	PV      int            `json:"pv,omitempty"`
	MaxStr  int            `json:"ms,omitempty"`
	AV      int            `json:"av,omitempty"`
	DV      int            `json:"dv,omitempty"`
	Range   int            `json:"rg,omitempty"`
	Light   int            `json:"lt,omitempty"`
	Effect  string         `json:"ef,omitempty"`
	Power   int            `json:"pw,omitempty"`
	Count   int            `json:"c,omitempty"`
	Value   int            `json:"v,omitempty"`
	Weight  int            `json:"w,omitempty"`
	Tier    int            `json:"t,omitempty"`
	Ident   bool           `json:"id2,omitempty"`
	Unid    string         `json:"u,omitempty"`
	Lore    string         `json:"lo,omitempty"`
	Bonus   map[int]int    `json:"bo,omitempty"` // attribute → bonus
	SimArt  int            `json:"sa,omitempty"`
	X       int            `json:"x"`
	Y       int            `json:"y"`
	Text    string         `json:"tx,omitempty"`
	Quest   int            `json:"q,omitempty"`
	Two     bool           `json:"two,omitempty"`
	Charges int            `json:"ch,omitempty"`
	Elem    string         `json:"el,omitempty"` // fire frost shock acid poison
	Extra   map[string]int `json:"ex,omitempty"`
	Rarity  string         `json:"ra,omitempty"`   // common uncommon rare ultrarare legendary mythic unique (rarity.go)
	Cat     string         `json:"cat,omitempty"`  // special category: biblical occult omni ... (rarity.go)
	Look    string         `json:"look,omitempty"` // transmog: name of the appearance shown instead (attire.go)
	Region  string         `json:"reg,omitempty"`  // culture sphere of regional attire (attire.go)
}

func (it *Item) Display() string {
	if it.Kind == "artifact" && !it.Ident {
		return "{{M|" + it.Unid + "}}"
	}
	n := it.ShownName()
	if it.Count > 1 {
		n = fmt.Sprintf("%s x%d", n, it.Count)
	}
	return n
}

func (it *Item) Stack() bool {
	switch it.Kind {
	case "ammo", "food", "tonic", "grenade", "scrap":
		return true
	}
	return false
}

type Actor struct {
	ID       int             `json:"id"`
	Name     string          `json:"n"`
	Proper   bool            `json:"pr,omitempty"` // named person, no article
	Glyph    rune            `json:"g"`
	F        byte            `json:"f"`
	X        int             `json:"x"`
	Y        int             `json:"y"`
	Tpl      string          `json:"tp"`
	Faction  string          `json:"fa"`
	Level    int             `json:"lv"`
	Attr     [6]int          `json:"at"`
	HP       int             `json:"hp"`
	MaxHP    int             `json:"mhp"`
	AVNat    int             `json:"av"`
	DVNat    int             `json:"dv"`
	Speed    int             `json:"sp"`
	Energy   int             `json:"en"`
	Body     []*BodyPart     `json:"bd"`
	Inv      []*Item         `json:"inv,omitempty"`
	Status   map[string]int  `json:"st,omitempty"`
	AI       string          `json:"ai"` // hostile wander town follower guard still
	Role     string          `json:"ro,omitempty"`
	Species  string          `json:"spc,omitempty"`
	Gold     int             `json:"gold,omitempty"`
	Boss     bool            `json:"boss,omitempty"`
	SimMon   int             `json:"sm,omitempty"`
	SimPer   int             `json:"spe,omitempty"`
	Desc     string          `json:"ds,omitempty"`
	Powers   map[string]int  `json:"pow,omitempty"`
	Cool     map[string]int  `json:"cd,omitempty"`
	Flags    map[string]bool `json:"fl,omitempty"`
	Kinship  bool            `json:"kin,omitempty"`
	Secret   int             `json:"sec,omitempty"`
	Quest    int             `json:"qu,omitempty"`
	HomeX    int             `json:"hx"`
	HomeY    int             `json:"hy"`
	Angry    bool            `json:"ang,omitempty"` // provoked personally
	Leader   int             `json:"ld,omitempty"`
	Title    string          `json:"ti,omitempty"`
	XP       int             `json:"xpv,omitempty"`
	Religion int             `json:"rel,omitempty"`
	State    int             `json:"sta,omitempty"`
	Talked   bool            `json:"tk,omitempty"`
	Teach    string          `json:"teach,omitempty"`
	Caste    string          `json:"caste,omitempty"` // social stratum (society.go)
	Rank     int             `json:"rank,omitempty"`  // standing inside the stratum, 0 lowest (society.go)
	House    string          `json:"house,omitempty"` // family name shared by a household (society.go)
	Kin      []int           `json:"kins,omitempty"`  // actor ids of relatives in the same zone (society.go)
	Order    string          `json:"ord,omitempty"`   // companion order (companions.go)
	Comp     *Companion      `json:"comp,omitempty"`  // companion record (companions.go)
}

func (a *Actor) Has(f string) bool { return a.Flags != nil && a.Flags[f] }
func (a *Actor) Set(f string) {
	if a.Flags == nil {
		a.Flags = map[string]bool{}
	}
	a.Flags[f] = true
}

func (a *Actor) St(s string) int {
	if a.Status == nil {
		return 0
	}
	return a.Status[s]
}

func (a *Actor) AddSt(s string, n int) {
	if a.Status == nil {
		a.Status = map[string]int{}
	}
	if a.Status[s] < n {
		a.Status[s] = n
	}
}

func (a *Actor) Pow(id string) int {
	if a.Powers == nil {
		return 0
	}
	return a.Powers[id]
}

func (a *Actor) Score(i int) int {
	v := a.Attr[i]
	for _, p := range a.Body {
		if p.Item != nil && p.Item.Bonus != nil && !p.Severed {
			v += p.Item.Bonus[i]
		}
	}
	if i == STR && a.St("mighty") > 0 {
		v += 6
	}
	return v
}

func (a *Actor) M(i int) int { return Mod(a.Score(i)) }

func (a *Actor) AV() int {
	av := a.AVNat
	for _, p := range a.Body {
		if p.Item != nil && !p.Severed {
			av += p.Item.AV
		}
	}
	if l := a.Pow("carapace"); l > 0 {
		av += 2 + l/2
	}
	if a.St("frozen") > 0 {
		av += 2
	}
	av += shieldAV(a)
	return av
}

func (a *Actor) DV() int {
	dv := 6 + a.DVNat + a.M(AGI)
	for _, p := range a.Body {
		if p.Item != nil && !p.Severed {
			dv += p.Item.DV
		}
	}
	if a.St("stunned") > 0 || a.St("frozen") > 0 || a.St("prone") > 0 {
		dv -= 6
	}
	return dv
}

func (a *Actor) MA() int { return 4 + a.M(WIL) }

func (a *Actor) Quick() int {
	q := a.Speed
	if a.St("hasted") > 0 {
		q += 50
	}
	if a.St("sprint") > 0 {
		q += 100
	}
	if a.St("slowed") > 0 || a.St("frozen") > 0 {
		q -= 40
	}
	return maxi(q, 20)
}

func (a *Actor) LightRadius() int {
	r := 0
	for _, p := range a.Body {
		if p.Item != nil && p.Item.Light > r && !p.Severed {
			r = p.Item.Light
		}
	}
	for _, it := range a.Inv {
		if it.Kind == "light" && it.Light > r && it.Extra != nil && it.Extra["lit"] > 0 {
			r = it.Light
		}
	}
	return r
}

// Weapons returns the wielded weapons (or natural attacks) for this turn's melee swing.
func (a *Actor) Weapons() []*Item {
	var out []*Item
	for _, p := range a.Body {
		if p.Severed {
			continue
		}
		if p.Slot == "hand" {
			if p.Item != nil && (p.Item.Kind == "weapon" || p.Item.Kind == "artifact" && p.Item.DN > 0) {
				out = append(out, p.Item)
			} else if p.Item == nil && p.Natural != nil {
				out = append(out, p.Natural)
			}
		} else if p.Natural != nil && p.Item == nil {
			out = append(out, p.Natural)
		}
	}
	if len(out) == 0 {
		out = append(out, fist())
	}
	return out
}

func (a *Actor) Missile() *Item {
	for _, p := range a.Body {
		if !p.Severed && p.Item != nil && p.Item.Kind == "ranged" {
			return p.Item
		}
	}
	return nil
}

func (a *Actor) PartFor(slot string) *BodyPart {
	for _, p := range a.Body {
		if p.Slot == slot && !p.Severed && p.Item == nil {
			return p
		}
	}
	for _, p := range a.Body {
		if p.Slot == slot && !p.Severed {
			return p
		}
	}
	return nil
}

func (a *Actor) Equipped(it *Item) *BodyPart {
	for _, p := range a.Body {
		if p.Item == it {
			return p
		}
	}
	return nil
}

func fist() *Item {
	return &Item{Name: "fist", Kind: "weapon", Skill: "cudgel", DN: 1, DS: 2, PV: 0, MaxStr: 2}
}

func natural(name string, dn, ds, pv int, elem string) *Item {
	return &Item{Name: name, Kind: "weapon", Skill: "natural", DN: dn, DS: ds, PV: pv, MaxStr: 4, Elem: elem}
}

// MakeBody builds a body plan; Qud bodies are trees, ours are flat lists with limb flags.
func MakeBody(plan string) []*BodyPart {
	hum := func() []*BodyPart {
		return []*BodyPart{
			{Name: "Head", Slot: "head", Limb: true}, {Name: "Face", Slot: "face"}, {Name: "Body", Slot: "body"}, {Name: "Back", Slot: "back"},
			{Name: "Right Hand", Slot: "hand", Limb: true}, {Name: "Left Hand", Slot: "hand", Limb: true}, {Name: "Hands", Slot: "hands"},
			{Name: "Feet", Slot: "feet", Limb: true}, {Name: "Floating Nearby", Slot: "float"},
		}
	}
	switch plan {
	case "merfolk":
		b := hum()
		b[7] = &BodyPart{Name: "Tail", Slot: "tail", Limb: true}
		return b
	case "winged":
		b := hum()
		return append(b, &BodyPart{Name: "Wings", Slot: "wings", Limb: true})
	case "fourarm":
		b := hum()
		return append(b, &BodyPart{Name: "Lower Right Hand", Slot: "hand", Limb: true}, &BodyPart{Name: "Lower Left Hand", Slot: "hand", Limb: true})
	case "quadruped":
		return []*BodyPart{{Name: "Head", Slot: "head", Limb: true, Natural: natural("bite", 1, 4, 1, "")}, {Name: "Body", Slot: "body"},
			{Name: "Fore Legs", Slot: "feet", Limb: true}, {Name: "Hind Legs", Slot: "feet", Limb: true}, {Name: "Tail", Slot: "tail", Limb: true}}
	case "insect":
		return []*BodyPart{{Name: "Head", Slot: "head", Limb: true, Natural: natural("mandibles", 1, 4, 1, "")}, {Name: "Thorax", Slot: "body"},
			{Name: "Legs", Slot: "feet", Limb: true}, {Name: "Abdomen", Slot: "tail", Limb: true}}
	case "serpent":
		return []*BodyPart{{Name: "Head", Slot: "head", Limb: true, Natural: natural("fangs", 1, 3, 1, "poison")}, {Name: "Body", Slot: "body"}, {Name: "Tail", Slot: "tail", Limb: true}}
	case "avian":
		return []*BodyPart{{Name: "Head", Slot: "head", Limb: true, Natural: natural("beak", 1, 4, 1, "")}, {Name: "Body", Slot: "body"},
			{Name: "Wings", Slot: "wings", Limb: true}, {Name: "Talons", Slot: "feet", Limb: true, Natural: natural("talons", 1, 3, 1, "")}}
	case "fish":
		return []*BodyPart{{Name: "Head", Slot: "head", Limb: true, Natural: natural("jaws", 1, 6, 1, "")}, {Name: "Body", Slot: "body"}, {Name: "Fins", Slot: "tail", Limb: true}}
	case "ooze":
		return []*BodyPart{{Name: "Mass", Slot: "body", Natural: natural("pseudopod", 1, 4, 0, "acid")}}
	case "golem":
		return []*BodyPart{{Name: "Head", Slot: "head"}, {Name: "Body", Slot: "body"}, {Name: "Right Arm", Slot: "hand", Limb: true, Natural: natural("stone fist", 1, 8, 2, "")},
			{Name: "Left Arm", Slot: "hand", Limb: true, Natural: natural("stone fist", 1, 8, 2, "")}, {Name: "Legs", Slot: "feet", Limb: true}}
	case "tentacled":
		return []*BodyPart{{Name: "Maw", Slot: "head", Natural: natural("maw", 2, 6, 2, "")}, {Name: "Body", Slot: "body"},
			{Name: "Tentacle", Slot: "hand", Limb: true, Natural: natural("tentacle", 1, 6, 1, "")}, {Name: "Tentacle", Slot: "hand", Limb: true, Natural: natural("tentacle", 1, 6, 1, "")},
			{Name: "Tentacle", Slot: "tail", Limb: true, Natural: natural("tentacle", 1, 6, 1, "")}}
	}
	return hum()
}

// BodyPlanForSpecies maps Realmforge species ids onto body plans.
func BodyPlanForSpecies(id string) string {
	switch id {
	case "merfolk":
		return "merfolk"
	case "celestial":
		return "winged"
	case "chitin_swarm", "warp_fiend":
		return "fourarm"
	}
	return "humanoid"
}
