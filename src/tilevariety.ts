// Tile variety for sprite mode: deterministic per-tile variants (a hash of the tile's position, never
// Math.random), wall autotiling from the neighbour mask, genre/season tints, and the ambient animated
// tiles (water, lava, waterfalls, tall grass, trees, braziers, portals). Everything is painted into a
// small offscreen cache keyed by glyph + colours + variant + mask + animation frame, so a repaint of
// the whole map is a few hundred drawImage calls.
//
// It is a layer on top of sprites.ts: drawTileVariant() returns
//   TILE_NONE (0)     nothing to add: call drawProceduralCell as before
//   TILE_OVERLAY (1)  call drawProceduralCell first, then this was added on top (ground detail, wall edges)
//   TILE_REPLACED (2) the whole tile is drawn (animated kinds): skip drawProceduralCell
// Animation timing and the prefs live in spriteanim.ts.

import { tileAnimFrame, tileOptions, TILE_FRAMES } from './spriteanim';

type Ctx2 = CanvasRenderingContext2D | OffscreenCanvasRenderingContext2D;
type Cv = HTMLCanvasElement | OffscreenCanvas;

export const TILE_NONE = 0;
export const TILE_OVERLAY = 1;
export const TILE_REPLACED = 2;

// neighbour mask bits (a set bit = that neighbour is "the same kind": solid wall, or water)
export const N = 1, E = 2, S = 4, W = 8, NE = 16, SE = 32, SW = 64, NW = 128;

export interface TileContext {
	/** per-zone salt, e.g. (music.seed ^ depth * 7919): variants then differ between zones at the same x,y */
	salt?: number;
	/** frame.music.genre, tints the ground */
	genre?: string;
	/** 0 spring .. 3 winter (frame.music.season) */
	season?: number;
	/** -1 | 0 | 1 (frame.fx.wind): bends grass and trees */
	wind?: number;
	/** true when indoors/underground (no season tint, no snow) */
	indoors?: boolean;
}
let ctxInfo: TileContext = {};
let tintKey = '';
let tintCol = '';
let tintA = 0;

const GENRE_TINTS: [RegExp, string, number][] = [
	[/cyber|neon|synth/, '#00e5ff', 0.07],
	[/steam|brass|clock/, '#c98a2a', 0.07],
	[/dark|grim|gothic|horror|twilight/, '#6a2a8a', 0.08],
	[/waste|apoc|fall|dust/, '#b0703a', 0.08],
	[/fae|fairy|magic|myth/, '#7a4fe0', 0.06],
	[/sci|space|star|solar|atomic/, '#3a7bff', 0.06],
	[/bio|vat|flesh/, '#40c060', 0.06],
];
const SEASON_TINTS: [string, number][] = [['#8aff8a', 0.04], ['', 0], ['#d08030', 0.08], ['#a0c8ff', 0.09]];

/** Per frame (cheap): give the zone context. */
export function setTileContext(c: TileContext): void {
	ctxInfo = c;
	const g = (c.genre ?? '').toLowerCase();
	let col = '';
	let a = 0;
	for (const [re, cc, aa] of GENRE_TINTS) if (re.test(g)) { col = cc; a = aa; break; }
	if (!c.indoors && c.season != null && SEASON_TINTS[c.season]?.[1]) {
		if (!col) { col = SEASON_TINTS[c.season][0]; a = SEASON_TINTS[c.season][1]; }
		else { a += 0.03; }
	}
	tintCol = col;
	tintA = a;
	tintKey = col ? col + a.toFixed(2) : '';
}

// ---- hashing ---------------------------------------------------------------------------------

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

// ---- classification --------------------------------------------------------------------------

type Kind =
	| 'wall' | 'grass' | 'sand' | 'snow' | 'floor' | 'rough' // static
	| 'water' | 'lava' | 'fall' | 'tallgrass' | 'tree' | 'brazier' | 'portal'; // animated
const ANIMATED = new Set<Kind>(['water', 'lava', 'fall', 'tallgrass', 'tree', 'brazier', 'portal']);
const GROUND = new Set(['.', '∙', '·', ':']);
const WALLISH = '#%⊞∏╫▲';
const TREES = '♣♠↑τ';

export interface TileOpts {
	/** cell height when it differs from `size` (the map cell is about 0.6 as wide as tall) */
	chh?: number;
	/** palette indices of the cell's colours (the engine's, from frame.f / frame.b) */
	fgIdx?: number;
	bgIdx?: number;
	/** palette, for a few accent colours; optional */
	palette?: string[];
}

