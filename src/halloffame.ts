/**
 * Hall of fame (IndexedDB store `hall` of realmforge-wanderer) and deterministic daily / shared seeds.
 * Everything stays in the browser.
 */
import type { QudApi } from './qud';
import { open } from './store';
import { loadSaga, type SagaData } from './saga';

export interface HallEntry {
	id: string;
	name: string;
	species: string;
	calling: string;
	world: string;
	worldSeed: string;
	genre: string;
	planet: string;
	level: number;
	yearStart: number;
	yearEnd: number;
	days: number;
	km: number;
	kills: number;
	deedCount: number;
	goldPeak: number;
	deeds: string[];
	family: string;
	cause: string;
	retired: boolean;
	scenario: string;
	mode: string;
	score: number;
	place: string;
	/** daily / shared seed text when the life began from one */
	seedText: string;
	date: number;
	/** the life's saga (engine/saga.go), written when the life ended */
	saga?: SagaData | null;
}

/** seed text for the run being played, set by the title screen when a seeded life begins */
let pendingSeed = '';
export const setPendingSeed = (s: string) => (pendingSeed = s);
export const takePendingSeed = () => pendingSeed;

function req<T>(r: IDBRequest<T>): Promise<T> {
	return new Promise((res, rej) => {
		r.onsuccess = () => res(r.result);
		r.onerror = () => rej(r.error);
	});
}

export async function listHall(): Promise<HallEntry[]> {
	try {
		const db = await open();
		const all = await req<HallEntry[]>(db.transaction('hall').objectStore('hall').getAll());
		return all.sort((a, b) => b.score - a.score || b.date - a.date);
	} catch {
		return [];
	}
}

export async function deleteHall(id: string) {
	const db = await open();
	const tx = db.transaction('hall', 'readwrite');
	tx.objectStore('hall').delete(id);
	await new Promise<void>((res) => (tx.oncomplete = tx.onerror = () => res()));
}

/** Record the current run's life summary. `retired` ends it honourably instead of by death. */
export async function recordHall(api: QudApi, id: string, retired: boolean, seedText: string): Promise<HallEntry | null> {
	try {
		const meta = JSON.parse(api.meta());
		const h = JSON.parse(api.hall());
		const e: HallEntry = {
			id,
			name: meta.name,
			species: meta.species,
			calling: meta.calling,
			world: meta.world,
			worldSeed: String(meta.seed),
			genre: meta.genre,
			planet: h.planet || meta.planet || '',
			level: meta.level,
			yearStart: h.yearStart ?? meta.year,
			yearEnd: h.yearEnd ?? meta.year,
			days: h.days ?? 0,
			km: h.km ?? 0,
			kills: h.kills ?? 0,
			deedCount: h.deedCount ?? 0,
			goldPeak: h.goldPeak ?? 0,
			deeds: h.deeds ?? [],
			family: h.family ?? '',
			cause: retired ? 'retired' : h.cause || 'died',
			retired,
			scenario: h.scenario ?? '',
			mode: h.mode ?? 'classic',
			score: h.score ?? 0,
			place: meta.place ?? '',
			seedText,
			date: Date.now(),
			saga: loadSaga(api)
		};
		const db = await open();
		const tx = db.transaction('hall', 'readwrite');
		tx.objectStore('hall').put(e, id);
		await new Promise<void>((res, rej) => {
			tx.oncomplete = () => res();
			tx.onerror = () => rej(tx.error);
		});
		return e;
	} catch {
		return null;
	}
}

/** Hand this world's past lives to the engine (call right after newGame / load). */
export async function feedLegends(api: QudApi) {
	try {
		const seed = String(JSON.parse(api.meta()).seed);
		const mine = (await listHall()).filter((e) => e.worldSeed === seed).slice(0, 6);
		api.legends(
			JSON.stringify(
				mine.map((e) => ({ name: e.name, species: e.species, calling: e.calling, year: e.yearEnd, cause: e.cause, place: e.place, level: e.level, score: e.score, deeds: e.deeds.slice(0, 4) }))
			)
		);
	} catch {
		/* the hall is optional */
	}
}

export function exportHall(list: HallEntry[]) {
	const a = document.createElement('a');
	a.href = URL.createObjectURL(new Blob([JSON.stringify(list, null, 1)], { type: 'application/json' }));
	a.download = 'wanderer-hall-of-fame.json';
	a.click();
	setTimeout(() => URL.revokeObjectURL(a.href), 5000);
}

