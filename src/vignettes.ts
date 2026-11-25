/**
 * Procedural retro-storybook / pixel-wash vignette illustrations for key story beats.
 * Deterministic from (seed, title, kind); renders to a crisp low-res art buffer (180x76)
 * with 4x4 Bayer dithering and blits pixelated onto any Canvas2D target.
 */

export type VignetteKind =
	| 'origin'
	| 'journey'
	| 'battle'
	| 'siege'
	| 'delve'
	| 'ruin'
	| 'voyage'
	| 'sea'
	| 'shrine'
	| 'miracle'
	| 'crown'
	| 'triumph'
	| 'death'
	| 'epitaph';

export interface VignetteOpts {
	kind: VignetteKind;
	seed: string;
	title?: string;
	accent?: string;
	/** optional logical art resolution (default 180x76) */
	artW?: number;
	artH?: number;
}

type RGB = [number, number, number];

// ---------- deterministic PRNG ----------
function hash32(s: string): number {
	let h = 2166136261 >>> 0;
	for (let i = 0; i < s.length; i++) {
		h ^= s.charCodeAt(i);
		h = Math.imul(h, 16777619) >>> 0;
	}
	return h >>> 0;
}
function mulberry32(seed: number): () => number {
	let a = seed >>> 0 || 1;
	return () => {
		a = (a + 0x6d2b79f5) >>> 0;
		let t = Math.imul(a ^ (a >>> 15), a | 1);
		t ^= t + Math.imul(t ^ (t >>> 7), t | 61);
		return ((t ^ (t >>> 14)) >>> 0) / 4294967296;
	};
}

// 4x4 Bayer threshold matrix (0..15)
const BAYER = [
	0, 8, 2, 10,
	12, 4, 14, 6,
	3, 11, 1, 9,
	15, 7, 13, 5,
];

function hexToRgb(hex: string | undefined): RGB | null {
	if (!hex || !hex.startsWith('#')) return null;
	const h = hex.slice(1);
	if (h.length === 3) {
		return [parseInt(h[0] + h[0], 16), parseInt(h[1] + h[1], 16), parseInt(h[2] + h[2], 16)];
	}
	if (h.length === 6) {
		return [parseInt(h.slice(0, 2), 16), parseInt(h.slice(2, 4), 16), parseInt(h.slice(4, 6), 16)];
	}
	return null;
}

function mixRgb(a: RGB, b: RGB, t: number): RGB {
	return [
		Math.round(a[0] + (b[0] - a[0]) * t),
		Math.round(a[1] + (b[1] - a[1]) * t),
		Math.round(a[2] + (b[2] - a[2]) * t),
	];
}

// ---------- story beat mapping ----------
export function vignetteKindForChapter(kind: string, title = '', text = '', dead = false): VignetteKind {
	const k = kind.toLowerCase();
	const blob = `${title} ${text}`.toLowerCase();
	if (k === 'prologue') return 'origin';
	if (k === 'epilogue') return dead ? 'epitaph' : 'crown';
	if (k === 'end') return dead ? 'death' : 'triumph';
	if (k === 'foe') return /siege|wall|fort|castle|war/.test(blob) ? 'siege' : 'battle';
	if (k === 'love') return 'shrine';
	if (k === 'trial') {
		if (/sea|ocean|ship|sail|isle|coast|wave/.test(blob)) return 'voyage';
		if (/shrine|temple|altar|saint|omen|god|miracle/.test(blob)) return 'miracle';
		return 'delve';
	}
	if (k === 'first') {
		if (/sea|voyage|ship|sail/.test(blob)) return 'voyage';
		if (/ruin|crypt|cavern|dungeon|vault|deep/.test(blob)) return 'ruin';
		if (/slay|fought|battle|blood|blade/.test(blob)) return 'battle';
		return 'crown';
	}
	// road / wander / custom: inspect prose keywords for rich variety
	if (/sea|ocean|voyage|caravel|longship|sail|harbour|harbor/.test(blob)) return 'voyage';
	if (/siege|battle|war|blood|slain|blade|army/.test(blob)) return 'battle';
	if (/ruin|crypt|dungeon|cavern|abyss|vault|tomb|delve/.test(blob)) return 'delve';
	if (/shrine|temple|altar|oath|omen|miracle|blessing|prayer/.test(blob)) return 'shrine';
	if (/crown|throne|citadel|palace|triumph|king|queen|ruler/.test(blob)) return 'crown';
	return 'journey';
}

