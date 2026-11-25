// Natural terrain pixel art (12 x 20, see types.ts).
// Notes for the coordinator:
// - Ground is sparse on purpose (Qud style): textured specks and decals over the cell background, so trees,
//   rocks and props that leave their surroundings transparent sit on the same ground and never punch holes.
// - Remembered (out of sight) cells come as fg 1 / bg 20 for every glyph; fixed colours are then mapped to the
//   fg ramp (mono) so memory stays grey.
// - Animated here: water, lava, tar, hot springs, waterfalls, spray, tall grass, reeds, kelp, trees, geysers,
//   fumaroles, fire, updrafts, crystals, coral. Only the kinds tilevariety.isAnimatedTile lists get repainted
//   each frame; spray, geyser, fumarole, coral, crystal and hot spring need adding there to move.
// - '#' living rock is left to the build artist: remembered walls also arrive as fg 1 / bg 20, so rock cannot be
//   told apart from a remembered wall by colour.
import { type Rows, PW, PH, mk, over, flipX, blank } from './types';

// ---- helpers ------------------------------------------------------------------------------------

/** decal rows (any width) */
const dc = (...r: string[]): string[] => r;
/** bottom-aligned sprite: rows fill the lowest lines */
const bot = (...r: string[]): Rows => mk(...Array(Math.max(0, PH - r.length)).fill(''), ...r);

function recolour(rows: Rows, map: Record<string, string>): string[] {
	return rows.map((r) => r.replace(/./g, (c) => map[c] ?? c));
}
const MONO: Record<string, string> = {
	e: 'F', E: 'd', u: 'F', U: 'd', c: 'l', w: 'F', W: 'd', t: 'l', T: 'd', g: 'F', G: 'd', m: 'l', M: 'd',
	r: 'F', R: 'd', y: 'l', Y: 'd', x: 'L', z: 'o', p: 'F', n: 'l', q: 'l', v: 'l'
};
const mono = (r: Rows): string[] => recolour(r, MONO);

/** shift rows [y0, y1) by dx (no wrap) */
function bend(rows: Rows, dx: number, y0: number, y1: number): string[] {
	if (!dx) return rows.slice();
	return rows.map((r, y) => {
		if (y < y0 || y >= y1) return r;
		return dx > 0 ? ('.'.repeat(dx) + r).slice(0, PW) : (r + '.'.repeat(-dx)).slice(-dx, PW - dx);
	});
}
/** rotate rows [y0, y1) right by dx, wrapping */
function wrapX(rows: Rows, dx: number, y0 = 0, y1 = PH): string[] {
	const k = ((dx % PW) + PW) % PW;
	return rows.map((r, y) => (y < y0 || y >= y1 || !k ? r : r.slice(PW - k) + r.slice(0, PW - k)));
}
/** rotate rows down by dy, wrapping */
function wrapY(rows: Rows, dy: number): string[] {
	const k = ((dy % PH) + PH) % PH;
	return rows.map((_, y) => rows[(y - k + PH) % PH]);
}
function put(rows: string[], x: number, y: number, c: string): void {
	if (x < 0 || y < 0 || x >= PW || y >= PH) return;
	rows[y] = rows[y].slice(0, x) + c + rows[y].slice(x + 1);
}
const pick = <T,>(a: readonly T[], h: number, s = 0): T => a[(h >>> s) % a.length];

const SLOTS: [number, number][] = [[1, 1], [6, 3], [2, 7], [7, 9], [1, 13], [6, 15], [4, 11], [8, 5]];
/** n decals from `decs` at distinct slots chosen by h */
function scatter(base: Rows, decs: readonly string[][], h: number, n: number): string[] {
	let r = base.slice();
	const s0 = h % SLOTS.length;
	for (let i = 0; i < n; i++) {
		const d = decs[((h >>> (i * 3 + 5)) + i) % decs.length];
		const [sx, sy] = SLOTS[(s0 + i * 3) % SLOTS.length];
		r = over(r, d, Math.min(sx, PW - d[0].length), Math.min(sy, PH - d.length));
	}
	return r;
}

// ---- ground ------------------------------------------------------------------------------------

const SP: Rows[] = [
	mk(
		'............',
		'.D......d...',
		'......D.....',
		'...........D',
		'..d.........',
		'.........D..',
		'D...........',
		'......d.....',
		'...D......D.',
		'............',
		'........D...',
		'..D.........',
		'......d...D.',
		'.D..........',
		'..........d.',
		'....D.......',
		'..........D.',
		'.d....D.....',
		'..........d.',
		'...D........'
	),
	mk(
		'......d.....',
		'..D.........',
		'.........D..',
		'....d.......',
		'D...........',
		'..........d.',
		'...D....D...',
		'............',
		'.d..........',
		'......D...D.',
		'..D.........',
		'.........d..',
		'....D.......',
		'D.......D...',
		'............',
		'...d......D.',
		'........D...',
		'.D..........',
		'......d.....',
		'..........D.'
	),
	mk(
		'...D........',
		'..........D.',
		'.d....D.....',
		'............',
		'.........d..',
		'..D.........',
		'......D.....',
		'D.........D.',
		'....d.......',
		'..........d.',
		'.D......D...',
		'............',
		'...D.......D',
		'.......d....',
		'..d.........',
		'.........D..',
		'D....D......',
		'............',
		'...d.....D..',
		'........d...'
	)
];
/** a speckle base picked (and maybe mirrored) by h, recoloured */
function specks(h: number, map?: Record<string, string>, s = 0): string[] {
	let b: Rows = SP[(h >>> s) % 3];
	if ((h >>> (s + 2)) & 1) b = flipX(b);
	return map ? recolour(b, map) : b.slice();
}

const TUFT_A = dc('.l..F', '..lF.', '.dFd.');
const TUFT_B = dc('l.F', 'lFd', 'dFd');
const TUFT_C = dc('..l...', 'l.l.F.', '.lFlF.', '.dFFd.');
const SPROUT = dc('.l.', 'dFd');
const FERN = dc('..l.', 'lFl.', '.eE.', '..E.');
const flowerDec = (c: string, mid: string) => dc('.' + c + '.', c + mid + c, '.e.');
const flowerClump = (c: string, c2: string) => dc(c + '..' + c2, '.e.e', 'e.' + c + '.');
const PEB_A = dc('mg.', 'gGo');
const PEB_B = dc('.mg.', 'mggG', 'oGGo');
const PEB_C = dc('m...', 'G.g.', '...G');
const CLOD = dc('lF.', 'FdD');
const TWIG = dc('w...', '.Ww.', '...W');
const CRACK = dc('D....', '.D.DD', '..D..');
const CRACK2 = dc('..D.', '.D..', '.DD.', 'D..D');
const RIP_A = dc('..lLl...', '.l...lF.', 'd.....dd');
const RIP_B = dc('.lLlF', 'd...d');
const SHELL = dc('xq', 'qo');
const DRIFT = dc('...lLLl...', '.lLFFFFLl.', 'dFFFFFFFFd', '.dddddddd.');
const DRIFT_S = dc('.lLl.', 'lFFFl', 'dddd.');
const SALTC = dc('dd...', '..ddD', '....d');
const SALTC2 = dc('d...', '.dL.', '.Dd.', '...d');
const MOSS_A = dc('..lFl..', '.lFFFd.', 'lFdFFdD', '.dFdD..');
const MOSS_B = dc('lFl.', 'FdFd', '.dD.');
const CAP_A = dc('.lL.', 'lFFd', '.tT.');
const CAP_B = dc('lL', 'tT');
const CAP_C = dc('.rR.', 'rxRr', '.tt.');
const SPORE = dc('L..', '..l', '.L.');
const SHARD = dc('L', 'F', 'd');
const GLINT = dc('.x.', 'lLd');
const GLINT_B = dc('.x.', 'xcx', '.c.');
const EMBER = dc('n.', '.R');
const FLOWERS = ['q', 'y', 'x', 'p', 'n', 'c', 'r'];