function classify(g: string, fgIdx: number, bgIdx: number): Kind | null {
	if (g === '#' || g === '%') return 'wall';
	if (g === '~' || g === '≈') {
		if (bgIdx === 10 || ((fgIdx === 16 || fgIdx === 17) && g === '≈')) return 'lava';
		if (bgIdx === 4 || (g === '≈' && (fgIdx === 4 || fgIdx === 5 || fgIdx === 8 || fgIdx === 9))) return 'water';
		return null; // an item (trophy, lockpick) on the floor
	}
	if (g === '║') return 'fall';
	if (g === '"') return 'tallgrass';
	if (TREES.includes(g) && fgIdx !== 12 && fgIdx !== 13) return 'tree';
	if (g === 'Ψ') return 'brazier';
	if (g === 'Ø') return 'portal';
	if (GROUND.has(g)) {
		if (fgIdx === 6 || fgIdx === 7) return 'grass';
		if (fgIdx === 14 || fgIdx === 15) return 'sand';
		if (fgIdx === 9 || (fgIdx === 3 && ctxInfo.season === 3 && !ctxInfo.indoors)) return 'snow';
		if (fgIdx === 1 || fgIdx === 2 || fgIdx === 0) return 'floor';
		return 'rough';
	}
	return null;
}

/** True for glyph/colour pairs that animate: Play keeps a list of these visible cells and repaints them on each tickTileAnim() frame. */
export function isAnimatedTile(g: string, fgIdx: number, bgIdx = 0): boolean {
	const k = classify(g, fgIdx, bgIdx);
	return !!k && ANIMATED.has(k);
}

// ---- neighbour masks -------------------------------------------------------------------------

/** What counts as "the same" for autotiling a glyph: wall-like for walls, water for water. */
export function neighbourKind(g: string): 'wall' | 'water' | null {
	if (g === '#' || g === '%') return 'wall';
	if (g === '~' || g === '≈') return 'water';
	return null;
}

/**
 * Neighbour mask for the cell (gx, gy) of a glyph grid `glyphs` (w x h, row-major). Bits N E S W NE SE SW NW
 * are set where the neighbour is the same kind; off the grid counts as set. Pass the cell's own glyph.
 */
export function neighbourMask(glyphs: string[], w: number, h: number, gx: number, gy: number, g: string): number {
	const kind = neighbourKind(g);
	if (!kind) return 0;
	const same = (dx: number, dy: number): boolean => {
		const x = gx + dx, y = gy + dy;
		if (x < 0 || y < 0 || x >= w || y >= h) return true;
		const n = glyphs[y * w + x];
		return kind === 'wall' ? WALLISH.includes(n) && n !== ' ' : n === '~' || n === '≈';
	};
	return (same(0, -1) ? N : 0) | (same(1, 0) ? E : 0) | (same(0, 1) ? S : 0) | (same(-1, 0) ? W : 0) |
		(same(1, -1) ? NE : 0) | (same(1, 1) ? SE : 0) | (same(-1, 1) ? SW : 0) | (same(-1, -1) ? NW : 0);
}

// ---- cache -----------------------------------------------------------------------------------

const cache = new Map<string, Cv>();
const CACHE_MAX = 3000;

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
		/* fall through */
	}
	return null;
}

/** Drop every cached tile (palette change, resize is handled by the key). */
export function clearTileCache(): void {
	cache.clear();
}

// ---- main entry ------------------------------------------------------------------------------

/**
 * Paint the variant layer of a tile at (x, y). `size` is the cell width (`o.chh` the height, default size).
 * (wx, wy) is the tile position in the zone (hash seed); `neighbours` the mask from neighbourMask() (0 if unused).
 * Returns TILE_NONE / TILE_OVERLAY / TILE_REPLACED, see the top of the file.
 */
