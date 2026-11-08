/**
 * Procedural Canvas2D tile, environment, character, and creature sprite renderer
 * for Realmforge Wanderer. Turns the 80x25 map grid into varied procedural pixel/vector
 * miniatures while keeping UI boxes, sidebars, and logs in crisp monospace text.
 */

import { drawPersonSprite, drawCreatureSprite2, type QudActor } from './actorsprites';

const MAP_W = 80;
const MAP_H = 25;

function hashCell(x: number, y: number, salt = 0): number {
	let h = (x * 374761393 + y * 668265263 + salt * 1442695041) | 0;
	h = Math.imul(h ^ (h >>> 13), 1274126177);
	return (h ^ (h >>> 16)) >>> 0;
}

/**
 * Detects which cells inside the 80x25 map area belong to an overlay UI box or top banner
 * so they are always rendered as crisp text instead of map sprites.
 */
export function computeUiMask(glyphs: string[], bStr: string, w: number, h: number): Uint8Array {
	const mask = new Uint8Array(w * h);
	// Check top row banner (e.g. World map header, Look/Target prompt)
	let topDark = 0;
	for (let x = 1; x < MAP_W - 1; x++) {
		if (bStr.charCodeAt(x) - 65 === 20) topDark++;
	}
	if (topDark > 20) {
		for (let x = 0; x < MAP_W; x++) mask[x] = 1;
	}

	// Find any box corners ┌ ... ┐ and mark their bounding rectangles
	for (let y = 0; y < MAP_H; y++) {
		for (let x = 0; x < MAP_W; x++) {
			const i = y * w + x;
			if (glyphs[i] === '┌') {
				let x2 = x + 1;
				while (x2 < w && (glyphs[y * w + x2] === '─' || glyphs[y * w + x2] !== '┐')) {
					if (glyphs[y * w + x2] === '┐') break;
					x2++;
				}
				let y2 = y + 1;
				while (y2 < h && glyphs[y2 * w + x] !== '└') {
					y2++;
				}
				if (x2 < w && y2 < h) {
					for (let yy = y; yy <= y2; yy++) {
						for (let xx = x; xx <= x2; xx++) {
							mask[yy * w + xx] = 1;
						}
					}
				}
			}
		}
	}
	return mask;
}

/**
 * Attempt to draw a procedural sprite for a map cell at (gx, gy).
 * Returns true if the cell was drawn as a procedural sprite, or false to fall back to text.
 */
