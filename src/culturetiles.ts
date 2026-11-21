// Culture-specific building tiles for sprite mode. A thin layer over drawProceduralCell/drawTileVariant: for the
// wall, roof, door, window, gate, floor and fence glyphs of a town it paints brick bonding, timber framing, adobe
// plaster, bamboo canes, tile rows, lintel carvings, lanterns, banners and so on in the culture's colours, from the
// style the engine reports (`frame.style`, engine/styleinfo.go StyleInfo). Deterministic: a hash of the tile's
// position, never Math.random. Painted into small offscreen canvases keyed by everything that changes the look, so a
// repaint of the map is a handful of drawImage calls. Everything is static, so reduced motion needs nothing.
//
// Return values match tilevariety.ts: TILE_NONE when nothing was added, TILE_OVERLAY when drawn on top of the base
// cell (call this after drawProceduralCell and drawTileVariant).

import { TILE_NONE, TILE_OVERLAY, N, E, S, W, NE, NW } from './tilevariety';

type Ctx2 = CanvasRenderingContext2D | OffscreenCanvasRenderingContext2D;
type Cv = HTMLCanvasElement | OffscreenCanvas;

/** engine StyleLook as sent in the frame (`frame.style`) */
export interface CultureStyle {
	culture: number;
	sphere?: string;
	genre?: string;
	town: boolean;
	/** primary, secondary: stone timber adobe brick marble bone ironwork bamboo thatch glass obsidian coral ice hide hedge */
	materials: string[];
	accent: string;
	wallTint: string;
	roofTint: string;
	/** tile thatch slate dome pagoda flat sod sails */
	roof: string;
	/** lintel banner lantern totem gargoyle mural gear neon vine */
	ornament: string;
	seed: number;
}

let enabled = true;
/** Pref `cultureTiles` (default true). `townOnly` (default true) limits the look to town zones. */
export function setCultureTileOptions(o: { enabled?: boolean; townOnly?: boolean }): void {
	if (o.enabled != null) enabled = o.enabled;
	if (o.townOnly != null) townOnly = o.townOnly;
}
let townOnly = true;

const cache = new Map<string, Cv>();
const CACHE_MAX = 1500;
export function clearCultureTileCache(): void {
	cache.clear();
}

// ---- helpers ---------------------------------------------------------------------------------

function hash(x: number, y: number, salt: number): number {
	let h = (Math.imul(x | 0, 374761393) + Math.imul(y | 0, 668265263) + Math.imul(salt | 0, 1442695041)) | 0;
	h = Math.imul(h ^ (h >>> 13), 1274126177);
	return (h ^ (h >>> 16)) >>> 0;
}
function rnd(h: number, k: number): number {
	let v = Math.imul(h ^ Math.imul(k + 1, 0x9e3779b1), 0x85ebca6b);
	v ^= v >>> 15;
	v = Math.imul(v, 0xc2b2ae35);
	v ^= v >>> 13;
	return (v >>> 0) / 4294967296;
}
function rgb(hex: string): [number, number, number] {
	const m = /^#?([0-9a-f]{6})$/i.exec(hex || '');
	if (!m) return [128, 128, 128];
	const n = parseInt(m[1], 16);
	return [(n >> 16) & 255, (n >> 8) & 255, n & 255];
}
function rgba(hex: string, a: number): string {
	const [r, g, b] = rgb(hex);
	return `rgba(${r},${g},${b},${a})`;
}
/** hex shaded toward black (k<0) or white (k>0) */
function shade(hex: string, k: number): string {
	const [r, g, b] = rgb(hex);
	const t = k < 0 ? 0 : 255;
	const a = Math.abs(k);
	const f = (v: number) => Math.round(v + (t - v) * a);
	return `rgb(${f(r)},${f(g)},${f(b)})`;
}
function makeCanvas(w: number, h: number): Cv | null {
	try {
		if (typeof OffscreenCanvas !== 'undefined') return new OffscreenCanvas(w, h);
		if (typeof document !== 'undefined') {
			const c = document.createElement('canvas');
			c.width = w;
			c.height = h;
			return c;
		}
	} catch {
		/* ignore */
	}
	return null;
}

type Kind = 'wall' | 'roof' | 'door' | 'window' | 'gate' | 'woodfloor' | 'tilefloor' | 'fence';

const ROOF_GLYPHS = '^Λ⌂';
const FENCE_GLYPHS = '|╪‡';
function classify(g: string, fgIdx: number): Kind | null {
	switch (g) {
		case '#':
			return 'wall';
		case '+':
			return 'door';
		case '⊞':
			return 'window';
		case '∏':
		case '╫':
			return 'gate';
		case '=':
			return 'woodfloor';
		case '·':
			// ground glyph too: leave grass, sand and snow alone
			return fgIdx === 6 || fgIdx === 7 || fgIdx === 14 || fgIdx === 15 || fgIdx === 9 ? null : 'tilefloor';
	}
	if (ROOF_GLYPHS.includes(g)) return 'roof';
	if (FENCE_GLYPHS.includes(g)) return 'fence';
	return null;
}

// ---- painters (local coordinates 0..w x 0..h) --------------------------------------------------

const DARK = 'rgba(0,0,0,0.30)';
const LIGHT = 'rgba(255,255,255,0.16)';

