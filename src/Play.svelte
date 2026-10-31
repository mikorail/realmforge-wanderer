<script lang="ts">
	/** The game screen: draws the engine's character grid on a canvas and forwards keys and clicks. */
	import { onMount, onDestroy } from 'svelte';
	import type { QudApi, QudFrame } from './qud';
	import { saveRun, deleteRun } from './store';
	import { recordDeeds } from './chronicle';
	import { createMusic, type MusicHandle } from './music';
	import { loadSettings, saveSettings, PALETTES, type Settings as Prefs, type PaletteName } from './settings';
	import Settings from './Settings.svelte';
	import Touch from './Touch.svelte';

	let { api, first, world, onquit }: { api: QudApi; first: QudFrame; world: string | null; onquit: () => void } = $props();

	let canvas = $state<HTMLCanvasElement | null>(null);
	let wrap = $state<HTMLDivElement | null>(null);
	let gear = $state<HTMLButtonElement | null>(null);
	const enginePalette: string[] = JSON.parse(api.palette());
	let prefs = $state<Prefs>(loadSettings());
	let settingsOpen = $state(false);
	let palette: string[] = paletteFor(prefs.palette);
	let frame: QudFrame = first;
	let cw = 9;
	let chh = 16;
	let keys = 0;
	let worldPending = world;
	const music: MusicHandle = createMusic();
	// touch layout: coarse pointer or a narrow window
	let touch = $state(false);
	let ctlH = $state(0);
	const LONG_PRESS_MS = 500;
	let pressTimer: ReturnType<typeof setTimeout> | null = null;
	let pressAt: { x: number; y: number } | null = null;
	let longFired = false;

	function paletteFor(p: PaletteName): string[] {
		return p === 'qud' ? enginePalette : PALETTES[p];
	}

	function applyAudio() {
		music.setEnabled(prefs.music);
		music.setVolume(prefs.musicVolume);
		music.setStyle(prefs.musicStyle);
	}

	/** Host settings the engine reads (g.Pref). */
	function applyEngine() {
		api.pref('anim', prefs.animations);
		api.pref('hints', prefs.showHints);
	}

	/** Called by the settings pane on every change: store it, then re-apply what it touches. */
	function applyPrefs(next: Prefs) {
		prefs = next;
		saveSettings(next);
		palette = paletteFor(next.palette);
		applyAudio();
		applyEngine();
		resize();
	}

	async function persist() {
		const meta = JSON.parse(api.meta());
		if (frame.dead) {
			await recordDeeds(api, meta.seed);
			await deleteRun(meta.seed);
			return;
		}
		await saveRun({ ...meta, saved: Date.now() }, api.save(), worldPending ?? undefined);
		worldPending = null;
		// deeds go to the chronicle store and, when opened from Realmforge, back to it live
		void recordDeeds(api, meta.seed);
	}

	async function quit() {
		await persist();
		onquit();
	}

	const NUMPAD: Record<string, string> = { Numpad1: 'N1', Numpad2: 'N2', Numpad3: 'N3', Numpad4: 'N4', Numpad5: 'N5', Numpad6: 'N6', Numpad7: 'N7', Numpad8: 'N8', Numpad9: 'N9' };
	function onkey(e: KeyboardEvent) {
		// while the settings pane is open it owns the keyboard
		if (settingsOpen) return;
		if (e.metaKey || e.ctrlKey || e.altKey) return;
		if (e.code === 'Backquote' || e.key === '`') {
			e.preventDefault();
			settingsOpen = true;
			return;
		}
		const k = NUMPAD[e.code] ?? e.key;
		if (k === 'Shift' || k === 'Control' || k === 'Alt' || k === 'Meta' || k === 'CapsLock') return;
		e.preventDefault();
		send(k);
	}
	function send(k: string) {
		const f = api.key(k);
		if (!f || 'error' in f) return;
		frame = f;
		music.setState(f.music);
		draw();
		keys++;
		if (f.save || keys % prefs.autosaveEvery === 0) void persist();
		if (f.exit) void quit();
	}
	function cellAt(e: { clientX: number; clientY: number }): string {
		const r = canvas!.getBoundingClientRect();
		return `click:${Math.floor((e.clientX - r.left) / cw)},${Math.floor((e.clientY - r.top) / chh)}`;
	}
	function clearPress() {
		if (pressTimer) clearTimeout(pressTimer);
		pressTimer = null;
		pressAt = null;
	}
	/** Long-press on the map = look at that spot: 'x' then a click on the target. */
	function onpointerdown(e: PointerEvent) {
		if (!canvas || e.pointerType === 'mouse') return;
		longFired = false;
		clearPress();
		pressAt = { x: e.clientX, y: e.clientY };
		const pt = { clientX: e.clientX, clientY: e.clientY };
		pressTimer = setTimeout(() => {
			pressTimer = null;
			longFired = true;
			send('x');
			send(cellAt(pt));
			navigator.vibrate?.(15);
		}, LONG_PRESS_MS);
	}
	function onpointermove(e: PointerEvent) {
		if (pressAt && Math.hypot(e.clientX - pressAt.x, e.clientY - pressAt.y) > 10) clearPress();
	}
	function onclick(e: MouseEvent) {
		if (!canvas) return;
		if (longFired) {
			longFired = false;
			return;
		}
		send(cellAt(e));
	}

	function resize() {
		if (!canvas || !wrap) return;
		const W = wrap.clientWidth, H = wrap.clientHeight;
		const fit = Math.max(8, Math.floor(Math.min(H / frame.h, W / (frame.w * 0.6))));
		const scale = prefs.scale;
		chh = scale === 'auto' ? fit : Math.max(8, Math.floor(fit * scale));
		cw = Math.max(5, Math.floor(chh * 0.6));
		const dpr = window.devicePixelRatio || 1;
		canvas.width = frame.w * cw * dpr;
		canvas.height = frame.h * chh * dpr;
		canvas.style.width = `${frame.w * cw}px`;
		canvas.style.height = `${frame.h * chh}px`;
		canvas.getContext('2d')?.setTransform(dpr, 0, 0, dpr, 0, 0);
		draw();
	}

	function draw() {
		const f = frame;
		const ctx = canvas?.getContext('2d');
		if (!ctx) return;
		const glyphs = Array.from(f.g);
		ctx.font = `${Math.round(chh * 0.86)}px "Share Tech Mono", "DejaVu Sans Mono", Menlo, Consolas, monospace`;
		ctx.textAlign = 'center';
		ctx.textBaseline = 'middle';
		for (let i = 0; i < f.w * f.h; i++) {
			ctx.fillStyle = palette[f.b.charCodeAt(i) - 65];
			ctx.fillRect((i % f.w) * cw, Math.floor(i / f.w) * chh, cw + 0.5, chh + 0.5);
		}
		for (let i = 0; i < f.w * f.h; i++) {
			const g = glyphs[i];
			if (g === ' ') continue;
			const x = (i % f.w) * cw, y = Math.floor(i / f.w) * chh;
			ctx.fillStyle = palette[f.f.charCodeAt(i) - 65];
			if (g === '█') ctx.fillRect(x, y, cw + 0.5, chh + 0.5);
			else if (g === '─') ctx.fillRect(x, y + chh / 2, cw + 0.5, 1);
			else if (g === '│') ctx.fillRect(x + cw / 2, y, 1, chh + 0.5);
			else ctx.fillText(g, x + cw / 2, y + chh / 2 + 1);
		}
	}

	function closeSettings() {
		settingsOpen = false;
		gear?.focus();
	}

	$effect(() => {
		void ctlH;
		void touch;
		resize();
	});
	$effect(() => {
		if (!touch) return;
		const root = document.documentElement;
		const prev = root.style.overscrollBehavior;
		root.style.overscrollBehavior = 'none';
		return () => (root.style.overscrollBehavior = prev);
	});

	onMount(() => {
		const mq = window.matchMedia('(pointer: coarse), (max-width: 900px)');
		touch = mq.matches;
		const onmq = () => (touch = mq.matches);
		mq.addEventListener('change', onmq);
		applyAudio();
		applyEngine();
		music.setState(frame.music);
		resize();
		// the play area shrinks when touch controls appear: refit the grid whenever it changes size
		const ro = new ResizeObserver(() => resize());
		if (wrap) ro.observe(wrap);
		void persist();
		// fonts load after first paint: redraw once they are in
		void document.fonts?.ready.then(draw);
		return () => {
			mq.removeEventListener('change', onmq);
			ro.disconnect();
		};
	});
	onDestroy(() => {
		music.dispose();
		void persist();
	});
