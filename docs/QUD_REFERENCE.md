# Caves of Qud design reference for Realmforge Wanderer

Sources: wiki.cavesofqud.com pages Character creation family (Attributes, Callings, Castes, Mutated human), Mutations, Cybernetics, Skills, Combat, Effects, Liquid, Factions, Tinkering, Item Mods, Cooking, Artifact, Equipment, Quests, Sultan, Villages, World Map, Trading, Companions, Hunger, Thirst, Experience, Glimmer, Lairs, Legendary creature, Body parts. Numbers are as the wiki gives them (patch ~2.0.20x); everything is paraphrased. Pages that 404 or were stubs are flagged **[unverified]** and filled from general knowledge: Game modes, Lairs (stub), Legendary (stub), dismemberment details, Light, Critical hits (the Combat page has none).

Wanderer facts used below come from `CLAUDE.md`, `engine/chargen.go` (`PowerDefs`, `callingDefs`, `SkillIDs`), `engine/powers2.go`, `engine/combat.go`, `engine/entity.go`, `engine/hooks.go`, `engine/game.go` (`gainXP`, turn loop), `engine/talk.go` (`ritual`). Already being built elsewhere and therefore OUT OF SCOPE for the packages: realmforge link/chronicle, web polish/mobile/music, sandbox mode (`sandbox.go` exists), player home/base, wars, crafting depth (recipes/enchanting/mods, and by extension cooking recipes), cults and orders, codex (bestiary/atlas/books).

---------------------------------------------------------------------
# PART 1. MECHANICS REFERENCE
---------------------------------------------------------------------

## 1.1 Genotypes and character creation

- **Two genotypes.** *True Kin* (pure-strain humans from the arcologies: no mutations, may use cybernetics, 70 skill points/level, start license tier 2 for implants, a random starting implant they may decline for +1 Toughness and tier 0, "castes" instead of callings) and *Mutated Human* (mutations, no cybernetics/becoming nooks, 50 skill points/level, Putus Templar start hostile, generally worse starting gear and smaller tonic benefits).
- **Subtypes.** True Kin pick a *caste* (12 listed, grouped by arcology: Horticulturist, Priest of All Suns, Priest of All Moons, Syzygyrior; Artifex, Consul, Praetorian, Eunuch; Child of the Hearth, Child of the Wheel, Child of the Deep, Fuming God-Child). Each gives +2 to +4 to one or two attributes, a starting skill bundle, a ~+15 heat or cold resistance (by arcology) and sometimes +2 bleeding saves. Mutants pick a *calling* (Apostle, Arconaut, Greybeard, Gunslinger, Marauder, Pilgrim, Nomad, Scholar, Tinker, Warden, Water Merchant, Watervine Farmer): +2 (or +3/-1) to one attribute plus 2-4 starting skills. Examples: Marauder +2 STR, Axe/Charge/Butchery; Gunslinger +2 AGI, Pistol; Tinker +2 INT, Tinkering/Repair/Tinker I; Greybeard +3 WIL -1 STR, Cudgel/Berate/Calloused.
- **Morphotypes (mutants).** *Chimera* = physical mutations only, may sprout extra limbs. *Esper* = mental mutations only, draws more glimmer/hunters. Third option is none.
- **Attributes (6).** Strength, Agility, Toughness, Willpower, Intelligence, Ego. Modifier = floor((score - 16) / 2) (score 10 = -3, 16 = 0, 21 = +2).
  - STR: melee penetration PV, carry capacity (15 x STR lb), resisting forced movement.
  - AGI: DV (dodge) and to-hit; gates pistol/bow/short-blade/acrobatics skills.
  - TOU: HP, HP per level, regen, resists poison/illness.
  - WIL: MA (mental armor) and ability cooldown reduction (5 % of base per point above 16, capped 80 % at 32).
  - INT: +4 skill points per level per point above 10 (negative below); crafting, cooking, examining artifacts; gates tinker tiers (15/19/23/29 INT).
  - EGO: trade prices, Persuasion, recruiting, level of mental mutations.
- **Point buy.** Base 10 in each. Each +1 costs 1 point up to 18, then 2 per point. Caps at 24 before caste/calling bonuses; cumulative: 18 = 8 pts, 20 = 12, 22 = 16, 24 = 20. Mutants have 44 extra points; mutation pool at creation is 12 points (mutation costs 1-5 each, one defect allowed at -1..-4 which refunds points).
- **Presets** exist (pre-built builds), plus *game modes* **[unverified]**: Classic (permadeath, a single save slot that is deleted on death), Roleplay (save anywhere, no permadeath), Wander (no permadeath plus easier/exploration focus; in Qud 2.x Wander also trims the main-quest pressure) and a Daily seeded run.

## 1.2 Leveling

- **XP to reach level x:** total = 15 x^3 + 100 (L2 = 220, L5 = 1975, L10 = 15 100, L20 = 120 100, L40 = 960 100). No hard cap on the page (table stops at 40).
- **Kill XP:** each creature carries a base XP. Multiplier by *tier gap* where tier = floor(level / 5): killer tier <= victim tier = x1, one tier above = x0.5, two above = x0.1, three or more above = 0. Final-blow attacker gets it, shared with party/followers.
- **Other XP:** quest steps and quests, first entering a non-starting village (250 x zone tier), discovering lairs, turning in Resheph secrets/books to specific NPCs.
- **Per level:** HP gain from Toughness; skill points (70/50 + 4 x (INT - 10)); 1 mutation point (mutants); attribute points on a 3-level cadence: at level 3n, odd n gives 1 spendable attribute point, even n gives +1 to all attributes; mutation level cap = floor(level/2) + 1, hard cap 10. Rapid Advancement: at level 5 and every 10 after, pick a physical mutation for a permanent +3 levels outside the cap.
- **Legendary XP** = level x 250.

## 1.3 Mutations

- **Buying:** new mutation = 4 points (chosen from 3 random offers at later levels); +1 level = 1 point. Per-mutation creation costs run 1-5 (listed below). Defects *grant* points and cannot be levelled or bought later; they also arrive from gamma moths/brain brine.
- **Ego** adds temporary levels to mental mutations; **Glimmer** = sum of mental mutation levels (+Ego levels) and at 20+ summons psychic hunters (see 1.15).
- **Physical (sample, cost: effect):**
  1. Adrenal Control 4: Quickness boost 20 rounds, 200 cd.
  2. Beak 1: +1 Ego, pecks.
  3. Burrowing Claws 3: 4 hits dig through a wall.
  4. Carapace 3: +AV by level, -2 DV, no body armor.
  5. Corrosive Gas Generation 3: acid cloud, 40 cd.
  6. Double-muscled 3: +STR by level, daze on hit.
  7. Electrical Generation 4: stored charge, discharge 1d4 per 1000.
  8. Flaming Ray 4: 9-tile fire ray, 10 cd.
  9. Freezing Ray 5: 9-tile frost ray, 20 cd.
  10. Heightened Quickness 3: +Quickness by level.
  11. Horns 4: 20 % gore on melee, +AV.
  12. Multiple Arms 4: per extra arm % chance of an extra attack.
  13. Multiple Legs 5: move speed, carry capacity.
  14. Night Vision 1.
  15. Phasing 4: pass through solids for N rounds.
  16. Quills 3: retaliatory quills, +AV.
  17. Regeneration 4: faster healing, regrow limbs.
  18. Stinger (paralyzing venom) 4: 20 % paralyze on hit.
  19. Thick Fur 1: +5 heat and cold resist.
  20. Two-hearted 3: +TOU, longer sprint.
  21. Wings 4: flight, travel speed.
