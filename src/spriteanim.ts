// Sprite-mode animation state: the shared options (reduceMotion, tileVariety, tileAnim), the low
// frame-rate clock for ambient tile animation (water, lava, fire, wind, portals; painted by
// tilevariety.ts), and per-actor motion (idle bob, walk step, slide between tiles, attack lunge,
// death fade). It builds on actorsprites.ts: hit flashes go through its setSpriteAnimTick, and its
// own tick stays 0 so the person sprites do not bob a second time on top of actorOffset().
//
// Nothing here touches the canvas or the engine. Play.svelte asks for numbers and applies them.

import { setSpriteAnimTick, type QudActor } from './actorsprites';

export interface SpriteAnimPrefs {
	reduceMotion?: boolean; // the existing setting (also pass the prefers-reduced-motion match)
	tileVariety?: boolean; // new pref, default true: static per-tile variants and wall autotiling
	tileAnim?: boolean; // new pref, default true: ambient tile animation
}

const opts = { reduced: false, variety: true, anim: true };

/** Call whenever settings change (and once at start). `reduceMotion` freezes every motion; static variants still apply. */
export function setSpriteAnimPrefs(p: SpriteAnimPrefs): void {
	opts.reduced = !!p.reduceMotion;
	opts.variety = p.tileVariety !== false;
	opts.anim = p.tileAnim !== false;
	if (opts.reduced) resetActorMotion();
}

export function tileOptions(): Readonly<{ reduced: boolean; variety: boolean; anim: boolean }> {
	return opts;
}

// ---- ambient tile clock ----------------------------------------------------------------------

export const TILE_FPS = 7;
export const TILE_FRAMES = 8; // every ambient animation loops over 8 frames (about 1.1 s)
let tileFrame = 0;

/** Current ambient frame number (0 when motion is off). tilevariety.ts takes it modulo TILE_FRAMES. */
export function tileAnimFrame(): number {
	return opts.anim && !opts.reduced ? tileFrame : 0;
}

/** Advance the ambient clock. Returns true when the frame changed, i.e. the animated tiles want a repaint (about 7 times a second). */
export function tickTileAnim(now: number): boolean {
	if (!opts.anim || opts.reduced) return false;
	const f = Math.floor((now * TILE_FPS) / 1000);
	if (f === tileFrame) return false;
	tileFrame = f;
	return true;
}

// ---- actor motion ----------------------------------------------------------------------------

export const SLIDE_MS = 90;
const LUNGE_MS = 170;
const DEATH_MS = 380;
const STEP_MS = 200;
const FLASH_MS = 160;

interface Motion {
	cx: number; // cell the actor was last seen in
	cy: number;
	fx: number; // cell the slide started from
	fy: number;
	slideT0: number; // -1e9 when no slide
	steps: number; // moves counted, its parity picks the foot
	lungeT0: number;
	lx: number; // lunge direction (unit)
	ly: number;
	seen: number; // frame stamp of the last trackActors/actorOffset call
	last?: QudActor;
}
const motion = new Map<number, Motion>();
interface Ghost {
	a: QudActor;
	x: number;
	y: number;
	t0: number;
}
let ghosts: Ghost[] = [];
const flashes = new Map<number, number>(); // actor id -> until (ms)
let stamp = 0;

function resetActorMotion() {
	motion.clear();
	ghosts = [];
	flashes.clear();
	setSpriteAnimTick(0, new Set());
}

/** Forget every actor (new zone, new game, load). */
export function resetSpriteAnim(): void {
	resetActorMotion();
	tileFrame = 0;
}

function mo(id: number, x: number, y: number, now: number): Motion {
	let m = motion.get(id);
	if (!m) {
		m = { cx: x, cy: y, fx: x, fy: y, slideT0: -1e9, steps: 0, lungeT0: -1e9, lx: 0, ly: 0, seen: stamp };
		motion.set(id, m);
		return m;
	}
	m.seen = stamp;
	if (m.cx !== x || m.cy !== y) {
		const d = Math.hypot(x - m.cx, y - m.cy);
		if (d <= 1.5 && !opts.reduced) {
			// a step: slide from the old cell (unless one is mid-way, then keep going from where it is)
			m.fx = m.cx;
			m.fy = m.cy;
			m.slideT0 = now;
			m.steps++;
		} else {
			m.slideT0 = -1e9; // a jump (teleport, new zone): no slide
		}
		m.cx = x;
		m.cy = y;
	}
	return m;
}

/** Per-frame bookkeeping: pass `frame.actors` once each time the frame is drawn (also after a key press). Detects actors that vanished (death fade) and drops stale state. */
export function trackActors(actors: QudActor[] | undefined, now: number): void {
	stamp++;
	const list = actors ?? [];
	const prevIds = new Set(motion.keys());
	for (const a of list) {
		const m = mo(a.i, a.x, a.y, now);
		m.last = a;
		prevIds.delete(a.i);
	}
	if (prevIds.size) {
		const player = list.find((a) => a.p);
		const jumped = player && motion.get(player.i) ? false : true;
		const mass = prevIds.size > 3 && prevIds.size * 2 > motion.size;
		for (const id of prevIds) {
			const m = motion.get(id)!;
			motion.delete(id);
			if (!opts.reduced && !jumped && !mass && m.last && !m.last.p && now - Math.max(m.slideT0, 0) >= 0) {
				ghosts.push({ a: m.last, x: m.cx, y: m.cy, t0: now });
			}
		}
	}
	if (ghosts.length) ghosts = ghosts.filter((g) => now - g.t0 < DEATH_MS);
	for (const [id, until] of flashes) if (now > until) flashes.delete(id);
}

