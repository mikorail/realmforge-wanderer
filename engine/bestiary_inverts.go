package engine

// Bestiary catalogue: living invertebrates (insects, arachnids, crustaceans, molluscs and more).
// name|group|when|habitat|size|diet|temper|activity|flags|glyph+colour|rarity|attack|description
var _ = bsReg(`
# insects
Monarch butterfly|ins|L|4,6|1|h|passive|d|fly @avian|vO|1|proboscis|An orange butterfly that migrates thousands of miles over four generations.
Atlas moth|ins|L|7|1|h|passive|n|fly @avian|vO|3|wings|One of the largest moths, its wingtips patterned like cobra heads.
Luna moth|ins|L|6|1|h|passive|n|fly @avian|vG|3|wings|A pale green moth with long trailing tails that lives only a week.
Blue morpho|ins|L|7|1|h|passive|d|fly @avian|vB|2|proboscis|A butterfly whose wings flash an electric blue made by light, not pigment.
Western honey bee|ins|L|4,6|1|h|herd|d|fly poison|bY|1|sting|A colony insect that dances to tell its sisters where flowers are.
Asian giant hornet|ins|L|6,mtn|1|c|territorial|d|fly poison|bO|3|sting|A thumb-sized hornet whose scouts mark a hive for a slaughter by the swarm.
Bumblebee|ins|L|4,6,10|1|h|passive|d|fly poison|bY|1|sting|A furry bee that shivers to warm itself and flies in the cold.
Leafcutter ant|ins|L|7,5|1|h|herd|d||aG|1|mandibles|Ants that cut leaves to feed an underground fungus garden.
Army ant|ins|L|7|1|c|pack|a|bleed|aw|2|mandibles|A blind army of millions that devours everything in its path.
Bullet ant|ins|L|7|1|o|territorial|a|poison|ak|3|sting|An ant whose sting is said to feel like being shot.
Termite queen|ins|L|3,7|2|h|passive|a|burrow|aY|3|mandibles|A swollen egg-laying queen at the heart of a termite mound, laying an egg every few seconds.
Goliath beetle|ins|L|7|1|h|passive|d|armored fly|bK|3|horn|One of the heaviest insects, a striped beetle as big as a fist.
Hercules beetle|ins|L|7|2|h|territorial|n|armored fly|bG|3|horn|A beetle with a horn longer than its body that lifts many times its weight.
Stag beetle|ins|L|6|1|h|passive|n|armored fly|bw|2|mandibles|Males wrestle with antler-like jaws.
Bombardier beetle|ins|L|6,4|1|c|passive|n|acid|bO|3|spray|It sprays a boiling chemical jet from its rear.
Dung beetle|ins|L|3,4|1|s|passive|a|armored|bk|1|horn|It rolls dung balls guided by the stars of the Milky Way.
Firefly|ins|L|6,12|1|c|passive|n|fly|vY|1|bite|A beetle that flashes cold light to signal its mate.
Desert locust|ins|L|1,3|1|h|herd|d|fly|vG|1|mandibles|A grasshopper that turns gregarious and swarms by the billion, devouring crops.
Giant weta|ins|L|8|2|h|passive|n|armored|wk|4|mandibles|An island cricket heavier than a sparrow.
Cave cricket|ins|L|cave|1|s|skittish|a||jw|1|mandibles|A long-legged, humpbacked cricket of caves and cellars.
Praying mantis|ins|L|4,6,5|1|c|territorial|d|bleed|iG|1|claws|An ambush hunter with folded spiked forelegs that turns its head to watch you.
Orchid mantis|ins|L|7|1|c|passive|d|bleed|iM|3|claws|A pink mantis that looks exactly like an orchid flower.
Emperor dragonfly|ins|L|fresh,12|1|c|skittish|d|fly @avian|vB|1|mandibles|A large hawker dragonfly that hunts on the wing.
Walking stick|ins|L|7,6|1|h|passive|n||iw|2|claws|An insect that looks exactly like a twig.
Malaria mosquito|ins|L|12,7|1|h|herd|n|fly bleed|vK|1|proboscis|The deadliest animal of all, carrier of the fever that has killed billions.
Tsetse fly|ins|L|3,7|1|c|herd|d|fly bleed|vw|2|proboscis|A biting fly that carries sleeping sickness.
Cicada|ins|L|6,4|1|h|herd|d|fly|vG|1|proboscis|Its broods emerge every thirteen or seventeen years in deafening millions.
Giant water bug|ins|L|fresh,12|2|c|territorial|a|poison|bw|3|beak|A huge water bug that injects digestive saliva into fish and frogs.
Titan beetle|ins|L|7|2|h|passive|n|armored fly|bw|4|mandibles|A giant longhorn beetle whose jaws can snap a pencil.
# arachnids
Goliath birdeater|arach|L|7|2|c|territorial|n|poison|xw|3|fangs|The heaviest spider, which hisses by rubbing its legs and flicks stinging hairs.
Black widow|arach|L|1,4,6|1|c|passive|n|poison|xk|2|fangs|A glossy black spider with a red hourglass and a potent bite.
Brazilian wandering spider|arach|L|7,5|1|c|territorial|n|poison|xw|3|fangs|An aggressive hunting spider that roams the forest floor at night.
Golden orb-weaver|arach|L|7,5|1|c|passive|d||xY|2|fangs|It spins golden webs so strong they were once woven into cloth.
Jumping spider|arach|L|any|1|c|passive|d||xK|1|fangs|A tiny spider with huge eyes that stalks and pounces like a cat.
Funnel-web spider|arach|L|6,8|1|c|territorial|n|poison|xk|3|fangs|A glossy black spider whose venom can kill a man.
Camel spider|arach|L|1|2|c|territorial|n||xY|2|jaws|A fast desert solifuge with huge jaws, subject of many soldiers' tall tales.
Emperor scorpion|arach|L|7,3|2|c|passive|n|poison armored|SK|2|sting|A large black scorpion that glows under moonlight.
Deathstalker|arach|L|1|1|c|territorial|n|poison|sY|3|sting|A slender yellow scorpion with one of the strongest venoms of its kind.
Whip scorpion|arach|L|1,7|1|c|passive|n|acid|sk|2|spray|A vinegaroon that sprays acid from its whip-like tail.
Diving bell spider|arach|L|fresh,12|1|c|passive|a||xw|4|fangs|A spider that lives underwater in a bubble of air it spins and carries.
Cave harvestman|arach|L|cave|1|s|passive|a||xw|1|legs|A long-legged harvestman that gathers in shivering clusters on cave ceilings.
# crustaceans and others
Coconut crab|crust|L|coast,5|3|o|territorial|n|armored|KR|3|claws|The largest land arthropod, which climbs palms and cracks coconuts with its claws.
Japanese spider crab|crust|L|reef|4|s|passive|a|aquatic armored|Kr|4|claws|A crab with legs spanning wider than a man is tall.
Red king crab|crust|L|reefc|3|c|passive|a|aquatic armored|kR|2|claws|A spiny giant crab of the cold north.
Fiddler crab|crust|L|coast,12|1|s|skittish|d|armored|kO|1|claws|Males wave one huge claw to court and to duel.
Ghost crab|crust|L|coast|1|s|skittish|n|armored|kY|1|claws|A pale beach crab that dashes sideways into its burrow.
Hermit crab|crust|L|coast,reef|1|s|skittish|a|armored|kw|1|claws|A crab that lives in borrowed shells and trades up as it grows.
Christmas Island red crab|crust|L|coast,7|1|h|herd|a|armored|kR|3|claws|Each year tens of millions migrate from forest to sea in a red tide.
American lobster|crust|L|reefc|3|c|passive|n|aquatic armored|kR|2|claws|A big-clawed lobster that may live a century.
Mantis shrimp|crust|L|reefw|1|c|territorial|a|aquatic armored|kG|2|club|A shrimp whose punch boils the water and cracks aquarium glass.
Pistol shrimp|crust|L|reefw|1|c|passive|a|aquatic|kR|2|claw|It stuns prey with a snap that makes a bubble hotter than the sun's surface.
Antarctic krill|crust|L|seac,ice|1|h|herd|a|aquatic|kR|1|claws|A small shrimp in swarms so vast they outweigh every person alive.
Giant isopod|crust|L|reef|2|s|passive|a|aquatic armored|kK|3|mandibles|A deep-sea woodlouse the size of a cat that can go years without eating.
Woodlouse|crust|L|6,4,cave|1|s|passive|n|armored|kK|1|mandibles|A land crustacean that rolls into a ball.
Horseshoe crab|arth|L|coast,reef|2|s|passive|n|armored|kw|2|spine|An armoured relative of spiders with blue blood, unchanged for hundreds of millions of years.
Amazonian giant centipede|myria|L|7|2|c|territorial|n|poison|jR|3|forcipules|A foot-long centipede that hangs from cave roofs to snatch bats.
Giant African millipede|myria|L|7,3|2|h|passive|n|poison armored|jK|2|mandibles|A finger-thick millipede that curls into a spiral and oozes irritant.
House centipede|myria|L|any,cave|1|c|skittish|n||jw|1|forcipules|A fast many-legged hunter of cellars.
Velvet worm|myria|L|8,7|1|c|passive|n||jw|4|slime|A soft many-legged hunter that snares prey in jets of sticky slime.
# molluscs
Giant Pacific octopus|moll|L|reefc|4|c|skittish|n|aquatic @tentacled|OM|3|arms|The largest octopus, clever enough to unscrew jars and escape tanks.
Blue-ringed octopus|moll|L|reefw|1|c|passive|a|aquatic poison @tentacled|oY|3|beak|A tiny octopus whose blue rings flash before a bite that can kill.
Mimic octopus|moll|L|reefw|2|c|skittish|d|aquatic @tentacled|ow|4|arms|It impersonates lionfish, sea snakes and flatfish.
Common cuttlefish|moll|L|reef|2|c|skittish|d|aquatic @tentacled|ow|2|arms|It hypnotises prey with rippling waves of colour.
Giant squid|moll|L|reef,sea|6|c|territorial|a|aquatic @tentacled|XR|4|arms|A deep-sea squid with eyes the size of dinner plates, the kraken of legend.
Colossal squid|moll|L|reefc|6|c|territorial|a|aquatic @tentacled|XR|5|arms|The heaviest invertebrate, with swivelling hooks on its arms.
Humboldt squid|moll|L|sea|4|c|pack|n|aquatic @tentacled|xR|3|arms|A red devil squid that hunts in flashing packs.
Vampire squid|moll|L|reef|1|s|passive|a|aquatic @tentacled|oR|4|arms|A deep-sea cephalopod that turns its webbed cloak inside out when afraid.
Chambered nautilus|moll|L|reefw|2|c|passive|n|aquatic armored @tentacled|nw|3|arms|A shelled cephalopod that floats by adjusting gas in its chambers.
Giant clam|moll|L|reefw|4|f|passive|a|aquatic armored|cB|3|shell|A clam as heavy as a man that farms algae in its mantle.
Queen conch|moll|L|reefw|2|h|passive|a|aquatic armored|cO|2|shell|A sea snail with a flaring pink shell.
Geography cone|moll|L|reefw|1|c|passive|n|aquatic poison|cw|3|harpoon|A cone snail that harpoons fish with a venomous tooth.
Roman snail|moll|L|6,4|1|h|passive|n|armored|cw|1|shell|A large land snail long eaten as a delicacy.
Giant African land snail|moll|L|7,5|2|h|passive|n|armored|cw|2|shell|A snail as long as a hand that devours crops.
Leopard slug|moll|L|6|1|h|passive|n||cw|1|rasp|A spotted slug that mates hanging from a rope of mucus.
Sea hare|moll|L|reef,coast|2|h|passive|a|aquatic|cM|2|ink|A sea slug that squirts purple ink.
Blue dragon nudibranch|moll|L|seaw|1|c|passive|a|aquatic poison|cB|4|stingers|A tiny blue sea slug that eats men-of-war and keeps their stings.
# cnidarians, echinoderms, sponges, worms
Portuguese man o' war|cnid|L|seaw|2|c|passive|a|aquatic poison @tentacled|JM|2|tentacles|A floating colony with a purple sail and tentacles thirty paces long.
Lion's mane jellyfish|cnid|L|seac|4|c|passive|a|aquatic poison @tentacled|JO|3|tentacles|The largest jellyfish, its tentacles longer than a blue whale.
Box jellyfish|cnid|L|seaw,coast|2|c|passive|a|aquatic poison @tentacled|JY|3|tentacles|A nearly invisible jellyfish with eyes and a sting that can stop a heart.
Moon jellyfish|cnid|L|sea,reef|1|f|passive|a|aquatic @tentacled|jY|1|tentacles|A translucent drifting bell with four pale rings.
Immortal jellyfish|cnid|L|seaw|1|f|passive|a|aquatic regen @tentacled|jR|4|tentacles|A tiny jellyfish that can turn itself young again when it is hurt or old.
Brain coral|cnid|L|reefw|3|f|passive|a|aquatic armored|cy|1|polyps|A domed coral grooved like a brain, centuries old.
Staghorn coral|cnid|L|reefw|3|f|passive|a|aquatic armored|cO|1|polyps|A branching coral that builds the reef's forests.
Sea anemone|cnid|L|reef,coast|1|c|passive|a|aquatic poison|aM|1|tentacles|A flower-like animal that stings what touches it, except the clownfish.
Crown-of-thorns starfish|echi|L|reefw|2|c|passive|a|aquatic poison armored|eM|3|spines|A spiny many-armed starfish that eats the coral reef.
Common starfish|echi|L|reef,coast|1|c|passive|a|aquatic|eO|1|arms|A starfish that pushes its stomach out to digest clams in their shells.
Purple sea urchin|echi|L|reef,coast|1|h|passive|a|aquatic armored|eM|1|spines|A spined ball that grazes kelp forests to stubble.
Sea cucumber|echi|L|reef|2|s|passive|a|aquatic|eK|1|guts|A sausage of the sea floor that defends itself by spitting out its own guts.
Barrel sponge|misc|L|reefw|3|f|passive|a|aquatic armored|oO|2|spicules|A giant sponge that may be older than any tree.
Giant tube worm|worm|L|reef|2|f|passive|a|aquatic|wR|4|plume|A red-plumed worm of volcanic vents that lives off bacteria in its body.
Bobbit worm|worm|L|reefw|3|c|territorial|n|aquatic|wM|4|jaws|A three-pace-long worm that hides in sand and snaps fish in half.
Giant Gippsland earthworm|worm|L|6,4|2|s|passive|a|burrow|ww|4|mouth|An earthworm as long as a man that gurgles in its burrow.
Medicinal leech|worm|L|fresh,12|1|c|passive|a|bleed|lr|1|sucker|A leech still used by doctors to drain bruised blood.
Tardigrade|misc|L|any,cave|1|h|passive|a|regen|aw|2|stylets|A water bear smaller than a grain of sand that can survive boiling, freezing and the void.
`)