- **Mental (sample):**
  22. Beguiling 5: charm a creature (bonus HP), 50 cd.
  23. Clairvoyance 2: see a nearby area, 100 cd.
  24. Confusion 4: cone, 75 cd.
  25. Cryokinesis 4 / Pyrokinesis 4: chill/heat 3x3 over 3 rounds, 50 cd.
  26. Disintegration 3: 7x7 damage, 3-round exhaustion, 75 cd.
  27. Domination 5: control an adjacent creature N rounds, 75 cd (you inhabit it; manage its gear/points).
  28. Ego Projection 4: boost a physical attribute by 2 x Ego bonus, 200 cd.
  29. Force Wall 3: 9 tiles of force, N rounds.
  30. Kindle 1: ignite a tile at range 12, 50 cd.
  31. Mental Mirror 2: reflect failed mental attacks.
  32. Precognition 4: rewind to N rounds ago, 500 cd.
  33. Stunning Force 3: push+stun 7x7, 50 cd.
  34. Sunder Mind 4: psychic duel up to 10 rounds then bonus damage, 80 cd.
  35. Teleportation 5: random teleport within an uncertainty radius.
  36. Time Dilation 4: slow all within 9 for 15 rounds, 150 cd.
- **Defects (sample):** Brittle Bones -4 (+50 % blunt/fall dmg), Myopic -3 (sight 10), Carnivorous -2 (no satiation from non-meat), Narcolepsy -3, Evil Twin -3 (rival hunts you), Socially Repugnant -2 (-50 rep with every faction).
- Cooldown scales with Willpower; many effects scale with mutation level (duration, damage, radius).

## 1.4 Cybernetics (True Kin only)

- **License tier** = a budget. Installed implants cost *license points*; free = tier - sum(costs). True Kin start at tier 2. Raising tier is bought at a **becoming nook** with credit wedges: 1 each early, 2 from tier 9, 3 from 17, 4 from 25 (tier 2 -> 24 costs about 46). Declined starting implant: tier 0, +1 TOU.
- **Becoming nooks** (found in ruins/arcologies): install, uninstall (refunds points; some implants destroy on removal), and raise tier. Racks beside nooks hold stock. A temporary tier buff (Kesil Face) can leave a deficit that blocks new installs.
- **Slots:** Head, Face, Arm, Hands, Feet, Body, Back; an implant needs a free matching slot, some fit several.
- **Sample implants (cost, slot):** Tibular Hydrojets 1 Feet; Cherubic Visage 1 Face; Dermal Insulation 2 Body/Head/Back; Reactive Cranial Plating 2 Head; Pneumatic Pistons 2 Feet; Carbide Hand Bones 2 Hands; Dermal Plating 3 Body/Head/Back; Bionic Arm 3 Arm; Reactive Trauma Plate 4 Body; Magnetic Core 4 Body; Motorized Treads 4 Feet; Biodynamic Power Plant 6 Body; Gun Rack 6 Back; Cathedra 12 Back. (The wiki page gives costs/slots/weight but not effects; in-game they are: armor, speed, unarmed damage, extra arm, power source, storage...)
- Mutants cannot take cybernetics, True Kin cannot mutate. This is the core build fork.

## 1.5 Skills

- Currency: skill points (SP). Each tree has an unlock cost (buying the tree) and powers inside it with their own cost; many 0-cost powers are granted by the unlock. Water-ritual teachers sell skills for reputation equal to the SP cost.
- **Trees (unlock cost; contents):**
  - *Acrobatics* 75: DV boosts, Jump 100, Tumble 150.
  - *Axe* 50: proficiency 0, Charging Strike 100, Cleave / Dismember / Decapitate / Hook and Drag 150, Berserk! 300.
  - *Bow and Rifle* 100: Draw a Bead, Steady Hands 0; fire skills 150 and 200; Ultra Fire 300.
  - *Cooking and Gathering* 100: Meal Preparation 0, Harvestry, Butchery, Spicer 50, Carbide Chef 100.
  - *Cudgel* 50: proficiency, Charging Strike, Backswing 100; Bludgeon, Conk, Slam 150; Demolish 300.
  - *Customs and Folklore* 150: Tactful, Trash Divining.
  - *Endurance* 100: Shake It Off, Swimming, Poison Tolerance, Juicer 150, Weathered, Calloused 200, Longstrider 250.
  - *Heavy Weapon* 100: Strapping Shoulders, Tank 150, Sweep 200.
  - *Long Blade* 100: proficiency, stance upgrades 100, Lunge / Dueling Stance / Swipe 200, En Garde! 300.
  - *Multiweapon Fighting* 150: Flurry, proficiency 0; Expertise 150; Mastery 200.
  - *Persuasion* 100: Menacing Stare 0, Berate/Snake Oiler 150, Intimidate 200, Inspiring Presence 250, Proselytize 300.
  - *Physic* 50: Staunch Wounds 0, Amputate 50, Nostrums/Apothecary 100.
  - *Pistol* 100: Steady Hand; Sling and Run 50; Akimbo, Weak Spotter 150; Disarming Shot 100; Dead Shot 200; Empty the Clips 300; Fastest Gun 400.
  - *Self-discipline* 150: Meditate 0, Fasting Way 50, Iron Mind/Lionheart 100, Conatus/Mind over Body 150.
  - *Shield* 100: Block 0, Shield Slam 100, Swift/Deft/Staggering Block and Shield Wall 200.
  - *Short Blade* 50: expertise; Jab, Pointed Circle 100; Bloodletter, Hobble 150; Rejoinder, Shank 250.
  - *Single Weapon Fighting* 100: Opportune Attacks, Weapon Expertise 150, Penetrating Strikes 200, Weapon Mastery 300.
  - *Tactics* 100: Hurdle, Deft Throwing, Kickback 50, Charge 150, Juke 200.
  - *Tinkering* 100: Gadget Inspector 0 (INT 15), Disassemble, Scavenger, Deploy Turret, Repair (17), Tinker I (19), Reverse Engineer (25) 100; mines 50; Tinker II 200 (23), Tinker III 300 (29).
  - *Wayfaring* 100: Mind's Compass, Make Camp 0; terrain "Wilderness Lore" 25-50 each (reduces travel hazards/costs per biome).
  - *Nonlinearity*: Tomorrowful (precognition duration x2).
- **Shape:** powers are mostly *activated abilities with cooldowns* or *passive modifiers on attack*, grouped so each weapon class has a signature maneuver (cleave, lunge, jab, slam, charge).

