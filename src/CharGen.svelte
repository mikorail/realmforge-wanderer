<script lang="ts">
	/** Qud-style character creation: species → calling → attributes → powers → origin. Rules mirror engine/chargen.go. */
	let { options, onbegin, oncancel }: { options: any; onbegin: (spec: object) => void; oncancel: () => void } = $props();

	const steps = ['Species', 'Calling', 'Attributes', options.powersName, 'Origin'];
	let step = $state(0);
	let species = $state<any>(options.species[0]);
	let calling = $state<any>(options.callings[0]);
	let attr = $state([12, 12, 12, 12, 12, 12]);
	let powers = $state<Record<string, number>>({});
	let name = $state('');
	let start = $state<number>((options.starts.find((b: any) => b.Species === options.species[0]?.ID) ?? options.starts[0])?.ID ?? 0);

	const cost = (v: number) => {
		let c = 0;
		for (let s = 12; s < v; s++) c += s >= 18 ? 2 : 1;
		return c;
	};
	const attrSpent = $derived(attr.reduce((t, v) => t + cost(v), 0));
	const attrLeft = $derived(options.attrPoints - attrSpent);
	const bonus = (i: number) => (species?.Attr?.[i] ?? 0) + (calling?.attr?.[i] ?? 0);
	const mod = (v: number) => (v >= 16 ? Math.floor((v - 16) / 2) : -Math.floor((17 - v) / 2));
	const powerCost = (id: string, lv: number) => {
		const p = options.powers.find((x: any) => x.id === id);
		if (!p || lv <= 0) return 0;
		return p.defect ? p.cost : p.cost + (lv - 1);
	};
	const powerSpent = $derived(Object.entries(powers).reduce((t, [id, lv]) => t + powerCost(id, lv), 0));
	const powerLeft = $derived(options.powerPoints - powerSpent);
	const starts = $derived([...options.starts].sort((a: any, b: any) => Number(b.Species === species?.ID) - Number(a.Species === species?.ID)));

	function bump(i: number, d: number) {
		const nv = attr[i] + d;
		if (nv < 8 || nv > 24) return;
		const next = attr.slice();
		next[i] = nv;
		if (options.attrPoints - next.reduce((t, v) => t + cost(v), 0) < 0) return;
		attr = next;
	}
	function setPower(id: string, lv: number) {
		const next = { ...powers, [id]: Math.max(0, Math.min(3, lv)) };
		if (!next[id]) delete next[id];
		const spent = Object.entries(next).reduce((t, [k, v]) => t + powerCost(k, v), 0);
		if (options.powerPoints - spent < 0) return;
		powers = next;
	}
	function pickSpecies(s: any) {
		species = s;
		const firstHome = options.starts.find((b: any) => b.Species === s.ID);
		if (firstHome) start = firstHome.ID;
	}
	function begin() {
		onbegin({ name: name || 'Wanderer', species: species.ID, calling: calling.id, attr, powers, start });
	}
	function onkey(e: KeyboardEvent) {
		if (e.target instanceof HTMLInputElement) return;
		if (e.key === 'ArrowRight' && step < steps.length - 1) step++;
		else if (e.key === 'ArrowLeft' && step > 0) step--;
		else if (e.key === 'Escape') oncancel();
	}
</script>

<svelte:window onkeydown={onkey} />