export function drawTileVariant(
	ctx: Ctx2,
	glyph: string,
	fg: string,
	bg: string,
	x: number,
	y: number,
	size: number,
	wx: number,
	wy: number,
	neighbours: number,
	o: TileOpts = {},
): number {
	const opt = tileOptions();
	if (!opt.variety && !opt.anim) return TILE_NONE;
	const fgIdx = o.fgIdx ?? -1;
	const bgIdx = o.bgIdx ?? -1;
	const kind = classify(glyph, fgIdx, bgIdx);
	if (!kind) return TILE_NONE;
	const animated = ANIMATED.has(kind);
	if (!animated && !opt.variety) return TILE_NONE;

	const w = size;
	const h = o.chh ?? size;
	const hh = hash(wx, wy, ctxInfo.salt ?? 0);
	const frame = animated && opt.anim && !opt.reduced ? tileAnimFrame() : 0;
	const wind = ctxInfo.wind ?? 0;

	// the variant id seeds the painter, so the cache holds a few dozen looks per glyph, not one per tile
	let vid = animated ? Math.floor(rnd(hh, 99) * 32) : Math.floor(rnd(hh, 99) * 48);
	if (kind === 'floor') vid += 48 * (hash(wx >> 3, wy >> 2, ctxInfo.salt ?? 0) & 3); // coherent patches: planks / flags / plain
	const f8 = frame % TILE_FRAMES;
	const mask = kind === 'wall' || kind === 'water' ? neighbours & 255 : 0;
	let scale = 1;
	try {
		scale = Math.max(1, Math.round((ctx as CanvasRenderingContext2D).getTransform().a * 4) / 4);
	} catch {
		/* ignore */
	}
	const tint = kind === 'grass' || kind === 'sand' || kind === 'snow' || kind === 'floor' || kind === 'rough' ? tintKey : '';
	const key = `${kind}|${glyph}|${fg}|${bg}|${vid}|${mask}|${f8}|${wind}|${w.toFixed(1)}x${h.toFixed(1)}@${scale}|${tint}`;

	let cv = cache.get(key);
	if (!cv) {
		const pw = Math.max(1, Math.ceil(w * scale));
		const ph = Math.max(1, Math.ceil(h * scale));
		const c = makeCanvas(pw, ph);
		if (!c) {
			// no offscreen canvas: paint straight onto the target
			ctx.save();
			ctx.translate(x, y);
			paint(ctx, kind, glyph, fg, bg, w, h, vid, mask, f8, wind, tint ? tintCol : '', tintA, o);
			ctx.restore();
			return animated ? TILE_REPLACED : TILE_OVERLAY;
		}
		const cc = c.getContext('2d') as Ctx2;
		cc.scale(pw / w, ph / h);
		paint(cc, kind, glyph, fg, bg, w, h, vid, mask, f8, wind, tint ? tintCol : '', tintA, o);
		if (cache.size >= CACHE_MAX) cache.clear();
		cache.set(key, c);
		cv = c;
	}
	ctx.drawImage(cv as CanvasImageSource, x, y, w, h);
	return animated ? TILE_REPLACED : TILE_OVERLAY;
}

// ---- painters (local coordinates, 0..w x 0..h) -----------------------------------------------

const TAU = Math.PI * 2;
const FLOWERS = ['#ff8fb8', '#ffe066', '#ffffff', '#c8a0ff', '#ff9a5a'];

function paint(c: Ctx2, kind: Kind, g: string, fg: string, bg: string, w: number, h: number, vid: number, mask: number, f: number, wind: number, tint: string, tintAlpha: number, o: TileOpts) {
	const s = Math.imul(vid + 1, 2654435761) >>> 0; // painter seed
	c.save();
	c.beginPath();
	c.rect(0, 0, w, h);
	c.clip();
	c.lineCap = 'round';
	c.lineJoin = 'round';
	const lw = Math.max(1, w * 0.1);
	if (tint) {
		c.globalAlpha = tintAlpha;
		c.fillStyle = tint;
		c.fillRect(0, 0, w, h);
		c.globalAlpha = 1;
	}
	switch (kind) {
		case 'grass': ground(c, 'grass', fg, w, h, s, lw, vid); break;
		case 'sand': ground(c, 'sand', fg, w, h, s, lw, vid); break;
		case 'snow': ground(c, 'snow', fg, w, h, s, lw, vid); break;
		case 'floor': ground(c, 'floor', fg, w, h, s, lw, vid); break;
		case 'rough': ground(c, 'rough', fg, w, h, s, lw, vid); break;
		case 'wall': wall(c, fg, bg, w, h, s, lw, mask); break;
		case 'water': water(c, g, fg, w, h, s, lw, mask, f, false); break;
		case 'lava': water(c, g, fg, w, h, s, lw, mask, f, true); break;
		case 'fall': fall(c, fg, w, h, s, lw, f); break;
		case 'tallgrass': tallGrass(c, fg, w, h, s, lw, f, wind); break;
		case 'tree': tree(c, g, fg, w, h, s, f, wind); break;
		case 'brazier': brazier(c, fg, w, h, s, f, o); break;
		case 'portal': portal(c, fg, w, h, s, lw, f); break;
	}
	c.restore();
}

