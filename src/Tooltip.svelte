<script lang="ts" module>
	import type { QudActor } from './actorsprites';
	import { parseTip, lineKind, parseHere, hereLabels, type EngineTip } from './tips';
	import type { Keymap } from './keymap';

	export interface Tip {
		title: string;
		sub?: string;
		tone?: 'hostile' | 'friendly' | 'neutral';
		/** detailed engine lines (leading + good, - bad, ! warning) */
		lines?: string[];
	}

	/** map glyph to a short name, for terrain and things with no actor record */
	const LEGEND: Record<string, string> = {
		'#': 'wall', '+': 'door', '/': 'open door', '"': 'tall grass or crops', '.': 'ground', '∙': 'road', '·': 'floor',
		'♣': 'tree', 'τ': 'dead tree', '♠': 'brush or mushroom', '~': 'water', '≈': 'deep water or lava', '>': 'way down', '<': 'way up',
		'↑': 'updraft', 'o': 'well', '╥': 'altar', 'π': 'stall or anvil', '¶': 'inscribed stone', '%': 'rubble or ore', '=': 'bridge or floor',
		'O': 'pillar', '*': 'crystal or lantern', 'Ψ': 'brazier', '^': 'fire or trap', '▲': 'cliff', '║': 'waterfall', '░': 'ice, spray or cloud',
		'▒': 'glacier', '⊞': 'window', '∏': 'gate arch', '╫': 'portcullis', '≡': 'shelf or stands', 'Ω': 'statue', 'Ж': 'throne', '■': 'crates',
		'▬': 'sarcophagus', 'X': 'barricade', '¤': 'geyser or chandelier', 'Φ': 'machinery', '◊': 'rift', 'Ø': 'rift gate', ':': 'sand',
		',': 'moss or bones', '$': 'gold', '!': 'potion', '?': 'scroll', '[': 'armour', '(': 'tool', '|': 'weapon', '-': 'weapon', '\\': 'weapon',
	};

	/** what is at map cell (gx, gy), or null */
	export function describe(glyphs: string[], w: number, actors: QudActor[] | undefined, gx: number, gy: number): Tip | null {
		const a = actors?.find((q) => q.x === gx && q.y === gy);
		if (a) {
			const name = a.nm || a.t || a.r || 'someone';
			const what = [a.s, a.r && a.r !== a.nm ? a.r : '', a.ti].filter((s, i, l) => s && l.indexOf(s) === i && s !== name).join(', ');
			if (a.p) return { title: `${name} (you)`, sub: a.hp !== undefined ? `health ${a.hp}%` : undefined, tone: 'friendly' };
			const mood = a.ho ? 'hostile' : 'friendly';
			const hp = a.hp !== undefined && a.hp < 100 ? `, health ${a.hp}%` : '';
			return { title: name, sub: `${what ? what + ' — ' : ''}${mood}${hp}`, tone: a.ho ? 'hostile' : 'friendly' };
		}
		const g = glyphs[gy * w + gx];
		if (!g || g === ' ') return null;
		const n = LEGEND[g];
		return n ? { title: n, tone: 'neutral' } : null;
	}
</script>

