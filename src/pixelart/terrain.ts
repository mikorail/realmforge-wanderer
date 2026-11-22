import type { Rows } from './types';
import { natureSprite } from './terrain_nature';
import { buildSprite } from './terrain_build';

// Walls must never read as floor. Walls get a raised look (light top edge, dark base and sides where the wall
// ends) and floors are one step darker and flatter, so a pale cobble street and a rough stone wall differ.
// two steps down and flattened: the brightest floor tone is only a little lighter than its base
const DIM: Record<string, string> = { L: 's', l: 's', F: 'S', d: 'Z', D: 'Z' };
const LIFT: Record<string, string> = { D: 'd', d: 'F', F: 'l', l: 'L', L: 'L' };
const SINK: Record<string, string> = { L: 'l', l: 'F', F: 'd', d: 'D', D: 'o' };

const map = (rows: Rows, f: (ch: string, x: number, y: number) => string): string[] => rows.map((r, y) => r.split('').map((c, x) => (c === '.' ? c : f(c, x, y))).join(''));

function dimFloor(rows: Rows): Rows {
	return map(rows, (c) => DIM[c] ?? c);
}

/** nm: N=1 E=2 S=4 W=8 are set where the neighbour is the same kind (0 = unknown: treat every side as open) */
function bevelWall(rows: Rows, nm: number): Rows {
	const open = (bit: number) => nm === 0 || !(nm & bit);
	const n = open(1), e = open(2), s = open(4), w = open(8);
	const H = rows.length;
	return map(rows, (c, x, y) => {
		const W = rows[y].length;
		if (s && y >= H - 2) return y === H - 1 ? 'o' : (SINK[c] ?? c); // dark base where the wall ends
		if (n && y === 0) return LIFT[LIFT[c] ?? c] ?? c; // lit top edge
		if (e && x >= W - 1) return SINK[c] ?? c;
		if (w && x === 0) return LIFT[c] ?? c;
		return c;
	});
}

/** null = not covered (the old drawer runs). h = stable per-cell hash, frame = 0..7 ambient animation clock,
 *  nm = neighbour bitmask (N=1 E=2 S=4 W=8 NE=16 SE=32 SW=64 NW=128: same kind of tile around it). */
export function terrainSprite(g: string, fgIdx: number, bgIdx: number, h: number, frame: number, nm: number): Rows | null {
	const r = natureSprite(g, fgIdx, bgIdx, h, frame, nm) ?? buildSprite(g, fgIdx, bgIdx, h, frame, nm);
	if (!r) return null;
	if (g === '#') return bevelWall(r, nm);
	if (g === '·' || g === '=' || g === '∙' || (g === '.' && (fgIdx === 1 || fgIdx === 2 || fgIdx === 4 || fgIdx === 5 || fgIdx === 9))) return dimFloor(r);
	return r;
}
