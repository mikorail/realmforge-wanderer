<script lang="ts">
	/** A little sprite of a species in a calling's outfit, drawn with the same code as the map. */
	import { drawPersonSprite } from './actorsprites';
	import { drawPixelPortrait } from './pixelart/index';
	import { pixelBlit } from './pixelpass';
	import type { Pose } from './poses';

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
		const actor = { x: 0, y: 0, i: seed, s: species, c: calling, p: true };
		const pose = { phase: 0, attack: 0, cast: 0, hurt: 0, block: 0, face: 1 } as unknown as Pose;
		if (drawPixelPortrait(c, actor, '#c8b27a', '#0e2220', 0, 0, w, size, 0, pose)) return;
		// vector drawer through the pixel pass: coarse grid, hard edges, outline
		if (!pixelBlit(c, 0, 0, w, size, 24, 38, (pc, gw, gh) => (drawPersonSprite(pc, actor, '#ffffff', [], 0, 0, 0, gw, gh), true), { outline: true, levels: 7 }))
			drawPersonSprite(c, actor, '#ffffff', [], 0, 0, 0, w, size);
	});
</script>

<canvas bind:this={canvas} style:width="{size * 0.62}px" style:height="{size}px" aria-hidden="true"></canvas>

<style>
	canvas {
		display: block;
		flex: none;
	}
</style>
