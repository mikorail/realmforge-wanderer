package engine

import (
	"fmt"
	"strings"
)

// Trans-Continental Banking, Loans & Commodity Arbitrage (prefix bk).
// Merchant banking houses issuing promissory notes and sovereign loans.
// Regional commodity price arbitrage between biomes.
// Shady underworld loan sharks and debt bounty enforcers.

// BankLoan represents an active debt agreement with a bank or loan shark.
type BankLoan struct {
	ID           string `json:"id"`
	BankName     string `json:"bn"`
	Principal    int    `json:"pr"`
	InterestRate int    `json:"ir"`
	DueDay       int    `json:"due"`
	IsUnderworld bool   `json:"uw"`
	Defaulted    bool   `json:"def"`
}

// BankingState holds ledger loans, notes issued, and debt bounties.
type BankingState struct {
	Loans           []BankLoan `json:"loans,omitempty"`
	NotesIssued     int        `json:"ni"`
	DefaultBounty   int        `json:"db"`
	EnforcersSent   int        `json:"es"`
	ArbitrageProfit int        `json:"ap"`
}

func bkSt(g *Game) *BankingState {
	return g.ext("bk", func() any {
		return &BankingState{}
	}).(*BankingState)
}

// IssuePromissoryNote exchanges heavy coin for a lightweight, bank-backed promissory note.
func (g *Game) IssuePromissoryNote(amount int) (*Item, bool) {
	if g.P == nil || amount <= 0 {
		return nil, false
	}
	fee := maxi(1, amount*2/100) // 2% teller fee
	totalRequired := amount + fee

	if g.P.Gold < totalRequired {
		g.Msg(fmt.Sprintf("You need %d gold (%d principal + %d fee) to issue this note.", totalRequired, amount, fee))
		return nil, false
	}

	g.P.Gold -= totalRequired
	st := bkSt(g)
	st.NotesIssued++

	note := &Item{
		ID:     g.newID(),
		Name:   fmt.Sprintf("Promissory Note for %d Crowns", amount),
		Glyph:  '?',
		F:      CWhite,
		Kind:   "promissory_note",
		Value:  amount,
		Weight: 0,
		Lore:   fmt.Sprintf("A certificate of deposit for %d gold crowns, honorably redeemable at any merchant bank across the realm.", amount),
	}
	g.P.Inv = append(g.P.Inv, note)
	g.Msg(fmt.Sprintf("The teller stamps an official promissory note for %d crowns (fee: %d gold).", amount, fee))
	return note, true
}

// RedeemPromissoryNote cashes in a promissory note for sovereign gold crowns.
func (g *Game) RedeemPromissoryNote(it *Item) (int, bool) {
	if it == nil || it.Kind != "promissory_note" {
		return 0, false
	}

	amount := it.Value
	g.P.Gold += amount

	// Remove note from inventory
	for i, invItem := range g.P.Inv {
		if invItem == it {
			g.P.Inv = append(g.P.Inv[:i], g.P.Inv[i+1:]...)
			break
		}
	}

	g.Msg(fmt.Sprintf("The banker counts out %d bright gold crowns in exchange for the promissory note.", amount))
	return amount, true
}

// TakeBankLoan secures a sovereign loan from a recognized merchant banking house.
func (g *Game) TakeBankLoan(bankName string, amount int, termDays int) bool {
	if amount <= 0 || termDays <= 0 {
		return false
	}
	st := bkSt(g)
	currentDay := g.Day()

	loan := BankLoan{
		ID:           fmt.Sprintf("loan_%d_%d", currentDay, len(st.Loans)+1),
		BankName:     bankName,
		Principal:    amount,
		InterestRate: 5, // 5% standard interest
		DueDay:       currentDay + termDays,
		IsUnderworld: false,
		Defaulted:    false,
	}

	st.Loans = append(st.Loans, loan)
	g.P.Gold += amount
	g.Msg(fmt.Sprintf("The %s approves a sovereign credit line of %d gold, due on Day %d (5%% interest).", bankName, amount, loan.DueDay))
	return true
}

// TakeUnderworldLoan accepts dangerous financing from an underworld loan shark.
func (g *Game) TakeUnderworldLoan(amount int) bool {
	if amount <= 0 {
		return false
	}
	st := bkSt(g)
	currentDay := g.Day()

	loan := BankLoan{
		ID:           fmt.Sprintf("shark_%d_%d", currentDay, len(st.Loans)+1),
		BankName:     "Red Hook Loan Syndicate",
		Principal:    amount,
		InterestRate: 25, // 25% extortionate interest
		DueDay:       currentDay + 14,
		IsUnderworld: true,
		Defaulted:    false,
	}

	st.Loans = append(st.Loans, loan)
	g.P.Gold += amount
	g.Msg(fmt.Sprintf("A scarred money-lender slips %d gold into your purse: 'Two weeks. 25%% interest. Do not make us send the boys.'", amount))
	return true
}

