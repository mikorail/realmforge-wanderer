// Location soundscapes: quiet looping synthesised beds that say where you are, plus sparse one-shots.
// Pure WebAudio, no samples. The coordinator owns the AudioContext and destination; this builds its own
// nodes on them (master gain -> limiter -> dest) and is driven by the engine's `frame.sound` scene (SoundInfo).
// Each bed has its own gain node and is faded in/out (about 2 s) as the scene changes; sources of a bed are
// only alive while the bed is audible. One-shots are scheduled with a seeded rng and a short look-ahead.
import { SCALES, PROFILES } from './scales';

export interface SoundSceneIn {
	place: string;
	crowd: number;
	indoors: boolean;
	depth: number;
	night: boolean;
	weather: string;
	near: Record<string, boolean>;
	culture: string;
	genre: string;
	event: string;
}

export interface AmbientContext {
	ph?: number; // 0..1 day phase (0 = midnight, 0.5 = noon, 0.75 = dusk)
	dl?: number; // 0..1 daylight
	mood?: string; // music mood ('town', 'wild', etc.)
	danger?: number; // 0..1 danger level
	doom?: number; // 0..1 world-scale threat
	combat?: boolean;
	nearTavern?: boolean;
}

export interface Soundscape {
	set(scene: SoundSceneIn | null | undefined, ctx?: AmbientContext): void;
	setEnabled(on: boolean): void;
	setVolume(v: number): void;
	dispose(): void;
}

type Levels = Record<string, number>;
const clamp01 = (x: number) => Math.max(0, Math.min(1, x));

const TOWN_PLACES = new Set(['street', 'market', 'tavern', 'temple', 'forge', 'harbor']);
function isTown(s: SoundSceneIn, ctx?: AmbientContext): boolean {
	return TOWN_PLACES.has(s.place) || ctx?.mood === 'town';
}
/** Smooth 0.12..1 activity factor for markets: full by day, gently fading down through dusk/night. */
function marketActivity(s: SoundSceneIn, ctx?: AmbientContext): number {
	const ph = ctx?.ph;
	if (ph !== undefined && Number.isFinite(ph)) {
		if (ph >= 0.28 && ph <= 0.68) return 1;
		if (ph >= 0.20 && ph < 0.28) return 0.12 + 0.88 * ((ph - 0.20) / 0.08);
		if (ph > 0.68 && ph <= 0.84) return 1 - 0.88 * ((ph - 0.68) / 0.16);
		return 0.12;
	}
	if (ctx?.dl !== undefined && Number.isFinite(ctx.dl)) {
		return 0.12 + 0.88 * clamp01((ctx.dl - 0.15) / 0.55);
	}
	return s.night ? 0.14 : 1;
}
function bellWindow(ctx?: AmbientContext): '' | 'noon' | 'dusk' {
	const ph = ctx?.ph;
	if (ph === undefined || !Number.isFinite(ph)) return '';
	if (ph >= 0.44 && ph <= 0.56) return 'noon';
	if (ph >= 0.70 && ph <= 0.82) return 'dusk';
	return '';
}
function isWarTown(s: SoundSceneIn, ctx?: AmbientContext): boolean {
	if (s.event === 'war' || s.event === 'siege' || /war|siege|raid|battle/i.test(s.event || '')) return true;
	return isTown(s, ctx) && ((ctx?.danger ?? 0) >= 0.55 || !!ctx?.combat || (ctx?.doom ?? 0) >= 0.65);
}
function isNearTavern(s: SoundSceneIn, ctx?: AmbientContext): boolean {
	const n = s.near || {};
	return s.place === 'tavern' || !!n.tavern || !!n.inn || !!n.hearth || !!ctx?.nearTavern || (!!n.fire && isTown(s, ctx));
}

function hash(s: string): number {
	let h = 2166136261;
	for (let i = 0; i < s.length; i++) { h ^= s.charCodeAt(i); h = Math.imul(h, 16777619); }
	return h >>> 0;
}
function mulberry32(a: number): () => number {
	return () => {
		a = (a + 0x6d2b79f5) | 0;
		let t = Math.imul(a ^ (a >>> 15), 1 | a);
		t = (t + Math.imul(t ^ (t >>> 7), 61 | t)) ^ t;
		return ((t ^ (t >>> 14)) >>> 0) / 4294967296;
	};
}

