<script lang="ts">
	/** Key binding editor, opened from Settings. Edits the `keymap` pref (overrides only, see keymap.ts) and reports every change up. */
	import { onMount } from 'svelte';
	import {
		ACTIONS, CATEGORIES, PRESETS, applyPreset, bindKey, actionById, exportKeymap, findConflicts, holders, importKeymap,
		isBindable, keyLabel, keysFor, physKey, presetOf, resetAction, resetAll, unbindKey, type Keymap,
	} from './keymap';

	let { keymap, onchange, onclose }: { keymap: Keymap; onchange: (k: Keymap) => void; onclose: () => void } = $props();

	let capture = $state<{ id: string; mode: 'add' | 'set' } | null>(null);
	let pending = $state<{ id: string; key: string; mode: 'add' | 'set'; with: string[] } | null>(null);
	let text = $state('');
	let note = $state('');
	let dialog = $state<HTMLDivElement | null>(null);

	const conflicts = $derived(findConflicts(keymap));
	const preset = $derived(presetOf(keymap));
	const names = (ids: string[]) => ids.map((i) => actionById(i)?.name ?? i).join(', ');

	function startCapture(id: string, mode: 'add' | 'set') {
		pending = null;
		note = '';
		capture = { id, mode };
	}

	function got(phys: string) {
		const c = capture;
		if (!c) return;
		capture = null;
		if (!isBindable(phys)) {
			note = 'That key is reserved.';
			return;
		}
		const others = holders(keymap, phys, c.id);
		if (others.length) pending = { id: c.id, key: phys, mode: c.mode, with: others };
		else onchange(bindKey(keymap, c.id, phys, c.mode));
	}

	function confirmTake() {
		if (pending) onchange(bindKey(keymap, pending.id, pending.key, pending.mode));
		pending = null;
	}

	/** window capture phase: while binding, swallow the key so neither the game nor Settings sees it */
	function onkey(e: KeyboardEvent) {
		if (capture) {
			e.preventDefault();
			e.stopPropagation();
			if (e.key === 'Escape') {
				capture = null;
				return;
			}
			if (e.key === 'Shift' || e.key === 'Control' || e.key === 'Alt' || e.key === 'Meta') return;
			if (e.metaKey || e.ctrlKey || e.altKey) return;
			got(physKey(e));
			return;
		}
		if (e.key === 'Escape') {
			e.preventDefault();
			e.stopPropagation();
			if (pending) pending = null;
			else onclose();
			return;
		}
		// keep typing and Tab inside this dialog away from the game
		e.stopPropagation();
	}
	onMount(() => {
		window.addEventListener('keydown', onkey, true);
		return () => window.removeEventListener('keydown', onkey, true);
	});

	function doExport() {
		text = exportKeymap(keymap);
		note = 'Copy this text to keep or share your bindings.';
		void navigator.clipboard?.writeText(text).then(() => (note = 'Copied to the clipboard.'), () => {});
	}
	function doImport() {
		const r = importKeymap(text);
		if (!r.ok) {
			note = r.error;
			return;
		}
		onchange(r.keymap);
		note = r.conflicts.length ? `Imported with ${r.conflicts.length} conflict(s), shown above.` : 'Imported.';
	}
</script>

