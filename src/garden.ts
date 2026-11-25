// "Garden" music: generative, unhurried soundtrack styles that replace the bar-by-bar sequencer
// outside combat.
//
//   drift    — Brian Eno / C418: incommensurate loops (each voice repeats a short phrase on its own
//              prime-ish period, so the texture never lines up the same way twice), sparse felt-piano
//              phrases with rubato, a low pad, long reverb, and real silences between passages.
//   pastoral — Stardew: warm, tuneful 4/4 in a major mode; fingerpicked guitar or harp, a flute or
//              whistle melody built from a repeating motif (A A B A), soft shaker; seasons swap the band.
//   desert   — Caves of Qud: modal drones (hijaz / phrygian dominant / bayati …) through a slowly
//              breathing filter, ornamented synth or oud phrases, glassy bells, an odd-length analog
//              arpeggio that fades in and out, and a quiet hand drum in the open desert.
//
// A Scene is composed from a seeded rng (world seed + place + season + time of day), so a place keeps
// its own music. Scenes cross-fade when the key changes. All voices are plain WebAudio nodes.

import type { QudMusic } from './qud';
import { INST, type Kit } from './instruments';
import { SCALES, PROFILES, type Profile } from './scales';
import { profileFor, styleKey } from './culturemusic';

export type GardenStyle = 'drift' | 'pastoral' | 'desert';

const mtof = (m: number) => 440 * Math.pow(2, (m - 69) / 12);
const clamp = (v: number, a: number, b: number) => Math.min(b, Math.max(a, v));
function hash(s: string): number {
	let h = 2166136261;
	for (let i = 0; i < s.length; i++) h = Math.imul(h ^ s.charCodeAt(i), 16777619);
	return h >>> 0;
}
function rngOf(seed: number): () => number {
	let a = seed >>> 0;
	return () => {
		a = (a + 0x6d2b79f5) >>> 0;
		let t = a;
		t = Math.imul(t ^ (t >>> 15), t | 1);
		t ^= t + Math.imul(t ^ (t >>> 7), t | 61);
		return ((t ^ (t >>> 14)) >>> 0) / 4294967296;
	};
}
const pick = <T>(r: () => number, xs: T[]): T => xs[Math.floor(r() * xs.length) % xs.length];

const DESERT_BIOMES = new Set([1, 2, 3, 13, 14, 15, 16]);
const STRANGE_GENRES = new Set(['wasteland', 'twilight', 'cyberpunk', 'biopunk', 'atompunk']);
const UNDER = new Set(['dungeon', 'crypt', 'underdark', 'tower']);

/** the 'auto' style: which garden suits this moment */
export function gardenFor(m: QudMusic): GardenStyle {
	if (m.mood === 'dead') return 'drift';
	if (UNDER.has(m.mood)) return 'desert';
	if (m.mood === 'town' || m.mood === 'world') return STRANGE_GENRES.has(m.genre) && !m.feast ? 'desert' : 'pastoral';
	if (m.mood === 'sky' || m.mood === 'sea') return 'drift';
	// the wild
	if (DESERT_BIOMES.has(m.biome | 0) || STRANGE_GENRES.has(m.genre)) return 'desert';
	if (m.day && (m.season ?? 0) !== 3 && [4, 5, 6, 8].includes(m.biome | 0)) return 'pastoral';
	return 'drift';
}

export function gardenKey(m: QudMusic, gs: GardenStyle, cultural = true): string {
	return `${gs}|${cultural ? 1 : 0}|${m.seed}|${m.mood}|${m.genre}|${m.biome | 0}|${(m.season ?? 0) | 0}|${m.sphere ?? ''}|${styleKey(m)}|${m.day ? 1 : 0}|${m.feast ? 1 : 0}`;
}

export interface Garden {
	/** schedule audio up to the look-ahead; call from the music timer */
	tick(m: QudMusic, gs: GardenStyle, cultural: boolean, now: number, instant?: boolean): void;
	/** fade everything out (combat starts, or the style changed) */
	stop(at: number, fade?: number): void;
	readonly playing: boolean;
	dispose(): void;
}

interface Scene {
	key: string;
	bus: GainNode;
	run(until: number, m: QudMusic): void;
}

