// Pixel pass: every sprite is drawn by the vector-style drawers (arcs, ellipses, rotations) into a TINY
// scratch canvas, then hard-thresholded (no anti-aliasing), snapped to a small palette, given a dark
// outline and blitted with nearest-neighbour scaling. The result is chunky 2D pixel art instead of smooth
// vector shapes. The drawers do not change: they just draw into a grid of art pixels.

type Ctx = CanvasRenderingContext2D;

export interface PixelOpts {
	/** 1 art-pixel dark outline around opaque pixels (actors, portraits) */
	outline?: boolean;
	/** colour levels per channel after snapping (default 6 = 216 colours) */
	levels?: number;
	/** alpha cut-off 0..255 (default 110) */
	cut?: number;
}

interface Scratch {
	a: HTMLCanvasElement;
	ac: Ctx;
	b: HTMLCanvasElement;
	bc: Ctx;
	w: number;
	h: number;
}

const pool = new Map<string, Scratch>();

function scratch(w: number, h: number): Scratch {
	const k = w + 'x' + h;
	let s = pool.get(k);
	if (s) return s;
	const a = document.createElement('canvas');
	a.width = w;
	a.height = h;
	const b = document.createElement('canvas');
	b.width = w;
	b.height = h;
	const ac = a.getContext('2d', { willReadFrequently: true }) as Ctx;
	const bc = b.getContext('2d') as Ctx;
	s = { a, ac, b, bc, w, h };
	pool.set(k, s);
	if (pool.size > 24) pool.delete(pool.keys().next().value as string);
	return s;
}

/** the art grid for a cell of cw x chh css pixels: one art pixel is u device pixels, uniform. */
export function cellGrid(cw: number, chh: number, dpr: number): { w: number; h: number; u: number } {
	const u = Math.max(2, Math.round((cw * dpr) / 12));
	return { w: Math.max(6, Math.floor((cw * dpr) / u)), h: Math.max(8, Math.floor((chh * dpr) / u)), u };
}

/**
 * Draw into a gw x gh art grid and blit it, pixelated, into the css rect (dx, dy, dw, dh) of ctx.
 * `draw` gets the scratch context and the grid size and paints at (0, 0, gw, gh); return false to paint nothing.
 */
export function pixelBlit(ctx: Ctx, dx: number, dy: number, dw: number, dh: number, gw: number, gh: number, draw: (c: Ctx, w: number, h: number) => boolean, o: PixelOpts = {}): boolean {
	const s = scratch(gw, gh);
	const c = s.ac;
	c.setTransform(1, 0, 0, 1, 0, 0);
	c.globalAlpha = 1;
	c.clearRect(0, 0, gw, gh);
	c.save();
	c.imageSmoothingEnabled = false;
	let ok = false;
	try {
		ok = draw(c, gw, gh);
	} finally {
		c.restore();
	}
	if (!ok) return false;
	const img = c.getImageData(0, 0, gw, gh);
	const d = img.data;
	const cut = o.cut ?? 110;
	const L = Math.max(2, o.levels ?? 6) - 1;
	const step = 255 / L;
	const n = gw * gh;
	const solid = new Uint8Array(n);
	for (let i = 0; i < n; i++) {
		const p = i * 4;
		if (d[p + 3] < cut) {
			d[p + 3] = 0;
			continue;
		}
		// un-premultiply is already done by getImageData; snap each channel to a few levels
		d[p] = Math.round(d[p] / step) * step;
		d[p + 1] = Math.round(d[p + 1] / step) * step;
		d[p + 2] = Math.round(d[p + 2] / step) * step;
		d[p + 3] = 255;
		solid[i] = 1;
	}
	if (o.outline) {
		const out = new Uint8ClampedArray(d);
		for (let y = 0; y < gh; y++) {
			for (let x = 0; x < gw; x++) {
				const i = y * gw + x;
				if (solid[i]) continue;
				const near = (x > 0 && solid[i - 1]) || (x < gw - 1 && solid[i + 1]) || (y > 0 && solid[i - gw]) || (y < gh - 1 && solid[i + gw]);
				if (near) {
					const p = i * 4;
					out[p] = 12;
					out[p + 1] = 10;
					out[p + 2] = 16;
					out[p + 3] = 230;
				}
			}
		}
		d.set(out);
	}
	s.bc.putImageData(img, 0, 0);
	const prev = ctx.imageSmoothingEnabled;
	ctx.imageSmoothingEnabled = false;
	ctx.drawImage(s.b, dx, dy, dw, dh);
	ctx.imageSmoothingEnabled = prev;
	return true;
}

/** Paint one map cell through the pixel pass. `draw` paints a cell at (0, 0, w, h) of the art grid. */
export function pixelCell(ctx: Ctx, x: number, y: number, cw: number, chh: number, dpr: number, outline: boolean, draw: (c: Ctx, w: number, h: number) => boolean): boolean {
	const g = cellGrid(cw, chh, dpr);
	const dw = (g.w * g.u) / dpr, dh = (g.h * g.u) / dpr;
	return pixelBlit(ctx, x + (cw - dw) / 2, y + (chh - dh) / 2, dw, dh, g.w, g.h, draw, { outline });
}
