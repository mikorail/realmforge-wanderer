package engine

// Bestiary catalogue: living birds and the recently extinct ones.
// name|group|when|habitat|size|diet|temper|activity|flags|glyph+colour|rarity|attack|description
var _ = bsReg(`
# ratites and big ground birds
Common ostrich|bird|L|3,1|5|o|herd|d||OK|2|kick|The largest living bird, flightless and fast, with a kick that can kill a lion.
Emu|bird|L|1,3,6|4|o|skittish|d||Ow|2|kick|A shaggy flightless bird that drums deep booming calls across the outback.
Southern cassowary|bird|L|7|4|o|territorial|d||Ob|3|claws|A helmeted rainforest bird with a dagger claw on each foot.
Greater rhea|bird|L|4,3|4|o|herd|d||Oy|2|kick|A flightless bird of the pampas whose males brood the eggs of many hens.
Brown kiwi|bird|L|8|2|i|skittish|n||vw|4|beak|A flightless night bird with nostrils at the tip of its long bill.
Wild turkey|bird|L|6,4|3|o|herd|d||tw|1|spurs|A big woodland fowl that roosts in trees and gobbles at dawn.
Indian peafowl|bird|L|5,3|3|o|herd|d||tB|2|spurs|The peacock fans a train of a hundred eyes to court the hen.
Red junglefowl|bird|L|5,7|2|o|skittish|d||tr|2|spurs|The wild ancestor of the chicken, a shy pheasant of forest edges.
Common pheasant|bird|L|4,6|2|h|skittish|d||tO|1|spurs|A long-tailed fowl that bursts from cover with a clatter of wings.
Golden pheasant|bird|L|mtn,6|2|h|skittish|d||tO|4|spurs|A pheasant blazing gold and scarlet, shy in mountain thickets.
Western capercaillie|bird|L|9|3|h|territorial|c||tK|3|beak|A huge forest grouse whose cocks will attack anything during the spring lek.
Willow ptarmigan|bird|L|10,9|2|h|skittish|d||tY|1|beak|A grouse that turns white in winter and has feathered feet for the snow.
Greater sage-grouse|bird|L|2,4|2|h|herd|d||tw|3|beak|Males puff out yellow air sacs and pop them to court hens at dawn.
Great bustard|bird|L|4,2|4|h|skittish|d||tw|3|beak|One of the heaviest flying birds, striding the steppe in stately flocks.
Kori bustard|bird|L|3|4|o|skittish|d||tw|3|beak|A heavy savanna bird often followed by bee-eaters snatching flushed insects.
Secretarybird|bird|L|3|4|c|territorial|d||sK|3|kick|A long-legged eagle that stamps snakes to death.
Helmeted guineafowl|bird|L|3|2|o|herd|d||tK|1|beak|A spotted fowl that marches in noisy flocks through the bush.
# raptors
Golden eagle|bird|L|mtn,10,2,sky|4|c|predator|d|fly|EO|2|talons|A great eagle of the mountains that can take a fox or a young deer.
Bald eagle|bird|L|fresh,coast,9|4|p|territorial|d|fly|EY|2|talons|A white-headed sea eagle that steals fish from ospreys.
Harpy eagle|bird|L|7|4|c|predator|d|fly|EK|4|talons|A rainforest eagle with talons longer than a bear's claws that snatches sloths from the trees.
Philippine eagle|bird|L|7|4|c|predator|d|fly|Ew|5|talons|A shaggy-crested monkey-eating eagle of a few island forests.
Steller's sea eagle|bird|L|coast,seac|4|p|territorial|d|fly|EK|4|talons|The heaviest eagle, with a massive yellow bill for tearing salmon.
Martial eagle|bird|L|3|4|c|predator|d|fly|Ew|3|talons|The largest eagle of the savanna, able to kill a small antelope.
Crowned eagle|bird|L|7,5|4|c|predator|d|fly|Ew|4|talons|A powerful forest eagle that preys on monkeys.
Osprey|bird|L|fresh,coast|3|p|skittish|d|fly|vY|1|talons|A fish hawk that plunges feet first into the water.
Peregrine falcon|bird|L|mtn,coast,any,sky|2|c|skittish|d|fly|vK|2|talons|The fastest animal alive, stooping on pigeons at the speed of a falling stone.
Gyrfalcon|bird|L|10,11,sky|3|c|skittish|d|fly|vY|4|talons|The largest falcon, a white hunter of the Arctic prized by kings.
Common kestrel|bird|L|4,6|1|c|skittish|d|fly|vw|1|talons|A small falcon that hovers into the wind over road verges.
Red kite|bird|L|4,6|2|s|skittish|d|fly|vr|2|talons|A fork-tailed scavenger that once cleaned the streets of medieval towns.
Northern goshawk|bird|L|9,6|3|c|skittish|d|fly|vK|3|talons|A fierce forest hawk that chases prey through the trees.
Red-tailed hawk|bird|L|4,6,1|2|c|skittish|d|fly|vr|1|talons|A broad-winged hawk whose scream is borrowed for every eagle on stage.
Lammergeier|bird|L|mtn,sky|4|s|skittish|d|fly|VO|4|talons|The bearded vulture drops bones onto rocks to eat the marrow.
Andean condor|bird|L|mtn,sky|4|s|skittish|d|fly|VK|3|beak|A huge vulture that soars for hours without a wingbeat.
California condor|bird|L|mtn,1,sky|4|s|skittish|d|fly|VK|5|beak|A great vulture brought back from just twenty-two birds.
Griffon vulture|bird|L|mtn,1,3,sky|4|s|skittish|d|fly|Vw|2|beak|A pale-necked vulture that gathers in crowds at a carcass.
Lappet-faced vulture|bird|L|3,1|4|s|territorial|d|fly|Vr|3|beak|The largest vulture of the savanna, which bullies all others off the kill.
King vulture|bird|L|7|3|s|skittish|d|fly|VO|4|beak|A white vulture with a wattled face of orange, purple and red.
Turkey vulture|bird|L|4,6,1|3|s|skittish|d|fly|Vr|1|beak|It finds carrion by smell, a rare gift among birds.
Eurasian eagle-owl|bird|L|mtn,9,6|3|c|skittish|n|fly|OO|3|talons|A huge horned owl able to kill a fox.
Snowy owl|bird|L|10,11|3|c|skittish|a|fly|OY|3|talons|A white owl of the tundra that hunts lemmings under the midnight sun.
Barn owl|bird|L|4,6|2|c|skittish|n|fly|oY|1|talons|A ghost-faced owl that hunts by sound alone in total darkness.
Great grey owl|bird|L|9|3|c|skittish|n|fly|oK|3|talons|A huge-faced owl that plunges through snow to seize voles it can only hear.
Tawny owl|bird|L|6|2|c|skittish|n|fly|ow|1|talons|The owl whose hooting calls fill the woods at night.
Burrowing owl|bird|L|4,1|1|i|skittish|d|fly|ow|2|talons|A long-legged owl that nests in old prairie-dog burrows.
# water birds
Mute swan|bird|L|fresh,12,6|4|h|territorial|d|fly|SY|1|wings|A white swan that hisses and beats intruders with its wings.
Whooper swan|bird|L|fresh,10,9|4|h|skittish|d|fly|SY|2|wings|A wild swan whose bugling calls carry over the northern lakes.
Black swan|bird|L|fresh,12|4|h|territorial|d|fly|Sk|3|wings|A black swan with a scarlet bill, once a byword for the impossible.
Greylag goose|bird|L|fresh,12,4|3|h|herd|d|fly|gK|1|wings|The wild ancestor of the farmyard goose, honking in long skeins.
Bar-headed goose|bird|L|mtn,fresh,sky|3|h|herd|d|fly|gY|3|wings|It flies over the highest mountains in the world on its migration.
Canada goose|bird|L|fresh,4,12|3|h|herd|d|fly|gK|1|wings|A loud goose that defends its goslings fiercely.
Mallard|bird|L|fresh,12|2|o|herd|d|fly|gG|1|bill|A wild duck with an emerald head, ancestor of most farmyard ducks.
Mandarin duck|bird|L|fresh,6|2|o|skittish|d|fly|gO|3|bill|A gaudy tree-nesting duck, a symbol of faithful love.
Common eider|bird|L|coast,seac|2|o|herd|d|fly|gY|2|bill|A sea duck whose breast down lines the warmest quilts.
Grey heron|bird|L|fresh,12|3|p|skittish|d|fly|hK|1|beak|A patient grey fisher that stands motionless in the shallows.
Great egret|bird|L|12,fresh|3|p|skittish|d|fly|hY|1|beak|A white heron whose lace plumes were once worth more than gold.
Shoebill|bird|L|12|4|p|passive|d|fly|Hk|4|beak|A stork-like giant with a clog-shaped bill that stands still for hours waiting for lungfish.
White stork|bird|L|4,12|4|o|skittish|d|fly|hY|2|beak|A stork that nests on chimneys and is said to bring babies.
Marabou stork|bird|L|3,12|4|s|skittish|d|fly|Hk|2|beak|An undertaker bird with a bald head and a dangling throat pouch.
Saddle-billed stork|bird|L|12|4|p|skittish|d|fly|Hr|3|beak|A tall stork with a red-and-black bill topped by a yellow saddle.
Greater flamingo|bird|L|12,coast|3|f|herd|d|fly|fM|2|beak|A pink wader that feeds upside down, filtering shrimp from salt lakes.
Eurasian spoonbill|bird|L|12,coast|3|p|herd|d|fly|hY|2|beak|It sweeps its spoon-shaped bill through the shallows from side to side.
Scarlet ibis|bird|L|12,coast|2|p|herd|d|fly|iR|3|beak|A blazing red ibis of tropical mangroves.
Sacred ibis|bird|L|12,3|2|o|herd|d|fly|iY|2|beak|A white ibis with a black head, once mummified by the million in temples.
Common crane|bird|L|12,4|4|o|herd|d|fly|hK|2|beak|A grey crane whose dances and trumpeting are a sign of spring.
Red-crowned crane|bird|L|12|4|o|territorial|d|fly|hY|4|beak|A white crane with a red crown, a symbol of a thousand years of life.
Whooping crane|bird|L|12,4|4|o|skittish|d|fly|hY|5|beak|The tallest bird of its continent, once down to fifteen birds.
Great white pelican|bird|L|fresh,12,coast|4|p|herd|d|fly|PY|2|beak|Flocks of pelicans fish together, herding fish into the shallows.
Brown pelican|bird|L|coast,seaw|4|p|herd|d|fly|Pw|2|beak|A pelican that dives headlong into the sea to scoop fish.
Great cormorant|bird|L|coast,fresh|3|p|skittish|d|fly|ck|1|beak|A dark diving bird that dries its outspread wings after fishing.
Common loon|bird|L|fresh,9|3|p|skittish|d|fly|lK|2|beak|A diver whose wailing calls haunt northern lakes at night.
Great crested grebe|bird|L|fresh,12|2|p|skittish|d|fly|gw|1|beak|Courting pairs dance face to face, offering each other weed.
Common kingfisher|bird|L|fresh,6|1|p|skittish|d|fly|vB|1|beak|A flash of blue that dives from a perch into the stream.
Laughing kookaburra|bird|L|6,1|2|c|skittish|d|fly|vw|2|beak|A giant kingfisher whose cackling chorus greets the dawn.
# seabirds
Wandering albatross|bird|L|seac,coast,sky|4|p|skittish|a|fly|AY|3|beak|The longest wings of any bird; it circles the southern ocean for years without landing.
Northern gannet|bird|L|coast,seac|3|p|herd|d|fly|gY|2|beak|A seabird that plunges from great heights like a white spear.
Atlantic puffin|bird|L|coast,seac|1|p|herd|d|fly|pk|2|beak|A sea parrot with a striped bill that carries a row of fish at once.
Herring gull|bird|L|coast,fresh|2|s|skittish|d|fly|gY|1|beak|A loud, bold gull that follows fishing boats and raids picnics.
Arctic tern|bird|L|coast,seac,ice|1|p|territorial|d|fly|gY|2|beak|It flies from pole to pole every year and sees more daylight than any other creature.
Magnificent frigatebird|bird|L|seaw,coast|3|p|skittish|d|fly|Fk|2|beak|A pirate of the skies that robs other seabirds of their catch.
Emperor penguin|bird|L|ice,11,seac|4|p|herd|a|@avian|PK|3|beak|The largest penguin, whose males brood the egg through the polar winter.
King penguin|bird|L|coast,seac|3|p|herd|a||PK|3|beak|A tall penguin with orange ear patches that breeds in vast colonies.
Adelie penguin|bird|L|ice,11,coast|2|p|herd|a||pK|2|beak|A small penguin that steals stones from its neighbours' nests.
African penguin|bird|L|coast,seaw|2|p|herd|a||pK|3|beak|A braying penguin of warm southern shores.
# songbirds, parrots, others
Common raven|bird|L|any,mtn,10|2|o|skittish|a|fly|ck|1|beak|A clever black bird that plays games, holds grudges and talks to wolves.
Carrion crow|bird|L|4,6|1|o|skittish|d|fly|ck|1|beak|A crafty crow that drops nuts on roads for cars to crack.
Eurasian magpie|bird|L|4,6|1|o|skittish|d|fly|cB|1|beak|A black-and-white crow that recognises itself in a mirror.
Common starling|bird|L|4,6|1|o|herd|d|fly|vG|1|beak|In winter its flocks pour across the dusk sky in shape-shifting murmurations.
House sparrow|bird|L|4,6,1|1|o|herd|d|fly|vw|1|beak|A cheeky sparrow that has followed farmers since the first fields.
European robin|bird|L|6|1|i|skittish|d|fly|vR|1|beak|A red-breasted songbird that follows gardeners for worms.
Common nightingale|bird|L|6|1|i|skittish|n|fly|vw|2|beak|A plain brown bird with the richest song of the night.
Barn swallow|bird|L|4,6,12|1|i|skittish|d|fly|vB|1|beak|A fork-tailed swallow whose return marks the coming of summer.
Common swift|bird|L|4,6,sky|1|i|skittish|d|fly|vK|1|beak|It eats, sleeps and mates on the wing, landing only to nest.
Ruby-throated hummingbird|bird|L|6,4|1|h|skittish|d|fly|vG|2|beak|A thumb-sized bird whose wings beat fifty times a second.
Superb lyrebird|bird|L|6,8|2|i|skittish|d||vw|4|beak|A pheasant-sized songbird that mimics chainsaws, camera shutters and other birds.
Raggiana bird-of-paradise|bird|L|7|1|h|skittish|d|fly|vO|4|beak|Males dance upside down in cascades of orange plumes.
Resplendent quetzal|bird|L|7,mtn|1|h|skittish|d|fly|vG|4|beak|A cloud-forest trogon whose emerald tail plumes were worn by kings.
Toco toucan|bird|L|7,3|2|o|skittish|d|fly|vO|2|beak|A toucan whose huge orange bill sheds heat like a radiator.
Great hornbill|bird|L|7|3|h|skittish|d|fly|HY|3|beak|A hornbill with a casque on its bill; the female walls herself into her nest.
Scarlet macaw|bird|L|7|2|h|herd|d|fly|vR|3|beak|A red-blue-and-gold parrot that eats clay to soothe the poisons of its seeds.
Hyacinth macaw|bird|L|12,3|3|h|skittish|d|fly|vB|4|beak|The largest parrot, cobalt blue, that cracks palm nuts with its bill.
African grey parrot|bird|L|7|2|h|herd|d|fly|vK|3|beak|A grey parrot that learns hundreds of words.
Kea|bird|L|mtn,10|2|o|skittish|d|fly|vG|4|beak|A mountain parrot that dismantles cars and tents out of curiosity.
Kakapo|bird|L|8|3|h|passive|n||vG|5|beak|A flightless, moss-green night parrot that smells of honey.
Sulphur-crested cockatoo|bird|L|6,1|2|h|herd|d|fly|vY|2|beak|A white cockatoo that dances to music and raids crops in screeching flocks.
Rock dove|bird|L|any,mtn|1|h|herd|d|fly|vK|1|beak|The wild pigeon of sea cliffs, ancestor of every city pigeon.
Victoria crowned pigeon|bird|L|7|2|h|skittish|d||vB|4|beak|A turkey-sized blue pigeon with a crest of lace.
Greater roadrunner|bird|L|1|2|c|skittish|d||rw|2|beak|A desert cuckoo that runs down lizards and snakes.
Hoatzin|bird|L|12,7|2|h|skittish|d|fly|vO|4|beak|A stinking bird whose chicks have claws on their wings to climb back to the nest.
Common cuckoo|bird|L|6,4|1|i|skittish|d|fly|vK|1|beak|It lays its eggs in other birds' nests; its call names it.
European bee-eater|bird|L|4,3|1|i|herd|d|fly|vO|2|beak|A rainbow-coloured bird that knocks the stings from bees before eating them.
Hoopoe|bird|L|4,1|1|i|skittish|d|fly|vO|2|beak|A crested pink bird whose nest reeks to keep predators away.
Great spotted woodpecker|bird|L|6,9|1|i|skittish|d|fly|vK|1|beak|It drums on dead branches to claim its wood.
Wild budgerigar|bird|L|1,3|1|h|herd|d|fly|vG|1|beak|A tiny green parakeet that gathers in flocks of thousands at desert water holes.
Common nighthawk|bird|L|4,1|1|i|skittish|n|fly|vK|2|beak|A nightjar that booms as it dives at dusk.
Tufted puffin|bird|L|coast,seac|1|p|herd|d|fly|pk|3|beak|A dark puffin with golden head plumes.
# recently extinct birds
Dodo|bird|X1662|5,coast|3|h|passive|d||dK|5|beak|A flightless island pigeon, fat and fearless, gone within a century of the first ships.
Great auk|bird|X1844|coast,seac|3|p|passive|a||PK|5|beak|The original penguin, a flightless auk of northern seas hunted for feathers and oil.
Passenger pigeon|bird|X1914|6|1|h|herd|d|fly|vB|5|beak|Once the most numerous bird on earth; its flocks darkened the sky for days.
Carolina parakeet|bird|X1918|6,12|1|h|herd|d|fly|vG|5|beak|A green-and-yellow parrot of eastern woods that ate poisonous cockleburs.
Giant moa|bird|Xc. 1445|8,4,mtn|5|h|skittish|d||Mw|5|kick|A wingless browser taller than any man, hunted to extinction by the first settlers of its islands.
Haast's eagle|bird|Xc. 1445|8,4,mtn,sky|5|c|predator|d|fly|EK|5|talons|The largest eagle ever, which struck giant moa at the speed of a horse's gallop.
Elephant bird|bird|Xc. 1000|5,1|6|h|skittish|d||Ow|5|kick|A flightless giant whose eggs held as much as a hundred and fifty hen's eggs.
Labrador duck|bird|X1878|coast|2|i|skittish|d|fly|gK|5|bill|A sea duck of sandy bays that no one ever saw alive after its last decade.
Heath hen|bird|X1932|4|2|h|herd|d||tw|5|beak|A prairie chicken of coastal sandplains, so common it was once cheap servants' food.
Huia|bird|X1907|8|1|i|skittish|d|fly|vK|5|beak|Mated pairs had different bills, one to chisel and one to probe.
Ivory-billed woodpecker|bird|X1944|12,6|2|i|skittish|d|fly|vK|5|beak|The Lord God bird of old swamp forests, last seen for certain long ago.
Rodrigues solitaire|bird|X1761|5|3|h|passive|d||dw|5|wings|A flightless pigeon that struck rivals with knobs on its wings.
Red rail|bird|X1693|5|2|o|passive|d||tR|5|beak|A flightless island rail drawn to anything red.
Spectacled cormorant|bird|X1850|coast|3|p|passive|d||ck|5|beak|A flightless cormorant of a single island chain.
Pink-headed duck|bird|X1949|12|2|h|skittish|d|fly|gM|5|bill|A duck with a pink head from reedy jungle pools.
`)
