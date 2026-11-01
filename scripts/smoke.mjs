#!/usr/bin/env node
// Browser smoke test: build, serve dist/ with `vite preview`, then drive the real page in headless Chromium.
// Title → bundled sample world → CharGen → Play → new screens (cook, companions, journal tabs, char sheet tabs,
// settings + every music style) → walk → save & quit → reload → continue. Fails on any page/console error,
// a blank canvas or a dead window.qud.
// Usage: npm run smoke [-- --no-build] [--port 5197] [--headed]
import { spawn, execFileSync } from 'node:child_process';
import { createRequire } from 'node:module';
import { existsSync, mkdirSync, copyFileSync } from 'node:fs';
import { dirname, join, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';
import net from 'node:net';

const root = resolve(dirname(fileURLToPath(import.meta.url)), '..');
const args = process.argv.slice(2);
const flag = (f) => args.includes(f);
const opt = (f, d) => (args.includes(f) ? args[args.indexOf(f) + 1] : d);
const t0 = Date.now();
const log = (...a) => console.log(`[smoke ${((Date.now() - t0) / 1000).toFixed(1)}s]`, ...a);

function loadPlaywright() {
	const req = createRequire(join(root, 'package.json'));
	for (const p of ['playwright', '@playwright/test', '/opt/node-tools/node_modules/playwright']) {
		try {
			return req(p);
		} catch {}
	}
	throw new Error('playwright not found (expected a global install)');
}

function freePort(want) {
	return new Promise((res) => {
		const s = net.createServer();
		s.once('error', () => {
			const t = net.createServer();
			t.listen(0, '127.0.0.1', () => {
				const p = t.address().port;
				t.close(() => res(p));
			});
		});
		s.listen(want, '127.0.0.1', () => s.close(() => res(want)));
	});
}

function build() {
	log('vite build');
	execFileSync(process.execPath, [join(root, 'node_modules/vite/bin/vite.js'), 'build', '--logLevel', 'warn'], { cwd: root, stdio: 'inherit' });
	// the wasm goes straight into dist/ so the committed public/wasm/qud.wasm is never touched
	log('go build wasm → dist/wasm');
	mkdirSync(join(root, 'dist/wasm'), { recursive: true });
	execFileSync('go', ['build', '-ldflags=-s -w', '-o', 'dist/wasm/qud.wasm', './cmd/wasm'], { cwd: root, stdio: 'inherit', env: { ...process.env, GOOS: 'js', GOARCH: 'wasm' } });
	const goroot = execFileSync('go', ['env', 'GOROOT'], { encoding: 'utf8' }).trim();
	const exec = [join(goroot, 'lib/wasm/wasm_exec.js'), join(goroot, 'misc/wasm/wasm_exec.js')].find(existsSync);
	if (!exec) throw new Error('wasm_exec.js not found in GOROOT');
	copyFileSync(exec, join(root, 'dist/wasm/wasm_exec.js'));
}

async function waitHttp(url, ms = 20000) {
	const end = Date.now() + ms;
	while (Date.now() < end) {
		try {
			if ((await fetch(url)).ok) return;
		} catch {}
		await new Promise((r) => setTimeout(r, 150));
	}
	throw new Error(`server did not come up at ${url}`);
}

const fails = [];
const check = (ok, what) => {
	if (!ok) fails.push(what);
	log(ok ? 'ok  ' : 'FAIL', what);
};

async function main() {
	if (!flag('--no-build')) build();
	if (!existsSync(join(root, 'dist/index.html'))) throw new Error('dist/ missing: run without --no-build');
	const port = await freePort(Number(opt('--port', 5197)));
	const server = spawn(process.execPath, [join(root, 'node_modules/vite/bin/vite.js'), 'preview', '--port', String(port), '--strictPort', '--host', '127.0.0.1'], { cwd: root, stdio: ['ignore', 'pipe', 'pipe'] });
	let serverOut = '';
	server.stdout.on('data', (d) => (serverOut += d));
	server.stderr.on('data', (d) => (serverOut += d));
	const base = `http://127.0.0.1:${port}/`;
	const { chromium } = loadPlaywright();
	let browser;
	try {
		await waitHttp(base);
		log('preview on', base);
		browser = await chromium.launch({ headless: !flag('--headed'), args: ['--autoplay-policy=no-user-gesture-required'] });
		const page = await browser.newPage({ viewport: { width: 1400, height: 900 } });
		const errors = [];
		page.on('pageerror', (e) => errors.push(`pageerror: ${e.message}`));
		page.on('console', (m) => m.type() === 'error' && errors.push(`console: ${m.text()}`));
		page.on('requestfailed', (r) => errors.push(`requestfailed: ${r.url()} ${r.failure()?.errorText}`));
		page.on('dialog', (d) => d.accept());

		const frame = () => page.evaluate(() => window.qud?.frame?.() ?? null);
		const mode = async () => (await frame())?.mode;
		const press = async (...keys) => {
			for (const k of keys) await page.keyboard.press(k);
		};
		/** distinct colours on a grid of canvas samples; a blank canvas has 1 */
		const canvasColours = () =>
			page.evaluate(() => {
				const c = document.querySelector('canvas');
				const ctx = c?.getContext('2d');
				if (!c || !ctx || !c.width) return 0;
				const seen = new Set();
				const d = ctx.getImageData(0, 0, c.width, c.height).data;
				for (let y = 0; y < c.height; y += 7) for (let x = 0; x < c.width; x += 5) {
					const i = (y * c.width + x) * 4;
					seen.add((d[i] << 16) | (d[i + 1] << 8) | d[i + 2]);
				}
				return seen.size;
			});
		const escapeToMain = async () => {
			for (let i = 0; i < 4 && (await mode()) !== 'main'; i++) await press('Escape');
			return mode();
		};

		// title
		await page.goto(base);
		await page.locator('h1', { hasText: 'Wanderer' }).waitFor();
		const cards = page.locator('section .grid button.card');
		await cards.first().waitFor();
		await page.waitForFunction(() => typeof window.qud?.init === 'function', null, { timeout: 30000 });
		check((await cards.count()) >= 1, `title lists bundled sample worlds (${await cards.count()})`);

		// every bundled world parses in the engine
		const bad = await page.evaluate(async () => {
			const out = [];
			const list = await (await fetch('worlds/index.json')).json();
			for (const s of list) {
				const r = window.qud ?? null;
				if (!r) return ['engine not loaded'];
				const o = r.init(await (await fetch('worlds/' + s.file)).text());
				if (typeof o !== 'string') out.push(`${s.file}: ${o.error}`);
				else if (!JSON.parse(o).starts?.length) out.push(`${s.file}: no starts`);
			}
			return out;
		});
		check(bad.length === 0, `qud.init() accepts every bundled world ${bad.join('; ')}`);

		// chargen
		await cards.first().click();
		await page.locator('.cg').waitFor({ timeout: 30000 });
		check(await page.evaluate(() => typeof window.qud?.init === 'function'), 'window.qud API loaded');
		for (let i = 0; i < 10; i++) {
			const go = page.locator('.cg button.go');
			if ((await go.textContent())?.includes('Begin')) break;
			await go.click();
		}
		await page.locator('.cg input[maxlength="24"]').fill('Smoketest');
		await page.locator('.cg button.go', { hasText: 'Begin' }).click();
		await page.locator('canvas[aria-label="Game screen"]').waitFor();
		await page.waitForTimeout(300);
		check((await canvasColours()) > 3, `play canvas drawn (${await canvasColours()} colours)`);
		const meta = JSON.parse(await page.evaluate(() => window.qud.meta()));
		check(meta?.name === 'Smoketest' && !!meta.seed, `qud.meta() → ${meta?.name} seed ${meta?.seed}`);
		check((await escapeToMain()) === 'main', 'reached main mode');

		// the new screens
		await press('C');
		log('  C →', await mode());
		await escapeToMain();
		await press('o');
		check((await mode()) === 'comp' || (await mode()) === 'main', `o companions (${await mode()})`);
		await escapeToMain();
		await press('j');
		check((await mode()) === 'journal', 'j opens the journal');
		const seenTabs = new Set();
		for (let i = 0; i < 10; i++) {
			await press('Tab', 'ArrowDown', 'PageDown');
			const g = (await frame())?.g ?? '';
			for (const t of ['Cookbook', 'Calendar', 'Bestiary', 'Chronicle']) if (g.includes(t)) seenTabs.add(t);
		}
		check(seenTabs.has('Cookbook') && seenTabs.has('Calendar'), `journal tabs drawn (${[...seenTabs].join(', ')})`);
		await escapeToMain();
		await press('c');
		check((await mode()) === 'char', 'c opens the character sheet');
		for (let i = 0; i < 4; i++) {
			await press('Tab', 'ArrowDown');
			check((await mode()) === 'char', `char sheet tab ${i + 1}`);
		}
		await escapeToMain();
		check((await canvasColours()) > 3, 'canvas still drawn after screens');

		// settings: open, every music style, close
		await page.locator('button.gear').click();
		const dlg = page.locator('[role="dialog"]');
		await dlg.waitFor();
		const musicBox = dlg.locator('input[type="checkbox"]').first();
		if (!(await musicBox.isChecked())) await musicBox.check();
		const style = dlg.locator('label.row', { hasText: 'Music style' }).locator('select');
		const styles = await style.locator('option').evaluateAll((os) => os.map((o) => o.value));
		for (const s of styles) {
			await style.selectOption(s);
			await page.waitForTimeout(120);
		}
		check(styles.length >= 2, `music styles cycled: ${styles.join(', ')}`);
		const flav = dlg.locator('label.row', { hasText: 'Instruments' }).locator('select');
		if (await flav.count()) for (const v of await flav.locator('option').evaluateAll((os) => os.map((o) => o.value))) await flav.selectOption(v);
		await dlg.locator('button.close').click();
		await dlg.waitFor({ state: 'detached' });
		await page.locator('button.gear').click();
		await dlg.waitFor();
		await press('Escape');
		await dlg.waitFor({ state: 'detached' });
		check(true, 'settings open/close (button and Escape)');

		// walk
		for (const k of ['ArrowRight', 'ArrowRight', 'ArrowDown', 'ArrowDown', 'ArrowLeft', 'ArrowUp', 'Numpad6', 'Numpad3', 'Numpad8', '.', 'ArrowRight', 'ArrowRight']) {
			await press(k);
			if ((await mode()) !== 'main') await escapeToMain();
		}
		const before = JSON.parse(await page.evaluate(() => window.qud.meta()));
		const saveLen = await page.evaluate(() => window.qud.save().length);
		check(saveLen > 100, `qud.save() returns a save (${saveLen} bytes)`);

		// save & quit → reload → continue
		await page.locator('button.quit').click();
		await page.locator('h2', { hasText: 'Continue' }).waitFor();
		await page.reload();
		const cont = page.locator('.run button.main', { hasText: 'Smoketest' });
		await cont.waitFor();
		await cont.click();
		await page.locator('canvas[aria-label="Game screen"]').waitFor({ timeout: 30000 });
		await page.waitForTimeout(300);
		const after = JSON.parse(await page.evaluate(() => window.qud.meta()));
		check(after.name === 'Smoketest' && after.seed === before.seed && after.place === before.place, `continued run (${after.name} at ${after.place})`);
		check((await canvasColours()) > 3, 'canvas drawn after continue');
		await press('ArrowRight', 'j', 'Escape');
		check((await mode()) === 'main', 'engine answers keys after load');

		const real = errors.filter((e) => !/favicon/.test(e));
		for (const e of real) log('  ', e);
		check(real.length === 0, `no page/console errors (${real.length})`);
	} finally {
		await browser?.close();
		server.kill();
	}
	if (fails.length) {
		console.error(`\nsmoke FAILED (${fails.length}):\n - ${fails.join('\n - ')}`);
		if (serverOut.includes('rror')) console.error(serverOut);
		process.exit(1);
	}
	log(`smoke ok in ${((Date.now() - t0) / 1000).toFixed(1)}s`);
}

main().catch((e) => {
	console.error('smoke crashed:', e);
	process.exit(1);
});
