<script lang="ts">
	/**
	 * Toggleable parchment-styled Canvas2D minimap showing the current 80x25 zone.
	 * Renders discovered roads, walls/buildings, water, shrines/altars, market stalls,
	 * stairs/gates, hostile/neutral/friendly actors, and a pulsing player marker.
	 */
	import { onDestroy } from 'svelte';
	import type { QudFrame } from './qud';
	import type { QudActor } from './actorsprites';
	import { computeUiMask } from './sprites';

	let {
		frame,
		open = false,
		onclose,
		palette = [],
		reduceMotion = false,
	}: {
		frame: QudFrame;
		open?: boolean;
		onclose: () => void;
		palette?: string[];
		reduceMotion?: boolean;
	} = $props();

	const MAP_W = 80;
	const MAP_H = 25;
	const CW = 3;
	const CH = 4;
	const W = MAP_W * CW; // 240
	const H = MAP_H * CH; // 100

	const MAP_MODES = new Set(['main', 'target', 'look', 'dir', 'world']);

	const PLACE_TITLES: Record<string, string> = {
		street: 'Town Streets',
		market: 'Market Quarter',
		tavern: 'Tavern & Hearth',
		temple: 'Temple Precinct',
		forge: 'Smithy Quarter',
		harbor: 'Harborfront',
		farm: 'Farmlands',
		'wilds-forest': 'Woodlands',
		'wilds-grass': 'Grasslands',
		'wilds-desert': 'Wastes',
		'wilds-mountain': 'Highlands',
		'wilds-swamp': 'Mire',
		'wilds-snow': 'Frostlands',
		coast: 'Coastline',
		cave: 'Caverns',
		dungeon: 'Dungeon',
		crypt: 'Crypts',
		mine: 'Deep Mines',
		tower: 'Spire',
		sky: 'Skyward',
		underwater: 'Sunken Depths',
		'ship-deck': 'Ship Deck',
		'rail-station': 'Rail Station',
		camp: 'Encampment',
	};

	interface MapSnapshot {
		w: number;
		h: number;
		glyphs: string[];
		f: string;
		b: string;
		actors: QudActor[];
		mode: string;
		place: string;
		mood: string;
		depth: number;
		event: string;
		ph?: number;
		day: boolean;
		playerX: number;
		playerY: number;
	}

	let canvas = $state<HTMLCanvasElement | null>(null);
	let snap = $state<MapSnapshot | null>(null);

	let parchmentBuf: HTMLCanvasElement | null = null;
	let terrainBuf: HTMLCanvasElement | null = null;
	let rafId = 0;

	function hash2(x: number, y: number): number {
		let h = Math.imul(x + 374761393, 668265263) ^ Math.imul(y + 1442695041, 2246822519);
		h = Math.imul(h ^ (h >>> 13), 1274126177);
		return ((h ^ (h >>> 16)) >>> 0) / 4294967296;
	}

	function ensureParchment(): HTMLCanvasElement {
		if (parchmentBuf) return parchmentBuf;
		const c = document.createElement('canvas');
		c.width = W;
		c.height = H;
		const ctx = c.getContext('2d')!;

		// Warm vellum base gradient
		const grad = ctx.createRadialGradient(W * 0.5, H * 0.48, 18, W * 0.5, H * 0.5, W * 0.65);
		grad.addColorStop(0, '#f0e3c6');
		grad.addColorStop(0.72, '#e2cfa8');
		grad.addColorStop(1, '#cfb585');
		ctx.fillStyle = grad;
		ctx.fillRect(0, 0, W, H);

		// Subtle paper grain speckles
		for (let y = 0; y < H; y += 2) {
			for (let x = 0; x < W; x += 2) {
				const r = hash2(x, y);
				if (r > 0.86) {
					ctx.fillStyle = 'rgba(112, 82, 46, 0.06)';
					ctx.fillRect(x, y, 2, 2);
				} else if (r < 0.08) {
					ctx.fillStyle = 'rgba(255, 250, 236, 0.08)';
					ctx.fillRect(x, y, 2, 2);
				}
			}
		}

		// Faint cartographic grid rules
		ctx.strokeStyle = 'rgba(122, 92, 54, 0.11)';
		ctx.lineWidth = 1;
		for (let x = 30; x < W; x += 30) {
			ctx.beginPath();
			ctx.moveTo(x + 0.5, 0);
			ctx.lineTo(x + 0.5, H);
			ctx.stroke();
		}
		for (let y = 25; y < H; y += 25) {
			ctx.beginPath();
			ctx.moveTo(0, y + 0.5);
			ctx.lineTo(W, y + 0.5);
			ctx.stroke();
		}

		// Inner parchment frame rule
		ctx.strokeStyle = 'rgba(98, 68, 38, 0.38)';
		ctx.strokeRect(0.5, 0.5, W - 1, H - 1);

		parchmentBuf = c;
		return c;
	}

	function isWarmRedHex(hex: string | undefined): boolean {
		if (!hex || !hex.startsWith('#') || hex.length < 7) return false;
		const r = parseInt(hex.slice(1, 3), 16);
		const g = parseInt(hex.slice(3, 5), 16);
		const b = parseInt(hex.slice(5, 7), 16);
		return r > 170 && r > b * 1.6 && g < 160;
	}

	function buildSnapshot(f: QudFrame): MapSnapshot {
		const glyphs = Array.from(f.g);
		const actors = (f.actors ?? []).filter((a) => a.x >= 0 && a.x < MAP_W && a.y >= 0 && a.y < MAP_H);
		let px = -1;
		let py = -1;
		const me = actors.find((a) => a.p);
		if (me) {
			px = me.x;
			py = me.y;
		} else {
			for (let gy = 0; gy < Math.min(MAP_H, f.h); gy++) {
				for (let gx = 0; gx < Math.min(MAP_W, f.w); gx++) {
					if (glyphs[gy * f.w + gx] === '@') {
						px = gx;
						py = gy;
						break;
					}
				}
				if (px >= 0) break;
			}
		}
		return {
			w: f.w,
			h: f.h,
			glyphs,
			f: f.f,
			b: f.b,
			actors,
			mode: f.mode,
			place: f.sound?.place ?? '',
			mood: f.music?.mood ?? '',
			depth: f.music?.depth ?? 0,
			event: f.sound?.event ?? '',
			ph: f.fx?.ph,
			day: f.music?.day ?? true,
			playerX: px,
			playerY: py,
		};
	}

	function renderTerrain(s: MapSnapshot): HTMLCanvasElement {
		if (!terrainBuf) {
			terrainBuf = document.createElement('canvas');
			terrainBuf.width = W;
			terrainBuf.height = H;
		}
		const ctx = terrainBuf.getContext('2d')!;
		ctx.drawImage(ensureParchment(), 0, 0);

		const uiMask = s.mode === 'main' ? computeUiMask(s.glyphs, s.b, s.w, s.h) : null;
		const maxY = Math.min(MAP_H, s.h);
		const maxX = Math.min(MAP_W, s.w);

		for (let gy = 0; gy < maxY; gy++) {
			for (let gx = 0; gx < maxX; gx++) {
				const idx = gy * s.w + gx;
				if (uiMask && uiMask[idx]) continue;
				const g = s.glyphs[idx];
				const x = gx * CW;
				const y = gy * CH;

				if (!g || g === ' ') {
					// Undiscovered fog wash
					ctx.fillStyle = 'rgba(154, 128, 86, 0.28)';
					ctx.fillRect(x, y, CW, CH);
					continue;
				}

				const fgIdx = s.f.charCodeAt(idx) - 65;
				const fgHex = palette[fgIdx];

				// Streets and roads
				if (g === '∙' || g === '=') {
					ctx.fillStyle = '#c49858';
					ctx.fillRect(x, y, CW, CH);
					ctx.fillStyle = 'rgba(255, 245, 220, 0.35)';
					ctx.fillRect(x, y, CW, 1);
					continue;
				}

				// Buildings, walls, pillars, barricades
				if (g === '#' || g === 'O' || g === '⊞' || g === 'X' || g === '■' || g === '≡' || g === '▬') {
					ctx.fillStyle = '#473322';
					ctx.fillRect(x, y, CW, CH);
					ctx.fillStyle = '#6c5036';
					ctx.fillRect(x, y, CW, 1);
					continue;
				}

				// Doors
				if (g === '+' || g === '/') {
					ctx.fillStyle = '#9c5e26';
					ctx.fillRect(x, y, CW, CH);
					continue;
				}

				// Water, wells, waterfalls, lava
				if (g === '~' || g === '≈' || g === '║' || g === 'o') {
					if (g === '≈' && isWarmRedHex(fgHex)) {
						ctx.fillStyle = '#c94c24';
					} else if (g === '≈') {
						ctx.fillStyle = '#3b6d8c';
					} else {
						ctx.fillStyle = '#5c91ab';
					}
					ctx.fillRect(x, y, CW, CH);
					continue;
				}

				// Shrines, altars, statues, thrones, braziers
				if (g === '╥' || g === 'Ω' || g === '¶' || g === 'Ж' || g === 'Ψ') {
					ctx.fillStyle = '#d1971c';
					ctx.fillRect(x, y, CW, CH);
					ctx.fillStyle = '#fff3c4';
					ctx.fillRect(x + 1, y + 1, 1, 2);
					continue;
				}

				// Market stalls and gold
				if (g === 'π' || g === '$') {
					ctx.fillStyle = '#c45b2c';
					ctx.fillRect(x, y, CW, CH);
					ctx.fillStyle = '#f2c17e';
					ctx.fillRect(x, y, CW, 1);
					continue;
				}

				// Stairs, gates, portals
				if (g === '>' || g === '<' || g === '∏' || g === '╫' || g === '◊' || g === 'Ø' || g === '↑') {
					ctx.fillStyle = '#5e3d8a';
					ctx.fillRect(x, y, CW, CH);
					ctx.fillStyle = '#e5d4ff';
					ctx.fillRect(x + 1, y + 1, 1, 2);
					continue;
				}

				// Trees and foliage
				if (g === '♣' || g === 'τ' || g === '♠') {
					ctx.fillStyle = '#657a44';
					ctx.fillRect(x, y, CW, CH);
					continue;
				}

				// Grass and crops
				if (g === '"') {
					ctx.fillStyle = '#9cab6f';
					ctx.fillRect(x, y, CW, CH);
					continue;
				}

				// Cliffs, rubble, ice
				if (g === '▲' || g === '%') {
					ctx.fillStyle = '#7d7262';
					ctx.fillRect(x, y, CW, CH);
					continue;
				}
				if (g === '░' || g === '▒') {
					ctx.fillStyle = '#b4c6cc';
					ctx.fillRect(x, y, CW, CH);
					continue;
				}

				// Discovered open floor / ground / sand / moss
				if (g === '.' || g === '·' || g === ':' || g === ',') {
					ctx.fillStyle = g === '·' ? 'rgba(255, 249, 235, 0.58)' : 'rgba(248, 239, 218, 0.42)';
					ctx.fillRect(x, y, CW, CH);
					continue;
				}

				// Fallback discovered feature (items, machinery, etc.)
				ctx.fillStyle = 'rgba(244, 234, 210, 0.45)';
				ctx.fillRect(x, y, CW, CH);
			}
		}

		// Tiny compass rose "N" in top-right corner
		ctx.fillStyle = 'rgba(72, 48, 26, 0.65)';
		ctx.font = 'bold 8px Georgia, serif';
		ctx.textAlign = 'center';
		ctx.textBaseline = 'top';
		ctx.fillText('N', W - 9, 3);
		ctx.beginPath();
		ctx.moveTo(W - 9, 12);
		ctx.lineTo(W - 11, 17);
		ctx.lineTo(W - 7, 17);
		ctx.closePath();
		ctx.fill();

		return terrainBuf;
	}

	function paint(t: number) {
		const c = canvas;
		const s = snap;
		if (!c || !s || !open) return;

		const dpr = window.devicePixelRatio || 1;
		const tw = Math.round(W * dpr);
		const th = Math.round(H * dpr);
		if (c.width !== tw || c.height !== th) {
			c.width = tw;
			c.height = th;
		}
		const ctx = c.getContext('2d');
		if (!ctx) return;

		ctx.setTransform(dpr, 0, 0, dpr, 0, 0);
		ctx.drawImage(terrainBuf ?? renderTerrain(s), 0, 0);

		// Draw non-player actors
		for (const a of s.actors) {
			if (a.p) continue;
			const ax = a.x * CW + CW * 0.5;
			const ay = a.y * CH + CH * 0.5;
			if (a.ho) {
				// Hostile: crimson diamond
				ctx.fillStyle = '#3a0b0b';
				ctx.fillRect(ax - 2, ay - 2, 4, 4);
				ctx.fillStyle = '#c82828';
				ctx.fillRect(ax - 1.5, ay - 1.5, 3, 3);
			} else if (a.r && /merchant|trader|inn|tavern|barkeep|priest|healer|smith/i.test(a.r)) {
				// Friendly notable (merchant, innkeeper, healer)
				ctx.fillStyle = '#473008';
				ctx.fillRect(ax - 1.5, ay - 1.5, 3, 3);
				ctx.fillStyle = '#df9c22';
				ctx.fillRect(ax - 1, ay - 1, 2, 2);
			} else {
				// Neutral / friendly citizen or guard
				ctx.fillStyle = '#1f5e61';
				ctx.fillRect(ax - 1, ay - 1, 2.2, 2.2);
			}
		}

		// Pulsing player marker
		if (s.playerX >= 0 && s.playerY >= 0) {
			const px = s.playerX * CW + CW * 0.5;
			const py = s.playerY * CH + CH * 0.5;
			const pulse = reduceMotion ? 0.5 : 0.5 + 0.5 * Math.sin(t * 0.0055);
			const haloR = 3.2 + pulse * 2.6;

			ctx.beginPath();
			ctx.arc(px, py, haloR, 0, Math.PI * 2);
			ctx.fillStyle = `rgba(26, 122, 92, ${(0.22 + pulse * 0.24).toFixed(3)})`;
			ctx.fill();

			ctx.beginPath();
			ctx.arc(px, py, 2.5, 0, Math.PI * 2);
			ctx.fillStyle = '#145a44';
			ctx.fill();
			ctx.strokeStyle = '#fff8dc';
			ctx.lineWidth = 1;
			ctx.stroke();
		}
	}

	// Refresh snapshot and terrain buffer whenever frame changes
	$effect(() => {
		const f = frame;
		if (!f) return;
		if (MAP_MODES.has(f.mode) || !snap) {
			const next = buildSnapshot(f);
			snap = next;
			renderTerrain(next);
		}
		if (open) paint(performance.now());
	});

	// Run lightweight pulse animation loop only while open and motion is allowed
	$effect(() => {
		if (rafId) {
			cancelAnimationFrame(rafId);
			rafId = 0;
		}
		if (!open) return;
		if (reduceMotion) {
			paint(0);
			return;
		}
		const step = (t: number) => {
			paint(t);
			rafId = requestAnimationFrame(step);
		};
		rafId = requestAnimationFrame(step);
		return () => {
			if (rafId) {
				cancelAnimationFrame(rafId);
				rafId = 0;
			}
		};
	});

	onDestroy(() => {
		if (rafId) cancelAnimationFrame(rafId);
	});

	let title = $derived.by(() => {
		const s = snap;
		if (!s) return 'Zone Map';
		if (s.mode === 'world') return 'Overworld Atlas';
		const base =
			PLACE_TITLES[s.place] ??
			(s.mood ? s.mood.charAt(0).toUpperCase() + s.mood.slice(1) : 'Local Area');
		return s.depth > 0 ? `${base} · D${s.depth}` : base;
	});

	let timeLabel = $derived.by(() => {
		const s = snap;
		if (!s) return '';
		if (s.ph !== undefined) {
			if (s.ph >= 0.44 && s.ph <= 0.56) return 'Noon';
			if (s.ph >= 0.20 && s.ph < 0.30) return 'Dawn';
			if (s.ph >= 0.70 && s.ph <= 0.82) return 'Dusk';
			if (s.ph > 0.30 && s.ph < 0.70) return 'Day';
			return 'Night';
		}
		return s.day ? 'Day' : 'Night';
	});

	let eventTag = $derived.by(() => {
		const ev = snap?.event;
		if (!ev) return '';
		return ev.charAt(0).toUpperCase() + ev.slice(1);
	});
