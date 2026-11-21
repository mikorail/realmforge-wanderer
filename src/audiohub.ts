/**
 * The audio hub: one shared AudioContext and master limiter for the five depth-wave layers (soundscapes,
 * species voices, music layers and stingers, item foley, culture music), fed from each engine frame.
 * The older audio (music.ts, weathersfx.ts, gamesfx.ts) keeps its own contexts; the hub only sits beside them.
 *
 * Rules it keeps: the context is created on the first key press (never before a gesture), it is suspended
 * while the page is hidden or when no layer is on, everything is stopped on dispose, every layer is quiet
 * and fades in. Ducking of the base music goes through `opts.duckBase(factor)` (1 = no duck).
 */
import type { QudFrame } from './qud';
import { createSoundscape, type Soundscape } from './soundscape';
import { createVoices, type VoicesHandle } from './voices';
import { createMusicLayers, type MusicLayers } from './musiclayers';
import { createFoley, FOLEY_COVERS, type FoleyHandle } from './foley';
import { createCultureMusic, type CultureMusic } from './culturemusic';
import { setFoleyCovers } from './gamesfx';

export interface HubPrefs {
	sfx: boolean;
	sfxVolume: number;
	music: boolean;
	musicVolume: number;
	soundscapes: boolean;
	voices: boolean;
	speechBlips: boolean;
	musicLayers: boolean;
	stingers: boolean;
	foley: boolean;
	cultureMusic: boolean;
}

export interface AudioHub {
	/** feed one frame; `key` is set when it came from a key press (the first one starts the audio) */
	frame(f: QudFrame, key?: string): void;
	setPrefs(p: HubPrefs): void;
	dispose(): void;
}