/** plan maps a scene onto bed levels 0..1 (each bed has its own peak scale, all far below -26 dBFS). */
export function plan(s: SoundSceneIn | null | undefined, ctx?: AmbientContext): Levels {
	const L: Levels = {};
	if (!s || !s.place) return L;
	const add = (k: string, v: number) => { v = clamp01(v); if (v > 0.01) L[k] = Math.max(L[k] ?? 0, v); };
	const n = s.near || {};
	const out = s.indoors ? 0.35 : 1;
	const ev = s.event;
	let crowd = [0, 0.3, 0.65, 1][Math.max(0, Math.min(3, s.crowd | 0))];
	if (ev === 'plague') crowd *= 0.4;
	else if (ev === 'curfew') crowd *= 0.25;
	else if (ev === 'feast' || ev === 'festival') crowd = Math.min(1, crowd * 1.3 + 0.15);
	const windy = !!s.weather && /storm|tempest|blizzard|sandstorm|radstorm/.test(s.weather);
	const mAct = marketActivity(s, ctx);
	switch (s.place) {
		case 'street': add('street', 0.7 * out); add('crowd', crowd); break;
		case 'market': add('market', (0.35 + 0.65 * crowd) * mAct); add('street', 0.25 * out); break;
		case 'tavern': add('tavern', Math.max(0.45, crowd)); break;
		case 'temple': add('temple', 0.85); add('crowd', crowd * 0.35); break;
		case 'forge': add('forge', 0.8); add('crowd', crowd * 0.4); add('street', 0.2 * out); break;
		case 'harbor': add('harbor', 0.8 * (s.indoors ? 0.4 : 1)); add('crowd', crowd * 0.6); break;
		case 'farm': add('grass', 0.5 * out); break;
		case 'wilds-forest': add('forest', out); if (s.night) add('crickets', 0.7 * out); break;
		case 'wilds-grass': add('grass', out); if (s.night) add('crickets', 0.8 * out); break;
		case 'wilds-desert': add('desert', out); break;
		case 'wilds-mountain': add('mountain', out); break;
		case 'wilds-swamp': add('swamp', out); if (s.night) add('crickets', 0.5 * out); break;
		case 'wilds-snow': add('snow', out); break;
		case 'coast': add('harbor', 0.9 * out); add('grass', 0.2 * out); break;
		case 'cave': add('cave', 1); break;
		case 'dungeon': add('dungeon', 1); break;
		case 'crypt': add('crypt', 1); break;
		case 'mine': add('mine', 1); break;
		case 'tower': add('tower', 0.8); break;
		case 'sky': add('tower', 1); break;
		case 'underwater': add('underwater', 1); break;
		case 'ship-deck': add('shipdeck', 1); break;
		case 'rail-station': add('street', 0.5 * out); add('crowd', crowd); add('m-steam', 0.5); break;
		case 'camp': add('camp', 1); add(s.night ? 'crickets' : 'grass', 0.4); break;
	}
	if (windy && !s.indoors && s.depth === 0) add('mountain', 0.4);
	if (n.fire && s.place !== 'camp') add('camp', 0.45);
	if (n.forge && s.place !== 'forge') add('forge', 0.35);
	if (n.lava) add('lava', 1);
	if (n.waterfall) add('waterfall', 0.9);
	else if (n.fountain) add('fountain', 0.6);
	else if (n.water && s.place !== 'harbor' && s.place !== 'coast' && s.place !== 'shipdeck' && s.place !== 'underwater') add('brook', 0.5 * out);
	if (n.altar && s.place !== 'temple') add('temple', 0.35);
	if (isNearTavern(s, ctx)) add('hearth', s.place === 'tavern' ? 0.85 : 0.55);
	if (n.machine) {
		const g = s.genre;
		add(g === 'cyberpunk' ? 'm-cyber' : g === 'biopunk' ? 'm-bio' : g === 'steampunk' ? 'm-steam' : 'm-hum', 0.8);
	} else if (s.genre === 'cyberpunk' && (s.place === 'street' || s.place === 'market')) add('m-cyber', 0.3);
	return L;
}

// ---------------------------------------------------------------- the engine

interface Rig { srcs: AudioScheduledSourceNode[]; nodes: AudioNode[] }
interface Bed { out: GainNode; rig: Rig; level: number; kill: number }
interface Env { c: AudioContext; white: AudioBuffer; brown: AudioBuffer; rnd: () => number }
interface Bus { dry: AudioNode; wet: AudioNode }

/** per-bed full-level gain (roughly peak amplitude; keep every one under 0.045 = -27 dBFS) */
const GAIN: Record<string, number> = {
	street: 0.05, crowd: 0.09, market: 0.1, tavern: 0.1, temple: 0.03, forge: 0.06, harbor: 0.1, shipdeck: 0.08,
	forest: 0.1, grass: 0.09, desert: 0.1, mountain: 0.1, swamp: 0.05, snow: 0.09, cave: 0.04, dungeon: 0.08,
	crypt: 0.03, mine: 0.05, tower: 0.09, underwater: 0.1, camp: 0.04, hearth: 0.032, lava: 0.07, crickets: 0.025,
	'm-cyber': 0.03, 'm-steam': 0.05, 'm-bio': 0.05, 'm-hum': 0.04, brook: 0.05, waterfall: 0.06, fountain: 0.05,
};