export function createGarden(c: AudioContext, kit: Kit, out: AudioNode): Garden {
	// a long, dark hall: Eno and C418 live in the tail
	const hall = c.createConvolver();
	hall.buffer = impulse(c, 7, 3.4);
	const hallIn = c.createGain();
	hallIn.gain.value = 1;
	const hallLp = c.createBiquadFilter();
	hallLp.type = 'lowpass';
	hallLp.frequency.value = 4200;
	const hallOut = c.createGain();
	hallOut.gain.value = 0.45;
	hallIn.connect(hallLp).connect(hall).connect(hallOut).connect(out);

	const AHEAD = 0.6;
	/** per-style bus level, balanced to leave headroom for weather SFX and prevent clipping */
	const LEVEL: Record<GardenStyle, number> = { drift: 1.45, pastoral: 0.92, desert: 1.15 };
	let scene: Scene | null = null;
	const dying = new Set<GainNode>();

	// ---------- voices ----------
	function end(src: AudioScheduledSourceNode, ...nodes: AudioNode[]) {
		src.onended = () => { for (const n of nodes) { try { n.disconnect(); } catch { /* gone */ } } };
	}
	function o(type: OscillatorType, f: number, t: number, stop: number, detune = 0): OscillatorNode {
		const x = c.createOscillator();
		x.type = type;
		x.frequency.value = f;
		x.detune.value = detune;
		x.start(t);
		x.stop(stop);
		return x;
	}
	/** felt piano: a few soft partials, a muffled hammer, long decay that shortens up the keyboard */
	function piano(d: AudioNode, t: number, midi: number, vel: number) {
		const f = mtof(midi);
		const dec = clamp(6.5 - (midi - 48) * 0.07, 1.8, 7);
		const lp = c.createBiquadFilter();
		lp.type = 'lowpass';
		lp.frequency.setValueAtTime(700 + vel * 9000 + f * 1.5, t);
		lp.frequency.exponentialRampToValueAtTime(400 + f, t + dec);
		const g = c.createGain();
		g.gain.setValueAtTime(0, t);
		g.gain.linearRampToValueAtTime(vel, t + 0.006);
		g.gain.exponentialRampToValueAtTime(vel * 0.35, t + 0.4);
		g.gain.exponentialRampToValueAtTime(0.0001, t + dec);
		lp.connect(g).connect(d);
		const stop = t + dec + 0.05;
		let last: OscillatorNode | null = null;
		const nodes: AudioNode[] = [lp, g];
		for (const [r, a] of [[1, 1], [2.001, 0.32], [3.003, 0.1], [4.01, 0.05]] as [number, number][]) {
			const x = o('sine', f * r, t, stop, (midi * 7 + r * 3) % 5 - 2);
			const pg = c.createGain();
			pg.gain.setValueAtTime(a, t);
			pg.gain.exponentialRampToValueAtTime(a * 0.02, t + dec / r);
			x.connect(pg).connect(lp);
			nodes.push(x, pg);
			last = x;
		}
		const hs = c.createBufferSource();
		hs.buffer = kit.noise;
		const hl = c.createBiquadFilter();
		hl.type = 'lowpass';
		hl.frequency.value = 320;
		const hg = c.createGain();
		hg.gain.setValueAtTime(0.0001, t);
		hg.gain.linearRampToValueAtTime(vel * 0.42, t + 0.004);
		hg.gain.exponentialRampToValueAtTime(0.0001, t + 0.05);
		hs.connect(hl).connect(hg).connect(d);
		hs.start(t, Math.random());
		hs.stop(t + 0.06);
		end(hs, hl, hg);
		if (last) end(last, ...nodes);
	}
	/** electric piano (Rhodes-ish FM) */
	function ep(d: AudioNode, t: number, midi: number, vel: number) {
		const f = mtof(midi);
		const dec = clamp(5 - (midi - 48) * 0.05, 1.6, 5.5);
		const stop = t + dec + 0.05;
		const car = o('sine', f, t, stop);
		const mod = o('sine', f, t, stop);
		const mg = c.createGain();
		mg.gain.setValueAtTime(f * 1.6 * (0.5 + vel * 6), t);
		mg.gain.exponentialRampToValueAtTime(f * 0.08, t + 1.2);
		mod.connect(mg).connect(car.frequency);
		const g = c.createGain();
		g.gain.setValueAtTime(0, t);
		g.gain.linearRampToValueAtTime(vel, t + 0.004);
		g.gain.exponentialRampToValueAtTime(0.0001, t + dec);
		car.connect(g).connect(d);
		const tine = o('sine', f * 7.1, t, t + 0.2);
		const tg = c.createGain();
		tg.gain.setValueAtTime(vel * 0.12, t);
		tg.gain.exponentialRampToValueAtTime(0.0001, t + 0.15);
		tine.connect(tg).connect(d);
		end(tine, tg);
		end(car, mod, mg, g);
	}
	/** glassy FM bell */
	function glass(d: AudioNode, t: number, midi: number, vel: number, dec = 4.5) {
		const f = mtof(midi);
		const stop = t + dec + 0.05;
		const car = o('sine', f, t, stop, 3);
		const mod = o('sine', f * 3.5, t, stop);
		const mg = c.createGain();
		mg.gain.setValueAtTime(f * 1.4, t);
		mg.gain.exponentialRampToValueAtTime(f * 0.05, t + dec * 0.6);
		mod.connect(mg).connect(car.frequency);
		const g = c.createGain();
		g.gain.setValueAtTime(0, t);
		g.gain.linearRampToValueAtTime(vel, t + 0.003);
		g.gain.exponentialRampToValueAtTime(0.0001, t + dec);
		car.connect(g).connect(d);
		end(car, mod, mg, g);
	}
	/** slow, warm pad: detuned triangles + a quiet saw through a low-pass */
	function pad(d: AudioNode, t: number, midi: number, dur: number, vel: number, bright = 1, att = 3, rel = 4) {
		const f = mtof(midi);
		const stop = t + dur + rel + 0.1;
		const lp = c.createBiquadFilter();
		lp.type = 'lowpass';
		lp.frequency.value = clamp(f * 3 * bright + 300, 200, 4000);
		lp.Q.value = 0.6;
		const g = c.createGain();
		g.gain.setValueAtTime(0, t);
		g.gain.linearRampToValueAtTime(vel, t + att);
		g.gain.setValueAtTime(vel, t + Math.max(att, dur));
		g.gain.linearRampToValueAtTime(0, t + dur + rel);
		lp.connect(g).connect(d);
		const xs = [o('triangle', f, t, stop, -6), o('triangle', f, t, stop, 7), o('sawtooth', f / 2, t, stop, 2)];
		const sg = c.createGain();
		sg.gain.value = 0.25;
		xs[0].connect(lp);
		xs[1].connect(lp);
		xs[2].connect(sg).connect(lp);
		end(xs[0], xs[1], xs[2], sg, lp, g);
	}
	/** Qud drone: beating saws (root, root, optional fifth) through a filter that breathes */
	function drone(d: AudioNode, t: number, midi: number, dur: number, vel: number, lo = 260, hi = 900, fifthGain = 0.4) {
		const f = mtof(midi);
		const stop = t + dur + 0.1;
		const lp = c.createBiquadFilter();
		lp.type = 'lowpass';
		lp.frequency.value = (lo + hi) / 2;
		lp.Q.value = 1.25;
		const lfo = o('sine', 0.025 + (midi % 5) * 0.006, t, stop);
		const lg = c.createGain();
		lg.gain.value = (hi - lo) / 2;
		lfo.connect(lg).connect(lp.frequency);
		const g = c.createGain();
		g.gain.setValueAtTime(0, t);
		g.gain.linearRampToValueAtTime(vel, t + dur * 0.35);
		g.gain.setValueAtTime(vel, t + dur * 0.65);
		g.gain.linearRampToValueAtTime(0, t + dur);
		lp.connect(g).connect(d);
		const xs = [o('sawtooth', f, t, stop, -4), o('sawtooth', f, t, stop, 5), o('sawtooth', f * 1.5, t, stop, 1), o('sine', f / 2, t, stop)];
		const fg = c.createGain();
		fg.gain.value = fifthGain;
		xs[0].connect(lp);
		xs[1].connect(lp);
		xs[2].connect(fg).connect(lp);
		xs[3].connect(lp);
		end(xs[0], xs[1], xs[2], xs[3], fg, lfo, lg, lp, g);
	}
	/** Qud lead: square+saw, resonant low-pass, glide from the previous note and a late vibrato */
	function lead(d: AudioNode, t: number, midi: number, dur: number, vel: number, from?: number) {
		const f = mtof(midi);
		const stop = t + dur + 0.6;
		const xs = [o('square', f, t, stop, -5), o('sawtooth', f, t, stop, 6)];
		if (from !== undefined) for (const x of xs) { x.frequency.setValueAtTime(mtof(from), t); x.frequency.exponentialRampToValueAtTime(f, t + 0.09); }
		const vib = o('sine', 5.2, t, stop);
		const vg = c.createGain();
		vg.gain.setValueAtTime(0, t);
		vg.gain.linearRampToValueAtTime(14, t + Math.min(0.6, dur));
		vib.connect(vg);
		for (const x of xs) vg.connect(x.detune);
		const lp = c.createBiquadFilter();
		lp.type = 'lowpass';
		lp.Q.value = 3;
		lp.frequency.setValueAtTime(400, t);
		lp.frequency.linearRampToValueAtTime(1500 + vel * 6000, t + 0.08);
		lp.frequency.exponentialRampToValueAtTime(700, t + dur + 0.4);
		const g = c.createGain();
		g.gain.setValueAtTime(0, t);
		g.gain.linearRampToValueAtTime(vel * 0.5, t + 0.05);
		g.gain.setValueAtTime(vel * 0.45, t + dur);
		g.gain.linearRampToValueAtTime(0, t + dur + 0.5);
		xs[0].connect(lp);
		xs[1].connect(lp);
		lp.connect(g).connect(d);
		end(xs[0], xs[1], vib, vg, lp, g);
	}
	/** analog sequencer blip */
	function blip(d: AudioNode, t: number, midi: number, vel: number, cut: number) {
		const f = mtof(midi);
		const x = o('square', f, t, t + 0.4);
		const lp = c.createBiquadFilter();
		lp.type = 'lowpass';
		lp.Q.value = 4.2;
		lp.frequency.setValueAtTime(cut, t);
		lp.frequency.exponentialRampToValueAtTime(Math.max(180, f), t + 0.2);
		const g = c.createGain();
		g.gain.setValueAtTime(0.0001, t);
		g.gain.linearRampToValueAtTime(vel, t + 0.004);
		g.gain.exponentialRampToValueAtTime(0.0001, t + 0.32);
		x.connect(lp).connect(g).connect(d);
		end(x, lp, g);
	}
	function shaker(d: AudioNode, t: number, vel: number) {
		const s = c.createBufferSource();
		s.buffer = kit.noise;
		const hp = c.createBiquadFilter();
		hp.type = 'highpass';
		hp.frequency.value = 6500;
		const g = c.createGain();
		g.gain.setValueAtTime(0, t);
		g.gain.linearRampToValueAtTime(vel, t + 0.012);
		g.gain.exponentialRampToValueAtTime(0.0001, t + 0.07);
		s.connect(hp).connect(g).connect(d);
		s.start(t, Math.random());
		s.stop(t + 0.08);
		end(s, hp, g);
	}
	/** hand drum: dum (low) / tek (rim) */
	function drum(d: AudioNode, t: number, low: boolean, vel: number) {
		if (low) {
			const x = o('sine', 120, t, t + 0.35);
			x.frequency.setValueAtTime(130, t);
			x.frequency.exponentialRampToValueAtTime(62, t + 0.25);
			const g = c.createGain();
			g.gain.setValueAtTime(0.0001, t);
			g.gain.linearRampToValueAtTime(vel, t + 0.004);
			g.gain.exponentialRampToValueAtTime(0.0001, t + 0.3);
			x.connect(g).connect(d);
			end(x, g);
		} else {
			const s = c.createBufferSource();
			s.buffer = kit.noise;
			const bp = c.createBiquadFilter();
			bp.type = 'bandpass';
			bp.frequency.value = 2600;
			bp.Q.value = 2.5;
			const g = c.createGain();
			g.gain.setValueAtTime(0.0001, t);
			g.gain.linearRampToValueAtTime(vel, t + 0.003);
			g.gain.exponentialRampToValueAtTime(0.0001, t + 0.06);
			s.connect(bp).connect(g).connect(d);
			s.start(t, Math.random());
			s.stop(t + 0.07);
			end(s, bp, g);
		}
	}
	const inst = (name: string) => INST[name] ?? INST.flute;

	function newBus(t: number, fadeIn: number, level = 1): GainNode {
		const b = c.createGain();
		b.gain.setValueAtTime(0.0001, t);
		b.gain.linearRampToValueAtTime(level, t + fadeIn);
		b.connect(out);
		b.connect(hallIn);
		return b;
	}
	function retire(b: GainNode, t: number, fade: number) {
		b.gain.cancelScheduledValues(t);
		b.gain.setValueAtTime(Math.max(0.0001, b.gain.value), t);
		b.gain.linearRampToValueAtTime(0, t + fade);
		dying.add(b);
		setTimeout(() => { dying.delete(b); try { b.disconnect(); } catch { /* gone */ } }, (fade + 9) * 1000);
	}

	// ---------- shared composition helpers ----------
	interface Ctx { r: () => number; scale: number[]; base: number; m: QudMusic; cultural: boolean; prof: Profile | undefined }
	function resolveProfile(m: QudMusic, cultural: boolean): Profile | undefined {
		if (!cultural) return undefined;
		const base = m.sphere ? PROFILES[m.sphere] : undefined;
		return m.mstyle ? profileFor(m.mstyle, base) : base;
	}
	function deg(x: Ctx, i: number): number {
		const n = x.scale.length;
		return x.base + x.scale[((i % n) + n) % n] + 12 * Math.floor(i / n);
	}
	/** after a throttled tab wakes, skip missed events instead of firing them all at once */
	const fresh = (next: number, until: number) => (next < until - AHEAD - 0.5 ? until - AHEAD + 0.05 : next);
	function wxSoft(m: QudMusic): number {
		const w = m.wx ?? '';
		return /storm|blizzard|tempest/.test(w) ? 0.52 : /rain|snow|fog|smog|sand|ash/.test(w) ? 0.75 : 1;
	}

	// ---------- drift: Eno + C418 ----------
	const DRIFT_SCALES = new Set(['lydian', 'ionian', 'majPent', 'mixolydian', 'dorian', 'aeolian', 'minPent', 'yo', 'hirajoshi', 'insen', 'shang', 'ryukyu', 'tizita', 'hexa', 'pentachord']);
	const DRIFT_TIMBRES = new Set(['harp', 'koto', 'kora', 'kalimba', 'marimba', 'gamelan', 'musicbox', 'vibes', 'lute', 'guitar', 'bell']);
	function driftScene(key: string, m: QudMusic, cultural: boolean, t0: number): Scene {
		const r = rngOf(hash(key));
		const night = !m.day || m.mood === 'dead';
		const prof = resolveProfile(m, cultural);
		const profScales = prof?.scales.filter((s) => DRIFT_SCALES.has(s) && SCALES[s]) ?? [];
		let scaleName: string;
		if (m.mood === 'dead') scaleName = 'aeolian';
		else if (profScales.length && r() < (m.mstyle?.scaleWord ? 0.72 : 0.45)) scaleName = m.mstyle && r() < 0.7 ? profScales[0] : pick(r, profScales);
		else scaleName = night ? pick(r, ['dorian', 'aeolian', 'majPent', 'lydian']) : pick(r, ['lydian', 'ionian', 'majPent', 'mixolydian', 'dorian']);
		const x: Ctx = { r, scale: SCALES[scaleName] ?? SCALES.lydian, base: 50 + Math.floor(r() * 8), m, cultural, prof };
		const bus = newBus(t0, 6.5, LEVEL.drift);
		const cultVoice = prof ? [...prof.plucks, ...prof.leads].find((v) => DRIFT_TIMBRES.has(v) && INST[v]) : undefined;
		const voiceKinds = m.mood === 'sea'
			? (cultVoice ? ['glass', 'ep', cultVoice] : ['glass', 'ep', 'pad'])
			: m.mood === 'sky'
				? (cultVoice ? ['glass', cultVoice, 'ep'] : ['glass', 'glass', 'ep'])
				: (cultVoice ? ['ep', 'piano', cultVoice, 'glass'] : ['ep', 'piano', 'glass', 'ep']);
		const pace = clamp(1 / (prof?.tempo ?? 1), 0.85, 1.2);
		// Eno loops: 3–4 sparse incommensurate periods to avoid dense overlapping clusters
		const periods = [19.7, 23.3, 29.1, 34.7, 41.3, 47.9].sort(() => r() - 0.5).slice(0, 3 + (r() < 0.45 ? 1 : 0));
		const chordTones = [0, 2, 4, 6, 8, 9, 11];
		const loops = periods.map((p, i) => {
			const nn = 1 + (r() < 0.45 ? 1 : 0);
			const notes: [number, number][] = [];
			let at = 0;
			for (let k = 0; k < nn; k++) {
				notes.push([at, deg(x, pick(r, chordTones) + (i % 2 ? 7 : 0))]);
				at += 1.4 + r() * 2.8;
			}
			return { p: p * (night ? 1.28 : 1) * pace, notes, kind: voiceKinds[i % voiceKinds.length], vel: 0.028 + r() * 0.016, next: t0 + 3 + r() * p };
		});
		let phraseAt = t0 + 10 + r() * 22;
		let padAt = t0;
		let lastClusterAt = -10;
		// Shaped Eno/C418 breathing room: longer silences between passages
		let restFrom = t0 + 65 + r() * 85;
		let restTo = restFrom + 35 + r() * 40;
		const play = (kind: string, t: number, midi: number, vel: number) => {
			if (kind === 'piano') piano(bus, t, midi, vel);
			else if (kind === 'glass') glass(bus, t, midi + 12, vel * 0.6);
			else if (kind === 'pad') pad(bus, t, midi, 4.5, vel * 0.72, 0.8, 1.8, 3.5);
			else if (kind === 'ep') ep(bus, t, midi, vel);
			else inst(kind)(kit, bus, t, midi, 2.2, vel * 0.72);
		};
		return {
			key, bus,
			run(until, mm) {
				const soft = wxSoft(mm) * (mm.danger > 0.5 ? 0.72 : 1);
				const resting = (t: number) => t >= restFrom && t < restTo;
				padAt = fresh(padAt, until);
				phraseAt = fresh(phraseAt, until);
				for (const L of loops) if (L.next < until - AHEAD - 0.5) L.next += Math.ceil((until - AHEAD - L.next) / L.p) * L.p;
				if (padAt < until) {
					// low pad, thinned during rest silences for genuine breathing room
					const restMul = resting(padAt) ? 0.42 : 1;
					if (!resting(padAt) || r() < 0.45) {
						const root = pick(r, [0, 0, 3, 5, -2]);
						pad(bus, padAt, deg(x, root) - 12, 26, 0.024 * soft * restMul, 0.7, 4, 5);
						if (!resting(padAt) && r() < 0.45) pad(bus, padAt + 2.5, deg(x, root + 4) - 12, 22, 0.014 * soft, 0.7, 4, 5);
					}
					padAt += (28 + r() * 14) * pace;
				}
				for (const L of loops) {
					while (L.next < until) {
						const crowded = Math.abs(L.next - lastClusterAt) < 1.8;
						if (!resting(L.next) && !crowded && r() < 0.78) {
							for (const [dt, midi] of L.notes) play(L.kind, L.next + dt, midi, L.vel * soft * (mm.wx === 'fog' ? 0.8 : 1));
							lastClusterAt = L.next;
						}
						L.next += L.p;
					}
				}
				if (phraseAt < until) {
					// a C418-style piano thought: a few unhurried notes, rubato, a long last note
					if (!resting(phraseAt)) {
						let t = phraseAt;
						let i = 7 + Math.floor(r() * 5);
						const n = 3 + Math.floor(r() * 4);
						if (r() < 0.55) { piano(bus, t, deg(x, 0) - 12, 0.038 * soft); piano(bus, t + 0.03, deg(x, 4) - 12, 0.026 * soft); }
						for (let k = 0; k < n; k++) {
							piano(bus, t, deg(x, i), (0.048 + r() * 0.02) * soft);
							if (r() < 0.16) piano(bus, t, deg(x, i - 2), 0.024 * soft);
							i += pick(r, [-2, -1, -1, 1, 1, 2, 0, -3]);
							i = clamp(i, 4, 15);
							t += (k === n - 2 ? 2.0 : 0.75) + r() * 1.25;
						}
						lastClusterAt = phraseAt;
					}
					phraseAt += (30 + r() * 50) * pace;
				}
				if (until > restTo) { restFrom = restTo + 75 + r() * 110; restTo = restFrom + 35 + r() * 45; }
			},
		};
	}

	// ---------- pastoral: Stardew ----------
	const PROGS = [[0, 4, 5, 3], [0, 3, 0, 4], [0, 5, 3, 4], [3, 4, 2, 5], [0, 3, 4, 3], [5, 3, 0, 4]];
	const RHYTHMS_4 = [
		[[0, 2], [2, 1], [3, 1], [4, 4]],
		[[0, 1], [1, 1], [2, 2], [4, 2], [6, 2]],
		[[0, 3], [3, 1], [4, 2], [6, 2]],
		[[0, 2], [2, 2], [4, 1], [5, 1], [6, 2]],
		[[1, 1], [2, 1], [3, 1], [4, 4]],
	]; // [eighth offset, length in eighths] per 8-eighth bar
	const RHYTHMS_6 = [
		[[0, 2], [2, 1], [3, 3]],
		[[0, 3], [3, 2], [5, 1]],
		[[0, 1], [1, 2], [3, 1], [4, 2]],
		[[0, 2], [2, 2], [4, 2]],
	]; // [eighth offset, length in eighths] per 6-eighth (waltz / 6/8) bar
	const PASTORAL_SCALES = new Set(['ionian', 'mixolydian', 'majPent', 'lydian', 'dorian', 'yo', 'hexa', 'pentachord', 'shang', 'ryukyu', 'tizita', 'bati']);
	const NIGHT_PLUCKS = new Set(['harp', 'koto', 'kora', 'kalimba', 'lute']);
	function pastoralScene(key: string, m: QudMusic, cultural: boolean, t0: number): Scene {
		const r = rngOf(hash(key));
		const season = (m.season ?? 0) | 0;
		const night = !m.day;
		const prof = resolveProfile(m, cultural);
		const profScales = prof?.scales.filter((s) => PASTORAL_SCALES.has(s) && SCALES[s]) ?? [];
		const scaleName = profScales.length && r() < (m.mstyle?.scaleWord ? 0.78 : 0.55)
			? (m.mstyle && r() < 0.7 ? profScales[0] : pick(r, profScales))
			: (season === 2 && r() < 0.6 ? 'dorian' : pick(r, ['ionian', 'ionian', 'mixolydian', 'majPent', 'lydian']));
		const x: Ctx = { r, scale: SCALES[scaleName] ?? SCALES.ionian, base: 53 + Math.floor(r() * 7), m, cultural, prof };
		const bus = newBus(t0, 3.5, LEVEL.pastoral);
		// the band, by season (spring flute & harp, summer whistle & guitar, autumn fiddle & guitar, winter music box & piano)
		const band = [
			{ lead: 'flute', pluck: 'harp' },
			{ lead: 'whistle', pluck: 'guitar' },
			{ lead: 'violin', pluck: 'guitar' },
			{ lead: 'musicbox', pluck: 'piano' },
		][season];
		let leadName = night ? 'piano' : band.lead;
		let pluckName = night ? 'piano' : band.pluck;
		if (prof) {
			const cLead = prof.leads.find((v) => INST[v]);
			const cPluck = prof.plucks.find((v) => INST[v]);
			if (!night) {
				if (cLead) leadName = cLead;
				if (cPluck) pluckName = cPluck;
			} else if (cPluck && NIGHT_PLUCKS.has(cPluck) && r() < 0.45) {
				pluckName = cPluck;
			}
		}
		const triple = !!prof && (prof.feels[0] === 'waltz' || prof.feels[0] === 'six8');
		const steps = triple ? 6 : 8;
		const rhythms = triple ? RHYTHMS_6 : RHYTHMS_4;
		const tempoMul = clamp(prof?.tempo ?? 1, 0.85, 1.15);
		const bpm = clamp(((night ? 70 : 86) + r() * 12 - (season === 3 ? 8 : 0) + (m.feast ? 12 : 0)) * tempoMul, 58, 112);
		const e8 = 60 / bpm / 2;
		const prog = pick(r, PROGS);
		const progB = pick(r, PROGS);
		// a two-bar motif, re-used so the tune is memorable: A A B A
		const motif = (): [number, number, number][] => {
			const out: [number, number, number][] = [];
			for (let bar = 0; bar < 2; bar++) {
				const rh = pick(r, rhythms);
				let i = pick(r, [2, 4, 7, 4]);
				for (const [o8, l] of rh) { out.push([bar * steps + o8, l, i]); i += pick(r, [-1, 1, 1, -2, 2, 0, -1]); }
			}
			return out;
		};
		const A = motif();
		const B = motif();
		let bar = 0;
		let next = t0 + 0.2;
		const form = [0, 0, 1, 0];
		const playLead = (t: number, midi: number, dur: number, vel: number) => {
			if (leadName === 'piano') piano(bus, t, midi, vel * 0.95);
			else inst(leadName)(kit, bus, t, midi, dur, vel);
		};
		const playPluck = (t: number, midi: number, dur: number, vel: number) => {
			if (pluckName === 'piano') piano(bus, t, midi, vel * 0.72);
			else inst(pluckName)(kit, bus, t, midi, dur, vel);
		};
		return {
			key, bus,
			run(until, mm) {
				const soft = wxSoft(mm);
				const raining = /rain|storm|snow|blizzard/.test(mm.wx ?? '');
				next = fresh(next, until);
				while (next < until) {
					const sec = form[Math.floor(bar / 8) % 4];
					const chord = (sec === 1 ? progB : prog)[Math.floor(bar / 2) % 4];
					const root = deg(x, chord) - 12;
					const third = deg(x, chord + 2);
					const fifth = deg(x, chord + 4) - 12;
					if (bar % 8 === 0) pad(bus, next, root, steps * e8 * 3.5, 0.015 * soft, 0.75, 2.2, 3);
					// fingerpicking: crisp accompaniment sitting cleanly beneath the melody
					const pat = triple
						? [root, fifth, third, root + 12, fifth, third]
						: [root, fifth, third, fifth, root + 12, fifth, third, fifth];
					for (let k = 0; k < steps; k++) {
						const t = next + k * e8 + (k % 2 ? e8 * 0.04 : 0);
						if (night && k % 2) continue;
						playPluck(t, pat[k], e8 * 1.35, (k === 0 ? 0.052 : 0.032) * soft);
						if (!raining && !night && k % 2 === 1 && mm.day) shaker(bus, t, (k === 3 || k === steps - 1 ? 0.012 : 0.008) * soft);
					}
					// melody: rests through the intro and every 8th bar for breath
					const intro = bar < 4;
					if (!intro && bar % 8 !== 7) {
						const mot = sec === 1 ? B : A;
						const half = bar % 2;
						for (const [o8, l, i] of mot) {
							if (Math.floor(o8 / steps) !== half) continue;
							const t = next + (o8 % steps) * e8;
							playLead(t, deg(x, i + chord % 2), l * e8 * 0.92, 0.068 * soft);
						}
					}
					next += steps * e8;
					bar++;
					if (bar % 32 === 0) next += e8 * 6; // a breath between rounds
				}
			},
		};
	}

	// ---------- desert: Caves of Qud ----------
	const DESERT_SCALES = ['hijaz', 'phrygDom', 'bayati', 'phrygian', 'dorian', 'harmonicMinor'];
	function desertScene(key: string, m: QudMusic, cultural: boolean, t0: number): Scene {
		const r = rngOf(hash(key));
		const under = UNDER.has(m.mood);
		const prof = resolveProfile(m, cultural);
		const near = prof?.scales.filter((s) => DESERT_SCALES.includes(s) || ['uncanny', 'wholetone', 'octatonic', 'bhairav', 'todi', 'homayun', 'rast', 'marwa', 'hirajoshi', 'insen', 'pelog'].includes(s)) ?? [];
		const genreScales: Record<string, string[]> = { cyberpunk: ['aeolian', 'phrygian'], biopunk: ['wholetone', 'phrygDom'], atompunk: ['lydian', 'wholetone'], twilight: ['harmonicMinor', 'aeolian'], wasteland: ['phrygDom', 'minPent'] };
		const scaleName = near.length && r() < (m.mstyle?.scaleWord ? 0.85 : 0.7)
			? (m.mstyle && r() < 0.75 ? near[0] : pick(r, near))
			: pick(r, genreScales[m.genre] ?? DESERT_SCALES);
		const x: Ctx = { r, scale: SCALES[scaleName] ?? SCALES.hijaz, base: (under ? 40 : 45) + Math.floor(r() * 6), m, cultural, prof };
		const bus = newBus(t0, 7, LEVEL.desert);
		const leadPool = prof?.leads.filter((v) => INST[v]) ?? [];
		const leadVoice = leadPool.length && r() < (m.mstyle ? 0.75 : 0.55)
			? (m.mstyle && r() < 0.7 ? leadPool[0] : pick(r, leadPool))
			: 'synth';
		const fifthGain = prof?.drone === 'fifth' ? 0.45 : prof?.drone === 'none' ? 0.22 : 0.32;
		const tempoMul = clamp(prof?.tempo ?? 1, 0.85, 1.15);
		const droneDur = 26;
		let droneAt = t0;
		let phraseAt = t0 + 7 + r() * 12;
		let phraseEnd = t0;
		let bellAt = t0 + 5 + r() * 10;
		// the analog sequence: an odd-length pattern so it rolls against the bar
		const arpLen = pick(r, [5, 6, 7, 9]);
		const arp = Array.from({ length: arpLen }, () => pick(r, [0, 2, 4, 7, 4, 1, 9]));
		const arpBpm = clamp((92 + r() * 18) * tempoMul, 74, 120);
		const s16 = 60 / arpBpm / 4;
		let arpOn = !under || m.mood === 'tower' ? r() < 0.55 : false;
		let arpNext = t0 + 4;
		let arpI = 0;
		let arpSwitch = t0 + 36 + r() * 55;
		const drumOk = !under && m.day && m.mood !== 'sea';
		const maqsum = [1, 2, 0, 2, 1, 0, 2, 0]; // D T . T D . T .
		let drumNext = t0 + 12;
		let drumI = 0;
		let drumOn = drumOk && r() < 0.45;
		let drumSwitch = t0 + 30 + r() * 50;
		let lastLead: number | undefined;
		return {
			key, bus,
			run(until, mm) {
				const soft = wxSoft(mm);
				const tense = clamp(mm.danger, 0, 1);
				droneAt = fresh(droneAt, until);
				phraseAt = fresh(phraseAt, until);
				bellAt = fresh(bellAt, until);
				arpNext = fresh(arpNext, until);
				drumNext = fresh(drumNext, until);
				if (droneAt < until) {
					// crossfade overlap matched to the 35% attack / 35% release ramp for constant energy
					drone(bus, droneAt, x.base - 12, droneDur, (under ? 0.038 : 0.028) * soft, under ? 180 : 250, under ? 560 : 940 + 550 * tense, fifthGain);
					droneAt += droneDur * 0.7;
				}
				if (phraseAt < until) {
					// an ornamented modal phrase: grace notes, slides, a long held tone
					let t = phraseAt;
					let i = 7 + pick(r, [0, 1, 2, 4]);
					const n = 3 + Math.floor(r() * 4);
					for (let k = 0; k < n; k++) {
						const midi = deg(x, i);
						const len = k === n - 1 ? 2.6 + r() * 2 : 0.4 + r() * 0.7;
						if (leadVoice === 'synth') {
							if (r() < 0.28) lead(bus, t - 0.07, deg(x, i + 1), 0.06, 0.038 * soft);
							lead(bus, t, midi, len, 0.058 * soft, r() < 0.4 ? lastLead : undefined);
						} else inst(leadVoice)(kit, bus, t, midi, len, 0.058 * soft);
						lastLead = midi;
						t += len + (r() < 0.25 ? 0.45 : 0.08);
						i += pick(r, [-1, -1, 1, -2, 1, 0, 2]);
						i = clamp(i, 3, 12);
					}
					phraseEnd = t + 0.8;
					phraseAt = t + (under ? 16 : 10) + r() * (under ? 32 : 24);
				}
				if (bellAt < until) {
					if (bellAt > phraseEnd) glass(bus, bellAt, deg(x, pick(r, [7, 9, 11, 14])) + 12, 0.024 * soft, 5.5);
					bellAt += (under ? 9 : 13) + r() * 15;
				}
				if (until > arpSwitch) { arpOn = (!under || mm.mood === 'tower') && !arpOn && r() < 0.65; arpSwitch += 48 + r() * 68; }
				while (arpNext < until) {
					if (arpOn) {
						const duck = arpNext < phraseEnd ? 0.55 : 1;
						const cut = 560 + 1600 * (0.5 + 0.5 * Math.sin(arpNext * 0.11)) + 1400 * tense;
						blip(bus, arpNext, deg(x, arp[arpI % arpLen]), (arpI % arpLen === 0 ? 0.026 : 0.017) * soft * duck, cut);
					}
					arpI++;
					arpNext += s16 * 2;
				}
				if (until > drumSwitch) { drumOn = drumOk && !drumOn && r() < 0.55; drumSwitch += 35 + r() * 60; }
				while (drumNext < until) {
					const hit = maqsum[drumI % 8];
					if (drumOn && hit) drum(bus, drumNext, hit === 1, (hit === 1 ? 0.044 : 0.022) * soft);
					drumI++;
					drumNext += s16 * 2;
				}
			},
		};
	}

	return {
		get playing() { return scene !== null; },
		tick(m, gs, cultural, now, instant) {
			const key = gardenKey(m, gs, cultural);
			if (!scene || scene.key !== key) {
				const at = now + (instant ? 0.02 : 0.05);
				if (scene) retire(scene.bus, at, instant ? 0.05 : 5);
				const gap = !instant && scene ? 1.5 : 0;
				scene = gs === 'pastoral' ? pastoralScene(key, m, cultural, at + gap) : gs === 'desert' ? desertScene(key, m, cultural, at) : driftScene(key, m, cultural, at);
			}
			scene.run(now + AHEAD, m);
		},
		stop(at, fade = 3) {
			if (!scene) return;
			retire(scene.bus, at, fade);
			scene = null;
		},
		dispose() {
			scene = null;
			for (const b of dying) { try { b.disconnect(); } catch { /* gone */ } }
			try { hallIn.disconnect(); hallOut.disconnect(); } catch { /* gone */ }
		},
	};
}

/** stereo reverb impulse: decaying, darkening noise */
function impulse(c: AudioContext, secs: number, tau: number): AudioBuffer {
	const len = Math.floor(c.sampleRate * secs);
	const buf = c.createBuffer(2, len, c.sampleRate);
	for (let ch = 0; ch < 2; ch++) {
		const d = buf.getChannelData(ch);
		let lp = 0;
		for (let i = 0; i < len; i++) {
			const x = i / c.sampleRate;
			lp += (Math.random() * 2 - 1 - lp) * (0.45 - 0.37 * Math.min(1, x / secs));
			d[i] = lp * Math.exp(-x / (tau * 0.45)) * 1.4 * (x < 0.02 ? x / 0.02 : 1);
		}
	}
	return buf;
}
