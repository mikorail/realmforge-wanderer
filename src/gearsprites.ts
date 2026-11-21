// Layered gear over a person sprite: what each person on the map really wears and wields
// (`frame.gear`, engine/gearinfo.go GearLook keyed by actor id). Back pieces (cloak, cape, wings,
// banner) go behind the body (`drawGearBack`, before drawPersonSprite); body garment, feet, hands,
// head, shield, off-hand prop and weapon go over it (`drawGear` after). The body geometry mirrors
// actorsprites.ts drawPersonSprite (species height, width and head size, per-id size variation), so
// the layers sit on the same head, torso, hips and hands. Static layers are cached per gear, build
// and cell size in offscreen canvases; feet and the weapon arm are drawn live so a pose can move them.
// Poses (poses.ts, via GearPose from sprites.ts): the layers go through the same whole-body frame as
// drawPersonSprite (applyBodyFrame: lie down, mirror), the upper layers drop and lean about the hips,
// boots swing with the legs, the weapon arm and the left-hand group (glove, shield, prop) follow the
// posed shoulders, and posed sleeves are drawn live. With no pose (or the rest pose) every layer is
// drawn exactly as before.

export interface GearLook {
	h?: string; // head: helm greathelm hood hat cap circlet crown goggles mask turban bandana
	hc?: string;
	b?: string; // body: rags robe tunic apron coat gambeson leather mail scale lamellar plate kevlar frame carapace
	bc?: string;
	t?: string; // trim
	k?: string; // back: cloak cape quiver pack wings banner
	kc?: string;
	n?: string; // hands: gloves gauntlets
	nc?: string;
	f?: string; // feet: boots greaves sandals
	fc?: string;
	s?: string; // shield: round kite tower buckler riot energy
	sc?: string;
	w?: string; // weapon family
	wc?: string;
	wr?: number; // weapon rarity 0..6
	o?: string; // off-hand prop: purse book symbol lantern orb
	r?: number; // best rarity worn or held
	g?: string; // genre skin
}

/** Body proportions of a species (mirrors actorsprites SPECIES h/w/hd/head and its per-id variation). */
export interface GearBuild {
	h: number;
	w: number;
	hd: number;
	head: 'round' | 'long' | 'wide' | 'skull';
	fish?: boolean;
	arms?: 2 | 4;
	swarm?: boolean;
	vary: number;
}

/** Optional pose from the animation layer. rot: weapon-arm angle in radians about the shoulder, + = hand
 * forward/right (poses.ts limbAngles armR minus its rest 0.18). swing 0..1: an attack arc used when rot
 * is not given. step: walk phase 0..3 (feet lift; used only when legL/legR/liftL/liftR are not given).
 * dx, dy: whole-figure shift in px (dy must not carry the pose drop: pass that as `drop`).
 * The fields below mirror what drawPersonSprite applies for the same pose (sprites.ts builds both from
 * one limbAngles result), so the gear stays on the body:
 * - rotL: left-arm delta, limbAngles armL + 0.18 (same sign convention as rot); the left glove, shield
 *   and off-hand prop ride the posed left hand, and its sleeve is drawn from the shoulder.
 * - lean: torso lean in radians about the hips, + = leans right (canvas rotate(+lean)).
 * - drop: upper-body drop in px (+ = down); the feet stay on the ground line.
 * - legL, legR: leg deltas from rest in radians (+ = foot to the right), liftL, liftR foot lift in cell fractions.
 * - lie, mirror: the whole-body frame (poses.ts applyBodyFrame: lie about the body's middle, mirror about the cell centre). */
export interface GearPose {
	dx?: number;
	dy?: number;
	rot?: number;
	swing?: number;
	step?: number;
	rotL?: number;
	lean?: number;
	drop?: number;
	legL?: number;
	legR?: number;
	liftL?: number;
	liftR?: number;
	lie?: number;
	mirror?: boolean;
}

/** An arm counts as posed (drawn live instead of from the cached rest layers) beyond this delta; idle breathing stays below it. */
const ARM_POSED = 0.05;

/** True when drawGear will draw the posed right (weapon) sleeve itself: withGear then sets `ga` so drawPersonSprite skips its own right arm line. */
export function gearDrawsArm(g: GearLook, pose?: GearPose): boolean {
	const rot = pose?.rot ?? (pose?.swing ? Math.sin(Math.PI * Math.max(0, Math.min(1, pose.swing))) * 1.3 : 0);
	return !!g.b && Math.abs(rot) > ARM_POSED;
}

import { applyBodyFrame } from './poses';

type Ctx = CanvasRenderingContext2D;
type AnyCtx = CanvasRenderingContext2D | OffscreenCanvasRenderingContext2D;
type Hd = GearBuild['head'];

// ---------------------------------------------------------------- build (duplicated from actorsprites)

const SP: Record<string, [number, number, number, Hd, (0 | 1)?, (2 | 4)?, (0 | 1)?]> = {
	human: [0.92, 1, 1, 'round'],
	dwarf: [0.74, 1.32, 1.05, 'wide'],
	elf: [1, 0.78, 0.95, 'long'],
	void_elf: [1, 0.76, 0.95, 'long'],
	halfling: [0.6, 1.1, 1.1, 'round'],
	orc: [1, 1.35, 1, 'wide'],
	goblin: [0.62, 0.92, 1.2, 'long'],
	ogor: [1, 1.55, 0.9, 'wide'],
	troll: [1, 1.4, 0.95, 'long'],
	giant: [1, 1.5, 0.8, 'wide'],
	ratfolk: [0.74, 0.9, 0.95, 'long'],
	beastman: [0.96, 1.2, 1, 'long'],
	lizardfolk: [0.9, 1, 0.95, 'long'],
	saurian: [1, 1.15, 1, 'long'],
	merfolk: [0.9, 0.95, 1, 'round', 1],
	celestial: [1, 0.9, 0.95, 'round'],
	tomb_undead: [0.95, 0.8, 1, 'skull'],
	warp_fiend: [1, 1.05, 1, 'wide', 0, 4],
	chitin_swarm: [0.8, 1.2, 1, 'round', 0, 2, 1],
};

function hash(n: number, salt = 0): number {
	let h = (Math.imul(n + 0x9e3779b1, 0x85ebca6b) ^ Math.imul(salt + 1, 0xc2b2ae35)) >>> 0;
	h = Math.imul(h ^ (h >>> 15), 0x2c1b3c6d) >>> 0;
	h = Math.imul(h ^ (h >>> 12), 0x297a2d39) >>> 0;
	return (h ^ (h >>> 15)) >>> 0;
}
function strHash(s: string): number {
	let h = 2166136261;
	for (let i = 0; i < s.length; i++) h = Math.imul(h ^ s.charCodeAt(i), 16777619);
	return h >>> 0;
}

const buildCache = new Map<string, GearBuild>();

/** The build drawPersonSprite uses for this species and actor id. */
export function gearBuild(species: string | undefined, id: number): GearBuild {
	const sp = species || 'human';
	const key = sp + ':' + id;
	const hit = buildCache.get(key);
	if (hit) return hit;
	const seed = hash(id + 17);
	const vary = 1 + (((seed >>> 20) % 9) - 4) * 0.015;
	const row = SP[sp];
	let b: GearBuild;
	if (row) b = { h: row[0], w: row[1], hd: row[2], head: row[3], fish: !!row[4], arms: row[5] ?? 2, swarm: !!row[6], vary };
	else {
		const h = strHash(sp);
		b = { h: 0.7 + (h % 30) / 100, w: 0.85 + ((h >>> 5) % 50) / 100, hd: 0.9 + ((h >>> 9) % 25) / 100, head: (['round', 'long', 'wide'] as Hd[])[(h >>> 11) % 3], arms: 2, vary };
	}
	if (buildCache.size > 2000) buildCache.clear();
	buildCache.set(key, b);
	return b;
}

