// Limb poses for sprite-mode people and creatures: walk cycle (contact, passing, contact, passing),
// attack wind-up and strike, cast, hurt recoil, block guard, sleep, collapse, work (hammer, sweep) and
// idle breathing. spriteanim.ts keeps moving the whole sprite (slide, lunge, hop, fade); this file only
// says what the limbs do. It reads spriteanim's options (reduceMotion, tile clock) and keeps its own
// small per-actor trackers. actorsprites.ts (drawPersonSprite, drawCreatureSprite2) and gearsprites.ts
// (drawGear) read the pose themselves: poseOf -> limbAngles / creaturePose, and both apply the same
// whole-body frame through applyBodyFrame so worn gear lies down and mirrors with the body.
// Hot path: poseOf, limbAngles and creaturePose take an optional `out` object to fill instead of
// allocating; drawers keep one scratch object each.
//
// Clocks: idle breathing and work run on the 7 fps tile clock (tileAnimFrame), the walk, attack, cast,
// hurt, block and collapse run on the actor's own timers (a few hundred ms after an event). A still scene
// therefore costs nothing: poseActive() is false and poseOf() only reads the tile frame.
//
// Angle convention (limbs): radians from "straight down", positive = the tip swings to the RIGHT of the
// screen. A raised left arm is therefore about -2.5, a raised right arm +2.5. Positions are
// `pivot + len * (sin a, cos a)`. Cell fractions are in units of the cell (multiply by cw or chh).

import { tileAnimFrame, tileOptions } from './spriteanim';
import type { QudActor } from './actorsprites';
import type { HitEvent } from './hitfx';

type Ctx = CanvasRenderingContext2D;

// ---- prefs -----------------------------------------------------------------------------------

let enabled = true;
/** Pref `actorPoses` (default true). reduceMotion is read from spriteanim's tileOptions(). Off: every actor is at rest. */
export function setPosePrefs(p: { actorPoses?: boolean }): void {
	enabled = p.actorPoses !== false;
	if (!enabled) resetPoses();
}
const reduced = () => tileOptions().reduced;

// ---- the pose ---------------------------------------------------------------------------------

export interface Pose {
	/** walk frame: 0 contact, 1 passing, 2 contact (other foot), 3 passing */
	phase: 0 | 1 | 2 | 3;
	/** 1 while the actor is stepping (about 240 ms after each move), else 0 */
	walk: number;
	/** progress 0..1 of an attack, 0 = none. 0..0.4 wind-up (raise), 0.4..1 strike (swing arc) */
	attack: number;
	/** progress 0..1 of a cast (arms up, glow), 0 = none */
	cast: number;
	/** progress 0..1 of a hurt recoil, 0 = none */
	hurt: number;
	/** progress 0..1 of a block / parry guard, 0 = none */
	block: number;
	/** 1 when asleep (lying down), else 0. Static art: also shown under reduceMotion */
	sleep: number;
	/** collapse progress 0..1 (1 = fallen); 0 = alive */
	dead: number;
	/** 0..1 work cycle (hammer raised .. struck, sweep back .. forth); 0 when not working */
	work: number;
	workKind: '' | 'hammer' | 'sweep';
	/** idle breathing -1..1 (tile clock) */
	breath: number;
	/** tile clock step 0..7 for this actor (creature idle cycles) */
	tick: number;
	/** facing toward the last attack / move: 1 right, -1 left */
	face: 1 | -1;
	/** recoil and fall direction (1 = to the right) */
	dir: 1 | -1;
	/** 1 for a tracked, animating actor (idle cycles run); 0 for the static rest pose (pref off, reduceMotion, portraits of untracked actors) */
	live: 0 | 1;
}

const REST: Readonly<Pose> = Object.freeze({
	phase: 0, walk: 0, attack: 0, cast: 0, hurt: 0, block: 0, sleep: 0, dead: 0, work: 0, workKind: '', breath: 0, tick: 0, face: 1, dir: 1, live: 0,
}) as Pose;

/** Fill `out` (or return a fresh copy when not given) from REST plus overrides; no allocation when `out` is given. */
function restInto(out: Pose | undefined, sleep: number, dead: number, face: 1 | -1, dir: 1 | -1): Pose {
	if (!out) {
		if (!sleep && !dead && face === 1 && dir === 1) return REST;
		out = { ...REST };
	} else Object.assign(out, REST);
	out.sleep = sleep;
	out.dead = dead;
	out.face = face;
	out.dir = dir;
	return out;
}

// ---- trackers ---------------------------------------------------------------------------------

export const WALK_MS = 240;
const WALK_HALF_MS = 60; // contact lasts this long, then passing
const ATTACK_MS = 220;
const CAST_MS = 420;
const HURT_MS = 240;
const BLOCK_MS = 260;
const DEATH_MS = 380; // same as the death fade of spriteanim.ts

interface PS {
	x: number;
	y: number;
	steps: number;
	walkT0: number;
	attackT0: number;
	castT0: number;
	hurtT0: number;
	blockT0: number;
	face: 1 | -1;
	dir: 1 | -1;
	a: QudActor;
}
const st = new Map<number, PS>();
const dead = new Map<number, { a: QudActor; t0: number; face: 1 | -1; dir: 1 | -1 }>();
let sleepers = new Set<number>();
const NONE = -1e9;

export function resetPoses(): void {
	st.clear();
	dead.clear();
}

/** Ids of actors that are asleep (engine/poseinfo.go `PiSleepers`, or flag 'sleep' in QudActor.fl). */
export function setSleepers(ids: Iterable<number> | undefined): void {
	sleepers = new Set(ids ?? []);
}

/** Per-frame bookkeeping: call right after trackActors(frame.actors, now) with the same list. */
export function trackPoses(actors: QudActor[] | undefined, now: number): void {
	if (!enabled || reduced()) {
		if (st.size || dead.size) resetPoses();
		return;
	}
	const list = actors ?? [];
	const present = new Set<number>();
	for (const a of list) {
		present.add(a.i);
		let s = st.get(a.i);
		if (!s) {
			st.set(a.i, { x: a.x, y: a.y, steps: 0, walkT0: NONE, attackT0: NONE, castT0: NONE, hurtT0: NONE, blockT0: NONE, face: 1, dir: 1, a });
			continue;
		}
		s.a = a;
		if (s.x !== a.x || s.y !== a.y) {
			if (Math.hypot(a.x - s.x, a.y - s.y) <= 1.5) {
				s.walkT0 = now;
				s.steps++;
				if (a.x !== s.x) s.face = a.x > s.x ? 1 : -1;
			}
			s.x = a.x;
			s.y = a.y;
		}
	}
	const gone: number[] = [];
	for (const id of st.keys()) if (!present.has(id)) gone.push(id);
	if (gone.length) {
		const player = list.find((a) => a.p);
		const mass = gone.length > 3 && gone.length * 2 > st.size;
		for (const id of gone) {
			const s = st.get(id)!;
			st.delete(id);
			if (player && !mass && !s.a.p && !dead.has(id)) dead.set(id, { a: s.a, t0: now, face: s.face, dir: s.dir });
		}
	}
	for (const [id, d] of dead) if (now - d.t0 > 700) dead.delete(id);
}

