package engine

import (
	"fmt"
	"math"
)

// ---- weather and seasons ----
//
// Everything here is derived from the world seed, the parasang and the game clock, so the
// schedule is deterministic and needs almost nothing in the save. Weather rolls never touch g.R:
// they hash (world seed, region, 4-hour bucket); per-minute effects (dousing, lightning) hash
// (seed, turn, cell). Weather only acts outdoors: z == 0, not underwater, not a dark zone, and
// not on a roofed tile (floors, carpets, beds inside buildings).

const (
	DayLen       = 1440 // turns (minutes) per day
	SeasonDays   = 28
	YearDays     = SeasonDays * 4
	wxBucket     = 240 // weather holds for 4 hours
	wxRegion     = 4   // parasangs per weather region side
	startMinutes = 480 // turn 0 is 8 in the morning (see timeOfDay)
)

var SeasonNames = [4]string{"Spring", "Summer", "Autumn", "Winter"}

// WeatherKind is one kind of weather; all its text and effects live in this table.
type WeatherKind struct {
	ID      string
	Name    string
	Start   string // logged when it sets in where you stand
	Color   byte
	Sight   int // caps the field-of-view radius outdoors (0 = no cap)
	Wet     int // 0..3: chance to douse fires and burning creatures each minute
	Missile int // to-hit penalty on shots
	Slow    int // extra % move cost on foot
	Hurt    int // damage per hour of exposure (acid rain, fallout)
	Thirst  bool
	Bolt    bool   // lightning strikes
	Overlay string // rain storm snow fog dust ash spore none
	Glyph   rune
	GlyphF  byte
}