/** The idle bob drawPersonSprite adds (same formula); add it to pose.dy so gear moves with the body. */
export function personBob(id: number, gx: number, gy: number, chh: number, tick: number): number {
	const phase = id * 1.37 + gx * 0.7 + gy * 0.5;
	return tick > 0 ? Math.sin(tick * 0.22 + phase) * Math.max(0.6, chh * 0.03) : 0;
}

interface Geo {
	u: number;
	v: number;
	cx: number;
	base: number;
	W: number;
	headY: number;
	neckY: number;
	hipY: number;
	hw: number;
	hh: number;
	lw: number;
	lhx: number; // left (screen) hand
	lhy: number;
	rhx: number; // right hand: the weapon hand
	rhy: number;
	shx: number; // right shoulder
	shy: number;
}

function geo(b: GearBuild, x: number, y: number, u: number, v: number): Geo {
	const cx = x + u * 0.5;
	const base = y + v * 0.9;
	const H = v * 0.82 * b.h * b.vary;
	const W = u * 0.46 * b.w;
	const hr = u * 0.2 * b.hd * (1 + (b.h < 0.75 ? 0.15 : 0));
	const headY = base - H + hr;
	const neckY = headY + hr * 0.85;
	const hipY = base - H * 0.34;
	const hw = hr * (b.head === 'wide' ? 1.2 : b.head === 'long' ? 0.9 : 1);
	const hh = hr * (b.head === 'long' ? 1.2 : b.head === 'wide' ? 0.95 : 1);
	return {
		u, v, cx, base, W, headY, neckY, hipY, hw, hh,
		lw: Math.max(1.5, u * 0.14 * Math.min(1.3, b.w)),
		lhx: cx - W * 0.68, lhy: hipY,
		rhx: cx + W * 0.66, rhy: hipY - v * 0.02,
		shx: cx + W * 0.5, shy: neckY + v * 0.04,
	};
}

// ---------------------------------------------------------------- drawing helpers

function shade(col: string | undefined, amt: number): string {
	if (!col || col[0] !== '#' || col.length !== 7) return col ?? '#808080';
	const n = parseInt(col.slice(1), 16);
	const f = (q: number) => Math.max(0, Math.min(255, Math.round(q + amt * (amt > 0 ? 255 - q : q))));
	return `#${[(n >> 16) & 255, (n >> 8) & 255, n & 255].map((q) => f(q).toString(16).padStart(2, '0')).join('')}`;
}
const ell = (c: AnyCtx, cx: number, cy: number, rx: number, ry: number) => {
	c.beginPath();
	c.ellipse(cx, cy, Math.max(0.3, rx), Math.max(0.3, ry), 0, 0, Math.PI * 2);
	c.fill();
};
const poly = (c: AnyCtx, pts: number[]) => {
	c.beginPath();
	c.moveTo(pts[0], pts[1]);
	for (let i = 2; i < pts.length; i += 2) c.lineTo(pts[i], pts[i + 1]);
	c.closePath();
	c.fill();
};
const line = (c: AnyCtx, x0: number, y0: number, x1: number, y1: number, w: number) => {
	c.lineWidth = Math.max(1, w);
	c.beginPath();
	c.moveTo(x0, y0);
	c.lineTo(x1, y1);
	c.stroke();
};

const RARITY = ['', '', '#6aa8ff', '#6affff', '#ffb040', '#ff60ff', '#ffe060'];

function haftColour(g: string | undefined): string {
	return g === 'cyber' ? '#30343a' : g === 'waste' ? '#6a4a30' : g === 'steam' ? '#5a3a22' : '#7a5a3a';
}
function metalDefault(g: string | undefined): string {
	return g === 'cyber' ? '#3a3e46' : g === 'steam' ? '#c8a040' : g === 'waste' ? '#8a5a3a' : g === 'myth' ? '#b0773a' : '#c9ccd2';
}

// ---------------------------------------------------------------- layers

function drawBack(c: AnyCtx, g: GearLook, p: Geo) {
	const { u, v, cx, base, W, headY, neckY, hipY, hh } = p;
	const col = g.kc ?? '#4a3a3a';
	switch (g.k) {
		case 'cloak':
			c.fillStyle = col;
			poly(c, [cx - W * 0.56, neckY - v * 0.01, cx + W * 0.56, neckY - v * 0.01, cx + W * 0.76, base - v * 0.03, cx - W * 0.76, base - v * 0.03]);
			c.fillStyle = shade(col, -0.25);
			c.fillRect(cx - W * 0.76, base - v * 0.05, W * 1.52, Math.max(1, v * 0.02));
			break;
		case 'cape': {
			const low = hipY + (base - hipY) * 0.55;
			c.fillStyle = col;
			poly(c, [cx - W * 0.56, neckY, cx + W * 0.56, neckY, cx + W * 0.7, low, cx - W * 0.7, low]);
			if (g.t) {
				c.fillStyle = g.t;
				c.fillRect(cx - W * 0.7, low - Math.max(1, v * 0.02), W * 1.4, Math.max(1, v * 0.02));
			}
			break;
		}
		case 'wings':
			c.fillStyle = col;
			poly(c, [cx - W * 0.35, neckY + v * 0.04, cx - u * 0.5, headY - hh * 0.5, cx - u * 0.44, neckY + v * 0.08, cx - u * 0.36, hipY, cx - W * 0.3, hipY - v * 0.04]);
			poly(c, [cx + W * 0.35, neckY + v * 0.04, cx + u * 0.5, headY - hh * 0.5, cx + u * 0.44, neckY + v * 0.08, cx + u * 0.36, hipY, cx + W * 0.3, hipY - v * 0.04]);
			break;
		case 'banner': {
			const px = cx - W * 0.3;
			c.strokeStyle = '#7a5a3a';
			line(c, px, hipY, px, headY - hh * 2.2, Math.max(1, u * 0.06));
			c.fillStyle = col;
			poly(c, [px, headY - hh * 2.2, px - u * 0.3, headY - hh * 2.2, px - u * 0.3, headY - hh * 0.9, px - u * 0.15, headY - hh * 1.2, px, headY - hh * 0.9]);
			if (g.t) {
				c.fillStyle = g.t;
				ell(c, px - u * 0.15, headY - hh * 1.7, u * 0.05, u * 0.05);
			}
			break;
		}
		default:
			break;
	}
}

/** Sleeve colour (set as strokeStyle) and width for the body garment; the same values drawBody uses. */
function sleeveStyle(c: AnyCtx, g: GearLook, u: number): number {
	const k = g.b ?? '';
	const col = g.bc ?? '#7a6a5a';
	c.strokeStyle = k === 'kevlar' || k === 'apron' ? shade('#8a7a6a', -0.1) : k === 'frame' ? shade(col, -0.3) : col;
	return Math.max(1.4, u * 0.1) + (k === 'plate' || k === 'frame' ? 0.6 : 0);
}

