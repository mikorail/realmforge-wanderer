// People (the player, NPCs, companions) as hand-authored 12x20 pixel art.
//
// How a person is built (all layers are text grids or short pixel lists, see types.ts for letters):
//   build   one of six silhouettes (avg, slim, stocky, bulky, small, lean): torso grid, shoulder and hip anchors
//   legs    two hand-drawn leg sets (normal, thick) x five frames (stand, contact A, passing, contact B, passing),
//           resampled to the build's leg length; robes, coats and fish tails are derived from the leg mask
//   arms    pixel paths per arm state (down, swing fwd/back, raised, strike, guard, cast, aim)
//   head    one grid per head shape (round, long, wide, skull) + hair styles, beards, ears, snouts, horns, tusks
//   outfit  a kit per calling/role: cloth, legwear, headgear, back piece, held item, armour tier recolours
//   weapon  small grids anchored at the hand ('+'), one per orientation (rest, raised, forward, aimed)
// Part grids are fill-only. Slot characters are recoloured at the end per outfit and armour:
//   1 2 3 torso base/shadow/highlight, 9 trim, S B sleeve base/shadow, 4 5 legwear, 6 8 boots, 7 belt,
//   0 hand, # eye, C V I cloak/hood/hat base/shadow/highlight.
// A 1px outline ('o') is traced around everything at the end; sparkles and trails go on an unoutlined fx layer.
// Everything is drawn facing right; pose.face -1 mirrors it.

import { PW, PH, type Rows, type PCtx } from './types';
import type { QudActor } from '../actorsprites';
import type { Pose } from '../poses';

export interface PersonPixels {
	rows: Rows;
	/** colours for the skin/hair/accent letters (fg and bg come from the cell) */
	pal: Pick<PCtx, 'skin' | 'hair' | 'accent'>;
}

// ---------------------------------------------------------------- grid helpers

type Grid = string[][];
const grid = (): Grid => Array.from({ length: PH }, () => Array<string>(PW).fill('.'));
function put(g: Grid, x: number, y: number, c: string): void {
	if (c !== '.' && c !== ' ' && x >= 0 && x < PW && y >= 0 && y < PH) g[y][x] = c;
}
function stamp(g: Grid, rows: readonly string[], x: number, y: number): void {
	for (let r = 0; r < rows.length; r++) for (let c = 0; c < rows[r].length; c++) put(g, x + c, y + r, rows[r][c]);
}
/** stamp a grid whose '+' marks the hand; the '+' itself is not drawn */
function stampAt(g: Grid, rows: readonly string[], hx: number, hy: number): void {
	let ax = 0, ay = 0;
	for (let r = 0; r < rows.length; r++) {
		const i = rows[r].indexOf('+');
		if (i >= 0) {
			ax = i;
			ay = r;
		}
	}
	for (let r = 0; r < rows.length; r++) for (let c = 0; c < rows[r].length; c++) if (rows[r][c] !== '+') put(g, hx - ax + c, hy - ay + r, rows[r][c]);
}
function hashN(h: number, salt: number): number {
	let x = (Math.imul(h ^ (salt * 0x9e3779b1), 0x85ebca6b) ^ (h >>> 13)) >>> 0;
	x = Math.imul(x ^ (x >>> 15), 0x2c1b3c6d) >>> 0;
	return (x ^ (x >>> 12)) >>> 0;
}
const pick = <T,>(a: readonly T[], n: number): T => a[n % a.length];

// ---------------------------------------------------------------- species

type BuildId = 'avg' | 'slim' | 'stocky' | 'bulky' | 'small' | 'lean';
type HeadId = 'round' | 'long' | 'wide' | 'skull';
type HairStyle = 'short' | 'long' | 'bald' | 'topknot' | 'mohawk' | 'curly' | 'ponytail' | 'braid' | 'bob';
interface Sp {
	b: BuildId;
	head: HeadId;
	skin: string[];
	hair?: string[];
	styles?: HairStyle[];
	ears?: 'point' | 'small' | 'long' | 'rat' | 'fin';
	snout?: 0 | 1 | 2 | 3;
	tail?: 'rat' | 'lizard' | 'fish';
	horns?: 'small' | 'big' | 'ram';
	hornC?: [string, string];
	tusks?: boolean;
	beard?: 0 | 1 | 2; // 0 never, 1 some individuals, 2 always (big)
	eye: string;
	arms4?: boolean;
	wings?: boolean;
	halo?: boolean;
	scales?: boolean;
	bone?: boolean;
	feet?: 'bare' | 'hoof' | 'claw';
	slimOdds?: number; // 1 in n individuals use the slim build instead
}

const SKIN_HUMAN = ['#f0c7a0', '#d9a577', '#c8926a', '#b57d54', '#8a5a3b', '#5d3b27', '#482d1e', '#f5d9bd'];
const HAIR = ['#2a1b12', '#5a3a1e', '#6e3b1f', '#8a6a2a', '#c9a24a', '#d8c89e', '#a8441d', '#9e4b2b', '#d8d0c0', '#a8adb8', '#101010', '#1d2230', '#5c2330'];

const SPECIES: Record<string, Sp> = {
	human: { b: 'avg', head: 'round', skin: SKIN_HUMAN, hair: HAIR, styles: ['short', 'long', 'short', 'ponytail', 'topknot', 'braid', 'bob', 'long', 'curly', 'bald'], beard: 1, eye: 'z', slimOdds: 4 },
	dwarf: { b: 'stocky', head: 'wide', skin: ['#e0a98a', '#c98a68', '#b5744f'], hair: ['#8a3a18', '#5a2a12', '#c9c0b0', '#d9a441', '#9e4b2b'], styles: ['short', 'long', 'braid', 'bald'], beard: 2, eye: 'z' },
	elf: { b: 'slim', head: 'long', skin: ['#f2dcc4', '#e8cfae', '#d9c2a0'], hair: ['#e8dcc0', '#c9a24a', '#2a2018', '#9aa7b8', '#d8c89e'], styles: ['long', 'long', 'ponytail', 'braid', 'bob', 'short'], ears: 'point', eye: 'e' },
	void_elf: { b: 'slim', head: 'long', skin: ['#8a86a8', '#6c6890', '#a49cc0'], hair: ['#e8e4f4', '#2a2438', '#b8a8e8', '#a8adb8'], styles: ['long', 'short', 'ponytail', 'braid'], ears: 'point', eye: 'q' },
	halfling: { b: 'small', head: 'round', skin: ['#e8bd94', '#d4a279', '#c28e66'], hair: ['#6a4020', '#8a5a2a', '#3a2412', '#9e4b2b', '#c9a24a'], styles: ['curly', 'short', 'bob', 'curly'], ears: 'small', feet: 'bare', eye: 'z' },
	orc: { b: 'bulky', head: 'wide', skin: ['#6a8a3a', '#7a9a48', '#58762e'], hair: ['#181410', '#2a1b12'], styles: ['mohawk', 'topknot', 'bald', 'mohawk'], ears: 'point', snout: 1, tusks: true, eye: 'y' },
	goblin: { b: 'small', head: 'long', skin: ['#8aa04a', '#9ab058', '#788c3a'], ears: 'long', snout: 1, eye: 'y' },
	ogor: { b: 'bulky', head: 'wide', skin: ['#9a8478', '#8a7468', '#a89488'], snout: 1, horns: 'small', tusks: true, eye: 'y' },
	troll: { b: 'bulky', head: 'long', skin: ['#5a7a58', '#4a6a4c', '#6a8a60'], ears: 'long', snout: 2, tusks: true, eye: 'y' },
	giant: { b: 'bulky', head: 'wide', skin: ['#9aa8b8', '#8a98a8', '#aab4c0'], hair: ['#c9c0b0', '#6a6258'], styles: ['long', 'short', 'bald'], beard: 2, eye: 'U' },
	ratfolk: { b: 'lean', head: 'long', skin: ['#8a7a6a', '#6a5a4a', '#a89888'], ears: 'rat', snout: 3, tail: 'rat', feet: 'claw', eye: 'r' },
	beastman: { b: 'avg', head: 'long', skin: ['#8a5a30', '#6a4020', '#a87a48'], ears: 'long', snout: 2, horns: 'ram', feet: 'hoof', eye: 'y' },
	lizardfolk: { b: 'avg', head: 'long', skin: ['#4a8a58', '#5a9a68', '#6a8a40'], snout: 3, tail: 'lizard', scales: true, feet: 'claw', eye: 'y' },
	saurian: { b: 'avg', head: 'long', skin: ['#b8482a', '#c8803a', '#a8603a'], snout: 2, tail: 'lizard', horns: 'big', scales: true, feet: 'claw', eye: 'y' },
	merfolk: { b: 'avg', head: 'round', skin: ['#5aa8b8', '#4a98a8', '#7ab8c0'], hair: ['#2a6a8a', '#3aa090', '#8ad0c0'], styles: ['long', 'long', 'ponytail'], ears: 'fin', tail: 'fish', scales: true, eye: 'U' },
	celestial: { b: 'slim', head: 'round', skin: ['#f0dca0', '#f8e8b8', '#e8cc88'], hair: ['#fff4c8', '#f0d890'], styles: ['long', 'short'], wings: true, halo: true, eye: 'c' },
	tomb_undead: { b: 'slim', head: 'skull', skin: ['#d8d0b8', '#c8c0a8'], bone: true, feet: 'bare', eye: 'v' },
	warp_fiend: { b: 'avg', head: 'wide', skin: ['#b0328a', '#8a2a9a', '#c84a7a'], ears: 'point', snout: 1, tail: 'lizard', horns: 'big', hornC: ['z', 'R'], arms4: true, feet: 'hoof', eye: 'x' },
};

