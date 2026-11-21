// Culture-specific music: (1) pure helpers that map a culture's instrument / scale / form words onto the
// synth's timbres, scales and feels (profileFor, for music.ts and garden.ts to bias their choices), and (2) a quiet
// positional layer for live performers (bards, buskers, feast drummers, choirs) and ceremonies.
// Everything is WebAudio on the context and output node given by the coordinator.

import { INST, createKit, disposeKit, sweep, hit, drone2, type Kit } from './instruments';
import { FEELS, GENERIC, PROFILES, SCALES, type Profile } from './scales';

// ---------- frame data (engine/musicstyle.go MstStyle) ----------
export interface MstPerformer {
	id: number;
	kind: string; // bard minstrel innkeeper | temple choir | busker | drummers at a feast | street band
	x: number;
	y: number;
	distance: number;
	dx: number;
	dy: number;
	instrument: string;
}
export interface MusicStyleFrame {
	culture: number;
	sphere: string;
	instruments: string[];
	forms: string[];
	scaleWord: string;
	tempoHint: 'slow' | 'moderate' | 'lively' | string;
	feast: string;
	feastKind: string;
	song: string;
	rite: string; // wedding funeral birth age feast while a ceremony screen is up
	genre: string;
	performer?: MstPerformer;
}

// ---------- keyword maps ----------
export interface InstMap { inst?: string; kit?: string; drone?: 'tonic' | 'fifth' }
const INST_WORDS: [string, InstMap][] = [
	['throat singing', { inst: 'choir', drone: 'fifth' }], ['throat', { inst: 'choir', drone: 'fifth' }],
	['didgeridoo', { inst: 'brass', drone: 'tonic' }], ['bagpipe', { inst: 'bagpipe', drone: 'fifth' }],
	['frame drum', { kit: 'frame' }], ['kettledrum', { kit: 'timpani' }], ['timpani', { kit: 'timpani' }],
	['taiko', { kit: 'taiko' }], ['djembe', { kit: 'djembe' }], ['darbuka', { kit: 'darbuka' }], ['doumbek', { kit: 'darbuka' }],
	['tabla', { kit: 'tabla' }], ['bodhran', { kit: 'bodhran' }], ['tin', { kit: 'tincan' }], ['drum machine', { kit: 'electro' }],
	['gong', { kit: 'gong' }], ['cymbal', { kit: 'gong' }], ['rattle', { kit: 'frame' }], ['tambour', { kit: 'frame' }], ['drum', { kit: 'frame' }],
	['hurdy', { inst: 'violin', drone: 'fifth' }], ['panpipe', { inst: 'panpipe' }], ['pan flute', { inst: 'panpipe' }], ['pipe', { inst: 'flute' }],
	['oud', { inst: 'lute' }], ['lute', { inst: 'lute' }], ['lyre', { inst: 'lute' }], ['biwa', { inst: 'koto' }], ['koto', { inst: 'koto' }], ['guzheng', { inst: 'koto' }],
	['zither', { inst: 'koto' }], ['dulcimer', { inst: 'harp' }], ['harp', { inst: 'harp' }], ['harpsichord', { inst: 'harpsi' }], ['guitar', { inst: 'guitar' }],
	['banjo', { inst: 'mandolin' }], ['mandolin', { inst: 'mandolin' }], ['sitar', { inst: 'sitar', drone: 'fifth' }], ['veena', { inst: 'sitar' }], ['kora', { inst: 'kora' }],
	['erhu', { inst: 'erhu' }], ['rebec', { inst: 'violin' }], ['fiddle', { inst: 'violin' }], ['violin', { inst: 'violin' }], ['vielle', { inst: 'violin' }], ['viol', { inst: 'violin' }],
	['shawm', { inst: 'shawm' }], ['zurna', { inst: 'shawm' }], ['oboe', { inst: 'shawm' }], ['duduk', { inst: 'duduk' }],
	['shakuhachi', { inst: 'shaku' }], ['whistle', { inst: 'whistle' }], ['ocarina', { inst: 'whistle' }], ['flute', { inst: 'flute' }], ['reed', { inst: 'flute' }],
	['balafon', { inst: 'marimba' }], ['marimba', { inst: 'marimba' }], ['xylophone', { inst: 'marimba' }], ['kalimba', { inst: 'kalimba' }], ['mbira', { inst: 'kalimba' }],
	['gamelan', { inst: 'gamelan' }], ['music box', { inst: 'musicbox' }], ['vibraphone', { inst: 'vibes' }],
	['choir', { inst: 'choir' }], ['chant', { inst: 'choir' }], ['hymn', { inst: 'choir' }], ['voice', { inst: 'choir' }], ['sing', { inst: 'choir' }],
	['bell', { inst: 'bell' }], ['chime', { inst: 'bell' }], ['horn', { inst: 'brass' }], ['trumpet', { inst: 'brass' }], ['trombone', { inst: 'brass' }], ['brass', { inst: 'brass' }],
	['organ', { inst: 'organ' }], ['accordion', { inst: 'calliope' }], ['concertina', { inst: 'calliope' }], ['calliope', { inst: 'calliope' }],
	['harmonica', { inst: 'harmonica' }], ['theremin', { inst: 'theremin' }], ['synth', { inst: 'synthlead' }], ['radio', { inst: 'radio' }],
];
const PLUCKED = new Set(['lute', 'guitar', 'koto', 'kora', 'sitar', 'harp', 'mandolin', 'harpsi', 'kalimba', 'marimba', 'gamelan', 'musicbox', 'vibes']);

