// Entry point: pixel-art version of a map cell. Returns true when it drew the cell; false = use the old drawer.
import { blitSprite } from './render';
import { shift, flipX, type Rows } from './types';
import { terrainSprite } from './terrain';
import { personSprite } from './people';
import { creatureSprite } from './creatures';
import type { QudActor } from '../actorsprites';
import type { Pose } from '../poses';

/** things that stand on the ground and throw a shadow (the rest is ground, floor and walls) */
const OBJECTS = new Set('♣♠↑τΓφ‼ⁿ▲%*¥δOΩ╥π▬¶ΨЖŦ■Θ§≡∩Φ+/⊞∏╫|‡XΛo'.split(''));

let on = true;
export const setPixelArt = (v: boolean): void => {
	on = v;
};

function hash(x: number, y: number): number {
	let h = (x * 374761393 + y * 668265263) | 0;
	h = Math.imul(h ^ (h >>> 13), 1274126177);
	return (h ^ (h >>> 16)) >>> 0;
}

export function drawPixelTile(ctx: CanvasRenderingContext2D, g: string, fgIdx: number, bgIdx: number, fg: string, bg: string, gx: number, gy: number, x: number, y: number, cw: number, chh: number, frame: number, nm: number): boolean {
	if (!on) return false;
	const rows = terrainSprite(g, fgIdx, bgIdx, hash(gx, gy), frame, nm);
	if (!rows) return false;
	blitSprite(ctx, rows, x, y, cw, chh, { fg, bg }, OBJECTS.has(g) ? 'drop' : 'none');
	return true;
}

export function drawPixelActor(ctx: CanvasRenderingContext2D, a: QudActor, g: string, fg: string, bg: string, x: number, y: number, cw: number, chh: number, frame: number, pose: Pose): boolean {
	if (!on) return false;
	const h = hash(a.i, a.i >> 3);
	const person = !!(a.s || a.p);
	const r = person ? personSprite(a, h, pose, frame) : creatureSprite(a, g, h, frame);
	if (!r) return false;
	const rows = person ? r.rows : creatureMotion(r.rows, pose);
	blitSprite(ctx, rows, x, y, cw, chh, { fg, bg, ...r.pal }, pose.dead > 0.5 ? 'drop' : 'stand');
	return true;
}

/** creatures have one still sprite per frame: attack, hurt and death are done by moving and flipping it */
function creatureMotion(rows: Rows, pose: Pose): Rows {
	let out: Rows = rows;
	if (pose.dead > 0) {
		if (pose.dead > 0.5) {
			// fallen: belly up, settled on the ground
			const flipped = [...out].reverse();
			let last = flipped.length - 1;
			while (last > 0 && /^\.*$/.test(flipped[last])) last--;
			return shift(flipX(flipped), 0, flipped.length - 1 - last);
		}
		return shift(out, pose.dir, 1);
	}
	if (pose.attack > 0) {
		if (pose.face < 0) out = flipX(out);
		const lunge = pose.attack < 0.4 ? -1 : pose.attack < 0.7 ? 2 : 1; // draw back, then strike
		out = shift(out, lunge * (pose.face < 0 ? -1 : 1), 0);
	} else if (pose.hurt > 0) {
		out = shift(out, pose.hurt < 0.5 ? pose.dir : 0, 0);
	}
	return out;
}

/**
 * A big portrait of one actor: the real pixel sprite scaled by whole pixels when it exists; false = not covered.
 * (sx, sy, sw, sh) is the css rect; the sprite is bottom-centred in it.
 */
export function drawPixelPortrait(ctx: CanvasRenderingContext2D, a: QudActor, fg: string, bg: string, sx: number, sy: number, sw: number, sh: number, frame: number, pose: Pose): boolean {
	if (!on) return false;
	const h = hash(a.i, a.i >> 3);
	const r = a.s || a.p ? personSprite(a, h, pose, frame) : creatureSprite(a, ' ', h, frame);
	if (!r) return false;
	blitSprite(ctx, r.rows, sx, sy, sw, sh, { fg, bg, ...r.pal });
	return true;
}
