/**
 * Procedural pixel icons for items and statuses (engine/iconinfo.go supplies the specs).
 * Each icon is drawn once into a 16x16 grid, outlined, and cached as a tiny offscreen canvas; drawIcon scales it
 * (nearest neighbour) and adds the rarity frame and glow. Deterministic: only the spec's hash varies the art.
 */

export interface IconSpec {
	/** silhouette */
	k: string;
	/** material palette */
	m?: string;
	/** rarity tier 0 common .. 6 unique */
	r: number;
	/** effect tag: cursed poisoned brewed enchanted fire frost shock acid, cooking tags, status class */
	tg?: string;
	/** heirloom level 0..4 (items) or severity (statuses) */
	lv?: number;
	/** stable hash of the item name */
	h: number;
	/** weapon family */
	f?: string;
	/** status label */
	n?: string;
}

export interface IconRowInfo {
	y: number;
	x0: number;
	x1: number;
	i: IconSpec;
	n: string;
	d?: string;
}

export interface IconStatusInfo {
	y: number;
	x0: number;
	icons: IconSpec[];
}

const N = 16;
type Grid = (string | null)[];

interface Pal {
	d: string;
	m: string;
	l: string;
}

const MATS: Record<string, Pal> = {
	iron: { d: '#33383f', m: '#6b727c', l: '#a3abb5' },
	steel: { d: '#3a4858', m: '#8497ac', l: '#d0dce8' },
	bronze: { d: '#5a3a14', m: '#a8702a', l: '#e0a850' },
	wood: { d: '#3e2a14', m: '#7a5430', l: '#b08450' },
	leather: { d: '#3a2616', m: '#7b4e2c', l: '#a87a4c' },
	cloth: { d: '#3a3f5c', m: '#6b72a8', l: '#a8afdc' },
	bone: { d: '#6b6450', m: '#c8bfa0', l: '#f0ead0' },
	glass: { d: '#2a5560', m: '#6cb4c4', l: '#c8f0f8' },
	crystal: { d: '#4a2a7a', m: '#9a6af0', l: '#e0c8ff' },
	gold: { d: '#7a5a08', m: '#e0b020', l: '#fff0a0' },
	silver: { d: '#50565e', m: '#b8c0c8', l: '#f4f8ff' },
	stone: { d: '#3a3a3e', m: '#76767c', l: '#aaaab0' },
	bio: { d: '#1e4a2a', m: '#4aa050', l: '#a0e090' },
	plasma: { d: '#7a1a5a', m: '#ff4ad0', l: '#ffd0f4' },
};

const OUTLINE = '#0b0e10';
export const RARITY_COLORS = ['#8a9096', '#4fc16a', '#4a8ef0', '#38d6e0', '#f09a30', '#d85ae8', '#ffe14a'];
const CLOTH_HUES: Pal[] = [
	{ d: '#3a3f5c', m: '#6b72a8', l: '#a8afdc' },
	{ d: '#5c2a2a', m: '#a84a4a', l: '#e09090' },
	{ d: '#2a4c36', m: '#4a9068', l: '#98d8b0' },
	{ d: '#5c4a1e', m: '#b0903a', l: '#ecd080' },
	{ d: '#4a2a5c', m: '#8a52a8', l: '#d0a0e8' },
	{ d: '#2a4a5c', m: '#4a88a8', l: '#98d0e8' },
];
const LIQUIDS: Pal[] = [
	{ d: '#7a1a1a', m: '#e03a3a', l: '#ff9a9a' },
	{ d: '#1a3a7a', m: '#3a7ae0', l: '#9ac0ff' },
	{ d: '#1a6a2a', m: '#3ae05a', l: '#9affb0' },
	{ d: '#6a4a0a', m: '#e0a82a', l: '#ffe08a' },
	{ d: '#5a1a7a', m: '#b03ae0', l: '#e8a0ff' },
	{ d: '#0a6a6a', m: '#2ae0d0', l: '#9afff0' },
];
const TAG_COLORS: Record<string, string> = {
	cursed: '#a040e0', poisoned: '#5ae050', brewed: '#30d8c0', enchanted: '#ff80ff', fire: '#ff7a20', frost: '#80d8ff',
	shock: '#ffee40', acid: '#b0ff30', poison: '#5ae050',
	hearty: '#e07a40', spry: '#90e050', thick: '#a08060', mending: '#50e090', keen: '#f0e060', ember: '#ff6030', rime: '#a0e0ff',
	deep: '#4070d0', lasting: '#d0a050', venom: '#80d030',
};