const DMG = new Set(['hit', 'crit', 'burn', 'poison', 'bleed']);

/** Feed `frame.hits` once per frame (next to noteAttack/flashActor): triggers attack, cast, block, hurt and collapse poses. */
export function notePoseHits(hits: HitEvent[] | undefined, now: number): void {
	if (!enabled || reduced() || !hits) return;
	for (const e of hits) {
		const att = e.attackerId ? st.get(e.attackerId) : undefined;
		const tgt = e.targetId ? st.get(e.targetId) : undefined;
		if (att && e.kind !== 'death') {
			const spell = (!!e.elem || e.kind === 'heal') && !att.a.w && !(e.kind === 'heal' && e.attackerId === e.targetId && att.a.p);
			if (e.kind === 'heal' && !spell) {
				// a potion or plain heal: no pose
			} else if (spell) {
				att.castT0 = now;
			} else if (e.kind !== 'heal') {
				att.attackT0 = now;
				if (e.x !== att.x) att.face = e.x > att.x ? 1 : -1;
			}
		}
		if (tgt) {
			if (e.kind === 'block' || e.kind === 'parry') tgt.blockT0 = now;
			if (DMG.has(e.kind)) {
				tgt.hurtT0 = now;
				tgt.dir = att && att.x !== tgt.x ? (tgt.x > att.x ? 1 : -1) : ((-tgt.face) as 1 | -1);
			}
			if (e.kind === 'death' && !dead.has(e.targetId)) {
				dead.set(e.targetId, { a: tgt.a, t0: now, face: tgt.face, dir: tgt.dir });
				st.delete(e.targetId);
			}
		}
	}
}

// ---- reading a pose ---------------------------------------------------------------------------

/** progress (0,1) of a timer, 0 when idle */
function env(t0: number, ms: number, now: number): number {
	const p = (now - t0) / ms;
	return p < 0 || p >= 1 ? 0 : Math.max(p, 0.001);
}

const WORK_ROLES: Record<string, 'hammer' | 'sweep'> = {
	smith: 'hammer', armorer: 'hammer', weaponsmith: 'hammer', blacksmith: 'hammer', carpenter: 'hammer', miner: 'hammer', mason: 'hammer', tinker: 'hammer',
	farmer: 'sweep', sweeper: 'sweep', laborer: 'sweep', labourer: 'sweep', servant: 'sweep', innkeeper: 'sweep', cook: 'sweep',
};

const asleepNow = (id: number, a?: QudActor) => sleepers.has(id) || !!a?.fl?.some((f) => f === 'sleep' || f === 'asleep' || f === 'sleeping');

/** The pose of actor `id` at `now` (ms). Cheap; at rest under reduceMotion or when the `actorPoses` pref is off. Works for dying actors (ghosts) too. */
export function poseOf(id: number, now: number, out?: Pose): Pose {
	if (!enabled) return restInto(out, 0, 0, 1, 1);
	const red = reduced();
	const d = dead.get(id);
	if (d) return red ? restInto(out, 0, 0, 1, 1) : restInto(out, 0, Math.max(0.001, Math.min(1, (now - d.t0) / DEATH_MS)), d.face, d.dir);
	const s = st.get(id);
	if (!s) return sleepers.has(id) ? restInto(out, 1, 0, 1, id & 1 ? 1 : -1) : restInto(out, 0, 0, 1, 1);
	const asleep = asleepNow(id, s.a);
	const dir: 1 | -1 = id & 1 ? 1 : -1;
	if (red) return asleep ? restInto(out, 1, 0, 1, dir) : restInto(out, 0, 0, 1, 1);
	const f = tileAnimFrame();
	const tick = (f + id) & 7;
	const since = now - s.walkT0;
	const walking = since >= 0 && since < WALK_MS;
	const phase = ((s.steps * 2 + (since < WALK_HALF_MS ? 0 : 1)) & 3) as 0 | 1 | 2 | 3;
	const kind = !asleep && !walking && !s.a.ho && !s.a.p ? ((s.a.fl?.includes('work') ? 'hammer' : '') || WORK_ROLES[s.a.r ?? ''] || '') : '';
	const wk = (f + id) & 7;
	const o = out ?? ({} as Pose);
	o.phase = walking ? phase : 0;
	o.walk = walking ? 1 : 0;
	o.attack = env(s.attackT0, ATTACK_MS, now);
	o.cast = env(s.castT0, CAST_MS, now);
	o.hurt = env(s.hurtT0, HURT_MS, now);
	o.block = env(s.blockT0, BLOCK_MS, now);
	o.sleep = asleep ? 1 : 0;
	o.dead = 0;
	o.work = kind ? (wk < 4 ? wk / 3 : (7 - wk) / 3) : 0;
	o.workKind = kind;
	o.breath = Math.sin(asleep ? (Math.PI * ((f + id) & 15)) / 8 : (Math.PI * tick) / 4);
	o.tick = tick;
	o.face = s.face;
	o.dir = asleep ? dir : s.dir;
	o.live = 1;
	return o;
}

/**
 * Ghosts of dying actors (spriteanim actorGhosts): poseOf(id) already returns the collapse (`dead`
 * progress 0..1, lie + limbs), so when a drawer that reads poses paints a ghost, keep only the ghost's
 * fade and slide and drop spriteanim's own tip and sink, or the body would rotate twice. Returns the
 * motion unchanged when the actor has no collapse pose (poses off, reduceMotion). Mutates `m`.
 */
export function poseGhostMotion<M extends { rot: number; dy: number }>(m: M, id: number, now: number): M {
	if (!enabled || reduced() || !dead.has(id)) return m;
	m.rot = 0;
	m.dy = 0;
	return m;
}

/** True while any actor is mid walk step, attack, cast, hurt, block or collapse: repaint every animation frame then (OR it with spriteanim's motionActive). */
export function poseActive(now: number): boolean {
	if (!enabled || reduced()) return false;
	for (const s of st.values()) {
		if (now - s.walkT0 < WALK_MS || now - s.attackT0 < ATTACK_MS || now - s.castT0 < CAST_MS || now - s.hurtT0 < HURT_MS || now - s.blockT0 < BLOCK_MS) return true;
	}
	for (const d of dead.values()) if (now - d.t0 < DEATH_MS) return true;
	return false;
}

