package engine

import (
	"fmt"
	"sort"
	"strings"
)

// Body augments: an original take on Qud's cybernetics. One mechanical augment is skinned per genre
// (runic tattoos, blessed grafts, clockwork prosthetics, servo rigs, isotope implants, chrome
// cyberware, grown symbiotes, photosynthetic lattices). An augment is installed into a body part
// (BodyPart.Aug, at most one per part) by an installer NPC (role "augmenter") and costs augment
// capacity:
//
//	capacity = 3 + Toughness modifier (base score) + level/5, never below 1
//
// Effects are read live from the installed parts (Score, AV, DV, Quick, LightRadius, Pow, Weapons,
// hurt), so removing an augment restores the body at once and a severed limb silences whatever is
// fitted in it until the limb is regrown or healed. Uninstalled augments are items of Kind "augment"
// whose Effect is the AugDef id.

type AugDef struct {
	ID     string
	Slot   string // install slot: head face body back hand feet (feet also fits a tail)
	Cost   int    // augment capacity
	Rare   bool   // loot only; installers do not stock it
	Bonus  map[int]int
	AV     int
	DV     int
	Quick  int
	Light  int
	Power  string // grants this power (passive or active) at level 1 via Actor.Pow
	Resist bool   // elemental damage is cut by a third
	Weapon [3]int // natural weapon dice/sides/PV used by an empty hand
}

var AugDefs = []AugDef{
	{ID: "nighteye", Slot: "face", Cost: 1, Power: "darkvision"},
	{ID: "hawkeye", Slot: "face", Cost: 2, Bonus: map[int]int{AGI: 2}},
	{ID: "lumen", Slot: "head", Cost: 1, Light: 4},
	{ID: "cortex", Slot: "head", Cost: 2, Bonus: map[int]int{INT: 3}},
	{ID: "reflex", Slot: "head", Cost: 3, Rare: true, Quick: 15},
	{ID: "mindward", Slot: "head", Cost: 2, Bonus: map[int]int{WIL: 3}},
	{ID: "dermal", Slot: "body", Cost: 2, AV: 2},
	{ID: "heart", Slot: "body", Cost: 3, Power: "regen"},
	{ID: "lung", Slot: "body", Cost: 1, Power: "diving"},
	{ID: "ward", Slot: "body", Cost: 2, Resist: true},
	{ID: "medic", Slot: "body", Cost: 2, Power: "heal"},
	{ID: "blade", Slot: "hand", Cost: 2, Weapon: [3]int{1, 6, 3}},
	{ID: "servo", Slot: "hand", Cost: 2, Bonus: map[int]int{STR: 2}},
	{ID: "grip", Slot: "hand", Cost: 1, Power: "climbing"},
	{ID: "flamer", Slot: "hand", Cost: 3, Rare: true, Power: "flames"},
	{ID: "stride", Slot: "feet", Cost: 1, DV: 2},
	{ID: "sprinter", Slot: "feet", Cost: 2, Quick: 5, Power: "sprint"},
	{ID: "cloak", Slot: "back", Cost: 3, Rare: true, Power: "phase"},
	{ID: "bulwark", Slot: "back", Cost: 2, Power: "forcebubble"},
	{ID: "wings", Slot: "back", Cost: 4, Rare: true, Power: "wings"},
}

var augByID = func() map[string]*AugDef {
	m := map[string]*AugDef{}
	for i := range AugDefs {
		m[AugDefs[i].ID] = &AugDefs[i]
	}
	return m
}()

// AugSkin is how a genre presents augments: the kind of thing, who fits it, flavour and names.
type AugSkin struct {
	Noun      string // what an augment is called
	Installer string // installer NPC title
	Lore      string // flavour line on every augment
	Install   string // message when one is fitted ("%s" = augment name)
	Purist    string // what a purist says to a heavily augmented wanderer
	Names     map[string]string
}

