# Balance targets and headline numbers

Nothing here comes from a playtest. `go run ./cmd/balance [world.json]` prints every table below from the real engine code (Spawn, XPFor, gainXP, price, travel pace, deadline windows, fares, ambush odds). Exported accessors that mirror one expression live in `engine/balanceinfo.go`. Assumptions in the tool: gear tier = level/3 in every slot for both sides, key weapon skill = 1 + level/5, attribute points into the key attribute, 1v1, no crits or specials, builds: bruiser (warrior, STR 19 TOU 18), duelist (rogue, TOU 12), scholar (TOU 12). Powers, companions, sneak attacks and off-hand weapons are not modelled.

Units: a turn is a minute, a day is 1440 turns, a parasang is 160 km on the default map (about 5.3 days on grass, 4.4 on a road, 6.7 in forest, 13 in mountain forest).

## Targets

| Area | Target |
|---|---|
| Kills per level | 4 to 6 on-level kills (measured 3.5 at L1, 4.6 to 5.1 from L2 to L30) |
| 1v1 fight vs an on-level foe | lose about 20 to 40% of HP at the same level, a boss costs 60 to 100% |
| Full heal | about 30 h awake, 12 h in a bed, independent of level |
| Food | 1 ration a day at Endurance 0, 4 gold, so about 5 gold a day; water is free at wells |
| Paid work | gold per day of work about 2x the food bill |
| Deadline window | at least 1.3x the walk on an average biome (24 km/d), tight but possible in forest and rainforest |
| Road ambush | about 1 per 3 weeks of walking at Survival 0, rarer with Survival |
| Counter squads | at most one per 8 days, only with a grudge |

## Headline numbers (before the changes below unless marked)

- XP: need 52 at L1, 2482 at L10, 9457 at L20 (`15 x^3 / 2` steps). Kill XP is `10 + 5 L^2` (rat at L1 15, on-level at L10 510). Two tiers (10 levels) below pays 0.1 or 0. Flat XP rewards (guard trip 30, deadline 20+4L, incursion 60+10L) are 50 to 135% of a level at L1 and under 5% from L10.
- HP: bruiser 25 at L1, 94 at L10, 202 at L20, 342 at L30. TOU 12 builds were +1 HP a level (16, about 25, about 45); now floor 3 a level: 16, 49 at L10, 109 at L20, 193 at L30.
- Enemy HP (wolf at level): 17 at L3, 44 at L10, 85 at L20. Player HP grows about 2.4x faster than enemy HP, boss at L+3 is 76 to 112 HP.
- Fights (bruiser): average loss 20 to 30% at L5 to L10 (worst croc 135% at L5, crystalgolem 104% at L8). Duelist and scholar lose 90 to 270% at L5 to L10 and need 40 actions at L10 (see "left alone").
- Boss at L+3 vs bruiser: L1 to L5 you need 15 to 25 actions, it needs 2 to 4: effectively unwinnable without allies, tonics or ranged. L15 it is 5% loss, L20 36% (very noisy: armour is a 60% roll).
- Survival (Endurance 0): 720 hunger ticks a day. Ration was 400, now 600: 1.2 rations a day (4.8 gold), 1.6 drams of water. Hungry at 40 h, Famished 80 h, Thirsty 30 h. Start pack lasts 2.2 days of food (before), 3.3 (after); a L1 character with 55 gold buys about 11 days.
- Prices: weapon buy 17/77/177/317/498/718 for tier 0 to 5 (sell about 8%), body armour 12 to 488, heal tonic 35, priest full heal 10+3L (L10 40 gold). Average Loot() sell value 2.4 (tier 0) to 46 (tier 5).
- Income: quest 60+25L (85 at L1, 310 at L10), outlaw 3L gold, boss 40L. Caravan guard pay was `10+km/8` = 4.5 gold/day (below the food bill), now `10+3km/8` = 10.5 gold/day at 160 km. Trade ledger guard was 40+35d (9 gold/day), now 40+45d (10.9). Arbitrage (profit per run of 5 parasangs, about 27 days): grain 92, iron 164, spice 1231, silk 1693, relics 2771 (capital 85 to 2559), so trade dominates all other income once capital exists; limited by price volume (+1.2% per unit).
- Deadlines: window = (parasangs x km/22 x 1.35 + min + 1.5) days (was km/28). At 4 parasangs on grass 2.2x the walk, forest 1.7x, rainforest 1.1x, mountain forest 0.9x (still short), glacier 0.7x. Caravan kind (x0.75) is 1.5x on grass, 0.8x in rainforest. Reward at L1: plague 206, siege 258, hostage 155 (now 258), flood 116, caravan 51, festival 64. Hostage ransom is 120+18L average, so at L10 the old reward (290) barely covered it.
- Trade contracts: allowance was `9 + 4d` days, now `9 + 6d`, walking is 5.3 d per parasang, so the old one failed from about 8 parasangs.
- Travel: road ambush per crossing at Survival 0: 24% (road) to 40% (cap, 13 days of mountain). Survival 3 cuts it to 6 to 18%. Per day about 6%.
- Counter squads: 10% per zone entry, 30% per camp rest; with the old 4 to 7 day cooldown about 1.6 squads per 10 days while a grudge lasts; now 8 to 11 days.
- Healing: natural 1 HP per 28 to 48 min (period set by TOU only). Half heal took 16 h at L5 but 54 h at L20 and 80 h at L30 for the bruiser. Now ticks heal `1 + MaxHP/80`: 14 to 21 h awake and 6 to 9 h in a bed at every level.
- Aging (human 72, starts at about 24): old after about 30 years = 3400 game days at pace 1 (847 at x4, 282 at x12). At pace 1 aging never shows in a normal run.

