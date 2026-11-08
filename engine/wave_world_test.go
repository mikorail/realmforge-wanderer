package engine

import (
	"strings"
	"testing"
)

func (g *Game) wrSites() []*wrSiteData {
	if g == nil || g.W == nil {
		return nil
	}
	return wrReg(g.W).Sites
}

func TestWaveWorldBurials(t *testing.T) {
	g := newGame(t, nil)
	ix := buIdx(g.W)
	if ix == nil || len(ix.sites) == 0 {
		t.Fatalf("expected buIdx to populate burial/hoard/bunker sites")
	}
	kinds := map[string]int{}
	for _, s := range ix.sites {
		kinds[s.Kind]++
		if ix.byP[s.P] != s || ix.byID[s.ID] != s || ix.byNum[s.Num] != s {
			t.Fatalf("buIdx lookup mismatch for site %+v", s)
		}
	}
	if kinds["barrow"] == 0 && kinds["hoard"] == 0 {
		t.Fatalf("expected barrow or hoard sites in world, got %v", kinds)
	}

	digs := g.buDigSites()
	if len(digs) == 0 {
		t.Fatalf("expected buDigSites to return live hoard parasangs")
	}
	p0 := digs[0]
	if !g.buRivalLoot(p0) {
		t.Fatalf("expected buRivalLoot(%d) to succeed on unlooted hoard", p0)
	}
	if g.buRivalLoot(p0) {
		t.Fatalf("expected second buRivalLoot(%d) to return false", p0)
	}

	// Test bunker placement and buSearch hatch reveal
	bk := &buSite{
		Num:  99999,
		ID:   "bk:test",
		Kind: "bunker",
		P:    g.W.Idx(g.WX, g.WY),
		WX:   g.WX,
		WY:   g.WY,
		ZX:   g.Cur.ZX,
		ZY:   g.Cur.ZY,
		Name: "Vault under Test",
	}
	ix.byID[bk.ID] = bk
	ix.byNum[bk.Num] = bk
	defer func() {
		delete(ix.byID, bk.ID)
		delete(ix.byNum, bk.Num)
	}()
	st := buSt(g)
	rec := st.rec(bk.ID)
	rec.Placed = true
	rec.Stage = 0
	rec.Known = true
	rec.X, rec.Y = clampi(g.P.X+1, 2, MW-3), g.P.Y
	st.ByZone[g.Cur.Key] = bk.ID
	g.P.Attr[INT] = 30
	g.Skills["perception"] = 3
	for i := 0; i < 10 && rec.Stage == 0; i++ {
		buSearch(g)
	}
	if rec.Stage != 1 || g.Cur.At(rec.X, rec.Y) != TInscription {
		t.Fatalf("expected buSearch to reveal bunker hatch (Stage=%d, tile=%v)", rec.Stage, g.Cur.At(rec.X, rec.Y))
	}
}

func TestWaveWorldTradeways(t *testing.T) {
	g := newGame(t, nil)
	goods := g.tqGoodsHere()
	if len(goods) == 0 {
		t.Fatalf("expected tqGoodsHere to return genre goods")
	}
	good := goods[0]
	bd := g.localBurg()
	if bd == nil && len(g.W.Burgs) > 0 {
		bd = &g.W.Burgs[0]
	}
	if bd == nil {
		t.Skip("no burgs in world")
	}
	buyP := g.tqPrice(bd, good, true)
	sellP := g.tqPrice(bd, good, false)
	if buyP <= 0 || sellP <= 0 || buyP < sellP {
		t.Fatalf("expected positive buy >= sell price, got buy=%d sell=%d", buyP, sellP)
	}

	// Embargo makes good banned in bd.State
	g.tq().Ev = append(g.tq().Ev, &tqEvent{
		Kind:  "embargo",
		Realm: bd.State,
		Good:  good.ID,
		Start: -5,
		End:   g.tqToday() + 30,
	})
	if !g.tqBanned(bd.State, good) {
		t.Fatalf("expected %s to be banned under active embargo in state %d", good.ID, bd.State)
	}

	// tqNewItem and tqAdd merging & contract tracking
	it := g.tqNewItem(good, 4, buyP, 77)
	if it == nil || it.Kind != "goods" || it.Count != 4 || it.Extra["tq:c"] != 77 {
		t.Fatalf("unexpected tqNewItem: %+v", it)
	}
	inv0 := len(g.P.Inv)
	g.tqAdd(good, 3, 10, 0, false)
	g.tqAdd(good, 2, 20, 0, false)
	if len(g.P.Inv) != inv0+1 {
		t.Fatalf("expected tqAdd to merge matching trade goods into one stack, inv %d -> %d", inv0, len(g.P.Inv))
	}
	merged := g.P.Inv[len(g.P.Inv)-1]
	if merged.Count != 5 {
		t.Fatalf("expected merged stack count 5, got %d", merged.Count)
	}
}

