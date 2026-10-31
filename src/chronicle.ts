/**
 * The hero's deeds, kept per world seed in their own IndexedDB (survives the run's death), plus the
 * chronicle export that Realmforge can import. recordDeeds(api, seed) is the single hook for Play.svelte.
 */
import type { QudApi } from './qud';
import { postDeeds } from './link';

export interface Deed {
	kind: string;
	text: string;
	monster?: number;
	artifact?: number;
	cell: number;
	year: number;
}
export interface Chronicle {
	seed: string;
	world: string;
	hero: Record<string, unknown>;
	turns: number;
	deeds: Deed[];
	saved: number;
}

const DB = 'realmforge-wanderer-chronicle';

function open(): Promise<IDBDatabase> {
	return new Promise((res, rej) => {
		const r = indexedDB.open(DB, 1);
		r.onupgradeneeded = () => r.result.createObjectStore('chronicle');
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

export async function getChronicle(seed: string): Promise<Chronicle | null> {
	try {
		const db = await open();
		return (await req<Chronicle | undefined>(db.transaction('chronicle').objectStore('chronicle').get(seed))) ?? null;
	} catch {
		return null;
	}
}
export async function listChronicles(): Promise<Chronicle[]> {
	try {
		const db = await open();
		const all = await req<Chronicle[]>(db.transaction('chronicle').objectStore('chronicle').getAll());
		return all.sort((a, b) => b.saved - a.saved);
	} catch {
		return [];
	}
}

/** engine turn counter (game minutes) read from the save JSON without parsing it */
function turnsOf(api: QudApi): number {
	try {
		const m = /"turn":(\d+)/.exec(api.save());
		return m ? Number(m[1]) : 0;
	} catch {
		return 0;
	}
}

// writes are serialised so concurrent drains never lose deeds
let chain: Promise<unknown> = Promise.resolve();

/**
 * Drain the engine's pending deeds, append them to the stored chronicle for this seed, and (when
 * Wanderer was opened by Realmforge) push them to it live. Safe to call often.
 */
export function recordDeeds(api: QudApi, seed?: string): Promise<Deed[]> {
	let fresh: Deed[] = [];
	let meta: Record<string, unknown> = {};
	try {
		fresh = JSON.parse(api.deeds() || '[]');
		meta = JSON.parse(api.meta());
	} catch {
		return Promise.resolve([]);
	}
	const key = seed || String(meta.seed ?? '');
	if (!key) return Promise.resolve([]);
	if (!fresh.length) return Promise.resolve([]);
	const turns = turnsOf(api);
	const job = chain.then(async () => {
		const cur = (await getChronicle(key)) ?? { seed: key, world: String(meta.world ?? ''), hero: meta, turns: 0, deeds: [], saved: 0 };
		cur.deeds.push(...fresh);
		cur.hero = meta;
		cur.world = String(meta.world ?? cur.world);
		cur.turns = Math.max(cur.turns, turns);
		cur.saved = Date.now();
		try {
			const db = await open();
			const tx = db.transaction('chronicle', 'readwrite');
			tx.objectStore('chronicle').put(cur, key);
			await new Promise<void>((res) => (tx.oncomplete = tx.onerror = () => res()));
		} catch {
			/* storage unavailable: the live link still works */
		}
		postDeeds(key, fresh, cur.turns);
		return fresh;
	});
	chain = job.catch(() => {});
	return job;
}

export function exportChronicle(c: Chronicle) {
	const body = JSON.stringify({ seed: c.seed, world: c.world, hero: c.hero, turns: c.turns, deeds: c.deeds }, null, 1);
	const slug = (c.world || 'world').toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/^-|-$/g, '');
	const a = document.createElement('a');
	a.href = URL.createObjectURL(new Blob([body], { type: 'application/json' }));
	a.download = `${slug}-wanderer-chronicle.json`;
	a.click();
	setTimeout(() => URL.revokeObjectURL(a.href), 5000);
}
