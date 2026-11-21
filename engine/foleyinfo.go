package engine

import "strings"

// Item- and material-aware foley detection for the web shell (prefix fy). FoleyBegin snapshots inventory,
// equipment, gold, hunger and thirst before a key press; FoleyEnd diffs them, reads the new message-log lines
// and returns events {id, item{kind,family,material,rarity,hash,weight}, count, surface, cond, shoe, genre}.
// Read-only and cosmetic: no g.R, no game state changes (the snapshot lives in g.ext and has no exported
// fields, so saves ignore it).

type fyInvEnt struct {
	it *Item
	c  int
}

type fySnap struct {
	n, rep, turn   int
	last           string
	gold           int
	inv            []fyInvEnt
	worn           map[*Item]string // equipped items by slot
	hunger, thirst int
	x, y           int
	zone           string
}

type fyState struct{ s fySnap }

func fyCount(it *Item) int {
	if it.Count > 0 {
		return it.Count
	}
	return 1
}

func (g *Game) fyTake() fySnap {
	s := fySnap{n: len(g.Msgs), worn: map[*Item]string{}, hunger: g.Hunger, thirst: g.Thirst}
	if s.n > 0 {
		m := g.Msgs[s.n-1]
		s.last, s.turn, s.rep = m.T, m.Turn, m.Rep
	}
	if g.Cur != nil {
		s.zone = g.Cur.Key
	}
	if g.P != nil {
		s.gold, s.x, s.y = g.P.Gold, g.P.X, g.P.Y
		for _, it := range g.P.Inv {
			s.inv = append(s.inv, fyInvEnt{it, fyCount(it)})
		}
		for _, p := range g.P.Body {
			if p.Item != nil {
				s.worn[p.Item] = p.Slot
			}
		}
	}
	return s
}

// FoleyBegin records the state just before a key is handled.
func (g *Game) FoleyBegin() {
	st := g.ext("foley", func() any { return &fyState{} }).(*fyState)
	st.s = g.fyTake()
}

func fyHas(n string, ws ...string) bool {
	for _, w := range ws {
		if strings.Contains(n, w) {
			return true
		}
	}
	return false
}

// fyMaterial infers a coarse material from kind and name (items carry no material field).
func fyMaterial(it *Item) string {
	n := strings.ToLower(it.Name)
	switch {
	case fyHas(n, "plasma", "ion ", "photon", "laser", "energy"):
		return "plasma"
	case fyHas(n, "crystal", "quartz", "prism", "diamond", "ruby", "emerald", "sapphire", "gem"):
		return "crystal"
	case fyHas(n, "glass", "vial", "flask", "bottle", "lens", "mirror"):
		return "glass"
	case fyHas(n, "gold", "golden", "coin", "crown"):
		return "gold"
	case fyHas(n, "silver", "mithril", "moonsteel"):
		return "silver"
	case fyHas(n, "bronze", "brass", "copper"):
		return "bronze"
	case fyHas(n, "steel", "plate", "mail", "armor", "armour", "helm", "gauntlet", "greave", "cuirass", "chain", "titanium", "alloy", "kevlar", "hauberk"):
		return "steel"
	case fyHas(n, "iron", "rusty", "nail", "key", "hammer", "pick", "tongs", "anvil"):
		return "iron"
	case fyHas(n, "bone", "tusk", "fang", "skull", "horn", "ivory", "claw"):
		return "bone"
	case fyHas(n, "stone", "rock", "flint", "obsidian", "granite", "marble", "pebble"):
		return "stone"
	case fyHas(n, "leather", "hide", "fur", "boot", "jerkin", "pelt", "sandal", "belt", "glove"):
		return "leather"
	case fyHas(n, "cloth", "robe", "silk", "linen", "cotton", "tunic", "cloak", "wool", "scarf", "hat", "shirt", "rag", "wrap", "scroll", "book", "letter"):
		return "cloth"
	case fyHas(n, "wood", "oak", "bow", "staff", "club", "arrow", "bolt", "crate", "barrel", "chest", "pine", "ash ", "yew", "spear", "log", "plank"):
		return "wood"
	case fyHas(n, "flesh", "meat", "slime", "gland", "organ", "bio", "chitin", "scale", "egg", "carcass", "fungus", "mushroom", "berry", "herb"):
		return "bio"
	}
	switch it.Kind {
	case "armor":
		return "leather"
	case "weapon", "ranged", "grenade":
		return "iron"
	case "book", "letter", "scrap":
		return "cloth"
	case "food", "water", "tonic":
		return "bio"
	}
	return "iron"
}

