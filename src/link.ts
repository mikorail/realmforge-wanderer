/**
 * Live link with Realmforge. Realmforge opens Wanderer with ?link=1; Wanderer posts {type:'wanderer-ready'}
 * to window.opener and gets {type:'world', json} back. Deeds flow the other way as {type:'deeds', deeds, turns}.
 */
import type { QudApi } from './qud';
import { recordDeeds, type Deed } from './chronicle';

/** Realmforge origin(s) we talk to; hostnames may be localhost or 127.0.0.1 (same as this page). */
export const REALMFORGE_PORT = '5180';
export const isLinked = () => new URLSearchParams(location.search).get('link') === '1' && !!window.opener;

export function allowedOrigin(origin: string): boolean {
	try {
		const u = new URL(origin);
		return u.port === REALMFORGE_PORT && u.hostname === location.hostname && (u.hostname === 'localhost' || u.hostname === '127.0.0.1');
	} catch {
		return false;
	}
}

let target = '';
/** Tell the opener we are ready, then resolve with the world JSON it sends (and remember its origin). */
export function waitForWorld(onworld: (json: string) => void): () => void {
	const handler = (e: MessageEvent) => {
		if (e.source !== window.opener || !allowedOrigin(e.origin)) return;
		const d = e.data;
		if (d && d.type === 'world' && typeof d.json === 'string') {
			target = e.origin;
			onworld(d.json);
		}
	};
	window.addEventListener('message', handler);
	const origins = [`http://${location.hostname}:${REALMFORGE_PORT}`];
	const hello = () => origins.forEach((o) => window.opener?.postMessage({ type: 'wanderer-ready' }, o));
	hello();
	const t = setInterval(() => (target ? clearInterval(t) : hello()), 1500);
	return () => {
		clearInterval(t);
		window.removeEventListener('message', handler);
	};
}

export function postDeeds(seed: string, deeds: Deed[], turns: number) {
	if (!target || !window.opener || window.opener.closed) return;
	try {
		window.opener.postMessage({ type: 'deeds', seed, deeds, turns }, target);
	} catch {
		/* opener gone */
	}
}

/** Hook for Play.svelte: drain the engine's deeds and push them to Realmforge. Same as recordDeeds. */
export function sendDeeds(api: QudApi): Promise<Deed[]> {
	return recordDeeds(api);
}
