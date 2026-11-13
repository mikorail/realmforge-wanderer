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
- Companion banter: home town, own/enemy realm, temples, wonders, ruins, weather, feasts, meals and night camps, level-ups, grumbles, companion pairs (kin, rival realms), kill and near-death one-liners; per-companion and global cooldowns.
- Companion personal quests: avenge a kinsman, recover a family heirloom, make peace with an estranged relative, settle a debt at home; loyalty, perk and deed.
- Companions use active powers in combat (augments and natural): heal, flames/freeze/lightning/bolt on foes clear of allies, drain, sprint/phase to flee, force bubble, stun.
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
- Social world (`socialworld.go`): Realmforge social orders, genre forces, reforms, persecutions, famine ethics, intermarriage and prestige exported and used in towns and talk: unfree workers and constructs, notices, bound strata, reform movements with pamphlet / captive quests, refugee camps outside haven towns, hostile guards and markups for the persecuted, pickets, corp offices, vat clinics, commons gardens, smog, fallout and ley nodes, mixed households.

- Multiple planets and rifts (`planets.go`), slain beasts and broods (`slain.go`), villains with hideouts (`villains.go`), murim sects, clans and cults (`murim.go`), true-scale overland travel and 5x5 great cities (`travel.go`, `city.go`).
- Taming and beast mounts (`taming.go`), court politics (`politics.go`), world-scale endgame with five endings (`endgame.go`), auto-aim and calling prerequisites (`autoaim.go`), sprite mode per species, calling and creature (`actorsprites.ts`).
- Starting scenarios (`starts.go`), personal nemesis (`nemesis.go`), rival adventurers (`rival.go`), crafting from beasts (`beastcraft.go`).
- Hidden and buried things: secret doors (`secrets.go`), locks, chests and traps (`locks.go`), real shipwrecks and drowned cities (`wrecks.go`), barrows, bunkers and pirate hoards (`burials.go`).
- Life systems: clutter and collections (`clutter.go`), herbalism and alchemy (`alchemy.go`), stealth and assassination (`stealth.go`), trade routes and smuggling (`tradeways.go`), nests, living ruins and ghost towns (`nests.go`), family, titles, songs and duels (`family.go`), table games, arena betting and busking (`games.go`), spell research, pacts and masters (`arcane.go`), balloons, caravans, ferries and route closures (`journeys.go`), hall of fame and daily seeds (`halloffame.go`).

- Wave wiring complete: rival adventurers racing to buried hoards (`buDigSites`, `buRivalLoot`) and endgame culling rivals (`rvCull`); merchants valuing trade goods in `price()` and honest merchants refusing banned goods (`uwNoSale`); in-zone cliff launch for gliders (`jyCliffBump`, `jyZoneGlide`); footprint and blood trail tiles on the map (`ssTrailAt`); night-sight reading `alnight`; researched arcane powers for companions and the sandbox (`ar_<id>`, `arSandboxGrant`); predator nest defense range raised to 10 tiles (5 when sneaking); `fmHeir` wired to legacy runs (`fmContinueAsHeir`).
- Tactical combat and throwables (`tactics.go`): called shots at limbs (`head`, `arm`, `leg`, `wing`, `eye`), flanking, high ground, cover and weather ranged modifiers, special ammo (`bodkin`, `broadhead`, `fire`, `frost`, `poison`, `silver`, `emp`), tactical throwables (`net`, `bolas`, `caltrops`, `smokebomb`, `flashbang`), and squad morale (routs and boss rally cries).
- Quality-of-life and exploration (`qol.go`): auto-explore (`AutoExplore`), configurable auto-pickup rules (`qlAutoPickup`), message-log search and category filter (`FilterMsgs`), and self-writing Markdown adventure diary export (`ExportDiary`).