function line(c: Ctx2, x0: number, y0: number, x1: number, y1: number, col: string, lw: number) {
	c.strokeStyle = col;
	c.lineWidth = lw;
	c.beginPath();
	c.moveTo(x0, y0);
	c.lineTo(x1, y1);
	c.stroke();
}

function paintMaterial(c: Ctx2, mat: string, st: CultureStyle, w: number, h: number, mask: number, hv: number) {
	const lw = Math.max(1, w / 9);
	const tint = st.wallTint;
	c.fillStyle = rgba(tint, mat === 'glass' ? 0.25 : 0.34);
	c.fillRect(0, 0, w, h);
	switch (mat) {
		case 'brick': {
			const rows = 5, rh = h / rows;
			for (let r = 0; r < rows; r++) {
				const y = r * rh;
				line(c, 0, y, w, y, DARK, lw * 0.8);
				const off = r % 2 ? w * 0.25 : w * 0.75;
				line(c, off, y, off, y + rh, DARK, lw * 0.8);
				if (rnd(hv, r) < 0.25) {
					c.fillStyle = rgba(shade(tint, -0.3), 0.3);
					c.fillRect(rnd(hv, r + 9) * w * 0.5, y + 0.5, w * 0.45, rh - 1);
				}
			}
			break;
		}
		case 'stone':
		case 'obsidian': {
			const rows = 4, rh = h / rows;
			for (let r = 0; r < rows; r++) {
				const y = r * rh;
				line(c, 0, y, w, y, DARK, lw * 0.8);
				const off = ((r * 3 + (hv & 3)) % 5) / 5 * w + w * 0.15;
				line(c, off % w, y, off % w, y + rh, DARK, lw * 0.8);
				line(c, 0, y + 1, w * 0.5, y + 1, LIGHT, lw * 0.6);
			}
			if (mat === 'obsidian') line(c, w * 0.2, h * 0.9, w * 0.8, h * 0.15, 'rgba(180,140,255,0.28)', lw * 0.7);
			break;
		}
		case 'marble': {
			c.fillStyle = 'rgba(255,255,255,0.12)';
			c.fillRect(0, 0, w, h);
			for (let i = 0; i < 2; i++) {
				const y0 = rnd(hv, i) * h, y1 = rnd(hv, i + 5) * h;
				c.strokeStyle = 'rgba(80,90,110,0.30)';
				c.lineWidth = lw * 0.7;
				c.beginPath();
				c.moveTo(0, y0);
				c.quadraticCurveTo(w * 0.5, (y0 + y1) / 2 + (rnd(hv, i + 11) - 0.5) * h * 0.4, w, y1);
				c.stroke();
			}
			line(c, 0, h * 0.5, w, h * 0.5, DARK, lw * 0.6);
			break;
		}
		case 'timber': {
			// frame: posts at the open sides, a mid beam, a brace, pale infill between
			c.fillStyle = 'rgba(235,220,190,0.18)';
			c.fillRect(0, 0, w, h);
			const beam = shade(st.wallTint, -0.55);
			c.fillStyle = rgba(beam, 0.75);
			const pw = Math.max(1.5, w * 0.2);
			if (!(mask & W)) c.fillRect(0, 0, pw, h);
			if (!(mask & E)) c.fillRect(w - pw, 0, pw, h);
			c.fillRect(0, h * 0.46, w, Math.max(1.5, h * 0.09));
			if (!(mask & N)) c.fillRect(0, 0, w, Math.max(1.5, h * 0.09));
			if (hv & 1) line(c, pw, h * 0.46, w - pw, h * 0.08, rgba(beam, 0.7), lw);
			else line(c, pw, h * 0.08, w - pw, h * 0.46, rgba(beam, 0.7), lw);
			line(c, pw, h * 0.55, w - pw, h * 0.95, rgba(beam, 0.5), lw * 0.8);
			break;
		}
		case 'adobe': {
			c.fillStyle = 'rgba(255,230,190,0.10)';
			c.fillRect(0, 0, w, h);
			for (let i = 0; i < 6; i++) {
				c.fillStyle = rgba(shade(tint, rnd(hv, i) < 0.5 ? -0.3 : 0.3), 0.3);
				c.fillRect(rnd(hv, i + 20) * w, rnd(hv, i + 40) * h, Math.max(1, w * 0.16), Math.max(1, h * 0.05));
			}
			// rounded plaster shoulders at open corners
			const rr = Math.min(w, h) * 0.45;
			c.strokeStyle = rgba(shade(tint, -0.5), 0.45);
			c.lineWidth = lw;
			if (!(mask & N) && !(mask & W)) {
				c.beginPath();
				c.arc(rr, rr, rr, Math.PI, Math.PI * 1.5);
				c.stroke();
			}
			if (!(mask & N) && !(mask & E)) {
				c.beginPath();
				c.arc(w - rr, rr, rr, Math.PI * 1.5, Math.PI * 2);
				c.stroke();
			}
			// roof beam ends sticking out (vigas)
			if (!(mask & N)) {
				c.fillStyle = rgba('#5a3a20', 0.8);
				for (let i = 0; i < 2; i++) c.fillRect(w * (0.2 + i * 0.45), h * 0.1, w * 0.2, Math.max(1, h * 0.05));
			}
			break;
		}
		case 'bamboo': {
			const n = 3;
			for (let i = 0; i < n; i++) {
				const x = ((i + 0.5) * w) / n;
				line(c, x, 0, x, h, rgba(shade(tint, -0.4), 0.55), lw * 1.2);
				line(c, x - lw, 0, x - lw, h, LIGHT, lw * 0.5);
				for (let k = 1; k < 4; k++) {
					const y = ((k + rnd(hv, i * 4 + k) * 0.5) * h) / 4.5;
					line(c, x - lw * 1.2, y, x + lw * 1.2, y, DARK, lw * 0.7);
				}
			}
			line(c, 0, h * 0.5, w, h * 0.5, rgba(shade(tint, -0.55), 0.5), lw * 1.2);
			break;
		}
		case 'bone': {
			c.fillStyle = 'rgba(255,250,230,0.12)';
			c.fillRect(0, 0, w, h);
			for (let i = 0; i < 3; i++) {
				const x = ((i + 0.5) * w) / 3;
				c.strokeStyle = 'rgba(70,55,40,0.45)';
				c.lineWidth = lw;
				c.beginPath();
				c.moveTo(x, 0);
				c.quadraticCurveTo(x + (i % 2 ? 1 : -1) * w * 0.25, h * 0.5, x, h);
				c.stroke();
			}
			c.fillStyle = 'rgba(40,30,20,0.55)';
			c.fillRect(w * 0.45, h * 0.45, Math.max(1, w * 0.12), Math.max(1, h * 0.06));
			break;
		}
		case 'ironwork': {
			c.fillStyle = 'rgba(20,20,30,0.18)';
			c.fillRect(0, 0, w, h);
			line(c, 0, h * 0.5, w, h * 0.5, DARK, lw);
			line(c, w * 0.5, 0, w * 0.5, h, DARK, lw * 0.8);
			c.fillStyle = 'rgba(230,220,200,0.45)';
			const rv = Math.max(1, w * 0.09);
			for (const [x, y] of [[0.14, 0.1], [0.86, 0.1], [0.14, 0.9], [0.86, 0.9], [0.14, 0.5], [0.86, 0.5]]) c.fillRect(x * w - rv / 2, y * h - rv / 2, rv, rv);
			break;
		}
		case 'glass': {
			c.fillStyle = 'rgba(160,220,255,0.12)';
			c.fillRect(0, 0, w, h);
			line(c, 0, h * 0.8, w * 0.7, 0, 'rgba(255,255,255,0.35)', lw);
			line(c, w * 0.3, h, w, h * 0.25, 'rgba(255,255,255,0.22)', lw * 0.8);
			line(c, 0, h * 0.5, w, h * 0.5, 'rgba(20,40,60,0.35)', lw * 0.7);
			break;
		}
		case 'hide': {
			line(c, 0, h * 0.5, w, h * 0.5, DARK, lw * 0.8);
			c.fillStyle = 'rgba(40,25,10,0.5)';
			for (let i = 0; i < 4; i++) c.fillRect(((i + 0.3) * w) / 4, h * 0.5 - 1, Math.max(1, w * 0.1), 2);
			line(c, w * 0.5, 0, w * 0.5, h * 0.5, DARK, lw * 0.6);
			break;
		}
		case 'thatch': {
			for (let i = 0; i < 7; i++) {
				const x = rnd(hv, i) * w;
				line(c, x, 0, x + (rnd(hv, i + 9) - 0.5) * w * 0.4, h, rgba(shade(tint, rnd(hv, i + 3) < 0.5 ? -0.35 : 0.3), 0.45), lw * 0.7);
			}
			break;
		}
		case 'coral': {
			c.fillStyle = rgba(shade(tint, 0.3), 0.35);
			for (let i = 0; i < 4; i++) {
				c.beginPath();
				c.arc(rnd(hv, i) * w, rnd(hv, i + 7) * h, Math.max(1, w * 0.18), 0, Math.PI * 2);
				c.fill();
			}
			break;
		}
		case 'hedge': {
			for (let i = 0; i < 9; i++) {
				c.fillStyle = rgba(shade('#3f7a35', rnd(hv, i + 20) < 0.5 ? -0.3 : 0.25), 0.5);
				c.fillRect(rnd(hv, i) * w, rnd(hv, i + 9) * h, Math.max(1, w * 0.18), Math.max(1, h * 0.07));
			}
			break;
		}
		case 'ice': {
			line(c, 0, h, w, h * 0.3, 'rgba(255,255,255,0.35)', lw);
			line(c, w * 0.4, 0, w, h * 0.6, 'rgba(255,255,255,0.2)', lw * 0.8);
			break;
		}
		default: {
			line(c, 0, h * 0.5, w, h * 0.5, DARK, lw);
		}
	}
}