function grassGround(h: number): string[] {
	let r = scatter(specks(h), [TUFT_A, TUFT_B, TUFT_C, SPROUT, FERN], h, 2 + ((h >>> 20) % 3));
	const fRoll = (h >>> 24) % 6;
	if (fRoll === 0) {
		const c = pick(FLOWERS, h, 27);
		r = over(r, flowerDec(c, c === 'y' ? 'n' : 'y'), 1 + ((h >>> 9) % 8), 4 + ((h >>> 13) % 12));
	} else if (fRoll === 1) {
		const c1 = pick(FLOWERS, h, 27);
		const c2 = pick(FLOWERS, h, 19);
		r = over(r, flowerClump(c1, c2), 1 + ((h >>> 11) % 7), 3 + ((h >>> 15) % 13));
	}
	if ((h >>> 28) % 5 === 0) r = over(r, PEB_A, 2 + ((h >>> 3) % 7), 2 + ((h >>> 17) % 15));
	return r;
}
const earthGround = (h: number) => scatter(specks(h), [PEB_A, PEB_B, CLOD, TWIG, CRACK, PEB_C], h, 1 + ((h >>> 20) % 3));
const floorGround = (h: number) => scatter(specks(h), [PEB_A, PEB_C, CRACK, CLOD, CRACK2], h, 1 + ((h >>> 20) % 2));
function ashGround(h: number): string[] {
	let r = scatter(specks(h, { d: 'l' }), [CRACK, PEB_C, CRACK2], h, 1 + ((h >>> 20) % 2));
	if ((h >>> 23) % 3 === 0) r = over(r, EMBER, 2 + ((h >>> 5) % 8), 3 + ((h >>> 9) % 15));
	return r;
}
function sandGround(h: number): string[] {
	const decs = (h >>> 26) % 9 === 0 ? [RIP_A, RIP_B, SHELL] : [RIP_A, RIP_B, PEB_A];
	return scatter(specks(h, { D: 'd', d: 'F' }), decs, h, 1 + ((h >>> 20) % 3));
}
function snowGround(h: number): string[] {
	const a = specks(h, { D: 'l', d: 'd' });
	const b = specks(h, { D: 'F', d: 'L' }, 7);
	return scatter(over(a, wrapY(b, 7)), [DRIFT, DRIFT_S, DRIFT_S], h, 1 + ((h >>> 21) % 2));
}
function saltGround(h: number): string[] {
	const a = specks(h, { D: 'F', d: 'l' });
	const b = specks(h, { D: 'L', d: 'F' }, 5);
	return scatter(over(a, wrapY(b, 10)), [SALTC, SALTC2, CRACK2.map((s) => s.replace(/D/g, 'd'))], h, 2 + ((h >>> 20) % 2));
}
const roadGround = (h: number) =>
	scatter(over(specks(h, { D: 'd', d: 'D' }), specks(h, { D: 'F', d: 'l' }, 4)), [PEB_A, PEB_C, CLOD, PEB_B], h, 2 + ((h >>> 20) % 2));
function mossGround(h: number): string[] {
	let r = scatter(specks(h), [MOSS_A, MOSS_B, MOSS_B, SPROUT], h, 2 + ((h >>> 20) % 2));
	if ((h >>> 25) % 5 === 0) r = over(r, CAP_C, 1 + ((h >>> 7) % 7), 4 + ((h >>> 11) % 12));
	return r;
}
const fungusGround = (h: number) => scatter(specks(h), [CAP_A, CAP_B, CAP_C, SPORE, CAP_B], h, 2 + ((h >>> 20) % 3));
const crystalSand = (h: number) => scatter(specks(h, { D: 'd', d: 'l' }), [SHARD, GLINT, GLINT_B, SHARD, SPORE], h, 2 + ((h >>> 20) % 3));

const BONES = bot(
	'...oLLo.....',
	'..lLLLLd....',
	'..LzLLzd....',
	'..dLLLLd..lL',
	'...lddd..lLd',
	'.Ll.....lLd.',
	'.dLLLLLlLd..',
	'.Ld.oo.Ld...'
);

const QSWIRL = dc('..dddd..', '.dlFFld.', 'dlF..Fld', '.dlFFld.', '..dddd..');
function quicksand(h: number, f: number): string[] {
	const r = over(specks(h, { D: 'd', d: 'F' }), QSWIRL, 2 + ((h >>> 4) % 3), 6 + ((h >>> 8) % 6));
	const t = (f + h) & 7;
	if (t < 4) put(r, 5 + ((h >>> 4) % 3), 8 + ((h >>> 8) % 6) - (t >> 1), t < 3 ? 'o' : 'L');
	return r;
}

// ---- water, lava, tar ------------------------------------------------------------------------------

const WAVES = mk(
	'............',
	'..lLl.......',
	'.F...F......',
	'............',
	'.......lLl..',
	'......F...F.',
	'............',
	'...lL.......',
	'..F..F......',
	'............',
	'........lLl.',
	'.......F...F',
	'............',
	'............',
	'.lLl........',
	'F...F.......',
	'............',
	'......lLl...',
	'.....F...F..',
	'............'
);
const STEAM = mk(
	'....l.......',
	'...lL.......',
	'....L....l..',
	'.....l..lL..',
	'.........L..',
	'..l.......l.',
	'.lL.........',
	'..L.....l...',
	'...l...lL...',
	'........L...',
	'.....l...l..',
	'....lL......',
	'.....L......',
	'..........l.',
	'.l.......lL.',
	'.lL.......L.',
	'..L...l.....',
	'......lL....',
	'.......L....',
	'............'
);

const N = 1, E = 2, S = 4, W = 8;

/** foam on the sides that are not water */
function shore(r: string[], nm: number, f: number): void {
	const ph = (f >> 1) & 1;
	for (let i = 0; i < PW; i++) {
		const a = (i + ph) % 3 !== 0 ? 'L' : 'x';
		const b = (i + ph) % 3 === 0 ? 'l' : '.';
		if (!(nm & N)) { put(r, i, 0, a); if (b !== '.') put(r, i, 1, b); }
		if (!(nm & S)) { put(r, i, PH - 1, a === 'x' ? 'l' : 'L'); if (b !== '.') put(r, i, PH - 2, b); }
	}
	for (let y = 0; y < PH; y++) {
		const a = (y + ph) % 3 !== 0 ? 'L' : 'x';
		const b = (y + ph) % 4 === 0 ? 'l' : '.';
		if (!(nm & W)) { put(r, 0, y, a); if (b !== '.') put(r, 1, y, b); }
		if (!(nm & E)) { put(r, PW - 1, y, a === 'x' ? 'l' : 'L'); if (b !== '.') put(r, PW - 2, y, b); }
	}
}