// ---- keyframe tables --------------------------------------------------------------------------

type KF = readonly (readonly [number, number])[];
function kf(t: KF, x: number): number {
	if (x <= t[0][0]) return t[0][1];
	for (let i = 1; i < t.length; i++) {
		if (x <= t[i][0]) {
			const [x0, y0] = t[i - 1];
			const [x1, y1] = t[i];
			return y0 + ((y1 - y0) * (x - x0)) / (x1 - x0 || 1);
		}
	}
	return t[t.length - 1][1];
}

export interface Limbs {
	/** radians from straight down, + = tip to the right of the screen */
	armL: number;
	armR: number;
	legL: number;
	legR: number;
	/** radians, + = leans right (upper body, about the hips) */
	torsoLean: number;
	/** cell fraction, + = down */
	headBob: number;
	/** extra (not in the brief): whole body drop and shift in cell fractions, foot lift in cell fractions, lie angle of the whole body in radians about the feet */
	drop: number;
	dx: number;
	liftL: number;
	liftR: number;
	lie: number;
}

const REST_LIMBS: Readonly<Limbs> = { armL: -0.18, armR: 0.18, legL: -0.04, legR: 0.04, torsoLean: 0, headBob: 0, drop: 0, dx: 0, liftL: 0, liftR: 0, lie: 0 };

type Track = { [K in keyof Limbs]?: KF } & { sgn?: boolean }; // sgn: torsoLean, dx and lie follow pose.dir

// generic overlays, sampled at pose[key] (0..1). Later ones in LAYER_ORDER win per field.
const TRACKS: Record<'block' | 'attack' | 'cast' | 'hurt' | 'sleep' | 'dead', Track> = {
	block: {
		armL: [[0, -0.18], [0.2, 2.4], [0.8, 2.4], [1, -0.18]],
		armR: [[0, 0.18], [0.2, 1.0], [0.8, 1.0], [1, 0.18]],
		torsoLean: [[0, 0], [0.2, 0.07], [0.8, 0.07], [1, 0]],
		drop: [[0, 0], [0.2, 0.025], [0.8, 0.025], [1, 0]],
	},
	attack: {
		armR: [[0, 0.18], [0.4, 2.7], [0.55, 2.2], [0.75, 1.3], [1, 0.2]],
		armL: [[0, -0.18], [0.4, -0.9], [1, -0.2]],
		torsoLean: [[0, 0], [0.4, -0.12], [0.6, 0.2], [1, 0]],
		drop: [[0, 0], [0.4, -0.01], [0.6, 0.02], [1, 0]],
		headBob: [[0, 0], [0.4, -0.01], [0.6, 0.015], [1, 0]],
	},
	cast: {
		armL: [[0, -0.18], [0.3, -2.5], [0.8, -2.7], [1, -0.2]],
		armR: [[0, 0.18], [0.3, 2.5], [0.8, 2.7], [1, 0.2]],
		torsoLean: [[0, 0], [0.3, -0.06], [0.8, -0.06], [1, 0]],
		headBob: [[0, 0], [0.3, -0.02], [0.8, -0.02], [1, 0]],
	},
	hurt: {
		sgn: true,
		armL: [[0, -0.18], [0.3, -1.0], [1, -0.18]],
		armR: [[0, 0.18], [0.3, 1.0], [1, 0.18]],
		torsoLean: [[0, 0], [0.3, 0.38], [1, 0]],
		dx: [[0, 0], [0.3, 0.05], [1, 0]],
		headBob: [[0, 0], [0.3, -0.015], [1, 0]],
		drop: [[0, 0], [0.3, 0.012], [1, 0]],
	},
	sleep: {
		sgn: true,
		armL: [[0, -0.5], [1, -0.5]],
		armR: [[0, 0.5], [1, 0.5]],
		legL: [[0, -0.12], [1, -0.12]],
		legR: [[0, 0.12], [1, 0.12]],
		lie: [[0, 1.45], [1, 1.45]],
		torsoLean: [[0, 0], [1, 0]],
	},
	dead: {
		sgn: true,
		armL: [[0, -0.18], [0.4, -1.1], [1, -0.9]],
		armR: [[0, 0.18], [0.4, 1.1], [1, 0.9]],
		legL: [[0, -0.04], [1, -0.35]],
		legR: [[0, 0.04], [1, 0.3]],
		lie: [[0, 0], [0.5, 0.7], [1, 1.5]],
		torsoLean: [[0, 0], [0.5, 0.2], [1, 0]],
		headBob: [[0, 0], [1, 0.015]],
	},
};
const LAYER_ORDER = ['block', 'attack', 'cast', 'hurt', 'sleep', 'dead'] as const;

// walk rows: [arm swing, legL, legR, liftL, liftR, body bob] (arms swing opposite: armL = rest + s, armR = rest - s)
const WALK: readonly (readonly [number, number, number, number, number, number])[] = [
	[0.35, -0.3, 0.14, 0, 0, 0.012],
	[0, -0.05, 0.05, 0, 0.06, -0.018],
	[-0.35, -0.14, 0.3, 0, 0, 0.012],
	[0, -0.05, 0.05, 0.06, 0, -0.018],
];

// work: [value at work=0, value at work=1] per field
const WORK: Record<'hammer' | 'sweep', { [K in 'armL' | 'armR' | 'torsoLean' | 'drop' | 'headBob']: readonly [number, number] }> = {
	hammer: { armR: [0.3, 2.3], armL: [-0.5, -0.5], torsoLean: [0.14, -0.04], drop: [0.012, 0], headBob: [0.01, 0] },
	sweep: { armR: [0.7, 1.3], armL: [0.4, 1.0], torsoLean: [0.1, -0.1], drop: [0.01, 0.01], headBob: [0.01, 0.01] },
};

export interface Build {
	/** height 0.6..1 (short people take shorter strides) */
	h?: number;
	/** width multiplier (wider stance) */
	w?: number;
}

/** Rest arm angle: armL rests at -ARM_REST, armR at +ARM_REST (limbAngles at rest). Drawers rotate by the difference. */
export const ARM_REST = 0.18;
/** Rest leg angle for a build of width `w`: legL rests at -legRest(w), legR at +legRest(w). */
export const legRest = (w = 1): number => 0.04 + (w - 1) * 0.05;

