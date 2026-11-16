/**
 * Replays: the engine records a compact key log (engine/replay.go); this module stores it per run in its own
 * IndexedDB (`realmforge-wanderer-replays`, so store.ts and its version stay untouched), cuts highlight frame
 * clips as a fallback when key playback desyncs, exports / imports `.rfreplay` files (deflated JSON via fflate),
 * and drives playback through window.qud.replay* (cmd/wasm/main.go).
 */
import { deflateSync, inflateSync, strToU8, strFromU8 } from 'fflate';
import type { QudApi, QudFrame } from './qud';
import { loadRun } from './store';

const DB = 'realmforge-wanderer-replays';
const CLIP_FRAMES = 30;
const MAX_CLIPS = 40;
const RING = 36;

export interface Highlight {
	k: number;
	t: number;
	kind: 'deed' | 'level' | 'boss' | 'death';
	text: string;
}
export interface ReplayMeta {
	/** `live:<worldSeed>` while the run goes on, the hall entry id once it ended */
	id: string;
	seed: string;
	name: string;
	species: string;
	calling: string;
	level: number;
	world: string;
	genre: string;
	dead: boolean;
	saved: number;
	keys: number;
	highlights: number;
	final: boolean;
}
export interface FrameLite {
	w: number;
	h: number;
	g: string;
	f: string;
	b: string;
}
export interface Clip {
	hl: Highlight;
	frames: FrameLite[];
}
export interface ReplayInfo {
	n: number;
	total: number;
	turn: number;
	diverged: string;
	done: boolean;
	hl: Highlight[];
	name: string;
	trunc: boolean;
	dead: boolean;
	exit: boolean;
	mode: string;
}
type RpFrame = QudFrame & { rpHl?: number; info?: ReplayInfo | string; error?: string };
/** window.qud replay calls added to cmd/wasm/main.go */
export interface ReplayApi extends QudApi {
	replay(): string;
	replayRestore(json: string): string;
	replayHl(): string;
	replayOpen(world: string, data: string): RpFrame | { error: string };
	replayStep(n: number): RpFrame | null;
	replayTo(n: number): RpFrame | null;
	replayHighlight(dir: number): RpFrame | null;
	replayRestart(): RpFrame | null;
	replayClose(): void;
}
export const hasReplayApi = (a: QudApi): a is ReplayApi => typeof (a as ReplayApi).replayOpen === 'function';

// ---- storage ----
function open(): Promise<IDBDatabase> {
	return new Promise((res, rej) => {
		const r = indexedDB.open(DB, 1);
		r.onupgradeneeded = () => {
			for (const st of ['meta', 'data', 'world', 'clips']) if (!r.result.objectStoreNames.contains(st)) r.result.createObjectStore(st);
		};
		r.onsuccess = () => res(r.result);
		r.onerror = () => rej(r.error);
	});
}
function req<T>(r: IDBRequest<T>): Promise<T> {
	return new Promise((res, rej) => {
		r.onsuccess = () => res(r.result);
		r.onerror = () => rej(r.error);
	});
}
const done = (tx: IDBTransaction) =>
	new Promise<void>((res, rej) => {
		tx.oncomplete = () => res();
		tx.onerror = () => rej(tx.error);
		tx.onabort = () => rej(tx.error);
	});
const pack = (s: string) => deflateSync(strToU8(s), { level: 6 });
const unpack = (b: Uint8Array) => strFromU8(inflateSync(b));

