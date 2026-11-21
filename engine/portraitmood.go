package engine

import (
	"strconv"
	"strings"
)

// Portrait faces (prefix pm): the face layer and mood of the person in a talk, trade or character-sheet
// portrait. Read-only: it derives a small description from real state (lives opinion, mood and memories,
// recent mourning in lifesim, culture-pack temper, role, caste, wounds, status effects, the player's age,
// scars, bounty, fatigue) and the web shell draws it (src/portraitfx.ts). Deterministic: hashes of ids,
// never g.R. Nothing is saved.

// PortraitFace is what the shell needs to paint a face over the sprite's head.
type PortraitFace struct {
	Mood      string // neutral pleased warm amused wary angry afraid sad grieving proud scornful tired ill drunk sleepy dying
	Intensity int    // 0..3
	Age       string // child young prime mature old venerable
	Wounds    int    // 0..3 (0 unhurt .. 3 near death)
	Scar      string // "" | brow | cheek | across | eye
	Marking   string // "" | tattoo | warpaint | totem
	MarkColor string // hex, "" = shell default
	Accessory string // "" | hat | circlet | hood | goggles | veil | eyepatch | pipe
	Gaze      string // toward | away
}

var pmMoods = map[string]bool{"neutral": true, "pleased": true, "warm": true, "amused": true, "wary": true, "angry": true,
	"afraid": true, "sad": true, "grieving": true, "proud": true, "scornful": true, "tired": true, "ill": true,
	"drunk": true, "sleepy": true, "dying": true}

func pmHash(a *Actor, salt string) uint64 {
	return HashStr(salt + ":" + a.Name + ":" + strconv.Itoa(a.ID))
}

func pmWounds(a *Actor) int {
	if a.MaxHP <= 0 {
		return 0
	}
	f := a.HP * 100 / a.MaxHP
	switch {
	case f >= 90:
		return 0
	case f >= 60:
		return 1
	case f >= 30:
		return 2
	}
	return 3
}

// pmAge: the stage for a person: the player's aging stage, a lives person's age against the species lifespan,
// or a stable guess from title and role.
func (g *Game) pmAge(a *Actor, lp *lvPerson) string {
	names := []string{"young", "prime", "mature", "old", "venerable"}
	if a == g.P {
		return names[clampi(agStageNow(g), 0, 4)]
	}
	if lp != nil && g.W != nil {
		life := float64(lsLifespan(lp.Species))
		if lp.Species == "" {
			life = float64(lsLifespan(a.Species))
		}
		f := float64(lp.lvAge(lvYear(g))) / life
		switch {
		case f < 0.14:
			return "child"
		case f < 0.28:
			return "young"
		case f < 0.55:
			return "prime"
		case f < 0.75:
			return "mature"
		case f < 0.92:
			return "old"
		}
		return "venerable"
	}
	t := strings.ToLower(a.Title + " " + a.Role)
	switch {
	case strings.Contains(t, "child") || strings.Contains(t, "urchin"):
		return "child"
	case strings.Contains(t, "elder"):
		return "old"
	case strings.Contains(t, "sage") || strings.Contains(t, "hermit"):
		return "venerable"
	}
	return []string{"young", "prime", "prime", "prime", "mature", "mature", "old"}[pmHash(a, "age")%7]
}

func pmHeadGear(a *Actor) string {
	for _, p := range a.Body {
		if p.Severed || p.Item == nil || (p.Slot != "head" && p.Slot != "face") {
			continue
		}
		n := strings.ToLower(p.Item.Name)
		switch {
		case strings.Contains(n, "goggle"):
			return "goggles"
		case strings.Contains(n, "veil"):
			return "veil"
		case strings.Contains(n, "circlet") || strings.Contains(n, "crown") || strings.Contains(n, "diadem"):
			return "circlet"
		case strings.Contains(n, "hood") || strings.Contains(n, "cowl"):
			return "hood"
		case strings.Contains(n, "hat") || strings.Contains(n, "cap") || strings.Contains(n, "turban"):
			return "hat"
		}
	}
	return ""
}

func (g *Game) pmAccessory(a *Actor, lp *lvPerson, scar string) string {
	if a == g.P {
		for _, s := range g.ijSt().Inj {
			if s.ID == "blind" {
				return "eyepatch"
			}
		}
	} else if scar == "eye" {
		return "eyepatch"
	}
	if h := pmHeadGear(a); h != "" {
		return h
	}
	if a == g.P {
		return ""
	}
	r := strings.ToLower(a.Role)
	h := pmHash(a, "acc")
	switch r {
	case "priest", "notable", "noble":
		return "circlet"
	case "tinker", "smith":
		if h%3 == 0 {
			return "goggles"
		}
	case "merchant", "farmer":
		return "hat"
	case "rogue", "bandit", "scout":
		if h%2 == 0 {
			return "hood"
		}
	case "elder", "sailor", "villager":
		if h%5 == 0 {
			return "pipe"
		}
	}
	return ""
}