</script>

<svelte:window onkeydown={onkey} onresize={resize} />

<!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_static_element_interactions -->
<div class="screen" bind:this={wrap} onclick={onclick} {onpointerdown} {onpointermove} onpointerup={clearPress} onpointercancel={clearPress} oncontextmenu={(e) => touch && e.preventDefault()} style:bottom={touch ? `${ctlH}px` : '0'}>
	<canvas bind:this={canvas} aria-label="Game screen"></canvas>
</div>
{#if touch}
	<Touch {send} bind:height={ctlH} />
{/if}
<div class="chrome">
	<button class="gear" bind:this={gear} onclick={() => (settingsOpen = true)} title="Settings (`)" aria-label="Settings">⚙</button>
	<button class="quit" onclick={quit} title="Save and return to the title screen">Save &amp; quit</button>
</div>
{#if settingsOpen}
	<Settings settings={prefs} onchange={applyPrefs} onclose={closeSettings} />
{/if}

<style>
	.screen {
		position: fixed;
		inset: 0;
		display: flex;
		touch-action: manipulation;
		-webkit-touch-callout: none;
		user-select: none;
		/* safe: when the grid is wider than a small screen, scroll instead of clipping its left edge */
		align-items: safe center;
		justify-content: safe center;
		overflow: auto;
	}
	canvas {
		display: block;
		cursor: crosshair;
	}
	.chrome {
		position: fixed;
		top: 6px;
		right: 8px;
		display: flex;
		gap: 4px;
	}
	.gear,
	.quit {
		padding: 2px 8px;
		font-size: 12px;
		opacity: 0.6;
	}
	.gear {
		font-size: 14px;
		line-height: 1.3;
	}
	.gear:hover,
	.quit:hover {
		opacity: 1;
	}
</style>
