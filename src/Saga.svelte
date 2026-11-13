<script lang="ts">
	/** Full-screen saga reader: chapter pages, journey map, stats, epitaph, copy / download / share. */
	import { onMount } from 'svelte';
	import { drawSagaMap, sagaText, sagaStatsLine, downloadSagaText, downloadSagaMap, CHAPTER_COLORS, type SagaData } from './saga';

	let { saga, onclose }: { saga: SagaData; onclose: () => void } = $props();
	let page = $state(0);
	let mapIdx = $state(0);
	let note = $state('');
	let canvas = $state<HTMLCanvasElement | null>(null);
	const last = $derived(saga.chapters.length - 1);
	const ch = $derived(saga.chapters[page]);
	const colorOf = (i: number) => CHAPTER_COLORS[i % CHAPTER_COLORS.length];

	$effect(() => {
		void mapIdx;
		if (canvas) drawSagaMap(canvas, saga, mapIdx);
	});

	async function copy(withSeed: boolean) {
		const t = sagaText(saga, withSeed);
		try {
			await navigator.clipboard.writeText(t);
			note = withSeed ? 'Copied the saga with the world and seed.' : 'Copied the saga.';
		} catch {
			note = 'Clipboard unavailable: use Download.';
		}
	}
	function onkey(e: KeyboardEvent) {
		if (e.key === 'Escape') onclose();
		else if (e.key === 'ArrowRight' || e.key === 'PageDown') page = Math.min(last, page + 1);
		else if (e.key === 'ArrowLeft' || e.key === 'PageUp') page = Math.max(0, page - 1);
		else return;
		e.preventDefault();
		e.stopPropagation();
	}
	onMount(() => {
		window.addEventListener('keydown', onkey, true);
		return () => window.removeEventListener('keydown', onkey, true);
	});
</script>

<div class="saga" role="dialog" aria-label="Saga">
	<header>
		<h1>{saga.title}</h1>
		<div class="dim">{saga.subtitle}</div>
		<div class="dim">{saga.chronicler}</div>
	</header>
	<div class="body">
		<article>
			<h2 style:color={colorOf(page)}>{ch.title}</h2>
			<p>{ch.text}</p>
			{#if page === last}
				<pre class="epitaph">{saga.epitaph}</pre>
			{/if}
			<nav>
				<button onclick={() => (page = Math.max(0, page - 1))} disabled={page === 0}>← Prev</button>
				<span class="dim">{page + 1} / {saga.chapters.length}</span>
				<button onclick={() => (page = Math.min(last, page + 1))} disabled={page === last}>Next →</button>
			</nav>
			<ol class="toc">
				{#each saga.chapters as c, i (i)}
					<li><button class:cur={i === page} onclick={() => (page = i)}><span class="sw" style:background={colorOf(i)}></span>{c.title}</button></li>
				{/each}
			</ol>
		</article>
		<aside>
			{#if saga.maps.length}
				<div class="tabs">
					{#each saga.maps as m, i (m.planet)}
						<button class:cur={i === mapIdx} onclick={() => (mapIdx = i)}>{m.name}</button>
					{/each}
				</div>
				<canvas bind:this={canvas} aria-label="Journey map"></canvas>
				<div class="dim legend">green dot: where it began · {saga.dead ? 'red' : 'white'} dot: where it ended · line colour follows the chapters</div>
			{:else}
				<p class="dim">No journey was recorded.</p>
			{/if}
			<div class="stats">{sagaStatsLine(saga)}</div>
		</aside>
	</div>
	<footer>
		<button onclick={() => copy(false)}>Copy as text</button>
		<button onclick={() => downloadSagaText(saga)}>Download .txt</button>
		<button onclick={() => canvas && downloadSagaMap(canvas, saga)} disabled={!saga.maps.length}>Download map PNG</button>
		<button onclick={() => copy(true)}>Share</button>
		<button onclick={onclose}>Close</button>
		{#if note}<span class="dim">{note}</span>{/if}
	</footer>
</div>

<style>
	.saga {
		position: fixed;
		inset: 0;
		z-index: 50;
		background: #0b1415;
		color: #98875f;
		overflow: auto;
		padding: 16px 20px;
		font-family: 'Share Tech Mono', monospace;
		display: flex;
		flex-direction: column;
		gap: 12px;
	}
	h1 {
		color: #cfc041;
		font-weight: 400;
		font-size: 20px;
		margin: 0 0 4px;
	}
	h2 {
		font-weight: 400;
		font-size: 16px;
		margin: 0 0 8px;
	}
	.dim {
		color: #3f7f7a;
		font-size: 12px;
	}
	.body {
		display: flex;
		gap: 24px;
		flex-wrap: wrap;
		flex: 1;
	}
	article {
		flex: 1 1 360px;
		max-width: 640px;
		line-height: 1.6;
		font-size: 14px;
	}
	aside {
		flex: 1 1 360px;
		min-width: 0;
	}
	aside canvas {
		width: 100%;
		max-width: 640px;
		image-rendering: pixelated;
		border: 1px solid #155352;
		display: block;
	}
	.epitaph {
		border-left: 2px solid #155352;
		padding: 4px 12px;
		color: #cfc041;
		white-space: pre-wrap;
		font-family: inherit;
	}
	nav,
	.tabs,
	footer {
		display: flex;
		gap: 6px;
		align-items: center;
		flex-wrap: wrap;
		margin: 8px 0;
	}
	.toc {
		list-style: none;
		padding: 0;
		margin: 12px 0 0;
		display: grid;
		gap: 2px;
	}
	.toc button {
		width: 100%;
		text-align: left;
		display: flex;
		align-items: center;
		gap: 6px;
	}
	.cur {
		color: #cfc041;
	}
	.sw {
		width: 10px;
		height: 10px;
		display: inline-block;
	}
	.stats {
		margin-top: 8px;
		color: #cfc041;
		font-size: 13px;
	}
	.legend {
		margin-top: 4px;
	}
</style>