## 1.6 Combat

- **To-hit:** 1d20 + AGI mod + weapon hit bonus must beat target DV. On-hit effects (fire, daze, stun) fire on any hit. Stuck / overburdened targets are DV -10. Some attacks (hook-and-drag, latch) always hit.
- **DV** from AGI, armor DV, shields, stance/skills, status. **AV** reduces and gates damage.
- **Penetration:** roll three exploding d10-style dice ("-1..8"), each succeeds on target number 5 - PV + AV. All fail = no damage; at least one succeeds = one penetration; all three succeed = another penetration and re-roll with PV-2 (strength weapons only), repeating while all three succeed. Damage = weapon dice per penetration, summed. PV = AV gives about 40 % per roll; PV - AV >= 6 is certain. So heavy armor makes weak weapons do nothing and strong weapons do many multiples.
- **Resistances** are % per element. **Asleep**: -12 DV and +4 penetration for attackers. Dazed, stunned, paralyzed, prone etc. alter DV (below).
- **Shields:** without skill 25 % block chance, once/round; skill raises it and adds AV on block; slam attack.
- **Multiple weapons / criticals:** not covered on the wiki Combat page **[unverified]**. In-game: each wielded weapon (and each natural/extra-arm weapon) swings per round, off-hand has a to-hit penalty reduced by Multiweapon skills; criticals exist as a natural-20 style bonus that Long Blade/Short Blade skills hook. Wanderer already approximates both.

## 1.7 Status effects (numbers)

Bleeding (damage/turn = amount; saves by TOU), Poisoned (increment/2 per turn, no regen, healing halved, becomes *ill* after), Stunned (cannot act, DV 0), Paralyzed (DV 0), Frozen (no physical actions), Burning (dmg/turn), Confused (random action; -level DV and MA), Asleep (see above, wakes dazed on damage), Dazed (-4 AGI -4 INT -10 move speed), Prone (-6 AGI -5 DV, -80 move speed), Hobbled (-50 % speed), Shaken (-level DV), Disoriented (-level DV and MA), Shamed (-4 DV/to-hit/WIL/EGO, -10 % quickness), Sluggish, Frenzied (+10 quickness), War trance (+5 quickness +6 WIL), Syphoned (life drain per turn), Choking on ash/gas/spores (2/turn), **Beguiled** and **Proselytized** (recruitment effects: the creature becomes a follower; Beguile ignores creatures with mental shields, Proselytize works on non-hostile, has follower limits), Glotrot (flat bad prices), Metabolizing (food buffs), Inspired (cooking).

## 1.8 Body parts, equipment, dismemberment

- A creature is a list of body parts each with a slot type: Head, Face, Body, Back, Arm, Hands, Feet, Hand (weapon/shield), Missile Weapon, Thrown Weapon, Floating Nearby, plus Tail/Fin/Tread/Roots etc.
- Human default: Head, Face, Back, Body, 2 Arms, 2 Hands, Feet, 2 missile slots, thrown slot, floating slot. A hand slot takes anything but armors give no bonus there.
- Multiple parts of one type average their AV/DV; other bonuses (resists) stack fully.
- **Dismemberment:** a severed limb drops its items and leaves empty slots until regrown (regeneration/ubernostrum). Axe tree (Dismember/Decapitate) causes it deliberately; cleave also cracks armor (temporary AV loss). **[details partly unverified]**
- Item mods: up to 3; Qud rarity weights Common 2000 / Uncommon 800 / Rare 210 / Rare2 50 / Rare3 3 (crafting package owns mods).

## 1.9 Liquids and thirst

- Liquids are measured in **drams**; waterskin = 64. 4 drams weigh 1 lb. Fresh water is *the* currency: 1 dram = $1 value.
- **Pools** on tiles: < 200 drams puddle (walk), 200-1999 pool (wade), ~2000+ deep (swim). Pools under 200 may evaporate (volume-weighted evaporativity). Touching liquids mix; names stack adjectives ("inky sugary bloody water").
- **Containers:** pour amounts, fill matching containers, drink. Acid eats organic containers, lava burns non-glass, damaged containers leak onto the holder.
- **Liquid personalities:** water (cleans, steams), lava (hot, light), acid (damage), slime (confusion when drunk), oil (flammable, slippery), honey (sticky, 2 $/dram), asphalt (very sticky), sap/wax (sticky), gel/ink (slippery/stains), brown sludge (rusts), salt (brackish), cider (runny), convalessence (cooling, light), cloning draught / neutron flux (light; clones followers / explodes).
- **Thirst clock (the only thing that can kill you by itself):** meter drops by Quickness/5 per action (~20), x0.66 with Fasting Way, x0.16 with Mind over Body, none asleep. Thresholds: >30 000 Tumescent, <=30 000 Quenched, <=20 000 Thirsty, <=10 000 Parched, 0 Dehydrated (each action a TOU save or 2 damage; natural healing off). One drink = +10 000, max 50 000 (overdrink = vomiting + putrescence pool). Auto-drinks at Thirsty by default.
- **Hunger** is soft: +1 per turn; Hungry at 1200 (2400 Fasting Way, 7200 Mind over Body, 14 400 both), Famished at 2400 (4800 / 14 400 / 28 800): lose metabolic effects and -10 Quickness. Meal resets to 0; snack -200. **Travelling more than a parasang on the world map makes you Hungry immediately.**

## 1.10 Reputation, factions, water ritual

- Reputation is an integer per faction. Bands: <= -600 hostile to all; -599..-250 hostile unless docile; -249..249 neutral (aggressive types still hostile); >= 250 friendly (pettable); >= 600 friendly. Demeanor docile / neutral / aggressive shifts the table. Neutral demeanor = hostile in holy places.
- **Water ritual** (with any non-hostile legendary/named creature through dialogue): share a secret +50, share gossip +100, ask for a secret -50, **recruit** (cost = max(50, level-based), unlimited followers), end. Killing someone you ritualed = **oathbreaker**: -100..-200 rep with every faction. Many faction members teach a skill in exchange for rep = SP cost.
- Start reps: Girsh -800, Templar -700, Mamon -600, Glow-Wights -650, Cannibals/Seekers/Sultan cult/Urchins -500, Robots -475, beasts -475, Cats -300, Hermits -100, Wardens -150, Phyta -10, Mechanimists/Merchants/Coiled Lamb/Daughters/Barathrumites 0, Dogs +300. Each procedural village gets its own faction. Hostility can fade if you stay nearby ~two weeks and rep was >= neutral. Attacking a faction's friend turns the faction hostile.

## 1.11 Trading

- Price multiplier m = clamp(0.07 x (EgoMod + 2 if Snake Oiler) + 0.35, 0.05, 0.95); sell = value x m; buy = value / m. Ego 0 => 0.35 (buy/sell ratio ~8:1), Ego +9 => 0.95. Good/bad faction rep shifts Ego by +-1; very good/bad by +-3. Glotrot fixes m = 0.14.
- Items have *variable* value (scales with Ego) or *constant* value (gold: better $/lb). Change is settled in drams (rounding favors trader). Followers do not charge you; a trader that follows you stops restocking. Trade goods: copper nugget $10/lb, silver $50, gold $200, bronze ingot $20 (5 lb).
- Only fresh water balances a trade; other liquids must be sold with their container.