<div class="backdrop">
	<div class="dialog" role="dialog" aria-modal="true" aria-labelledby="kb-title" bind:this={dialog}>
		<header>
			<h2 id="kb-title">Key bindings</h2>
			<button class="close" onclick={onclose} aria-label="Close key bindings">✕</button>
		</header>

		<div class="row">
			<span>Preset</span>
			<select value={preset} onchange={(e) => { if (e.currentTarget.value) onchange(applyPreset(e.currentTarget.value)); pending = null; capture = null; }}>
				{#if !preset}<option value="">Custom</option>{/if}
				{#each PRESETS as p (p.id)}
					<option value={p.id}>{p.name}</option>
				{/each}
			</select>
		</div>
		{#if preset}
			<p class="hint">{PRESETS.find((p) => p.id === preset)?.note}</p>
		{/if}
		<div class="row">
			<span>Everything back to stock</span>
			<button onclick={() => { onchange(resetAll()); pending = null; capture = null; }}>Reset all</button>
		</div>

		{#if conflicts.length}
			<div class="warn" role="alert">
				{#each conflicts as c (c.key)}
					<div><kbd>{keyLabel(c.key)}</kbd> is used by {names(c.actions)}. The first listed action wins.</div>
				{/each}
			</div>
		{/if}
		{#if pending}
			<div class="warn" role="alert">
				<kbd>{keyLabel(pending.key)}</kbd> is already used by {names(pending.with)}.
				<button onclick={confirmTake}>Take it</button>
				<button onclick={() => (pending = null)}>Cancel</button>
			</div>
		{/if}
		{#if capture}
			<div class="cap" role="status">Press a key for “{actionById(capture.id)?.name}” (Esc cancels)…</div>
		{/if}
		{#if note}<p class="hint">{note}</p>{/if}

		{#each CATEGORIES as [cat, label] (cat)}
			<h3>{label}</h3>
			<div class="list">
				{#each ACTIONS.filter((x) => x.cat === cat) as act (act.id)}
					{@const ks = keysFor(keymap, act.id)}
					{@const changed = Array.isArray(keymap[act.id])}
					<div class="item" class:cur={capture?.id === act.id}>
						<span class="name">{act.name}{#if changed}<i title="changed from default"> •</i>{/if}</span>
						<span class="keys">
							{#each ks as k (k)}
								{#if act.fixed}
									<kbd>{keyLabel(k)}</kbd>
								{:else}
									<button class="chip" title="Remove {keyLabel(k)}" aria-label="Remove {keyLabel(k)} from {act.name}" onclick={() => onchange(unbindKey(keymap, act.id, k))}><kbd>{keyLabel(k)}</kbd></button>
								{/if}
							{:else}
								<em>unbound</em>
							{/each}
							{#if !act.fixed}
								<button onclick={() => startCapture(act.id, 'set')} title="Replace with a new key">Set</button>
								<button onclick={() => startCapture(act.id, 'add')} title="Add another key" aria-label="Add another key for {act.name}">+</button>
								{#if changed}<button onclick={() => onchange(resetAction(keymap, act.id))} title="Back to default" aria-label="Reset {act.name}">↺</button>{/if}
							{/if}
						</span>
					</div>
				{/each}
			</div>
		{/each}

		<h3>Share</h3>
		<textarea rows="5" spellcheck="false" bind:value={text} placeholder="Paste bindings JSON here, or press Export"></textarea>
		<div class="row">
			<span></span>
			<span class="btns"><button onclick={doExport}>Export</button><button onclick={doImport} disabled={!text.trim()}>Import</button></span>
		</div>
		<p class="hint">Menus, shops, dialogue and every text prompt (console, palette, naming) always take raw keys. Bindings apply on the map, look, aim and world-map screens.</p>
	</div>
</div>

<style>
	.backdrop {
		position: fixed;
		inset: 0;
		z-index: 110;
		display: flex;
		align-items: center;
		justify-content: center;
		padding: 16px;
		box-sizing: border-box;
		background: rgba(2, 10, 9, 0.72);
	}
	.dialog {
		width: min(520px, 100%);
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
	.item {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 10px;
		min-height: 28px;
		font-size: 13px;
		padding: 1px 4px;
	}
	.item.cur {
		background: #0d2b29;
		outline: 1px solid #cfc041;
	}
	.name i {
		color: #cfc041;
		font-style: normal;
	}
	.keys {
		display: flex;
		flex-wrap: wrap;
		gap: 4px;
		justify-content: flex-end;
		align-items: center;
	}
	.keys em {
		color: #3f7f7a;
		font-size: 12px;
	}
	.btns {
		display: flex;
		gap: 6px;
	}
	.chip {
		padding: 0;
		border: 0;
		background: none;
		cursor: pointer;
	}
	button {
		font: inherit;
		font-size: 12px;
	}
	.warn {
		margin: 8px 0;
		padding: 6px 8px;
		font-size: 13px;
		color: #ff8080;
		border: 1px solid #6b2a2a;
		background: #2a0000;
	}
	.cap {
		margin: 8px 0;
		padding: 6px 8px;
		font-size: 13px;
		color: #cfc041;
		border: 1px solid #cfc041;
	}
	.hint {
		margin: 6px 0;
		font-size: 12px;
		color: #3f7f7a;
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
	textarea {
		width: 100%;
		box-sizing: border-box;
		font: inherit;
		font-size: 12px;
		background: #0d2b29;
		color: var(--ink);
		border: 1px solid #155352;
		padding: 6px;
		resize: vertical;
	}
	kbd {
		font: inherit;
		font-size: 13px;
		color: #cfc041;
		background: #0d2b29;
		border: 1px solid #155352;
		padding: 0 5px;
	}
</style>
