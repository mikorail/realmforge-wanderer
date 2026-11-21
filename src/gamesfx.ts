// Event sound effects and adaptive audio, all synthesised with WebAudio on their own AudioContext.
//
// Covers what weathersfx.ts does not: its playCombatSfx has only swing/hit/crit/bow/spell/block/levelup/coin/ui,
// and its beds are weather only. This file adds footsteps by surface, doors, pickups, equip, eat/drink, quest
// complete, hits by weapon class, gunshot, miss, parry, hurt, death, spells by element, UI move/select/back/error,
// page turn, lock click and alarm, plus the adaptive layer: heartbeat at low HP, a boss drone, a lowpass muffle
// underwater or when deafened, and a reverb tail in caves versus dry outdoors.
//
// Events come from `frame.sfx` (engine/sfxinfo.go SfxBegin/SfxEnd, ids like "step:grass", "hit:blade"); when the
// engine does not send it, inferFromDiff() guesses a few from consecutive frames. `frame.sfxs` (SfxScene) adds
// cave/underwater/deaf/boss/hostile-count facts that frame.music lacks.
import type { QudFrame, QudMusic } from './qud';

export interface SceneState {
	indoors: boolean;
	cave: boolean;
	underwater: boolean;
	deaf: boolean;
	/** 0..1: hostiles in view, low HP, boss, night */
	tension: number;
	/** 0..1 how low the player's HP is, for the heartbeat (derived from tension inputs when not given) */
	hp: number;
	boss: boolean;
}

export interface GameSfxHandle {
	/** play one event id ("step:stone", "hit:blade", "coin", "ui:select", ...) */
	play(id: string): void;
	/** feed a frame (after a key press): plays frame.sfx or the fallback guess, and updates the scene */
	handleFrame(f: QudFrame, key?: string): void;
	setScene(s: Partial<SceneState>): void;
	setVolume(v: number): void;
	/** pref `sfx` AND `sfxEvents` */
	setEnabled(on: boolean): void;
	/** honour the reduceMotion pref: no pulsing heartbeat */
	setReduceMotion(on: boolean): void;
	dispose(): void;
}

export type EventId = string;

type FrameEx = QudFrame & {
	sfx?: string[];
	sfxs?: { cave?: boolean; uw?: boolean; deaf?: boolean; boss?: boolean; n?: number };
};

const clamp01 = (x: number) => Math.min(1, Math.max(0, x));

/** tensionFor combines hostiles in view, HP, boss and time of day into 0..1 (engine danger already folds the first three). */
export function tensionFor(m: QudMusic | null, hostiles = 0, boss = false): number {
	if (!m || m.mood === 'world' || m.mood === 'dead') return 0;
	let t = m.danger * 0.8;
	t = Math.max(t, Math.min(1, hostiles * 0.25));
	if (boss) t = Math.max(t, 0.85);
	t += (1 - clamp01(m.hp)) * 0.3;
	if (!m.day) t += 0.08;
	if (m.mood === 'crypt' || m.mood === 'underdark') t += 0.05;
	return clamp01(t);
}

/** sceneFor derives the scene from a frame. */
export function sceneFor(f: FrameEx): SceneState {
	const m = f.music;
	const x = f.sfxs;
	const cave = x?.cave ?? (m.mood === 'dungeon' || m.mood === 'underdark' || m.mood === 'crypt' || m.mood === 'tower');
	const hostiles = x?.n ?? (m.combat ? 1 : 0);
	const boss = !!x?.boss;
	return {
		indoors: !!f.fx?.ind,
		cave,
		underwater: !!x?.uw,
		deaf: !!x?.deaf,
		tension: tensionFor(m, hostiles, boss),
		hp: m.mood === 'dead' || m.mood === 'world' ? 1 : clamp01(m.hp),
		boss,
	};
}