var augSkins = map[string]AugSkin{
	"fantasy": {Noun: "rune-graft", Installer: "runewright", Lore: "Silver ink and a hedge-witch's blessing, worked into the flesh.",
		Install: "The runewright chants as the %s is inked and bound into you.", Purist: "\"So many marks on one body. The old gods frown on that much meddling.\"",
		Names: map[string]string{"nighteye": "Owl-Rune of Night Sight", "hawkeye": "Hawk-Eye Sigil", "lumen": "Wisp-Crown Rune", "cortex": "Sage's Brow-Rune",
			"reflex": "Quicksilver Nerve-Graft", "mindward": "Ward-Knot of the Will", "dermal": "Ironbark Skin-Rune", "heart": "Troll-Heart Graft",
			"lung": "Selkie-Breath Rune", "ward": "Elemental Ward-Tattoo", "medic": "Mending Rune", "blade": "Wolf-Claw Graft", "servo": "Ox-Strength Arm-Rune",
			"grip": "Spider-Grip Glyph", "flamer": "Salamander Palm-Brand", "stride": "Hare-Foot Rune", "sprinter": "Stag-Leg Graft",
			"cloak": "Shadow-Mantle Rune", "bulwark": "Aegis Back-Sigil", "wings": "Griffon-Wing Graft"}},
	"mythic": {Noun: "blessed graft", Installer: "graft-hierophant", Lore: "A gift of the gods, sewn in with golden thread and a libation.",
		Install: "The hierophant pours a libation and sews the %s into you.", Purist: "\"Hubris. The gods punish mortals who wear too many of their gifts.\"",
		Names: map[string]string{"nighteye": "Eye of Nyx", "hawkeye": "Eye of Artemis", "lumen": "Torch of Hecate", "cortex": "Brow of Athena",
			"reflex": "Nerve of Achilles", "mindward": "Seal of Apollo", "dermal": "Hide of the Nemean Lion", "heart": "Liver of Prometheus",
			"lung": "Gills of Triton", "ward": "Styx-Dipped Skin", "medic": "Touch of Asclepius", "blade": "Talon of the Harpy", "servo": "Arm of Heracles",
			"grip": "Hand of Arachne", "flamer": "Palm of Hephaestus", "stride": "Sandal-Mark of Atalanta", "sprinter": "Winged Heels of Hermes",
			"cloak": "Helm-Shade of Hades", "bulwark": "Shield-Mark of Athena", "wings": "Wings of Daedalus"}},
	"steampunk": {Noun: "clockwork prosthetic", Installer: "clockwork surgeon", Lore: "Brass, springs and gutta-percha, wound tight and oiled weekly.",
		Install: "Under a whiff of ether the surgeon fits the %s with tiny screwdrivers.", Purist: "\"More brass than blood, are we? Unnatural, sir. Quite unnatural.\"",
		Names: map[string]string{"nighteye": "Brass Night-Lens", "hawkeye": "Telescopic Monocle", "lumen": "Limelight Coronet", "cortex": "Difference-Engine Cortex",
			"reflex": "Escapement Nerve-Regulator", "mindward": "Phrenological Governor", "dermal": "Riveted Dermal Plates", "heart": "Galvanic Heart Pump",
			"lung": "Bellows-Lung", "ward": "Asbestos-Lined Torso", "medic": "Clockwork Suture Engine", "blade": "Spring-Loaded Sword-Arm", "servo": "Piston Forearm",
			"grip": "Grapnel Fingers", "flamer": "Naphtha Gauntlet-Jet", "stride": "Sprung Ankle-Joints", "sprinter": "Steam-Piston Legs",
			"cloak": "Aetheric Shroud Harness", "bulwark": "Pressure-Shield Boiler", "wings": "Ornithopter Back-Frame"}},
	"dieselpunk": {Noun: "servo rig", Installer: "rig mechanic", Lore: "Stamped steel and hydraulic line, bolted on in a field hospital.",
		Install: "The mechanic bites a cigarette and bolts the %s into place.", Purist: "\"Half machine. The front made a lot of you. Don't expect me to like it.\"",
		Names: map[string]string{"nighteye": "IR Eye Rig", "hawkeye": "Sniper Optic Rig", "lumen": "Helmet Arc-Lamp Rig", "cortex": "Cryptograph Skull Relay",
			"reflex": "Pervitin Nerve Pump", "mindward": "Propaganda-Proof Skull Plate", "dermal": "Dermal Steel Weave", "heart": "Diesel Heart Booster",
			"lung": "Frogman Lung Tank", "ward": "Flame-Retardant Chest Rig", "medic": "Morphine Drip Rig", "blade": "Trench-Spike Arm", "servo": "Hydraulic Servo Arm",
			"grip": "Piton Claw Hand", "flamer": "Flamethrower Arm Rig", "stride": "Shock-Absorber Shins", "sprinter": "Servo Leg Rig",
			"cloak": "Smoke-Screen Pack", "bulwark": "Armor-Plate Back Rig", "wings": "Rocket Back-Pack Rig"}},
	"atompunk": {Noun: "isotope implant", Installer: "atomic surgeon", Lore: "Glowing gently, guaranteed safe by the Atomic Commission.",
		Install: "The surgeon gives a thumbs-up and slots the %s in with a satisfying click.", Purist: "\"All that glowing junk in you. Folks around here like their insides natural.\"",
		Names: map[string]string{"nighteye": "Glow-Pupil Implant", "hawkeye": "Atomic Scope Eye", "lumen": "Radium Halo Implant", "cortex": "Positronic Brain Booster",
			"reflex": "Isotope Reflex Booster", "mindward": "Lead-Lined Thought Shield", "dermal": "Rad-Plate Skin Implant", "heart": "Isotope Heart Cell",
			"lung": "Big-Lung Bottle Implant", "ward": "Hazmat Organ Liner", "medic": "Stimpak Reservoir Implant", "blade": "Ripper Arm Implant", "servo": "Atomic Servo Bicep",
			"grip": "Magno-Grip Palm", "flamer": "Plasma Palm Implant", "stride": "Sprung Hover-Heels", "sprinter": "Jet-Leg Implant",
			"cloak": "Stealth-Field Spine Implant", "bulwark": "Force-Field Belt Implant", "wings": "Jetpack Spine Mount"}},
	"cyberpunk": {Noun: "cyberware", Installer: "cyberdoc", Lore: "Chrome and fiber, black-market firmware, a serial number filed off.",
		Install: "The cyberdoc jacks you under and wires the %s into your nervous system.", Purist: "\"How much of you is still meat, choom? Careful. Chrome eats the soul.\"",
		Names: map[string]string{"nighteye": "Low-Light Cybereyes", "hawkeye": "Smartlink Optics", "lumen": "Neon Halo Implant", "cortex": "Neural Coprocessor",
			"reflex": "Wired Reflexes", "mindward": "Neural Firewall", "dermal": "Subdermal Armor", "heart": "Nanite Heart",
			"lung": "Oxygen Reservoir", "ward": "Thermal Dampening Mesh", "medic": "Biomonitor Auto-Injector", "blade": "Mantis Blades", "servo": "Gorilla Arm",
			"grip": "Gecko Palms", "flamer": "Plasma Projector Arm", "stride": "Reinforced Tendons", "sprinter": "Sprint Actuators",
			"cloak": "Optical Camo Spine", "bulwark": "Hardlight Shell Emitter", "wings": "Grav-Lift Spine"}},
	"biopunk": {Noun: "grown symbiote", Installer: "flesh-grafter", Lore: "Vat-grown, warm to the touch, it hums softly in time with your pulse.",
		Install: "The flesh-grafter coaxes the %s into you; it roots, squirms and settles.", Purist: "\"You are more colony than person now. The pure-strain among us will not touch you.\"",
		Names: map[string]string{"nighteye": "Tapetum Eye-Splice", "hawkeye": "Raptor Lens Symbiote", "lumen": "Biolume Scalp Colony", "cortex": "Neural Coral Symbiote",
			"reflex": "Hummingbird Nerve Graft", "mindward": "Fungal Mind-Mantle", "dermal": "Chitin Plate Symbiote", "heart": "Axolotl Heart-Graft",
			"lung": "Whale Lung Symbiote", "ward": "Tardigrade Skin Weave", "medic": "Clot-Worm Symbiote", "blade": "Mantis Scythe Graft", "servo": "Gorilla Muscle Graft",
			"grip": "Gecko Pad Symbiote", "flamer": "Bombardier Gland", "stride": "Cricket Tendon Splice", "sprinter": "Cheetah Muscle Splice",
			"cloak": "Cuttlefish Skin Symbiote", "bulwark": "Pillbug Shell Bloom", "wings": "Membrane Wing Bud"}},
	"solarpunk": {Noun: "photosynthetic lattice", Installer: "lattice-gardener", Lore: "A living lattice of leaf-cells and graphene, fed by sunlight and kept by a co-op.",
		Install: "The lattice-gardener lays the %s along your skin; it takes root in the warm light.", Purist: "\"That much grafting? We try to live lightly here. Your body too.\"",
		Names: map[string]string{"nighteye": "Moth-Eye Lattice", "hawkeye": "Prism Sight Lattice", "lumen": "Sun-Store Glow Lattice", "cortex": "Mycelial Memory Lattice",
			"reflex": "Hummingbird Pulse Lattice", "mindward": "Calm-Root Lattice", "dermal": "Bark-Weave Lattice", "heart": "Chlorophyll Heart Lattice",
			"lung": "Algae Lung Lattice", "ward": "Succulent Skin Lattice", "medic": "Aloe Mending Lattice", "blade": "Thorn-Vine Lattice", "servo": "Bamboo-Fiber Arm Lattice",
			"grip": "Tendril Grip Lattice", "flamer": "Solar Lens Palm", "stride": "Spring-Reed Lattice", "sprinter": "Wind-Turbine Leg Lattice",
			"cloak": "Chameleon Leaf Lattice", "bulwark": "Sun-Shield Leaf Array", "wings": "Seed-Kite Wing Lattice"}},
}

