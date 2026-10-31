package engine

import "fmt"

// Artifact complexity and examination (QUD 1.12). Unidentified artifacts reveal
// themselves in three steps: shape, then function, then name. Each successful
// examination reveals one step (two when the roll beats the target by 8+).
// Step lives in Item.Extra["ex:step"]; complexity is derived from tier, rarity
// and category, so nothing new is stored on the item.

// exFunctionText is what an artifact seems to do, by Item.Effect id.
var exFunctionText = map[string]string{
	"teleport":  "bend space, sending whatever it touches somewhere else.",
	"heal":      "knit flesh together when held against a wound.",
	"lightning": "gather a charge and loose it as lightning.",
	"flames":    "breathe heat and set things alight.",
	"freeze":    "pull the warmth out of whatever it is aimed at.",
	"phase":     "let its holder slip partly out of the world.",
	"sense":     "show things that are hidden from ordinary eyes.",
	"forcewall": "raise a wall of force that nothing passes.",
}

// exComplexity is 1..10: higher tier, rarer rarity, a category and relic origin all add complexity.
func exComplexity(it *Item) int {
	if it == nil {
		return 1
	}
	cx := 1 + it.Tier + rarityIndex(it.Rarity)
	if it.Cat != "" {
		cx++
	}
	if it.SimArt > 0 {
		cx += 2
	}
	return clampi(cx, 1, 10)
}

// exStep is how many of shape (1), function (2) and name (3) are known.
func exStep(it *Item) int { return it.Extra["ex:step"] }

func exSetStep(it *Item, n int) {
	if it.Extra == nil {
		it.Extra = map[string]int{}
	}
	it.Extra["ex:step"] = n
}

func exWeightWord(it *Item) string {
	switch {
	case it.Weight <= 0:
		return "its weight is hard to judge"
	case it.Weight <= 1:
		return "it is light"
	case it.Weight <= 4:
		return "it has a middling heft"
	default:
		return "it is heavy for its size"
	}
}

// exDescribe gives the partial description of an unidentified artifact for the current step.
// Identified items return nil; Describe then uses its normal text.
func exDescribe(it *Item) []string {
	if it.Ident {
		return nil
	}
	shape := fmt.Sprintf("Shape: {{M|%s}}, %s.", it.Unid, exWeightWord(it))
	switch exStep(it) {
	case 0:
		return []string{"An object of unknown make: {{M|" + it.Unid + "}}.", "Examine it closely to learn more (Tinkering or Lore help)."}
	case 1:
		return []string{shape, "Examine it again to learn what it does."}
	default:
		fn, ok := exFunctionText[it.Effect]
		if !ok {
			fn = "do something, though what is still unclear."
		}
		return []string{shape, "It seems to " + fn, "Examine it again to learn its name."}
	}
}

// exExamine rolls Tinkering (Lore for relics) + INT against the item's complexity and
// returns (ok, msg). ok is true when a step was revealed; the caller shows the popup then.
// Ordinary failure wastes the turn. A bad failure on a complex (4+) artifact breaks it,
// or it bursts and hurts the examiner. Breaking removes the item from the inventory.
func (g *Game) exExamine(it *Item) (bool, string) {
	if it.Ident {
		return false, "You already know what it is."
	}
	cx := exComplexity(it)
	skill := g.Skills["tinkering"]
	if it.SimArt > 0 {
		skill = g.Skills["lore"]
	}
	roll := g.R.Int(1, 20) + skill*3 + g.P.M(INT)
	target := 4 + 2*cx
	g.spend(1000)

	plain := "You turn it over and over but can't make sense of it."
	switch {
	case roll >= target:
		steps := 1
		if roll-target >= 8 {
			steps = 2
		}
		st := exStep(it)
		for i := 0; i < steps && st < 3; i++ {
			st++
		}
		exSetStep(it, st)
		if st >= 3 {
			it.Ident = true
			g.gainXP(20 + cx*8)
			msg := fmt.Sprintf("{{M|You figure it out: it is %s!}}", it.Name)
			if it.SimArt > 0 {
				msg += " {{W|A relic from the chronicles of this world.}}"
			}
			return true, msg
		}
		if st == 1 {
			return true, "{{M|You work out the shape of it.}}"
		}
		return true, "{{M|You work out what it does.}}"
	case roll >= target-5 || cx < 4:
		return false, plain
	}

	if g.R.Chance(0.6) {
		g.removeInv(it)
		return false, "{{R|Something inside it snaps. The thing falls apart in your hands.}}"
	}
	g.removeInv(it)
	g.hurt(g.P, g.R.Dice(1, 6)+cx/2, nil, "an artifact's backlash")
	return false, "{{R|It sparks and bursts in your hands!}}"
}