function water(deep: boolean, h: number, frame: number, nm: number, steam: boolean, still: boolean): string[] {
	const f = still ? 0 : frame & 7;
	const r = blank();
	for (let y = 0; y < PH; y++)
		for (let x = 0; x < PW; x++) {
			if (deep ? (x + y * 2) % 4 === 0 : (x + y * 3) % 6 === 0) put(r, x, y, deep ? 'D' : 'd');
			else if (deep && (x + y * 3) % 7 === 3) put(r, x, y, 'd');
		}
	const off = (f * 3) >> 1;
	let wv = wrapX(WAVES, off, 0, 7);
	wv = wrapX(wv, -off, 7, 14);
	wv = wrapX(wv, off, 14, PH);
	if (deep) wv = recolour(wv, { l: 'F', L: 'l', F: 'd' });
	let out = over(r, wv);
	if (steam) out = over(out, wrapY(STEAM, -((f * 5) >> 1)));
	if (!still && ((f + (h >>> 9)) & 7) < 2) {
		const gx = h % 11, gy = 1 + ((h >>> 4) % 17);
		put(out, gx, gy, 'x');
		put(out, gx + 1, gy, 'L');
	}
	if (!still) shore(out, nm, f);
	return out;
}

const LAVA = mk(
	'FlLlFdDoDdFl',
	'lLyLlFdoodFF',
	'FlLlFdDooDdF',
	'dFlFdDoooDdl',
	'DdFdDoooDdFl',
	'oDddDooDdFlL',
	'ooDDooDdFlLy',
	'DooooDdFlLlF',
	'dDooDdFlLlFd',
	'FdDDdFlLlFdD',
	'lFddFlLlFdDo',
	'LlFFlLyLFdDo',
	'lLlFFlLlFdoo',
	'FlFdDdFlFdDo',
	'dFdDooDdFFdD',
	'DdDoooooDdFd',
	'oDoooooooDdF',
	'oDDoooooDdFl',
	'dFdDooDDdFlL',
	'FlFdDDdFlLlF'
);

function lava(h: number, frame: number, nm: number, tar: boolean): string[] {
	let r: string[] = (h >>> 3) & 1 ? wrapY(flipX(LAVA), 7) : LAVA.slice();
	r = wrapX(r, (h >>> 5) % 12);
	const f = (frame + ((h >>> 11) & 1) * 2) & 7;
	if (f >= 4) r = recolour(r, { F: 'l', l: 'L' });
	const t = (frame + h) & 7;
	const bx = 2 + ((h >>> 13) % 8), by = 3 + ((h >>> 17) % 14);
	if (t === 1) put(r, bx, by, 'l');
	if (t === 2) put(r, bx, by, 'L');
	if (t === 3 || t === 4) { put(r, bx, by, 'L'); put(r, bx + 1, by, 'y'); put(r, bx, by - 1, 'y'); }
	if (t === 5) { put(r, bx - 1, by - 1, 'x'); put(r, bx + 1, by - 1, 'x'); }
	// cooled crust where the pool meets the bank
	for (let i = 0; i < PW; i++) {
		if (!(nm & N)) { put(r, i, 0, 'o'); put(r, i, 1, i % 2 ? 'D' : 'd'); }
		if (!(nm & S)) { put(r, i, PH - 1, 'o'); put(r, i, PH - 2, i % 2 ? 'd' : 'D'); }
	}
	for (let y = 0; y < PH; y++) {
		if (!(nm & W)) { put(r, 0, y, 'o'); put(r, 1, y, y % 2 ? 'D' : 'd'); }
		if (!(nm & E)) { put(r, PW - 1, y, 'o'); put(r, PW - 2, y, y % 2 ? 'd' : 'D'); }
	}
	if (tar) r = recolour(r, { F: 'D', l: 'd', L: 'l', y: 'L', x: 'l', d: 'o', D: 'o' });
	return r;
}

// ---- falls, spray, ice, cloud, glacier --------------------------------------------------------------

const FALL_COLS = [
	'UUuuUUuUUUuuUUUuUUuU',
	'uxuucuuxcuuucuxuucuu',
	'cxxcuucxxxcuucxxcuuc',
	'uuxccuuuxcuuuxccuuuu',
	'xcuuxxcuuuxxcuuuxcuu',
	'cuuxcuuuxxcuuuxcuuxc',
	'uxcuuuxccuuxcuuuxcuu',
	'uucxxcuuuucxxcuuuxxc',
	'xcuuuxcuucuuxcuuucxu',
	'uuxcuuuxcuuuxcuuxcuu',
	'UuuUuuUuUUuuUuUuuUUu',
	'UUoUUUoUUUUoUUUUoUUU'
];
const FALLS: Rows = Array.from({ length: PH }, (_, y) => FALL_COLS.map((c) => c[y]).join(''));

const DROPS = mk(
	'c.....x.....',
	'....c.....c.',
	'..x.........',
	'.......c....',
	'c........x..',
	'....x.......',
	'..........c.',
	'.c....c.....',
	'........x...',
	'...c........',
	'x.........c.',
	'.....c......',
	'..c.....x...',
	'..........c.',
	'....x.......',
	'c......c....',
	'.........x..',
	'...c........',
	'......x...c.',
	'.c..........'
);

const ICE_STREAK = dc('..L', '.L.', 'L..');
const ICE_CRACK = dc('D...', '.DD.', '...D', '..D.');
const iceGround = (h: number) => scatter(specks(h, { D: 'd', d: 'F' }), [ICE_STREAK, ICE_CRACK, GLINT, ICE_STREAK], h, 2 + ((h >>> 20) % 2));

const CLOUD = dc('..lLLl.....', '.lLLLLLl...', 'lLLFFLLLLl.', 'dFFFFFFFFd.', '.dddddddd..');
const CLOUD_S = dc('.lLl..', 'lLFLLl', 'dFFFFd', '.dddd.');
const cloudGround = (h: number) => scatter(specks(h, { D: 'l', d: 'd' }), [CLOUD, CLOUD_S, CLOUD_S], h, 2);

const GLACIER = mk(
	'..lLLl...cc.',
	'.lLFFcd.lLLc',
	'lLFccd..lFcd',
	'dFccdd.dFcd.',
	'.dddd...ddd.',
	'............',
	'....cLLl....',
	'..cLLFFcl...',
	'.cLFFccFcd..',
	'.dFcccccdd..',
	'..ddddddd...',
	'...........c',
	'lLl.........',
	'FFcd....lLLl',
	'cccd...lFFcc',
	'dddd..dFccdd',
	'......dddd..',
	'..lLl.......',
	'.lFFcd...cL.',
	'.dddd....dd.'
);

// ---- grasses, reeds, kelp, crops ------------------------------------------------------------------

