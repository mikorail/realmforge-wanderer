/**
 * Species and creature voices plus speech blips (shell side, cosmetic). Fed by `frame.vox`
 * (engine/voxinfo.go). Everything is a formant synthesis on the shared AudioContext: a sawtooth or pulse
 * source (plus noise) into three band-pass formant filters, with a per-species table (pitch, formant shift,
 * breathiness, growl, special colour) for the 19 Realmforge species ids of actorsprites.ts and a hash for
 * unknown ones. Creatures (no species) sound by body plan and bestiary size. Nine kinds (pain death alert
 * greet laugh growl chatter battlecry sigh), each a recipe of 1 to 4 short syllables of 80 to 600 ms.
 * `speak` events and `blips()` make an Animalese-style stutter of syllables while a dialogue line appears.
 *
 * Loudness: voices pass a soft ceiling of about -24 dBFS, blips about -33 dBFS (before the master volume,
 * which is at most 1), both fade in over 1.6 s after creation or enabling. At most 4 voices sound at once,
 * events are throttled per id, per kind and per half second, and far voices are dropped. Nothing runs while
 * disabled or after dispose; every source is stopped at the end of its syllable.
 */

export type VoxKind = 'pain' | 'death' | 'alert' | 'greet' | 'laugh' | 'growl' | 'chatter' | 'battlecry' | 'sigh' | 'speak';

export interface VoxEvent {
	kind: VoxKind;
	species?: string;
	body?: string;
	tpl?: string;
	size?: number;
	x: number;
	y: number;
	id: number;
	player?: boolean;
	role?: string;
	caste?: string;
	boss?: boolean;
	pitchSeed: number;
	register?: string;
	mood?: string;
	text?: string;
	len?: number;
}

export interface VoiceSpeech {
	species?: string;
	register?: string;
	mood?: string;
	text: string;
	id?: number;
}

export interface VoicesHandle {
	/** voice the events of one frame; `speak` events go to blips() when blips are on */
	play(events: VoxEvent[] | undefined, listener: { x: number; y: number }): void;
	/** a dialogue line appeared */
	blips(speech: VoiceSpeech): void;
	/** master on/off (the sfx pref) */
	setEnabled(on: boolean): void;
	/** 0..1, multiplies both layers */
	setVolume(v: number): void;
	/** creature and species voices on/off (pref `voices`, default true) */
	setVoices(on: boolean): void;
	/** speech blips on/off (pref `speechBlips`, default false) */
	setBlips(on: boolean): void;
	dispose(): void;
}

// ---------------------------------------------------------------- tables

/** formants F1 F2 F3 (Hz) of a male-ish vowel; species shift scales them */
const VOW: Record<string, [number, number, number]> = {
	a: [730, 1090, 2440],
	e: [530, 1840, 2480],
	i: [300, 2200, 2950],
	o: [570, 840, 2410],
	u: [320, 870, 2240],
};

interface Sp {
	f0: number; // base fundamental, Hz
	fs: number; // formant shift
	breath: number; // noise share 0..1
	growl: number; // amplitude-modulation depth 0..1
	vow: string; // preferred vowels
	choir?: number; // second oscillator detune, cents
	vib?: [number, number]; // vibrato Hz, cents
	click?: boolean; // chitin clicks
	hollow?: boolean; // low Q, dark
	pulse?: number; // pulse duty instead of saw
}

