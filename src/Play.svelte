<script lang="ts">
	/** The game screen: draws the engine's character grid on a canvas and forwards keys and clicks. */
	import { onMount, onDestroy } from 'svelte';
	import type { QudApi, QudFrame } from './qud';
	import { saveRun, deleteRun } from './store';

	let { api, first, world, onquit }: { api: QudApi; first: QudFrame; world: string | null; onquit: () => void } = $props();

	let canvas = $state<HTMLCanvasElement | null>(null);
	let wrap = $state<HTMLDivElement | null>(null);
	let palette: string[] = JSON.parse(api.palette());
	let frame: QudFrame = first;
	let cw = 9;
	let chh = 16;
	let keys = 0;
	let worldPending = world;

	async function persist() {
		const meta = JSON.parse(api.meta());
		if (frame.dead) {
			await deleteRun(meta.seed);
			return;
		}
		await saveRun({ ...meta, saved: Date.now() }, api.save(), worldPending ?? undefined);
		worldPending = null;
	}

	async function quit() {
		await persist();
		onquit();
	}

	const NUMPAD: Record<string, string> = { Numpad1: 'N1', Numpad2: 'N2', Numpad3: 'N3', Numpad4: 'N4', Numpad5: 'N5', Numpad6: 'N6', Numpad7: 'N7', Numpad8: 'N8', Numpad9: 'N9' };
	function onkey(e: KeyboardEvent) {
		if (e.metaKey || e.ctrlKey || e.altKey) return;
		const k = NUMPAD[e.code] ?? e.key;
		if (k === 'Shift' || k === 'Control' || k === 'Alt' || k === 'Meta' || k === 'CapsLock') return;
		e.preventDefault();
		send(k);
	}
	function send(k: string) {
		const f = api.key(k);
		if (!f || 'error' in f) return;
		frame = f;
		draw();
		keys++;
		if (f.save || keys % 150 === 0) void persist();
		if (f.exit) void quit();
	}
	function onclick(e: MouseEvent) {
		if (!canvas) return;
		const r = canvas.getBoundingClientRect();
		send(`click:${Math.floor((e.clientX - r.left) / cw)},${Math.floor((e.clientY - r.top) / chh)}`);
	}

	function resize() {
		if (!canvas || !wrap) return;
		const W = wrap.clientWidth, H = wrap.clientHeight;
		chh = Math.max(8, Math.floor(Math.min(H / frame.h, W / (frame.w * 0.6))));
		cw = Math.max(5, Math.floor(chh * 0.6));
		const dpr = window.devicePixelRatio || 1;
		canvas.width = frame.w * cw * dpr;
		canvas.height = frame.h * chh * dpr;
		canvas.style.width = `${frame.w * cw}px`;
		canvas.style.height = `${frame.h * chh}px`;
		canvas.getContext('2d')?.setTransform(dpr, 0, 0, dpr, 0, 0);
		draw();
	}

	function draw() {
		const f = frame;
		const ctx = canvas?.getContext('2d');
		if (!ctx) return;
		const glyphs = Array.from(f.g);
		ctx.font = `${Math.round(chh * 0.86)}px "Share Tech Mono", "DejaVu Sans Mono", Menlo, Consolas, monospace`;
		ctx.textAlign = 'center';
		ctx.textBaseline = 'middle';
		for (let i = 0; i < f.w * f.h; i++) {
			ctx.fillStyle = palette[f.b.charCodeAt(i) - 65];
			ctx.fillRect((i % f.w) * cw, Math.floor(i / f.w) * chh, cw + 0.5, chh + 0.5);
		}
		for (let i = 0; i < f.w * f.h; i++) {
			const g = glyphs[i];
			if (g === ' ') continue;
			const x = (i % f.w) * cw, y = Math.floor(i / f.w) * chh;
			ctx.fillStyle = palette[f.f.charCodeAt(i) - 65];
			if (g === '█') ctx.fillRect(x, y, cw + 0.5, chh + 0.5);
			else if (g === '─') ctx.fillRect(x, y + chh / 2, cw + 0.5, 1);
			else if (g === '│') ctx.fillRect(x + cw / 2, y, 1, chh + 0.5);
			else ctx.fillText(g, x + cw / 2, y + chh / 2 + 1);
		}
	}

	onMount(() => {
		resize();
		void persist();
		// fonts load after first paint: redraw once they are in
		void document.fonts?.ready.then(draw);
	});
	onDestroy(() => void persist());
</script>

<svelte:window onkeydown={onkey} onresize={resize} />

<!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_static_element_interactions -->
<div class="screen" bind:this={wrap} onclick={onclick}>
	<canvas bind:this={canvas} aria-label="Game screen"></canvas>
</div>
<button class="quit" onclick={quit} title="Save and return to the title screen">Save &amp; quit</button>

<style>
	.screen {
		position: fixed;
		inset: 0;
		display: flex;
		align-items: center;
		justify-content: center;
		overflow: hidden;
	}
	canvas {
		display: block;
		cursor: crosshair;
	}
	.quit {
		position: fixed;
		top: 6px;
		right: 8px;
		padding: 2px 8px;
		font-size: 12px;
		opacity: 0.6;
	}
	.quit:hover {
		opacity: 1;
	}
</style>
