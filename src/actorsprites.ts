// Sprite-mode portraits for actors: every species has its own build, skin and features, every
// calling and role its own outfit and gear, and each individual varies (hair, cloth, build, tint)
// by their id. The engine sends who stands where (`frame.actors`, engine/sprinfo.go); the glyph
// grid still decides what is visible, so nothing here changes the game.
//
// Poses (src/poses.ts): drawPersonSprite and drawCreatureSprite2 read the actor's pose
// (`poseOf(actor.i, performance.now())`, or the one the caller passes so gear and body agree) and move
// the limbs. At the static rest pose (poses off, reduceMotion, untracked actors) every pose value is
// zero and each posed branch falls back to the exact old drawing calls, so nothing changes there.
// Sign conventions used below (canvas y points down, rotate(+a) turns clockwise on screen):
// - a hanging limb at angle d from its rest (poses.ts: + = tip to the right) is drawn with rotate(-d)
//   about its root, or with the inline rotation (ex, ey) -> (ex cos d + ey sin d, -ex sin d + ey cos d);
// - an upright part (torso lean, lie, a held weapon) leaning right by a is drawn with rotate(+a).

import { poseOf, limbAngles, creaturePose, castGlow, applyBodyFrame, drawCastGlow, ARM_REST, legRest, type Pose, type Limbs, type CreatureParams } from './poses';

