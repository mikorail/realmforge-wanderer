/** Loads the Go engine (public/wasm/qud.wasm + Go's wasm_exec.js) once per page. */
export interface QudFrame {
	w: number;
	h: number;
	/** glyphs, one rune per cell */
	g: string;
	/** palette index + 65 per cell */
	f: string;
	b: string;
	save: boolean;
	exit: boolean;
	dead: boolean;
	mode: string;
}
type Err = { error: string };
export interface QudApi {
	palette(): string;
	init(world: string): string | Err;
	newGame(spec: string): QudFrame | Err;
	load(save: string): QudFrame | Err;
	key(k: string): QudFrame | Err | null;
	frame(): QudFrame | null;
	save(): string;
	meta(): string;
	deeds(): string;
}

declare global {
	interface Window {
		qud?: QudApi;
		Go?: new () => { importObject: WebAssembly.Imports; run(i: WebAssembly.Instance): Promise<void> };
	}
}

let loading: Promise<QudApi> | null = null;

function script(src: string) {
	return new Promise<void>((res, rej) => {
		const s = document.createElement('script');
		s.src = src;
		s.onload = () => res();
		s.onerror = () => rej(new Error(`could not load ${src}`));
		document.head.appendChild(s);
	});
}

export function loadQud(): Promise<QudApi> {
	if (window.qud) return Promise.resolve(window.qud);
	if (loading) return loading;
	loading = (async () => {
		if (!window.Go) await script(import.meta.env.BASE_URL + 'wasm/wasm_exec.js');
		const go = new window.Go!();
		const ready = new Promise<void>((res) => window.addEventListener('qud-ready', () => res(), { once: true }));
		const resp = await fetch(import.meta.env.BASE_URL + 'wasm/qud.wasm');
		if (!resp.ok) throw new Error(`qud.wasm missing (${resp.status})`);
		const { instance } = await WebAssembly.instantiate(await resp.arrayBuffer(), go.importObject);
		void go.run(instance);
		await ready;
		return window.qud!;
	})();
	loading.catch(() => (loading = null));
	return loading;
}
