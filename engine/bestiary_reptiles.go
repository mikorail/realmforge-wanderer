package engine

// Bestiary catalogue: living reptiles and amphibians.
// name|group|when|habitat|size|diet|temper|activity|flags|glyph+colour|rarity|attack|description
var _ = bsReg(`
# crocodilians
Nile crocodile|rept|L|fresh,12,3|6|c|predator|a||KG|2|jaws|A river ambusher that waits a year for the wildebeest crossing.
Saltwater crocodile|rept|L|12,coast,fresh,seaw|6|c|predator|a||Kg|3|jaws|The largest living reptile, swimming between islands and taking anything that drinks.
American alligator|rept|L|12,fresh|5|c|territorial|a||Kg|2|jaws|A broad-snouted reptile that bellows in the swamps so the water dances on its back.
Mugger crocodile|rept|L|fresh,12|5|c|predator|a||Kw|3|jaws|A marsh crocodile that balances sticks on its snout to lure nesting birds.
Gharial|rept|L|fresh|5|p|skittish|a||Kg|4|jaws|A slender-snouted fish-eater whose males wear a pot-shaped knob on the nose.
Black caiman|rept|L|fresh,12,7|5|c|predator|n||Kk|3|jaws|The largest predator of the flooded forest, black as old leather.
Spectacled caiman|rept|L|fresh,12|4|c|territorial|n||kg|1|jaws|A small caiman with a bony ridge between its eyes like spectacles.
Chinese alligator|rept|L|fresh,12|4|c|skittish|n||kK|5|jaws|A small alligator of the great river that sleeps the winter in burrows.
# lizards
Komodo dragon|rept|L|3,5|5|c|predator|d|poison|DK|4|jaws|The largest lizard, a venomous ambusher of island grasslands.
Asian water monitor|rept|L|12,fresh,5|4|c|skittish|d||Lg|2|claws|A swimming monitor that scavenges riverbanks and village dumps.
Perentie|rept|L|1|4|c|territorial|d||Lw|3|claws|A desert monitor that stands on its hind legs to look around.
Nile monitor|rept|L|fresh,3|4|c|territorial|d||Lg|2|claws|A big river lizard that digs up crocodile eggs.
Marine iguana|rept|L|coast|3|h|passive|d||ik|4|claws|The only sea-going lizard, grazing algae from the rocks and sneezing out salt.
Green iguana|rept|L|7,5|3|h|skittish|d||iG|1|claws|A big tree lizard that drops into rivers to escape danger.
Rhinoceros iguana|rept|L|1,5|3|h|territorial|d||iK|4|horns|An island iguana with bony horns on its snout.
Gila monster|rept|L|1|2|c|passive|n|poison|lO|3|bite|A beaded orange-and-black lizard that chews venom into its bite.
Mexican beaded lizard|rept|L|1,5|3|c|passive|n|poison|lk|4|bite|A dark beaded lizard, a venomous relative of the Gila monster.
Thorny devil|rept|L|1|1|i|passive|d|armored|lO|3|spines|A spiky desert lizard that drinks dew through grooves in its skin.
Frilled lizard|rept|L|3,6|2|i|territorial|d||lO|2|bite|It spreads a ruff of skin and runs on its hind legs to bluff its foes.
Panther chameleon|rept|L|7|1|i|passive|d||lG|3|tongue|A chameleon that flashes colours of mood with a tongue longer than its body.
Tokay gecko|rept|L|7,5|1|i|territorial|n||lB|2|bite|A loud blue-spotted gecko that bites and will not let go.
Leaf-tailed gecko|rept|L|7|1|i|passive|n||lw|4|bite|A gecko so like a dead leaf it disappears when it stops moving.
Shingleback skink|rept|L|1,3|2|h|passive|d|armored|lw|2|bite|A slow lizard with a stumpy tail shaped like its head.
Common basilisk|rept|L|7,fresh|2|i|skittish|d||lG|2|claws|The Jesus Christ lizard, which sprints across the surface of rivers.
Flying dragon|rept|L|7|1|i|skittish|d|fly|lO|3|bite|A lizard that glides between trees on wings of rib and skin.
Tuatara|rept|L|8|2|i|passive|n||lG|5|bite|The last of an order that thrived with the dinosaurs, with a third eye on its crown.
Sand lizard|rept|L|4,6|1|i|skittish|d||lG|1|bite|A small green lizard of heaths and dunes.
# snakes
King cobra|rept|L|7,5|4|c|territorial|d|poison @serpent|SG|3|fangs|The longest venomous snake, which rears to look a man in the eye and eats other snakes.
Indian cobra|rept|L|5,3,4|3|c|territorial|c|poison @serpent|Sk|2|fangs|A hooded cobra revered and feared in equal measure.
Black mamba|rept|L|3|4|c|territorial|d|poison @serpent|SK|3|fangs|A fast grey snake named for the inky lining of its mouth.
Inland taipan|rept|L|1|3|c|passive|d|poison @serpent|Sw|4|fangs|The most venomous land snake, shy and rarely seen.
Gaboon viper|rept|L|7|3|c|passive|n|poison @serpent|Sw|3|fangs|A heavy, beautifully patterned viper with the longest fangs of any snake.
Puff adder|rept|L|3,1|3|c|territorial|n|poison @serpent|Sw|2|fangs|A sluggish adder that puffs and hisses before it strikes.
Western diamondback rattlesnake|rept|L|1,4|3|c|territorial|n|poison @serpent|Sw|1|fangs|A pit viper that rattles its tail as a warning.
Common European adder|rept|L|6,9,4|2|c|skittish|d|poison @serpent|sK|1|fangs|A zig-zag backed viper of heaths, found even beyond the Arctic circle.
Sidewinder|rept|L|1|2|c|skittish|n|poison @serpent|sY|2|fangs|A horned desert rattlesnake that throws its body sideways over the sand.
Eastern coral snake|rept|L|6,5|2|c|passive|n|poison @serpent|sR|3|fangs|Red touching yellow, kills a fellow: a small, bright, venomous snake.
Bushmaster|rept|L|7|4|c|territorial|n|poison @serpent|Sw|4|fangs|The longest viper of the jungle, waiting for weeks beside game trails.
Fer-de-lance|rept|L|7,5|3|c|territorial|n|poison @serpent|Sw|2|fangs|A lance-headed pit viper of plantations and forests.
Yellow-bellied sea snake|rept|L|seaw|2|p|passive|a|poison aquatic @serpent|sY|2|fangs|A sea snake that drifts in slicks of floating weed and never comes ashore.
Banded sea krait|rept|L|reefw,coast|2|p|passive|a|poison @serpent|sB|2|fangs|A striped sea snake that hunts eels in the reef and lays its eggs on land.
Green anaconda|rept|L|12,fresh,7|5|c|predator|a|@serpent|SG|3|coils|The heaviest snake, which drowns caimans and deer in the flooded forest.
Reticulated python|rept|L|7,5|5|c|predator|n|@serpent|Sw|3|coils|The longest snake in the world, patterned like woven cloth.
Burmese python|rept|L|5,12|5|c|territorial|n|@serpent|Sw|2|coils|A heavy python of jungle and swamp that may swallow a deer whole.
Boa constrictor|rept|L|7,5|4|c|passive|n|@serpent|Sw|2|coils|A patient tree and ground snake that squeezes its prey.
Emerald tree boa|rept|L|7|3|c|passive|n|@serpent|sG|3|fangs|A green boa that coils over branches and strikes birds from the air.
Grass snake|rept|L|6,12,fresh|2|p|skittish|d|@serpent|sG|1|bite|A harmless water snake that plays dead and stinks when caught.
Milk snake|rept|L|6,4|2|c|skittish|n|@serpent|sR|1|bite|A harmless snake that mimics the coral snake's bands.
Paradise tree snake|rept|L|7|2|c|skittish|d|fly @serpent|sG|3|bite|It flattens its body and glides from treetops.
Blind snake|rept|L|any|1|i|passive|a|burrow @serpent|sw|2|bite|A tiny burrowing snake that looks like an earthworm.
# turtles
Galapagos giant tortoise|rept|L|5,1|5|h|passive|d|armored|TK|4|beak|A domed tortoise that may live for more than a century and a half.
Aldabra giant tortoise|rept|L|5,3|5|h|passive|d|armored|TK|4|beak|An island giant tortoise that grazes like a cow.
Desert tortoise|rept|L|1|2|h|passive|d|armored burrow|tw|2|beak|A tortoise that spends most of its life underground.
Leopard tortoise|rept|L|3|3|h|passive|d|armored|tw|2|beak|A spotted savanna tortoise.
Leatherback sea turtle|rept|L|sea|6|h|passive|a|aquatic armored|TK|3|beak|The largest turtle, which dives deeper than many whales to eat jellyfish.
Green sea turtle|rept|L|seaw,reefw|5|h|passive|a|aquatic armored|TG|2|beak|A turtle that grazes sea grass and returns to the beach where it hatched.
Hawksbill sea turtle|rept|L|reefw|4|h|passive|a|aquatic armored|Tw|3|beak|A reef turtle with a hooked bill that feeds on sponges.
Loggerhead sea turtle|rept|L|sea,reef|5|c|passive|a|aquatic armored|Tw|2|beak|A big-headed turtle that crushes conchs in its jaws.
Alligator snapping turtle|rept|L|fresh,12|4|c|territorial|a|armored|Tk|3|jaws|It lies open-mouthed on the river bed, wiggling a worm-like lure on its tongue.
Common snapping turtle|rept|L|fresh,12|3|c|territorial|a|armored|tK|2|jaws|A long-necked turtle that strikes like a snake.
Painted turtle|rept|L|fresh,12|1|o|skittish|d|armored|tG|1|beak|A small pond turtle that basks in rows on logs.
Box turtle|rept|L|6,4|1|o|passive|d|armored|tw|2|beak|A turtle that closes its hinged shell tight like a box.
Matamata|rept|L|fresh,7|3|p|passive|a|armored|tw|4|jaws|A turtle like a pile of leaves that sucks fish in with a gulp.
Pig-nosed turtle|rept|L|fresh|3|o|skittish|a|armored|tK|4|beak|A turtle with flippers and a snout like a pig.
# amphibians
Cane toad|amph|L|7,5,12|2|o|passive|n|poison|tw|1|bite|A huge warty toad with poison glands, a plague wherever it is carried.
Common toad|amph|L|6,4|1|i|passive|n|poison|tw|1|bite|A warty toad that walks back to the pond where it was born.
Common frog|amph|L|6,12,fresh|1|i|skittish|a||fG|1|bite|A pond frog that spawns in great masses each spring.
American bullfrog|amph|L|fresh,12|2|c|skittish|n||fG|1|bite|A huge frog that eats anything it can fit in its mouth, including snakes.
Goliath frog|amph|L|fresh,7|3|c|skittish|n||fG|4|bite|The largest frog, the size of a house cat.
Golden poison frog|amph|L|7|1|i|passive|d|poison|fY|4|bite|A tiny yellow frog with enough poison in its skin to kill ten men.
Strawberry poison-dart frog|amph|L|7|1|i|passive|d|poison|fR|3|bite|A red frog with blue legs that carries its tadpoles to water in tree bromeliads.
Red-eyed tree frog|amph|L|7|1|i|skittish|n||fG|2|bite|It flashes red eyes and orange feet to startle predators.
Wallace's flying frog|amph|L|7|1|i|skittish|n|fly|fG|3|bite|A frog that glides between trees on webbed feet.
Glass frog|amph|L|7|1|i|passive|n||fG|3|bite|A frog so transparent its beating heart can be seen through its belly.
Wood frog|amph|L|9,10|1|i|skittish|d||fw|2|bite|A frog that freezes solid in winter and thaws back to life in spring.
Desert rain frog|amph|L|1,coast|1|i|passive|n|burrow|fw|4|bite|A round little frog that squeaks indignantly when disturbed.
Axolotl|amph|L|fresh|2|c|passive|a|aquatic regen|aM|4|bite|A feathery-gilled salamander that never grows up and can regrow its limbs.
Olm|amph|L|cave,fresh|1|c|passive|a|aquatic|aY|4|bite|A blind pale salamander of flooded caves, once thought to be a baby dragon.
Fire salamander|amph|L|6,mtn|1|i|passive|n|poison|aO|2|bite|A black-and-yellow salamander once believed to live in fire.
Japanese giant salamander|amph|L|fresh,mtn|4|c|passive|n|aquatic|AK|4|jaws|A salamander as long as a man, unchanged for thirty million years.
Hellbender|amph|L|fresh,6|3|c|passive|a|aquatic|aw|3|jaws|A flat, wrinkled salamander of fast rocky streams.
Great crested newt|amph|L|6,12,fresh|1|i|passive|n|poison|ak|2|bite|A warty newt whose males grow a jagged crest in spring.
Caecilian|amph|L|7|2|i|passive|a|burrow @serpent|sk|3|bite|A legless burrowing amphibian; its young eat their mother's skin.
`)
