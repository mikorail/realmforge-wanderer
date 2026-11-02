package engine

// Bestiary catalogue: extinct animals of deep time, from Pleistocene megafauna back to the Ediacaran.
// They live only in lost worlds (and Pleistocene giants on cold steppes in some genres); fossils everywhere.
// name|group|when|habitat|size|diet|temper|activity|flags|glyph+colour|rarity|attack|description
var _ = bsReg(`
# Pleistocene megafauna
Woolly mammoth|mam|Late Pleistocene:0.01|10,9,2,4|6|h|herd|d|ivory|MW|3|tusks|A shaggy elephant of the mammoth steppe with spiralling tusks; the last ones lived on a lonely Arctic island.
Columbian mammoth|mam|Late Pleistocene:0.012|4,3|6|h|herd|d|ivory|Mw|3|tusks|A huge, sparsely haired mammoth of warmer plains, taller than any living elephant.
American mastodon|mam|Late Pleistocene:0.011|9,6,12|6|h|herd|d|ivory|Mr|3|tusks|A stocky browser of spruce forests and swamps, distinct from the true mammoths.
Straight-tusked elephant|mam|Middle Pleistocene:0.1|6,4|6|h|herd|d|ivory|My|4|tusks|A towering elephant of interglacial forests, larger than any elephant now alive.
Woolly rhinoceros|mam|Late Pleistocene:0.014|10,2,4|6|h|territorial|d||RW|3|horn|A shaggy rhino with a flattened front horn for sweeping snow off the grass.
Elasmotherium|mam|Late Pleistocene:0.039|2,4|6|h|territorial|d||Rw|4|horn|The Siberian unicorn, a giant rhino of the steppe with one great horn on its brow.
Cave bear|mam|Late Pleistocene:0.024|9,mtn,cave|5|h|territorial|a||BW|3|claws|A huge vegetarian bear whose bones fill whole caves where it slept the winter.
Short-faced bear|mam|Late Pleistocene:0.011|4,9|6|c|predator|a||BO|4|claws|A long-legged giant bear that stood twice a man's height on its hind legs.
Smilodon|mam|Late Pleistocene:0.01|4,3,6|5|c|predator|n||Lw|3|sabres|The sabre-toothed cat, a muscular ambusher with dagger canines a forearm long.
Homotherium|mam|Late Pleistocene:0.03|10,2,4|5|c|pack|a||Lw|4|sabres|The scimitar cat, long-legged and short-sabred, which ran down young mammoths.
American lion|mam|Late Pleistocene:0.011|4,9|5|c|pack|n||LO|4|claws|A giant lion of the ice-age plains, a quarter larger than any lion now.
Cave lion|mam|Late Pleistocene:0.013|10,2,4|5|c|pack|n||LW|3|claws|A maneless lion painted on cave walls by the people who feared it.
Dire wolf|mam|Late Pleistocene:0.013|4,9,6|4|c|pack|n||Cw|2|bite|A heavy wolf with a crushing bite, trapped by the thousand in tar pits.
Giant beaver|mam|Late Pleistocene:0.012|fresh,12,9|4|h|skittish|a||RK|3|teeth|A beaver as big as a bear with teeth like chisels a hand long.
Megatherium|mam|Late Pleistocene:0.012|3,4|6|h|territorial|d||GO|4|claws|A giant ground sloth as tall as an elephant when it reared on its hind legs.
Glyptodon|mam|Late Pleistocene:0.012|3,4|5|h|passive|d|armored|Gw|3|tail|A car-sized armadillo with a domed shell of bony plates.
Doedicurus|mam|Late Pleistocene:0.011|4|5|h|territorial|d|armored|GK|4|tail club|A glyptodont with a spiked mace on the end of its tail.
Toxodon|mam|Late Pleistocene:0.012|3,12|5|h|herd|d||Hw|3|teeth|A rhino-like hoofed beast of South American grasslands that Darwin dug up.
Macrauchenia|mam|Late Pleistocene:0.012|3,4|5|h|herd|d||Gw|4|teeth|A long-necked, camel-like grazer with nostrils high on its head, perhaps a short trunk.
Megaloceros|mam|Late Pleistocene:0.008|4,6,10|5|h|herd|c||DO|3|antlers|The giant deer, called the Irish elk, with antlers spanning more than a man's height.
Steppe bison|mam|Late Pleistocene:0.01|10,2,4|6|h|herd|d||Bw|2|horns|A long-horned bison of the mammoth steppe, painted in ochre on cave walls.
Giant short-faced kangaroo|mam|Late Pleistocene:0.04|1,3|5|h|skittish|d||Kw|4|kick|A heavy kangaroo that walked rather than hopped and browsed tall trees.
Diprotodon|mam|Late Pleistocene:0.044|1,3,6|6|h|herd|d||Hw|3|teeth|The largest marsupial ever, a wombat the size of a rhinoceros.
Thylacoleo|mam|Late Pleistocene:0.04|6,3|4|c|predator|n||lw|4|thumb claw|The marsupial lion, with shearing blade teeth and a killing claw on each thumb.
Megalania|rept|Late Pleistocene:0.05|1,3|6|c|predator|d|poison|DO|4|jaws|A giant monitor lizard as long as a boat, top predator of the ancient outback.
Gigantopithecus|mam|Middle Pleistocene:0.3|7,5,mtn|6|h|territorial|d|@humanoid|GK|5|fists|The largest ape that ever lived, a bamboo-eating giant of southern forests.
Castoroides|mam|Middle Pleistocene:0.1|fresh,12|4|h|skittish|a||Rw|4|teeth|A long-bodied giant beaver of ice-age marshes that may never have built dams.
Cave hyena|mam|Late Pleistocene:0.03|4,10,cave|4|c|pack|n||yW|3|jaws|A big spotted hyena that dragged bones into its den caves.
Wild horse of the steppe|mam|Late Pleistocene:0.01|10,2,4|5|h|herd|d||hO|2|teeth|A dun horse of the mammoth steppe, the commonest animal in cave paintings.
Saiga of the steppe|mam|Late Pleistocene:0.02|2,10|3|h|herd|d||aY|3|horns|An ice-age saiga that grazed from the far west to the Bering land bridge.
Teratornis|bird|Late Pleistocene:0.011|4,1,sky|5|s|skittish|d|fly|VK|4|beak|A giant condor-like bird that soared over ice-age tar pits.
Pleistocene musk ox|mam|Late Pleistocene:0.01|10,11|5|h|herd|d||Mw|3|horns|A tall helmeted musk ox of the glacial edge.
Megalochelys|rept|Middle Pleistocene:0.5|5,3|6|h|passive|d|armored|TK|4|beak|An enormous land tortoise taller than a man with a shell like an upturned boat.
Hippopotamus antiquus|mam|Middle Pleistocene:0.1|fresh,6|6|h|territorial|n||HM|4|tusks|A giant hippo that wallowed in the rivers of the far north between ice ages.
Woolly mammoth calf|mam|Late Pleistocene:0.04|10,2|4|h|herd|d||mW|4|trunk|A young mammoth with a coat of russet fluff, following its matriarch.
# Cenozoic
Paraceratherium|mam|Oligocene:30|2,4,1|7|h|herd|d||PO|4|teeth|The largest land mammal ever, a hornless rhino tall enough to browse treetops.
Andrewsarchus|mam|Eocene:42|coast,4|5|c|predator|a||Ak|4|jaws|A huge-skulled meat-eater known from a single skull longer than a man's arm.
Basilosaurus|mam|Eocene:37|seaw|7|c|predator|a|aquatic @serpent|Wb|4|teeth|An eel-shaped early whale with tiny hind legs and terrible teeth.
Ambulocetus|mam|Eocene:48|fresh,coast|4|c|predator|a||Ag|4|jaws|The walking whale, a crocodile-like ancestor of whales that hunted from the shallows.
Pakicetus|mam|Eocene:50|fresh,4|3|c|skittish|a||aw|4|teeth|A wolf-sized land mammal whose ear bones show it was the first step to the whales.
Megalodon|fish|Pliocene:3.6|sea,reef|7|c|predator|a|aquatic|FW|4|teeth|The largest shark that ever lived, with teeth like hands that bit through whale ribs.
Livyatan|mam|Miocene:9|sea|7|c|predator|a|aquatic|WK|5|teeth|A raptorial sperm whale with the largest teeth of any animal ever, rival to megalodon.
Titanoboa|rept|Paleocene:60|12,7,fresh|6|c|predator|a|@serpent|SG|4|coils|A snake longer than a bus that squeezed crocodiles in the first rainforests.
Phorusrhacos|bird|Miocene:15|4,3|5|c|predator|d||Tw|3|beak|A terror bird, flightless and fast, which struck with a hatchet-shaped beak.
Kelenken|bird|Miocene:15|4|5|c|predator|d||TK|4|beak|The tallest terror bird, with the largest skull of any bird known.
Gastornis|bird|Eocene:50|6,8|5|h|territorial|d||Gw|3|beak|A huge flightless bird once thought a predator, now thought to crack nuts and seeds.
Argentavis|bird|Miocene:6|mtn,4,sky|5|c|skittish|d|fly|VK|4|talons|The heaviest flying bird known, with wings as wide as a small aeroplane.
Pelagornis|bird|Miocene:10|seaw,coast|4|p|skittish|d|fly|AY|4|beak|A bony-toothed seabird with the widest wingspan of any bird ever.
Entelodon|mam|Oligocene:30|4,6|5|o|predator|d||Qk|3|jaws|A hell pig, a long-skulled omnivore that bit its rivals on the face.
Daeodon|mam|Miocene:20|4|5|o|predator|d||QK|4|jaws|The largest hell pig, a bone-crushing scavenger the size of a bison.
Chalicotherium|mam|Miocene:10|6,5|5|h|passive|d||Gw|4|claws|A horse relative that walked on its knuckles and pulled down branches with long claws.
Deinotherium|mam|Miocene:8|5,6,3|6|h|herd|d|ivory|Ek|4|tusks|An elephant cousin whose tusks curved downward from its lower jaw.
Platybelodon|mam|Miocene:13|12,4|6|h|herd|d|ivory|Ew|4|tusks|The shovel-tusker, which scooped plants with its flattened lower tusks.
Amphicyon|mam|Miocene:12|4,6|5|c|predator|a||Bw|3|bite|A bear dog, a heavy hunter halfway between a bear and a wolf.
Thylacosmilus|mam|Pliocene:3.5|3,4|4|c|predator|n||lw|4|sabres|A sabre-toothed marsupial whose fangs grew throughout its life.
Hyracotherium|mam|Eocene:52|6,5|2|h|skittish|d||hw|2|teeth|The dawn horse, no bigger than a fox, with toes instead of hooves.
Moropus|mam|Miocene:20|4|5|h|passive|d||Gw|4|claws|A clawed chalicothere with a horse's head and a gorilla's posture.
Deinogalerix|mam|Miocene:7|5|2|i|skittish|n||nw|4|bite|A giant island hedgehog with a long snout and no spines.
Purussaurus|rept|Miocene:8|fresh,12|7|c|predator|a||KG|4|jaws|A caiman longer than a bus, master of the ancient swamps.
Megalochelys atlas|rept|Pliocene:4|5,3|6|h|passive|d|armored|Tw|4|beak|A giant tortoise of the ancient east with a shell taller than a man.
Indricotherium|mam|Oligocene:28|2,4|7|h|passive|d||Pw|4|teeth|A colossal hornless rhino of the arid plains, cousin to Paraceratherium.
Brontotherium|mam|Eocene:35|4,6|6|h|herd|d||RK|3|horns|A thunder beast with a forked horn on its nose, bones once taken for those of giants.
Uintatherium|mam|Eocene:45|12,6|6|h|territorial|d|ivory|Rw|4|tusks|A six-horned, sabre-tusked plant-eater with a tiny brain.
Arsinoitherium|mam|Oligocene:30|12,5|6|h|territorial|d||Rw|4|horns|A rhino-like beast with two huge conical horns side by side over its nose.
Smilodectes|mam|Eocene:48|5|2|h|skittish|d|@humanoid|pw|4|bite|An early lemur-like primate of the steaming Eocene forests.
Leptictidium|mam|Eocene:47|6|1|i|skittish|n||jw|4|bite|A tiny insectivore that ran on its hind legs with a twitching trunk-like snout.
Odobenocetops|mam|Pliocene:4|seaw,coast|5|c|passive|a|aquatic ivory|Wy|5|tusks|A walrus-faced whale with one tusk far longer than the other.
Josephoartigasia|mam|Pliocene:3|12,4|5|h|passive|a||Rw|4|teeth|The largest rodent ever, a guinea-pig cousin the size of a bull.
Megapiranha|fish|Miocene:9|fresh|3|c|pack|a|aquatic|uR|4|teeth|A piranha four times the size of any alive, with zig-zag teeth.
# Mesozoic: dinosaurs
Tyrannosaurus|dino|Late Cretaceous:68|5,6,12,4|6|c|predator|a||TR|3|jaws|The tyrant lizard king, a bone-crushing hunter with the strongest bite of any land animal.
Tarbosaurus|dino|Late Cretaceous:70|12,4|6|c|predator|a||Tr|4|jaws|A close cousin of Tyrannosaurus that hunted the river plains of the east.
Albertosaurus|dino|Late Cretaceous:70|6,8|5|c|pack|a||TO|3|jaws|A lean tyrannosaur whose bones are found together, perhaps a hunting pack.
Daspletosaurus|dino|Late Cretaceous:76|4,6|6|c|predator|a||TK|4|jaws|A heavy tyrannosaur, the frightful lizard of the coastal plains.
Giganotosaurus|dino|Late Cretaceous:97|5,4,12|6|c|predator|a||Gr|4|jaws|One of the largest meat-eaters ever, with shearing teeth like steak knives.
Carcharodontosaurus|dino|Late Cretaceous:95|12,5|6|c|predator|a||Gw|4|jaws|The shark-toothed lizard, a giant hunter of a land of rivers and mangroves.
Spinosaurus|dino|Late Cretaceous:97|fresh,12,7|6|p|predator|a||SO|4|jaws|A sail-backed river giant with a crocodile snout that hunted huge fish.
Baryonyx|dino|Early Cretaceous:125|12,fresh|5|p|territorial|a||Bg|3|claws|A fish-eating dinosaur with a great hooked thumb claw.
Allosaurus|dino|Late Jurassic:150|4,6,3|5|c|predator|a||AO|2|jaws|The top predator of its age, which hacked at prey with a mouth like a hatchet.
Ceratosaurus|dino|Late Jurassic:153|4,12|5|c|predator|a||Cr|3|jaws|A horned predator with a blade-like crest on its snout.
Torvosaurus|dino|Late Jurassic:150|6,4|6|c|predator|a||Tw|4|jaws|One of the largest Jurassic hunters, heavy-skulled and savage.
Carnotaurus|dino|Late Cretaceous:70|4,1|5|c|predator|d||Cr|3|horns|A bull-horned predator with tiny arms and long fast legs.
Majungasaurus|dino|Late Cretaceous:70|3,1|5|c|predator|d||Mr|4|jaws|An island predator known to have eaten its own kind.
Dilophosaurus|dino|Early Jurassic:190|12,6|4|c|predator|a||dY|3|jaws|A slender hunter with a pair of thin crests on its head.
Coelophysis|dino|Late Triassic:210|1,4,12|3|c|pack|d||dG|2|teeth|A small, hollow-boned early dinosaur found in great herds.
Herrerasaurus|dino|Late Triassic:230|5,4|4|c|predator|a||dO|3|jaws|One of the earliest dinosaurs, a lean predator with a flexible jaw.
Velociraptor|dino|Late Cretaceous:75|1,2|3|c|pack|a|bleed|vR|2|sickle claw|A turkey-sized feathered hunter with a sickle claw on each foot.
Deinonychus|dino|Early Cretaceous:115|5,6,4|4|c|pack|a|bleed|dR|2|sickle claw|The terrible claw, a wolf-sized feathered predator that may have hunted in groups.
Utahraptor|dino|Early Cretaceous:130|4,6|5|c|pack|a|bleed|UR|3|sickle claw|The largest raptor, heavy and feathered, with a killing claw as long as a dagger.
Troodon|dino|Late Cretaceous:76|6,9,12|3|c|predator|n||dY|3|claws|A big-eyed, big-brained feathered hunter of the polar night.
Oviraptor|dino|Late Cretaceous:75|1,2|3|o|skittish|d||oM|2|beak|A crested, parrot-beaked dinosaur found brooding its own eggs.
Gallimimus|dino|Late Cretaceous:70|1,4,3|4|o|herd|d||oY|2|beak|An ostrich-like dinosaur that ran in flocks across the sands.
Struthiomimus|dino|Late Cretaceous:75|4,6|4|o|herd|d||ow|2|beak|An ostrich mimic, swift and toothless.
Deinocheirus|dino|Late Cretaceous:70|12,fresh|6|o|passive|d||Dw|4|claws|A humpbacked, duck-billed giant with arms as long as a man.
Therizinosaurus|dino|Late Cretaceous:70|6,5|6|h|territorial|d||Tg|4|scythe claws|A pot-bellied plant-eater with scythe claws a forearm long.
Microraptor|dino|Early Cretaceous:120|6,7|1|c|skittish|a|fly|vB|3|claws|A four-winged feathered dinosaur that glided between trees in glossy black plumage.
Archaeopteryx|dino|Late Jurassic:150|1,coast|1|c|skittish|d|fly|vw|3|claws|The first bird, with teeth, clawed wings and a long bony tail.
Compsognathus|dino|Late Jurassic:150|coast,1|1|c|skittish|d||dG|2|teeth|A chicken-sized hunter of lizards on dry island shores.
Sinosauropteryx|dino|Early Cretaceous:124|6,12|1|c|skittish|d||dO|3|teeth|A small dinosaur with a ginger-and-white striped tail, the first known with feathers.
Yutyrannus|dino|Early Cretaceous:125|6,9|5|c|predator|a||TW|4|jaws|A shaggy-feathered tyrannosaur of cold forests.
Brachiosaurus|dino|Late Jurassic:154|5,6,4|7|h|herd|d||BG|3|tail|A giraffe-shaped sauropod whose front legs were longer than its hind legs.
Brontosaurus|dino|Late Jurassic:152|4,12|7|h|herd|d||Bw|2|tail|The thunder lizard, a long-necked giant once thought never to have existed.
Apatosaurus|dino|Late Jurassic:152|4,12|7|h|herd|d||Aw|2|tail|A massive long-necked sauropod with a whip-like tail.
Diplodocus|dino|Late Jurassic:152|4,6|7|h|herd|d||Dw|2|tail|A very long sauropod whose tail may have cracked like a bullwhip.
Argentinosaurus|dino|Late Cretaceous:95|5,4|7|h|herd|d||AK|4|tail|Among the largest land animals ever, heavier than a dozen elephants.
Patagotitan|dino|Late Cretaceous:100|5,4|7|h|herd|d||PK|4|tail|A titanosaur longer than a blue whale, found in a quarry on a sheep farm.
Mamenchisaurus|dino|Late Jurassic:160|6,12|7|h|herd|d||Mg|3|tail|A sauropod with a neck half the length of its body.
Camarasaurus|dino|Late Jurassic:150|4,6|7|h|herd|d||Cw|2|tail|The commonest giant of its plains, with spoon-shaped teeth.
Amargasaurus|dino|Early Cretaceous:129|4,1|6|h|herd|d||Ar|4|tail|A small sauropod with a double row of tall spines along its neck.
Plateosaurus|dino|Late Triassic:214|1,4|5|h|herd|d||Pw|2|claws|An early long-necked plant-eater that walked on its hind legs.
Stegosaurus|dino|Late Jurassic:152|4,6|6|h|territorial|d|armored|SG|2|thagomizer|A plated plant-eater with four spikes on its tail and a brain the size of a walnut.
Kentrosaurus|dino|Late Jurassic:152|3,4|5|h|territorial|d|armored|Kg|3|spikes|A smaller stegosaur bristling with spines along its back and tail.
Ankylosaurus|dino|Late Cretaceous:68|4,6|6|h|territorial|d|armored|AK|3|tail club|A tank of a dinosaur armoured head to tail with a bone club on its tail.
Euoplocephalus|dino|Late Cretaceous:76|4,6|5|h|territorial|d|armored|EK|3|tail club|An ankylosaur that even had armoured eyelids.
Nodosaurus|dino|Late Cretaceous:110|6,coast|5|h|passive|d|armored|Nw|3|spikes|A clubless armoured dinosaur studded with knobs.
Triceratops|dino|Late Cretaceous:67|4,6|6|h|territorial|d|armored|TG|2|horns|A three-horned, frilled plant-eater that faced down Tyrannosaurus.
Styracosaurus|dino|Late Cretaceous:75|4,6|5|h|herd|d||Sw|3|horns|A horned dinosaur whose frill bore a crown of long spikes.
Pachyrhinosaurus|dino|Late Cretaceous:72|9,6|5|h|herd|d||Pw|3|horns|A horned dinosaur of the polar forests with a thick bony boss on its nose.
Protoceratops|dino|Late Cretaceous:75|1,2|3|h|herd|d||pY|2|beak|A sheep-sized horned dinosaur found locked in combat with a Velociraptor.
Psittacosaurus|dino|Early Cretaceous:120|6,12|3|h|herd|d||pG|2|beak|A parrot-beaked plant-eater with a fringe of bristles on its tail.
Pachycephalosaurus|dino|Late Cretaceous:68|4,mtn|4|h|territorial|d||Pw|3|domed skull|A dome-headed dinosaur whose skull was a foot of solid bone.
Parasaurolophus|dino|Late Cretaceous:75|12,6,8|6|h|herd|d||PO|2|tail|A duck-billed dinosaur that trumpeted through a long hollow crest.
Corythosaurus|dino|Late Cretaceous:76|12,6|6|h|herd|d||Cy|2|tail|A helmet-crested duckbill whose calls carried far across the deltas.
Edmontosaurus|dino|Late Cretaceous:67|12,9,6|6|h|herd|d||Ew|1|tail|A huge duckbill that migrated in vast herds along the coasts.
Maiasaura|dino|Late Cretaceous:76|4,6|6|h|herd|d||Mg|2|tail|The good mother lizard, which nested in colonies and cared for its young.
Lambeosaurus|dino|Late Cretaceous:75|12,6|6|h|herd|d||Lw|3|tail|A duckbill with a hatchet-shaped crest.
Iguanodon|dino|Early Cretaceous:125|6,12|6|h|herd|d||IG|2|thumb spike|One of the first dinosaurs ever described, with a spike for a thumb.
Ouranosaurus|dino|Early Cretaceous:115|1,3|5|h|herd|d||Or|3|thumb spike|A sail-backed iguanodont of hot river plains.
Muttaburrasaurus|dino|Early Cretaceous:105|6,4|5|h|herd|d||Mw|4|thumb spike|A plant-eater with a hollow bump on its snout, perhaps to call.
Hypsilophodon|dino|Early Cretaceous:125|6,coast|2|h|skittish|d||dG|2|beak|A small, swift plant-eater that lived in herds along the ancient coasts.
Leaellynasaura|dino|Early Cretaceous:110|9,6|2|h|skittish|a||dw|4|beak|A tiny big-eyed dinosaur of the polar forests and their months of darkness.
Mussaurus|dino|Late Triassic:215|4|4|h|herd|d||Mw|4|claws|An early sauropod relative first known from tiny hatchlings.
# Mesozoic: pterosaurs
Pteranodon|ptero|Late Cretaceous:86|coast,sea,sky|4|p|skittish|d|fly|PY|2|beak|A toothless pterosaur with a long backward crest that skimmed the inland sea for fish.
Quetzalcoatlus|ptero|Late Cretaceous:68|4,3,sky|6|c|predator|d|fly|QO|4|beak|A giraffe-tall pterosaur that strode the plains spearing small dinosaurs.
Hatzegopteryx|ptero|Late Cretaceous:66|5,sky|6|c|predator|d|fly|HK|5|beak|A heavy-jawed giant pterosaur, top predator of an island of dwarf dinosaurs.
Rhamphorhynchus|ptero|Late Jurassic:150|coast,seaw|2|p|skittish|d|fly|vR|2|teeth|A long-tailed pterosaur with a diamond vane at its tail tip and forward-pointing teeth.
Pterodactylus|ptero|Late Jurassic:150|coast,fresh|2|p|skittish|d|fly|vw|2|beak|The first pterosaur ever named, a small short-tailed flier of lagoons.
Dimorphodon|ptero|Early Jurassic:195|coast|2|c|skittish|d|fly|vO|3|teeth|A big-headed early pterosaur with two kinds of teeth.
Tapejara|ptero|Early Cretaceous:112|7,coast|3|o|skittish|d|fly|vM|3|beak|A pterosaur with a huge sail-like crest, perhaps brightly coloured.
Pterodaustro|ptero|Early Cretaceous:105|12,fresh|3|f|herd|d|fly|vM|3|beak|A filter-feeding pterosaur with a thousand bristle teeth, pink like a flamingo.
Tupandactylus|ptero|Early Cretaceous:112|7|3|o|skittish|d|fly|vO|4|beak|A pterosaur whose head crest was taller than its body.
# Mesozoic: marine reptiles and others
Mosasaurus|rept|Late Cretaceous:68|sea,reef|7|c|predator|a|aquatic|MB|3|jaws|A sea-going lizard as long as a whale, the terror of the last Mesozoic seas.
Tylosaurus|rept|Late Cretaceous:85|sea,reef|7|c|predator|a|aquatic|TB|3|jaws|A long-snouted mosasaur that rammed its prey.
Globidens|rept|Late Cretaceous:80|reefw|5|c|territorial|a|aquatic|Gb|4|jaws|A mosasaur with ball-shaped teeth for crushing shellfish.
Elasmosaurus|rept|Late Cretaceous:80|sea|6|p|territorial|a|aquatic|Eb|3|jaws|A plesiosaur whose neck alone was longer than a bus.
Plesiosaurus|rept|Early Jurassic:190|sea,reef|5|p|skittish|a|aquatic|Pb|2|jaws|A long-necked, four-flippered marine reptile, the shape of the lake monster legends.
Kronosaurus|rept|Early Cretaceous:110|sea,reef|7|c|predator|a|aquatic|KB|4|jaws|A short-necked pliosaur with a skull as long as a man is tall.
Liopleurodon|rept|Late Jurassic:160|sea,reef|7|c|predator|a|aquatic|LB|4|jaws|A pliosaur that hunted by smell through the Jurassic seas.
Ichthyosaurus|rept|Early Jurassic:190|sea|4|p|skittish|a|aquatic|IB|2|jaws|A dolphin-shaped reptile with huge eyes that gave birth at sea.
Shonisaurus|rept|Late Triassic:215|sea|7|c|passive|a|aquatic|SB|4|jaws|A giant ichthyosaur as long as a sperm whale.
Ophthalmosaurus|rept|Late Jurassic:160|sea,reef|5|p|skittish|a|aquatic|Ob|3|jaws|An ichthyosaur with eyes as large as grapefruit for hunting in the deep.
Nothosaurus|rept|Middle Triassic:240|coast,reef|4|p|territorial|a||NB|3|teeth|A seal-like reptile with needle teeth that hunted from the shore.
Placodus|rept|Middle Triassic:240|reef,coast|4|c|passive|a|aquatic|Pw|3|teeth|A shell-crushing reptile that plucked molluscs off the sea floor.
Archelon|rept|Late Cretaceous:75|sea|6|c|passive|a|aquatic armored|AB|3|beak|The largest sea turtle ever, as long as a rowing boat.
Sarcosuchus|rept|Early Cretaceous:112|fresh,12|7|c|predator|a||SG|4|jaws|The super-croc, a crocodile as long as a bus that ambushed dinosaurs at the water's edge.
Deinosuchus|rept|Late Cretaceous:80|fresh,12,coast|7|c|predator|a||DG|4|jaws|A giant alligator relative with teeth the size of bananas.
Dakosaurus|rept|Late Jurassic:150|sea|5|c|predator|a|aquatic|DB|4|jaws|A sea crocodile with flippers and a tail fin, nicknamed Godzilla.
Postosuchus|rept|Late Triassic:220|4,12|5|c|predator|a||PR|3|jaws|A heavy Triassic predator, cousin of crocodiles, that rivalled the first dinosaurs.
Desmatosuchus|rept|Late Triassic:215|4,12|5|h|passive|d|armored|Dw|3|spikes|An armoured plant-eater with huge spikes over its shoulders.
Tanystropheus|rept|Middle Triassic:242|coast,fresh|4|p|passive|a||Tw|4|jaws|A reptile whose neck was longer than its body and tail together.
Lystrosaurus|syn|Early Triassic:250|1,4,12|3|h|herd|d||lw|2|tusks|A pig-like mammal relative that survived the Great Dying and spread across the world.
Ammonite|moll|Late Cretaceous:70|reef,sea|2|c|passive|a|aquatic armored @tentacled|nW|1|arms|A coiled, chambered cephalopod whose shells were taken for snakes turned to stone.
Belemnite|moll|Early Jurassic:180|sea,reef|2|c|skittish|a|aquatic @tentacled|nK|2|arms|A squid-like animal whose bullet-shaped guards are called thunderbolts.
Xiphactinus|fish|Late Cretaceous:85|sea|6|p|predator|a|aquatic|XY|3|jaws|A huge bulldog fish, found fossilised with a whole fish it had swallowed.
Leedsichthys|fish|Middle Jurassic:165|sea|7|f|passive|a|aquatic|LY|4|mouth|The largest bony fish ever, a gentle plankton-eating giant.
Cretoxyrhina|fish|Late Cretaceous:90|sea|6|c|predator|a|aquatic|FY|3|teeth|The Ginsu shark, a fast slashing hunter of the Cretaceous inland sea.
Morganucodon|mam|Early Jurassic:200|6,4|1|i|skittish|n||rw|3|bite|A shrew-like early mammal that hunted insects in the shadows of the dinosaurs.
Repenomamus|mam|Early Cretaceous:125|6|2|c|skittish|n||rK|4|bite|A badger-sized mammal found with a baby dinosaur in its stomach.
Beelzebufo|amph|Late Cretaceous:70|12,fresh|2|c|territorial|n||fG|4|bite|The devil frog, a giant armoured frog that may have eaten baby dinosaurs.
Meganeuropsis|ins|Early Permian:280|12,fresh|2|c|skittish|d|fly @avian|vG|4|mandibles|The largest insect ever, a griffinfly with a wingspan wider than a crow's.
# Paleozoic
Dimetrodon|syn|Early Permian:285|1,12,4|4|c|predator|d||DR|2|teeth|A sail-backed predator, closer to mammals than to dinosaurs, with two kinds of teeth.
Edaphosaurus|syn|Early Permian:290|12,4|4|h|herd|d||Ew|3|teeth|A sail-backed plant-eater whose sail bore knobby crossbars.
Inostrancevia|syn|Late Permian:255|4,12|5|c|predator|a||IR|4|sabres|One of the largest gorgonopsians, a sabre-toothed hunter of the Permian.
Gorgonops|syn|Late Permian:255|1,4|4|c|predator|a||GR|3|sabres|A gorgonopsian, a dog-like sabre-toothed predator of the age before the dinosaurs.
Moschops|syn|Middle Permian:265|4,12|5|h|herd|d||Mw|3|skull|A stocky plant-eater with a thick skull that butted heads with rivals.
Estemmenosuchus|syn|Middle Permian:267|12,6|5|o|territorial|d||Ew|4|horns|A big-headed beast with strange horn-like bosses on its skull.
Scutosaurus|rept|Late Permian:260|1,12|5|h|passive|d|armored|Sw|3|teeth|A squat, armoured plant-eater with a spiked face.
Diplocaulus|amph|Early Permian:280|fresh,12|3|c|passive|a|aquatic|aY|3|jaws|An amphibian with a boomerang-shaped head.
Eryops|amph|Early Permian:295|fresh,12|4|c|territorial|a||AG|3|jaws|A heavy, crocodile-like amphibian of Permian swamps.
Prionosuchus|amph|Middle Permian:270|fresh|6|c|predator|a|aquatic|PG|4|jaws|The largest amphibian ever, a gharial-shaped giant of tropical rivers.
Arthropleura|myria|Late Carboniferous:300|12,7,cave|5|h|passive|a|armored|Aw|3|mandibles|A millipede as long as a car that crawled the floors of coal forests.
Meganeura|ins|Late Carboniferous:300|12,fresh,cave|2|c|skittish|d|fly @avian|vG|3|mandibles|A griffinfly the size of a hawk, possible only in the oxygen-rich air of the coal age.
Pulmonoscorpius|arach|Early Carboniferous:340|12,cave|3|c|territorial|a|poison|SR|3|sting|A scorpion as long as a forearm that hunted the first land forests.
Hylonomus|rept|Late Carboniferous:312|12,7|1|i|skittish|d||lw|3|bite|One of the first reptiles, a small lizard-like creature found inside fossil tree stumps.
Tiktaalik|fish|Late Devonian:375|fresh,12|3|c|passive|a||Tg|3|jaws|A fish with a neck and wrists that propped itself up in the shallows.
Ichthyostega|amph|Late Devonian:365|fresh,12|3|c|passive|a||Ig|3|jaws|One of the first four-legged animals, half fish and half salamander.
Acanthostega|amph|Late Devonian:365|fresh|2|c|passive|a|aquatic|ag|3|jaws|An early tetrapod with eight fingers on each hand that rarely left the water.
Dunkleosteus|fish|Late Devonian:375|sea,reef|6|c|predator|a|aquatic armored|DK|3|blade jaws|An armoured fish with self-sharpening jaws of bone that bit like a guillotine.
Helicoprion|fish|Early Permian:280|sea|5|c|predator|a|aquatic|HK|4|tooth whorl|A shark-relative with a spiral saw of teeth in its lower jaw.
Stethacanthus|fish|Late Devonian:370|reef|3|c|skittish|a|aquatic|fK|3|teeth|A small shark with an anvil-shaped crest of spines on its back.
Cladoselache|fish|Late Devonian:370|sea|3|c|skittish|a|aquatic|fw|2|teeth|One of the earliest true sharks, sleek and fast.
Rhizodus|fish|Early Carboniferous:340|fresh,12|6|c|predator|a|aquatic|Rw|4|fangs|A lobe-finned fish with fangs as long as fingers that hunted early tetrapods.
Hyneria|fish|Late Devonian:365|fresh|6|c|predator|a|aquatic|Hw|4|jaws|A giant lobe-fin that may have snatched animals from the riverbank.
Coccosteus|fish|Middle Devonian:385|reef,fresh|2|c|skittish|a|aquatic armored|fw|2|blade jaws|A small armoured placoderm fish.
Eurypterus|arth|Silurian:420|reef,fresh|3|c|skittish|a|aquatic|kw|2|pincers|A sea scorpion of lagoons and brackish seas.
Jaekelopterus|arth|Early Devonian:390|fresh,reef|6|c|predator|a|aquatic armored|JR|4|pincers|The largest arthropod ever, a sea scorpion longer than a man with toothed pincers.
Pterygotus|arth|Silurian:425|reef,sea|5|c|predator|a|aquatic armored|Pr|3|pincers|A giant sea scorpion with long spiked claws.
Trilobite|arth|Cambrian:500|reef|1|s|passive|a|aquatic armored|tw|1|spines|A segmented sea-floor arthropod with crystal eyes, among the most successful animals ever.
Walliserops|arth|Early Devonian:400|reef|1|s|passive|a|aquatic armored|tO|3|trident|A spiny trilobite with a three-pronged fork on its head.
Anomalocaris|arth|Cambrian:508|reef,sea|3|c|territorial|a|aquatic|AR|2|grasping arms|The odd shrimp, a swimming hunter with two spiked arms before a round mouth, top predator of the Cambrian sea.
Opabinia|arth|Cambrian:508|reef|1|c|passive|a|aquatic|oM|3|claw trunk|A five-eyed creature with a clawed trunk that made scientists laugh when first shown.
Hallucigenia|misc|Cambrian:508|reef|1|s|passive|a|aquatic @insect|hR|3|spines|A worm with spines on its back and legs below, once reconstructed upside down.
Marrella|arth|Cambrian:508|reef|1|s|passive|a|aquatic|mM|2|legs|The lace crab, a tiny swimming arthropod with sweeping horns, commonest of its age.
Pikaia|misc|Cambrian:508|reef|1|f|skittish|a|aquatic @fish|pY|3|mouth|A small swimming worm with a stiff rod down its back, an early cousin of all backboned animals.
Wiwaxia|moll|Cambrian:508|reef|1|h|passive|a|aquatic armored|wO|3|spines|A slug-like animal covered in scales and tall spines.
Orthoceras|moll|Ordovician:460|reef,sea|2|c|passive|a|aquatic armored @tentacled|nw|2|arms|A straight-shelled nautiloid whose fossils are polished into ornaments.
Cameroceras|moll|Ordovician:470|reef,sea|6|c|predator|a|aquatic armored @tentacled|CY|4|arms|A giant straight-shelled nautiloid, the largest predator of the Ordovician sea.
Sacabambaspis|fish|Ordovician:450|reef,coast|2|f|passive|a|aquatic armored|fw|3|mouth|A jawless armoured fish with forward-facing eyes and a fixed grin.
Crinoid|echi|Silurian:430|reef|2|f|passive|a|aquatic|eW|1|arms|A sea lily, a stalked echinoderm that filtered the water with feathery arms.
Dickinsonia|misc|Ediacaran:560|reef|2|f|passive|a|aquatic|dO|3|none|A flat, quilted oval that lay on the microbial mats of the Ediacaran sea floor.
Charnia|misc|Ediacaran:570|reef|2|f|passive|a|aquatic|cG|3|none|A frond-shaped organism that stood rooted in the deep sea floor, first found by a schoolboy.
Spriggina|misc|Ediacaran:555|reef|1|s|passive|a|aquatic|sO|3|none|A segmented creature with a horseshoe-shaped head, perhaps an early arthropod.
Kimberella|moll|Ediacaran:555|reef|1|h|passive|a|aquatic|kw|3|rasp|A slug-like animal that left fan-shaped scratch marks as it grazed the mats.
Tribrachidium|misc|Ediacaran:555|reef|1|f|passive|a|aquatic|tY|4|none|A disc with three spiral arms, unlike any animal living.
Cyclomedusa|misc|Ediacaran:600|reef|1|f|passive|a|aquatic|cY|3|none|A ringed disc once taken for a jellyfish, perhaps the holdfast of a frond.
Pteridinium|misc|Ediacaran:550|reef|2|f|passive|a|aquatic|pG|4|none|A three-vaned quilted organism that grew buried in the sand.
`)