const TGRASS: Rows[] = [
	mk(
		'............',
		'............',
		'............',
		'....L.......',
		'....l...L...',
		'...lF...l...',
		'.L.lF..lF...',
		'.l.lF..lF.L.',
		'.lF.lF.lFdl.',
		'..lFlFlF.dF.',
		'..lFdlFd.lF.',
		'.l.FdlFdlFd.',
		'.lFdFlFdlFd.',
		'..lFdFFdFd..',
		'..FdFdFFdFd.',
		'.dFdFdFdFdd.',
		'.DdFdDdFdDd.',
		'..DdDdDdDD..',
		'...oDoDDo...',
		'............'
	),
	mk(
		'............',
		'............',
		'..L......L..',
		'..l......l..',
		'..lF..L..lF.',
		'...lF.l..lF.',
		'.L.lF.lF.lF.',
		'.l.lFdlF.lFd',
		'.lF.lFlFlFd.',
		'..lFlFdlFd..',
		'..lFdlFdlF.L',
		'L..lFlFdlFdl',
		'lF.lFdlFlFdF',
		'.lFdlFdFlFd.',
		'..lFdFFdFd..',
		'.dlFdFdFFdd.',
		'.dFdFdFdFdd.',
		'.DdFdDdFdDd.',
		'..DdDdDdDD..',
		'...o.o.o.o..'
	)
];
const SWAY = [0, 1, 1, 0, 0, -1, -1, 0];

function tallGrass(h: number, f: number): string[] {
	let r = TGRASS[h & 1].slice();
	if ((h >>> 5) % 5 === 0) {
		const c = pick(FLOWERS, h, 9);
		put(r, 4, 3, c);
		put(r, 9, 4, c);
	}
	const s = SWAY[(f + h) & 7];
	r = bend(r, s, 0, 8);
	return r;
}

const REEDS = mk(
	'........l...',
	'...l....F...',
	'..wW....F..l',
	'..wW...wW..F',
	'..wW...wW..F',
	'..WW..lwW.wW',
	'...F..FWW.wW',
	'.l.F..F.F.wW',
	'.F.F.lF.F.WW',
	'.FlF.F..FlF.',
	'.lFd.F.lFdF.',
	'..FdlF.FdF..',
	'..lFFd.FdF..',
	'.l.FdFlFd.l.',
	'.FlFdFFd..F.',
	'..lFdFFdlFd.',
	'.lFdFdFdFd..',
	'.dDdDdDdDdD.',
	'..u..c...u..',
	'.D.o.D.o.D..'
);

const KELP = mk(
	'...l......l.',
	'..lF.....lF.',
	'..lFd...lFd.',
	'...lF...lF..',
	'...lFd.lFd..',
	'..lFd..lF...',
	'..lFd...lF..',
	'.lFd.l..lFd.',
	'.lF..lF..lF.',
	'..lF.lFd.lF.',
	'..lFdlF..lFd',
	'...lFlFd.lF.',
	'..lFd.lF.lF.',
	'..lF..lFdlF.',
	'...lF.lF.lF.',
	'..lFd.lFlFd.',
	'..lF..lFlF..',
	'..lFd.lFlF..',
	'.olFdolFlFo.',
	'.oDdDoDdDDo.'
);

const CROPS = mk(
	'............',
	'............',
	'..L....L....',
	'.lFd..lFd..l',
	'.lFd..lFd..l',
	'.dFd..dFd..d',
	'..d.L..d.L..',
	'..dlFd.dlFd.',
	'..dlFd.dlFd.',
	'..ddFd.ddFd.',
	'.L.d.d.Ld.d.',
	'lFdd.dlFdd.d',
	'lFd..dlFd..d',
	'dFd..ddFd..d',
	'.d.d..dd.d..',
	'.d.d..d..d..',
	'.dEd.Ed.EdE.',
	'.eEe.eEeeEe.',
	'WwWWwWWwWWwW',
	'.W.W.W.W.W.W'
);

// ---- trees ----------------------------------------------------------------------------------------

const OAKS: Rows[] = [
	mk(
		'....oddo....',
		'..odlLLldo..',
		'.odlLLFlFdo.',
		'.dlLlFFlFFd.',
		'odlLFFlFFdFo',
		'dlLFFdFFlFdo',
		'dlFFdFFlLFdo',
		'dFlFFdFlFFdo',
		'odFFdFFFdFDo',
		'.dFdFdDdFdDo',
		'.oDdDdwDdDo.',
		'..oDDowWDoo.',
		'....owWo....',
		'....owWWo...',
		'....owWWo...',
		'....wwWWo...',
		'...owwWWo...',
		'..owwWWWWo..',
		'..ooWo.oWoo.',
		'...o.o.o.o..'
	),
	mk(
		'.....od.....',
		'....dlLdo...',
		'...dlLFdo...',
		'...lLFlFdo..',
		'..dlFFlFFd..',
		'..lLFdFlFdo.',
		'..lFFlFFdFo.',
		'.dlFdFFdFFdo',
		'.dFFlFdFlFdo',
		'.dFdFFdFFdDo',
		'..dFdFDdFDo.',
		'..oDdDwDDo..',
		'...ooWwoo...',
		'.....wWo....',
		'.....wWo....',
		'.....wWo....',
		'.....wWo....',
		'....wwWWo...',
		'...owWoWWo..',
		'....o.o.o...'
	),
	mk(
		'............',
		'..odlo.odo..',
		'.dlLLdodlLd.',
		'dlLFFldlLFdo',
		'dLFFlFdLFFdo',
		'lFFdFFlFFdFo',
		'dFlFFdFFlFdo',
		'dFFdFlFdFFDo',
		'.dFFdFFdFdDo',
		'.dDdFdFDdDo.',
		'..oDDwWDDo..',
		'...ooWwoo...',
		'.....wWo....',
		'....wwWo....',
		'....wWWo....',
		'....wWWo....',
		'....wWWo....',
		'...owwWWo...',
		'..oowWoWoo..',
		'...o.o.o.o..'
	)
];
const FRUIT_AT: [number, number][] = [[3, 5], [7, 4], [5, 8], [8, 7], [2, 7]];

const PINES: Rows[] = [
	mk(
		'.....l......',
		'.....lo.....',
		'....lFdo....',
		'....lLdo....',
		'...lLFFdo...',
		'...dDdDDo...',
		'....lFdo....',
		'...lLFFdo...',
		'..lLFFdFdo..',
		'..dDdFdDDo..',
		'...lLFFdo...',
		'..lLFdFFdo..',
		'.lLFFdFFFdo.',
		'.lFFdFFdFDdo',
		'.dDdDdDDdDDo',
		'....owWo....',
		'.....wWo....',
		'.....wWo....',
		'...owwWWo...',
		'....o.o.o...'
	),
	mk(
		'.....lo.....',
		'.....Ld.....',
		'....lLdo....',
		'....dFDo....',
		'....lLdo....',
		'...lLFdo....',
		'...dFdDDo...',
		'...lLFFdo...',
		'..lLFdFdo...',
		'..dDFdDDDo..',
		'..lLFFdFo...',
		'.lLFdFFdDo..',
		'.dDdFdDdDDo.',
		'.lLFFdFFdFo.',
		'lLFdFFdFFdDo',
		'dDdDdDdDDDDo',
		'.....wWo....',
		'.....wWo....',
		'....owWWo...',
		'....o.o.o...'
	)
];

