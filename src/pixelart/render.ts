// Draws pixel sprites (types.ts) into a map cell with nearest-neighbour scaling.
import { PW, PH, type Rows, type PCtx } from './types';

type RGB = [number, number, number];

function parse(c: string | undefined, d: RGB): RGB {
	if (!c) return d;
	if (c[0] === '#') {
		let h = c.slice(1);
		if (h.length === 3) h = h.split('').map((x) => x + x).join('');
		const n = parseInt(h, 16);
		if (!Number.isNaN(n)) return [(n >> 16) & 255, (n >> 8) & 255, n & 255];
	}
	const m = c.match(/(\d+)[ ,]+(\d+)[ ,]+(\d+)/);
	return m ? [+m[1], +m[2], +m[3]] : d;
}
const mix = (a: RGB, b: RGB, t: number): RGB => [a[0] + (b[0] - a[0]) * t, a[1] + (b[1] - a[1]) * t, a[2] + (b[2] - a[2]) * t];
const dark = (a: RGB, t: number) => mix(a, [8, 6, 14], t);
const light = (a: RGB, t: number) => mix(a, [255, 252, 240], t);

const FIXED: Record<string, RGB> = {
	w: [150, 98, 52], W: [92, 58, 32], t: [214, 188, 128], T: [160, 132, 84], g: [140, 144, 150], G: [78, 82, 92],
	m: [200, 206, 214], M: [112, 118, 130], e: [86, 160, 70], E: [38, 92, 52], u: [60, 120, 200], U: [28, 58, 128],
	c: [120, 220, 230], r: [200, 56, 52], R: [112, 28, 34], y: [236, 196, 68], Y: [160, 120, 30], x: [244, 244, 240],
	z: [20, 18, 24], p: [140, 80, 190], n: [232, 130, 40], q: [236, 150, 180], v: [110, 230, 100]
};

function lookup(ch: string, p: PCtx): RGB | null {
	const fg = parse(p.fg, [200, 200, 200]);
	switch (ch) {
		case '.': return null;
		case 'o': return dark(parse(p.bg, [10, 10, 14]), 0.7);
		case 'D': return dark(fg, 0.62);
		case 'd': return dark(fg, 0.34);
		case 'F': return fg;
		case 'l': return light(fg, 0.32);
		case 'L': return light(fg, 0.62);
		case 'k': return parse(p.skin, [214, 160, 120]);
		case 'K': return dark(parse(p.skin, [214, 160, 120]), 0.3);
		case 'j': return light(parse(p.skin, [214, 160, 120]), 0.28);
		case 'h': return parse(p.hair, [90, 56, 34]);
		case 'H': return dark(parse(p.hair, [90, 56, 34]), 0.4);
		case 'a': return parse(p.accent, [170, 60, 60]);
		case 'A': return dark(parse(p.accent, [170, 60, 60]), 0.4);
		default: return FIXED[ch] ?? null;
	}
}

const cache = new Map<string, HTMLCanvasElement>();

function compile(rows: Rows, p: PCtx): HTMLCanvasElement {
	const key = rows.join('') + '|' + p.fg + '|' + p.bg + '|' + (p.skin ?? '') + '|' + (p.hair ?? '') + '|' + (p.accent ?? '');
	let cv = cache.get(key);
	if (cv) return cv;
	if (cache.size > 3000) cache.clear();
	cv = document.createElement('canvas');
	cv.width = PW;
	cv.height = PH;
	const c = cv.getContext('2d')!;
	const img = c.createImageData(PW, PH);
	for (let y = 0; y < PH; y++) {
		const row = rows[y] ?? '';
		for (let x = 0; x < PW; x++) {
			const col = lookup(row[x] ?? '.', p);
			if (!col) continue;
			const i = (y * PW + x) * 4;
			img.data[i] = col[0];
			img.data[i + 1] = col[1];
			img.data[i + 2] = col[2];
			img.data[i + 3] = 255;
		}
	}
	c.putImageData(img, 0, 0);
	cache.set(key, cv);
	return cv;
}

const silCache = new Map<string, { cv: HTMLCanvasElement; low: number; x0: number; x1: number }>();

/** the sprite as a flat black silhouette, plus its lowest opaque row and that row's extent (for the contact shadow) */
function silhouette(rows: Rows): { cv: HTMLCanvasElement; low: number; x0: number; x1: number } {
	const key = rows.join('');
	let s = silCache.get(key);
	if (s) return s;
	if (silCache.size > 3000) silCache.clear();
	const cv = document.createElement('canvas');
	cv.width = PW;
	cv.height = PH;
	const c = cv.getContext('2d')!;
	const img = c.createImageData(PW, PH);
	let low = -1, x0 = PW, x1 = -1;
	for (let y = 0; y < PH; y++) {
		const row = rows[y] ?? '';
		for (let x = 0; x < PW; x++) {
			if ((row[x] ?? '.') === '.') continue;
			const i = (y * PW + x) * 4;
			img.data[i + 3] = 255;
			if (y > low) {
				low = y;
				x0 = x;
				x1 = x;
			} else if (y === low) {
				if (x < x0) x0 = x;
				if (x > x1) x1 = x;
			}
		}
	}
	c.putImageData(img, 0, 0);
	s = { cv, low, x0, x1 };
	silCache.set(key, s);
	return s;
}

export type Shadow = 'none' | 'drop' | 'stand';

/** blit a sprite into the css rect of one map cell, pixelated, bottom-centred.
 *  shadow: 'drop' = a flat shadow thrown down-right (trees, props); 'stand' = drop + a contact shadow under the feet (actors). */
export function blitSprite(ctx: CanvasRenderingContext2D, rows: Rows, x: number, y: number, cw: number, chh: number, p: PCtx, shadow: Shadow = 'none'): void {
	const dpr = window.devicePixelRatio || 1;
	let s = Math.min((cw * dpr) / PW, (chh * dpr) / PH);
	if (s >= 2) s = Math.floor(s); // whole device pixels per art pixel when there is room
	const dw = (PW * s) / dpr, dh = (PH * s) / dpr;
	const prev = ctx.imageSmoothingEnabled;
	ctx.imageSmoothingEnabled = false;
	const dx = x + (cw - dw) / 2, dy = y + (chh - dh), px = dw / PW;
	if (shadow !== 'none') {
		const sil = silhouette(rows);
		ctx.save();
		ctx.beginPath();
		ctx.rect(x, y, cw, chh);
		ctx.clip();
		if (shadow === 'stand' && sil.low >= 0) {
			// contact shadow: a flat dark patch under the feet, one art pixel wider each side
			ctx.globalAlpha = 0.5;
			ctx.fillStyle = '#000';
			const row = Math.min(PH - 1, sil.low + 1);
			ctx.fillRect(dx + (sil.x0 - 1) * px, dy + row * px, (sil.x1 - sil.x0 + 3) * px, px);
		}
		ctx.globalAlpha = 0.38;
		ctx.drawImage(sil.cv, dx + px, dy + px, dw, dh); // thrown down and to the right (light from the top left)
		ctx.restore();
	}
	ctx.drawImage(compile(rows, p), dx, dy, dw, dh);
	ctx.imageSmoothingEnabled = prev;
}