/** an unknown species still gets a stable look picked from its name */
function strHash(s: string): number {
	let h = 2166136261;
	for (let i = 0; i < s.length; i++) h = Math.imul(h ^ s.charCodeAt(i), 16777619);
	return h >>> 0;
}
const ODD_SKIN = ['#7a9a8a', '#a08a5a', '#8a7aa8', '#b87a6a', '#6a8aa8', '#a8a070', '#9a6a8a', '#6a9a6a'];
function speciesFor(id: string): Sp {
	const k = SPECIES[id];
	if (k) return k;
	const h = strHash(id);
	return {
		b: (['avg', 'avg', 'slim', 'bulky', 'lean'] as BuildId[])[h % 5],
		head: (['round', 'long', 'wide'] as HeadId[])[(h >>> 4) % 3],
		skin: [ODD_SKIN[(h >>> 7) % 8], ODD_SKIN[(h >>> 10) % 8]],
		hair: (h >>> 13) & 1 ? HAIR : undefined,
		styles: ['short', 'long', 'mohawk'],
		ears: ([undefined, 'point', 'long', 'small'] as Sp['ears'][])[(h >>> 14) % 4],
		snout: ((h >>> 16) % 3) as 0 | 1 | 2,
		horns: ([undefined, undefined, 'small', 'big'] as Sp['horns'][])[(h >>> 18) % 4],
		tail: ([undefined, undefined, 'lizard', 'rat'] as Sp['tail'][])[(h >>> 20) % 4],
		tusks: ((h >>> 22) & 3) === 0,
		eye: pick(['z', 'y', 'r', 'c'], h >>> 24),
	};
}

// ---------------------------------------------------------------- builds

interface Build {
	hb: number; // head bottom row (exclusive)
	neck: [number, number, number] | null; // x, y, width
	torso: string[];
	tx: number;
	ty: number;
	al: number; // far (left) shoulder column
	ar: number; // near (right, weapon) shoulder column
	L: number; // sleeve length in px
	aw: 1 | 2; // arm thickness
	legs: 'n' | 't';
	lrows: number;
}

const T_AVG = ['319912', '311112', '311112', '777777', '399992'];
const T_SLIM = ['3992', '3112', '3112', '7777', '3992'];
const T_BULKY = ['319912', '311112', '311112', '311112', '777777', '399992'];
const T_SMALL = ['3992', '7777', '3992'];
const T_LEAN = ['3992', '3112', '7777', '3992'];

const BUILDS: Record<BuildId, Build> = {
	avg: { hb: 8, neck: [5, 8, 2], torso: T_AVG, tx: 3, ty: 9, al: 2, ar: 9, L: 5, aw: 1, legs: 'n', lrows: 5 },
	slim: { hb: 7, neck: [5, 7, 2], torso: T_SLIM, tx: 4, ty: 8, al: 3, ar: 8, L: 5, aw: 1, legs: 'n', lrows: 6 },
	stocky: { hb: 11, neck: null, torso: T_AVG, tx: 3, ty: 11, al: 2, ar: 9, L: 4, aw: 2, legs: 't', lrows: 3 },
	bulky: { hb: 8, neck: [4, 7, 4], torso: T_BULKY, tx: 3, ty: 8, al: 2, ar: 9, L: 6, aw: 2, legs: 't', lrows: 5 },
	small: { hb: 13, neck: null, torso: T_SMALL, tx: 4, ty: 13, al: 3, ar: 8, L: 3, aw: 1, legs: 'n', lrows: 3 },
	lean: { hb: 10, neck: null, torso: T_LEAN, tx: 4, ty: 10, al: 3, ar: 8, L: 4, aw: 1, legs: 'n', lrows: 5 },
};

// legs: 8 wide, 5 rows; far leg 5/8 (darker), near leg 4/6. Frames: stand, contact A (near leg ahead),
// passing (far foot lifted), contact B (far leg ahead), passing (near foot lifted).
const LEGS_N: string[][] = [
	['..5544..', '..5544..', '..5544..', '..8866..', '..88666.'],
	['..5544..', '.55.44..', '.55..44.', '88...66.', '88...666'],
	['..5544..', '..5544..', '..5.44..', '.88.44..', '....666.'],
	['..5544..', '..44.55.', '.44..55.', '.66..88.', '66...888'],
	['..5544..', '..5544..', '..55.4..', '..55.66.', '..888...'],
];
const LEGS_T: string[][] = [
	['.555444.', '.555444.', '.555444.', '.888666.', '.8886666'],
	['.555444.', '555.444.', '555..444', '888..666', '888..666'],
	['.555444.', '.555444.', '.55.444.', '888.444.', '....666.'],
	['.555444.', '.444555.', '444..555', '666..888', '666..888'],
	['.555444.', '.555444.', '.555.44.', '.555.666', '.888....'],
];
const LEG_ROWS: Record<number, number[]> = { 3: [0, 3, 4], 4: [0, 2, 3, 4], 5: [0, 1, 2, 3, 4], 6: [0, 1, 1, 2, 3, 4] };

// fish tail for merfolk (6 wide), sways with the walk phase
const FISH = ['.kkkK.', '.kkkK.', '..kKK.', '..cu..', '.ccuu.'];

// ---------------------------------------------------------------- heads (fill only, facing right)

const HEADS: Record<HeadId, string[]> = {
	round: ['.kkkk.', 'kkkkkk', 'kkjkkk', 'kk#k#K', 'kkkkkK', '.kkKK.'],
	long: ['.kkk.', 'kkkkk', 'kkjkk', 'kk#k#', 'kkkkK', 'kkkkK', '.kkK.'],
	wide: ['.kkkkk.', 'kkkkkkk', 'kkjkkkk', 'kk#kk#K', 'kkkkkkK', '.kkkkK.'],
	skull: ['.jjjj.', 'jkkkkk', 'kz#kz#', 'kkkzkK', '.kzkzK', '.kkkk.'],
};

type Cell = [number, number, string];
/** hair cells relative to the head box */
function hairCells(style: HairStyle, head: string[]): Cell[] {
	const W = head[0].length, H = head.length;
	if (style === 'bald') return [];
	const out: Cell[] = [];
	const on = (x: number, y: number) => head[y]?.[x] !== undefined && head[y][x] !== '.';
	const sh = (x: number, y: number) => (x >= W - 2 && y <= 1) || (x === W - 1) ? 'H' : 'h';
	if (style === 'mohawk') {
		for (let x = 1; x < W - 1; x++) out.push([x, 0, 'h']);
		for (let x = 2; x < W - 1; x++) out.push([x, -1, x === W - 2 ? 'H' : 'h']);
		return out;
	}
	for (let y = 0; y < 2; y++) for (let x = 0; x < W; x++) if (on(x, y)) out.push([x, y, sh(x, y)]);
	out.push([0, 2, 'h'], [1, 2, 'h']);
	if (style === 'long') {
		for (let y = 3; y < H; y++) out.push([0, y, 'h']);
		out.push([0, H, 'H'], [0, H + 1, 'H'], [1, H, 'H'], [W - 1, 2, 'H']);
	} else if (style === 'topknot') out.push([2, -1, 'h'], [2, -2, 'h'], [3, -1, 'H']);
	else if (style === 'curly') out.push([1, -1, 'h'], [3, -1, 'h'], [W - 2, -1, 'H'], [W - 1, 2, 'H']);
	else if (style === 'ponytail') out.push([-1, 2, 'h'], [-1, 3, 'H'], [-2, 4, 'H'], [-1, 1, 'h']);
	else if (style === 'braid') {
		out.push([-1, 2, 'h'], [-1, 3, 'H'], [-1, 4, 'h'], [-1, 5, 'H'], [-1, 6, 'h'], [-1, 7, 'y']);
	} else if (style === 'bob') {
		for (let y = 2; y < H - 1; y++) out.push([0, y, 'h'], [W - 1, y, 'H']);
		out.push([0, H - 1, 'H']);
	}
	return out;
}
function beardCells(big: boolean, W: number, H: number): Cell[] {
	const out: Cell[] = [];
	for (let x = 0; x < W; x++) out.push([x, H - 2, x === W - 1 ? 'H' : 'h']);
	for (let x = 1; x < W; x++) out.push([x, H - 1, x >= W - 2 ? 'H' : 'h']);
	out.push([2, H - 2, 'K']); // mouth
	if (big) {
		for (let x = 1; x < W - 1; x++) out.push([x, H, x >= W - 2 ? 'H' : 'h']);
		for (let x = 2; x < W - 1; x++) out.push([x, H + 1, x === W - 2 ? 'H' : 'h']);
		out.push([3, H + 2, 'H']);
	} else for (let x = 2; x < W - 1; x++) out.push([x, H, 'H']);
	return out;
}
function earCells(kind: NonNullable<Sp['ears']>, W: number): Cell[] {
	switch (kind) {
		case 'point': return [[-1, 3, 'k'], [-1, 2, 'k'], [-2, 1, 'K'], [W, 3, 'K'], [W, 2, 'K'], [W + 1, 1, 'K']];
		case 'small': return [[-1, 3, 'k'], [-1, 2, 'K'], [W, 3, 'K']];
		case 'long': return [[-1, 3, 'k'], [-2, 3, 'k'], [-3, 2, 'K'], [W, 3, 'K'], [W + 1, 2, 'K']];
		case 'rat': return [[-1, -1, 'k'], [0, -1, 'k'], [-1, 0, 'q'], [0, 0, 'k'], [3, -2, 'k'], [4, -2, 'K'], [3, -1, 'q'], [4, -1, 'K']];
		case 'fin': return [[-1, 3, 'c'], [-1, 2, 'c'], [-2, 1, 'u'], [-2, 2, 'u'], [W, 3, 'u'], [W, 2, 'c']];
	}
}
function snoutCells(s: number, W: number, rat: boolean): Cell[] {
	if (s === 1) return [[W, 4, 'k']];
	if (s === 2) return [[W, 3, 'k'], [W, 4, 'k'], [W + 1, 4, 'K'], [W, 5, 'K']];
	if (s === 3) return [[W, 3, 'k'], [W + 1, 3, 'k'], [W + 2, 3, rat ? 'q' : 'K'], [W, 4, 'K'], [W + 1, 4, 'K']];
	return [];
}
function hornCells(kind: NonNullable<Sp['horns']>, W: number, c: [string, string]): Cell[] {
	const [a, b] = c;
	switch (kind) {
		case 'small': return [[1, -1, a], [W - 2, -1, a]];
		case 'big': return [[1, -1, a], [0, -2, b], [W - 2, -1, a], [W - 1, -2, b]];
		case 'ram': return [[0, -1, a], [-1, 0, a], [-2, 1, b], [-2, 2, a], [-1, 3, b], [W - 1, -1, a], [W, 0, a], [W + 1, 1, b]];
	}
}

// ---------------------------------------------------------------- outfits