func TestWaveWorldEndgame(t *testing.T) {
	g := newGame(t, nil)
	kind := g.egPick()
	if kind == "" || egDefs[kind] == nil {
		t.Fatalf("expected egPick to return valid threat kind, got %q", kind)
	}
	g.egRise()
	st := g.eg()
	if st.Phase != 1 || st.Kind == "" || len(st.Occ) == 0 {
		t.Fatalf("expected egRise to enter phase 1 with occupied origin, got phase=%d kind=%q occ=%v", st.Phase, st.Kind, st.Occ)
	}
	gl, fg, bg, ok := g.egOverlay(st.Origin)
	if !ok || gl == 0 || (fg == 0 && bg == 0) {
		t.Fatalf("expected egOverlay at origin %d to be active, got gl=%q fg=%d bg=%d ok=%v", st.Origin, gl, fg, bg, ok)
	}
	occ0 := len(st.Occ)
	g.egSpread(egDay(g) + 5)
	if len(st.Occ) <= occ0 {
		t.Fatalf("expected egSpread to expand occupied parasangs from %d, got %d", occ0, len(st.Occ))
	}
	score, notes := g.egReadiness()
	if score < 0 || score > 100 || len(notes) == 0 {
		t.Fatalf("unexpected egReadiness score=%d notes=%v", score, notes)
	}
}

func TestWaveWorldFamily(t *testing.T) {
	g := newGame(t, nil)
	st := g.fm()
	st.Big = 1
	st.DuelW = 5
	st.Champs = 1
	st.Songs = append(st.Songs,
		&fmSong{Deed: 1, Text: "slew the iron wyrm", Day: 1, Sung: 2},
		&fmSong{Deed: 2, Text: "held the bridge", Day: 2, Sung: 2},
		&fmSong{Deed: 3, Text: "freed the captives", Day: 3, Sung: 2},
	)
	g.fmEvalTitles()
	if st.Titles["dragonbane"] == 0 || st.Titles["duelist"] == 0 || st.Titles["champion"] == 0 || st.Titles["songsung"] == 0 {
		t.Fatalf("expected fmEvalTitles to grant earned titles, got %v", st.Titles)
	}
	st.Worn = "dragonbane"
	smith := &Actor{ID: g.newID(), Name: "Borin", Role: "smith"}
	if mod := g.fmPriceMod(smith); mod >= 1.0 {
		t.Fatalf("expected dragonbane title to give smith discount (< 1.0), got %v", mod)
	}

	st.Spouse = &fmSpouse{ID: g.newID(), Name: "Lyra", Species: g.P.Species, Level: 4, Love: 80}
	st.Kids = append(st.Kids, &fmChild{
		ID:      g.newID(),
		Name:    "Rowan",
		Species: g.P.Species,
		Born:    g.Day() - YearDays*4,
		Attr:    [6]int{14, 14, 14, 14, 14, 14},
		Calling: "warrior",
		Gift:    "a keen eye",
		Bond:    10,
	})
	sum := g.fmSummary()
	if !strings.Contains(sum, "Lyra") || !strings.Contains(sum, "Rowan") || !strings.Contains(sum, "songs sung") {
		t.Fatalf("unexpected fmSummary: %q", sum)
	}
	heir := g.fmHeir()
	if heir == nil || heir.Name != "Rowan" {
		t.Fatalf("expected fmHeir to select Rowan, got %+v", heir)
	}
	if !g.fmContinueAsHeir() || g.P.Name != "Rowan" {
		t.Fatalf("expected fmContinueAsHeir to switch player to Rowan, got %q", g.P.Name)
	}
}