const SNAGS: Rows[] = [
	mk(
		'..l.......l.',
		'..ld....lFo.',
		'...ld..lFo..',
		'.l..Fd.Fo.l.',
		'.ld.lFdFolFo',
		'..lFdlFFdFo.',
		'....lFFdo...',
		'..l.lFdo....',
		'..lFlFdo....',
		'....lFdo....',
		'....lFDo....',
		'....lFdo....',
		'....lFdo....',
		'....lFDo....',
		'....lFdo....',
		'...llFdDo...',
		'..lFdFdDdo..',
		'.ldo.FdoDdo.',
		'.o..oo.o..o.',
		'....o.o.o...'
	),
	mk(
		'........l...',
		'.l.....lFo..',
		'.lF..l.Fo...',
		'..lFdlFFd.l.',
		'...lFFdo.lFo',
		'....lFdolFo.',
		'.l..lFFFdo..',
		'.lFdlFdo....',
		'...lFFdo....',
		'...lFdo.....',
		'...lFdDo....',
		'....lFdo....',
		'....lFdo....',
		'....lFDo....',
		'....lFdo....',
		'....lFdDo...',
		'...llFdDdo..',
		'..lFo.FdoDo.',
		'..o..oo.o.o.',
		'....o.o.o...'
	)
];

const PALM = mk(
	'....lLl.....',
	'..lLFFFl.l..',
	'.lFdo.lFLFl.',
	'lFd..lLFdoFl',
	'Fd..lFWWldoF',
	'd..lFdWwoFd.',
	'...Fd.wWo.Fd',
	'...d..wWo..d',
	'......wWo...',
	'.....wWo....',
	'.....wWo....',
	'.....WWo....',
	'.....wWo....',
	'.....wWo....',
	'.....WWo....',
	'......wWo...',
	'......wWo...',
	'......wWo...',
	'.....owWWo..',
	'.....o.o.o..'
);

const CACTI: Rows[] = [
	mk(
		'............',
		'.....lF.....',
		'....lLFd....',
		'....lLFdo...',
		'....lLFdo...',
		'.lF.lLFdo...',
		'lLFdlLFdo...',
		'lLFdlLFdolF.',
		'lLFdlLFdlLFd',
		'lLFdlLFdlLFd',
		'.lFFlLFdlLFd',
		'..dDlLFdFFdo',
		'....lLFdDDo.',
		'....lLFdo...',
		'....lLFdo...',
		'....lLFdo...',
		'....lLFdo...',
		'...olLFdoo..',
		'..TTdDDDoTT.',
		'...T.o.o.T..'
	),
	bot(
		'...lLl......',
		'..lLFFd.....',
		'..lFFFdlLl..',
		'..dFFdlLFFd.',
		'...dDolFFFd.',
		'.lLl.FdFFdo.',
		'lLFFdoFdDo..',
		'lFFFdoFdo...',
		'dFFdo.Fdo...',
		'.dDo.dFDo...',
		'..TT.ddDoTT.',
		'...T.o.o.T..'
	)
];

const BAMBOO = mk(
	'..l....lF...',
	'.lFd..lFd.l.',
	'..lFd.lF.lFd',
	'..lFd.lFd.lF',
	'..DDd.DDd.lF',
	'..lFd.lFdlFd',
	'.llFd.lFd.DD',
	'lFdFd.lFd.lF',
	'..lFdlDDd.lF',
	'..DDdFlFd.lF',
	'..lFd.lFd.DD',
	'..lFd.lFd.lF',
	'..lFd.lFdlFd',
	'..DDd.lFdFd.',
	'..lFd.DDd.lF',
	'..lFd.lFd.lF',
	'..lFd.lFd.DD',
	'.olFdolFdolF',
	'.oDDdoDDdoDD',
	'..o.o.o.o.o.'
);

// ---- shrubs, fungi, flowers, stumps ------------------------------------------------------------------

const BUSHES: Rows[] = [
	bot(
		'....odlo....',
		'..odlLLldo..',
		'.dlLFlFFFdo.',
		'.lLFFdFlFdo.',
		'dlFlFFFdFFdo',
		'dFFdFlFFdFDo',
		'dFlFFdFFFdDo',
		'.dFFdFdFdDDo',
		'.dDdFDdDDDo.',
		'..oDDoWDDoo.',
		'...oooWooo..',
		'....o.o.o...'
	),
	bot(
		'...odlo.....',
		'..dlLLdo.od.',
		'.dlLFFFdolLd',
		'.lLFdFFdlFFo',
		'dlFFFdFdFFdo',
		'dFFdFFFdFdDo',
		'.dFFdFdDdDo.',
		'..dDdDDDDo..',
		'...oooWooo..',
		'....o.o.o...'
	)
];
const BERRY_AT: [number, number][] = [[3, 12], [7, 11], [5, 14], [9, 13]];

const GIANT_MUSH = mk(
	'............',
	'....oddo....',
	'..odlLLldo..',
	'.dlLLxLFFdo.',
	'.lLxLFFlFFd.',
	'dlLFFFFxFFdo',
	'dLFFlFFFFFdo',
	'dFFFFFdFFFDo',
	'oDdDDDDDDDDo',
	'.oDzDzDzDzo.',
	'....tttT....',
	'....ttTT....',
	'....xtTT....',
	'....ttTT....',
	'....ttTTo...',
	'....ttTTo...',
	'....ttTTo...',
	'...tttTTTo..',
	'..ottTTTTo..',
	'...o.o.o.o..'
);
const GIANT_MUSH_B = mk(
	'............',
	'....oddo....',
	'..odlLLldo..',
	'.dlLLcLFFdo.',
	'.lLcLFFlFFd.',
	'dlLFFFFcFFdo',
	'dLFFlFFFFFdo',
	'dFFFFFdFFFDo',
	'oDdDDDDDDDDo',
	'.oDcDcDcDco.',
	'....tttT....',
	'....ttTT....',
	'....ctTT....',
	'....ttTT....',
	'....ttTTo...',
	'....ttTTo...',
	'....ttTTo...',
	'...tttTTTo..',
	'..ottTTTTo..',
	'...o.o.o.o..'
);
const SMALL_MUSH = bot(
	'......lLl...',
	'..lL.lLFFd..',
	'.lLFdoDDDo..',
	'.dDDo..tT...',
	'...tT..tT...',
	'...tT.lLl...',
	'...tTlLFFd..',
	'..otToDtTDo.',
	'...o.o.o.o..'
);
const SMALL_MUSH_B = bot(
	'....lLLl....',
	'...lLxLFd...',
	'...dDDDDd...',
	'.lL..tT..lL.',
	'lLFd.tT.lLFd',
	'dDDo.tT.dDDo',
	'.tT..tT..tT.',
	'otTootTootTo',
	'..o..oo..o..'
);

const FLOWER_BED = bot(
	'........lL..',
	'..lL...lFyF.',
	'.lFyF...dFd.',
	'..dFd....e..',
	'...e....Ee..',
	'...eE..e.e..',
	'..Ee.eE..e..',
	'..e..e..Ee..',
	'..EeeEe.eE..',
	'...EoEoEoE..'
);
const FLOWER_BED_B = bot(
	'....lL......',
	'...lFqF..lL.',
	'.lL.dFd.lFyF',
	'lFcF.e...dFd',
	'.dFd.Ee...e.',
	'..e..eE..Ee.',
	'..Ee.e..e.e.',
	'..eEeeEe.Ee.',
	'..EeEeEeEeE.',
	'...EoEoEoE..'
);
const LILY = bot(
	'....lL......',
	'...lFyF.....',
	'..eedFdee...',
	'.evEeeeeeE..',
	'.eeeEeeeEE..',
	'..EEeeeEE...',
	'...cEEEEc...',
	'....c..c....'
);
const STUMP = bot(
	'...tttttt...',
	'..tTTTTTTt..',
	'.tTttTTttTt.',
	'.tTtTwtTtTW.',
	'.wtTttttTtW.',
	'.wwtTTTTtWWo',
	'owwwWwWWWWWo',
	'.oooooooooo.'
);