const CUE_VIGNETTE: Record<string, VignetteKind> = {
	discovery: 'delve',
	relic: 'shrine',
	legend: 'crown',
	wedding: 'shrine',
	ceremony: 'shrine',
	funeral: 'epitaph',
	omen: 'miracle',
	prophecy: 'miracle',
	rift: 'miracle',
	dungeon: 'delve',
	tower: 'ruin',
	death: 'death',
	ending: 'triumph',
	victory: 'crown',
	quest: 'journey',
};

const CUE_LABELS: Record<string, string> = {
	discovery: 'New Discovery',
	relic: 'Relic Uncovered',
	legend: 'A Legend Forged',
	wedding: 'Sacred Union',
	ceremony: 'Solemn Rite',
	funeral: 'Final Farewell',
	omen: 'An Omen Appears',
	prophecy: 'Prophecy Spoken',
	rift: 'The World Turns',
	dungeon: 'Into the Depths',
	tower: 'The Spire Looms',
	death: 'The Tale Ends',
	ending: 'A World Shaped',
	victory: 'Triumph',
	quest: 'A New Road',
};

export function vignetteKindForCue(cueKind: string): VignetteKind | null {
	return CUE_VIGNETTE[cueKind.toLowerCase()] ?? null;
}

export function cueLabel(cueKind: string): string {
	return CUE_LABELS[cueKind.toLowerCase()] ?? 'Chronicle Beat';
}

// ---------- pixel-wash drawing primitives ----------
interface Painter {
	w: number;
	h: number;
	data: Uint8ClampedArray;
	px(x: number, y: number, c: RGB, a?: number): void;
	rect(x: number, y: number, w: number, h: number, c: RGB, a?: number): void;
	ditherBand(y0: number, y1: number, top: RGB, bot: RGB): void;
	ditherGlow(cx: number, cy: number, rad: number, c: RGB, strength?: number): void;
	line(x0: number, y0: number, x1: number, y1: number, c: RGB, a?: number): void;
	ridge(baseY: number, amp: number, freq: number, phase: number, c: RGB, seed: number): void;
}

