/**
 * Runs live in IndexedDB, deflated: the world file a run was started on plus the engine's save.
 * One run per world seed. Meta is kept beside them so the title screen can list runs cheaply.
 */
import { deflateSync, inflateSync, gunzipSync, strToU8, strFromU8 } from 'fflate';

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

export function open(): Promise<IDBDatabase> {
	return new Promise((res, rej) => {
		const r = indexedDB.open(DB, 2);
		// v2 only adds the hall store; existing stores and their data are kept
		r.onupgradeneeded = () => {
			for (const st of ['meta', 'save', 'world', 'hall']) if (!r.result.objectStoreNames.contains(st)) r.result.createObjectStore(st);
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

/**
 * New writes are gzip through the browser's CompressionStream (native and off the main thread, so a big autosave
 * does not freeze play). Reads sniff the header: gzip (1f 8b) is streamed, anything else is an old zlib deflate and
 * goes through inflateSync, so earlier saves keep loading. No CompressionStream: fall back to fflate deflate.
 */
const hasStreams = typeof CompressionStream !== 'undefined' && typeof DecompressionStream !== 'undefined';

async function pipe(bytes: Uint8Array, t: { readable: ReadableStream<Uint8Array>; writable: WritableStream<Uint8Array> }): Promise<Uint8Array> {
	const w = t.writable.getWriter();
	void w.write(bytes as never).then(() => w.close());
	return new Uint8Array(await new Response(t.readable).arrayBuffer());
}

async function pack(s: string): Promise<Uint8Array> {
	const raw = strToU8(s);
	if (hasStreams) {
		try {
			return await pipe(raw, new CompressionStream('gzip') as never);
		} catch {
			/* fall through */
		}
	}
	return deflateSync(raw, { level: 6 });
}

async function unpack(b: Uint8Array): Promise<string> {
	if (b.length > 2 && b[0] === 0x1f && b[1] === 0x8b) {
		if (hasStreams) return strFromU8(await pipe(b, new DecompressionStream('gzip') as never));
		return strFromU8(gunzipSync(b));
	}
	return strFromU8(inflateSync(b));
}

export async function saveRun(meta: RunMeta, save: string, world?: string) {
	if (!save) return;
	const db = await open();
	// compress first: an IndexedDB transaction closes if it waits on anything but its own requests
	const [packedSave, packedWorld] = await Promise.all([pack(save), world ? pack(world) : Promise.resolve(null)]);
	const tx = db.transaction(['meta', 'save', 'world'], 'readwrite');
	tx.objectStore('meta').put(meta, meta.seed);
	tx.objectStore('save').put(packedSave, meta.seed);
	if (packedWorld) tx.objectStore('world').put(packedWorld, meta.seed);
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
	return s && w ? { save: await unpack(s), world: await unpack(w) } : null;
}

export async function deleteRun(seed: string) {
	const db = await open();
	const tx = db.transaction(['meta', 'save', 'world'], 'readwrite');
	for (const st of ['meta', 'save', 'world']) tx.objectStore(st).delete(seed);
	await new Promise<void>((res) => (tx.oncomplete = () => res()));
}
