/**
 * Sprite-mode weather and light: a transparent canvas stacked over the map (80x25 cells). Rain, snow,
 * fog, dust, spores, lightning, heat shimmer, a day-night tint with soft light pools cut out around
 * torches and the player, hit flash. Purely cosmetic and shell-side; the engine only sends
 * `frame.fx` (engine/fxinfo.go). Gradients are pre-rendered, particles live in typed arrays (<= 250).
 */
import type { QudFrame, QudFx } from './qud';

const MAP_W = 80;
const MAP_H = 25;
const MAXP = 250;
const FRAME_MS = 33;

type RGB = [number, number, number];

function mulberry32(a: number) {
	return () => {
		a = (a + 0x6d2b79f5) | 0;
		let t = Math.imul(a ^ (a >>> 15), 1 | a);
		t = (t + Math.imul(t ^ (t >>> 7), 61 | t)) ^ t;
		return ((t ^ (t >>> 14)) >>> 0) / 4294967296;
	};
}
const rnd = mulberry32(0x5eed1234);

function glowSprite(rgb: RGB | null, size = 128): HTMLCanvasElement {
	const c = document.createElement('canvas');
	c.width = c.height = size;
	const g = c.getContext('2d')!;
	const [r, gg, b] = rgb ?? [255, 255, 255];
	const gr = g.createRadialGradient(size / 2, size / 2, 0, size / 2, size / 2, size / 2);
	gr.addColorStop(0, `rgba(${r},${gg},${b},1)`);
	gr.addColorStop(0.35, `rgba(${r},${gg},${b},0.75)`);
	gr.addColorStop(0.7, `rgba(${r},${gg},${b},0.25)`);
	gr.addColorStop(1, `rgba(${r},${gg},${b},0)`);
	g.fillStyle = gr;
	g.fillRect(0, 0, size, size);
	return c;
}

const GLOW_RGB: Record<string, RGB> = {
	torch: [255, 150, 60],
	fire: [255, 120, 40],
	lava: [255, 80, 30],
	lantern: [255, 205, 110],
	crystal: [190, 110, 255],
	glow: [200, 90, 230],
	altar: [255, 220, 120],
	rift: [170, 80, 255],
	player: [255, 220, 150],
};

/** colour of the particles / banks per weather kind */
function wxColor(id: string): RGB {
	switch (id) {
		case 'acidrain': case 'fallout': case 'radstorm': case 'miasma': return [150, 230, 110];
		case 'sporerain': case 'spores': return [220, 130, 230];
		case 'shimmer': return [200, 170, 255];
		case 'pollen': case 'sandstorm': return id === 'pollen' ? [255, 220, 110] : [225, 190, 120];
		case 'bloodmoon': return [220, 60, 60];
		case 'smog': case 'steamfog': return [170, 150, 100];
		case 'ashfall': case 'ashrain': return [170, 170, 170];
		case 'snow': case 'blizzard': return [245, 248, 255];
		case 'fog': return [225, 230, 235];
		default: return [170, 200, 255];
	}
}

export interface FxHandle {
	setFrame(f: QudFrame): void;
	/** cell size in CSS px and device pixel ratio; call on every resize */
	layout(cw: number, chh: number, dpr: number): void;
	setOptions(o: { enabled: boolean; reduced: boolean }): void;
	/** player shake in px while a hit lands */
	shake(): { x: number; y: number };
	/** called every animation frame while animating (for sprite bob) */
	onTick(cb: ((t: number) => void) | null): void;
	dispose(): void;
}

