// Gzip the bundled worlds in dist/worlds (run after vite build): <name>.json -> <name>.json.gz, the plain file
// stays as the fallback. Skips index.json. Usage: node scripts/gzip-worlds.mjs [dir]
import { readdirSync, readFileSync, writeFileSync } from 'node:fs';
import { gzipSync } from 'node:zlib';
import { join } from 'node:path';

const dir = process.argv[2] || 'dist/worlds';
let saved = 0;
for (const f of readdirSync(dir)) {
	if (!f.endsWith('.json') || f === 'index.json') continue;
	const raw = readFileSync(join(dir, f));
	const gz = gzipSync(raw, { level: 9 });
	writeFileSync(join(dir, f + '.gz'), gz);
	saved += raw.length - gz.length;
	console.log(`${f}: ${(raw.length / 1e6).toFixed(1)} MB -> ${(gz.length / 1e6).toFixed(1)} MB`);
}
console.log(`saved ${(saved / 1e6).toFixed(1)} MB of download`);
