// Creatures by body plan (quadruped, saurian, avian, fish, insect/arachnid/crab, serpent, ooze, tentacled, golem)
// as hand-authored pixel art. Data first: BASES holds the silhouettes (two frames each), KINDS maps a creature
// kind onto a base plus palette, eye colour and small overlays, and classify() picks the kind from the actor's
// template, name, bestiary group, body plan or glyph. Size z (1..7) shrinks the art toward its target width.
//
// Authoring alphabet (on top of the PALETTE in types.ts). Art is drawn WITHOUT the outline: finish() adds a
// 1px 'o' outline around every filled pixel after shrinking, so small and large versions both stay crisp.
//   *  eye (replaced by the kind's eye colour)       +  tooth, horn, claw, bone (white x)
//   %  marking: accent a when this individual is marked, else the base F
//   &  dark marking: A when marked, else d             !  glow core (fire y, frost c, acid v ...)
import type { Rows, PCtx } from './types';
import type { QudActor } from '../actorsprites';
import { PW, PH, flipX } from './types';

export interface CreaturePixels {
	rows: Rows;
	pal: Pick<PCtx, 'skin' | 'hair' | 'accent'>;
}

// ---------------------------------------------------------------- authoring helpers

type Patch = Record<number, string>;
interface Base {
	f: [string[], string[]]; // two animation frames, PW x PH, unoutlined
	fly: boolean; // airborne or swimming: shrinks about its centre, not its feet
	top: number; // first sprite row of the authored art (overlays use the authored row index)
	sz: number; // the bestiary size it is drawn at: smaller creatures shrink, larger keep it
}

const row12 = (r: string) => r.padEnd(PW, '.').slice(0, PW);

/**
 * a base silhouette. `rows` top to bottom; it lands so its last row is sprite row 18 (outline on 19),
 * or starts at `top` when given (floaters). `p1` replaces whole rows (by authored index) for frame 1;
 * bob1 lifts frame 1 by a pixel (idle breathing, hovering).
 */
function def(rows: string[], p1: Patch = {}, o: { top?: number; fly?: boolean; bob1?: boolean } = {}): Omit<Base, 'sz'> {
	const top = o.top ?? PH - 1 - rows.length;
	const place = (src: string[], dy: number) => {
		const out = Array.from({ length: PH }, () => '.'.repeat(PW));
		src.forEach((r, i) => {
			const y = top + i + dy;
			if (y >= 0 && y < PH) out[y] = row12(r);
		});
		return out;
	};
	const r1 = rows.map((r, i) => p1[i] ?? r);
	return { f: [place(rows, 0), place(r1, o.bob1 ? -1 : 0)], fly: !!o.fly, top };
}

/** overlay authored rows onto a frame: '.' keeps, ' ' erases, anything else paints */
function overlay(fr: string[], ov: Patch | undefined, top: number): string[] {
	if (!ov) return fr;
	const out = fr.map((r) => r.split(''));
	for (const k in ov) {
		const y = top + +k;
		if (y < 0 || y >= PH) continue;
		const s = ov[k];
		for (let x = 0; x < s.length && x < PW; x++) {
			if (s[x] === '.') continue;
			out[y][x] = s[x] === ' ' ? '.' : s[x];
		}
	}
	return out.map((r) => r.join(''));
}

// ---------------------------------------------------------------- the silhouettes (face right)

