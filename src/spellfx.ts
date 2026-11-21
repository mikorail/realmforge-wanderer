/**
 * Spell and effect particles (shell side, cosmetic), in the style of hitfx.ts: it draws on its own transparent
 * canvas stacked over the map (a second overlay beside the hit canvas).
 *  - `frame.hits` events with an `elem` get a burst at the target cell: fire embers, frost shards, shock arcs, acid
 *    drips, poison clouds, holy light, dark wisps, psychic rings, plain arcane sparks. Ranged hits (the attacker is
 *    more than one cell away, origin from `frame.origins` or `frame.actors`) also get a projectile streak along the
 *    line from the attacker to the target first.
 *  - `frame.statuses` (engine StatusFx) draw small looping auras around the sprite: burning, poisoned, bleeding,
 *    frozen, blessed, cursed, shielded, hasted, slowed, stunned, asleep, plus soft glows for magic items on the floor
 *    (glow-uncommon/rare/legendary with id 0) and wielded rare items (same kinds with the carrier's id).
 *  - reduceMotion: no rAF loop, no particles: a short static glyph-like marker on hits and a static marker per status.
 * One rAF loop that runs only while something is alive (status auras keep it going at about 30 fps). Deterministic:
 * hashes of position and event index, never Math.random.
 * Prefs (the coordinator adds them): `spellFx` (default true).
 */

import type { HitMetrics } from './hitfx';

export interface SpellHit {
	kind: string;
	x: number;
	y: number;
	n?: number;
	attackerId: number;
	targetId: number;
	elem?: string;
	player?: boolean;
}

export interface StatusFx {
	id: number;
	x: number;
	y: number;
	kind: string;
}

export interface SpellFxOptions {
	enabled: boolean;
	reduced: boolean;
}

export interface SpellFxHandle {
	setOptions(o: Partial<SpellFxOptions>): void;
	/**
	 * Feed `frame.hits` once per key frame. `origins` is `frame.origins` (attacker id string -> [x, y]) and `actors`
	 * `frame.actors` (id, x, y); either gives the attacker's cell for projectile streaks.
	 */
	push(
		hits: SpellHit[] | undefined,
		origins?: Record<string, number[]> | undefined,
		actors?: { id: number; x: number; y: number }[] | undefined,
	): void;
	/** feed `frame.statuses` every frame (also an empty or missing list to clear) */
	setStatuses(list: StatusFx[] | undefined): void;
	/** call after a resize or zone change; drops everything in flight when `clear` */
	layout(clear?: boolean): void;
	dispose(): void;
}

type Elem = 'fire' | 'frost' | 'shock' | 'acid' | 'poison' | 'holy' | 'dark' | 'psychic' | 'arcane';

interface Part {
	t0: number;
	/** ms before it starts (projectile travel) */
	delay: number;
	life: number;
	elem: Elem;
	/** target cell */
	x: number;
	y: number;
	/** projectile origin cell, or null for a burst only */
	fx: number | null;
	fy: number | null;
	travel: number;
	seed: number;
}

const COL: Record<Elem, string> = {
	fire: '#ff9a3c',
	frost: '#8fe6ff',
	shock: '#fff06a',
	acid: '#b6ff4a',
	poison: '#b98cff',
	holy: '#fff2b0',
	dark: '#c26bff',
	psychic: '#ff8fe0',
	arcane: '#a9c4ff',
};
const MARK: Record<Elem, string> = { fire: '^', frost: '*', shock: 'z', acid: '%', poison: '~', holy: '+', dark: '@', psychic: 'o', arcane: '*' };

const BURST_MS = 520;
const MAX_PARTS = 40;
const FRAME_MS = 33;
const TAU = Math.PI * 2;

function elemOf(e: string | undefined): Elem | null {
	switch (e) {
		case 'fire':
		case 'frost':
		case 'shock':
		case 'acid':
		case 'poison':
		case 'holy':
		case 'dark':
		case 'psychic':
		case 'arcane':
			return e;
	}
	if (e === 'lightning') return 'shock';
	if (e === 'ice' || e === 'cold') return 'frost';
	if (e === 'mind') return 'psychic';
	if (e === 'necrotic' || e === 'shadow') return 'dark';
	if (e === 'radiant') return 'holy';
	return null;
}