export async function listReplays(): Promise<ReplayMeta[]> {
	try {
		const db = await open();
		const all = await req<ReplayMeta[]>(db.transaction('meta').objectStore('meta').getAll());
		return all.sort((a, b) => b.saved - a.saved);
	} catch {
		return [];
	}
}
export async function hasReplay(id: string): Promise<boolean> {
	try {
		const db = await open();
		return !!(await req(db.transaction('meta').objectStore('meta').get(id)));
	} catch {
		return false;
	}
}
export async function deleteReplay(id: string) {
	const db = await open();
	const tx = db.transaction(['meta', 'data', 'clips'], 'readwrite');
	for (const st of ['meta', 'data', 'clips']) tx.objectStore(st).delete(id);
	await done(tx);
	await pruneWorlds();
}
/** worlds are shared by seed; drop the ones no replay refers to */
async function pruneWorlds() {
	const db = await open();
	const metas = await req<ReplayMeta[]>(db.transaction('meta').objectStore('meta').getAll());
	const keep = new Set(metas.map((m) => m.seed));
	const keys = await req<IDBValidKey[]>(db.transaction('world').objectStore('world').getAllKeys());
	const tx = db.transaction('world', 'readwrite');
	for (const k of keys) if (!keep.has(String(k))) tx.objectStore('world').delete(k);
	await done(tx);
}
async function put(meta: ReplayMeta, data: string, world: string | null, clips: Clip[] | null) {
	const db = await open();
	const tx = db.transaction(['meta', 'data', 'world', 'clips'], 'readwrite');
	tx.objectStore('meta').put(meta, meta.id);
	tx.objectStore('data').put(pack(data), meta.id);
	if (world) tx.objectStore('world').put(pack(world), meta.seed);
	if (clips) tx.objectStore('clips').put(pack(JSON.stringify(clips)), meta.id);
	await done(tx);
}
export async function loadReplay(id: string): Promise<{ meta: ReplayMeta; data: string; world: string; clips: Clip[] } | null> {
	const db = await open();
	const meta = await req<ReplayMeta | undefined>(db.transaction('meta').objectStore('meta').get(id));
	if (!meta) return null;
	const [d, w, c] = await Promise.all([
		req<Uint8Array | undefined>(db.transaction('data').objectStore('data').get(id)),
		req<Uint8Array | undefined>(db.transaction('world').objectStore('world').get(meta.seed)),
		req<Uint8Array | undefined>(db.transaction('clips').objectStore('clips').get(id))
	]);
	if (!d || !w) return null;
	return { meta, data: unpack(d), world: unpack(w), clips: c ? JSON.parse(unpack(c)) : [] };
}

// ---- recorder (called from Play.svelte) ----
let ring: FrameLite[] = [];
let clips: Clip[] = [];
let lastHl = 0;
let enabled = true;
export const setReplayRecording = (on: boolean) => (enabled = on);

/** Reset when a run starts or resumes. */
export function replayReset() {
	ring = [];
	clips = [];
	lastHl = 0;
}
export function loadClipsFrom(c: Clip[]) {
	clips = c.slice(-MAX_CLIPS);
}

/** After each engine frame: keep the last frames; cut a clip when the engine says a highlight happened. */
export function replayTap(api: QudApi, f: QudFrame) {
	if (!enabled || !hasReplayApi(api)) return;
	ring.push({ w: f.w, h: f.h, g: f.g, f: f.f, b: f.b });
	if (ring.length > RING) ring.shift();
	const n = (f as RpFrame).rpHl ?? 0;
	if (n < lastHl) lastHl = n; // fresh game
	if (n > lastHl) {
		lastHl = n;
		try {
			const hl = JSON.parse(api.replayHl() || 'null') as Highlight | null;
			if (hl) {
				clips.push({ hl, frames: ring.slice(-CLIP_FRAMES) });
				if (clips.length > MAX_CLIPS) clips.splice(0, clips.length - MAX_CLIPS);
			}
		} catch {
			/* a missing clip is not worth an error */
		}
	}
}

async function ensureWorld(seed: string, world?: string): Promise<string | null> {
	const db = await open();
	if (await req(db.transaction('world').objectStore('world').getKey(seed))) return null;
	if (world) return world;
	const run = await loadRun(seed).catch(() => null);
	return run ? run.world : null;
}
function metaOf(api: QudApi, id: string, final: boolean, data: string): ReplayMeta {
	const m = JSON.parse(api.meta());
	const d = JSON.parse(data);
	return { id, seed: m.seed, name: m.name, species: m.species, calling: m.calling, level: m.level, world: m.world, genre: m.genre, dead: !!m.dead,
		saved: Date.now(), keys: d.n ?? 0, highlights: (d.hl ?? []).length, final };
}

/** Autosave: store the live replay (one per run). Call right after saveRun so the world copy is available. */
export async function replaySaveLive(api: QudApi, seed: string, world?: string) {
	if (!enabled || !hasReplayApi(api)) return;
	try {
		const data = api.replay();
		if (!data) return;
		await put(metaOf(api, 'live:' + seed, false, data), data, await ensureWorld(seed, world), clips);
	} catch {
		/* replay storage is best effort */
	}
}

/** The run ended (death or retire): file the replay under the hall entry id and drop the live one. Call before deleteRun. */
export async function replayFinish(api: QudApi, seed: string, hallId: string) {
	if (!enabled || !hasReplayApi(api)) return;
	try {
		const data = api.replay();
		if (!data) return;
		const w = await ensureWorld(seed);
		await put(metaOf(api, hallId, true, data), data, w, clips);
		const db = await open();
		const tx = db.transaction(['meta', 'data', 'clips'], 'readwrite');
		for (const st of ['meta', 'data', 'clips']) tx.objectStore(st).delete('live:' + seed);
		await done(tx);
	} catch {
		/* best effort */
	}
}

