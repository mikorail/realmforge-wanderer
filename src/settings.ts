/** Player preferences, persisted in localStorage. Plain module: the UI holds the value in $state and passes it around. */

/** auto picks drift / pastoral / desert by place (garden.ts); chiptune and ambient are the original sequencer */
export type MusicStyle = 'auto' | 'drift' | 'pastoral' | 'desert' | 'chiptune' | 'ambient';
export const MUSIC_STYLES: MusicStyle[] = ['auto', 'drift', 'pastoral', 'desert', 'chiptune', 'ambient'];
export type InstrumentFlavour = 'cultural' | 'chiptune';
export type PaletteName = 'qud' | 'classic' | 'contrast';

export interface Settings {
	music: boolean;
	/** 0..1 */
	musicVolume: number;
	musicStyle: MusicStyle;
	/** ambient weather sound effects (src/weathersfx.ts), separate from the music */
	sfx: boolean;
	/** 0..1 */
	sfxVolume: number;
	/** settings revision for one-time migrations (2 = garden styles exist) */
	musicRev?: number;
	/** cultural: per-land synthesised instruments; chiptune: pulse/triangle/noise only */
	instrumentFlavour: InstrumentFlavour;
	/** 'auto' fits the grid to the window; a number multiplies the fitted cell height */
	scale: 'auto' | number;
	/** 'qud' uses the engine palette unchanged */
	palette: PaletteName;
	animations: boolean;
	/** keys between autosaves */
	autosaveEvery: number;
	showHints: boolean;
}

export const DEFAULT_SETTINGS: Settings = {
	music: true,
	musicVolume: 0.35,
	musicStyle: 'auto',
	sfx: true,
	sfxVolume: 0.5,
	musicRev: 2,
	instrumentFlavour: 'cultural',
	scale: 'auto',
	palette: 'qud',
	animations: true,
	autosaveEvery: 150,
	showHints: true,
};

export const SCALE_OPTIONS = [0.75, 1, 1.25, 1.5] as const;
export const AUTOSAVE_OPTIONS = [50, 150, 400] as const;

const KEY = 'wanderer:settings';

function clean(raw: unknown): Settings {
	const r: Record<string, unknown> = typeof raw === 'object' && raw !== null ? (raw as Record<string, unknown>) : {};
	const s: Settings = { ...DEFAULT_SETTINGS };
	if (typeof r.music === 'boolean') s.music = r.music;
	const vol = r.musicVolume;
	if (typeof vol === 'number' && Number.isFinite(vol)) s.musicVolume = Math.min(1, Math.max(0, vol));
	if (typeof r.sfx === 'boolean') s.sfx = r.sfx;
	const sv = r.sfxVolume;
	if (typeof sv === 'number' && Number.isFinite(sv)) s.sfxVolume = Math.min(1, Math.max(0, sv));
	if (MUSIC_STYLES.includes(r.musicStyle as MusicStyle)) s.musicStyle = r.musicStyle as MusicStyle;
	// before the garden styles every saved settings object said 'chiptune' (the old default): move it to auto once
	if (r.musicRev !== 2 && s.musicStyle === 'chiptune') s.musicStyle = 'auto';
	s.musicRev = 2;
	if (r.instrumentFlavour === 'cultural' || r.instrumentFlavour === 'chiptune') s.instrumentFlavour = r.instrumentFlavour;
	const sc = r.scale;
	if (sc === 'auto') s.scale = 'auto';
	else if (typeof sc === 'number' && (SCALE_OPTIONS as readonly number[]).includes(sc)) s.scale = sc;
	const p = r.palette;
	if (p === 'qud' || p === 'classic' || p === 'contrast') s.palette = p;
	if (typeof r.animations === 'boolean') s.animations = r.animations;
	const a = r.autosaveEvery;
	if (typeof a === 'number' && (AUTOSAVE_OPTIONS as readonly number[]).includes(a)) s.autosaveEvery = a;
	if (typeof r.showHints === 'boolean') s.showHints = r.showHints;
	return s;
}

export function loadSettings(): Settings {
	try {
		const raw = localStorage.getItem(KEY);
		return clean(raw ? JSON.parse(raw) : null);
	} catch {
		return clean(null);
	}
}

export function saveSettings(s: Settings): void {
	try {
		localStorage.setItem(KEY, JSON.stringify(s));
	} catch {
		// storage blocked or full: the choice lasts until the page is closed
	}
}

/**
 * Replacement palettes. Same index order as the engine palette (engine/ui.go):
 * 0..17 are the Qud colours k K y Y b B g G c C r R m M w W o O,
 * 18..22 are the backgrounds bg, bgLight, bgDark, bgSel, bgRed.
 */
export const PALETTES: Record<'classic' | 'contrast', string[]> = {
	classic: [
		'#000000', '#555555', '#aaaaaa', '#ffffff',
		'#0000aa', '#5555ff', '#00aa00', '#55ff55',
		'#00aaaa', '#55ffff', '#aa0000', '#ff5555',
		'#aa00aa', '#ff55ff', '#aa5500', '#ffff55',
		'#c05000', '#ff8800',
		'#000000', '#1a1a1a', '#080808', '#303030', '#2a0000',
	],
	contrast: [
		'#101010', '#909090', '#e0e0e0', '#ffffff',
		'#3b6cff', '#66b3ff', '#00c800', '#66ff66',
		'#00e5ff', '#aef6ff', '#ff4040', '#ff8080',
		'#e040ff', '#ffa0ff', '#ffb000', '#ffff00',
		'#ff7000', '#ffb060',
		'#000000', '#1a1a1a', '#000000', '#3a3a3a', '#3a0000',
	],
};
