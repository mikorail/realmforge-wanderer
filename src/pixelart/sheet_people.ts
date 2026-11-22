// Dev page /sheet-people.html: people sprites only, at 6x.
import { PW, PH } from './types';
import { personSprite, SAMPLE_PEOPLE } from './people';
import { blitSprite } from './render';
const root = document.getElementById('o')!;
const Z = 6;
const q = new URLSearchParams(location.search);
const from = +(q.get('from') ?? 0), n = +(q.get('n') ?? 60);
for (const s of SAMPLE_PEOPLE.slice(from, from + n)) {
	const pose = { phase: 0, attack: 0, cast: 0, hurt: 0, block: 0, face: 1, ...s.pose } as never;
	const r = personSprite(s.actor, 777, pose, 0);
	if (!r) continue;
	const wrap = document.createElement('div');
	wrap.style.cssText = 'display:inline-block;margin:2px;text-align:center;width:' + (PW * Z + 6) + 'px';
	const cv = document.createElement('canvas');
	cv.width = PW * Z;
	cv.height = PH * Z;
	const c = cv.getContext('2d')!;
	c.fillStyle = '#0e2220';
	c.fillRect(0, 0, cv.width, cv.height);
	c.setTransform(Z, 0, 0, Z, 0, 0);
	blitSprite(c, r.rows, 0, 0, PW, PH, { fg: '#c8b27a', bg: '#0e2220', ...r.pal });
	wrap.append(cv, Object.assign(document.createElement('div'), { textContent: s.label.slice(0, 12) }));
	root.append(wrap);
}