export function shareText(e: HallEntry): string {
	const code = e.seedText ? `Seed ${e.seedText} · ` : '';
	const base = `Wanderer: ${code}${e.name}, level ${e.level} ${e.species} ${e.calling}, ${e.world} · score ${e.score} (${e.retired ? 'retired' : 'fell'} after ${e.days} days)`;
	if (!e.saga) return base;
	const ep = e.saga.epitaph.split('\n').pop()?.replace(/^"|"$/g, '') ?? '';
	return `${base}\n${e.saga.title}${ep ? `: "${ep}"` : ''}`;
}

// ---- daily / shared seeds ----

export function todaySeed(): string {
	const d = new Date();
	return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`;
}

function hash(s: string): number {
	let h = 2166136261 >>> 0;
	for (let i = 0; i < s.length; i++) {
		h ^= s.charCodeAt(i);
		h = Math.imul(h, 16777619) >>> 0;
	}
	return h;
}
/** mulberry32 */
export function seedRng(s: string): () => number {
	let a = hash(s) || 1;
	return () => {
		a = (a + 0x6d2b79f5) >>> 0;
		let t = a;
		t = Math.imul(t ^ (t >>> 15), t | 1);
		t ^= t + Math.imul(t ^ (t >>> 7), t | 61);
		return ((t ^ (t >>> 14)) >>> 0) / 4294967296;
	};
}

/** which bundled sample world a seed picks */
export function seedWorldIndex(seed: string, n: number): number {
	return Math.floor(seedRng(seed + ':world')() * Math.max(n, 1));
}

const SYL = ['ka', 'lo', 'mi', 'ren', 'tha', 'vo', 'su', 'dar', 'el', 'no', 'bri', 'ya', 'gor', 'ixa', 'pe', 'qu'];

/** Deterministic character from a seed text and the world's chargen options (mirrors CharGen's point-buy rules). */
export function seedSpec(options: any, seed: string): { spec: Record<string, unknown>; desc: string } {
	const rnd = seedRng(seed + ':char');
	const pick = <T,>(a: T[]): T => a[Math.floor(rnd() * a.length)];
	const species = pick<any>(options.species);
	const calling = pick<any>(options.callings);
	const homes = options.starts.filter((b: any) => b.Species === species.ID);
	const start = pick<any>(homes.length ? homes : options.starts).ID;
	const cost = (v: number) => {
		let c = 0;
		for (let s = 12; s < v; s++) c += s >= 18 ? 2 : 1;
		return c;
	};
	const attr = [12, 12, 12, 12, 12, 12];
	const left = () => options.attrPoints - attr.reduce((t, v) => t + cost(v), 0);
	for (const [k, v] of Object.entries(calling.req ?? {})) {
		const i = +k;
		while (attr[i] + (species.Attr?.[i] ?? 0) < (v as number) && left() > 0) attr[i]++;
	}
	const w = (i: number) => (calling.attr?.[i] ? 3 : 1);
	for (let guard = 0; guard < 400; guard++) {
		const open = attr.map((_, i) => i).filter((i) => attr[i] < 20 && cost(attr[i] + 1) - cost(attr[i]) <= left());
		if (!open.length) break;
		let r = rnd() * open.reduce((t, i) => t + w(i), 0);
		let p = open[0];
		for (const i of open) {
			r -= w(i);
			if (r <= 0) {
				p = i;
				break;
			}
		}
		attr[p]++;
	}
	const pcost = (id: string) => {
		const p = options.powers.find((x: any) => x.id === id);
		return p ? p.cost : 0;
	};
	let powers: Record<string, number> = {};
	let spent = 0;
	const pool = options.powers.filter((p: any) => !p.defect).map((p: any) => ({ p, k: rnd() })).sort((a: any, b: any) => a.k - b.k);
	for (const { p } of pool) {
		if (spent + pcost(p.id) <= options.powerPoints && rnd() < 0.5) {
			powers[p.id] = 1;
			spent += pcost(p.id);
		}
	}
	const scen: any[] = (options.scenarios ?? []).filter((s: any) => s.id !== 'mirror');
	const sp = scen.filter((s) => s.kind === 'power');
	const tw = scen.filter((s) => s.kind === 'twist');
	const parts: string[] = [];
	if (sp.length && rnd() < 0.4) parts.push(pick(sp).id);
	if (tw.length && rnd() < 0.4) parts.push(pick(tw).id);
	const name = (pick(SYL) + pick(SYL) + (rnd() < 0.5 ? pick(SYL) : '')).replace(/^./, (c) => c.toUpperCase());
	const scenario = parts.join('+');
	return {
		spec: { name, species: species.ID, calling: calling.id, attr, powers, start, genotype: 'mutant', morph: '', gameMode: 'classic', scenario },
		desc: `${name}, ${species.Name} ${calling.name}${scenario ? ' · ' + scenario.replace('+', ' + ') : ''}`
	};
}