const SP: Record<string, Sp> = {
	human: { f0: 125, fs: 1, breath: 0.08, growl: 0, vow: 'aeiou' },
	elf: { f0: 200, fs: 1.13, breath: 0.12, growl: 0, vow: 'eiau', vib: [5, 14] },
	void_elf: { f0: 165, fs: 1.08, breath: 0.3, growl: 0, vow: 'eiu', choir: 14, vib: [4, 10] },
	dwarf: { f0: 82, fs: 0.88, breath: 0.1, growl: 0.28, vow: 'aouE'.toLowerCase() },
	halfling: { f0: 178, fs: 1.05, breath: 0.08, growl: 0, vow: 'aeio' },
	orc: { f0: 68, fs: 0.84, breath: 0.15, growl: 0.6, vow: 'aou' },
	goblin: { f0: 210, fs: 1.04, breath: 0.12, growl: 0.22, vow: 'eai', pulse: 0.3 },
	ogor: { f0: 58, fs: 0.8, breath: 0.15, growl: 0.5, vow: 'ou' },
	troll: { f0: 54, fs: 0.78, breath: 0.2, growl: 0.7, vow: 'ou' },
	giant: { f0: 42, fs: 0.7, breath: 0.12, growl: 0.4, vow: 'oua' },
	ratfolk: { f0: 235, fs: 1.1, breath: 0.1, growl: 0.1, vow: 'ei', pulse: 0.25 },
	beastman: { f0: 96, fs: 0.95, breath: 0.22, growl: 0.55, vow: 'aoe' },
	dragonborn: { f0: 86, fs: 0.92, breath: 0.45, growl: 0.35, vow: 'aoe' },
	lizardfolk: { f0: 102, fs: 1, breath: 0.42, growl: 0.12, vow: 'ei' },
	saurian: { f0: 84, fs: 0.92, breath: 0.4, growl: 0.4, vow: 'aoe' },
	merfolk: { f0: 172, fs: 1.04, breath: 0.1, growl: 0, vow: 'uoi', vib: [8, 55] },
	starborn: { f0: 150, fs: 1.02, breath: 0.2, growl: 0, vow: 'aou', choir: 22, vib: [4.5, 18] },
	celestial: { f0: 160, fs: 1.05, breath: 0.18, growl: 0, vow: 'aou', choir: 20, vib: [4.5, 16] },
	tomb_revenant: { f0: 74, fs: 0.8, breath: 0.6, growl: 0.1, vow: 'ou', hollow: true },
	tomb_undead: { f0: 74, fs: 0.8, breath: 0.6, growl: 0.1, vow: 'ou', hollow: true },
	warp_fiend: { f0: 72, fs: 0.9, breath: 0.2, growl: 0.55, vow: 'aoe', choir: 55 },
	chitin_swarm: { f0: 150, fs: 1.1, breath: 0.35, growl: 0, vow: 'ei', click: true },
};
SP.tomb = SP.tomb_undead;
SP.revenant = SP.tomb_undead;
SP.dragon = SP.dragonborn;
SP.star = SP.starborn;
SP.voidelf = SP.void_elf;

function strHash(s: string): number {
	let h = 2166136261;
	for (let i = 0; i < s.length; i++) h = Math.imul(h ^ s.charCodeAt(i), 16777619);
	return h >>> 0;
}

/** a known species, or a stable voice hashed from the id */
function spOf(id: string | undefined): Sp {
	const key = (id ?? 'human').toLowerCase().replace(/[\s-]+/g, '_');
	const known = SP[key];
	if (known) return known;
	const h = strHash(key);
	return {
		f0: 60 + (h % 170),
		fs: 0.8 + ((h >>> 8) % 40) / 100,
		breath: ((h >>> 12) % 5) * 0.08,
		growl: ((h >>> 15) % 4) * 0.15,
		vow: ['aeiou', 'aou', 'eia', 'oue'][(h >>> 18) & 3],
		choir: (h >>> 20) & 1 ? 12 : undefined,
	};
}

// ---------------------------------------------------------------- syllables

interface Syl {
	t: number; // start offset, s
	dur: number;
	f: [number, number]; // fundamental start/end (Hz); for humanoid recipes a multiplier of the base
	v: [string, string]; // vowels (ignored with bands)
	amp: number;
	src?: 'saw' | 'pulse' | 'noise' | 'sine';
	duty?: number;
	noise?: number; // noise share of the mix, 0..1 (src 'noise' = 1)
	am?: [number, number]; // rate Hz, depth 0..1
	sq?: boolean; // square AM (clicks)
	q?: number;
	fs?: number; // formant shift
	choir?: number;
	vib?: [number, number];
	bands?: Array<[number, number, number]>; // freq, Q, gain: replaces the vowel formants
	sweep?: number; // bands end / start frequency ratio
}

const syl = (t: number, dur: number, f0: number, f1: number, v0: string, v1: string, amp = 0.8, extra: Partial<Syl> = {}): Syl => ({
	t,
	dur,
	f: [f0, f1],
	v: [v0, v1],
	amp,
	...extra,
});

