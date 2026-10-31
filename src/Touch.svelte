<script lang="ts">
	/** On-screen controls for touch devices: an 8-way d-pad with wait, and a bar of action keys. */
	let { send, height = $bindable(0) }: { send: (key: string) => void; height?: number } = $props();

	const PAD: [string, string, string][] = [
		['↖', 'N7', 'Move northwest'], ['↑', 'ArrowUp', 'Move north'], ['↗', 'N9', 'Move northeast'],
		['←', 'ArrowLeft', 'Move west'], ['·', '.', 'Wait'], ['→', 'ArrowRight', 'Move east'],
		['↙', 'N1', 'Move southwest'], ['↓', 'ArrowDown', 'Move south'], ['↘', 'N3', 'Move southeast'],
	];
	const ACTIONS: [string, string, string][] = [
		['i', 'i', 'Inventory'], ['a', 'a', 'Powers'], ['g', 'g', 'Get'], ['<', '<', 'Stairs up'], ['>', '>', 'Stairs down'], ['c', 'c', 'Character'],
		['f', 'f', 'Fire'], ['x', 'x', 'Look'], ['t', 't', 'Throw'], ['s', 's', 'Search'], ['R', 'R', 'Rest'], ['S', 'S', 'Save'],
		['Esc', 'Escape', 'Menu'], ['Ent', 'Enter', 'Enter'], ['Tab', 'Tab', 'Tab'], ['y', 'y', 'Yes'], ['n', 'n', 'No'],
	];

	let timer: ReturnType<typeof setTimeout> | null = null;
	let rep: ReturnType<typeof setInterval> | null = null;
	function stop() {
		if (timer) clearTimeout(timer);
		if (rep) clearInterval(rep);
		timer = rep = null;
	}
	/** d-pad: press sends at once, holding repeats */
	function down(e: PointerEvent, key: string) {
		e.preventDefault();
		stop();
		send(key);
		timer = setTimeout(() => {
			rep = setInterval(() => send(key), 130);
		}, 380);
	}
</script>

<svelte:window onpointerup={stop} onpointercancel={stop} onblur={stop} />

<div class="touch" bind:clientHeight={height} role="group" aria-label="Touch controls">
	<div class="pad">
		{#each PAD as [label, key, name] (key)}
			<button type="button" aria-label={name} onpointerdown={(e) => down(e, key)} onpointerleave={stop} onclick={(e) => e.detail === 0 && send(key)} oncontextmenu={(e) => e.preventDefault()}>{label}</button>
		{/each}
	</div>
	<div class="acts">
		{#each ACTIONS as [label, key, name] (key)}
			<button type="button" aria-label={name} title={name} onclick={() => send(key)} oncontextmenu={(e) => e.preventDefault()}>{label}</button>
		{/each}
	</div>
</div>

<style>
	.touch {
		position: fixed;
		left: 0;
		right: 0;
		bottom: 0;
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 10px;
		box-sizing: border-box;
		padding: 6px max(8px, env(safe-area-inset-right)) max(6px, env(safe-area-inset-bottom)) max(8px, env(safe-area-inset-left));
		background: #020a09;
		border-top: 1px solid #155352;
		user-select: none;
		-webkit-user-select: none;
		-webkit-touch-callout: none;
		touch-action: manipulation;
	}
	.pad {
		display: grid;
		grid-template-columns: repeat(3, 44px);
		grid-auto-rows: 44px;
		gap: 3px;
		flex: none;
	}
	.acts {
		display: grid;
		grid-template-columns: repeat(6, minmax(34px, 44px));
		grid-auto-rows: 40px;
		gap: 3px;
		justify-content: end;
		flex: 1;
	}
	button {
		padding: 0;
		font-size: 16px;
		min-width: 0;
		touch-action: none;
	}
	.acts button {
		font-size: 14px;
		touch-action: manipulation;
	}
	@media (max-width: 480px) {
		.pad {
			grid-template-columns: repeat(3, 40px);
			grid-auto-rows: 40px;
		}
		.acts {
			grid-template-columns: repeat(6, minmax(30px, 1fr));
			grid-auto-rows: 38px;
		}
	}
</style>
