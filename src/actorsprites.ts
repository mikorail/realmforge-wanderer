// Sprite-mode portraits for actors: every species has its own build, skin and features, every
// calling and role its own outfit and gear, and each individual varies (hair, cloth, build, tint)
// by their id. The engine sends who stands where (`frame.actors`, engine/sprinfo.go); the glyph
// grid still decides what is visible, so nothing here changes the game.

export interface QudActor {
	x: number;
	y: number;
	i: number; // actor id: the seed of this one's variation
	s?: string; // species id
	c?: string; // the player's calling
	r?: string; // role (guard, merchant, elder …)
	t?: string; // creature template
	b?: string; // creature body plan
	gr?: string; // bestiary group (mam bird rept dino fish ins …)
	z?: number; // bestiary size 1..7
	p?: boolean; // the player
	w?: string; // wielded weapon skill
	a?: number; // armour 0 none .. 3 heavy
	k?: string; // caste
}

type Head = 'round' | 'long' | 'wide' | 'skull';
type Ears = 'none' | 'point' | 'round' | 'rat' | 'long';
type Tail = 'none' | 'rat' | 'lizard' | 'fish';
type Horns = 'none' | 'small' | 'big' | 'ram';
interface Look {
	h: number; // height 0.6..1
	w: number; // width multiplier
	hd: number; // head size multiplier
	skin: string[]; // tones: one per individual
	hair?: string[]; // hair colours (none = bald by species)
	head: Head;
	ears: Ears;
	snout: number; // 0..1
	tail: Tail;
	horns: Horns;
	tusks?: boolean;
	beard?: boolean;
	arms?: 2 | 4;
	wings?: boolean;
	halo?: boolean;
	scales?: boolean;
	fur?: boolean;
	eye: string;
	fish?: boolean; // fish tail instead of legs
	swarm?: boolean; // a drifting cluster of small bodies
	bone?: boolean; // exposed bone, tatters
	glow?: string; // aura colour
}

const SKIN_HUMAN = ['#f0c7a0', '#d9a577', '#b57d54', '#8a5a3b', '#5d3b27', '#f5d9bd'];
const HAIR = ['#2a1b12', '#5a3a1e', '#8a6a2a', '#c9a24a', '#a8441d', '#d8d0c0', '#101010'];

const SPECIES: Record<string, Look> = {
	human: { h: 0.92, w: 1, hd: 1, skin: SKIN_HUMAN, hair: HAIR, head: 'round', ears: 'round', snout: 0, tail: 'none', horns: 'none', eye: '#1a1410' },
	dwarf: { h: 0.74, w: 1.32, hd: 1.05, skin: ['#e0a98a', '#c98a68', '#b5744f'], hair: ['#8a3a18', '#5a2a12', '#c9c0b0', '#d9a441'], head: 'wide', ears: 'round', snout: 0, tail: 'none', horns: 'none', beard: true, eye: '#1a1410' },
	elf: { h: 1, w: 0.78, hd: 0.95, skin: ['#f2dcc4', '#e8cfae', '#d9c2a0'], hair: ['#e8dcc0', '#c9a24a', '#2a2018', '#9aa7b8'], head: 'long', ears: 'point', snout: 0, tail: 'none', horns: 'none', eye: '#3a6a3a' },
	void_elf: { h: 1, w: 0.76, hd: 0.95, skin: ['#8a86a8', '#6c6890', '#a49cc0'], hair: ['#e8e4f4', '#2a2438', '#b8a8e8'], head: 'long', ears: 'point', snout: 0, tail: 'none', horns: 'none', eye: '#e8a8ff', glow: '#8a4fc8' },
	halfling: { h: 0.6, w: 1.1, hd: 1.1, skin: ['#e8bd94', '#d4a279', '#c28e66'], hair: ['#6a4020', '#8a5a2a', '#3a2412'], head: 'round', ears: 'point', snout: 0, tail: 'none', horns: 'none', eye: '#1a1410' },
	orc: { h: 1, w: 1.35, hd: 1, skin: ['#6a8a3a', '#7a9a48', '#58762e'], hair: ['#181410', '#2a1b12'], head: 'wide', ears: 'point', snout: 0.25, tail: 'none', horns: 'none', tusks: true, eye: '#e8d040' },
	goblin: { h: 0.62, w: 0.92, hd: 1.2, skin: ['#8aa04a', '#9ab058', '#788c3a'], head: 'long', ears: 'long', snout: 0.35, tail: 'none', horns: 'none', eye: '#ffd83a' },
	ogor: { h: 1, w: 1.55, hd: 0.9, skin: ['#9a8478', '#8a7468', '#a89488'], head: 'wide', ears: 'round', snout: 0.3, tail: 'none', horns: 'small', tusks: true, eye: '#e8c040' },
	troll: { h: 1, w: 1.4, hd: 0.95, skin: ['#5a7a58', '#4a6a4c', '#6a8a60'], head: 'long', ears: 'long', snout: 0.5, tail: 'none', horns: 'none', tusks: true, eye: '#e8e060' },
	giant: { h: 1, w: 1.5, hd: 0.8, skin: ['#9aa8b8', '#8a98a8', '#aab4c0'], hair: ['#c9c0b0', '#6a6258'], head: 'wide', ears: 'round', snout: 0, tail: 'none', horns: 'none', beard: true, eye: '#2a3a5a' },
	ratfolk: { h: 0.74, w: 0.9, hd: 0.95, skin: ['#8a7a6a', '#6a5a4a', '#a89888'], head: 'long', ears: 'rat', snout: 0.8, tail: 'rat', horns: 'none', fur: true, eye: '#d84a4a' },
	beastman: { h: 0.96, w: 1.2, hd: 1, skin: ['#8a5a30', '#6a4020', '#a87a48'], head: 'long', ears: 'long', snout: 0.55, tail: 'none', horns: 'ram', fur: true, eye: '#e8c040' },
	lizardfolk: { h: 0.9, w: 1, hd: 0.95, skin: ['#4a8a58', '#5a9a68', '#6a8a40'], head: 'long', ears: 'none', snout: 0.7, tail: 'lizard', horns: 'none', scales: true, eye: '#f0d040' },
	saurian: { h: 1, w: 1.15, hd: 1, skin: ['#b8482a', '#c8803a', '#a8603a'], head: 'long', ears: 'none', snout: 0.55, tail: 'lizard', horns: 'big', scales: true, eye: '#ffd24a' },
	merfolk: { h: 0.9, w: 0.95, hd: 1, skin: ['#5aa8b8', '#4a98a8', '#7ab8c0'], hair: ['#2a6a8a', '#3aa090', '#8ad0c0'], head: 'round', ears: 'long', snout: 0, tail: 'fish', horns: 'none', scales: true, fish: true, eye: '#1a3a5a' },
	celestial: { h: 1, w: 0.9, hd: 0.95, skin: ['#f0dca0', '#f8e8b8', '#e8cc88'], hair: ['#fff4c8', '#f0d890'], head: 'round', ears: 'none', snout: 0, tail: 'none', horns: 'none', wings: true, halo: true, eye: '#ffffff', glow: '#fff0a0' },
	tomb_undead: { h: 0.95, w: 0.8, hd: 1, skin: ['#d8d0b8', '#c8c0a8'], head: 'skull', ears: 'none', snout: 0, tail: 'none', horns: 'none', bone: true, eye: '#58ff9a', glow: '#3aaa6a' },
	warp_fiend: { h: 1, w: 1.05, hd: 1, skin: ['#b0328a', '#8a2a9a', '#c84a7a'], head: 'wide', ears: 'point', snout: 0.2, tail: 'lizard', horns: 'big', arms: 4, eye: '#ffffff', glow: '#d83aff' },
	chitin_swarm: { h: 0.8, w: 1.2, hd: 1, skin: ['#2a2218', '#3a2a1c', '#1c1c24'], head: 'round', ears: 'none', snout: 0, tail: 'none', horns: 'none', swarm: true, eye: '#ff4a4a' },
};