function ground(c: Ctx2, k: 'grass' | 'sand' | 'snow' | 'floor' | 'rough', fg: string, w: number, h: number, s: number, lw: number, vid: number) {
	const r = rnd(s, 0);
	c.fillStyle = fg;
	c.strokeStyle = fg;
	c.lineWidth = Math.max(1, w * 0.07);
	if (k === 'grass') {
		if (r < 0.34) {
			// a tuft
			c.globalAlpha = 0.85;
			const bx = w * (0.25 + rnd(s, 1) * 0.5), by = h * (0.7 + rnd(s, 2) * 0.15);
			for (let i = -1; i <= 1; i++) {
				c.beginPath();
				c.moveTo(bx + i * w * 0.07, by);
				c.quadraticCurveTo(bx + i * w * 0.12, by - h * 0.1, bx + i * w * 0.2, by - h * (0.14 + rnd(s, 3 + i) * 0.06));
				c.stroke();
			}
		} else if (r < 0.5) {
			// flowers
			const n = 1 + Math.floor(rnd(s, 1) * 3);
			for (let i = 0; i < n; i++) {
				c.fillStyle = FLOWERS[Math.floor(rnd(s, 5 + i) * FLOWERS.length)];
				const fx = w * (0.2 + rnd(s, 8 + i) * 0.6), fy = h * (0.35 + rnd(s, 12 + i) * 0.4);
				c.globalAlpha = 1;
				c.fillRect(fx, fy, Math.max(2, w * 0.14), Math.max(2, w * 0.14));
				c.fillStyle = '#ffe9a0';
				c.fillRect(fx + w * 0.04, fy + w * 0.04, Math.max(1, w * 0.06), Math.max(1, w * 0.06));
			}
		} else if (r < 0.62) {
			// moss patch
			c.globalAlpha = 0.28;
			c.beginPath();
			c.ellipse(w * (0.3 + rnd(s, 1) * 0.4), h * (0.5 + rnd(s, 2) * 0.3), w * 0.3, h * 0.13, 0, 0, TAU);
			c.fill();
		} else if (r < 0.72) {
			pebbles(c, w, h, s, '#9a9a9a', 0.7);
		}
	} else if (k === 'sand') {
		c.globalAlpha = 0.55;
		if (r < 0.45) {
			for (let i = 0; i < 2; i++) {
				const ry = h * (0.35 + i * 0.28 + rnd(s, 1) * 0.08);
				c.beginPath();
				c.moveTo(w * 0.1, ry);
				c.quadraticCurveTo(w * 0.5, ry - h * (0.07 + rnd(s, 2 + i) * 0.04), w * 0.9, ry);
				c.stroke();
			}
		} else if (r < 0.7) {
			pebbles(c, w, h, s, fg, 0.7);
		} else if (r < 0.8) {
			c.fillStyle = '#fff';
			c.globalAlpha = 0.5;
			c.beginPath();
			c.arc(w * (0.3 + rnd(s, 1) * 0.4), h * 0.7, w * 0.07, 0, TAU);
			c.fill();
		}
	} else if (k === 'snow') {
		c.fillStyle = '#ffffff';
		if (r < 0.5) {
			// a drift: lit crest, shaded foot
			const cx = w * (0.3 + rnd(s, 1) * 0.4), cy = h * (0.62 + rnd(s, 2) * 0.1);
			c.globalAlpha = 0.55;
			c.beginPath();
			c.ellipse(cx, cy, w * 0.34, h * 0.1, 0, Math.PI, TAU);
			c.fill();
			c.globalAlpha = 0.25;
			c.fillStyle = '#6890c0';
			c.fillRect(cx - w * 0.3, cy, w * 0.6, Math.max(1, h * 0.03));
		} else if (r < 0.65) {
			crack(c, w, h, s, '#9ac0e8', 0.6);
		} else if (r < 0.8) {
			c.globalAlpha = 0.9;
			c.fillRect(w * (0.25 + rnd(s, 1) * 0.5), h * (0.3 + rnd(s, 2) * 0.4), 2, 2);
		}
	} else if (k === 'floor') {
		const patch = Math.floor(vid / 48); // 0 planks, 1..3 flagstones / plain
		if (patch === 0) {
			// planks with seams and a knot
			c.globalAlpha = 0.28;
			const rows = 3;
			for (let i = 1; i < rows; i++) {
				const ry = (h * i) / rows;
				c.beginPath();
				c.moveTo(0, ry);
				c.lineTo(w, ry);
				c.stroke();
				const sx = w * (0.2 + rnd(s, i) * 0.6);
				c.beginPath();
				c.moveTo(sx, ry - h / rows);
				c.lineTo(sx, ry);
				c.stroke();
			}
			if (rnd(s, 9) < 0.35) {
				c.globalAlpha = 0.4;
				c.beginPath();
				c.ellipse(w * (0.25 + rnd(s, 10) * 0.5), h * (0.2 + rnd(s, 11) * 0.6), w * 0.05, h * 0.03, 0, 0, TAU);
				c.stroke();
			}
		} else if (patch === 1 || patch === 2) {
			// flagstones, one offset seam row, a worn corner
			c.globalAlpha = 0.25;
			const sy = h * (0.45 + rnd(s, 1) * 0.1);
			c.beginPath();
			c.moveTo(0, sy);
			c.lineTo(w, sy);
			c.moveTo(w * (0.3 + rnd(s, 2) * 0.3), 0);
			c.lineTo(w * (0.3 + rnd(s, 2) * 0.3), sy);
			c.moveTo(w * (0.15 + rnd(s, 3) * 0.5), sy);
			c.lineTo(w * (0.15 + rnd(s, 3) * 0.5), h);
			c.stroke();
			if (rnd(s, 4) < 0.3) crack(c, w, h, s, fg, 0.35);
		}
		if (rnd(s, 20) < 0.07) puddle(c, w, h, s);
		else if (rnd(s, 21) < 0.2) {
			c.globalAlpha = 0.2;
			c.fillStyle = '#000';
			c.beginPath();
			c.ellipse(w * rnd(s, 22), h * (0.3 + rnd(s, 23) * 0.5), w * 0.18, h * 0.06, 0.3, 0, TAU);
			c.fill();
		}
	} else {
		// rough ground: stone, dirt, ash
		if (r < 0.25) pebbles(c, w, h, s, fg, 0.65);
		else if (r < 0.4) crack(c, w, h, s, fg, 0.5);
		else if (r < 0.48) {
			c.fillStyle = '#4c8a3c';
			c.globalAlpha = 0.35;
			c.beginPath();
			c.ellipse(w * (0.3 + rnd(s, 1) * 0.4), h * (0.6 + rnd(s, 2) * 0.2), w * 0.26, h * 0.1, 0, 0, TAU);
			c.fill();
		} else if (r < 0.53) puddle(c, w, h, s);
	}
}

