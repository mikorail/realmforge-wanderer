<script lang="ts">
	/** A small popup of actions at the pointer: arrows move, Enter runs, Esc closes; outside click or scroll closes. */
	import { onMount } from 'svelte';
	import type { CtxItem } from './ctxmenu';

	let { items, x, y, onpick, onclose }: { items: CtxItem[]; x: number; y: number; onpick: (id: string) => void; onclose: () => void } = $props();

	let sel = $state(0);
	let box = $state<HTMLDivElement | null>(null);
	let pos = $state({ left: 0, top: 0 });

	function key(e: KeyboardEvent) {
		if (e.key === 'ArrowDown') sel = (sel + 1) % items.length;
		else if (e.key === 'ArrowUp') sel = (sel + items.length - 1) % items.length;
		else if (e.key === 'Home') sel = 0;
		else if (e.key === 'End') sel = items.length - 1;
		else if (e.key === 'Enter' || e.key === ' ') onpick(items[sel].id);
		else if (e.key === 'Escape' || e.key === 'ContextMenu') onclose();
		else return;
		e.preventDefault();
		e.stopPropagation();
	}
	function outside(e: Event) {
		if (box && e.target instanceof Node && box.contains(e.target)) return;
		onclose();
	}

	onMount(() => {
		pos = { left: x, top: y };
		// keep the popup on screen once its size is known
		requestAnimationFrame(() => {
			if (!box) return;
			const r = box.getBoundingClientRect();
			pos = { left: Math.max(4, Math.min(x, window.innerWidth - r.width - 4)), top: Math.max(4, Math.min(y, window.innerHeight - r.height - 4)) };
			box.focus();
		});
		// capture phase so the game never sees keys meant for the menu
		window.addEventListener('keydown', key, true);
		window.addEventListener('pointerdown', outside, true);
		window.addEventListener('wheel', onclose, { passive: true, capture: true });
		window.addEventListener('scroll', onclose, true);
		window.addEventListener('resize', onclose);
		window.addEventListener('blur', onclose);
		return () => {
			window.removeEventListener('keydown', key, true);
			window.removeEventListener('pointerdown', outside, true);
			window.removeEventListener('wheel', onclose, true);
			window.removeEventListener('scroll', onclose, true);
			window.removeEventListener('resize', onclose);
			window.removeEventListener('blur', onclose);
		};
	});
</script>

<div class="ctx" bind:this={box} role="menu" tabindex="-1" aria-label="Actions" style:left="{pos.left}px" style:top="{pos.top}px" oncontextmenu={(e) => e.preventDefault()}>
	{#each items as it, i (it.id)}
		<button class="row {it.tone ?? ''}" class:sel={i === sel} role="menuitem" onpointerenter={() => (sel = i)} onclick={() => onpick(it.id)}>
			<span>{it.label}</span>
			{#if it.hint}<kbd>{it.hint}</kbd>{/if}
		</button>
	{/each}
</div>

<style>
	.ctx {
		position: fixed;
		z-index: 40;
		min-width: 170px;
		max-width: 320px;
		padding: 3px;
		display: flex;
		flex-direction: column;
		font: 13px/1.3 'Share Tech Mono', 'DejaVu Sans Mono', Menlo, Consolas, monospace;
		background: rgba(8, 14, 16, 0.96);
		border: 1px solid #2a5a62;
		border-radius: 4px;
		box-shadow: 0 4px 14px rgba(0, 0, 0, 0.55);
		outline: none;
	}
	.row {
		display: flex;
		justify-content: space-between;
		gap: 14px;
		padding: 5px 8px;
		text-align: left;
		background: transparent;
		color: #cfd8d6;
		border: 0;
		border-radius: 2px;
		font: inherit;
		cursor: pointer;
		/* comfortable tap target on touch */
		min-height: 28px;
		align-items: center;
	}
	.row.sel {
		background: #17363c;
		color: #fff;
	}
	.row.hostile {
		color: #e88;
	}
	.row.warn {
		color: #e8c070;
	}
	kbd {
		font: inherit;
		color: #6f8f94;
	}
	@media (pointer: coarse) {
		.row {
			min-height: 38px;
		}
	}
</style>
