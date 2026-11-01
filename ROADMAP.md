# Realmforge Wanderer roadmap

## Done
- World loading: Realmforge "Wanderer World" export (`engine/world.go` WorldData), bundled sample worlds, IndexedDB saves.
- Character creation: species, callings, point-buy, genre-named powers and defects, origins.
- Zones: 80×25 zones, 3×3 per parasang, strata, deep layers, sky isles, sea floor.
- Towns and great cities: districts from real economic sectors, walls, gates, landmarks.
- Dungeons and megastructures: ruins, catacombs, mines, warrens, outlaw hideouts, towers, Spire of Trials, Bottomless Abyss.
- Combat: d20 to-hit, penetration, dismemberment, bleeding, elements, fire.
- AI: shadowcast FOV, light and darkvision, BFS pursuit.
- Factions and reputation from the sim's realms, faiths, relations and wars.
- Talk: rituals, secrets from history, gossip from real events, trade, quests, ferries, lore books.
- Powers (incl. swimming, diving, climbing, spells, time dilation and more).
- Rarity tiers and categories, boss drops.
- Inventory: weight, backpacks, encumbrance, tabs, compare.
- Attire: regional clothes per sphere, tailors, transmog.
- Skill checks and inner voices; social strata, households, kin.
- Weather and seasons: schedule per region, effects outdoors, lightning, overlays.
- Cooking: ingredients, meals with tags, cookbook, local dishes, recipe swaps.
- Companions: recruitment, orders, gear, loyalty, XP, departure.
- Body augments: 20 augments skinned per genre, installers in cities, capacity.
- Companion augments: installers fit and remove augments on companions (own capacity); they keep them through save/load and after parting ways.
- Companion food: give a dish (meal tags + `fed`) or a raw ingredient; shared cooked meals give a small loyalty bonus (once every 6 hours).
- Adaptive chiptune soundtrack: mood, genre, biome, danger, hp, day/night, depth.
- Soundtrack follows weather and seasons: rain patter, thunder on lightning strikes, sparse high snow, fog drones, sand/ash noise wash, slower heat; seasons shift key and mode.

- Local calendars and feasts from Realmforge (month names, town festivals).
- Seasonal wild herbs (spring greens, summer berries, autumn mushrooms/nuts, winter roots/lichen), merchants' seasonal specials, named festival dishes sold from 3 days before a feast; cookbook marks seasonal and festival dishes (`seasons2.go`).
- Weather-aware NPC schedules: townsfolk shelter under roofs in storms, blizzards, sandstorms, acid/fallout rain and walk home after; outdoor stalls close, guards keep their posts, birds and pack beasts make for cover.
- Ambient weather sound effects separate from the music bus (`src/weathersfx.ts`, own volume): rain and droplets, storm rumbles and thunder, wind, sand, ash crackle, birds, crickets, surf.
- Plug-in hook registry (`engine/hooks.go`), first-time tips, touch controls, ambient music style.
- Garden soundtrack (`src/garden.ts`): drift (Eno / C418 phasing loops, sparse piano, silences), pastoral (Stardew fingerpicking and motif melodies by season), desert (Qud modal drones, synth phrases, analog arp, hand drum); "Adaptive" picks by place and is the new default.
- Sandbox mode (F2), player homes and stash (H, F3), joinable wars with ranks (W), workbench crafting with bits, schematics, mods and infusions (T), cults/orders/guilds (O), codex journal (bestiary, atlas, library, people).
- Realmforge link: chronicle export/import, live "Play in Wanderer", time passes in the sim, outlaw cap.
- Qud design study: `docs/QUD_REFERENCE.md`.

- Qud parity (wave 2): mutation levels, genotypes True Kin / Mutant, morphotypes, glimmer and hunters (`mutations.go`, `glimmer.go`, `V`); artifact complexity and stepwise examination (`examine.go`); liquids, pools, pour/fill/mix, water drams as currency, survival clocks (`liquids.go`, `survival.go`, `trade2.go`, `P`); 19 skill trees with 27 maneuvers, Qud status table, XP curve (`skilltrees.go`, `maneuvers.go`, `statuses.go`, `leveling.go`, `K`); legendary creatures, ritual v2 with reputation bands, oathbreaking (`legendary.go`, `ritual2.go`); main quest spine from real history, 5 village quest kinds, zone tiers, special biomes and terrain hazards, classic/roleplay/wander modes (`questchain.go`, `tiers.go`, `biomes.go`, `modes.go`, `Q`).

- Wave 3: two new genres from Realmforge, wasteland (post-apocalypse) and twilight (fallen golden age), skinned across every genre table, with sample worlds Zotbozdiv and Sirati; multi-currency realms with money changers (`currency.go`); living world: caravans, travellers, armies, plague, invasion, desertion, bandits, regional prices, captives you can only free, news screen `N` (`worldsim.go`); underworld: laws, contraband, gate searches, stealing `X`, bounties and jail, fences, black market, fight pit with betting, ledger `U` (`crime.go`, `underworld.go`); mod saloons and mutation houses (`parlors.go`); treasure maps `B`/digging `D` and bounty boards with capture (`hunts.go`); thought cabinet `J`, traits, Luck/Composure/Fame/Perception (`thoughts.go`, `traits.go`); Insight, hearts, stat gates `U`-adjacent (`insight.go`, `gates.go`); cultural soundtrack with synthesised instruments, scales and feels per sphere, combat and feast layers (`src/music.ts`, `instruments.ts`, `scales.ts`). Cross-feature links in `engine/wiring.go`.

## Planned / ideas
- Export social orders, genre forces and persecution from Realmforge and use them in towns and talk.
- Companion banter and personal quests drawn from their kin and home town.
- Companions using granted active augment powers (heal, flames, phase) in combat.
- Music listening pass in a real browser for the garden styles (drift / pastoral / desert): tune note density, the length of drift silences and instrument balance by ear.