function pebbles(c: Ctx2, w: number, h: number, s: number, col: string, a: number) {
	c.fillStyle = col;
	c.globalAlpha = a;
	const n = 2 + Math.floor(rnd(s, 30) * 2);
	for (let i = 0; i < n; i++) {
		const px = w * (0.15 + rnd(s, 31 + i) * 0.7), py = h * (0.4 + rnd(s, 35 + i) * 0.45);
		c.beginPath();
		c.ellipse(px, py, Math.max(1.2, w * 0.07), Math.max(1, w * 0.05), 0, 0, TAU);
		c.fill();
	}
}
function crack(c: Ctx2, w: number, h: number, s: number, col: string, a: number) {
	c.strokeStyle = col;
	c.globalAlpha = a;
	c.lineWidth = 1;
	let px = w * (0.15 + rnd(s, 40) * 0.2), py = h * (0.2 + rnd(s, 41) * 0.2);
	c.beginPath();
	c.moveTo(px, py);
	for (let i = 0; i < 4; i++) {
		px += w * (0.12 + rnd(s, 42 + i) * 0.1);
		py += h * (rnd(s, 46 + i) - 0.35) * 0.3 + h * 0.1;
		c.lineTo(px, py);
	}
	c.stroke();
}
function puddle(c: Ctx2, w: number, h: number, s: number) {
	const px = w * (0.35 + rnd(s, 50) * 0.3), py = h * (0.55 + rnd(s, 51) * 0.2);
	c.globalAlpha = 0.5;
	c.fillStyle = '#3a6aa8';
	c.beginPath();
	c.ellipse(px, py, w * 0.26, h * 0.1, 0, 0, TAU);
	c.fill();
	c.globalAlpha = 0.6;
	c.fillStyle = '#cfe4ff';
	c.fillRect(px - w * 0.12, py - h * 0.03, w * 0.12, Math.max(1, h * 0.025));
}

