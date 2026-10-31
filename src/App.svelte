<script lang="ts">
	/** Title → choose a world (bundled sample or a Realmforge export) → chargen → play. Saved runs continue from the title. */
	import { onMount } from 'svelte';
	import { loadQud, type QudApi, type QudFrame } from './qud';
	import { listRuns, loadRun, deleteRun, type RunMeta } from './store';
	import CharGen from './CharGen.svelte';
	import Play from './Play.svelte';
	import { listChronicles, exportChronicle, type Chronicle } from './chronicle';
	import { isLinked, waitForWorld } from './link';

	interface Sample {
		file: string;
		name: string;
		genre: string;
		year: number;
		blurb: string;
	}

	let phase = $state<'title' | 'loading' | 'chargen' | 'play' | 'error'>('title');
	let error = $state('');
	let samples = $state<Sample[]>([]);
	let runs = $state<RunMeta[]>([]);
	let options = $state<any>(null);
	let api: QudApi | null = null;
	let frame = $state<QudFrame | null>(null);
	let worldText: string | null = null;
	let busy = $state('');
	let chronicles = $state<Chronicle[]>([]);
	let linked = $state(false);
	let stopLink: (() => void) | null = null;

	onMount(async () => {
		runs = await listRuns();
		chronicles = await listChronicles();
		if (isLinked()) {
			linked = true;
			phase = 'loading';
			busy = 'Waiting for Realmforge to hand over the world…';
			stopLink = waitForWorld((json) => void linkedWorld(json));
		}
		try {
			samples = await (await fetch(import.meta.env.BASE_URL + 'worlds/index.json')).json();
		} catch {
			samples = [];
		}
		// warm the engine while the player reads the title
		loadQud().then((a) => (api = a)).catch(() => {});
	});

	/** World snapshot pushed by Realmforge: continue this seed's run on it, or go to chargen. */
	async function linkedWorld(json: string) {
		if (phase === 'play' || phase === 'chargen') return; // never yank a running game
		let seed = '';
		try {
			seed = String(JSON.parse(json).seed ?? '');
		} catch {
			return fail(new Error('Realmforge sent an unreadable world'));
		}
		const run = (await listRuns()).find((r) => r.seed === seed);
		if (!run) return startWorld(json);
		try {
			phase = 'loading';
			busy = `Waking ${run.name} in the changed world…`;
			const a = await engine();
			const saved = await loadRun(seed);
			if (!saved) return startWorld(json);
			const opts = a.init(json);
			if (typeof opts !== 'string') throw new Error(opts.error);
			const f = a.load(saved.save);
			if ('error' in f) throw new Error(f.error);
			worldText = json; // Play stores the newer snapshot with the next autosave
			frame = f;
			phase = 'play';
		} catch (e) {
			fail(e);
		}
	}

	async function engine() {
		api ??= await loadQud();
		return api;
	}

	function fail(e: unknown) {
		error = e instanceof Error ? e.message : String(e);
		phase = 'error';
	}

	async function startWorld(text: string) {
		try {
			busy = 'Reading the chronicle…';
			phase = 'loading';
			const a = await engine();
			const opts = a.init(text);
			if (typeof opts !== 'string') throw new Error(opts.error);
			options = JSON.parse(opts);
			worldText = text;
			phase = 'chargen';
		} catch (e) {
			fail(e);
		}
	}

	async function pickSample(s: Sample) {
		phase = 'loading';
		busy = `Fetching ${s.name}…`;
		try {
			const r = await fetch(import.meta.env.BASE_URL + 'worlds/' + s.file);
			if (!r.ok) throw new Error(`could not load ${s.file}`);
			await startWorld(await r.text());
		} catch (e) {
			fail(e);
		}
	}

	async function pickFile(e: Event) {
		const f = (e.currentTarget as HTMLInputElement).files?.[0];
		if (!f) return;
		await startWorld(await f.text());
	}

	function begin(spec: object) {
		if (!api) return;
		const f = api.newGame(JSON.stringify(spec));
		if ('error' in f) return fail(f.error);
		frame = f;
		phase = 'play';
	}

	async function resume(r: RunMeta) {
		try {
			phase = 'loading';
			busy = `Waking ${r.name}…`;
			const a = await engine();
			const run = await loadRun(r.seed);
			if (!run) throw new Error('save is missing');
			const opts = a.init(run.world);
			if (typeof opts !== 'string') throw new Error(opts.error);
			const f = a.load(run.save);
			if ('error' in f) throw new Error(f.error);
			worldText = null;
			frame = f;
			phase = 'play';
		} catch (e) {
			fail(e);
		}
	}

	async function forget(r: RunMeta) {
		if (!confirm(`Abandon ${r.name}? This deletes the save.`)) return;
		await deleteRun(r.seed);
		runs = await listRuns();
	}

	async function toTitle() {
		frame = null;
		phase = 'title';
		runs = await listRuns();
		chronicles = await listChronicles();
	}
