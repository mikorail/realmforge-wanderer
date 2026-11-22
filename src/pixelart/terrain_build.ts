// Hand-authored pixel art for man-made tiles: walls, floors, doors, gates, stairs, furniture, devices.
// Light comes from the top left: highlights (L l) on top and left edges, shade (d D) on the right and a
// 1px dark underside. Ramp letters follow the tile's own colour, fixed letters stay what the object is.
// Colour indices (engine/ui.go): 0 black 1 dgray 2 gray 3 white 4 dblue 5 blue 6 dgreen 7 green 8 dcyan
// 9 cyan 10 dred 11 red 12 dmagenta 13 magenta 14 brown 15 yellow 16 dorange 17 orange; bg 18 Bg 19 light 20 dark.
// Remembered (out of sight) cells come as fg 1 / bg 20, so every drawing must also read in dark grey.
import { PW, PH, type Rows, mk } from './types';

type Grid = string[][];
const grid = (c = '.'): Grid => Array.from({ length: PH }, () => Array<string>(PW).fill(c));
const toRows = (g: Grid): Rows => g.map((r) => r.join(''));
const fromRows = (r: Rows): Grid => r.map((s) => s.split(''));
function put(g: Grid, x: number, y: number, c: string): void {
	if (c !== '.' && x >= 0 && x < PW && y >= 0 && y < PH) g[y][x] = c;
}
function hx(a: number, b = 0, c = 0): number {
	let n = (a * 374761393 + b * 668265263 + c * 1442695041) | 0;
	n = Math.imul(n ^ (n >>> 13), 1274126177);
	return (n ^ (n >>> 16)) & 0x7fffffff;
}
const DARK: Record<string, string> = { L: 'l', l: 'F', F: 'd', d: 'D', D: 'o' };
const LITE: Record<string, string> = { o: 'D', D: 'd', d: 'F', F: 'l', l: 'L' };
const darker = (c: string) => DARK[c] ?? c;
const lighter = (c: string) => LITE[c] ?? c;

/** strict art literal: rows that are not 20 x 12 are recorded (checked by a node script), then padded */
export const ART_ERRORS: string[] = [];
function art(...r: string[]): Rows {
	if (r.length !== PH) ART_ERRORS.push(`rows=${r.length}: ${r[0]}`);
	r.forEach((s, i) => {
		if (s.length !== PW) ART_ERRORS.push(`row ${i} len ${s.length}: ${s}`);
	});
	return mk(...r);
}
/** overlay a list of [x, y, ch] */
function dots(g: Grid, pts: [number, number, string][]): void {
	for (const [x, y, c] of pts) put(g, x, y, c);
}

const N = 1, E = 2, S = 4, W = 8;

// ---------------------------------------------------------------------------------------------- walls

function brick(v: number): Grid {
	const g = grid('D');
	for (let y = 0; y < PH; y++) {
		const c = y >> 2, r = y & 3, off = c & 1 ? 3 : 0;
		if (r === 3) continue;
		for (let x = 0; x < PW; x++) {
			const p = x + off, bx = p % 6, bi = Math.floor(p / 6);
			if (bx === 5) continue;
			const straddle = off === 3 && bi !== 1;
			const tone = straddle ? 0 : hx(v, c, bi) % 6;
			let ch = r === 0 ? 'l' : r === 1 ? 'F' : 'd';
			if (bx === 0 && r < 2) ch = r === 0 ? 'L' : 'l';
			if (bx === 4 && r > 0) ch = 'd';
			if (tone === 4) ch = darker(ch);
			if (tone === 5 && r === 1) ch = 'l';
			g[y][x] = ch;
		}
	}
	if (v % 3 === 0) put(g, 3 + (v % 5), 1 + ((v >> 2) % 4) * 4, 'D');
	if (v % 5 === 1) put(g, 8 - (v % 3), 9, 'D');
	return g;
}

function ashlar(v: number): Grid {
	const g = grid('D');
	for (let y = 0; y < PH; y++) {
		const c = Math.floor(y / 5), r = y % 5, off = c & 1 ? 3 : 0;
		if (r === 4) continue;
		for (let x = 0; x < PW; x++) {
			const p = x + off, bx = p % 6, bi = Math.floor(p / 6);
			if (bx === 5) continue;
			let ch = 'F';
			if (r === 0 || bx === 0) ch = 'l';
			if (r === 0 && bx === 0) ch = 'L';
			if (r === 3 || (bx === 4 && r > 0)) ch = 'd';
			const straddle = off === 3 && bi !== 1;
			if (!straddle && hx(v, c, bi) % 7 === 0 && ch === 'F') ch = 'd';
			g[y][x] = ch;
		}
	}
	// pits and chips in the faces
	for (let k = 0; k < 3; k++) {
		const n = hx(v, k, 99);
		const x = 1 + (n % 10), y = 1 + ((n >> 4) % 18);
		if (g[y][x] === 'F') g[y][x] = n & 64 ? 'd' : 'l';
	}
	return g;
}

/** irregular rubble-stone courses: [height incl. joint row, vertical joint columns] */
const ROUGH: [number, number[]][] = [
	[6, [3, 8]],
	[5, [0, 6]],
	[5, [4, 9]],
	[4, [1, 7]]
];
function rough(v: number): Grid {
	const g = grid('D');
	let y0 = 0;
	for (const [hgt, js] of ROUGH) {
		const isJ = (x: number) => js.includes(((x % PW) + PW) % PW);
		for (let r = 0; r < hgt - 1; r++) {
			for (let x = 0; x < PW; x++) {
				if (isJ(x)) continue;
				const lft = isJ(x - 1), rgt = isJ(x + 1);
				let ch = 'F';
				if (r === 0 || lft) ch = 'l';
				if (r === hgt - 2 || rgt) ch = 'd';
				if ((r === 0 || r === hgt - 2) && (lft || rgt)) ch = 'D'; // knocked-off corners
				if (ch === 'F' && hx(x, y0 + r, v) % 9 === 0) ch = 'd';
				g[y0 + r][x] = ch;
			}
		}
		y0 += hgt;
	}
	return g;
}

function timber(v: number, nm: number): Grid {
	const g = grid('D');
	for (let y = 0; y < PH; y++) {
		const r = y & 3, k = y >> 2;
		if (r === 3) continue;
		for (let x = 0; x < PW; x++) {
			let ch = r === 0 ? 'l' : r === 1 ? 'F' : 'd';
			if (r === 1 && hx(x >> 1, k, v) % 5 === 0) ch = 'd'; // grain
			if (r === 0 && hx(x, k, v + 3) % 7 === 0) ch = 'L';
			g[y][x] = ch;
		}
		// a knot on some logs
		const n = hx(v, k, 5);
		if (n % 3 === 0 && r === 1) {
			const kx = 3 + (n >> 3) % 6;
			g[y][kx] = 'D';
			g[y][kx + 1] = 'd';
		}
		// butt joint
		if (n % 4 === 1) {
			const sx = 2 + (n >> 5) % 8;
			g[y][sx] = 'D';
			if (r === 0) g[y][sx + 1] = 'L';
		}
	}
	// rounded log ends where the wall stops
	for (let k = 0; k < 5; k++) {
		const y = k * 4;
		if (!(nm & W)) {
			g[y][0] = 'D';
			g[y + 1][0] = 'l';
			g[y + 1][1] = 'L';
			g[y + 2][0] = 'D';
		}
		if (!(nm & E)) {
			g[y][11] = 'D';
			g[y + 1][11] = 'd';
			g[y + 2][11] = 'D';
		}
	}
	return g;
}

function adobe(v: number, nm: number): Grid {
	const g = grid('F');
	for (let y = 0; y < PH; y++)
		for (let x = 0; x < PW; x++) {
			const n = hx(x, y, v) % 23;
			if (n === 0) g[y][x] = 'l';
			else if (n === 1) g[y][x] = 'd';
		}
	const kind = v & 3;
	if (kind === 1) {
		// a hairline crack
		let x = 3 + (v % 5);
		for (let y = 5; y < 12; y++) {
			put(g, x, y, y === 5 ? 'D' : 'd');
			x += (hx(v, y) & 1) ? 1 : -1;
			x = Math.max(1, Math.min(10, x));
		}
	} else if (kind === 2) {
		// plaster fallen off: mudbrick shows through
		const pat = ['dddddd', 'lFdlFd', 'DDDDDD', 'FdlFdl', 'dddddd'];
		for (let r = 0; r < pat.length; r++) for (let i = 0; i < 6; i++) put(g, 3 + i, 9 + r, pat[r][i]);
		for (let i = 0; i < 6; i++) put(g, 3 + i, 14, 'l');
	}
	if (!(nm & N)) {
		for (let x = 0; x < PW; x++) {
			g[0][x] = 'L';
			g[1][x] = 'l';
		}
		if (!(v & 4)) {
			// protruding roof beams (vigas)
			dots(g, [[1, 3, 'w'], [2, 3, 'w'], [1, 4, 'W'], [2, 4, 'W'], [8, 3, 'w'], [9, 3, 'w'], [8, 4, 'W'], [9, 4, 'W'], [3, 3, 'd'], [10, 3, 'd']]);
		}
	}
	if (!(nm & S)) for (let x = 0; x < PW; x++) g[17][x] = g[18][x] = 'd';
	return g;
}