// ---- rock, crystal, coral -----------------------------------------------------------------------------

const BOULDER = bot(
	'....olllo...',
	'...olLLLlo..',
	'..olLLLLLld.',
	'.olLLLLlldFo',
	'.dllllldFFFo',
	'dFFFFFdFFFdo',
	'dFlFFFdFFdDo',
	'dFFFFdFFFdDo',
	'dlFFFdFFddDo',
	'dFFFFdFFdDDo',
	'dFFdFFdFdDDo',
	'odFFFFdFdDo.',
	'.odDdFdDDDo.',
	'..oDDDDDDo..',
	'...oooooo...',
	'..o.o.o.o...'
);
const RUBBLE = bot(
	'....lLl.....',
	'...lLFFd....',
	'..lLFFdDo...',
	'..lFFdDDolL.',
	'.lLl.dDolLFd',
	'lLFFdoo.lFdD',
	'lFFdDo.lFFdo',
	'dFdDDolLFdDo',
	'.ooooodDDoo.'
);
const ORE_VEIN: [number, number, string][] = [[3, 10, 'L'], [4, 10, 'l'], [4, 11, 'F'], [5, 12, 'F'], [8, 9, 'l'], [8, 10, 'F'], [7, 13, 'l'], [2, 13, 'F'], [6, 15, 'l']];
const SULFUR_KNOBS = dc('.L.', 'LlF', '.d.');
const SNOWDRIFT = bot(
	'....lLLl....',
	'..lLLLLLLl..',
	'.lLLFFFLLLl.',
	'lLFFFFFFFFLd',
	'FFFFFFFFFFFd',
	'dFFFFFFFFFdd',
	'ddFFFFFFFddD',
	'.ddddddddDD.'
);

const CLIFFS: Rows[] = [
	mk(
		'............',
		'.....ol.....',
		'....olLd....',
		'....lLFdo...',
		'...olLFFdo..',
		'...lLFdFDo..',
		'..olLFFdFdo.',
		'..lLFdDFFDo.',
		'.olLFFdFFDdo',
		'.lLFdDDdFFDo',
		'.lFFFFdFdFDo',
		'olLFdFFFdDDo',
		'lLFFFdDFFdDd',
		'dDdDDDdDDDDD',
		'FFFFdFFFdFDd',
		'lFdFFFdDFDDd',
		'dFFFdFFFFdDD',
		'dDdFFdDdFdDD',
		'DdDdDDdDDDDD',
		'DDDDDDDDDDDD'
	),
	mk(
		'............',
		'..ol......o.',
		'..lLd....lLo',
		'.olLFo..olFd',
		'.lLFFdoolLFd',
		'.lFdFDdlLFFo',
		'olFFFdDlFdFo',
		'lLFdFFdFFdDo',
		'lFFFdDFFdFDd',
		'lFdFFFdFFFDd',
		'dDDdDDDdDDDD',
		'lLFFdFFlFFdD',
		'lFFdFFFFdFDd',
		'FFdFFdFFFdDd',
		'lFFFFdDFFdDD',
		'dDdDDDdDDdDD',
		'lFFdFFFlFFdD',
		'FdFFFdFFdFDD',
		'DdDdDDdDdDDD',
		'DDDDDDDDDDDD'
	)
];

const CRYSTAL = mk(
	'............',
	'............',
	'.....L......',
	'.....Lo.....',
	'....lLFo....',
	'....lLFo....',
	'....lLFd..L.',
	'..L.lLFd.lFo',
	'.lLolLFdolFo',
	'.lLFlLFdlLFo',
	'.lLFlLFdlFdo',
	'.lLFdLFdlFdo',
	'..lFdlFdlFd.',
	'..lFdlFDlFd.',
	'..dFdlFDdDd.',
	'..dDdlFDdDo.',
	'.odDDdDDDDo.',
	'.oGgGGgGGGo.',
	'ogGGgGGGgGGo',
	'.o.o.o.o.o..'
);
const CRYSTAL_B = mk(
	'............',
	'............',
	'............',
	'...L....L...',
	'...Lo..lLo..',
	'..lLFo.lLFo.',
	'..lLFd.lLFd.',
	'.LlLFd.lLFdL',
	'lFlLFdlLFdlF',
	'lFdLFdlLFdFd',
	'.lFdLFdLFdF.',
	'.lFdFFdLFd..',
	'..dFdFDlFd..',
	'..dFdFDdFd..',
	'..dDdlFDdD..',
	'..dDdlFDdDo.',
	'.odDDdDDDDo.',
	'.oGgGGgGGGo.',
	'ogGGgGGGgGGo',
	'.o.o.o.o.o..'
);
const CRYSTAL_TIPS: [number, number][] = [[5, 2], [10, 6], [2, 7]];
const CRYSTAL_TIPS_B: [number, number][] = [[3, 3], [8, 3], [1, 7], [11, 7]];

const CORAL = bot(
	'..l.....l...',
	'.lF.l..lF.l.',
	'.lF.lF.lFdF.',
	'..lFlFd.lFd.',
	'.l.lFd.lFd..',
	'.lF.lFlFd.l.',
	'..lFdlFd.lFd',
	'...lFFFdlFd.',
	'....lFFFFd..',
	'.....lFd....',
	'..l..lFd....',
	'.lFd.lFd.lL.',
	'.lFFdlFdlFFd',
	'.dFFdlFddFFd',
	'odDDDdDDdDDo',
	'.oooooooooo.'
);

// ---- holes, stairs, sky ----------------------------------------------------------------------------

const STAIR_DOWN = bot(
	'.oLLLLLLLLd.',
	'.lFFFFFFFFdo',
	'.lzzzzzzzzdo',
	'.lLLLLLLLzdo',
	'.lFFFFFFFzdo',
	'.lzzzzzzzzdo',
	'.ldLLLLLzzdo',
	'.lddFFFFzzdo',
	'.lzzzzzzzzdo',
	'.lzzdLLzzzdo',
	'.lzzzddzzzdo',
	'.lzzzzzzzzdo',
	'.dddddddddDo',
	'..oooooooooo'
);
const STAIR_UP = bot(
	'....oxxo....',
	'....oLLo....',
	'...oLLLLo...',
	'...lFFFFdo..',
	'..oLLLLLLo..',
	'..lFFFFFFdo.',
	'..dddddddDo.',
	'.oLLLLLLLLo.',
	'.lFFFFFFFFdo',
	'.ddddddddDDo',
	'oLLLLLLLLLLo',
	'lFFFFFFFFFFd',
	'dddddddddDDo',
	'LLLLLLLLLLLd',
	'FFFFFFFFFFFd',
	'dddddddddDDD',
	'oooooooooooo'
);
function chasm(h: number): string[] {
	const base = blank().map(() => 'o'.repeat(PW));
	return over(base, specks(h, { D: 'z', d: 'D' }));
}
const SKYWISP = dc('lLl..', '.dFFl');
const voidSky = (h: number) => scatter(specks(h, { D: 'd', d: 'D' }), [SKYWISP, SKYWISP, dc('lL', '.d')], h, 1 + ((h >>> 20) % 2));