export interface QudActor {
	x: number;
	y: number;
	i: number; // actor id: the seed of this one's variation
	gx?: boolean; // gearsprites.ts draws this one's gear: the built-in outfit item and back piece stay off
	ga?: boolean; // gearsprites.ts draws the posed right (weapon) sleeve: skip the built-in right arm line (the hand dot stays)
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
	tr?: string; // tribe/subrace color tag e.g. tr_hex:#skin:#accent
	fl?: string[]; // optional actor flags
	ge?: boolean; // gear below is authoritative (paper doll, portraits): no calling defaults
	hd?: string; // worn head item name
	bk?: string; // worn back item name (cloak, mantle)
	ft?: string; // worn footwear name
	hn?: string; // worn handwear name
	bi?: string; // worn body item name
	sh?: boolean; // carries a shield
	nm?: string; // display name (tooltips)
	ti?: string; // title
	hp?: number; // health percent
	ho?: boolean; // hostile to the player
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
	boots?: string; // footwear colour
	gloves?: string; // handwear colour
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

/** Worn items from the engine (paper doll and portraits) replace the outfit's head, back, shield and feet. */
function applyGear(out: Outfit, a: QudActor) {
	const hd = (a.hd ?? '').toLowerCase();
	out.head = !hd
		? 'none'
		: /crown|diadem|tiara/.test(hd)
			? 'crown'
			: /circlet|laurel|halo/.test(hd)
				? 'circlet'
				: /helm|kabuto|casque|sallet|armet|bascinet|visor|morion|shako/.test(hd)
					? 'helm'
					: /hood|cowl|cap-?hood/.test(hd)
						? 'hood'
						: /turban|pagri|kolah|keffiyeh|headcloth/.test(hd)
							? 'turban'
							: /goggle|lens|monocle/.test(hd)
								? 'goggles'
								: /mask|veil|face/.test(hd)
									? 'mask'
									: /hat|caping|malakhai|sedge|straw|brim/.test(hd)
										? 'hat'
										: 'cap';
	const bk = (a.bk ?? '').toLowerCase();
	if (bk) out.back = /cape|mantle|haori|shawl|sash/.test(bk) ? 'cape' : 'cloak';
	else if (out.back === 'cape' || out.back === 'cloak') out.back = 'none';
	out.shield = !!a.sh;
	if (a.p && !a.w) out.item = 'none';
	const ft = (a.ft ?? '').toLowerCase();
	out.boots = ft ? (/greave|sabaton|plate|steel|iron|mail|armou?r/.test(ft) ? '#8a909a' : '#5a3a22') : undefined;
	const hn = (a.hn ?? '').toLowerCase();
	out.gloves = hn ? (/gauntlet|plate|steel|iron|mail/.test(hn) ? '#aeb4be' : '#6a4a2a') : undefined;
	void a.bi;
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
	if (a.ge) applyGear(out, a);
	if (a.k && /high|noble|lord|elite/i.test(a.k)) out.fine = true;
	if (a.gx) {
		out.item = 'none';
		out.back = 'none';
	}
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


let spriteAnimTick = 0;
let spriteHitFlashSet = new Set<string>();

/** Updates the global sprite animation tick and optional set of hit-flashing actor IDs/names. */
export function setSpriteAnimTick(tick: number, hitFlashActors?: Set<string | number>) {
	spriteAnimTick = Number.isFinite(tick) ? tick : 0;
	spriteHitFlashSet = new Set<string>();
	if (hitFlashActors) {
		for (const id of hitFlashActors) spriteHitFlashSet.add(String(id));
	}
}

/** Parses a tribe/subrace color tag of the form "tr_hex:#skin:#accent" (or embedded in flags/caste). */
export function parseTribeHex(src?: string | string[]): { skin?: string; accent?: string } | null {
	if (!src) return null;
	const list = Array.isArray(src) ? src : [src];
	for (const item of list) {
		const m = /tr_hex:(#[0-9a-fA-F]{6})(?::(#[0-9a-fA-F]{6}))?/.exec(item);
		if (m) return { skin: m[1], accent: m[2] };
	}
	return null;
}

function actorTribeColors(a: QudActor): { skin?: string; accent?: string } | null {
	return parseTribeHex(a.tr) ?? parseTribeHex(a.fl) ?? parseTribeHex(a.k) ?? parseTribeHex(a.s);
}

function actorAnimBobAndFlash(a: QudActor, chh: number, bgIdx: number): { bobY: number; flash: boolean } {
	const phase = (a.i * 1.37) + (a.x * 0.7) + (a.y * 0.5);
	const bobY = spriteAnimTick > 0 ? Math.sin(spriteAnimTick * 0.22 + phase) * Math.max(0.6, chh * 0.03) : 0;
	const flash = bgIdx === 18 || spriteHitFlashSet.has(String(a.i));
	return { bobY, flash };
}

// scratch objects for the pose (no allocation per draw; the drawers are not re-entrant)
const PERSON_POSE = {} as Pose;
const PERSON_LIMBS = {} as Limbs;
const PERSON_BUILD = { h: 1, w: 1 };
const CREATURE_POSE = {} as Pose;
const CREATURE_PARAMS = {} as CreatureParams;

/** rotate(+a) about (px, py): the upright-part lean (torso). save() only when a != 0; the caller restores when a != 0. */
function leanAbout(c: Ctx, a: number, px: number, py: number) {
	if (!a) return;
	c.save();
	c.translate(px, py);
	c.rotate(a); // + = clockwise on screen = the top leans to the RIGHT
	c.translate(-px, -py);
}

/** A posed leg: rect from the hip down to the (lifted) foot, rotated by d about the hip; rotate(-d) moves the foot RIGHT for d > 0. */
function posedLeg(c: Ctx, hx: number, hipY: number, ground: number, lift: number, d: number, lw: number, v: number, left: boolean, legCol: string, footCol: string, boots: string | undefined) {
	c.save();
	if (d) {
		c.translate(hx, hipY);
		c.rotate(-d);
		c.translate(-hx, -hipY);
	}
	const fy = ground - lift * v; // foot lift: the leg shortens, the foot rises
	const fx = left ? hx - lw * 0.7 : hx - lw * 0.3; // the old feet: the left one centred, the right one toes-out
	c.fillStyle = legCol;
	c.fillRect(hx - lw / 2, hipY, lw, fy - hipY);
	c.fillStyle = footCol;
	c.fillRect(fx, fy - v * 0.04, lw * 1.4, v * 0.04);
	if (boots) {
		c.fillStyle = boots;
		c.fillRect(hx - lw * 0.6, fy - v * 0.2, lw * 1.2, v * 0.2);
		c.fillRect(fx, fy - v * 0.04, lw * 1.4, v * 0.04);
	}
	c.restore();
}

/**
 * a person: species body plan, calling/role outfit, and per-individual variation. `pose` (optional):
 * the caller's pose for this actor (sprites.ts passes the one it also gives the gear layer); without
 * it the pose is read here with poseOf(a.i, performance.now()).
 */
export function drawPersonSprite(c: Ctx, a: QudActor, fg: string, palette: string[], bgIdx: number, x: number, y: number, cw: number, chh: number, pose?: Pose) {
	const seed = hash(a.i + 17);
	const sp = a.s ?? 'human';
	const L = lookFor(sp);
	const o = outfitFor(a, seed);
	const trCol = actorTribeColors(a);
	const { bobY, flash } = actorAnimBobAndFlash(a, chh, bgIdx);
	y += bobY;
	const skin = flash ? '#ffffff' : (trCol?.skin ?? pick(L.skin, seed >>> 3));
	const hair = L.hair ? pick(L.hair, seed >>> 7) : undefined;
	const cloth = flash ? '#ff5555' : shade(trCol?.accent ?? pick(o.cloth, seed >>> 11), ((seed >>> 16) % 7 - 3) * 0.04);
	const dark = shade(cloth, -0.35);
	const steel = '#c9ccd2';
	const u = cw;
	const v = chh;
	const vary = 1 + (((seed >>> 20) % 9) - 4) * 0.015;
	// pose: at rest every value below is 0 (arms/legs exactly at their rest angles)
	const P = pose ?? poseOf(a.i, performance.now(), PERSON_POSE);
	PERSON_BUILD.h = L.h;
	PERSON_BUILD.w = L.w;
	const Lm = limbAngles(P, PERSON_BUILD, PERSON_LIMBS);
	const cx0 = x + u * 0.5; // cell centre: ring, aura, swarm, lie/mirror pivot
	const ground = y + v * 0.9; // feet line: the feet stay here when the body drops
	const cx = cx0 + Lm.dx * u; // recoil shift (+ = right)
	const base = ground + Lm.drop * v; // body drop (+ = down): hips, torso and head move, feet do not
	const H = v * 0.82 * L.h * vary;
	const W = u * 0.46 * L.w;
	const hr = u * 0.2 * L.hd * (1 + (L.h < 0.75 ? 0.15 : 0));
	const neckY = base - H + hr + hr * 0.85; // = old headY + hr * 0.85 (the neck does not bob with the head)
	const headY = base - H + hr + Lm.headBob * v; // head bob (+ = down)
	const hipY = base - H * 0.34;
	const lean = Lm.torsoLean; // + = upper body leans right, rotate(+lean) about the hips
	const dAL = Lm.armL + ARM_REST; // arm deltas from rest (+ = hand moves right)
	const dAR = Lm.armR - ARM_REST;
	const shut = P.sleep > 0 || P.dead > 0.5; // eyes closed

	c.save();
	// rings: the player and companions are marked as before
	const ring = a.p ? '#cfc041' : bgIdx === 21 ? palette[9] || '#55ffff' : null;
	if (ring) {
		c.strokeStyle = ring;
		c.lineWidth = 1;
		c.beginPath();
		c.ellipse(cx0, y + v * 0.92, u * 0.4, v * 0.06, 0, 0, Math.PI * 2);
		c.stroke();
	}
	if (L.glow) {
		const gr = c.createRadialGradient(cx0, y + v * 0.5, 1, cx0, y + v * 0.5, v * 0.55);
		gr.addColorStop(0, L.glow + '66');
		gr.addColorStop(1, L.glow + '00');
		c.fillStyle = gr;
		c.fillRect(x, y, u, v);
	}

	if (L.swarm) {
		// a chitin swarm: a cluster of scuttling bodies (not posed)
		for (let k = 0; k < 9; k++) {
			const hh = hash(a.i, k + 3);
			const bx = cx0 + (((hh % 100) / 100) - 0.5) * u * 0.8;
			const by = y + v * 0.45 + (((hh >>> 8) % 100) / 100) * v * 0.42;
			c.fillStyle = k % 3 === 0 ? shade(skin, 0.25) : skin;
			ell(c, bx, by, u * 0.1, v * 0.045);
			c.strokeStyle = skin;
			line(c, bx - u * 0.1, by + v * 0.04, bx + u * 0.1, by - v * 0.03, 1);
		}
		c.fillStyle = L.eye;
		ell(c, cx0, y + v * 0.5, u * 0.04, u * 0.04);
		c.restore();
		return;
	}

	// whole-body frame: lie down (sleep, death) about the body's middle, mirror when striking left
	// (same helper and arguments as gearsprites drawGear, so worn gear follows)
	const mirror = P.attack > 0 && P.face < 0;
	if (Lm.lie || mirror) applyBodyFrame(c, Lm.lie, mirror, cx0, ground, H, u);

	// wings and tail go behind the body (wings and cape lean with the torso)
	leanAbout(c, lean, cx, hipY);
	if (L.wings) {
		c.fillStyle = '#f4ecd0';
		poly(c, [cx - W * 0.4, neckY + v * 0.04, cx - u * 0.52, headY - hr * 0.6, cx - u * 0.38, hipY, cx - W * 0.3, hipY - v * 0.04]);
		poly(c, [cx + W * 0.4, neckY + v * 0.04, cx + u * 0.52, headY - hr * 0.6, cx + u * 0.38, hipY, cx + W * 0.3, hipY - v * 0.04]);
	}
	if (o.back === 'cape' || o.back === 'cloak') {
		c.fillStyle = shade(dark, -0.1);
		poly(c, [cx - W * 0.55, neckY, cx + W * 0.55, neckY, cx + W * 0.7, base - v * 0.04, cx - W * 0.7, base - v * 0.04]);
	}
	if (lean) c.restore();
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
		const lw = Math.max(1.5, u * 0.14 * Math.min(1.3, L.w));
		const lr = legRest(L.w);
		// leg deltas from rest (+ = foot moves right); rounding noise of the width term is treated as rest
		const dLL = Math.abs(Lm.legL + lr) < 1e-6 ? 0 : Lm.legL + lr;
		const dLR = Math.abs(Lm.legR - lr) < 1e-6 ? 0 : Lm.legR - lr;
		if (!dLL && !dLR && !Lm.liftL && !Lm.liftR) {
			// standing: the old two fixed legs (hips may drop, the feet stay on the ground line)
			c.fillStyle = L.bone ? skin : dark;
			c.fillRect(cx - W * 0.32 - lw / 2, hipY, lw, ground - hipY);
			c.fillRect(cx + W * 0.32 - lw / 2, hipY, lw, ground - hipY);
			c.fillStyle = shade(dark, -0.3);
			c.fillRect(cx - W * 0.32 - lw * 0.7, ground - v * 0.04, lw * 1.4, v * 0.04);
			c.fillRect(cx + W * 0.32 - lw * 0.3, ground - v * 0.04, lw * 1.4, v * 0.04);
			if (o.boots) {
				c.fillStyle = o.boots;
				c.fillRect(cx - W * 0.32 - lw * 0.6, ground - v * 0.2, lw * 1.2, v * 0.2);
				c.fillRect(cx + W * 0.32 - lw * 0.6, ground - v * 0.2, lw * 1.2, v * 0.2);
				c.fillRect(cx - W * 0.32 - lw * 0.7, ground - v * 0.04, lw * 1.4, v * 0.04);
				c.fillRect(cx + W * 0.32 - lw * 0.3, ground - v * 0.04, lw * 1.4, v * 0.04);
			}
		} else {
			// walking / falling: each leg swings from its hip, the lifted foot rises
			const legCol = L.bone ? skin : dark;
			const footCol = shade(dark, -0.3);
			posedLeg(c, cx - W * 0.32, hipY, ground, Lm.liftL, dLL, lw, v, true, legCol, footCol, o.boots);
			posedLeg(c, cx + W * 0.32, hipY, ground, Lm.liftR, dLR, lw, v, false, legCol, footCol, o.boots);
		}
	}

	// torso, arms and head lean together about the hips (restored at the end)
	leanAbout(c, lean, cx, hipY);
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
	// primary arms swing from the shoulders: the rest hand is rotated about the shoulder by the delta
	// (ex, ey) -> (ex cos d + ey sin d, -ex sin d + ey cos d), so d > 0 moves the hand to the RIGHT
	// (same formula as gearsprites' weapon arm). A second pair (four arms) stays at rest like the gear sleeves.
	const shY = neckY + v * 0.04;
	const lsx = cx - W * 0.5;
	const rsx = cx + W * 0.5;
	let lhx = cx - W * 0.68;
	let lhy = hipY - v * 0.0;
	let rhx = cx + W * 0.66;
	let rhy = hipY - v * 0.02;
	if (dAL) {
		const ca = Math.cos(dAL), sa = Math.sin(dAL), ex = lhx - lsx, ey = lhy - shY;
		lhx = lsx + ex * ca + ey * sa;
		lhy = shY - ex * sa + ey * ca;
	}
	if (dAR) {
		const ca = Math.cos(dAR), sa = Math.sin(dAR), ex = rhx - rsx, ey = rhy - shY;
		rhx = rsx + ex * ca + ey * sa;
		rhy = shY - ex * sa + ey * ca;
	}
	const pairs = L.arms === 4 ? 2 : 1;
	for (let k = 0; k < pairs; k++) {
		const off = k * v * 0.1;
		if (k === 0) {
			line(c, lsx, shY, lhx, lhy, aw);
			// the gear layer draws the posed weapon sleeve itself (`ga`): no second arm under it
			if (!a.ga) line(c, rsx, shY, rhx, rhy, aw);
		} else {
			line(c, cx - W * 0.5, neckY + v * 0.04 + off, cx - W * 0.68, hipY - v * 0.0 + off * 0.5, aw);
			line(c, cx + W * 0.5, neckY + v * 0.04 + off, cx + W * 0.66, hipY - v * 0.02 + off * 0.5, aw);
		}
	}
	c.fillStyle = o.gloves ?? skin;
	ell(c, lhx, lhy, u * 0.05, u * 0.05);
	ell(c, rhx, rhy, u * 0.05, u * 0.05);
	if (o.shield) {
		// shield rides the left hand (rest: the old cx - 0.78 W, hipY - 0.1 v)
		const sx = dAL ? lhx - W * 0.1 : cx - W * 0.78;
		const sy = dAL ? lhy - v * 0.1 : hipY - v * 0.1;
		c.fillStyle = shade(o.cloth[0], -0.1);
		ell(c, sx, sy, u * 0.14, v * 0.12);
		c.fillStyle = o.trim;
		ell(c, sx, sy, u * 0.05, v * 0.04);
	}
	if (!dAR) drawItem(c, o.item, cx + W * 0.66, hipY - v * 0.02, u, v, o.trim, steel, 1);
	else if (o.item !== 'none') {
		// held item at the posed hand, tipped by half the arm swing: rotate(+dAR/2) turns the item's up
		// direction clockwise (forward/right), matching gearsprites' weapon (ctx.rotate(rot * 0.5))
		c.save();
		c.translate(rhx, rhy);
		c.rotate(dAR * 0.5);
		drawItem(c, o.item, 0, 0, u, v, o.trim, steel, 1);
		c.restore();
	}

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
	// asleep or fallen: thin closed lids at the lower half of the eye
	const eh = shut ? Math.max(0.6, es * 0.35) : es * 1.1;
	const eo = shut ? es * 0.6 : 0;
	c.fillRect(cx - hw * 0.5, headY - hh2 * 0.1 + eo, es, eh);
	c.fillRect(cx + hw * 0.5 - es, headY - hh2 * 0.1 + eo, es, eh);
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
			c.fillRect(cx - hw * 0.45, headY - hh2 * 0.05 + eo, es, eh);
			c.fillRect(cx + hw * 0.45 - es, headY - hh2 * 0.05 + eo, es, eh);
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
	// cast: a soft glow at both (posed) hands, alpha = castGlow (0 when not casting: nothing drawn)
	const glow = castGlow(P);
	if (glow > 0) {
		const gc = L.glow ?? '#9fd8ff';
		drawCastGlow(c, lhx, lhy, u * 0.4, gc, glow);
		drawCastGlow(c, rhx, rhy, u * 0.4, gc, glow);
	}
	if (lean) c.restore(); // the torso lean
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

const CREATURE_PLANS = new Set(['quadruped', 'saurian', 'avian', 'fish', 'insect', 'arachnid', 'crab', 'serpent', 'ooze', 'tentacled', 'golem']);
const QUAD_LEGS = [-0.62, -0.3, 0.3, 0.62];

/** rotate(-d) about (px, py): a hanging part (leg) swung by d, + = its tip moves RIGHT. save() only when d != 0; the caller restores when d != 0. */
function swingAbout(c: Ctx, d: number, px: number, py: number) {
	if (!d) return;
	c.save();
	c.translate(px, py);
	c.rotate(-d);
	c.translate(-px, -py);
}

/** rotate(+a) about (px, py) (clockwise on screen). save() only when a != 0; the caller restores when a != 0. */
function turnAbout(c: Ctx, a: number, px: number, py: number) {
	if (!a) return;
	c.save();
	c.translate(px, py);
	c.rotate(a);
	c.translate(-px, -py);
}

/**
 * a beast, monster or machine: silhouette by body plan, size by species, tint and markings by individual.
 * Posed by poses.ts creaturePose (see the header there): legs alternate (legA/legB), wings flap, serpents
 * undulate, fish wag, oozes squash, tentacles sway, golems stomp; bob dy, recoil dx, hurt tilt, death lie,
 * cast glow; mirrored when facing left (the art faces right). Every pose value is 0 at the static rest
 * pose and each posed expression then reduces to the old drawing. `pose`: optional, as for drawPersonSprite.
 */
export function drawCreatureSprite2(c: Ctx, a: QudActor, g: string, fg: string, palette: string[], bgIdx: number, x: number, y: number, cw: number, chh: number, pose?: Pose) {
	const seed = hash(a.i + 91);
	const plan = bodyPlanOf(a, g);
	const P0 = pose ?? poseOf(a.i, performance.now(), CREATURE_POSE);
	if (plan === 'humanoid' && !a.gr && (a.s || a.r)) {
		drawPersonSprite(c, a, fg, palette, bgIdx, x, y, cw, chh, P0);
		return;
	}
	const u = cw;
	const v = chh;
	const size = a.z ? 0.55 + a.z * 0.08 : g >= 'A' && g <= 'Z' ? 1 : 0.8;
	const trCol = actorTribeColors(a);
	const { bobY, flash } = actorAnimBobAndFlash(a, chh, bgIdx);
	y += bobY;
	if (!CREATURE_PLANS.has(plan)) {
		// a monster in rough human shape: hunched, horned or crested by individual (posed as a person)
		drawPersonSprite(c, { ...a, s: a.s ?? ['orc', 'goblin', 'troll', 'beastman', 'ogor', 'lizardfolk', 'warp_fiend'][seed % 7] }, fg, palette, bgIdx, x, y, cw, chh, P0);
		return;
	}
	const C = creaturePose(plan, P0, CREATURE_PARAMS);
	const sz = size * (1 + (((seed >>> 4) % 9) - 4) * 0.03);
	const body = flash ? '#ff6666' : shade(trCol?.skin ?? fg, (((seed >>> 9) % 7) - 3) * 0.05);
	const belly = shade(body, 0.28);
	const dark = shade(body, -0.35);
	const cx = x + u * 0.5;
	const cy = y + v * 0.6 + C.dy * v; // body bob (+ = down); the ground line stays, so legs stretch and squash
	const ground = y + v * 0.9;
	const mark = (seed >>> 14) % 4; // 0 plain 1 stripes 2 spots 3 two-tone
	const hl = C.lunge * u * 0.5; // head thrust toward the target (+ = forward, the art's right), < 0 winding up

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

	// ---- pose frame (inner save; the letter badge below stays upright and unmirrored)
	c.save();
	if (C.dx) c.translate(C.dx * u, 0); // recoil shift, + = right
	// hurt tilt about the feet: rotate(+t) is clockwise on screen, so t > 0 tips the top to the RIGHT
	// (creaturePose gives t = -dir * 0.22: the body rocks against the push)
	if (C.tilt) {
		c.translate(cx, ground);
		c.rotate(C.tilt);
		c.translate(-cx, -ground);
	}
	// facing left: mirror about the cell centre (the art faces right)
	if (P0.face < 0) {
		c.translate(cx, 0);
		c.scale(-1, 1);
		c.translate(-cx, 0);
	}
	const ooze = plan === 'ooze';
	if (C.lie && !ooze) {
		// death: roll belly-up about the body centre (lie 0..1.5 -> 0..pi, sign = fall direction, rotate(+)
		// clockwise), sinking toward the ground. Rotating about the feet instead would throw the body out
		// of its cell (the map clips each sprite to its cell).
		const k = Math.min(1, Math.abs(C.lie) / 1.5);
		const sink = (ground - cy) * 0.6 * k;
		c.translate(cx, cy + sink);
		c.rotate((C.lie < 0 ? -1 : 1) * k * Math.PI);
		c.translate(-cx, -cy);
	}
	// squash and stretch about the ground point (ooze breathing, sleeping curl); a dead ooze flattens
	let sqx = C.sx;
	let sqy = C.sy;
	if (ooze && P0.dead > 0) {
		sqx += 0.3 * P0.dead;
		sqy -= 0.55 * P0.dead;
	}
	if (sqx || sqy) {
		c.translate(cx, ground);
		c.scale(1 + sqx, 1 + sqy);
		c.translate(-cx, -ground);
	}

	switch (plan) {
		case 'quadruped': {
			const rx = u * 0.34 * sz;
			const ry = v * 0.17 * sz;
			// legs: swung about their tops; i even legA, odd legB, so the hind pair and the fore pair each alternate
			c.fillStyle = dark;
			for (let i = 0; i < 4; i++) {
				const lx = QUAD_LEGS[i];
				const th = i & 1 ? C.legB : C.legA;
				swingAbout(c, th, cx + rx * lx, cy);
				c.fillRect(cx + rx * lx - lw * 0.4, cy, lw * 0.8, ground - cy);
				if (th) c.restore();
			}
			c.fillStyle = body;
			ell(c, cx, cy - ry * 0.2, rx, ry);
			markings(rx, ry);
			// head toward the right
			const hx = cx + rx * 0.95 + hl;
			const hy = cy - ry * 0.75;
			ell(c, hx, hy, u * 0.14 * sz, u * 0.12 * sz);
			ell(c, hx + u * 0.12 * sz, hy + u * 0.03, u * 0.09 * sz, u * 0.06 * sz);
			poly(c, [hx - u * 0.05, hy - u * 0.1 * sz, hx - u * 0.02, hy - u * 0.2 * sz, hx + u * 0.05, hy - u * 0.1 * sz]);
			// tail, wagged about its root (rotate(+) clockwise)
			c.strokeStyle = body;
			c.lineWidth = lw * 0.7;
			const tx0 = cx - rx * 0.95;
			const ty0 = cy - ry * 0.4;
			turnAbout(c, C.tail * 0.8, tx0, ty0);
			c.beginPath();
			c.moveTo(tx0, ty0);
			c.quadraticCurveTo(cx - rx * 1.4, cy - ry * 1.5, cx - rx * 1.2, cy - ry * 2.0);
			c.stroke();
			if (C.tail * 0.8) c.restore();
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
			swingAbout(c, C.legA, cx - rx * 0.4, cy);
			c.fillRect(cx - rx * 0.4 - lw * 0.4, cy, lw * 0.8, ground - cy);
			if (C.legA) c.restore();
			swingAbout(c, C.legB, cx + rx * 0.4, cy);
			c.fillRect(cx + rx * 0.4 - lw * 0.4, cy, lw * 0.8, ground - cy);
			if (C.legB) c.restore();
			c.fillStyle = body;
			ell(c, cx, cy - ry * 0.2, rx, ry);
			markings(rx, ry);
			// tail swish about its root (rotate(+) clockwise)
			turnAbout(c, C.tail * 0.5, cx - rx * 0.75, cy - ry * 0.15);
			poly(c, [cx - rx * 0.9, cy - ry * 0.5, cx - u * 0.52, ground - v * 0.08, cx - rx * 0.6, cy + ry * 0.2]);
			if (C.tail * 0.5) c.restore();
			const hx = cx + rx * 1.0 + hl;
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
			// legs: the feet swing by sin(leg) x leg length (+ = right)
			const legH = ground - cy - ry;
			c.strokeStyle = '#c8a050';
			line(c, cx - rx * 0.4, cy + ry, cx - rx * 0.4 + Math.sin(C.legA) * legH, ground, 1);
			line(c, cx + rx * 0.4, cy + ry, cx + rx * 0.4 + Math.sin(C.legB) * legH, ground, 1);
			c.fillStyle = body;
			ell(c, cx, cy, rx, ry);
			markings(rx, ry);
			// wings, flapped about their roots: + wing = raised. The left wing turns clockwise (rotate(+w):
			// its up-left tip swings up), the right one counter-clockwise (rotate(-w))
			const wa = C.wing * 0.6;
			c.fillStyle = dark;
			turnAbout(c, wa, cx - rx * 0.2, cy - ry * 0.6);
			poly(c, [cx - rx * 0.2, cy - ry * 0.6, cx - u * 0.48, cy - ry * 1.9, cx - u * 0.4, cy + ry * 0.2]);
			if (wa) c.restore();
			turnAbout(c, -wa, cx + rx * 0.2, cy - ry * 0.6);
			poly(c, [cx + rx * 0.2, cy - ry * 0.6, cx + u * 0.48, cy - ry * 1.9, cx + u * 0.4, cy + ry * 0.2]);
			if (wa) c.restore();
			const hx = cx + rx * 0.7 + hl;
			c.fillStyle = body;
			ell(c, hx, cy - ry * 1.1, u * 0.09 * sz, u * 0.09 * sz);
			c.fillStyle = '#e8b030';
			poly(c, [hx + u * 0.08, cy - ry * 1.15, hx + u * 0.2, cy - ry * 1.0, hx + u * 0.08, cy - ry * 0.9]);
			c.fillStyle = eye;
			c.fillRect(hx, cy - ry * 1.2, Math.max(1, u * 0.04), Math.max(1, u * 0.04));
			break;
		}
		case 'fish': {
			const rx = u * 0.34 * sz;
			const ry = v * 0.14 * sz;
			c.fillStyle = body;
			ell(c, cx, cy, rx, ry);
			markings(rx, ry);
			// tail fin wag about its root (rotate(+) clockwise), plus a little of the body wave
			const fa = C.tail * 0.5 + C.undAmp * Math.sin(C.undPhase) * 0.3;
			turnAbout(c, fa, cx - rx * 0.8, cy);
			poly(c, [cx - rx * 0.8, cy, cx - rx * 1.35, cy - ry * 1.1, cx - rx * 1.35, cy + ry * 1.1]);
			if (fa) c.restore();
			c.fillStyle = dark;
			poly(c, [cx - rx * 0.1, cy - ry * 0.9, cx + rx * 0.3, cy - ry * 1.8, cx + rx * 0.4, cy - ry * 0.7]);
			c.fillStyle = eye;
			c.fillRect(cx + rx * 0.55 + hl, cy - ry * 0.3, Math.max(1, u * 0.05), Math.max(1, u * 0.05));
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
				// alternating tripods: right legs take legA, legB, legA.., left legs the other one;
				// the foot shifts by leg x 0.1 cell (+ = right)
				const tR = k & 1 ? C.legB : C.legA;
				const tL = k & 1 ? C.legA : C.legB;
				line(c, cx - rx * 0.7, ly, cx - u * 0.46 + tL * u * 0.1, ly + ry * (0.7 + k * 0.3), c.lineWidth);
				line(c, cx + rx * 0.7, ly, cx + u * 0.46 + tR * u * 0.1, ly + ry * (0.7 + k * 0.3), c.lineWidth);
			}
			c.fillStyle = body;
			ell(c, cx, cy, rx, ry);
			if (plan !== 'crab') ell(c, cx - rx * 0.9, cy, rx * 0.7, ry * 0.8);
			markings(rx, ry);
			ell(c, cx + rx * 0.95 + hl, cy - ry * 0.1, rx * 0.5, ry * 0.7);
			if (plan === 'crab') {
				// claws rise with `wing` (+ = up), the forward one thrusts with the lunge
				const cr = C.wing * v * 0.15;
				c.fillStyle = dark;
				ell(c, cx - rx * 1.35, cy - ry * 1.2 - cr, rx * 0.4, ry * 0.7);
				ell(c, cx + rx * 1.35 + hl, cy - ry * 1.2 - cr, rx * 0.4, ry * 0.7);
			}
			if (a.t && /bee|wasp|fly|moth|dragon|beetle/.test(a.t) || (seed & 3) === 0) {
				c.fillStyle = 'rgba(220,240,255,0.55)';
				ell(c, cx - rx * 0.3, cy - ry * 1.5, rx * 0.8, ry * 0.7);
				ell(c, cx + rx * 0.3, cy - ry * 1.5, rx * 0.8, ry * 0.7);
			}
			c.fillStyle = eye;
			c.fillRect(cx + rx * 1.1 + hl, cy - ry * 0.3, Math.max(1, u * 0.05), Math.max(1, u * 0.05));
			break;
		}
		case 'serpent': {
			// undulation: the curve's control points swell and shrink out of phase (amplitude undAmp),
			// the head end rides the wave and thrusts forward with the lunge
			const und = C.undAmp;
			const ph = C.undPhase;
			c.lineWidth = lw * 1.1 * sz;
			c.lineCap = 'round';
			c.beginPath();
			const amp = v * 0.11 * sz;
			const ho = amp * und * Math.sin(ph + 3.2); // head offset (+ = down)
			c.moveTo(x + u * 0.08, cy + amp + amp * und * Math.sin(ph + 4.5));
			c.bezierCurveTo(x + u * 0.28, cy - amp * 2 * (1 + und * Math.sin(ph)), x + u * 0.5, cy + amp * 2 * (1 + und * Math.sin(ph + 1.6)), x + u * 0.74 + hl, cy - amp * 0.4 + ho);
			c.stroke();
			ell(c, x + u * 0.78 + hl, cy - amp * 0.5 + ho, u * 0.1 * sz, u * 0.075 * sz);
			c.fillStyle = eye;
			c.fillRect(x + u * 0.8 + hl, cy - amp * 0.7 + ho, Math.max(1, u * 0.04), Math.max(1, u * 0.04));
			c.strokeStyle = '#d83a3a';
			line(c, x + u * 0.86 + hl, cy - amp * 0.5 + ho, x + u * 0.96 + hl, cy - amp * 0.7 + ho, 1);
			break;
		}
		case 'ooze': {
			// squash and stretch is the frame scale above
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
				// sway: each tentacle's bend and tip drift sideways, phase-shifted along the row (+ = right)
				const sw = C.sway ? Math.sin(C.undPhase + k * 0.8) * C.sway * u : 0;
				c.strokeStyle = body;
				c.beginPath();
				c.moveTo(cx + k * u * 0.12, cy - v * 0.1);
				c.quadraticCurveTo(cx + k * u * 0.2 + ((seed >>> (k + 4)) & 3) * 2 + sw * 0.25, cy + v * 0.12, cx + k * u * 0.16 + sw * 0.15, ground);
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
			// legs swing about their tops (rotate(-d): + = foot right)
			const gA = C.legA * 0.6;
			const gB = C.legB * 0.6;
			swingAbout(c, gA, cx - bw / 2 + u * 0.04 + bw * 0.19, cy + v * 0.2);
			c.fillRect(cx - bw / 2 + u * 0.04, cy + v * 0.2, bw * 0.38, ground - cy - v * 0.2);
			if (gA) c.restore();
			swingAbout(c, gB, cx + bw * 0.1 + bw * 0.19, cy + v * 0.2);
			c.fillRect(cx + bw * 0.1, cy + v * 0.2, bw * 0.38, ground - cy - v * 0.2);
			if (gB) c.restore();
			if (C.stomp > 0.5) {
				// the foot comes down: two dust puffs at the ground beside the feet
				c.save();
				c.globalAlpha *= 0.55 * C.stomp;
				c.fillStyle = belly;
				ell(c, cx - bw / 2 - u * 0.04, ground - v * 0.015, u * 0.09, v * 0.025);
				ell(c, cx + bw / 2 + u * 0.04, ground - v * 0.015, u * 0.09, v * 0.025);
				c.restore();
			}
			c.fillStyle = body;
			c.fillRect(cx - u * 0.14, cy - v * 0.38, u * 0.28, v * 0.16);
			c.fillStyle = bgIdx === 22 ? '#ffff55' : '#ffb03a';
			c.fillRect(cx - u * 0.08, cy - v * 0.32, Math.max(1, u * 0.16), Math.max(1, v * 0.03));
			break;
		}
	}
	// cast: a soft glow over the body (alpha = cast envelope; nothing drawn otherwise)
	if (C.glow > 0) drawCastGlow(c, cx, cy, v * 0.5, '#9fd8ff', C.glow);
	c.restore(); // pose frame
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

/** Unified entry point for drawing either a person or creature sprite with animation and hit-flash support. */
export function drawActorSprite(c: Ctx, a: QudActor, g: string, fg: string, palette: string[], bgIdx: number, x: number, y: number, cw: number, chh: number) {
	if (a.s || g === '@') drawPersonSprite(c, a, fg, palette, bgIdx, x, y, cw, chh);
	else drawCreatureSprite2(c, a, g, fg, palette, bgIdx, x, y, cw, chh);
}
