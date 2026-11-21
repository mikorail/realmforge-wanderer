<script lang="ts">
	import { drawStatusIcon, type IconStatusInfo } from './icons';

	/** Status icons on the sidebar's hunger row, over the grid canvas. Hover for names. Mouse and touch. */
	let { canvas, cell, status, cols = 112, enabled = true }: {
		canvas: HTMLCanvasElement | null;
		cell: () => { cw: number; chh: number };
		status: IconStatusInfo | undefined;
		cols?: number;
		enabled?: boolean;
	} = $props();

	let strip = $state<HTMLCanvasElement | null>(null);
	let box = $state({ left: 0, top: 0, w: 0, h: 0, size: 16, n: 0 });
	let label = $state('');

	function layout() {
		if (!canvas || !status) return;
		const r = canvas.getBoundingClientRect();
		const { cw, chh } = cell();
		const size = Math.max(8, Math.round(chh));
		const room = (cols - status.x0) * cw;
		const n = Math.max(0, Math.min(status.icons.length, Math.floor(room / (size + 1))));
		box = { left: r.left + status.x0 * cw, top: r.top + status.y * chh, w: n * (size + 1), h: size, size, n };
	}

	$effect(() => {
		status;
		canvas;
		layout();
	});

	$effect(() => {
		const c = strip;
		const st = status;
		if (!c || !st || box.n === 0) return;
		const dpr = window.devicePixelRatio || 1;
		c.width = Math.max(1, Math.round(box.w * dpr));
		c.height = Math.max(1, Math.round(box.h * dpr));
		const ctx = c.getContext('2d');
		if (!ctx) return;
		ctx.setTransform(dpr, 0, 0, dpr, 0, 0);
		ctx.clearRect(0, 0, box.w, box.h);
		for (let i = 0; i < box.n; i++) drawStatusIcon(ctx, st.icons[i], i * (box.size + 1), 0, box.size);
	});

	function over(e: PointerEvent) {
		if (!status) return;
		const r = (e.currentTarget as HTMLElement).getBoundingClientRect();
		const i = Math.floor((e.clientX - r.left) / (box.size + 1));
		const s = status.icons[i];
		const more = status.icons.length - box.n;
		label = s ? (s.n ?? s.k) : '';
		if (!s && more > 0) label = `${more} more`;
	}
</script>

<svelte:window onresize={layout} />

{#if enabled && status && box.n > 0}
	<div class="strip" style:left="{box.left}px" style:top="{box.top}px" style:width="{box.w}px" style:height="{box.h}px" onpointermove={over} onpointerleave={() => (label = '')} title={label} role="img" aria-label="Active conditions">
		<canvas bind:this={strip} style:width="{box.w}px" style:height="{box.h}px"></canvas>
	</div>
{/if}

<style>
	.strip {
		position: fixed;
		z-index: 5;
	}
	canvas {
		display: block;
		image-rendering: pixelated;
	}
</style>