/** omitR / omitL: leave out that primary sleeve (drawn live from the posed shoulder instead). */
function drawBody(c: AnyCtx, g: GearLook, p: Geo, b: GearBuild, omitR: boolean, omitL = false) {
	const { u, v, cx, base, W, neckY, hipY } = p;
	const col = g.bc ?? '#7a6a5a';
	const dk = shade(col, -0.3);
	const lt = shade(col, 0.25);
	const tH = hipY + v * 0.04 - neckY;
	const torso = (bottom = hipY + v * 0.04, flare = 0.44) => poly(c, [cx - W * 0.52, neckY, cx + W * 0.52, neckY, cx + W * flare, bottom, cx - W * flare, bottom]);
	const k = g.b ?? '';
	if (!k) return;
	c.fillStyle = col;
	switch (k) {
		case 'robe':
			poly(c, [cx - W * 0.52, neckY, cx + W * 0.52, neckY, cx + W * 0.64, base - v * 0.03, cx - W * 0.64, base - v * 0.03]);
			c.fillStyle = g.t ?? lt;
			c.fillRect(cx - W * 0.06, neckY, W * 0.12, base - v * 0.03 - neckY);
			c.fillRect(cx - W * 0.64, base - v * 0.05, W * 1.28, Math.max(1, v * 0.02));
			break;
		case 'rags': {
			const bot = hipY + v * 0.09;
			c.beginPath();
			c.moveTo(cx - W * 0.5, neckY);
			c.lineTo(cx + W * 0.5, neckY);
			c.lineTo(cx + W * 0.46, bot);
			for (let i = 1; i <= 5; i++) c.lineTo(cx + W * (0.46 - i * 0.184), bot - (i % 2 ? v * 0.05 : 0));
			c.closePath();
			c.fill();
			c.fillStyle = dk;
			c.fillRect(cx - W * 0.25, neckY + tH * 0.3, W * 0.22, tH * 0.22);
			break;
		}
		case 'apron':
			c.fillStyle = shade('#8a7a6a', -0.1);
			torso();
			c.fillStyle = col;
			c.fillRect(cx - W * 0.36, neckY + v * 0.03, W * 0.72, hipY + v * 0.14 - neckY - v * 0.03);
			c.strokeStyle = dk;
			line(c, cx - W * 0.3, neckY + v * 0.03, cx - W * 0.45, neckY, 1);
			line(c, cx + W * 0.3, neckY + v * 0.03, cx + W * 0.45, neckY, 1);
			break;
		case 'coat': {
			const low = hipY + (base - hipY) * 0.45;
			poly(c, [cx - W * 0.53, neckY, cx + W * 0.53, neckY, cx + W * 0.56, low, cx - W * 0.56, low]);
			c.strokeStyle = dk;
			line(c, cx, neckY + v * 0.03, cx, low, 1);
			c.fillStyle = dk;
			poly(c, [cx - W * 0.36, neckY, cx, neckY + v * 0.08, cx + W * 0.36, neckY, cx, neckY - v * 0.02]);
			break;
		}
		case 'gambeson':
			torso(hipY + v * 0.08, 0.48);
			c.strokeStyle = dk;
			for (let i = 1; i < 4; i++) line(c, cx - W * 0.46, neckY + tH * i * 0.25, cx + W * 0.46, neckY + tH * i * 0.25, 1);
			break;
		case 'leather':
			torso(hipY + v * 0.06);
			c.fillStyle = dk;
			ell(c, cx + W * 0.5, neckY + v * 0.03, u * 0.1, v * 0.05);
			c.strokeStyle = dk;
			line(c, cx - W * 0.42, neckY + v * 0.02, cx + W * 0.4, hipY - v * 0.02, 1);
			break;
		case 'mail': {
			torso(hipY + v * 0.09, 0.5);
			c.fillStyle = dk;
			const st = Math.max(2, u * 0.12);
			for (let yy = neckY + st * 0.6; yy < hipY + v * 0.08; yy += st)
				for (let xx = cx - W * 0.42 + ((Math.round((yy - neckY) / st) & 1) * st) / 2; xx < cx + W * 0.42; xx += st) c.fillRect(xx, yy, 1, 1);
			break;
		}
		case 'scale':
		case 'lamellar': {
			torso(hipY + v * 0.08, 0.48);
			c.strokeStyle = k === 'scale' ? dk : lt;
			const rows = 4;
			for (let i = 1; i <= rows; i++) {
				const yy = neckY + (tH * i) / (rows + 0.5);
				if (k === 'scale') {
					c.lineWidth = 1;
					c.beginPath();
					for (let j = -2; j <= 2; j++) c.arc(cx + j * W * 0.2, yy - v * 0.015, W * 0.1, 0, Math.PI);
					c.stroke();
				} else line(c, cx - W * 0.44, yy, cx + W * 0.44, yy, 1);
			}
			if (k === 'lamellar' && g.t) {
				c.strokeStyle = g.t;
				line(c, cx - W * 0.2, neckY, cx - W * 0.2, hipY, 1);
				line(c, cx + W * 0.2, neckY, cx + W * 0.2, hipY, 1);
			}
			break;
		}
		case 'plate':
			torso(hipY + v * 0.06);
			c.fillStyle = lt;
			poly(c, [cx - W * 0.34, neckY + v * 0.02, cx + W * 0.34, neckY + v * 0.02, cx + W * 0.26, neckY + tH * 0.65, cx, neckY + tH * 0.75, cx - W * 0.26, neckY + tH * 0.65]);
			c.fillStyle = col;
			ell(c, cx - W * 0.54, neckY + v * 0.04, u * 0.11, v * 0.06);
			ell(c, cx + W * 0.54, neckY + v * 0.04, u * 0.11, v * 0.06);
			c.fillStyle = dk;
			c.fillRect(cx - W * 0.44, hipY + v * 0.02, W * 0.88, Math.max(1, v * 0.04));
			break;
		case 'kevlar':
			c.fillStyle = shade(col, 0.15);
			torso();
			c.fillStyle = col;
			c.fillRect(cx - W * 0.4, neckY + v * 0.03, W * 0.8, tH * 0.72);
			c.fillStyle = dk;
			c.fillRect(cx - W * 0.3, neckY + tH * 0.3, W * 0.22, tH * 0.14);
			c.fillRect(cx + W * 0.08, neckY + tH * 0.3, W * 0.22, tH * 0.14);
			break;
		case 'frame':
			torso(hipY + v * 0.06, 0.5);
			c.fillStyle = lt;
			c.fillRect(cx - W * 0.7, neckY - v * 0.02, W * 0.36, v * 0.07);
			c.fillRect(cx + W * 0.34, neckY - v * 0.02, W * 0.36, v * 0.07);
			c.strokeStyle = dk;
			line(c, cx - W * 0.32, hipY, cx - W * 0.32, p.base - v * 0.06, Math.max(1, u * 0.05));
			line(c, cx + W * 0.32, hipY, cx + W * 0.32, p.base - v * 0.06, Math.max(1, u * 0.05));
			c.fillStyle = g.g === 'cyber' ? '#40e0ff' : g.g === 'steam' ? '#ffb040' : '#80ff80';
			ell(c, cx, neckY + tH * 0.4, u * 0.05, u * 0.05);
			break;
		case 'carapace':
			torso(hipY + v * 0.06, 0.46);
			c.strokeStyle = lt;
			c.lineWidth = 1;
			for (let i = 1; i < 4; i++) {
				c.beginPath();
				c.arc(cx, neckY + tH * i * 0.25 - W * 0.3, W * 0.5, Math.PI * 0.25, Math.PI * 0.75);
				c.stroke();
			}
			break;
		default: // tunic
			torso(hipY + v * 0.08, 0.48);
			break;
	}
	// belt
	if (k !== 'robe' && k !== 'rags') {
		c.fillStyle = g.t ?? dk;
		c.fillRect(cx - W * 0.44, hipY - v * 0.02, W * 0.88, Math.max(1, v * 0.03));
	}
	// sleeves over the person's arms, stopping short of the hands
	const sleeve = k === 'kevlar' || k === 'apron' ? shade('#8a7a6a', -0.1) : k === 'frame' ? dk : col;
	c.strokeStyle = sleeve;
	const aw = Math.max(1.4, u * 0.1) + (k === 'plate' || k === 'frame' ? 0.6 : 0);
	const pairs = b.arms === 4 ? [0, 1] : [0];
	for (const j of pairs) {
		const off = j * v * 0.1;
		if (!omitL || j > 0) line(c, cx - W * 0.5, neckY + v * 0.04 + off, cx - W * 0.64, hipY - v * 0.03 + off * 0.5, aw);
		if (!omitR || j > 0) line(c, cx + W * 0.5, neckY + v * 0.04 + off, cx + W * 0.62, hipY - v * 0.05 + off * 0.5, aw);
	}
}

/** Footwear on the ground line p.base. With pose legs (legL/legR/liftL/liftR) each boot swings with its
 * leg about the hip (hipY): rotate(-d), + d = foot to the right, and rises by the lift; else the old
 * 4-step lift from `pose.step`. */
