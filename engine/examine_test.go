package engine

import (
	"strings"
	"testing"
)

func TestExComplexityRange(t *testing.T) {
	low := &Item{Kind: "artifact", Tier: 0}
	high := &Item{Kind: "artifact", Tier: 5, Rarity: "mythic", Cat: "occult", SimArt: 1}
	if c := exComplexity(low); c != 1 {
		t.Fatalf("low tier complexity = %d, want 1", c)
	}
	if c := exComplexity(high); c != 10 {
		t.Fatalf("high complexity = %d, want clamp at 10", c)
	}
	mid := &Item{Kind: "artifact", Tier: 2, Rarity: "rare"}
	if exComplexity(mid) <= exComplexity(low) || exComplexity(mid) >= exComplexity(high) {
		t.Fatalf("complexity not ordered: low %d mid %d high %d", exComplexity(low), exComplexity(mid), exComplexity(high))
	}
}

func TestExDescribeSteps(t *testing.T) {
	it := &Item{Kind: "artifact", Unid: "humming disc", Effect: "teleport", Weight: 2}
	if l := exDescribe(it); len(l) == 0 || !strings.Contains(l[0], "humming disc") {
		t.Fatalf("step 0 should show the unidentified name: %v", l)
	}
	exSetStep(it, 1)
	if l := strings.Join(exDescribe(it), "\n"); !strings.Contains(l, "Shape:") || strings.Contains(l, "teleport") {
		t.Fatalf("step 1 should show shape only: %s", l)
	}
	exSetStep(it, 2)
	if l := strings.Join(exDescribe(it), "\n"); !strings.Contains(l, "It seems to") {
		t.Fatalf("step 2 should show function: %s", l)
	}
	it.Ident = true
	if exDescribe(it) != nil {
		t.Fatal("identified items should return nil from exDescribe")
	}
}

func TestExExamineProgress(t *testing.T) {
	g := newGame(t, nil)
	g.Skills["tinkering"] = 3
	it := g.MakeOddity(g.R, 0)
	g.give(it)
	for i := 0; i < 5 && !it.Ident; i++ {
		before := exStep(it)
		ok, msg := g.exExamine(it)
		if msg == "" {
			t.Fatal("examine returned empty message")
		}
		if ok && exStep(it) <= before && !it.Ident {
			t.Fatalf("successful examine did not advance step (step %d)", exStep(it))
		}
	}
	if !it.Ident {
		t.Fatalf("item not identified after repeated examinations, step %d", exStep(it))
	}
	if ok, _ := g.exExamine(it); ok {
		t.Fatal("examining an identified item should report ok=false")
	}
}

func TestExExamineBadFailureBreaksOrBursts(t *testing.T) {
	g := newGame(t, nil)
	g.Skills["tinkering"] = 0
	broke := false
	for i := 0; i < 60; i++ {
		it := &Item{ID: g.newID(), Kind: "artifact", Name: "test relic", Unid: "black prism", Tier: 5, Rarity: "mythic", Cat: "occult", Effect: "flames"}
		g.give(it)
		ok, msg := g.exExamine(it)
		if ok || it.Ident {
			t.Fatalf("skill 0 examine should not succeed on complexity %d: %s", exComplexity(it), msg)
		}
		if strings.Contains(msg, "snaps") || strings.Contains(msg, "bursts") {
			broke = true
			for _, o := range g.P.Inv {
				if o == it {
					t.Fatal("broken artifact still in inventory")
				}
			}
		}
		if g.P.HP <= 0 {
			g.P.HP = g.P.MaxHP
		}
	}
	if !broke {
		t.Fatal("expected at least one breakage or burst over 60 bad examinations")
	}
}