/** inferFromDiff is the shell-side fallback: a few events guessed from two consecutive frames. */
export function inferFromDiff(prev: FrameEx | null, next: FrameEx, key?: string): EventId[] {
	const out: EventId[] = [];
	if (!prev) return out;
	const a = prev.music, b = next.music;
	if (b.mood === 'dead' && a.mood !== 'dead') out.push('death');
	else if (b.hp < a.hp - 0.001 && b.mood !== 'world') out.push('hurt');
	if (prev.mode !== next.mode) {
		if (next.mode === 'main' || next.mode === 'world') out.push('ui:back');
		else if (prev.mode === 'main' || prev.mode === 'world') out.push('ui:select');
	} else if (next.mode !== 'main' && next.mode !== 'world' && next.mode !== 'dead' && key) {
		if (key === 'Escape') out.push('ui:back');
		else if (key === 'Enter' || key === ' ') out.push('ui:select');
		else if (key.startsWith('Arrow') || /^[jkhl2468]$/.test(key)) out.push('ui:move');
	}
	if (next.mode === 'main' && prev.mode === 'main' && a.mood === b.mood && a.depth === b.depth && (key?.startsWith('Arrow') || /^[hjklyubn1-9]$/.test(key ?? ''))) {
		// the map changed under a move key: a step on the terrain of the mood
		const surf = b.mood === 'town' ? 'stone' : b.mood === 'dungeon' || b.mood === 'crypt' || b.mood === 'tower' || b.mood === 'underdark' ? 'stone' : b.mood === 'sea' ? 'water' : b.wx === 'snow' || b.wx === 'blizzard' ? 'snow' : 'grass';
		if (prev.g !== next.g) out.push('step:' + surf);
	}
	return out;
}

const rnd = (a: number, b: number) => a + Math.random() * (b - a);
const pv = (f: number, amt = 0.06) => f * (1 + (Math.random() * 2 - 1) * amt);

/** createGameSfx: the live layer for the game screen, on its own AudioContext. */
/** ids another layer (foley.ts) plays now; skipped while it is on and the frame carries foley events (set by audiohub.ts) */
let foleyCovers: ReadonlySet<string> | null = null;
export function setFoleyCovers(s: ReadonlySet<string> | null): void {
	foleyCovers = s;
}