/** accent trims common to every wall: eave line on top, plinth below, then one ornament chosen by the hash */
function paintWallTrim(c: Ctx2, st: CultureStyle, w: number, h: number, mask: number, hv: number) {
	const lw = Math.max(1, w / 9);
	const sec = st.materials[1] ?? st.materials[0];
	if (!(mask & N)) {
		c.fillStyle = rgba(st.roofTint, 0.8);
		c.fillRect(0, 0, w, Math.max(1.5, h * 0.1));
		c.fillStyle = 'rgba(0,0,0,0.25)';
		c.fillRect(0, Math.max(1.5, h * 0.1), w, Math.max(1, h * 0.04));
	}
	if (!(mask & S)) {
		c.fillStyle = rgba(sec === 'timber' ? '#5a3c22' : shade(st.wallTint, -0.45), 0.75);
		c.fillRect(0, h - Math.max(1.5, h * 0.12), w, Math.max(1.5, h * 0.12));
	}
	const r = rnd(hv, 77);
	switch (st.ornament) {
		case 'lintel':
			if (!(mask & N) || r < 0.3) {
				c.fillStyle = rgba(shade(st.wallTint, 0.4), 0.55);
				const y = h * 0.2;
				for (let i = 0; i < 3; i++) c.fillRect(w * (0.12 + i * 0.3), y, w * 0.18, Math.max(1, h * 0.05));
				c.fillStyle = rgba(st.accent, 0.5);
				c.fillRect(w * 0.1, y + h * 0.08, w * 0.8, Math.max(1, h * 0.03));
			}
			break;
		case 'banner':
			if (r < 0.28) {
				const bx = w * (0.3 + rnd(hv, 5) * 0.2), bw = w * 0.32;
				c.fillStyle = rgba(st.accent, 0.9);
				c.fillRect(bx, h * 0.14, bw, h * 0.5);
				c.fillStyle = 'rgba(0,0,0,0.3)';
				c.beginPath();
				c.moveTo(bx, h * 0.64);
				c.lineTo(bx + bw / 2, h * 0.52);
				c.lineTo(bx + bw, h * 0.64);
				c.fill();
				c.fillStyle = 'rgba(255,255,255,0.4)';
				c.fillRect(bx + bw * 0.35, h * 0.26, bw * 0.3, Math.max(1, h * 0.08));
			}
			break;
		case 'lantern':
			if (r < 0.3 && !(mask & S)) {
				c.fillStyle = 'rgba(255,200,90,0.35)';
				c.beginPath();
				c.arc(w * 0.5, h * 0.55, w * 0.45, 0, Math.PI * 2);
				c.fill();
				c.fillStyle = '#ffd36a';
				c.fillRect(w * 0.5 - lw, h * 0.5, lw * 2, lw * 2.6);
				line(c, w * 0.5, h * 0.3, w * 0.5, h * 0.5, 'rgba(40,20,10,0.7)', lw * 0.6);
			}
			break;
		case 'totem':
			if (r < 0.2) {
				c.fillStyle = rgba(shade(st.accent, -0.2), 0.85);
				c.fillRect(w * 0.38, h * 0.15, w * 0.24, h * 0.7);
				c.fillStyle = 'rgba(255,240,210,0.8)';
				c.fillRect(w * 0.38, h * 0.25, w * 0.24, Math.max(1, h * 0.06));
				c.fillRect(w * 0.38, h * 0.5, w * 0.24, Math.max(1, h * 0.06));
				c.fillStyle = 'rgba(0,0,0,0.7)';
				c.fillRect(w * 0.43, h * 0.3, Math.max(1, w * 0.06), Math.max(1, h * 0.05));
				c.fillRect(w * 0.53, h * 0.3, Math.max(1, w * 0.06), Math.max(1, h * 0.05));
			}
			break;
		case 'gargoyle':
			if (!(mask & N) && (!(mask & W) || !(mask & E)) && r < 0.6) {
				const left = !(mask & W);
				c.fillStyle = 'rgba(40,40,46,0.85)';
				c.beginPath();
				const x0 = left ? 0 : w, dx = left ? 1 : -1;
				c.moveTo(x0, h * 0.1);
				c.lineTo(x0 + dx * w * 0.5, h * 0.2);
				c.lineTo(x0 + dx * w * 0.35, h * 0.38);
				c.lineTo(x0, h * 0.3);
				c.fill();
				c.fillStyle = '#ff6a4a';
				c.fillRect(x0 + dx * w * 0.3 - (left ? 0 : 1), h * 0.2, 1, 1);
			}
			break;
		case 'mural':
			if (r < 0.35) {
				const mx = w * 0.15, my = h * 0.25;
				c.fillStyle = rgba(st.accent, 0.6);
				c.fillRect(mx, my, w * 0.7, h * 0.4);
				c.fillStyle = rgba(shade(st.accent, 0.55), 0.8);
				c.beginPath();
				c.arc(w * 0.38, h * 0.4, Math.max(1, w * 0.12), 0, Math.PI * 2);
				c.fill();
				c.fillStyle = rgba(shade(st.accent, -0.5), 0.8);
				c.fillRect(w * 0.5, h * 0.45, w * 0.28, Math.max(1, h * 0.06));
			}
			break;
		case 'gear':
			if (r < 0.3) {
				const cx = w * 0.5, cy = h * 0.5, rr = Math.min(w, h * 0.5) * 0.32;
				c.strokeStyle = rgba(st.accent, 0.9);
				c.lineWidth = Math.max(1, lw);
				c.beginPath();
				c.arc(cx, cy, rr, 0, Math.PI * 2);
				c.stroke();
				for (let i = 0; i < 6; i++) {
					const a = (i / 6) * Math.PI * 2;
					line(c, cx + Math.cos(a) * rr, cy + Math.sin(a) * rr, cx + Math.cos(a) * rr * 1.5, cy + Math.sin(a) * rr * 1.5, rgba(st.accent, 0.9), lw * 1.3);
				}
			}
			break;
		case 'neon': {
			c.strokeStyle = rgba(st.accent, 0.95);
			c.lineWidth = Math.max(1, lw * 0.9);
			c.shadowColor = st.accent;
			c.shadowBlur = Math.max(2, w * 0.4);
			if (!(mask & N)) {
				c.beginPath();
				c.moveTo(0, h * 0.16);
				c.lineTo(w, h * 0.16);
				c.stroke();
			}
			if (!(mask & W)) {
				c.beginPath();
				c.moveTo(lw, 0);
				c.lineTo(lw, h);
				c.stroke();
			}
			c.shadowBlur = 0;
			break;
		}
		case 'vine':
			if (r < 0.55) {
				c.strokeStyle = 'rgba(70,140,60,0.8)';
				c.lineWidth = lw * 0.8;
				const x = w * (0.2 + rnd(hv, 3) * 0.6);
				c.beginPath();
				c.moveTo(x, 0);
				c.bezierCurveTo(x + w * 0.3, h * 0.3, x - w * 0.3, h * 0.55, x + w * 0.1, h * (0.55 + rnd(hv, 4) * 0.4));
				c.stroke();
				c.fillStyle = 'rgba(110,190,80,0.8)';
				for (let i = 0; i < 3; i++) c.fillRect(x + (rnd(hv, 10 + i) - 0.5) * w * 0.5, h * (0.2 + i * 0.22), Math.max(1, w * 0.12), Math.max(1, w * 0.12));
			}
			break;
	}
}