function makePainter(w: number, h: number): Painter {
	const data = new Uint8ClampedArray(w * h * 4);
	const px = (x: number, y: number, c: RGB, a = 1) => {
		const ix = x | 0;
		const iy = y | 0;
		if (ix < 0 || iy < 0 || ix >= w || iy >= h) return;
		const idx = (iy * w + ix) * 4;
		if (a >= 0.999) {
			data[idx] = c[0];
			data[idx + 1] = c[1];
			data[idx + 2] = c[2];
			data[idx + 3] = 255;
		} else if (a > 0.01) {
			const inv = 1 - a;
			data[idx] = (data[idx] * inv + c[0] * a) | 0;
			data[idx + 1] = (data[idx + 1] * inv + c[1] * a) | 0;
			data[idx + 2] = (data[idx + 2] * inv + c[2] * a) | 0;
			data[idx + 3] = 255;
		}
	};
	const rect = (x: number, y: number, rw: number, rh: number, c: RGB, a = 1) => {
		const x0 = Math.max(0, x | 0);
		const y0 = Math.max(0, y | 0);
		const x1 = Math.min(w, (x + rw) | 0);
		const y1 = Math.min(h, (y + rh) | 0);
		for (let yy = y0; yy < y1; yy++) for (let xx = x0; xx < x1; xx++) px(xx, yy, c, a);
	};
	const ditherBand = (y0: number, y1: number, top: RGB, bot: RGB) => {
		const span = Math.max(1, y1 - y0);
		for (let y = Math.max(0, y0 | 0); y < Math.min(h, y1 | 0); y++) {
			const t = (y - y0) / span;
			const steps = 4;
			const scaled = t * steps;
			const lo = Math.min(steps - 1, Math.floor(scaled));
			const frac = scaled - lo;
			const cA = mixRgb(top, bot, lo / steps);
			const cB = mixRgb(top, bot, (lo + 1) / steps);
			for (let x = 0; x < w; x++) {
				const b = BAYER[((y & 3) << 2) | (x & 3)] / 16;
				px(x, y, frac > b ? cB : cA);
			}
		}
	};
	const ditherGlow = (cx: number, cy: number, rad: number, c: RGB, strength = 1) => {
		const x0 = Math.max(0, Math.floor(cx - rad));
		const x1 = Math.min(w - 1, Math.ceil(cx + rad));
		const y0 = Math.max(0, Math.floor(cy - rad));
		const y1 = Math.min(h - 1, Math.ceil(cy + rad));
		for (let y = y0; y <= y1; y++) {
			for (let x = x0; x <= x1; x++) {
				const d = Math.hypot(x - cx, y - cy) / rad;
				if (d >= 1) continue;
				const falloff = (1 - d) * (1 - d) * strength;
				const b = BAYER[((y & 3) << 2) | (x & 3)] / 16;
				if (falloff > b) px(x, y, c, 0.55 + 0.45 * falloff);
			}
		}
	};
	const line = (x0: number, y0: number, x1: number, y1: number, c: RGB, a = 1) => {
		let ix0 = x0 | 0, iy0 = y0 | 0;
		const ix1 = x1 | 0, iy1 = y1 | 0;
		const dx = Math.abs(ix1 - ix0), sx = ix0 < ix1 ? 1 : -1;
		const dy = -Math.abs(iy1 - iy0), sy = iy0 < iy1 ? 1 : -1;
		let err = dx + dy;
		for (let guard = 0; guard < 512; guard++) {
			px(ix0, iy0, c, a);
			if (ix0 === ix1 && iy0 === iy1) break;
			const e2 = 2 * err;
			if (e2 >= dy) { err += dy; ix0 += sx; }
			if (e2 <= dx) { err += dx; iy0 += sy; }
		}
	};
	const ridge = (baseY: number, amp: number, freq: number, phase: number, c: RGB, seed: number) => {
		for (let x = 0; x < w; x++) {
			const t = x * freq + phase;
			const n = Math.sin(t) * 0.55 + Math.sin(t * 2.3 + seed) * 0.3 + Math.sin(t * 5.1 + seed * 1.7) * 0.15;
			const top = Math.round(baseY - n * amp);
			for (let y = Math.max(0, top); y < h; y++) px(x, y, c);
		}
	};
	return { w, h, data, px, rect, ditherBand, ditherGlow, line, ridge };
}

function drawStars(p: Painter, r: () => number, count: number, maxY: number, c1: RGB, c2: RGB) {
	for (let i = 0; i < count; i++) {
		const x = (r() * p.w) | 0;
		const y = (r() * maxY) | 0;
		const bright = r() < 0.25;
		p.px(x, y, bright ? c1 : c2, bright ? 1 : 0.7);
		if (bright && r() < 0.35) {
			p.px(x - 1, y, c2, 0.45);
			p.px(x + 1, y, c2, 0.45);
			p.px(x, y - 1, c2, 0.45);
			p.px(x, y + 1, c2, 0.45);
		}
	}
}

// ---------- scene painters ----------
const INK: RGB = [11, 20, 21];
const DEEP_TEAL: RGB = [21, 56, 60];
const MID_TEAL: RGB = [34, 94, 92];
const SEA_SLATE: RGB = [63, 127, 122];
const PARCHMENT: RGB = [168, 150, 108];
const GOLD: RGB = [207, 192, 65];
const PALE: RGB = [232, 224, 186];
const EMBER: RGB = [224, 114, 62];
const CRIMSON: RGB = [156, 46, 42];
const ROSE: RGB = [204, 116, 124];