export function drawProceduralCell(
	ctx: CanvasRenderingContext2D,
	g: string,
	fgIdx: number,
	bgIdx: number,
	fg: string,
	palette: string[],
	gx: number,
	gy: number,
	x: number,
	y: number,
	cw: number,
	chh: number,
	actor?: QudActor,
): boolean {
	if (gx >= MAP_W || gy >= MAP_H) return false;
	const h = hashCell(gx, gy);
	const cx = x + cw * 0.5;
	const cy = y + chh * 0.5;

	// Walls & unique architectural materials (#)
	if (g === "#") {
		ctx.save();
		ctx.fillStyle = fg;
		ctx.strokeStyle = fg;
		if (fgIdx === 14) {
			// Timber / Wood wall (notched horizontal logs)
			ctx.lineWidth = Math.max(1, cw * 0.1);
			for (let k = 0; k < 3; k++) {
				const ly = y + chh * (0.2 + k * 0.3);
				ctx.beginPath();
				ctx.moveTo(x + cw * 0.05, ly);
				ctx.lineTo(x + cw * 0.95, ly);
				ctx.stroke();
			}
			ctx.fillRect(x + cw * 0.45, y + chh * 0.05, cw * 0.1, chh * 0.9);
		} else if (fgIdx === 4 || fgIdx === 5) {
			// Brick & Adobe wall (staggered courses)
			ctx.lineWidth = 1;
			for (let k = 0; k < 4; k++) {
				const by = y + chh * (k * 0.25);
				ctx.fillRect(x + cw * 0.05, by, cw * 0.9, chh * 0.2);
				ctx.fillStyle = palette[18] || "#041312";
				ctx.fillRect(x + cw * (0.25 + ((k % 2) * 0.3)), by, 1, chh * 0.2);
				ctx.fillStyle = fg;
			}
		} else if (fgIdx === 6 || fgIdx === 7) {
			// Bamboo / Thorn hedge
			ctx.lineWidth = Math.max(1, cw * 0.12);
			for (let k = 0; k < 3; k++) {
				const bx = x + cw * (0.2 + k * 0.3);
				ctx.beginPath();
				ctx.moveTo(bx, y + chh * 0.95);
				ctx.lineTo(bx, y + chh * 0.05);
				ctx.stroke();
			}
		} else {
			// Dressed stone / marble / metal ashlar block
			ctx.lineWidth = Math.max(1, cw * 0.08);
			ctx.strokeRect(x + cw * 0.1, y + chh * 0.08, cw * 0.8, chh * 0.84);
			ctx.fillRect(x + cw * 0.2, y + chh * 0.18, cw * 0.6, chh * 0.64);
		}
		ctx.restore();
		return true;
	}

	// Latticed Window (⊞)
	if (g === "⊞") {
		ctx.save();
		ctx.fillStyle = fg;
		ctx.strokeStyle = fg;
		ctx.lineWidth = Math.max(1, cw * 0.1);
		ctx.strokeRect(x + cw * 0.15, y + chh * 0.15, cw * 0.7, chh * 0.7);
		ctx.beginPath();
		ctx.moveTo(cx, y + chh * 0.15);
		ctx.lineTo(cx, y + chh * 0.85);
		ctx.moveTo(x + cw * 0.15, cy);
		ctx.lineTo(x + cw * 0.85, cy);
		ctx.stroke();
		ctx.restore();
		return true;
	}

	// Gate Arch (∏) & Portcullis (╫)
	if (g === "∏" || g === "╫") {
		ctx.save();
		ctx.fillStyle = fg;
		ctx.strokeStyle = fg;
		ctx.lineWidth = Math.max(1.2, cw * 0.12);
		ctx.beginPath();
		ctx.moveTo(x + cw * 0.15, y + chh * 0.95);
		ctx.lineTo(x + cw * 0.15, y + chh * 0.35);
		ctx.quadraticCurveTo(cx, y + chh * 0.05, x + cw * 0.85, y + chh * 0.35);
		ctx.lineTo(x + cw * 0.85, y + chh * 0.95);
		ctx.stroke();
		if (g === "╫") {
			// Portcullis bars
			ctx.lineWidth = 1;
			for (let k = 0; k < 3; k++) {
				const bx = x + cw * (0.3 + k * 0.2);
				ctx.beginPath();
				ctx.moveTo(bx, y + chh * 0.25);
				ctx.lineTo(bx, y + chh * 0.95);
				ctx.stroke();
			}
		}
		ctx.restore();
		return true;
	}

	// Timber plank floor (=) & Scaffolding (≡)
	if (g === "=" || g === "≡") {
		ctx.save();
		ctx.strokeStyle = fg;
		ctx.lineWidth = Math.max(1, cw * 0.09);
		for (let k = 0; k < 3; k++) {
			const py = y + chh * (0.22 + k * 0.28);
			ctx.beginPath();
			ctx.moveTo(x + cw * 0.1, py);
			ctx.lineTo(x + cw * 0.9, py);
			ctx.stroke();
		}
		if (g === "≡") {
			ctx.beginPath();
			ctx.moveTo(x + cw * 0.25, y + chh * 0.1);
			ctx.lineTo(x + cw * 0.75, y + chh * 0.9);
			ctx.stroke();
		}
		ctx.restore();
		return true;
	}

	// 1. Ground & micro-flora scatter (., ∙, ·, :, ')
	if (g === '.' || g === '∙' || g === '·' || g === ':' || g === ',') {
		ctx.save();
		ctx.fillStyle = fg;
		ctx.strokeStyle = fg;
		if (g === ',') {
			// Bones / skeletal remains
			ctx.lineWidth = Math.max(1, cw * 0.12);
			ctx.beginPath();
			ctx.moveTo(x + cw * 0.2, y + chh * 0.72);
			ctx.lineTo(x + cw * 0.8, y + chh * 0.35);
			ctx.moveTo(x + cw * 0.25, y + chh * 0.35);
			ctx.lineTo(x + cw * 0.75, y + chh * 0.72);
			ctx.stroke();
			ctx.beginPath();
			ctx.arc(cx, y + chh * 0.42, cw * 0.18, 0, Math.PI * 2);
			ctx.fill();
			ctx.restore();
			return true;
		}
		if (fgIdx === 6 || fgIdx === 7) {
			// Grass blades with procedural lean and count
			const count = 2 + (h % 3);
			ctx.lineWidth = Math.max(1, cw * 0.1);
			for (let k = 0; k < count; k++) {
				const bx = x + cw * (0.22 + ((k * 0.27 + ((h >>> (k * 3)) & 7) * 0.03) % 0.6));
				const by = y + chh * (0.62 + ((h >>> (k * 2)) & 3) * 0.07);
				const lean = (((h >>> (k * 4)) & 7) - 3.5) * cw * 0.06;
				const bh = chh * (0.18 + ((h >>> (k + 5)) & 3) * 0.05);
				ctx.beginPath();
				ctx.moveTo(bx, by);
				ctx.quadraticCurveTo(bx + lean * 0.5, by - bh * 0.6, bx + lean, by - bh);
				ctx.stroke();
			}
			if ((h & 15) === 0) {
				// Wildflower speck
				ctx.fillStyle = palette[(h & 16) ? 15 : 13] || fg;
				ctx.fillRect(cx + ((h & 3) - 1.5), cy, 2, 2);
			}
			ctx.restore();
			return true;
		}
		if (fgIdx === 14 || fgIdx === 15) {
			// Sand ripples or desert pebbles
			ctx.globalAlpha = 0.65;
			ctx.lineWidth = 1;
			const ry = y + chh * (0.45 + (h & 3) * 0.08);
			ctx.beginPath();
			ctx.moveTo(x + cw * 0.15, ry);
			ctx.quadraticCurveTo(cx, ry - chh * 0.08, x + cw * 0.85, ry);
			ctx.stroke();
			ctx.restore();
			return true;
		}
		// Stone / dirt / ash / snow speckles
		const dots = 1 + (h % 3);
		ctx.globalAlpha = 0.7;
		for (let k = 0; k < dots; k++) {
			const dx = x + cw * (0.2 + (((h >>> (k * 4)) & 15) / 16) * 0.6);
			const dy = y + chh * (0.3 + (((h >>> (k * 4 + 2)) & 15) / 16) * 0.45);
			ctx.fillRect(dx, dy, Math.max(1.5, cw * 0.14), Math.max(1.5, cw * 0.14));
		}
		ctx.restore();
		return true;
	}

	// 2. Tall grass, reeds, crops (")
	if (g === '"') {
		ctx.save();
		ctx.strokeStyle = fg;
		ctx.lineWidth = Math.max(1, cw * 0.11);
		for (let k = 0; k < 4; k++) {
			const bx = x + cw * (0.18 + k * 0.21);
			const lean = (((h >>> (k * 3)) & 7) - 3.5) * cw * 0.07;
			const topY = y + chh * (0.18 + ((h >>> (k * 2)) & 3) * 0.06);
			ctx.beginPath();
			ctx.moveTo(bx, y + chh * 0.86);
			ctx.quadraticCurveTo(bx + lean * 0.4, y + chh * 0.5, bx + lean, topY);
			ctx.stroke();
		}
		ctx.restore();
		return true;
	}

	// 2b. Plant species glyphs from engine/flora.go: palm Γ, cactus φ, bamboo ‼, flower °, stump ⁿ
	if (g === 'Γ' || g === 'φ' || g === '‼' || g === '°' || g === 'ⁿ') {
		ctx.save();
		ctx.fillStyle = fg;
		ctx.strokeStyle = fg;
		ctx.lineWidth = Math.max(1, cw * 0.12);
		if (g === 'Γ') {
			// palm: curved trunk and drooping fronds
			const tx = cx + (((h & 3) - 1.5) * cw) / 16;
			ctx.strokeStyle = palette[14] || fg;
			ctx.beginPath();
			ctx.moveTo(cx, y + chh * 0.9);
			ctx.quadraticCurveTo(cx - cw * 0.08, y + chh * 0.55, tx, y + chh * 0.28);
			ctx.stroke();
			ctx.strokeStyle = fg;
			for (let k = 0; k < 5; k++) {
				const a = Math.PI * (1.05 + k * 0.22);
				ctx.beginPath();
				ctx.moveTo(tx, y + chh * 0.28);
				ctx.quadraticCurveTo(tx + Math.cos(a) * cw * 0.3, y + chh * 0.28 + Math.sin(a) * chh * 0.18, tx + Math.cos(a) * cw * 0.44, y + chh * 0.42);
				ctx.stroke();
			}
		} else if (g === 'φ') {
			// cactus: column with two arms
			const w = cw * 0.2;
			ctx.fillRect(cx - w / 2, y + chh * 0.22, w, chh * 0.66);
			ctx.fillRect(x + cw * 0.18, y + chh * 0.42, w * 0.8, chh * 0.2);
			ctx.fillRect(x + cw * 0.18, y + chh * 0.3, w * 0.8, chh * 0.16);
			ctx.fillRect(cx + w / 2, y + chh * 0.5, cw * 0.18, chh * 0.12);
			ctx.fillRect(x + cw * 0.66, y + chh * 0.36, w * 0.8, chh * 0.2);
		} else if (g === '‼') {
			// bamboo: jointed canes
			for (let k = 0; k < 3; k++) {
				const bx = x + cw * (0.25 + k * 0.25);
				ctx.beginPath();
				ctx.moveTo(bx, y + chh * 0.92);
				ctx.lineTo(bx + (((h >>> k) & 1) - 0.5) * cw * 0.08, y + chh * 0.1);
				ctx.stroke();
				ctx.fillRect(bx - cw * 0.07, y + chh * (0.35 + k * 0.12), cw * 0.14, 1);
			}
		} else if (g === '°') {
			// flower or floating pad: petals round a centre
			const r = cw * 0.11;
			for (let k = 0; k < 5; k++) {
				const a = (k / 5) * Math.PI * 2 + (h & 7);
				ctx.beginPath();
				ctx.arc(cx + Math.cos(a) * r * 1.3, cy + Math.sin(a) * r * 1.3, r, 0, Math.PI * 2);
				ctx.fill();
			}
			ctx.fillStyle = palette[15] || '#cfc041';
			ctx.beginPath();
			ctx.arc(cx, cy, r * 0.8, 0, Math.PI * 2);
			ctx.fill();
		} else {
			// stump: low ringed cylinder
			ctx.fillRect(cx - cw * 0.22, y + chh * 0.55, cw * 0.44, chh * 0.3);
			ctx.globalAlpha = 0.6;
			ctx.fillStyle = palette[15] || fg;
			ctx.beginPath();
			ctx.ellipse(cx, y + chh * 0.55, cw * 0.22, chh * 0.07, 0, 0, Math.PI * 2);
			ctx.fill();
		}
		ctx.restore();
		return true;
	}

	// 3. Trees, pines, giant mushrooms, dead trees (♣, ♠, ↑, τ)
	if (g === '♣' || g === '♠' || g === '↑' || g === 'τ') {
		ctx.save();
		if (fgIdx === 12 || fgIdx === 13) {
			// Giant bioluminescent mushroom
			ctx.fillStyle = '#c8b8a0';
			ctx.fillRect(cx - cw * 0.12, y + chh * 0.48, cw * 0.24, chh * 0.4);
			ctx.fillStyle = fg;
			ctx.beginPath();
			ctx.arc(cx, y + chh * 0.48, cw * 0.42, Math.PI, 0);
			ctx.closePath();
			ctx.fill();
			// Spots
			ctx.fillStyle = palette[3] || '#ffffff';
			ctx.globalAlpha = 0.75;
			ctx.fillRect(cx - cw * 0.18, y + chh * 0.3, 2, 2);
			ctx.fillRect(cx + cw * 0.1, y + chh * 0.34, 2, 2);
			ctx.restore();
			return true;
		}
		if (g === 'τ') {
			// Dead tree / skeletal branches
			ctx.strokeStyle = fg;
			ctx.lineWidth = Math.max(1.2, cw * 0.14);
			ctx.beginPath();
			ctx.moveTo(cx, y + chh * 0.88);
			ctx.lineTo(cx, y + chh * 0.28);
			ctx.moveTo(cx, y + chh * 0.55);
			ctx.lineTo(x + cw * 0.2, y + chh * 0.32);
			ctx.moveTo(cx, y + chh * 0.45);
			ctx.lineTo(x + cw * 0.8, y + chh * 0.25);
			ctx.stroke();
			ctx.restore();
			return true;
		}
		// Trunk
		ctx.fillStyle = palette[16] || '#8b5a2b';
		ctx.fillRect(cx - cw * 0.11, y + chh * 0.55, cw * 0.22, chh * 0.34);
		ctx.fillStyle = fg;
		if (g === '↑' || (h & 3) === 0) {
			// Conifer / tiered pine
			for (let t = 0; t < 3; t++) {
				const ty = y + chh * (0.14 + t * 0.18);
				const tw = cw * (0.28 + t * 0.1);
				ctx.beginPath();
				ctx.moveTo(cx, ty);
				ctx.lineTo(cx + tw, ty + chh * 0.26);
				ctx.lineTo(cx - tw, ty + chh * 0.26);
				ctx.closePath();
				ctx.fill();
			}
		} else {
			// Broadleaf / canopy clusters varied by hash
			const rMain = cw * 0.34;
			ctx.beginPath();
			ctx.arc(cx, y + chh * 0.38, rMain, 0, Math.PI * 2);
			ctx.arc(cx - cw * 0.18, y + chh * 0.48, rMain * 0.78, 0, Math.PI * 2);
			ctx.arc(cx + cw * 0.18, y + chh * 0.48, rMain * 0.78, 0, Math.PI * 2);
			ctx.fill();
		}
		ctx.restore();
		return true;
	}

	// 4. Walls, rock, and ore seams (#, %)
	if (g === '#' || g === '%') {
		ctx.save();
		ctx.fillStyle = fg;
		ctx.globalAlpha = 0.28;
		ctx.fillRect(x + 0.5, y + 0.5, cw - 1, chh - 1);
		ctx.globalAlpha = 0.85;
		ctx.strokeStyle = fg;
		ctx.lineWidth = 1;
		// Procedural stone blocks / crags
		const midY = y + chh * (0.45 + ((h & 3) - 1.5) * 0.05);
		ctx.strokeRect(x + 1, y + 1, cw - 2, chh - 2);
		ctx.beginPath();
		ctx.moveTo(x + 1, midY);
		ctx.lineTo(x + cw - 1, midY);
		const splitX = x + cw * (0.35 + ((h >>> 3) & 3) * 0.1);
		ctx.moveTo(splitX, y + 1);
		ctx.lineTo(splitX, midY);
		ctx.stroke();
		if (g === '%') {
			// Ore vein sparkles
			ctx.fillStyle = palette[15] || '#ffd700';
			ctx.fillRect(x + cw * 0.3, y + chh * 0.28, 2.5, 2.5);
			ctx.fillRect(x + cw * 0.62, y + chh * 0.62, 2.5, 2.5);
		}
		ctx.restore();
		return true;
	}

	// 4b. Landforms (engine/tglandforms.go): cliffs and peaks (▲), waterfalls (║), fumaroles (∴), glacier ice (▒)
	if (g === '▲') {
		ctx.save();
		ctx.fillStyle = fg;
		ctx.beginPath();
		ctx.moveTo(x + cw * 0.08, y + chh * 0.9);
		ctx.lineTo(x + cw * (0.42 + (h & 3) * 0.04), y + chh * 0.14);
		ctx.lineTo(x + cw * 0.92, y + chh * 0.9);
		ctx.closePath();
		ctx.fill();
		ctx.globalAlpha = 0.45;
		ctx.fillStyle = palette[18] || '#041312';
		ctx.fillRect(x + cw * 0.5, y + chh * 0.45, cw * 0.08, chh * 0.4);
		ctx.restore();
		return true;
	}
	if (g === '║') {
		ctx.save();
		ctx.strokeStyle = fg;
		ctx.lineWidth = Math.max(1, cw * 0.1);
		for (let k = 0; k < 3; k++) {
			const sx = x + cw * (0.25 + k * 0.25);
			const off = ((h >>> (k * 3)) & 7) / 7;
			ctx.beginPath();
			ctx.moveTo(sx, y + chh * (0.05 + off * 0.2));
			ctx.lineTo(sx, y + chh * (0.6 + off * 0.35));
			ctx.stroke();
		}
		ctx.restore();
		return true;
	}
	if (g === '∴') {
		ctx.save();
		ctx.fillStyle = fg;
		const r = Math.max(1.2, cw * 0.12);
		for (const [px, py] of [[0.5, 0.3], [0.3, 0.7], [0.7, 0.7]]) {
			ctx.beginPath();
			ctx.arc(x + cw * px, y + chh * py, r, 0, Math.PI * 2);
			ctx.fill();
		}
		ctx.restore();
		return true;
	}
	if (g === '▒') {
		ctx.save();
		ctx.strokeStyle = fg;
		ctx.globalAlpha = 0.6;
		ctx.lineWidth = 1;
		ctx.beginPath();
		ctx.moveTo(x + cw * 0.1, y + chh * (0.35 + (h & 3) * 0.05));
		ctx.lineTo(x + cw * 0.9, y + chh * (0.3 + ((h >>> 2) & 3) * 0.05));
		ctx.moveTo(x + cw * 0.2, y + chh * 0.72);
		ctx.lineTo(x + cw * 0.75, y + chh * 0.68);
		ctx.stroke();
		ctx.restore();
		return true;
	}

	// 5. Water & Lava waves (~, ≈)
	if (g === '~' || g === '≈') {
		ctx.save();
		ctx.strokeStyle = fg;
		ctx.lineWidth = Math.max(1, cw * 0.11);
		const rows = g === '≈' ? 2 : 1;
		for (let r = 0; r < rows; r++) {
			const wy = y + chh * (rows === 2 ? 0.36 + r * 0.3 : 0.52);
			ctx.beginPath();
			ctx.moveTo(x + cw * 0.12, wy);
			ctx.quadraticCurveTo(x + cw * 0.35, wy - chh * 0.1, cx, wy);
			ctx.quadraticCurveTo(x + cw * 0.65, wy + chh * 0.1, x + cw * 0.88, wy);
			ctx.stroke();
		}
		ctx.restore();
		return true;
	}

	// 6. Crystals (*), Altars (╥), Braziers (Ψ), Statues (Ω), Doors (+), Stairs (<, >), Portals (Ø)
	if (g === '*' || g === '◊') {
		ctx.save();
		ctx.fillStyle = fg;
		ctx.beginPath();
		ctx.moveTo(cx, y + chh * 0.12);
		ctx.lineTo(x + cw * 0.78, cy);
		ctx.lineTo(cx, y + chh * 0.88);
		ctx.lineTo(x + cw * 0.22, cy);
		ctx.closePath();
		ctx.fill();
		ctx.restore();
		return true;
	}

	if (g === 'Ψ') {
		ctx.save();
		// Brazier bowl + flame
		ctx.fillStyle = palette[2] || '#999';
		ctx.fillRect(x + cw * 0.22, y + chh * 0.62, cw * 0.56, chh * 0.22);
		ctx.fillStyle = fg;
		ctx.beginPath();
		ctx.moveTo(cx, y + chh * 0.16);
		ctx.quadraticCurveTo(x + cw * 0.8, y + chh * 0.45, x + cw * 0.68, y + chh * 0.62);
		ctx.lineTo(x + cw * 0.32, y + chh * 0.62);
		ctx.quadraticCurveTo(x + cw * 0.2, y + chh * 0.45, cx, y + chh * 0.16);
		ctx.fill();
		ctx.restore();
		return true;
	}

	if (g === '╥') {
		ctx.save();
		ctx.fillStyle = fg;
		ctx.fillRect(x + cw * 0.14, y + chh * 0.32, cw * 0.72, chh * 0.14);
		ctx.fillRect(x + cw * 0.24, y + chh * 0.46, cw * 0.14, chh * 0.38);
		ctx.fillRect(x + cw * 0.62, y + chh * 0.46, cw * 0.14, chh * 0.38);
		ctx.fillRect(x + cw * 0.1, y + chh * 0.8, cw * 0.8, chh * 0.1);
		ctx.restore();
		return true;
	}

	// 7a. Who is really standing here (species, calling, role, body plan: engine/sprinfo.go)
	if (actor && (g === '@' || /[A-Za-zψ]/.test(g))) {
		if (actor.s || g === '@') drawPersonSprite(ctx, actor, fg, palette, bgIdx, x, y, cw, chh);
		else drawCreatureSprite2(ctx, actor, g, fg, palette, bgIdx, x, y, cw, chh);
		return true;
	}

	// 7. Character (@) & Creature (a-z, A-Z, ψ) Sprites
	if (g === '@') {
		drawCharacterSprite(ctx, fg, palette, fgIdx, bgIdx, h, x, y, cw, chh);
		return true;
	}

	const code = g.charCodeAt(0);
	if ((code >= 65 && code <= 90 && g !== 'X') || (code >= 97 && code <= 122) || g === 'ψ') {
		drawCreatureSprite(ctx, g, fg, palette, fgIdx, bgIdx, h, x, y, cw, chh);
		return true;
	}

	return false;
}

