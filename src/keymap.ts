/**
 * Key remapping layer. The shell normalises a DOM KeyboardEvent into an engine key name ('i', 'ArrowUp', 'N7', 'F4'…)
 * before `qud.key(k)`; `mapKey(e, mode)` does that normalisation and applies the player's bindings on the way.
 *
 * Model
 *  - An ACTION (ACTIONS below) has one fixed engine key (`key`): the key the engine already understands for it (taken from
 *    engine/palette_data.go `paEntries` and engine/input.go `keyMain`). The engine itself is never changed.
 *  - A BINDING is a list of PHYSICAL key names for an action. Physical name = `KeyboardEvent.key` ('i', 'I', 'ArrowUp', 'F4', ' '),
 *    except numpad digits, which are `e.code` ('Numpad1'..'Numpad9') so they stay distinct from the top row.
 *  - The pref `keymap` holds only the overrides: { actionId: [physical…] }. {} = stock keys, and then mapKey is exactly the old
 *    `NUMPAD[e.code] ?? e.key` behaviour. An override REPLACES the action's default physical keys ([] = unbound).
 *  - A default physical key whose action was overridden and that nobody else claims is swallowed (mapKey returns null), so
 *    moving Inventory off 'i' does not leave 'i' opening it anyway.
 *
 * Modes. Remapping only runs on map-play screens (REMAP_MODES: main, target, look, dir, world). Every other engine mode
 * bypasses it and gets the raw key, so menu letters (talk options, shop rows, number picks) never move. Text entry therefore
 * bypasses automatically; the modes that read typed text, for the record (TEXT_MODES): cn (cheat console), pal (palette),
 * wpname (place naming), tmname (pet/beast naming), pq (pet screens incl. rename), wed (world editor prompts), ggf (forge
 * naming), plus the character creator which is a Svelte screen and never goes through Play's key handler.
 */

export type Keymap = Record<string, string[]>;
export type Category = 'movement' | 'actions' | 'screens' | 'combat' | 'system';

export interface ActionDef {
	id: string;
	name: string;
	cat: Category;
	/** engine key sent to qud.key */
	key: string;
	/** default physical keys */
	defaults: string[];
	/** shown but not rebindable */
	fixed?: boolean;
}

export const CATEGORIES: [Category, string][] = [
	['movement', 'Movement'],
	['actions', 'Actions'],
	['screens', 'Screens'],
	['combat', 'Combat'],
	['system', 'System'],
];

/** screens where the map keys apply and remapping runs; anything else gets the raw key */
export const REMAP_MODES = new Set(['main', 'target', 'look', 'dir', 'world']);
/** engine modes that read typed text (all bypass remapping because they are not in REMAP_MODES) */
export const TEXT_MODES = ['cn', 'pal', 'wpname', 'tmname', 'pq', 'wed', 'ggf'];

function a(id: string, name: string, cat: Category, key: string, defaults?: string[]): ActionDef {
	return { id, name, cat, key, defaults: defaults ?? [key] };
}