## Changes (file, old to new)

See the end of the session report; summary:

- `engine/travel.go` tvPass hunger/thirst tick: `Turn%(2+End) != 0` to `== 0` (road, rail, caravan and air travel were using the inverted pattern: 3 to 5 times too hungry with Endurance).
- `engine/items.go` MakeFood Power 400 to 600.
- `engine/game.go` and `engine/starts.go` level-up HP floor `maxi(1, 3+TOU)` to `maxi(3, ...)`.
- `engine/game.go` + `engine/fatigue.go` natural and sleep healing step `1` to `1 + MaxHP/80` (new `regenStep`).
- `engine/deadlines.go` assumed walking speed `28` to `22` km/day (named `dlWalkKmPerDay`), hostage reward base 120 to 200.
- `engine/tradeways.go` contract allowance 4 to 6 days per parasang (`tqDaysPerParasang`), guard fee `40+35d` to `40+45d` (`tqGuardFee`).
- `engine/caravans.go` guard pay `10+km/8` to `10+3km/8` (`cvGuardPay`), fare unchanged (`cvFare`).
- `engine/counters.go` squad cooldown 4 to 8 days (`ctCooldownDays`).
- `engine/creatures.go` Spawn: HP per level above the template 4 to 5, natural attack penetration per level above the template `up/4` to `up/3`.

## Left alone (judged wrong or doubtful)

- Low-STR melee builds hit an armour wall: weapon PV is 2 to 5 plus the STR modifier (negative below 16), foes reach AV 7 to 17 by L15 to L30, so duelist and scholar need 15 actions at L8 and thousands at L20 to kill an on-level foe. Needs a design call (floor the STR term for the player, or scale weapon PV with skill).
- Late-game foes are template levels 1 to 11 scaled up, so from L15 enemy offence lags player HP and PV; the new HP and PV scaling is a nudge, not a fix. Bosses at L+3 are 10:1 deadly at L1 to L5 and trivial at L15.
- Overland difficulty is `1 + distance/5` parasangs, so an area level costs 25 days of walking; depth (+2 a level) is the real difficulty ramp.
- `challenges.go` "hungry" mod adds a hunger tick when `Turn%(2+End) != 0`, so Endurance makes it worse, not better.
- Priest healing costs 10+3L for a full heal (40 gold at L10): cheaper than 4 days of food, so it removes natural healing as a cost in any town.
- Trade arbitrage outpaces all other income by 10x to 100x once capital exists.
- Water is priced at about 2 gold a dram but is free at wells, so the printed food bill (8 gold) overstates real cost; money buys food, not water.
- Mountain forest, wetland, glacier deadlines stay shorter than the walk; the route is not a straight line in practice.
- Aging does nothing at pace 1.
- Numbers after the changes were measured with creatures.go at `up/2` PV scaling; it was then lowered to `up/3` unmeasured.