func (g *Game) pmScar(a *Actor) string {
	if a == g.P {
		for _, s := range g.ijSt().Inj {
			switch s.ID {
			case "scar":
				return "across"
			case "blind":
				return "eye"
			}
		}
		return ""
	}
	h := pmHash(a, "scar")
	switch strings.ToLower(a.Role) {
	case "guard", "soldier", "bandit", "warrior", "smith":
		if h%3 == 0 || a.Level >= 6 && h%2 == 0 {
			return []string{"brow", "cheek", "across", "eye"}[(h>>8)%4]
		}
	}
	if a.Boss || a.Level >= 10 {
		return []string{"brow", "cheek", "across"}[(h>>8)%3]
	}
	return ""
}

func (g *Game) pmMarking(a *Actor, lp *lvPerson) (string, string) {
	if id := trTribeOfActor(a); id != "" {
		v := trVariantOfActor(g, a)
		col := v.AccentHex
		if v.Totem != "" {
			return "totem", col
		}
		return "warpaint", col
	}
	if a.Species == "orc" && a.Role == "" {
		return "warpaint", ""
	}
	h := pmHash(a, "mark")
	if strings.ToLower(a.Role) == "bandit" && h%2 == 0 {
		return "warpaint", ""
	}
	if lp != nil && lp.Culture > 0 && (lp.Culture+int(h%7))%6 == 0 {
		return "tattoo", ""
	}
	return "", ""
}

func pmTemper(g *Game, a *Actor, lp *lvPerson) string {
	if lp == nil {
		return ""
	}
	if sp := g.dvVoice(a); sp != nil {
		return sp.Temper
	}
	return dvTemperFromTraits(lp)
}

