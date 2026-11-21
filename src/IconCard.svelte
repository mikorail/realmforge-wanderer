<script lang="ts">
	import { drawIcon, type IconRowInfo } from './icons';

	/** Hover card for item list rows (inventory, trade, stash, cooking, display cases). Mouse only. */
	let { canvas, cell, rows, enabled = true }: {
		canvas: HTMLCanvasElement | null;
		cell: () => { cw: number; chh: number };
		rows: IconRowInfo[] | undefined;
		enabled?: boolean;
	} = $props();

	let pos = $state<{ gx: number; gy: number; x: number; y: number } | null>(null);
	let iconCanvas = $state<HTMLCanvasElement | null>(null);

	const hit = $derived.by(() => {
		if (!enabled || !pos || !rows) return null;
		return rows.find((r) => r.y === pos!.gy && pos!.gx >= r.x0 && pos!.gx < r.x1) ?? null;
	});

	function move(e: PointerEvent) {
		if (e.pointerType !== 'mouse' || !canvas) return;
		const r = canvas.getBoundingClientRect();
		const { cw, chh } = cell();
		pos = { gx: Math.floor((e.clientX - r.left) / cw), gy: Math.floor((e.clientY - r.top) / chh), x: e.clientX, y: e.clientY };
	}
	function hide() {
		pos = null;
	}

	$effect(() => {
		const c = canvas;
		if (!c) return;
		c.addEventListener('pointermove', move);
		c.addEventListener('pointerleave', hide);
		c.addEventListener('pointerdown', hide);
		window.addEventListener('keydown', hide);
		return () => {
			c.removeEventListener('pointermove', move);
			c.removeEventListener('pointerleave', hide);
			c.removeEventListener('pointerdown', hide);
			window.removeEventListener('keydown', hide);
		};
	});

	$effect(() => {
		const c = iconCanvas, h = hit;
		if (!c || !h) return;
		const ctx = c.getContext('2d');
		if (!ctx) return;
		ctx.clearRect(0, 0, c.width, c.height);
		drawIcon(ctx, h.i, 0, 0, 48);
	});

	const left = $derived(pos ? Math.min(pos.x + 16, (typeof window !== 'undefined' ? window.innerWidth : 9999) - 290) : 0);
	const top = $derived(pos ? Math.min(pos.y + 18, (typeof window !== 'undefined' ? window.innerHeight : 9999) - 100) : 0);
</script>

{#if hit}
	<div class="card" style:left="{left}px" style:top="{top}px" role="tooltip">
		<canvas bind:this={iconCanvas} width="48" height="48"></canvas>
		<div class="txt">
			<b>{hit.n}</b>
			{#if hit.d}<span>{hit.d}</span>{/if}
		</div>
	</div>
{/if}

<style>
	.card {
		position: fixed;
		z-index: 21;
		display: flex;
		gap: 8px;
		max-width: 280px;
		padding: 6px 8px;
		font: 12px/1.35 'Share Tech Mono', 'DejaVu Sans Mono', Menlo, Consolas, monospace;
		background: rgba(8, 14, 16, 0.94);
		color: #cfd8d6;
		border: 1px solid #2a5a62;
		border-radius: 3px;
		pointer-events: none;
	}
	canvas {
		width: 48px;
		height: 48px;
		flex: none;
		image-rendering: pixelated;
	}
	.txt {
		display: flex;
		flex-direction: column;
		min-width: 0;
	}
	b {
		font-weight: normal;
		color: #fff;
	}
	span {
		color: #9ab;
	}
</style>
