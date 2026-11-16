/**
 * Gamepad support (Gamepad API, "standard" mapping). Polls with requestAnimationFrame only while a pad is connected, the
 * module is started and the page is visible. Everything is turned into engine key names and handed to `send`, so the pad
 * goes through the same path as the keyboard (and bypasses key remapping on purpose).
 *
 *  left stick / d-pad   8-way move on the map, look, aim and world screens; 4-way in menus. Repeat after a delay.
 *  right stick          on the map: opens Look ('x') then moves the cursor; in look/aim: moves the cursor
 *  A  confirm           map: 'g' (pick up / interact); elsewhere Enter
 *  B  cancel            Escape, except on the map (where Escape would save and quit)
 *  X  inventory         map: 'i'; elsewhere Tab
 *  Y  character         map: 'c'
 *  LB                   map: look 'x'; look/aim: Tab (cycle target); menus: ArrowLeft
 *  RB                   map: fire 'f'; look/aim: 'f'; menus: ArrowRight
 *  LT palette ';'  RT world map 'w' (menus: PageUp / PageDown)
 *  Start                settings (opts.onSettings)     Select  journal 'j'
 *  L3                   wait '.'     R3  tap = auto-explore 'v'; hold = radial quick menu (stick picks, release runs it)
 */

export interface GamepadOptions {
	enabled?: boolean;
	/** stick deadzone 0.1..0.7 */
	deadzone?: number;
	/** ms between repeats while held */
	repeatMs?: number;
	/** rumble when HP drops */
	rumble?: boolean;
	/** Start button; without it Start sends '`' */
	onSettings?: () => void;
}

type SendFn = (key: string) => void;
type FrameFn = () => any;

const opt: Required<Omit<GamepadOptions, 'onSettings'>> & { onSettings?: () => void } = { enabled: true, deadzone: 0.3, repeatMs: 110, rumble: false };

let send: SendFn | null = null;
let getFrame: FrameFn | null = null;
let running = false;
let raf = 0;
let padIndex = -1;

const DIAG_MODES = new Set(['main', 'target', 'look', 'dir', 'world']);
const CURSOR_MODES = new Set(['target', 'look', 'dir']);
// order: E SE S SW W NW N NE (screen angles, y down)
const DIR8 = ['ArrowRight', 'N3', 'ArrowDown', 'N1', 'ArrowLeft', 'N7', 'ArrowUp', 'N9'];

const MENU: [string, string][] = [
	['Inventory', 'i'], ['Character', 'c'], ['Abilities', 'a'], ['Search', 's'],
	['Rest', 'R'], ['Eat', 'e'], ['Drink', 'd'], ['Throw', 't'],
];
// MENU index follows DIR8 order but starting at N: N NE E SE S SW W NW
const MENU_DIR = [6, 7, 0, 1, 2, 3, 4, 5];

interface Rep {
	dir: number; // -1 none
	next: number;
}
const repL: Rep = { dir: -1, next: 0 };
const repR: Rep = { dir: -1, next: 0 };
let prevBtn: boolean[] = [];
let r3Down = 0;
let menuEl: HTMLDivElement | null = null;
let menuSel = -1;
let lastHp = -1;
let lookOpened = false;

export function setGamepadOptions(o: GamepadOptions): void {
	if (typeof o.enabled === 'boolean') opt.enabled = o.enabled;
	if (typeof o.deadzone === 'number') opt.deadzone = Math.min(0.7, Math.max(0.1, o.deadzone));
	if (typeof o.repeatMs === 'number') opt.repeatMs = Math.min(400, Math.max(50, o.repeatMs));
	if (typeof o.rumble === 'boolean') opt.rumble = o.rumble;
	if ('onSettings' in o) opt.onSettings = o.onSettings;
	if (running) {
		if (!opt.enabled) halt();
		else wake();
	}
}

export function startGamepad(sendFn: SendFn, frameFn: FrameFn, o?: GamepadOptions): void {
	stopGamepad();
	send = sendFn;
	getFrame = frameFn;
	running = true;
	if (o) setGamepadOptions(o);
	window.addEventListener('gamepadconnected', onConnect);
	window.addEventListener('gamepaddisconnected', onDisconnect);
	document.addEventListener('visibilitychange', onVisible);
	window.addEventListener('blur', releaseAll);
	scan();
	wake();
}

