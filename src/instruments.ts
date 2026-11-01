// Procedural WebAudio instruments (no samples). Every voice is a handful of nodes that are
// disposed by a sweep, and a voice limiter drops low-priority notes when the graph is busy.

export interface Kit {
	c: AudioContext;
	noise: AudioBuffer;
	ends: number[];
	trash: { t: number; n: AudioNode[] }[];
	ksMax: number;
}
export type Inst = (k: Kit, d: AudioNode, t: number, midi: number, dur: number, vel: number) => void;

const mtof = (m: number) => 440 * Math.pow(2, (m - 69) / 12);
const clamp = (v: number, a: number, b: number) => Math.min(b, Math.max(a, v));

export function createKit(c: AudioContext): Kit {
	const len = Math.floor(c.sampleRate * 2);
	const noise = c.createBuffer(1, len, c.sampleRate);
	const d = noise.getChannelData(0);
	for (let i = 0; i < len; i++) d[i] = Math.random() * 2 - 1;
	return { c, noise, ends: [], trash: [], ksMax: (c.sampleRate / 128) * 0.85 };
}

/** dispose finished voices; call every scheduler tick */
export function sweep(k: Kit): void {
	const now = k.c.currentTime;
	for (let i = k.trash.length - 1; i >= 0; i--) {
		if (k.trash[i].t < now) {
			for (const n of k.trash[i].n) { try { n.disconnect(); } catch { /* gone */ } }
			k.trash.splice(i, 1);
		}
	}
}
export function disposeKit(k: Kit): void {
	for (const x of k.trash) for (const n of x.n) { try { n.disconnect(); } catch { /* gone */ } }
	k.trash.length = 0;
	k.ends.length = 0;
}

function claim(k: Kit, t: number, dur: number, prio: number): boolean {
	const now = k.c.currentTime;
	for (let i = k.ends.length - 1; i >= 0; i--) if (k.ends[i] < now) k.ends.splice(i, 1);
	if (k.ends.length >= (prio >= 2 ? 46 : prio === 1 ? 32 : 20)) return false;
	k.ends.push(t + dur);
	return true;
}
function bin(k: Kit, t: number, ...n: AudioNode[]) { k.trash.push({ t, n }); }

function osc(c: AudioContext, type: OscillatorType, f: number, t: number, end: number): OscillatorNode {
	const o = c.createOscillator();
	o.type = type;
	o.frequency.value = f;
	o.start(t);
	o.stop(end + 0.03);
	return o;
}
function gainNode(c: AudioContext): GainNode { return c.createGain(); }
/** attack / sustain / release; returns the end time */
function adsr(g: AudioParam, t: number, a: number, pk: number, sus: number, dur: number, rel: number): number {
	const hold = t + Math.max(a + 0.01, dur);
	g.setValueAtTime(0, t);
	g.linearRampToValueAtTime(pk, t + a);
	g.linearRampToValueAtTime(pk * sus, hold);
	g.linearRampToValueAtTime(0, hold + rel);
	return hold + rel;
}
/** percussive exponential decay */
function pdec(g: AudioParam, t: number, pk: number, dec: number): number {
	g.setValueAtTime(0, t);
	g.linearRampToValueAtTime(Math.max(pk, 0.0006), t + 0.002);
	g.exponentialRampToValueAtTime(0.0005, t + 0.002 + dec);
	return t + dec + 0.01;
}
function vibrato(c: AudioContext, o: OscillatorNode, t: number, end: number, cents: number, hz: number, delay: number): AudioNode[] {
	if (cents <= 0) return [];
	const l = osc(c, 'sine', hz, t, end);
	const g = gainNode(c);
	g.gain.setValueAtTime(0, t);
	g.gain.linearRampToValueAtTime(cents, t + delay + 0.25);
	l.connect(g).connect(o.detune);
	return [l, g];
}
function noiseSrc(k: Kit, t: number, len: number): AudioBufferSourceNode {
	const s = k.c.createBufferSource();
	s.buffer = k.noise;
	s.start(t, Math.random() * 1.7);
	s.stop(t + len);
	return s;
}