/** After api.load(save) on resume: hand the stored live log back so one replay covers the whole run. */
export async function replayResume(api: QudApi, seed: string) {
	replayReset();
	if (!hasReplayApi(api)) return;
	try {
		const r = await loadReplay('live:' + seed);
		if (!r) return;
		const err = api.replayRestore(r.data);
		if (!err) loadClipsFrom(r.clips);
	} catch {
		/* the engine starts a base-save log instead */
	}
}

// ---- export / import ----
export async function exportReplayBytes(id: string): Promise<Uint8Array | null> {
	const r = await loadReplay(id);
	if (!r) return null;
	return deflateSync(strToU8(JSON.stringify({ rfreplay: 1, meta: r.meta, data: r.data, world: r.world, clips: r.clips })), { level: 9 });
}
export async function downloadReplay(id: string) {
	const b = await exportReplayBytes(id);
	if (!b) return;
	const m = (await listReplays()).find((x) => x.id === id);
	const url = URL.createObjectURL(new Blob([b as BlobPart], { type: 'application/octet-stream' }));
	const a = document.createElement('a');
	a.href = url;
	a.download = `${(m?.name ?? 'replay').replace(/[^\w-]+/g, '_')}.rfreplay`;
	a.click();
	setTimeout(() => URL.revokeObjectURL(url), 1000);
}
export async function importReplay(file: File): Promise<ReplayMeta> {
	const j = JSON.parse(strFromU8(inflateSync(new Uint8Array(await file.arrayBuffer()))));
	if (j?.rfreplay !== 1 || !j.meta || !j.data || !j.world) throw new Error('not a Wanderer replay file');
	const meta: ReplayMeta = { ...j.meta, id: 'imp:' + j.meta.seed + ':' + Date.now(), final: true, saved: Date.now() };
	await put(meta, j.data, j.world, j.clips ?? []);
	return meta;
}

// ---- player ----
export interface PlayerState {
	frame: QudFrame | null;
	info: ReplayInfo | null;
}
export type Speed = 1 | 4 | 16;
const KEYS_PER_SEC: Record<Speed, number> = { 1: 4, 4: 16, 16: 64 };

export interface ReplayPlayer {
	state(): PlayerState;
	setSpeed(s: Speed): void;
	play(): void;
	pause(): void;
	playing(): boolean;
	step(): void;
	nextHighlight(): void;
	prevHighlight(): void;
	seek(n: number): void;
	restart(): void;
	close(): void;
}

/** Opens a fresh engine game from the stored world + spec and feeds the keys. onchange fires after every update. */
export async function openPlayer(api: QudApi, id: string, onchange: (s: PlayerState) => void): Promise<ReplayPlayer> {
	if (!hasReplayApi(api)) throw new Error('this engine build has no replay support (rebuild the wasm)');
	const r = await loadReplay(id);
	if (!r) throw new Error('replay data is missing');
	const first = api.replayOpen(r.world, r.data);
	if ('error' in first && first.error) throw new Error(first.error);
	let st: PlayerState = { frame: first as QudFrame, info: parseInfo(first as RpFrame) };
	let speed: Speed = 4;
	let timer: ReturnType<typeof setInterval> | null = null;
	let acc = 0;
	const emit = (f: RpFrame | null) => {
		if (!f) return;
		st = { frame: f, info: parseInfo(f) ?? st.info };
		onchange(st);
		if (st.info && (st.info.done || st.info.diverged)) stop();
	};
	const stop = () => {
		if (timer) clearInterval(timer);
		timer = null;
	};
	const tick = () => {
		acc += KEYS_PER_SEC[speed] / 20;
		const n = Math.floor(acc);
		if (n > 0) {
			acc -= n;
			emit(api.replayStep(n));
		}
	};
	return {
		state: () => st,
		setSpeed(s) {
			speed = s;
		},
		play() {
			if (timer || st.info?.done || st.info?.diverged) return;
			timer = setInterval(tick, 50);
			onchange(st);
		},
		pause() {
			stop();
			onchange(st);
		},
		playing: () => !!timer,
		step() {
			stop();
			emit(api.replayStep(1));
		},
		nextHighlight() {
			emit(api.replayHighlight(1));
		},
		prevHighlight() {
			emit(api.replayHighlight(-1));
		},
		seek(n) {
			emit(api.replayTo(n));
		},
		restart() {
			stop();
			emit(api.replayRestart());
		},
		close() {
			stop();
			api.replayClose();
		}
	};
}
function parseInfo(f: RpFrame): ReplayInfo | null {
	if (!f.info) return null;
	try {
		return typeof f.info === 'string' ? JSON.parse(f.info) : f.info;
	} catch {
		return null;
	}
}

/** Palette from the engine as CSS colours. */
export function replayPalette(api: QudApi): string[] {
	try {
		return JSON.parse(api.palette());
	} catch {
		return [];
	}
}
