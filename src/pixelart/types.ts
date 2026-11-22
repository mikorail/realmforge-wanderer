// Hand-authored pixel art. A sprite is PH rows of PW characters; one character = one art pixel.
// Sprites are drawn with nearest-neighbour scaling (no smoothing, no vectors) by render.ts.
//
// PALETTE (one char per pixel):
//   .  transparent
//   o  outline (near black, tinted by the cell background)
//   ramp of the tile's own colour (fg): D darkest, d dark, F base, l light, L lightest
//   floor tones (the tile colour sunk towards the background): s light, S base, Z dark
//   skin ramp (species/individual): K dark, k base, j light
//   hair ramp: H dark, h base
//   accent (a second cloth/trim colour): A dark, a base
//   fixed colours: w wood, W dark wood, t tan/sand, T dark sand, g stone, G dark stone, m metal, M dark metal,
//     e leaf green, E dark green, u water blue, U deep blue, c cyan, r red, R dark red, y gold, Y dark gold,
//     x white, z black, p purple, n orange, q pink, v bright green
// Everything not listed is transparent (and a bug: keep to this table).

export const PW = 12;
export const PH = 20;
export type Rows = readonly string[];

/** colours a sprite is drawn with (css hex strings) */
export interface PCtx {
	fg: string;
	bg: string;
	skin?: string;
	hair?: string;
	accent?: string;
}

/** a row-set builder that checks the size once in dev: pad or trim to PW x PH */
export function mk(...rows: string[]): Rows {
	const out: string[] = [];
	for (let y = 0; y < PH; y++) out.push((rows[y] ?? '').padEnd(PW, '.').slice(0, PW));
	return out;
}

/** a blank PW x PH sprite */
export const blank = (): string[] => Array.from({ length: PH }, () => '.'.repeat(PW));

/** draw `top` over `base` (top's '.' is see-through), with an offset */
export function over(base: Rows, top: Rows, dx = 0, dy = 0): string[] {
	const out = base.map((r) => r.split(''));
	for (let y = 0; y < top.length; y++) {
		for (let x = 0; x < top[y].length; x++) {
			const c = top[y][x];
			const ty = y + dy, tx = x + dx;
			if (c !== '.' && ty >= 0 && ty < out.length && tx >= 0 && tx < PW) out[ty][tx] = c;
		}
	}
	return out.map((r) => r.join(''));
}

/** mirror left-right */
export const flipX = (rows: Rows): string[] => rows.map((r) => r.split('').reverse().join(''));

/** shift by dx, dy (pixels leave the sprite) */
export function shift(rows: Rows, dx: number, dy: number): string[] {
	return over(blank(), rows, dx, dy);
}