// ---------- plucked strings: Karplus-Strong where the loop delay allows, filtered saw otherwise ----------
interface PluckP { type: OscillatorType; bright: number; dec: number; bend: number; q: number }
const PL: Record<string, PluckP> = {
	lute: { type: 'sawtooth', bright: 2.2, dec: 1, bend: 0, q: 0.7 },
	guitar: { type: 'sawtooth', bright: 3, dec: 1.2, bend: 0, q: 0.8 },
	koto: { type: 'triangle', bright: 5, dec: 1, bend: 0.4, q: 1.2 },
	kora: { type: 'triangle', bright: 3.5, dec: 1.4, bend: 0, q: 0.8 },
	sitar: { type: 'sawtooth', bright: 7, dec: 1.8, bend: 0.6, q: 5 },
	harp: { type: 'triangle', bright: 2.4, dec: 1.8, bend: 0, q: 0.6 },
	mandolin: { type: 'sawtooth', bright: 4, dec: 0.55, bend: 0, q: 1 },
	harpsi: { type: 'sawtooth', bright: 8, dec: 0.5, bend: 0, q: 1.4 },
};
function pluck(name: string): Inst {
	const p = PL[name];
	return (k, d, t, midi, dur, vel) => {
		const f = mtof(midi);
		const T = clamp(0.35 + p.dec * 0.55 + dur * 0.7, 0.3, 2.4);
		if (!claim(k, t, T, 1)) return;
		const c = k.c;
		if (f <= k.ksMax) {
			const per = 1 / f;
			const src = c.createBufferSource();
			src.buffer = k.noise;
			const bg = gainNode(c);
			bg.gain.setValueAtTime(vel * 1.6, t);
			bg.gain.setValueAtTime(0, t + per);
			const dl = c.createDelay(0.1);
			dl.delayTime.value = Math.max(0.0002, per - 1 / c.sampleRate);
			const lp = c.createBiquadFilter();
			lp.type = 'lowpass';
			lp.frequency.value = Math.min(11000, f * p.bright * 2 + 500);
			// Web Audio's low-pass Q is in dB: anything above 0 adds a resonant bump, which inside this
			// feedback loop pushed the loop gain past 1 and made low or repeated notes run away (×3000).
			// A negative Q keeps the filter below unity everywhere; the feedback is also capped.
			lp.Q.value = -6;
			const fb = gainNode(c);
			fb.gain.value = Math.min(0.996, Math.pow(0.001, per / T));
			const og = gainNode(c);
			og.gain.setValueAtTime(1, t);
			og.gain.setValueAtTime(1, t + T * 0.7);
			og.gain.linearRampToValueAtTime(0, t + T);
			src.connect(bg).connect(dl);
			dl.connect(lp);
			lp.connect(fb).connect(dl);
			lp.connect(og).connect(d);
			src.start(t, Math.random() * 1.7);
			src.stop(t + per + 0.01);
			bin(k, t + T + 0.1, src, bg, dl, lp, fb, og);
		} else {
			const o = osc(c, p.type, f, t, t + T);
			if (p.bend) {
				o.frequency.setValueAtTime(f * Math.pow(2, p.bend / 12), t);
				o.frequency.exponentialRampToValueAtTime(f, t + 0.08);
			}
			const lp = c.createBiquadFilter();
			lp.type = 'lowpass';
			lp.Q.value = p.q;
			lp.frequency.setValueAtTime(Math.min(f * p.bright * 3 + 800, 14000), t);
			lp.frequency.exponentialRampToValueAtTime(Math.max(f * 1.3, 300), t + T * 0.5);
			const g = gainNode(c);
			pdec(g.gain, t, vel * 1.15, T);
			o.connect(lp).connect(g).connect(d);
			bin(k, t + T + 0.1, o, lp, g);
		}
	};
}

// ---------- bowed / reed ----------
interface BowP { type: OscillatorType; f1: number; f2: number; q: number; vib: number; vr: number; att: number; breath: number; gain: number }
const BW: Record<string, BowP> = {
	erhu: { type: 'sawtooth', f1: 650, f2: 2100, q: 3, vib: 24, vr: 5.6, att: 0.07, breath: 0, gain: 3 },
	violin: { type: 'sawtooth', f1: 450, f2: 1800, q: 2.5, vib: 18, vr: 5.8, att: 0.09, breath: 0, gain: 2.6 },
	duduk: { type: 'square', f1: 520, f2: 1250, q: 5, vib: 8, vr: 4.5, att: 0.12, breath: 0.25, gain: 3.2 },
	shawm: { type: 'sawtooth', f1: 1100, f2: 2600, q: 3, vib: 0, vr: 5, att: 0.03, breath: 0.08, gain: 3 },
	bagpipe: { type: 'sawtooth', f1: 900, f2: 2200, q: 2.5, vib: 0, vr: 5, att: 0.02, breath: 0, gain: 2.8 },
};
function bowed(name: string): Inst {
	const p = BW[name];
	return (k, d, t, midi, dur, vel) => {
		if (!claim(k, t, dur + 0.2, 1)) return;
		const c = k.c;
		const f = mtof(midi);
		const end = t + Math.max(dur, 0.15) + 0.2;
		const o = osc(c, p.type, f, t, end);
		const o2 = osc(c, p.type, f, t, end);
		o2.detune.value = 7;
		const mix = gainNode(c);
		const b1 = c.createBiquadFilter(); b1.type = 'bandpass'; b1.frequency.value = p.f1; b1.Q.value = p.q;
		const b2 = c.createBiquadFilter(); b2.type = 'bandpass'; b2.frequency.value = p.f2; b2.Q.value = p.q;
		const lp = c.createBiquadFilter(); lp.type = 'lowpass'; lp.frequency.value = Math.max(900, f * 3); lp.Q.value = 0.5;
		const lg = gainNode(c); lg.gain.value = 0.35;
		const g = gainNode(c);
		for (const n of [o, o2]) { n.connect(b1); n.connect(b2); n.connect(lp); }
		b1.connect(mix); b2.connect(mix); lp.connect(lg).connect(mix);
		mix.connect(g).connect(d);
		const vn = vibrato(c, o, t, end, p.vib, p.vr, 0.1);
		const nodes: AudioNode[] = [o, o2, mix, b1, b2, lp, lg, g, ...vn];
		if (p.breath > 0) {
			const ns = noiseSrc(k, t, end - t);
			const nb = c.createBiquadFilter(); nb.type = 'bandpass'; nb.frequency.value = Math.min(6000, f * 2.5); nb.Q.value = 1.5;
			const ng = gainNode(c); ng.gain.value = p.breath * 0.5;
			ns.connect(nb).connect(ng).connect(mix);
			nodes.push(ns, nb, ng);
		}
		const e = adsr(g.gain, t, p.att, vel * p.gain * 0.5, 0.85, dur, 0.1);
		bin(k, e + 0.1, ...nodes);
	};
}