var weatherKinds = []WeatherKind{
	{ID: "clear", Name: "Clear", Start: "The sky clears.", Color: CCyan, Overlay: "none"},
	{ID: "overcast", Name: "Overcast", Start: "Clouds roll in and the light goes flat.", Color: CGray, Overlay: "none"},
	{ID: "rain", Name: "Rain", Start: "{{B|Rain begins to fall.}}", Color: CBlue, Sight: 20, Wet: 2, Missile: 1, Overlay: "rain", Glyph: '\'', GlyphF: CBlue},
	{ID: "storm", Name: "Thunderstorm", Start: "{{B|Thunder rolls: a storm breaks overhead.}}", Color: CBlue, Sight: 13, Wet: 3, Missile: 4, Slow: 10, Bolt: true, Overlay: "storm", Glyph: '/', GlyphF: CBlue},
	{ID: "snow", Name: "Snow", Start: "{{Y|Snow starts to drift down.}}", Color: CWhite, Sight: 14, Wet: 1, Missile: 1, Slow: 20, Overlay: "snow", Glyph: '*', GlyphF: CWhite},
	{ID: "blizzard", Name: "Blizzard", Start: "{{Y|A blizzard howls in, white and blinding.}}", Color: CWhite, Sight: 7, Wet: 1, Missile: 4, Slow: 45, Overlay: "snow", Glyph: '*', GlyphF: CWhite},
	{ID: "fog", Name: "Fog", Start: "{{y|A thick fog creeps in.}}", Color: CGray, Sight: 7, Overlay: "fog"},
	{ID: "sandstorm", Name: "Sandstorm", Start: "{{W|A wall of sand sweeps over you.}}", Color: CYellow, Sight: 5, Missile: 3, Slow: 15, Thirst: true, Overlay: "dust", Glyph: '∙', GlyphF: CYellow},
	{ID: "ashfall", Name: "Ash fall", Start: "{{K|Grey ash sifts down from a smouldering sky.}}", Color: CDarkGray, Sight: 11, Missile: 1, Thirst: true, Overlay: "ash", Glyph: '·', GlyphF: CGray},
	{ID: "spores", Name: "Spore drift", Start: "{{m|Clouds of spores drift between the stalks.}}", Color: CMagenta, Sight: 11, Overlay: "spore", Glyph: ',', GlyphF: CDarkMagenta},
	{ID: "shimmer", Name: "Prism haze", Start: "{{M|The air fills with glittering crystal motes.}}", Color: CMagenta, Sight: 12, Missile: 2, Overlay: "spore", Glyph: '·', GlyphF: CMagenta},
	{ID: "heat", Name: "Scorching sun", Start: "{{O|The sun beats down mercilessly.}}", Color: COrange, Thirst: true, Overlay: "none"},
	{ID: "frost", Name: "Hard frost", Start: "{{C|A bitter, still cold settles in.}}", Color: CCyan, Overlay: "none"},
	// genre variants (swapped in by genreWeather)
	{ID: "acidrain", Name: "Acid rain", Start: "{{G|Acid rain hisses on every surface.}}", Color: CGreen, Sight: 18, Wet: 2, Missile: 1, Hurt: 1, Overlay: "rain", Glyph: '\'', GlyphF: CGreen},
	{ID: "smog", Name: "Smog", Start: "{{K|Choking smog rolls through, thick with soot.}}", Color: CBrown, Sight: 9, Missile: 1, Thirst: true, Overlay: "fog"},
	{ID: "fallout", Name: "Fallout drizzle", Start: "{{G|A warm, faintly glowing drizzle begins.}}", Color: CGreen, Sight: 20, Wet: 2, Missile: 1, Hurt: 1, Overlay: "rain", Glyph: '\'', GlyphF: CGreen},
	{ID: "sporerain", Name: "Spore rain", Start: "{{m|Sticky spore-laden rain patters down.}}", Color: CMagenta, Sight: 18, Wet: 2, Missile: 1, Overlay: "rain", Glyph: '\'', GlyphF: CMagenta},
	{ID: "pollen", Name: "Pollen drift", Start: "{{W|Golden pollen drifts on a warm breeze.}}", Color: CYellow, Overlay: "spore", Glyph: '·', GlyphF: CYellow},
	{ID: "tempest", Name: "Tempest of the Sky-Father", Start: "{{B|The Sky-Father's tempest breaks: thunder answers thunder.}}", Color: CBlue, Sight: 13, Wet: 3, Missile: 4, Slow: 10, Bolt: true, Overlay: "storm", Glyph: '/', GlyphF: CBlue},
	{ID: "radstorm", Name: "Rad storm", Start: "{{G|A rad storm rolls in, the sky crawling green and the air tasting of metal.}}", Color: CGreen, Sight: 10, Wet: 1, Missile: 3, Slow: 10, Hurt: 1, Overlay: "dust", Glyph: '∙', GlyphF: CGreen},
	{ID: "ashrain", Name: "Ash rain", Start: "{{K|Grey rain falls thick with ash and drags soot down the walls.}}", Color: CDarkGray, Sight: 14, Wet: 2, Missile: 1, Overlay: "ash", Glyph: '\'', GlyphF: CGray},
	{ID: "bloodmoon", Name: "Blood moon", Start: "{{r|The moon swells and bleeds red. Something far off howls.}}", Color: CRed, Sight: 14, Missile: 1, Overlay: "spore", Glyph: '·', GlyphF: CDarkRed},
	{ID: "miasma", Name: "Miasma", Start: "{{y|A sweet, rotten miasma seeps up from the drains.}}", Color: CDarkGreen, Sight: 7, Hurt: 1, Overlay: "fog"},
	{ID: "steamfog", Name: "Coal fog", Start: "{{K|A yellow coal fog settles over everything.}}", Color: CBrown, Sight: 8, Missile: 1, Overlay: "fog"},
}

// genreWeather swaps base kinds for genre-flavoured ones some of the time.
var genreWeather = map[string][]struct {
	From, To string
	P        float64
}{
	"cyberpunk":  {{"rain", "acidrain", 0.6}, {"overcast", "smog", 0.35}, {"fog", "smog", 0.5}},
	"dieselpunk": {{"overcast", "smog", 0.5}, {"fog", "smog", 0.6}},
	"steampunk":  {{"fog", "steamfog", 0.7}, {"overcast", "steamfog", 0.3}},
	"atompunk":   {{"rain", "fallout", 0.25}},
	"biopunk":    {{"rain", "sporerain", 0.5}, {"fog", "spores", 0.4}},
	"solarpunk":  {{"clear", "pollen", 0.25}},
	"mythic":     {{"storm", "tempest", 0.5}},
	"wasteland":  {{"storm", "radstorm", 0.7}, {"rain", "fallout", 0.3}, {"sandstorm", "radstorm", 0.3}, {"overcast", "ashrain", 0.12}},
	"twilight":   {{"fog", "miasma", 0.5}, {"clear", "bloodmoon", 0.15}, {"overcast", "ashrain", 0.3}, {"rain", "ashrain", 0.3}},
}

var wxByID = func() map[string]*WeatherKind {
	m := map[string]*WeatherKind{}
	for i := range weatherKinds {
		m[weatherKinds[i].ID] = &weatherKinds[i]
	}
	return m
}()

