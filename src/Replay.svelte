<script lang="ts">
	/**
	 * Replay viewer. With `id` it plays that replay (fresh engine game fed the recorded keys, speed 1x/4x/16x,
	 * step, pause, highlight jumps, desync notice, highlight frame clips as the fallback); without `id` it lists
	 * stored replays (play, export, delete, import). Own grid renderer: Play.svelte's canvas drawing is not exported.
	 */
	import { onMount, onDestroy } from 'svelte';
	import type { QudApi, QudFrame } from './qud';
	import { loadQud } from './qud';
	import {
		listReplays, deleteReplay, downloadReplay, importReplay, loadReplay, openPlayer, replayPalette, hasReplayApi,
		type ReplayMeta, type ReplayPlayer, type PlayerState, type Speed, type Clip, type FrameLite, type Highlight
	} from './replay';

	let { api: apiProp, id: idProp = '', onclose }: { api?: QudApi | null; id?: string; onclose: () => void } = $props();

	let id = $state(idProp);
	let api: QudApi | null = apiProp ?? null;
	let list = $state<ReplayMeta[]>([]);
	let note = $state('');
	let err = $state('');
	let st = $state<PlayerState>({ frame: null, info: null });
	let speed = $state<Speed>(4);
	let running = $state(false);
	let clips = $state<Clip[]>([]);
	let clipMode = $state<Clip | null>(null);
	let clipI = $state(0);
	let clipTimer: ReturnType<typeof setInterval> | null = null;
	let player: ReplayPlayer | null = null;
	let canvas = $state<HTMLCanvasElement | null>(null);
	let palette: string[] = [];

	const info = $derived(st.info);
	const pct = $derived(info && info.total ? Math.round((info.n / info.total) * 100) : 0);

	async function refresh() {
		list = await listReplays();
	}

	async function start(rid: string) {
		err = '';
		stopAll();
		try {
			api = api ?? (await loadQud());
			palette = replayPalette(api);
			const r = await loadReplay(rid);
			clips = r?.clips ?? [];
			id = rid;
			player = await openPlayer(api, rid, (s) => {
				st = s;
				running = !!player?.playing();
				draw();
			});
			player.setSpeed(speed);
			draw();
		} catch (e) {
			err = e instanceof Error ? e.message : String(e);
			id = '';
		}
	}

	function stopAll() {
		if (clipTimer) clearInterval(clipTimer);
		clipTimer = null;
		clipMode = null;
		player?.close();
		player = null;
		running = false;
	}

	function drawFrame(f: FrameLite | QudFrame) {
		const c = canvas;
		if (!c || !palette.length) return;
		const ctx = c.getContext('2d');
		if (!ctx) return;
		const cw = 9;
		const ch = 16;
		c.width = f.w * cw;
		c.height = f.h * ch;
		ctx.font = `${ch - 2}px 'Share Tech Mono', Menlo, Consolas, monospace`;
		ctx.textBaseline = 'top';
		const glyphs = Array.from(f.g);
		for (let i = 0; i < f.w * f.h; i++) {
			const x = (i % f.w) * cw;
			const y = Math.floor(i / f.w) * ch;
			ctx.fillStyle = palette[f.b.charCodeAt(i) - 65] ?? '#000';
			ctx.fillRect(x, y, cw, ch);
			const g = glyphs[i];
			if (g && g !== ' ') {
				ctx.fillStyle = palette[f.f.charCodeAt(i) - 65] ?? '#fff';
				ctx.fillText(g, x, y + 1);
			}
		}
	}
	function draw() {
		if (clipMode) drawFrame(clipMode.frames[clipI]);
		else if (st.frame) drawFrame(st.frame);
	}
	$effect(() => {
		void canvas;
		void clipI;
		void clipMode;
		draw();
	});

	function playPause() {
		if (!player) return;
		if (player.playing()) player.pause();
		else player.play();
		running = player.playing();
	}
	function setSpeed(s: Speed) {
		speed = s;
		player?.setSpeed(s);
	}
	function onTimeline(e: Event) {
		if (!info) return;
		const n = Math.round((Number((e.currentTarget as HTMLInputElement).value) / 1000) * info.total);
		player?.seek(n);
	}
	function jump(h: Highlight) {
		player?.seek(h.k);
	}
	function playClip(c: Clip) {
		player?.pause();
		clipMode = c;
		clipI = 0;
		if (clipTimer) clearInterval(clipTimer);
		clipTimer = setInterval(() => {
			if (!clipMode) return;
			if (clipI < clipMode.frames.length - 1) clipI++;
			else {
				if (clipTimer) clearInterval(clipTimer);
				clipTimer = null;
			}
		}, 200);
	}

	async function del(m: ReplayMeta) {
		if (!confirm(`Delete the replay of ${m.name}?`)) return;
		await deleteReplay(m.id);
		await refresh();
	}
	async function onImport(e: Event) {
		const f = (e.currentTarget as HTMLInputElement).files?.[0];
		if (!f) return;
		try {
			const m = await importReplay(f);
			note = `Imported the replay of ${m.name}.`;
			await refresh();
		} catch (x) {
			note = x instanceof Error ? x.message : 'Import failed.';
		}
	}

	function onkey(e: KeyboardEvent) {
		if (!id) {
			if (e.key === 'Escape') onclose();
			return;
		}
		if (e.key === 'Escape') {
			e.preventDefault();
			e.stopPropagation();
			stopAll();
			if (idProp) onclose();
			else {
				id = '';
				void refresh();
			}
		} else if (e.key === ' ') {
			e.preventDefault();
			playPause();
		} else if (e.key === '.' || e.key === 'ArrowRight') {
			e.preventDefault();
			player?.step();
		} else if (e.key === 'n') player?.nextHighlight();
		else if (e.key === 'p') player?.prevHighlight();
		else if (e.key === '1') setSpeed(1);
		else if (e.key === '2') setSpeed(4);
		else if (e.key === '3') setSpeed(16);
		else return;
		e.stopPropagation();
	}

	onMount(() => {
		window.addEventListener('keydown', onkey, true);
		void refresh();
		if (idProp) void start(idProp);
	});
	onDestroy(() => {
		window.removeEventListener('keydown', onkey, true);
		stopAll();
	});
	// reference to keep the type guard import used (engine build without replay support)
	const supported = $derived(!api || hasReplayApi(api));