type HeadGear = 'none' | 'helm' | 'greathelm' | 'hood' | 'cap' | 'hat' | 'pointy' | 'crown' | 'circlet' | 'turban' | 'goggles' | 'mask' | 'bandana' | 'mitre' | 'feathered' | 'widebrim' | 'cowl' | 'wreath';
type Back = 'none' | 'cloak' | 'cape' | 'pack' | 'quiver';
type Item = 'none' | 'sword' | 'dagger' | 'axe' | 'mace' | 'hammer' | 'spear' | 'staff' | 'bow' | 'gun' | 'wand' | 'whip' | 'knives' | 'purse' | 'book' | 'lantern' | 'wrench' | 'orb' | 'lute' | 'flask' | 'scroll' | 'rapier' | 'horn' | 'greataxe' | 'censer';
type Body = 'tunic' | 'robe' | 'coat' | 'apron' | 'rags';
interface Kit {
	cloth: string[]; // accent colour choices
	body: Body;
	legs: [string, string];
	boots: [string, string];
	belt: string;
	head: HeadGear;
	mask?: boolean;
	back: Back;
	item: Item;
	shield?: boolean;
	armor: number;
	fine?: boolean;
	tabard?: boolean;
	stripes?: boolean;
	bare?: boolean; // bare arms
	ermine?: boolean;
	apronC?: [string, string];
	cl: [string, string, string]; // cloak/hood/hat colours (base, shadow, light)
	tip?: string[]; // staff head, top to bottom
}
const kit = (o: Partial<Kit>): Kit => ({ cloth: ['#7a5a3a'], body: 'tunic', legs: ['w', 'W'], boots: ['W', 'z'], belt: 'W', head: 'none', back: 'none', item: 'none', armor: 0, cl: ['A', 'z', 'a'], ...o });

const KITS: Record<string, Kit> = {
	// callings (the player)
	warrior: kit({ cloth: ['#8a2a2a'], head: 'helm', back: 'cape', item: 'sword', shield: true, armor: 2, legs: ['M', 'G'], cl: ['a', 'A', 'a'] }),
	scout: kit({ cloth: ['#4a6a3a'], head: 'hood', back: 'quiver', item: 'bow', armor: 1, legs: ['T', 'W'], cl: ['e', 'E', 'e'] }),
	scholar: kit({ cloth: ['#3a4a8a'], body: 'robe', head: 'cap', item: 'book', cl: ['A', 'z', 'a'], fine: true }),
	tinker: kit({ cloth: ['#8a5a2a'], head: 'goggles', back: 'pack', item: 'wrench', armor: 1, body: 'apron', legs: ['M', 'G'] }),
	mystic: kit({ cloth: ['#5a3a8a'], body: 'robe', head: 'hood', item: 'staff', cl: ['a', 'A', 'a'], fine: true, tip: ['c', 'Y'] }),
	merchant: kit({ cloth: ['#8a2a4a'], body: 'coat', head: 'hat', item: 'purse', fine: true, cl: ['A', 'z', 'a'], legs: ['T', 'W'] }),
	rogue: kit({ cloth: ['#3a3a44'], head: 'hood', mask: true, back: 'cloak', item: 'knives', armor: 1, legs: ['G', 'z'], boots: ['z', 'z'], cl: ['d', 'D', 'F'] }),
	// roles
	guard: kit({ cloth: ['#4a5a8a', '#7a4a2a', '#2a6a4a'], head: 'helm', item: 'spear', shield: true, armor: 2, tabard: true, legs: ['M', 'G'] }),
	soldier: kit({ cloth: ['#7a3a3a', '#4a5a4a', '#3a4a7a'], head: 'helm', item: 'sword', armor: 2, tabard: true, legs: ['M', 'G'] }),
	knight: kit({ cloth: ['#2a4a8a', '#8a2a2a', '#e0d8c0'], head: 'greathelm', back: 'cape', item: 'sword', shield: true, armor: 3, tabard: true, cl: ['a', 'A', 'a'] }),
	merchantR: kit({ cloth: ['#8a2a4a', '#9a6a1a', '#2a6a5a'], body: 'coat', head: 'hat', item: 'purse', fine: true, legs: ['T', 'W'] }),
	smith: kit({ cloth: ['#5a4a3a', '#6a3a2a', '#4a4a52'], body: 'apron', bare: true, item: 'hammer', legs: ['G', 'z'] }),
	cook: kit({ cloth: ['#8a6a4a', '#5a6a8a'], body: 'apron', apronC: ['x', 'g'], head: 'cap', cl: ['x', 'g', 'x'] }),
	priest: kit({ cloth: ['#e8e4d8', '#d8d0b0'], body: 'robe', head: 'circlet', item: 'staff', fine: true, tip: ['y', 'Y'] }),
	monk: kit({ cloth: ['#c8803a', '#8a5a2a', '#7a7a6a'], body: 'robe', item: 'staff', tip: ['W'], belt: 'Y' }),
	elder: kit({ cloth: ['#8a8a98', '#6a5a7a'], body: 'robe', item: 'staff', tip: ['w', 'W'] }),
	mage: kit({ cloth: ['#3a3a9a', '#6a2a8a', '#2a6a7a'], body: 'robe', head: 'pointy', item: 'staff', fine: true, cl: ['a', 'A', 'a'], tip: ['c', 'Y'] }),
	scholarR: kit({ cloth: ['#3a4a8a', '#4a3a7a'], body: 'robe', head: 'cap', item: 'book', cl: ['A', 'z', 'a'] }),
	apothecary: kit({ cloth: ['#3a6a4a', '#5a7a3a'], body: 'coat', head: 'cap', item: 'lantern', cl: ['E', 'z', 'e'] }),
	bandit: kit({ cloth: ['#5a4a2a', '#3a3a3a', '#6a2a2a'], head: 'bandana', item: 'dagger', armor: 1, legs: ['G', 'z'] }),
	thief: kit({ cloth: ['#3a3a44', '#4a3a2a'], head: 'hood', mask: true, back: 'cloak', item: 'knives', armor: 1, legs: ['G', 'z'], boots: ['z', 'z'], cl: ['d', 'D', 'F'] }),
	hunter: kit({ cloth: ['#4a6a3a', '#6a5a3a'], head: 'hood', back: 'quiver', item: 'bow', armor: 1, legs: ['T', 'W'], cl: ['E', 'z', 'e'] }),
	huntsman: kit({ cloth: ['#3a5a34', '#5a4a2e'], head: 'hood', back: 'quiver', item: 'bow', armor: 1, legs: ['T', 'W'], cl: ['E', 'z', 'e'] }),
	warden: kit({ cloth: ['#2e523a', '#485838'], head: 'cap', back: 'cloak', item: 'spear', armor: 2, tabard: true, legs: ['G', 'z'], cl: ['E', 'z', 'e'] }),
	notable: kit({ cloth: ['#7a2a5a', '#2a4a7a', '#8a6a1a'], head: 'circlet', back: 'cape', fine: true, cl: ['A', 'z', 'a'], belt: 'Y' }),
	noble: kit({ cloth: ['#7a2a5a', '#2a4a7a', '#8a2a2a'], head: 'crown', back: 'cape', fine: true, ermine: true, cl: ['r', 'R', 'r'], belt: 'Y', legs: ['A', 'z'] }),
	villager: kit({ cloth: ['#8a6a4a', '#6a7a4a', '#7a5a6a', '#5a6a8a', '#8a7a3a', '#6a5a4a', '#4a6a7a', '#8a4a52', '#5a7a5a', '#7a6a3a', '#6a4a7a', '#9a6a4a'] }),
	farmer: kit({ cloth: ['#8a7a3a', '#6a7a4a', '#7a5a3a'], head: 'hat', cl: ['t', 'T', 'x'], legs: ['T', 'W'] }),
	sailor: kit({ cloth: ['#3a5a8a', '#2a3a6a'], head: 'bandana', stripes: true, legs: ['x', 'g'] }),
	beggar: kit({ cloth: ['#7a6a50', '#5a5040', '#6a5a4a'], body: 'rags', legs: ['T', 'W'], boots: ['k', 'K'], belt: 'T' }),
	desert: kit({ cloth: ['#c8b080', '#a08a60'], head: 'turban', cl: ['x', 'g', 'x'], legs: ['t', 'T'] }),
	miner: kit({ cloth: ['#6a5a4a', '#5a5a5a'], head: 'helm', item: 'hammer', legs: ['G', 'z'] }),
	inquisitor: kit({ cloth: ['#6e1d26', '#2e2a34', '#4e1e2c'], body: 'coat', head: 'widebrim', back: 'cloak', item: 'censer', armor: 1, fine: true, cl: ['z', 'z', 'G'], legs: ['G', 'z'], belt: 'Y' }),
	pilgrim: kit({ cloth: ['#8a6f48', '#745e40', '#6d6252'], body: 'robe', head: 'widebrim', back: 'pack', item: 'staff', cl: ['w', 'W', 't'], tip: ['w', 'W'], belt: 'T' }),
	chronicler: kit({ cloth: ['#38487a', '#5a324e', '#325a52'], body: 'robe', head: 'cap', item: 'scroll', fine: true, cl: ['A', 'z', 'a'] }),
	librarian: kit({ cloth: ['#2d5a62', '#4c3866', '#5a4432'], body: 'robe', head: 'goggles', item: 'book', fine: true, cl: ['A', 'z', 'a'] }),
	bard: kit({ cloth: ['#9a2e46', '#2e7458', '#6a3488', '#b06424'], head: 'feathered', back: 'cape', item: 'lute', fine: true, cl: ['a', 'A', 'y'], legs: ['T', 'W'], belt: 'Y' }),
	alchemist: kit({ cloth: ['#3a6a58', '#4a4e78', '#685432'], body: 'apron', apronC: ['t', 'T'], head: 'goggles', back: 'pack', item: 'flask', legs: ['G', 'z'] }),
	fence: kit({ cloth: ['#3b3844', '#4a3642', '#3a443c'], body: 'coat', head: 'hat', back: 'cloak', item: 'purse', cl: ['G', 'z', 'g'], legs: ['G', 'z'], boots: ['z', 'z'] }),
	nomad: kit({ cloth: ['#b88248', '#9a5838', '#c8a66a'], head: 'turban', back: 'cloak', item: 'spear', cl: ['t', 'T', 'x'], legs: ['T', 'W'], tip: ['x', 'm'] }),
	assassin: kit({ cloth: ['#24242c', '#362028', '#202c2a'], head: 'cowl', mask: true, back: 'cloak', item: 'knives', armor: 1, cl: ['z', 'z', 'G'], legs: ['z', 'z'], boots: ['z', 'z'] }),
	artificer: kit({ cloth: ['#7c5028', '#4a5a68', '#6a422c'], body: 'apron', head: 'goggles', back: 'pack', item: 'wrench', armor: 1, fine: true, legs: ['M', 'G'], belt: 'Y' }),
	druid: kit({ cloth: ['#345c36', '#4a5e32', '#524834'], body: 'robe', head: 'wreath', back: 'cloak', item: 'staff', cl: ['E', 'z', 'e'], tip: ['v', 'e'], belt: 'W' }),
	herald: kit({ cloth: ['#2a4e8c', '#982834', '#2e6a4a'], head: 'feathered', back: 'cape', item: 'horn', tabard: true, fine: true, cl: ['y', 'Y', 'x'], legs: ['x', 'g'], belt: 'Y' }),
	executioner: kit({ cloth: ['#262428', '#4c1e22'], head: 'cowl', mask: true, bare: true, item: 'greataxe', armor: 1, cl: ['z', 'z', 'G'], legs: ['z', 'z'], boots: ['z', 'z'], belt: 'M' }),
	duellist: kit({ cloth: ['#2c5868', '#8a2a38', '#343c68'], body: 'coat', head: 'feathered', back: 'cape', item: 'rapier', armor: 1, fine: true, cl: ['a', 'A', 'x'], legs: ['G', 'z'], belt: 'Y' }),
};
const ROLE_ALIAS: [RegExp, string][] = [
	[/^warden/, 'warden'],
	[/^huntsman/, 'huntsman'],
	[/inquisitor|witch[- ]?hunter|zealot|templar[- ]?judge/, 'inquisitor'],
	[/executioner|headsman/, 'executioner'],
	[/duell?ist|fencer|swashbuckler|bravo/, 'duellist'],
	[/herald|crier|trumpeter|messenger/, 'herald'],
	[/bard|minstrel|troubadour|skald|musician|singer/, 'bard'],
	[/druid|archdruid|grove[- ]?keeper|wilder/, 'druid'],
	[/artificer|engineer|inventor|clockwork|mechanist/, 'artificer'],
	[/alchem|transmuter|elixir/, 'alchemist'],
	[/chronicler|annalist|historian|archivist/, 'chronicler'],
	[/librar|curator|bookkeeper/, 'librarian'],
	[/pilgrim|wayfarer|penitent/, 'pilgrim'],
	[/fence|smuggler|broker|black[- ]?market/, 'fence'],
	[/assassin|cutthroat|hitman|slayer|shadowblade/, 'assassin'],
	[/nomad|caravan|desert|bedouin|wanderer/, 'nomad'],
	[/^guard|watch|sentr/, 'guard'],
	[/soldier|militia|captain|sergeant|mercenar|warrior|fighter/, 'soldier'],
	[/knight|paladin|champion|templar/, 'knight'],
	[/merchant|trader|shop|vendor|peddler|banker/, 'merchantR'],
	[/smith|armou?rer|carpenter|mason|tinker|wright|cooper/, 'smith'],
	[/miner|digger|quarr/, 'miner'],
	[/cook|innkeep|baker|brewer|butcher|tavern|bartend/, 'cook'],
	[/priest|cleric|acolyte|healer|shrine|oracle|bishop/, 'priest'],
	[/monk|hermit|ascetic/, 'monk'],
	[/elder|chief|sage|headman/, 'elder'],
	[/mage|wizard|sorcer|warlock|witch|magus|mystic|shaman/, 'mage'],
	[/scholar|scribe|teacher|student/, 'scholarR'],
	[/apothecar|herbal|doctor|physician/, 'apothecary'],
	[/bandit|raider|brigand|outlaw|pirate/, 'bandit'],
	[/rogue|thief|spy|cutpurse/, 'thief'],
	[/hunter|scout|ranger|archer|trapper/, 'hunter'],
	[/noble|king|queen|lord|lady|prince|duke|baron|count/, 'noble'],
	[/notable|mayor|magistrate|envoy|official|courtier/, 'notable'],
	[/farm|herd|shepherd|peasant|field|garden/, 'farmer'],
	[/sailor|fisher|boat|deckhand/, 'sailor'],
	[/beggar|vagrant|urchin|refugee|outcast/, 'beggar'],
];
function kitFor(a: QudActor): Kit {
	if (a.p && a.c && KITS[a.c]) return KITS[a.c];
	const r = (a.r ?? '').toLowerCase();
	if (r) {
		if (KITS[r]) return r === 'merchant' ? KITS.merchantR : r === 'scholar' ? KITS.scholarR : KITS[r];
		for (const [re, k] of ROLE_ALIAS) if (re.test(r)) return KITS[k];
	}
	return KITS.villager;
}