function paintOrigin(p: Painter, r: () => number, accent: RGB) {
	const { w, h } = p;
	const horizon = Math.round(h * 0.62);
	p.ditherBand(0, horizon, [14, 29, 34], mixRgb(MID_TEAL, accent, 0.25));
	drawStars(p, r, 26, horizon - 10, PALE, SEA_SLATE);
	// low sun or moon behind the hills
	const sx = Math.round(w * (0.28 + r() * 0.44));
	const sy = Math.round(horizon - 10 - r() * 12);
	p.ditherGlow(sx, sy, 22, mixRgb(GOLD, accent, 0.35), 0.75);
	p.ditherGlow(sx, sy, 7, PALE, 1);
	// layered ridges
	p.ridge(horizon - 4, 9, 0.045, r() * 6, [18, 44, 48], r() * 10);
	p.ridge(horizon + 4, 6, 0.06, r() * 6, [13, 30, 33], r() * 10);
	p.rect(0, horizon + 6, w, h - horizon - 6, INK);
	// winding road receding to horizon
	const roadEnd = sx + Math.round((r() - 0.5) * 18);
	for (let y = horizon + 3; y < h; y++) {
		const t = (y - horizon - 3) / Math.max(1, h - horizon - 3);
		const cx = roadEnd + (w * 0.5 - roadEnd) * Math.pow(t, 1.35) + Math.sin(t * 5.2) * 8 * (1 - t * 0.4);
		const rw = 1.5 + t * 14;
		for (let dx = -rw; dx <= rw; dx++) {
			const edge = Math.abs(dx) / rw;
			p.px(cx + dx, y, edge > 0.78 ? SEA_SLATE : mixRgb(DEEP_TEAL, PARCHMENT, 0.32 * (1 - edge)), 0.85);
		}
	}
	// lone hooded traveler silhouette with staff
	const tx = Math.round(w * 0.48);
	const ty = h - 18;
	p.rect(tx - 2, ty, 5, 4, INK); // hood
	p.rect(tx - 3, ty + 4, 7, 9, INK); // cloak
	p.px(tx - 4, ty + 11, INK); p.px(tx + 4, ty + 12, INK);
	p.line(tx + 5, ty - 2, tx + 6, ty + 13, PARCHMENT, 0.9); // walking staff
	p.px(tx + 5, ty - 3, GOLD); // lantern tip
	p.ditherGlow(tx + 5, ty - 3, 7, GOLD, 0.55);
	// waystone / bare tree on side
	const wx = tx < w / 2 ? Math.round(w * 0.78) : Math.round(w * 0.2);
	p.rect(wx - 2, horizon + 4, 4, 11, DEEP_TEAL);
	p.px(wx, horizon + 6, accent);
}

function paintBattle(p: Painter, r: () => number, accent: RGB) {
	const { w, h } = p;
	const wallY = Math.round(h * 0.56);
	p.ditherBand(0, wallY + 4, [24, 14, 18], [92, 34, 28]);
	p.ditherGlow(w * 0.35 + r() * w * 0.3, wallY + 2, 34, EMBER, 0.75);
	// rising smoke plumes
	for (let i = 0; i < 4; i++) {
		const cx = Math.round(w * (0.15 + i * 0.23 + r() * 0.08));
		p.ditherGlow(cx, wallY - 12 - r() * 14, 16 + r() * 10, [36, 24, 26], 0.7);
	}
	// fortress wall & crenellations
	p.rect(0, wallY, w, h - wallY, [20, 22, 26]);
	for (let x = 2; x < w; x += 9) {
		if (r() < 0.14) continue; // breached gap
		p.rect(x, wallY - 5, 5, 5, [28, 30, 34]);
		p.rect(x, wallY - 5, 5, 1, mixRgb(EMBER, PARCHMENT, 0.4), 0.5);
	}
	// twin keep towers
	const t1 = Math.round(w * 0.18), t2 = Math.round(w * 0.76);
	for (const tx of [t1, t2]) {
		p.rect(tx - 8, wallY - 22, 16, 26, [16, 18, 22]);
		p.rect(tx - 10, wallY - 25, 4, 4, [16, 18, 22]);
		p.rect(tx - 2, wallY - 25, 4, 4, [16, 18, 22]);
		p.rect(tx + 6, wallY - 25, 4, 4, [16, 18, 22]);
		p.rect(tx - 1, wallY - 14, 2, 5, GOLD); // lit arrow slit
	}
	// spears & tattered banners in foreground
	for (let i = 0; i < 9; i++) {
		const bx = Math.round(w * 0.08 + i * (w * 0.1) + (r() - 0.5) * 6);
		const lean = Math.round((r() - 0.5) * 8);
		const topY = wallY - 12 - Math.round(r() * 14);
		p.line(bx, h - 1, bx + lean, topY, INK);
		if (i % 3 === 1) {
			const flagCol = i === 4 ? accent : CRIMSON;
			for (let fy = 0; fy < 7; fy++) {
				const flen = 10 - (fy > 4 ? (fy - 3) * 2 : 0) + ((fy + i) % 2);
				p.rect(bx + lean + 1, topY + 2 + fy, flen, 1, flagCol);
			}
		}
	}
	// flames along wall base + drifting sparks
	for (let i = 0; i < 24; i++) {
		const fx = (r() * w) | 0;
		const fy = wallY + 4 + ((r() * (h - wallY - 8)) | 0);
		p.ditherGlow(fx, fy, 4 + r() * 5, i % 2 ? EMBER : GOLD, 0.7);
	}
	for (let i = 0; i < 42; i++) {
		const sx = (r() * w) | 0;
		const sy = (r() * (h - 6)) | 0;
		p.px(sx, sy, r() < 0.4 ? PALE : EMBER);
	}
}