/** An actor attacks toward cell (tx, ty): it lunges a third of a cell that way and back (about 170 ms). */
export function noteAttack(attackerId: number, tx: number, ty: number, now: number): void {
	if (opts.reduced) return;
	const m = motion.get(attackerId);
	if (!m) return;
	const dx = tx - m.cx;
	const dy = ty - m.cy;
	const l = Math.hypot(dx, dy) || 1;
	m.lx = dx / l;
	m.ly = dy / l;
	m.lungeT0 = now;
}

/** Make an actor fade out now (Play knows a death from the log). trackActors also does it when an actor disappears. */
export function markDeath(id: number, now: number): void {
	if (opts.reduced) return;
	const m = motion.get(id);
	if (!m?.last) return;
	ghosts.push({ a: m.last, x: m.cx, y: m.cy, t0: now });
	motion.delete(id);
}

/** Flash an actor white-hot for about 160 ms (uses the hit-flash support of actorsprites.ts). Works under reduceMotion too: it is not movement. */
export function flashActor(id: number, now: number): void {
	flashes.set(id, now + FLASH_MS);
	syncFlash();
}
/** Call each frame before drawing sprites so expired flashes clear. */
export function syncFlash(): void {
	setSpriteAnimTick(0, new Set(flashes.keys()));
}

export interface ActorMotion {
	/** offset in CELL units (multiply by cw / chh): slide, lunge, idle bob, hop, sink */
	dx: number;
	dy: number;
	/** tilt in radians about the cell's bottom centre (alternating foot while walking) */
	rot: number;
	/** 0 or 1: which foot of the 2-frame walk this is; 0 when standing still */
	step: 0 | 1;
	/** 1 normally; below 1 only for a fading ghost */
	alpha: number;
	/** true while sliding or lunging: do not clip the sprite to its own cell, and repaint the cell it came from */
	moving: boolean;
	/** the cell it slid from (valid when sliding) */
	fromX: number;
	fromY: number;
}

const REST: ActorMotion = { dx: 0, dy: 0, rot: 0, step: 0, alpha: 1, moving: false, fromX: 0, fromY: 0 };

/** Motion of actor `id` standing at cell (x, y) at time `now` (ms). Safe to call every frame; under reduceMotion it is always at rest. */
export function actorOffset(id: number, x: number, y: number, now: number): ActorMotion {
	if (opts.reduced) return REST;
	const m = mo(id, x, y, now);
	let dx = 0;
	let dy = 0;
	let rot = 0;
	let step: 0 | 1 = 0;
	let moving = false;
	const sp = (now - m.slideT0) / SLIDE_MS;
	if (sp >= 0 && sp < 1) {
		const e = 1 - (1 - sp) * (1 - sp); // ease out
		dx = (m.fx - x) * (1 - e);
		dy = (m.fy - y) * (1 - e);
		dy -= Math.sin(Math.PI * sp) * 0.07; // small hop
		moving = true;
	}
	const since = now - m.slideT0;
	if (since >= 0 && since < STEP_MS) {
		step = (m.steps & 1) as 0 | 1;
		rot = (step ? 1 : -1) * 0.06 * Math.sin((Math.PI * since) / STEP_MS);
	}
	const lp = (now - m.lungeT0) / LUNGE_MS;
	if (lp >= 0 && lp < 1) {
		const k = Math.sin(Math.PI * lp) * 0.38;
		dx += m.lx * k;
		dy += m.ly * k;
		moving = true;
	}
	if (!moving && since >= STEP_MS) dy += Math.sin(now / 520 + id * 1.7) * 0.035; // idle bob, about 1 px
	return { dx, dy, rot, step, alpha: 1, moving, fromX: m.fx, fromY: m.fy };
}

export interface ActorGhost {
	actor: QudActor;
	x: number;
	y: number;
	motion: ActorMotion; // alpha fades 1 -> 0, dy sinks, rot tips
}

/** Actors that just vanished (died), still fading for ~380 ms: draw them at (x, y) with `motion.alpha`. Empty under reduceMotion. */
export function actorGhosts(now: number): ActorGhost[] {
	if (opts.reduced || !ghosts.length) return [];
	const out: ActorGhost[] = [];
	for (const g of ghosts) {
		const p = (now - g.t0) / DEATH_MS;
		if (p < 0 || p >= 1) continue;
		out.push({ actor: g.a, x: g.x, y: g.y, motion: { dx: 0, dy: p * 0.12, rot: p * 0.35, step: 0, alpha: 1 - p, moving: true, fromX: g.x, fromY: g.y } });
	}
	return out;
}

/** True while any slide, lunge or death fade is in progress: repaint every animation frame then, otherwise only on tickTileAnim. */
export function motionActive(now: number): boolean {
	if (opts.reduced) return false;
	for (const m of motion.values()) {
		if (now - m.slideT0 < SLIDE_MS || now - m.lungeT0 < LUNGE_MS) return true;
	}
	return ghosts.some((g) => now - g.t0 < DEATH_MS);
}

/** Apply an ActorMotion to the canvas (inside save/restore): translate by the offset, tilt about the feet, fade. */
export function applyActorMotion(c: CanvasRenderingContext2D, m: ActorMotion, x: number, y: number, cw: number, chh: number): void {
	c.translate(m.dx * cw, m.dy * chh);
	if (m.rot) {
		const px = x + cw / 2;
		const py = y + chh * 0.92;
		c.translate(px, py);
		c.rotate(m.rot);
		c.translate(-px, -py);
	}
	if (m.alpha < 1) c.globalAlpha *= Math.max(0, m.alpha);
}