const WEAPON: Record<string, Item> = {
	longblade: 'sword', shortblade: 'dagger', axe: 'axe', cudgel: 'mace', flail: 'mace', hook: 'axe', missile: 'bow', spear: 'spear', polearm: 'spear',
	staff: 'staff', whip: 'whip', fist: 'none', thrown: 'knives', firearm: 'gun', heavy: 'hammer', wand: 'wand', mace: 'mace', sword: 'sword',
	dagger: 'dagger', bow: 'bow', gun: 'gun', hammer: 'hammer',
};

// ---------------------------------------------------------------- weapons and held items ('+' = hand)

interface Held {
	rest: string[];
	up?: string[];
	fwd?: string[];
	aim?: string[];
	aim2?: string[];
}
const HELD: Partial<Record<Item, Held>> = {
	sword: { rest: ['.+.', 'YMY', '.m.', '.m.', '.m.'], up: ['.x.', '.m.', '.m.', '.m.', 'YMY', '.+.'], fwd: ['.Y...', '+Mmmx', '.Y...'] },
	dagger: { rest: ['+', 'Y', 'm'], up: ['x', 'm', 'Y', '+'], fwd: ['+Ymx'] },
	axe: { rest: ['+..', 'w..', 'wmx', 'wMm'], up: ['wmx', 'wMm', 'w..', 'w..', '+..'], fwd: ['+wwwm', '...mM'] },
	mace: { rest: ['.+.', '.w.', 'mxm', '.M.'], up: ['.x.', 'mmM', '.w.', '.w.', '.+.'], fwd: ['...xm', '+wwmM', '...MM'] },
	hammer: { rest: ['.+.', '.w.', 'xmM'], up: ['xmM', 'mMM', '.w.', '.w.', '.+.'], fwd: ['...x', '+wwm', '...M'] },
	bow: { rest: ['.w.', 'x.w', 'x.w', 'x.+', 'x.w', 'x.w', '.w.'], aim: ['x.w..', 'x..w.', 'xww+m', 'x..w.', 'x.w..'], aim2: ['.xw..', '.x.w.', '.x.+.', '.x.w.', '.xw..'] },
	gun: { rest: ['.M', '.M', '.M', '.+', '.W', 'WW'], aim: ['......', 'WW+MMM', '.W....'] },
	wand: { rest: ['.c', 'w.', '+.'], up: ['c', 'w', '+'], fwd: ['+wc'] },
	whip: { rest: ['+.', 'W.', 'WW', '.W'], up: ['.W', 'W.', 'W.', '+.'], fwd: ['+WWW.', '....W'] },
	purse: { rest: ['.+', 'yY', 'YY'] },
	book: { rest: ['.+', 'Rx', 'Rx'] },
	lantern: { rest: ['.+.', '.M.', 'MyM', '.M.'] },
	wrench: { rest: ['.+.', '.M.', 'M.M'], up: ['M.M', '.M.', '.M.', '.+.'], fwd: ['+MM.', '...M', '..M.'] },
	orb: { rest: ['xc', 'cc', '+.'] },
	lute: { rest: ['.+.', '.w.', 'wtw', 'tWt'], up: ['tWt', 'wtw', '.w.', '.+.'] },
	flask: { rest: ['.+.', '.x.', 'cvc', '.v.'] },
	scroll: { rest: ['+t', 'tx', 'tx', '.t'] },
	rapier: { rest: ['.+.', 'yYy', '.x.', '.m.', '.x.'], up: ['.x.', '.m.', '.x.', 'yYy', '.+.'], fwd: ['.y....', '+Yxmxm', '.y....'] },
	horn: { rest: ['.+.', '.Y.', 'yYy'], up: ['yYy', '.Y.', '.+.'] },
	greataxe: { rest: ['+...', 'w...', 'wmxm', 'wMmM', 'w..M'], up: ['wmxm', 'wMmM', 'w...', 'w...', '+...'], fwd: ['+wwww', '..mmM', '..xMM'] },
	censer: { rest: ['.+.', '.Y.', 'yry', '.Y.'] },
};
const isPole = (it: Item) => it === 'spear' || it === 'staff';

// shields use the cell colour ramp so they read as the owner's colours
const SHIELD = ['lFF', 'lmF', 'FFd', '.d.'];
const SHIELD_FRONT = ['llFF', 'lFmF', 'lFFd', 'FFdd', '.Fd.'];

// ---------------------------------------------------------------- arms

type ArmState = 'down' | 'fwd' | 'back' | 'up' | 'strike' | 'guard' | 'cast' | 'aim' | 'hold';
/** sleeve cells and hand, as (forward, down) offsets from the shoulder. `out` shapes mirror for the far arm. */
function armShape(s: ArmState, L: number): { cells: [number, number][]; hand: [number, number]; out: boolean } {
	const col = (n: number): [number, number][] => Array.from({ length: n }, (_, i) => [0, i] as [number, number]);
	switch (s) {
		case 'down': return { cells: col(L), hand: [0, L], out: false };
		case 'hold': return { cells: col(L - 1), hand: [0, L - 1], out: false };
		case 'fwd': return { cells: col(L - 1), hand: [1, L - 1], out: false };
		case 'back': return { cells: col(L - 1), hand: [-1, L - 1], out: false };
		case 'up': {
			const n = Math.max(2, L - 2);
			return { cells: Array.from({ length: n }, (_, i) => [0, -i] as [number, number]), hand: [0, -n], out: false };
		}
		case 'strike': return L >= 4 ? { cells: [[0, 0], [0, 1]], hand: [-1, 2], out: false } : { cells: [[0, 0]], hand: [-1, 1], out: false };
		case 'guard': return { cells: [[0, 0]], hand: [-1, 1], out: false };
		case 'cast': return { cells: [[0, 0], [1, -1]], hand: [1, -2], out: true };
		case 'aim': return { cells: [[0, 0], [1, 0]], hand: [1, 1], out: true };
	}
}

// ---------------------------------------------------------------- the composer

