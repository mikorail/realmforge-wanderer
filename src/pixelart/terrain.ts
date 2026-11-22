import type { Rows } from './types';
import { natureSprite } from './terrain_nature';
import { buildSprite } from './terrain_build';
/** null = not covered (the old drawer runs). h = stable per-cell hash, frame = 0..7 ambient animation clock,
 *  nm = neighbour bitmask (N=1 E=2 S=4 W=8 NE=16 SE=32 SW=64 NW=128: same kind of tile around it). */
export function terrainSprite(g: string, fgIdx: number, bgIdx: number, h: number, frame: number, nm: number): Rows | null {
	return natureSprite(g, fgIdx, bgIdx, h, frame, nm) ?? buildSprite(g, fgIdx, bgIdx, h, frame, nm);
}