function paintDelve(p: Painter, r: () => number, accent: RGB) {
	const { w, h } = p;
	// deep cavernous stone backdrop
	p.ditherBand(0, h, [12, 22, 26], [6, 12, 14]);
	const cx = Math.round(w * 0.5);
	const archW = Math.round(w * 0.24);
	const archTop = Math.round(h * 0.16);
	// chasm opening inside archway
	for (let y = archTop; y < h; y++) {
		for (let x = cx - archW; x <= cx + archW; x++) {
			const dx = (x - cx) / archW;
			const dy = y < archTop + 20 ? (y - (archTop + 20)) / 20 : 0;
			if (dx * dx + dy * dy <= 1) {
				const depth = (y - archTop) / Math.max(1, h - archTop);
				p.px(x, y, mixRgb([6, 14, 18], [16, 44, 48], depth * 0.45));
			}
		}
	}
	// descending stone steps into the abyss
	for (let s = 0; s < 6; s++) {
		const sy = Math.round(h * 0.56 + s * 5);
		const sw = Math.round(archW * (0.45 + s * 0.12));
		p.rect(cx - sw, sy, sw * 2, 2, mixRgb(DEEP_TEAL, SEA_SLATE, 0.25 + s * 0.08));
		p.rect(cx - sw, sy + 2, sw * 2, 3, INK);
	}
	// carved stone voussoirs around the arch
	for (let a = 0; a <= 18; a++) {
		const ang = Math.PI + (a / 18) * Math.PI;
		const rx = cx + Math.cos(ang) * (archW + 3);
		const ry = archTop + 20 + Math.sin(ang) * 22;
		p.rect(rx - 2, ry - 2, 5, 4, a % 2 ? DEEP_TEAL : [28, 66, 68]);
	}
	// flanking pillars
	p.rect(cx - archW - 10, archTop + 20, 10, h - archTop - 20, DEEP_TEAL);
	p.rect(cx + archW, archTop + 20, 10, h - archTop - 20, DEEP_TEAL);
	p.rect(cx - archW - 2, archTop + 20, 2, h - archTop - 20, SEA_SLATE, 0.45);
	p.rect(cx + archW, archTop + 20, 2, h - archTop - 20, SEA_SLATE, 0.45);
	// twin flickering wall sconces + warm torchlight pools
	const ty = Math.round(h * 0.46);
	for (const tx of [cx - archW - 5, cx + archW + 5]) {
		p.ditherGlow(tx, ty - 2, 24, EMBER, 0.65);
		p.ditherGlow(tx, ty - 3, 10, GOLD, 0.85);
		p.rect(tx - 1, ty, 2, 7, PARCHMENT);
		p.rect(tx - 1, ty - 3, 2, 3, PALE);
	}
	// mysterious abyssal glow / crystal glint at bottom of chasm
	p.ditherGlow(cx, h - 6, 16, accent, 0.55);
	// hanging roots / stalactites
	for (let i = 0; i < 12; i++) {
		const sx = Math.round(8 + r() * (w - 16));
		const len = 4 + ((r() * 12) | 0);
		p.rect(sx, 0, 2, len, INK);
	}
}