const WORK_KEYS = { hammer: Object.keys(WORK.hammer), sweep: Object.keys(WORK.sweep) } as Record<'hammer' | 'sweep', ('armL' | 'armR' | 'torsoLean' | 'drop' | 'headBob')[]>;
const TRACK_KEYS = {} as Record<(typeof LAYER_ORDER)[number], (keyof Limbs)[]>;
for (const key of LAYER_ORDER) TRACK_KEYS[key] = Object.keys(TRACKS[key]).filter((k) => k !== 'sgn') as (keyof Limbs)[];

/** Limb angles of a person in `pose` (fills `out` when given). The caller mirrors the canvas about the cell centre when `pose.attack > 0 && pose.face < 0`. */
export function limbAngles(pose: Pose, build: Build = {}, out?: Limbs): Limbs {
	const o: Limbs = out ? Object.assign(out, REST_LIMBS) : { ...REST_LIMBS };
	const h = build.h ?? 1;
	const w = build.w ?? 1;
	o.legL -= (w - 1) * 0.05;
	o.legR += (w - 1) * 0.05;
	// idle breathing
	o.headBob = pose.breath * 0.008;
	o.armL += pose.breath * 0.03;
	o.armR -= pose.breath * 0.03;
	if (pose.workKind) {
		const t = WORK[pose.workKind];
		for (const k of WORK_KEYS[pose.workKind]) o[k] = t[k][0] + (t[k][1] - t[k][0]) * pose.work;
	}
	if (pose.walk > 0) {
		const r = WALK[pose.phase];
		const amp = (0.75 + 0.25 * h) * pose.walk;
		o.armL = REST_LIMBS.armL + r[0] * pose.walk;
		o.armR = REST_LIMBS.armR - r[0] * pose.walk;
		o.legL = r[1] * amp;
		o.legR = r[2] * amp;
		o.liftL = r[3] * pose.walk;
		o.liftR = r[4] * pose.walk;
		o.drop = r[5] * pose.walk;
		o.torsoLean = 0;
		o.headBob = r[5] * pose.walk;
	}
	for (const key of LAYER_ORDER) {
		const t = pose[key];
		if (t <= 0) continue;
		const tr = TRACKS[key];
		for (const k of TRACK_KEYS[key]) {
			let v = kf(tr[k]!, t);
			if (tr.sgn && (k === 'torsoLean' || k === 'dx' || k === 'lie')) v *= pose.dir;
			o[k] = v;
		}
	}
	return o;
}

/** 0..1 glow strength for the casting hands (draw a soft circle at both hands with this alpha). */
export const castGlow = (pose: Pose): number => (pose.cast > 0 ? Math.sin(Math.PI * pose.cast) : 0);

/**
 * The whole-body frame shared by drawPersonSprite and drawGear (call inside save/restore, before
 * drawing the body). `cx` is the CELL centre x (not shifted by pose dx), `ground` the feet line
 * (y + 0.9 cell, with the idle bob), `H` the body height in px, `u` the cell width.
 * - lie (sleep, death): canvas rotate(+lie) about the body's middle (cx, ground - H/2); canvas
 *   rotation is clockwise on screen (y down), so lie > 0 swings the head to the RIGHT (pose.dir 1 =
 *   falls to the right). The pivot is lowered toward the ground by k = |lie|/1.4 so the lying body
 *   rests about 0.12 cell above the feet line, and the figure is scaled down (to about 0.95 cell long)
 *   so the head is not clipped by the cell.
 * - mirror: scale(-1, 1) about cx (attack toward the left: the sprite's weapon hand is its right).
 */
export function applyBodyFrame(c: Ctx, lie: number, mirror: boolean, cx: number, ground: number, H: number, u: number): void {
	if (lie) {
		const k = Math.min(1, Math.abs(lie) / 1.4);
		const my = ground - H * 0.5;
		const s = 1 - k * (1 - Math.min(1, (u * 0.95) / Math.max(1, H)));
		c.translate(cx, my + (H * 0.5 - u * 0.12) * k);
		c.rotate(lie);
		if (s !== 1) c.scale(s, s);
		c.translate(-cx, -my);
	}
	if (mirror) {
		c.translate(cx, 0);
		c.scale(-1, 1);
		c.translate(-cx, 0);
	}
}

/** A soft cast glow (radial gradient) at (x, y); `col` must be #rrggbb. Allocates a gradient: only call while casting. */
export function drawCastGlow(c: Ctx, x: number, y: number, r: number, col: string, a: number): void {
	glowAt(c, x, y, r, col, a);
}

// ---- creatures ---------------------------------------------------------------------------------

export interface CreatureParams {
	/** leg swing in radians for the two diagonal pairs (quadruped, saurian, insect, arachnid, crab, golem) */
	legA: number;
	legB: number;
	/** wing flap in radians above horizontal (avian) */
	wing: number;
	/** body wave: phase in radians and amplitude 0..1 (serpent; fish body) */
	undPhase: number;
	undAmp: number;
	/** tail wag in radians (quadruped, saurian, fish) */
	tail: number;
	/** squash and stretch added to 1 (ooze; also a sleeping curl) */
	sx: number;
	sy: number;
	/** tentacle sway amplitude in radians (tentacled); phase is undPhase */
	sway: number;
	/** 0..1 foot-down thump (golem) */
	stomp: number;
	/** head/body thrust toward the target in cell fractions (attack), negative while winding up */
	lunge: number;
	/** body bob (cell fraction, + = down), shift on recoil (cell fraction), tilt and lie in radians */
	dy: number;
	dx: number;
	tilt: number;
	lie: number;
	/** 0..1 cast glow */
	glow: number;
}

type Gain = { walk?: number; idle?: number; atk?: number; hurt?: number; cast?: number; block?: number; sleep?: number; ph?: number };
type PlanTable = { [K in 'legA' | 'legB' | 'wing' | 'undAmp' | 'tail' | 'sx' | 'sy' | 'sway' | 'stomp' | 'lunge' | 'dy']?: Gain };

const LEGS_Q: PlanTable = {
	legA: { walk: 0.55 },
	legB: { walk: 0.55, ph: Math.PI },
	tail: { walk: 0.3, idle: 0.25, ph: 1 },
	lunge: { atk: 0.3 },
	dy: { walk: 0.02, idle: 0.008, ph: Math.PI / 2, sleep: 0.06, hurt: 0.02 },
	sy: { sleep: -0.15 },
	sx: { sleep: 0.1 },
};

