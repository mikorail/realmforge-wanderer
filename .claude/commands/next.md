---
description: Pick up the remaining Wanderer work (health, taming, politics, endgame, merge) on this machine
argument-hint: health | taming | politics | endgame | merge
---

You are working locally in the Realmforge Wanderer repo (Go engine in `engine/`, Vite/Svelte shell in `src/`; Realmforge is the sibling repo `../realmforge`). Read `CLAUDE.md`, `AGENTS.md` and `ROADMAP.md` first.

Task: **$ARGUMENTS**. If that is empty, list the options below with one line each and ask which one to do.

## Setup (local machine)
- Start from the latest development branch: `git fetch origin && git switch -c <task>-local origin/claude/dreamy-cerf-kcp93i` (it has sailing on top of `master`; if that branch is gone, use `origin/master`).
- System node may be too old: npm scripts go through `scripts/run-with-nvm.sh`. Go 1.23+ is needed for the engine and `npm run wasm`.
- New features go in their own files and register through `engine/hooks.go` (zone/turn/talk/kill/enter/side hooks, `g.ext` saved state). Check the taken-key list in `hooks.go` before claiming a key.
- Never draw `g.R` in world-reading or flavour code that would shift existing rng streams; use side rngs / hashes seeded from the world seed and zone keys.
- Checks before every commit: `gofmt -l engine`, `go vet ./engine ./cmd/...`, `go test ./engine/`, `go run ./cmd/fuzz public/worlds/ember-crown.json 10000`; after shell or wasm-bridge changes also `npm run wasm` and `npm run smoke`. Commit after each finished piece.

## Options
### health
Health pass. Write a goal-driven playtest bot `cmd/playtest` (`go run ./cmd/playtest world.json [days] [seed]`): travel to towns, trade, eat, cook, recruit a companion, fight wildlife and bandits, descend a dungeon, take a train if a station exists, rest, save/load. Report deaths and causes, gold/price curve, hunger/thirst failures, NPCs and actors per zone, zone generation time (p50/p95/max), save size over time, items lying in zones, messages per hour, panics. Fix what it reveals. Known bugs to fix too:
- sprite mode (`src/sprites.ts`) draws magenta ♣/♠ flora (cherry blossom, neon birch) as mushrooms;
- surface `TMushroom` patches from `procassets.go` outside fungal biomes get underground species (glowcap) in `flora.go`;
- walking into `TShallow` drinks instead of wading (`Interact=true`), so oasis and crater-lake rings act as walls;
- terrain is harsh on the mountainous sample planets (escarpments on 10-15% of parasangs, lots of land above 3,500 m thin air): tune `tgCliffDrop` / `tgThinAir` and neighbours in `tglandforms.go` / `tgelev.go`, report before/after shares;
- sailing gaps: merchant ships use a fixed menu instead of the trade screen; a stolen ship can't be recovered.

### taming
New file `engine/taming.go` (prefix `tm`). Tame non-hostile or wounded wildlife from the bestiary (`zwildlife.go`, `bsSpecies`) by feeding the right food for its diet over several attempts (Survival/Persuasion checks; predators, big and extinct species harder). Tamed animals become beast companions (`companions.go`: orders, loyalty from feeding, natural attacks, levels, player-given names, permanent death). Tamed rideable species (horse, camel, yak, elephant, reindeer, llama, ostrich, mammoth, Megaloceros, Triceratops, …) become mounts through the transport mount system (`transport.go` `trMounts`, `trPaceMul`): stats from size/speed/habitat, flying mounts with stamina, swimming mounts for rivers only (ships cover the sea). Ostlers can sell bestiary breeds. Tests in `engine/taming_test.go`.

### politics
New files (prefixes `po`, `fa`). Court offices and political factions from the world data (ideology, social order, sim people), audiences, office tasks (taxes, treaties, conspiracies, war contingents via `war.go`), backing factions, intrigue (bribes, blackmail, assassination contracts), succession crises, coups and revolutions tied to `socialworld.go` movements, genre-skinned titles, a rare path to becoming ruler with decrees. Founding your own faith or order: preach for followers per town, tenets from Realmforge theologies/doctrines, holy sites, rites, schisms, persecution, recognition; extend `orders.go` for player-founded orders. Deeds for everything. Screens on free keys. Tests in `engine/politics_test.go` and `engine/faith_test.go`.

### endgame
New files (prefix `eg`). After a mid-game threshold a world-scale threat rises, chosen deterministically from the world (awakening titan from the Realmforge monsters, rift invasion via `planets.go`, great plague from `hazards*.go`, world war via `war.go`, Abyss cult, or a genre one: rogue AI, gene plague, the Last Winter, the bomb, the dust). It spreads on the `worldsim.go` day clock (occupied parasangs, refugees, news, prices, dialogue). The player unites factions, gathers relics and megastructure keys, and fights a multi-phase finale with allies; several endings recorded as deeds; the world continues afterwards. Spine chapter on `Q`, world-map overlay, sidebar warning, a `MusicState` doom value. Tests in `engine/endgame_test.go`.

### merge
Merge the finished development branch into `master`: check `git log origin/master..origin/claude/dreamy-cerf-kcp93i`, run all checks above plus `npm run build` and `npm run smoke`, then open a PR to `master` (or merge locally if the owner prefers) and push.

When done, update `CLAUDE.md` (engine map) and `ROADMAP.md`, and summarise what changed and what the checks showed.