## 1.12 Tinkering, mods, artifacts

- **Bits** (quality tiers 1..7+): items are made of bits; Tinkering lets you *see* an item's bits. **Disassemble** (INT 15): you always get the item's most significant bit and each other bit with 50 %. **Reverse Engineer** (INT 25): on disassembly 25 % to learn the item's schematic plus 15 % per mod to learn the mod. **Data disks** teach schematics/mods you could not otherwise build.
- **Tinker I/II/III** unlock low/medium/high schematics (+1 free schematic each). **Repair** mends broken items for bits. **Mods** cost bits = highest bit tier of the mod + item tier + 1 per existing mod; max 3 mods. Mod examples by bit tier: 1 Sharp (+1 PV), Masterwork (crit), Sturdy; 2 Flaming/Freezing/Electrified, Reinforced (+1 AV), Counterweighted; 3 Serrated (dismember), Spiked (bleed), Six-fingered (+1 AGI); 5 Refractive; 6 Beamsplitter, Displacer; 7 Nanon.
- **Artifacts** must be *examined* (Tinkering skill; an INT roll against item **complexity**, failure can break it, or pay an NPC water to do it); unidentified ones show a generic name ("strange tubes"). Weight hints at identity. *Gadget Inspector* improves examination.
- **Relics** come from sultan history: found randomly, with mods fixed by the sultan's theme (Glass, Ice, Salt, Stars, Time, Travel, Circuitry) on weapons, armor or shields.

## 1.13 Cooking (crafting package owns recipes; mechanics here for hooks)

- Cook at a campfire or clay oven (3 cookings until you next become Hungry/Famished). Skills limit ingredients: Meal Preparation 2, Spicer 3. *Basic effects* (one ingredient: honey +3 disease save, resists +10..15, stat +2..4) and *triggered effects* (one ingredient supplies a trigger like "whenever you become afraid" at 8-50 %, another supplies the result such as emit frost); random-effect ingredients exist. 10 % tasty bonus; Carbide Chef: pick from 3 outcomes and learn the recipe. Effects expire when Hungry/Famished (metabolizing).

## 1.14 World, villages, quests, history