func WeatherByID(id string) *WeatherKind { return wxByID[id] }

// Weather is the weather at one parasang and minute.
type Weather struct {
	Kind   *WeatherKind
	Temp   int // °C
	Season int
}

// per-biome climate: humidity (0..1) and fallback temperature when the world file has none
var biomeHumid = [17]float64{0.45, 0.06, 0.12, 0.3, 0.35, 0.5, 0.45, 0.65, 0.6, 0.4, 0.3, 0.3, 0.55, 0.3, 0.25, 0.45, 0.15}
var biomeTemp = [17]int{14, 30, 2, 26, 14, 25, 12, 26, 10, 0, -6, -15, 14, 18, 8, 12, 35}

// DayOf is the 0-based calendar day of a turn; SeasonOfDay its season (0 spring … 3 winter).
func DayOf(turn int) int { return (turn + startMinutes) / DayLen }
func SeasonOfDay(day int) int {
	if day < 0 {
		day = 0
	}
	return (day / SeasonDays) % 4
}

func (g *Game) Day() int    { return DayOf(g.Turn) }
func (g *Game) Season() int { return SeasonOfDay(g.Day()) }

func wxMix(h uint64) uint64 {
	h = (h ^ (h >> 30)) * 0xbf58476d1ce4e5b9
	h = (h ^ (h >> 27)) * 0x94d049bb133111eb
	return h ^ (h >> 31)
}

func wxHash(seed uint64, a, b int) float64 {
	h := wxMix(seed ^ uint64(a)*0x9e3779b97f4a7c15 ^ uint64(b)*0xc2b2ae3d27d4eb4f)
	return float64(h>>11) / float64(1<<53)
}

func (w *WorldData) wxSeed() uint64 { return HashStr(w.Seed + ":weather") }

func (w *WorldData) biomeAt(p int) int {
	if p < 0 || p >= len(w.Biome) {
		return 4
	}
	b := w.Biome[p]
	if b < 0 || b >= len(biomeHumid) {
		return 4
	}
	return b
}

// TempAt: the parasang's base temperature, swung by season and the hour of day.
func (w *WorldData) TempAt(p, turn int) int {
	b := w.biomeAt(p)
	base := float64(biomeTemp[b])
	if len(w.Temp) == len(w.Biome) && p >= 0 && p < len(w.Temp) {
		base = float64(w.Temp[p])
	}
	amp := 5.0
	switch {
	case b == 5 || b == 7:
		amp = 2
	case b == 0:
		amp = 3
	}
	if base < 8 && b != 0 {
		amp += 4
	}
	if b == 2 || b == 9 || b == 10 || b == 11 {
		amp += 3
	}
	if p >= 0 && p < len(w.Height) && w.Height[p] >= 70 {
		amp += 2
	}
	minutes := turn + startMinutes
	frac := float64(minutes%(YearDays*DayLen)) / float64(YearDays*DayLen)
	season := math.Sin(2 * math.Pi * (frac - 0.125)) // peaks mid-summer, troughs mid-winter
	hour := float64(minutes%DayLen) / 60
	diurnal := 3.0
	if b == 1 || b == 2 {
		diurnal = 8
	}
	day := math.Cos(2 * math.Pi * (hour - 15) / 24)
	jitter := (wxHash(w.wxSeed()^0x7e, p, minutes/DayLen) - 0.5) * 4
	return int(math.Round(base + amp*season + diurnal*day + jitter))
}