function marble(v: number): Grid {
	const g = grid('F');
	for (let s = 0; s < 2; s++) {
		const top = s * 10;
		for (let x = 0; x < PW; x++) {
			g[top + 9][x] = 'D';
			g[top + 8][x] = 'd';
			g[top][x] = 'l';
		}
		for (let y = top; y < top + 9; y++) {
			g[y][11] = 'D';
			g[y][10] = 'd';
			g[y][0] = 'l';
		}
		g[top][0] = 'L';
		g[top][1] = 'L';
		// a vein running down the slab
		const n = hx(v, s, 11);
		let x = 2 + (n % 6);
		for (let y = top + 1; y < top + 8; y++) {
			put(g, x, y, 'd');
			if (x + 1 < 10) put(g, x + 1, y, 'l');
			if ((n >> (y - top)) & 1) x = Math.min(8, x + 1);
		}
		if (n & 256) put(g, 2 + ((n >> 9) % 6), top + 3 + ((n >> 12) % 4), 'd');
	}
	return g;
}

function metal(v: number): Grid {
	const g = grid('F');
	for (let y = 0; y < PH; y++) {
		const r = y % 10;
		for (let x = 0; x < PW; x++) {
			let ch = 'F';
			if (x - r === 2 || x - r === 3) ch = 'l'; // sheen
			if (r === 0 || x === 0) ch = 'l';
			if (r === 8 || x === 10) ch = 'd';
			if (r === 9 || x === 11) ch = 'D';
			g[y][x] = ch;
		}
	}
	for (const py of [0, 10])
		for (const [rx, ry] of [[2, 2], [8, 2], [2, 7], [8, 7]]) {
			put(g, rx, py + ry, 'L');
			put(g, rx, py + ry + 1, 'D');
		}
	const kind = v & 3;
	if (kind === 1) {
		for (const y of [13, 15]) for (let x = 3; x <= 7; x++) {
			put(g, x, y, 'D');
			put(g, x, y + 1, 'l');
		}
	} else if (kind === 2) {
		dots(g, [[3, 9, 'n'], [4, 9, 'R'], [7, 18, 'R'], [6, 19, 'n'], [1, 8, 'R']]);
	} else if (kind === 3) {
		for (let x = 0; x < 11; x++) for (const y of [14, 15]) put(g, x, y, (x + y) % 4 < 2 ? 'y' : 'z');
	}
	return g;
}

/** faceted glass, ice or obsidian: a diamond lattice that tiles over 12 x 20 */
function crystal(v: number): Grid {
	const g = grid('F');
	for (let y = 0; y < PH; y++)
		for (let x = 0; x < PW; x++) {
			const u = (5 * x + 3 * y) % 30, w = (((5 * x - 3 * y) % 30) + 30) % 30;
			let ch = w >= 15 ? 'd' : 'F';
			if (u < 15 && w < 15) ch = 'l';
			if (u < 5) ch = 'L';
			if (w < 5) ch = 'D';
			g[y][x] = ch;
		}
	const n = hx(v, 7);
	put(g, 1 + (n % 10), 1 + ((n >> 4) % 18), 'x');
	return g;
}

function bone(v: number): Grid {
	const g = grid('D');
	for (let y = 0; y < PH; y++)
		for (let x = 0; x < PW; x++) {
			const c = x % 4, k = x >> 2, knob = (y + k * 7) % 10;
			let ch = c === 0 ? 'D' : c === 1 ? 'L' : c === 2 ? 'l' : 'd';
			if (knob === 0 || knob === 1) {
				// knuckle of the bone swells into the gap
				if (c === 0) ch = knob === 0 ? 'l' : 'd';
				if (c === 1) ch = 'L';
			}
			g[y][x] = ch;
		}
	for (const y of [4, 14]) for (let x = 0; x < PW; x++) {
		g[y][x] = x % 3 === 0 ? 'w' : 'W';
		g[y + 1][x] = 'W';
	}
	if (v % 3 === 0) {
		const sk = ['lLLLl', 'LzLzL', 'lLLLl', '.dLd.'];
		for (let r = 0; r < 4; r++) for (let i = 0; i < 5; i++) put(g, 3 + i, 8 + r, sk[r][i]);
		put(g, 3, 11, 'D');
		put(g, 7, 11, 'D');
	}
	return g;
}

function bamboo(v: number): Grid {
	const g = grid('d');
	const OFF = [0, 3, 6, 2];
	for (let y = 0; y < PH; y++)
		for (let x = 0; x < PW; x++) {
			const c = x % 3, k = (x / 3) | 0, node = (y + OFF[k] + (v & 1) * 5) % 10;
			let ch = c === 0 ? 'l' : c === 1 ? 'F' : 'd';
			if (node === 0) ch = 'D';
			else if (node === 1 && c < 2) ch = 'L';
			g[y][x] = ch;
		}
	for (let x = 0; x < PW; x++) {
		g[7][x] = x % 3 === 1 ? 'w' : 'W';
	}
	return g;
}

function hedge(v: number): Grid {
	const g = grid('D');
	const leaf = ['.lF', 'lFd', '.dD'];
	for (let gy = -1; gy < 6; gy++)
		for (let gx = 0; gx < 4; gx++) {
			const ox = gx * 3 + (gy & 1 ? 1 : 0) + (hx(gx, gy, 3) & 1), oy = gy * 4 + ((hx(gx, gy, 5) & 1) as number);
			for (let r = 0; r < 3; r++) for (let i = 0; i < 3; i++) {
				const x = (ox + i) % PW, y = oy + r;
				if (y >= 0 && y < PH) put(g, x, y, leaf[r][i]);
			}
		}
	for (let k = 0; k < 4; k++) {
		const n = hx(v, k, 9);
		put(g, n % 12, (n >> 4) % 20, 'W');
	}
	if (v % 5 === 0) put(g, 4 + (v % 4), 6, 'r');
	return g;
}

function coral(v: number): Grid {
	const g = grid('F');
	for (let cy = 0; cy < 5; cy++)
		for (let cx = 0; cx < 4; cx++) {
			const n = hx(cx, cy, v);
			const x = cx * 3 + (n % 2), y = cy * 4 + ((n >> 2) % 3);
			put(g, x + 1, y + 1, 'D');
			put(g, x, y, 'l');
			put(g, x + 1, y, 'l');
			put(g, x + 2, y + 1, 'd');
			put(g, x + 1, y + 2, 'd');
		}
	for (let k = 0; k < 3; k++) {
		const n = hx(v, k, 17);
		put(g, n % 12, (n >> 4) % 20, 'L');
	}
	return g;
}

/** light the open top and left of a wall, shade the open right, a dark underside where it stops */
function edgeShade(g: Grid, nm: number): Grid {
	if (!(nm & N)) for (let x = 0; x < PW; x++) g[0][x] = g[0][x] === 'D' ? 'l' : 'L';
	if (!(nm & W)) for (let y = 0; y < PH; y++) g[y][0] = lighter(g[y][0]);
	if (!(nm & E)) for (let y = 0; y < PH; y++) g[y][11] = darker(g[y][11]);
	if (!(nm & S)) {
		for (let x = 0; x < PW; x++) {
			g[18][x] = darker(g[18][x]);
			g[19][x] = 'o';
		}
	}
	return g;
}

function wall(fg: number, bg: number, v: number, nm: number): Rows {
	let g: Grid;
	switch (fg) {
		case 14: g = timber(v, nm); break;
		case 10: case 11: g = brick(v); break;
		case 0: case 1: g = rough(v); break;
		case 3: g = marble(v); break;
		case 12: case 13: g = crystal(v); break;
		case 15: g = bone(v); break;
		case 16: case 17: g = adobe(v, nm); break;
		case 7: g = bamboo(v); break;
		case 6: g = hedge(v); break;
		case 9: g = bg === 19 ? crystal(v) : coral(v); break;
		case 8: case 4: case 5: g = metal(v); break;
		default: g = ashlar(v);
	}
	return toRows(edgeShade(g, nm));
}