function drawFeet(c: AnyCtx, g: GearLook, p: Geo, b: GearBuild, pose: GearPose | undefined, hipY: number) {
	if (!g.f || b.fish || b.swarm) return;
	const { v, cx, base, W, lw } = p;
	const legs = !!pose && (pose.legL !== undefined || pose.legR !== undefined || pose.liftL !== undefined || pose.liftR !== undefined);
	const step = pose?.step ?? 0;
	const lift = step === 1 ? v * 0.03 : step === 3 ? -v * 0.03 : 0;
	const col = g.fc ?? '#5a3a22';
	for (let side = -1; side <= 1; side += 2) {
		const fx = cx + side * W * 0.32;
		const up = legs ? ((side < 0 ? pose!.liftL : pose!.liftR) ?? 0) * v : side < 0 ? Math.max(0, lift) : Math.max(0, -lift);
		const d = legs ? ((side < 0 ? pose!.legL : pose!.legR) ?? 0) : 0;
		if (d) {
			c.save();
			c.translate(fx, hipY);
			c.rotate(-d);
			c.translate(-fx, -hipY);
		}
		const by = base - up;
		c.fillStyle = col;
		if (g.f === 'sandals') {
			c.fillRect(fx - lw * 0.7, by - v * 0.02, lw * 1.4, Math.max(1, v * 0.02));
			c.fillRect(fx - lw * 0.5, by - v * 0.07, lw, Math.max(1, v * 0.015));
		} else {
			const top = g.f === 'greaves' ? v * 0.26 : v * 0.17;
			c.fillRect(fx - lw * 0.6, by - top, lw * 1.2, top);
			c.fillRect(fx - lw * 0.6 + side * lw * 0.15, by - v * 0.04, lw * 1.35, v * 0.04);
			if (g.f === 'greaves') {
				c.fillStyle = shade(col, 0.3);
				c.fillRect(fx - lw * 0.6, by - top, lw * 1.2, Math.max(1, v * 0.03));
			}
		}
		if (d) c.restore();
	}
}

/** Gloves. noR / noL: leave that hand out (a posed hand is drawn live with drawGlove). */
function drawHands(c: AnyCtx, g: GearLook, p: Geo, noR: boolean, noL = false) {
	if (!g.n) return;
	const { u, v } = p;
	const col = g.nc ?? '#6a4a2a';
	const r = g.n === 'gauntlets' ? u * 0.065 : u * 0.052;
	c.fillStyle = col;
	if (!noL) ell(c, p.lhx, p.lhy, r, r);
	if (!noR) ell(c, p.rhx, p.rhy, r, r);
	if (g.n === 'gauntlets') {
		c.fillStyle = shade(col, 0.25);
		if (!noL) c.fillRect(p.lhx - r, p.lhy - r - v * 0.03, r * 2, v * 0.03);
		if (!noR) c.fillRect(p.rhx - r, p.rhy - r - v * 0.03, r * 2, v * 0.03);
	}
}

/** One glove at (hx, hy) (same look as drawHands). */
function drawGlove(c: AnyCtx, g: GearLook, u: number, v: number, hx: number, hy: number) {
	if (!g.n) return;
	const col = g.nc ?? '#6a4a2a';
	const r = g.n === 'gauntlets' ? u * 0.065 : u * 0.052;
	c.fillStyle = col;
	ell(c, hx, hy, r, r);
	if (g.n === 'gauntlets') {
		c.fillStyle = shade(col, 0.25);
		c.fillRect(hx - r, hy - r - v * 0.03, r * 2, v * 0.03);
	}
}

function drawHead(c: AnyCtx, g: GearLook, p: Geo) {
	if (!g.h) return;
	const { u, cx, headY, hw, hh } = p;
	const col = g.hc ?? '#8a909a';
	const es = Math.max(1, u * 0.07);
	c.fillStyle = col;
	switch (g.h) {
		case 'helm':
			c.beginPath();
			c.arc(cx, headY - hh * 0.1, hw * 1.1, Math.PI, 0);
			c.fill();
			c.fillRect(cx - hw * 1.12, headY - hh * 0.14, hw * 2.24, hh * 0.22);
			c.fillStyle = shade(col, -0.3);
			c.fillRect(cx - Math.max(0.5, hw * 0.1), headY - hh * 0.1, Math.max(1, hw * 0.2), hh * 0.55);
			break;
		case 'greathelm':
			c.beginPath();
			c.arc(cx, headY - hh * 0.15, hw * 1.12, Math.PI, 0);
			c.fill();
			c.fillRect(cx - hw * 1.12, headY - hh * 0.16, hw * 2.24, hh * 1.05);
			c.fillStyle = '#101014';
			c.fillRect(cx - hw * 0.8, headY - hh * 0.08, hw * 1.6, Math.max(1, hh * 0.16));
			c.fillStyle = g.t ?? '#b03a3a';
			poly(c, [cx - hw * 0.12, headY - hh * 1.15, cx + hw * 0.12, headY - hh * 1.15, cx + hw * 0.5, headY - hh * 1.75, cx, headY - hh * 1.35]);
			break;
		case 'hood':
			c.beginPath();
			c.ellipse(cx, headY - hh * 0.05, hw * 1.22, hh * 1.2, 0, 0, Math.PI * 2);
			c.ellipse(cx, headY + hh * 0.18, hw * 0.72, hh * 0.72, 0, 0, Math.PI * 2);
			c.fill('evenodd');
			poly(c, [cx - hw * 1.2, headY + hh * 0.2, cx + hw * 1.2, headY + hh * 0.2, cx + hw * 1.5, headY + hh * 1.15, cx - hw * 1.5, headY + hh * 1.15]);
			break;
		case 'hat':
			c.fillRect(cx - hw * 1.6, headY - hh * 0.5, hw * 3.2, hh * 0.18);
			c.fillRect(cx - hw * 0.8, headY - hh * 1.4, hw * 1.6, hh * 0.95);
			c.fillStyle = g.t ?? shade(col, 0.3);
			c.fillRect(cx - hw * 0.8, headY - hh * 0.62, hw * 1.6, hh * 0.12);
			break;
		case 'cap':
			c.beginPath();
			c.arc(cx, headY - hh * 0.25, hw * 1.0, Math.PI, 0);
			c.fill();
			c.fillRect(cx - hw * 1.1, headY - hh * 0.3, hw * 2.2, hh * 0.14);
			break;
		case 'circlet':
			c.fillRect(cx - hw * 0.95, headY - hh * 0.62, hw * 1.9, Math.max(1, hh * 0.14));
			c.fillStyle = g.t && g.t !== col ? g.t : '#5ad0ff';
			ell(c, cx, headY - hh * 0.68, hw * 0.18, hw * 0.18);
			break;
		case 'crown':
			poly(c, [cx - hw * 0.9, headY - hh * 0.55, cx - hw * 0.9, headY - hh * 1.4, cx - hw * 0.45, headY - hh * 0.9, cx, headY - hh * 1.5, cx + hw * 0.45, headY - hh * 0.9, cx + hw * 0.9, headY - hh * 1.4, cx + hw * 0.9, headY - hh * 0.55]);
			c.fillStyle = '#d83a3a';
			ell(c, cx, headY - hh * 0.75, hw * 0.12, hw * 0.12);
			break;
		case 'goggles':
			c.fillStyle = '#3a3a40';
			c.fillRect(cx - hw * 1.05, headY - hh * 0.28, hw * 2.1, hh * 0.16);
			c.fillStyle = g.g === 'cyber' ? '#ff40c0' : '#9ad8e8';
			ell(c, cx - hw * 0.45, headY - hh * 0.15, hw * 0.33, hw * 0.28);
			ell(c, cx + hw * 0.45, headY - hh * 0.15, hw * 0.33, hw * 0.28);
			c.fillStyle = col === '#25252c' ? '#c8a040' : col;
			ell(c, cx - hw * 0.45, headY - hh * 0.15, hw * 0.12, hw * 0.1);
			ell(c, cx + hw * 0.45, headY - hh * 0.15, hw * 0.12, hw * 0.1);
			break;
		case 'mask':
			c.fillRect(cx - hw * 0.95, headY - hh * 0.35, hw * 1.9, hh * 0.5);
			c.fillStyle = '#ffe9a0';
			c.fillRect(cx - hw * 0.5, headY - hh * 0.1, es, es * 0.8);
			c.fillRect(cx + hw * 0.5 - es, headY - hh * 0.1, es, es * 0.8);
			break;
		case 'turban':
			ell(c, cx, headY - hh * 0.5, hw * 1.12, hh * 0.68);
			c.strokeStyle = shade(col, -0.3);
			line(c, cx - hw * 0.9, headY - hh * 0.3, cx + hw * 0.8, headY - hh * 0.8, 1);
			if (g.t) {
				c.fillStyle = g.t;
				ell(c, cx, headY - hh * 0.75, hw * 0.14, hw * 0.14);
			}
			break;
		case 'bandana':
			c.fillRect(cx - hw * 1.0, headY - hh * 0.5, hw * 2, hh * 0.3);
			poly(c, [cx + hw, headY - hh * 0.4, cx + hw * 1.7, headY - hh * 0.1, cx + hw, headY - hh * 0.2]);
			break;
		default:
			break;
	}
}

