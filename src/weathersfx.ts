// Ambient weather sound effects, separate from the music bus and with their own volume.
// Pure WebAudio, no samples: looping noise beds through filters (rain hiss, wind, surf, ember bed) whose
// levels follow the weather, plus short scheduled events (droplets, distant rumbles, thunder on lightning,
// ash crackle, bird chirps, crickets, wave swells). Driven by frame.music: wx is only set outdoors by the
// engine, so silence underground and indoors comes for free.
import type { QudMusic } from './qud';

export interface SfxHandle {
	setState(m: QudMusic): void;
	setVolume(v: number): void;
	setEnabled(on: boolean): void;
	dispose(): void;
}

/** what should be audible, 0..1 per layer */
export interface SfxTargets {
	rain: number; // hiss bed + droplets
	storm: number; // distant rumbles
	wind: number; // howling band of noise
	sand: number; // brighter, grittier wind
	ash: number; // crackle + low ember bed
	birds: number;
	crickets: number;
	surf: number;
}

const ZERO: SfxTargets = { rain: 0, storm: 0, wind: 0, sand: 0, ash: 0, birds: 0, crickets: 0, surf: 0 };

const RAINY: Record<string, number> = { rain: 0.6, acidrain: 0.6, fallout: 0.45, sporerain: 0.5, ashrain: 0.55, storm: 1, tempest: 1 };

/** targetsFor maps the engine's music state onto layer levels. */
export function targetsFor(m: QudMusic | null): SfxTargets {
	const t: SfxTargets = { ...ZERO };
	if (!m || !m.wx || m.mood === 'dead' || m.mood === 'world') return t;
	const wx = m.wx;
	const calm = m.danger < 0.5;
	t.rain = RAINY[wx] ?? 0;
	if (wx === 'storm' || wx === 'tempest') { t.storm = 1; t.wind = 0.35; }
	if (wx === 'radstorm') { t.storm = 0.6; t.sand = 0.8; t.rain = 0.2; }
	if (wx === 'blizzard') t.wind = 1;
	if (wx === 'snow') t.wind = 0.25;
	if (wx === 'frost') t.wind = 0.12;
	if (wx === 'sandstorm') { t.sand = 1; t.wind = 0.4; }
	if (wx === 'ashfall' || wx === 'ashrain') { t.ash = 1; t.wind = Math.max(t.wind, 0.15); }
	const fair = wx === 'clear' || wx === 'overcast' || wx === 'pollen' || wx === 'heat';
	const season = m.season ?? 0;
	if (fair && calm && m.mood === 'wild' && m.day && (season === 0 || season === 1)) t.birds = wx === 'overcast' ? 0.5 : 1;
	if (fair && calm && (m.mood === 'wild' || m.mood === 'town') && !m.day && season === 1) t.crickets = m.mood === 'town' ? 0.5 : 1;
	if (m.shore || m.mood === 'sea') t.surf = t.storm > 0 || t.wind > 0.5 ? 1 : 0.7;
	return t;
}

/** per-layer loudness at full target, tuned by offline render to sit under the music */
const LEVEL = { rain: 0.16, drop: 0.12, wind: 0.6, sand: 0.18, ash: 0.06, crack: 0.1, rumble: 0.3, thunder: 0.45, bird: 0.08, cricket: 0.04, surf: 0.3 };

export interface SfxCore {
	setTargets(t: SfxTargets, tau?: number): void;
	/** schedule events between now and now + ahead (seconds of context time) */
	tick(ahead: number): void;
	thunder(at?: number): void;
	output: GainNode;
	dispose(): void;
}