function paintWall(c: Ctx2, st: CultureStyle, w: number, h: number, mask: number, hv: number) {
	paintMaterial(c, st.materials[0] ?? 'stone', st, w, h, mask, hv);
	paintWallTrim(c, st, w, h, mask, hv);
	// shade where the wall is open to the south-east (a little relief)
	if (!(mask & E)) {
		c.fillStyle = 'rgba(0,0,0,0.12)';
		c.fillRect(w - Math.max(1, w * 0.1), 0, Math.max(1, w * 0.1), h);
	}
}

function paintRoof(c: Ctx2, st: CultureStyle, w: number, h: number, mask: number, hv: number) {
	const lw = Math.max(1, w / 9);
	const t = st.roofTint;
	c.fillStyle = rgba(t, 0.55);
	c.fillRect(0, 0, w, h);
	switch (st.roof) {
		case 'tile':
		case 'pagoda': {
			const rows = 5, rh = h / rows;
			for (let r = 0; r < rows; r++) {
				const y = r * rh;
				line(c, 0, y + rh, w, y + rh, DARK, lw * 0.8);
				const off = r % 2 ? 0 : w / 2;
				c.strokeStyle = rgba(shade(t, -0.4), 0.5);
				c.lineWidth = lw * 0.7;
				for (let x = off; x <= w + 1; x += w) {
					c.beginPath();
					c.arc(x, y, w * 0.5, 0.2, Math.PI - 0.2);
					c.stroke();
				}
			}
			if (st.roof === 'pagoda' && !(mask & S)) {
				c.fillStyle = rgba(shade(t, 0.3), 0.8);
				c.fillRect(0, h - Math.max(1.5, h * 0.1), w, Math.max(1.5, h * 0.1));
			}
			break;
		}
		case 'thatch':
			for (let i = 0; i < 9; i++) {
				const x = rnd(hv, i) * w;
				line(c, x, 0, x + (rnd(hv, i + 9) - 0.5) * w * 0.3, h, rgba(shade(t, rnd(hv, i + 4) < 0.5 ? -0.4 : 0.35), 0.5), lw * 0.7);
			}
			c.fillStyle = 'rgba(0,0,0,0.18)';
			c.fillRect(0, h - Math.max(1.5, h * 0.12), w, Math.max(1.5, h * 0.12));
			break;
		case 'slate': {
			const rows = 4, rh = h / rows;
			for (let r = 0; r < rows; r++) {
				const y = r * rh;
				line(c, 0, y, w, y, 'rgba(0,0,0,0.4)', lw * 0.8);
				const off = ((r * 2 + (hv & 1)) % 3) * (w / 3);
				line(c, off, y, off, y + rh, 'rgba(0,0,0,0.3)', lw * 0.7);
			}
			line(c, 0, h * 0.5, w, h * 0.5, LIGHT, lw * 0.6);
			break;
		}
		case 'dome': {
			c.fillStyle = rgba(shade(t, 0.35), 0.55);
			c.beginPath();
			c.ellipse(w * 0.5, h * 0.55, w * 0.46, h * 0.45, 0, Math.PI, 0);
			c.fill();
			line(c, w * 0.5, h * 0.05, w * 0.5, h * 0.5, DARK, lw * 0.7);
			c.fillStyle = rgba(st.accent, 0.9);
			c.fillRect(w * 0.45, 0, Math.max(1, w * 0.1), Math.max(1, h * 0.12));
			break;
		}
		case 'flat':
			c.fillStyle = 'rgba(255,255,255,0.12)';
			c.fillRect(0, 0, w, h);
			c.strokeStyle = DARK;
			c.lineWidth = lw;
			c.strokeRect(lw * 0.5, lw * 0.5, w - lw, h - lw);
			if (rnd(hv, 3) < 0.4) {
				c.fillStyle = 'rgba(30,30,40,0.5)';
				c.fillRect(w * 0.3, h * 0.35, w * 0.4, h * 0.25);
			}
			break;
		case 'sod':
			for (let i = 0; i < 9; i++) {
				c.fillStyle = rgba(shade('#4a8a3a', rnd(hv, i + 20) < 0.5 ? -0.3 : 0.3), 0.55);
				c.fillRect(rnd(hv, i) * w, rnd(hv, i + 9) * h, Math.max(1, w * 0.16), Math.max(1, h * 0.07));
			}
			break;
		case 'sails': {
			const n = 2;
			for (let i = 0; i < n; i++) {
				const x = (i * w) / n;
				c.strokeStyle = rgba(shade(t, -0.4), 0.55);
				c.lineWidth = lw * 0.8;
				c.beginPath();
				c.moveTo(x, 0);
				c.quadraticCurveTo(x + w / n / 2, h * 0.55, x + w / n, 0);
				c.stroke();
			}
			c.fillStyle = rgba(st.accent, 0.55);
			c.fillRect(0, h - Math.max(1.5, h * 0.14), w, Math.max(1.5, h * 0.14));
			break;
		}
	}
	if (st.ornament === 'lantern' && !(mask & S) && rnd(hv, 6) < 0.4) {
		c.fillStyle = '#ffd36a';
		c.fillRect(w * 0.45, h - lw * 2.5, lw * 1.6, lw * 2.2);
	}
}