function wall(c: Ctx2, fg: string, bg: string, w: number, h: number, s: number, lw: number, m: number) {
	const open = (b: number) => !(m & b);
	const e = Math.max(1, w * 0.1);
	// lit edges where the wall meets open ground (top and left catch light, bottom and right fall into shade)
	if (open(N)) { c.globalAlpha = 0.45; c.fillStyle = '#fff'; c.fillRect(0, 0, w, e); }
	if (open(W)) { c.globalAlpha = 0.3; c.fillStyle = '#fff'; c.fillRect(0, 0, e, h); }
	if (open(S)) { c.globalAlpha = 0.4; c.fillStyle = '#000'; c.fillRect(0, h - e * 1.4, w, e * 1.4); }
	if (open(E)) { c.globalAlpha = 0.3; c.fillStyle = '#000'; c.fillRect(w - e, 0, e, h); }
	// outer corners: round them off with the ground colour
	const rr = Math.min(w, h) * 0.26;
	c.globalAlpha = 1;
	c.fillStyle = bg;
	const cap = (cx: number, cy: number, a0: number) => {
		c.beginPath();
		c.moveTo(cx, cy);
		c.arc(cx, cy, rr, a0, a0 + Math.PI / 2);
		c.closePath();
		c.fill();
	};
	if (open(N) && open(W)) { c.beginPath(); c.moveTo(0, 0); c.lineTo(rr, 0); c.arc(rr, rr, rr, -Math.PI / 2, Math.PI, true); c.closePath(); c.fill(); }
	if (open(N) && open(E)) { c.beginPath(); c.moveTo(w, 0); c.lineTo(w, rr); c.arc(w - rr, rr, rr, 0, -Math.PI / 2, true); c.closePath(); c.fill(); }
	if (open(S) && open(W)) { c.beginPath(); c.moveTo(0, h); c.lineTo(0, h - rr); c.arc(rr, h - rr, rr, Math.PI, Math.PI / 2, true); c.closePath(); c.fill(); }
	if (open(S) && open(E)) { c.beginPath(); c.moveTo(w, h); c.lineTo(w - rr, h); c.arc(w - rr, h - rr, rr, Math.PI / 2, 0, true); c.closePath(); c.fill(); }
	void cap;
	// inner corners: both sides solid, the diagonal open: a shaded notch
	c.fillStyle = '#000';
	c.globalAlpha = 0.35;
	const q = Math.min(w, h) * 0.22;
	if (!open(N) && !open(W) && open(NW)) c.fillRect(0, 0, q, q);
	if (!open(N) && !open(E) && open(NE)) c.fillRect(w - q, 0, q, q);
	if (!open(S) && !open(W) && open(SW)) c.fillRect(0, h - q, q, q);
	if (!open(S) && !open(E) && open(SE)) c.fillRect(w - q, h - q, q, q);
	// a lone pillar gets a cap ring
	if ((m & 15) === 0) {
		c.globalAlpha = 0.5;
		c.strokeStyle = '#fff';
		c.lineWidth = 1;
		c.beginPath();
		c.ellipse(w / 2, h * 0.4, w * 0.3, h * 0.14, 0, 0, TAU);
		c.stroke();
	}
	// wear: crack or moss, by variant
	const r = rnd(s, 0);
	if (r < 0.14) crack(c, w, h, s, '#000', 0.55);
	else if (r < 0.24 && open(S)) {
		c.globalAlpha = 0.4;
		c.fillStyle = '#4c8a3c';
		c.beginPath();
		c.ellipse(w * (0.25 + rnd(s, 1) * 0.5), h * 0.86, w * 0.22, h * 0.07, 0, 0, TAU);
		c.fill();
	}
}

function water(c: Ctx2, g: string, fg: string, w: number, h: number, s: number, lw: number, m: number, f: number, lava: boolean) {
	const ph = (f + Math.floor(rnd(s, 0) * 8)) % 8;
	const a = (ph / 8) * TAU;
	if (lava) {
		// the whole tile breathes
		c.globalAlpha = 0.22 + 0.12 * Math.sin(a);
		c.fillStyle = '#ff8a1a';
		c.fillRect(0, 0, w, h);
		c.globalAlpha = 0.35;
		c.fillStyle = '#2a0800';
		c.beginPath();
		c.ellipse(w * (0.3 + rnd(s, 1) * 0.4), h * (0.3 + rnd(s, 2) * 0.4), w * 0.2, h * 0.1, 0, 0, TAU); // crust
		c.fill();
	} else {
		c.globalAlpha = 0.16;
		c.fillStyle = fg;
		c.fillRect(0, 0, w, h);
	}
	c.strokeStyle = lava ? '#ffd24a' : fg;
	c.lineWidth = Math.max(1, w * 0.1);
	const rows = g === '≈' ? 3 : 2;
	for (let r = 0; r < rows; r++) {
		const wy = h * (rows === 3 ? 0.2 + r * 0.3 : 0.3 + r * 0.36) + (lava ? Math.sin(a + r) * h * 0.02 : 0);
		const dir = r % 2 ? -1 : 1;
		const sh = a * dir + r * 1.7; // waves drift sideways: flow
		c.globalAlpha = r === 0 ? 0.85 : 0.55;
		c.beginPath();
		for (let i = 0; i <= 6; i++) {
			const px = (w * i) / 6;
			const py = wy + Math.sin((i / 6) * TAU + sh) * h * 0.06;
			if (i) c.lineTo(px, py);
			else c.moveTo(px, py);
		}
		c.stroke();
	}
	if (lava) {
		// bubbles that swell and pop
		for (let k = 0; k < 2; k++) {
			const t = ((ph + k * 4) % 8) / 8;
			if (t < 0.75) {
				c.globalAlpha = 0.8 * (1 - t);
				c.fillStyle = '#ffe28a';
				c.beginPath();
				c.arc(w * (0.25 + rnd(s, 3 + k) * 0.5), h * (0.3 + rnd(s, 5 + k) * 0.45), 0.5 + t * w * 0.1, 0, TAU);
				c.fill();
			}
		}
	} else {
		// glints
		c.fillStyle = '#fff';
		for (let k = 0; k < 2; k++) {
			if ((ph + k * 3) % 8 < 2) {
				c.globalAlpha = 0.9;
				c.fillRect(w * (0.15 + rnd(s, 3 + k) * 0.65), h * (0.2 + rnd(s, 6 + k) * 0.6), Math.max(2, w * 0.14), Math.max(1, h * 0.04));
			}
		}
	}
	// shore: foam on sides that are not water
	if (!lava) {
		c.fillStyle = '#fff';
		const t = Math.max(1, w * 0.08);
		const foam = (x0: number, y0: number, len: number, horiz: boolean) => {
			c.globalAlpha = 0.5 + 0.2 * Math.sin(a + x0 + y0);
			const n = 4;
			for (let i = 0; i < n; i++) {
				const o = ((i + (ph % 2) * 0.5) / n) * len;
				if (horiz) c.fillRect(x0 + o, y0, len / n * 0.55, t);
				else c.fillRect(x0, y0 + o, t, len / n * 0.55);
			}
		};
		if (!(m & N)) foam(0, 0, w, true);
		if (!(m & S)) foam(0, h - t, w, true);
		if (!(m & W)) foam(0, 0, h, false);
		if (!(m & E)) foam(w - t, 0, h, false);
	}
}