const ART: Record<string, Omit<Base, 'sz'>> = {
	// ---- mammals
	canine: def([
		'........L.d.',
		'........lFd.',
		'.L.....lF*Fz',
		'.lF...lFFkk.',
		'..lLLLlFFk..',
		'..FF%%%FFk..',
		'..dFFFFFFd..',
		'..dKkkkkKd..',
		'..Fd....dF..',
		'..Fd....dF..',
		'..DD....DD..',
	], { 2: '.......lF*Fz', 3: 'LlF...lFFkk.', 8: '..F.d..d.F..', 9: '.F...d.d..F.', 10: '.D...D.D..D.' }),
	feline: def([
		'.L......L.d.',
		'l.......lFFd',
		'l......lF*FF',
		'.l.....FFkkz',
		'..lLLLlFFk..',
		'..FF%F%FFk..',
		'..dF%F%FFd..',
		'..dKkkkkKd..',
		'..Fd....dF..',
		'..DD....DD..',
	], { 0: 'L.......L.d.', 1: '.l......lFFd', 8: '.F..d..d..F.', 9: '.D..D..D..D.' }),
	bovine: def([
		'.....lLl....',
		'....lLLFl.+.',
		'...lLFFFFl+.',
		'..lFFFFFFFFF',
		'.lFFF%FFFF*F',
		'.FFFF%FFFFFF',
		'.dFFFFFFFdFz',
		'ddFFFFFFFdhh',
		'd.dkkkkkkdh.',
		'..dKKKKKKd..',
		'..Fd...dF...',
		'..Fd...dF...',
		'..DD...DD...',
	], { 7: '.dFFFFFFFdhh', 8: 'd.dkkkkkkdh.', 11: '.F..d.d..F..', 12: '.D..D.D..D..' }),
	deer: def([
		'.......+.+..',
		'........++.+',
		'........l++.',
		'........lFF.',
		'.......lF*Fd',
		'.......FFkkz',
		'.j....lFFk..',
		'.lLLLLlFFk..',
		'.FFF%F%FFk..',
		'.dFFFFFFFd..',
		'..dkkkkkKd..',
		'..Fd...dF...',
		'..Fd...dF...',
		'..Fd...dF...',
		'..DD...DD...',
	], { 12: '..F.d.d.F...', 13: '.F..d.d..F..', 14: '.D..D.D..D..' }),
	rodent: def([
		'.......lL...',
		'.....lLFFF..',
		'....lFFFF*Fq',
		'...lFF%FFkk.',
		'.l.FFFFFkk..',
		'l.ldKkkKd...',
		'.....D..D...',
	], { 4: 'l..FFFFFkk..', 5: '.lldKkkKd...', 6: '....D...D...' }),
	bear: def([
		'..lLLl......',
		'.lLFFFFl....',
		'lLFFFFFFl.l.',
		'FFFFFFFFlLF.',
		'FFF%FFFFF*kz',
		'dFFFFFFFdFk.',
		'dFFFFFFFFd..',
		'.dKkkkkKFd..',
		'.Fd....dF...',
		'.Fd....dF...',
		'DDD...DDD...',
	], { 0: '............', 1: '..lLLl......', 2: '.lLFFFFl..l.', 8: 'Fd......dF..', 9: 'Fd......dF..', 10: 'DD.....DDD..' }),
	horse: def([
		'.........L..',
		'........hlF.',
		'.......hlF*.',
		'.......hFFFl',
		'......hFFdFF',
		'......hFF.kz',
		'.hlLLLlFFd..',
		'hhFFFFFFFF..',
		'h.FF%FFFFF..',
		'h.dFFFFFFd..',
		'h.dkkkkkKd..',
		'..Fd....dF..',
		'..Fd....dF..',
		'..Fd....dF..',
		'..Fd....dF..',
		'..DD....DD..',
	], { 7: '.hFFFFFFFF..', 8: 'hhFF%FFFFF..', 9: 'h.dFFFFFFd..', 10: '..dkkkkkKd..', 13: '..F.d..d.F..', 14: '.F..d..d..F.', 15: '.D..D..D..D.' }),
	boar: def([
		'...HhHhH....',
		'.lLLLLLLl...',
		'lFFFFFFFFlL.',
		'FFF%FFFFFF*F',
		'dFFFFFFFFFFq',
		'dFFFFFFFdF+.',
		'.dKkkkKdd...',
		'.Fd...dF....',
		'.DD...DD....',
	], { 0: '..HhHhH.....', 7: 'Fd.....dF...', 8: 'D.D...D.D...' }),
	bat: def([
		'L..........L',
		'Fl..l..l..lF',
		'dFl.lFFl.lFd',
		'ddFlF**FlFdd',
		'.ddDFFFFDdd.',
		'..d.dFFd.d..',
		'.....DD.....',
		'............',
	], {
		0: '............', 1: '....l..l....', 2: '....lFFl....', 3: '..llF**Fll..',
		4: '.lFdFFFFdFl.', 5: 'lFddDFFDddFl', 6: 'Fdd..DD..ddF', 7: 'd..........d',
	}, { top: 4, fly: true }),
	elephant: def([
		'..lLLLLl....',
		'.lLFFFFFlLl.',
		'lLFFFFFdlFFl',
		'FFFFFFdFdF*F',
		'FFF%FFdFdFFF',
		'FFFFFFdFDFFF',
		'dFFFFFFdDdFF',
		'dFFFFFFFFF+F',
		'dFFFFFFFFd+F',
		'.dkkkkkkkd.F',
		'.FFd..FFd..d',
		'.FFd..FFd...',
		'.FFd..FFd...',
		'.DDD..DDD...',
	], { 7: 'dFFFFFFFFF+F', 8: 'dFFFFFFFFd+.', 9: '.dkkkkkkkdF.', 10: '.FFd..FFdd..', 13: '.DDD..DDD...' }),
	brute: def([
		'....lLLl....',
		'...lLFFFl...',
		'...lF*F*F...',
		'..lLFkkkF...',
		'.lLFFFFFFFl.',
		'lFFFFFFFFFFl',
		'FFFkkkkkkFFF',
		'FFdkkkkkkdFF',
		'Fd.FFFFFF.dF',
		'Fd.dFFFFd.dF',
		'kk.dF..Fd.kk',
		'...FF..FF...',
		'...DD..DD...',
	], { 10: 'Fd.dF..Fd.dF', 11: 'kk.FF..FF.kk', 12: '...DD..DD...' }, { bob1: true }),

	// ---- birds
	smallbird: def([
		'.....lLl....',
		'.....lF*y...',
		'...lLFFkk...',
		'..lFFdFkk...',
		'.dFddFkk....',
		'dd..dKk.....',
		'.....Y.Y....',
	], { 4: '.dFdlFkk....', 3: '..lFFFdkk...', 5: 'd...dKk.....' }, { bob1: true }),
	raptorbird: def([
		'...L........',
		'...Fl.......',
		'...dFl......',
		'...ddFl.....',
		'....ddFl.lL.',
		'..dddFFFFF*y',
		'dddFFFFFFkk.',
		'.dd.dkkkk...',
		'.....Y..Y...',
		'............',
		'............',
	], {
		0: '............', 1: '............', 2: '............', 3: '............',
		4: '.........lL.', 5: '..dddFFFFF*y', 6: 'dddFFFFFFkk.', 7: '.dd.dFFFk...',
		8: '....dFFlY...', 9: '....dFl.....', 10: '....Dd......',
	}, { top: 3, fly: true }),
	bigbird: def([
		'.........lL.',
		'.........F*y',
		'.........F..',
		'.........F..',
		'........lF..',
		'...hlLLlF...',
		'.hhFFFFFF...',
		'hhFFF%FFF...',
		'.hdFFFFFd...',
		'...dFFFd....',
		'....k.k.....',
		'....k.k.....',
		'....k..k....',
		'....k..k....',
		'...kK..kK...',
	], { 12: '....k..k....', 13: '...k...k....', 14: '..kK...kK...' }),

	// ---- reptiles and amphibians
	lizard: def([
		'........lLl.',
		'...lLLlLFF*F',
		'lllFF%F%FFkk',
		'...dFFFFFdk.',
		'..Fd....Fd..',
		'.D.......D..',
	], { 2: '.llFF%F%FFkk', 1: 'l..lLLlLFF*F', 4: '...Fd..Fd...', 5: '...D....D...' }),
	snake: def([
		'......lLLl..',
		'......F*FFr.',
		'......dFk..r',
		'.......Fk...',
		'......lFk...',
		'..lLLlFFd...',
		'.lFFFFFFFFl.',
		'lFF%F%F%FFFl',
		'dFFFFFFFFFFd',
		'.ddKKKKKKdd.',
	], { 0: '.....lLLl...', 1: '.....F*FF...', 2: '.....dFk....', 3: '......lFk...' }),
	croc: def([
		'..l.l.l.....',
		'.lLlLlLlL...',
		'lFFFFFFFFlLl',
		'FFF%F%FFFF*F',
		'dFFFFFFFF+x+',
		'ddKkkkkKdFFF',
		'..Fd..dF....',
		'.DD..DD.....',
	], { 4: 'dFFFFFFFFF.+', 5: 'ddKkkkkKdk+F', 6: '.Fd....dF...', 7: 'DD....DD....' }),
	turtle: def([
		'...lLLLl....',
		'..lL%FF%l...',
		'.lF%FFF%Fl..',
		'lF%FFFFF%Fjj',
		'dDDDDDDDDDk*',
		'.kK.....kKkk',
		'.KK.....KK..',
	], { 3: 'lF%FFFFF%Fj.', 4: 'dDDDDDDDDDjk', 5: '.kK.....kKk*', 6: 'KK.......KK.' }),
	frog: def([
		'..lL....Ll..',
		'..L*l..l*L..',
		'.lFFFFFFFFl.',
		'.FFkkkkkkFF.',
		'dFFF%FF%FFFd',
		'dDFFFFFFFFDd',
		'DD.kK..Kk.DD',
	], { 4: 'dFFF%FF%FFFd', 5: '.dFFFFFFFFd.', 6: 'DDdkK..KkdDD' }, { bob1: true }),

	// ---- dinosaurs
	raptor: def([
		'.........lL.',
		'........lF*F',
		'........FFkx',
		'lL....lLFk..',
		'FFllLlFFFk..',
		'.ddFF%FFFd..',
		'....dFFFdk..',
		'.....FdF....',
		'.....F..F...',
		'.....F...F..',
		'....+D...D+.',
	], { 3: '......lLFk..', 4: 'lLllLlFFFk..', 5: 'FddFF%FFFd..', 8: '.....F.F....', 9: '....F..F....', 10: '...+D..D+...' }),
	theropod: def([
		'.......lLLl.',
		'......lLFFFl',
		'......FF*FFF',
		'......FFFFFF',
		'......d+x+x+',
		'.....lFkkkkd',
		'L...lLFFkD..',
		'Fl.lFFFFk.k.',
		'dFFFF%FFkk..',
		'.ddFFFFFFk..',
		'...dFFFFFd..',
		'....dFFFFd..',
		'....dFFdFF..',
		'....dF..FF..',
		'....dF...F..',
		'...DDF...F..',
		'.......DDD..',
	], { 4: '......dFFFFF', 5: '.....lF+x+xd', 6: 'L...lLFkkkd.', 7: 'Fl.lFFFFkD..', 14: '....dF..F...', 15: '....dF..F...', 16: '...DD..DD...' }),
	sauropod: def([
		'.........lL.',
		'.........F*F',
		'.........F..',
		'........lF..',
		'........FF..',
		'.......lF...',
		'.......FF...',
		'......lFF...',
		'...lLLlFF...',
		'.lLFFFFFFF..',
		'lFFF%FFFFFF.',
		'FFFF%FFFFFF.',
		'dFFFFFFFFFd.',
		'd.dkkkkkkdd.',
		'..FFd..FFd..',
		'..FFd..FFd..',
		'..FFd..FFd..',
		'..DDD..DDD..',
	], { 0: '..........lL', 1: '.........lF*', 2: '.........F..', 12: 'dFFFFFFFFFd.', 13: '.ddkkkkkkdd.' }),
	armoured: def([
		'....a.a.....',
		'...aAaAa....',
		'..lAlAlAl...',
		'.lLLLLLLLl..',
		'lFFFFFFFFFl.',
		'FF%FF%FFFFFl',
		'dFFFFFFFFF*F',
		'd+dFFFFFFdFk',
		'+.dkkkkkd...',
		'..FFd.FFd...',
		'..FFd.FFd...',
		'..DDD.DDD...',
	], { 7: 'ddFFFFFFFdFk', 8: '++dkkkkkd...', 10: '.FFd...FFd..', 11: '.DDD...DDD..' }),

	// ---- sea
	fish: def([
		'.....lL.....',
		'L..lLFFFl...',
		'Fl.FF%FFF*F.',
		'FFFFF%FFFFFz',
		'Fd.dkkkkkkd.',
		'd...dKKkd...',
		'......d.....',
	], { 1: '...lLFFFl...', 2: 'L.lFF%FFF*F.', 3: 'FFFFF%FFFFFz', 4: 'd.ddkkkkkkd.', 5: '....dKKkd...' }, { top: 8, fly: true }),
	shark: def([
		'......L.....',
		'.....lF.....',
		'L...lLFl....',
		'Fl.lFFFFFFl.',
		'FFFFF%FFF*Fl',
		'Fd.dkkkk+x+k',
		'd...dkkkkkk.',
		'......dK....',
	], { 2: '....lLFl....', 3: 'L..lFFFFFFl.', 4: 'FFFFF%FFF*Fl', 5: 'd.ddkkkkkkkk', 6: '....dkk+x+k.' }, { top: 7, fly: true }),
	eel: def([
		'.lL....lLl..',
		'lFFl..lFF*F.',
		'd..dFFd...k.',
		'....dd......',
	], { 0: '....lL...lL.', 1: '.lLlFFl.lF*F', 2: 'lFd..dFFd.k.', 3: 'd...........' }, { top: 10, fly: true }),
	squid: def([
		'....lLLl....',
		'...lLFFFl...',
		'...LFFFFFl..',
		'...FF%FFFF..',
		'...F*FF*FF..',
		'..lFFFFFFFl.',
		'.lF.dFdFd.Fl',
		'lF..F.F.F..F',
		'F..lF.F.Fl.d',
		'd..F..F..F..',
		'...d.dD..d..',
	], { 7: '.F..F.F.F.F.', 8: 'F..F..F..F.d', 9: 'd..lF.F.Fl..', 10: '..d..dD.d...' }, { top: 5, fly: true }),
	crab: def([
		'.l........l.',
		'lFl......lFl',
		'F+F......F+F',
		'.d..l**l..d.',
		'.dlLFFFFLld.',
		'..FFF%%FFF..',
		'.ddFFFFFFdd.',
		'd.d.dddd.d.d',
		'D..D....D..D',
	], { 0: '............', 1: '.l........l.', 2: 'lFl......lFl', 3: '+d..l**l..d+', 7: '.dd.dddd.dd.', 8: 'D.D......D.D' }),
	jelly: def([
		'....lLLl....',
		'...lLLFFl...',
		'..lLFFFFFl..',
		'..FFFFFFFF..',
		'..d%d%d%dd..',
		'...l.F.l.F..',
		'...l.F.l..F.',
		'..l..l.F..l.',
		'..F...l...F.',
	], { 0: '............', 1: '...lLLLl....', 2: '..lLFFFFFl..', 3: '.lFFFFFFFFl.', 4: '.dd%d%d%ddd.', 6: '..l..F.l.F..', 7: '..l.l..F..l.', 8: '.F...l....F.' }, { top: 5, fly: true }),

	// ---- insects and arachnids
	beetle: def([
		'.........l..',
		'...lLLLl.l..',
		'..lLFFF%lFF.',
		'..FFF%FFdF*.',
		'..dFFFFFdFF.',
		'...dddddDd..',
		'..d.d..d.d..',
		'.D..D..D..D.',
	], { 6: '...dd..dd...', 7: '..D..DD..D..' }),
	ant: def([
		'..........l.',
		'.........l..',
		'.lLl..lL.FL.',
		'lFFFl.dFlF*F',
		'dF%Fd.dd.dd.',
		'.dd.d.d.d...',
		'.D..D..D..D.',
	], { 5: '..d.d.d.d...', 6: '.D...DD...D.' }),
	fly: def([
		'...lL..Ll...',
		'..lLLllLLl..',
		'...lLLLLl...',
		'..dF%F%FF*..',
		'..dF%F%FFd..',
		'...d.d.d....',
	], { 0: '............', 1: '............', 2: '..lllLLlll..', 3: '.ldF%F%FF*l.', 4: 'l.dF%F%FFd.l', 5: '...d.d.d....' }, { top: 6, fly: true }),
	spider: def([
		'....lLLl....',
		'...lFF%Fl...',
		'...FF%%FF...',
		'.ll.dFFd.ll.',
		'lF.lFrrFl.Fl',
		'F.lFdFFdFl.F',
		'F.F.dkkd.F.F',
		'd.d..++..d.d',
		'D.D......D.D',
	], { 3: '.l..dFFd..l.', 4: 'lFllFrrFllFl', 5: 'F..FdFFdF..F', 6: 'F.F.dkkd.F.F', 7: '.d.d.++.d.d.', 8: '.D.D....D.D.' }),
	scorpion: def([
		'..lLl.......',
		'.lF.Fl......',
		'.F...+......',
		'.F..........',
		'.dF.....l.lL',
		'..FlLlLlF*FF',
		'..dFFFFFFFl.',
		'...d.d.d.d..',
		'..D.D.D.D...',
	], { 0: '...lLl......', 1: '..lF.Fl.....', 2: '..F...+.....', 3: '.F..........', 4: '.dF.....lL..', 7: '..d.d.d.d...', 8: '...D.D.D.D..' }),
	centipede: def([
		'..........l.',
		'.lLlLlLlLlF*',
		'lFdFdFdFdFFk',
		'd.d.d.d.d.d.',
		'.D.D.D.D.D..',
	], { 3: '.d.d.d.d.d.d', 4: 'D.D.D.D.D.D.' }),

	// ---- oozes and small slow things
	ooze: def([
		'....lLLl....',
		'...lLLFFl...',
		'..lLFFFFFl..',
		'..lF*FF*Fd..',
		'.lFFFFFFFFd.',
		'.FFF%FFFFFd.',
		'dFFFFFFFFFFd',
		'DddddddddddD',
	], { 0: '............', 1: '...lLLLll...', 2: '..lLFFFFFl..', 3: '.lLF*FF*FFd.', 4: '.lFFFFFFFFd.', 5: 'lFFF%FFFFFFd' }),
	snail: def([
		'...lLLl.....',
		'..lFddFl....',
		'..FdlFdF.l.l',
		'..FdddFF.k.k',
		'..dFFFd.kk*.',
		'jjkkkkkkkk..',
	], { 2: '..FdlFdF..l.l'.slice(0, 12), 3: '..FdddFF..k.', 4: '..dFFFd..kk*', 5: '.jjkkkkkkkk.' }),
	mimic: def([
		'..WwwwwwwW..',
		'.Ww*wwww*wW.',
		'.W+x+x+x+xW.',
		'.WRRRrRRRRW.',
		'.WRRRRrRRRW.',
		'.Wx+x+x+x+W.',
		'.WwwwywwwwW.',
		'.WwwwYwwwwW.',
		'.WWWWWWWWWW.',
	], { 0: '............', 1: '..WwwwwwwW..', 2: '.Ww*wwww*wW.', 3: '.W+x+x+x+xW.', 4: '.WRRRrRRRRW.' }),

	// ---- undead
	skeleton: def([
		'....lLLl....',
		'...lLLLFl...',
		'...L*FF*F...',
		'...lFzzFd...',
		'....F+F+....',
		'.....lF.....',
		'..lLFlFFFd..',
		'..F.lDFD.F..',
		'..F.lDFD.F..',
		'..l..lFd..l.',
		'..d..dFd..d.',
		'....lFFFd...',
		'....F...F...',
		'....F...F...',
		'....l...F...',
		'....F...F...',
		'...lF...FD..',
	], { 9: '..l..lFd...l', 10: '..d..dFd...d', 14: '....l....F..', 15: '...F.....F..', 16: '..lF.....FD.' }),
	ghost: def([
		'....lLLl....',
		'...lLLLLl...',
		'..lLLLLLLl..',
		'..LL*LL*Ll..',
		'..lLLzzLLl..',
		'.lLLLLLLLLl.',
		'lLLLLLLLLLLl',
		'l.lLLLLLLl.l',
		'..lLLLLLLl..',
		'..lLlLLlLl..',
		'..l.lL.lL.l.',
		'.....l...l..',
	], { 9: '..lLLlLLlL..', 10: '.l.lL.lL.l..', 11: '....l...l...' }, { top: 4, fly: true, bob1: true }),
	zombie: def([
		'......lLl...',
		'.....lFFFl..',
		'.....F*F*F..',
		'.....lFzFd..',
		'...aaaAaFFFF',
		'...aAaaa..Fd',
		'...aaAaa....',
		'...AaaaA....',
		'...a.aa.....',
		'...FF.FF....',
		'...F...F....',
		'...F...F....',
		'..DD..DD....',
	], { 4: '...aaaAaFFF.', 5: '...aAaaa.FFd', 10: '...F...F....', 11: '...F....F...', 12: '..DD....DD..' }),

	// ---- constructs, spirits, plants
	golem: def([
		'....lLLLl...',
		'....L!F!F...',
		'....lFFFd...',
		'.lLLLLlFFFl.',
		'lLFFFdFFFFFl',
		'FFFdFF!FdFFF',
		'FFF.dFFFd.FF',
		'dFd.FFdFF.dF',
		'dDd.dFFFd.dD',
		'....FFdFF...',
		'...lFF.FFl..',
		'...FFd.dFF..',
		'...FFd.dFF..',
		'..DDDD.DDDD.',
	], { 11: '...FFd..dFF.', 12: '..FFd...dFF.', 13: '.DDDD...DDDD' }),
	elemental: def([
		'.....l...l..',
		'....lL..lL..',
		'...lLL.lLl..',
		'..lLL!lLLl..',
		'..LL!!!LLl..',
		'.lL!*!*!Ll..',
		'.lL!!!!!LFl.',
		'lFL!!!!!LFFl',
		'lFFL!!!LFFFl',
		'.dFFLLLFFFd.',
		'..dFFFFFFd..',
		'...dFFFFd...',
		'....dFFd....',
		'...lD..Dl...',
	], { 0: '....l...l...', 1: '...lL..lL...', 2: '..lLL.lLl...', 3: '..lLL!lLLl..', 12: '...dFFFFd...', 13: '....dDDd....' }),
	treant: def([
		'...lLLLl....',
		'.lLLFFLLl...',
		'lLFFFFFFFl..',
		'FFFdFFFFFFl.',
		'dFFFFdFFFFF.',
		'.dFwwWwFFd..',
		'..ww*w*w....',
		'.w.wwWww.w..',
		'w..WwwwW..w.',
		'...wwWww....',
		'...WwwwW....',
		'..ww.W.ww...',
		'.WW..W..WW..',
	], { 0: '....lLLl....', 1: '..lLLFFLLl..', 2: '.lLFFFFFFFl.', 3: 'lFFdFFFFFFF.', 11: '..ww.W.ww...', 12: '..WW.W.WW...' }),
	mushroom: def([
		'...lLLLl....',
		'..lL%LL%l...',
		'.lFFFFFFFl..',
		'dFF%FFF%FFd.',
		'.ddddddddd..',
		'...jj*j*j...',
		'...jkkkkk...',
		'..k.kkkk.k..',
		'....kK.kK...',
	], { 7: '...kkkkkk...', 8: '...kK..kK...' }, { bob1: true }),

	// ---- great monsters
	dragon: def([
		'.L..........',
		'.Fl......+..',
		'.dFl....+lL.',
		'.ddFl...lF*F',
		'..ddFl..FFkx',
		'..dddFlFFk..',
		'...ddFFFFk..',
		'l...lFFFFk..',
		'Fl.lFF%FFFk.',
		'dFlFFFFFFFk.',
		'.dFFFFFFFFd.',
		'..ddkkkkkd..',
		'...dFd.dFd..',
		'...dF..dF...',
		'..DD..DD....',
	], {
		0: '............', 1: '.........+..', 2: '........+lL.', 3: '........lF*F', 4: '........FFkx',
		5: 'l....lFlFFk.', 6: 'Fl..lFFFFk..', 7: 'dFl.dFFFFk..', 8: 'ddFdFF%FFFk.', 9: 'dddFFFFFFFk.',
	}),
	demon: def([
		'.l..+..+..l.',
		'lFl.lLLl.lFl',
		'dFFlL*L*lFFd',
		'ddFFlFzFFFdd',
		'd.dLLFFFFLd.',
		'..lFFkkFFl..',
		'..FFkkkkFF..',
		'..k.FFFF.k..',
		'....FddF....',
		'...lF..Fd...',
		'...FF..FF...',
		'..DD....DD..',
	], {
		0: '....+..+....', 1: '.l..lLLl..l.', 2: 'lFllL*L*llFl', 3: 'dFFFlFzFFFFd',
		4: 'dddLLFFFFLdd', 5: 'd.lFFkkFFl.d',
	}, { bob1: true }),
	horror: def([
		'...lLLLLl...',
		'..lLFFFFFl..',
		'.lLFxxxxFFl.',
		'.LFxx**xxFd.',
		'.FFxx*zxxFd.',
		'.FFFxxxxFFd.',
		'.dFFFFFFFFd.',
		'lFdFdFFdFdFl',
		'F.lF.dF.Fl.F',
		'd.F..F..F..d',
		'.lF.lF..lF..',
		'.D..D....D..',
	], { 8: 'F..lF.dFl..F', 9: 'd.F..F..F..d', 10: '..lF.lF.lF..', 11: '..D...D..D..' }),
	worm: def([
		'.....lLLl...',
		'....lFxxFl..',
		'....FxRRxF..',
		'....FxRRxF..',
		'....dFxxFd..',
		'....lFFFFd..',
		'....FF%FFd..',
		'...lFFFFd...',
		'...FF%FFd...',
		'..lFFFFFd...',
		'..FFFFFFFd..',
		'TTdFFFFFFdTT',
		'tTTTTTTTTTTt',
	], {
		0: '......lLLl..', 1: '.....lFxxFl.', 2: '.....FxRRxF.', 3: '.....FxRRxF.', 4: '.....dFxxFd.',
		5: '....lFFFFd..', 6: '....FF%FFd..',
	}),
	hydra: def([
		'....lL...lL.',
		'...lF*..lF*F',
		'.lL.Fk..FFx+',
		'lF*lFk.lFk..',
		'FF+lFdlFk...',
		'.Fk.lFFd....',
		'..lLFFFFl...',
		'.lFFF%FFFl..',
		'lFFFF%FFFFl.',
		'dFFFFFFFFFd.',
		'.ddkkkkkdd..',
		'..dFd..dFd..',
		'..DD....DD..',
	], {
		0: '...lL...lL..', 1: '..lF*F.lF*F.', 2: 'lL.FFx..FFkx', 3: 'lF*lFk.lFk..', 4: 'FFx+FdlFk...',
		11: '...dFd.dFd..', 12: '...DD..DD...',
	}),
	griffin: def([
		'.L..........',
		'.Fl......lL.',
		'.dFl....lF*y',
		'..dFl...FFyy',
		'..ddFl.lFFk.',
		'L..ddFlFFk..',
		'l.lLFFFFFF..',
		'.lFFF%FFFFd.',
		'..dFFFFFFFd.',
		'..dkkkkkkKd.',
		'..Fd....dF..',
		'..Fd....dF..',
		'..YY....DD..',
	], {
		0: '............', 1: '.........lL.', 2: '..L.....lF*y', 3: '.lFl....FFyy', 4: '.ddFl..lFFk.',
		10: '.F..d..d..F.', 11: '.F..d..d..F.', 12: '.Y..Y..D..D.',
	}),
	basilisk: def([
		'....a.a.a...',
		'...aAaAaAa..',
		'..lLLLLLLlL.',
		'.lFFFFFFFF*!',
		'lFFF%F%FFFkx',
		'dFFFFFFFFdk+',
		'.ddkkkkkkd..',
		'.Fd.dF.dF...',
		'DD..DD..DD..',
	], {
		3: '.lFFFFFFFF*F', 4: 'lFFF%F%FFFk+', 5: 'dFFFFFFFFdk.', 7: '..Fd.dF.dF..', 8: '..DD.DD.DD..',
	}),
	mantis: def([
		'........l.l.',
		'........lF*F',
		'.......lFFk.',
		'......lF.lF.',
		'.....lF..F+.',
		'..lLLF...+..',
		'.lFFF%Fl....',
		'lFF%FFFd....',
		'dFFFFFFd....',
		'..d.d..d....',
		'.D..D...D...',
	], {
		3: '......lF..lF', 4: '.....lF...F+', 5: '..lLLF....+.', 9: '...d.d.d....', 10: '..D..D..D...',
	}),
	leviathan: def([
		'.....+.+....',
		'....lLLLl...',
		'...lF*FFFl..',
		'...FF+x+xF..',
		'..lFFkkkkd..',
		'.lFFF%FFd.L.',
		'lFF%FFFFdFl.',
		'dFFFFFFFdF..',
		'.ddFFFFFdd..',
		'...ddddd....',
	], {
		3: '...FFFFFFF..', 4: '..lFF+x+xd..', 5: '.lFFF%FFd..L', 6: 'lFF%FFFFd.Fl', 7: 'dFFFFFFFddF.',
	}, { top: 5, fly: true }),
};

