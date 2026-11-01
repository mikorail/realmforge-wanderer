// Procedural adaptive soundtrack. Pure WebAudio, no assets, no deps.
// Everything is generated from (mood, genre, culture sphere, biome, seed) with a deterministic PRNG;
// danger / combat / hp / day / depth / weather adapt the playback continuously without restarting.
// Two instrument flavours: "cultural" (synthesised per-land instruments, src/instruments.ts, tables in
// src/scales.ts) and "pure chiptune" (pulse / triangle / noise only).
import type { QudMusic } from './qud';
import { loadSettings, type InstrumentFlavour, type MusicStyle } from './settings';
import { createGarden, gardenFor, type Garden, type GardenStyle } from './garden';
import { FEELS, GENERIC, GENRE_COLOUR, MOOD_LEADS, PROFILES, SCALES, type GenreColour, type Profile } from './scales';
import { INST, clap, createKit, disposeKit, drone2, hit, sweep, tex, type Kit, type TexKind } from './instruments';

export interface MusicHandle {
	setState(m: QudMusic): void;
	setVolume(v: number): void;
	setEnabled(on: boolean): void;
	setStyle(s: MusicStyle): void;
	/** instrument flavour; also picked up from the saved settings, so wiring it is optional */
	setFlavour(f: InstrumentFlavour): void;
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
/** bitcrush-ish: quantise loudness to 32 levels (fine enough for quiet voices) */
const q16 = (v: number) => Math.round(clamp(v, 0, 1) * 31) / 31;

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
interface MoodCfg {
	lo: number; // bpm at danger 0
	hi: number; // bpm at danger 1
	scales: string[];
	minDanger: number; // floor for percussion/arps (world overture is always marching)
	bright?: number; // filter cutoff multiplier (default 1)
	sparse?: number; // 0..1: thins out lead, arps and hats
}
const MOODS: Record<Mood, MoodCfg> = {
	town: { lo: 88, hi: 140, scales: ['ionian', 'lydian', 'majPent'], minDanger: 0, bright: 1.3 },
	wild: { lo: 78, hi: 148, scales: ['dorian', 'mixolydian', 'minPent', 'aeolian'], minDanger: 0 },
	dungeon: { lo: 64, hi: 150, scales: ['phrygian', 'locrian', 'aeolian'], minDanger: 0, bright: 0.8, sparse: 0.5 },
	tower: { lo: 74, hi: 144, scales: ['aeolian', 'dorian', 'harmonicMinor'], minDanger: 0 },
	sky: { lo: 80, hi: 142, scales: ['lydian', 'majPent', 'ionian'], minDanger: 0 },
	sea: { lo: 72, hi: 140, scales: ['dorian', 'mixolydian', 'aeolian'], minDanger: 0 },
	underdark: { lo: 70, hi: 140, scales: ['wholetone', 'locrian', 'phrygDom'], minDanger: 0 },
	crypt: { lo: 70, hi: 140, scales: ['phrygian', 'harmonicMinor', 'aeolian'], minDanger: 0 },
	world: { lo: 100, hi: 142, scales: ['mixolydian', 'ionian', 'lydian'], minDanger: 0.35, bright: 1.3 },
	dead: { lo: 52, hi: 52, scales: ['aeolian'], minDanger: 0 },
};

interface GenreCfg {
	bpm: number;
	scales: string[];
	w: number; // chance the genre scale list overrides the mood's
	arpBias: number; // lowers the danger at which arpeggios start
	leadDay: number;
}
const GENRES: Record<string, GenreCfg> = {
	fantasy: { bpm: 1, scales: ['ionian', 'dorian', 'mixolydian'], w: 0.25, arpBias: 0, leadDay: 0.25 },
	mythic: { bpm: 0.96, scales: ['dorian', 'phrygian', 'aeolian'], w: 0.5, arpBias: 0, leadDay: 0.25 },
	steampunk: { bpm: 1, scales: ['mixolydian', 'dorian', 'harmonicMinor'], w: 0.5, arpBias: 0.05, leadDay: 0.25 },
	dieselpunk: { bpm: 1.02, scales: ['dorian', 'mixolydian', 'aeolian'], w: 0.5, arpBias: 0, leadDay: 0.25 },
	atompunk: { bpm: 1.05, scales: ['lydian', 'mixolydian', 'majPent'], w: 0.5, arpBias: 0.05, leadDay: 0.25 },
	cyberpunk: { bpm: 1.08, scales: ['aeolian', 'phrygian', 'dorian'], w: 0.6, arpBias: 0.2, leadDay: 0.125 },
	biopunk: { bpm: 0.98, scales: ['wholetone', 'phrygDom', 'locrian'], w: 0.5, arpBias: 0.05, leadDay: 0.125 },
	solarpunk: { bpm: 0.97, scales: ['majPent', 'lydian', 'ionian'], w: 0.6, arpBias: 0, leadDay: 0.25 },
	wasteland: { bpm: 0.94, scales: ['minPent', 'mixolydian', 'phrygian'], w: 0.5, arpBias: 0, leadDay: 0.5 },
	twilight: { bpm: 0.9, scales: ['harmonicMinor', 'aeolian', 'phrygian'], w: 0.5, arpBias: 0.05, leadDay: 0.5 },
};

// functional-harmony Markov chain over scale degrees (0 = tonic)
const TRANS: number[][] = [[3, 4, 5, 1, 2, 6], [4, 6, 4], [5, 3], [4, 0, 1, 4], [0, 5, 0], [3, 1, 4], [0, 2, 0]];
const IVS = [-2, -1, -1, 1, 1, 2, 0, 3, -3];
const IVS_UP = [1, 1, 1, 2, 0, 1, 2, -1];
const ARPS: number[][] = [[0, 1, 2, 1], [0, 1, 2, 3, 2, 1], [0, 2, 1, 2], [0, 1, 2, 3]]; // offsets into [0,2,4,+octave]

// ---------- composition ----------
interface NoteEv { s: number; d: number; l: number }
interface Voices { lead: string; harm: string; bass: string; pad: string; kit: string; drone: 'none' | 'tonic' | 'fifth'; tex: string }
interface Piece {
	key: string;
	dead: boolean;
	spb: number;
	groups: number[];
	gIdx: number[]; // step -> group index when a group starts there, else -1
	gOf: number[]; // step -> first step of its group
	half: number; // start of the middle group
	swing: number;
	scale: number[];
	base: number;
	chords: number[][]; // [section][bar] -> scale index of chord root
	mel: NoteEv[][][]; // [section][bar] -> notes
	form: number[]; // section order, each 4 bars
	bars: number;
	cfg: MoodCfg;
	gen: GenreCfg;
	arpStyle: number;
	style: MusicStyle;
	cult: boolean;
	v: Voices;
	tower: boolean;
	feast: boolean;
	under: boolean;
	tempo: number;
}

function pieceKey(m: QudMusic, style: MusicStyle, flavour: InstrumentFlavour, fight = false, variant = 0): string {
	return `${style}|${flavour}|${m.mood}|${m.genre}|${m.seed}|${m.biome | 0}|${(m.season ?? 0) | 0}|${m.sphere ?? ''}|${m.feast ? 1 : 0}|${fight ? 1 : 0}|${variant}`;
}

// exotic / microtonal scales sound broken on these synths: fold them onto a close tonal relative
const TAME: Record<string, string> = {
	wholetone: 'aeolian', octatonic: 'harmonicMinor', uncanny: 'phrygian', locrian: 'phrygian', marwa: 'harmonicMinor', todi: 'harmonicMinor',
	slendro: 'majPent', rast: 'mixolydian', bayati: 'phrygian', homayun: 'hijaz', ukrDorian: 'dorian', bhairav: 'hijaz', pelog: 'hirajoshi',
};
const ODD_FEELS = new Set(['seven8', 'five8', 'nine8']);
// tried-and-true progressions over 7-note scale degrees (0 = tonic), 4 bars each
const PROG_MAJ = [[0, 4, 5, 3], [0, 5, 3, 4], [0, 3, 4, 0], [0, 3, 0, 4], [5, 3, 0, 4], [0, 1, 4, 0]];
const PROG_MIN = [[0, 5, 2, 6], [0, 3, 4, 0], [0, 6, 5, 4], [0, 5, 3, 4], [0, 3, 6, 2], [0, 2, 3, 4]];
const PROG_FIGHT = [[0, 5, 6, 4], [0, 0, 5, 6], [0, 6, 5, 6], [0, 3, 5, 4]];

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

function layout(groups: number[]) {
	const gIdx: number[] = [], gOf: number[] = [];
	let pos = 0;
	groups.forEach((L, gi) => {
		for (let i = 0; i < L; i++) { gIdx.push(i === 0 ? gi : -1); gOf.push(pos); }
		pos += L;
	});
	let half = 0;
	for (let i = 0; i < Math.floor(groups.length / 2); i++) half += groups[i];
	return { gIdx, gOf, half, spb: pos };
}

function makePiece(m: QudMusic, style: MusicStyle, flavour: InstrumentFlavour, fight = false, variant = 0): Piece {
	const key = pieceKey(m, style, flavour, fight, variant);
	const rng = mulberry32(hash(key));
	const pick = <T,>(a: T[]): T => a[Math.floor(rng() * a.length)];
	const gen = GENRES[m.genre] ?? GENRES.fantasy;
	const col: GenreColour = GENRE_COLOUR[m.genre] ?? {};
	const cfg = MOODS[m.mood] ?? MOODS.wild;
	const dead = m.mood === 'dead';
	const sph: Profile | null = m.sphere ? PROFILES[m.sphere] ?? null : null;
	const prof = sph ?? GENERIC;
	const dark = m.mood === 'dungeon' || m.mood === 'underdark' || m.mood === 'crypt';
	const feast = !dead && !!m.feast && m.mood === 'town';
	const tower = m.mood === 'tower';

	// scale: the land's own modes first, then a genre / mood flavour
	let scaleName: string;
	if (dead) scaleName = 'aeolian';
	else if (fight) scaleName = pick(['aeolian', 'harmonicMinor', 'dorian', 'phrygian']);
	else if (sph && rng() < (dark ? 0.3 : 0.5)) scaleName = pick(sph.scales);
	else if (col.scales && rng() < (col.scaleW ?? 0)) scaleName = pick(col.scales);
	else scaleName = rng() < gen.w ? pick(gen.scales) : pick(cfg.scales);
	scaleName = TAME[scaleName] ?? scaleName;
	const scale = dead ? SCALES[scaleName] : seasonScale(SCALES[scaleName] ?? SCALES.aeolian, (m.season ?? 0) | 0, rng());
	const n = scale.length;

	// rhythmic feel
	let feelId = 'four';
	if (!dead) {
		feelId = feast && prof.feastFeel ? prof.feastFeel : pick(prof.feels);
		if (col.feels && rng() < (col.feelW ?? 0)) feelId = pick(col.feels);
		if (ODD_FEELS.has(feelId) && rng() < 0.8) feelId = 'four';
		if (fight) feelId = rng() < 0.75 ? 'four' : 'six8';
	}
	const feel = FEELS[feelId] ?? FEELS.four;
	let groups = feel.groups.slice();
	if (feelId === 'seven8') { const r = Math.floor(rng() * 3); groups = groups.slice(r).concat(groups.slice(0, r)); }
	const { gIdx, gOf, half, spb } = layout(groups);
	const base = 45 + ((hash(key + 'k') + (m.biome | 0) * 5 + 12 + (dead ? 0 : SEASON_SHIFT[(m.season ?? 0) & 3])) % 12);
	const center = Math.round(n * 0.9);

	// instruments
	const leadPool = feast && prof.feastLeads ? prof.feastLeads : prof.leads;
	let lead = pick(leadPool);
	const lm = MOOD_LEADS[m.mood];
	if (lm && rng() < lm.w) lead = pick(lm.leads);
	if (col.lead && rng() < (col.leadW ?? 0)) lead = col.lead;
	if (dead) lead = prof.lament ?? 'duduk';
	let harm = pick(prof.plucks);
	if (col.harm && rng() < (col.harmW ?? 0)) harm = col.harm;
	const bass = col.bass && rng() < (col.bassW ?? 0) ? col.bass : prof.bass;
	let pad = prof.pad ?? (m.mood === 'crypt' ? 'organ' : dark ? 'wet' : 'sine');
	if (col.pad && rng() < (col.padW ?? 0)) pad = col.pad;
	let kit = feast && prof.feastKit ? prof.feastKit : prof.kit;
	if (col.kit && rng() < (col.kitW ?? 0)) kit = col.kit;
	const v: Voices = { lead, harm, bass, pad, kit, drone: dark && prof.drone === 'none' ? 'tonic' : prof.drone, tex: col.texture ?? '' };
	const common = { key, dead, spb, groups, gIdx, gOf, half, swing: feel.swing, scale, base, cfg, gen, style, cult: flavour === 'cultural', v, tower, feast, under: dark, tempo: prof.tempo * (feast ? 1.1 : 1) * (fight ? 1.15 : 1) };

	if (dead) {
		const ds = [n + 4, n + 3, n + 2, n + 1, n, n - 1, n - 2, n - 3];
		const mel: NoteEv[][] = [0, 1, 2, 3].map((b) =>
			b < 3
				? [{ s: 0, d: ds[b * 2], l: 8 }, { s: 8, d: ds[b * 2 + 1], l: 8 }]
				: [{ s: 0, d: ds[6], l: 6 }, { s: 6, d: ds[7], l: 10 }]);
		return { ...common, chords: [[0, 5, 3, 0]], mel: [mel], form: [0, 0, 0, 0], bars: 4, arpStyle: 0 };
	}

	const progression = (start: number, end: number): number[] => {
		const p = [start];
		for (let i = 1; i < 3; i++) p.push(pick(TRANS[p[i - 1] % TRANS.length]) % n);
		p.push(end % n);
		return p;
	};
	const onsetsFor = (): number[] => {
		const out: number[] = [];
		const dens = 0.3 + rng() * 0.35;
		let pos = 0;
		for (const L of groups) {
			out.push(pos);
			for (let o = 2; o < L; o += 2) if (rng() < dens) out.push(pos + o);
			pos += L;
		}
		return out;
	};
	const ivs = tower ? IVS_UP : IVS;
	const motif = () => {
		const on = onsetsFor();
		const iv = [0];
		for (let i = 1; i < on.length; i++) {
			let s = pick(ivs);
			if (i > 1 && Math.abs(iv[i - 1] - (iv[i - 2] ?? 0)) >= 3) s = s > 0 ? -1 : 1; // step back after a leap
			iv.push(iv[i - 1] + s);
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
			if (gIdx[s] >= 0) d = snap(d, chord, n); // strong beats land on chord tones
			out.push({ s, d, l: Math.min(6, (mo.on[i + 1] ?? spb) - s) });
		}
		return out;
	};
	const cadence = (chord: number): NoteEv[] => {
		if (tower) {
			// an ascending run that lands on a chord tone: the tower climbs
			const end = snap(center + 1 + Math.floor(rng() * 2), chord, n);
			return [0, 1, 2, 3].map((i) => ({ s: Math.max(0, spb - 8) + i * 2, d: end - (3 - i), l: 2 }));
		}
		const t2 = snap(center + Math.floor(rng() * 3) - 1, chord, n);
		const t1 = t2 + (rng() < 0.5 ? 1 : -1);
		return [{ s: 0, d: t1, l: Math.max(2, groups[0] - 2) }, { s: groups[0], d: t2, l: spb - groups[0] }];
	};
	const section = (chords: number[]): NoteEv[][] => {
		const mo = motif();
		const st0 = snap(center + Math.floor(rng() * 3) - 1 - (tower ? 2 : 0), chords[0], n);
		const seq = tower ? 2 : rng() < 0.5 ? 2 : -2;
		return [
			realize(mo, st0, chords[0], 0),
			realize(mo, st0, chords[1], rng() < 0.5 ? 1 : -1),
			realize(mo, st0 + seq, chords[2], 0),
			cadence(chords[3]),
		];
	};

	let chA: number[], chB: number[];
	if (n === 7) {
		const minor = scale[2] === 3;
		const pool = fight ? PROG_FIGHT : minor ? PROG_MIN : PROG_MAJ;
		chA = pick(pool);
		chB = pick(pool.filter((p) => p !== chA));
		if (chB[0] === 0) chB = [chB[1], chB[2], chB[3], chB[0] === 0 ? 4 : chB[0]];
	} else {
		chA = progression(0, 4);
		chB = progression(pick([3, 5]) % n, 4);
	}
	const chA2 = chA.slice(0, 3).concat([0]);
	const melA = section(chA);
	const melA2 = melA.slice(0, 3).concat([cadence(0)]); // A' = A with a resolving ending
	const melB = section(chB);

	return {
		...common,
		chords: [chA, chA2, chB],
		mel: [melA, melA2, melB],
		form: [0, 1, 2, 0],
		bars: 16,
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
	let style: MusicStyle = 'auto';
	let flavour: InstrumentFlavour = 'cultural';
	try { flavour = loadSettings().instrumentFlavour; } catch { /* default */ }
	let flavT = 0;
	let fxT = 0;
	let clNow = 0;
	let kit: Kit | null = null;
	let garden: Garden | null = null;
	let fxIn!: GainNode;
	let roomPre!: BiquadFilterNode;
	let roomOut!: GainNode;
	let delayOut!: GainNode;
	let springOut!: GainNode;
	let started = false;
	let disposed = false;
	let st: QudMusic | null = null;
	let cur: Player | null = null;
	let activeKey = '';
	let pendingKey: string | null = null;
	// combat gets its own theme, held a few seconds so it doesn't flap as foes step in and out of view
	let fight = false;
	let fightUntil = 0;
	// a place's theme is re-composed after a couple of passes so a long stay doesn't loop one tune
	let variant = 0;
	let loops = 0;
	let baseKey = '';
	function keyNow(): string {
		const b = pieceKey(st!, style, flavour, fight, 0);
		if (b !== baseKey) { baseKey = b; variant = 0; }
		return pieceKey(st!, style, flavour, fight, variant);
	}
	function updateFight(on: boolean) {
		const now = performance.now();
		if (on) { fight = true; fightUntil = now + 8000; }
		else if (fight && now > fightUntil) fight = false;
	}
	let bpm = 80;
	let dNow = 0;
	let lastCut = -1;
	const fading = new Set<ReturnType<typeof setTimeout>>();
	let suspendT: ReturnType<typeof setTimeout> | null = null;

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
				comp.threshold.value = -20;
				comp.knee.value = 10;
				comp.ratio.value = 8;
				comp.attack.value = 0.003;
				comp.release.value = 0.25;
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
				// effect sends: every player bus feeds fxIn; applyFx sets how much of each return is heard
				fxIn = c.createGain();
				roomPre = c.createBiquadFilter();
				roomPre.type = 'lowpass';
				roomPre.frequency.value = 5200;
				fxIn.connect(roomPre);
				const room = c.createConvolver();
				room.buffer = impulse(c, 2.6, 2.2, 0);
				roomOut = c.createGain();
				roomOut.gain.value = 0;
				roomPre.connect(room).connect(roomOut).connect(tone);
				const dl = c.createDelay(1);
				dl.delayTime.value = 0.375;
				const fb = c.createGain();
				fb.gain.value = 0.35;
				dl.connect(fb).connect(dl);
				delayOut = c.createGain();
				delayOut.gain.value = 0;
				fxIn.connect(dl);
				dl.connect(delayOut).connect(tone);
				const spring = c.createConvolver();
				spring.buffer = impulse(c, 1.4, 1.1, 0.031);
				springOut = c.createGain();
				springOut.gain.value = 0;
				fxIn.connect(spring).connect(springOut).connect(tone);
				kit = createKit(c);
				garden = createGarden(c, kit, tone);
			} catch {
				ctx = null;
				return false;
			}
		}
		if (ctx.state !== 'running') ctx.resume().catch(() => {});
		applyVolume();
		return true;
	}


	/** synthetic reverb impulse: decaying noise; `chirp` > 0 adds the repeating dispersive slap of a spring tank */
	function impulse(c: AudioContext, secs: number, tau: number, chirp: number): AudioBuffer {
		const len = Math.floor(c.sampleRate * secs);
		const buf = c.createBuffer(2, len, c.sampleRate);
		for (let ch = 0; ch < 2; ch++) {
			const d = buf.getChannelData(ch);
			let lp = 0;
			for (let i = 0; i < len; i++) {
				const x = i / c.sampleRate;
				lp += ((Math.random() * 2 - 1) - lp) * (0.5 - 0.35 * Math.min(1, x / secs)); // darkens as it decays
				let a = Math.exp(-x / (tau * 0.4));
				if (chirp > 0) a *= 0.55 + 0.45 * Math.cos((2 * Math.PI * x) / chirp);
				d[i] = lp * a * 1.6;
			}
		}
		return buf;
	}

	function applyVolume() {
		if (!ctx) return;
		// the very first start fades in over a couple of seconds; later changes are quick
		const tau = enabled && !started ? 0.7 : 0.06;
		if (enabled) started = true;
		master.gain.setTargetAtTime(enabled ? volume * 0.8 : 0, ctx.currentTime, tau);
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
	/** soft pad / bell voice for the ambient style */
	function soft(p: Player, type: OscillatorType, midi: number, t: number, dur: number, vel: number, attack: number) {
		const c = ctx!;
		const o = c.createOscillator();
		const g = c.createGain();
		o.type = type;
		o.frequency.value = mtof(midi);
		o.detune.value = hrand(Math.round(midi), Math.round(t * 1000), 3) * 10 - 5;
		g.gain.setValueAtTime(0, t);
		g.gain.linearRampToValueAtTime(vel, t + attack);
		g.gain.setValueAtTime(vel, t + Math.max(attack, dur * 0.5));
		g.gain.linearRampToValueAtTime(0, t + dur + 0.3);
		o.connect(g).connect(p.bus);
		o.start(t);
		o.stop(t + dur + 0.35);
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
		bus.connect(fxIn);
		const hat = c.createBiquadFilter();
		hat.type = 'highpass';
		hat.frequency.value = 7000;
		hat.connect(bus);
		const snr = c.createBiquadFilter();
		snr.type = 'bandpass';
		snr.frequency.value = 1800;
		snr.Q.value = 0.8;
		snr.connect(bus);
		loops = 0;
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
		const amb = p.style === 'ambient';
		const x = Math.pow(clamp(d, 0, 1), 0.9) * (amb ? 0.4 : 1);
		return (p.cfg.lo + (p.cfg.hi - p.cfg.lo) * x) * p.gen.bpm * p.tempo * (1 + 0.1 * clNow) * (st && !st.day ? 0.96 : 1) * (amb ? 0.55 : 1) * wxCfg(st?.wx).tempo;
	}

	function scheduleStep(p: Player): number {
		const pc = p.piece;
		const m = st!;
		const spb = pc.spb;
		const n = pc.scale.length;
		const pitch = (idx: number) => pc.scale[((idx % n) + n) % n] + 12 * Math.floor(idx / n);
		const seed = hash(pc.key);
		const wx = pc.dead ? WX0 : wxCfg(m.wx);
		const cult = pc.cult && kit !== null;
		const bus = p.bus;

		// continuous adaptation
		const dTarget = pc.dead ? 0 : Math.max(clamp(m.danger, 0, 1), pc.cfg.minDanger);
		dNow += (dTarget - dNow) * 0.1;
		const d = dNow;
		const clT = pc.dead || !m.combat ? 0 : 0.35 + 0.65 * clamp(m.danger, 0, 1);
		clNow += (clT - clNow) * 0.08;
		const cl = clNow;
		bpm += (targetBpm(pc, d) - bpm) * 0.08;
		const dur = 60 / bpm / 4;

		const t0 = p.nextT;
		const t = pc.swing && p.step % 4 === 2 ? t0 + pc.swing * dur : t0;
		const step = p.step;
		const bar = p.bar;
		const sec = pc.form[Math.floor(bar / 4) % 4];
		const bi = bar % 4;
		const chord = pc.chords[sec][bi];
		const day = m.day;
		const gi = pc.gIdx[step];
		const isG = gi >= 0;
		const off = step - pc.gOf[step];
		const leadBase = pc.base + 12;
		const leadDuty = pc.dead ? 0.5 : day ? pc.gen.leadDay : 0.5;
		const rootHz = mtof(pc.base);
		const sparse = pc.cfg.sparse ?? 0;
		const ambient = pc.style === 'ambient' && !pc.dead;
		const season = (m.season ?? 0) | 0;
		const play = (name: string, midi: number, at: number, len: number, vel: number) => (INST[name] ?? INST.lute)(kit!, bus, at, midi, len, vel);

		// shared layers: drones, dungeon hum, texture beds, seasonal sparkle
		const extras = () => {
			if (pc.dead) return;
			if (step === 0 && bar % 2 === 0) {
				const len = spb * dur * 2.05;
				if (cult) {
					if (pc.v.drone !== 'none' && d < 0.7) {
						drone2(kit!, bus, t, pc.base - 12, len, 0.04, 'sawtooth');
						if (pc.v.drone === 'fifth') drone2(kit!, bus, t, pc.base - 5, len, 0.025, 'sawtooth');
					}
					if (pc.under) drone2(kit!, bus, t, pc.base - 24, len, 0.07, 'triangle', 0.5);
				} else if (pc.under) drone(p, pc.base - 24, t, len, 0.09);
			}
			if (!kit) return;
			const tx = pc.v.tex;
			if (tx === 'geiger') {
				if (hrand(bar, step, seed + 20) < 0.06 + (day ? 0 : 0.05) + 0.05 * d) tex(kit, bus, t, 'geiger' as TexKind, 0.05);
				if (isG && gi === 0 && bar % 4 === 1 && hrand(bar, step, seed + 22) < 0.4) tex(kit, bus, t, 'static', 0.03);
			} else if (tx === 'steam') {
				if (step === 0 && bar % 2 === 1 && hrand(bar, step, seed + 20) < 0.5) tex(kit, bus, t, 'hiss', 0.05);
				if (isG && d > 0.2 && hrand(bar, step, seed + 21) < 0.3) tex(kit, bus, t, 'tick', 0.04);
			} else if (tx === 'spore') {
				if (hrand(bar, step, seed + 20) < 0.06) tex(kit, bus, t, 'bubble', 0.05);
			}
			if (pc.under && hrand(bar, step, seed + 23) < 0.035) tex(kit, bus, t, 'drip', 0.045);
			if (cult && step === 0 && bi === 0 && !(wx.sparse && d > 0.5)) {
				if (season === 3 && hrand(bar, 0, seed + 30) < 0.35) play('bell', pc.base + 24 + pitch(chord + 4), t, 1.2, 0.045);
				else if (season === 0 && hrand(bar, 0, seed + 31) < 0.3) {
					const g = pc.base + 24 + pitch(chord + 2);
					play('whistle', g, t, dur * 2, 0.04);
					play('whistle', g + 2, t + dur * 2.5, dur * 3, 0.035);
				}
			}
		};

		if (ambient) {
			// slow, soft: one pad chord per bar, a sparse lead on strong beats, long bass
			if (step === 0) {
				if (cult) {
					for (const o of [0, 2, 4]) play(pc.v.pad, pc.base + 12 + pitch(chord + o), t, spb * dur * 0.95, 0.045);
					play(pc.v.bass, pc.base - 12 + pitch(chord), t, spb * dur * 0.9, 0.11);
				} else {
					const pad = day ? 'triangle' : 'sine';
					for (const o of [0, 2, 4]) soft(p, pad, pc.base + 12 + pitch(chord + o), t, spb * dur * 0.95, 0.045, 0.5);
					tri(p, pc.base - 12 + pitch(chord), t, spb * dur * 0.9, 0.13);
				}
			}
			for (const ev of pc.mel[sec][bi]) {
				if (ev.s !== step || !isG) continue;
				if (hrand(bar, step, seed + 5) < 0.4 + sparse * 0.4) continue;
				const pm = leadBase + pitch(ev.d) - (day ? 0 : 12);
				if (cult) play(pc.v.lead, pm, t, ev.l * dur * 1.4, 0.075);
				else soft(p, 'triangle', pm, t, ev.l * dur * 1.4, 0.075, 0.12);
			}
			extras();
			p.nextT = t0 + dur;
			p.step++;
			if (p.step >= spb) { p.step = 0; p.bar++; if (p.bar >= pc.bars) p.bar = 0; }
			return p.nextT;
		}

		// lead
		for (const ev of pc.mel[sec][bi]) {
			if (ev.s !== step) continue;
			const strong = isG;
			if (!strong && !pc.dead && hrand(bar, step, seed) > 0.55 + 0.45 * d - sparse * 0.5) continue;
			if (strong && !pc.dead && d < 0.6 && hrand(bar, step, seed + 9) < sparse * 0.35) continue;
			if (!strong && wx.sparse && hrand(bar, step, seed + 5) < wx.sparse) continue;
			const len = ev.l * dur * (pc.dead ? 1 : 0.92);
			const pm = leadBase + wx.lift + pitch(ev.d) - (day ? 0 : 12);
			const vel = pc.dead ? 0.1 : (0.09 + 0.025 * d) * wx.soft;
			if (cult) play(pc.v.lead, pm, t, pc.dead ? len * 1.3 : len, vel);
			else pulse(p, leadDuty, pm, t, len, vel, hrand(bar, step, 7) * 8 - 4);
		}

		// lament: a slow bell toll and a low drone under the melody
		if (pc.dead && cult && step === 0) {
			play('bell', pc.base - 12, t, 2, 0.08);
			if (bar === 0) drone2(kit!, bus, t, pc.base - 12, spb * dur * 4.1, 0.05, 'sawtooth');
		}

		// harmony voice: calm rolled chord tones, arpeggio as danger rises
		const arpAt = 0.35 - pc.gen.arpBias;
		if (!pc.dead || !cult) {
			if (d < arpAt && cl < 0.3) {
				if (cult) {
					if (isG && !(wx.sparse && hrand(bar, step, seed + 6) < wx.sparse)) {
						const kk = [0, 2, 4, 2][gi % 4];
						play(pc.v.harm, pc.base + 12 + wx.lift + pitch(chord + kk), t, pc.groups[gi] * dur, 0.07 * wx.soft);
					}
				} else {
					if (step === 0) pulse(p, 0.25, pc.base + 12 + wx.lift + pitch(chord + 2), t, pc.half * dur * 0.95, 0.06 * wx.soft, 6);
					if (step === pc.half && !(wx.sparse && hrand(bar, step, seed + 6) < wx.sparse)) pulse(p, 0.25, pc.base + 12 + wx.lift + pitch(chord + 4), t, (spb - pc.half) * dur * 0.95, 0.06 * wx.soft, 6);
				}
			} else {
				const div = d < 0.7 && cl < 0.6 ? 2 : 1;
				if (step % div === 0 && !(sparse && d < 0.7 && hrand(bar, step, seed + 4) < sparse * 0.5) && !(wx.sparse && hrand(bar, step, seed + 6) < wx.sparse)) {
					const pat = ARPS[pc.arpStyle];
					const k = pat[(step / div) % pat.length];
					const idx = k === 3 ? chord + n : chord + k * 2;
					const pm = pc.base + 12 + wx.lift + pitch(idx);
					if (cult) play(pc.v.harm, pm, t, dur * div * 1.2, (0.06 + 0.02 * d) * wx.soft);
					else pulse(p, 0.125, pm, t, dur * div * 0.8, (0.06 + 0.02 * d) * wx.soft, 5);
				}
			}
			// sustained pad (cultural): night and calm moments
			if (cult && step === 0 && cl < 0.5 && (d < 0.5 || !day) && (bar % 2 === 0 || (GENRE_COLOUR[m.genre]?.padW ?? 0) > 0.5)) {
				play(pc.v.pad, pc.base + 12 + pitch(chord), t, spb * dur * 1.9, 0.04);
				play(pc.v.pad, pc.base + 12 + pitch(chord + 4), t, spb * dur * 1.9, 0.035);
			}
		}

		// bass
		{
			let hitB = false;
			let idx = chord;
			if (d < 0.3) hitB = isG && (gi === 0 || (gi % 2 === 0 && pc.groups.length >= 4));
			else if (d < 0.6) hitB = isG;
			else hitB = step % 2 === 0;
			if (hitB) {
				if (d < 0.6) idx = chord + [0, 0, 4, 0, 0, 0, 4, n][(bar + Math.max(gi, 0) * 2) % 8];
				else idx = chord + ((step / 2) % 2 ? n : 0);
				const blen = (d < 0.6 ? (isG ? pc.groups[gi] : 2) : 2) * dur * 0.9;
				const pm = pc.base - 12 + pitch(idx);
				if (cult) play(pc.v.bass, pm, t, blen, 0.14);
				else tri(p, pm, t, blen, 0.15);
			}
		}

		// percussion: calm → dangerous / combat / feast
		const pl = pc.dead ? 0 : Math.max(d, cl * 0.95, pc.feast ? 0.55 : 0);
		const drum = (role: 0 | 1 | 2 | 3, vel: number) => {
			if (cult) hit(kit!, bus, t, pc.v.kit, role, vel, rootHz);
			else if (role === 0 || role === 3) kick(bus, t, 0.3);
			else if (role === 1) noise(p.snr, false, t, 0.12, 0.08, 1.3);
			else noise(p.hat, true, t, off % 8 === 4 ? 0.07 : 0.035, vel > 0.1 ? 0.05 : 0.035, 1);
		};
		if (!pc.dead && pl > 0.12) {
			if (gi === 0) drum(0, cl > 0.6 ? 0.38 : 0.3);
			else if (gi > 0) {
				if (pc.groups.length >= 4 && gi % 2 === 0) { if (pl > 0.4) drum(0, 0.26); }
				else if (pl > 0.3) drum(1, 0.2);
			}
			if (!isG && pl > 0.65 && step % 2 === 0 && hrand(bar, step, seed + 1) < 0.25) drum(0, 0.18);
			if (cl > 0.55 && gi === 0) drum(3, 0.3);
			if (cl > 0.5 && !isG && off % 2 === 0 && hrand(bar, step, seed + 8) < 0.5) drum(1, 0.14);
			if (pl > 0.2) {
				const hdiv = pl < 0.5 ? 4 : 2;
				const fill = pl > 0.8 && off % hdiv !== 0 && hrand(bar, step, seed + 2) < 0.4;
				const skip = sparse > 0 && pl < 0.7 && hrand(bar, step, seed + 3) < sparse * 0.6;
				if ((off % hdiv === 0 || fill) && !skip) drum(2, fill ? 0.08 : 0.12);
			}
			if (cult && pc.feast && isG && gi % 2 === 1) clap(kit!, bus, t, 0.12);
		}

		// combat ostinato: a pulsing figure on root / fifth that thickens with danger
		if (!pc.dead && cl > 0.3 && (cl > 0.65 ? off % 2 === 0 : off % 4 === 0)) {
			const k = Math.floor(step / 2) + bar;
			const pm = pc.base + pitch(chord + [0, 0, 4, 0, 0, 2, 0, 4][k % 8]);
			const ov = 0.05 + 0.06 * cl;
			if (cult) play(pc.v.bass, pm, t, dur * 2, ov);
			else pulse(p, 0.125, pm + 12, t, dur * 1.6, ov, 0);
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
		extras();

		// heartbeat at low hp (bypasses the piece bus so it never crossfades away)
		if (!pc.dead && m.hp < 0.4 && (step === 0 || step === pc.half)) {
			const sev = clamp((0.4 - m.hp) / 0.4, 0, 1);
			kick(tone, t, 0.2 + 0.25 * sev, 0.2);
			kick(tone, t + dur * 3, 0.12 + 0.2 * sev, 0.16);
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
			} else if (p.bar >= pc.bars) {
				p.bar = 0;
				if (++loops >= 2 && pendingKey === null && st && !fight) {
					variant++;
					pendingKey = keyNow();
				}
			}
		}
		return p.nextT;
	}

	const UNDER = new Set(['dungeon', 'underdark', 'crypt']);
	function gardenStyle(): GardenStyle | null {
		if (!st || fight) return null;
		if (style === 'auto') return gardenFor(st);
		if (style === 'drift' || style === 'pastoral' || style === 'desert') return style;
		return null;
	}
	function applyFx(now: number) {
		if (!fxIn || !st) return;
		const on = flavour === 'cultural';
		const col = GENRE_COLOUR[st.genre] ?? {};
		const under = UNDER.has(st.mood);
		let room = 0.1 + (under ? 0.22 : 0) + (st.mood === 'crypt' ? 0.12 : 0) + (st.mood === 'tower' ? 0.1 : 0) + (st.mood === 'sea' ? 0.06 : 0) + (col.room ?? 0);
		let delay = (st.mood === 'sky' ? 0.3 : 0) + (st.mood === 'tower' ? 0.1 : 0) + (col.delay ?? 0);
		let spring = col.spring ?? 0;
		if (!on) { room = 0; delay = 0; spring = 0; }
		roomOut.gain.setTargetAtTime(room, now, 0.6);
		delayOut.gain.setTargetAtTime(delay, now, 0.6);
		springOut.gain.setTargetAtTime(spring, now, 0.6);
		roomPre.frequency.setTargetAtTime(under ? 2200 : 5200, now, 0.6);
	}

	function tick() {
		const c = ctx;
		if (!c || !enabled || !st) return;
		const now = c.currentTime;
		if (kit) sweep(kit);

		// brightness: day opens the filter, night and depth darken it; the sea is muffled and swells
		const sea = flavour === 'cultural' && st.mood === 'sea';
		const underMul = flavour === 'cultural' && UNDER.has(st.mood) ? 0.85 : 1;
		const swell = sea ? 1 + 0.25 * Math.sin(now * 0.55) : 1;
		const cut = (st.day ? 7000 : 2600) * Math.max(0.4, 1 - Math.max(0, st.depth) * 0.04) * (MOODS[st.mood]?.bright ?? 1) * (style === 'ambient' ? 0.6 : 1) * wxCfg(st.wx).cut * (sea ? 0.4 * swell : 1) * underMul;
		if (Math.abs(cut - lastCut) > (sea ? 20 : 1)) {
			tone.frequency.setTargetAtTime(cut, now, 0.4);
			lastCut = cut;
		}
		if (now - fxT > 0.5) {
			fxT = now;
			applyFx(now);
			if (fight && !st.combat && performance.now() > fightUntil) {
				fight = false;
				const k = keyNow();
				pendingKey = k === activeKey ? null : k;
			}
		}

		// garden styles (drift / pastoral / desert) play outside combat; the sequencer keeps the fights
		const gs = gardenStyle();
		if (gs && garden) {
			if (cur) { retire(cur, now + 0.05, 2.5); cur = null; activeKey = ''; }
			garden.tick(st, gs, flavour === 'cultural', now);
			return;
		}
		if (garden?.playing) { garden.stop(now + 0.05, 2.5); activeKey = ''; pendingKey = keyNow(); }

		if (!cur || cur.done) {
			if (pendingKey !== null) {
				cur = newPlayer(makePiece(st, style, flavour, fight, variant), now + 0.05);
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
				cur = newPlayer(makePiece(st, style, flavour, fight, variant), at);
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
	function requeue() {
		if (!st) return;
		const key = keyNow();
		pendingKey = key === activeKey ? null : key;
		if (enabled && ensure()) startTimer();
	}
	function syncFlavour() {
		const now = Date.now();
		if (now - flavT < 1500) return;
		flavT = now;
		try {
			const f = loadSettings().instrumentFlavour;
			if (f !== flavour) { flavour = f; requeue(); }
		} catch { /* settings unavailable */ }
	}

	return {
		setState(m: QudMusic) {
			if (disposed) return;
			st = m;
			syncFlavour();
			const bolt = m.bolt ?? 0;
			if (lastBolt >= 0 && bolt > lastBolt && enabled && ensure()) thunder();
			lastBolt = bolt;
			updateFight(m.combat ?? false);
			const key = keyNow();
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
		setStyle(sty: MusicStyle) {
			if (disposed) return;
			flavT = 0;
			syncFlavour();
			if (sty === style) return;
			style = sty;
			requeue();
		},
		setFlavour(f: InstrumentFlavour) {
			if (disposed || f === flavour) return;
			flavour = f;
			flavT = Date.now();
			requeue();
		},
		setEnabled(on: boolean) {
			if (disposed) return;
			enabled = on;
			if (on) {
				if (suspendT !== null) { clearTimeout(suspendT); suspendT = null; }
				if (!ensure()) return;
				applyVolume();
				if (cur) cur.nextT = Math.max(cur.nextT, ctx!.currentTime + 0.05);
				startTimer();
			} else {
				stopTimer();
				applyVolume();
				const c = ctx;
				if (suspendT !== null) clearTimeout(suspendT);
				if (c) suspendT = setTimeout(() => { suspendT = null; if (!enabled && c.state === 'running') c.suspend().catch(() => {}); }, 400);
			}
		},
		dispose() {
			disposed = true;
			stopTimer();
			fading.forEach(clearTimeout);
			fading.clear();
			if (suspendT !== null) { clearTimeout(suspendT); suspendT = null; }
			cur = null;
			pendingKey = null;
			garden?.dispose();
			garden = null;
			if (kit) disposeKit(kit);
			kit = null;
			const c = ctx;
			ctx = null;
			if (c) c.close().catch(() => {});
		},
	};
}
