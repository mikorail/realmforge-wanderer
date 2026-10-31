// Procedural adaptive chiptune soundtrack. Pure WebAudio, no assets, no deps.
// Everything is generated from (mood, genre, biome, seed) with a deterministic PRNG;
// danger / hp / day / depth adapt the playback continuously without restarting.
import type { QudMusic } from './qud';

export interface MusicHandle {
	setState(m: QudMusic): void;
	setVolume(v: number): void;
	setEnabled(on: boolean): void;
	dispose(): void;
}

type Mood = QudMusic['mood'];

// ---------- math helpers ----------
function hash(s: string): number {
	let h = 2166136261;
	for (let i = 0; i < s.length; i++) {
		h ^= s.charCodeAt(i);
		h = Math.imul(h, 16777619);
	}
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
/** stateless deterministic noise in [0,1) */
function hrand(a: number, b: number, c: number): number {
	let h = (Math.imul(a, 374761393) + Math.imul(b, 668265263) + Math.imul(c, 2147483647)) >>> 0;
	h = Math.imul(h ^ (h >>> 13), 1274126177) >>> 0;
	return ((h ^ (h >>> 16)) >>> 0) / 4294967296;
}
const clamp = (v: number, a: number, b: number) => Math.min(b, Math.max(a, v));
const mtof = (m: number) => 440 * Math.pow(2, (m - 69) / 12);
/** bitcrush-ish: quantise loudness to 16 levels */
const q16 = (v: number) => Math.round(clamp(v, 0, 1) * 15) / 15;

// ---------- weather & season colour ----------
interface WxCfg {
	tempo: number; // bpm multiplier
	patter: number; // 0..1 soft 16th-note rain hats
	sparse: number; // 0..1 chance to drop weak lead / arp notes
	lift: number; // semitones the lead + pad move (snow sits higher)
	soft: number; // velocity multiplier for melodic voices
	drone: number; // 0..1 low sustained drone (fog)
	wash: number; // 0..1 filtered noise wash (sand, ash, blizzard wind)
	washHz: number;
	cut: number; // master low-pass multiplier
}
const WX0: WxCfg = { tempo: 1, patter: 0, sparse: 0, lift: 0, soft: 1, drone: 0, wash: 0, washHz: 900, cut: 1 };
const WX: Record<string, Partial<WxCfg>> = {
	rain: { tempo: 1.03, patter: 0.5 },
	acidrain: { tempo: 1.03, patter: 0.5, cut: 0.85 },
	fallout: { tempo: 1.02, patter: 0.45, cut: 0.85 },
	sporerain: { tempo: 1.02, patter: 0.45, cut: 0.85 },
	storm: { tempo: 1.06, patter: 0.85, cut: 0.9 },
	tempest: { tempo: 1.06, patter: 0.85, cut: 0.9 },
	snow: { tempo: 0.94, sparse: 0.4, lift: 12, soft: 0.75, cut: 1.1 },
	blizzard: { tempo: 0.92, sparse: 0.55, lift: 12, soft: 0.65, wash: 0.55, washHz: 1600, cut: 0.9 },
	frost: { tempo: 0.95, sparse: 0.25, lift: 12, soft: 0.85 },
	fog: { tempo: 0.95, sparse: 0.2, soft: 0.85, drone: 1, cut: 0.6 },
	smog: { tempo: 0.95, sparse: 0.2, soft: 0.85, drone: 1, wash: 0.25, washHz: 500, cut: 0.55 },
	steamfog: { tempo: 0.95, sparse: 0.2, soft: 0.85, drone: 1, cut: 0.6 },
	sandstorm: { tempo: 1.02, wash: 1, washHz: 1100, cut: 0.75 },
	ashfall: { tempo: 0.96, wash: 0.6, washHz: 700, soft: 0.9, cut: 0.7 },
	spores: { drone: 0.5, sparse: 0.15 },
	shimmer: { lift: 12, sparse: 0.2, soft: 0.85 },
	pollen: { tempo: 0.98, lift: 0 },
	heat: { tempo: 0.86, sparse: 0.15, soft: 0.9 },
	overcast: { cut: 0.85 },
};
function wxCfg(id: string | undefined): WxCfg {
	return { ...WX0, ...(id ? WX[id] : undefined) };
}
const SEASON_SHIFT = [0, 2, -1, -3]; // key transposition per season (spring … winter)
/** subtly recolour a scale for the season: summer brightens, autumn/winter darken */
function seasonScale(scale: number[], season: number, r: number): number[] {
	if (scale.length !== 7) return scale;
	const s = scale.slice();
	const swap = (from: number, to: number) => {
		const i = s.indexOf(from);
		if (i >= 0 && !s.includes(to)) s[i] = to;
	};
	if (season === 1 && r < 0.4) swap(5, 6); // lydian-ish raised fourth
	else if (season === 2 && r < 0.5) swap(11, 10); // mixolydian-ish flat seventh
	else if (season === 3 && r < 0.6) { swap(9, 8); swap(4, 3); } // flatten sixth / third
	return s.sort((a, b) => a - b);
}

// ---------- music theory tables ----------
const SCALES: Record<string, number[]> = {
	ionian: [0, 2, 4, 5, 7, 9, 11],
	mixolydian: [0, 2, 4, 5, 7, 9, 10],
	lydian: [0, 2, 4, 6, 7, 9, 11],
	dorian: [0, 2, 3, 5, 7, 9, 10],
	aeolian: [0, 2, 3, 5, 7, 8, 10],
	phrygian: [0, 1, 3, 5, 7, 8, 10],
	locrian: [0, 1, 3, 5, 6, 8, 10],
	wholetone: [0, 2, 4, 6, 8, 10],
	harmonicMinor: [0, 2, 3, 5, 7, 8, 11],
	phrygDom: [0, 1, 4, 5, 7, 8, 10],
	minPent: [0, 3, 5, 7, 10],
	majPent: [0, 2, 4, 7, 9],
};

interface MoodCfg {
	lo: number; // bpm at danger 0
	hi: number; // bpm at danger 1
	scales: string[];
	minDanger: number; // floor for percussion/arps (world overture is always marching)
}
const MOODS: Record<Mood, MoodCfg> = {
	town: { lo: 84, hi: 140, scales: ['ionian', 'mixolydian', 'majPent'], minDanger: 0 },
	wild: { lo: 78, hi: 148, scales: ['dorian', 'mixolydian', 'minPent', 'aeolian'], minDanger: 0 },
	dungeon: { lo: 70, hi: 150, scales: ['phrygian', 'locrian', 'aeolian'], minDanger: 0 },
	tower: { lo: 74, hi: 144, scales: ['aeolian', 'dorian', 'harmonicMinor'], minDanger: 0 },
	sky: { lo: 80, hi: 142, scales: ['lydian', 'majPent', 'ionian'], minDanger: 0 },
	sea: { lo: 72, hi: 140, scales: ['dorian', 'mixolydian', 'aeolian'], minDanger: 0 },
	underdark: { lo: 70, hi: 140, scales: ['wholetone', 'locrian', 'phrygDom'], minDanger: 0 },
	crypt: { lo: 70, hi: 140, scales: ['phrygian', 'harmonicMinor', 'aeolian'], minDanger: 0 },
	world: { lo: 96, hi: 140, scales: ['mixolydian', 'ionian', 'dorian'], minDanger: 0.35 },
	dead: { lo: 52, hi: 52, scales: ['aeolian'], minDanger: 0 },
};

interface GenreCfg {
	bpm: number;
	scales: string[];
	w: number; // chance the genre scale list overrides the mood's
	spb: number; // steps (16ths) per bar: 16 = 4/4, 12 = waltz / 6-8 feel
	swing: number;
	arpBias: number; // lowers the danger at which arpeggios start
	leadDay: number;
}
const GENRES: Record<string, GenreCfg> = {
	fantasy: { bpm: 1, scales: ['ionian', 'dorian', 'mixolydian'], w: 0.25, spb: 16, swing: 0, arpBias: 0, leadDay: 0.25 },
	mythic: { bpm: 0.96, scales: ['dorian', 'phrygian', 'aeolian'], w: 0.5, spb: 16, swing: 0, arpBias: 0, leadDay: 0.25 },
	steampunk: { bpm: 1, scales: ['mixolydian', 'dorian', 'harmonicMinor'], w: 0.5, spb: 12, swing: 0, arpBias: 0.05, leadDay: 0.25 },
	dieselpunk: { bpm: 1.02, scales: ['dorian', 'mixolydian', 'aeolian'], w: 0.5, spb: 16, swing: 0.28, arpBias: 0, leadDay: 0.25 },
	atompunk: { bpm: 1.05, scales: ['lydian', 'mixolydian', 'majPent'], w: 0.5, spb: 16, swing: 0.12, arpBias: 0.05, leadDay: 0.25 },
	cyberpunk: { bpm: 1.08, scales: ['aeolian', 'phrygian', 'dorian'], w: 0.6, spb: 16, swing: 0, arpBias: 0.2, leadDay: 0.125 },
	biopunk: { bpm: 0.98, scales: ['wholetone', 'phrygDom', 'locrian'], w: 0.5, spb: 16, swing: 0, arpBias: 0.05, leadDay: 0.125 },
	solarpunk: { bpm: 0.97, scales: ['majPent', 'lydian', 'ionian'], w: 0.6, spb: 16, swing: 0, arpBias: 0, leadDay: 0.25 },
};

// functional-harmony Markov chain over scale degrees (0 = tonic)
const TRANS: number[][] = [[3, 4, 5, 1, 2, 6], [4, 6, 4], [5, 3], [4, 0, 1, 4], [0, 5, 0], [3, 1, 4], [0, 2, 0]];

const ONSETS16 = [[0, 4, 8, 12], [0, 3, 6, 8, 12], [0, 2, 4, 8, 10, 12], [0, 4, 6, 8, 12, 14], [0, 6, 8, 14], [0, 2, 4, 6, 8, 12], [0, 3, 6, 10, 12], [0, 4, 8, 10, 12, 14]];
const ONSETS12 = [[0, 6], [0, 4, 6, 8], [0, 3, 6, 9], [0, 6, 8, 10], [0, 2, 4, 6, 9], [0, 3, 6, 8, 10]];
const IVS = [-2, -1, -1, 1, 1, 2, 0, 3, -3];
const ARPS: number[][] = [[0, 1, 2, 1], [0, 1, 2, 3, 2, 1], [0, 2, 1, 2], [0, 1, 2, 3]]; // offsets into [0,2,4,+octave]

// ---------- composition ----------
interface NoteEv { s: number; d: number; l: number }
interface Piece {
	key: string;
	dead: boolean;
	spb: number;
	scale: number[];
	base: number;
	chords: number[][]; // [section][bar] -> scale index of chord root
	mel: NoteEv[][][]; // [section][bar] -> notes
	form: number[]; // section order, each 4 bars
	bars: number;
	cfg: MoodCfg;
	gen: GenreCfg;
	arpStyle: number;
}

function pieceKey(m: QudMusic): string {
	return `${m.mood}|${m.genre}|${m.seed}|${m.biome | 0}|${(m.season ?? 0) | 0}`;
}

function snap(d: number, c: number, n: number): number {
	let best = c, bd = 1e9;
	for (let k = -1; k <= 2; k++) {
		for (const o of [0, 2, 4]) {
			const t = c + o + k * n;
			const dd = Math.abs(t - d);
			if (dd < bd) { bd = dd; best = t; }
		}
	}
	return best;
}

function makePiece(m: QudMusic): Piece {
	const key = pieceKey(m);
	const rng = mulberry32(hash(key));
	const pick = <T,>(a: T[]): T => a[Math.floor(rng() * a.length)];
	const gen = GENRES[m.genre] ?? GENRES.fantasy;
	const cfg = MOODS[m.mood] ?? MOODS.wild;
	const dead = m.mood === 'dead';
	const scaleName = dead ? 'aeolian' : rng() < gen.w ? pick(gen.scales) : pick(cfg.scales);
	const scale = dead ? SCALES[scaleName] : seasonScale(SCALES[scaleName], (m.season ?? 0) | 0, rng());
	const n = scale.length;
	const spb = dead ? 16 : gen.spb;
	const base = 45 + ((hash(key + 'k') + (m.biome | 0) * 5 + 12 + (dead ? 0 : SEASON_SHIFT[(m.season ?? 0) & 3])) % 12);
	const center = Math.round(n * 0.9);

	if (dead) {
		const ds = [n + 4, n + 3, n + 2, n + 1, n, n - 1, n - 2, n - 3];
		const mel: NoteEv[][] = [0, 1, 2, 3].map((b) =>
			b < 3
				? [{ s: 0, d: ds[b * 2], l: 8 }, { s: 8, d: ds[b * 2 + 1], l: 8 }]
				: [{ s: 0, d: ds[6], l: 6 }, { s: 6, d: ds[7], l: 10 }]);
		return { key, dead, spb, scale, base, chords: [[0, 5, 3, 0]], mel: [mel], form: [0, 0, 0, 0], bars: 4, cfg, gen, arpStyle: 0 };
	}

	const progression = (start: number, end: number): number[] => {
		const p = [start];
		for (let i = 1; i < 3; i++) p.push(pick(TRANS[p[i - 1] % TRANS.length]) % n);
		p.push(end % n);
		return p;
	};
	const onsets = spb === 12 ? ONSETS12 : ONSETS16;
	const motif = () => {
		const on = pick(onsets);
		const iv = [0];
		for (let i = 1; i < on.length; i++) {
			const prev = iv[i - 1] === 0 ? 0 : 0;
			let s = pick(IVS);
			if (i > 1 && Math.abs(iv[i - 1] - (iv[i - 2] ?? 0)) >= 3) s = s > 0 ? -1 : 1; // step back after a leap
			iv.push(iv[i - 1] + s + prev);
		}
		return { on, iv };
	};
	const realize = (mo: { on: number[]; iv: number[] }, start: number, chord: number, tweak: number): NoteEv[] => {
		const out: NoteEv[] = [];
		for (let i = 0; i < mo.on.length; i++) {
			const s = mo.on[i];
			let d = start + mo.iv[i];
			if (i === mo.on.length - 1) d += tweak;
			d = clamp(d, 0, Math.floor(n * 1.8));
			if (s % (spb / 2) === 0) d = snap(d, chord, n); // strong beats land on chord tones
			out.push({ s, d, l: Math.min(6, (mo.on[i + 1] ?? spb) - s) });
		}
		return out;
	};
	const cadence = (chord: number): NoteEv[] => {
		const t2 = snap(center + Math.floor(rng() * 3) - 1, chord, n);
		const t1 = t2 + (rng() < 0.5 ? 1 : -1);
		return [{ s: 0, d: t1, l: spb / 2 - 2 }, { s: spb / 2, d: t2, l: spb / 2 }];
	};
	const section = (chords: number[]): NoteEv[][] => {
		const mo = motif();
		const st0 = snap(center + Math.floor(rng() * 3) - 1, chords[0], n);
		const seq = rng() < 0.5 ? 2 : -2;
		return [
			realize(mo, st0, chords[0], 0),
			realize(mo, st0, chords[1], rng() < 0.5 ? 1 : -1),
			realize(mo, st0 + seq, chords[2], 0),
			cadence(chords[3]),
		];
	};

	const chA = progression(0, 4);
	const chA2 = chA.slice(0, 3).concat([0]);
	const chB = progression(pick([3, 5]) % n, 4);
	const melA = section(chA);
	const melA2 = melA.slice(0, 3).concat([cadence(0)]); // A' = A with a resolving ending
	const melB = section(chB);

	return {
		key, dead, spb, scale, base,
		chords: [chA, chA2, chB],
		mel: [melA, melA2, melB],
		form: [0, 1, 2, 0],
		bars: 16, cfg, gen,
		arpStyle: Math.floor(rng() * ARPS.length),
	};
}

// ---------- engine ----------
interface Player {
	piece: Piece;
	bus: GainNode;
	hat: BiquadFilterNode;
	snr: BiquadFilterNode;
	bar: number;
	step: number;
	nextT: number;
	done: boolean;
}

export function createMusic(): MusicHandle {
	let ctx: AudioContext | null = null;
	let master: GainNode;
	let tone: BiquadFilterNode;
	let rumble: BiquadFilterNode; // thunder
	let washF: BiquadFilterNode; // sand / ash / wind noise wash
	let lastBolt = -1;
	let waves: Record<string, PeriodicWave> = {};
	let noiseLong: AudioBuffer;
	let noiseShort: AudioBuffer;
	let timer: ReturnType<typeof setInterval> | null = null;
	let enabled = true;
	let volume = 0.35;
	let disposed = false;
	let st: QudMusic | null = null;
	let cur: Player | null = null;
	let activeKey = '';
	let pendingKey: string | null = null;
	let bpm = 80;
	let dNow = 0;
	let lastCut = -1;
	const fading = new Set<ReturnType<typeof setTimeout>>();

	function pulseWave(c: AudioContext, duty: number): PeriodicWave {
		const N = 48;
		const re = new Float32Array(N + 1), im = new Float32Array(N + 1);
		for (let k = 1; k <= N; k++) {
			re[k] = Math.sin(2 * Math.PI * k * duty) / (Math.PI * k);
			im[k] = (1 - Math.cos(2 * Math.PI * k * duty)) / (Math.PI * k);
		}
		return c.createPeriodicWave(re, im);
	}
	function lfsrBuffer(c: AudioContext, bits: 15 | 7, len: number, hold: number): AudioBuffer {
		const buf = c.createBuffer(1, len, c.sampleRate);
		const d = buf.getChannelData(0);
		let reg = 1;
		let out = 1;
		for (let i = 0; i < len; i++) {
			if (i % hold === 0) {
				const bit = (reg ^ (reg >> 1)) & 1;
				reg = (reg >> 1) | (bit << (bits - 1));
				out = reg & 1 ? 1 : -1;
			}
			d[i] = out;
		}
		return buf;
	}

	function ensure(): boolean {
		if (disposed) return false;
		if (!ctx) {
			try {
				const AC: typeof AudioContext | undefined =
					window.AudioContext ?? (window as unknown as { webkitAudioContext?: typeof AudioContext }).webkitAudioContext;
				if (!AC) return false;
				const c = new AC();
				ctx = c;
				master = c.createGain();
				master.gain.value = 0;
				const comp = c.createDynamicsCompressor();
				comp.threshold.value = -16;
				comp.knee.value = 12;
				comp.ratio.value = 6;
				comp.attack.value = 0.004;
				comp.release.value = 0.2;
				tone = c.createBiquadFilter();
				tone.type = 'lowpass';
				tone.frequency.value = 6000;
				tone.Q.value = 0.5;
				tone.connect(comp);
				rumble = c.createBiquadFilter();
				rumble.type = 'lowpass';
				rumble.frequency.value = 220;
				rumble.Q.value = 0.7;
				rumble.connect(tone);
				washF = c.createBiquadFilter();
				washF.type = 'bandpass';
				washF.frequency.value = 900;
				washF.Q.value = 0.6;
				washF.connect(tone);
				comp.connect(master);
				master.connect(c.destination);
				waves = { '0.125': pulseWave(c, 0.125), '0.25': pulseWave(c, 0.25), '0.5': pulseWave(c, 0.5) };
				noiseLong = lfsrBuffer(c, 15, 32768, 2);
				noiseShort = lfsrBuffer(c, 7, 8192, 1);
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
		if (!ctx) return;
		master.gain.setTargetAtTime(enabled ? volume * 0.8 : 0, ctx.currentTime, 0.06);
	}

	// ----- voices -----
	function pulse(p: Player, duty: number, midi: number, t: number, dur: number, vel: number, detune: number) {
		const c = ctx!;
		const o = c.createOscillator();
		const g = c.createGain();
		o.setPeriodicWave(waves[String(duty)] ?? waves['0.25']);
		o.frequency.value = mtof(midi);
		o.detune.value = detune;
		const v = q16(vel);
		const a = 0.004;
		g.gain.setValueAtTime(0, t);
		g.gain.linearRampToValueAtTime(v, t + a);
		// stepped decay for the crunchy 8-bit envelope
		g.gain.setValueAtTime(q16(vel * 0.8), t + dur * 0.4);
		g.gain.setValueAtTime(q16(vel * 0.6), t + dur * 0.75);
		g.gain.linearRampToValueAtTime(0, t + dur + 0.03);
		o.connect(g).connect(p.bus);
		o.start(t);
		o.stop(t + dur + 0.05);
		o.onended = () => g.disconnect();
	}
	function tri(p: Player, midi: number, t: number, dur: number, vel: number) {
		const c = ctx!;
		const o = c.createOscillator();
		const g = c.createGain();
		o.type = 'triangle';
		o.frequency.value = mtof(midi);
		const v = q16(vel);
		g.gain.setValueAtTime(0, t);
		g.gain.linearRampToValueAtTime(v, t + 0.005);
		g.gain.setValueAtTime(v, t + dur * 0.8);
		g.gain.linearRampToValueAtTime(0, t + dur + 0.02);
		o.connect(g).connect(p.bus);
		o.start(t);
		o.stop(t + dur + 0.04);
		o.onended = () => g.disconnect();
	}
	function noise(dest: AudioNode, short: boolean, t: number, dur: number, vel: number, rate: number) {
		const c = ctx!;
		const s = c.createBufferSource();
		const g = c.createGain();
		s.buffer = short ? noiseShort : noiseLong;
		s.loop = true;
		s.playbackRate.value = rate;
		g.gain.setValueAtTime(q16(vel), t);
		g.gain.setValueAtTime(q16(vel * 0.5), t + dur * 0.4);
		g.gain.linearRampToValueAtTime(0, t + dur);
		s.connect(g).connect(dest);
		s.start(t);
		s.stop(t + dur + 0.02);
		s.onended = () => g.disconnect();
	}
	function kick(dest: AudioNode, t: number, vel: number, len = 0.14) {
		const c = ctx!;
		const o = c.createOscillator();
		const g = c.createGain();
		o.type = 'sine';
		o.frequency.setValueAtTime(len > 0.14 ? 90 : 150, t);
		o.frequency.exponentialRampToValueAtTime(40, t + len * 0.8);
		g.gain.setValueAtTime(q16(vel), t);
		g.gain.linearRampToValueAtTime(0, t + len);
		o.connect(g).connect(dest);
		o.start(t);
		o.stop(t + len + 0.03);
		o.onended = () => g.disconnect();
	}

	/** a long smooth noise swell (weather wash); not quantised so it stays soft */
	function wash(t: number, dur: number, vel: number, rate: number) {
		const c = ctx!;
		const s = c.createBufferSource();
		const g = c.createGain();
		s.buffer = noiseLong;
		s.loop = true;
		s.playbackRate.value = rate;
		g.gain.setValueAtTime(0, t);
		g.gain.linearRampToValueAtTime(vel, t + dur * 0.45);
		g.gain.linearRampToValueAtTime(0, t + dur);
		s.connect(g).connect(washF);
		s.start(t);
		s.stop(t + dur + 0.02);
		s.onended = () => g.disconnect();
	}
	/** sustained low triangle with a slow swell (fog drone) */
	function drone(p: Player, midi: number, t: number, dur: number, vel: number) {
		const c = ctx!;
		const o = c.createOscillator();
		const g = c.createGain();
		o.type = 'triangle';
		o.frequency.value = mtof(midi);
		g.gain.setValueAtTime(0, t);
		g.gain.linearRampToValueAtTime(vel, t + dur * 0.35);
		g.gain.setValueAtTime(vel, t + dur * 0.7);
		g.gain.linearRampToValueAtTime(0, t + dur + 0.05);
		o.connect(g).connect(p.bus);
		o.start(t);
		o.stop(t + dur + 0.1);
		o.onended = () => g.disconnect();
	}
	/** a distant thunder hit: low noise rumble + a soft sub thump */
	function thunder() {
		const c = ctx;
		if (!c || !enabled) return;
		const t = c.currentTime + 0.02 + Math.random() * 0.25;
		const s = c.createBufferSource();
		const g = c.createGain();
		s.buffer = noiseLong;
		s.loop = true;
		s.playbackRate.value = 0.35;
		g.gain.setValueAtTime(0, t);
		g.gain.linearRampToValueAtTime(0.5, t + 0.04);
		g.gain.exponentialRampToValueAtTime(0.18, t + 0.6);
		g.gain.exponentialRampToValueAtTime(0.001, t + 2.4);
		s.connect(g).connect(rumble);
		s.start(t);
		s.stop(t + 2.5);
		s.onended = () => g.disconnect();
		kick(tone, t, 0.3, 0.45);
	}

	// ----- player lifecycle -----
	function newPlayer(piece: Piece, t: number): Player {
		const c = ctx!;
		const bus = c.createGain();
		bus.gain.setValueAtTime(0, t);
		bus.gain.linearRampToValueAtTime(1, t + 0.12);
		bus.connect(tone);
		const hat = c.createBiquadFilter();
		hat.type = 'highpass';
		hat.frequency.value = 7000;
		hat.connect(bus);
		const snr = c.createBiquadFilter();
		snr.type = 'bandpass';
		snr.frequency.value = 1800;
		snr.Q.value = 0.8;
		snr.connect(bus);
		return { piece, bus, hat, snr, bar: 0, step: 0, nextT: t, done: false };
	}
	function retire(p: Player, t: number, fade: number) {
		const c = ctx;
		if (!c) return;
		p.bus.gain.cancelScheduledValues(t);
		p.bus.gain.setValueAtTime(1, t);
		p.bus.gain.linearRampToValueAtTime(0, t + fade);
		const h = setTimeout(() => {
			fading.delete(h);
			try { p.bus.disconnect(); p.hat.disconnect(); p.snr.disconnect(); } catch { /* already gone */ }
		}, Math.max(0, (t + fade - c.currentTime) * 1000) + 600);
		fading.add(h);
	}

	function targetBpm(p: Piece, d: number): number {
		if (p.dead) return 52;
		const x = Math.pow(clamp(d, 0, 1), 0.9);
		return (p.cfg.lo + (p.cfg.hi - p.cfg.lo) * x) * p.gen.bpm * (st && !st.day ? 0.96 : 1) * wxCfg(st?.wx).tempo;
	}

	function scheduleStep(p: Player): number {
		const pc = p.piece;
		const m = st!;
		const spb = pc.spb;
		const n = pc.scale.length;
		const pitch = (idx: number) => pc.scale[((idx % n) + n) % n] + 12 * Math.floor(idx / n);
		const seed = hash(pc.key);
		const wx = pc.dead ? WX0 : wxCfg(m.wx);

		// continuous adaptation
		const dTarget = pc.dead ? 0 : Math.max(clamp(m.danger, 0, 1), pc.cfg.minDanger);
		dNow += (dTarget - dNow) * 0.1;
		const d = dNow;
		bpm += (targetBpm(pc, d) - bpm) * 0.08;
		const dur = 60 / bpm / 4;

		const t0 = p.nextT;
		const sw = pc.gen.swing;
		const t = p.step % 4 === 2 && spb === 16 && sw ? t0 + sw * dur : t0;
		const step = p.step;
		const bar = p.bar;
		const sec = pc.form[Math.floor(bar / 4) % 4];
		const bi = bar % 4;
		const chord = pc.chords[sec][bi];
		const day = m.day;
		const leadBase = pc.base + 12 + (day ? 0 : -0);
		const leadDuty = pc.dead ? 0.5 : day ? pc.gen.leadDay : 0.5;
		const half = spb / 2;

		// lead
		for (const ev of pc.mel[sec][bi]) {
			if (ev.s !== step) continue;
			const strong = ev.s % half === 0;
			if (!strong && !pc.dead && hrand(bar, step, seed) > 0.55 + 0.45 * d) continue;
			if (!strong && wx.sparse && hrand(bar, step, seed + 5) < wx.sparse) continue;
			const len = (ev.l + (pc.dead ? 0 : 0)) * dur * (pc.dead ? 1 : 0.92);
			pulse(p, leadDuty, leadBase + wx.lift + pitch(ev.d) - (day ? 0 : 12), t, len, pc.dead ? 0.12 : (0.13 + 0.03 * d) * wx.soft, hrand(bar, step, 7) * 8 - 4);
		}

		// harmony voice: pad when calm, arpeggio as danger rises
		const arpAt = 0.35 - pc.gen.arpBias;
		if (d < arpAt) {
			if (step === 0) pulse(p, 0.25, pc.base + 12 + wx.lift + pitch(chord + 2), t, half * dur * 0.95, 0.07 * wx.soft, 6);
			if (step === half && !(wx.sparse && hrand(bar, step, seed + 6) < wx.sparse)) pulse(p, 0.25, pc.base + 12 + wx.lift + pitch(chord + 4), t, half * dur * 0.95, 0.07 * wx.soft, 6);
		} else {
			const div = d < 0.7 ? 2 : 1;
			if (step % div === 0 && !(wx.sparse && hrand(bar, step, seed + 6) < wx.sparse)) {
				const pat = ARPS[pc.arpStyle];
				const k = pat[(step / div) % pat.length];
				const idx = k === 3 ? chord + n : chord + k * 2;
				pulse(p, 0.125, pc.base + 12 + wx.lift + pitch(idx), t, dur * div * 0.8, (0.07 + 0.03 * d) * wx.soft, 5);
			}
		}

		// bass
		const bdiv = spb === 16 ? (d < 0.3 ? 8 : d < 0.6 ? 4 : 2) : d < 0.3 ? 6 : d < 0.6 ? 3 : 2;
		if (step % bdiv === 0) {
			const k = step / bdiv;
			const offs = [0, 0, 4, 0, 0, 0, 4, n];
			const idx = chord + (bdiv === 2 ? (k % 2 ? n : 0) : offs[k % offs.length]);
			tri(p, pc.base - 12 + pitch(idx), t, bdiv * dur * 0.9, 0.34);
		}

		// percussion
		if (!pc.dead) {
			if (d > 0.12) {
				const kicks = spb === 16 ? (d > 0.4 ? [0, 8] : [0]) : [0];
				if (kicks.includes(step) || (d > 0.65 && spb === 16 && step % 2 === 0 && step !== 4 && step !== 12 && hrand(bar, step, seed + 1) < 0.25))
					kick(p.bus, t, 0.55);
			}
			if (d > 0.3 && step === (spb === 16 ? 4 : 6 )) noise(p.snr, false, t, 0.12, 0.34, 1.3);
			if (d > 0.3 && spb === 16 && step === 12) noise(p.snr, false, t, 0.12, 0.34, 1.3);
			if (d > 0.2) {
				const hdiv = spb === 16 ? (d < 0.5 ? 4 : 2) : d < 0.5 ? 3 : 2;
				const fill = d > 0.8 && step % hdiv !== 0 && hrand(bar, step, seed + 2) < 0.4;
				if (step % hdiv === 0 || fill) noise(p.hat, true, t, step % (hdiv * 2) === hdiv ? 0.07 : 0.035, fill ? 0.09 : 0.14, 1);
			}
		}

		// weather colour
		if (!pc.dead) {
			// rain patter: soft, slightly random 16th hats on top of whatever percussion plays
			if (wx.patter && hrand(bar, step, seed + 9) < wx.patter * (step % 2 ? 0.55 : 0.9))
				noise(p.hat, true, t, 0.025, 0.05 + 0.04 * wx.patter, 1.6 + hrand(bar, step, seed + 10) * 0.6);
			if (wx.drone && step === 0) drone(p, pc.base - 12 + pitch(chord), t, spb * dur * 1.05, 0.11 * wx.drone);
			if (wx.wash && step === 0) {
				washF.frequency.setTargetAtTime(wx.washHz, t, 0.5);
				wash(t, spb * dur * 1.1, 0.07 * wx.wash, 0.7 + hrand(bar, 0, seed + 11) * 0.5);
			}
		}

		// heartbeat at low hp (bypasses the piece bus so it never crossfades away)
		if (!pc.dead && m.hp < 0.4 && step % half === 0) {
			const sev = clamp((0.4 - m.hp) / 0.4, 0, 1);
			kick(tone, t, 0.35 + 0.4 * sev, 0.2);
			kick(tone, t + dur * 3, 0.2 + 0.3 * sev, 0.16);
		}

		// advance
		p.nextT = t0 + dur;
		p.step++;
		if (p.step >= spb) {
			p.step = 0;
			p.bar++;
			if (pc.dead) {
				if (p.bar >= pc.bars) {
					p.done = true;
					retire(p, p.nextT - dur * spb * 0.5, dur * spb * 0.5);
				}
			} else if (p.bar >= pc.bars) p.bar = 0;
		}
		return p.nextT;
	}

	function tick() {
		const c = ctx;
		if (!c || !enabled || !st) return;
		const now = c.currentTime;

		// brightness: day opens the filter, night and depth darken it
		const cut = (st.day ? 7000 : 2600) * Math.max(0.4, 1 - Math.max(0, st.depth) * 0.04) * wxCfg(st.wx).cut;
		if (Math.abs(cut - lastCut) > 1) {
			tone.frequency.setTargetAtTime(cut, now, 0.4);
			lastCut = cut;
		}

		if (!cur || cur.done) {
			if (pendingKey !== null) {
				cur = newPlayer(makePiece(st), now + 0.05);
				activeKey = pendingKey;
				pendingKey = null;
				bpm = targetBpm(cur.piece, dNow);
			} else return;
		}
		let guard = 0;
		while (cur && !cur.done && cur.nextT < now + 0.12 && guard++ < 64) {
			if (cur.nextT < now - 0.1) cur.nextT = now + 0.03; // tab was throttled: don't burst
			if (cur.step === 0 && pendingKey !== null) {
				// swap to the new piece exactly on the bar line
				const at = cur.nextT;
				retire(cur, at, 0.5);
				cur = newPlayer(makePiece(st), at);
				activeKey = pendingKey;
				pendingKey = null;
			}
			scheduleStep(cur);
		}
	}

	function startTimer() {
		if (timer === null) timer = setInterval(tick, 25);
	}
	function stopTimer() {
		if (timer !== null) { clearInterval(timer); timer = null; }
	}

	return {
		setState(m: QudMusic) {
			if (disposed) return;
			st = m;
			const bolt = m.bolt ?? 0;
			if (lastBolt >= 0 && bolt > lastBolt && enabled && ensure()) thunder();
			lastBolt = bolt;
			const key = pieceKey(m);
			if (key !== activeKey && key !== pendingKey) pendingKey = key;
			else if (key === activeKey) pendingKey = null;
			if (!enabled) return;
			if (!ensure()) return;
			startTimer();
		},
		setVolume(v: number) {
			volume = clamp(Number.isFinite(v) ? v : 0.35, 0, 1);
			applyVolume();
		},
		setEnabled(on: boolean) {
			if (disposed) return;
			enabled = on;
			if (on) {
				if (!ensure()) return;
				applyVolume();
				if (cur) cur.nextT = Math.max(cur.nextT, ctx!.currentTime + 0.05);
				startTimer();
			} else {
				stopTimer();
				applyVolume();
				const c = ctx;
				if (c) setTimeout(() => { if (!enabled && c.state === 'running') c.suspend().catch(() => {}); }, 400);
			}
		},
		dispose() {
			disposed = true;
			stopTimer();
			fading.forEach(clearTimeout);
			fading.clear();
			cur = null;
			const c = ctx;
			ctx = null;
			if (c) c.close().catch(() => {});
		},
	};
}
