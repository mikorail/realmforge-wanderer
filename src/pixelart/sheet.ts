// Dev page /sheet.html: every pixel sprite at 8x so the art can be judged by eye. No game needed.
import { PW, PH, type Rows } from './types';
import { terrainSprite } from './terrain';
import { personSprite, SAMPLE_PEOPLE } from './people';
import { creatureSprite, SAMPLE_CREATURES } from './creatures';
import { blitSprite } from './render';

const root = document.getElementById('o')!;
const Z = 8;
function cell(rows: Rows, label: string, fg = '#c8b27a', bg = '#0e2220', pal: Record<string, string> = {}) {
	const wrap = document.createElement('div');
	wrap.style.cssText = 'display:inline-block;margin:3px;text-align:center;width:' + (PW * Z + 4) + 'px';
	const cv = document.createElement('canvas');
	cv.width = PW * Z;
	cv.height = PH * Z;
	const c = cv.getContext('2d')!;
	c.fillStyle = bg;
	c.fillRect(0, 0, cv.width, cv.height);
	c.setTransform(Z, 0, 0, Z, 0, 0);
	blitSprite(c, rows, 0, 0, PW, PH, { fg, bg, ...pal });
	wrap.append(cv, Object.assign(document.createElement('div'), { textContent: label }));
	root.append(wrap);
}
const q = new URLSearchParams(location.search);
const section = (t: string) => root.append(Object.assign(document.createElement('h3'), { textContent: t }));
const COLS = ['#c8b27a', '#8fb36b', '#7aa0c8', '#b86a5a', '#a0a0a0', '#d8d8c0', '#6a8a50', '#c8903a'];
if (q.get('only') !== 'people' && q.get('only') !== 'creatures') {
	section('terrain');
	const gl = (q.get('glyphs') ?? '#.,"~≈♣♠τ▲▒╥║∴ⁿ°φΓ‼+/\\|-_=:;*%^<>[]{}()$&!?\'`@Ω¤§∏⊞≡╫δΘ∩↑◊ψΨ·∙O').split('');
	for (const g of gl) for (let k = 0; k < COLS.length; k += 3) {
		const r = terrainSprite(g, 0, 0, 12345 + k * 777, 0, 255);
		if (r) cell(r, g, COLS[(k + g.charCodeAt(0)) % COLS.length]);
	}
}
if (q.get('only') !== 'terrain' && q.get('only') !== 'creatures') {
	section('people');
	for (const s of SAMPLE_PEOPLE) {
		const pose = { phase: 0, attack: 0, cast: 0, hurt: 0, block: 0, face: 1, ...s.pose } as never;
		for (let f = 0; f < 2; f++) {
			const r = personSprite(s.actor, 777, pose, f * 4);
			if (r) cell(r.rows, s.label + (f ? ' f1' : ''), '#c8b27a', '#0e2220', r.pal);
		}
	}
}
if (q.get('only') !== 'terrain' && q.get('only') !== 'people') {
	section('creatures');
	for (const s of SAMPLE_CREATURES) for (let f = 0; f < 2; f++) {
		const r = creatureSprite(s.actor, s.glyph, 777, f * 4);
		if (r) cell(r.rows, s.label + (f ? ' f1' : ''), '#c8b27a', '#0e2220', r.pal);
	}
}