class Pix {
	g: Grid = new Array(N * N).fill(null);
	p(x: number, y: number, c: string) {
		x = Math.round(x);
		y = Math.round(y);
		if (x >= 0 && y >= 0 && x < N && y < N) this.g[y * N + x] = c;
	}
	r(x: number, y: number, w: number, h: number, c: string) {
		for (let j = 0; j < h; j++) for (let i = 0; i < w; i++) this.p(x + i, y + j, c);
	}
	l(x0: number, y0: number, x1: number, y1: number, c: string) {
		const dx = Math.abs(x1 - x0), dy = Math.abs(y1 - y0), sx = x0 < x1 ? 1 : -1, sy = y0 < y1 ? 1 : -1;
		let e = dx - dy;
		for (let n = 0; n < 64; n++) {
			this.p(x0, y0, c);
			if (x0 === x1 && y0 === y1) break;
			const e2 = 2 * e;
			if (e2 > -dy) {
				e -= dy;
				x0 += sx;
			}
			if (e2 < dx) {
				e += dx;
				y0 += sy;
			}
		}
	}
	/** filled ellipse */
	e(cx: number, cy: number, rx: number, ry: number, c: string) {
		for (let y = Math.floor(cy - ry); y <= Math.ceil(cy + ry); y++)
			for (let x = Math.floor(cx - rx); x <= Math.ceil(cx + rx); x++) {
				const dx = (x - cx) / (rx + 0.35), dy = (y - cy) / (ry + 0.35);
				if (dx * dx + dy * dy <= 1) this.p(x, y, c);
			}
	}
	d(cx: number, cy: number, r: number, c: string) {
		this.e(cx, cy, r, r, c);
	}
	ring(cx: number, cy: number, r: number, c: string) {
		for (let a = 0; a < 48; a++) this.p(cx + Math.cos((a / 48) * 6.2832) * r, cy + Math.sin((a / 48) * 6.2832) * r, c);
	}
	mirror() {
		const o = this.g.slice();
		for (let y = 0; y < N; y++) for (let x = 0; x < N; x++) this.g[y * N + x] = o[y * N + (N - 1 - x)];
	}
}

function pal(spec: IconSpec): Pal {
	if (spec.m === 'cloth') return CLOTH_HUES[spec.h % CLOTH_HUES.length];
	return MATS[spec.m ?? ''] ?? MATS.iron;
}

const WOOD = MATS.wood;
const GOLD = MATS.gold;

