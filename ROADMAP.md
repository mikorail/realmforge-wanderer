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

## In progress
- Local calendars and feasts from Realmforge (month names, town festivals).

## Planned / ideas
- Export social orders, genre forces and persecution from Realmforge and use them in towns and talk.
- Companion banter and personal quests drawn from their kin and home town.
- Companions using granted active augment powers (heal, flames, phase) in combat.
- Seasonal ingredients and festival dishes tied to the calendar.
- Weather-aware NPC schedules (shelter indoors during storms).
- Ambient weather sound effects separate from the music bus, with their own volume setting.
