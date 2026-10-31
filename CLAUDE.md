# Realmforge Wanderer

Caves-of-Qud-style ASCII roguelike played across worlds made in Realmforge (`../realmforge`). Separate app: Realmforge only exports world files (Export dialog → "Wanderer World", or `npx tsx scripts/adventure-world.ts` there); this game loads them. No LLM/AI-model API calls.

## Stack
- Go 1.23, no deps (module `wanderer`): `engine/` is the whole game and draws a 112×34 character grid; `cmd/wasm` exposes `window.qud` (`palette`, `init(worldJSON)` → chargen options, `newGame(spec)`, `load(save)`, `key(k)`, `save()`, `meta()`, `deeds()`).
- Web shell: Vite 8 + Svelte 5 runes (no SvelteKit), `fflate`, Share Tech Mono. `src/App.svelte` (title: continue runs, bundled sample worlds, load a Realmforge .json), `CharGen.svelte` (species → calling → point-buy → genre-named powers/defects → origin; rules mirror `engine/chargen.go`), `Play.svelte` (canvas grid, keys incl. numpad, click-to-walk, autosave every 150 keys), `qud.ts` (wasm loader), `store.ts` (IndexedDB `realmforge-wanderer`: meta/save/world per world seed, deflated).

## Run
- System node is too old: npm scripts go through `scripts/run-with-nvm.sh`. Dev: `scripts/dev.sh` (port **5195**), preview config "realmforge-wanderer" in `claude-projects/.claude/launch.json`.
- `npm run wasm` rebuilds `public/wasm/qud.wasm` + `wasm_exec.js` after any Go change (reload the page; the engine is cached on `window.qud`). `npm run build` = wasm + vite build.
- `public/worlds/` holds bundled sample worlds + `index.json` (regenerate from Realmforge: `npx tsx scripts/adventure-world.ts <years> <genre> <seed> <template>`).
- Tests: `go test ./engine/` (scenario tests against the bundled worlds: combat, death, ritual, quests, trade, sky falls, breath, powers, stairs, travel, save/load, every species × calling).
- Dev tools: `go run ./cmd/fuzz world.json [steps]` (random keys, zone warps, save/load round trips: run after engine changes); `go run ./cmd/peek world.json talk show key:c show down show` prints screens as text.

## Engine map (`engine/`)
`world.go` WorldData schema (must match Realmforge `src/lib/adventure/export.ts`) · `genre.go` per-genre `Skin` (callings, power names, materials, gear, ritual, currency) · `zone.go` 80×25 zones, 3×3 per parasang; z>0 strata and Realmforge deep layers, z=−1 sky isles, sea floor; towns with the sim's notables, ruins with chronicle inscriptions, crypts of dead rulers holding their relics, wonders, sim-monster lairs · `terrain.go` · `creatures.go` · `items.go` · `entity.go` (body plans, limbs) · `combat.go` (d20+AGI vs DV, up to 3 penetration rolls PV vs AV chaining, dismember/bleed/stun, elements, fire) · `ai.go` (shadowcast FOV, light/darkvision, BFS pursuit) · `faction.go` (rep per realm/faith/creature faction from home realm relations and wars; NPC generation) · `talk.go` (ritual → rep + secret from history, gossip from real events, trade, quests, ferries, lore books) · `powers.go` · `input.go` · `render.go` · `game.go` (turn/energy loop, survival, world map, save/load, deeds).