function hash(n: number, salt = 0): number {
	let h = (Math.imul(n + 0x9e3779b1, 0x85ebca6b) ^ Math.imul(salt + 1, 0xc2b2ae35)) >>> 0;
	h = Math.imul(h ^ (h >>> 15), 0x2c1b3c6d) >>> 0;
	h = Math.imul(h ^ (h >>> 12), 0x297a2d39) >>> 0;
	return (h ^ (h >>> 15)) >>> 0;
}
const pick = <T,>(a: T[], h: number): T => a[h % a.length];
function strHash(s: string): number {
	let h = 2166136261;
	for (let i = 0; i < s.length; i++) h = Math.imul(h ^ s.charCodeAt(i), 16777619);
	return h >>> 0;
}

/** an unknown species still gets its own stable face: features picked from its id */
function lookFor(sp: string): Look {
	const known = SPECIES[sp];
	if (known) return known;
	const h = strHash(sp);
	const hue = h % 360;
	const base = `hsl(${hue} 38% 52%)`;
	return {
		h: 0.7 + (h % 30) / 100,
		w: 0.85 + ((h >>> 5) % 50) / 100,
		hd: 0.9 + ((h >>> 9) % 25) / 100,
		skin: [base, `hsl(${hue} 34% 44%)`, `hsl(${hue} 42% 60%)`],
		hair: (h >>> 3) & 1 ? HAIR : undefined,
		head: (['round', 'long', 'wide'] as Head[])[(h >>> 11) % 3],
		ears: (['none', 'point', 'round', 'long'] as Ears[])[(h >>> 13) % 4],
		snout: ((h >>> 15) % 4) * 0.2,
		tail: (['none', 'none', 'rat', 'lizard'] as Tail[])[(h >>> 17) % 4],
		horns: (['none', 'none', 'small', 'big'] as Horns[])[(h >>> 19) % 4],
		tusks: ((h >>> 21) & 3) === 0,
		beard: ((h >>> 22) & 3) === 0,
		eye: '#1a1410',
	};
}

// ---------------------------------------------------------------- outfits

interface Outfit {
	cloth: string[]; // body colours
	trim: string;
	head: 'none' | 'helm' | 'hood' | 'cap' | 'hat' | 'goggles' | 'mask' | 'crown' | 'turban' | 'circlet' | 'bandana';
	back: 'none' | 'cloak' | 'pack' | 'quiver' | 'cape';
	item: 'none' | 'sword' | 'dagger' | 'axe' | 'club' | 'bow' | 'spear' | 'staff' | 'wand' | 'book' | 'hammer' | 'purse' | 'wrench' | 'orb' | 'lantern' | 'knives';
	shield?: boolean;
	armor?: number; // 0..3 when not given by the engine
	fine?: boolean; // richer cloth
}

const CALLINGS: Record<string, Outfit> = {
	warrior: { cloth: ['#7a2a2a', '#6a3a2a'], trim: '#c9ccd2', head: 'helm', back: 'cape', item: 'sword', shield: true, armor: 2 },
	scout: { cloth: ['#4a6a3a', '#5a6a40'], trim: '#8a6a3a', head: 'hood', back: 'quiver', item: 'bow', armor: 1 },
	scholar: { cloth: ['#3a4a8a', '#4a3a7a'], trim: '#e8d890', head: 'cap', back: 'none', item: 'book', armor: 0 },
	tinker: { cloth: ['#8a5a2a', '#6a6a70'], trim: '#e0a030', head: 'goggles', back: 'pack', item: 'wrench', armor: 1 },
	mystic: { cloth: ['#5a3a8a', '#2a6a7a'], trim: '#f0e0a0', head: 'hood', back: 'none', item: 'staff', armor: 0 },
	merchant: { cloth: ['#8a2a4a', '#9a6a1a'], trim: '#f0c850', head: 'hat', back: 'none', item: 'purse', armor: 0, fine: true },
	rogue: { cloth: ['#2a2a32', '#3a2a3a'], trim: '#a83a3a', head: 'mask', back: 'cloak', item: 'knives', armor: 1 },
};

const ROLES: Record<string, Outfit> = {
	guard: { cloth: ['#4a5a7a', '#5a4a3a'], trim: '#c9ccd2', head: 'helm', back: 'none', item: 'spear', shield: true, armor: 2 },
	soldier: { cloth: ['#6a3a3a', '#4a5a4a'], trim: '#c9ccd2', head: 'helm', back: 'none', item: 'sword', armor: 2 },
	merchant: { cloth: ['#8a2a4a', '#9a6a1a', '#2a6a5a'], trim: '#f0c850', head: 'hat', back: 'none', item: 'purse', fine: true },
	smith: { cloth: ['#5a4a3a', '#4a3a30'], trim: '#c07030', head: 'none', back: 'none', item: 'hammer', armor: 1 },
	priest: { cloth: ['#e8e4d8', '#d8d0b0'], trim: '#e8c850', head: 'circlet', back: 'none', item: 'staff', fine: true },
	elder: { cloth: ['#8a8a98', '#6a5a7a'], trim: '#d8d0c0', head: 'none', back: 'none', item: 'staff' },
	scholar: { cloth: ['#3a4a8a', '#4a3a7a'], trim: '#e8d890', head: 'cap', back: 'none', item: 'book' },
	apothecary: { cloth: ['#3a6a4a', '#5a7a3a'], trim: '#d8d0a0', head: 'cap', back: 'none', item: 'lantern' },
	bandit: { cloth: ['#4a3a2a', '#3a3a3a'], trim: '#8a3a2a', head: 'bandana', back: 'none', item: 'dagger', armor: 1 },
	notable: { cloth: ['#7a2a5a', '#2a4a7a', '#8a6a1a'], trim: '#f0d060', head: 'circlet', back: 'cape', item: 'none', fine: true },
	noble: { cloth: ['#7a2a5a', '#2a4a7a', '#8a6a1a'], trim: '#f0d060', head: 'crown', back: 'cape', item: 'none', fine: true },
	villager: { cloth: ['#8a6a4a', '#6a7a4a', '#7a5a6a', '#5a6a8a', '#8a7a3a', '#6a5a4a'], trim: '#b8a078', head: 'none', back: 'none', item: 'none' },
	farmer: { cloth: ['#8a7a3a', '#6a7a4a'], trim: '#c8b060', head: 'hat', back: 'none', item: 'none' },
	sailor: { cloth: ['#3a5a8a', '#e0e0e8'], trim: '#c9ccd2', head: 'bandana', back: 'none', item: 'none' },
};