/** createSfxCore builds the graph on any (online or offline) context and connects it to dest. */
export function createSfxCore(c: BaseAudioContext, dest: AudioNode, rnd: () => number = Math.random): SfxCore {
	const out = c.createGain();
	out.gain.value = 1;
	const lim = c.createDynamicsCompressor();
	lim.threshold.value = -10;
	lim.knee.value = 6;
	lim.ratio.value = 12;
	lim.attack.value = 0.003;
	lim.release.value = 0.2;
	out.connect(lim).connect(dest);

	// 4 s of white noise and of brown noise, looped with random offsets
	const len = Math.floor(c.sampleRate * 4);
	const white = c.createBuffer(1, len, c.sampleRate);
	const brown = c.createBuffer(1, len, c.sampleRate);
	{
		const w = white.getChannelData(0), b = brown.getChannelData(0);
		let last = 0;
		for (let i = 0; i < len; i++) {
			const x = rnd() * 2 - 1;
			w[i] = x;
			last = (last + 0.02 * x) / 1.02;
			b[i] = last * 3.5;
		}
	}
	const nodes: AudioNode[] = [];
	const sources: AudioScheduledSourceNode[] = [];
	function loop(buf: AudioBuffer, rate = 1): AudioBufferSourceNode {
		const s = c.createBufferSource();
		s.buffer = buf;
		s.loop = true;
		s.playbackRate.value = rate;
		s.start(c.currentTime, rnd() * 3.5);
		sources.push(s);
		return s;
	}
	function filt(type: BiquadFilterType, f: number, q = 0.7): BiquadFilterNode {
		const b = c.createBiquadFilter();
		b.type = type;
		b.frequency.value = f;
		b.Q.value = q;
		nodes.push(b);
		return b;
	}
	function gain(v = 0): GainNode {
		const g = c.createGain();
		g.gain.value = v;
		nodes.push(g);
		return g;
	}
	function lfo(freq: number, depth: number, param: AudioParam) {
		const o = c.createOscillator();
		o.frequency.value = freq;
		const g = gain(depth);
		o.connect(g).connect(param);
		o.start();
		sources.push(o);
	}

	// ---- beds ----
	const rainG = gain();
	loop(white).connect(filt('highpass', 500)).connect(filt('lowpass', 7000)).connect(rainG).connect(out);
	const windG = gain();
	const windF = filt('bandpass', 520, 2.2);
	lfo(0.07, 260, windF.frequency);
	const windSwell = gain(0.75);
	lfo(0.11, 0.25, windSwell.gain);
	loop(brown).connect(windF).connect(windSwell).connect(windG).connect(out);
	const sandG = gain();
	const sandF = filt('bandpass', 2400, 0.9);
	lfo(0.13, 700, sandF.frequency);
	loop(white, 0.8).connect(sandF).connect(sandG).connect(out);
	const ashG = gain();
	loop(brown, 0.6).connect(filt('lowpass', 300)).connect(ashG).connect(out);
	const surfG = gain();
	const surfSwell = gain(0.55);
	lfo(0.09, 0.45, surfSwell.gain); // waves roll in about every 11 s
	const surfF = filt('lowpass', 900, 0.5);
	lfo(0.09, 450, surfF.frequency);
	loop(white).connect(surfF).connect(surfSwell).connect(surfG).connect(out);
	// event buses (their level is the layer target too, so events fade with the weather)
	const dropBus = gain();
	dropBus.connect(out);
	const rumbleBus = gain();
	rumbleBus.connect(out);
	const crackBus = gain();
	crackBus.connect(out);
	const birdBus = gain();
	birdBus.connect(out);
	const crickBus = gain();
	crickBus.connect(out);
	const thunderBus = gain(1);
	thunderBus.connect(out);

	let tg: SfxTargets = { ...ZERO };
	const next = { drop: 0, rumble: 0, crack: 0, bird: 0, cricket: 0 };

	function setTargets(t: SfxTargets, tau = 1.2) {
		tg = t;
		const now = c.currentTime;
		const to = (p: AudioParam, v: number) => {
			if (tau <= 0) p.setValueAtTime(v, now);
			else p.setTargetAtTime(v, now, tau);
		};
		to(rainG.gain, t.rain * LEVEL.rain);
		to(dropBus.gain, t.rain * LEVEL.drop);
		to(windG.gain, t.wind * LEVEL.wind);
		to(sandG.gain, t.sand * LEVEL.sand);
		to(ashG.gain, t.ash * LEVEL.ash);
		to(crackBus.gain, t.ash * LEVEL.crack);
		to(rumbleBus.gain, t.storm * LEVEL.rumble);
		to(birdBus.gain, t.birds * LEVEL.bird);
		to(crickBus.gain, t.crickets * LEVEL.cricket);
		to(surfG.gain, t.surf * LEVEL.surf);
	}

	// ---- one-shot voices ----
	function burst(bus: AudioNode, t: number, dur: number, type: BiquadFilterType, f: number, q: number, peak: number, buf = white, rate = 1) {
		const s = c.createBufferSource();
		s.buffer = buf;
		s.playbackRate.value = rate;
		const b = c.createBiquadFilter();
		b.type = type;
		b.frequency.value = f;
		b.Q.value = q;
		const g = c.createGain();
		g.gain.setValueAtTime(0, t);
		g.gain.linearRampToValueAtTime(peak, t + Math.min(0.004, dur / 4));
		g.gain.exponentialRampToValueAtTime(0.0005, t + dur);
		s.connect(b).connect(g).connect(bus);
		s.start(t, rnd() * 3);
		s.stop(t + dur + 0.05);
		s.onended = () => { s.disconnect(); b.disconnect(); g.disconnect(); };
	}
	function drop(t: number) {
		burst(dropBus, t, 0.012 + rnd() * 0.02, 'bandpass', 1500 + rnd() * 4000, 6, 0.4 + rnd() * 0.6);
	}
	function rumble(t: number) {
		const dur = 3 + rnd() * 3;
		const s = c.createBufferSource();
		s.buffer = brown;
		s.loop = true;
		s.playbackRate.value = 0.5 + rnd() * 0.3;
		const b = c.createBiquadFilter();
		b.type = 'lowpass';
		b.frequency.value = 110 + rnd() * 60;
		const g = c.createGain();
		g.gain.setValueAtTime(0, t);
		g.gain.linearRampToValueAtTime(0.5 + rnd() * 0.5, t + 0.8 + rnd());
		g.gain.exponentialRampToValueAtTime(0.001, t + dur);
		s.connect(b).connect(g).connect(rumbleBus);
		s.start(t, rnd() * 3);
		s.stop(t + dur + 0.1);
		s.onended = () => { s.disconnect(); b.disconnect(); g.disconnect(); };
	}
	function thunder(at?: number) {
		const t = at ?? c.currentTime + 0.05 + rnd() * 0.3;
		// the crack, then a long rolling rumble
		burst(thunderBus, t, 0.35, 'highpass', 900, 0.5, LEVEL.thunder * 0.6);
		const dur = 4 + rnd() * 2;
		const s = c.createBufferSource();
		s.buffer = brown;
		s.loop = true;
		s.playbackRate.value = 0.45;
		const b = c.createBiquadFilter();
		b.type = 'lowpass';
		b.frequency.setValueAtTime(900, t);
		b.frequency.exponentialRampToValueAtTime(140, t + 1.5);
		const g = c.createGain();
		g.gain.setValueAtTime(0, t);
		g.gain.linearRampToValueAtTime(LEVEL.thunder, t + 0.06);
		g.gain.setTargetAtTime(LEVEL.thunder * 0.4, t + 0.3, 0.4);
		g.gain.setTargetAtTime(0.0001, t + 1.6, 0.9);
		s.connect(b).connect(g).connect(thunderBus);
		s.start(t, rnd() * 3);
		s.stop(t + dur);
		s.onended = () => { s.disconnect(); b.disconnect(); g.disconnect(); };
	}
	function crack(t: number) {
		const n = 1 + Math.floor(rnd() * 3);
		for (let i = 0; i < n; i++) burst(crackBus, t + i * (0.01 + rnd() * 0.03), 0.004 + rnd() * 0.01, 'highpass', 1800 + rnd() * 2500, 0.8, 0.5 + rnd() * 0.5);
	}
	function tone(bus: AudioNode, t: number, dur: number, f0: number, f1: number, peak: number) {
		const o = c.createOscillator();
		o.type = 'sine';
		o.frequency.setValueAtTime(f0, t);
		o.frequency.exponentialRampToValueAtTime(f1, t + dur);
		const g = c.createGain();
		g.gain.setValueAtTime(0, t);
		g.gain.linearRampToValueAtTime(peak, t + dur * 0.2);
		g.gain.exponentialRampToValueAtTime(0.0005, t + dur);
		o.connect(g).connect(bus);
		o.start(t);
		o.stop(t + dur + 0.02);
		o.onended = () => { o.disconnect(); g.disconnect(); };
	}
	function bird(t: number) {
		// a short phrase of 2-5 chirps from one bird: its own pitch and contour
		const base = 2400 + rnd() * 2600;
		const up = rnd() < 0.5;
		const n = 2 + Math.floor(rnd() * 4);
		let at = t;
		for (let i = 0; i < n; i++) {
			const d = 0.05 + rnd() * 0.08;
			const f = base * (1 + (rnd() - 0.5) * 0.15);
			tone(birdBus, at, d, up ? f * 0.8 : f * 1.25, up ? f * 1.25 : f * 0.8, 0.6 + rnd() * 0.4);
			at += d + 0.03 + rnd() * 0.1;
		}
	}
	function cricket(t: number) {
		// a cricket chirp: 3-4 quick pulses of a ~4.5 kHz tone
		const f = 4200 + rnd() * 700;
		const n = 3 + Math.floor(rnd() * 2);
		for (let i = 0; i < n; i++) tone(crickBus, t + i * 0.032, 0.022, f, f * 0.98, 1);
	}

	function tick(ahead: number) {
		const now = c.currentTime;
		const end = now + ahead;
		const run = (k: keyof typeof next, on: boolean, gap: () => number, fn: (t: number) => void) => {
			if (!on) { next[k] = 0; return; }
			if (next[k] < now) next[k] = now + gap() * rnd();
			while (next[k] < end) {
				fn(next[k]);
				next[k] += gap();
			}
		};
		run('drop', tg.rain > 0.05, () => 0.02 + rnd() * 0.12 / Math.max(tg.rain, 0.2), drop);
		run('rumble', tg.storm > 0.05, () => 7 + rnd() * 14, rumble);
		run('crack', tg.ash > 0.05, () => 0.15 + rnd() * 0.9, crack);
		run('bird', tg.birds > 0.05, () => 1.5 + rnd() * 5, bird);
		run('cricket', tg.crickets > 0.05, () => 0.35 + rnd() * 0.5, cricket);
	}

	return {
		setTargets,
		tick,
		thunder,
		output: out,
		dispose() {
			for (const s of sources) { try { s.stop(); } catch { /* already stopped */ } s.disconnect(); }
			for (const n of nodes) n.disconnect();
			out.disconnect();
			lim.disconnect();
		},
	};
}