</script>

<div class="rp" role="dialog" aria-label="Replay">
	{#if !id}
		<h2>Replays</h2>
		<div class="bar">
			<label class="file">Import file<input type="file" accept=".rfreplay" onchange={onImport} hidden /></label>
			<button onclick={onclose}>Close</button>
		</div>
		{#if note}<p class="dim">{note}</p>{/if}
		{#if err}<p class="bad">{err}</p>{/if}
		{#if !supported}<p class="bad">This engine build has no replay support.</p>{/if}
		{#if !list.length}<p class="dim">No replays yet. Runs record one while the Record replays setting is on.</p>{/if}
		{#each list as m (m.id)}
			<div class="row">
				<span class="grow"><b>{m.name}</b> · level {m.level} {m.species} {m.calling} <span class="dim">— {m.world}, {m.keys} keys, {m.highlights} highlights, {m.final ? (m.dead ? 'fell' : 'ended') : 'in progress'}</span></span>
				<button onclick={() => start(m.id)}>Watch</button>
				<button onclick={() => downloadReplay(m.id)}>Export</button>
				<button class="x" onclick={() => del(m)} aria-label="Delete replay of {m.name}">✕</button>
			</div>
		{/each}
	{:else}
		<div class="bar">
			<b>{info?.name ?? ''}</b>
			<span class="dim">turn {info?.turn ?? 0} · key {info?.n ?? 0}/{info?.total ?? 0} ({pct}%)</span>
			<button onclick={() => { stopAll(); if (idProp) onclose(); else { id = ''; void refresh(); } }}>Close</button>
		</div>
		{#if info?.diverged}
			<p class="bad">{info.diverged}. The key playback no longer matches the recorded run{clips.length ? '; the highlight clips below still show what happened' : ''}.</p>
		{:else if info?.trunc}
			<p class="dim">The recording hit its length cap, so playback ends early.</p>
		{/if}
		<canvas bind:this={canvas} class="grid" aria-label="Replay screen"></canvas>
		{#if clipMode}<p class="dim">Clip: {clipMode.hl.kind} — {clipMode.hl.text} (frame {clipI + 1}/{clipMode.frames.length}) <button onclick={() => (clipMode = null)}>Back to playback</button></p>{/if}
		<div class="ctl">
			<button onclick={() => player?.restart()} title="Restart">⏮</button>
			<button onclick={playPause} disabled={!!info?.diverged}>{running ? 'Pause' : 'Play'}</button>
			<button onclick={() => player?.step()} title="One key (.)" disabled={!!info?.diverged}>Step</button>
			{#each [1, 4, 16] as s (s)}
				<button class:on={speed === s} onclick={() => setSpeed(s as Speed)}>{s}x</button>
			{/each}
			<button onclick={() => player?.prevHighlight()} title="Previous highlight (p)">◀ Highlight</button>
			<button onclick={() => player?.nextHighlight()} title="Next highlight (n)">Highlight ▶</button>
		</div>
		<input class="tl" type="range" min="0" max="1000" value={info && info.total ? Math.round((info.n / info.total) * 1000) : 0} oninput={onTimeline} aria-label="Timeline" />
		<div class="marks">
			{#each info?.hl ?? [] as h, i (i)}
				<button class="mk {h.kind}" style:left="{info && info.total ? (h.k / info.total) * 100 : 0}%" onclick={() => jump(h)} title="{h.kind}: {h.text}" aria-label="{h.kind}: {h.text}"></button>
			{/each}
		</div>
		<ul class="hl">
			{#each info?.hl ?? [] as h, i (i)}
				<li>
					<button class="link" onclick={() => jump(h)}><span class="k {h.kind}">{h.kind}</span> turn {h.t}: {h.text}</button>
					{#each clips.filter((c) => c.hl.k === h.k && c.hl.text === h.text) as c (c)}<button class="link" onclick={() => playClip(c)}>[clip]</button>{/each}
				</li>
			{/each}
		</ul>
		{#if info?.diverged}
			<ul class="hl">
				{#each clips.filter((c) => !(info?.hl ?? []).some((h) => h.k === c.hl.k && h.text === c.hl.text)) as c, i (i)}
					<li><button class="link" onclick={() => playClip(c)}>[clip] {c.hl.kind}: {c.hl.text}</button></li>
				{/each}
			</ul>
		{/if}
		<p class="dim">Space play/pause · . step · n/p highlights · 1 2 3 speed · Esc close</p>
	{/if}
</div>

<style>
	.rp {
		position: fixed;
		inset: 0;
		z-index: 60;
		background: var(--bg, #041312);
		overflow: auto;
		padding: 12px 16px;
		box-sizing: border-box;
	}
	h2 {
		color: #cfc041;
		font-weight: 400;
		font-size: 16px;
		border-bottom: 1px solid #155352;
	}
	.bar,
	.ctl {
		display: flex;
		gap: 6px;
		align-items: center;
		flex-wrap: wrap;
		margin: 6px 0;
	}
	.row {
		display: flex;
		gap: 4px;
		margin-bottom: 4px;
		align-items: center;
	}
	.grow {
		flex: 1;
	}
	.file {
		border: 1px solid #155352;
		padding: 6px 10px;
		cursor: pointer;
		background: #0d2b29;
	}
	.dim {
		color: #3f7f7a;
		font-size: 13px;
	}
	.bad {
		color: #f15f22;
	}
	.grid {
		max-width: 100%;
		height: auto;
		image-rendering: pixelated;
		border: 1px solid #155352;
	}
	.on {
		background: #1b4d48;
		color: #fff;
	}
	.tl {
		width: 100%;
	}
	.marks {
		position: relative;
		height: 14px;
	}
	.mk {
		position: absolute;
		top: 0;
		width: 6px;
		height: 12px;
		padding: 0;
		border: 0;
		background: #cfc041;
		transform: translateX(-3px);
	}
	.mk.level {
		background: #77bfcf;
	}
	.mk.boss {
		background: #d74200;
	}
	.mk.death {
		background: #ffffff;
	}
	.hl {
		list-style: none;
		padding: 0;
		margin: 6px 0;
		font-size: 13px;
		max-height: 30vh;
		overflow: auto;
	}
	.link {
		background: none;
		border: 0;
		padding: 2px 4px;
		text-align: left;
		color: var(--ink, #b1c9c3);
	}
	.k {
		color: #cfc041;
	}
	.k.boss {
		color: #d74200;
	}
	.k.level {
		color: #77bfcf;
	}
	.k.death {
		color: #fff;
	}
	@media (prefers-reduced-motion: reduce) {
		.rp {
			scroll-behavior: auto;
		}
	}
</style>