function rnd(seed: number, k: number): number {
	let v = Math.imul((seed | 0) ^ Math.imul(k + 1, 0x9e3779b1), 0x85ebca6b);
	v ^= v >>> 15;
	v = Math.imul(v, 0xc2b2ae35);
	v ^= v >>> 13;
	return (v >>> 0) / 4294967296;
}

export function createSpellFx(canvas: HTMLCanvasElement, getMetrics: () => HitMetrics): SpellFxHandle {
	const ctx = canvas.getContext('2d')!;
	const opt: SpellFxOptions = { enabled: true, reduced: false };
	let parts: Part[] = [];
	let statuses: StatusFx[] = [];
	let raf = 0;
	let last = 0;
	let sizedFor = '';

	function fit(): HitMetrics {
		const m = getMetrics();
		const k = `${m.cw}|${m.chh}|${m.dpr}|${m.cols}|${m.rows}`;
		if (k !== sizedFor) {
			sizedFor = k;
			canvas.width = Math.max(1, Math.round(m.cols * m.cw * m.dpr));
			canvas.height = Math.max(1, Math.round(m.rows * m.chh * m.dpr));
			canvas.style.width = `${m.cols * m.cw}px`;
			canvas.style.height = `${m.rows * m.chh}px`;
		}
		return m;
	}

	function loops(): boolean {
		return opt.enabled && !opt.reduced && statuses.length > 0;
	}
	function alive(): boolean {
		return parts.length > 0 || loops();
	}
	function start() {
		if (!raf && opt.enabled) raf = requestAnimationFrame(frame);
	}
	function clearAll() {
		const m = getMetrics();
		ctx.setTransform(1, 0, 0, 1, 0, 0);
		ctx.clearRect(0, 0, canvas.width || m.cols * m.cw, canvas.height || m.rows * m.chh);
	}

	// ---- element bursts ------------------------------------------------------------------------

	function burst(p: Part, k: number, cw: number, chh: number) {
		const cx = (p.x + 0.5) * cw, cy = (p.y + 0.5) * chh;
		const col = COL[p.elem];
		const fade = 1 - k;
		const px = Math.max(1, cw * 0.14);
		ctx.fillStyle = col;
		ctx.strokeStyle = col;
		switch (p.elem) {
			case 'fire':
				for (let i = 0; i < 9; i++) {
					const a = rnd(p.seed, i) * TAU, d = (0.15 + rnd(p.seed, i + 20) * 0.45) * cw * (0.4 + k);
					const x = cx + Math.cos(a) * d * 0.8, y = cy + Math.sin(a) * d * 0.4 - k * chh * (0.25 + rnd(p.seed, i + 40) * 0.5);
					ctx.globalAlpha = Math.max(0, fade * 0.95);
					ctx.fillStyle = i % 3 === 0 ? '#ffe070' : i % 3 === 1 ? '#ff7a2a' : '#ff4a1a';
					ctx.fillRect(x, y, px, px);
				}
				ctx.globalAlpha = Math.max(0, fade * 0.35);
				ctx.fillStyle = '#ff7a2a';
				ctx.beginPath();
				ctx.arc(cx, cy, cw * (0.3 + 0.5 * k), 0, TAU);
				ctx.fill();
				break;
			case 'frost':
				for (let i = 0; i < 7; i++) {
					const a = (i / 7) * TAU + rnd(p.seed, i) * 0.6, d = cw * (0.2 + 0.9 * k) * (0.7 + rnd(p.seed, i + 9) * 0.5);
					const x = cx + Math.cos(a) * d, y = cy + Math.sin(a) * d * 0.8;
					ctx.globalAlpha = Math.max(0, fade);
					ctx.fillStyle = i % 2 ? '#e8faff' : col;
					ctx.beginPath();
					ctx.moveTo(x + Math.cos(a) * cw * 0.28, y + Math.sin(a) * cw * 0.28);
					ctx.lineTo(x + Math.cos(a + 2.4) * cw * 0.12, y + Math.sin(a + 2.4) * cw * 0.12);
					ctx.lineTo(x + Math.cos(a - 2.4) * cw * 0.12, y + Math.sin(a - 2.4) * cw * 0.12);
					ctx.closePath();
					ctx.fill();
				}
				break;
			case 'shock':
				ctx.globalAlpha = Math.max(0, fade);
				ctx.lineWidth = Math.max(1, cw * 0.14);
				ctx.lineJoin = 'round';
				for (let a = 0; a < 3; a++) {
					const ang = rnd(p.seed, a) * TAU;
					ctx.beginPath();
					ctx.moveTo(cx, cy);
					let x = cx, y = cy;
					for (let s = 1; s <= 4; s++) {
						x += Math.cos(ang + (rnd(p.seed, a * 7 + s) - 0.5) * 1.6) * cw * 0.45 * (0.6 + k);
						y += Math.sin(ang + (rnd(p.seed, a * 7 + s + 30) - 0.5) * 1.6) * chh * 0.28 * (0.6 + k);
						ctx.lineTo(x, y);
					}
					ctx.stroke();
				}
				break;
			case 'acid':
				for (let i = 0; i < 6; i++) {
					const x = cx + (rnd(p.seed, i) - 0.5) * cw * 1.1, y = cy - chh * 0.25 + k * chh * (0.5 + rnd(p.seed, i + 9) * 0.6);
					ctx.globalAlpha = Math.max(0, fade);
					ctx.fillStyle = i % 2 ? '#b6ff4a' : '#7ad22a';
					ctx.beginPath();
					ctx.ellipse(x, y, px * 0.7, px * 1.2, 0, 0, TAU);
					ctx.fill();
				}
				break;
			case 'poison':
				for (let i = 0; i < 5; i++) {
					const a = rnd(p.seed, i) * TAU, d = cw * 0.35 * (0.4 + k);
					ctx.globalAlpha = Math.max(0, fade * 0.45);
					ctx.fillStyle = i % 2 ? '#9a6ee0' : '#6fbf4a';
					ctx.beginPath();
					ctx.arc(cx + Math.cos(a) * d, cy + Math.sin(a) * d * 0.7 - k * chh * 0.2, cw * (0.3 + 0.35 * k), 0, TAU);
					ctx.fill();
				}
				break;
			case 'holy': {
				ctx.globalAlpha = Math.max(0, fade * 0.5);
				const g = ctx.createLinearGradient(0, cy - chh * 1.6, 0, cy + chh * 0.4);
				g.addColorStop(0, 'rgba(255,242,176,0)');
				g.addColorStop(1, 'rgba(255,242,176,0.9)');
				ctx.fillStyle = g;
				ctx.fillRect(cx - cw * 0.35, cy - chh * 1.6, cw * 0.7, chh * 2);
				ctx.globalAlpha = Math.max(0, fade);
				ctx.fillStyle = '#fffbe0';
				for (let i = 0; i < 5; i++) {
					const x = cx + (rnd(p.seed, i) - 0.5) * cw * 1.2, y = cy - k * chh * (0.3 + rnd(p.seed, i + 9) * 0.8);
					ctx.fillRect(x, y, px * 0.8, px * 0.8);
				}
				break;
			}
			case 'dark':
				ctx.globalAlpha = Math.max(0, fade * 0.8);
				ctx.lineWidth = Math.max(1, cw * 0.12);
				for (let i = 0; i < 3; i++) {
					const a0 = rnd(p.seed, i) * TAU;
					ctx.beginPath();
					for (let s = 0; s <= 8; s++) {
						const tt = s / 8, a = a0 + tt * 2.6, r = cw * (0.15 + 0.75 * tt) * (0.5 + k);
						const x = cx + Math.cos(a) * r, y = cy + Math.sin(a) * r * 0.7 - tt * chh * 0.2;
						if (s) ctx.lineTo(x, y);
						else ctx.moveTo(x, y);
					}
					ctx.strokeStyle = i % 2 ? '#c26bff' : '#6a2a9a';
					ctx.stroke();
				}
				break;
			case 'psychic':
				ctx.lineWidth = Math.max(1, cw * 0.1);
				for (let i = 0; i < 3; i++) {
					const kk = Math.min(1, Math.max(0, k * 1.3 - i * 0.18));
					ctx.globalAlpha = Math.max(0, (1 - kk) * 0.9);
					ctx.beginPath();
					ctx.ellipse(cx, cy, cw * (0.2 + 0.9 * kk), chh * (0.12 + 0.5 * kk), 0, 0, TAU);
					ctx.stroke();
				}
				break;
			default:
				// arcane: plain sparks
				for (let i = 0; i < 8; i++) {
					const a = (i / 8) * TAU + rnd(p.seed, i) * 0.5, d = cw * (0.1 + 0.8 * k) * (0.6 + rnd(p.seed, i + 9) * 0.6);
					ctx.globalAlpha = Math.max(0, fade);
					ctx.fillStyle = i % 2 ? '#ffffff' : col;
					ctx.fillRect(cx + Math.cos(a) * d - px / 2, cy + Math.sin(a) * d * 0.8 - px / 2, px, px);
				}
		}
		ctx.globalAlpha = 1;
	}

	function projectile(p: Part, t: number, cw: number, chh: number) {
		if (p.fx == null || p.fy == null) return;
		const k = Math.min(1, (t - p.t0) / p.travel);
		const x0 = (p.fx + 0.5) * cw, y0 = (p.fy + 0.5) * chh;
		const x1 = (p.x + 0.5) * cw, y1 = (p.y + 0.5) * chh;
		const hx = x0 + (x1 - x0) * k, hy = y0 + (y1 - y0) * k;
		const tk = Math.max(0, k - 0.35);
		const tx = x0 + (x1 - x0) * tk, ty = y0 + (y1 - y0) * tk;
		const col = COL[p.elem];
		ctx.lineCap = 'round';
		ctx.strokeStyle = col;
		ctx.globalAlpha = 0.35;
		ctx.lineWidth = Math.max(2, cw * 0.38);
		ctx.beginPath();
		ctx.moveTo(tx, ty);
		ctx.lineTo(hx, hy);
		ctx.stroke();
		ctx.globalAlpha = 0.95;
		ctx.lineWidth = Math.max(1, cw * 0.16);
		ctx.beginPath();
		ctx.moveTo(tx, ty);
		ctx.lineTo(hx, hy);
		ctx.stroke();
		ctx.fillStyle = '#ffffff';
		ctx.beginPath();
		ctx.arc(hx, hy, Math.max(1.2, cw * 0.17), 0, TAU);
		ctx.fill();
		// a few trailing sparks, jittered by hash
		ctx.fillStyle = col;
		for (let i = 0; i < 4; i++) {
			const s = tk + (k - tk) * rnd(p.seed, i);
			ctx.fillRect(x0 + (x1 - x0) * s + (rnd(p.seed, i + 9) - 0.5) * cw * 0.5, y0 + (y1 - y0) * s + (rnd(p.seed, i + 19) - 0.5) * chh * 0.3, 1.5, 1.5);
		}
		ctx.globalAlpha = 1;
	}

	// ---- status auras --------------------------------------------------------------------------

	const GLOW: Record<string, [string, number]> = {
		'glow-uncommon': ['110,230,120', 0.5],
		'glow-rare': ['110,170,255', 0.7],
		'glow-legendary': ['255,210,90', 0.95],
	};
	const STATIC_MARK: Record<string, [string, string]> = {
		burning: ['^', '#ff8a2a'],
		poisoned: ['%', '#9be05a'],
		bleeding: ['\'', '#d0283a'],
		frozen: ['*', '#8fe6ff'],
		blessed: ['+', '#fff2b0'],
		cursed: ['-', '#c26bff'],
		shielded: ['o', '#8fb8ff'],
		hasted: ['>', '#ffe070'],
		slowed: ['v', '#9ab0c0'],
		stunned: ['?', '#ffe070'],
		asleep: ['z', '#c8d8ff'],
	};

	/** corner index per kind so several statuses on one actor do not overlap in static mode */
	function drawStatus(s: StatusFx, t: number, cw: number, chh: number, idx: number) {
		const cx = (s.x + 0.5) * cw, cy = (s.y + 0.5) * chh;
		const ph = (s.x * 7 + s.y * 13 + s.id) % 11;
		const tt = t / 1000 + ph * 0.37;
		const px = Math.max(1, cw * 0.13);
		const g = GLOW[s.kind];
		if (g) {
			const wield = s.id !== 0;
			const pulse = 0.7 + 0.3 * Math.sin(tt * 2.4);
			const r = cw * (wield ? 0.55 : 0.9);
			const ox = wield ? cw * 0.38 : 0, oy = wield ? chh * 0.08 : 0;
			for (let i = 3; i >= 1; i--) {
				ctx.globalAlpha = (g[1] * 0.16 * pulse) / (i * 0.8);
				ctx.fillStyle = `rgb(${g[0]})`;
				ctx.beginPath();
				ctx.arc(cx + ox, cy + oy, r * (0.5 + i * 0.3), 0, TAU);
				ctx.fill();
			}
			ctx.globalAlpha = 0.9 * pulse;
			ctx.fillStyle = '#ffffff';
			const a = tt * 1.3;
			ctx.fillRect(cx + ox + Math.cos(a) * r * 0.7 - px / 2, cy + oy + Math.sin(a) * r * 0.5 - px / 2, px, px);
			ctx.globalAlpha = 1;
			return;
		}
		ctx.lineWidth = Math.max(1, cw * 0.1);
		switch (s.kind) {
			case 'burning':
				for (let i = 0; i < 3; i++) {
					const k = (tt * 1.6 + i / 3) % 1;
					ctx.globalAlpha = 1 - k;
					ctx.fillStyle = i === 1 ? '#ffe070' : '#ff6a1a';
					ctx.fillRect(cx + (i - 1) * cw * 0.28 + Math.sin(tt * 5 + i) * px, cy + chh * 0.2 - k * chh * 0.75, px * 1.2, px * 1.8);
				}
				break;
			case 'poisoned':
				for (let i = 0; i < 3; i++) {
					const k = (tt * 0.9 + i / 3) % 1;
					ctx.globalAlpha = 0.9 * (1 - k);
					ctx.strokeStyle = '#8fdc4a';
					ctx.beginPath();
					ctx.arc(cx + (i - 1) * cw * 0.3, cy + chh * 0.1 - k * chh * 0.6, px * (0.6 + k * 0.5), 0, TAU);
					ctx.stroke();
				}
				break;
			case 'bleeding':
				for (let i = 0; i < 2; i++) {
					const k = (tt * 1.2 + i / 2) % 1;
					ctx.globalAlpha = 1 - k * 0.6;
					ctx.fillStyle = '#c0242c';
					ctx.fillRect(cx + (i ? 0.2 : -0.25) * cw, cy - chh * 0.05 + k * chh * 0.5, px * 0.9, px * 1.6);
				}
				break;
			case 'frozen':
				ctx.globalAlpha = 0.55;
				ctx.strokeStyle = '#bfefff';
				ctx.strokeRect(cx - cw * 0.45, cy - chh * 0.45, cw * 0.9, chh * 0.9);
				ctx.fillStyle = 'rgba(160,225,255,0.22)';
				ctx.fillRect(cx - cw * 0.45, cy - chh * 0.45, cw * 0.9, chh * 0.9);
				ctx.globalAlpha = 0.5 + 0.5 * Math.sin(tt * 4);
				ctx.fillStyle = '#ffffff';
				ctx.fillRect(cx + cw * 0.2, cy - chh * 0.3, px, px);
				break;
			case 'blessed':
				ctx.globalAlpha = 0.65 + 0.3 * Math.sin(tt * 2.2);
				ctx.strokeStyle = '#ffe48a';
				ctx.beginPath();
				ctx.ellipse(cx, cy - chh * 0.46, cw * 0.32, chh * 0.07, 0, 0, TAU);
				ctx.stroke();
				ctx.fillStyle = '#fffbe0';
				ctx.fillRect(cx + Math.cos(tt * 2) * cw * 0.4, cy - chh * 0.1 - ((tt * 0.8) % 1) * chh * 0.3, px * 0.8, px * 0.8);
				break;
			case 'cursed':
				for (let i = 0; i < 3; i++) {
					const a = tt * 1.5 + (i / 3) * TAU;
					ctx.globalAlpha = 0.85;
					ctx.fillStyle = i % 2 ? '#c26bff' : '#5a2a7a';
					ctx.fillRect(cx + Math.cos(a) * cw * 0.5, cy + Math.sin(a) * chh * 0.35, px, px);
				}
				break;
			case 'shielded':
				ctx.globalAlpha = 0.35 + 0.2 * Math.sin(tt * 3);
				ctx.strokeStyle = '#8fb8ff';
				ctx.lineWidth = Math.max(1, cw * 0.12);
				ctx.beginPath();
				ctx.ellipse(cx, cy, cw * 0.62, chh * 0.5, 0, 0, TAU);
				ctx.stroke();
				ctx.fillStyle = 'rgba(140,184,255,0.12)';
				ctx.fill();
				break;
			case 'hasted':
				for (let i = 0; i < 3; i++) {
					const k = (tt * 2 + i / 3) % 1;
					ctx.globalAlpha = 0.8 * (1 - k);
					ctx.fillStyle = '#ffe070';
					ctx.fillRect(cx + cw * 0.3 - k * cw * 0.9, cy - chh * 0.2 + i * chh * 0.2, cw * 0.35, Math.max(1, px * 0.6));
				}
				break;
			case 'slowed':
				for (let i = 0; i < 2; i++) {
					const k = (tt * 0.6 + i / 2) % 1;
					ctx.globalAlpha = 0.8 * (1 - k);
					ctx.strokeStyle = '#9ab0c0';
					ctx.beginPath();
					ctx.moveTo(cx - cw * 0.2, cy - chh * 0.3 + k * chh * 0.5);
					ctx.lineTo(cx, cy - chh * 0.15 + k * chh * 0.5);
					ctx.lineTo(cx + cw * 0.2, cy - chh * 0.3 + k * chh * 0.5);
					ctx.stroke();
				}
				break;
			case 'stunned':
				for (let i = 0; i < 3; i++) {
					const a = tt * 2.4 + (i / 3) * TAU;
					ctx.globalAlpha = 0.95;
					ctx.fillStyle = '#ffe070';
					const x = cx + Math.cos(a) * cw * 0.38, y = cy - chh * 0.45 + Math.sin(a) * chh * 0.06;
					ctx.fillRect(x - px * 0.5, y - px * 0.5, px, px);
					ctx.fillRect(x - px * 1.2, y - px * 0.2, px * 2.4, px * 0.4);
				}
				break;
			case 'asleep': {
				const k = (tt * 0.5 + idx * 0.3) % 1;
				ctx.globalAlpha = 1 - k;
				ctx.fillStyle = '#c8d8ff';
				ctx.font = `bold ${Math.round(chh * (0.3 + 0.2 * k))}px "Share Tech Mono", monospace`;
				ctx.textAlign = 'center';
				ctx.fillText('z', cx + cw * 0.3 + k * cw * 0.3, cy - chh * 0.2 - k * chh * 0.5);
				break;
			}
		}
		ctx.globalAlpha = 1;
	}

	/** reduced motion: a static glyph-like marker in a corner of the cell */
	function drawStatic(list: StatusFx[], cw: number, chh: number) {
		const per = new Map<number, number>();
		ctx.textAlign = 'center';
		ctx.textBaseline = 'middle';
		ctx.font = `bold ${Math.max(6, Math.round(chh * 0.42))}px "Share Tech Mono", monospace`;
		for (const s of list) {
			const ck = s.y * 4096 + s.x;
			const i = per.get(ck) ?? 0;
			per.set(ck, i + 1);
			const x = s.x * cw + (i % 2 ? cw * 0.78 : cw * 0.22), y = s.y * chh + (i < 2 ? chh * 0.2 : chh * 0.8);
			const g = GLOW[s.kind];
			if (g) {
				ctx.fillStyle = `rgb(${g[0]})`;
				ctx.beginPath();
				ctx.arc(x, y, Math.max(1.5, cw * 0.14), 0, TAU);
				ctx.fill();
				continue;
			}
			const m = STATIC_MARK[s.kind];
			if (!m) continue;
			ctx.lineWidth = 2;
			ctx.strokeStyle = 'rgba(0,0,0,0.8)';
			ctx.strokeText(m[0], x, y);
			ctx.fillStyle = m[1];
			ctx.fillText(m[0], x, y);
		}
	}

	// ---- frame ---------------------------------------------------------------------------------

	function render(t: number) {
		const m = fit();
		const { cw, chh, dpr } = m;
		ctx.setTransform(dpr, 0, 0, dpr, 0, 0);
		ctx.clearRect(0, 0, m.cols * cw, m.rows * chh);
		if (!opt.enabled) return;
		if (opt.reduced) {
			parts = parts.filter((p) => t - p.t0 < 500);
			ctx.textAlign = 'center';
			ctx.textBaseline = 'middle';
			ctx.font = `bold ${Math.round(chh * 0.7)}px "Share Tech Mono", monospace`;
			for (const p of parts) {
				ctx.lineWidth = 3;
				ctx.strokeStyle = 'rgba(0,0,0,0.8)';
				const x = (p.x + 0.5) * cw, y = (p.y + 0.5) * chh;
				ctx.strokeText(MARK[p.elem], x, y);
				ctx.fillStyle = COL[p.elem];
				ctx.fillText(MARK[p.elem], x, y);
			}
			drawStatic(statuses, cw, chh);
			return;
		}
		// glows first (under the effects), then auras, then bursts
		let idx = 0;
		for (const s of statuses) if (GLOW[s.kind]) drawStatus(s, t, cw, chh, idx++);
		for (const s of statuses) if (!GLOW[s.kind]) drawStatus(s, t, cw, chh, idx++);
		parts = parts.filter((p) => t - p.t0 - p.delay < p.life);
		for (const p of parts) {
			const e = t - p.t0;
			if (e < 0) continue;
			if (e < p.delay) projectile(p, t, cw, chh);
			else burst(p, (e - p.delay) / p.life, cw, chh);
		}
		ctx.globalAlpha = 1;
	}

	function frame(t: number) {
		raf = 0;
		if (t - last < FRAME_MS && parts.length === 0) {
			// status-only: about 30 fps
			if (alive()) start();
			return;
		}
		last = t;
		render(performance.now());
		if (opt.reduced) {
			// static markers only need a repaint on change; keep going just while hits are fading
			if (parts.length) start();
			return;
		}
		if (alive()) start();
	}

	return {
		setOptions(o) {
			Object.assign(opt, o);
			if (!opt.enabled) {
				parts = [];
				if (raf) cancelAnimationFrame(raf);
				raf = 0;
				clearAll();
				return;
			}
			if (opt.reduced) parts = parts.map((p) => ({ ...p, delay: 0 }));
			render(performance.now());
			start();
		},
		push(hits, origins, actors) {
			if (!opt.enabled || !hits || !hits.length) return;
			const t = performance.now();
			const pos = new Map<number, [number, number]>();
			if (actors) for (const a of actors) pos.set(a.id, [a.x, a.y]);
			// origins (positions at the start of the key) win over where actors stand now
			if (origins) for (const k of Object.keys(origins)) pos.set(Number(k), [origins[k][0], origins[k][1]]);
			let n = 0;
			for (let i = 0; i < hits.length && parts.length < MAX_PARTS; i++) {
				const e = hits[i];
				if (e.kind === 'miss' || e.kind === 'block' || e.kind === 'parry' || e.kind === 'heal' || e.kind === 'death') {
					if (!(e.kind === 'heal' && elemOf(e.elem) === 'holy')) continue;
				}
				const o = e.attackerId ? pos.get(e.attackerId) : undefined;
				const dist = o ? Math.max(Math.abs(o[0] - e.x), Math.abs(o[1] - e.y)) : 0;
				let el = elemOf(e.elem);
				if (e.kind === 'burn') el = 'fire';
				else if (e.kind === 'poison') el = 'poison';
				const ranged = !!o && dist > 1;
				if (!el) {
					if (!ranged) continue; // a plain melee blow has no spell effect
					el = 'arcane';
				}
				const seed = (e.x * 31 + e.y * 17 + e.targetId * 7 + i * 13) | 0;
				const travel = ranged && !opt.reduced ? Math.min(320, 90 + dist * 28) : 0;
				parts.push({
					t0: t + n * 20,
					delay: opt.reduced ? 0 : travel,
					life: opt.reduced ? 500 : BURST_MS,
					elem: el,
					x: e.x,
					y: e.y,
					fx: ranged && o ? o[0] : null,
					fy: ranged && o ? o[1] : null,
					travel: Math.max(1, travel),
					seed,
				});
				n++;
			}
			if (n) {
				if (opt.reduced) render(performance.now());
				start();
			}
		},
		setStatuses(list) {
			const next = list ? list.slice(0, 80) : [];
			const same =
				next.length === statuses.length && next.every((s, i) => s.x === statuses[i].x && s.y === statuses[i].y && s.kind === statuses[i].kind && s.id === statuses[i].id);
			statuses = next;
			if (!opt.enabled) return;
			if (opt.reduced) {
				if (!same || !parts.length) render(performance.now());
				return;
			}
			if (statuses.length) start();
			else if (!parts.length) clearAll();
		},
		layout(clear) {
			sizedFor = '';
			if (clear) {
				parts = [];
				statuses = [];
			}
			render(performance.now());
			start();
		},
		dispose() {
			if (raf) cancelAnimationFrame(raf);
			raf = 0;
			parts = [];
			statuses = [];
		},
	};
}
