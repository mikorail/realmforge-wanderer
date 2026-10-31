/**
 * Runs live in IndexedDB, deflated: the world file a run was started on plus the engine's save.
 * One run per world seed. Meta is kept beside them so the title screen can list runs cheaply.
 */
import { deflateSync, inflateSync, strToU8, strFromU8 } from 'fflate';

const DB = 'realmforge-wanderer';

export interface RunMeta {
	seed: string;
	name: string;
	level: number;
	calling: string;
	species: string;
	world: string;
	genre: string;
	year: number;
	place: string;
	dead: boolean;
	saved: number;
}

function open(): Promise<IDBDatabase> {
	return new Promise((res, rej) => {
		const r = indexedDB.open(DB, 1);
		r.onupgradeneeded = () => {
			r.result.createObjectStore('meta');
			r.result.createObjectStore('save');
			r.result.createObjectStore('world');
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

const pack = (s: string) => deflateSync(strToU8(s), { level: 6 });
const unpack = (b: Uint8Array) => strFromU8(inflateSync(b));

export async function saveRun(meta: RunMeta, save: string, world?: string) {
	if (!save) return;
	const db = await open();
	const tx = db.transaction(['meta', 'save', 'world'], 'readwrite');
	tx.objectStore('meta').put(meta, meta.seed);
	tx.objectStore('save').put(pack(save), meta.seed);
	if (world) tx.objectStore('world').put(pack(world), meta.seed);
	await new Promise<void>((res, rej) => {
		tx.oncomplete = () => res();
		tx.onerror = () => rej(tx.error);
	});
}

export async function listRuns(): Promise<RunMeta[]> {
	try {
		const db = await open();
		const all = await req<RunMeta[]>(db.transaction('meta').objectStore('meta').getAll());
		return all.sort((a, b) => b.saved - a.saved);
	} catch {
		return [];
	}
}

export async function loadRun(seed: string): Promise<{ save: string; world: string } | null> {
	const db = await open();
	const tx = db.transaction(['save', 'world']);
	const [s, w] = await Promise.all([req<Uint8Array | undefined>(tx.objectStore('save').get(seed)), req<Uint8Array | undefined>(tx.objectStore('world').get(seed))]);
	return s && w ? { save: unpack(s), world: unpack(w) } : null;
}

export async function deleteRun(seed: string) {
	const db = await open();
	const tx = db.transaction(['meta', 'save', 'world'], 'readwrite');
	for (const st of ['meta', 'save', 'world']) tx.objectStore(st).delete(seed);
	await new Promise<void>((res) => (tx.oncomplete = () => res()));
}