function paintVoyage(p: Painter, r: () => number, accent: RGB) {
	const { w, h } = p;
	const seaY = Math.round(h * 0.54);
	p.ditherBand(0, seaY, [9, 18, 28], [22, 54, 68]);
	drawStars(p, r, 38, seaY - 6, PALE, SEA_SLATE);
	// moon & reflection path
	const mx = Math.round(w * (0.2 + r() * 0.25));
	const my = Math.round(seaY * 0.38);
	p.ditherGlow(mx, my, 18, SEA_SLATE, 0.55);
	p.ditherGlow(mx, my, 5, PALE, 1);
	// sea base
	p.ditherBand(seaY, h, [14, 46, 58], [8, 22, 28]);
	// moon shimmer across waves
	for (let y = seaY + 2; y < h; y += 2) {
		const spread = 4 + (y - seaY) * 0.45;
		const sx = mx + Math.round((r() - 0.5) * spread);
		const sw = 3 + ((r() * 6) | 0);
		p.rect(sx, y, sw, 1, y < seaY + 10 ? PALE : SEA_SLATE, 0.7);
	}
	// ship silhouette (caravel / longship)
	const bx = Math.round(w * 0.6);
	const by = seaY + 4;
	// hull
	for (let dy = 0; dy < 7; dy++) {
		const hw = 16 - dy * 1.4;
		p.rect(bx - hw, by + dy, hw * 2, 1, INK);
	}
	p.line(bx - 16, by, bx - 22, by - 5, INK); // prow
	p.line(bx + 15, by, bx + 19, by - 4, INK); // stern
	// mast & billowing sail
	p.rect(bx - 1, by - 25, 2, 25, INK);
	for (let sy = 0; sy < 17; sy++) {
		const curve = Math.sin((sy / 16) * Math.PI) * 5;
		const left = bx - 8 + curve * 0.3;
		const right = bx + 9 + curve;
		p.rect(left, by - 22 + sy, right - left, 1, sy % 5 === 2 ? mixRgb(PARCHMENT, accent, 0.35) : PARCHMENT);
	}
	// lantern at stern
	p.px(bx + 17, by - 4, GOLD);
	p.ditherGlow(bx + 17, by - 4, 8, GOLD, 0.65);
	// foreground rolling wave crests
	for (let band = 0; band < 3; band++) {
		const wy = seaY + 8 + band * 8;
		const col = band === 2 ? INK : mixRgb(DEEP_TEAL, MID_TEAL, 0.3 * band);
		for (let x = 0; x < w; x++) {
			const crest = Math.round(Math.sin(x * 0.09 + band * 2.1) * 2.2);
			p.px(x, wy + crest, SEA_SLATE, 0.55);
			p.rect(x, wy + crest + 1, 1, 3, col);
		}
	}
}

function paintShrine(p: Painter, r: () => number, accent: RGB) {
	const { w, h } = p;
	const floorY = Math.round(h * 0.72);
	p.ditherBand(0, floorY, [12, 22, 30], [26, 58, 62]);
	drawStars(p, r, 22, floorY - 14, PALE, SEA_SLATE);
	const cx = Math.round(w * 0.5);
	// slanted celestial light beams
	for (let b = -1; b <= 1; b++) {
		const topX = cx + b * 14 - 12;
		const botX = cx + b * 18 + 8;
		for (let y = 0; y < floorY + 6; y++) {
			const t = y / (floorY + 6);
			const xMid = topX + (botX - topX) * t;
			const half = 3 + t * 9;
			const xHi = (xMid + half) | 0;
			for (let x = (xMid - half) | 0; x <= xHi; x++) {
				const bay = BAYER[((y & 3) << 2) | (x & 3)] / 16;
				const intensity = (1 - Math.abs(x - xMid) / half) * (b === 0 ? 0.65 : 0.38);
				if (intensity > bay) p.px(x, y, mixRgb(GOLD, accent, 0.35), 0.55);
			}
		}
	}
	// floating halo / sigil ring above the altar
	const hy = Math.round(h * 0.34);
	p.ditherGlow(cx, hy, 22, mixRgb(GOLD, accent, 0.4), 0.7);
	for (let a = 0; a < 48; a++) {
		const ang = (a / 48) * Math.PI * 2;
		p.px(cx + Math.cos(ang) * 11, hy + Math.sin(ang) * 11, PALE);
		if (a % 2 === 0) p.px(cx + Math.cos(ang) * 7, hy + Math.sin(ang) * 7, GOLD, 0.8);
	}
	p.rect(cx - 1, hy - 3, 2, 6, PALE);
	p.rect(cx - 3, hy - 1, 6, 2, PALE);
	// ground mist & stepped stone altar
	p.rect(0, floorY, w, h - floorY, [14, 30, 34]);
	p.rect(cx - 28, floorY + 4, 56, 4, DEEP_TEAL);
	p.rect(cx - 20, floorY, 40, 4, MID_TEAL);
	p.rect(cx - 12, floorY - 8, 24, 8, SEA_SLATE);
	p.rect(cx - 14, floorY - 10, 28, 2, PARCHMENT);
	// omen motes
	for (let i = 0; i < 24; i++) {
		const mx = cx + Math.round((r() - 0.5) * w * 0.7);
		const my = Math.round(6 + r() * (floorY - 4));
		p.px(mx, my, r() < 0.5 ? GOLD : PALE);
	}
}