// PortraitMood describes the face of the speaker (or the player on the character sheet). Never nil-unsafe:
// a nil actor gives a neutral face.
func (g *Game) PortraitMood(a *Actor) PortraitFace {
	f := PortraitFace{Mood: "neutral", Age: "prime", Gaze: "toward"}
	if a == nil {
		return f
	}
	var lp *lvPerson
	if a != g.P && g.W != nil {
		lp = lvPersonOf(g, a)
	}
	f.Wounds = pmWounds(a)
	f.Age = g.pmAge(a, lp)
	f.Scar = g.pmScar(a)
	f.Marking, f.MarkColor = g.pmMarking(a, lp)
	f.Accessory = g.pmAccessory(a, lp, f.Scar)

	mood, inten := "neutral", 0
	set := func(m string, i int) { mood, inten = m, clampi(i, 0, 3) }
	hpPct := 100
	if a.MaxHP > 0 {
		hpPct = a.HP * 100 / a.MaxHP
	}
	role := strings.ToLower(a.Role)
	t := strings.ToLower(a.Title)

	if a == g.P {
		ij := g.ijSt()
		switch {
		case hpPct < 20:
			set("dying", 3)
		case a.St("ill") > 0 || a.St("plague") > 0 || a.St("poisoned") > 0:
			set("ill", 2)
		case a.St("drunk") > 0 || a.St("tipsy") > 0:
			set("drunk", 2)
		case ij.Down:
			set("sleepy", 3)
		case ij.Fat >= ijTiredAt*1.6:
			set("sleepy", 2)
		case ij.Fat >= ijTiredAt:
			set("tired", 1+int(ij.Fat/(ijTiredAt*2)))
		case a.St("shamed") > 0:
			set("sad", 2)
		case a.St("shaken") > 0 || a.St("terrified") > 0:
			set("afraid", 2)
		case g.uwWantedHere() > 0:
			set("wary", 1+clampi(g.uwWantedHere()/150, 0, 2))
		case a.St("inspired") > 0 || a.St("mighty") > 0:
			set("proud", 1)
		case hpPct < 50:
			set("wary", 1)
		}
		f.Mood, f.Intensity = mood, inten
		f.Gaze = "toward"
		if mood == "dying" || mood == "sleepy" || mood == "drunk" {
			f.Gaze = "away"
		}
		return f
	}

	hostile := g.Hostile(a)
	temper := pmTemper(g, a, lp)
	op := 0
	if lp != nil {
		op = lp.Opinion
		for _, m := range lp.Mem {
			if d := lvDay(g) - m.Day; d >= 0 && d < 20 {
				op += m.Weight / 2
			}
		}
	}
	mourning := false
	if lp != nil {
		if until, ok := lsSt(g).Mourn[lp.Key]; ok && until > lvDay(g) {
			mourning = true
		}
	}
	switch {
	case hpPct < 20:
		set("dying", 3)
	case a.St("asleep") > 0:
		set("sleepy", 2)
	case a.St("ill") > 0 || a.St("plague") > 0 || a.St("poisoned") > 0:
		set("ill", 2)
	case hostile && hpPct < 40:
		set("afraid", 2)
	case hostile:
		set("angry", 2+pmB2i(a.Angry || temper == "gruff"))
	case g.P != nil && g.uwWantedHere() > 0 && (role == "guard" || role == "soldier"):
		set("angry", 1+clampi(g.uwWantedHere()/150, 0, 1))
	case mourning:
		set("grieving", 2+pmB2i(lp.Mood < -50))
	case t == "drunk" || role == "drunk" || strings.Contains(t, "drunk"):
		set("drunk", 2)
	case lp != nil && lp.Mood <= -45:
		set("sad", 1+clampi(-lp.Mood/35, 0, 2))
	case lp != nil && lp.Fear != "" && temper == "anxious":
		set("afraid", 1)
	case op <= -40:
		set("scornful", 2+pmB2i(op <= -70))
	case op <= -15:
		set("wary", 1+pmB2i(op <= -30))
	case op >= 50 && (temper == "warm" || temper == "eager"):
		set("warm", 3)
	case op >= 30:
		set("pleased", 1+clampi((op-30)/25, 0, 2))
	case lp != nil && lp.Mood >= 55:
		set("pleased", 2)
	case temper == "wry":
		set("amused", 1+pmB2i(op >= 10))
	case temper == "proud" || f.Accessory == "circlet" || role == "noble" || a.Caste == "noble" || a.Caste == "ruler":
		set("proud", 1+pmB2i(temper == "proud"))
	case temper == "cold":
		set("scornful", 1)
	case temper == "anxious":
		set("wary", 1)
	case temper == "gruff":
		set("wary", 1)
	case temper == "warm":
		set("pleased", 1)
	case lp != nil && lp.Mood <= -15:
		set("tired", 1)
	case g.P != nil && g.attireAffinity(a) <= -40:
		set("wary", 1)
	case g.P != nil && g.attireAffinity(a) >= 100:
		set("pleased", 1)
	case f.Age == "old" || f.Age == "venerable":
		set("tired", 1)
	}
	if !pmMoods[mood] {
		mood, inten = "neutral", 0
	}
	f.Mood, f.Intensity = mood, inten
	switch mood {
	case "sad", "grieving", "afraid", "scornful", "tired", "dying", "ill", "sleepy", "drunk":
		f.Gaze = "away"
	}
	if mood == "scornful" || mood == "wary" {
		if pmHash(a, "gaze")%3 == 0 {
			f.Gaze = "away"
		}
	}
	return f
}

func pmB2i(b bool) int {
	if b {
		return 1
	}
	return 0
}

// PortraitMoodJS converts a face for the wasm frame (syscall/js cannot take structs).
// Hook: cmd/wasm/main.go, in the portrait map: "mood": engine.PortraitMoodJS(g.PortraitMood(actor)).
func PortraitMoodJS(f PortraitFace) map[string]any {
	m := map[string]any{"mood": f.Mood, "intensity": f.Intensity, "age": f.Age, "wounds": f.Wounds, "gaze": f.Gaze}
	if f.Scar != "" {
		m["scar"] = f.Scar
	}
	if f.Marking != "" {
		m["marking"] = f.Marking
	}
	if f.MarkColor != "" {
		m["markColor"] = f.MarkColor
	}
	if f.Accessory != "" {
		m["accessory"] = f.Accessory
	}
	return m
}

// PortraitFaceOf is the face for the portrait just reserved (nil when none): the speaker, or the player on the sheet.
// Hook: cmd/wasm/main.go: if pf := g.PortraitFaceOf(pi); pf != nil { portrait["mood"] = engine.PortraitMoodJS(*pf) }
func (g *Game) PortraitFaceOf(pi *PortraitInfo) *PortraitFace {
	if pi == nil || g.P == nil {
		return nil
	}
	a := g.Talk
	if pi.Kind == "sheet" || a == nil {
		a = g.P
	}
	f := g.PortraitMood(a)
	return &f
}