/** the silhouette for an item kind */
function item(q: Pix, spec: IconSpec) {
	const c = pal(spec);
	const hv = spec.h % 8;
	const liquid = LIQUIDS[hv % LIQUIDS.length];
	switch (spec.k) {
		case 'sword':
			q.l(5, 10, 12, 3, c.m);
			q.l(6, 10, 13, 3, c.l);
			q.l(5, 11, 12, 4, c.d);
			q.l(3, 9, 7, 13, GOLD.m);
			q.l(4, 12, 2, 14, WOOD.m);
			q.p(1, 14, GOLD.l);
			q.p(13, 2, c.l);
			break;
		case 'dagger':
			q.l(6, 9, 11, 4, c.m);
			q.l(7, 9, 12, 4, c.l);
			q.l(4, 8, 7, 11, GOLD.m);
			q.l(5, 11, 3, 13, WOOD.m);
			break;
		case 'axe':
			q.l(3, 13, 11, 5, WOOD.m);
			q.r(8, 2, 5, 2, c.m);
			q.r(9, 4, 6, 3, c.m);
			q.r(10, 7, 4, 1, c.m);
			q.r(14, 3, 1, 4, c.l);
			q.l(9, 3, 9, 6, c.d);
			break;
		case 'mace':
			q.l(3, 13, 10, 6, WOOD.m);
			q.d(11, 4, 3, c.m);
			q.p(11, 4, c.l);
			q.p(10, 3, c.l);
			q.p(11, 0, c.d);
			q.p(15, 4, c.d);
			q.p(8, 4, c.d);
			q.p(11, 8, c.d);
			break;
		case 'spear':
			q.l(2, 14, 11, 5, WOOD.m);
			q.l(11, 5, 13, 3, c.m);
			q.r(12, 3, 2, 2, c.m);
			q.p(14, 2, c.l);
			q.p(13, 2, c.l);
			q.p(14, 3, c.l);
			q.p(10, 6, GOLD.m);
			break;
		case 'staff':
			q.l(3, 14, 11, 4, WOOD.m);
			q.l(4, 14, 12, 4, WOOD.d);
			q.d(12, 3, 2, MATS.crystal.m);
			q.p(11, 2, MATS.crystal.l);
			break;
		case 'fist':
			q.r(4, 6, 8, 6, c.m);
			for (let i = 0; i < 4; i++) q.r(4 + i * 2, 4, 2, 3, c.l);
			q.r(3, 8, 2, 3, c.m);
			q.r(4, 12, 8, 2, c.d);
			break;
		case 'flail':
			q.l(2, 14, 7, 9, WOOD.m);
			q.p(8, 8, c.d);
			q.p(9, 6, c.d);
			q.p(10, 5, c.d);
			q.d(12, 4, 2, c.m);
			q.p(11, 3, c.l);
			q.p(12, 1, c.d);
			q.p(14, 4, c.d);
			break;
		case 'whip':
			q.r(2, 12, 3, 3, WOOD.m);
			for (const [x, y] of [[5, 11], [6, 10], [7, 9], [8, 9], [9, 10], [10, 11], [11, 10], [12, 8], [12, 6], [11, 4], [9, 3]]) q.p(x, y, c.m);
			break;
		case 'hook':
			q.l(3, 14, 10, 4, WOOD.m);
			q.p(11, 3, c.m);
			q.p(12, 3, c.m);
			q.p(13, 4, c.m);
			q.p(13, 5, c.m);
			q.p(12, 6, c.m);
			q.p(11, 5, c.l);
			break;
		case 'chakram':
			q.ring(8, 8, 5, c.m);
			q.ring(8, 8, 4, c.l);
			q.ring(8, 8, 3, c.m);
			q.p(8, 2, c.l);
			q.p(8, 14, c.d);
			break;
		case 'boomerang':
			q.l(3, 3, 5, 11, c.m);
			q.l(4, 3, 6, 11, c.m);
			q.l(5, 11, 13, 12, c.m);
			q.l(5, 10, 13, 11, c.l);
			break;
		case 'bolas':
			q.d(4, 11, 2, c.m);
			q.d(12, 11, 2, c.m);
			q.d(8, 4, 2, c.m);
			q.l(5, 10, 7, 6, WOOD.l);
			q.l(11, 10, 9, 6, WOOD.l);
			q.p(8, 11, WOOD.l);
			q.l(5, 11, 11, 11, WOOD.l);
			break;
		case 'sling':
			q.l(4, 13, 7, 8, WOOD.m);
			q.l(12, 13, 9, 8, WOOD.m);
			q.e(8, 6, 2, 1.5, WOOD.l);
			q.p(8, 5, c.m);
			q.p(8, 6, c.l);
			break;
		case 'blowgun':
			q.l(2, 13, 13, 3, c.m);
			q.l(2, 12, 13, 2, c.l);
			q.p(1, 14, c.d);
			q.p(14, 2, GOLD.l);
			break;
		case 'crossbow':
			q.r(7, 3, 2, 11, WOOD.m);
			q.l(2, 6, 7, 4, c.m);
			q.l(14, 6, 9, 4, c.m);
			q.l(2, 6, 7, 9, WOOD.l);
			q.l(14, 6, 9, 9, WOOD.l);
			q.r(7, 2, 2, 3, c.l);
			break;
		case 'bow':
			for (let y = 2; y <= 14; y++) q.p(5 + Math.round(Math.pow((y - 8) / 6, 2) * -3 + 3), y, WOOD.m);
			q.l(8, 2, 8, 14, WOOD.l);
			q.l(3, 8, 14, 8, WOOD.d);
			q.p(14, 8, c.l);
			q.p(13, 7, c.m);
			q.p(13, 9, c.m);
			break;
		case 'gun':
			q.r(2, 5, 11, 3, c.m);
			q.r(2, 5, 11, 1, c.l);
			q.r(4, 8, 3, 5, WOOD.m);
			q.r(8, 8, 1, 2, c.d);
			q.p(13, 6, c.d);
			q.p(14, 6, c.d);
			break;
		case 'shield':
			q.r(3, 2, 10, 7, c.m);
			q.r(4, 9, 8, 2, c.m);
			q.r(6, 11, 4, 2, c.m);
			q.r(7, 13, 2, 1, c.m);
			q.r(3, 2, 10, 1, c.l);
			q.r(3, 2, 1, 7, c.l);
			q.r(12, 3, 1, 6, c.d);
			q.d(8, 7, 2, GOLD.m);
			q.p(7, 6, GOLD.l);
			break;
		case 'helm':
			q.e(8, 7, 5, 5, c.m);
			q.r(3, 8, 10, 5, c.m);
			q.r(4, 4, 3, 2, c.l);
			q.r(5, 9, 6, 2, c.d);
			q.r(7, 8, 2, 5, c.d);
			q.r(3, 13, 10, 1, c.d);
			q.p(8, 1, GOLD.m);
			break;
		case 'mask':
			q.e(8, 8, 5, 6, c.m);
			q.r(5, 6, 2, 2, OUTLINE);
			q.r(9, 6, 2, 2, OUTLINE);
			q.r(7, 9, 2, 2, c.d);
			q.r(6, 12, 4, 1, c.d);
			q.r(5, 3, 3, 1, c.l);
			break;
		case 'cloak':
			q.r(5, 2, 6, 2, c.l);
			for (let y = 4; y <= 14; y++) q.r(8 - Math.floor((y - 2) / 2) - 1, y, 2 * Math.floor((y - 2) / 2) + 2, 1, y % 3 === 0 ? c.d : c.m);
			q.p(7, 4, GOLD.m);
			q.p(8, 4, GOLD.l);
			break;
		case 'gloves':
			q.r(5, 7, 7, 6, c.m);
			for (let i = 0; i < 4; i++) q.r(5 + i * 2 - (i === 3 ? 1 : 0), 3 + (i === 1 || i === 2 ? -1 : 0), 2, 5, c.m);
			q.r(3, 8, 2, 3, c.m);
			q.r(5, 13, 7, 1, c.d);
			q.r(5, 7, 7, 1, c.l);
			break;
		case 'boots':
			q.r(5, 2, 4, 9, c.m);
			q.r(5, 10, 9, 3, c.m);
			q.r(5, 2, 4, 2, c.l);
			q.r(5, 13, 9, 1, c.d);
			q.r(5, 4, 4, 1, GOLD.m);
			break;
		case 'plate':
			q.r(4, 3, 8, 10, c.m);
			q.r(1, 3, 3, 4, c.l);
			q.r(12, 3, 3, 4, c.l);
			q.r(6, 2, 4, 2, OUTLINE);
			q.r(4, 9, 8, 1, c.d);
			q.r(7, 4, 2, 9, c.l);
			q.r(4, 12, 8, 2, c.d);
			q.p(8, 9, GOLD.m);
			break;
		case 'robe':
			for (let y = 2; y <= 14; y++) {
				const w = 4 + Math.floor((y - 2) * 0.6);
				q.r(8 - Math.floor(w / 2), y, w, 1, c.m);
			}
			q.r(6, 2, 4, 2, c.l);
			q.r(3, 5, 2, 5, c.d);
			q.r(11, 5, 2, 5, c.d);
			q.r(5, 7, 6, 1, GOLD.m);
			q.l(8, 4, 8, 14, c.d);
			break;
		case 'ammo':
			for (const dx of [0, 3, 6]) {
				q.l(3 + dx, 13, 5 + dx, 3, WOOD.l);
				q.p(5 + dx, 2, c.l);
				q.p(4 + dx, 3, c.m);
				q.p(6 + dx, 3, c.m);
				q.p(3 + dx, 14, GOLD.l);
			}
			break;
		case 'meat':
			q.d(6, 6, 4, '#a8442c');
			q.d(5, 5, 2, '#d8785a');
			q.l(8, 9, 12, 13, MATS.bone.m);
			q.d(13, 14, 1.2, MATS.bone.l);
			q.d(11, 12, 1, MATS.bone.l);
			break;
		case 'loaf':
			q.e(8, 9, 6, 4, '#c08a40');
			q.e(8, 7, 5, 3, '#d8a458');
			q.l(5, 6, 6, 8, '#a06a28');
			q.l(8, 5, 9, 8, '#a06a28');
			q.l(11, 6, 12, 8, '#a06a28');
			break;
		case 'bowl':
			q.e(8, 9, 6, 2, '#d8d0c0');
			q.r(3, 9, 10, 1, '#d8d0c0');
			q.e(8, 11, 4, 2, '#a89880');
			q.e(8, 8, 5, 1, '#c08a40');
			q.p(6, 4, '#c8d0d4');
			q.p(7, 3, '#c8d0d4');
			q.p(9, 4, '#c8d0d4');
			q.p(10, 3, '#c8d0d4');
			break;
		case 'flask':
			q.d(8, 10, 4, MATS.glass.m);
			q.r(7, 3, 2, 4, MATS.glass.l);
			q.r(6, 2, 4, 1, '#8a5a30');
			q.e(8, 11, 3, 2.5, '#3a7ae0');
			q.p(6, 8, '#ffffff');
			break;
		case 'potion':
			q.d(8, 10, 4, MATS.glass.m);
			q.r(7, 4, 2, 3, MATS.glass.l);
			q.r(6, 2, 4, 2, '#8a5a30');
			q.e(8, 11, 3, 2.5, liquid.m);
			q.p(7, 11, liquid.l);
			q.p(6, 8, '#ffffff');
			break;
		case 'grenade':
			q.d(8, 10, 4, c.m);
			q.d(7, 9, 1.5, c.l);
			q.r(6, 4, 4, 2, c.d);
			q.r(9, 3, 3, 1, GOLD.m);
			q.ring(12, 4, 1.5, GOLD.l);
			q.r(5, 11, 6, 1, c.d);
			break;
		case 'book': {
			const cv = CLOTH_HUES[hv % CLOTH_HUES.length];
			q.r(3, 2, 10, 12, cv.m);
			q.r(3, 2, 2, 12, cv.d);
			q.r(12, 3, 1, 10, '#e8e0c8');
			q.r(6, 4, 5, 2, GOLD.m);
			q.r(6, 8, 5, 1, cv.l);
			q.r(6, 10, 4, 1, cv.l);
			break;
		}
		case 'scroll':
			q.r(4, 3, 8, 10, '#e8dcb8');
			q.r(3, 2, 10, 2, '#c8b078');
			q.r(3, 12, 10, 2, '#c8b078');
			for (let y = 5; y < 11; y += 2) q.r(5, y, 6, 1, '#8a7a50');
			q.p(8, 9, '#a03030');
			break;
		case 'letter':
			q.r(2, 4, 12, 9, '#e8e0c8');
			q.l(2, 4, 8, 9, '#b0a070');
			q.l(14, 4, 8, 9, '#b0a070');
			q.r(7, 8, 2, 2, '#a03030');
			break;
		case 'key':
			q.ring(5, 5, 2.5, GOLD.m);
			q.p(5, 3, GOLD.l);
			q.l(7, 7, 13, 13, GOLD.m);
			q.l(8, 7, 14, 13, GOLD.d);
			q.p(11, 12, GOLD.l);
			q.p(12, 11, GOLD.l);
			q.p(13, 14, GOLD.m);
			break;
		case 'lantern':
			q.r(5, 3, 6, 1, c.m);
			q.r(7, 1, 2, 2, c.d);
			q.r(4, 4, 8, 8, '#e0b030');
			q.r(6, 6, 4, 4, '#fff0a0');
			q.r(4, 12, 8, 2, c.m);
			q.r(4, 4, 1, 8, c.d);
			q.r(11, 4, 1, 8, c.d);
			break;
		case 'gear':
			q.d(8, 8, 4, c.m);
			for (let a = 0; a < 8; a++) q.r(7 + Math.round(Math.cos((a * Math.PI) / 4) * 5), 7 + Math.round(Math.sin((a * Math.PI) / 4) * 5), 2, 2, c.m);
			q.d(8, 8, 1.5, OUTLINE);
			q.p(6, 6, c.l);
			break;
		case 'herb': {
			const leaf = spec.tg && TAG_COLORS[spec.tg] ? TAG_COLORS[spec.tg] : '#4aa050';
			q.l(8, 14, 8, 6, '#3a6a30');
			q.e(5, 7, 2.5, 1.5, '#4aa050');
			q.e(11, 5, 2.5, 1.5, '#4aa050');
			q.e(5, 11, 2.5, 1.5, '#3a8a40');
			q.e(11, 10, 2.5, 1.5, '#3a8a40');
			q.d(8, 4, 2, leaf);
			break;
		}
		case 'egg':
			q.e(8, 9, 4, 5, '#e8e0c8');
			q.e(7, 7, 2, 2.5, '#fff8e8');
			q.p(9, 11, '#a89868');
			q.p(10, 9, '#a89868');
			q.p(7, 12, '#a89868');
			break;
		case 'fossil': {
			const f = MATS.stone;
			q.d(8, 8, 6, f.m);
			q.ring(8, 8, 4, f.l);
			q.ring(8, 8, 2, f.d);
			q.p(8, 8, f.l);
			q.l(8, 2, 8, 14, f.d);
			break;
		}
		case 'coin':
			q.d(8, 8, 6, GOLD.m);
			q.ring(8, 8, 5, GOLD.l);
			q.r(7, 5, 2, 6, GOLD.d);
			q.p(5, 5, GOLD.l);
			break;
		case 'aug':
			q.r(3, 4, 10, 8, '#2a4a3a');
			q.r(5, 6, 6, 4, '#4ae0a0');
			for (let i = 0; i < 4; i++) {
				q.r(4 + i * 2, 2, 1, 2, '#c8d0d4');
				q.r(4 + i * 2, 12, 1, 2, '#c8d0d4');
			}
			q.p(7, 7, '#ffffff');
			break;
		case 'relic': {
			const cc = MATS.crystal;
			q.r(4, 12, 8, 2, c.d);
			q.r(6, 10, 4, 2, c.m);
			q.d(8, 6, 3.5, cc.m);
			q.p(7, 5, cc.l);
			q.p(8, 4, cc.l);
			q.p(8, 1, GOLD.l);
			q.p(3, 5, GOLD.l);
			q.p(13, 5, GOLD.l);
			break;
		}
		case 'gem': {
			const gc = [MATS.crystal, MATS.glass, { d: '#7a1a1a', m: '#e03a3a', l: '#ff9a9a' }, { d: '#1a6a2a', m: '#3ae05a', l: '#9affb0' }, MATS.plasma][hv % 5];
			q.r(5, 3, 6, 1, gc.l);
			q.r(3, 4, 10, 3, gc.m);
			q.r(4, 7, 8, 2, gc.m);
			q.r(5, 9, 6, 1, gc.d);
			q.r(6, 10, 4, 1, gc.d);
			q.r(7, 11, 2, 1, gc.d);
			q.r(4, 4, 3, 2, gc.l);
			q.l(8, 4, 8, 10, gc.l);
			break;
		}
		case 'ring':
			q.ring(8, 9, 4, GOLD.m);
			q.ring(8, 9, 3, GOLD.m);
			q.d(8, 4, 2, MATS.crystal.m);
			q.p(7, 3, MATS.crystal.l);
			break;
		case 'instrument':
			q.e(6, 10, 4, 4, WOOD.m);
			q.d(6, 10, 1.2, OUTLINE);
			q.l(8, 8, 13, 3, WOOD.l);
			q.p(13, 2, GOLD.m);
			q.l(4, 8, 8, 12, WOOD.d);
			break;
		case 'pet':
			q.e(8, 10, 3.5, 3, c.m);
			q.d(4, 6, 1.5, c.m);
			q.d(7, 3, 1.5, c.m);
			q.d(10, 3, 1.5, c.m);
			q.d(13, 6, 1.5, c.m);
			break;
		case 'bones':
			q.e(8, 6, 5, 4.5, '#e8e0c8');
			q.r(5, 9, 6, 3, '#d8d0b8');
			q.r(5, 5, 2, 2, OUTLINE);
			q.r(9, 5, 2, 2, OUTLINE);
			q.r(7, 8, 2, 1, '#8a8068');
			for (let i = 0; i < 3; i++) q.r(6 + i * 2, 12, 1, 2, '#e8e0c8');
			break;
		case 'tool':
			q.l(4, 13, 11, 6, WOOD.m);
			q.r(8, 2, 6, 3, c.m);
			q.r(8, 2, 6, 1, c.l);
			q.r(13, 3, 2, 4, c.m);
			q.r(7, 3, 2, 2, c.d);
			break;
		case 'curio':
			q.e(8, 9, 4.5, 4.5, CLOTH_HUES[hv % CLOTH_HUES.length].m);
			q.r(6, 2, 4, 3, CLOTH_HUES[hv % CLOTH_HUES.length].d);
			q.r(5, 13, 6, 1, CLOTH_HUES[hv % CLOTH_HUES.length].d);
			q.r(5, 8, 6, 1, GOLD.m);
			q.p(6, 7, '#ffffff');
			break;
		case 'junk':
		default:
			q.e(8, 10, 5, 4, MATS.leather.m);
			q.r(6, 3, 4, 3, MATS.leather.m);
			q.r(5, 6, 6, 1, GOLD.d);
			q.p(6, 8, MATS.leather.l);
			q.p(7, 8, MATS.leather.l);
			q.r(6, 2, 4, 1, MATS.leather.d);
	}
}