/** recipes for a speaking species: f in multiples of the base pitch */
function personSyls(kind: Exclude<VoxKind, 'speak'>, sp: Sp, h: number): Syl[] {
	const pickV = (v: string) => (sp.vow.includes(v) ? v : sp.vow[h % sp.vow.length]);
	let r: Syl[];
	switch (kind) {
		case 'pain':
			r = [syl(0, 0.22, 1.4, 0.95, 'a', 'o', 0.9)];
			break;
		case 'death':
			r = [syl(0, 0.3, 1.15, 0.85, 'a', 'o', 0.9), syl(0.28, 0.32, 0.8, 0.4, 'o', 'u', 0.7, { noise: 0.4 })];
			break;
		case 'alert':
			r = [syl(0, 0.2, 0.9, 1.45, 'e', 'a', 1)];
			break;
		case 'greet':
			r = [syl(0, 0.12, 1, 1.05, 'o', 'o', 0.7), syl(0.15, 0.22, 1.25, 1, 'e', 'a', 0.7)];
			break;
		case 'laugh':
			r = [0, 1, 2, 3].map((i) => syl(i * 0.13, 0.09, 1.25 - i * 0.07, 1.1 - i * 0.07, 'a', 'a', 0.7 - i * 0.08, { noise: 0.35 }));
			break;
		case 'growl':
			r = [syl(0, 0.5, 0.75, 0.62, 'o', 'u', 0.9, { am: [38, 0.55], noise: 0.15 })];
			break;
		case 'chatter':
			r = [0, 1, 2].map((i) => {
				const vs = 'aeiou';
				return syl(i * 0.1, 0.07, 1 + ((h >>> (i * 3)) % 5) * 0.05, 1, vs[(h >>> (i * 2)) % 5], vs[(h >>> (i * 2 + 1)) % 5], 0.55);
			});
			break;
		case 'battlecry':
			r = [syl(0, 0.46, 1, 1.35, 'a', 'o', 1, { am: [45, 0.25] }), syl(0.36, 0.2, 1.3, 0.95, 'o', 'a', 0.8)];
			break;
		default: // sigh
			r = [syl(0, 0.55, 0.9, 0.62, 'a', 'o', 0.6, { noise: 0.8 })];
	}
	for (const s of r) {
		s.v = [pickV(s.v[0]), pickV(s.v[1])];
		s.noise = Math.min(0.92, (s.noise ?? 0) + sp.breath);
		if (sp.growl > 0 && !s.am) s.am = [34 + (h % 20), sp.growl * 0.6];
		else if (sp.growl > 0 && s.am) s.am = [s.am[0], Math.min(0.9, s.am[1] + sp.growl * 0.35)];
		if (sp.click) {
			s.am = [36, 0.95];
			s.sq = true;
		}
		if (sp.pulse) {
			s.src = 'pulse';
			s.duty = sp.pulse;
		}
		s.fs = sp.fs;
		s.choir = sp.choir;
		s.vib = sp.vib;
		s.q = sp.hollow ? 3 : 8;
	}
	return r;
}

const HISS: Array<[number, number, number]> = [[4200, 1.5, 1.2], [6500, 2, 0.8], [2800, 1.2, 0.4]];