// WeatherAt is the deterministic weather schedule: one roll per region and 4-hour bucket, read
// through the parasang's own climate (humidity, temperature, biome).
func (w *WorldData) WeatherAt(p, turn int) Weather {
	if turn < 0 {
		turn = 0
	}
	bucket := turn / wxBucket
	t0 := bucket * wxBucket // the kind is fixed for the whole bucket
	season := SeasonOfDay(DayOf(turn))
	wx := Weather{Temp: w.TempAt(p, turn), Season: season}
	b := w.biomeAt(p)
	px, py := 0, 0
	if w.W > 0 {
		px, py = p%w.W, p/w.W
	}
	rx, ry := px/wxRegion, py/wxRegion
	s := w.wxSeed()
	// fronts drift eastward: time and region share one smooth noise field
	wet := noise2(s, float64(bucket)*0.35-float64(rx)*0.5, float64(ry)*0.6)
	wind := noise2(s+7, float64(bucket)*0.5-float64(rx)*0.4, float64(ry)*0.5+3.3)
	fogR := wxHash(s^0xf06, rx*977+ry, bucket)
	swapR := wxHash(s^0x5a9, rx*977+ry, bucket)
	hum := biomeHumid[b]
	switch season {
	case 0:
		hum += 0.05
	case 1:
		if b == 3 || b == 5 || b == 7 {
			hum += 0.15 // monsoon
		} else {
			hum -= 0.05
		}
	case 2:
		hum += 0.08
	}
	temp := w.TempAt(p, t0)
	hour := ((t0 + startMinutes) % DayLen) / 60
	kind := "clear"
	precip := wet > 1-hum*0.8
	heavy := wet > 1-hum*0.4
	switch {
	case (b == 13 || b == 16) && (wind > 0.55 || wet > 0.6):
		kind = "ashfall"
	case (b == 1 || b == 2) && wind > 0.68 && !heavy:
		kind = "sandstorm"
	case precip && temp <= 1:
		kind = "snow"
		if heavy && wind > 0.45 {
			kind = "blizzard"
		}
	case precip && heavy && wind > 0.4:
		kind = "storm"
	case precip:
		kind = "rain"
	case b == 15 && wind > 0.6:
		kind = "spores"
	case b == 14 && wind > 0.62:
		kind = "shimmer"
	case b != 1 && fogR < fogChance(b, hum, hour):
		kind = "fog"
	case wet > 1-hum*0.8-0.18:
		kind = "overcast"
	case temp >= 33:
		kind = "heat"
	case temp <= -12:
		kind = "frost"
	}
	for _, sw := range genreWeather[w.Genre] {
		if sw.From == kind && swapR < sw.P {
			kind = sw.To
			break
		}
	}
	wx.Kind = wxByID[kind]
	return wx
}

func fogChance(b int, hum float64, hour int) float64 {
	c := hum * 0.18
	if b == 12 || b == 8 || b == 0 {
		c += 0.1
	}
	if hour >= 4 && hour <= 9 {
		c *= 2.2 // morning mists
	}
	return c
}

// ---- the player's surroundings ----

// outdoors: weather reaches this zone at all.
func outdoors(z *Zone) bool {
	return z != nil && z.Z == 0 && !z.Underwater && !z.Dark && !z.Sky
}

// roofed tiles: interiors of buildings keep the weather out.
func roofed(t Terrain) bool {
	return t == TFloor || t == TCarpet || t == TBed || t == TSeat || t == TShelf || t == TThrone
}

func (g *Game) exposedAt(x, y int) bool {
	z := g.Cur
	return outdoors(z) && z.In(x, y) && !roofed(z.T[y*MW+x])
}

// WeatherHere is the weather over the player's parasang (the zone's on foot, the cursor-free
// travel position on the world map).
func (g *Game) WeatherHere() Weather {
	if g.wxForce != nil {
		return *g.wxForce
	}
	return g.W.WeatherAt(g.W.Idx(g.WX, g.WY), g.Turn)
}

// activeWeather is the weather that applies to the current zone, nil indoors / underground.
func (g *Game) activeWeather() *Weather {
	if g.OnWorld || !outdoors(g.Cur) {
		return nil
	}
	w := g.WeatherHere()
	if w.Kind == nil {
		return nil
	}
	return &w
}

func (g *Game) wxRoll(salt uint64, i int) float64 {
	return wxHash(g.W.wxSeed()^salt, g.Turn, i)
}

// weatherSight caps the FOV radius on the surface.
func (g *Game) weatherSight(radius int) int {
	if w := g.activeWeather(); w != nil && w.Kind.Sight > 0 && !roofed(g.Cur.At(g.P.X, g.P.Y)) {
		return mini(radius, w.Kind.Sight)
	}
	return radius
}

// weatherMissile is the to-hit penalty for shots fired in the open.
func (g *Game) weatherMissile(a *Actor) int {
	if w := g.activeWeather(); w != nil && g.exposedAt(a.X, a.Y) {
		return w.Kind.Missile
	}
	return 0
}

// weatherStep: snow and storms slow you down; freezing makes ice and puddles slick.
func (g *Game) weatherStep(nx, ny, cost int) int {
	w := g.activeWeather()
	if w == nil || !g.exposedAt(nx, ny) || g.P.Pow("wings") > 0 {
		return cost
	}
	cost += cost * w.Kind.Slow / 100
	t := g.Cur.At(nx, ny)
	if w.Temp <= 0 && (t == TIce || t == TShallow) && g.wxRoll(0x511c, ny*MW+nx) < 0.2 {
		g.Msg("{{C|You slip on the ice and go down hard.}}")
		cost += 1000
	}
	return cost
}