const PLANS: Record<string, PlanTable> = {
	quadruped: LEGS_Q,
	saurian: { ...LEGS_Q, legA: { walk: 0.4 }, legB: { walk: 0.4, ph: Math.PI }, tail: { walk: 0.5, idle: 0.3, ph: 1 } },
	avian: {
		legA: { walk: 0.45 },
		legB: { walk: 0.45, ph: Math.PI },
		wing: { walk: 0.5, idle: 0.12, atk: 0.9, hurt: 0.8, cast: 0.7 },
		dy: { walk: 0.03, idle: 0.008, ph: Math.PI / 2, sleep: 0.05 },
		lunge: { atk: 0.2 },
		sy: { sleep: -0.2 },
	},
	serpent: { undAmp: { walk: 0.35, idle: 0.12, atk: 0.3, hurt: 0.4 }, lunge: { atk: 0.35 }, dy: { sleep: 0.05 }, sy: { sleep: -0.2 } },
	insect: { legA: { walk: 0.6 }, legB: { walk: 0.6, ph: Math.PI }, dy: { walk: 0.012, idle: 0.006 }, lunge: { atk: 0.2 }, sy: { sleep: -0.15 } },
	arachnid: { legA: { walk: 0.6 }, legB: { walk: 0.6, ph: Math.PI }, dy: { walk: 0.012, idle: 0.006 }, lunge: { atk: 0.25 }, sy: { sleep: -0.15 } },
	crab: { legA: { walk: 0.5 }, legB: { walk: 0.5, ph: Math.PI }, dy: { walk: 0.01, idle: 0.006 }, lunge: { atk: 0.2 }, wing: { atk: 0.6, idle: 0.15, cast: 0.5 }, sy: { sleep: -0.15 } },
	fish: { tail: { walk: 0.6, idle: 0.35 }, undAmp: { walk: 0.25, idle: 0.1 }, dy: { idle: 0.015, walk: 0.012, ph: 1 }, lunge: { atk: 0.2 } },
	ooze: {
		sy: { walk: 0.12, idle: 0.06, atk: -0.1, hurt: -0.25, block: -0.12, sleep: -0.1 },
		sx: { walk: -0.1, idle: -0.05, atk: 0.15, hurt: 0.25, block: 0.12, sleep: 0.1, ph: 0 },
		lunge: { atk: 0.2 },
	},
	tentacled: { sway: { walk: 0.3, idle: 0.35, atk: 0.4, hurt: 0.5, cast: 0.3 }, dy: { idle: 0.015, walk: 0.015, ph: 1 }, lunge: { atk: 0.15 } },
	golem: {
		legA: { walk: 0.35 },
		legB: { walk: 0.35, ph: Math.PI },
		stomp: { walk: 1, ph: Math.PI / 2 },
		dy: { walk: 0.03, ph: Math.PI / 2, hurt: 0.01 },
		lunge: { atk: 0.15 },
	},
};

const ATK_K: KF = [[0, 0], [0.4, -0.6], [0.6, 1], [1, 0]];
const HURT_K: KF = [[0, 0], [0.25, 1], [1, 0]];
const BELL: KF = [[0, 0], [0.3, 1], [0.8, 1], [1, 0]];

const PLAN_KEYS: Record<string, (keyof PlanTable)[]> = {};
const DEFAULT_PLAN: PlanTable = { dy: { idle: 0.008, walk: 0.02, ph: Math.PI / 2 } };
for (const k of Object.keys(PLANS)) PLAN_KEYS[k] = Object.keys(PLANS[k]) as (keyof PlanTable)[];
const DEFAULT_KEYS = Object.keys(DEFAULT_PLAN) as (keyof PlanTable)[];

/** Per-body-plan parameters for `pose` (fills `out` when given): plans are quadruped, saurian, avian, fish, insect, arachnid, crab, serpent, ooze, tentacled, golem (anything else gets a gentle idle bob). The static rest pose (live 0) gives all zeros, so drawers render their old art. */
export function creaturePose(bodyPlan: string, pose: Pose, out?: CreatureParams): CreatureParams {
	const p: CreatureParams = out ?? ({} as CreatureParams);
	p.legA = p.legB = p.wing = p.undPhase = p.undAmp = p.tail = p.sx = p.sy = p.sway = p.stomp = p.lunge = p.dy = p.dx = p.tilt = p.lie = p.glow = 0;
	const table = PLANS[bodyPlan] ?? DEFAULT_PLAN;
	const keys = PLAN_KEYS[bodyPlan] ?? DEFAULT_KEYS;
	const walk = pose.walk;
	const cycW = (pose.phase * Math.PI) / 2;
	const cycI = (pose.tick * Math.PI) / 4;
	const atk = pose.attack > 0 ? kf(ATK_K, pose.attack) : 0;
	const hurt = pose.hurt > 0 ? kf(HURT_K, pose.hurt) : 0;
	const cast = pose.cast > 0 ? Math.sin(Math.PI * pose.cast) : 0;
	const block = pose.block > 0 ? kf(BELL, pose.block) : 0;
	// idle cycles only for tracked actors: the static rest pose (live 0) stays exactly still
	const idleOn = pose.live ? (pose.sleep ? 0.5 : 1) : 0;
	for (const k of keys) {
		const g = table[k]!;
		const ph = g.ph ?? 0;
		p[k] =
			(g.walk ? walk * g.walk * Math.sin(cycW + ph) : 0) +
			(g.idle ? (1 - walk) * idleOn * g.idle * Math.sin(cycI + ph) : 0) +
			(g.atk ? g.atk * atk : 0) +
			(g.hurt ? g.hurt * hurt : 0) +
			(g.cast ? g.cast * cast : 0) +
			(g.block ? g.block * block : 0) +
			(g.sleep ? g.sleep * pose.sleep : 0);
	}
	// attack and hurt do not undulate or wag with the 4-frame cycle: the amplitudes above only grow
	if (p.undAmp < 0) p.undAmp = -p.undAmp;
	p.undPhase = walk ? cycW : cycI;
	if (p.stomp < 0) p.stomp = 0;
	p.glow = cast;
	p.dx = pose.hurt > 0 ? pose.dir * 0.05 * hurt : 0;
	p.tilt = -pose.dir * 0.22 * hurt;
	p.lie = pose.dead > 0 ? pose.dir * (pose.dead < 0.5 ? pose.dead * 1.4 : 0.7 + (pose.dead - 0.5) * 1.6) : 0;
	if (pose.dead > 0) p.dy += 0.04 * pose.dead;
	return p;
}

/** Body plan of an actor (same rules as actorsprites.ts bodyPlanOf, without the glyph fallback). */
export function planOf(a: QudActor): string {
	if (a.b) return a.b;
	switch (a.gr) {
		case 'mam': case 'syn': return 'quadruped';
		case 'bird': case 'ptero': return 'avian';
		case 'rept': case 'dino': case 'amph': return 'saurian';
		case 'fish': return 'fish';
		case 'ins': case 'arth': return 'insect';
		case 'arach': return 'arachnid';
		case 'crust': return 'crab';
		case 'myria': case 'worm': return 'serpent';
		case 'moll': return 'ooze';
		case 'cnid': case 'echi': return 'tentacled';
	}
	return 'humanoid';
}

