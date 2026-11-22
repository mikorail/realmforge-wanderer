/**
 * Shell side of the rich tooltips (engine/tipinfo.go, frame field `tip`) and of the "What can I do here" screen
 * (engine/hereinfo.go, frame field `here`).
 *
 * - `parseTip(json)`: frame.tip / qud.tipAt(x, y) string -> EngineTip. Lines starting "+" are good, "-" bad, "!" warnings.
 * - `hereLabels(here, keymap)`: for every key cell the engine drew in mode "here" (engine key names), the label of the
 *   player's own binding when it differs from what the engine printed, so rebinds show correctly.
 */
import { ACTIONS, keyLabel, keysFor, type Keymap } from './keymap';

export interface EngineTip {
	t: string;
	tone?: 'hostile' | 'friendly' | 'neutral';
	l: string[];
	x?: number;
	y?: number;
}

export type TipLineKind = 'good' | 'bad' | 'warn' | 'plain';

export function parseTip(raw: unknown): EngineTip | null {
	if (!raw) return null;
	try {
		const o = (typeof raw === 'string' ? JSON.parse(raw) : raw) as EngineTip;
		if (!o || typeof o.t !== 'string' || !Array.isArray(o.l)) return null;
		return o;
	} catch {
		return null;
	}
}

export function lineKind(line: string): { kind: TipLineKind; text: string } {
	const c = line[0];
	if (c === '+') return { kind: 'good', text: line.slice(1) };
	if (c === '-') return { kind: 'bad', text: line.slice(1) };
	if (c === '!') return { kind: 'warn', text: line.slice(1) };
	return { kind: 'plain', text: line };
}

export interface HereCell {
	x: number;
	y: number;
	w: number;
	k: string;
}
export interface HereInfo {
	cells: HereCell[];
}

export function parseHere(raw: unknown): HereInfo | null {
	if (!raw) return null;
	try {
		const o = (typeof raw === 'string' ? JSON.parse(raw) : raw) as HereInfo;
		return o && Array.isArray(o.cells) ? o : null;
	} catch {
		return null;
	}
}

/** the physical keys (labels) the player uses for an engine key under their keymap, or null when no action owns it */
export function engineKeyLabel(engineKey: string, km: Keymap): string | null {
	const owners = ACTIONS.filter((a) => a.key === engineKey);
	if (!owners.length) return null;
	const out: string[] = [];
	for (const a of owners) for (const p of keysFor(km, a.id)) if (!out.includes(p)) out.push(p);
	if (!out.length) return '(unbound)';
	return out.slice(0, 2).map(keyLabel).join('/');
}

export interface HereLabel extends HereCell {
	label: string;
}

/** only the cells whose label differs from the engine key (stock bindings need no overlay) */
export function hereLabels(here: HereInfo | null, km: Keymap | undefined): HereLabel[] {
	if (!here || !km || !Object.keys(km).length) return [];
	const out: HereLabel[] = [];
	for (const c of here.cells) {
		const label = engineKeyLabel(c.k, km);
		if (label !== null && label !== keyLabel(c.k) && label !== c.k) out.push({ ...c, label });
	}
	return out;
}