/** corner marks for tags and heirloom level */
function overlays(q: Pix, spec: IconSpec) {
	const t = spec.tg;
	if (t && TAG_COLORS[t] && spec.k !== 'status') {
		const col = TAG_COLORS[t];
		q.r(11, 11, 4, 4, OUTLINE);
		q.r(12, 12, 2, 2, col);
		if (t === 'cursed') {
			q.p(12, 12, '#ffffff');
			q.p(13, 12, '#ffffff');
		} else if (t === 'enchanted') {
			q.p(12, 11, '#ffffff');
			q.p(11, 12, '#ffffff');
		}
	}
	const lv = spec.lv ?? 0;
	for (let i = 0; i < lv; i++) q.p(1 + i * 2, 1, '#ffe14a');
}

function status(q: Pix, spec: IconSpec) {
	const lv = spec.lv ?? 1;
	switch (spec.k) {
		case 'hunger':
			q.ring(8, 8, 5, '#d8d0c0');
			q.ring(8, 8, 3, '#a89880');
			q.l(2, 2, 2, 7, '#c8d0d4');
			q.l(14, 2, 14, 7, '#c8d0d4');
			if (lv > 1) q.r(7, 7, 2, 2, '#e03a3a');
			break;
		case 'thirst':
			q.e(8, 10, 4, 4, '#4aa0f0');
			q.l(8, 2, 5, 8, '#4aa0f0');
			q.l(8, 2, 11, 8, '#4aa0f0');
			q.p(6, 9, '#c8e8ff');
			q.p(6, 10, '#c8e8ff');
			if (lv > 1) q.r(7, 12, 2, 1, '#2a60b0');
			break;
		case 'meal':
			item(q, { k: 'bowl', r: 0, h: 0 });
			q.p(12, 2, '#6ae070');
			q.p(13, 2, '#6ae070');
			q.p(12, 1, '#6ae070');
			break;
		case 'injury':
			q.r(2, 5, 12, 6, '#e8e0c8');
			q.r(7, 4, 2, 8, '#d03a3a');
			q.r(5, 7, 6, 2, '#d03a3a');
			q.r(2, 5, 12, 1, '#c8c0a8');
			q.r(2, 10, 12, 1, '#c8c0a8');
			break;
		case 'mutation':
			for (let y = 1; y < 15; y++) {
				const x1 = 8 + Math.round(Math.sin(y * 0.8) * 4), x2 = 8 - Math.round(Math.sin(y * 0.8) * 4);
				q.p(x1, y, '#5ae070');
				q.p(x2, y, '#e05ae0');
				if (y % 3 === 0) q.l(x1, y, x2, y, '#c8d0d4');
			}
			break;
		case 'stance':
			item(q, { k: 'shield', m: 'steel', r: 0, h: 0 });
			break;
		case 'noise':
			q.r(2, 6, 3, 4, '#c8d0d4');
			q.l(5, 6, 8, 3, '#c8d0d4');
			q.l(5, 10, 8, 13, '#c8d0d4');
			q.r(8, 3, 1, 10, '#c8d0d4');
			q.ring(9, 8, 3, '#f0c040');
			q.ring(9, 8, 5, '#f0c040');
			break;
		case 'scent':
			for (let i = 0; i < 3; i++)
				for (let y = 2; y < 14; y++) q.p(4 + i * 4 + Math.round(Math.sin(y * 0.7 + i) * 1.5), y, '#a0d060');
			break;
		case 'weather':
			weather(q, spec.tg ?? '');
			break;
		case 'time':
			timeIcon(q, spec.tg ?? 'day');
			break;
		default:
			statusClass(q, spec.tg ?? 'other');
	}
}