function drawCharacterSprite(
	ctx: CanvasRenderingContext2D,
	fg: string,
	palette: string[],
	fgIdx: number,
	bgIdx: number,
	h: number,
	x: number,
	y: number,
	cw: number,
	chh: number,
) {
	ctx.save();
	const cx = x + cw * 0.5;
	const isPlayer = fgIdx === 3 && bgIdx !== 22;

	// Companion or boss ring
	if (bgIdx === 21 || isPlayer) {
		ctx.strokeStyle = isPlayer ? '#cfc041' : palette[9] || '#55ffff';
		ctx.lineWidth = 1;
		ctx.beginPath();
		ctx.ellipse(cx, y + chh * 0.86, cw * 0.38, chh * 0.08, 0, 0, Math.PI * 2);
		ctx.stroke();
	}

	// Cloak / torso
	ctx.fillStyle = fg;
	ctx.beginPath();
	ctx.moveTo(x + cw * 0.22, y + chh * 0.84);
	ctx.lineTo(x + cw * 0.28, y + chh * 0.42);
	ctx.lineTo(x + cw * 0.72, y + chh * 0.42);
	ctx.lineTo(x + cw * 0.78, y + chh * 0.84);
	ctx.closePath();
	ctx.fill();

	// Head
	ctx.beginPath();
	ctx.arc(cx, y + chh * 0.27, cw * 0.23, 0, Math.PI * 2);
	ctx.fill();

	// Eyes
	ctx.fillStyle = '#041312';
	ctx.fillRect(cx - cw * 0.1, y + chh * 0.24, Math.max(1, cw * 0.07), Math.max(1, cw * 0.07));
	ctx.fillRect(cx + cw * 0.04, y + chh * 0.24, Math.max(1, cw * 0.07), Math.max(1, cw * 0.07));

	// Procedural hat / helm / crown / hood + held item
	const gear = isPlayer ? 0 : (h + fgIdx) % 4;
	ctx.strokeStyle = palette[15] || '#ffff55';
	ctx.lineWidth = Math.max(1, cw * 0.1);
	if (gear === 0) {
		// Heroic plume / circlet & blade
		ctx.beginPath();
		ctx.moveTo(x + cw * 0.82, y + chh * 0.72);
		ctx.lineTo(x + cw * 0.88, y + chh * 0.22);
		ctx.stroke();
	} else if (gear === 1) {
		// Staff / spear
		ctx.beginPath();
		ctx.moveTo(x + cw * 0.16, y + chh * 0.86);
		ctx.lineTo(x + cw * 0.16, y + chh * 0.14);
		ctx.stroke();
	} else if (gear === 2) {
		// Wide-brimmed traveller hat
		ctx.fillStyle = fg;
		ctx.fillRect(x + cw * 0.16, y + chh * 0.12, cw * 0.68, chh * 0.07);
	}
	ctx.restore();
}