const WEAPON_ITEM: Record<string, Outfit['item']> = {
	longblade: 'sword', shortblade: 'dagger', axe: 'axe', cudgel: 'club', missile: 'bow', spear: 'spear', staff: 'staff', flail: 'club',
	whip: 'wand', fist: 'none', hook: 'club', thrown: 'knives', firearm: 'wrench', heavy: 'hammer',
};

function shade(col: string, amt: number): string {
	if (col[0] !== '#' || col.length !== 7) return col;
	const n = parseInt(col.slice(1), 16);
	const f = (v: number) => Math.max(0, Math.min(255, Math.round(v + amt * (amt > 0 ? 255 - v : v))));
	return `#${[(n >> 16) & 255, (n >> 8) & 255, n & 255].map((v) => f(v).toString(16).padStart(2, '0')).join('')}`;
}

function outfitFor(a: QudActor, h: number): Outfit {
	let o: Outfit | undefined;
	if (a.p && a.c) o = CALLINGS[a.c];
	if (!o && a.r) o = ROLES[a.r];
	if (!o) o = ROLES.villager;
	const out = { ...o };
	if (a.w && a.p) {
		const it = WEAPON_ITEM[a.w];
		if (it) out.item = it;
		if (a.w === 'missile') out.back = 'quiver';
	}
	if (a.a !== undefined && a.a >= 0) out.armor = a.a;
	if (a.k && /high|noble|lord|elite/i.test(a.k)) out.fine = true;
	void h;
	return out;
}

// ---------------------------------------------------------------- drawing helpers

type Ctx = CanvasRenderingContext2D;
interface Box {
	x: number;
	y: number;
	w: number;
	h: number;
}
const ell = (c: Ctx, cx: number, cy: number, rx: number, ry: number) => {
	c.beginPath();
	c.ellipse(cx, cy, Math.max(0.3, rx), Math.max(0.3, ry), 0, 0, Math.PI * 2);
	c.fill();
};
const poly = (c: Ctx, pts: number[], close = true) => {
	c.beginPath();
	c.moveTo(pts[0], pts[1]);
	for (let i = 2; i < pts.length; i += 2) c.lineTo(pts[i], pts[i + 1]);
	if (close) c.closePath();
	c.fill();
};
const line = (c: Ctx, x0: number, y0: number, x1: number, y1: number, w: number) => {
	c.lineWidth = Math.max(1, w);
	c.beginPath();
	c.moveTo(x0, y0);
	c.lineTo(x1, y1);
	c.stroke();
};

function drawItem(c: Ctx, item: Outfit['item'], hx: number, hy: number, u: number, v: number, trim: string, steel: string, flip = 1) {
	c.strokeStyle = steel;
	c.fillStyle = steel;
	const lw = Math.max(1, u * 0.07);
	switch (item) {
		case 'sword':
			line(c, hx, hy, hx + flip * u * 0.06, hy - v * 0.42, lw * 1.2);
			c.strokeStyle = trim;
			line(c, hx - u * 0.1, hy - v * 0.04, hx + u * 0.1, hy - v * 0.04, lw);
			break;
		case 'dagger':
			line(c, hx, hy, hx + flip * u * 0.03, hy - v * 0.22, lw);
			break;
		case 'knives':
			line(c, hx, hy, hx + flip * u * 0.05, hy - v * 0.2, lw);
			line(c, hx - flip * u * 0.55, hy, hx - flip * u * 0.6, hy - v * 0.2, lw);
			break;
		case 'axe':
			c.strokeStyle = '#7a5a3a';
			line(c, hx, hy + v * 0.04, hx, hy - v * 0.4, lw);
			c.fillStyle = steel;
			poly(c, [hx, hy - v * 0.4, hx + flip * u * 0.22, hy - v * 0.34, hx + flip * u * 0.2, hy - v * 0.2, hx, hy - v * 0.24]);
			break;
		case 'club':
			c.strokeStyle = '#7a5a3a';
			line(c, hx, hy, hx + flip * u * 0.03, hy - v * 0.32, lw * 1.6);
			break;
		case 'hammer':
			c.strokeStyle = '#7a5a3a';
			line(c, hx, hy + v * 0.04, hx, hy - v * 0.34, lw);
			c.fillStyle = steel;
			c.fillRect(hx - u * 0.12, hy - v * 0.4, u * 0.24, v * 0.1);
			break;
		case 'bow':
			c.strokeStyle = '#8a6a3a';
			c.lineWidth = lw;
			c.beginPath();
			c.arc(hx + flip * u * 0.1, hy - v * 0.14, v * 0.26, -Math.PI / 2 - 0.2, Math.PI / 2 + 0.2, flip < 0);
			c.stroke();
			break;
		case 'spear':
			c.strokeStyle = '#8a6a3a';
			line(c, hx, hy + v * 0.06, hx, hy - v * 0.55, lw);
			c.fillStyle = steel;
			poly(c, [hx, hy - v * 0.66, hx - u * 0.06, hy - v * 0.52, hx + u * 0.06, hy - v * 0.52]);
			break;
		case 'staff':
			c.strokeStyle = '#8a6a3a';
			line(c, hx, hy + v * 0.06, hx, hy - v * 0.6, lw);
			c.fillStyle = trim;
			ell(c, hx, hy - v * 0.62, u * 0.07, u * 0.07);
			break;
		case 'wand':
			line(c, hx, hy, hx + flip * u * 0.12, hy - v * 0.2, lw);
			break;
		case 'orb':
			c.fillStyle = trim;
			ell(c, hx, hy - v * 0.06, u * 0.1, u * 0.1);
			break;
		case 'book':
			c.fillStyle = '#7a3a2a';
			c.fillRect(hx - u * 0.09, hy - v * 0.12, u * 0.18, v * 0.14);
			c.fillStyle = '#e8e0c0';
			c.fillRect(hx - u * 0.07, hy - v * 0.1, u * 0.14, v * 0.03);
			break;
		case 'purse':
			c.fillStyle = '#6a4a2a';
			ell(c, hx, hy, u * 0.09, v * 0.06);
			c.fillStyle = trim;
			ell(c, hx, hy - v * 0.06, u * 0.04, v * 0.02);
			break;
		case 'wrench':
			line(c, hx, hy, hx + flip * u * 0.04, hy - v * 0.3, lw * 1.2);
			c.fillStyle = trim;
			ell(c, hx + flip * u * 0.04, hy - v * 0.32, u * 0.08, u * 0.06);
			break;
		case 'lantern':
			c.fillStyle = '#ffd24a';
			ell(c, hx, hy, u * 0.07, v * 0.07);
			break;
		default:
			break;
	}
}

