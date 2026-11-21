// Layered combat music and story-beat stingers on top of the base soundtrack (music.ts / garden.ts).
// Pure WebAudio on a context and output node given by the coordinator (nothing is created here but nodes).
//  - stems: a percussion layer (kit by culture profile / genre) that follows danger and hostile count, a low
//    drone + brass motif for bosses (the motif is a hash of the boss id, so the same boss always has it), and a
//    high tremolo string layer at low player HP;
//  - one-shots: a stinger when a fight starts, a short cadence when the last hostile dies or flees (not when the
//    player leaves the zone), and 1.5 to 4 s stingers for engine cues (engine/cueinfo.go);
//  - ducking: every stinger calls the duck callback (about 3 to 5 dB) and also drives `duckNode`.
// Tempo: music.ts keeps its bpm private, so the beat grid is FREE-RUNNING (default 100 bpm, 8th-note steps) unless
// the coordinator calls setTempo(bpm). Peaks (before the master, at volume 1): stems below -22 dBFS, stingers
// below -20 dBFS (each bus has its own limiter-like compressor). Nothing runs while disabled, hidden or disposed.
import type { QudMusic } from './qud';
import { INST, createKit, disposeKit, drone2, hit, sweep, tex, type Kit } from './instruments';
import { GENERIC, GENRE_COLOUR, PROFILES, SCALES, type Profile } from './scales';

export interface MusicExtras {
	hostiles: number;
	boss: boolean;
	bossId: number;
	hpFrac: number;
	inCombat: boolean;
	/** hash of the zone key (engine CueScene "zone"); optional, falls back to music seed/mood/depth */
	zone?: number;
}
export interface StingerCue { kind: string; intensity: number; genre?: string; sphere?: string }

export interface MusicLayers {
	/** call on every frame; `extras` comes from frame.cuex (engine CueScene), mapped 1:1 */
	setMusic(m: QudMusic | null, extras?: Partial<MusicExtras>): void;
	/** frame.cues, may be empty or undefined */
	cue(cues: StingerCue[] | null | undefined): void;
	/** duck the base music by `db` dB for `seconds` (also fires the onDuck callback) */
	duck(db: number, seconds: number): void;
	/** the coordinator's callback, invoked as a stinger starts: (db, seconds) */
	onDuck(cb: ((db: number, seconds: number) => void) | null): void;
	/** a unity gain node animated by duck(): route a same-context base music output through it */
	getDuckNode(): GainNode;
	/** optional: tell the layers the base music tempo; re-anchors the grid */
	setTempo(bpm: number): void;
	setEnabled(on: boolean): void;
	setVolume(v: number): void;
	/** pref musicLayers: the stems (percussion, boss, tension) */
	setLayers(on: boolean): void;
	/** pref stingers: fight-start, victory and cue stingers */
	setStingers(on: boolean): void;
	dispose(): void;
}

// ---------- helpers ----------
const clamp = (v: number, a: number, b: number) => Math.min(b, Math.max(a, v));
const mtof = (m: number) => 440 * Math.pow(2, (m - 69) / 12);
function hash(s: string): number {
	let h = 2166136261;
	for (let i = 0; i < s.length; i++) { h ^= s.charCodeAt(i); h = Math.imul(h, 16777619); }
	return h >>> 0;
}
function seeded(seed: number): () => number {
	let a = seed >>> 0;
	return () => {
		a = (a + 0x6d2b79f5) >>> 0;
		let t = a;
		t = Math.imul(t ^ (t >>> 15), t | 1);
		t ^= t + Math.imul(t ^ (t >>> 7), t | 61);
		return ((t ^ (t >>> 14)) >>> 0) / 4294967296;
	};
}

// ---------- musical identity of the current land ----------
interface Ident { prof: Profile; kit: string; style: 'war' | 'rattle'; scale: number[]; tonic: number; lead: string; root: number; seed: number }
const KIT_NAMES = new Set(['taiko', 'tabla', 'darbuka', 'djembe', 'bodhran', 'frame', 'gong', 'tincan', 'electro', 'brush', 'timpani']);