/** createWeatherSfx: the live layer for the game screen, on its own AudioContext (resumed on key presses). */
export function createWeatherSfx(): SfxHandle {
	let ctx: AudioContext | null = null;
	let core: SfxCore | null = null;
	let master: GainNode | null = null;
	let enabled = true;
	let volume = 0.5;
	let disposed = false;
	let lastBolt = -1;
	let state: QudMusic | null = null;
	let timer: ReturnType<typeof setInterval> | null = null;
	let suspendT: ReturnType<typeof setTimeout> | null = null;

	function ensure(): boolean {
		if (disposed) return false;
		if (!ctx) {
			try {
				const AC: typeof AudioContext | undefined =
					window.AudioContext ?? (window as unknown as { webkitAudioContext?: typeof AudioContext }).webkitAudioContext;
				if (!AC) return false;
				ctx = new AC();
				master = ctx.createGain();
				master.gain.value = 0;
				master.connect(ctx.destination);
				core = createSfxCore(ctx, master);
			} catch {
				ctx = null;
				return false;
			}
		}
		if (ctx.state !== 'running') ctx.resume().catch(() => {});
		applyVolume();
		return true;
	}
	function applyVolume() {
		if (!ctx || !master) return;
		master.gain.setTargetAtTime(enabled ? volume : 0, ctx.currentTime, 0.3);
	}
	function start() {
		if (timer !== null) return;
		timer = setInterval(() => core?.tick(0.4), 150);
	}
	function stop() {
		if (timer !== null) clearInterval(timer);
		timer = null;
	}
	return {
		setState(m: QudMusic) {
			if (disposed) return;
			state = m;
			const bolt = m.bolt ?? 0;
			const strike = lastBolt >= 0 && bolt > lastBolt;
			lastBolt = bolt;
			if (!enabled || !ensure()) return;
			core!.setTargets(targetsFor(m));
			if (strike) core!.thunder();
			start();
		},
		setVolume(v: number) {
			volume = Math.min(1, Math.max(0, Number.isFinite(v) ? v : 0.5));
			applyVolume();
		},
		setEnabled(on: boolean) {
			if (disposed) return;
			enabled = on;
			if (on) {
				if (suspendT !== null) { clearTimeout(suspendT); suspendT = null; }
				if (!state || !ensure()) return;
				core!.setTargets(targetsFor(state));
				start();
			} else {
				stop();
				applyVolume();
				const c = ctx;
				if (suspendT !== null) clearTimeout(suspendT);
				if (c) suspendT = setTimeout(() => { suspendT = null; if (!enabled && c.state === 'running') c.suspend().catch(() => {}); }, 600);
			}
		},
		dispose() {
			disposed = true;
			stop();
			if (suspendT !== null) clearTimeout(suspendT);
			core?.dispose();
			core = null;
			const c = ctx;
			ctx = null;
			if (c) c.close().catch(() => {});
		},
	};
}