/** a person: species body plan, calling/role outfit, and per-individual variation */
export function drawPersonSprite(c: Ctx, a: QudActor, fg: string, palette: string[], bgIdx: number, x: number, y: number, cw: number, chh: number) {
	const seed = hash(a.i + 17);
	const sp = a.s ?? 'human';
	const L = lookFor(sp);
	const o = outfitFor(a, seed);
	const skin = pick(L.skin, seed >>> 3);
	const hair = L.hair ? pick(L.hair, seed >>> 7) : undefined;
	const cloth = shade(pick(o.cloth, seed >>> 11), ((seed >>> 16) % 7 - 3) * 0.04);
	const dark = shade(cloth, -0.35);
	const steel = '#c9ccd2';
	const u = cw;
	const v = chh;
	const vary = 1 + (((seed >>> 20) % 9) - 4) * 0.015;
	const cx = x + u * 0.5;
	const base = y + v * 0.9;
	const H = v * 0.82 * L.h * vary;
	const W = u * 0.46 * L.w;
	const hr = u * 0.2 * L.hd * (1 + (L.h < 0.75 ? 0.15 : 0));
	const headY = base - H + hr;
	const neckY = headY + hr * 0.85;
	const hipY = base - H * 0.34;

	c.save();
	// rings: the player and companions are marked as before
	const ring = a.p ? '#cfc041' : bgIdx === 21 ? palette[9] || '#55ffff' : null;
	if (ring) {
		c.strokeStyle = ring;
		c.lineWidth = 1;
		c.beginPath();
		c.ellipse(cx, y + v * 0.92, u * 0.4, v * 0.06, 0, 0, Math.PI * 2);
		c.stroke();
	}
	if (L.glow) {
		const gr = c.createRadialGradient(cx, y + v * 0.5, 1, cx, y + v * 0.5, v * 0.55);
		gr.addColorStop(0, L.glow + '66');
		gr.addColorStop(1, L.glow + '00');
		c.fillStyle = gr;
		c.fillRect(x, y, u, v);
	}

	if (L.swarm) {
		// a chitin swarm: a cluster of scuttling bodies
		for (let k = 0; k < 9; k++) {
			const hh = hash(a.i, k + 3);
			const bx = cx + (((hh % 100) / 100) - 0.5) * u * 0.8;
			const by = y + v * 0.45 + (((hh >>> 8) % 100) / 100) * v * 0.42;
			c.fillStyle = k % 3 === 0 ? shade(skin, 0.25) : skin;
			ell(c, bx, by, u * 0.1, v * 0.045);
			c.strokeStyle = skin;
			line(c, bx - u * 0.1, by + v * 0.04, bx + u * 0.1, by - v * 0.03, 1);
		}
		c.fillStyle = L.eye;
		ell(c, cx, y + v * 0.5, u * 0.04, u * 0.04);
		c.restore();
		return;
	}

	// wings and tail go behind the body
	if (L.wings) {
		c.fillStyle = '#f4ecd0';
		poly(c, [cx - W * 0.4, neckY + v * 0.04, cx - u * 0.52, headY - hr * 0.6, cx - u * 0.38, hipY, cx - W * 0.3, hipY - v * 0.04]);
		poly(c, [cx + W * 0.4, neckY + v * 0.04, cx + u * 0.52, headY - hr * 0.6, cx + u * 0.38, hipY, cx + W * 0.3, hipY - v * 0.04]);
	}
	if (o.back === 'cape' || o.back === 'cloak') {
		c.fillStyle = shade(dark, -0.1);
		poly(c, [cx - W * 0.55, neckY, cx + W * 0.55, neckY, cx + W * 0.7, base - v * 0.04, cx - W * 0.7, base - v * 0.04]);
	}
	if (L.tail !== 'none') {
		c.strokeStyle = skin;
		c.lineWidth = Math.max(1, u * (L.tail === 'rat' ? 0.05 : 0.09));
		c.beginPath();
		const tx = cx + W * 0.4;
		c.moveTo(tx, hipY);
		c.quadraticCurveTo(tx + u * 0.3, hipY + v * 0.02, tx + u * 0.34, base - v * (L.tail === 'rat' ? 0.3 : 0.14));
		c.stroke();
	}

	// legs (or a fish tail)
	if (L.fish) {
		c.fillStyle = skin;
		poly(c, [cx - W * 0.4, hipY, cx + W * 0.4, hipY, cx + W * 0.2, base - v * 0.12, cx, base - v * 0.04, cx - W * 0.2, base - v * 0.12]);
		c.fillStyle = shade(skin, 0.2);
		poly(c, [cx - W * 0.2, base - v * 0.12, cx - u * 0.36, base, cx, base - v * 0.1, cx + u * 0.36, base, cx + W * 0.2, base - v * 0.12]);
	} else {
		c.fillStyle = L.bone ? skin : dark;
		const lw = Math.max(1.5, u * 0.14 * Math.min(1.3, L.w));
		c.fillRect(cx - W * 0.32 - lw / 2, hipY, lw, base - hipY);
		c.fillRect(cx + W * 0.32 - lw / 2, hipY, lw, base - hipY);
		c.fillStyle = shade(dark, -0.3);
		c.fillRect(cx - W * 0.32 - lw * 0.7, base - v * 0.04, lw * 1.4, v * 0.04);
		c.fillRect(cx + W * 0.32 - lw * 0.3, base - v * 0.04, lw * 1.4, v * 0.04);
	}

	// torso
	const armor = o.armor ?? 0;
	c.fillStyle = armor >= 3 ? '#8a909a' : armor === 2 ? shade(cloth, -0.1) : cloth;
	poly(c, [cx - W * 0.5, neckY, cx + W * 0.5, neckY, cx + W * 0.42, hipY + v * 0.04, cx - W * 0.42, hipY + v * 0.04]);
	if (armor >= 2) {
		c.fillStyle = armor >= 3 ? '#b8bec8' : '#9a8a6a';
		poly(c, [cx - W * 0.36, neckY + v * 0.02, cx + W * 0.36, neckY + v * 0.02, cx + W * 0.3, neckY + (hipY - neckY) * 0.62, cx - W * 0.3, neckY + (hipY - neckY) * 0.62]);
		c.fillStyle = armor >= 3 ? '#c9ccd2' : '#b0a07a';
		ell(c, cx - W * 0.52, neckY + v * 0.04, u * 0.09, v * 0.05);
		ell(c, cx + W * 0.52, neckY + v * 0.04, u * 0.09, v * 0.05);
	}
	if (L.scales || L.fur) {
		c.strokeStyle = shade(cloth, 0.2);
		c.lineWidth = 1;
	}
	// belt and trim
	c.fillStyle = o.trim;
	c.fillRect(cx - W * 0.44, hipY - v * 0.02, W * 0.88, Math.max(1, v * 0.03));
	if (o.fine) {
		c.fillRect(cx - W * 0.05, neckY, W * 0.1, hipY - neckY);
	}
	if (o.back === 'pack') {
		c.fillStyle = '#7a5a3a';
		c.fillRect(cx + W * 0.34, neckY + v * 0.04, u * 0.16, (hipY - neckY) * 0.8);
	}
	if (o.back === 'quiver') {
		c.fillStyle = '#6a4a2a';
		c.fillRect(cx - W * 0.56, neckY - v * 0.02, u * 0.1, (hipY - neckY) * 0.8);
		c.strokeStyle = '#e0d8c0';
		line(c, cx - W * 0.52, neckY - v * 0.02, cx - W * 0.5, neckY - v * 0.07, 1);
	}
	if (L.bone) {
		// ribs and tatters
		c.strokeStyle = shade(skin, -0.3);
		for (let k = 0; k < 3; k++) line(c, cx - W * 0.3, neckY + v * (0.05 + k * 0.045), cx + W * 0.3, neckY + v * (0.05 + k * 0.045), 1);
	}

	// arms (two, or four)
	c.strokeStyle = o.armor && o.armor >= 2 ? '#8a909a' : cloth;
	const aw = Math.max(1.4, u * 0.1);
	const armPairs = L.arms === 4 ? [0, 1] : [0];
	for (const k of armPairs) {
		const off = k * v * 0.1;
		line(c, cx - W * 0.5, neckY + v * 0.04 + off, cx - W * 0.68, hipY - v * 0.0 + off * 0.5, aw);
		line(c, cx + W * 0.5, neckY + v * 0.04 + off, cx + W * 0.66, hipY - v * 0.02 + off * 0.5, aw);
	}
	c.fillStyle = skin;
	ell(c, cx - W * 0.68, hipY, u * 0.05, u * 0.05);
	ell(c, cx + W * 0.66, hipY - v * 0.02, u * 0.05, u * 0.05);
	if (o.shield) {
		c.fillStyle = shade(o.cloth[0], -0.1);
		ell(c, cx - W * 0.78, hipY - v * 0.1, u * 0.14, v * 0.12);
		c.fillStyle = o.trim;
		ell(c, cx - W * 0.78, hipY - v * 0.1, u * 0.05, v * 0.04);
	}
	drawItem(c, o.item, cx + W * 0.66, hipY - v * 0.02, u, v, o.trim, steel, 1);

	// head
	c.fillStyle = skin;
	const hw = hr * (L.head === 'wide' ? 1.2 : L.head === 'long' ? 0.9 : 1);
	const hh2 = hr * (L.head === 'long' ? 1.2 : L.head === 'wide' ? 0.95 : 1);
	if (L.head === 'skull') {
		ell(c, cx, headY, hw, hh2);
		c.fillStyle = shade(skin, -0.2);
		c.fillRect(cx - hw * 0.5, headY + hh2 * 0.55, hw, hh2 * 0.5);
	} else {
		ell(c, cx, headY, hw, hh2);
	}
	// ears
	c.fillStyle = skin;
	const ey = headY - hh2 * 0.1;
	if (L.ears === 'point') {
		poly(c, [cx - hw * 0.9, ey, cx - hw * 1.9, ey - hh2 * 0.9, cx - hw * 0.7, ey - hh2 * 0.4]);
		poly(c, [cx + hw * 0.9, ey, cx + hw * 1.9, ey - hh2 * 0.9, cx + hw * 0.7, ey - hh2 * 0.4]);
	} else if (L.ears === 'long') {
		poly(c, [cx - hw * 0.8, ey, cx - hw * 2.1, ey - hh2 * 0.3, cx - hw * 0.7, ey - hh2 * 0.5]);
		poly(c, [cx + hw * 0.8, ey, cx + hw * 2.1, ey - hh2 * 0.3, cx + hw * 0.7, ey - hh2 * 0.5]);
	} else if (L.ears === 'rat') {
		ell(c, cx - hw * 0.8, ey - hh2 * 0.55, hw * 0.5, hw * 0.55);
		ell(c, cx + hw * 0.8, ey - hh2 * 0.55, hw * 0.5, hw * 0.55);
		c.fillStyle = '#d8a0a0';
		ell(c, cx - hw * 0.8, ey - hh2 * 0.55, hw * 0.25, hw * 0.3);
		ell(c, cx + hw * 0.8, ey - hh2 * 0.55, hw * 0.25, hw * 0.3);
		c.fillStyle = skin;
	} else if (L.ears === 'round') {
		ell(c, cx - hw * 0.95, ey, hw * 0.25, hw * 0.3);
		ell(c, cx + hw * 0.95, ey, hw * 0.25, hw * 0.3);
	}
	// snout
	if (L.snout > 0) {
		c.fillStyle = shade(skin, 0.08);
		ell(c, cx, headY + hh2 * 0.4, hw * (0.35 + L.snout * 0.35), hh2 * (0.3 + L.snout * 0.2));
		c.fillStyle = '#2a1a1a';
		ell(c, cx, headY + hh2 * (0.3 + L.snout * 0.15), hw * 0.1, hw * 0.08);
	}
	// hair, beard
	if (hair && L.head !== 'skull') {
		c.fillStyle = hair;
		const style = (seed >>> 24) % 4;
		c.beginPath();
		c.arc(cx, headY - hh2 * 0.1, hw * 1.02, Math.PI, 0);
		c.fill();
		if (style === 1 || style === 3) {
			c.fillRect(cx - hw * 1.02, headY - hh2 * 0.1, hw * 0.28, hh2 * (style === 3 ? 1.5 : 0.9));
			c.fillRect(cx + hw * 0.74, headY - hh2 * 0.1, hw * 0.28, hh2 * (style === 3 ? 1.5 : 0.9));
		}
		if (style === 2) ell(c, cx, headY - hh2 * 0.95, hw * 0.35, hw * 0.35);
	}
	if (L.beard) {
		c.fillStyle = hair ?? '#c9c0b0';
		poly(c, [cx - hw * 0.85, headY + hh2 * 0.15, cx + hw * 0.85, headY + hh2 * 0.15, cx + hw * 0.5, headY + hh2 * 1.9, cx, headY + hh2 * 2.1, cx - hw * 0.5, headY + hh2 * 1.9]);
	}
	if (a.r === 'elder' && !L.beard) {
		c.fillStyle = '#e8e4dc';
		poly(c, [cx - hw * 0.7, headY + hh2 * 0.3, cx + hw * 0.7, headY + hh2 * 0.3, cx, headY + hh2 * 1.8]);
	}
	// horns, tusks
	if (L.horns !== 'none') {
		c.fillStyle = '#e8dcc0';
		const big = L.horns === 'big' || L.horns === 'ram';
		const hy = headY - hh2 * 0.7;
		if (L.horns === 'ram') {
			c.strokeStyle = '#e8dcc0';
			c.lineWidth = Math.max(1.4, u * 0.07);
			c.beginPath();
			c.arc(cx - hw * 0.95, hy + hh2 * 0.2, hw * 0.55, -1, 3.4);
			c.stroke();
			c.beginPath();
			c.arc(cx + hw * 0.95, hy + hh2 * 0.2, hw * 0.55, -0.26, 4.1, true);
			c.stroke();
		} else {
			poly(c, [cx - hw * 0.7, hy + hh2 * 0.3, cx - hw * (big ? 1.5 : 1.1), hy - hh2 * (big ? 1.1 : 0.5), cx - hw * 0.25, hy + hh2 * 0.1]);
			poly(c, [cx + hw * 0.7, hy + hh2 * 0.3, cx + hw * (big ? 1.5 : 1.1), hy - hh2 * (big ? 1.1 : 0.5), cx + hw * 0.25, hy + hh2 * 0.1]);
		}
	}
	if (L.tusks) {
		c.fillStyle = '#f0e8d0';
		poly(c, [cx - hw * 0.5, headY + hh2 * 0.5, cx - hw * 0.35, headY + hh2 * 0.5, cx - hw * 0.45, headY + hh2 * 0.05]);
		poly(c, [cx + hw * 0.5, headY + hh2 * 0.5, cx + hw * 0.35, headY + hh2 * 0.5, cx + hw * 0.45, headY + hh2 * 0.05]);
	}
	// eyes
	c.fillStyle = L.eye;
	const es = Math.max(1, u * 0.07);
	c.fillRect(cx - hw * 0.5, headY - hh2 * 0.1, es, es * 1.1);
	c.fillRect(cx + hw * 0.5 - es, headY - hh2 * 0.1, es, es * 1.1);
	if (L.halo) {
		c.strokeStyle = '#ffe680';
		c.lineWidth = Math.max(1, u * 0.06);
		c.beginPath();
		c.ellipse(cx, headY - hh2 * 1.3, hw * 0.85, hh2 * 0.25, 0, 0, Math.PI * 2);
		c.stroke();
	}

	// headgear
	c.fillStyle = o.trim;
	switch (o.head) {
		case 'helm':
			c.fillStyle = armor >= 2 ? '#aeb4be' : '#8a909a';
			c.beginPath();
			c.arc(cx, headY - hh2 * 0.1, hw * 1.08, Math.PI, 0);
			c.fill();
			c.fillRect(cx - hw * 1.08, headY - hh2 * 0.12, hw * 2.16, hh2 * 0.28);
			c.fillStyle = '#b03a3a';
			poly(c, [cx - hw * 0.1, headY - hh2 * 1.1, cx + hw * 0.1, headY - hh2 * 1.1, cx + hw * 0.5, headY - hh2 * 1.7, cx, headY - hh2 * 1.3]);
			break;
		case 'hood':
			c.fillStyle = shade(cloth, -0.15);
			c.beginPath();
			c.arc(cx, headY - hh2 * 0.05, hw * 1.12, Math.PI * 0.95, Math.PI * 0.05);
			c.fill();
			c.fillStyle = skin;
			ell(c, cx, headY + hh2 * 0.15, hw * 0.7, hh2 * 0.7);
			c.fillStyle = L.eye;
			c.fillRect(cx - hw * 0.45, headY - hh2 * 0.05, es, es * 1.1);
			c.fillRect(cx + hw * 0.45 - es, headY - hh2 * 0.05, es, es * 1.1);
			break;
		case 'cap':
			c.fillStyle = shade(cloth, -0.1);
			c.beginPath();
			c.arc(cx, headY - hh2 * 0.25, hw * 1.0, Math.PI, 0);
			c.fill();
			c.fillRect(cx - hw * 1.1, headY - hh2 * 0.3, hw * 2.2, hh2 * 0.14);
			break;
		case 'hat':
			c.fillStyle = shade(cloth, -0.2);
			c.fillRect(cx - hw * 1.6, headY - hh2 * 0.5, hw * 3.2, hh2 * 0.18);
			c.fillRect(cx - hw * 0.8, headY - hh2 * 1.4, hw * 1.6, hh2 * 0.95);
			c.fillStyle = o.trim;
			c.fillRect(cx - hw * 0.8, headY - hh2 * 0.62, hw * 1.6, hh2 * 0.12);
			break;
		case 'goggles':
			c.fillStyle = '#4a4a50';
			c.fillRect(cx - hw * 1.05, headY - hh2 * 0.5, hw * 2.1, hh2 * 0.14);
			c.fillStyle = '#9ad8e8';
			ell(c, cx - hw * 0.45, headY - hh2 * 0.15, hw * 0.35, hw * 0.3);
			ell(c, cx + hw * 0.45, headY - hh2 * 0.15, hw * 0.35, hw * 0.3);
			break;
		case 'mask':
			c.fillStyle = '#25252c';
			c.fillRect(cx - hw * 0.95, headY - hh2 * 0.35, hw * 1.9, hh2 * 0.5);
			c.fillStyle = '#ffe9a0';
			c.fillRect(cx - hw * 0.5, headY - hh2 * 0.1, es, es * 0.8);
			c.fillRect(cx + hw * 0.5 - es, headY - hh2 * 0.1, es, es * 0.8);
			break;
		case 'bandana':
			c.fillStyle = shade(cloth, 0.1);
			c.fillRect(cx - hw * 1.0, headY - hh2 * 0.5, hw * 2, hh2 * 0.3);
			poly(c, [cx + hw, headY - hh2 * 0.4, cx + hw * 1.7, headY - hh2 * 0.1, cx + hw, headY - hh2 * 0.2]);
			break;
		case 'crown':
			c.fillStyle = '#f0d060';
			poly(c, [cx - hw * 0.9, headY - hh2 * 0.55, cx - hw * 0.9, headY - hh2 * 1.4, cx - hw * 0.45, headY - hh2 * 0.9, cx, headY - hh2 * 1.5, cx + hw * 0.45, headY - hh2 * 0.9, cx + hw * 0.9, headY - hh2 * 1.4, cx + hw * 0.9, headY - hh2 * 0.55]);
			break;
		case 'circlet':
			c.fillStyle = '#f0d060';
			c.fillRect(cx - hw * 0.95, headY - hh2 * 0.62, hw * 1.9, hh2 * 0.14);
			ell(c, cx, headY - hh2 * 0.68, hw * 0.18, hw * 0.18);
			break;
		case 'turban':
			c.fillStyle = cloth;
			ell(c, cx, headY - hh2 * 0.5, hw * 1.1, hh2 * 0.7);
			break;
		default:
			break;
	}
	c.restore();
}

