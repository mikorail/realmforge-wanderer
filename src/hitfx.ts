/**
 * Combat feel (shell side, cosmetic): floating damage numbers, a white flash on the struck cell, screen shake
 * when the player is hurt, death puffs, optional telegraph tints and a very short hit pause. Fed by
 * `frame.hits` (engine/hitfx.go). Draws on its own transparent canvas stacked over the map; one rAF loop that
 * runs only while something is alive. reduceMotion: numbers stay for 600 ms without moving, no flash, shake,
 * puffs or pause.
 */

export type HitKind = 'hit' | 'crit' | 'miss' | 'block' | 'parry' | 'heal' | 'death' | 'burn' | 'poison' | 'bleed';

export interface HitEvent {
	kind: HitKind;
	/** map cell of the target (same as the grid cell) */
	x: number;
	y: number;
	/** damage or heal amount, 0 for miss/block/parry/death */
	n: number;
	attackerId: number;
	targetId: number;
	elem?: 'fire' | 'frost' | 'shock' | 'acid' | 'poison' | 'dark' | 'holy';
	boss: boolean;
	/** the target is the player */
	player: boolean;
}

export interface HitMetrics {
	cw: number;
	chh: number;
	dpr: number;
	/** grid size in cells */
	cols: number;
	rows: number;
}

export interface HitFxOptions {
	/** floating numbers and flashes */
	numbers: boolean;
	shake: boolean;
	pause: boolean;
	reduced: boolean;
}

export interface HitFxHandle {
	setOptions(o: Partial<HitFxOptions>): void;
	/** feed `frame.hits` once per key frame */
	push(events: HitEvent[] | undefined): void;
	/**
	 * Resolves after the hit pause when the events just pushed called for one (a crit, or a hit by or on a boss);
	 * null when there is nothing to wait for, so callers can skip the await. Never drops input: the shell keeps a
	 * key queue and awaits this between keys.
	 */
	pause(): Promise<void> | null;
	/** cells to tint as "will be hit next turn", if the engine ever sends them (frame.tele); [] clears */
	setTelegraphs(cells: { x: number; y: number }[] | undefined): void;
	/** call after a resize or zone change; drops everything in flight when `clear` */
	layout(clear?: boolean): void;
	dispose(): void;
}

interface Num {
	t0: number;
	x: number;
	y: number;
	text: string;
	color: string;
	size: number;
	life: number;
	rise: number;
	dx: number;
}
interface Flash {
	t0: number;
	x: number;
	y: number;
	life: number;
}
interface Puff {
	t0: number;
	x: number;
	y: number;
	life: number;
	seed: number;
}

const NUM_MS = 850;
const STATIC_MS = 600;
const FLASH_MS = 130;
const PUFF_MS = 480;
const SHAKE_MS = 240;
const PAUSE_MS = 60;
const MAX_NUMS = 40;

const ELEM_COLOR: Record<string, string> = {
	fire: '#ff9a3c',
	frost: '#8fe6ff',
	shock: '#fff06a',
	acid: '#b6ff4a',
	poison: '#b98cff',
	dark: '#c26bff',
	holy: '#fff2b0',
};

function style(e: HitEvent): { text: string; color: string; size: number } {
	switch (e.kind) {
		case 'crit':
			return { text: `${e.n}!`, color: '#ffd23c', size: 1.5 };
		case 'miss':
			return { text: 'miss', color: '#9a9a9a', size: 0.8 };
		case 'block':
			return { text: 'block', color: '#a9c4d8', size: 0.85 };
		case 'parry':
			return { text: 'parry', color: '#d8e6f0', size: 0.9 };
		case 'heal':
			return { text: `+${e.n}`, color: '#5cf07a', size: 1 };
		case 'burn':
			return { text: `${e.n}`, color: '#ff8a2a', size: 0.85 };
		case 'poison':
			return { text: `${e.n}`, color: '#9be05a', size: 0.85 };
		case 'bleed':
			return { text: `${e.n}`, color: '#c0242c', size: 0.85 };
		default:
			if (e.player) return { text: `-${e.n}`, color: '#ff4a4a', size: 1.1 };
			return { text: `${e.n}`, color: (e.elem && ELEM_COLOR[e.elem]) || '#fff3e0', size: 1 };
	}
}