// RepayBankLoan settles an outstanding loan and its accrued interest.
func (g *Game) RepayBankLoan(loanID string) bool {
	st := bkSt(g)
	for i := range st.Loans {
		l := &st.Loans[i]
		if l.ID == loanID {
			totalOwed := l.Principal + (l.Principal * l.InterestRate / 100)
			if g.P.Gold < totalOwed {
				g.Msg(fmt.Sprintf("You need %d gold to satisfy this debt. You only have %d.", totalOwed, g.P.Gold))
				return false
			}

			g.P.Gold -= totalOwed
			if l.Defaulted {
				st.DefaultBounty = maxi(0, st.DefaultBounty-totalOwed)
			}
			g.Msg(fmt.Sprintf("You settle the debt with %s in full, paying %d crowns (principal + interest).", l.BankName, totalOwed))
			st.Loans = append(st.Loans[:i], st.Loans[i+1:]...)
			return true
		}
	}
	g.Msg("No matching loan record found in the ledger.")
	return false
}

// GetCommodityPrice calculates regional arbitrage prices for cross-continental trade.
func (g *Game) GetCommodityPrice(commodityID, biome string) int {
	b := strings.ToLower(biome)
	c := strings.ToLower(commodityID)

	switch c {
	case "silk":
		// Produced in temperate/grasslands; scarce in mountains/tundra
		if strings.Contains(b, "tundra") || strings.Contains(b, "mountain") || strings.Contains(b, "snow") {
			return 110 // High scarcity markup
		}
		if strings.Contains(b, "grass") || strings.Contains(b, "forest") || strings.Contains(b, "city") {
			return 40 // Local surplus price
		}
		return 75

	case "spices":
		// Produced in desert/savanna; scarce in cold north
		if strings.Contains(b, "desert") || strings.Contains(b, "savanna") {
			return 25 // Producer price
		}
		if strings.Contains(b, "tundra") || strings.Contains(b, "taiga") || strings.Contains(b, "ice") {
			return 95 // Extreme cold markup
		}
		return 60

	case "abyssal_salt":
		// Mined deep underground; prized on surface
		if strings.Contains(b, "cave") || strings.Contains(b, "deep") || strings.Contains(b, "dungeon") {
			return 30
		}
		return 85

	case "timber":
		// Abundant in forests; scarce in desert/plains
		if strings.Contains(b, "forest") || strings.Contains(b, "jungle") {
			return 15
		}
		if strings.Contains(b, "desert") || strings.Contains(b, "badlands") {
			return 60
		}
		return 35

	case "fine_vintage":
		// Produced in coastal hills; expensive on frontiers
		if strings.Contains(b, "coast") || strings.Contains(b, "hills") {
			return 35
		}
		if strings.Contains(b, "swamp") || strings.Contains(b, "frontier") {
			return 90
		}
		return 55

	case "bog_iron":
		// Dug from wetlands; needed in dry forge towns
		if strings.Contains(b, "swamp") || strings.Contains(b, "marsh") {
			return 20
		}
		if strings.Contains(b, "desert") || strings.Contains(b, "plains") {
			return 70
		}
		return 45
	}

	return 50
}

// CheckDebtDefault checks if any active loans have exceeded their due date.
func (g *Game) CheckDebtDefault() {
	st := bkSt(g)
	currentDay := g.Day()

	for i := range st.Loans {
		l := &st.Loans[i]
		if !l.Defaulted && currentDay > l.DueDay {
			l.Defaulted = true
			penalty := l.Principal + (l.Principal * l.InterestRate / 100)
			st.DefaultBounty += penalty
			g.Msg(fmt.Sprintf("DEFAULT NOTICE: Your loan from %s has defaulted! A bounty of %d crowns has been placed on your debt.", l.BankName, penalty))
			g.SpawnDebtEnforcers()
		}
	}
}