// ---------------------------------------------------------------- creatures

function bodyPlanOf(a: QudActor, glyph: string): string {
	if (a.b) return a.b;
	switch (a.gr) {
		case 'mam':
		case 'syn':
			return 'quadruped';
		case 'bird':
			return 'avian';
		case 'rept':
		case 'dino':
			return 'saurian';
		case 'ptero':
			return 'avian';
		case 'amph':
			return 'saurian';
		case 'fish':
			return 'fish';
		case 'ins':
			return 'insect';
		case 'arach':
			return 'arachnid';
		case 'crust':
			return 'crab';
		case 'myria':
			return 'serpent';
		case 'worm':
			return 'serpent';
		case 'moll':
			return 'ooze';
		case 'cnid':
			return 'tentacled';
		case 'echi':
			return 'tentacled';
		case 'arth':
			return 'insect';
	}
	if ('vPYwy'.includes(glyph)) return 'avian';
	if ('bsaISj'.includes(glyph)) return 'insect';
	if ('ψJXO'.includes(glyph)) return 'tentacled';
	return 'humanoid';
}

/** a beast, monster or machine: silhouette by body plan, size by species, tint and markings by individual */
export function drawCreatureSprite2(c: Ctx, a: QudActor, g: string, fg: string, palette: string[], bgIdx: number, x: number, y: number, cw: number, chh: number) {
	const seed = hash(a.i + 91);
	const plan = bodyPlanOf(a, g);
	if (plan === 'humanoid' && !a.gr && (a.s || a.r)) {
		drawPersonSprite(c, a, fg, palette, bgIdx, x, y, cw, chh);
		return;
	}
	const u = cw;
	const v = chh;
	const size = a.z ? 0.55 + a.z * 0.08 : g >= 'A' && g <= 'Z' ? 1 : 0.8;
	const sz = size * (1 + (((seed >>> 4) % 9) - 4) * 0.03);
	const body = shade(fg, (((seed >>> 9) % 7) - 3) * 0.05);
	const belly = shade(body, 0.28);
	const dark = shade(body, -0.35);
	const cx = x + u * 0.5;
	const cy = y + v * 0.6;
	const ground = y + v * 0.9;
	const mark = (seed >>> 14) % 4; // 0 plain 1 stripes 2 spots 3 two-tone

	c.save();
	c.fillStyle = body;
	c.strokeStyle = body;
	const lw = Math.max(1, u * 0.1);
	const eye = bgIdx === 22 ? '#ffff55' : '#101818';
	const markings = (rx: number, ry: number) => {
		if (mark === 1) {
			c.strokeStyle = dark;
			c.lineWidth = 1;
			for (let k = -1; k <= 1; k++) line(c, cx + k * rx * 0.5, cy - ry * 0.8, cx + k * rx * 0.5 + rx * 0.1, cy + ry * 0.8, 1);
		} else if (mark === 2) {
			c.fillStyle = dark;
			for (let k = 0; k < 4; k++) ell(c, cx + (((seed >>> (k * 3)) & 7) / 7 - 0.5) * rx * 1.4, cy + (((seed >>> (k * 2 + 1)) & 7) / 7 - 0.5) * ry * 1.2, u * 0.04, u * 0.04);
		} else if (mark === 3) {
			c.fillStyle = belly;
			ell(c, cx, cy + ry * 0.4, rx * 0.85, ry * 0.45);
		}
		c.fillStyle = body;
	};

	switch (plan) {
		case 'quadruped': {
			const rx = u * 0.34 * sz;
			const ry = v * 0.17 * sz;
			// legs
			c.fillStyle = dark;
			for (const lx of [-0.62, -0.3, 0.3, 0.62]) c.fillRect(cx + rx * lx - lw * 0.4, cy, lw * 0.8, ground - cy);
			c.fillStyle = body;
			ell(c, cx, cy - ry * 0.2, rx, ry);
			markings(rx, ry);
			// head toward the right
			const hx = cx + rx * 0.95;
			const hy = cy - ry * 0.75;
			ell(c, hx, hy, u * 0.14 * sz, u * 0.12 * sz);
			ell(c, hx + u * 0.12 * sz, hy + u * 0.03, u * 0.09 * sz, u * 0.06 * sz);
			poly(c, [hx - u * 0.05, hy - u * 0.1 * sz, hx - u * 0.02, hy - u * 0.2 * sz, hx + u * 0.05, hy - u * 0.1 * sz]);
			// tail
			c.strokeStyle = body;
			c.lineWidth = lw * 0.7;
			c.beginPath();
			c.moveTo(cx - rx * 0.95, cy - ry * 0.4);
			c.quadraticCurveTo(cx - rx * 1.4, cy - ry * 1.5, cx - rx * 1.2, cy - ry * 2.0);
			c.stroke();
			c.fillStyle = eye;
			c.fillRect(hx + u * 0.02, hy - u * 0.03, Math.max(1, u * 0.05), Math.max(1, u * 0.05));
			if ((a.z ?? 0) >= 5) {
				c.strokeStyle = '#f0e8d0';
				c.lineWidth = Math.max(1, u * 0.05);
				c.beginPath();
				c.arc(hx + u * 0.2 * sz, hy + u * 0.1, u * 0.1 * sz, -1.2, 1.2);
				c.stroke();
			}
			break;
		}
		case 'saurian': {
			const rx = u * 0.34 * sz;
			const ry = v * 0.14 * sz;
			c.fillStyle = dark;
			c.fillRect(cx - rx * 0.4 - lw * 0.4, cy, lw * 0.8, ground - cy);
			c.fillRect(cx + rx * 0.4 - lw * 0.4, cy, lw * 0.8, ground - cy);
			c.fillStyle = body;
			ell(c, cx, cy - ry * 0.2, rx, ry);
			markings(rx, ry);
			poly(c, [cx - rx * 0.9, cy - ry * 0.5, cx - u * 0.52, ground - v * 0.08, cx - rx * 0.6, cy + ry * 0.2]);
			const hx = cx + rx * 1.0;
			const hy = cy - ry * 1.0;
			ell(c, hx, hy, u * 0.12 * sz, u * 0.09 * sz);
			ell(c, hx + u * 0.12 * sz, hy + u * 0.02, u * 0.1 * sz, u * 0.05 * sz);
			if ((seed & 3) === 0 || a.gr === 'dino') {
				c.fillStyle = dark;
				for (let k = 0; k < 4; k++) poly(c, [cx - rx * 0.6 + k * rx * 0.4, cy - ry * 1.0, cx - rx * 0.5 + k * rx * 0.4, cy - ry * 1.5, cx - rx * 0.4 + k * rx * 0.4, cy - ry * 1.0]);
			}
			c.fillStyle = eye;
			c.fillRect(hx + u * 0.02, hy - u * 0.03, Math.max(1, u * 0.05), Math.max(1, u * 0.05));
			break;
		}
		case 'avian': {
			const rx = u * 0.2 * sz;
			const ry = v * 0.17 * sz;
			c.strokeStyle = '#c8a050';
			line(c, cx - rx * 0.4, cy + ry, cx - rx * 0.4, ground, 1);
			line(c, cx + rx * 0.4, cy + ry, cx + rx * 0.4, ground, 1);
			c.fillStyle = body;
			ell(c, cx, cy, rx, ry);
			markings(rx, ry);
			// wings
			c.fillStyle = dark;
			poly(c, [cx - rx * 0.2, cy - ry * 0.6, cx - u * 0.48, cy - ry * 1.9, cx - u * 0.4, cy + ry * 0.2]);
			poly(c, [cx + rx * 0.2, cy - ry * 0.6, cx + u * 0.48, cy - ry * 1.9, cx + u * 0.4, cy + ry * 0.2]);
			c.fillStyle = body;
			ell(c, cx + rx * 0.7, cy - ry * 1.1, u * 0.09 * sz, u * 0.09 * sz);
			c.fillStyle = '#e8b030';
			poly(c, [cx + rx * 0.7 + u * 0.08, cy - ry * 1.15, cx + rx * 0.7 + u * 0.2, cy - ry * 1.0, cx + rx * 0.7 + u * 0.08, cy - ry * 0.9]);
			c.fillStyle = eye;
			c.fillRect(cx + rx * 0.7, cy - ry * 1.2, Math.max(1, u * 0.04), Math.max(1, u * 0.04));
			break;
		}
		case 'fish': {
			const rx = u * 0.34 * sz;
			const ry = v * 0.14 * sz;
			c.fillStyle = body;
			ell(c, cx, cy, rx, ry);
			markings(rx, ry);
			poly(c, [cx - rx * 0.8, cy, cx - rx * 1.35, cy - ry * 1.1, cx - rx * 1.35, cy + ry * 1.1]);
			c.fillStyle = dark;
			poly(c, [cx - rx * 0.1, cy - ry * 0.9, cx + rx * 0.3, cy - ry * 1.8, cx + rx * 0.4, cy - ry * 0.7]);
			c.fillStyle = eye;
			c.fillRect(cx + rx * 0.55, cy - ry * 0.3, Math.max(1, u * 0.05), Math.max(1, u * 0.05));
			break;
		}
		case 'insect':
		case 'arachnid':
		case 'crab': {
			const legs = plan === 'arachnid' ? 4 : plan === 'crab' ? 3 : 3;
			const rx = u * 0.22 * sz;
			const ry = v * 0.12 * sz;
			c.strokeStyle = dark;
			c.lineWidth = Math.max(1, u * 0.06);
			for (let k = 0; k < legs; k++) {
				const ly = cy - ry * 0.4 + k * ry * 0.55;
				line(c, cx - rx * 0.7, ly, cx - u * 0.46, ly + ry * (0.7 + k * 0.3), c.lineWidth);
				line(c, cx + rx * 0.7, ly, cx + u * 0.46, ly + ry * (0.7 + k * 0.3), c.lineWidth);
			}
			c.fillStyle = body;
			ell(c, cx, cy, rx, ry);
			if (plan !== 'crab') ell(c, cx - rx * 0.9, cy, rx * 0.7, ry * 0.8);
			markings(rx, ry);
			ell(c, cx + rx * 0.95, cy - ry * 0.1, rx * 0.5, ry * 0.7);
			if (plan === 'crab') {
				c.fillStyle = dark;
				ell(c, cx - rx * 1.35, cy - ry * 1.2, rx * 0.4, ry * 0.7);
				ell(c, cx + rx * 1.35, cy - ry * 1.2, rx * 0.4, ry * 0.7);
			}
			if (a.t && /bee|wasp|fly|moth|dragon|beetle/.test(a.t) || (seed & 3) === 0) {
				c.fillStyle = 'rgba(220,240,255,0.55)';
				ell(c, cx - rx * 0.3, cy - ry * 1.5, rx * 0.8, ry * 0.7);
				ell(c, cx + rx * 0.3, cy - ry * 1.5, rx * 0.8, ry * 0.7);
			}
			c.fillStyle = eye;
			c.fillRect(cx + rx * 1.1, cy - ry * 0.3, Math.max(1, u * 0.05), Math.max(1, u * 0.05));
			break;
		}
		case 'serpent': {
			c.lineWidth = lw * 1.1 * sz;
			c.lineCap = 'round';
			c.beginPath();
			const amp = v * 0.11 * sz;
			c.moveTo(x + u * 0.08, cy + amp);
			c.bezierCurveTo(x + u * 0.28, cy - amp * 2, x + u * 0.5, cy + amp * 2, x + u * 0.74, cy - amp * 0.4);
			c.stroke();
			ell(c, x + u * 0.78, cy - amp * 0.5, u * 0.1 * sz, u * 0.075 * sz);
			c.fillStyle = eye;
			c.fillRect(x + u * 0.8, cy - amp * 0.7, Math.max(1, u * 0.04), Math.max(1, u * 0.04));
			c.strokeStyle = '#d83a3a';
			line(c, x + u * 0.86, cy - amp * 0.5, x + u * 0.96, cy - amp * 0.7, 1);
			break;
		}
		case 'ooze': {
			c.beginPath();
			c.arc(cx, cy + v * 0.04, u * 0.34 * sz, Math.PI, 0);
			c.lineTo(cx + u * 0.4 * sz, ground);
			c.lineTo(cx - u * 0.4 * sz, ground);
			c.closePath();
			c.fill();
			c.fillStyle = belly;
			ell(c, cx - u * 0.1, cy - v * 0.06, u * 0.07, u * 0.05);
			c.fillStyle = eye;
			c.fillRect(cx - u * 0.12, cy, Math.max(1, u * 0.06), Math.max(1, u * 0.06));
			c.fillRect(cx + u * 0.08, cy, Math.max(1, u * 0.06), Math.max(1, u * 0.06));
			break;
		}
		case 'tentacled': {
			c.beginPath();
			c.arc(cx, cy - v * 0.1, u * 0.3 * sz, Math.PI, 0);
			c.fill();
			c.lineWidth = Math.max(1, u * 0.06);
			for (let k = -2; k <= 2; k++) {
				c.strokeStyle = body;
				c.beginPath();
				c.moveTo(cx + k * u * 0.12, cy - v * 0.1);
				c.quadraticCurveTo(cx + k * u * 0.2 + ((seed >>> (k + 4)) & 3) * 2, cy + v * 0.12, cx + k * u * 0.16, ground);
				c.stroke();
			}
			c.fillStyle = eye;
			c.fillRect(cx - u * 0.1, cy - v * 0.18, Math.max(1, u * 0.06), Math.max(1, u * 0.06));
			c.fillRect(cx + u * 0.06, cy - v * 0.18, Math.max(1, u * 0.06), Math.max(1, u * 0.06));
			break;
		}
		case 'golem': {
			const bw = u * 0.5 * sz;
			c.fillStyle = body;
			c.fillRect(cx - bw / 2, cy - v * 0.2, bw, v * 0.4);
			c.fillStyle = dark;
			c.fillRect(cx - bw / 2 - u * 0.1, cy - v * 0.16, u * 0.1, v * 0.3);
			c.fillRect(cx + bw / 2, cy - v * 0.16, u * 0.1, v * 0.3);
			c.fillRect(cx - bw / 2 + u * 0.04, cy + v * 0.2, bw * 0.38, ground - cy - v * 0.2);
			c.fillRect(cx + bw * 0.1, cy + v * 0.2, bw * 0.38, ground - cy - v * 0.2);
			c.fillStyle = body;
			c.fillRect(cx - u * 0.14, cy - v * 0.38, u * 0.28, v * 0.16);
			c.fillStyle = bgIdx === 22 ? '#ffff55' : '#ffb03a';
			c.fillRect(cx - u * 0.08, cy - v * 0.32, Math.max(1, u * 0.16), Math.max(1, v * 0.03));
			break;
		}
		default: {
			// a monster in rough human shape: hunched, horned or crested by individual
			drawPersonSprite(c, { ...a, s: a.s ?? ['orc', 'goblin', 'troll', 'beastman', 'ogor', 'lizardfolk', 'warp_fiend'][seed % 7] }, fg, palette, bgIdx, x, y, cw, chh);
			c.restore();
			return;
		}
	}
	// letter badge keeps the roguelike reading
	if (chh >= 14) {
		c.fillStyle = palette[3] || '#ffffff';
		c.font = `${Math.max(8, Math.round(chh * 0.4))}px "Share Tech Mono", monospace`;
		c.textAlign = 'right';
		c.textBaseline = 'bottom';
		c.fillText(g, x + cw - 0.5, y + chh);
	}
	c.restore();
}