function paintCrown(p: Painter, r: () => number, accent: RGB) {
	const { w, h } = p;
	const horizon = Math.round(h * 0.66);
	p.ditherBand(0, horizon, [16, 30, 40], mixRgb(MID_TEAL, GOLD, 0.35));
	const cx = Math.round(w * 0.5);
	const sy = Math.round(h * 0.42);
	// geometric sunburst rays behind the citadel
	for (let i = 0; i < 16; i++) {
		const ang = (i / 16) * Math.PI * 2;
		const len = i % 2 === 0 ? 52 : 34;
		p.line(cx, sy, cx + Math.cos(ang) * len, sy + Math.sin(ang) * len, mixRgb(GOLD, accent, 0.25), 0.5);
	}
	p.ditherGlow(cx, sy, 28, GOLD, 0.85);
	p.ditherGlow(cx, sy, 10, PALE, 1);
	// mountain ridge
	p.ridge(horizon, 7, 0.05, r() * 6, DEEP_TEAL, r() * 10);
	p.rect(0, horizon + 4, w, h - horizon - 4, INK);
	// soaring citadel spires
	const spires: [number, number, number][] = [
		[cx - 26, 20, 7],
		[cx - 15, 28, 8],
		[cx, 38, 10],
		[cx + 15, 27, 8],
		[cx + 26, 19, 7],
	];
	for (const [sx, sh, sw] of spires) {
		const top = horizon + 4 - sh;
		p.rect(sx - (sw >> 1), top + 7, sw, sh, INK);
		// pointed roof
		for (let ry = 0; ry < 8; ry++) {
			const rw = Math.max(1, Math.round((ry / 7) * sw));
			p.rect(sx - (rw >> 1), top + ry, rw, 1, INK);
		}
		// banner pennant
		p.line(sx, top - 4, sx, top, PARCHMENT);
		p.rect(sx + 1, top - 4, 4, 2, accent);
		// lit golden windows
		for (let wy = top + 11; wy < horizon + 1; wy += 6) {
			p.px(sx, wy, GOLD);
		}
	}
	// golden triumph motes
	for (let i = 0; i < 20; i++) {
		p.px((r() * w) | 0, (r() * horizon) | 0, r() < 0.5 ? GOLD : PALE);
	}
}