<div class="cg">
	<header>
		<h1>{options.world} <span>· Year {options.year} · {options.genre}</span></h1>
		<nav>
			{#each steps as s, i (s)}
				<button class:on={step === i} onclick={() => (step = i)}>{i + 1}. {s}</button>
			{/each}
		</nav>
	</header>

	<main>
		{#if step === 0}
			<p class="hint">Peoples of this world. Common ones have homelands you can start from.</p>
			<div class="grid">
				{#each options.species as s (s.ID)}
					<button class="card" class:on={species?.ID === s.ID} onclick={() => pickSpecies(s)}>
						<b>{s.Name}</b>
						{#if s.Common}<em>{s.Common} culture{s.Common > 1 ? 's' : ''}</em>{:else}<em class="dim">a stranger here</em>{/if}
						<small>{s.Desc}</small>
						<small class="mods">
							{#each Object.entries(s.Attr ?? {}) as [k, v] (k)}<span class:neg={(v as number) < 0}>{options.attrNames[+k].slice(0, 3).toUpperCase()} {(v as number) > 0 ? '+' : ''}{v}</span>{/each}
							{#if s.Plan !== 'humanoid'}<span class="plan">{s.Plan} body</span>{/if}
						</small>
					</button>
				{/each}
			</div>
		{:else if step === 1}
			<div class="grid">
				{#each options.callings as c (c.id)}
					<button class="card" class:on={calling?.id === c.id} onclick={() => (calling = c)}>
						<b>{c.name}</b>
						<small>{c.desc}</small>
						<small class="mods">
							{#each Object.entries(c.attr) as [k, v] (k)}<span>{options.attrNames[+k].slice(0, 3).toUpperCase()} +{v}</span>{/each}
							{#each Object.entries(c.skills) as [k, v] (k)}<span class="plan">{k} {v}</span>{/each}
						</small>
					</button>
				{/each}
			</div>
		{:else if step === 2}
			<p class="hint">Points left: <b class:neg={attrLeft < 0}>{attrLeft}</b> · above 18 costs 2 per point · modifier = (score − 16) / 2</p>
			<table>
				<tbody>
					{#each options.attrNames as a, i (a)}
						<tr>
							<td>{a}</td>
							<td><button onclick={() => bump(i, -1)} aria-label="Lower {a}">−</button></td>
							<td class="num">{attr[i]}</td>
							<td><button onclick={() => bump(i, 1)} aria-label="Raise {a}">+</button></td>
							<td class="dim">{bonus(i) ? `${bonus(i) > 0 ? '+' : ''}${bonus(i)} species/calling` : ''}</td>
							<td class="num total">{attr[i] + bonus(i)}</td>
							<td class="dim">({mod(attr[i] + bonus(i)) >= 0 ? '+' : ''}{mod(attr[i] + bonus(i))})</td>
						</tr>
					{/each}
				</tbody>
			</table>
		{:else if step === 3}
			<p class="hint">{options.powersName} points left: <b class:neg={powerLeft < 0}>{powerLeft}</b> · levels up to 3 at creation (+1 each) · defects give points back</p>
			<div class="grid">
				{#each options.powers as p (p.id)}
					{@const lv = powers[p.id] ?? 0}
					<div class="card" class:on={lv > 0} class:defect={p.defect}>
						<b>{p.name}</b>
						<em>{p.defect ? `defect, +${-p.cost}` : `${p.cost} pts${p.active ? ' · active' : ''}`}</em>
						<small>{p.desc}</small>
						<div class="row">
							{#if p.defect}
								<button onclick={() => setPower(p.id, lv ? 0 : 1)}>{lv ? 'Remove' : 'Take'}</button>
							{:else}
								<button onclick={() => setPower(p.id, lv - 1)} disabled={!lv}>−</button>
								<span class="num">{lv ? `lv ${lv}` : '—'}</span>
								<button onclick={() => setPower(p.id, lv + 1)}>+</button>
							{/if}
						</div>
					</div>
				{/each}
			</div>
		{:else}
			<label class="name">Name <input bind:value={name} maxlength="24" placeholder="Wanderer" /></label>
			<p class="hint">Where do you set out from? Your home realm starts friendly; its enemies don't.</p>
			<div class="list">
				{#each starts as b (b.ID)}
					<button class:on={start === b.ID} onclick={() => (start = b.ID)}>
						<b>{b.Name}</b> {b.Capital ? '★' : ''} <span class="dim">— {b.Realm} · {b.Culture}{b.Notables ? ` · ${b.Notables} notable${b.Notables > 1 ? 's' : ''}` : ''} {b.Species === species?.ID ? '· your people' : ''}</span>
					</button>
				{/each}
			</div>
		{/if}
	</main>

	<footer>
		<button onclick={oncancel}>Cancel</button>
		<span class="sum">{species?.Name} {calling?.name} · {Object.keys(powers).length} {options.powersName.toLowerCase()}</span>
		{#if step < steps.length - 1}
			<button class="go" onclick={() => step++}>Next →</button>
		{:else}
			<button class="go" onclick={begin} disabled={attrLeft < 0 || powerLeft < 0}>Begin</button>
		{/if}
	</footer>
</div>

<style>
	.cg {
		display: flex;
		flex-direction: column;
		height: 100%;
		max-width: 1100px;
		width: 100%;
		margin: 0 auto;
		padding: 12px 16px;
		box-sizing: border-box;
		min-height: 0;
	}
	h1 {
		color: #cfc041;
		font-weight: 400;
		font-size: 20px;
		margin: 0 0 8px;
	}
	h1 span {
		color: #40a4b9;
		font-size: 14px;
	}
	nav {
		display: flex;
		gap: 4px;
		flex-wrap: wrap;
	}
	main {
		flex: 1;
		overflow: auto;
		padding: 12px 0;
		min-height: 0;
	}
	button {
		font: inherit;
		background: #0d2b29;
		color: #b1c9c3;
		border: 1px solid #155352;
		padding: 6px 10px;
		cursor: pointer;
		text-align: left;
	}
	button.on,
	.card.on {
		border-color: #cfc041;
		color: #fff;
		background: #1b4d48;
	}
	button:disabled {
		opacity: 0.4;
		cursor: default;
	}
	.grid {
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(230px, 1fr));
		gap: 8px;
	}
	.card {
		display: flex;
		flex-direction: column;
		gap: 4px;
		border: 1px solid #155352;
		background: #0d2b29;
		padding: 8px 10px;
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
		line-height: 1.35;
	}
	.card.defect b {
		color: #d74200;
	}
	.mods {
		display: flex;
		flex-wrap: wrap;
		gap: 6px;
	}
	.mods span {
		color: #00c420;
	}
	.mods .neg,
	.neg {
		color: #d74200;
	}
	.mods .plan {
		color: #b154cf;
	}
	.row {
		display: flex;
		align-items: center;
		gap: 8px;
		margin-top: auto;
	}
	.hint {
		color: #40a4b9;
		margin: 0 0 10px;
	}
	.dim {
		color: #3f7f7a;
	}
	table td {
		padding: 3px 8px;
	}
	.num {
		text-align: center;
		min-width: 2.5em;
		color: #fff;
	}
	.total {
		color: #cfc041;
	}
	.name {
		display: flex;
		gap: 8px;
		align-items: center;
		margin-bottom: 12px;
	}
	input {
		font: inherit;
		background: #020a09;
		color: #fff;
		border: 1px solid #40a4b9;
		padding: 6px 8px;
	}
	.list {
		display: flex;
		flex-direction: column;
		gap: 3px;
	}
	footer {
		display: flex;
		gap: 10px;
		align-items: center;
		border-top: 1px solid #155352;
		padding-top: 10px;
	}
	.sum {
		flex: 1;
		color: #40a4b9;
	}
	.go {
		color: #cfc041;
		border-color: #cfc041;
	}
	@media (max-width: 600px) {
		.cg {
			padding: 8px 16px;
		}
	}
</style>