function identity(m: QudMusic | null): Ident {
	const prof = (m?.sphere && PROFILES[m.sphere]) || GENERIC;
	const genre = m?.genre ?? 'fantasy';
	const gc = GENRE_COLOUR[genre] ?? {};
	const seed = (m?.seed ?? 1) >>> 0;
	const h = hash(`${m?.sphere ?? ''}|${genre}|${seed}`);
	// kit: genre first for the machine / junk genres, else the culture's own kit
	let kit = prof.kit;
	let style: 'war' | 'rattle' = 'war';
	if (genre === 'cyberpunk') kit = 'electro';
	else if (genre === 'steampunk' || genre === 'dieselpunk' || genre === 'wasteland') kit = 'tincan';
	else if (genre === 'twilight') kit = 'timpani';
	else if (genre === 'biopunk') { kit = 'frame'; style = 'rattle'; } // bone rattles: only the noisy roles
	else if (genre === 'atompunk') kit = 'brush';
	else if (gc.kit && (gc.kitW ?? 0) >= 0.6 && !m?.sphere) kit = gc.kit;
	if (!KIT_NAMES.has(kit)) kit = 'frame';
	// scale: genre colour when it asks for one, else the culture's, else a minor mode
	let names: string[] = [];
	if (gc.scales && (h % 100) / 100 < (gc.scaleW ?? 0)) names = gc.scales;
	else if (prof.scales.length) names = prof.scales;
	else names = ['aeolian', 'dorian', 'phrygian'];
	const scale = SCALES[names[h % names.length]] ?? SCALES.aeolian;
	const tonic = 41 + (seed % 7); // F2..B2
	let lead = prof.leads[(h >>> 3) % prof.leads.length] ?? 'flute';
	if (gc.lead && (gc.leadW ?? 0) >= 0.45) lead = gc.lead;
	if (!INST[lead]) lead = 'lute';
	return { prof, kit, style, scale, tonic, lead, root: mtof(tonic), seed };
}
function degree(id: Ident, d: number, octaveBase = 0): number {
	const n = id.scale.length;
	const o = Math.floor(d / n);
	return id.tonic + octaveBase + id.scale[((d % n) + n) % n] + 12 * o;
}

/** the boss motif: 4 to 6 degrees + beat lengths, a pure function of the boss id */
export function bossMotif(bossId: number): { deg: number[]; beats: number[] } {
	const r = seeded(hash('bm' + bossId));
	const n = 4 + Math.floor(r() * 3);
	const deg: number[] = [0];
	for (let i = 1; i < n; i++) {
		const step = [-2, -1, 1, 2, 3][Math.floor(r() * 5)];
		deg.push(clamp(deg[i - 1] + step, -3, 6));
	}
	deg[n - 1] = [0, 4, -3][Math.floor(r() * 3)]; // ends on tonic, fifth or low tonic
	const beats = deg.map(() => [1, 1, 0.5, 1.5, 2][Math.floor(r() * 5)]);
	beats[n - 1] = 2;
	return { deg, beats };
}

// ---------- stinger recipes ----------
interface Ev { t: number; inst: string; midi: number; dur: number; vel: number }
interface Recipe { evs: Ev[]; drone?: { midi: number; dur: number; vel: number; type?: OscillatorType }[]; drums?: { t: number; kit: string; role: 0 | 1 | 2 | 3; vel: number }[]; tex?: { t: number; kind: 'hiss' | 'static'; vel: number }[]; len: number }

const PRIO: Record<string, number> = {
	death: 9, ending: 9, legend: 8, rift: 7, victory: 6, levelup: 6, quest: 6, relic: 6, wedding: 5, funeral: 5, ceremony: 4, prophecy: 5, omen: 5,
	discovery: 4, crossroads: 4, deadline: 4, fightstart: 7, city: 1, dungeon: 2, tower: 2,
};

