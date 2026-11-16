<script lang="ts">
	/** The game screen: draws the engine's character grid on a canvas and forwards keys and clicks. */
	import { onMount, onDestroy } from 'svelte';
	import type { QudApi, QudFrame } from './qud';
	import { saveRun, deleteRun } from './store';
	import { recordDeeds } from './chronicle';
	import { recordHall, takePendingSeed } from './halloffame';
	import { createMusic, type MusicHandle } from './music';
	import { createWeatherSfx, type SfxHandle } from './weathersfx';
	import { loadSettings, saveSettings, PALETTES, type Settings as Prefs, type PaletteName } from './settings';
	import { computeUiMask, drawProceduralCell } from './sprites';
	import { drawActorSprite, type QudActor } from './actorsprites';
	import { createEffects, drawActorShadow, bobOffset, type FxHandle } from './effects';
	import Tooltip, { describe, type Tip } from './Tooltip.svelte';
	import Settings from './Settings.svelte';
	import Saga from './Saga.svelte';
	import { loadSaga, type SagaData } from './saga';
	import Touch from './Touch.svelte';
	import { mapKey, setKeymap } from './keymap';
	import { startGamepad, stopGamepad, setGamepadOptions } from './gamepad';
	import { createGameSfx } from './gamesfx';
	import Replay from './Replay.svelte';
	import { replayTap, replaySaveLive, replayFinish, setReplayRecording } from './replay';
	import ContextMenu from './ContextMenu.svelte';
	import { ctxItems, ctxRun, ctxCursor, isMenuKey, type CtxItem } from './ctxmenu';
	import { drawTileVariant, neighbourMask, setTileContext, isAnimatedTile, TILE_REPLACED } from './tilevariety';
	import { setSpriteAnimPrefs, tickTileAnim, trackActors, actorOffset, applyActorMotion, resetSpriteAnim } from './spriteanim';

	let { api, first, world, onquit }: { api: QudApi; first: QudFrame; world: string | null; onquit: () => void } = $props();

	let canvas = $state<HTMLCanvasElement | null>(null);
	let fxCanvas = $state<HTMLCanvasElement | null>(null);
	let fxh: FxHandle | null = null;
	let actorCells: { g: string; fgIdx: number; bgIdx: number; fg: string; gx: number; gy: number; a: QudActor }[] = [];
	let uiMaskNow: Uint8Array | null = null;
	/** sprite-mode state of the last draw, for the animation tick */
	let glyphsNow: string[] = [];
	let whoNow = new Map<number, QudActor>();
	let animCells: number[] = [];
	const slideDirty = new Map<number, number>();
	let reducedMq: MediaQueryList | null = null;
	let wrap = $state<HTMLDivElement | null>(null);
	let gear = $state<HTMLButtonElement | null>(null);
	const enginePalette: string[] = JSON.parse(api.palette());
	let prefs = $state<Prefs>(loadSettings());
	let settingsOpen = $state(false);
	let saga = $state<SagaData | null>(null);
	let dead = $state(first.dead);
	let rpId = $state('');
	/** hall of fame id of this life once it has ended: the replay is filed under it */
	let hid = $state('');
	let ctxm = $state<{ items: CtxItem[]; x: number; y: number; cx: number; cy: number } | null>(null);
	let palette: string[] = paletteFor(prefs.palette);
	let frame: QudFrame = first;
	let cw = 9;
	let chh = 16;
	let keys = 0;
	let worldPending = world;
	const music: MusicHandle = createMusic();
	const sfx: SfxHandle = createWeatherSfx();
	const gsfx = createGameSfx();
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
		music.setFlavour(prefs.instrumentFlavour);
		sfx.setEnabled(prefs.sfx);
		sfx.setVolume(prefs.sfxVolume);
		// thunder plays on the SFX bus when it is on, so the music leaves it out
		music.setThunder(!(prefs.sfx && prefs.sfxVolume > 0));
		gsfx.setEnabled(prefs.sfx && prefs.sfxEvents);
		gsfx.setVolume(prefs.sfxVolume);
		gsfx.setReduceMotion(prefs.reduceMotion);
	}

	/** Host settings the engine reads (g.Pref). */
	function applyEngine() {
		api.pref('anim', prefs.animations);
		api.pref('hints', prefs.showHints);
		api.pref('autoaim', prefs.autoAim);
		api.pref('sprites', prefs.renderStyle === 'sprites');
		api.pref('contextMenu', prefs.contextMenu);
		api.pref('recordReplay', prefs.recordReplay);
		setReplayRecording(prefs.recordReplay);
		setKeymap(prefs.keymap);
		setGamepadOptions({ enabled: prefs.gamepad, deadzone: prefs.gamepadDeadzone, repeatMs: prefs.gamepadRepeat, rumble: prefs.gamepadRumble, onSettings: () => (settingsOpen = true) });
	}

	function fxOn(): boolean {
		return prefs.renderStyle === 'sprites' && prefs.weatherFx;
	}
	function applyFx() {
		fxh?.setOptions({ enabled: fxOn(), reduced: prefs.reduceMotion || !!reducedMq?.matches });
		setSpriteAnimPrefs({ reduceMotion: prefs.reduceMotion || !!reducedMq?.matches, tileVariety: prefs.tileVariety, tileAnim: prefs.tileAnim });
		fxh?.setFrame(frame);
	}
	/** Paint one map cell as a sprite (tile variant over the base sprite); false when the glyph has no sprite. */
	function paintSprite(ctx: CanvasRenderingContext2D, f: QudFrame, i: number, a?: QudActor): boolean {
		const g = glyphsNow[i];
		const gx = i % f.w, gy = Math.floor(i / f.w);
		const x = gx * cw, y = gy * chh;
		const fgIdx = f.f.charCodeAt(i) - 65, bgIdx = f.b.charCodeAt(i) - 65;
		const fg = palette[fgIdx];
		const base = drawProceduralCell(ctx, g, fgIdx, bgIdx, fg, palette, gx, gy, x, y, cw, chh, a);
		if (a || !prefs.tileVariety) return base;
		const tv = drawTileVariant(ctx, g, fg, palette[bgIdx], x, y, cw, gx, gy, neighbourMask(glyphsNow, f.w, f.h, gx, gy, g), { chh, fgIdx, bgIdx, palette });
		return base || tv === TILE_REPLACED;
	}
	/** Fill a cell's background and repaint what stands on it. */
	function repaintCell(ctx: CanvasRenderingContext2D, i: number) {
		const f = frame;
		if (i < 0 || i >= f.w * f.h || (uiMaskNow && uiMaskNow[i])) return;
		const x = (i % f.w) * cw, y = Math.floor(i / f.w) * chh;
		ctx.fillStyle = palette[f.b.charCodeAt(i) - 65];
		ctx.fillRect(x, y, cw + 0.5, chh + 0.5);
		if (glyphsNow[i] !== ' ') paintSprite(ctx, f, i, whoNow.get(i));
	}
	/** Animation tick: ambient tiles (water, fire, grass), then the actors: idle bob, walk step, slide, and a shake for the player when hit. */
	function bobTick(t: number) {
		const ctx = canvas?.getContext('2d');
		if (!ctx || !fxOn() || frame.mode !== 'main' || prefs.reduceMotion || reducedMq?.matches) return;
		const mainFont = `${Math.round(chh * 0.86)}px "Share Tech Mono", "DejaVu Sans Mono", Menlo, Consolas, monospace`;
		if (prefs.tileAnim && prefs.tileVariety && animCells.length && tickTileAnim(t)) {
			for (const i of animCells) repaintCell(ctx, i);
		}
		if (!actorCells.length) {
			ctx.font = mainFont;
			return;
		}
		const sh = fxh?.shake() ?? { x: 0, y: 0 };
		for (const c of actorCells) {
			const x = c.gx * cw, y = c.gy * chh;
			const m = actorOffset(c.a.i, c.gx, c.gy, t);
			const fromI = m.fromY * frame.w + m.fromX;
			const slid = prefs.tileAnim && m.moving && (m.fromX !== c.gx || m.fromY !== c.gy) && Math.abs(m.fromX - c.gx) <= 1 && Math.abs(m.fromY - c.gy) <= 1 && m.fromX < 80 && m.fromY < 25;
			if (slid) {
				slideDirty.set(c.a.i, fromI);
				repaintCell(ctx, fromI);
			} else if (slideDirty.has(c.a.i)) {
				repaintCell(ctx, slideDirty.get(c.a.i)!);
				slideDirty.delete(c.a.i);
			}
			ctx.save();
			ctx.beginPath();
			if (slid) {
				const x0 = Math.min(c.gx, m.fromX) * cw, y0 = Math.min(c.gy, m.fromY) * chh;
				ctx.rect(x0, y0, (Math.abs(c.gx - m.fromX) + 1) * cw, (Math.abs(c.gy - m.fromY) + 1) * chh);
			} else ctx.rect(x, y, cw, chh);
			ctx.clip();
			ctx.fillStyle = palette[c.bgIdx];
			ctx.fillRect(x, y, cw + 0.5, chh + 0.5);
			if (frame.fx) drawActorShadow(ctx, x, y, cw, chh, frame.fx);
			if (c.a.p) ctx.translate(sh.x, 0);
			if (prefs.tileAnim) applyActorMotion(ctx, m, x, y, cw, chh);
			else ctx.translate(0, bobOffset(c.a.i, t) * Math.max(0.6, chh / 18));
			drawProceduralCell(ctx, c.g, c.fgIdx, c.bgIdx, c.fg, palette, c.gx, c.gy, x, y, cw, chh, c.a);
			ctx.restore();
		}
		ctx.font = mainFont;
	}
	function lookup(gx: number, gy: number): Tip | null {
		if (prefs.renderStyle !== 'sprites' || frame.mode !== 'main' || gx >= 80 || gy >= 25) return null;
		if (uiMaskNow && uiMaskNow[gy * frame.w + gx]) return null;
		return describe(Array.from(frame.g), frame.w, frame.actors, gx, gy);
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

	let hallDone = false;
	let retired = false;
	const seedText = takePendingSeed();
	async function persist() {
		if (retired) return;
		const meta = JSON.parse(api.meta());
		if (frame.dead) {
			await recordDeeds(api, meta.seed);
			if (!hallDone) {
				hallDone = true;
				const id = `${meta.seed}|${Date.now()}`;
				await recordHall(api, id, false, seedText);
				await replayFinish(api, meta.seed, id);
				hid = id;
			}
			await deleteRun(meta.seed);
			return;
		}
		await saveRun({ ...meta, saved: Date.now() }, api.save(), worldPending ?? undefined);
		worldPending = null;
		void replaySaveLive(api, meta.seed);
		// deeds go to the chronicle store and, when opened from Realmforge, back to it live
		void recordDeeds(api, meta.seed);
	}

	/** End the run honourably: it enters the hall as retired and the save is removed. */
	async function retire() {
		if (!confirm('Retire this character? The run ends and enters the Hall of Fame.')) return;
		const meta = JSON.parse(api.meta());
		await recordDeeds(api, meta.seed);
		hallDone = true;
		retired = true;
		const id = `${meta.seed}|${Date.now()}`;
		await recordHall(api, id, !frame.dead, seedText);
		await replayFinish(api, meta.seed, id);
		await deleteRun(meta.seed);
		onquit();
	}

	async function quit() {
		await persist();
		onquit();
	}

	function onkey(e: KeyboardEvent) {
		// while the settings pane is open it owns the keyboard
		if (settingsOpen || saga || rpId) return;
		if (!ctxm && prefs.contextMenu && isMenuKey(e, frame.mode)) {
			e.preventDefault();
			const c = ctxCursor();
			if (c && canvas) {
				const r = canvas.getBoundingClientRect();
				openCtx(r.left + (c.x + 0.5) * cw, r.top + (c.y + 1) * chh, c.x, c.y);
			}
			return;
		}
		if (frame.dead && (e.key === 'S' || e.key === 's')) {
			e.preventDefault();
			readSaga();
			return;
		}
		if (e.metaKey || e.ctrlKey || e.altKey) return;
		if (e.code === 'Backquote' || e.key === '`') {
			e.preventDefault();
			settingsOpen = true;
			return;
		}
		const k = mapKey(e, frame.mode);
		if (k === null) {
			e.preventDefault();
			return;
		}
		if (k === 'Shift' || k === 'Control' || k === 'Alt' || k === 'Meta' || k === 'CapsLock') return;
		e.preventDefault();
		send(k);
	}
	function readSaga() {
		saga = loadSaga(api);
	}
	function send(k: string) {
		const f = api.key(k);
		if (!f || 'error' in f) return;
		applyFrame(f, k);
	}
	function applyFrame(f: QudFrame, k?: string) {
		frame = f;
		replayTap(api, f);
		dead = f.dead;
		music.setState(f.music);
		sfx.setState(f.music);
		gsfx.handleFrame(f, k);
		draw();
		fxh?.setFrame(f);
		keys++;
		if (f.save || keys % prefs.autosaveEvery === 0) void persist();
		if (f.exit) void quit();
	}
	function cellAt(e: { clientX: number; clientY: number }): string {
		const r = canvas!.getBoundingClientRect();
		return `click:${Math.floor((e.clientX - r.left) / cw)},${Math.floor((e.clientY - r.top) / chh)}`;
	}
	function cellXY(e: { clientX: number; clientY: number }) {
		const r = canvas!.getBoundingClientRect();
		return { cx: Math.floor((e.clientX - r.left) / cw), cy: Math.floor((e.clientY - r.top) / chh) };
	}
	function openCtx(px: number, py: number, cx: number, cy: number): boolean {
		if (!prefs.contextMenu || frame.dead) return false;
		const items = ctxItems(cx, cy);
		if (!items.length) return false;
		ctxm = { items, x: px, y: py, cx, cy };
		return true;
	}
	function ctxPick(id: string) {
		const c = ctxm;
		ctxm = null;
		if (!c) return;
		const f = ctxRun(id, c.cx, c.cy);
		if (f) applyFrame(f);
	}
	function onctx(e: MouseEvent) {
		if (!prefs.contextMenu) {
			if (touch) e.preventDefault();
			return;
		}
		e.preventDefault();
		if (!canvas) return;
		const c = cellXY(e);
		openCtx(e.clientX, e.clientY, c.cx, c.cy);
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
			const c = cellXY(pt);
			if (prefs.contextMenu && openCtx(pt.clientX, pt.clientY, c.cx, c.cy)) {
				navigator.vibrate?.(15);
				return;
			}
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
		if (ctxm) {
			ctxm = null;
			return;
		}
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
		fxh?.layout(cw, chh, dpr);
		applyFx();
		draw();
	}

	function draw() {
		const f = frame;
		const ctx = canvas?.getContext('2d');
		if (!ctx) return;
		const glyphs = Array.from(f.g);
		const spritesOn = prefs.renderStyle === 'sprites';
		// menus, dialogue and overlays are text: never paint creature sprites over their letters
		const useSprites = spritesOn && ['main', 'target', 'look', 'dir', 'world'].includes(f.mode);
		const uiMask = useSprites ? computeUiMask(glyphs, f.b, f.w, f.h) : null;
		uiMaskNow = uiMask;
		actorCells = [];
		animCells = [];
		glyphsNow = glyphs;
		if (useSprites) {
			const mu = f.music as unknown as Record<string, any> | undefined;
			const fxi = f.fx as unknown as Record<string, any> | undefined;
			setTileContext({ salt: (mu?.seed ?? 0) ^ ((mu?.depth ?? 0) * 7919), genre: mu?.genre, season: mu?.season, wind: fxi?.wind, indoors: fxi?.ind });
			trackActors(f.actors, performance.now());
		}
		const shadows = fxOn() && f.mode === 'main';
		const who = new Map<number, QudActor>();
		if (useSprites) for (const a of f.actors ?? []) who.set(a.y * f.w + a.x, a);
		whoNow = who;
		const mainFont = `${Math.round(chh * 0.86)}px "Share Tech Mono", "DejaVu Sans Mono", Menlo, Consolas, monospace`;
		ctx.font = mainFont;
		ctx.textAlign = 'center';
		ctx.textBaseline = 'middle';
		for (let i = 0; i < f.w * f.h; i++) {
			ctx.fillStyle = palette[f.b.charCodeAt(i) - 65];
			ctx.fillRect((i % f.w) * cw, Math.floor(i / f.w) * chh, cw + 0.5, chh + 0.5);
		}
		for (let i = 0; i < f.w * f.h; i++) {
			const g = glyphs[i];
			if (g === ' ') continue;
			const gx = i % f.w;
			const gy = Math.floor(i / f.w);
			const x = gx * cw;
			const y = gy * chh;
			const fgIdx = f.f.charCodeAt(i) - 65;
			const bgIdx = f.b.charCodeAt(i) - 65;
			const fg = palette[fgIdx];
			if (useSprites && uiMask && !uiMask[i] && gx < 80 && gy < 25) {
				const who1 = who.get(i);
				if (who1 && shadows && f.fx) drawActorShadow(ctx, x, y, cw, chh, f.fx);
				if (paintSprite(ctx, f, i, who1)) {
					if (who1 && shadows) actorCells.push({ g, fgIdx, bgIdx, fg, gx, gy, a: who1 });
					else if (!who1 && shadows && isAnimatedTile(g, fgIdx, bgIdx)) animCells.push(i);
					ctx.font = mainFont;
					ctx.textAlign = 'center';
					ctx.textBaseline = 'middle';
					continue;
				}
			}
			ctx.fillStyle = fg;
			if (g === '█') ctx.fillRect(x, y, cw + 0.5, chh + 0.5);
			else if (g === '─') ctx.fillRect(x, y + chh / 2, cw + 0.5, 1);
			else if (g === '│') ctx.fillRect(x + cw / 2, y, 1, chh + 0.5);
			else ctx.fillText(g, x + cw / 2, y + chh / 2 + 1);
		}
		if (spritesOn && f.portrait) drawPortrait(ctx, f);
	}

	/** The engine reserves a blank rectangle on talk, trade and the character sheet: paint the speaker or the player there, large and framed. */
	function drawPortrait(ctx: CanvasRenderingContext2D, f: QudFrame) {
		const p = f.portrait;
		if (!p) return;
		const px = p.x * cw, py = p.y * chh, pw = p.w * cw, ph = p.h * chh;
		const bg = palette[f.b.charCodeAt(p.y * f.w + p.x) - 65] ?? '#101418';
		ctx.save();
		ctx.beginPath();
		ctx.rect(px, py, pw, ph);
		ctx.clip();
		ctx.fillStyle = bg;
		ctx.fillRect(px, py, pw, ph);
		const gr = ctx.createRadialGradient(px + pw / 2, py + ph * 0.55, 2, px + pw / 2, py + ph * 0.55, Math.max(pw, ph) * 0.75);
		gr.addColorStop(0, 'rgba(255,255,255,0.18)');
		gr.addColorStop(1, 'rgba(255,255,255,0)');
		ctx.fillStyle = gr;
		ctx.fillRect(px, py, pw, ph);
		ctx.fillStyle = 'rgba(0,0,0,0.3)';
		ctx.fillRect(px, py + ph * 0.9, pw, ph * 0.1);
		// a sprite is built for a cell about 0.6 as wide as it is tall: keep that shape, as large as fits
		const sh = ph - 4;
		const sw = Math.min(pw - 4, sh * 0.62);
		drawActorSprite(ctx, p.actor, ' ', '#ffffff', palette, 0, px + (pw - sw) / 2, py + 2, sw, sh);
		ctx.restore();
		ctx.strokeStyle = palette[11] ?? '#3a8a9a';
		ctx.lineWidth = 1;
		ctx.strokeRect(px + 0.5, py + 0.5, pw - 1, ph - 1);
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
		reducedMq = window.matchMedia('(prefers-reduced-motion: reduce)');
		const onred = () => applyFx();
		reducedMq.addEventListener('change', onred);
		if (fxCanvas) {
			fxh = createEffects(fxCanvas);
			fxh.onTick(bobTick);
		}
		mq.addEventListener('change', onmq);
		applyAudio();
		applyEngine();
		music.setState(frame.music);
		sfx.setState(frame.music);
		resize();
		// the play area shrinks when touch controls appear: refit the grid whenever it changes size
		const ro = new ResizeObserver(() => resize());
		if (wrap) ro.observe(wrap);
		void persist();
		resetSpriteAnim();
		startGamepad((k) => { if (!settingsOpen && !saga) send(k); }, () => frame, { enabled: prefs.gamepad, onSettings: () => (settingsOpen = true) });
		// fonts load after first paint: redraw once they are in
		void document.fonts?.ready.then(draw);
		return () => {
			reducedMq?.removeEventListener('change', onred);
			fxh?.dispose();
			fxh = null;
			mq.removeEventListener('change', onmq);
			ro.disconnect();
			stopGamepad();
		};
	});
	onDestroy(() => {
		music.dispose();
		sfx.dispose();
		gsfx.dispose();
		void persist();
	});