export function createSoundscape(ctx: AudioContext, dest: AudioNode): Soundscape {
	let enabled = true, volume = 1, disposed = false;
	let scene: SoundSceneIn | null = null;
	let amb: AmbientContext | undefined = undefined;
	let lastPh: number | undefined = undefined;
	let lastBellAt = -999;
	let sceneKey = '';
	let rnd = mulberry32(1);

	// master: gain -> limiter -> dest
	const master = ctx.createGain();
	master.gain.value = 0;
	const lim = ctx.createDynamicsCompressor();
	lim.threshold.value = -28; lim.knee.value = 0; lim.ratio.value = 20; lim.attack.value = 0.005; lim.release.value = 0.25;
	master.connect(lim).connect(dest);

	// one-shot bus: dry, plus a long echo (feedback delay) used by caves and dungeons
	const osBus = ctx.createGain();
	osBus.connect(master);
	const echoIn = ctx.createGain();
	const d1 = ctx.createDelay(1.5), d2 = ctx.createDelay(1.5), fb = ctx.createGain(), fbLp = ctx.createBiquadFilter(), echoOut = ctx.createGain();
	d1.delayTime.value = 0.37; d2.delayTime.value = 0.61; fb.gain.value = 0.58; fbLp.type = 'lowpass'; fbLp.frequency.value = 2200; echoOut.gain.value = 0.7;
	echoIn.connect(d1); echoIn.connect(d2); d1.connect(fbLp); d2.connect(fbLp); fbLp.connect(fb); fb.connect(d1); fb.connect(d2);
	fbLp.connect(echoOut); echoOut.connect(master);
	const bus: Bus = { dry: osBus, wet: echoIn };

	// noise buffers
	const len = Math.floor(ctx.sampleRate * 4);
	const white = ctx.createBuffer(1, len, ctx.sampleRate), brown = ctx.createBuffer(1, len, ctx.sampleRate);
	{
		const w = white.getChannelData(0), b = brown.getChannelData(0);
		const r = mulberry32(7);
		let last = 0;
		for (let i = 0; i < len; i++) {
			const x = r() * 2 - 1;
			w[i] = x;
			last = (last + 0.02 * x) / 1.02;
			b[i] = last * 3.5;
		}
	}
	const env: Env = { c: ctx, white, brown, rnd: () => rnd() };

	// ------------------------------------------------------------ beds
	const noise = (r: Rig, buf: AudioBuffer, seedOff: number) => {
		const s = ctx.createBufferSource();
		s.buffer = buf; s.loop = true;
		s.start(0, (seedOff % 3.5));
		r.srcs.push(s);
		return s;
	};
	const filt = (r: Rig, type: BiquadFilterType, f: number, q = 0.7) => {
		const b = ctx.createBiquadFilter();
		b.type = type; b.frequency.value = f; b.Q.value = q;
		r.nodes.push(b);
		return b;
	};
	const gain = (r: Rig, v: number) => { const g = ctx.createGain(); g.gain.value = v; r.nodes.push(g); return g; };
	const lfo = (r: Rig, f: number, depth: number, target: AudioParam, type: OscillatorType = 'sine') => {
		const o = ctx.createOscillator(), g = ctx.createGain();
		o.type = type; o.frequency.value = f; g.gain.value = depth;
		o.connect(g).connect(target);
		o.start();
		r.srcs.push(o); r.nodes.push(g);
	};
	/** noise -> filter chain -> amp (with modulators) -> out */
	const chain = (r: Rig, out: AudioNode, o: { buf?: AudioBuffer; f: number; q?: number; type?: BiquadFilterType; base: number; mods?: [number, number][]; fmod?: [number, number]; seed?: number; f2?: [BiquadFilterType, number] }) => {
		const s = noise(r, o.buf ?? white, o.seed ?? Math.floor(rnd() * 3500) / 1000);
		const f = filt(r, o.type ?? 'bandpass', o.f, o.q ?? 0.8);
		const a = gain(r, o.base);
		s.connect(f);
		let tail: AudioNode = f;
		if (o.f2) { const f2 = filt(r, o.f2[0], o.f2[1]); tail.connect(f2); tail = f2; }
		tail.connect(a).connect(out);
		for (const [fr, dp] of o.mods ?? []) lfo(r, fr, dp, a.gain);
		if (o.fmod) lfo(r, o.fmod[0], o.fmod[1], f.frequency);
		return { f, a };
	};
	const murmur = (r: Rig, out: AudioNode, f: number, rate: number) => {
		chain(r, out, { f, q: 1.1, base: 0.55, mods: [[rate, 0.3], [rate * 1.57, 0.2], [0.37, 0.2]], f2: ['lowpass', f * 2.2] });
		chain(r, out, { f: f * 1.7, q: 1.5, base: 0.25, mods: [[rate * 0.8, 0.15], [0.21, 0.1]], seed: 1.7 });
	};

	function buildBed(id: string, out: GainNode, r: Rig): void {
		const s = scene;
		switch (id) {
			case 'tavern': murmur(r, out, 520, 5.2); chain(r, out, { buf: brown, f: 160, type: 'lowpass', base: 0.5 }); break;
			case 'market': murmur(r, out, 850, 6.1); chain(r, out, { f: 2400, q: 0.6, base: 0.06, mods: [[0.5, 0.04]] }); break;
			case 'crowd': murmur(r, out, 650, 4.8); break;
			case 'street':
				chain(r, out, { buf: brown, f: 260, type: 'lowpass', base: 0.5, mods: [[0.13, 0.25]] });
				chain(r, out, { f: 900, q: 0.5, base: 0.1, mods: [[0.09, 0.08]], fmod: [0.05, 200] });
				break;
			case 'forge':
				chain(r, out, { buf: brown, f: 380, type: 'lowpass', base: 0.45, mods: [[0.22, 0.4]] }); // bellows breathing
				chain(r, out, { f: 1200, q: 0.7, base: 0.04, mods: [[0.22, 0.04]] });
				break;
			case 'temple': {
				const cu = s?.culture ?? '';
				const sc = SCALES[PROFILES[cu]?.scales[0] ?? 'dorian'] ?? SCALES.dorian;
				const root = 110 * Math.pow(2, (hash(cu || 'x') % 7) / 12);
				const pad = gain(r, 0.22);
				lfo(r, 0.07, 0.08, pad.gain);
				const fs: [number, number][] = [[650, 5], [1080, 8], [2500, 12]], fg = [1, 0.6, 0.2];
				const f0 = gain(r, 1);
				fs.forEach(([f, q], i) => { const b = filt(r, 'bandpass', f, q), g = gain(r, fg[i]); f0.connect(b); b.connect(g).connect(pad); });
				const notes = [0, sc[Math.min(2, sc.length - 1)], sc[Math.min(4, sc.length - 1)], 12];
				notes.forEach((st, i) => {
					for (const det of [-6, 5]) {
						const o = ctx.createOscillator();
						o.type = 'sawtooth'; o.frequency.value = root * Math.pow(2, st / 12); o.detune.value = det;
						const g = gain(r, i === 0 ? 0.35 : 0.22);
						o.connect(g).connect(f0); o.start(); r.srcs.push(o);
						lfo(r, 4.6 + i * 0.3, 3, o.detune);
					}
				});
				pad.connect(out);
				break;
			}
			case 'harbor':
				chain(r, out, { buf: brown, f: 500, type: 'lowpass', base: 0.5, mods: [[0.11, 0.3], [0.19, 0.15]] });
				chain(r, out, { f: 1800, q: 0.6, base: 0.12, mods: [[0.11, 0.1]] });
				break;
			case 'shipdeck':
				chain(r, out, { buf: brown, f: 420, type: 'lowpass', base: 0.5, mods: [[0.14, 0.3]] });
				chain(r, out, { f: 2600, q: 2, base: 0.06, mods: [[0.08, 0.05]], fmod: [0.06, 400] }); // rigging
				chain(r, out, { f: 600, q: 1, base: 0.12, mods: [[0.05, 0.1]] });
				break;
			case 'forest': chain(r, out, { f: 3200, q: 0.7, base: 0.2, mods: [[0.17, 0.15], [0.43, 0.1]], f2: ['lowpass', 6000] }); chain(r, out, { f: 400, q: 0.6, base: 0.12, mods: [[0.08, 0.1]] }); break;
			case 'grass': chain(r, out, { f: 520, q: 0.7, base: 0.4, mods: [[0.1, 0.25]], fmod: [0.07, 150] }); break;
			case 'desert':
				chain(r, out, { f: 420, q: 0.8, base: 0.35, mods: [[0.09, 0.25]], fmod: [0.06, 120] });
				chain(r, out, { type: 'highpass', f: 4500, base: 0.05, mods: [[0.13, 0.04]] }); // sand hiss
				break;
			case 'mountain':
				chain(r, out, { f: 330, q: 2, base: 0.4, mods: [[0.08, 0.3]], fmod: [0.05, 90] });
				chain(r, out, { f: 1100, q: 5, base: 0.06, mods: [[0.06, 0.05]], fmod: [0.04, 300] });
				break;
			case 'swamp':
				chain(r, out, { buf: brown, f: 300, type: 'lowpass', base: 0.4, mods: [[0.12, 0.2]] });
				chain(r, out, { f: 2200, q: 1, base: 0.04, mods: [[0.3, 0.03]] });
				break;
			case 'snow': chain(r, out, { f: 700, q: 1.2, base: 0.4, mods: [[0.07, 0.3]], fmod: [0.05, 200] }); chain(r, out, { f: 2000, q: 3, base: 0.04, mods: [[0.05, 0.04]] }); break;
			case 'cave': chain(r, out, { buf: brown, f: 180, type: 'lowpass', base: 0.35, mods: [[0.05, 0.15]] }); chain(r, out, { f: 800, q: 4, base: 0.03, mods: [[0.03, 0.03]] }); break;
			case 'dungeon': chain(r, out, { buf: brown, f: 110, type: 'lowpass', base: 0.6, mods: [[0.06, 0.3]] }); break;
			case 'crypt': chain(r, out, { buf: brown, f: 90, type: 'lowpass', base: 0.2, mods: [[0.04, 0.1]] }); break;
			case 'mine': chain(r, out, { buf: brown, f: 160, type: 'lowpass', base: 0.4, mods: [[0.05, 0.2]] }); chain(r, out, { f: 500, q: 3, base: 0.03, mods: [[0.04, 0.03]] }); break;
			case 'tower':
				chain(r, out, { f: 600, q: 6, base: 0.35, mods: [[0.09, 0.25]], fmod: [0.07, 250] });
				chain(r, out, { f: 300, q: 2, base: 0.25, mods: [[0.06, 0.2]] });
				break;
			case 'underwater':
				chain(r, out, { buf: brown, f: 260, type: 'lowpass', base: 0.6, mods: [[0.1, 0.25]] });
				chain(r, out, { f: 120, q: 1, base: 0.3, mods: [[0.07, 0.2]] });
				break;
			case 'camp': chain(r, out, { buf: brown, f: 200, type: 'lowpass', base: 0.2, mods: [[0.3, 0.1]] }); break;
			case 'hearth':
				chain(r, out, { buf: brown, f: 185, type: 'lowpass', base: 0.42, mods: [[0.08, 0.16], [0.23, 0.08]] });
				chain(r, out, { f: 520, q: 1.4, base: 0.07, mods: [[0.14, 0.05]] });
				break;
			case 'lava': chain(r, out, { buf: brown, f: 140, type: 'lowpass', base: 0.6, mods: [[0.2, 0.3], [0.53, 0.2]] }); break;
			case 'crickets': {
				const g = gain(r, 0), g2 = gain(r, 0);
				const mk = (f: number, rate: number, gg: GainNode) => {
					const o = ctx.createOscillator(), am = gain(r, 0.5);
					o.frequency.value = f; o.connect(am).connect(gg); o.start(); r.srcs.push(o);
					lfo(r, rate, 0.5, am.gain, 'square');
				};
				mk(4300, 18, g); mk(5100, 23, g2);
				lfo(r, 0.13, 0.4, g.gain); lfo(r, 0.09, 0.4, g2.gain);
				g.gain.value = 0.5; g2.gain.value = 0.4;
				g.connect(out); g2.connect(out);
				break;
			}
			case 'm-cyber': {
				const g = gain(r, 0.5);
				for (const [f, v] of [[100, 0.5], [200, 0.3], [700, 0.08]] as [number, number][]) {
					const o = ctx.createOscillator(), gg = gain(r, v);
					o.type = 'sawtooth'; o.frequency.value = f; o.connect(gg).connect(g); o.start(); r.srcs.push(o);
				}
				const lp = filt(r, 'lowpass', 1200, 0.5);
				g.connect(lp).connect(out);
				lfo(r, 0.3, 0.1, g.gain);
				break;
			}
			case 'm-steam':
				chain(r, out, { type: 'highpass', f: 3000, base: 0.3, mods: [[0.25, 0.25]] });
				chain(r, out, { buf: brown, f: 120, type: 'lowpass', base: 0.4, mods: [[0.9, 0.15]] });
				break;
			case 'm-bio':
				chain(r, out, { buf: brown, f: 220, type: 'lowpass', base: 0.5, mods: [[0.4, 0.3], [0.17, 0.15]] });
				chain(r, out, { f: 900, q: 6, base: 0.05, mods: [[0.7, 0.04]], fmod: [0.2, 200] });
				break;
			case 'm-hum': {
				const g = gain(r, 0.5);
				for (const f of [50, 100, 150]) { const o = ctx.createOscillator(), gg = gain(r, 0.4 / (f / 50)); o.frequency.value = f; o.connect(gg).connect(g); o.start(); r.srcs.push(o); }
				g.connect(out); lfo(r, 0.2, 0.08, g.gain);
				break;
			}
			case 'brook': chain(r, out, { f: 1600, q: 0.7, base: 0.3, mods: [[3.1, 0.15], [5.3, 0.1], [0.2, 0.1]] }); break;
			case 'waterfall': chain(r, out, { f: 1800, q: 0.4, base: 0.45, mods: [[0.15, 0.1]], f2: ['lowpass', 5000] }); chain(r, out, { buf: brown, f: 300, type: 'lowpass', base: 0.4 }); break;
			case 'fountain': chain(r, out, { f: 2800, q: 0.8, base: 0.25, mods: [[7, 0.12], [11, 0.08]] }); break;
		}
	}

	const beds = new Map<string, Bed>();
	const stopRig = (r: Rig) => {
		for (const s of r.srcs) { try { s.stop(); } catch { /* already stopped */ } }
		for (const n of r.nodes) { try { n.disconnect(); } catch { /* ok */ } }
		for (const s of r.srcs) { try { s.disconnect(); } catch { /* ok */ } }
	};
	function applyBeds(L: Levels) {
		const now = ctx.currentTime;
		for (const id of Object.keys(L)) {
			let b = beds.get(id);
			if (!b) {
				const out = ctx.createGain();
				out.gain.value = 0;
				out.connect(master);
				b = { out, rig: { srcs: [], nodes: [] }, level: 0, kill: 0 };
				b.rig.nodes.push(out);
				buildBed(id, out, b.rig);
				beds.set(id, b);
			}
			if (b.kill) { clearTimeout(b.kill); b.kill = 0; }
			b.level = L[id];
			b.out.gain.cancelScheduledValues(now);
			b.out.gain.setTargetAtTime(L[id] * (GAIN[id] ?? 0.04), now, 0.7);
		}
		for (const [id, b] of beds) {
			if (L[id] || b.kill) continue;
			b.level = 0;
			b.out.gain.cancelScheduledValues(now);
			b.out.gain.setTargetAtTime(0, now, 0.7);
			b.kill = window.setTimeout(() => { stopRig(b.rig); beds.delete(id); }, 3600);
		}
	}

	// ------------------------------------------------------------ one-shots
	const live: number[] = []; // end times of voices in flight
	const room = (t: number) => { while (live.length && live[0] < ctx.currentTime) live.shift(); if (live.length >= 12) return false; live.push(t); live.sort((a, b) => a - b); return true; };
	const ampEnv = (g: GainNode, t: number, a: number, d: number, peak: number) => {
		g.gain.setValueAtTime(0.0001, t);
		g.gain.linearRampToValueAtTime(peak, t + a);
		g.gain.exponentialRampToValueAtTime(0.0001, t + a + d);
	};
	const hook = (src: AudioScheduledSourceNode, nodes: AudioNode[]) => { src.onended = () => nodes.forEach(n => { try { n.disconnect(); } catch { /* ok */ } }); };
	const send = (g: AudioNode, b: Bus, wet: number, dry = 1) => {
		const d = ctx.createGain(); d.gain.value = dry; g.connect(d).connect(b.dry);
		if (wet > 0) { const w = ctx.createGain(); w.gain.value = wet; g.connect(w).connect(b.wet); }
	};
	function tone(t: number, f0: number, f1: number, type: OscillatorType, a: number, d: number, peak: number, wet = 0, vib = 0) {
		if (!room(t + a + d)) return;
		const o = ctx.createOscillator(), g = ctx.createGain();
		o.type = type;
		o.frequency.setValueAtTime(f0, t);
		o.frequency.exponentialRampToValueAtTime(Math.max(20, f1), t + a + d);
		if (vib) { const l = ctx.createOscillator(), lg = ctx.createGain(); l.frequency.value = 6; lg.gain.value = vib; l.connect(lg).connect(o.frequency); l.start(t); l.stop(t + a + d + 0.05); }
		ampEnv(g, t, a, d, peak);
		o.connect(g); send(g, bus, wet);
		o.start(t); o.stop(t + a + d + 0.05); hook(o, [o, g]);
	}
	function burst(t: number, d: number, type: BiquadFilterType, f: number, q: number, peak: number, wet = 0, a = 0.005, f1?: number) {
		if (!room(t + a + d)) return;
		const s = ctx.createBufferSource(), fl = ctx.createBiquadFilter(), g = ctx.createGain();
		s.buffer = env.white; fl.type = type; fl.frequency.setValueAtTime(f, t); fl.Q.value = q;
		if (f1) fl.frequency.exponentialRampToValueAtTime(f1, t + a + d);
		ampEnv(g, t, a, d, peak);
		s.connect(fl).connect(g); send(g, bus, wet);
		s.start(t, rnd() * 3); s.stop(t + a + d + 0.05); hook(s, [s, fl, g]);
	}
	/** a vowel-ish voice: sawtooth through two formants */
	function voice(t: number, f0: number, f1: number, dur: number, peak: number, form: [number, number]) {
		if (!room(t + dur)) return;
		const o = ctx.createOscillator(), g = ctx.createGain(), b1 = ctx.createBiquadFilter(), b2 = ctx.createBiquadFilter();
		o.type = 'sawtooth';
		o.frequency.setValueAtTime(f0, t); o.frequency.linearRampToValueAtTime(f1, t + dur);
		b1.type = b2.type = 'bandpass'; b1.frequency.value = form[0]; b2.frequency.value = form[1]; b1.Q.value = 6; b2.Q.value = 8;
		ampEnv(g, t, dur * 0.25, dur * 0.75, peak);
		o.connect(b1).connect(g); o.connect(b2).connect(g); send(g, bus, 0);
		o.start(t); o.stop(t + dur + 0.05); hook(o, [o, b1, b2, g]);
	}
	const bell = (t: number, f: number, peak: number, wet = 0.3) => {
		[[1, 1], [2.0, 0.5], [2.76, 0.35], [5.4, 0.15]].forEach(([r, v]) => tone(t, f * r, f * r * 0.998, 'sine', 0.004, 4.5 / r ** 0.3, peak * v, wet));
	};
	/** Distant church/temple bells: soft warm lowpass-filtered chimes with gentle attack. */
	function distantBell(t: number, f: number, strikes: number) {
		const lp = ctx.createBiquadFilter();
		lp.type = 'lowpass';
		lp.frequency.value = 1450;
		send(lp, bus, 0.45, 0.85);
		const partials: [number, number][] = [[1, 1], [2.0, 0.45], [2.76, 0.28], [4.07, 0.12]];
		for (let k = 0; k < strikes; k++) {
			const st = t + k * 1.85;
			const decayScale = k === strikes - 1 ? 1.25 : 0.95;
			for (const [r, v] of partials) {
				if (!room(st + 4.0)) continue;
				const o = ctx.createOscillator(), g = ctx.createGain();
				o.type = 'sine';
				o.frequency.setValueAtTime(f * r, st);
				o.frequency.exponentialRampToValueAtTime(Math.max(20, f * r * 0.997), st + 3.8 * decayScale);
				ampEnv(g, st, 0.015, (3.8 * decayScale) / r ** 0.35, 0.018 * v);
				o.connect(g).connect(lp);
				o.start(st);
				o.stop(st + 4.1 * decayScale);
				hook(o, [o, g]);
			}
		}
		window.setTimeout(() => { try { lp.disconnect(); } catch { /* ok */ } }, Math.ceil((strikes * 1.85 + 5.5) * 1000));
	}
	/** Distant war drums: faint, muffled 4-beat cadence for besieged or war-torn towns. */
	function warDrumCadence(t: number) {
		const hits: [number, number, number, number][] = [
			[0, 68, 42, 0.026],
			[0.42, 60, 38, 0.018],
			[0.72, 64, 40, 0.022],
			[1.24, 56, 36, 0.025],
		];
		for (const [off, f0, f1, peak] of hits) {
			const st = t + off;
			if (!room(st + 0.65)) continue;
			const o = ctx.createOscillator(), lp = ctx.createBiquadFilter(), g = ctx.createGain();
			o.type = 'triangle';
			o.frequency.setValueAtTime(f0, st);
			o.frequency.exponentialRampToValueAtTime(f1, st + 0.55);
			lp.type = 'lowpass';
			lp.frequency.value = 240;
			ampEnv(g, st, 0.008, 0.55, peak);
			o.connect(lp).connect(g);
			send(g, bus, 0.4, 0.8);
			o.start(st);
			o.stop(st + 0.6);
			hook(o, [o, lp, g]);
		}
	}
	/** Warm tavern fiddle motif: gentle 4-5 note bowed folk phrase in the local culture scale. */
	function fiddlePhrase(t: number, cu: string) {
		const sc = SCALES[PROFILES[cu]?.scales[0] ?? 'dorian'] ?? SCALES.dorian;
		const root = 293.66 * Math.pow(2, ((hash(cu || 'tavern') % 5) - 2) / 12);
		const len = 4 + (rnd() > 0.5 ? 1 : 0);
		let idx = Math.floor(rnd() * Math.min(4, sc.length));
		let cur = t;
		for (let i = 0; i < len; i++) {
			const deg = sc[Math.max(0, Math.min(sc.length - 1, idx))];
			const freq = root * Math.pow(2, deg / 12);
			const dur = i === len - 1 ? 0.68 : rr(0.36, 0.52);
			if (room(cur + dur + 0.1)) {
				const o1 = ctx.createOscillator(), o2 = ctx.createOscillator();
				const bp = ctx.createBiquadFilter(), lp = ctx.createBiquadFilter(), g = ctx.createGain();
				const vib = ctx.createOscillator(), vg = ctx.createGain();
				o1.type = 'sawtooth'; o1.frequency.setValueAtTime(freq, cur);
				o2.type = 'triangle'; o2.frequency.setValueAtTime(freq * 1.002, cur);
				vib.frequency.value = 5.2; vg.gain.value = freq * 0.008;
				vib.connect(vg).connect(o1.frequency);
				bp.type = 'bandpass'; bp.frequency.value = 950; bp.Q.value = 1.6;
				lp.type = 'lowpass'; lp.frequency.value = 2200;
				ampEnv(g, cur, 0.09, dur, 0.015);
				o1.connect(bp); o2.connect(bp);
				bp.connect(lp).connect(g);
				send(g, bus, 0.18, 0.9);
				o1.start(cur); o2.start(cur); vib.start(cur);
				o1.stop(cur + dur + 0.12); o2.stop(cur + dur + 0.12); vib.stop(cur + dur + 0.12);
				hook(o1, [o1, o2, vib, vg, bp, lp, g]);
			}
			cur += dur * 0.88;
			idx = Math.max(0, Math.min(sc.length - 1, idx + (rnd() > 0.5 ? 1 : -1)));
		}
	}

	interface Shot { gap: [number, number]; w: (s: SoundSceneIn, L: Levels) => number; fire: (t: number, s: SoundSceneIn) => void }
	const rr = (a: number, b: number) => a + rnd() * (b - a);
	const day = (s: SoundSceneIn) => !s.night && !/rain|storm|tempest|blizzard|sandstorm|fog/.test(s.weather || '');
	const chirp = (t: number, base: number) => { const n = 2 + Math.floor(rnd() * 3); for (let i = 0; i < n; i++) tone(t + i * rr(0.09, 0.14), base * rr(0.9, 1.1), base * rr(1.2, 1.5), 'sine', 0.01, 0.07, 0.028); };
	const SHOTS: Record<string, Shot> = {
		clink: { gap: [3, 8], w: (s, L) => L.tavern ? 1 : 0, fire: t => { const f = rr(1900, 2600); tone(t, f, f, 'sine', 0.002, 0.3, 0.025); tone(t, f * 1.5, f * 1.5, 'sine', 0.002, 0.2, 0.015); } },
		laugh: { gap: [14, 30], w: (s, L) => L.tavern || (s.event === 'festival' && L.crowd) ? 1 : 0, fire: t => { const f = rr(150, 210); for (let i = 0; i < 4; i++) voice(t + i * 0.13, f * (1 - i * 0.05), f * (0.9 - i * 0.05), 0.11, 0.028, [800, 1300]); } },
		hawker: {
			gap: [5, 11],
			w: (s, L) => {
				if (!L.market) return 0;
				const act = marketActivity(s, amb);
				return act > 0.22 ? act : 0;
			},
			fire: (t, s) => {
				const act = marketActivity(s, amb);
				const peak = 0.012 + 0.018 * act;
				const f = rr(180, 280);
				voice(t, f, f * 1.45, 0.35, peak, [700, 1150]);
				voice(t + 0.4, f * 1.4, f * 1.1, 0.4, peak, [550, 1000]);
			},
		},
		steps: { gap: [4, 9], w: (s, L) => (L.street && !s.indoors) ? 1 : 0, fire: t => { for (let i = 0; i < 3; i++) burst(t + i * 0.42, 0.07, 'lowpass', 500, 0.7, 0.03); } },
		cart: { gap: [25, 50], w: (s, L) => (L.street && !s.indoors) ? 1 : 0, fire: t => burst(t, 3.2, 'lowpass', 160, 0.7, 0.03, 0, 1.0) },
		clang: { gap: [3, 7], w: (s, L) => L.forge ? 1 : 0, fire: t => { const f = rr(480, 640); [1, 2.4, 4.1].forEach((r, i) => tone(t, f * r, f * r, 'sine', 0.001, 0.9 / (1 + i), 0.04 / (1 + i))); burst(t, 0.03, 'highpass', 3000, 0.7, 0.03); } },
		bell: { gap: [18, 40], w: (s, L) => (L.temple && s.place === 'temple') || (s.event === 'festival' && L.crowd) ? 1 : 0, fire: t => { const root = 110 * Math.pow(2, (hash(scene?.culture ?? 'x') % 7) / 12); bell(t, root * 4, 0.03); } },
		townBell: { gap: [24, 44], w: (s) => (isTown(s, amb) && bellWindow(amb) ? 1 : 0), fire: (t, s) => { const root = 130.8 * Math.pow(2, (hash(s.culture || 'town') % 7) / 12); distantBell(t, root * 2, bellWindow(amb) === 'noon' ? 3 : 2); } },
		toll: { gap: [10, 18], w: s => s.event === 'plague' ? 1 : 0, fire: t => bell(t, 110, 0.035, 0.4) },
		drum: { gap: [6, 14], w: s => s.event === 'war' ? 1 : 0, fire: t => { tone(t, 70, 40, 'sine', 0.005, 0.5, 0.04, 0.35); tone(t + 0.35, 60, 38, 'sine', 0.005, 0.5, 0.03, 0.35); } },
		wardrum: { gap: [7, 13], w: (s) => (isWarTown(s, amb) ? 1 : 0), fire: (t) => warDrumCadence(t) },
		fiddle: { gap: [13, 25], w: (s) => (isNearTavern(s, amb) ? 1 : 0), fire: (t, s) => fiddlePhrase(t, s.culture || 'tavern') },
		gull: { gap: [6, 16], w: (s, L) => (L.harbor && day(s) && !s.indoors) ? 1 : 0, fire: t => { const n = 2 + Math.floor(rnd() * 2); for (let i = 0; i < n; i++) tone(t + i * 0.45, rr(1700, 2100), rr(2400, 2900), 'sine', 0.05, 0.35, 0.025, 0, 40); } },
		slap: { gap: [3, 7], w: (s, L) => L.harbor || L.shipdeck ? 1 : 0, fire: t => burst(t, 0.3, 'lowpass', 700, 0.7, 0.03, 0, 0.08, 250) },
		creak: { gap: [5, 12], w: (s, L) => L.shipdeck ? 1 : L.harbor || L.mine || L.tower ? 0.5 : 0, fire: t => { const f = rr(70, 110); tone(t, f, f * rr(0.6, 1.3), 'sawtooth', 0.3, 0.6, 0.02, s_wet(), 3); } },
		chirp: { gap: [2.5, 7], w: (s, L) => (L.forest || L.grass) && day(s) && !s.indoors ? 1 : 0, fire: t => chirp(t, rr(2800, 3800)) },
		owl: { gap: [12, 28], w: (s, L) => L.forest && s.night && !s.indoors ? 1 : 0, fire: t => { tone(t, 420, 360, 'sine', 0.06, 0.3, 0.03, 0.2); tone(t + 0.55, 400, 330, 'sine', 0.06, 0.4, 0.03, 0.2); } },
		frog: { gap: [1.5, 5], w: (s, L) => L.swamp ? 1 : 0, fire: t => { const n = 2 + Math.floor(rnd() * 3), f = rr(160, 260); for (let i = 0; i < n; i++) tone(t + i * 0.16, f, f * 0.8, 'square', 0.01, 0.08, 0.012); } },
		drip: { gap: [2, 6], w: (s, L) => L.cave ? 1 : L.swamp || L.dungeon || L.crypt || L.mine ? 0.5 : 0, fire: t => { const f = rr(1100, 1900); tone(t, f * 1.4, f * 0.8, 'sine', 0.002, 0.07, 0.03, L_wet()); } },
		rumble: { gap: [20, 45], w: (s, L) => L.dungeon ? 1 : 0, fire: t => { burst(t, 2.5, 'lowpass', 100, 0.7, 0.04, 0.3, 0.6); } },
		clank: { gap: [25, 60], w: (s, L) => L.dungeon ? 1 : 0, fire: t => { const f = rr(300, 500); tone(t, f, f * 0.99, 'triangle', 0.001, 0.5, 0.025, 0.7); tone(t, f * 2.7, f * 2.7, 'sine', 0.001, 0.3, 0.012, 0.7); } },
		whisper: { gap: [30, 70], w: (s, L) => L.crypt ? 1 : 0, fire: t => burst(t, 1.6, 'bandpass', rr(1500, 2000), 5, 0.012, 0.6, 0.5, rr(1000, 1400)) },
		rock: { gap: [4, 12], w: (s, L) => L.mine ? 1 : 0, fire: t => { const n = 2 + Math.floor(rnd() * 3); for (let i = 0; i < n; i++) burst(t + i * rr(0.05, 0.15), 0.025, 'highpass', 1800, 1, 0.025, 0.3); } },
		bubble: { gap: [1.2, 4], w: (s, L) => L.underwater ? 1 : L.lava ? 0.7 : 0, fire: t => { const up = scene?.place === 'underwater'; const f = up ? rr(350, 600) : rr(110, 180); tone(t, f, f * 2.2, 'sine', 0.01, 0.12, 0.028); } },
		crackle: { gap: [0.15, 0.7], w: (s, L) => L.camp ? 1 : 0, fire: t => burst(t, rr(0.01, 0.03), 'highpass', rr(1200, 2500), 0.8, rr(0.01, 0.03)) },
		gear: { gap: [3, 8], w: (s, L) => L['m-steam'] ? 1 : 0, fire: t => { const n = 3 + Math.floor(rnd() * 3); for (let i = 0; i < n; i++) tone(t + i * 0.11, 900, 700, 'square', 0.001, 0.015, 0.01); burst(t + n * 0.11, 0.5, 'highpass', 3500, 0.7, 0.025, 0, 0.05); } },
		squelch: { gap: [3, 9], w: (s, L) => L['m-bio'] ? 1 : 0, fire: t => { const f = rr(160, 260); tone(t, f * 1.6, f * 0.6, 'sine', 0.01, 0.25, 0.03); burst(t, 0.2, 'lowpass', 600, 1, 0.015); } },
		zap: { gap: [4, 12], w: (s, L) => L['m-cyber'] ? 1 : 0, fire: t => { burst(t, 0.08, 'highpass', 4500, 0.8, 0.025); tone(t, rr(1200, 2000), 200, 'sawtooth', 0.002, 0.12, 0.012); } },
	};
	// echo send amounts depend on where we are: caves and dungeons are wet
	function L_wet() { const p = scene?.place; return p === 'cave' ? 0.9 : p === 'dungeon' || p === 'crypt' || p === 'mine' ? 0.7 : 0.15; }
	function s_wet() { return scene?.place === 'mine' || scene?.place === 'tower' ? 0.4 : 0; }

	const next = new Map<string, number>();
	let timer = 0;
	function tick() {
		if (disposed || !enabled || !scene || ctx.state !== 'running') return;
		const now = ctx.currentTime, horizon = now + 0.6;
		const Ls = plan(scene, amb);
		for (const [id, sh] of Object.entries(SHOTS)) {
			const w = sh.w(scene, Ls);
			if (w <= 0) { next.delete(id); continue; }
			let t = next.get(id);
			if (t === undefined || t < now - 1) t = now + rr(sh.gap[0], sh.gap[1]) * 0.5;
			while (t < horizon) {
				if (t >= now) sh.fire(t, scene);
				t += rr(sh.gap[0], sh.gap[1]) / w;
			}
			next.set(id, t);
		}
	}
	function startTimer() { if (!timer && !disposed && enabled && !document.hidden) timer = window.setInterval(tick, 250); }
	function stopTimer() { if (timer) { clearInterval(timer); timer = 0; } }
	const onVis = () => { if (document.hidden) stopTimer(); else if (scene?.place) startTimer(); };
	document.addEventListener('visibilitychange', onVis);

	function levelMaster() {
		const on = enabled && !disposed;
		master.gain.cancelScheduledValues(ctx.currentTime);
		master.gain.setTargetAtTime(on ? volume : 0, ctx.currentTime, 0.7);
	}
	function sceneToKey(s: SoundSceneIn | null | undefined, c?: AmbientContext): string {
		if (!s || !s.place) return '';
		const n = s.near || {};
		const mStep = Math.round(marketActivity(s, c) * 10);
		const bw = bellWindow(c);
		const wt = isWarTown(s, c) ? 1 : 0;
		const nt = isNearTavern(s, c) ? 1 : 0;
		let k = `${s.place}|${s.crowd}|${s.indoors ? 1 : 0}|${s.night ? 1 : 0}|${s.weather}|${s.genre}|${s.culture}|${s.event}|${s.depth}|${mStep}|${bw}|${wt}|${nt}|`;
		for (const key of Object.keys(n).sort()) if (n[key]) k += key + ',';
		return k;
	}

	function clearBeds() {
		for (const b of beds.values()) { if (b.kill) clearTimeout(b.kill); stopRig(b.rig); }
		beds.clear();
	}

	return {
		set(s, c) {
			if (disposed) return;
			const prevPh = lastPh;
			const curPh = c?.ph;
			lastPh = curPh;
			amb = c;
			// Trigger a soft distant church/temple bell right as noon (0.50) or dusk (0.75) arrives in a town
			if (
				enabled &&
				s &&
				ctx.state === 'running' &&
				isTown(s, c) &&
				prevPh !== undefined &&
				curPh !== undefined &&
				ctx.currentTime - lastBellAt > 25
			) {
				const crossedNoon = prevPh < 0.5 && curPh >= 0.5;
				const crossedDusk = prevPh < 0.75 && curPh >= 0.75;
				if (crossedNoon || crossedDusk) {
					lastBellAt = ctx.currentTime;
					const root = 130.8 * Math.pow(2, (hash(s.culture || 'town') % 7) / 12);
					distantBell(ctx.currentTime + 0.08, root * 2, crossedNoon ? 3 : 2);
				}
			}
			const key = sceneToKey(s, c);
			if (key === sceneKey) return;
			const prevPlace = scene?.place;
			sceneKey = key;
			scene = key ? s! : null;
			if (scene && scene.place !== prevPlace) { rnd = mulberry32(hash(scene.place + scene.genre + scene.culture)); next.clear(); }
			if (!enabled) return;
			applyBeds(scene ? plan(scene, amb) : {});
			if (scene) startTimer(); else stopTimer();
		},
		setEnabled(on) {
			if (disposed || on === enabled) return;
			enabled = on;
			levelMaster();
			if (on) {
				if (scene) { applyBeds(plan(scene, amb)); startTimer(); }
			} else {
				stopTimer(); next.clear();
				window.setTimeout(() => { if (!enabled) clearBeds(); }, 3000);
			}
		},
		setVolume(v) { volume = clamp01(v); levelMaster(); },
		dispose() {
			if (disposed) return;
			disposed = true;
			stopTimer(); document.removeEventListener('visibilitychange', onVis);
			clearBeds();
			for (const n of [osBus, echoIn, d1, d2, fb, fbLp, echoOut, master, lim]) { try { n.disconnect(); } catch { /* ok */ } }
		},
	};
}