// weatherTick runs once per game minute (from turn()).
func (g *Game) weatherTick() {
	if g.Cur == nil || g.Dead {
		return
	}
	if !outdoors(g.Cur) {
		return
	}
	w := g.WeatherHere()
	if w.Kind == nil {
		return
	}
	if w.Kind.ID != g.WxLast {
		if g.WxLast != "" {
			g.Msg(w.Kind.Start)
		}
		g.WxLast = w.Kind.ID
	}
	z := g.Cur
	k := w.Kind
	p := g.P
	if k.Wet > 0 {
		doused := false
		for i := range z.Fire {
			if z.Fire[i] > 0 && !roofed(z.T[i]) && g.wxRoll(0xd0, i) < float64(k.Wet)*0.12 {
				z.Fire[i] = 0
				doused = true
			}
		}
		if doused {
			g.computeLight(z)
		}
		for _, a := range append([]*Actor{p}, z.Actors...) {
			if a.HP > 0 && a.St("burning") > 0 && g.exposedAt(a.X, a.Y) && g.wxRoll(0xb0, a.Y*MW+a.X) < float64(k.Wet)*0.25 {
				delete(a.Status, "burning")
				if a == p {
					g.Msg("{{B|The " + lower(k.Name) + " puts out the flames on you.}}")
				}
			}
		}
	}
	exposed := g.exposedAt(p.X, p.Y)
	if exposed {
		// cold burns calories, heat and dust dry you out
		if w.Temp < 0 && g.Turn%maxi(2, 4+w.Temp/5) == 0 {
			g.Hunger++
		}
		if (w.Temp >= 30 || k.Thirst) && g.Turn%3 == 0 {
			g.Thirst++
		}
		if k.Hurt > 0 && g.Turn%60 == 0 && p.Pow("carapace") == 0 {
			g.Msg("{{G|The " + lower(k.Name) + " stings your skin.}}")
			g.hurt(p, k.Hurt, nil, lower(k.Name))
		}
	}
	if k.Bolt && g.wxRoll(0xb017, 0) < 1.0/45 {
		g.lightning()
	}
}

func lower(s string) string {
	b := []byte(s)
	for i, c := range b {
		if c >= 'A' && c <= 'Z' {
			b[i] = c + 32
		}
	}
	return string(b)
}

// tall terrain draws the lightning
func tallTerrain(t Terrain) bool {
	return t == TTree || t == TDeadTree || t == TMushroom || t == TPillar || t == TStatue || t == TTrunk
}

// lightning strikes something tall, or now and then a creature standing in the open.
func (g *Game) lightning() {
	g.boltN++
	z := g.Cur
	if g.wxRoll(0xa1, 1) < 0.25 {
		var cand []*Actor
		for _, a := range append([]*Actor{g.P}, z.Actors...) {
			if a.HP > 0 && g.exposedAt(a.X, a.Y) {
				cand = append(cand, a)
			}
		}
		if len(cand) > 0 {
			a := cand[int(g.wxRoll(0xa2, 2)*float64(len(cand)))%len(cand)]
			dmg := 3 + int(g.wxRoll(0xa3, 3)*6)
			if a == g.P {
				g.Msg(fmt.Sprintf("{{Y|Lightning strikes you!}} {{R|%d}} damage.", dmg))
			} else if g.visible(a.X, a.Y) {
				g.Msg("{{Y|Lightning strikes " + g.the(a) + "!}}")
			}
			a.AddSt("stunned", 1)
			g.hurt(a, dmg, nil, "a lightning strike")
			return
		}
	}
	best := -1
	for k := 0; k < 16; k++ {
		i := int(g.wxRoll(0xa4, 10+k)*float64(MW*MH)) % (MW * MH)
		if roofed(z.T[i]) {
			continue
		}
		if best < 0 || tallTerrain(z.T[i]) && !tallTerrain(z.T[best]) {
			best = i
		}
	}
	if best < 0 {
		return
	}
	x, y := best%MW, best/MW
	t := z.T[best]
	if t.D().Burn || tallTerrain(t) {
		z.Fire[best] = uint8(4 + int(g.wxRoll(0xa5, best)*5))
		g.computeLight(z)
	}
	if g.visible(x, y) {
		g.Msg("{{Y|Lightning strikes the " + t.D().Name + "!}}")
	} else if g.wxRoll(0xa6, 4) < 0.3 {
		g.Msg("{{K|Thunder rumbles nearby.}}")
	}
}