<script lang="ts">
	// Optional wiring (all may be left out): `engineTip` = frame.tip (cursor / inventory / trade detail, shown as a panel at the
	// canvas's lower right), `deep` = (x, y) => qud.tipAt(x, y) string for richer hover text, `here` = frame.here and `keymap`
	// = prefs.keymap so the "What can I do here" screen shows the player's rebound keys.
	let { canvas, cell, lookup, engineTip = null, deep = null, here = null, keymap = {} }: {
		canvas: HTMLCanvasElement | null;
		cell: () => { cw: number; chh: number };
		lookup: (gx: number, gy: number) => Tip | null;
		engineTip?: string | EngineTip | null;
		deep?: ((gx: number, gy: number) => string | null | undefined) | null;
		here?: string | { cells: { x: number; y: number; w: number; k: string }[] } | null;
		keymap?: Keymap;
	} = $props();

	let panel = $derived(parseTip(engineTip));
	let overlay = $derived(hereLabels(parseHere(here), keymap));
	let box = $state({ left: 0, top: 0, w: 0, h: 0, cw: 8, chh: 16 });
	function measure() {
		if (!canvas) return;
		const r = canvas.getBoundingClientRect();
		const { cw, chh } = cell();
		box = { left: r.left, top: r.top, w: r.width, h: r.height, cw, chh };
	}
	$effect(() => {
		void panel;
		void overlay;
		measure();
	});

	let tip = $state<Tip | null>(null);
	let at = $state({ x: 0, y: 0 });
	let timer: ReturnType<typeof setTimeout> | null = null;
	let lastCell = '';

	function hide() {
		if (timer) clearTimeout(timer);
		timer = null;
		tip = null;
		lastCell = '';
	}
	function move(e: PointerEvent) {
		if (e.pointerType !== 'mouse' || !canvas) return;
		const r = canvas.getBoundingClientRect();
		const { cw, chh } = cell();
		const gx = Math.floor((e.clientX - r.left) / cw), gy = Math.floor((e.clientY - r.top) / chh);
		const key = `${gx},${gy}`;
		at = { x: e.clientX, y: e.clientY };
		if (key === lastCell) return;
		hide();
		lastCell = key;
		if (gx < 0 || gy < 0 || gx >= 80 || gy >= 25) return;
		timer = setTimeout(() => {
			timer = null;
			tip = lookup(gx, gy);
			const d = parseTip(deep ? deep(gx, gy) : null);
			if (d) tip = { title: d.t, sub: d.l.map((x) => lineKind(x).text).join(' · '), tone: d.tone ?? tip?.tone ?? 'neutral', lines: d.l };
		}, 250);
	}

	$effect(() => {
		const c = canvas;
		if (!c) return;
		c.addEventListener('pointermove', move);
		c.addEventListener('pointerleave', hide);
		c.addEventListener('pointerdown', hide);
		window.addEventListener('keydown', hide);
		return () => {
			c.removeEventListener('pointermove', move);
			c.removeEventListener('pointerleave', hide);
			c.removeEventListener('pointerdown', hide);
			window.removeEventListener('keydown', hide);
			hide();
		};
	});
</script>

{#if tip}
	<div class="tip {tip.tone ?? 'neutral'}" style:left="{Math.min(at.x + 14, (typeof window !== 'undefined' ? window.innerWidth : 9999) - 230)}px" style:top="{at.y + 16}px" role="tooltip">
		<b>{tip.title}</b>
		{#if tip.lines}
			{#each tip.lines as ln}{@const k = lineKind(ln)}<span class="l {k.kind}">{k.text}</span>{/each}
		{:else if tip.sub}<span>{tip.sub}</span>{/if}
	</div>
{/if}

{#if panel}
	<div class="tip panel {panel.tone ?? 'neutral'}" style:right="{Math.max(8, (typeof window !== 'undefined' ? window.innerWidth : 0) - box.left - box.w + 8)}px" style:bottom="{Math.max(8, (typeof window !== 'undefined' ? window.innerHeight : 0) - box.top - box.h + box.chh * 7)}px" role="tooltip">
		<b>{panel.t}</b>
		{#each panel.l as ln}{@const k = lineKind(ln)}<span class="l {k.kind}">{k.text}</span>{/each}
	</div>
{/if}

{#each overlay as c}
	<span class="rebound" style:left="{box.left + c.x * box.cw}px" style:top="{box.top + c.y * box.chh}px" style:width="{c.w * box.cw}px" style:height="{box.chh}px">[{c.label}]</span>
{/each}

<style>
	.tip {
		position: fixed;
		z-index: 20;
		max-width: 220px;
		padding: 3px 8px;
		font: 12px/1.35 'Share Tech Mono', 'DejaVu Sans Mono', Menlo, Consolas, monospace;
		background: rgba(8, 14, 16, 0.92);
		color: #cfd8d6;
		border: 1px solid #2a5a62;
		border-radius: 3px;
		pointer-events: none;
		display: flex;
		flex-direction: column;
	}
	.tip b {
		font-weight: normal;
		color: #fff;
	}
	.tip span {
		color: #9ab;
	}
	.tip.panel {
		max-width: 340px;
		padding: 5px 9px;
		gap: 1px;
	}
	.tip .l.good {
		color: #8d9;
	}
	.tip .l.bad {
		color: #e88;
	}
	.tip .l.warn {
		color: #ec7;
	}
	.rebound {
		position: fixed;
		z-index: 19;
		background: #0b1316;
		color: #e8d36a;
		font: 12px/1.2 'Share Tech Mono', 'DejaVu Sans Mono', Menlo, Consolas, monospace;
		pointer-events: none;
		overflow: hidden;
		white-space: nowrap;
	}
	.tip.hostile {
		border-color: #a33;
	}
	.tip.hostile span {
		color: #e88;
	}
	.tip.friendly span {
		color: #8d9;
	}
</style>
