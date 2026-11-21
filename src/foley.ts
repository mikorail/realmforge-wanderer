// Item- and material-aware foley, synthesised with WebAudio on the coordinator's context.
//
// Events come from `frame.foley` (engine/foleyinfo.go FoleyBegin/FoleyEnd). About 25 recipes (noise bursts
// through filters, short resonant pings, FM metals, sine thumps) are parameterised by material, weight, kind,
// weapon family, genre gun profile, shoe and surface to give roughly 60 distinct sounds. Pitch gets a small
// seeded variation (hash of item name + a per-event counter), never Math.random.
//
// Doubling with gamesfx.ts: FOLEY_COVERS lists the gamesfx ids this layer replaces. The coordinator should skip
// them in gamesfx handleFrame when foley is enabled AND the frame carries a non-empty `foley` array (so a frame
// the engine could not classify still falls back to the generic sound). Ids not listed (door, doorclose, hurt,
// death, crit, parry, cast:*, levelup, quest, alarm, ui:*) stay with gamesfx.

export interface FoleyItem {
	kind: string;
	family: string;
	material: string;
	rarity: string;
	hash: number;
	weight: number;
	name?: string;
}
export interface FoleyEvent {
	id: string;
	item?: FoleyItem;
	count: number;
	surface: string;
	cond?: string;
	shoe: string;
	genre: string;
	gun?: string;
}
export interface FoleyHandle {
	play(events: FoleyEvent[]): void;
	setEnabled(on: boolean): void;
	setVolume(v: number): void;
	dispose(): void;
}

/** gamesfx ids replaced by foley (skip them in gamesfx handleFrame when foley is on and frame.foley is non-empty). */
export const FOLEY_COVERS: ReadonlySet<string> = new Set([
	'pickup', 'coin', 'equip', 'eat', 'drink', 'page', 'lock', 'miss', 'block',
	'step:stone', 'step:grass', 'step:sand', 'step:snow', 'step:wood', 'step:metal', 'step:water',
	'hit:blade', 'hit:blunt', 'hit:pierce', 'hit:gun', 'hit:bow', 'shot:gun', 'shot:bow',
]);

// material: [base pitch Hz, ring seconds, FM ratio, brightness 0..1]
const MAT: Record<string, [number, number, number, number]> = {
	iron: [620, 0.22, 2.7, 0.55], steel: [900, 0.38, 3.1, 0.7], bronze: [520, 0.45, 2.4, 0.5], gold: [1100, 0.5, 2.0, 0.6],
	silver: [1500, 0.6, 2.2, 0.75], crystal: [2400, 0.7, 1.5, 0.9], glass: [2000, 0.4, 1.0, 0.85], stone: [300, 0.1, 1.4, 0.3],
	bone: [700, 0.08, 1.8, 0.4], wood: [320, 0.09, 1.2, 0.3], leather: [200, 0.06, 1, 0.2], cloth: [150, 0.05, 1, 0.3],
	bio: [200, 0.06, 1, 0.15], plasma: [1300, 0.35, 5, 0.8],
};
const mat = (m?: string) => MAT[m ?? 'iron'] ?? MAT.iron;