const SZ: Record<string, number> = {
	canine: 4, feline: 4, bovine: 5, deer: 4, rodent: 1, bear: 5, horse: 5, boar: 4, bat: 1, elephant: 6, brute: 4,
	smallbird: 1, raptorbird: 3, bigbird: 4, lizard: 2, snake: 3, croc: 5, turtle: 3, frog: 1, raptor: 3, theropod: 6,
	sauropod: 7, armoured: 6, fish: 1, shark: 5, eel: 1, squid: 3, crab: 1, jelly: 1, beetle: 1, ant: 1, fly: 1,
	spider: 2, scorpion: 2, centipede: 2, ooze: 3, snail: 1, mimic: 3, skeleton: 3, ghost: 3, zombie: 3, golem: 5,
	elemental: 4, treant: 5, mushroom: 2, dragon: 6, demon: 4, horror: 5, worm: 4,
	hydra: 6, griffin: 5, basilisk: 5, mantis: 3, leviathan: 7,
};
const BASES: Record<string, Base> = Object.fromEntries(Object.entries(ART).map(([k, b]) => [k, { ...b, sz: SZ[k] ?? 4 }]));

// ---------------------------------------------------------------- kinds: base + colours + overlays

const MAM_BELLY = ['#e6d8b8', '#d4c098', '#f0e6d0', '#c8b08a'];
const REPT_BELLY = ['#d8cf8a', '#e0d8a8', '#c8c078', '#b8c890'];
const FISH_BELLY = ['#e4ecf0', '#d0dce4', '#f0f0e0'];
const DARK_MARK = ['#3a2a1a', '#2a2420', '#5a3a20', '#1e1e24'];
const BRIGHT_MARK = ['#d83a2a', '#e8c030', '#3a8ad8', '#2a2a2a', '#e8e0d0'];
const MANE = ['#5a3a1e', '#8a6a2a', '#2a1b12', '#c9a24a', '#d8d0c0'];
const RAGS = ['#5a5040', '#4a4a3a', '#6a5a48', '#3a3a44'];
const FLESH = ['#a8a090', '#9a8a78', '#b8b0a0'];

