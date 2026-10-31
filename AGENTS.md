# AGENTS.md: Realmforge Wanderer

Caves-of-Qud-style ASCII roguelike (Go engine compiled to WASM, Vite + Svelte 5 shell) played across worlds exported from Realmforge (github.com/mikorail/realmforge). Read `CLAUDE.md` first (engine map, every system and its hooks). Open work: `ROADMAP.md`. Qud mechanics and the ranked gap list: `docs/QUD_REFERENCE.md`. Ready-made prompts for open packages: `docs/AGENT_PROMPTS.md`.

## Commands
- Setup on a fresh machine or cloud session: `scripts/setup.sh`.
- `go vet ./engine && go test ./engine` (scenario tests run against `public/worlds/*.json`).
- `go run ./cmd/fuzz public/worlds/ember-crown.json 20000` after engine changes (random keys, zone warps, save/load round trips; must print `ok`).
- `go run ./cmd/peek public/worlds/ember-crown.json talk show` prints rendered screens as text (handy without a browser).
- `npm run wasm` rebuilds `public/wasm/qud.wasm` (commit it: the site is static and loads it directly). `npm run dev` serves on 5195. Node 22+ is needed for Vite 8; scripts go through `scripts/run-with-nvm.sh`.

## Rules that will bite you
- **Determinism:** zone generation must be reproducible from the world seed and zone key. Use the `r *Rng` passed in or `NewRng(seed + ":purpose:" + key)` side streams; never `g.R` inside generation and never math/rand.
- **Plug in, don't edit the core:** new features go in their own file and register hooks from `init()` (`engine/hooks.go`). Save state with `g.ext(key, mk)`, not new `Game` fields. Check the `keyMain` switch before claiming a main key.
- **Name prefixes:** every package-level helper in a feature file gets a short prefix (`mu`, `lq`, `sk`, `lg`, `qc`, `cx`, `lm`, `aug`…) so parallel agents never collide in package `engine`.
- **World schema:** `engine/world.go` `WorldData` mirrors Realmforge `src/lib/adventure/export.ts`. Change both together; keep new fields `omitempty`/optional so old world files and saves still load.
- **Saves:** `Game` is JSON; keep field tags stable. Old saves must keep loading (test with a save made before your change).
- **Render:** the screen is a 112×34 grid; map 80×25 top-left, sidebar from x=81, log under the map. Use `Wrap`/`TextB` markup `{{R|text}}`; colours are palette indices.
- **No AI model API calls** anywhere in this project.
- **Copy style:** short, concrete UI text; keep commit messages short and factual.

## Working with several agents at once
1. Split work by files. Each agent creates its own `engine/<feature>.go` + `_test.go` and registers hooks; shared files (`game.go`, `input.go`, `render.go`, `talk.go`, `combat.go`, `zone.go`) are edited only by the coordinator, who wires the one-line hooks agents report.
2. If the package fails to build because of another agent's half-written file, wait and retry; never edit it.
3. Agents report: functions added, exact hook lines, test results. The coordinator wires, runs the full tests and the fuzzer, rebuilds the wasm, then commits.
4. Branch per task in cloud sessions (`claude/<topic>`), open a PR to `master`.