// ---------- flutes ----------
interface FluteP { breath: number; harm: number; att: number; vib: number; vr: number; bend: number }
const FL: Record<string, FluteP> = {
	flute: { breath: 0.35, harm: 0.18, att: 0.06, vib: 14, vr: 5, bend: 0 },
	whistle: { breath: 0.2, harm: 0.05, att: 0.03, vib: 8, vr: 6, bend: 0 },
	panpipe: { breath: 0.7, harm: 0.04, att: 0.09, vib: 0, vr: 5, bend: 0 },
	shaku: { breath: 0.8, harm: 0.25, att: 0.1, vib: 20, vr: 4.6, bend: -0.6 },
};
function flute(name: string): Inst {
	const p = FL[name];
	return (k, d, t, midi, dur, vel) => {
		if (!claim(k, t, dur + 0.2, 1)) return;
		const c = k.c;
		const f = mtof(midi);
		const end = t + Math.max(dur, 0.15) + 0.15;
		const o = osc(c, 'sine', f, t, end);
		if (p.bend) {
			o.frequency.setValueAtTime(f * Math.pow(2, p.bend / 12), t);
			o.frequency.exponentialRampToValueAtTime(f, t + 0.12);
		}
		const o2 = osc(c, 'sine', f * 2, t, end);
		const hg = gainNode(c); hg.gain.value = p.harm;
		const g = gainNode(c);
		const ns = noiseSrc(k, t, end - t);
		const nb = c.createBiquadFilter(); nb.type = 'bandpass'; nb.frequency.value = clamp(f * 2, 500, 8000); nb.Q.value = 1.8;
		const ng = gainNode(c); ng.gain.value = p.breath * 0.55;
		o.connect(g); o2.connect(hg).connect(g);
		ns.connect(nb).connect(ng).connect(g);
		g.connect(d);
		const vn = vibrato(c, o, t, end, p.vib, p.vr, 0.2);
		const e = adsr(g.gain, t, p.att, vel * 1.15, 0.9, dur, 0.1);
		bin(k, e + 0.1, o, o2, hg, g, ns, nb, ng, ...vn);
	};
}

// ---------- mallets & bells ----------
type Part = [number, number, number]; // ratio, gain, decay multiplier
function mallet(parts: Part[], dec: number, shimmer = 0): Inst {
	return (k, d, t, midi, dur, vel) => {
		const f = mtof(midi);
		const T = dec * clamp(0.7 + dur * 0.5, 0.7, 1.5);
		if (!claim(k, t, T, 1)) return;
		const c = k.c;
		const nodes: AudioNode[] = [];
		const add = (fr: number, gn: number, dm: number) => {
			if (fr > 14000) return;
			const o = osc(c, 'sine', fr, t, t + T * dm);
			const g = gainNode(c);
			pdec(g.gain, t, vel * gn * 0.9, T * dm);
			o.connect(g).connect(d);
			nodes.push(o, g);
		};
		for (const [r, gn, dm] of parts) add(f * r, gn, dm);
		if (shimmer) add(f + shimmer, parts[0][1] * 0.7, parts[0][2]);
		bin(k, t + T + 0.1, ...nodes);
	};
}
function fmBell(ratio: number, index: number, dec: number): Inst {
	return (k, d, t, midi, dur, vel) => {
		const f = mtof(midi);
		const T = dec * clamp(0.7 + dur * 0.4, 0.7, 1.4);
		if (!claim(k, t, T, 1)) return;
		const c = k.c;
		const car = osc(c, 'sine', f, t, t + T);
		const mod = osc(c, 'sine', f * ratio, t, t + T);
		const mg = gainNode(c);
		mg.gain.setValueAtTime(f * index, t);
		mg.gain.exponentialRampToValueAtTime(f * 0.05, t + T * 0.6);
		mod.connect(mg).connect(car.frequency);
		const g = gainNode(c);
		pdec(g.gain, t, vel * 0.95, T);
		car.connect(g).connect(d);
		bin(k, t + T + 0.1, car, mod, mg, g);
	};
}