/** one free-text instrument word onto a timbre (INST key), a drum kit and/or a drone style; {} when unknown */
export function mapInstrument(word: string): InstMap {
	const w = word.toLowerCase();
	for (const [k, v] of INST_WORDS) if (w.includes(k)) return v.inst && !INST[v.inst] ? {} : v;
	return {};
}

const SCALE_WORDS: [string, string][] = [
	['hirajoshi', 'hirajoshi'], ['insen', 'insen'], ['pelog', 'pelog'], ['slendro', 'slendro'], ['bhairav', 'bhairav'], ['todi', 'todi'], ['marwa', 'marwa'],
	['hijaz', 'hijaz'], ['bayati', 'bayati'], ['homayun', 'homayun'], ['rast', 'rast'], ['maqam', 'rast'], ['makam', 'rast'], ['dastgah', 'homayun'], ['raga', 'bhairav'], ['microton', 'rast'], ['quarter', 'rast'],
	['tizita', 'tizita'], ['ambassel', 'ambassel'], ['whole-tone', 'wholetone'], ['whole tone', 'wholetone'], ['wholetone', 'wholetone'], ['octatonic', 'octatonic'], ['diminished', 'octatonic'], ['chromatic', 'octatonic'],
	['harmonic minor', 'harmonicMinor'], ['phrygian dominant', 'phrygDom'], ['phrygian', 'phrygian'], ['dorian', 'dorian'], ['lydian', 'lydian'], ['mixolydian', 'mixolydian'], ['locrian', 'locrian'],
	['aeolian', 'aeolian'], ['ionian', 'ionian'], ['minor pent', 'minPent'], ['major pent', 'majPent'], ['pentatonic', 'majPent'], ['pentachord', 'pentachord'], ['hexatonic', 'hexa'], ['six-note', 'hexa'],
	['minor', 'aeolian'], ['major', 'ionian'], ['modal', 'dorian'], ['yo scale', 'yo'], ['ryukyu', 'ryukyu'], ['eerie', 'uncanny'], ['uncanny', 'uncanny'],
];
/** a scale description onto a SCALES key, or '' */
export function mapScale(word: string): string {
	const w = word.toLowerCase();
	if (w.includes('pentatonic') && w.includes('minor')) return 'minPent';
	for (const [k, v] of SCALE_WORDS) if (w.includes(k) && SCALES[v]) return v;
	return '';
}