function fall(c: Ctx2, fg: string, w: number, h: number, s: number, lw: number, f: number) {
	c.globalAlpha = 0.2;
	c.fillStyle = fg;
	c.fillRect(w * 0.1, 0, w * 0.8, h);
	c.strokeStyle = '#fff';
	c.lineWidth = Math.max(1, w * 0.08);
	const period = h * 0.45;
	c.setLineDash([h * 0.25, h * 0.2]);
	for (let i = 0; i < 4; i++) {
		c.lineDashOffset = -((f + i * 2 + Math.floor(rnd(s, i) * 4)) % 8) * (period / 8);
		c.globalAlpha = 0.5 + (i % 2) * 0.3;
		c.beginPath();
		c.moveTo(w * (0.2 + i * 0.2), 0);
		c.lineTo(w * (0.2 + i * 0.2), h);
		c.stroke();
	}
	c.setLineDash([]);
}

function tallGrass(c: Ctx2, fg: string, w: number, h: number, s: number, lw: number, f: number, wind: number) {
	c.strokeStyle = fg;
	c.lineWidth = Math.max(1, w * 0.11);
	const a = (f / 8) * TAU;
	for (let k = 0; k < 5; k++) {
		const bx = w * (0.12 + k * 0.19);
		const base = (rnd(s, k) - 0.5) * w * 0.12 + wind * w * 0.08;
		const sway = Math.sin(a + k * 0.9 + rnd(s, 9) * 3) * w * (0.1 + Math.abs(wind) * 0.06);
		const topY = h * (0.16 + rnd(s, 10 + k) * 0.14);
		c.globalAlpha = 0.7 + (k % 2) * 0.3;
		c.beginPath();
		c.moveTo(bx, h * 0.9);
		c.quadraticCurveTo(bx + base * 0.4 + sway * 0.3, h * 0.55, bx + base + sway, topY);
		c.stroke();
	}
}

function tree(c: Ctx2, g: string, fg: string, w: number, h: number, s: number, f: number, wind: number) {
	const a = (f / 8) * TAU;
	const sway = Math.sin(a + rnd(s, 0) * 6) * w * 0.05 + wind * w * 0.04;
	const cx = w / 2;
	// trunk
	c.strokeStyle = '#6b4a2a';
	c.lineWidth = Math.max(1.5, w * 0.14);
	c.globalAlpha = 1;
	c.beginPath();
	c.moveTo(cx, h * 0.92);
	c.lineTo(cx + sway * 0.4, h * 0.5);
	c.stroke();
	c.fillStyle = fg;
	if (g === '♠') {
		// conifer: three tiers
		for (let t = 0; t < 3; t++) {
			const ty = h * (0.62 - t * 0.2), half = w * (0.38 - t * 0.09), off = sway * (0.5 + t * 0.35);
			c.beginPath();
			c.moveTo(cx + off, ty - h * 0.26);
			c.lineTo(cx + off + half, ty);
			c.lineTo(cx + off - half, ty);
			c.closePath();
			c.fill();
		}
	} else if (g === '↑') {
		c.beginPath();
		c.ellipse(cx + sway, h * 0.36, w * 0.2, h * 0.3, 0, 0, TAU);
		c.fill();
	} else if (g === 'τ') {
		// bare branches
		c.strokeStyle = '#8a6a44';
		c.lineWidth = Math.max(1, w * 0.07);
		for (let i = -1; i <= 1; i++) {
			c.beginPath();
			c.moveTo(cx + sway * 0.4, h * 0.55);
			c.lineTo(cx + sway + i * w * 0.3, h * 0.2 + Math.abs(i) * h * 0.08);
			c.stroke();
		}
	} else {
		// round canopy of three lobes
		for (const [ox, oy, r] of [[0, 0.34, 0.3], [-0.2, 0.46, 0.22], [0.2, 0.46, 0.22]] as const) {
			c.beginPath();
			c.ellipse(cx + sway * (1.2 - oy) + ox * w, h * oy, w * r, h * r * 0.9, 0, 0, TAU);
			c.fill();
		}
		c.fillStyle = '#fff';
		c.globalAlpha = 0.18;
		c.beginPath();
		c.ellipse(cx + sway - w * 0.1, h * 0.26, w * 0.12, h * 0.08, 0, 0, TAU);
		c.fill();
	}
}