function cloud(q: Pix, col = '#b8c4cc', y = 6) {
	q.e(6, y + 1, 3, 2.5, col);
	q.e(10, y, 3.5, 3, col);
	q.r(4, y + 1, 9, 3, col);
	q.r(4, y + 3, 9, 1, '#8a98a2');
}

function weather(q: Pix, w: string) {
	const l = w.toLowerCase();
	if (/storm|tempest|thunder|lightning/.test(l)) {
		cloud(q, '#6a7480', 4);
		q.l(9, 8, 6, 11, '#ffee40');
		q.l(6, 11, 10, 11, '#ffee40');
		q.l(10, 11, 7, 15, '#ffee40');
	} else if (/snow|blizzard|flurr/.test(l)) {
		cloud(q, '#c8d4dc', 3);
		for (const [x, y] of [[4, 10], [8, 12], [12, 10], [6, 14], [10, 14]]) {
			q.p(x, y, '#ffffff');
			q.p(x - 1, y, '#c8e8ff');
			q.p(x + 1, y, '#c8e8ff');
			q.p(x, y - 1, '#c8e8ff');
			q.p(x, y + 1, '#c8e8ff');
		}
	} else if (/rain|drizzle|acid|shower/.test(l)) {
		cloud(q, '#8a98a8', 3);
		const col = /acid/.test(l) ? '#b0ff30' : '#4aa0f0';
		for (const [x, y] of [[4, 10], [7, 12], [10, 10], [13, 12], [6, 14], [11, 14]]) q.l(x, y, x - 1, y + 2, col);
	} else if (/fog|mist|haze|smog|prism|spore|pollen/.test(l)) {
		const col = /spore|pollen/.test(l) ? '#c8e060' : /prism/.test(l) ? '#c8a0f8' : '#b8c4cc';
		for (let y = 3; y < 14; y += 3) q.r(2 + (y % 2) * 2, y, 11, 1, col);
		if (/spore|pollen/.test(l)) for (const [x, y] of [[4, 5], [10, 8], [6, 12], [12, 4]]) q.d(x, y, 1, '#e0f080');
	} else if (/sand|dust|ash|fallout|cinder/.test(l)) {
		const col = /ash|fallout/.test(l) ? '#9a9aa0' : '#d8b060';
		for (let i = 0; i < 18; i++) q.p(2 + ((i * 7) % 12), 2 + ((i * 5) % 12), col);
		q.l(2, 9, 13, 6, col);
		q.l(2, 12, 13, 9, col);
	} else if (/heat|hot|scorch/.test(l)) {
		q.d(8, 8, 4, '#ffb020');
		for (let a = 0; a < 8; a++) q.p(8 + Math.cos((a * Math.PI) / 4) * 6, 8 + Math.sin((a * Math.PI) / 4) * 6, '#ff7a20');
		q.d(8, 8, 2, '#fff0a0');
	} else if (/frost|ice|cold/.test(l)) {
		snowflake(q, '#a0e0ff');
	} else if (/clear|sun/.test(l)) {
		timeIcon(q, 'day');
	} else {
		cloud(q);
	}
}