</script>

<svelte:window onkeydown={onkey} onresize={resize} />

<!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_static_element_interactions -->
<div class="screen" bind:this={wrap} onclick={onclick} {onpointerdown} {onpointermove} onpointerup={clearPress} onpointercancel={clearPress} oncontextmenu={onctx} style:bottom={touch ? `${ctlH}px` : '0'}>
	<div class="stack">
		<canvas bind:this={canvas} aria-label="Game screen"></canvas>
		<canvas class="fx" bind:this={fxCanvas} aria-hidden="true"></canvas>
	</div>
</div>
{#if !touch}
	<Tooltip {canvas} cell={() => ({ cw, chh })} {lookup} />
{/if}
{#if ctxm}
	<ContextMenu items={ctxm.items} x={ctxm.x} y={ctxm.y} onpick={ctxPick} onclose={() => (ctxm = null)} />
{/if}
{#if touch}
	<Touch {send} bind:height={ctlH} />
{/if}
<div class="chrome">
	{#if dead}<button class="quit" style:opacity="1" onclick={readSaga} title="Read the saga of this life (S)">Read your saga</button>{/if}
	{#if dead && hid}<button class="quit" style:opacity="1" onclick={() => (rpId = hid)} title="Watch this life again">Watch replay</button>{/if}
	<button class="gear" bind:this={gear} onclick={() => (settingsOpen = true)} title="Settings (`)" aria-label="Settings">⚙</button>
	<button class="quit" onclick={retire} title="End this life honourably and enter the Hall of Fame">Retire</button>
	<button class="quit" onclick={quit} title="Save and return to the title screen">Save &amp; quit</button>
</div>
{#if saga}
	<Saga {saga} onclose={() => (saga = null)} />
{/if}
{#if rpId}
	<Replay {api} id={rpId} onclose={() => (rpId = '')} />
{/if}
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
	.stack {
		position: relative;
		flex: none;
	}
	.fx {
		position: absolute;
		left: 0;
		top: 0;
		pointer-events: none;
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