function drawPacks(c: AnyCtx, g: GearLook, p: Geo) {
	const { u, v, cx, W, neckY, hipY } = p;
	if (g.k === 'pack') {
		c.fillStyle = g.kc ?? '#7a5a3a';
		c.fillRect(cx + W * 0.34, neckY + v * 0.04, u * 0.17, (hipY - neckY) * 0.8);
		c.fillStyle = shade(g.kc ?? '#7a5a3a', -0.3);
		c.fillRect(cx + W * 0.34, neckY + v * 0.04 + (hipY - neckY) * 0.35, u * 0.17, Math.max(1, v * 0.02));
	} else if (g.k === 'quiver') {
		c.fillStyle = g.kc ?? '#6a4a2a';
		c.fillRect(cx - W * 0.58, neckY - v * 0.02, u * 0.1, (hipY - neckY) * 0.8);
		c.strokeStyle = '#e0d8c0';
		line(c, cx - W * 0.54, neckY - v * 0.02, cx - W * 0.6, neckY - v * 0.08, 1);
		line(c, cx - W * 0.5, neckY - v * 0.02, cx - W * 0.46, neckY - v * 0.08, 1);
	}
}

function drawShield(c: AnyCtx, g: GearLook, p: Geo) {
	if (!g.s) return;
	const { u, v, cx, base, W, neckY, hipY } = p;
	const sx = cx - W * 0.8;
	const sy = hipY - v * 0.1;
	const col = g.sc ?? '#7a5a3a';
	const trim = g.t ?? '#c9ccd2';
	c.fillStyle = col;
	switch (g.s) {
		case 'buckler':
			ell(c, p.lhx - u * 0.02, p.lhy - v * 0.02, u * 0.09, u * 0.09);
			c.fillStyle = trim;
			ell(c, p.lhx - u * 0.02, p.lhy - v * 0.02, u * 0.03, u * 0.03);
			break;
		case 'kite':
			poly(c, [sx - u * 0.15, sy - v * 0.16, sx + u * 0.15, sy - v * 0.16, sx + u * 0.13, sy, sx, sy + v * 0.2, sx - u * 0.13, sy]);
			c.fillStyle = trim;
			c.fillRect(sx - Math.max(0.5, u * 0.02), sy - v * 0.15, Math.max(1, u * 0.04), v * 0.3);
			break;
		case 'tower': {
			const top = neckY;
			c.fillRect(sx - u * 0.14, top, u * 0.28, base - v * 0.08 - top);
			c.fillStyle = shade(col, -0.3);
			c.fillRect(sx - u * 0.14, top, u * 0.28, Math.max(1, v * 0.02));
			c.fillStyle = trim;
			ell(c, sx, top + (base - top) * 0.42, u * 0.04, u * 0.04);
			break;
		}
		case 'riot':
			c.fillRect(sx - u * 0.14, neckY, u * 0.28, hipY + v * 0.14 - neckY);
			c.fillStyle = 'rgba(160,200,230,0.55)';
			c.fillRect(sx - u * 0.1, neckY + v * 0.02, u * 0.2, v * 0.07);
			break;
		case 'energy':
			c.fillStyle = 'rgba(90,220,255,0.28)';
			ell(c, sx, sy, u * 0.17, v * 0.15);
			c.strokeStyle = 'rgba(140,240,255,0.9)';
			c.lineWidth = 1;
			c.beginPath();
			c.ellipse(sx, sy, u * 0.17, v * 0.15, 0, 0, Math.PI * 2);
			c.stroke();
			break;
		default: // round
			ell(c, sx, sy, u * 0.15, v * 0.12);
			c.strokeStyle = shade(col, -0.35);
			c.lineWidth = 1;
			c.beginPath();
			c.ellipse(sx, sy, u * 0.15, v * 0.12, 0, 0, Math.PI * 2);
			c.stroke();
			c.fillStyle = trim;
			ell(c, sx, sy, u * 0.05, v * 0.04);
			break;
	}
}

function drawProp(c: AnyCtx, g: GearLook, p: Geo) {
	if (!g.o || g.s) return;
	const { u, v } = p;
	const hx = p.lhx;
	const hy = p.lhy;
	switch (g.o) {
		case 'purse':
			c.fillStyle = '#6a4a2a';
			ell(c, hx, hy + v * 0.03, u * 0.09, v * 0.06);
			c.fillStyle = g.t ?? '#f0c850';
			ell(c, hx, hy - v * 0.02, u * 0.04, v * 0.02);
			break;
		case 'book':
			c.fillStyle = '#7a3a2a';
			c.fillRect(hx - u * 0.1, hy - v * 0.1, u * 0.18, v * 0.14);
			c.fillStyle = '#e8e0c0';
			c.fillRect(hx - u * 0.08, hy - v * 0.08, u * 0.14, Math.max(1, v * 0.03));
			break;
		case 'symbol':
			c.fillStyle = g.t ?? '#e8c850';
			c.fillRect(hx - Math.max(0.5, u * 0.02), hy - v * 0.12, Math.max(1, u * 0.04), v * 0.14);
			c.fillRect(hx - u * 0.06, hy - v * 0.09, u * 0.12, Math.max(1, v * 0.025));
			break;
		case 'lantern':
			c.fillStyle = 'rgba(255,210,74,0.3)';
			ell(c, hx, hy + v * 0.04, u * 0.16, u * 0.16);
			c.fillStyle = '#ffd24a';
			ell(c, hx, hy + v * 0.04, u * 0.06, v * 0.05);
			break;
		case 'orb':
			c.fillStyle = g.t ?? '#a0d0ff';
			ell(c, hx, hy - v * 0.04, u * 0.08, u * 0.08);
			break;
		default:
			break;
	}
}