export function createGameSfx(): GameSfxHandle {
	let ctx: AudioContext | null = null;
	let master: GainNode | null = null;
	let muffle: BiquadFilterNode | null = null;
	let wet: GainNode | null = null;
	let bus: GainNode | null = null; // dry+send point every event connects to
	let droneG: GainNode | null = null;
	let droneNodes: AudioScheduledSourceNode[] = [];
	let noiseBuf: AudioBuffer | null = null;
	let timer: ReturnType<typeof setInterval> | null = null;
	let suspendT: ReturnType<typeof setTimeout> | null = null;
	let enabled = true;
	let reduce = false;
	let volume = 0.5;
	let disposed = false;
	let nextBeat = 0;
	let lastStep = 0;
	let prevFrame: FrameEx | null = null;
	let scene: SceneState = { indoors: false, cave: false, underwater: false, deaf: false, tension: 0, hp: 1, boss: false };
	const listeners: Array<[EventTarget, string, EventListener]> = [];

	function listen(t: EventTarget, ev: string, fn: EventListener) {
		t.addEventListener(ev, fn);
		listeners.push([t, ev, fn]);
	}
	function onHide() {
		stopTimer();
		if (ctx && ctx.state === 'running') ctx.suspend().catch(() => {});
	}
	function onShow() {
		if (disposed || !enabled || !ctx) return;
		ctx.resume().catch(() => {});
		startTimer();
	}
	if (typeof document !== 'undefined') {
		listen(document, 'visibilitychange', () => (document.hidden ? onHide() : onShow()));
		listen(window, 'pagehide', onHide);
		listen(window, 'pageshow', onShow);
	}

	function ensure(): AudioContext | null {
		if (disposed || !enabled) return null;
		if (!ctx) {
			try {
				const AC: typeof AudioContext | undefined =
					window.AudioContext ?? (window as unknown as { webkitAudioContext?: typeof AudioContext }).webkitAudioContext;
				if (!AC) return null;
				const c = new AC();
				ctx = c;
				master = c.createGain();
				master.gain.value = 0;
				const lim = c.createDynamicsCompressor();
				lim.threshold.value = -8;
				lim.knee.value = 8;
				lim.ratio.value = 10;
				lim.attack.value = 0.003;
				lim.release.value = 0.15;
				muffle = c.createBiquadFilter();
				muffle.type = 'lowpass';
				muffle.frequency.value = 18000;
				muffle.Q.value = 0.5;
				bus = c.createGain();
				bus.gain.value = 1;
				// reverb send: a generated decaying-noise impulse, wet level follows the scene
				const conv = c.createConvolver();
				const len = Math.floor(c.sampleRate * 1.8);
				const ir = c.createBuffer(2, len, c.sampleRate);
				for (let ch = 0; ch < 2; ch++) {
					const d = ir.getChannelData(ch);
					for (let i = 0; i < len; i++) d[i] = (Math.random() * 2 - 1) * Math.pow(1 - i / len, 2.6);
				}
				conv.buffer = ir;
				wet = c.createGain();
				wet.gain.value = 0.03;
				bus.connect(muffle);
				bus.connect(conv);
				conv.connect(wet);
				wet.connect(muffle);
				muffle.connect(lim).connect(master).connect(c.destination);
				noiseBuf = c.createBuffer(1, c.sampleRate, c.sampleRate);
				const nd = noiseBuf.getChannelData(0);
				for (let i = 0; i < nd.length; i++) nd[i] = Math.random() * 2 - 1;
				// boss drone: two detuned saws through a low lowpass, silent until needed
				droneG = c.createGain();
				droneG.gain.value = 0;
				const lp = c.createBiquadFilter();
				lp.type = 'lowpass';
				lp.frequency.value = 150;
				for (const f of [41.2, 43.0, 61.7]) {
					const o = c.createOscillator();
					o.type = 'sawtooth';
					o.frequency.value = f;
					o.connect(lp);
					o.start();
					droneNodes.push(o);
				}
				lp.connect(droneG).connect(bus);
				droneNodes.push(lfo(c, 0.13, 30, lp.frequency));
			} catch {
				ctx = null;
				return null;
			}
		}
		if (ctx.state !== 'running' && !(typeof document !== 'undefined' && document.hidden)) ctx.resume().catch(() => {});
		applyVolume();
		applyScene();
		startTimer();
		return ctx;
	}
	function lfo(c: AudioContext, f: number, depth: number, param: AudioParam): AudioScheduledSourceNode {
		const o = c.createOscillator();
		o.frequency.value = f;
		const g = c.createGain();
		g.gain.value = depth;
		o.connect(g).connect(param);
		o.start();
		return o;
	}
	function applyVolume() {
		if (!ctx || !master) return;
		master.gain.setTargetAtTime(enabled ? volume : 0, ctx.currentTime, 0.08);
	}
	function applyScene() {
		if (!ctx || !muffle || !wet || !droneG) return;
		const t = ctx.currentTime;
		const cut = scene.underwater ? 420 : scene.deaf ? 650 : 18000;
		muffle.frequency.setTargetAtTime(cut, t, 0.25);
		wet.gain.setTargetAtTime(scene.underwater ? 0.35 : scene.cave ? 0.5 : scene.indoors ? 0.2 : 0.03, t, 0.4);
		const d = scene.boss ? 0.07 + scene.tension * 0.04 : scene.tension > 0.7 ? (scene.tension - 0.7) * 0.1 : 0;
		droneG.gain.setTargetAtTime(enabled ? d : 0, t, 1.2);
	}
	function startTimer() {
		if (timer !== null || disposed || !enabled) return;
		timer = setInterval(tick, 100);
	}
	function stopTimer() {
		if (timer !== null) clearInterval(timer);
		timer = null;
	}
	function tick() {
		if (!ctx || ctx.state !== 'running' || !enabled) return;
		// heartbeat: below 35% HP, faster and louder as HP drops
		if (!reduce && scene.hp < 0.35 && scene.hp > 0) {
			const now = ctx.currentTime;
			if (nextBeat < now) nextBeat = now + 0.05;
			if (nextBeat < now + 0.25) {
				const sev = 1 - scene.hp / 0.35;
				const a = 0.14 + sev * 0.2;
				thump(nextBeat, a);
				thump(nextBeat + 0.16, a * 0.7);
				nextBeat += 0.45 + scene.hp * 1.6;
			}
		}
	}

	// ---- synth primitives (all start at `t`, auto-disconnect) ----
	function tone(type: OscillatorType, f0: number, f1: number, dur: number, peak: number, delay = 0, t0?: number) {
		const c = ctx, b = bus;
		if (!c || !b) return;
		const t = (t0 ?? c.currentTime) + delay;
		const o = c.createOscillator();
		const g = c.createGain();
		o.type = type;
		o.frequency.setValueAtTime(f0, t);
		o.frequency.exponentialRampToValueAtTime(Math.max(20, f1), t + dur);
		g.gain.setValueAtTime(0.0001, t);
		g.gain.linearRampToValueAtTime(peak, t + Math.min(0.01, dur * 0.2));
		g.gain.exponentialRampToValueAtTime(0.0005, t + dur);
		o.connect(g).connect(b);
		o.start(t);
		o.stop(t + dur + 0.02);
		o.onended = () => { o.disconnect(); g.disconnect(); };
	}
	function noise(dur: number, type: BiquadFilterType, f0: number, f1: number, q: number, peak: number, delay = 0, attack = 0.004) {
		const c = ctx, b = bus;
		if (!c || !b || !noiseBuf) return;
		const t = c.currentTime + delay;
		const s = c.createBufferSource();
		s.buffer = noiseBuf;
		s.loop = true;
		const f = c.createBiquadFilter();
		f.type = type;
		f.Q.value = q;
		f.frequency.setValueAtTime(f0, t);
		if (f1 !== f0) f.frequency.exponentialRampToValueAtTime(Math.max(20, f1), t + dur);
		const g = c.createGain();
		g.gain.setValueAtTime(0.0001, t);
		g.gain.linearRampToValueAtTime(peak, t + Math.min(attack, dur * 0.4));
		g.gain.exponentialRampToValueAtTime(0.0005, t + dur);
		s.connect(f).connect(g).connect(b);
		s.start(t, Math.random() * 0.8);
		s.stop(t + dur + 0.02);
		s.onended = () => { s.disconnect(); f.disconnect(); g.disconnect(); };
	}
	function thump(t: number, a: number) {
		tone('sine', 62, 38, 0.14, a, 0, t);
	}

	// ---- recipes ----
	const STEP: Record<string, () => void> = {
		grass: () => noise(0.09, 'bandpass', pv(1500), 900, 0.8, 0.07),
		stone: () => { noise(0.05, 'bandpass', pv(2200), 1600, 1.4, 0.09); tone('sine', pv(150), 90, 0.06, 0.06); },
		wood: () => { tone('triangle', pv(210), 130, 0.08, 0.12); noise(0.04, 'bandpass', pv(900), 700, 1.2, 0.06); },
		sand: () => noise(0.14, 'highpass', pv(3200), 2500, 0.5, 0.05, 0, 0.03),
		snow: () => { noise(0.12, 'bandpass', pv(1100), 600, 0.7, 0.08, 0, 0.025); },
		water: () => { noise(0.16, 'lowpass', pv(1400), 500, 0.6, 0.1, 0, 0.02); tone('sine', pv(520), pv(900), 0.08, 0.05, 0.03); },
		metal: () => { tone('square', pv(780), 700, 0.05, 0.05); noise(0.05, 'bandpass', pv(3400), 3000, 3, 0.09); tone('sine', pv(1320), 1250, 0.18, 0.04, 0.01); },
	};
	const HIT: Record<string, () => void> = {
		blade: () => { noise(0.1, 'bandpass', pv(2400), 1500, 1.6, 0.3); tone('triangle', pv(220), 90, 0.1, 0.22); },
		blunt: () => { noise(0.08, 'lowpass', pv(380), 200, 1, 0.45); tone('sine', pv(120), 45, 0.14, 0.45); },
		pierce: () => { noise(0.05, 'highpass', pv(3000), 3000, 1, 0.25); tone('triangle', pv(300), 110, 0.07, 0.25); },
	};
	const ELEM: Record<string, () => void> = {
		fire: () => { noise(0.3, 'bandpass', 600, 2400, 1.2, 0.22, 0, 0.1); tone('sawtooth', pv(180), 360, 0.26, 0.12); },
		frost: () => { tone('sine', pv(1760), 2400, 0.3, 0.12); tone('sine', pv(2300), 3100, 0.26, 0.08, 0.05); noise(0.2, 'highpass', 5000, 6500, 1, 0.07); },
		shock: () => { noise(0.18, 'bandpass', pv(3000), 800, 4, 0.3); tone('sawtooth', pv(90), 900, 0.12, 0.14); },
		acid: () => { noise(0.3, 'bandpass', pv(1800), 700, 2, 0.16, 0, 0.05); tone('sine', 600, 200, 0.28, 0.08); },
		poison: () => { tone('sine', pv(280), 170, 0.32, 0.16); tone('sine', pv(284), 168, 0.32, 0.12, 0.02); noise(0.2, 'lowpass', 500, 250, 1, 0.08); },
		holy: () => { for (const [i, f] of [523.25, 659.25, 783.99].entries()) tone('sine', pv(f, 0.01), f * 1.002, 0.4, 0.12, i * 0.06); },
		dark: () => { tone('sawtooth', pv(110), 55, 0.4, 0.14); tone('sine', pv(73), 40, 0.45, 0.18); noise(0.3, 'lowpass', 300, 120, 1, 0.1, 0, 0.1); },
		arcane: () => { tone('sine', pv(360), 920, 0.22, 0.22); tone('triangle', pv(720), 480, 0.24, 0.14, 0.04); },
	};

	function ui(kind: string) {
		switch (kind) {
			case 'move': tone('triangle', pv(540, 0.03), 520, 0.035, 0.1); break;
			case 'select': tone('sine', pv(620, 0.02), 880, 0.06, 0.16); tone('sine', 1180, 1240, 0.05, 0.08, 0.04); break;
			case 'back': tone('sine', pv(560, 0.02), 380, 0.07, 0.14); break;
			case 'error': tone('square', 170, 150, 0.1, 0.12); tone('square', 140, 120, 0.12, 0.12, 0.09); break;
		}
	}

	function dispatch(id: string) {
		const [head, arg = ''] = id.split(':');
		switch (head) {
			case 'step': {
				const now = performance.now();
				if (now - lastStep < 55) return;
				lastStep = now;
				(STEP[arg] ?? STEP.stone)();
				return;
			}
			case 'door': tone('triangle', pv(140), 85, 0.2, 0.2); noise(0.22, 'bandpass', pv(500), 320, 1.5, 0.13, 0, 0.06); tone('sine', pv(90), 70, 0.12, 0.1, 0.2); return;
			case 'doorclose': noise(0.12, 'lowpass', pv(500), 220, 1, 0.3); tone('sine', pv(100), 55, 0.16, 0.3); return;
			case 'pickup': tone('sine', pv(740), 1100, 0.07, 0.14); tone('sine', pv(990), 1300, 0.08, 0.1, 0.05); noise(0.04, 'highpass', 4000, 4000, 1, 0.05); return;
			case 'coin': tone('sine', pv(1320), 1325, 0.09, 0.2); tone('sine', pv(1760), 1765, 0.2, 0.22, 0.07); tone('triangle', 3520, 3500, 0.12, 0.04, 0.07); return;
			case 'equip': noise(0.09, 'bandpass', pv(2800), 1800, 3, 0.2); tone('square', pv(420), 300, 0.05, 0.08); tone('sine', pv(960), 900, 0.14, 0.07, 0.03); return;
			case 'eat': for (let i = 0; i < 3; i++) noise(0.05, 'bandpass', pv(1400), 800, 1, 0.14, i * 0.1); return;
			case 'drink': for (let i = 0; i < 3; i++) { tone('sine', pv(380), pv(620), 0.08, 0.12, i * 0.11); noise(0.06, 'lowpass', 800, 400, 1, 0.05, i * 0.11); } return;
			case 'levelup': for (const [i, f] of [440, 554.37, 659.25, 880, 1108.7].entries()) tone('triangle', pv(f, 0.005), f * 1.003, 0.2, 0.2, i * 0.08); return;
			case 'quest': for (const [i, f] of [392, 523.25, 659.25, 783.99].entries()) { tone('triangle', f, f * 1.002, 0.35, 0.16, i * 0.12); tone('sine', f * 2, f * 2, 0.3, 0.05, i * 0.12); } return;
			case 'hit': (HIT[arg] ?? HIT.blade)(); return;
			case 'shot': if (arg === 'gun') { noise(0.22, 'lowpass', 2600, 400, 0.8, 0.6, 0, 0.002); tone('sine', 130, 40, 0.18, 0.5); noise(0.5, 'bandpass', 700, 300, 0.5, 0.12, 0.06, 0.05); } else { tone('triangle', pv(460), 170, 0.12, 0.24); noise(0.08, 'highpass', 1800, 1800, 1.5, 0.18); noise(0.1, 'bandpass', 2500, 1200, 2, 0.14, 0.12); } return;
			case 'miss': noise(0.16, 'bandpass', pv(900), 2400, 1.4, 0.2, 0, 0.07); return;
			case 'block': tone('square', pv(980), 620, 0.09, 0.2); noise(0.06, 'bandpass', 2600, 2600, 4, 0.28); tone('sine', 1500, 1480, 0.25, 0.05, 0.02); return;
			case 'parry': noise(0.05, 'bandpass', 3200, 3200, 5, 0.35); tone('sine', pv(1760), 1740, 0.3, 0.1); tone('square', pv(1250), 900, 0.07, 0.1); return;
			case 'crit': HIT.blade(); noise(0.14, 'bandpass', 1400, 1400, 2.4, 0.3); tone('sawtooth', 520, 140, 0.16, 0.22); tone('sine', 880, 1320, 0.12, 0.18, 0.02); return;
			case 'hurt': tone('sawtooth', pv(200), 80, 0.16, 0.2); noise(0.1, 'lowpass', 600, 300, 1, 0.3); return;
			case 'death': tone('sawtooth', 220, 40, 0.9, 0.22); tone('sine', 110, 30, 1.1, 0.3); noise(0.9, 'lowpass', 700, 80, 1, 0.14, 0, 0.1); return;
			case 'cast': (ELEM[arg] ?? ELEM.arcane)(); return;
			case 'ui': ui(arg); return;
			case 'page': noise(0.18, 'bandpass', pv(3500), 1500, 0.8, 0.12, 0, 0.05); noise(0.1, 'highpass', 5500, 5500, 1, 0.06, 0.12); return;
			case 'lock': tone('square', pv(1400), 1300, 0.03, 0.12); tone('square', pv(900), 800, 0.04, 0.14, 0.07); noise(0.03, 'highpass', 4500, 4500, 2, 0.1, 0.07); return;
			case 'alarm': for (let i = 0; i < 4; i++) { tone('square', 880, 880, 0.12, 0.13, i * 0.24); tone('square', 660, 660, 0.12, 0.13, i * 0.24 + 0.12); } return;
		}
	}

	function play(id: string) {
		if (disposed || !enabled || !id) return;
		if (typeof document !== 'undefined' && document.hidden) return;
		if (!ensure()) return;
		try { dispatch(id); } catch { /* a failed voice must never break input */ }
	}

	return {
		play,
		handleFrame(f: QudFrame, key?: string) {
			if (disposed) return;
			const fx = f as FrameEx;
			scene = sceneFor(fx);
			if (enabled) {
				if (ctx) applyScene();
				let ev = Array.isArray(fx.sfx) ? fx.sfx : inferFromDiff(prevFrame, fx, key);
				if (foleyCovers && (fx as { foley?: unknown[] }).foley?.length) ev = ev.filter((e) => !foleyCovers!.has(e));
				// UI feedback in menus still comes from the shell: the engine reports world events only
				if (Array.isArray(fx.sfx) && prevFrame && fx.mode !== 'main' && fx.mode !== 'world' && fx.mode !== 'dead' && key) {
					ev.push(...inferFromDiff(prevFrame, fx, key).filter((e) => e.startsWith('ui:')));
				}
				for (const e of ev) play(e);
			}
			prevFrame = fx;
		},
		setScene(s: Partial<SceneState>) {
			scene = { ...scene, ...s };
			if (ctx) applyScene();
		},
		setVolume(v: number) {
			volume = clamp01(Number.isFinite(v) ? v : 0.5);
			applyVolume();
		},
		setEnabled(on: boolean) {
			if (disposed) return;
			enabled = on;
			if (on) {
				if (suspendT !== null) { clearTimeout(suspendT); suspendT = null; }
				if (ctx) { applyVolume(); applyScene(); startTimer(); if (!document.hidden) ctx.resume().catch(() => {}); }
			} else {
				stopTimer();
				applyVolume();
				if (droneG && ctx) droneG.gain.setTargetAtTime(0, ctx.currentTime, 0.1);
				const c = ctx;
				if (suspendT !== null) clearTimeout(suspendT);
				if (c) suspendT = setTimeout(() => { suspendT = null; if (!enabled && c.state === 'running') c.suspend().catch(() => {}); }, 500);
			}
		},
		setReduceMotion(on: boolean) {
			reduce = on;
		},
		dispose() {
			disposed = true;
			stopTimer();
			if (suspendT !== null) clearTimeout(suspendT);
			for (const [t, ev, fn] of listeners) t.removeEventListener(ev, fn);
			listeners.length = 0;
			for (const n of droneNodes) { try { (n as OscillatorNode).stop(); } catch { /* stopped */ } n.disconnect(); }
			droneNodes = [];
			const c = ctx;
			ctx = null;
			master = muffle = wet = bus = droneG = null;
			if (c) c.close().catch(() => {});
		},
	};
}