func fyItem(it *Item) map[string]any {
	kind := it.Kind
	switch {
	case fyHas(strings.ToLower(it.Name), "gem", "jewel", "ruby", "emerald", "sapphire", "diamond", "crystal"):
		kind = "gem"
	case fyHas(strings.ToLower(it.Name), " key", "key ", "lockpick"):
		kind = "key"
	case fyHas(strings.ToLower(it.Name), "pickaxe", "hammer", "shovel", "tongs", "saw", "tool", "wrench"):
		if kind != "weapon" {
			kind = "tool"
		}
	case fyHas(strings.ToLower(it.Name), "bone", "skull"):
		if kind != "weapon" && kind != "armor" {
			kind = "bone"
		}
	case fyHas(strings.ToLower(it.Name), "bottle", "flask", "vial") && kind != "tonic":
		kind = "glass"
	}
	fam := wpFamOf(it)
	if fam == "" && (it.Kind == "weapon" || it.Kind == "ranged") {
		fam = fyFamFromName(it.Name)
	}
	r := it.Rarity
	if r == "" {
		r = "common"
	}
	return map[string]any{"kind": kind, "family": fam, "material": fyMaterial(it), "rarity": r,
		"hash": int(HashStr(it.Name) % 1000), "weight": it.Weight, "name": it.Name}
}

// fyFamFromName maps legacy weapon names onto the wpFamilies-style families plus a few blade ones.
func fyFamFromName(name string) string {
	n := strings.ToLower(name)
	switch {
	case fyHas(n, "whip", "lash", "scourge"):
		return "whip"
	case fyHas(n, "crossbow", "arbalest"):
		return "crossbow"
	case fyHas(n, "sling"):
		return "sling"
	case fyHas(n, "bow"):
		return "longbow"
	case fyHas(n, "pistol", "revolver"):
		return "pistol"
	case fyHas(n, "rifle", "musket", "carbine", "arquebus", "cannon", "gun", "blaster"):
		return "longgun"
	case fyHas(n, "spear", "lance", "pike", "javelin", "trident"):
		return "spear"
	case fyHas(n, "axe", "hatchet", "cleaver"):
		return "axe"
	case fyHas(n, "mace", "hammer", "club", "maul", "cudgel", "flail"):
		return "mace"
	case fyHas(n, "dagger", "dirk", "stiletto", "knife", "rapier"):
		return "dagger"
	case fyHas(n, "staff"):
		return "staff"
	case fyHas(n, "shield", "buckler"):
		return "shield"
	case fyHas(n, "fist", "knuckle", "cestus"):
		return "fist"
	}
	return "sword"
}

func (g *Game) fyShoe() string {
	if g.P == nil {
		return "bare"
	}
	for _, p := range g.P.Body {
		if p.Severed {
			continue
		}
		ln := strings.ToLower(p.Name)
		if p.Slot == "feet" {
			if p.Item != nil {
				in := strings.ToLower(p.Item.Name)
				if fyHas(in, "sandal", "slipper", "wrap", "shoe") {
					return "sandals"
				}
				return "boots"
			}
			if fyHas(ln, "hoof") {
				return "hooves"
			}
		}
		if fyHas(ln, "hoof") {
			return "hooves"
		}
	}
	return "bare"
}

func fySurface(t Terrain) string {
	switch t {
	case TGrass, TTallGrass, TMoss, TBush, TReeds, TCrop, TFungus, TMushroom, TPalm:
		return "grass"
	case TDirt, TRoad:
		return "dirt"
	}
	return sfxSurface(t)
}

// fyCond reads wet / mud / snow from terrain and weather.
func (g *Game) fyCond(t Terrain) string {
	switch t {
	case TShallow, TDeep, TQuicksand, TTar, TSpray:
		return "mud"
	case TSnow, TSnowDrift, TGlacier, TIce:
		return "snow"
	}
	if w := g.activeWeather(); w != nil && w.Kind != nil {
		if w.Kind.Wet >= 2 {
			if t == TDirt || t == TGrass || t == TTallGrass || t == TRoad || t == TMoss {
				return "mud"
			}
			return "wet"
		}
		if w.Kind.ID == "snow" || w.Kind.ID == "blizzard" {
			return "snow"
		}
	}
	return ""
}

