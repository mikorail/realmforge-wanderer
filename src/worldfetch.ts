/**
 * Fetch a bundled world. A production build ships a gzip copy next to each world (`scripts/gzip-worlds.mjs`):
 * try `<file>.gz` first and unpack it in the browser, else fall back to the plain file (dev server, old deploys).
 * `onProgress` gets 0..1 while the body downloads (when the server tells the size).
 */
const base = () => import.meta.env.BASE_URL + 'worlds/';

async function readAll(r: Response, onProgress?: (f: number) => void): Promise<Uint8Array> {
	const total = Number(r.headers.get('content-length')) || 0;
	if (!r.body) return new Uint8Array(await r.arrayBuffer());
	const reader = r.body.getReader();
	const parts: Uint8Array[] = [];
	let got = 0;
	for (;;) {
		const { done, value } = await reader.read();
		if (done) break;
		parts.push(value);
		got += value.length;
		if (total && onProgress) onProgress(Math.min(1, got / total));
	}
	const out = new Uint8Array(got);
	let o = 0;
	for (const p of parts) {
		out.set(p, o);
		o += p.length;
	}
	return out;
}

async function gunzip(bytes: Uint8Array): Promise<string> {
	const stream = new Blob([bytes as BlobPart]).stream().pipeThrough(new DecompressionStream('gzip'));
	return await new Response(stream).text();
}

export async function fetchWorld(file: string, onProgress?: (f: number) => void): Promise<string> {
	if (typeof DecompressionStream !== 'undefined') {
		try {
			const r = await fetch(base() + file + '.gz');
			if (r.ok) {
				const bytes = await readAll(r, onProgress);
				if (bytes.length > 2 && bytes[0] === 0x1f && bytes[1] === 0x8b) return await gunzip(bytes);
				// some servers add Content-Encoding: gzip to a .gz file, so the browser already unpacked it: JSON starts with '{'
				if (bytes.length > 2 && bytes[0] === 0x7b) return new TextDecoder().decode(bytes);
				// anything else (a dev server answering a missing file with index.html) falls through to the plain file
			}
		} catch {
			/* fall back to the plain file */
		}
	}
	const r = await fetch(base() + file);
	if (!r.ok) throw new Error(`could not load ${file}`);
	const bytes = await readAll(r, onProgress);
	return new TextDecoder().decode(bytes);
}