export interface FormMap { feel?: string; mood?: 'lament' | 'lilt' | 'lull' | 'round' | 'call' | 'march' }
const FORM_WORDS: [string, FormMap][] = [
	['dirge', { feel: 'four', mood: 'lament' }], ['lament', { feel: 'four', mood: 'lament' }], ['elegy', { feel: 'four', mood: 'lament' }], ['requiem', { feel: 'four', mood: 'lament' }],
	['lullaby', { feel: 'waltz', mood: 'lull' }], ['cradle', { feel: 'waltz', mood: 'lull' }],
	['call and response', { feel: 'tresillo', mood: 'call' }], ['call-and-response', { feel: 'tresillo', mood: 'call' }], ['antiphon', { feel: 'tresillo', mood: 'call' }],
	['round', { feel: 'six8', mood: 'round' }], ['canon', { feel: 'six8', mood: 'round' }], ['reel', { feel: 'six8' }], ['jig', { feel: 'nine8' }], ['dance', { feel: 'six8' }],
	['ballad', { feel: 'six8', mood: 'lilt' }], ['lay', { feel: 'six8', mood: 'lilt' }], ['epic', { feel: 'four' }], ['saga', { feel: 'four' }], ['march', { feel: 'four', mood: 'march' }],
	['work song', { feel: 'swing' }], ['shanty', { feel: 'swing' }], ['hymn', { feel: 'four' }], ['waltz', { feel: 'waltz' }], ['drone', { feel: 'five8' }],
];
export function mapForm(word: string): FormMap {
	const w = word.toLowerCase();
	for (const [k, v] of FORM_WORDS) if (w.includes(k) && (!v.feel || FEELS[v.feel])) return v;
	return {};
}

const uniq = (a: string[]) => a.filter((v, i) => v && a.indexOf(v) === i);

/** what a style's words add to a profile: leads, plucks, kit, drone, scales, feels, mood */
export interface Bias { leads: string[]; plucks: string[]; kit?: string; drone?: 'tonic' | 'fifth'; scale?: string; feels: string[]; moods: string[]; tempo: number }
export function biasOf(style: MusicStyleFrame | null | undefined): Bias {
	const b: Bias = { leads: [], plucks: [], feels: [], moods: [], tempo: 1 };
	if (!style) return b;
	for (const w of style.instruments ?? []) {
		const m = mapInstrument(w);
		if (m.inst) (PLUCKED.has(m.inst) ? b.plucks : b.leads).push(m.inst);
		if (m.kit && !b.kit) b.kit = m.kit;
		if (m.drone && !b.drone) b.drone = m.drone;
	}
	b.leads = uniq(b.leads); b.plucks = uniq(b.plucks);
	b.scale = mapScale(style.scaleWord ?? '') || undefined;
	for (const w of style.forms ?? []) {
		const f = mapForm(w);
		if (f.feel) b.feels.push(f.feel);
		if (f.mood) b.moods.push(f.mood);
	}
	b.feels = uniq(b.feels); b.moods = uniq(b.moods);
	if (style.tempoHint === 'slow') b.tempo = 0.8;
	else if (style.tempoHint === 'lively') b.tempo = 1.12;
	if (b.moods.includes('lull')) b.tempo *= 0.9;
	return b;
}

/**
 * A culture's profile: the base profile (PROFILES[sphere] or GENERIC) with the pack's instruments, scale word and
 * song forms moved to the front of its choices. Pure; returns `base` itself when there is nothing to add.
 * Call it from music.ts / garden.ts where they read `PROFILES[sphere]`.
 */
export function profileFor(style: MusicStyleFrame | null | undefined, base?: Profile): Profile {
	const b0 = base ?? (style && PROFILES[style.sphere]) ?? GENERIC;
	if (!style) return b0;
	const b = biasOf(style);
	if (!b.leads.length && !b.plucks.length && !b.kit && !b.drone && !b.scale && !b.feels.length && b.tempo === 1) return b0;
	const leads = uniq([...b.leads, ...b.plucks.slice(0, 1), ...b0.leads]).slice(0, 5);
	const feastLeads = uniq([...b.leads, ...b.plucks.slice(0, 1), ...(b0.feastLeads ?? b0.leads)]).slice(0, 4);
	return {
		...b0,
		scales: uniq([...(b.scale ? [b.scale] : []), ...b0.scales]),
		feels: uniq([...b.feels, ...b0.feels]),
		leads, feastLeads,
		plucks: uniq([...b.plucks, ...b0.plucks]).slice(0, 4),
		kit: b.kit ?? b0.kit,
		feastKit: b.kit ?? b0.feastKit,
		drone: b.drone ?? b0.drone,
		tempo: b0.tempo * b.tempo,
		lament: b.leads.includes('choir') || b.moods.includes('lament') ? (b.leads.includes('choir') ? 'choir' : b0.lament) : b0.lament,
	};
}