// ---------- sustained organs / pads / synths ----------
function sustained(parts: [number, number][], att: number, rel: number, gain: number, chiff = false): Inst {
	return (k, d, t, midi, dur, vel) => {
		if (!claim(k, t, dur + rel, 1)) return;
		const c = k.c;
		const f = mtof(midi);
		const end = t + Math.max(dur, 0.1) + rel + 0.05;
		const g = gainNode(c);
		const nodes: AudioNode[] = [g];
		for (const [r, gn] of parts) {
			if (f * r > 12000) continue;
			const o = osc(c, 'sine', f * r, t, end);
			const pg = gainNode(c);
			pg.gain.value = gn / parts.length;
			o.connect(pg).connect(g);
			nodes.push(o, pg);
		}
		if (chiff) {
			const ns = noiseSrc(k, t, 0.05);
			const nb = c.createBiquadFilter(); nb.type = 'bandpass'; nb.frequency.value = clamp(f * 3, 800, 6000); nb.Q.value = 2;
			const ng = gainNode(c);
			pdec(ng.gain, t, vel * 0.25, 0.05);
			ns.connect(nb).connect(ng).connect(d);
			nodes.push(ns, nb, ng);
		}
		g.connect(d);
		const e = adsr(g.gain, t, att, vel * gain, 0.9, dur, rel);
		bin(k, e + 0.1, ...nodes);
	};
}
function filtered(type: OscillatorType, detunes: number[], lpFrom: number, lpTo: number, q: number, att: number, sus: number, rel: number, gain: number, vibCents = 0, sweepT = 0.15): Inst {
	return (k, d, t, midi, dur, vel) => {
		if (!claim(k, t, dur + rel, 1)) return;
		const c = k.c;
		const f = mtof(midi);
		const end = t + Math.max(dur, 0.1) + rel + 0.05;
		const lp = c.createBiquadFilter();
		lp.type = 'lowpass';
		lp.Q.value = q;
		lp.frequency.setValueAtTime(clamp(f * lpFrom, 80, 14000), t);
		lp.frequency.exponentialRampToValueAtTime(clamp(f * lpTo, 80, 14000), t + sweepT);
		const g = gainNode(c);
		const nodes: AudioNode[] = [lp, g];
		detunes.forEach((dt, i) => {
			const o = osc(c, type, f, t, end);
			o.detune.value = dt;
			o.connect(lp);
			nodes.push(o);
			if (i === 0) nodes.push(...vibrato(c, o, t, end, vibCents, 5.5, 0.3));
		});
		lp.connect(g).connect(d);
		const e = adsr(g.gain, t, att, vel * gain, sus, dur, rel);
		bin(k, e + 0.1, ...nodes);
	};
}
const theremin: Inst = (k, d, t, midi, dur, vel) => {
	if (!claim(k, t, dur + 0.2, 1)) return;
	const c = k.c;
	const f = mtof(midi);
	const end = t + Math.max(dur, 0.2) + 0.2;
	const o = osc(c, 'sine', f, t, end);
	o.frequency.setValueAtTime(f * 0.94, t);
	o.frequency.exponentialRampToValueAtTime(f, t + 0.14);
	const o2 = osc(c, 'sine', f * 2, t, end);
	const hg = gainNode(c); hg.gain.value = 0.15;
	const g = gainNode(c);
	o.connect(g); o2.connect(hg).connect(g); g.connect(d);
	const vn = vibrato(c, o, t, end, 32, 6, 0.3);
	const e = adsr(g.gain, t, 0.12, vel * 1.3, 0.9, dur, 0.15);
	bin(k, e + 0.1, o, o2, hg, g, ...vn);
};
const choir: Inst = (k, d, t, midi, dur, vel) => {
	if (!claim(k, t, dur + 0.6, 1)) return;
	const c = k.c;
	const f = mtof(midi);
	const end = t + Math.max(dur, 0.3) + 0.6;
	const g = gainNode(c);
	const nodes: AudioNode[] = [g];
	for (const dt of [-6, 7]) {
		const o = osc(c, 'sawtooth', f, t, end);
		o.detune.value = dt;
		nodes.push(o, ...vibrato(c, o, t, end, 12, 5, 0.3));
		for (const [fr, q, gn] of [[800, 7, 1], [1150, 7, 0.6], [2900, 7, 0.3]]) {
			const b = c.createBiquadFilter(); b.type = 'bandpass'; b.frequency.value = fr; b.Q.value = q;
			const bg = gainNode(c); bg.gain.value = gn;
			o.connect(b).connect(bg).connect(g);
			nodes.push(b, bg);
		}
	}
	g.connect(d);
	const e = adsr(g.gain, t, 0.35, vel * 2.2, 0.9, dur, 0.5);
	bin(k, e + 0.1, ...nodes);
};
const wet: Inst = (k, d, t, midi, dur, vel) => {
	if (!claim(k, t, dur + 1, 1)) return;
	const c = k.c;
	const f = mtof(midi);
	const end = t + Math.max(dur, 0.5) + 1;
	const lp = c.createBiquadFilter(); lp.type = 'lowpass'; lp.Q.value = 3; lp.frequency.value = clamp(f * 3, 450, 1800);
	const g = gainNode(c);
	const nodes: AudioNode[] = [lp, g];
	const lfo = osc(c, 'sine', 0.3, t, end);
	const lg = gainNode(c); lg.gain.value = 14;
	lfo.connect(lg);
	nodes.push(lfo, lg);
	for (const dt of [-9, 0, 10]) {
		const o = osc(c, 'sawtooth', f, t, end);
		o.detune.value = dt;
		lg.connect(o.detune);
		o.connect(lp);
		nodes.push(o);
	}
	lp.connect(g).connect(d);
	const e = adsr(g.gain, t, 0.55, vel * 0.7, 0.9, dur, 1);
	bin(k, e + 0.1, ...nodes);
};
const harmonica: Inst = (k, d, t, midi, dur, vel) => {
	if (!claim(k, t, dur + 0.2, 1)) return;
	const c = k.c;
	const f = mtof(midi);
	const end = t + Math.max(dur, 0.15) + 0.15;
	const o = osc(c, 'square', f, t, end);
	const o2 = osc(c, 'sawtooth', f, t, end);
	o2.detune.value = 6;
	const bp = c.createBiquadFilter(); bp.type = 'bandpass'; bp.frequency.value = 1600; bp.Q.value = 0.9;
	const lp = c.createBiquadFilter(); lp.type = 'lowpass'; lp.frequency.value = 4200;
	const g = gainNode(c);
	o.connect(bp); o2.connect(bp); bp.connect(lp).connect(g).connect(d);
	const ns = noiseSrc(k, t, end - t);
	const nb = c.createBiquadFilter(); nb.type = 'bandpass'; nb.frequency.value = clamp(f * 3, 800, 6000); nb.Q.value = 1;
	const ng = gainNode(c); ng.gain.value = 0.06;
	ns.connect(nb).connect(ng).connect(g);
	const vn = vibrato(c, o, t, end, 14, 5.2, 0.25);
	const e = adsr(g.gain, t, 0.04, vel * 2.4, 0.85, dur, 0.1);
	bin(k, e + 0.1, o, o2, bp, lp, g, ns, nb, ng, ...vn);
};
const radio: Inst = (k, d, t, midi, dur, vel) => {
	if (!claim(k, t, dur + 0.2, 1)) return;
	const c = k.c;
	const f = mtof(midi);
	const end = t + Math.max(dur, 0.2) + 0.2;
	const o = osc(c, 'triangle', f, t, end);
	const o2 = osc(c, 'square', f, t, end);
	o2.detune.value = -9 + Math.random() * 18;
	const bp = c.createBiquadFilter(); bp.type = 'bandpass'; bp.frequency.value = 1100; bp.Q.value = 1.1;
	const g = gainNode(c);
	const sg = gainNode(c); sg.gain.value = 0.5;
	o.connect(bp); o2.connect(sg).connect(bp);
	bp.connect(g).connect(d);
	const wob = osc(c, 'sine', 0.5 + Math.random() * 0.6, t, end);
	const wg = gainNode(c); wg.gain.value = 22 + Math.random() * 25;
	wob.connect(wg); wg.connect(o.detune); wg.connect(o2.detune);
	const ns = noiseSrc(k, t, end - t);
	const hp = c.createBiquadFilter(); hp.type = 'highpass'; hp.frequency.value = 3000;
	const ng = gainNode(c); ng.gain.value = 0.04;
	ns.connect(hp).connect(ng).connect(g);
	const e = adsr(g.gain, t, 0.02, vel * 2.2, 0.8, dur, 0.12);
	bin(k, e + 0.1, o, o2, bp, g, sg, wob, wg, ns, hp, ng);
};
const softPad: Inst = (k, d, t, midi, dur, vel) => {
	if (!claim(k, t, dur + 0.5, 1)) return;
	const c = k.c;
	const f = mtof(midi);
	const end = t + Math.max(dur, 0.4) + 0.5;
	const g = gainNode(c);
	const nodes: AudioNode[] = [g];
	for (const [type, dt] of [['sine', -4], ['triangle', 5]] as [OscillatorType, number][]) {
		const o = osc(c, type, f, t, end);
		o.detune.value = dt;
		o.connect(g);
		nodes.push(o);
	}
	g.connect(d);
	const e = adsr(g.gain, t, 0.4, vel * 0.9, 0.9, dur, 0.5);
	bin(k, e + 0.1, ...nodes);
};

