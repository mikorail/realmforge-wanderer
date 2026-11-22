<script lang="ts">
	/** The game screen: draws the engine's character grid on a canvas and forwards keys and clicks. */
	import { pixelCell, pixelBlit } from './pixelpass';
	import { drawPixelTile, drawPixelActor, drawPixelPortrait, setPixelArt } from './pixelart/index';
	import { tileAnimFrame as pxFrame } from './spriteanim';
	import { poseOf as pxPoseOf, type Pose as PxPose } from './poses';
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
	import { createAudioHub, type HubPrefs } from './audiohub';
	import Replay from './Replay.svelte';
	import { replayTap, replaySaveLive, replayFinish, setReplayRecording } from './replay';
	import ContextMenu from './ContextMenu.svelte';
	import { ctxItems, ctxRun, ctxCursor, isMenuKey, type CtxItem } from './ctxmenu';
	import { drawTileVariant, neighbourMask, setTileContext, isAnimatedTile, TILE_REPLACED } from './tilevariety';
	import { setSpriteAnimPrefs, tickTileAnim, trackActors, actorOffset, applyActorMotion, resetSpriteAnim, noteAttack, flashActor, markDeath } from './spriteanim';
	import { createHitFx, type HitFxHandle } from './hitfx';
	import { setGearFrame, setGearPrefs } from './gearsprites';
	import { setPosePrefs, trackPoses, notePoseHits, setSleepers, resetPoses } from './poses';
	import { applyCultureTile, setCultureTileOptions } from './culturetiles';
	import { createSpellFx, type SpellFxHandle } from './spellfx';
	import { drawFace, portraitBreath } from './portraitfx';
	import IconCard from './IconCard.svelte';
	import StatusStrip from './StatusStrip.svelte';
	import type { IconRowInfo, IconStatusInfo } from './icons';

	let { api, first, world, onquit }: { api: QudApi; first: QudFrame; world: string | null; onquit: () => void } = $props();

	let canvas = $state<HTMLCanvasElement | null>(null);
	let fxCanvas = $state<HTMLCanvasElement | null>(null);
	let fxh: FxHandle | null = null;
	let hitCanvas = $state<HTMLCanvasElement | null>(null);
	let hitfx: HitFxHandle | null = null;
	let spellCanvas = $state<HTMLCanvasElement | null>(null);
	let spellfx: SpellFxHandle | null = null;
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
	let iconRows = $state<IconRowInfo[] | undefined>(undefined);
	let iconStatus = $state<IconStatusInfo | undefined>(undefined);
	let iconsRaw = '';
	let statusRaw = '';
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
	// shared context for soundscapes, voices, music layers, foley and culture music; ducks the base music while stingers and performers play
	const hub = createAudioHub({ duckBase: (factor) => music.setVolume(prefs.musicVolume * factor) });
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
		hub.setPrefs(prefs as HubPrefs);
	}

	/** Host settings the engine reads (g.Pref). */
	function applyEngine() {
		api.pref('anim', prefs.animations);
		api.pref('hints', prefs.showHints);
		api.pref('autoaim', prefs.autoAim);
		api.pref('sprites', prefs.renderStyle === 'sprites');
		api.pref('contextMenu', prefs.contextMenu);
		api.pref('routines', prefs.routines);
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
		hitfx?.setOptions({ numbers: prefs.hitNumbers, shake: prefs.hitShake, pause: prefs.hitPause, reduced: prefs.reduceMotion || !!reducedMq?.matches });
		spellfx?.setOptions({ enabled: prefs.spellFx, reduced: prefs.reduceMotion || !!reducedMq?.matches });
		setCultureTileOptions({ enabled: prefs.cultureTiles });
		setPixelArt(prefs.pixelArt);
		setGearPrefs({ gearSprites: prefs.gearSprites });
		setPosePrefs({ actorPoses: prefs.actorPoses });
		setSpriteAnimPrefs({ reduceMotion: prefs.reduceMotion || !!reducedMq?.matches, tileVariety: prefs.tileVariety, tileAnim: prefs.tileAnim });
		fxh?.setFrame(frame);
	}
	/** Paint one map cell as a sprite (tile variant over the base sprite); false when the glyph has no sprite. */
	const pxScratch = {} as PxPose;
	// box-drawing glyphs as pixel lines: [up, down, left, right] arms, 1 single, 2 double
	const BOX: Record<string, [number, number, number, number]> = {
		'─': [0, 0, 1, 1], '│': [1, 1, 0, 0], '┌': [0, 1, 0, 1], '┐': [0, 1, 1, 0], '└': [1, 0, 0, 1], '┘': [1, 0, 1, 0],
		'├': [1, 1, 0, 1], '┤': [1, 1, 1, 0], '┬': [0, 1, 1, 1], '┴': [1, 0, 1, 1], '┼': [1, 1, 1, 1],
		'═': [0, 0, 2, 2], '║': [2, 2, 0, 0], '╔': [0, 2, 0, 2], '╗': [0, 2, 2, 0], '╚': [2, 0, 0, 2], '╝': [2, 0, 2, 0],
		'╠': [2, 2, 0, 2], '╣': [2, 2, 2, 0], '╦': [0, 2, 2, 2], '╩': [2, 0, 2, 2], '╬': [2, 2, 2, 2]
	};
	/** a frame piece drawn from whole art pixels (the current fillStyle), centred in the cell; false = not a box glyph */
	function drawBoxGlyph(ctx: CanvasRenderingContext2D, g: string, x: number, y: number, cw: number, chh: number): boolean {
		const b = BOX[g];
		if (!b) return false;
		const dpr = window.devicePixelRatio || 1;
		const px = Math.max(1, Math.round((chh / 20) * dpr)) / dpr;
		const cx = Math.round((x + cw / 2) / px) * px, cy = Math.round((y + chh / 2) / px) * px;
		const arm = (n: number, horiz: boolean, neg: boolean) => {
			if (!n) return;
			const t = n === 2 ? px * 3 : px * 2; // double lines are a heavier pair
			const len = (horiz ? cw : chh) / 2 + 0.5;
			const off = -t / 2;
			if (horiz) ctx.fillRect(neg ? cx - len : cx - t / 2, cy + off, len + t / 2, t);
			else ctx.fillRect(cx + off, neg ? cy - len : cy - t / 2, t, len + t / 2);
		};
		arm(b[0], false, true);
		arm(b[1], false, false);
		arm(b[2], true, true);
		arm(b[3], true, false);
		return true;
	}
	/** canvas text font: VT323 (a pixel font) sized to the cell, or the smooth mono face */
	function fontStr(): string {
		if (prefs.pixelFont) return `${Math.max(8, Math.round(Math.min(chh * 1.04, cw * 2.4)))}px "VT323", "Share Tech Mono", Menlo, monospace`;
		return `${Math.round(chh * 0.86)}px "Share Tech Mono", "DejaVu Sans Mono", Menlo, Consolas, monospace`;
	}
	function paintSprite(ctx: CanvasRenderingContext2D, f: QudFrame, i: number, a?: QudActor): boolean {
		if (prefs.pixelArt) {
			const gx0 = i % f.w, gy0 = Math.floor(i / f.w);
			const fi = f.f.charCodeAt(i) - 65, bi = f.b.charCodeAt(i) - 65;
			const x1 = gx0 * cw, y1 = gy0 * chh;
			if (a) {
				if (drawPixelActor(ctx, a, glyphsNow[i], palette[fi], palette[bi], x1, y1, cw, chh, pxFrame(), pxPoseOf(a.i, performance.now(), pxScratch))) return true;
			} else if (drawPixelTile(ctx, glyphsNow[i], fi, bi, palette[fi], palette[bi], gx0, gy0, x1, y1, cw, chh, pxFrame(), neighbourMask(glyphsNow, f.w, f.h, gx0, gy0, glyphsNow[i]))) return true;
		}
		if (!prefs.pixelArt) return paintSpriteRaw(ctx, f, i, a, (i % f.w) * cw, Math.floor(i / f.w) * chh, cw, chh);
		const x0 = (i % f.w) * cw, y0 = Math.floor(i / f.w) * chh;
		return pixelCell(ctx, x0, y0, cw, chh, window.devicePixelRatio || 1, !!a, (c, w, h) => paintSpriteRaw(c, f, i, a, 0, 0, w, h));
	}
	function paintSpriteRaw(ctx: CanvasRenderingContext2D, f: QudFrame, i: number, a: QudActor | undefined, x: number, y: number, cw: number, chh: number): boolean {
		const g = glyphsNow[i];
		const gx = i % f.w, gy = Math.floor(i / f.w);
		const fgIdx = f.f.charCodeAt(i) - 65, bgIdx = f.b.charCodeAt(i) - 65;
		const fg = palette[fgIdx];
		const base = drawProceduralCell(ctx, g, fgIdx, bgIdx, fg, palette, gx, gy, x, y, cw, chh, a);
		if (a) return base;
		const nm = neighbourMask(glyphsNow, f.w, f.h, gx, gy, g);
		let replaced = false;
		if (prefs.tileVariety) replaced = drawTileVariant(ctx, g, fg, palette[bgIdx], x, y, cw, gx, gy, nm, { chh, fgIdx, bgIdx, palette }) === TILE_REPLACED;
		if (prefs.cultureTiles && !replaced) applyCultureTile(ctx, g, fgIdx, bgIdx, x, y, cw, chh, gx, gy, f.style, nm);
		return base || replaced;
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
		const mainFont = fontStr();
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
			if (prefs.pixelArt && drawPixelActor(ctx, c.a, c.g, c.fg, palette[c.bgIdx], x, y, cw, chh, pxFrame(), pxPoseOf(c.a.i, t, pxScratch))) { /* pixel art */ }
			else if (prefs.pixelArt) pixelCell(ctx, x, y, cw, chh, window.devicePixelRatio || 1, true, (pc, w, h) => { drawProceduralCell(pc, c.g, c.fgIdx, c.bgIdx, c.fg, palette, c.gx, c.gy, 0, 0, w, h, c.a); return true; });
			else drawProceduralCell(ctx, c.g, c.fgIdx, c.bgIdx, c.fg, palette, c.gx, c.gy, x, y, cw, chh, c.a);
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
	/** keys wait their turn so a short hit pause never drops one */
	const keyQ: string[] = [];
	let pumping = false;
	async function send(k: string) {
		keyQ.push(k);
		if (pumping) return;
		pumping = true;
		try {
			while (keyQ.length) {
				const key = keyQ.shift()!;
				const f = api.key(key);
				if (!f || 'error' in f) continue;
				applyFrame(f, key);
				const p = hitfx?.pause();
				if (p) await p;
			}
		} finally {
			pumping = false;
		}
	}
	function applyFrame(f: QudFrame, k?: string) {
		frame = f;
		replayTap(api, f);
		dead = f.dead;
		music.setState(f.music);
		sfx.setState(f.music);
		gsfx.handleFrame(f, k);
		hub.frame(f, k);
		draw();
		fxh?.setFrame(f);
		hitfx?.setTelegraphs(typeof f.tele === 'string' ? (JSON.parse(f.tele) as { x: number; y: number }[]) : undefined);
		setSleepers(f.sleepers);
		if ((f.icons ?? '') !== iconsRaw) {
			iconsRaw = f.icons ?? '';
			iconRows = iconsRaw ? (JSON.parse(iconsRaw) as IconRowInfo[]) : undefined;
		}
		if ((f.status ?? '') !== statusRaw) {
			statusRaw = f.status ?? '';
			iconStatus = statusRaw ? (JSON.parse(statusRaw) as IconStatusInfo) : undefined;
		}
		spellfx?.setStatuses(f.statuses);
		if (f.hits?.length) {
			hitfx?.push(f.hits);
			spellfx?.push(f.hits, f.origins, f.actors?.map((a) => ({ id: a.i, x: a.x, y: a.y })));
			const t = performance.now();
			notePoseHits(f.hits, t);
			for (const e of f.hits) {
				if (e.attackerId && e.kind !== 'heal' && e.kind !== 'death') noteAttack(e.attackerId, e.x, e.y, t);
				if (e.targetId && (e.kind === 'hit' || e.kind === 'crit' || e.kind === 'burn' || e.kind === 'poison' || e.kind === 'bleed')) flashActor(e.targetId, t);
			}
		}
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
		if (prefs.pixelArt && prefs.pixelSnap && scale === 'auto') {
			// one art pixel = u whole device pixels (sprites are 12x20): crisp, even pixels
			const u = Math.floor(Math.min((W * dpr) / (frame.w * 12), (H * dpr) / (frame.h * 20)));
			if (u >= 1 && (20 * u) / dpr >= chh * 0.8) {
				cw = (12 * u) / dpr;
				chh = (20 * u) / dpr;
			}
		}
		canvas.width = frame.w * cw * dpr;
		canvas.height = frame.h * chh * dpr;
		canvas.style.width = `${frame.w * cw}px`;
		canvas.style.height = `${frame.h * chh}px`;
		canvas.getContext('2d')?.setTransform(dpr, 0, 0, dpr, 0, 0);
		fxh?.layout(cw, chh, dpr);
		hitfx?.layout();
		spellfx?.layout();
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
			trackPoses(f.actors, performance.now());
			setGearFrame(f.gear);
		}
		const shadows = fxOn() && f.mode === 'main';
		const who = new Map<number, QudActor>();
		if (useSprites) for (const a of f.actors ?? []) who.set(a.y * f.w + a.x, a);
		whoNow = who;
		const mainFont = fontStr();
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
			if (prefs.pixelArt && drawBoxGlyph(ctx, g, x, y, cw, chh)) continue;
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
		ctx.fillStyle = 'rgba(255,255,255,0.07)';
		ctx.fillRect(px + pw * 0.12, py + ph * 0.1, pw * 0.76, ph * 0.8);
		ctx.fillStyle = 'rgba(255,255,255,0.06)';
		ctx.fillRect(px + pw * 0.24, py + ph * 0.2, pw * 0.52, ph * 0.6);
		ctx.fillStyle = 'rgba(0,0,0,0.3)';
		ctx.fillRect(px, py + ph * 0.9, pw, ph * 0.1);
		// a sprite is built for a cell about 0.6 as wide as it is tall: keep that shape, as large as fits
		const sh = ph - 4;
		const sw = Math.min(pw - 4, sh * 0.62);
		const reduce = prefs.reduceMotion || !!reducedMq?.matches;
		const sx = px + (pw - sw) / 2;
		const sy = py + 2 + portraitBreath(performance.now(), sh, reduce);
		if (prefs.pixelArt && drawPixelPortrait(ctx, p.actor, '#c8b27a', bg, sx, sy, sw, sh, pxFrame(), pxPoseOf(p.actor.i, performance.now(), pxScratch))) {
			/* hand-drawn pixel sprite */
		} else if (prefs.pixelArt) {
			// vector drawers through the pixel pass: a coarse grid, hard edges, outline
			pixelBlit(ctx, sx, sy, sw, sh, 26, 42, (c, w, h) => {
				drawActorSprite(c, p.actor, ' ', '#ffffff', palette, 0, 0, 0, w, h);
				if (prefs.portraitFaces && p.mood && p.actor.s) drawFace(c, p.mood, p.actor.s, 0, 0, w, h, {}, { id: p.actor.i, reduceMotion: true, key: p.kind + ':' + p.actor.i });
				return true;
			}, { outline: true, levels: 7 });
		} else {
		drawActorSprite(ctx, p.actor, ' ', '#ffffff', palette, 0, sx, sy, sw, sh);
		// a face with the speaker's mood, for people only (creatures have a template and no species)
		if (prefs.portraitFaces && p.mood && p.actor.s) {
			if (drawFace(ctx, p.mood, p.actor.s, sx, sy, sw, sh, {}, { id: p.actor.i, reduceMotion: reduce, key: p.kind + ':' + p.actor.i })) faceRedraw();
		}
		}
		ctx.restore();
		ctx.strokeStyle = palette[11] ?? '#3a8a9a';
		ctx.lineWidth = 1;
		ctx.strokeRect(px + 0.5, py + 0.5, pw - 1, ph - 1);
	}

	/** a blink or an expression change is under way: repaint the portrait soon */
	let faceTimer: ReturnType<typeof setTimeout> | null = null;
	function faceRedraw() {
		if (faceTimer) return;
		faceTimer = setTimeout(() => {
			faceTimer = null;
			if (frame.portrait) draw();
		}, 80);
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
		if (spellCanvas) spellfx = createSpellFx(spellCanvas, () => ({ cw, chh, dpr: window.devicePixelRatio || 1, cols: 80, rows: 25 }));
		if (hitCanvas) hitfx = createHitFx(hitCanvas, () => ({ cw, chh, dpr: window.devicePixelRatio || 1, cols: 80, rows: 25 }));
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
		resetPoses();
		startGamepad((k) => { if (!settingsOpen && !saga) send(k); }, () => frame, { enabled: prefs.gamepad, onSettings: () => (settingsOpen = true) });
		// fonts load after first paint: redraw once they are in
		void Promise.resolve(document.fonts?.load('16px "VT323"')).finally(() => void document.fonts?.ready.then(draw));
		return () => {
			reducedMq?.removeEventListener('change', onred);
			fxh?.dispose();
			fxh = null;
			hitfx?.dispose();
			hitfx = null;
			spellfx?.dispose();
			spellfx = null;
			mq.removeEventListener('change', onmq);
			ro.disconnect();
			stopGamepad();
		};
	});
	onDestroy(() => {
		music.dispose();
		sfx.dispose();
		gsfx.dispose();
		hub.dispose();
		void persist();
	});
