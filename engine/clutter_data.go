package engine

// Clutter catalogue (prefix cl): 21 collections of six small things, genre skins, and wild finds.
// APPEND-ONLY: Item.Extra["cl:i"] stores the definition index+1 in saves.
//
// Set block:  @id|Title|where tags|adjectives per genre
// Item line:  name|glyph|colour|weight (hundredths)|value|flavour|skin overrides (genre letter=name;...)
// Genre letters: f fantasy, m mythic, s steampunk, d dieselpunk, a atompunk, c cyberpunk, b biopunk, o solarpunk,
// w wasteland, t twilight. Colour letters: g gray w white y yellow b brown c cyan m magenta r red G green B blue
// o orange k darkgray C darkcyan.
// Where tags: home kitchen tavern study shop forge mine temple crypt ruin dungeon outdoors coast nature.

const clSetData = `@cutlery|Cutlery|home,kitchen,ruin|f=pewter,m=bronze,s=brass,d=tin,a=chrome,c=polymer,b=bone,o=bamboo,w=scavenged,t=tarnished
spoon|/|g|10|3|Dented and much licked, but it still does its job.|
fork|Y|g|10|3|One tine bent back, as if it lost an argument with a stew.|
table knife|-|g|15|4|Dull enough to be safe at dinner.|
ladle|J|g|30|5|Someone's good soup came out of this.|
salt spoon|,|g|5|4|A tiny spoon for a tiny luxury.|w=bent salt scoop
napkin ring|o|g|10|6|Engraved with initials nobody remembers.|
@tavern|Tavern ware|home,tavern,ruin,dungeon|f=pewter,m=clay,s=brass,d=enamel,a=melamine,c=plastic,b=gourd,o=reed,w=tin,t=cracked
cup|u|b|15|3|Rimmed with old wine stains.|w=bottle cap
plate|O|w|20|4|Chipped at the edge, scrubbed thin.|
bowl|U|b|25|4|Holds exactly one portion of regret.|
goblet|Y|y|20|9|Too fine for the table it was found on.|
tankard|T|b|35|6|A dent shaped like a thrown punch.|
jug|j|b|50|5|Smells of last winter's cider.|
@kitchen|Kitchen bits|home,kitchen|f=iron,m=stone,s=copper,d=steel,a=aluminium,c=ceramic,b=horn,o=wooden,w=rusted,t=blackened
whisk|W|g|10|4|Bent wires, still springy.|
grater|#|g|20|4|Three scars across your knuckle waiting to happen.|
skewer|!|g|10|2|For meat, or for a rude guest.|
pestle|i|w|40|5|Worn smooth by years of grinding.|
tea strainer|o|g|10|6|Fine mesh, fine manners.|
nutcracker|X|b|30|5|Jaws like a small dog.|
@trinket|Pocket trinkets|home,shop,ruin,outdoors|f=brass,m=bone,s=brass,d=tin,a=bakelite,c=glass,b=chitin,o=wooden,w=scrap,t=ash-grey
button|o|y|5|2|Lost off a good coat.|w=bottle cap;a=transistor;c=dead data chip
thimble|n|g|5|3|Someone's dowry was measured in these, once.|s=brass cog;b=fingerbone cap
bone dice|:|w|5|4|Six pips, six regrets.|b=gene-sample vial
marble|o|c|5|2|Swirled glass, like a frozen eye.|
lucky coin|c|y|5|8|Worn smooth by a worrying thumb.|c=dead data chip;w=old bottle cap
pocket charm|*|m|5|10|Meant to turn aside bad luck. Evidently did not.|
@stationery|Stationery|study,home|f=vellum,m=papyrus,s=foolscap,d=carbon,a=mimeo,c=printed,b=bark-paper,o=recycled,w=salvaged,t=faded
quill|/|w|5|5|The nib is split but the vane is lovely.|c=stylus;w=pen
inkpot|u|k|30|6|Half dry, half black, all stains.|
ledger page|=|w|5|4|Columns of debts, most of them crossed out.|
love letter|~|r|5|9|Never sent, or never answered. Hard to say which.|
sealing wax|i|r|10|5|A stub of red, still warm to the thumb.|
signet stamp|#|y|20|14|The seal of somebody who thought they were important.|
@scholar|Scholar's things|study,ruin|f=horn-rimmed,m=obsidian,s=brass,d=steel,a=pocket,c=smart,b=lensed,o=solar,w=cracked,t=ink-stained
spectacles|8|c|10|12|Thick lenses and one bent arm.|c=cracked phone
magnifier|O|c|20|10|Turns small print into large disappointment.|
bookmark|]|r|5|3|Still marks a page someone never finished.|
calipers|A|g|30|9|Measures things that are not worth measuring.|
star chart scrap|*|w|5|7|A corner of sky someone cared about.|
pocket globe|O|B|40|13|The whole world, thumb-sized.|
@toys|Toys|home,outdoors,ruin|f=wooden,m=clay,s=clockwork,d=tin,a=plastic,c=holo,b=grown,o=woven,w=pre-war,t=stitched
doll|&|m|30|8|One eye missing, the other watching you.|w=pre-war action figure
toy soldier|i|r|10|6|Painted red, chipped at the nose.|
spinning top|v|y|10|3|Still wobbles on its stub.|
toy horse|h|b|40|7|Its wheels are gone, but not its dignity.|
rattle|j|y|20|3|Rattles faintly with a lost bead.|
hoop|o|b|50|2|Somebody's afternoon, rolled away.|
@tools|Hand tools|forge,home,mine,outdoors|f=iron,m=bronze,s=brass,d=steel,a=chrome,c=carbon,b=bone,o=salvaged,w=rusty,t=blackened
nail|!|g|5|1|Bent at the head from someone's fury.|
horseshoe|n|g|90|6|Found where luck ran out.|
file|/|g|30|4|Teeth nearly worn to nothing.|
awl|!|b|20|3|Has punched holes in a lot of leather.|
chisel|/|g|30|4|Edge nicked on something harder than wood.|
wrench|T|g|60|5|A good friend to anything that rusts shut.|
@smithy|Smithy scraps|forge,shop|f=iron,m=bronze,s=brass,d=steel,a=cast,c=alloy,b=resin,o=scrap,w=rusty,t=slag
whetstone|=|k|40|4|Hollowed by a thousand edges.|
rivet|o|g|5|2|Popped from a better-made thing.|
hinge|H|g|40|3|Squeaks even in your pocket.|
padlock|0|g|50|9|Locked, with no key anywhere near.|
ingot stub|=|y|70|11|A thumb of cooled metal.|
bellows nozzle|<|g|40|4|Smells of old smoke.|
@mining|Miner's kit|mine,dungeon|f=tallow,m=pitch,s=lamp,d=fuse,a=carbide,c=wire,b=glow,o=salvage,w=salvage,t=ash
candle stub|!|y|5|2|Burned to a thumb's width.|c=dead battery;w=dead battery
miner's token|c|b|5|6|Stamped with a shaft number, long since collapsed.|
ore sample|*|b|40|9|Heavy for its size, and a little pretty.|
lamp wick|~|w|5|2|Charred, trimmed, forgotten.|
tallow lump|o|y|30|2|Greasy, grey, useful.|
rope|S|b|100|5|A fathom of hard-used cord.|
@jewel|Jewellery|shop,temple,crypt,home|f=silver,m=bronze,s=brass,d=nickel,a=chrome,c=circuit,b=pearl,o=shell,w=bottle-cap,t=jet
ring box|o|m|20|20|Velvet inside, empty now.|
brooch|*|y|10|24|A small pin with a smaller stone.|
hairpin|/|y|5|10|Slim, pretty, and sharper than it looks.|
signet ring|o|y|5|30|Cut for someone with a bigger finger.|
locket|O|y|10|28|A curl of hair inside, brown as dust.|
earring|c|y|5|14|Its twin is somewhere else.|
@vanity|Vanity case|home,shop|f=silver,m=ivory,s=brass,d=chrome,a=chrome,c=mirror,b=shell,o=shell,w=cracked,t=jet
comb|=|b|10|3|Missing two teeth, still doing its best.|
mirror shard|<|c|10|5|Shows part of your face and part of the past.|
razor|/|g|20|6|Honed to a grudge.|
powder pot|u|w|20|5|A ghost of rouge in the lid.|
hairbrush|B|b|30|4|Long strands still tangled in it.|
perfume vial|!|m|10|13|Faint lilac when you pull the stopper.|
@ritual|Ritual items|temple,crypt|f=silver,m=ochre,s=brass,d=lead,a=cast,c=etched,b=bone,o=seed,w=scrap,t=pale
prayer beads|o|m|10|8|Worn to the shape of anxious fingers.|
incense stick|!|b|5|3|Spent, but still sweet.|
votive idol|&|y|20|12|A small god with a smaller smile.|
censer chip|u|g|20|6|Ash gathers at the rim.|
ritual bell|n|y|30|14|Rings thin and far away.|
grave coin|c|y|5|10|Meant to pay for something on the far shore.|
@sewing|Sewing basket|home|f=bone,m=bronze,s=brass,d=steel,a=chrome,c=fibre,b=thorn,o=hemp,w=scavenged,t=dark
needle|/|g|5|2|Eye bright, point sharp.|b=thorn needle
thread spool|O|b|10|3|A fingernail's worth of colour left.|
pincushion|o|r|10|3|Still bristling.|
scissors|X|g|20|5|Slightly crossed, always sharp.|
darning egg|o|b|30|3|Smooth, warm, vaguely smug.|
tape measure|=|y|10|4|Marked in units the world has outgrown.|
@music|Music makers|home,shop,tavern|f=wooden,m=reed,s=clockwork,d=tin,a=transistor,c=synth,b=reed,o=bamboo,w=scrap,t=bone
tin whistle|/|g|10|5|Plays three notes, all sad.|c=dead synth
music box|#|y|50|30|Winds down mid-lullaby.|c=pocket synth;a=transistor radio
jaw harp|u|g|10|4|Twangs in the teeth.|
tuning fork|Y|g|20|7|Remembers one note, forever.|
reed pipe|!|G|10|3|Hollow and sweet.|
hand bell|n|y|40|8|A small voice with a loud opinion.|
@games|Gamblers' bits|dungeon,ruin,home,tavern|f=bone,m=clay,s=brass,d=tin,a=plastic,c=data,b=chitin,o=wooden,w=bottle-cap,t=ash
playing card|#|w|5|2|The queen has been crossed out.|
knucklebones|:|w|10|3|Cast so often they gleam.|
loaded die|:|w|5|9|Weighted on one side, proudly.|
gaming token|c|y|5|3|Worth something at one table only.|
coin purse|p|b|10|7|Empty, but it jingles optimistically.|
chess pawn|i|w|10|5|The smallest piece with the greatest story.|
@travel|Travel gear|outdoors,ruin,dungeon|f=leather,m=bone,s=brass,d=canvas,a=nylon,c=carbon,b=hide,o=woven,w=scrap,t=ash
key ring|o|g|10|6|Twelve keys. Nothing here fits any of them.|
compass|O|y|20|15|The needle still points at something.|
canteen cap|o|g|10|2|Dented and tethered to nothing.|
snuff box|u|y|30|12|A sneeze waiting to happen.|
pipe|b|b|20|8|Bitten stem, cold bowl.|
flint and steel|x|g|20|3|A spark, if you ask nicely.|
@hearth|Hearth and lamp|home,temple,ruin|f=brass,m=clay,s=brass,d=tin,a=chrome,c=glass,b=bone,o=ceramic,w=cracked,t=black
lantern glass|O|c|30|6|Smoke-black on one side.|
candlestick|!|y|40|8|Wax down one flank like frozen tears.|
snuffer|T|g|20|3|A tiny bell for a tiny dark.|
tinderbox|u|b|40|4|Dry as bone inside.|
fire poker|/|g|90|4|Hooked at the tip.|
kettle lid|O|g|30|3|Warped, but still fits.|
@sailor|Sailor's kit|outdoors,coast,ruin|f=whalebone,m=bronze,s=brass,d=steel,a=aluminium,c=carbon,b=bone,o=reed,w=scrap,t=barnacled
fishhook|j|g|5|2|Barbed, bright, patient.|
net weight|o|k|40|2|Lead, pocked with salt.|
belaying pin|!|b|90|5|Good for ropes and for arguments.|
lead line|S|g|70|4|Knotted every fathom.|
tar pot|u|k|50|3|Black, sticky, enduring.|
scrimshaw tooth|)|w|40|20|A sailing ship carved on it, tiny and fine.|
@relic|Old relics|dungeon,crypt,ruin,temple|f=bronze,m=stone,s=brass,d=lead,a=glass,c=etched,b=bone,o=ceramic,w=pre-war,t=jet
tarnished medal|o|y|10|14|Awarded for something you do not remember.|
idol fragment|&|g|50|11|A hand, a foot, a hint of a smile.|
map scrap|~|b|5|9|A coast that may not exist.|
engraved plaque|=|y|60|16|Names that nobody reads anymore.|
glass bead|o|C|5|6|A drop of old sky.|
tally stick|/|b|30|5|Notched with debts, in a language nobody speaks.|
@nature|Naturalist's finds|nature|
crow feather|~|k|2|3|Glossy black, with a whisper of blue.|
cowrie shell|c|w|5|6|Once currency, still pretty.|
quartz crystal|*|w|20|9|Cloudy, sharp, cold in the palm.|
amber bead|o|o|5|18|A fly sleeps in it, ten thousand years deep.|
antler tip|)|w|40|10|Chewed at one end by something hungry.|
animal tooth|)|w|10|8|A canine, yellowed.|
`