export const INST: Record<string, Inst> = {
	lute: pluck('lute'), guitar: pluck('guitar'), koto: pluck('koto'), kora: pluck('kora'), sitar: pluck('sitar'),
	harp: pluck('harp'), mandolin: pluck('mandolin'), harpsi: pluck('harpsi'),
	erhu: bowed('erhu'), violin: bowed('violin'), duduk: bowed('duduk'), shawm: bowed('shawm'), bagpipe: bowed('bagpipe'),
	flute: flute('flute'), whistle: flute('whistle'), panpipe: flute('panpipe'), shaku: flute('shaku'),
	bell: fmBell(3.5, 2.2, 2.6),
	gamelan: mallet([[1, 1, 1], [2.76, 0.45, 0.5], [5.4, 0.2, 0.25]], 1.7, 5),
	marimba: mallet([[1, 1, 1], [4, 0.35, 0.3], [9.9, 0.1, 0.12]], 0.55),
	kalimba: mallet([[1, 1, 1], [5.4, 0.3, 0.18], [9, 0.1, 0.1]], 1),
	musicbox: mallet([[1, 1, 1], [5.9, 0.25, 0.3], [8.2, 0.12, 0.2]], 1.3),
	vibes: mallet([[1, 1, 1], [4, 0.3, 0.3]], 2),
	calliope: sustained([[1, 1], [2, 0.8], [3, 0.6], [4, 0.5], [6, 0.3]], 0.02, 0.05, 1.5, true),
	organ: sustained([[0.5, 0.7], [1, 1], [2, 0.5], [3, 0.35], [4, 0.2]], 0.12, 0.35, 1.7),
	brass: filtered('sawtooth', [0, 5], 1.2, 4, 1.5, 0.05, 0.8, 0.1, 0.8, 0, 0.12),
	synthlead: filtered('sawtooth', [-9, 9], 7, 3, 5, 0.005, 0.6, 0.06, 0.8),
	synthbass: filtered('sawtooth', [0], 8, 1.5, 9, 0.005, 0.7, 0.05, 1.1, 0, 0.12),
	theremin, choir, wet, harmonica, radio, sine: softPad,
};

