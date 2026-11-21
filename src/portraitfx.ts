/**
 * Portrait faces (shell side, cosmetic): eyes, brows, mouth, cheeks, wrinkles by age, wounds, scars, markings and a
 * few accessories drawn over the head of the sprite that `drawActorSprite` paints in a portrait. The engine says
 * what the face shows (`frame.portrait.mood`, engine/portraitmood.go `PortraitMoodJS`); this file only draws it.
 *
 * Use: call `drawFace` right after `drawActorSprite`, with the SAME x, y, w, h box the sprite got. It returns true
 * while it wants another frame (a blink, an expression change): redraw the portrait about every 60-100 ms then.
 * Idle motion (blink every few seconds, a faint breath) stops under reduceMotion; the art stays. Deterministic:
 * hashes of the actor id and the face, never Math.random. Simple shapes only.
 */

export interface PortraitFace {
	mood: string; // neutral pleased warm amused wary angry afraid sad grieving proud scornful tired ill drunk sleepy dying
	intensity: number; // 0..3
	age: string; // child young prime mature old venerable
	wounds: number; // 0..3
	gaze: string; // toward | away
	scar?: string; // brow | cheek | across | eye
	marking?: string; // tattoo | warpaint | totem
	markColor?: string;
	accessory?: string; // hat circlet hood goggles veil eyepatch pipe
}

export interface FaceColors {
	skin?: string; // default: sampled from the pixels the sprite just drew
	eye?: string; // default by species
	ink?: string; // default '#1a1410'
}

export interface HeadBox {
	cx: number;
	cy: number; // centre of the head ellipse
	hw: number; // half width
	hh: number; // half height
	eyeY: number; // centre of the eyes row
	eyeDx: number; // eye centre offset from cx
	es: number; // eye size the sprite uses (px)
}

export interface FaceOpts {
	id?: number; // actor id (head size varies per individual in actorsprites.ts)
	headBox?: HeadBox; // override of headBoxFor
	now?: number; // ms, default performance.now()
	reduceMotion?: boolean;
	headgear?: boolean; // also draw hat/circlet/hood (the sprite already draws worn head items: default false)
	key?: string; // identity of the portrait for the expression transition (default species:id)
}

// ---------------------------------------------------------------- head geometry (mirrors actorsprites.ts drawPersonSprite)