function paintDoor(c: Ctx2, st: CultureStyle, w: number, h: number, hv: number) {
	const lw = Math.max(1, w / 9);
	const wood = st.materials[0] === 'ironwork' || st.materials[0] === 'glass' ? '#4a4e58' : st.materials[0] === 'bone' ? '#d8d0b8' : '#6a4426';
	c.fillStyle = rgba(wood, 0.55);
	c.fillRect(w * 0.12, h * 0.12, w * 0.76, h * 0.84);
	for (let i = 1; i < 3; i++) line(c, (w * i) / 3, h * 0.14, (w * i) / 3, h * 0.95, 'rgba(0,0,0,0.3)', lw * 0.7);
	// frame in the accent colour
	c.strokeStyle = rgba(st.accent, 0.9);
	c.lineWidth = Math.max(1.2, lw * 1.2);
	c.strokeRect(w * 0.12, h * 0.12, w * 0.76, h * 0.84);
	// lintel by ornament
	switch (st.ornament) {
		case 'lintel':
			c.fillStyle = rgba(shade(st.wallTint, 0.45), 0.9);
			c.fillRect(w * 0.04, h * 0.04, w * 0.92, Math.max(1.5, h * 0.08));
			c.fillStyle = rgba(st.accent, 0.8);
			c.fillRect(w * 0.3, h * 0.06, w * 0.4, Math.max(1, h * 0.03));
			break;
		case 'banner':
			c.fillStyle = rgba(st.accent, 0.95);
			c.fillRect(w * 0.7, 0, w * 0.2, h * 0.3);
			break;
		case 'lantern':
			c.fillStyle = 'rgba(255,200,90,0.35)';
			c.beginPath();
			c.arc(w * 0.88, h * 0.15, w * 0.3, 0, Math.PI * 2);
			c.fill();
			c.fillStyle = '#ffd36a';
			c.fillRect(w * 0.88 - lw, h * 0.1, lw * 2, lw * 2.6);
			break;
		case 'neon':
			c.strokeStyle = rgba(st.accent, 0.95);
			c.shadowColor = st.accent;
			c.shadowBlur = Math.max(2, w * 0.35);
			c.strokeRect(w * 0.06, h * 0.06, w * 0.88, h * 0.9);
			c.shadowBlur = 0;
			break;
		case 'gargoyle':
		case 'totem':
			c.fillStyle = 'rgba(30,30,36,0.9)';
			c.fillRect(w * 0.02, h * 0.02, w * 0.18, h * 0.2);
			c.fillRect(w * 0.8, h * 0.02, w * 0.18, h * 0.2);
			break;
		case 'gear':
			c.strokeStyle = rgba(st.accent, 0.95);
			c.lineWidth = lw;
			c.beginPath();
			c.arc(w * 0.5, h * 0.3, Math.min(w, h) * 0.14, 0, Math.PI * 2);
			c.stroke();
			break;
		default:
			c.fillStyle = rgba(st.accent, 0.85);
			c.fillRect(w * 0.1, h * 0.05, w * 0.8, Math.max(1.5, h * 0.06));
	}
	c.fillStyle = '#e8d070';
	c.fillRect(w * 0.7, h * 0.55, Math.max(1, lw * 1.2), Math.max(1, lw * 1.2));
}