/** two slightly detuned oscillators + low-pass: tanpura / pipe drone / dungeon hum */
export function drone2(k: Kit, d: AudioNode, t: number, midi: number, dur: number, vel: number, type: OscillatorType = 'sawtooth', beatHz = 0.6): void {
	if (!claim(k, t, dur, 2)) return;
	const c = k.c;
	const f = mtof(midi);
	const end = t + dur + 0.1;
	const lp = c.createBiquadFilter(); lp.type = 'lowpass'; lp.frequency.value = clamp(f * 4, 150, 1200); lp.Q.value = 0.7;
	const g = gainNode(c);
	const o = osc(c, type, f, t, end);
	const o2 = osc(c, type, f + beatHz, t, end);
	o.connect(lp); o2.connect(lp); lp.connect(g).connect(d);
	g.gain.setValueAtTime(0, t);
	g.gain.linearRampToValueAtTime(vel, t + dur * 0.35);
	g.gain.setValueAtTime(vel, t + dur * 0.7);
	g.gain.linearRampToValueAtTime(0, t + dur);
	bin(k, end + 0.1, o, o2, lp, g);
}

// ---------- percussion ----------
interface PercSpec { f0: number; f1: number; dec: number; tone: number; nHz: number; nQ: number; nDec: number; noise: number; ring?: number; rel?: boolean }
const X: PercSpec = { f0: 0, f1: 0, dec: 0.05, tone: 0, nHz: 3000, nQ: 1, nDec: 0.05, noise: 0 };
const ps = (o: Partial<PercSpec>): PercSpec => ({ ...X, ...o });
/** roles: 0 low, 1 mid, 2 high ghost, 3 accent */
const KITS: Record<string, PercSpec[]> = {
	taiko: [ps({ f0: 110, f1: 52, dec: 0.5, tone: 1, noise: 0.15, nHz: 300, nQ: 0.7, nDec: 0.08 }), ps({ f0: 200, f1: 150, dec: 0.12, tone: 0.6, noise: 0.5, nHz: 1800, nQ: 1.5 }), ps({ f0: 320, f1: 260, dec: 0.07, tone: 0.35, noise: 0.4, nHz: 3000 }), ps({ f0: 140, f1: 60, dec: 0.6, tone: 1, noise: 0.5, nHz: 900, nQ: 0.8, nDec: 0.1 })],
	tabla: [ps({ f0: 90, f1: 140, dec: 0.38, tone: 1, noise: 0.08, nHz: 400 }), ps({ f0: 2, f1: 2, dec: 0.22, tone: 0.8, ring: 2.8, rel: true }), ps({ f0: 3, f1: 3, dec: 0.3, tone: 0.5, ring: 2, rel: true }), ps({ f0: 120, f1: 160, dec: 0.1, tone: 0.5, noise: 0.7, nHz: 2500, nQ: 1.2, nDec: 0.06 })],
	darbuka: [ps({ f0: 200, f1: 105, dec: 0.25, tone: 1, noise: 0.1, nHz: 500 }), ps({ f0: 420, f1: 380, dec: 0.08, tone: 0.5, noise: 0.5, nHz: 3200, nQ: 1.2 }), ps({ f0: 600, f1: 560, dec: 0.05, tone: 0.3, noise: 0.5, nHz: 4500 }), ps({ f0: 300, f1: 250, dec: 0.1, tone: 0.3, noise: 0.9, nHz: 3800, nQ: 0.9, nDec: 0.09 })],
	djembe: [ps({ f0: 130, f1: 70, dec: 0.4, tone: 1, noise: 0.12, nHz: 350 }), ps({ f0: 260, f1: 200, dec: 0.14, tone: 0.8, noise: 0.3, nHz: 1500 }), ps({ f0: 520, f1: 420, dec: 0.05, tone: 0.3, noise: 0.5, nHz: 3500 }), ps({ f0: 500, f1: 400, dec: 0.05, tone: 0.3, noise: 0.95, nHz: 3500, nQ: 1, nDec: 0.09 })],
	bodhran: [ps({ f0: 100, f1: 68, dec: 0.3, tone: 1, noise: 0.2, nHz: 400 }), ps({ f0: 160, f1: 120, dec: 0.14, tone: 0.8, noise: 0.3, nHz: 900 }), ps({ f0: 230, f1: 190, dec: 0.08, tone: 0.5, noise: 0.5, nHz: 2500 }), ps({ f0: 110, f1: 75, dec: 0.34, tone: 1, noise: 0.5, nHz: 800 })],
	frame: [ps({ f0: 120, f1: 80, dec: 0.22, tone: 1, noise: 0.25, nHz: 700 }), ps({ nHz: 5000, nQ: 1.2, nDec: 0.12, noise: 0.7 }), ps({ nHz: 6500, nQ: 1, nDec: 0.04, noise: 0.55 }), ps({ f0: 140, f1: 90, dec: 0.25, tone: 1, noise: 0.6, nHz: 4500, nDec: 0.14 })],
	gong: [ps({ f0: 0.5, f1: 0.5, dec: 1.7, tone: 1, ring: 2.4, rel: true, noise: 0.08, nHz: 300 }), ps({ f0: 300, f1: 300, dec: 0.5, tone: 0.8, ring: 2.76 }), ps({ f0: 900, f1: 900, dec: 0.25, tone: 0.5, ring: 2.4 }), ps({ f0: 0.5, f1: 0.5, dec: 2.2, tone: 1, ring: 1.6, rel: true, noise: 0.2, nHz: 500, nDec: 0.3 })],
	tincan: [ps({ f0: 140, f1: 100, dec: 0.12, tone: 0.7, noise: 0.6, nHz: 600, nQ: 0.6 }), ps({ noise: 0.9, nHz: 2400, nQ: 6, nDec: 0.12, f0: 1500, f1: 1500, dec: 0.05, tone: 0.1 }), ps({ noise: 0.8, nHz: 4200, nQ: 8, nDec: 0.05 }), ps({ noise: 0.9, nHz: 1800, nQ: 0.5, nDec: 0.3 })],
	electro: [ps({ f0: 150, f1: 45, dec: 0.16, tone: 1 }), ps({ f0: 220, f1: 180, dec: 0.06, tone: 0.5, noise: 0.8, nHz: 1900, nQ: 0.8, nDec: 0.12 }), ps({ noise: 0.6, nHz: 8000, nQ: 0.7, nDec: 0.04 }), ps({ noise: 0.9, nHz: 1500, nQ: 0.6, nDec: 0.16 })],
	brush: [ps({ f0: 90, f1: 50, dec: 0.2, tone: 0.8 }), ps({ noise: 0.7, nHz: 5000, nQ: 0.5, nDec: 0.22 }), ps({ noise: 0.5, nHz: 7000, nQ: 0.6, nDec: 0.06 }), ps({ noise: 0.9, nHz: 3000, nQ: 1, nDec: 0.1, f0: 300, f1: 250, dec: 0.05, tone: 0.3 })],
	timpani: [ps({ f0: 1, f1: 0.97, dec: 0.9, tone: 1, rel: true, noise: 0.08, nHz: 250 }), ps({ f0: 1.5, f1: 1.45, dec: 0.5, tone: 0.5, rel: true }), ps({ f0: 2, f1: 2, dec: 0.2, tone: 0.2, rel: true }), ps({ f0: 1, f1: 0.95, dec: 1.2, tone: 1, rel: true, noise: 0.25, nHz: 400, nDec: 0.1 })],
};