// ---------- deterministic phrase grammar ----------
function hash32(s: string): number {
	let h = 2166136261 >>> 0;
	for (let i = 0; i < s.length; i++) { h ^= s.charCodeAt(i); h = Math.imul(h, 16777619) >>> 0; }
	return h >>> 0;
}
function rng(seed: number): () => number {
	let a = seed >>> 0 || 1;
	return () => { a = (a + 0x6d2b79f5) >>> 0; let t = a; t = Math.imul(t ^ (t >>> 15), t | 1); t ^= t + Math.imul(t ^ (t >>> 7), t | 61); return ((t ^ (t >>> 14)) >>> 0) / 4294967296; };
}

export interface PhraseEvent { t: number; kind: 'lead' | 'bass' | 'drone' | 'drum'; midi: number; dur: number; vel: number; role?: 0 | 1 | 2 | 3 }
export interface Phrase { events: PhraseEvent[]; length: number; bars: number; lead: string; kit: string; bpm: number }

export type Mood = 'plain' | 'feast' | 'lament' | 'wedding' | 'drums';

/** pure and deterministic: the same (id, culture, kind, mood) always gives the same phrase */
export function genPhrase(id: number, style: MusicStyleFrame, kind: string, mood: Mood): Phrase {
	const base = PROFILES[style.sphere] ?? GENERIC;
	const prof = profileFor(style, base);
	const r = rng(hash32(`${id}:${style.culture}:${kind}:${mood}`));
	const pick = <T,>(a: T[]): T => a[Math.floor(r() * a.length) % a.length];

	let scaleKey = prof.scales.length ? pick(prof.scales.slice(0, 3)) : pick(['dorian', 'mixolydian', 'aeolian', 'majPent', 'ionian']);
	if (mood === 'lament') scaleKey = prof.scales.find((s) => ['aeolian', 'phrygian', 'minPent', 'hijaz', 'bayati', 'insen', 'hirajoshi', 'todi', 'bhairav'].includes(s)) ?? 'aeolian';
	else if (mood === 'wedding') scaleKey = prof.scales.find((s) => ['majPent', 'ionian', 'mixolydian', 'lydian', 'yo', 'rast'].includes(s)) ?? 'majPent';
	const scale = SCALES[scaleKey] ?? SCALES.dorian;

	let feelKey: string;
	if (mood === 'feast' || mood === 'drums') feelKey = prof.feastFeel ?? pick(prof.feels);
	else if (mood === 'wedding') feelKey = 'six8';
	else if (mood === 'lament') feelKey = 'four';
	else feelKey = pick(prof.feels);
	const feel = FEELS[feelKey] ?? FEELS.four;

	const pluck = pick(prof.plucks.length ? prof.plucks : ['lute']);
	let lead: string;
	if (kind === 'temple choir' || mood === 'lament') lead = mood === 'lament' && kind !== 'temple choir' ? (prof.lament ?? 'duduk') : 'choir';
	else if (kind === 'bard' || kind === 'minstrel') lead = pluck;
	else lead = pick(mood === 'feast' ? (prof.feastLeads ?? prof.leads) : prof.leads);
	const wanted = mapInstrument(kind).inst; // unused words stay harmless
	void wanted;
	if (!INST[lead]) lead = 'lute';
	const kit = mood === 'feast' || mood === 'drums' ? (prof.feastKit ?? prof.kit) : prof.kit;

	const bpmBase = 84 * prof.tempo * (mood === 'feast' || mood === 'drums' ? 1.2 : mood === 'lament' ? 0.62 : mood === 'wedding' ? 1.05 : 1);
	const bpm = Math.max(44, Math.min(132, bpmBase * (0.95 + r() * 0.1)));
	const stepSec = 60 / bpm / 4;
	const bars = r() < 0.5 ? 8 : 16;
	const secLen = bars === 8 ? 2 : 4; // bars per section, 4 sections: A A B A
	const spb = feel.spb;
	const root = 52 + (hash32(`root:${style.culture}:${id}`) % 9); // E3..C4

	const n = scale.length;
	const note = (deg: number) => root + 12 * Math.floor(deg / n) + scale[((deg % n) + n) % n];
	const starts: number[] = []; let acc = 0;
	for (const g of feel.groups) { starts.push(acc); acc += g; }

	type Nt = { s: number; len: number; deg: number };
	const makeBar = (rest: number, cadence: boolean, prevDeg: number): Nt[] => {
		const onsets: number[] = [];
		for (let s = 0; s < spb; s += 2) {
			if (starts.includes(s) || r() < (mood === 'lament' ? 0.15 : 0.45)) if (r() > rest || s === 0) onsets.push(s);
		}
		if (onsets.length === 0) onsets.push(0);
		const out: Nt[] = [];
		let deg = prevDeg;
		onsets.forEach((s, i) => {
			const step = pick([-2, -1, -1, 0, 1, 1, 2, 3]);
			deg = Math.max(-2, Math.min(n + 3, deg + step));
			const nxt = i + 1 < onsets.length ? onsets[i + 1] : spb;
			out.push({ s, len: Math.max(2, nxt - s), deg });
		});
		if (cadence) { const l = out[out.length - 1]; l.deg = 0; l.len = spb - l.s; }
		return out;
	};
	const makeSection = (rest: number): Nt[][] => {
		const bs: Nt[][] = []; let d = pick([0, 2, 4]);
		for (let i = 0; i < secLen; i++) { const bar = makeBar(rest, false, d); bs.push(bar); d = bar[bar.length - 1].deg; }
		return bs;
	};
	const A = makeSection(0.1), B = makeSection(0.25);
	const Aend = A.map((x) => x.map((y) => ({ ...y })));
	const lastA = Aend[Aend.length - 1]; lastA[lastA.length - 1].deg = 0; // A' resolves home
	const sections = [A, A, B, Aend];

	const ev: PhraseEvent[] = [];
	const barSec = spb * stepSec;
	const leadVel = mood === 'lament' ? 0.42 : 0.5;
	const melodic = kind !== 'drummers at a feast';
	const drums = mood === 'feast' || mood === 'drums' || mood === 'wedding';
	sections.forEach((sec, si) => {
		sec.forEach((bar, bi) => {
			const b = si * secLen + bi;
			const t0 = b * barSec;
			if (melodic) {
				bar.forEach((nt, i) => {
					const accent = starts.includes(nt.s) ? 1 : 0.8;
					ev.push({ t: t0 + nt.s * stepSec + (feel.swing && nt.s % 4 === 2 ? feel.swing * stepSec : 0), kind: 'lead', midi: note(nt.deg) + 12, dur: nt.len * stepSec * (mood === 'lament' ? 1.05 : 0.95), vel: leadVel * accent * (i === 0 ? 1 : 0.92) });
				});
				if (mood !== 'lament' && mood !== 'drums') {
					ev.push({ t: t0, kind: 'bass', midi: root - 12 + (b % 2 && r() < 0.4 ? scale[Math.min(4, n - 1)] : 0), dur: barSec * 0.9, vel: 0.32 });
					for (const s of starts.slice(1)) if (r() < 0.6) ev.push({ t: t0 + s * stepSec, kind: 'bass', midi: root + scale[(s + b) % Math.min(3, n)], dur: stepSec * 3, vel: 0.2 });
				}
			}
			if (prof.drone !== 'none' && mood !== 'drums' && b % secLen === 0) {
				ev.push({ t: t0, kind: 'drone', midi: root - 24 + (prof.drone === 'fifth' ? 7 : 0), dur: barSec * secLen, vel: 0.12 });
			}
			if (drums) {
				starts.forEach((s, gi) => ev.push({ t: t0 + s * stepSec, kind: 'drum', midi: 0, dur: 0.2, role: gi === 0 ? 0 : 1, vel: gi === 0 ? 0.42 : 0.28 }));
				for (let s = 2; s < spb; s += 4) if (!starts.includes(s) && r() < 0.45) ev.push({ t: t0 + s * stepSec, kind: 'drum', midi: 0, dur: 0.1, role: 2, vel: 0.16 });
			} else if (mood === 'lament' && bi === 0) {
				ev.push({ t: t0, kind: 'drum', midi: 0, dur: 0.5, role: 0, vel: 0.2 });
			}
		});
	});
	ev.sort((x, y) => x.t - y.t);
	return { events: ev, length: bars * barSec, bars, lead, kit, bpm };
}