function drawCreatureSprite(
	ctx: CanvasRenderingContext2D,
	g: string,
	fg: string,
	palette: string[],
	fgIdx: number,
	bgIdx: number,
	h: number,
	x: number,
	y: number,
	cw: number,
	chh: number,
) {
	ctx.save();
	const cx = x + cw * 0.5;
	const cy = y + chh * 0.54;
	const isUpper = g >= 'A' && g <= 'Z';

	ctx.fillStyle = fg;
	ctx.strokeStyle = fg;

	if ('vPYwy'.includes(g)) {
		// Winged / avian creature
		ctx.beginPath();
		ctx.moveTo(cx, y + chh * 0.28);
		ctx.lineTo(x + cw * 0.9, y + chh * 0.18);
		ctx.lineTo(x + cw * 0.72, y + chh * 0.62);
		ctx.lineTo(cx, y + chh * 0.78);
		ctx.lineTo(x + cw * 0.28, y + chh * 0.62);
		ctx.lineTo(x + cw * 0.1, y + chh * 0.18);
		ctx.closePath();
		ctx.fill();
	} else if ('bsaISj'.includes(g)) {
		// Insectoid / arachnid
		ctx.beginPath();
		ctx.ellipse(cx, cy, cw * 0.26, chh * 0.24, 0, 0, Math.PI * 2);
		ctx.fill();
		ctx.lineWidth = Math.max(1, cw * 0.09);
		for (let k = -1; k <= 1; k++) {
			const ly = cy + k * chh * 0.12;
			ctx.beginPath();
			ctx.moveTo(x + cw * 0.1, ly - chh * 0.06);
			ctx.lineTo(x + cw * 0.9, ly + chh * 0.06);
			ctx.stroke();
		}
	} else if ('ψJXO'.includes(g)) {
		// Tentacled / eldritch / ooze
		ctx.beginPath();
		ctx.arc(cx, y + chh * 0.38, cw * 0.32, Math.PI, 0);
		ctx.lineTo(x + cw * 0.82, y + chh * 0.78);
		ctx.lineTo(x + cw * 0.62, y + chh * 0.62);
		ctx.lineTo(cx, y + chh * 0.82);
		ctx.lineTo(x + cw * 0.38, y + chh * 0.62);
		ctx.lineTo(x + cw * 0.18, y + chh * 0.78);
		ctx.closePath();
		ctx.fill();
	} else {
		// Beast / humanoid monster silhouette
		const r = isUpper ? cw * 0.38 : cw * 0.3;
		ctx.beginPath();
		ctx.arc(cx, cy - chh * 0.06, r, 0, Math.PI * 2);
		ctx.fill();
		// Legs
		ctx.fillRect(cx - r * 0.75, cy + chh * 0.08, cw * 0.14, chh * 0.26);
		ctx.fillRect(cx + r * 0.35, cy + chh * 0.08, cw * 0.14, chh * 0.26);
		// Horns / ears for upper-case or beasts
		if (isUpper || (h & 1) === 0) {
			ctx.beginPath();
			ctx.moveTo(cx - r * 0.6, cy - chh * 0.2);
			ctx.lineTo(cx - r * 0.95, y + chh * 0.14);
			ctx.lineTo(cx - r * 0.2, cy - chh * 0.24);
			ctx.moveTo(cx + r * 0.6, cy - chh * 0.2);
			ctx.lineTo(cx + r * 0.95, y + chh * 0.14);
			ctx.lineTo(cx + r * 0.2, cy - chh * 0.24);
			ctx.fill();
		}
	}

	// Glowing eyes
	ctx.fillStyle = bgIdx === 22 ? '#ffff55' : '#041312';
	ctx.fillRect(cx - cw * 0.12, cy - chh * 0.1, Math.max(1.2, cw * 0.08), Math.max(1.2, cw * 0.08));
	ctx.fillRect(cx + cw * 0.05, cy - chh * 0.1, Math.max(1.2, cw * 0.08), Math.max(1.2, cw * 0.08));

	// Draw small glyph badge in bottom-right when cell is large enough so roguelike letter readability stays 100% intact
	if (chh >= 14) {
		ctx.fillStyle = palette[3] || '#ffffff';
		ctx.font = `${Math.max(8, Math.round(chh * 0.45))}px "Share Tech Mono", monospace`;
		ctx.textAlign = 'right';
		ctx.textBaseline = 'bottom';
		ctx.fillText(g, x + cw - 0.5, y + chh);
	}
	ctx.restore();
}
