<script lang="ts">
	/** Title-screen hall of fame: past lives by score, filter by world, delete, export, share. */
	import { onMount } from 'svelte';
	import { listHall, deleteHall, exportHall, shareText, type HallEntry } from './halloffame';

	let { onclose }: { onclose: () => void } = $props();
	let all = $state<HallEntry[]>([]);
	let world = $state('');
	let open = $state('');
	let note = $state('');
	const worlds = $derived([...new Set(all.map((e) => e.world))]);
	const shown = $derived(world ? all.filter((e) => e.world === world) : all);

	onMount(async () => {
		all = await listHall();
	});

	async function del(e: HallEntry) {
		if (!confirm(`Remove ${e.name} from the hall?`)) return;
		await deleteHall(e.id);
		all = await listHall();
	}
	async function share(e: HallEntry) {
		const t = shareText(e);
		try {
			await navigator.clipboard.writeText(t);
			note = 'Copied: ' + t;
		} catch {
			note = t;
		}
	}
	function onkey(ev: KeyboardEvent) {
		if (ev.key === 'Escape') onclose();
	}
</script>

<svelte:window onkeydown={onkey} />

<section class="hof">
	<h2>Hall of Fame</h2>
	<div class="bar">
		<select bind:value={world} aria-label="Filter by world">
			<option value="">All worlds</option>
			{#each worlds as w (w)}<option value={w}>{w}</option>{/each}
		</select>
		<button onclick={() => exportHall(shown)} disabled={!shown.length}>Export JSON</button>
		<button onclick={onclose}>Close</button>
	</div>
	{#if note}<p class="dim note">{note}</p>{/if}
	{#if !shown.length}
		<p class="dim">No lives recorded yet. A life enters the hall when it ends, by death or by retiring.</p>
	{/if}
	{#each shown as e, i (e.id)}
		<div class="run">
			<button class="main" onclick={() => (open = open === e.id ? '' : e.id)}>
				<span class="rank">{i + 1}.</span> <b>{e.name}</b> · <span class="score">{e.score}</span> · level {e.level} {e.species} {e.calling}
				<span class="dim">— {e.world}, {e.retired ? 'retired' : 'fell'} year {e.yearEnd}{e.seedText ? ` · seed ${e.seedText}` : ''}</span>
			</button>
			<button onclick={() => share(e)} title="Copy seed and score" aria-label="Share {e.name}">Share</button>
			<button class="x" onclick={() => del(e)} aria-label="Remove {e.name}">✕</button>
		</div>
		{#if open === e.id}
			<div class="detail">
				<div class="dim">
					{e.genre}{e.planet ? ', ' + e.planet : ''} · years {e.yearStart}-{e.yearEnd} · {e.days} days · {e.km} km · {e.kills} kills · {e.deedCount} deeds · gold peak {e.goldPeak}
					{#if e.scenario}· scenario {e.scenario}{/if} · {e.mode}
				</div>
				<div>{e.retired ? 'Retired honourably.' : e.cause}</div>
				{#if e.family}<div class="dim">{e.family}</div>{/if}
				<ul>
					{#each e.deeds as d (d)}<li>{d}</li>{/each}
				</ul>
			</div>
		{/if}
	{/each}
</section>

<style>
	.hof {
		margin-bottom: 24px;
	}
	h2 {
		color: #cfc041;
		font-weight: 400;
		font-size: 16px;
		border-bottom: 1px solid #155352;
		padding-bottom: 4px;
	}
	.bar {
		display: flex;
		gap: 4px;
		margin-bottom: 8px;
		flex-wrap: wrap;
	}
	.run {
		display: flex;
		gap: 4px;
		margin-bottom: 4px;
	}
	.run .main {
		flex: 1;
		text-align: left;
	}
	.rank,
	.score {
		color: #cfc041;
	}
	.dim {
		color: #3f7f7a;
	}
	.note {
		font-size: 12px;
		word-break: break-word;
	}
	.detail {
		border-left: 2px solid #155352;
		padding: 4px 12px;
		margin: 0 0 8px;
		font-size: 13px;
		line-height: 1.5;
	}
	ul {
		margin: 4px 0 0;
		padding-left: 18px;
		color: #98875f;
	}
</style>