function paintWindow(c: Ctx2, st: CultureStyle, w: number, h: number, hv: number) {
	const lw = Math.max(1, w / 9);
	const m = st.materials[0];
	const x0 = w * 0.14, x1 = w * 0.86, y0 = h * 0.22, y1 = h * 0.8;
	c.fillStyle = rgba(st.wallTint, 0.4);
	c.fillRect(0, 0, w, h);
	c.fillStyle = 'rgba(255,220,130,0.30)';
	if (m === 'bamboo' || st.roof === 'pagoda') {
		// paper screen with a lattice
		c.fillRect(x0, y0, x1 - x0, y1 - y0);
		for (let i = 1; i < 3; i++) line(c, x0 + ((x1 - x0) * i) / 3, y0, x0 + ((x1 - x0) * i) / 3, y1, rgba('#5a3a20', 0.8), lw * 0.7);
		line(c, x0, (y0 + y1) / 2, x1, (y0 + y1) / 2, rgba('#5a3a20', 0.8), lw * 0.7);
	} else if (st.roof === 'dome' || m === 'adobe') {
		// pointed arch
		c.beginPath();
		c.moveTo(x0, y1);
		c.lineTo(x0, y0 + (y1 - y0) * 0.35);
		c.quadraticCurveTo(w / 2, y0 - (y1 - y0) * 0.35, x1, y0 + (y1 - y0) * 0.35);
		c.lineTo(x1, y1);
		c.closePath();
		c.fill();
		c.strokeStyle = rgba(st.accent, 0.9);
		c.lineWidth = lw;
		c.stroke();
	} else {
		c.fillRect(x0, y0, x1 - x0, y1 - y0);
		line(c, w / 2, y0, w / 2, y1, rgba(shade(st.wallTint, -0.6), 0.8), lw);
		line(c, x0, (y0 + y1) / 2, x1, (y0 + y1) / 2, rgba(shade(st.wallTint, -0.6), 0.8), lw);
		c.strokeStyle = rgba(shade(st.wallTint, -0.6), 0.85);
		c.lineWidth = lw;
		c.strokeRect(x0, y0, x1 - x0, y1 - y0);
	}
	// shutters in the accent colour
	if (st.ornament !== 'neon' && rnd(hv, 8) < 0.7) {
		c.fillStyle = rgba(st.accent, 0.8);
		c.fillRect(0, y0, Math.max(1, w * 0.12), y1 - y0);
		c.fillRect(w - Math.max(1, w * 0.12), y0, Math.max(1, w * 0.12), y1 - y0);
	} else if (st.ornament === 'neon') {
		c.strokeStyle = rgba(st.accent, 0.95);
		c.shadowColor = st.accent;
		c.shadowBlur = Math.max(2, w * 0.35);
		c.lineWidth = lw;
		c.strokeRect(x0, y0, x1 - x0, y1 - y0);
		c.shadowBlur = 0;
	}
	// sill / flower box
	c.fillStyle = rgba(shade(st.wallTint, -0.45), 0.85);
	c.fillRect(w * 0.08, y1, w * 0.84, Math.max(1.5, h * 0.06));
	if (st.ornament === 'vine' || (st.ornament === 'lantern' && rnd(hv, 9) < 0.3)) {
		c.fillStyle = st.ornament === 'vine' ? '#e86a8a' : '#ffd36a';
		for (let i = 0; i < 3; i++) c.fillRect(w * (0.2 + i * 0.25), y1 - lw * 1.5, lw * 1.3, lw * 1.3);
	}
}