// ---- fallback renderers -------------------------------------------------------------------------

export interface PoseColors {
	skin: string;
	cloth: string;
	dark?: string;
	trim?: string;
	steel?: string;
	hair?: string;
	eye?: string;
	glow?: string;
}

function ln(c: Ctx, x0: number, y0: number, x1: number, y1: number, w: number) {
	c.lineWidth = w;
	c.lineCap = 'round';
	c.beginPath();
	c.moveTo(x0, y0);
	c.lineTo(x1, y1);
	c.stroke();
}
function el(c: Ctx, x: number, y: number, rx: number, ry: number) {
	c.beginPath();
	c.ellipse(x, y, Math.max(0.3, rx), Math.max(0.3, ry), 0, 0, Math.PI * 2);
	c.fill();
}
function pg(c: Ctx, pts: number[]) {
	c.beginPath();
	c.moveTo(pts[0], pts[1]);
	for (let i = 2; i < pts.length; i += 2) c.lineTo(pts[i], pts[i + 1]);
	c.closePath();
	c.fill();
}
function glowAt(c: Ctx, x: number, y: number, r: number, col: string, a: number) {
	const g = c.createRadialGradient(x, y, 0.5, x, y, r);
	g.addColorStop(0, col);
	g.addColorStop(1, col + '00');
	c.save();
	c.globalAlpha *= a;
	c.fillStyle = g;
	c.beginPath();
	c.arc(x, y, r, 0, Math.PI * 2);
	c.fill();
	c.restore();
}

/** A held weapon continuing the arm in direction `th` (same convention as the limbs). */
function heldItem(c: Ctx, skill: string | undefined, hx: number, hy: number, th: number, u: number, v: number, col: PoseColors) {
	if (!skill || skill === 'fist' || skill === 'natural') return;
	const sx = Math.sin(th);
	const sy = Math.cos(th);
	const wood = '#7a5a3a';
	const steel = col.steel ?? '#c9ccd2';
	const len = skill === 'spear' || skill === 'staff' ? v * 0.5 : skill === 'dagger' ? v * 0.16 : v * 0.3;
	const bx = hx - sx * len * 0.2;
	const by = hy - sy * len * 0.2;
	const tx = hx + sx * len;
	const ty = hy + sy * len;
	if (skill === 'bow') {
		c.strokeStyle = wood;
		c.lineWidth = 1;
		c.beginPath();
		c.arc(hx, hy, v * 0.16, th - 1.2, th + 1.2);
		c.stroke();
		return;
	}
	const handleOnly = skill === 'heavy' || skill === 'hammer' || skill === 'flail' || skill === 'staff';
	c.strokeStyle = handleOnly ? wood : steel;
	ln(c, bx, by, tx, ty, Math.max(1, u * 0.08));
	if (skill === 'heavy' || skill === 'hammer') {
		c.fillStyle = steel;
		el(c, tx, ty, u * 0.14, u * 0.1);
	} else if (skill === 'spear') {
		c.fillStyle = steel;
		pg(c, [tx + sx * v * 0.1, ty + sy * v * 0.1, tx - sy * u * 0.07, ty + sx * u * 0.07, tx + sy * u * 0.07, ty - sx * u * 0.07]);
	}
}

/** A simple posed stick-and-shape person for the coordinator when drawPersonSprite cannot take a pose. Cell (x, y, cw, chh); `build` from the species height/width if known. */
export function drawPosedPerson(c: Ctx, a: QudActor, pose: Pose, x: number, y: number, cw: number, chh: number, col: PoseColors, build: Build = { h: 0.95, w: 1 }): void {
	const L = limbAngles(pose, build);
	const u = cw;
	const v = chh;
	const bh = build.h ?? 0.95;
	const bw = build.w ?? 1;
	const cx = x + u * 0.5 + L.dx * u;
	const base0 = y + v * 0.9;
	const base = base0 + L.drop * v;
	const H = v * 0.82 * bh;
	const W = u * 0.46 * bw;
	const hr = u * 0.2;
	const headY = base - H + hr + L.headBob * v;
	const neckY = headY + hr * 0.85;
	const hipY = base - H * 0.34;
	const dark = col.dark ?? '#3a3a40';
	c.save();
	if (L.lie) {
		const k = Math.min(1, Math.abs(L.lie) / 1.4);
		c.translate(cx, base0 - H * 0.5 + (H * 0.5 - u * 0.12) * k);
		c.rotate(L.lie);
		c.translate(-cx, -(base0 - H * 0.5));
	}
	if (pose.attack > 0 && pose.face < 0) {
		c.translate(cx, 0);
		c.scale(-1, 1);
		c.translate(-cx, 0);
	}
	// legs
	const legLen = base - hipY;
	const lw = Math.max(1.5, u * 0.14);
	c.strokeStyle = dark;
	c.fillStyle = shadeBoot(dark);
	for (const [sx, th, lift] of [[-1, L.legL, L.liftL], [1, L.legR, L.liftR]] as const) {
		const hx = cx + sx * W * 0.32;
		const fx = hx + Math.sin(th) * legLen;
		const fy = hipY + Math.cos(th) * legLen - lift * v;
		ln(c, hx, hipY, fx, fy, lw);
		el(c, fx + sx * lw * 0.2, fy, lw * 0.75, v * 0.025);
	}
	// upper body, leaning about the hips
	c.save();
	c.translate(cx, hipY);
	c.rotate(L.torsoLean);
	c.translate(-cx, -hipY);
	c.fillStyle = col.cloth;
	pg(c, [cx - W * 0.5, neckY, cx + W * 0.5, neckY, cx + W * 0.42, hipY + v * 0.04, cx - W * 0.42, hipY + v * 0.04]);
	c.fillStyle = col.trim ?? dark;
	c.fillRect(cx - W * 0.44, hipY - v * 0.02, W * 0.88, Math.max(1, v * 0.03));
	// arms
	const aLen = (hipY - neckY) * 0.95;
	const aw = Math.max(1.4, u * 0.1);
	const shY = neckY + v * 0.04;
	const hands: number[] = [];
	for (const [sx, th] of [[-1, L.armL], [1, L.armR]] as const) {
		const sh = cx + sx * W * 0.5;
		const hx = sh + Math.sin(th) * aLen;
		const hy = shY + Math.cos(th) * aLen;
		c.strokeStyle = col.cloth;
		ln(c, sh, shY, hx, hy, aw);
		c.fillStyle = col.skin;
		el(c, hx, hy, u * 0.05, u * 0.05);
		hands.push(hx, hy);
	}
	heldItem(c, a.w, hands[2], hands[3], L.armR, u, v, col);
	if (a.sh) {
		c.fillStyle = col.trim ?? dark;
		el(c, hands[0], hands[1], u * 0.12, v * 0.1);
	}
	// head
	c.fillStyle = col.skin;
	el(c, cx, headY, hr, hr);
	if (col.hair) {
		c.fillStyle = col.hair;
		c.beginPath();
		c.ellipse(cx, headY - hr * 0.1, hr * 1.02, hr * 0.9, 0, Math.PI, Math.PI * 2);
		c.fill();
	}
	c.fillStyle = col.eye ?? '#101010';
	if (pose.sleep || pose.dead > 0.5) ln(c, cx - hr * 0.5, headY + hr * 0.1, cx + hr * 0.5, headY + hr * 0.1, 1);
	else {
		el(c, cx - hr * 0.4, headY + hr * 0.05, 0.6, 0.6);
		el(c, cx + hr * 0.4, headY + hr * 0.05, 0.6, 0.6);
	}
	const gl = castGlow(pose);
	if (gl > 0) {
		const gc = col.glow ?? '#9fd8ff';
		glowAt(c, hands[0], hands[1], u * 0.4, gc, gl);
		glowAt(c, hands[2], hands[3], u * 0.4, gc, gl);
	}
	c.restore();
	c.restore();
}