// --------------------------------------------------------------------------------------------- floors
// Floors stay a step darker and lower in contrast than walls, so people and items read on top of them.

function planks(v: number): Rows {
	const g = grid('d');
	for (let y = 0; y < PH; y++) {
		const r = y & 3, k = y >> 2;
		const jx = (k * 7 + 3 + (hx(v, k) % 3)) % PW;
		for (let x = 0; x < PW; x++) {
			let ch = r === 3 ? 'D' : r === 0 ? 'F' : 'd';
			if (r === 1 && hx(x, y, v) % 9 === 0) ch = 'F';
			if (r === 0 && hx(x, y, v) % 13 === 0) ch = 'l';
			if (r === 2 && hx(x, y, v + 1) % 11 === 0) ch = 'D';
			if (x === jx && r < 3) ch = 'D';
			g[y][x] = ch;
		}
		if (r === 1) {
			put(g, (jx + 1) % PW, y, 'D'); // nail by the joint
		}
	}
	return toRows(g);
}

/** bridge or ship deck: boards with dark caulking and iron nails over the beams */
function deck(v: number): Rows {
	const g = grid('F');
	for (let y = 0; y < PH; y++) {
		const r = y & 3, k = y >> 2;
		const dark = hx(v, k, 2) % 4 === 0;
		for (let x = 0; x < PW; x++) {
			let ch = r === 0 ? 'l' : r === 1 ? 'F' : r === 2 ? 'd' : 'o';
			if (dark && r < 3) ch = darker(ch);
			if (r === 1 && (x === 1 || x === 10)) ch = 'M';
			g[y][x] = ch;
		}
	}
	return toRows(g);
}

function cobbles(v: number): Rows {
	const stone = ['DlFFD', 'lFFdd', 'FFddd', 'DdddD'];
	const g = grid('o');
	for (let c = 0; c < 4; c++) {
		const off = c & 1 ? 3 : 0;
		for (let r = 0; r < 4; r++)
			for (let x = 0; x < PW; x++) {
				const p = x + off, bx = p % 6, bi = Math.floor(p / 6);
				if (bx === 5) continue;
				let ch = stone[r][bx];
				const straddle = off === 3 && bi !== 1;
				const t = straddle ? 0 : hx(v, c, bi) % 5;
				if (t === 0 && !straddle) ch = darker(ch);
				if (t === 1 && ch === 'F') ch = 'l';
				g[c * 5 + r][x] = ch;
			}
		for (let x = 0; x < PW; x++) g[c * 5 + 4][x] = 'D';
	}
	for (let x = 0; x < PW; x++) if (x % 6 === 5) for (let y = 0; y < PH; y++) if (g[y][x] === 'o') g[y][x] = 'D';
	return toRows(g);
}

function tiles(v: number): Rows {
	const g = grid('D');
	const motif = v % 9 === 0;
	for (let ty = 0; ty < 4; ty++)
		for (let tx = 0; tx < 2; tx++) {
			const alt = (tx + ty) & 1;
			for (let r = 0; r < 4; r++)
				for (let i = 0; i < 5; i++) {
					let ch = r === 0 ? 'F' : 'd';
					if (alt) ch = r === 0 ? 'd' : 'D';
					if (i === 0 && r === 0) ch = 'l';
					if (i === 4 && r > 0) ch = darker(ch);
					g[ty * 5 + r][tx * 6 + i] = ch;
				}
			if (motif) {
				put(g, tx * 6 + 2, ty * 5 + 1, 'y');
				put(g, tx * 6 + 1, ty * 5 + 2, 'Y');
				put(g, tx * 6 + 3, ty * 5 + 2, 'Y');
				put(g, tx * 6 + 2, ty * 5 + 3, 'y');
			}
		}
	for (let y = 0; y < PH; y++) for (let x = 0; x < PW; x++) if (x % 6 === 5 || y % 5 === 4) g[y][x] = 'o';
	return toRows(g);
}

/** large flagstones: [row, joint columns] per band, horizontal joints at 6, 13 and 19 */
const FLAGS: [number, number, number[]][] = [
	[0, 5, [4, 11]],
	[7, 12, [7]],
	[14, 18, [2, 9]]
];
function flagstones(v: number): Rows {
	const g = grid('F');
	for (const y of [6, 13, 19]) for (let x = 0; x < PW; x++) g[y][x] = 'd';
	for (const [a, b, js] of FLAGS)
		for (let y = a; y <= b; y++)
			for (let x = 0; x < PW; x++) {
				if (js.includes(x)) g[y][x] = 'd';
				else if (y === a || js.includes((x + PW - 1) % PW)) g[y][x] = hx(x, y, v) % 3 ? 'F' : 'l';
				else if (hx(x, y, v + 7) % 17 === 0) g[y][x] = 'd';
			}
	const n = hx(v, 3);
	if (n % 4 === 0) {
		// a cracked flag
		const cx = 1 + (n >> 3) % 8;
		dots(g, [[cx, 9, 'D'], [cx + 1, 10, 'D'], [cx + 1, 11, 'd']]);
	}
	return toRows(g);
}

function carpet(v: number): Rows {
	const g = grid('F');
	for (let y = 0; y < PH; y++)
		for (let x = 0; x < PW; x++) {
			const a = (x + y) & 3, b = (((x - y) % 4) + 4) % 4;
			if (a === 0 || b === 0) g[y][x] = 'd';
			else if (a === 2 && b === 2) g[y][x] = ((x >> 2) + (y >> 2)) & 1 ? 'y' : 'l';
		}
	if (v % 7 === 0) put(g, 1 + (v % 9), 3 + (v % 13), 'D'); // a worn spot
	return toRows(g);
}

/** packed earth or a road: dust, cart ruts and pebbles */
function earth(v: number): Rows {
	const g = grid('F');
	for (let y = 0; y < PH; y++)
		for (let x = 0; x < PW; x++) {
			const n = hx(x, y, v) % 19;
			if (n < 2) g[y][x] = 'd';
			else if (n === 2) g[y][x] = 'l';
		}
	for (const rx of [2, 9]) for (let y = 0; y < PH; y++) if (hx(rx, y, v) % 4) put(g, rx, y, 'd');
	for (let k = 0; k < 3; k++) {
		const n = hx(v, k, 31);
		const x = n % 11, y = (n >> 4) % 19;
		put(g, x, y, 'g');
		put(g, x + 1, y, 'G');
		put(g, x, y + 1, 'D');
	}
	return toRows(g);
}

// ------------------------------------------------------------------------------- doors, gates, stairs

function doorFrame(g: Grid): void {
	for (let x = 0; x < PW; x++) {
		g[0][x] = 'w';
		g[1][x] = 'W';
	}
	for (let y = 2; y < PH; y++) {
		g[y][0] = 'w';
		g[y][11] = 'W';
	}
}

function doorClosed(locked: boolean, v: number): Rows {
	const g = grid();
	doorFrame(g);
	for (let y = 2; y < PH; y++)
		for (let x = 1; x <= 10; x++) {
			const p = (x - 1) % 3;
			let ch = p === 0 ? 'l' : p === 1 ? 'F' : 'd';
			if (y === 2 && p === 0) ch = 'L';
			if (hx(x, y, v) % 23 === 0) ch = darker(ch);
			g[y][x] = ch;
		}
	dots(g, [[1, 2, 'W'], [10, 2, 'W'], [1, 3, 'W'], [10, 3, 'W'], [2, 2, 'W'], [9, 2, 'W']]);
	for (const by of [5, 15])
		for (let x = 1; x <= 10; x++) {
			g[by][x] = 'm';
			g[by + 1][x] = 'M';
		}
	for (const by of [5, 15]) dots(g, [[2, by, 'x'], [9, by, 'x']]);
	for (let x = 1; x <= 10; x++) g[19][x] = 'o';
	if (locked) {
		dots(g, [[5, 9, 'm'], [6, 9, 'm'], [4, 10, 'M'], [7, 10, 'M'], [4, 11, 'y'], [5, 11, 'y'], [6, 11, 'y'], [7, 11, 'Y'],
			[4, 12, 'y'], [5, 12, 'z'], [6, 12, 'y'], [7, 12, 'Y'], [4, 13, 'Y'], [5, 13, 'Y'], [6, 13, 'Y'], [7, 13, 'Y']]);
	} else {
		dots(g, [[8, 10, 'y'], [9, 10, 'Y'], [8, 11, 'Y'], [9, 11, 'o']]);
	}
	return toRows(g);
}