func fyTarget(low string) string {
	switch {
	case fyHas(low, "skeleton", "bone", "lich", "wight", "revenant"):
		return "bone"
	case fyHas(low, "golem", "gargoyle", "statue", "earth", "rock"):
		return "stone"
	case fyHas(low, "treant", "tree", "woodwose", "mimic", "crate", "barrel", "dummy"):
		return "wood"
	case fyHas(low, "robot", "automaton", "drone", "android", "turret", "cyborg", "clockwork", "tin "):
		return "metal"
	case fyHas(low, "slime", "ooze", "jelly", "blob"):
		return "bio"
	case fyHas(low, "knight", "guard", "soldier", "captain", "legionary", "paladin", "plate"):
		return "armour"
	}
	return "flesh"
}

func fyGun(genre, fam string) string {
	switch genre {
	case "fantasy", "mythic", "steampunk":
		return "flintlock"
	case "dieselpunk", "wasteland", "twilight":
		if fam == "longgun" {
			return "rifle"
		}
		return "revolver"
	case "atompunk":
		return "zap"
	case "cyberpunk":
		if fam == "longgun" {
			return "railgun"
		}
		return "plasma"
	case "biopunk", "solarpunk":
		return "zap"
	}
	return "revolver"
}

// FoleyEnd returns the foley events caused since FoleyBegin, as []any for the wasm frame ("foley" field).
func (g *Game) FoleyEnd() []any {
	st := g.ext("foley", func() any { return &fyState{} }).(*fyState)
	old, now := st.s, g.fyTake()
	if g.P == nil {
		return nil
	}
	genre := wpGenre(g.Genre)
	var out []any
	seen := map[string]bool{}
	surf, shoe, cond := "stone", g.fyShoe(), ""
	if g.Cur != nil {
		t := g.Cur.At(now.x, now.y)
		surf, cond = fySurface(t), g.fyCond(t)
	}
	add := func(id string, it *Item, count int) {
		key := id
		if it != nil {
			key += "|" + it.Name
		}
		if seen[key] || len(out) >= 8 {
			return
		}
		seen[key] = true
		ev := map[string]any{"id": id, "count": count, "surface": surf, "cond": cond, "shoe": shoe, "genre": genre}
		if it != nil {
			ev["item"] = fyItem(it)
		}
		out = append(out, ev)
	}
	var fresh []Msg
	if k := now.n - old.n; k > 0 && k <= len(g.Msgs) {
		fresh = g.Msgs[len(g.Msgs)-k:]
	} else if now.n > 0 && (now.last != old.last || now.turn != old.turn || now.rep != old.rep) {
		fresh = g.Msgs[now.n-1:]
	}
	var lows []string
	for _, m := range fresh {
		lows = append(lows, strings.ToLower(m.T))
	}
	anyMsg := func(ws ...string) bool {
		for _, l := range lows {
			if fyHas(l, ws...) {
				return true
			}
		}
		return false
	}
	oldInv := map[*Item]int{}
	for _, e := range old.inv {
		oldInv[e.it] = e.c
	}
	nowInv := map[*Item]int{}
	for _, e := range now.inv {
		nowInv[e.it] = e.c
	}
	// equipment
	for it, slot := range now.worn {
		if _, was := old.worn[it]; !was {
			fyEquip(add, it, slot, true)
		}
	}
	for it, slot := range old.worn {
		if _, is := now.worn[it]; !is {
			fyEquip(add, it, slot, false)
		}
	}
	// consumption: inventory shrank and hunger/thirst fell or the log says so
	eating := anyMsg("you eat", "you devour", "you swallow", "you chew")
	drinking := anyMsg("you drink", "you quaff", "you sip", "you gulp")
	smoking := anyMsg("you smoke", "you puff", "you inhale")
	trading := anyMsg("you buy", "you sell", "you purchase", "you trade", "you sold", "you bought")
	giving := anyMsg("you give", "you hand", "you offer", "you gift")
	// inventory diff
	for it, c := range oldInv {
		d := c - nowInv[it]
		if d <= 0 {
			continue
		}
		switch {
		case eating && (it.Kind == "food"):
			add("eat", it, d)
		case drinking && (it.Kind == "water" || it.Kind == "tonic"):
			if it.Kind == "tonic" {
				add("potion", it, d)
			} else {
				add("drink", it, d)
			}
		case smoking:
			add("smoke", it, d)
		case it.Kind == "tonic" && now.hunger == old.hunger && anyMsg("you use", "you apply", "you inject"):
			add("potion", it, d)
		case trading:
			add("trade", it, maxi(it.Value, 1)*d)
		case giving:
			add("give", it, d)
		default:
			if _, worn := now.worn[it]; !worn && old.worn[it] == "" {
				add("drop", it, d)
			}
		}
	}
	for it, c := range nowInv {
		if d := c - oldInv[it]; d > 0 {
			if trading {
				add("trade", it, maxi(it.Value, 1)*d)
			} else {
				add("pickup", it, d)
			}
		}
	}
	if now.gold != old.gold {
		d := now.gold - old.gold
		if d < 0 {
			d = -d
		}
		if trading || anyMsg("haggle", "bargain") {
			add("trade", nil, d)
			if anyMsg("haggle", "bargain") {
				add("haggle", nil, 1)
			}
		} else if now.gold > old.gold {
			add("coin", nil, d)
		} else {
			add("coin:spend", nil, d)
		}
	}
	// messages
	w := (*Item)(nil)
	if ws := g.P.Weapons(); len(ws) > 0 {
		w = ws[0]
	}
	if m := g.P.Missile(); m != nil {
		w = m
	}
	for _, low := range lows {
		mine := strings.HasPrefix(low, "you ") || strings.HasPrefix(low, "{{")
		switch {
		case mine && fyHas(low, "you open the chest", "you open the crate", "you open the trunk", "you open the barrel", "you open the coffer", "you open the box", "you open the container", "you open the cabinet", "you open the sarcophagus"):
			ci := &Item{Name: strings.TrimPrefix(low[strings.Index(low, "open the ")+9:], "the "), Kind: "container"}
			add("open", ci, 1)
		case fyHas(low, "unlock", "lockpick", "the lock clicks", "you pick the lock"):
			add("lock", nil, 1)
		case fyHas(low, "the lock jams", "lock is jammed"):
			add("jam", nil, 1)
		case mine && fyHas(low, "you reload", "you load the", "you chamber"):
			add("reload", w, 1)
		case fyHas(low, "jams", "misfire", "clicks empty", "out of ammo"):
			add("jam", w, 1)
		case mine && fyHas(low, "you read", "turn the page", "you open the book"):
			add("page", nil, 1)
		case mine && fyHas(low, "you write", "you scribe", "you journal", "you jot", "you record", "you inscribe"):
			add("write", nil, 1)
		case mine && fyHas(low, "you cook", "you fry", "you roast", "you bake", "you simmer", "you boil", "you grill"):
			add("craft:cook", nil, 1)
		case mine && fyHas(low, "you chop", "you slice", "you carve", "you butcher", "you skin", "you gut"):
			add("craft:chop", nil, 1)
		case mine && fyHas(low, "you forge", "you smith", "you hammer", "you temper", "you repair", "you sharpen", "you craft", "you fashion"):
			add("craft:hammer", nil, 1)
		case mine && fyHas(low, "you grind", "you crush", "you mill", "you pound"):
			add("craft:grind", nil, 1)
		case mine && fyHas(low, "you brew", "you distill", "you mix", "you concoct", "you infuse", "you transmute"):
			add("craft:brew", nil, 1)
		case fyHas(low, "fail to penetrate"):
			add("impact:armour", w, 1)
		case fyHas(low, "parr"):
		case fyHas(low, " for ") && fyHas(low, " damage"):
			tgt := fyTarget(low)
			if fyHas(low, "bash", "shield") {
				tgt = "shield"
			}
			if w != nil && mine {
				if w.Kind == "ranged" || fyHas(low, "you shoot", "you fire") {
					add("shot", w, 1)
				} else {
					add("swing", w, 1)
				}
			}
			add("impact:"+tgt, w, 1)
		case mine && fyHas(low, " miss"):
			if w != nil {
				add("swing", w, 1)
			}
		}
	}
	// footsteps
	if (now.x != old.x || now.y != old.y) && old.zone == now.zone && g.Cur != nil && g.Mode != "world" && !g.OnWorld {
		add("step", nil, 1)
	}
	if fy, ok := g.fyGenreGun(genre, w); ok {
		for _, o := range out {
			if ev, ok := o.(map[string]any); ok && (ev["id"] == "shot" || ev["id"] == "reload") {
				ev["gun"] = fy
			}
		}
	}
	return out
}

func (g *Game) fyGenreGun(genre string, w *Item) (string, bool) {
	if w == nil {
		return "", false
	}
	fam := wpFamOf(w)
	if fam == "" {
		fam = fyFamFromName(w.Name)
	}
	if fam == "pistol" || fam == "longgun" || fam == "repeater" {
		return fyGun(genre, fam), true
	}
	return "", false
}

func fyEquip(add func(string, *Item, int), it *Item, slot string, on bool) {
	switch it.Kind {
	case "weapon", "ranged":
		if on {
			add("draw", it, 1)
		} else {
			add("sheathe", it, 1)
		}
	default:
		if on {
			add("equip", it, 1)
		} else {
			add("unequip", it, 1)
		}
	}
	_ = slot
}