// augInstallerDistricts: where each genre's installers keep shop (first district the city has).
var augInstallerDistricts = map[string][]string{
	"fantasy": {"temple", "academy", "artisans", "forge", "market"}, "mythic": {"temple", "academy", "forge", "market"},
	"steampunk": {"forge", "academy", "artisans", "market"}, "dieselpunk": {"forge", "barracks", "market"},
	"atompunk": {"forge", "academy", "market"}, "cyberpunk": {"slums", "forge", "market"},
	"biopunk": {"academy", "farms", "market"}, "solarpunk": {"academy", "farms", "artisans", "market"},
}

func (g *Game) augSkin() AugSkin {
	if s, ok := augSkins[g.Genre]; ok {
		return s
	}
	return augSkins["fantasy"]
}

// AugName is the genre name of an augment.
func (g *Game) AugName(id string) string {
	if n := g.augSkin().Names[id]; n != "" {
		return n
	}
	return titleCase(id)
}

var augSlotNames = map[string]string{"head": "head", "face": "face", "body": "torso", "back": "back", "hand": "arm", "feet": "legs"}

// augFits: can an augment of this slot go into the part?
func augFits(slot string, p *BodyPart) bool {
	return p.Slot == slot || slot == "feet" && p.Slot == "tail"
}

// augOn returns the active augment def of a part, or nil when empty or the limb is severed.
func augOn(p *BodyPart) *AugDef {
	if p.Aug == nil || p.Severed {
		return nil
	}
	return augByID[p.Aug.Effect]
}