// ---------- the live layer ----------
export interface CultureMusicOpts { onDuck?: (amount: number) => void }
export interface CultureMusic {
	/** call every frame with `frame.mstyle` (or null) and `frame.mode` */
	set(style: MusicStyleFrame | null | undefined, mode?: string): void;
	setEnabled(on: boolean): void;
	setVolume(v: number): void;
	dispose(): void;
}

const RANGE = 14; // tiles
const PEAK = 0.045; // about -27 dBFS at the layer output before the master
const LOOKAHEAD = 0.7;
const TALK_MODES = new Set(['talk', 'trade', 'shop', 'confirm']);
const CEREMONY_MODES = new Set(['fwc']);

export function createCultureMusic(ctx: AudioContext, dest: AudioNode, opts: CultureMusicOpts = {}): CultureMusic {
	const kit: Kit = createKit(ctx);
	const out = ctx.createGain(); out.gain.value = 0;
	const comp = ctx.createDynamicsCompressor();
	comp.threshold.value = -34; comp.knee.value = 12; comp.ratio.value = 10; comp.attack.value = 0.01; comp.release.value = 0.3;
	const lp = ctx.createBiquadFilter(); lp.type = 'lowpass'; lp.frequency.value = 6000; lp.Q.value = 0.5;
	const pan = ctx.createStereoPanner();
	const bus = ctx.createGain(); bus.gain.value = 1; // instruments feed this
	bus.connect(lp); lp.connect(pan); pan.connect(comp); comp.connect(out); out.connect(dest);

	let enabled = true, volume = 1, disposed = false;
	let timer: ReturnType<typeof setInterval> | null = null;
	let key = '';
	let phrase: Phrase | null = null;
	let base = 0, idx = 0;
	let lastLevel = 0;
	let duckSent = 0;
	let active: number[] = []; // end times of voices we started
	let wantOn = false;

	const stopTimer = () => { if (timer) { clearInterval(timer); timer = null; } };
	const sendDuck = (a: number) => {
		a = Math.round(a * 20) / 20;
		if (a !== duckSent) { duckSent = a; try { opts.onDuck?.(a); } catch { /* host callback */ } }
	};

	function tick(): void {
		if (disposed) return;
		if (ctx.state !== 'running') { stopTimer(); return; }
		sweep(kit);
		const now = ctx.currentTime;
		active = active.filter((e) => e > now);
		if (!phrase || !wantOn) {
			if (lastLevel === 0 && now > 0) stopTimer();
			return;
		}
		const ph = phrase;
		let guard = 0;
		while (guard++ < 200) {
			const e = ph.events[idx];
			const at = base + e.t;
			if (at > now + LOOKAHEAD) break;
			idx++;
			if (idx >= ph.events.length) { idx = 0; base += ph.length; }
			if (at < now - 0.05) continue;
			if (active.length >= 12) continue;
			play(ph, e, Math.max(at, now));
		}
	}
	function play(ph: Phrase, e: PhraseEvent, at: number): void {
		const v = e.vel;
		if (e.kind === 'drum') { hit(kit, bus, at, ph.kit, e.role ?? 1, v, 110); active.push(at + 0.3); return; }
		if (e.kind === 'drone') { drone2(kit, bus, at, e.midi, Math.min(e.dur, 8), v, 'sine'); active.push(at + e.dur); return; }
		const inst = INST[e.kind === 'bass' ? 'lute' : ph.lead] ?? INST.lute;
		inst(kit, bus, at, e.midi, e.dur, v);
		active.push(at + Math.min(e.dur, 2));
	}
	function startTimer(): void {
		if (!timer && !disposed) timer = setInterval(tick, 220);
	}

	function moodOf(style: MusicStyleFrame, p: MstPerformer, mode: string): Mood {
		if (CEREMONY_MODES.has(mode) || style.rite) {
			if (style.rite === 'funeral') return 'lament';
			if (style.rite === 'wedding' || style.rite === 'birth' || style.rite === 'age') return 'wedding';
			return 'feast';
		}
		if (p.kind === 'drummers at a feast') return 'drums';
		if (style.feast) return 'feast';
		return 'plain';
	}

	function set(style: MusicStyleFrame | null | undefined, mode = ''): void {
		if (disposed) return;
		const p = style?.performer;
		const talkingClose = !!p && TALK_MODES.has(mode) && p.distance <= 5;
		const ceremony = !!style && (CEREMONY_MODES.has(mode) || !!style.rite);
		// a ceremony is its own performer: the whole procession, centred on the player
		const perf: MstPerformer | undefined = p ?? (ceremony && style ? { id: 7000 + style.culture, kind: 'ceremony', x: 0, y: 0, distance: 3, dx: 0, dy: 0, instrument: '' } : undefined);
		const on = enabled && !!style && !!perf && perf.distance <= RANGE && !talkingClose && !document.hidden;
		wantOn = on;
		const now = ctx.currentTime;
		if (!on || !style || !perf) {
			if (lastLevel !== 0) { out.gain.cancelScheduledValues(now); out.gain.setTargetAtTime(0, now, 0.5); lastLevel = 0; }
			sendDuck(0);
			return;
		}
		const mood = moodOf(style, perf, mode);
		const k = `${perf.id}:${style.culture}:${perf.kind}:${mood}:${style.feast}`;
		if (k !== key || !phrase) {
			key = k;
			phrase = genPhrase(perf.id, style, perf.kind, mood);
			// the old phrase fades (gain target below), the new one starts after a short gap
			out.gain.cancelScheduledValues(now);
			out.gain.setTargetAtTime(0, now, 0.15);
			base = now + 0.9; idx = 0; active = [];
		}
		const d = Math.max(0, perf.distance);
		const near = Math.max(0, 1 - d / RANGE);
		const level = Math.min(PEAK, PEAK * near * near * 1.4) * volume * (mood === 'drums' ? 0.9 : 1);
		const startOk = now >= base - 0.4; // fade in over ~1.8 s, after the gap
		out.gain.cancelScheduledValues(now);
		out.gain.setTargetAtTime(startOk ? level : 0, now, startOk ? 0.7 : 0.15);
		lastLevel = level || 0.0001;
		lp.frequency.setTargetAtTime(700 + 6500 * near * near, now, 0.3);
		const len = Math.hypot(perf.dx, perf.dy) || 1;
		pan.pan.setTargetAtTime(ceremony && !p ? 0 : Math.max(-0.85, Math.min(0.85, (perf.dx / len) * Math.min(1, d / 5) * 0.85)), now, 0.2);
		sendDuck(d < 5 ? 0.35 * (1 - d / 5) : 0);
		startTimer();
	}

	function setEnabled(on: boolean): void {
		enabled = on;
		if (!on) {
			wantOn = false;
			const now = ctx.currentTime;
			out.gain.cancelScheduledValues(now); out.gain.setTargetAtTime(0, now, 0.3);
			lastLevel = 0; key = ''; phrase = null; active = [];
			sendDuck(0);
			setTimeout(() => { if (!enabled && !disposed) stopTimer(); }, 1500);
		}
	}
	function setVolume(v: number): void { volume = Math.max(0, Math.min(1, v)); }
	function dispose(): void {
		disposed = true; wantOn = false;
		stopTimer();
		sendDuck(0);
		try { out.gain.cancelScheduledValues(0); out.gain.value = 0; } catch { /* closed */ }
		for (const n of [bus, lp, pan, comp, out]) { try { n.disconnect(); } catch { /* gone */ } }
		disposeKit(kit);
		phrase = null; active = [];
	}
	return { set, setEnabled, setVolume, dispose };
}
