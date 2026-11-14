package engine

// Cross-feature links between plug-in files that were built in parallel and
// expose hook variables instead of calling each other directly.
func init() {
	// insight.go horrors and frenzy shake the Composure kept by the thought cabinet (traits.go)
	inComposureHit = func(g *Game, n int) {
		s := thSt(g)
		s.Comp = clampi(s.Comp-n, 0, maxi(s.CompMax, 1))
	}
	// bounty hunters (hunts.go) track the bounties crimes put on you (crime.go) and defaulted debts (banking_economy.go)
	bhPlayerBounty = func(g *Game, state int) int {
		return g.uwBounty(state) + g.nxBounty(state) + g.bkBounty(state) + gcCyberHeatBounty(g)
	}
	// delivered invitations (letters.go) become ceremonies you can attend (ceremonies.go)
	fwInvites = func(g *Game) []fwInv {
		var out []fwInv
		for _, iv := range g.lnInvites() {
			out = append(out, fwInv{From: iv.From, Burg: iv.Burg, Day: iv.Day, Kind: iv.Kind})
		}
		return out
	}
	nxRivalDossier = func(g *Game) []string { return g.rvDossier() } // rival section of the Hunters screen (nemesis.go)
}
