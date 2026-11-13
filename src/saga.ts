/** The end-of-life saga (engine/saga.go): types, plain-text export, and the journey map drawn on a canvas. */

export interface SagaChapter {
	kind: string;
	title: string;
	text: string;
	day0: number;
	day1: number;
	t0: number;
	t1: number;
}
export interface SagaMarker {
	k: string;
	d: number;
	pl: number;
	q: number;
	t: string;
}
export interface SagaMap {
	planet: number;
	name: string;
	w: number;
	h: number;
	biome: string;
	height: string;
	towns: number[];
}
export interface SagaData {
	title: string;
	subtitle: string;
	chronicler: string;
	name: string;
	world: string;
	seed: string;
	dead: boolean;
	chapters: SagaChapter[];
	epitaph: string;
	recap: string;
	stats: { days: number; km: number; kills: number; deeds: number; level: number; score: number; years: number };
	track: number[];
	markers: SagaMarker[];
	maps: SagaMap[];
}

export const CHAPTER_COLORS = ['#e0c060', '#e07050', '#d070c0', '#70b0e0', '#70e0a0', '#f0f0a0', '#e09040', '#a0a0ff', '#ff8080', '#80e0e0'];

export function loadSaga(api: { saga(): string }): SagaData | null {
	try {
		const s = JSON.parse(api.saga());
		return s && Array.isArray(s.chapters) && s.chapters.length ? (s as SagaData) : null;
	} catch {
		return null;
	}
}

export function sagaStatsLine(s: SagaData): string {
	const t = s.stats;
	return `${t.days} days · ${t.km} km · ${t.kills} slain · ${t.deeds} deeds · level ${t.level}`;
}

export function sagaText(s: SagaData, withSeed = false): string {
	const parts = [s.title, s.subtitle, s.chronicler, ''];
	for (const c of s.chapters) parts.push(c.title.toUpperCase(), c.text, '');
	parts.push(s.epitaph.split('\n').map((l) => '  ' + l).join('\n'), '', sagaStatsLine(s));
	if (withSeed) parts.push(`World: ${s.world} · seed ${s.seed}`);
	return parts.join('\n');
}

const BIOME_COL: Record<number, [number, number, number]> = {
	0: [18, 40, 70], 1: [200, 170, 90], 2: [170, 160, 140], 3: [170, 160, 80], 4: [100, 160, 70], 5: [40, 120, 50], 6: [60, 130, 60],
	7: [30, 100, 60], 8: [40, 110, 80], 9: [60, 100, 80], 10: [150, 160, 150], 11: [230, 240, 245], 12: [70, 110, 100], 13: [90, 80, 80],
	14: [150, 120, 200], 15: [140, 90, 120], 16: [200, 80, 40]
};

/** Draw the journey over one of the saga's planet maps. Returns false when there is nothing to draw. */
export function drawSagaMap(canvas: HTMLCanvasElement, s: SagaData, mapIdx: number, cell = 8): boolean {
	const m = s.maps[mapIdx];
	const ctx = canvas.getContext('2d');
	if (!m || !ctx) return false;
	canvas.width = m.w * cell;
	canvas.height = m.h * cell;
	for (let i = 0; i < m.w * m.h; i++) {
		const b = m.biome.charCodeAt(i) - 97;
		const h = m.height.charCodeAt(i) - 97;
		let [r, g, bl] = BIOME_COL[b] ?? [90, 90, 90];
		if (b !== 0) {
			const k = 0.75 + Math.min(h, 25) * 0.025;
			r *= k;
			g *= k;
			bl *= k;
		}
		ctx.fillStyle = `rgb(${r | 0},${g | 0},${bl | 0})`;
		ctx.fillRect((i % m.w) * cell, Math.floor(i / m.w) * cell, cell, cell);
	}
	ctx.fillStyle = 'rgba(255,255,255,0.35)';
	for (const p of m.towns) ctx.fillRect((p % m.w) * cell + cell / 2 - 1, Math.floor(p / m.w) * cell + cell / 2 - 1, 2, 2);
	const colorAt = (idx: number) => {
		for (let i = 0; i < s.chapters.length; i++) {
			const c = s.chapters[i];
			if (c.kind !== 'prologue' && c.kind !== 'epilogue' && idx >= c.t0 && idx < c.t1) return CHAPTER_COLORS[i % CHAPTER_COLORS.length];
		}
		return CHAPTER_COLORS[0];
	};
	const n = s.track.length / 3;
	ctx.lineWidth = Math.max(2, cell / 3);
	ctx.lineJoin = ctx.lineCap = 'round';
	let prev: [number, number] | null = null;
	let first: [number, number] | null = null;
	let last: [number, number] | null = null;
	for (let i = 0; i < n; i++) {
		if (s.track[i * 3] !== m.planet) {
			prev = null;
			continue;
		}
		const q = s.track[i * 3 + 1];
		const pt: [number, number] = [(q % m.w) * cell + cell / 2, Math.floor(q / m.w) * cell + cell / 2];
		if (prev && Math.abs(prev[0] - pt[0]) < cell * 20 && Math.abs(prev[1] - pt[1]) < cell * 20) {
			ctx.strokeStyle = colorAt(i);
			ctx.beginPath();
			ctx.moveTo(prev[0], prev[1]);
			ctx.lineTo(pt[0], pt[1]);
			ctx.stroke();
		}
		prev = pt;
		first ??= pt;
		last = pt;
	}
	const glyph: Record<string, string> = { kill: '✕', discovery: '?', kinship: '♥', death: '†', companion: '†', rift: 'Ø', end: '★', endgame: '★', nemesis: '!', rival: '!', slay: '✕', relic: '◆' };
	ctx.font = `bold ${Math.round(cell * 1.5)}px monospace`;
	ctx.textAlign = 'center';
	ctx.textBaseline = 'middle';
	for (const mk of s.markers) {
		if (mk.pl !== m.planet || !glyph[mk.k]) continue;
		const x = (mk.q % m.w) * cell + cell / 2;
		const y = Math.floor(mk.q / m.w) * cell + cell / 2;
		ctx.fillStyle = '#000';
		ctx.fillText(glyph[mk.k], x + 1, y + 1);
		ctx.fillStyle = '#fff';
		ctx.fillText(glyph[mk.k], x, y);
	}
	if (first) {
		ctx.fillStyle = '#7f7';
		ctx.beginPath();
		ctx.arc(first[0], first[1], cell * 0.8, 0, 7);
		ctx.fill();
	}
	if (last) {
		ctx.fillStyle = s.dead ? '#f55' : '#fff';
		ctx.beginPath();
		ctx.arc(last[0], last[1], cell * 0.8, 0, 7);
		ctx.fill();
	}
	return true;
}

export function downloadBlob(blob: Blob, name: string) {
	const a = document.createElement('a');
	a.href = URL.createObjectURL(blob);
	a.download = name;
	a.click();
	setTimeout(() => URL.revokeObjectURL(a.href), 5000);
}

const slug = (s: string) => s.toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/^-|-$/g, '') || 'saga';
export const downloadSagaText = (s: SagaData) => downloadBlob(new Blob([sagaText(s, true)], { type: 'text/plain' }), `saga-${slug(s.name)}.txt`);
export function downloadSagaMap(canvas: HTMLCanvasElement, s: SagaData) {
	canvas.toBlob((b) => b && downloadBlob(b, `journey-${slug(s.name)}.png`));
}