function snowflake(q: Pix, col: string) {
	q.l(8, 1, 8, 14, col);
	q.l(2, 8, 14, 8, col);
	q.l(3, 3, 13, 13, col);
	q.l(13, 3, 3, 13, col);
	q.d(8, 8, 1, '#ffffff');
}

function timeIcon(q: Pix, ph: string) {
	if (ph === 'night') {
		q.d(8, 8, 5, '#e8e4c0');
		q.d(11, 6, 4.5, OUTLINE);
		q.p(5, 10, '#c8c4a0');
		q.p(3, 3, '#ffffff');
		q.p(13, 12, '#ffffff');
		return;
	}
	if (ph === 'dawn' || ph === 'dusk') {
		const c = ph === 'dawn' ? '#ffb060' : '#ff7050';
		q.e(8, 12, 5, 4, c);
		q.r(0, 12, 16, 4, OUTLINE);
		q.r(1, 12, 14, 1, '#8a6a4a');
		for (const [x, y] of [[8, 3], [3, 6], [13, 6], [1, 10], [15, 10]]) q.p(x, y, c);
		return;
	}
	q.d(8, 8, 3.5, '#ffd040');
	q.p(7, 7, '#fff8c0');
	for (let a = 0; a < 8; a++) {
		const dx = Math.cos((a * Math.PI) / 4), dy = Math.sin((a * Math.PI) / 4);
		q.l(8 + dx * 5.5, 8 + dy * 5.5, 8 + dx * 7, 8 + dy * 7, '#ffb020');
	}
}