</script>

<svelte:window onkeydown={onkey} onresize={resize} />

<!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_static_element_interactions -->
<div class="screen" bind:this={wrap} onclick={onclick} {onpointerdown} {onpointermove} onpointerup={clearPress} onpointercancel={clearPress} oncontextmenu={onctx} style:bottom={touch ? `${ctlH}px` : '0'}>
	<div class="stack">
		<canvas bind:this={canvas} aria-label="Game screen"></canvas>
		<canvas class="fx" bind:this={fxCanvas} aria-hidden="true"></canvas>
		<canvas class="fx" bind:this={spellCanvas} aria-hidden="true"></canvas>
		<canvas class="fx" bind:this={hitCanvas} aria-hidden="true"></canvas>
	</div>
</div>
{#if !touch}
	<Tooltip {canvas} cell={() => ({ cw, chh })} {lookup} engineTip={frame.tip ?? null} deep={(x, y) => api.tipAt(x, y)} here={frame.here ?? null} keymap={prefs.keymap} />
	<IconCard {canvas} cell={() => ({ cw, chh })} rows={iconRows} enabled={prefs.itemIcons} />
{/if}
<StatusStrip {canvas} cell={() => ({ cw, chh })} status={iconStatus} cols={frame.w} enabled={prefs.itemIcons} />
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