func TestWaveWorldJourneys(t *testing.T) {
	g := newGame(t, nil)
	if len(jyDefs) == 0 || len(g.jyAvail()) == 0 {
		t.Fatalf("expected jyDefs and jyAvail to be populated")
	}
	g.jy().Glider = true
	g.Cur.Z = 0
	cx, cy := clampi(g.P.X+1, 2, MW-10), g.P.Y
	g.Cur.Set(cx, cy, TCliff)
	for step := 2; step <= 6; step++ {
		g.Cur.Set(g.P.X+step, cy, TFloor)
		if a := g.Cur.ActorAt(g.P.X+step, cy); a != nil {
			a.X, a.Y = 1, 1
		}
	}
	if !g.jyCliffBump(cx, cy) || g.Mode != "jy" {
		t.Fatalf("expected jyCliffBump to open jy menu on TCliff with glider")
	}
	g.jyClose()
	x0 := g.P.X
	if !g.jyZoneGlide(1, 0) || g.P.X <= x0+1 {
		t.Fatalf("expected jyZoneGlide(1,0) to move player across cliff, x %d -> %d", x0, g.P.X)
	}
	if blocked, _ := g.jyBlocked(g.W.Idx(g.WX, g.WY)); blocked {
		t.Fatalf("expected home town parasang not to be blocked by route puzzle")
	}
}

func TestWaveWorldLocksSecretsWrecks(t *testing.T) {
	g := newGame(t, nil)
	clearAround(g)

	// 1. locks.go: lkZone places locks in dungeon/ship zones
	dzn := newZone("test.lk.0.0.1", g.WX, g.WY, 0, 0, 1)
	dzn.Kind = "caravel"
	for i := range dzn.T {
		dzn.T[i] = TFloor
	}
	lkZone(g, dzn, NewRng("test:lk"))
	if len(g.lkSt().Locks) == 0 {
		t.Fatalf("expected lkZone on caravel zone to register a lock")
	}

	// 2. secrets.go: sxSearch reveals nearby secret door
	sx := &sxSecret{
		K:  "door",
		X:  clampi(g.P.X+1, 2, MW-3),
		Y:  g.P.Y,
		Fx: g.P.X,
		Fy: g.P.Y,
		Ix: clampi(g.P.X+2, 2, MW-2),
		Iy: g.P.Y,
	}
	g.Cur.Set(sx.X, sx.Y, TWall)
	sxSt(g).Z[g.Cur.Key] = &sxZone{S: []*sxSecret{sx}}
	g.P.Attr[INT] = 30
	g.Skills["perception"] = 3
	for i := 0; i < 10 && !sx.Found; i++ {
		sxSearch(g)
	}
	if !sx.Found || g.Cur.At(sx.X, sx.Y) != TDoor {
		t.Fatalf("expected sxSearch to reveal secret door (Found=%v, tile=%v)", sx.Found, g.Cur.At(sx.X, sx.Y))
	}

	// 3. wrecks.go: wrSites / wrReg builds sunken sites on open sea parasangs
	sites := g.wrSites()
	if len(sites) == 0 {
		t.Fatalf("expected wrSites to return sunken wreck/city/hoard sites")
	}
	s0 := sites[0]
	if got := g.wrSite(s0.P%g.W.W, s0.P/g.W.W); got != s0 {
		t.Fatalf("expected wrSite to return %+v, got %+v", s0, got)
	}
}
