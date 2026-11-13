<script lang="ts">
	/** Qud-style character creation: species → calling → attributes → powers → origin. Rules mirror engine/chargen.go. */
	import Portrait from './Portrait.svelte';
	let { options, onbegin, oncancel }: { options: any; onbegin: (spec: object) => void; oncancel: () => void } = $props();

	let step = $state(0);
	let genotype = $state<string>('mutant');
	let morph = $state<string>('');
	let gameMode = $state<string>('classic');
	const truekin = $derived(genotype === 'truekin');
	let power = $state<string>('');
	let twist = $state<string>('');
	const challenges: any[] = options.challenges ?? [];
	let mods = $state<string[]>([]);
	function toggleMod(id: string) {
		mods = mods.includes(id) ? mods.filter((m) => m !== id) : [...mods, id];
	}
	const scenarios: any[] = options.scenarios ?? [];
	const mirror = $derived(twist === 'mirror');
	const scenario = $derived([power, twist].filter(Boolean).join('+'));
	const moves = $derived(scenarios.some((s: any) => s.moves && (s.id === power || s.id === twist)));
	const steps = $derived(
		mirror
			? ['Scenario', 'Origin']
			: truekin
				? ['Scenario', 'Genotype', 'Species', 'Caste', 'Attributes', 'Origin']
				: ['Scenario', 'Genotype', 'Species', 'Calling', 'Attributes', options.powersName, 'Origin']
	);
	const cur = $derived(steps[step]);
	const callingList = $derived(truekin ? options.castes : options.callings);
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
	const attrTotal = $derived(options.attrPoints + (truekin ? 2 : 0));
	const attrLeft = $derived(attrTotal - attrSpent);
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
		if (attrTotal - next.reduce((t, v) => t + cost(v), 0) < 0) return;
		attr = next;
	}
	/** attribute minimums the chosen calling asks for (score before the calling's own bonus) */
	const reqMiss = $derived(
		Object.entries(truekin ? {} : (calling?.req ?? {}))
			.map(([k, v]) => ({ i: +k, need: v as number, have: attr[+k] + (species?.Attr?.[+k] ?? 0) }))
			.filter((r) => r.have < r.need)
	);
	/** spend the points at random; weight 3 on the calling's own attributes when shaped */
	function roll(shaped: boolean) {
		let next = [12, 12, 12, 12, 12, 12];
		const left = () => attrTotal - next.reduce((t, v) => t + cost(v), 0);
		if (shaped) {
			for (const [k, v] of Object.entries(truekin ? {} : (calling?.req ?? {}))) {
				const i = +k;
				while (next[i] + (species?.Attr?.[i] ?? 0) < (v as number) && left() > 0) next[i]++;
			}
		}
		const w = (i: number) => (shaped && calling?.attr?.[i] ? 3 : 1);
		for (let guard = 0; guard < 400; guard++) {
			const open = next.map((v, i) => i).filter((i) => next[i] < 20 && cost(next[i] + 1) - cost(next[i]) <= left());
			if (!open.length) break;
			let r = Math.random() * open.reduce((t, i) => t + w(i), 0);
			let pick = open[0];
			for (const i of open) {
				r -= w(i);
				if (r <= 0) {
					pick = i;
					break;
				}
			}
			next[pick]++;
		}
		attr = next;
	}
	function setPower(id: string, lv: number) {
		const next = { ...powers, [id]: Math.max(0, Math.min(3, lv)) };
		if (!next[id]) delete next[id];
		const spent = Object.entries(next).reduce((t, [k, v]) => t + powerCost(k, v), 0);
		if (options.powerPoints - spent < 0) return;
		powers = next;
	}
	const allowed = (p: any) => p.defect || !morph || (morph === 'chimera' ? !p.mental : p.mental);
	const shownPowers = $derived(options.powers.filter(allowed));
	function pickGenotype(id: string) {
		genotype = id;
		calling = (id === 'truekin' ? options.castes : options.callings)[0];
		if (id === 'truekin') {
			powers = {};
			morph = '';
		}
	}
	function pickMorph(id: string) {
		morph = id;
		const next: Record<string, number> = {};
		for (const [k, v] of Object.entries(powers)) if (allowed(options.powers.find((x: any) => x.id === k))) next[k] = v;
		powers = next;
	}
	function pickSpecies(s: any) {
		species = s;
		const firstHome = options.starts.find((b: any) => b.Species === s.ID);
		if (firstHome) start = firstHome.ID;
	}
	const pickOne = <T,>(a: T[]): T => a[Math.floor(Math.random() * a.length)];
	/** Mirror Life: roll species, calling, attributes, powers and birthplace; the choice is then locked. */
	function rollMirror() {
		genotype = 'mutant';
		morph = '';
		species = pickOne(options.species);
		calling = pickOne(options.callings);
		const homes = options.starts.filter((b: any) => b.Species === species.ID);
		start = pickOne<any>(homes.length ? homes : options.starts).ID;
		roll(true);
		let next: Record<string, number> = {};
		const pool = options.powers.filter((p: any) => !p.defect).sort(() => Math.random() - 0.5);
		for (const p of pool) {
			const t = { ...next, [p.id]: 1 };
			if (options.powerPoints - Object.entries(t).reduce((n, [k, v]) => n + powerCost(k, v as number), 0) >= 0 && Math.random() < 0.5) next = t;
		}
		powers = next;
	}
	function pickScenario(s: any) {
		if (s.kind === 'power') power = power === s.id ? '' : s.id;
		else {
			twist = twist === s.id ? '' : s.id;
			if (twist === 'mirror') rollMirror();
		}
	}
	function begin() {
		onbegin({ name: name || 'Wanderer', species: species.ID, calling: calling.id, attr, powers: truekin ? {} : powers, start, genotype, morph: truekin ? '' : morph, gameMode, scenario, mods });
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
		{#if cur === 'Scenario'}
			<p class="hint">How does your story begin? Take the standard start, or one power start and one twist.</p>
			<div class="grid">
				<button class="card" class:on={!power && !twist} onclick={() => { power = ''; twist = ''; }}>
					<b>Standard</b>
					<small>An ordinary wanderer sets out from home.</small>
				</button>
			</div>
			<p class="hint" style="margin-top:12px">Power start (pick one)</p>
			<div class="grid">
				{#each scenarios.filter((s: any) => s.kind === 'power') as s (s.id)}
					<button class="card" class:on={power === s.id} onclick={() => pickScenario(s)}>
						<b>{s.name}</b>
						<small>{s.pitch}</small>
						<small class="neg">{s.down}</small>
					</button>
				{/each}
			</div>
			<p class="hint" style="margin-top:12px">Twist (optional)</p>
			<div class="grid">
				{#each scenarios.filter((s: any) => s.kind === 'twist') as s (s.id)}
					<button class="card" class:on={twist === s.id} onclick={() => pickScenario(s)}>
						<b>{s.name}</b>
						<small>{s.pitch}</small>
						<small class="neg">{s.down}</small>
					</button>
				{/each}
			</div>
			{#if challenges.length}
				<p class="hint" style="margin-top:12px">Challenge (optional, raises your score)</p>
				<div class="grid">
					{#each challenges as c (c.ID)}
						<label class="card" class:on={mods.includes(c.ID)}>
							<input type="checkbox" checked={mods.includes(c.ID)} onchange={() => toggleMod(c.ID)} />
							<b>{c.Name} <small>x{c.Mul}</small></b>
							<small>{c.Desc}</small>
						</label>
					{/each}
				</div>
			{/if}
			{#if mirror}
				<p class="hint" style="margin-top:12px">
					Fate dealt you: <b>{species?.Name} {calling?.name}</b>, {Object.keys(powers).length} {options.powersName.toLowerCase()}.
					<button onclick={rollMirror}>Re-roll</button>
				</p>
			{/if}
		{:else if cur === 'Genotype'}
			<p class="hint">What are you? True Kin are pure-strain: no mutations, 2 more attribute points, a caste, and body augments instead.</p>
			<div class="grid">
				{#each options.genotypes as g (g.ID)}
					<button class="card" class:on={genotype === g.ID} onclick={() => pickGenotype(g.ID)}>
						<b>{g.Name}</b>
						<small>{g.Desc}</small>
					</button>
				{/each}
			</div>
			{#if !truekin}
				<p class="hint" style="margin-top:12px">Morphotype</p>
				<div class="grid">
					{#each options.morphotypes as m (m.ID)}
						<button class="card" class:on={morph === m.ID} onclick={() => pickMorph(m.ID)}>
							<b>{m.Name}</b>
							<small>{m.Desc}</small>
						</button>
					{/each}
				</div>
			{/if}
		{:else if cur === 'Species'}
			<p class="hint">Peoples of this world. Common ones have homelands you can start from.</p>
			<div class="grid">
				{#each options.species as s (s.ID)}
					<button class="card" class:on={species?.ID === s.ID} onclick={() => pickSpecies(s)}>
						<Portrait species={s.ID} calling={truekin ? '' : calling?.id} seed={11} size={72} />
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
		{:else if cur === 'Calling' || cur === 'Caste'}
			<div class="grid">
				{#each callingList as c (c.id)}
					<button class="card" class:on={calling?.id === c.id} onclick={() => (calling = c)}>
						<Portrait species={species?.ID} calling={c.id} seed={11} size={72} />
						<b>{c.name}</b>
						<small>{c.desc}</small>
						<small class="mods">
							{#each Object.entries(c.attr) as [k, v] (k)}<span>{options.attrNames[+k].slice(0, 3).toUpperCase()} +{v}</span>{/each}
							{#each Object.entries(c.skills) as [k, v] (k)}<span class="plan">{k} {v}</span>{/each}
							{#each Object.entries(c.req ?? {}) as [k, v] (k)}<span class="neg">needs {options.attrNames[+k].slice(0, 3).toUpperCase()} {v}</span>{/each}
						</small>
					</button>
				{/each}
			</div>
		{:else if cur === 'Attributes'}
			<p class="hint">Points left: <b class:neg={attrLeft < 0}>{attrLeft}</b> · above 18 costs 2 per point · modifier = (score − 16) / 2</p>
			<p class="hint">
				<button onclick={() => roll(false)}>Roll at random</button>
				{#if !truekin}<button onclick={() => roll(true)}>Roll for my calling</button>{/if}
				<small>press again for a new roll</small>
			</p>
			{#each reqMiss as r (r.i)}
				<p class="hint"><b class="neg">{calling.name} needs {options.attrNames[r.i]} {r.need}</b> · you have {r.have}</p>
			{/each}
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
		{:else if cur === options.powersName}
			<p class="hint">{options.powersName} points left: <b class:neg={powerLeft < 0}>{powerLeft}</b> · levels up to 3 at creation (+1 each) · defects give points back</p>
			<div class="grid">
				{#each shownPowers as p (p.id)}
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
			{#if moves}<p class="hint">Your scenario chooses where you begin.</p>{/if}
			<p class="hint">Where do you set out from? Your home realm starts friendly; its enemies don't.</p>
			<label class="name">Mode
				<select bind:value={gameMode}>
					{#each options.modes as m (m.ID)}<option value={m.ID}>{m.Name}: {m.Desc}</option>{/each}
				</select>
			</label>
			{#if mirror}
				<p class="hint">Fate chose your birthplace: <b>{options.starts.find((b: any) => b.ID === start)?.Name}</b>.</p>
			{/if}
			<div class="list" class:locked={mirror}>
				{#each starts as b (b.ID)}
					<button class:on={start === b.ID} disabled={mirror} onclick={() => (start = b.ID)}>
						<b>{b.Name}</b> {b.Capital ? '★' : ''} <span class="dim">— {b.Realm} · {b.Culture}{b.Notables ? ` · ${b.Notables} notable${b.Notables > 1 ? 's' : ''}` : ''} {b.Species === species?.ID ? '· your people' : ''}</span>
					</button>
				{/each}
			</div>
		{/if}
	</main>

	<footer>
		<button onclick={oncancel}>Cancel</button>
		<span class="sum">{scenario ? scenario.replace('+', ' + ') + ' · ' : ''}{species?.Name} {calling?.name}{truekin ? '' : ` · ${Object.keys(powers).length} ${options.powersName.toLowerCase()}`}</span>
		{#if step < steps.length - 1}
			<button class="go" onclick={() => step++}>Next →</button>
		{:else}
			<button class="go" onclick={begin} disabled={attrLeft < 0 || powerLeft < 0 || reqMiss.length > 0}>Begin</button>
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
	select {
		font: inherit;
		background: #020a09;
		color: #fff;
		border: 1px solid #40a4b9;
		padding: 6px 8px;
		max-width: 100%;
	}
	input {
		font: inherit;
		background: #020a09;
		color: #fff;
		border: 1px solid #40a4b9;
		padding: 6px 8px;
	}
	.list.locked {
		display: none;
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