function paintGate(c: Ctx2, st: CultureStyle, w: number, h: number, glyph: string) {
	const lw = Math.max(1, w / 9);
	c.fillStyle = rgba(st.wallTint, 0.35);
	c.fillRect(0, 0, w, h);
	c.fillStyle = rgba(st.roofTint, 0.85);
	c.fillRect(0, 0, w, Math.max(1.5, h * 0.14));
	c.fillStyle = rgba(st.accent, 0.85);
	c.fillRect(0, Math.max(1.5, h * 0.14), w, Math.max(1, h * 0.05));
	if (glyph === '╫') {
		for (let i = 1; i < 4; i++) line(c, (w * i) / 4, h * 0.2, (w * i) / 4, h, 'rgba(30,30,36,0.7)', lw);
		line(c, 0, h * 0.55, w, h * 0.55, 'rgba(30,30,36,0.7)', lw);
	}
	if (st.ornament === 'banner' || st.ornament === 'lantern') {
		c.fillStyle = st.ornament === 'banner' ? rgba(st.accent, 0.95) : '#ffd36a';
		c.fillRect(w * 0.06, h * 0.2, Math.max(1.5, w * 0.2), st.ornament === 'banner' ? h * 0.4 : h * 0.14);
		c.fillRect(w * 0.74, h * 0.2, Math.max(1.5, w * 0.2), st.ornament === 'banner' ? h * 0.4 : h * 0.14);
	}
}

function paintFloor(c: Ctx2, st: CultureStyle, w: number, h: number, hv: number, wood: boolean) {
	const lw = Math.max(1, w / 9);
	if (wood) {
		c.fillStyle = rgba(st.materials[0] === 'bamboo' ? '#b9a56a' : '#8a6238', 0.22);
		c.fillRect(0, 0, w, h);
		for (let i = 1; i < 4; i++) line(c, 0, (h * i) / 4, w, (h * i) / 4, 'rgba(0,0,0,0.22)', lw * 0.7);
		const j = rnd(hv, 1) * w;
		line(c, j, 0, j, h / 4, 'rgba(0,0,0,0.22)', lw * 0.7);
		line(c, (j + w / 2) % w, h / 2, (j + w / 2) % w, (3 * h) / 4, 'rgba(0,0,0,0.22)', lw * 0.7);
		return;
	}
	// tile floor: checks in wall and accent tint
	c.fillStyle = rgba(st.wallTint, 0.22);
	c.fillRect(0, 0, w, h);
	const odd = (hv & 1) === 1;
	c.fillStyle = rgba(st.accent, 0.20);
	c.fillRect(odd ? 0 : w / 2, 0, w / 2, h / 2);
	c.fillRect(odd ? w / 2 : 0, h / 2, w / 2, h / 2);
	line(c, 0, h / 2, w, h / 2, 'rgba(0,0,0,0.18)', lw * 0.6);
	line(c, w / 2, 0, w / 2, h, 'rgba(0,0,0,0.18)', lw * 0.6);
}