function statusClass(q: Pix, k: string) {
	switch (k) {
		case 'poison':
			q.e(8, 10, 4, 4, '#5ae050');
			q.l(8, 2, 5, 8, '#5ae050');
			q.l(8, 2, 11, 8, '#5ae050');
			q.p(6, 9, '#c8ffc0');
			q.r(7, 11, 1, 1, '#1a5a1a');
			q.r(9, 11, 1, 1, '#1a5a1a');
			break;
		case 'bleed':
			q.e(8, 10, 4, 4, '#d83030');
			q.l(8, 2, 5, 8, '#d83030');
			q.l(8, 2, 11, 8, '#d83030');
			q.p(6, 9, '#ff9a9a');
			break;
		case 'burn':
			q.e(8, 10, 4, 4, '#ff7a20');
			q.l(8, 1, 5, 9, '#ff7a20');
			q.l(8, 1, 11, 9, '#ff7a20');
			q.e(8, 11, 2, 2.5, '#ffe060');
			q.p(8, 10, '#ffffff');
			break;
		case 'blessed':
			q.ring(8, 4, 3, '#ffe14a');
			q.r(7, 6, 2, 8, '#fff0a0');
			q.r(4, 8, 8, 2, '#fff0a0');
			break;
		case 'sleepy':
			q.r(2, 2, 5, 1, '#8ab0f0');
			q.l(6, 3, 2, 7, '#8ab0f0');
			q.r(2, 7, 5, 1, '#8ab0f0');
			q.r(8, 8, 6, 1, '#6a90d0');
			q.l(13, 9, 9, 14, '#6a90d0');
			q.r(9, 14, 6, 1, '#6a90d0');
			break;
		case 'sick':
			q.d(8, 8, 6, '#a0c860');
			q.r(5, 6, 2, 2, OUTLINE);
			q.r(9, 6, 2, 2, OUTLINE);
			for (let x = 5; x < 11; x++) q.p(x, 11 + (x % 2), '#3a5a20');
			break;
		case 'frost':
			snowflake(q, '#a0e0ff');
			break;
		case 'dazed':
			for (const [x, y] of [[4, 4], [11, 3], [8, 9], [3, 11], [12, 12]]) {
				q.p(x, y, '#ffe14a');
				q.p(x - 1, y, '#ffb020');
				q.p(x + 1, y, '#ffb020');
				q.p(x, y - 1, '#ffb020');
				q.p(x, y + 1, '#ffb020');
			}
			break;
		case 'haste':
			q.l(9, 1, 4, 9, '#ffe14a');
			q.l(4, 9, 11, 9, '#ffe14a');
			q.l(11, 9, 6, 15, '#ffe14a');
			q.l(9, 2, 5, 9, '#ffb020');
			break;
		case 'air':
			for (let y = 3; y <= 11; y += 4) {
				q.r(2, y, 9, 1, '#c8d4dc');
				q.p(11, y - 1, '#c8d4dc');
				q.p(12, y - 1, '#c8d4dc');
				q.p(12, y, '#c8d4dc');
			}
			break;
		case 'meal':
			item(q, { k: 'bowl', r: 0, h: 0 });
			break;
		default:
			q.r(6, 2, 4, 8, '#f0c040');
			q.r(6, 12, 4, 2, '#f0c040');
			q.r(7, 3, 1, 5, OUTLINE);
	}
}