</script>

{#if open}
	<aside class="minimap-card" aria-label="District minimap">
		<header class="mm-head">
			<span class="mm-title" title={title}>{title}</span>
			<span class="mm-meta">
				{#if eventTag}<span class="mm-badge">{eventTag}</span>{/if}
				{#if timeLabel}<span class="mm-time">{timeLabel}</span>{/if}
				<button
					type="button"
					class="mm-close"
					onclick={(e) => {
						e.currentTarget.blur();
						onclose();
					}}
					aria-label="Hide minimap"
					title="Hide minimap"
				>×</button>
			</span>
		</header>
		<div class="mm-frame">
			<canvas bind:this={canvas} style:width="{W}px" style:height="{H}px" aria-label="{title} minimap"></canvas>
		</div>
		<footer class="mm-legend" aria-hidden="true">
			<span><i style:background="#c49858"></i>Road</span>
			<span><i style:background="#5c91ab"></i>Water</span>
			<span><i style:background="#d1971c"></i>Shrine</span>
			<span><i style:background="#c45b2c"></i>Stall</span>
			<span><i style:background="#5e3d8a"></i>Gate</span>
			<span><i style:background="#c82828"></i>Foe</span>
		</footer>
	</aside>
{/if}

<style>
	.minimap-card {
		position: fixed;
		top: 34px;
		right: 8px;
		z-index: 15;
		width: 254px;
		padding: 5px 6px 5px;
		background: linear-gradient(165deg, #efe1c1 0%, #dfc99f 100%);
		border: 1px solid #6b4c2a;
		box-shadow:
			inset 0 0 0 1px rgba(255, 248, 228, 0.55),
			0 6px 18px rgba(10, 8, 4, 0.55);
		border-radius: 3px;
		color: #3a2614;
		font-family: 'Share Tech Mono', 'DejaVu Sans Mono', monospace;
		user-select: none;
		pointer-events: auto;
	}
	.mm-head {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 6px;
		margin-bottom: 4px;
		padding-bottom: 3px;
		border-bottom: 1px solid rgba(107, 76, 42, 0.35);
	}
	.mm-title {
		font-size: 11px;
		font-weight: 700;
		letter-spacing: 0.02em;
		color: #2f1d0e;
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}
	.mm-meta {
		display: flex;
		align-items: center;
		gap: 4px;
		flex-shrink: 0;
	}
	.mm-badge {
		font-size: 9px;
		padding: 0 4px;
		border-radius: 2px;
		background: rgba(152, 46, 32, 0.16);
		border: 1px solid rgba(152, 46, 32, 0.45);
		color: #7c1f14;
		text-transform: uppercase;
	}
	.mm-time {
		font-size: 10px;
		color: #6b4c2a;
	}
	.mm-close {
		border: 1px solid rgba(107, 76, 42, 0.45);
		background: rgba(255, 248, 232, 0.55);
		color: #3a2614;
		border-radius: 2px;
		width: 16px;
		height: 16px;
		padding: 0;
		line-height: 14px;
		font-size: 12px;
		cursor: pointer;
	}
	.mm-close:hover {
		background: rgba(107, 76, 42, 0.2);
	}
	.mm-frame {
		border: 1px solid #7d5b35;
		background: #e2cfa8;
		overflow: hidden;
		line-height: 0;
	}
	.mm-frame canvas {
		display: block;
		cursor: default;
	}
	.mm-legend {
		display: flex;
		flex-wrap: wrap;
		justify-content: space-between;
		gap: 4px;
		margin-top: 4px;
		font-size: 9px;
		color: #5a3e22;
	}
	.mm-legend span {
		display: inline-flex;
		align-items: center;
		gap: 3px;
	}
	.mm-legend i {
		display: inline-block;
		width: 6px;
		height: 6px;
		border-radius: 1px;
		border: 1px solid rgba(47, 29, 14, 0.35);
	}
</style>