interface Kind {
	b: string; // base
	eye: string; // fixed palette char for '*'
	skin?: string[];
	acc?: string[];
	hair?: string[];
	mark?: number; // 0..4: chance in quarters that an individual shows its % & markings
	ov?: Patch; // overlay on the authored rows of the base
	glow?: string; // '!' colour
}

const KINDS: Record<string, Kind> = {
	canine: { b: 'canine', eye: 'y', skin: MAM_BELLY, acc: DARK_MARK, mark: 2 },
	feline: { b: 'feline', eye: 'y', skin: MAM_BELLY, acc: DARK_MARK, mark: 3 },
	lion: { b: 'feline', eye: 'y', skin: MAM_BELLY, acc: DARK_MARK, hair: MANE, mark: 0, ov: { 0: '.......hh...', 1: '.......h....', 2: '......h.....', 3: '......hh....', 4: '......hh....', 5: '.......h....' } },
	bovine: { b: 'bovine', eye: 'z', skin: MAM_BELLY, acc: DARK_MARK, hair: MANE, mark: 1 },
	deer: { b: 'deer', eye: 'z', skin: MAM_BELLY, acc: ['#f0e8d8', '#e8dcc0'], mark: 1 },
	antelope: { b: 'deer', eye: 'z', skin: MAM_BELLY, acc: DARK_MARK, mark: 2, ov: { 0: '......+     ', 1: '.......+    ', 2: '........l   ' } },
	rodent: { b: 'rodent', eye: 'z', skin: MAM_BELLY, acc: DARK_MARK, mark: 1 },
	bear: { b: 'bear', eye: 'z', skin: MAM_BELLY, acc: DARK_MARK, mark: 0 },
	horse: { b: 'horse', eye: 'z', skin: MAM_BELLY, acc: DARK_MARK, hair: MANE, mark: 1 },
	zebra: { b: 'horse', eye: 'z', skin: ['#f0f0e8'], acc: ['#1e1e24'], hair: ['#1e1e24'], mark: 4, ov: { 7: '..a.a.a.a...', 8: '...a.a.a....', 9: '..a.a.a.a...', 11: '..a.....a...', 13: '..a.....a...' } },
	camel: { b: 'horse', eye: 'z', skin: MAM_BELLY, acc: DARK_MARK, hair: ['#8a6a3a'], mark: 0, ov: { 4: '....lL......', 5: '...lFFl.....', 6: '.hlLFFlFFd..' } },
	boar: { b: 'boar', eye: 'z', skin: MAM_BELLY, acc: DARK_MARK, hair: MANE, mark: 1 },
	bat: { b: 'bat', eye: 'r', skin: MAM_BELLY, mark: 0 },
	elephant: { b: 'elephant', eye: 'z', skin: MAM_BELLY, mark: 0 },
	mammoth: { b: 'elephant', eye: 'z', skin: MAM_BELLY, hair: MANE, mark: 0, ov: { 0: '..hHhHhh....', 1: '.hh.........', 2: 'hh..........', 9: '.HhhhhhhhH..' } },
	ape: { b: 'brute', eye: 'z', skin: ['#c8a888', '#a88868', '#d8b898'], mark: 0 },
	yeti: { b: 'brute', eye: 'c', skin: ['#c8d8e8', '#b8c8d8'], mark: 0 },
	troll: { b: 'brute', eye: 'y', skin: ['#8aa070', '#7a9060', '#9aa878'], mark: 0, ov: { 3: '..lLF+k+F...' } },
	brute: { b: 'brute', eye: 'r', skin: FLESH, mark: 0 },
	smallbird: { b: 'smallbird', eye: 'z', skin: MAM_BELLY, acc: BRIGHT_MARK, mark: 1 },
	raptorbird: { b: 'raptorbird', eye: 'y', skin: MAM_BELLY, mark: 0 },
	phoenix: { b: 'raptorbird', eye: 'y', skin: ['#ffd860', '#ffb030'], mark: 0, ov: { 0: '...y........', 1: '...nl.......', 6: 'nnrFFFFFFkk.', 7: '.nr.dkkkk...' } },
	pterosaur: { b: 'raptorbird', eye: 'z', skin: MAM_BELLY, mark: 0, ov: { 4: '....ddFl.lLa', 5: '..dddFFFFF*y' } },
	bigbird: { b: 'bigbird', eye: 'z', skin: ['#d89878', '#c88868', '#e8b898'], hair: MANE, acc: DARK_MARK, mark: 1 },
	lizard: { b: 'lizard', eye: 'y', skin: REPT_BELLY, acc: DARK_MARK, mark: 2 },
	salamander: { b: 'lizard', eye: 'y', skin: ['#ffb040', '#ffd060'], acc: ['#ffd84a', '#ff8a2a'], mark: 4 },
	snake: { b: 'snake', eye: 'y', skin: REPT_BELLY, acc: BRIGHT_MARK, mark: 2 },
	croc: { b: 'croc', eye: 'y', skin: REPT_BELLY, acc: DARK_MARK, mark: 2 },
	turtle: { b: 'turtle', eye: 'z', skin: ['#9a9a68', '#8a9a70', '#a8a080'], acc: ['#3a3020', '#5a4a28', '#d8c890'], mark: 3 },
	frog: { b: 'frog', eye: 'y', skin: REPT_BELLY, acc: BRIGHT_MARK, mark: 2 },
	raptor: { b: 'raptor', eye: 'y', skin: REPT_BELLY, acc: DARK_MARK, mark: 2 },
	theropod: { b: 'theropod', eye: 'y', skin: REPT_BELLY, acc: DARK_MARK, mark: 2 },
	hadrosaur: { b: 'theropod', eye: 'z', skin: REPT_BELLY, acc: DARK_MARK, mark: 1, ov: { 0: '.......lLLl.', 4: '......dFFFFF', 5: '.....lFkkkkd', 1: '.....alLFFFl' } },
	sauropod: { b: 'sauropod', eye: 'z', skin: REPT_BELLY, acc: DARK_MARK, mark: 2 },
	armoured: { b: 'armoured', eye: 'z', skin: REPT_BELLY, acc: ['#b85a3a', '#c89a3a', '#5a8a4a'], mark: 1 },
	ceratops: {
		b: 'armoured', eye: 'z', skin: REPT_BELLY, acc: ['#b85a3a', '#c89a3a', '#5a8a4a'], mark: 1,
		ov: { 0: '            ', 1: '            ', 2: '.......aa...', 3: '.lLLLLaAAa..', 4: 'lFFFFFaAAal+', 5: 'FF%FF%FaalF+', 6: 'dFFFFFFFFF*F', 7: 'ddFFFFFFFdFk', 8: ' .dkkkkkd...' },
	},
	fish: { b: 'fish', eye: 'z', skin: FISH_BELLY, acc: BRIGHT_MARK, mark: 2 },
	shark: { b: 'shark', eye: 'z', skin: FISH_BELLY, acc: DARK_MARK, mark: 1 },
	whale: { b: 'shark', eye: 'z', skin: FISH_BELLY, mark: 0, ov: { 0: '      ', 1: '      ', 2: '....lLLl....', 5: 'Fd.dkkkkkkkk' } },
	eel: { b: 'eel', eye: 'y', skin: FISH_BELLY, acc: DARK_MARK, mark: 1 },
	leech: { b: 'eel', eye: 'r', skin: ['#a84a4a'], mark: 0 },
	squid: { b: 'squid', eye: 'y', skin: MAM_BELLY, acc: BRIGHT_MARK, mark: 2 },
	crab: { b: 'crab', eye: 'z', skin: MAM_BELLY, acc: DARK_MARK, mark: 1 },
	jelly: { b: 'jelly', eye: 'z', skin: MAM_BELLY, acc: ['#f0a0d0', '#a0e0f0', '#f0e0a0'], mark: 3 },
	beetle: { b: 'beetle', eye: 'r', acc: BRIGHT_MARK, mark: 1 },
	ant: { b: 'ant', eye: 'z', acc: DARK_MARK, mark: 1 },
	fly: { b: 'fly', eye: 'r', acc: ['#1e1e24', '#e8c030', '#d83a2a'], mark: 3 },
	spider: { b: 'spider', eye: 'r', skin: ['#5a3030', '#3a3030'], acc: BRIGHT_MARK, mark: 2 },
	scorpion: { b: 'scorpion', eye: 'r', acc: DARK_MARK, mark: 0 },
	centipede: { b: 'centipede', eye: 'r', skin: ['#d8a050', '#c86030'], mark: 0 },
	ooze: { b: 'ooze', eye: 'z', acc: ['#f0f0e0', '#a0e080'], mark: 1 },
	snail: { b: 'snail', eye: 'z', skin: ['#c8b098', '#a8b0a0', '#d0a890'], mark: 0 },
	mimic: { b: 'mimic', eye: 'y', mark: 0 },
	skeleton: { b: 'skeleton', eye: 'r', mark: 0 },
	lich: { b: 'skeleton', eye: 'c', acc: ['#6a3a9a', '#3a2a5a'], mark: 0, ov: { 0: '...y.y.y....', 6: '..aaFaFaaa..', 7: '..a.aDaDaa..', 8: '..a.aDaDaa..', 11: '...aaaaaaa..', 12: '...aaaaaaa..', 13: '...aaaaaaa..', 14: '...aAaaaAa..' } },
	ghost: { b: 'ghost', eye: 'z', mark: 0 },
	wraith: { b: 'ghost', eye: 'r', mark: 0, ov: { 0: '....dDDd....', 1: '...dDDDDd...', 2: '..dDDDDDDd..', 3: '..dD*DD*Dd..', 4: '..dDDDDDDd..' } },
	zombie: { b: 'zombie', eye: 'r', acc: RAGS, mark: 4 },
	mummy: { b: 'zombie', eye: 'r', acc: ['#d8c8a0', '#c8b890'], mark: 4 },
	golem: { b: 'golem', eye: 'y', mark: 0, glow: 'n' },
	elemental: { b: 'elemental', eye: 'x', mark: 0, glow: 'y' },
	wisp: { b: 'elemental', eye: 'x', mark: 0, glow: 'c' },
	treant: { b: 'treant', eye: 'y', mark: 0 },
	mushroom: { b: 'mushroom', eye: 'z', skin: ['#e8dcc0', '#d8c8b0', '#c8c0d0'], acc: ['#f0f0e0', '#e8d090'], mark: 3 },
	dragon: { b: 'dragon', eye: 'y', skin: REPT_BELLY, acc: DARK_MARK, mark: 2 },
	elder_dragon: { b: 'dragon', eye: '!', skin: ['#f0d060', '#e8a838'], acc: ['#d83a2a', '#6a2a8a'], mark: 4, glow: 'y', ov: { 1: '.Fl....y.+..', 2: '.dFl...+ylL.', 7: 'l.a.lFFFFk..', 8: 'FlaaFF!FFFk.' } },
	hydra: { b: 'hydra', eye: 'y', skin: REPT_BELLY, acc: ['#8a2a2a', '#2a5a4a', '#5a3a7a'], mark: 3 },
	griffin: { b: 'griffin', eye: 'y', skin: ['#f0e4c0', '#e8d098'], acc: ['#c89a3a', '#8a5a2a'], mark: 2 },
	basilisk: { b: 'basilisk', eye: 'v', skin: REPT_BELLY, acc: ['#4a8a3a', '#8a6a2a', '#5a3a6a'], mark: 3, glow: 'v' },
	mantis: { b: 'mantis', eye: 'y', skin: ['#c8e090', '#a8c870'], acc: ['#2a6a30', '#8a6a2a'], mark: 2 },
	leviathan: { b: 'leviathan', eye: 'c', skin: FISH_BELLY, acc: ['#2a4a7a', '#3a6a6a'], mark: 3, glow: 'c' },
	kraken: { b: 'squid', eye: 'r', skin: ['#d8a0c0', '#c090b0'], acc: ['#6a2a7a', '#9a2a4a'], mark: 4, ov: { 0: '...+lLLl+...', 1: '..+lLFFFl+..', 4: '...F*!F*!F..' } },
	death_knight: { b: 'skeleton', eye: 'c', acc: ['#3a3a48', '#5a2a3a'], mark: 0, ov: { 0: '...mMMMMm...', 1: '..mMLLLLMm..', 6: '..mMFmFMMm..', 7: '..m.mDmDMm..', 8: '..m.mDmDMm..', 11: '...mMMMMm...' } },
	beholder: { b: 'horror', eye: 'r', acc: ['#8a3a9a', '#a84a4a'], mark: 0, ov: { 0: '..*..*..*...', 1: '..lL.l.Ll...', 4: '.FFxx*!xxFd.' }, glow: 'y' },
	owl: { b: 'raptorbird', eye: 'y', skin: ['#e8dcc8', '#d8c8b0'], acc: DARK_MARK, mark: 1, ov: { 3: '...ddFl..l.l', 4: '....ddFl.lLl' } },
	wolf_alpha: { b: 'canine', eye: 'r', skin: ['#e8e8e0', '#d8d0c0'], acc: ['#e8e8f0', '#2a2420'], hair: MANE, mark: 3, ov: { 1: '.......hlFd.', 3: '.lF...hlFFkk', 4: '..lLLhhlFFk.' } },
	crystal_golem: { b: 'golem', eye: 'c', mark: 0, glow: 'c', ov: { 0: '....cLLLc...', 3: '.cLLLLcFFFc.', 5: 'FFFdFFcFdFFF' } },
	demon: { b: 'demon', eye: 'y', skin: ['#c84a3a', '#a83a3a', '#8a2a4a'], mark: 0 },
	gargoyle: { b: 'demon', eye: 'r', skin: ['#9a9a9a', '#8a8a90'], mark: 0 },
	horror: { b: 'horror', eye: 'r', mark: 0 },
	worm: { b: 'worm', eye: 'z', acc: DARK_MARK, mark: 1 },
};