function build(spec: IconSpec): HTMLCanvasElement {
	const q = new Pix();
	const statusKinds = ['hunger', 'thirst', 'meal', 'status', 'injury', 'mutation', 'stance', 'noise', 'scent', 'weather', 'time'];
	if (statusKinds.includes(spec.k)) status(q, spec);
	else {
		item(q, spec);
		overlays(q, spec);
	}
	if ((spec.h & 1) === 1 && ['sword', 'dagger', 'axe', 'mace', 'spear', 'staff', 'flail', 'hook', 'bow', 'crossbow', 'tool', 'key'].includes(spec.k)) q.mirror();
	const c = document.createElement('canvas');
	c.width = N;
	c.height = N;
	const x = c.getContext('2d');
	if (!x) return c;
	const img = x.createImageData(N, N);
	const set = (i: number, hex: string) => {
		const n = parseInt(hex.slice(1), 16);
		img.data[i * 4] = (n >> 16) & 255;
		img.data[i * 4 + 1] = (n >> 8) & 255;
		img.data[i * 4 + 2] = n & 255;
		img.data[i * 4 + 3] = 255;
	};
	// outline: empty cells that touch a filled one
	for (let y = 0; y < N; y++)
		for (let xx = 0; xx < N; xx++) {
			const i = y * N + xx;
			const col = q.g[i];
			if (col) set(i, col);
			else if (
				(xx > 0 && q.g[i - 1]) || (xx < N - 1 && q.g[i + 1]) || (y > 0 && q.g[i - N]) || (y < N - 1 && q.g[i + N])
			)
				set(i, OUTLINE);
		}
	x.putImageData(img, 0, 0);
	return c;
}

const cache = new Map<string, HTMLCanvasElement>();

function keyOf(s: IconSpec): string {
	const varied = s.h % 8;
	return `${s.k}|${s.m ?? ''}|${s.tg ?? ''}|${s.lv ?? 0}|${s.f ?? ''}|${varied}|${s.h & 1}`;
}

/** the cached 16x16 art of a spec (no frame) */
export function iconArt(spec: IconSpec): HTMLCanvasElement {
	const k = keyOf(spec);
	let c = cache.get(k);
	if (!c) {
		c = build(spec);
		if (cache.size > 600) cache.clear();
		cache.set(k, c);
	}
	return c;
}

/** draw an item icon with its rarity frame; (x, y) is the top-left in CSS pixels, size the side length */
export function drawIcon(ctx: CanvasRenderingContext2D, spec: IconSpec, x: number, y: number, size: number) {
	const art = iconArt(spec);
	ctx.save();
	ctx.imageSmoothingEnabled = false;
	const r = Math.max(0, Math.min(6, spec.r | 0));
	ctx.fillStyle = 'rgba(8,14,16,0.85)';
	ctx.fillRect(x, y, size, size);
	if (r >= 3) {
		ctx.shadowColor = RARITY_COLORS[r];
		ctx.shadowBlur = Math.max(2, size / 6);
	}
	const lw = Math.max(1, Math.round(size / 16));
	ctx.strokeStyle = RARITY_COLORS[r];
	ctx.lineWidth = lw;
	ctx.strokeRect(x + lw / 2, y + lw / 2, size - lw, size - lw);
	ctx.shadowBlur = 0;
	const pad = Math.max(1, Math.round(size / 16));
	ctx.drawImage(art, x + pad, y + pad, size - 2 * pad, size - 2 * pad);
	ctx.restore();
}

/** status icon: no rarity frame; a faint tinted plate by severity */
export function drawStatusIcon(ctx: CanvasRenderingContext2D, spec: IconSpec, x: number, y: number, size: number) {
	const art = iconArt(spec);
	ctx.save();
	ctx.imageSmoothingEnabled = false;
	ctx.fillStyle = (spec.lv ?? 1) > 1 ? 'rgba(90,20,20,0.7)' : 'rgba(8,14,16,0.7)';
	ctx.fillRect(x, y, size, size);
	ctx.drawImage(art, x, y, size, size);
	ctx.restore();
}