export const ACTIONS: ActionDef[] = [
	// movement
	a('mN', 'Move north', 'movement', 'ArrowUp', ['ArrowUp', 'Numpad8']),
	a('mS', 'Move south', 'movement', 'ArrowDown', ['ArrowDown', 'Numpad2']),
	a('mW', 'Move west', 'movement', 'ArrowLeft', ['ArrowLeft', 'Numpad4']),
	a('mE', 'Move east', 'movement', 'ArrowRight', ['ArrowRight', 'Numpad6']),
	a('mNW', 'Move northwest', 'movement', 'N7', ['Numpad7', 'Home']),
	a('mNE', 'Move northeast', 'movement', 'N9', ['Numpad9', 'PageUp']),
	a('mSW', 'Move southwest', 'movement', 'N1', ['Numpad1', 'End']),
	a('mSE', 'Move southeast', 'movement', 'N3', ['Numpad3', 'PageDown']),
	a('wait', 'Wait one turn', 'movement', '.', ['.', 'Numpad5']),
	a('explore', 'Auto-explore', 'movement', 'v', ['v', 'Tab']),
	a('up', 'Stairs up / world map', 'movement', '<'),
	a('down', 'Stairs down', 'movement', '>'),
	a('world', 'World map', 'movement', 'w', ['w', 'M']),
	// actions
	a('pickup', 'Pick up', 'actions', 'g', ['g', ',']),
	a('look', 'Look', 'actions', 'x', ['x', 'L']),
	a('search', 'Search', 'actions', 's'),
	a('eat', 'Eat', 'actions', 'e'),
	a('drink', 'Drink', 'actions', 'd'),
	a('rest', 'Rest until healed', 'actions', 'R'),
	a('restwhile', 'Rest a while', 'actions', 'z'),
	a('cook', 'Cook', 'actions', 'C'),
	a('craft', 'Workbench', 'actions', 'T'),
	a('dig', 'Dig', 'actions', 'D'),
	a('home', 'Home / stash', 'actions', 'H'),
	a('gohome', 'Return home', 'actions', 'F3'),
	a('steal', 'Steal', 'actions', 'X'),
	a('companions', 'Companions', 'actions', 'o'),
	a('orders', 'Companion orders', 'actions', 'O'),
	a('war', 'War', 'actions', 'W'),
	// screens
	a('inv', 'Inventory', 'screens', 'i'),
	a('char', 'Character', 'screens', 'c'),
	a('journal', 'Journal', 'screens', 'j'),
	a('rep', 'Reputation', 'screens', 'r'),
	a('log', 'Message log', 'screens', 'm'),
	a('skills', 'Skills', 'screens', 'K'),
	a('mutations', 'Mutations', 'screens', 'V'),
	a('liquids', 'Liquids', 'screens', 'P'),
	a('hunts', 'Hunts', 'screens', 'B'),
	a('news', 'News', 'screens', 'N'),
	a('thoughts', 'Thought cabinet', 'screens', 'J'),
	a('quest', 'Quest spine', 'screens', 'Q'),
	a('ledger', 'Ledger', 'screens', 'U'),
	a('court', 'Court', 'screens', 'A'),
	a('hunters', 'Hunters', 'screens', 'I'),
	a('endgame', 'Endgame war room', 'screens', 'E'),
	a('ship', 'Ship', 'screens', 'Z'),
	a('murim', 'Murim sects', 'screens', 'G'),
	a('brew', 'Brewing', 'screens', 'F4'),
	a('sleep', 'Sleep', 'screens', 'F5'),
	a('trade', 'Trade ledger', 'screens', 'F6'),
	a('family', 'Family', 'screens', 'F7'),
	a('arcane', 'Arcane', 'screens', 'F8'),
	a('collect', 'Collections', 'screens', 'F9'),
	a('journalplus', 'Journal+ (places, achievements)', 'screens', 'F10'),
	a('legacy', 'Legacy', 'screens', 'F12'),
	a('palette', 'Command palette', 'screens', ';'),
	// combat
	a('fire', 'Fire (auto-aim)', 'combat', 'f'),
	a('aim', 'Aim by hand', 'combat', 'F'),
	a('throw', 'Throw', 'combat', 't'),
	a('abilities', 'Abilities / powers', 'combat', 'a'),
	// system
	a('save', 'Save', 'system', 'S'),
	a('help', 'Help', 'system', '?'),
	a('console', 'Cheat console', 'system', '/'),
	a('sandbox', 'Sandbox', 'system', 'F2'),
	{ id: 'settings', name: 'Settings', cat: 'system', key: '`', defaults: ['`'], fixed: true },
	{ id: 'quit', name: 'Save & quit / back', cat: 'system', key: 'Escape', defaults: ['Escape'], fixed: true },
];

const ACTION_LIST = ACTIONS;
const BY_ID = new Map(ACTION_LIST.map((x) => [x.id, x]));
export function actionById(id: string): ActionDef | undefined {
	return BY_ID.get(id);
}

/** physical keys that cannot be bound (they would lock the player out) */
const RESERVED = new Set(['Escape', '`', 'Shift', 'Control', 'Alt', 'Meta', 'CapsLock', 'Dead', 'Unidentified', 'NumLock', 'ScrollLock', 'ContextMenu']);
export function isBindable(phys: string): boolean {
	return !!phys && !RESERVED.has(phys);
}

const NUMPAD: Record<string, string> = { Numpad1: 'N1', Numpad2: 'N2', Numpad3: 'N3', Numpad4: 'N4', Numpad5: 'N5', Numpad6: 'N6', Numpad7: 'N7', Numpad8: 'N8', Numpad9: 'N9' };

/** physical name used in bindings */
export function physKey(e: { key: string; code: string }): string {
	return NUMPAD[e.code] ? e.code : e.key;
}

const LABELS: Record<string, string> = {
	' ': 'Space', ArrowUp: '↑', ArrowDown: '↓', ArrowLeft: '←', ArrowRight: '→',
	Numpad1: 'Num1', Numpad2: 'Num2', Numpad3: 'Num3', Numpad4: 'Num4', Numpad5: 'Num5', Numpad6: 'Num6', Numpad7: 'Num7', Numpad8: 'Num8', Numpad9: 'Num9',
	PageUp: 'PgUp', PageDown: 'PgDn',
};
export function keyLabel(phys: string): string {
	return LABELS[phys] ?? phys;
}

// ---------- state ----------

interface Compiled {
	active: boolean;
	map: Map<string, string>; // physical -> engine key
	swallow: Set<string>;
}
let cur: Compiled = { active: false, map: new Map(), swallow: new Set() };
let curSrc = '{}';