// ---------------------------------------------------------------- classification

type Rule = [RegExp, string];
// kinds that win in any group (the supernatural, the built and a few name clashes)
const SPECIAL: Rule[] = [
	[/komodo|monitor|megalania|perentie/, 'croc'],
	[/flying dragon|thorny devil|basilisk lizard|common basilisk|frilled/, 'lizard'],
	[/tasmanian devil/, 'canine'],
	[/seadragon|seahorse/, 'fish'],
	[/dragonfly|damselfly/, 'fly'],
	[/nudibranch/, 'snail'],
	[/hydra/, 'hydra'],
	[/griffin|gryphon|hippogriff/, 'griffin'],
	[/elder dragon|ancient dragon|archdragon|dragon lord|wyrmlord/, 'elder_dragon'],
	[/dragon|wyvern|drake|wyrm/, 'dragon'],
	[/death knight|dread knight|grave knight|blackguard/, 'death_knight'],
	[/lich/, 'lich'],
	[/skeleton|bones|ossuary/, 'skeleton'],
	[/wraith|shade|spectre|specter|banshee/, 'wraith'],
	[/ghost|phantom|spirit|sylph|haunt/, 'ghost'],
	[/mummy/, 'mummy'],
	[/zombie|ghoul|wight|draugr|revenant|corpse/, 'zombie'],
	[/gargoyle/, 'gargoyle'],
	[/crystal golem|crystalgolem|gem golem|prism golem/, 'crystal_golem'],
	[/golem|construct|automaton|turret|sentinel|clockwork/, 'golem'],
	[/wisp/, 'wisp'],
	[/elemental|magma|ember|flame|inferno/, 'elemental'],
	[/phoenix/, 'phoenix'],
	[/treant|dryad|shambler|cactoid|mandrake|vine|bud pet|bud-pet/, 'treant'],
	[/spore|fung|mushroom|myconid|toadstool/, 'mushroom'],
	[/demon|fiend|\bimp\b|devil|hellhound/, 'demon'],
	[/kraken/, 'kraken'],
	[/beholder|eye tyrant|gazer/, 'beholder'],
	[/horror|mindleech|eldritch|abomination|aberration/, 'horror'],
	[/mimic/, 'mimic'],
	[/cloudjelly|jellyfish|man o|medusa|anemone/, 'jelly'],
	[/\booze\b|slime|pudding|\bblob\b|gelatin/, 'ooze'],
	[/leviathan|sea serpent|jormungandr/, 'leviathan'],
	[/sandworm|purple worm|great worm/, 'worm'],
	[/basilisk|cockatrice/, 'basilisk'],
	[/dire wolf|alpha wolf|fenrir|warg/, 'wolf_alpha'],
	[/salamander/, 'salamander'],
	[/yeti|sasquatch|bigfoot/, 'yeti'],
	[/troll|ogre|ogor/, 'troll'],
];
const MAMMAL: Rule[] = [
	[/\bbat\b|flying fox|colugo|vampire/, 'bat'],
	[/mammoth|mastodon/, 'mammoth'],
	[/elephant|deinotherium|platybelodon|indricother|paraceratherium|elasmotherium|arsinoither|brontother|uintather|rhino/, 'elephant'],
	[/whale|dolphin|orca|porpoise|narwhal|beluga|baiji|\bseal\b|sea lion|walrus|manatee|dugong|sea cow|basilosaurus|livyatan|ambulocetus|pakicetus|odobenocetops/, 'whale'],
	[/bear|panda|amphicyon/, 'bear'],
	[/lion|smilodon|homotherium/, 'lion'],
	[/\bcat\b|tiger|leopard|jaguar|lynx|cougar|cheetah|puma|serval|caracal|ocelot|bobcat|fossa|genet|thylacosmilus|thylacoleo|wildcat|stalker/, 'feline'],
	[/zebra|quagga/, 'zebra'],
	[/camel|dromedary|llama|guanaco|vicuna/, 'camel'],
	[/horse|\bass\b|donkey|kiang|tarpan|onager|okapi|giraffe|tapir|macrauchenia|chalicother|moropus|hyracotherium/, 'horse'],
	[/boar|\bpig\b|\bhog\b|warthog|peccary|hippo|entelodon|daeodon|armadillo|pangolin|aardvark|glyptodon|doedicurus|anteater|wombat|diprotodon|toxodon|tapir/, 'boar'],
	[/bison|buffalo|\box\b|yak|aurochs|gaur|banteng|cattle|\bcow\b|\bbull\b|musk|takin|anoa|moschops|estemmenosuchus/, 'bovine'],
	[/deer|elk|moose|reindeer|caribou|stag|muntjac|chital|sambar|megaloceros|pudu/, 'deer'],
	[/antelope|gazelle|goat|ibex|sheep|mouflon|argali|chamois|springbok|impala|oryx|kudu|eland|gnu|wildebeest|bongo|nilgai|addax|saiga|dik|klipspringer|pronghorn|markhor|blackbuck|hartebeest|topi|sitatunga|bluebuck|gerenuk|nilgai/, 'antelope'],
	[/ape\b|gorilla|chimp|bonobo|orangutan|monkey|baboon|macaque|lemur|gibbon|mandrill|gelada|loris|tarsier|marmoset|capuchin|gigantopithecus|sloth|megatherium|aye/, 'ape'],
	[/wolf|\bdog\b|fox|jackal|coyote|dhole|hyena|thylacine|dingo|aardwolf|otter|badger|wolverine|raccoon|skunk|marten|fisher|binturong|devil|andrewsarchus|gorgonops|inostrancevia|unit/, 'canine'],
	[/rat\b|mouse|squirrel|rabbit|hare|shrew|mole|hedgehog|lemming|chipmunk|marmot|pika|hyrax|jerboa|gerbil|vole|chinchilla|beaver|porcupine|capybara|melomys|prairie|ferret|stoat|weasel|mink|mongoose|meerkat|kangaroo|wallaby|quokka|bandicoot|numbat|glider|opossum|koala|platypus|echidna|agouti|coati|kinkajou|castoroides|josephoartigasia|morganucodon|repenomamus|leptictidium|deinogalerix/, 'rodent'],
	[/dimetrodon|edaphosaurus|lystrosaurus/, 'lizard'],
];
const BIRD: Rule[] = [
	[/owl/, 'owl'],
	[/eagle|hawk|falcon|kestrel|vulture|condor|kite|osprey|harrier|goshawk|buzzard|albatross|frigate|gannet|lammergeier|gyrfalcon|\broc\b|harpy|pelagornis|caracara/, 'raptorbird'],
	[/ostrich|emu|rhea|cassowary|dodo|kiwi|moa|elephant bird|turkey|peafowl|flamingo|heron|stork|crane|egret|ibis|spoonbill|shoebill|marabou|secretary|bustard|penguin|auk|swan|goose|pelican|solitaire|capercaillie|roadrunner/, 'bigbird'],
];
const REPTILE: Rule[] = [
	[/snake|viper|cobra|mamba|python|\bboa\b|adder|krait|taipan|rattle|anaconda|titanoboa|sidewinder|bushmaster|fer de lance|caecilian/, 'snake'],
	[/croc|alligator|caiman|gharial|deinosuchus|sarcosuchus|purussaurus|dakosaurus|postosuchus|desmatosuchus|prionosuchus|eryops|giant salamander/, 'croc'],
	[/turtle|tortoise|terrapin|archelon|megalochelys|matamata|placodus|scutosaurus/, 'turtle'],
	[/plesio|elasmosaurus|ichthyo|mosasaur|tylosaurus|kronosaurus|liopleurodon|nothosaur|ophthalmo|shonisaurus|globidens/, 'shark'],
	[/frog|toad|beelzebufo/, 'frog'],
	[/tanystropheus/, 'sauropod'],
];
const DINO: Rule[] = [
	[/apatosaurus|brachiosaurus|brontosaurus|diplodocus|argentinosaurus|patagotitan|camarasaurus|mamenchisaurus|amargasaurus|plateosaurus|mussaurus/, 'sauropod'],
	[/triceratops|styracosaurus|protoceratops|pachyrhinosaurus|psittacosaurus|ceratops/, 'ceratops'],
	[/stegosaurus|kentrosaurus|ankylosaurus|nodosaurus|euoplocephalus/, 'armoured'],
	[/iguanodon|edmontosaurus|parasaurolophus|corythosaurus|lambeosaurus|maiasaura|muttaburrasaurus|ouranosaurus|hadro|pachycephalosaurus/, 'hadrosaur'],
	[/rex|tyrannosaurus|allosaurus|giganotosaurus|spinosaurus|carcharodonto|carnotaurus|ceratosaurus|albertosaurus|tarbosaurus|daspletosaurus|torvosaurus|majungasaurus|baryonyx|yutyrannus|therizinosaurus|deinocheirus/, 'theropod'],
	[/archaeopteryx|microraptor/, 'smallbird'],
];
const FISH: Rule[] = [
	[/shark|megalodon|dunkleosteus|cretoxyrhina|helicoprion|cladoselache|stethacanthus|dogfish|wobbegong|sawfish|hyneria|xiphactinus|leedsichthys|rhizodus/, 'shark'],
	[/\beel\b|lamprey|hagfish|oarfish|gulper/, 'eel'],
];
const INSECT: Rule[] = [
	[/\bant\b|termite/, 'ant'],
	[/bee\b|wasp|hornet|\bfly\b|moth|butterfly|mosquito|cicada|locust|firefly|meganeur|morpho|shardmoth|gnat|midge/, 'fly'],
	[/centipede|millipede|arthropleura|velvet worm|crawler/, 'centipede'],
	[/scorpion|eurypter|pterygotus|jaekelopterus/, 'scorpion'],
	[/spider|tarantula|birdeater|harvestman|widow/, 'spider'],
	[/crab|lobster|isopod|krill|shrimp|trilobite|walliserops|woodlouse|horseshoe|marrella|anomalocaris|opabinia/, 'crab'],
	[/mantis/, 'mantis'],
	[/stick|weta|cricket|beetle|weevil|water bug/, 'beetle'],
];
const MOLLUSC: Rule[] = [
	[/octopus|squid|cuttlefish|nautilus|ammonite|belemnite|orthoceras|cameroceras/, 'squid'],
	[/snail|slug|conch|cone|sea hare|clam|kimberella|wiwaxia/, 'snail'],
	[/coral|sponge|crinoid|starfish|urchin|cucumber/, 'ooze'],
	[/\bleech\b/, 'leech'],
];