// travelWeather: crossing a parasang on the world map in this weather (slower in snow, hungrier
// in the cold, thirstier in the heat). Returns the adjusted travel cost.
func (g *Game) travelWeather(np, cost int) int {
	w := g.W.WeatherAt(np, g.Turn)
	if g.wxForce != nil {
		w = *g.wxForce
	}
	if w.Kind == nil {
		return cost
	}
	if g.P.Pow("wings") == 0 {
		cost += cost * w.Kind.Slow / 100
	}
	if w.Temp < 0 {
		g.Hunger += cost / maxi(2, 4+w.Temp/5)
	}
	if w.Temp >= 30 || w.Kind.Thirst {
		g.Thirst += cost / 3
	}
	if w.Kind.ID != g.WxLast {
		if g.WxLast != "" {
			g.Msg(w.Kind.Start)
		}
		g.WxLast = w.Kind.ID
	}
	return cost
}

// ---- UI ----

func tempWord(t int) string {
	switch {
	case t <= -15:
		return "{{C|bitter}}"
	case t <= 0:
		return "{{C|freezing}}"
	case t <= 8:
		return "{{c|cold}}"
	case t <= 16:
		return "{{y|cool}}"
	case t <= 24:
		return "{{G|mild}}"
	case t <= 31:
		return "{{W|warm}}"
	case t <= 38:
		return "{{O|hot}}"
	}
	return "{{R|scorching}}"
}

func wxCode(c byte) string {
	for k, v := range colorCode {
		if v == c {
			return string(k)
		}
	}
	return "y"
}

// weatherLine is the side-panel line under the time of day.
func (g *Game) weatherLine() string {
	sn := SeasonNames[g.Season()]
	if w := g.activeWeather(); w != nil {
		return fmt.Sprintf("{{c|%s}} · {{%s|%s}} {{K|%d°}}", sn, wxCode(w.Kind.Color), w.Kind.Name, w.Temp)
	}
	return "{{c|" + sn + "}} {{K|· out of the weather}}"
}

// weatherInfo is the world-map info line for a parasang.
func (g *Game) weatherInfo(p int) string {
	w := g.W.WeatherAt(p, g.Turn)
	return fmt.Sprintf("%s: {{%s|%s}}, %s {{K|(%d°)}}", SeasonNames[w.Season], wxCode(w.Kind.Color), w.Kind.Name, tempWord(w.Temp), w.Temp)
}

func floorMod(a, n int) int {
	a %= n
	if a < 0 {
		a += n
	}
	return a
}

// weatherOverlayKind is the kind the map overlay draws this frame (nil = none).
func (g *Game) weatherOverlayKind() *WeatherKind {
	w := g.activeWeather()
	if w == nil || w.Kind.Overlay == "none" {
		return nil
	}
	return w.Kind
}

// overlayTile restyles one visible, open-air tile.
func (g *Game) overlayTile(k *WeatherKind, x, y int, t Terrain, gl rune, f byte, anim int) (rune, byte) {
	if roofed(t) {
		return gl, f
	}
	if k.Overlay == "fog" {
		if d := maxi(abs(x-g.P.X), abs(y-g.P.Y)*2); d > 4 && f != CDarkGray && f != CBlack {
			f = CGray
			if d > 8 {
				f = CDarkGray
			}
		}
		return gl, f
	}
	d := t.D()
	if !d.Walk || d.Water > 0 || t == TVoid || t == TChasm || t == TLava {
		return gl, f
	}
	col, ph, n := x, y, 12
	switch k.Overlay {
	case "rain":
		ph = y - anim
	case "storm":
		col, ph, n = x+(y-anim*2)/2, y-anim*2, 7
	case "snow":
		col, ph, n = x-anim/6, y-anim/3, 15
	case "dust":
		col, ph, n = y, x-anim*2, 9
	case "ash":
		col, ph, n = x, y-anim/4, 17
	case "spore":
		col, ph, n = x+anim/5, y-anim/5, 19
	}
	h := wxMix(uint64(col)*0x9e3779b97f4a7c15 + 0x5eed)
	if floorMod(ph, n) == int(h%uint64(n)) {
		return k.Glyph, k.GlyphF
	}
	return gl, f
}