function doorOpen(): Rows {
	const g = grid();
	doorFrame(g);
	for (let y = 2; y < PH; y++) for (let x = 1; x <= 10; x++) g[y][x] = 'o';
	for (let x = 3; x <= 10; x++) {
		g[17][x] = 'D';
		g[18][x] = 'D';
		g[19][x] = 'd';
	}
	for (let y = 2; y < PH; y++) {
		g[y][1] = 'l';
		g[y][2] = 'd';
	}
	for (const by of [5, 15]) dots(g, [[1, by, 'm'], [2, by, 'M'], [1, by + 1, 'M'], [2, by + 1, 'M']]);
	return toRows(g);
}

/** stone piers and an arch; `bars` drops an iron portcullis into the opening */
function gate(bars: boolean, v: number): Rows {
	const g = grid();
	// piers
	for (let y = 0; y < PH; y++)
		for (const [a, b] of [[0, 2], [9, 11]])
			for (let x = a; x <= b; x++) {
				let ch = 'F';
				if (x === a) ch = 'l';
				if (x === b) ch = 'd';
				if (y % 4 === 3) ch = 'D';
				if (y % 4 === 0 && x === a) ch = 'L';
				g[y][x] = ch;
			}
	// arch ring above the opening; open spans per row
	const span: Record<number, [number, number]> = { 2: [5, 6], 3: [4, 7] };
	for (let y = 0; y < 5; y++)
		for (let x = 3; x <= 8; x++) {
			const s = span[y];
			const open = y >= 4 || (s && x >= s[0] && x <= s[1]);
			if (!open) g[y][x] = y === 0 ? 'l' : (x + y) & 1 ? 'F' : 'd';
		}
	dots(g, [[5, 0, 'L'], [6, 0, 'L'], [5, 1, 'l'], [6, 1, 'd'], [4, 2, 'D'], [7, 2, 'D'], [3, 3, 'D'], [8, 3, 'D']]);
	// shadow under the arch
	for (let y = 2; y < 7; y++) for (let x = 3; x <= 8; x++) if (g[y][x] === '.') g[y][x] = y < 5 ? 'o' : (y === 5 ? 'o' : '.');
	for (let x = 3; x <= 8; x++) g[6][x] = (x & 1) ? 'o' : '.';
	if (bars) {
		for (let y = 2; y < PH; y++) for (let x = 3; x <= 8; x++) if (g[y][x] === '.') g[y][x] = 'o';
		for (let y = 3; y < 19; y++) for (const bx of [3, 6]) {
			g[y][bx] = 'm';
			g[y][bx + 1] = 'M';
		}
		for (const hy of [8, 13]) for (let x = 3; x <= 8; x++) {
			g[hy][x] = 'm';
			g[hy + 1][x] = 'M';
		}
		for (const bx of [3, 6]) {
			g[19][bx] = 'M';
			g[19][bx + 1] = 'o';
		}
	}
	if (v % 4 === 0) put(g, 1, 9, 'D');
	return toRows(g);
}

function stairsDown(): Rows {
	const g = grid();
	for (let y = 7; y < PH; y++) for (let x = 0; x < PW; x++) g[y][x] = 'o';
	for (let x = 1; x <= 10; x++) g[8][x] = x === 1 ? 'L' : x === 10 ? 'l' : 'L';
	const tread = ['l', 'F', 'd', 'D', 'o'];
	for (let k = 0; k < 5; k++) {
		const y = 9 + k * 2;
		for (let x = 2; x <= 9; x++) {
			g[y][x] = tread[k];
			if (y + 1 < PH) g[y + 1][x] = darker(tread[k]);
		}
	}
	for (let y = 9; y < 19; y++) {
		g[y][1] = 'd';
		g[y][10] = 'D';
	}
	for (let x = 2; x <= 9; x++) g[18][x] = 'z';
	return toRows(g);
}

function stairsUp(): Rows {
	const g = grid();
	const LEFT = [3, 3, 2, 2, 1];
	for (let k = 0; k < 5; k++) {
		const y = 4 + k * 3, l = LEFT[k], r = 11 - l;
		for (let x = l; x <= r; x++) {
			g[y][x] = x === l ? 'L' : x === r ? 'F' : 'l';
			g[y + 1][x] = x === r ? 'd' : 'F';
			g[y + 2][x] = x === r ? 'D' : 'd';
		}
		put(g, l - 1, y, 'o');
		put(g, l - 1, y + 1, 'o');
		put(g, l - 1, y + 2, 'o');
		put(g, r + 1, y, 'o');
		put(g, r + 1, y + 1, 'o');
		put(g, r + 1, y + 2, 'o');
	}
	for (let x = 3; x <= 8; x++) {
		g[2][x] = 'o';
		g[3][x] = x === 3 || x === 8 ? 'o' : 'z';
	}
	for (let x = 0; x < PW; x++) g[19][x] = 'o';
	return toRows(g);
}

// ------------------------------------------------------------------------------- fences and roofs

function fence(g0: string, v: number, nm: number): Rows {
	const g = grid();
	const vertical = nm !== 0 && (nm & (N | S)) !== 0 && (nm & (E | W)) === 0;
	if (g0 === '‡') {
		// palisade of sharpened logs
		for (let k = 0; k < 4; k++) {
			const x0 = k * 3, top = 2 + (hx(v, k) % 2);
			put(g, x0 + 1, top, 'L');
			for (let y = top + 1; y < PH; y++) {
				g[y][x0] = 'l';
				g[y][x0 + 1] = 'F';
				g[y][x0 + 2] = 'd';
			}
			put(g, x0, top + 1, 'o');
			put(g, x0 + 2, top + 1, 'o');
		}
		for (let x = 0; x < PW; x++) {
			g[8][x] = x % 3 === 1 ? 'w' : 'W';
			g[19][x] = 'o';
		}
		return toRows(g);
	}
	if (vertical) {
		for (let y = 2; y < PH; y++) {
			g[y][4] = 'o';
			g[y][5] = 'l';
			g[y][6] = 'd';
			g[y][7] = 'o';
		}
		dots(g, [[5, 1, 'L'], [6, 1, 'o'], [5, 0, 'o']]);
		for (const ry of [6, 12]) dots(g, [[3, ry, 'o'], [8, ry, 'o'], [3, ry + 1, 'D'], [8, ry + 1, 'D']]);
		return toRows(g);
	}
	// rails across the whole tile so a row of fences joins up
	for (const ry of [9, 14])
		for (let x = 0; x < PW; x++) {
			g[ry - 1][x] = 'o';
			g[ry][x] = 'l';
			g[ry + 1][x] = 'd';
			g[ry + 2][x] = 'o';
		}
	for (const px of [2, 8]) {
		for (let y = 6; y < PH; y++) {
			g[y][px - 1] = 'o';
			g[y][px] = 'l';
			g[y][px + 1] = 'd';
			g[y][px + 2] = 'o';
		}
		dots(g, [[px, 5, 'L'], [px + 1, 5, 'o'], [px, 4, 'o']]);
	}
	if (g0 === '╪') {
		// a gate: diagonal brace and an iron latch between the posts
		for (let i = 0; i < 4; i++) dots(g, [[4 + i, 15 - i * 2, 'l'], [4 + i, 16 - i * 2, 'd']]);
		dots(g, [[7, 11, 'm'], [7, 12, 'M']]);
	}
	return toRows(g);
}

function roof(v: number): Rows {
	const g = grid();
	for (let y = 0; y < PH; y++) {
		const r = y & 3, c = y >> 2, off = c & 1 ? 2 : 0;
		for (let x = 0; x < PW; x++) {
			const bx = (x + off) & 3;
			let ch: string;
			if (r === 3) ch = bx === 1 || bx === 2 ? 'd' : 'D';
			else if (bx === 0) ch = r === 2 ? 'd' : 'D';
			else if (r === 0) ch = 'l';
			else ch = bx === 3 ? 'd' : bx === 1 && r === 1 ? 'l' : 'F';
			g[y][x] = ch;
		}
	}
	const n = hx(v, 1);
	if (n % 3 === 0) put(g, 1 + (n >> 3) % 10, 1 + ((n >> 7) % 4) * 4, 'D'); // a slipped tile
	return toRows(g);
}

// ------------------------------------------------------------------------------- fire