// AugCapacity is the actor's augment capacity.
func (a *Actor) AugCapacity() int {
	return maxi(1, 3+Mod(a.Attr[TOU])+a.Level/5)
}

// AugLoad is the capacity used by every installed augment (severed limbs still count).
func (a *Actor) AugLoad() int {
	n := 0
	for _, p := range a.Body {
		if p.Aug != nil {
			if d := augByID[p.Aug.Effect]; d != nil {
				n += d.Cost
			}
		}
	}
	return n
}

// ---- stat hooks (called from entity.go / combat.go) ----

func (a *Actor) augPow(id string) int {
	n := 0
	for _, p := range a.Body {
		if d := augOn(p); d != nil && d.Power == id {
			n = 1
		}
	}
	return n
}

func (a *Actor) augQuick() int {
	q := 0
	for _, p := range a.Body {
		if d := augOn(p); d != nil {
			q += d.Quick
		}
	}
	return q
}

func (a *Actor) augLight() int {
	r := 0
	for _, p := range a.Body {
		if d := augOn(p); d != nil && d.Light > r {
			r = d.Light
		}
	}
	return r
}

var augResistCauses = map[string]bool{"fire": true, "frost": true, "lightning": true, "a shock": true, "acid": true}

// augResist cuts elemental damage by a third for a warded actor.
func augResist(d *Actor, n int, cause string) int {
	if !augResistCauses[cause] {
		return n
	}
	for _, p := range d.Body {
		if ad := augOn(p); ad != nil && ad.Resist {
			return (n*2 + 2) / 3
		}
	}
	return n
}

