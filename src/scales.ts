// Scales, rhythmic feels and culture/genre/mood instrument palettes for the procedural soundtrack.
// Pure data (instrument names are keys into instruments.ts), so it costs nothing at runtime.

/** semitone offsets; fractions are quarter-tone-ish maqam / gamelan approximations */
export const SCALES: Record<string, number[]> = {
	ionian: [0, 2, 4, 5, 7, 9, 11],
	mixolydian: [0, 2, 4, 5, 7, 9, 10],
	lydian: [0, 2, 4, 6, 7, 9, 11],
	dorian: [0, 2, 3, 5, 7, 9, 10],
	aeolian: [0, 2, 3, 5, 7, 8, 10],
	phrygian: [0, 1, 3, 5, 7, 8, 10],
	locrian: [0, 1, 3, 5, 6, 8, 10],
	wholetone: [0, 2, 4, 6, 8, 10],
	harmonicMinor: [0, 2, 3, 5, 7, 8, 11],
	phrygDom: [0, 1, 4, 5, 7, 8, 10],
	minPent: [0, 3, 5, 7, 10],
	majPent: [0, 2, 4, 7, 9],
	// east / south-east asia
	pelog: [0, 1, 3, 7, 8],
	slendro: [0, 2.4, 4.8, 7.2, 9.6],
	hirajoshi: [0, 2, 3, 7, 8],
	insen: [0, 1, 5, 7, 10],
	yo: [0, 2, 5, 7, 9],
	ryukyu: [0, 4, 5, 7, 11],
	shang: [0, 2, 5, 7, 10],
	// near east (maqam-like)
	hijaz: [0, 1, 4, 5, 7, 8, 10],
	rast: [0, 2, 3.5, 5, 7, 9, 10.5],
	bayati: [0, 1.5, 3, 5, 7, 8, 10],
	homayun: [0, 1.5, 4, 5, 7, 8, 10],
	// raga-like
	bhairav: [0, 1, 4, 5, 7, 8, 11],
	todi: [0, 1, 3, 6, 7, 8, 11],
	marwa: [0, 1, 4, 6, 9, 11],
	// others
	ukrDorian: [0, 2, 3, 6, 7, 9, 10],
	tizita: [0, 2, 3, 7, 9],
	ambassel: [0, 1, 5, 7, 8],
	bati: [0, 4, 5, 7, 11],
	nahua: [0, 2, 5, 7, 8],
	pentachord: [0, 2, 4, 5, 7],
	hexa: [0, 2, 4, 5, 7, 9],
	octatonic: [0, 1, 3, 4, 6, 7, 9, 10],
	uncanny: [0, 1, 3.5, 5.5, 7, 8, 10.5],
};

/**
 * Rhythmic feels in 16th-note steps. `groups` are the beat groups that make up a bar
 * (4,4,4,4 = 4/4; 6,6 = 6/8; 4,4,6 = 7/8 aksak; 6,6,6,6 = 12/8; 6,6,4 = tresillo 3+3+2).
 */
export interface Feel { spb: number; groups: number[]; swing: number }
export const FEELS: Record<string, Feel> = {
	four: { spb: 16, groups: [4, 4, 4, 4], swing: 0 },
	swing: { spb: 16, groups: [4, 4, 4, 4], swing: 0.3 },
	waltz: { spb: 12, groups: [4, 4, 4], swing: 0 },
	six8: { spb: 12, groups: [6, 6], swing: 0 },
	nine8: { spb: 18, groups: [6, 6, 6], swing: 0 },
	seven8: { spb: 14, groups: [4, 4, 6], swing: 0 },
	twelve8: { spb: 24, groups: [6, 6, 6, 6], swing: 0 },
	tresillo: { spb: 16, groups: [6, 6, 4], swing: 0 },
	five8: { spb: 10, groups: [6, 4], swing: 0 },
};

export interface Profile {
	scales: string[];
	feels: string[];
	leads: string[];
	plucks: string[];
	bass: string;
	kit: string;
	/** sustained tonic drone: none, tonic, or fifth (tanpura / pipes / throat) */
	drone: 'none' | 'tonic' | 'fifth';
	tempo: number;
	pad?: string;
	feastLeads?: string[];
	feastKit?: string;
	feastFeel?: string;
	lament?: string;
}