- Tribes and colored/biome subrace variations (`tribes.go`, `src/actorsprites.ts`): colored/subrace variations for every species (Crimson Bloodfang Orcs, Azure Frost-Tusk Orcs, Redcap Butcher Goblins, Sun-Gold/Silver-Moon/Emerald/Obsidian Elves, Red-Beard/Rime-Beard Dwarves, Crimson/Azure/Emerald/Golden Saurians & Lizardfolk, Moss-Back/Glacier Trolls, Woad-Painted/Red-Ochre Humans, etc.) with console colors, hex skin/scale/fur/war-paint tags, totems, procedural tribal encampments, chieftains, shamans, parleys, war-paint buffs, and rites of strength.
- Smarter combat AI and weather in combat (`combat_ai.go`): archers kiting to keep distance, packs and tribal warriors flanking opposite allies, wounded foes retreating to bandage and rally, rain dousing fire, fog concealing archers, and wind carrying scent to downwind predators.
- Planet traits, rift storms, star-map and story arcs (`world_story.go`, `city_life.go`): planet traits (`low_gravity`, `toxic_air`, `tidally_locked`, `dead_ruin`), seasonal rift storms, interactive Star-Map screen (`starmap`), off-world goods selling dear, historical library books revealing dungeons, twisted/outdated rumours, historical dreams on rest, 3-verse start-of-run prophecies feeding endgame endings, companion personal arcs and lethal-blow sacrifice, NPC epilogues, detective murder mysteries and courtroom accusations, city shift jobs, and city night-life (night guilds, smugglers, ghosts).
- Deep murim, taming, crafting and presentation (`murim_deep.go`, `taming_deep.go`, `craft_explore.go`, `src/actorsprites.ts`, `src/weathersfx.ts`): hidden masters disguised as beggars, sect grudges and avengers, martial lineage passed to heirs, closed-door seclusion breakthroughs, forbidden demonic arts and qi-deviation, breeding tamed animals, teaching tricks (`fetch`, `guard`, `hunt`), pack animals with saddlebags, wildlife animal reputation, workbench scrap inventions, item enchanting and cursed items, sport fishing and free-diving, cartography survey maps, HUD quest compass arrow, ASCII photo mode, sprite animation (idle/walk/hit flash), and procedural WebAudio combat/UI sound effects.

- Unique architecture and monumental gates (engine/architecture.go, engine/terrain.go, src/sprites.ts): 11 culturally distinct wall materials (TWoodWall, TBrickWall, TStoneWall, TMarbleWall, TObsidianWall, TBoneWall, TAdobeWall, TBambooWall, TCoralWall, TMetalWall, THedgeWall), authentic non-square floor plans (yurts/rotundas, L-shaped manors, courtyard riads/siheyuan, octagonal dwarven keeps, partitioned multi-room homes, apsidal longhouses, colonnaded pavilions), and monumental dual-towered gatehouses with bastion crenellations, archways (TGateArch), portcullises (TGateClosed), and latticed windows (TWindow).
- Living world & public life (engine/living_world.go): packed inns and taverns (innkeepers, servers, tavern bards, dice gamblers, buying rounds for the tavern), live plaza spectacles (gallows executions with heralds and pardons/rescues, street/field honor duels with clashing blades and wagering/mediation), and actively working citizens (field farmers tending crop rows, blacksmiths at anvils, street hawkers, guard patrol pairs, forest woodcutters, wildland hunters/trappers, miners, and fishers).

## Planned / ideas
- Music listening pass in a real browser for the garden styles (drift / pastoral / desert): tune note density, the length of drift silences and instrument balance by ear.

### Ideas, not started
**Genre signature systems** (each genre gets one rule only it has)
- Cyberpunk: hacking terminals, doors, turrets, cameras and ads, with a heat meter that brings corporate security.
- Wasteland: radiation zones, decontamination, salvage runs, water purifiers as survival currency.
- Biopunk: a breeding vat that grows companions, organs and crops from samples with unpredictable mutations.
- Solarpunk: gardens and energy: tend crops, wind and sun power for machines, seed trading.
- Steampunk and dieselpunk: boiler and fuel management for vehicles and automatons you build.
- Fantasy and mythic: gods that answer prayer with real effects, omens read from the sky.
- Twilight: a decay clock where everything worsens unless you restore ruins and relics.

**Decisions and events**
- Dilemma events (short choice cards at crossroads with consequences that stack, in the style of Four Seas), oaths and vows with a bonus and a lasting enforceable penalty, a karma ledger that shapes how the world answers.

**Tools for players**
- World editor-lite (tweak a species, name a culture, place a landmark, re-export), challenge modifiers at creation (no healing, permanent injuries, one companion, double prices, no fast travel), a run recap page that tells the life as a short story with a map, a gentle scripted tutorial first hour.

**Depth in existing systems**
- Permanent injuries and scars (healers, prosthetics, augments fix them), sleep and fatigue with inns, tents and dreams, noise and smell as stealth inputs, time-sensitive quests with real deadlines, rival factions offering the same prize.

**Combat, world, murim, taming, crafting** (from earlier brainstorms)
- Called shots, flanking and cover, special ammo, smarter foes, weather in combat.
- Planet traits, rift storms, a star-map screen, goods from other planets, libraries that reveal real events, wrong rumours, dreams, a prophecy, NPC epilogues, companion arcs, legacy runs (`fmHeir` exports the heir), detective mysteries, city jobs, night life.
- Hidden masters, grudges, martial lineage, seclusion, forbidden arts; breeding, tricks, pack animals, animal reputation.
- Inventions, enchanting and cursed items, fishing and diving, cartography, auto-travel and auto-explore, quest arrow, log search, diary, photo mode; sprite animation, more tile variety, combat sound effects.