/** effective physical keys of an action under an override map */
export function keysFor(km: Keymap, id: string): string[] {
	const d = BY_ID.get(id);
	if (!d) return [];
	if (d.fixed) return d.defaults;
	const o = km[id];
	return Array.isArray(o) ? o : d.defaults;
}

/** physical key -> action ids that use it (more than one = conflict) */
export function usage(km: Keymap): Map<string, string[]> {
	const m = new Map<string, string[]>();
	for (const d of ACTION_LIST) {
		for (const k of keysFor(km, d.id)) {
			const l = m.get(k);
			if (l) l.push(d.id);
			else m.set(k, [d.id]);
		}
	}
	return m;
}

export interface Conflict {
	key: string;
	actions: string[];
}
export function findConflicts(km: Keymap): Conflict[] {
	const out: Conflict[] = [];
	for (const [key, actions] of usage(km)) if (actions.length > 1) out.push({ key, actions });
	return out;
}

function compile(km: Keymap): Compiled {
	const map = new Map<string, string>();
	const swallow = new Set<string>();
	let overridden = false;
	for (const d of ACTION_LIST) {
		if (d.fixed) continue;
		const o = km[d.id];
		if (!Array.isArray(o)) continue;
		overridden = true;
		for (const k of d.defaults) swallow.add(k);
	}
	if (!overridden) return { active: false, map, swallow };
	// first action in table order wins a conflicted key
	for (const d of ACTION_LIST) {
		if (d.fixed) continue;
		for (const k of keysFor(km, d.id)) {
			if (!map.has(k)) map.set(k, d.key);
			swallow.delete(k);
		}
	}
	return { active: true, map, swallow };
}

/** install the player's overrides (call whenever the `keymap` pref changes) */
export function setKeymap(km: Keymap | undefined | null): void {
	const clean = cleanKeymap(km);
	const src = JSON.stringify(clean);
	if (src === curSrc) return;
	curSrc = src;
	cur = compile(clean);
}

/**
 * The one entry point for Play.svelte: DOM key event -> engine key, or null to drop the event.
 * `mode` is `frame.mode`; outside REMAP_MODES the raw key is returned untouched.
 */
export function mapKey(e: KeyboardEvent, mode?: string): string | null {
	const raw = NUMPAD[e.code] ?? e.key;
	if (!cur.active || (mode !== undefined && !REMAP_MODES.has(mode))) return raw;
	const p = physKey(e);
	const hit = cur.map.get(p);
	if (hit !== undefined) return hit;
	if (cur.swallow.has(p)) return null;
	return raw;
}

// ---------- editing helpers (pure: return a new keymap) ----------

function norm(km: Keymap): Keymap {
	const out: Keymap = {};
	for (const d of ACTION_LIST) {
		if (d.fixed) continue;
		const o = km[d.id];
		if (!Array.isArray(o)) continue;
		const same = o.length === d.defaults.length && o.every((k) => d.defaults.includes(k));
		if (!same) out[d.id] = o;
	}
	return out;
}

/** who else holds this key (excluding `except`) */
export function holders(km: Keymap, key: string, except?: string): string[] {
	return (usage(km).get(key) ?? []).filter((id) => id !== except);
}

/**
 * Bind `key` to `id`. mode 'add' keeps the action's other keys, 'set' replaces them. Other actions holding the key lose it
 * (the caller shows a conflict warning first and only calls this once the player agrees).
 */
export function bindKey(km: Keymap, id: string, key: string, mode: 'add' | 'set'): Keymap {
	const d = BY_ID.get(id);
	if (!d || d.fixed || !isBindable(key)) return km;
	const next: Keymap = { ...km };
	for (const other of holders(km, key, id)) {
		const od = BY_ID.get(other);
		if (od && !od.fixed) next[other] = keysFor(km, other).filter((k) => k !== key);
	}
	const mine = keysFor(next, id);
	next[id] = mode === 'set' ? [key] : mine.includes(key) ? mine : [...mine, key];
	return norm(next);
}

export function unbindKey(km: Keymap, id: string, key: string): Keymap {
	const d = BY_ID.get(id);
	if (!d || d.fixed) return km;
	return norm({ ...km, [id]: keysFor(km, id).filter((k) => k !== key) });
}

export function resetAction(km: Keymap, id: string): Keymap {
	const next = { ...km };
	delete next[id];
	return next;
}

export function resetAll(): Keymap {
	return {};
}

// ---------- presets ----------

export interface Preset {
	id: string;
	name: string;
	note: string;
	map: Keymap;
}