/** one drum hit. `root` (Hz) tunes the kits whose specs are relative (tabla, gong, timpani). */
export function hit(k: Kit, d: AudioNode, t: number, kit: string, role: 0 | 1 | 2 | 3, vel: number, root: number): void {
	const specs = KITS[kit] ?? KITS.frame;
	const s = specs[role];
	if (!claim(k, t, Math.max(s.dec, s.nDec) + 0.05, role === 0 || role === 3 ? 2 : 0)) return;
	const c = k.c;
	const nodes: AudioNode[] = [];
	if (s.tone > 0) {
		const f0 = s.rel ? root * s.f0 : s.f0;
		const f1 = s.rel ? root * s.f1 : s.f1;
		const o = osc(c, 'sine', f0, t, t + s.dec);
		if (f1 !== f0) { o.frequency.setValueAtTime(f0, t); o.frequency.exponentialRampToValueAtTime(Math.max(20, f1), t + s.dec * 0.6); }
		const g = gainNode(c);
		pdec(g.gain, t, vel * s.tone * 1.1, s.dec);
		o.connect(g).connect(d);
		nodes.push(o, g);
		if (s.ring) {
			const r = osc(c, 'triangle', f1 * s.ring, t, t + s.dec * 1.3);
			const rg = gainNode(c);
			pdec(rg.gain, t, vel * s.tone * 0.5, s.dec * 1.3);
			r.connect(rg).connect(d);
			nodes.push(r, rg);
		}
	}
	if (s.noise > 0) {
		const ns = noiseSrc(k, t, s.nDec + 0.05);
		const bp = c.createBiquadFilter(); bp.type = 'bandpass'; bp.frequency.value = s.nHz; bp.Q.value = s.nQ;
		const g = gainNode(c);
		pdec(g.gain, t, vel * s.noise * 1.4, s.nDec);
		ns.connect(bp).connect(g).connect(d);
		nodes.push(ns, bp, g);
	}
	bin(k, t + Math.max(s.dec * 1.3, s.nDec) + 0.15, ...nodes);
}

