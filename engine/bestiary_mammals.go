package engine

// Bestiary catalogue: living mammals and the recently extinct ones.
// name|group|when|habitat|size|diet|temper|activity|flags|glyph+colour|rarity|attack|description
var _ = bsReg(`
# deer, antelope, wild cattle
Red deer|mam|L|6,8,9,4|4|h|herd|c||dw|1|antlers|A tall deer of open woods; in autumn the stags roar and clash antlers over the hinds.
Roe deer|mam|L|6,4,8|3|h|skittish|c||dw|1|antlers|A small dainty deer that barks like a dog when startled and vanishes into the brush.
Fallow deer|mam|L|6,4|4|h|herd|c||dW|2|antlers|A spotted deer with broad palmate antlers, long kept in parks and hunting grounds.
White-tailed deer|mam|L|6,4,9|4|h|herd|c||dw|1|antlers|It flashes the white underside of its tail as an alarm before bounding away.
Mule deer|mam|L|4,2,mtn|4|h|herd|c||dw|2|antlers|Named for its huge ears, it bounds on all four legs at once over rough ground.
Sika deer|mam|L|6,8|4|h|herd|c||dW|2|antlers|A spotted deer of eastern islands whose stags give a piercing, whistling call.
Sambar|mam|L|5,7|4|h|skittish|n||dw|2|antlers|A large dark deer of tropical forests that stamps and honks to warn of tigers.
Chital|mam|L|3,5|4|h|herd|d||dO|2|antlers|A gold-and-white spotted deer that follows langur monkeys to eat the fruit they drop.
Muntjac|mam|L|5,6,7|2|o|skittish|c||dO|2|tusks|A tiny barking deer with short tusks and a habit of browsing alone.
Water deer|mam|L|12,6|3|h|skittish|c||dw|3|tusks|An antlerless deer whose bucks bear long fang-like tusks for duelling.
Moose|mam|L|9,12|5|h|territorial|c||Mw|2|antlers|The largest deer, long-legged and broad-antlered, and more dangerous than any bear in the rut.
Elk|mam|L|9,4,6|5|h|herd|c||Ew|2|antlers|A great deer of the northern forests whose bugling echoes through autumn valleys.
Reindeer|mam|L|10,9|4|h|herd|a||dY|1|antlers|Both sexes carry antlers; vast herds migrate across the tundra following the lichen.
Pere David's deer|mam|L|12|4|h|herd|d||dw|4|antlers|A marsh deer with backward antlers, saved from extinction in a single imperial park.
Pudu|mam|L|8|2|h|skittish|c||dr|3|antlers|The smallest deer, barely knee-high, hiding in the cold rainforests of the far south.
Pronghorn|mam|L|4,2|4|h|herd|d||aO|2|horns|The fastest runner of its continent, built to outrun cheetahs that no longer exist.
Springbok|mam|L|1,3|3|h|herd|d||aO|1|horns|It leaps stiff-legged high into the air, pronking, to show predators it is fit.
Thomson's gazelle|mam|L|3|3|h|herd|d||aO|1|horns|A small gazelle of the savanna with a black side stripe and a twitching tail.
Dorcas gazelle|mam|L|1|3|h|herd|d||aY|2|horns|A desert gazelle that can live its whole life without drinking.
Saiga|mam|L|2,4|3|h|herd|d||aY|3|horns|A steppe antelope with a strange bulbous nose that warms winter air and filters summer dust.
Impala|mam|L|3|4|h|herd|d||aO|1|horns|A graceful red antelope that scatters in leaps of ten paces when a predator charges.
Blue wildebeest|mam|L|3,4|5|h|herd|d||Wb|1|horns|A bearded, horned antelope whose herds migrate in their millions across the plains.
Hartebeest|mam|L|3|4|h|herd|d||aw|2|horns|A long-faced antelope with ringed horns, often standing sentinel on termite mounds.
Topi|mam|L|3|4|h|herd|d||ar|2|horns|A reddish antelope that poses on mounds and is said to bluff alarm calls to keep mates.
Common eland|mam|L|3,4|5|h|herd|d||Ew|2|horns|The largest antelope, heavy as an ox yet able to clear a fence from a standstill.
Greater kudu|mam|L|3,5|5|h|skittish|c||Kw|2|horns|A grey-striped antelope whose males carry long spiralling horns.
Bongo|mam|L|7|4|h|skittish|n||Kr|4|horns|A shy chestnut forest antelope with white stripes and lyre-shaped horns.
Sitatunga|mam|L|12|4|h|skittish|c||aw|3|horns|A marsh antelope with splayed hooves that hides underwater with only its nostrils showing.
Gerenuk|mam|L|3,1|3|h|skittish|d||aO|3|horns|A long-necked gazelle that stands on its hind legs to browse thorn trees.
Oryx|mam|L|1,3|4|h|herd|d||AY|2|horns|A white desert antelope with rapier horns, perhaps the root of the unicorn tale.
Addax|mam|L|1|4|h|herd|n||AY|4|horns|A pale desert antelope with twisted horns that never needs to drink.
Sable antelope|mam|L|3,5|5|h|territorial|d||Ak|3|horns|A black antelope with great curved horns that will face down a lion.
Klipspringer|mam|L|3,mtn|2|h|skittish|d||ao|3|horns|A tiny antelope that stands on the tips of its hooves on bare rock.
Dik-dik|mam|L|3|2|h|skittish|c||ao|2|horns|A tiny antelope that mates for life and marks its territory with tears from its eye glands.
Nilgai|mam|L|3,4|5|h|herd|d||Ab|2|horns|The blue bull of the plains, the largest antelope of its subcontinent.
Blackbuck|mam|L|4,3|3|h|herd|d||ak|3|horns|A black-and-white antelope with corkscrew horns, sacred to many villages.
Chamois|mam|L|mtn,6,9|3|h|skittish|d||aw|2|horns|A goat-antelope of high crags with small hooked horns.
Alpine ibex|mam|L|mtn|4|h|herd|d||Iw|2|horns|A mountain goat with huge ridged horns that climbs sheer dams to lick the salt.
Markhor|mam|L|mtn|4|h|skittish|d||IK|4|horns|A wild goat with corkscrew horns as long as a man's arm.
Mountain goat|mam|L|mtn,10|4|h|skittish|d||IY|2|horns|A shaggy white goat of the high cliffs with black dagger horns.
Bighorn sheep|mam|L|mtn,2|4|h|herd|d||Iw|2|horns|Rams crash their curled horns together with a crack heard a mile away.
Mouflon|mam|L|mtn,6|3|h|herd|d||ir|3|horns|A wild sheep with curled horns, ancestor of domestic flocks.
Argali|mam|L|mtn,2|5|h|herd|d||Iw|3|horns|The largest wild sheep, carrying horns heavier than the rest of its skeleton.
Takin|mam|L|mtn,8|5|h|herd|d||TW|4|horns|A stocky golden goat-antelope of misty bamboo forests, said to be the golden fleece.
Musk ox|mam|L|10,11|5|h|herd|d||Mw|2|horns|When wolves come the herd forms a ring of lowered horns around the calves.
American bison|mam|L|4,9|6|h|herd|d||Bw|2|horns|A humped shaggy giant whose herds once darkened the plains from horizon to horizon.
European bison|mam|L|6|6|h|herd|d||Bw|4|horns|The wisent, last wild ox of the old forests, brought back from a handful of captives.
African buffalo|mam|L|3,12|5|h|territorial|a||Bk|2|horns|Known as the black death, it circles back to ambush the hunter who wounded it.
Water buffalo|mam|L|12,5|5|h|territorial|d||Bk|2|horns|A massive grey buffalo that wallows in mud and will charge a tiger.
Gaur|mam|L|5,7|6|h|herd|c||Bk|3|horns|The largest wild cattle, a black bull with white stockings and a ridged back.
Banteng|mam|L|5|5|h|skittish|c||Bo|4|horns|A wild ox of tropical forests with white rump and stockings.
Yak|mam|L|mtn,10|5|h|herd|d||BK|2|horns|A long-haired ox of the high plateaus that can graze where the air is thin.
Anoa|mam|L|7|3|h|skittish|d||ak|4|horns|A dwarf buffalo of island rainforests, short-tempered for its size.
Wild boar|mam|L|6,5,4|4|o|territorial|n||qw|1|tusks|A bristled wild pig that roots up the forest floor and charges when cornered.
Warthog|mam|L|3|3|o|skittish|d|ivory|qK|1|tusks|It kneels to graze and backs into its burrow so its tusks face the door.
Babirusa|mam|L|7|3|o|skittish|d|ivory|qK|4|tusks|A pig of island forests whose upper tusks grow through the snout and curl back towards the face.
Giant forest hog|mam|L|7|4|o|territorial|n||qk|4|tusks|The largest wild pig, black-haired and heavy as a man.
Collared peccary|mam|L|1,5,3|3|o|herd|a||qK|2|tusks|A pig-like hoofed animal that lives in noisy musky bands.
Pygmy hippopotamus|mam|L|7,12|4|h|skittish|n||Hm|4|tusks|A shy forest hippo the size of a large pig that sweats a pink, sun-blocking oil.
Hippopotamus|mam|L|12,3,fresh|6|h|territorial|n|ivory|HM|2|tusks|Barrel-bodied and quick to anger, it kills more people than any lion.
Giraffe|mam|L|3|6|h|herd|d||GO|2|horns|The tallest animal, browsing acacia crowns with a tongue the length of a forearm.
Okapi|mam|L|7|5|h|skittish|d||Gr|4|horns|A forest giraffe with zebra-striped legs, unknown to science until the age of steam.
Dromedary|mam|L|1|5|h|herd|d||CO|2|teeth|A one-humped camel that can lose a quarter of its weight to thirst and drink it back in minutes.
Bactrian camel|mam|L|2|5|h|herd|d||Cw|3|teeth|A two-humped camel of cold deserts that eats snow when there is no water.
Guanaco|mam|L|mtn,2,4|4|h|herd|d||aO|2|teeth|A wild ancestor of the llama that spits at rivals and drinks salt water.
Vicuna|mam|L|mtn,10|3|h|herd|d||aO|3|teeth|A slender wild camel of the high puna whose fleece was reserved for kings.
Plains zebra|mam|L|3|5|h|herd|d||ZY|1|teeth|Its stripes confuse biting flies and dazzle lions in a fleeing herd.
Grevy's zebra|mam|L|3,1|5|h|herd|d||ZY|3|teeth|The largest zebra, narrow-striped with big round ears.
Przewalski's horse|mam|L|2,4|5|h|herd|d||hO|3|teeth|The last truly wild horse, stocky and dun with an upright mane.
African wild ass|mam|L|1|4|h|skittish|d||hK|4|teeth|A grey ass of stony deserts with striped legs, ancestor of the donkey.
Kiang|mam|L|mtn,2|5|h|herd|d||hr|3|teeth|The largest wild ass, roaming the high plateau in tight groups.
Tapir|mam|L|7,12|5|h|skittish|n||Tk|3|snout|A shy forest browser with a short trunk that flees into water when chased.
White rhinoceros|mam|L|3|6|h|herd|d||Rk|4|horn|A square-lipped grazing rhino, the largest of its kind.
Black rhinoceros|mam|L|3,1|6|h|territorial|c||Rk|4|horn|A hook-lipped browser with poor eyes and a short temper.
Indian rhinoceros|mam|L|12,4|6|h|territorial|c||RK|4|horn|A one-horned rhino whose folds of skin look like riveted armour.
Sumatran rhinoceros|mam|L|7|5|h|skittish|c||Rr|5|horn|The smallest and hairiest rhino, a living cousin of the woolly rhinoceros.
African bush elephant|mam|L|3,5|6|h|herd|a|ivory|EK|2|tusks|The largest land animal, led by an old matriarch who remembers every water hole.
African forest elephant|mam|L|7|6|h|skittish|a|ivory|Ek|4|tusks|A smaller elephant of the deep forest with straight tusks, gardener of the jungle.
Asian elephant|mam|L|5,7|6|h|herd|a|ivory|EK|3|tusks|A small-eared elephant long tamed for war, timber and temple processions.
# carnivores
Lion|mam|L|3|5|c|pack|n||lO|2|claws|A maned cat that hunts in prides and roars to be heard five miles off.
Tiger|mam|L|5,7,9|5|c|predator|n||TO|3|claws|The largest cat, a solitary striped ambusher of forest and reed bed.
Leopard|mam|L|3,5,7,mtn|4|c|predator|n||lW|2|claws|A rosetted cat that hauls its kills into trees out of reach of lions.
Jaguar|mam|L|7,12|4|c|predator|n||lW|3|claws|A heavy spotted cat that kills by biting through the skull.
Snow leopard|mam|L|mtn,10|4|c|predator|c||lY|4|claws|A pale long-tailed cat of the high mountains, seldom seen even by those who live there.
Cheetah|mam|L|3,1|4|c|skittish|d||lW|3|claws|The fastest land animal; it loses most of its kills to bigger predators.
Cougar|mam|L|mtn,6,9,1|4|c|predator|c||lO|2|claws|A tawny cat of mountains and forests from the far north to the far south.
Eurasian lynx|mam|L|9,6|3|c|skittish|c||lw|3|claws|A tufted-eared cat with huge paws for snow, a hunter of roe deer.
Bobcat|mam|L|6,1,4|3|c|skittish|n||lw|2|claws|A bob-tailed wildcat that hunts rabbits at the edges of fields.
Caracal|mam|L|1,3|3|c|skittish|n||lO|3|claws|A red cat with black ear tufts that leaps to snatch birds from the air.
Serval|mam|L|3,12|3|c|skittish|c||lW|3|claws|A long-legged spotted cat that hunts by sound in tall grass.
Ocelot|mam|L|7,5|2|c|skittish|n||lW|3|claws|A small spotted cat of the jungle floor.
Clouded leopard|mam|L|7|3|c|skittish|n||lK|4|claws|A tree cat with the longest canines for its size of any living cat.
Wildcat|mam|L|6,1,4|2|c|skittish|n||lw|2|claws|The wild ancestor of the house cat, striped and fierce.
Sand cat|mam|L|1|2|c|skittish|n||lY|4|claws|A tiny desert cat with furred soles that hunts vipers by night.
Pallas's cat|mam|L|2,mtn|2|c|skittish|c||lK|4|claws|A stocky, flat-faced steppe cat with a permanently grumpy look.
Fishing cat|mam|L|12,fresh|3|p|skittish|n||lK|4|claws|A webbed-footed cat that scoops fish from the shallows.
Grey wolf|mam|L|9,6,10,4|4|c|pack|n||Cy|2|bite|A pack hunter that runs its prey down over miles of snow.
Arctic wolf|mam|L|10,11|4|c|pack|a||CY|3|bite|A white wolf of the high Arctic that may never have learned to fear people.
Ethiopian wolf|mam|L|mtn|3|c|skittish|d||cO|5|bite|A slender red wolf of the highlands that hunts giant mole-rats alone.
Red wolf|mam|L|12,6|4|c|pack|n||Cr|5|bite|A rare tawny wolf of coastal swamps.
Coyote|mam|L|4,1,6|3|o|skittish|n||cw|1|bite|A clever wild dog whose yipping chorus carries across the plains.
Golden jackal|mam|L|3,1,4|3|o|skittish|n||cO|2|bite|A scavenger and opportunist that follows lions and people alike.
Dhole|mam|L|5,7,mtn|3|c|pack|d||cr|3|bite|A whistling red dog that hunts in clans large enough to drive off tigers.
African wild dog|mam|L|3|3|c|pack|c||cW|3|bite|A painted dog whose packs share every kill, feeding the old and the young first.
Maned wolf|mam|L|3,4|4|o|skittish|n||cr|4|bite|A fox on stilts that eats as much fruit as meat.
Red fox|mam|L|6,4,9,10|2|o|skittish|n||cr|1|bite|A russet hunter that pounces on mice under the snow.
Arctic fox|mam|L|10,11|2|o|skittish|a||cY|2|bite|Its coat turns white in winter; it trails polar bears for scraps.
Fennec fox|mam|L|1|1|o|skittish|n||cY|3|bite|The smallest fox, with ears longer than its face to shed desert heat.
Raccoon dog|mam|L|6,12|2|o|skittish|n||cK|3|bite|A fox-like dog that is the only canine to sleep through winter.
Spotted hyena|mam|L|3|4|c|pack|n||yW|2|jaws|A clan hunter with jaws strong enough to crack bone; its whoops sound like laughter.
Striped hyena|mam|L|1,3|4|s|skittish|n||yy|3|jaws|A shaggy, solitary scavenger of dry hills that raises its mane to look bigger.
Aardwolf|mam|L|3|2|i|skittish|n||yW|4|bite|A small hyena that eats almost nothing but termites.
Brown bear|mam|L|9,6,mtn,10|5|o|predator|a||Bw|2|claws|A humped bear that fattens on salmon and berries before its winter sleep.
Polar bear|mam|L|11,ice,10|6|c|predator|a||BY|4|claws|The largest land carnivore, a white hunter of seals on the sea ice.
American black bear|mam|L|6,9|5|o|territorial|n||Bk|2|claws|A forest bear that climbs trees and raids berry patches and honey.
Asian black bear|mam|L|6,mtn|4|o|territorial|n||Bk|3|claws|A black bear with a white crescent on its chest.
Sun bear|mam|L|7|3|o|territorial|d||BK|4|claws|The smallest bear, with a tongue long enough to lick honey from bee nests.
Sloth bear|mam|L|5,3|4|i|territorial|n||Bk|4|claws|A shaggy bear that vacuums termites through a gap in its teeth.
Spectacled bear|mam|L|mtn,8|4|h|skittish|c||Bk|4|claws|The bear of the cloud forest, ringed pale around the eyes.
Giant panda|mam|L|mtn,6|5|h|passive|a||PY|5|bite|A black-and-white bear that eats bamboo for half of every day.
Red panda|mam|L|mtn,6|2|h|skittish|c||pr|4|claws|A ruddy tree-dweller with a ringed tail and a false thumb for gripping bamboo.
Wolverine|mam|L|9,10|3|c|territorial|a||wk|4|claws|A small, savage weasel that drives bears from their kills.
European badger|mam|L|6,4|3|o|territorial|n|burrow|bK|2|claws|A burrowing badger whose setts may be centuries old.
Honey badger|mam|L|3,1|3|o|territorial|a|armored|bK|3|claws|Loose skin and no fear: it shrugs off bee stings and cobra bites.
Eurasian otter|mam|L|fresh,6,12|3|p|skittish|n||oW|2|bite|A sleek river hunter that marks its banks with sweet-smelling spraint.
Giant otter|mam|L|fresh,7|4|p|territorial|d||Ow|4|bite|A wolf of the river; families of these otters will mob a caiman.
Sea otter|mam|L|coast,seac|3|c|skittish|d||ow|3|paws|It floats on its back and cracks sea urchins on a stone balanced on its chest.
Stoat|mam|L|6,9,10|1|c|skittish|a||ew|2|bite|A slender weasel that turns white in winter; the ermine of royal robes.
Pine marten|mam|L|9,6|2|c|skittish|n||ew|2|bite|An agile tree weasel that hunts squirrels through the canopy.
Fisher|mam|L|9|3|c|skittish|n||ek|3|bite|A forest weasel and one of the few predators that hunts porcupines.
Striped skunk|mam|L|6,4|2|o|passive|n|poison|sk|2|bite|It stamps its feet in warning before spraying a reek that lasts for days.
Meerkat|mam|L|1,3|1|i|skittish|d|burrow|mY|2|bite|A mongoose that posts sentries on its hind legs to watch for eagles.
Indian grey mongoose|mam|L|3,5|2|c|skittish|d||mw|2|bite|A mongoose famous for its duels with cobras.
Fossa|mam|L|7|3|c|predator|c||fO|4|claws|A long-tailed island hunter, half cat and half mongoose, that stalks lemurs.
Binturong|mam|L|7|3|o|passive|n||bk|4|claws|A bearcat with a gripping tail that smells of buttered popcorn.
Common genet|mam|L|3,6|2|c|skittish|n||gW|2|bite|A spotted, ringed-tail climber that hunts birds by night.
Ringtail|mam|L|1,mtn|1|o|skittish|n||rW|3|bite|A small ringed-tailed cousin of the raccoon, kept by miners to catch mice.
Raccoon|mam|L|6,12|2|o|skittish|n||rK|1|claws|A masked forager with nimble hands that washes its food in streams.
Coati|mam|L|7,5|2|o|herd|d||rw|2|claws|A long-snouted relative of the raccoon that roams the forest in noisy troops.
Kinkajou|mam|L|7|2|h|passive|n||kO|3|claws|A honey bear of the canopy that drinks nectar with a long tongue.
# marine mammals
Harbour seal|mam|L|coast,sea|4|p|skittish|a||sK|1|bite|A spotted seal that hauls out on sandbars and watches people with dark eyes.
Grey seal|mam|L|coast,seac|4|p|skittish|a||sK|2|bite|A large seal with a long Roman nose whose moans were taken for drowned sailors.
Leopard seal|mam|L|seac,ice|5|c|predator|a|aquatic|SK|4|bite|A spotted hunter of the southern ice that ambushes penguins at the floe edge.
Elephant seal|mam|L|coast,seac|6|p|territorial|a||SK|3|bite|Bulls with inflatable trunks fight bloody battles for beaches of females.
Walrus|mam|L|coast,ice,seac|6|c|territorial|a|ivory|WK|3|tusks|A tusked giant of the Arctic that hauls itself onto ice with its teeth.
California sea lion|mam|L|coast,sea|4|p|skittish|d||sw|2|bite|A barking sea lion that crowds harbour piers.
Steller sea lion|mam|L|coast,seac|5|p|territorial|d||Sw|3|bite|The largest sea lion, its bulls maned like a lion.
West Indian manatee|mam|L|fresh,seaw,coast|5|h|passive|d|aquatic|Mc|4|snout|A slow sea cow that grazes on sea grass, perhaps the mermaid of sailors' tales.
Dugong|mam|L|seaw,reefw|5|h|passive|d|aquatic|Mc|4|snout|A sea cow with a whale's tail that grazes the sea meadows.
Bottlenose dolphin|mam|L|sea,reef|4|p|skittish|a|aquatic|DB|1|teeth|A playful dolphin that rides bow waves and calls companions by name.
Orca|mam|L|sea,seac|6|c|pack|a|aquatic|OK|3|teeth|The killer whale hunts in matrilineal pods with their own languages and tactics.
Narwhal|mam|L|seac,ice|5|p|skittish|a|aquatic ivory|NY|4|tusk|A pale whale whose single spiral tusk was sold as unicorn horn.
Beluga|mam|L|seac|5|p|skittish|a|aquatic|BY|3|teeth|The white canary of the sea, chirping and whistling under the ice.
Sperm whale|mam|L|sea|7|p|territorial|a|aquatic|WK|3|teeth|A square-headed whale that dives a mile deep to battle giant squid.
Humpback whale|mam|L|sea|7|f|passive|a|aquatic|Wb|2|flukes|A long-finned whale whose males sing songs that change from year to year.
Blue whale|mam|L|sea|7|f|passive|a|aquatic|WB|4|flukes|The largest animal that has ever lived, fed on clouds of tiny krill.
Fin whale|mam|L|sea|7|f|passive|a|aquatic|Wy|3|flukes|The greyhound of the sea, second only to the blue whale in size.
Grey whale|mam|L|sea,coast|7|f|passive|a|aquatic|WK|3|flukes|A barnacled whale that sifts mud from the sea floor for food.
Bowhead whale|mam|L|seac,ice|7|f|passive|a|aquatic|Wk|4|flukes|An Arctic whale that breaks ice with its skull and may live two hundred years.
Amazon river dolphin|mam|L|fresh,7|4|p|skittish|a|aquatic|DM|4|teeth|A pink dolphin of flooded forests said to take human form and court villagers.
Harbour porpoise|mam|L|seac,coast|3|p|skittish|a|aquatic|dK|2|teeth|A small shy porpoise of cold coasts, heard puffing before it is seen.
# primates
Chimpanzee|mam|L|7,5|4|o|territorial|d|@humanoid|Pk|3|fists|An ape that makes tools, hunts monkeys and wages war on neighbouring troops.
Bonobo|mam|L|7|4|o|skittish|d|@humanoid|Pk|4|fists|A peaceable ape whose troops are led by females.
Western gorilla|mam|L|7|5|h|territorial|d|@humanoid|GK|4|fists|The largest primate; a silverback beats his chest to warn rather than fight.
Bornean orangutan|mam|L|7|5|h|passive|d|@humanoid|Or|4|fists|A red ape that lives alone in the canopy and builds a fresh nest every night.
Lar gibbon|mam|L|7|2|h|skittish|d|@humanoid|pK|3|fists|A swinging ape whose duets with its mate ring through the dawn forest.
Mandrill|mam|L|7|4|o|territorial|d|@humanoid|PB|3|fangs|The largest monkey, with a blue-and-red face and fearsome canines.
Olive baboon|mam|L|3,mtn|3|o|herd|d|@humanoid|pw|1|fangs|A troop-living baboon that will mob a leopard.
Gelada|mam|L|mtn|3|h|herd|d|@humanoid|pr|3|fangs|A grass-eating monkey of the highlands with a bleeding-heart patch on its chest.
Japanese macaque|mam|L|6,mtn|3|o|herd|d|@humanoid|pr|3|bite|The snow monkey, which bathes in hot springs through the winter.
Rhesus macaque|mam|L|5,6,mtn|3|o|herd|d|@humanoid|pw|1|bite|A bold monkey that thrives in temples and towns.
Proboscis monkey|mam|L|12,7|3|h|herd|d|@humanoid|pO|4|bite|A pot-bellied swamp monkey whose males have huge drooping noses.
Golden snub-nosed monkey|mam|L|mtn,6|3|h|herd|d|@humanoid|pO|4|bite|A golden-furred monkey of snowy mountain forests.
Mantled howler monkey|mam|L|7|3|h|herd|c|@humanoid|pk|2|bite|Its roars at dawn are the loudest calls of any land animal.
Spider monkey|mam|L|7|3|h|skittish|d|@humanoid|pk|3|bite|A long-limbed monkey that swings by its gripping tail.
Capuchin|mam|L|7,5|2|o|skittish|d|@humanoid|pW|2|bite|A clever monkey that cracks nuts with stone hammers.
Common marmoset|mam|L|7,5|1|o|skittish|d|@humanoid|pY|2|bite|A tiny tufted monkey that gnaws holes in trees to drink the gum.
Ring-tailed lemur|mam|L|5,1|2|o|herd|d|@humanoid|pK|3|bite|A sunbathing lemur that waves its stinking tail in duels.
Aye-aye|mam|L|7|2|i|skittish|n|@humanoid|pk|5|bite|A lemur that taps on bark and hooks grubs out with a long skeletal finger.
Slow loris|mam|L|7|1|o|passive|n|@humanoid poison|pw|4|bite|A wide-eyed primate with a venomous bite, mixed from glands in its elbows.
Tarsier|mam|L|7|1|i|skittish|n|@humanoid|pw|4|bite|Each of its eyes is as large as its brain.
# rodents, rabbits, small mammals
Red squirrel|mam|L|9,6|1|h|skittish|d||rr|1|bite|A tufted-eared squirrel that buries seeds and forgets half of them, planting forests.
Grey squirrel|mam|L|6|1|h|skittish|d||ry|1|bite|A bold park squirrel that raids bird feeders.
Indian giant squirrel|mam|L|5|2|h|skittish|d||rm|3|bite|A squirrel the size of a cat, coloured maroon, cream and black.
Chipmunk|mam|L|6,9|1|h|skittish|d|burrow|rw|1|bite|A striped ground squirrel that stuffs its cheeks with seeds.
Alpine marmot|mam|L|mtn,10|2|h|skittish|d|burrow|rw|2|bite|A fat mountain squirrel that whistles at eagles and sleeps half the year.
Black-tailed prairie dog|mam|L|4|1|h|herd|d|burrow|rO|1|bite|It lives in towns of a thousand burrows and has calls for each kind of predator.
North American beaver|mam|L|fresh,9,6|3|h|skittish|n||Rw|2|teeth|An engineer that fells trees and dams streams into ponds.
Eurasian beaver|mam|L|fresh,9,6|3|h|skittish|n||Rw|2|teeth|The old-world beaver, back in rivers it vanished from centuries ago.
Capybara|mam|L|12,fresh,3|4|h|herd|c||Rw|2|teeth|The largest rodent, so placid other animals sit on its back.
Crested porcupine|mam|L|3,1,6|3|h|territorial|n|armored|Pw|2|quills|It rattles its quills and runs backwards into its attacker.
North American porcupine|mam|L|9,6|3|h|passive|n|armored|Pk|2|quills|A climbing porcupine with thirty thousand barbed quills.
Brown rat|mam|L|any,cave|1|o|skittish|n||rw|1|bite|The follower of people across every sea.
House mouse|mam|L|any|1|o|skittish|n||rK|1|bite|A tiny grey mouse that has spread wherever grain is stored.
Wood mouse|mam|L|6,4|1|o|skittish|n||rw|1|bite|A big-eared mouse of hedges and woods.
Brown lemming|mam|L|10|1|h|herd|a||rO|1|bite|Its numbers boom and crash every few years, feeding every predator of the tundra.
Desert jerboa|mam|L|1,2|1|h|skittish|n|burrow|jY|2|bite|A hopping mouse on kangaroo legs with a tasselled tail.
Naked mole-rat|mam|L|3,1|1|h|passive|a|burrow|rm|4|teeth|A hairless, cold-blooded mammal that lives in colonies ruled by a queen.
Agouti|mam|L|7|2|h|skittish|d||rO|2|teeth|A long-legged rodent and the only animal that can open a brazil nut.
Patagonian mara|mam|L|4,2|3|h|herd|d||rw|3|teeth|A long-legged rodent that looks like a small deer and mates for life.
Chinchilla|mam|L|mtn,2|1|h|skittish|c||rK|4|teeth|A mountain rodent with the densest fur of any land animal.
European hare|mam|L|4,6|2|h|skittish|c||hw|1|kick|It boxes in spring and can outrun a fox in open fields.
Snowshoe hare|mam|L|9|2|h|skittish|n||hY|1|kick|A hare that turns white in winter, with feet like snowshoes.
Arctic hare|mam|L|10,11|2|h|herd|a||hY|2|kick|A white hare of the high Arctic that gathers in groups of hundreds.
Black-tailed jackrabbit|mam|L|1,4|2|h|skittish|n||hw|1|kick|A desert hare whose huge ears shed heat.
European rabbit|mam|L|4,6|1|h|herd|c|burrow|hw|1|kick|A burrowing rabbit that lives in warrens and thumps the ground at danger.
American pika|mam|L|mtn|1|h|skittish|d||hw|3|bite|A tiny rabbit of the rock slides that dries hay for the winter.
European hedgehog|mam|L|6,4|1|i|passive|n|armored|nw|1|spines|It rolls into a ball of spines when afraid.
Common shrew|mam|L|6,4,12|1|i|skittish|a||nK|1|bite|It must eat every few hours or starve.
European mole|mam|L|6,4|1|i|passive|a|burrow|nk|2|claws|A velvet digger with spade hands that rarely comes to the surface.
Star-nosed mole|mam|L|12|1|i|skittish|a|burrow|nk|4|claws|A mole whose fleshy pink star of tentacles feels its way in the dark mud.
Nine-banded armadillo|mam|L|4,1,5|2|i|skittish|n|armored burrow|aK|2|claws|A banded armoured digger that can hold its breath and walk across river beds.
Giant armadillo|mam|L|3,7|3|i|passive|n|armored burrow|Ak|4|claws|The largest armadillo, with a scythe-like middle claw.
Giant anteater|mam|L|3,12|4|i|territorial|d||Ak|3|claws|It eats thirty thousand ants a day and can disembowel a jaguar with its claws.
Ground pangolin|mam|L|3|2|i|passive|n|armored|aw|4|claws|A scaly anteater that rolls into an armoured ball.
Aardvark|mam|L|3|4|i|skittish|n|burrow|Aw|3|claws|The earth pig, which digs a burrow faster than a man with a spade.
Three-toed sloth|mam|L|7|2|h|passive|a||sw|2|claws|So slow that algae grows in its fur.
Rock hyrax|mam|L|3,1,mtn|1|h|herd|d||hK|2|bite|A guinea-pig-like rock dweller whose nearest relatives are elephants.
Common vampire bat|mam|L|7,5,cave|1|c|skittish|n|fly @avian bleed|bk|2|fangs|A bat that runs on the ground and laps the blood of sleeping cattle.
Little brown bat|mam|L|6,9,cave|1|i|skittish|n|fly @avian|bw|1|bite|A tiny bat that eats its own weight in insects every night.
Greater horseshoe bat|mam|L|6,cave|1|i|skittish|n|fly @avian|bw|2|bite|A bat with a fleshy horseshoe on its nose that focuses its calls.
Indian flying fox|mam|L|5,7|2|h|skittish|n|fly @avian|bO|2|bite|A fruit bat with a wingspan wider than a man is tall.
Ghost bat|mam|L|1,cave|2|c|skittish|n|fly @avian|bY|4|bite|A pale carnivorous bat that eats other bats.
Colugo|mam|L|7|2|h|passive|n|fly|bK|4|claws|A gliding mammal wrapped in a cloak of skin from chin to tail.
# marsupials and monotremes
Red kangaroo|mam|L|1,3|5|h|herd|c||Kr|2|kick|The largest marsupial, hopping at speed on great hind feet.
Eastern grey kangaroo|mam|L|4,6|4|h|herd|c||KK|1|kick|Bucks box for mates, balancing on their tails to kick.
Red-necked wallaby|mam|L|6,4|3|h|skittish|c||kw|2|kick|A small kangaroo of open woodland.
Quokka|mam|L|6|2|h|passive|n||kw|3|bite|A small wallaby of a few islands that seems always to be smiling.
Koala|mam|L|6|2|h|passive|n||kK|3|claws|A tree-dweller that sleeps twenty hours a day on a diet of gum leaves.
Common wombat|mam|L|6,mtn|3|h|territorial|n|burrow|kw|2|bite|A burrowing marsupial with a backside of cartilage that blocks its tunnel.
Tasmanian devil|mam|L|6,8|2|c|territorial|n||kk|3|jaws|A screaming black scavenger with the strongest bite for its size of any mammal.
Virginia opossum|mam|L|6,12|2|o|passive|n||ky|1|bite|It plays dead when threatened, drooling and reeking.
Sugar glider|mam|L|6,7|1|h|skittish|n|fly|kK|3|bite|A pocket-sized marsupial that glides from tree to tree.
Numbat|mam|L|6|1|i|skittish|d||kr|4|bite|A striped termite-eater with a tongue half the length of its body.
Platypus|mam|L|fresh,6|2|i|passive|n|poison|nw|4|spur|An egg-laying mammal with a duck's bill and venomous spurs on its heels.
Short-beaked echidna|mam|L|6,1,4|2|i|passive|a|armored|nw|3|spines|A spiny egg-layer that finds ants by electric sense in its snout.
# recently extinct
Thylacine|mam|X1936|6,8|4|c|skittish|n||Cw|5|jaws|The striped Tasmanian tiger, a marsupial wolf that could open its jaws impossibly wide.
Aurochs|mam|X1627|6,4|6|h|territorial|d||Bk|4|horns|The great wild ox of the old forests, ancestor of all cattle; the last cow died in a royal forest.
Quagga|mam|X1883|3,4|5|h|herd|d||Zw|4|teeth|A zebra striped only at the front, fading to plain brown behind.
Steller's sea cow|mam|X1768|seac,coast|7|h|passive|a|aquatic|Mc|5|snout|A colossal kelp-eating sea cow, hunted out within thirty years of its discovery.
Caribbean monk seal|mam|X1952|coast,seaw|4|p|passive|a||sw|5|bite|A gentle warm-water seal, the first animal described by sea explorers of that ocean.
Bluebuck|mam|X1800|4|4|h|herd|d||ab|5|horns|A blue-grey antelope, the first large mammal of its continent lost in recorded times.
Tarpan|mam|X1909|4,6|5|h|herd|d||hK|5|teeth|The mouse-grey wild horse of the steppe and the forest edge.
Atlas bear|mam|X1870|mtn,1|5|o|predator|a||Bw|5|claws|A shaggy reddish bear of the desert mountains, captured for the arenas of the ancients.
Barbary lion|mam|X1922|mtn,1|5|c|predator|n||LO|5|claws|A dark-maned mountain lion once kept in royal menageries.
Javan tiger|mam|X1976|7|5|c|predator|n||TO|5|claws|A small, dark island tiger with thin dense stripes.
Caspian tiger|mam|X1970|6,4,12|5|c|predator|n||TO|5|claws|A great tiger of reed beds and river forests in the steppe.
Schomburgk's deer|mam|X1938|12|4|h|herd|d||dw|5|antlers|A swamp deer with basket-shaped antlers of many tines.
Falkland Islands wolf|mam|X1876|4,coast|3|c|passive|a||cw|5|bite|An island fox so tame it walked up to sailors and was killed for it.
Pig-footed bandicoot|mam|X1950|1,4|1|h|skittish|n||kw|5|bite|A long-legged marsupial with feet like little hooves.
Toolache wallaby|mam|X1943|4|3|h|skittish|d||kK|5|kick|Perhaps the most graceful of the kangaroos, faster than a greyhound.
Baiji|mam|X2006|fresh|4|p|skittish|a|aquatic|DY|5|teeth|A pale river dolphin, goddess of the great river in legend.
Bramble Cay melomys|mam|X2016|coast|1|h|skittish|n||rw|5|bite|A small rodent of a single sandy cay, drowned by rising seas.
Western black rhinoceros|mam|X2011|3|6|h|territorial|c||Rk|5|horn|A desert subspecies of black rhino, poached out within living memory.
Great Plains wolf|mam|X1926|4|4|c|pack|n||Cy|5|bite|The buffalo wolf that followed the great herds across the prairie.
`)