export const PRESETS: Preset[] = [
	{ id: 'default', name: 'Default', note: 'Arrows or numpad to move, mnemonic letters.', map: {} },
	{
		id: 'vi',
		name: 'Vi keys',
		note: 'h j k l y u b n move; Journal moves to q.',
		map: {
			mW: ['h', 'ArrowLeft', 'Numpad4'],
			mS: ['j', 'ArrowDown', 'Numpad2'],
			mN: ['k', 'ArrowUp', 'Numpad8'],
			mE: ['l', 'ArrowRight', 'Numpad6'],
			mNW: ['y', 'Numpad7', 'Home'],
			mNE: ['u', 'Numpad9', 'PageUp'],
			mSW: ['b', 'Numpad1', 'End'],
			mSE: ['n', 'Numpad3', 'PageDown'],
			journal: ['q'],
		},
	},
	{
		id: 'wasd',
		name: 'WASD (laptop, no numpad)',
		note: 'WASD + Q E Z C move; Abilities 1, Search 2, Eat 3, Drink 4, Rest a while 5, Character v, World map M.',
		map: {
			mN: ['w', 'ArrowUp'],
			mW: ['a', 'ArrowLeft'],
			mS: ['s', 'ArrowDown'],
			mE: ['d', 'ArrowRight'],
			mNW: ['q', 'Home'],
			mNE: ['e', 'PageUp'],
			mSW: ['z', 'End'],
			mSE: ['c', 'PageDown'],
			world: ['M'],
			explore: ['Tab'],
			abilities: ['1'],
			search: ['2'],
			eat: ['3'],
			drink: ['4'],
			restwhile: ['5'],
			char: ['v'],
		},
	},
	{
		id: 'left',
		name: 'Left-handed (IJKL)',
		note: 'i j k l u o m . move on the right hand; Inventory e, Journal q, Companions p, Log n, Wait h, Eat 3.',
		map: {
			mN: ['i', 'ArrowUp'],
			mW: ['j', 'ArrowLeft'],
			mS: ['k', 'ArrowDown'],
			mE: ['l', 'ArrowRight'],
			mNW: ['u', 'Home'],
			mNE: ['o', 'PageUp'],
			mSW: ['m', 'End'],
			mSE: ['.', 'PageDown'],
			wait: ['h', 'Numpad5'],
			inv: ['e'],
			eat: ['3'],
			journal: ['q'],
			companions: ['p'],
			log: ['n'],
		},
	},
];

export function applyPreset(id: string): Keymap {
	const p = PRESETS.find((x) => x.id === id);
	return p ? norm(structuredCloneMap(p.map)) : {};
}
function structuredCloneMap(m: Keymap): Keymap {
	const o: Keymap = {};
	for (const k of Object.keys(m)) o[k] = [...m[k]];
	return o;
}

/** the preset the keymap equals, or '' for a custom one */
export function presetOf(km: Keymap): string {
	const s = JSON.stringify(sortKm(norm(km)));
	for (const p of PRESETS) if (JSON.stringify(sortKm(norm(p.map))) === s) return p.id;
	return '';
}
function sortKm(km: Keymap): [string, string[]][] {
	return Object.keys(km).sort().map((k) => [k, [...km[k]].sort()]);
}

// ---------- validation, import / export ----------

/** keep only known actions with sane string arrays; unknown junk is dropped */
export function cleanKeymap(raw: unknown): Keymap {
	const out: Keymap = {};
	if (typeof raw !== 'object' || raw === null || Array.isArray(raw)) return out;
	for (const [id, v] of Object.entries(raw as Record<string, unknown>)) {
		const d = BY_ID.get(id);
		if (!d || d.fixed || !Array.isArray(v)) continue;
		const keys: string[] = [];
		for (const k of v) if (typeof k === 'string' && k.length > 0 && k.length <= 12 && isBindable(k) && !keys.includes(k)) keys.push(k);
		out[id] = keys.slice(0, 6);
	}
	return norm(out);
}

export function exportKeymap(km: Keymap): string {
	return JSON.stringify({ wanderer: 'keymap', version: 1, keymap: norm(km) }, null, 2);
}

export function importKeymap(text: string): { ok: true; keymap: Keymap; conflicts: Conflict[] } | { ok: false; error: string } {
	let data: unknown;
	try {
		data = JSON.parse(text);
	} catch {
		return { ok: false, error: 'That is not valid JSON.' };
	}
	const r = data as { keymap?: unknown };
	const body = typeof data === 'object' && data !== null && 'keymap' in (data as object) ? r.keymap : data;
	if (typeof body !== 'object' || body === null || Array.isArray(body)) return { ok: false, error: 'Expected an object of action -> keys.' };
	const km = cleanKeymap(body);
	if (!Object.keys(km).length && Object.keys(body as object).length) return { ok: false, error: 'No known actions found in that text.' };
	return { ok: true, keymap: km, conflicts: findConflicts(km) };
}