/** a clap / hand-clap for feasts */
export function clap(k: Kit, d: AudioNode, t: number, vel: number): void {
	hit(k, d, t, 'electro', 3, vel * 0.8, 100);
}

// ---------- texture one-shots ----------
export type TexKind = 'geiger' | 'static' | 'hiss' | 'bubble' | 'drip' | 'tick';
export function tex(k: Kit, d: AudioNode, t: number, kind: TexKind, vel: number): void {
	if (!claim(k, t, 0.4, 0)) return;
	const c = k.c;
	if (kind === 'geiger' || kind === 'tick' || kind === 'static' || kind === 'hiss') {
		const len = kind === 'geiger' ? 0.012 : kind === 'tick' ? 0.03 : kind === 'static' ? 0.2 : 0.4;
		const ns = noiseSrc(k, t, len + 0.02);
		const f = c.createBiquadFilter();
		f.type = kind === 'static' ? 'bandpass' : kind === 'tick' ? 'bandpass' : 'highpass';
		f.frequency.value = kind === 'geiger' ? 2800 : kind === 'static' ? 1800 : kind === 'tick' ? 3500 : 4500;
		f.Q.value = kind === 'tick' ? 5 : 0.8;
		const g = gainNode(c);
		if (kind === 'hiss') { g.gain.setValueAtTime(0, t); g.gain.linearRampToValueAtTime(vel, t + 0.12); g.gain.linearRampToValueAtTime(0, t + len); }
		else if (kind === 'static') { for (let i = 0; i < 4; i++) g.gain.setValueAtTime(Math.random() < 0.6 ? vel * Math.random() : 0, t + i * 0.05); g.gain.setValueAtTime(0, t + len); }
		else { g.gain.setValueAtTime(vel, t); g.gain.setValueAtTime(0, t + len); }
		ns.connect(f).connect(g).connect(d);
		bin(k, t + len + 0.1, ns, f, g);
	} else {
		const f0 = kind === 'bubble' ? 300 + Math.random() * 300 : 1500 + Math.random() * 700;
		const o = osc(c, 'sine', f0, t, t + 0.3);
		o.frequency.setValueAtTime(f0, t);
		o.frequency.exponentialRampToValueAtTime(kind === 'bubble' ? f0 * 2.6 : f0 * 0.8, t + (kind === 'bubble' ? 0.09 : 0.15));
		const g = gainNode(c);
		pdec(g.gain, t, vel, kind === 'bubble' ? 0.11 : 0.25);
		o.connect(g).connect(d);
		bin(k, t + 0.4, o, g);
	}
}