const WOB = [0, 1, 0, -1, 1, 0, -1, 0];
/** a pixel flame between columns x0..x1, standing on baseY; f = animation frame 0..7 */
function flame(g: Grid, x0: number, x1: number, baseY: number, maxH: number, f: number, seed = 0): void {
	const c = (x0 + x1) / 2, half = (x1 - x0) / 2 + 0.5;
	for (let x = x0; x <= x1; x++) {
		const d = Math.abs(x - c) / half;
		let H = Math.round(maxH * (1 - d * d * 0.85)) + WOB[(x * 5 + f * 3 + seed) & 7];
		if (x === x0 || x === x1) H = Math.min(H, Math.round(maxH * 0.4));
		for (let t = 0; t < H; t++) {
			const ad = Math.abs(x - c);
			const edge = x === x0 || x === x1 || t >= H - 1;
			const inner = ad <= half * 0.5 && t < H * 0.65;
			const core = ad <= 1 && t > 0 && t < H * 0.35;
			put(g, x, baseY - t, core ? 'x' : inner ? 'y' : edge ? (t >= H - 1 && t > 2 ? 'r' : 'R') : 'n');
		}
	}
	const sx = x0 + ((f * 3 + seed) % (x1 - x0 + 1));
	put(g, sx, baseY - maxH - 1 - (f & 1), f & 2 ? 'y' : 'n');
}

function fire(v: number, f: number): Rows {
	const g = grid();
	flame(g, 1, 10, 17, 12, f, v);
	for (let x = 0; x < PW; x++) {
		g[18][x] = (x + f) % 3 === 0 ? 'n' : 'R';
		g[19][x] = (x + v) % 4 === 0 ? 'R' : 'D';
	}
	return toRows(g);
}

// ------------------------------------------------------------------------------- hand-drawn props

const STATUE = art(
	'............',
	'....oooo....',
	'...oLlFdo...',
	'...olFFdo...',
	'...olFddo...',
	'....oddo....',
	'...oLlFdo...',
	'..oLlFFFdo..',
	'..olLLLldo..',
	'..olFFFFdo..',
	'..olFFFFdo..',
	'..olFFFddo..',
	'.olFFFFFddo.',
	'.olFFFFFdDo.',
	'oooooooooooo',
	'oLLlllllllDo',
	'olddddddddDo',
	'olFFFFFFFFDo',
	'oDDDDDDDDDDo',
	'............'
);

const OBELISK = art(
	'.....oo.....',
	'....oLdo....',
	'...oLlFdo...',
	'...olFFdo...',
	'...olFFdo...',
	'...olDDdo...',
	'...olFDdo...',
	'...olDDdo...',
	'...olFFdo...',
	'...olFFdo...',
	'...olFFdo...',
	'...olFFdo...',
	'..ooLlFdoo..',
	'..oLllllDo..',
	'..odddddDo..',
	'.oLlllllldo.',
	'.odddddddDo.',
	'oLllllllllDo',
	'odddddddddDo',
	'oooooooooooo'
);

const PILLAR = art(
	'oooooooooooo',
	'oLLLLLLLLLlo',
	'oddddddddDDo',
	'.oLlllllldo.',
	'..oLlFFdDo..',
	'..olLdFdDo..',
	'..olLdFdDo..',
	'..olLdFdDo..',
	'..olLdFdDo..',
	'..olLdFdDo..',
	'..olLdFdDo..',
	'..olLdFdDo..',
	'..olLdFdDo..',
	'..olLdFdDo..',
	'..olLdFdDo..',
	'..olLdFdDo..',
	'.oLlllllldo.',
	'oLllllllllDo',
	'odddddddddDo',
	'oooooooooooo'
);

const CORAL_DOME = art(
	'............',
	'............',
	'............',
	'............',
	'............',
	'............',
	'....oooo....',
	'..ooLlFFoo..',
	'.oLlFxFFddo.',
	'.oLFFFFdxdo.',
	'oLlFxFFFFdDo',
	'olFFFFdFxdDo',
	'olFxFFFFddDo',
	'odFFFFxFdDDo',
	'odddFFFdddDo',
	'oDdDdoodDdDo',
	'oDDDDoodDDDo',
	'oooooooooooo',
	'............',
	'............'
);

const ALTAR = art(
	'............',
	'............',
	'............',
	'............',
	'............',
	'..y......y..',
	'..n......n..',
	'.oxo....oxo.',
	'.oxo....oxo.',
	'.oto....oto.',
	'oLLLLLLLLLLo',
	'olllllllllDo',
	'oddddddddDDo',
	'.olFrrrRFdo.',
	'.olFryrRFdo.',
	'.olFrrrRFdo.',
	'.olFyyyYFdo.',
	'.olFFFFFFdo.',
	'oLllllllldDo',
	'oooooooooooo'
);

const STALL = art(
	'............',
	'............',
	'oooooooooooo',
	'olxxllxxllxo',
	'oFxxFFxxFFxo',
	'oFxxFFxxFFxo',
	'odmmddmmddmo',
	'.dm.dm.dm.d.',
	'.w........W.',
	'.w........W.',
	'.w........W.',
	'.w.rR.eE.yW.',
	'.wrrRReEyYW.',
	'otttttttttto',
	'owWwwWwwWwWo',
	'owWwwWwwWwWo',
	'owWwwWwwWwWo',
	'oWWWWWWWWWWo',
	'oooooooooooo',
	'............'
);

const ANVIL = art(
	'............',
	'............',
	'............',
	'............',
	'............',
	'............',
	'............',
	'............',
	'...oooooooo.',
	'.ooLLLLLLLlo',
	'oLlllllllldo',
	'..ooFFFFdo..',
	'....oFFdo...',
	'...oFFFddo..',
	'..oLlFFFdDo.',
	'.owwwwwwwWWo',
	'.oWwwwwwwWWo',
	'.owwwwwwwWWo',
	'.oooooooooo.',
	'............'
);

const SARCOPHAGUS = art(
	'............',
	'............',
	'............',
	'............',
	'............',
	'............',
	'.oooooooooo.',
	'oLlllllllldo',
	'olFFFLlFFFdo',
	'olFFFldFFFdo',
	'olFFLllFFFdo',
	'olFFlLLdFFdo',
	'olFFlFFdFFdo',
	'olFFlFFdFFdo',
	'olFFFlddFFdo',
	'oddddddddDDo',
	'odFdddFddDDo',
	'odddddddddDo',
	'oooooooooooo',
	'............'
);

/** a stele: grey stone, the carved words in the tile's colour (two layouts by hash) */
function stele(v: number): Rows {
	const body = v & 1
		? ['..omFFgFGo..', '..omggggGo..', '..omFgFFGo..', '..omggggGo..', '..omFFFgGo..', '..omggggGo..', '..omgFgFGo..', '..omggggGo..', '..omFFgFGo..']
		: ['..omgFFgGo..', '..omggggGo..', '..omFFFFGo..', '..omggggGo..', '..omFgFFGo..', '..omggggGo..', '..omFFgFGo..', '..omggggGo..', '..omgFFgGo..'];
	return art(
		'............',
		'............',
		'............',
		'....oooo....',
		'...omggGo...',
		'..omggggGo..',
		...body,
		'..omggggGo..',
		'..omgggGGo..',
		'.omggggggGo.',
		'.oGGGGGGGGo.',
		'.oooooooooo.'
	);
}

const BED = art(
	'............',
	'............',
	'............',
	'............',
	'.oooooooooo.',
	'.owwwwwwwWo.',
	'.oWWWWWWWWo.',
	'oWoxxxxxxoWo',
	'oWmxxxxxxmWo',
	'oWlLLLLLLlWo',
	'oWFFFFFFFdWo',
	'oWFFlFFlFdWo',
	'oWFFFFFFFdWo',
	'oWFFlFFlFdWo',
	'oWFFFFFFFdWo',
	'oWddddddddWo',
	'owwwwwwwwwWo',
	'oWWWWWWWWWWo',
	'oWo......oWo',
	'oo........oo'
);

const WELL = art(
	'............',
	'.....oo.....',
	'....owWo....',
	'...owwwWo...',
	'..owwwwwWo..',
	'.oWWWWWWWWo.',
	'..w......W..',
	'..wwwwwwwWm.',
	'..w...t..W..',
	'..w...t..W..',
	'..w..mMM.W..',
	'..w..MMG.W..',
	'.ommmmmmmmo.',
	'omgFFFFFFgGo',
	'omgdLFFFdgGo',
	'omggGggGggGo',
	'oGgggGgggGGo',
	'ogGggggGggGo',
	'.oooooooooo.',
	'............'
);