const UPDRAFT = mk(
	'..l.....L...',
	'..l.....l...',
	'..F.........',
	'.......l..L.',
	'....L..l..l.',
	'....l..F....',
	'....F.....F.',
	'.L..........',
	'.l......L...',
	'.F......l...',
	'......L.F...',
	'..L...l.....',
	'..l...F..L..',
	'..F......l..',
	'.........F..',
	'....L.......',
	'.L..l....L..',
	'.l..F....l..',
	'.F.......F..',
	'............'
);

// ---- fire, vents, springs --------------------------------------------------------------------------

const FIRE_TOPS: Rows[] = [
	mk('............', '............', '......y.....', '......n.....', '..y..yn.....', '..n..nyn..y.', '..nn.nyn..n.', '.rnyrnyyn.n.', '.rnyynyynrn.'),
	mk('............', '.....y......', '.....n...y..', '.y...ny..n..', '.n..nyn..n..', '.nn.nyn.rn..', '..n.ryyn.n..', '..nrnyyn.nr.', '.rnyyyynnyr.'),
	mk('............', '............', '............', '.......y....', '...y...ny...', '...n..nyn...', '..rn..nyn.y.', '..rny.nyynn.', '.rnyynnyynr.')
];
const FIRE_BODY = bot(
	'rnyyxyyynnr.',
	'rnyxxyxyynr.',
	'rnyxxxxxynrr',
	'Rnyxxxxxxynr',
	'Rrnyxxxxynnr',
	'RrnyyxxyynrR',
	'.Rrnnyyynrr.',
	'.RRrnnnnrR..',
	'..RRrrrrRR..',
	'...oRRRRo...',
	'....o..o....'
);
function fire(h: number, frame: number): string[] {
	const t = (frame + h) & 7;
	let r = over(FIRE_BODY, FIRE_TOPS[t % 3].slice(0, 9));
	if (t & 1) r = r.map((row, y) => (y >= 9 ? recolour([row], { x: 'y', y: 'x' })[0] : row));
	return r;
}

const GEYSER_MOUND = bot(
	'....oUUo....',
	'..ttdUUdtt..',
	'.tTtlFFldtT.',
	'tTlLFFFFlTtT',
	'TtFFdFFdFFtT',
	'dTdFFdFFdTdo',
	'.oooooooooo.'
);
const PLUME = mk(
	'....lL......',
	'...lLLl.....',
	'....lFL.....',
	'.....LL.....',
	'....lLFl....',
	'.....LL.....',
	'......lL....',
	'.....lLL....',
	'.....LFl....',
	'......L.....',
	'.....lL.....',
	'.....LLl....',
	'......L.....'
);
function geyser(h: number, frame: number): string[] {
	const t = (frame + h) & 7;
	let pl = wrapY(mk(...PLUME), -((t * 5) >> 1)).slice(0, 13);
	if (t >= 6) pl = pl.map((r, y) => (y < 7 ? '.'.repeat(PW) : r));
	return over(GEYSER_MOUND, pl);
}

const VENT = dc('.lF.', 'lFod', 'dnrD', '.dD.');
const PUFF = dc('.l.', 'lLl', '.l.');
function fumarole(h: number, frame: number): string[] {
	let r = over(specks(h), VENT, 4, 6);
	r = over(r, VENT, 1, 13);
	r = over(r, VENT, 7, 13);
	const t = (frame + h) & 7;
	if (t < 6) r = over(r, t < 3 ? PUFF : SPORE.map((s) => s.replace(/L/g, 'l')), 5, 4 - (t >> 1));
	if (((t + 3) & 7) < 5) r = over(r, PUFF, 8, 11 - (((t + 3) & 7) >> 1));
	return r;
}

const BEEHIVE = mk(
	'............',
	'............',
	'WwwwwwWWwW..',
	'oWWWWWWWWWo.',
	'....WWo.....',
	'....lFd.....',
	'...lLFFd....',
	'...dddDD....',
	'..lLFFFFd...',
	'..lFFFFFd...',
	'..dddddDD...',
	'..lLFFFFd...',
	'..lFFoFFd...',
	'..ddooDDD...',
	'...lFFFd....',
	'...dddD.....',
	'....dD......',
	'............',
	'............',
	'............'
);
const BEE_AT: [number, number][] = [[9, 6], [10, 9], [1, 8], [9, 13], [0, 11], [8, 16]];

const BARK = mk(
	'lFdolLFdolFd',
	'lFdolLFdoFFd',
	'FFdolFFdolFd',
	'lFdDolFFdolF',
	'lFdDolLFdolF',
	'lFFdolLFdoFd',
	'lFdolLFFdolF',
	'FFdolFFdDolF',
	'lFdolLDDdolF',
	'lFdolDooDolF',
	'lFdolDDdFolF',
	'lFFdolLFdoFd',
	'lFdolLFdolFd',
	'FFdolLFFdoFd',
	'lFdolFFdDolF',
	'lFdDolLFdolF',
	'lFdolLFFdoFd',
	'FFdolLFdolFd',
	'lFdolFFdolFd',
	'lFdolLFdolFd'
);

// ---- entry --------------------------------------------------------------------------------------

const SWAY_TREE = [0, 0, 0, 1, 1, 0, 0, 0];
const BERRIES = ['r', 'p', 'y', 'x'];

function tree(rows: Rows, h: number, frame: number, top: number): string[] {
	return bend(rows, SWAY_TREE[(frame + h) & 7], 0, top);
}