export function createHitFx(canvas: HTMLCanvasElement, getMetrics: () => HitMetrics): HitFxHandle {
	const ctx = canvas.getContext('2d')!;
	const opt: HitFxOptions = { numbers: true, shake: true, pause: true, reduced: false };
	let nums: Num[] = [];
	let flashes: Flash[] = [];
	let puffs: Puff[] = [];
	let tele: { x: number; y: number }[] = [];
	let raf = 0;
	let shakeAt = -1e9;
	let shakeAmp = 0;
	let wantPause = false;
	let sizedFor = '';
	const shakeEl = canvas.parentElement;

	function now(): number {
		return performance.now();
	}

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

	function alive(): boolean {
		return nums.length > 0 || flashes.length > 0 || puffs.length > 0 || tele.length > 0 || now() - shakeAt < SHAKE_MS;
	}

	function start() {
		if (!raf) raf = requestAnimationFrame(frame);
	}

	function applyShake(t: number) {
		if (!shakeEl) return;
		const d = t - shakeAt;
		if (opt.reduced || !opt.shake || d < 0 || d >= SHAKE_MS) {
			if (shakeEl.style.transform) shakeEl.style.transform = '';
			return;
		}
		const k = 1 - d / SHAKE_MS;
		const a = shakeAmp * k * k;
		shakeEl.style.transform = `translate(${(Math.sin(d * 0.21) * a).toFixed(1)}px, ${(Math.cos(d * 0.33) * a * 0.6).toFixed(1)}px)`;
	}

	function frame(t: number) {
		raf = 0;
		const m = fit();
		const { cw, chh, dpr } = m;
		ctx.setTransform(dpr, 0, 0, dpr, 0, 0);
		ctx.clearRect(0, 0, m.cols * cw, m.rows * chh);
		// telegraphs
		if (tele.length) {
			// a pixel-checker warning: alternate art pixels, the checker flips on a slow beat
			const pulse = opt.reduced ? 0.5 : 0.35 + 0.25 * (Math.sin(t * 0.009) > 0 ? 1 : 0);
			const px = Math.max(1, Math.round((chh / 20) * dpr)) / dpr;
			const flip = opt.reduced ? 0 : Math.floor(t / 260) & 1;
			ctx.fillStyle = `rgba(230,50,40,${pulse})`;
			for (const c of tele) {
				const x0 = c.x * cw, y0 = c.y * chh;
				for (let yy = 0, iy = 0; yy < chh - 0.01; yy += px, iy++)
					for (let xx = 0, ix = 0; xx < cw - 0.01; xx += px, ix++) if (((ix + iy + flip) & 1) === 0) ctx.fillRect(x0 + xx, y0 + yy, px, px);
			}
		}
		// flashes
		if (flashes.length) {
			flashes = flashes.filter((f) => t - f.t0 < f.life);
			for (const f of flashes) {
				const a = 0.75 * (1 - (t - f.t0) / f.life);
				ctx.fillStyle = `rgba(255,255,255,${a.toFixed(3)})`;
				ctx.fillRect(f.x * cw, f.y * chh, cw, chh);
			}
		}
		// puffs
		if (puffs.length) {
			puffs = puffs.filter((p) => t - p.t0 < p.life);
			for (const p of puffs) {
				const k = (t - p.t0) / p.life;
				const cx = (p.x + 0.5) * cw, cy = (p.y + 0.55) * chh;
				for (let i = 0; i < 7; i++) {
					const ang = ((i + p.seed * 0.37) / 7) * Math.PI * 2 + p.seed;
					const r = (0.25 + 0.7 * k) * cw * (0.7 + 0.5 * ((i * 7 + p.seed) % 3) / 3);
					const px = cx + Math.cos(ang) * r;
					const py = cy + Math.sin(ang) * r * 0.8 - k * chh * 0.35;
					ctx.fillStyle = i % 3 === 0 ? `rgba(230,200,150,${(0.55 * (1 - k)).toFixed(3)})` : `rgba(170,170,170,${(0.5 * (1 - k)).toFixed(3)})`;
					ctx.beginPath();
					ctx.arc(px, py, Math.max(1, cw * (0.22 + 0.2 * k)), 0, Math.PI * 2);
					ctx.fill();
				}
			}
		}
		// numbers
		if (nums.length) {
			nums = nums.filter((n) => t - n.t0 < n.life);
			ctx.textAlign = 'center';
			ctx.textBaseline = 'middle';
			ctx.lineJoin = 'round';
			for (const n of nums) {
				const k = (t - n.t0) / n.life;
				const rise = opt.reduced ? 0 : n.rise * chh * Math.min(1, k * 1.6);
				const alpha = opt.reduced ? (k > 0.8 ? 1 - (k - 0.8) / 0.2 : 1) : k < 0.55 ? 1 : 1 - (k - 0.55) / 0.45;
				const pop = opt.reduced || k > 0.15 ? 1 : 1 + (1 - k / 0.15) * 0.35;
				const px = Math.round(chh * 0.8 * n.size * pop);
				ctx.font = `bold ${px}px "Share Tech Mono", "DejaVu Sans Mono", Menlo, Consolas, monospace`;
				const x = (n.x + 0.5) * cw + n.dx * cw;
				const y = n.y * chh + chh * 0.2 - rise;
				ctx.globalAlpha = Math.max(0, alpha);
				ctx.lineWidth = Math.max(2, px * 0.18);
				ctx.strokeStyle = 'rgba(0,0,0,0.85)';
				ctx.strokeText(n.text, x, y);
				ctx.fillStyle = n.color;
				ctx.fillText(n.text, x, y);
			}
			ctx.globalAlpha = 1;
		}
		applyShake(t);
		if (alive()) start();
		else {
			ctx.clearRect(0, 0, m.cols * cw, m.rows * chh);
			applyShake(t + SHAKE_MS);
		}
	}

	return {
		setOptions(o) {
			Object.assign(opt, o);
			if (!opt.numbers) {
				nums = [];
				flashes = [];
				puffs = [];
			}
			if (opt.reduced) {
				flashes = [];
				puffs = [];
			}
			start();
		},
		push(events) {
			wantPause = false;
			if (!events || !events.length) return;
			const t = now();
			const perCell = new Map<number, number>();
			let hurtPlayer = 0;
			for (const e of events) {
				if (e.kind === 'death') {
					if (opt.numbers && !opt.reduced) puffs.push({ t0: t, x: e.x, y: e.y, life: PUFF_MS, seed: (e.x * 31 + e.y * 17 + e.targetId) % 11 });
					continue;
				}
				if (opt.numbers) {
					const s = style(e);
					const ck = e.y * 4096 + e.x;
					const idx = perCell.get(ck) ?? 0;
					perCell.set(ck, idx + 1);
					// later numbers on the same cell sit higher and to the side so they stay readable
					if (nums.length < MAX_NUMS) {
						nums.push({
							t0: t,
							x: e.x,
							y: e.y - idx * 0.7,
							text: s.text,
							color: s.color,
							size: s.size,
							life: opt.reduced ? STATIC_MS : NUM_MS,
							rise: e.kind === 'crit' ? 1.5 : 1.1,
							dx: idx ? (idx % 2 ? 0.35 : -0.35) : 0,
						});
					}
					const dmg = e.kind === 'hit' || e.kind === 'crit' || e.kind === 'burn' || e.kind === 'poison' || e.kind === 'bleed';
					if (dmg && !opt.reduced) flashes.push({ t0: t, x: e.x, y: e.y, life: e.kind === 'crit' ? FLASH_MS * 1.5 : FLASH_MS });
				}
				if (e.player && (e.kind === 'hit' || e.kind === 'crit')) hurtPlayer = Math.max(hurtPlayer, e.n + (e.kind === 'crit' ? 2 : 0) + (e.boss ? 2 : 0));
				if (e.kind === 'crit' || (e.boss && e.kind === 'hit')) wantPause = true;
			}
			if (hurtPlayer > 0 && opt.shake && !opt.reduced) {
				const m = getMetrics();
				shakeAmp = Math.min(9, 1.5 + hurtPlayer * 0.55) * Math.max(0.7, m.cw / 9);
				shakeAt = t;
			}
			start();
		},
		pause() {
			if (!wantPause) return null;
			wantPause = false;
			if (!opt.pause || opt.reduced) return null;
			return new Promise<void>((res) => setTimeout(res, PAUSE_MS));
		},
		setTelegraphs(cells) {
			tele = cells ? cells.slice(0, 40) : [];
			if (tele.length) start();
			else if (!alive()) {
				const m = getMetrics();
				ctx.clearRect(0, 0, m.cols * m.cw, m.rows * m.chh);
			}
		},
		layout(clear) {
			sizedFor = '';
			if (clear) {
				nums = [];
				flashes = [];
				puffs = [];
				tele = [];
			}
			start();
		},
		dispose() {
			if (raf) cancelAnimationFrame(raf);
			raf = 0;
			nums = flashes = puffs = [];
			if (shakeEl) shakeEl.style.transform = '';
		},
	};
}