/** Weapon in local coordinates: hand at (0,0), up = -y. Returns the tip for the glint. */
function drawWeapon(c: Ctx, g: GearLook, u: number, v: number): [number, number] {
	const m = g.wc ?? metalDefault(g.g);
	const wood = haftColour(g.g);
	const lw = Math.max(1, u * 0.07);
	const trim = g.t ?? '#c9a24a';
	c.strokeStyle = m;
	c.fillStyle = m;
	switch (g.w) {
		case 'sword':
			line(c, 0, 0, u * 0.05, -v * 0.42, lw * 1.2);
			c.strokeStyle = trim;
			line(c, -u * 0.1, -v * 0.05, u * 0.11, -v * 0.04, lw);
			return [u * 0.05, -v * 0.42];
		case 'dagger':
		case 'tknife':
			line(c, 0, 0, u * 0.03, -v * (g.w === 'dagger' ? 0.22 : 0.18), lw);
			c.strokeStyle = trim;
			line(c, -u * 0.06, -v * 0.03, u * 0.06, -v * 0.03, 1);
			return [u * 0.03, -v * 0.22];
		case 'axe':
		case 'taxe': {
			const L = g.w === 'axe' ? 0.4 : 0.28;
			c.strokeStyle = wood;
			line(c, 0, v * 0.04, 0, -v * L, lw);
			c.fillStyle = m;
			poly(c, [0, -v * L, u * 0.22, -v * (L - 0.06), u * 0.2, -v * (L - 0.2), 0, -v * (L - 0.16)]);
			return [u * 0.22, -v * (L - 0.06)];
		}
		case 'mace':
			c.strokeStyle = wood;
			line(c, 0, v * 0.03, 0, -v * 0.3, lw);
			c.fillStyle = m;
			ell(c, 0, -v * 0.33, u * 0.08, u * 0.08);
			poly(c, [-u * 0.12, -v * 0.33, 0, -v * 0.36, u * 0.12, -v * 0.33, 0, -v * 0.3]);
			return [0, -v * 0.38];
		case 'hammer':
			c.strokeStyle = wood;
			line(c, 0, v * 0.04, 0, -v * 0.34, lw);
			c.fillRect(-u * 0.12, -v * 0.4, u * 0.24, v * 0.1);
			return [u * 0.12, -v * 0.4];
		case 'spear':
		case 'javelin': {
			const L = g.w === 'spear' ? 0.56 : 0.44;
			c.strokeStyle = wood;
			line(c, 0, v * 0.08, 0, -v * L, lw);
			c.fillStyle = m;
			poly(c, [0, -v * (L + 0.12), -u * 0.06, -v * (L - 0.02), u * 0.06, -v * (L - 0.02)]);
			return [0, -v * (L + 0.12)];
		}
		case 'flail':
			c.strokeStyle = wood;
			line(c, 0, v * 0.02, u * 0.02, -v * 0.18, lw);
			c.fillStyle = '#6a6e76';
			ell(c, u * 0.06, -v * 0.22, 1, 1);
			ell(c, u * 0.1, -v * 0.25, 1, 1);
			c.fillStyle = m;
			ell(c, u * 0.15, -v * 0.27, u * 0.08, u * 0.08);
			return [u * 0.15, -v * 0.3];
		case 'whip':
			c.strokeStyle = wood;
			line(c, 0, 0, u * 0.03, -v * 0.1, lw * 1.2);
			c.strokeStyle = '#5a3a22';
			c.lineWidth = 1;
			c.beginPath();
			c.moveTo(u * 0.03, -v * 0.1);
			c.quadraticCurveTo(u * 0.35, -v * 0.3, u * 0.32, v * 0.06);
			c.stroke();
			return [u * 0.32, v * 0.06];
		case 'staff':
			c.strokeStyle = wood;
			line(c, 0, v * 0.08, 0, -v * 0.6, lw);
			c.fillStyle = trim;
			ell(c, 0, -v * 0.62, u * 0.07, u * 0.07);
			return [0, -v * 0.64];
		case 'fist':
			c.fillRect(-u * 0.07, -v * 0.05, u * 0.14, v * 0.06);
			c.fillStyle = shade(m, 0.3);
			for (let i = -1; i <= 1; i++) ell(c, i * u * 0.045, -v * 0.06, 1, 1);
			return [0, -v * 0.06];
		case 'hook':
			c.strokeStyle = wood;
			line(c, 0, v * 0.03, 0, -v * 0.28, lw);
			c.strokeStyle = m;
			c.lineWidth = lw;
			c.beginPath();
			c.arc(u * 0.08, -v * 0.3, u * 0.08, Math.PI, Math.PI * 1.9);
			c.stroke();
			return [u * 0.16, -v * 0.31];
		case 'great':
			line(c, 0, 0, u * 0.04, -v * 0.6, lw * 1.8);
			c.strokeStyle = trim;
			line(c, -u * 0.14, -v * 0.06, u * 0.15, -v * 0.05, lw);
			c.strokeStyle = wood;
			line(c, 0, 0, -u * 0.01, v * 0.07, lw);
			return [u * 0.04, -v * 0.6];
		case 'chakram':
			c.lineWidth = Math.max(1, u * 0.04);
			c.beginPath();
			c.arc(u * 0.02, -v * 0.08, u * 0.1, 0, Math.PI * 2);
			c.stroke();
			return [u * 0.12, -v * 0.08];
		case 'boomerang':
			c.strokeStyle = wood;
			c.lineWidth = lw;
			c.beginPath();
			c.moveTo(-u * 0.04, -v * 0.18);
			c.lineTo(u * 0.04, -v * 0.04);
			c.lineTo(u * 0.18, -v * 0.12);
			c.stroke();
			return [u * 0.18, -v * 0.12];
		case 'bolas':
			c.strokeStyle = '#8a7a5a';
			line(c, 0, 0, -u * 0.08, v * 0.1, 1);
			line(c, 0, 0, u * 0.1, v * 0.08, 1);
			c.fillStyle = '#6a6e76';
			ell(c, -u * 0.08, v * 0.1, u * 0.04, u * 0.04);
			ell(c, u * 0.1, v * 0.08, u * 0.04, u * 0.04);
			return [0, 0];
		case 'sling':
			c.strokeStyle = '#8a7a5a';
			line(c, 0, 0, u * 0.06, v * 0.14, 1);
			c.fillStyle = '#6a4a2a';
			ell(c, u * 0.06, v * 0.15, u * 0.04, v * 0.025);
			return [u * 0.06, v * 0.15];
		case 'blowgun':
			c.strokeStyle = wood;
			line(c, -u * 0.06, v * 0.04, u * 0.22, -v * 0.38, Math.max(1, lw * 0.8));
			return [u * 0.22, -v * 0.38];
		case 'bow':
			c.strokeStyle = wood;
			c.lineWidth = lw;
			c.beginPath();
			c.arc(u * 0.02, -v * 0.14, v * 0.27, -Math.PI / 2 - 0.25, Math.PI / 2 + 0.25, false);
			c.stroke();
			c.strokeStyle = '#d8d0b8';
			line(c, u * 0.02 - Math.sin(0.25) * v * 0.27, -v * 0.14 - Math.cos(0.25) * v * 0.27, u * 0.02 - Math.sin(0.25) * v * 0.27, -v * 0.14 + Math.cos(0.25) * v * 0.27, 1);
			return [u * 0.02 + v * 0.27, -v * 0.14];
		case 'crossbow':
			c.strokeStyle = wood;
			line(c, -u * 0.06, 0, u * 0.24, -v * 0.08, lw * 1.2);
			c.strokeStyle = m;
			line(c, u * 0.22, -v * 0.2, u * 0.25, v * 0.04, lw);
			return [u * 0.25, -v * 0.08];
		case 'atlatl':
			c.strokeStyle = wood;
			line(c, 0, 0, u * 0.04, -v * 0.2, lw);
			c.strokeStyle = '#a89070';
			line(c, -u * 0.04, v * 0.06, u * 0.1, -v * 0.46, 1);
			return [u * 0.1, -v * 0.46];
		case 'pistol':
			c.fillStyle = shade(m, -0.2);
			c.fillRect(-u * 0.03, -v * 0.06, u * 0.06, v * 0.07);
			c.fillStyle = m;
			c.fillRect(-u * 0.03, -v * 0.09, u * 0.24, v * 0.04);
			return [u * 0.21, -v * 0.07];
		case 'longgun':
		case 'repeater':
			c.strokeStyle = wood;
			line(c, -u * 0.12, v * 0.06, u * 0.04, -v * 0.04, lw * 1.6);
			c.strokeStyle = m;
			line(c, u * 0.02, -v * 0.03, u * 0.34, -v * 0.24, lw * (g.w === 'repeater' ? 1.6 : 1.1));
			if (g.w === 'repeater') {
				c.fillStyle = shade(m, -0.3);
				ell(c, u * 0.1, -v * 0.02, u * 0.06, u * 0.06);
			}
			return [u * 0.34, -v * 0.24];
		case 'wand':
			line(c, 0, 0, u * 0.1, -v * 0.2, lw);
			c.fillStyle = trim;
			ell(c, u * 0.1, -v * 0.21, u * 0.035, u * 0.035);
			return [u * 0.1, -v * 0.21];
		case 'orb':
			c.fillStyle = trim;
			ell(c, 0, -v * 0.06, u * 0.09, u * 0.09);
			return [0, -v * 0.06];
		default:
			return [0, 0];
	}
}

