# Handoff: remaining work for another agent (agy / gemini)

State: master at 6dd1ecc (waves 1 to 5 done, 5 commits not pushed). Read `CLAUDE.md` first (engine map, plug-in hooks, prefixes). Each task below is a ready prompt. Run them one at a time or two in parallel on different files.

## Rules (apply to every task)
- Go 1.23 engine in `engine/`, no dependencies, module `wanderer`. Web shell is Svelte 5 in `src/`.
- Put new features in NEW files and register from `init()` (see `engine/hooks.go`). Edit shared files only for one-line hooks. Report every shared edit.
- Saved state through `g.ext(key, mk)`; every map and pointer must be rebuilt by `mk` (see `extFillNil`). Planet-local keys go into `unLocalKeys` from your `init()`.
- Generation uses side rngs only (`NewRng(seed+":xx:"+...)`), never `g.R`. `g.R` is only for play-time rolls. New zone hooks must add things only, never change existing generation.
- Pick a prefix not in use: `grep -ho 'ext("[a-z0-9]*"' engine/*.go | sort -u`. Used so far include jb rm sa wf fq rs fv cx wk hn sq mg dz dr lb ht sy.
- No new single-letter main keys (all taken). Use `paEntries` palette entries and talk options.
- No AI/LLM API calls anywhere. Text comes from templates, world data and culture packs.
- No tests or test suites. Build once at the end: `go build ./engine/ && go vet ./engine/`. Then `bash scripts/run-with-nvm.sh npm run wasm` and `go run ./cmd/fuzz public/worlds/iron-lantern.json 4000`.
- Use they/them for people of unknown gender. Keep wording plain.
- Do not commit or push unless asked.
- Never open a browser or play audio unless told. If a browser is opened, close the tab afterwards.

## Task 1: religions in play, books and lore, ruins (wave 6A)
New files `engine/faithplay.go` (prefix `fp`), `engine/lorelibrary.go` (prefix `ll`), `engine/ancientlayers.go` (prefix `al2`, check it is free).
1. Shrines with different blessings per generated faith (read `ReligionData` and `Lore`), offerings, pilgrimages to named holy sites, heresy trials and conversions that change standing with faith and realm. Audit `rituals.go`, `orders.go`, `visions.go`, `realmstory.go` first and do not duplicate them.
2. A lore library: readable in-world books and scrolls built from real events, with a collection log screen. Audit `literature.go` (`GenBook`) and `codex.go` and reuse them.
3. Ruins and ancient layers: pre-collapse sites holding lore fragments that, once collected, reveal one real piece of the world's history (use `WorldData.Events`). Link to `legendlore.go` leads read-only.
Each gets a screen (mode name = prefix) and a palette entry.

## Task 2: legacy mode and endgame (wave 6B)
New files `engine/legacyplay.go` (prefix `lp`), `engine/finalthreat.go` (prefix `ft`).
1. Legacy: when the player dies and has a descendant (see `family.go` `fmHeir`, `aging.go`, `halloffame.go`), offer to continue as that heir. Inherit part of the gear and reputation, and the old character's deeds become local legends told by NPCs and on gravestones.
2. Endgame: a world-ending threat generated from the world (read `endgame.go`, `endgame_play.go`, `deadlines.go`, `politics.go`), with a final quest chain and a new-game-plus option (harder world, kept legacy items). Audit those files first.

## Task 3: cleanup (small, mostly one-liners)
- Wire the nil rumour sinks into `engine/rumourmill.go`: `saRumourSink`, `fqRumourSink`, `dzRumourSink`, `mgRumourSink`, `sqRumourSink`, `htRumourSink`. Read `rmRum` and the `rmAdd` pattern first.
- Add `dzCheckMod(kind)` to `checkMod` in `engine/checks.go`.
- Poached goods: add `|| it.Extra["hnp"] > 0` to `uwIllegal` in `engine/crime.go`.
- Sprite roles `huntsman` and `warden`: add to `roleColor` in `engine/faction.go` and `ROLES` in `src/actorsprites.ts` (plus a pixel outfit in `src/pixelart/people.ts`).
- Heist guard shifts should read `engine/routines.go` instead of a hash.
- Hunting contest at festivals: needs a public hook in `engine/festivals.go`.

## Task 4: play and balance (needs a human or a browser)
Waves 1 to 5 were built and fuzzed but never played. Open each new screen and flow once: contract board, rumours, social moves, wayfarers, faction work, realm affairs, festival grounds on a feast day, mercenary hall, workshop, hunting, a siege near a war, wildlife, afflictions, dungeon set pieces, legendary arms, heists, lodgings. Then run `go run ./cmd/balance` and check rewards and prices. All numbers are guesses.