const THRONE = art(
	'............',
	'............',
	'.oo......oo.',
	'.oLo....oLo.',
	'.olooooooDo.',
	'.olLlllllDo.',
	'.olrrrrrRDo.',
	'.olrryrrRDo.',
	'.olrrrrrRDo.',
	'.olrryrrRDo.',
	'.olrrrrrRDo.',
	'.olRRRRRRDo.',
	'ooLlllllldoo',
	'oLlrrrrrRdDo',
	'olFFFFFFFFDo',
	'oddddddddDDo',
	'olDo....olDo',
	'olDo....olDo',
	'olDo....olDo',
	'oooo....oooo'
);

const DUMMY = art(
	'............',
	'....oooo....',
	'...otttTo...',
	'...otrtTo...',
	'...otttTo...',
	'....otTo....',
	'oooootTooooo',
	'owwwwwwwwwWo',
	'ooootttToooo',
	'...ottttTo..',
	'...otWWtTo..',
	'...ottytTo..',
	'...ottttTo..',
	'...otttTTo..',
	'....oywyo...',
	'.....wW.....',
	'.....wW.....',
	'....owWo....',
	'..oowwWWoo..',
	'..oooooooo..'
);

const CHANDELIER = art(
	'.....Mm.....',
	'.....Mm.....',
	'.....Mm.....',
	'.....Mm.....',
	'.....Mm.....',
	'.n.n.Mm.n.n.',
	'.y.y.Mm.y.y.',
	'.x.x.Mm.x.x.',
	'.x.x.Mm.x.x.',
	'ommmmyymmmMo',
	'.oMMMMMMMMo.',
	'...oyMMyo...',
	'....oyyo....',
	'.....oo.....',
	'............',
	'............',
	'............',
	'............',
	'............',
	'............'
);

const LANTERN = art(
	'.....W......',
	'.....W......',
	'.....W......',
	'.....W......',
	'....oWWo....',
	'...oWWWWo...',
	'..oFlLLlFo..',
	'..oddddDDo..',
	'..oFLxxLFo..',
	'..oFLxxLdo..',
	'..oddddDDo..',
	'..oFlLLldo..',
	'...oWWWWo...',
	'....oWWo....',
	'.....rR.....',
	'.....rR.....',
	'......R.....',
	'............',
	'............',
	'............'
);

const TRAP = art(
	'............',
	'............',
	'............',
	'............',
	'............',
	'............',
	'............',
	'............',
	'............',
	'............',
	'............',
	'.x..x..x..x.',
	'.m..m..m..m.',
	'.M..M..M..M.',
	'oLllllllllDo',
	'oFdFdFdFdFDo',
	'oddddddddDDo',
	'.oooooooooo.',
	'............',
	'............'
);

const MACHINE = art(
	'............',
	'............',
	'..mM....mM..',
	'..mM....mM..',
	'oooooooooooo',
	'oLlllllllldo',
	'olFFFFFFFFdo',
	'olFxxxFDFDdo',
	'olFxzxFFFFdo',
	'olFxxxFDFDdo',
	'olFFFFFFFFdo',
	'olDDDDDDDDdo',
	'olFDFDFDFDdo',
	'olFDFDFDFDdo',
	'olFFFFFFFFdo',
	'olFMmmmmMFdo',
	'oddddddddDDo',
	'oMMo....oMMo',
	'oooo....oooo',
	'............'
);

const BRAZIER_BASE = art(
	'............',
	'............',
	'............',
	'............',
	'............',
	'............',
	'............',
	'............',
	'............',
	'..RnRRnRnR..',
	'ommmmmmmmmMo',
	'.oMmmmmmMGo.',
	'..oMMMMMGo..',
	'...oGMMGo...',
	'....oMGo....',
	'...oMooMo...',
	'..oMo..oMo..',
	'.oMo....oMo.',
	'.oo......oo.',
	'............'
);

const BALLISTA = art(
	'.....oo.....',
	'....oxmo....',
	'....omMo....',
	'.....mM.....',
	'oo...mM...oo',
	'oWoooWwoooWo',
	'.oWWWwwWWWo.',
	'..oWtwwtWo..',
	'...otwwto...',
	'....twwt....',
	'.....ww.....',
	'....owWo....',
	'....owWo....',
	'..oowwWWoo..',
	'.owwwwwwwWWo',
	'.oWWWWWWWWWo',
	'.oWo.oo.oWo.',
	'oMMMo..oMMMo',
	'oGMGo..oGMGo',
	'.ooo....ooo.'
);

// ------------------------------------------------------------------------------- procedural props

/** a 3/4-view crate: lid face on top, plank front with a frame and a diagonal brace */
function crate(g: Grid, x0: number, y0: number, w: number, h: number): void {
	const x1 = x0 + w - 1, y1 = y0 + h - 1;
	for (let y = y0; y <= y1; y++)
		for (let x = x0; x <= x1; x++) {
			let ch: string;
			if (x === x0 || x === x1 || y === y0 || y === y1) ch = 'o';
			else if (y <= y0 + 2) ch = y === y0 + 1 ? (x === x0 + 1 ? 'L' : 'l') : 'F';
			else if (y === y0 + 3) ch = 'D';
			else if (x === x0 + 1 || y === y0 + 4) ch = 'l';
			else if (x === x1 - 1 || y === y1 - 1) ch = 'd';
			else ch = 'F';
			put(g, x, y, ch);
		}
	// brace across the front
	const fy0 = y0 + 5, fy1 = y1 - 2, fx0 = x0 + 2, fx1 = x1 - 2;
	const n = Math.max(1, fy1 - fy0);
	for (let i = 0; i <= n; i++) put(g, fx0 + Math.round(((fx1 - fx0) * i) / n), fy0 + i, 'd');
	put(g, x0 + 1, y0 + 4, 'D');
	put(g, x1 - 1, y0 + 4, 'D');
}

/** a cask with staves shaded left to right and iron hoops */
function barrel(g: Grid, x0: number, y0: number, w: number, h: number): void {
	const x1 = x0 + w - 1, y1 = y0 + h - 1;
	const STAVE = ['l', 'L', 'l', 'F', 'F', 'F', 'd', 'd', 'D'];
	for (let y = y0; y <= y1; y++) {
		const bulge = y > y0 + 1 && y < y1 - 1 ? 0 : 1;
		const a = x0 + bulge, b = x1 - bulge;
		for (let x = a; x <= b; x++) {
			let ch: string;
			if (x === a || x === b || y === y0 || y === y1) ch = 'o';
			else if (y <= y0 + 2) ch = y === y0 + 1 ? 'F' : 'd'; // the lid seen from above
			else ch = STAVE[Math.min(STAVE.length - 1, Math.floor(((x - a) * STAVE.length) / (b - a + 1)))];
			put(g, x, y, ch);
		}
	}
	put(g, x0 + 3, y0 + 1, 'l');
	for (const hy of [y0 + 3, Math.round((y0 + y1) / 2), y1 - 2])
		for (let x = x0 + 1; x < x1; x++) if (g[hy][x] !== 'o' && g[hy][x] !== '.') g[hy][x] = x < x0 + 3 ? 'm' : 'M';
}

function crates(v: number): Rows {
	const g = grid();
	switch (v & 3) {
		case 0:
			crate(g, 0, 10, 11, 9);
			crate(g, 2, 3, 8, 8);
			break;
		case 1:
			crate(g, 0, 8, 7, 11);
			barrel(g, 6, 9, 6, 10);
			break;
		case 2:
			crate(g, 0, 11, 6, 8);
			crate(g, 6, 9, 6, 10);
			dots(g, [[2, 9, 'o'], [3, 9, 'o'], [1, 10, 'o'], [2, 10, 't'], [3, 10, 'T'], [4, 10, 'o']]);
			break;
		default:
			crate(g, 1, 6, 10, 13);
	}
	return toRows(g);
}

function cask(): Rows {
	const g = grid();
	barrel(g, 1, 4, 10, 15);
	dots(g, [[5, 13, 'm'], [6, 13, 'M'], [6, 14, 'M']]);
	return toRows(g);
}