export function createEffects(canvas: HTMLCanvasElement): FxHandle {
	const ctx = canvas.getContext('2d')!;
	const dark = document.createElement('canvas');
	const dctx = dark.getContext('2d')!;
	const glowW = glowSprite(null);
	const glowC: Record<string, HTMLCanvasElement> = {};
	const bankC = new Map<string, HTMLCanvasElement>();
	let cw = 9, chh = 16, W = MAP_W * 9, H = MAP_H * 16, dpr = 1, ds = 0.5;
	let enabled = false, reduced = false;
	let fx: QudFx | undefined;
	let active = false; // a map screen
	let raf = 0, last = 0, tickCb: ((t: number) => void) | null = null;
	let prevBolt: number | undefined, prevHp: number | undefined, flashAt = -1e9, hitAt = -1e9;
	let px = { x: 0, y: 0 };
	let ppx = 0, ppy = 0;
	// particles
	const X = new Float32Array(MAXP), Y = new Float32Array(MAXP), VX = new Float32Array(MAXP), VY = new Float32Array(MAXP), G = new Float32Array(MAXP), S = new Float32Array(MAXP);
	let kind = '', count = 0;

	function spawn(i: number, scatter: boolean) {
		X[i] = rnd() * W;
		Y[i] = scatter ? rnd() * H : -rnd() * H * 0.2;
		G[i] = H * (0.12 + rnd() * 0.88);
		S[i] = 0;
		const ov = fx?.ov ?? '', wind = fx?.wind ?? 0;
		if (ov === 'rain' || ov === 'storm') {
			VY[i] = chh * (26 + rnd() * 10);
			VX[i] = wind * VY[i] * 0.28 + (ov === 'storm' ? VY[i] * 0.08 : 0);
		} else if (ov === 'snow') {
			const bl = fx?.wx === 'blizzard';
			VY[i] = chh * (2.5 + rnd() * 2.5) * (bl ? 2.2 : 1);
			VX[i] = (bl ? (wind || 1) * chh * (12 + rnd() * 8) : wind * chh * 2) + (rnd() - 0.5) * chh;
		} else if (ov === 'dust') {
			VX[i] = (wind || 1) * chh * (16 + rnd() * 14);
			VY[i] = (rnd() - 0.4) * chh * 2;
		} else if (ov === 'ash') {
			VY[i] = chh * (1.5 + rnd() * 1.5);
			VX[i] = wind * chh * 2 + (rnd() - 0.5) * chh;
		} else {
			VY[i] = -chh * (0.3 + rnd() * 0.8);
			VX[i] = (wind * 0.8 + (rnd() - 0.5)) * chh;
		}
	}

	function sync() {
		const ov = fx?.ov ?? '';
		const k = fx && !fx.ind ? `${ov}|${fx.wx}` : '';
		const inten = fx?.in ?? 0;
		let n = 0;
		if (k) {
			if (ov === 'rain' || ov === 'storm') n = 40 + 210 * inten;
			else if (ov === 'snow') n = 60 + 190 * inten;
			else if (ov === 'dust') n = 120 + 130 * inten;
			else if (ov === 'ash') n = 90 + 60 * inten;
			else if (ov === 'spore') n = 50 + 40 * inten;
		}
		n = Math.min(MAXP, Math.floor(n));
		if (k !== kind) {
			kind = k;
			for (let i = 0; i < MAXP; i++) spawn(i, true);
		}
		count = n;
	}

	function bank(rgb: RGB): HTMLCanvasElement {
		const key = rgb.join(',');
		let b = bankC.get(key);
		if (!b) {
			b = glowSprite(rgb, 128);
			bankC.set(key, b);
		}
		return b;
	}

	function drawWeather(t: number, dt: number, animate: boolean) {
		if (!fx || fx.ind || !fx.wx) return;
		const ov = fx.ov ?? '', inten = fx.in ?? 0, col = wxColor(fx.wx);
		const rgb = `${col[0]},${col[1]},${col[2]}`;
		if (ov === 'fog' || fx.wx === 'blizzard') {
			const white = fx.wx === 'blizzard';
			const bw = W * 0.7, bh = H * 0.9;
			ctx.globalAlpha = (white ? 0.3 : 0.38) * (0.4 + inten * 0.6);
			for (let k = 0; k < 6; k++) {
				const sp = (white ? 0.09 : 0.02 + k * 0.004) * W * (fx.wind || 1);
				const x0 = (k * 0.37 * (W + bw) + (animate ? t * sp : 0)) % (W + bw);
				const x = sp >= 0 ? x0 - bw : W - x0;
				ctx.drawImage(bank(col), x, ((k * 53) % 7) / 7 * (H - bh * 0.6) - bh * 0.2, bw, bh);
			}
			ctx.globalAlpha = 1;
			if (white) {
				ctx.fillStyle = 'rgba(235,240,250,0.2)';
				ctx.fillRect(0, 0, W, H);
			}
		}
		if (ov === 'dust') {
			ctx.fillStyle = `rgba(${rgb},${0.1 + 0.12 * inten})`;
			ctx.fillRect(0, 0, W, H);
		}
		if (fx.wx === 'heat' && animate) {
			ctx.strokeStyle = 'rgba(255,225,160,0.05)';
			ctx.lineWidth = chh * 0.7;
			for (let k = 0; k < 5; k++) {
				const y0 = ((t * 0.012 * chh + k * H / 5) % H);
				ctx.beginPath();
				for (let x = 0; x <= W; x += 24) {
					const y = y0 + Math.sin(x * 0.03 + t * 0.002 + k) * chh * 0.35;
					if (x === 0) ctx.moveTo(x, y);
					else ctx.lineTo(x, y);
				}
				ctx.stroke();
			}
		}
		if (!count || !animate) return;
		const storm = ov === 'storm';
		const rain = ov === 'rain' || storm;
		if (rain) {
			ctx.strokeStyle = `rgba(${rgb},0.55)`;
			ctx.lineWidth = 1;
			ctx.beginPath();
			const hail: number[] = [];
			for (let i = 0; i < count; i++) {
				if (storm && i % 7 === 0) hail.push(i);
				if (S[i] > 0) {
					S[i] -= dt;
					if (S[i] <= 0) spawn(i, false);
					continue;
				}
				X[i] += VX[i] * dt;
				Y[i] += VY[i] * dt;
				if (Y[i] >= G[i]) {
					S[i] = 0.12;
					continue;
				}
				ctx.moveTo(X[i], Y[i]);
				ctx.lineTo(X[i] - VX[i] * 0.03, Y[i] - VY[i] * 0.03);
			}
			ctx.stroke();
			// splashes
			ctx.fillStyle = `rgba(${rgb},0.5)`;
			ctx.beginPath();
			for (let i = 0; i < count; i++) {
				if (S[i] > 0) {
					const r = 1 + (0.12 - S[i]) * 14;
					ctx.moveTo(X[i] + r * 1.6, G[i]);
					ctx.ellipse(X[i], G[i], r * 1.6, r * 0.6, 0, 0, 6.283);
				}
			}
			ctx.fill();
			if (hail.length) {
				ctx.fillStyle = 'rgba(240,248,255,0.85)';
				ctx.beginPath();
				for (const i of hail) {
					ctx.moveTo(X[i] + 1.6, Y[i]);
					ctx.arc(X[i], Y[i], 1.6, 0, 6.283);
				}
				ctx.fill();
			}
			for (let i = 0; i < count; i++) if (X[i] < -20 || X[i] > W + 20) spawn(i, false);
			return;
		}
		const snow = ov === 'snow';
		ctx.fillStyle = snow ? 'rgba(250,252,255,0.85)' : ov === 'ash' ? 'rgba(175,175,175,0.7)' : `rgba(${rgb},${ov === 'dust' ? 0.5 : 0.75})`;
		ctx.beginPath();
		for (let i = 0; i < count; i++) {
			X[i] += VX[i] * dt;
			Y[i] += VY[i] * dt;
			if (X[i] > W + 10) X[i] = -10;
			else if (X[i] < -10) X[i] = W + 10;
			if (Y[i] > H + 6 || Y[i] < -H * 0.25) spawn(i, false), (Y[i] = Y[i] < 0 && ov === 'spore' ? H + 4 : Y[i]);
			const sway = snow || ov === 'spore' ? Math.sin(t * 0.0012 + i) * cw * 0.6 : 0;
			const x = X[i] + sway;
			if (ov === 'dust') {
				ctx.rect(x, Y[i], 4 + (i % 3), 1);
			} else {
				const r = snow ? 1 + (i % 3) * 0.6 : ov === 'ash' ? 1.2 : 1.4 + (i % 3) * 0.5;
				ctx.moveTo(x + r, Y[i]);
				ctx.arc(x, Y[i], r, 0, 6.283);
			}
		}
		ctx.fill();
	}

	function drawLight(t: number, animate: boolean) {
		if (!fx) return;
		const night = 1 - fx.dl;
		const underground = (fx.d ?? 0) > 0 || fx.dk;
		let a = underground ? Math.min(0.82, 0.5 + (fx.d ?? 0) * 0.04) : fx.ind ? night * 0.3 : night * 0.58 + (fx.in ?? 0) * 0.1 * fx.dl;
		const flash = animate ? Math.max(0, flashLevel(t)) : 0;
		a *= 1 - flash * 0.75;
		const blood = fx.wx === 'bloodmoon' && !fx.ind;
		if (a > 0.02) {
			const dw = dark.width, dh = dark.height;
			dctx.globalCompositeOperation = 'source-over';
			dctx.globalAlpha = 1;
			dctx.clearRect(0, 0, dw, dh);
			dctx.fillStyle = blood ? `rgba(60,6,12,${a})` : `rgba(8,14,46,${a})`;
			dctx.fillRect(0, 0, dw, dh);
			dctx.globalCompositeOperation = 'destination-out';
			for (const l of fx.lights) {
				const f = animate ? flick(l, t) : 1;
				const R = l.r * chh * 1.15 * f * ds * dpr;
				dctx.globalAlpha = l.k === 'player' ? 0.92 : 1;
				dctx.drawImage(glowW, (l.x + 0.5) * cw * ds * dpr - R, (l.y + 0.5) * chh * ds * dpr - R, R * 2, R * 2);
			}
			dctx.globalAlpha = 1;
			ctx.drawImage(dark, 0, 0, W, H);
			// warm glow where the dark is cut open
			ctx.globalCompositeOperation = 'lighter';
			for (const l of fx.lights) {
				if (l.k === 'player') continue;
				const f = animate ? flick(l, t) : 1;
				const R = l.r * chh * 0.95 * f;
				let sp = glowC[l.k];
				if (!sp) sp = glowC[l.k] = glowSprite(GLOW_RGB[l.k] ?? GLOW_RGB.torch);
				ctx.globalAlpha = Math.min(0.4, 0.1 + a * 0.45) * (0.8 + 0.2 * f);
				ctx.drawImage(sp, (l.x + 0.5) * cw - R, (l.y + 0.5) * chh - R, R * 2, R * 2);
			}
			ctx.globalAlpha = 1;
			ctx.globalCompositeOperation = 'source-over';
		}
		// dawn and dusk wash
		if (!underground && fx.dl > 0 && fx.dl < 1) {
			const w = 1 - Math.abs(2 * fx.dl - 1);
			ctx.fillStyle = `rgba(255,140,60,${0.17 * w * (fx.ind ? 0.4 : 1)})`;
			ctx.fillRect(0, 0, W, H);
		}
		if (fx.wx === 'frost' && !fx.ind) {
			ctx.fillStyle = 'rgba(150,200,255,0.07)';
			ctx.fillRect(0, 0, W, H);
		}
		if (flash > 0) {
			ctx.fillStyle = `rgba(235,240,255,${flash * 0.6})`;
			ctx.fillRect(0, 0, W, H);
		}
	}

	function flick(l: { x: number; y: number; fl: number }, t: number): number {
		const ph = l.x * 1.7 + l.y * 2.9;
		return 1 + l.fl * (0.09 * Math.sin(t * 0.011 + ph) + 0.06 * Math.sin(t * 0.027 + ph * 2.3) + 0.04 * Math.sin(t * 0.059 + ph * 0.7));
	}

	function flashLevel(t: number): number {
		const d = t - flashAt;
		if (d < 0 || d > 420) return 0;
		if (d < 90) return 0.9 * (1 - d / 160);
		if (d < 150) return 0.1;
		return 0.7 * (1 - (d - 150) / 270);
	}

	function render(t: number, dt: number, animate: boolean) {
		ctx.setTransform(dpr, 0, 0, dpr, 0, 0);
		ctx.clearRect(0, 0, W, H);
		if (!enabled || !active || !fx) return;
		drawWeather(t, dt, animate);
		drawLight(t, animate);
		const hd = t - hitAt;
		if (animate && hd >= 0 && hd < 380) {
			const k = 1 - hd / 380;
			const gr = ctx.createRadialGradient(W / 2, H / 2, Math.min(W, H) * 0.3, W / 2, H / 2, Math.max(W, H) * 0.7);
			gr.addColorStop(0, 'rgba(200,20,20,0)');
			gr.addColorStop(1, `rgba(200,20,20,${0.4 * k})`);
			ctx.fillStyle = gr;
			ctx.fillRect(0, 0, W, H);
			const R = chh * 2.2;
			ctx.globalAlpha = 0.5 * k;
			ctx.drawImage(bank([230, 40, 40]), (ppx + 0.5) * cw - R, (ppy + 0.5) * chh - R, R * 2, R * 2);
			ctx.globalAlpha = 1;
			px = { x: Math.round(Math.sin(hd * 0.09) * 2 * k), y: 0 };
		} else px = { x: 0, y: 0 };
	}

	function loop(now: number) {
		raf = requestAnimationFrame(loop);
		if (document.hidden || now - last < FRAME_MS) return;
		const dt = Math.min(0.06, (now - last) / 1000);
		last = now;
		render(now, dt, true);
		tickCb?.(now);
	}
	function run() {
		const want = enabled && !reduced && !document.hidden;
		if (want && !raf) {
			last = performance.now();
			raf = requestAnimationFrame(loop);
		} else if (!want && raf) {
			cancelAnimationFrame(raf);
			raf = 0;
		}
	}
	const onVis = () => run();
	document.addEventListener('visibilitychange', onVis);

	function size() {
		W = MAP_W * cw;
		H = MAP_H * chh;
		canvas.width = Math.round(W * dpr);
		canvas.height = Math.round(H * dpr);
		canvas.style.width = `${W}px`;
		canvas.style.height = `${H}px`;
		dark.width = Math.max(1, Math.round(W * dpr * ds));
		dark.height = Math.max(1, Math.round(H * dpr * ds));
		dctx.setTransform(1, 0, 0, 1, 0, 0);
		kind = '';
		sync();
	}

	return {
		setFrame(f) {
			const map = f.mode === 'main' || f.mode === 'target';
			active = map && !f.portrait;
			fx = f.fx;
			const m = f.music;
			if (m) {
				const now = performance.now();
				if (prevBolt !== undefined && (m.bolt ?? 0) > prevBolt && !reduced) flashAt = now;
				if (m.bolt !== undefined) prevBolt = m.bolt;
				if (prevHp !== undefined && m.hp < prevHp - 0.001 && !reduced) hitAt = now;
				prevHp = m.hp;
			}
			const p = f.actors?.find((a) => a.p);
			if (p) (ppx = p.x), (ppy = p.y);
			sync();
			if (!raf) render(0, 0, false);
		},
		layout(w, h, d) {
			cw = w;
			chh = h;
			dpr = d;
			size();
			if (!raf) render(0, 0, false);
		},
		setOptions(o) {
			enabled = o.enabled;
			reduced = o.reduced;
			canvas.style.display = enabled ? 'block' : 'none';
			run();
			if (!raf) render(0, 0, false);
		},
		shake: () => px,
		onTick(cb) {
			tickCb = cb;
		},
		dispose() {
			if (raf) cancelAnimationFrame(raf);
			raf = 0;
			document.removeEventListener('visibilitychange', onVis);
		},
	};
}

/** Soft ellipse under a sprite; in daylight it is thrown away from the sun (east in the morning, west at dusk). */
export function drawActorShadow(c: CanvasRenderingContext2D, x: number, y: number, cw: number, chh: number, fx?: QudFx) {
	const day = fx ? fx.dl : 0.5;
	const outdoor = fx && !fx.ind;
	const frac = fx ? Math.min(1, Math.max(0, (fx.ph * 24 - 6) / 14)) : 0.5;
	const dx = outdoor && day > 0.1 ? (frac - 0.5) * 2 * cw * 0.45 : 0;
	c.fillStyle = `rgba(0,0,0,${outdoor ? 0.12 + day * 0.14 : 0.2})`;
	c.beginPath();
	c.ellipse(x + cw / 2 + dx, y + chh * 0.92, cw * (0.38 + Math.abs(dx) / cw * 0.4), chh * 0.07, 0, 0, 6.283);
	c.fill();
}

/** Idle bob in px for an actor id at time t (ms): about 1 px, out of step between actors. */
export function bobOffset(id: number, t: number): number {
	return Math.sin(t / 520 + id * 1.7);
}