const P = (p: Profile): Profile => p;
export const PROFILES: Record<string, Profile> = {
	sinic: P({ scales: ['majPent', 'shang', 'yo'], feels: ['four', 'four', 'six8'], leads: ['erhu', 'flute', 'koto'], plucks: ['koto', 'harp'], bass: 'lute', kit: 'taiko', drone: 'none', tempo: 0.95, feastLeads: ['flute', 'erhu'], feastKit: 'taiko', feastFeel: 'four', lament: 'erhu' }),
	japonic: P({ scales: ['hirajoshi', 'insen', 'yo', 'ryukyu'], feels: ['four', 'six8', 'four'], leads: ['shaku', 'koto', 'flute'], plucks: ['koto'], bass: 'lute', kit: 'taiko', drone: 'none', tempo: 0.92, feastLeads: ['flute', 'koto'], feastKit: 'taiko', feastFeel: 'six8', lament: 'shaku' }),
	indic: P({ scales: ['bhairav', 'todi', 'dorian', 'mixolydian', 'lydian', 'marwa'], feels: ['four', 'seven8', 'six8', 'four'], leads: ['sitar', 'flute', 'shawm'], plucks: ['sitar', 'harp'], bass: 'lute', kit: 'tabla', drone: 'fifth', tempo: 0.95, feastLeads: ['shawm', 'sitar'], feastKit: 'tabla', feastFeel: 'six8', lament: 'shawm' }),
	sea: P({ scales: ['pelog', 'slendro'], feels: ['four', 'six8'], leads: ['gamelan', 'flute', 'gamelan'], plucks: ['gamelan', 'kalimba'], bass: 'lute', kit: 'gong', drone: 'none', tempo: 0.93, pad: 'sine', feastLeads: ['gamelan'], feastKit: 'gong', feastFeel: 'four', lament: 'flute' }),
	steppe: P({ scales: ['yo', 'minPent', 'dorian'], feels: ['six8', 'four'], leads: ['erhu', 'shawm', 'flute'], plucks: ['lute'], bass: 'lute', kit: 'frame', drone: 'fifth', tempo: 1.0, feastLeads: ['erhu'], feastKit: 'frame', feastFeel: 'six8', lament: 'erhu' }),
	persian: P({ scales: ['bayati', 'rast', 'homayun', 'hijaz'], feels: ['seven8', 'six8', 'four'], leads: ['flute', 'violin', 'lute'], plucks: ['lute', 'mandolin'], bass: 'lute', kit: 'frame', drone: 'tonic', tempo: 0.97, feastLeads: ['violin', 'shawm'], feastKit: 'darbuka', feastFeel: 'six8', lament: 'duduk' }),
	arabic: P({ scales: ['hijaz', 'rast', 'bayati', 'phrygian', 'aeolian'], feels: ['four', 'six8', 'swing'], leads: ['lute', 'shawm', 'violin', 'flute'], plucks: ['lute', 'harp'], bass: 'lute', kit: 'darbuka', drone: 'tonic', tempo: 1.0, feastLeads: ['shawm', 'violin'], feastKit: 'darbuka', feastFeel: 'six8', lament: 'duduk' }),
	mande: P({ scales: ['majPent', 'ionian', 'mixolydian'], feels: ['twelve8', 'six8'], leads: ['kora', 'marimba', 'kora'], plucks: ['kora'], bass: 'lute', kit: 'djembe', drone: 'none', tempo: 1.0, feastLeads: ['kora', 'marimba'], feastKit: 'djembe', feastFeel: 'twelve8', lament: 'flute' }),
	yoruba: P({ scales: ['majPent', 'dorian', 'mixolydian'], feels: ['twelve8', 'six8'], leads: ['kalimba', 'flute', 'marimba'], plucks: ['kalimba', 'marimba'], bass: 'lute', kit: 'djembe', drone: 'none', tempo: 1.0, feastLeads: ['marimba', 'kalimba'], feastKit: 'djembe', feastFeel: 'twelve8', lament: 'flute' }),
	bantu: P({ scales: ['ionian', 'majPent', 'mixolydian'], feels: ['twelve8', 'six8', 'swing'], leads: ['kalimba', 'marimba'], plucks: ['marimba', 'kalimba'], bass: 'lute', kit: 'djembe', drone: 'none', tempo: 1.0, feastLeads: ['marimba'], feastKit: 'djembe', feastFeel: 'twelve8', lament: 'flute' }),
	ethiopic: P({ scales: ['tizita', 'ambassel', 'bati', 'minPent'], feels: ['six8', 'twelve8'], leads: ['violin', 'lute', 'flute'], plucks: ['harp', 'lute'], bass: 'lute', kit: 'frame', drone: 'none', tempo: 0.97, feastLeads: ['violin', 'flute'], feastKit: 'frame', feastFeel: 'six8', lament: 'violin' }),
	nahua: P({ scales: ['nahua', 'minPent', 'pentachord'], feels: ['four', 'six8'], leads: ['flute', 'whistle'], plucks: ['marimba'], bass: 'lute', kit: 'taiko', drone: 'tonic', tempo: 0.93, feastLeads: ['whistle', 'flute'], feastKit: 'taiko', feastFeel: 'six8', lament: 'whistle' }),
	maya: P({ scales: ['minPent', 'nahua', 'dorian'], feels: ['four', 'seven8'], leads: ['whistle', 'marimba'], plucks: ['marimba'], bass: 'lute', kit: 'taiko', drone: 'tonic', tempo: 0.93, feastLeads: ['whistle', 'marimba'], feastKit: 'taiko', feastFeel: 'four', lament: 'whistle' }),
	andean: P({ scales: ['minPent', 'minPent', 'aeolian'], feels: ['tresillo', 'six8', 'four'], leads: ['panpipe', 'shaku', 'flute'], plucks: ['mandolin'], bass: 'lute', kit: 'bodhran', drone: 'none', tempo: 0.98, feastLeads: ['panpipe', 'mandolin'], feastKit: 'bodhran', feastFeel: 'tresillo', lament: 'panpipe' }),
	turtle: P({ scales: ['minPent', 'majPent', 'yo'], feels: ['four'], leads: ['flute', 'whistle'], plucks: ['harp'], bass: 'lute', kit: 'frame', drone: 'tonic', tempo: 0.88, feastLeads: ['flute'], feastKit: 'frame', feastFeel: 'four', lament: 'flute' }),
	inuit: P({ scales: ['pentachord', 'shang', 'minPent'], feels: ['four', 'five8'], leads: ['choir', 'duduk', 'whistle'], plucks: ['harp'], bass: 'lute', kit: 'frame', drone: 'fifth', tempo: 0.85, pad: 'choir', feastLeads: ['choir'], feastKit: 'frame', feastFeel: 'four', lament: 'choir' }),
	polynesian: P({ scales: ['majPent', 'hexa', 'ionian'], feels: ['six8', 'four'], leads: ['flute', 'mandolin'], plucks: ['mandolin', 'kalimba'], bass: 'lute', kit: 'taiko', drone: 'none', tempo: 0.98, feastLeads: ['mandolin', 'flute'], feastKit: 'taiko', feastFeel: 'six8', lament: 'flute' }),
	norse: P({ scales: ['dorian', 'aeolian', 'phrygian'], feels: ['four', 'six8'], leads: ['violin', 'brass', 'flute'], plucks: ['lute', 'harp'], bass: 'lute', kit: 'bodhran', drone: 'fifth', tempo: 0.9, feastLeads: ['violin'], feastKit: 'bodhran', feastFeel: 'waltz', lament: 'violin' }),
	celtic: P({ scales: ['dorian', 'mixolydian', 'aeolian'], feels: ['six8', 'swing', 'nine8'], leads: ['whistle', 'violin', 'bagpipe', 'flute'], plucks: ['harp', 'lute'], bass: 'lute', kit: 'bodhran', drone: 'fifth', tempo: 1.0, feastLeads: ['violin', 'whistle'], feastKit: 'bodhran', feastFeel: 'six8', lament: 'bagpipe' }),
	latin: P({ scales: ['ionian', 'harmonicMinor', 'mixolydian', 'aeolian', 'phrygian'], feels: ['waltz', 'tresillo', 'four', 'six8'], leads: ['violin', 'flute', 'organ'], plucks: ['guitar', 'lute', 'harp'], bass: 'lute', kit: 'frame', drone: 'none', tempo: 1.0, feastLeads: ['violin', 'guitar'], feastKit: 'bodhran', feastFeel: 'waltz', lament: 'violin' }),
	slavic: P({ scales: ['ukrDorian', 'harmonicMinor', 'aeolian', 'dorian'], feels: ['seven8', 'six8', 'four'], leads: ['violin', 'bagpipe', 'mandolin'], plucks: ['mandolin', 'lute'], bass: 'lute', kit: 'frame', drone: 'fifth', tempo: 1.0, feastLeads: ['violin', 'mandolin'], feastKit: 'frame', feastFeel: 'seven8', lament: 'violin' }),
	eldritch: P({ scales: ['wholetone', 'octatonic', 'uncanny', 'locrian'], feels: ['five8', 'seven8'], leads: ['theremin', 'choir', 'bell'], plucks: ['bell', 'musicbox'], bass: 'lute', kit: 'timpani', drone: 'tonic', tempo: 0.85, pad: 'wet', feastLeads: ['theremin'], feastKit: 'timpani', feastFeel: 'seven8', lament: 'choir' }),
};