const BOOKS = ['rR', 'uU', 'eE', 'yY', 'pz', 'nR', 'wW', 'xg', 'RD', 'UD'];
function bookshelf(v: number): Rows {
	const g = grid('o');
	for (let x = 0; x < PW; x++) {
		g[0][x] = 'L';
		g[1][x] = 'd';
	}
	for (let y = 0; y < PH; y++) {
		g[y][0] = y === 0 ? 'L' : 'l';
		g[y][11] = 'D';
	}
	for (const sy of [7, 13, 19])
		for (let x = 1; x < 11; x++) g[sy][x] = sy === 19 ? 'D' : 'F';
	for (const [top, bot] of [[2, 6], [8, 12], [14, 18]]) {
		let x = 1;
		while (x < 11) {
			const n = hx(v, top, x);
			const wdt = n % 3 === 0 ? 2 : 1;
			if (n % 11 === 0) {
				x += 1; // a gap
				continue;
			}
			const col = BOOKS[(n >> 3) % BOOKS.length];
			const hgt = 3 + ((n >> 7) % 3);
			for (let i = 0; i < wdt && x + i < 11; i++)
				for (let y = bot - hgt + 1; y <= bot; y++) g[y][x + i] = wdt === 2 && i === 1 ? col[1] : col[0];
			if (hgt >= 4 && wdt === 2) g[bot - 1][x] = 'y';
			x += wdt;
		}
	}
	return toRows(g);
}

function stands(): Rows {
	const g = grid();
	const TIER = ['L', 'l', 'F', 'd', 'd', 'D'];
	for (let k = 0; k < 3; k++)
		for (let r = 0; r < 6; r++)
			for (let x = 0; x < PW; x++) {
				let ch = TIER[r];
				if (r >= 3 && x % 6 === 5) ch = 'o';
				g[1 + k * 6 + r][x] = ch;
			}
	for (let x = 0; x < PW; x++) g[19][x] = 'o';
	return toRows(g);
}

function barricade(): Rows {
	const g = grid();
	const diag = (flip: boolean) => {
		for (let t = 0; t <= 14; t++) {
			const y = 4 + t, x = 1 + Math.round((t * 8) / 14);
			const px = flip ? 10 - x : x;
			put(g, px - 1, y, 'o');
			put(g, px, y, 'l');
			put(g, px + 1, y, 'd');
			put(g, px + 2, y, 'o');
		}
	};
	diag(true);
	diag(false);
	for (let x = 0; x < PW; x++) {
		g[10][x] = 'o';
		g[11][x] = 'l';
		g[12][x] = 'd';
		g[13][x] = 'o';
	}
	dots(g, [[1, 3, 'm'], [1, 2, 'x'], [10, 3, 'm'], [10, 2, 'x'], [0, 3, 'o'], [2, 3, 'o'], [9, 3, 'o'], [11, 3, 'o']]);
	dots(g, [[3, 11, 'M'], [8, 11, 'M']]);
	return toRows(g);
}

const FLAGS_C: [string, string][] = [['r', 'R'], ['y', 'Y'], ['u', 'U'], ['e', 'E'], ['x', 'g'], ['q', 'R'], ['n', 'R']];
function bunting(v: number, f: number): Rows {
	const g = grid();
	const ROPE = [3, 3, 4, 4, 5, 5, 5, 5, 4, 4, 3, 3];
	for (let x = 0; x < PW; x++) put(g, x, ROPE[x], 'W');
	const sway = f >= 4 ? 1 : 0;
	[1, 5, 9].forEach((px, i) => {
		const [c, d] = FLAGS_C[(v + i * 3) % FLAGS_C.length];
		const y = Math.max(ROPE[px], ROPE[px + 1], ROPE[px + 2]) + 1;
		put(g, px, y, c);
		put(g, px + 1, y, c);
		put(g, px + 2, y, d);
		put(g, px + sway, y + 1, c);
		put(g, px + 1 + sway, y + 1, d);
		put(g, px + 1 + sway, y + 2, d);
	});
	return toRows(g);
}

function chimney(f: number): Rows {
	const g = grid();
	for (let y = 7; y < PH; y++) {
		const r = (y - 7) % 3, c = Math.floor((y - 7) / 3), off = c & 1 ? 2 : 0;
		for (let x = 2; x <= 9; x++) {
			const bx = (x + off) % 4;
			let ch = r === 2 || bx === 3 ? 'D' : r === 0 ? 'l' : 'F';
			if (x === 2 && ch !== 'D') ch = 'l';
			if (x === 9) ch = ch === 'D' ? 'o' : 'd';
			g[y][x] = ch;
		}
		g[y][1] = 'o';
		g[y][10] = 'o';
	}
	for (let x = 1; x <= 10; x++) {
		g[4][x] = x === 1 || x === 10 ? 'o' : 'o';
		g[5][x] = x === 10 ? 'd' : 'l';
		g[6][x] = 'D';
	}
	g[5][1] = 'L';
	for (let x = 3; x <= 8; x++) g[4][x] = 'z';
	// smoke puffs climbing with the frame
	for (let k = 0; k < 2; k++) {
		const t = (f + k * 4) & 7, y = 3 - (t >> 1), x = 4 + ((t >> 1) & 1) + k;
		if (y < 0) continue;
		put(g, x, y, 'g');
		put(g, x + 1, y, 'G');
		if (t > 3) put(g, x, y - 1 < 0 ? y : y - 1, 'G');
	}
	return toRows(g);
}

function skylight(): Rows {
	const g = grid();
	for (let y = 2; y <= 17; y++)
		for (let x = 0; x < PW; x++) {
			const d = Math.abs(x - 5.5) / 6 + Math.abs(y - 9.5) / 8;
			if (d > 1) continue;
			let ch: string;
			if (d > 0.8) ch = d > 0.92 ? 'o' : 'W';
			else if (x === 5 || x === 6 || y === 9 || y === 10) ch = 'w';
			else ch = x + y < 14 ? 'l' : 'F';
			g[y][x] = ch;
		}
	dots(g, [[3, 7, 'L'], [4, 6, 'L'], [8, 12, 'd'], [7, 13, 'd']]);
	return toRows(g);
}

function rift(f: number): Rows {
	const g = grid();
	for (let y = 1; y <= 18; y++) {
		const w = Math.round(3.2 * Math.sin((Math.PI * (y - 0.5)) / 18.5)) + ((y + f) % 4 === 0 ? 1 : 0);
		const sh = (y * 3 + f) % 5 === 0 ? 1 : (y * 7 + f) % 6 === 0 ? -1 : 0;
		const a = 6 - w + sh, b = 5 + w + sh;
		for (let x = a; x <= b; x++) {
			const e = Math.min(x - a, b - x);
			put(g, x, y, e === 0 ? 'F' : e === 1 ? 'l' : 'L');
		}
		if (w >= 2) put(g, 5 + sh + ((y + f) & 1), y, (y + f) % 3 ? 'x' : 'L');
	}
	for (let k = 0; k < 3; k++) {
		const n = hx(f, k, 77);
		put(g, n % 12, 1 + (n >> 4) % 18, k & 1 ? 'q' : 'p');
	}
	return toRows(g);
}

function portal(f: number): Rows {
	const g = grid();
	const cx = 5.5, cy = 9.5;
	const SW = ['L', 'l', 'F', 'd'];
	for (let y = 0; y < PH; y++)
		for (let x = 0; x < PW; x++) {
			const dx = (x - cx) / 6, dy = (y - cy) / 10, d = dx * dx + dy * dy;
			if (d > 1) continue;
			if (d > 0.82) g[y][x] = 'o';
			else if (d > 0.5) g[y][x] = x + y < 11 ? 'm' : x + y > 19 ? 'G' : 'g';
			else {
				const ang = Math.atan2(y - cy, (x - cx) * 1.7);
				const val = Math.floor((ang / (2 * Math.PI)) * 6 + Math.sqrt(d) * 5 - f * 0.75);
				g[y][x] = d < 0.04 ? 'x' : SW[((val % 4) + 4) % 4];
			}
		}
	// runes on the ring, pulsing
	const rune = f & 4 ? 'L' : 'F';
	dots(g, [[5, 1, rune], [1, 9, rune], [10, 10, rune], [6, 18, rune]]);
	for (let x = 2; x <= 9; x++) put(g, x, 19, 'o');
	return toRows(g);
}