// Wild finds. Line: name|glyph|colour|weight|value|tag|life(days; 0 = keeps)|kind|where|biomes|seasons|flavour
// kind: i ingredient, c clutter. where: tree bush ground forest cave coast water hive desert home rain(boost)
// seasons: letters p u a w (spring summer autumn winter), empty = all. Names that match a nature-set item above make that item.
const clNatData = `apple|o|r|20|6|spry|14|i|tree|4,5,6,8|ua|A windfall, bruised on one side.
pear|o|G|20|6|spry|12|i|tree|4,6|ua|Honey-sweet where it has not bruised.
plum|o|m|10|5|spry|9|i|tree|6,8,4|u|Dusty-skinned and heavy with juice.
wild cherry|o|r|5|4|spry|7|i|tree|5,6,9|u|Small, dark, sour.
fig|o|m|10|7|spry|8|i|tree|1,3,5|ua|Split open and sticky.
date|o|b|10|6|lasting|40|i|tree|1,3|ua|Chewy, dark, endlessly sweet.
bitter orange|o|o|20|6|ember|20|i|tree|5,7|wp|Sharp as a slap.
mango|o|o|30|9|spry|8|i|tree|7,5|u|Fragrant, and sticky to the elbow.
walnut|o|b|10|4|lasting|60|i|tree|6,8|a|Hard shell, wrinkled mind.
hazelnut|o|b|5|3|lasting|60|i|tree|6,9|a|Small and round, a squirrel's rent.
chestnut|o|b|10|4|hearty|45|i|tree|6,8|a|Spiny husk, glossy prize.
acorn|o|b|5|1|lasting|60|i|tree|6,8,9|a|Oak's whole future in a cap.
blackberry|o|m|5|3|spry|5|i|bush|4,6,8,9|ua|Purple fingers guaranteed.
blueberry|o|B|5|4|keen|5|i|bush|9,10,6|u|Dusty-blue and tart.
wild strawberry|o|r|5|5|spry|4|i|bush|4,6|pu|Tiny, sweet, gone in a breath.
juniper berry|o|B|5|3|keen|30|i|bush|9,2,10|ua|Resinous and sharp.
cranberry|o|r|5|4|rime|20|i|bush|12,9,10|a|Tart enough to wake the dead.
field mushroom|T|w|10|3|thick|4|i|ground,rain|4,6,8|ua|Pale cap, pink gills, earthy.
chanterelle|T|o|10|8|hearty|4|i|forest,rain|6,8,9|ua|Golden and faintly apricot.
puffball|O|w|20|3|hearty|5|i|ground|4,3,6|ua|A white ball that coughs smoke when kicked.
morel|T|b|5|12|keen|4|i|forest|6,8,9|p|Honeycombed, nutty, dear.
fly agaric|T|r|10|5|venom|6|i|forest,rain|9,6,8|ua|Red with white flecks. Do not.
glowcap|T|C|5|10|keen|8|i|cave|-|-|Glimmers faintly in the dark.
honeycomb|#|y|30|14|spry|0|i|hive|4,5,6,8|ups|Drips gold into your palm.
wild bird egg|o|w|5|3|hearty|10|i|forest|4,6,8,9|pu|Speckled, warm from a nest.
duck egg|o|w|10|4|hearty|10|i|water|12,4,6,8|pu|Pale blue-green and surprisingly heavy.
dried mushrooms|T|b|5|6|keen|0|i|home|-|-|A string of leathery caps.
dried fruit|o|m|10|5|spry|0|i|home|-|-|Chewy, wrinkled, long-lasting.
honey jar|u|y|40|9|spry|0|i|home|-|-|Sealed with wax.
salted fish|~|B|30|5|deep|0|i|home|-|-|Hard as a plank.
crow feather|~|k|2|3||0|c|ground|-|-|x
cowrie shell|c|w|5|6||0|c|coast|-|-|x
quartz crystal|*|w|20|9||0|c|ground,cave|-|-|x
amber bead|o|o|5|18||0|c|coast,forest|-|-|x
antler tip|)|w|40|10||0|c|forest|6,8,9,10|aw|x
animal tooth|)|w|10|8||0|c|ground,cave|-|-|x
driftwood|/|b|60|2||0|c|coast|-|-|Silvered by salt, light as a bone.
scallop shell|c|w|10|3||0|c|coast|-|-|Ribbed like a tiny fan.
sea glass|o|G|5|5||0|c|coast|-|-|Smoothed by the sea to a jewel.
pretty pebble|o|c|10|1||0|c|ground,water|-|-|Smooth, warm, vaguely lucky.
river agate|o|o|10|7||0|c|water|-|-|Banded orange, like a tiny sunset.
clay lump|o|b|50|2||0|c|water|-|-|Cool, grey and malleable.
rock salt|*|w|30|3||0|c|desert|1,2,13,16|-|Pink-white and bitter.
rib bone|)|w|40|1||0|c|ground|-|-|Bleached by sun, gnawed by something.
jawbone|)|w|50|2||0|c|ground,cave|-|-|A dry grin with half its teeth.
animal skull|0|w|60|4||0|c|ground,cave|-|-|Hollow eyes, patient manner.
shed antler|Y|w|100|12||0|c|forest|6,8,9,10|aw|Forked and heavy, a stag's lost crown.
`
