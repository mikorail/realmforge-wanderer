/** Context-menu helpers: the engine's menu rows and thin wrappers over window.qud.ctx / ctxDo / ctxAt. */
import type { QudFrame } from './qud';

export interface CtxItem {
	id: string;
	label: string;
	hint?: string;
	tone?: 'hostile' | 'warn' | '';
}

interface CtxApi {
	ctx?(x: number, y: number): string;
	ctxDo?(id: string, x: number, y: number): QudFrame | { error: string } | null;
	ctxAt?(): string;
}

function api(): CtxApi {
	return (window.qud ?? {}) as CtxApi;
}

export function ctxItems(x: number, y: number): CtxItem[] {
	try {
		const s = api().ctx?.(x, y);
		return s ? (JSON.parse(s) as CtxItem[]) : [];
	} catch {
		return [];
	}
}

export function ctxRun(id: string, x: number, y: number): QudFrame | null {
	const f = api().ctxDo?.(id, x, y);
	return f && !('error' in f) ? f : null;
}

/** keyboard route: the look cursor (else the player) as a map cell */
export function ctxCursor(): { x: number; y: number } | null {
	try {
		const s = api().ctxAt?.();
		if (!s) return null;
		const [x, y] = s.split(',').map(Number);
		return Number.isFinite(x) && Number.isFinite(y) ? { x, y } : null;
	} catch {
		return null;
	}
}

/** keys that open the menu at the cursor: the Menu key, Shift+F10, or `a` while the look cursor is up */
export function isMenuKey(e: KeyboardEvent, mode: string): boolean {
	if (e.key === 'ContextMenu' || (e.key === 'F10' && e.shiftKey)) return true;
	return mode === 'look' && (e.key === 'a' || e.key === 'A') && !e.metaKey && !e.ctrlKey && !e.altKey;
}