function glint(c: AnyCtx, x: number, y: number, r: number, col: string, strong: boolean) {
	if (strong) {
		const gr = c.createRadialGradient(x, y, 0, x, y, r * 2.4);
		gr.addColorStop(0, col + '88');
		gr.addColorStop(1, col + '00');
		c.fillStyle = gr;
		c.fillRect(x - r * 2.4, y - r * 2.4, r * 4.8, r * 4.8);
	}
	c.fillStyle = col;
	poly(c, [x, y - r, x + r * 0.25, y - r * 0.25, x + r, y, x + r * 0.25, y + r * 0.25, x, y + r, x - r * 0.25, y + r * 0.25, x - r, y, x - r * 0.25, y - r * 0.25]);
	c.fillStyle = '#ffffff';
	ell(c, x, y, Math.max(0.5, r * 0.22), Math.max(0.5, r * 0.22));
}

// ---------------------------------------------------------------- caching and entry points

/** A stable key for a gear look (cache it with the actor id if you cache whole sprites). */
export function gearKey(g: GearLook): string {
	return [g.h, g.hc, g.b, g.bc, g.t, g.k, g.kc, g.n, g.nc, g.f, g.fc, g.s, g.sc, g.w, g.wc, g.wr, g.o, g.r, g.g].map((v) => v ?? '').join('|');
}

type Canv = HTMLCanvasElement | OffscreenCanvas;
const cache = new Map<string, Canv>();

function makeCanvas(w: number, h: number): Canv | null {
	if (typeof OffscreenCanvas !== 'undefined') return new OffscreenCanvas(w, h);
	if (typeof document !== 'undefined') {
		const cv = document.createElement('canvas');
		cv.width = w;
		cv.height = h;
		return cv;
	}
	return null;
}

const PADX = 0.6; // cell fractions of room around the cell (spears, banners, crowns, cloaks)
const PADY = 0.7;

/** The resolution scale for cached layers, read from the current transform (before any pose frame). */
function cacheScale(c: Ctx): number {
	const t = typeof c.getTransform === 'function' ? c.getTransform() : null;
	return Math.min(4, Math.max(1, t ? Math.hypot(t.a, t.b) : 1));
}

/** `scale`: pass cacheScale() taken before the pose frame, so a lying (scaled) body reuses the layers. */
function cached(c: Ctx, key: string, cw: number, chh: number, paint: (k: AnyCtx, x0: number, y0: number) => void, scale?: number): Canv | null {
	const s = scale ?? cacheScale(c);
	const full = `${key}|${cw.toFixed(2)}|${chh.toFixed(2)}|${s.toFixed(2)}`;
	let cv = cache.get(full);
	if (cv) return cv;
	const w = Math.ceil(cw * (1 + 2 * PADX) * s);
	const h = Math.ceil(chh * (1 + 2 * PADY) * s);
	const made = makeCanvas(w, h);
	if (!made) return null;
	const k: AnyCtx | null = made instanceof HTMLCanvasElement ? made.getContext('2d') : (made as OffscreenCanvas).getContext('2d');
	if (!k) return null;
	cv = made;
	k.scale(s, s);
	paint(k, cw * PADX, chh * PADY);
	if (cache.size > 800) cache.clear();
	cache.set(full, cv);
	return cv;
}

function blit(c: Ctx, cv: Canv, x: number, y: number, cw: number, chh: number) {
	c.drawImage(cv as CanvasImageSource, x - cw * PADX, y - chh * PADY, cw * (1 + 2 * PADX), chh * (1 + 2 * PADY));
}

function buildKey(b: GearBuild): string {
	return `${b.h}|${b.w}|${b.hd}|${b.head}|${b.fish ? 1 : 0}|${b.arms ?? 2}|${b.vary}`;
}

/** Back pieces (cloak, cape, wings, banner): call BEFORE drawPersonSprite with the same cell. */
export function drawGearBack(ctx: Ctx, g: GearLook, build: GearBuild, x: number, y: number, cw: number, chh: number, pose?: GearPose) {
	drawGear(ctx, g, build, x, y, cw, chh, pose, 'back');
}

/**
 * Gear layers over a person sprite in cell (x, y, cw, chh). layers 'all' (default) paints everything;
 * pass 'front' when the back was drawn before the sprite with drawGearBack. Order: back, body,
 * feet, hands, head, pack/quiver, shield, off-hand prop, weapon in the right (screen) hand, glint.
 */
/** rotate(+a) about (px, py): the torso lean (+ = leans right). save() only when a != 0; the caller restores when a != 0. */
function leanAt(c: Ctx, a: number, px: number, py: number) {
	if (!a) return;
	c.save();
	c.translate(px, py);
	c.rotate(a);
	c.translate(-px, -py);
}

/** Rotate the point (ex, ey) about the shoulder (sx, sy) by arm delta d (+ = a hanging hand moves RIGHT):
 * (dx, dy) -> (dx cos d + dy sin d, -dx sin d + dy cos d). Writes into ROT_OUT (no allocation). */
const ROT_OUT = { x: 0, y: 0 };
function rotArm(sx: number, sy: number, ex: number, ey: number, d: number) {
	const dx = ex - sx;
	const dy = ey - sy;
	const ca = Math.cos(d);
	const sa = Math.sin(d);
	ROT_OUT.x = sx + dx * ca + dy * sa;
	ROT_OUT.y = sy - dx * sa + dy * ca;
}

