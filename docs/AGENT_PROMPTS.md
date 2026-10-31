# Ready prompts for open work packages

Each prompt is self-contained for one agent (local subagent or a Claude Code cloud session). Before starting, check `ROADMAP.md`: a package marked "in progress" is already being built elsewhere. Every agent follows `AGENTS.md` (own files only, hooks from `init()`, name prefix, report hook lines) and reads `CLAUDE.md` plus its section of `docs/QUD_REFERENCE.md`.

Common finish for every package: `gofmt -w` your files, `go vet ./engine`, `go test ./engine -run <Prefix> -v`, then report the functions added, the exact one-line hooks for shared files, and test results. Do not edit shared files (`game.go`, `input.go`, `render.go`, `talk.go`, `combat.go`, `zone.go`, `chargen.go` unless listed) and do not commit unless you are the coordinator.

---

## WP1 Mutations, genotypes, glimmer (prefix `mu`)
Files: `engine/mutations.go`, `engine/glimmer.go`, `engine/mutations_test.go`; may also edit `engine/chargen.go` and `src/CharGen.svelte`.
Reference: QUD_REFERENCE 1.1, 1.2, 1.3, 1.15 (glimmer) and the WP1 sketch.
- Mutation levels: spend MP (`Game.MP`) to raise an owned mutation; level cap from character level; rapid advancement at the reference's levels. Expose `func (g *Game) mutLevel(id string) int` (other packages call it; do not define `powerLevel`).
- Classify existing `PowerDefs` as physical or mental. Morphotypes: Chimera (physical only, chance of an extra limb on level-up), Esper (mental only, more glimmer).
- Glimmer = sum of mental mutation levels; above thresholds psychic hunters (genre-skinned) appear via `enterHooks`/`zoneHooks`.
- Genotypes (`CharSpec.Genotype`, `Game.Genotype` already exist): True Kin start with no mutations, +2 attribute points, genre-skinned castes instead of callings (body augments in `augments.go` are their counterpart); Mutant = current flow.
- `CharOptions` exposes genotypes, morphotypes, castes and game modes (`classic|roleplay|wander`, `CharSpec.GameMode`); `CharGen.svelte` gets a first "Genotype" step (+ morphotype for mutants) and a "Mode" select on Origin.
- Tests: MP spend within cap, rapid advancement, True Kin without mutations, glimmer hunters, CharOptions JSON. Also `npx vite build`.

## WP2 Artifact complexity and examination (prefix `ex`)
Files: `engine/examine.go`, `engine/examine_test.go`. (Cybernetics are already covered by `augments.go`.)
Reference: QUD_REFERENCE 1.12.
- Items get a complexity from tier, rarity and category; `func (g *Game) exExamine(it *Item) (ok bool, msg string)` rolls Tinkering + INT against it, with breakage or a small explosion on bad failures of high-complexity artifacts; partial identification steps (shape → function → name) shown in `Describe` via `exDescribe(it) []string`.
- Report the hook lines for `powers.go` `examine()` and `items.go` `Describe`.

## WP3 Liquids, survival, water trade (prefix `lq`)
Files: `engine/liquids.go`, `engine/survival.go`, `engine/trade2.go`, `engine/liquids_test.go`.
Reference: QUD_REFERENCE 1.9, 1.11 and the WP3 sketch. Coordinate with `cooking.go` (`slowHunger`, meals) and `weather.go` (heat thirst, cold hunger): call or respect them, do not duplicate.
- Liquid table (fresh, salt, blood, oil, acid, slime, honey, wine, lava, genre liquids) with drink/splash/stand effects; containers hold drams of one liquid; pools via `Zone.Liq` ids ≥5 (report a render hook for colours).
- Pour / fill / mix screen (`mainKeys["P"]`), impure water from mixing.
- Fresh-water drams as a second currency: `lqDrams()`, `lqPay(n)`; Ego-based `lqPriceFactor(buy bool)` for `price()`.
- `lqSurvivalTick()` with the reference's hunger/thirst states, to replace the counters in `turn()`; Survival skill forages drinkable water and tells safe from brackish pools.