</script>

{#if phase === 'play' && api && frame}
	<Play {api} first={frame} world={worldText} onquit={toTitle} />
{:else if phase === 'chargen'}
	<div class="cgwrap"><CharGen {options} onbegin={begin} oncancel={toTitle} /></div>
{:else}
	<main class="title">
		<pre class="logo" aria-hidden="true">{`  ╔═╗ ╔═╗ ╔═╗ ╦  ╔╦╗ ╔═╗ ╔═╗ ╦═╗ ╔═╗ ╔═╗
  ╠╦╝ ║╣  ╠═╣ ║  ║║║ ╠╣  ║ ║ ╠╦╝ ║ ╦ ║╣
  ╩╚═ ╚═╝ ╩ ╩ ╩═╝╩ ╩ ╚   ╚═╝ ╩╚═ ╚═╝ ╚═╝`}</pre>
		<h1>Wanderer</h1>
		<p class="sub">An ASCII roguelike across worlds forged in Realmforge: their realms, faiths, monsters and centuries of history.</p>

		{#if phase === 'loading'}
			<p class="pulse">{busy}</p>
		{:else if phase === 'error'}
			<p class="err">{error}</p>
			<button onclick={toTitle}>Back</button>
		{:else}
			{#if runs.length}
				<section>
					<h2>Continue</h2>
					{#each runs as r (r.seed)}
						<div class="run">
							<button class="main" onclick={() => resume(r)}>
								<b>{r.name}</b> · level {r.level} {r.species} {r.calling}
								<span class="dim">— {r.place}, {r.world} ({r.genre}, year {r.year})</span>
							</button>
							<button class="x" onclick={() => forget(r)} aria-label="Abandon {r.name}">✕</button>
						</div>
					{/each}
				</section>
			{/if}

			{#if chronicles.length}
				<section>
					<h2>Chronicles</h2>
					{#each chronicles as c (c.seed)}
						<div class="run">
							<button class="main" onclick={() => exportChronicle(c)}>
								Export chronicle: <b>{c.world}</b>
								<span class="dim">— {String(c.hero.name ?? 'a wanderer')}, {c.deeds.length} deed{c.deeds.length === 1 ? '' : 's'}</span>
							</button>
						</div>
					{/each}
				</section>
			{/if}

			<section>
				<h2>New life</h2>
				<div class="grid">
					{#each samples as s (s.file)}
						<button class="card" onclick={() => pickSample(s)}>
							<b>{s.name}</b>
							<em>{s.genre} · year {s.year}</em>
							<small>{s.blurb}</small>
						</button>
					{/each}
					<label class="card file">
						<b>Your own world</b>
						<em>Realmforge → Export → Wanderer world</em>
						<small>Load the .json file Realmforge saved.</small>
						<input type="file" accept=".json,application/json" onchange={pickFile} />
					</label>
				</div>
			</section>
		{/if}
	</main>
{/if}

<style>
	.title {
		max-width: 960px;
		margin: 0 auto;
		padding: 24px 16px 48px;
	}
	.logo {
		color: #cfc041;
		font-size: clamp(7px, 1.9vw, 16px);
		line-height: 1.15;
		margin: 0;
		overflow: hidden;
	}
	h1 {
		color: #77bfcf;
		font-weight: 400;
		letter-spacing: 0.4em;
		margin: 8px 0 4px;
	}
	.sub {
		color: #40a4b9;
		margin: 0 0 24px;
	}
	h2 {
		color: #cfc041;
		font-weight: 400;
		font-size: 16px;
		border-bottom: 1px solid #155352;
		padding-bottom: 4px;
	}
	section {
		margin-bottom: 24px;
	}
	.grid {
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(240px, 1fr));
		gap: 8px;
	}
	.card {
		display: flex;
		flex-direction: column;
		gap: 4px;
		text-align: left;
		padding: 10px 12px;
	}
	.card b {
		color: #77bfcf;
	}
	.card em {
		color: #98875f;
		font-style: normal;
		font-size: 12px;
	}
	.card small {
		font-size: 12px;
		line-height: 1.4;
	}
	.file {
		border: 1px dashed #40a4b9;
		background: #0d2b29;
		cursor: pointer;
	}
	.file input {
		font: inherit;
		font-size: 12px;
		max-width: 100%;
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
	.dim {
		color: #3f7f7a;
	}
	.err {
		color: #d74200;
	}
	.pulse {
		animation: pulse 1.4s ease-in-out infinite;
	}
	@keyframes pulse {
		50% {
			opacity: 0.4;
		}
	}
	.cgwrap {
		position: fixed;
		inset: 0;
	}
</style>