function silkRack(f: number): Rows {
	const g = grid();
	for (let y = 3; y < 19; y++) {
		g[y][1] = 'w';
		g[y][10] = 'W';
		g[y][0] = 'o';
		g[y][11] = 'o';
	}
	for (const ty of [5, 10, 15]) {
		for (let x = 1; x <= 10; x++) {
			g[ty][x] = x === 1 || x === 10 ? g[ty][x] : 'F';
			g[ty + 1][x] = 'W';
			g[ty + 2][x] = x === 1 || x === 10 ? g[ty + 2][x] : 'o';
		}
		for (let x = 2; x <= 9; x += 2) put(g, x + ((ty >> 2) & 1), ty, (x + f) % 4 === 0 ? 'x' : 'L');
		put(g, 2, ty - 1, 'l');
		put(g, 6, ty - 1, 'l');
	}
	for (let x = 0; x < PW; x++) {
		g[2][x] = x === 0 || x === 11 ? 'o' : 'w';
		g[19][x] = x === 0 || x === 11 ? '.' : 'o';
	}
	return toRows(g);
}

function lanternSwing(f: number): Rows {
	const SW = [0, 0, 0, 1, 0, 0, 0, -1][f & 7];
	const g = grid();
	const src = fromRows(LANTERN);
	for (let y = 0; y < PH; y++)
		for (let x = 0; x < PW; x++) {
			let c = src[y][x];
			if (c === '.') continue;
			if (f & 1) c = c === 'x' ? 'L' : c;
			put(g, y >= 4 ? x + SW : x, y, c);
		}
	return toRows(g);
}

function withFlicker(base: Rows, f: number): Rows {
	const g = fromRows(base);
	for (let y = 0; y < PH; y++)
		for (let x = 0; x < PW; x++) {
			const c = g[y][x];
			const on = ((x + f) & 1) === 0;
			if (c === 'n' && y < 7) g[y][x] = on ? 'n' : '.';
			if (c === 'y' && y < 7) g[y][x] = on ? 'y' : 'n';
		}
	return toRows(g);
}

function domeGlow(f: number): Rows {
	if (!(f & 4)) return CORAL_DOME;
	return CORAL_DOME.map((r) => r.replace(/x/g, 'L'));
}

function machine(v: number, f: number): Rows {
	const g = fromRows(MACHINE);
	const NEEDLE: [number, number][] = [[3, 7], [4, 7], [5, 7], [5, 8], [5, 9], [5, 8], [4, 7], [3, 8]];
	const [nx, ny] = NEEDLE[(f + v) & 7];
	put(g, nx, ny, 'z');
	const LIT = ['r', 'v', 'y', 'c'];
	put(g, 7, 7, f & 2 ? LIT[v & 3] : 'D');
	put(g, 9, 7, f & 4 ? 'v' : 'D');
	put(g, 7, 9, (f + 1) & 2 ? 'y' : 'D');
	put(g, 9, 9, f % 3 === 0 ? 'r' : 'D');
	if (f & 1) {
		put(g, 2 + (f & 2 ? 1 : 0), 1, 'g');
		put(g, 8, 0, 'G');
	}
	return toRows(g);
}

function brazier(v: number, f: number): Rows {
	const g = fromRows(BRAZIER_BASE);
	flame(g, 2, 9, 8, 7, f, v);
	return toRows(g);
}

function altar(f: number): Rows {
	const g = fromRows(ALTAR);
	const tips: [number, string][] = [[2, f & 1 ? 'y' : 'x'], [9, f & 2 ? 'x' : 'y']];
	for (const [x, c] of tips) {
		put(g, x, 5, c);
		put(g, x + (f & 4 ? 1 : 0), 4, f % 3 ? 'n' : '.');
	}
	return toRows(g);
}

function chandelier(f: number): Rows {
	return withFlicker(CHANDELIER, f);
}

function wellSprite(f: number): Rows {
	const g = fromRows(WELL);
	put(g, 4 + ((f >> 1) & 3), 14, 'L'); // glint moving on the water
	return toRows(g);
}

function pillar(v: number): Rows {
	if ((v & 3) !== 3) return PILLAR;
	const g = fromRows(PILLAR);
	dots(g, [[6, 8, 'o'], [5, 9, 'o'], [6, 10, 'o'], [6, 11, 'D']]);
	return toRows(g);
}

/** a casement window set in a stone surround: wood frame, cross mullion, glass in the tile's colour */
function window_(v: number): Rows {
	const g = grid('g');
	for (let y = 0; y < PH; y++)
		for (let x = 0; x < PW; x++) {
			const c = Math.floor(y / 5), off = c & 1 ? 3 : 0;
			if (y % 5 === 4 || (x + off) % 6 === 5) g[y][x] = 'G';
			else if (y % 5 === 0 || (x + off) % 6 === 0) g[y][x] = 'm';
		}
	// lintel, opening, sill
	for (let x = 1; x <= 11; x++) g[2][x] = 'G';
	for (let y = 3; y <= 14; y++)
		for (let x = 2; x <= 10; x++) {
			let ch: string;
			if (y === 3 || y === 14 || x === 2 || x === 10 || x === 6 || y === 8) ch = x === 2 || y === 3 ? 'w' : 'W';
			else {
				const px = x < 6 ? x - 3 : x - 7, py = y < 8 ? y - 4 : y - 9;
				ch = px === 0 && py === 0 ? 'L' : px + py <= 1 ? 'l' : px === 2 || py >= 3 ? 'd' : 'F';
			}
			g[y][x] = ch;
		}
	for (let x = 1; x <= 11; x++) {
		g[15][x] = x === 1 ? 'x' : 'm';
		g[16][x] = 'G';
	}
	if (v % 4 === 0) {
		// a flower box
		for (let x = 3; x <= 9; x++) g[14][x] = x & 1 ? 'r' : 'e';
	}
	return toRows(g);
}

// ------------------------------------------------------------------------------- dispatch

/** glyphs whose look changes with `frame` (they need repainting each animation tick) */
export const BUILD_ANIMATED = '^Ψ¤*~◊Ø∩Φ§╥oO';

const memo = new Map<string, Rows | null>();

function make(g: string, fg: number, bg: number, v: number, f: number, nm: number): Rows | null {
	switch (g) {
		case '#':
			return wall(fg, bg, v, nm);
		case '+':
			return doorClosed(fg === 11, v);
		case '/':
			return doorOpen();
		case '⊞':
			return window_(v);
		case '∏':
			return gate(false, v);
		case '╫':
			return gate(true, v);
		case '>':
			return stairsDown();
		case '<':
			return stairsUp();
		case '=':
			if (fg === 10) return BED;
			if (bg === 4) return deck(v);
			return planks(v);
		case '·':
			if (fg === 9) return tiles(v);
			if (fg === 1 || fg === 2) return cobbles(v);
			return null;
		case '.':
			if (fg === 10) return carpet(v);
			if (fg === 1 && bg === 18) return flagstones(v);
			return null;
		case '∙':
			return earth(v);
		case '|':
		case '╪':
		case '‡':
			return fence(g, v, nm);
		case 'Λ':
			return roof(v);
		case '^':
			return fg === 13 ? TRAP : fire(v, f);
		case 'o':
			return fg === 5 ? wellSprite(f) : null;
		case 'O':
			return fg === 9 && bg === 4 ? domeGlow(f) : pillar(v);
		case 'Ω':
			return fg === 15 ? OBELISK : STATUE;
		case '╥':
			return altar(f);
		case 'π':
			return fg === 2 ? ANVIL : STALL;
		case '▬':
			return SARCOPHAGUS;
		case '¶':
			return stele(v);
		case '◊':
			return fg === 9 ? skylight() : rift(f);
		case 'Ø':
			return portal(f);
		case 'Φ':
			return machine(v, f);
		case 'Ψ':
			return fg === 1 && bg === 20 ? BALLISTA : brazier(v, f);
		case 'Ж':
			return THRONE;
		case 'Ŧ':
			return DUMMY;
		case '■':
			return crates(v);
		case 'Θ':
			return cask();
		case '§':
			return silkRack(f);
		case '≡':
			return fg === 1 ? stands() : bookshelf(v);
		case 'X':
			return barricade();
		case '*':
			return fg === 15 ? lanternSwing(f) : null;
		case '~':
			return fg === 11 ? bunting(v, f) : null;
		case '¤':
			return fg === 15 ? chandelier(f) : null;
		case '∩':
			return chimney(f);
	}
	return null;
}

export function buildSprite(g: string, fgIdx: number, bgIdx: number, h: number, frame: number, nm: number): Rows | null {
	const v = h & 15, f = BUILD_ANIMATED.includes(g) ? frame & 7 : 0;
	const key = g + '|' + fgIdx + '|' + bgIdx + '|' + v + '|' + f + '|' + nm;
	const hit = memo.get(key);
	if (hit !== undefined) return hit;
	if (memo.size > 4000) memo.clear();
	const r = make(g, fgIdx, bgIdx, v, f, nm);
	memo.set(key, r);
	return r;
}