export function stopGamepad(): void {
	running = false;
	window.removeEventListener('gamepadconnected', onConnect);
	window.removeEventListener('gamepaddisconnected', onDisconnect);
	document.removeEventListener('visibilitychange', onVisible);
	window.removeEventListener('blur', releaseAll);
	halt();
	send = null;
	getFrame = null;
	padIndex = -1;
}

function halt() {
	if (raf) cancelAnimationFrame(raf);
	raf = 0;
	releaseAll();
}

function releaseAll() {
	closeMenu();
	repL.dir = repR.dir = -1;
	prevBtn = [];
	r3Down = 0;
	lookOpened = false;
}

function scan() {
	padIndex = -1;
	try {
		const pads = navigator.getGamepads?.() ?? [];
		let fallback = -1;
		for (const p of pads) {
			if (!p || !p.connected) continue;
			if (p.mapping === 'standard') {
				padIndex = p.index;
				return;
			}
			if (fallback < 0) fallback = p.index;
		}
		padIndex = fallback;
	} catch {
		padIndex = -1;
	}
}

function onConnect() {
	scan();
	wake();
}
function onDisconnect() {
	scan();
	if (padIndex < 0) halt();
}
function onVisible() {
	if (document.hidden) halt();
	else wake();
}

function wake() {
	if (!running || !opt.enabled || document.hidden || padIndex < 0 || raf) return;
	raf = requestAnimationFrame(tick);
}

function tick(t: number) {
	raf = 0;
	if (!running || !opt.enabled || document.hidden) return;
	let pad: Gamepad | null | undefined = null;
	try {
		pad = navigator.getGamepads?.()[padIndex];
	} catch {
		pad = null;
	}
	if (!pad || !pad.connected) {
		scan();
		if (padIndex < 0) return;
	} else {
		poll(pad, t);
	}
	raf = requestAnimationFrame(tick);
}

// ---------- polling ----------

function dirOfVector(x: number, y: number, diag: boolean): number {
	const dz = opt.deadzone;
	if (Math.hypot(x, y) < dz) return -1;
	if (!diag) return Math.abs(x) > Math.abs(y) ? (x > 0 ? 0 : 4) : y > 0 ? 2 : 6;
	const ang = Math.atan2(y, x);
	return ((Math.round(ang / (Math.PI / 4)) % 8) + 8) % 8;
}

function repeat(r: Rep, dir: number, t: number, fire: (d: number) => void) {
	if (dir < 0) {
		r.dir = -1;
		return;
	}
	if (dir !== r.dir) {
		r.dir = dir;
		r.next = t + Math.max(opt.repeatMs * 3, 280);
		fire(dir);
	} else if (t >= r.next) {
		r.next = t + opt.repeatMs;
		fire(dir);
	}
}

function pressed(pad: Gamepad, i: number): boolean {
	const b = pad.buttons[i];
	return !!b && (b.pressed || b.value > 0.5);
}