function brazier(c: Ctx2, fg: string, w: number, h: number, s: number, f: number, o: TileOpts) {
	const cx = w / 2;
	// glow
	const flick = 0.75 + 0.25 * rnd(s + f * 977, 1);
	const gr = c.createRadialGradient(cx, h * 0.45, 0, cx, h * 0.45, w * 0.9);
	gr.addColorStop(0, fg);
	gr.addColorStop(1, 'rgba(0,0,0,0)');
	c.globalAlpha = 0.28 * flick;
	c.fillStyle = gr;
	c.fillRect(0, 0, w, h);
	// bowl
	c.globalAlpha = 1;
	c.fillStyle = o.palette?.[2] ?? '#999';
	c.fillRect(w * 0.22, h * 0.64, w * 0.56, h * 0.2);
	c.fillRect(w * 0.34, h * 0.84, w * 0.08, h * 0.1);
	c.fillRect(w * 0.58, h * 0.84, w * 0.08, h * 0.1);
	// flame tongues, each flickering on its own
	c.fillStyle = fg;
	for (let t = 0; t < 3; t++) {
		const k = rnd(s + f * 131 + t * 17, 2 + t);
		const tx = cx + (t - 1) * w * 0.17 + (k - 0.5) * w * 0.08;
		const ht = h * (t === 1 ? 0.5 : 0.34) * (0.7 + k * 0.5);
		c.beginPath();
		c.moveTo(tx - w * 0.12, h * 0.66);
		c.quadraticCurveTo(tx - w * 0.1, h * 0.66 - ht * 0.5, tx + (k - 0.5) * w * 0.1, h * 0.66 - ht);
		c.quadraticCurveTo(tx + w * 0.1, h * 0.66 - ht * 0.5, tx + w * 0.12, h * 0.66);
		c.closePath();
		c.fill();
	}
	c.fillStyle = '#ffe9a0';
	c.globalAlpha = 0.9;
	c.beginPath();
	c.ellipse(cx, h * 0.6, w * 0.09, h * (0.1 + rnd(s + f * 7, 9) * 0.06), 0, 0, TAU);
	c.fill();
	if (rnd(s + f * 53, 11) < 0.4) {
		c.globalAlpha = 0.8;
		c.fillStyle = '#ffd060';
		c.fillRect(cx + (rnd(s + f, 12) - 0.5) * w * 0.5, h * (0.1 + rnd(s + f * 3, 13) * 0.2), 1.5, 1.5); // spark
	}
}

function portal(c: Ctx2, fg: string, w: number, h: number, s: number, lw: number, f: number) {
	const cx = w / 2, cy = h / 2;
	const a = (f / 8) * TAU;
	c.fillStyle = fg;
	c.globalAlpha = 0.18 + 0.1 * Math.sin(a);
	c.beginPath();
	c.ellipse(cx, cy, w * 0.42, h * 0.44, 0, 0, TAU);
	c.fill();
	c.strokeStyle = fg;
	c.lineWidth = Math.max(1, w * 0.09);
	for (let i = 0; i < 3; i++) {
		const rx = w * (0.4 - i * 0.12), ry = h * (0.42 - i * 0.12);
		const dir = i % 2 ? -1 : 1;
		c.globalAlpha = 0.95 - i * 0.2;
		c.beginPath();
		c.ellipse(cx, cy, rx, ry, 0, a * dir + i * 2, a * dir + i * 2 + Math.PI * 1.3);
		c.stroke();
	}
	c.fillStyle = '#fff';
	c.globalAlpha = 0.6 + 0.3 * Math.sin(a * 2);
	c.beginPath();
	c.arc(cx, cy, Math.max(1, w * 0.07), 0, TAU);
	c.fill();
}