export function createAudioHub(opts: { duckBase: (factor: number) => void }): AudioHub {
	let ctx: AudioContext | null = null;
	let master: GainNode | null = null;
	let sc: Soundscape | null = null;
	let voices: VoicesHandle | null = null;
	let layers: MusicLayers | null = null;
	let foley: FoleyHandle | null = null;
	let cm: CultureMusic | null = null;
	let prefs: HubPrefs | null = null;
	let disposed = false;
	let anyOn = false;
	// ducking: a stinger ducks for its length, performers duck by a fraction; the base music gets the smaller factor
	let stingerFactor = 1;
	let performerFactor = 1;
	let stingerTimer: ReturnType<typeof setTimeout> | null = null;
	let lastDuck = 1;
	const pushDuck = () => {
		const f = Math.min(stingerFactor, performerFactor);
		if (Math.abs(f - lastDuck) < 0.005) return;
		lastDuck = f;
		try {
			opts.duckBase(f);
		} catch {
			/* a failed duck must never break input */
		}
	};

	function ensure(): boolean {
		if (ctx) return true;
		if (disposed) return false;
		const AC = window.AudioContext ?? (window as unknown as { webkitAudioContext?: typeof AudioContext }).webkitAudioContext;
		if (!AC) return false;
		try {
			ctx = new AC();
		} catch {
			return false;
		}
		master = ctx.createGain();
		master.gain.value = 1;
		// a gentle ceiling for the sum of the layers
		const lim = ctx.createDynamicsCompressor();
		lim.threshold.value = -20;
		lim.knee.value = 6;
		lim.ratio.value = 12;
		lim.attack.value = 0.005;
		lim.release.value = 0.25;
		master.connect(lim);
		lim.connect(ctx.destination);
		sc = createSoundscape(ctx, master);
		voices = createVoices(ctx, master);
		layers = createMusicLayers(ctx, master, {});
		foley = createFoley(ctx, master);
		cm = createCultureMusic(ctx, master, {
			onDuck: (amount) => {
				performerFactor = 1 - Math.max(0, Math.min(0.35, amount));
				pushDuck();
			},
		});
		layers.onDuck((db, seconds) => {
			stingerFactor = Math.pow(10, -Math.abs(db) / 20);
			pushDuck();
			if (stingerTimer) clearTimeout(stingerTimer);
			stingerTimer = setTimeout(() => {
				stingerTimer = null;
				stingerFactor = 1;
				pushDuck();
			}, Math.max(0.3, seconds) * 1000 + 500);
		});
		if (prefs) applyPrefs(prefs);
		return true;
	}

	function applyPrefs(p: HubPrefs) {
		const sfxOn = p.sfx && p.sfxVolume > 0;
		const musicOn = p.music && p.musicVolume > 0;
		sc?.setEnabled(sfxOn && p.soundscapes);
		sc?.setVolume(p.sfxVolume);
		voices?.setEnabled(sfxOn);
		voices?.setVolume(p.sfxVolume);
		voices?.setVoices(p.voices);
		voices?.setBlips(p.speechBlips);
		foley?.setEnabled(sfxOn && p.foley);
		foley?.setVolume(p.sfxVolume);
		layers?.setEnabled(musicOn);
		layers?.setVolume(p.musicVolume);
		layers?.setLayers(p.musicLayers);
		layers?.setStingers(p.stingers);
		cm?.setEnabled(musicOn && p.cultureMusic);
		cm?.setVolume(p.musicVolume);
		// the older sfx layer skips the sounds foley now plays, only while foley is on
		setFoleyCovers(sfxOn && p.foley ? FOLEY_COVERS : null);
		anyOn = (sfxOn && (p.soundscapes || p.voices || p.speechBlips || p.foley)) || (musicOn && (p.musicLayers || p.stingers || p.cultureMusic));
		syncRun();
	}

	/** run the context only while something is on and the page is visible */
	function syncRun() {
		if (!ctx) return;
		const want = anyOn && !document.hidden;
		try {
			if (want && ctx.state === 'suspended') void ctx.resume();
			else if (!want && ctx.state === 'running') void ctx.suspend();
		} catch {
			/* ignore */
		}
	}
	const onVis = () => syncRun();
	document.addEventListener('visibilitychange', onVis);
	window.addEventListener('pagehide', onVis);

	return {
		frame(f, key) {
			if (disposed) return;
			if (!ctx) {
				if (!key || !anyOnEver(prefs)) return; // wait for a key press (a user gesture) and a layer that is on
				if (!ensure()) return;
			}
			if (!anyOn) return;
			syncRun();
			const me = f.actors?.find((a) => a.p);
			const listener = me ? { x: me.x, y: me.y } : { x: 40, y: 12 };
			try {
				sc?.set(f.sound);
				voices?.play(f.vox, listener);
				layers?.setMusic(f.music, f.cuex);
				layers?.cue(f.cues);
				foley?.play(f.foley ?? []);
				cm?.set(f.mstyle, f.mode);
			} catch {
				/* audio must never break the game */
			}
		},
		setPrefs(p) {
			prefs = p;
			if (ctx) applyPrefs(p);
			else setFoleyCovers(null);
		},
		dispose() {
			if (disposed) return;
			disposed = true;
			document.removeEventListener('visibilitychange', onVis);
			window.removeEventListener('pagehide', onVis);
			if (stingerTimer) clearTimeout(stingerTimer);
			setFoleyCovers(null);
			sc?.dispose();
			voices?.dispose();
			layers?.dispose();
			foley?.dispose();
			cm?.dispose();
			try {
				master?.disconnect();
				void ctx?.close();
			} catch {
				/* ignore */
			}
			ctx = null;
			opts.duckBase(1);
		},
	};
}

function anyOnEver(p: HubPrefs | null): boolean {
	if (!p) return false;
	const sfxOn = p.sfx && p.sfxVolume > 0;
	const musicOn = p.music && p.musicVolume > 0;
	return (sfxOn && (p.soundscapes || p.voices || p.speechBlips || p.foley)) || (musicOn && (p.musicLayers || p.stingers || p.cultureMusic));
}