/** recipes for a creature by body plan: f absolute Hz, already scaled by size */
function creatureSyls(body: string, kind: Exclude<VoxKind, 'speak'>, size: number, h: number): Syl[] {
	const sc = Math.pow(2, (4 - Math.max(1, Math.min(7, size))) * 0.34); // size 1 high .. 7 low
	const role = kind === 'pain' ? 'hurt' : kind === 'death' ? 'die' : kind === 'alert' || kind === 'growl' || kind === 'battlecry' ? 'fight' : 'idle';
	const big = kind === 'battlecry' ? 1.25 : 1;
	switch (body) {
		case 'quadruped':
			if (role === 'hurt') return [syl(0, 0.18, 330 * sc, 240 * sc, 'e', 'a', 0.9)];
			if (role === 'die') return [syl(0, 0.5, 260 * sc, 90 * sc, 'a', 'u', 0.85, { noise: 0.15 })];
			if (kind === 'growl') return [syl(0, 0.55, 95 * sc, 80 * sc, 'o', 'u', 0.9, { am: [38, 0.6], noise: 0.18 })];
			if (role === 'fight') return [syl(0, 0.13, 260 * sc, 150 * sc, 'a', 'o', 1, { noise: 0.2 }), syl(0.2, 0.13, 250 * sc, 140 * sc, 'a', 'o', 0.8, { noise: 0.2 })];
			if (kind === 'sigh') return [syl(0, 0.4, 150 * sc, 100 * sc, 'o', 'u', 0.5, { noise: 0.8 })];
			if (kind === 'laugh') return [0, 1, 2, 3].map((i) => syl(i * 0.09, 0.05, 200 * sc, 160 * sc, 'a', 'a', 0.5, { noise: 0.6 }));
			return [syl(0, 0.12, 190 * sc, 150 * sc, 'o', 'o', 0.45, { noise: 0.4 })];
		case 'saurian':
			if (role === 'hurt') return [syl(0, 0.22, 140 * sc, 95 * sc, 'a', 'o', 0.9, { noise: 0.5, am: [30, 0.4] })];
			if (role === 'die') return [syl(0, 0.6, 130 * sc, 40 * sc, 'a', 'u', 0.9, { am: [24, 0.6], noise: 0.4 })];
			if (kind === 'battlecry' || kind === 'growl') return [syl(0, 0.55, 78 * sc, 62 * sc * big, 'a', 'o', 1, { am: [22, 0.7], noise: 0.3 })];
			if (role === 'fight') return [syl(0, 0.35, 1, 1, 'a', 'a', 0.9, { src: 'noise', bands: HISS, sweep: 0.8 }), syl(0.25, 0.2, 90 * sc, 70 * sc, 'a', 'o', 0.7, { am: [24, 0.5] })];
			return [syl(0, kind === 'sigh' ? 0.5 : 0.22, 1, 1, 'a', 'a', 0.5, { src: 'noise', bands: HISS, sweep: 0.9 })];
		case 'avian': {
			const hi = 1900 / Math.sqrt(sc);
			if (role === 'hurt') return [syl(0, 0.16, hi * 1.2, hi * 0.8, 'i', 'a', 0.85, { src: 'pulse', duty: 0.3 })];
			if (role === 'die') return [syl(0, 0.5, hi, hi * 0.35, 'i', 'a', 0.85, { src: 'pulse', duty: 0.3 })];
			if (role === 'fight') return [syl(0, 0.3 * big, hi * 0.8, hi * 1.5, 'i', 'a', 1, { src: 'pulse', duty: 0.25, noise: 0.2, am: [60, 0.3] })];
			if (kind === 'sigh') return [syl(0, 0.4, hi * 0.25, hi * 0.2, 'o', 'u', 0.45)];
			return [0, 1, 2].map((i) => syl(i * 0.11, 0.06, hi * (1.3 + (((h >>> i) & 1) ? 0.25 : 0)), hi * 1.1, 'i', 'e', 0.6, { src: 'sine' }));
		}
		case 'fish':
			return [0, 1, 2].slice(0, role === 'die' ? 3 : role === 'fight' ? 2 : 2).map((i) => syl(i * 0.12, 0.09, 260 / sc * (1 + i * 0.1), 520 / sc, 'o', 'u', 0.8, { src: 'sine', am: [14, 0.5], noise: 0.12 }));
		case 'insect':
		case 'arachnid':
		case 'crab':
			if (role === 'fight' && kind !== 'alert') return [syl(0, 0.4, 170 * sc, 190 * sc, 'a', 'a', 0.8, { am: [80, 0.7], noise: 0.2, bands: [[900, 4, 1], [2300, 5, 0.7], [4200, 3, 0.4]] })];
			return [syl(0, role === 'die' ? 0.4 : 0.25, 1, 1, 'a', 'a', 0.7, { src: 'noise', am: [26, 1], sq: true, bands: [[4800, 3, 1], [6800, 3, 0.7], [3200, 3, 0.5]], sweep: role === 'die' ? 0.6 : 1 })];
		case 'serpent':
			return [syl(0, role === 'fight' ? 0.45 : role === 'die' ? 0.55 : 0.3, 1, 1, 'a', 'a', 0.8, { src: 'noise', bands: HISS, sweep: role === 'die' ? 0.5 : 1.1 })];
		case 'ooze':
			return [syl(0, 0.22, 210 * sc, 80 * sc, 'o', 'u', 0.85, { src: 'saw', am: [18, 0.6], noise: 0.35, q: 4 })];
		case 'tentacled':
			return [
				syl(0, 0.1, 1, 1, 'a', 'a', 0.9, { src: 'noise', bands: [[650, 2, 1], [1500, 2, 0.5]], sweep: 0.5 }),
				syl(0.02, 0.14, 130 * sc, 55 * sc, 'o', 'o', 0.7, { src: 'sine' }),
			];
		case 'golem':
			return [syl(0, kind === 'sigh' ? 0.5 : role === 'die' ? 0.6 : 0.4, 58 * sc, 42 * sc, 'o', 'u', 0.9, { am: [11, 0.7], noise: 0.45, bands: [[200, 5, 1], [720, 4, 0.6], [1600, 3, 0.3]] })];
		default:
			return [];
	}
}

