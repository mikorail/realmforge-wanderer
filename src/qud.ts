/** Loads the Go engine (public/wasm/qud.wasm + Go's wasm_exec.js) once per page. */
export interface QudFrame {
	w: number;
	h: number;
	/** glyphs, one rune per cell */
	g: string;
	/** palette index + 65 per cell */
	f: string;
	b: string;
	save: boolean;
	exit: boolean;
	dead: boolean;
	mode: string;
	/** hits, misses, heals and deaths of the last key (engine/hitfx.go) */
	hits?: import('./hitfx').HitEvent[];
	/** cells a visible enemy will hit next turn, when the engine knows (not yet produced) */
	tele?: { x: number; y: number }[];
	/** what the soundtrack should play (engine/game.go MusicState) */
	music: QudMusic;
	/** who stands where on the map (engine/sprinfo.go), for sprite mode */
	actors?: import('./actorsprites').QudActor[];
	/** weather, daylight and lights for the sprite-mode effects layer (engine/fxinfo.go) */
	fx?: QudFx;
	/** worn and wielded gear per person, keyed by actor id (engine/gearinfo.go) */
	gear?: Record<string, import('./gearsprites').GearLook>;
	/** culture building style of the zone (engine/styleinfo.go) */
	style?: import('./culturetiles').CultureStyle;
	/** status auras and item glows (engine/styleinfo.go) */
	statuses?: import('./spellfx').StatusFx[];
	/** where attackers stood at the start of the key, by actor id */
	origins?: Record<string, number[]>;
	/** item icons per list row and sidebar status icons, as JSON (engine/iconinfo.go) */
	icons?: string;
	status?: string;
	/** actor ids asleep (engine/poseinfo.go) */
	sleepers?: number[];
	/** place soundscape, voices, combat-layer extras, stingers, item foley, culture music (audiohub.ts) */
	sound?: import('./soundscape').SoundSceneIn;
	vox?: import('./voices').VoxEvent[];
	cuex?: import('./musiclayers').MusicExtras;
	cues?: import('./musiclayers').StingerCue[];
	foley?: import('./foley').FoleyEvent[];
	mstyle?: import('./culturemusic').MusicStyleFrame;
	/** a rectangle of grid cells the engine blanked for a large sprite: the speaker (talk, trade) or the paper doll (character sheet) */
	tip?: string;
	here?: string;
	portrait?: { x: number; y: number; w: number; h: number; kind: 'talk' | 'trade' | 'sheet'; actor: import('./actorsprites').QudActor; mood?: import('./portraitfx').PortraitFace };
}
export interface QudMusic {
	mood: 'town' | 'wild' | 'dungeon' | 'tower' | 'sky' | 'sea' | 'underdark' | 'crypt' | 'world' | 'dead';
	danger: number;
	genre: string;
	biome: number;
	depth: number;
	day: boolean;
	hp: number;
	seed: number;
	wx?: string; // weather kind id when outdoors (engine/weather.go)
	season?: number; // 0 spring … 3 winter
	doom?: number; // 0..1 world-scale threat
	bolt?: number; // lightning strike counter; a rise means thunder now
	sphere?: string; // culture sphere of the land (instruments, scales)
	feast?: boolean; // feast day in this town
	combat?: boolean; // a hostile is in sight
	shore?: boolean; // open water fills much of this outdoor zone (surf SFX)
	/** the place's culture pack music (instruments, scale word, forms); set by music.setState when the cultureMusic pref is on */
	mstyle?: import('./culturemusic').MusicStyleFrame;
}
export interface QudFx {
	wx?: string; // weather kind id (outdoors only)
	ov?: string; // overlay: rain storm snow fog dust ash spore none
	in?: number; // intensity 0..1
	wind: number; // -1 | 0 | 1
	ph: number; // 0..1 through the day, 0 = midnight
	dl: number; // daylight 0..1
	ind?: boolean; // indoors or underground
	dk?: boolean; // dark zone
	d?: number; // depth underground
	lights: { x: number; y: number; r: number; k: string; fl: number }[];
}
type Err = { error: string };
export interface QudApi {
	palette(): string;
	init(world: string): string | Err;
	newGame(spec: string): QudFrame | Err;
	load(save: string): QudFrame | Err;
	key(k: string): QudFrame | Err | null;
	frame(): QudFrame | null;
	save(): string;
	meta(): string;
	/** host setting for the engine (anim, hints) */
	ctx(x: number, y: number): string;
	ctxDo(id: string, x: number, y: number): QudFrame | Err | null;
	ctxAt(): string;
	tipAt(x: number, y: number): string;
	pref(key: string, on: boolean): void;
	deeds(): string;
	/** life summary + score (engine/halloffame.go) */
	hall(): string;
	/** the end-of-life saga as JSON (engine/saga.go SagaData) */
	saga(): string;
	/** past lives of this world, JSON array (engine HFLegend) */
	legends(json: string): void;
}

declare global {
	interface Window {
		qud?: QudApi;
		Go?: new () => { importObject: WebAssembly.Imports; run(i: WebAssembly.Instance): Promise<void> };
	}
}

let loading: Promise<QudApi> | null = null;

function script(src: string) {
	return new Promise<void>((res, rej) => {
		const s = document.createElement('script');
		s.src = src;
		s.onload = () => res();
		s.onerror = () => rej(new Error(`could not load ${src}`));
		document.head.appendChild(s);
	});
}

export function loadQud(): Promise<QudApi> {
	if (window.qud) return Promise.resolve(window.qud);
	if (loading) return loading;
	loading = (async () => {
		if (!window.Go) await script(import.meta.env.BASE_URL + 'wasm/wasm_exec.js');
		const go = new window.Go!();
		const ready = new Promise<void>((res) => window.addEventListener('qud-ready', () => res(), { once: true }));
		const resp = await fetch(import.meta.env.BASE_URL + 'wasm/qud.wasm');
		if (!resp.ok) throw new Error(`qud.wasm missing (${resp.status})`);
		const { instance } = await WebAssembly.instantiate(await resp.arrayBuffer(), go.importObject);
		void go.run(instance);
		await ready;
		return window.qud!;
	})();
	loading.catch(() => (loading = null));
	return loading;
}
