<script lang="ts">
	/** A little sprite of a species in a calling's outfit, drawn with the same code as the map. */
	import { drawPersonSprite } from './actorsprites';

	let { species, calling = '', seed = 7, size = 56 }: { species: string; calling?: string; seed?: number; size?: number } = $props();
	let canvas = $state<HTMLCanvasElement | null>(null);

	$effect(() => {
		const c = canvas?.getContext('2d');
		if (!c || !canvas) return;
		const w = size * 0.62;
		const dpr = window.devicePixelRatio || 1;
		canvas.width = w * dpr;
		canvas.height = size * dpr;
		c.setTransform(dpr, 0, 0, dpr, 0, 0);
		c.clearRect(0, 0, w, size);
		drawPersonSprite(c, { x: 0, y: 0, i: seed, s: species, c: calling, p: true }, '#ffffff', [], 0, 0, 0, w, size);
	});
</script>

<canvas bind:this={canvas} style:width="{size * 0.62}px" style:height="{size}px" aria-hidden="true"></canvas>

<style>
	canvas {
		display: block;
		flex: none;
	}
</style>