// ---- items ----

func augValue(d *AugDef) int {
	v := 40 + 60*d.Cost*d.Cost
	if d.Rare {
		v *= 2
	}
	return v
}

// MakeAugment builds an uninstalled augment item.
func (g *Game) MakeAugment(id string) *Item {
	d := augByID[id]
	if d == nil {
		return nil
	}
	it := &Item{ID: g.newID(), Kind: "augment", Effect: id, Name: g.AugName(id), Glyph: '§', F: CCyan, Value: augValue(d), Weight: 1,
		Rarity: "uncommon", Lore: g.augSkin().Lore, AV: d.AV, DV: d.DV, Light: d.Light}
	if d.Rare {
		it.Rarity, it.F = "rare", CBlue
	}
	if len(d.Bonus) > 0 {
		it.Bonus = map[int]int{}
		for k, v := range d.Bonus {
			it.Bonus[k] = v
		}
	}
	if d.Weapon[0] > 0 {
		it.Skill, it.DN, it.DS, it.PV, it.MaxStr = "shortblade", d.Weapon[0], d.Weapon[1], d.Weapon[2], 4
	}
	return it
}

// augLoot rolls a side-rng chance of an augment (any, rare ones favoured) for boss hoards and vaults.
func (g *Game) augLoot(key string, chance float64) *Item {
	r := NewRng(g.Seed + ":aug:" + key)
	if !r.Chance(chance) {
		return nil
	}
	var pool []string
	for _, d := range AugDefs {
		pool = append(pool, d.ID)
		if d.Rare {
			pool = append(pool, d.ID)
		}
	}
	return g.MakeAugment(Pick(r, pool))
}