- **World:** 80 x 25 parasang map, each = 3 x 3 zones of 80 x 25 tiles (18 000 surface zones); terrain types (flower fields, salt marsh/dunes, canyons, hills, jungle, deep jungle, ruins, mountains, rivers, etc.); region tier 1-8 sets creature level of surface/near-surface zones; deeper strata go down "many layers" and raise tier; biomes (slimy, tarry, rusty, fungal) re-skin zones with rare loot. Travel time per parasang 300 turns (flowers) to 900 (deep jungle). Story locations fixed; villages/ruins/lairs/historic sites regenerated each game.
- **Villages:** random names; mayor (or 3 parliamentarians), tinker, apothecary, warden, merchant; one creature always offers a procedural village quest; two generated history events; 1/20 per event the village is abandoned (empty but may keep an oven); first entry of a non-starting village = 250 x zone tier XP; starting-biome villages grant a skill or rep perk; parliamentary villages teach several skills via the ritual.
- **Quests:** steps (required/optional) with XP on step and finish; rewards = XP, rep, items; some have branching outcomes. Start: Joppa line (Fetch Argyve a Knickknack x2 -> Weirdwire Conduit.. Eureka! -> A Canticle for Barathrum) or random-village line (Village Quest -> A Signal in the Noise); core chain: More Than a Willing Spirit -> Decoding the Signal -> The Earl of Omonporch -> Grave Thoughts -> A Call to Arms -> The Assessment -> Pax Klanq, I Presume? -> Tomb of the Eaters -> The Golem -> Reclamation -> We Are Starfreight. Side sagas (Kith and Kin, What's Eating the Watervine?).
- **Sultans/history:** six sultans (five random, each an element/theme; Resheph fixed). Each has a cult historic site populated by factions that revered them; statues/paintings/engravings reveal "secrets" (logged in the journal and sold to faction notables who love or hate that sultan); relics enter the world with themed mods.
- **Lairs [stub]:** one surface zone plus N zones beneath, stairs down in each; bottom zone holds a legendary owner (merchant owners bring guards).
- **Legendary creatures:** unique variants (magenta): +1 all attributes, double HP, level x1.5, XP = level x 250, 0-2 physical and 0-2 mental mutations at level 1-4 (not robots/True Kin), faction opinions roll randomly. Appear as merchants, lair owners, historic-site cultists, wanderers. Village legendaries (governor/mayor/queen...) get +3/+1 Int-Will-Ego and a flee-at-HP% threshold.

## 1.15 Companions, glimmer, light

- **Followers** travel with you, fight, gain XP shared with leader. Recruit by Beguiling, Proselytize, ritual rep spend, Rebuke Robot, pet start, budding (cloning draught). Soft limits: Beguile and Proselytize one follower each. Commands cost a tenth of a turn: attack target, defensive, follow distance, move, stay, use ability. **Domination** lets you inhabit one temporarily (eat, equip, spend points as them).
- **Glimmer:** player-only number = sum of mental mutation levels. >= 20: "You are being watched" and Seekers of the Sightless Way send psychic hunters; >= 40: extradimensional hunters (orange). Esper hunters by tier: Osprey 20-34, Harrier 35-49, Owl 50-64, Condor 65-79, Strix 80-94, Eagle 95-109, Rukh 110+. Hunters match your level, copy your costlier mental mutations, travel in up to 5, spawn on new zone entry. Being favored by the Seekers cuts spawns 30-50 %. 10 % chance to gain +1 Ego when you personally kill one.
- **Light/vision [unverified]:** zones are lit/dark; light sources have radius; Night Vision and Myopic defects change sight; dark zones hide creatures outside light. Wanderer already does shadowcast FOV with light/darkvision.

---------------------------------------------------------------------
# PART 2. WHAT WANDERER ALREADY HAS (quick audit)
---------------------------------------------------------------------

- 6 attributes with Qud modifier formula `Mod`, point-buy chargen (species -> calling -> buy -> powers/defects -> origin), 7 callings (stat bonuses + skill ranks), 11 skill ids with ranks 1-2 and one-line effects (`SkillIDs`), skills buyable from teachers (`talk.go`).
- 43 `PowerDefs` (mostly activated, cooldown, flat; **no mutation levels**, no morphotypes, cost field only for creation), 5 defects, Skin-renamed ("Mutations"/"manifest").
- Combat: d20+AGI vs DV, up to 3 penetration rolls PV vs AV chaining (`penetrate`), crit flag, dismember/bleed/stun, elements, fire, multiple weapons, shields (`shieldAV`), natural weapons, body plans with severable limbs.
- Status map `Actor.Status` (frozen, stunned, burning, poisoned, bleeding, confused, phased, mighty, shielded, hasted, slowed, prone, corroding, sprint) but no registry, no Dazed/Shaken/Shamed/Asleep/Hobbled/Frenzied/Sluggish.
- Level-up: XP quadratic `60 l^2 + 40`, flat XP, +3+TOU HP, SP+1 (+1 at INT 18), MP+1, AP every 3 levels. No tier falloff, no 6-level all-attribute bump, no rapid advancement. Skill points not on a 70/50 scale; skills are rank-based not tree-based.
- Hunger/thirst: simple counters (`Hunger`, `Thirst`) with Hungry 1200/Famished 2400 and thirst 900/1800; drinking from water tiles (`drinkHere`); damage at starvation. No drams, no containers, no mixing, no pools, no metabolizing.
- Faction rep integer per realm/faith/creature faction, ritual gives rep + secret (`talk.go ritual`), gossip, secrets sold; tonics; trade with per-currency prices; recruit companion via persuasion/ritual (`follower` AI).
- Items: kinds weapon/armor/ranged/ammo/light/food/water/tonic/grenade/book/artifact/scrap/letter/trophy; rarity tiers + special categories (`rarity.go`); artifact identification via unid name; inventory weight/backpack; attire.
- World: 80x25 zones, 3x3 per parasang, strata z>0, deep layers, sky isles, sea floor; towns with notables, cities by tier, landmarks, dungeons (siteAt registry), towers, boss lairs, crypts with relics, wonders; quests (generated), ferries.
- Hooks: modeKeys/modeDraws/mainKeys/talkHooks/turnHooks/zoneHooks/killHooks/enterHooks/sideHooks + `g.ext`.

---------------------------------------------------------------------
# PART 3. GAP LIST, RANKED BY "FEELS LIKE QUD"
---------------------------------------------------------------------

Impact: **A** = defines Qud's identity, **B** = strongly noticeable, **C** = flavor. Each gap names the work package (WP) it lands in. Key bindings in sketches are suggestions: movement uses hjklyubn, and . < > g , i c j r m ? w M f t a x L R z s d e T S are taken, so new main-screen keys should be capitals not in that list (V, B, G, H, K, Q, X, Y, Z, O, P, U, W, E, F...) or an F-key. Verify against `input.go` before choosing.

## Ranked gaps

| # | Gap | Imp | Have | Missing | WP |
|---|-----|-----|------|---------|----|
| 1 | Leveled mutations (MP spend, level cap, level-scaled effects, rapid advancement) | A | one-time pick of ~43 flat powers; `g.MP` counter exists | per-level scaling, buy mutation at 4 MP from 3 offers, +1 level per MP, cap floor(L/2)+1, Rapid Advancement | 1 |
| 2 | Morphotypes + mutant/True Kin fork | A | species -> calling; Skin renames | genotype pick, Chimera/Esper restrictions, True Kin no mutations but 70 SP | 1 |
| 3 | Cybernetics + becoming nooks + license tier | A | none | implants, license points, slots, nook UI | 2 |
| 4 | Thirst/water-dram economy: waterskins, drams-as-currency, pools, mixing | A | binary thirst, water tiles, currency per Skin | containers with drams, pour/drink/fill, liquid kinds, evaporation, salty water, selling water to balance trades | 3 |
| 5 | Skill trees with named maneuvers (Charge, Cleave, Lunge, Jab, Slam, Shield Slam, Flurry...) | A | rank 1-2 per skill, text-only effects | tree structure with costs, activated maneuvers w/ cooldowns, stance toggles | 4 |
| 6 | Status-effect registry (Dazed, Prone, Shaken, Asleep, Hobbled, Shamed...) with numeric Qud effects | B | ad-hoc map | declarative table + tick + DV/AGI/speed hooks, infliction by skills and mutations | 4 |
| 7 | Glimmer + psychic hunters | A | none | glimmer computation, bird-ranked hunter spawn on zone entry, Ego-absorb | 1 |
| 8 | Legendary creatures with names, mutations, rituals | A | boss monsters, notables | magenta legendaries, level x1.5, 2x HP, mutations, own faction opinions, ritual-recruitable, lair owners | 5 |
| 9 | Water ritual full option set (secret/gossip/ask/recruit with cost, oathbreaker) and rep bands | B | single ritual -> rep + secret | gossip sale, ask-for-secret, recruit cost, oathbreaker penalty, demeanor table | 5 |
| 10 | Companions: commands, Domination, XP sharing, limits | B | follower AI, persuasion rank 2 recruit | command menu, follow distance, stay, ability use, domination takeover, limits per method | 5 |
| 11 | Main quest chain (Qud-style spine) and village side quests with step/XP/rep rewards | A | generated quests, chronicle | multi-step spine across worlds, village quest per town, branching outcomes | 6 |
| 12 | XP model: 15L^3+100 curve, tier falloff, discovery XP, +1-all-attributes every 6 levels | B | quadratic XP, flat | tier multiplier, discovery/quest-step XP, SP from INT, attribute cadence | 4 (or 1) |
| 13 | Ego-based price formula + faction-rep price shift + trade UI with change in drams | B | per-currency prices | m = clamp(0.07 x EgoMod + 0.35), constant vs variable value items | 3 |
| 14 | Artifact complexity and examination (INT roll, can break) | B | Unid names, lore skill identifies | complexity per artifact, INT/Tinkering roll, failure outcomes, NPC identification for water | 2 |
| 15 | Region/zone tiers, strata scaling, biomes (slimy/tarry/rusty/fungal) | B | strata, layers | tier-driven creature level and loot rarity; biome overlays and liquid pools in zones | 6 |
| 16 | Wayfaring/Wilderness Lore terrain skills; world-map hunger on travel | B | `survival` skill | per-terrain lore, camp-making, travel makes Hungry | 3 / 6 |
| 17 | Game modes: Classic (permadeath), Roleplay (save anywhere), Wander | B | permadeath + autosave | mode select at chargen, rules in `g.ext`, host hint | 6 |
| 18 | Defects expansion and mutation bundles | C | 5 defects | Evil Twin, Narcolepsy, Socially Repugnant, Carnivorous, Myopic... | 1 |
| 19 | Metabolizing and snack/meal distinction | C | food kind | meal/snack effects, famished -10 quickness, Fasting Way style skills | 3 |
| 20 | Item-naming by player ("Inspired" item naming), relic mod naming | C | rarity names | prompt to name equipped item on level up | 4 |

## Detailed sketches

### Gap 1-2, 7, 18: Mutation leveling, genotypes, glimmer (WP1)

Data model (in `g.ext("mut")`, struct `mutState`): `Lv map[string]int` (base levels), `Rapid map[string]int` (+3 each), `Genotype string` (`mutant`|`truekin`), `Morph string` (`chimera`|`esper`|``), `Pending []string` (3 offered ids), `Defects []string`. Keep `Actor.Powers[id]` as the effective level so existing `usePower`/`Pow(id)` code keeps working: after every change run `syncPowers(g)` which writes `Powers[id] = min(Lv+Rapid, cap) + egoBonus` for mental ids.

Specific steps:
1. `mutations.go`: table `MutDef{ID, Phys bool, Cost int, CD, Scale}` extending (not replacing) `PowerDefs` by id, with helpers `mutCap(level) = min(10, level/2+1)`, `buyMutation(g, id)` (4 MP), `levelMutation(g, id)` (1 MP). Scaling functions per id: damage dice grow, cooldown = base x (1 - 0.05 x max(0, WIL-16)) capped 0.2, duration +1 per level. Wire into the existing power use path via a new function `powerLevel(a, id)` that `usePower` calls (grep for the places that use the power's integer level; add a `lvl` argument).
2. Mutation screen: `modeKeys["mutate"]`, `modeDraws["mutate"]`, opened with `mainKeys["V"]` (or from the char screen `c`). Lists owned mutations with level, MP cost to raise, "learn new" (shows 3 offers re-rolled at each level-up from a seeded Rng so it is deterministic: `NewRng(seed ^ level)`).
3. Rapid advancement: `killHooks` no; hook into level-up. Wrap `gainXP`: add `levelUpHooks` in `mutations.go` by polling in a `turnHooks` function that compares `g.P.Level` with `ext.LastLevel` and, at level 5, 15, 25..., sets `ext.RapidDue=true`, prompting the player to pick a physical mutation (message + opens the mutate screen).
4. Morphotype at chargen is a UI concern: add to `CharSpec` a `Morph` field (JSON, optional) and handle in `g.ext` init through a `newGameHooks` equivalent; since hooks.go has none, use `enterHooks` first-call or `turnHooks` first tick to apply the morph restriction (Chimera filters to Phys, Esper to mental, extra limb roll for Chimera: append a `BodyPart{Slot:"hand",Limb:true}`).
5. `glimmer.go`: `glimmer(g) = sum levels of mental mutations (+ Ego bonus)`; `enterHooks` entry spawning a hunter with probability `clamp((glimmer-15)/60,0,.9)` when glimmer >= 20: creature template named after the bird tier table (Osprey...Rukh, Skin-renamed), level = player level, copying 1-2 of the player's mental powers at matching level, faction "hunters" (hostile, never neutral). 5-count cap rule: extra rolls at 0.35 chance each. `killHooks`: 10 % +1 EGO when killed by player and creature has `hunter` flag. `sideHooks`: show "Glimmer N" line when >= 15. Be careful: the player does not need to know about "mental" classification elsewhere; store it in the `MutDef.Phys` flag.
6. Defects (WP1 owns): add the missing defect ids to `PowerDefs` with `Defect:true` and implement via `turnHooks`/`killHooks`: Narcolepsy (1 in 400 turns while hostile in view set `asleep`), Evil Twin (spawn a mirror hunter once at level 5 with player's name), Socially Repugnant (AddRep(-50) to all factions at start; `g.AddRep`), Carnivorous (non-meat food gives no hunger relief: needs a food-flag check in the eat path), Myopic (sight radius 10).

Tests: `mutations_test.go` for cap formula, MP spending, deterministic offers, glimmer hunter spawns once per zone entry.

### Gap 3, 14: Cybernetics and artifact complexity (WP2)

`cyber.go`: state `cyberState{Tier int, Installed []string}` in `g.ext("cyber")`. `CyberDef{ID, Name, Cost, Slots []string, Destroyed bool, Apply func(a *Actor, on bool)}` — implement ~14 from the wiki list with Skin-renamed names (genre skin gives "chrome"/"augment"/"rune-graft"). Effects: use existing hooks into `Actor` (AV, DV via `Item`-like modifiers): simplest is a virtual `Item` with `Bonus`, `AV`, `DV` placed in a new `BodyPart{Slot:"implant"}` that `Actor.AV()/DV()/Score` already sum if they iterate `Body`. If they only iterate slot names, add the new slots to the Skin slot list in `MakeBody` (one-line change inside `entity.go`, flag in handoff).
Becoming nook: a placeable terrain/furniture feature spawned through `zoneHooks` in ruins/temples/labs (probability via rng from `zn` seed) with `Item{Kind:"nook"}` or an `Actor` with `Role:"nook"`. Interacting (bump) opens `modeKeys["nook"]`: install, uninstall (refund), raise tier (price 1/2/3/4 `Skin.Currency` units by tier brackets 1-8, 9-16, 17-24, 25+). Racks sell implants (use `talkHooks` and a `Role:"cyberdealer"` actor in cities with tier ≥ 2).
Tier cap tied to genotype: when `Genotype == "mutant"` the nook refuses ("Your flesh rejects it") — shares state with WP1 through `g.ext("mut")` read-only; define the key string constant in `cyber.go` and read with a local small struct so there is no import cycle (same package).
Complexity: `examine.go`: each artifact gets `Extra["cx"]` complexity 1-6 on first sight (derived from its `Tier`/`Rarity`); `examineArtifact(g, it)` rolls `d20 + M(INT) + 3 x tinkering rank` vs `10 + 4 x cx`; success identifies (sets `Ident`), failure by 10+ breaks it, in between wastes a turn. `mainKeys` hook on the item action menu is not available; use `talkHooks` to add "Identify for N drams" with a `Role` tinker/scholar NPC and add a `mainKeys["X"]` quick-examine of the item under cursor. Do NOT touch `rarity.go`.

### Gap 4, 13, 16, 19: Liquids, thirst and trade math (WP3)

`liquids.go`: `Liquid{ID, Name, Color, Evap, Sticky, Slippery, Flammable, Drink func(g *Game, drams int)}` table (water, brackish, salt, acid, oil, honey, slime, blood, ink, wine/ale (Skin), lava). State: `liquidState` in `g.ext("liquids")`: `Pools map[string]*Pool` keyed `z:x:y` with `{Liq string; Drams int}`; per-zone cache so mixing and evaporation (every 60 turns pools < 200 lose 1 dram; `turnHooks`) are cheap. Items: `Kind:"vessel"` with `Extra["cap"]=64`, contents encoded as `Extra["dr"]` drams + `Text` liquid id (waterskin, canteen; skin-named). Keep legacy `Kind:"water"` items convertible: on load, turn each into a full waterskin (use `enterHooks` once).
UI: `modeKeys["liquid"]` opened by `mainKeys["V"]`? (V taken by WP1, use `mainKeys["Y"]`... pick a different key and note it in handoff): pour, fill from pool at feet, drink N drams, dump. Drinking calls `Liquid.Drink`; mixing creates adjectives list on the pool (`Mix []string`).
Thirst rewrite (replace body of the counters in `turn()` without editing game.go): set `g.Thirst` semantics via a `turnHooks` function that *overrides* effect by keeping the existing counter but scaling: expose helper `thirstStage(g)` returning Quenched/Thirsty/Parched/Dehydrated using Qud thresholds converted to the engine's scale. Because editing the `turn()` core is out of the hook design, the pragmatic route: WP3 owns a small, explicit edit in `game.go` lines ~405-425 (the hunger/thirst block) to call `survivalTick(g)` in `liquids.go`; list exactly that edit in handoff. `survivalTick` implements: thirst drops by quickness/5 per player action, x0.66 Fasting Way, x0.16 Mind over Body analog (Endurance rank 3), none when asleep; Dehydrated = TOU save or 2 dmg and no regen; Hungry 1200/Famished 2400 stages, Famished -10 quickness; world-map travel > one parasang marks Hungry.
Trade: `trade2.go` (or in `liquids.go`): `priceMult(g, a)` = clamp(0.07 x (EgoMod + 2 if persuasion rank>=2) + 0.35, .05, .95) + rep shift (+-1 Ego at |rep|>=250, +-3 at >=600). Replace existing price functions' multiplier via a function variable `priceHook func(g *Game, base int, buying bool) int` that talk.go calls; if no such seam exists, add one one-line indirection in `talk.go` (flag in handoff). Fresh water as barter change: add "Sell N drams" option via `talkHooks` for merchants; make the Skin currency optional "water-only" for the desert-like genres (flag on Skin: `WaterCurrency bool`).
Wayfaring: `Wilderness lore` as per-terrain bonuses in `survival`: -travel hunger on lore'd biome. Store in the skills package (WP4 owns the tree; WP3 supplies `terrainLore(g, terrain) bool`).

### Gap 5, 6, 12, 20: Skill trees, maneuvers, statuses, XP (WP4)

`skilltrees.go`: `TreeDef{ID, Name, Unlock int, Attr int (gate), Nodes []NodeDef}`, `NodeDef{ID, Name, Cost int, Req string, Kind (passive|active|stance), Cool int, Apply}`; ~12 trees from the wiki: Axe, Cudgel, LongBlade, ShortBlade, Bow, Pistol, Heavy, Shield, MultiWeapon, Tactics, Acrobatics, Endurance, SelfDiscipline, Persuasion, Physic, Tinkering (node list only; tinker recipes belong to crafting), Customs&Folklore. Current `Skills map[string]int` rank values are migrated: rank 1 => tree unlocked + proficiency, rank 2 => first 150-cost node (done in `migrateSkills` on first load with a `g.ext("trees")` version flag). Keep `skillLv()` working by computing rank from unlocked-node count so combat code doesn't change.
SP economy: tree unlock 50-150 and nodes 50-400, but Wanderer's SP is +1/level. Map: 1 SP = 25 Qud points; tree unlock = 2-6 SP, nodes 0-16 SP, level-up grants 2 (+1 per 2 INT above 10 over a floor of 0) — pick numbers that give a first dozen levels a real choice (target: ~3 nodes per 2 levels). Document the numbers at the top of the file as constants.
Maneuvers as active powers: register in the existing abilities screen by writing into `Actor.Powers` + `Cool` using ids `sk_<node>` and handling them in `usePower`'s fallthrough? That path lives in powers.go; instead register `mainKeys["Z"]`-style ability menu extension: `modeKeys["maneuver"]` listing unlocked actives. Implement: Charging Strike (move N tiles and hit with +crit), Cleave (hit all adjacent, cracks armor: `AddSt(target,"cracked",5)` -> AV-2), Dismember (calls existing `g.dismember(d)` with chance), Lunge, Jab, Hobble, Slam (stun + knockback), Shield Slam, Flurry (extra swings next round), Dueling Stance (toggle: +DV vs single foe), Akimbo (pistol dual), Dead Shot, Sling and Run, Berate, Intimidate (Shaken), Menacing Stare, Staunch Wounds/Amputate (physic), Meditate (cooldown reset), Hurdle.
`statuses.go`: declarative `StatusDef{ID, Name, Glyph, DV, AGI, INT, WIL, EGO, Speed, MA, Tick func(g,a)}` table with the Qud numbers from 1.7; central `statMod(a, field)` used by new code; to affect existing code, post-process `Actor.DV()`/`Quick()`: add a one-line call `a.statusMods()` inside those two functions (flag the edit). Include the new ones: dazed, prone, shaken, disoriented, shamed, hobbled, asleep (-12 DV, +4 PV for attackers: `penetrate` hook), sluggish, frenzied, wartrance, cracked, syphoned, ill (after poison), glotrot, metabolizing, inspired.
XP model: `leveling.go` (in this WP): replace `XPFor` constants? `XPFor` lives in game.go; WP4 owns a one-function swap `XPFor(l) = (15*l^3+100) / 8` (scaled for our smaller XP numbers) and the `killXP(attackerLevel, victim)` tier multiplier via a function variable `killXPHook` used in `kill()`. Discovery XP in `enterHooks`: first entry of town/ruin/lair zone awards `250 x tier / 25`. +1 all attributes every 6 levels via a `turnHooks` level watcher. On level-up, 5 % chance (100 % with Carbide-chef analog) to prompt naming the main-hand weapon: `modeKeys["nameitem"]`, text input mode — check whether `modeKeys` supports text entry; otherwise reuse the chargen name-entry pattern.
Tests: `skilltrees_test.go` (migration keeps combat rank), `statuses_test.go` (prone DV, asleep penetration).

### Gap 8, 9, 10: Legendaries, ritual options, companions (WP5)

`legendary.go`: `makeLegendary(g, base *Actor, rng *Rng) *Actor`: +1 all attributes, MaxHP x2, Level x1.5, XP = Level x 250 via `Actor.XP`, name from a Skin-flavored generator (titles like "the Pale Marshal of ..."), 0-2 physical + 0-2 mental `Powers` at level 1-4 (from WP1's physical/mental lists — duplicate a small id list locally to stay independent), faction opinions rolled (Love/Like/Dislike/Hate stored in `g.ext("legend")`). Glyph F color magenta (`CMagenta` — check name in render.go colors). Placement via `zoneHooks`: wandering legendary in 1 of ~12 wilderness zones, lair owner in bottom level of dungeon sites already registered in `dungeon.go` (look at `siteAt` boss selection; add a flag `Boss` + `Legendary`), cultist leader in ruins with chronicle inscriptions (read them, don't alter). Merchant legendaries get hired guards (2 `Role:"guard"` followers).
`ritual2.go`: extends the existing ritual through `talkHooks` (append options after the base ritual when `a.Talked` or Proper): "Share gossip (+100)" (needs a known gossip from `g.Secrets` kind gossip — reuse `g.addSecret` records, mark sold), "Ask for a secret (-50)", "Recruit (cost max(50, level x 4) rep)", "Learn a skill (cost = SP x 10 rep)" for NPCs with `Teach`. Oathbreaker: `killHooks` — if victim has `Flags["rituald"]`, `gainRepAll(-150, "oathbreaker")`. Rep bands function `attitude(rep, demeanor)` returns hostile/neutral/friendly per the wiki table; expose `repBand(rep)` for the faction screen (render tag via `sideHooks`/factions screen only if existing renderer exposes a text-per-faction; otherwise skip).
`companions.go`: `modeKeys["order"]` opened with `mainKeys["O"]`: choose follower -> Attack target (target mode reuse via `g.Mode="target"` pattern), Stay, Follow close/far (`Actor.State` field or `Flags["stay"]`), Use ability. `AI=="follower"` branch in `ai.go` reads flags; add the flag check inside that branch (flag the edit, ~5 lines). Limits: count followers by `Flags["via_beguile"]` etc.; Beguile replaces prior. XP sharing: `killHooks`: when `src` is a follower of player, `gainXP(base/2)`; when player kills, followers' `XP` +25 %. Domination (`dominate` power id, mental cost 5): `g.ext("dom")` stores original `P`; swap `g.P` pointer to the follower for N rounds (`turnHooks` countdown) — verify `g.P` is a plain pointer and that no cached index is kept; safe fallback if risky: "control" mode that lets the player issue orders and equip the follower via its inventory UI instead of body-swap. Tests with scenario world.

### Gap 11, 15, 17: Quest spine, tiers/biomes, game modes (WP6)

`questchain.go`: spine `[]Step{ID, Title, Trigger, Reward}`; per-world flavor from `WorldData` (use existing names: the ruling house, capital, a ruin, the sultan analog) so the 10-step spine is: arrive at origin village (errand: fetch X) -> repair a relay/shrine (rare item) -> decode signal (visit a ruin with a chronicle inscription, read it: reuse the lore book `Item.Lore`) -> meet the sage (legendary from WP5 if present; else notable) -> clear a lair/tower floor -> relic in a crypt (use the existing dead-ruler crypt relic as the macguffin) -> endgame spire. State `g.ext("spine")` `{Step int, Flags map}`; progress hooks in `killHooks` (boss kill), `enterHooks` (site zone), `talkHooks` ("Report" option). Rewards: XP via `g.gainXP`, rep, a named item. Branching choice at one step (keep vs give relic) stored in flags and altering the final line only.
Village side quest per town: `zoneHooks` on town zones; `talkHooks` on the town's quest-giver (use `Quest` field on Actor already present) — check `talk.go` quest generation and only add a variety layer (step-based: "fetch + return", "kill lair owner", "escort", "deliver letter"), no duplicate of existing kinds.
Journal: spine appears via `sideHooks` single line "Quest: <title>"; full list in the journal if there is an extensible hook (otherwise `modeKeys["spine"]` at `mainKeys["Q"]`).
`tiers.go`: `zoneTier(g, wx, wy, z)` = clamp(1 + terrain danger/2 + |z|/2, 1, 8) used in `zoneHooks` to scale newly spawned hostile levels: after zone generation, for each hostile non-boss `Actor` set `Level = max(Level, tier*3-2)` and rescale `MaxHP`/attrs through a helper (check `creatures.go` for the existing scaling helper, call it, don't duplicate). Biomes: `biomes.go`: deterministic per-parasang biome from world seed (slimy/tarry/rusty/fungal; Skin-named), implemented by `zoneHooks` repainting a fraction of floor tiles with glyph/color overlay and sprinkling pools of the biome's liquid (WP3's `Pool` — until WP3 lands, use ground items of kind `"scrap"` only and gate liquid pools behind `g.ext("liquids")` existing; define a tiny shared constant file instead of cross-importing).
`modes.go`: Classic/Roleplay/Wander as `g.ext("mode")`: `Mode string`. Classic = current permadeath (delete save on death: check how `die()` handles save in game.go and the web store), Roleplay = on death restore the latest manual save (`S` key exists — verify), Wander = no death (respawn at last town with penalty). Mode select needs a `CharSpec.Mode` field plus `CharGen.svelte` radio (web polish package owns the Svelte file; coordinate by adding only the Go field and exposing options in `CharOptions` JSON `modes` array; UI wiring is a follow-up).

---------------------------------------------------------------------
# PART 4. WORK PACKAGES (independent, each in its own new files)
---------------------------------------------------------------------

Each package uses only the hook registry and `g.ext(key, ...)`; any unavoidable core edit is listed ("core touch") so the integrator can serialise them. Packages are independent of each other except for two soft read-only couplings noted in brackets.

| WP | Name | New files | Gaps | Core touch |
|----|------|-----------|------|------------|
| 1 | Mutation depth, genotypes, glimmer hunters | `engine/mutations.go`, `engine/glimmer.go`, `engine/mutations_test.go` | 1, 2, 7, 18 | `CharSpec` gains `Genotype`, `Morph`; one call to `powerLevel` in the power-use path in `powers.go`/`powers2.go` |
| 2 | Cybernetics, becoming nooks, artifact complexity | `engine/cyber.go`, `engine/examine.go`, `engine/cyber_test.go` | 3, 14 | possibly new body slot "implant" in `entity.go MakeBody` (read-only for tier if WP1 absent) |
| 3 | Liquids, water economy, survival clocks, trade math | `engine/liquids.go`, `engine/trade2.go`, `engine/survival.go`, `engine/liquids_test.go` | 4, 13, 16, 19 | replace hunger/thirst block in `game.go turn()` with `survivalTick(g)`; one price indirection in `talk.go` |
| 4 | Skill trees, maneuvers, status registry, XP model | `engine/skilltrees.go`, `engine/maneuvers.go`, `engine/statuses.go`, `engine/leveling.go`, tests | 5, 6, 12, 20 | `DV()`/`Quick()`/`penetrate` read `statusMods`; `XPFor` swap; `skillLv` reads trees |
| 5 | Legendaries, full water ritual, reputation bands, companion orders and domination | `engine/legendary.go`, `engine/ritual2.go`, `engine/companions.go`, tests | 8, 9, 10 | follower branch in `ai.go` honours stay/follow flags |
| 6 | Quest spine, village quest variety, zone tiers, biomes, game modes | `engine/questchain.go`, `engine/tiers.go`, `engine/biomes.go`, `engine/modes.go`, tests | 11, 15, 17 | `CharSpec.Mode`; death handler consults `modes.go`; `CharOptions` adds `modes` |

Suggested build order for value: WP1 and WP4 (combat/character identity) first, then WP3 (water economy is Qud's most distinctive loop), WP5, WP2, WP6. WP1 vs WP4 both touch power use and `CharSpec` — assign WP4 the abilities menu extension and WP1 the mutation screen, and have them agree only on the `powerLevel(a, id)` function name.

## Handoff rules for every agent (from CLAUDE.md)
- Go 1.23, no dependencies, no LLM calls. Register from `init()` in your own file; do not edit `game.go`/`input.go`/`render.go` beyond the listed core touch (keep each to a few lines).
- Persist only through `g.ext(key, mk)`; keep state JSON-serialisable; use `NewRng`/zone-seeded rngs for determinism.
- Genre text via `Skin` (`g.Sk` / `g.skin()`): names for mutations, implants, liquids, ritual must read right in every genre; fall back to neutral wording.
- Do not run tests/build/commit unless asked (memory rule); when asked, `go test ./engine/` and `go run ./cmd/fuzz world.json` after engine changes, `npm run wasm` for the browser build.
- Out of scope (other agents): realmforge link/chronicle, web polish/mobile/music, sandbox, player home/base, wars, crafting/enchanting/mods and cooking recipes, cults and orders, codex.