// SpawnDebtEnforcers summons ruthless bounty collectors to shake down the defaulting debtor.
func (g *Game) SpawnDebtEnforcers() []*Actor {
	if g.Cur == nil {
		return nil
	}
	zn := g.Cur
	st := bkSt(g)
	st.EnforcersSent++

	var enforcers []*Actor
	ex := clampi(g.P.X+3, 1, MW-2)
	ey := clampi(g.P.Y+1, 1, MH-2)

	chief := &Actor{
		ID:      g.newID(),
		Name:    "Bailiff Grimshaw",
		Proper:  true,
		Glyph:   'E',
		F:       CRed,
		X:       ex,
		Y:       ey,
		Tpl:     "debt_bailiff",
		Faction: "outlaws",
		Level:   8,
		HP:      55,
		MaxHP:   55,
		AVNat:   5,
		DVNat:   8,
		Speed:   100,
		AI:      "hostile",
		Boss:    true,
		Desc:    "A hulking debt enforcer clutching iron writs of foreclosure and a weighted truncheon.",
	}
	zn.Actors = append(zn.Actors, chief)
	enforcers = append(enforcers, chief)

	// Thug
	tx := clampi(ex-1, 1, MW-2)
	ty := clampi(ey+1, 1, MH-2)
	thug := &Actor{
		ID:      g.newID(),
		Name:    "Roughhouse Goon",
		Glyph:   'g',
		F:       CBrown,
		X:       tx,
		Y:       ty,
		Tpl:     "debt_thug",
		Faction: "outlaws",
		Level:   6,
		HP:      35,
		MaxHP:   35,
		AVNat:   3,
		DVNat:   9,
		Speed:   105,
		AI:      "hostile",
		Desc:    "A scarred muscle-for-hire hired by the syndicates to break bones.",
	}
	zn.Actors = append(zn.Actors, thug)
	enforcers = append(enforcers, thug)

	g.Msg("Heavy footsteps block your path. Debt collection enforcers step out from the shadows: 'Time to pay what you owe!'")
	return enforcers
}

// bkBounty returns the current outstanding default bounty for wiring into hunts and crime.
func (g *Game) bkBounty(state int) int {
	st := bkSt(g)
	return st.DefaultBounty
}

func init() {
	unLocalKeys = append(unLocalKeys, "bk")

	turnHooks = append(turnHooks, func(g *Game) {
		// Daily audit of loans
		if g.Turn%1440 == 0 {
			g.CheckDebtDefault()
		}
	})

	zoneHooks = append(zoneHooks, func(g *Game, zn *Zone, r *Rng) {
		if zn == nil {
			return
		}
		if strings.Contains(strings.ToLower(zn.Kind), "market") || strings.Contains(strings.ToLower(zn.Kind), "harbour") {
			// Market districts host a banking table
			x, y, ok := zn.FreeSpot(r, func(x, y int) bool { return zn.At(x, y) == TFloor || zn.At(x, y) == TCobble })
			if ok {
				zn.Set(x, y, TStall)
			}
		}
	})

	talkHooks = append(talkHooks, func(g *Game, a *Actor) []Option {
		if a == nil || (a.Role != "banker" && a.Role != "merchant" && a.Role != "usurer") {
			return nil
		}
		st := bkSt(g)
		var opts []Option

		opts = append(opts, Option{
			Text: "Convert 200 gold into a merchant promissory note (fee: 4 gold)",
			Do: func() {
				g.IssuePromissoryNote(200)
			},
		})

		// Check for unredeemed notes in inventory
		for _, it := range g.P.Inv {
			if it != nil && it.Kind == "promissory_note" {
				noteRef := it
				opts = append(opts, Option{
					Text: fmt.Sprintf("Redeem %s for full cash value", it.Name),
					Do: func() {
						g.RedeemPromissoryNote(noteRef)
					},
				})
				break
			}
		}

		if len(st.Loans) == 0 {
			opts = append(opts, Option{
				Text: "Apply for a 500 gold sovereign bank loan (5% interest, 30 days)",
				Do: func() {
					g.TakeBankLoan("Iron Vault Mercantile Bank", 500, 30)
				},
			})
			opts = append(opts, Option{
				Text: "Seek urgent underworld financing (400 gold, 25% interest, 14 days)",
				Do: func() {
					g.TakeUnderworldLoan(400)
				},
			})
		} else {
			for _, l := range st.Loans {
				loanID := l.ID
				bankName := l.BankName
				totalOwed := l.Principal + (l.Principal * l.InterestRate / 100)
				opts = append(opts, Option{
					Text: fmt.Sprintf("Repay outstanding debt to %s (%d gold)", bankName, totalOwed),
					Do: func() {
						g.RepayBankLoan(loanID)
					},
				})
			}
		}

		return opts
	})
}