// augEffects describes what an augment does.
func (g *Game) augEffects(d *AugDef) []string {
	var l []string
	keys := make([]int, 0, len(d.Bonus))
	for k := range d.Bonus {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	for _, k := range keys {
		l = append(l, fmt.Sprintf("+%d %s", d.Bonus[k], AttrNames[k]))
	}
	if d.AV > 0 {
		l = append(l, fmt.Sprintf("+%d AV", d.AV))
	}
	if d.DV > 0 {
		l = append(l, fmt.Sprintf("+%d DV", d.DV))
	}
	if d.Quick > 0 {
		l = append(l, fmt.Sprintf("+%d quickness", d.Quick))
	}
	if d.Light > 0 {
		l = append(l, fmt.Sprintf("light radius %d", d.Light))
	}
	if d.Resist {
		l = append(l, "elemental damage cut by a third")
	}
	if d.Weapon[0] > 0 {
		l = append(l, fmt.Sprintf("natural weapon %dd%d →%d in an empty hand", d.Weapon[0], d.Weapon[1], d.Weapon[2]))
	}
	if d.Power != "" {
		act := "grants"
		if pd := powerByID[d.Power]; pd != nil && pd.Active {
			act = "activate (a) for"
		}
		l = append(l, act+" "+g.Sk.PowerName(d.Power))
	}
	return l
}

// augDescribe is the Describe block of an augment item.
func (g *Game) augDescribe(it *Item) []string {
	d := augByID[it.Effect]
	if d == nil {
		return nil
	}
	sk := g.augSkin()
	l := []string{fmt.Sprintf("%s for the %s. Capacity cost {{W|%d}} (you: %d/%d).", titleCase(sk.Noun), augSlotNames[d.Slot], d.Cost, g.P.AugLoad(), g.P.AugCapacity())}
	l = append(l, "{{G|"+strings.Join(g.augEffects(d), " · ")+"}}")
	l = append(l, fmt.Sprintf("{{K|A %s can install or remove it.}}", sk.Installer))
	return l
}

// ---- abilities ----

// augAbilities: active powers granted by working augments the player does not already have as powers.
func (g *Game) augAbilities() []Ability {
	var out []Ability
	seen := map[string]bool{}
	for _, p := range g.P.Body {
		d := augOn(p)
		if d == nil || d.Power == "" || seen[d.Power] || g.P.Powers[d.Power] > 0 {
			continue
		}
		pd := powerByID[d.Power]
		if pd == nil || !pd.Active {
			continue
		}
		seen[d.Power] = true
		out = append(out, Ability{ID: d.Power, Name: p.Aug.Name, Target: pd.Target, Cool: g.P.Cool[d.Power]})
	}
	return out
}

// augSevered is called when a limb is cut off.
func (g *Game) augSevered(a *Actor, p *BodyPart) {
	if a == g.P && p.Aug != nil {
		g.Msg(fmt.Sprintf("{{R|Your %s falls silent.}}", p.Aug.Name))
	}
}

// ---- installers ----

// augInstallerSpot is the city sub-zone that holds the town's installer, or ok=false.
func (g *Game) augInstallerSpot(cp *cityPlan) ([2]int, bool) {
	prefs := augInstallerDistricts[g.Genre]
	if prefs == nil {
		prefs = augInstallerDistricts["fantasy"]
	}
	order := append([][2]int{{1, 1}}, sideOrder...)
	for _, k := range prefs {
		for _, s := range order {
			if cp.districts[s] == k {
				return s, true
			}
		}
	}
	if cp.tier == 2 {
		return [2]int{1, 1}, true
	}
	return [2]int{}, false
}

// placeInstaller adds the town's augment installer to its district zone (side rng; other zones untouched).
func (g *Game) placeInstaller(zn *Zone, bd *BurgData, cp *cityPlan) {
	if zn.Z != 0 || cp.tier == 0 {
		return
	}
	s, ok := g.augInstallerSpot(cp)
	if !ok || s != [2]int{zn.ZX, zn.ZY} {
		return
	}
	r := NewRng(g.Seed + ":aug:" + zn.Key)
	x, y, ok := zn.FreeSpot(r, func(x, y int) bool { return zn.At(x, y) == TCobble || zn.At(x, y) == TFloor })
	if !ok {
		return
	}
	a := g.makeNPC("augmenter", bd, g.W.cultByID[bd.Culture], r)
	a.Title = g.augSkin().Installer
	a.Desc = fmt.Sprintf("A %s of %s, who fits and removes %ss.", a.Title, bd.Name, g.augSkin().Noun)
	a.AI = "still"
	a.X, a.Y, a.HomeX, a.HomeY = x, y, x, y
	a.Gold = r.Int(200, 400) + a.Level*25
	var pool []string
	for _, d := range AugDefs {
		if !d.Rare {
			pool = append(pool, d.ID)
		}
	}
	n := r.Int(4, 6)
	for i := 0; i < n && len(pool) > 0; i++ {
		k := r.Int(0, len(pool)-1)
		a.Inv = append(a.Inv, g.MakeAugment(pool[k]))
		pool = append(pool[:k], pool[k+1:]...)
	}
	zn.Actors = append(zn.Actors, a)
}

func (g *Game) augFee(d *AugDef) int {
	return (25 + 20*d.Cost) * (100 - g.Skills["persuasion"]*10) / 100
}

// augFreePart is a working, empty body part of the player that fits the augment.
func (g *Game) augFreePart(d *AugDef) *BodyPart {
	for _, p := range g.P.Body {
		if augFits(d.Slot, p) && !p.Severed && p.Aug == nil {
			return p
		}
	}
	return nil
}

// augCanInstall explains why an augment cannot be fitted ("" when it can).
func (g *Game) augCanInstall(d *AugDef) string {
	if g.augFreePart(d) == nil {
		return "no free " + augSlotNames[d.Slot]
	}
	if g.P.AugLoad()+d.Cost > g.P.AugCapacity() {
		return fmt.Sprintf("needs %d capacity, %d free", d.Cost, g.P.AugCapacity()-g.P.AugLoad())
	}
	return ""
}

// InstallAug fits an augment item into the player (no fee); it returns "" or the reason it failed.
func (g *Game) InstallAug(it *Item) string {
	d := augByID[it.Effect]
	if it.Kind != "augment" || d == nil {
		return "not an augment"
	}
	if why := g.augCanInstall(d); why != "" {
		return why
	}
	p := g.augFreePart(d)
	p.Aug = it
	g.removeInv(it)
	return ""
}

// RemoveAug takes the augment out of a part and returns it to the pack.
func (g *Game) RemoveAug(p *BodyPart) *Item {
	it := p.Aug
	if it == nil {
		return nil
	}
	p.Aug = nil
	g.P.Inv = append(g.P.Inv, it)
	return it
}

// augMenu: the installer's install/remove dialogue.
func (g *Game) augMenu(a *Actor) {
	sk := g.augSkin()
	g.Popup = Wrap(fmt.Sprintf("\"Which %s shall it be?\" {{K|Capacity %d/%d (3 + Toughness modifier + level/5).}}", sk.Noun, g.P.AugLoad(), g.P.AugCapacity()), 100)
	g.Opts = nil
	add := func(t string, f func()) { g.Opts = append(g.Opts, Option{t, f}) }
	fit := func(it *Item, price int, fromStock bool) {
		if g.P.Gold < price {
			g.talkMenu("\"Come back with coin.\"")
			return
		}
		if fromStock {
			removeItem(&a.Inv, it)
			g.P.Inv = append(g.P.Inv, it)
		}
		if why := g.InstallAug(it); why != "" {
			if fromStock {
				removeItem(&g.P.Inv, it)
				a.Inv = append(a.Inv, it)
			}
			g.talkMenu("\"I can't fit that: " + why + ".\"")
			return
		}
		g.P.Gold -= price
		a.Gold += price
		g.Turn += 120
		g.Msg("{{G|" + fmt.Sprintf(sk.Install, it.Name) + "}}")
		g.augMenu(a)
	}
	for _, it := range g.P.Inv {
		it := it
		d := augByID[it.Effect]
		if it.Kind != "augment" || d == nil {
			continue
		}
		fee := g.augFee(d)
		if why := g.augCanInstall(d); why != "" {
			add(fmt.Sprintf("{{K|Install %s (%s)}}", it.Name, why), func() { g.augMenu(a) })
			continue
		}
		add(fmt.Sprintf("Install your %s — %d %s", it.Name, fee, g.Sk.Currency), func() { fit(it, fee, false) })
	}
	for _, it := range a.Inv {
		it := it
		d := augByID[it.Effect]
		if it.Kind != "augment" || d == nil {
			continue
		}
		price := g.augFee(d) + g.price(it, true)
		if why := g.augCanInstall(d); why != "" {
			add(fmt.Sprintf("{{K|Buy and install %s (%s)}}", it.Name, why), func() { g.augMenu(a) })
			continue
		}
		add(fmt.Sprintf("Buy and install %s {{K|(%s)}} — %d %s", it.Name, strings.Join(g.augEffects(d), ", "), price, g.Sk.Currency), func() { fit(it, price, true) })
	}
	for _, p := range g.P.Body {
		p := p
		if p.Aug == nil {
			continue
		}
		d := augByID[p.Aug.Effect]
		if d == nil {
			continue
		}
		fee := g.augFee(d) / 2
		add(fmt.Sprintf("Remove the %s from your %s — %d %s", p.Aug.Name, strings.ToLower(p.Name), fee, g.Sk.Currency), func() {
			if g.P.Gold < fee {
				g.talkMenu("\"Come back with coin.\"")
				return
			}
			g.P.Gold -= fee
			a.Gold += fee
			g.Turn += 60
			it := g.RemoveAug(p)
			g.Msg(fmt.Sprintf("{{Y|The %s is taken out and handed back to you.}}", it.Name))
			g.augMenu(a)
		})
	}
	add("Enough for now.", func() { g.talkMenu(g.greeting(a)) })
	g.Sel = 0
}

func removeItem(list *[]*Item, it *Item) {
	for i, x := range *list {
		if x == it {
			*list = append((*list)[:i], (*list)[i+1:]...)
			return
		}
	}
}

// ---- society ----

// augAttitude: -1 purist (dislikes heavy augmentation), +1 enthusiast, 0 indifferent; from the NPC's faith.
func (g *Game) augAttitude(a *Actor) int {
	if a == nil || g.W == nil || a.Role == "augmenter" {
		return 0
	}
	rel := g.W.relByID[a.Religion]
	if rel == nil {
		return 0
	}
	switch {
	case rel.Theology == "animism" || rel.Theology == "pantheism" || rel.Doctrine == "ancestral":
		return -1
	case rel.Doctrine == "esoteric" || rel.Doctrine == "liberation":
		return 1
	}
	return 0
}

func (g *Game) augHeavy() bool { return g.P.AugLoad() >= 4 }

// augPriceMod scales what the NPC charges: purists overcharge the heavily augmented, enthusiasts give a little.
func (g *Game) augPriceMod(a *Actor) float64 {
	switch att := g.augAttitude(a); {
	case att < 0 && g.augHeavy():
		return 1.2
	case att > 0 && g.P.AugLoad() > 0:
		return 0.92
	}
	return 1
}

// augRemark is a purist's greeting to a heavily augmented wanderer, or "".
func (g *Game) augRemark(a *Actor) string {
	if g.augAttitude(a) < 0 && g.augHeavy() {
		return g.augSkin().Purist
	}
	return ""
}

// augSheet lists installed augments for the character sheet.
func (g *Game) augSheet() []string {
	sk := g.augSkin()
	l := []string{fmt.Sprintf("{{W|%ss}}  capacity {{W|%d/%d}}  {{K|(3 + Toughness modifier + level/5; installed by a %s)}}", titleCase(sk.Noun), g.P.AugLoad(), g.P.AugCapacity(), sk.Installer), ""}
	any := false
	for _, p := range g.P.Body {
		if p.Aug == nil {
			continue
		}
		any = true
		d := augByID[p.Aug.Effect]
		st := ""
		if p.Severed {
			st = " {{R|(limb lost: inactive)}}"
		}
		eff := ""
		if d != nil {
			eff = strings.Join(g.augEffects(d), " · ")
		}
		l = append(l, fmt.Sprintf("{{c|%-16s}} {{Y|%s}}%s", p.Name, p.Aug.Name, st), "    {{K|"+eff+"}}")
	}
	if !any {
		l = append(l, fmt.Sprintf("{{K|No %ss installed. Look for a %s in a larger town.}}", sk.Noun, sk.Installer))
	}
	return l
}