function natural(g: string, fgIdx: number, bgIdx: number, h: number, frame: number, nm: number, mem: boolean): Rows | null {
	const t = (frame + h) & 7;
	switch (g) {
		case '.':
			if (fgIdx === 10) return null; // carpet
			if (fgIdx === 6 || fgIdx === 7) return grassGround(h);
			if (fgIdx === 15) return sandGround(h);
			if (fgIdx === 14) return earthGround(h);
			if (fgIdx === 3) return bgIdx === 19 ? saltGround(h) : snowGround(h);
			if (fgIdx === 1) return bgIdx === 20 ? ashGround(h) : floorGround(h);
			return earthGround(h);
		case '∙':
			return roadGround(h);
		case ',':
			if (fgIdx === 3 || fgIdx === 2) return BONES;
			if (fgIdx === 12 || fgIdx === 13) return fungusGround(h);
			return mossGround(h);
		case ':':
			if (fgIdx === 12 || fgIdx === 13) return crystalSand(h);
			if (fgIdx === 14) return quicksand(h, frame);
			return floorGround(h);
		case '"':
			if (bgIdx === 4) return bend(KELP, SWAY[t], 0, 9);
			if (fgIdx === 8 || fgIdx === 9) return bend(REEDS, SWAY[t], 0, 6);
			if (fgIdx >= 14 && fgIdx <= 17) return bend(CROPS, SWAY[t] > 0 ? 1 : 0, 0, 6);
			return tallGrass(h, frame);
		case '♣': {
			const r = OAKS[h % 3].slice();
			if ((h >>> 7) % 4 === 0) for (const [x, y] of FRUIT_AT) put(r, x, y, (h >>> 9) & 1 ? 'q' : 'r');
			return tree(r, h, frame, 5);
		}
		case '♠':
			if (fgIdx === 13) return (h & 1) ? GIANT_MUSH_B : GIANT_MUSH;
			if (fgIdx === 12) return (h & 1) ? SMALL_MUSH_B : SMALL_MUSH;
			{
				const r = BUSHES[h & 1].slice();
				if ((h >>> 6) % 3 === 0) {
					const c = pick(BERRIES, h, 11);
					for (const [x, y] of BERRY_AT) put(r, x - (h & 1 ? 1 : 0), y - (h & 1 ? 0 : 1), c);
				}
				return r;
			}
		case '↑':
			if (fgIdx === 9) return wrapY(UPDRAFT, -((frame * 5) >> 1));
			return tree(PINES[h & 1], h, frame, 4);
		case 'τ':
			return tree(SNAGS[h & 1], h, frame, 4);
		case 'Γ':
			return tree(PALM, h, frame, 8);
		case 'φ': {
			const r = CACTI[h & 1].slice();
			if ((h >>> 8) % 4 === 0) {
				if (h & 1) { put(r, 4, 8, 'r'); put(r, 9, 10, 'r'); }
				else { put(r, 5, 0, 'q'); put(r, 6, 0, 'q'); }
			}
			return r;
		}
		case '‼':
			return tree(BAMBOO, h, frame, 3);
		case '°': {
			if (bgIdx === 4) return LILY;
			const bed = (h >>> 2) & 1 ? FLOWER_BED_B : FLOWER_BED;
			return h & 1 ? flipX(bed) : bed;
		}
		case 'ⁿ': {
			const r = STUMP.slice();
			const mod = (h >>> 5) % 4;
			if (mod === 0) { put(r, 1, 16, 'e'); put(r, 2, 17, 'E'); put(r, 1, 17, 'e'); }
			else if (mod === 1) { put(r, 8, 16, 'q'); put(r, 9, 16, 'y'); put(r, 9, 17, 't'); }
			return r;
		}
		case '*': {
			if (fgIdx === 15 || fgIdx === 16 || fgIdx === 17 || mem) return null; // lanterns
			const alt = (h & 1) === 1;
			const r = (alt ? CRYSTAL_B : CRYSTAL).slice();
			const tips = alt ? CRYSTAL_TIPS_B : CRYSTAL_TIPS;
			if (t < 2) {
				const [x, y] = tips[(h >>> 3) % tips.length];
				put(r, x, y, 'x');
				if (y + 1 < PH) put(r, x, y + 1, 'L');
			} else if (t === 4 || t === 5) {
				const [x, y] = tips[((h >>> 3) + 1) % tips.length];
				put(r, x, y, 'c');
			}
			return r;
		}
		case '¥': {
			const r = CORAL.slice();
			put(r, 8 + (h % 3), 8 - t, 'c');
			put(r, 2 + (h % 2), 8 - ((t + 4) & 7), 'c');
			return r;
		}
		case '%': {
			if (bgIdx === 19 && fgIdx === 3) return SNOWDRIFT;
			const sh = ((h >>> 3) % 3) - 1;
			if (bgIdx === 20 && fgIdx !== 1) {
				if (fgIdx === 16 || fgIdx === 17) {
					const r = recolour(BOULDER, { L: 'm', l: 'g', F: 'g', d: 'G', D: 'G' });
					for (const [x, y, c] of ORE_VEIN) put(r, x, y, c);
					return bend(r, sh, 0, PH);
				}
				if (fgIdx === 12) return bend(recolour(BOULDER, { l: 'd', F: 'D', d: 'o', D: 'o' }), sh, 0, PH);
				if (fgIdx === 15) {
					let r = over(BOULDER, SULFUR_KNOBS, 2, 9);
					r = over(r, SULFUR_KNOBS, 6, 12);
					return bend(r, sh, 0, PH);
				}
				const b = BOULDER.slice();
				if ((h >>> 6) % 4 === 0) {
					// mossy boulder cap
					put(b, 4, 6, 'e'); put(b, 5, 6, 'e'); put(b, 6, 6, 'E'); put(b, 5, 7, 'E');
				} else if ((h >>> 6) % 4 === 1) {
					// quartz/crystal glint embedded in stone
					put(b, 3, 10, 'x'); put(b, 4, 10, 'c'); put(b, 7, 12, 'L');
				}
				return bend(b, sh, 0, PH);
			}
			if (fgIdx === 1) return bend(RUBBLE, sh, 0, PH);
			return null;
		}
		case '▲':
			return CLIFFS[h & 1];
		case '║':
			return wrapY(FALLS, (frame * 5) >> 1);
		case '░':
			if (bgIdx === 4) return over(specks(h, { D: 'u', d: 'c' }), wrapY(DROPS, (frame * 5) >> 1));
			if (bgIdx === 19) return cloudGround(h);
			return iceGround(h);
		case '▒':
			return (h & 1 ? flipX(wrapY(GLACIER, 10)) : wrapY(GLACIER, (h >>> 3) % 20));
		case '≈':
			if (bgIdx === 10 || fgIdx === 16 || fgIdx === 17) return lava(h, frame, nm, false);
			if (fgIdx === 1 && bgIdx === 0) return lava(h, frame, nm, true);
			if (bgIdx === 4 || fgIdx === 4 || fgIdx === 5 || fgIdx === 8 || fgIdx === 9) return water(true, h, frame, nm, false, false);
			if (mem) return water(true, h, 0, 255, false, true);
			return null;
		case '~':
			if (bgIdx === 10) return lava(h, frame, nm, false);
			if (bgIdx === 4) return water(false, h, frame, nm, false, false);
			if (bgIdx === 8) return water(false, h, frame, nm, true, false);
			if (mem) return water(false, h, 0, 255, false, true);
			return null; // bunting, items
		case '¤':
			return fgIdx === 3 && bgIdx === 8 ? geyser(h, frame) : null;
		case '∴':
			return fumarole(h, frame);
		case '>':
			return STAIR_DOWN;
		case '<':
			return STAIR_UP;
		case ' ':
			if (bgIdx === 20 && fgIdx === 1) return chasm(h);
			if (bgIdx === 20 && fgIdx === 5) return voidSky(h);
			return null;
		case '^':
			return fgIdx === 10 || fgIdx === 11 || fgIdx === 15 || fgIdx === 16 || fgIdx === 17 ? fire(h, frame) : null;
		case '█':
			return (h >>> 4) % 4 === 0 ? over(BARK, dc('e.', 'Ee', '.e'), 9, 3 + ((h >>> 6) % 14)) : BARK;
		case 'δ': {
			const r = BEEHIVE.slice();
			put(r, BEE_AT[t % 6][0], BEE_AT[t % 6][1], 'z');
			put(r, BEE_AT[(t + 3) % 6][0], BEE_AT[(t + 3) % 6][1], 'z');
			return r;
		}
	}
	return null;
}

/** pixel sprite for a natural tile glyph, or null when another drawer owns it */
export function natureSprite(g: string, fgIdx: number, bgIdx: number, h: number, frame: number, nm: number): Rows | null {
	const mem = fgIdx === 1 && bgIdx === 20;
	const r = natural(g, fgIdx, bgIdx, h >>> 0, frame & 7, nm, mem);
	if (!r) return null;
	return mem ? mono(r) : r;
}