## WP4 Skill trees, maneuvers, statuses, leveling (prefix `sk`)
Files: `engine/skilltrees.go`, `engine/maneuvers.go`, `engine/statuses.go`, `engine/leveling.go`, `engine/skilltrees_test.go`.
Reference: QUD_REFERENCE 1.2, 1.5, 1.6, 1.7 and the WP4 sketch.
- Each skill becomes a tree of 3–6 named powers bought with SP (rank = powers owned, capped 3 for old checks), stored in `g.ext("skills")`; screen `mainKeys["K"]`.
- Maneuvers as activatable abilities (Lunge, Cleave, Dismember, Charge, Bludgeon, Jab, Hobble, Sure Shot, Tumble, Jump, Shake It Off, Berate, Intimidate, Proselytize, Butchery, Trailblazer, Repair, Trash Divining…): `skAbilities() []Ability` (merged into `Abilities()`), `skUse(id, tx, ty)`.
- Status table with the reference's numbers (prone, dazed, shaken, terrified, beguiled, exhausted… plus the existing ones): `skStatusEffect(a, name)` for `statusTick`, `skStatusDV(a)`, `skStatusToHit(a)`.
- XP curve `XPForLevel(l)` and `skOnLevel()` (SP/MP/AP per level by genotype).

## WP5 Legendary creatures and the full water ritual (prefix `lg`)
Files: `engine/legendary.go`, `engine/ritual2.go`, `engine/legendary_test.go`. (Companions are already in `companions.go`; use its API, e.g. its recruit/convert helpers.)
Reference: QUD_REFERENCE 1.10, legendary creatures and lairs, WP5 sketch.
- Named legendary versions of ordinary creatures per biome and tier (`zoneHooks`), boosted stats, guaranteed rare+ drop (`bossDrop`), lair notes, deeds when slain.
- Water ritual v2: reputation bands (Hated…Revered) and what each unlocks; share or buy secrets with currency or drams; learn a skill, recipe or power from a favourite faction; ask about locations; per-faction ritual counts — as `talkHooks` beside the existing ritual.

## WP6 Main quest spine, village quests, tiers, biomes, game modes (prefix `qc`)
Files: `engine/questchain.go`, `engine/tiers.go`, `engine/biomes.go`, `engine/modes.go`, `engine/questchain_test.go`.
Reference: QUD_REFERENCE 1.14, game modes, WP6 sketch. Coordinate biome hazards with `weather.go` (it already handles storms, sandstorms, cold and heat: add only terrain hazards such as spores, crystal resonance, lava heat, deep cold without warm clothing).
- A 5–7 chapter main arc from the world's real history (a chronicle relic, a dead ruler's lost heir, the Spire of Trials / Bottomless Abyss late, a sim legend as a boss), started by the home elder (`talkHooks`), tracked via enter/kill hooks; `qcChapter()` for the sidebar.
- Five more village quest templates: escort, investigate an inscription, broker peace between feuding households (`society.go` House/Kin), rescue a missing person from a dungeon, recover stolen goods from an outlaw hideout.
- `qcTier(wx, wy, z)` 1–8.
- Modes (`Game.GameMode`): classic (permadeath), roleplay (`qcCheckpoint()` save taken on entering a town, `qcOnDeath()` offers restore), wander (no kill XP via `qcKillXP`, creatures start indifferent, discovery and quest XP ×2).

## From the roadmap (not started)
- Export social orders, genre forces and persecution from Realmforge and use them in towns and talk.
- Companion banter and personal quests drawn from their kin and home town.
- Companions using granted active augment powers in combat.
- Seasonal ingredients and festival dishes tied to the calendar.
- Weather-aware NPC schedules (shelter indoors during storms).
- Ambient weather sound effects on their own bus with a volume setting.