/** instrument fall-backs when the land has no known culture sphere */
export const GENERIC: Profile = P({ scales: [], feels: ['four'], leads: ['lute', 'flute', 'bell', 'violin'], plucks: ['lute', 'harp', 'guitar'], bass: 'lute', kit: 'frame', drone: 'none', tempo: 1.0, feastLeads: ['violin', 'flute'], feastKit: 'bodhran', feastFeel: 'six8', lament: 'duduk' });

/** a genre colours any land: chance-weighted instrument overrides and a texture bed */
export interface GenreColour {
	lead?: string; leadW?: number;
	harm?: string; harmW?: number;
	pad?: string; padW?: number;
	bass?: string; bassW?: number;
	kit?: string; kitW?: number;
	feels?: string[]; feelW?: number;
	scales?: string[]; scaleW?: number;
	texture?: 'geiger' | 'steam' | 'spore' | 'static' | 'hum';
	room?: number; // extra reverb
	spring?: number; // spring reverb send (atompunk)
	delay?: number;
}
export const GENRE_COLOUR: Record<string, GenreColour> = {
	fantasy: {},
	mythic: { pad: 'choir', padW: 0.5, lead: 'bell', leadW: 0.15, room: 0.1 },
	steampunk: { lead: 'calliope', leadW: 0.5, harm: 'harpsi', harmW: 0.25, feels: ['waltz', 'six8'], feelW: 0.4, texture: 'steam' },
	dieselpunk: { lead: 'brass', leadW: 0.55, kit: 'brush', kitW: 0.55, feels: ['swing'], feelW: 0.7, bass: 'lute' },
	atompunk: { lead: 'theremin', leadW: 0.6, harm: 'vibes', harmW: 0.6, feels: ['swing', 'four'], feelW: 0.4, spring: 0.4 },
	cyberpunk: { lead: 'synthlead', leadW: 0.6, harm: 'synthlead', harmW: 0.4, bass: 'synthbass', bassW: 0.9, kit: 'electro', kitW: 0.85, texture: 'hum' },
	biopunk: { pad: 'wet', padW: 1, lead: 'duduk', leadW: 0.35, texture: 'spore', room: 0.15 },
	solarpunk: { lead: 'kalimba', leadW: 0.5, harm: 'kalimba', harmW: 0.5, pad: 'sine', padW: 0.6 },
	wasteland: { lead: 'harmonica', leadW: 0.45, harm: 'guitar', harmW: 0.6, kit: 'tincan', kitW: 0.75, feels: ['swing', 'four'], feelW: 0.5, scales: ['minPent', 'mixolydian', 'phrygian', 'dorian'], scaleW: 0.5, texture: 'geiger' },
	twilight: { lead: 'musicbox', leadW: 0.4, harm: 'harpsi', harmW: 0.6, pad: 'organ', padW: 0.85, kit: 'timpani', kitW: 0.6, feels: ['waltz', 'nine8'], feelW: 0.5, scales: ['harmonicMinor', 'aeolian', 'phrygian', 'dorian'], scaleW: 0.5, room: 0.3 },
};

/** a mood nudges the lead choice too (chance `w`) */
export const MOOD_LEADS: Record<string, { leads: string[]; w: number }> = {
	sky: { leads: ['bell', 'musicbox', 'flute', 'harp'], w: 0.55 },
	crypt: { leads: ['organ', 'choir', 'duduk', 'bell'], w: 0.6 },
	underdark: { leads: ['theremin', 'choir', 'duduk', 'bell'], w: 0.45 },
	dungeon: { leads: ['duduk', 'choir', 'lute'], w: 0.35 },
	tower: { leads: ['bell', 'harp', 'vibes', 'flute'], w: 0.5 },
	sea: { leads: ['flute', 'duduk', 'harp'], w: 0.4 },
};