function paintFence(c: Ctx2, st: CultureStyle, w: number, h: number, hv: number) {
	const lw = Math.max(1, w / 9);
	const m = st.materials[0];
	const col = m === 'ironwork' || m === 'glass' ? '#3a3e48' : m === 'bamboo' ? '#9aa650' : m === 'bone' ? '#d8d0b8' : m === 'stone' || m === 'marble' ? '#8a8a90' : '#7a5434';
	for (let i = 0; i < 3; i++) {
		const x = ((i + 0.5) * w) / 3;
		c.fillStyle = rgba(col, 0.95);
		c.fillRect(x - lw * 0.7, h * 0.18, lw * 1.4, h * 0.78);
		if (m === 'ironwork' || m === 'glass') {
			c.beginPath();
			c.moveTo(x - lw, h * 0.18);
			c.lineTo(x, h * 0.06);
			c.lineTo(x + lw, h * 0.18);
			c.fill();
		} else if (m === 'bone') {
			c.fillStyle = 'rgba(255,255,240,0.9)';
			c.fillRect(x - lw, h * 0.1, lw * 2, lw * 1.6);
		}
	}
	line(c, 0, h * 0.4, w, h * 0.4, rgba(col, 0.95), lw);
	line(c, 0, h * 0.72, w, h * 0.72, rgba(col, 0.95), lw);
	if (st.ornament === 'vine') {
		c.fillStyle = 'rgba(90,170,70,0.85)';
		for (let i = 0; i < 4; i++) c.fillRect(rnd(hv, i) * w, h * (0.3 + rnd(hv, i + 5) * 0.5), Math.max(1, lw * 1.3), Math.max(1, lw * 1.3));
	}
}

// ---- public ----------------------------------------------------------------------------------

/**
 * Add the culture's look over the base sprite of the cell (gx, gy), whose glyph and palette indices are given.
 * (x, y) is the cell's pixel origin and (cw, chh) its size; `nm` is the neighbour mask from tilevariety's
 * neighbourMask (bits N E S W NE SE SW NW, set = a neighbouring wall). `style` is frame.style (undefined or a
 * non-town zone does nothing unless townOnly is off). Returns TILE_OVERLAY when something was drawn, else TILE_NONE.
 */
export function applyCultureTile(
	ctx: Ctx2,
	glyph: string,
	fgIdx: number,
	bgIdx: number,
	x: number,
	y: number,
	cw: number,
	chh: number,
	gx: number,
	gy: number,
	style: CultureStyle | null | undefined,
	nm: number,
): number {
	void bgIdx;
	if (!enabled || !style || (townOnly && !style.town)) return TILE_NONE;
	const kind = classify(glyph, fgIdx);
	if (!kind) return TILE_NONE;
	const hv = hash(gx, gy, style.seed);
	const vid = hv & 15;
	const mask = kind === 'wall' || kind === 'roof' ? nm & 255 : 0;
	let scale = 1;
	try {
		scale = Math.max(1, Math.round((ctx as CanvasRenderingContext2D).getTransform().a * 4) / 4);
	} catch {
		/* ignore */
	}
	const key = `${kind}|${glyph}|${style.materials.join(',')}|${style.wallTint}|${style.roofTint}|${style.accent}|${style.roof}|${style.ornament}|${vid}|${mask}|${cw.toFixed(1)}x${chh.toFixed(1)}@${scale}`;
	let cv = cache.get(key);
	const paint = (c: Ctx2, w: number, h: number) => {
		// the variant id (not the full hash) seeds the painter so the cache stays small
		const v = Math.imul(vid + 1, 0x9e3779b1) >>> 0;
		switch (kind) {
			case 'wall':
				return paintWall(c, style, w, h, mask, v);
			case 'roof':
				return paintRoof(c, style, w, h, mask, v);
			case 'door':
				return paintDoor(c, style, w, h, v);
			case 'window':
				return paintWindow(c, style, w, h, v);
			case 'gate':
				return paintGate(c, style, w, h, glyph);
			case 'woodfloor':
				return paintFloor(c, style, w, h, v, true);
			case 'tilefloor':
				return paintFloor(c, style, w, h, v, false);
			case 'fence':
				return paintFence(c, style, w, h, v);
		}
	};
	if (!cv) {
		const pw = Math.max(1, Math.ceil(cw * scale));
		const ph = Math.max(1, Math.ceil(chh * scale));
		const c = makeCanvas(pw, ph);
		if (!c) {
			ctx.save();
			ctx.translate(x, y);
			paint(ctx, cw, chh);
			ctx.restore();
			return TILE_OVERLAY;
		}
		const cc = c.getContext('2d') as Ctx2;
		cc.scale(pw / cw, ph / chh);
		paint(cc, cw, chh);
		if (cache.size >= CACHE_MAX) cache.clear();
		cache.set(key, c);
		cv = c;
	}
	ctx.drawImage(cv as CanvasImageSource, x, y, cw, chh);
	return TILE_OVERLAY;
}

// keep the unused neighbour bits referenced so the import documents the mask semantic
export const CULTURE_MASK_BITS = { N, E, S, W, NE, NW };