function recipe(kind: string, inten: number, id: Ident): Recipe {
	const I = clamp(Math.round(inten) || 1, 1, 3);
	const v = [0, 0.6, 0.75, 0.9][I];
	const evs: Ev[] = [];
	const e = (t: number, inst: string, midi: number, dur: number, vel: number) => { if (evs.length < 12) evs.push({ t, inst, midi, dur, vel }); };
	const r: Recipe = { evs, len: 2.4 };
	const T = id.tonic + 24; // stinger register: two octaves over the drone tonic
	const prof = id.prof;
	const feastLead = prof.feastLeads?.[0] && INST[prof.feastLeads[0]] ? prof.feastLeads[0] : id.lead;
	const lament = prof.lament && INST[prof.lament] ? prof.lament : 'duduk';
	switch (kind) {
		case 'discovery': { // wonder: rising bell arpeggio over a soft pad
			const n = 4 + I;
			for (let i = 0; i < n; i++) e(i * 0.2, 'bell', degree(id, [0, 2, 4, 7, 9, 11][i], 24), 1.2, v * (0.6 + i * 0.08));
			e(0, 'sine', T - 12, 2.6, v * 0.5);
			r.len = n * 0.2 + 1.6; break;
		}
		case 'levelup': { // bright: major pentatonic, regardless of the land
			const mp = SCALES.majPent;
			const m = (k: number) => T + mp[k % 5] + 12 * Math.floor(k / 5);
			[0, 2, 4, 5, 7].slice(0, 3 + I).forEach((k, i) => e(i * 0.14, i % 2 ? 'marimba' : 'bell', m(k), 0.9, v * 0.8));
			e(0.6, 'harp', m(7), 1.4, v * 0.7);
			r.len = 2.2; break;
		}
		case 'quest': { // short resolved fanfare on the land's lead
			e(0, id.lead, degree(id, 0, 12), 0.35, v * 0.8);
			e(0.3, id.lead, degree(id, 4, 12), 0.35, v * 0.8);
			e(0.6, id.lead, degree(id, 7, 12), 1.5, v);
			e(0.6, 'harp', degree(id, 0, 12), 1.8, v * 0.6);
			r.len = 2.4; break;
		}
		case 'relic': { // glitter: high, scattered bells
			const rn = seeded(hash('relic' + id.seed));
			for (let i = 0; i < 4 + I; i++) e(i * 0.17 + rn() * 0.05, i % 2 ? 'bell' : 'musicbox', degree(id, Math.floor(rn() * 8) + 3, 24), 1.0, v * 0.6);
			e(0, 'sine', T - 12, 2.2, v * 0.4);
			r.len = 2.6; break;
		}
		case 'legend': { // fanfare: brass over timpani
			const d = [0, 0, 4, 7, 4, 7, 11];
			const t0 = [0, 0.28, 0.56, 0.84, 1.4, 1.68, 2.1];
			d.slice(0, 4 + I).forEach((k, i) => e(t0[i], 'brass', degree(id, k, 12), i === d.slice(0, 4 + I).length - 1 ? 1.6 : 0.4, v));
			r.drums = [{ t: 0, kit: 'timpani', role: 3, vel: v * 0.9 }, { t: 0.56, kit: 'timpani', role: 0, vel: v * 0.7 }];
			r.len = 3.4; break;
		}
		case 'wedding': { // lilting folk phrase: long-short pairs
			const d = [4, 2, 0, 2, 4, 5, 4, 2, 0];
			let t = 0;
			d.slice(0, 6 + I).forEach((k, i) => { e(t, i % 2 ? feastLead : feastLead, degree(id, k, 12), 0.5, v * 0.75); t += i % 2 ? 0.16 : 0.32; });
			e(0, prof.plucks[0] && INST[prof.plucks[0]] ? prof.plucks[0] : 'harp', degree(id, 0, 0), 1.8, v * 0.6);
			e(t, 'harp', degree(id, 0, 12), 1.0, v * 0.6);
			r.len = t + 1.2; break;
		}
		case 'ceremony': {
			e(0, id.lead, degree(id, 0, 12), 1.0, v * 0.7); e(0.8, id.lead, degree(id, 4, 12), 1.4, v * 0.7); e(0, 'sine', T - 12, 2.2, v * 0.4);
			r.len = 2.4; break;
		}
		case 'funeral': { // lament: slow falling line
			const d = [4, 3, 2, 1, 0];
			d.slice(0, 3 + I).forEach((k, i) => e(i * 0.7, lament, degree(id, k, 12), 1.0, v * 0.75));
			r.drone = [{ midi: id.tonic, dur: 3.6, vel: 0.2 * v }];
			r.len = 3.6; break;
		}
		case 'omen': { // eerie shimmer: a sliding theremin over a bell cluster and hiss
			e(0, 'theremin', degree(id, 4, 24) + 0.5, 2.6, v * 0.5);
			e(0.4, 'bell', degree(id, 1, 24) + 0.4, 1.8, v * 0.35);
			e(0.9, 'bell', degree(id, 3, 24) - 0.4, 1.8, v * 0.3);
			r.drone = [{ midi: id.tonic, dur: 3.2, vel: 0.15 * v }];
			r.tex = [{ t: 0.2, kind: 'hiss', vel: 0.05 }];
			r.len = 3.2; break;
		}
		case 'prophecy': {
			e(0, 'choir', degree(id, 0, 12), 3.0, v * 0.5); e(0.2, 'choir', degree(id, 4, 12), 2.8, v * 0.4); e(1.2, 'bell', degree(id, 7, 24), 1.8, v * 0.5);
			r.tex = [{ t: 0.3, kind: 'hiss', vel: 0.04 }];
			r.len = 3.6; break;
		}
		case 'crossroads': { // an unresolved question: up to the fifth and hold
			e(0, id.lead, degree(id, 0, 12), 0.6, v * 0.7); e(0.5, id.lead, degree(id, 3, 12), 0.6, v * 0.7); e(1.0, id.lead, degree(id, 4, 12), 1.8, v * 0.7);
			r.len = 2.8; break;
		}
		case 'deadline': case 'fightstart': { // dread: a low cluster
			r.drone = [{ midi: id.tonic - 5, dur: 2.2, vel: 0.22 * v, type: 'sawtooth' }, { midi: id.tonic - 4 + 0.35, dur: 2.2, vel: 0.18 * v, type: 'sawtooth' }, { midi: id.tonic + 1, dur: 2.2, vel: 0.14 * v, type: 'triangle' }];
			r.drums = [{ t: 0.05, kit: kind === 'fightstart' ? id.kit : 'timpani', role: kind === 'fightstart' ? 3 : 0, vel: v * 0.8 }];
			if (kind === 'fightstart') e(0.05, 'brass', id.tonic + 12, 0.8, v * 0.55);
			r.len = 2.2; break;
		}
		case 'rift': { // a world turns: ascending whole-tone bells with a sweep
			const w = SCALES.wholetone;
			for (let i = 0; i < 6; i++) e(i * 0.22, 'bell', T + w[i % 6] + 12 * Math.floor(i / 6), 1.4, v * (0.5 + i * 0.07));
			e(0, 'sine', T - 12, 3.0, v * 0.5);
			r.tex = [{ t: 0, kind: 'hiss', vel: 0.05 }];
			r.len = 3.4; break;
		}
		case 'city': { e(0, 'harp', degree(id, 0, 12), 0.8, v * 0.7); e(0.25, 'harp', degree(id, 2, 12), 0.8, v * 0.7); e(0.5, 'harp', degree(id, 4, 12), 1.2, v * 0.7); r.len = 1.8; break; }
		case 'tower': { e(0, 'bell', degree(id, 0, 24), 1.2, v * 0.6); e(0.35, 'bell', degree(id, 4, 24), 1.6, v * 0.6); r.len = 2.0; break; }
		case 'dungeon': {
			r.drone = [{ midi: id.tonic - 5, dur: 2.0, vel: 0.2 * v }];
			r.drums = [{ t: 0.1, kit: 'timpani', role: 0, vel: v * 0.6 }];
			r.len = 2.0; break;
		}
		case 'death': { // a falling line, then a low sigh
			[7, 6, 4, 3, 2, 0].forEach((k, i) => e(i * 0.5, 'duduk', degree(id, k, 12), 0.9, v * (0.8 - i * 0.07)));
			r.drone = [{ midi: id.tonic - 12, dur: 3.8, vel: 0.2 * v }];
			r.len = 4.0; break;
		}
		case 'ending': { // a grand held chord with bells
			e(0, 'choir', degree(id, 0, 12), 3.8, v * 0.5); e(0, 'choir', degree(id, 4, 12), 3.8, v * 0.4); e(0.1, 'brass', degree(id, 0, 0), 3.0, v * 0.5);
			[0, 2, 4, 7].forEach((k, i) => e(0.5 + i * 0.35, 'bell', degree(id, k, 24), 1.8, v * 0.55));
			r.len = 4.0; break;
		}
		case 'victory': { // V then I cadence
			e(0, id.lead, degree(id, 4, 12), 0.4, v * 0.75);
			e(0.45, id.lead, degree(id, 0, 12), 1.1, v * 0.85);
			e(0.45, 'harp', degree(id, 0, 0), 1.4, v * 0.6);
			r.drums = [{ t: 0.45, kit: id.kit, role: 3, vel: v * 0.6 }];
			r.len = 1.8; break;
		}
		default: return { evs: [], len: 0 };
	}
	return r;
}