**Talking to people**
- Languages: Realmforge already generates a language per culture; signs, books and speech stay unreadable until you learn some of it (from locals, teachers, books, a translator companion or gadget).
- NPCs who remember what you did to them (gifts, favours, insults) and spread it by gossip; moods and personalities that change daily talk options.

**Gear that grows**
- Named weapons and armour that level up from your deeds (a title, a history line, a new power); item provenance for any item found; reforging two items into one.

**Look and feel**
- Weather and light in sprite mode (rain and snow streaks, torch glow, shadows, day-night tint); hover tooltips naming what a sprite is; footstep sounds by terrain and a short voice blip per species.

**Playing it**
- Key rebinding, right-click action menus, gamepad support; replay and sharing a run as a keystroke list (the engine is deterministic); a run map image showing your path, key deeds and graves.

**World events you take part in**
- Disasters as scenes (pull people out of a quake, hold a bridge, fight a fire), pilgrimages and processions to join, monster migrations and territory fights to watch, join or exploit.

**Aging and time**
- Your character ages: species lifespans, prime years, old-age penalties, wisdom bonuses; ties into heirs and legacy runs.

**Big: people and stories**
- Full NPC lives: a biography per notable person built from Realmforge's sim data (birth, family, career, rises and falls, rivals, loves, losses, a secret, a goal, a fear), ordinary townsfolk with a short bio (trade, a family fact, a gripe), a People page per person in the journal, and talk topics whose answers depend on trust.
- Knowledge per person: each NPC knows only some facts (who they saw, where they have been, what they heard); you question and cross-check them like a detective, so gossip is a real information system.
- A relationship web (who loves, hates, owes or works for whom) where your actions ripple (kill a rival and the baker owes you, the rival's family wants revenge).
- **Heavy life simulation (planned, not started; classic agent simulation, no AI-model calls).** Plan, in three files that share one data model (`lvPerson` with stable key, traits, goal, fear, secret, family, friends, rivals, life events, memories of you, opinion, known facts; state `g.ext("lv")`, planet-local):
  - `lives.go`: who people are. Realmforge notables plus about a dozen generated townsfolk per town with names from the culture, biography built from the sim's real events in their lifetime, family tree, personality, goals and fears; binds the actors of a town zone to the living people present; talk topics (their life, what they want, family, other people); memory of what you did to them; a People screen.
  - `lifesim.go`: what they do. A day-clock simulation with a per-call budget (full detail near the player and for notables, coarse yearly rates elsewhere, caught up when you arrive): work, travel between towns, courtship and marriage, births and growing up, careers, crime and punishment, feuds, migration, goals that succeed or fail, illness, war casualties, death and inheritance, shops changing hands; news lines for what happens.
  - `knowledge.go`: what they know. Facts (sightings, events, rumours that distort as they spread, secrets, locations), who knows what, questioning with trust and checks, cross-checking conflicting accounts, case notes, tie-in to crimes and the sim's events.

- Generated campaigns: multi-act plots built from the world's history (a conspiracy for a throne, a lost heir, a curse from an old sin, a heist of a legendary vault) with clues scattered across towns and dungeons and several plots per world; full questline ladders for each faction, order and sect.
- Cross-genre incursions: a cyberpunk raider or a fantasy knight arrives through a rift in the wrong world with strange gear.
- Content packs: a data-file layer to define new items, creatures, quests, NPCs and dialogue without recompiling, shared as packs.
- Ghost runs: share a past life as a small file that another player meets in their world (builds on the hall of fame legends).
- Dynasty play: play one family across centuries, choose an heir at death, your deeds become history (ties legacy runs, aging, marriage, hall of fame).

**Under the hood**
- Smaller saves (drop unvisited zone data), faster zone generation on slow machines, shareable world cards.

**Wiring left from the life-systems wave**
- Rivals racing you to buried hoards (`buDigSites`, `buRivalLoot`) and the endgame culling rivals (`rvCull`); merchants valuing trade goods in `price()` and honest merchants refusing banned goods; the in-zone cliff launch for gliders; footprint tiles on the map (`ssTrailAt`); night-sight reading `alnight`; researched powers for companions and the sandbox; predator nest range; `fmHeir` wired to a legacy run.

### Dropped on purpose
- Player settlements, accessibility work, a playtest bot and extra test suites, ruling a realm with decrees, founding a faith or order.


## Brainstormed Future Ideas & Expansions (Completed)

### 1. Architectural & Urban Evolution (`engine/urban_evolve.go`)
- [x] Multi-story rooftops and subterranean cellar networks beneath every building with cat burglar skylights (`TSkylight`) and chimney descents (`TChimney`).
- [x] Destructible city gates, barricades (`TBarricade`), and siege ballistas (`TBallista`) during wartime and rebellions.
- [x] Seasonal urban dressing: lantern festivals with glowing paper lanterns (`TLantern`), autumn harvest bunting (`TBunting`), and winter snow drifts (`TSnowDrift`) along roof cornices.
- [x] Player-commissioned guildhalls, custom estate construction, and commemorative victory monuments (`TMonument`) in city plazas via town elder dialogue hooks.

### 2. Living Society & Emergent Drama (`engine/society_emergent.go`)
- [x] Dynamic crime scenes with chalk outlines, town guard murder investigations, and public courtroom trials with jury verdicts.
- [x] Royal wedding processions, funeral corteges, and triumphant army marches parade down city high streets.
- [x] Labor guild strikes with picket lines, bread-line riots at civic granaries, and spontaneous philosophical tavern debates that boil over into brawls.
- [x] Travelling theatre troupes staging satiric plays and epic ballads recounting the player's own past exploits and deeds.
- [x] Seasonal livestock drives with drovers/sheepdogs and regional harvest trade fairs bringing exotic merchant caravans to town squares.

### 3. Deep Ecology & Frontier Life (`engine/ecology_frontier.go`)
- [x] Dynamic lumberjack camps that physically fell forest tiles and float log rafts downriver to sawmills.
- [x] Nomadic shepherd flocks with sheepdogs migrating between high mountain pastures in summer and lowland valleys in winter.
- [x] Seasonal apex predator migrations and monster herd breeding seasons with maternal alpha broods.
- [x] Wind-driven wild grassfires and volcanic cinder storms that scorch biomes and leave fertile ash beds (`TAshBed`) yielding rare wild blooms.

### 4. Cultural & Subrace Depth (`engine/culture_depth.go`)
- [x] Inter-tribal moots, grand martial kurultais, and ancestral memory shrines that unlock lost subrace combat stances (`mountain_giant`, `desert_viper`, `steppe_hawk`, `deep_mole`).
- [x] Fully mobile nomadic tent-cities (`TTent`) and wagon caravans that pack up and migrate across the world map with the seasons.
- [x] Subterranean mushroom-forest silkworm farms (`TSilkwormRack`) and bioluminescent coral cultivation domes (`TCoralDome`) on the sea floor.

### 5. Interactive Tactical Environment (`engine/tactical_env.go`)
- [x] Dynamic tavern brawls: flipping tables for cover (`TCrate`/`TSeat` +4 DV), swinging from chandeliers (`TChandelier` flying drop-kick), throwing tankards, and kicking foes into braziers (`TBrazier` burning damage).
- [x] Environmental combat hazards: cutting scaffold ropes to crush enemies (`TScaffold` collapse), shooting beehives (`TBeehive` enraged swarm), and puncturing wine/oil barrels (`TBarrel` slick/flammable pools).

### 6. High-Seas, Deep Caverns & Dynastic Faith Systems
- [x] **Naval Sailing, Flagships & High-Seas Piracy** (`engine/naval.go`): Sailing caravels, war longships, and submarine bathyspheres; mounted swivel cannon grapeshot; gangplank boarding; and multi-tile Abyssal Kraken encounters.
- [x] **Dynastic Bloodline Curses, Heirs & Pretender Quests** (`engine/dynasties_rogue.go`): Ancestral traits ('Dragon-Heart', 'Red Miasma', 'Giant Stature', 'Seer Lunacy'), deed-accruing family heirlooms, generational heir continuation, and royal pretender citadel infiltration quests.
- [x] **Subterranean Verticality, Spore Canopies & Acoustic Stealth** (`engine/subterranean.go`): Giant mushroom canopy leaping and drop ambushes, spore blindness clouds, vibration acoustic stealth, and blind abyssal predators tracking footstep sound.
- [x] **Thaumaturgical Smog, Spell Backfires & Crystalline Mutations** (`engine/thaumaturgy.go`): Ambient arcane smog, spell research backfires (temporal echoes, localized gravity inversion, warp cascades), mana sickness, and crystalline skin mutations attracting mana leeches.
- [x] **Living Faiths: Divine Oaths, Inquisitors & Shrines** (`engine/saints_faith.go`): Sacred vows (Oath of the Iron Fist, Oath of the Sunken Eye), taboo violation apostasy, inquisitor hit squads, and living saint shrine prayers.
- [x] **Trans-Continental Banking, Loans & Commodity Arbitrage** (`engine/banking_economy.go`): Weightless promissory notes, bank credit and loan sharks, regional commodity arbitrage across biomes, debt default bounties, and debt enforcers.