// ---------------------------------------------------------------- blips

type Contour = (u: number, i: number, seed: number) => number; // semitones
const CONTOUR: Record<string, Contour> = {
	formal: () => 0,
	blunt: (u) => 2 - 6 * u,
	flowery: (u) => 4 * Math.sin(Math.PI * u),
	terse: (_u, i) => (i % 2 ? 1.5 : -1),
	earthy: (_u, i, s) => (((s >>> (i % 24)) & 7) - 3.5) * 0.9,
	courtly: (u) => 3 * Math.sin(Math.PI * u * 0.7) - 0.5,
};

// ---------------------------------------------------------------- the handle

const MAX_VOICES = 4;
const PRIORITY: Record<string, number> = { death: 6, alert: 5, pain: 5, battlecry: 4, growl: 3, greet: 3, sigh: 2, laugh: 2, chatter: 1, speak: 0 };
const KIND_GAIN: Record<string, number> = { pain: 0.85, death: 1, alert: 0.9, greet: 0.6, laugh: 0.6, growl: 0.9, chatter: 0.45, battlecry: 1, sigh: 0.5 };

export function createVoices(ctx: AudioContext, dest: AudioNode): VoicesHandle {
	let enabled = true;
	let voicesOn = true;
	let blipsOn = false;
	let volume = 1;
	let disposed = false;
	let active = 0;
	const sources = new Set<AudioScheduledSourceNode>();
	const lastKind = new Map<string, number>();
	const lastId = new Map<number, number>();
	const recent: number[] = [];
	let blipSources: AudioScheduledSourceNode[] = [];
	let blipGains: GainNode[] = [];
	let lastBlipAt = 0;

	// output chain: bus -> soft ceiling -> master -> dest (voices -24 dBFS, blips -33 dBFS ceiling)
	function ceiling(c: number): WaveShaperNode {
		const n = 2049;
		const curve = new Float32Array(n);
		for (let i = 0; i < n; i++) {
			const x = (i / (n - 1)) * 2 - 1;
			curve[i] = c * Math.tanh(x / c);
		}
		const w = ctx.createWaveShaper();
		w.curve = curve;
		return w;
	}
	const master = ctx.createGain();
	master.gain.value = 0;
	master.connect(dest);
	const vbus = ctx.createGain();
	vbus.gain.value = 0.12;
	const vCeil = ceiling(0.063);
	vbus.connect(vCeil).connect(master);
	const bbus = ctx.createGain();
	bbus.gain.value = 0.05;
	const bCeil = ceiling(0.022);
	bbus.connect(bCeil).connect(master);

	const noiseBuf = (() => {
		const b = ctx.createBuffer(1, ctx.sampleRate, ctx.sampleRate);
		const d = b.getChannelData(0);
		let s = 0x1234567;
		for (let i = 0; i < d.length; i++) {
			s = (Math.imul(s, 1664525) + 1013904223) >>> 0;
			d[i] = (s / 2147483648 - 1) * 0.6;
		}
		return b;
	})();
	const pulseWaves = new Map<number, PeriodicWave>();
	function pulseWave(duty: number): PeriodicWave {
		const key = Math.round(duty * 100);
		let w = pulseWaves.get(key);
		if (!w) {
			const N = 48;
			const re = new Float32Array(N);
			const im = new Float32Array(N);
			for (let n = 1; n < N; n++) im[n] = (2 / (n * Math.PI)) * Math.sin(n * Math.PI * (key / 100));
			w = ctx.createPeriodicWave(re, im);
			pulseWaves.set(key, w);
		}
		return w;
	}

	function fade(to: number) {
		const t = ctx.currentTime;
		master.gain.cancelScheduledValues(t);
		master.gain.setValueAtTime(master.gain.value, t);
		master.gain.linearRampToValueAtTime(to, t + 1.6);
	}
	fade(volume);

	function track(n: AudioScheduledSourceNode) {
		sources.add(n);
		return n;
	}

	/** one syllable into `out`; calls done when its last source ended */
	function playSyl(s: Syl, t0: number, out: AudioNode, gain: number, base: number, seed: number, done: () => void, peak = 1) {
		const dur = s.dur;
		const end = t0 + dur;
		const env = ctx.createGain();
		const att = Math.min(0.025, dur * 0.25);
		env.gain.setValueAtTime(0.0001, t0);
		env.gain.linearRampToValueAtTime(Math.max(0.0002, s.amp * gain * peak), t0 + att);
		env.gain.setValueAtTime(Math.max(0.0002, s.amp * gain * peak), t0 + dur * 0.55);
		env.gain.exponentialRampToValueAtTime(0.0001, end);
		env.connect(out);
		const pre = ctx.createGain();
		const noiseOnly = s.src === 'noise';
		const am = ctx.createGain();
		const depth = s.am ? s.am[1] : 0;
		am.gain.value = 1 - depth * 0.5;
		const mixTo = am;
		const mine: AudioScheduledSourceNode[] = [];
		const hz = (v: number) => (base > 0 ? v * base : v);
		const f0 = hz(s.f[0]);
		const f1 = hz(s.f[1]);
		const nz = noiseOnly ? 1 : Math.min(1, s.noise ?? 0);
		if (!noiseOnly) {
			const vg = ctx.createGain();
			vg.gain.value = 1 - nz * 0.8;
			const mk = (cents: number): OscillatorNode => {
				const o = ctx.createOscillator();
				if (s.src === 'pulse') o.setPeriodicWave(pulseWave(s.duty ?? 0.3));
				else o.type = s.src === 'sine' ? 'sine' : 'sawtooth';
				o.frequency.setValueAtTime(Math.max(20, f0), t0);
				o.frequency.linearRampToValueAtTime(Math.max(20, f1), end);
				if (cents) o.detune.value = cents;
				o.connect(vg);
				mine.push(o);
				return o;
			};
			const oscs: OscillatorNode[] = [mk(0)];
			if (s.choir) oscs.push(mk(s.choir));
			if (s.vib) {
				const l = ctx.createOscillator();
				l.frequency.value = s.vib[0];
				const lg = ctx.createGain();
				lg.gain.value = s.vib[1];
				l.connect(lg);
				for (const o of oscs) lg.connect(o.detune);
				mine.push(l);
			}
			vg.connect(mixTo);
		}
		if (nz > 0.001) {
			const n = ctx.createBufferSource();
			n.buffer = noiseBuf;
			n.loop = true;
			const ng = ctx.createGain();
			ng.gain.value = noiseOnly ? 1 : nz * 0.9;
			n.connect(ng).connect(mixTo);
			n.start(t0, (seed % 1000) / 1000);
			mine.push(n);
		}
		let lfo: OscillatorNode | null = null;
		if (s.am) {
			lfo = ctx.createOscillator();
			lfo.type = s.sq ? 'square' : 'sine';
			lfo.frequency.value = s.am[0];
			const lg = ctx.createGain();
			lg.gain.value = depth * 0.5;
			lfo.connect(lg).connect(am.gain);
			mine.push(lfo);
		}
		void pre;
		// formants
		const sh = s.fs ?? 1;
		const q = s.q ?? 8;
		if (s.bands) {
			const sw = s.sweep ?? 1;
			for (const [fr, bq, bg] of s.bands) {
				const bp = ctx.createBiquadFilter();
				bp.type = 'bandpass';
				bp.Q.value = bq;
				bp.frequency.setValueAtTime(fr * sh, t0);
				if (sw !== 1) bp.frequency.linearRampToValueAtTime(fr * sh * sw, end);
				const g = ctx.createGain();
				g.gain.value = bg * 2.5;
				mixTo.connect(bp).connect(g).connect(env);
			}
		} else {
			const a = VOW[s.v[0]] ?? VOW.a;
			const b = VOW[s.v[1]] ?? VOW.a;
			const gs = [3.2, 2.2, 1.3];
			for (let i = 0; i < 3; i++) {
				const bp = ctx.createBiquadFilter();
				bp.type = 'bandpass';
				bp.Q.value = q + i * 2;
				bp.frequency.setValueAtTime(a[i] * sh, t0);
				bp.frequency.linearRampToValueAtTime(b[i] * sh, end);
				const g = ctx.createGain();
				g.gain.value = gs[i];
				mixTo.connect(bp).connect(g).connect(env);
			}
		}
		let left = mine.length;
		const fin = () => {
			if (--left > 0) return;
			try {
				env.disconnect();
				am.disconnect();
			} catch {
				/* already gone */
			}
			done();
		};
		for (const m of mine) {
			track(m);
			m.onended = () => {
				sources.delete(m);
				m.disconnect();
				fin();
			};
			if (m !== mine[mine.length - 1] || true) {
				try {
					if (!(m instanceof AudioBufferSourceNode)) m.start(t0);
					m.stop(end + 0.06);
				} catch {
					/* already started */
				}
			}
		}
		if (mine.length === 0) done();
	}

	function distGain(ev: VoxEvent, l: { x: number; y: number }): { g: number; pan: number } | null {
		if (ev.player) return { g: 1, pan: 0 };
		const dx = ev.x - l.x;
		const dy = ev.y - l.y;
		const d = Math.max(Math.abs(dx), Math.abs(dy));
		if (d > 22) return null;
		const g = d <= 1 ? 1 : 1 / (1 + (d - 1) * 0.22);
		return { g, pan: Math.max(-0.8, Math.min(0.8, dx / 14)) };
	}

	function voiceOf(ev: VoxEvent, kind: Exclude<VoxKind, 'speak'>, g: number, pan: number) {
		const seed = ev.pitchSeed >>> 0;
		const jit = 1 + (((seed >>> 4) % 13) - 6) / 100; // +-6%
		let syls: Syl[];
		let base = 0;
		if (ev.species || !ev.body || ev.body === 'humanoid') {
			const sp = spOf(ev.species ?? ev.tpl ?? 'human');
			base = sp.f0 * jit;
			if (ev.role === 'child') base *= 1.35;
			else if (ev.role === 'elder') base *= 0.93;
			if (ev.boss) base *= 0.8;
			syls = personSyls(kind, sp, seed);
		} else {
			syls = creatureSyls(ev.body, kind, ev.size ?? 3, seed);
			for (const s of syls) {
				s.f = [s.f[0] * jit, s.f[1] * jit];
				if (ev.boss) {
					s.f = [s.f[0] * 0.8, s.f[1] * 0.8];
					s.dur *= 1.2;
				}
			}
		}
		if (!syls.length) return;
		const t0 = ctx.currentTime + 0.02;
		const out = ctx.createGain();
		out.gain.value = g * (KIND_GAIN[kind] ?? 0.7) * (ev.boss ? 1.1 : 1);
		const pn = ctx.createStereoPanner();
		pn.pan.value = pan;
		out.connect(pn).connect(vbus);
		active++;
		let pending = syls.length;
		const fin = () => {
			if (--pending > 0) return;
			active = Math.max(0, active - 1);
			try {
				pn.disconnect();
				out.disconnect();
			} catch {
				/* already gone */
			}
		};
		for (const s of syls) {
			// species recipes use multiples of the base pitch, creatures absolute Hz
			playSyl(s, t0 + s.t, out, 1, base, seed, fin);
		}
	}

	function admit(ev: VoxEvent, now: number): boolean {
		const lk = lastKind.get(ev.kind) ?? -9;
		if (now - lk < 0.09) return false;
		const li = lastId.get(ev.id) ?? -9;
		if (now - li < (ev.kind === 'death' ? 0.4 : 0.3)) return false;
		while (recent.length && now - recent[0] > 0.5) recent.shift();
		if (recent.length >= 5 && !ev.player) return false;
		return true;
	}

	function play(events: VoxEvent[] | undefined, listener: { x: number; y: number }) {
		if (disposed || !enabled || !events || !events.length) return;
		if (ctx.state !== 'running') return;
		const now = ctx.currentTime;
		const list = events.slice().sort((a, b) => (b.player ? 1 : 0) - (a.player ? 1 : 0) || (PRIORITY[b.kind] ?? 0) - (PRIORITY[a.kind] ?? 0));
		for (const ev of list) {
			if (ev.kind === 'speak') {
				if (blipsOn) blips({ species: ev.species, register: ev.register, mood: ev.mood, text: ev.text ?? 'x'.repeat(ev.len ?? 0), id: ev.id });
				continue;
			}
			if (!voicesOn || active >= MAX_VOICES) continue;
			const dg = distGain(ev, listener);
			if (!dg || !admit(ev, now)) continue;
			lastKind.set(ev.kind, now);
			lastId.set(ev.id, now);
			recent.push(now);
			voiceOf(ev, ev.kind, dg.g, dg.pan);
		}
		if (lastId.size > 64) lastId.clear();
	}

	function stopBlips() {
		const t = ctx.currentTime;
		for (const g of blipGains) {
			try {
				g.gain.cancelScheduledValues(t);
				g.gain.setTargetAtTime(0, t, 0.01);
			} catch {
				/* gone */
			}
		}
		for (const s of blipSources) {
			try {
				s.stop(t + 0.05);
			} catch {
				/* gone */
			}
		}
		blipSources = [];
		blipGains = [];
	}

	function blips(sp: VoiceSpeech) {
		if (disposed || !enabled || !blipsOn || ctx.state !== 'running') return;
		const text = (sp.text ?? '').replace(/\s+/g, ' ').trim();
		if (!text) return;
		const t = ctx.currentTime;
		if (t - lastBlipAt < 0.12) return;
		lastBlipAt = t;
		stopBlips();
		const sv = spOf(sp.species);
		const seed = strHash((sp.species ?? '') + String(sp.id ?? 0));
		const reg = CONTOUR[sp.register ?? ''] ? (sp.register as string) : 'formal';
		const contour = CONTOUR[reg];
		const mood = sp.mood ?? 'neutral';
		const n = Math.max(1, Math.min(14, Math.ceil(text.length / 2.6)));
		let semi0 = 0;
		let gap = 0.062;
		let dur = 0.046;
		let amp = 0.7;
		let noise = 0.04;
		let shake = false;
		let growl = 0;
		if (reg === 'terse') (gap = 0.085), (dur = 0.03);
		if (reg === 'courtly') (gap = 0.075), (dur = 0.055);
		if (reg === 'earthy') gap = 0.058;
		switch (mood) {
			case 'angry':
			case 'scornful':
				semi0 = -3;
				noise = 0.2;
				growl = 0.35;
				break;
			case 'afraid':
			case 'wary':
				semi0 = 3.5;
				shake = true;
				gap *= 0.88;
				break;
			case 'warm':
			case 'pleased':
			case 'amused':
				amp = 0.5;
				noise = 0.02;
				break;
			case 'sad':
			case 'grieving':
			case 'tired':
				semi0 = -2.5;
				gap *= 1.3;
				dur *= 1.15;
				amp = 0.55;
				break;
		}
		const base = Math.max(70, Math.min(330, sv.f0 * 1.25));
		const vs = 'aeiou';
		for (let i = 0; i < n; i++) {
			const ch = text.charCodeAt(Math.floor((i * text.length) / n)) || 97;
			const u = n > 1 ? i / (n - 1) : 0;
			const semi = semi0 + contour(u, i, seed) + (ch % 7) - 3;
			const f = base * Math.pow(2, semi / 12);
			const ts = t + 0.03 + i * gap;
			const s: Syl = {
				t: 0,
				dur,
				f: [f, f * (shake ? 1.04 : 0.985)],
				v: [vs[ch % 5], vs[(ch >>> 2) % 5]],
				amp,
				noise: Math.min(0.5, noise + sv.breath * 0.3),
				fs: sv.fs,
				q: 7,
				choir: sv.choir ? sv.choir * 0.5 : undefined,
				vib: shake ? [9, 45] : undefined,
				am: growl || sv.growl ? [48, Math.min(0.6, growl + sv.growl * 0.3)] : undefined,
			};
			// blips go on their own quiet bus
			const bg = ctx.createGain();
			bg.gain.value = 1;
			bg.connect(bbus);
			blipGains.push(bg);
			const before = new Set(sources);
			playSyl(s, ts, bg, 1, 0, seed + i, () => {}, 1);
			for (const src of sources) if (!before.has(src)) blipSources.push(src);
		}
	}

	return {
		play,
		blips,
		setEnabled(on) {
			if (disposed || on === enabled) return;
			enabled = on;
			if (!on) {
				stopBlips();
				stopAll();
				master.gain.cancelScheduledValues(ctx.currentTime);
				master.gain.value = 0;
			} else fade(volume);
		},
		setVolume(v) {
			volume = Math.max(0, Math.min(1, v));
			if (enabled && !disposed) {
				const t = ctx.currentTime;
				master.gain.cancelScheduledValues(t);
				master.gain.setValueAtTime(master.gain.value, t);
				master.gain.linearRampToValueAtTime(volume, t + 0.2);
			}
		},
		setVoices(on) {
			voicesOn = on;
		},
		setBlips(on) {
			blipsOn = on;
			if (!on) stopBlips();
		},
		dispose() {
			if (disposed) return;
			disposed = true;
			stopAll();
			try {
				master.disconnect();
				vbus.disconnect();
				bbus.disconnect();
				vCeil.disconnect();
				bCeil.disconnect();
			} catch {
				/* already gone */
			}
		},
	};

	function stopAll() {
		for (const s of sources) {
			try {
				s.stop();
			} catch {
				/* not started or already stopped */
			}
		}
		sources.clear();
		blipSources = [];
		blipGains = [];
		active = 0;
	}
}