function paintDeath(p: Painter, r: () => number, accent: RGB) {
	const { w, h } = p;
	const knollY = Math.round(h * 0.68);
	p.ditherBand(0, knollY, [9, 14, 18], [20, 38, 44]);
	drawStars(p, r, 32, knollY - 12, PALE, SEA_SLATE);
	// pale moon with quiet halo
	const mx = Math.round(w * 0.26);
	const my = Math.round(h * 0.26);
	p.ditherGlow(mx, my, 20, SEA_SLATE, 0.5);
	for (let y = my - 6; y <= my + 6; y++) {
		for (let x = mx - 6; x <= mx + 6; x++) {
			if (Math.hypot(x - mx, y - my) <= 5.5 && Math.hypot(x - (mx + 3), y - (my - 1)) > 4.8) {
				p.px(x, y, PALE);
			}
		}
	}
	// quiet knoll silhouette
	for (let x = 0; x < w; x++) {
		const dx = (x - w * 0.55) / (w * 0.55);
		const hill = Math.round(knollY + dx * dx * 12);
		for (let y = hill; y < h; y++) p.px(x, y, y === hill ? DEEP_TEAL : INK);
	}
	// weathered headstone & small stone cairn
	const gx = Math.round(w * 0.56);
	const gy = knollY - 13;
	p.rect(gx - 5, gy + 3, 11, 14, [34, 54, 56]);
	p.rect(gx - 4, gy + 1, 9, 2, [44, 68, 70]);
	p.rect(gx - 2, gy, 5, 1, SEA_SLATE);
	// faint carved mark
	p.rect(gx, gy + 5, 1, 5, PARCHMENT, 0.7);
	p.rect(gx - 2, gy + 7, 5, 1, PARCHMENT, 0.7);
	// bare twisted bough entering from right
	p.line(w - 1, knollY - 10, gx + 20, knollY - 22, INK);
	p.line(gx + 34, knollY - 16, gx + 26, knollY - 29, INK);
	p.line(gx + 26, knollY - 20, gx + 14, knollY - 24, INK);
	// drifting ash & petals
	for (let i = 0; i < 26; i++) {
		const ax = (r() * w) | 0;
		const ay = (r() * (h - 6)) | 0;
		const c = i % 4 === 0 ? mixRgb(ROSE, accent, 0.4) : PALE;
		p.px(ax, ay, c, 0.75);
		if (i % 5 === 0) p.px(ax + 1, ay + 1, c, 0.4);
	}
}

function drawFrame(p: Painter, accent: RGB) {
	const { w, h } = p;
	const edge = mixRgb(DEEP_TEAL, accent, 0.25);
	for (let x = 0; x < w; x++) {
		p.px(x, 0, edge);
		p.px(x, h - 1, edge);
	}
	for (let y = 0; y < h; y++) {
		p.px(0, y, edge);
		p.px(w - 1, y, edge);
	}
	// corner ornaments
	const c = mixRgb(GOLD, accent, 0.4);
	for (const [cx, cy, dx, dy] of [
		[2, 2, 1, 1],
		[w - 3, 2, -1, 1],
		[2, h - 3, 1, -1],
		[w - 3, h - 3, -1, -1],
	]) {
		p.px(cx, cy, c);
		p.px(cx + dx * 2, cy, c, 0.7);
		p.px(cx, cy + dy * 2, c, 0.7);
	}
}

/**
 * Draw a deterministic procedural retro-storybook vignette onto `canvas`.
 */
export function drawVignette(canvas: HTMLCanvasElement, opts: VignetteOpts): void {
	const ctx = canvas.getContext('2d');
	if (!ctx) return;
	const artW = opts.artW ?? 180;
	const artH = opts.artH ?? 76;
	const scale = 2;
	canvas.width = artW * scale;
	canvas.height = artH * scale;

	const r = mulberry32(hash32(`${opts.seed}|${opts.kind}|${opts.title ?? ''}`));
	const accent = hexToRgb(opts.accent) ?? GOLD;
	const p = makePainter(artW, artH);

	switch (opts.kind) {
		case 'origin':
		case 'journey':
			paintOrigin(p, r, accent);
			break;
		case 'battle':
		case 'siege':
			paintBattle(p, r, accent);
			break;
		case 'delve':
		case 'ruin':
			paintDelve(p, r, accent);
			break;
		case 'voyage':
		case 'sea':
			paintVoyage(p, r, accent);
			break;
		case 'shrine':
		case 'miracle':
			paintShrine(p, r, accent);
			break;
		case 'crown':
		case 'triumph':
			paintCrown(p, r, accent);
			break;
		case 'death':
		case 'epitaph':
			paintDeath(p, r, accent);
			break;
	}
	drawFrame(p, accent);

	const off = document.createElement('canvas');
	off.width = artW;
	off.height = artH;
	const octx = off.getContext('2d');
	if (!octx) return;
	const img = octx.createImageData(artW, artH);
	img.data.set(p.data);
	octx.putImageData(img, 0, 0);

	ctx.imageSmoothingEnabled = false;
	ctx.clearRect(0, 0, canvas.width, canvas.height);
	ctx.drawImage(off, 0, 0, canvas.width, canvas.height);
}