export function createFoley(ctx: AudioContext, dest: AudioNode): FoleyHandle {
	const MAXV = 10;
	const out = ctx.createGain(); // peak trim: voices <= ~0.22, limiter below, out <= 0.35 -> under -20 dBFS
	out.gain.value = 0;
	const lim = ctx.createDynamicsCompressor();
	lim.threshold.value = -26;
	lim.knee.value = 6;
	lim.ratio.value = 12;
	lim.attack.value = 0.002;
	lim.release.value = 0.12;
	lim.connect(out);
	out.connect(dest);
	let enabled = true;
	let volume = 0.5;
	let disposed = false;
	let voices = 0;
	let counter = 0;
	const live = new Set<AudioScheduledSourceNode>();
	let noiseBuf: AudioBuffer | null = null;
	let seed = 1;

	const level = () => 0.35 * Math.min(1, Math.max(0, volume));
	function applyLevel(fade: number) {
		const t = ctx.currentTime;
		out.gain.cancelScheduledValues(t);
		out.gain.setValueAtTime(out.gain.value, t);
		out.gain.linearRampToValueAtTime(enabled ? level() : 0, t + fade);
	}
	applyLevel(1.6);

	const rnd = () => {
		seed = (seed * 1664525 + 1013904223) >>> 0;
		return seed / 4294967296;
	};
	const jit = (f: number, a = 0.05) => f * (1 + (rnd() * 2 - 1) * a);

	function getNoise(): AudioBuffer {
		if (noiseBuf) return noiseBuf;
		const n = Math.floor(ctx.sampleRate * 1.2);
		const b = ctx.createBuffer(1, n, ctx.sampleRate);
		const d = b.getChannelData(0);
		let s = 12345;
		for (let i = 0; i < n; i++) {
			s = (s * 1103515245 + 12345) >>> 0;
			d[i] = s / 2147483648 - 1;
		}
		return (noiseBuf = b);
	}
	function track(src: AudioScheduledSourceNode, end: number, nodes: AudioNode[]) {
		live.add(src);
		src.onended = () => {
			live.delete(src);
			for (const n of nodes) try { n.disconnect(); } catch { /* gone */ }
		};
		src.stop(end);
	}
	function env(g: GainNode, t: number, peak: number, dur: number, att = 0.003) {
		g.gain.setValueAtTime(0.0001, t);
		g.gain.linearRampToValueAtTime(peak, t + att);
		g.gain.exponentialRampToValueAtTime(0.0001, t + Math.max(dur, att + 0.01));
	}
	// ---- primitives ----
	function noise(dur: number, type: BiquadFilterType, f0: number, f1: number, q: number, gain: number, at = 0, att = 0.004) {
		const t = ctx.currentTime + 0.01 + at;
		const s = ctx.createBufferSource();
		s.buffer = getNoise();
		const f = ctx.createBiquadFilter();
		f.type = type;
		f.Q.value = q;
		f.frequency.setValueAtTime(Math.max(20, f0), t);
		f.frequency.exponentialRampToValueAtTime(Math.max(20, f1), t + dur);
		const g = ctx.createGain();
		env(g, t, gain, dur, att);
		s.connect(f); f.connect(g); g.connect(lim);
		s.start(t, rnd() * 0.5);
		track(s, t + dur + 0.05, [s, f, g]);
	}
	function tone(type: OscillatorType, f0: number, f1: number, dur: number, gain: number, at = 0, att = 0.003) {
		const t = ctx.currentTime + 0.01 + at;
		const o = ctx.createOscillator();
		o.type = type;
		o.frequency.setValueAtTime(Math.max(20, f0), t);
		o.frequency.exponentialRampToValueAtTime(Math.max(20, f1), t + dur);
		const g = ctx.createGain();
		env(g, t, gain, dur, att);
		o.connect(g); g.connect(lim);
		o.start(t);
		track(o, t + dur + 0.05, [o, g]);
	}
	/** FM metal: carrier at f, modulator at f*ratio with decaying index */
	function fm(f: number, ratio: number, idx: number, dur: number, gain: number, at = 0) {
		const t = ctx.currentTime + 0.01 + at;
		const c = ctx.createOscillator(), m = ctx.createOscillator();
		const mg = ctx.createGain(), g = ctx.createGain();
		c.frequency.value = f; m.frequency.value = f * ratio;
		mg.gain.setValueAtTime(f * idx, t);
		mg.gain.exponentialRampToValueAtTime(1, t + dur);
		env(g, t, gain, dur, 0.002);
		m.connect(mg); mg.connect(c.frequency); c.connect(g); g.connect(lim);
		c.start(t); m.start(t);
		track(c, t + dur + 0.05, [c, mg, g]);
		track(m, t + dur + 0.05, [m]);
	}
	const ping = (f: number, dur: number, gain: number, at = 0) => tone('sine', f, f * 0.995, dur, gain, at, 0.001);

	// ---- recipes ----
	/** 1 clank: metal on metal by material and weight */
	const clank = (m: string, w: number, g = 0.16) => {
		const [f, ring, ratio, br] = mat(m); const k = 1 + Math.min(w, 12) * 0.06;
		fm(jit(f / k), ratio, 1.5 + br * 2, ring * k, g);
		noise(0.04, 'highpass', 2500 * (0.5 + br), 3000, 1, g * 0.6);
		if (w >= 6) tone('sine', 110, 70, 0.14, g * 0.8);
	};
	/** 2 jingle: n bright pings (mail, keys, chains) */
	const jingle = (m: string, n: number, g = 0.07) => {
		const [f, ring] = mat(m);
		for (let i = 0; i < n; i++) ping(jit(f * (1.6 + rnd()), 0.08), 0.06 + ring * 0.25, g, i * (0.025 + rnd() * 0.035));
	};
	/** 3 creak: leather, lid, wood */
	const creak = (f: number, dur: number, g = 0.1) => {
		tone('sawtooth', jit(f), f * (1.2 + rnd() * 0.5), dur, g * 0.5, 0, 0.04);
		noise(dur, 'bandpass', f * 3, f * 4, 4, g * 0.5, 0, 0.05);
	};
	/** 4 rustle: cloth, paper */
	const rustle = (bright: number, n = 3, g = 0.1) => {
		for (let i = 0; i < n; i++) noise(0.05 + rnd() * 0.06, 'bandpass', 1500 + bright * 3500, 900 + bright * 2000, 0.8, g, i * 0.045, 0.01);
	};
	/** 5 thump */
	const thump = (f: number, g = 0.2, dur = 0.14) => tone('sine', f, f * 0.45, dur, g, 0, 0.002);
	/** 6 clink: glass or small hard objects */
	const clink = (f: number, g = 0.1) => { ping(jit(f), 0.12, g); ping(jit(f * 1.5), 0.08, g * 0.6, 0.012); };
	/** 7 chink: coin chink scaled by amount */
	const chink = (amount: number, g = 0.1) => {
		const n = Math.min(6, 1 + Math.floor(Math.log2(Math.max(1, amount)) / 2));
		for (let i = 0; i < n; i++) { const f = jit(2400 + rnd() * 900); ping(f, 0.1, g, i * (0.03 + rnd() * 0.03)); ping(f * 1.5, 0.06, g * 0.5, i * 0.04); }
	};
	/** 8 tinkle: gem */
	const tinkle = (g = 0.07) => { for (let i = 0; i < 4; i++) ping(jit(2600 + i * 700 + rnd() * 400), 0.18, g, i * 0.05); };
	/** 9 squish: food, flesh */
	const squish = (g = 0.14) => { noise(0.1, 'lowpass', 900, 250, 1.2, g); tone('sine', jit(180), 90, 0.08, g * 0.6); };
	/** 10 slosh: liquid */
	const slosh = (g = 0.1) => { for (let i = 0; i < 3; i++) noise(0.16, 'bandpass', jit(500 + i * 120), 250, 2, g, i * 0.07, 0.03); };
	/** 11 fizz */
	const fizz = (dur = 0.5, g = 0.05) => noise(dur, 'highpass', 6000, 9000, 0.6, g, 0.05, 0.1);
	/** 12 gulp */
	const gulp = (n: number, g = 0.12) => { for (let i = 0; i < n; i++) { tone('sine', jit(300), 520, 0.07, g, i * 0.13); noise(0.05, 'lowpass', 700, 400, 1, g * 0.4, i * 0.13); } };
	/** 13 crunch */
	const crunch = (n: number, g = 0.12) => { for (let i = 0; i < n; i++) noise(0.04, 'bandpass', jit(1800), 900, 1.4, g, i * 0.09, 0.002); };
	/** 14 sizzle */
	const sizzle = (g = 0.06) => noise(0.9, 'highpass', 4000, 7000, 0.7, g, 0, 0.1);
	/** 15 chop */
	const chop = (g = 0.16) => { noise(0.05, 'bandpass', 1400, 600, 2, g); thump(140, g * 0.9, 0.07); };
	/** 16 anvil hammer */
	const anvil = (g = 0.14) => { clank('steel', 8, g); ping(jit(1800), 0.5, g * 0.4, 0.01); };
	/** 17 grind */
	const grind = (g = 0.09) => { noise(0.5, 'bandpass', 900, 700, 1.2, g, 0, 0.08); tone('sawtooth', 70, 60, 0.5, g * 0.3, 0, 0.1); };
	/** 18 bubble */
	const bubble = (n: number, g = 0.08) => { for (let i = 0; i < n; i++) tone('sine', jit(300 + rnd() * 300), 700 + rnd() * 500, 0.07, g, i * (0.07 + rnd() * 0.08)); };
	/** 19 whoosh: swing by weight */
	const whoosh = (w: number, g = 0.09) => noise(0.16 + w * 0.02, 'bandpass', 500 + (12 - Math.min(w, 12)) * 150, 2200, 1.3, g, 0, 0.06);
	/** 20 crack: whip */
	const crack = (g = 0.15) => { noise(0.03, 'highpass', 3500, 6000, 1, g, 0.07, 0.001); whoosh(2, g * 0.6); };
	/** 21 twang: bow */
	const twang = (g = 0.1) => { tone('triangle', jit(180), 150, 0.2, g, 0, 0.001); noise(0.2, 'bandpass', 2600, 1200, 2, g * 0.5, 0.06, 0.05); };
	/** 22 thunk: crossbow */
	const thunk = (g = 0.14) => { thump(120, g); noise(0.04, 'bandpass', 1800, 900, 2, g * 0.7); };
	/** 23 whirr: sling */
	const whirr = (g = 0.08) => noise(0.4, 'bandpass', 600, 1800, 3, g, 0, 0.12);
	/** 24 gun report by genre profile */
	const gun = (p: string, g = 0.18) => {
		switch (p) {
			case 'flintlock': noise(0.3, 'lowpass', 2200, 300, 0.7, g, 0.02, 0.002); thump(100, g); noise(0.04, 'highpass', 3000, 3000, 1, g * 0.4); break;
			case 'revolver': noise(0.16, 'bandpass', 1800, 500, 1.1, g); thump(150, g * 0.9, 0.1); noise(0.25, 'lowpass', 500, 200, 0.6, g * 0.3, 0.05); break;
			case 'rifle': noise(0.35, 'lowpass', 3000, 250, 0.8, g); thump(90, g, 0.2); noise(0.5, 'lowpass', 400, 120, 0.5, g * 0.3, 0.08); break;
			case 'plasma': tone('sawtooth', 1400, 200, 0.28, g * 0.6); noise(0.2, 'bandpass', 3000, 800, 3, g * 0.4); break;
			case 'railgun': tone('sine', 90, 2400, 0.12, g * 0.4); noise(0.05, 'highpass', 4000, 4000, 1, g); thump(70, g, 0.3); break;
			default: tone('square', 1800, 300, 0.1, g * 0.5); noise(0.08, 'bandpass', 4000, 2000, 2, g * 0.3); // energy zap
		}
	};
	/** 25 step by shoe, surface, condition */
	const step = (shoe: string, surf: string, cond: string, g = 0.1) => {
		const hard = surf === 'stone' || surf === 'metal' || surf === 'wood';
		if (surf === 'water') { noise(0.18, 'bandpass', 700, 300, 1.2, g, 0, 0.02); tone('sine', jit(300), 200, 0.08, g * 0.5); return; }
		if (shoe === 'hooves') { noise(0.05, 'bandpass', hard ? 2200 : 900, 600, 2, g * 1.2, 0, 0.002); thump(hard ? 200 : 110, g, 0.08); }
		else if (shoe === 'boots') {
			noise(0.06, 'bandpass', jit(hard ? 1500 : 700), 400, 1.4, g, 0, 0.003); thump(surf === 'wood' ? 150 : 100, g * 0.8, 0.09);
			if (surf === 'metal') ping(jit(1100), 0.12, g * 0.4);
			if (surf === 'stone') noise(0.03, 'highpass', 3500, 3500, 1, g * 0.3);
		} else if (shoe === 'sandals') { noise(0.07, 'bandpass', jit(1800), 900, 1, g * 0.7, 0, 0.004); thump(130, g * 0.4, 0.05); }
		else { noise(0.06, 'lowpass', jit(900), 350, 0.8, g * 0.7, 0, 0.006); thump(surf === 'wood' ? 130 : 90, g * 0.5, 0.06); }
		if (surf === 'grass') noise(0.1, 'highpass', 4500, 3000, 0.6, g * 0.35, 0.01, 0.02);
		if (surf === 'sand') noise(0.12, 'highpass', 3000, 1800, 0.6, g * 0.5, 0, 0.02);
		if (surf === 'snow' || cond === 'snow') noise(0.12, 'bandpass', 2600, 1400, 1.4, g * 0.6, 0.01, 0.01);
		if (cond === 'mud') { noise(0.14, 'lowpass', 500, 180, 1.4, g * 1.1, 0.02, 0.01); tone('sine', 140, 70, 0.1, g * 0.4, 0.02); }
		else if (cond === 'wet') { noise(0.1, 'highpass', 3000, 2000, 0.8, g * 0.5, 0.02, 0.01); }
	};
	/** 26 lock click */
	const lockClick = (g = 0.1) => { ping(jit(1500), 0.04, g); ping(jit(1000), 0.05, g, 0.07); noise(0.02, 'highpass', 4000, 4000, 1, g * 0.4, 0.07); };
	/** 27 jam: dull dry click */
	const jam = (g = 0.1) => { tone('square', 220, 160, 0.04, g * 0.7); tone('square', 180, 140, 0.05, g * 0.7, 0.09); };

	const weightOf = (it?: FoleyItem) => (it ? Math.max(1, it.weight || 1) : 1);

	function armour(it: FoleyItem, on: boolean) {
		const m = it.material, w = weightOf(it), s = on ? 1 : 0.8;
		if (m === 'steel' || m === 'iron' || m === 'bronze') {
			if (w >= 6) { clank(m, w, 0.14 * s); noise(0.2, 'bandpass', 1200, 700, 2, 0.06, 0.04); } else jingle(m, 5, 0.06 * s);
		} else if (m === 'leather') creak(on ? 180 : 150, 0.2, 0.1 * s);
		else if (m === 'cloth') rustle(0.4, on ? 4 : 3, 0.1 * s);
		else if (m === 'bone' || m === 'stone') { clank(m, w, 0.1); }
		else if (m === 'plasma' || m === 'crystal' || m === 'glass') { clink(1800, 0.08); tone('sine', 600, 900, 0.15, 0.05, 0.03); }
		else { rustle(0.3, 2, 0.1); thump(120, 0.1); }
	}
	function weaponDraw(it: FoleyItem, draw: boolean) {
		const fam = it.family, m = it.material;
		if (fam === 'longbow' || fam === 'sling' || fam === 'blowgun') { creak(draw ? 200 : 170, 0.12, 0.08); return; }
		if (fam === 'pistol' || fam === 'longgun' || fam === 'repeater' || fam === 'crossbow') { noise(0.05, 'bandpass', 1500, 900, 2, 0.1); tone('square', draw ? 300 : 240, 200, 0.03, 0.06, 0.05); return; }
		if (fam === 'staff' || fam === 'spear' || fam === 'mace' || fam === 'whip' || fam === 'flail') { creak(150, 0.1, 0.07); thump(120, 0.08, 0.06); return; }
		// blade: ring on draw, scrape on sheathe
		if (draw) { noise(0.22, 'bandpass', 2000, 5000, 5, 0.1); ping(jit(mat(m)[0] * 2), 0.35, 0.06, 0.1); }
		else { noise(0.22, 'bandpass', 5000, 2000, 5, 0.09); clank('iron', 2, 0.08); }
	}
	function itemSound(it: FoleyItem | undefined, kind: 'pickup' | 'drop' | 'give', n: number) {
		const g = kind === 'drop' ? 0.9 : 1;
		if (!it) { thump(130, 0.08 * g, 0.06); return; }
		const w = weightOf(it);
		const k = it.kind;
		if (k === 'gem') tinkle(0.07 * g);
		else if (k === 'glass') clink(2000, 0.09 * g);
		else if (k === 'book' || k === 'letter' || k === 'scrap') rustle(0.7, 2, 0.1 * g);
		else if (k === 'food') squish(0.12 * g);
		else if (k === 'water' || k === 'tonic') slosh(0.1 * g);
		else if (k === 'bone') { for (let i = 0; i < 3; i++) clank('bone', 1, 0.07 * g); }
		else if (k === 'key') jingle('iron', 4, 0.07 * g);
		else if (k === 'tool') { thump(110, 0.12 * g, 0.08); clank('iron', 5, 0.08 * g); }
		else if (k === 'armor') armour(it, true);
		else if (k === 'weapon' || k === 'ranged' || k === 'ammo' || k === 'grenade') { clank(it.material, w, 0.1 * g); if (n > 1) clank(it.material, w, 0.05); }
		else if (it.material === 'cloth' || it.material === 'leather') rustle(0.3, 2, 0.1 * g);
		else if (it.material === 'wood') { thump(200, 0.12 * g, 0.07); noise(0.04, 'bandpass', 900, 500, 2, 0.08); }
		else clink(mat(it.material)[0], 0.08 * g);
		if (kind === 'drop' && k !== 'gem' && k !== 'book') thump(90, 0.08 * Math.min(w, 8) / 4, 0.1);
	}
	function impact(tgt: string, it: FoleyItem | undefined, shot: boolean) {
		const fam = it?.family ?? '', w = weightOf(it);
		switch (tgt) {
			case 'armour': clank('steel', 6, 0.15); break;
			case 'shield': clank('bronze', 7, 0.14); thump(100, 0.1, 0.1); noise(0.06, 'bandpass', 600, 300, 2, 0.1); break;
			case 'wood': noise(0.06, 'bandpass', 900, 400, 2.5, 0.15); thump(180, 0.14, 0.07); break;
			case 'stone': noise(0.05, 'highpass', 3000, 2500, 1, 0.12); ping(jit(2200), 0.05, 0.06); thump(160, 0.08, 0.06); break;
			case 'metal': clank('iron', w, 0.14); break;
			case 'bone': for (let i = 0; i < 2; i++) noise(0.03, 'bandpass', 1800, 1000, 3, 0.12, i * 0.03); break;
			case 'bio': squish(0.15); break;
			default: // flesh by family
				if (fam === 'mace' || fam === 'flail' || fam === 'staff' || fam === 'fist') { thump(110, 0.2, 0.12); noise(0.05, 'bandpass', 1100, 500, 1.5, 0.12); noise(0.04, 'bandpass', 2000, 1000, 3, 0.07, 0.05); }
				else if (fam === 'axe' || fam === 'taxe' || fam === 'great') { thump(95, 0.2, 0.12); noise(0.06, 'bandpass', 900, 400, 1.2, 0.14); }
				else if (fam === 'dagger' || fam === 'tknife') { noise(0.05, 'bandpass', 3800, 2200, 4, 0.12); ping(jit(2800), 0.08, 0.04); }
				else if (fam === 'spear' || fam === 'javelin') { noise(0.07, 'bandpass', 1600, 800, 2, 0.12); thump(130, 0.12, 0.07); }
				else if (shot) { thump(120, 0.14, 0.08); noise(0.05, 'bandpass', 1500, 700, 1.4, 0.1); }
				else { noise(0.06, 'bandpass', 2400, 900, 2, 0.12); thump(140, 0.14, 0.08); }
		}
	}
	function swing(it: FoleyItem | undefined) {
		const fam = it?.family ?? '', w = weightOf(it);
		if (fam === 'whip') crack();
		else if (fam === 'longbow' || fam === 'atlatl') twang();
		else if (fam === 'crossbow') thunk();
		else if (fam === 'sling') whirr();
		else if (fam === 'spear' || fam === 'javelin') noise(0.14, 'bandpass', 1200, 3200, 2.2, 0.09, 0, 0.03); // thrust whoosh
		else if (fam === 'dagger' || fam === 'tknife') noise(0.07, 'bandpass', 3500, 5000, 3, 0.08, 0, 0.015);
		else if (fam === 'axe' || fam === 'mace' || fam === 'great') whoosh(Math.max(w, 7), 0.1);
		else whoosh(w, 0.08);
	}
	function shotArrowOrGun(e: FoleyEvent) {
		const fam = e.item?.family ?? '';
		if (e.gun || fam === 'pistol' || fam === 'longgun' || fam === 'repeater') gun(e.gun ?? 'zap');
		else if (fam === 'crossbow') thunk();
		else if (fam === 'sling') { whirr(); thump(120, 0.05, 0.05); }
		else if (fam === 'blowgun') noise(0.08, 'bandpass', 2500, 1500, 2, 0.06);
		else { twang(); noise(0.25, 'bandpass', 3000, 1800, 2, 0.04, 0.08, 0.06); } // arrow whisper
	}

	function one(e: FoleyEvent) {
		seed = ((e.item?.hash ?? 7) * 2654435761 + (counter++) * 40503 + e.id.length * 977) >>> 0;
		const it = e.item;
		switch (e.id) {
			case 'equip': case 'unequip': if (it) armour(it, e.id === 'equip'); else rustle(0.3, 3); break;
			case 'draw': case 'sheathe': if (it) weaponDraw(it, e.id === 'draw'); break;
			case 'pickup': itemSound(it, 'pickup', e.count); break;
			case 'drop': itemSound(it, 'drop', e.count); break;
			case 'give': itemSound(it, 'give', e.count); break;
			case 'coin': chink(e.count); break;
			case 'coin:spend': chink(e.count, 0.08); break;
			case 'trade': chink(e.count); if (it) itemSound(it, 'give', 1); break;
			case 'haggle': rustle(0.5, 2, 0.06); tone('triangle', 260, 300, 0.12, 0.05, 0.05); break;
			case 'eat': crunch(Math.min(4, 2 + (it && it.material === 'bio' ? 1 : 0))); if (it && /soup|stew|broth|porridge|jelly/i.test(it.name ?? '')) slosh(0.08); break;
			case 'drink': gulp(3); break;
			case 'potion': slosh(0.1); fizz(0.5, 0.04); gulp(2, 0.08); break;
			case 'smoke': noise(0.6, 'bandpass', 1500, 600, 0.8, 0.05, 0, 0.15); sizzle(0.025); break;
			case 'craft:cook': sizzle(); bubble(3, 0.05); break;
			case 'craft:chop': chop(); chop(0.12); break;
			case 'craft:hammer': anvil(); break;
			case 'craft:grind': grind(); break;
			case 'craft:brew': bubble(6); fizz(0.6, 0.03); break;
			case 'open': creak(it?.material === 'iron' || it?.material === 'steel' ? 260 : 170, 0.3, 0.1); lockClick(0.06); break;
			case 'lock': lockClick(); break;
			case 'jam': jam(); break;
			case 'reload': noise(0.05, 'bandpass', 1500, 900, 2, 0.08); tone('square', 320, 220, 0.03, 0.06, 0.08); clank('iron', 2, 0.07); break;
			case 'page': rustle(0.8, 2, 0.1); break;
			case 'write': noise(0.35, 'bandpass', 3500, 2500, 3, 0.04, 0, 0.1); noise(0.2, 'bandpass', 3200, 2400, 3, 0.03, 0.4, 0.05); break;
			case 'swing': swing(it); break;
			case 'shot': shotArrowOrGun(e); break;
			case 'step': step(e.shoe, e.surface, e.cond ?? '');
				break;
			default:
				if (e.id.startsWith('impact:')) impact(e.id.slice(7), it, false);
		}
	}

	return {
		play(events) {
			if (disposed || !enabled || ctx.state === 'closed') return;
			let n = 0;
			for (const e of events) {
				if (n >= 8) break;
				if (live.size > MAXV * 6 || voices >= MAXV) break; // drop when over budget
				voices++;
				try { one(e); } catch { /* an odd event must not break the frame */ }
				n++;
				setTimeout(() => { voices = Math.max(0, voices - 1); }, 140);
			}
		},
		setEnabled(on) {
			enabled = on;
			if (disposed) return;
			if (!on) for (const s of live) try { s.stop(); } catch { /* stopped */ }
			applyLevel(on ? 1.6 : 0.05);
		},
		setVolume(v) { volume = v; if (!disposed) applyLevel(0.1); },
		dispose() {
			disposed = true;
			for (const s of live) try { s.stop(); } catch { /* stopped */ }
			live.clear();
			try { lim.disconnect(); out.disconnect(); } catch { /* gone */ }
		},
	};
}