export function drawGear(ctx: Ctx, g: GearLook, build: GearBuild, x: number, y: number, cw: number, chh: number, pose?: GearPose, layers: 'all' | 'back' | 'front' = 'all') {
	if (build.swarm || cw < 3 || chh < 3) return;
	const sc = cacheScale(ctx); // before the pose frame: a lying (scaled) body keeps using the same cached layers
	const px = x + (pose?.dx ?? 0);
	const py = y + (pose?.dy ?? 0);
	const drop = pose?.drop ?? 0; // upper body only (+ = down)
	const uy = py + drop;
	const lean = pose?.lean ?? 0;
	const lie = pose?.lie ?? 0;
	const mirror = !!pose?.mirror;
	const gk = gearKey(g);
	const bk = buildKey(build);
	const p = geo(build, px, uy, cw, chh); // upper body (hips, torso, hands, head)
	// whole-body frame, the same call drawPersonSprite makes: pivot at the CELL centre (x, not px) and the
	// ground line of the bobbed cell (py, not uy); H as in drawPersonSprite
	const framed = !!(lie || mirror);
	if (framed) {
		ctx.save();
		applyBodyFrame(ctx, lie, mirror, x + cw * 0.5, py + chh * 0.9, chh * 0.82 * build.h * build.vary, cw);
	}
	if (layers !== 'front' && g.k && g.k !== 'pack' && g.k !== 'quiver') {
		const cv = cached(ctx, `B|${gk}|${bk}`, cw, chh, (k, x0, y0) => drawBack(k, g, geo(build, x0, y0, cw, chh)), sc);
		if (cv) {
			leanAt(ctx, lean, p.cx, p.hipY); // the cloak leans with the torso, like the sprite's cape
			blit(ctx, cv, px, uy, cw, chh);
			if (lean) ctx.restore();
		}
	}
	if (layers === 'back') {
		if (framed) ctx.restore();
		return;
	}
	let rot = pose?.rot;
	if (rot === undefined && pose?.swing) rot = Math.sin(Math.PI * Math.max(0, Math.min(1, pose.swing))) * 1.3;
	const rotL = pose?.rotL ?? 0;
	const posedR = !!rot && Math.abs(rot) > ARM_POSED; // right (weapon) hand drawn at its posed place
	const posedL = Math.abs(rotL) > ARM_POSED; // left hand group (glove, shield, prop) follows the left arm
	const slR = posedR && !!g.b; // posed sleeves drawn live (gearDrawsArm: the sprite then skips its right arm)
	const slL = posedL && !!g.b;
	// body garment (cached; posed sleeves left out)
	leanAt(ctx, lean, p.cx, p.hipY);
	const body = cached(ctx, `F1|${gk}|${bk}|${slR ? 1 : 0}${slL ? 1 : 0}`, cw, chh, (k, x0, y0) => drawBody(k, g, geo(build, x0, y0, cw, chh), build, slR, slL), sc);
	if (body) blit(ctx, body, px, uy, cw, chh);
	if (lean) ctx.restore();
	// feet move with the walk: on the ground line (no drop, no lean), swung about the dropped hips
	ctx.save();
	drawFeet(ctx, g, drop ? geo(build, px, py, cw, chh) : p, build, pose, p.hipY);
	ctx.restore();
	// the rest of the upper body leans about the hips
	leanAt(ctx, lean, p.cx, p.hipY);
	// posed left hand: the rest hand rotated about the left shoulder by rotL
	let lhx = p.lhx;
	let lhy = p.lhy;
	const lsx = p.cx - p.W * 0.5;
	if (posedL) {
		rotArm(lsx, p.shy, p.lhx, p.lhy, rotL);
		lhx = ROT_OUT.x;
		lhy = ROT_OUT.y;
	}
	// posed right hand
	let hx = p.rhx;
	let hy = p.rhy;
	if (posedR) {
		rotArm(p.shx, p.shy, p.rhx, p.rhy, rot!);
		hx = ROT_OUT.x;
		hy = ROT_OUT.y;
	}
	// live sleeves: the rest sleeve end rotated about the shoulder, stopping short of the hand like the cached ones
	if (slL || slR) {
		const aw = sleeveStyle(ctx, g, cw);
		if (slL) {
			rotArm(lsx, p.shy, p.cx - p.W * 0.64, p.hipY - chh * 0.03, rotL);
			line(ctx, lsx, p.shy, ROT_OUT.x, ROT_OUT.y, aw);
		}
		if (slR) {
			rotArm(p.shx, p.shy, p.cx + p.W * 0.62, p.hipY - chh * 0.05, rot!);
			line(ctx, p.shx, p.shy, ROT_OUT.x, ROT_OUT.y, aw);
		}
	}
	// hands, head, packs, shield, prop, armour glint (cached at rest; posed hands drawn below)
	const top = cached(ctx, `F2|${gk}|${bk}|${posedR ? 1 : 0}${posedL ? 1 : 0}`, cw, chh, (k, x0, y0) => {
		const q = geo(build, x0, y0, cw, chh);
		drawHands(k, g, q, posedR, posedL);
		drawHead(k, g, q);
		drawPacks(k, g, q);
		if (!posedL) {
			drawShield(k, g, q);
			drawProp(k, g, q);
		}
		if ((g.r ?? 0) >= 2 && (g.wr ?? 0) < 2) {
			const col = RARITY[Math.min(6, g.r ?? 0)];
			if (g.h) glint(k, q.cx + q.hw * 0.7, q.headY - q.hh * 0.6, Math.max(1.2, cw * 0.09), col, false);
			else glint(k, q.cx + q.W * 0.25, q.neckY + (q.hipY - q.neckY) * 0.3, Math.max(1.2, cw * 0.09), col, false);
		}
	}, sc);
	if (top) blit(ctx, top, px, uy, cw, chh);
	if (posedL && (g.n || g.s || g.o)) {
		// the left-hand group, cached at rest and moved by the hand's offset (it translates, it does not turn)
		const left = cached(ctx, `F3|${gk}|${bk}`, cw, chh, (k, x0, y0) => {
			const q = geo(build, x0, y0, cw, chh);
			drawGlove(k, g, cw, chh, q.lhx, q.lhy);
			drawShield(k, g, q);
			drawProp(k, g, q);
		}, sc);
		if (left) blit(ctx, left, px + (lhx - p.lhx), uy + (lhy - p.lhy), cw, chh);
	}
	if (posedR) drawGlove(ctx, g, cw, chh, hx, hy);
	// weapon, live (the arm may be posed)
	if (g.w) {
		ctx.save();
		ctx.translate(hx, hy);
		// the weapon tips forward by half the arm swing: rotate(+rot/2) turns its up direction clockwise (to the right)
		if (rot) ctx.rotate(rot * 0.5);
		const [tx, ty] = drawWeapon(ctx, g, cw, chh);
		const wr = g.wr ?? 0;
		if (wr >= 2) glint(ctx, tx, ty, Math.max(1.4, cw * 0.11), RARITY[Math.min(6, wr)], wr >= 3);
		ctx.restore();
	}
	if (lean) ctx.restore();
	if (framed) ctx.restore();
}

// ---------------------------------------------------------------- frame state and the one-call wrapper

let gearNow: Record<string, GearLook> | undefined;
let gearTick = 0;
let gearOn = true;

/** Per frame: the engine's `frame.gear` and the sprite animation tick (the same one given to setSpriteAnimTick). */
export function setGearFrame(gear: Record<string, GearLook> | undefined, tick?: number) {
	gearNow = gear;
	if (tick !== undefined) gearTick = Number.isFinite(tick) ? tick : 0;
}
export function setGearTick(tick: number) {
	gearTick = Number.isFinite(tick) ? tick : 0;
}
export function setGearPrefs(p: { gearSprites?: boolean }) {
	if (p.gearSprites !== undefined) gearOn = p.gearSprites;
}
export function gearFor(id: number): GearLook | undefined {
	return gearOn ? gearNow?.[String(id)] : undefined;
}

interface PersonLike {
	i: number;
	x: number;
	y: number;
	s?: string;
	a?: number;
}

/**
 * Paint a person with their gear: back layer, then `draw(actor)` (drawPersonSprite) with the
 * outfit's own worn pieces turned off, then the front layers. Without gear (or with the pref off) it
 * just calls draw(actor). `pose.dy` is added to the sprite's own idle bob.
 */
export function withGear<A extends PersonLike>(ctx: Ctx, actor: A, x: number, y: number, cw: number, chh: number, draw: (a: A) => void, pose?: GearPose) {
	const g = gearFor(actor.i);
	if (!g) {
		draw(actor);
		return;
	}
	const b = gearBuild(actor.s, actor.i);
	const bob = personBob(actor.i, actor.x, actor.y, chh, gearTick);
	const ps: GearPose = { ...pose, dy: (pose?.dy ?? 0) + bob };
	drawGearBack(ctx, g, b, x, y, cw, chh, ps);
	// the sprite's own head, back, shield, boots, gloves and AV armour give way to the gear layers
	// (actorsprites applyGear with `ge`); `gx` asks outfitFor to drop its held item too (one-line hook)
	// `ga`: drawGear draws the posed weapon sleeve, so the sprite leaves out its own right arm line (no ghost arm)
	draw({ ...actor, ge: true, hd: undefined, bk: undefined, ft: undefined, hn: undefined, bi: undefined, sh: false, a: g.b ? 0 : actor.a, gx: true, ga: gearDrawsArm(g, pose) });
	drawGear(ctx, g, b, x, y, cw, chh, ps, 'front');
}

/** Drop cached layers (call when the palette or cell size changes a lot; the cache also caps itself). */
export function clearGearCache() {
	cache.clear();
	buildCache.clear();
}