function shadeBoot(col: string): string {
	return col; // boots share the leg colour in the fallback
}

export interface CreatureColors {
	body: string;
	belly?: string;
	dark?: string;
	eye?: string;
	glow?: string;
}

/** A simple posed creature (quadruped, saurian, avian, fish, insect/arachnid/crab, serpent, ooze, tentacled, golem) for the coordinator to use when drawCreatureSprite2 cannot take a pose. */
export function drawPosedCreature(c: Ctx, a: QudActor, pose: Pose, x: number, y: number, cw: number, chh: number, col: CreatureColors, plan: string = planOf(a)): void {
	const p = creaturePose(plan === 'humanoid' ? 'quadruped' : plan, pose);
	const u = cw;
	const v = chh;
	const s = a.z ? 0.55 + a.z * 0.08 : 0.8;
	const body = col.body;
	const belly = col.belly ?? body;
	const dark = col.dark ?? '#202020';
	const eye = col.eye ?? '#101818';
	const ground = y + v * 0.9;
	const f = pose.face;
	c.save();
	c.translate(x + u * 0.5 + p.dx * u, ground);
	c.rotate(p.tilt + p.lie);
	c.scale(f, 1);
	c.translate(p.lunge * u * 0.5, p.dy * v);
	c.fillStyle = body;
	c.strokeStyle = body;
	const lw = Math.max(1, u * 0.1);
	const lunge = p.lunge * u;
	const sx = 1 + p.sx;
	const sy = 1 + p.sy;
	switch (plan) {
		case 'avian': {
			const cy = -v * 0.36 * s;
			c.strokeStyle = dark;
			for (const [k, th] of [[-1, p.legA], [1, p.legB]] as const) ln(c, k * u * 0.07, cy + v * 0.08 * s, k * u * 0.07 + Math.sin(th) * v * 0.16, -v * 0.01, 1);
			c.fillStyle = body;
			el(c, 0, cy, u * 0.2 * s * sx, v * 0.15 * s * sy);
			const w = 0.3 + p.wing;
			const wl = u * 0.42 * s;
			c.fillStyle = belly;
			for (const k of [-1, 1]) pg(c, [k * u * 0.12 * s, cy - v * 0.02, k * (u * 0.12 * s + Math.cos(w) * wl), cy - Math.sin(w) * wl, k * u * 0.16 * s, cy + v * 0.08 * s]);
			c.fillStyle = body;
			el(c, lunge + u * 0.06, cy - v * 0.17 * s, u * 0.1 * s, u * 0.1 * s);
			c.fillStyle = '#e8b030';
			pg(c, [lunge + u * 0.15 * s, cy - v * 0.18 * s, lunge + u * 0.27 * s, cy - v * 0.15 * s, lunge + u * 0.15 * s, cy - v * 0.13 * s]);
			c.fillStyle = eye;
			el(c, lunge + u * 0.09, cy - v * 0.19 * s, 0.7, 0.7);
			break;
		}
		case 'serpent': {
			c.fillStyle = body;
			const n = 8;
			let hx = 0;
			let hy = 0;
			for (let i = 0; i < n; i++) {
				const t = i / (n - 1);
				const px = (-0.42 + 0.84 * t) * u * s + (i === n - 1 ? lunge : 0);
				const py = -v * 0.12 * s + Math.sin(p.undPhase + i * 0.9) * p.undAmp * v * 0.18 * s;
				el(c, px, py, u * (0.05 + 0.07 * Math.sin(Math.PI * (0.15 + 0.85 * t))) * s * 1.2, u * (0.05 + 0.05 * Math.sin(Math.PI * (0.15 + 0.85 * t))) * s);
				hx = px;
				hy = py;
			}
			c.fillStyle = eye;
			el(c, hx + u * 0.03, hy - u * 0.02, 0.7, 0.7);
			break;
		}
		case 'insect':
		case 'arachnid':
		case 'crab': {
			const cy = -v * 0.16 * s;
			const rx = u * 0.26 * s * sx;
			const ry = v * 0.1 * s * sy;
			const pairs = plan === 'arachnid' ? 4 : 3;
			c.strokeStyle = dark;
			for (let i = 0; i < pairs; i++) {
				const th = i & 1 ? p.legB : p.legA;
				const k = (i - (pairs - 1) / 2) * 0.35;
				for (const sd of [-1, 1]) {
					const bx = sd * rx * (0.5 + 0.1 * i * 0 + 0.2);
					const kx = sd * (rx + u * 0.12) + k * u * 0.1;
					const fx = sd * (rx + u * 0.24) + Math.sin(th * sd * (i & 1 ? -1 : 1)) * u * 0.12 + k * u * 0.1;
					c.beginPath();
					c.moveTo(bx, cy);
					c.lineTo(kx, cy - v * 0.06);
					c.lineTo(fx, -v * 0.01);
					c.lineWidth = 1;
					c.stroke();
				}
			}
			c.fillStyle = body;
			el(c, 0, cy, rx, ry);
			el(c, rx * 0.8 + lunge, cy - ry * 0.3, u * 0.09 * s, u * 0.08 * s);
			if (plan === 'crab') {
				const up = 0.2 + p.wing;
				for (const sd of [-1, 1]) el(c, sd * (rx + u * 0.06) + (sd > 0 ? lunge : 0), cy - ry * 0.5 - up * v * 0.2, u * 0.08 * s, u * 0.06 * s);
			}
			c.fillStyle = eye;
			el(c, rx * 0.9 + lunge, cy - ry * 0.5, 0.7, 0.7);
			break;
		}
		case 'fish': {
			const cy = -v * 0.4 * s;
			const rx = u * 0.3 * s * sx;
			const ry = v * 0.14 * s * sy;
			const wag = Math.sin(p.tail) * v * 0.18 * s + Math.sin(p.undPhase) * p.undAmp * v * 0.05;
			c.fillStyle = belly;
			pg(c, [-rx * 0.9, cy, -rx - u * 0.22 * s, cy - v * 0.1 + wag, -rx - u * 0.22 * s, cy + v * 0.1 + wag]);
			c.fillStyle = body;
			el(c, lunge * 0.5, cy, rx, ry);
			pg(c, [-u * 0.05, cy - ry * 0.8, u * 0.08, cy - ry * 0.8, 0, cy - ry * 1.5]);
			c.fillStyle = eye;
			el(c, rx * 0.6, cy - ry * 0.2, 0.8, 0.8);
			break;
		}
		case 'ooze': {
			const rx = u * 0.34 * s * sx;
			const ry = v * 0.3 * s * sy;
			c.globalAlpha *= 0.9;
			el(c, 0, -ry, rx, ry);
			c.globalAlpha /= 0.9;
			c.fillStyle = belly;
			el(c, -rx * 0.3, -ry * 1.3, rx * 0.25, ry * 0.2);
			c.fillStyle = eye;
			el(c, -rx * 0.25 + lunge, -ry * 0.9, 0.9, 0.9);
			el(c, rx * 0.25 + lunge, -ry * 0.9, 0.9, 0.9);
			break;
		}
		case 'tentacled': {
			const cy = -v * 0.5 * s;
			c.strokeStyle = body;
			for (let i = 0; i < 5; i++) {
				const bx = (i - 2) * u * 0.1 * s;
				let px = bx;
				let py = cy + v * 0.05;
				c.beginPath();
				c.moveTo(px, py);
				for (let j = 1; j <= 4; j++) {
					px = bx + Math.sin(p.undPhase + i * 0.8 + j * 0.7) * p.sway * u * 0.3 * j * 0.5;
					py = cy + v * 0.05 + j * (-cy - v * 0.05) * 0.25;
					c.lineTo(px, py);
				}
				c.lineWidth = lw * 0.8;
				c.stroke();
			}
			c.fillStyle = body;
			el(c, lunge * 0.5, cy, u * 0.26 * s, v * 0.17 * s);
			c.fillStyle = eye;
			el(c, -u * 0.08 + lunge * 0.5, cy, 0.9, 0.9);
			el(c, u * 0.08 + lunge * 0.5, cy, 0.9, 0.9);
			break;
		}
		case 'golem': {
			const tw = u * 0.22 * s;
			const hipH = v * 0.3 * s;
			c.fillStyle = dark;
			for (const [k, th, lift] of [[-1, p.legA, p.stomp], [1, p.legB, 0]] as const) {
				const fy = -lift * v * 0.05;
				c.strokeStyle = dark;
				ln(c, k * tw * 0.5, -hipH, k * tw * 0.5 + Math.sin(th) * hipH * 0.5, fy, lw * 1.8);
			}
			c.fillStyle = body;
			c.fillRect(-tw, -hipH - v * 0.34 * s, tw * 2, v * 0.34 * s);
			c.fillStyle = belly;
			el(c, 0, -hipH - v * 0.5 * s, tw * 0.6, tw * 0.55);
			c.strokeStyle = body;
			const shY = -hipH - v * 0.3 * s;
			for (const [k, th] of [[-1, -p.legB], [1, -p.legA]] as const) {
				const hx = k * (tw + u * 0.04) + Math.sin(th) * v * 0.25 * s;
				ln(c, k * tw, shY, hx + (k > 0 ? lunge : 0), shY + Math.cos(th) * v * 0.25 * s, lw * 1.8);
			}
			c.fillStyle = eye === '#101818' ? '#ffcc55' : eye;
			el(c, -tw * 0.25, -hipH - v * 0.5 * s, 0.8, 0.8);
			el(c, tw * 0.25, -hipH - v * 0.5 * s, 0.8, 0.8);
			if (p.stomp > 0.8) {
				c.fillStyle = belly;
				c.globalAlpha *= 0.5;
				el(c, -tw, 0, u * 0.1, v * 0.02);
				c.globalAlpha /= 0.5;
			}
			break;
		}
		default: {
			// quadruped, saurian and anything else on four legs
			const sau = plan === 'saurian';
			const cy = -v * (sau ? 0.24 : 0.3) * s;
			const rx = u * 0.34 * s * sx;
			const ry = v * (sau ? 0.11 : 0.15) * s * sy;
			const ll = v * (sau ? 0.14 : 0.2) * s;
			c.strokeStyle = dark;
			for (const [px, th] of [[rx * 0.65, p.legA], [rx * 0.25, p.legB], [-rx * 0.25, p.legA], [-rx * 0.65, p.legB]] as const) {
				ln(c, px, cy + ry * 0.4, px + Math.sin(th) * ll, cy + ry * 0.4 + Math.cos(th) * ll, lw);
			}
			c.strokeStyle = body;
			c.beginPath();
			c.moveTo(-rx * 0.9, cy - ry * 0.2);
			c.quadraticCurveTo(-rx - u * 0.1, cy - ry + Math.sin(p.tail) * v * 0.1, -rx - u * (sau ? 0.28 : 0.16), cy - ry * (sau ? 0.2 : 1.2) + Math.sin(p.tail) * v * 0.2);
			c.lineWidth = lw * (sau ? 1.3 : 0.8);
			c.stroke();
			c.fillStyle = body;
			el(c, 0, cy, rx, ry);
			c.fillStyle = belly;
			el(c, 0, cy + ry * 0.4, rx * 0.8, ry * 0.4);
			c.fillStyle = body;
			el(c, rx + lunge, cy - ry * 0.6, u * 0.12 * s, v * 0.09 * s);
			c.fillStyle = eye;
			el(c, rx + lunge + u * 0.03, cy - ry * 0.75, 0.8, 0.8);
		}
	}
	if (p.glow > 0) glowAt(c, 0, -v * 0.35 * s, v * 0.5, col.glow ?? '#9fd8ff', p.glow);
	c.restore();
}