const first = (rules: Rule[], s: string): string | null => {
	for (const [re, k] of rules) if (re.test(s)) return k;
	return null;
};

/** humanoids with a template are people the old drawer handles (bandits, sirens, troglodytes keep theirs) */
const PEOPLE = /bandit|archer|chief|outlaw|siren|troglodyte|guard|merchant|cultist|knight|priest|mage/;

function classify(a: QudActor | undefined, g: string): string | null {
	const s = a ? ` ${a.t ?? ''} ${a.nm ?? ''} `.toLowerCase().replace(/[_'-]/g, ' ') : '';
	if (a) {
		const sp = first(SPECIAL, s);
		if (sp) return sp;
		const z = a.z ?? 3;
		switch (a.gr) {
			case 'mam':
			case 'syn':
				return first(MAMMAL, s) ?? (a.b === 'fish' ? 'whale' : z <= 2 ? 'rodent' : z <= 4 ? 'canine' : z <= 5 ? 'bovine' : 'elephant');
			case 'bird':
				return first(BIRD, s) ?? (z >= 3 ? 'raptorbird' : 'smallbird');
			case 'ptero':
				return 'pterosaur';
			case 'rept':
				return first(REPTILE, s) ?? (z >= 5 ? 'croc' : 'lizard');
			case 'amph':
				return first(REPTILE, s) ?? (z >= 4 ? 'croc' : 'lizard');
			case 'dino':
				return first(DINO, s) ?? (z >= 5 ? 'theropod' : 'raptor');
			case 'fish':
				return first(FISH, s) ?? (z >= 5 ? 'shark' : 'fish');
			case 'ins':
			case 'arach':
			case 'crust':
			case 'myria':
			case 'arth':
				return first(INSECT, s) ?? (a.gr === 'arach' ? 'spider' : a.gr === 'crust' || a.gr === 'arth' ? 'crab' : a.gr === 'myria' ? 'centipede' : 'beetle');
			case 'worm':
				return /leech/.test(s) ? 'leech' : z >= 3 ? 'worm' : 'leech';
			case 'moll':
				return first(MOLLUSC, s) ?? 'snail';
			case 'cnid':
				return 'jelly';
			case 'echi':
			case 'misc':
				return 'ooze';
		}
		const kw = first(MAMMAL, s) ?? first(BIRD, s) ?? first(DINO, s) ?? first(REPTILE, s) ?? first(FISH, s) ?? first(INSECT, s) ?? first(MOLLUSC, s);
		if (kw) return kw;
		switch (a.b) {
			case 'quadruped':
				return 'canine';
			case 'saurian':
				return 'lizard';
			case 'avian':
				return g === 'b' ? 'bat' : 'raptorbird';
			case 'fish':
				return 'fish';
			case 'insect':
				return 'beetle';
			case 'arachnid':
				return 'spider';
			case 'crab':
				return 'crab';
			case 'serpent':
				return 'snake';
			case 'ooze':
				return 'ooze';
			case 'tentacled':
				return 'horror';
			case 'golem':
				return 'golem';
			case 'humanoid':
				return g === '@' || PEOPLE.test(s) ? null : 'brute';
		}
	}
	return GLYPH[g] ?? null;
}

/** no actor (or nothing known): the roguelike letter alone */
const GLYPH: Record<string, string> = {
	r: 'rodent', b: 'bat', d: 'canine', C: 'canine', f: 'feline', B: 'bear', q: 'boar', s: 'snake', S: 'scorpion',
	x: 'spider', j: 'centipede', z: 'skeleton', Z: 'zombie', W: 'worm', g: 'golem', G: 'golem', D: 'dragon',
	E: 'elemental', F: 'shark', e: 'eel', k: 'crab', X: 'horror', H: 'horror', o: 'ooze', J: 'jelly', v: 'fly',
	K: 'croc', u: 'frog', R: 'raptorbird', M: 'mammoth', T: 'theropod', I: 'ant', a: 'ant', l: 'leech',
};

// ---------------------------------------------------------------- size, finish, outline

/** target art width (unoutlined) per bestiary size 1..7 */
const TARGET_W = [0, 6, 8, 9, 10, 12, 12, 12];

function bbox(fr: string[]) {
	let x0 = PW, x1 = -1, y0 = PH, y1 = -1;
	fr.forEach((r, y) => {
		for (let x = 0; x < PW; x++) {
			if (r[x] === '.') continue;
			if (x < x0) x0 = x;
			if (x > x1) x1 = x;
			if (y < y0) y0 = y;
			if (y > y1) y1 = y;
		}
	});
	return { x0, x1, y0, y1 };
}

/** nearest-block downscale: eyes survive, a block keeps its commonest colour when at least half filled */
function shrink(fr: string[], s: number, fly: boolean, box: ReturnType<typeof bbox>): string[] {
	const w = box.x1 - box.x0 + 1, h = box.y1 - box.y0 + 1;
	const nw = Math.max(2, Math.round(w * s)), nh = Math.max(2, Math.round(h * s));
	const out = Array.from({ length: PH }, () => Array<string>(PW).fill('.'));
	const ox = Math.round((box.x0 + box.x1 + 1) / 2 - nw / 2);
	const oy = fly ? Math.round((box.y0 + box.y1 + 1) / 2 - nh / 2) : box.y1 + 1 - nh;
	for (let dy = 0; dy < nh; dy++) {
		const sy0 = box.y0 + Math.floor((dy * h) / nh), sy1 = Math.max(sy0 + 1, box.y0 + Math.floor(((dy + 1) * h) / nh));
		for (let dx = 0; dx < nw; dx++) {
			const sx0 = box.x0 + Math.floor((dx * w) / nw), sx1 = Math.max(sx0 + 1, box.x0 + Math.floor(((dx + 1) * w) / nw));
			const cnt: Record<string, number> = {};
			let filled = 0, total = 0, eye = false;
			for (let y = sy0; y < sy1; y++) for (let x = sx0; x < sx1; x++) {
				total++;
				const c = fr[y][x];
				if (c === '.') continue;
				if (c === '*') eye = true;
				filled++;
				cnt[c] = (cnt[c] ?? 0) + 1;
			}
			let best = '.';
			if (eye) best = '*';
			else if (filled * 2 >= total) {
				let n = 0;
				for (const c in cnt) if (cnt[c] > n) (n = cnt[c]), (best = c);
			}
			const tx = ox + dx, ty = oy + dy;
			if (tx >= 0 && tx < PW && ty >= 0 && ty < PH) out[ty][tx] = best;
		}
	}
	return out.map((r) => r.join(''));
}

/** 4-neighbour 1px outline around every filled pixel */
function outline(fr: string[]): string[] {
	const g = fr.map((r) => r.split(''));
	const on = (x: number, y: number) => x >= 0 && x < PW && y >= 0 && y < PH && fr[y][x] !== '.';
	for (let y = 0; y < PH; y++) for (let x = 0; x < PW; x++) {
		if (fr[y][x] !== '.') continue;
		if (on(x - 1, y) || on(x + 1, y) || on(x, y - 1) || on(x, y + 1)) g[y][x] = 'o';
	}
	return g.map((r) => r.join(''));
}

const pick = <T,>(arr: T[] | undefined, h: number): T | undefined => (arr && arr.length ? arr[h % arr.length] : undefined);

function elemGlow(a: QudActor | undefined, k: Kind): string {
	const t = `${a?.t ?? ''} ${a?.nm ?? ''} ${(a?.fl ?? []).join(' ')}`;
	if (/frost|ice|snow|crystal/.test(t)) return 'c';
	if (/shock|storm|spark/.test(t)) return 'c';
	if (/acid|poison|spore|venom/.test(t)) return 'v';
	if (/fire|magma|lava|ember|flame/.test(t)) return 'y';
	return k.glow ?? 'y';
}

export function creatureSprite(a: QudActor | undefined, g: string, h: number, frame: number): CreaturePixels | null {
	const kk = classify(a, g);
	if (!kk) return null;
	const k = KINDS[kk];
	const base = k && BASES[k.b];
	if (!base) return null;
	const f = ((frame + (h & 3)) >> 2) & 1;
	let fr = overlay(base.f[f], k.ov, base.top);
	// size: shrink toward the target width of the bestiary size (no growing: art is drawn at its natural size)
	const z = a?.z;
	if (z && z >= 1 && z < base.sz) {
		const box = bbox(fr);
		const w = box.x1 - box.x0 + 1;
		// keep frame 0's box for both frames so the animation does not breathe in scale
		const b0 = bbox(overlay(base.f[0], k.ov, base.top));
		const s = Math.max(0.6, TARGET_W[z] / (b0.x1 - b0.x0 + 1));
		if (s < 0.95 && w > 2) fr = shrink(fr, s, base.fly, { ...box, x0: Math.min(box.x0, b0.x0), x1: Math.max(box.x1, b0.x1) });
	}
	const marked = ((h >>> 3) & 3) < (k.mark ?? 0);
	const glow = elemGlow(a, k);
	fr = fr.map((r) =>
		r.replace(/[*+%&!]/g, (c) => (c === '*' ? k.eye : c === '+' ? 'x' : c === '%' ? (marked ? 'a' : 'F') : c === '&' ? (marked ? 'A' : 'd') : glow)),
	);
	fr = outline(fr);
	if (a?.fl?.includes('left')) fr = flipX(fr);
	return {
		rows: fr,
		pal: { skin: pick(k.skin, h >>> 5), hair: pick(k.hair, h >>> 8), accent: pick(k.acc, h >>> 11) },
	};
}

// ---------------------------------------------------------------- the sprite sheet

export interface CreatureSample { label: string; actor: QudActor; glyph: string; fg?: string }
let sid = 1;
const S = (label: string, glyph: string, fg: string, act: Partial<QudActor>): CreatureSample => ({ label, glyph, fg, actor: { x: 0, y: 0, i: sid++, ...act } });

export const SAMPLE_CREATURES: CreatureSample[] = [
	// mammals
	S('wolf', 'C', '#a0a0a0', { t: 'wolf', b: 'quadruped' }),
	S('red fox z2', 'c', '#c87a3a', { t: 'bs_red_fox', gr: 'mam', z: 2 }),
	S('jaguar', 'f', '#d8b040', { t: 'jaguar', b: 'quadruped' }),
	S('lion z5', 'l', '#c8a060', { t: 'bs_lion', gr: 'mam', z: 5 }),
	S('wildcat z2', 'l', '#a08868', { t: 'bs_wildcat', gr: 'mam', z: 2 }),
	S('bison z6', 'B', '#7a5a3a', { t: 'bs_american_bison', gr: 'mam', z: 6 }),
	S('red deer z4', 'd', '#a8703a', { t: 'bs_red_deer', gr: 'mam', z: 4 }),
	S('ibex z4', 'I', '#b0a080', { t: 'bs_alpine_ibex', gr: 'mam', z: 4 }),
	S('rat', 'r', '#8a6a4a', { t: 'rat', b: 'quadruped' }),
	S('mouse z1', 'r', '#a0a0a0', { t: 'bs_house_mouse', gr: 'mam', z: 1 }),
	S('bear', 'B', '#7a5030', { t: 'bear', b: 'quadruped' }),
	S('horse z5', 'h', '#8a5a30', { t: 'bs_przewalski_s_horse', gr: 'mam', z: 5 }),
	S('zebra z5', 'Z', '#e8e8e0', { t: 'bs_plains_zebra', gr: 'mam', z: 5 }),
	S('camel z5', 'C', '#c8a870', { t: 'bs_dromedary', gr: 'mam', z: 5 }),
	S('boar', 'q', '#7a5a3a', { t: 'boar', b: 'quadruped' }),
	S('bat', 'b', '#6a6070', { t: 'bat', b: 'avian' }),
	S('mammoth', 'M', '#8a5a30', { t: 'mammoth', b: 'quadruped' }),
	S('elephant z6', 'E', '#9a9a98', { t: 'bs_asian_elephant', gr: 'mam', z: 6 }),
	S('gorilla z5', 'G', '#4a4a4a', { t: 'bs_western_gorilla', gr: 'mam', z: 5 }),
	S('orca z6', 'O', '#3a3a48', { t: 'bs_orca', gr: 'mam', z: 6, b: 'fish' }),
	// birds
	S('robin z1', 'v', '#8a6a4a', { t: 'bs_european_robin', gr: 'bird', z: 1 }),
	S('raven z2', 'c', '#3a3a48', { t: 'bs_common_raven', gr: 'bird', z: 2 }),
	S('eagle z4', 'E', '#8a5a30', { t: 'bs_golden_eagle', gr: 'bird', z: 4 }),
	S('roc', 'R', '#d8b040', { t: 'roc', b: 'avian' }),
	S('ostrich z5', 'O', '#5a4a40', { t: 'bs_common_ostrich', gr: 'bird', z: 5 }),
	S('phoenix', 'P', '#e88a30', { t: 'phoenix', b: 'avian' }),
	S('pteranodon', 'P', '#a89878', { t: 'bs_pteranodon', gr: 'ptero', z: 4 }),
	// reptiles, amphibians
	S('lizard z2', 'l', '#6a9a4a', { t: 'bs_sand_lizard', gr: 'rept', z: 2 }),
	S('salamander', 'L', '#d84a3a', { t: 'salamander', b: 'quadruped' }),
	S('snake', 's', '#5a9a4a', { t: 'snake', b: 'serpent' }),
	S('cobra z3', 'S', '#a8883a', { t: 'bs_indian_cobra', gr: 'rept', z: 3 }),
	S('croc', 'K', '#4a6a3a', { t: 'croc', b: 'quadruped' }),
	S('tortoise z3', 't', '#8a7a4a', { t: 'bs_leopard_tortoise', gr: 'rept', z: 3 }),
	S('toad', 'u', '#5a7a3a', { t: 'toad', b: 'quadruped' }),
	S('dart frog z1', 'f', '#3a8ad8', { t: 'bs_strawberry_poison_dart_frog', gr: 'amph', z: 1 }),
	// dinosaurs
	S('velociraptor z3', 'v', '#a8703a', { t: 'bs_velociraptor', gr: 'dino', z: 3 }),
	S('deinonychus z4', 'd', '#6a7a4a', { t: 'bs_deinonychus', gr: 'dino', z: 4 }),
	S('t. rex z6', 'T', '#6a5a3a', { t: 'bs_tyrannosaurus', gr: 'dino', z: 6 }),
	S('iguanodon z6', 'I', '#7a8a5a', { t: 'bs_iguanodon', gr: 'dino', z: 6 }),
	S('brachiosaurus z7', 'B', '#7a8a7a', { t: 'bs_brachiosaurus', gr: 'dino', z: 7 }),
	S('stegosaurus z6', 'S', '#6a7a4a', { t: 'bs_stegosaurus', gr: 'dino', z: 6 }),
	S('triceratops z6', 'T', '#8a6a4a', { t: 'bs_triceratops', gr: 'dino', z: 6 }),
	// sea
	S('trout z2', 'u', '#8a9a6a', { t: 'bs_brown_trout', gr: 'fish', z: 2 }),
	S('clownfish z1', 'u', '#e88a30', { t: 'bs_clownfish', gr: 'fish', z: 1 }),
	S('shark', 'F', '#8a9098', { t: 'shark', b: 'fish' }),
	S('eel', 'e', '#c8b040', { t: 'eel', b: 'serpent' }),
	S('octopus z2', 'o', '#c86a5a', { t: 'bs_mimic_octopus', gr: 'moll', z: 2 }),
	S('kraken', 'X', '#a04a9a', { t: 'kraken', b: 'tentacled' }),
	S('crab', 'k', '#c84a3a', { t: 'crab', b: 'insect' }),
	S('moon jelly z1', 'j', '#a0c8e8', { t: 'bs_moon_jellyfish', gr: 'cnid', z: 1 }),
	S('cloudjelly', 'J', '#c060c0', { t: 'cloudjelly', b: 'tentacled' }),
	// insects, arachnids
	S('beetle', 'b', '#3a7a3a', { t: 'beetle', b: 'insect' }),
	S('army ant z1', 'a', '#8a3a2a', { t: 'bs_army_ant', gr: 'ins', z: 1 }),
	S('hornet z1', 'b', '#d8a030', { t: 'bs_asian_giant_hornet', gr: 'ins', z: 1 }),
	S('shardmoth', 'v', '#70d8e0', { t: 'shardmoth', b: 'avian' }),
	S('spider', 'x', '#8a3a8a', { t: 'spider', b: 'insect' }),
	S('scorpion', 'S', '#d88a3a', { t: 'scorpion', b: 'insect' }),
	S('centipede', 'j', '#d88a3a', { t: 'centipede', b: 'insect' }),
	// slimes, small slow things
	S('ooze', 'o', '#5aaa4a', { t: 'ooze', b: 'ooze' }),
	S('snail z1', 'c', '#a8784a', { t: 'bs_roman_snail', gr: 'moll', z: 1 }),
	S('mimic', '■', '#c8803a', { t: 'mimic', b: 'ooze' }),
	S('sandworm', 'W', '#c8a040', { t: 'sandworm', b: 'serpent' }),
	S('leech', 'l', '#8a2a2a', { t: 'leech', b: 'serpent' }),
	// undead
	S('skeleton', 'z', '#e0dcd0', { t: 'skeleton', b: 'humanoid' }),
	S('lich', 'L', '#c0a0e0', { t: 'lich', b: 'humanoid' }),
	S('phantom', 'p', '#90d8e0', { t: 'phantom', b: 'humanoid' }),
	S('ashwraith', 'W', '#707070', { t: 'ashwraith', b: 'humanoid' }),
	S('ghoul', 'Z', '#7a9a5a', { t: 'ghoul', b: 'humanoid' }),
	S('mummy', 'z', '#c8b070', { t: 'mummy', b: 'humanoid' }),
	// constructs, spirits, plants
	S('golem', 'g', '#909090', { t: 'golem', b: 'golem' }),
	S('crystal golem', 'G', '#c070d0', { t: 'crystalgolem', b: 'golem' }),
	S('magma', 'E', '#e8803a', { t: 'magma', b: 'golem' }),
	S('wisp', 'w', '#70d8e0', { t: 'wisp', b: 'ooze' }),
	S('treant', '♣', '#3a7a3a', { t: 'treant', b: 'golem' }),
	S('sporefolk', 'm', '#c060c0', { t: 'sporefolk', b: 'humanoid' }),
	// monsters
	S('wyvern', 'W', '#4a7a4a', { t: 'wyvern', b: 'avian' }),
	S('dragon', 'D', '#b83a2a', { t: 'red dragon' }),
	S('elder dragon', 'D', '#d88a28', { t: 'elder dragon' }),
	S('hydra', 'H', '#3a7a4a', { t: 'hydra', b: 'saurian' }),
	S('griffin', 'G', '#c89a40', { t: 'griffin', b: 'avian' }),
	S('basilisk', 'B', '#4a7a3a', { t: 'basilisk', b: 'saurian' }),
	S('mantis', 'm', '#5a9a4a', { t: 'giant mantis', gr: 'ins', z: 3 }),
	S('leviathan', 'L', '#3a6a9a', { t: 'leviathan', b: 'fish', z: 7 }),
	S('death knight', 'K', '#7a7a8a', { t: 'death knight', b: 'humanoid' }),
	S('beholder', 'X', '#8a3a9a', { t: 'beholder', b: 'tentacled' }),
	S('dire wolf', 'C', '#7a7a82', { t: 'dire wolf', b: 'quadruped' }),
	S('imp', 'i', '#c84a3a', { t: 'imp', b: 'avian' }),
	S('gargoyle', 'Y', '#909090', { t: 'gargoyle', b: 'avian' }),
	S('horror', 'H', '#a04a9a', { t: 'horror', b: 'tentacled' }),
	S('yeti', 'Y', '#e8e8f0', { t: 'yeti', b: 'humanoid' }),
	S('troll', 'T', '#5a7a4a', { nm: 'cave troll', b: 'humanoid' }),
	// fallbacks by body plan / glyph
	S('quadruped ?', 'Q', '#a08060', { t: 'oddbeast', b: 'quadruped' }),
	S('glyph x', 'x', '#8a8a8a', {}),
];