function poll(pad: Gamepad, t: number) {
	const frame = getFrame?.();
	const mode: string = frame?.mode ?? 'main';
	const diag = DIAG_MODES.has(mode);
	const s = send;
	if (!s) return;

	rumbleCheck(pad, frame);

	// radial menu: owns the sticks while open
	if (menuEl) {
		const x = pad.axes[0] ?? 0;
		const y = pad.axes[1] ?? 0;
		const d = dirOfVector(x, y, true);
		menuSel = d < 0 ? -1 : MENU_DIR.indexOf(d);
		paintMenu();
		if (!pressed(pad, 11)) {
			const pick = menuSel >= 0 ? MENU[menuSel][1] : null;
			closeMenu();
			r3Down = 0;
			prevBtn[11] = false;
			if (pick) s(pick);
		}
		return;
	}

	// movement: d-pad wins over the left stick
	let dx = (pressed(pad, 15) ? 1 : 0) - (pressed(pad, 14) ? 1 : 0);
	let dy = (pressed(pad, 13) ? 1 : 0) - (pressed(pad, 12) ? 1 : 0);
	let ld: number;
	if (dx || dy) ld = dirOfVector(dx, dy, diag);
	else ld = dirOfVector(pad.axes[0] ?? 0, pad.axes[1] ?? 0, diag);
	repeat(repL, ld, t, (d) => s(DIR8[d]));

	// right stick: cursor (opens Look from the map)
	const rd = dirOfVector(pad.axes[2] ?? 0, pad.axes[3] ?? 0, diag);
	if (rd < 0) lookOpened = false;
	if (mode === 'main') {
		if (rd >= 0 && !lookOpened) {
			lookOpened = true;
			s('x');
		}
		repR.dir = -1;
	} else {
		repeat(repR, rd, t, (d) => s(DIR8[d]));
	}

	// buttons (edge triggered)
	const now: boolean[] = [];
	for (let i = 0; i < 16; i++) now[i] = pressed(pad, i);
	const edge = (i: number) => now[i] && !prevBtn[i];
	const main = mode === 'main';
	const cursor = CURSOR_MODES.has(mode);
	if (edge(0)) s(main ? 'g' : 'Enter');
	if (edge(1) && !main) s('Escape');
	if (edge(2)) s(main ? 'i' : 'Tab');
	if (edge(3) && main) s('c');
	if (edge(4)) s(main ? 'x' : cursor ? 'Tab' : 'ArrowLeft');
	if (edge(5)) s(main || cursor ? 'f' : 'ArrowRight');
	if (edge(6)) s(main ? ';' : 'PageUp');
	if (edge(7)) s(main ? 'w' : 'PageDown');
	if (edge(8)) s('j');
	if (edge(9)) {
		if (opt.onSettings) opt.onSettings();
		else s('`');
	}
	if (edge(10)) s('.');
	if (edge(11)) r3Down = t;
	if (now[11] && r3Down && main && t - r3Down > 320) {
		openMenu();
		r3Down = 0;
	}
	if (!now[11] && prevBtn[11] && r3Down) {
		if (main && t - r3Down <= 320) s('v');
		r3Down = 0;
	}
	prevBtn = now;
}

// ---------- rumble ----------

function rumbleCheck(pad: Gamepad, frame: any) {
	const hp = frame?.music?.hp;
	if (typeof hp !== 'number') return;
	if (lastHp >= 0 && hp < lastHp - 0.001 && opt.rumble) {
		const act = (pad as any).vibrationActuator;
		const drop = lastHp - hp;
		try {
			void act?.playEffect?.('dual-rumble', {
				startDelay: 0,
				duration: hp <= 0 ? 500 : 140,
				weakMagnitude: 0.35,
				strongMagnitude: Math.min(1, 0.25 + drop * 2),
			});
		} catch {
			// no haptics on this pad
		}
	}
	lastHp = hp;
}

// ---------- quick menu ----------

function openMenu() {
	if (menuEl) return;
	const el = document.createElement('div');
	el.setAttribute('aria-hidden', 'true');
	el.style.cssText = 'position:fixed;left:50%;top:50%;width:300px;height:300px;margin:-150px 0 0 -150px;z-index:90;pointer-events:none;border-radius:50%;background:rgba(4,19,18,.82);border:1px solid #40a4b9;font:13px "Share Tech Mono",monospace;color:#cfeaea';
	MENU.forEach(([label], i) => {
		const ang = (MENU_DIR[i] * Math.PI) / 4;
		const it = document.createElement('div');
		it.dataset.i = String(i);
		it.textContent = label;
		it.style.cssText = `position:absolute;left:${150 + Math.cos(ang) * 100}px;top:${150 + Math.sin(ang) * 100}px;transform:translate(-50%,-50%);padding:2px 7px;border:1px solid #155352;background:#0d2b29;white-space:nowrap`;
		el.appendChild(it);
	});
	document.body.appendChild(el);
	menuEl = el;
	menuSel = -1;
}

function paintMenu() {
	if (!menuEl) return;
	for (const c of Array.from(menuEl.children) as HTMLElement[]) {
		const on = Number(c.dataset.i) === menuSel;
		c.style.color = on ? '#041312' : '';
		c.style.background = on ? '#cfc041' : '#0d2b29';
	}
}

function closeMenu() {
	menuEl?.remove();
	menuEl = null;
	menuSel = -1;
}