type State = 'idle' | 'walk' | 'windup' | 'strike' | 'cast' | 'block' | 'hurt' | 'sleep' | 'dead';
function stateOf(p: Pose): State {
	if ((p.dead ?? 0) > 0.5) return 'dead';
	if ((p.sleep ?? 0) > 0) return 'sleep';
	if ((p.hurt ?? 0) > 0 || (p.dead ?? 0) > 0) return 'hurt';
	if ((p.attack ?? 0) > 0) return p.attack < 0.4 ? 'windup' : 'strike';
	if ((p.cast ?? 0) > 0) return 'cast';
	if ((p.block ?? 0) > 0) return 'block';
	if ((p.work ?? 0) > 0) return p.workKind === 'sweep' ? 'walk' : p.work < 0.5 ? 'windup' : 'strike';
	if ((p.walk ?? 0) > 0) return 'walk';
	return 'idle';
}

function parseTribe(a: QudActor): { skin?: string; accent?: string } | null {
	const srcs = [a.tr, ...(a.fl ?? []), a.k, a.s];
	for (const s of srcs) {
		const m = s ? /tr_hex:(#[0-9a-fA-F]{6})(?::(#[0-9a-fA-F]{6}))?/.exec(s) : null;
		if (m) return { skin: m[1], accent: m[2] };
	}
	return null;
}

/** worn items from the engine (paper doll, portraits) override the kit's head, back, shield and feet */
function applyGear(k: Kit, a: QudActor): Kit {
	const o = { ...k };
	const hd = (a.hd ?? '').toLowerCase();
	o.mask = false;
	o.head = !hd ? 'none'
		: /crown|diadem|tiara/.test(hd) ? 'crown'
		: /circlet|laurel|halo/.test(hd) ? 'circlet'
		: /great ?helm|close helm|armet|bascinet|visor/.test(hd) ? 'greathelm'
		: /helm|kabuto|casque|sallet|morion|shako/.test(hd) ? 'helm'
		: /mitre|biretta/.test(hd) ? 'mitre'
		: /feather|plume|tricorne|beret/.test(hd) ? 'feathered'
		: /wide[- ]?brim|pilgrim|inquisitor|hunter hat/.test(hd) ? 'widebrim'
		: /wreath|antler|leaf crown/.test(hd) ? 'wreath'
		: /wizard|pointed|witch/.test(hd) ? 'pointy'
		: /cowl|executioner/.test(hd) ? 'cowl'
		: /hood/.test(hd) ? 'hood'
		: /turban|pagri|kolah|keffiyeh|headcloth/.test(hd) ? 'turban'
		: /goggle|lens|monocle/.test(hd) ? 'goggles'
		: /mask|veil|face/.test(hd) ? 'mask'
		: /bandana|kerchief|scarf/.test(hd) ? 'bandana'
		: /hat|caping|malakhai|sedge|straw|brim/.test(hd) ? 'hat'
		: 'cap';
	const bk = (a.bk ?? '').toLowerCase();
	o.back = bk ? (/cape|mantle|haori|shawl|sash/.test(bk) ? 'cape' : /pack|bag|satchel/.test(bk) ? 'pack' : /quiver/.test(bk) ? 'quiver' : 'cloak') : o.back === 'quiver' || o.back === 'pack' ? o.back : 'none';
	o.shield = !!a.sh;
	const ft = (a.ft ?? '').toLowerCase();
	o.boots = ft ? (/greave|sabaton|plate|steel|iron|mail|armou?r/.test(ft) ? ['m', 'M'] : /sandal/.test(ft) ? ['t', 'K'] : ['W', 'z']) : ['k', 'K'];
	return o;
}

interface Look {
	sp: Sp;
	b: Build;
	k: Kit;
	item: Item;
	armor: number;
	hairStyle: HairStyle;
	beard: 0 | 1 | 2;
	tip: string[];
}

function lookOf(a: QudActor, h: number): Look {
	const sp = speciesFor(a.s ?? 'human');
	let k = kitFor(a);
	if (a.ge) k = applyGear(k, a);
	if (a.k && /high|noble|lord|elite/i.test(a.k)) k = { ...k, fine: true };
	let bId = sp.b;
	if (sp.slimOdds && !a.p && hashN(h, 3) % sp.slimOdds === 0) bId = 'slim';
	let item: Item = k.item;
	if (a.w) item = WEAPON[a.w] ?? item;
	else if (a.p && a.ge) item = 'none';
	const armor = a.a !== undefined && a.a >= 0 ? Math.min(3, a.a) : k.armor;
	let head = k.head;
	if (!a.ge && armor >= 3 && head === 'helm') head = 'greathelm';
	if (head !== k.head) k = { ...k, head };
	const styles = sp.hair ? sp.styles ?? ['short'] : ['bald' as HairStyle];
	const hairStyle = a.p ? styles[0] : pick(styles, hashN(h, 5));
	const beard: 0 | 1 | 2 = sp.beard === 2 ? 2 : sp.beard === 1 && hashN(h, 7) % 4 === 0 ? 1 : 0;
	const tip = k.tip ?? (item === 'spear' ? ['x', 'm'] : ['W']);
	return { sp, b: BUILDS[bId], k, item, armor, hairStyle, beard, tip };
}

/** slot letters to palette letters for this outfit and armour */
function slotMap(L: Look): Record<string, string> {
	const { k, armor, sp } = L;
	const m: Record<string, string> = {
		'1': 'a', '2': 'A', '3': 'a', '9': k.fine ? 'y' : 'A', S: 'a', B: 'A', '4': k.legs[0], '5': k.legs[1], '6': k.boots[0], '8': k.boots[1],
		'7': k.fine && k.belt === 'W' ? 'Y' : k.belt, '0': 'k', '#': sp.eye, C: k.cl[0], V: k.cl[1], I: k.cl[2],
	};
	if (armor === 1) Object.assign(m, { '1': 'w', '2': 'W', '3': 't', '9': 'T' });
	if (armor === 2) Object.assign(m, { '1': 'g', '2': 'G', '3': 'm', '9': 'G', S: 'g', B: 'G' });
	if (armor >= 3) Object.assign(m, { '1': 'm', '2': 'M', '3': 'x', '9': 'M', S: 'm', B: 'M', '0': 'M', '4': 'M', '5': 'G', '6': 'm', '8': 'M' });
	if (k.bare || sp.bone) Object.assign(m, { S: 'k', B: 'K' });
	if (sp.feet === 'bare' && k.boots[0] !== 'm') Object.assign(m, { '6': 'k', '8': 'K' });
	if (sp.feet === 'hoof' && armor < 3) Object.assign(m, { '6': 'z', '8': 'z' });
	if (sp.feet === 'claw' && k.boots[0] === 'W') Object.assign(m, { '6': 'K', '8': 'K' });
	return m;
}

function legFrame(st: State, phase: number): number {
	if (st === 'walk') return [1, 2, 3, 4][phase & 3];
	if (st === 'strike') return 1;
	return 0;
}

function legsGrid(b: Build, frame: number): string[] {
	const src = (b.legs === 't' ? LEGS_T : LEGS_N)[frame];
	return (LEG_ROWS[b.lrows] ?? LEG_ROWS[5]).map((i) => src[i]);
}

/** a robe or coat skirt drawn over the leg mask: spans each row, flares at the hem, feet peek out */
function skirt(legs: string[], rowsCovered: number, flare: boolean): string[] {
	return legs.map((row, y) => {
		if (y >= rowsCovered) return row;
		let x0 = row.search(/[^.]/), x1 = row.length - 1 - row.split('').reverse().join('').search(/[^.]/);
		if (x0 < 0) return row;
		if (flare && y >= rowsCovered - 2) {
			x0 = Math.max(0, x0 - 1);
			x1 = Math.min(row.length - 1, x1 + 1);
		}
		const last = y === legs.length - 1;
		const out = row.split('');
		for (let x = x0; x <= x1; x++) {
			if (last && (out[x] === '6' || out[x] === '8')) continue;
			out[x] = y === rowsCovered - 1 ? '2' : x === x0 ? '3' : x === x1 ? '2' : '1';
		}
		return out.join('');
	});
}

function outline(g: Grid): void {
	const filled = g.map((r) => r.map((c) => c !== '.'));
	for (let y = 0; y < PH; y++) for (let x = 0; x < PW; x++) {
		if (filled[y][x]) continue;
		if ((x > 0 && filled[y][x - 1]) || (x < PW - 1 && filled[y][x + 1]) || (y > 0 && filled[y - 1][x]) || (y < PH - 1 && filled[y + 1][x])) g[y][x] = 'o';
	}
}

// generic lying body (fill only): head left, feet right; eyes '#'
const LIE = ['....kk......', '..hk#k......', '.hhkkK3S0.6.', '.hhhK117446.', '..hh.227558.'];

function swarm(h: number, frame: number): PersonPixels {
	const g = grid();
	const t = frame >> 1;
	const spots: [number, number][] = [[5, 4], [3, 7], [7, 7], [5, 9], [2, 11], [8, 11], [4, 13], [6, 13], [3, 16], [8, 16], [5, 17], [6, 6]];
	for (let i = 0; i < spots.length; i++) {
		const j = hashN(h + t, i);
		const x = spots[i][0] + ((j & 3) === 0 ? 1 : (j & 3) === 1 ? -1 : 0), y = spots[i][1] + ((j >> 2) & 1);
		put(g, x, y, 'k');
		put(g, x + 1, y, 'K');
		if (i % 3 === 0) put(g, x + 1, y - 1, 'r');
	}
	outline(g);
	return { rows: g.map((r) => r.join('')), pal: { skin: '#3a2a1c', hair: '#2a2218', accent: '#5a1a1a' } };
}

export function personSprite(a: QudActor, h: number, pose: Pose, frame: number): PersonPixels | null {
	if (a.s === 'chitin_swarm') return swarm(h, frame);
	const L = lookOf(a, h);
	const { sp, b, k } = L;
	const st = stateOf(pose);
	const face = pose.face === -1 ? -1 : 1;
	const localDir = (pose.dir === -1 ? -1 : 1) * face;

	// palette
	const tribe = parseTribe(a);
	const skin = tribe?.skin ?? pick(sp.skin, hashN(h, 11));
	const hair = sp.hair ? (/elder/.test(a.r ?? '') ? '#d8d0c0' : pick(sp.hair, hashN(h, 13))) : skin;
	const accent = tribe?.accent ?? (a.p ? k.cloth[0] : pick(k.cloth, hashN(h, 17)));
	const pal = { skin, hair, accent };

	const g = grid();
	const fx = grid();
	const map = slotMap(L);

	if (st === 'sleep' || st === 'dead') {
		const lie = LIE.map((r) => (sp.hair && L.hairStyle !== 'bald' ? r : r.replace(/[hH]/g, 'K')));
		stamp(g, lie, 0, 14);
		if (k.body === 'robe') for (let y = 15; y < 19; y++) for (let x = 6; x < 10; x++) if (/[45]/.test(g[y][x])) g[y][x] = '1';
		if (sp.snout) put(g, 6 - 1, 14, 'k');
		if (sp.horns) put(g, 1, 15, sp.hornC?.[0] ?? 't');
		if (sp.ears === 'point' || sp.ears === 'long') put(g, 2, 14, 'k');
		if (sp.tusks) put(g, 5, 15, 'x');
		if (sp.tail === 'lizard' || sp.tail === 'rat') put(g, 10, 18, sp.tail === 'rat' ? 'q' : 'k'), put(g, 10, 17, '.');
		if (sp.bone) for (let x = 6; x < 8; x++) if (g[17][x] === '1') g[17][x] = 'K';
		map['#'] = 'K';
		outline(g);
		if (st === 'sleep') {
			const z = ((frame >> 2) & 1) === 0 ? [[5, 10], [6, 10], [5, 11], [4, 12], [5, 12]] : [[7, 8], [8, 8], [7, 9], [6, 10], [7, 10]];
			for (const [x, y] of z) put(fx, x, y, 'x');
		} else for (const x of [4, 5, 6, 7, 8]) put(fx, x, 19, 'R');
		let rows = finish(g, fx, map);
		const flip = st === 'dead' ? localDir * face > 0 : face < 0;
		if (flip) rows = rows.map((r) => r.split('').reverse().join(''));
		return { rows, pal };
	}

	// ---- pose parameters
	const contact = st === 'walk' && (pose.phase & 1) === 0;
	const breathe = st === 'idle' && (((frame >> 2) + (h & 1)) & 1) === 1;
	const dy = contact || st === 'strike' ? 1 : 0; // upper body drop
	const dx = st === 'hurt' ? localDir : 0; // upper body recoil
	const hdy = dy + (breathe ? 1 : 0);
	const legF = legFrame(st, pose.phase);
	const item = L.item;
	const pole = isPole(item);
	const ranged = item === 'bow' || item === 'gun';
	const melee = !ranged && item !== 'none' && !['purse', 'book', 'lantern', 'orb', 'lute', 'flask', 'scroll', 'horn', 'censer'].includes(item);

	let nearArm: ArmState = 'down', farArm: ArmState = 'down';
	let ori: keyof Held | 'pole' | 'poleUp' | 'poleFwd' | 'hide' = 'rest';
	if (st === 'walk') {
		const ph = pose.phase & 3;
		nearArm = ph === 0 ? 'back' : ph === 2 ? 'fwd' : 'down';
		farArm = ph === 0 ? 'fwd' : ph === 2 ? 'back' : 'down';
	}
	if (pole && (st === 'idle' || st === 'walk' || st === 'hurt')) {
		nearArm = 'hold';
		ori = 'pole';
	}
	if (st === 'windup') {
		if (ranged) {
			nearArm = item === 'bow' ? 'aim' : 'strike';
			farArm = 'fwd';
			ori = 'aim';
		} else if (pole) {
			nearArm = item === 'staff' ? 'up' : 'strike';
			ori = item === 'staff' ? 'poleUp' : 'poleFwd';
		} else {
			nearArm = 'up';
			farArm = 'fwd';
			ori = melee ? 'up' : 'hide';
		}
	} else if (st === 'strike') {
		if (ranged) {
			nearArm = item === 'bow' ? 'aim' : 'strike';
			farArm = 'fwd';
			ori = item === 'bow' ? 'aim2' : 'aim';
		} else {
			nearArm = melee || pole ? 'strike' : 'aim';
			farArm = 'back';
			ori = pole ? 'poleFwd' : melee ? 'fwd' : 'hide';
		}
	} else if (st === 'cast') {
		if (item === 'staff') {
			nearArm = 'up';
			ori = 'poleUp';
			farArm = 'cast';
		} else {
			nearArm = 'cast';
			farArm = 'fwd';
			ori = item === 'wand' || item === 'orb' ? 'up' : 'hide';
		}
	} else if (st === 'block') {
		nearArm = 'guard';
		farArm = 'fwd';
		ori = pole ? 'poleUp' : melee ? 'up' : 'hide';
	} else if (st === 'hurt') {
		nearArm = 'cast';
		farArm = 'cast';
		if (!pole) ori = 'rest';
	}
	if (k.shield && st !== 'block' && st !== 'hurt') farArm = 'down';

	// ---- head box
	const head = HEADS[sp.head];
	const W = head[0].length, H = head.length;
	const snout = sp.snout ?? 0;
	let hx = 6 - Math.floor(W / 2) - (snout >= 2 ? 1 : 0);
	let hy = b.hb - H;
	if (sp.horns === 'big' && hy < 2) hy = 2;
	hx += dx;
	hy += hdy;
	const tw = b.torso[0].length;
	const ty = b.ty + dy;
	const ly = b.ty + b.torso.length;

	// ---- back layer: wings, cloak, pack, quiver, tail
	if (sp.wings) {
		const flap = ((frame >> 1) & 1) === 1 || st === 'cast' ? -1 : 0;
		for (const [x, y, c] of [[1, 5, 'x'], [2, 6, 'x'], [1, 6, 'm'], [1, 7, 'x'], [1, 8, 'm'], [1, 9, 'g'], [10, 5, 'x'], [9, 6, 'x'], [10, 6, 'm'], [10, 7, 'x'], [10, 8, 'm'], [10, 9, 'g']] as Cell[]) put(g, x, y + ty - 9 + flap, c);
	}
	if (k.back === 'cloak' || k.back === 'cape') {
		const bottom = k.back === 'cloak' ? 17 : Math.min(17, ly + 1);
		for (let y = ty; y <= bottom; y++) {
			const wide = y > ty + 1;
			const x0 = b.tx - (wide ? 2 : 1) + dx, x1 = b.tx + tw + (wide ? 1 : 0) + dx;
			const trail = contact && y >= bottom - 1 ? -1 : 0;
			for (let x = x0; x <= x1; x++) put(g, x + trail, y, y === bottom ? 'V' : x === x0 ? 'I' : x === x1 ? 'V' : 'C');
		}
	} else if (k.back === 'pack') {
		stamp(g, ['tt.', 'wW.', 'wW.', 'WW.'], b.al - 2 + dx, ty - 1);
	} else if (k.back === 'quiver') {
		stamp(g, ['x.r', '.W.', 'W..'], b.al - 1 + dx, ty - 2);
	}
	if (sp.tail === 'lizard') {
		const sw = st === 'walk' ? pose.phase & 1 : (frame >> 2) & 1;
		stamp(g, ['..k', '.kK', 'kK.', sw ? 'K..' : 'kK.'], b.tx - 3, ly - 1);
	} else if (sp.tail === 'rat') {
		const sw = (frame >> 2) & 1;
		stamp(g, ['..q', '.q.', 'q..', sw ? 'q..' : '.q.', sw ? '.q.' : 'q..'], b.tx - 3, ly - 1);
	}

	// ---- arms
	const drawArm = (s: ArmState, near: boolean): [number, number] => {
		const sh = armShape(s, b.L);
		const sx = (near ? b.ar : b.al) + dx;
		const sgn = sh.out && !near ? -1 : 1;
		const outward = near ? 1 : -1;
		const sleeve = near ? 'S' : 'B';
		sh.cells.forEach(([cx, cy], i) => {
			const x = sx + cx * sgn, y = ty + cy;
			const c = k.body === 'rags' && i >= sh.cells.length - 1 && !L.armor ? 'k' : sleeve;
			put(g, x, y, i === 0 && !near && L.armor < 2 ? '3' : c);
			if (b.aw === 2 && cx === 0) put(g, x + outward, y, near ? 'B' : sleeve);
		});
		const hxp = sx + sh.hand[0] * sgn, hyp = ty + sh.hand[1];
		put(g, hxp, hyp, '0');
		if (b.aw === 2 && sh.hand[0] === 0) put(g, hxp + outward, hyp, '0');
		return [hxp, hyp];
	};
	const farHand = drawArm(farArm, false);
	if (sp.arms4) {
		for (let i = 0; i < b.L - 2; i++) {
			put(g, b.al - 1 + dx, ty + 2 + i, 'B');
			put(g, b.ar + 1 + dx, ty + 2 + i, 'S');
		}
		put(g, b.al - 1 + dx, ty + b.L, '0');
		put(g, b.ar + 1 + dx, ty + b.L, '0');
	}
	if (k.shield && st !== 'block') stamp(g, SHIELD, farHand[0] - 1, Math.min(farHand[1] - 3, 15));
	if (item === 'knives' && (st === 'idle' || st === 'walk' || st === 'strike')) stampAt(g, HELD.dagger!.rest, farHand[0], farHand[1]);

	// ---- legs
	let legs = legsGrid(b, legF);
	if (sp.tail === 'fish') {
		const sway = st === 'walk' ? [0, 1, 0, -1][pose.phase & 3] : 0;
		stamp(g, FISH.slice(0, 2), b.tx + tw / 2 - 3, ly);
		stamp(g, FISH.slice(2), b.tx + tw / 2 - 3 + sway, ly + 2);
	} else {
		if (k.body === 'robe') legs = skirt(legs, legs.length, true);
		else if (k.body === 'coat') legs = skirt(legs, Math.min(2, legs.length - 1), false);
		if (k.body === 'apron') legs = legs.map((r, y) => (y < 2 ? r.replace(/[45]/g, (c, x: number) => (x === r.search(/[45]/) ? 'w' : 'W')) : r));
		if (k.body === 'rags') legs = legs.map((r, y) => (y === legs.length - 2 ? r.replace(/[45]/g, 'k') : r));
		stamp(g, legs, b.tx + tw / 2 - 4, ly);
	}

	// ---- torso
	const tx = b.tx + dx;
	stamp(g, b.torso, tx, ty);
	if (b.neck) for (let i = 0; i < b.neck[2]; i++) put(g, b.neck[0] + i + dx, b.neck[1] + dy, 'K');
	if (L.armor === 2) for (let y = 0; y < b.torso.length; y++) for (let x = 0; x < tw; x++) if (g[ty + y]?.[tx + x] === '1' && (x + y) % 2 === 1) g[ty + y][tx + x] = '2';
	if (k.tabard && L.armor >= 2) {
		for (let y = 0; y < b.torso.length; y++) for (let x = 1; x < tw - 1; x++) {
			const c = g[ty + y][tx + x];
			if (c !== '7') g[ty + y][tx + x] = x === tw - 2 ? 'A' : 'a';
		}
		put(g, tx + Math.floor(tw / 2) - (tw > 4 ? 1 : 0), ty + 1, 'y');
	}
	if (k.stripes) for (let y = 0; y < b.torso.length; y++) if (y % 2 === 1) for (let x = 0; x < tw; x++) if (/[123]/.test(g[ty + y][tx + x])) g[ty + y][tx + x] = 'x';
	if (k.body === 'apron') {
		const [ac, ad] = k.apronC ?? ['w', 'W'];
		for (let y = 1; y < b.torso.length; y++) for (let x = 1; x < tw - 1; x++) g[ty + y][tx + x] = y === b.torso.length - 2 && !k.apronC ? 'W' : x === tw - 2 ? ad : ac;
		put(g, tx + 1, ty, ad);
		put(g, tx + tw - 2, ty, ad);
		if (k.apronC) for (let y = 0; y < 2; y++) for (let x = 0; x < 8; x++) if (/[45wW]/.test(g[ly + y]?.[b.tx + tw / 2 - 4 + x] ?? '')) g[ly + y][b.tx + tw / 2 - 4 + x] = x < 4 ? ac : ad;
	}
	if (k.body === 'rags') for (let x = 0; x < tw; x += 2) put(g, tx + x, ty + b.torso.length - 1, '.');
	if (sp.bone && L.armor === 0) for (let y = 1; y < b.torso.length - 1; y += 1) for (let x = 1; x < tw - 1; x++) if (y !== b.torso.length - 2) g[ty + y][tx + x] = y % 2 ? 'k' : 'K';
	if (k.body === 'robe' || k.fine) put(g, tx + Math.floor(tw / 2) - 1, ty + b.torso.length - 2, k.fine ? 'y' : '7');
	if (L.armor >= 3) {
		put(g, b.al - 1 + dx, ty, 'x');
		put(g, b.al + dx, ty, 'm');
		put(g, b.ar + dx, ty, 'm');
		put(g, b.ar + 1 + dx, ty, 'M');
	}
	if (k.shield && st === 'block') stamp(g, SHIELD_FRONT, tx + tw - 4, ty);

	// ---- head
	const hg = head.map((r) => r.split(''));
	if (st === 'hurt') for (const r of hg) for (let x = 0; x < r.length; x++) if (r[x] === '#') r[x] = 'K';
	if (sp.scales) for (let y = 1; y < H; y++) for (let x = 0; x < W; x++) if (hg[y][x] === 'k' && (x + 2 * y) % 4 === 0) hg[y][x] = 'K';
	const headCells: Cell[] = [];
	for (let y = 0; y < H; y++) for (let x = 0; x < W; x++) if (hg[y][x] !== '.') headCells.push([x, y, hg[y][x]]);
	const P = (cs: Cell[]) => cs.forEach(([x, y, c]) => put(g, hx + x, hy + y, c));
	if (sp.ears && k.head !== 'hood' && k.head !== 'greathelm') P(earCells(sp.ears, W));
	P(headCells);
	const hairOn = k.head !== 'hood' && k.head !== 'cowl' && k.head !== 'greathelm' && k.head !== 'turban';
	if (hairOn || L.hairStyle === 'long') P(hairCells(hairOn ? L.hairStyle : 'long', head).filter(([, y]) => hairOn || y >= 2));
	if (L.beard && k.head !== 'greathelm') P(beardCells(L.beard === 2, W, H));
	P(gearCells(k, W, H));
	if (snout && k.head !== 'greathelm') P(snoutCells(snout, W, sp.ears === 'rat'));
	if (sp.tusks && k.head !== 'greathelm') P(snout >= 2 ? [[W, 5, 'x'], [W - 2, 5, 'x']] : [[1, H - 1, 'x'], [W - 2, H - 1, 'x'], [1, H - 2, 'x'], [W - 2, H - 2, 'x']]);
	if (sp.horns) P(hornCells(sp.horns, W, sp.hornC ?? ['t', 'T']));
	if (sp.halo) for (let x = 1; x < W - 1; x++) put(fx, hx + x, hy - 2, (x + (frame >> 2)) % 3 ? 'y' : 'x');

	// ---- cloak mantle and clasp over the shoulders
	if (k.back === 'cloak' || k.back === 'cape') {
		for (let x = 0; x < tw; x++) put(g, tx + x, ty, k.ermine ? (x % 2 ? 'x' : 'z') : 'C');
		if (k.ermine) put(g, tx, ty, 'x');
		put(g, tx + Math.floor(tw / 2), ty, k.fine || k.ermine ? 'y' : 'V');
	}

	// ---- near arm and the held item
	const hand = drawArm(nearArm, true);
	const held = HELD[item];
	const [px, py] = hand;
	if (pole) {
		const tip = L.tip;
		if (ori === 'pole' || ori === 'poleUp') {
			const top = ori === 'poleUp' ? 0 : Math.max(0, hy - 2);
			const bottom = ori === 'poleUp' ? Math.min(18, py + 4) : 18;
			const cx = px + 1;
			for (let y = top; y <= bottom; y++) put(g, cx, y, y - top < tip.length ? tip[y - top] : 'w');
			if (ori === 'poleUp' && item === 'staff') for (const [ox, oy] of ((frame >> 1) & 1 ? [[-1, 0], [1, 1]] : [[1, 0], [-1, 1]])) put(fx, cx + ox, top + oy, 'c');
		} else if (ori === 'poleFwd') {
			const reach = st === 'strike' ? 11 : 9;
			for (let x = px - 4; x <= reach; x++) put(g, x, py, x > reach - tip.length ? [...tip].reverse()[reach - x] : 'w');
		}
	} else if (held && ori !== 'hide') {
		const rows = (ori === 'pole' || ori === 'poleUp' || ori === 'poleFwd' ? held.rest : held[ori as keyof Held]) ?? held.rest;
		stampAt(g, rows, px, py);
	}

	// ---- fx: swing trail, muzzle flash, spell sparkles
	if (st === 'strike' && melee && !pole) for (const [x, y] of [[px + 2, py - 4], [px + 3, py - 3], [px + 3, py - 2]]) put(fx, x, y, 'x');
	if (st === 'strike' && item === 'gun') for (const [x, y, c] of [[11, py, 'y'], [11, py - 1, 'n'], [10, py - 2, 'n']] as Cell[]) put(fx, x, y, c);
	if (st === 'strike' && item === 'bow') for (const x of [9, 10, 11]) put(fx, x, py, 'x');
	if (st === 'cast') {
		const spark = (frame >> 1) & 1 ? [[1, -1, 'c'], [-1, -2, 'x'], [1, -3, 'c']] : [[0, -2, 'x'], [2, -2, 'c'], [-1, -1, 'c']];
		const [sx, sy] = item === 'staff' ? farHand : hand;
		for (const [ox, oy, c] of spark as Cell[]) put(fx, sx + ox, sy + oy, c);
	}

	outline(g);
	let rows = finish(g, fx, map);
	if (face < 0) rows = rows.map((r) => r.split('').reverse().join(''));
	return { rows, pal };
}

/** headgear cells relative to the head box */
function gearCells(k: Kit, W: number, H: number): Cell[] {
	const out: Cell[] = [];
	const row = (y: number, x0: number, x1: number, f: (x: number) => string) => {
		for (let x = x0; x <= x1; x++) out.push([x, y, f(x)]);
	};
	switch (k.head) {
		case 'helm':
			row(-1, 1, W - 2, (x) => (x === 1 ? 'x' : x >= W - 2 ? 'M' : 'm'));
			row(0, 0, W - 1, (x) => (x === 0 ? 'x' : x >= W - 2 ? 'M' : 'm'));
			row(1, 0, W - 1, () => 'M');
			out.push([3, 2, 'M'], [3, 3, 'M'], [0, 2, 'M'], [0, 3, 'M']);
			break;
		case 'greathelm':
			row(-1, 1, W - 2, (x) => (x === 1 ? 'x' : 'm'));
			for (let y = 0; y < H - 1; y++) row(y, 0, W - 1, (x) => (y === 3 && x >= 1 ? 'z' : x === 0 && y < 3 ? 'x' : x === W - 1 ? 'M' : 'm'));
			out.push([W - 2, 4, 'z'], [2, -2, 'a'], [3, -2, 'a'], [3, -3, 'a']);
			break;
		case 'hood':
			row(-1, 0, W - 2, (x) => (x < 2 ? 'I' : 'C'));
			row(0, -1, W, (x) => (x < 1 ? 'I' : x === W ? 'V' : 'C'));
			row(1, -1, W, (x) => (x === W ? 'V' : x < 1 ? 'I' : 'C'));
			out.push([2, 2, 'V'], [3, 2, 'V'], [4, 2, 'V']);
			for (let y = 2; y < H; y++) out.push([-1, y, 'C'], [0, y, 'C'], [W, y, 'V']);
			row(H, -1, 1, () => 'V');
			out.push([-1, -1, 'V']);
			break;
		case 'cap':
			row(-1, 0, W - 2, (x) => (x === 0 ? 'I' : 'C'));
			row(0, 0, W, (x) => (x === W ? 'V' : 'C'));
			out.push([0, 1, 'y']);
			break;
		case 'hat':
			row(0, -1, W, () => 'V');
			row(-1, 1, W - 2, () => (k.fine ? 'y' : 'A'));
			row(-2, 1, W - 2, (x) => (x === 1 ? 'I' : 'C'));
			break;
		case 'pointy':
			row(0, -1, W, () => 'V');
			row(-1, 0, W - 1, (x) => (x === 0 ? 'I' : 'C'));
			row(-2, 1, W - 2, (x) => (x === 1 ? 'I' : 'C'));
			out.push([2, -3, 'C'], [3, -3, 'C'], [3, -4, 'C'], [4, -4, 'V'], [k.fine ? 2 : 3, -1, k.fine ? 'y' : 'C']);
			break;
		case 'crown':
			row(-1, 0, W - 1, (x) => (x === Math.floor(W / 2) ? 'r' : x === W - 1 ? 'Y' : 'y'));
			out.push([0, -2, 'y'], [Math.floor(W / 2), -2, 'y'], [W - 1, -2, 'Y']);
			break;
		case 'circlet':
			row(1, 0, W - 1, (x) => (x === 3 ? 'c' : x === W - 1 ? 'Y' : 'y'));
			break;
		case 'turban':
			row(-2, 1, W - 2, () => 'C');
			row(-1, 0, W - 1, (x) => (x % 2 ? 'V' : 'C'));
			row(0, 0, W - 1, (x) => (x % 2 ? 'C' : 'V'));
			row(1, 0, W - 1, () => 'C');
			out.push([3, 0, 'y']);
			break;
		case 'goggles':
			row(1, 0, W - 1, () => 'W');
			out.push([2, 1, 'c'], [3, 1, 'M'], [4, 1, 'c']);
			break;
		case 'mask':
			break;
		case 'bandana':
			row(0, 0, W - 1, (x) => (x >= W - 2 ? 'A' : 'a'));
			row(1, 0, W - 1, () => 'A');
			out.push([-1, 1, 'a'], [-2, 2, 'A']);
			break;
		case 'mitre':
			row(0, 0, W - 1, (x) => (x === Math.floor(W / 2) ? 'y' : 'C'));
			row(-1, 0, W - 1, () => 'y');
			row(-2, 1, W - 2, (x) => (x === 1 ? 'I' : 'C'));
			row(-3, 2, W - 3, () => 'C');
			break;
		case 'feathered':
			row(0, -1, W, () => 'V');
			row(-1, 0, W - 1, (x) => (x === 0 ? 'I' : 'C'));
			out.push([0, -2, 'r'], [-1, -3, 'y'], [1, -2, 'r']);
			break;
		case 'widebrim':
			row(0, -2, W + 1, (x) => (x < 0 ? 'C' : 'V'));
			row(-1, 1, W - 2, () => (k.fine ? 'y' : 'V'));
			row(-2, 1, W - 2, (x) => (x === 1 ? 'I' : 'C'));
			row(-3, 2, W - 3, () => 'C');
			break;
		case 'cowl':
			row(-1, 0, W - 1, (x) => (x === 0 ? 'I' : 'C'));
			row(0, -1, W, () => 'C');
			row(1, -1, W, () => 'V');
			for (let y = 2; y < H; y++) out.push([-1, y, 'C'], [0, y, 'V'], [W, y, 'V']);
			break;
		case 'wreath':
			row(0, 0, W - 1, (x) => (x % 2 === 0 ? 'e' : 'E'));
			out.push([0, -1, 'v'], [W - 1, -1, 'e'], [Math.floor(W / 2), -1, 'y']);
			break;
	}
	if (k.mask || k.head === 'mask') {
		row(4, 1, W - 1, (x) => (x === W - 1 ? 'V' : 'C'));
		row(5, 1, W - 1, () => 'V');
	}
	return out;
}

/** recolour slots, lay the fx layer on top, return rows */
function finish(g: Grid, fx: Grid, map: Record<string, string>): string[] {
	return g.map((r, y) => r.map((c, x) => (fx[y][x] !== '.' ? fx[y][x] : map[c] ?? c)).join(''));
}

// ---------------------------------------------------------------- sprite sheet samples

/** the sprite sheet (/sheet.html) draws each of these: { label, actor, poses } */
export interface PersonSample { label: string; actor: QudActor; pose?: Partial<Pose> }
let sid = 100;
const A = (o: Partial<QudActor>): QudActor => ({ x: 0, y: 0, i: sid++, ...o });
const walk = (phase: 0 | 1 | 2 | 3): Partial<Pose> => ({ walk: 1, phase });
export const SAMPLE_PEOPLE: PersonSample[] = [
	// the player, one per calling
	{ label: 'warrior', actor: A({ s: 'human', p: true, c: 'warrior' }) },
	{ label: 'scout', actor: A({ s: 'elf', p: true, c: 'scout' }) },
	{ label: 'scholar', actor: A({ s: 'human', p: true, c: 'scholar' }) },
	{ label: 'tinker', actor: A({ s: 'dwarf', p: true, c: 'tinker' }) },
	{ label: 'mystic', actor: A({ s: 'void_elf', p: true, c: 'mystic' }) },
	{ label: 'merchant', actor: A({ s: 'halfling', p: true, c: 'merchant' }) },
	{ label: 'rogue', actor: A({ s: 'human', p: true, c: 'rogue' }) },
	// walk cycle
	{ label: 'walk 0', actor: A({ s: 'human', p: true, c: 'warrior', i: 1 }), pose: walk(0) },
	{ label: 'walk 1', actor: A({ s: 'human', p: true, c: 'warrior', i: 1 }), pose: walk(1) },
	{ label: 'walk 2', actor: A({ s: 'human', p: true, c: 'warrior', i: 1 }), pose: walk(2) },
	{ label: 'walk 3', actor: A({ s: 'human', p: true, c: 'warrior', i: 1 }), pose: walk(3) },
	// combat poses
	{ label: 'wind-up', actor: A({ s: 'human', p: true, c: 'warrior', i: 1 }), pose: { attack: 0.2 } },
	{ label: 'strike', actor: A({ s: 'human', p: true, c: 'warrior', i: 1 }), pose: { attack: 0.7 } },
	{ label: 'block', actor: A({ s: 'human', p: true, c: 'warrior', i: 1 }), pose: { block: 0.5 } },
	{ label: 'hurt', actor: A({ s: 'human', p: true, c: 'warrior', i: 1 }), pose: { hurt: 0.4, dir: -1 } },
	{ label: 'cast', actor: A({ s: 'void_elf', p: true, c: 'mystic', i: 2 }), pose: { cast: 0.5 } },
	{ label: 'face left', actor: A({ s: 'elf', p: true, c: 'scout', i: 3 }), pose: { face: -1 } },
	{ label: 'bow shot', actor: A({ s: 'elf', p: true, c: 'scout', i: 3 }), pose: { attack: 0.3 } },
	{ label: 'asleep', actor: A({ s: 'human', r: 'villager' }), pose: { sleep: 1 } },
	{ label: 'dead', actor: A({ s: 'orc', r: 'bandit' }), pose: { dead: 1, dir: 1 } },
	// roles
	{ label: 'guard', actor: A({ s: 'human', r: 'guard' }) },
	{ label: 'guard thrust', actor: A({ s: 'human', r: 'guard' }), pose: { attack: 0.7 } },
	{ label: 'soldier', actor: A({ s: 'orc', r: 'soldier' }) },
	{ label: 'knight', actor: A({ s: 'human', r: 'knight' }) },
	{ label: 'merchant', actor: A({ s: 'human', r: 'merchant' }) },
	{ label: 'priest', actor: A({ s: 'human', r: 'priest' }) },
	{ label: 'noble', actor: A({ s: 'elf', r: 'noble' }) },
	{ label: 'smith', actor: A({ s: 'dwarf', r: 'smith' }), pose: { work: 0.3, workKind: 'hammer' } },
	{ label: 'beggar', actor: A({ s: 'human', r: 'beggar' }) },
	{ label: 'mage', actor: A({ s: 'human', r: 'mage' }), pose: { cast: 0.6 } },
	{ label: 'thief', actor: A({ s: 'ratfolk', r: 'thief' }) },
	{ label: 'elder', actor: A({ s: 'human', r: 'elder' }) },
	{ label: 'farmer', actor: A({ s: 'halfling', r: 'farmer' }) },
	{ label: 'sailor', actor: A({ s: 'merfolk', r: 'sailor' }) },
	{ label: 'cook', actor: A({ s: 'giant', r: 'cook' }) },
	{ label: 'apothecary', actor: A({ s: 'goblin', r: 'apothecary' }) },
	{ label: 'nomad', actor: A({ s: 'lizardfolk', r: 'nomad' }) },
	{ label: 'inquisitor', actor: A({ s: 'human', r: 'inquisitor' }) },
	{ label: 'pilgrim', actor: A({ s: 'human', r: 'pilgrim' }) },
	{ label: 'chronicler', actor: A({ s: 'elf', r: 'chronicler' }) },
	{ label: 'librarian', actor: A({ s: 'halfling', r: 'librarian' }) },
	{ label: 'bard', actor: A({ s: 'human', r: 'bard' }) },
	{ label: 'alchemist', actor: A({ s: 'dwarf', r: 'alchemist' }) },
	{ label: 'fence', actor: A({ s: 'ratfolk', r: 'fence' }) },
	{ label: 'assassin', actor: A({ s: 'void_elf', r: 'assassin' }) },
	{ label: 'artificer', actor: A({ s: 'dwarf', r: 'artificer' }) },
	{ label: 'druid', actor: A({ s: 'elf', r: 'druid' }) },
	{ label: 'herald', actor: A({ s: 'human', r: 'herald' }) },
	{ label: 'executioner', actor: A({ s: 'orc', r: 'executioner' }) },
	{ label: 'duellist', actor: A({ s: 'human', r: 'duellist' }) },
	// species and armour tiers / weapons
	{ label: 'goblin dagger', actor: A({ s: 'goblin', r: 'bandit', w: 'shortblade' }) },
	{ label: 'ogor axe', actor: A({ s: 'ogor', w: 'axe', a: 1 }) },
	{ label: 'troll mace', actor: A({ s: 'troll', w: 'cudgel', a: 0 }), pose: { attack: 0.2 } },
	{ label: 'beastman spear', actor: A({ s: 'beastman', w: 'spear', a: 1 }) },
	{ label: 'saurian plate', actor: A({ s: 'saurian', r: 'soldier', a: 3 }) },
	{ label: 'celestial', actor: A({ s: 'celestial', r: 'priest' }) },
	{ label: 'undead', actor: A({ s: 'tomb_undead', w: 'longblade' }) },
	{ label: 'warp fiend', actor: A({ s: 'warp_fiend', w: 'whip' }) },
	{ label: 'gunner', actor: A({ s: 'dwarf', r: 'soldier', w: 'firearm', a: 2 }), pose: { attack: 0.7 } },
	{ label: 'swarm', actor: A({ s: 'chitin_swarm' }) },
	{ label: 'tribe hex', actor: A({ s: 'human', r: 'villager', tr: 'tr_hex:#c08060:#2a7a8a' }) },
];