type HeadKind = 'round' | 'long' | 'wide' | 'skull';
const SP: Record<string, [number, number, HeadKind]> = {
	// height, head size, head shape
	human: [0.92, 1, 'round'], dwarf: [0.74, 1.05, 'wide'], elf: [1, 0.95, 'long'], void_elf: [1, 0.95, 'long'],
	halfling: [0.6, 1.1, 'round'], orc: [1, 1, 'wide'], goblin: [0.62, 1.2, 'long'], ogor: [1, 0.9, 'wide'],
	troll: [1, 0.95, 'long'], giant: [1, 0.8, 'wide'], ratfolk: [0.74, 0.95, 'long'], beastman: [0.96, 1, 'long'],
	lizardfolk: [0.9, 0.95, 'long'], saurian: [1, 1, 'long'], merfolk: [0.9, 1, 'round'], celestial: [1, 0.95, 'round'],
	tomb_undead: [0.95, 1, 'skull'], warp_fiend: [1, 1, 'wide'], chitin_swarm: [0.8, 1, 'round'],
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

/**
 * Where the head is for a person sprite drawn into the box (x, y, w, h) (the same box given to drawActorSprite).
 * Recomputes the geometry of drawPersonSprite; if actorsprites.ts changes its head proportions, update SP above or
 * pass an explicit `headBox` to drawFace.
 */
export function headBoxFor(species: string, id: number, x: number, y: number, w: number, h: number): HeadBox {
	let ht: number, hd: number, kind: HeadKind;
	const k = SP[species];
	if (k) [ht, hd, kind] = k;
	else {
		const hs = strHash(species);
		ht = 0.7 + (hs % 30) / 100;
		hd = 0.9 + ((hs >>> 9) % 25) / 100;
		kind = (['round', 'long', 'wide'] as HeadKind[])[(hs >>> 11) % 3];
	}
	const seed = hash(id + 17);
	const vary = 1 + (((seed >>> 20) % 9) - 4) * 0.015;
	const base = y + h * 0.9;
	const H = h * 0.82 * ht * vary;
	const hr = w * 0.2 * hd * (1 + (ht < 0.75 ? 0.15 : 0));
	const cy = base - H + hr;
	const hw = hr * (kind === 'wide' ? 1.2 : kind === 'long' ? 0.9 : 1);
	const hh = hr * (kind === 'long' ? 1.2 : kind === 'wide' ? 0.95 : 1);
	const es = Math.max(1, w * 0.07);
	return { cx: x + w * 0.5, cy, hw, hh, eyeY: cy - hh * 0.1 + es * 0.55, eyeDx: hw * 0.5 - es * 0.5, es };
}

/** Vertical breath offset in px for the whole portrait sprite (0 under reduceMotion). Add it to the sprite's y AND to drawFace's y. */
export function portraitBreath(now: number, h: number, reduceMotion?: boolean): number {
	return reduceMotion ? 0 : Math.sin(now / 1500) * h * 0.004;
}

// ---------------------------------------------------------------- species categories

type Cat = 'plain' | 'beast' | 'scaled' | 'glow' | 'machine' | 'orc' | 'bone';
function catOf(sp: string): Cat {
	const s = sp.toLowerCase();
	if (/robot|construct|golem|automaton|android|machine|drone|cyborg|clockwork|mech/.test(s)) return 'machine';
	if (/lizard|saurian|dragon|drake|naga|kobold|serpent/.test(s)) return 'scaled';
	if (/rat|beast|wolf|cat|fox|bear|were|fur|kin$/.test(s) && !/human|dwarf|elf/.test(s)) return 'beast';
	if (/star|celest|void|spirit|ghost|wisp|angel|fey|fae|lumin/.test(s)) return 'glow';
	if (/orc|ogor|troll|ogre|goblin|gnoll/.test(s)) return 'orc';
	if (/undead|skel|tomb|lich|bone/.test(s)) return 'bone';
	return 'plain';
}
const GLOW: Record<string, string> = { celestial: '#fff0a0', void_elf: '#e8a8ff', warp_fiend: '#d83aff', tomb_undead: '#58ff9a' };

// ---------------------------------------------------------------- expression parameters

interface Ex {
	tilt: number; // brows: + inner ends down (angry), - inner ends up (sad, afraid)
	lift: number; // brows raised, -1..1
	asym: number; // one brow higher (scorn, doubt)
	lid: number; // 0 open .. 1 shut
	eye: number; // eye size multiplier
	curve: number; // mouth: + smile, - frown
	open: number; // mouth opening 0..1
	wide: number; // mouth width multiplier
	cheek: number; // blush 0..1
	pale: number; // wash 0..1
	green: number;
	gx: number; // gaze -1..1
	gy: number;
	tear: number;
	sweat: number;
	bags: number; // under-eye shadows
}
const ZERO: Ex = { tilt: 0, lift: 0, asym: 0, lid: 0, eye: 1, curve: 0, open: 0, wide: 1, cheek: 0, pale: 0, green: 0, gx: 0, gy: 0, tear: 0, sweat: 0, bags: 0 };
type Partial_ = Partial<Ex>;
// the value each field reaches at intensity 3 (scaled down for lower intensity)
const MOODS: Record<string, Partial_> = {
	neutral: {},
	pleased: { curve: 0.55, lift: 0.15, cheek: 0.2, lid: 0.08 },
	warm: { curve: 0.85, lift: 0.2, cheek: 0.6, lid: 0.18, wide: 1.15 },
	amused: { curve: 0.7, lift: 0.1, asym: 0.5, lid: 0.15, open: 0.15, cheek: 0.15 },
	wary: { tilt: 0.35, lid: 0.22, curve: -0.12, asym: 0.2, gx: 0.3 },
	angry: { tilt: 0.95, lift: -0.3, lid: 0.3, curve: -0.6, open: 0.2, eye: 0.92, cheek: 0.35 },
	afraid: { tilt: -0.7, lift: 0.8, eye: 1.3, curve: -0.2, open: 0.45, pale: 0.25, sweat: 1 },
	sad: { tilt: -0.75, lift: -0.05, lid: 0.25, curve: -0.55, gy: 0.5, tear: 0.25 },
	grieving: { tilt: -0.9, lift: 0.05, lid: 0.35, curve: -0.7, open: 0.15, gy: 0.7, tear: 1, pale: 0.12 },
	proud: { tilt: 0.12, lift: 0.3, lid: 0.28, curve: 0.2, wide: 0.85, gy: -0.3 },
	scornful: { tilt: 0.3, asym: 0.9, lid: 0.35, curve: -0.3, wide: 0.8, gy: -0.2 },
	tired: { lid: 0.45, curve: -0.2, lift: -0.2, bags: 1, gy: 0.3 },
	ill: { lid: 0.3, curve: -0.35, pale: 0.35, green: 0.7, sweat: 0.6, bags: 0.8, tilt: -0.3 },
	drunk: { lid: 0.4, curve: 0.5, open: 0.2, cheek: 0.9, asym: 0.5, gx: 0.3 },
	sleepy: { lid: 0.85, curve: -0.05, bags: 0.7, lift: -0.3 },
	dying: { lid: 0.65, curve: -0.4, open: 0.3, pale: 0.55, bags: 1, tilt: -0.4, gy: 0.4 },
};
function target(f: PortraitFace, sign: number): Ex {
	const m = MOODS[f.mood] ?? MOODS.neutral;
	const k = Math.min(1, 0.3 + 0.25 * Math.max(0, Math.min(3, f.intensity)));
	const e: Ex = { ...ZERO };
	for (const key of Object.keys(m) as (keyof Ex)[]) {
		const v = m[key] as number;
		e[key] = key === 'eye' || key === 'wide' ? 1 + (v - 1) * k : v * k;
	}
	if (f.gaze === 'away') {
		e.gx = sign * 0.85;
		e.gy = Math.max(e.gy, 0.15);
	} else e.gx *= sign;
	if (f.wounds >= 2) e.pale = Math.max(e.pale, 0.15 * f.wounds);
	if (f.wounds >= 3) e.lid = Math.max(e.lid, 0.3);
	return e;
}
const lerp = (a: number, b: number, t: number) => a + (b - a) * t;
function mix(a: Ex, b: Ex, t: number): Ex {
	const o = { ...a };
	for (const k of Object.keys(a) as (keyof Ex)[]) o[k] = lerp(a[k], b[k], t);
	return o;
}

// transition memory per portrait
interface Mem { cur: Ex; from: Ex; to: Ex; sig: string; t0: number }
const mem = new Map<string, Mem>();
const TRANS = 200;
const ease = (t: number) => t * t * (3 - 2 * t);

// ---------------------------------------------------------------- skin sampling

const skinCache = new Map<string, string>();
function sampleSkin(c: CanvasRenderingContext2D, hb: HeadBox, cacheKey: string): string {
	const hit = skinCache.get(cacheKey);
	if (hit) return hit;
	let col = '#d8b090';
	try {
		const m = c.getTransform();
		const px = hb.cx - hb.hw * 0.72, py = hb.cy + hb.hh * 0.3;
		const dx = Math.round(m.a * px + m.c * py + m.e), dy = Math.round(m.b * px + m.d * py + m.f);
		const d = c.getImageData(dx, dy, 1, 1).data;
		if (d[3] > 200) col = `rgb(${d[0]},${d[1]},${d[2]})`;
	} catch {
		/* tainted or unavailable: fallback tone */
	}
	if (skinCache.size > 400) skinCache.clear();
	skinCache.set(cacheKey, col);
	return col;
}

// ---------------------------------------------------------------- drawing

function ell(c: CanvasRenderingContext2D, x: number, y: number, rx: number, ry: number) {
	c.beginPath();
	c.ellipse(x, y, Math.max(0.1, rx), Math.max(0.1, ry), 0, 0, Math.PI * 2);
	c.fill();
}
function seg(c: CanvasRenderingContext2D, x1: number, y1: number, x2: number, y2: number, w: number) {
	c.lineWidth = w;
	c.beginPath();
	c.moveTo(x1, y1);
	c.lineTo(x2, y2);
	c.stroke();
}

/**
 * Draw the face over the sprite head. (x, y, w, h) is the box the sprite was drawn into. `species` picks the head
 * geometry and the feature category (ears, tusks and snouts are already in the sprite; this adds whiskers, scales,
 * glow, visor). Returns true while another frame is wanted.
 */
export function drawFace(c: CanvasRenderingContext2D, face: PortraitFace, species: string, x: number, y: number, w: number, h: number, colors: FaceColors = {}, opts: FaceOpts = {}): boolean {
	const id = opts.id ?? 0;
	const hb = opts.headBox ?? headBoxFor(species, id, x, y, w, h);
	const now = opts.now ?? (typeof performance !== 'undefined' ? performance.now() : 0);
	const still = !!opts.reduceMotion;
	const cat = catOf(species);
	const ink = colors.ink ?? '#1a1410';
	const skin = colors.skin ?? sampleSkin(c, hb, `${species}:${id}:${Math.round(w)}x${Math.round(h)}`);
	const glow = GLOW[species] ?? (cat === 'glow' ? '#c8e8ff' : cat === 'machine' ? '#58e0ff' : undefined);
	const eyeCol = colors.eye ?? glow ?? (cat === 'scaled' || cat === 'orc' ? '#e8c040' : '#2a1c12');
	const sign = hash(id, 5) & 1 ? 1 : -1;

	// expression with a ~200 ms transition
	const sig = `${face.mood}:${face.intensity}:${face.gaze}:${face.wounds}`;
	const key = opts.key ?? `${species}:${id}`;
	const tgt = target(face, sign);
	let m = mem.get(key);
	if (!m) {
		m = { cur: tgt, from: tgt, to: tgt, sig, t0: now - TRANS };
		if (mem.size > 40) mem.clear();
		mem.set(key, m);
	}
	if (m.sig !== sig) {
		m.from = still ? tgt : m.cur;
		m.to = tgt;
		m.sig = sig;
		m.t0 = now;
	}
	const tt = still ? 1 : Math.min(1, (now - m.t0) / TRANS);
	m.cur = tt >= 1 ? m.to : mix(m.from, m.to, ease(tt));
	let wantFrame = tt < 1;
	const e = m.cur;

	// idle: a blink every 3.2-6 s (150 ms), a faint mouth breath
	let blink = 0;
	if (!still) {
		const period = 3200 + (hash(id, 9) % 2800);
		const ph = (now + (hash(id, 11) % period)) % period;
		if (ph < 150) blink = Math.sin((ph / 150) * Math.PI);
		wantFrame = true;
	}
	const lid = Math.min(1, e.lid + blink * (1 - e.lid));
	const breath = still ? 0 : Math.sin(now / 1500) * 0.5 + 0.5;

	const { cx, cy, hw, hh } = hb;
	const age = face.age;
	const child = age === 'child';
	const ex = hb.eyeDx, ey = hb.eyeY;
	const rw = Math.max(1.1, hw * 0.24) * e.eye * (child ? 1.25 : 1);
	const rh = Math.max(1, hw * 0.2) * e.eye * (child ? 1.25 : 1);
	const lw = Math.max(0.8, hw * 0.07);

	c.save();
	// keep to the head (and a little beyond for accessories)
	c.lineCap = 'round';

	// 1) erase the sprite's own eye dots, then wash skin tone for illness and wounds
	c.fillStyle = skin;
	for (const s of [-1, 1]) c.fillRect(cx + s * ex - hb.es * 0.9, ey - hb.es * 0.9, hb.es * 1.8, hb.es * 1.8);
	if (e.pale > 0.02 || e.green > 0.02) {
		c.save();
		c.beginPath();
		c.ellipse(cx, cy, hw * 0.98, hh * 0.98, 0, 0, Math.PI * 2);
		c.clip();
		c.fillStyle = `rgba(235,235,225,${e.pale * 0.35})`;
		c.fillRect(cx - hw, cy - hh * 0.1, hw * 2, hh * 1.2);
		if (e.green > 0.02) {
			c.fillStyle = `rgba(110,160,70,${e.green * 0.3})`;
			c.fillRect(cx - hw, cy - hh * 0.1, hw * 2, hh * 1.2);
		}
		c.restore();
	}

	// 2) cheeks
	if (e.cheek > 0.03 && cat !== 'machine' && cat !== 'bone') {
		c.fillStyle = `rgba(230,90,90,${0.35 * e.cheek})`;
		for (const s of [-1, 1]) ell(c, cx + s * hw * 0.62, cy + hh * 0.42, hw * 0.2, hh * 0.14);
	}
	// under-eye shadow
	if (e.bags > 0.05 && cat !== 'machine') {
		c.strokeStyle = `rgba(60,40,70,${0.3 * e.bags})`;
		for (const s of [-1, 1]) {
			c.lineWidth = lw;
			c.beginPath();
			c.arc(cx + s * ex, ey + rh * 0.3, rw * 1.1, 0.2, Math.PI - 0.2);
			c.stroke();
		}
	}

	// 3) eyes
	const gx = e.gx * rw * 0.45, gy = e.gy * rh * 0.4;
	if (cat === 'machine') {
		// a visor line: it narrows with lids and flares with fear/anger
		const vh = Math.max(1.2, hw * 0.2) * (1 - lid * 0.7) * Math.min(1.3, e.eye);
		c.fillStyle = '#12151a';
		c.fillRect(cx - hw * 0.78, ey - vh * 0.9, hw * 1.56, vh * 1.8);
		c.fillStyle = eyeCol;
		const lit = 0.4 + 0.6 * (1 - lid);
		c.globalAlpha = lit;
		c.fillRect(cx - hw * 0.66 + gx, ey - vh * 0.45, hw * 1.32, vh * 0.9);
		c.globalAlpha = 1;
	} else {
		for (const s of [-1, 1]) {
			const exx = cx + s * ex;
			const ly = lid * (s === sign ? 1 : 1 - e.asym * 0.2);
			if (glow) {
				const gr = c.createRadialGradient(exx, ey, 0, exx, ey, rw * 2.6);
				gr.addColorStop(0, glow + '88');
				gr.addColorStop(1, glow + '00');
				c.fillStyle = gr;
				c.fillRect(exx - rw * 2.6, ey - rw * 2.6, rw * 5.2, rw * 5.2);
			}
			// white of the eye (not for glowing or reptile eyes)
			c.fillStyle = glow ? eyeCol : '#f4efe6';
			ell(c, exx, ey, rw, rh * Math.max(0.12, 1 - ly));
			if (!glow && ly < 0.92) {
				c.fillStyle = cat === 'scaled' || cat === 'orc' ? eyeCol : ink;
				const pr = Math.min(rw, rh) * 0.62;
				ell(c, exx + gx, ey + gy, pr, pr * Math.max(0.2, 1 - ly));
				if (cat === 'scaled') {
					c.fillStyle = ink;
					ell(c, exx + gx, ey + gy, pr * 0.25, pr);
				}
			}
			// lid line
			c.strokeStyle = ink;
			c.lineWidth = lw;
			c.beginPath();
			const top = ey - rh * (1 - 2 * ly);
			c.moveTo(exx - rw, ey);
			c.quadraticCurveTo(exx, top - rh * 0.2, exx + rw, ey);
			c.stroke();
			if (face.mood === 'drunk' && ly < 0.9) {
				c.fillStyle = 'rgba(200,60,60,0.35)';
				ell(c, exx, ey + rh * 0.2, rw * 0.9, rh * 0.25);
			}
		}
	}

	// 4) brows
	if (cat !== 'machine') {
		c.strokeStyle = cat === 'bone' ? 'rgba(0,0,0,0)' : ink;
		const bw = rw * 1.5;
		for (const s of [-1, 1]) {
			const by = ey - rh * 1.9 - e.lift * hh * 0.28 - (s === sign ? e.asym * hh * 0.22 : 0);
			const inner = by + e.tilt * hh * 0.22;
			const outer = by - e.tilt * hh * 0.12;
			const bx = cx + s * ex;
			seg(c, bx - s * bw, inner, bx + s * bw, outer, lw * 1.3);
			// inner end is toward the nose: sign flip handled by using -s for inner
		}
	}

	// 5) mouth
	const mw = hw * 0.34 * e.wide;
	const lipSnout = cat === 'beast' ? 0.62 : cat === 'scaled' ? 0.55 : 0.5;
	const my = Math.min(cy + hh * 0.92, cy + hh * lipSnout + breath * 0.3);
	if (cat !== 'bone') {
		c.strokeStyle = cat === 'machine' ? '#12151a' : '#5a2a2a';
		c.fillStyle = cat === 'machine' ? '#12151a' : '#3a1414';
		if (cat === 'machine') {
			for (let i = -1; i <= 1; i++) seg(c, cx - mw * 0.9, my + i * hh * 0.07, cx + mw * 0.9, my + i * hh * 0.07, lw);
		} else {
			const cv = e.curve * hh * 0.28;
			const tilt = face.mood === 'drunk' ? hh * 0.08 : face.mood === 'scornful' ? sign * hh * 0.1 : 0;
			if (e.open > 0.06) {
				ell(c, cx, my + hh * 0.05, mw * (0.35 + e.open * 0.4), hh * 0.08 + e.open * hh * 0.2);
			}
			c.lineWidth = lw * 1.1;
			c.beginPath();
			c.moveTo(cx - mw, my - cv * 0.5 + tilt);
			c.quadraticCurveTo(cx, my + cv * 1.1, cx + mw, my - cv * 0.5 - tilt);
			c.stroke();
		}
	} else {
		c.strokeStyle = 'rgba(30,25,20,0.6)';
		for (let i = -2; i <= 2; i++) seg(c, cx + i * mw * 0.35, my - hh * 0.06, cx + i * mw * 0.35, my + hh * 0.08, lw * 0.8);
	}

	// species flavour
	if (cat === 'beast') {
		c.strokeStyle = 'rgba(240,230,210,0.7)';
		for (const s of [-1, 1]) for (const d of [-0.12, 0.05]) seg(c, cx + s * hw * 0.25, my + hh * d, cx + s * hw * 0.95, my + hh * (d - 0.1), lw * 0.7);
	} else if (cat === 'scaled') {
		c.fillStyle = 'rgba(0,0,0,0.18)';
		for (let i = 0; i < 5; i++) ell(c, cx + (i - 2) * hw * 0.3, cy - hh * 0.55 + (i % 2) * hh * 0.1, hw * 0.1, hw * 0.07);
		for (const s of [-1, 1]) ell(c, cx + s * hw * 0.7, cy + hh * 0.15, hw * 0.09, hw * 0.07);
	} else if (cat === 'orc') {
		c.fillStyle = '#f0e8d0';
		for (const s of [-1, 1]) {
			c.beginPath();
			c.moveTo(cx + s * mw * 0.9, my);
			c.lineTo(cx + s * mw * 1.15, my - hh * 0.2);
			c.lineTo(cx + s * mw * 0.65, my - hh * 0.04);
			c.fill();
		}
	}

	// 6) wrinkles by age
	if (!child && cat !== 'machine' && cat !== 'bone') {
		c.strokeStyle = 'rgba(40,25,15,0.32)';
		c.lineWidth = Math.max(0.6, lw * 0.7);
		const lv = age === 'mature' ? 1 : age === 'old' ? 2 : age === 'venerable' ? 3 : 0;
		if (lv >= 1) for (const dy of [0, 1].slice(0, lv >= 2 ? 2 : 1)) seg(c, cx - hw * 0.4, ey - rh * 3 - dy * hh * 0.12, cx + hw * 0.4, ey - rh * 3 - dy * hh * 0.12, c.lineWidth);
		if (lv >= 2) for (const s of [-1, 1]) {
			seg(c, cx + s * (ex + rw * 1.1), ey - rh * 0.3, cx + s * (ex + rw * 1.9), ey - rh * 0.8, c.lineWidth);
			seg(c, cx + s * (ex + rw * 1.1), ey + rh * 0.1, cx + s * (ex + rw * 1.9), ey + rh * 0.3, c.lineWidth);
		}
		if (lv >= 3) for (const s of [-1, 1]) {
			c.beginPath();
			c.arc(cx + s * hw * 0.55, my - hh * 0.25, hh * 0.35, s > 0 ? 2.2 : 0.4, s > 0 ? 3.2 : 1.4);
			c.stroke();
			seg(c, cx + s * hw * 0.35, my + hh * 0.3, cx + s * hw * 0.35, my + hh * 0.5, c.lineWidth);
		}
	}

	// 7) wounds, scar
	if (face.wounds >= 1) {
		c.fillStyle = 'rgba(110,50,90,0.35)';
		ell(c, cx + sign * hw * 0.6, cy + hh * 0.3, hw * 0.2, hh * 0.15);
	}
	if (face.wounds >= 2) {
		c.strokeStyle = 'rgba(150,20,20,0.85)';
		seg(c, cx - sign * hw * 0.7, cy - hh * 0.55, cx - sign * hw * 0.4, cy - hh * 0.15, lw);
	}
	if (face.wounds >= 3) {
		c.strokeStyle = 'rgba(160,15,15,0.9)';
		seg(c, cx + sign * hw * 0.9, cy + hh * 0.1, cx + sign * hw * 0.85, cy + hh * 0.75, lw * 1.2);
		c.fillStyle = 'rgba(150,15,15,0.8)';
		ell(c, cx + sign * hw * 0.85, cy + hh * 0.8, lw * 0.9, lw * 1.2);
	}
	if (face.scar) {
		c.strokeStyle = 'rgba(235,225,215,0.75)';
		const sw = Math.max(0.9, lw * 1.1);
		const sx = sign * -1;
		if (face.scar === 'brow') seg(c, cx + sx * hw * 0.25, ey - rh * 3.2, cx + sx * hw * 0.8, ey - rh * 1.2, sw);
		else if (face.scar === 'cheek') seg(c, cx + sx * hw * 0.35, ey + rh * 1.3, cx + sx * hw * 0.85, ey + rh * 3.4, sw);
		else if (face.scar === 'across') {
			seg(c, cx + sx * hw * 0.2, ey - rh * 3, cx + sx * hw * 0.95, ey + rh * 3.8, sw);
			c.strokeStyle = 'rgba(120,70,70,0.5)';
			for (let i = 1; i < 4; i++) {
				const t = i / 4;
				const px = cx + sx * hw * (0.2 + 0.75 * t), py = ey - rh * 3 + rh * 6.8 * t;
				seg(c, px - hw * 0.07, py, px + hw * 0.07, py, sw * 0.7);
			}
		} else if (face.scar === 'eye') seg(c, cx + sx * ex, ey - rh * 2.6, cx + sx * ex, ey + rh * 2.8, sw);
	}

	// 8) markings
	if (face.marking) {
		const mc = face.markColor ?? (face.marking === 'tattoo' ? 'rgba(40,70,120,0.7)' : '#d8e8ff');
		c.strokeStyle = mc;
		c.fillStyle = mc;
		const mw2 = Math.max(0.9, lw * 1.1);
		if (face.marking === 'warpaint') {
			for (const s of [-1, 1]) for (let i = 0; i < 2; i++) seg(c, cx + s * hw * 0.35, ey + rh * (1.7 + i * 1.3), cx + s * hw * 0.95, ey + rh * (1.3 + i * 1.3), mw2 * 1.2);
		} else if (face.marking === 'tattoo') {
			for (const s of [-1, 1]) for (let i = 0; i < 3; i++) ell(c, cx + s * (hw * 0.55 + i * hw * 0.12), ey + rh * (2.3 + i * 0.6), mw2 * 0.6, mw2 * 0.6);
			seg(c, cx, cy - hh * 0.9, cx, cy - hh * 0.55, mw2 * 0.8);
		} else {
			// totem: a brow chevron and a chin line
			c.beginPath();
			c.moveTo(cx - hw * 0.35, cy - hh * 0.5);
			c.lineTo(cx, cy - hh * 0.8);
			c.lineTo(cx + hw * 0.35, cy - hh * 0.5);
			c.lineWidth = mw2 * 1.3;
			c.stroke();
			seg(c, cx, my + hh * 0.2, cx, cy + hh * 0.95, mw2);
		}
	}

	// 9) drops
	if (e.tear > 0.08) {
		c.fillStyle = 'rgba(150,200,255,0.85)';
		const ty = ey + rh * 1.2 + e.tear * hh * 0.35 + (still ? 0 : ((now / 700) % 1) * hh * 0.1);
		ell(c, cx - sign * ex, ty, lw * 0.9, lw * 1.4);
		if (e.tear > 0.6) ell(c, cx + sign * ex, ty - hh * 0.05, lw * 0.9, lw * 1.4);
	}
	if (e.sweat > 0.1) {
		c.fillStyle = 'rgba(190,225,255,0.85)';
		ell(c, cx + sign * hw * 0.92, cy - hh * 0.65, lw * 1.1, lw * 1.7);
	}

	// 10) accessories
	const acc = face.accessory;
	if (acc === 'goggles') {
		c.strokeStyle = '#6a5a40';
		c.lineWidth = lw * 1.4;
		seg(c, cx - hw, ey - rh * 0.3, cx + hw, ey - rh * 0.3, lw * 1.4);
		for (const s of [-1, 1]) {
			c.fillStyle = 'rgba(120,200,220,0.35)';
			c.beginPath();
			c.arc(cx + s * ex, ey, rw * 1.5, 0, Math.PI * 2);
			c.fill();
			c.stroke();
		}
	} else if (acc === 'eyepatch') {
		const pxx = cx - sign * ex;
		c.strokeStyle = '#1a1a1e';
		seg(c, cx - hw * 0.95, ey - rh * 2.2, cx + hw * 0.95, ey - rh * 0.5, lw * 0.9);
		c.fillStyle = '#1a1a1e';
		ell(c, pxx, ey, rw * 1.5, rh * 1.5);
	} else if (acc === 'veil') {
		c.fillStyle = 'rgba(230,225,240,0.6)';
		c.beginPath();
		c.moveTo(cx - hw * 0.95, ey + rh * 0.8);
		c.lineTo(cx + hw * 0.95, ey + rh * 0.8);
		c.lineTo(cx + hw * 0.7, cy + hh * 1.05);
		c.lineTo(cx - hw * 0.7, cy + hh * 1.05);
		c.fill();
	} else if (acc === 'pipe') {
		c.strokeStyle = '#4a3020';
		seg(c, cx + hw * 0.25, my + hh * 0.05, cx + hw * 0.95, my + hh * 0.3, lw * 1.3);
		c.fillStyle = '#4a3020';
		ell(c, cx + hw * 1.02, my + hh * 0.2, hw * 0.13, hw * 0.16);
		if (!still) {
			c.fillStyle = 'rgba(220,220,220,0.35)';
			ell(c, cx + hw * 1.05, my - hh * 0.3 - ((now / 900) % 1) * hh * 0.4, hw * 0.1, hw * 0.1);
			wantFrame = true;
		}
	} else if (opts.headgear && (acc === 'hat' || acc === 'circlet' || acc === 'hood')) {
		c.fillStyle = acc === 'circlet' ? '#e8c850' : '#4a3a5a';
		if (acc === 'circlet') c.fillRect(cx - hw * 0.8, cy - hh * 0.75, hw * 1.6, Math.max(1, lw * 1.2));
		else if (acc === 'hat') {
			c.fillRect(cx - hw * 1.3, cy - hh * 0.85, hw * 2.6, Math.max(1, lw * 1.4));
			c.fillRect(cx - hw * 0.8, cy - hh * 1.5, hw * 1.6, hh * 0.7);
		} else {
			c.beginPath();
			c.arc(cx, cy - hh * 0.1, hw * 1.1, Math.PI, 0);
			c.fill();
		}
	}

	c.restore();
	return wantFrame;
}