// ---------- the layers ----------
export function createMusicLayers(ctx: AudioContext, dest: AudioNode, opts?: { onDuck?: (db: number, seconds: number) => void }): MusicLayers {
	const kit: Kit = createKit(ctx);
	let enabled = true, layersOn = true, stingersOn = true, disposed = false;
	let volume = 1;
	let duckCb = opts?.onDuck ?? null;

	// graph: stemBus -> stemComp -> master ; stingBus -> stingComp -> master ; master -> dest
	const master = ctx.createGain();
	master.gain.value = 0;
	master.gain.setTargetAtTime(volume, ctx.currentTime, 0.6); // fades in over ~1.5 s
	master.connect(dest);
	const lim = (thr: number) => {
		const c = ctx.createDynamicsCompressor();
		c.threshold.value = thr; c.knee.value = 3; c.ratio.value = 14; c.attack.value = 0.002; c.release.value = 0.2;
		return c;
	};
	const stemBus = ctx.createGain(); stemBus.gain.value = 0.12;
	const stemComp = lim(-26);
	stemBus.connect(stemComp).connect(master);
	const stingBus = ctx.createGain(); stingBus.gain.value = 0.09;
	const stingComp = lim(-24);
	stingBus.connect(stingComp).connect(master);
	const duckNode = ctx.createGain();

	// ----- grid (free-running unless setTempo is called) -----
	let bpm = 100;
	let epoch = ctx.currentTime;
	const beat = () => 60 / bpm;
	const quant = (t: number, sub = 0.5) => {
		const g = beat() * sub;
		return epoch + Math.ceil((t - epoch) / g - 1e-6) * g;
	};

	// ----- state -----
	let music: QudMusic | null = null;
	let ex: MusicExtras = { hostiles: 0, boss: false, bossId: 0, hpFrac: 1, inCombat: false };
	let id = identity(null);
	let idKey = '';
	let percEnv = 0, bossEnv = 0, tensEnv = 0;
	let nextStep = 0, bossNext = 0, bossIdPlaying = 0;
	let lastTick = 0;
	let timer: ReturnType<typeof setInterval> | null = null;
	let sweepT: ReturnType<typeof setTimeout> | null = null;
	let stingT: ReturnType<typeof setTimeout> | null = null;

	// tension voice (persistent while active)
	let tens: { o: OscillatorNode[]; lfo: OscillatorNode; g: GainNode; nodes: AudioNode[] } | null = null;
	let tensStopT: ReturnType<typeof setTimeout> | null = null;

	function startTension() {
		if (tens || disposed) return;
		const nodes: AudioNode[] = [];
		const g = ctx.createGain(); g.gain.value = 0;
		const trem = ctx.createGain(); trem.gain.value = 0.55;
		const bp = ctx.createBiquadFilter(); bp.type = 'bandpass'; bp.frequency.value = 2600; bp.Q.value = 0.8;
		const lfo = ctx.createOscillator(); lfo.type = 'sine'; lfo.frequency.value = 8.5;
		const depth = ctx.createGain(); depth.gain.value = 0.45;
		lfo.connect(depth).connect(trem.gain);
		const o: OscillatorNode[] = [];
		// a minor-second cluster two octaves up, three slightly detuned saws
		for (const [semi, det] of [[0, -6], [1, 5], [7, 0]] as [number, number][]) {
			const s = ctx.createOscillator(); s.type = 'sawtooth';
			s.frequency.value = mtof(id.tonic + 36 + semi); s.detune.value = det;
			s.connect(bp); o.push(s);
		}
		bp.connect(trem).connect(g).connect(stemBus);
		const t = ctx.currentTime;
		o.forEach((s) => s.start(t)); lfo.start(t);
		nodes.push(g, trem, bp, depth);
		tens = { o, lfo, g, nodes };
	}
	function stopTension() {
		if (!tens) return;
		const t = tens;
		tens = null;
		try { t.g.gain.cancelScheduledValues(ctx.currentTime); t.g.gain.setTargetAtTime(0, ctx.currentTime, 0.2); } catch { /* ctx closed */ }
		const end = ctx.currentTime + 1.2;
		try { t.o.forEach((s) => s.stop(end)); t.lfo.stop(end); } catch { /* already stopped */ }
		const h = setTimeout(() => { for (const n of [...t.o, t.lfo, ...t.nodes]) { try { n.disconnect(); } catch { /* gone */ } } if (tensStopT === h) tensStopT = null; }, 1500);
		tensStopT = h;
	}

	// ----- timers -----
	function armSweep() {
		if (sweepT || disposed) return;
		sweepT = setTimeout(() => {
			sweepT = null;
			sweep(kit);
			if (kit.trash.length && !disposed) armSweep();
		}, 3000);
	}
	function active(): boolean { return enabled && layersOn && (percEnv > 0.01 || bossEnv > 0.01 || tensEnv > 0.01 || wantPerc() > 0 || wantBoss() || wantTens() > 0); }
	function ensureTimer() {
		if (timer || disposed || !active()) return;
		lastTick = ctx.currentTime;
		timer = setInterval(tick, 200);
	}
	function stopTimer() { if (timer) { clearInterval(timer); timer = null; } }

	const wantPerc = () => (enabled && layersOn && ex.inCombat && music && music.mood !== 'dead' ? clamp((music.danger ?? 0) * 0.6 + Math.min(ex.hostiles, 6) / 6 * 0.4, 0, 1) : 0);
	const wantBoss = () => enabled && layersOn && ex.inCombat && ex.boss && music?.mood !== 'dead';
	const wantTens = () => (enabled && layersOn && music && music.mood !== 'dead' && music.mood !== 'world' && ex.hpFrac < 0.35 && ex.hpFrac > 0 ? clamp((0.35 - ex.hpFrac) / 0.3, 0.25, 1) : 0);

	function tick() {
		if (disposed) return;
		const now = ctx.currentTime;
		const dt = clamp(now - lastTick, 0, 1);
		lastTick = now;
		if (document.hidden || ctx.state !== 'running') return;
		// envelopes: up over about 1.8 s, down over about 2.5 s
		const slew = (env: number, target: number) => (target > env ? Math.min(target, env + dt / 1.8) : Math.max(target, env - dt / 2.5));
		percEnv = slew(percEnv, wantPerc());
		bossEnv = slew(bossEnv, wantBoss() ? 1 : 0);
		tensEnv = slew(tensEnv, wantTens());
		const horizon = now + 0.7;
		sweep(kit);
		if (percEnv > 0.02) schedulePerc(now, horizon);
		else nextStep = 0;
		if (bossEnv > 0.02) scheduleBoss(now, horizon); else bossNext = 0;
		// tension
		if (tensEnv > 0.02) {
			startTension();
			if (tens) tens.g.gain.setTargetAtTime(0.32 * tensEnv, now, 0.5);
		} else if (tens) stopTension();
		if (!active()) { stopTimer(); armSweep(); }
	}

	function schedulePerc(now: number, horizon: number) {
		const g = beat() / 2;
		if (nextStep * g + epoch < now) nextStep = Math.ceil((now - epoch) / g);
		const lv = percEnv;
		let n = 0;
		while (epoch + nextStep * g < horizon && n < 6) {
			const i = nextStep++;
			const t = epoch + i * g;
			if (t < now + 0.01) continue;
			const pos = i % 8;
			const rn = seeded(hash(`d${id.seed}:${i}`))();
			const vel = 0.3 * (0.55 + 0.45 * lv) * percEnv;
			const play = (role: 0 | 1 | 2 | 3, x: number) => { if (n < 6) { hit(kit, stemBus, t, id.kit, role, vel * x, id.root); n++; } };
			if (id.style === 'rattle') {
				if (pos % 2 === 0 && lv > 0.1) play(1, 0.8);
				if (pos % 2 === 1 && lv > 0.4 && rn < 0.7) play(2, 0.6);
				if (pos === 0 && lv > 0.6) play(0, 0.5);
				continue;
			}
			if (pos === 0) play(lv > 0.85 && (i >> 3) % 2 === 0 ? 3 : 0, 1);
			else if (pos === 4 && lv > 0.15) play(0, 0.8);
			else if ((pos === 2 || pos === 6) && lv > 0.4 && rn < 0.9) play(1, 0.6);
			else if (pos === 7 && lv > 0.6 && rn < 0.5) play(1, 0.5);
			else if (pos % 2 === 1 && lv > 0.75 && rn < 0.6) play(2, 0.35);
		}
	}

	function scheduleBoss(now: number, horizon: number) {
		const seg = beat() * 8; // 2 bars of 4/4
		if (bossIdPlaying !== ex.bossId) { bossIdPlaying = ex.bossId; bossNext = 0; }
		if (bossNext === 0) bossNext = quant(now + 0.05, 1);
		if (bossNext > horizon) return;
		const t = bossNext;
		bossNext += seg * 0.7; // overlap: drone2 ramps in and out over its own length
		const e = bossEnv;
		drone2(kit, stemBus, t, id.tonic - 12, seg, 0.3 * e, 'sawtooth', 0.5);
		drone2(kit, stemBus, t, id.tonic - 5, seg, 0.18 * e, 'triangle', 0.4);
		// the motif, once per segment, starting on the second beat
		const mo = bossMotif(ex.bossId);
		let bt = 1;
		mo.deg.forEach((d, i) => {
			if (bt + mo.beats[i] <= 8) INST.brass(kit, stemBus, t + bt * beat(), degree(id, d, 12), mo.beats[i] * beat() * 0.95, 0.3 * e);
			bt += mo.beats[i];
		});
		armSweep();
	}

	// ----- stingers -----
	let curPrio = 0, curEnd = 0;
	let queue: { kind: string; inten: number; id: Ident; prio: number }[] = [];
	let lastFight = -99, lastVictory = -99, fightStart = 0;

	function duck(db: number, seconds: number) {
		const t = ctx.currentTime;
		const g = duckNode.gain;
		try {
			g.cancelScheduledValues(t);
			g.setValueAtTime(g.value, t);
			g.linearRampToValueAtTime(Math.pow(10, -clamp(db, 0, 12) / 20), t + 0.12);
			g.setValueAtTime(Math.pow(10, -clamp(db, 0, 12) / 20), t + Math.max(0.2, seconds - 0.3));
			g.linearRampToValueAtTime(1, t + Math.max(0.5, seconds + 0.5));
		} catch { /* ctx closed */ }
		try { duckCb?.(db, seconds); } catch { /* coordinator callback failed */ }
	}

	function playStinger(kind: string, inten: number, who: Ident, prio: number) {
		const r = recipe(kind, inten, who);
		if (!r.evs.length && !r.drone && !r.drums && !r.tex) return;
		const start = quant(ctx.currentTime + 0.03, 0.5);
		const t0 = Math.min(start, ctx.currentTime + 0.4);
		for (const x of r.evs) INST[x.inst]?.(kit, stingBus, t0 + x.t, x.midi, x.dur, x.vel);
		for (const d of r.drone ?? []) drone2(kit, stingBus, t0, d.midi, d.dur, d.vel, d.type ?? 'sawtooth', 0.7);
		for (const d of r.drums ?? []) hit(kit, stingBus, t0 + d.t, d.kit, d.role, d.vel, who.root);
		for (const x of r.tex ?? []) tex(kit, stingBus, t0 + x.t, x.kind, x.vel);
		duck(inten >= 3 ? 5 : inten === 2 ? 4 : 3, r.len);
		curPrio = prio;
		curEnd = t0 + r.len;
		armSweep();
		if (stingT) clearTimeout(stingT);
		stingT = setTimeout(() => {
			stingT = null;
			curPrio = 0;
			const q = queue.shift();
			if (q && !disposed && enabled && stingersOn) playStinger(q.kind, q.inten, q.id, q.prio);
		}, Math.max(100, (curEnd - ctx.currentTime) * 1000 + 80));
	}

	function request(kind: string, inten: number, who: Ident) {
		if (disposed || !enabled || !stingersOn || document.hidden || ctx.state !== 'running') return;
		const prio = PRIO[kind] ?? 3;
		if (curPrio && ctx.currentTime < curEnd) {
			if (prio < 4) return; // minor beats are dropped while something plays
			if (prio <= curPrio && queue.length >= 2) return;
			if (queue.some((q) => q.kind === kind)) return;
			queue.push({ kind, inten, id: who, prio });
			queue.sort((a, b) => b.prio - a.prio);
			if (queue.length > 2) queue.length = 2;
			return;
		}
		playStinger(kind, inten, who, prio);
	}

	// ----- public -----
	let prevZone = '', prevCombat: boolean | null = null, prevHostiles = 0;
	function setMusic(m: QudMusic | null, extras?: Partial<MusicExtras>) {
		if (disposed) return;
		music = m;
		ex = {
			hostiles: extras?.hostiles ?? 0,
			boss: !!extras?.boss,
			bossId: extras?.bossId ?? 0,
			hpFrac: extras?.hpFrac ?? clamp(m?.hp ?? 1, 0, 1),
			inCombat: extras?.inCombat ?? (!!m?.combat && (extras?.hostiles ?? 1) > 0),
			zone: extras?.zone,
		};
		const key = `${m?.sphere ?? ''}|${m?.genre ?? ''}|${m?.seed ?? 0}`;
		if (key !== idKey) {
			idKey = key;
			id = identity(m);
			if (tens) { try { tens.o.forEach((s, k) => { s.frequency.setTargetAtTime(mtof(id.tonic + 36 + [0, 1, 7][k]), ctx.currentTime, 0.5); }); } catch { /* gone */ } }
		}
		const zone = ex.zone != null ? String(ex.zone) : `${m?.mood}|${m?.seed}|${m?.depth}`;
		const dead = !m || m.mood === 'dead';
		if (prevCombat === null) { prevCombat = ex.inCombat; prevHostiles = ex.hostiles; prevZone = zone; }
		else {
			const now = ctx.currentTime;
			if (ex.inCombat && !prevCombat && !dead && now - lastFight > 6) {
				lastFight = now; fightStart = now;
				request('fightstart', ex.boss ? 3 : 2, id);
			} else if (!ex.inCombat && prevCombat && !dead && zone === prevZone && ex.hpFrac > 0 && now - fightStart > 1.5 && now - lastVictory > 10) {
				lastVictory = now;
				request('victory', prevHostiles > 3 ? 2 : 1, id);
			}
			prevCombat = ex.inCombat; prevHostiles = ex.hostiles; prevZone = zone;
		}
		ensureTimer();
	}

	function cue(cues: StingerCue[] | null | undefined) {
		if (disposed || !cues || !cues.length) return;
		for (const c of cues.slice(0, 4)) {
			const who = c.sphere || c.genre ? identity({ ...(music ?? ({} as QudMusic)), sphere: c.sphere || music?.sphere, genre: c.genre || music?.genre || 'fantasy', seed: music?.seed ?? 1 } as QudMusic) : id;
			request(c.kind, c.intensity, who);
		}
	}

	function applyEnabled() {
		const on = enabled;
		try { master.gain.cancelScheduledValues(ctx.currentTime); master.gain.setTargetAtTime(on ? volume : 0, ctx.currentTime, on ? 0.6 : 0.15); } catch { /* ctx closed */ }
		if (!on) {
			stopTimer(); stopTension();
			queue = []; curPrio = 0;
			percEnv = bossEnv = tensEnv = 0; nextStep = 0; bossNext = 0;
		} else ensureTimer();
	}

	return {
		setMusic, cue, duck,
		onDuck(cb) { duckCb = cb; },
		getDuckNode: () => duckNode,
		setTempo(b: number) {
			if (!Number.isFinite(b) || b < 40 || b > 220) return;
			bpm = b; epoch = ctx.currentTime; nextStep = 0; bossNext = 0;
		},
		setEnabled(on: boolean) { enabled = on; applyEnabled(); },
		setVolume(v: number) {
			volume = clamp(Number.isFinite(v) ? v : 1, 0, 1);
			if (enabled) { try { master.gain.setTargetAtTime(volume, ctx.currentTime, 0.1); } catch { /* ctx closed */ } }
		},
		setLayers(on: boolean) {
			layersOn = on;
			if (!on) { percEnv = bossEnv = tensEnv = 0; stopTimer(); stopTension(); armSweep(); } else ensureTimer();
		},
		setStingers(on: boolean) { stingersOn = on; if (!on) { queue = []; } },
		dispose() {
			if (disposed) return;
			disposed = true;
			stopTimer();
			if (sweepT) clearTimeout(sweepT);
			if (stingT) clearTimeout(stingT);
			if (tensStopT) clearTimeout(tensStopT);
			sweepT = stingT = tensStopT = null;
			if (tens) {
				const t = tens; tens = null;
				try { t.o.forEach((s) => s.stop()); t.lfo.stop(); } catch { /* stopped */ }
				for (const n of [...t.o, t.lfo, ...t.nodes]) { try { n.disconnect(); } catch { /* gone */ } }
			}
			disposeKit(kit);
			for (const n of [stemBus, stemComp, stingBus, stingComp, master, duckNode]) { try { n.disconnect(); } catch { /* gone */ } }
			queue = [];
		},
	};
}

/**
 * Helper for a base music that only has setVolume (music.ts keeps its master gain private): returns a duck
 * callback that eases the volume down by `db` and back over `seconds`. `setVol` applies a 0..1 volume and
 * `baseVol` returns the user's current volume. dispose() stops any ramp in progress and restores the volume.
 */
export function volumeDucker(setVol: (v: number) => void, baseVol: () => number): { duck(db: number, seconds: number): void; dispose(): void } {
	let h: ReturnType<typeof setInterval> | null = null;
	const stop = () => { if (h) { clearInterval(h); h = null; } };
	return {
		duck(db: number, seconds: number) {
			stop();
			const g = Math.pow(10, -clamp(db, 0, 12) / 20);
			const t0 = performance.now();
			const total = (seconds + 0.5) * 1000;
			h = setInterval(() => {
				const t = performance.now() - t0;
				let f: number;
				if (t < 120) f = 1 - (1 - g) * (t / 120);
				else if (t < total - 500) f = g;
				else f = g + (1 - g) * clamp((t - (total - 500)) / 500, 0, 1);
				setVol(baseVol() * f);
				if (t >= total) { stop(); setVol(baseVol()); }
			}, 50);
		},
		dispose() { stop(); setVol(baseVol()); },
	};
}
