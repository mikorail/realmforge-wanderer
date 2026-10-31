<script lang="ts">
	/** Modal settings pane. Every change is passed up through onchange and applied live by the game screen. */
	import { onMount } from 'svelte';
	import { AUTOSAVE_OPTIONS, SCALE_OPTIONS, type PaletteName, type Settings } from './settings';

	let { settings, onchange, onclose }: { settings: Settings; onchange: (s: Settings) => void; onclose: () => void } = $props();

	let dialog = $state<HTMLDivElement | null>(null);
	let firstControl = $state<HTMLInputElement | null>(null);

	onMount(() => firstControl?.focus());

	function set<K extends keyof Settings>(key: K, value: Settings[K]) {
		onchange({ ...settings, [key]: value });
	}

	const PALETTE_OPTIONS: [PaletteName, string][] = [
		['qud', 'Qud (default)'],
		['classic', 'Classic terminal'],
		['contrast', 'High contrast'],
	];

	const KEYS: [string, string][] = [
		['Move', 'arrow keys or numpad 1–9'],
		['Inventory', 'i'],
		['Character', 'c'],
		['Journal', 'j'],
		['Reputation', 'r'],
		['Powers', 'a'],
		['Fire · Throw', 'f · t'],
		['Search', 's'],
		['Look', 'x'],
		['World map', 'w'],
		['Stairs', '< · >'],
		['Rest', 'R'],
		['Save', 'S'],
		['Save & quit', 'Esc'],
		['Settings', '`'],
	];

	/** Keys stay in the dialog: nothing typed here reaches the game, and Esc closes this pane instead of quitting. */
	function onkeydown(e: KeyboardEvent) {
		if (e.key === 'Escape') return; // handled once on the window
		e.stopPropagation();
		if (e.key === 'Tab' && dialog) {
			const items = Array.from(dialog.querySelectorAll<HTMLElement>('button, input, select')).filter(
				(el) => !(el as HTMLInputElement).disabled,
			);
			if (!items.length) return;
			const i = items.indexOf(document.activeElement as HTMLElement);
			const last = items.length - 1;
			if (e.shiftKey && i <= 0) {
				e.preventDefault();
				items[last].focus();
			} else if (!e.shiftKey && i === last) {
				e.preventDefault();
				items[0].focus();
			}
		}
	}
</script>

<svelte:window
	onkeydown={(e) => {
		// Esc closes the pane even when focus has left it (the game screen ignores keys while it is open)
		if (e.key === 'Escape') {
			e.preventDefault();
			onclose();
		}
	}}
/>

<div class="backdrop">
	<div
		class="dialog"
		role="dialog"
		aria-modal="true"
		aria-labelledby="settings-title"
		bind:this={dialog}
		{onkeydown}
	>
		<header>
			<h2 id="settings-title">Settings</h2>
			<button class="close" onclick={onclose} aria-label="Close settings">✕</button>
		</header>

		<h3>Sound</h3>
		<label class="row">
			<span>Music</span>
			<input
				type="checkbox"
				bind:this={firstControl}
				checked={settings.music}
				onchange={(e) => set('music', e.currentTarget.checked)}
			/>
		</label>
		<label class="row" class:off={!settings.music}>
			<span>Volume <output>{Math.round(settings.musicVolume * 100)}%</output></span>
			<input
				type="range"
				min="0"
				max="1"
				step="0.05"
				value={settings.musicVolume}
				disabled={!settings.music}
				oninput={(e) => set('musicVolume', Number(e.currentTarget.value))}
			/>
		</label>

		<h3>Display</h3>
		<label class="row">
			<span>Cell size</span>
			<select value={String(settings.scale)} onchange={(e) => set('scale', e.currentTarget.value === 'auto' ? 'auto' : Number(e.currentTarget.value))}>
				<option value="auto">Auto fit</option>
				{#each SCALE_OPTIONS as s (s)}
					<option value={String(s)}>{Math.round(s * 100)}%</option>
				{/each}
			</select>
		</label>
		<label class="row">
			<span>Palette</span>
			<select value={settings.palette} onchange={(e) => set('palette', e.currentTarget.value as PaletteName)}>
				{#each PALETTE_OPTIONS as [id, label] (id)}
					<option value={id}>{label}</option>
				{/each}
			</select>
		</label>
		<label class="row">
			<span>Animations</span>
			<input type="checkbox" checked={settings.animations} onchange={(e) => set('animations', e.currentTarget.checked)} />
		</label>
		<label class="row">
			<span>Show hints</span>
			<input type="checkbox" checked={settings.showHints} onchange={(e) => set('showHints', e.currentTarget.checked)} />
		</label>

		<h3>Saving</h3>
		<label class="row">
			<span>Autosave</span>
			<select value={String(settings.autosaveEvery)} onchange={(e) => set('autosaveEvery', Number(e.currentTarget.value))}>
				{#each AUTOSAVE_OPTIONS as n (n)}
					<option value={String(n)}>every {n} keys</option>
				{/each}
			</select>
		</label>

		<h3>Keys</h3>
		<dl class="keys">
			{#each KEYS as [action, key] (action)}
				<dt>{action}</dt>
				<dd><kbd>{key}</kbd></dd>
			{/each}
		</dl>
	</div>
</div>

<style>
	.backdrop {
		position: fixed;
		inset: 0;
		z-index: 100;
		display: flex;
		align-items: center;
		justify-content: center;
		padding: 16px;
		box-sizing: border-box;
		background: rgba(2, 10, 9, 0.72);
	}
	.dialog {
		width: min(460px, 100%);
		max-height: 100%;
		overflow: auto;
		box-sizing: border-box;
		padding: 14px 18px 18px;
		background: #041312;
		color: var(--ink);
		border: 1px solid #40a4b9;
		box-shadow: 0 0 0 1px #155352, 0 12px 40px rgba(0, 0, 0, 0.6);
	}
	header {
		display: flex;
		align-items: center;
		justify-content: space-between;
	}
	h2 {
		color: #cfc041;
		font-weight: 400;
		font-size: 18px;
		letter-spacing: 0.2em;
		margin: 0;
	}
	h3 {
		color: #77bfcf;
		font-weight: 400;
		font-size: 14px;
		border-bottom: 1px solid #155352;
		padding-bottom: 4px;
		margin: 18px 0 6px;
	}
	.close {
		padding: 2px 8px;
	}
	.row {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 12px;
		min-height: 30px;
		font-size: 14px;
	}
	.row.off {
		opacity: 0.45;
	}
	.row span {
		display: flex;
		gap: 8px;
		align-items: baseline;
	}
	output {
		color: #3f7f7a;
		font-size: 12px;
	}
	select {
		font: inherit;
		font-size: 13px;
		background: #0d2b29;
		color: var(--ink);
		border: 1px solid #155352;
		padding: 3px 6px;
		max-width: 60%;
	}
	input[type='checkbox'] {
		width: 16px;
		height: 16px;
		accent-color: #40a4b9;
		cursor: pointer;
	}
	input[type='range'] {
		width: 55%;
		accent-color: #40a4b9;
	}
	input:disabled {
		cursor: not-allowed;
	}
	.keys {
		display: grid;
		grid-template-columns: auto 1fr;
		gap: 4px 16px;
		margin: 0;
		font-size: 13px;
	}
	.keys dt {
		color: var(--ink);
	}
	.keys dd {
		margin: 0;
	}
	kbd {
		font: inherit;
		color: #cfc041;
		background: #0d2b29;
		border: 1px solid #155352;
		padding: 0 5px;
	}
</style>
