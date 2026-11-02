package engine

// Bestiary catalogue: living fishes of sea, reef and river.
// name|group|when|habitat|size|diet|temper|activity|flags|glyph+colour|rarity|attack|description
var _ = bsReg(`
# sharks and rays
Great white shark|fish|L|sea,reef|6|c|predator|a|aquatic|FY|4|teeth|A torpedo of a shark that hurls itself out of the sea onto seals.
Tiger shark|fish|L|seaw,reefw|5|c|predator|a|aquatic|Fw|3|teeth|The garbage can of the sea, which eats turtles, birds and licence plates.
Bull shark|fish|L|seaw,fresh,coast|5|c|predator|a|aquatic|Fw|3|teeth|A stocky shark that swims far up rivers into fresh water.
Great hammerhead|fish|L|seaw,reefw|5|c|predator|a|aquatic|FK|3|teeth|It pins stingrays to the sand with its wide hammer-shaped head.
Scalloped hammerhead|fish|L|seaw,reefw|5|c|territorial|a|aquatic pack|FK|3|teeth|Hammerheads school by the hundred around undersea mountains.
Oceanic whitetip shark|fish|L|seaw|5|c|predator|a|aquatic|FY|4|teeth|A long-finned shark of the open sea that follows shipwrecks.
Blue shark|fish|L|sea|4|c|territorial|a|aquatic|fB|2|teeth|A slender blue shark that crosses whole oceans.
Shortfin mako|fish|L|sea|5|c|predator|a|aquatic|FB|4|teeth|The fastest shark, leaping high from the water when hooked.
Whale shark|fish|L|seaw,reefw|7|f|passive|a|aquatic|FB|3|teeth|The largest fish, a spotted giant that sieves plankton.
Basking shark|fish|L|seac|7|f|passive|a|aquatic|FK|3|teeth|A huge plankton-eating shark that cruises the surface with its mouth agape.
Greenland shark|fish|L|seac,ice|6|s|passive|a|aquatic|Fk|4|teeth|A slow shark of the icy deep that may live four hundred years.
Goblin shark|fish|L|reef|5|c|territorial|a|aquatic|Fm|5|jaws|A pink deep-sea shark with a blade snout and jaws that shoot forward to bite.
Frilled shark|fish|L|reef|4|c|passive|a|aquatic @serpent|fK|5|teeth|An eel-like deep-sea shark with three hundred trident teeth.
Nurse shark|fish|L|reefw|4|c|passive|n|aquatic|fw|1|jaws|A sluggish reef shark that sucks prey from crevices.
Whitetip reef shark|fish|L|reefw|4|c|territorial|n|aquatic pack|fK|2|teeth|A slim reef shark that hunts in packs through the coral at night.
Epaulette shark|fish|L|reefw|2|c|skittish|a|aquatic|fw|3|jaws|A small shark that walks over reef flats on its fins at low tide.
Wobbegong|fish|L|reefw|4|c|territorial|a|aquatic|fw|3|jaws|A carpet shark with a fringed beard that lies hidden on the reef.
Spiny dogfish|fish|L|seac,reefc|3|c|skittish|a|aquatic pack|fK|1|teeth|A small shark with poison-tipped spines before its fins.
Giant manta ray|fish|L|seaw,reefw|6|f|passive|a|aquatic|MK|3|wings|A ray as wide as a boat that somersaults through plankton clouds.
Southern stingray|fish|L|reefw|3|c|passive|a|aquatic poison|mw|1|barb|A flat ray that hides under the sand with a venomous barb on its tail.
Spotted eagle ray|fish|L|reefw|4|c|skittish|a|aquatic|mB|2|barb|A spotted ray that flies through the water in flocks.
Marbled electric ray|fish|L|reef|3|c|passive|a|aquatic shock|mw|2|shock|A ray that stuns its prey with a jolt from organs in its wings.
Largetooth sawfish|fish|L|fresh,seaw,coast|5|c|skittish|a|aquatic|Sw|4|saw|A shark-like ray with a toothed saw it slashes through schools of fish.
# open sea fish
Atlantic bluefin tuna|fish|L|sea|5|p|skittish|a|aquatic pack|uB|3|jaws|A warm-blooded racing fish that crosses the ocean to spawn.
Swordfish|fish|L|sea|5|p|territorial|a|aquatic|XB|3|sword|A great fish that slashes through schools with its flat sword.
Atlantic sailfish|fish|L|seaw|4|p|skittish|a|aquatic|xB|3|bill|A fish with a towering sail fin, among the fastest in the sea.
Blue marlin|fish|L|seaw|5|p|skittish|a|aquatic|XB|4|bill|A huge billfish that stuns tuna with blows of its spear.
Ocean sunfish|fish|L|sea|6|f|passive|d|aquatic|OY|4|mouth|The heaviest bony fish, a giant floating head that basks at the surface.
Mahi-mahi|fish|L|seaw|3|p|skittish|d|aquatic|uG|2|jaws|A golden-green fish that flashes colour as it dies.
Atlantic herring|fish|L|seac|1|f|herd|a|aquatic|uY|1|mouth|Shoals of billions once fed the seaports of a whole continent.
Atlantic cod|fish|L|seac,reefc|3|c|herd|a|aquatic|uw|1|jaws|The fish that built fleets and fed empires.
Atlantic salmon|fish|L|fresh,seac|3|p|herd|a|aquatic|uM|2|jaws|It leaps waterfalls to spawn in the stream of its birth.
Sockeye salmon|fish|L|fresh,seac|3|p|herd|a|aquatic|uR|2|jaws|It turns scarlet as it swims upriver to spawn and die.
Flying fish|fish|L|seaw|1|f|herd|d|aquatic|uB|1|mouth|It glides above the waves on wing-like fins to escape tuna.
Oarfish|fish|L|sea,reef|6|f|passive|a|aquatic @serpent|SY|5|mouth|A silver ribbon of a fish longer than a ship's mast, the sea serpent of old tales.
Opah|fish|L|sea|4|c|passive|a|aquatic|uR|4|jaws|A round red moonfish, the only fish with warm blood throughout its body.
Atlantic mackerel|fish|L|sea,coast|1|p|herd|a|aquatic|uB|1|jaws|A barred, shimmering fish that travels in vast restless shoals.
Barracuda|fish|L|reefw,seaw|4|p|territorial|d|aquatic|xY|2|teeth|A long fanged fish drawn to anything that glints.
# reef fish
Clownfish|fish|L|reefw|1|o|passive|d|aquatic|uO|1|bite|An orange fish that lives safely among the stinging tentacles of an anemone.
Blue tang|fish|L|reefw|1|h|herd|d|aquatic|uB|1|spine|A bright blue surgeonfish with scalpel spines at the base of its tail.
Parrotfish|fish|L|reefw|3|h|herd|d|aquatic|uG|1|beak|It crunches coral and excretes the white sand of tropical beaches.
Humphead wrasse|fish|L|reefw|5|c|passive|d|aquatic|UG|4|jaws|A giant wrasse with a bulging forehead that eats poisonous starfish.
Red lionfish|fish|L|reefw|2|c|territorial|a|aquatic poison|uR|2|spines|A frilled fish with venomous spines that herds smaller fish into corners.
Reef stonefish|fish|L|reefw|2|c|passive|a|aquatic poison|uw|3|spines|The most venomous fish, indistinguishable from a lump of coral.
Pufferfish|fish|L|reefw|2|o|passive|a|aquatic poison|uY|2|beak|It swells into a spiky ball and is deadly to eat unless carefully prepared.
Moray eel|fish|L|reefw|4|c|territorial|n|aquatic @serpent|eG|2|jaws|An eel with a second set of jaws in its throat that hides in crevices.
Garden eel|fish|L|reefw|1|f|skittish|d|aquatic @serpent|ew|2|mouth|Colonies sway from their burrows like grass, vanishing into the sand at a shadow.
Seahorse|fish|L|reefw,coast|1|c|passive|d|aquatic|hO|2|snout|A fish that swims upright and whose males carry the eggs.
Leafy seadragon|fish|L|reef|1|c|passive|d|aquatic|hG|4|snout|A seahorse cousin dressed in leaf-like fronds that drifts like seaweed.
Giant grouper|fish|L|reefw|5|c|territorial|a|aquatic|UK|3|jaws|A huge reef grouper that can swallow a small shark whole.
Mandarinfish|fish|L|reefw|1|i|passive|c|aquatic|uB|3|mouth|A tiny psychedelic dragonet that dances at dusk.
# deep sea
Anglerfish|fish|L|reef|2|c|passive|a|aquatic|aY|3|jaws|A deep-sea fish that dangles a glowing lure before a mouth full of glass teeth.
Gulper eel|fish|L|reef|3|c|passive|a|aquatic @serpent|eK|4|jaws|A deep-sea eel that is mostly mouth, with a glowing tail tip.
Viperfish|fish|L|reef|2|c|territorial|a|aquatic|uK|3|fangs|A deep-sea fish with fangs so long they cannot fit inside its mouth.
Coelacanth|fish|L|reef|5|c|passive|n|aquatic|CB|5|jaws|A lobe-finned fish thought extinct for sixty-six million years until one was caught.
Blobfish|fish|L|reef|2|o|passive|a|aquatic|uM|3|mouth|A gelatinous fish of crushing depths that sags into a frown when brought up.
Hatchetfish|fish|L|reef,sea|1|f|herd|n|aquatic|uY|2|mouth|A silver deep-sea fish that lights its belly to vanish against the glow above.
# fresh water
Red-bellied piranha|fish|L|fresh,7|1|c|pack|a|aquatic|uR|2|teeth|Shoals of these sharp-toothed fish strip a carcass in minutes.
Arapaima|fish|L|fresh,7|5|p|skittish|a|aquatic|UR|4|jaws|One of the largest river fish, breathing air through a lung-like bladder.
Electric eel|fish|L|fresh,7|4|c|territorial|a|aquatic shock @serpent|EY|3|shock|A knifefish that kills its prey with a discharge strong enough to stop a horse.
Northern pike|fish|L|fresh,9,6|3|p|territorial|a|aquatic|xG|1|teeth|A long ambush predator of weedy lakes, called the water wolf.
Wels catfish|fish|L|fresh,6|5|c|territorial|n|aquatic|CK|3|jaws|A giant catfish that lunges at pigeons on the riverbank.
Mekong giant catfish|fish|L|fresh|6|h|passive|a|aquatic|CK|5|mouth|One of the largest freshwater fish, as heavy as a bear.
Beluga sturgeon|fish|L|fresh,seac|6|p|passive|a|aquatic armored|Cy|5|mouth|A bony-plated giant of inland seas, prized for its roe.
Paddlefish|fish|L|fresh|4|f|passive|a|aquatic|Cy|4|mouth|A filter-feeding fish with a long paddle snout covered in electric sensors.
Alligator gar|fish|L|fresh,12|5|p|territorial|a|aquatic armored|Gw|3|jaws|An armoured fish with a crocodile snout that has hardly changed since the dinosaurs.
Common carp|fish|L|fresh,12|3|o|passive|a|aquatic|uO|1|mouth|A bronze-scaled bottom feeder, kept in ponds since ancient times.
Brown trout|fish|L|fresh,6,9|2|p|skittish|a|aquatic|uw|1|jaws|A spotted trout of clear cold streams.
Rainbow trout|fish|L|fresh,mtn|2|p|skittish|a|aquatic|uM|1|jaws|A trout with a pink band along its flank.
Archerfish|fish|L|fresh,12|1|i|passive|d|aquatic|uY|3|spit|It shoots insects off overhanging leaves with jets of water.
Mudskipper|fish|L|12,coast|1|i|skittish|d||uw|2|bite|A fish that walks on its fins over mangrove mud and breathes through its skin.
African lungfish|fish|L|fresh,12|3|c|passive|a|aquatic burrow|lw|3|jaws|It survives drought sealed in a mucus cocoon underground for years.
Blind cave fish|fish|L|cave,fresh|1|o|passive|a|aquatic|uM|3|mouth|A pink eyeless fish of underground rivers.
Siamese fighting fish|fish|L|fresh,12|1|c|territorial|a|aquatic|uR|2|bite|Males flare their fins and fight to the death over territory.
European eel|fish|L|fresh,sea|2|c|skittish|n|aquatic @serpent|ew|1|jaws|It crosses an ocean to spawn in a sea of weed nobody has ever seen it reach.
Lamprey|fish|L|fresh,seac|2|c|territorial|a|aquatic bleed @serpent|ek|2|sucker|A jawless fish that rasps into its hosts with a disc of teeth.
Hagfish|fish|L|reef|2|s|passive|a|aquatic @serpent|ew|2|rasp|A slime eel that ties itself in knots and fills the sea around it with slime.
`)
